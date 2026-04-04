# Grafana (Observability Dashboards)

Visualization and analytics platform for metrics, logs, and traces from multiple data sources.

## Dashboards

### Import a dashboard

```bash
# Via UI: Dashboards -> Import -> Enter ID from grafana.com
# Popular IDs:
#   1860   Node Exporter Full
#   3662   Prometheus 2.0 Overview
#   7587   Kubernetes Cluster
#   12006  Docker and Host Monitoring
```

### Export a dashboard

```bash
# UI: Dashboard -> Settings -> JSON Model -> Copy
# Or via API:
curl -s http://admin:admin@localhost:3000/api/dashboards/uid/abc123 | jq '.dashboard' > dashboard.json
```

### Import via API

```bash
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H 'Content-Type: application/json' \
  -d '{"dashboard": '"$(cat dashboard.json)"', "overwrite": true}'
```

## Panels

### Common panel types

```bash
# Time series   — line/area/bar charts over time
# Stat          — single big number with sparkline
# Gauge         — value on a gauge with thresholds
# Table         — tabular data
# Bar chart     — categorical comparisons
# Heatmap       — histogram buckets over time
# Logs          — log lines (from Loki, Elasticsearch)
# Alert list    — active alerts
```

### Panel query examples (Prometheus)

```bash
# Request rate:
#   rate(http_requests_total[5m])
#
# Error percentage:
#   sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m])) * 100
#
# Memory usage:
#   (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100
#
# Legend format:
#   {{instance}} - {{method}}
```

### Value mappings

```bash
# Map numeric values to text:
#   0 -> "Down" (red)
#   1 -> "Up" (green)
# Useful for Stat panels showing service status
```

### Thresholds

```bash
# Set color thresholds:
#   0-70:  green
#   70-90: yellow
#   90+:   red
# Applied via Panel -> Standard options -> Thresholds
```

## Variables (Template Variables)

### Query variable (dynamic label values)

```bash
# Name: instance
# Type: Query
# Data source: Prometheus
# Query: label_values(up, instance)
# Refresh: On dashboard load
#
# Use in panels: rate(http_requests_total{instance="$instance"}[5m])
```

### Custom variable

```bash
# Name: environment
# Type: Custom
# Values: production, staging, development
```

### Interval variable

```bash
# Name: interval
# Type: Interval
# Values: 1m, 5m, 15m, 1h
# Use: rate(http_requests_total[$interval])
```

### Chained variables

```bash
# Variable "datacenter": label_values(up, datacenter)
# Variable "instance": label_values(up{datacenter="$datacenter"}, instance)
```

### Multi-value and all

```bash
# Enable Multi-value and Include All option
# In query: rate(http_requests_total{instance=~"$instance"}[5m])
# The =~ operator handles multi-value (pipe-separated regex)
```

## Annotations

### Query-based annotations

```bash
# Data source: Prometheus
# Query: changes(deployment_timestamp[1m]) > 0
# Title: Deployment
# Tags: deploy
```

### Manual annotations

```bash
# Click on graph -> Add annotation -> Fill in description
# Visible as vertical lines on time series panels
```

### API annotations

```bash
curl -X POST http://admin:admin@localhost:3000/api/annotations \
  -H 'Content-Type: application/json' \
  -d '{"text":"Deployment v2.1","tags":["deploy"],"time":'$(date +%s000)'}'
```

## Alerting

### Alert rule (Grafana Alerting)

```bash
# In panel editor -> Alert tab:
#   Condition: WHEN avg() OF query(A, 5m, now) IS ABOVE 0.9
#   Evaluate every: 1m
#   For: 5m
#   Notifications: Send to Slack channel
```

### Contact points

```bash
# Alerting -> Contact points -> New:
#   Slack:     webhook URL
#   PagerDuty: integration key
#   Email:     SMTP settings in grafana.ini
#   Webhook:   custom HTTP endpoint
```

### Notification policies

