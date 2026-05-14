// Package math pre-processes LaTeX-flavoured markdown into terminal-readable
// Unicode before glamour (CLI) or goldmark (iOS) sees it. glamour and goldmark
// have no math extension, so $...$ and $$...$$ otherwise leak verbatim into
// rendered output — see ADR-0006 for the design rationale.
package math

import (
	"regexp"
	"strings"
)

// Preprocess rewrites LaTeX math in md to Unicode prose:
//
//   - $$ ... $$  → 2-space-indented block, on its own line, with LaTeX
//     commands substituted to Unicode.
//   - $ ... $    → inline Unicode replacement, dollars stripped.
//   - LaTeX commands inside math segments (\bmod, \cdot, \frac, \mathbb, …)
//     map to Unicode per latexCmd / structural rewrites below.
//
// Fenced code blocks (``` … ```) are passed through untouched: shell
// variables, PostgreSQL DELIMITER $$, etc. stay intact.
//
// Pure ASCII paragraphs with no $ are a no-op early-exit; the function is
// safe to call on every render.
func Preprocess(md string) string {
	if !strings.ContainsRune(md, '$') {
		return md
	}
	segs := splitFences(md)
	for i := 0; i < len(segs); i += 2 {
		segs[i] = processProse(segs[i])
	}
	return strings.Join(segs, "")
}

// splitFences alternates prose / fenced / prose / fenced / ... segments.
// Even-indexed entries are prose (safe to transform); odd-indexed entries
// include the opening + closing ``` lines and everything between.
func splitFences(s string) []string {
	lines := strings.Split(s, "\n")
	var out []string
	var buf strings.Builder
	inFence := false
	for i, line := range lines {
		isFence := strings.HasPrefix(strings.TrimLeft(line, " \t"), "```")
		if isFence && !inFence {
			out = append(out, buf.String())
			buf.Reset()
			inFence = true
			buf.WriteString(line)
			if i < len(lines)-1 {
				buf.WriteString("\n")
			}
		} else if isFence && inFence {
			buf.WriteString(line)
			if i < len(lines)-1 {
				buf.WriteString("\n")
			}
			out = append(out, buf.String())
			buf.Reset()
			inFence = false
		} else {
			buf.WriteString(line)
			if i < len(lines)-1 {
				buf.WriteString("\n")
			}
		}
	}
	if buf.Len() > 0 {
		out = append(out, buf.String())
	}
	if len(out)%2 == 1 {
		out = append(out, "")
	}
	return out
}

