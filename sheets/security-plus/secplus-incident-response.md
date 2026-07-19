# Incident Response, Automation & Investigation Data (Security+ SY0-701 Objectives 4.7, 4.8, 4.9)

> Automation and orchestration (SOAR — Security Orchestration, Automation, and Response) use cases and trade-offs, the seven-phase incident response lifecycle mapped to NIST SP 800-61, digital forensics from legal hold to reporting, and which log or data source answers which investigative question.

## Objective Map

```
# 4.7 — Explain the importance of automation and orchestration related to
#       secure operations (use cases, benefits, other considerations)
# 4.8 — Explain appropriate incident response activities
#       (process, training, testing, root cause analysis, threat hunting,
#        digital forensics)
# 4.9 — Given a scenario, use data sources to support an investigation
#       (log data, other data sources)
#
# Weighting: Domain 4 (Security Operations) is 28% of SY0-701 — the
# single heaviest domain. 4.8 phase-ordering questions and 4.9
# "which log tells you X" questions are near-guaranteed.
```

## 4.7 Automation and Scripting — Use Cases

```
# User provisioning
# - Script/workflow creates accounts, group memberships, mailbox, and
#   entitlements on hire; disables them all on termination
# - Kills two classic audit findings: orphaned accounts + inconsistent
#   entitlements. Exam cue: "employee's access removed the moment HR
#   marks them terminated" → automated user (de)provisioning
# Resource provisioning
# - Infrastructure as Code (IaC — Terraform, CloudFormation, Ansible)
#   builds servers/VMs/containers from a template every time
# - Exam cue: "identical, repeatable server builds" → IaC/automation
# Guard rails
# - Automated policy checks that prevent or auto-revert unsafe actions
#   (e.g. block creating a world-readable storage bucket, revert a
#   security group opened to 0.0.0.0/0)
# - Exam cue: "prevent admins from making risky changes without slowing
#   them down" → guard rails
# Security groups
# - Scripted management of network access rules (cloud firewall
#   objects); automation keeps rule sets consistent across environments
# Ticket creation / escalation
# - SIEM (Security Information and Event Management) alert fires →
#   ticket auto-created, severity set, on-call paged if unacknowledged
# - Exam cue: "analysts spend hours manually opening tickets" →
#   automated ticket creation; "no one responded for 4 hours" →
#   automated escalation
# Enabling/disabling services and access
# - Playbook disables a compromised account, isolates a host, blocks an
#   IP at the firewall — in seconds, 24/7, no human in the loop
# Continuous integration and testing (CI — Continuous Integration)
# - Every commit triggers build + unit tests + SAST (Static Application
#   Security Testing) + dependency/secret scanning before merge
# - Exam cue: "catch vulnerable code before it reaches production" →
#   automated security testing in the CI pipeline
# Integrations and APIs (Application Programming Interfaces)
# - APIs are the glue: SOAR calls the EDR (Endpoint Detection and
#   Response) API to isolate a host, the firewall API to block an IP,
#   the IdP (Identity Provider) API to force a password reset
# - No API = no automation; "integrations" questions want the answer
#   that connects existing tools rather than buying a new one
```

## 4.7 Benefits of Automation

```
# Efficiency / time saving      — machines do repetitive toil; analysts
#                                 do analysis
# Enforcing baselines           — config drift auto-detected and
#                                 auto-remediated; every host matches
#                                 the secure baseline continuously
# Standard infrastructure
#   configurations              — templates produce identical builds;
#                                 no snowflake servers
# Scaling in a secure manner    — 10 or 10,000 instances get the same
#                                 hardened config; security keeps pace
#                                 with growth
# Employee retention            — removing boring toil reduces analyst
#                                 burnout and turnover (yes, this is a
#                                 listed exam benefit — don't dismiss it
#                                 as a distractor)
# Reaction time                 — machine speed: contain in seconds,
#                                 not after the analyst's coffee
# Workforce multiplier          — a small team achieves the output of a
#                                 large one; exam cue: "small security
#                                 team, growing environment" →
#                                 automation as workforce multiplier
```

