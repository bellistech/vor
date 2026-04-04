# Jaeger (Distributed Tracing)

Open-source distributed tracing platform for monitoring and troubleshooting microservices, providing request flow visualization, latency analysis, and root cause detection across service boundaries.

## Deployment

### All-in-one (development)

```bash
# Docker — all components in one container
docker run -d --name jaeger \
  -p 6831:6831/udp \
  -p 16686:16686 \
  -p 4317:4317 \
  -p 4318:4318 \
  jaegertracing/all-in-one:latest

# Port reference:
# 4317  — OTLP gRPC receiver
# 4318  — OTLP HTTP receiver
# 6831  — Jaeger Thrift compact (UDP, legacy)
# 16686 — Jaeger UI
# 14268 — Jaeger Thrift HTTP (legacy)
```

### Production (distributed)

```bash
# Collector — receives spans and writes to storage
docker run -d --name jaeger-collector \
  -p 4317:4317 -p 4318:4318 \
  -e SPAN_STORAGE_TYPE=elasticsearch \
  -e ES_SERVER_URLS=http://elasticsearch:9200 \
  jaegertracing/jaeger-collector:latest

# Query — serves the UI
docker run -d --name jaeger-query \
  -p 16686:16686 \
  -e SPAN_STORAGE_TYPE=elasticsearch \
  -e ES_SERVER_URLS=http://elasticsearch:9200 \
  jaegertracing/jaeger-query:latest

# Ingester — reads from Kafka and writes to storage
docker run -d --name jaeger-ingester \
  -e SPAN_STORAGE_TYPE=elasticsearch \
  -e KAFKA_CONSUMER_BROKERS=kafka:9092 \
  -e ES_SERVER_URLS=http://elasticsearch:9200 \
  jaegertracing/jaeger-ingester:latest
```

## Storage Backends

```bash
# Memory (development only, data lost on restart)
SPAN_STORAGE_TYPE=memory

# Elasticsearch / OpenSearch
SPAN_STORAGE_TYPE=elasticsearch
ES_SERVER_URLS=http://es1:9200,http://es2:9200
ES_INDEX_PREFIX=jaeger
ES_NUM_SHARDS=5
ES_NUM_REPLICAS=1

# Cassandra
SPAN_STORAGE_TYPE=cassandra
CASSANDRA_SERVERS=cass1,cass2,cass3
CASSANDRA_KEYSPACE=jaeger_v1_dc1

# Kafka (buffer between collector and storage)
SPAN_STORAGE_TYPE=kafka
KAFKA_PRODUCER_BROKERS=kafka:9092
KAFKA_TOPIC=jaeger-spans

# Badger (embedded, single-node)
SPAN_STORAGE_TYPE=badger
BADGER_DIRECTORY_VALUE=/data/values
BADGER_DIRECTORY_KEY=/data/keys
```

## Sampling Strategies

### Remote sampling configuration

```json
{
  "service_strategies": [
    {
      "service": "frontend",
      "type": "probabilistic",
      "param": 0.5
    },
    {
      "service": "payment",
      "type": "probabilistic",
      "param": 1.0
    },
    {
      "service": "catalog",
      "type": "ratelimiting",
      "param": 10
    }
  ],
  "default_strategy": {
    "type": "probabilistic",
    "param": 0.1
  }
}
```

### Sampling types

```bash
# const — sample all (1) or none (0)
# type: const, param: 1

# probabilistic — sample a percentage
# type: probabilistic, param: 0.1      # 10% of traces

# ratelimiting — sample N traces per second
# type: ratelimiting, param: 10         # 10 traces/sec

# remote — fetch strategy from collector
# type: remote                          # polls collector for config
```

## Jaeger Query API

```bash
# List services
curl http://localhost:16686/api/services

# List operations for a service
curl http://localhost:16686/api/services/frontend/operations

# Search traces
curl "http://localhost:16686/api/traces?service=frontend&limit=20"

# Search with filters
curl "http://localhost:16686/api/traces?\
service=frontend&\
operation=/api/users&\
start=$(date -d '1 hour ago' +%s)000000&\
end=$(date +%s)000000&\
minDuration=100ms&\
maxDuration=5s&\
tags={\"http.status_code\":\"500\"}&\
limit=50"

# Get a specific trace
curl http://localhost:16686/api/traces/<trace-id>

# Compare two traces
# UI: http://localhost:16686/trace/<id1>...<id2>
```

