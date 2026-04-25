# OpenTelemetry (Observability Framework)

Vendor-neutral CNCF observability framework providing APIs, SDKs, an OTLP wire protocol, and a pluggable Collector for generating, processing, and exporting traces, metrics, and logs across distributed systems in any language to any backend.

## Setup

OpenTelemetry (OTel) is the merger of OpenTracing and OpenCensus, hosted by the CNCF. The project consists of four pieces that work together: **(1)** language-specific APIs and SDKs that emit telemetry, **(2)** a vendor-neutral wire protocol called OTLP (OpenTelemetry Protocol), **(3)** a standalone Collector binary that receives, processes, and re-exports telemetry, and **(4)** a set of cross-language Semantic Conventions that pin attribute names so dashboards work across services.

```bash
# The four pieces of OpenTelemetry
# 1. SDK         per-language API + implementation
# 2. OTLP        wire format (gRPC :4317, HTTP :4318)
# 3. Collector   standalone process: receivers -> processors -> exporters
# 4. SemConv     cross-language attribute name standards
```

### Three signals — stability cadence

The three signals stabilized at different times. The order of GA matters because it explains why most production stacks have first-class traces, mature metrics, and still-evolving logs.

| Signal | Spec status | API stable | SDK stable | OTLP stable |
|--------|-------------|------------|------------|-------------|
| Traces | Stable (1.0) | 2021-02 | 2021-02 | 2021-04 |
| Metrics | Stable (1.0) | 2022-09 | 2022-10 | 2022-11 |
| Logs | Stable (1.0) | 2023-12 | 2023-12 | 2024-02 |
| Profiling | Development | — | — | — |

### Language SDK status (April 2026 snapshot)

Different languages have different signal maturity. The "stable" column refers to the API/SDK for that signal in that language; a language can be at GA for traces but still beta for logs.

| Language | Traces | Metrics | Logs | Auto-instr |
|----------|--------|---------|------|------------|
| Java | Stable | Stable | Stable | Stable (agent) |
| Python | Stable | Stable | Stable | Stable |
| Go | Stable | Stable | Stable | Beta (eBPF) |
| Node.js / JS | Stable | Stable | Stable | Stable |
| .NET | Stable | Stable | Stable | Stable |
| Ruby | Stable | Stable | Beta | Stable |
| PHP | Stable | Stable | Stable | Stable (ext) |
| Rust | Stable | Stable | Stable | None (manual) |
| Erlang/Elixir | Stable | Stable | Beta | Stable |
| Swift | Beta | Beta | Beta | Beta |
| C++ | Stable | Stable | Stable | None (manual) |

```bash
# Quick install — one of:
# Python:  pip install opentelemetry-distro opentelemetry-exporter-otlp
# Node:    npm install @opentelemetry/sdk-node @opentelemetry/auto-instrumentations-node
# Go:      go get go.opentelemetry.io/otel go.opentelemetry.io/otel/sdk
# Java:    download opentelemetry-javaagent.jar from github releases
# .NET:    dotnet add package OpenTelemetry.Extensions.Hosting
# Rust:    cargo add opentelemetry opentelemetry_sdk opentelemetry-otlp
```

## Three Signals

OpenTelemetry models observability as three first-class signals: **traces**, **metrics**, and **logs**. This is the unified data model that replaces the legacy "three pillars" — three vendor-siloed tools that don't share IDs, schemas, or context propagation.

### The contrast

```bash
# Legacy "three pillars" — three siloed tools
#   Traces    -> Jaeger        (own SDK, own UI)
#   Metrics   -> Prometheus    (own scrape, own PromQL)
#   Logs      -> ELK / Splunk  (own ingest, own query)
#
# Problem: cannot pivot trace -> metrics -> logs by trace_id

# OpenTelemetry — one model, one wire, one ID
#   All signals share trace_id + span_id
#   All signals share resource attributes
#   All signals exit via OTLP
#   Backend rendering is interchangeable
```

### Signal definitions

- **Trace** — a tree of spans representing the execution of a single distributed request, sharing one `trace_id` across services.
- **Metric** — a numerical measurement aggregated over time (Counter, UpDownCounter, Histogram, Gauge), labeled by attributes.
- **Log** — a timestamped record (structured or unstructured) optionally correlated to an active span via `trace_id` + `span_id`.

```bash
# All three signals share Resource attributes:
#   service.name           "payments-api"
#   service.version        "1.4.2"
#   deployment.environment "production"
#   host.name              "payments-7d8f-abc"
#   k8s.pod.name           "payments-7d8f-abc"
#
# This is what enables cross-signal pivoting in the backend.
```

## Concepts — Span, Trace, Context

### Span

A **span** is a unit of work — a function call, a database query, an HTTP request — with a name, start time, end time (duration), structured attributes, events (sub-timestamps), links (to other traces), and a status.

```bash
# Span shape (logical)
# {
#   trace_id:       16-byte hex (32 chars)
#   span_id:        8-byte hex (16 chars)
#   parent_span_id: 8-byte hex (or empty for root)
#   name:           "GET /api/users/:id"
#   kind:           CLIENT | SERVER | PRODUCER | CONSUMER | INTERNAL
#   start_time_unix_nano: 1714000000000000000
#   end_time_unix_nano:   1714000000050000000
#   attributes:     [{key:"http.method", value:"GET"}, ...]
#   events:         [{time, name, attributes}, ...]
#   links:          [{trace_id, span_id, attributes}, ...]
#   status:         {code: OK | ERROR | UNSET, message: "..."}
# }
```

### Trace

A **trace** is the tree of spans sharing one `trace_id`. The root span has no `parent_span_id`. Children point to parents via `parent_span_id`.

```bash
# Trace shape
#   trace_id = abc123...  (16 bytes, 32 hex chars)
#
#   span A (root)        kind=SERVER     trace_id=abc, span_id=001, parent=
#   |- span B            kind=INTERNAL   trace_id=abc, span_id=002, parent=001
#   |  |- span C         kind=CLIENT     trace_id=abc, span_id=003, parent=002
#   |  '- span D         kind=CLIENT     trace_id=abc, span_id=004, parent=002
#   '- span E            kind=INTERNAL   trace_id=abc, span_id=005, parent=001
```

### SpanContext

The **SpanContext** is the immutable serializable identity of a span — it travels across process boundaries.

```bash
# SpanContext (W3C Trace Context binary form)
#   trace_id      16 bytes  (must be globally unique, never all-zero)
#   span_id        8 bytes  (must be unique within trace, never all-zero)
#   trace_flags    1 byte   (bit 0 = sampled; other bits reserved)
#   trace_state   <=512 char ASCII  (vendor-specific propagation, key=value comma list)
#   is_remote      bool     (was this context received from another process?)
```

### W3C Traceparent header

The on-the-wire form of SpanContext. Sent as the HTTP header `traceparent`.

```bash
# Format (single line): 00-<trace_id>-<span_id>-<flags>
#
# Example:
traceparent: 00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01
#            ^^ ^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^ ^^^^^^^^^^^^^^^^ ^^
#            |  trace_id (16 bytes / 32 hex)   span_id (8/16)   flags
#            version (always "00" today)
#
# trace_flags:
#   01 = sampled (downstream MUST record)
#   00 = not sampled (downstream MAY drop)
#
# tracestate: vendor1=value1,vendor2=value2  (max 32 entries, ASCII only)
```

```bash
# Generate from CLI for testing:
TRACE_ID=$(openssl rand -hex 16)
SPAN_ID=$(openssl rand -hex 8)
echo "traceparent: 00-${TRACE_ID}-${SPAN_ID}-01"
```

## Concepts — Resource

A **Resource** is the immutable bundle of attributes that describes the entity producing telemetry — what app, what version, what host, what pod. Every span/metric/log emitted by a process inherits the same Resource. This is the join key that lets backends correlate signals.

### Required and recommended Resource attributes

```bash
# Required (semconv minimum)
service.name              "payments-api"             # MUST be set; falls back to "unknown_service" if missing
service.version           "1.4.2"

# Recommended
service.instance.id       "payments-api-7d8f-abc"    # globally unique
deployment.environment    "production"
host.name                 "node-23.us-east-1.compute"
host.id                   "i-0123456789abcdef0"
host.arch                 "amd64"
os.type                   "linux"
os.description            "Ubuntu 24.04.1 LTS"
process.pid               1234
process.runtime.name      "python"
process.runtime.version   "3.12.4"
container.id              "<docker-container-id>"
container.image.name      "ghcr.io/example/payments"
container.image.tag       "1.4.2"
k8s.namespace.name        "payments"
k8s.pod.name              "payments-7d8f-abc"
k8s.pod.uid               "<uuid>"
k8s.deployment.name       "payments"
k8s.node.name             "node-23"
cloud.provider            "aws"
cloud.region              "us-east-1"
cloud.availability_zone   "us-east-1a"
```

### Resource detectors

```bash
# OTEL_RESOURCE_DETECTORS — comma-separated list of auto-detectors
export OTEL_RESOURCE_DETECTORS="env,host,os,process,container,k8s,aws_ec2,aws_eks"

# Each detector populates a subset of attributes from the runtime:
#   env       — reads OTEL_RESOURCE_ATTRIBUTES
#   host      — host.name, host.arch
#   os        — os.type, os.description
#   process   — process.pid, process.runtime.*
#   container — container.id from /proc/self/cgroup
#   k8s       — k8s.* from K8S_* env + downward API
#   aws_ec2   — host.id, cloud.region from IMDSv2
#   gcp       — cloud.* from metadata.google.internal
#   azure     — cloud.* from azure IMDS
```

## Concepts — Metrics Instruments

