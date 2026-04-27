# OpenTelemetry — ELI5

> OpenTelemetry is a universal language for "what happened" inside your software, plus a universal pipe to ship it anywhere.

## Prerequisites

You should be comfortable with the idea that programs run on computers and that those programs sometimes do things you would like to know about. If you have read **ramp-up/linux-kernel-eli5**, **ramp-up/docker-eli5**, **ramp-up/kubernetes-eli5**, **ramp-up/prometheus-eli5**, and **ramp-up/grafana-eli5** you already have every concept you need. If you have not, do not worry: every weird word in this sheet is in the **Vocabulary** table near the bottom, and every command is paste-and-runnable.

If a line in a code block starts with `$`, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## Plain English

### Imagine your software is a city

Pretend you are the mayor of a city. The city has thousands of buildings. Each building is a service, like a coffee shop, a bakery, a bank, a post office. People walk between buildings doing things. Somebody buys coffee at the coffee shop and the coffee shop walks over to the bakery to get the croissant. The bakery walks over to the bank to deposit the money. The bank walks over to the post office to mail a receipt.

You, the mayor, want to know **what is happening in your city.** Are the bakeries fast or slow? Is the bank running out of money? Did the post office crash? Which coffee shop made somebody wait the longest this morning?

To answer those questions, you would need three different things:

- **Numbers that go up and down over time.** "How many cups of coffee did we sell this hour?" "How many people are in the bank right now?" "What is the longest wait time today?" Those are **metrics.**
- **Diaries kept by every building.** "9:14am, somebody bought a latte." "9:15am, the espresso machine made a weird noise." "9:16am, we ran out of oat milk." Those are **logs.**
- **A complete map of one customer's journey.** "Sarah walked into the coffee shop at 9:14, ordered, the shop called the bakery for a croissant, the bakery called the bank to take payment, the bank called the post office for a receipt, and Sarah walked out at 9:18 with everything." Those are **traces.**

Now imagine that every building in your city is a different shape. The coffee shop tracks things in feet. The bakery tracks things in meters. The bank tracks things in cubits. The post office uses smoke signals. The mayor's office cannot make sense of any of it. Every time you ask a question you have to learn a new dialect.

That is what software was like before OpenTelemetry. Every programming language and every tool wrote down "what happened" in its own incompatible format. Datadog had its dialect. New Relic had its dialect. Jaeger had its dialect. Zipkin had its dialect. Prometheus had its dialect. If you wanted to switch backends, you had to rip every line of instrumentation out of every service and re-write it.

OpenTelemetry is the city saying: **everybody speaks the same language now.** One standard for metrics. One standard for logs. One standard for traces. Every building speaks it. Every tool that wants to read it can read it. And there is one universal **post office**, called the **OTel Collector**, that picks up the language from every building and ships it to whichever wall-of-screens the mayor wants: Grafana, Datadog, Honeycomb, New Relic, Splunk, anything.

That is OpenTelemetry. One language, many backends. One library to instrument, many places to send.

### A second picture: OpenTelemetry as electricity

Think of electricity. Before electricity was standardized, every appliance had its own little generator. Your toaster came with a toaster-generator. Your fridge came with a fridge-generator. They were not compatible. If you wanted to upgrade your fridge you had to throw out the old fridge-generator and buy a new one with the new fridge.

Then somebody invented the wall socket. The wall socket says: **all power comes out of here in the same shape.** Toasters plug in. Fridges plug in. Phones plug in. Lamps plug in. The appliance does not care where the electricity comes from. It just plugs into the socket. The socket does not care what the appliance is. It just provides the same shape of electricity to anybody who plugs in.

OpenTelemetry is the wall socket of telemetry. Your application plugs into the OTel SDK. The OTel SDK provides telemetry data in a standard shape. That data flows through the OTel Collector. The Collector ships it to whatever backend you want. If you change backends, you change the Collector config. You do not change a single line of application code.

That is the killer feature. **One instrumentation, switch backends without re-instrumenting.**

### A third picture: the megaphone and the post office

Your application is a person standing on a street corner with things to say. The OTel SDK is the megaphone you hand them. The megaphone takes whatever they want to shout — "I just served a request in 47 milliseconds!" "I just hit an error!" "Customer 73 spent 14 seconds in checkout!" — and turns it into a standardized shape called **OTLP** (OpenTelemetry Protocol).

The OTel Collector is the post office down the street. It listens for OTLP shouts coming from every megaphone in the neighborhood. It collects them. It batches them. It can rewrite labels. It can filter out the boring ones. It can split them into different mail trucks: metrics go to Prometheus, logs go to Loki, traces go to Tempo, the high-importance ones also go to Datadog as a backup.

The megaphone is in your app. The post office is a separate process. You instrument once. You configure the post office however you like.

## What Even Is OpenTelemetry

OpenTelemetry, often shortened to **OTel**, is a CNCF (Cloud Native Computing Foundation) project that defines:

1. A **specification** — what telemetry data should look like, what it means, what fields it has, how to propagate it across service boundaries.
2. A set of **language SDKs** — libraries you import into your application code (Go, Java, Python, JavaScript, .NET, Ruby, Rust, PHP, Swift, C++, and more) that produce telemetry in the spec's shape.
3. The **OTLP wire format** — a Protobuf-based protocol for shipping telemetry over gRPC (port 4317 by default) or HTTP (port 4318 by default).
4. The **OTel Collector** — a vendor-neutral, single-binary agent and gateway that receives, processes, and exports telemetry.
5. **Semantic conventions** — a registry of standardized attribute names so that "the HTTP method" is always called `http.request.method`, not `method` or `httpmethod` or `HTTPMethod` depending on which library you imported.

OpenTelemetry was formed in 2019 by merging two previous projects, **OpenTracing** and **OpenCensus**, both of which were trying to solve the same problem from different angles. The merger gave the industry a single, blessed standard.

The **OTel Collector** graduated from CNCF in 2024, meaning it is officially considered production-grade, vendor-neutral, and stable. The **language SDKs** graduate per signal: traces stabilized in 2021, metrics in 2023, logs in 2024, and **profiling** is still in development as a fourth signal as of 2026.

You will see different stability statuses for different parts:

- **Stable / GA** — frozen API, safe for production, breaking changes require a major version bump.
- **Beta** — usable but the API may change.
- **Experimental / Alpha** — playground, expect breakage.
- **Development** — design phase, not in releases yet.

The matrix matters because, for example, the Java SDK's metrics module was GA before its logs module was GA. Always check the spec page for the language and signal you are using.

## The Three Signals

OpenTelemetry is built around the idea of **signals.** A signal is a category of telemetry data. There are three stable signals as of 2026, and a fourth signal (profiling) in development.

```
                +----------------------+
                |   Your Application   |
                +----------+-----------+
                           |
              +------------+-------------+
              |            |             |
              v            v             v
         +---------+  +--------+  +-----------+
         | Metrics |  |  Logs  |  |  Traces   |
         +----+----+  +---+----+  +-----+-----+
              |           |             |
              +-----------+-------------+
                          |
                          v OTLP (gRPC :4317 / HTTP :4318)
                  +----------------+
                  | OTel Collector |
                  +-------+--------+
                          |
        +-----------+-----+-----+-----------+
        |           |           |           |
        v           v           v           v
   Prometheus    Loki        Tempo      Datadog / NewRelic / ...
   (metrics)    (logs)      (traces)
```

### Signal 1: Metrics

**Metrics** are numbers that change over time. They are perfect for dashboards, alerts, and understanding aggregate behavior. "How many requests per second?" "How much memory is the heap using?" "How many database connections are open?" "What is the 99th-percentile latency?"

Metrics are produced by **instruments**, which are owned by a **meter**, which comes from a **MeterProvider.**

Common instruments:

- **Counter** — only goes up. "Requests served." "Bytes sent." "Login attempts."
- **UpDownCounter** — goes up and down. "Active connections." "Items in queue." "Concurrent users."
- **Histogram** — distribution of values. "Request latency." "Response size." Histograms record buckets so you can compute percentiles.
- **Gauge** — instantaneous reading. "Current temperature." "Disk free percentage."
- **ObservableCounter** — counter that the SDK polls by calling a callback. Use when you cannot easily increment in code.
- **ObservableUpDownCounter** — like ObservableCounter but bidirectional.
- **ObservableGauge** — gauge that the SDK polls.

Instruments are either **synchronous** (you call `counter.Add(1)` from your code) or **asynchronous** (the SDK calls a callback you registered, on its own schedule). Asynchronous instruments are also called **observable** instruments.

```
Synchronous:
  request_handler:
    counter.Add(1, attrs)         <-- you call this
    histogram.Record(elapsed, attrs)

Asynchronous:
  meter.RegisterCallback(func(observer):
    observer.Observe(memory_in_use())  <-- SDK calls this on a schedule
  )
```

