package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSearchJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		query     string
		wantCount bool // expect at least 1 result
		wantError bool
	}{
		{"exact match", "curl", true, false},
		{"case insensitive", "CURL", true, false},
		{"partial", "Variable", true, false},
		{"empty returns all", "", true, false},
		{"no match", "zzzznonexistentzzzz", false, false},
		{"too long", strings.Repeat("q", 513), false, true},
		{"null byte", "test\x00evil", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SearchJSON(tt.query)
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
			results, ok := data["results"].([]any)
			if !ok {
				t.Fatal("expected results array")
			}
			if tt.wantCount && len(results) == 0 {
				t.Error("expected at least 1 result")
			}
			if !tt.wantCount && len(results) > 0 {
				t.Errorf("expected 0 results, got %d", len(results))
			}
		})
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"bash", "bash", 0},
		{"bash", "bsh", 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_"+tt.b, func(t *testing.T) {
			got := levenshtein(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}
