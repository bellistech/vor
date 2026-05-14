package math

import (
	"strings"
	"testing"
)

func TestPreprocess(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		// --- No-op fast path ---
		{"empty", "", ""},
		{"plain prose", "hello world", "hello world"},
		{"no dollars in markdown", "# Heading\n\nText with `code`.\n", "# Heading\n\nText with `code`.\n"},

		// --- Inline math ---
		{"inline simple variable", `Let $x$ be a number.`, "Let x be a number."},
		{"inline two vars", `if $\gcd(a, n) = 1$, then...`, "if gcd(a, n) = 1, then..."},
		{"inline bmod", `compute $x \bmod n$`, "compute x mod n"},
		{"inline cdot", `$a \cdot b$`, "a · b"},
		{"inline equiv pmod", `$x \equiv y \pmod{n}$`, "x ≡ y (mod n)"},
		{"inline frac", `the ratio $\frac{4}{3}$`, "the ratio 4/3"},
		{"inline mathbb Z", `$\mathbb{Z}/n\mathbb{Z}$`, "ℤ/nℤ"},
		{"inline phi", `$\phi(n) = (p-1)(q-1)$`, "φ(n) = (p-1)(q-1)"},
		{"inline approx", `$r \approx 1.333$`, "r ≈ 1.333"},
		{"inline leq", `$|K| \geq 256$`, "|K| ≥ 256"},
		{"inline forall", `$\forall x \in \mathbb{N}$`, "∀ x ∈ ℕ"},
		{"inline arrow", `$f: A \to B$`, "f: A → B"},

		// --- Display math: $$ ... $$ ---
		{"block simple", "$$x = y$$", "\n  x = y\n"},
		{
			"block with frac",
			`$$r = \frac{|encoded|}{|input|} = \frac{4}{3}$$`,
			"\n  r = |encoded|/|input| = 4/3\n",
		},
		{
			"block with sum",
			`$$x = \sum_{i=1}^k a_i$$`,
			"\n  x = Σ_{i=1}^k a_i\n",
		},
		{
			"block with lfloor rfloor",
			`$$q_i = \lfloor r_{i-1} / r_i \rfloor$$`,
			"\n  q_i = ⌊ r_{i-1} / r_i ⌋\n",
		},
		{
			"block HMAC",
			`$$\text{HMAC}(K, m) = H((K \oplus \text{opad}) \| H((K \oplus \text{ipad}) \| m))$$`,
			"\n  HMAC(K, m) = H((K ⊕ opad) ‖ H((K ⊕ ipad) ‖ m))\n",
		},
		{
			"block ECDSA s",
			`$$s = k^{-1}(h + r \cdot d) \bmod n$$`,
			"\n  s = k^{-1}(h + r · d) mod n\n",
		},
		{
			"block CRT sum",
			`$$x = \sum_{i=1}^k a_i M_i y_i$$`,
			"\n  x = Σ_{i=1}^k a_i M_i y_i\n",
		},
		{
			"block sqrt with braces",
			`$$d = \sqrt{x^2 + y^2}$$`,
			"\n  d = √(x^2 + y^2)\n",
		},
		{
			"block error probability",
			`$$\Pr[\text{false positive}] \leq 4^{-k}$$`,
			"\n  Pr[false positive] ≤ 4^{-k}\n",
		},
		{
			"block xrightarrow",
			`$$R_i \xrightarrow{p_{ij}} R_j$$`,
			"\n  R_i ─p_{ij}→ R_j\n",
		},
		{
			"block mathcal",
			`$$\mathcal{A} = (\mathcal{D}, \mathcal{T}, \mathcal{M})$$`,
			"\n  A = (D, T, M)\n",
		},

		// --- Code-block safety: must NOT touch fenced content ---
		{
			"fenced code with $$ stays intact",
			"```sql\nCREATE FUNCTION foo() RETURNS void AS $$\nBEGIN\n  RAISE NOTICE 'x';\nEND;\n$$ LANGUAGE plpgsql;\n```\n",
			"```sql\nCREATE FUNCTION foo() RETURNS void AS $$\nBEGIN\n  RAISE NOTICE 'x';\nEND;\n$$ LANGUAGE plpgsql;\n```\n",
		},
		{
			"fenced code with $PATH stays intact",
			"```bash\necho $PATH\n```\n",
			"```bash\necho $PATH\n```\n",
		},
		{
			"fenced shell with shell-variable then prose with math",
			"```sh\necho $HOME\n```\n\nlet $x$ be a value.\n",
			"```sh\necho $HOME\n```\n\nlet x be a value.\n",
		},

		// --- Mixed flow ---
		{
			"prose with inline and following block",
			"Given $a$ and $b$:\n\n$$\\gcd(a,b) = r_k$$\n\nDone.",
			"Given a and b:\n\n\n  gcd(a,b) = r_k\n\n\nDone.",
		},
		{
			"two consecutive blocks",
			"$$x = 1$$\n$$y = 2$$",
			"\n  x = 1\n\n\n  y = 2\n",
		},

		// --- Realistic samples drawn from real detail pages ---
		{
			"jwt base64url ratio",
			`$$r = \frac{|\text{encoded}|}{|\text{input}|} = \frac{4}{3} \approx 1.333$$`,
			"\n  r = |encoded|/|input| = 4/3 ≈ 1.333\n",
		},
		{
			"number-theory CRT statement",
			`$$\mathbb{Z}/M\mathbb{Z} \to \mathbb{Z}/m_1\mathbb{Z} \times \cdots \times \mathbb{Z}/m_k\mathbb{Z}$$`,
			"\n  ℤ/Mℤ → ℤ/m_1ℤ × ⋯ × ℤ/m_kℤ\n",
		},
		{
			"miller-rabin condition",
			`$$a^{2^r d} \equiv -1 \pmod{n}$$`,
			"\n  a^{2^r d} ≡ -1 (mod n)\n",
		},

		// --- Idempotency: running twice equals running once ---
		{"idempotent on plain", "no math here", "no math here"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Preprocess(tc.in)
			if got != tc.want {
				t.Errorf("\ninput:    %q\nexpected: %q\ngot:      %q", tc.in, tc.want, got)
			}
		})
	}
}