OpenTelemetry defines six instrument kinds. Choosing the right kind matters because the SDK's aggregation, the wire encoding, and the backend's query semantics all depend on it.

### Synchronous instruments (recorded inline)

| Instrument | Direction | Aggregation | Use for |
|------------|-----------|-------------|---------|
| `Counter` | monotonic up | Sum | request count, bytes sent, errors |
| `UpDownCounter` | up or down | Sum | active connections, queue depth |
| `Histogram` | distribution | ExplicitBucketHistogram (default) or ExponentialHistogram | request duration, payload size |

### Asynchronous instruments (callback-based)

| Instrument | Direction | Aggregation | Use for |
|------------|-----------|-------------|---------|
| `ObservableCounter` | monotonic up | Sum | total CPU seconds, total bytes read |
| `ObservableUpDownCounter` | up or down | Sum | memory allocations, current goroutines |
| `ObservableGauge` | snapshot | LastValue | temperature, queue size at scrape time |

### Canonical mapping (memorize this)

```bash
# Request count                  -> Counter           (always int64)
# Errors                         -> Counter           (always int64)
# Bytes sent / received          -> Counter           (int64)
# Active connections             -> UpDownCounter     (int64)
# In-flight requests             -> UpDownCounter     (int64)
# Request latency                -> Histogram         (float64 seconds)
# Payload size                   -> Histogram         (int64 bytes)
# Memory used                    -> ObservableGauge   (int64 bytes)
# CPU utilization                -> ObservableGauge   (float64 ratio 0-1)
# Queue depth (sampled)          -> ObservableGauge   (int64)
# Total CPU time consumed        -> ObservableCounter (float64 seconds)
```

### Histogram bucket boundaries

```bash
# Default explicit bucket boundaries (seconds, OTel spec):
#   [0, 5, 10, 25, 50, 75, 100, 250, 500, 750,
#    1000, 2500, 5000, 7500, 10000]    -- in milliseconds for http.server.duration
#
# Override per-view if your latencies are smaller (e.g. RPC < 100ms):
#   [0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10]
```

## SDK — Provider Architecture

Every language SDK follows the same provider-processor-exporter pattern. The Provider is the root object; it owns a Resource and a list of Processors; each Processor wraps an Exporter that ships telemetry off-host.

### Architecture diagram (logical)

```bash
#  user code
#      |  tracer.Start(...) / counter.Add(...) / logger.Emit(...)
#      v
#  +--------------------+
#  |  TracerProvider    |    one per process, holds Resource
#  |  MeterProvider     |
#  |  LoggerProvider    |
#  +---------+----------+
#            |
#            v
#  +--------------------+
#  |  Processor         |    Span: BatchSpanProcessor / SimpleSpanProcessor
#  |  (Reader for       |    Metric: PeriodicExportingMetricReader
#  |   metrics)         |    Log: BatchLogRecordProcessor
#  +---------+----------+
#            |
#            v
#  +--------------------+
#  |  Exporter          |    OTLP-gRPC, OTLP-HTTP, Console, Prometheus, ...
#  +---------+----------+
#            |
#            v
#  network / file / scrape endpoint
```

### Span processors

```bash
# SimpleSpanProcessor    — synchronous; export on every span end. DEBUG ONLY.
# BatchSpanProcessor     — async queue; batches spans; production default.
#   tunables (env or constructor):
#     OTEL_BSP_MAX_QUEUE_SIZE         default 2048
#     OTEL_BSP_MAX_EXPORT_BATCH_SIZE  default 512
#     OTEL_BSP_SCHEDULE_DELAY         default 5000 ms
#     OTEL_BSP_EXPORT_TIMEOUT         default 30000 ms
```

### Metric readers

```bash
# PeriodicExportingMetricReader  — push: exports every N seconds via OTLP.
#     OTEL_METRIC_EXPORT_INTERVAL  default 60000 ms
#     OTEL_METRIC_EXPORT_TIMEOUT   default 30000 ms
# PrometheusReader (or PrometheusExporter) — pull: serves /metrics on :9464.
# ManualReader                   — for tests, exports only when triggered.
```

### Log record processors

```bash
# SimpleLogRecordProcessor       — synchronous; debug only.
# BatchLogRecordProcessor        — production default. Same tunables as BSP.
```

## SDK — Exporters

| Exporter | Signals | Transport | Default port | Status |
|----------|---------|-----------|--------------|--------|
| `OTLP` (gRPC) | T/M/L | gRPC over HTTP/2 | 4317 | Stable, recommended |
| `OTLPHTTP` | T/M/L | HTTP/protobuf or HTTP/JSON | 4318 | Stable |
| `Console` / `stdout` | T/M/L | stderr | — | Debug only |
| `Prometheus` | M | HTTP scrape | 9464 | Stable, pull |
| `PrometheusRemoteWrite` | M | HTTP push | — | Stable, push |
| `Jaeger` | T | Thrift / gRPC | 14250 / 14268 | **DEPRECATED** — use OTLP |
| `Zipkin` | T | HTTP/JSON | 9411 | Stable |
| `OTLP File` | T/M/L | newline-delimited JSON | — | Beta |

```bash
# Modern story: OTLP everywhere.
# Jaeger 1.35+ accepts OTLP natively on :4317/:4318.
# Tempo accepts OTLP natively on :4317/:4318.
# Prometheus 2.47+ accepts OTLP-metric pushes on /api/v1/otlp/v1/metrics.
# Loki accepts OTLP-logs natively (OTEP 2024-09).
#
# Bottom line: use OTLP gRPC for everything, switch only if backend can't.
```

## OTLP Wire Format

OTLP is the OpenTelemetry Protocol. It defines protobuf message schemas and gRPC services for shipping all three signals. Defined in the `opentelemetry-proto` repository.

### Proto file layout

```bash
# opentelemetry/proto/
# +-- common/v1/common.proto                  AnyValue, KeyValue, InstrumentationScope
# +-- resource/v1/resource.proto              Resource (KeyValue list + dropped count)
# +-- trace/v1/trace.proto                    Span, ResourceSpans, ScopeSpans
# +-- metrics/v1/metrics.proto                Metric (Sum, Gauge, Histogram, ExpHistogram, Summary)
# +-- logs/v1/logs.proto                      LogRecord, ResourceLogs, ScopeLogs
# +-- collector/trace/v1/trace_service.proto      ExportTraceServiceRequest/Response
# +-- collector/metrics/v1/metrics_service.proto  ExportMetricsServiceRequest/Response
# +-- collector/logs/v1/logs_service.proto        ExportLogsServiceRequest/Response
```

### gRPC service definitions

```bash
# service TraceService {
#   rpc Export(ExportTraceServiceRequest) returns (ExportTraceServiceResponse) {}
# }
# service MetricsService {
#   rpc Export(ExportMetricsServiceRequest) returns (ExportMetricsServiceResponse) {}
# }
# service LogsService {
#   rpc Export(ExportLogsServiceRequest) returns (ExportLogsServiceResponse) {}
# }

# Default ports:
#   gRPC          4317   (h2 cleartext or h2 over TLS)
#   HTTP/protobuf 4318   (POST /v1/traces, /v1/metrics, /v1/logs)
#   HTTP/JSON     4318   (same paths, Content-Type: application/json)
```

### HTTP encoding

```bash
# HTTP/protobuf endpoint:
POST /v1/traces  HTTP/1.1
Host: collector.example.com:4318
Content-Type: application/x-protobuf
Content-Length: <bytes>
<protobuf-encoded ExportTraceServiceRequest>

# HTTP/JSON endpoint (same path, different content-type):
POST /v1/traces  HTTP/1.1
Content-Type: application/json
{"resourceSpans":[{"resource":{"attributes":[{"key":"service.name","value":{"stringValue":"x"}}]},"scopeSpans":[...]}]}

# gzip compression supported:
Content-Encoding: gzip
```

### gRPC status codes returned by the receiver

```bash
# OK / unset                   — accepted
# INVALID_ARGUMENT             — malformed request, do not retry
# RESOURCE_EXHAUSTED           — backpressure; retry with backoff
# UNAVAILABLE                  — endpoint down; retry
# DEADLINE_EXCEEDED            — timeout; retry
# UNAUTHENTICATED              — wrong API key; do not retry
```

## Semantic Conventions

The Semantic Conventions registry defines the canonical attribute names for every common subsystem. Stick to these names — your dashboards, alerts, and tail-sampling rules become portable across services.

### HTTP

```bash
# server (incoming HTTP request)
http.request.method            "GET" | "POST" | ...        # required
http.response.status_code      200                          # required for response
http.route                     "/api/users/:id"             # parameterized template
url.full                       "https://api.example.com/api/users/42?x=1"
url.scheme                     "https" | "http"
url.path                       "/api/users/42"
url.query                      "x=1"
server.address                 "api.example.com"
server.port                    443
network.protocol.name          "http"
network.protocol.version       "1.1" | "2" | "3"
user_agent.original            "curl/8.5"

# client (outgoing HTTP request) — same names; "client" semantics
```

### Database

```bash
db.system                      "postgresql" | "mysql" | "mongodb" | "redis" | ...
db.name                        "users_prod"
db.user                        "appuser"
db.statement                   "SELECT * FROM users WHERE id = ?"   # use parameterized
db.operation                   "SELECT" | "INSERT" | "UPDATE" | "DELETE"
db.sql.table                   "users"
network.peer.address           "db-primary.internal"
network.peer.port              5432
```

### Messaging

```bash
messaging.system               "kafka" | "rabbitmq" | "sqs" | "pulsar"
messaging.operation            "publish" | "receive" | "process"
messaging.destination.name     "orders.created"
messaging.destination.kind     "queue" | "topic"
messaging.message.id           "0123-abcd"
messaging.kafka.partition      3
messaging.kafka.message.offset 1024
messaging.kafka.consumer.group "order-processor"
```

