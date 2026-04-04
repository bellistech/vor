# Threat Modeling (Systematic Threat Identification and Risk Assessment)

Threat modeling is a structured approach to identifying, quantifying, and prioritizing
potential threats to a system. By analyzing data flows, trust boundaries, and attack
surfaces before code is written, teams can design security controls that address the
most impactful risks.

---

## STRIDE Threat Classification

```bash
# STRIDE — systematic threat identification per component

# S — Spoofing (violates Authentication)
# Can an attacker impersonate a user or service?
# Mitigation: strong auth, mutual TLS, signed tokens

# T — Tampering (violates Integrity)
# Can an attacker modify data in transit or at rest?
# Mitigation: digital signatures, MACs, input validation

# R — Repudiation (violates Non-repudiation)
# Can an attacker deny performing an action?
# Mitigation: audit logging, digital signatures, timestamps

# I — Information Disclosure (violates Confidentiality)
# Can an attacker access unauthorized data?
# Mitigation: encryption, access controls, data classification

# D — Denial of Service (violates Availability)
# Can an attacker exhaust resources?
# Mitigation: rate limiting, redundancy, CDN

# E — Elevation of Privilege (violates Authorization)
# Can an attacker gain unauthorized access?
# Mitigation: least privilege, RBAC, sandboxing
```

### Applying STRIDE per Component

```bash
# For each component, ask all 6 STRIDE questions
# Example: REST API /api/v1/users/{id}
# S: unauthenticated access?      → auth middleware
# T: parameter tampering?          → input validation
# R: unlogged access attempts?     → audit middleware
# I: IDOR / data leakage?         → authorization check
# D: resource exhaustion?          → rate limiter
# E: privilege escalation?         → role-based access
```

---

## DREAD Risk Scoring

```bash
# Semi-quantitative scoring (1-10 per dimension)
# D — Damage Potential:    1=trivial, 10=full compromise
# R — Reproducibility:     1=rare race, 10=every time
# E — Exploitability:      1=custom exploit, 10=URL bar
# A — Affected Users:      1=single user, 10=all users
# D — Discoverability:     1=source audit, 10=public knowledge

# Score = (D + R + E + A + D) / 5
# High: > 7, Medium: 4-7, Low: < 4

# Example: SQL injection in login form
# D:9 R:10 E:8 A:10 D:7 → Score: 8.8 (HIGH)
```

### Risk Matrix

```bash
# Likelihood (1-5) x Impact (1-5) = Risk Score
#          Impact:  1    2    3    4    5
# Like. 5  |  5 | 10 | 15 | 20 | 25 |
#       4  |  4 |  8 | 12 | 16 | 20 |
#       3  |  3 |  6 |  9 | 12 | 15 |
#       2  |  2 |  4 |  6 |  8 | 10 |
#       1  |  1 |  2 |  3 |  4 |  5 |
# 1-5: Low (accept), 6-12: Medium, 13-19: High, 20-25: Critical
```

---

## Data Flow Diagrams

### DFD Elements

```bash
# Four element types:
# External Entity (rectangle) — users, external services
# Process (circle)            — API handlers, workers
# Data Store (parallel lines) — databases, caches, files
# Data Flow (arrow)           — labeled with protocol + data type
```

### Trust Boundaries

```bash
# Trust boundaries separate zones with different privilege levels
# Every flow crossing a boundary is an attack surface

# Common boundaries:
# Internet ↔ DMZ (firewall, WAF)
# DMZ ↔ Internal network (API gateway)
# Application ↔ Database (connection pool)
# Container ↔ Host (runtime)
# Cloud VPC ↔ Public internet (security groups)

# For each crossing, document:
# - What data crosses?
# - Is it encrypted in transit?
# - Is the sender authenticated?
# - Is the data validated on receipt?
```

### DFD Levels

```bash
# Level 0: Context diagram — single process, all external entities
# Level 1: System diagram — major subsystems, data stores, trust boundaries
# Level 2: Component diagram — individual components, internal flows

# Example Level 1:
# [Browser] --HTTPS--> (API Gateway) --gRPC--> (Auth Service) --> [User DB]
#                          |--gRPC--> (Order Service) --> [Order DB]
#                          |--gRPC--> (Payment) --HTTPS--> [Stripe]
```

---

## Attack Trees

```bash
# Root = attacker goal, branches = decomposition
# AND nodes: all children must succeed
# OR nodes:  any child suffices

# Example: Steal credentials (OR)
# ├── SQL injection (cost: $500, prob: 0.7)
# ├── Phishing (AND): email ($200) + hosting ($300) = $500, prob: 0.12
# └── Session hijack (OR):
#     ├── XSS ($300, prob: 0.4)
#     └── Network sniffing ($100, prob: 0.6) ← if no HTTPS
#
# Cheapest path: $100 (sniffing) → priority: enforce HTTPS

# Annotate nodes with: cost, skill, detection likelihood, time
# OR cost = min(children), AND cost = sum(children)
# OR prob = 1 - prod(1-p_i), AND prob = prod(p_i)
```

