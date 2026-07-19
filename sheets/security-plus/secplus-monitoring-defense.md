# Security Monitoring & Enterprise Defense (Security+ SY0-701 Objectives 4.4 + 4.5)

> Alerting and monitoring concepts and tools (4.4) plus modifying enterprise capabilities to enhance security (4.5): SIEM, SCAP, SNMP, NetFlow, DLP, firewalls, IDS/IPS, web/DNS filtering, secure protocol selection, email authentication (SPF/DKIM/DMARC), FIM, NAC, EDR/XDR, and user behavior analytics.

## Monitoring Computing Resources (4.4)

```
# What gets monitored — the exam names three resource classes:
#
# Systems         — hosts/servers/endpoints: CPU, memory, disk, process
#                   list, service state, auth logs (Windows Event Log,
#                   Linux /var/log/auth.log, journald), registry changes
# Applications    — app-specific logs (web server access/error logs,
#                   database audit logs, API gateway logs), error rates,
#                   response times, failed logins to the app itself
# Infrastructure  — network devices (routers, switches, firewalls,
#                   wireless controllers), hypervisors, cloud control
#                   plane (e.g. AWS CloudTrail), physical sensors
#                   (badge readers, cameras), power/HVAC
#
# Exam cue: a question that lists "servers, applications, and network
# devices" and asks for ONE pane of glass → SIEM (Security Information
# and Event Management) with log aggregation.
```

## Monitoring Activities (4.4)

```
# Log aggregation — collect logs from many sources into one searchable
#   store (syslog → SIEM). Normalization = converting different vendor
#   formats into one schema; correlation = linking events across
#   sources into one incident (3 failed VPN logins + 1 success from a
#   new country = correlated alert).
#
# Alerting — automated notification when a rule/threshold/anomaly
#   fires. Push (email, SMS, ticket, pager) not pull.
#
# Scanning — scheduled vulnerability scans, configuration-compliance
#   scans, port scans of your own estate. Continuous or periodic.
#
# Reporting — human-readable rollups for management/compliance:
#   trend reports, executive dashboards, audit evidence.
#
# Archiving — long-term log retention (compliance often mandates
#   1 yr+; PCI DSS: 1 year, 3 months immediately available).
#   Archived ≠ deleted; must remain tamper-evident (WORM storage,
#   hashing) for forensic/legal admissibility.
#
# Alert response and remediation/validation:
#   Quarantine     — isolate the affected host/file/mailbox so the
#                    threat cannot spread while you investigate
#                    (EDR network isolation, AV file quarantine, NAC
#                    quarantine VLAN)
#   Alert tuning   — adjusting rules/thresholds to cut FALSE POSITIVES
#                    without creating false negatives; the fix for
#                    "analysts are overwhelmed by alerts" and
#                    "alert fatigue" questions
#
# Exam cue keywords:
#   "too many alerts / analysts ignore alerts"      → alert tuning
#   "stop malware spreading during investigation"   → quarantine
#   "prove logs weren't altered"                    → hashing / WORM archive
#   "combine events from many sources"              → log aggregation (SIEM)
```

## Monitoring Tools (4.4)

