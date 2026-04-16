package cscore

import (
	"encoding/json"
	"testing"
)

func TestLearnPathJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid category", "shell", false},
		{"empty", "", true},
		{"nonexistent", "nonexistent", true},
		{"path traversal", "../hack", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := LearnPathJSON(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
			} else {
				path, ok := data["path"].([]any)
				if !ok {
					t.Fatal("expected path array")
				}
				if len(path) != 2 {
					t.Errorf("expected 2 entries in shell path, got %d", len(path))
				}
			}
		})
	}
}

func TestLearnPathJSON_Order(t *testing.T) {
	initTestRegistry()
	result := LearnPathJSON("shell")
	var resp learnPathResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// zsh has 0 prereqs, bash has 2 prereqs (terminal, linux-basics)
	// so zsh should come first
	if len(resp.Path) >= 2 {
		if resp.Path[0].Name != "zsh" {
			t.Errorf("expected zsh first (0 prereqs), got %s", resp.Path[0].Name)
		}
		if resp.Path[1].Name != "bash" {
			t.Errorf("expected bash second (2 prereqs), got %s", resp.Path[1].Name)
		}
	}
}
