# OpenTelemetry (Observability Framework)

Vendor-neutral observability framework providing APIs, SDKs, and the Collector for generating, collecting, and exporting traces, metrics, and logs across distributed systems.

## SDK Setup (Go)

### Initialize TracerProvider

```bash
# main.go — bootstrap OpenTelemetry tracing
# import "go.opentelemetry.io/otel"
# import "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
# import sdktrace "go.opentelemetry.io/otel/sdk/trace"
#
# exporter, _ := otlptracegrpc.New(ctx,
#     otlptracegrpc.WithEndpoint("localhost:4317"),
#     otlptracegrpc.WithInsecure(),
# )
# tp := sdktrace.NewTracerProvider(
#     sdktrace.WithBatcher(exporter),
#     sdktrace.WithResource(resource.NewWithAttributes(
#         semconv.SchemaURL,
#         semconv.ServiceNameKey.String("my-service"),
#     )),
# )
# otel.SetTracerProvider(tp)
# defer tp.Shutdown(ctx)
```

### Initialize MeterProvider

```bash
# import "go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
# import sdkmetric "go.opentelemetry.io/otel/sdk/metric"
#
# metricExporter, _ := otlpmetricgrpc.New(ctx,
#     otlpmetricgrpc.WithEndpoint("localhost:4317"),
#     otlpmetricgrpc.WithInsecure(),
# )
# mp := sdkmetric.NewMeterProvider(
#     sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
# )
# otel.SetMeterProvider(mp)
```

## Creating Spans

```bash
# tracer := otel.Tracer("my-service")
# ctx, span := tracer.Start(ctx, "operation-name")
# defer span.End()
#
# span.SetAttributes(attribute.String("user.id", userID))
# span.AddEvent("cache miss", trace.WithAttributes(
#     attribute.String("key", cacheKey),
# ))
# span.SetStatus(codes.Error, "something failed")
# span.RecordError(err)
```

## Recording Metrics

```bash
# meter := otel.Meter("my-service")
#
# counter, _ := meter.Int64Counter("http.requests.total")
# counter.Add(ctx, 1, metric.WithAttributes(
#     attribute.String("method", "GET"),
#     attribute.Int("status", 200),
# ))
#
# histogram, _ := meter.Float64Histogram("http.request.duration")
# histogram.Record(ctx, 0.042, metric.WithAttributes(
#     attribute.String("route", "/api/users"),
# ))
#
# gauge, _ := meter.Int64UpDownCounter("connections.active")
# gauge.Add(ctx, 1)
```

## Propagation

```bash
# Context propagation across services
# import "go.opentelemetry.io/otel/propagation"
#
# otel.SetTextMapPropagator(
#     propagation.NewCompositeTextMapPropagator(
#         propagation.TraceContext{},     # W3C Trace Context
#         propagation.Baggage{},          # W3C Baggage
#     ),
# )
#
# HTTP headers injected:
#   traceparent: 00-<trace-id>-<span-id>-<flags>
#   tracestate:  vendor=value
#   baggage:     key=value
```

## Collector Configuration

### otel-collector-config.yaml

```yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317       # gRPC receiver
      http:
        endpoint: 0.0.0.0:4318       # HTTP receiver

processors:
  batch:
    timeout: 5s
    send_batch_size: 1024
  memory_limiter:
    check_interval: 1s
    limit_mib: 512
    spike_limit_mib: 128
  attributes:
    actions:
      - key: environment
        value: production
        action: upsert
  filter:
    error_mode: ignore
    traces:
      span:
        - 'attributes["http.target"] == "/healthz"'

exporters:
  otlp:
    endpoint: jaeger:4317
    tls:
      insecure: true
  prometheus:
    endpoint: 0.0.0.0:8889
  otlphttp:
    endpoint: https://tempo.example.com:4318

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlp]
    metrics:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [prometheus]
    logs:
      receivers: [otlp]
      processors: [memory_limiter, batch]
      exporters: [otlphttp]
```

## Run Collector