```
# SCAP — Security Content Automation Protocol (NIST SP 800-126):
#   a suite of open standards that lets tools measure and enforce
#   security configuration automatically. Component languages:
#     OVAL  — Open Vulnerability and Assessment Language (how to test)
#     XCCDF — Extensible Configuration Checklist Description Format
#             (the checklist/benchmark itself)
#     CPE   — Common Platform Enumeration (names products)
#     CVE   — Common Vulnerabilities and Exposures (names flaws)
#     CVSS  — Common Vulnerability Scoring System (scores severity 0-10)
#   Exam cue: "automate configuration-compliance checking across
#   vendors" / "machine-readable security checklist" → SCAP
#
# Benchmarks — hardening baselines you scan against: CIS (Center for
#   Internet Security) Benchmarks, DISA STIGs (Security Technical
#   Implementation Guides), vendor baselines (Microsoft Security
#   Baseline). Consumed by SCAP-validated scanners.
#
# Agents vs agentless:
#   Agent-based  — software installed on each host; deep visibility,
#                  works offline/off-network, real-time; cost =
#                  deployment/maintenance overhead
#   Agentless    — scans over the network with credentials (SSH/WinRM/
#                  SNMP/API); nothing to install; only sees the host
#                  when reachable, shallower data
#
# SIEM — Security Information and Event Management:
#   aggregation + normalization + correlation + alerting + dashboards
#   + retention. Examples: Splunk, Microsoft Sentinel, QRadar, Elastic.
#   SOAR (Security Orchestration, Automation, and Response) sits on
#   top: playbooks/runbooks that automate the RESPONSE (auto-disable
#   account, auto-quarantine). Exam: SIEM = detect/correlate,
#   SOAR = automate response.
#
# Antivirus / anti-malware — signature + heuristic + behavior engines
#   on endpoints; quarantine on detection. Modern evolution = EDR.
#
# DLP — Data Loss Prevention (also under 4.5): detects/blocks
#   sensitive data (PII, PHI, card numbers) leaving via email, web,
#   USB, cloud sync. Pattern-matches content (regex for SSNs/PANs),
#   fingerprints documents, enforces policy (block/encrypt/alert).
#
# SNMP traps — Simple Network Management Protocol:
#   Manager polls agents (GET) on UDP 161; agents PUSH unsolicited
#   TRAP messages to the manager on UDP 162 when something happens
#   (interface down, threshold crossed). Trap = asynchronous,
#   device-initiated alert. MIB = Management Information Base (the
#   variable tree, addressed by OIDs).
#   SNMPv1/v2c: community strings in CLEARTEXT → insecure.
#   SNMPv3: authentication + encryption (authPriv) → required answer.
#
# NetFlow — flow telemetry from routers/switches (Cisco; IPFIX is the
#   IETF standard, sFlow is sampled): records WHO talked to WHOM
#   (src/dst IP, ports, protocol, bytes, packets, timestamps) but NOT
#   packet payloads. Use: baseline traffic, spot exfiltration,
#   beaconing, DDoS, top talkers.
#   Exam cue: "identify large transfer to unknown external host
#   without capturing packets" → NetFlow.
#   Full payloads needed → packet capture (Wireshark/tcpdump) instead.
#
# Vulnerability scanners — Nessus, OpenVAS, Qualys: enumerate known
#   CVEs and misconfigurations; credentialed scans see more (installed
#   packages, registry) with fewer false positives than
#   non-credentialed. Scanner ASSESSES, it does not exploit
#   (exploitation = penetration test).
```

## Firewall (4.5)

```
# Rules / access lists (ACLs — Access Control Lists):
#   Ordered top-down, FIRST match wins, implicit deny at the end.
#   Tuple: source IP, destination IP, protocol, source port, dest port,
#   action (permit/deny). Most-specific rules go first.
#
#   Example rule set (edge firewall):
#   10  permit tcp any → 203.0.113.10 port 443     # public web
#   20  permit tcp 198.51.100.0/24 → any port 22   # admin SSH from mgmt
#   30  deny   ip  any → any  log                  # explicit deny + log
#
# Ports/protocols — filtering by service port (L4). Know the classics:
#   22 SSH, 23 telnet, 25 SMTP, 53 DNS, 80 HTTP, 110 POP3, 143 IMAP,
#   443 HTTPS, 445 SMB, 389 LDAP, 636 LDAPS, 3389 RDP.
#
# Screened subnet (the SY0-701 term for DMZ — demilitarized zone):
#   segment between two firewalls (or two interfaces of one) hosting
#   public-facing services (web, mail, DNS). Internet ↔ screened
#   subnet allowed; screened subnet → internal LAN tightly restricted.
#   Exam cue: "host a public web server without exposing the internal
#   network" → screened subnet.
#
# Firewall generations (recurring distractor set):
#   Packet filter (stateless)  — per-packet ACL only
#   Stateful                   — tracks connections, allows replies
#   NGFW (next-generation)     — application-aware (L7), user identity,
#                                integrated IPS, TLS inspection
#   WAF (web application FW)   — protects HTTP apps from SQLi/XSS
#                                (OWASP Top 10); NOT a network firewall
#   UTM (unified threat mgmt)  — all-in-one SMB box (FW+AV+filter+IPS)
```

