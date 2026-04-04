# Grafana Mimir (Long-Term Metrics Storage)

Horizontally scalable, multi-tenant TSDB for long-term Prometheus metrics storage, successor to Cortex.

## Architecture

### Write Path

```
Prometheus ──> Distributor ──> Ingester ──> Object Storage (S3/GCS)
                  │                │
                  │                └── WAL (local disk)
                  └── Hash ring (consistent hashing)
```

### Read Path

```
Grafana/PromQL ──> Query Frontend ──> Querier ──> Ingester (recent)
                       │                 └──────> Store Gateway (historical)
                       └── Query Scheduler
```

### Backend Path

```
Object Storage <── Compactor (merges blocks, deduplicates)
       │
       └──> Store Gateway (indexes + caches blocks for queries)
```

## Installation

### Helm Chart (Kubernetes)

```bash
# Add the Grafana Helm repo
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install Mimir distributed
helm install mimir grafana/mimir-distributed -n mimir --create-namespace \
  --set minio.enabled=true

# Install with custom values
helm install mimir grafana/mimir-distributed -n mimir --create-namespace \
  -f custom-values.yaml
```

### Single Binary (development)

```bash
# Download latest release
curl -Lo mimir https://github.com/grafana/mimir/releases/latest/download/mimir-linux-amd64
chmod +x mimir

# Run with config
./mimir -config.file=mimir.yaml -target=all
```

### Docker

```bash
# Run monolithic mode
docker run -d --name mimir \
  -v $(pwd)/mimir.yaml:/etc/mimir/mimir.yaml \
  -p 9009:9009 \
  grafana/mimir:latest \
  -config.file=/etc/mimir/mimir.yaml
```

## Configuration

### Minimal Config (mimir.yaml)

```yaml
target: all                            # monolithic mode

multitenancy_enabled: false            # disable for single-tenant

server:
  http_listen_port: 9009
  grpc_listen_port: 9095

distributor:
  ring:
    kvstore:
      store: memberlist

ingester:
  ring:
    kvstore:
      store: memberlist
    replication_factor: 3

blocks_storage:
  backend: s3
  s3:
    endpoint: minio:9000
    bucket_name: mimir-blocks
    access_key_id: mimir
    secret_access_key: supersecret
    insecure: true
  tsdb:
    dir: /data/ingester
  bucket_store:
    sync_dir: /data/bucket-sync

compactor:
  data_dir: /data/compactor
  sharding_ring:
    kvstore:
      store: memberlist

store_gateway:
  sharding_ring:
    kvstore:
      store: memberlist

ruler_storage:
  backend: s3
  s3:
    endpoint: minio:9000
    bucket_name: mimir-ruler
    access_key_id: mimir
    secret_access_key: supersecret
    insecure: true

memberlist:
  join_members: [mimir-memberlist:7946]
```

### Multi-Tenant Configuration

```yaml
multitenancy_enabled: true

limits:
  max_global_series_per_user: 1500000
  max_global_series_per_metric: 50000
  ingestion_rate: 100000               # samples/sec per tenant
  ingestion_burst_size: 500000
  max_label_names_per_series: 30
  max_label_value_length: 2048
  compactor_blocks_retention_period: 365d
```

## Remote Write (Prometheus to Mimir)

### Prometheus Config

```yaml
# prometheus.yml
remote_write:
  - url: http://mimir:9009/api/v1/push
    headers:
      X-Scope-OrgID: tenant-1         # required if multi-tenant
    queue_config:
      max_samples_per_send: 1000
      batch_send_deadline: 5s
      min_backoff: 30ms
      max_backoff: 5s
```

### Grafana Agent / Alloy

```bash
# grafana-agent flow config
prometheus.remote_write "mimir" {
  endpoint {
    url = "http://mimir:9009/api/v1/push"
    headers = {
      "X-Scope-OrgID" = "tenant-1",
    }
  }
}
```

## Ruler (Recording Rules & Alerts)

### Configure Rules Backend

```yaml
ruler:
  rule_path: /data/ruler
  alertmanager_url: http://alertmanager:9093
  ring:
    kvstore:
      store: memberlist
```

### Upload Rules via API

```bash
# Create a rule namespace
curl -X POST http://mimir:9009/prometheus/config/v1/rules/my-namespace \
  -H "Content-Type: application/yaml" \
  -H "X-Scope-OrgID: tenant-1" \
  -d '
groups:
  - name: my-rules
    rules:
      - record: job:http_requests:rate5m
        expr: sum(rate(http_requests_total[5m])) by (job)
      - alert: HighErrorRate
        expr: sum(rate(http_requests_total{status=~"5.."}[5m])) > 0.1
        for: 5m
        labels:
          severity: critical
'

# List rule namespaces
curl http://mimir:9009/prometheus/config/v1/rules \
  -H "X-Scope-OrgID: tenant-1"

# Delete a rule namespace
curl -X DELETE http://mimir:9009/prometheus/config/v1/rules/my-namespace \
  -H "X-Scope-OrgID: tenant-1"
```

