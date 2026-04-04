# Thanos (Global Prometheus at Scale)

Thanos extends Prometheus with long-term storage on object stores, a global query view across multiple Prometheus instances, data deduplication, downsampling for efficient historical queries, and distributed rule evaluation for scalable alerting.

## Architecture Components

```text
# Thanos component topology
#
# Prometheus + Sidecar ──> Object Storage (S3/GCS/Azure)
#                    │
# Prometheus + Sidecar ──> Object Storage
#        │                       │
#   Query ──────> Store Gateway ─┘
#     │
#   Query Frontend (caching, splitting)
#     │
#   Compactor (compaction, downsampling, retention)
#     │
#   Ruler (global recording/alerting rules)
#
# Alternative ingest: Receive (remote-write target, no sidecar needed)
```

## Sidecar (Per-Prometheus)

```yaml
# Run alongside each Prometheus instance
# Uploads TSDB blocks to object storage
# Serves real-time data to Query via Store API

# prometheus.yml (must have external_labels for dedup)
global:
  external_labels:
    cluster: us-east-1
    replica: prometheus-0
```

```bash
# Run sidecar
thanos sidecar \
  --tsdb.path=/prometheus/data \
  --prometheus.url=http://localhost:9090 \
  --objstore.config-file=bucket.yaml \
  --grpc-address=0.0.0.0:10901 \
  --http-address=0.0.0.0:10902
```

## Receive (Remote-Write Target)

```bash
# Alternative to sidecar: Prometheus remote-writes to Receive
thanos receive \
  --tsdb.path=/thanos/receive \
  --grpc-address=0.0.0.0:10901 \
  --http-address=0.0.0.0:10902 \
  --remote-write.address=0.0.0.0:19291 \
  --objstore.config-file=bucket.yaml \
  --label="receive_replica=\"$(hostname)\"" \
  --tsdb.retention=6h
```

```yaml
# Prometheus remote_write config
remote_write:
  - url: http://thanos-receive:19291/api/v1/receive
    queue_config:
      max_samples_per_send: 1000
      batch_send_deadline: 10s
      max_shards: 30
```

## Query (Global View)

```bash
# Query federates across all stores (sidecars, store gateways, receivers)
thanos query \
  --http-address=0.0.0.0:9090 \
  --grpc-address=0.0.0.0:10901 \
  --store=sidecar-1:10901 \
  --store=sidecar-2:10901 \
  --store=store-gateway:10901 \
  --store=ruler:10901 \
  --store.sd-dns-interval=30s \
  --query.replica-label=replica \
  --query.auto-downsampling

# Service discovery for stores (DNS-based)
thanos query \
  --http-address=0.0.0.0:9090 \
  --store=dnssrv+_grpc._tcp.thanos-store.monitoring.svc

# Access Thanos Query UI at http://localhost:9090
# Fully compatible with PromQL
```

## Query Frontend (Caching + Splitting)

```bash
# Sits in front of Query, provides caching and query splitting
thanos query-frontend \
  --http-address=0.0.0.0:9090 \
  --query-frontend.downstream-url=http://thanos-query:9090 \
  --query-range.split-interval=24h \
  --query-range.max-retries-per-request=3 \
  --labels.split-interval=24h \
  --query-frontend.log-queries-longer-than=10s

# Memcached caching
thanos query-frontend \
  --http-address=0.0.0.0:9090 \
  --query-frontend.downstream-url=http://thanos-query:9090 \
  --query-range.response-cache-config="type: MEMCACHED
config:
  addresses: memcached:11211
  max_idle_connections: 100
  timeout: 500ms"
```

## Store Gateway (Object Storage Reader)

```bash
# Serves historical data from object storage
thanos store \
  --data-dir=/thanos/store \
  --objstore.config-file=bucket.yaml \
  --grpc-address=0.0.0.0:10901 \
  --http-address=0.0.0.0:10902 \
  --index-cache-size=1GB \
  --chunk-pool-size=4GB

# With index cache (memcached)
thanos store \
  --data-dir=/thanos/store \
  --objstore.config-file=bucket.yaml \
  --index-cache.config="type: MEMCACHED
config:
  addresses: memcached:11211
  max_item_size: 5MB"
```

## Compactor

```bash
# Compacts TSDB blocks, applies downsampling, enforces retention
thanos compact \
  --data-dir=/thanos/compact \
  --objstore.config-file=bucket.yaml \
  --http-address=0.0.0.0:10902 \
  --retention.resolution-raw=30d \
  --retention.resolution-5m=180d \
  --retention.resolution-1h=365d \
  --compact.concurrency=4 \
  --downsample.concurrency=4 \
  --wait                                 # run continuously (not one-shot)

# Downsampling creates aggregated blocks:
#   Raw -> 5m resolution (after 40h)
#   5m  -> 1h resolution (after 10d)
```

## Ruler (Global Rules)

```bash
# Evaluates recording and alerting rules against global view
thanos rule \
  --data-dir=/thanos/ruler \
  --objstore.config-file=bucket.yaml \
  --grpc-address=0.0.0.0:10901 \
  --http-address=0.0.0.0:10902 \
  --query=thanos-query:9090 \
  --alertmanagers.url=http://alertmanager:9093 \
  --rule-file=/etc/thanos/rules/*.yaml \
  --label="ruler_cluster=\"global\""
```

