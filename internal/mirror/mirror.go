// Package mirror copies a music library onto a device, transferring only files
// that are new or whose size or modification time differ. It is built for FAT
// targets (Rockbox iPods): times compare within a 2-second window and paths
// compare case-insensitively when deciding what to delete. Hidden files and
// directories (.DS_Store, ._* resource forks) are ignored on both sides.
package mirror

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"
)

// modifyWindow absorbs FAT's 2-second timestamp resolution.
const modifyWindow = 2 * time.Second

// tmpSuffix marks partially written files; they are renamed into place only
// after a complete, synced copy.
const tmpSuffix = ".hmuzik-tmp"

// Action is what Mirror did (or would do, in a dry run) to one path.
type Action string

const (
	ActionCopy   Action = "copy"
	ActionDelete Action = "delete"
)

type Options struct {
	// Delete removes files under dst that no longer exist under src.
	Delete bool
	// DryRun reports actions without writing to dst.
	DryRun bool
	// Report, if set, is called for every copy and delete with a path relative
	// to the root.
	Report func(a Action, rel string)
	// Exclude lists filepath.Match patterns tested against each file's base
	// name. Matching files are neither copied nor, with Delete, removed from
	// dst.
	Exclude []string
}

type Result struct {
	Copied  int
	Deleted int
	Skipped int
	Bytes   int64
}

// deviceRootMarkers exist only at the top of an iPod; finding one in dst means
// --delete would wipe everything on the device outside the library.
var deviceRootMarkers = []string{".rockbox", "iPod_Control"}

// Mirror makes dst match src. dst must already exist and must stay the same
// directory for the whole run: a missing or unmounted device fails rather than
// silently filling the parent filesystem. Errors on individual files are
// collected and returned together and the sync continues past them, except
// device-level failures (disk full, read-only remount, I/O error), which stop
// it. Mirror never unmounts or ejects dst.
func Mirror(src, dst string, opts Options) (Result, error) {
	var res Result
	if _, err := requireDir(src, "source"); err != nil {
		return res, err
	}
	root, err := requireDir(dst, "destination")
	if err != nil {
		return res, fmt.Errorf("%w (is the device mounted?)", err)
	}
	for _, pat := range opts.Exclude {
		if _, err := filepath.Match(pat, ""); err != nil {
			return res, fmt.Errorf("exclude pattern %q: %w", pat, err)
		}
	}
	files, err := listFiles(src, opts.Exclude)
	if err != nil {
		return res, err
	}
	// mounted reports an error once dst no longer refers to the directory
	// checked above, e.g. after the device is unplugged mid-run.
	mounted := func() error {
		info, err := os.Stat(dst)
		if err != nil || !os.SameFile(root, info) {
			return fmt.Errorf("destination %s changed or disappeared mid-sync (device unmounted?); stopping", dst)
		}
		return nil
	}

	var errs []error
	if opts.Delete {
		if len(files) == 0 {
			return res, fmt.Errorf("source %s has no files; refusing to delete from destination", src)
		}
		for _, m := range deviceRootMarkers {
			if _, err := os.Stat(filepath.Join(dst, m)); err == nil {
				return res, fmt.Errorf("destination %s contains %s, so it looks like a device root; refusing to delete (point --dest at the music folder)", dst, m)
			}
		}
		n, derr := deleteExtra(dst, files, opts, mounted)
		res.Deleted = n
		if derr != nil {
			errs = append(errs, derr)
		}
		if isFatal(derr) {
			return res, errors.Join(errs...)
		}
	}

	for _, rel := range files {
		copied, n, cerr := syncFile(filepath.Join(src, rel), filepath.Join(dst, rel), opts.DryRun, mounted)
		if cerr != nil {
			errs = append(errs, fmt.Errorf("copy %s: %w", rel, cerr))
			if isFatal(cerr) {
				break
			}
			continue
		}
		if !copied {
			res.Skipped++
			continue
		}
		res.Copied++
		res.Bytes += n
		report(opts, ActionCopy, rel)
	}
	return res, errors.Join(errs...)
}

func report(opts Options, a Action, rel string) {
	if opts.Report != nil {
		opts.Report(a, rel)
	}
}

// errUnmounted marks a run stopped because dst changed underneath it.
var errUnmounted = errors.New("destination unmounted")

// isFatal reports whether err means the device itself is unusable, so every
// remaining file would fail the same way.
func isFatal(err error) bool {
	return errors.Is(err, errUnmounted) || errors.Is(err, syscall.ENOSPC) ||
		errors.Is(err, syscall.EROFS) || errors.Is(err, syscall.EIO)
}

