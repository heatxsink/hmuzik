package recentlyadded

import (
	"reflect"
	"testing"
	"time"
)

func tk(path, aa, alb string, disc, track int, t time.Time) *Track {
	return &Track{Path: path, AlbumArtist: aa, Album: alb, Disc: disc, Track: track, Ctime: t}
}

func TestGroupByAlbum_BucketsAndSorts(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	tracks := []*Track{
		tk("/a/3.flac", "ArtistA", "AlbumA", 1, 3, now.Add(-3*24*time.Hour)),
		tk("/a/1.flac", "ArtistA", "AlbumA", 1, 1, now.Add(-1*24*time.Hour)),
		tk("/a/2.flac", "ArtistA", "AlbumA", 1, 2, now.Add(-2*24*time.Hour)),
		tk("/b/1.flac", "ArtistB", "AlbumB", 1, 1, now.Add(-10*24*time.Hour)),
	}
	albums := GroupByAlbum(tracks)
	if len(albums) != 2 {
		t.Fatalf("want 2 albums, got %d", len(albums))
	}
	var a *Album
	for _, al := range albums {
		if al.Key == "ArtistA\x00AlbumA" {
			a = al
		}
	}
	if a == nil {
		t.Fatal("missing ArtistA/AlbumA")
	}
	wantOrder := []string{"/a/1.flac", "/a/2.flac", "/a/3.flac"}
	got := []string{a.Tracks[0].Path, a.Tracks[1].Path, a.Tracks[2].Path}
	if !reflect.DeepEqual(got, wantOrder) {
		t.Errorf("track order: want %v, got %v", wantOrder, got)
	}
	if !a.LatestCtime.Equal(now.Add(-1 * 24 * time.Hour)) {
		t.Errorf("LatestCtime: want %v, got %v", now.Add(-1*24*time.Hour), a.LatestCtime)
	}
}

func TestGroupByAlbum_TaglessAreOrphans(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	tracks := []*Track{
		tk("/orphan/x.mp3", "", "", 0, 0, now),
		tk("/orphan/y.mp3", "", "", 0, 0, now),
		tk("/known/1.flac", "ArtistA", "AlbumA", 1, 1, now),
	}
	albums := GroupByAlbum(tracks)
	if len(albums) != 3 {
		t.Fatalf("want 3 albums (2 orphans + 1 real), got %d", len(albums))
	}
}

func TestGroupByAlbum_AlbumArtistDistinguishesCompilations(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	tracks := []*Track{
		tk("/c/1.flac", "Various Artists", "Comp1", 1, 1, now),
		tk("/c/2.flac", "Various Artists", "Comp1", 1, 2, now),
	}
	albums := GroupByAlbum(tracks)
	if len(albums) != 1 {
		t.Fatalf("want 1 compilation album, got %d", len(albums))
	}
}

func TestPlaylist_WindowFiltering(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	tracks := []*Track{
		tk("/a/1.flac", "ArtistA", "AlbumA", 1, 1, now.Add(-12*time.Hour)),
		tk("/a/2.flac", "ArtistA", "AlbumA", 1, 2, now.Add(-3*24*time.Hour)),
		tk("/a/3.flac", "ArtistA", "AlbumA", 1, 3, now.Add(-10*24*time.Hour)),
	}
	albums := GroupByAlbum(tracks)

	got := Playlist(albums, 24*time.Hour, now)
	if !reflect.DeepEqual(got, []string{"/a/1.flac"}) {
		t.Errorf("1d window: want only /a/1.flac, got %v", got)
	}

	got = Playlist(albums, 7*24*time.Hour, now)
	if !reflect.DeepEqual(got, []string{"/a/1.flac", "/a/2.flac"}) {
		t.Errorf("7d window: want 1.flac then 2.flac in album order, got %v", got)
	}

	got = Playlist(albums, 30*24*time.Hour, now)
	if !reflect.DeepEqual(got, []string{"/a/1.flac", "/a/2.flac", "/a/3.flac"}) {
		t.Errorf("30d window: want all in album order, got %v", got)
	}
}

func TestPlaylist_AlbumOrderByRecency(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	tracks := []*Track{
		tk("/old/1.flac", "Old", "Old", 1, 1, now.Add(-20*24*time.Hour)),
		tk("/old/2.flac", "Old", "Old", 1, 2, now.Add(-25*24*time.Hour)),
		tk("/new/1.flac", "New", "New", 1, 1, now.Add(-2*24*time.Hour)),
		tk("/new/2.flac", "New", "New", 1, 2, now.Add(-3*24*time.Hour)),
	}
	albums := GroupByAlbum(tracks)
	got := Playlist(albums, 30*24*time.Hour, now)
	want := []string{"/new/1.flac", "/new/2.flac", "/old/1.flac", "/old/2.flac"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("album order: want %v, got %v", want, got)
	}
}

func TestPlaylist_PartialAlbumKeepsAlbumPosition(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	// Album with tracks 1..5, only track 3 was re-touched yesterday.
	tracks := []*Track{
		tk("/a/1.flac", "ArtistA", "AlbumA", 1, 1, now.Add(-200*24*time.Hour)),
		tk("/a/2.flac", "ArtistA", "AlbumA", 1, 2, now.Add(-200*24*time.Hour)),
		tk("/a/3.flac", "ArtistA", "AlbumA", 1, 3, now.Add(-12*time.Hour)),
		tk("/a/4.flac", "ArtistA", "AlbumA", 1, 4, now.Add(-200*24*time.Hour)),
		tk("/a/5.flac", "ArtistA", "AlbumA", 1, 5, now.Add(-200*24*time.Hour)),
	}
	albums := GroupByAlbum(tracks)
	got := Playlist(albums, 24*time.Hour, now)
	if !reflect.DeepEqual(got, []string{"/a/3.flac"}) {
		t.Errorf("partial album: want only /a/3.flac, got %v", got)
	}
}

func TestPlaylist_MultiDiscOrdering(t *testing.T) {
	now := time.Date(2026, 5, 13, 0, 0, 0, 0, time.UTC)
	tracks := []*Track{
		tk("/a/d2t2.flac", "ArtistA", "AlbumA", 2, 2, now.Add(-1*time.Hour)),
		tk("/a/d1t2.flac", "ArtistA", "AlbumA", 1, 2, now.Add(-1*time.Hour)),
		tk("/a/d2t1.flac", "ArtistA", "AlbumA", 2, 1, now.Add(-1*time.Hour)),
		tk("/a/d1t1.flac", "ArtistA", "AlbumA", 1, 1, now.Add(-1*time.Hour)),
	}
	albums := GroupByAlbum(tracks)
	got := Playlist(albums, 24*time.Hour, now)
	want := []string{"/a/d1t1.flac", "/a/d1t2.flac", "/a/d2t1.flac", "/a/d2t2.flac"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("multi-disc order: want %v, got %v", want, got)
	}
}

func TestIsAudioFile(t *testing.T) {
	cases := map[string]bool{
		"song.flac":     true,
		"song.FLAC":     true,
		"song.mp3":      true,
		"song.m4a":      true,
		"song.m4p":      true,
		"song.aif":      true,
		"song.aiff":     true,
		"song.aifc":     true,
		"song.ogg":      true,
		"song.wav":      true,
		"cover.jpg":     false,
		"notes.txt":     false,
		"song":          false,
		"song.flac.bak": false,
	}
	for name, want := range cases {
		if got := isAudioFile(name); got != want {
			t.Errorf("isAudioFile(%q) = %v, want %v", name, got, want)
		}
	}
}
