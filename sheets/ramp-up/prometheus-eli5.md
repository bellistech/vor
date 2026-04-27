# Prometheus — ELI5

> Prometheus is a tireless doctor who walks the hospital floor every fifteen seconds, takes everyone's vitals, writes them on a chart with timestamps, and lets you ask questions about the chart later.

## Prerequisites

You should have read [linux-kernel-eli5](#) first so the words "process," "port," "file," and "binary" feel friendly. You do not need to know any "monitoring." You do not need to know what a "time series database" is. You do not need to know SQL. By the end of this sheet you will know what Prometheus is, what it scrapes, why it scrapes, what a metric is, what a label is, what PromQL is, why your alerts are firing at 3am, and what to do when somebody says "we have a cardinality problem."

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## Plain English

### What even is Prometheus?

Imagine a hospital. A really big hospital. Hundreds of patients in hundreds of rooms. Each patient has a heart rate, a blood pressure, a temperature, a breathing rate, a glucose level, an oxygen level. You are the head doctor. You want to know if any patient is getting sick. You want to know if any patient is about to die. You want to know if any patient had a fever last Tuesday between 2pm and 3pm so you can figure out what happened.

If every patient just shouted their numbers at you all day long, you would go deaf. If you had to walk to every patient and ask them every minute, you would never stop walking. You would never sleep. You would never eat lunch.

So you hire a robot. The robot has wheels. The robot has a clipboard. Every fifteen seconds the robot rolls down the hallway, peeks into each patient's room, reads the little display next to the bed (the patient already wrote their numbers on it), writes those numbers on the clipboard, and rolls back to the office. The robot does this forever. The robot never gets tired. The robot keeps every clipboard, with the timestamp, in a giant filing cabinet. Later, when you want to know "did patient #42 have a fever last Tuesday?", you ask the robot, and the robot pulls up the right clipboard and says "yes, between 2:14pm and 2:48pm, their temperature was 39.2 degrees."

**That robot is Prometheus.**

The patients are your computers, your applications, your databases, your routers. The "little display next to the bed" is a special web page each program publishes called `/metrics` that has all its numbers on it. The robot rolling down the hallway is Prometheus, scraping each `/metrics` page on a schedule. The filing cabinet is the time series database (the "TSDB"). The questions you ask later are written in a query language called PromQL.

That is the whole product. Everything in this sheet is just details on top of that picture. If the picture goes fuzzy, come back here and remember: doctor, robot, clipboard, filing cabinet, questions later.

### A different picture: the meter reader

Here is another way to think about it. Imagine the old days when somebody from the electric company would walk house to house once a month, read the dial on the meter on the side of your house, write down the number on a clipboard, and walk away. They didn't ring the doorbell. They didn't ask permission. They didn't make the meter do anything. The meter was already running. They just looked at it and wrote down the number.

That is exactly what Prometheus does. Your application has a meter (the `/metrics` page). The meter is always running. Prometheus is the meter reader. It walks up, looks at the numbers, writes them down with a timestamp, and walks away. It never tells your application "go faster" or "go slower." It just observes.

The meter reader picture is best for understanding **the pull model.** Prometheus pulls. Your application does not push.

### A third picture: the weather station

Imagine a weather station on a hill. The weather station has a thermometer, a barometer, a wind speed gauge, a rain gauge, a humidity sensor. Every five minutes, somebody at the office calls the weather station on the radio and says "what are your readings?" The weather station replies "temperature 18, pressure 1013, wind 12 km/h east, rain 0, humidity 67." The office writes those numbers on a long roll of paper with a timestamp and pins it to the wall. After a year, the paper is huge and you can see exactly what the weather did every five minutes for an entire year.

That is also Prometheus. The weather station is your application. The radio call is the scrape. The long roll of paper is the TSDB. After a year you can scroll back and ask "what was the temperature on the morning of June 4th at 7am?" and find the answer.

### Why so many pictures?

Because nobody can see Prometheus. It is just numbers in a database. Different pictures help with different ideas.

The **doctor and robot** picture is best for understanding that Prometheus collects vitals from many things on a schedule.

The **meter reader** picture is best for understanding pull versus push.

The **weather station** picture is best for understanding the long roll of timestamped data and why you can ask historical questions.

If one picture is not clicking, switch to another. Whichever feels right is the one you should keep in your head.

## The Pull Model

This is the single most important thing about Prometheus, and it is the thing that confuses people coming from other monitoring systems. Prometheus **pulls.** Your application does not **push.**

### What does "pull" mean?

Pull means Prometheus reaches out and asks. Every fifteen seconds (or whatever interval you set), Prometheus opens a connection to each target and says "GET /metrics please." The target replies with a big text file full of numbers. Prometheus parses the text, stores the numbers, and closes the connection.

The target does not know when the next scrape is coming. The target does not pick the interval. The target does not decide where the data goes. The target just sits there with a `/metrics` endpoint open, and Prometheus shows up whenever it feels like it.

### What does "push" mean?

Push is the opposite. In a push system, your application phones home. It opens a connection to a central server and says "here are my numbers, please store them." Graphite works this way. StatsD works this way. InfluxDB can work either way. CloudWatch works this way. Datadog (mostly) works this way.

In push systems, the application is the active party. It picks when to send. It picks what to send. It picks where to send. The central server is passive — it just sits there and listens.

### Pull versus push, side by side

```
PULL (Prometheus)                    PUSH (Graphite, StatsD)
─────────────────                    ───────────────────────
                                      ┌──────────────┐
   ┌──────────┐                       │  app A       │─┐
   │  app A   │ ◄──── /metrics        │              │ │ stats UDP
   │ :metrics │                       └──────────────┘ │
   └──────────┘                                        │
                                      ┌──────────────┐ │
   ┌──────────┐                       │  app B       │─┤
   │  app B   │ ◄──── /metrics        │              │ │
   │ :metrics │                       └──────────────┘ ▼
   └──────────┘                                     ┌─────────┐
       ▲                                            │ StatsD/ │
       │ scrape every 15s                           │Graphite │
   ┌───┴───────┐                                    └─────────┘
   │Prometheus │
   │  (scraper)│
   └───────────┘
```

In pull, the arrow points away from Prometheus. Prometheus does the asking.
In push, the arrows point toward the server. The apps do the sending.

### Why pull? Why does Prometheus do it this way?

Because pull is great for finding out when things are sick. Push systems have a fundamental blind spot: if a target dies, it stops pushing, and you might never know it died. You just stop seeing data from it. You have to write a separate "is the target alive?" check, and now you have two systems.

Pull bakes "is the target alive?" into the scrape itself. If Prometheus tries to scrape a target and the target doesn't answer, Prometheus knows immediately. It records `up = 0` for that target. You can alert on `up == 0` and you will get paged when anything dies.

Pull also means Prometheus controls the rate. No application can flood Prometheus by misbehaving and pushing too much. Prometheus says "I'll come ask you every fifteen seconds, no more, no less." If your application generates a million metrics, Prometheus is going to try to download a million metrics every fifteen seconds, and you'll see your scrape duration go up, but the *rate* is fixed. There is no thundering herd of pushed metrics swamping the receiver.

Pull also means service discovery is centralized. Prometheus knows about all the targets. No application needs to know "where is the metrics server?" because no application is sending anywhere. Each application just exposes `/metrics`.

### When pull is bad

Pull has a real weakness: short-lived jobs. Imagine a cron job that runs for five seconds every hour. Prometheus scrapes every fifteen seconds. The cron job will be done before Prometheus ever shows up. You will never see its metrics.

For that case, Prometheus has an escape hatch called the **Pushgateway.** The cron job pushes its metrics to the Pushgateway just before exiting. The Pushgateway holds those metrics in memory. Prometheus scrapes the Pushgateway. The numbers survive. We will talk more about Pushgateway later, including a long warning about when **not** to use it.

### Pull means Prometheus must reach the target

This is where networking and firewalls bite you. If Prometheus is in one network and your application is in another network, Prometheus has to be able to open a TCP connection to your application's metrics port. If a firewall blocks that, no data. If the application is behind NAT and not reachable, no data. If the application is on the user's laptop and the laptop sleeps, no data.

For the cases where Prometheus can't reach back, Pushgateway and remote-write systems exist. But for the normal case — servers in your own data center or your own Kubernetes cluster — pull works great because Prometheus can reach everything.

## Time Series Data Model

Prometheus stores **time series.** This is a fancy word for "a number that changes over time, with a label saying what the number means."

### The shape of a single time series

A time series is two things:

1. **A name plus labels** that uniquely identify the series.
2. **A long list of (timestamp, value) pairs** — the samples.

For example, here is one time series:

```
http_requests_total{method="GET", status="200", path="/api/users"}
```

That whole thing on the left is the **identity** of the series. The metric name is `http_requests_total`. The labels are `method=GET`, `status=200`, and `path=/api/users`. Together that combination is one specific series.

The samples for that series might look like this:

```
timestamp                value
1714190400000  ──────►   42091
1714190415000  ──────►   42103
1714190430000  ──────►   42117
1714190445000  ──────►   42131
1714190460000  ──────►   42145
...
```

Every fifteen seconds (because that's the scrape interval) Prometheus appends a new (timestamp, value) pair to the end of this list. Forever. Until you delete it.

Now imagine you also have:

```
http_requests_total{method="GET", status="200", path="/api/orders"}
http_requests_total{method="GET", status="404", path="/api/users"}
http_requests_total{method="POST", status="200", path="/api/users"}
http_requests_total{method="POST", status="500", path="/api/orders"}
```

Each of those is a **completely separate time series.** They all share the same metric name (`http_requests_total`) but they have different labels, so they are different series.

### Metric name + labels = unique series

This is the key sentence. Read it slowly:

> A unique combination of metric name plus labels is one time series.

If you change a single label value — even one character — you have a different series. `path="/api/users"` and `path="/api/users/"` (note the trailing slash) are two different series. `status="200"` and `status="2xx"` are two different series. `pod="pod-abc123"` and `pod="pod-abc124"` are two different series.

This sounds simple but it is the source of about half the trouble people have with Prometheus. Every unique combination of labels creates a new series. Every new series uses memory. If you put a high-cardinality label in there (like a user ID, or a request ID, or a timestamp), you can blow up Prometheus by creating millions or billions of series. We will spend a lot of time on this later.

### What is a sample?

A sample is one (timestamp, value) pair. Just two numbers. The timestamp is milliseconds since 1970 (the Unix epoch). The value is a 64-bit float.

```
sample = (timestamp_ms, float64_value)
```

That's it. A sample has no string in it. The labels live on the series, not on the sample. The sample is just two numbers. Prometheus stores billions of samples in a very compressed format.

### A picture: metric → series → samples

```
metric name: http_requests_total
   │
   ├── series 1: {method=GET, status=200, path=/users}
   │       samples: (t1,42), (t2,43), (t3,44), (t4,45), (t5,46), ...
   │
   ├── series 2: {method=GET, status=404, path=/users}
   │       samples: (t1, 0), (t2, 0), (t3, 1), (t4, 1), (t5, 2), ...
   │
   ├── series 3: {method=POST, status=200, path=/orders}
   │       samples: (t1,18), (t2,19), (t3,19), (t4,20), (t5,21), ...
   │
   └── series 4: {method=POST, status=500, path=/orders}
           samples: (t1, 0), (t2, 0), (t3, 0), (t4, 1), (t5, 1), ...
```

One metric name. Four series. Each series has its own list of samples.

### Why store data this way?

Because timestamped numbers compress amazingly well when you store them in column form. The Prometheus TSDB (the storage engine) uses a trick called **delta-of-delta encoding** for timestamps and **XOR encoding** for values. After compression, the average sample takes about **1.3 bytes on disk.** That is the size of a single character. You can store billions of samples in a few gigabytes.

The whole storage engine was rewritten for Prometheus 2.0 (released November 2017) to get this kind of efficiency. Before 2.0, Prometheus had a much slower storage engine that struggled past a few hundred thousand series. After 2.0, a single Prometheus server can comfortably handle ten million active series.

## Metric Types

Prometheus has four official metric types. They are not really enforced by the wire format — under the hood, every sample is just a float — but the types tell you (and PromQL) what the number means and what kinds of operations make sense on it.

### Counter

A counter is a number that **only ever goes up.** Or stays the same. Or, in one specific case, gets reset to zero (when the process restarts).

Examples:

- `http_requests_total` — the total number of HTTP requests handled since the process started.
- `bytes_sent_total` — the total bytes sent over the network since boot.
- `errors_total` — the total errors encountered since the process started.

Counters are great because they tell the truth even if you miss scrapes. If you scrape at t=0 and see 100, and you scrape at t=60 and see 250, you know there were 150 requests in that minute, regardless of what happened in between.

You almost never look at the raw value of a counter. You look at its **rate** — how fast it's going up — using `rate()` or `increase()`. We'll get there.

By convention, counters end in `_total`. (Not strictly enforced, but the convention is strong.)

### Gauge

A gauge is a number that **can go up or down.**

Examples:

- `memory_used_bytes` — how much memory the process is using right now.
- `cpu_temperature_celsius` — how hot the CPU is right now.
- `queue_length` — how many items are in a queue right now.
- `active_connections` — how many TCP connections are open right now.

Gauges are like the temperature reading on a thermometer. They tell you the value at the moment of the scrape. If you miss a scrape, you miss what was happening at that moment, and you can't reconstruct it.

You usually look at gauges directly, or use `avg_over_time()`, `min_over_time()`, `max_over_time()` to summarize them.

### Histogram

A histogram is a way to measure **distributions.** Like response times. You don't just want the average response time — you want to know "what fraction of requests took less than 100ms? What fraction took more than 1 second?"

A histogram counts how many observations fell into each of a set of **buckets.** You define the buckets up front (for example, "less than 5ms, less than 10ms, less than 25ms, less than 50ms, less than 100ms, less than 250ms, less than 500ms, less than 1s, less than 2.5s, less than 5s, less than 10s, +Inf").

Each bucket is its own counter. So a histogram is actually **N+3 series** under one logical metric:

- `http_request_duration_seconds_bucket{le="0.005"}` — count of requests faster than 5ms (cumulative)
- `http_request_duration_seconds_bucket{le="0.01"}` — count faster than 10ms
- `http_request_duration_seconds_bucket{le="0.025"}` — count faster than 25ms
- ... one for each bucket boundary ...
- `http_request_duration_seconds_bucket{le="+Inf"}` — count of all requests
- `http_request_duration_seconds_sum` — sum of all observed durations
- `http_request_duration_seconds_count` — total number of observations (same as the +Inf bucket)

Note the `le` label. `le` means "less than or equal to." Buckets are **cumulative.** The bucket `le="0.1"` includes everything up to 100ms — including all the requests that landed in `le="0.005"`, `le="0.01"`, `le="0.025"`, and `le="0.05"`. Cumulative buckets are weird but they make a particular query (`histogram_quantile()`) easy to compute.

**Histogram bucket layout (cumulative):**

```
observation: 73ms

le=0.005    [           ]   ← did not reach this bucket (≤5ms)
le=0.01     [           ]   ← did not reach this bucket (≤10ms)
le=0.025    [           ]   ← did not reach this bucket (≤25ms)
le=0.05     [           ]   ← did not reach this bucket (≤50ms)
le=0.1      [   +1      ]   ← falls in this bucket and all larger
le=0.25     [   +1      ]   ← also incremented
le=0.5      [   +1      ]
le=1.0      [   +1      ]
le=2.5      [   +1      ]
le=5.0      [   +1      ]
le=+Inf     [   +1      ]   ← always incremented (catches everything)
```

So a single observation increments **every bucket whose boundary it crosses or doesn't reach.** That is the cumulative property.

You compute a quantile (like the 99th percentile) by looking at all the buckets and figuring out which bucket the 99th-percentile observation falls in. PromQL has a function for this: `histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))`. We will look at it later.

### Summary

A summary is the older sibling of histogram. Same idea (measure a distribution) but the math happens at the source instead of in PromQL.

A summary exposes:

- `http_request_duration_seconds{quantile="0.5"}` — pre-computed median
- `http_request_duration_seconds{quantile="0.9"}` — pre-computed 90th percentile
- `http_request_duration_seconds{quantile="0.99"}` — pre-computed 99th percentile
- `http_request_duration_seconds_sum` — sum of observations
- `http_request_duration_seconds_count` — count of observations

The summary computes the quantiles inside the application using a streaming algorithm. The advantage: the percentiles are exact. The disadvantage: you cannot **aggregate** them. If you have ten servers each exposing a `quantile=0.99`, you cannot average those ten numbers to get "the global 99th percentile across all ten servers." Averaging quantiles is mathematically wrong. Histograms can be aggregated; summaries cannot.

This is why histogram has won the popularity contest. Histograms aggregate. Summaries don't.

### Native histograms (Prometheus 2.40 preview, 3.0 GA)

Prometheus 2.40 added a preview of **native histograms** — a much smarter histogram with exponentially-spaced buckets that doesn't require you to pick boundaries up front. Prometheus 3.0 (released November 2024) made native histograms generally available. They use far less storage and produce much better quantile estimates. They are the future. If you are starting today, they are worth learning about, but classic histograms are still the most common in production for now.

## Labels (and the Cardinality Explosion)

Labels are the most powerful and most dangerous feature in Prometheus.

### What is a label?

A label is a key-value pair attached to a metric to slice it different ways. Like:

```
http_requests_total{method="GET", status="200", path="/users"}
http_requests_total{method="POST", status="500", path="/orders"}
```

`method`, `status`, `path` are label names. `"GET"`, `"POST"`, `"200"`, `"500"`, `"/users"`, `"/orders"` are label values.

You can put almost anything in labels: the HTTP method, the status code, the route, the instance hostname, the data center, the customer tier, the version, the build hash, anything that helps you slice and dice.

### What is cardinality?

**Cardinality** is the number of unique combinations of label values.

If you have a metric `http_requests_total{method, status, path}` and:

- `method` can be one of: GET, POST, PUT, DELETE, PATCH (5 values)
- `status` can be one of: 200, 301, 400, 401, 403, 404, 500, 502, 503 (9 values)
- `path` can be one of: /users, /orders, /products, /search (4 values)

Then the cardinality is at most `5 × 9 × 4 = 180` series. That's totally fine. Prometheus can handle a hundred and eighty series in its sleep.

### The cardinality explosion

Now imagine you decide to add `user_id` as a label. Why not, right? It would be useful to slice requests by user.

If you have a million users, your cardinality just became `5 × 9 × 4 × 1,000,000 = 180,000,000`.

That is **one hundred and eighty million series.** Each active series in Prometheus uses about 3 KB of memory just for the index, plus storage for samples. 180 million series × 3 KB = 540 GB of RAM **for the index alone.** Your Prometheus server now needs more memory than your laptop has. It will crash. Your monitoring is gone. You will get paged at 3am because the alerting also crashed.

This is the **cardinality explosion**, and it is the single most common way to wreck a Prometheus deployment. The rule of thumb is: never put high-cardinality data in labels. No user IDs. No request IDs. No email addresses. No IP addresses (well, mostly — internal node IPs are fine; user IPs are not). No URLs with parameters in them. No timestamps as label values. No UUIDs. No hashes. Nothing that has more than a few hundred unique values.

### What goes in labels, what doesn't

Good labels (low cardinality):

- `method` — handful of HTTP methods
- `status_code` — handful of HTTP statuses
- `route_template` — like `/users/:id`, not `/users/12345`
- `service` — handful of microservices
- `instance` — your hosts, fine if you have hundreds, dangerous if millions
- `job` — the Prometheus job name
- `data_center` or `region` — handful
- `version` or `build` — handful at any one time

Bad labels (high cardinality):

- `user_id` — millions
- `request_id` — every single request
- `email` — every user
- `url` (with query parameters) — every unique query
- `customer_uuid` — every customer
- `trace_id` — every trace
- `pod_name` (in Kubernetes with rolling deploys) — every pod ever
- `timestamp` — uncountable

If you find yourself wanting to put one of the bad ones in a label, the answer is almost always **logs** or **traces**, not metrics. Prometheus is for aggregate behavior. For per-request data, you want a logging system (Loki, Elasticsearch) or a tracing system (Jaeger, Tempo). Don't try to make Prometheus do that job.

### How do I see my cardinality?

Use the `promtool` command:

```bash
$ promtool tsdb analyze /var/lib/prometheus/data
```

It prints which metrics and which labels are using the most series. The biggest offenders go at the top. If you see one metric with a million series, that is your problem. If you see one label with a million unique values, that is your problem.

You can also query Prometheus itself:

```
topk(10, count by (__name__)({__name__=~".+"}))
```

This finds the ten metric names with the most series. That'll usually show you what's eating your memory.

## The Scrape

Let's walk through one scrape from start to finish, slowly, so you can see exactly what happens.

### What is a scrape?

A scrape is one HTTP GET request from Prometheus to a target's `/metrics` endpoint, plus the parsing of the response.

```
Prometheus ──── GET /metrics HTTP/1.1 ────► target:9100
target     ──── 200 OK + body ────► Prometheus
```

The body of the response is text in a specific format called the **text exposition format** (or, in newer versions, the OpenMetrics format, which is a slightly stricter superset).

### What does /metrics look like?

It looks like this:

```
# HELP node_cpu_seconds_total Seconds the CPUs spent in each mode.
# TYPE node_cpu_seconds_total counter
node_cpu_seconds_total{cpu="0",mode="idle"} 1234.56
node_cpu_seconds_total{cpu="0",mode="user"} 78.91
node_cpu_seconds_total{cpu="0",mode="system"} 12.34
node_cpu_seconds_total{cpu="1",mode="idle"} 1235.67
node_cpu_seconds_total{cpu="1",mode="user"} 79.01
node_cpu_seconds_total{cpu="1",mode="system"} 12.45

# HELP node_memory_MemFree_bytes Memory information field MemFree_bytes.
# TYPE node_memory_MemFree_bytes gauge
node_memory_MemFree_bytes 1.073741824e+09

# HELP node_filesystem_avail_bytes Filesystem space available to non-root users in bytes.
# TYPE node_filesystem_avail_bytes gauge
node_filesystem_avail_bytes{device="/dev/sda1",fstype="ext4",mountpoint="/"} 5.4e+10
```

Every line is one of three things:

- `# HELP metric_name Some description.` — human-readable description.
- `# TYPE metric_name (counter|gauge|histogram|summary|untyped)` — the metric type.
- A sample line: `metric_name{label="value", ...} value [timestamp_optional]`

The format is plain text. You can `curl` it. You can `grep` it. You can read it. That is on purpose. Prometheus designers wanted exposition to be debuggable from a terminal.

### A scrape timeline

Imagine the scrape interval is 15 seconds. Here is what happens at each tick:

```
t=0    Prometheus connects to target:9100 → GET /metrics → 200 OK
       parses 5,000 lines, stores 5,000 samples with timestamp t=0
       closes connection
t=15   Prometheus connects to target:9100 → GET /metrics → 200 OK
       parses 5,000 lines, stores 5,000 samples with timestamp t=15
       closes connection
t=30   Prometheus connects to target:9100 → GET /metrics → 200 OK
       parses 5,000 lines, stores 5,000 samples with timestamp t=30
       closes connection
...
```

This goes on forever. Each scrape is independent. Each scrape produces one sample per series. Each scrape is timestamped at the moment Prometheus *started* the scrape.

### What if the target is down?

If the target doesn't respond, or the connection times out, Prometheus records:

```
up{instance="target:9100", job="node"} = 0
```

If the scrape succeeds, Prometheus records:

```
up{instance="target:9100", job="node"} = 1
```

The `up` metric is automatic. You don't have to do anything. Every target gets an `up` series. This is how you know if a target is alive.

### Scrape duration

Prometheus also records `scrape_duration_seconds` for each scrape — how long it took to scrape that target. If your `/metrics` endpoint is slow to generate (lots of metrics, or some metrics are computed by walking a directory), the scrape duration goes up. If the scrape duration exceeds the scrape interval, you have a problem: each scrape will be done after the next one was supposed to start. Prometheus will eventually give up on those targets. Watch this metric.

### Configuration

Scrape config lives in `prometheus.yml`. Here is a tiny example:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'node'
    static_configs:
      - targets:
          - 'localhost:9100'
          - 'web1:9100'
          - 'web2:9100'

  - job_name: 'app'
    metrics_path: '/metrics'
    static_configs:
      - targets:
          - 'app1:8080'
          - 'app2:8080'
```

Each `job_name` is a logical group of targets. Each target gets a `job` label and an `instance` label automatically. So scraping `web1:9100` under `job_name: node` produces metrics labeled `{job="node", instance="web1:9100"}`.

## Service Discovery

Listing every target by hand in `static_configs` is fine for three or four servers. But if you have a Kubernetes cluster with 500 pods that come and go every minute, listing them by hand would be insane.

**Service discovery (SD)** lets Prometheus ask another system "what targets should I scrape?" and update the list dynamically.

### Static config (`static_configs`)

Just a list. You write it. It doesn't change unless you reload Prometheus. Good for a small fixed environment.

```yaml
scrape_configs:
  - job_name: 'static'
    static_configs:
      - targets: ['host1:9100', 'host2:9100']
```

### File SD (`file_sd_configs`)

Prometheus watches a JSON or YAML file on disk. You (or another program) can edit that file. Prometheus picks up changes within a few seconds without a full reload. Useful for "I want to script my own discovery."

```yaml
scrape_configs:
  - job_name: 'file-sd'
    file_sd_configs:
      - files:
          - '/etc/prometheus/targets/*.json'
        refresh_interval: 30s
```

The JSON file looks like:

```json
[
  {
    "targets": ["host1:9100", "host2:9100"],
    "labels": {
      "datacenter": "dc1",
      "team": "platform"
    }
  }
]
```

### Kubernetes SD (`kubernetes_sd_configs`)

Prometheus talks to the Kubernetes API and discovers pods, services, endpoints, nodes, and ingresses. This is how the Prometheus Operator (and the kube-prometheus-stack) make scraping a thousand-pod cluster work effortlessly.

```yaml
scrape_configs:
  - job_name: 'k8s-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
```

The role can be: `pod`, `service`, `endpoints`, `endpointslice`, `ingress`, `node`. Each role gives you a different set of `__meta_kubernetes_*` labels you can use for relabeling.

### Consul SD (`consul_sd_configs`)

If you use HashiCorp Consul for service discovery, Prometheus can read services straight out of Consul.

```yaml
scrape_configs:
  - job_name: 'consul'
    consul_sd_configs:
      - server: 'consul.local:8500'
        services: ['web', 'api']
```

### EC2 SD (`ec2_sd_configs`)

Talks to the AWS EC2 API to discover instances. Tags become labels. Useful for "scrape every instance with tag `env=prod`."

### DNS SD (`dns_sd_configs`)

Resolves a DNS SRV or A record and treats the answers as targets. Cheap and simple if you already have DNS records.

```yaml
scrape_configs:
  - job_name: 'dns'
    dns_sd_configs:
      - names: ['_prometheus._tcp.example.com']
        type: 'SRV'
```

### HTTP SD (`http_sd_configs`)

You write a tiny HTTP service that returns JSON listing your targets. Prometheus polls your service. Bring-your-own-discovery for any system Prometheus doesn't natively support.

```yaml
scrape_configs:
  - job_name: 'http-sd'
    http_sd_configs:
      - url: 'http://my-discovery-svc/targets'
```

The endpoint returns the same JSON shape as file_sd:

```json
[
  {"targets": ["host1:9100"], "labels": {"team": "data"}}
]
```

### Other built-in SDs

Prometheus has built-in SDs for: Azure, GCE, OpenStack, Hetzner, Linode, Vultr, DigitalOcean, Marathon, Mesos, Nomad, Triton, OVHcloud, Lightsail, Scaleway, Eureka, IONOS, Kuma, Uyuni, Puppet DB, and a few others. Most of them work the same way: tell Prometheus the API endpoint, and it discovers what's there.

## Relabeling

This is the most powerful feature in Prometheus. It is also the most confusing. We are going to go slow.

### What problem does relabeling solve?

Service discovery gives you a list of targets, each with a bunch of metadata labels. Most of those labels start with `__meta_` and are temporary — Prometheus throws them away unless you tell it to keep them. Some labels start with `__` (like `__address__`, the network address to scrape) and are special — they control behavior.

Relabeling lets you:

- **Drop targets** you don't want to scrape (`action: drop`)
- **Keep targets** that match a pattern (`action: keep`)
- **Rewrite labels** before storing (`action: replace`)
- **Set the address** to scrape (`action: replace` on `__address__`)
- **Hash and bucket** targets across multiple Prometheus servers (`action: hashmod`)
- **Translate metadata into permanent labels** (the most common use)

### The shape of a relabel rule

```yaml
relabel_configs:
  - source_labels: [__meta_kubernetes_pod_label_app]
    target_label: app
    action: replace
```

This says: "Take the value of the label `__meta_kubernetes_pod_label_app` and copy it into a label called `app`." So if the pod has a Kubernetes label `app: web`, the resulting time series will have a Prometheus label `app="web"`.

### Common relabel actions

#### `replace` (the default)

Take some source labels, run them through a regex, write the result into a target label.

```yaml
- source_labels: [__address__]
  regex: '([^:]+):.*'
  target_label: hostname
  replacement: '$1'
```

This grabs the hostname out of `host:port` and stores it in a label called `hostname`.

#### `keep`

Drop the target unless the regex matches. Used to filter "I only want to scrape pods that have annotation X."

```yaml
- source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
  regex: 'true'
  action: keep
```

Now you only scrape pods annotated `prometheus.io/scrape=true`. Other pods get silently discarded before scraping.

#### `drop`

The opposite of keep. Drop the target if the regex matches.

```yaml
- source_labels: [__meta_kubernetes_pod_label_skip_monitoring]
  regex: 'true'
  action: drop
```

#### `labelmap`

Bulk-copy labels matching a regex into new labels. Common pattern for Kubernetes:

```yaml
- action: labelmap
  regex: '__meta_kubernetes_pod_label_(.+)'
```

Every Kubernetes pod label `__meta_kubernetes_pod_label_app` becomes a Prometheus label `app`, and so on.

#### `hashmod`

Hash a label and modulo it. Used for sharding: if you have ten Prometheus servers and want each to scrape one-tenth of the targets, every server uses `hashmod` with `modulus: 10` and a different `regex` to keep only its slice.

```yaml
- source_labels: [__address__]
  modulus: 10
  target_label: __tmp_hash
  action: hashmod
- source_labels: [__tmp_hash]
  regex: '3'
  action: keep
```

This would make this Prometheus server scrape only the targets whose hash mod 10 equals 3.

### Two relabeling phases

There are actually **two** relabeling stages, and people mix them up constantly.

**`relabel_configs`** runs **before** the scrape. It decides whether to scrape a target at all, what `__address__` to scrape, what labels go on the resulting series. Almost all relabeling lives here.

**`metric_relabel_configs`** runs **after** the scrape, on the metrics themselves. It can drop specific metrics by name, drop high-cardinality labels, rewrite metric names, or filter individual samples.

```yaml
metric_relabel_configs:
  - source_labels: [__name__]
    regex: 'go_gc_.+'
    action: drop
```

This drops every Go garbage collection metric after the scrape. Useful when you don't care about GC stats and you want to save storage.

### Putting it together: a full Kubernetes pod scrape config

```yaml
- job_name: 'kubernetes-pods'
  kubernetes_sd_configs:
    - role: pod
  relabel_configs:
    # Only scrape pods annotated with prometheus.io/scrape=true
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
      action: keep
      regex: true
    # Use the annotation prometheus.io/path if it exists
    - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
      action: replace
      target_label: __metrics_path__
      regex: (.+)
    # Use the annotation prometheus.io/port to override the default port
    - source_labels: [__address__, __meta_kubernetes_pod_annotation_prometheus_io_port]
      action: replace
      regex: ([^:]+)(?::\d+)?;(\d+)
      replacement: $1:$2
      target_label: __address__
    # Copy all pod labels into Prometheus labels
    - action: labelmap
      regex: __meta_kubernetes_pod_label_(.+)
    # Add namespace and pod name as labels
    - source_labels: [__meta_kubernetes_namespace]
      action: replace
      target_label: kubernetes_namespace
    - source_labels: [__meta_kubernetes_pod_name]
      action: replace
      target_label: kubernetes_pod_name
```

This is the standard pattern. Read it slowly. Each block does one job. Together they turn the Kubernetes API into a fully labeled set of scrape targets.

## PromQL

PromQL is the question language. You ask Prometheus questions in PromQL.

### Two kinds of vector

PromQL has two fundamental types: **instant vectors** and **range vectors.** This is the most important distinction in PromQL. Get this and the rest is easy. Miss this and nothing makes sense.

#### Instant vector

A snapshot. One sample per series at one moment.

```
http_requests_total
```

That returns: for every series matching `http_requests_total`, the most recent sample (within the last 5 minutes). If you have 500 series of `http_requests_total`, you get 500 numbers back. Each number is the latest value.

#### Range vector

A history. A range of samples per series over a time window.

```
http_requests_total[5m]
```

That returns: for every series matching `http_requests_total`, every sample within the last 5 minutes. If your scrape interval is 15s, that's 20 samples per series. If you have 500 series, you get 500 lists of 20 samples — that is 10,000 samples in total.

You cannot directly graph a range vector. Range vectors only show up as the input to functions like `rate()` and `avg_over_time()`. The functions reduce the range back down to an instant vector.

### Selectors and matchers

Filter series by metric name and labels.

```
http_requests_total{method="GET"}              # exact match
http_requests_total{method!="GET"}             # not equal
http_requests_total{path=~"/api/.+"}           # regex match
http_requests_total{path!~"/health.*"}         # regex not match
http_requests_total{method="GET", status="200"}  # AND
```

You can omit the metric name if you specify at least one label matcher:

```
{__name__=~"http_.*", method="GET"}
```

This selects every series whose metric name starts with `http_` AND has `method=GET`.

### `offset` modifier

Look back in time.

```
http_requests_total offset 1h
```

Returns the value of `http_requests_total` from one hour ago.

### `@` modifier (added in Prometheus 2.25)

Pin a query to a specific timestamp.

```
http_requests_total @ 1714190400
```

Returns the value at exactly that Unix timestamp. Useful when you want to compare "now" to a specific moment.

### Aggregation operators

Reduce many series into fewer.

```
sum(http_requests_total)                      # one number, total across everything
sum by (method) (http_requests_total)         # one per method
sum without (instance) (http_requests_total)  # collapse instance dimension
avg(node_cpu_seconds_total)                   # average
max(node_cpu_seconds_total)                   # max
min(node_cpu_seconds_total)                   # min
count(up)                                     # how many series exist
stddev(node_load1)                            # standard deviation
topk(5, http_requests_total)                  # top 5
bottomk(5, http_requests_total)               # bottom 5
quantile(0.95, http_request_duration_seconds) # quantile of an instant vector
```

`by` and `without` work the same way: `by` says "keep only these labels," `without` says "drop these labels." They are the inverse of each other.

### Binary operators

Math between two vectors, or between a vector and a scalar.

```
http_requests_total / 1024            # divide every sample by 1024
http_requests_total * 8               # multiply every sample by 8
http_requests_total > 1000            # filter: keep only series above 1000
http_requests_total - http_requests_total offset 1h   # delta vs an hour ago
```

When two vectors are involved, Prometheus needs to know how to **match** them. The default matching is "match on identical label sets" — `one-to-one`. If the label sets don't match exactly, you must say how:

```
left_metric / on(instance) right_metric                        # match only on `instance`
left_metric / ignoring(method) right_metric                    # match on everything except `method`
left_metric / on(instance) group_left right_metric             # one-to-many
left_metric / on(instance) group_right(extra_label) right_metric  # many-to-one with extra label
```

If you don't specify and the labels don't match, you get the legendary error: `many-to-many matching not allowed: matching labels must be unique on one side`. We will see that error a lot.

### Subqueries

A subquery is a range vector built from a query result. Syntax: `<query>[<range>:<resolution>]`.

```
max_over_time(rate(http_requests_total[5m])[1h:1m])
```

That says: "Run `rate(http_requests_total[5m])` every 1 minute over the last 1 hour, then take the max."

Subqueries are powerful but expensive. Use sparingly. They can blow up if `<range>` is huge.

### `rate()` versus `irate()` versus `increase()`

These three are the most common counter functions, and people mix them up.

#### `rate(metric[range])`

Average per-second rate of increase of a counter, over the range. This is the function you want **almost always.**

```
rate(http_requests_total[5m])
```

That says: "Look at the last 5 minutes of samples. Compute the average per-second rate of increase. Return that as the current value."

For example, if `http_requests_total` was 1000 five minutes ago and is 1300 now, that's 300 requests in 300 seconds, so `rate(http_requests_total[5m])` returns 1.0 — one request per second on average.

`rate()` is **counter-aware.** If the counter resets (because the process restarted and the value went back to zero), `rate()` detects this and adjusts. It does not see the reset as a sudden drop to negative infinity.

#### `irate(metric[range])`

Per-second rate based on the **last two samples in the range.** Reactive. Spiky.

```
irate(http_requests_total[5m])
```

This still requires a 5-minute range (for `irate` to find at least two samples), but it only uses the most recent two. It tells you what the rate is **right now**, not the average over five minutes.

`irate()` is good for graphs where you want sharp visible spikes. `irate()` is bad for alerting because tiny noise can flip your alert on and off. Use `rate()` for alerts.

#### `increase(metric[range])`

The total amount the counter increased over the range. Equal to `rate(metric[range]) * range_in_seconds`. Counter-aware (handles resets the same way).

```
increase(http_requests_total[1h])
```

How many requests in the last hour. If the counter went from 1000 to 4600, this returns 3600.

#### Quick rules

- Use `rate()` for alerts and dashboards. Almost always.
- Use `irate()` for "what is happening this very second" graphs.
- Use `increase()` when you want a total count over a window, not a per-second rate.
- The range for `rate()` should be at least 4× the scrape interval. So if you scrape every 15s, use `rate(...[1m])` minimum. `[5m]` is the standard default.

### `avg_over_time()` and friends

For gauges. Compute statistics over a range.

```
avg_over_time(node_load1[5m])
max_over_time(node_temperature_celsius[1h])
min_over_time(node_memory_free_bytes[24h])
sum_over_time(node_network_bytes_recv[1h])
count_over_time(up{job="api"}[1h])
```

`*_over_time` functions take a range vector and return an instant vector. They reduce time, not labels.

### `histogram_quantile()`

The function for computing quantiles from a histogram.

```
histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))
```

That says: "Take the 5-minute rate of every histogram bucket. From those rates, compute the 99th percentile of the request duration."

The argument **must be a histogram with an `le` label.** If you pass anything else, you get strange or empty results. If you pass a summary's `quantile=0.99` series, that doesn't work — `histogram_quantile()` is only for histograms.

Common gotcha: people forget to wrap the bucket in `rate()`. The bucket is a counter (it goes up forever). `histogram_quantile()` of the raw counter would give you the quantile across all time, which is meaningless. You almost always want `rate(*_bucket[5m])` first.

To compute a 99th percentile per route:

```
histogram_quantile(
  0.99,
  sum by (le, route) (
    rate(http_request_duration_seconds_bucket[5m])
  )
)
```

The `sum by (le, route)` groups buckets across instances, so you get one quantile per route, not one per route per instance.

## Recording Rules

Some queries are expensive. Some are run all the time. If you have a dashboard that runs `histogram_quantile(0.99, sum by (le, route) (rate(http_request_duration_seconds_bucket[5m])))` every 5 seconds for every panel, your Prometheus server will be sad.

A **recording rule** runs a query on a schedule and stores the result as a new metric. Then your dashboard reads the new metric, which is one cheap lookup, instead of running the expensive query every time.

```yaml
groups:
  - name: latency-rules
    interval: 30s
    rules:
      - record: route:http_request_duration_seconds:p99
        expr: |
          histogram_quantile(0.99,
            sum by (le, route) (
              rate(http_request_duration_seconds_bucket[5m])
            )
          )