Metrics also have a notion of **temporality.** A metric can be reported as **cumulative** (the running total since process start) or **delta** (the change since the last report). Different backends prefer different temporalities. Prometheus wants cumulative. Datadog often wants delta. The Collector can convert between them via the `cumulativetodelta` and `deltatocumulative` processors.

### Signal 2: Logs

**Logs** are events with structure. A log entry has a timestamp, a severity, a body (the message), and attributes (structured key-value pairs).

OpenTelemetry's logs signal stabilized in 2024. The intent is **not** to replace your existing log libraries — it is to give them a standardized backend. Your code keeps using `slog`, `logrus`, `log4j`, `Serilog`, `winston`, whatever — but you install a **bridge** that takes log records from those libraries and emits them as OTLP log records.

A log record has:

- `timestamp` — when it happened.
- `observed_timestamp` — when the SDK noticed it (might differ from `timestamp` if the bridge replays old logs).
- `severity_number` and `severity_text` — INFO, WARN, ERROR, etc.
- `body` — the human message.
- `attributes` — structured fields.
- `trace_id` and `span_id` — automatically attached if a trace is in scope, so logs link to traces.

This last bit is huge. When a log is emitted inside an active span, the SDK automatically tags it with the trace and span IDs. Click the log in Loki, jump to the trace in Tempo. That is the killer demo.

### Signal 3: Traces

**Traces** are trees of timed operations across one or more services. A trace shows you exactly where the time went on a single user request.

A trace is made of **spans.** Each span has:

- A **trace ID** — 16 bytes, identifies the whole trace.
- A **span ID** — 8 bytes, identifies this one span.
- A **parent span ID** — what called it. The root span has no parent.
- A name — `GET /api/orders`, `db.query`, `kafka.publish`.
- A start time and an end time.
- A **status** — Ok, Error, or Unset.
- A **kind** — Server, Client, Producer, Consumer, or Internal.
- **Attributes** — key/value tags (`http.request.method=GET`, `db.system=postgres`).
- **Events** — timestamped occurrences inside the span (`exception thrown`, `cache miss`).
- **Links** — references to other spans (used in fan-out / fan-in scenarios where one logical operation has multiple parents).

```
Trace tree for: GET /api/orders/42

[Server: GET /api/orders/42                                       ] 180ms
   |
   +-- [Internal: validate request                ] 2ms
   +-- [Client: GET orders-service /orders/42   ] 65ms
   |       |
   |       +-- [Server: GET /orders/42                    ] 60ms
   |              |
   |              +-- [Client: db.query SELECT * FROM... ] 40ms
   |
   +-- [Client: GET inventory /stock?sku=...    ] 80ms
   |       |
   |       +-- [Server: GET /stock                        ] 75ms
   |
   +-- [Internal: render JSON           ] 5ms
```

Each rectangle is a span. The bracket nesting shows parent/child. The numbers are wall-clock duration. You can see at a glance which call was slow.

Traces are produced by **tracers**, which come from a **TracerProvider.** A span lifecycle is `Start → set attributes → record events → set status → End`. After `End`, the span is handed to a **SpanProcessor**, which usually batches it and hands it to a **SpanExporter**, which ships it via OTLP.

## Metrics — Deep Dive

### Meter, MeterProvider, Instruments

The hierarchy mirrors traces:

```
MeterProvider          (one per process, holds resources, views, exporters)
   |
   +-- Meter           (one per "instrumentation library" — e.g. "myapp.payments")
        |
        +-- Counter
        +-- UpDownCounter
        +-- Histogram
        +-- Gauge
        +-- ObservableCounter
        +-- ObservableGauge
        +-- ObservableUpDownCounter
```

You typically configure one MeterProvider at process startup and register exporters and views on it. Then anywhere in your code you do `meter := provider.Meter("payments")` and create instruments off the meter.

### Synchronous vs Asynchronous Instruments

**Synchronous instruments** are updated inline. You call `counter.Add(1, attrs)` in your code wherever the event happens. The SDK records that update against the current time.