func requireDir(path, what string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", what, path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s %s: not a directory", what, path)
	}
	return info, nil
}

// listFiles returns regular files under root as sorted relative paths,
// leaving out excluded ones.
func listFiles(root string, exclude []string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && isHidden(d.Name()) {
			return skip(d)
		}
		if !d.Type().IsRegular() || strings.HasSuffix(d.Name(), tmpSuffix) || isExcluded(d.Name(), exclude) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}
	slices.Sort(files)
	return files, nil
}

func isHidden(name string) bool { return strings.HasPrefix(name, ".") }

// isExcluded reports whether name matches any pattern. Patterns are validated
// up front in Mirror, so Match can't fail here.
func isExcluded(name string, patterns []string) bool {
	for _, pat := range patterns {
		if ok, _ := filepath.Match(pat, name); ok {
			return true
		}
	}
	return false
}

// skip leaves a hidden entry out of a walk, pruning it whole if it is a
// directory.
func skip(d fs.DirEntry) error {
	if d.IsDir() {
		return filepath.SkipDir
	}
	return nil
}

// needsCopy reports whether dst is missing or differs from src in size or
// modification time.
func needsCopy(src os.FileInfo, dst string) (bool, error) {
	info, err := os.Stat(dst)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	if info.Size() != src.Size() {
		return true, nil
	}
	return src.ModTime().Sub(info.ModTime()).Abs() > modifyWindow, nil
}

func syncFile(src, dst string, dryRun bool, mounted func() error) (copied bool, n int64, err error) {
	info, err := os.Stat(src)
	if err != nil {
		return false, 0, err
	}
	need, err := needsCopy(info, dst)
	if err != nil || !need {
		return false, 0, err
	}
	if dryRun {
		return true, info.Size(), nil
	}
	if err := mounted(); err != nil {
		return false, 0, fmt.Errorf("%w: %w", errUnmounted, err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, 0, err
	}
	n, err = copyFile(src, dst, info.ModTime())
	return err == nil, n, err
}

// copyFile writes src to a temp file beside dst, syncs it, stamps src's
// modification time, and renames it over dst so an interrupted copy never
// leaves a truncated track in place.
func copyFile(src, dst string, mtime time.Time) (int64, error) {
	in, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() { _ = in.Close() }()

	tmp := dst + tmpSuffix
	out, err := os.Create(tmp)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(out, in)
	if err == nil {
		err = out.Sync()
	}
	if cerr := out.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chtimes(tmp, mtime, mtime)
	}
	if err == nil {
		err = os.Rename(tmp, dst)
	}
	if err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	return n, nil
}

// deleteExtra removes files under dst with no source counterpart, matching
// case-insensitively because FAT treats "Foo" and "foo" as the same file.
// Leftover temp files are removed, and so are directories emptied by the
// deletes.
func deleteExtra(dst string, files []string, opts Options, mounted func() error) (int, error) {
	keep := make(map[string]struct{}, len(files))
	for _, rel := range files {
		keep[strings.ToLower(rel)] = struct{}{}
	}
	var (
		dirs    []string
		deleted int
		errs    []error
	)
	err := filepath.WalkDir(dst, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		if path == dst {
			return nil
		}
		if isHidden(d.Name()) {
			return skip(d)
		}
		rel, err := filepath.Rel(dst, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, path)
			return nil
		}
		if _, ok := keep[strings.ToLower(rel)]; ok || isExcluded(d.Name(), opts.Exclude) {
			return nil
		}
		if opts.DryRun {
			report(opts, ActionDelete, rel)
			deleted++
			return nil
		}
		if merr := mounted(); merr != nil {
			return fmt.Errorf("%w: %w", errUnmounted, merr)
		}
		if rerr := os.Remove(path); rerr != nil {
			rerr = fmt.Errorf("delete %s: %w", rel, rerr)
			if isFatal(rerr) {
				return rerr
			}
			errs = append(errs, rerr)
			return nil
		}
		report(opts, ActionDelete, rel)
		deleted++
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	if !opts.DryRun && err == nil {
		// Deepest first, so parents empty out after their children.
		for _, dir := range slices.Backward(dirs) {
			if entries, rerr := os.ReadDir(dir); rerr == nil && len(entries) == 0 {
				_ = os.Remove(dir)
			}
		}
	}
	return deleted, errors.Join(errs...)
}