## 4.7 Other Considerations (Automation Drawbacks)

```
# Complexity            — automated workflows are systems too; they need
#                         design, testing, and documentation; a buggy
#                         playbook does damage at machine speed
# Cost                  — SOAR licensing, engineering time to build and
#                         maintain playbooks; upfront investment before
#                         payoff
# Single point of failure — if everything flows through one automation
#                         server/orchestrator and it dies, provisioning,
#                         containment, and ticketing all stop
# Technical debt        — quick-hack scripts accumulate; author leaves;
#                         nobody understands the 400-line bash script
#                         that provisions accounts
# Ongoing supportability — automation must be maintained as APIs change,
#                         tools are replaced, and staff turn over;
#                         "the script broke when the vendor updated
#                         their API" → supportability consideration
```

## SOAR vs SIEM (Context for 4.7)

| Aspect | SIEM (Security Information and Event Management) | SOAR (Security Orchestration, Automation, and Response) |
|---|---|---|
| Primary job | Collect, correlate, alert on log/event data | Act on alerts: orchestrate tools, run playbooks |
| Output | Alerts, dashboards, reports | Automated response actions, case management |
| Human role | Analyst investigates alerts | Analyst approves/reviews automated actions |
| Exam cue | "correlate logs from many sources" | "automatically respond to alerts using playbooks" |
| Playbook | n/a | Documented, often automated, step-by-step response to a specific incident type (e.g. phishing playbook) |
| Runbook | n/a | The (semi-)automated procedure steps a playbook executes — playbook = the plan, runbook = the how |

## 4.8 Incident Response Process (NIST SP 800-61 Aligned)

```
# The SY0-701 seven phases, in ORDER — ordering questions are guaranteed:
#   1. Preparation
#   2. Detection
#   3. Analysis
#   4. Containment
#   5. Eradication
#   6. Recovery
#   7. Lessons learned
#
# Mnemonic: "People Don't Always Contain Every Ransomware Locally"
#   (Preparation, Detection, Analysis, Containment, Eradication,
#    Recovery, Lessons learned)
# Older mnemonic PICERL (Preparation, Identification, Containment,
#   Eradication, Recovery, Lessons learned) — SY0-701 splits
#   Identification into Detection + Analysis; NIST SP 800-61 groups
#   them as "Detection & Analysis" and groups Containment/Eradication/
#   Recovery into one phase. Same lifecycle, different granularity.
#
# NIST SP 800-61 (Computer Security Incident Handling Guide) mapping:
#   NIST 1. Preparation                      = Preparation
#   NIST 2. Detection & Analysis             = Detection + Analysis
#   NIST 3. Containment, Eradication & Recovery
#                                            = Containment + Eradication
#                                              + Recovery
#   NIST 4. Post-Incident Activity           = Lessons learned
```

