# Security Operations — Deep Dive

Theoretical foundations of SOC design, SIEM correlation engines, threat intelligence lifecycle, intrusion analysis models, SOAR orchestration, and ML-based anomaly detection.

## SOC Maturity Model

### CMM-Based SOC Maturity Levels

SOC maturity is commonly assessed across five levels, based on the Capability Maturity Model:

**Level 1 — Initial (Ad Hoc)**
- No formal SOC structure; security monitoring is reactive
- Alerts are handled by IT operations staff, not dedicated analysts
- No documented incident response procedures
- SIEM may exist but is poorly configured with default rules
- Detection is primarily signature-based with minimal tuning
- Metrics are not tracked; no visibility into detection effectiveness

**Level 2 — Managed (Defined)**
- Dedicated SOC team with defined roles (Tier 1/2/3)
- Documented incident response plan and basic playbooks
- SIEM deployed with tuned correlation rules (50-100 active rules)
- Regular vulnerability scanning with remediation tracking
- Basic metrics tracked (alert volume, MTTA, ticket closure time)
- Threat intelligence consumed but not operationalized

**Level 3 — Defined (Proactive)**
- Formal detection engineering program (rules reviewed quarterly)
- Threat hunting program with regular cadence (weekly/biweekly)
- SOAR deployed for high-volume incident types (phishing, malware)
- Purple teaming exercises conducted quarterly
- Metrics-driven improvement (false positive rate, MTTD, MTTR tracked and trended)
- Threat intelligence integrated into SIEM and automated workflows
- Log coverage assessment performed; gaps documented and tracked

**Level 4 — Quantitatively Managed**
- Detection coverage mapped to MITRE ATT&CK with gap analysis
- Automated detection testing (continuous validation)
- ML-based anomaly detection supplementing rule-based detection
- Full automation of Tier 0 workflows (auto-enrichment, auto-close known FPs)
- Business-aligned risk metrics reported to executive leadership
- Tabletop and live-fire exercises with measurable improvement tracking
- Cross-functional integration (SOC + IT Ops + DevOps + Legal)

**Level 5 — Optimizing**
- Predictive analytics and proactive threat prevention
- Continuous detection engineering with automated rule lifecycle management
- Zero-trust integration: SOC telemetry drives dynamic access decisions
- Threat intelligence production (not just consumption); sharing with ISACs
- Research program: SOC team publishes findings, develops novel detection methods
- Full kill chain coverage validated through continuous simulation
- Sub-hour MTTD and MTTR for critical assets

### Maturity Assessment Framework

Key dimensions to evaluate:

```
Dimension         | Level 1        | Level 3          | Level 5
------------------|----------------|------------------|------------------
People            | Shared IT role | Dedicated tiers  | Specialized + research
Process           | Ad hoc         | Documented       | Continuously optimized
Technology        | Basic SIEM     | SIEM + SOAR      | SIEM + SOAR + ML + XDR
Threat Intel      | None           | Consumed         | Produced + shared
Detection         | Vendor rules   | Custom + tuned   | Continuously validated
Response          | Manual         | Playbook-guided  | Automated + orchestrated
Metrics           | None           | Basic tracking   | Predictive analytics
Coverage          | Unknown        | Partially mapped | ATT&CK validated
```

## SIEM Correlation Engine Theory

### Event Processing Pipeline

A SIEM correlation engine processes events through a multi-stage pipeline:

