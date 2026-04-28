package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTempHistoryFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tui-history")
	prev := historyFileOverride
	historyFileOverride = path
	t.Cleanup(func() { historyFileOverride = prev })
	return path
}

func TestLoadHistory_NoFile(t *testing.T) {
	withTempHistoryFile(t)
	if got := LoadHistory(); got != nil {
		t.Errorf("expected nil history when file missing; got %v", got)
	}
}

func TestSaveLoad_RoundTrip(t *testing.T) {
	withTempHistoryFile(t)
	want := []string{"first", "second", "third"}
	if err := SaveHistory(want); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	got := LoadHistory()
	if len(got) != len(want) {
		t.Fatalf("got %d entries, want %d", len(got), len(want))
	}
	for i, e := range got {
		if e != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, e, want[i])
		}
	}
}

func TestSaveHistory_CapsAtMax(t *testing.T) {
	withTempHistoryFile(t)
	// Build 60 entries, save → only the newest 50 should land
	var entries []string
	for i := 0; i < 60; i++ {
		entries = append(entries, "entry-"+string(rune('A'+i%26))+string(rune('A'+i/26)))
	}
	if err := SaveHistory(entries); err != nil {
		t.Fatal(err)
	}
	got := LoadHistory()
	if len(got) != historyMax {
		t.Errorf("got %d entries, want %d (cap)", len(got), historyMax)
	}
	// Newest entries should be preserved
	if got[len(got)-1] != entries[len(entries)-1] {
		t.Errorf("newest entry not preserved: got %q, want %q",
			got[len(got)-1], entries[len(entries)-1])
	}
}

func TestSaveHistory_EmptyList(t *testing.T) {
	withTempHistoryFile(t)
	if err := SaveHistory(nil); err != nil {
		t.Fatalf("SaveHistory(nil): %v", err)
	}
	if got := LoadHistory(); len(got) != 0 {
		t.Errorf("empty save → load got %d entries, want 0", len(got))
	}
}

func TestLoadHistory_IgnoresBlankLines(t *testing.T) {
	path := withTempHistoryFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "first\n\n  \nsecond\n   \n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got := LoadHistory()
	if len(got) != 2 {
		t.Errorf("got %v (len %d), want 2 entries (blanks/whitespace dropped)", got, len(got))
	}
}

func TestPushHistory_DropsEmpty(t *testing.T) {
	got := pushHistory(nil, "  ")
	if got != nil {
		t.Errorf("blank entry should not be added; got %v", got)
	}
	got = pushHistory(nil, "")
	if got != nil {
		t.Errorf("empty entry should not be added; got %v", got)
	}
}

func TestPushHistory_DedupesConsecutive(t *testing.T) {
	got := pushHistory([]string{"a", "b"}, "b")
	if len(got) != 2 {
		t.Errorf("consecutive dup should be collapsed; got %v", got)
	}
	got = pushHistory([]string{"a", "b"}, "a")
	if len(got) != 3 {
		t.Errorf("non-consecutive dup should be allowed; got %v", got)
	}
}

func TestPushHistory_CapsAtMax(t *testing.T) {
	var seed []string
	for i := 0; i < historyMax; i++ {
		seed = append(seed, string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	got := pushHistory(seed, "fresh")
	if len(got) != historyMax {
		t.Errorf("expected cap at %d after one push at max; got %d", historyMax, len(got))
	}
	if got[len(got)-1] != "fresh" {
		t.Errorf("newest entry should be at the end; got %q", got[len(got)-1])
	}
}

func TestPushHistory_TrimsWhitespace(t *testing.T) {
	got := pushHistory(nil, "  hello  ")
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("expected [hello]; got %v", got)
	}
}

// Integration: drive the TUI through filter / commit / recall.

func TestTUI_FilterCommitPushesHistory(t *testing.T) {
	withTempHistoryFile(t)
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter")) // → topics
	m = step(t, m, keyMsg("/"))     // start filter
	m = step(t, m, keyMsg("z"))     // type 'z'
	m = step(t, m, keyMsg("enter")) // commit
	if len(m.history) != 1 {
		t.Errorf("expected history len 1 after commit, got %d (%v)", len(m.history), m.history)
	}
	if m.history[0] != "z" {
		t.Errorf("history[0] = %q, want z", m.history[0])
	}
}

func TestTUI_FilterUpRecallsLastEntry(t *testing.T) {
	withTempHistoryFile(t)
	m := New(testRegistry(t))
	m.history = []string{"older", "newer"}
	m = step(t, m, keyMsg("enter")) // → topics
	m = step(t, m, keyMsg("/"))     // start filter
	m = step(t, m, keyMsg("up"))    // recall newest
	if m.filter.Value() != "newer" {
		t.Errorf("filter after up = %q, want newer", m.filter.Value())
	}
	m = step(t, m, keyMsg("up"))    // recall older
	if m.filter.Value() != "older" {
		t.Errorf("filter after second up = %q, want older", m.filter.Value())
	}
	m = step(t, m, keyMsg("up"))    // already at oldest — should clamp
	if m.filter.Value() != "older" {
		t.Errorf("filter clamped at oldest = %q, want older", m.filter.Value())
	}
}

func TestTUI_FilterDownReturnsToLive(t *testing.T) {
	withTempHistoryFile(t)
	m := New(testRegistry(t))
	m.history = []string{"older", "newer"}
	m = step(t, m, keyMsg("enter"))
	m = step(t, m, keyMsg("/"))
	m = step(t, m, keyMsg("up"))   // → newer
	m = step(t, m, keyMsg("up"))   // → older
	m = step(t, m, keyMsg("down")) // → newer
	if m.filter.Value() != "newer" {
		t.Errorf("after down: %q, want newer", m.filter.Value())
	}
	m = step(t, m, keyMsg("down")) // → live (cleared)
	if m.filter.Value() != "" {
		t.Errorf("after second down: %q, want empty (live)", m.filter.Value())
	}
}

func TestTUI_HistoryPersistsToDisk(t *testing.T) {
	path := withTempHistoryFile(t)
	m := New(testRegistry(t))
	m = step(t, m, keyMsg("enter"))
	m = step(t, m, keyMsg("/"))
	m = step(t, m, keyMsg("z"))
	m = step(t, m, keyMsg("enter"))

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read history file: %v", err)
	}
	if !strings.Contains(string(data), "z") {
		t.Errorf("history file should contain z; got %q", data)
	}
}

func TestTUI_TypingClearsHistoryRecall(t *testing.T) {
	withTempHistoryFile(t)
	m := New(testRegistry(t))
	m.history = []string{"older", "newer"}
	m = step(t, m, keyMsg("enter"))
	m = step(t, m, keyMsg("/"))
	m = step(t, m, keyMsg("up")) // recall newer
	if m.historyIdx == -1 {
		t.Errorf("after up, historyIdx should be set; got -1")
	}
	m = step(t, m, keyMsg("x")) // typing exits recall mode
	if m.historyIdx != -1 {
		t.Errorf("typing should reset historyIdx to -1; got %d", m.historyIdx)
	}
}
