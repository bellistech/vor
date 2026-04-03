// Package calc implements a simple arithmetic expression evaluator.
// Supports: +, -, *, /, % (mod), ** (power), parentheses, negative numbers.
// Also provides hex (0x), octal (0o), binary (0b) literal parsing and
// base conversion output.
package calc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"
)

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

// parser is a recursive descent parser for arithmetic expressions.
type parser struct {
	input string
	pos   int
	err   error
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
	return val
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}
