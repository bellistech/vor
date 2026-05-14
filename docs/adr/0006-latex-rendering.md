# ADR-0006 — LaTeX Math Rendering Strategy for Terminal & WebView

**Status:** Accepted — Implemented V1 in Tier 1.5 sprint (2026-05-14). Unicode-substitution preprocessor shipped at `internal/render/math/preprocess.go`. Multi-line `\begin{cases}` support deferred to V2.
**Date:** 2026-05-13 (proposed) → 2026-05-14 (accepted + V1 shipped)
**Deciders:** stevie@bellis.tech
**Tags:** rendering, terminal, glamour, goldmark, latex, north-star, implemented

## Context

Vör's 737 detail pages include ~600 with LaTeX math: number theory (`detail/cs-theory/number-theory-crypto.md`), JWT crypto (`detail/security/jwt.md`), AI/ML risk math, network protocols. The source uses standard markdown LaTeX:

```markdown
Given integers $a$ and $b$, compute $\gcd(a, b)$ … if $\gcd(a, n) = 1$, then $x \bmod n$ is $a^{-1} \bmod n$.

$$r_{i+1} = r_{i-1} - q_i \cdot r_i$$
$$\text{HMAC}(K, m) = H((K \oplus \text{opad}) \| H((K \oplus \text{ipad}) \| m))$$
```

Both renderers in the pipeline — charmbracelet/glamour (CLI terminal) and yuin/goldmark (iOS WebView via `pkg/cscore/markdown.go`) — **have no math extension**. They pass `$...$` and `$$...$$` through verbatim. Result: terminal users see `$\bmod$`, `$\cdot$`, `$$...$$` literal in their cheatsheet, directly violating the **North Star** ("avoid leaving terminal to web-search") for the pedagogically densest content in the corpus.

Item #21 (`make audit-display`) confirmed the bug: 17 of 20 sampled detail pages leaked raw LaTeX before this work began.

## Decision

**Accepted (2026-05-14). Implemented V1 as a markdown-preprocess layer in `internal/render/math/preprocess.go`, called from both `internal/render/render.go` (terminal) and `pkg/cscore/markdown.go` (iOS WebView).**

Pre-substitute LaTeX commands to Unicode equivalents **before** the markdown renderer sees the content. Single shared Go package — terminal and WebView render identically. No JS dependency, no external math engine, no asset bundle.

KaTeX, MathJax, and full LaTeX-to-Unicode libraries were considered and rejected (see Options table).

## Forces / Constraints

- **North Star:** "never leave the terminal." Math must render readably in a glamour pane — Unicode is the only character set both terminal and HTML can render without dependencies.
- **Paste-ready output:** users copy snippets from `cs -d` into shell. Output must remain selectable text, not images or ANSI art. KaTeX/MathJax both produce HTML — terminal-incompatible.
- **CLI-iOS parity:** the iOS app renders via `cscore.RenderMarkdownToHTML` (goldmark + Chroma). Adding KaTeX to iOS would require shipping the KaTeX JS bundle (~280 KB minified) plus fonts; CLI couldn't use it at all. A second renderer would also fork the output between platforms.
- **Latency budget:** Vör's first-render TTFB is sub-100ms. A LaTeX→Unicode regex pass on a 1500-line file is sub-millisecond. KaTeX in headless mode (via `headless-chrome` or `katex-node`) would add ~50-200ms per page.
- **Offline-by-default:** Vör must run airgapped. No CDN-hosted JS for math.
- **Binary size:** Vör is already 47MB embedded. Adding a JS runtime (~10MB) or extensive math tables (~1MB) is regressive.
- **Source-format stability:** authors should keep writing `$\gcd$` and `$$x = y$$` — the established LaTeX-in-markdown convention. The preprocessor must accept current source unchanged, NOT require source rewrites.

## Options Considered