### RPC (gRPC, Connect, Twirp)

```bash
rpc.system                     "grpc" | "connect_rpc" | "apache_dubbo"
rpc.service                    "payments.v1.PaymentService"
rpc.method                     "Charge"
rpc.grpc.status_code           0   # 0=OK, 1=CANCELLED, ..., 16=UNAUTHENTICATED
```

### Cloud / Faas

```bash
cloud.provider                 "aws" | "gcp" | "azure"
cloud.region                   "us-east-1"
faas.invocation_id             "<uuid>"
faas.trigger                   "http" | "pubsub" | "timer"
```

## SDK — Python

```bash
# Install
pip install opentelemetry-api opentelemetry-sdk opentelemetry-exporter-otlp
pip install opentelemetry-distro                          # auto-instrumentation
opentelemetry-bootstrap --action=install                  # install detected libs

# Manual setup (sketch)
# from opentelemetry import trace
# from opentelemetry.sdk.trace import TracerProvider
# from opentelemetry.sdk.trace.export import BatchSpanProcessor
# from opentelemetry.sdk.resources import Resource, SERVICE_NAME
# from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
#
# resource = Resource.create({SERVICE_NAME: "payments-api"})
# provider = TracerProvider(resource=resource)
# provider.add_span_processor(
#     BatchSpanProcessor(OTLPSpanExporter(endpoint="localhost:4317", insecure=True))
# )
# trace.set_tracer_provider(provider)
#
# tracer = trace.get_tracer(__name__)
# with tracer.start_as_current_span("charge") as span:
#     span.set_attribute("payment.amount", 42.50)
#     span.set_attribute("payment.currency", "USD")
#     try:
#         do_work()
#     except Exception as e:
#         span.record_exception(e)
#         span.set_status(trace.Status(trace.StatusCode.ERROR, str(e)))
#         raise
```

```bash
# Auto-instrumentation — zero code change
pip install opentelemetry-distro opentelemetry-exporter-otlp
opentelemetry-bootstrap --action=install
opentelemetry-instrument \
  --traces_exporter otlp \
  --metrics_exporter otlp \
  --logs_exporter otlp \
  --service_name payments-api \
  --exporter_otlp_endpoint http://localhost:4317 \
  python app.py
```

```bash
# Metrics
# from opentelemetry import metrics
# from opentelemetry.sdk.metrics import MeterProvider
# from opentelemetry.sdk.metrics.export import PeriodicExportingMetricReader
# from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
#
# reader = PeriodicExportingMetricReader(OTLPMetricExporter(endpoint="localhost:4317", insecure=True))
# metrics.set_meter_provider(MeterProvider(metric_readers=[reader]))
# meter = metrics.get_meter(__name__)
# request_counter = meter.create_counter("http.server.requests", unit="1")
# request_counter.add(1, {"http.request.method": "GET", "http.route": "/api/users/:id"})
```

## SDK — Go

```bash
# Install
go get go.opentelemetry.io/otel \
       go.opentelemetry.io/otel/sdk \
       go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc \
       go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc \
       go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp

# Bootstrap (sketch)
# import (
#   "go.opentelemetry.io/otel"
#   "go.opentelemetry.io/otel/sdk/resource"
#   sdktrace "go.opentelemetry.io/otel/sdk/trace"
#   "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
#   semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
# )
#
# exp, _ := otlptracegrpc.New(ctx,
#     otlptracegrpc.WithEndpoint("localhost:4317"),
#     otlptracegrpc.WithInsecure(),
# )
# res, _ := resource.Merge(resource.Default(), resource.NewWithAttributes(
#     semconv.SchemaURL,
#     semconv.ServiceName("payments-api"),
#     semconv.ServiceVersion("1.4.2"),
# ))
# tp := sdktrace.NewTracerProvider(
#     sdktrace.WithBatcher(exp),
#     sdktrace.WithResource(res),
#     sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
# )
# otel.SetTracerProvider(tp)
# defer tp.Shutdown(ctx)
```

```bash
# Manual span
# tracer := otel.Tracer("payments-api")
# ctx, span := tracer.Start(ctx, "charge",
#     trace.WithSpanKind(trace.SpanKindServer),
#     trace.WithAttributes(attribute.String("payment.currency", "USD")))
# defer span.End()
#
# if err := chargeCard(ctx, amt); err != nil {
#     span.SetStatus(codes.Error, err.Error())
#     span.RecordError(err)
#     return err
# }

# HTTP server middleware
# import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
# handler := otelhttp.NewHandler(mux, "http.server")
# http.ListenAndServe(":8080", handler)

# HTTP client
# client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

# gRPC server
# import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
# srv := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

# Metrics
# meter := otel.Meter("payments-api")
# counter, _ := meter.Int64Counter("http.server.requests")
# counter.Add(ctx, 1, metric.WithAttributes(attribute.String("http.route", "/charge")))
# hist, _ := meter.Float64Histogram("http.server.duration", metric.WithUnit("s"))
# hist.Record(ctx, 0.042, metric.WithAttributes(attribute.String("http.route", "/charge")))
```

## SDK — Java

```bash
# Maven
# <dependency>
#   <groupId>io.opentelemetry</groupId>
#   <artifactId>opentelemetry-api</artifactId>
#   <version>1.40.0</version>
# </dependency>
# <dependency>
#   <groupId>io.opentelemetry</groupId>
#   <artifactId>opentelemetry-sdk</artifactId>
# </dependency>
# <dependency>
#   <groupId>io.opentelemetry</groupId>
#   <artifactId>opentelemetry-exporter-otlp</artifactId>
# </dependency>

# Bootstrap (programmatic)
# OpenTelemetry sdk = OpenTelemetrySdk.builder()
#     .setTracerProvider(SdkTracerProvider.builder()
#         .setResource(Resource.getDefault().merge(Resource.create(
#             Attributes.builder().put("service.name", "payments-api").build())))
#         .addSpanProcessor(BatchSpanProcessor.builder(
#             OtlpGrpcSpanExporter.builder()
#                 .setEndpoint("http://localhost:4317").build()).build())
#         .build())
#     .setPropagators(ContextPropagators.create(W3CTraceContextPropagator.getInstance()))
#     .buildAndRegisterGlobal();
#
# Tracer tracer = sdk.getTracer("payments-api", "1.4.2");
# Span span = tracer.spanBuilder("charge")
#     .setSpanKind(SpanKind.SERVER)
#     .setAttribute("payment.currency", "USD")
#     .startSpan();
# try (Scope s = span.makeCurrent()) {
#     doWork();
# } catch (Throwable t) {
#     span.recordException(t);
#     span.setStatus(StatusCode.ERROR, t.getMessage());
#     throw t;
# } finally {
#     span.end();
# }

# Auto-instrumentation — Java agent JAR (no code changes)
java -javaagent:opentelemetry-javaagent.jar \
  -Dotel.service.name=payments-api \
  -Dotel.exporter.otlp.endpoint=http://localhost:4317 \
  -Dotel.exporter.otlp.protocol=grpc \
  -Dotel.traces.sampler=parentbased_traceidratio \
  -Dotel.traces.sampler.arg=0.1 \
  -jar payments-api.jar
```

## SDK — Node.js

```bash
# Install
npm install --save \
  @opentelemetry/api \
  @opentelemetry/sdk-node \
  @opentelemetry/auto-instrumentations-node \
  @opentelemetry/exporter-trace-otlp-grpc \
  @opentelemetry/exporter-metrics-otlp-grpc \
  @opentelemetry/sdk-metrics

# tracing.js — load BEFORE app code
# const { NodeSDK } = require('@opentelemetry/sdk-node');
# const { OTLPTraceExporter } = require('@opentelemetry/exporter-trace-otlp-grpc');
# const { getNodeAutoInstrumentations } = require('@opentelemetry/auto-instrumentations-node');
# const { Resource } = require('@opentelemetry/resources');
# const { SemanticResourceAttributes } = require('@opentelemetry/semantic-conventions');
#
# const sdk = new NodeSDK({
#   resource: new Resource({
#     [SemanticResourceAttributes.SERVICE_NAME]: 'payments-api',
#     [SemanticResourceAttributes.SERVICE_VERSION]: '1.4.2',
#   }),
#   traceExporter: new OTLPTraceExporter({ url: 'http://localhost:4317' }),
#   instrumentations: [getNodeAutoInstrumentations()],
# });
# sdk.start();

# Run with zero-code instrumentation
node --require ./tracing.js app.js
# or, if installed globally:
node --require @opentelemetry/auto-instrumentations-node/register app.js

# Manual span
# const { trace, context, SpanStatusCode } = require('@opentelemetry/api');
# const tracer = trace.getTracer('payments-api');
# tracer.startActiveSpan('charge', span => {
#   try { doWork(); span.setStatus({ code: SpanStatusCode.OK }); }
#   catch (e) { span.recordException(e); span.setStatus({ code: SpanStatusCode.ERROR, message: e.message }); throw e; }
#   finally { span.end(); }
# });
```

## SDK — Rust

