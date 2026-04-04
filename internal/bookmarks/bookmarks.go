package bookmarks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

var bookmarkFile string

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	bookmarkFile = filepath.Join(home, ".config", "cs", "bookmarks.json")
}

// Load returns the current bookmark list.
func Load() []string {
	if bookmarkFile == "" {
		return nil
	}
	data, err := os.ReadFile(bookmarkFile)
	if err != nil {
		return nil
	}
	var marks []string
	json.Unmarshal(data, &marks)
	return marks
}

// Save writes the bookmark list to disk.
func Save(marks []string) error {
	if bookmarkFile == "" {
		return nil
	}
	dir := filepath.Dir(bookmarkFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(marks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(bookmarkFile, data, 0644)
}

// Toggle adds or removes a topic from bookmarks. Returns true if added, false if removed.
func Toggle(topic string) (bool, error) {
	marks := Load()
	for i, m := range marks {
		if m == topic {
			marks = append(marks[:i], marks[i+1:]...)
			return false, Save(marks)
		}
	}
	marks = append(marks, topic)
	sort.Strings(marks)
	return true, Save(marks)
}

// IsBookmarked returns true if a topic is bookmarked.
func IsBookmarked(topic string) bool {
	for _, m := range Load() {
		if m == topic {
			return true
		}
	}
	return false
}

// List returns all bookmarked topics sorted.
func List() []string {
	marks := Load()
	sort.Strings(marks)
	return marks
}
