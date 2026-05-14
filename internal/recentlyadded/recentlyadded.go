package recentlyadded

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/dhowden/tag"
)

type Track struct {
	Path        string
	Ctime       time.Time
	AlbumArtist string
	Album       string
	Disc        int
	Track       int
}

type Album struct {
	Key         string
	LatestCtime time.Time
	Tracks      []*Track
}

func isAudioFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".aif", ".aiff", ".aifc", ".flac", ".m4a", ".m4p", ".mp3", ".ogg", ".wav":
		return true
	}
	return false
}

// Scan walks root once and returns audio files whose ctime is at or after
// since. Files older than since are skipped without opening (no tag read), so
// cost scales with recent activity rather than library size. Errors on
// individual files are logged to stderr and the walk continues.
func Scan(root string, since time.Time) ([]*Track, error) {
	tracks := []*Track{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintln(os.Stderr, "walk:", err)
			return nil
		}
		if d.IsDir() || !isAudioFile(d.Name()) {
			return nil
		}
		info, ierr := d.Info()
		if ierr != nil {
			fmt.Fprintln(os.Stderr, "stat:", path, ierr)
			return nil
		}
		if info.Size() == 0 {
			return nil
		}
		ctime := fileCtime(info)
		if ctime.Before(since) {
			return nil
		}
		t := &Track{Path: path, Ctime: ctime}
		readTags(path, t)
		tracks = append(tracks, t)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk %s: %w", root, err)
	}
	return tracks, nil
}

func readTags(path string, t *Track) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open:", path, err)
		return
	}
	defer func() { _ = f.Close() }()
	m, err := tag.ReadFrom(f)
	if err != nil {
		if !errors.Is(err, tag.ErrNoTagsFound) {
			fmt.Fprintln(os.Stderr, "tag:", path, err)
		}
		return
	}
	t.AlbumArtist = strings.TrimSpace(m.AlbumArtist())
	if t.AlbumArtist == "" {
		t.AlbumArtist = strings.TrimSpace(m.Artist())
	}
	t.Album = strings.TrimSpace(m.Album())
	t.Disc, _ = m.Disc()
	t.Track, _ = m.Track()
}

// GroupByAlbum buckets tracks by (AlbumArtist, Album). Tracks missing either
// field become single-track pseudo-albums keyed by file path so they are still
// included but never merged with real albums. Each album's tracks are sorted
// by (Disc, Track, Path); LatestCtime is the max ctime across its tracks.
func GroupByAlbum(tracks []*Track) []*Album {
	byKey := make(map[string]*Album)
	for _, t := range tracks {
		key := albumKey(t)
		a, ok := byKey[key]
		if !ok {
			a = &Album{Key: key}
			byKey[key] = a
		}
		a.Tracks = append(a.Tracks, t)
		if t.Ctime.After(a.LatestCtime) {
			a.LatestCtime = t.Ctime
		}
	}
	albums := make([]*Album, 0, len(byKey))
	for _, a := range byKey {
		sort.SliceStable(a.Tracks, func(i, j int) bool {
			ti, tj := a.Tracks[i], a.Tracks[j]
			if ti.Disc != tj.Disc {
				return ti.Disc < tj.Disc
			}
			if ti.Track != tj.Track {
				return ti.Track < tj.Track
			}
			return ti.Path < tj.Path
		})
		albums = append(albums, a)
	}
	return albums
}

func albumKey(t *Track) string {
	if t.AlbumArtist == "" || t.Album == "" {
		return "\x00orphan\x00" + t.Path
	}
	return t.AlbumArtist + "\x00" + t.Album
}

// Playlist returns the ordered file paths for the given window. Tracks whose
// ctime is older than now-window are dropped; tracks within the window retain
// their album-internal (Disc, Track) order. Albums are ordered by
// LatestCtime descending, with album key as a stable tiebreaker.
func Playlist(albums []*Album, window time.Duration, now time.Time) []string {
	cutoff := now.Add(-window)
	type entry struct {
		album  *Album
		tracks []*Track
	}
	entries := make([]entry, 0, len(albums))
	for _, a := range albums {
		var kept []*Track
		for _, t := range a.Tracks {
			if !t.Ctime.Before(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) > 0 {
			entries = append(entries, entry{album: a, tracks: kept})
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if !entries[i].album.LatestCtime.Equal(entries[j].album.LatestCtime) {
			return entries[i].album.LatestCtime.After(entries[j].album.LatestCtime)
		}
		return entries[i].album.Key < entries[j].album.Key
	})
	total := 0
	for _, e := range entries {
		total += len(e.tracks)
	}
	out := make([]string, 0, total)
	for _, e := range entries {
		for _, t := range e.tracks {
			out = append(out, t.Path)
		}
	}
	return out
}

// WriteCmusPlaylist writes paths newline-delimited to the given file. cmus
// reads this format directly from ~/.config/cmus/playlists/.
func WriteCmusPlaylist(path string, lines []string) error {
	body := strings.Join(lines, "\n")
	if len(lines) > 0 {
		body += "\n"
	}
	return os.WriteFile(path, []byte(body), 0644)
}