```
Raw Log Ingestion
  |
  v
Stage 1: Parsing
  - Identify log format (syslog, CEF, LEEF, JSON, W3C, custom)
  - Extract fields into normalized schema
  - Regular expressions, delimiters, or structured parsers
  - Handle multi-line logs (stack traces, certificate data)
  |
  v
Stage 2: Normalization
  - Map vendor-specific field names to common schema
  - Example: Palo Alto "srcip" = Cisco "src_addr" = normalized "source.ip"
  - Common schemas: CIM (Splunk), ECS (Elastic), OCSF
  - Normalize timestamps to UTC
  - Resolve hostnames to IPs (or vice versa)
  |
  v
Stage 3: Enrichment
  - GeoIP lookup for external IPs
  - Asset context (CMDB lookup: asset owner, criticality, OS)
  - User context (directory lookup: department, title, location)
  - Threat intel lookup (IoC match: known bad IP/domain/hash)
  - Vulnerability context (is this asset vulnerable to this attack?)
  |
  v
Stage 4: Correlation
  - Apply correlation rules against enriched events
  - Multiple correlation types (see below)
  - State machines track multi-event sequences
  - Sliding windows for time-based correlations
  |
  v
Stage 5: Alert Generation
  - Create alert with severity, affected assets, and evidence
  - Deduplicate: group related alerts into single incident
  - Route to appropriate queue (Tier 1/2/3 based on severity)
  - Trigger automated response if SOAR integration exists
```

### Correlation Engine Architectures

**Rule-based correlation (traditional):**
```
Architecture: Event stream --> Rule evaluation engine --> Alerts

Rule types:
1. Simple filter: single event matches condition
   IF event.type = "authentication" AND event.outcome = "failure"
   THEN create alert

2. Aggregation: count events in time window
   IF COUNT(authentication_failure WHERE source.ip = X) > 10
   IN WINDOW 5 minutes
   THEN create alert "Brute Force from {X}"

3. Sequence: ordered events in time window
   IF event_A(auth_failure, count > 5)
   FOLLOWED BY event_B(auth_success)
   WHERE event_A.source.ip = event_B.source.ip
   IN WINDOW 15 minutes
   THEN create alert "Successful Brute Force"

4. State machine: track entity through states
   Entity: source.ip
   State transitions:
     NORMAL --[scan_detected]--> RECON
     RECON --[exploit_attempt]--> ATTACK
     ATTACK --[auth_success]--> COMPROMISED
     Any state --[24h_no_activity]--> NORMAL
   Alert on: transition to COMPROMISED

Pros: Deterministic, explainable, low compute cost
Cons: Cannot detect unknown patterns, requires manual rule writing
```

**Statistical correlation:**
```
Architecture: Event stream --> Baseline computation --> Deviation detection

Approach:
1. Compute baselines per entity (user, host, network segment)
   - Normal login hours: 08:00-18:00 weekdays
   - Normal data transfer volume: 50-200 MB/day
   - Normal DNS query rate: 100-500/hour

2. Detect deviations beyond threshold (typically 2-3 sigma)
   - Login at 03:00 on Saturday from new country
   - Data transfer of 5 GB in 1 hour
   - DNS query rate of 5000/hour

3. Score deviations based on magnitude and context
   - 3 sigma deviation on critical asset = high alert
   - 2 sigma deviation on dev workstation = low alert

Pros: Detects novel attacks, adapts to changing environments
Cons: Requires baseline period, generates more false positives during
      legitimate changes (software rollout, business travel)
```

**Graph-based correlation:**
```
Architecture: Events --> Graph construction --> Pattern matching

Nodes: entities (users, hosts, IPs, files, processes)
Edges: relationships (logged_into, connected_to, executed, accessed)

Detection patterns:
- Unusual graph diameter (attacker traverses many hops)
- New edges between previously unconnected nodes
- Subgraph matching against known attack patterns
- Community detection (identify lateral movement clusters)

Example attack graph:
  phishing_email --> user_click --> malware_download
  malware_download --> process_exec --> c2_connection
  c2_connection --> credential_dump --> lateral_movement
  lateral_movement --> data_access --> exfiltration

The graph structure reveals the attack narrative that individual
events cannot show in isolation.

Pros: Reveals complex multi-stage attacks, visual investigation support
Cons: High compute/memory cost, graph construction latency
```

## Log Normalization and Enrichment

### The Normalization Problem

Different vendors represent the same event in completely different formats. Normalization maps them to a common schema.

**Example: failed SSH login across three log formats:**

