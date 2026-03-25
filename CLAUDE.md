# cs — Cheatsheet CLI

Single-binary Go CLI with 200 embedded markdown cheatsheets across 32 categories. Built-in calculator and subnet calculator, fuzzy search, shell completions. Better than man pages.

## Build

```bash
make build          # build ./cs binary
make install        # install to /usr/local/bin
make test           # go test ./... -count=1 -race
make lint           # go vet + staticcheck
make fmt            # gofmt -s -w .
```

## Architecture

- `sheets.go` — root-level `go:embed sheets/*/*.md`
- `internal/registry/` — Sheet struct, parsing, search, filtering, fuzzy match (prefix → substring → Levenshtein)
- `internal/render/` — glamour terminal rendering, TTY detection, pager, PlainOutput for piping
- `internal/custom/` — user overlay sheets from `~/.config/cs/sheets/`
- `internal/calc/` — expression calculator (arithmetic, hex/oct/bin, bitwise ops)
- `internal/subnet/` — CIDR subnet calculator
- `cmd/cs/main.go` — CLI entry point, stdlib `flag`
- `sheets/<category>/<topic>.md` — 200 embedded cheatsheets across 32 categories

## Adding Sheets

1. Create `sheets/<category>/<topic>.md`
2. Format: H1 = title, one-liner, H2 = sections, H3 = subsections, bash code blocks
3. Include `## References` section with official docs, RFCs, man pages
4. Rebuild: `make build`

## Conventions

- Go 1.24, minimal deps (glamour + x/term)
- No zerolog — simple stderr for errors
- No cobra — stdlib flag
- Build flags: `-trimpath -s -w`
- Version injection: `-X main.version=$(VERSION)`
