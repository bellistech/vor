# Incident Response (Blue Team IR Methodology and Commands)

Quick-reference for the full NIST SP 800-61 incident response lifecycle with
practical commands, evidence handling, containment playbooks, and post-mortem
templates.

---

## 1. NIST IR Lifecycle Overview

```
Prepare --> Detect & Analyze --> Contain --> Eradicate --> Recover --> Lessons Learned
   |              |                 |            |            |              |
   v              v                 v            v            v              v
 Tooling      Triage/IOC       Short+Long    Root cause   Restore       Post-mortem
 Runbooks     Severity         term hold     removal      services      Update playbooks
```

### Phase Checklist

| Phase | Key Actions |
|-------|-------------|
| Prepare | Asset inventory, contact lists, jump bag, playbooks, tabletop exercises |
| Detect & Analyze | Alert triage, IOC collection, scope assessment, severity rating |
| Contain | Network isolation, credential rotation, block IOCs, preserve evidence |
| Eradicate | Remove malware, patch vulnerabilities, close unauthorized access |
| Recover | Restore from clean backups, monitored re-introduction, validation |
| Lessons Learned | Timeline reconstruction, root-cause analysis, control improvements |

---

## 2. Preparation

### Jump Bag Essentials

```bash
# Minimum IR toolkit on a prepared USB drive
# - Live Linux distro (SIFT, REMnux, or Kali)
# - Write blockers (hardware or software)
# - Evidence collection scripts
# - Forensic imaging tools (dc3dd, ewfacquire)
# - Pre-compiled static binaries (busybox, netcat, tcpdump)
# - Chain of custody forms (printed)
# - Network cables, USB-to-serial adapters
```

### Contact Sheet Template

```
IR Lead:            ___________________  Phone: ___________
Security Analyst:   ___________________  Phone: ___________
IT Operations:      ___________________  Phone: ___________
Legal Counsel:      ___________________  Phone: ___________
Communications/PR:  ___________________  Phone: ___________
CISO:               ___________________  Phone: ___________
External IR Firm:   ___________________  Phone: ___________
Law Enforcement:    ___________________  Phone: ___________
Cyber Insurance:    ___________________  Phone: ___________
```

---

## 3. Detection and Analysis

### Severity Classification

| Level | Name | Description | Response Time |
|-------|------|-------------|---------------|
| P1 | Critical | Active data exfil, ransomware spreading, root compromise | Immediate |
| P2 | High | Confirmed intrusion, lateral movement detected | < 1 hour |
| P3 | Medium | Suspicious activity, single host compromise | < 4 hours |
| P4 | Low | Policy violation, reconnaissance detected | < 24 hours |

### Initial Triage Commands

```bash
# Capture volatile data FIRST (order of volatility)
# 1. Registers, cache
# 2. Memory (RAM)
# 3. Network state
# 4. Running processes
# 5. Disk
# 6. Remote logging / monitoring data
# 7. Physical configuration / network topology
# 8. Archival media / backups

# --- System snapshot ---
date -u > /tmp/ir_$(hostname)_$(date +%Y%m%d).txt
uname -a >> /tmp/ir_$(hostname)_$(date +%Y%m%d).txt
uptime >> /tmp/ir_$(hostname)_$(date +%Y%m%d).txt
who -a >> /tmp/ir_$(hostname)_$(date +%Y%m%d).txt

# --- Network state (volatile) ---
ss -tunap > /tmp/ir_netstat.txt
ip addr show > /tmp/ir_ipaddr.txt
ip route show > /tmp/ir_routes.txt
arp -a > /tmp/ir_arp.txt
iptables -L -n -v > /tmp/ir_iptables.txt

# --- Running processes ---
ps auxwwf > /tmp/ir_ps.txt
ls -la /proc/*/exe 2>/dev/null > /tmp/ir_proc_exe.txt
ls -la /proc/*/fd 2>/dev/null > /tmp/ir_proc_fd.txt

# --- Logged-in users and recent logins ---
w > /tmp/ir_who.txt
last -aiF > /tmp/ir_last.txt
lastb -aiF 2>/dev/null > /tmp/ir_lastb.txt

# --- Scheduled tasks ---
for u in $(cut -d: -f1 /etc/passwd); do
  crontab -l -u "$u" 2>/dev/null && echo "--- $u ---"
done > /tmp/ir_crontabs.txt
ls -la /etc/cron.* > /tmp/ir_crondirs.txt

# --- Open files ---
lsof -nP > /tmp/ir_lsof.txt
```

### IOC Collection

```bash
# Collect indicators of compromise

# File hashes of suspicious binaries
sha256sum /path/to/suspicious_file
md5sum /path/to/suspicious_file

# Collect all hashes in a directory
find /tmp /var/tmp /dev/shm -type f -exec sha256sum {} \; > /tmp/ir_hashes.txt

# DNS cache (if systemd-resolved)
resolvectl statistics
resolvectl query --cache-dump 2>/dev/null

# Recent file modifications (last 24 hours)
find / -mtime -1 -type f -not -path "/proc/*" -not -path "/sys/*" \
  2>/dev/null > /tmp/ir_recent_files.txt

# Check for unusual SUID/SGID binaries
find / -perm /6000 -type f 2>/dev/null > /tmp/ir_suid_sgid.txt

# Yara scanning (if yara installed)
yara -r /path/to/rules.yar /path/to/scan/
```