## IDS / IPS (4.5)

```
# IDS — Intrusion Detection System: DETECTS and alerts (passive,
#   usually off a SPAN/mirror port or tap; out-of-band).
# IPS — Intrusion Prevention System: detects and BLOCKS (inline,
#   in the traffic path; can drop packets/reset sessions).
# NIDS/NIPS = network-based; HIDS/HIPS = host-based.
#
# Detection methods:
#   Signature-based — matches known attack patterns; low false
#     positives; CANNOT catch zero-days/novel attacks; needs updates
#   Trend/anomaly/behavior-based — baselines "normal," alerts on
#     deviation; CAN catch zero-days; higher false-positive rate;
#     needs a training/learning period
#
# Error vocabulary (exam staple):
#   False positive — alert fired, no real attack (wastes analyst time)
#   False negative — real attack, no alert (the dangerous one)
#   True positive/negative — correct outcomes
#
# Exam cues:
#   "detect but do not disrupt traffic"           → IDS (passive)
#   "actively block attacks inline"               → IPS
#   "failed to detect a brand-new exploit"        → signature-based limit
#   "alerts on deviation from baseline"           → anomaly/trend-based
#   "risk of inline device failing"               → fail-open vs fail-closed
```

## Web Filter (4.5)

```
# Agent-based — filtering client installed on each endpoint; policy
#   follows the device off-network (roaming laptops). Cue: "enforce
#   web policy for remote/traveling users" → agent-based filter.
#
# Centralized proxy — all traffic funneled through a forward proxy
#   (explicit config/PAC file or transparent). Inspects, caches,
#   authenticates users, logs, applies policy at one choke point.
#
# URL scanning — checks each requested Universal Resource Locator
#   against threat intel / known-bad lists at request time.
# Content categorization — sites classified into categories (gambling,
#   social media, adult, malware); policy allows/blocks by category.
# Block rules — explicit allow/deny lists per user/group/category/time.
# Reputation — score for a site/domain/IP based on age, history,
#   hosting, threat feeds; low-reputation = block even if uncategorized.
#   Cue: "block newly registered / low-trust domains" → reputation.
```

## Operating System Security (4.5)

```
# Group Policy (Windows) — GPOs (Group Policy Objects) applied through
#   Active Directory to enforce settings at scale: password policy,
#   lockout, disable USB storage, software restriction/AppLocker,
#   audit policy. Order: Local → Site → Domain → OU (last writer wins).
#   Cue: "centrally enforce settings on all domain Windows machines"
#   → Group Policy.
#
# SELinux (Security-Enhanced Linux) — MANDATORY access control (MAC)
#   in the Linux kernel: labels on processes/files; policy decides,
#   not the file owner (contrast: standard Linux permissions = DAC,
#   discretionary). Modes: enforcing / permissive (log only) /
#   disabled. AppArmor = same idea, path-based (Ubuntu/SUSE).
#   Cue: "confine a compromised daemon on Linux" / "kernel-level MAC"
#   → SELinux.
```

## Implementation of Secure Protocols (4.5)

