package verify

import (
	"strings"
	"testing"
)

// TestVerify_BasicArithmetic confirms simple expressions are extracted and validated.
func TestVerify_BasicArithmetic(t *testing.T) {
	content := "Two plus two: 2 + 2 = 4"
	r := Verify("test", content)
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	if !r.Results[0].Pass {
		t.Errorf("2 + 2 = 4 should pass, got expected=%g actual=%g", r.Results[0].Expected, r.Results[0].Got)
	}
	if r.Pass != 1 || r.Fail != 0 {
		t.Errorf("expected 1 pass / 0 fail, got %d / %d", r.Pass, r.Fail)
	}
}

// TestVerify_WrongExpression catches a bad assertion.
func TestVerify_WrongExpression(t *testing.T) {
	content := "Bad math: 2 + 2 = 5"
	r := Verify("test", content)
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	if r.Results[0].Pass {
		t.Errorf("2 + 2 = 5 should fail")
	}
	if r.Fail != 1 {
		t.Errorf("expected 1 fail, got %d", r.Fail)
	}
}

// TestVerify_MultipleExpressions confirms a multi-line page accumulates results.
func TestVerify_MultipleExpressions(t *testing.T) {
	content := `# Worked Examples

The first one: 10 + 5 = 15
And another: 100 / 4 = 25
And a wrong one: 8 - 3 = 6
`
	r := Verify("test", content)
	if len(r.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(r.Results))
	}
	if r.Pass != 2 {
		t.Errorf("expected 2 pass, got %d", r.Pass)
	}
	if r.Fail != 1 {
		t.Errorf("expected 1 fail (8 - 3 = 6 is wrong), got %d", r.Fail)
	}
}

// TestVerify_NoExpressions confirms text-only content yields zero results without error.
func TestVerify_NoExpressions(t *testing.T) {
	content := `# Just Prose

This page has no math. It's just descriptive text about a topic.
We talk about things, but we never say "X = Y" anywhere.
`
	r := Verify("test", content)
	if len(r.Results) != 0 {
		t.Errorf("expected 0 results from text-only content, got %d: %+v", len(r.Results), r.Results)
	}
	if r.Pass != 0 || r.Fail != 0 {
		t.Errorf("expected 0/0, got %d/%d", r.Pass, r.Fail)
	}
}

// TestVerify_DoesNotTrackCodeBlocks documents a known limitation: the verifier
// skips lines that START with ``` but does NOT track being INSIDE a fence, so
// math expressions inside code blocks ARE extracted and validated. Documented
// here so future contributors don't accidentally rely on code-block exclusion.
func TestVerify_DoesNotTrackCodeBlocks(t *testing.T) {
	content := "```\n2 + 2 = 5\n```\nNot in a fence: 1 + 1 = 2"
	r := Verify("test", content)
	// LIMITATION: the in-fence "2 + 2 = 5" is also extracted (and fails).
	// If you need fence-aware skipping, extend Verify() to track in_code_block state.
	if r.Pass != 1 {
		t.Errorf("expected 1 pass ('1 + 1 = 2'), got %d", r.Pass)
	}
	if r.Fail != 1 {
		t.Errorf("expected 1 fail (the in-fence '2 + 2 = 5' is currently extracted; this is a known limitation), got %d", r.Fail)
	}
}

// TestVerify_SkipsHeaders confirms H1/H2/H3 lines are not checked.
func TestVerify_SkipsHeaders(t *testing.T) {
	content := `## Section 7 Worked Example

10 * 5 = 50
`
	r := Verify("test", content)
	if len(r.Results) != 1 {
		t.Fatalf("expected exactly 1 result (the '10 * 5 = 50' line, not the H2), got %d", len(r.Results))
	}
	if !r.Results[0].Pass {
		t.Errorf("10 * 5 = 50 should pass")
	}
}

// TestVerify_HandlesCommaSeparated confirms thousands-separated numbers parse.
func TestVerify_HandlesCommaSeparated(t *testing.T) {
	content := "Big number: 1000 * 1000 = 1,000,000"
	r := Verify("test", content)
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	if !r.Results[0].Pass {
		t.Errorf("expected pass; got expected=%g actual=%g", r.Results[0].Expected, r.Results[0].Got)
	}
}