```

Now `route:http_request_duration_seconds:p99` is a real metric in your TSDB. The pipeline is: Prometheus runs the rule every 30 seconds, computes the result, stores the result as a new series with the labels `le` and `route` (well, mostly — `histogram_quantile` strips `le`).

### Naming convention

By convention, recording rules use a name with **colons** like `aggregation:metric:operation`. Example: `instance:node_cpu:rate5m`. Colons are illegal in normal metric names but legal in recording rule outputs. The colon convention is your hint to the reader: "this isn't raw, it was computed."

### Recording rule pipeline

```
   raw scrapes
        │
        ▼
   ┌──────────────────────┐
   │  TSDB (raw series)   │
   └──────────────────────┘
            │
            │ recording rule runs every interval
            ▼
   ┌────────────────────────────────────────┐
   │  PromQL: rate(http_..._bucket[5m])     │
   │  → histogram_quantile(0.99, ...)       │
   └────────────────────────────────────────┘
            │
            ▼
   ┌──────────────────────────────────────────┐
   │  TSDB (route:http_...:p99)               │  (cheap to query)
   └──────────────────────────────────────────┘
            │
            ▼
        dashboards / alerts
```

## Alerting Rules

An **alerting rule** is a recording rule that fires alerts when a condition is true.

```yaml
groups:
  - name: latency-alerts
    interval: 30s
    rules:
      - alert: HighRequestLatency
        expr: |
          histogram_quantile(0.99,
            sum by (le, route) (
              rate(http_request_duration_seconds_bucket[5m])
            )
          ) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "99p latency is over 1 second on {{ $labels.route }}"
          description: "Route {{ $labels.route }} has had 99p latency above 1 second for 5 minutes."
