# Stack Overflow CLI Lookup (`-so` / `--stack-overflow`)

Optional bonus feature — terminal-rendered Stack Overflow search for the residual cases the offline encyclopedia doesn't cover.

> vör's primary mode is the **offline encyclopedia**: 812 sheets, 722 detail pages, 55 ramp-up topics, all embedded in the binary. The default `vor` invocation makes zero network calls. This page documents an opt-in shortcut that hits api.stackexchange.com when (and only when) you've configured a Stack Exchange API key. Without a key, the flag prints onboarding text and exits cleanly.

## Why It Exists

The 800+ embedded sheets cover ~95% of routine work. The remaining ~5% — fresh error messages, version-specific gotchas, "I just hit something nobody has put in a sheet yet" — is the use case Stack Overflow is uniquely good at. Adding a key-gated lookup gives you a one-flag shortcut for that residual case **without** changing what vör is for everyone else.

## Quick Start

```bash
# 1. Get a key (one-time, free, no OAuth)
#    See "Generating a Stack Exchange API Key" below.

# 2. Save it locally — outside any git repo
mkdir -p ~/.config/cs
chmod 700 ~/.config/cs
echo "STACK_OVERFLOW_API_KEY=<the-key-you-copied>" > ~/.config/cs/secrets.env
chmod 600 ~/.config/cs/secrets.env

# 3. Use it
vor -so "lvm cannot extend logical volume"
vor --stack-overflow "EADDRINUSE address already in use"
```

## Generating a Stack Exchange API Key

The Stack Exchange API uses an *application key* — not a personal token, not OAuth, not a client secret. The key alone gets you 10,000 requests per IP per day. No callback URL, no consent flow, no per-user state.

### Step-by-step

1. **Sign in** at https://stackapps.com/users/login. Any Stack Exchange account works (your existing Stack Overflow login is fine — the SE network is single-sign-on).
2. **Open the registration page**: https://stackapps.com/apps/oauth/register
3. **Fill in the form**:
   - **Application Name**: `vor-cli` (any name; visible only to you)
   - **Description**: `Personal CLI cheatsheet tool` (one line is enough)
   - **OAuth Domain**: `localhost` (literally the word — required field, but we never use OAuth)
   - **Application Website**: `https://github.com/bellistech/cs` (or any URL you control)
   - **Application Icon**: leave blank
   - **Enable Client Side OAuth Flow**: leave **unchecked**
   - **Disable Application**: leave unchecked
