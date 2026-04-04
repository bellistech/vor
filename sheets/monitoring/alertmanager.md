# Alertmanager (Prometheus Alert Routing)

Handles alerts from Prometheus and other sources, providing deduplication, grouping, inhibition, silencing, and routing to notification receivers like email, Slack, PagerDuty, and webhooks.

## Configuration Structure

### alertmanager.yml

```yaml
global:
  resolve_timeout: 5m                    # mark alert resolved if not re-fired
  smtp_smarthost: 'smtp.example.com:587'
  smtp_from: 'alertmanager@example.com'
  smtp_auth_username: 'alerts@example.com'
  smtp_auth_password: 'secret'
  slack_api_url: 'https://hooks.slack.com/services/T00/B00/xxx'

route:
  receiver: 'default-slack'              # fallback receiver
  group_by: ['alertname', 'cluster']     # group alerts by these labels
  group_wait: 30s                        # wait before sending first notification
  group_interval: 5m                     # wait before sending updates to group
  repeat_interval: 4h                    # re-notify after this interval
  routes:
    - match:
        severity: critical
      receiver: 'pagerduty-critical'
      group_wait: 10s
      repeat_interval: 1h
    - match_re:
        service: ^(payment|billing)$
      receiver: 'finance-team'
    - match:
        severity: warning
      receiver: 'slack-warnings'
      repeat_interval: 12h
    - matchers:
        - alertname =~ "CPU.*|Memory.*"
        - team = infrastructure
      receiver: 'infra-slack'

receivers:
  - name: 'default-slack'
    slack_configs:
      - channel: '#alerts'
        title: '{{ .GroupLabels.alertname }}'
        text: >-
          {{ range .Alerts }}
          *{{ .Labels.instance }}*: {{ .Annotations.description }}
          {{ end }}

  - name: 'pagerduty-critical'
    pagerduty_configs:
      - service_key: '<pagerduty-integration-key>'
        severity: critical
        description: '{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary }}'

  - name: 'finance-team'
    email_configs:
      - to: 'finance-oncall@example.com'
        send_resolved: true

  - name: 'slack-warnings'
    slack_configs:
      - channel: '#warnings'
        send_resolved: true

  - name: 'infra-slack'
    slack_configs:
      - channel: '#infra-alerts'

inhibit_rules:
  - source_matchers:
      - severity = critical
    target_matchers:
      - severity = warning
    equal: ['alertname', 'cluster']      # suppress warning if critical exists
  - source_matchers:
      - alertname = NodeDown
    target_matchers:
      - alertname =~ ".+"
    equal: ['instance']                  # suppress all alerts for a down node
```

## Prometheus Alert Rules

```yaml
# prometheus/rules/alerts.yml
groups:
  - name: node-alerts
    rules:
      - alert: HighCPU
        expr: 100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80
        for: 10m
        labels:
          severity: warning
          team: infrastructure
        annotations:
          summary: "High CPU on {{ $labels.instance }}"
          description: "CPU usage above 80% for 10 minutes (current: {{ $value | printf \"%.1f\" }}%)"

      - alert: HighMemory
        expr: (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 90
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Memory critical on {{ $labels.instance }}"

      - alert: DiskSpaceLow
        expr: (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) * 100 < 10
        for: 15m
        labels:
          severity: warning
        annotations:
          summary: "Disk space below 10% on {{ $labels.instance }}"

      - alert: InstanceDown
        expr: up == 0
        for: 3m
        labels:
          severity: critical
        annotations:
          summary: "Instance {{ $labels.instance }} is down"
```

## amtool (CLI)

```bash
# Check configuration syntax
amtool check-config alertmanager.yml

# Show current alerts
amtool alert --alertmanager.url=http://localhost:9093

# Show active alerts with filters
amtool alert query alertname=HighCPU severity=critical

# Create a silence
amtool silence add \
  --alertmanager.url=http://localhost:9093 \
  --author="oncall" \
  --comment="Maintenance window" \
  --duration=2h \
  alertname=HighCPU instance=web-01

# Silence with matchers
amtool silence add \
  alertname=~"Disk.*" \
  cluster=staging \
  --duration=4h \
  --comment="Staging disk expansion"

# List active silences
amtool silence query

# Expire (remove) a silence
amtool silence expire <silence-id>

# Expire all silences
amtool silence expire $(amtool silence query -q)

# Show routing tree
amtool config routes show

# Test routing — which receiver matches?
amtool config routes test-routing \
  --tree \
  alertname=HighCPU severity=critical team=infrastructure
```

