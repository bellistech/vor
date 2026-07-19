# Data Protection & Resilience (Security+ SY0-701 Objectives 3.3–3.4)

> Compare and implement strategies to protect data (types, classifications, states, securing methods) and resilience/recovery in security architecture (HA, sites, backups, testing, power) — the tables-and-tradeoffs heart of Domain 3.

## Objective Map

```
# 3.3 Compare and contrast concepts and strategies to protect data
#   - Data types           (regulated, trade secret, IP, legal, financial,
#                           human-readable, non-human-readable)
#   - Data classifications (sensitive, confidential, public, restricted,
#                           private, critical)
#   - General considerations (data states: at rest / in transit / in use;
#                           data sovereignty; geolocation)
#   - Methods to secure data (geographic restrictions, encryption, hashing,
#                           masking, tokenization, obfuscation, segmentation,
#                           permission restrictions)
#
# 3.4 Explain the importance of resilience and recovery in security
#     architecture
#   - High availability    (load balancing vs clustering)
#   - Site considerations  (hot / warm / cold, geographic dispersion)
#   - Platform diversity, multi-cloud systems
#   - Continuity of operations (COOP)
#   - Capacity planning    (people, technology, infrastructure)
#   - Testing              (tabletop, failover, simulation, parallel processing)
#   - Backups              (onsite/offsite, frequency, encryption, snapshots,
#                           recovery, replication, journaling)
#   - Power                (generators, UPS)
#
# Exam weight: Domain 3 (Security Architecture) = 18% of SY0-701.
# These two objectives are almost entirely compare/contrast — expect
# "which BEST describes…" and "which site type…" table-lookup questions.
```

## Data Types (3.3)

```
# Regulated data — protected by law or industry regulation; mishandling
#   carries legal penalty. Examples:
#     PII  (Personally Identifiable Information)      → privacy laws (GDPR,
#           CCPA — California Consumer Privacy Act)
#     PHI  (Protected Health Information)             → HIPAA (Health
#           Insurance Portability and Accountability Act)
#     Cardholder data                                 → PCI DSS (Payment Card
#           Industry Data Security Standard — contractual, not law, but
#           treated as regulated on the exam)
#   Exam cue: any mention of "compliance requirement" or a named law
#   → the data type is REGULATED.
#
# Trade secret — proprietary business info deriving value from secrecy
#   (formula, recipe, process, customer list). Protected by keeping it
#   secret (NDAs — Non-Disclosure Agreements), not by registration.
#   Lose secrecy → lose protection. Exam cue: "secret formula",
#   "manufacturing process the competitor wants".
#
# Intellectual property (IP) — creations of the mind with legal
#   ownership: patents (registered inventions), copyrights (creative
#   works), trademarks (brand identifiers). Unlike trade secrets, IP is
#   protected THROUGH disclosure + registration.
#   Exam cue: "source code", "designs", "patent filings".
#
# Legal information — attorney-client privileged material, contracts,
#   litigation holds, case files. Often subject to e-discovery and
#   retention mandates.
#
# Financial information — bank records, payroll, earnings statements,
#   payment data. Overlaps regulated (SOX — Sarbanes-Oxley for public
#   companies; GLBA — Gramm-Leach-Bliley for financial institutions).
#
# Human-readable — data a person can directly interpret: documents,
#   spreadsheets, printed reports, email text.
# Non-human-readable — requires machine processing to interpret:
#   binaries, encrypted blobs, barcodes, machine code, serialized data.
#   Exam trap: non-human-readable ≠ secure. A proprietary binary format
#   is obfuscation at best, not encryption.
```

## Data Classifications (3.3)

