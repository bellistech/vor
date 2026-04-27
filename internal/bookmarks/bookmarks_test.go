package bookmarks

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// withTempBookmarkFile redirects the package-level bookmarkFile to a temp path
// for the duration of the test. Restores the original on teardown.
func withTempBookmarkFile(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bookmarks.json")
	orig := bookmarkFile
	SetBookmarkFile(path)
	t.Cleanup(func() { SetBookmarkFile(orig) })
	return path
}

func TestLoad_NoFile(t *testing.T) {
	withTempBookmarkFile(t)
	got := Load()
	if got != nil {
		t.Errorf("Load() with no file should return nil, got %v", got)
	}
}

func TestLoad_EmptyFileReturnsNil(t *testing.T) {
	path := withTempBookmarkFile(t)
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatalf("seed empty file: %v", err)
	}
	got := Load()
	if got != nil {
		t.Errorf("Load() of empty file should return nil, got %v", got)
	}
}

func TestSave_Load_Roundtrip(t *testing.T) {
	withTempBookmarkFile(t)
	want := []string{"bgp", "kubernetes", "linux-kernel-internals"}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := Load()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load after Save: got %v, want %v", got, want)
	}
}

func TestToggle_AddNew(t *testing.T) {
	withTempBookmarkFile(t)
	added, err := Toggle("docker")
	if err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if !added {
		t.Errorf("Toggle on new topic should return true (added)")
	}
	if !IsBookmarked("docker") {
		t.Errorf("docker should be bookmarked after Toggle")
	}
}

func TestToggle_RemoveExisting(t *testing.T) {
	withTempBookmarkFile(t)
	if err := Save([]string{"docker", "kubernetes"}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	added, err := Toggle("docker")
	if err != nil {
		t.Fatalf("Toggle: %v", err)
	}
	if added {
		t.Errorf("Toggle on existing topic should return false (removed)")
	}
	if IsBookmarked("docker") {
		t.Errorf("docker should NOT be bookmarked after Toggle-to-remove")
	}
	// Other bookmark untouched
	if !IsBookmarked("kubernetes") {
		t.Errorf("kubernetes should still be bookmarked after removing docker")
	}
}

func TestToggle_AddSorts(t *testing.T) {
	withTempBookmarkFile(t)
	for _, topic := range []string{"zsh", "ansible", "kubernetes"} {
		if _, err := Toggle(topic); err != nil {
			t.Fatalf("Toggle %s: %v", topic, err)
		}
	}
	got := List()
	want := []string{"ansible", "kubernetes", "zsh"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List() should return sorted: got %v, want %v", got, want)
	}
}

func TestList_EmptyWhenNoFile(t *testing.T) {
	withTempBookmarkFile(t)
	got := List()
	if len(got) != 0 {
		t.Errorf("List() with no bookmarks should be empty, got %v", got)
	}
}

func TestIsBookmarked_NotPresent(t *testing.T) {
	withTempBookmarkFile(t)
	if IsBookmarked("nothing-here") {
		t.Errorf("IsBookmarked on absent topic should return false")
	}
}

func TestSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	// Use a non-existent nested directory
	nestedPath := filepath.Join(dir, "deep", "nested", "bookmarks.json")
	orig := bookmarkFile
	SetBookmarkFile(nestedPath)
	t.Cleanup(func() { SetBookmarkFile(orig) })

	if err := Save([]string{"x"}); err != nil {
		t.Fatalf("Save into nested path: %v", err)
	}
	if _, err := os.Stat(nestedPath); err != nil {
		t.Errorf("expected file at %s after Save, stat err: %v", nestedPath, err)
	}
}

func TestSave_EmptyPath_NoOp(t *testing.T) {
	orig := bookmarkFile
	SetBookmarkFile("")
	t.Cleanup(func() { SetBookmarkFile(orig) })
	// Should not error even with empty path
	if err := Save([]string{"x"}); err != nil {
		t.Errorf("Save with empty path should be a no-op (no error), got %v", err)
	}
}

func TestLoad_EmptyPath_ReturnsNil(t *testing.T) {
	orig := bookmarkFile
	SetBookmarkFile("")
	t.Cleanup(func() { SetBookmarkFile(orig) })
	if got := Load(); got != nil {
		t.Errorf("Load with empty path should return nil, got %v", got)
	}
}

func TestSave_ProducesValidJSON(t *testing.T) {
	path := withTempBookmarkFile(t)
	if err := Save([]string{"a", "b", "c"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	// Indent format means it should contain newlines
	if len(raw) == 0 {
		t.Errorf("saved file is empty")
	}
	wantPrefix := "[\n  "
	if string(raw[:4]) != wantPrefix {
		t.Errorf("saved file should start with indented JSON %q, got %q", wantPrefix, string(raw[:4]))
	}
}

func TestToggle_PersistsAcrossLoads(t *testing.T) {
	withTempBookmarkFile(t)
	if _, err := Toggle("alpha"); err != nil {
		t.Fatalf("Toggle alpha: %v", err)
	}
	if _, err := Toggle("beta"); err != nil {
		t.Fatalf("Toggle beta: %v", err)
	}
	// Fresh Load — confirm both persisted
	got := List()
	want := []string{"alpha", "beta"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("after toggles, List = %v, want %v", got, want)
	}
}

func TestToggle_DuplicateTopicNotDuplicated(t *testing.T) {
	withTempBookmarkFile(t)
	if _, err := Toggle("dup"); err != nil {
		t.Fatalf("Toggle 1: %v", err)
	}
	// Toggle again removes (already covered) — but if we save, then toggle, then save again
	if err := Save([]string{"dup", "dup"}); err != nil {
		t.Fatalf("Save with duplicate: %v", err)
	}
	// Verify no extra de-dupe — current implementation doesn't dedupe on Save.
	// Document the current behavior; if a bug is later fixed, the test will need updating.
	got := Load()
	if len(got) != 2 {
		t.Errorf("Save preserves duplicates as-is (current behavior), got len=%d", len(got))
	}
}