## HTTP API

```bash
# List alerts
curl http://localhost:9093/api/v2/alerts

# Post an alert
curl -X POST http://localhost:9093/api/v2/alerts \
  -H 'Content-Type: application/json' \
  -d '[{
    "labels": {"alertname": "TestAlert", "severity": "warning"},
    "annotations": {"summary": "Test alert fired"},
    "startsAt": "2024-01-01T00:00:00Z",
    "generatorURL": "http://prometheus:9090"
  }]'

# List silences
curl http://localhost:9093/api/v2/silences

# Create a silence via API
curl -X POST http://localhost:9093/api/v2/silences \
  -H 'Content-Type: application/json' \
  -d '{
    "matchers": [{"name": "alertname", "value": "HighCPU", "isRegex": false}],
    "startsAt": "2024-01-01T00:00:00Z",
    "endsAt": "2024-01-01T04:00:00Z",
    "createdBy": "api",
    "comment": "Scheduled maintenance"
  }'

# Get status
curl http://localhost:9093/api/v2/status

# Get receiver list
curl http://localhost:9093/api/v2/receivers
```

## High Availability

```bash
# Run multiple Alertmanager instances in a cluster
# They gossip to deduplicate notifications
alertmanager \
  --config.file=alertmanager.yml \
  --cluster.listen-address=0.0.0.0:9094 \
  --cluster.peer=alertmanager-1:9094 \
  --cluster.peer=alertmanager-2:9094

# Prometheus points to all instances
# prometheus.yml:
# alerting:
#   alertmanagers:
#     - static_configs:
#         - targets:
#             - alertmanager-0:9093
#             - alertmanager-1:9093
#             - alertmanager-2:9093
```

## Template Functions

```bash
# Available in receiver templates:
# {{ .GroupLabels }}         — labels used for grouping
# {{ .CommonLabels }}        — labels shared by all alerts in group
# {{ .CommonAnnotations }}   — annotations shared by all alerts
# {{ .ExternalURL }}         — Alertmanager URL
# {{ .Alerts }}              — list of alerts in the group
# {{ .Alerts.Firing }}       — only firing alerts
# {{ .Alerts.Resolved }}     — only resolved alerts
# {{ range .Alerts }}
#   {{ .Labels.instance }}
#   {{ .Annotations.summary }}
#   {{ .StartsAt }}
#   {{ .EndsAt }}
# {{ end }}
```

## Tips

- Use `group_by` with `alertname` and a service/cluster label to avoid notification floods from correlated alerts
- Set `group_wait` to 30s-60s to batch related alerts before the first notification fires
- Use inhibition rules to suppress warnings when critical alerts exist for the same target
- Always set `send_resolved: true` on receivers so the team knows when issues auto-resolve
- Use `amtool config routes test-routing` to verify which receiver handles a given label set before deploying
- Run at least 2 Alertmanager instances in cluster mode for HA; they deduplicate via gossip
- Use `repeat_interval` wisely: too short creates alert fatigue, too long risks missed follow-ups (4h-12h is typical)
- Create silence templates for recurring maintenance windows to speed up oncall workflows
- Use regex matchers (`match_re`) to route related services to the same team receiver
- Set `resolve_timeout` to 5m so stale alerts resolve when Prometheus stops sending them
- Use the `/api/v2/alerts` endpoint to integrate external systems that generate alerts

## See Also

prometheus, grafana, victoriametrics, opentelemetry

## References

- [Alertmanager Documentation](https://prometheus.io/docs/alerting/latest/alertmanager/)
- [Alertmanager Configuration](https://prometheus.io/docs/alerting/latest/configuration/)
- [Alertmanager Routing Tree](https://prometheus.io/docs/alerting/latest/notification_examples/)
- [amtool CLI](https://github.com/prometheus/alertmanager#amtool)
- [Prometheus Alerting Rules](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/)
