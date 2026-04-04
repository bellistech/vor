# Grafana Tempo (Distributed Tracing)

Open-source, cost-effective distributed tracing backend that requires only object storage (S3/GCS/Azure) and scales horizontally.

## Architecture

### Core Components

```
                  ┌─────────────┐
    traces ──────>│ Distributor  │──────> Ingester ──────> Object Storage
                  └─────────────┘           │                   │
                                            v                   v
                  ┌─────────────┐     ┌──────────┐      ┌────────────┐
    queries ─────>│   Querier   │<────│ Ingester │      │ Compactor  │
                  └─────────────┘     └──────────┘      └────────────┘
                        │                                      │
                        └──────── Query Frontend <─────────────┘
```

### Component Roles

```bash
# Distributor — receives spans (OTLP, Jaeger, Zipkin), hashes trace ID, forwards to ingesters
# Ingester    — batches spans into blocks, writes to object storage
# Compactor   — merges small blocks into larger ones, enforces retention
# Querier     — searches ingesters (recent) + object storage (historical)
# Query Frontend — splits/caches/retries queries, serves the search API
# Metrics Generator — derives RED metrics from ingested spans
```

## Installation

### Docker Compose (quickstart)

```bash
# Clone the examples repo
git clone https://github.com/grafana/tempo.git
cd tempo/example/docker-compose/local

# Start Tempo + Grafana + k6-tracing load generator
docker compose up -d
```

### Helm Chart (Kubernetes)

```bash
# Add the Grafana Helm repo
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install Tempo distributed (microservices mode)
helm install tempo grafana/tempo-distributed -n tracing --create-namespace

# Install Tempo monolithic (single binary)
helm install tempo grafana/tempo -n tracing --create-namespace
```

### Single Binary

```bash
# Download latest release
curl -Lo tempo https://github.com/grafana/tempo/releases/latest/download/tempo_$(uname -s | tr A-Z a-z)_amd64
chmod +x tempo

# Run with config
./tempo -config.file=tempo.yaml
```

## Configuration

### Minimal Config (tempo.yaml)

```yaml
server:
  http_listen_port: 3200

distributor:
  receivers:
    otlp:
      protocols:
        grpc:
          endpoint: 0.0.0.0:4317
        http:
          endpoint: 0.0.0.0:4318
    jaeger:
      protocols:
        thrift_http:
          endpoint: 0.0.0.0:14268

storage:
  trace:
    backend: local                    # local, s3, gcs, azure
    local:
      path: /var/tempo/traces
    wal:
      path: /var/tempo/wal
    block:
      bloom_filter_false_positive: 0.05

compactor:
  compaction:
    block_retention: 336h             # 14 days
```

### S3 Backend

```yaml
storage:
  trace:
    backend: s3
    s3:
      bucket: tempo-traces
      endpoint: s3.us-east-1.amazonaws.com
      region: us-east-1
      # access_key and secret_key via env vars or IAM role
```

### GCS Backend

```yaml
storage:
  trace:
    backend: gcs
    gcs:
      bucket_name: tempo-traces
      # Uses Application Default Credentials
```

## TraceQL

### Basic Queries

```bash
# Find traces by service name
{ resource.service.name = "frontend" }

# Filter by span name
{ name = "HTTP GET" }

# Filter by status
{ status = error }

# Filter by duration (spans over 500ms)
{ duration > 500ms }

# Filter by attribute
{ span.http.status_code >= 400 }

# Combine conditions (same span)
{ span.http.method = "POST" && duration > 1s }

# Combine conditions (different spans in same trace — pipeline)
{ resource.service.name = "frontend" } | { status = error }
```

### Advanced TraceQL

```bash
# Count spans matching a condition
{ resource.service.name = "api" } | count() > 5

# Aggregate span durations
{ resource.service.name = "database" } | avg(duration) > 100ms

# Select specific fields
{ resource.service.name = "frontend" } | select(span.http.url, duration)

# Structural operators — find parent-child relationships
{ resource.service.name = "frontend" } >> { resource.service.name = "backend" }

# Sibling spans
{ name = "auth" } ~ { name = "fetch-data" }

# Regex match
{ span.http.url =~ "/api/v[12]/users.*" }
```

## Ingesting Traces

### OpenTelemetry Collector Config