| Option | Terminal? | iOS? | Offline? | Binary cost | Verdict |
|---|---|---|---|---|---|
| **Unicode substitution preprocessor** (chosen) | ✅ Native | ✅ Native | ✅ | ~6KB Go | **Accepted** |
| **KaTeX (JS) for WebView, leave CLI broken** | ❌ | ✅ | with bundled JS | +280KB JS + fonts | Rejected — splits platforms |
| **KaTeX server-side (headless render)** | ASCII-art only | ✅ as image | ✅ if bundled | +10MB Node runtime | Rejected — Node dep, latency, ugly CLI |
| **MathJax (JS)** | ❌ | ✅ | with bundled JS | +1MB JS + fonts | Rejected — heavier than KaTeX, same drawbacks |
| **Pango / figlet ASCII art** | ✅ as art | ✅ as preformatted text | ✅ | +2MB lib + fonts | Rejected — loses paste-ability, distorts inline flow |
| **Wrap math in `$$ ... $$` blocks and let glamour pass through** (status quo) | Broken | Broken | ✅ | 0 | Rejected — this is the bug |
| **Strip `$` markers, leave LaTeX commands as text** | Marginal improvement | Marginal | ✅ | trivial | Rejected — `\bmod` still illegible |
| **Goldmark extension** (e.g. `mathjax-render-go`) | Same as MathJax | Same | varies | varies | Rejected — most produce HTML or images, not Unicode |
| **External tool: `unicodeit`, `pandoc --to plain`** | ✅ | n/a | requires install | external dep | Rejected — Vör is single-binary by mandate |

## Why Unicode substitution wins

- **One renderer, one output.** The preprocessor sits *before* both glamour and goldmark, so terminal AND WebView see the same Unicode. Source-of-truth lives in Go core.
- **Excellent Unicode coverage for technical math.** Modern fonts cover Greek (α…Ω), set-theory (∈∋⊂⊆∪∩∅∀∃), comparisons (≤≥≠≡≈), arrows (→↔⇒), brackets (⌊⌋⌈⌉⟨⟩), operators (·×÷⊕⊗Σ∏∫√), logic (∧∨¬∴∵), calculus (∂∇∞). What's missing — stacked fractions, true 2D layout — wasn't going to render in a terminal anyway.
- **Paste-friendly.** Output stays as text. `Σ_{i=1}^k a_i M_i y_i` survives copy-paste into another terminal, an editor, a chat client.
- **Zero new dependencies.** Pure Go regex + string substitution. Tests run in 0.2s.
- **No source migration.** Existing 600+ math-bearing detail pages render correctly after the change, with no edits required.

## Architecture (V1 — shipped)

```
internal/render/math/preprocess.go
  ├── Preprocess(md string) string                    — public entry point
  ├── splitFences(s) []string                         — protects ``` … ``` blocks
  ├── processProse(s) string                          — walks lines, skips code
  │   ├── isCodeLine(ln)                              — tab-indent / 4-space indent guard
  │   └── processLine(ln)                             — stashes inline backticks
  └── substitute(s) string                            — regex passes
      ├── structural: \frac, \sqrt, \text, \mathbb,   — capture-group rewrites
      │   \mathcal, \mathrm, \mathit, \hat,
      │   \overline, \pmod, \xrightarrow (1-deep
      │   nesting), \left/\right
      └── cmdRE single-pass regex + latexCmd map      — ~100 Greek + operator entries

internal/render/render.go::Render(content)
  └── content = math.Preprocess(content)              — runs before glamour

pkg/cscore/markdown.go::RenderMarkdownToHTML(md)
  └── md = math.Preprocess(md)                        — runs before goldmark
```

**Skipped contexts** (preserved verbatim):
- Fenced code blocks (```…```)
- Lines starting with `\t` (Make recipes — `$$@`, `$$^` survive)
- Lines indented 4+ spaces (Markdown indented code blocks)
- Inline backtick spans (`` `code` ``) — stashed before regex, restored after