```
Syslog (Linux):
  Feb 10 14:23:01 web01 sshd[12345]: Failed password for admin
  from 192.168.1.100 port 45678 ssh2

Windows Security Event (JSON):
  {"EventID": 4625, "TimeCreated": "2026-02-10T14:23:01Z",
   "TargetUserName": "admin", "IpAddress": "192.168.1.100",
   "LogonType": 10, "Status": "0xC000006D"}

Palo Alto NGFW (CEF):
  CEF:0|Palo Alto|NGFW|10.0|auth-fail|Authentication Failed|5|
  src=192.168.1.100 dst=10.0.0.5 suser=admin cs1=ssh

Normalized to common schema:
  {
    "timestamp": "2026-02-10T14:23:01Z",
    "event.category": "authentication",
    "event.outcome": "failure",
    "source.ip": "192.168.1.100",
    "destination.ip": "10.0.0.5",
    "user.name": "admin",
    "process.name": "sshd",
    "event.provider": "linux_sshd"
  }
```

### Common Event Schemas

**Elastic Common Schema (ECS):**
- Hierarchical field naming: `source.ip`, `destination.port`, `process.name`
- 800+ defined fields across 30+ field sets
- Extensible: custom fields allowed under `custom.*` namespace
- Used by Elastic SIEM, increasingly adopted industry-wide

**Splunk Common Information Model (CIM):**
- Data model-based: Authentication, Network Traffic, Endpoint, Web, etc.
- Each model defines expected fields and accelerated data summaries
- Tags and event types map raw events to CIM data models

**OCSF (Open Cybersecurity Schema Framework):**
- Vendor-neutral schema developed by AWS and Splunk
- Category-based: System Activity, Findings, Network Activity, etc.
- Each category defines event classes with required and optional attributes
- Designed for cross-product interoperability

### Enrichment Pipeline

```
Raw normalized event
  |
  +-- GeoIP enrichment
  |     Input: source.ip = 185.220.101.42
  |     Output: source.geo.country = "DE"
  |             source.geo.city = "Frankfurt"
  |             source.geo.asn = 24940
  |             source.geo.org = "Hetzner Online"
  |
  +-- Asset enrichment (CMDB lookup)
  |     Input: destination.ip = 10.0.1.50
  |     Output: asset.name = "prod-web-01"
  |             asset.owner = "web-team"
  |             asset.criticality = "high"
  |             asset.os = "Ubuntu 22.04"
  |             asset.business_unit = "e-commerce"
  |
  +-- User enrichment (directory lookup)
  |     Input: user.name = "jsmith"
  |     Output: user.full_name = "John Smith"
  |             user.department = "Engineering"
  |             user.title = "Senior Developer"
  |             user.manager = "Jane Doe"
  |             user.risk_score = 35
  |
  +-- Threat intel enrichment
  |     Input: source.ip = 185.220.101.42
  |     Output: threat.indicator.type = "ip"
  |             threat.indicator.confidence = "high"
  |             threat.indicator.description = "Tor exit node"
  |             threat.indicator.feed = "abuse.ch"
  |
  +-- Vulnerability enrichment
        Input: asset.name = "prod-web-01"
        Output: asset.cve_count = 3
                asset.critical_cves = ["CVE-2026-1234"]
                asset.last_scan = "2026-04-03"
```

## Threat Intelligence Lifecycle

### Six Phases

**Phase 1: Direction (Planning and Requirements)**
- Define intelligence requirements based on organizational risk profile
- Identify priority intelligence requirements (PIRs):
  - Which threat actors target our industry?
  - What TTPs are currently active against our technology stack?
  - Which vulnerabilities are being actively exploited in the wild?
- Establish intelligence dissemination plan (who gets what, how often)

