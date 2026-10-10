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
}

type Result struct {
	Copied  int
	Deleted int
	Skipped int
	Bytes   int64
}

// Mirror makes dst match src. dst must already exist: a missing mount point
// should fail rather than silently fill the parent filesystem. Errors on
// individual files are collected and returned together; the sync continues
// past them. Mirror never unmounts or ejects dst.
func Mirror(src, dst string, opts Options) (Result, error) {
	var res Result
	if err := requireDir(src, "source"); err != nil {
		return res, err
	}
	if err := requireDir(dst, "destination"); err != nil {
		return res, fmt.Errorf("%w (is the device mounted?)", err)
	}
	files, err := listFiles(src)
	if err != nil {
		return res, err
	}

	var errs []error
	if opts.Delete {
		if len(files) == 0 {
			return res, fmt.Errorf("source %s has no files; refusing to delete from destination", src)
		}
		n, derr := deleteExtra(dst, files, opts)
		res.Deleted = n
		if derr != nil {
			errs = append(errs, derr)
		}
	}

	for _, rel := range files {
		copied, n, cerr := syncFile(filepath.Join(src, rel), filepath.Join(dst, rel), opts.DryRun)
		if cerr != nil {
			errs = append(errs, fmt.Errorf("copy %s: %w", rel, cerr))
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

func requireDir(path, what string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s %s: %w", what, path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s %s: not a directory", what, path)
	}
	return nil
}

// listFiles returns regular files under root as sorted relative paths.
func listFiles(root string) ([]string, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != root && isHidden(d.Name()) {
			return skip(d)
		}
		if !d.Type().IsRegular() || strings.HasSuffix(d.Name(), tmpSuffix) {
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

func syncFile(src, dst string, dryRun bool) (copied bool, n int64, err error) {
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
func deleteExtra(dst string, files []string, opts Options) (int, error) {
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
		if _, ok := keep[strings.ToLower(rel)]; ok {
			return nil
		}
		report(opts, ActionDelete, rel)
		deleted++
		if opts.DryRun {
			return nil
		}
		if rerr := os.Remove(path); rerr != nil {
			errs = append(errs, fmt.Errorf("delete %s: %w", rel, rerr))
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}
	if !opts.DryRun {
		// Deepest first, so parents empty out after their children.
		for _, dir := range slices.Backward(dirs) {
			if entries, rerr := os.ReadDir(dir); rerr == nil && len(entries) == 0 {
				_ = os.Remove(dir)
			}
		}
	}
	return deleted, errors.Join(errs...)
}
