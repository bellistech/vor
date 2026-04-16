package cscore

import (
	"encoding/json"
	"testing"
)

func TestRelatedJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
		wantCount int
	}{
		{"bash has related", "bash", false, 1},       // zsh (fish not in test registry)
		{"zsh has related", "zsh", false, 1},          // bash
		{"empty", "", true, 0},
		{"nonexistent", "nonexistent", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RelatedJSON(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
				return
			}
			related, ok := data["related"].([]any)
			if !ok {
				t.Fatal("expected related array")
			}
			if len(related) != tt.wantCount {
				t.Errorf("expected %d related, got %d", tt.wantCount, len(related))
			}
		})
	}
}
