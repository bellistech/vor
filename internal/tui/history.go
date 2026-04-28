package tui

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// historyMax caps the persisted filter-history length. Older entries roll off.
const historyMax = 50

// historyFile returns the path to the TUI filter-history cache file. Empty
// string if HOME is unavailable (mobile sandboxes set this directly).
func historyFile() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".cache", "cs", "tui-history")
}

// historyFileOverride lets tests redirect the path. Empty = use default.
var historyFileOverride string

func resolvedHistoryFile() string {
	if historyFileOverride != "" {
		return historyFileOverride
	}
	return historyFile()
}

// LoadHistory reads the persisted filter-history from disk, newest-last.
// Returns nil on any error — history is best-effort, never fatal.
func LoadHistory() []string {
	path := resolvedHistoryFile()
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var entries []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			entries = append(entries, line)
		}
	}
	if len(entries) > historyMax {
		entries = entries[len(entries)-historyMax:]
	}
	return entries
}

// SaveHistory writes the history list to disk via temp+rename for atomicity.
// Caps at historyMax entries (oldest dropped). Best-effort — returns error
// for the caller to log but errors are not fatal to the TUI.
func SaveHistory(entries []string) error {
	path := resolvedHistoryFile()
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if len(entries) > historyMax {
		entries = entries[len(entries)-historyMax:]
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tui-history-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	w := bufio.NewWriter(tmp)
	for _, e := range entries {
		w.WriteString(e)
		w.WriteByte('\n')
	}
	if err := w.Flush(); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// pushHistory appends entry to history with these rules:
//
//  1. Empty / whitespace-only entries are dropped.
//  2. Duplicate of the most recent entry is collapsed (no consecutive dupes).
//  3. The list is capped at historyMax (oldest entries roll off).
//
// Returns the updated slice; caller should reassign.
func pushHistory(history []string, entry string) []string {
	entry = strings.TrimSpace(entry)
	if entry == "" {
		return history
	}
	if n := len(history); n > 0 && history[n-1] == entry {
		return history
	}
	history = append(history, entry)
	if len(history) > historyMax {
		history = history[len(history)-historyMax:]
	}
	return history
}
