package calc

import (
	"math"
	"testing"
)

func TestEval(t *testing.T) {
	tests := []struct {
		expr string
		want float64
	}{
		{"2+2", 4},
		{"10 - 3", 7},
		{"3 * 4", 12},
		{"15 / 3", 5},
		{"10 % 3", 1},
		{"2 ** 10", 1024},
		{"(2 + 3) * 4", 20},
		{"-5 + 3", -2},
		{"0xFF", 255},
		{"0b1010", 10},
		{"0o77", 63},
		{"0xFF + 1", 256},
		{"pi", math.Pi},
		{"e", math.E},
		{"sqrt(144)", 12},
		{"abs(-42)", 42},
		{"2 ** 3 ** 2", 512}, // right-associative: 2^(3^2)=2^9=512
		{"1.5 * 2", 3},
		{"1e3 + 1", 1001},
	}

	for _, tt := range tests {
		got, err := Eval(tt.expr)
		if err != nil {
			t.Errorf("Eval(%q) error: %v", tt.expr, err)
			continue
		}
		if math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("Eval(%q) = %g, want %g", tt.expr, got, tt.want)
		}
	}
}

func TestEvalErrors(t *testing.T) {
	errors := []string{
		"1 / 0",
		"1 % 0",
		"(1 + 2",
		"",
		"abc",
	}
	for _, expr := range errors {
		_, err := Eval(expr)
		if err == nil {
			t.Errorf("Eval(%q) expected error, got nil", expr)
		}
	}
}

func TestFormat(t *testing.T) {
	out := Format(255)
	if out == "" {
		t.Fatal("Format returned empty string")
	}
	// Should contain hex
	if !contains(out, "0xFF") {
		t.Errorf("Format(255) missing hex, got: %s", out)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestEvalWithUnits(t *testing.T) {
	tests := []struct {
		expr     string
		wantVal  float64
		wantUnit string
	}{
		{"10GB / 2", 5e9, "data"},
		{"1KiB", 1024, "data"},
		{"10Gbps / 8", 1.25e9, "rate"},
		{"500ms * 2", 1, "time"},
	}

	for _, tt := range tests {
		got, err := EvalWithUnits(tt.expr)
		if err != nil {
			t.Errorf("EvalWithUnits(%q) error: %v", tt.expr, err)
			continue
		}
		if math.Abs(got.Value-tt.wantVal)/math.Max(1, math.Abs(tt.wantVal)) > 0.001 {
			t.Errorf("EvalWithUnits(%q) value = %g, want %g", tt.expr, got.Value, tt.wantVal)
		}
		if got.Unit != tt.wantUnit {
			t.Errorf("EvalWithUnits(%q) unit = %q, want %q", tt.expr, got.Unit, tt.wantUnit)
		}
	}
}