| Classification | Typical meaning | Example | Handling implication |
|---|---|---|---|
| Public | No harm if disclosed; approved for release | Press releases, marketing site | Minimal controls; integrity still matters |
| Private | Personal or internal data not for outsiders | Employee directory, internal memos | Internal-only access |
| Sensitive | Disclosure causes some harm to org or persons | IP, aggregated PII, internal financials | Access controls + encryption |
| Confidential | Serious damage if disclosed; often NDA-bound | Trade secrets, M&A plans, source code | Strong encryption, need-to-know, logging |
| Restricted | Highest-harm tier; legally/contractually restricted | PHI, cardholder data, keys | Strictest controls, formal authorization per access |
| Critical | Availability-focused: org cannot operate without it | Real-time transaction DB, AD/IdP | HA, replication, priority in DR (Disaster Recovery) |

```
# Classification-scheme notes for the exam:
# - Terms are ORG-DEFINED; schemes vary. SY0-701 tests the relative
#   ordering and the definitions above, not one canonical hierarchy.
# - "Critical" is the odd one out: it is about AVAILABILITY impact
#   (can we operate without it?), while the others are about
#   CONFIDENTIALITY impact (who may see it?).
# - Government scheme (contrast, may appear as distractor):
#   Unclassified < Confidential < Secret < Top Secret.
# - Classification drives control selection AND cost: over-classifying
#   wastes money (everything encrypted, everything restricted),
#   under-classifying creates breach/liability exposure.
# - Data owner assigns classification; data custodian/steward implements
#   the controls (owner vs custodian split is a recurring exam question —
#   see secplus-governance-risk).
```

## Data States (3.3)

| State | Where it lives | Primary protections | Example |
|---|---|---|---|
| At rest | Storage: disk, SSD, tape, object store, DB files | FDE (Full-Disk Encryption), file/volume/DB encryption (AES-256), access controls | Laptop drive, S3 bucket, backup tape |
| In transit | Moving across a network | TLS (Transport Layer Security) 1.2/1.3, IPsec VPN, SSH; certificate validation | HTTPS session, site-to-site VPN |
| In use | RAM / CPU while processed | Memory encryption, secure enclaves (Intel SGX, AMD SEV, TPM-backed), homomorphic encryption (niche), access controls | Decrypted record in application memory |

```
# Exam cues:
#   "stolen laptop"                    → data at rest → full-disk encryption
#   "capture on the wire / sniffing"   → data in transit → TLS / IPsec
#   "protect while being processed" or
#   "RAM scraping / memory dump"       → data in use → secure enclave
# Data in use is the hardest state to protect — the CPU needs plaintext.
# Answer keywords for in-use protection: "secure enclave", "TEE (Trusted
# Execution Environment)", "confidential computing".
```

## Data Sovereignty & Geolocation (3.3)

```
# Data sovereignty — data is subject to the LAWS of the country where it
#   is physically stored (and sometimes where it is collected/processed).
#   Example: EU personal data under GDPR (General Data Protection
#   Regulation) has transfer restrictions out of the EU/EEA.
#   Exam cue: "data stored in country X must comply with X's laws",
#   "cloud provider must keep data in-region" → data sovereignty.
#
# Geolocation — knowing/controlling WHERE data or a user physically is
#   (GPS, IP geolocation, geofencing). Feeds two controls:
#   1. Geographic restrictions / geofencing — block or allow access by
#      location (e.g., deny logins from outside home country; keep a
#      cloud workload pinned to eu-west-1).
#   2. Compliance evidence — prove data never left a jurisdiction.
#
# Distractor trap: sovereignty is a LEGAL concept; geolocation is a
# TECHNICAL capability. A question about "which laws apply" is
# sovereignty; a question about "restrict logins by country" is
# geographic restriction.
```

## Methods to Secure Data (3.3)

