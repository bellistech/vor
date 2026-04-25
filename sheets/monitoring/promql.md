# PromQL (Prometheus Query Language)

Functional query language for selecting and aggregating Prometheus time-series data — the lingua franca of metrics, alerting rules, recording rules, and Grafana dashboards.

## Setup

PromQL is the query language consumed by the Prometheus server, by recording/alerting rule engines, and by every Grafana panel that targets a Prometheus data source. You write PromQL in three places: the built-in Prometheus expression browser at `http://prom:9090/graph`, the HTTP API endpoints `/api/v1/query` and `/api/v1/query_range`, and rule YAML loaded by the server at startup.

```bash
# Built-in expression browser
http://prom:9090/graph

# Instant query — single evaluation at "now"
curl -G --data-urlencode 'query=up' http://prom:9090/api/v1/query

# Instant query at a specific timestamp
curl -G \
  --data-urlencode 'query=up' \
  --data-urlencode 'time=2026-04-25T12:00:00Z' \
  http://prom:9090/api/v1/query

# Range query — evaluation at every step over a window
curl -G \
  --data-urlencode 'query=rate(http_requests_total[5m])' \
  --data-urlencode 'start=2026-04-25T11:00:00Z' \
  --data-urlencode 'end=2026-04-25T12:00:00Z' \
  --data-urlencode 'step=30s' \
  http://prom:9090/api/v1/query_range
```

The data model is a stream of (timestamp, value) pairs called a sample. Every sample lives inside a unique time series identified by its metric name plus its set of label key/value pairs. A sample is the atomic unit; a time series is the indexed sequence of samples; PromQL operates on collections of time series.

```bash
# A single sample
{ timestamp: 1714046400.000, value: 42.0 }

# A single time series (logical view)
http_requests_total{job="api", instance="host:9090", method="GET", status="200"}
  -> [
    (1714046370.000, 11200),
    (1714046400.000, 11250),
    (1714046430.000, 11302)
  ]
```

A query returns one of four data types: a scalar (single number, no labels), a string (rare), an instant vector (a set of series, one sample each at the same timestamp), or a range vector (a set of series, multiple samples each across a time window). Most expressions evaluate to instant vectors.

```bash
# Scalar
42

# Instant vector — one sample per series at evaluation time
up{job="api"} 1
up{job="db"}  1
up{job="lb"}  0

# Range vector — many samples per series across a window
http_requests_total{job="api"}[1m]
  -> [(t-60s, 11000), (t-45s, 11015), (t-30s, 11030), (t-15s, 11045), (t, 11060)]
```

Instant queries return immediately at one timestamp. Range queries return one instant evaluation per `step` between `start` and `end`. Grafana panels run range queries; alert evaluation runs instant queries; the expression browser supports both via the Console (instant) and Graph (range) tabs.

```bash
# Verify a Prometheus reachable
curl -s http://prom:9090/-/healthy
curl -s http://prom:9090/-/ready

# List all metric names
curl -G --data-urlencode 'match[]={__name__=~".+"}' \
  http://prom:9090/api/v1/label/__name__/values

# List all label names for a metric
curl -G --data-urlencode 'match[]=http_requests_total' \
  http://prom:9090/api/v1/labels
```

## Data Model

Every metric in Prometheus is a function-like name that maps a set of labels to a numerical value over time. The metric name is conventionally written as a separate identifier but is internally stored as the special label `__name__`. The unique identity of a series is the entire label set including `__name__`; change any label and you have a different series.

```bash
# Conventional notation
http_requests_total{job="api", instance="host:9090", method="GET", status="200"}

# Equivalent canonical form (every series IS a label set)
{__name__="http_requests_total", job="api", instance="host:9090", method="GET", status="200"}

# Two distinct series — different status label
http_requests_total{job="api", method="GET", status="200"}
http_requests_total{job="api", method="GET", status="500"}
```

The `__name__` label is selectable like any other and is the only label allowed to start with a double underscore (other `__*__` labels are reserved for internal use and are stripped at ingestion).

```bash
# Selecting by __name__ — the explicit form
{__name__="up"}

# Selecting by __name__ regex — match every metric ending in _total
{__name__=~".+_total"}

# Equivalent shorthand — bare metric name
up
```

Prometheus exposes four metric types in its exposition format. PromQL itself is type-agnostic — it sees only floats — but functions assume specific types and behave incorrectly on the wrong one.

```bash
# Counter — monotonically increasing, only-resets-on-process-restart
# Suffix convention: _total
http_requests_total
errors_total

# Gauge — can go up or down
node_memory_MemAvailable_bytes
node_filesystem_free_bytes
node_load1

# Histogram — explodes into _bucket{le="..."} + _count + _sum on the wire
http_request_duration_seconds_bucket{le="0.1"}
http_request_duration_seconds_bucket{le="0.5"}
http_request_duration_seconds_bucket{le="+Inf"}
http_request_duration_seconds_count
http_request_duration_seconds_sum

# Summary — pre-computed quantiles + _count + _sum on the wire
rpc_duration_seconds{quantile="0.5"}
rpc_duration_seconds{quantile="0.9"}
rpc_duration_seconds{quantile="0.99"}
rpc_duration_seconds_count
rpc_duration_seconds_sum
```

Native histograms (Prometheus 2.40+, stable in 3.0) replace the bucket explosion with a single series whose value carries an entire bucket vector. They use the same metric name as the classic form but appear without the `_bucket` suffix.

```bash
# Native histogram (single series, structured value)
http_request_duration_seconds

# Classic histogram (many series with _bucket suffix)
http_request_duration_seconds_bucket{le="0.1"}
http_request_duration_seconds_bucket{le="+Inf"}
http_request_duration_seconds_count
http_request_duration_seconds_sum
```

Metric naming convention — counters end `_total`, durations end `_seconds`, sizes end `_bytes`, ratios end `_ratio`, info-only metrics end `_info`. The `up` metric is special: Prometheus injects it for every scrape (`1` if the scrape succeeded, `0` if not).

```bash
# Canonical "is the target alive" check
up                                  # all targets
up{job="api"}                       # all instances of one job
up{job="api"} == 0                  # only down instances
sum by (job) (up)                   # how many up per job
count by (job) (up == 0)            # how many down per job
```

## Selectors — Instant Vector

An instant vector selector returns one sample per matching series at the query's evaluation time. The bare metric name is shorthand for `{__name__="metric"}`. Add label matchers in curly braces; multiple matchers are ANDed together.

```bash
# Bare metric — all series of this name
up

# Single label matcher
up{job="api"}

# Multiple matchers (AND)
up{job="api", env="prod"}

# Same with explicit __name__
{__name__="up", job="api", env="prod"}
```

Four matcher operators exist. `=` and `!=` test exact equality; `=~` and `!~` test against an RE2 regular expression that is anchored on both ends (an implicit `^...$`).

```bash
# Equal
http_requests_total{method="GET"}

# Not equal
http_requests_total{method!="GET"}

# Regex match (anchored — ^...$ implicit)
http_requests_total{status=~"5.."}            # all 5xx
http_requests_total{status=~"4..|5.."}        # 4xx or 5xx
http_requests_total{path=~"/api/v[12]/.*"}

# Negated regex match
http_requests_total{path!~"/health|/ready"}

# Empty string matches series that don't have the label at all
http_requests_total{env=""}                   # series missing env label
http_requests_total{env!=""}                  # series WITH env label
```

A selector that matches no series produces an empty result, not an error. Alerting rules that should fire when a metric goes missing must use `absent()` or `absent_over_time()` rather than relying on an empty selector.

```bash
# Returns empty result silently when no series match
up{job="nonexistent"}

# Use absent() to detect "metric is missing"
absent(up{job="api"})
```

By default, an instant selector pulls the most recent sample within the staleness window (default 5 minutes). Sample more than 5 minutes old is considered stale and not returned. The `@` modifier pins evaluation to a specific Unix timestamp.

