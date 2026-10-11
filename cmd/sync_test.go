package cmd

import (
	"reflect"
	"testing"
)

func TestSyncTargets(t *testing.T) {
	music, playlists := syncTargets("/home/u", "/run/media/u/IPOD")
	want := []syncTarget{
		{name: "music", source: "/home/u/Music/Artists", dest: "/run/media/u/IPOD/Music/Artists", exclude: []string{"cover.jpg"}},
		{name: "playlists", source: "/home/u/Music/Playlists", dest: "/run/media/u/IPOD/Playlists"},
	}
	for i, got := range []syncTarget{music, playlists} {
		if !reflect.DeepEqual(got, want[i]) {
			t.Errorf("target %d: want %+v, got %+v", i, want[i], got)
		}
	}
}

func TestSyncTargetWithOverrides(t *testing.T) {
	base := syncTarget{name: "music", source: "/src", dest: "/dst"}
	cases := []struct {
		name, source, dest string
		want               syncTarget
	}{
		{"none", "", "", base},
		{"source", "/other", "", syncTarget{name: "music", source: "/other", dest: "/dst"}},
		{"dest", "", "/mnt", syncTarget{name: "music", source: "/src", dest: "/mnt"}},
		{"both", "/other", "/mnt", syncTarget{name: "music", source: "/other", dest: "/mnt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := base.withOverrides(tc.source, tc.dest); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("want %+v, got %+v", tc.want, got)
			}
		})
	}
}