```
# 1. PREPARATION — before anything happens
#    - Write the IR plan, define the CSIRT (Computer Security Incident
#      Response Team) roles, build the call tree / out-of-band comms
#    - Stage the jump bag: forensic laptop, write blockers, blank
#      drives, chain-of-custody forms, contact lists
#    - Train staff, run exercises, deploy logging/EDR so detection is
#      even possible
#    - Exam cue: "creating the plan / assembling the team / buying
#      forensic kits" → preparation
# 2. DETECTION — realizing something happened
#    - SIEM alert, EDR alert, IDS (Intrusion Detection System)
#      signature, user report, third-party notification
#    - Exam cue: "an alert fires / help desk gets a report of odd
#      behavior" → detection
# 3. ANALYSIS — is it real, how bad, how wide?
#    - Triage: true positive or false positive? Scope: which hosts,
#      accounts, data? Severity/priority assignment
#    - Exam cue: "determine whether the alert is a real incident and
#      its scope/impact" → analysis
# 4. CONTAINMENT — stop the bleeding
#    - Isolate/quarantine hosts (network isolation preserves RAM
#      evidence — usually preferred over pulling power), disable
#      accounts, block C2 (command and control) IPs/domains, segment
#    - Short-term (isolate now) vs long-term (rebuild behind the scenes)
#    - Exam cue: "prevent the malware from SPREADING / limit the
#      damage" → containment
# 5. ERADICATION — remove the cause
#    - Delete malware, remove persistence (scheduled tasks, registry
#      run keys, rogue accounts), patch the exploited vulnerability,
#      reimage compromised hosts
#    - Exam cue: "REMOVE the malware / eliminate the root cause /
#      reimage" → eradication
# 6. RECOVERY — back to normal
#    - Restore from known-good backups, rebuild, re-enable services,
#      return systems to production, heightened monitoring for
#      re-infection
#    - Exam cue: "restore systems to normal operation / bring services
#      back online" → recovery
# 7. LESSONS LEARNED — after-action
#    - Post-incident review meeting (blameless), after-action report,
#      update the IR plan/playbooks/controls, feed findings back into
#      Preparation
#    - Exam cue: "meeting after the incident to improve the process /
#      update documentation" → lessons learned
```

## 4.8 Training and Testing

```
# Training — everyone with an IR role knows the plan BEFORE the
#   incident: responders drill tools/procedures, users learn how to
#   report, executives learn their decision points. Untrained plan =
#   shelfware.
```

| Exercise type | What happens | Production impact | Cost | Exam cue |
|---|---|---|---|---|
| Tabletop exercise | Discussion-based walkthrough of a scenario around a table; talk through the plan, no systems touched | None | Low | "discussion-based", "least impact/cost", "walk through the plan verbally" |
| Simulation | Hands-on rehearsal of a realistic scenario (e.g. simulated phishing, mock ransomware, red/purple team drill); actions actually performed in a test or controlled prod setting | Possible | Higher | "realistic hands-on test of the response", "test the team's actual actions" |

```
# Related (from 5.x but appears as distractors here):
#   Failover test — actually swing to the DR (Disaster Recovery) site
#   Parallel processing test — run DR systems alongside prod
# Tabletop < simulation < failover in both realism and risk.
```

## 4.8 Root Cause Analysis and Threat Hunting

```
# Root cause analysis (RCA)
# - Answers "WHY did this happen?" — not just the malware, but the
#   unpatched server, the missing MFA (Multi-Factor Authentication),
#   the process gap that let it in
# - Techniques: 5 Whys (ask "why" iteratively), fishbone/Ishikawa
#   diagram
# - Fix the symptom only → the incident recurs. RCA output feeds
#   lessons learned and eradication (patch the actual entry point).
# - Exam cue: "determine the underlying reason the incident occurred"
#   → root cause analysis
#
# Threat hunting
# - PROACTIVE, hypothesis-driven search for adversaries already inside
#   who evaded automated detection — you hunt WITHOUT an alert
# - Inputs: threat intelligence, IoCs (Indicators of Compromise),
#   TTPs (Tactics, Techniques, and Procedures), MITRE ATT&CK matrix
# - Hypothesis example: "If APT X targets our sector using scheduled-
#   task persistence, we should query all endpoints for anomalous
#   scheduled tasks."
# - Exam cue: "proactively search for threats that bypassed existing
#   controls / assume breach and go looking" → threat hunting
#   (vs detection = reactive, alert-driven)
```

## 4.8 Digital Forensics

