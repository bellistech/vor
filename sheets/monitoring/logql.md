# LogQL (Loki Query Language)

Loki's query language for log streams — modeled on PromQL but for logs; pipeline of `{stream selector} | filters | parsers | metric aggregation` returning either log streams or Prometheus-style metric vectors.

## Setup

LogQL is the query language baked into Grafana Loki — the log database from Grafana Labs that pairs with Prometheus. It is intentionally modeled on PromQL: same selector syntax `{label=value}`, same range vector syntax `[5m]`, same aggregation operators (`sum`, `avg`, `topk`, `quantile`, `rate`). The major divergence is the **log pipeline**: a `|`-separated chain of filter / parse / format stages that operate on log lines AFTER the stream selector matches them.

Two query types exist:

- **Log queries** — return log streams (lines + labels). Example: `{app="api"} |= "error"` shows lines containing "error".
- **Metric queries** — wrap log queries in a range aggregation to return Prometheus vectors. Example: `rate({app="api"} |= "error" [5m])` returns errors per second.

Where you run them:

```bash
# Grafana Explore — paste expression in the LogQL field, hit Run query
# Grafana Loki datasource panel — same expression, Time series or Logs panel
# Loki HTTP API direct
curl -G -s "http://loki:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={app="api"} |= "error"' \
  --data-urlencode 'start=2024-01-01T00:00:00Z' \
  --data-urlencode 'end=2024-01-01T01:00:00Z' \
  --data-urlencode 'step=60s'
# logcli — official CLI
logcli query '{app="api"} |= "error"' --since=1h
logcli query '{app="api"} |= "error"' --tail
# loki ruler — recording / alerting rules
```

The canonical query shape:

```bash
{app="api", env="prod"}                     # 1. stream selector — choose streams (REQUIRED)
  |= "POST /api/users"                      # 2. line filters — narrow lines
  | logfmt                                  # 3. parser — extract labels
  | level="error"                           # 4. label filter — narrow on extracted labels
  | line_format "{{.level}} {{.path}}"      # 5. line format — rewrite line (optional)
```

Wrap that in a range aggregation for a metric:

```bash
sum by (level) (
  rate({app="api", env="prod"} |= "POST" | logfmt [5m])
)
```

LogQL v2 (Loki 2.0+, Sept 2020) added: parsers (`logfmt`, `json`, `regexp`, `pattern`, `unpack`), label filters with operators (`>`, `>=`, `<`, `<=`, `==`, `!=`), `unwrap` for numeric range aggregations, `line_format` and `label_format`, and the `drop` / `keep` stages (Loki 2.7+, 2.9+).

## Data Model

A **log stream** in Loki is uniquely identified by its label set. Each unique combination of labels forms a separate stream — an append-only sequence of `(timestamp, line)` entries. Two log lines with identical labels go into the same stream; one different label value creates a new stream.

```bash
# These are TWO DIFFERENT streams (different label sets):
{app="api", env="prod", level="info"}    "ts=... msg=hello"
{app="api", env="prod", level="error"}   "ts=... msg=oops"

# These are the SAME stream (identical label set, two entries):
{app="api", env="prod", level="info"}    "ts=10:00 msg=hello"
{app="api", env="prod", level="info"}    "ts=10:01 msg=world"
```

Loki's **golden rule**: labels are LOW-CARDINALITY. Same as Prometheus. An index entry exists per stream, so a `userId` label with millions of values produces millions of streams and a wrecked index. The recommended cap is single-digit thousands of streams per tenant.

The mental model contrast:

| System         | Indexed by         | Search lines by                    |
|----------------|--------------------|-----------------------------------|
| Loki           | labels (low-card)  | full-text on lines (post-fetch)   |
| Elasticsearch  | full-text terms    | inverted index on every word       |
| grep on disk   | nothing            | linear scan of all bytes           |

So Loki "indexes the metadata, greps the content." This is why Loki is cheap to operate (small index) but query latency depends on how much log data your stream selector pulls.

```bash
# Good — small label set, narrowed by line filter
{app="api", env="prod"} |= "user_id=42"

# Bad — userId as a label, ballooning streams
{app="api", env="prod", user_id="42"}     # don't do this
```

Log lines themselves are byte-stream chunks (snappy/gzip compressed) stored in object storage (S3/GCS/Azure Blob/filesystem). The index is BoltDB-shipper, TSDB (recommended Loki 2.8+), or Cassandra/DynamoDB (legacy).

## Stream Selectors `{label=value}`

The first part of every LogQL query is the stream selector inside curly braces — the same syntax as Prometheus, with the same four matchers:

| Operator | Meaning                                           |
|----------|---------------------------------------------------|
| `=`      | exact equality                                    |
| `!=`     | not equal                                         |
| `=~`     | matches RE2 regex (anchored — full string match)  |
| `!~`     | does not match RE2 regex                          |

```bash
{app="api"}                              # exactly app=api
{app!="api"}                             # any app except api (BUT must combine with another positive matcher)
{app=~"api|web"}                         # api OR web (regex alternation)
{app=~"frontend-.+"}                     # any app starting with frontend-
{app!~"test-.*"}                         # not starting with test-
{app="api", env="prod", region="us-east-1"}  # AND of all matchers
```

Multiple matchers inside `{}` are ANDed. There is no OR between selectors at this level — use a regex alternation `=~"api|web"` for OR.

**The required-positive-matcher rule**: at least one matcher must be `=` or `=~` (positive). This prevents queries like `{app!="test"}` from scanning every stream in the cluster.

```bash
# Error: stream selector is empty
{app!="test"}

# Fix: add a positive matcher
{env="prod", app!="test"}
```

Regex matchers use Google's RE2 — fully anchored, so `=~"foo"` matches the label value `foo` exactly, not "containsfoo". Use `=~".*foo.*"` for substring match.

```bash
{path=~".*\\.html"}                      # ends with .html (regex needs escaping)
{path=~".+\\.json|.+\\.yaml"}            # OR of suffixes
```

Common label sources (depends on your shipper config):

```bash
{app="...", instance="...", env="...", cluster="..."}     # service-level
{job="...", filename="...", namespace="...", pod="..."}   # Kubernetes promtail
{container="...", image_tag="...", host="..."}            # docker
```

## Log Pipelines

After the stream selector, optional pipeline stages process each matched log line in order. Stages are separated by `|` and are evaluated left-to-right:

```bash
{app="api"}                              # stream selector
  |= "POST"                              # 1. line filter (contains)
  |~ "user_id=\\d+"                      # 2. line filter (regex match)
  | logfmt                               # 3. parser (extracts labels)
  | level="error"                        # 4. label filter
  | duration > 100ms                     # 5. label filter (numeric)
  | line_format "{{.level}} {{.msg}}"    # 6. line format (rewrite line)
  | label_format svc="{{.app}}"          # 7. label format (rename label)
  | drop instance, host                  # 8. drop stage (remove labels)
```

Stage taxonomy:

| Stage                | Examples                                       | Effect                                   |
|----------------------|-----------------------------------------------|------------------------------------------|
| Line filter          | `\|= "x"`, `!= "x"`, `\|~ "re"`, `!~ "re"`    | drops lines that don't match              |
| Parser               | `\| logfmt`, `\| json`, `\| regexp`, `\| pattern`, `\| unpack` | extracts labels from line content |
| Label filter         | `\| key="v"`, `\| key>5`                       | drops lines whose labels don't match      |
| Line format          | `\| line_format "tmpl"`                        | rewrites the line text                   |
| Label format         | `\| label_format new=old`                       | renames or rewrites labels                |
| Decolorize           | `\| decolorize`                                 | strips ANSI escape codes from line        |
| Drop / keep          | `\| drop l1, l2`, `\| keep l1, l2`              | removes / retains specific labels         |
| Unwrap               | `\| unwrap latency`                             | exposes a numeric label for aggregation   |

