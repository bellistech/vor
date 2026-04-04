// Package calc implements a simple arithmetic expression evaluator.
// Supports: +, -, *, /, % (mod), ** (power), parentheses, negative numbers.
// Also provides hex (0x), octal (0o), binary (0b) literal parsing and
// base conversion output. Supports unit suffixes (KB, MB, GB, Gbps, ms, etc.).
package calc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

// Unit represents a dimensional unit.
type Unit struct {
	Name   string
	Factor float64
	Kind   string // "data", "rate", "time", "none"
}

var unitTable = map[string]Unit{
	// Data units (base-10)
	"b":     {Name: "B", Factor: 1, Kind: "data"},
	"bytes": {Name: "B", Factor: 1, Kind: "data"},
	"byte":  {Name: "B", Factor: 1, Kind: "data"},
	"kb":    {Name: "KB", Factor: 1e3, Kind: "data"},
	"mb":    {Name: "MB", Factor: 1e6, Kind: "data"},
	"gb":    {Name: "GB", Factor: 1e9, Kind: "data"},
	"tb":    {Name: "TB", Factor: 1e12, Kind: "data"},
	"pb":    {Name: "PB", Factor: 1e15, Kind: "data"},
	// Data units (base-2)
	"kib": {Name: "KiB", Factor: 1024, Kind: "data"},
	"mib": {Name: "MiB", Factor: 1024 * 1024, Kind: "data"},
	"gib": {Name: "GiB", Factor: 1024 * 1024 * 1024, Kind: "data"},
	"tib": {Name: "TiB", Factor: 1024 * 1024 * 1024 * 1024, Kind: "data"},
	// Network rate
	"bps":  {Name: "bps", Factor: 1, Kind: "rate"},
	"kbps": {Name: "Kbps", Factor: 1e3, Kind: "rate"},
	"mbps": {Name: "Mbps", Factor: 1e6, Kind: "rate"},
	"gbps": {Name: "Gbps", Factor: 1e9, Kind: "rate"},
	"tbps": {Name: "Tbps", Factor: 1e12, Kind: "rate"},
	// Time
	"ns":  {Name: "ns", Factor: 1e-9, Kind: "time"},
	"us":  {Name: "μs", Factor: 1e-6, Kind: "time"},
	"ms":  {Name: "ms", Factor: 1e-3, Kind: "time"},
	"s":   {Name: "s", Factor: 1, Kind: "time"},
	"sec": {Name: "s", Factor: 1, Kind: "time"},
	"min": {Name: "min", Factor: 60, Kind: "time"},
	"hr":  {Name: "hr", Factor: 3600, Kind: "time"},
}

// UnitResult holds a value with optional unit info.
type UnitResult struct {
	Value float64
	Unit  string // empty if dimensionless
}

// Eval evaluates an arithmetic expression and returns the result.
func Eval(expr string) (float64, error) {
	p := &parser{input: expr}
	result := p.parseExpr()
	if p.err != nil {
		return 0, p.err
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return 0, fmt.Errorf("unexpected character at position %d: '%c'", p.pos, p.input[p.pos])
	}
	return result, nil
}

// EvalWithUnits evaluates an expression that may contain unit suffixes.
func EvalWithUnits(expr string) (UnitResult, error) {
	p := &parser{input: expr, unitAware: true}
	result := p.parseExpr()
	if p.err != nil {
		return UnitResult{}, p.err
	}
	p.skipSpaces()
	if p.pos < len(p.input) {
		return UnitResult{}, fmt.Errorf("unexpected character at position %d: '%c'", p.pos, p.input[p.pos])
	}
	return UnitResult{Value: result, Unit: p.resultUnit}, nil
}