```bash
# Cargo.toml
# [dependencies]
# opentelemetry = { version = "0.27", features = ["trace", "metrics", "logs"] }
# opentelemetry_sdk = { version = "0.27", features = ["rt-tokio"] }
# opentelemetry-otlp = { version = "0.27", features = ["grpc-tonic"] }
# tracing = "0.1"
# tracing-subscriber = "0.3"
# tracing-opentelemetry = "0.28"

# Bootstrap (sketch)
# use opentelemetry::global;
# use opentelemetry_otlp::WithExportConfig;
# use tracing_subscriber::layer::SubscriberExt;
#
# fn init_tracer() {
#     let exporter = opentelemetry_otlp::SpanExporter::builder()
#         .with_tonic()
#         .with_endpoint("http://localhost:4317")
#         .build().unwrap();
#     let provider = opentelemetry_sdk::trace::TracerProvider::builder()
#         .with_batch_exporter(exporter, opentelemetry_sdk::runtime::Tokio)
#         .with_resource(opentelemetry_sdk::Resource::new(vec![
#             opentelemetry::KeyValue::new("service.name", "payments-api"),
#         ]))
#         .build();
#     global::set_tracer_provider(provider.clone());
#
#     let otel_layer = tracing_opentelemetry::layer().with_tracer(provider.tracer("payments-api"));
#     let subscriber = tracing_subscriber::Registry::default()
#         .with(otel_layer)
#         .with(tracing_subscriber::fmt::layer());
#     tracing::subscriber::set_global_default(subscriber).unwrap();
# }

# Use the tracing crate normally; the layer bridges to OTel
# #[tracing::instrument(fields(payment.currency = %currency))]
# async fn charge(amount: f64, currency: &str) -> Result<()> { ... }
```

## SDK — .NET

```bash
# Install (project file or CLI)
dotnet add package OpenTelemetry
dotnet add package OpenTelemetry.Extensions.Hosting
dotnet add package OpenTelemetry.Exporter.OpenTelemetryProtocol
dotnet add package OpenTelemetry.Instrumentation.AspNetCore
dotnet add package OpenTelemetry.Instrumentation.Http

# Program.cs (ASP.NET Core)
# builder.Services.AddOpenTelemetry()
#     .ConfigureResource(r => r.AddService("payments-api", serviceVersion: "1.4.2"))
#     .WithTracing(t => t
#         .AddAspNetCoreInstrumentation()
#         .AddHttpClientInstrumentation()
#         .AddOtlpExporter(o => o.Endpoint = new Uri("http://localhost:4317")))
#     .WithMetrics(m => m
#         .AddAspNetCoreInstrumentation()
#         .AddHttpClientInstrumentation()
#         .AddOtlpExporter(o => o.Endpoint = new Uri("http://localhost:4317")));

# .NET maps OTel Span <-> System.Diagnostics.Activity
# private static readonly ActivitySource Source = new("Payments");
# using var activity = Source.StartActivity("charge", ActivityKind.Server);
# activity?.SetTag("payment.currency", "USD");
# try { /* work */ }
# catch (Exception ex)
# {
#     activity?.SetStatus(ActivityStatusCode.Error, ex.Message);
#     activity?.RecordException(ex);
#     throw;
# }

# Auto-instrumentation (zero-code, alpha->beta)
# Set env vars:
export CORECLR_ENABLE_PROFILING=1
export CORECLR_PROFILER='{918728DD-259F-4A6A-AC2B-B85E1B658318}'
export CORECLR_PROFILER_PATH=/path/to/OpenTelemetry.AutoInstrumentation.Native.so
export DOTNET_ADDITIONAL_DEPS=/path/to/AdditionalDeps
export OTEL_DOTNET_AUTO_HOME=/path/to/OpenTelemetry.AutoInstrumentation
export OTEL_SERVICE_NAME=payments-api
```

## SDK — Other

| Language | Package(s) | Auto-instr | Notes |
|----------|------------|------------|-------|
| Ruby | `opentelemetry-sdk`, `opentelemetry-instrumentation-all` | Stable | `OpenTelemetry::SDK.configure { ... }` |
| PHP | `open-telemetry/sdk`, `open-telemetry/opentelemetry-auto-*` | Stable (extension) | `pecl install opentelemetry` then OTEL_PHP_AUTOLOAD_ENABLED=true |
| Erlang/Elixir | `opentelemetry`, `opentelemetry_api` (hex) | Stable | `:opentelemetry` app supervisor |
| Swift | `OpenTelemetry-Swift` | Beta | `swift package add` |
| C++ | `opentelemetry-cpp` | Manual only | CMake-built static/shared libs |
| Lua/OpenResty | `lua-resty-opentelemetry` (community) | Beta | nginx-only |

```bash
# Ruby quickstart
# require 'opentelemetry/sdk'
# require 'opentelemetry/instrumentation/all'
# OpenTelemetry::SDK.configure do |c|
#   c.service_name = 'payments-api'
#   c.use_all
# end

# PHP — zero-code via the extension
# php -d extension=otel_instrumentation \
#     -d otel.service.name=payments-api \
#     -d otel.exporter.otlp.endpoint=http://localhost:4317 \
#     app.php

# Elixir
# config :opentelemetry,
#   resource: [service: %{name: "payments-api"}],
#   span_processor: :batch,
#   traces_exporter: :otlp
```

## Manual Instrumentation Patterns

The canonical idiom: **wrap every public function with a span**, set attributes for inputs you care about, set status on error, propagate context across async boundaries.

### Wrapper pattern per language

```bash
# Python
# def with_span(name):
#     def deco(fn):
#         def wrap(*a, **kw):
#             with tracer.start_as_current_span(name) as span:
#                 try: return fn(*a, **kw)
#                 except Exception as e:
#                     span.record_exception(e)
#                     span.set_status(trace.Status(trace.StatusCode.ERROR))
#                     raise
#         return wrap
#     return deco
#
# @with_span("charge")
# def charge(...): ...

# Go (defer pattern)
# func charge(ctx context.Context, amt float64) (err error) {
#     ctx, span := tracer.Start(ctx, "charge")
#     defer func() {
#         if err != nil {
#             span.SetStatus(codes.Error, err.Error())
#             span.RecordError(err)
#         }
#         span.End()
#     }()
#     ...
# }

# Java (try-with-resources)
# Span span = tracer.spanBuilder("charge").startSpan();
# try (Scope s = span.makeCurrent()) {
#     ...
# } catch (Throwable t) {
#     span.recordException(t); span.setStatus(StatusCode.ERROR); throw t;
# } finally { span.end(); }

# Node (startActiveSpan)
# tracer.startActiveSpan('charge', async span => {
#   try { ... }
#   catch (e) { span.recordException(e); span.setStatus({code: SpanStatusCode.ERROR}); throw e; }
#   finally { span.end(); }
# });

# Rust (#[instrument])
# #[tracing::instrument]
# async fn charge(...) -> Result<()> { ... }

# .NET (Activity)
# using var activity = Source.StartActivity("charge");
# try { ... }
# catch (Exception e) { activity?.SetStatus(ActivityStatusCode.Error, e.Message); throw; }
```

### Span events vs attributes

```bash
# attributes  — describe the WHOLE span (set anytime, ideally up front)
#   span.SetAttributes(payment.amount=42.50, payment.currency="USD")
#
# events      — describe a POINT IN TIME inside the span
#   span.AddEvent("cache_miss", {key="user:42"})
#   span.AddEvent("retry", {attempt=2, backoff_ms=100})
```

### Status codes

```bash
# StatusCode:
#   UNSET   default; treat as success
#   OK      explicitly successful (rarely needed)
#   ERROR   the operation failed; description = error message
#
# Spec rule: do NOT set status to OK inside libraries — let the user decide.
# Spec rule: DO set status to ERROR for unhandled exceptions / 5xx.
# Spec rule: 4xx HTTP responses on the SERVER side are NOT errors by default.
```

## Auto-Instrumentation

Auto-instrumentation injects span/metric/log emission into a process without code changes. Three flavors:

### 1. Agent / monkey-patch (Python, Node, .NET, Java)

```bash
# Java — bytecode agent
java -javaagent:opentelemetry-javaagent.jar -jar app.jar

# Python — opentelemetry-instrument wraps the entrypoint
opentelemetry-instrument python app.py

# Node — preload module
node --require @opentelemetry/auto-instrumentations-node/register app.js

# .NET — CLR profiler (CORECLR_*) env vars
```

### 2. eBPF agent (Go, C++, Rust)

```bash
# OpenTelemetry eBPF Auto-Instrumentation (Go beta)
# kubectl apply -f https://github.com/open-telemetry/opentelemetry-go-instrumentation/...
#
# Attaches uprobes to known function symbols (net/http handlers,
# database/sql drivers, gRPC stubs) and emits OTLP without recompile.
```

### 3. Library-level instrumentation (every language)

```bash
# Each ecosystem has a registry of instrumentation packages:
#   Python:  opentelemetry-instrumentation-flask, -django, -requests, -psycopg2, ...
#   Node:    @opentelemetry/instrumentation-express, -http, -pg, -mongodb, ...
#   Go:      go.opentelemetry.io/contrib/instrumentation/...
#   Java:    auto-discovered by the agent JAR
#   .NET:    OpenTelemetry.Instrumentation.AspNetCore, .Http, .SqlClient, ...
```

### Common OTEL_* env vars used by all auto-instrumentation

```bash
export OTEL_SERVICE_NAME=payments-api
export OTEL_RESOURCE_ATTRIBUTES="deployment.environment=prod,service.version=1.4.2"
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
export OTEL_TRACES_SAMPLER=parentbased_traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.1
export OTEL_PROPAGATORS=tracecontext,baggage
export OTEL_LOGS_EXPORTER=otlp
export OTEL_METRICS_EXPORTER=otlp
```

## Context Propagation

OpenTelemetry uses **propagators** to serialize SpanContext to and from headers. The W3C Trace Context propagator is the default and the only one you should use unless interoperating with legacy systems.

### Propagator chain

