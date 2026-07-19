# Security Controls, Fundamentals & Change Management (Security+ SY0-701 Objectives 1.1–1.3)

> Security control categories and types, CIA/AAA/non-repudiation, gap analysis, Zero Trust planes, physical security, deception technology, and change management — the foundation Domain 1 questions are built from.

## 1.1 Security Control Categories

```
# A CATEGORY answers "WHO/WHAT implements the control?"
# (contrast with TYPE, which answers "WHAT EFFECT does it have?")

# Technical (logical) — implemented by SYSTEMS/technology
#   firewalls, ACLs (access control lists), antivirus, IDS/IPS
#   (intrusion detection/prevention systems), encryption, MFA
#   (multi-factor authentication), DLP (data loss prevention)
# Managerial (administrative) — implemented by DECISIONS/oversight
#   policies, risk assessments, security planning, vulnerability
#   management program, vendor assessment, SOPs at the policy level
# Operational — implemented by PEOPLE doing day-to-day procedures
#   security awareness training, guard patrols, log review,
#   incident response exercises, account provisioning procedures,
#   change management execution, backup verification
# Physical — implemented in the REAL WORLD to limit physical access
#   fences, locks, bollards, badges, lighting, guards-as-barrier,
#   access control vestibules, sensors, cameras

# Exam discrimination trick:
#   done by a computer         → technical
#   written/decided by mgmt    → managerial
#   performed by a human daily → operational
#   touch it with your hands   → physical
# A guard is BOTH physical (body blocking a door) and operational
#   (patrol procedure) — the question's wording picks the lens
```

## 1.1 Security Control Types

```
# A TYPE answers "WHAT EFFECT does the control have on an event?"

# Preventive   — stops the event BEFORE it happens
#                (firewall rule, door lock, least privilege)
# Deterrent    — discourages the ATTEMPT; doesn't physically stop it
#                (warning signs, visible cameras, login banners)
# Detective    — identifies/records the event DURING or AFTER
#                (IDS, log review, CCTV footage review, audits)
# Corrective   — fixes/restores AFTER the event
#                (restore from backup, patch after exploit,
#                 re-image infected host, fire suppression)
# Compensating — substitute when the primary control is not feasible
#                (network segmentation for an unpatchable legacy
#                 system; TOTP when smart cards can't be deployed)
# Directive    — mandates/communicates required behavior
#                (AUP — acceptable use policy, "authorized personnel
#                 only" signage, SOPs, policy requiring encryption)

# Deterrent vs preventive: deterrent works on the attacker's MIND
#   (they could still proceed); preventive works on the ACTION
#   (they cannot proceed). A LOCK prevents; a SIGN deters.
# Detective vs corrective: detective tells you it happened;
#   corrective undoes the damage.
# Directive vs managerial: directive is a TYPE (mandates behavior),
#   managerial is a CATEGORY (who implements). A policy is usually
#   managerial (category) + directive (type).
# Compensating cue words: "legacy", "cannot patch", "not feasible",
#   "in the meantime", "alternative control"
```

## 1.1 Category × Type Matrix (Memorize One Example Per Cell)

| | Preventive | Deterrent | Detective | Corrective | Compensating | Directive |
|---|---|---|---|---|---|---|
| **Technical** | Firewall rule, MFA | Login warning banner | IDS, SIEM alert | Restore backup, quarantine malware | Segmentation for unpatchable host | Group policy forcing screen lock |
| **Managerial** | Hiring background checks | Sanction policy (threat of discipline) | Scheduled internal audit | Post-incident policy revision | Risk acceptance with extra insurance | AUP, security policy |
| **Operational** | Guard checking badges at door | Guard patrol visibility | Log review, guard reviewing CCTV | IR runbook execution, re-imaging | Manual log review when SIEM absent | SOP (standard operating procedure) |
| **Physical** | Door lock, bollard, fence | Warning signage, visible camera | Motion sensor, CCTV recording | Fire extinguisher/suppression | Portable generator for failed UPS | "Authorized personnel only" sign |

```
# The 1.1 PBQ is very often literally this matrix: drag examples
# into category/type buckets. Practice until each cell is reflex.
# One control can hold multiple types simultaneously — a visible
# camera is deterrent (seen) + detective (recorded footage).
```

## 1.2 CIA Triad, Non-Repudiation, and Gap Analysis