```bash
# Docker
docker run -d --name otel-collector \
  -p 4317:4317 -p 4318:4318 -p 8889:8889 \
  -v $(pwd)/otel-collector-config.yaml:/etc/otelcol/config.yaml \
  otel/opentelemetry-collector-contrib:latest

# Binary
otelcol --config=otel-collector-config.yaml

# Validate config
otelcol validate --config=otel-collector-config.yaml
```

## Auto-Instrumentation

```bash
# Go — HTTP server auto-instrumentation
# import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
#
# handler := otelhttp.NewHandler(mux, "server",
#     otelhttp.WithMessageEvents(otelhttp.ReadEvents, otelhttp.WriteEvents),
# )
# http.ListenAndServe(":8080", handler)

# Go — HTTP client
# client := &http.Client{
#     Transport: otelhttp.NewTransport(http.DefaultTransport),
# }

# Go — gRPC server
# import "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
# grpc.NewServer(
#     grpc.StatsHandler(otelgrpc.NewServerHandler()),
# )
```

## Sampling

```bash
# Always sample
# sdktrace.WithSampler(sdktrace.AlwaysSample())

# Never sample
# sdktrace.WithSampler(sdktrace.NeverSample())

# Probabilistic — sample 10%
# sdktrace.WithSampler(sdktrace.TraceIDRatioBased(0.1))

# Parent-based (respect upstream decision, fallback to ratio)
# sdktrace.WithSampler(sdktrace.ParentBased(
#     sdktrace.TraceIDRatioBased(0.1),
# ))

# Tail sampling in collector (sample errors + slow requests)
# processors:
#   tail_sampling:
#     decision_wait: 10s
#     policies:
#       - name: errors
#         type: status_code
#         status_code: {status_codes: [ERROR]}
#       - name: slow
#         type: latency
#         latency: {threshold_ms: 1000}
```

## Environment Variables

```bash
export OTEL_SERVICE_NAME="my-service"
export OTEL_EXPORTER_OTLP_ENDPOINT="http://localhost:4317"
export OTEL_EXPORTER_OTLP_PROTOCOL="grpc"       # grpc | http/protobuf
export OTEL_TRACES_SAMPLER="parentbased_traceidratio"
export OTEL_TRACES_SAMPLER_ARG="0.1"             # 10% sampling
export OTEL_RESOURCE_ATTRIBUTES="deployment.environment=prod,service.version=1.2.3"
export OTEL_LOG_LEVEL="info"
export OTEL_PROPAGATORS="tracecontext,baggage"
export OTEL_METRICS_EXPORTER="otlp"
export OTEL_LOGS_EXPORTER="otlp"
```

## Semantic Conventions

```bash
# Standard attribute keys (semconv)
# HTTP:  http.method, http.status_code, http.url, http.route
# DB:    db.system, db.statement, db.name, db.operation
# RPC:   rpc.system, rpc.service, rpc.method
# Net:   net.peer.name, net.peer.port, net.transport
# K8s:   k8s.pod.name, k8s.namespace.name, k8s.deployment.name
```

## Tips

- Use `ParentBased` sampler to respect upstream sampling decisions while setting your own default ratio
- Always call `span.End()` with defer immediately after `tracer.Start()` to avoid span leaks
- Set `memory_limiter` processor in the Collector to prevent OOM under load spikes
- Put `memory_limiter` first in the processor chain so it can drop data before other processors buffer it
- Use `batch` processor to reduce export overhead and network round trips
- Set `OTEL_SERVICE_NAME` env var as a fallback even when configuring in code
- Use tail sampling in the Collector (not head sampling in SDK) when you need error-based or latency-based decisions
- Filter health check spans in the Collector to reduce noise and storage costs
- Use semantic conventions for attribute names so dashboards and queries work across services
- Prefer gRPC OTLP exporter over HTTP for lower overhead in high-throughput services
- Run the Collector as a sidecar or daemonset, not as a single central gateway, for resilience
- Export resource attributes like `service.version` and `deployment.environment` for filtering in backends

## See Also

jaeger, prometheus, grafana, fluentbit, victoriametrics

## References

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [OpenTelemetry Collector Configuration](https://opentelemetry.io/docs/collector/configuration/)
- [Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [OTLP Specification](https://opentelemetry.io/docs/specs/otlp/)