```bash
# OTEL_PROPAGATORS=tracecontext,baggage   (default)
#
# Inject order = chain order:
#   1. tracecontext  -> writes "traceparent" + "tracestate"
#   2. baggage       -> writes "baggage"
#
# Extract is also chain-ordered; first to find a context wins.
```

### Supported propagators

| Propagator | Headers written | Use for |
|------------|----------------|---------|
| `tracecontext` | `traceparent`, `tracestate` | W3C standard, modern |
| `baggage` | `baggage` | W3C standard, app-level kv |
| `b3` | `b3` (single) | Legacy Zipkin |
| `b3multi` | `x-b3-traceid`, `x-b3-spanid`, `x-b3-sampled`, `x-b3-parentspanid`, `x-b3-flags` | Legacy Zipkin (multi) |
| `jaeger` | `uber-trace-id` | Legacy Jaeger |
| `xray` | `X-Amzn-Trace-Id` | AWS X-Ray |
| `ottrace` | `ot-tracer-traceid`, ... | Legacy OpenTracing |

### Inject / Extract example (Go)

```bash
# Outbound — inject context into HTTP request headers
# req, _ := http.NewRequestWithContext(ctx, "GET", "https://api/v1", nil)
# otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))
#
# Inbound — extract context from HTTP request headers
# ctx := otel.GetTextMapPropagator().Extract(r.Context(),
#         propagation.HeaderCarrier(r.Header))
# ctx, span := tracer.Start(ctx, "handle.request")
```

### Baggage

```bash
# Baggage = arbitrary key-value pairs propagated alongside trace context.
# Common use: tenant id, feature flag, user-tier — read by downstream services.
#
# Set in producer:
# ctx, _ = baggage.NewMember("tenant.id", "acme-corp")
# ctx = baggage.ContextWithBaggage(ctx, bag)
#
# Read in consumer:
# bag := baggage.FromContext(ctx)
# tenant := bag.Member("tenant.id").Value()
#
# Wire format: baggage: tenant.id=acme-corp,feature.beta=on
```

## The Collector

The OpenTelemetry Collector is a standalone process you run on each node (or as a gateway) that receives telemetry, processes it (batching, filtering, sampling, attribute editing), and re-exports it to one or more backends. Two distributions:

```bash
# core    — github.com/open-telemetry/opentelemetry-collector
#           minimal: otlp + a few core processors
# contrib — github.com/open-telemetry/opentelemetry-collector-contrib
#           the kitchen sink: 100+ receivers, processors, exporters
```

### Receivers (selection)

| Receiver | Signals | Purpose |
|----------|---------|---------|
| `otlp` | T/M/L | OTLP gRPC + HTTP (the default ingest) |
| `jaeger` | T | Thrift compact / binary / HTTP / gRPC |
| `zipkin` | T | HTTP/JSON v1 + v2 |
| `prometheus` | M | Scrape Prometheus targets |
| `prometheusremotewrite` | M | Receive Prom remote-write |
| `hostmetrics` | M | CPU, memory, disk, net of the host |
| `filelog` | L | Tail and parse log files |
| `kafka` | T/M/L | Pull from Kafka topics |
| `kubeletstats` | M | Pull from kubelet `/stats/summary` |
| `k8s_cluster` | M | Cluster-level metrics from API server |
| `k8s_events` | L | Watch Kubernetes events |
| `syslog` | L | RFC 5424 / 3164 over UDP/TCP |
| `journald` | L | Read systemd journal |
| `windowseventlog` | L | Windows Event Log |
| `mysql`/`postgresql`/`redis`/... | M | Receiver per data store |

### Processors (selection)

| Processor | Purpose |
|-----------|---------|
| `batch` | Group items before export — required for production |
| `memory_limiter` | Drop data when RSS exceeds limit — prevents OOM |
| `attributes` | Add/update/delete/hash span/log attributes |
| `resource` | Add/update Resource-level attributes |
| `transform` | OTTL (OpenTelemetry Transformation Language) — flexible attribute editing |
| `filter` | Drop spans/metrics/logs by predicate |
| `tail_sampling` | Sample after seeing the whole trace |
| `probabilistic_sampler` | Hash-based head sampling at the collector |
| `metricstransform` | Rename, scale, aggregate metrics |
| `deltatocumulative` | Convert delta metrics to cumulative for Prometheus |
| `cumulativetodelta` | Convert cumulative metrics to delta |
| `groupbyattrs` | Re-group spans/metrics by attribute key |
| `k8sattributes` | Auto-tag with k8s.pod.name etc by IP lookup |
| `routing` | Send to different pipelines based on attribute |

### Exporters (selection)

| Exporter | Signals | Backend |
|----------|---------|---------|
| `otlp` | T/M/L | Any OTLP receiver (gRPC) |
| `otlphttp` | T/M/L | Any OTLP receiver (HTTP) |
| `prometheus` | M | Pull endpoint :8889/metrics |
| `prometheusremotewrite` | M | Push to Prom/Mimir/Cortex/Thanos/VM |
| `jaeger` | T | Jaeger (deprecated; use OTLP) |
| `zipkin` | T | Zipkin |
| `loki` | L | Grafana Loki |
| `debug` (formerly `logging`) | T/M/L | stderr — for debugging |
| `file` | T/M/L | newline-delimited JSON to file |
| `kafka` | T/M/L | Kafka topics |
| `awsxray` | T | AWS X-Ray |
| `awscloudwatchlogs` | L | CloudWatch Logs |
| `datadog` | T/M/L | Datadog |
| `splunk_hec` | T/M/L | Splunk HEC |
| `elasticsearch` | T/L | ES |

## Collector Configuration

YAML structure with five top-level keys: `receivers`, `processors`, `exporters`, `extensions`, `service`. Pipelines under `service.pipelines` wire them together per signal.

```bash
# /etc/otelcol/config.yaml structure
receivers:    {<name>: {<config>}, ...}
processors:   {<name>: {<config>}, ...}
exporters:    {<name>: {<config>}, ...}
extensions:   {<name>: {<config>}, ...}
service:
  extensions: [list of extension names]
  telemetry:
    logs: {level: info}
    metrics: {address: 0.0.0.0:8888}
  pipelines:
    traces:
      receivers:  [list]
      processors: [list]
      exporters:  [list]
    metrics: {receivers, processors, exporters}
    logs:    {receivers, processors, exporters}
```

### Validate and run

```bash
# Validate config syntax (does not connect anywhere)
otelcol validate --config=/etc/otelcol/config.yaml

# Run in foreground
otelcol --config=/etc/otelcol/config.yaml

# Docker (contrib distribution)
docker run -d --name otelcol \
  -p 4317:4317 -p 4318:4318 -p 8889:8889 \
  -v $(pwd)/config.yaml:/etc/otelcol-contrib/config.yaml \
  otel/opentelemetry-collector-contrib:0.116.0

# Multiple --config flags merge in order (later wins)
otelcol --config=/base.yaml --config=/override.yaml
```

### Environment variable substitution

```bash
# In YAML: ${env:VAR}  or  ${env:VAR:-default}
# exporters:
#   otlp:
#     endpoint: ${env:OTEL_BACKEND:-tempo:4317}
#     headers:
#       authorization: Bearer ${env:OTEL_API_KEY}
```

## Collector — Common Pipelines

Three production-grade reference pipelines you can copy and adapt.

### Trace pipeline (otlp -> batch -> otlp to Tempo)

```bash
# receivers:
#   otlp:
#     protocols:
#       grpc: {endpoint: 0.0.0.0:4317}
#       http: {endpoint: 0.0.0.0:4318}
# processors:
#   memory_limiter:
#     check_interval: 1s
#     limit_mib: 512
#     spike_limit_mib: 128
#   batch:
#     timeout: 5s
#     send_batch_size: 1024
#     send_batch_max_size: 2048
# exporters:
#   otlp/tempo:
#     endpoint: tempo.observability:4317
#     tls: {insecure: true}
#     sending_queue: {enabled: true, num_consumers: 4, queue_size: 5000}
#     retry_on_failure: {enabled: true, initial_interval: 5s, max_interval: 30s}
# service:
#   pipelines:
#     traces:
#       receivers: [otlp]
#       processors: [memory_limiter, batch]
#       exporters: [otlp/tempo]
```

### Metric pipeline (otlp -> memory_limiter -> batch -> prometheusremotewrite)

```bash
# exporters:
#   prometheusremotewrite:
#     endpoint: https://mimir.observability/api/v1/push
#     headers:
#       X-Scope-OrgID: tenant-1
#     resource_to_telemetry_conversion: {enabled: true}
# service:
#   pipelines:
#     metrics:
#       receivers:  [otlp, hostmetrics]
#       processors: [memory_limiter, batch]
#       exporters:  [prometheusremotewrite]
```

### Log pipeline (filelog -> resource -> batch -> otlphttp to Loki)

```bash
# receivers:
#   filelog:
#     include: [/var/log/app/*.log]
#     start_at: end
#     operators:
#       - type: json_parser
#         parse_to: body
# processors:
#   resource:
#     attributes:
#       - {key: service.name, value: payments-api, action: upsert}
# exporters:
#   otlphttp/loki:
#     endpoint: https://loki.observability/otlp
# service:
#   pipelines:
#     logs:
#       receivers:  [filelog]
#       processors: [resource, memory_limiter, batch]
#       exporters:  [otlphttp/loki]
```

### Deployment topology — agent + gateway

```bash
# Agent     — DaemonSet on each k8s node (or sidecar in each pod)
#             - light load (batch, memory_limiter)
#             - hostmetrics, filelog from /var/log/pods, k8s_events
#             - exports OTLP to gateway
#
# Gateway   — Deployment, scale-out behind LB
#             - heavier processors (tail_sampling, transform)
#             - rate limit, auth, multi-tenant routing
#             - fans out to backends (Tempo, Mimir, Loki, Datadog, ...)
#
# Why two layers? Agent has node-local context; gateway has whole-trace context
# for tail sampling and centralized backend credentials.
```

