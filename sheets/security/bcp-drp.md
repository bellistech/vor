# BCP/DRP (Business Continuity & Disaster Recovery Planning)

> Plan for, respond to, and recover from disruptive events using BIA, recovery strategies, backup methods, and structured testing to maintain critical operations.

## BCP/DRP Lifecycle

```
# BCP/DRP lifecycle phases
# 1. Project initiation — scope, team, management support
# 2. Business Impact Analysis (BIA) — identify critical functions
# 3. Risk assessment — threats to business continuity
# 4. Recovery strategy development — how to restore operations
# 5. Plan development — document procedures and responsibilities
# 6. Training and awareness — ensure staff know their roles
# 7. Testing and exercises — validate the plan works
# 8. Plan maintenance — review, update, improve continuously

# BCP vs DRP
# BCP = sustaining business operations during and after disruption
# DRP = restoring IT systems and infrastructure after a disaster
# BCP is broader — DRP is a subset focused on technology recovery
```

## Business Impact Analysis (BIA)

```
# BIA identifies critical business functions and their dependencies
# Steps:
# 1. Identify all business processes and functions
# 2. Determine which are critical (mission-essential)
# 3. Assess impact of disruption over time
# 4. Identify dependencies (systems, data, people, facilities)
# 5. Establish recovery priorities and time objectives
# 6. Document resource requirements for recovery

# Impact categories assessed:
# Financial — lost revenue, penalties, extra expenses
# Operational — inability to deliver products/services
# Legal/Regulatory — compliance violations, contractual breaches
# Reputational — customer trust, brand damage
# Safety — risk to life, health, environment

# BIA output:
# - Ranked list of critical business functions
# - Recovery time objectives per function
# - Resource dependencies and minimum requirements
# - Impact escalation curves (impact vs. time)
```

## Recovery Time Objectives

```
# MTD — Maximum Tolerable Downtime
# Longest time a function can be disrupted before
# unacceptable consequences (business failure, regulatory breach)
# Set by business leadership based on BIA
# Example: payment processing MTD = 4 hours

# RTO — Recovery Time Objective
# Target time to restore a function after disruption
# Must be less than MTD (RTO < MTD)
# Includes: detection + response + recovery + verification
# Example: payment processing RTO = 2 hours

# RPO — Recovery Point Objective
# Maximum acceptable data loss measured in time
# Determines backup frequency
# RPO = 0 → synchronous replication (zero data loss)
# RPO = 1 hour → backups/replication at least hourly
# RPO = 24 hours → daily backups sufficient
# Example: transaction database RPO = 15 minutes

# WRT — Work Recovery Time
# Time to verify system integrity and catch up on backlog
# after technical recovery is complete
# RTO + WRT ≤ MTD
# Example: WRT = 1 hour to verify data integrity and process queue

# MTPD — Maximum Tolerable Period of Disruption
# Equivalent to MTD in some frameworks (ISO 22301 uses MTPD)
# The point beyond which the organization cannot survive

# Timeline:
# [Disruption]→ [Detection] → [Recovery (RTO)] → [Catch-up (WRT)] → [Normal]
#                                                                     ↑
# ←————————————————— MTD ——————————————————————————————————————————————→
# ←——— RTO ———→←— WRT —→
# ←— RPO —→ (data loss window, measured backward from disruption)

# MTBF — Mean Time Between Failures
# Average time a system runs before failing
# Higher MTBF = more reliable

# MTTR — Mean Time To Repair
# Average time to restore a failed system
# Lower MTTR = faster recovery

# Availability = MTBF / (MTBF + MTTR)
```

## Recovery Strategies

### Recovery Sites