```yaml
# /etc/thanos/rules/global-rules.yaml
groups:
  - name: global-recording
    interval: 1m
    rules:
      - record: cluster:http_requests:rate5m
        expr: sum(rate(http_requests_total[5m])) by (cluster)

  - name: global-alerting
    rules:
      - alert: GlobalHighErrorRate
        expr: |
          sum(rate(http_requests_total{status=~"5.."}[5m])) by (cluster)
          / sum(rate(http_requests_total[5m])) by (cluster) > 0.05
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "High error rate in cluster {{ $labels.cluster }}"
```

## Object Storage Configuration

```yaml
# bucket.yaml — S3
type: S3
config:
  bucket: thanos-metrics
  endpoint: s3.us-east-1.amazonaws.com
  region: us-east-1
  access_key: ${AWS_ACCESS_KEY_ID}
  secret_key: ${AWS_SECRET_ACCESS_KEY}

# bucket.yaml — GCS
type: GCS
config:
  bucket: thanos-metrics

# bucket.yaml — Azure
type: AZURE
config:
  storage_account: thanosmetrics
  storage_account_key: ${AZURE_STORAGE_KEY}
  container: thanos

# bucket.yaml — MinIO (S3-compatible)
type: S3
config:
  bucket: thanos
  endpoint: minio:9000
  access_key: minio
  secret_key: minio123
  insecure: true
```

## Deduplication

```yaml
# Dedup requires consistent external_labels across replicas
# Each Prometheus sets a unique replica label:
global:
  external_labels:
    cluster: us-east-1
    replica: prometheus-0    # unique per replica

# Query deduplicates using --query.replica-label
# thanos query --query.replica-label=replica

# Compactor also deduplicates during compaction:
# thanos compact --deduplication.replica-label=replica
```

## Kubernetes Deployment (Helm)

```bash
# Using bitnami chart
helm repo add bitnami https://charts.bitnami.com/bitnami
helm install thanos bitnami/thanos \
  --namespace monitoring \
  --set objstoreConfig="$(cat bucket.yaml)" \
  --set query.replicaLabel=replica \
  --set compactor.retentionResolutionRaw=30d \
  --set compactor.retentionResolution5m=180d \
  --set compactor.retentionResolution1h=365d

# Using kube-thanos manifests
git clone https://github.com/thanos-io/kube-thanos.git
cd kube-thanos
kubectl apply -f manifests/
```

## Querying via PromQL

```bash
# Thanos Query is PromQL-compatible
# Access via http://thanos-query:9090

# Cross-cluster query
curl -G http://thanos-query:9090/api/v1/query \
  --data-urlencode 'query=sum(rate(http_requests_total[5m])) by (cluster)'

# Range query with downsampling
curl -G http://thanos-query:9090/api/v1/query_range \
  --data-urlencode 'query=avg(node_cpu_seconds_total{mode="idle"})' \
  --data-urlencode 'start=2024-01-01T00:00:00Z' \
  --data-urlencode 'end=2024-01-31T00:00:00Z' \
  --data-urlencode 'step=1h' \
  --data-urlencode 'max_source_resolution=1h'

# Check stores
curl http://thanos-query:9090/api/v1/stores | jq .
```

## Health and Metrics

```bash
# All components expose /metrics for Prometheus scraping
# Key metrics:
# thanos_objstore_bucket_operations_total         — object store ops
# thanos_objstore_bucket_operation_duration_seconds — latency
# thanos_store_series_gate_queries_concurrent      — concurrent queries
# thanos_compact_group_compactions_total           — compaction progress
# thanos_query_store_apis_dns_lookups_total        — store discovery

# Health endpoints
curl http://thanos-sidecar:10902/-/healthy
curl http://thanos-query:9090/-/healthy
curl http://thanos-store:10902/-/ready
```

## Tips

- Always set unique `external_labels` on each Prometheus instance; without them, deduplication cannot work
- Use Query Frontend with caching (memcached or Redis) to reduce load on Store Gateway for repeated queries
- The Compactor must run as a singleton; running multiple compactors causes data corruption
- Enable auto-downsampling on Query (`--query.auto-downsampling`) for automatic resolution selection on long ranges
- Set retention per resolution: keep raw data short (30d), 5m medium (180d), 1h long (1y+)
- Use `--store.sd-dns-interval` with DNS-based service discovery for dynamic store registration in Kubernetes
- Receive mode is simpler than Sidecar when Prometheus cannot reach object storage directly
- Store Gateway's index cache is critical for query performance; size it generously (1-2 GB minimum)
- The Ruler evaluates rules against the global view; use it for cross-cluster alerts that single Prometheus cannot compute
- Monitor `thanos_objstore_bucket_operation_failures_total` to catch storage backend issues early

## See Also

- prometheus, grafana, loki, alertmanager, victoriametrics

## References

- [Thanos Documentation](https://thanos.io/tip/thanos/getting-started.md/)
- [Thanos Components](https://thanos.io/tip/components/)
- [Thanos GitHub](https://github.com/thanos-io/thanos)
- [kube-thanos (Kubernetes Manifests)](https://github.com/thanos-io/kube-thanos)
- [Thanos Design Documents](https://thanos.io/tip/proposals-done/)