// Format formats a result for display, showing integer form when possible
// plus hex/oct/bin conversions for integer values.
func Format(val float64) string {
	var sb strings.Builder

	if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) {
		i := int64(val)
		sb.WriteString(fmt.Sprintf("  = %d", i))
		if i >= 0 && i <= math.MaxInt64 {
			sb.WriteString(fmt.Sprintf("\n  hex  0x%X", i))
			sb.WriteString(fmt.Sprintf("\n  oct  0o%o", i))
			sb.WriteString(fmt.Sprintf("\n  bin  0b%b", i))
		}
	} else {
		sb.WriteString(fmt.Sprintf("  = %g", val))
	}
	return sb.String()
}

// FormatWithUnit formats a unit-aware result.
func FormatWithUnit(r UnitResult) string {
	if r.Unit == "" {
		return Format(r.Value)
	}

	var sb strings.Builder
	// Auto-scale the result to best unit
	scaled, unit := autoScale(r.Value, r.Unit)
	if scaled == math.Trunc(scaled) && !math.IsInf(scaled, 0) {
		sb.WriteString(fmt.Sprintf("  = %s %s", formatInt(int64(scaled)), unit))
	} else {
		sb.WriteString(fmt.Sprintf("  = %s %s", formatFloat(scaled), unit))
	}

	// Also show raw value
	if r.Value == math.Trunc(r.Value) && !math.IsInf(r.Value, 0) {
		sb.WriteString(fmt.Sprintf("\n  raw  %s", formatInt(int64(r.Value))))
	} else {
		sb.WriteString(fmt.Sprintf("\n  raw  %s", formatFloat(r.Value)))
	}
	return sb.String()
}

func autoScale(val float64, unit string) (float64, string) {
	abs := math.Abs(val)
	switch unit {
	case "data":
		switch {
		case abs >= 1e15:
			return val / 1e15, "PB"
		case abs >= 1e12:
			return val / 1e12, "TB"
		case abs >= 1e9:
			return val / 1e9, "GB"
		case abs >= 1e6:
			return val / 1e6, "MB"
		case abs >= 1e3:
			return val / 1e3, "KB"
		default:
			return val, "B"
		}
	case "rate":
		switch {
		case abs >= 1e12:
			return val / 1e12, "Tbps"
		case abs >= 1e9:
			return val / 1e9, "Gbps"
		case abs >= 1e6:
			return val / 1e6, "Mbps"
		case abs >= 1e3:
			return val / 1e3, "Kbps"
		default:
			return val, "bps"
		}
	case "time":
		switch {
		case abs >= 3600:
			return val / 3600, "hr"
		case abs >= 60:
			return val / 60, "min"
		case abs >= 1:
			return val, "s"
		case abs >= 1e-3:
			return val * 1e3, "ms"
		case abs >= 1e-6:
			return val * 1e6, "μs"
		default:
			return val * 1e9, "ns"
		}
	}
	return val, unit
}

func formatInt(i int64) string {
	s := fmt.Sprintf("%d", i)
	if len(s) <= 3 {
		return s
	}
	// Add thousand separators
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	if neg {
		return "-" + string(result)
	}
	return string(result)
}

func formatFloat(f float64) string {
	return fmt.Sprintf("%.4g", f)
}

// parser is a recursive descent parser for arithmetic expressions.
type parser struct {
	input      string
	pos        int
	err        error
	unitAware  bool
	resultUnit string
}

func (p *parser) skipSpaces() {
	for p.pos < len(p.input) && unicode.IsSpace(rune(p.input[p.pos])) {
		p.pos++
	}
}

func (p *parser) peek() byte {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) next() byte {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		return 0
	}
	ch := p.input[p.pos]
	p.pos++
	return ch
}

// parseExpr handles addition and subtraction (lowest precedence).
func (p *parser) parseExpr() float64 {
	if p.err != nil {
		return 0
	}
	left := p.parseTerm()
	for {
		switch p.peek() {
		case '+':
			p.next()
			left += p.parseTerm()
		case '-':
			p.next()
			left -= p.parseTerm()
		default:
			return left
		}
	}
}

