package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCategoriesJSON(t *testing.T) {
	initTestRegistry()
	result := CategoriesJSON()

	var cats []categorySummary
	if err := json.Unmarshal([]byte(result), &cats); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	if len(cats) != 2 {
		t.Errorf("expected 2 categories, got %d", len(cats))
	}
	// Should be sorted by name
	for i := 1; i < len(cats); i++ {
		if cats[i].Name < cats[i-1].Name {
			t.Errorf("not sorted: %s before %s", cats[i-1].Name, cats[i].Name)
		}
	}
}

func TestCategoryTopicsJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
		wantCount int
	}{
		{"valid shell", "shell", false, 2},
		{"valid networking", "networking", false, 1},
		{"empty", "", true, 0},
		{"nonexistent", "nonexistent", false, 0},
		{"path traversal", "../hack", true, 0},
		{"too long", strings.Repeat("c", 129), true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CategoryTopicsJSON(tt.input)
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
			topics, ok := data["topics"].([]any)
			if !ok {
				t.Fatal("expected topics array")
			}
			if len(topics) != tt.wantCount {
				t.Errorf("expected %d topics, got %d", tt.wantCount, len(topics))
			}
		})
	}
}