```
# Confidentiality — only authorized parties can READ data
#   controls: encryption, access controls, DLP, data masking
#   violated by: eavesdropping, exfiltration, shoulder surfing
# Integrity — data is not modified without authorization; changes
#   are detectable
#   controls: hashing (SHA-256), digital signatures, version
#   control, file integrity monitoring (FIM)
#   violated by: on-path tampering, unauthorized DB edits, bit rot
# Availability — data/systems are reachable when needed
#   controls: redundancy, backups, load balancing, patching, UPS
#   violated by: DoS/DDoS, ransomware, hardware failure, disaster
#
# Exam mapping cue:
#   "prevent disclosure/reading"       → confidentiality
#   "detect modification/tampering"    → integrity
#   "keep the service up / uptime"     → availability
#
# Non-repudiation — the sender/actor CANNOT deny having sent/acted
#   mechanism: DIGITAL SIGNATURE (hash of message encrypted with the
#   sender's PRIVATE key; anyone verifies with the public key)
#   Signature gives integrity + authentication + non-repudiation.
#   A MAC/HMAC (shared key) gives integrity + authentication but
#   NOT non-repudiation — both parties hold the same key, either
#   could have produced it. Classic distractor.
#
# Gap analysis — compare CURRENT security posture against a DESIRED
#   baseline/framework (NIST CSF, ISO 27001, CIS) and enumerate the
#   deltas; output feeds the remediation roadmap
#   cue: "where are we vs where we should be", "against the
#   framework", "identify missing controls" → gap analysis
```

## 1.2 AAA — Authentication, Authorization, Accounting

```
# Identification  — CLAIM an identity (username, badge ID) — no proof
# Authentication  — PROVE the claimed identity (password, token,
#                   biometric, certificate)
# Authorization   — what the proven identity MAY DO (permissions)
# Accounting      — RECORD what was done (logs, audit trail; enables
#                   non-repudiation of actions and billing/forensics)
#
# Authenticating PEOPLE — factors:
#   something you know (password/PIN), something you have (smart
#   card, TOTP app, security key), something you are (biometric),
#   somewhere you are (geolocation), something you do (behavioral)
# Authenticating SYSTEMS/devices — no human present:
#   digital certificates (802.1X EAP-TLS machine certs), pre-shared
#   keys, hardware attestation via TPM (trusted platform module),
#   API keys / service tokens for software-to-software
#   cue: "ensure only corporate laptops join the network" →
#   certificate-based device authentication (802.1X)
#
# Authorization MODELS (preview — full detail in cs secplus-iam):
#   RBAC  — role-based: permissions bundled into job roles
#   ABAC  — attribute-based: policy over user/resource/environment
#           attributes (department=HR AND time<18:00)
#   DAC   — discretionary: resource OWNER grants access (NTFS ACLs)
#   MAC   — mandatory: system-enforced labels/clearances
#           (Top Secret; SELinux) — user cannot re-grant
#   Rule-based — global rules applied to all (firewall ACL, time-of-day)
# AAA protocols: RADIUS (network access), TACACS+ (device admin) —
#   see the acronym section of cs security-plus-sy0-701
```

## 1.2 Zero Trust

```
# Principle: NEVER trust, ALWAYS verify — no implicit trust from
# network location; every request is authenticated, authorized, and
# continuously evaluated. Model source: NIST SP 800-207.
# The exam splits the architecture into two planes — know which
# component lives in which plane (favorite matching PBQ):

# CONTROL PLANE — decides. Components:
#   Adaptive identity        — identity validation adjusts to context
#                              (location, device posture, behavior);
#                              risky context → step-up authentication
#   Threat scope reduction   — limit blast radius: least privilege,
#                              minimal implicit trust zones, segment
#   Policy-driven access control — decisions from defined policy over
#                              identity + context, not IP address
#   Policy Engine (PE)       — the BRAIN: evaluates policy + signals,
#                              renders grant/deny per request
#   Policy Administrator (PA)— executes the PE's decision: establishes
#                              or tears down the communication path,
#                              issues session credentials to the PEP
#   (PE + PA together = the Policy Decision Point, PDP)

# DATA PLANE — enforces and carries traffic. Components:
#   Implicit trust zones     — minimized areas where traffic flows
#                              without re-verification (shrink these)
#   Subject/system           — the requesting user/service and the
#                              device it acts through
#   Policy Enforcement Point (PEP) — the GATE in the traffic path:
#                              enables/blocks the connection per the
#                              PA's instruction; the only data-plane
#                              component that talks to the control plane
#
# Flow: subject → PEP → (PEP consults PA, PA asks PE) → PE decides
#       → PA configures PEP → session allowed to the resource
# Exam cues:
#   "decides whether to grant"            → Policy Engine
#   "establishes/manages the session path"→ Policy Administrator
#   "sits inline with the traffic"        → Policy Enforcement Point
#   "MFA prompt only when login looks risky" → adaptive identity
#   "reduce lateral movement"             → threat scope reduction
# Deep dive: cs zero-trust
```