// parseTerm handles multiplication, division, and modulo.
func (p *parser) parseTerm() float64 {
	if p.err != nil {
		return 0
	}
	left := p.parsePower()
	for {
		switch p.peek() {
		case '*':
			p.next()
			if p.peek() == '*' {
				// oops, this is ** (power), put back
				p.pos--
				return left
			}
			left *= p.parsePower()
		case '/':
			p.next()
			right := p.parsePower()
			if right == 0 {
				p.err = fmt.Errorf("division by zero")
				return 0
			}
			left /= right
		case '%':
			p.next()
			right := p.parsePower()
			if right == 0 {
				p.err = fmt.Errorf("modulo by zero")
				return 0
			}
			left = math.Mod(left, right)
		default:
			return left
		}
	}
}

// parsePower handles exponentiation (right-associative).
func (p *parser) parsePower() float64 {
	if p.err != nil {
		return 0
	}
	base := p.parseUnary()
	p.skipSpaces()
	if p.pos+1 < len(p.input) && p.input[p.pos] == '*' && p.input[p.pos+1] == '*' {
		p.pos += 2
		exp := p.parsePower() // right-associative
		return math.Pow(base, exp)
	}
	return base
}

// parseUnary handles unary plus/minus.
func (p *parser) parseUnary() float64 {
	if p.err != nil {
		return 0
	}
	switch p.peek() {
	case '+':
		p.next()
		return p.parseUnary()
	case '-':
		p.next()
		return -p.parseUnary()
	default:
		return p.parseAtom()
	}
}