```

### How alerts work

Prometheus evaluates the `expr` every `interval`. If the result is non-empty (i.e., the condition is true for at least one series), each series is a **pending** alert.

If the alert continues to be pending for at least the `for:` duration, it transitions to **firing.** Firing alerts get sent to Alertmanager.

If the condition stops being true, the alert resolves. Alertmanager sends a "resolved" notification.

The `for:` clause is your patience setting. If you alert with `for: 0s`, you'll get paged every time a metric blips. If you alert with `for: 5m`, the condition has to be sustained for five minutes before anyone gets paged. Tune this carefully.

### Alert states

```
inactive ──── condition becomes true ────► pending
pending  ──── condition stays true for 'for:' duration ────► firing
pending  ──── condition becomes false ────► inactive
firing   ──── condition becomes false ────► inactive (with resolved notification)
```

### Labels and annotations

`labels:` are added to the alert and used by Alertmanager for routing (more later). Like `severity=warning` or `team=platform`.

`annotations:` are templated text for humans — what the alert says when it pages you. They use Go templating with `{{ $labels.foo }}` and `{{ $value }}`.

## Alertmanager

Prometheus only fires alerts. It doesn't send pages, emails, Slack messages, or PagerDuty incidents. **Alertmanager** does that.

Prometheus pushes firing alerts to Alertmanager. Alertmanager handles routing, grouping, deduplication, silences, and inhibition.

### The Alertmanager pipeline

```
Prometheus ─────► Alertmanager
                       │
                       ▼
                ┌────────────┐
                │   group    │  alerts with same labels are batched
                └────────────┘
                       │
                       ▼
                ┌────────────┐
                │ inhibition │  some alerts suppress others
                └────────────┘
                       │
                       ▼
                ┌────────────┐
                │  silences  │  manual mute (eg during maintenance)
                └────────────┘
                       │
                       ▼
                ┌────────────┐
                │   route    │  decide receiver based on labels
                └────────────┘
                       │
                       ▼
                ┌────────────┐
                │  receiver  │  email / Slack / PagerDuty / webhook
                └────────────┘
