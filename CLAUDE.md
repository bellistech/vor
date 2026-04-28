# Vör — Cheatsheet CLI (binary `vor`, legacy alias `cs`)

Single-binary Go CLI with 814 embedded markdown cheatsheets and 737 deep-dive theory pages across 63 categories. Built-in calculator (unit-aware), subnet calculator, fuzzy search, interactive TUI, REST API daemon, shell completions, bookmarks, cross-references, export, learning paths, math verification. Covers 11 certification domains (CCNP DC/Enterprise, CCIE EI/SP/Security/Automation, JNCIE-SP/SEC, Linux+, CISSP, C|RAGE) plus the `ramp-up/` curriculum (55 ELI5-voiced sheets — kernel, all major network protocols, security/auth, observability, IaC, CI/CD, languages, databases, web servers, cloud, daily-use tools, fundamental networking concepts, network automation).

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
- `internal/secrets/` — opt-in credential loader (env-or-file at `~/.config/cs/secrets.env`, mode-0600 warn, redaction helper) for bonus features only; default offline path never touches it
- `internal/stackoverflow/` — optional Stack Exchange API client + 24h disk cache for the `-so` / `--stack-overflow` bonus flag; gated on `STACK_OVERFLOW_API_KEY`; default `vor` invocation never invokes this package
- `internal/tui/` — interactive TUI (bubbletea + bubbles, category browser, fuzzy filter, content viewer)
- `cmd/cs/main.go` — CLI entry point, stdlib `flag`, REST API server
- `sheets/<category>/<topic>.md` — 814 embedded cheatsheets across 63 categories
- `sheets/ramp-up/<topic>-eli5.md` — narrative-shaped ELI5 ramp-up curriculum (one comprehensive sheet per topic; 55 topics as of S5)
- `detail/<category>/<topic>.md` — 737 deep-dive theory/math pages
- `scripts/audit-see-also.sh` — gate that detects broken `## See Also` references; wired into `make lint` (`make audit-see-also-strict` for the un-allowlisted view)
- `.ci/see-also-allowlist.txt` — pre-S2 broken-ref baseline; future drift is detected

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

## Verbosity Bias

**Sheets should err on the VERBOSE side, not the lean side.** A 2500-line sheet that thoroughly covers every operational corner is preferred over a 1500-line sheet that hits the bare DoD. The North Star ("never leave the terminal to web-search") is better served by exhaustive coverage than by tight prose.

Concretely:
- DoD line targets are **floors, not ceilings**. Hitting 1500 is acceptable; hitting 2200+ is better.
- When agents land at-or-near the floor, pad inline with extended worked examples, more error messages, more vocabulary entries, more diagnostic recipes.
- When dispatching agents, suggest **stretch targets** (e.g. "≥1500 lines, lean toward 2000+") not just floors.
- Repetition for emphasis is fine. Same fact stated three different ways across different sections beats one terse statement that requires the reader to remember it elsewhere.

## Agent Dispatch Discipline (Stuck Protocol)

When a content-writing agent stalls (600s watchdog, no progress, partial response):

**DO NOT retry with a leaner prompt.** That pattern fails ~50% of the time on long-form content (1500+ line targets).

**DO break the work into chunks** — dispatch N parallel agents, each writing ONE section to a separate file (e.g. `/tmp/<topic>-eli5.part1.md`, `.part2.md`, `.part3.md`), then concatenate them into the target file with `cat ... > sheets/ramp-up/<topic>-eli5.md`.

Pattern:
```
Agent 1: write part 1 (Prerequisites + What Even Is + early sections)        → /tmp/X.part1.md
Agent 2: write part 2 (mid-content sections, the technical heart)            → /tmp/X.part2.md
Agent 3: write part 3 (Common Errors + Hands-On + Confusions)                → /tmp/X.part3.md
Agent 4: write part 4 (Vocabulary table + Try This + See Also + References)  → /tmp/X.part4.md
Then: cat /tmp/X.part{1,2,3,4}.md > sheets/ramp-up/X-eli5.md
```

Each chunk is ~400 lines — well under the agent stall threshold. Total assembled file ≥1500 lines.

The chunked approach is also faster (4 agents in parallel vs 1 retrying serially) and degrades gracefully (if one chunk stalls, the other 3 still ship; you only re-run the missing chunk).

**CRITICAL: wait for ALL completion notifications before `cat`-concatenating.** A file appearing on disk does not mean its agent has finished — agents stream output, so `wc -l /tmp/X.part1.md` mid-write reports a partial count. Concatenating before notification lost ~125 lines on `network-automation-eli5` (committed 2f34e17 to recover). Rule: only concat once every chunk agent has emitted its `<task-notification status=completed>`. The end-of-file marker (e.g. `<!-- chunk1 end -->`) is your visual confirmation the chunk is whole; verify with `tail -1 /tmp/X.partN.md` for each before concat.