| Method | What it does | Reversible? | Best-fit data state | Canonical use case |
|---|---|---|---|---|
| Geographic restrictions | Limit access/storage by location (geofencing, region pinning) | n/a | Any | Sovereignty compliance, block foreign logins |
| Encryption | Cipher + key renders data unreadable without key | Yes (with key) | Rest, transit, (use w/ enclave) | FDE on laptops, TLS, encrypted backups |
| Hashing | One-way fixed-length digest (SHA-256, bcrypt) | No | Rest | Password storage, integrity verification |
| Masking | Hide part of a value, show remainder | No (displayed form) | In use / display | Show last 4 of card: `****-****-****-1234` |
| Tokenization | Replace value with a non-mathematical token; real value in a secure vault | Yes (via vault lookup) | Rest / in use | PCI DSS scope reduction, mobile payments (Apple Pay) |
| Obfuscation | Make data/code hard to interpret (umbrella term) | Sometimes | Rest | Source-code obfuscation, data scrambling in test envs |
| Segmentation | Isolate data into separate network/storage zones | n/a | Rest / transit | Cardholder-data environment on its own VLAN |
| Permission restrictions | Least-privilege ACLs (Access Control Lists) / RBAC | n/a | All | Only payroll group reads salary table |

```
# Compare/contrast the exam LOVES:
#
# Encryption vs hashing:
#   Encryption is REVERSIBLE (decrypt with key) — use when you need the
#   data back. Hashing is ONE-WAY — use for passwords and integrity.
#   "Verify file has not changed" → hashing. "Read it later" → encryption.
#
# Tokenization vs encryption:
#   Tokenization has NO mathematical relationship between token and
#   original — nothing to crack; original sits in a token vault.
#   Encryption is math on the data — key compromise exposes everything.
#   Exam cue: "remove systems from PCI DSS audit scope" → tokenization.
#   Exam cue: "no mathematical relationship" → tokenization.
#
# Masking vs tokenization:
#   Masking is a DISPLAY control (partial hiding, usually irreversible in
#   the shown form) — receipts, support screens.
#   Tokenization SUBSTITUTES the stored value and can be reversed by the
#   vault. "Customer service sees only last 4 digits" → masking.
#
# Obfuscation is the umbrella (masking, tokenization, and steganography
#   are all forms of obfuscation). If obfuscation appears alongside
#   masking/tokenization as options, pick the MORE SPECIFIC term that
#   matches the scenario.
#
# Data Loss Prevention (DLP) — not listed in 3.3's method list but pairs
#   with it in questions: DLP inspects data leaving endpoints/network/
#   cloud and blocks policy violations (e.g., emailing SSNs). "Prevent
#   employees emailing PII" → DLP (see secplus-monitoring-defense).
```

## High Availability: Load Balancing vs Clustering (3.4)

| Attribute | Load balancing | Clustering |
|---|---|---|
| What it is | Distributor spreads requests across independent servers | Multiple servers act as ONE logical system with shared state |
| Server awareness | Servers unaware of each other | Nodes know each other; heartbeat between nodes |
| Primary goal | Scale out + share workload (and HA via health checks) | Fault tolerance / instant failover of a stateful service |
| Failure behavior | Health check fails → stop sending traffic to dead node | Standby node takes over service + shared storage/state |
| Typical layer | Stateless tiers: web, application | Stateful tiers: databases, file services, hypervisors |
| Modes | Active/active (all nodes serve) | Active/active or active/passive (standby waits) |
| Example | HAProxy/F5/ELB in front of 6 web servers | SQL Server failover cluster, Windows Server Failover Clustering |

```
# THE distinction (favorite exam question): load-balanced servers are
# INDEPENDENT and merely share traffic; clustered nodes form a SINGLE
# LOGICAL UNIT with awareness of each other and (usually) shared state.
#
# Active/active — all nodes process work; capacity adds up; watch for
#   overload when one fails (surviving nodes absorb its share).
# Active/passive — standby idles until failover; simpler, capacity of
#   one node, costs double for the idle standby.
#
# Load balancer bonus concepts that appear as answers:
#   persistence/affinity ("sticky sessions") — same client → same server
#   health checks — remove failed nodes automatically
#   scheduling: round-robin, least-connections, weighted
# The load balancer itself must be HA (pair of LBs, VRRP — Virtual
#   Router Redundancy Protocol) or it becomes the single point of failure.
```