```
# Hot site
# - Fully equipped, mirrors production environment
# - Real-time or near-real-time data replication
# - Switchover in minutes to hours
# - Most expensive option
# - RTO: minutes to 1 hour
# - Best for: mission-critical operations with near-zero MTD

# Warm site
# - Hardware and network in place, partially configured
# - Data restored from recent backups (not real-time)
# - Switchover in hours to days
# - Moderate cost
# - RTO: 4–24 hours
# - Best for: important functions with moderate MTD

# Cold site
# - Empty facility with power, HVAC, network connectivity
# - No hardware or data pre-installed
# - Requires equipment delivery and full setup
# - Least expensive option
# - RTO: days to weeks
# - Best for: non-critical functions with long MTD

# Mobile site
# - Self-contained trailer/container with computing equipment
# - Can be deployed to any location
# - RTO: hours to days
# - Best for: distributed operations, field offices

# Reciprocal agreement
# - Two organizations agree to host each other in disaster
# - Low cost, but capacity and compatibility concerns
# - Difficult to enforce, limited scalability
# - RTO: variable

# Cloud DR (DRaaS — Disaster Recovery as a Service)
# - Infrastructure replicated to cloud provider
# - Pay-per-use model (reduced standby cost)
# - Automated failover and failback
# - RTO: minutes to hours (depending on tier)
# - Best for: organizations wanting hot-site capabilities at lower cost
```

### Site Selection Criteria

```
# Geographic separation — far enough to avoid same disaster
#   Minimum: different power grid, flood zone, seismic zone
#   Rule of thumb: at least 100 miles / 160 km from primary
# Network connectivity — sufficient bandwidth for replication
# Accessibility — staff can reach the site during a disaster
# Security — physical and logical security comparable to primary
# Regulatory compliance — data sovereignty and jurisdictional requirements
# Cost — one-time setup + ongoing operational expenses
# Capacity — sufficient to handle critical workloads
# Vendor proximity — access to replacement hardware and support
```

## Backup Strategies

```
# Full backup
# - Copies all data regardless of changes
# - Longest backup time, largest storage
# - Fastest restore (single backup set)
# - Clear archive bit on all files

# Incremental backup
# - Copies only data changed since LAST BACKUP (any type)
# - Fastest backup, smallest storage per run
# - Slowest restore (needs full + all incrementals in order)
# - Clears archive bit on backed-up files

# Differential backup
# - Copies all data changed since LAST FULL backup
# - Moderate backup time (grows daily)
# - Faster restore than incremental (full + latest differential)
# - Does NOT clear archive bit

# Restore comparison:
# Full only:       restore latest full
# Full + Diff:     restore latest full + latest differential
# Full + Incr:     restore latest full + each incremental in order

# Example: Full on Sunday, daily increments Mon–Fri
# Wednesday failure → restore: Sun full + Mon incr + Tue incr + Wed incr

# Example: Full on Sunday, daily differentials Mon–Fri
# Wednesday failure → restore: Sun full + Wed differential
```

### 3-2-1 Backup Rule

```
# 3 — Keep at least 3 copies of data
#     (1 primary + 2 backups)
# 2 — Store on at least 2 different media types
#     (disk + tape, local + cloud, SSD + HDD)
# 1 — Keep at least 1 copy offsite
#     (cloud storage, remote office, tape vault)

# Extended: 3-2-1-1-0
# 1 — One copy offline or air-gapped (ransomware protection)
# 0 — Zero errors (verified backups with integrity checks)

# RPO alignment:
# RPO = 0         → synchronous replication
# RPO = 15 min    → near-continuous replication (CDP)
# RPO = 1 hour    → hourly snapshots/replication
# RPO = 4 hours   → 4-hourly backups
# RPO = 24 hours  → nightly full or differential
# RPO = 1 week    → weekly full backups
```

## DR Testing Types

```
# Checklist / Document review (lowest maturity)
# - Distribute plan; reviewers check for completeness
# - No simulation, no actual recovery
# - Finds: missing procedures, outdated contacts

# Tabletop exercise
# - Key personnel walk through a scenario verbally
# - No actual systems affected
# - Finds: logic gaps, role confusion, decision bottlenecks
# - Duration: 2–4 hours

# Walkthrough / Structured walkthrough
# - Team walks through each step of the plan
# - May include visiting recovery sites
# - Finds: logistical issues, resource gaps
# - Duration: half day to full day

# Simulation test
# - Realistic scenario presented (e.g., simulated ransomware)
# - Teams execute response procedures in a controlled environment
# - No actual production impact
# - Finds: communication failures, procedure errors
# - Duration: 1–2 days

# Parallel test
# - Recovery systems brought online at the DR site
# - Production systems continue to run (no actual cutover)
# - Validates that DR systems can handle the workload
# - Finds: capacity issues, replication lag, configuration drift
# - Duration: 1–3 days

# Full interruption test (highest maturity)
# - Production systems shut down; operations move to DR site
# - Most realistic but highest risk
# - Only perform after successful parallel tests
# - Validates true failover and failback procedures
# - Finds: everything — the ultimate validation
# - Duration: hours to days
# - Risk: if DR fails, real outage occurs
```

