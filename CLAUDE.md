# Vör — Cheatsheet CLI (binary `vor`, legacy alias `cs`)

Single-binary Go CLI with 772 embedded markdown cheatsheets and 722 deep-dive theory pages across 63 categories. Built-in calculator (unit-aware), subnet calculator, fuzzy search, interactive TUI, REST API daemon, shell completions, bookmarks, cross-references, export, learning paths, math verification. Covers 11 certification domains (CCNP DC/Enterprise, CCIE EI/SP/Security/Automation, JNCIE-SP/SEC, Linux+, CISSP, C|RAGE) plus the `ramp-up/` curriculum (15 ELI5-voiced sheets and growing — kernel, networking protocols, security, observability).

## North Star

> **cs, the cheat sheet application, should have the ability to avoid leaving terminal while working on a system to open a web browser and search Google.**

Every sheet must be powerful enough that a terminal-bound developer never needs to open a web browser or web-search for routine work in that language/tool. If a section makes the reader think "I'd better google this," it's incomplete.

Concretely, every sheet ships against this bar:

- **Self-contained** — never "see the docs for X"; include the syntax, the flags, the error message, the fix.
- **Standard-library coverage** — not a tutorial; the *useful* corners of the stdlib named with one-line summaries and a snippet.
- **Ecosystem tools** — debugger, profiler, formatter, linter, package manager flags and idioms in-sheet.
- **Common error messages** — the exact text the compiler/runtime emits, with the canonical fix.
- **Version differences** — anything that changed in the last 3 major releases gets a version note.
- **CLI flags** — every tool's most-used flags listed with what they do.
- **Cross-link densely** — `See Also` connects every sheet to its neighbours so navigation stays in-terminal.
- **Render check** — open a couple of sections in `cs` after writing; if a section is unreadable in a terminal the sheet failed.

## Build

```bash
make build          # build ./cs binary
make install        # install to /usr/local/bin
make test           # go test ./... -count=1 -race
make lint           # go vet + staticcheck
make fmt            # gofmt -s -w .
```

## Architecture

- `sheets.go` — root-level `go:embed sheets/*/*.md` + `go:embed detail/*/*.md`
- `internal/registry/` — Sheet struct (with SeeAlso, Prerequisites, Complexity fields), parsing, search, filtering, fuzzy match, Related(), SeeAlsoCoverage()
- `internal/render/` — glamour terminal rendering, TTY detection, pager, PlainOutput for piping
- `internal/custom/` — user overlay sheets from `~/.config/cs/sheets/`
- `internal/calc/` — expression calculator (arithmetic, hex/oct/bin, bitwise ops, unit-aware: KB/MB/GB/Gbps/ms)
- `internal/subnet/` — CIDR subnet calculator (IPv4 + IPv6)
- `internal/bookmarks/` — bookmark management (`~/.config/cs/bookmarks.json`)
- `internal/verify/` — math verification for detail pages (parses expressions, evaluates via calc)
- `internal/tui/` — interactive TUI (bubbletea + bubbles, category browser, fuzzy filter, content viewer)
- `cmd/cs/main.go` — CLI entry point, stdlib `flag`, REST API server
- `sheets/<category>/<topic>.md` — 772 embedded cheatsheets across 63 categories
- `sheets/ramp-up/<topic>-eli5.md` — narrative-shaped ELI5 ramp-up curriculum (one comprehensive sheet per topic)
- `detail/<category>/<topic>.md` — 722 deep-dive theory/math pages

## Adding Sheets

1. Create `sheets/<category>/<topic>.md`
2. Format: H1 = title, one-liner, H2 = sections, H3 = subsections, bash code blocks
3. Include `## See Also` with related topic names (must resolve to existing sheets — no dangling refs)
4. Include `## References` section with official docs, RFCs, man pages
5. Optionally create `detail/<category>/<topic>.md` for deep dive
6. Rebuild: `make build`
7. **Apply the North Star checklist** — every snippet paste-and-runnable, exact error/flag/API names, broken-then-fixed pairs in gotchas, version notes on recently-changed features.

## Conventions

- Go 1.24, deps: glamour, x/term, bubbletea, bubbles
- No zerolog — simple stderr for errors
- No cobra — stdlib flag
- Build flags: `-trimpath -s -w`
- Version injection: `-X main.version=$(VERSION)`
- REST API uses stdlib net/http (no external router)