## Sampling — Head vs Tail

### Head sampling (in the SDK, before export)

Decision is made when the trace begins, based purely on `trace_id` (no attributes yet). Cheap, deterministic across services if every service uses the same sampler.

```bash
# Built-in samplers (Trace API):
#   AlwaysOn               sample 100%
#   AlwaysOff              sample 0%
#   TraceIDRatioBased(p)   hash trace_id, sample if hash < p*MAX
#   ParentBased(default)   honor upstream sampled-flag; fall back to default
#                          for root spans
#
# Common production setting:
export OTEL_TRACES_SAMPLER=parentbased_traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.1                        # 10% of root traces
```

```bash
# ParentBased policy (full)
# parentbased_always_on               root=on,    parent=respect
# parentbased_always_off              root=off,   parent=respect
# parentbased_traceidratio            root=ratio, parent=respect
```

### Tail sampling (in the Collector, after seeing whole trace)

Buffer all spans for `decision_wait` seconds, then apply policies. Lets you keep 100% of errors and slow traces while dropping 99% of normal traffic.

```bash
# processors:
#   tail_sampling:
#     decision_wait: 10s                # buffer per trace
#     num_traces: 50000                 # max traces buffered
#     expected_new_traces_per_sec: 1000
#     policies:
#       - name: errors
#         type: status_code
#         status_code: {status_codes: [ERROR]}
#       - name: slow
#         type: latency
#         latency: {threshold_ms: 1000}
#       - name: rare-endpoint
#         type: string_attribute
#         string_attribute:
#           key: http.route
#           values: [/api/admin/.*]
#           enabled_regex_matching: true
#       - name: keep-1pc
#         type: probabilistic
#         probabilistic: {sampling_percentage: 1}
#
# Pipeline ordering MATTERS: tail_sampling MUST come before batch on export.
# A trace MUST land entirely on one collector instance — load-balance by trace_id
# (use loadbalancing exporter in front of tail-sampling gateway).
```

### Probabilistic sampler (cheap head-style at the collector)

```bash
# processors:
#   probabilistic_sampler:
#     sampling_percentage: 10
#     hash_seed: 22
#   # decision based on trace_id hash; consistent across collectors
```

## The Logs Bridge

OpenTelemetry Logs is the youngest signal. The strategy: **don't replace your logging library** — bridge it. The OTel API defines a `LogRecord` type that mirrors structured logs; per-language bridges hook into the existing logging framework and emit OTLP.

### Bridges per language

| Language | Bridge package | Hooks into |
|----------|----------------|------------|
| Python | `opentelemetry-sdk._logs` + standard handler | `logging` module |
| Java | `opentelemetry-log4j-2.17` / `opentelemetry-logback-1.0` | log4j2 / logback |
| Node | `@opentelemetry/winston-transport` / `@opentelemetry/instrumentation-pino` | winston / pino |
| Go (1.21+) | `go.opentelemetry.io/contrib/bridges/otelslog` | `slog` |
| .NET | `OpenTelemetry.Logs` | `Microsoft.Extensions.Logging` |
| Ruby | `opentelemetry-logs-api` (beta) | `logger` |

### Trace-log correlation (auto-injected)

```bash
# When you log inside an active span, the bridge auto-injects:
#   trace_id  (16 bytes hex)
#   span_id   (8 bytes hex)
#   trace_flags (1 byte)
#
# Backend (Loki, ES, Splunk) shows these alongside the log line; clicking
# a log pivots to the trace in Tempo/Jaeger.
```

### Go slog bridge

```bash
# import "go.opentelemetry.io/contrib/bridges/otelslog"
# logger := otelslog.NewLogger("payments-api")
# logger.InfoContext(ctx, "charge succeeded", "amount", 42.50, "currency", "USD")
# # trace_id and span_id from ctx are auto-attached to the LogRecord
```

### Python standard logging bridge

```bash
# from opentelemetry._logs import set_logger_provider
# from opentelemetry.sdk._logs import LoggerProvider, LoggingHandler
# from opentelemetry.sdk._logs.export import BatchLogRecordProcessor
# from opentelemetry.exporter.otlp.proto.grpc._log_exporter import OTLPLogExporter
# import logging
#
# provider = LoggerProvider()
# provider.add_log_record_processor(BatchLogRecordProcessor(OTLPLogExporter()))
# set_logger_provider(provider)
# logging.getLogger().addHandler(LoggingHandler(level=logging.INFO))
# logging.info("charge succeeded amount=%s", 42.50)
```

## Resource Detectors

Detectors scan the runtime environment and populate Resource attributes. Configure via `OTEL_RESOURCE_DETECTORS` (comma-separated) — order matters because later detectors can overwrite earlier values for the same key.

| Detector | Reads | Sets |
|----------|-------|------|
| `env` | `OTEL_RESOURCE_ATTRIBUTES` env | arbitrary k=v |
| `host` | hostname, /sys/class/dmi | `host.name`, `host.id`, `host.arch` |
| `os` | uname / /etc/os-release | `os.type`, `os.description` |
| `process` | /proc/self | `process.pid`, `process.command_line`, `process.runtime.*` |
| `container` | /proc/self/cgroup | `container.id` |
| `k8s` | downward API + service account | `k8s.pod.name`, `k8s.namespace.name`, `k8s.pod.uid` |
| `aws_ec2` | IMDSv2 (169.254.169.254) | `host.id`, `cloud.region`, `cloud.account.id` |
| `aws_eks` | EKS cluster ARN | `k8s.cluster.name`, `cloud.*` |
| `aws_ecs` | ECS metadata endpoint | `aws.ecs.task.arn`, `cloud.*` |
| `aws_lambda` | `AWS_LAMBDA_*` env | `faas.*`, `cloud.*` |
| `gcp` | metadata.google.internal | `host.id`, `cloud.region`, `gcp.*` |
| `azure_vm` | Azure IMDS | `host.id`, `cloud.region`, `azure.*` |

```bash
# Typical k8s pod
export OTEL_RESOURCE_DETECTORS="env,host,os,process,container,k8s"
# pod.spec.containers[].env injects:
#   OTEL_RESOURCE_ATTRIBUTES=k8s.pod.name=$(POD_NAME),k8s.namespace.name=$(NS),...
# via the downward API
```

## Environment Variables

OpenTelemetry standardizes a large set of `OTEL_*` env vars so any SDK can be configured without code changes.

### Service identity

```bash
OTEL_SERVICE_NAME=payments-api                       # required for production
OTEL_RESOURCE_ATTRIBUTES="key1=v1,key2=v2"           # comma-separated kv
OTEL_RESOURCE_DETECTORS="env,host,os,process,k8s"
```

### Exporter — OTLP

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://collector:4317    # one endpoint for all signals
OTEL_EXPORTER_OTLP_PROTOCOL=grpc                     # grpc | http/protobuf | http/json
OTEL_EXPORTER_OTLP_HEADERS="Authorization=Bearer xxx"
OTEL_EXPORTER_OTLP_TIMEOUT=10000                     # milliseconds, default 10000
OTEL_EXPORTER_OTLP_COMPRESSION=gzip                  # gzip | none
OTEL_EXPORTER_OTLP_INSECURE=true                     # disable TLS for local dev

# Per-signal overrides (take precedence over the unsuffixed forms)
OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://traces:4318/v1/traces
OTEL_EXPORTER_OTLP_METRICS_ENDPOINT=http://metrics:4318/v1/metrics
OTEL_EXPORTER_OTLP_LOGS_ENDPOINT=http://logs:4318/v1/logs
OTEL_EXPORTER_OTLP_TRACES_PROTOCOL=http/protobuf
```

### Sampling

```bash
OTEL_TRACES_SAMPLER=parentbased_traceidratio
OTEL_TRACES_SAMPLER_ARG=0.1                          # 10%

# All sampler names:
#   always_on / always_off
#   traceidratio
#   parentbased_always_on / parentbased_always_off / parentbased_traceidratio
#   jaeger_remote (with arg "endpoint=http://...,initial_sampling_rate=0.1")
#   xray
```

### Exporter selection

```bash
OTEL_TRACES_EXPORTER=otlp                            # otlp | jaeger | zipkin | console | none
OTEL_METRICS_EXPORTER=otlp                           # otlp | prometheus | console | none
OTEL_LOGS_EXPORTER=otlp                              # otlp | console | none
```

### Batch span processor tuning

```bash
OTEL_BSP_MAX_QUEUE_SIZE=2048                         # default 2048
OTEL_BSP_MAX_EXPORT_BATCH_SIZE=512                   # default 512
OTEL_BSP_SCHEDULE_DELAY=5000                         # default 5000 ms
OTEL_BSP_EXPORT_TIMEOUT=30000                        # default 30000 ms
```

### Metric reader tuning

```bash
OTEL_METRIC_EXPORT_INTERVAL=60000                    # default 60000 ms
OTEL_METRIC_EXPORT_TIMEOUT=30000                     # default 30000 ms
```

### Propagation

```bash
OTEL_PROPAGATORS=tracecontext,baggage                # default
# Other: b3, b3multi, jaeger, xray, ottrace
```

### Diagnostic / SDK behavior

```bash
OTEL_SDK_DISABLED=false                              # set true to no-op all SDKs
OTEL_LOG_LEVEL=info                                  # error | warn | info | debug | trace
OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT=4096
OTEL_ATTRIBUTE_COUNT_LIMIT=128
OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT=128
OTEL_SPAN_EVENT_COUNT_LIMIT=128
OTEL_SPAN_LINK_COUNT_LIMIT=128
```

## Common Errors and Fixes

Exact error strings you will see, mapped to the actual problem and the fix.

### `rpc error: code = DeadlineExceeded desc = context deadline exceeded`

```bash
# Cause:    Exporter blocked > OTEL_EXPORTER_OTLP_TIMEOUT (default 10s).
# Symptom:  Batches dropped, spans missing in backend.
# Fix:      Increase timeout AND check collector backpressure / queue size.
export OTEL_EXPORTER_OTLP_TIMEOUT=30000
# Also tune sending_queue.queue_size in the exporter on the collector side.
```

### `rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing dial tcp: lookup ...: no such host"`