**Phase 2: Collection**
- **Open source (OSINT):** Threat reports, blog posts, CVE databases, paste sites, social media, dark web monitoring
- **Commercial feeds:** Recorded Future, Mandiant, CrowdStrike, Cisco Talos
- **Community sharing:** ISACs (Information Sharing and Analysis Centers), MISP instances, STIX/TAXII feeds
- **Internal sources:** SOC alerts, incident reports, vulnerability scans, red team findings
- **Technical collection:** Honeypots, malware sandboxes, DNS sinkholes, network sensors

**Phase 3: Processing**
- Convert raw data into structured intelligence
- Parse IOCs from unstructured reports (regex extraction of IPs, domains, hashes)
- Normalize indicators to common format (STIX 2.1)
- Deduplicate across sources
- Validate accuracy (false positive check, confirm indicators are still active)

**Phase 4: Analysis**
- Correlate indicators with internal telemetry (have we seen this?)
- Attribute activity to threat actors or campaigns
- Assess relevance to organization (does this threat target our industry/technology?)
- Determine confidence level (how reliable is the source? confirmed by multiple sources?)
- Produce actionable intelligence (what should we do about this?)

**Phase 5: Dissemination**
- **Strategic intelligence:** Executive briefings, board reports, risk assessments (quarterly)
- **Operational intelligence:** Campaign analysis, threat actor profiles for SOC leadership (weekly)
- **Tactical intelligence:** IOCs, detection rules, YARA rules for SOC analysts (daily/real-time)
- Deliver through appropriate channels: SIEM integration, email reports, threat intel platform, dashboards

**Phase 6: Feedback**
- Evaluate whether intelligence was timely, relevant, and actionable
- Track how intelligence was used (detection rules created, investigations triggered)
- Refine collection requirements based on gaps identified
- Measure intelligence program effectiveness (coverage of threat landscape, time to operationalize new intel)

## Diamond Model of Intrusion Analysis

The Diamond Model provides a framework for analyzing and correlating cyber intrusion events. Every intrusion event has four core features:

```
                    Adversary
                       /\
                      /  \
                     /    \
                    /      \
     Capability ---/--------\--- Infrastructure
                  /   Event   \
                 /      |      \
                /       |       \
               /________|________\
                      Victim

Core features:
  Adversary: the threat actor (nation-state, criminal group, insider)
  Capability: the tools and techniques used (malware, exploit, TTP)
  Infrastructure: the resources used to deliver (C2 servers, domains, email)
  Victim: the target (organization, system, person, data)

Meta-features:
  Timestamp: when the event occurred
  Phase: which kill chain phase
  Result: success or failure
  Direction: adversary-to-victim or victim-to-adversary
  Methodology: specific technique (phishing, drive-by, supply chain)
  Resources: what the adversary invested (money, time, knowledge)
```

### Analytical Application

**Pivoting across the diamond:**
```
Known: Malware hash (Capability)
  |
  Pivot to Infrastructure:
    Where does this malware phone home? (C2 domains/IPs)
  |
  Pivot to Victim:
    Which of our endpoints communicated with this C2?
  |
  Pivot to Adversary:
    Which threat actor is known to use this malware family?
  |
  Pivot back to Capability:
    What other tools does this actor use?
  |
  Loop: search for those tools in our environment
```

**Activity threads and groups:**
- An activity thread links multiple diamond events into a sequence (one intrusion campaign)
- Activity groups cluster related threads that share adversary or infrastructure features
- This enables attribution: "these 15 events across 3 organizations share the same C2 infrastructure and malware family, likely the same adversary"

## Kill Chain Analysis in SOC

### Lockheed Martin Cyber Kill Chain

