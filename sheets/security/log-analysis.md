# Security Log Analysis (Detection Patterns and Query Techniques)

Practical reference for analyzing auth logs, syslog, journalctl, auditd,
and web server access logs to detect attacks, privilege escalation, and
anomalous behavior.

---

## 1. Auth Log Analysis (/var/log/auth.log)

### Common Patterns

```bash
# Location varies by distro:
# Debian/Ubuntu: /var/log/auth.log
# RHEL/CentOS:   /var/log/secure

# --- Failed SSH logins ---
grep "Failed password" /var/log/auth.log

# Count failed logins per IP
grep "Failed password" /var/log/auth.log | \
  awk '{print $(NF-3)}' | sort | uniq -c | sort -rn | head -20

# Count failed logins per username
grep "Failed password" /var/log/auth.log | \
  awk '{for(i=1;i<=NF;i++) if($i=="for") print $(i+1)}' | \
  sort | uniq -c | sort -rn | head -20

# --- Successful logins ---
grep "Accepted" /var/log/auth.log | \
  awk '{print $1,$2,$3,$9,$11}' | sort | uniq -c | sort -rn

# Successful login from unusual source after failures (compromise indicator)
grep -E "Failed password|Accepted" /var/log/auth.log | \
  awk '/Failed password/{fail[$NF]++} /Accepted/{if(fail[$NF]>3) print}'

# --- Invalid/unknown users ---
grep "Invalid user" /var/log/auth.log | \
  awk '{print $8}' | sort | uniq -c | sort -rn | head -20

# --- Root login attempts ---
grep -E "Failed password.*root|Accepted.*root" /var/log/auth.log

# --- su / sudo activity ---
grep "sudo:" /var/log/auth.log | grep -v "session"
grep "su:" /var/log/auth.log

# Sudo failures (possible privesc attempts)
grep "sudo:" /var/log/auth.log | grep "NOT"
# Output: user NOT in sudoers ; TTY=pts/0 ; PWD=/home/user ; USER=root ; COMMAND=/bin/bash

# --- SSH key authentication ---
grep "Accepted publickey" /var/log/auth.log | \
  awk '{print $1,$2,$3,$9,$11,$NF}'

# --- Account changes ---
grep -E "useradd|userdel|usermod|groupadd|passwd" /var/log/auth.log

# --- PAM events ---
grep "pam_unix" /var/log/auth.log | grep -v "session opened\|session closed"
```

### Brute Force Detection

```bash
# More than 10 failures from same IP in auth.log
grep "Failed password" /var/log/auth.log | \
  awk '{print $(NF-3)}' | sort | uniq -c | \
  awk '$1 > 10 {print $0}' | sort -rn

# Distributed brute force (many IPs, same username)
grep "Failed password" /var/log/auth.log | \
  grep "for root" | \
  awk '{print $(NF-3)}' | sort -u | wc -l
# If count is very high, likely a botnet attack

# Time-based analysis (failures per hour)
grep "Failed password" /var/log/auth.log | \
  awk '{print $1,$2,substr($3,1,2)":00"}' | \
  sort | uniq -c | sort -rn
```

---

## 2. Syslog Analysis

```bash
# --- Kernel messages ---
grep "kernel:" /var/log/syslog | grep -iE "segfault|oom|error|panic"

# --- Service failures ---
grep -iE "failed|error|fatal" /var/log/syslog | \
  grep -v "Failed password"   # exclude auth noise

# --- Unusual process starts ---
grep "systemd.*Started" /var/log/syslog | \
  awk '{for(i=6;i<=NF;i++) printf "%s ",$i; print ""}' | \
  sort | uniq -c | sort -rn | head -20

# --- Network events ---
grep -iE "dhcp|dns|interface" /var/log/syslog

# --- Cron execution ---
grep "CRON" /var/log/syslog | tail -20

# --- Suspicious cron commands ---
grep "CRON" /var/log/syslog | \
  grep -iE "curl|wget|nc|ncat|python|perl|bash -i|/dev/tcp"

# --- USB device connections ---
grep -iE "usb|removable" /var/log/syslog

# --- Firewall drops ---
grep "IPTABLES-DROP\|UFW BLOCK" /var/log/syslog | \
  awk '{for(i=1;i<=NF;i++) if($i~/SRC=/) print $i}' | \
  sort | uniq -c | sort -rn | head -20
```

---

## 3. journalctl Security Queries