```bash
# At a specific time (seconds since epoch)
up @ 1714046400

# At "start" of the range query
up @ start()

# At "end" of the range query
up @ end()

# Offset — shift the lookback window backwards
up offset 1h
http_requests_total offset 1h
```

The `offset` modifier and the `@` modifier are independent and composable.

```bash
# Compare current rate to rate one week ago
rate(http_requests_total[5m])
  /
rate(http_requests_total[5m] offset 1w)
```

## Selectors — Range Vector

A range vector selector returns the samples of each matching series over a time window. The duration is suffixed in square brackets and uses Prometheus duration syntax: `ms`, `s`, `m`, `h`, `d`, `w`, `y`. Range vectors cannot be displayed directly — the expression browser will reject them. They exist solely as input to range-vector functions.

```bash
# Last 5 minutes of samples for each series
http_requests_total[5m]

# Other valid durations
metric[30s]
metric[1m]
metric[2h]
metric[1d]
metric[1w]

# Compound durations (Prometheus 2.7+)
metric[1h30m]
metric[1d12h]
```

Range vectors are consumed by `rate()`, `irate()`, `increase()`, `delta()`, `idelta()`, `resets()`, `changes()`, `deriv()`, `predict_linear()`, and the entire `*_over_time()` family. Without a wrapping function, the query errors:

```bash
# WRONG — range vector cannot be a final result
http_requests_total[5m]
# Error: invalid expression type "range vector" for range query, must be Scalar or instant Vector

# RIGHT — wrapped in a range-vector function
rate(http_requests_total[5m])
```

The window must be at least 4× the scrape interval to give `rate()` enough samples to handle a counter reset reliably. With a 30s scrape interval, `[2m]` is the absolute minimum and `[5m]` is the recommended default.

```bash
# 30s scrape interval — fragile, only ~2 samples
rate(http_requests_total[1m])

# 30s scrape interval — robust, ~10 samples
rate(http_requests_total[5m])
```

`offset` and `@` work on range vectors too.

```bash
# Rate over the 5-minute window ending one hour ago
rate(http_requests_total[5m] offset 1h)

# Same window pinned at a specific timestamp
rate(http_requests_total[5m] @ 1714046400)
```

## Selectors — Subqueries

A subquery evaluates an instant-vector expression repeatedly across a window, producing a synthetic range vector that any range-vector function can consume. Syntax: `<expr>[<window>:<resolution>]`. Resolution is optional and defaults to the global evaluation interval.

```bash
# Evaluate rate() every 30s over the last 5 minutes
rate(http_requests_total[1m])[5m:30s]

# This produces a range vector you can feed to a *_over_time() function
max_over_time(rate(http_requests_total[1m])[5m:30s])
```

The canonical use case is "max-of-rates" — find the peak rate over a longer window than rate's range can cover. You cannot just write `max_over_time(rate(metric[5m]))` because `max_over_time` needs a range vector and `rate(metric[5m])` is an instant vector.

```bash
# Peak per-second request rate observed in any 1m window over the last 5m
max_over_time(rate(http_requests_total[1m])[5m:30s])

# Average p99 latency over the last hour, computed every 1m
avg_over_time(
  histogram_quantile(0.99,
    sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
  )[1h:1m]
)
```

Subqueries are expensive — they trigger a "second-pass evaluation" where Prometheus runs the inner query at every resolution step and stores the synthetic samples in memory. Use recording rules to precompute the inner expression whenever a subquery would run on a dashboard or alert.

```bash
# Subquery — slow, recomputes inner each evaluation step
max_over_time(rate(http_requests_total[1m])[5m:30s])

# Recording rule (precomputed) — cheap
job:http_requests:rate1m
# In rules.yml:
# - record: job:http_requests:rate1m
#   expr: rate(http_requests_total[1m])
# Then query:
max_over_time(job:http_requests:rate1m[5m])
```

## Operators — Arithmetic

PromQL supports `+`, `-`, `*`, `/`, `%` (modulo), `^` (power) on scalars, vectors, and combinations. Vector-vector operations require label matching: only series with identical label sets on both sides are paired, and the result keeps the matched labels but drops `__name__`.

```bash
# Scalar-scalar
2 + 3                           # = 5

# Vector-scalar — applied to every sample
node_memory_MemTotal_bytes / 1024 / 1024 / 1024     # bytes -> GiB

# Vector-vector — implicit one-to-one matching on identical label sets
node_filesystem_avail_bytes / node_filesystem_size_bytes
```

When the two sides have different label sets, you must specify how to match. `on(labels)` keeps only those labels for matching; `ignoring(labels)` keeps everything except those labels.

```bash
# Match only on instance label
metric_a / on(instance) metric_b

# Match on everything except job label
metric_a / ignoring(job) metric_b
```

Many-to-one and one-to-many require `group_left()` or `group_right()`. The "left/right" refers to which side has the *many*. The arg list inside the parens specifies labels to copy from the "one" side onto the "many" result.

```bash
# Many-to-one: requests-per-version where versions live in build_info{version="..."}
sum by (instance) (rate(http_requests_total[5m]))
  * on(instance) group_left(version)
sum by (instance, version) (build_info)

# One-to-many: spread a per-region SLO over instances
slo_target_per_region
  * on(region) group_right(instance)
sum by (instance, region) (rate(metric[5m]))
```

The result of any vector-vector arithmetic strips `__name__` because the result is no longer "the same metric". Wrap with `label_replace()` if you need a name back.

```bash
# Result has no __name__ — that's expected
node_filesystem_avail_bytes / node_filesystem_size_bytes
```

## Operators — Comparison

Comparison operators are `==`, `!=`, `>`, `<`, `>=`, `<=`. In vector context they act as filters — only series whose value satisfies the comparison are returned, and the original value is preserved.

```bash
# Filter: only return series with value > 100
http_requests_total > 100

# Filter: only down targets
up == 0

# Filter: error rate above 1 percent
rate(errors_total[5m]) / rate(requests_total[5m]) > 0.01
```

The `bool` modifier converts the comparison into a 0/1 result on every series rather than filtering.

```bash
# Filter — only series where rate > 100 are returned
rate(http_requests_total[5m]) > 100

# bool — every series returned, value is 1 if true else 0
rate(http_requests_total[5m]) > bool 100

# Useful inside aggregations
sum(up == bool 0)                # how many targets are down
```

Two vectors can be compared directly — same matching rules as arithmetic.

```bash
# Series where memory used exceeds memory limit
container_memory_usage_bytes > container_spec_memory_limit_bytes

# With matching qualifier
metric_a > on(instance) metric_b
```

The canonical alert pattern is `expr > threshold` with `for: <duration>` to require persistence.

```bash
# Alert when CPU has been above 80% for 5 minutes
- alert: HighCPU
  expr: avg by (instance) (rate(node_cpu_seconds_total{mode!="idle"}[5m])) > 0.8
  for: 5m
```

## Operators — Logical / Set

Set operators combine instant vectors based on label-set membership. They are NOT applied per sample value; they only test for the presence of matching label sets.

```bash
# AND — keeps series from left side that ALSO appear on right side
metric_a and metric_b

# OR — union: series from left, plus series from right that aren't in left
metric_a or metric_b

# UNLESS — series from left that DO NOT appear on right (set difference)
metric_a unless metric_b
```

Matching uses identical label sets by default; refine with `on()` or `ignoring()`.

```bash
# Keep only error rates for endpoints that have non-zero traffic
rate(errors_total[5m]) and on(endpoint) (rate(requests_total[5m]) > 0)

# All up targets, falling back to "up offset 1h" for any that are currently missing
up or up offset 1h

# Targets that are down AND not in maintenance
up == 0 unless on(instance) maintenance_mode == 1
```

A common alerting idiom uses `unless` to suppress alerts during known events.

```bash
# Fire only outside the maintenance window
ALERT_EXPR unless on(cluster) maintenance{active="1"}
```

## Operator Precedence

Operators bind from highest to lowest precedence as follows. Same-precedence operators are left-associative except `^`, which is right-associative.