```
Phase 1: Reconnaissance
  Adversary: identifies targets, gathers intelligence
  SOC detection: monitor for scanning, OSINT mentions, domain registration
  Log sources: web server logs, DNS logs, honeypot logs
  Example: repeated LinkedIn lookups of employees, Shodan queries for our IPs

Phase 2: Weaponization
  Adversary: creates deliverable payload (exploit + backdoor)
  SOC detection: generally not detectable (happens in adversary environment)
  Exception: if adversary tests payload against our decoy infrastructure

Phase 3: Delivery
  Adversary: transmits weapon to victim (email, web, USB)
  SOC detection: email gateway, web proxy, endpoint AV
  Log sources: email logs, proxy logs, endpoint logs
  Example: phishing email with malicious attachment detected by email gateway

Phase 4: Exploitation
  Adversary: triggers the vulnerability
  SOC detection: IDS/IPS, endpoint exploit prevention, application logs
  Log sources: IDS alerts, endpoint telemetry, application error logs
  Example: Word document exploits CVE-XXXX, detected by exploit prevention

Phase 5: Installation
  Adversary: installs backdoor for persistent access
  SOC detection: EDR (new service, scheduled task, registry modification)
  Log sources: endpoint telemetry, file integrity monitoring
  Example: new service created at 03:00, detected by EDR behavioral rule

Phase 6: Command and Control (C2)
  Adversary: establishes communication channel
  SOC detection: DNS anomaly, proxy logs, network IDS, threat intel matching
  Log sources: DNS logs, proxy logs, firewall logs, NetFlow
  Example: beaconing to DGA domain every 60 seconds, detected by DNS analytics

Phase 7: Actions on Objectives
  Adversary: achieves goal (exfiltration, destruction, espionage)
  SOC detection: DLP, anomaly detection, file access auditing
  Log sources: DLP alerts, file access logs, database audit logs
  Example: 5 GB of data uploaded to cloud storage at 02:00, DLP alert
```

### Kill Chain Coverage Matrix

```
For each kill chain phase, assess:
  - Which log sources provide visibility?
  - Which detection rules exist?
  - What is the estimated detection probability?
  - What response actions are available?

Phase         | Log Sources     | Rules | Det.Prob. | Response
--------------|-----------------|-------|-----------|----------
Recon         | DNS, honeypot   | 3     | 15%       | Block IP
Delivery      | Email, proxy    | 12    | 70%       | Quarantine
Exploitation  | IDS, EDR        | 8     | 55%       | Isolate host
Installation  | EDR, FIM        | 10    | 60%       | Quarantine file
C2            | DNS, proxy, FW  | 15    | 65%       | Block domain
Actions       | DLP, DB audit   | 5     | 30%       | Isolate + alert

Gaps: Reconnaissance and Actions on Objectives have lowest detection
probability — prioritize detection engineering in these phases.
```

## SOAR Workflow Orchestration

### Orchestration Architecture

```
SOAR Platform
  |
  +-- Playbook Engine
  |     - Visual workflow editor (drag-and-drop)
  |     - Conditional branching (if/else based on enrichment results)
  |     - Parallel execution (enrich multiple IOCs simultaneously)
  |     - Loops (iterate over list of affected hosts)
  |     - Human approval gates (require analyst confirmation for destructive actions)
  |
  +-- Integration Layer
  |     - 300+ pre-built integrations (SIEM, EDR, firewall, email, ticketing)
  |     - REST API connectors for custom integrations
  |     - Bidirectional: read data from tools AND take actions
  |     - Credential vault for secure API key storage
  |
  +-- Case Management
  |     - Incident tracking with timeline and evidence
  |     - Analyst assignment and workload balancing
  |     - SLA tracking and escalation
  |     - Evidence chain of custody
  |
  +-- Reporting and Metrics
        - Playbook execution statistics
        - Mean time to respond by incident type
        - Analyst productivity metrics
        - ROI calculation (time saved through automation)
```

### Playbook Design Patterns

**Pattern 1: Triage and Enrich**
```
Trigger: SIEM alert arrives
  |
  +-- Extract observables (IPs, domains, hashes, users)
  |
  +-- Parallel enrichment:
  |     +-- VirusTotal lookup (each hash, IP, domain)
  |     +-- Threat intel platform lookup
  |     +-- CMDB lookup (affected asset details)
  |     +-- Directory lookup (user details)
  |     +-- Recent alert history for same source
  |
  +-- Score: combine enrichment results into risk score
  |
  +-- Route:
        Score > 80: escalate to Tier 2 with pre-built case
        Score 40-80: assign to Tier 1 with enrichment summary
        Score < 40: auto-close with documented rationale
```

