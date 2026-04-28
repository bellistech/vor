# ADR-0001 — Web Search Fallback for Unmatched Queries

**Status:** Accepted — Implemented in S6-A (2026-04-28). Stack Exchange backend shipped as opt-in `-so` / `--stack-overflow` flag. cheat.sh / Wikipedia / paid-CSE backends explicitly deferred — see "Implementation Notes" below.
**Date:** 2026-04-25 (proposed) → 2026-04-28 (accepted + implemented)
**Deciders:** stevie@bellis.tech
**Tags:** search, north-star, fallback, third-party-api, implemented

## Context

The North Star (committed to `CLAUDE.md`) is:

> *cs, the cheat sheet application, should have the ability to avoid leaving terminal while working on a system to open a web browser and search Google.*

Today's content sprint shipped ~50,000 lines across 25 verbose sheets, but local coverage will always have gaps. When `cs -s <query>` returns nothing useful, the user's only recourse is to leave the terminal and search Google — directly violating the vision. We need a path for the tool to fall through to a *web* search source without breaking the in-terminal contract.

## Decision

**Accepted (2026-04-28). Implemented as `-so` / `--stack-overflow` opt-in flag in S6-A.**

The Stack Exchange API was the leading candidate at the original proposal, and the implementation confirmed that choice. The shipped architecture is **more conservative** than the original proposal — see "Implementation Notes" below for the deviations and why.

Direct Google scraping remains **explicitly rejected**. cheat.sh / Wikipedia / paid-CSE backends are deferred to a possible future S6-B if user demand materializes; nothing from those tiers is required to honor the North Star for the current threat model.

## Forces / Constraints

- **ToS.** Public CLI — every user inherits the legal posture. Google's ToS forbids automated queries; shipping a Google scraper puts users at risk of IP captchas, rate limits, and (in extreme readings) CFAA exposure.
- **Anti-bot reality.** Google fingerprints TLS handshake (JA3/JA4), HTTP/2 SETTINGS frame order, header order, and behavioral cadence. User-Agent spoofing alone is ineffective. Even `curl-impersonate` (Chrome TLS+HTTP fingerprint clone) gets captcha'd from datacenter IPs.
- **Key-distribution problem.** APIs that require keys (Google Custom Search, Brave Search, Kagi) cannot ship a baked-in key — that key gets exhausted, abused, or banned. User must BYO key.
- **Quality of results.** Cheatsheet/programming queries are the dominant use case — generic web search is overkill.
- **Latency budget.** Terminal users expect sub-second feel. Network calls + parsing + render must stay tight.
- **Privacy.** Users running offline, in airgapped environments, or behind corporate proxies should not have their queries silently leaving the box. Fallback must be opt-in or at minimum visible.

## Options Considered

| Option | Free? | Auth? | Result quality for `cs` use case | Verdict |
|---|---|---|---|---|
| **Direct Google scraping** | Free | None (UA spoof) | Best results, but brittle | **Rejected** — ToS violation, captcha'd, user IP burned |
| **`curl-impersonate` against Google** | Free | None | Same as above with extra steps | **Rejected** — same reasons + tooling overhead |
| **Google Custom Search JSON API** | 100/day free, $5/1000 after | API key + `cx` engine ID | Excellent (real Google results) | Viable as opt-in via user-supplied key |
| **Brave Search API** | 2000/month free tier | API key | Good — Brave's own index, generous free tier | Viable as opt-in via user-supplied key |
| **Bing Search API** | n/a | n/a | n/a | **Retired August 2025** |
| **Kagi Search API** | Paid only | API key | Excellent | Viable for paying users |
| **SerpAPI / Bright Data SERP** | Paid | API key | Excellent (legally scrapes Google for you) | Viable but expensive |
| **DuckDuckGo Instant Answer API** | Free | None | Limited — only definitions, calculators | Useful as a *minor* fallback, silent on most queries |
| **DuckDuckGo HTML endpoint** | Free | None | Decent | ToS unclear; brittle |
| **SearXNG (self-hosted meta-search)** | Free | None | Aggregates many engines | Heavy — requires user to run a service |
| **cheat.sh** | Free | None | **Designed for exactly this** — terminal-formatted cheatsheets | **Strong candidate** |
| **Stack Exchange API** | 10K/day per IP | None for read | Excellent for programming questions | **Leading candidate** |
| **Wikipedia REST API** | Free | None | Great for "what is X" definitional queries | Useful complement |
| **tldr-pages** | Free, offline once cached | None | Simplified man pages | Useful complement; offline-friendly |

## Why Stack Exchange API is the leading candidate