---

## 4. Evidence Preservation

### Memory Acquisition

```bash
# Using LiME (Linux Memory Extractor) — kernel module
sudo insmod lime-$(uname -r).ko "path=/evidence/memory.lime format=lime"

# Using AVML (Microsoft's Acquire Volatile Memory for Linux)
sudo ./avml /evidence/memory.lime

# Verify memory dump
sha256sum /evidence/memory.lime > /evidence/memory.lime.sha256

# Using Volatility 3 for analysis
vol3 -f /evidence/memory.lime linux.pslist
vol3 -f /evidence/memory.lime linux.bash
vol3 -f /evidence/memory.lime linux.netstat
vol3 -f /evidence/memory.lime linux.lsof
```

### Disk Imaging

```bash
# Create forensic disk image with dc3dd (preferred over dd)
dc3dd if=/dev/sda of=/evidence/disk.dd hash=sha256 log=/evidence/disk.log

# Alternative with dcfldd
dcfldd if=/dev/sda of=/evidence/disk.dd hash=sha256 \
  hashlog=/evidence/disk.hashlog bs=4096

# Verify image integrity
sha256sum /evidence/disk.dd
# Compare with hash in log file

# Mount image read-only for examination
mount -o ro,loop,noexec,nosuid /evidence/disk.dd /mnt/evidence
```

### Timeline Creation

```bash
# Generate filesystem timeline with plaso/log2timeline
log2timeline.py /evidence/timeline.plaso /evidence/disk.dd

# Filter and output timeline
psort.py -o l2tcsv /evidence/timeline.plaso \
  "date > '2026-01-01' AND date < '2026-01-15'" \
  -w /evidence/timeline.csv

# Quick filesystem timeline with find
find /mnt/evidence -printf '%T+ %p\n' 2>/dev/null | sort > /tmp/ir_fs_timeline.txt

# MAC times for specific directory
stat /mnt/evidence/tmp/* 2>/dev/null > /tmp/ir_stat_tmp.txt
```

### Chain of Custody Log

```
CHAIN OF CUSTODY RECORD
=======================
Case Number:    IR-2026-____
Item Number:    EVID-____
Description:    ________________________________
Serial/Asset:   ________________________________
Hash (SHA-256): ________________________________

Date/Time (UTC)   | Action           | From        | To          | Signature
-------------------|------------------|-------------|-------------|----------
                   | Collected        |             |             |
                   | Transferred      |             |             |
                   | Stored           |             |             |
                   | Analyzed         |             |             |
                   | Returned         |             |             |
```

---

## 5. Containment Strategies

### Short-Term Containment

```bash
# Network isolation — block all traffic except IR team
iptables -I INPUT -s <ir_team_ip> -j ACCEPT
iptables -I OUTPUT -d <ir_team_ip> -j ACCEPT
iptables -A INPUT -j DROP
iptables -A OUTPUT -j DROP

# Disable compromised account (do NOT delete)
usermod -L compromised_user
passwd -l compromised_user

# Kill malicious process but preserve evidence first
cp /proc/<pid>/exe /evidence/malware_sample_$(date +%s)
cp /proc/<pid>/maps /evidence/proc_maps_$(date +%s)
kill -STOP <pid>   # STOP first, preserve state
# Only kill after evidence collected:
# kill -9 <pid>

# Block known-bad IP at firewall
iptables -I INPUT -s <bad_ip> -j DROP
iptables -I OUTPUT -d <bad_ip> -j DROP

# Block malicious domain at DNS level
echo "0.0.0.0 malicious.example.com" >> /etc/hosts
```

### Long-Term Containment

```bash
# Move to isolated VLAN (switch-level, coordinate with network team)
# Rebuild from known-good image if available
# Rotate ALL credentials the compromised system had access to

# Rotate SSH keys
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519_new -C "rotated $(date +%F)"

# Force password changes for affected users
chage -d 0 affected_user

# Revoke and regenerate API keys / tokens
# (application-specific — check each service)

# Update firewall rules to block C2 channels
# Add IOCs to blocklists across the environment
```

---

## 6. Eradication

```bash
# Remove malware artifacts
rm -f /path/to/malware       # Only after evidence is preserved
rm -f /tmp/.hidden_backdoor

# Remove persistence mechanisms
# Check and clean:
#   - crontabs (crontab -l -u <user>)
#   - systemd services (/etc/systemd/system/)
#   - init scripts (/etc/init.d/)
#   - .bashrc / .profile modifications
#   - authorized_keys entries
#   - at jobs (atq)

# Remove rogue SSH keys
grep -r "AAAA" /home/*/.ssh/authorized_keys
# Remove unauthorized entries

# Remove rogue systemd services
systemctl list-unit-files --state=enabled | grep suspicious
systemctl disable --now suspicious.service
rm /etc/systemd/system/suspicious.service
systemctl daemon-reload

# Patch the vulnerability that was exploited
apt update && apt upgrade -y   # Debian/Ubuntu
# or
yum update -y                  # RHEL/CentOS
```

