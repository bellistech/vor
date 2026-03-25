# IDS/IPS (Intrusion Detection and Prevention Systems)

Configuration, rule writing, tuning, and management for Suricata, Snort,
OSSEC/Wazuh, and fail2ban in defensive environments.

---

## 1. Suricata

### Installation and Setup

```bash
# Debian/Ubuntu
apt install suricata suricata-update

# RHEL/CentOS
yum install epel-release
yum install suricata

# Verify installation
suricata --build-info | head -20

# Configure network interface
# /etc/suricata/suricata.yaml
# af-packet:
#   - interface: eth0
#     cluster-id: 99
#     cluster-type: cluster_flow
#     defrag: yes

# Set HOME_NET
# vars:
#   address-groups:
#     HOME_NET: "[10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16]"
#     EXTERNAL_NET: "!$HOME_NET"

# Enable IPS mode (inline)
# nfqueue mode in suricata.yaml or use af-packet with copy-mode: ips
```

### Rule Management

```bash
# Update rules with suricata-update
suricata-update

# List available rule sources
suricata-update list-sources

# Enable additional sources
suricata-update enable-source et/open
suricata-update enable-source oisf/trafficid

# Update and reload
suricata-update
suricatasc -c reload-rules

# Custom rules directory
# /etc/suricata/rules/local.rules
```

### Rule Syntax

```bash
# Basic rule format:
# action proto src_ip src_port -> dst_ip dst_port (options;)

# Detect SSH brute force
alert ssh $EXTERNAL_NET any -> $HOME_NET 22 (msg:"Possible SSH brute force"; \
  flow:to_server,established; \
  threshold:type both,track by_src,count 5,seconds 60; \
  classtype:attempted-admin; sid:1000001; rev:1;)

# Detect outbound connection to known C2 port
alert tcp $HOME_NET any -> $EXTERNAL_NET 4444 (msg:"Outbound C2 - port 4444"; \
  flow:to_server,established; \
  classtype:trojan-activity; sid:1000002; rev:1;)

# Detect SQL injection in HTTP
alert http $EXTERNAL_NET any -> $HOME_NET any (msg:"SQL Injection attempt"; \
  flow:to_server,established; \
  content:"UNION"; nocase; content:"SELECT"; nocase; distance:0; \
  http_uri; \
  classtype:web-application-attack; sid:1000003; rev:1;)

# Detect DNS exfiltration (long subdomain)
alert dns any any -> any any (msg:"Possible DNS exfil - long query"; \
  dns.query; content:"|00|"; byte_test:1,>,50,0,relative; \
  classtype:bad-unknown; sid:1000004; rev:1;)

# Detect executable download
alert http $EXTERNAL_NET any -> $HOME_NET any (msg:"EXE download over HTTP"; \
  flow:to_client,established; \
  filemagic:"PE32"; \
  classtype:policy-violation; sid:1000005; rev:1;)
```

### Logging (eve.json and fast.log)

```bash
# fast.log — one-line-per-alert (quick triage)
# /var/log/suricata/fast.log
tail -f /var/log/suricata/fast.log

# eve.json — full JSON event log (detailed analysis)
# Parse with jq
# All alerts
cat /var/log/suricata/eve.json | jq 'select(.event_type=="alert")'

# Top 10 alert signatures
cat /var/log/suricata/eve.json | \
  jq -r 'select(.event_type=="alert") | .alert.signature' | \
  sort | uniq -c | sort -rn | head -10

# Alerts from specific source IP
cat /var/log/suricata/eve.json | \
  jq 'select(.event_type=="alert" and .src_ip=="10.0.0.50")'

# DNS queries
cat /var/log/suricata/eve.json | \
  jq 'select(.event_type=="dns") | {ts: .timestamp, query: .dns.rrname}'

# HTTP transactions
cat /var/log/suricata/eve.json | \
  jq 'select(.event_type=="http") | {ts: .timestamp, host: .http.hostname, url: .http.url, status: .http.status}'

# Flow records
cat /var/log/suricata/eve.json | \
  jq 'select(.event_type=="flow") | {src: .src_ip, dst: .dest_ip, proto: .proto, bytes: .flow.bytes_toserver}'
```