## 1.2 Physical Security

```
# Bollards           — short posts stopping VEHICLES (ram-raid
#                      protection); do not stop pedestrians
# Access control vestibule — two interlocked doors, second opens only
#                      after first closes (formerly "mantrap");
#                      defeats TAILGATING/piggybacking
# Fencing            — perimeter barrier; height matters (~1 m deters
#                      casual, ~2.4 m + barbed top deters determined)
# Video surveillance — CCTV; deterrent when visible + detective via
#                      recording; add motion detection/analytics
# Security guard     — flexible human judgment; two-person integrity
#                      (two guards/persons for sensitive areas)
# Access badge       — proximity/smart card; combine with PIN or
#                      biometric for multi-factor door entry
# Lighting           — deterrent + improves camera/guard effectiveness;
#                      eliminate shadowed approaches
# Sensors:
#   Infrared   — detects body HEAT movement (PIR); indoor motion
#   Pressure   — detects WEIGHT on floor/mat/fence line
#   Microwave  — emits RF, reads reflections; penetrates thin walls,
#                covers larger areas, more false positives
#   Ultrasonic — sound waves above hearing, reads echo changes;
#                sensitive to airflow; often paired with PIR
#                (dual-tech sensor: both must trip → fewer false alarms)
#
# Exam cues:
#   "car driven into the lobby"        → bollards
#   "attacker follows employee inside" → access control vestibule
#     (training against holding the door = operational control)
#   "detect intruder in server room after hours" → motion sensor (IR)
#   "verify WHO entered, not just that someone did" → badge + camera
# Deep dive: cs physical-security
```

## 1.2 Deception and Disruption Technology

```
# Honeypot  — decoy SYSTEM that looks real/vulnerable; any touch is
#             by definition suspicious → high-signal alerting, and it
#             wastes attacker time + reveals TTPs (tactics,
#             techniques, and procedures)
#             low-interaction (emulated services, cheap) vs
#             high-interaction (real OS, richer intel, more risk)
# Honeynet  — an entire decoy NETWORK of honeypots (fake subnet with
#             servers, workstations, services)
# Honeyfile — decoy FILE with enticing name ("passwords.xlsx",
#             "payroll-2026.docx"); access triggers an alert; also
#             marks exfiltrated data
# Honeytoken— decoy DATA/credential that has no legitimate use: fake
#             API key, bogus DB row, fake AWS credentials, dummy email
#             address; any USE of it anywhere proves compromise and
#             traces the leak path
#
# Scale cue: token (datum) < file < pot (host) < net (network)
# "Detect insider snooping on a share"     → honeyfile
# "Detect stolen credentials being used"   → honeytoken
# "Study attacker techniques safely"       → honeypot / honeynet
# Legal note: entrapment concerns are a distractor — honeypots on
#   your own network for detection/research are standard practice
```

## 1.3 Change Management — Business Processes

```
# Why the exam cares: unmanaged change is a top cause of outages and
# security regressions; Domain 1 tests the PROCESS vocabulary.

# Approval process   — change request (RFC) submitted → reviewed →
#                      approved by CAB (change advisory board) or
#                      change owner BEFORE any work happens.
#                      "Technician patched prod without approval" →
#                      the violated step is APPROVAL.
# Ownership          — every change has a named owner accountable for
#                      driving it end-to-end (not necessarily the
#                      person typing the commands)
# Stakeholders       — everyone affected must be identified and
#                      informed (app owners, help desk, users, security)
# Impact analysis    — BEFORE approval: what could break, who is
#                      affected, risk of doing it vs NOT doing it
#                      (security patches: risk of not changing counts)
# Test results       — evidence from a sandbox/staging environment
#                      that the change works, reviewed before approval
# Backout plan       — documented steps to RETURN TO THE PREVIOUS
#                      state if the change fails (snapshot, saved
#                      config, uninstall path) — written BEFORE
#                      implementation, not improvised during failure
# Maintenance window — agreed time period (usually low-usage) when
#                      the change may be executed; limits user impact
# SOP (standard operating procedure) — the documented routine process;
#                      change management itself runs as an SOP, and
#                      routine pre-approved changes ("standard
#                      changes") follow SOPs without full CAB review
#
# Emergency change — expedited path for active incidents; approval is
#   abbreviated/retroactive but STILL documented afterwards
```

## 1.3 Change Management — Technical Implications