## Site Considerations (3.4)

| Site type | What's in place | Time to operational (RTO ballpark) | Relative cost | Data currency |
|---|---|---|---|---|
| Hot site | Full duplicate: power, HVAC, hardware, software, near-real-time data replication | Minutes–hours | $$$$ (highest) | Seconds–minutes behind |
| Warm site | Power, connectivity, some/most hardware; data restored from backups | Hours–days | $$$ (middle) | As fresh as last backup shipment |
| Cold site | Building, power, HVAC only — NO hardware, NO data | Days–weeks | $ (lowest) | None on site |
| Geographic dispersion | Sites far enough apart that one disaster cannot hit both | n/a | n/a | n/a |

```
# Cost/RTO tradeoff is the whole question: pay more → recover faster.
#   "Fastest recovery, cost no object"        → hot site
#   "Balance of cost and recovery time"       → warm site
#   "Cheapest option, long outage acceptable" → cold site
# Mobile site (truck/trailer datacenter) may appear as a distractor —
#   it's a variant, usually wrong unless the scenario says "portable".
#
# Geographic dispersion: recovery site must NOT share the disaster
#   footprint (different flood plain, seismic zone, power grid,
#   hurricane path). Exam cue: "both datacenters lost in the same
#   hurricane" → the fix is geographic dispersion, not a better site tier.
# Rule-of-thumb distances (not codified in SY0-701 but common in
#   questions): far enough that a regional disaster (grid failure,
#   storm) cannot affect both sites simultaneously.
#
# Related metrics (defined in 5.2 / risk, used here):
#   RTO (Recovery Time Objective)  — max tolerable time to restore service
#   RPO (Recovery Point Objective) — max tolerable data loss, measured in
#                                    time ("we can lose at most 15 min")
#   MTTR (Mean Time To Repair)     — average time to fix a failed component
#   MTBF (Mean Time Between Failures) — average uptime between failures
# Site choice is driven by RTO; backup/replication frequency by RPO.
```

## Platform Diversity, Multi-Cloud, COOP, Capacity Planning (3.4)

```
# Platform diversity — avoid a monoculture: mix of vendors, OSes,
#   hardware so one vulnerability or vendor failure can't take down
#   everything. Exam cue: "single vulnerability affected every system"
#   → the missing control is platform diversity.
#   Tradeoff: more platforms = more patching/skills overhead.
#
# Multi-cloud systems — workloads across ≥2 cloud providers (AWS +
#   Azure + GCP). Benefits: no single-provider outage or lock-in,
#   sovereignty options. Costs: duplicated expertise, inconsistent
#   security controls, integration complexity.
#   Exam cue: "cloud provider region outage took the business down"
#   → multi-cloud (or multi-region) resilience.
#
# Continuity of operations (COOP) — the umbrella plan for continuing
#   MISSION-ESSENTIAL functions during and after disruption (term comes
#   from US federal continuity directives). Business Continuity Plan
#   (BCP) = keep operating; Disaster Recovery Plan (DRP) = restore IT
#   after the event; DRP is a subset of BCP. Exam wording: "ensure
#   essential functions continue during a disruption" → COOP/BCP.
#
# Capacity planning — resilience fails when the fallback lacks capacity.
#   Three axes SY0-701 names explicitly:
#   People         — enough trained staff, cross-training, succession
#                    plans, on-call coverage (one admin who knows the
#                    system = a single point of failure).
#   Technology     — licenses, compute headroom, scalability of apps,
#                    surge capacity (autoscaling).
#   Infrastructure — power, cooling, bandwidth, rack space at both
#                    primary and recovery sites.
#   Exam cue: "failover succeeded but the DR site could not handle the
#   production load" → capacity planning failure.
```

## Resilience Testing (3.4)