```

### Routing

A routing tree decides which receiver each alert goes to.

```yaml
route:
  receiver: 'default'
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  routes:
    - matchers:
        - severity = critical
      receiver: 'pagerduty'
    - matchers:
        - severity = warning
      receiver: 'slack-warnings'
    - matchers:
        - team = data
      receiver: 'data-team-slack'

receivers:
  - name: 'default'
    email_configs:
      - to: 'oncall@example.com'
  - name: 'pagerduty'
    pagerduty_configs:
      - service_key: 'ABCDEFG'
  - name: 'slack-warnings'
    slack_configs:
      - api_url: 'https://hooks.slack.com/...'
        channel: '#alerts'
  - name: 'data-team-slack'
    slack_configs:
      - channel: '#data-alerts'
```

### Routing tree picture

```
                  alert arrives
                       │
                       ▼
            ┌──────────────────────┐
            │ route: receiver=     │
            │   default            │
            │ matchers: (none —    │
            │  catch-all)          │
            └──────────────────────┘
                       │
       ┌───────────────┼───────────────┐
       ▼               ▼               ▼
  severity=         severity=        team=
  critical?         warning?         data?
       │               │               │
       ▼               ▼               ▼
  pagerduty       slack-warnings   data-team-slack
```

The first matching child wins (unless `continue: true`). Default receiver is the leaf.

### Grouping

`group_by` says: "alerts with the same values in these labels should be one notification, not many." If 50 servers all fire `HighDiskUsage` at the same time, group_by `[alertname]` gives you one Slack message listing 50 servers, not 50 separate messages.

`group_wait` is how long to wait before sending the first batch of a new group (so you can collect more alerts).
`group_interval` is how long to wait between sending different alerts in an already-existing group.
`repeat_interval` is how often to re-notify if the alert is still firing.

### Inhibition

When alert A fires, suppress alert B. Useful when A is a symptom of B, or when one big alert should drown out many smaller ones.

```yaml
inhibit_rules:
  - source_matchers:
      - alertname = ClusterDown
    target_matchers:
      - alertname = ServiceDown
    equal: ['cluster']
