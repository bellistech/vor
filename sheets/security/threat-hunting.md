# Threat Hunting (Proactive Threat Detection and MITRE ATT&CK Mapping)

Systematic approach to proactive threat detection using MITRE ATT&CK,
log analysis, behavioral indicators, and hunting queries for Linux environments.

---

## 1. MITRE ATT&CK — Common TTPs

### Initial Access (TA0001)

| Technique | ID | What to Hunt |
|-----------|----|-------------|
| Phishing | T1566 | Unusual email attachments, links to newly registered domains |
| Exploit Public-Facing App | T1190 | WAF logs, unexpected POST to admin paths, SQLi patterns |
| Valid Accounts | T1078 | Off-hours logins, impossible travel, credential stuffing patterns |
| External Remote Services | T1133 | VPN/SSH from unusual geolocations or TOR exit nodes |

### Execution (TA0002)

| Technique | ID | What to Hunt |
|-----------|----|-------------|
| Command & Scripting Interpreter | T1059 | bash/python/perl spawned by web server process |
| Scheduled Task/Job | T1053 | New cron entries, at jobs, systemd timers |
| User Execution | T1204 | Executable downloads followed by execution |

### Persistence (TA0003)

| Technique | ID | What to Hunt |
|-----------|----|-------------|
| Create Account | T1136 | New entries in /etc/passwd, useradd in auth logs |
| SSH Authorized Keys | T1098.004 | Changes to authorized_keys files |
| Systemd Service | T1543.002 | New .service files in /etc/systemd/system/ |
| Cron Job | T1053.003 | Unexpected crontab modifications |
| Boot/Logon Init Scripts | T1037 | Changes to rc.local, .bashrc, .profile |

### Privilege Escalation (TA0004)

| Technique | ID | What to Hunt |
|-----------|----|-------------|
| Sudo Abuse | T1548.003 | sudo commands from unexpected users |
| SUID/SGID Exploitation | T1548.001 | New SUID binaries, unusual SUID in /tmp |
| Kernel Exploit | T1068 | Kernel panics, unexpected module loads |

### Lateral Movement (TA0008)

| Technique | ID | What to Hunt |
|-----------|----|-------------|
| SSH | T1021.004 | SSH from server-to-server (not from admin workstations) |
| Remote Services | T1021 | Unusual internal connections on management ports |

### Exfiltration (TA0010)

| Technique | ID | What to Hunt |
|-----------|----|-------------|
| Exfil Over C2 Channel | T1041 | Large outbound transfers to single IP |
| Exfil Over DNS | T1048.003 | High volume TXT queries, long subdomain names |
| Exfil Over Web Service | T1567 | Uploads to cloud storage (S3, GDrive, Dropbox) |

---

## 2. Suspicious Process Detection

```bash
# Processes running from unusual locations
ps auxwwf | grep -E '/tmp/|/var/tmp/|/dev/shm/|/run/user/'

# Processes with deleted binaries (common for in-memory malware)
ls -la /proc/*/exe 2>/dev/null | grep '(deleted)'

# Processes running as root that shouldn't be
ps -eo user,pid,ppid,cmd --sort=-pcpu | head -30

# Hidden processes (compare ps to /proc)
diff <(ps -eo pid --no-headers | sort -n) \
     <(ls /proc | grep -E '^[0-9]+$' | sort -n)

# Processes with unusual parent-child relationships
# Web server spawning shell:
ps -eo pid,ppid,user,cmd | grep -E '(apache|nginx|www-data)' | grep -v grep
pstree -p $(pgrep -f apache2 | head -1) 2>/dev/null

# Processes making outbound connections
ss -tunap | awk '$5 !~ /127\.0\.0\.1|::1/ && $2 > 0'

# Check for process injection / hollow processes
# Compare /proc/<pid>/exe to /proc/<pid>/maps
readlink /proc/<pid>/exe
cat /proc/<pid>/maps | head -5

# Unusual environment variables in process
cat /proc/<pid>/environ | tr '\0' '\n' | grep -iE 'proxy|password|token|key'
```

---

## 3. Persistence Mechanism Checks