**Pipeline ordering matters for performance**: line filters are pushed down to the chunk decoder and execute before any parser. Always put exact-match line filters as far left as possible.

```bash
# Slow — parses every line, then filters
{app="api"} | logfmt | level="error"

# Fast — drops 99% of lines before parsing
{app="api"} |= "level=error" | logfmt | level="error"
```

## Line Filters

Line filters operate on the raw log line text (before parsing). Four operators:

| Op    | Meaning                                  |
|-------|------------------------------------------|
| `\|=` | line contains substring                  |
| `!=`  | line does not contain substring          |
| `\|~` | line matches RE2 regex (unanchored)      |
| `!~`  | line does not match RE2 regex            |

```bash
{app="api"} |= "ERROR"                   # case-sensitive substring
{app="api"} |= "ERROR" != "expected"     # contains ERROR but not "expected"
{app="api"} |~ "(?i)error"               # case-insensitive (RE2 flag)
{app="api"} |~ "user_id=\\d+"            # regex (note: backslashes need escaping)
{app="api"} !~ "/healthz|/readyz"        # exclude health probes
```

Multiple line filters chain with implicit AND:

```bash
{app="api"} |= "POST" |= "users" != "expected" |~ "5\\d{2}"
# contains POST AND contains users AND not contains expected AND matches 5xx
```

**Line filters are case-sensitive by default**. Use the `(?i)` RE2 flag for case-insensitive regex:

```bash
{app="api"} |~ "(?i)error|warn|fatal"
```

**Why line filters are special**: the Loki query engine pushes line filter predicates down to the chunk-fetch stage, so the chunk reader skips over irrelevant lines without copying or parsing them. This is the single biggest performance optimization in LogQL.

```bash
# 100x faster — line filter pushed down
{cluster="prod", namespace="api"} |= "OutOfMemory"

# 100x slower — same number of matches but the engine has to parse first
{cluster="prod", namespace="api"} | json | message=~".*OutOfMemory.*"
```

Newline-aware: the line filter sees the full original line including any trailing newline trimmed at ingestion.

## Label Filters

Label filters narrow on labels — either streamselector labels or labels extracted by an upstream parser. Syntax `| key OP value`:

| Op         | Type                       |
|------------|----------------------------|
| `=`, `!=`  | string equality / inequality |
| `=~`, `!~` | regex match / not-match (RE2) |
| `>`, `>=`, `<`, `<=` | numeric (with units: `100ms`, `1.5s`, `10MB`, `1KiB`) |
| `==`       | numeric equality (note: `=` works for strings, `==` for numbers — `=` also works for numbers in v2) |

```bash
{app="api"} | logfmt | level="error"             # string
{app="api"} | logfmt | status_code >= 500        # numeric
{app="api"} | logfmt | duration > 100ms          # duration unit
{app="api"} | logfmt | size > 10MB               # bytes unit (case-sensitive: KB/MB/GB or KiB/MiB/GiB)
{app="api"} | logfmt | path=~"/api/.+"           # regex on extracted label
{app="api"} | logfmt | path!~"/healthz"          # exclude
{app="api"} | logfmt | level="error" or level="warn"     # OR within label filters
{app="api"} | logfmt | level="error", path="/api/users"  # implicit AND (comma)
```

Numeric units recognized:

```bash
# Time:    ns, us, µs, ms, s, m, h
| duration > 1.5s
| latency_ns > 500000ns
# Bytes:   B, KB, MB, GB, TB, PB, KiB, MiB, GiB, TiB, PiB
| body_size > 1MB
| chunk > 256KiB
```

**Label filters apply AFTER the parser that produced the labels**:

```bash
# Error: status_code is not a stream label, must come after parser
{app="api"} | status_code >= 500

# Fix: add | logfmt or | json first
{app="api"} | logfmt | status_code >= 500
```

OR vs AND: comma is AND, the keyword `or` is OR (within a single label filter expression).

```bash
| level="error" or level="fatal"             # OR
| level="error", path="/api"                  # AND (comma)
| level=~"error|fatal", path="/api"           # equivalent OR-then-AND
```

## Parsers — logfmt

`| logfmt` parses Brian Ketelsen's logfmt format (`key=value key="value with spaces" key2=val2`) and exposes each key as a label. This is the default for Go services using `logrus`, `zap`, `slog`, or `kit/log`.

```bash
# Input line:
# ts=2024-01-15T10:00:00Z level=error msg="db timeout" duration=120ms user_id=42
{app="api"} | logfmt
# After parsing, these labels are available:
#   ts="2024-01-15T10:00:00Z"
#   level="error"
#   msg="db timeout"
#   duration="120ms"
#   user_id="42"
```

Filter on extracted labels:

```bash
{app="api"} | logfmt | level="error"
{app="api"} | logfmt | level=~"error|fatal" | duration > 100ms
{app="api"} | logfmt | user_id="42"
```

**Renaming on extract** (Loki 2.5+): rename a key on the way out, useful when keys have spaces, dots, dashes, or LogQL keywords:

```bash
# Input: trace.id=abc level=info
{app="api"} | logfmt trace_id="trace.id"

# Strict mode (Loki 2.9+): error if extraction fails
{app="api"} | logfmt --strict
```

The `--keep-empty` flag retains keys with empty values rather than dropping them.

```bash
{app="api"} | logfmt --keep-empty
```

If a line is not valid logfmt, the parser sets `__error__="LogfmtParserErr"` on that line so you can filter it:

```bash
# Show only lines that failed to parse
{app="api"} | logfmt | __error__ != ""

# Drop parse-failure lines from a metric query
{app="api"} | logfmt | __error__ = ""
```

## Parsers — json

`| json` parses JSON log lines and flattens nested objects with dot-notation paths. JSON is the format produced by `bunyan`, `pino`, `winston`, AWS CloudWatch, and Kubernetes events.

```bash
# Input line:
# {"level":"error","msg":"db timeout","ctx":{"user":{"id":42},"req_id":"abc"}}
{app="api"} | json
# Labels exposed:
#   level="error"
#   msg="db timeout"
#   ctx_user_id="42"
#   ctx_req_id="abc"
```

Default behavior flattens with `_` separator. Arrays produce indexed labels: `items_0_name`, `items_1_name`.

**Selective extraction** with JSONPath-like expressions (Loki 2.0+) avoids cardinality blowup on big nested objects:

```bash
# Only extract specific fields, optionally rename them
{app="api"} | json level, msg, user_id="ctx.user.id", req_id="ctx.req_id"

# Array element by index
{app="api"} | json first_item="items[0].name"

# Nested paths
{app="api"} | json owner="metadata.labels.owner"
```

JSONPath syntax inside the quoted argument:

```bash
| json a="$.b.c"          # nested object
| json x="$[0]"           # first array element
| json y="$.list[1].id"   # second element's id
| json k="$['weird key']" # bracket form for keys with spaces / dots
```

If a line is not valid JSON, `__error__="JSONParserErr"` is set:

```bash
{app="api"} | json | __error__ != ""    # show parse failures
```

JSON parser **does not error on missing fields** — the requested label is simply unset (or empty). Use the `__error__` label to differentiate "field absent" from "JSON malformed."

```bash
# Real-world: API access log
{app="api"} | json
  | path=~"/api/v1/.+"
  | status_code >= 500
  | duration_ms > 200
```