- **Free at the read tier**: 10,000 queries/day per IP, no auth required for read-only access.
- **No ToS landmine**: explicitly public, documented for programmatic use (`api.stackexchange.com/docs`).
- **JSON, terminal-friendly**: structured response — easy to render answers as plain text.
- **Signal-to-noise**: questions are voted, accepted answers are gold, results are aligned with `cs`'s programming-and-tools audience.
- **Filters built-in**: search by tag (`[python]`, `[regex]`, etc.) maps cleanly to existing `cs` categories.
- **Stable**: API has been stable for over a decade; Stack Exchange has a strong track record of not breaking it.

`cheat.sh` is the second-best complement: it's curl-native, returns formatted terminal output, and is the canonical "cheatsheet from the wire" service. It covers different ground than SE (recipes/cheats vs. Q&A).

## Proposed Architecture (when work resumes)

```
cs -s <query>
   ├── 1. Local search (registry.Search) → if hits, done
   ├── 2. Bookmarks / custom sheets → if hits, done
   ├── 3. Fallback chain (opt-in via flag or config):
   │      a. Stack Exchange API   (programming, default-on)
   │      b. cheat.sh              (cheatsheet content)
   │      c. Wikipedia summary     (definitions, "what is X")
   │      d. User-configured       (Brave / Google CSE / Kagi via API key)
   └── 4. Cache responses in ~/.cache/cs/ with TTL
```