### Thresholds and Suppress

```bash
# /etc/suricata/threshold.config

# Suppress alerts from trusted scanner
suppress gen_id 1, sig_id 2100498, track by_src, ip 10.0.0.100

# Rate-limit noisy rule (max 10 alerts per 60 seconds per source)
threshold gen_id 1, sig_id 2100498, type limit, \
  track by_src, count 10, seconds 60

# Threshold — only alert after N occurrences
threshold gen_id 1, sig_id 1000001, type threshold, \
  track by_src, count 5, seconds 300

# Both — alert once per window after threshold met
threshold gen_id 1, sig_id 1000001, type both, \
  track by_src, count 10, seconds 60
```

### Performance Tuning

```bash
# /etc/suricata/suricata.yaml key performance settings

# Thread configuration (match CPU cores)
# threading:
#   set-cpu-affinity: yes
#   detect-thread-ratio: 1.0

# Memory limits
# stream:
#   memcap: 256mb
# flow:
#   memcap: 256mb

# Packet capture tuning
# af-packet:
#   - interface: eth0
#     threads: 4
#     ring-size: 200000
#     buffer-size: 65536

# Check performance stats
suricatasc -c dump-counters | python3 -m json.tool

# Monitor dropped packets
cat /var/log/suricata/stats.log | grep "capture.kernel_drops"
```

---

## 2. Snort

### Rule Syntax

```bash
# Snort rule format (similar to Suricata):
# action proto src_ip src_port -> dst_ip dst_port (options;)

# Detect ICMP ping sweep
alert icmp $EXTERNAL_NET any -> $HOME_NET any (msg:"ICMP Ping Sweep"; \
  itype:8; detection_filter:track by_src,count 10,seconds 5; \
  classtype:attempted-recon; sid:1000010; rev:1;)

# Detect web shell access
alert tcp $EXTERNAL_NET any -> $HOME_NET $HTTP_PORTS (msg:"Web shell detected"; \
  flow:to_server,established; \
  content:"cmd="; http_uri; \
  content:"whoami"; http_uri; \
  classtype:web-application-attack; sid:1000011; rev:1;)

# Preprocessor configuration (snort.conf)
# preprocessor sfportscan: proto { all } \
#   memcap { 10000000 } \
#   sense_level { medium }
```

### Management

```bash
# Test configuration
snort -T -c /etc/snort/snort.conf

# Run in IDS mode
snort -A full -q -c /etc/snort/snort.conf -i eth0

# Run in IPS/inline mode
snort -Q --daq afpacket -i eth0:eth1 -c /etc/snort/snort.conf

# Update rules with PulledPork
pulledpork.pl -c /etc/snort/pulledpork.conf -l

# Validate rules
snort -T -c /etc/snort/snort.conf --warn-all
```

---

## 3. OSSEC / Wazuh

### Agent Deployment

```bash
# --- Wazuh Manager ---
# Install manager (Debian/Ubuntu)
curl -s https://packages.wazuh.com/key/GPG-KEY-WAZUH | \
  gpg --dearmor -o /usr/share/keyrings/wazuh.gpg
echo "deb [signed-by=/usr/share/keyrings/wazuh.gpg] \
  https://packages.wazuh.com/4.x/apt/ stable main" > \
  /etc/apt/sources.list.d/wazuh.list
apt update && apt install wazuh-manager

systemctl enable --now wazuh-manager

# --- Wazuh Agent ---
apt install wazuh-agent

# Configure agent to connect to manager
# /var/ossec/etc/ossec.conf:
# <client>
#   <server>
#     <address>MANAGER_IP</address>
#   </server>
# </client>

systemctl enable --now wazuh-agent

# Register agent on manager
/var/ossec/bin/manage_agents   # Interactive
# Or automated:
/var/ossec/bin/agent-auth -m MANAGER_IP

# List connected agents (on manager)
/var/ossec/bin/agent_control -l
```