```
# Allow lists / deny lists — application control: an allow list
#   permits ONLY listed software/traffic (default-deny; stronger),
#   a deny list blocks listed items (default-allow; weaker).
#   A new app version may need its hash/path re-added to the allow
#   list or it will be blocked post-change.
# Restricted activities — the approval covers a defined SCOPE; work
#   outside it (a "while I'm in here" extra tweak) requires a new
#   change request
# Downtime — expected unavailability during the window; communicate
#   it; use redundancy/rolling updates to shrink it
# Service restart — many changes only take effect after restarting a
#   service/daemon (systemctl restart) — brief interruption
# Application restart — likewise for the app tier; sessions may drop;
#   plan for connection draining
# Legacy applications — old apps with no vendor support or fragile
#   dependencies; may break on OS/library updates; often the reason
#   for COMPENSATING controls (segmentation) instead of patching
# Dependencies — change X forces change Y (library upgrade requires
#   framework upgrade; cert renewal requires trust-store update);
#   map them in impact analysis to sequence the work
```

## 1.3 Change Management — Documentation and Version Control

```
# Updating diagrams — network/architecture diagrams must reflect the
#   change immediately (stale diagrams sabotage the NEXT incident
#   response and the next impact analysis)
# Updating policies/procedures — if the change alters how work is
#   done, the SOPs/policies referencing it must be revised in the
#   same change ticket
# Version control — track versions of configs, code, firewall rule
#   sets, and documents; provides history, diff, rollback target,
#   and an audit trail of WHO changed WHAT WHEN (git for configs,
#   document versioning for policies). Version control is what makes
#   a backout plan trivially executable.
#
# Exam framing: documentation/version control close the loop — a
# change is not COMPLETE until the docs match reality.
```

## Exam Cues (Objective → Keyword → Answer)

| Objective | Question keyword/scenario | Answer |
|---|---|---|
| 1.1 | Control implemented by technology/systems | Technical category |
| 1.1 | Policies, risk assessments, oversight | Managerial category |
| 1.1 | Day-to-day tasks done by people | Operational category |
| 1.1 | Warning sign / visible camera discourages | Deterrent type |
| 1.1 | Lock/firewall stops it outright | Preventive type |
| 1.1 | Logs/IDS reveal it after the fact | Detective type |
| 1.1 | Restore backup / re-image after incident | Corrective type |
| 1.1 | "Cannot patch legacy system, alternative?" | Compensating control |
| 1.1 | AUP/signage mandating behavior | Directive type |
| 1.2 | Prove sender can't deny → mechanism | Digital signature (non-repudiation) |
| 1.2 | HMAC vs signature for non-repudiation | Signature only (HMAC key is shared) |
| 1.2 | Compare posture vs framework baseline | Gap analysis |
| 1.2 | Logs of who did what | Accounting (third A of AAA) |
| 1.2 | Only corporate devices may connect | Device auth via certificates (802.1X) |
| 1.2 | Zero Trust "brain" deciding access | Policy Engine (control plane) |
| 1.2 | Zero Trust component in the traffic path | Policy Enforcement Point (data plane) |
| 1.2 | Step-up MFA on risky sign-in | Adaptive identity |
| 1.2 | Stop vehicles near entrance | Bollards |
| 1.2 | Prevent tailgating through doors | Access control vestibule |
| 1.2 | Detect use of stolen/planted credentials | Honeytoken |
| 1.2 | Decoy file alerts on access | Honeyfile |
| 1.2 | Full decoy network for research | Honeynet |
| 1.3 | Step skipped when tech "just patched prod" | Approval process |
| 1.3 | "If the upgrade fails, then what?" | Backout plan |
| 1.3 | Assess what the change could break first | Impact analysis |
| 1.3 | When may the disruptive work happen | Maintenance window |
| 1.3 | New binary blocked after app update | Update the allow list |
| 1.3 | Fragile unpatchable app in the change path | Legacy application → compensating control |
| 1.3 | Track config history + enable rollback | Version control |
| 1.3 | Network map wrong after the migration | Update diagrams (documentation) |

## See Also

security-plus-sy0-701, secplus-cryptography, secplus-hardening, secplus-iam, secplus-governance-risk, secplus-security-awareness, zero-trust, physical-security, access-control-models, network-access-control, dot1x, risk-management, security-governance, threat-modeling

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/training/resources/exam-objectives)
- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/pubs/sp/800/207/final)
- [NIST SP 800-53 Rev. 5 — Security and Privacy Controls](https://csrc.nist.gov/pubs/sp/800/53/r5/upd1/final)
- [NIST SP 800-63-3 — Digital Identity Guidelines (AAL, authentication factors)](https://pages.nist.gov/800-63-3/)
- [NIST Cybersecurity Framework 2.0](https://www.nist.gov/cyberframework)
- [ISO/IEC 27002:2022 — Information security controls](https://www.iso.org/standard/75652.html)
- [ITIL 4 change enablement practice overview](https://www.axelos.com/certifications/itil-service-management)