```
# Three decisions the objective names:
#   Protocol selection  — pick the secure protocol (SSH not telnet)
#   Port selection      — use the secure service port (443 not 80);
#                         nonstandard ports = obscurity, not security
#   Transport method    — how encryption is applied: dedicated secure
#                         protocol (SSH), TLS-wrapped (HTTPS, LDAPS,
#                         implicit FTPS), STARTTLS upgrade in-band
#                         (SMTP 587, LDAP 389→TLS), or tunneled (VPN /
#                         IPsec when the app protocol can't be secured)
```

| Insecure protocol | Port | Secure replacement | Port | Notes |
|---|---|---|---|---|
| Telnet | 23/tcp | SSH (Secure Shell) | 22/tcp | Also replaces rsh/rlogin; SCP/SFTP ride SSH |
| HTTP | 80/tcp | HTTPS (HTTP over TLS) | 443/tcp | TLS 1.2/1.3 only; HSTS to force it |
| FTP (File Transfer Protocol) | 21/tcp (+20) | SFTP (SSH File Transfer) | 22/tcp | Single port, firewall-friendly |
| FTP | 21/tcp | FTPS (FTP over TLS) | 990/tcp implicit, 21 explicit | Distinct from SFTP — TLS, not SSH |
| SNMPv1/v2c | 161/162 udp | SNMPv3 (authPriv) | 161/162 udp | Same ports; v3 adds auth + encryption |
| DNS (cleartext) | 53 udp/tcp | DoT (DNS over TLS) | 853/tcp | Dedicated port |
| DNS (cleartext) | 53 udp/tcp | DoH (DNS over HTTPS) | 443/tcp | Blends into HTTPS traffic |
| LDAP | 389/tcp | LDAPS (LDAP over TLS/SSL) | 636/tcp | Or STARTTLS on 389 |
| POP3 (Post Office Protocol) | 110/tcp | POP3S | 995/tcp | Implicit TLS |
| IMAP (Internet Message Access Protocol) | 143/tcp | IMAPS | 993/tcp | Implicit TLS |
| SMTP relay (cleartext) | 25/tcp | SMTP + STARTTLS (submission) | 587/tcp | 465 = implicit TLS submission |
| NTP (Network Time Protocol) | 123/udp | NTS (Network Time Security) | 123 + 4460/tcp KE | RFC 8915 |
| HTTP/Telnet device mgmt | — | SSH/HTTPS mgmt plane | 22/443 | Disable insecure mgmt interfaces |
| TFTP (Trivial FTP) | 69/udp | SFTP/SCP | 22/tcp | TFTP has no auth at all |

```
# Exam cue: given a packet capture or port number, map insecure → its
# secure sibling. "Credentials visible in capture on port 23" →
# replace telnet with SSH. "Encrypt directory-service queries" → LDAPS
# 636. "Time sync integrity" → NTS.
```

## DNS Filtering (4.5)

```
# Resolver refuses (or sinkholes) lookups for known-malicious domains:
#   phishing, malware C2 (command and control), newly registered
#   domains. Enforced at internal resolver, protective DNS service
#   (e.g. Cisco Umbrella, Quad9), or firewall.
# Sinkhole — resolver answers with a controlled IP so you can log/
#   identify infected clients that attempt C2 callbacks.
# Cue: "block users from resolving phishing domains" / "stop malware
# contacting C2 by name" → DNS filtering; "find infected hosts by
# their lookups" → DNS sinkhole + logs.
```

## Email Security (4.5)