```bash
# Route alerts by label:
#   severity=critical -> PagerDuty
#   severity=warning  -> Slack
#   Default           -> Email
```

## Data Sources

### Add via UI

```bash
# Configuration -> Data Sources -> Add
# Common:
#   Prometheus:    http://prometheus:9090
#   Loki:          http://loki:3100
#   InfluxDB:      http://influxdb:8086
#   Elasticsearch: http://elasticsearch:9200
#   PostgreSQL:    host=db port=5432
#   MySQL:         host=db port=3306
```

### Add via API

```bash
curl -X POST http://admin:admin@localhost:3000/api/datasources \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Prometheus",
    "type": "prometheus",
    "url": "http://prometheus:9090",
    "access": "proxy",
    "isDefault": true
  }'
```

## Provisioning

### Data sources (provisioning/datasources/ds.yml)

```bash
# apiVersion: 1
# datasources:
#   - name: Prometheus
#     type: prometheus
#     access: proxy
#     url: http://prometheus:9090
#     isDefault: true
#   - name: Loki
#     type: loki
#     access: proxy
#     url: http://loki:3100
```

### Dashboards (provisioning/dashboards/dashboards.yml)

```bash
# apiVersion: 1
# providers:
#   - name: default
#     folder: ''
#     type: file
#     options:
#       path: /var/lib/grafana/dashboards
#       foldersFromFilesStructure: true
```

### Alerting provisioning

```bash
# Place YAML files in /etc/grafana/provisioning/alerting/
```

## CLI & API

### grafana-cli

```bash
grafana-cli plugins install grafana-piechart-panel
grafana-cli plugins ls
grafana-cli admin reset-admin-password newpassword
```

### Common API endpoints

```bash
# List dashboards
curl -s http://admin:admin@localhost:3000/api/search | jq '.[].title'

# Get dashboard by UID
curl -s http://admin:admin@localhost:3000/api/dashboards/uid/abc123

# List data sources
curl -s http://admin:admin@localhost:3000/api/datasources | jq '.[].name'

# Health check
curl -s http://localhost:3000/api/health
```

### API key

```bash
curl -X POST http://admin:admin@localhost:3000/api/auth/keys \
  -H 'Content-Type: application/json' \
  -d '{"name":"ci","role":"Editor"}'
# Use: -H "Authorization: Bearer <key>"
```

## Tips

- Use variables for instance, job, and environment to make dashboards reusable across teams.
- `$__rate_interval` in Prometheus queries automatically picks the correct range based on scrape interval and resolution.
- Provisioning via YAML files makes Grafana config reproducible and version-controlled.
- Export dashboards as JSON and store in Git. Import via provisioning for infrastructure-as-code.
- `grafana-cli admin reset-admin-password` recovers a locked-out admin account.
- Set `GF_SECURITY_ADMIN_PASSWORD` environment variable for Docker deployments.
- Use folders to organize dashboards by team or service.
- The Explore view is better than dashboards for ad-hoc queries and debugging.

## See Also

- prometheus
- nginx
- haproxy
- caddy
- ssh-tunneling

## References

- [Grafana Documentation](https://grafana.com/docs/grafana/latest/)
- [Grafana Dashboard Guide](https://grafana.com/docs/grafana/latest/dashboards/)
- [Grafana Alerting](https://grafana.com/docs/grafana/latest/alerting/)
- [Grafana Variables](https://grafana.com/docs/grafana/latest/dashboards/variables/)
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)
- [Grafana HTTP API Reference](https://grafana.com/docs/grafana/latest/developers/http_api/)
- [Grafana Data Sources](https://grafana.com/docs/grafana/latest/datasources/)
- [Grafana Panels and Visualizations](https://grafana.com/docs/grafana/latest/panels-visualizations/)
- [Grafana Dashboard Library](https://grafana.com/grafana/dashboards/)
- [Grafana Plugin Directory](https://grafana.com/grafana/plugins/)
- [Grafana GitHub Repository](https://github.com/grafana/grafana)