var (
	// Single-line $$X$$ only. Multi-line block math (rare: ~2% of corpus,
	// almost all \begin{cases}…\end{cases} forms) is left for the audit to
	// flag manually — see ADR-0006 §"Known limitations".
	blockMath  = regexp.MustCompile(`\$\$([^$\n]+)\$\$`)
	inlineMath = regexp.MustCompile(`\$([^\$\n]+?)\$`)
	// Backtick-inline-code span; content is preserved verbatim.
	backtickSpan = regexp.MustCompile("`[^`\n]*`")

	// Structural rewrites: commands that take {arg} groups.
	// Ordered so the "simple" single-arg commands clear inner braces
	// BEFORE multi-arg or nested commands see them.
	textRE    = regexp.MustCompile(`\\text\{([^{}]*)\}`)
	mathbbRE  = regexp.MustCompile(`\\mathbb\{([A-Za-z])\}`)
	mathcalRE = regexp.MustCompile(`\\mathcal\{([^{}]*)\}`)
	mathrmRE  = regexp.MustCompile(`\\mathrm\{([^{}]*)\}`)
	mathitRE  = regexp.MustCompile(`\\mathit\{([^{}]*)\}`)
	hatRE     = regexp.MustCompile(`\\hat\{([^{}]*)\}`)
	overRE    = regexp.MustCompile(`\\overline\{([^{}]*)\}`)
	pmodRE    = regexp.MustCompile(`\s*\\pmod\{([^{}]*)\}`)
	xrightRE  = regexp.MustCompile(`\\xrightarrow\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	xleftRE   = regexp.MustCompile(`\\xleftarrow\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	fracRE    = regexp.MustCompile(`\\frac\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	sqrtRE    = regexp.MustCompile(`\\sqrt\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`)
	leftRE    = regexp.MustCompile(`\\left([(\[{|.])`)
	rightRE   = regexp.MustCompile(`\\right([)\]}|.])`)

	// Single-pass replacement for simple `\cmd` commands. The regex matches
	// either a letter-only command (\bmod, \cdots) or a one-char command
	// (\,, \;, \|, \_). Lookup is map-based, so order/length collisions
	// (\cdot vs \cdots) disappear — the regex always grabs the longest
	// letter-run, then the map picks the value.
	cmdRE = regexp.MustCompile(`\\([A-Za-z]+|.)`)
)

var mathbbMap = map[string]string{
	"Z": "ℤ", "N": "ℕ", "R": "ℝ", "Q": "ℚ", "C": "ℂ", "P": "ℙ",
	"F": "𝔽", "H": "ℍ", "E": "𝔼",
}

// latexCmd: lookup table for simple `\cmd` → unicode replacements.
// All keys are command names WITHOUT the leading backslash.
var latexCmd = map[string]string{
	// Greek lowercase
	"alpha": "α", "beta": "β", "gamma": "γ", "delta": "δ",
	"epsilon": "ε", "varepsilon": "ε", "zeta": "ζ", "eta": "η",
	"theta": "θ", "vartheta": "ϑ", "iota": "ι", "kappa": "κ",
	"lambda": "λ", "mu": "μ", "nu": "ν", "xi": "ξ",
	"pi": "π", "varpi": "ϖ", "rho": "ρ", "varrho": "ϱ",
	"sigma": "σ", "varsigma": "ς", "tau": "τ", "upsilon": "υ",
	"phi": "φ", "varphi": "φ", "chi": "χ", "psi": "ψ", "omega": "ω",
	// Greek uppercase
	"Gamma": "Γ", "Delta": "Δ", "Theta": "Θ", "Lambda": "Λ",
	"Xi": "Ξ", "Pi": "Π", "Sigma": "Σ", "Upsilon": "Υ",
	"Phi": "Φ", "Psi": "Ψ", "Omega": "Ω",
	// Operators
	"cdot": "·", "cdots": "⋯", "ldots": "…", "vdots": "⋮", "dots": "…",
	"times": "×", "div": "÷", "ast": "∗", "pm": "±", "mp": "∓",
	"oplus": "⊕", "otimes": "⊗", "circ": "∘", "bullet": "•", "star": "⋆",
	"sum": "Σ", "prod": "∏", "int": "∫", "oint": "∮",
	"sqrt": "√", "bmod": "mod", "mod": "mod",
	// Comparisons
	"leqslant": "≤", "geqslant": "≥", "neq": "≠",
	"equiv": "≡", "approx": "≈", "sim": "∼", "simeq": "≃",
	"cong": "≅", "propto": "∝",
	"leq": "≤", "geq": "≥", "le": "≤", "ge": "≥",
	"ll": "≪", "gg": "≫",
	// Set theory
	"in": "∈", "notin": "∉", "ni": "∋",
	"subset": "⊂", "subseteq": "⊆", "supset": "⊃", "supseteq": "⊇",
	"cup": "∪", "cap": "∩", "setminus": "∖", "emptyset": "∅",
	"forall": "∀", "exists": "∃", "nexists": "∄",
	"wedge": "∧", "vee": "∨", "neg": "¬", "lnot": "¬",
	"land": "∧", "lor": "∨",
	// Arrows
	"Leftrightarrow": "⇔", "Rightarrow": "⇒", "Leftarrow": "⇐",
	"leftrightarrow": "↔", "rightarrow": "→", "leftarrow": "←",
	"mapsto": "↦", "to": "→", "gets": "←",
	// Delimiters
	"lfloor": "⌊", "rfloor": "⌋", "lceil": "⌈", "rceil": "⌉",
	"langle": "⟨", "rangle": "⟩",
	"top": "⊤", "bot": "⊥",
	// Calculus / analysis
	"partial": "∂", "nabla": "∇", "infty": "∞",
	// Logic
	"therefore": "∴", "because": "∵",
	"models": "⊨", "vdash": "⊢",
	// Spacing
	"quad": "  ", "qquad": "    ",
	",": " ", ";": " ", ":": " ", "!": "",
	// Functions
	"gcd": "gcd", "lcm": "lcm", "log": "log", "ln": "ln",
	"exp": "exp", "sin": "sin", "cos": "cos", "tan": "tan",
	"min": "min", "max": "max", "sup": "sup", "inf": "inf",
	"det": "det", "dim": "dim", "arg": "arg", "deg": "deg",
	"Pr": "Pr", "Adv": "Adv",
	// Escapes (\\| etc.)
	"|": "‖", "#": "#", "$": "$", "%": "%", "&": "&",
	"_": "_", "{": "{", "}": "}",
}

func processProse(s string) string {
	if !strings.ContainsRune(s, '$') {
		return s
	}

	// Walk line-by-line so we can skip Make recipe lines (tab-indented) and
	// indented code blocks (4+ leading spaces). Inline backtick spans are
	// stashed before regex passes and restored afterwards to keep `$$VAR`
	// references in tables and inline code from looking like block math.
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		if isCodeLine(ln) {
			continue
		}
		if !strings.ContainsRune(ln, '$') {
			continue
		}
		lines[i] = processLine(ln)
	}
	return strings.Join(lines, "\n")
}

// isCodeLine returns true for lines that markdown treats as code:
// tab-indented (Make recipes, some indented blocks) or four-space
// indented after a blank line. We use a conservative heuristic: any
// line with a leading tab, or 4+ leading spaces, is treated as code.
func isCodeLine(ln string) bool {
	if strings.HasPrefix(ln, "\t") {
		return true
	}
	if strings.HasPrefix(ln, "    ") {
		// Only treat as code if the indent really does precede non-blank
		// content (avoid swallowing wrapped prose that happens to start
		// with four spaces — unlikely, but cheap to check).
		return strings.TrimSpace(ln) != ""
	}
	return false
}

func processLine(ln string) string {
	// Stash inline backtick spans so $$VAR$$ inside `code` survives.
	var stash []string
	ln = backtickSpan.ReplaceAllStringFunc(ln, func(m string) string {
		stash = append(stash, m)
		return "\x00BTCK\x00"
	})

	// Stash escaped dollars (\$ inside math means "literal $"). Without
	// this the inlineMath regex stops at \$ and leaves a stranded "$" plus
	// unprocessed LaTeX commands behind it (seen in lattice-crypto's
	// `$A \xleftarrow{\$} \mathbb{Z}_q^n$` form).
	ln = strings.ReplaceAll(ln, `\$`, "\x00ESCDLR\x00")

	ln = blockMath.ReplaceAllStringFunc(ln, func(m string) string {
		inner := m[2 : len(m)-2]
		inner = strings.TrimSpace(inner)
		inner = substitute(inner)
		return "\n  " + inner + "\n"
	})

	ln = inlineMath.ReplaceAllStringFunc(ln, func(m string) string {
		inner := m[1 : len(m)-1]
		return substitute(inner)
	})

	// Restore escaped dollars as literal "$".
	ln = strings.ReplaceAll(ln, "\x00ESCDLR\x00", "$")
	// Restore backtick spans.
	for _, s := range stash {
		ln = strings.Replace(ln, "\x00BTCK\x00", s, 1)
	}
	return ln
}

// substitute rewrites LaTeX commands inside a math segment.
// Structural commands (those with {arg} captures) run first, so their
// inner braces don't confuse later regex passes.
func substitute(s string) string {
	// 1) Single-arg structural commands clear their inner braces.
	s = textRE.ReplaceAllString(s, "$1")
	s = mathbbRE.ReplaceAllStringFunc(s, func(m string) string {
		sub := mathbbRE.FindStringSubmatch(m)
		if v, ok := mathbbMap[sub[1]]; ok {
			return v
		}
		return sub[1]
	})
	s = mathcalRE.ReplaceAllString(s, "$1")
	s = mathrmRE.ReplaceAllString(s, "$1")
	s = mathitRE.ReplaceAllString(s, "$1")
	s = hatRE.ReplaceAllString(s, "$1̂")
	s = overRE.ReplaceAllString(s, "$1̄")
	s = pmodRE.ReplaceAllString(s, " (mod $1)")
	s = xrightRE.ReplaceAllString(s, "─$1→")
	s = xleftRE.ReplaceAllString(s, "←$1─")

	// 2) Multi-arg structural commands. Run twice so nested \frac{\frac{}}{...}
	//    gets a chance to resolve.
	for i := 0; i < 2; i++ {
		s = fracRE.ReplaceAllString(s, "$1/$2")
		s = sqrtRE.ReplaceAllString(s, "√($1)")
	}

	// 3) \left( / \right) — strip the size-modifier, keep the delimiter.
	s = leftRE.ReplaceAllString(s, "$1")
	s = rightRE.ReplaceAllString(s, "$1")

	// 4) Simple commands via single-pass regex + map lookup.
	s = cmdRE.ReplaceAllStringFunc(s, func(m string) string {
		// m is "\name" — strip the leading backslash
		name := m[1:]
		if v, ok := latexCmd[name]; ok {
			return v
		}
		return m
	})

	// 5) Tidy: collapse empty braces left over.
	s = strings.ReplaceAll(s, "{}", "")

	return s
}