```
# SPF — Sender Policy Framework: DNS TXT record listing which mail
#   servers may send FOR your domain. Receiver checks the connecting
#   server's IP against the record (envelope MAIL FROM domain).
#
#   example.com.  TXT  "v=spf1 ip4:203.0.113.0/24 include:_spf.google.com -all"
#     -all = hard fail (reject others)   ~all = soft fail   ?all = neutral
#
# DKIM — DomainKeys Identified Mail: sending server SIGNS the message
#   (headers+body hash) with a private key; receiver fetches the
#   public key from DNS to verify integrity + domain authenticity.
#
#   selector1._domainkey.example.com.  TXT  "v=DKIM1; k=rsa; p=MIIBIjANBg..."
#     (message carries a DKIM-Signature: header naming the selector)
#
# DMARC — Domain-based Message Authentication, Reporting, and
#   Conformance: policy that tells receivers what to DO when SPF/DKIM
#   fail, requires ALIGNMENT (the visible From: domain must match the
#   SPF or DKIM domain), and provides aggregate reports.
#
#   _dmarc.example.com.  TXT  "v=DMARC1; p=reject; rua=mailto:dmarc@example.com; pct=100"
#     p=none (monitor only) → p=quarantine (spam-folder) → p=reject
#
# How the three interact:
#   1. SPF: is the sending IP authorized for the envelope domain?
#   2. DKIM: does the signature verify against the DNS public key?
#   3. DMARC: does at least ONE of SPF/DKIM pass AND align with the
#      From: header domain? If not → apply p= policy; send rua reports.
#   SPF alone breaks on forwarding; DKIM alone doesn't bind the
#   visible From:; DMARC ties both to what the user actually sees.
#
# Email gateway — inbound/outbound mail relay doing spam filtering,
#   AV, sandboxing attachments, URL rewriting, DLP, and enforcing
#   SPF/DKIM/DMARC. Sits in front of the mail server (often the MX).
#
# Exam cues:
#   "which servers may send for the domain"     → SPF
#   "digitally sign outgoing mail via DNS key"  → DKIM
#   "policy for failures + reporting"           → DMARC
#   "stop spoofed mail claiming to be your CEO's domain" → DMARC p=reject
#   "scan/quarantine attachments before delivery" → email (secure) gateway
```

## File Integrity Monitoring (4.5)

```
# FIM — records cryptographic hashes/attributes of critical files
#   (system binaries, configs, web content) and alerts on ANY change.
#   Tools: Tripwire, OSSEC/Wazuh, Windows SFC (System File Checker) at
#   a basic level. PCI DSS explicitly requires FIM.
# Cue: "detect unauthorized changes to system files / web pages" /
#   "alert when a config file is modified" → FIM.
```

## DLP, NAC, EDR/XDR, User Behavior Analytics (4.5)

```
# DLP — Data Loss Prevention (enterprise deployment view):
#   Endpoint DLP  — blocks USB copy, print, clipboard of tagged data
#   Network DLP   — inspects email/web/FTP egress for patterns
#   Cloud DLP     — API-based scanning of SaaS/cloud storage
#   Actions: block, quarantine, encrypt, alert, coach the user
#   Cue: "employee emailed a spreadsheet of SSNs" / "block card
#   numbers leaving via email" → DLP
#
# NAC — Network Access Control (often 802.1X port-based, EAP +
#   RADIUS): authenticates the DEVICE/user before granting LAN/WLAN
#   access and checks POSTURE (patched? AV running? disk encrypted?).
#   Non-compliant → quarantine/remediation VLAN or guest VLAN.
#   Agent (persistent/dissolvable) or agentless posture checks.
#   Cue: "verify health of device before it joins the network" /
#   "unmanaged laptop plugged into a conference-room jack" → NAC
#
# EDR — Endpoint Detection and Response: continuous endpoint telemetry
#   (process trees, file/registry/network events), behavioral
#   detection, threat hunting, and RESPONSE actions (kill process,
#   isolate host, roll back). Beyond signature AV.
# XDR — Extended Detection and Response: EDR expanded to correlate
#   endpoint + network + email + identity + cloud telemetry in one
#   platform.
#   Cue: "detect fileless malware / living-off-the-land on laptops"
#   → EDR; "correlate endpoint, email, and cloud alerts in one
#   console" → XDR
#
# User behavior analytics (UBA/UEBA — User and Entity Behavior
#   Analytics): machine-learning baselines of per-user/entity normal
#   (login times, geos, volumes, apps) and flags deviations —
#   the insider-threat and compromised-credential detector.
#   Cue: "account downloads 10 GB at 3 a.m. from a new country" /
#   "detect insider threat with valid credentials" → UBA/UEBA
```