### Custom Rules

```xml
<!-- /var/ossec/etc/rules/local_rules.xml -->

<!-- Detect multiple failed SSH logins -->
<group name="custom,sshd,">
  <rule id="100001" level="10" frequency="5" timeframe="120">
    <if_matched_sid>5710</if_matched_sid>
    <description>Multiple SSH failed logins (brute force)</description>
    <group>authentication_failures,</group>
  </rule>
</group>

<!-- Detect new user creation -->
<group name="custom,account,">
  <rule id="100002" level="12">
    <if_sid>5901</if_sid>
    <match>useradd</match>
    <description>New user account created</description>
    <group>account_changed,</group>
  </rule>
</group>

<!-- Detect file integrity change in critical directory -->
<group name="custom,syscheck,">
  <rule id="100003" level="14">
    <if_sid>550</if_sid>
    <match>/etc/shadow</match>
    <description>Shadow file modified</description>
    <group>file_integrity,</group>
  </rule>
</group>
```

### Active Response

```xml
<!-- /var/ossec/etc/ossec.conf -->

<!-- Block attacking IP with iptables -->
<active-response>
  <command>firewall-drop</command>
  <location>local</location>
  <rules_id>100001</rules_id>
  <timeout>3600</timeout>
</active-response>

<!-- Custom active response script -->
<command>
  <name>custom-block</name>
  <executable>custom-block.sh</executable>
  <timeout_allowed>yes</timeout_allowed>
</command>

<active-response>
  <command>custom-block</command>
  <location>local</location>
  <level>12</level>
  <timeout>600</timeout>
</active-response>
```

```bash
# Check active responses
/var/ossec/bin/agent_control -L

# List blocked IPs
/var/ossec/active-response/bin/firewall-drop.sh list

# Manually unblock
/var/ossec/active-response/bin/firewall-drop.sh delete - 10.0.0.50
```

---

## 4. fail2ban

### Configuration

```bash
# Install
apt install fail2ban   # Debian/Ubuntu
yum install fail2ban   # RHEL/CentOS

# Main config: /etc/fail2ban/jail.local (override jail.conf)
cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5
banaction = iptables-multiport
action = %(action_mwl)s
ignoreip = 127.0.0.1/8 10.0.0.0/24

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
bantime = 86400

[sshd-ddos]
enabled = true
port = ssh
filter = sshd-ddos
logpath = /var/log/auth.log
maxretry = 6
bantime = 3600
EOF
```

### Advanced Patterns

```bash
# Custom filter: /etc/fail2ban/filter.d/custom-webapp.conf
[Definition]
failregex = ^<HOST> -.*"(GET|POST).*HTTP.*" 403
            ^<HOST> -.*"(GET|POST).*(\.php|\.asp|\.aspx|\.jsp).*" 404
            ^<HOST> -.*"(GET|POST).*(wp-login|administrator|admin).*" (401|403)
ignoreregex =

# Custom jail for the filter
# In /etc/fail2ban/jail.local:
# [custom-webapp]
# enabled = true
# port = http,https
# filter = custom-webapp
# logpath = /var/log/nginx/access.log
# maxretry = 10
# findtime = 300
# bantime = 7200

# Recidive jail (ban repeat offenders longer)
# [recidive]
# enabled = true
# filter = recidive
# logpath = /var/log/fail2ban.log
# bantime = 604800
# findtime = 86400
# maxretry = 3
```

### Management

```bash
# Status
fail2ban-client status
fail2ban-client status sshd

# Currently banned IPs
fail2ban-client get sshd banned

# Manually ban/unban
fail2ban-client set sshd banip 10.0.0.50
fail2ban-client set sshd unbanip 10.0.0.50

# Test filter against log file
fail2ban-regex /var/log/auth.log /etc/fail2ban/filter.d/sshd.conf

# Reload after config changes
fail2ban-client reload

# Check fail2ban log
tail -f /var/log/fail2ban.log
```