**Asynchronous instruments** (also called **observable** instruments) work the other way around: you register a callback once, and the SDK calls your callback on a schedule (driven by the metric reader's collection interval). Inside the callback you call `observer.Observe(value, attrs)` with whatever the current value is. Use these for things you can poll but cannot easily increment, like memory usage, queue depth, or CPU temperature.

```
Counter (sync):              every event in code calls counter.Add(...)
ObservableCounter (async):   SDK calls callback every N seconds, you Observe(total)

UpDownCounter (sync):        you Add(+1) and Add(-1) at relevant code paths
ObservableUpDownCounter:     SDK polls, you Observe(current_size)

Histogram (sync):            you Record(elapsed_ms) at end of operation
                              (no async equivalent — histograms only sync)

Gauge (sync, since 2024):    you Record(temp) when you sample
ObservableGauge (async):     SDK polls, you Observe(current_temp)
```

### Aggregations and Views

A **view** lets you change how a specific instrument is aggregated, renamed, or filtered. You can drop attributes (high cardinality killers), change a histogram's buckets, rename a metric, or drop a metric entirely. Views live on the MeterProvider.

Common aggregations:

- `sum` — the running total (Counter, UpDownCounter).
- `last_value` — the most recent observation (Gauge).
- `explicit_bucket_histogram` — fixed bucket boundaries you choose (Histogram default in many SDKs).
- `exponential_histogram` — adaptive buckets, much better resolution at lower cost.
- `drop` — discard.

## Traces — Deep Dive

### Tracer, TracerProvider, Span

```
TracerProvider               (one per process; holds resources, samplers, processors, exporters)
   |
   +-- Tracer                (one per instrumentation library — e.g. "myapp.http")
        |
        +-- Span             (created via tracer.Start(ctx, "operation"))
             |
             +-- attributes
             +-- events
             +-- links
             +-- status
             +-- end()
```

### SpanContext, Trace ID, Span ID

A **SpanContext** is the opaque ID that travels between services. It contains:

- `trace_id` — 16 bytes, hex-encoded as 32 lowercase chars.
- `span_id` — 8 bytes, hex-encoded as 16 lowercase chars.
- `trace_flags` — 1 byte, mostly used for the "sampled" bit.
- `trace_state` — vendor-specific key/value list.
- `is_remote` — whether this context was extracted from a network request (vs. created locally).

When a service makes an outgoing request, it injects the SpanContext into the request (typically via a `traceparent` header). When a service receives a request, it extracts the SpanContext and uses it as the parent of any new spans it creates. That is how a single trace ID flows from the browser to the API gateway to twelve microservices to a database, stitching them all into one tree.

### Parent-Child Relationships

A new span is either a **root span** (no parent) or a **child span** of some existing span. The parent-child relationship is recorded in the new span's `parent_span_id` field. You set the parent by passing a `Context` that already has a span in it:

```go
ctx, parent := tracer.Start(context.Background(), "outer")
defer parent.End()

childCtx, child := tracer.Start(ctx, "inner")  // ctx carries parent
defer child.End()
```

If you start a span with a `Context` that has no span, you create a root span (and a new trace ID).

### Attributes, Events, Links, Status

- **Attributes** — key/value tags. `http.request.method=GET`, `user.id=42`, `cache.hit=false`. Use semantic-convention names where possible.
- **Events** — timestamped log lines inside the span. `span.AddEvent("cache_miss", attrs)`. Useful for marking interesting moments without creating child spans.
- **Links** — pointers to other spans. Used when one operation logically has multiple causes, like a Kafka consumer processing a batch of messages where each message had its own producer trace.
- **Status** — `Ok`, `Error`, or `Unset`. Set to `Error` when the operation failed. `Unset` is the default — let backends and processors decide what to do.

### Span Kind

The `SpanKind` is a hint to the backend about what role the span plays:

- `INTERNAL` — internal computation, not crossing a process boundary.
- `SERVER` — a server-side handler (you received a request).
- `CLIENT` — a client-side call (you made a request).
- `PRODUCER` — you sent a message to a queue.
- `CONSUMER` — you received a message from a queue.

Most automated dashboards rely on this. Get it right.

## Logs — Deep Dive

The Logs signal stabilized in 2024. The OTel approach to logs is **bridges, not direct usage.** Your code keeps using whatever log library you already use. You install an OTel bridge module that converts those library's records into OTel LogRecords as they are emitted, then ships them via the standard SDK pipeline.

Bridges exist for:

- Java: Log4j2, Logback, JUL.
- Python: stdlib `logging`.
- Go: `slog`, `logr`, `zap`, `logrus`.
- JavaScript: `winston`, `pino`, `bunyan`.
- .NET: `ILogger`.
- Ruby: stdlib `Logger`.

When a log is emitted inside an active span, the bridge automatically attaches the current trace ID and span ID to the LogRecord. That is what lets Grafana jump from a log line to a trace.

## Context Propagation

This is the magic that makes traces work across services. When service A calls service B over HTTP, A injects the active trace context into the request headers; B extracts it from the incoming headers and uses it as the parent of any new spans it creates.

The W3C TraceContext standard defines two headers:

```
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
             ||  |                              |                || trace flags
             ||  |                              |span-id        sampled bit
             ||  |trace-id                      |
             ||version
             ||
```

Format: `version-traceid-spanid-flags`.

```
tracestate: vendor1=value1,vendor2=value2
```

`tracestate` carries vendor-specific or per-system metadata that should propagate alongside the trace, e.g. routing hints or sampling decisions. It is comma-separated, key=value, and ordered with the most recent entry first.

OpenTelemetry uses **propagators** to inject and extract these headers. The default propagator is the W3C TraceContext propagator. Other propagators exist:

- `tracecontext` — W3C standard (default).
- `baggage` — propagates Baggage (see below).
- `b3` / `b3multi` — Zipkin's B3 format (single header or multi-header).
- `jaeger` — Jaeger's `uber-trace-id` header.
- `xray` — AWS X-Ray's `X-Amzn-Trace-Id` header.
- `ottrace` — legacy OpenTracing format.

You usually configure a **composite** propagator that does several at once: `tracecontext,baggage` is the common minimum.

### Baggage

**Baggage** is a separate, parallel context-propagation mechanism. It carries key/value pairs across service boundaries that are NOT automatically attached to spans (you have to opt in). Use it for things like `user.tier=gold` or `feature.flags=darkmode,beta` that you want every service in the chain to see.

```
baggage: userId=12345,featureFlag=darkmode,region=us-east-1
```

Baggage propagates over the wire in the `baggage` HTTP header.

## Resources

A **Resource** describes the entity producing the telemetry. It is a set of attributes attached to every metric, log, and span emitted by the SDK. The most important resource attribute is **`service.name`** — without it, your data is essentially anonymous and most backends will reject it or hide it.

Common resource attributes (from semantic conventions):

- `service.name` — required. "checkout-api", "image-resizer".
- `service.version` — "1.4.2", git sha, build number.
- `service.namespace` — logical grouping, e.g. "payments".
- `service.instance.id` — unique per replica, e.g. pod name or UUID.
- `deployment.environment` — "production", "staging", "dev".
- `host.name`, `host.id`, `host.type`.
- `os.type`, `os.description`.
- `process.pid`, `process.runtime.name`, `process.runtime.version`.
- `k8s.pod.name`, `k8s.namespace.name`, `k8s.deployment.name`, `k8s.node.name`, `k8s.cluster.name`.
- `cloud.provider`, `cloud.region`, `cloud.availability_zone`.
- `container.id`, `container.name`, `container.image.name`.

The SDK has **resource detectors** that auto-populate many of these from the environment. The Collector's `k8sattributes` processor can also enrich telemetry with Kubernetes metadata.

## Semantic Conventions

This is the secret weapon of OpenTelemetry. Without it, every team would call HTTP method `httpMethod`, `http_method`, `method`, `request.method`, `Method`, etc. With it, every team calls it `http.request.method`, and every dashboard, alert, and processor knows what that means.

Semantic conventions cover:

- **HTTP** — `http.request.method`, `http.response.status_code`, `http.route`, `url.path`, `url.full`, `server.address`, `server.port`.
- **Database** — `db.system`, `db.statement`, `db.operation`, `db.name`.
- **Messaging** — `messaging.system`, `messaging.destination.name`, `messaging.operation`.
- **RPC** — `rpc.system`, `rpc.service`, `rpc.method`.
- **Network** — `network.peer.address`, `network.peer.port`, `network.protocol.name`.
- **Errors** — `exception.type`, `exception.message`, `exception.stacktrace`, `exception.escaped`.
- **Code** — `code.function`, `code.namespace`, `code.filepath`, `code.lineno`.
- **Cloud** — `cloud.provider`, `cloud.region`, `cloud.account.id`.
- **Kubernetes** — `k8s.pod.name`, etc.
- **Faas** — `faas.invocation_id`, `faas.cold_start`.

The conventions evolve. The HTTP semantic conventions stabilized at version **1.27** in 2024 — that is when names like `http.method` were officially renamed to `http.request.method`. If you are reading old code or old docs, you will see both. The Collector's `transform` processor with OTTL is great for migrating attribute names mid-flight.

```
Old (pre-1.27)              New (1.27+)
---------------------       ----------------------------
http.method                 http.request.method
http.status_code            http.response.status_code
http.url                    url.full
http.target                 url.path
http.host                   server.address
net.peer.ip                 network.peer.address
net.peer.port               network.peer.port
db.statement (still ok)     db.statement
```

Schema URLs are how OTel tracks which version of semconv a resource or attribute set is using. The SDK stamps a `schema_url` on each Resource and InstrumentationScope so consumers can migrate appropriately.

## SDKs

The OTel SDKs implement the spec for a specific language. They typically include:

- The **API** — the interface your application code uses (`Tracer`, `Span`, `Counter`, etc.).
- The **SDK** — the default implementation of the API (samplers, processors, exporters).
- Default exporters (OTLP at minimum, often plus stdout/console).
- A set of **instrumentation libraries** — separate modules that auto-instrument popular libraries (HTTP clients, web frameworks, database drivers, queues).

Stable language SDKs as of 2026:

- **Go** — `go.opentelemetry.io/otel`. Stable for traces, metrics, logs.
- **Java** — `io.opentelemetry`. Stable for all signals; ships a separate Java agent for zero-code auto-instrumentation.
- **Python** — `opentelemetry-api`, `opentelemetry-sdk`, `opentelemetry-instrumentation-*`. Stable.
- **JavaScript / Node.js** — `@opentelemetry/api`, `@opentelemetry/sdk-node`. Stable for traces and metrics.
- **.NET** — uses `System.Diagnostics.Activity` for traces (which IS the OTel API in .NET) plus `OpenTelemetry.*` packages.
- **Ruby** — `opentelemetry-sdk`. Stable.
- **PHP** — `open-telemetry/sdk`. Stable.
- **Rust** — `opentelemetry`, `opentelemetry-otlp`. Maturing.
- **Swift** — community-driven. Maturing.
- **C++** — `opentelemetry-cpp`. Stable for traces.
- **Erlang/Elixir** — `opentelemetry`, `opentelemetry_api`. Stable.

Distinguishing **API vs SDK** matters. The API is what your application and library code uses. The SDK is what the operator (or main package) wires up at startup. Library authors should depend on the API only — that way the same library works whether the application has wired up an SDK or not. If no SDK is wired up, the API calls become no-ops.

## Auto-Instrumentation

Manual instrumentation means your code calls the OTel API explicitly. Auto-instrumentation means a separate component injects instrumentation into popular libraries automatically, with zero code changes. Different ecosystems do this differently:

- **Java** — the **Java agent** (`opentelemetry-javaagent.jar`). Attach with `-javaagent:opentelemetry-javaagent.jar`. It bytecode-rewrites the JDK and dozens of popular libraries (Spring, Hibernate, JDBC, Kafka, gRPC) at JVM startup.
- **Python** — `opentelemetry-instrument`. Run your app as `opentelemetry-instrument python app.py`. Uses Python's import hook to monkey-patch supported libraries (Flask, Django, requests, psycopg2, etc.).
- **Node.js** — `@opentelemetry/auto-instrumentations-node`. Use the `--require @opentelemetry/auto-instrumentations-node/register` flag, or use environment-variable-based startup with `NODE_OPTIONS`.
- **.NET** — `opentelemetry-dotnet-instrumentation`. CLR profiler-based, similar to Java agent.
- **Ruby** — `opentelemetry-instrumentation-all`. Loads auto-instrumentation for many gems.
- **Go** — Go is statically compiled, so true zero-code agents are harder. The community provides **otel-go-instrumentation**, an eBPF-based auto-instrumenter that attaches to running Go binaries and emits spans without recompilation. Other eBPF projects like **Beyla** (from Grafana) provide language-agnostic auto-instrumentation by tapping kernel-level network calls.
- **eBPF auto-instrumentation** — Beyla, **otel-go-instrumentation**, and similar projects attach eBPF probes to syscalls, libc, or Go runtime symbols to derive spans for HTTP, gRPC, and SQL traffic without modifying application code. Limited to network-observable behavior, but zero overhead and zero code changes.

The OTel Operator on Kubernetes can inject auto-instrumentation into Pods automatically by mutating Pod specs based on annotations. You add `instrumentation.opentelemetry.io/inject-java: "true"` and the Operator wires up the Java agent at Pod start.

## The OTel Collector

The **OTel Collector** is a single Go binary that acts as both an **agent** (next to your app, possibly as a sidecar or DaemonSet) and a **gateway** (a separate cluster-level service). It is the universal post office.

```
                    OTel Collector internals
+-----------------------------------------------------------------+
|                                                                 |
|   Receivers   ->  Processors  ->  Exporters                     |
|                                                                 |
|   otlp        ->  batch        ->  otlp                         |
|   prometheus  ->  memory_limit ->  prometheusremotewrite        |
|   jaeger      ->  k8sattributes->  loki                         |
|   zipkin      ->  attributes    -> tempo                        |
|   hostmetrics ->  filter        -> debug                        |
|   syslog      ->  transform     -> kafka                        |
|   filelog     ->  tail_sampling -> elasticsearch                |
|   kafka       ->  resource      -> datadog                      |
|                                                                 |
|                  Connectors (signal-to-signal)                  |
|                  Extensions (health, zpages, etc.)              |
+-----------------------------------------------------------------+
```

A Collector configuration has five sections:

- **receivers** — where data comes in.
- **processors** — what happens to it on the way through.
- **exporters** — where data goes out.
- **connectors** — bridges between pipelines (treat data from one pipeline as input to another).
- **extensions** — non-pipeline features like health endpoints and zPages.

And then **service.pipelines** wires them up for each signal:

```yaml
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch, k8sattributes]
      exporters: [otlp]
    metrics:
      receivers: [otlp, prometheus]
      processors: [batch]
      exporters: [prometheusremotewrite]
    logs:
      receivers: [otlp, filelog]
      processors: [batch]
      exporters: [loki]
```

Each signal (logs, metrics, traces) has its own pipeline. You can have multiple pipelines per signal (e.g. `traces/sampled` and `traces/unsampled`).

### OTLP Receiver and OTLP Exporter

The OTLP wire format has two transports:

- **OTLP/gRPC** on port **4317** — Protobuf over gRPC. Default for SDK-to-Collector hops.
- **OTLP/HTTP** on port **4318** — Protobuf or JSON over HTTPS. Use when gRPC is awkward (browsers, some proxies).

The Collector ships with an `otlp` receiver that listens on **both** transports by default. That is why you will frequently see two OTel ports exposed: 4317 (gRPC) and 4318 (HTTP). They carry the same payload, just different envelopes.

```
                +------+
       gRPC --->|      |
       :4317    | OTLP |
       HTTP --->|recvr |
       :4318    +------+
```

## Pipelines (Logs / Metrics / Traces)

Each **pipeline** in the Collector handles one signal. A pipeline is defined by a name (the signal type, optionally with a slash-separated suffix for multiple pipelines of the same signal), a list of receivers, processors, and exporters.

```yaml
service:
  pipelines:
    traces/all:           # name: signal/optional-suffix
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp/tempo]

    traces/sampled:
      receivers: [otlp]
      processors: [batch, tail_sampling]
      exporters: [otlp/datadog]
```

A receiver can be referenced by multiple pipelines. So can a processor or an exporter (though processors used in multiple pipelines have **separate state** in each pipeline — that bites people).

## Common Receivers

The Contrib distribution has dozens of receivers. The most common ones:

- `otlp` — OTLP/gRPC and OTLP/HTTP. The default front door.
- `prometheus` — scrapes Prometheus metrics endpoints. Reads `scrape_configs` just like Prometheus does.
- `jaeger` — accepts Jaeger-protocol spans (compact thrift, binary thrift, gRPC, HTTP).
- `zipkin` — accepts Zipkin v1/v2 JSON spans on HTTP.
- `fluentforward` — Fluentd's forward protocol; ingest from `fluentd` / `fluent-bit`.
- `hostmetrics` — scrapes the host's CPU, memory, disk, filesystem, network, load, paging, processes.
- `kafka` — consumes telemetry from a Kafka topic.
- `opencensus` — accepts legacy OpenCensus telemetry.
- `syslog` — receives syslog (RFC5424 / RFC3164) over UDP/TCP/TLS.
- `k8sevents` — watches Kubernetes events via the API server.
- `filelog` — tails files (replacement for fluentd/fluentbit for log collection).
- `dockerstats` — scrapes Docker container stats.
- `redis` — scrapes Redis metrics.
- `mysql`, `postgresql`, `mongodb` — DB metrics receivers.
- `awsxray` — receives AWS X-Ray traces.
- `awsfirehose` — receives Kinesis Firehose payloads.

## Common Processors

Processors transform telemetry on its way through the Collector. The order matters: each processor runs after the previous one. The recommended skeleton for any pipeline is:

```yaml
processors:
  memory_limiter:    # always first; backpressure to receivers if RAM is short
    limit_mib: 800
    spike_limit_mib: 200
    check_interval: 5s

  batch:             # always last (or near-last); send in efficient chunks
    send_batch_size: 8192
    timeout: 200ms
    send_batch_max_size: 10000
```

Common processors:

- `batch` — buffer and send in chunks; reduces network and backend load.
- `memory_limiter` — bounds Collector memory; refuses new data when over limit.
- `attributes` — add, update, delete, hash, extract attributes.
- `resource` — modify resource attributes.
- `transform` — apply OTTL transformations (rename, cast, conditionally set).
- `filter` — drop telemetry matching OTTL conditions.
- `tail_sampling` — sample traces *after* observing the full trace tree (so you can keep the slow ones, the error ones, etc.).
- `probabilistic_sampler` — head-based random sampling at percentage X.
- `k8sattributes` — enrich telemetry with Kubernetes metadata (pod name, namespace, labels, annotations) by querying the kube API.
- `redaction` — drop or hash specified attributes (e.g. PII fields).
- `groupbyattrs` — re-aggregate metrics by changing grouping keys.
- `metricstransform` — rename metrics, add labels, scale values.
- `cumulativetodelta` / `deltatocumulative` — switch temporality.
- `routing` — route based on attribute (now mostly replaced by the `routing` connector).
- `resourcedetection` — discover and inject resource attributes from EC2, GCE, Azure, K8s, etc.

## Common Exporters

Where the data goes out:

- `otlp` — OTLP/gRPC to another OTel-compatible endpoint (another Collector, Tempo, Loki via OTel-compatible mode, Honeycomb, Lightstep, Dynatrace, etc.).
- `otlphttp` — OTLP/HTTP variant.
- `prometheus` — exposes a Prometheus-compatible scrape endpoint on the Collector.
- `prometheusremotewrite` — pushes via Prometheus' remote-write protocol to Mimir, Cortex, VictoriaMetrics, Thanos Receive, AMP, etc.
- `loki` — pushes logs to Grafana Loki.
- `tempo` — pushes traces to Grafana Tempo (essentially OTLP under the hood).
- `jaeger` — pushes spans in Jaeger format (deprecated; prefer OTLP to Jaeger Collector).
- `zipkin` — pushes spans in Zipkin format.
- `debug` — print to stdout. Indispensable while developing.
- `file` — write to a local JSON-lines file.
- `kafka` — publish to a Kafka topic.
- `elasticsearch` — push to Elasticsearch / OpenSearch.
- `datadog` — native Datadog format, includes APM and metrics.
- `newrelic` — New Relic OTLP endpoint.
- `honeycomb` — Honeycomb OTLP endpoint with API key.
- `lightstep` — Lightstep OTLP endpoint.
- `dynatrace` — Dynatrace OTLP endpoint.
- `splunk_hec` — Splunk HTTP Event Collector.
- `awsxray`, `awscloudwatchlogs`, `awsemf` — AWS native exporters.
- `googlecloud` — GCP native (Cloud Monitoring, Cloud Logging, Cloud Trace).

## OTTL — The Transform Language

**OTTL** stands for **OpenTelemetry Transform Language**. It is a small DSL embedded in the Collector for inspecting and modifying telemetry. You write OTTL inside `transform` and `filter` processors, and inside the new generation of OTel routing connectors.

OTTL works on contexts: `resource`, `scope`, `span`, `spanevent`, `metric`, `datapoint`, `log`. Each context exposes the relevant fields and lets you call functions on them.

```yaml
processors:
  transform:
    error_mode: ignore
    trace_statements:
      - context: span
        statements:
          - set(attributes["http.url"], attributes["url.full"])
          - delete_key(attributes, "http.flavor")
          - set(status.code, STATUS_CODE_ERROR) where attributes["http.response.status_code"] >= 500

    log_statements:
      - context: log
        statements:
          - set(severity_text, "ERROR") where severity_number >= 17
          - replace_pattern(body, "password=\\S+", "password=***")
```

OTTL functions include `set`, `delete_key`, `keep_keys`, `truncate_all`, `convert_case`, `replace_pattern`, `convert_sum_to_gauge`, `concat`, `IsMatch`, `IsString`, and many more. The function library is extensive and grows release-over-release.

## Distros vs Contrib

The Collector ships in several flavors:

- **otelcol** (core) — only the most generic components: `otlp` receiver/exporter, `batch`/`memory_limiter` processors, `debug`/`file` exporters. Tiny binary.
- **otelcol-contrib** — core plus everything in the `contrib` repo. Hundreds of components: every vendor exporter, every cloud receiver, every advanced processor. Larger binary, much more capability. This is what most users want.
- **otelcol-k8s** — a flavor optimized for Kubernetes (Helm chart default).
- **Custom distro** — built with `ocb` (OpenTelemetry Collector Builder), you specify exactly which components to include in a manifest YAML and produce your own slim binary.

Vendors also publish their own distros (Splunk, AWS Distro for OTel "ADOT", Sumo Logic, Datadog Agent OTel, etc.), all of which are valid OTel implementations with vendor-tuned defaults and extra components.

## Sampling

You almost never want to keep 100% of traces in production. **Sampling** is how you decide which traces to keep.

### Head Sampling

**Head sampling** means the decision is made at the start of the trace, in the first SDK that creates the root span. The decision propagates via the `sampled` flag in `traceparent`. Every downstream service honors it. The big advantages: cheap, predictable, no buffering. The big disadvantage: you decide before you know whether the trace is interesting.

Common head samplers:

- `AlwaysOnSampler` — keep everything.
- `AlwaysOffSampler` — drop everything (useful for tests).
- `TraceIDRatioBased(0.1)` — keep 10% based on trace ID hash.
- `ParentBased(...)` — defer to the parent's decision if remote, else delegate.

Recommended production default: `ParentBased(TraceIDRatioBased(0.05))` — when this service is the root, sample 5%; otherwise honor the upstream decision.

### Tail Sampling

**Tail sampling** means the decision is made at the *end* of the trace, after you have observed all spans. This requires buffering all spans for some time window, which is why it lives in the Collector's `tail_sampling` processor (not the SDK). It is expensive but powerful: you can keep all errors, all slow traces, all traces from a given user, while dropping the boring ones.

```yaml
processors:
  tail_sampling:
    decision_wait: 30s
    num_traces: 50000
    expected_new_traces_per_sec: 1000
    policies:
      - name: errors
        type: status_code
        status_code: { status_codes: [ERROR] }
      - name: slow
        type: latency
        latency: { threshold_ms: 1000 }
      - name: probabilistic
        type: probabilistic
        probabilistic: { sampling_percentage: 5 }
```

For tail sampling to work correctly across multiple Collector replicas, you need to ensure **all spans of a single trace land on the same Collector replica**. The standard trick is a **load-balancing exporter** in front of the tail-sampling Collectors that hashes by trace ID.

```
                         +------------------+
                         | LB exporter pool |
   apps -> OTLP -> Coll  +------+-----------+
                                |hash(trace_id)
                  +-------------+-------------+
                  |             |             |
                  v             v             v
            Tail-Coll-1   Tail-Coll-2   Tail-Coll-3
```

### Probabilistic vs Parent-Based

Probabilistic samplers select traces at random with a fixed probability. Parent-based samplers defer to the parent's decision. The two compose: `ParentBased(TraceIDRatioBased(p))` says "if my parent already made a decision, follow it; otherwise sample with probability p."

## Cardinality and the Drop-Metric Pattern

**Cardinality** is the number of unique attribute combinations on a metric. Each unique combination becomes a separate time series in the backend. High cardinality is the #1 way to blow up Prometheus, Mimir, or any TSDB.

Bad: `requests_total{user_id="..."}` — one series per user. With a million users, that is a million series. Your TSDB will die.

Better: `requests_total{tier="..."}` — a few series total.

OTel's tools for this:

- **Views** — drop attributes from instruments before they leave the SDK.
- **`attributes` / `resource` processors** — strip attributes in the Collector.
- **`filter` processor with OTTL** — drop entire metrics or data points.
- **`groupbyattrs` processor** — re-aggregate metrics with different grouping.

A common pattern is the **drop pattern**: emit the high-cardinality version to a low-volume backend (logs/traces) and the low-cardinality version to the metrics TSDB.

## Common Errors

Real verbatim errors and what they mean.

```
rpc error: code = Unavailable desc = error reading from server: EOF
```
SDK can't reach the OTLP gRPC endpoint. Either the Collector isn't running, the port is wrong (4317 vs 4318), or there's a TLS mismatch (insecure client to TLS server, or vice versa). Set `OTEL_EXPORTER_OTLP_INSECURE=true` for plaintext.

```
OTLP exporter returned: failed to send to localhost:4317
```
Same family. Check Collector is up, port is right, no firewall in between, container networking allows the connection.

```
trace dropped due to sampling decision
```
Head sampler said no. Expected if you have less-than-100% sampling. Don't be alarmed.

```
resource attribute service.name not configured
```
You forgot to set `OTEL_SERVICE_NAME` or to set `service.name` in the resource. Most backends require it to render anything.

```
pipeline contains dangling reference: receivers: [otlp/foo]
```
Your `service.pipelines` references a receiver/processor/exporter that isn't defined in the corresponding top-level section. Spelling counts.

```
receiver "otlp" already exists
```
You declared two receivers with the same name. Use suffixes: `otlp/internal`, `otlp/external`.

```
metric "requests_total" has both monotonic and cumulative aggregations; cannot reconcile temporality
```
Two different sources are emitting the same metric name with different definitions (e.g. one is a Counter and another is reporting it as a Histogram, or one is delta and one cumulative). Pick one definition and either rename or transform the other.

```
collector lifecycle: shutdown returned context deadline exceeded
```
Collector took too long to drain on shutdown. Tune `service.telemetry.metrics.level` and your batch processor `timeout`. In Kubernetes, set `terminationGracePeriodSeconds` higher than your `batch.timeout`.

```
instrumentation library "myapp.payments" exceeds attribute count limit
```
A span or metric data point has too many attributes (default limit is 128). Either reduce attributes or raise `OTEL_ATTRIBUTE_COUNT_LIMIT`. High counts almost always indicate accidentally putting unbounded values (URLs, IDs) where they don't belong.

```
the export operation cannot complete: resource_exhausted
```
Backend is rate-limiting or full. Lower batch size, add retry/backoff, scale the backend.

```
context deadline exceeded
```
Generic timeout. Increase `timeout` in the OTLP exporter config or chase the underlying network slowness.

## Hands-On

### 1. Install the Collector binary

```
$ curl -fL -o otelcol-contrib.tar.gz \
    https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/v0.111.0/otelcol-contrib_0.111.0_linux_amd64.tar.gz
$ tar -xzf otelcol-contrib.tar.gz
$ ./otelcol-contrib --version
otelcol-contrib version 0.111.0
```

### 2. List components in your distribution

```
$ ./otelcol-contrib components
buildinfo:
  command: otelcol-contrib
  description: OpenTelemetry Collector Contrib
  version: 0.111.0
receivers:
  - name: otlp
  - name: prometheus
  - name: jaeger
  ...
```

### 3. Validate a config without starting

```
$ ./otelcol-contrib validate --config=./collector.yaml
2026-04-27  INFO  config is valid
```

### 4. Start the Collector

```
$ ./otelcol-contrib --config=./collector.yaml
2026-04-27  INFO  Starting otelcol-contrib...   Version=0.111.0
2026-04-27  INFO  Everything is ready. Begin running and processing data.
```

### 5. A minimal collector.yaml

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  batch:
    timeout: 200ms

exporters:
  debug:
    verbosity: detailed

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    metrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
    logs:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
```

### 6. Health check endpoint

```
$ curl http://localhost:13133
{"status":"Server available"}
```

### 7. zPages (live in-process diagnostics)

```
$ curl http://localhost:55679/debug/tracez
$ curl http://localhost:55679/debug/pipelinez
$ curl http://localhost:55679/debug/extensionz
```

### 8. Run a Python app with auto-instrumentation

```
$ pip install opentelemetry-distro opentelemetry-exporter-otlp
$ opentelemetry-bootstrap -a install
$ OTEL_SERVICE_NAME=myservice \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
  OTEL_TRACES_EXPORTER=otlp \
  OTEL_METRICS_EXPORTER=otlp \
  OTEL_LOGS_EXPORTER=otlp \
  opentelemetry-instrument python app.py
```

### 9. Run a Java app with the Java agent

```
$ curl -L -o opentelemetry-javaagent.jar \
    https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar
$ OTEL_SERVICE_NAME=myservice \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
  java -javaagent:./opentelemetry-javaagent.jar -jar app.jar
```

### 10. Run a Node app with auto-instrumentation

```
$ npm install --save @opentelemetry/api \
                    @opentelemetry/auto-instrumentations-node
$ OTEL_SERVICE_NAME=myservice \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \
  node --require @opentelemetry/auto-instrumentations-node/register app.js
```

### 11. Universal env-var-only instrumentation

```
$ OTEL_SERVICE_NAME=myservice \
  OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317 \
  OTEL_RESOURCE_ATTRIBUTES="deployment.environment=staging,service.version=1.4.2" \
  OTEL_TRACES_SAMPLER=parentbased_traceidratio \
  OTEL_TRACES_SAMPLER_ARG=0.05 \
  ./app
```

### 12. Send a test span with otel-cli

```
$ otel-cli span --service test --name "demo" --kind client --duration 250ms
$ otel-cli exec --service test --name "ls-home" -- ls -la $HOME
$ otel-cli span "operation" -- ./real-cmd
```

### 13. Generate load with telemetrygen

```
$ go install github.com/open-telemetry/opentelemetry-collector-contrib/cmd/telemetrygen@latest
$ telemetrygen traces --otlp-endpoint=localhost:4317 --otlp-insecure --duration=30s
$ telemetrygen metrics --otlp-endpoint=localhost:4317 --otlp-insecure --rate=100
$ telemetrygen logs --otlp-endpoint=localhost:4317 --otlp-insecure
```

### 14. Generate load with tracegen

```
$ tracegen --otlp-endpoint=localhost:4317 --otlp-insecure --rate=100 --duration=60s
```

### 15. Generate load with otelgen

```
$ otelgen traces --rate=50 --otel-endpoint=localhost:4317 --insecure --duration=30s
```

### 16. Deploy on Kubernetes with the OTel Operator

```
$ kubectl apply -f https://github.com/open-telemetry/opentelemetry-operator/releases/latest/download/opentelemetry-operator.yaml
$ kubectl apply -f - <<'EOF'
apiVersion: opentelemetry.io/v1beta1
kind: OpenTelemetryCollector
metadata:
  name: otel-collector
spec:
  mode: deployment
  config:
    receivers:
      otlp:
        protocols:
          grpc:
          http:
    exporters:
      debug:
    service:
      pipelines:
        traces:
          receivers: [otlp]
          exporters: [debug]
EOF
```

### 17. Helm install the Collector

```
$ helm repo add open-telemetry https://open-telemetry.github.io/opentelemetry-helm-charts
$ helm repo update
$ helm install otel-collector open-telemetry/opentelemetry-collector \
    --values values.yaml \
    --set mode=deployment
```

### 18. Port-forward and watch a Collector

```
$ kubectl port-forward svc/otel-collector 4317:4317 4318:4318 13133:13133
$ kubectl logs -l app=otel-collector -f | grep -i error
$ kubectl describe pod -l app=otel-collector
```

### 19. Health & status

```
$ curl http://localhost:13133            # health
$ curl http://localhost:8888/metrics     # collector self-metrics
$ curl http://localhost:55679/debug/tracez
```

### 20. Build a custom distro with ocb

```
$ go install go.opentelemetry.io/collector/cmd/builder@latest
$ cat > otel-builder.yaml <<'EOF'
dist:
  name: my-otelcol
  output_path: ./dist
receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.111.0
processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.111.0
exporters:
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.111.0
EOF
$ builder --config=otel-builder.yaml
```

### 20a. Generate a default config

```
$ otelcol-builder generate --config=otel-builder.yaml
```

### 21. Tail and grep for errors

```
$ ./otelcol-contrib --config=collector.yaml 2>&1 | grep -Ei 'error|warn'
```

### 22. Use feature gates

```
$ ./otelcol-contrib --feature-gates=+telemetry.useOtelForInternalMetrics --config=collector.yaml
$ kubectl exec otel-collector-0 -- /otelcol --feature-gates=+telemetry.useOtelForInternalMetrics components
```

### 23. Switch a span to an OTLP/HTTP exporter

```yaml
exporters:
  otlphttp:
    endpoint: https://api.honeycomb.io
    headers:
      x-honeycomb-team: ${env:HONEYCOMB_API_KEY}
```

### 24. Use a connector to derive metrics from spans

```yaml
connectors:
  spanmetrics:
    histogram:
      explicit:
        buckets: [10ms, 50ms, 100ms, 500ms, 1s, 5s]
    dimensions:
      - name: http.request.method
      - name: http.response.status_code
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp/tempo, spanmetrics]
    metrics:
      receivers: [spanmetrics]
      exporters: [prometheusremotewrite]