| Test type | What happens | Disruption to production | Cost/effort | Validates |
|---|---|---|---|---|
| Tabletop exercise | Team talks through a scenario around a table; no systems touched | None | Lowest | Plan completeness, roles, decision-making |
| Simulation | Realistic scripted scenario exercised (may include some technical steps) in a test setting | Low | Medium | Procedures + team response under realistic pressure |
| Failover test | Actually switch production to the backup system/site | Possible outage window | High | The failover mechanism itself really works |
| Parallel processing | Run recovery systems ALONGSIDE production; compare results; production never interrupted | None | High | DR site can genuinely carry the workload |

```
# Disambiguation the exam tests:
#   "discussion-based, no systems touched"        → tabletop
#   "realistic drill / scripted scenario"         → simulation
#   "switch operations to the backup site"        → failover test
#   "both sites process simultaneously, compare
#    output, production untouched"                → parallel processing
# Progression logic: tabletop → simulation → parallel → failover
#   (increasing realism and risk). Start cheap, prove the plan on paper
#   before betting production on it.
# Test regularly AND after major changes; an untested DR plan is
#   assumed broken.
```

## Backups (3.4)

| Backup type | What it copies | Archive-bit behavior | Backup speed | Restore needs |
|---|---|---|---|---|
| Full | Everything, every time | Cleared | Slowest, most storage | 1 set: the full |
| Incremental | Changes since the LAST backup of ANY type | Cleared | Fastest, least storage | Full + EVERY incremental since, in order |
| Differential | Changes since the LAST FULL | NOT cleared | Grows daily | 2 sets: full + latest differential |

```
# Restore-count worked examples (classic exam math):
#
# Schedule A: Full on Sunday, INCREMENTAL Mon–Sat.
#   Failure Thursday afternoon → restore Sunday full + Mon + Tue + Wed
#   incrementals (+ Thu morning's if taken) = 4–5 restore operations,
#   applied IN ORDER. One corrupt incremental breaks the chain from
#   that point forward.
#
# Schedule B: Full on Sunday, DIFFERENTIAL Mon–Sat.
#   Failure Thursday afternoon → restore Sunday full + WEDNESDAY night's
#   differential = 2 restore operations. (Each differential contains
#   everything since the full, so only the newest one is needed.)
#
# Tradeoff summary:
#   Incremental — fastest nightly backup, slowest/most fragile restore.
#   Differential — nightly backup grows through the week, restore is
#     always exactly two sets.
#   "Minimize BACKUP window"  → incremental.
#   "Minimize RESTORE time (without daily fulls)" → differential.
#
# Other 3.4 backup terms:
# Onsite backup  — fast restore, cheap, but shares site disasters
# Offsite backup — survives site loss (tape rotation, cloud); slower
#   3-2-1 rule (best practice, common correct answer): 3 copies,
#   2 different media, 1 offsite.
# Frequency — set by RPO: "lose at most 1 hour" → back up (or replicate
#   journal) at least hourly.
# Encryption — backups MUST be encrypted (at rest and in transit to the
#   backup target); stolen tape/bucket = breach if plaintext.
# Snapshots — point-in-time image of a VM/volume/filesystem (VSS —
#   Volume Shadow Copy Service, LVM, EBS snapshots). Near-instant,
#   great for pre-change rollback; NOT a substitute for backups if
#   stored on the same array as the source.
# Replication — continuous copy to another system/site.
#   Synchronous  — write confirmed at both sides first; zero data loss
#                  (RPO≈0); distance-limited by latency.
#   Asynchronous — lag between sites; some loss possible; any distance.
# Journaling — log every change/transaction (DB write-ahead logs,
#   journaled filesystems); replay the journal on top of a restore to
#   roll forward to the moment of failure → drives RPO toward zero.
#   Exam cue: "recover to the exact point of failure" → journaling.
# Recovery — actually test restores. A backup that has never been
#   restored is Schrödinger's backup; "backups completed nightly but
#   restore failed" → missing recovery testing.
```