```bash
# --- Authentication failures (last 24 hours) ---
journalctl -u ssh --since "24 hours ago" --no-pager | grep -i "failed"

# --- All auth events ---
journalctl _COMM=sshd --since "24 hours ago" --no-pager

# --- Priority-based filtering ---
# Emergency (0) through Error (3)
journalctl -p err --since "24 hours ago" --no-pager

# Critical and above
journalctl -p crit --since "7 days ago" --no-pager

# --- Kernel messages ---
journalctl -k --since "24 hours ago" --no-pager
journalctl -k | grep -iE "module|segfault|oom-killer|apparmor|selinux"

# --- Boot-related events ---
journalctl -b -1    # Previous boot (useful after crash)
journalctl --list-boots   # List all boots

# --- Service-specific ---
journalctl -u nginx --since "1 hour ago" --no-pager
journalctl -u docker --since "1 hour ago" --no-pager

# --- User-specific ---
journalctl _UID=1001 --since "24 hours ago" --no-pager

# --- Sudo events ---
journalctl _COMM=sudo --since "24 hours ago" --no-pager

# --- New units started (potential persistence) ---
journalctl --since "7 days ago" | grep -i "Started" | \
  awk '{for(i=5;i<=NF;i++) printf "%s ",$i; print ""}' | \
  sort -u

# --- Failed systemd units ---
systemctl --failed

# --- JSON output for automated processing ---
journalctl -u ssh --since "1 hour ago" -o json --no-pager | \
  python3 -c "import sys,json; [print(json.loads(l).get('MESSAGE','')) for l in sys.stdin]"

# --- Follow live (tail) ---
journalctl -f -u ssh
journalctl -f -p warning
```

---

## 4. auditd Analysis (aureport / ausearch)

### aureport (Summary Reports)

```bash
# Authentication report
aureport --auth --start today

# Login report
aureport --login --start today --summary

# Failed events
aureport --failed --start today

# Executable report (what was run)
aureport -x --start today --summary

# File access report
aureport -f --start today --summary

# User report
aureport -u --start today --summary

# Account modification events
aureport -m --start today

# Anomaly report
aureport --anomaly

# Key-based report (using audit rule keys)
aureport -k --start today
```

### ausearch (Detailed Search)

```bash
# Search by audit rule key
ausearch -k identity --start today          # identity changes
ausearch -k sudoers --start today           # sudoers modifications
ausearch -k modules --start today           # kernel module events
ausearch -k exec --start today              # process execution
ausearch -k cron --start today              # cron changes

# Search by file
ausearch -f /etc/passwd --start today
ausearch -f /etc/shadow --start today

# Search by user
ausearch -ua 1001 --start today             # by UID
ausearch -ua root --start today             # by username

# Search by syscall
ausearch -sc execve --start today           # program execution
ausearch -sc connect --start today          # network connections
ausearch -sc open --start today             # file opens

# Search by success/failure
ausearch --success no --start today         # failed operations

# Search by event type
ausearch -m USER_AUTH --start today         # authentication
ausearch -m USER_CMD --start today          # commands via sudo
ausearch -m SYSCALL --start today           # syscalls

# Combine filters
ausearch -ua root -sc execve --success yes --start today

# Output in interpretable format
ausearch -k identity --start today -i       # human-readable

# Pipe to aureport for summarization
ausearch -sc execve --start today --raw | aureport -x --summary
```

---

## 5. Web Server Log Attack Patterns

### Apache / Nginx Access Logs