```

### 25. Drop high-cardinality attributes

```yaml
processors:
  attributes:
    actions:
      - key: user.id
        action: delete
      - key: session.id
        action: hash
      - key: http.url
        action: extract
        pattern: ^https?://[^/]+(?P<path>/[^?]*)
```

### 26. Filter out noisy health-check spans

```yaml
processors:
  filter:
    error_mode: ignore
    traces:
      span:
        - 'attributes["http.route"] == "/healthz"'
        - 'attributes["http.route"] == "/readyz"'
```

### 27. Tail-sample errors and slow traces

```yaml
processors:
  tail_sampling:
    decision_wait: 30s
    policies:
      - name: errors
        type: status_code
        status_code: { status_codes: [ERROR] }
      - name: slow
        type: latency
        latency: { threshold_ms: 1000 }
      - name: rare
        type: probabilistic
        probabilistic: { sampling_percentage: 1 }
```

### 28. K8s metadata enrichment

```yaml
processors:
  k8sattributes:
    auth_type: serviceAccount
    passthrough: false
    extract:
      metadata:
        - k8s.namespace.name
        - k8s.pod.name
        - k8s.pod.uid
        - k8s.deployment.name
        - k8s.node.name
      labels:
        - tag_name: app
          key: app.kubernetes.io/name