---

## 5. Custom Rule Writing Guidelines

```bash
# Rule writing best practices:

# 1. Start with a clear detection goal (map to MITRE ATT&CK)
# 2. Write the rule in test/alert mode first, never block immediately
# 3. Test against PCAP or log samples before deploying

# Test Suricata rule against PCAP
suricata -r /evidence/capture.pcap -c /etc/suricata/suricata.yaml \
  -S /etc/suricata/rules/local.rules -l /tmp/test_output/

# Test Snort rule against PCAP
snort -r /evidence/capture.pcap -c /etc/snort/snort.conf \
  -A console -q

# 4. Include descriptive msg, classtype, reference, and metadata
# 5. Use specific content matches (avoid overly broad patterns)
# 6. Set appropriate thresholds to reduce noise
# 7. Document the rule with comments explaining detection logic
# 8. Review and tune weekly based on false positive/negative rates

# Rule performance: content matches are faster than PCRE
# Put the most unique/rare content match first (fast_pattern)
alert http any any -> any any (msg:"Webshell upload"; \
  flow:to_server,established; \
  content:"POST"; http_method; \
  content:".php"; http_uri; \
  file_data; content:"<?php"; fast_pattern; \
  content:"eval("; distance:0; \
  classtype:web-application-attack; sid:1000020; rev:1;)
```

---

## 6. False Positive Management

```bash
# 1. Triage: is it a true positive, false positive, or benign true positive?

# 2. For false positives, choose a strategy:
#    a. Suppress — hide alerts from specific source/dest
#    b. Threshold — reduce alert volume, don't eliminate
#    c. Tune rule — add negation content or refine match
#    d. Disable rule — last resort for consistently noisy rules

# Suricata: disable a rule
# /etc/suricata/disable.conf
# 2100498

# Suricata: modify a rule
# /etc/suricata/modify.conf
# 2100498 "from_string" "to_string"

# Wazuh: tune rule level to 0 (effectively disable)
# <rule id="100099" level="0">
#   <if_sid>5710</if_sid>
#   <srcip>10.0.0.100</srcip>
#   <description>Suppress scanner alerts</description>
# </rule>

# 3. Document every suppression with:
#    - Date, analyst, rule SID
#    - Reason for suppression
#    - Review date (re-evaluate quarterly)
```

---

## Tips

- Deploy IDS sensors at network boundaries, between VLANs, and on critical
  hosts (HIDS).
- Use IDS in detection mode initially; only switch to IPS/blocking after
  thorough tuning.
- Keep rule sets updated daily; new threats emerge constantly.
- Monitor sensor health: check for dropped packets, CPU usage, and disk space.
- Correlate IDS alerts with other log sources (SIEM) for higher-confidence
  detections.
- Maintain separate rule files for custom rules; never edit vendor rule files
  directly.
- Test rule changes in a lab environment with representative traffic before
  production deployment.
- Set up automated alerting for high-severity events (email, Slack, PagerDuty).
- Review and tune rules weekly during initial deployment, monthly thereafter.

---

## References

- [Suricata Documentation](https://docs.suricata.io/)
- [Suricata Rule Writing](https://docs.suricata.io/en/latest/rules/)
- [Snort 3 Documentation](https://www.snort.org/documents)
- [Wazuh Documentation](https://documentation.wazuh.com/)
- [OSSEC Documentation](https://www.ossec.net/docs/)
- [fail2ban Documentation](https://www.fail2ban.org/wiki/index.php/Main_Page)
- [Emerging Threats Open Rules](https://rules.emergingthreats.net/)
- [MITRE ATT&CK](https://attack.mitre.org/)
- [SANS IDS/IPS Resources](https://www.sans.org/reading-room/whitepapers/detection/)
- [CIS Benchmark for Suricata](https://www.cisecurity.org/)
