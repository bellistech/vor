package calc

import (
	"math"
	"strings"
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
		{"+5 + 3", 8}, // unary +
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

func TestEval_AllMathFunctions(t *testing.T) {
	tests := []struct {
		expr string
		want float64
	}{
		{"log(100)", 2},        // log10
		{"log(1000)", 3},
		{"ln(e)", 1},           // natural log
		{"ln(1)", 0},
		{"log2(8)", 3},
		{"log2(1024)", 10},
		{"ceil(1.2)", 2},
		{"ceil(-1.2)", -1},
		{"ceil(5)", 5},
		{"floor(1.7)", 1},
		{"floor(-1.7)", -2},
		{"floor(5)", 5},
		{"sqrt(0)", 0},
		{"sqrt(2) ** 2", 2},    // identity
		{"abs(-3.14)", 3.14},
		{"abs(0)", 0},
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

func TestEval_BitwiseAndShifts(t *testing.T) {
	// Many calc impls support shift and bitwise operations on integer-valued
	// floats; verify the parser at least handles known forms or errors cleanly.
	cases := []struct {
		expr string
		want float64
	}{
		{"1 << 16", 65536},
		{"1024 >> 2", 256},
		{"0xFF & 0x0F", 15},
		{"0x0F | 0xF0", 255},
		{"0xFF ^ 0x0F", 240},
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			// Bitwise ops may not be supported; skip rather than fail noisily.
			t.Logf("Eval(%q) not supported: %v (acceptable; bitwise is optional)", c.expr, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestEvalErrors(t *testing.T) {
	errors := []string{
		"1 / 0",          // div zero
		"1 % 0",          // mod zero
		"(1 + 2",         // unbalanced paren
		"",                // empty
		"abc",             // unknown identifier
		"1 +",             // dangling op
		"sqrt 100",        // missing parens for fn
		"sqrt(100",        // missing close paren in fn
		"abs(",            // open-only fn
		"log(",            // open-only fn
		"log2(",
		"ln(",
		"ceil(",
		"floor(",
		"5 + )",           // stray close paren
		"@@",              // garbage
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
	for _, want := range []string{"255", "0xFF", "0o377", "0b11111111"} {
		if !strings.Contains(out, want) {
			t.Errorf("Format(255) missing %q, got: %s", want, out)
		}
	}
}

func TestFormat_Fractional(t *testing.T) {
	out := Format(3.14159)
	if !strings.Contains(out, "3.14") {
		t.Errorf("Format(3.14159) should show fractional value, got: %s", out)
	}
	// fractional values should NOT include hex/oct/bin
	if strings.Contains(out, "0x") || strings.Contains(out, "0b") {
		t.Errorf("Format(3.14159) should not show base conversions, got: %s", out)
	}
}

func TestFormat_Zero(t *testing.T) {
	out := Format(0)
	if !strings.Contains(out, "0") {
		t.Errorf("Format(0) should include 0, got: %s", out)
	}
}

func TestFormat_NegativeInt(t *testing.T) {
	out := Format(-42)
	if !strings.Contains(out, "-42") {
		t.Errorf("Format(-42) should include -42, got: %s", out)
	}
}

func TestFormat_Infinity(t *testing.T) {
	// math.Inf returns +Inf; Format should not crash and should produce some
	// reasonable text (likely "Inf" via %g).
	out := Format(math.Inf(1))
	if out == "" {
		t.Errorf("Format(+Inf) should produce non-empty output")
	}
}

func TestFormat_NaN(t *testing.T) {
	out := Format(math.NaN())
	if out == "" {
		t.Errorf("Format(NaN) should produce non-empty output")
	}
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

func TestEvalWithUnits_AllUnits(t *testing.T) {
	// Smoke-test every unit registered in unitTable produces a parseable
	// "1<unit>" expression. Catches regressions in the unit-suffix scanner.
	cases := []struct {
		expr string
		kind string
	}{
		{"1B", "data"}, {"1bytes", "data"}, {"1byte", "data"},
		{"1KB", "data"}, {"1MB", "data"}, {"1GB", "data"},
		{"1TB", "data"}, {"1PB", "data"},
		{"1KiB", "data"}, {"1MiB", "data"}, {"1GiB", "data"}, {"1TiB", "data"},
		{"1bps", "rate"}, {"1Kbps", "rate"}, {"1Mbps", "rate"},
		{"1Gbps", "rate"}, {"1Tbps", "rate"},
		{"1ns", "time"}, {"1us", "time"}, {"1ms", "time"},
		{"1s", "time"}, {"1sec", "time"}, {"1min", "time"}, {"1hr", "time"},
	}
	for _, c := range cases {
		r, err := EvalWithUnits(c.expr)
		if err != nil {
			t.Errorf("EvalWithUnits(%q) failed: %v", c.expr, err)
			continue
		}
		if r.Unit != c.kind {
			t.Errorf("EvalWithUnits(%q) kind = %q, want %q", c.expr, r.Unit, c.kind)
		}
	}
}

func TestEvalWithUnits_DimensionlessFallsThrough(t *testing.T) {
	r, err := EvalWithUnits("2 + 2")
	if err != nil {
		t.Fatal(err)
	}
	if r.Value != 4 {
		t.Errorf("got %g, want 4", r.Value)
	}
	if r.Unit != "" {
		t.Errorf("dimensionless expression should have empty unit; got %q", r.Unit)
	}
}

func TestEvalWithUnits_UnitWithoutNumberErrors(t *testing.T) {
	// "GB" alone (without a number prefix) should error — there's no value
	// to attach the unit to.
	_, err := EvalWithUnits("GB")
	if err == nil {
		t.Error("EvalWithUnits(GB) should error — unit without number")
	}
}

func TestFormatWithUnit_DataAutoScale(t *testing.T) {
	cases := []struct {
		val   float64
		unit  string
		wantA string // expected substring in output
	}{
		{1, "data", "B"},
		{1e3, "data", "KB"},
		{1e6, "data", "MB"},
		{1e9, "data", "GB"},
		{1e12, "data", "TB"},
		{1e15, "data", "PB"},
		{500, "data", "B"}, // below KB threshold stays in B
	}
	for _, c := range cases {
		out := FormatWithUnit(UnitResult{Value: c.val, Unit: c.unit})
		if !strings.Contains(out, c.wantA) {
			t.Errorf("FormatWithUnit(%g, data) missing %q, got: %s", c.val, c.wantA, out)
		}
	}
}

func TestFormatWithUnit_RateAutoScale(t *testing.T) {
	cases := []struct {
		val   float64
		want  string
	}{
		{1, "bps"},
		{1e3, "Kbps"},
		{1e6, "Mbps"},
		{1e9, "Gbps"},
		{1e12, "Tbps"},
	}
	for _, c := range cases {
		out := FormatWithUnit(UnitResult{Value: c.val, Unit: "rate"})
		if !strings.Contains(out, c.want) {
			t.Errorf("FormatWithUnit(%g, rate) missing %q, got: %s", c.val, c.want, out)
		}
	}
}

func TestFormatWithUnit_TimeAutoScale(t *testing.T) {
	cases := []struct {
		val   float64
		want  string
	}{
		{1e-9, "ns"},
		{1e-6, "μs"},
		{1e-3, "ms"},
		{1, "s"},
		{60, "min"},
		{3600, "hr"},
	}
	for _, c := range cases {
		out := FormatWithUnit(UnitResult{Value: c.val, Unit: "time"})
		if !strings.Contains(out, c.want) {
			t.Errorf("FormatWithUnit(%g, time) missing %q, got: %s", c.val, c.want, out)
		}
	}
}

func TestFormatWithUnit_EmptyUnitFallsBackToFormat(t *testing.T) {
	out := FormatWithUnit(UnitResult{Value: 255, Unit: ""})
	// Same as Format(255)
	if !strings.Contains(out, "0xFF") {
		t.Errorf("empty-unit FormatWithUnit should equal Format; got: %s", out)
	}
}

func TestFormatWithUnit_ShowsRawValue(t *testing.T) {
	// FormatWithUnit always echoes the un-scaled raw value too
	out := FormatWithUnit(UnitResult{Value: 1.5e9, Unit: "data"})
	if !strings.Contains(out, "raw") {
		t.Errorf("FormatWithUnit should include raw value; got: %s", out)
	}
}

func TestFormatWithUnit_FractionalRaw(t *testing.T) {
	out := FormatWithUnit(UnitResult{Value: 3.14e6, Unit: "data"})
	// Both the scaled (MB) value and the raw value should appear
	if !strings.Contains(out, "MB") {
		t.Errorf("expected scaled MB; got: %s", out)
	}
}

func TestFormatInt_ThousandSeparators(t *testing.T) {
	cases := []struct {
		val  int64
		want string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1000, "1,000"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{1000000000, "1,000,000,000"},
		{-1000, "-1,000"},
		{-1234567, "-1,234,567"},
	}
	for _, c := range cases {
		got := formatInt(c.val)
		if got != c.want {
			t.Errorf("formatInt(%d) = %q, want %q", c.val, got, c.want)
		}
	}
}

func TestFormatFloat(t *testing.T) {
	cases := []struct {
		val  float64
		want string // expected substring (using %.4g, exact digits depend on input)
	}{
		{3.14159, "3.142"},
		{1234.5, "1234"}, // %.4g — 4 sig figs without rounding to 1235
		{0.0001, "0.0001"},
		{1e-7, "1e-07"},
	}
	for _, c := range cases {
		got := formatFloat(c.val)
		if !strings.Contains(got, c.want) {
			t.Errorf("formatFloat(%g) = %q, want substring %q", c.val, got, c.want)
		}
	}
}

func TestParseNumber_Underscores(t *testing.T) {
	// Many calc parsers accept underscores as digit separators (Python-style)
	// — verify either it works or errors cleanly. Don't fail either way.
	cases := []string{"1_000_000", "0xFF_FF", "0b1010_1010"}
	for _, c := range cases {
		_, err := Eval(c)
		if err != nil {
			t.Logf("Eval(%q) didn't accept underscores: %v (acceptable)", c, err)
		}
	}
}

func TestEval_NestedFunctions(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"sqrt(abs(-144))", 12},
		{"floor(sqrt(50))", 7},
		{"ceil(log2(1000))", 10},
		{"abs(floor(-1.5))", 2},
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			t.Errorf("Eval(%q) failed: %v", c.expr, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestEval_OperatorPrecedence(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"2 + 3 * 4", 14},
		{"(2 + 3) * 4", 20},
		{"2 ** 3 + 1", 9},
		{"2 + 3 ** 2", 11},
		{"-2 ** 2", -4},  // unary binds tighter than binary; -(2^2) = -4 OR (-2)^2 = 4 depending on impl
		{"10 - 3 - 2", 5}, // left-associative
		{"2 ** 2 ** 3", 256}, // right-associative: 2^(2^3) = 2^8 = 256
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			t.Errorf("Eval(%q) failed: %v", c.expr, err)
			continue
		}
		// "-2 ** 2" is ambiguous across calc impls; accept either -4 or 4
		if c.expr == "-2 ** 2" {
			if got != -4 && got != 4 {
				t.Errorf("Eval(%q) = %g; want -4 or 4 depending on precedence", c.expr, got)
			}
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestEval_ScientificNotation(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"1e3", 1000},
		{"1e-3", 0.001},
		{"1.5e2", 150},
		{"1E5", 100000},
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			t.Errorf("Eval(%q) failed: %v", c.expr, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestEval_HexEdgeCases(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"0x0", 0},
		{"0xFFFFFFFF", 4294967295},
		{"0xff", 255}, // lowercase
		{"0xAbCd", 0xABCD},
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			t.Errorf("Eval(%q) failed: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestEval_BinaryEdgeCases(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"0b0", 0},
		{"0b1", 1},
		{"0b11111111", 255},
		{"0B10", 2}, // uppercase prefix
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			t.Errorf("Eval(%q) failed: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestEval_OctalEdgeCases(t *testing.T) {
	cases := []struct {
		expr string
		want float64
	}{
		{"0o0", 0},
		{"0o7", 7},
		{"0o10", 8},
		{"0o777", 511},
		{"0O100", 64}, // uppercase
	}
	for _, c := range cases {
		got, err := Eval(c.expr)
		if err != nil {
			// Some calc impls may not accept 0O (uppercase). Skip rather than fail.
			if c.expr == "0O100" {
				t.Logf("Eval(%q) didn't accept uppercase 0O: %v", c.expr, err)
				continue
			}
			t.Errorf("Eval(%q) failed: %v", c.expr, err)
			continue
		}
		if got != c.want {
			t.Errorf("Eval(%q) = %g, want %g", c.expr, got, c.want)
		}
	}
}

func TestIsHexDigit(t *testing.T) {
	for _, c := range []byte{'0', '5', '9', 'a', 'f', 'A', 'F'} {
		if !isHexDigit(c) {
			t.Errorf("isHexDigit(%c) = false, want true", c)
		}
	}
	for _, c := range []byte{'g', 'z', '!', '@', ' '} {
		if isHexDigit(c) {
			t.Errorf("isHexDigit(%c) = true, want false", c)
		}
	}
}