**Pattern 2: Contain and Remediate**
```
Trigger: analyst confirms true positive
  |
  +-- Approval gate: analyst confirms containment action
  |
  +-- Parallel containment:
  |     +-- Isolate host (EDR API)
  |     +-- Block source IP (firewall API)
  |     +-- Disable user account (Active Directory API)
  |     +-- Block file hash (endpoint policy API)
  |
  +-- Verify containment:
  |     +-- Check host isolation status
  |     +-- Confirm firewall rule applied
  |     +-- Confirm account disabled
  |
  +-- Notify:
  |     +-- SOC manager (email)
  |     +-- Asset owner (Slack/Teams)
  |     +-- Ticketing system (create incident)
  |
  +-- Schedule:
        +-- Follow-up investigation task (4 hours)
        +-- Containment review (24 hours)
```

**Pattern 3: Threat Intelligence Operationalization**
```
Trigger: new threat report published (RSS feed, TAXII poll)
  |
  +-- Parse report: extract IOCs (IPs, domains, hashes, YARA rules)
  |
  +-- Deduplicate: check if IOCs already exist in threat intel platform
  |
  +-- Retrospective search: query SIEM for historical matches
  |
  +-- If matches found:
  |     +-- Create incident
  |     +-- Assign to Tier 2
  |     +-- Include affected hosts and timeline
  |
  +-- If no matches:
  |     +-- Add IOCs to SIEM watchlist for future detection
  |     +-- Update firewall block lists
  |     +-- Update email gateway block lists
  |
  +-- Report: summarize new intelligence and actions taken
```

## Security Metrics and KPIs

### Building a Metrics Program

Effective security metrics follow the SMART criteria and align with business risk:

**Detection effectiveness metrics:**

```
MTTD (Mean Time to Detect):
  Definition: average time from initial compromise to first detection
  Calculation: sum(detection_time - compromise_time) / count(incidents)
  Target: <24 hours for critical assets, <72 hours for all assets
  Limitation: compromise time is often estimated, not precisely known

MTTA (Mean Time to Acknowledge):
  Definition: average time from alert generation to analyst assignment
  Calculation: sum(acknowledge_time - alert_time) / count(alerts)
  Target: <15 minutes for critical alerts, <1 hour for high alerts

MTTR (Mean Time to Respond):
  Definition: average time from detection to containment
  Calculation: sum(containment_time - detection_time) / count(incidents)
  Target: <4 hours for critical, <24 hours for high

Dwell time:
  Definition: total time adversary has access to environment
  Calculation: eradication_time - initial_compromise_time
  Industry benchmark: 16 days median (Mandiant M-Trends 2024)
  Target: <48 hours
```

**Operational efficiency metrics:**

```
Alert volume: total alerts generated per day/week/month
  Trend analysis: increasing volume may indicate poor tuning or new threats

True positive rate: alerts confirmed as actual incidents / total alerts
  Target: >30% (below 30% indicates excessive false positives)
  Formula: TP / (TP + FP)

False positive rate: alerts confirmed as benign / total alerts
  Target: <50% aggregate, <30% per individual rule
  Action: disable or retune rules with >70% FP rate

Alert-to-incident ratio: how many alerts create an actual incident
  Typical: 1000:1 to 100:1
  Lower ratio = better detection fidelity

Analyst throughput: alerts processed per analyst per shift
  Typical: 20-40 alerts per 8-hour shift for Tier 1
  If consistently >50: analyst is rushing, quality suffers
  If consistently <15: analyst may be underutilized or alerts are too complex
```

**Coverage metrics:**