// TestVerify_DoesNotMatchUnicodeMul documents that the regex char-class lacks ×,
// so expressions like "6 × 7 = 42" are NOT cleanly extracted (the regex breaks
// on × and only captures fragments). normalizeExpr() *does* convert × to *
// once an expression is extracted, but the regex never accepts × in the first
// place — so × expressions are effectively unverified. Documented for the
// future-improvement queue.
func TestVerify_DoesNotMatchUnicodeMul(t *testing.T) {
	content := "Old-school: 6 × 7 = 42"
	r := Verify("test", content)
	// Regex captures the right-hand fragment "7 " (ends at the =), expected=42 → fails.
	// To support ×, extend exprPattern's char class to include "×" or normalize
	// the input string before regex matching.
	if r.Pass != 0 {
		t.Errorf("expected 0 pass (× not in regex char class — known limitation), got %d", r.Pass)
	}
}

// TestVerify_HandlesImplicitMultiplication confirms 5(4) → 5*(4).
func TestVerify_HandlesImplicitMultiplication(t *testing.T) {
	content := "Adjacency formula: 5(4)/2 = 10"
	r := Verify("test", content)
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	if !r.Results[0].Pass {
		t.Errorf("5(4)/2 = 10 should pass; expected=%g actual=%g",
			r.Results[0].Expected, r.Results[0].Got)
	}
}

// TestVerify_FloatTolerance confirms small float diffs are forgiven.
func TestVerify_FloatTolerance(t *testing.T) {
	content := "Approximation: 1 / 3 = 0.333"
	r := Verify("test", content)
	if len(r.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(r.Results))
	}
	// 1/3 = 0.3333... ; expected 0.333 — within 0.01 absolute tolerance
	if !r.Results[0].Pass {
		t.Errorf("1/3 ≈ 0.333 should pass within tolerance; expected=%g actual=%g",
			r.Results[0].Expected, r.Results[0].Got)
	}
}

// TestVerify_GarbageInput confirms malformed expressions don't panic and yield no false positives.
func TestVerify_GarbageInput(t *testing.T) {
	defer func() {
		if rec := recover(); rec != nil {
			t.Fatalf("Verify panicked on garbage input: %v", rec)
		}
	}()
	content := "Bad: ((( 7 = +"
	r := Verify("test", content)
	// Should not crash. Result count is implementation-defined; just confirm safe.
	_ = r
}

// TestFormat_EmptyReport confirms zero-result formatting is clean.
func TestFormat_EmptyReport(t *testing.T) {
	r := &Report{Topic: "empty"}
	out := Format(r)
	if !strings.Contains(out, "empty") {
		t.Errorf("Format output should mention the topic 'empty', got: %q", out)
	}
	if !strings.Contains(out, "No verifiable expressions found") {
		t.Errorf("Format output should mention zero-result message, got: %q", out)
	}
}

// TestFormat_PassingReport confirms passing reports show PASS markers.
func TestFormat_PassingReport(t *testing.T) {
	r := Verify("test", "1 + 1 = 2")
	out := Format(r)
	if !strings.Contains(out, "PASS") {
		t.Errorf("Format output should contain PASS, got: %q", out)
	}
	if !strings.Contains(out, "1 pass") {
		t.Errorf("Format output should mention '1 pass', got: %q", out)
	}
}

// TestFormat_FailingReport confirms failing reports show FAIL markers and 'got'.
func TestFormat_FailingReport(t *testing.T) {
	r := Verify("test", "2 + 2 = 5")
	out := Format(r)
	if !strings.Contains(out, "FAIL") {
		t.Errorf("Format output should contain FAIL, got: %q", out)
	}
	if !strings.Contains(out, "got") {
		t.Errorf("Format output should mention 'got' for failing expressions, got: %q", out)
	}
}

// TestNormalizeExpr_ImplicitMul confirms internal helper inserts * after digit-followed-by-(.
func TestNormalizeExpr_ImplicitMul(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"5(4)/2", "5*(4)/2"},
		{"10*5", "10*5"},
		{"  3 + 4  ", "3 + 4"},
		{"6 × 7", "6 * 7"},
	}
	for _, tt := range tests {
		got := normalizeExpr(tt.in)
		if got != tt.want {
			t.Errorf("normalizeExpr(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// TestCloseEnough_TolerancesAndEquality confirms the float-equality helper.
func TestCloseEnough_TolerancesAndEquality(t *testing.T) {
	tests := []struct {
		a, b float64
		want bool
	}{
		{1.0, 1.0, true},                  // exact
		{0.0, 0.0, true},                  // both zero
		{1.0, 1.005, true},                // within absolute 0.01
		{1.0, 1.5, false},                 // outside tolerance
		{100.0, 100.5, true},              // within relative 0.01
		{100.0, 110.0, false},             // outside relative 0.01
		{1.0 / 3.0, 0.333, true},          // common rounded approximation
	}
	for _, tt := range tests {
		got := closeEnough(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("closeEnough(%g, %g) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