```bash
# --- Cron ---
# All user crontabs
for user in $(cut -d: -f1 /etc/passwd); do
  echo "=== $user ===" && crontab -l -u "$user" 2>/dev/null
done

# System cron directories
ls -laR /etc/cron.* /var/spool/cron/ 2>/dev/null

# Recently modified cron files
find /etc/cron* /var/spool/cron -mtime -7 2>/dev/null

# --- Systemd ---
# Non-vendor services
systemctl list-unit-files --state=enabled --no-pager | \
  grep -v '/usr/lib/systemd'

# Recently created service files
find /etc/systemd/system /run/systemd/system -name "*.service" \
  -mtime -30 2>/dev/null

# --- SSH keys ---
find / -name "authorized_keys" -exec echo "=== {} ===" \; \
  -exec cat {} \; 2>/dev/null

# Keys added recently
find / -name "authorized_keys" -mtime -7 2>/dev/null

# --- Init scripts ---
ls -la /etc/rc.local /etc/init.d/ 2>/dev/null
grep -r "^[^#]" /etc/rc.local 2>/dev/null

# --- Shell profiles (backdoor in login scripts) ---
find /home /root -name ".bashrc" -o -name ".bash_profile" \
  -o -name ".profile" | xargs grep -l "curl\|wget\|nc\|ncat\|/dev/tcp" 2>/dev/null

# --- LD_PRELOAD hijacking ---
cat /etc/ld.so.preload 2>/dev/null
env | grep LD_PRELOAD
find / -name "ld.so.preload" 2>/dev/null

# --- Kernel modules ---
lsmod | sort
# Compare against known-good baseline
diff <(lsmod | awk '{print $1}' | sort) /baseline/modules_known_good.txt
```

---

## 4. Network Anomaly Indicators

```bash
# Unusual listening ports
ss -tlnp | awk '$4 !~ /127\.0\.0\.1|::1/'

# Connections to known-bad ports (IRC, crypto mining, etc.)
ss -tunap | grep -E ':6667|:6668|:6669|:4444|:5555|:3333|:8333|:9999'

# Large outbound data transfers
ss -tunap | awk '$2 > 1000000'  # Send queue > 1MB

# DNS exfiltration indicators
# Unusually long DNS queries (> 50 chars in subdomain)
tcpdump -i any -n port 53 -c 100 2>/dev/null | \
  awk '{print length($8), $8}' | sort -rn | head -20

# Beaconing detection — connections at regular intervals
# Export connection logs and look for periodic patterns
journalctl -u auditd --since "1 hour ago" | \
  grep -oP 'daddr=\K[^ ]+' | sort | uniq -c | sort -rn | head -20

# Connections to TOR exit nodes
# Download TOR exit list: https://check.torproject.org/torbulkexitlist
# Compare against active connections
comm -12 <(ss -tn | awk '{print $5}' | cut -d: -f1 | sort -u) \
         <(sort tor_exit_nodes.txt) 2>/dev/null

# Unusual ICMP (potential tunnel)
tcpdump -i any icmp -c 50 -n 2>/dev/null | \
  awk '{print $3, $5, length}' | sort | uniq -c | sort -rn
```

---

## 5. Lateral Movement Indicators

```bash
# SSH lateral movement — server-to-server SSH
grep "Accepted" /var/log/auth.log | \
  awk '{print $1,$2,$3,$9,$11}' | sort | uniq -c | sort -rn

# Look for SSH from non-admin IPs
grep "Accepted publickey" /var/log/auth.log | \
  grep -v "KNOWN_ADMIN_IPS"

# RDP / VNC from unexpected sources (if applicable)
ss -tnp | grep -E ':3389|:5900|:5901'

# Internal scanning (port sweep from single host)
journalctl -k --since "1 hour ago" | grep -i "drop" | \
  awk '{print $NF}' | sort | uniq -c | sort -rn | head -10

# WinRM/PSRemoting on Linux (unusual)
ss -tnp | grep ':5985\|:5986'

# Credential access — mass /etc/shadow reads
ausearch -k shadow_access 2>/dev/null | tail -20
```

---

## 6. Data Exfiltration Signs

```bash
# Large file compression before transfer
find /tmp /var/tmp /home -name "*.tar.gz" -o -name "*.zip" \
  -o -name "*.7z" -o -name "*.rar" -size +50M 2>/dev/null

# Unusual use of curl/wget
ps aux | grep -E 'curl|wget' | grep -v grep
# Check command history for upload commands
grep -hE 'curl.*-T|curl.*--upload|curl.*POST.*-d @|scp|rsync' \
  /home/*/.bash_history /root/.bash_history 2>/dev/null

# Base64 encoding (data staging)
ps aux | grep base64 | grep -v grep
grep -hE 'base64|xxd|openssl enc' \
  /home/*/.bash_history /root/.bash_history 2>/dev/null

# DNS tunneling — high TXT query volume
tcpdump -i any -n 'udp port 53' -c 500 2>/dev/null | \
  grep -c "TXT"

# Outbound traffic volume by destination
ss -tn | awk '{print $5}' | cut -d: -f1 | sort | uniq -c | sort -rn | head -20
```