```
# Legal hold
# - Formal notice (usually from legal counsel) to PRESERVE all data
#   relevant to actual/anticipated litigation or investigation
# - Suspends normal retention/deletion schedules — auto-purge of
#   mailboxes/logs must stop for custodians under hold
# - Exam cue: "lawsuit anticipated, ensure emails are not deleted" →
#   legal hold
#
# Chain of custody
# - Documented, unbroken record of WHO collected/handled evidence,
#   WHAT it is, WHEN and WHERE each transfer happened, and WHY
# - Every handoff signed; evidence sealed/tagged; gaps = evidence can
#   be challenged/inadmissible in court
# - Exam cue: "prove evidence was not tampered with / track who handled
#   it / admissible in court" → chain of custody
#
# Acquisition — collecting the evidence
# - Order of volatility (RFC 3227): capture the MOST volatile FIRST:
#     1. CPU registers, CPU cache
#     2. RAM (memory), routing table, ARP cache, process table,
#        kernel statistics
#     3. Temporary file systems / swap / pagefile
#     4. Disk (non-volatile storage)
#     5. Remote logging and monitoring data (on other systems)
#     6. Physical configuration, network topology
#     7. Archival media (backups, tapes)
#   Mnemonic: registers → RAM → swap → disk → remote logs → config →
#   backups. Exam cue: "which do you collect FIRST" → RAM before disk,
#   always; "collect LAST" → archival/backup media.
# - Image, don't touch: bit-for-bit forensic image with a write blocker;
#   hash original and copy (SHA-256) — matching hashes prove integrity;
#   analyze the COPY, never the original
# - Live acquisition for RAM/encrypted volumes (powering off destroys
#   RAM and may lock encrypted disks); network isolation > power pull
#   when malware is memory-resident
#
# Preservation
# - Protect evidence integrity after collection: write blockers, sealed
#   evidence bags, restricted-access storage, hashes recorded, copies
#   only for analysis
#
# Reporting
# - Document everything: what was found, how it was collected, tools
#   used, hash values, timeline of events; written for a non-technical
#   audience (lawyers, executives, court); repeatable by another
#   examiner
#
# E-discovery (electronic discovery)
# - Legal process of identifying, collecting, and PRODUCING
#   electronically stored information (ESI) for litigation — emails,
#   documents, chat logs, databases
# - Works with legal hold: hold preserves, e-discovery produces
# - Exam cue: "produce electronic records requested in a lawsuit" →
#   e-discovery
```

## 4.9 Log Data — Which Log Answers Which Question

```
# Firewall logs      — allowed/denied connections: src/dst IP, port,
#                      protocol, rule that matched. Traffic METADATA,
#                      not payload.
# Application logs   — what the app did: logins, errors, transactions,
#                      SQL errors, stack traces (web server access logs
#                      show full request URLs — injection attempts live
#                      here)
# Endpoint logs      — host-level activity from EDR/antivirus/host
#                      agents: process launches, file writes, USB
#                      events, malware detections
# OS-specific security logs — Windows Security Event Log (logon 4624,
#                      failed logon 4625, account created 4720, log
#                      cleared 1102), Linux /var/log/auth.log or
#                      /var/log/secure, sudo logs, journald
# IPS/IDS logs       — signature/anomaly matches on network traffic:
#                      exploit attempts, port scans, known-malware C2
#                      patterns; IPS (Intrusion Prevention System) also
#                      logs what it BLOCKED
# Network logs       — flow data (NetFlow/sFlow/IPFIX), switch/router
#                      logs, DHCP leases, DNS query logs, VPN
#                      (Virtual Private Network) connection logs
# Metadata           — data about data: email headers (true sender
#                      path), file metadata (author, timestamps), image
#                      EXIF (GPS coordinates), mobile call records —
#                      who/when/where without content
```

| Investigative question | Best data source |
|---|---|
| "Did any host connect to this known-bad IP?" | Firewall logs / network flow logs |
| "Was the connection attempt blocked or allowed?" | Firewall logs (action field) |
| "Who failed to log in 500 times at 3 AM?" | OS security logs (Windows Event ID 4625 / auth.log) |
| "What process spawned this malware / what did it write to disk?" | Endpoint (EDR) logs |
| "Was a known exploit signature seen on the wire?" | IDS/IPS logs |
| "What SQL injection strings hit our web app?" | Application / web server access logs |
| "Which internal host asked DNS for the C2 domain?" | Network logs (DNS query logs) |
| "Who really sent this spoofed email?" | Metadata (email headers) |
| "Where was this photo taken?" | Metadata (EXIF GPS data) |
| "Exactly what data left the network in that session?" | Packet capture (full content) |
| "Was the server vulnerable before the breach?" | Vulnerability scan (historical report) |
| "Did someone clear their tracks on the DC?" | OS security logs (Windows Event ID 1102 — audit log cleared) |

