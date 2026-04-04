# SIEM (Security Information and Event Management)

Centralized platform for collecting, normalizing, correlating, and analyzing security events across an organization's infrastructure.

## Log Collection Architecture

```bash
# Syslog forwarding (rsyslog)
# /etc/rsyslog.d/50-siem-forward.conf
*.* @@siem-collector.internal:514    # TCP
*.* @siem-collector.internal:514     # UDP

# Syslog-ng forwarding with TLS
destination d_siem {
  network("siem.internal" port(6514)
    transport("tls")
    tls(ca-dir("/etc/syslog-ng/ca.d"))
  );
};
log { source(s_sys); destination(d_siem); };

# Filebeat agent configuration (filebeat.yml)
filebeat.inputs:
  - type: log
    enabled: true
    paths:
      - /var/log/auth.log
      - /var/log/syslog
      - /var/log/apache2/*.log
    fields:
      environment: production
      host_role: webserver

output.elasticsearch:
  hosts: ["https://siem-es:9200"]
  username: "filebeat"
  password: "${BEAT_PASSWORD}"
  ssl.certificate_authorities: ["/etc/pki/tls/ca.pem"]

# Windows Event Forwarding (WEF) via PowerShell
wecutil qc /q:true
wecutil cs subscription.xml

# NXLog agent for Windows
<Input in_eventlog>
    Module im_msvistalog
    Query <QueryList><Query><Select Path="Security">*</Select></Query></QueryList>
</Input>
<Output out_siem>
    Module om_tcp
    Host siem-collector.internal
    Port 1514
    OutputType Syslog_TLS
</Output>
```

## Log Normalization

```bash
# Logstash filter pipeline (normalize auth logs)
filter {
  if [type] == "syslog" {
    grok {
      match => { "message" => "%{SYSLOGTIMESTAMP:timestamp} %{SYSLOGHOST:hostname} %{DATA:program}(?:\[%{POSINT:pid}\])?: %{GREEDYDATA:message}" }
    }
    date {
      match => [ "timestamp", "MMM  d HH:mm:ss", "MMM dd HH:mm:ss" ]
      target => "@timestamp"
    }
    # Normalize user field across sources
    if [program] == "sshd" {
      grok {
        match => { "message" => "Failed password for %{USER:user} from %{IP:src_ip}" }
        add_tag => ["auth_failure"]
      }
    }
  }
  # GeoIP enrichment
  if [src_ip] {
    geoip {
      source => "src_ip"
      target => "geoip"
    }
  }
  # Threat intel enrichment
  translate {
    field => "src_ip"
    destination => "threat_intel"
    dictionary_path => "/etc/logstash/ioc_list.yml"
  }
}

# Elasticsearch index template
PUT _index_template/security-events
{
  "index_patterns": ["security-*"],
  "template": {
    "settings": { "number_of_replicas": 1 },
    "mappings": {
      "properties": {
        "src_ip":    { "type": "ip" },
        "dst_ip":    { "type": "ip" },
        "user":      { "type": "keyword" },
        "action":    { "type": "keyword" },
        "severity":  { "type": "integer" },
        "@timestamp": { "type": "date" }
      }
    }
  }
}
```

## Correlation Rules

```bash
# Splunk correlation searches (SPL)
# Brute force detection (>10 failures in 5 min from single source)
index=auth sourcetype=syslog action=failure
| stats count by src_ip, user, _time span=5m
| where count > 10
| sendalert brute_force_detected

# Lateral movement (auth from internal to multiple hosts)
index=auth action=success src_ip=10.0.0.0/8
| stats dc(dest_host) as unique_hosts values(dest_host) by src_ip user
| where unique_hosts > 5
| sendalert lateral_movement

# Data exfiltration (large outbound transfers)
index=firewall direction=outbound
| stats sum(bytes_out) as total_bytes by src_ip
| where total_bytes > 1073741824
| sendalert possible_exfiltration

# Elasticsearch Watcher (failed login correlation)
PUT _watcher/watch/brute_force
{
  "trigger": { "schedule": { "interval": "5m" } },
  "input": {
    "search": {
      "request": {
        "indices": ["security-*"],
        "body": {
          "query": {
            "bool": {
              "must": [
                { "match": { "action": "failure" } },
                { "range": { "@timestamp": { "gte": "now-5m" } } }
              ]
            }
          },
          "aggs": {
            "by_source": {
              "terms": { "field": "src_ip", "min_doc_count": 10 }
            }
          }
        }
      }
    }
  },
  "actions": {
    "notify": {
      "webhook": { "host": "soc-alerts.internal", "port": 8443, "path": "/alert" }
    }
  }
}

# Wazuh rules (custom correlation)
<group name="custom_brute_force">
  <rule id="100001" level="10" frequency="8" timeframe="120">
    <if_matched_sid>5710</if_matched_sid>
    <same_source_ip />
    <description>Brute force attack detected (8+ failures in 2 min)</description>
    <mitre><id>T1110</id></mitre>
  </rule>
</group>
```

## Use Case Library

