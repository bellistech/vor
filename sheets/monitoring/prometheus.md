# Prometheus (Monitoring & Alerting)

Pull-based monitoring system with a multi-dimensional data model and powerful query language (PromQL).

## PromQL Basics

### Instant vector (current values)

```bash
# up                                       # all targets, 1=up 0=down
# node_cpu_seconds_total                   # all CPU time series
# http_requests_total{method="GET"}        # filter by label
# http_requests_total{status=~"5.."}       # regex match
# http_requests_total{status!="200"}       # not equal
# {__name__=~"http_.*"}                    # match metric name by regex
```

### Range vector (values over time)

```bash
# http_requests_total[5m]                  # last 5 minutes of samples
# http_requests_total[1h]                  # last 1 hour
```

### Offset

```bash
# http_requests_total offset 1h           # value 1 hour ago
# rate(http_requests_total[5m] offset 1d) # rate 1 day ago
```

## Rate & Counters

### rate (per-second rate of a counter)

```bash
# rate(http_requests_total[5m])                          # requests per second
# rate(http_requests_total{method="POST"}[5m])
# rate(node_network_receive_bytes_total[5m]) * 8         # bits per second
```

### irate (instant rate, last two samples)

```bash
# irate(http_requests_total[5m])           # spikier, more responsive
```

### increase (total increase over range)

```bash
# increase(http_requests_total[1h])        # total requests in last hour
```

## Histogram

### Histogram quantiles

```bash
# histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))  # p99
# histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))  # p95
# histogram_quantile(0.50, rate(http_request_duration_seconds_bucket[5m]))  # p50 (median)
```

### Average from histogram

```bash
# rate(http_request_duration_seconds_sum[5m])
# /
# rate(http_request_duration_seconds_count[5m])
```

### Histogram by label

```bash
# histogram_quantile(0.99,
#   sum(rate(http_request_duration_seconds_bucket[5m])) by (le, handler)
# )
```

## Aggregation

### Sum, avg, count, min, max

```bash
# sum(rate(http_requests_total[5m]))                     # total RPS
# sum by (method)(rate(http_requests_total[5m]))         # RPS per method
# sum without (instance)(rate(http_requests_total[5m]))  # drop instance label
# avg(node_memory_MemAvailable_bytes) by (instance)
# count(up == 1)                                         # number of up targets
# min(node_filesystem_avail_bytes) by (instance)
# max(node_cpu_seconds_total) by (instance, mode)
```

### topk and bottomk

```bash
# topk(5, rate(http_requests_total[5m]))                 # top 5 by request rate
# bottomk(3, node_filesystem_avail_bytes)                # lowest 3 free space
```

## Vector Matching

### Arithmetic between metrics

```bash
# node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes     # used memory
# (node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes)
# / node_memory_MemTotal_bytes * 100                               # memory % used
```

### Label matching

```bash
# metric_a / on(instance) metric_b                                 # match on instance
# metric_a * on(instance) group_left(job) metric_b                 # many-to-one
# metric_a / ignoring(status) metric_b                             # ignore label
```

## Alerting Rules

### Alert definition (rules.yml)

```bash
# groups:
#   - name: example
#     rules:
#       - alert: HighErrorRate
#         expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
#         for: 10m
#         labels:
#           severity: critical
#         annotations:
#           summary: "High error rate on {{ $labels.instance }}"
#           description: "Error rate is {{ $value | humanizePercentage }}"
#
#       - alert: InstanceDown
#         expr: up == 0
#         for: 5m
#         labels:
#           severity: warning
#         annotations:
#           summary: "Instance {{ $labels.instance }} is down"
```

## Recording Rules

### Pre-compute expensive queries

```bash
# groups:
#   - name: precomputed
#     interval: 30s
#     rules:
#       - record: job:http_requests_total:rate5m
#         expr: sum by (job)(rate(http_requests_total[5m]))
#
#       - record: instance:node_memory_usage:ratio
#         expr: 1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)
```

## Targets & Scrape Config

### prometheus.yml

```bash
# global:
#   scrape_interval: 15s
#   evaluation_interval: 15s
#
# rule_files:
#   - "rules/*.yml"
#
# scrape_configs:
#   - job_name: "prometheus"
#     static_configs:
#       - targets: ["localhost:9090"]
#
#   - job_name: "node"
#     static_configs:
#       - targets: ["10.0.0.1:9100", "10.0.0.2:9100"]
#
#   - job_name: "app"
#     metrics_path: /metrics
#     scheme: https
#     static_configs:
#       - targets: ["app.example.com:8080"]
#         labels:
#           env: production
```

### Service discovery

```bash
#   - job_name: "kubernetes-pods"
#     kubernetes_sd_configs:
#       - role: pod
#     relabel_configs:
#       - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
#         action: keep
#         regex: true
```

## Relabeling

### Common relabel actions

```bash
#     relabel_configs:
#       - source_labels: [__address__]           # extract port
#         regex: '(.*):(\d+)'
#         target_label: instance
#         replacement: '${1}'
#
#       - source_labels: [__meta_ec2_tag_Name]   # use EC2 tag as label
#         target_label: instance
#
#       - action: drop                           # drop targets
#         source_labels: [__meta_kubernetes_pod_phase]
#         regex: (Failed|Succeeded)
#
#       - action: labelmap                       # map meta labels
#         regex: __meta_kubernetes_pod_label_(.+)
```

## Common Queries

### HTTP metrics

```bash
# Error rate:
# sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))
#
# Request rate by handler:
# sum by (handler)(rate(http_requests_total[5m]))
#
# Latency p99 by handler:
# histogram_quantile(0.99, sum by (le, handler)(rate(http_request_duration_seconds_bucket[5m])))
```

### Node metrics

```bash
# CPU usage %:
# 100 - (avg by (instance)(irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)
#
# Memory usage %:
# (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100
#
# Disk usage %:
# (1 - node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100
```

## Tips

- Use `rate()` for counters, never raw counter values. Counters only go up (and reset).
- `rate()` needs a range of at least 2 scrape intervals. For 15s scrapes, use `[1m]` minimum.
- `irate()` is more responsive to spikes but noisier. Use `rate()` for alerting.
- Recording rules reduce query load for dashboards that run expensive queries repeatedly.
- `sum by (label)` keeps specified labels. `sum without (label)` drops specified labels.
- `histogram_quantile` operates on the `le` (less-than-or-equal) label from histogram buckets.
- Label cardinality is the primary cause of Prometheus memory issues. Avoid labels with unbounded values (user IDs, request IDs).
- `up` is a built-in metric: 1 if the target was scraped successfully, 0 if not.

## See Also

- grafana
- nginx
- haproxy
- caddy
- bind
- ssh-tunneling

## References

- [Prometheus Documentation](https://prometheus.io/docs/introduction/overview/)
- [PromQL Querying Basics](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [PromQL Functions Reference](https://prometheus.io/docs/prometheus/latest/querying/functions/)
- [PromQL Operators](https://prometheus.io/docs/prometheus/latest/querying/operators/)
- [Prometheus Configuration](https://prometheus.io/docs/prometheus/latest/configuration/configuration/)
- [Recording Rules](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/)
- [Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
- [Service Discovery](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#scrape_config)
- [Metric Types (Counter, Gauge, Histogram, Summary)](https://prometheus.io/docs/concepts/metric_types/)
- [Instrumentation Best Practices](https://prometheus.io/docs/practices/instrumentation/)
- [Prometheus GitHub Repository](https://github.com/prometheus/prometheus)