## API Operations

```bash
# Query (PromQL)
curl -G http://mimir:9009/prometheus/api/v1/query \
  -H "X-Scope-OrgID: tenant-1" \
  --data-urlencode 'query=up'

# Range query
curl -G http://mimir:9009/prometheus/api/v1/query_range \
  -H "X-Scope-OrgID: tenant-1" \
  --data-urlencode 'query=rate(http_requests_total[5m])' \
  --data-urlencode 'start=2024-01-01T00:00:00Z' \
  --data-urlencode 'end=2024-01-01T01:00:00Z' \
  --data-urlencode 'step=60'

# Label names
curl http://mimir:9009/prometheus/api/v1/labels \
  -H "X-Scope-OrgID: tenant-1"

# Series metadata
curl -G http://mimir:9009/prometheus/api/v1/series \
  -H "X-Scope-OrgID: tenant-1" \
  --data-urlencode 'match[]={__name__=~"http_.*"}'

# Check readiness
curl http://mimir:9009/ready

# Ring status
curl http://mimir:9009/distributor/ring
curl http://mimir:9009/ingester/ring
curl http://mimir:9009/store-gateway/ring
curl http://mimir:9009/compactor/ring
```

## Compactor

```yaml
compactor:
  data_dir: /data/compactor
  compaction_interval: 1h
  cleanup_interval: 15m
  tenant_cleanup_delay: 6h
  deletion_delay: 12h                  # wait before physically deleting blocks
  sharding_ring:
    kvstore:
      store: memberlist
```

## Memberlist (Gossip Ring)

```yaml
memberlist:
  join_members:
    - mimir-gossip-ring.mimir.svc.cluster.local:7946
  bind_port: 7946
  abort_if_cluster_join_fails: false
  rejoin_interval: 30s
```

## Grafana Datasource

```yaml
# grafana/provisioning/datasources/mimir.yaml
apiVersion: 1
datasources:
  - name: Mimir
    type: prometheus
    access: proxy
    url: http://mimir:9009/prometheus
    jsonData:
      httpHeaderName1: X-Scope-OrgID
    secureJsonData:
      httpHeaderValue1: tenant-1
```

## Migration from Cortex

```bash
# Mimir is backward-compatible with Cortex configs
# 1. Replace cortex binary with mimir
# 2. Rename config keys (most are identical)
# 3. Key renames:
#    - cortex.storage.engine → removed (TSDB only)
#    - cortex.chunk_store   → removed (blocks only)
#    - cortex.schema        → removed

# Block format is compatible — no data migration needed
# Existing blocks in S3/GCS work with Mimir as-is
```

## Tips

- Start with monolithic mode (`-target=all`) for development, split into microservices for production
- Set `ingestion_rate` and `max_global_series_per_user` per tenant to prevent noisy neighbors
- Use memberlist for ring coordination instead of Consul/etcd to reduce infrastructure dependencies
- The compactor must run as a singleton or with sharding enabled; running multiple unsharded compactors corrupts data
- Enable query sharding (`query_frontend.parallelize_shardable_queries: true`) for 10-20x faster queries
- Set `compactor_blocks_retention_period` per tenant to control long-term storage costs
- Monitor `cortex_ingester_memory_series` to track active series and detect cardinality explosions
- Use the `/distributor/ring` endpoint to verify all ingesters are healthy and balanced
- Store-gateway lazy-loads block indexes; give it enough memory for the index cache
- Set `max_query_lookback` to prevent users from running unbounded queries that overload the system
- Mimir's ruler is API-compatible with Cortex; existing recording rules and alerts migrate without changes
- Use shuffle sharding to isolate tenant query and write workloads from each other

## See Also

- prometheus
- thanos
- victoriametrics
- grafana
- alertmanager

## References

- [Grafana Mimir Documentation](https://grafana.com/docs/mimir/latest/)
- [Mimir GitHub Repository](https://github.com/grafana/mimir)
- [Mimir Architecture](https://grafana.com/docs/mimir/latest/references/architecture/)
- [Migrating from Cortex to Mimir](https://grafana.com/docs/mimir/latest/set-up/migrate/migrate-from-cortex/)
- [Mimir Helm Chart](https://github.com/grafana/mimir/tree/main/operations/helm/charts/mimir-distributed)
- [Prometheus Remote Write Spec](https://prometheus.io/docs/concepts/remote_write_spec/)
