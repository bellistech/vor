# VictoriaMetrics (Time Series Database)

High-performance, cost-effective time series database compatible with Prometheus, offering better compression, lower resource usage, long-term storage, and horizontal scaling via cluster mode.

## Single-Node Setup

```bash
# Docker
docker run -d --name victoriametrics \
  -p 8428:8428 \
  -v vmdata:/victoria-metrics-data \
  victoriametrics/victoria-metrics:latest \
  -retentionPeriod=90d \
  -storageDataPath=/victoria-metrics-data

# Binary
victoria-metrics \
  -storageDataPath=/var/lib/victoriametrics \
  -retentionPeriod=1y \
  -httpListenAddr=:8428 \
  -memory.allowedPercent=60
```

## Cluster Mode

```bash
# vmstorage — stores time series data
vmstorage \
  -storageDataPath=/data/vmstorage \
  -retentionPeriod=1y \
  -vminsertAddr=:8400 \
  -vmselectAddr=:8401

# vminsert — accepts writes, distributes to vmstorage
vminsert \
  -storageNode=vmstorage-1:8400 \
  -storageNode=vmstorage-2:8400 \
  -storageNode=vmstorage-3:8400 \
  -httpListenAddr=:8480 \
  -replicationFactor=2

# vmselect — queries across vmstorage nodes
vmselect \
  -storageNode=vmstorage-1:8401 \
  -storageNode=vmstorage-2:8401 \
  -storageNode=vmstorage-3:8401 \
  -httpListenAddr=:8481 \
  -dedup.minScrapeInterval=15s
```

## Data Ingestion

### Prometheus remote_write

```yaml
# prometheus.yml
remote_write:
  - url: http://victoriametrics:8428/api/v1/write
    queue_config:
      max_samples_per_send: 10000
      capacity: 50000
      max_shards: 30
```

### Import via API

```bash
# Prometheus text format
curl -d 'metric_name{label="value"} 123' \
  http://localhost:8428/api/v1/import/prometheus

# JSON line format
curl -d '{"metric":{"__name__":"cpu","host":"srv1"},"values":[0.5],"timestamps":[1700000000]}' \
  http://localhost:8428/api/v1/import

# CSV import
curl -d 'metric_name,label,value,timestamp
cpu_usage,host=srv1,0.75,1700000000' \
  http://localhost:8428/api/v1/import/csv

# InfluxDB line protocol
curl -d 'cpu,host=srv1 usage=0.75 1700000000000000000' \
  http://localhost:8428/write
```

## Querying (PromQL + MetricsQL)

```bash
# Standard PromQL works
curl "http://localhost:8428/api/v1/query?query=up"

# Range query
curl "http://localhost:8428/api/v1/query_range?\
query=rate(http_requests_total[5m])&\
start=2024-01-01T00:00:00Z&\
end=2024-01-01T01:00:00Z&\
step=60s"

# MetricsQL extensions (not in standard PromQL)
# range_first(m[1h])            — first value in range
# range_last(m[1h])             — last value in range
# running_sum(m[1h])            — running sum
# rollup_rate(m[1d])            — rate over rollup window
# label_set(m, "env", "prod")   — add/override label
# label_del(m, "instance")      — remove label
# limit_offset(5, 10, m)        — pagination (limit 5, offset 10)
# count_values_over_time("val", m[1h])  — distinct value histogram

# Export all data for a metric
curl "http://localhost:8428/api/v1/export?match[]=cpu_usage"

# Export in native format (faster)
curl "http://localhost:8428/api/v1/export/native?match[]=cpu_usage" > dump.bin
```

## vmagent (Scraping & Forwarding)

```bash
# vmagent replaces Prometheus for scraping
vmagent \
  -promscrape.config=prometheus.yml \
  -remoteWrite.url=http://vminsert:8480/insert/0/prometheus/api/v1/write \
  -remoteWrite.tmpDataPath=/tmp/vmagent-buffer \
  -remoteWrite.maxDiskUsagePerURL=1GB

# vmagent features:
# - Scrapes Prometheus targets
# - Service discovery (K8s, Consul, EC2, etc.)
# - On-disk buffering when remote is down
# - Relabeling (same as Prometheus)
# - Streaming aggregation
# - Multiple remote_write destinations
```