```

If `ClusterDown` is firing for cluster X, suppress all `ServiceDown` alerts for the same cluster X. The whole cluster being down is the obvious problem; service-by-service alerts are noise.

### Silences

A silence is a manual mute. "Don't page me about this for the next four hours, I'm doing maintenance."

```bash
$ amtool silence add alertname=HighDiskUsage instance=foo --duration=4h --comment="rebooting for upgrade"
```

Silences are stored in Alertmanager's state. They expire automatically.

### Receivers

Built-in: email, PagerDuty, Slack, OpsGenie, VictorOps, Webhook, Pushover, Telegram, WeChat, Discord, Microsoft Teams (via webhook), and a generic webhook for anything else.

## Exporters

An **exporter** is a small program that talks to something that doesn't speak Prometheus, and exposes a `/metrics` endpoint that does. Exporters translate.

### `node_exporter`

The most famous exporter. Runs on a Linux/Unix host, reads `/proc` and `/sys`, exposes hundreds of host-level metrics: CPU, memory, disk, network, filesystem, kernel statistics. Lives at `localhost:9100/metrics` by default.

```bash
$ wget https://github.com/prometheus/node_exporter/releases/download/v1.8.2/node_exporter-1.8.2.linux-amd64.tar.gz
$ tar xvzf node_exporter-*.tar.gz
$ ./node_exporter-1.8.2.linux-amd64/node_exporter
```

### `blackbox_exporter`

Probes endpoints from the outside. HTTP, HTTPS, TCP, ICMP, DNS. Lives at `localhost:9115`. You don't scrape `node_exporter` style — you tell `blackbox_exporter` what URL to probe, and it returns metrics about that probe.

```yaml
scrape_configs:
  - job_name: 'blackbox'
    metrics_path: /probe
    params:
      module: [http_2xx]
    static_configs:
      - targets:
          - https://example.com
          - https://api.example.com
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: blackbox-exporter:9115
```

That weird relabel dance moves the target URL into a query parameter for `blackbox_exporter` while keeping the original URL as the `instance` label. It is a famous and confusing pattern. Read it carefully.

### `mysqld_exporter`

Connects to a MySQL/MariaDB server, runs `SHOW GLOBAL STATUS` and various other queries, exposes the results as metrics. Lives at `localhost:9104`.

### `postgres_exporter`

Same idea for PostgreSQL. Connects to Postgres, runs introspection queries, exposes metrics. Lives at `localhost:9187`.

### `snmp_exporter`

Talks to anything that speaks SNMP — switches, routers, UPSs, printers — using a generator-built MIB to OID map. Translates SNMP MIB objects into Prometheus metrics. Lives at `localhost:9116`.

You configure an SNMP "module" (a map of which OIDs to walk for which device) and then scrape:

```yaml
- job_name: 'snmp'
  static_configs:
    - targets: ['router1', 'router2']
  metrics_path: /snmp
  params:
    module: [if_mib]
  relabel_configs:
    - source_labels: [__address__]
      target_label: __param_target
    - source_labels: [__param_target]
      target_label: instance
    - target_label: __address__
      replacement: snmp-exporter:9116
```

(Same relabel pattern as blackbox.)

### `jmx_exporter`

For Java applications. Loads as a Java agent, reads JMX MBeans, exposes them as Prometheus metrics. Used heavily for Kafka, Cassandra, Tomcat, anything Java.

### Other notable exporters

- `redis_exporter`, `memcached_exporter`, `nginx-prometheus-exporter`, `apache_exporter`, `haproxy_exporter`
- `kafka_exporter`, `rabbitmq_exporter`, `consul_exporter`, `etcd` (built-in metrics)
- `windows_exporter` (Windows host counterpart of node_exporter)
- `cadvisor` (container metrics: CPU/mem per container)
- `kube-state-metrics` (Kubernetes object state: deployments, pods, etc.)

The list is long. There is an exporter for almost everything. Search "prometheus X exporter" and one usually exists.

## Pushgateway (and when NOT to use it)

The Pushgateway is a small server that holds pushed metrics until Prometheus scrapes it. Designed for **batch jobs** that finish before Prometheus can scrape them.

### How it works

Your batch job does its work, computes its final metrics, then before exiting:

```bash
echo "job_last_success_unixtime $(date +%s)" | curl --data-binary @- \
  http://pushgateway:9091/metrics/job/nightly_backup
```

That stores the metric in Pushgateway under group `{job="nightly_backup"}`. Prometheus scrapes Pushgateway like any other target. The metric appears in your Prometheus.

### When to use Pushgateway

- One-shot batch jobs that exit before scrape-time.
- Cron jobs that run for seconds.
- CI/CD pipelines that want to record final state.

### When NOT to use Pushgateway

- **It is NOT a queue or buffer.** Don't use it to batch up metrics from many sources. It does not fan in. It does not aggregate. It just holds the last value pushed to a group.
- **It is NOT for service-to-service push.** If you have a long-running service, expose `/metrics` directly. Don't push.
- **It does not delete old metrics.** Once you push, the metric lives in Pushgateway forever (or until you DELETE it). Old metrics from old jobs stick around until you clean them up. The `up` semantics are wrong: a Pushgateway-backed metric still shows the last value even if the original job died horribly.
- **It is a single point of failure.** All those batch metrics flow through one process. Lose Pushgateway, lose the visibility into batch jobs.

A good rule: if you find yourself using Pushgateway for anything other than short-lived batch jobs, you are probably misusing it.

## Long-Term Storage

Prometheus stores data on local disk. By default, retention is 15 days. You can crank that up to months if you have the disk, but a single Prometheus is not great at storing years of data, and it is not clusterable — every Prometheus is its own island.

For long-term storage and global query, four big projects exist. They all support the Prometheus **remote_write** protocol (more on that next), so Prometheus can stream samples to them in real-time.

### Mimir (Grafana Labs)

Multi-tenant, horizontally scalable, blob-storage backed. Hosted on S3/GCS/Azure Blob. Speaks PromQL natively. Same wire compatibility as Prometheus. Battle-tested at huge scale (tens of millions of active series per tenant).

### Thanos

Built around Prometheus's local TSDB blocks. A "sidecar" runs alongside each Prometheus and uploads completed blocks to S3. Other Thanos components (Querier, Store Gateway, Compactor) provide global view, dedup, and downsampling. Lower operational burden than Mimir for small/medium deployments.

### Cortex

The grandfather. Mimir was forked from Cortex. Still maintained, still used, but most new deployments choose Mimir or Thanos.

### VictoriaMetrics

A from-scratch reimplementation. Not a fork of Prometheus — its own TSDB. Speaks the Prometheus protocols and has its own query language (MetricsQL) that is a superset of PromQL. Known for very low memory usage and very high ingestion rate per CPU core. Both single-node and cluster modes.

### Trade-offs

If you want one box: VictoriaMetrics single-node, or just Prometheus with bigger retention.
If you want S3-backed simplicity: Thanos.
If you want enterprise-scale multi-tenant: Mimir.
If you want a from-scratch performant TSDB: VictoriaMetrics cluster.

## Federation

Federation is when one Prometheus scrapes another Prometheus. It is the simplest way to get a global view.

### Use case: hierarchy

Each data center has its own Prometheus scraping its own targets. A "global" Prometheus federates from each DC Prometheus, pulling only **aggregated** metrics, not raw ones.

```
                    ┌──────────────────────┐
                    │ Global Prometheus    │
                    │ /federate            │
                    └──────────┬───────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                ▼                ▼
       ┌────────────┐    ┌────────────┐   ┌────────────┐
       │ DC1 Prom   │    │ DC2 Prom   │   │ DC3 Prom   │
       │ /metrics   │    │ /metrics   │   │ /metrics   │
       │ /federate  │    │ /federate  │   │ /federate  │
       └────────────┘    └────────────┘   └────────────┘
              │                │                │
       ┌──────┴──────┐  ┌──────┴──────┐  ┌──────┴──────┐
       │ all targets │  │ all targets │  │ all targets │
       └─────────────┘  └─────────────┘  └─────────────┘
```

The global server scrapes `/federate?match[]=...` from each leaf, getting only the recording rule outputs (those `aggregation:metric:rate5m` series). Raw cardinality stays in the DC. Aggregated rollups go to the global.

```yaml
- job_name: 'federate'
  scrape_interval: 30s
  honor_labels: true
  metrics_path: '/federate'
  params:
    'match[]':
      - '{__name__=~"job:.*"}'
      - 'up'
  static_configs:
    - targets:
        - 'dc1-prometheus:9090'
        - 'dc2-prometheus:9090'
```

`honor_labels: true` is critical. Without it, the global Prometheus would overwrite the labels from the leaf Prometheus, and you'd lose track of which DC the metric came from.

Federation is **not** a long-term storage strategy. It is a hierarchy strategy. For long-term, use remote_write to Mimir/Thanos/etc.

## Remote Write

`remote_write` is a protocol Prometheus uses to stream samples to a remote endpoint in real time. Most long-term storage systems are remote_write receivers.

```yaml
remote_write:
  - url: "https://mimir.example.com/api/v1/push"
    basic_auth:
      username: tenant1
      password: xxxx