```yaml
# otel-collector-config.yaml
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317

exporters:
  otlp/tempo:
    endpoint: tempo:4317
    tls:
      insecure: true

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlp/tempo]
```

### Verify Ingestion

```bash
# Check distributor is receiving spans
curl -s http://localhost:3200/metrics | grep tempo_distributor_spans_received_total
```

## Sampling Strategies

### Head Sampling (at the SDK/Collector)

```yaml
# otel-collector — probabilistic sampler
processors:
  probabilistic_sampler:
    sampling_percentage: 10            # keep 10% of traces

service:
  pipelines:
    traces:
      processors: [probabilistic_sampler]
```

### Tail Sampling (at the Collector)

```yaml
# otel-collector — tail_sampling processor
processors:
  tail_sampling:
    decision_wait: 10s
    policies:
      - name: errors
        type: status_code
        status_code: {status_codes: [ERROR]}
      - name: slow-requests
        type: latency
        latency: {threshold_ms: 1000}
      - name: sample-rest
        type: probabilistic
        probabilistic: {sampling_percentage: 5}
```

## Span Metrics (Metrics Generator)

```yaml
# Enable in tempo.yaml
metrics_generator:
  registry:
    external_labels:
      source: tempo
  storage:
    path: /var/tempo/generator/wal
    remote_write:
      - url: http://prometheus:9090/api/v1/write
  processor:
    service_graphs:
      enabled: true
    span_metrics:
      enabled: true
      dimensions:
        - http.method
        - http.status_code
```

## Querying via API

```bash
# Search traces by tags
curl -G http://localhost:3200/api/search \
  --data-urlencode 'q={resource.service.name="frontend"}' \
  --data-urlencode 'limit=20'

# Get a specific trace by ID
curl http://localhost:3200/api/traces/01020304050607080102030405060708

# Check Tempo readiness
curl http://localhost:3200/ready

# Flush ingesters (force write to backend)
curl -X POST http://localhost:3200/flush

# Get build info
curl http://localhost:3200/api/status/buildinfo
```

## Grafana Integration

### Add Tempo Datasource

```yaml
# grafana/provisioning/datasources/tempo.yaml
apiVersion: 1
datasources:
  - name: Tempo
    type: tempo
    access: proxy
    url: http://tempo:3200
    jsonData:
      tracesToMetrics:
        datasourceUid: prometheus
      tracesToLogs:
        datasourceUid: loki
        filterByTraceID: true
      nodeGraph:
        enabled: true
      serviceMap:
        datasourceUid: prometheus
```

## Tips

- Deploy in microservices mode for production; monolithic mode is fine for dev and small clusters
- Set `bloom_filter_false_positive` to 0.01 for large deployments to reduce query time at the cost of more storage
- Use tail sampling at the collector layer to keep error and slow traces while sampling the rest
- Enable the metrics generator to get RED metrics (rate, errors, duration) without a separate pipeline
- Connect Tempo to Loki (traces-to-logs) and Prometheus (traces-to-metrics) for full observability
- Tune `max_bytes_per_trace` to prevent a single runaway trace from consuming excessive resources
- Use `query_frontend.search.max_duration` to limit how far back TraceQL searches can go
- Compactor block retention should match your data retention policy to avoid orphaned blocks
- Set `ingester.lifecycler.ring.replication_factor` to 3 in production for durability
- Enable multi-tenancy via the `X-Scope-OrgID` header when serving multiple teams
- Use the service graph feature to auto-generate dependency maps from trace data
- Monitor `/metrics` endpoint for `tempo_ingester_traces_created_total` and `tempo_discarded_spans_total`

## See Also

- opentelemetry
- jaeger
- grafana
- prometheus
- loki

## References

- [Grafana Tempo Documentation](https://grafana.com/docs/tempo/latest/)
- [TraceQL Reference](https://grafana.com/docs/tempo/latest/traceql/)
- [Tempo GitHub Repository](https://github.com/grafana/tempo)
- [Tempo Architecture](https://grafana.com/docs/tempo/latest/operations/architecture/)
- [OpenTelemetry Collector Exporters](https://opentelemetry.io/docs/collector/configuration/#exporters)
- [Grafana Tempo Helm Charts](https://github.com/grafana/helm-charts/tree/main/charts/tempo-distributed)