```
ATT&CK coverage: % of relevant techniques with at least one detection rule
  Measurement: map all active SIEM rules to ATT&CK technique IDs
  Calculate: covered_techniques / total_relevant_techniques
  Target: >70% for commonly used techniques in your threat model

Log source coverage: % of critical assets sending logs to SIEM
  Measurement: compare CMDB asset list against SIEM data sources
  Target: 100% of critical assets, >90% of all assets

Vulnerability scan coverage: % of assets scanned in last 30 days
  Target: 100% of internet-facing, >95% of internal
```

## Alert Fatigue Analysis

### Causes of Alert Fatigue

Alert fatigue occurs when analysts are overwhelmed by alert volume, leading to missed true positives. Root causes:

**Volume-driven fatigue:**
- Too many correlation rules producing low-value alerts
- Default vendor rules deployed without tuning
- Overlapping rules that fire on the same event
- No alert deduplication or grouping

**Quality-driven fatigue:**
- High false positive rates on specific rules
- Missing context in alerts (analyst must manually gather information)
- No severity differentiation (all alerts appear equally urgent)
- Alerts without actionable guidance (what should the analyst do?)

**Process-driven fatigue:**
- No triage criteria (analyst does not know when to escalate vs close)
- Manual enrichment required for every alert
- No feedback loop (FP reports do not lead to rule tuning)
- Alert routing not aligned with analyst expertise

### Quantifying Alert Fatigue

```
Fatigue indicators:
  - Increasing average time to acknowledge (MTTA trending up)
  - Decreasing investigation depth (fewer enrichment steps per alert)
  - Increasing auto-close rate without investigation
  - Increasing analyst turnover
  - Missed true positives that were in the alert queue

Measurement approach:
  1. Sample 100 alerts per week
  2. Classify each as: TP (actionable), FP (not actionable), BTP (real but expected)
  3. Track the ratio over time
  4. If FP + BTP consistently >70%, fatigue risk is high

Remediation:
  - Disable rules with >80% FP rate
  - Implement auto-enrichment to reduce manual investigation time
  - Create tiered alert queues (critical/high/medium/low)
  - Implement auto-close for known benign true positives
  - Conduct weekly rule review: top 10 noisiest rules
  - Target: <500 actionable alerts per day per SOC team
```

## ML-Based Anomaly Detection in SIEM

### Approaches

**User and Entity Behavior Analytics (UEBA):**

UEBA builds behavioral baselines for users and entities (hosts, applications, network devices) and flags deviations.

```
Baseline features per user:
  - Login times (histogram of hourly login frequency)
  - Login locations (set of source IPs/geolocations)
  - Systems accessed (set of destination hosts/applications)
  - Data volume transferred (daily distribution)
  - Command patterns (frequency of privileged operations)
  - Peer group behavior (compare user to similar role/department)

Anomaly detection algorithms:
  1. Statistical: z-score on each feature; flag if >3 sigma
  2. Isolation Forest: unsupervised, efficient for high-dimensional data
  3. Local Outlier Factor (LOF): density-based, good for clustered data
  4. Autoencoder: neural network trained on normal behavior;
     high reconstruction error = anomaly

Example UEBA detection:
  Normal: jsmith logs in 08:00-18:00 from office IP, accesses 3 apps
  Anomaly: jsmith logs in at 03:00 from foreign IP, accesses 12 apps,
           downloads 2 GB from file server
  Risk score: 95 (multiple high-deviation features simultaneously)
```

**Network Traffic Analysis (NTA):**

