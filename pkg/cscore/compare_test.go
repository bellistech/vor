package cscore

import (
	"encoding/json"
	"testing"
)

func TestCompareJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		a, b      string
		wantError bool
	}{
		{"valid", "bash", "curl", false},
		{"same topic", "bash", "bash", false},
		{"empty a", "", "curl", true},
		{"empty b", "bash", "", true},
		{"nonexistent a", "nonexistent", "curl", true},
		{"nonexistent b", "bash", "nonexistent", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CompareJSON(tt.a, tt.b)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
			} else {
				if data["a"] == nil || data["b"] == nil {
					t.Error("expected a and b fields")
				}
			}
		})
	}
}

func TestCompareJSON_Structure(t *testing.T) {
	initTestRegistry()
	result := CompareJSON("bash", "curl")
	var resp compareResponse
	if err := json.Unmarshal([]byte(result), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.A.Name != "bash" {
		t.Errorf("A.Name = %q, want bash", resp.A.Name)
	}
	if resp.B.Name != "curl" {
		t.Errorf("B.Name = %q, want curl", resp.B.Name)
	}
	if len(resp.AllSections) == 0 {
		t.Error("expected all_sections")
	}
}