## 4.9 Other Data Sources

```
# Vulnerability scans
# - Point-in-time list of known weaknesses per host; during an
#   investigation, answers "was the exploited CVE (Common
#   Vulnerabilities and Exposures) present and unpatched?"
# Automated reports
# - Scheduled SIEM/tool summaries — trend data, top talkers, recurring
#   alerts; good for spotting when behavior CHANGED
# Dashboards
# - Real-time visual status (SIEM/SOC — Security Operations Center —
#   wallboards); exam cue: "at-a-glance, real-time view for the SOC /
#   executives" → dashboard
# Packet captures (pcap)
# - FULL packet content via tcpdump/Wireshark — the only source that
#   shows the actual payload/exfiltrated bytes (if unencrypted)
# - Trade-off: huge storage, limited retention, TLS blinds you; flow
#   logs are the lightweight metadata alternative
# - Exam cue: "reconstruct exactly what data was transmitted" →
#   packet capture
```

## Exam Cues — Keyword → Answer

| Question wording contains... | Answer |
|---|---|
| "prove evidence integrity / admissible in court / who handled it" | Chain of custody |
| "preserve data for pending litigation" | Legal hold |
| "produce electronic records for a lawsuit" | E-discovery |
| "collect first / most volatile" | RAM (order of volatility) |
| "verify the forensic image matches the original" | Hashing (SHA-256) |
| "which phase removes malware / reimages" | Eradication |
| "which phase stops the spread / isolates the host" | Containment |
| "which phase restores systems to production" | Recovery |
| "meeting afterward to improve the process" | Lessons learned |
| "discussion-based, low-cost IR plan test" | Tabletop exercise |
| "hands-on realistic IR rehearsal" | Simulation |
| "find the underlying reason it happened" | Root cause analysis |
| "proactively search for undetected adversaries" | Threat hunting |
| "automatically execute response steps from a playbook" | SOAR |
| "correlate logs from many sources into alerts" | SIEM |
| "accounts created/disabled automatically on hire/term" | Automated user provisioning |
| "prevent risky configurations automatically" | Guard rails |
| "small team handles large environment" | Workforce multiplier (automation benefit) |
| "the automation script nobody maintains broke" | Technical debt / ongoing supportability |
| "all automation runs through one server" | Single point of failure |
| "see the actual payload/content of traffic" | Packet capture |
| "real-time at-a-glance SOC status" | Dashboard |
| "was the missing patch present before the breach" | Vulnerability scan |

## See Also

security-plus-sy0-701, secplus-monitoring-defense, secplus-governance-risk, incident-response, forensics, siem, ids-ips, log-analysis, threat-hunting, mitre-attack, security-operations, bcp-drp, nist

## References

- CompTIA Security+ SY0-701 Exam Objectives — https://www.comptia.org/certifications/security
- NIST SP 800-61 Rev. 2, Computer Security Incident Handling Guide — https://csrc.nist.gov/pubs/sp/800/61/r2/final
- NIST SP 800-86, Guide to Integrating Forensic Techniques into Incident Response — https://csrc.nist.gov/pubs/sp/800/86/final
- RFC 3227, Guidelines for Evidence Collection and Archiving — https://www.rfc-editor.org/rfc/rfc3227
- MITRE ATT&CK — https://attack.mitre.org/
- NIST SP 800-83, Guide to Malware Incident Prevention and Handling — https://csrc.nist.gov/pubs/sp/800/83/r1/final
- CISA Incident Response Playbooks — https://www.cisa.gov/resources-tools/resources/federal-government-cybersecurity-incident-and-vulnerability-response-playbooks