## Parsers — regexp

`| regexp "..."` runs a Go RE2 regex with **named capture groups** and exposes each named group as a label. Use this when the line format is neither logfmt nor JSON.

```bash
# Input line:
# 2024-01-15 10:00:00 INFO  HandleLogin user=alice ip=10.0.0.1 took 25ms
{app="api"} | regexp "(?P<level>\\w+)\\s+(?P<func>\\w+)\\s+user=(?P<user>\\S+)\\s+ip=(?P<ip>\\S+)\\s+took\\s+(?P<dur>\\S+)"
# Labels: level, func, user, ip, dur
```

Note the **double-escaping** — LogQL strings are double-quoted, so `\d` becomes `\\d`, `\s` becomes `\\s`, `\.` becomes `\\.`.

```bash
# Apache combined log format extract via regexp
{job="apache"} | regexp "^(?P<ip>\\S+) \\S+ \\S+ \\[(?P<ts>[^\\]]+)\\] \"(?P<method>\\S+) (?P<path>\\S+) (?P<proto>\\S+)\" (?P<status>\\d+) (?P<size>\\d+|-)"
```

Unnamed groups `(...)` are ignored — only `(?P<name>...)` produces labels.

If the regex doesn't match a line, no labels are added and `__error__="RegexpParserErr"` is set.

```bash
# Show parse failures
{app="api"} | regexp "..." | __error__ != ""
```