```bash
# 1. ^                                  (right-associative)
# 2. *, /, %, atan2
# 3. +, -
# 4. ==, !=, <=, >=, <, >
# 5. and, unless
# 6. or
```

Effects in practice:

```bash
# 2 + 3 * 4    = 14, not 20  (multiplication binds tighter)
# 2 ^ 3 ^ 2    = 512, not 64 (right-associative)
# a == b and c = (a == b) and c
# a or b and c = a or (b and c)         (and binds tighter than or)
```

When uncertain, wrap with parentheses. The expression browser is forgiving about whitespace but strict about precedence.

```bash
# Ambiguous to readers — parenthesize
sum(rate(metric[5m])) by (job) > 100

# Explicit and unambiguous
(sum by (job) (rate(metric[5m]))) > 100
```

## Aggregation Operators

Aggregation operators collapse multiple series into fewer series along a chosen label dimension. Syntax: `<aggr>(<expr>)` aggregates everything into one series; `<aggr> by (labels) (<expr>)` groups by those labels; `<aggr> without (labels) (<expr>)` keeps every label except those listed.

```bash
# Available aggregators
sum            # add values
min            # minimum value
max            # maximum value
avg            # arithmetic mean
group          # constant 1, useful for set ops
stddev         # population standard deviation
stdvar         # population variance
count          # number of series
count_values   # number of series per distinct value (takes a label-name argument)
bottomk        # k smallest series (takes int argument)
topk           # k largest series (takes int argument)
quantile       # quantile of values (takes 0..1 argument)
```

Group-by clauses can be placed before or after the expression — both forms are valid; "before" reads left-to-right, "after" mirrors SQL `GROUP BY`.

```bash
# Both equivalent
sum by (job) (rate(http_requests_total[5m]))
sum(rate(http_requests_total[5m])) by (job)

# without — keep all labels EXCEPT instance
sum without (instance) (rate(http_requests_total[5m]))
```

Common patterns:

```bash
# Total request rate per job
sum by (job) (rate(http_requests_total[5m]))

# Mean CPU per instance (across cores)
avg by (instance) (rate(node_cpu_seconds_total{mode="user"}[5m]))

# How many series per metric (cardinality check)
count by (__name__) ({__name__=~".+"})

# Histogram of values
count_values("le", up)              # how many series at each up value (0 or 1)

# Top-5 endpoints by request rate
topk(5, sum by (path) (rate(http_requests_total[5m])))

# Bottom-5 instances by free memory
bottomk(5, node_memory_MemAvailable_bytes)

# 50th-percentile request rate across all instances
quantile(0.5, rate(http_requests_total[5m]))
```

`topk` and `bottomk` keep the original labels — they don't aggregate, they select. Because they preserve labels, they can produce different series at different timestamps in a range query, which leads to broken-looking Grafana lines.

```bash
# May "flicker" in Grafana as the top-5 set changes over time
topk(5, rate(http_requests_total[5m]))
```

`group()` is a useful no-op aggregator that returns 1 per group — useful as a label-set generator inside `or` clauses.

## Functions — Rate Family

This is the most error-prone area of PromQL. Six functions look similar; choose carefully.

```bash
# rate(counter[range])
#   Per-second average increase across the entire range.
#   Use this for graphs and alerts on counter metrics.
#   Handles counter resets transparently (an apparent decrease
#   is treated as a reset and adjusted).
rate(http_requests_total[5m])

# irate(counter[range])
#   Per-second instantaneous rate — uses ONLY the last two samples
#   in the range. Very responsive, very noisy. Good for
#   troubleshooting at high resolution; bad for alerts.
irate(http_requests_total[5m])

# increase(counter[range])
#   Total increase across the range, in absolute units (not per-second).
#   Mathematically: rate(...) * range_in_seconds, with reset handling.
increase(http_requests_total[5m])         # requests in the last 5m

# delta(gauge[range])
#   Difference between first and last sample in the range, no reset
#   handling. Use ONLY on gauges. Negative values are valid.
delta(node_memory_MemAvailable_bytes[1h])

# idelta(gauge[range])
#   Difference between the last two samples. Gauge analog of irate.
idelta(node_memory_MemAvailable_bytes[5m])

# resets(counter[range])
#   Count of detected counter resets in the range.
#   Useful to detect process restarts.
resets(http_requests_total[1h])

# changes(gauge[range])
#   Count of value changes in the range. Useful on gauges/state metrics.
changes(node_boot_time_seconds[1h])

# deriv(gauge[range])
#   Per-second derivative computed via simple linear regression.
#   For gauges only.
deriv(node_filesystem_free_bytes[1h])
```

The "always rate, almost never irate" rule:

```bash
# RIGHT — for any graph or alert
rate(http_requests_total[5m])

# WRONG — irate creates spikes when scrapes shift slightly
# irate is for short ad-hoc inspection only
irate(http_requests_total[5m])
```

The "rate before sum, never sum before rate" rule applies because counters reset independently — summing first throws away the per-series reset events:

```bash
# RIGHT
sum by (job) (rate(http_requests_total[5m]))

# WRONG — the sum can decrease (= "negative rate") on a single instance reset
rate(sum by (job) (http_requests_total)[5m])
```

`increase()` is rate scaled — pick whichever fits the question:

```bash
# Per-second
rate(http_requests_total[5m])

# Total over the window
increase(http_requests_total[5m])

# These two are equivalent (mathematically; ignoring reset edge cases):
increase(metric[5m]) == rate(metric[5m]) * 300
```

## Functions — Time

Time-related functions return scalars or vectors based on the wall clock at evaluation time. They are essential for alerting on time-since-event.

```bash
# Current Unix timestamp in seconds
time()                              # = 1714046400

# Timestamp of each sample's arrival
timestamp(metric)                   # one sample per series, value = its timestamp

# How long since a metric was last updated
time() - timestamp(last_event_time_seconds)

# How long since process started
time() - process_start_time_seconds

# Date components — every function takes optional vector argument,
# defaulting to current evaluation time.
day_of_month()                      # 1-31
day_of_week()                       # 0=Sunday .. 6=Saturday
day_of_year()                       # 1-365 (366 in leap years)
days_in_month()                     # 28-31
hour()                              # 0-23
minute()                            # 0-59
month()                             # 1-12
year()                              # e.g. 2026

# Pass a vector to compute per-series
day_of_week(timestamp(metric))
```

Use these for office-hours-only alerts:

```bash
# Suppress overnight alerts (UTC)
ALERT_EXPR
  and on() (hour() >= 9 < 18)
  and on() (day_of_week() > 0 < 6)
```

## Functions — Math

Standard math functions operate per-sample on instant vectors and return a same-shape vector.

```bash
abs(v)                              # absolute value
ceil(v)                             # round up
floor(v)                            # round down
round(v)                            # round to nearest int
round(v, m)                         # round to nearest multiple of m
exp(v)                              # e^v
ln(v)                               # natural log
log2(v)                             # base-2 log
log10(v)                            # base-10 log
sqrt(v)                             # square root
sgn(v)                              # sign: -1, 0, or 1

# Trig (Prometheus 2.26+)
sin(v) cos(v) tan(v)
asin(v) acos(v) atan(v)
sinh(v) cosh(v) tanh(v)
asinh(v) acosh(v) atanh(v)
deg(v)                              # radians -> degrees
rad(v)                              # degrees -> radians
pi()                                # constant 3.14159...

# atan2 (Prometheus 2.36+)
metric_a atan2 metric_b             # binary operator, NOT a function call
```

Clamp helpers limit values to a range — useful when noisy data must stay inside a sane bound.

```bash
clamp(metric, 0, 100)               # min=0, max=100 — value pinned to [0,100]
clamp_min(metric, 0)                # never less than 0
clamp_max(metric, 1)                # never more than 1

# Sanitize a ratio that should be in [0,1]
clamp(rate(errors[5m]) / rate(requests[5m]), 0, 1)
```

`scalar()` and `vector()` convert between types.