```bash
# Log format (combined):
# IP - - [date] "METHOD /path HTTP/1.1" status size "referer" "user-agent"

# --- SQL Injection attempts ---
grep -iE "union.*select|select.*from|insert.*into|drop.*table|or.*1=1|'--" \
  /var/log/nginx/access.log

# --- Directory traversal ---
grep -E "\.\./|\.\.%2f|\.\.%252f" /var/log/nginx/access.log

# --- Web shell access ---
grep -iE "(cmd|shell|exec|system|passthru)=" /var/log/nginx/access.log

# --- Scanner/enumeration ---
grep -E "\.(php|asp|aspx|jsp|cgi|env|git|bak|old|sql|conf)" \
  /var/log/nginx/access.log | grep " 404 "

# --- WordPress/CMS attacks ---
grep -iE "wp-login|wp-admin|xmlrpc\.php|wp-config" \
  /var/log/nginx/access.log

# --- XSS attempts ---
grep -iE "<script|javascript:|onerror=|onload=|alert\(" \
  /var/log/nginx/access.log

# --- Suspicious user agents ---
grep -iE "nikto|sqlmap|nmap|masscan|zgrab|gobuster|dirbuster|wfuzz" \
  /var/log/nginx/access.log

# --- Unusual HTTP methods ---
grep -vE '"(GET|POST|HEAD) ' /var/log/nginx/access.log

# --- Large response sizes (potential data exfil) ---
awk '{if($10 > 10000000) print $1,$7,$9,$10}' /var/log/nginx/access.log | \
  sort -t' ' -k4 -rn | head -20

# --- Top IPs by request count ---
awk '{print $1}' /var/log/nginx/access.log | \
  sort | uniq -c | sort -rn | head -20

# --- Top IPs with 4xx/5xx errors ---
awk '$9 ~ /^[45]/ {print $1,$9}' /var/log/nginx/access.log | \
  sort | uniq -c | sort -rn | head -20

# --- Requests per hour (detect spikes) ---
awk '{print $4}' /var/log/nginx/access.log | \
  cut -d: -f1-2 | sort | uniq -c | sort -rn | head -24
```

### Error Logs

```bash
# PHP errors that might indicate exploitation
grep -iE "fatal error|parse error|eval|base64_decode|assert" \
  /var/log/nginx/error.log

# ModSecurity alerts (if enabled)
grep -i "modsecurity" /var/log/nginx/error.log | \
  grep -oP 'id "\K[^"]+' | sort | uniq -c | sort -rn
```

---

## 6. Privilege Escalation Indicators

```bash
# --- Sudo to root ---
grep "sudo:.*COMMAND=" /var/log/auth.log | grep "USER=root"

# --- su to root ---
grep "su:" /var/log/auth.log | grep "root"

# --- Unexpected root processes ---
ps -eo user,pid,ppid,cmd | awk '$1=="root"' | \
  grep -vE "sshd|systemd|kworker|cron|rsyslog|agetty"

# --- SUID exploitation (audit log) ---
ausearch -sc execve --start today -i | grep -i "suid\|sgid"

# --- Capability changes ---
ausearch -m PROCTITLE --start today | grep -i "cap_"

# --- setuid/setgid syscalls ---
ausearch -sc setuid --start today
ausearch -sc setgid --start today

# --- Kernel exploit indicators ---
journalctl -k | grep -iE "segfault|general protection|stack|overflow"
dmesg | grep -iE "segfault|oops|bug|rip"

# --- Cron-based privesc ---
# Look for writable files in cron that run as root
find /etc/cron* -writable 2>/dev/null
ls -la /etc/crontab

# --- PATH hijacking detection ---
# Check if any PATH directories are writable by non-root
echo $PATH | tr ':' '\n' | while read d; do
  [ -w "$d" ] && echo "WRITABLE: $d"
done
```

---

## 7. Log Correlation

### Multi-Source Correlation

```bash
# Timeline of events around a suspected incident
# Combine auth, syslog, and audit for a time window

echo "=== AUTH LOG ===" > /tmp/correlated.txt
grep "2026-01-15T1[4-6]" /var/log/auth.log >> /tmp/correlated.txt 2>/dev/null

echo "=== SYSLOG ===" >> /tmp/correlated.txt
grep "Jan 15 1[4-6]" /var/log/syslog >> /tmp/correlated.txt 2>/dev/null

echo "=== AUDIT ===" >> /tmp/correlated.txt
ausearch --start "01/15/2026" "14:00:00" --end "01/15/2026" "17:00:00" \
  >> /tmp/correlated.txt 2>/dev/null

echo "=== WEB ACCESS ===" >> /tmp/correlated.txt
grep "15/Jan/2026:1[4-6]" /var/log/nginx/access.log >> /tmp/correlated.txt 2>/dev/null

# Sort by timestamp for unified timeline
sort -t' ' -k1,2 /tmp/correlated.txt | less
```

### Cross-Log Detection Patterns

```bash
# Pattern: SSH login followed by suspicious activity
# 1. Find successful SSH login
login_time=$(grep "Accepted" /var/log/auth.log | tail -1 | awk '{print $3}')
login_user=$(grep "Accepted" /var/log/auth.log | tail -1 | awk '{print $9}')

# 2. Check what they did after login
grep "$login_user" /var/log/auth.log | grep -A5 "Accepted"
ausearch -ua "$login_user" --start recent

# Pattern: Web attack followed by shell access
# 1. Find attacking IP from web logs
attacker_ip=$(grep -iE "union.*select|cmd=" /var/log/nginx/access.log | \
  awk '{print $1}' | sort -u | head -1)

# 2. Check if same IP appears in SSH logs
grep "$attacker_ip" /var/log/auth.log

# Pattern: New user creation followed by SSH login
grep "useradd" /var/log/auth.log
# Cross-reference with SSH accepted logins for that user
```

