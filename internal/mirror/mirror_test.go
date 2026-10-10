package mirror

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
	"time"
)

var base = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// p joins a slash-separated relative path onto root.
func p(root, rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }

func write(t *testing.T, path, body string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

type recorder struct{ copies, deletes []string }

func (r *recorder) report(a Action, rel string) {
	switch a {
	case ActionCopy:
		r.copies = append(r.copies, rel)
	case ActionDelete:
		r.deletes = append(r.deletes, rel)
	}
}

func TestMirror_CopiesOnlyNewAndChanged(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	write(t, p(src, "A/new.flac"), "new", base)
	write(t, p(src, "A/same.flac"), "same", base)
	write(t, p(dst, "A/same.flac"), "same", base.Add(time.Second)) // inside FAT window
	write(t, p(src, "A/retagged.flac"), "tag2", base.Add(time.Hour))
	write(t, p(dst, "A/retagged.flac"), "tag1", base) // same size, newer mtime
	write(t, p(src, "A/grown.flac"), "longer", base)
	write(t, p(dst, "A/grown.flac"), "short", base) // same mtime, new size

	var r recorder
	res, err := Mirror(src, dst, Options{Report: r.report})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"A/grown.flac", "A/new.flac", "A/retagged.flac"}
	if !reflect.DeepEqual(r.copies, want) {
		t.Errorf("copies: want %v, got %v", want, r.copies)
	}
	if res.Copied != 3 || res.Skipped != 1 || res.Bytes != int64(len("longer")+len("new")+len("tag2")) {
		t.Errorf("result: %+v", res)
	}
	if got := read(t, p(dst, "A/retagged.flac")); got != "tag2" {
		t.Errorf("retagged content: %q", got)
	}
	info, err := os.Stat(p(dst, "A/new.flac"))
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(base) {
		t.Errorf("mtime not preserved: %v", info.ModTime())
	}

	// A second run is a no-op.
	res, err = Mirror(src, dst, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Copied != 0 || res.Skipped != 4 {
		t.Errorf("second run: %+v", res)
	}
}

func TestMirror_DeleteRemovesExtrasCaseInsensitively(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	write(t, p(src, "Face to Face/Big Choice/01.flac"), "x", base)
	write(t, p(dst, "face to face/big choice/01.flac"), "x", base) // same file on FAT
	write(t, p(dst, "Gone/Album/01.flac"), "old", base)
	write(t, p(dst, "Kept/01.flac"+tmpSuffix), "partial", base)

	var r recorder
	res, err := Mirror(src, dst, Options{Delete: true, DryRun: true, Report: r.report})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Gone/Album/01.flac", "Kept/01.flac" + tmpSuffix}
	slices.Sort(r.deletes)
	if !reflect.DeepEqual(r.deletes, want) || res.Deleted != 2 {
		t.Errorf("dry run deletes: want %v, got %v (%+v)", want, r.deletes, res)
	}
	if _, err := os.Stat(p(dst, "Gone/Album/01.flac")); err != nil {
		t.Errorf("dry run deleted a file: %v", err)
	}

	if _, err := Mirror(src, dst, Options{Delete: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p(dst, "Gone")); !os.IsNotExist(err) {
		t.Errorf("emptied directory not removed: %v", err)
	}
	if _, err := os.Stat(p(dst, "face to face/big choice/01.flac")); err != nil {
		t.Errorf("case-variant match deleted: %v", err)
	}
}

func TestMirror_DryRunWritesNothing(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	write(t, p(src, "A/1.flac"), "x", base)
	res, err := Mirror(src, dst, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Copied != 1 {
		t.Errorf("want 1 planned copy, got %+v", res)
	}
	entries, err := os.ReadDir(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("dry run wrote %d entries", len(entries))
	}
}

func TestMirror_RefusesMissingDestination(t *testing.T) {
	src := t.TempDir()
	write(t, p(src, "A/1.flac"), "x", base)
	dst := filepath.Join(t.TempDir(), "IPOD", "Music")
	if _, err := Mirror(src, dst, Options{}); err == nil {
		t.Fatal("want error for missing destination")
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("destination was created: %v", err)
	}
}

func TestMirror_RefusesDeleteFromEmptySource(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	write(t, p(dst, "A/1.flac"), "x", base)
	if _, err := Mirror(src, dst, Options{Delete: true}); err == nil {
		t.Fatal("want error for empty source with --delete")
	}
	if _, err := os.Stat(p(dst, "A/1.flac")); err != nil {
		t.Errorf("file deleted: %v", err)
	}
}

func TestMirror_IgnoresHiddenFiles(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	write(t, p(src, "A/1.flac"), "x", base)
	write(t, p(src, "A/.DS_Store"), "junk", base)
	write(t, p(src, ".git/config"), "junk", base)
	write(t, p(dst, ".rockbox/config.cfg"), "keep", base)
	write(t, p(dst, "A/._1.flac"), "keep", base)

	var r recorder
	if _, err := Mirror(src, dst, Options{Delete: true, Report: r.report}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"A/1.flac"}; !reflect.DeepEqual(r.copies, want) {
		t.Errorf("copies: want %v, got %v", want, r.copies)
	}
	if len(r.deletes) != 0 {
		t.Errorf("deleted hidden files: %v", r.deletes)
	}
	for _, rel := range []string{".rockbox/config.cfg", "A/._1.flac"} {
		if _, err := os.Stat(p(dst, rel)); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
}