```bash
# scalar(<single-element-vector>) -> scalar; NaN if the vector has !=1 element
scalar(some_metric{job="api"})

# vector(<scalar>) -> instant vector with no labels
vector(42)

# Combine for arithmetic between scalar-ish vector and a vector
http_requests / scalar(sum(http_requests))
```

## Functions — Histogram

Classic histograms (the dominant form prior to native histograms) expose three distinct series per metric: cumulative `<name>_bucket{le="..."}`, total `<name>_count`, and total `<name>_sum`. Bucket boundaries are inclusive upper bounds; the `+Inf` bucket is mandatory and equals `_count`.

```bash
# Wire format
http_request_duration_seconds_bucket{le="0.005"}    1235
http_request_duration_seconds_bucket{le="0.01"}     1500
http_request_duration_seconds_bucket{le="0.025"}    2120
http_request_duration_seconds_bucket{le="0.05"}     2400
http_request_duration_seconds_bucket{le="0.1"}      2700
http_request_duration_seconds_bucket{le="0.25"}     2950
http_request_duration_seconds_bucket{le="0.5"}      3000
http_request_duration_seconds_bucket{le="1"}        3050
http_request_duration_seconds_bucket{le="2.5"}      3060
http_request_duration_seconds_bucket{le="5"}        3065
http_request_duration_seconds_bucket{le="10"}       3070
http_request_duration_seconds_bucket{le="+Inf"}     3072
http_request_duration_seconds_count                 3072
http_request_duration_seconds_sum                   1247.5
```

`histogram_quantile(quantile, vector)` interpolates the requested quantile from the buckets. The `vector` argument MUST be the rate of the `_bucket` series, with `le` preserved through aggregation.

```bash
# THE canonical p99 latency pattern
histogram_quantile(0.99,
  sum by (le, route) (rate(http_request_duration_seconds_bucket[5m]))
)

# Across the whole service (no per-route breakdown)
histogram_quantile(0.99,
  sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
)

# Multiple quantiles at once via union of separate queries (in Grafana, separate queries)
histogram_quantile(0.50, sum by (le) (rate(metric_bucket[5m])))
histogram_quantile(0.90, sum by (le) (rate(metric_bucket[5m])))
histogram_quantile(0.99, sum by (le) (rate(metric_bucket[5m])))
```

The two cardinal mistakes:

```bash
# WRONG — must take rate of bucket, not raw bucket
histogram_quantile(0.99, http_request_duration_seconds_bucket)

# WRONG — must aggregate by le before passing to histogram_quantile
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))

# RIGHT
histogram_quantile(0.99,
  sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
)
```

Native histograms (Prometheus 2.40+) replace the bucket explosion with a single series per histogram. They unlock additional functions:

```bash
# Native-histogram functions (operate on the SINGLE native histogram series)
histogram_count(metric)             # request count
histogram_sum(metric)               # sum of observations
histogram_avg(metric)               # = sum / count
histogram_fraction(lower, upper, metric)
                                    # fraction of observations in [lower, upper]
histogram_stddev(metric)
histogram_stdvar(metric)
histogram_quantile(q, metric)       # also works on classic with sum-by-le wrap

# Average request latency (works on both forms via the equivalent expression)
rate(http_request_duration_seconds_sum[5m])
  /
rate(http_request_duration_seconds_count[5m])
```

Bucket selection matters: too few buckets gives bad quantile fidelity; too many explodes cardinality. The canonical default for HTTP latency is `0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10`.

## Functions — Sort + Pick

`sort()` and `sort_desc()` order an instant vector by sample value. They affect display only; aggregations downstream are order-independent.

```bash
sort(rate(http_requests_total[5m]))         # ascending
sort_desc(rate(http_requests_total[5m]))    # descending

# Combine with topk for stable display
sort_desc(topk(10, rate(http_requests_total[5m])))
```

`topk(k, v)` and `bottomk(k, v)` are aggregation operators — they return the k series with the largest/smallest values. As noted, they preserve the original label set.

```bash
topk(5, sum by (instance) (rate(http_requests_total[5m])))
bottomk(3, node_memory_MemAvailable_bytes)
```

`quantile(q, v)` is the aggregation form — quantile across series at a single timestamp. Distinct from `histogram_quantile()`, which operates inside a histogram.

```bash
# 95th percentile of free-memory values across all instances
quantile by (job) (0.95, node_memory_MemAvailable_bytes)
```

## Functions — Label Manipulation

`label_replace()` lets you create or rewrite a label using regex extraction from another label.

```bash
# Signature
label_replace(v, dst_label, replacement, src_label, regex)

# Extract host from instance="host:port"
label_replace(up, "host", "$1", "instance", "([^:]+):.*")

# The same with named capture (Go regexp doesn't support named captures
# the way you may expect — use $1, $2, ...)
label_replace(metric, "version", "$1.$2", "fullversion", "(\\d+)\\.(\\d+)\\..*")

# Drop a label by setting it to empty
label_replace(metric, "noisy_label", "", "noisy_label", ".*")

# Add a constant label
label_replace(metric, "team", "platform", "", "")
```

`label_join()` concatenates the values of multiple labels into a destination label.

```bash
# Signature
label_join(v, dst_label, separator, src_label_1, src_label_2, ...)

# Build a "service:env" identifier
label_join(metric, "service_env", ":", "service", "env")

# Build a path from segments
label_join(metric, "path", "/", "namespace", "deployment", "pod")
```

Label functions are read-only on input — they always return new vectors with adjusted labels.

## Functions — Predict

`predict_linear(range_vector, t_seconds)` fits a least-squares line to the data in the range vector and extrapolates `t` seconds into the future. The classic use is "alert when disk will be full".

```bash
# Predict free space 4 hours from now, given the last 1 hour trend.
predict_linear(node_filesystem_free_bytes[1h], 4 * 3600)

# Will be negative if linear trend leads to exhaustion within the prediction horizon
predict_linear(node_filesystem_free_bytes{mountpoint="/"}[1h], 4 * 3600) < 0

# Will run out within an hour
predict_linear(node_filesystem_free_bytes[1h], 3600) < 0
  and on(instance) node_filesystem_free_bytes / node_filesystem_size_bytes < 0.2
```

`deriv(range_vector)` computes the per-second slope using linear regression — useful for "is this metric trending up or down".

```bash
# Increasing memory usage
deriv(node_memory_used_bytes[1h]) > 0

# Pageable disk space leaking
deriv(node_filesystem_free_bytes[6h]) < -1024 * 1024     # < -1 MiB/s
```

## Functions — _over_time Aggregations

These functions take a range vector and return one sample per series, collapsing the time dimension. They are the time-axis analog of `sum/min/max/etc.`.

```bash
avg_over_time(metric[5m])           # arithmetic mean of samples in window
min_over_time(metric[5m])           # minimum
max_over_time(metric[5m])           # maximum
sum_over_time(metric[5m])           # total sum (of values, not increase)
count_over_time(metric[5m])         # how many samples in window
quantile_over_time(0.95, metric[5m])
stddev_over_time(metric[5m])
stdvar_over_time(metric[5m])
last_over_time(metric[5m])          # most recent sample (Prom 2.26+)
present_over_time(metric[5m])       # 1 if at least one sample present
mad_over_time(metric[5m])           # median absolute deviation (Prom 2.46+)
```

The canonical "max-of-rates" pattern uses subqueries:

```bash
# Peak per-second request rate observed in any 1m window over the last 1 hour
max_over_time(rate(http_requests_total[1m])[1h:30s])

# Average p99 latency over the last hour (smoothed)
avg_over_time(
  histogram_quantile(0.99,
    sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
  )[1h:1m]
)
```

`present_over_time` is useful for "did this metric exist at all in the last X" — a complement to `absent_over_time`.

```bash
# Did the deploy event metric appear in the last 30 minutes
present_over_time(deploy_event_total[30m])
```

## Functions — Absent

A selector that matches nothing returns an empty result, not zero. Alerts that should fire when a metric goes missing therefore need explicit absence checks.