4. Click **Register Your Application**.
5. The next page shows three values. **You only need `Key`.** It looks like `XYZab1c2DE3fGh4iJK5lMN==`. Copy that string. Ignore `Client Id` and `Client Secret` (those are for OAuth, which we don't use).

### Where to put the key

Two methods. Pick one. Env beats file when both are present.

**Method A — file (persistent, recommended for daily use)**

```bash
mkdir -p ~/.config/cs
chmod 700 ~/.config/cs
touch ~/.config/cs/secrets.env
chmod 600 ~/.config/cs/secrets.env

$EDITOR ~/.config/cs/secrets.env
# Add the line:
STACK_OVERFLOW_API_KEY=XYZab1c2DE3fGh4iJK5lMN==
```

The file format is `KEY=VALUE` per line. Blank lines and `#` comments are ignored. `export KEY=VALUE` is also accepted (so the file is shell-source-compatible: `set -a; source ~/.config/cs/secrets.env`).

If the file is group- or world-readable, vör prints a one-shot warning to stderr the first time it's loaded:

```
warning: /home/you/.config/cs/secrets.env is group/world-readable; chmod 600 /home/you/.config/cs/secrets.env
```

The warning fires once per process and never blocks the lookup — but you should fix it.

**Method B — env var (ephemeral, recommended for one-shot tests)**

```bash
read -rs STACK_OVERFLOW_API_KEY      # silent prompt — does NOT go to shell history
export STACK_OVERFLOW_API_KEY
vor -so "test query"
```

Avoid `export STACK_OVERFLOW_API_KEY=…` typed directly: that lands in shell history.

## Usage

```bash
vor -so help                         # onboarding text (this page in summary form)
vor -so "<query>"                    # short form
vor --stack-overflow "<query>"       # long form (same dispatch)

vor stack-overflow-cli               # this sheet (offline)
```

The query can be anything Stack Overflow's search accepts: an error message, a question, a `[tag]` filter, a phrase in quotes. vör passes it through to `https://api.stackexchange.com/2.3/search/advanced?q=<your-query>&site=stackoverflow`.

### Worked examples

```bash
$ vor -so "lvm cannot extend"
# Stack Overflow: lvm cannot extend
*5 result(s).*

## 1. lvextend fails: "Insufficient free space"
**✓** · 12↑ · 3 answer(s) · 2024-08-13 · tags: `lvm`, `linux`
The PV needs free extents — check with `vgs`. Common fix:
` ` `
sudo pvresize /dev/sda3
sudo lvextend -l +100%FREE /dev/mapper/vg-root
sudo resize2fs /dev/mapper/vg-root
` ` `
→ <https://stackoverflow.com/q/12345678>
…

*Powered by Stack Exchange — content licensed CC BY-SA 4.0. Quota remaining: 9999 / 10000.*

$ vor -so "lvm cannot extend"
# (second call within 24h is a cache hit — zero network, zero quota)
```

## How It Works

```
        ┌─────────────────────────────┐
   you →│  vor -so "<query>"          │
        └──────────────┬──────────────┘
                       ▼
        ┌─────────────────────────────┐
        │ check ~/.cache/cs/          │── hit (≤24h) ──▶ render & exit
        │ stackoverflow/<sha256>.json │
        └──────────────┬──────────────┘
                       │ miss / stale / corrupt
                       ▼
        ┌─────────────────────────────┐
        │ load STACK_OVERFLOW_API_KEY │── missing ──▶ friendly error + exit 1
        │ env first, then secrets.env │
        └──────────────┬──────────────┘
                       ▼
        ┌─────────────────────────────┐
        │ HTTPS GET                   │── 4xx ──▶ redacted error
        │ api.stackexchange.com/2.3/  │── 5xx ──▶ ErrServerError
        │ search/advanced?q=&key=     │── 429 ──▶ ErrRateLimited
        └──────────────┬──────────────┘
                       │ 200 OK
                       ▼
        ┌─────────────────────────────┐
        │ atomically write JSON to    │
        │ ~/.cache/cs/stackoverflow/  │
        └──────────────┬──────────────┘
                       ▼
                  render via the same glamour pipeline
                  used by every other vor command
```

## REST Endpoint (Gauntlets-Law Parity)

If you run `vor serve`, the same lookup is available over HTTP:

```bash
vor serve &                          # starts on 127.0.0.1:9876 by default
curl 'http://127.0.0.1:9876/api/stackoverflow?q=lvm+extend' | jq .
```

Response codes:

| Code | Meaning |
|------|---------|
| 200  | result body |
| 400  | missing `q` parameter or empty query |
| 401  | invalid API key |
| 429  | rate-limited / quota exhausted |
| 502  | upstream Stack Exchange error |
| 503  | server has no `STACK_OVERFLOW_API_KEY` configured |

The server reads its own `STACK_OVERFLOW_API_KEY` from env or `~/.config/cs/secrets.env` of the user running `vor serve`. The cache is shared with the CLI (same `~/.cache/cs/stackoverflow/` directory).

## Quota & Rate Limiting

| Mode | Daily limit |
|------|------------:|
| Anonymous (no key) | 300 / day per IP |
| With application key | 10,000 / day per IP |

Each call costs 1 quota point; daily reset at midnight UTC. The 24h cache cuts repeat queries to zero quota. The CLI prints `quota_remaining` in the rendered footer:

```
*Powered by Stack Exchange — content licensed CC BY-SA 4.0. Quota remaining: 9842 / 10000.*
```

If a call returns `quota_remaining = 0` and zero items, vör maps that to `ErrRateLimited` and surfaces a friendly message rather than treating it as success.

## Cache

Path: `~/.cache/cs/stackoverflow/<sha256(query)>.json`
TTL: 24 hours (hardcoded; clear manually to force a refresh)
Format: typed JSON of `{stored_at: <unix>, result: {...}}` — cache is treated as **untrusted input** on read; corrupt entries are silently missed, not fatal.

```bash
# clear all cached responses
rm -rf ~/.cache/cs/stackoverflow/

# inspect a cached query
ls ~/.cache/cs/stackoverflow/
jq . ~/.cache/cs/stackoverflow/<filename>.json
```

The cache is **shared by query hash, not by key** — two users on the same box benefit from each other's cache. The cache file never contains the API key (the key only ever appears in the outbound URL).

## Privacy & Redaction Guarantees

The package treats the key as a tracked secret end-to-end:

1. **URL construction** — the key is added via `url.Values{}.Set("key", key)`, never via string concatenation. Go's `url` package handles encoding.
2. **Error wrapping** — every error path that could surface the URL or its components passes through `redactErr()`, which replaces literal occurrences of the key with `***` before bubbling up.
3. **Cache writes** — the cache stores only the *response body* (`Result` struct: questions + quota), never the request URL or headers.
4. **Logs** — vör never logs the URL or response headers. The only thing printed for a successful call is the rendered Markdown.

There is one place the key is briefly visible to the system: `/proc/<pid>/cmdline` *during* the HTTPS request, because the URL contains it. This is the same exposure as `curl https://...?key=...` and is the standard accepted profile for Stack Exchange's API model. If your threat model rules out brief proc-tree exposure, don't use this feature.

`make audit-secrets` is the second line of defense — a Makefile gate that greps the source tree for credential-shaped strings and exits 1 on hit. It runs as part of `make lint` so accidental commits are caught at PR time.

## Troubleshooting

### `cs: -so requires STACK_OVERFLOW_API_KEY (env var or /home/you/.config/cs/secrets.env)`

You haven't configured a key. Follow "Generating a Stack Exchange API Key" above.

### `stack-overflow: invalid api key: <message>`

The key is set but the API is rejecting it. Possible causes:

- The key was copied with surrounding whitespace. Re-edit `~/.config/cs/secrets.env` and trim.
- The app was disabled at https://stackapps.com/apps. Re-enable or re-register.
- You're behind a corporate proxy that mangles query strings. Test with `curl 'https://api.stackexchange.com/2.3/info?site=stackoverflow&key=<key>'`.

### `stack-overflow: rate limited or quota exhausted`

You're at 0 / 10,000 for the day, or the API is throttling your IP. Wait until midnight UTC, or use `~/.cache/cs/stackoverflow/` for already-fetched queries.

### `stack-overflow: server error: status 5xx`

Stack Exchange API is having a bad time. Retry in a few minutes; the cache covers anything you've already queried.

### `warning: ...secrets.env is group/world-readable; chmod 600`

```bash
chmod 600 ~/.config/cs/secrets.env
chmod 700 ~/.config/cs
```

### Want to disable the feature

Just don't set the key. The flag still parses, but `vor -so "x"` will exit 1 with the friendly nudge. The default `vor` experience is unaffected.

## Anti-Features (Explicitly Out of Scope)

These are intentionally NOT supported. Each would dilute the offline-first North Star and add code paths users would have to think about:

- Multi-site search (Server Fault, Super User, Unix & Linux SE) — possibly in a future S6-B
- OAuth flow / per-user access tokens — read-only is enough
- Posting answers, voting, comment retrieval — read-only stays read-only
- AI summarization of results — vör is a reference tool, not a generative one
- Reddit / GitHub Issues / other Q&A sources — same opt-in fatigue
- TTL configuration via flag — 24h is the right default; clear cache manually if you need fresh

## Tips

- Wrap multi-word queries in shell quotes: `vor -so "lvm cannot extend"` not `vor -so lvm cannot extend`.
- Use `[tag]` to scope: `vor -so "[bash] heredoc EOF unexpected"`.
- Pipe through `less` if results are long: `vor -so "..." | less -R` (vör's render pipeline already pages on TTY by default).
- Combine with the offline encyclopedia: when you find a topic worth memorizing, add a custom sheet via `vor --add` so you don't need to query again.

## See Also

- `troubleshooting/linux-errors` — common Linux error messages and fixes
- `troubleshooting/http-errors` — HTTP status codes and what they mean
- `shell/bash` — the bash reference (where most stuck-moments originate)
- `networking/curl` — for hand-rolled requests if you want to bypass the wrapper

## References

- Stack Exchange API docs: <https://api.stackexchange.com/docs>
- App registration: <https://stackapps.com/apps/oauth/register>
- Authentication overview (we use only the `key` flow): <https://api.stackexchange.com/docs/authentication>
- Throttle / quota docs: <https://api.stackexchange.com/docs/throttle>
- Stack Exchange content license (CC BY-SA 4.0): <https://stackoverflow.com/help/licensing>
- RFC 7234 — HTTP cache semantics (the model the 24h disk cache follows)
- vör's `internal/stackoverflow` package source — the audit trail for redaction guarantees
