# Loki (Log Aggregation System)

Grafana Loki is a horizontally scalable, multi-tenant log aggregation system that indexes only metadata labels rather than full log content, using LogQL for querying, with storage backends on object stores like S3 or GCS, and tight integration with Grafana for visualization.

## Architecture

```text
# Loki components (monolithic or microservices mode)
#
# Write path:  Agent -> Distributor -> Ingester -> Storage
# Read path:   Query Frontend -> Querier -> Ingester + Storage
# Background:  Compactor (index dedup, retention)
#
# Deployment modes:
#   monolithic    — all components in one process (small scale)
#   simple-scalable — read, write, backend targets (medium)
#   microservices — each component independent (large scale)
```

## Installation

```bash
# Docker (monolithic)
docker run -d --name loki \
  -p 3100:3100 \
  -v loki-config:/etc/loki \
  -v loki-data:/loki \
  grafana/loki:latest -config.file=/etc/loki/local-config.yaml

# Helm chart (Kubernetes)
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update
helm install loki grafana/loki-stack \
  --namespace monitoring --create-namespace \
  --set promtail.enabled=true \
  --set grafana.enabled=true

# Binary
wget https://github.com/grafana/loki/releases/latest/download/loki-linux-amd64.zip
unzip loki-linux-amd64.zip
chmod +x loki-linux-amd64
./loki-linux-amd64 -config.file=loki-config.yaml

# LogCLI (command-line query tool)
wget https://github.com/grafana/loki/releases/latest/download/logcli-linux-amd64.zip
unzip logcli-linux-amd64.zip
chmod +x logcli-linux-amd64
sudo mv logcli-linux-amd64 /usr/local/bin/logcli
```

## Loki Configuration

```yaml
# loki-config.yaml (simple scalable)
auth_enabled: false

server:
  http_listen_port: 3100
  grpc_listen_port: 9096

common:
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

schema_config:
  configs:
    - from: 2024-01-01
      store: tsdb
      object_store: filesystem
      schema: v13
      index:
        prefix: index_
        period: 24h

storage_config:
  # For S3 backend:
  # aws:
  #   s3: s3://us-east-1/loki-chunks
  #   bucketnames: loki-chunks
  filesystem:
    directory: /loki/chunks

limits_config:
  reject_old_samples: true
  reject_old_samples_max_age: 168h      # 7 days
  max_entries_limit_per_query: 5000
  ingestion_rate_mb: 10
  ingestion_burst_size_mb: 20

compactor:
  working_directory: /loki/compactor
  compaction_interval: 10m
  retention_enabled: true
  retention_delete_delay: 2h
  retention_delete_worker_count: 150

# Per-tenant retention
# limits_config:
#   retention_period: 720h               # 30 days default
```

## LogQL (Query Language)

### Stream Selectors

```logql
# Basic label matching
{job="nginx"}
{job="nginx", env="production"}
{job=~"nginx|apache"}                   # regex match
{job!="debug"}                          # not equal
{namespace=~"prod-.*"}                  # regex prefix
```

### Filter Expressions

```logql
# Line contains
{job="nginx"} |= "error"

# Line does not contain
{job="nginx"} != "healthcheck"

# Regex match
{job="nginx"} |~ "status=[45]\\d{2}"

# Regex not match
{job="nginx"} !~ "GET /health"

# Chain multiple filters
{job="nginx"} |= "error" != "timeout" |~ "5\\d{2}"
```

### Parsers

```logql
# JSON parser
{job="app"} | json

# JSON with specific fields
{job="app"} | json level="level", msg="message"

# Logfmt parser
{job="app"} | logfmt

# Regex parser
{job="nginx"} | regexp `(?P<ip>\S+) \S+ \S+ \[(?P<ts>[^\]]+)\] "(?P<method>\S+) (?P<path>\S+).*" (?P<status>\d+) (?P<size>\d+)`

# Pattern parser (simpler than regex)
{job="nginx"} | pattern `<ip> - - [<ts>] "<method> <path> <_>" <status> <size>`

# Label filter after parsing
{job="nginx"} | json | status >= 400
{job="app"} | logfmt | level="error" | duration > 5s
```

### Metric Queries