```bash
# absent(v) -> 1-with-fallback-labels when v is empty; otherwise empty.
absent(up{job="api"})

# Returns: {job="api"} 1   when no series matches up{job="api"}
# Returns: empty            when up{job="api"} has any series
```

Crucially, `absent()` synthesizes a label set from the matchers in its argument — it copies any literal `=` matchers into the result. Regex matchers contribute nothing. Use this to keep alert annotations meaningful.

```bash
# Result when api is missing:
absent(up{job="api", env="prod"})
# -> {job="api", env="prod"} 1

# Regex matchers are NOT copied into the synthesized label set
absent(up{job=~"api|web"})
# -> {} 1                  # no labels at all on the synthetic series
```

For "the metric has been missing for X minutes" use `absent_over_time` (Prom 2.30+).

```bash
# Fire if no samples for the last 5 minutes
absent_over_time(up{job="api"}[5m])

# Critical alert: API has been completely missing for 10 minutes
- alert: APITargetMissing
  expr: absent_over_time(up{job="api"}[10m])
  for: 0m
  labels:
    severity: critical
  annotations:
    summary: "API target missing for 10m on job={{ $labels.job }}"
```

## The rate() Caveat

`rate()` is the most subtle PromQL function. Internals worth knowing:

1. It computes `(last_value - first_value) / (last_time - first_time)`, with extrapolation to the boundaries of the range.
2. When it detects a counter reset (any apparent decrease), it adds back the pre-reset value to maintain monotonicity.
3. With fewer than 2 samples in the range, it returns no data.
4. It "extrapolates" — if your samples don't quite fill the range, it scales the result up to fill it.

```bash
# At 30s scrape_interval:
rate(http_requests_total[1m])     # ~2 samples — fragile, brittle
rate(http_requests_total[2m])     # ~4 samples — minimum safe
rate(http_requests_total[5m])     # ~10 samples — recommended default
```

Rule of thumb: range duration must be at least 4x the scrape interval. With 15s scrape, `[1m]` is the minimum; with 30s, `[2m]`; with 60s, `[5m]`.

For Grafana panels, use the `$__rate_interval` template variable, which Grafana sets to `max(scrape_interval, 4*step)`.

```bash
# Grafana panel — adapts to dashboard time range and scrape interval
rate(http_requests_total[$__rate_interval])
```

`irate()` uses only the last two samples and is much more responsive but very noisy. It is appropriate for narrow inspection (a 5-minute zoomed view in Grafana) but never for alerts.

```bash
# Use irate ONLY for ad-hoc troubleshooting at high resolution
irate(http_requests_total[1m])
```

Rate cannot be applied to gauges; use `delta()` or `deriv()` for those.

```bash
# Gauge change over time
delta(node_memory_MemAvailable_bytes[1h])
deriv(node_memory_MemAvailable_bytes[1h])
```

## Recording Rules

Recording rules precompute frequent or expensive queries on a fixed interval and store the result as a new metric, allowing instant retrieval. They live in YAML files loaded by Prometheus at startup or via `SIGHUP`.

```bash
# /etc/prometheus/rules/recording.yml
groups:
  - name: api_aggregations
    interval: 30s
    rules:
      - record: job:http_requests:rate5m
        expr: sum by (job) (rate(http_requests_total[5m]))

      - record: job:http_request_errors:rate5m
        expr: sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))

      - record: job:http_request_error_ratio:5m
        expr: |
          job:http_request_errors:rate5m
            /
          job:http_requests:rate5m

      - record: job_route:http_request_duration:p99_5m
        expr: |
          histogram_quantile(0.99,
            sum by (job, route, le) (rate(http_request_duration_seconds_bucket[5m]))
          )
```