```

Prometheus buffers samples in memory and sends them in batches. The protocol is Snappy-compressed protobuf over HTTPS. Each batch contains many series and many samples.

### remote_write best practices

- Batch settings (`max_samples_per_send`, `max_shards`, `capacity`) matter when you have high write rates. Defaults are fine for most.
- Drop labels before sending. Use `write_relabel_configs` to drop high-cardinality labels you don't need remotely.
- Watch `prometheus_remote_storage_*` metrics. They tell you if the queue is backing up.
- If the remote dies, Prometheus buffers up to a few hours on disk via the **WAL** (write-ahead log). After that, samples are dropped.

### remote_write 2.0 (Prometheus 3.0)

Prometheus 3.0 introduced **Remote Write 2.0**, which sends additional metadata with samples (units, exemplars, native histograms more efficiently) and supports multi-tenant metadata flowing end-to-end. If you are setting up new infrastructure, prefer 2.0 endpoints.

### Agent mode (Prometheus 2.30)

Prometheus 2.30 added an **agent mode** which strips out most of Prometheus's features and turns it into a pure scrape-and-remote_write daemon. No local storage, no PromQL queries, no UI. Just scrape, buffer, push. Useful when you want a lightweight collector at the edge that streams everything to a central system.

```bash
$ prometheus --config.file=agent.yml --enable-feature=agent
```

## Common Errors

Real strings you will see at 3am. Memorize the fix.

### `parse error: unclosed brace`

You wrote a PromQL query like `http_requests_total{method="GET"` and forgot the closing brace. Add `}`.

### `many-to-many matching not allowed: matching labels must be unique on one side`

Binary operation between two vectors where multiple series on each side share the same label set. Add `on(...)`, `ignoring(...)`, `group_left`, or `group_right` to disambiguate.

```
# bad
metric_a / metric_b

# good
metric_a / on(instance) metric_b
metric_a / on(instance) group_left metric_b
```

### `vector cannot contain metrics with the same labelset`

A query produced two samples with identical labels at the same timestamp. Usually from a relabel or aggregation that collapsed too many dimensions. Add a `by` clause that includes a distinguishing label.

### `query processing would load too many samples into memory in query execution`

Prometheus protected itself. Your query was going to read too many samples. Either narrow the time range, narrow the label matcher, or precompute via a recording rule.

### `range query exceeds maximum resolution`

Your `--query.max-samples` cap was hit. Same fix: narrow the query, or raise the limit (`--query.max-samples=200000000`) carefully.

### `out-of-order sample`

A sample arrived with a timestamp **before** an already-stored sample for the same series. Often from clock skew on the scrape target, or a target restart that rewinds a counter. Newer Prometheus versions (2.39+) support out-of-order ingestion if enabled. Otherwise, fix the clock or the target.

### `sample limit exceeded`

A target exposed more samples in one scrape than `sample_limit` (default unlimited; sometimes set to e.g. 50000). Prometheus dropped the entire scrape. Lower the cardinality at the source, or raise `sample_limit`.

### `scrape duration exceeded`

The scrape took longer than the scrape interval. Either the target is slow to render `/metrics`, or the network is slow, or the metric set is huge. Increase scrape_interval, or speed up the target's exposition.

### `targets dropped because of relabeling`

A relabel rule with `action: keep` rejected the target. This is normal if the rule is intentional. If it's not, check the `__meta_*` labels on the discovered target — your filter doesn't match what's actually there.

### `alert was firing but is now resolved`

Alertmanager telling you a previously-firing alert went back to OK. Not an error — informational.

### `alertmanager 422 / unrecognized labels`

You configured a routing tree that referenced a label that no alert has, or a receiver expected fields that aren't present. Check your route's `matchers` and your receiver config against the actual alert payload. Use `amtool` to test.

### `context deadline exceeded`

A scrape or query took longer than its timeout. Same fix as scrape duration: speed up, raise the timeout, or shrink the workload.

### `cannot serve label values: query timed out in expression evaluation`

A label values request was too expensive. Common when you have millions of unique values for a label. Same answer as before: lower cardinality.

### `dropped Prometheus exposition`

In OpenMetrics format with strict parsing enabled, sometimes a metric line fails. Look at the actual line — usually a label value with an unescaped quote, or a malformed `# TYPE`.

### `head out of order chunk` / `tsdb mmap chunk`

TSDB internal complaints. Usually transient. If persistent, you may have a corrupt block — check `prometheus_tsdb_*` metrics, possibly delete corrupted blocks (with care), restart.

## Hands-On

These are real commands. Run them. Watch what happens.

### Versions and basic startup

```bash
$ prometheus --version
prometheus, version 3.0.1 (branch: HEAD, revision: ...)

$ prometheus --help
# prints the giant help

$ prometheus --config.file=prometheus.yml
# starts Prometheus on :9090

$ prometheus --config.file=prometheus.yml \
    --storage.tsdb.path=/data \
    --storage.tsdb.retention.time=30d \
    --web.enable-lifecycle \
    --web.enable-admin-api
```

### Validate config and rules with promtool

```bash
$ promtool check config prometheus.yml
Checking prometheus.yml
 SUCCESS: 1 rule files found
 SUCCESS: prometheus.yml is valid prometheus config file syntax

$ promtool check rules /etc/prometheus/rules/*.yml
Checking /etc/prometheus/rules/latency.yml
  SUCCESS: 4 rules found

$ promtool check metrics < /tmp/exported_metrics.txt
# pipe a /metrics file in to validate exposition format
```

### Run a query from the command line

```bash
$ promtool query instant http://localhost:9090 'up'
up{instance="localhost:9100", job="node"} => 1 @[1714190445.000]
up{instance="localhost:9090", job="prometheus"} => 1 @[1714190445.000]

$ promtool query range http://localhost:9090 \
    'rate(http_requests_total[5m])' \
    --start='2026-04-27T00:00:00Z' \
    --end='2026-04-27T01:00:00Z' \
    --step=60s

$ promtool query series http://localhost:9090 --match='up{job="node"}'

$ promtool query labels http://localhost:9090 instance
```

### Analyze the TSDB

```bash
$ promtool tsdb analyze /var/lib/prometheus/data
# prints the top label values and cardinality offenders

$ promtool tsdb list /var/lib/prometheus/data
# lists blocks

$ promtool tsdb create-blocks-from openmetrics /tmp/old.txt /var/lib/prometheus/data
# import historical data
```

### Curl Prometheus directly

```bash
$ curl http://localhost:9090/metrics
# Prometheus's own metrics. Yes, Prometheus monitors itself.

$ curl 'http://localhost:9090/api/v1/query?query=up'
{"status":"success","data":{"resultType":"vector","result":[...]}}

$ curl 'http://localhost:9090/api/v1/query_range?query=rate(http_requests_total[5m])&start=1714190000&end=1714193600&step=15'

$ curl 'http://localhost:9090/api/v1/series?match[]=up'

$ curl http://localhost:9090/api/v1/labels
{"status":"success","data":["__name__","instance","job",...]}

$ curl http://localhost:9090/api/v1/label/job/values
{"status":"success","data":["node","prometheus","app"]}

$ curl http://localhost:9090/api/v1/targets
# all current targets, with state and last-scrape info

$ curl http://localhost:9090/api/v1/rules
# all loaded recording + alerting rules

$ curl http://localhost:9090/api/v1/alerts
# currently-firing alerts

$ curl -X POST http://localhost:9090/-/reload
# hot-reload config (requires --web.enable-lifecycle)

$ curl -X DELETE 'http://localhost:9090/api/v1/admin/tsdb/delete_series?match[]={job="dead-job"}'
# delete series (requires --web.enable-admin-api)
```

### Pretty-print JSON with jq

```bash
$ curl -s 'http://localhost:9090/api/v1/query?query=up' | jq '.data.result[] | {instance: .metric.instance, value: .value[1]}'
{"instance":"localhost:9100","value":"1"}
{"instance":"localhost:9090","value":"1"}
```

### Alertmanager CLI

```bash
$ amtool check-config alertmanager.yml
Checking 'alertmanager.yml'  SUCCESS

$ amtool config show
# prints currently loaded config

$ amtool alert query
# list firing alerts

$ amtool silence add alertname=HighDiskUsage instance=foo --duration=4h --comment="rebooting"
# silence creation

$ amtool silence query
# list active silences

$ amtool silence expire <silence-id>
# end a silence early
```

### Kubernetes commands

```bash
$ kubectl port-forward svc/prometheus-server 9090:9090
# now :9090 on your laptop is the Prometheus in the cluster

$ kubectl exec -it prometheus-0 -- promtool tsdb analyze /prometheus

$ kubectl logs -l app.kubernetes.io/name=prometheus --tail=200
```

### Docker

```bash
$ docker run -p 9090:9090 -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus

$ docker run -d -p 9100:9100 --net=host prom/node-exporter

$ docker run -d -p 9115:9115 -v $(pwd)/blackbox.yml:/etc/blackbox_exporter/config.yml prom/blackbox-exporter
```

### Opening the UI from the terminal

You don't have to leave the terminal. Either pipe to your favorite local browser-from-terminal (`w3m http://localhost:9090/graph`) or just hit the API:

```bash
$ curl -G --data-urlencode 'query=rate(http_requests_total[5m])' http://localhost:9090/api/v1/query | jq

$ curl -s http://localhost:9090/api/v1/status/runtimeinfo | jq
{
  "status": "success",
  "data": {
    "startTime": "2026-04-27T00:00:00Z",
    "CWD": "/",
    "reloadConfigSuccess": true,
    "lastConfigTime": "2026-04-27T00:00:00Z",
    "corruptionCount": 0,
    "goroutineCount": 47,
    ...
  }
}
```

### Parse alertmanager webhook payload

If you write a custom receiver that takes a webhook from Alertmanager:

```bash
$ cat /tmp/webhook-body.json | jq '.alerts[] | {name: .labels.alertname, severity: .labels.severity, summary: .annotations.summary}'
```

### Push from a script (Pushgateway)

```bash
$ cat <<EOF | curl --data-binary @- http://pushgateway:9091/metrics/job/nightly_backup
# TYPE backup_last_success_unixtime gauge
backup_last_success_unixtime $(date +%s)
EOF

$ curl -X DELETE http://pushgateway:9091/metrics/job/nightly_backup
# remove a group from Pushgateway
```

### Scrape one target manually for debugging

```bash
$ curl -s http://target:9100/metrics | head -50
$ curl -s http://target:9100/metrics | grep -E '^node_load1 '
$ curl -s http://target:9100/metrics | wc -l
6843
```

If `wc -l` prints 50,000 you have a cardinality problem on that target.

### Test PromQL without Prometheus running

```bash
$ promtool test rules tests.yml
# unit-test recording and alerting rules with golden inputs
```

## Common Confusions

Pairs people mix up. Keep this list near you.

### Counter vs gauge

Counter only goes up (and resets on restart). Gauge can go up or down. If your metric represents "the total number of X ever," it's a counter. If it represents "the current X right now," it's a gauge. When in doubt, ask: "if this process restarts, should this number reset to zero?" If yes, counter. If no, gauge.

### `rate()` vs `irate()` vs `increase()`

`rate()` averages over a window. `irate()` uses the last two samples. `increase()` returns the total over a window (no per-second division). For alerts and dashboards, almost always `rate()`. For instant-spike visibility on a graph, `irate()`. For "total events in the last hour," `increase()`.

### `histogram_quantile()` requires a histogram with `le` label

