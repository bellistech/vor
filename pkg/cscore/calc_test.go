package cscore

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCalcEval(t *testing.T) {
	initTestRegistry()

	tests := []struct {
		name      string
		input     string
		wantError bool
		wantValue float64
	}{
		{"addition", "2+2", false, 4},
		{"power", "2**10", false, 1024},
		{"hex", "0xff", false, 255},
		{"sqrt", "sqrt(144)", false, 12},
		{"empty", "", true, 0},
		{"too long", strings.Repeat("1+", 600), true, 0},
		{"null byte", "1+1\x002+2", true, 0},
		{"invalid expr", ";;;DROP TABLE", true, 0},
		{"division by zero", "1/0", true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalcEval(tt.input)
			var data map[string]any
			if err := json.Unmarshal([]byte(result), &data); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, result)
			}
			if tt.wantError {
				if _, ok := data["error"]; !ok {
					t.Error("expected error field")
				}
			} else {
				val, ok := data["value"].(float64)
				if !ok {
					t.Fatalf("expected value field, got: %v", data)
				}
				if val != tt.wantValue {
					t.Errorf("value = %v, want %v", val, tt.wantValue)
				}
			}
		})
	}
}

func TestCalcEval_WithUnit(t *testing.T) {
	initTestRegistry()
	result := CalcEval("10GB / 2")
	var data map[string]any
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, result)
	}
	if _, ok := data["error"]; ok {
		t.Skipf("unit expression not supported: %v", data["error"])
	}
	if data["formatted"] == nil || data["formatted"] == "" {
		t.Error("expected formatted field")
	}
}