Naming convention is `<level>:<metric>:<operations>` (Prometheus best practice from Volz/Brazil's "Up & Running"):

```bash
# instance-level rate of node CPU
instance:node_cpu:rate5m

# job-level p99 latency
job:http_request_duration:p99_5m

# cluster-level error budget consumed
cluster:slo_error_budget:rate30d
```

Rules within a group evaluate sequentially, so a later rule can use the output of an earlier rule in the same group. Across groups, evaluation is parallel.

```bash
# Load the rules
prometheus --config.file=prometheus.yml
# rule_files in prometheus.yml:
# rule_files:
#   - "/etc/prometheus/rules/*.yml"

# Reload without restart
curl -X POST http://prom:9090/-/reload

# Verify rules
curl -s http://prom:9090/api/v1/rules | jq
```

Promtool validates rule files before deployment:

```bash
promtool check rules /etc/prometheus/rules/*.yml
promtool test rules /etc/prometheus/tests/*.yml
```

## Alerting Rules

Alerting rules share the same YAML structure as recording rules but use `alert` instead of `record`. They produce a "firing" alert whenever the expression returns a non-empty vector for the duration specified by `for`.

```bash
# /etc/prometheus/rules/alerts.yml
groups:
  - name: api_alerts
    interval: 30s
    rules:
      - alert: APIHighErrorRate
        expr: |
          sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
            /
          sum by (job) (rate(http_requests_total[5m]))
            > 0.01
        for: 10m
        labels:
          severity: critical
          team: platform
        annotations:
          summary: "High API error rate on {{ $labels.job }}"
          description: |
            {{ $labels.job }} 5xx error rate is {{ $value | humanizePercentage }}
            (threshold 1%) for 10 minutes.
          runbook_url: "https://runbooks.example.com/api-high-error-rate"
          dashboard_url: "https://grafana.example.com/d/api-overview"

      - alert: APITargetDown
        expr: up{job="api"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "API target {{ $labels.instance }} is down"

      - alert: APIP99LatencyHigh
        expr: |
          histogram_quantile(0.99,
            sum by (job, le) (rate(http_request_duration_seconds_bucket[5m]))
          ) > 0.5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "{{ $labels.job }} p99 latency: {{ $value | humanizeDuration }}"
```

Key fields:

```bash
# alert       — alert name (becomes alertname label)
# expr        — PromQL expression; alert fires when this returns non-empty
# for         — duration the expression must keep returning non-empty before
#               the alert moves from PENDING to FIRING. Defaults to 0s.
# labels      — extra labels added to the alert (drives Alertmanager routing)
# annotations — non-routing metadata supporting templating
# keep_firing_for — (Prom 2.42+) keep firing for this duration even after
#                   expression returns empty (useful for flappy alerts)
```

Alert states:

```bash
# inactive — expression returns empty
# pending  — expression returned non-empty for less than `for` duration
# firing   — expression has returned non-empty for at least `for` duration
```

Inspect:

```bash
# Active alerts in Prometheus
curl -s http://prom:9090/api/v1/alerts | jq

# View loaded alert rules
curl -s http://prom:9090/api/v1/rules?type=alert | jq
```

## Alert Templating

Annotations and labels support Go-template syntax with a Prometheus-specific function set.

```bash
# Variables
{{ $labels.<name> }}               # any label on the firing series
{{ $value }}                       # the metric value at firing time

# Built-in functions (used via | pipe)
{{ $value | humanize }}            # 1234.5 -> "1.234k"
{{ $value | humanize1024 }}        # 1024 -> "1ki"
{{ $value | humanizeDuration }}    # 65 -> "1m 5s"
{{ $value | humanizePercentage }}  # 0.0123 -> "1.23%"
{{ $value | humanizeTimestamp }}
{{ $value | printf "%.2f" }}       # standard printf

# Iteration
{{ range $i, $v := .Alerts }} ... {{ end }}

# Conditionals
{{ if eq $labels.severity "critical" }} ... {{ end }}
```

Canonical examples:

```bash
annotations:
  summary: "High error rate on {{ $labels.instance }}: {{ $value | humanize }} req/sec"
  description: |
    Service {{ $labels.service }} is returning {{ $value | humanizePercentage }}
    5xx responses on {{ $labels.instance }} for the last 10 minutes.
    Recent value: {{ $value | printf "%.4f" }}
```

For multiline annotations, use YAML block scalars:

```bash
annotations:
  description: |
    Multiple
    lines
    here.
```

## Common Patterns

The patterns below are catalog entries — every paste-and-runnable building block you reach for daily.

```bash
# --- Error rate (HTTP) ---
sum by (job) (rate(http_requests_total{status=~"5.."}[5m]))
  /
sum by (job) (rate(http_requests_total[5m]))


# --- p99 latency (HTTP) ---
histogram_quantile(0.99,
  sum by (le, route) (rate(http_request_duration_seconds_bucket[5m]))
)


# --- p50/p90/p99 latency overlays ---
histogram_quantile(0.50, sum by (le) (rate(metric_bucket[5m])))
histogram_quantile(0.90, sum by (le) (rate(metric_bucket[5m])))
histogram_quantile(0.99, sum by (le) (rate(metric_bucket[5m])))


# --- USE method (Utilization, Saturation, Errors) ---
# Utilization (CPU) — fraction of time NOT idle
1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle"}[5m]))

# Saturation — load1 normalized by core count
node_load1
  /
count by (instance) (count by (instance, cpu) (node_cpu_seconds_total))

# Errors — network receive errors
rate(node_network_receive_errs_total[5m])
rate(node_network_transmit_errs_total[5m])


# --- RED method (Rate, Errors, Duration) ---
# Rate
sum by (service) (rate(http_requests_total[5m]))

# Errors
sum by (service) (rate(http_requests_total{status=~"5.."}[5m]))

# Duration (p99)
histogram_quantile(0.99,
  sum by (service, le) (rate(http_request_duration_seconds_bucket[5m]))
)


# --- SLO availability (30-day rolling) ---
1 -
  (
    sum(increase(http_requests_total{status=~"5.."}[30d]))
      /
    sum(increase(http_requests_total[30d]))
  )

# Burn rate — multiplied error budget consumption
(
  sum(rate(http_requests_total{status=~"5.."}[1h]))
    /
  sum(rate(http_requests_total[1h]))
) / 0.001                             # 0.001 = SLO error budget


# --- Capacity headroom (disk fill prediction) ---
predict_linear(node_filesystem_free_bytes{mountpoint="/"}[1h], 4 * 3600) < 0
  and on(instance, mountpoint)
node_filesystem_free_bytes / node_filesystem_size_bytes < 0.2


# --- Memory pressure ---
1 -
  node_memory_MemAvailable_bytes
    /
  node_memory_MemTotal_bytes


# --- Top-N highest CPU pods ---
topk(10,
  sum by (pod) (rate(container_cpu_usage_seconds_total[5m]))
)


# --- Pod restart loops ---
increase(kube_pod_container_status_restarts_total[1h]) > 5


# --- Deployment / version skew ---
count by (cluster) (
  count by (cluster, version) (build_info)
) > 1


# --- Rate joined with version label ---
sum by (instance) (rate(http_requests_total[5m]))
  * on(instance) group_left(version)
build_info
```

## Aggregation Pitfalls

Counter-reset awareness is the single subtlest property of `rate()`. Aggregations performed before `rate()` lose this awareness because individual instances reset at different times.

```bash
# WRONG — sum of counters can decrease when one instance restarts
rate(sum by (job) (http_requests_total)[5m])

# RIGHT — rate each series first, then sum
sum by (job) (rate(http_requests_total[5m]))
```

The general rule: `rate()` (and its kin) must always be the innermost operation on a counter. Apply aggregation outside.

```bash
# WRONG patterns
sum(metric_total) / count(metric_total)               # average rate, no!
delta(sum(http_requests_total)[1h])                   # subject to reset
increase(sum(http_requests_total)[1h])                # subject to reset

# RIGHT
sum(rate(metric_total[5m]))
avg(rate(metric_total[5m]))
sum(increase(http_requests_total[1h]))                 # increase per series, then sum
```

For histograms, the rule is "histogram_quantile of sum-by-le" — never sum the buckets unaggregated, never apply histogram_quantile to a single instance's buckets if you want a global view.

```bash
# RIGHT — global p99 across all instances/routes
histogram_quantile(0.99,
  sum by (le) (rate(http_request_duration_seconds_bucket[5m]))
)

# WRONG — averages the per-instance quantiles, mathematically meaningless
avg(histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m])))
```

You can keep extra labels by adding them to the `by()` clause:

```bash
# Per-route p99
histogram_quantile(0.99,
  sum by (le, route) (rate(http_request_duration_seconds_bucket[5m]))
)

# Per-route, per-cluster p99
histogram_quantile(0.99,
  sum by (le, cluster, route) (rate(http_request_duration_seconds_bucket[5m]))
)
```

## Vector Matching Pitfalls

Binary vector-vector operators require label-set matching. By default both sides must have the EXACT SAME label set or the result is empty.

```bash
# Series A: {job="api", instance="host1"}
# Series B: {job="api", instance="host1", version="1.2"}
# A / B   -> empty (label sets differ)

# Use ignoring(version) to drop version on the right side
A / ignoring(version) B

# Or on(job, instance) to specify the matching labels
A / on(job, instance) B
```

When more than one series on the right matches the same series on the left (or vice versa), the operation fails with "many-to-one" or "one-to-many" errors unless you specify `group_left()` or `group_right()`.

```bash
# Many-to-one: multiple metrics on the left match a single info series on the right
sum by (instance) (rate(http_requests_total[5m]))
  * on(instance) group_left(version)
sum by (instance, version) (build_info)
```

The `group_left(labels)` clause copies the listed labels from the right onto the result; `group_right` is the mirror. The "left/right" in the name refers to which side has the *many*, not which side gives labels.

```bash
# group_left = "left side has the many; copy labels from right"
left_series   * on(matching_labels) group_left(extra_labels)   right_series

# group_right = "right side has the many; copy labels from left"
left_series   * on(matching_labels) group_right(extra_labels)  right_series
```

Common patterns:

```bash
# Annotate a metric with kubernetes pod labels
sum by (pod, namespace) (rate(container_cpu_usage_seconds_total[5m]))
  * on(pod, namespace) group_left(workload)
kube_pod_labels

# Annotate latency with the route label from a separate metric
histogram_quantile(0.99, sum by (le, instance) (rate(metric_bucket[5m])))
  * on(instance) group_left(route)
route_info
```

## Common Errors and Fixes

PromQL errors are sometimes opaque. Each one below quotes the exact text and gives the fix.

```bash
# ERROR: many-to-many matching not allowed: matching labels must be unique on one side
# CAUSE: vector-vector op where labels collide on both sides.
# FIX:  use on()/ignoring() to narrow matching, then group_left()/group_right()
#       so one side disambiguates.
metric_a / on(instance) group_left metric_b


# ERROR: found duplicate series for the match group
# CAUSE: multiple series on the matching side share the same matching labels.
# FIX:  add more labels to on(), or filter the offending side.
# Diagnose:
count by (matching, labels, here) (problem_metric) > 1


# ERROR: vector contains metrics with the same labelset after applying rule labels
# CAUSE: a recording rule's `expr` produced two output series with identical labels
#        (often because the aggregation dropped a label that was distinguishing them).
# FIX:  include the missing label in `by()`, or rename the rule to disambiguate.


# ERROR: query timed out in expression evaluation
# CAUSE: cardinality * range too large; query exceeded --query.timeout (default 2m).
# FIX:  shrink the range, aggregate earlier, precompute via recording rule,
#       or for one-offs raise --query.timeout=5m on the server (operator only).


# ERROR: query result has too many time series
# CAUSE: --query.max-samples or per-query-cardinality limit hit.
# FIX:  add a more selective label matcher, aggregate before returning,
#       or split the query into smaller windows.


# ERROR: expanding series: ...: cannot find rangeable metric in expression
# CAUSE: a function that needs a range vector got an instant vector.
# FIX:  add [<duration>] to the selector.
rate(http_requests_total[5m])           # not rate(http_requests_total)


# ERROR: parse error: unexpected ...
# CAUSE: PromQL syntax error — typically a misplaced operator or a bare metric
#        where a range vector was needed.
# FIX:  paste expression in the Prometheus expression browser to see the
#       caret position. Wrap with parens. Ensure [duration] is on the inside
#       of the rate-style function.


# ERROR: invalid expression type "range vector" for range query
# CAUSE: trying to graph or display a range vector directly.
# FIX:  wrap with rate() / increase() / *_over_time() etc.
rate(http_requests_total[5m])


# ERROR: unknown function with name "..."
# CAUSE: typo, or the function exists in a newer Prometheus than the server.
# Examples: atan2 (2.36+), histogram_avg (2.40+), mad_over_time (2.46+).
# FIX:  upgrade or replace.


# ERROR: 1:7: parse error: unsupported parameter "step" in instant query
# CAUSE: passed step= to /api/v1/query (instant); step is for /api/v1/query_range.
# FIX:  use the correct endpoint.


# Alertmanager-side: alerts firing without expected labels
# CAUSE: labels added in the rule don't appear because the firing series
#        already had a label of the same name (rule labels can override but
#        do not merge).
# FIX:  pick non-conflicting label names, or aggregate to drop the original.
```

## Common Gotchas

```bash
# bad: rate without a range
rate(counter)
# fix:
rate(counter[5m])


# bad: aggregator clause outside parens, ambiguous syntax
sum(rate(metric[5m])) by (instance) > 100
# fix: explicit
(sum by (instance) (rate(metric[5m]))) > 100


# bad: histogram_quantile without sum-by-le
histogram_quantile(0.99, rate(metric_bucket[5m]))
# fix:
histogram_quantile(0.99, sum by (le) (rate(metric_bucket[5m])))


# bad: alert with no `for:` fires on transient spikes
- alert: HighX
  expr: rate(metric[5m]) > 100
# fix: add `for:`
- alert: HighX
  expr: rate(metric[5m]) > 100
  for: 5m


# bad: rate range too short for scrape interval
rate(metric[1m])                # at 30s scrape, only 2 samples
# fix:
rate(metric[5m])                # 4× scrape minimum, 10× scrape recommended


# bad: counter-reset broken by manual subtraction
http_requests_total - http_requests_total offset 5m
# fix: use rate() or increase(), they handle resets
increase(http_requests_total[5m])


# bad: irate in production alerts
- alert: X
  expr: irate(metric[5m]) > 100
# fix: use rate
- alert: X
  expr: rate(metric[5m]) > 100
  for: 5m


# bad: avg() across instances of a histogram_quantile()
avg(histogram_quantile(0.99, rate(metric_bucket[5m])))
# fix: histogram_quantile of sum
histogram_quantile(0.99, sum by (le) (rate(metric_bucket[5m])))


# bad: regex matcher when literal would do (slower)
metric{path=~"/health"}
# fix:
metric{path="/health"}


# bad: unbounded selector
{__name__=~".+"}
# fix: always include a job or instance matcher
{__name__=~".+", job="api"}


# bad: missing label on annotation template — silently empty
summary: "{{ $labels.misspelled }} is high"
# fix: typo-check via promtool
promtool check rules rules.yml


# bad: alert that fires on transient missing data
- alert: X
  expr: up == 0
# fix: require persistence
- alert: X
  expr: up == 0
  for: 2m


# bad: dividing by zero produces +Inf/NaN that pollutes graphs
errors / requests
# fix: clamp_min the denominator
errors / clamp_min(requests, 1)


# bad: range too long for short-lived counters (process restart erases history)
rate(http_requests_total[1d])
# fix: keep windows under typical uptime, prefer increase() with a recording rule
```

## Performance — Query Cost

PromQL evaluation cost scales linearly with series cardinality, with samples evaluated, and with subquery resolution steps. Three rules of thumb:

```bash
# 1) Cardinality kills.
#    A high-cardinality label (userId, fullPath, error_message) can balloon
#    metrics into millions of series. Each query touches every matching series.

# 2) Range × scrape_rate = samples evaluated.
#    rate(metric[1h]) on a 5s scrape touches 720 samples per series.
#    Multiply by series count, then by step count for a range query.

# 3) Subqueries multiply.
#    expr[1h:30s] runs `expr` 120 times. If `expr` was already heavy,
#    the subquery is heavier still.
```

Operational diagnostics:

```bash
# Total active series in this Prometheus
prometheus_tsdb_head_series

# Samples ingested per second
rate(prometheus_tsdb_head_samples_appended_total[5m])

# Series per metric — find the offenders
topk(20, count by (__name__) ({__name__=~".+"}))

# Slowest queries
topk(10, prometheus_engine_query_duration_seconds{slice="inner_eval"})
```

The HTTP API exposes timing details via the `stats` parameter:

```bash
curl -G \
  --data-urlencode 'query=rate(http_requests_total[5m])' \
  --data-urlencode 'stats=true' \
  http://prom:9090/api/v1/query | jq
```

For range queries, `step` controls evaluation density. Big windows with tiny steps are expensive:

```bash
# Cheap — 60 evaluations
curl -G \
  --data-urlencode 'query=rate(metric[5m])' \
  --data-urlencode 'start=2026-04-25T11:00:00Z' \
  --data-urlencode 'end=2026-04-25T12:00:00Z' \
  --data-urlencode 'step=60s' \
  http://prom:9090/api/v1/query_range

# Expensive — 3600 evaluations
curl -G \
  --data-urlencode 'query=rate(metric[5m])' \
  --data-urlencode 'start=2026-04-25T11:00:00Z' \
  --data-urlencode 'end=2026-04-25T12:00:00Z' \
  --data-urlencode 'step=1s' \
  http://prom:9090/api/v1/query_range
```

Recording rules pay the evaluation cost once per interval rather than per query, which is the single highest-leverage optimization.

```bash
# Bad: dashboard query, runs every refresh, every panel
sum by (job) (rate(http_requests_total[5m]))

# Good: recorded once every 30s, cheap thereafter
job:http_requests:rate5m
```

## Cardinality and Best Practices

Cardinality control is the single most important operational consideration in any Prometheus deployment.

```bash
# Avoid as labels:
# - userId, sessionId, requestId
# - fullPath (tokens with IDs in them)
# - exact error message text
# - timestamp (already a coordinate)
# - email addresses

# Prefer as labels:
# - environment (prod, staging, dev)
# - job, instance, region
# - status code class (2xx/4xx/5xx)
# - route pattern (/users/:id, not /users/12345)
# - error code or category, not free-form
```

Find cardinality offenders:

```bash
# Top 20 metric names by series count
topk(20, count by (__name__) ({__name__=~".+"}))

# Within one metric, top labels by distinct value count
count(count by (label_name) (some_metric))

# Active label values endpoint
curl -G --data-urlencode 'match[]=some_metric' \
  http://prom:9090/api/v1/label/some_label/values | jq '.data | length'
```

Histogram bucket selection:

```bash
# Default HTTP latency buckets (suit most APIs)
[0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]

# Each bucket multiplies series count of the histogram by 1
# 11 buckets * 100 instances * 5 routes = 5500 series for that metric alone
```

Native histograms (Prom 2.40+) replace bucket explosion with one series — adopt where possible:

```bash
# Enable native histogram parsing in Prometheus
prometheus --enable-feature=native-histograms
```

## Federation and Remote Write

Federation pulls a curated subset of metrics from one Prometheus into another. The `/federate` endpoint takes a `match[]` selector and returns a snapshot in exposition format.

```bash
# Fetch all up{} and rate-summarized metrics from a downstream Prom
curl -G \
  --data-urlencode 'match[]={__name__=~"job:.*"}' \
  --data-urlencode 'match[]={__name__="up"}' \
  http://downstream-prom:9090/federate

# In a global Prometheus's prometheus.yml:
scrape_configs:
  - job_name: 'federate'
    scrape_interval: 30s
    honor_labels: true
    metrics_path: '/federate'
    params:
      'match[]':
        - '{__name__=~"job:.*"}'
        - '{__name__="up"}'
    static_configs:
      - targets:
          - 'region-a-prom:9090'
          - 'region-b-prom:9090'
```

Federation is intended for "regional → global" aggregation, NOT for long-term storage and NOT for full-fidelity replication. It scales poorly past a few thousand series.

For long-term storage and global aggregation, use `remote_write` to a Mimir/Thanos/Cortex/VictoriaMetrics receiver:

```bash
# prometheus.yml
remote_write:
  - url: "https://mimir.example.com/api/v1/push"
    headers:
      X-Scope-OrgID: "tenant1"
    basic_auth:
      username: "$USER"
      password_file: "/etc/prom/mimir.pass"
    queue_config:
      capacity: 5000
      max_shards: 50
      max_samples_per_send: 1000
    write_relabel_configs:
      - source_labels: [__name__]
        regex: 'go_.*'
        action: drop
```

For long-term reads, configure `remote_read`:

```bash
remote_read:
  - url: "https://mimir.example.com/prometheus/api/v1/read"
    headers:
      X-Scope-OrgID: "tenant1"
```

The general decision tree:

```bash
# Small (< 10M active series) — local Prometheus suffices, federation for global view.
# Medium (10M-100M) — remote_write to single Mimir/Thanos/VictoriaMetrics cluster.
# Large (> 100M)    — remote_write to multi-tenant horizontally-scaled backend,
#                     downsample old data, careful retention policies.
```

## Idioms

A short tour of operational habits that separate seasoned PromQL from noviceware.

```bash
# Always rate before aggregating.
sum by (job) (rate(metric[5m]))                  # right
rate(sum by (job) (metric)[5m])                  # wrong


# Always histogram_quantile of sum-by-le.
histogram_quantile(0.99, sum by (le) (rate(metric_bucket[5m])))


# Always add `for:` to alerts (≥ 2× scrape interval).
- alert: X
  expr: ...
  for: 5m


# Always use `_total` suffix on counters; rate(_total) is your hint.
rate(http_requests_total[5m])


# Recording-rule naming: <level>:<metric>:<operations>
# instance:cpu_usage:rate5m
# job:http_requests:rate5m
# cluster:slo_error_budget:30d


# In Grafana, prefer $__rate_interval over hard-coded ranges.
rate(metric[$__rate_interval])


# Reload Prometheus config without restart.
curl -X POST http://prom:9090/-/reload


# Validate config and rules before deploy.
promtool check config prometheus.yml
promtool check rules /etc/prometheus/rules/*.yml
promtool test rules tests/*.yml


# Use the up metric as the canonical liveness signal.
up{job="api"} == 0
sum by (job) (up)
count by (job) (up == 0)


# Wrap everything in count(...) when debugging cardinality.
count(metric)
count by (label) (metric)


# Pin a value with vector() for arithmetic.
metric > vector(100)


# Drop noisy labels using sum without().
sum without (instance) (rate(metric[5m]))


# Add labels using label_replace().
label_replace(metric, "team", "platform", "", "")
```

## Tips

```bash
# Exposition format quick reference (text/plain; version=0.0.4)
# Lines starting with # HELP describe metrics
# Lines starting with # TYPE declare type
# Sample lines are: <metric>{<labels>} <value> [<timestamp>]
# HELP http_requests_total Number of HTTP requests
# TYPE http_requests_total counter
http_requests_total{method="GET",status="200"} 1027 1714046400000


# Metric naming conventions
# Counters end _total                               (http_requests_total)
# Durations end _seconds                            (http_request_duration_seconds)
# Sizes end _bytes                                  (process_resident_memory_bytes)
# Ratios end _ratio                                 (cpu_usage_ratio)
# Info-only metrics end _info, value always 1       (build_info{version="1.2.3"})


# Label naming conventions
# lowercase_with_underscores
# avoid prefix double-underscores (reserved)
# meta-labels are __<lowercase>__ (e.g., __name__)


# /metrics endpoint expectation
# Reachable via HTTP GET, no auth (or bearer token if secured)
# Idempotent — Prometheus scrapes every scrape_interval


# Useful API endpoints
http://prom:9090/api/v1/query
http://prom:9090/api/v1/query_range
http://prom:9090/api/v1/series
http://prom:9090/api/v1/labels
http://prom:9090/api/v1/label/<name>/values
http://prom:9090/api/v1/targets
http://prom:9090/api/v1/rules
http://prom:9090/api/v1/alerts
http://prom:9090/api/v1/status/config
http://prom:9090/api/v1/status/runtimeinfo
http://prom:9090/api/v1/status/buildinfo
http://prom:9090/api/v1/status/tsdb
http://prom:9090/-/reload                       # POST to reload config
http://prom:9090/-/healthy
http://prom:9090/-/ready


# Useful CLI tools
prometheus --config.file=prometheus.yml
promtool check config prometheus.yml
promtool check rules rules.yml
promtool test rules tests/*.yml
promtool query instant http://localhost:9090 'up'
promtool query range  http://localhost:9090 'rate(metric[5m])' \
  --start=2026-04-25T11:00:00Z --end=2026-04-25T12:00:00Z --step=30s
promtool tsdb analyze /var/lib/prometheus
promtool tsdb list /var/lib/prometheus


# Version-specific feature notes
# trig functions sin/cos/tan and friends ........... Prometheus 2.26+
# atan2 binary operator ............................ Prometheus 2.36+
# native histograms (experimental) ................. Prometheus 2.40+
# absent_over_time() ............................... Prometheus 2.30+
# present_over_time() .............................. Prometheus 2.29+
# last_over_time() ................................. Prometheus 2.26+
# mad_over_time() .................................. Prometheus 2.46+
# histogram_avg() / histogram_stddev() / native ..... Prometheus 2.40+
# keep_firing_for ................................... Prometheus 2.42+
# @ modifier ........................................ Prometheus 2.25+ (--enable-feature=promql-at-modifier prior to GA)


# Default ports
# Prometheus       9090
# node_exporter    9100
# alertmanager     9093
# pushgateway      9091
# blackbox_exp     9115


# Time-zone caveat
# Date/time functions are UTC. Convert in Grafana panel options
# or in the query with manual offsets if you need local time.


# Counters never decrease (except on reset).
# A "negative rate" indicates either a counter reset (handled by rate())
# or a metric mistakenly typed as counter when it's a gauge.


# The `up` metric is auto-created — you never expose it manually.
# Its value is 1 if the most recent scrape succeeded, 0 otherwise.


# `scrape_duration_seconds`, `scrape_samples_scraped`,
# `scrape_samples_post_metric_relabeling`, `scrape_series_added`
# are also auto-created per scrape.
```

## See Also

- prometheus
- grafana
- alertmanager
- loki
- opentelemetry
- mimir

## References

- Prometheus query basics: https://prometheus.io/docs/prometheus/latest/querying/basics/
- Prometheus query operators: https://prometheus.io/docs/prometheus/latest/querying/operators/
- Prometheus query functions: https://prometheus.io/docs/prometheus/latest/querying/functions/
- Prometheus query examples: https://prometheus.io/docs/prometheus/latest/querying/examples/
- Recording rules: https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/
- Alerting rules: https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/
- Best practices: https://prometheus.io/docs/practices/
- Naming conventions: https://prometheus.io/docs/practices/naming/
- Histograms and summaries: https://prometheus.io/docs/practices/histograms/
- Federation: https://prometheus.io/docs/prometheus/latest/federation/
- HTTP API: https://prometheus.io/docs/prometheus/latest/querying/api/
- Native histograms: https://prometheus.io/docs/specs/native_histograms/
- promtool: https://prometheus.io/docs/prometheus/latest/command-line/promtool/
- "Prometheus: Up & Running" — Brian Brazil and Julien Pluquet (O'Reilly)
- "Cloud Native Observability with Prometheus" — Rob Skillington (Manning)