## Power (3.4)

```
# UPS (Uninterruptible Power Supply) — battery bridge for SHORT outages
#   (minutes): conditions power (sags/surges), carries the load until
#   the generator starts or systems shut down gracefully.
#   Types (may appear as detail distractors): standby/offline,
#   line-interactive, double-conversion online (always-on inverter,
#   cleanest power, zero transfer time).
#
# Generator — diesel/natural-gas for LONG outages (hours–days); takes
#   seconds–minutes to start and stabilize → ALWAYS paired with a UPS
#   to cover the gap. Needs fuel contracts + periodic load testing.
#
# Exam cues:
#   "ride through a 5-minute outage / graceful shutdown" → UPS
#   "operate for 3 days without utility power"           → generator
#   "generator takes 60 s to start; servers reboot"      → add a UPS
# Related redundancy: dual power supplies fed from separate PDUs
#   (Power Distribution Units) on separate circuits, A/B utility feeds.
```

## Exam-Cue Keyword → Answer Map

| Question keyword/phrase | Reflex answer |
|---|---|
| "subject to the laws of the country where stored" | Data sovereignty |
| "block logins from foreign countries" | Geographic restriction / geofencing |
| "no mathematical relationship to original value" | Tokenization |
| "reduce PCI DSS audit scope" | Tokenization |
| "show only last four digits" | Masking |
| "one-way, verify integrity, store passwords" | Hashing |
| "stolen laptop, data unreadable" | Full-disk encryption (data at rest) |
| "protect data being processed in memory" | Secure enclave / data in use |
| "servers share client traffic, unaware of each other" | Load balancing |
| "nodes act as one system, heartbeat, shared storage" | Clustering |
| "fully equipped duplicate site, minutes to switch" | Hot site |
| "empty building with power and HVAC only" | Cold site |
| "one storm knocked out both datacenters" | Geographic dispersion |
| "one vulnerability hit every identical system" | Platform diversity |
| "discussion-based walkthrough of the DR plan" | Tabletop exercise |
| "run DR site alongside production, compare results" | Parallel processing |
| "restore = full + every backup since, in order" | Incremental |
| "restore = full + most recent one only" | Differential |
| "recover to the exact moment of failure" | Journaling / transaction log replay |
| "zero data loss replication" | Synchronous replication (RPO ≈ 0) |
| "3 copies, 2 media, 1 offsite" | 3-2-1 backup rule |
| "bridge power until generator spins up" | UPS |
| "maximum tolerable data loss in time" | RPO |
| "maximum tolerable downtime" | RTO |

## See Also

security-plus-sy0-701, secplus-architecture, secplus-hardening, secplus-governance-risk, secplus-incident-response, secplus-monitoring-defense, secplus-cryptography, bcp-drp, cloud-security, security-architecture, risk-management, pki, tls

## References

- CompTIA Security+ SY0-701 Exam Objectives — https://www.comptia.org/certifications/security
- NIST SP 800-34 Rev. 1 — Contingency Planning Guide for Federal Information Systems — https://csrc.nist.gov/publications/detail/sp/800-34/rev-1/final
- NIST SP 800-111 — Guide to Storage Encryption Technologies for End User Devices — https://csrc.nist.gov/publications/detail/sp/800-111/final
- NIST SP 800-57 Part 1 — Recommendation for Key Management — https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final
- PCI Security Standards Council — Tokenization Product Security Guidelines — https://www.pcisecuritystandards.org/
- GDPR (Regulation (EU) 2016/679) — https://eur-lex.europa.eu/eli/reg/2016/679/oj
- FEMA Continuity Resource Toolkit (COOP) — https://www.fema.gov/emergency-managers/national-preparedness/continuity
- RFC 8446 — The Transport Layer Security (TLS) Protocol Version 1.3 — https://www.rfc-editor.org/rfc/rfc8446