```

### 29. Switch metric temporality

```yaml
processors:
  cumulativetodelta: {}
service:
  pipelines:
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, cumulativetodelta, batch]
      exporters: [datadog]
```

### 30. Route based on attribute (routing connector)

```yaml
connectors:
  routing:
    table:
      - statement: route() where attributes["deployment.environment"] == "prod"
        pipelines: [traces/prod]
      - statement: route() where attributes["deployment.environment"] == "staging"
        pipelines: [traces/staging]
service:
  pipelines:
    traces/in:
      receivers: [otlp]
      exporters: [routing]
    traces/prod:
      receivers: [routing]
      exporters: [otlp/datadog]
    traces/staging:
      receivers: [routing]
      exporters: [debug]
```

### 31. Receiver/exporter alternatives

```
$ telegraf --config telegraf.conf      # telegraf with otlp output as alternative

JAEGER_AGENT_HOST=jaeger-agent ./app   # legacy Jaeger envvars (deprecated; OTLP preferred)

opentelemetry-bootstrap -a install     # auto-detect Python instrumentation packages
otel-instrumentation-runtime-node      # Node runtime instrumentation
```

### 32. Run otel-arrow (high-throughput)

```yaml
receivers:
  otelarrow:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
        keepalive:
          server_parameters:
            time: 30s
