package cscore

import (
	"encoding/json"
	"sync"
	"testing"
)

func TestBookmarkToggle(t *testing.T) {
	initTestRegistry()
	dir := t.TempDir()
	SetDataDir(dir)

	// Toggle on
	result := BookmarkToggle("bash")
	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	if data["bookmarked"] != true {
		t.Error("expected bookmarked=true after first toggle")
	}

	// Toggle off
	result = BookmarkToggle("bash")
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if data["bookmarked"] != false {
		t.Error("expected bookmarked=false after second toggle")
	}
}

func TestBookmarkToggle_Invalid(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name  string
		topic string
	}{
		{"empty", ""},
		{"path traversal", "../etc/passwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BookmarkToggle(tt.topic)
			var data map[string]any
			json.Unmarshal([]byte(result), &data)
			if _, ok := data["error"]; !ok {
				t.Error("expected error field")
			}
		})
	}
}

func TestBookmarkList(t *testing.T) {
	initTestRegistry()
	dir := t.TempDir()
	SetDataDir(dir)

	BookmarkToggle("bash")
	BookmarkToggle("curl")

	result := BookmarkList()
	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	marks, ok := data["bookmarks"].([]any)
	if !ok {
		t.Fatal("expected bookmarks array")
	}
	if len(marks) != 2 {
		t.Errorf("expected 2 bookmarks, got %d", len(marks))
	}
}

func TestBookmarkIsStarred(t *testing.T) {
	initTestRegistry()
	dir := t.TempDir()
	SetDataDir(dir)

	if BookmarkIsStarred("bash") {
		t.Error("bash should not be starred initially")
	}
	BookmarkToggle("bash")
	if !BookmarkIsStarred("bash") {
		t.Error("bash should be starred after toggle")
	}
}

func TestBookmarkConcurrent(t *testing.T) {
	initTestRegistry()
	dir := t.TempDir()
	SetDataDir(dir)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			BookmarkToggle("bash")
		}()
	}
	wg.Wait()
	// Should not panic or corrupt data
	result := BookmarkList()
	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON after concurrent access: %v", err)
	}
}