The regex must use Go RE2 syntax (no backreferences, no lookaround, no `(?<name>...)` PCRE form — it's strictly `(?P<name>...)`).

`regexp` is **the most expensive parser** — prefer `pattern` for fixed-position fields and `logfmt`/`json` for structured logs.

## Parsers — pattern

`| pattern "..."` (Loki 2.3+) is a simpler, faster alternative to `regexp` for fixed-position formats. The pattern uses `<name>` for capture and `<_>` to skip a section.

```bash
# Input: 10.0.0.1 - alice [15/Jan/2024:10:00:00 +0000] "GET /api/users HTTP/1.1" 200 1234
{job="nginx"} | pattern "<ip> <_> <user> [<ts>] \"<method> <path> <_>\" <status> <size>"
# Labels: ip, user, ts, method, path, status, size
```

Rules:

- `<name>` captures non-whitespace by default (greedy until next literal).
- `<_>` discards a section (anonymous capture).
- Literal characters between captures must match exactly (so the brackets, spaces, and quotes above are required).
- Pattern is **anchored from the left** by default — the first character of the line must match the first character of the pattern (or a `<name>` / `<_>`).
- A pattern starting with `<_>` skips arbitrary leading content; same with trailing.

```bash
# Skip leading garbage, capture method+path, ignore tail
| pattern "<_> <method> <path> <_>"

# Apache common log
{job="apache"} | pattern "<ip> - - [<ts>] \"<method> <path> <_>\" <status> <size>"

# Kubernetes audit log fragment
{app="apiserver"} | pattern "<_> verb=<verb> uri=<uri> code=<code> <_>"
```

Why prefer `pattern` over `regexp`:

- 5-10x faster (no regex backtracking machinery).
- Easier to read.
- No double-escaping nightmares.

Falls back to `regexp` only when you need character classes, alternation, or quantifiers.

If pattern fails to match a line, `__error__="PatternParserErr"` is set.

## Parsers — unpack

`| unpack` (Loki 2.0+) decodes the special `pack` format produced by Promtail's `pack` stage. Promtail can pack labels into the log line as a JSON envelope `{"_entry":"original log line", "label1":"v1", "label2":"v2"}` to avoid creating many streams. `| unpack` reverses that:

```bash
# Stored in Loki as:
# {"_entry":"db connection refused","container":"api","level":"error"}
{job="myapp"} | unpack
# After unpack: line is "db connection refused", labels container="api" level="error" added
```

Useful when you want to keep cardinality LOW in Loki streams but still query/aggregate by label-like fields. Pair with `| json` if you also have JSON inside the unpacked entry:

```bash
{job="myapp"} | unpack | json
```

`unpack` errors set `__error__="JSONParserErr"` (it's effectively a JSON parse with the `_entry` convention).

## Line Format

`| line_format "tmpl"` rewrites the log line text using a Go `text/template` evaluated against the labels. Use it to extract just the part of the line you want, after parsing.

```bash
# After parsing JSON, replace the verbose JSON with a one-liner:
{app="api"}
  | json
  | line_format "{{.level}} {{.msg}} dur={{.duration_ms}}ms user={{.user_id}}"
# Output line: "error db timeout dur=120ms user=42"
```

Inside the template:

- `{{.label}}` — value of label `label` (Go field-access syntax; label name must be a valid Go identifier).
- `{{ .label | upper }}` — pipe through built-in or registered functions.
- Conditionals: `{{ if eq .level "error" }}!!! {{end}}{{.msg}}`
- Math: `{{ printf "%.2f" (divf .total .count) }}`

LogQL adds custom template functions (Loki 2.4+):

```bash
{{.dur | duration}}                  # parse "120ms" -> 0.12 seconds
{{.bytes | bytes}}                   # parse "1KB" -> 1024
{{.path | trimPrefix "/api/"}}       # string ops: trimPrefix, trimSuffix, trim, replace
{{.line | regexReplaceAll "id=\\d+" "id=*"}}   # regex replace
{{.user | b64enc}}                   # base64 encode / b64dec
{{.host | sha256}}                   # sha1, sha256
{{ now | date "2006-01-02" }}        # Go time formatting
```

`upper`, `lower`, `title`, `default`, `quote`, `printf`, `repeat`, `contains`, `hasPrefix`, `hasSuffix` — all from `Sprig`.

```bash
# Real-world: collapse a chatty JSON log to one searchable line
{app="webhook"}
  | json
  | line_format "{{.level | upper}} {{.svc}} ev={{.event}} target={{.target}} ms={{.latency_ms}}"
```

`line_format` does not change labels; it only rewrites the line text.

## Label Format

`| label_format ...` (Loki 2.0+) renames labels or rewrites label values via templates.

Two forms:

```bash
# Rename: new_name = existing_label
| label_format svc=app, env_short=environment

# Template: rewrite a label value
| label_format service="{{.app}}-{{.env}}"

# Combine in one stage
| label_format svc=app, region="{{.cluster}}-{{.zone}}"
```

The `=` form (no quotes) is rename; the `="..."` form (template string) is rewrite.

```bash
# Real-world: normalize a noisy label
{app="api"}
  | label_format env="{{.environment | lower}}"
  | env="prod"
```

You cannot reference a label that the pipeline hasn't yet produced — order parsers before `label_format` references.

```bash
# Bug: status_code not yet extracted
{app="api"} | label_format bucket="{{.status_code}}xx" | logfmt

# Fix: parse first, then label_format
{app="api"} | logfmt | label_format bucket="{{div .status_code 100 | printf \"%dxx\"}}"
```

`label_format` does not change the line text.

## Decolorize

`| decolorize` (Loki 2.7+) strips ANSI color escape codes (`\x1b[...m`) from the line. Useful for logs that include terminal coloring (`go test`, dev servers) when you want clean line filters or template output.

```bash
{app="ci"} | decolorize |= "FAIL"
# Without decolorize, "FAIL" inside red ANSI codes won't match if you also matched against the codes
```

Apply BEFORE other line filters that would be confused by ANSI sequences.

```bash
{app="dev-server"} | decolorize | line_format "{{.}}"
```

Performance impact is negligible — it's a single linear scan.

## Drop / Keep Stage

`| drop label1, label2` (Loki 2.7+) removes labels from the result. `| keep label1, label2` (Loki 2.9+) keeps only the listed labels and drops everything else.

```bash
{app="api"} | logfmt | drop instance, host, ts
# Useful before metric aggregation to reduce cardinality of the output series

{app="api"} | logfmt | keep level, path
# Inverse — only level and path survive (plus the original stream-selector labels)
```

Both stages also accept a key=value form to drop only when the value matches:

```bash
| drop level="debug"
| drop level="debug", level="info"   # drop both debug AND info lines'  labels (not the lines)
```

`drop` and `keep` operate on labels, not on lines — they don't filter rows. To filter rows, use a label filter (`| level!="debug"`).

Common pattern: parse, filter, drop noisy labels, then aggregate.

```bash
sum by (svc) (
  rate({env="prod"}
    | logfmt
    | level="error"
    | drop instance, pod, container
    | label_format svc=app
    [5m]
  )
)
```

## Metric Queries — Aggregations Over Logs

Wrap a log query in a **range aggregation** `op({selector + pipeline} [range])` to convert log streams into Prometheus-style metric vectors. Range aggregations operate on a sliding time window:

| Aggregation              | What it does                                           |
|--------------------------|--------------------------------------------------------|
| `count_over_time(... [r])` | count of log lines in range r                          |
| `rate(... [r])`            | per-second rate of log lines (count / r seconds)       |
| `bytes_over_time(... [r])` | total bytes of log content in range                    |
| `bytes_rate(... [r])`      | per-second bytes rate                                  |
| `sum_over_time(... | unwrap x [r])` | sum of unwrapped numeric values                     |
| `avg_over_time(... | unwrap x [r])` | average                                              |
| `min_over_time`, `max_over_time`, `stddev_over_time`, `stdvar_over_time`, `quantile_over_time(q, ...)` | the rest |
| `first_over_time`, `last_over_time` | first / last value in range                            |
| `absent_over_time(... [r])` | 1 if NO matching log lines in range, else nothing      |

```bash
# Error log rate per service over 5 minutes
sum by (app) (
  rate({env="prod"} |= "ERROR" [5m])
)

# Total log bytes per pod over 1h
sum by (pod) (
  bytes_over_time({namespace="api"} [1h])
)

# Count of 5xx responses per route, last 5m
sum by (path) (
  count_over_time(
    {app="api"} | json | status_code >= 500 [5m]
  )
)
```

Range duration syntax is the same as PromQL: `5s`, `30s`, `1m`, `5m`, `1h`, `1d`. Loki supports up to `30d` but anything over a day is usually expensive.

The output of a range aggregation is a Prometheus-compatible vector — feed it into instant aggregations (`sum`, `avg`, `topk`, `quantile`) just like PromQL.

## Unwrap and Range Aggregations

`| unwrap label` (Loki 2.0+) tells the range aggregation to operate on the NUMERIC value of a parsed label instead of counting lines. Required for any aggregation that's not `count_over_time` / `rate` / `bytes_*`.

```bash
# Latency p99 from JSON-formatted access log
quantile_over_time(0.99,
  {app="api"}
    | json
    | unwrap latency_ms
    [5m]
)

# Average request size
avg_over_time(
  {app="api"} | logfmt | unwrap size [5m]
)

# Sum of bytes processed
sum_over_time(
  {app="batch"} | json | unwrap bytes_processed [1h]
)

# Stddev of duration
stddev_over_time(
  {app="api"} | logfmt | unwrap duration [5m]
)
```

`unwrap` value parsing:

- A bare numeric label (`123`, `1.5`) parses as a float.
- Duration-suffixed (`120ms`, `1.5s`) — use `| unwrap duration(latency)` to convert.
- Bytes-suffixed (`1KB`, `5MiB`) — use `| unwrap bytes(size)`.

```bash
# Convert duration string -> seconds
quantile_over_time(0.99, {app="api"} | logfmt | unwrap duration(req_dur) [5m])

# Convert bytes string -> bytes
sum_over_time({app="api"} | logfmt | unwrap bytes(body_size) [1h])
```

If a line's unwrap label is missing or unparseable, the line is skipped from the aggregation (and `__error__` is set so you can filter / count failures).

```bash
# Count of unparseable lines
sum(count_over_time({app="api"} | logfmt | unwrap duration_ms | __error__!="" [5m]))
```

The full range aggregations table (with `unwrap`):

```bash
sum_over_time({s} | logfmt | unwrap x [5m])
avg_over_time({s} | logfmt | unwrap x [5m])
min_over_time({s} | logfmt | unwrap x [5m])
max_over_time({s} | logfmt | unwrap x [5m])
stddev_over_time({s} | logfmt | unwrap x [5m])
stdvar_over_time({s} | logfmt | unwrap x [5m])
quantile_over_time(0.95, {s} | logfmt | unwrap x [5m])
first_over_time({s} | logfmt | unwrap x [5m])
last_over_time({s} | logfmt | unwrap x [5m])
absent_over_time({s} | logfmt | unwrap x [5m])
rate({s} | logfmt | unwrap x [5m])           # rate works with unwrap too — gives per-second sum
rate_counter({s} | logfmt | unwrap x [5m])   # treats unwrap value as a counter (per-stream delta / range)
```

## The Same Aggregation Operators as PromQL

Once you have a metric vector from a range aggregation, you apply the same INSTANT aggregations as PromQL on top:

```bash
sum, avg, min, max, count
stddev, stdvar
topk(N, ...), bottomk(N, ...)
sort, sort_desc
quantile(q, ...)
group       # Loki 2.5+
```

All accept `by (label, ...)` to group, or `without (label, ...)` to drop labels:

```bash
# Top 10 noisiest streams (full label set)
topk(10, count_over_time({env="prod"}[5m]))

# Per-service error rate, summed across pods
sum by (service) (rate({env="prod"} |= "ERROR" [5m]))

# Percentage of errors per service
sum by (service) (rate({env="prod"} |= "ERROR" [5m]))
/
sum by (service) (rate({env="prod"} [5m]))
* 100

# p99 latency per route
quantile by (route) (0.99,
  rate({app="api"} | logfmt | unwrap duration_ms [5m])
)
```

Binary arithmetic and comparison work too — same as PromQL:

```bash
# Error rate over 5m, but only show services with > 0.1/s
(sum by (svc) (rate({env="prod"} |= "ERROR" [5m]))) > 0.1

# Ratio of errors to total
sum by (svc) (rate({env="prod"} |= "ERROR" [5m]))
/ ignoring(level)
sum by (svc) (rate({env="prod"} [5m]))
```

`label_replace` and `label_join` are also available with the same PromQL semantics.

```bash
label_replace(
  rate({app="api"} |= "ERROR" [5m]),
  "service", "$1", "app", "(.+)"
)
```

## Common Patterns

```bash
# 1. Error rate per service
sum by (service) (
  rate({env="prod"} |= "ERROR" [5m])
)

# 2. p99 latency from logs (JSON access log)
quantile_over_time(0.99,
  {app="api"} | json | unwrap latency_ms [5m]
) by (route)

# 3. Top 10 noisiest streams (find log spammers)
topk(10, sum by (app, instance) (count_over_time({env="prod"}[5m])))

# 4. JSON event filtering (find specific events)
{app="webhook"} | json | event="user_signup"

# 5. Multi-line stack traces — parse via regexp with multi-line trick
{app="api"} |~ "(?ms)Exception:.*at.*"

# 6. Find a needle by request ID
{env="prod"} |= "req_id=abc123"

# 7. Audit log — who did what
{app="apiserver"} | json | verb="delete" | line_format "{{.user.username}} -> {{.objectRef.resource}}/{{.objectRef.name}}"

# 8. Log volume per service (capacity planning)
sum by (app) (bytes_rate({env="prod"} [1h]))

# 9. Rate of HTTP 5xx (alerting)
sum by (svc) (
  rate({env="prod"} | logfmt | status_code >= 500 [5m])
)

# 10. Slow query log
{app="db"} | logfmt | duration > 1s

# 11. OOM events per node
count_over_time({severity="critical"} |~ "OutOfMemory|OOMKilled" [10m])

# 12. Active users per hour (from access log)
count(sum by (user_id) (
  count_over_time({app="api"} | json [1h])
))

# 13. Endpoint hit ratio
sum by (path) (rate({app="api"} | json [5m]))

# 14. SLO burn-rate over 30d (slow burn alert)
sum(rate({env="prod"} | logfmt | status_code >= 500 [30d])) /
sum(rate({env="prod"} [30d])) > 0.001

# 15. Show only logs lacking a trace_id (poor instrumentation)
{app="api"} | json | trace_id=""

# 16. Group log volume by HTTP method
sum by (method) (rate({app="api"} | logfmt [5m]))

# 17. Distribution of request sizes (heat map source)
sum by (size_bucket) (
  rate({app="api"} | logfmt | label_format size_bucket="{{ if lt .size 1024 }}small{{ else if lt .size 1048576 }}medium{{ else }}large{{ end }}" [5m])
)

# 18. Count of distinct users
count(count by (user_id) (count_over_time({app="api"} | json [1h])))
```

## The PromQL-LogQL Crossover

Every LogQL **metric** query (rate / count_over_time / unwrap-aggregation) returns Prometheus-compatible time series. This unlocks several patterns:

- **Same Grafana panel** — mix Prometheus and Loki datasources via the "Mixed" datasource and overlay them. RPS from Prometheus, error rate from Loki.
- **Alertmanager integration** — Loki ruler evaluates LogQL metric queries on schedule and fires alerts the same way Prometheus does.
- **Recording rules** — pre-compute expensive log queries to a Prometheus-style series for cheap dashboard reads.
- **Dashboard variables** — `label_values({app="api"}, level)` populates a Grafana variable from Loki's labels.

```bash
# In a Grafana Mixed-datasource panel:
# Series A (Prometheus):  rate(http_requests_total{app="api"}[5m])
# Series B (Loki):        rate({app="api"} |= "ERROR" [5m])
# Series C (Loki):        rate({app="api"} | logfmt | level="warn" [5m])
```

LogQL operators (`+`, `-`, `*`, `/`, `>`, `<`, `==`, `!=`) follow PromQL vector matching rules: `on()`, `ignoring()`, `group_left`, `group_right`. Series with identical labels combine; mismatched labels need the explicit matching modifier.

```bash
# Error % per service
(sum by (svc) (rate({env="prod"} |= "ERROR" [5m])))
/ on (svc)
(sum by (svc) (rate({env="prod"} [5m])))
* 100
```

## Alerting via LogQL

The Loki **ruler** component runs LogQL queries on a schedule and pushes alerts to Alertmanager. Configuration is YAML, identical structure to Prometheus alerting rules.

```bash
# /etc/loki/rules/<tenant>/<group>.yaml
groups:
  - name: app-errors
    interval: 1m
    rules:
      - alert: HighErrorRate
        expr: |
          sum by (service) (rate({env="prod"} |= "ERROR" [5m])) > 0.5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "{{ $labels.service }} error rate > 0.5/s"
          description: "Error rate {{ $value }}/s in {{ $labels.service }}"

      - alert: NoLogs
        expr: |
          absent_over_time({app="critical-api"}[10m])
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "No logs received from critical-api"

      - alert: SlowRequests
        expr: |
          quantile_over_time(0.99, {app="api"} | json | unwrap duration_ms [5m]) > 1000
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "p99 latency > 1s in {{ $labels.app }}"
```

Recording rules pre-compute expensive queries:

```bash
groups:
  - name: log-rates
    interval: 1m
    rules:
      - record: app:errors:rate5m
        expr: sum by (app) (rate({env="prod"} |= "ERROR" [5m]))
```

Loki ruler can write recording rule outputs to a remote write endpoint (any Prometheus-compatible TSDB — Mimir, Cortex, Thanos, VictoriaMetrics) so the resulting metrics live alongside your Prometheus data.

```bash
# loki config snippet
ruler:
  storage:
    type: local
    local: { directory: /etc/loki/rules }
  rule_path: /tmp/loki/rules-temp
  alertmanager_url: http://alertmanager:9093
  ring: { kvstore: { store: inmemory } }
  enable_api: true
  remote_write:
    enabled: true
    client:
      url: http://mimir:9009/api/v1/push
```

`for:` works identically to Prometheus — alert is `pending` then `firing` after the duration passes.

## Common Errors and Fixes

Exact error texts you'll see and how to fix them:

```bash
# "stream selector is empty: matcher is required"
# Cause: stream selector has no positive (=, =~) matcher, or is missing entirely.
{app!="test"}                            # error
# Fix: add at least one positive matcher
{env="prod", app!="test"}

# "querier: query timed out"
# Cause: query exceeds Loki's query timeout (default 1m), too much data scanned.
{env="prod"}                             # 30 days of all prod logs — times out
# Fix: narrow time range, narrow stream selector, push line filter early
{env="prod", app="api"} |= "checksum mismatch"

# "max-streams-matchers exceeded"
# Cause: regex stream matcher matches too many distinct streams (> max_query_series, default 500-1000).
{app=~".+"}                              # matches every app — error
# Fix: be more specific
{app=~"api|web|worker"}

# "max-entries-limit exceeded"
# Cause: instant-log query result exceeds max_entries_limit_per_query (default 5000).
{env="prod"} | json                      # too many lines
# Fix: aggregate to a metric query, or narrow time range / filters
sum(count_over_time({env="prod"} [5m]))

# "parse error: unexpected SELECTOR" / "parse error: unexpected IDENTIFIER"
# Cause: syntax error in stream selector — usually missing comma, unbalanced quote, or label name.
{app="api" env="prod"}                   # missing comma — error
# Fix:
{app="api", env="prod"}

# "parse error at line 1, col 25: syntax error"
# Cause: malformed pipeline stage — usually missing | or wrong operator
{app="api"} = "ERROR"                    # = is not a line filter operator
# Fix:
{app="api"} |= "ERROR"

# "max-query-length exceeded (you requested 30d, the maximum is 7d)"
# Cause: time range exceeds tenant's max_query_length.
# Fix: shrink the range, or get the limit raised in Loki config.

# "too many outstanding requests"
# Cause: queue full (max_outstanding_per_tenant, default 100). Many concurrent queries.
# Fix: throttle Grafana refresh / dashboard concurrency, or scale querier.

# "logfmt parser failed" / __error__="LogfmtParserErr"
# Cause: line is not valid logfmt (unbalanced quotes, malformed key=value).
# Fix: filter out failures, or pre-process via line_format / regexp.
{app="api"} | logfmt | __error__ = ""

# "JSON parser failed" / __error__="JSONParserErr"
# Cause: line is not valid JSON.
# Fix: filter `|= "{"` first to skip non-JSON lines, or use regexp parser instead.

# "unwrap: not a numeric value"
# Cause: unwrapped label has non-numeric content.
# Fix: filter to numeric-only lines, or use duration() / bytes() conversion
| unwrap duration(latency)

# "regexp parser failed" / __error__="RegexpParserErr"
# Cause: regex compiled OK but doesn't match the line.
# Fix: this is normal for lines that don't fit; filter __error__="" to skip.
```

## Common Gotchas

Side-by-side broken and fixed:

```bash
# Gotcha 1 — stream selector with only negative matchers
# Bad:
{app!="healthz"}
# Error: "stream selector is empty: matcher is required"
# Fixed:
{env="prod", app!="healthz"}

# Gotcha 2 — line filter AFTER expensive parse
# Bad:
{app="api"} | json | message=~".*OutOfMemory.*"
# (parses every line, then regex-matches)
# Fixed:
{app="api"} |= "OutOfMemory" | json
# (line filter pushed down to chunk decoder; 100x faster)

# Gotcha 3 — high-cardinality label
# Bad: promtail config that labels by user_id
- match: { selector: '{job="api"}' }
  stages:
    - regex: { expression: 'user_id=(?P<user_id>\d+)' }
    - labels: { user_id: ~ }   # ← this is the bug
# Effect: every distinct user creates a new stream, index explodes.
# Fixed: don't promote user_id to a label; query lines with `|= "user_id=42"` or parse at query time.

# Gotcha 4 — using regexp when logfmt/json/pattern would work
# Bad:
{app="api"} | regexp "level=(?P<level>\\w+) msg=\"(?P<msg>[^\"]+)\" duration=(?P<dur>\\S+)"
# Fixed: logfmt is built for this
{app="api"} | logfmt

# Gotcha 5 — missing unwrap for numeric aggregation
# Bad:
quantile_over_time(0.99, {app="api"} | logfmt [5m])
# Error: "quantile_over_time requires unwrap"
# Fixed:
quantile_over_time(0.99, {app="api"} | logfmt | unwrap duration_ms [5m])

# Gotcha 6 — backslash escaping in regex
# Bad:
{app="api"} |~ "\d+"
# Error: "regex parse error" (because \d isn't a valid escape inside a quoted LogQL string)
# Fixed: double-escape
{app="api"} |~ "\\d+"

# Gotcha 7 — JSON array indexing without brackets
# Bad:
{app="api"} | json first="items.0.name"
# (silent: returns nothing; "items.0" is interpreted as object key "items.0", not items[0])
# Fixed:
{app="api"} | json first="items[0].name"

# Gotcha 8 — case-sensitive line filter missing matches
# Bad:
{app="api"} |= "Error"
# (misses "ERROR", "error", "ErRoR" lines)
# Fixed:
{app="api"} |~ "(?i)error"

# Gotcha 9 — naming a label with a Go-template-incompatible character
# Bad: label "trace.id" can't be referenced as {{.trace.id}} (Go thinks it's nested .trace.id)
{app="api"} | json | line_format "trace={{.trace.id}}"
# Fixed: rename on extract
{app="api"} | json trace_id="trace.id" | line_format "trace={{.trace_id}}"

# Gotcha 10 — pipeline stage out of order
# Bad: filter before parser produces the label
{app="api"} | level="error" | logfmt
# (level filter sees no parsed level; matches nothing)
# Fixed:
{app="api"} | logfmt | level="error"

# Gotcha 11 — using = for substring match (PromQL habit)
# Bad:
{app="api"} | logfmt | path="users"
# (only matches when path is EXACTLY "users")
# Fixed:
{app="api"} | logfmt | path=~".*users.*"
# Or even better, line filter first:
{app="api"} |= "users" | logfmt | path="/api/users"

# Gotcha 12 — too-large time range
# Bad:
sum(count_over_time({env="prod"} [30d]))
# Error: "max-query-length exceeded"
# Fixed: split with subquery or recording rule, or shorten range
sum(count_over_time({env="prod"} [7d]))

# Gotcha 13 — stream selector with empty regex
# Bad:
{app=~""}
# Error: "regex must not be empty"
# Fixed: omit the matcher or use `.+`
{app=~".+"}

# Gotcha 14 — bytes unit case-sensitivity
# Bad:
| body_size > 10mb
# Error: "invalid bytes value"
# Fixed: MB or MiB
| body_size > 10MB

# Gotcha 15 — comparing an extracted string to a number
# Bad: status_code is a string until coerced
| status_code > 500
# (works in v2.4+, but pre-2.4 needed unwrap)
# Modern fix:
| status_code >= 500
# Compatibility fix (older Loki):
| __error__="" | unwrap status_code | __value__ >= 500
```

## Performance Tips

Push filters left, push parses right.

```bash
# Slow: parse every line, then filter
{app="api"} | logfmt | level="error" | path=~"/api/users"

# Fast: drop 99% of lines via line filter, then parse only the survivors
{app="api"} |= "level=error" |= "/api/users" | logfmt | level="error" | path="/api/users"
```

Prefer exact `=` over `=~` regex on stream selectors — exact matching uses the inverted index directly, while regex matching falls back to a regex evaluation per stream.

```bash
# Faster
{app="api"}
# Slower (still OK for small alternations)
{app=~"api|web"}
# Slowest (matches every stream)
{app=~".+"}
```

Minimize range duration. A `[1h]` range scans 12x more chunks than `[5m]`. For instant queries (single-point dashboards), pick the smallest range that still gives you the smoothing you want.

Use the cheapest parser:

| Parser  | Relative cost | When                              |
|---------|---------------|-----------------------------------|
| logfmt  | 1x            | logrus, zap, slog, stdlib slog    |
| pattern | 1.5x          | nginx, apache, fixed-position     |
| json    | 2x            | bunyan, pino, structured JSON     |
| regexp  | 5x            | nothing else fits                 |

Avoid creating high-cardinality labels at ingest. The Loki golden rule: **labels are for streams, content is for grep**. The Prometheus rule of thumb (< 10K active series per metric) translates to < 10K active streams per Loki tenant.

```bash
# Loki tunables to know:
# chunk_target_size: 1.5MB           # target chunk size before flushing
# max_chunk_age: 1h                  # force flush after this
# max_streams_per_user: 10000        # cap on streams per tenant
# max_query_series: 500              # cap on output series for metric query
# max_query_length: 721h             # cap on time range
# query_timeout: 1m                  # per-query timeout
# split_queries_by_interval: 30m     # split big queries for parallelism
# max_outstanding_per_tenant: 100    # queue depth
```

Use **recording rules** for dashboards that hit the same expensive query repeatedly:

```bash
groups:
  - name: cached-rates
    interval: 30s
    rules:
      - record: app:log_rate:5m
        expr: sum by (app) (rate({env="prod"}[5m]))
```

Then your dashboard reads `app:log_rate:5m` from Prometheus / Mimir at near-zero cost.

The Loki `query-frontend` and `query-scheduler` components split big queries into time-shards (default 30m) and run them in parallel — make sure they're enabled for any production deployment.

## Cardinality and Best Practices

Same rules as Prometheus, with one twist: **log content is searchable**, so you don't need labels to find things.

DO use labels for:

- Service identity: `app`, `service`, `component`
- Environment: `env`, `cluster`, `region`, `namespace`
- Stable infrastructure: `host`, `instance`, `pod` (within reason)
- Severity: `level`, `severity` (small enum)

DON'T use labels for:

- User identifiers: `user_id`, `email`, `account_id`
- Request identifiers: `trace_id`, `request_id`, `span_id`
- URLs / paths with parameters: `/api/users/42`
- Dynamic IPs (in the long tail)
- Error messages or stack traces
- Anything that can produce > ~100 distinct values

```bash
# DO promote a label only if the cardinality is bounded and stable.
# DON'T panic — high-cardinality fields can still be queried VIA the line:
{app="api"} |= "user_id=42"
{app="api"} | json | trace_id="abc-123"
```

The "filter by label, search by content" mantra:

```bash
# Label = how I narrow which streams to fetch
{app="api", env="prod"}

# Content (line filter / parser) = how I find specific lines within those streams
{app="api", env="prod"} |= "request_id=abc" | json | user_id="42"
```

Cardinality limits are enforced by Loki and exposed in metrics:

```bash
loki_ingester_streams_total          # total active streams
loki_distributor_lines_received_total
loki_distributor_bytes_received_total
loki_request_duration_seconds         # latency
loki_query_frontend_queries_total
loki_logql_querystats_*              # per-query stats
```

Alert on stream growth:

```bash
- alert: LokiStreamsGrowing
  expr: increase(loki_ingester_streams_total[1h]) > 1000
  for: 30m
```

If a label balloons, fix the shipper config:

```bash
# promtail: don't promote dynamic fields
- regex: { expression: 'user_id=(?P<user_id>\d+)' }
  # NOT followed by a labels stage that promotes user_id

# Use static labels at the agent layer, parse dynamic fields at query time.
```

## The /api/v1 HTTP API

Loki exposes a Prometheus-style HTTP API at `/loki/api/v1/...`:

```bash
# Instant metric query (single point in time)
GET /loki/api/v1/query
  ?query=<logql>
  &time=<rfc3339|unix>
  &limit=<n>          # for log queries

# Range query (graph / time-series result)
GET /loki/api/v1/query_range
  ?query=<logql>
  &start=<rfc3339|unix>
  &end=<rfc3339|unix>
  &step=<duration>     # resolution
  &limit=<n>
  &direction=forward|backward

# Label autocomplete — list label names
GET /loki/api/v1/labels
  ?start=<...>&end=<...>

# Label-values autocomplete — list values for a given label
GET /loki/api/v1/label/<name>/values
  ?start=<...>&end=<...>

# Stream listing — list active streams matching a selector
GET /loki/api/v1/series
  ?match[]={app="api"}
  &start=<...>&end=<...>

# Tail (websocket / chunked stream)
GET /loki/api/v1/tail
  ?query=<logql>
  &limit=<n>
  &start=<...>
  &delay_for=<seconds>
```

curl examples:

```bash
# Instant metric
curl -G "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query=sum(rate({app="api"} |= "ERROR" [5m]))' \
  --data-urlencode 'time=2024-01-15T10:00:00Z'

# Range
curl -G "http://loki:3100/loki/api/v1/query_range" \
  --data-urlencode 'query={app="api"} |= "ERROR"' \
  --data-urlencode 'start=2024-01-15T09:00:00Z' \
  --data-urlencode 'end=2024-01-15T10:00:00Z' \
  --data-urlencode 'step=60s' \
  --data-urlencode 'limit=1000' \
  --data-urlencode 'direction=backward'

# Label values for autocomplete
curl -G "http://loki:3100/loki/api/v1/label/app/values"

# Active streams matching a selector
curl -G "http://loki:3100/loki/api/v1/series" \
  --data-urlencode 'match[]={env="prod"}'

# Tail (curl can't websocket; use logcli instead)
logcli query --tail '{app="api"}'
```

Multi-tenant: pass `X-Scope-OrgID: <tenant>` header.

```bash
curl -H 'X-Scope-OrgID: team-a' -G \
  "http://loki:3100/loki/api/v1/query" \
  --data-urlencode 'query={app="api"}'
```

Response shapes (matrix / vector / streams) are the same as Prometheus:

```bash
{
  "status": "success",
  "data": {
    "resultType": "matrix",         # matrix | vector | streams
    "result": [
      {
        "metric": { "app": "api" },  # or "stream": for log queries
        "values": [[1705314000, "0.5"], [1705314060, "0.6"]]
      }
    ],
    "stats": { ... }                # query stats
  }
}
```

Useful tools:

```bash
logcli                          # official CLI from grafana/loki repo
logcli query '{app="api"}' --since=1h --output=jsonl
logcli labels                   # list label names
logcli labels app               # values for label app
logcli series '{env="prod"}'

promtail-tool                   # pipeline debugging (Promtail config validation)
```

## Promtail / Grafana Agent / OpenTelemetry Collector

The shippers that get logs INTO Loki define what labels exist (and therefore what your stream selectors can match on). Three options:

**Promtail** — Loki's native shipper. Tail files, scrape Kubernetes pod logs, journal, syslog, GCP / AWS log streams. Pipeline stages parse-at-ingest:

```bash
# /etc/promtail/config.yaml
server: { http_listen_port: 9080 }
clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: kubernetes-pods
    kubernetes_sd_configs: [{ role: pod }]
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        target_label: app
      - source_labels: [__meta_kubernetes_namespace]
        target_label: namespace
    pipeline_stages:
      - cri: {}                          # parse CRI (containerd) format
      - json:
          expressions:
            level: level
            msg: msg
      - labels: { level: ~ }             # promote level to a label
      - timestamp:
          source: time
          format: RFC3339
      - multiline:                       # join stack traces
          firstline: '^\d{4}-\d{2}-\d{2}'
          max_lines: 128
          max_wait_time: 3s
      - tenant: { source: tenant_id }    # multi-tenancy
```

The `pipeline_stages` are evaluated in order and can: parse (json, regex, logfmt, cri, docker, pack), transform (template, replace, decolorize), promote labels, set the timestamp, drop lines, control multiline batching.

**Grafana Agent** (now "Grafana Alloy" since 2024) — superset of Promtail with the same pipeline stages plus Prometheus scraping, OTLP receivers, and remote-write of metrics. Same ingest semantics for Loki.

**OpenTelemetry Collector** — vendor-neutral. Use the `loki` exporter to push logs:

```bash
# otel-collector-config.yaml
receivers:
  filelog:
    include: [/var/log/myapp/*.log]
    operators:
      - type: regex_parser
        regex: 'level=(?P<level>\w+) msg="(?P<msg>[^"]+)"'
processors:
  resource:
    attributes:
      - key: service.name
        value: api
        action: upsert
exporters:
  loki:
    endpoint: http://loki:3100/loki/api/v1/push
    labels:
      resource:
        service.name: "service"            # promote OTel resource attributes to Loki labels
        deployment.environment: "env"
service:
  pipelines:
    logs: { receivers: [filelog], processors: [resource], exporters: [loki] }
```

The OTel approach lets you ship the same logs to Loki AND another backend (e.g., for compliance archival) with one collector.

Multi-line: stack traces and other multi-line log entries need to be joined into a single Loki log line at ingest. Promtail's `multiline` stage and OTel's `recombine` operator do this.

```bash
# Promtail multiline for Java stack traces
- multiline:
    firstline: '^\S+'              # any line not starting with whitespace begins a new entry
    max_lines: 200
    max_wait_time: 3s
```

Tenant headers: when Loki runs multi-tenant (-auth.enabled=true), every push must include `X-Scope-OrgID: <tenant>`. Promtail does this via the `tenant` pipeline stage or static `tenant_id` config.

## Idioms

```bash
# 1. Canonical service-error pipeline:
{app="X", env="prod"} | logfmt | level="error" | line_format "{{.msg}} ({{.path}})"

# 2. Log volume by service (capacity planning):
sum by (app) (bytes_rate({env="prod"}[5m]))

# 3. Find a needle in a haystack (the trace_id pattern):
{env="prod"} |= "trace_id=01H..." | json

# 4. Debug a specific user's session:
{app="api"} | json | user_id="42" | line_format "{{.method}} {{.path}} {{.status_code}} {{.duration_ms}}ms"

# 5. Top-N noisy streams:
topk(10, sum by (app, level) (count_over_time({env="prod"}[5m])))

# 6. Rate of HTTP errors per route:
sum by (path) (rate({app="api"} | json | status_code >= 500 [5m]))

# 7. Latency p99 by route:
quantile_over_time(0.99, {app="api"} | json | unwrap duration_ms [5m]) by (path)

# 8. Detect log-volume drop (canary missing logs):
absent_over_time({app="canary"}[5m])

# 9. Rolling diff — compare error rate now vs 1h ago:
sum(rate({env="prod"} |= "ERROR" [5m]))
  /
sum(rate({env="prod"} |= "ERROR" [5m] offset 1h))

# 10. Group small enum values, bucket large ones:
sum by (status_class) (
  rate({app="api"}
    | json
    | label_format status_class="{{ if lt .status_code 300 }}2xx{{ else if lt .status_code 400 }}3xx{{ else if lt .status_code 500 }}4xx{{ else }}5xx{{ end }}"
    [5m])
)

# 11. Detect missing field (instrumentation regression):
sum(count_over_time({app="api"} | json | trace_id="" [5m]))

# 12. Convert log lines into single-line JSON for piping:
{app="api"} | json | line_format "{{toJson .}}"

# 13. Tail a service in real time:
logcli query --tail '{app="api", env="prod"}'

# 14. Count unique users per hour (HyperLogLog approximation via grouping):
count(count by (user_id) (count_over_time({app="api"} | json | user_id!="" [1h])))

# 15. Rolling 95th-percentile alert pattern:
quantile_over_time(0.95, {app="api"} | logfmt | unwrap dur [5m]) > 0.5

# 16. Slowest 10 endpoints:
topk(10, avg by (path) (avg_over_time({app="api"} | logfmt | unwrap duration [5m])))

# 17. Total log bytes by tenant (cost dashboard):
sum by (tenant) (bytes_over_time({}[1h]))

# 18. Service is "down" if no logs for 10m:
absent_over_time({app=~"api|web|worker"}[10m])

# 19. Compare two deploys:
{app="api"} |= "version=2.0" |= "ERROR" / {app="api"} |= "version=2.0"

# 20. The Big-Number panel for "errors today":
sum(count_over_time({env="prod"} |= "ERROR" [24h]))
```

## Tips

- **Always paste-test in Grafana Explore first.** The query field shows incremental result counts so you can iterate.
- **Time range matters.** A query that's snappy at 5m can be a planet-scorcher at 7d. Test scaling.
- **Use the Loki query inspector.** Grafana Explore has an "Inspector" panel showing total bytes scanned, time spent, and chunks fetched. Anything > 1GB scanned for a single query is suspect.
- **Stats endpoint:** `GET /loki/api/v1/index/stats?query={app="api"}` returns chunk count, stream count, and bytes — useful for capacity / cost calculations.
- **Volume endpoint** (Loki 2.9+): `GET /loki/api/v1/index/volume?query={app="api"}&aggregateBy=labels` returns per-label volume — find your noisiest streams.
- **Use `--analyze-labels`** in logcli to inspect cardinality:

  ```bash
  logcli analyze-labels '{env="prod"}'
  ```

- **Use `logcli stats`** to see byte volume, chunk counts, stream counts:

  ```bash
  logcli stats '{env="prod"}' --since=1h
  ```

- **Don't depend on log line ordering across streams.** Within a stream lines are timestamp-ordered; across streams the merge depends on `direction=` and step.
- **Watch your Grafana Loki datasource `maxLines` setting.** Default 1000; raise for explore queries (but lower for dashboards).
- **Recording rules pre-compute LogQL → Prometheus series.** Push them to Mimir / VictoriaMetrics for persistent metrics that survive Loki retention.
- **For multi-tenant clusters always send `X-Scope-OrgID`.** Forgetting it routes you to the default tenant.
- **`__error__` is your friend.** Filter `__error__=""` on aggregations, filter `__error__!=""` to debug parser failures.
- **Use bracket notation in JSON parser** for keys with special characters: `| json x="$['weird.key']"`.
- **Subqueries (Loki 2.4+):** `count_over_time((rate({app="api"}[5m]))[1h:5m])` works the same as PromQL — useful for alerting.
- **Per-query stats are exposed in metrics** — use `loki_logql_querystats_bytes_processed_per_seconds` to find slow queries.
- **Loki uses RE2 regex** — no backreferences, no lookahead/lookbehind. If you need them, parse first then post-filter via `__error__` and label filters.
- **Loki 2.x vs 3.x:** Loki 3.0 (Apr 2024) defaults to TSDB index, deprecates BoltDB-shipper, requires schema v13. Query syntax is unchanged.
- **`drop` and `keep`** are post-Loki-2.7 / 2.9 — if your cluster is older, simulate via `label_format empty=""` and rely on aggregation `by()` clauses.
- **Avoid OR in stream selectors when an alternation regex would do** — `{app="api"} or {app="web"}` is invalid; use `{app=~"api|web"}`.
- **Beware of greedy regex in `pattern`** — `<name>` matches non-whitespace, but the literal between captures must be exact, including spacing. A double space in the pattern won't match a single space in the line.
- **For Kubernetes deploys, get `loki-stack` or the `loki-distributed` Helm chart.** Production deploys split out distributor / ingester / querier / compactor / query-frontend.
- **The `pack` / `unpack` flow is for label-cardinality reduction.** Prefer it over high-cardinality labels at ingest.
- **Snappy compression is the default chunk format** — fast decompression, ~3x compression ratio for typical logs.
- **TSDB index (Loki 2.8+) is faster than BoltDB for series-heavy workloads.** Migrate via schema config.
- **Use `logcli query --batch=N`** to paginate large result sets without hitting `max-entries-limit`.

## See Also

- loki — the log database that hosts LogQL
- prometheus — the metrics database whose query language inspired LogQL
- promql — the query language LogQL is modeled on (operators, range vectors, aggregations)
- grafana — the canonical UI for running LogQL via Explore and dashboards
- alertmanager — receives alerts fired by the Loki ruler from LogQL expressions
- opentelemetry — the vendor-neutral shipper that can push logs to Loki via the loki exporter

## References

- LogQL reference: https://grafana.com/docs/loki/latest/logql/
- Query examples: https://grafana.com/docs/loki/latest/query/
- Log queries: https://grafana.com/docs/loki/latest/query/log_queries/
- Metric queries: https://grafana.com/docs/loki/latest/query/metric_queries/
- Template functions: https://grafana.com/docs/loki/latest/query/template_functions/
- Loki HTTP API: https://grafana.com/docs/loki/latest/reference/api/
- Loki ruler / alerting: https://grafana.com/docs/loki/latest/alert/
- Promtail pipeline stages: https://grafana.com/docs/loki/latest/send-data/promtail/stages/
- Best practices: https://grafana.com/docs/loki/latest/best-practices/
- Cardinality / label discipline: https://grafana.com/blog/2020/04/21/how-labels-in-loki-can-make-log-queries-faster-and-easier/
- The Loki Story (Grafana blog series): https://grafana.com/blog/2022/12/22/the-future-of-cloud-native-log-management/
- LogQL grammar (source of truth): https://github.com/grafana/loki/blob/main/pkg/logql/syntax/parser.go
- RE2 regex syntax: https://github.com/google/re2/wiki/Syntax
- logfmt format: https://brandur.org/logfmt
- OpenTelemetry Loki exporter: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/exporter/lokiexporter
