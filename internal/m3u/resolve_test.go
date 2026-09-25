package m3u

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func touch(t *testing.T, root string, rel ...string) {
	t.Helper()
	for _, r := range rel {
		p := filepath.Join(root, r)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolve(t *testing.T) {
	root := t.TempDir()
	touch(t, root,
		"KUNZITE/VISUALS (DELUXE)/01. LEMON SWAYZE.flac",
		"KUNZITE/VISUALS (DELUXE)/16. AZURITE.flac",
		"MF DOOM/Operation Doomsday/05. Go With the Flow.flac",
		"MF DOOM/Operation Doomsday/06. Go With the Flow.flac",
		"Jeff Mills/The Trip To Vega/07 - Jeff Mills - The Trip To Vega - March of the Purple Orbs.flac",
		"Drake/Other Song.wav",
		"Drake/Views/Fair Trade.flac",
		"Dupes/A/01. Intro.flac",
		"Dupes/B/01. Intro.flac",
	)
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{"artist case and album renamed", "Kunzite/VISUALS/01. LEMON SWAYZE.flac", "KUNZITE/VISUALS (DELUXE)/01. LEMON SWAYZE.flac", nil},
		{"track renumbered", "Kunzite/VISUALS/14. AZURITE.flac", "KUNZITE/VISUALS (DELUXE)/16. AZURITE.flac", nil},
		{"title case, number breaks tie", "MF DOOM/Operation Doomsday/06. Go with the Flow.flac", "MF DOOM/Operation Doomsday/06. Go With the Flow.flac", nil},
		{"new filename pattern", "Jeff Mills/The Trip To Vega/07. March Of The Purple Orbs.flac", "Jeff Mills/The Trip To Vega/07 - Jeff Mills - The Trip To Vega - March of the Purple Orbs.flac", nil},
		{"existing album never searches artist", "Drake/Views/Other Song.flac", "", ErrNotFound},
		{"unknown artist", "Nobody/Album/01. Song.flac", "", ErrNotFound},
		{"relative path is not searched", "", "", ErrNotFound},
		{"ambiguous without number", "Dupes/C/01. Intro.flac", "", ErrAmbiguous},
	}
	r := NewResolver()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := filepath.Join(root, tt.in)
			if tt.in == "" {
				in = "01. LEMON SWAYZE.flac"
			}
			got, err := r.Resolve(in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err: want %v, got %v", tt.wantErr, err)
			}
			if tt.want != "" && got != filepath.Join(root, tt.want) {
				t.Errorf("want %s, got %s", tt.want, got)
			}
		})
	}
}

func TestSplitName(t *testing.T) {
	tests := []struct {
		in    string
		num   int
		title string
	}{
		{"/a/01. LEMON  SWAYZE.flac", 1, "lemon swayze"},
		{"/a/07 - Jeff Mills - Lyra.flac", 7, "jeff mills - lyra"},
		{"/a/10.flac", -1, "10"},
		{"/a/Fair Trade.wav", -1, "fair trade"},
	}
	for _, tt := range tests {
		num, title := splitName(tt.in)
		if num != tt.num || title != tt.title {
			t.Errorf("%s: want (%d, %q), got (%d, %q)", tt.in, tt.num, tt.title, num, title)
		}
	}
}
