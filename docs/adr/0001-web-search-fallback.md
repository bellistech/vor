# ADR-0001 — Web Search Fallback for Unmatched Queries

**Status:** Proposed — Deferred (revisit later)
**Date:** 2026-04-25
**Deciders:** stevie@bellis.tech
**Tags:** search, north-star, fallback, third-party-api

## Context

The North Star (committed to `CLAUDE.md`) is:

> *cs, the cheat sheet application, should have the ability to avoid leaving terminal while working on a system to open a web browser and search Google.*

Today's content sprint shipped ~50,000 lines across 25 verbose sheets, but local coverage will always have gaps. When `cs -s <query>` returns nothing useful, the user's only recourse is to leave the terminal and search Google — directly violating the vision. We need a path for the tool to fall through to a *web* search source without breaking the in-terminal contract.

## Decision

**Deferred.** No web-search fallback is implemented in this commit. This ADR captures the analysis so future work can resume without re-deriving the landscape.

When the work is picked up, the **leading candidate is the Stack Exchange API**, with `cheat.sh` as a complementary source. Direct Google scraping is **explicitly rejected**.

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

## Status / Next Action

**Deferred.** Revisit when:
- Local content gaps become user-visible pain (telemetry: track empty-result `cs -s` queries — add later).
- A specific user request lands ("I wanted X but cs doesn't have it").
- Or proactively, after the next 1-2 content sprints close more gaps.

When revisited, start with Stack Exchange API as the v1 fallback. Build a 1-2 day spike, evaluate, then expand.

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