**UX requirements:**
- Always tag the source: `[stackexchange]`, `[cheat.sh]`, `[wikipedia]`, `[google-cse]`.
- Default behavior: opt-in via `cs --web <query>` or `cs -s --web <query>`. Never silently send queries off-box.
- Config file at `~/.config/cs/config.yaml` for API keys, source preferences, cache TTL.
- `cs --offline` flag for users who want to forbid all network calls.
- Rate-limit per source on the client side (don't blow the SE 10K daily quota in a runaway script).

## Decision Drivers (for the future implementer)

When this is picked up, the order of work:

1. Implement Stack Exchange backend in `internal/search/stackexchange/` — read-only client, `?site=stackoverflow` and `?site=unix.stackexchange` as the default sites, tag-mapping from cs categories.
2. Add `--web` flag to `cs -s`. Default off. Config file flag to make it default-on if user wants.
3. Add response caching in `~/.cache/cs/` keyed by `(source, normalized-query)`.
4. Always show source in output.
5. Then add cheat.sh as second source.
6. Then add Wikipedia summary as third source.
7. Optional API-key sources (Brave, Google CSE, Kagi) last — they need config plumbing.

## Consequences

**If we deferred and never implemented:**
- The North Star's promise has a known gap: missing local content forces a tab-open. Acceptable short-term given today's massive content expansion; not acceptable long-term.

**If we implement as proposed:**
- + The North Star is honored even for queries we don't have local content for.
- + Stack Exchange's 10K/day/IP budget is plenty for individual users.
- + No baked-in keys, no ToS landmines.
- − Network-time latency added to fallback queries (mitigated by cache).
- − A new dimension of failure modes: network errors, API outages, rate limits — all need graceful degradation.
- − Privacy footprint expands: queries leave the box. Must be opt-in and source-tagged.

**If we instead chose Google scraping (REJECTED):**
- − Distributed ToS violation, every user inherits the risk.
- − Brittle — breaks every time Google updates anti-bot.
- − Captcha-burns user IPs in the wild.
- − Gives `cs` a reputation problem.

## Implementation Notes (2026-04-28, S6-A)

The shipped feature **deviates from the original proposed architecture** in three deliberate ways:

### 1. Explicit opt-in per-query, not auto-fallback chain

**Proposed:** `cs -s <query>` falls through `local → bookmarks → web sources → cache` automatically.
**Shipped:** separate `-so` / `--stack-overflow` flag. Default `vor -s <query>` is unchanged — pure offline, never touches the network.

**Why:** the North Star reframed during S5 as *"vör is an offline tech encyclopedia available at the CLI to avoid interrupting workflow to search the web for reference/syntax/etc."* The 813 sheets, 722 detail pages, and 55 ramp-up topics ARE the product. Auto-fallback would dilute that — every `-s` query becomes "did this hit local or did it leak off-box?". Explicit opt-in makes the network call a deliberate user action, not a side-effect.

The user invokes `-so` when they know the offline corpus likely doesn't have what they need (fresh error message, version-specific gotcha, niche stack trace). For everything else, `-s` stays offline.

### 2. Single backend, not the proposed multi-source chain

**Proposed:** Stack Exchange → cheat.sh → Wikipedia → optional paid CSEs, all chained.
**Shipped:** Stack Exchange only. Each additional source would be a separate flag or sub-source: `-so` is read as "stack overflow", not "search online".

**Why:** the original proposal's "fallback chain" assumes the user wants ANY answer. Practical experience says the user usually wants a SPECIFIC kind of answer — for "EADDRINUSE port 8080", Stack Overflow is the right source; for "what is the Halting Problem", Wikipedia is the right source; for `tar` recipes, cheat.sh is the right source. Mixing these in one fallback chain produces noise.

If user demand for cheat.sh / Wikipedia surfaces later, S6-B can ship them as separate flags (`-cs` for cheat.sh, `-wp` for wikipedia) following the same opt-in + key-gated (where applicable) pattern.

### 3. Key-required, not anonymous-by-default

**Proposed:** open with anonymous Stack Exchange (300 req/day per IP) and bump to keyed only on demand.
**Shipped:** key-required. No key configured → friendly nudge and exit 1.

**Why:**
- The 300/day anonymous floor is fragile: a shared NAT (CI runner, university, café) hits it fast and leaves you stuck.
- The key registration flow is genuinely 60 seconds (no OAuth, no callback, no PII) — well-documented in `vor stack-overflow-cli`.
- Forcing the user through key setup once also forces them to acknowledge the privacy trade — the file `~/.config/cs/secrets.env` is a visible artifact of "yes, I opted into network calls."

### What was actually built

```
cmd/vor/main.go
  ├── -so / --stack-overflow flag
  ├── -so help                        — onboarding text
  ├── doStackOverflow(query)          — cache-first dispatch
  └── /api/stackoverflow REST endpoint — Gauntlets-Law parity

internal/secrets/
  ├── Load(name) — env first, ~/.config/cs/secrets.env second
  ├── Redact(s, secret) — error/log scrubbing
  └── File-mode 0600 warning (one-shot via sync.Once)

internal/stackoverflow/
  ├── client.go — HTTPS to api.stackexchange.com/2.3/search/advanced
  │   - URL via url.Values{} (key never string-concat'd into log lines)
  │   - 10s http.Client.Timeout, 12s context timeout in dispatch
  │   - User-Agent: vor-cli/<version>
  │   - pagesize=10 (terminal-friendly)
  │   - filter=withbody (named built-in filter)
  │   - gzip auto-decompression handled by Go's transport (verified by test)
  │   - All error paths run through redactErr() — key cannot leak
  │   - Captures backoff field; surfaced in render footer when non-zero
  ├── render.go — Markdown for the existing glamour pipeline
  │   - "Powered by Stack Exchange" + CC BY-SA 4.0 footer (ToS compliance)
  ├── cache.go — ~/.cache/cs/stackoverflow/<sha256(query)>.json
  │   - 24h TTL (well over the 1/min throttle floor in the API spec)
  │   - Atomic write via temp+rename
  │   - Untrusted-input on read — corrupt entries silently miss
  └── ~85% test coverage, all hermetic via httptest

scripts/audit-secrets.sh  — defense-in-depth source-grep gate
.gitignore                — secrets.env, *.secrets, .envrc additions
sheets/troubleshooting/stack-overflow-cli.md — full setup walkthrough
Makefile                  — make audit-secrets, wired into make lint
```

5 commits: `4a65e8c` → `f8b2be5` → `3150f54` → `453b10c` → `7802495` (post-spec-audit fix).

### What was NOT built (deferred to S6-B if/when needed)

- **cheat.sh backend** — separate flag, no key needed; complements SE for recipe-style queries.
- **Wikipedia REST backend** — separate flag, definitional queries.
- **Multi-site Stack Exchange** (`--site=serverfault`, `--site=unix.stackexchange`) — currently hardcoded to `stackoverflow`.
- **Auto-fallback from `-s`** — explicitly rejected per "Implementation Notes" #1 above.
- **Paid CSE backends** (Brave, Google CSE, Kagi) — only if user demand justifies the config-plumbing overhead.
- **Post / vote / comment** Stack Exchange operations — read-only stays read-only.
- **AI summarization** of results — out of scope; vör is a reference tool, not a generative one.

### Status

**Accepted and shipped 2026-04-28.** Reopen this ADR (or supersede it with ADR-0002) only if:
- A second backend is added (cheat.sh, Wikipedia, etc.) — that's a separate decision worth its own ADR.
- The auto-fallback decision needs to be revisited (e.g., real-world user feedback shows the explicit-opt-in model creates friction).
- Stack Exchange API ToS or quota model changes materially.

## References

- North Star: [`CLAUDE.md`](../../CLAUDE.md) — `## North Star` section
- Stack Exchange API docs: https://api.stackexchange.com/docs
- cheat.sh: https://github.com/chubin/cheat.sh
- DuckDuckGo Instant Answer API: https://duckduckgo.com/api
- Brave Search API: https://api.search.brave.com/
- Google Programmable Search Engine: https://developers.google.com/custom-search/v1/overview
- Wikipedia REST API: https://en.wikipedia.org/api/rest_v1/

## Revision History

| Date | Change | Author |
|---|---|---|
| 2026-04-25 | Initial proposal, deferred, Stack Exchange API flagged as lead | stevie@bellis.tech |
| 2026-04-28 | Accepted + implemented as `-so` opt-in flag (S6-A); architecture deviated from original proposal in 3 documented ways | stevie@bellis.tech |