Only histograms have `le`. Summaries have `quantile`. If you call `histogram_quantile()` on a summary (or on a gauge), you get garbage or nothing. The function name has "histogram" in it for a reason.

### `quantile()` requires summary with `quantile` label

The plain `quantile()` aggregator works on instant vectors, computing a quantile across them. It's not the same as `histogram_quantile()`. Don't mix them up. `quantile(0.99, http_request_duration_seconds)` only works if your metric is a summary with a `quantile` label, and it's computing across instances. Different math.

### Pushgateway is NOT a queue

Pushgateway holds the **last** value pushed to a group. It does not queue. It does not aggregate. It does not buffer. It is for short-lived batch jobs only. Push another value and the previous one is overwritten.

### What is `up`?

`up` is a metric automatically created by Prometheus for every scrape attempt. `up=1` means the scrape succeeded. `up=0` means the scrape failed (target unreachable, returned non-200, or returned bad exposition). `up` is the canonical "is this target alive?" check.

### Why does my query return nothing?

A bunch of possible reasons:

- The metric name doesn't exist (typo).
- A label matcher doesn't match anything (wrong value).
- The query is asking for a moment when no data was scraped.
- Sample limit dropped the scrape silently.
- Relabeling silently dropped the target.
- The target is `up=0` and your query needs recent data.

Strategy: peel back. Try `up{job="..."}`. Then `count({__name__=~".+", job="..."})`. Then `count({__name__="your_metric"})`. Then add label matchers one at a time.

### Counter resets and `rate()`

When a process restarts, its counters reset to zero. `rate()` knows this. It looks at consecutive samples; if the next sample is **less** than the previous, `rate()` assumes a reset and treats the next sample as the new starting point. So `rate()` is robust to restarts. Don't write `delta()` on a counter — it doesn't have the reset logic.

### `relabel_configs` vs `metric_relabel_configs`

`relabel_configs` runs **before** the scrape and decides whether and how to scrape. `metric_relabel_configs` runs **after** the scrape on the metrics themselves. Use `relabel_configs` to filter targets and shape labels; use `metric_relabel_configs` to drop specific metrics or labels in the response.

### Service discovery vs static config

Static config is a hand-written list. SD is dynamic discovery from an external system. SD is required for any environment where targets come and go (Kubernetes, EC2 with ASGs). Static is fine for a small fixed fleet.

### Recording rules vs alerting rules

Recording rules pre-compute and **store** a query result as a new metric. Alerting rules evaluate a query and **fire alerts** when the result is non-empty. Both use the same Prometheus query engine. Only alerting rules talk to Alertmanager.

### Alertmanager grouping vs Prometheus grouping

Alertmanager's `group_by` controls which alerts get bundled into one notification. Prometheus's `by` clause in PromQL controls which labels survive an aggregation. Different concepts. Same word.

### One-shot job pattern with Pushgateway

For a cron job:

```bash
#!/bin/bash
START=$(date +%s)
do_the_work
END=$(date +%s)
DURATION=$((END - START))

cat <<EOF | curl --data-binary @- http://pushgateway:9091/metrics/job/cron_xyz/instance/$(hostname)
# TYPE cron_last_success_unixtime gauge
cron_last_success_unixtime $END
# TYPE cron_duration_seconds gauge
cron_duration_seconds $DURATION
EOF
```

The job pushes its final state. Prometheus scrapes Pushgateway. You alert on `time() - cron_last_success_unixtime{job="cron_xyz"} > 3600` if the job hasn't succeeded in an hour.

### What is a histogram exemplar?

An exemplar is a single observed value associated with a histogram bucket — a "for instance" example. They link metrics to traces: "this 99th-percentile slow request had trace ID abc123 — go look at it." Exemplars are exposed as `metric_bucket{...} value # exemplar_label="value"` in the OpenMetrics format. Grafana can show them as dots on a histogram heatmap.

### Cardinality explosion from `user_id` labels

The most common production disaster. Someone adds `user_id` as a label to a metric "to be helpful." Three weeks later Prometheus is using 200 GB of RAM and crashing. Don't put high-cardinality data in labels. Logs, not metrics.

### `remote_write` best practices

- Use a single consistent endpoint per destination. Don't fan out from one Prometheus to ten remote_writes — use one well-targeted one.
- Use `write_relabel_configs` to drop labels and series you don't need to send.
- Watch your queue lag (`prometheus_remote_storage_samples_pending`).
- Don't expect remote_write to be a queue; if the remote dies for hours, samples will eventually be dropped.

### `instance` vs `job`

`instance` is the host:port being scraped. `job` is the logical job name from your scrape config. One job has many instances. Don't confuse them in queries.

### Process exit doesn't equal target down

If a process exits cleanly, Prometheus sees `up=0` on the next scrape attempt. But if the process is wedged and still serving HTTP but not actually doing useful work, `up=1` is misleading. Combine `up` with health checks (e.g., a custom `service_health` metric the app maintains).

### `histogram_quantile()` on a single bucket

Doesn't work — quantiles need the cumulative distribution. You need every bucket of the histogram, not just one.

### Native histograms vs classic histograms

Native histograms (Prom 2.40 preview, 3.0 GA) have exponential buckets that adjust automatically. Classic histograms (the ones with `_bucket{le=...}` series) require you to pre-pick boundaries. Both can coexist. New code should consider native histograms; existing dashboards still mostly use classic.

### `sum_over_time()` is not the same as `sum()`

`sum_over_time()` sums one series across time. `sum()` sums many series at one moment. Different axes. If you want the total events over the last hour: `sum(increase(events_total[1h]))` (sum the per-series increase). If you want the average value of a gauge over the last hour for one series: `avg_over_time(my_gauge[1h])`.

### `for:` versus `interval:`

`for:` (in alerting rules) is "how long the condition must hold before firing." `interval:` (in rule groups) is "how often Prometheus evaluates the rule." A rule with `interval: 1m` and `for: 5m` is evaluated once a minute and fires when it's been true for five evaluations.

### `evaluation_interval` versus `scrape_interval`

`scrape_interval` is how often Prometheus scrapes targets. `evaluation_interval` is how often it runs recording and alerting rules. They are configured separately. `evaluation_interval` defaults to the same as `scrape_interval` if not set.

## Vocabulary