---

## 7. Recovery

```bash
# Restore from known-good backup
# Verify backup integrity first
sha256sum backup_file.tar.gz

# Restore with validation
tar -xzf backup_file.tar.gz -C /restored/
diff -rq /restored/ /expected/

# Monitored re-introduction
# - Enable enhanced logging on restored systems
# - Set up alerting for IOCs related to this incident
# - Gradually restore network access

# Verify clean state
rkhunter --check --skip-keypress
chkrootkit
clamscan -r /

# Monitor for re-infection (first 72 hours are critical)
auditctl -w /etc/passwd -p wa -k post_ir_monitor
auditctl -w /etc/shadow -p wa -k post_ir_monitor
auditctl -w /tmp -p x -k post_ir_monitor
```

---

## 8. Lessons Learned / Post-Mortem

### Post-Mortem Template

```markdown
# Incident Post-Mortem: IR-2026-XXXX

## Summary
- **Incident type:** [malware / unauthorized access / data breach / DoS / etc.]
- **Severity:** P1 / P2 / P3 / P4
- **Duration:** [detection time] to [resolution time]
- **Impact:** [systems affected, data exposed, business impact]
- **Detection method:** [alert / user report / threat hunt / external notification]

## Timeline (UTC)
| Time | Event |
|------|-------|
| YYYY-MM-DD HH:MM | Initial compromise (estimated) |
| YYYY-MM-DD HH:MM | First alert triggered |
| YYYY-MM-DD HH:MM | IR team engaged |
| YYYY-MM-DD HH:MM | Containment achieved |
| YYYY-MM-DD HH:MM | Eradication complete |
| YYYY-MM-DD HH:MM | Recovery complete |

## Root Cause
[Detailed technical explanation of how the incident occurred]

## What Went Well
- [Item 1]
- [Item 2]

## What Could Be Improved
- [Item 1 — with action item and owner]
- [Item 2 — with action item and owner]

## Action Items
| # | Action | Owner | Due Date | Status |
|---|--------|-------|----------|--------|
| 1 |        |       |          |        |

## Indicators of Compromise
- File hashes: [SHA-256 values]
- IP addresses: [C2 IPs]
- Domains: [malicious domains]
- File paths: [malware locations]
- User agents: [suspicious UAs]
```

### Communication Template (Internal)

```
SECURITY INCIDENT NOTIFICATION — [CLASSIFICATION]
===================================================
Incident ID:   IR-2026-XXXX
Severity:      P__
Status:        [Active / Contained / Resolved]

SUMMARY:
[One-paragraph description of the incident and current status]

IMPACT:
- Systems affected: [list]
- Data at risk: [description]
- Business functions impacted: [list]

CURRENT ACTIONS:
- [What the IR team is doing now]

REQUIRED ACTIONS:
- [What recipients need to do — e.g., change passwords, avoid certain systems]

NEXT UPDATE:
- Expected by [time] or sooner if status changes

Contact: [IR Lead name and phone]
```

---

## Tips

- Always collect volatile evidence first (memory, network state, processes) before
  touching disk.
- Never run forensic tools on the evidence drive itself; work on copies.
- Document everything with timestamps in UTC.
- Take screenshots or photos of screens showing anomalous activity.
- Use write-blockers when imaging disks.
- Do not tip off the attacker during investigation if possible; avoid actions
  they might detect (killing their process, changing firewall rules they monitor).
- Preserve logs off-host immediately; attackers routinely clear local logs.
- If in doubt about legal requirements, engage legal counsel before taking action
  on evidence.
- Test your IR plan regularly with tabletop exercises.
- Keep a "lessons learned" backlog and review it quarterly.

---

## See Also

- forensics, log-analysis, threat-hunting, auditd, ids-ips

## References

- [NIST SP 800-61 Rev. 2 — Computer Security Incident Handling Guide](https://csrc.nist.gov/publications/detail/sp/800-61/rev-2/final)
- [NIST SP 800-86 — Guide to Integrating Forensic Techniques](https://csrc.nist.gov/publications/detail/sp/800-86/final)
- [SANS Incident Handler's Handbook](https://www.sans.org/white-papers/33901/)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [FIRST — Forum of Incident Response and Security Teams](https://www.first.org/)
- [RFC 2350 — Expectations for Computer Security Incident Response](https://datatracker.ietf.org/doc/html/rfc2350)
- [Volatility 3 Documentation](https://volatility3.readthedocs.io/)
- [AVML — Acquire Volatile Memory for Linux](https://github.com/microsoft/avml)