---

## 8. One-Liners for Security Events

```bash
# Top 10 source IPs for failed SSH
grep "Failed password" /var/log/auth.log | \
  awk '{print $(NF-3)}' | sort | uniq -c | sort -rn | head -10

# Top targeted usernames
grep "Failed password" /var/log/auth.log | \
  sed -n 's/.*for \(invalid user \)\?\([^ ]*\) from.*/\2/p' | \
  sort | uniq -c | sort -rn | head -10

# Successful logins outside business hours (before 7am, after 7pm)
grep "Accepted" /var/log/auth.log | \
  awk -F: '{h=$1; sub(/.*T/,"",h); if(h<7||h>19) print}'

# Unique SSH client versions (fingerprinting)
grep "SSH" /var/log/auth.log | \
  grep -oP 'SSH-[^ ]+' | sort | uniq -c | sort -rn

# Failed sudo attempts by user
grep "sudo:" /var/log/auth.log | grep "NOT in sudoers" | \
  awk -F: '{print $NF}' | sort | uniq -c | sort -rn

# Count all 4xx errors per URL in web logs
awk '$9 ~ /^4/ {print $9,$7}' /var/log/nginx/access.log | \
  sort | uniq -c | sort -rn | head -20

# Extract all IPs that triggered firewall blocks
grep "IPTABLES-DROP\|UFW BLOCK" /var/log/syslog | \
  grep -oP 'SRC=\K[0-9.]+' | sort | uniq -c | sort -rn | head -20

# Find commands run via sudo in last 24 hours
grep "sudo:" /var/log/auth.log | grep "COMMAND=" | \
  sed 's/.*COMMAND=//' | sort | uniq -c | sort -rn | head -20

# Connections per minute (detect DoS)
awk '{print $4}' /var/log/nginx/access.log | \
  cut -d: -f1-3 | sort | uniq -c | sort -rn | head -10

# Extract POST requests with large bodies (potential upload/exfil)
awk '$6=="\"POST" && $10>100000 {print $1,$7,$10}' \
  /var/log/nginx/access.log | sort -t' ' -k3 -rn
```

---

## Tips

- Centralize logs to a SIEM or remote syslog server; local logs can be tampered
  with by attackers.
- Set log retention policies that meet compliance requirements (typically 90
  days minimum, 1 year for many regulations).
- Normalize timestamps to UTC across all log sources.
- Create baseline dashboards for normal log volume; sudden drops may indicate
  log tampering, sudden spikes may indicate attacks.
- Use logrotate to manage disk space, but ensure rotated logs are also forwarded
  to central storage.
- Enable verbose logging during incident response, but be mindful of disk space.
- Automate recurring queries as cron jobs that alert on anomalies.
- Protect log integrity with append-only permissions or immutable flags where
  possible: `chattr +a /var/log/auth.log`.
- Build a library of grep/awk one-liners for your environment and share with
  the team.
- Test your log pipeline regularly; verify that events from every source
  reach your SIEM.

---

## See Also

- auditd, ids-ips, threat-hunting, incident-response, forensics

## References

- [NIST SP 800-92 — Guide to Computer Security Log Management](https://csrc.nist.gov/publications/detail/sp/800-92/final)
- [SANS SEC555 — SIEM with Tactical Analytics](https://www.sans.org/cyber-security-courses/siem-with-tactical-analytics/)
- [MITRE ATT&CK Data Sources](https://attack.mitre.org/datasources/)
- [Linux Audit Documentation (auditd)](https://man7.org/linux/man-pages/man8/auditd.8.html)
- [aureport Manual](https://man7.org/linux/man-pages/man8/aureport.8.html)
- [ausearch Manual](https://man7.org/linux/man-pages/man8/ausearch.8.html)
- [journalctl Manual](https://www.freedesktop.org/software/systemd/man/journalctl.html)
- [Elastic Common Schema (ECS)](https://www.elastic.co/guide/en/ecs/current/index.html)
- [Sigma Rules — Generic Log Signatures](https://github.com/SigmaHQ/sigma)
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