```

### 33. Get pods running the Operator

```
$ kubectl get pods -l app.kubernetes.io/name=opentelemetry-operator -A
$ kubectl get instrumentations -A
$ kubectl get opentelemetrycollectors -A
```

## Common Confusions

A list of things that bite everyone at least once.

### 1. OTel vs OpenTracing vs OpenCensus

Three projects, one survivor. **OpenTracing** (2016) defined a tracing API but no SDK. **OpenCensus** (2017) defined both API and SDK and added metrics. They overlapped. In 2019 they merged into **OpenTelemetry**. OpenTracing and OpenCensus are deprecated; both have shims that bridge their APIs to OTel for migration.

### 2. OTel SDK vs OTel API

The **API** is the surface your application code calls. It's defined to be cheap and stable. The **SDK** is the implementation that actually does sampling, batching, exporting. Library authors should depend on the API only, never the SDK. Application/main packages bring in the SDK and wire it up.

### 3. Auto-Instrumentation vs Manual Instrumentation

Auto-instrumentation = a separate component bytecode-rewrites or monkey-patches popular libraries to emit telemetry without code changes. Manual = you call the OTel API yourself in your code. Real-world apps use both: auto for the obvious stuff (HTTP, DB, queue), manual for business logic.

### 4. Resource Attributes vs Span Attributes

**Resource attributes** describe the entity producing the telemetry. They are set once per process and attached to every signal. `service.name`, `host.name`, `k8s.pod.name`. **Span attributes** are per-span and describe the operation. `http.request.method`, `db.statement`. Putting `user.id` on the resource means every span pretends to be from one user — wrong. It belongs as a span attribute.

### 5. Counter vs UpDownCounter vs Gauge

A **Counter** only goes up (monotonic). A **UpDownCounter** can go up or down. A **Gauge** is an instantaneous reading. Counters and UpDownCounters track **changes** (deltas added together). Gauges track **values** (snapshots). If you find yourself "setting" a counter, you want a gauge. If you find yourself adding negative numbers to a counter, you want an UpDownCounter.

### 6. Cumulative vs Delta Temporality

Same data, two reporting styles. **Cumulative** reports the running total since process start. **Delta** reports the change since the last report. Prometheus likes cumulative. StatsD-era backends like delta. The Collector can convert with `cumulativetodelta` and `deltatocumulative` processors. Mixing temporalities for the same metric is an error.

### 7. Head Sampling vs Tail Sampling

**Head** — decide at trace start, propagate the decision. Cheap, predictable, but you decide blind. **Tail** — buffer all spans, decide at trace end. Expensive but lets you keep all errors and all slow traces. They compose: head-sample to 50% to control costs, then tail-sample for errors at 100%.

### 8. Why two OTLP receivers (gRPC 4317 and HTTP 4318)?

Same payload, different envelopes. gRPC is preferred for in-data-center hops (streaming, multiplexed, smaller wire format). HTTP is preferred where gRPC is awkward (browsers, CDNs, simple HTTPS proxies, AWS Lambda destinations). The Collector listens on both by default. The SDK speaks one or the other; configure with `OTEL_EXPORTER_OTLP_PROTOCOL=grpc` or `=http/protobuf`.

### 9. Semconv Version Churn

Semantic conventions evolve. The HTTP attributes you saw 2 years ago are not exactly the names today. `http.method` became `http.request.method` in semconv 1.27. Both names will appear in the wild for years. Use OTTL in the Collector to migrate as data flows through.

### 10. What is a Connector?

A **connector** is a Collector component that takes data from one pipeline and feeds it into another. Use cases: `spanmetrics` connector turns spans into metrics; `routing` connector splits a pipeline based on attributes; `count` connector counts items as a metric; `forward` connector wires two pipelines together. Connectors are pipeline-to-pipeline. Receivers are external-to-pipeline. Exporters are pipeline-to-external.

### 11. Pipeline Reference Cycles

If two pipelines feed each other through connectors, you can build infinite loops. The Collector validates the graph and rejects cycles at startup. The error reads `pipeline contains cycle: traces -> traces`. Break the cycle.

### 12. Collector Core vs Contrib Distros

`otelcol` (core) ships with a tiny component set. `otelcol-contrib` ships with the kitchen sink. Most users want contrib. Vendors ship their own distros tuned for their backends. You can also build a custom distro with `ocb`.

### 13. Trace Shows Up But Metrics Don't

Each signal has its own pipeline, exporter, and configuration. If your traces work but metrics don't, check the metrics pipeline specifically: did you wire `OTEL_METRICS_EXPORTER`? Is the metrics receiver and exporter listed in the metrics pipeline? Does the backend support metrics OTLP (some only support traces)?

### 14. Trace Context Across HTTP vs gRPC vs Kafka vs SQS

HTTP and gRPC use the W3C `traceparent`/`tracestate` headers. Kafka uses message headers (the OTel Kafka instrumentation injects/extracts there). SQS uses message attributes (the OTel AWS SDK instrumentation handles it). Each protocol has its own injection point, but the OTel propagator API abstracts them. The thing you must do: install the propagator on both sides, and use a carrier-agnostic propagator (`tracecontext` for HTTP/gRPC headers; protocol-specific carrier for Kafka/SQS).

### 15. Logs Don't Replace Your Logger

OTel's logs signal is a **transport**, not a replacement for your logging library. You keep using `slog`, `logback`, `winston`. You install a bridge that converts those library's records into OTel LogRecords. The killer feature is the auto-correlation between logs and traces.

### 16. Why does `service.name` matter so much?

It is the primary index in nearly every observability backend. Without it, your data is anonymous. The SDK will warn if not set; some backends reject the payload outright. Set it via `OTEL_SERVICE_NAME` or `OTEL_RESOURCE_ATTRIBUTES=service.name=...` at the very minimum.

### 17. SDK is "stable" but my exporter is "alpha"

Each component has its own stability label. The **API** is stable, the **SDK** is stable, but a specific exporter or receiver may still be alpha. Check the README of the component you are using.

### 18. The Operator's Target Allocator

In Kubernetes-scale Prometheus scraping, the **target allocator** is a sidecar to the Operator that knows which Prometheus targets each Collector replica should scrape, balancing the load. It also handles ServiceMonitor and PodMonitor CRs. Without it, every Collector replica would scrape every target — pure waste.

## Vocabulary

| Term | Plain English |
|---|---|
| OpenTelemetry | The CNCF project that defines a vendor-neutral observability standard |
| OTel | Common shorthand for OpenTelemetry |
| CNCF | Cloud Native Computing Foundation; hosts Kubernetes, Prometheus, OTel, and more |
| OTLP | OpenTelemetry Protocol — the wire format for shipping telemetry |
| OTLP/gRPC | OTLP transported over gRPC, default port 4317 |
| OTLP/HTTP | OTLP transported over HTTP/HTTPS, default port 4318 |
| otelcol | The core OTel Collector binary |
| otelcol-contrib | The OTel Collector binary built with the kitchen-sink component set |
| OpenTelemetry Operator | Kubernetes operator that manages Collector and Instrumentation custom resources |
| target allocator | Operator sidecar that distributes Prometheus scrape targets across Collector replicas |
| Collector | The vendor-neutral telemetry agent/gateway binary |
| distribution | A specific build of the Collector with a chosen set of components |
| signal | A category of telemetry data (metrics, logs, traces, profiles) |
| metric | A number that changes over time |
| log | An event with structure (timestamp, severity, body, attributes) |
| trace | A tree of timed operations spanning one or more services |
| telemetry | Data about what your system is doing |
| observability | The practice of inferring system state from telemetry |
| instrumentation | Code that produces telemetry |
| auto-instrumentation | A component that injects instrumentation into popular libraries automatically |
| manual instrumentation | Code where you call the OTel API yourself |
| library instrumentation | Instrumentation packaged separately as plug-in modules per library |
| eBPF auto-instrumentation | Kernel-level auto-instrumentation tapping syscalls and library symbols |
| Beyla | Grafana's eBPF auto-instrumenter |
| otel-go-instrumentation | OTel project's eBPF auto-instrumenter for Go |
| ddtrace | Datadog's tracer (alternative to OTel SDKs) |
| Prometheus instrumentation | Native Prometheus client libraries (alternative to OTel for metrics) |
| Micrometer | JVM metrics facade that can target OTel as a backend |
| API | The interface your application code calls |
| SDK | The implementation that wires the API to exporters |
| propagator | Component that injects/extracts trace context across service boundaries |
| traceparent | W3C standard HTTP header carrying trace ID, span ID, and flags |
| tracestate | W3C standard HTTP header carrying vendor-specific trace metadata |
| W3C TraceContext | The standard for HTTP trace context propagation |
| b3 propagation | Zipkin's trace propagation format |
| Jaeger propagator | Jaeger's `uber-trace-id` propagation format |
| Baggage | A separate context-propagation mechanism for arbitrary key/value pairs |
| span | One timed operation in a trace |
| span context | The opaque ID (trace ID, span ID, flags) that travels between services |
| span ID | 8 bytes uniquely identifying a span |
| trace ID | 16 bytes uniquely identifying a whole trace |
| parent span ID | The ID of the span that called this span |
| root span | A span with no parent — the start of a trace |
| span events | Timestamped log lines inside a span |
| span attributes | Per-span key/value tags |
| span links | References from one span to other spans (often other traces) |
| span status | Ok, Error, or Unset |
| span kind | Server, Client, Producer, Consumer, or Internal |
| tracer | Object that creates spans for one instrumentation library |
| tracer provider | Process-wide factory for tracers |
| sampler | Component that decides whether a span gets recorded |
| AlwaysOnSampler | Keep every span |
| AlwaysOffSampler | Drop every span |
| ParentBased | Sampler that defers to the parent's decision when one exists |
| TraceIDRatioBased | Sampler that selects a percentage of trace IDs |
| head sampling | Decide at trace start; propagate the decision |
| tail sampling | Decide at trace end after observing all spans |
| processor | Collector component that transforms telemetry on the way through |
| span processor | SDK-side component that processes spans before export |
| batch span processor | Buffers spans and exports in batches |
| simple span processor | Exports each span synchronously (only for testing) |
| exporter | Component that ships telemetry to a backend |
| span exporter | Implementation that exports spans |
| OTLP exporter | Exporter that ships via OTLP |
| console exporter | Exporter that prints to stdout (debugging) |
| Jaeger exporter | Deprecated; use OTLP to a Jaeger Collector instead |
| Zipkin exporter | Exporter that ships in Zipkin v2 format |
| meter | Object that creates instruments for one instrumentation library |
| meter provider | Process-wide factory for meters |
| instrument | Object that records metric measurements |
| synchronous instrument | Instrument updated inline by code (Counter, etc.) |
| asynchronous instrument | Instrument updated by SDK callback (Observable*) |
| observable instrument | Synonym for asynchronous instrument |
| Counter | Monotonic instrument; only goes up |
| UpDownCounter | Bidirectional instrument |
| Gauge | Instantaneous-value instrument |
| Histogram | Distribution-recording instrument |
| ObservableCounter | Async monotonic instrument |
| ObservableGauge | Async instantaneous-value instrument |
| ObservableUpDownCounter | Async bidirectional instrument |
| view | Configuration that overrides aggregation for a specific instrument |
| aggregation | How measurements combine into a metric data point |
| sum | Aggregation that adds measurements |
| last_value | Aggregation that keeps only the most recent measurement |
| explicit_bucket_histogram | Histogram with operator-configured bucket boundaries |
| exponential_histogram | Histogram with exponentially-spaced buckets |
| drop | Aggregation that discards measurements |
| temporal aggregation | The choice between cumulative and delta reporting |
| cumulative | Running total since process start |
| delta | Change since the last report |
| monotonic | Only goes up |
| semconv | Semantic conventions; standardized attribute names |
| service.name | Required resource attribute identifying the service |
| service.version | Resource attribute for the running version |
| service.namespace | Resource attribute grouping related services |
| service.instance.id | Resource attribute for one specific replica |
| deployment.environment | Resource attribute for prod/staging/dev |
| host.name | Resource attribute for the machine hostname |
| k8s.pod.name | Resource attribute for the Kubernetes pod name |
| k8s.namespace.name | Resource attribute for the Kubernetes namespace |
| http.request.method | Span attribute for the HTTP method |
| http.response.status_code | Span attribute for the HTTP response status code |
| http.route | Span attribute for the matched URL route pattern |
| db.system | Span attribute identifying the database system |
| db.statement | Span attribute holding the SQL/CQL statement |
| messaging.system | Span attribute identifying the messaging system |
| messaging.destination.name | Span attribute for the queue/topic name |
| network.peer.address | Span attribute for the remote network address |
| exception.type | Event attribute for the exception class |
| exception.message | Event attribute for the exception message |
| exception.stacktrace | Event attribute for the stack trace |
| code.function | Span attribute for the function name |
| code.namespace | Span attribute for the package or module |
| Resource | The set of attributes describing the entity producing telemetry |
| resource attribute | A key/value pair on a Resource |
| schema URL | URL stamping which semconv version a Resource was emitted under |
| schema migration | The process of upgrading attribute names across semconv versions |
| OTel Collector receiver | Collector component that ingests data |
| OTel Collector processor | Collector component that transforms data |
| OTel Collector exporter | Collector component that ships data out |
| OTel Collector extension | Collector component for non-pipeline features (health, zPages) |
| OTel Collector connector | Collector component bridging two pipelines |
| OTel Collector pipeline | Wire-up of receivers, processors, exporters for one signal |
| otlp receiver | Receives OTLP/gRPC and OTLP/HTTP |
| prometheus receiver | Scrapes Prometheus metrics endpoints |
| prometheusremotewrite receiver | Accepts Prometheus remote-write payloads |
| prometheus exporter | Exposes a Prometheus scrape endpoint on the Collector |
| prometheusremotewrite exporter | Pushes via Prometheus remote-write |
| loki exporter | Pushes logs to Grafana Loki |
| tempo exporter | Pushes traces to Grafana Tempo |
| jaeger receiver | Accepts Jaeger-formatted spans |
| zipkin receiver | Accepts Zipkin-formatted spans |
| OTTL | OpenTelemetry Transform Language; small DSL inside transform/filter |
| routing connector | Connector that routes data to other pipelines based on attributes |
| forward connector | Connector that wires one pipeline directly to another |
| count connector | Connector that emits a metric for each item |
| spanmetrics connector | Connector that derives RED metrics from spans |
| batch processor | Buffers and sends in chunks |
| memory_limiter processor | Bounds Collector memory usage |
| k8sattributes processor | Enriches telemetry with Kubernetes metadata |
| transform processor | Applies OTTL transformations |
| attributes processor | Adds, modifies, or deletes attributes |
| filter processor | Drops telemetry matching OTTL conditions |
| tail_sampling processor | Buffers traces and decides which to keep at trace end |
| probabilistic_sampler processor | Random sampling at fixed percentage |
| redaction processor | Drops or hashes specified attributes |
| OTel Operator | Kubernetes operator for Collector and Instrumentation CRs |
| Instrumentation CR | Kubernetes custom resource describing auto-instrumentation injection |
| exporter endpoint | The URL/host:port a Collector exporter ships to |
| OTLP/HTTP json vs protobuf | Two encodings of OTLP/HTTP; protobuf is default and recommended |
| OTLP/gRPC compression | gRPC supports gzip and zstd compression |
| zPages | In-process diagnostics endpoints (tracez, pipelinez, etc.) |

## Version Notes

- **OTel Collector** went **GA** in 2023 and **graduated** from CNCF in 2024 — production-grade, vendor-neutral.
- **OpenTelemetry Java SDK** went **GA** in 2021 (the first SDK to fully stabilize).
- **Traces** signal stable in 2021 across all major SDKs.
- **Metrics** signal stable in 2023 across all major SDKs.
- **Logs SDK** stable in 2024 — bridges for major log libraries followed quickly.
- **Profiling Signal** is a **fourth signal** still in active development as of 2026 (eBPF-driven CPU and memory profiles, OTLP-encoded).
- **Semantic Conventions 1.27** stabilized HTTP, RPC, and DB attribute names in 2024 (`http.method` → `http.request.method`, etc.).
- **OpAMP** (Open Agent Management Protocol) is the standardizing channel for remote-managing fleets of Collectors and SDKs; reaching maturity in 2025–2026.
- **otel-arrow** is a high-throughput OTLP-over-Arrow transport variant that can dramatically reduce CPU and bandwidth for very high-volume telemetry.
- **OTel Operator + target allocator** — the Kubernetes deployment model is now the dominant one; the target allocator removed the "every replica scrapes everything" problem.
- **Native histograms** — OTel exponential histograms map cleanly onto Prometheus native histograms, available in Prometheus 2.40+.

## Try This

1. Stand up an `otelcol-contrib` locally with the minimal `collector.yaml` from step 5 above. Confirm `curl http://localhost:13133` returns `Server available`.
2. Run `telemetrygen traces --otlp-endpoint=localhost:4317 --otlp-insecure --duration=10s`. Watch the `debug` exporter print spans on the Collector's stdout.
3. Run `telemetrygen metrics --otlp-endpoint=localhost:4317 --otlp-insecure --duration=10s`. Now you have metrics flowing too.
4. Add a `prometheus` exporter to your Collector pipeline. Scrape `localhost:8889/metrics` with curl. You will see your metrics in Prometheus format.
5. Auto-instrument a Python "hello world" Flask app with `opentelemetry-instrument`. Hit `/`. Watch a span fly through the Collector to the `debug` exporter.
6. Add a `transform` processor that renames `http.method` to `http.request.method`. Re-run the telemetrygen and confirm the rename.
7. Add a `tail_sampling` processor that keeps only error traces. Generate a mix of OK and ERROR spans (use `otel-cli span --status-code Error`). Confirm only the errors hit the exporter.
8. Spin up a small Tempo container, add an `otlp` exporter targeting Tempo, and view the traces in Grafana.
9. Add a `spanmetrics` connector. See RED metrics derived from your spans without any new instrumentation.
10. Deploy on `kind` with the OTel Operator and an Instrumentation CR. Annotate a Pod with `instrumentation.opentelemetry.io/inject-java: "true"` and watch the Java agent attach automatically.