```
Features extracted from network flows:
  - Bytes per flow (distribution)
  - Packets per flow (distribution)
  - Connection duration (distribution)
  - Inter-arrival time (periodicity detection for beaconing)
  - Protocol distribution (% HTTP, HTTPS, DNS, SSH, other)
  - Destination entropy (high = scanning, low = normal targeted access)
  - DNS query patterns (domain length, character distribution, TLD usage)

Beaconing detection algorithm:
  1. For each source IP, group outbound connections by destination
  2. Calculate inter-arrival time distribution per destination
  3. Compute Jitter = stddev(inter_arrival_times) / mean(inter_arrival_times)
  4. Low jitter (<0.1) + regular interval = likely beaconing
  5. High volume + low jitter + unknown destination = C2 candidate

DGA (Domain Generation Algorithm) detection:
  Feature extraction per domain:
    - Length of domain name
    - Character entropy (Shannon entropy)
    - Ratio of consonants to vowels
    - N-gram frequency (bigram/trigram distribution)
    - Presence in Alexa/Tranco top-1M list

  ML classifier (Random Forest or LSTM):
    Training data: known good domains + known DGA domains
    Output: probability that domain is algorithmically generated
    Threshold: flag if probability > 0.85
```

### Challenges of ML in Security Operations

**False positive management:**
- ML models produce probabilistic scores, not binary verdicts
- Thresholds must be tuned per environment; default thresholds rarely work
- New models generate a flood of alerts during the learning period
- Legitimate behavior changes (new hire, role change, project shift) trigger false positives

**Model maintenance:**
- Baseline drift: user behavior changes over time, model must adapt
- Retraining frequency: weekly or monthly depending on environment volatility
- Feature engineering: requires deep understanding of both security and data science
- Adversarial robustness: attackers can slowly shift behavior to retrain the baseline (boiling frog attack)

**Operational integration:**
- ML alerts must include explanation of why the anomaly was flagged
- Analysts need to understand the features that contributed to the score
- Integration with existing triage workflows (ML alerts treated same as rule-based alerts)
- Feedback loop: analyst classification (TP/FP) must feed back into model training

### Recommended ML Integration Strategy

```
Phase 1: Shadow mode (weeks 1-4)
  - Deploy ML model alongside existing rules
  - ML generates scores but does NOT create alerts
  - Analysts review ML scores retrospectively
  - Tune thresholds based on TP/FP analysis

Phase 2: Advisory mode (weeks 5-12)
  - ML adds risk scores to existing rule-based alerts
  - High ML score + rule match = elevated priority
  - ML-only detections (no matching rule) flagged for review in batch
  - Continue tuning based on analyst feedback

Phase 3: Detection mode (week 13+)
  - ML generates independent alerts for high-confidence anomalies
  - ML risk scores modify alert priority across all alert types
  - Automated enrichment triggered by ML alerts
  - Regular model performance reviews (monthly)

Phase 4: Autonomous mode (mature SOCs only)
  - ML triggers automated containment for very high confidence detections
  - Human approval gate only for destructive actions
  - Continuous model retraining with feedback loop
  - A/B testing of model versions in production
```

## See Also

- endpoint-security, cloud-security

## References

- [NIST SP 800-61 Rev 2 — Computer Security Incident Handling Guide](https://csrc.nist.gov/publications/detail/sp/800-61/rev-2/final)
- [NIST SP 800-150 — Guide to Cyber Threat Information Sharing](https://csrc.nist.gov/publications/detail/sp/800-150/final)
- [Diamond Model of Intrusion Analysis (Caltagirone, Pendergast, Betz)](https://apps.dtic.mil/sti/pdfs/ADA586960.pdf)
- [Lockheed Martin Cyber Kill Chain](https://www.lockheedmartin.com/en-us/capabilities/cyber/cyber-kill-chain.html)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [MITRE D3FEND — Defensive Technique Knowledge Base](https://d3fend.mitre.org/)
- [Mandiant M-Trends Annual Report](https://www.mandiant.com/m-trends)
- [SANS SOC Survey](https://www.sans.org/white-papers/)
- [Elastic Common Schema (ECS)](https://www.elastic.co/guide/en/ecs/current/index.html)
- [OCSF — Open Cybersecurity Schema Framework](https://schema.ocsf.io/)
- [Gartner SOAR Market Guide](https://www.gartner.com/reviews/market/security-orchestration-automation-and-response-solutions)
- [Carnegie Mellon SEI — SOC Maturity Model](https://resources.sei.cmu.edu/library/asset-view.cfm?assetid=546588)