## DR Plan Structure

```
# Essential sections of a DR plan:
# 1. Purpose and scope
# 2. Roles and responsibilities (DR team, alternates)
# 3. Activation criteria and declaration authority
# 4. Notification and escalation procedures
# 5. Emergency contact list (internal + external)
# 6. System inventory and priorities (from BIA)
# 7. Recovery procedures (step-by-step per system)
# 8. Communication plan (stakeholders, media, customers)
# 9. Alternate site procedures and logistics
# 10. Data restoration procedures
# 11. Failback procedures (return to primary)
# 12. Testing schedule and results log
# 13. Plan maintenance schedule
# 14. Appendices: vendor contacts, diagrams, credentials

# Incident declaration authority
# - Who can declare a disaster? (typically CIO, CISO, or BCP manager)
# - Criteria for declaration (outage duration, scope, severity)
# - Escalation path if primary authority is unavailable
```

## Crisis Communication

```
# Internal communication
# - Emergency notification system (mass notification, call tree)
# - Status updates at regular intervals
# - Clear chain of command and spokesperson designation
# - Employee safety accounting

# External communication
# - Customer notification (what happened, impact, timeline)
# - Regulatory notification (breach, material event)
# - Media/PR response (single spokesperson, prepared statements)
# - Partner/vendor coordination
# - Legal review of all external communications

# Communication channels (primary + backup)
# - Email (may be unavailable during IT disaster)
# - SMS/text messaging
# - Phone/conference bridge
# - Out-of-band messaging (e.g., personal mobile, satellite phone)
# - Physical meeting location (rally point)
```

## Cloud DR (DRaaS)

```
# DRaaS — Disaster Recovery as a Service
# Cloud provider manages DR infrastructure

# Pilot light
# - Minimal core infrastructure running in cloud
# - Database replicated, servers pre-configured but stopped
# - Scale up when disaster declared
# - RTO: hours
# - Cost: low (pay for minimal running resources)

# Warm standby
# - Scaled-down version of production running in cloud
# - All components active at reduced capacity
# - Scale up and redirect traffic when needed
# - RTO: minutes to hours
# - Cost: moderate

# Multi-site active-active
# - Full production running in multiple regions/clouds
# - Traffic distributed across all sites
# - Automatic failover with no downtime
# - RTO: near zero
# - RPO: near zero (synchronous replication)
# - Cost: high (running full duplicate infrastructure)

# Key cloud DR considerations
# - Data sovereignty (where is replicated data stored?)
# - Bandwidth costs for replication
# - Provider lock-in and portability
# - Testing capabilities (spin up DR on demand)
# - SLA guarantees from the provider
```

## Tips

- RTO + WRT must always be less than or equal to MTD; if not, your recovery strategy is inadequate.
- RPO determines backup frequency — if RPO is 1 hour, nightly backups are insufficient.
- Test your DR plan at least annually; tabletop exercises should be quarterly.
- Never perform a full interruption test without first succeeding at parallel testing.
- Cloud DRaaS has transformed cost economics — hot-site capability at warm-site pricing.
- Document lessons learned after every test and every real incident.

## See Also

- risk-management, incident-response, security-operations, backup, cloud-security

## References

- [NIST SP 800-34 Rev 1 — Contingency Planning Guide for Federal Information Systems](https://csrc.nist.gov/publications/detail/sp/800-34/rev-1/final)
- [ISO 22301:2019 — Business Continuity Management Systems](https://www.iso.org/standard/75106.html)
- [ISO 22313:2020 — Business Continuity Management Systems Guidance](https://www.iso.org/standard/75107.html)
- [NIST SP 800-53 Rev 5 — CP (Contingency Planning) Control Family](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [BCI Good Practice Guidelines](https://www.thebci.org/training-qualifications/good-practice-guidelines.html)
- [DRI International Professional Practices](https://drii.org/resources/professionalpractices/EN)
- [AWS Well-Architected — Reliability Pillar](https://docs.aws.amazon.com/wellarchitected/latest/reliability-pillar/)