## Exam Cue Reflex Table (4.4 + 4.5)

| Question keyword/scenario | Reflex answer |
|---|---|
| Aggregate + correlate logs from many sources | SIEM |
| Automate the response playbook to alerts | SOAR |
| Too many false positives, alert fatigue | Alert tuning |
| Isolate infected host but keep evidence | Quarantine (EDR isolation) |
| Machine-readable compliance/config checking standard | SCAP |
| Hardening checklist to scan against | CIS Benchmark / STIG |
| Device pushes unsolicited event to manager (UDP 162) | SNMP trap |
| Who-talked-to-whom traffic metadata, no payload | NetFlow |
| Full packet payload needed | Packet capture |
| Identify known CVEs on hosts | Vulnerability scanner (credentialed) |
| Public server, isolated from internal LAN | Screened subnet |
| First-match, implicit deny rule list | Firewall ACL |
| Detect only, out-of-band, SPAN port | IDS |
| Block inline | IPS |
| Missed a zero-day | Signature-based detection limitation |
| Web policy that follows roaming laptops | Agent-based web filter |
| Block by site category or low reputation | Web filter (categorization/reputation) |
| Centrally enforce Windows settings | Group Policy (GPO) |
| Kernel-enforced labels on Linux processes | SELinux (MAC) |
| Cleartext credentials on port 23/21/80 | Replace with SSH/SFTP/HTTPS |
| Encrypt DNS inside HTTPS | DoH (DoT = port 853) |
| Stop users resolving malicious domains | DNS filtering |
| Authorized sending IPs in DNS TXT | SPF |
| Cryptographic signature on outbound mail | DKIM |
| Failure policy + reporting + alignment | DMARC |
| Alert on modified system/config files | File integrity monitoring |
| Sensitive data leaving the org | DLP |
| Posture check before network admission | NAC (802.1X) |
| Endpoint telemetry + response actions | EDR (multi-domain → XDR) |
| Valid account acting abnormally | User behavior analytics (UEBA) |

## See Also

security-plus-sy0-701, secplus-hardening, secplus-architecture, secplus-incident-response, secplus-attack-indicators, secplus-asset-vuln-management, secplus-iam, siem, ids-ips, waf, email-gateway, network-access-control, dot1x, endpoint-security, log-analysis, network-defense, vulnerability-scanning, cis-benchmarks, firewall-design, tls

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-126 Rev. 3 — SCAP 1.3](https://csrc.nist.gov/publications/detail/sp/800-126/rev-3/final)
- [NIST SP 800-92 — Guide to Computer Security Log Management](https://csrc.nist.gov/publications/detail/sp/800-92/final)
- [NIST SP 800-94 — Guide to Intrusion Detection and Prevention Systems](https://csrc.nist.gov/publications/detail/sp/800-94/final)
- [RFC 7208 — Sender Policy Framework (SPF)](https://www.rfc-editor.org/rfc/rfc7208)
- [RFC 6376 — DomainKeys Identified Mail (DKIM)](https://www.rfc-editor.org/rfc/rfc6376)
- [RFC 7489 — DMARC](https://www.rfc-editor.org/rfc/rfc7489)
- [RFC 7858 — DNS over TLS](https://www.rfc-editor.org/rfc/rfc7858) / [RFC 8484 — DNS over HTTPS](https://www.rfc-editor.org/rfc/rfc8484)
- [RFC 8915 — Network Time Security](https://www.rfc-editor.org/rfc/rfc8915)
- [RFC 3411–3418 — SNMPv3](https://www.rfc-editor.org/rfc/rfc3411)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks)
- [SELinux Project documentation](https://selinuxproject.org/)
- [IEEE 802.1X — Port-Based Network Access Control](https://standards.ieee.org/ieee/802.1X/7345/)