---

## 7. Hunting Queries

### journalctl

```bash
# Failed SSH logins in last 24 hours
journalctl -u ssh --since "24 hours ago" | grep -i "failed"

# Sudo usage
journalctl --since "24 hours ago" | grep -i "sudo"

# User account changes
journalctl --since "7 days ago" | grep -iE "useradd|usermod|userdel|groupadd"

# Service start/stop events
journalctl --since "24 hours ago" | grep -iE "Started|Stopped|Failed" | \
  grep -v "session"

# Kernel messages (module loads, OOM, segfaults)
journalctl -k --since "7 days ago" | grep -iE "module|segfault|oom"
```

### auditd

```bash
# File access audit events
ausearch -f /etc/shadow --start recent
ausearch -f /etc/passwd --start recent

# Process execution events
ausearch -sc execve --start today | aureport -x --summary

# Failed syscalls (potential exploitation)
ausearch --success no --start today | aureport --summary

# Network connections by process
ausearch -sc connect --start today 2>/dev/null | tail -30

# Privilege escalation events
ausearch -m USER_AUTH,USER_ACCT --start today
```

### syslog / auth.log

```bash
# Brute force detection (>10 failed logins from same IP)
grep "Failed password" /var/log/auth.log | \
  awk '{print $(NF-3)}' | sort | uniq -c | sort -rn | \
  awk '$1 > 10'

# Successful login after failures (possible compromise)
grep -E "Failed password|Accepted" /var/log/auth.log | \
  grep -B5 "Accepted" | grep "Failed"

# su/sudo abuse
grep -E "su:|sudo:" /var/log/auth.log | grep -v "session opened"
```

---

## 8. Sigma Rules Basics

```yaml
# Sigma rule format — portable detection rule
title: Suspicious Process Execution from /tmp
id: a1b2c3d4-e5f6-7890-abcd-ef1234567890
status: experimental
description: Detects process execution from /tmp directory
author: Your SOC
date: 2026/01/01
logsource:
    category: process_creation
    product: linux
detection:
    selection:
        Image|startswith:
            - '/tmp/'
            - '/var/tmp/'
            - '/dev/shm/'
    condition: selection
falsepositives:
    - Legitimate software installers
    - Package managers during updates
level: high
tags:
    - attack.execution
    - attack.t1059
```

```bash
# Convert Sigma rules for your SIEM
# Install sigmac or sigma-cli
pip install sigma-cli

# Convert to Elasticsearch/ELK query
sigma convert -t elasticsearch -p ecs_windows rule.yml

# Convert to Splunk query
sigma convert -t splunk rule.yml

# Convert to grep pattern (quick local hunting)
# Manual approach for Linux process creation:
grep -rE '/tmp/|/var/tmp/|/dev/shm/' /var/log/audit/audit.log
```

---

## Tips

- Hunt on a regular schedule (weekly minimum), not just after incidents.
- Build and maintain a baseline of normal activity; deviations are your leads.
- Focus on TTPs, not IOCs; attackers change indicators, but techniques persist.
- Layer your hunts: combine network, host, and log data for higher confidence.
- Automate recurring hunts and graduate confirmed patterns into detection rules.
- Use threat intelligence feeds to prioritize which TTPs to hunt for.
- Document every hunt with hypothesis, data sources, findings, and follow-ups.
- Share findings with the SOC team to improve overall detection coverage.
- False positives are data; they help you tune detections and understand your
  environment better.

---

## References

- [MITRE ATT&CK for Linux](https://attack.mitre.org/matrices/enterprise/linux/)
- [MITRE ATT&CK Navigator](https://mitre-attack.github.io/attack-navigator/)
- [Sigma Rules Repository](https://github.com/SigmaHQ/sigma)
- [SANS Threat Hunting Summit Resources](https://www.sans.org/cyber-security-summit/archives/threat-hunting)
- [ThreatHunter Playbook](https://threathunterplaybook.com/)
- [Elastic Detection Rules](https://github.com/elastic/detection-rules)
- [Awesome Threat Detection](https://github.com/0x4D31/awesome-threat-detection)
- [NIST SP 800-150 — Guide to Cyber Threat Information Sharing](https://csrc.nist.gov/publications/detail/sp/800-150/final)