```bash
# Cause:    Collector hostname does not resolve, or wrong port.
# Fix:      Verify endpoint string and DNS:
getent hosts otelcol.observability
nc -zv otelcol.observability 4317
# Common mistake: using http://host:4317 with gRPC exporter — gRPC wants host:port (no scheme).
```

### `rpc error: code = Unavailable desc = "..." service config: ..."`

```bash
# Cause:    Collector receiver not running or wrong protocol on that port.
# Fix:      Confirm receiver: 4317=gRPC, 4318=HTTP. They are NOT interchangeable.
otelcol --config=config.yaml validate
```

### `context canceled`

```bash
# Cause:    Trace context lost across an async boundary (goroutine, callback, etc).
#           The span ended before the child code ran.
# Fix:      Pass ctx through every async call. Never use background ctx inside
#           a request handler.
# Bad:   go func() { tracer.Start(context.Background(), "child") }()
# Good:  go func(ctx context.Context) { tracer.Start(ctx, "child") }(ctx)
```

### `Setting attribute X overwrites previous value`  (warning)

```bash
# Cause:    Same attribute key set twice on one span.
# Fix:      Use one canonical key. Common foot-gun: setting both
#           "http.method" (deprecated) and "http.request.method" (current).
```

### `Instrumentation is not initialized` / `tracer not configured`

```bash
# Cause:    Auto-instrumentation didn't load before app code.
# Fix:
#   Java:   ensure -javaagent: flag is on the JVM cmdline, not just classpath.
#   Python: must run via `opentelemetry-instrument python app.py`, not bare python.
#   Node:   --require ./tracing.js MUST come before the script path.
#   .NET:   verify CORECLR_PROFILER env vars are set in the same process.
```

### `service.name not set, defaulting to "unknown_service:<process>"`

```bash
# Cause:    Resource missing service.name.
# Symptom:  Backend shows traces under "unknown_service".
# Fix:      Always set OTEL_SERVICE_NAME or service.name in resource.
export OTEL_SERVICE_NAME=payments-api
```

### `BatchSpanProcessor: queue is full, dropping span`

```bash
# Cause:    Span generation rate > export rate; queue filled.
# Fix:      Tune BSP env vars and/or exporter sending_queue.
export OTEL_BSP_MAX_QUEUE_SIZE=8192
export OTEL_BSP_MAX_EXPORT_BATCH_SIZE=2048
# Or sample more aggressively to reduce span volume.
```

### `error exporting items: Permanent error: rpc error: code = ResourceExhausted`

```bash
# Cause:    Backend quota / rate limit / max body size hit.
# Fix:      Reduce batch size, enable compression, sample more.
exporters:
  otlp:
    compression: gzip
    sending_queue: {queue_size: 1000}
```

### `Trace ID is invalid: must be 16 bytes`

```bash
# Cause:    Manually constructing SpanContext with wrong-length ID, or all-zero.
# Fix:      Use SDK's IDGenerator. Never hardcode a trace_id of all zeros.
```

## Common Gotchas

Concrete bad/good patterns. Memorize the bad column — it's where 90% of OTel pain lives.

### 1. Ending span outside its activation scope leaks context

```bash
# bad — span ended after scope exited; subsequent code misses span
# def handler():
#     span = tracer.start_span("op")
#     # span never made current!
#     do_work()
#     span.end()
#
# good — use start_as_current_span (Python) / WithContext (Go) / makeCurrent (Java)
# def handler():
#     with tracer.start_as_current_span("op"):
#         do_work()      # span is current here
#     # span ended automatically
```

### 2. High-cardinality attribute on a metric -> cardinality explosion

```bash
# bad — userId on a Counter creates one time series per user (millions)
# counter.Add(1, attribute.String("user.id", uid))
#
# good — userId belongs on traces, NEVER on metrics
# span.SetAttributes(attribute.String("user.id", uid))     # trace: ok, one record
# counter.Add(1, attribute.String("http.route", route))    # metric: low cardinality
#
# Rule of thumb: metric attribute cardinality should be < 10000 unique combos.
```

### 3. Wrong span kind

```bash
# bad — sending HTTP from this service, but kind=SERVER
# span = tracer.start_span("call_payment_api", kind=SpanKind.SERVER)
#
# good — outbound = CLIENT, inbound = SERVER
# span = tracer.start_span("call_payment_api", kind=SpanKind.CLIENT)
#
# Span kinds:
#   SERVER     — inbound (request received)
#   CLIENT     — outbound (request sent)
#   PRODUCER   — async send (queue, kafka publish)
#   CONSUMER   — async receive (queue, kafka consume)
#   INTERNAL   — neither inbound nor outbound (default)
```

### 4. Missing service.name -> "unknown_service"

```bash
# bad — no Resource attributes set
# tracer = trace.get_tracer("x")
#
# good — always set service.name
# resource = Resource.create({SERVICE_NAME: "payments-api"})
# provider = TracerProvider(resource=resource)
```

### 5. BatchSpanProcessor with default queue under load

```bash
# bad — default 2048 queue drops spans at >2k spans/sec burst
#
# good — tune for actual throughput
export OTEL_BSP_MAX_QUEUE_SIZE=8192
export OTEL_BSP_MAX_EXPORT_BATCH_SIZE=2048
export OTEL_BSP_SCHEDULE_DELAY=2000
```

### 6. Too many spans per request

```bash
# bad — instrumenting every helper / stdlib call -> 500 spans/request
# good — span only at meaningful boundaries (handler, DB call, RPC, expensive compute)
#        attributes and events handle finer detail
```

### 7. Setting status OK from a library

```bash
# bad — library forces status OK, overrides app's later ERROR set
# good — libraries leave status UNSET; only the app sets ERROR or OK
```

### 8. Mixing `http.method` and `http.request.method`

```bash
# bad — old + new semconv on same span (most languages emit warnings)
# good — pick one version of semconv, stick with it (the SDK helper does this for you)
```

### 9. Forgetting to flush on shutdown

```bash
# bad — process exits, BatchSpanProcessor still has spans in queue, lost forever
#
# good — call shutdown / forceFlush before exit
# defer tp.Shutdown(ctx)         (Go)
# trace.get_tracer_provider().shutdown()      (Python)
# sdk.close()                    (Java)
# await sdk.shutdown()           (Node)
```

### 10. Cardinality from path parameters

```bash
# bad — http.route = "/api/users/42"  (one timeseries per user_id)
# good — http.route = "/api/users/:id" (one timeseries for the route)
# Most HTTP framework instrumentation does this automatically — verify yours does.
```

### 11. Sampling at collector but tail-sampler split across replicas

```bash
# bad — gateway scaled to N replicas; trace's spans hit different replicas;
#       tail_sampling sees only partial trace; wrong decision.
# good — put a `loadbalancing` exporter in front of tail-sampling collectors,
#        keyed by trace_id, so all spans of one trace land on one collector.
```

### 12. Exposing collector to the internet without auth

```bash
# bad — :4317 open to 0.0.0.0 with no auth
# good — bind to internal network, add `bearertokenauth` extension or mTLS
# extensions:
#   bearertokenauth:
#     scheme: Bearer
#     token: ${env:COLLECTOR_TOKEN}
# receivers:
#   otlp:
#     protocols:
#       grpc:
#         auth: {authenticator: bearertokenauth}
```

## Performance and Cost

Telemetry has direct cost dimensions. Each can balloon without warning.

### The four cost axes

| Axis | What drives it | Typical limit |
|------|----------------|---------------|
| Span throughput | Spans/sec exported | Network + backend ingest QPS |
| Span attribute count | Attributes per span | OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT (default 128) |
| Metric cardinality | Unique attribute combos per metric | < 10k per metric, < 1M per backend |
| Log volume | Bytes/sec exported | Storage cost ~$0.50–$5 per GB ingested |

### Cost-optimization checklist

```bash
# 1. Sample aggressively for high-volume services
export OTEL_TRACES_SAMPLER=parentbased_traceidratio
export OTEL_TRACES_SAMPLER_ARG=0.01      # 1%
# Use tail-sampling to keep 100% errors + slow.

# 2. Drop attributes you don't query
# processors:
#   attributes:
#     actions:
#       - {key: http.user_agent.original, action: delete}
#       - {key: url.full, action: delete}             # often huge

# 3. Drop spans you don't need (health checks, metrics scrapes)
# processors:
#   filter:
#     traces:
#       span:
#         - 'attributes["http.target"] == "/healthz"'
#         - 'attributes["http.target"] == "/metrics"'

# 4. Batch — never export one span at a time
# processors:
#   batch: {timeout: 5s, send_batch_size: 1024}

# 5. Use OTLP gRPC over OTLP HTTP — fewer bytes, less CPU
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc

# 6. Compression
export OTEL_EXPORTER_OTLP_COMPRESSION=gzip

# 7. Memory-limit the collector
# processors:
#   memory_limiter:
#     limit_mib: 1024
#     spike_limit_mib: 256
#     check_interval: 1s

# 8. Drop unused metric streams via views (SDK)
# in MeterProvider config:
#   views: [{instrument: "*", aggregation: drop}]   # match by name

# 9. Histogram bucket count — fewer buckets = less storage
# default 15 explicit buckets; consider exponential histogram for adaptive boundaries.
```