---

## PASTA (Process for Attack Simulation and Threat Analysis)

```bash
# Seven-stage risk-centric methodology

# Stage 1: Define Objectives
# Business impact, compliance, risk tolerance

# Stage 2: Define Technical Scope
# Architecture, infrastructure, technology stack

# Stage 3: Application Decomposition
# DFDs, trust boundaries, entry points, data classification

# Stage 4: Threat Analysis
# Relevant threats from CAPEC, MITRE ATT&CK, industry reports

# Stage 5: Vulnerability Analysis
# Map threats to CVEs, OWASP Top 10, static analysis findings

# Stage 6: Attack Modeling
# Attack trees, attack path simulation

# Stage 7: Risk and Impact Analysis
# Risk scores, prioritized mitigations, residual risk
```

---

## LINDDUN Privacy Threat Modeling

```bash
# Privacy-focused methodology (complement to STRIDE)

# L — Linkability:          correlating items across contexts
# I — Identifiability:      identifying subjects from data
# N — Non-repudiation:      unable to deny when they should be able to
# D — Detectability:        inferring existence of data
# D — Disclosure:           personal data exposed to unauthorized parties
# U — Unawareness:          users unaware of data collection/processing
# N — Non-compliance:       violating privacy regulations (GDPR, etc.)
```

---

## Threat Modeling Tools

### Microsoft Threat Modeling Tool

```bash
# Free DFD creation with auto-generated STRIDE threats
# Download: https://aka.ms/threatmodelingtool
# Templates: Azure, SDL generic, Medical Device, Custom
# Workflow: draw DFD → add trust boundaries → review threats → export report
```

### OWASP Threat Dragon

```bash
# Open-source web-based threat modeling
npm install -g owasp-threat-dragon
threat-dragon
# Or: https://www.threatdragon.com/
# STRIDE-per-element, JSON export, GitHub integration
```

### Threagile (Threat Modeling as Code)

```bash
# YAML-based threat modeling
docker pull threagile/threagile
docker run --rm -v $(pwd):/app/work threagile/threagile \
  -model /app/work/threagile.yaml -output /app/work/output
# Outputs: risks report, DFD, technical report
```

---

## Threat Libraries

```bash
# CAPEC — 559+ attack patterns organized by mechanism
# https://capec.mitre.org/

# CWE — 900+ software weakness types; CWE Top 25 annual list
# https://cwe.mitre.org/

# MITRE ATT&CK — 14 tactics, 200+ techniques, 600+ sub-techniques
# https://attack.mitre.org/

# OWASP Top 10 — web application risk categories
# https://owasp.org/www-project-top-ten/

# Mapping: CAPEC-66 (SQL Injection) → CWE-89 → ATT&CK T1190
```

---

## SDLC Integration

```bash
# When to threat model:
# - New feature design (before coding)
# - Architecture changes (new service/data flow)
# - New external integration
# - Compliance requirement
# - Post-incident review
# - Quarterly review of existing models

# Lightweight agile process:
# Time-box: 60-90 minutes per feature
# Participants: developer, architect, security engineer
# Output: 3-5 prioritized threats with mitigation tickets
# Store models in version control alongside code
```

---

## Tips

- Start with a Level 0 DFD to define boundaries before diving into details
- Focus STRIDE on trust boundary crossings — that is where most threats manifest
- DREAD scoring is subjective; have multiple people score and average
- Attack trees are most valuable for high-risk scenarios, not every threat
- PASTA stages 3, 4, and 7 are the most critical for agile abbreviation
- LINDDUN is essential for GDPR-regulated systems — STRIDE misses privacy threats
- Use MITRE ATT&CK to validate models against real adversary behavior
- Review models after every incident — missing threats indicate model gaps
- Store threat models in version control so they evolve with the system
- Threat modeling is most effective when done by the team that builds the system

---

## See Also

- mitre-attack
- incident-response
- zero-trust

## References

- [Microsoft STRIDE](https://learn.microsoft.com/en-us/azure/security/develop/threat-modeling-tool-threats)
- [Microsoft Threat Modeling Tool](https://aka.ms/threatmodelingtool)
- [OWASP Threat Dragon](https://www.threatdragon.com/)
- [Threagile](https://threagile.io/)
- [CAPEC](https://capec.mitre.org/)
- [MITRE ATT&CK](https://attack.mitre.org/)
- [LINDDUN](https://linddun.org/)
- [OWASP ASVS](https://owasp.org/www-project-application-security-verification-standard/)
