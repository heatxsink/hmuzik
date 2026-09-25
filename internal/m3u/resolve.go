package m3u

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrNotFound  = errors.New("no matching track found")
	ErrAmbiguous = errors.New("multiple matching tracks found")
)

var audioExts = map[string]bool{
	".aif": true, ".aiff": true, ".aifc": true, ".flac": true, ".m4a": true,
	".m4p": true, ".mp3": true, ".ogg": true, ".wav": true,
}

// trackNumberRe matches a leading track number such as "01. ", "01 - " or "7 ".
var trackNumberRe = regexp.MustCompile(`^(\d{1,3})(?:\s*[.\-_]\s*|\s+)`)

// Resolver finds the current location of playlist entries whose files were
// renamed after the playlist was saved (case changes, a renamed album
// directory, or a new filename pattern). It searches the entry's album
// directory, then its artist directory, and matches on track title.
type Resolver struct {
	listings map[string][]string
}

func NewResolver() *Resolver {
	return &Resolver{listings: map[string][]string{}}
}

// Resolve returns the path of the file that path most likely refers to now.
// The artist directory is searched only when the album directory is gone, so
// a track missing from an existing album never matches elsewhere.
func (r *Resolver) Resolve(path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", ErrNotFound
	}
	album := filepath.Dir(path)
	base, ok := findDirFold(album)
	if !ok {
		base, ok = findDirFold(filepath.Dir(album))
	}
	if !ok {
		return "", ErrNotFound
	}
	files, err := r.audioFiles(base)
	if err != nil {
		return "", err
	}
	num, title := splitName(path)
	return pickMatch(files, num, title)
}

func (r *Resolver) audioFiles(dir string) ([]string, error) {
	if files, ok := r.listings[dir]; ok {
		return files, nil
	}
	files := []string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && audioExts[strings.ToLower(filepath.Ext(p))] {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	r.listings[dir] = files
	return files, nil
}

// pickMatch returns the single file whose title matches, using the track
// number only to break ties.
func pickMatch(files []string, num int, title string) (string, error) {
	var matches []string
	for _, f := range files {
		_, t := splitName(f)
		if t == title || strings.HasSuffix(t, " - "+title) {
			matches = append(matches, f)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) == 0 {
		return "", ErrNotFound
	}
	var numbered []string
	for _, m := range matches {
		if n, _ := splitName(m); n == num && num >= 0 {
			numbered = append(numbered, m)
		}
	}
	if len(numbered) == 1 {
		return numbered[0], nil
	}
	return "", ErrAmbiguous
}

// splitName returns the leading track number (-1 if absent) and the
// normalized title of an audio file path.
func splitName(path string) (int, string) {
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	num := -1
	if m := trackNumberRe.FindStringSubmatch(name); m != nil {
		num, _ = strconv.Atoi(m[1])
		name = name[len(m[0]):]
	}
	return num, strings.Join(strings.Fields(strings.ToLower(name)), " ")
}

// findDirFold returns dir, matching each missing path component
// case-insensitively against its parent's entries.
func findDirFold(dir string) (string, bool) {
	if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
		return dir, true
	}
	parent := filepath.Dir(dir)
	if parent == dir {
		return "", false
	}
	p, ok := findDirFold(parent)
	if !ok {
		return "", false
	}
	entries, err := os.ReadDir(p)
	if err != nil {
		return "", false
	}
	for _, e := range entries {
		if e.IsDir() && strings.EqualFold(e.Name(), filepath.Base(dir)) {
			return filepath.Join(p, e.Name()), true
		}
	}
	return "", false
}