### Span volume math (rough)

```bash
# Avg span size on the wire (OTLP gRPC + gzip): ~300 bytes
# 1000 RPS * 5 spans/req = 5000 spans/sec
# 5000 * 300 = 1.5 MB/sec = 130 GB/day
# At 100% sample. Plan storage and network accordingly.
# Drop to 1% sample with tail-keep-errors: ~1.3 GB/day.
```

## Backend Integration — Vendor-neutral

OTel's promise: ship OTLP, render in anything. Below: pairing of OSS and commercial backends with the OTel exporter you'd use.

### Open-source stacks

| Backend | Signals | Talk OTLP? | Exporter |
|---------|---------|------------|----------|
| Jaeger 1.35+ | T | yes (4317/4318) | `otlp` |
| Tempo | T | yes (4317/4318) | `otlp` |
| Zipkin | T | no (Zipkin v2 JSON) | `zipkin` |
| Prometheus 2.47+ | M | yes (HTTP `/api/v1/otlp/v1/metrics`) | `otlphttp` |
| Mimir / Cortex / Thanos | M | via remote-write | `prometheusremotewrite` |
| VictoriaMetrics | M | via remote-write or OTLP | `prometheusremotewrite` or `otlphttp` |
| Loki | L | yes (`/otlp/v1/logs` since 3.0) | `otlphttp` |
| Elasticsearch | T/L | partial (APM agents speak OTLP) | `elasticsearch` or `otlp` |
| OpenSearch | T/L | partial | `opensearch` (contrib) |
| ClickHouse | T/M/L | via contrib exporter | `clickhouse` (contrib) |
| Grafana Beyla | T/M | yes | OTLP-native |

### Commercial backends (all accept OTLP directly)

| Vendor | OTLP endpoint pattern |
|--------|----------------------|
| Datadog | `https://otlp-intake.<site>/v1/<traces|metrics|logs>` (or via DD Agent) |
| New Relic | `https://otlp.nr-data.net:4317` (api-key header) |
| Honeycomb | `https://api.honeycomb.io:443` (x-honeycomb-team header) |
| Splunk Observability | `https://ingest.<realm>.signalfx.com/v2/trace/otlp` |
| Lightstep / ServiceNow | `https://ingest.lightstep.com:443` |
| Elastic Cloud | `https://<deployment>.apm.<region>.cloud.es.io:443` |
| Dynatrace | `https://<env>.live.dynatrace.com/api/v2/otlp` |
| AWS X-Ray | via OpenTelemetry Collector AWS distro (ADOT) |
| Google Cloud Trace | via Google exporter or Collector `googlecloud` exporter |
| Azure Monitor | via Application Insights exporter |

```bash
# Generic OTLP -> commercial vendor
export OTEL_EXPORTER_OTLP_ENDPOINT=https://otlp.example-vendor.io:443
export OTEL_EXPORTER_OTLP_HEADERS="api-key=${VENDOR_API_KEY}"
export OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

### "Render with anything" — the dashboard side

```bash
# Grafana — can read from Tempo (traces), Mimir/Prom (metrics), Loki (logs)
#   pivots between datasources via shared trace_id
# Jaeger UI — traces only
# Kibana — traces + logs
# Honeycomb / Datadog / NR / Splunk — full UI for all three signals
```

## Idioms

Crisp rules to internalize. Each is a thing your future self will thank you for.

### Tracing idioms

```bash
# 1. Every public function gets a span — name = function name or operation name.
# 2. Span names are LOW cardinality — never include user_id / order_id in the name.
# 3. Span attributes follow semconv — never invent http_method when http.request.method exists.
# 4. Trace context propagates through every async boundary — pass ctx, never use Background.
# 5. Errors recorded as span events with span.RecordError() AND status = ERROR.
# 6. Span.kind is set explicitly: SERVER inbound, CLIENT outbound, INTERNAL otherwise.
# 7. Root span = the entry point of the request (HTTP handler, gRPC method, queue consumer).
# 8. defer/finally span.End() — always — paired with span creation.
```

### Metric idioms

```bash
# 1. Counter for monotonic up; UpDownCounter for active values; Histogram for distributions.
# 2. Names follow semconv: http.server.duration, db.client.connections.usage, system.cpu.utilization.
# 3. Units are SI: seconds (not ms), bytes (not KB), 1 (dimensionless).
# 4. Attributes are LOW cardinality — http.route, http.method, http.status_code.
# 5. Never use user_id, order_id, request_id as metric attributes.
# 6. Histogram bucket boundaries match your SLO (e.g., bucket at 500ms if SLO is 500ms).
```

### Log idioms

```bash
# 1. Logs structured — JSON or attribute key-value, never free-form formatted strings.
# 2. Logs INSIDE a span auto-correlate with trace_id + span_id via the bridge.
# 3. Log severity: TRACE/DEBUG/INFO/WARN/ERROR/FATAL — match OTel severity_number.
# 4. Don't log every span — spans ARE your timing data. Logs add narrative when traces don't.
```

### Deployment idioms

```bash
# 1. Run a Collector — never have apps export OTLP directly to the backend.
#    Reason: decouple deploys, centralize auth/credentials, batch + retry in one place.
# 2. K8s: deploy collector as DaemonSet (one per node).
#    Optionally a Deployment (gateway) behind it for tail-sampling.
# 3. Sidecar pattern for non-k8s: collector as a sidecar container per pod / VM.
# 4. Use `loadbalancing` exporter -> tail-sampling collectors when scaling out gateway.
# 5. Always set memory_limiter FIRST in the processor chain.
# 6. Always set batch LAST in the processor chain (just before export).
```

### Configuration idioms

```bash
# 1. Use OTEL_* env vars — never hardcode endpoints in source.
# 2. service.name set in three places (defense in depth):
#    - OTEL_SERVICE_NAME env
#    - resource.SERVICE_NAME in code
#    - k8s downward API (OTEL_RESOURCE_ATTRIBUTES)
# 3. SDK version pinned in lockfile — never auto-upgrade in CI.
# 4. SemConv version stamped in Resource (semconv.SchemaURL) — backends use it.
```

## Tips

- Use `ParentBased` sampler to respect upstream sampling decisions while setting your own default ratio.
- Always call `span.End()` with defer/try-with-resources/finally immediately after `tracer.Start()` to avoid span leaks.
- Set `memory_limiter` processor FIRST in the Collector to drop data before downstream processors buffer it.
- Use `batch` processor LAST to coalesce items and reduce export overhead and network round trips.
- Set `OTEL_SERVICE_NAME` env var as a fallback even when configuring in code; missing service.name shows as "unknown_service".
- Use tail sampling in the Collector (not head sampling in SDK) when you need error-based or latency-based decisions.
- Filter health-check and `/metrics` spans in the Collector to reduce noise and storage costs.
- Use semantic conventions for attribute names so dashboards and queries work across services.
- Prefer gRPC OTLP exporter over HTTP for lower overhead in high-throughput services.
- Run the Collector as a DaemonSet on each k8s node (or sidecar), with optional gateway for tail-sampling.
- Export resource attributes like `service.version` and `deployment.environment` for filtering in backends.
- Validate collector configs with `otelcol validate --config=...` before reload.
- Use the `debug` exporter (formerly `logging`) during development to see what's actually being emitted.
- Bound metric cardinality: avoid putting user_id, order_id, or trace_id as metric attributes.
- Use `loadbalancing` exporter to ensure all spans of one trace land on the same tail-sampling collector replica.
- Keep span attribute count under 32 in hot paths; the spec limit is 128 but each attribute costs CPU.
- Set `OTEL_EXPORTER_OTLP_COMPRESSION=gzip` for cross-region exports.
- Use `transform` processor (OTTL) for complex attribute editing instead of multiple `attributes` processors.
- For long-running batch jobs, use `force_flush` before exit to drain the BSP queue.
- Avoid `SimpleSpanProcessor` in production — synchronous export blocks request paths.
- Hash sensitive attributes (PII like email, phone) with the `attributes` processor `hash` action before export.
- For multi-tenant OTel, route by an attribute via the `routing` connector to per-tenant exporters.
- Pin OTel collector image to a specific version tag, not `:latest`, to avoid silent breakage.
- When debugging "missing trace", first check `service.telemetry.metrics` on the collector — `otelcol_receiver_accepted_spans` and `otelcol_exporter_sent_spans` tell you where data is dropping.

## See Also

prometheus, promql, logql, grafana, jaeger, loki, polyglot

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Specification](https://github.com/open-telemetry/opentelemetry-specification)
- [OpenTelemetry Proto (OTLP)](https://github.com/open-telemetry/opentelemetry-proto)
- [Semantic Conventions](https://github.com/open-telemetry/semantic-conventions)
- [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector)
- [OpenTelemetry Collector Contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib)
- [OTEL Environment Variables](https://opentelemetry.io/docs/specs/otel/configuration/sdk-environment-variables/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
- [W3C Baggage](https://www.w3.org/TR/baggage/)
- [Language SDK Status](https://opentelemetry.io/docs/languages/)
- [Auto-Instrumentation Registry](https://opentelemetry.io/ecosystem/registry/)
- [OTLP HTTP Specification](https://opentelemetry.io/docs/specs/otlp/#otlphttp)
- [OTLP gRPC Specification](https://opentelemetry.io/docs/specs/otlp/#otlpgrpc)