// parseAtom handles numbers, parentheses, and constants.
func (p *parser) parseAtom() float64 {
	if p.err != nil {
		return 0
	}

	// Parentheses
	if p.peek() == '(' {
		p.next()
		val := p.parseExpr()
		if p.next() != ')' {
			p.err = fmt.Errorf("missing closing parenthesis")
			return 0
		}
		return val
	}

	// Named constants / functions
	p.skipSpaces()
	if p.pos < len(p.input) && unicode.IsLetter(rune(p.input[p.pos])) {
		start := p.pos
		for p.pos < len(p.input) && (unicode.IsLetter(rune(p.input[p.pos])) || unicode.IsDigit(rune(p.input[p.pos]))) {
			p.pos++
		}
		name := strings.ToLower(p.input[start:p.pos])

		// Check if it's a unit (only in unit-aware mode and only after a number)
		if p.unitAware {
			if _, ok := unitTable[name]; ok {
				// This is a unit suffix — but we got here from a letter start
				// which means it's not after a number. Put it back.
				p.pos = start
				p.err = fmt.Errorf("unexpected unit without number: %s", name)
				return 0
			}
		}

		switch name {
		case "pi":
			return math.Pi
		case "e":
			return math.E
		case "sqrt":
			if p.peek() != '(' {
				p.err = fmt.Errorf("sqrt requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Sqrt(val)
		case "abs":
			if p.peek() != '(' {
				p.err = fmt.Errorf("abs requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Abs(val)
		case "log":
			if p.peek() != '(' {
				p.err = fmt.Errorf("log requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Log10(val)
		case "ln":
			if p.peek() != '(' {
				p.err = fmt.Errorf("ln requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Log(val)
		case "log2":
			if p.peek() != '(' {
				p.err = fmt.Errorf("log2 requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Log2(val)
		case "ceil":
			if p.peek() != '(' {
				p.err = fmt.Errorf("ceil requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Ceil(val)
		case "floor":
			if p.peek() != '(' {
				p.err = fmt.Errorf("floor requires parentheses")
				return 0
			}
			p.next()
			val := p.parseExpr()
			if p.next() != ')' {
				p.err = fmt.Errorf("missing closing parenthesis")
				return 0
			}
			return math.Floor(val)
		default:
			p.err = fmt.Errorf("unknown function or constant: %s", name)
			return 0
		}
	}

	// Number literals (decimal, hex, octal, binary)
	return p.parseNumber()
}

func (p *parser) parseNumber() float64 {
	p.skipSpaces()
	if p.pos >= len(p.input) {
		p.err = fmt.Errorf("unexpected end of expression")
		return 0
	}

	start := p.pos

	// Check for 0x, 0o, 0b prefixes
	if p.pos+1 < len(p.input) && p.input[p.pos] == '0' {
		prefix := p.input[p.pos+1]
		switch prefix {
		case 'x', 'X':
			p.pos += 2
			hexStart := p.pos
			for p.pos < len(p.input) && isHexDigit(p.input[p.pos]) {
				p.pos++
			}
			if p.pos == hexStart {
				p.err = fmt.Errorf("invalid hex literal")
				return 0
			}
			val, err := strconv.ParseInt(p.input[hexStart:p.pos], 16, 64)
			if err != nil {
				p.err = fmt.Errorf("invalid hex literal: %s", p.input[start:p.pos])
				return 0
			}
			return float64(val)
		case 'o', 'O':
			p.pos += 2
			octStart := p.pos
			for p.pos < len(p.input) && p.input[p.pos] >= '0' && p.input[p.pos] <= '7' {
				p.pos++
			}
			if p.pos == octStart {
				p.err = fmt.Errorf("invalid octal literal")
				return 0
			}
			val, err := strconv.ParseInt(p.input[octStart:p.pos], 8, 64)
			if err != nil {
				p.err = fmt.Errorf("invalid octal literal: %s", p.input[start:p.pos])
				return 0
			}
			return float64(val)
		case 'b', 'B':
			p.pos += 2
			binStart := p.pos
			for p.pos < len(p.input) && (p.input[p.pos] == '0' || p.input[p.pos] == '1') {
				p.pos++
			}
			if p.pos == binStart {
				p.err = fmt.Errorf("invalid binary literal")
				return 0
			}
			val, err := strconv.ParseInt(p.input[binStart:p.pos], 2, 64)
			if err != nil {
				p.err = fmt.Errorf("invalid binary literal: %s", p.input[start:p.pos])
				return 0
			}
			return float64(val)
		}
	}

	// Decimal number (integer or float)
	for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9') {
		p.pos++
	}
	if p.pos < len(p.input) && p.input[p.pos] == '.' {
		p.pos++
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9') {
			p.pos++
		}
	}
	// Scientific notation
	if p.pos < len(p.input) && (p.input[p.pos] == 'e' || p.input[p.pos] == 'E') {
		p.pos++
		if p.pos < len(p.input) && (p.input[p.pos] == '+' || p.input[p.pos] == '-') {
			p.pos++
		}
		for p.pos < len(p.input) && (p.input[p.pos] >= '0' && p.input[p.pos] <= '9') {
			p.pos++
		}
	}

	if p.pos == start {
		p.err = fmt.Errorf("expected number at position %d", p.pos)
		return 0
	}

	val, err := strconv.ParseFloat(p.input[start:p.pos], 64)
	if err != nil {
		p.err = fmt.Errorf("invalid number: %s", p.input[start:p.pos])
		return 0
	}

	// Check for unit suffix in unit-aware mode
	if p.unitAware && p.pos < len(p.input) && unicode.IsLetter(rune(p.input[p.pos])) {
		unitStart := p.pos
		for p.pos < len(p.input) && unicode.IsLetter(rune(p.input[p.pos])) {
			p.pos++
		}
		unitName := strings.ToLower(p.input[unitStart:p.pos])
		if u, ok := unitTable[unitName]; ok {
			val *= u.Factor
			if p.resultUnit == "" {
				p.resultUnit = u.Kind
			}
		} else {
			// Not a known unit, put it back
			p.pos = unitStart
		}
	}

	return val
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