**Single-line scope.** V1 matches `$$X$$` and `$X$` on a single line only. Multi-line `$$\begin{cases}…\end{cases}$$` blocks (~2% of corpus, ~210 instances) are NOT processed — they're allowlisted in `.ci/display-allowlist.txt` for the audit and addressed in V2.

## Consequences

**As shipped (V1):**
- + Two flagship math-heavy pages (`number-theory-crypto`, `jwt`) render zero LaTeX leakage. Inline math reads as natural Unicode prose.
- + Single source of truth: terminal and iOS use identical substitution, no platform drift.
- + No new runtime dependencies, no binary-size regression, no latency penalty.
- + Source files untouched — existing 600+ math-bearing pages work as-is.
- − `\frac{a}{b}` renders as `a/b` (single-line), not as a stacked fraction. Acceptable for terminal; HTML loses the visual richness KaTeX would provide.
- − Multi-line `\begin{cases}` blocks pass through unchanged in V1 (audit-display flags them, allowlisted as known V2 work).
- − `\frac` and similar commands appearing OUTSIDE `$...$` delimiters (raw markdown LaTeX in tables) are not processed — they're an authoring bug, not a rendering bug. The audit surfaces these.

**If we had chosen KaTeX (REJECTED):**
- − CLI terminal would lose math entirely (KaTeX outputs HTML).
- − iOS bundle grows ~280KB minimum.
- − Loses paste-ability — math becomes spans of styled elements.
- − Source-of-truth splits between Go core and JS shim.

**If we had chosen do-nothing (status quo):**
- − 17/20 sampled detail pages remain broken (the bug we set out to fix).

## Known V2 work

1. **Multi-line block math.** Currently `$$\nX\n$$` and `$$X\nY\nZ$$` with `\begin{cases}` only render the opening `$$` line in some cases. A paragraph-aware matcher (`$$...$$` may span lines but not blank lines) would close this gap. ~210 instances in corpus.
2. **Source-level `\frac` in tables.** Some detail pages use raw `\frac` outside `$...$` in markdown table cells. Three paths: (a) fix source by wrapping in `$...$`; (b) extend preprocessor to recognise table-cell context; (c) leave as-is with audit-display warnings. **Preferred: (a)** — keeps the preprocessor simple and source consistent.
3. **`\begin{cases}` → vertical-bar rendering.** When V2 multi-line matching lands, render `\begin{cases} a & x>0 \\ b & x≤0 \end{cases}` as a 2-line block with vertical bar prefix.
4. **Optional: KaTeX for iOS WebView only.** If iOS users want stacked fractions and proper rendering, the WebView path can additionally pipe through KaTeX *after* the Unicode preprocessor. Decision deferred until user demand emerges.

## References

- North Star: [`CLAUDE.md`](../../CLAUDE.md) — `## North Star` section
- Item #21 (audit script that surfaced this bug): [`scripts/audit-display.sh`](../../scripts/audit-display.sh)
- Item #22 implementation: [`internal/render/math/preprocess.go`](../../internal/render/math/preprocess.go)
- Battle plan: [`battle-plan-2026-05-06.md`](../../battle-plan-2026-05-06.md) — Tier 1.5
- charmbracelet/glamour: https://github.com/charmbracelet/glamour
- yuin/goldmark: https://github.com/yuin/goldmark
- KaTeX (rejected option): https://katex.org/
- Unicode Mathematical Operators block: https://www.unicode.org/charts/PDF/U2200.pdf

## Revision History

| Date | Change | Author |
|---|---|---|
| 2026-05-13 | Initial proposal as item #24 in Tier 1.5 of battle-plan-2026-05-06.md | stevie@bellis.tech |
| 2026-05-14 | Accepted + V1 shipped: single-line `$$X$$` / `$X$`, ~100 Unicode substitutions, code-block & backtick safety, integrated into both render paths | stevie@bellis.tech |
