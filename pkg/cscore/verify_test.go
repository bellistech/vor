package cscore

import (
	"encoding/json"
	"testing"
)

func TestVerifyJSON(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid with expressions", "bash", false},
		{"no detail", "curl", true},
		{"empty", "", true},
		{"nonexistent", "nonexistent", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VerifyJSON(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
			} else {
				if data["topic"] != "bash" {
					t.Errorf("topic = %v, want bash", data["topic"])
				}
			}
		})
	}
}