## Span Model

```bash
# A span contains:
# - TraceID    (128-bit, shared by all spans in a trace)
# - SpanID     (64-bit, unique per span)
# - ParentID   (64-bit, 0 for root span)
# - Operation  (name of the operation)
# - Service    (source service)
# - StartTime  (microsecond timestamp)
# - Duration   (microseconds)
# - Tags       (key-value metadata)
# - Logs       (timestamped events within span)
# - References (CHILD_OF or FOLLOWS_FROM)
# - Process    (service info, hostname, IP)
```

## Kubernetes Deployment

```yaml
# Jaeger Operator — simplest way to run in K8s
# kubectl create namespace observability
# kubectl apply -f https://github.com/jaegertracing/jaeger-operator/releases/latest/download/jaeger-operator.yaml -n observability

# Simple Jaeger instance
apiVersion: jaegertracing.io/v1
kind: Jaeger
metadata:
  name: production
  namespace: observability
spec:
  strategy: production          # allInOne | production | streaming
  storage:
    type: elasticsearch
    options:
      es:
        server-urls: https://elasticsearch:9200
        index-prefix: jaeger
    esIndexCleaner:
      enabled: true
      numberOfDays: 14          # retain 14 days
      schedule: "55 23 * * *"   # run daily at 23:55
  collector:
    replicas: 2
    maxReplicas: 5
    resources:
      limits:
        cpu: "1"
        memory: 1Gi
  query:
    replicas: 2
```

## SPM (Service Performance Monitoring)

```bash
# Enable Service Performance Monitoring in Jaeger
# Requires Prometheus metrics from spans

# Collector flags
--prometheus.server-url=http://prometheus:9090
--prometheus.query.support-spanmetrics-connector=true

# Generates RED metrics from spans:
#   Rate    — requests per second
#   Errors  — error rate
#   Duration — latency percentiles (p50, p95, p99)
```

## Index Management

```bash
# Elasticsearch index cleanup
# Jaeger creates daily indices: jaeger-span-YYYY-MM-DD
docker run --rm \
  -e ROLLOVER=1 \
  jaegertracing/jaeger-es-index-cleaner:latest \
  14 \
  http://elasticsearch:9200

# Cassandra schema migration
docker run --rm \
  -e CASSANDRA_SERVERS=cass1 \
  -e CASSANDRA_KEYSPACE=jaeger_v1_dc1 \
  jaegertracing/jaeger-cassandra-schema:latest
```

## Tips

- Use the OTLP receiver (ports 4317/4318) instead of legacy Jaeger protocols for new deployments
- Set index rollover with daily indices and a cleanup job to control Elasticsearch storage growth
- Use Kafka as a buffer between collectors and storage to absorb traffic spikes without losing spans
- Configure per-service sampling strategies to sample critical paths (payments) at 100% and high-volume paths lower
- Use rate-limiting sampling for services with unpredictable traffic to cap storage costs
- Tag spans with business attributes (order ID, user tier) to enable targeted trace searches
- Enable SPM (Service Performance Monitoring) to get RED metrics directly from trace data
- Set `ES_NUM_SHARDS` based on your data volume; too many shards on small clusters hurts performance
- Use the Jaeger Operator in Kubernetes for simplified lifecycle management and auto-scaling
- Compare traces side-by-side in the UI to debug regressions between deployments
- Set reasonable `maxDuration` filters when searching to avoid pulling massive trace trees

## See Also

opentelemetry, prometheus, grafana, elasticsearch, kafka

## References

- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [Jaeger Architecture](https://www.jaegertracing.io/docs/architecture/)
- [Jaeger Operator for Kubernetes](https://github.com/jaegertracing/jaeger-operator)
- [OpenTelemetry to Jaeger Migration](https://www.jaegertracing.io/docs/apis/#opentelemetry-protocol)
- [Jaeger Sampling](https://www.jaegertracing.io/docs/sampling/)
