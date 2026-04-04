// Package verify validates mathematical expressions in detail pages
// by parsing worked examples and checking them against the calculator.
package verify

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/bellistech/cs/internal/calc"
)

// Result represents the verification of a single expression.
type Result struct {
	Expression string
	Expected   float64
	Got        float64
	Pass       bool
	Line       string
}

// Report holds verification results for a topic.
type Report struct {
	Topic   string
	Results []Result
	Pass    int
	Fail    int
}

// expression patterns to find in detail pages:
// - "calculation = result" patterns (e.g., "5(4)/2 = 10")
// - table cells with numeric results
var exprPattern = regexp.MustCompile(`\$?(\d[\d\s\.\+\-\*/\(\)\^]+)\s*=\s*([0-9,\.]+)\$?`)

// Verify checks mathematical expressions in a detail page's content.
func Verify(topic, content string) *Report {
	r := &Report{Topic: topic}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		// Skip markdown headers, code blocks, and text-heavy lines
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "```") {
			continue
		}
		if strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "-") || strings.HasPrefix(trimmed, "*") {
			// Check bullet points and blockquotes too, but only if they contain math
			if !strings.Contains(trimmed, "=") || !containsDigit(trimmed) {
				continue
			}
		}

		matches := exprPattern.FindAllStringSubmatch(trimmed, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}
			expr := normalizeExpr(match[1])
			expectedStr := strings.ReplaceAll(match[2], ",", "")

			expected, err := parseFloat(expectedStr)
			if err != nil {
				continue
			}

			got, err := calc.Eval(expr)
			if err != nil {
				continue // skip expressions the calc can't handle
			}

			pass := closeEnough(got, expected)
			result := Result{
				Expression: match[1],
				Expected:   expected,
				Got:        got,
				Pass:       pass,
				Line:       trimmed,
			}
			r.Results = append(r.Results, result)
			if pass {
				r.Pass++
			} else {
				r.Fail++
			}
		}
	}
	return r
}

// Format produces a human-readable report.
func Format(r *Report) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\033[1;32mVerify: %s\033[0m\n\n", r.Topic))

	if len(r.Results) == 0 {
		sb.WriteString("  No verifiable expressions found.\n")
		return sb.String()
	}

	for _, res := range r.Results {
		status := "\033[1;32mPASS\033[0m"
		if !res.Pass {
			status = "\033[1;31mFAIL\033[0m"
		}
		sb.WriteString(fmt.Sprintf("  [%s] %s = %g", status, res.Expression, res.Expected))
		if !res.Pass {
			sb.WriteString(fmt.Sprintf(" (got %g)", res.Got))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(fmt.Sprintf("\n  Results: %d pass, %d fail, %d total\n",
		r.Pass, r.Fail, len(r.Results)))
	return sb.String()
}

func normalizeExpr(s string) string {
	s = strings.TrimSpace(s)
	// Replace × with *
	s = strings.ReplaceAll(s, "×", "*")
	// Replace common fraction notation: N(N-1)/2 → N*(N-1)/2
	// Handle patterns like "5(4)" → "5*(4)"
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		result = append(result, s[i])
		if i+1 < len(s) && isDigit(s[i]) && s[i+1] == '(' {
			result = append(result, '*')
		}
	}
	return string(result)
}

func containsDigit(s string) bool {
	for _, c := range s {
		if c >= '0' && c <= '9' {
			return true
		}
	}
	return false
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var val float64
	_, err := fmt.Sscanf(s, "%f", &val)
	return val, err
}

func closeEnough(a, b float64) bool {
	if a == b {
		return true
	}
	diff := math.Abs(a - b)
	if diff < 0.01 {
		return true
	}
	// Relative tolerance
	max := math.Max(math.Abs(a), math.Abs(b))
	if max == 0 {
		return diff < 0.01
	}
	return diff/max < 0.01
}