| Term | Plain English |
| --- | --- |
| Prometheus | The monitoring server that scrapes targets and stores time series. |
| Time series | A unique combination of metric name + labels, with a list of timestamped values. |
| Sample | One (timestamp, value) pair belonging to a series. |
| Metric | A measurable thing, like "requests per second" or "memory usage". |
| Metric name | The base name of a metric, like `http_requests_total`. |
| Label | A key=value tag attached to a metric, like `method="GET"`. |
| Label name | The key part of a label, like `method`. |
| Label value | The value part of a label, like `GET`. |
| Cardinality | The number of unique time series. |
| Cardinality explosion | When labels create millions of series and Prometheus runs out of memory. |
| TSDB | Time Series Database. The storage engine that holds samples on disk. |
| Block | A 2-hour chunk of TSDB data, eventually compacted into longer blocks. |
| WAL | Write-Ahead Log. Where Prometheus records samples before flushing to blocks. |
| Compaction | Merging multiple short blocks into one longer block. |
| Retention | How long Prometheus keeps data before deleting it. |
| Scrape | The act of fetching `/metrics` from a target. |
| Scrape interval | How often Prometheus scrapes (default 15s). |
| Scrape duration | How long one scrape took. |
| Scrape timeout | Maximum time Prometheus will wait for a scrape. |
| `/metrics` | The HTTP endpoint where targets expose their metrics. |
| Exposition format | The text format that `/metrics` uses (`name{labels} value`). |
| OpenMetrics | A stricter superset of the exposition format, standardized. |
| Target | A single thing being scraped (host:port + path). |
| Job | A logical group of targets, named in scrape_configs. |
| Instance | The specific host:port of one target, automatically labeled. |
| `up` metric | Auto-generated metric: 1 if scrape succeeded, 0 if not. |
| Pull model | Prometheus reaches out to scrape targets. |
| Push model | Targets send metrics to a central server (NOT how Prometheus works for normal cases). |
| Counter | A metric that only goes up (or resets on restart). |
| Gauge | A metric that can go up and down. |
| Histogram | A distribution metric with cumulative buckets. |
| Summary | A distribution metric with pre-computed quantiles in the application. |
| Native histogram | A newer histogram with exponential auto-scaling buckets (Prom 3.0). |
| Classic histogram | The old-style histogram with `_bucket{le=...}` series. |
| Bucket | One slice of a histogram, counting observations up to a boundary. |
| `le` label | "Less than or equal to" — the boundary label on a histogram bucket. |
| `+Inf` bucket | The bucket with no upper limit; counts every observation. |
| `_total` suffix | Convention for counter metric names. |
| `_sum` suffix | The total of all observations on a histogram or summary. |
| `_count` suffix | The total number of observations on a histogram or summary. |
| Exemplar | A specific observed value (often with a trace ID) attached to a bucket. |
| `quantile` label | The label on a summary that says which percentile (0.5, 0.9, 0.99...). |
| Service Discovery (SD) | A mechanism for Prometheus to find targets dynamically. |
| `static_configs` | A hand-written list of targets in the scrape config. |
| `file_sd_configs` | SD that reads targets from a JSON/YAML file on disk. |
| `kubernetes_sd_configs` | SD using the Kubernetes API. |
| `consul_sd_configs` | SD using HashiCorp Consul. |
| `ec2_sd_configs` | SD using AWS EC2 instance lists. |
| `dns_sd_configs` | SD that resolves DNS records. |
| `http_sd_configs` | SD that polls a custom HTTP endpoint for target lists. |
| Relabeling | Rewriting labels (and target addresses) before storing. |
| `relabel_configs` | Relabeling that runs before scraping. Decides what to scrape and how to label it. |
| `metric_relabel_configs` | Relabeling that runs after scraping on the returned metrics. |
| `__address__` | Special meta-label: the network address being scraped. |
| `__metrics_path__` | Special meta-label: the URL path being scraped. |
| `__scheme__` | Special meta-label: http or https. |
| `__name__` | Special label: the metric name (label-form of name). |
| `__meta_*` | Temporary discovery metadata labels (dropped unless promoted). |
| `keep` action | Drop targets that don't match. |
| `drop` action | Drop targets that do match. |
| `replace` action | Rewrite a label using a regex. |
| `labelmap` action | Bulk-copy labels matching a regex. |
| `hashmod` action | Hash a label and modulo for sharding. |
| PromQL | Prometheus's query language. |
| Instant vector | One sample per series at one moment. |
| Range vector | A range of samples per series over a window. |
| Selector | A query that picks series by metric name and labels. |
| Range selector | A selector that returns a range vector, e.g. `metric[5m]`. |
| Offset modifier | Look back in time, e.g. `metric offset 1h`. |
| `@` modifier | Pin a query to a specific Unix timestamp. |
| `rate()` | Per-second average rate over a window, counter-aware. |
| `irate()` | Per-second rate from the last two samples in a window. |
| `increase()` | Total increase over a window, counter-aware. |
| `delta()` | Total change of a gauge over a window. NOT counter-aware. |
| `idelta()` | Last-two-samples change of a gauge. |
| `*_over_time` functions | `avg_over_time`, `max_over_time`, etc. — reduce time, not labels. |
| `histogram_quantile()` | Compute a quantile from histogram buckets. |
| `quantile()` | Compute a quantile across series at one moment. |
| `sum()`, `avg()`, etc. | Aggregation operators that reduce labels. |
| `by` clause | Keep only the listed labels after aggregation. |
| `without` clause | Drop the listed labels after aggregation. |
| `topk()` / `bottomk()` | Top-k / bottom-k series by value. |
| `count()` aggregator | Count of series, not values. |
| Subquery | A range vector built from a query result, syntax `<query>[<range>:<step>]`. |
| `on()` matching | When binary-operating, match only on these labels. |
| `ignoring()` matching | Match on everything except these labels. |
| `group_left` | One-to-many matching where left side has duplicate labels. |
| `group_right` | Many-to-one matching where right side has duplicate labels. |
| `vector(s)` | Convert a scalar to a constant instant vector. |
| Recording rule | A pre-computed query result stored as a new metric. |
| Rule group | A group of rules evaluated together at the same interval. |
| Alerting rule | A rule that fires alerts when its expression returns non-empty. |
| `for:` clause | How long the condition must hold before firing. |
| `expr:` clause | The PromQL expression for the rule. |
| `labels:` (in alerts) | Labels added to the alert for routing. |
| `annotations:` (in alerts) | Templated text shown to humans. |
| Pending alert | An alert whose condition is true but `for:` hasn't elapsed yet. |
| Firing alert | An alert that has met `for:` and is being sent to Alertmanager. |
| Resolved alert | A previously-firing alert that has stopped firing. |
| Alertmanager | The server that handles routing, grouping, silencing of alerts. |
| Receiver | A target for alerts: PagerDuty, Slack, email, webhook, etc. |
| Routing tree | The configuration that decides which receiver each alert goes to. |
| `group_by` | Alertmanager: which labels to bundle alerts on. |
| `group_wait` | Alertmanager: delay before sending the first batch of a group. |
| `group_interval` | Alertmanager: delay between batches in an existing group. |
| `repeat_interval` | Alertmanager: how often to re-notify if still firing. |
| Inhibition | Alertmanager: one alert suppresses another. |
| Silence | Alertmanager: manual mute for a label set, time-bounded. |
| `amtool` | The Alertmanager CLI. |
| `promtool` | The Prometheus CLI. |
| Exporter | A program that translates non-Prometheus metrics into `/metrics` form. |
| `node_exporter` | Host-level metrics exporter for Linux/Unix. |
| `windows_exporter` | Same idea for Windows. |
| `blackbox_exporter` | Probes external endpoints (HTTP, ICMP, DNS, TCP). |
| `mysqld_exporter` | MySQL/MariaDB exporter. |
| `postgres_exporter` | PostgreSQL exporter. |
| `snmp_exporter` | Generic SNMP-to-Prometheus translator. |
| `jmx_exporter` | Java JMX exporter (Java agent). |
| `cadvisor` | Per-container metrics. |
| `kube-state-metrics` | Kubernetes API object state as metrics. |
| Pushgateway | Server that receives pushed metrics from short-lived jobs. |
| Federation | One Prometheus scraping `/federate` from another Prometheus. |
| `/federate` | Prometheus endpoint that returns matched series for federation. |
| `honor_labels` | Scrape config: whether to keep target labels over Prometheus's. |
| `remote_write` | Protocol for streaming samples to a remote receiver. |
| `remote_read` | Protocol for querying a remote source as if it were local. |
| Remote Write 2.0 | Newer protocol with metadata, native histograms, multi-tenancy (Prom 3.0). |
| Mimir | Grafana Labs's horizontally-scalable Prometheus storage backend. |
| Thanos | Sidecar-based Prometheus long-term storage with S3 backing. |
| Cortex | Grandfather of Mimir; multi-tenant Prometheus backend. |
| VictoriaMetrics | Independent TSDB compatible with Prometheus protocols. |
| Agent mode | Prometheus mode with no local storage, only scrape and remote_write. |
| `prometheus.yml` | Default config filename. |
| `--config.file` | CLI flag pointing at the config file. |
| `--storage.tsdb.path` | CLI flag for the data directory. |
| `--storage.tsdb.retention.time` | CLI flag for retention duration. |
| `--web.enable-lifecycle` | CLI flag to allow `/-/reload` endpoint. |
| `--web.enable-admin-api` | CLI flag to allow admin operations like deleting series. |
| Hot reload | Reloading config without restarting (`/-/reload` POST). |
| Snappy | The compression algorithm used in remote_write. |
| Protobuf | The binary protocol format used in remote_write. |
| OpenMetrics | The standardized exposition format spec. |
| Recording-rule colon convention | Recording-rule output names use colons like `aggregation:metric:operation`. |
| HTTP API v1 | Prometheus's REST query API at `/api/v1/...`. |
| WAL replay | On restart, Prometheus replays the WAL to reconstruct in-memory state. |
| Out-of-order ingestion | Newer feature allowing samples with timestamps before the latest. |
| Stale marker | A special NaN value indicating a series stopped being exposed. |
| `staleness_delta` | How long a sample is considered "current" (default 5 min). |
| Scrape sample limit | Per-target hard cap on number of samples per scrape. |
| `sample_limit` | The config field for that cap. |
| `query.max-samples` | Server-wide cap on samples a query can load. |
| `query.timeout` | Maximum query duration. |
| `query.lookback-delta` | How far back instant queries look (default 5m). |

## Try This

Pick a Linux machine you have. Install Prometheus and node_exporter. Watch your own laptop's vitals.

```bash
# 1. Install node_exporter
$ wget https://github.com/prometheus/node_exporter/releases/download/v1.8.2/node_exporter-1.8.2.linux-amd64.tar.gz
$ tar xvzf node_exporter-1.8.2.linux-amd64.tar.gz
$ ./node_exporter-1.8.2.linux-amd64/node_exporter &

# 2. See its metrics raw
$ curl -s http://localhost:9100/metrics | head -30
$ curl -s http://localhost:9100/metrics | wc -l

# 3. Install Prometheus
$ wget https://github.com/prometheus/prometheus/releases/download/v3.0.1/prometheus-3.0.1.linux-amd64.tar.gz
$ tar xvzf prometheus-3.0.1.linux-amd64.tar.gz
$ cd prometheus-3.0.1.linux-amd64

# 4. Make a config
$ cat > prometheus.yml <<EOF
global:
  scrape_interval: 15s
scrape_configs:
  - job_name: 'node'
    static_configs:
      - targets: ['localhost:9100']
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
EOF

# 5. Run Prometheus
$ ./prometheus --config.file=prometheus.yml &

# 6. Wait 30s, then query
$ promtool query instant http://localhost:9090 'up'
$ promtool query instant http://localhost:9090 'rate(node_cpu_seconds_total{mode="idle"}[1m])'
$ promtool query instant http://localhost:9090 'node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes'

# 7. Cause some activity, then look at it later
$ for i in $(seq 1 100); do curl -s http://localhost:9090/metrics > /dev/null; done

# 8. See your own scrapes counted
$ promtool query instant http://localhost:9090 'rate(prometheus_http_requests_total[1m])'
```

When you're done, kill both processes (`pkill node_exporter`, `pkill prometheus`).

Variations to try:
- Add a `recording_rules.yml` and watch a new metric appear.
- Add an alerting rule that fires when CPU idle is below 10%, then run `yes > /dev/null` to spike CPU.
- Run `blackbox_exporter` and probe `https://example.com`. Watch `probe_success`.
- Run `pushgateway` and push a manual metric. See it appear in your queries.

## Where to Go Next

You now know what Prometheus is, how it scrapes, what it stores, what PromQL is, what alerts are, and how Alertmanager routes them. The natural next stops:

- **[grafana](#)** — the dashboard tool everyone pairs with Prometheus. Grafana queries Prometheus, draws graphs.
- **[snmp-exporter](#)** — for monitoring routers, switches, UPSs, anything that speaks SNMP.
- **[netflow-ipfix](#)** — for traffic flow telemetry, complementary to metric-based monitoring.
- **[sflow](#)** — sampled flow telemetry, similar concept to NetFlow.
- **[model-driven-telemetry](#)** — gNMI/streaming telemetry, the pull-or-push successor for network gear.
- **[ip-sla](#)** — Cisco's synthetic probing, often combined with SNMP exporter.
- **[snmp](#)** — protocol-level SNMP if you want to understand what `snmp_exporter` is doing under the hood.
- **[kubernetes-eli5](#)** — the system that gave Prometheus most of its modern relevance.
- **[docker-eli5](#)** — the unit of deployment most Prometheus targets live in.
- **[tcp-eli5](#)** — the protocol every scrape rides on.

Then, when you're ready for advanced:

- Deeper PromQL: subqueries, `<aggr>_over_time`, `predict_linear()`, vector matching tricks.
- Long-term storage: pick one of Mimir / Thanos / VictoriaMetrics and learn it.
- Mixins: pre-built dashboard and alert bundles for common services.
- The Prometheus operator: declarative scrape configs as Kubernetes CRDs.
- SLOs and burn-rate alerts: the Google SRE pattern for alerting on long-term reliability.

## See Also

- [monitoring/prometheus](#) — the cheat-sheet version of this with denser commands.
- [monitoring/grafana](#)
- [monitoring/snmp-exporter](#)
- [monitoring/netflow-ipfix](#)
- [monitoring/sflow](#)
- [monitoring/model-driven-telemetry](#)
- [monitoring/ip-sla](#)
- [networking/snmp](#)
- [ramp-up/linux-kernel-eli5](#)
- [ramp-up/kubernetes-eli5](#)
- [ramp-up/tcp-eli5](#)
- [ramp-up/docker-eli5](#)

## References

- Prometheus official documentation — <https://prometheus.io/docs/>
- "Prometheus: Up & Running" by Brian Brazil (O'Reilly, 2nd ed. 2024) — the canonical book.
- Brian Brazil's blog at Robust Perception — <https://www.robustperception.io/blog>
- The original SoundCloud blog post that introduced Prometheus — "Prometheus: Monitoring at SoundCloud" (Matthias Stueber, Björn Rabenstein, 2015).
- OpenMetrics specification — <https://openmetrics.io/>
- "Site Reliability Engineering" (the SRE book), Chapters 6 ("Monitoring Distributed Systems") and 10 ("Practical Alerting"), Google.
- "Implementing Service Level Objectives" by Alex Hidalgo (O'Reilly, 2020) — for the alerting-on-SLOs pattern.
- Prometheus release notes — <https://github.com/prometheus/prometheus/blob/main/CHANGELOG.md>
- Version history highlights:
  - **2.0** (Nov 2017) — TSDB rewrite, dramatically smaller and faster storage.
  - **2.30** (Sep 2021) — agent mode for scrape-and-forward deployments.
  - **2.40** (Oct 2022) — native histograms preview.
  - **2.45** (Jun 2023) — first LTS release; remote_write 2.0 RC.
  - **3.0** (Nov 2024) — native histograms GA, UTF-8 label names, remote_write 2.0 GA, OTLP ingestion.