```bash
# Use Case 1: Impossible Travel
# User authenticates from geographically distant locations within short time
index=auth action=success
| iplocation src_ip
| sort user _time
| autoregress City as prev_City p=1
| autoregress _time as prev_time p=1
| eval time_diff=_time-prev_time
| eval distance=haversine(lat, lon, prev_lat, prev_lon)
| eval speed=distance/(time_diff/3600)
| where speed > 800 AND time_diff < 7200

# Use Case 2: Service Account Anomaly
# Service accounts authenticating from unexpected sources
index=auth user=svc_* action=success
| lookup approved_sources.csv user OUTPUT approved_ips
| where NOT cidrmatch(approved_ips, src_ip)

# Use Case 3: Privilege Escalation Chain
# Normal user -> admin in short window
index=auth (action=success AND (role=admin OR group=administrators))
| join user [search index=auth action=success role=user earliest=-1h]
| where user_role_before != "admin"

# Use Case 4: Beaconing Detection
# Regular interval callbacks to external IPs
index=firewall direction=outbound
| bucket _time span=1m
| stats count by src_ip, dest_ip, _time
| timechart span=1m count by dest_ip
| foreach * [eval jitter_<<FIELD>>=abs(<<FIELD>>-avg(<<FIELD>>))]
```

## SOAR Integration

```bash
# Trigger automated response via SOAR webhook
# TheHive alert creation
curl -XPOST https://thehive.internal:9000/api/alert \
  -H "Authorization: Bearer $THEHIVE_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Brute Force from 203.0.113.50",
    "type": "external",
    "source": "SIEM",
    "severity": 2,
    "tlp": 2,
    "tags": ["brute-force", "T1110"],
    "description": "10+ failed SSH attempts in 5 minutes"
  }'

# XSOAR (Demisto) playbook trigger
curl -XPOST https://xsoar.internal/incident \
  -H "Authorization: $XSOAR_KEY" \
  -d '{"name":"Brute Force","type":"Authentication","severity":2}'

# Shuffle SOAR workflow trigger
curl -XPOST https://shuffle.internal/api/v1/hooks/$HOOK_ID \
  -H "Content-Type: application/json" \
  -d '{"alert_type":"brute_force","source_ip":"203.0.113.50"}'
```

## Retention and Compliance

```bash
# Elasticsearch ILM policy (hot/warm/cold/delete)
PUT _ilm/policy/security-retention
{
  "policy": {
    "phases": {
      "hot":    { "actions": { "rollover": { "max_size": "50gb", "max_age": "1d" } } },
      "warm":   { "min_age": "7d",  "actions": { "shrink": { "number_of_shards": 1 } } },
      "cold":   { "min_age": "30d", "actions": { "freeze": {} } },
      "delete": { "min_age": "365d", "actions": { "delete": {} } }
    }
  }
}

# Compliance log retention requirements
# PCI-DSS: 1 year online, 1 year archive
# HIPAA: 6 years
# SOX: 7 years
# GDPR: Minimize retention, document justification
# SOC 2: Defined in policy, typically 1 year

# Splunk data retention (indexes.conf)
[security_events]
frozenTimePeriodInSecs = 31536000  # 365 days
maxTotalDataSizeMB = 500000
coldToFrozenDir = /archive/splunk/security_events
```

## Platform Comparison

```bash
# ELK Stack (Elasticsearch + Logstash + Kibana)
# - Open source core, paid (Elastic Security) for SIEM features
# - Scales horizontally, flexible schema
# - Requires significant tuning and maintenance
# - Best for: custom deployments, large scale, budget-conscious

# Splunk Enterprise
# - Commercial, license by daily ingest volume
# - Powerful SPL query language
# - Extensive app ecosystem
# - Best for: enterprise SOCs, compliance-heavy environments

# Wazuh (OSSEC fork + ELK)
# - Fully open source SIEM + XDR
# - Built-in FIM, vulnerability detection, compliance
# - Agent-based with manager/cluster architecture
# - Best for: open-source-first, compliance, endpoint+network

# Grafana Loki + Promtail
# - Log aggregation (not full SIEM)
# - Label-based indexing (cheaper storage)
# - Pairs with Grafana dashboards
# - Best for: cloud-native, Kubernetes environments

# QRadar (IBM)
# - Enterprise SIEM with built-in correlation
# - License by EPS (events per second)
# - Strong compliance reporting
# - Best for: regulated industries, IBM shops
```

## Tips

- Start with high-value log sources first: authentication, firewall, DNS, endpoint
- Normalize timestamps to UTC across all sources before correlation
- Tune correlation rules iteratively; start with high thresholds and lower as you understand baselines
- Use lookup tables for approved IPs, service accounts, and known-good hashes to reduce false positives
- Implement log source health monitoring; a silent source is worse than a noisy one
- Tag events with MITRE ATT&CK technique IDs for structured threat coverage analysis
- Design retention policies per compliance requirement before ingesting data
- Test correlation rules against historical data (replay attacks) before going live
- Monitor SIEM ingest rates and license consumption daily to avoid surprises
- Document every use case with expected alert volume, response playbook, and owner
- Keep a "tuning log" of every rule modification for audit trail

## See Also

- Suricata for network-level event generation
- osquery for endpoint telemetry feeding SIEM
- MITRE ATT&CK for detection coverage mapping
- WAF for web application event sources
- CIS Benchmarks for compliance-driven log requirements

## References

- [Elastic SIEM Documentation](https://www.elastic.co/guide/en/security/current/index.html)
- [Splunk Security Essentials](https://splunkbase.splunk.com/app/3435/)
- [Wazuh Documentation](https://documentation.wazuh.com/current/)
- [NIST SP 800-92 Guide to Security Log Management](https://csrc.nist.gov/publications/detail/sp/800-92/final)
- [MITRE ATT&CK Data Sources](https://attack.mitre.org/datasources/)
- [Sigma Rules Repository](https://github.com/SigmaHQ/sigma)
- [TheHive Project](https://thehive-project.org/)