func TestPreprocessIdempotent(t *testing.T) {
	inputs := []string{
		`Let $x$ be a value.`,
		`$$x = y$$`,
		`The ratio $\frac{4}{3} \approx 1.333$.`,
		"```sql\nAS $$ ... $$\n```\n",
	}
	for _, in := range inputs {
		first := Preprocess(in)
		second := Preprocess(first)
		if first != second {
			t.Errorf("not idempotent for %q:\n  first:  %q\n  second: %q", in, first, second)
		}
	}
}

func TestPreprocessNoLatexCanary(t *testing.T) {
	// After preprocessing real-world math, none of the LaTeX canaries used by
	// scripts/audit-display.sh should remain. If they do, the audit will
	// keep failing even after the fix.
	canaries := []string{
		`\bmod`, `\cdot`, `\equiv`, `\sum`, `\sqrt`, `\frac`, `\mathbb`,
		`\pmod`, `\lfloor`, `\rfloor`, `\lceil`, `\rceil`, `\gcd`, `\phi`,
		`\oplus`, `\Rightarrow`, `\rightarrow`, `\leftarrow`,
		`\text{`, `\mathcal`, `\mathit`, `\mathrm`,
	}
	samples := []string{
		`$\gcd(a,n) = 1$`,
		`$$x \bmod n$$`,
		`$$r = \frac{4}{3}$$`,
		`$$\mathbb{Z}/n\mathbb{Z}$$`,
		`$$\sum_{i=1}^k a_i$$`,
		`$$\sqrt{x^2+y^2}$$`,
		`$$\text{HMAC}(K, m) = H((K \oplus \text{opad}))$$`,
		`$\lfloor r/q \rfloor$`,
		`$$x \equiv y \pmod{n}$$`,
		`$f: A \rightarrow B$`,
	}
	for _, s := range samples {
		out := Preprocess(s)
		for _, c := range canaries {
			if strings.Contains(out, c) {
				t.Errorf("canary %q leaked through Preprocess(%q) = %q", c, s, out)
			}
		}
		if strings.Contains(out, "$$") {
			t.Errorf("raw $$ left in Preprocess(%q) = %q", s, out)
		}
	}
}

func TestPreprocessFenceSafety(t *testing.T) {
	// Code blocks must pass through unchanged. This is a stricter check
	// than the table tests above: we hash the input and expect the same
	// substring to survive in the output.
	in := "```sql\nCREATE FUNCTION f() AS $$\nSELECT 1;\n$$ LANGUAGE sql;\n```"
	out := Preprocess(in)
	if out != in {
		t.Errorf("fence content mutated:\n  in:  %q\n  out: %q", in, out)
	}
}