```logql
# Count log lines per second
rate({job="nginx"}[5m])

# Count errors per minute
sum(rate({job="nginx"} |= "error" [1m])) by (host)

# Bytes rate
bytes_rate({job="nginx"}[5m])

# Count over time
count_over_time({job="app"} |= "error" [1h])

# Quantile of extracted values
quantile_over_time(0.95, {job="app"} | json | unwrap duration [5m]) by (endpoint)

# Top 10 endpoints by error rate
topk(10, sum(rate({job="nginx"} | json | status >= 500 [5m])) by (path))

# Absent (alert when no logs)
absent_over_time({job="critical-service"}[15m])
```

## LogCLI Usage

```bash
# Set environment
export LOKI_ADDR=http://localhost:3100

# Query logs
logcli query '{job="nginx"}'
logcli query '{job="nginx"} |= "error"' --limit=100

# Live tail
logcli query '{job="nginx"}' --tail

# Time range
logcli query '{job="nginx"}' --from="2024-01-01T00:00:00Z" --to="2024-01-02T00:00:00Z"

# Metric query
logcli query 'rate({job="nginx"}[5m])'

# Output formats
logcli query '{job="nginx"}' --output=jsonl
logcli query '{job="nginx"}' --output=raw

# List labels and values
logcli labels
logcli labels job
logcli series '{job="nginx"}'
```

## Promtail Configuration

```yaml
# promtail-config.yaml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: system
    static_configs:
      - targets: [localhost]
        labels:
          job: syslog
          host: myhost
          __path__: /var/log/syslog

  - job_name: nginx
    static_configs:
      - targets: [localhost]
        labels:
          job: nginx
          __path__: /var/log/nginx/*.log

    pipeline_stages:
      - regex:
          expression: '(?P<ip>\S+) .* "(?P<method>\S+) (?P<path>\S+) .*" (?P<status>\d+)'
      - labels:
          method:
          status:
      - metrics:
          http_requests_total:
            type: Counter
            source: status
            description: "Total HTTP requests"
            config:
              action: inc
```

## Grafana Alloy Configuration

```hcl
// alloy config for Loki
loki.source.file "logs" {
  targets    = [
    {__path__ = "/var/log/*.log", job = "system", host = "myhost"},
    {__path__ = "/var/log/nginx/*.log", job = "nginx"},
  ]
  forward_to = [loki.write.default.receiver]
}

loki.write "default" {
  endpoint {
    url = "http://loki:3100/loki/api/v1/push"
  }
}
```

## Loki HTTP API

```bash
# Push logs
curl -X POST http://localhost:3100/loki/api/v1/push \
  -H "Content-Type: application/json" \
  -d '{
    "streams": [{
      "stream": {"job": "test", "level": "info"},
      "values": [
        ["'$(date +%s)000000000'", "test log message"]
      ]
    }]
  }'

# Query logs
curl -G http://localhost:3100/loki/api/v1/query_range \
  --data-urlencode 'query={job="nginx"}' \
  --data-urlencode 'start=1704067200' \
  --data-urlencode 'end=1704153600' \
  --data-urlencode 'limit=100' | jq .

# Get labels
curl http://localhost:3100/loki/api/v1/labels | jq .

# Get label values
curl http://localhost:3100/loki/api/v1/label/job/values | jq .

# Ready check
curl http://localhost:3100/ready

# Metrics
curl http://localhost:3100/metrics
```

## Tips

- Keep label cardinality low; high cardinality (user IDs, request IDs) as labels destroys performance
- Use filter expressions and parsers to extract fields at query time instead of adding more labels
- `|=` (contains) is faster than `|~` (regex); prefer exact string matching when possible
- Use `rate()` and `count_over_time()` for alerting instead of raw log queries
- Set `reject_old_samples_max_age` to prevent late-arriving logs from corrupting older chunks
- The compactor handles retention deletion; without it, data accumulates indefinitely
- Use `logcli` for ad-hoc queries and debugging; it is faster than waiting for Grafana dashboards
- Split Loki into read/write/backend targets for horizontal scaling before going full microservices
- Structured logging (JSON/logfmt) is far more queryable than unstructured text
- Use `absent_over_time()` in alerts to detect when expected log streams disappear
- Chunk size and flush intervals affect both ingest performance and query latency; tune for your volume

## See Also

- prometheus, grafana, thanos, alertmanager

## References

- [Loki Documentation](https://grafana.com/docs/loki/latest/)
- [LogQL Reference](https://grafana.com/docs/loki/latest/query/)
- [Loki Best Practices](https://grafana.com/docs/loki/latest/best-practices/)
- [Promtail Configuration](https://grafana.com/docs/loki/latest/send-data/promtail/configuration/)
- [Grafana Alloy Documentation](https://grafana.com/docs/alloy/latest/)
