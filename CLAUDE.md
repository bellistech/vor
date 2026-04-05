# cs — Cheatsheet CLI

Single-binary Go CLI with 685 embedded markdown cheatsheets and 685 deep-dive theory pages across 59 categories. Built-in calculator (unit-aware), subnet calculator, fuzzy search, interactive TUI, REST API daemon, shell completions, bookmarks, cross-references, export, learning paths, math verification. Covers 11 certification domains (CCNP DC/Enterprise, CCIE EI/SP/Security/Automation, JNCIE-SP/SEC, Linux+, CISSP, C|RAGE).

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
- `sheets/<category>/<topic>.md` — 685 embedded cheatsheets across 59 categories
- `detail/<category>/<topic>.md` — 685 deep-dive theory/math pages

## Adding Sheets

1. Create `sheets/<category>/<topic>.md`
2. Format: H1 = title, one-liner, H2 = sections, H3 = subsections, bash code blocks
3. Include `## See Also` with related topic names
4. Include `## References` section with official docs, RFCs, man pages
5. Optionally create `detail/<category>/<topic>.md` for deep dive
6. Rebuild: `make build`

## Conventions

- Go 1.24, deps: glamour, x/term, bubbletea, bubbles
- No zerolog — simple stderr for errors
- No cobra — stdlib flag
- Build flags: `-trimpath -s -w`
- Version injection: `-X main.version=$(VERSION)`
- REST API uses stdlib net/http (no external router)