## Where to Go Next

- **monitoring/opentelemetry** — the dense reference cheatsheet for the same topic.
- **monitoring/grafana**, **monitoring/grafana-tempo** — the Grafana side of the visualization story.
- **monitoring/prometheus** — the OG metrics path; OTel and Prometheus interoperate via `prometheusremotewrite` and `prometheus` exporter.
- **monitoring/sflow**, **monitoring/netflow-ipfix**, **monitoring/model-driven-telemetry** — network telemetry adjacent worlds.
- **ramp-up/grafana-eli5**, **ramp-up/prometheus-eli5** — sister ELI5 sheets in this learning path.
- **ramp-up/kubernetes-eli5**, **ramp-up/docker-eli5** — where most OTel deployments live.
- **ramp-up/tcp-eli5** — to understand the network underneath OTLP.

## See Also

- monitoring/opentelemetry
- monitoring/grafana
- monitoring/grafana-tempo
- monitoring/prometheus
- monitoring/sflow
- monitoring/netflow-ipfix
- monitoring/model-driven-telemetry
- ramp-up/grafana-eli5
- ramp-up/prometheus-eli5
- ramp-up/kubernetes-eli5
- ramp-up/docker-eli5
- ramp-up/linux-kernel-eli5
- ramp-up/tcp-eli5

## References

- opentelemetry.io/docs — the canonical OpenTelemetry documentation site.
- "Cloud Native Observability with OpenTelemetry" by Alex Boten — Packt, 2022 — the most readable book on the subject.
- CNCF OTel Collector docs — github.com/open-telemetry/opentelemetry-collector
- otel-arrow project — github.com/open-telemetry/otel-arrow — high-throughput OTLP-Arrow transport
- Semantic conventions registry — opentelemetry.io/docs/specs/semconv
- OTLP spec — github.com/open-telemetry/opentelemetry-proto
- OTel Operator pattern — github.com/open-telemetry/opentelemetry-operator
