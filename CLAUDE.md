# cs — Cheatsheet CLI

Single-binary Go CLI with 97 embedded markdown cheatsheets. Better than man pages.

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
- `internal/registry/` — Sheet struct, parsing, search, filtering
- `internal/render/` — glamour terminal rendering, TTY detection, pager
- `internal/custom/` — user overlay sheets from `~/.config/cs/sheets/`
- `cmd/cs/main.go` — CLI entry point, stdlib `flag`
- `sheets/<category>/<topic>.md` — embedded cheatsheets

## Adding Sheets

1. Create `sheets/<category>/<topic>.md`
2. Format: H1 = title, one-liner, H2 = sections, H3 = subsections, bash code blocks
3. Rebuild: `make build`

## Conventions

- Go 1.24, minimal deps (glamour + x/term)
- No zerolog — simple stderr for errors
- No cobra — stdlib flag
- Build flags: `-trimpath -s -w`
- Version injection: `-X main.version=$(VERSION)`