## Retention & Storage

```bash
# Set retention
-retentionPeriod=90d                  # 90 days
-retentionPeriod=1y                   # 1 year
-retentionPeriod=5y                   # 5 years

# Downsampling via recording rules (vmalert)
# Keep 15s data for 30d, 1m data for 1y, 5m data for 5y

# Force merge (compact old data)
curl -X POST "http://localhost:8428/internal/force_merge"

# Delete series
curl -d 'match[]=old_metric{env="staging"}' \
  http://localhost:8428/api/v1/admin/tsdb/delete_series

# Snapshot for backup
curl -X POST "http://localhost:8428/snapshot/create"
# Returns: {"status":"ok","snapshot":"20240101T120000Z-abc123"}

# List snapshots
curl "http://localhost:8428/snapshot/list"

# Delete snapshot
curl -X POST "http://localhost:8428/snapshot/delete?snapshot=20240101T120000Z-abc123"
```

## vmalert (Alerting & Recording Rules)

```bash
# vmalert evaluates Prometheus alerting and recording rules
vmalert \
  -rule=/etc/vmalert/rules/*.yml \
  -datasource.url=http://vmselect:8481/select/0/prometheus \
  -remoteWrite.url=http://vminsert:8480/insert/0/prometheus \
  -notifier.url=http://alertmanager:9093 \
  -evaluationInterval=15s
```

## Deduplication

```bash
# Deduplicate samples from HA Prometheus pairs
-dedup.minScrapeInterval=15s

# How it works:
# If multiple samples exist for the same time series
# within the dedup interval, only the sample with
# the highest value is kept (deterministic choice).
# This lets you run 2 Prometheus instances scraping
# the same targets without double-counting.
```

## Monitoring VictoriaMetrics

```bash
# Built-in metrics at /metrics
curl http://localhost:8428/metrics

# Key metrics to watch:
# vm_rows_inserted_total           — ingestion rate
# vm_slow_queries_total            — queries hitting slow path
# vm_active_merges                 — background merge activity
# vm_merge_speed                   — merge throughput
# vm_free_disk_space_bytes         — remaining disk
# vm_data_size_bytes               — total data on disk
# vm_cache_entries                 — cache utilization
# vm_http_request_duration_seconds — API latencies
```

## Tips

- Start with single-node VictoriaMetrics; it handles millions of samples/sec and often eliminates the need for clustering
- Use `vmagent` instead of Prometheus for scraping when you only need VictoriaMetrics as storage
- Set `-dedup.minScrapeInterval` equal to your Prometheus scrape interval to deduplicate HA pairs
- Use MetricsQL extensions like `keep_last_value` and `default` to handle missing data in dashboards
- Enable on-disk buffering in vmagent (`-remoteWrite.tmpDataPath`) to survive remote storage outages
- Create snapshots before upgrades; they are instantaneous and use hard links (no extra disk)
- Set `-retentionPeriod` generously; VictoriaMetrics compresses to ~0.7 bytes/sample vs Prometheus ~1.5
- Use `-memory.allowedPercent=60` to leave headroom for OS page cache, which speeds up queries
- Run `force_merge` during off-peak hours to compact data and reclaim disk after large deletes
- Prefer native export format over JSON for backups and migrations; it is significantly faster
- Use vmalert for recording rules to pre-compute expensive queries and reduce dashboard load

## See Also

prometheus, alertmanager, grafana, opentelemetry

## References

- [VictoriaMetrics Documentation](https://docs.victoriametrics.com/)
- [VictoriaMetrics Cluster](https://docs.victoriametrics.com/cluster-victoriametrics/)
- [MetricsQL Reference](https://docs.victoriametrics.com/metricsql/)
- [vmagent Documentation](https://docs.victoriametrics.com/vmagent/)
- [vmalert Documentation](https://docs.victoriametrics.com/vmalert/)
