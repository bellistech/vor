# Risk Management

> Identify, analyze, evaluate, and treat risks to organizational assets using quantitative and qualitative methods, frameworks, and continuous monitoring.

## Risk Assessment Process

```
# Risk assessment lifecycle
# 1. System characterization — define scope, assets, boundaries
# 2. Threat identification — enumerate threat sources and events
# 3. Vulnerability identification — discover weaknesses
# 4. Control analysis — evaluate existing/planned safeguards
# 5. Likelihood determination — estimate probability of exploitation
# 6. Impact analysis — determine consequence severity
# 7. Risk determination — combine likelihood × impact
# 8. Control recommendations — propose risk treatments
# 9. Results documentation — produce risk register/report

# Key inputs at each stage
# Assets → what are we protecting?
# Threats → what could go wrong?
# Vulnerabilities → where are the gaps?
# Controls → what's already in place?
# Likelihood × Impact → how bad and how likely?
```

## Risk Components

```
# Risk = Threat × Vulnerability × Impact
# (conceptual formula — not literal multiplication)

# Threat source — entity or event with potential to cause harm
#   Natural: earthquake, flood, hurricane, fire
#   Human: insider threat, external attacker, social engineering
#   Environmental: power failure, HVAC failure, water damage
#   Technical: hardware failure, software bug, network outage

# Vulnerability — weakness that can be exploited
#   Technical: unpatched software, misconfiguration, weak crypto
#   Administrative: lack of policy, poor training, no background checks
#   Physical: unlocked doors, no cameras, poor access control

# Asset — resource with value to the organization
#   Tangible: servers, network equipment, facilities
#   Intangible: data, intellectual property, reputation, brand
```

## Quantitative Risk Analysis

```
# Asset Value (AV)
# Total value of the asset (replacement cost, revenue impact, etc.)

# Exposure Factor (EF)
# Percentage of asset loss from a single incident (0.0 to 1.0)
# Example: fire destroys 40% of data center → EF = 0.40

# Single Loss Expectancy (SLE)
# SLE = AV × EF
# Example: AV = $500,000, EF = 0.40 → SLE = $200,000

# Annualized Rate of Occurrence (ARO)
# Expected frequency of the threat per year
# Example: flood occurs once every 10 years → ARO = 0.1
# Example: malware incident 3 times per year → ARO = 3.0

# Annualized Loss Expectancy (ALE)
# ALE = SLE × ARO
# Example: SLE = $200,000, ARO = 0.1 → ALE = $20,000/year
# Example: SLE = $50,000, ARO = 3.0 → ALE = $150,000/year

# Total Risk (portfolio)
# Total ALE = sum of individual ALEs for all threat-asset pairs
```

### Cost-Benefit Analysis

```
# Value of safeguard = (ALE_before) - (ALE_after) - (Annual Cost of Safeguard)

# Example:
# ALE before control:  $150,000
# ALE after control:   $30,000
# Annual control cost: $50,000
# Value = $150,000 - $30,000 - $50,000 = $70,000 (positive = justified)

# Return on Security Investment (ROSI)
# ROSI = (ALE_before - ALE_after - Control_cost) / Control_cost × 100%
# ROSI = ($150,000 - $30,000 - $50,000) / $50,000 × 100% = 140%

# Rule: implement control only if value > 0 (or ROSI > 0%)
# Never spend more on protection than the asset is worth
```

## Qualitative Risk Analysis

### Risk Matrix

```
# Likelihood levels (1–5)
# 1 = Rare        (< 1% chance per year)
# 2 = Unlikely    (1–10%)
# 3 = Possible    (11–50%)
# 4 = Likely      (51–90%)
# 5 = Almost Certain (> 90%)

# Impact levels (1–5)
# 1 = Negligible  (minimal disruption)
# 2 = Minor       (some impact, easily recoverable)
# 3 = Moderate    (significant impact, recovery possible)
# 4 = Major       (severe impact, difficult recovery)
# 5 = Critical    (catastrophic, existential threat)

# Risk score = Likelihood × Impact
#
#              Impact →
#  Likelihood  1   2   3    4    5
#  ↓
#  5           5  10  15   20   25   ← Almost Certain
#  4           4   8  12   16   20   ← Likely
#  3           3   6   9   12   15   ← Possible
#  2           2   4   6    8   10   ← Unlikely
#  1           1   2   3    4    5   ← Rare

# Risk ratings
# 1–4   = Low (accept)
# 5–9   = Medium (monitor, consider mitigation)
# 10–16 = High (mitigate)
# 17–25 = Critical (immediate action required)
```

### Delphi Method

```
# Anonymous expert consensus for risk estimation
# 1. Select panel of subject matter experts
# 2. Each expert independently rates risks
# 3. Compile and share results anonymously
# 4. Experts revise ratings after seeing others' input
# 5. Repeat until consensus reached (usually 2–4 rounds)
# Advantage: removes groupthink and dominant personality bias
```

## Risk Treatment Options

```
# Avoid — eliminate the risk entirely
#   Stop the activity, remove the asset, exit the market
#   Example: don't store credit card data → no PCI risk

# Mitigate (Reduce) — lower likelihood or impact
#   Apply controls, patches, training, monitoring
#   Example: install firewall, deploy IDS, encrypt data

# Transfer (Share) — shift risk to a third party
#   Insurance, outsourcing, SLAs, contractual clauses
#   Example: cyber liability insurance, managed SOC

# Accept — acknowledge and tolerate the risk
#   Risk falls within appetite, cost of control exceeds potential loss
#   Requires formal management sign-off and documentation
#   Example: accept downtime risk for non-critical dev environment
```

## Risk Appetite, Tolerance, and Capacity

```
# Risk appetite — amount of risk the organization is willing to pursue
#   Set by board/executive leadership
#   Strategic-level statement
#   Example: "We accept moderate risk to achieve market growth"

# Risk tolerance — acceptable deviation from risk appetite
#   Operational-level boundaries
#   Example: "System downtime must not exceed 4 hours per quarter"

# Risk capacity — maximum risk the org can absorb before failure
#   Hard limit determined by resources, capital, regulatory requirements
#   Example: "A $10M loss would trigger insolvency"

# Relationship: Capacity > Appetite > Tolerance
# Risk threshold — the specific trigger point for action
```

## Risk Register

```
# Essential fields in a risk register:
# - Risk ID (unique identifier)
# - Risk description (what could happen)
# - Category (operational, strategic, financial, compliance)
# - Risk owner (accountable individual)
# - Threat source and vulnerability
# - Likelihood rating (1–5 or quantitative probability)
# - Impact rating (1–5 or quantitative dollar value)
# - Inherent risk score (before controls)
# - Existing controls (current safeguards)
# - Residual risk score (after controls)
# - Risk treatment plan (avoid/mitigate/transfer/accept)
# - Target risk level (desired residual risk)
# - Status (open, in treatment, closed, accepted)
# - Review date (next reassessment)

# Residual risk = Inherent risk - Control effectiveness
# If residual risk > risk tolerance → additional treatment required
# Management must formally accept all residual risk
```

## Risk Frameworks

### NIST SP 800-30

```
# NIST Risk Assessment Guide
# 1. Prepare for assessment — scope, assumptions, constraints
# 2. Conduct assessment:
#    a. Identify threat sources and events
#    b. Identify vulnerabilities and predisposing conditions
#    c. Determine likelihood of occurrence
#    d. Determine magnitude of impact
#    e. Determine risk (likelihood × impact)
# 3. Communicate results — risk report to stakeholders
# 4. Maintain assessment — ongoing monitoring and updates

# NIST RMF (800-37) steps:
# 1. Categorize (FIPS 199)
# 2. Select controls (800-53)
# 3. Implement controls
# 4. Assess controls (800-53A)
# 5. Authorize system
# 6. Monitor continuously
```

### ISO 31000

```
# International standard for risk management
# Principles: integrated, structured, customized, inclusive,
#             dynamic, best available info, human/cultural factors,
#             continual improvement
# Framework: leadership → integration → design → implementation →
#            evaluation → improvement
# Process: scope/context → risk assessment (identify → analyze →
#          evaluate) → risk treatment → monitoring → communication
```

### FAIR (Factor Analysis of Information Risk)

```
# Quantitative risk analysis framework
# Risk = Loss Event Frequency (LEF) × Loss Magnitude (LM)
#
# LEF decomposition:
#   LEF = Threat Event Frequency (TEF) × Vulnerability (Vuln)
#   TEF = Contact Frequency × Probability of Action
#   Vuln = f(Threat Capability, Resistance Strength)
#
# LM decomposition:
#   Primary Loss = Productivity + Response + Replacement
#   Secondary Loss = Secondary LEF × Secondary LM
#   Secondary LM = Fines + Reputation + Competitive Advantage
```

### OCTAVE

```
# Operationally Critical Threat, Asset, and Vulnerability Evaluation
# Developed by CMU/SEI
# Self-directed assessment — organization's own team conducts it
# Three phases:
#   1. Build asset-based threat profiles (organizational view)
#   2. Identify infrastructure vulnerabilities (technological view)
#   3. Develop security strategy and plans (risk analysis)
# OCTAVE Allegro: streamlined version for rapid assessment
# Focus: critical information assets and their containers
```

### CRAMM

```
# CCTA Risk Analysis and Management Method (UK government origin)
# Three stages:
#   1. Asset identification and valuation
#   2. Threat and vulnerability assessment
#   3. Countermeasure selection and recommendation
# Uses a scoring matrix for threats (1–5) and vulnerabilities (1–3)
# Automated tool support for control selection
```

## Control Selection

```
# Control categories:
# Preventive — stop incidents before they occur
#   Firewalls, access control, encryption, security training
# Detective — identify incidents during or after occurrence
#   IDS/IPS, log monitoring, audit trails, SIEM
# Corrective — minimize impact and restore operations
#   Incident response, backups, patches, disaster recovery
# Deterrent — discourage potential attackers
#   Warning banners, security cameras, penalties
# Compensating — alternative controls when primary isn't feasible
#   Additional monitoring when patching isn't possible
# Recovery — restore systems to normal operation
#   Backup restoration, failover, DR sites

# Control types:
# Administrative — policies, procedures, training, background checks
# Technical (Logical) — software/hardware controls, encryption, ACLs
# Physical — locks, fences, guards, environmental controls
```

## Continuous Monitoring

```
# NIST SP 800-137 — continuous monitoring strategy
# Key activities:
#   Define monitoring strategy (metrics, frequencies, thresholds)
#   Establish metrics and measures
#   Implement monitoring program
#   Analyze data and report findings
#   Respond to findings (risk response)
#   Review and update strategy

# Key risk indicators (KRIs)
#   Leading indicators: predict future risk events
#     Example: number of unpatched critical vulnerabilities
#   Lagging indicators: measure past risk events
#     Example: number of incidents last quarter
#   Current indicators: real-time risk posture
#     Example: percentage of systems compliant with baseline

# Reassessment triggers:
#   Significant system changes
#   New threat intelligence
#   After security incidents
#   Regulatory/compliance changes
#   Periodic schedule (annually at minimum)
```

## Tips

- Quantitative analysis provides dollar values but requires reliable data; use qualitative when data is sparse.
- Risk appetite must be defined by senior leadership before the risk assessment begins.
- Residual risk must always be formally accepted by management in writing.
- FAIR is the best framework for translating risk into financial terms for executive communication.
- A risk register is a living document — review and update it at least quarterly.
- Never spend more on a control than the annual loss it prevents.

## See Also

- security-governance, threat-modeling, incident-response, bcp-drp, security-operations

## References

- [NIST SP 800-30 Rev 1 — Guide for Conducting Risk Assessments](https://csrc.nist.gov/publications/detail/sp/800-30/rev-1/final)
- [NIST SP 800-37 Rev 2 — Risk Management Framework](https://csrc.nist.gov/publications/detail/sp/800-37/rev-2/final)
- [NIST SP 800-137 — Continuous Monitoring](https://csrc.nist.gov/publications/detail/sp/800-137/final)
- [ISO 31000:2018 — Risk Management Guidelines](https://www.iso.org/standard/65694.html)
- [FAIR Institute — Factor Analysis of Information Risk](https://www.fairinstitute.org/)
- [OCTAVE — CMU SEI](https://resources.sei.cmu.edu/library/asset-view.cfm?assetid=51546)
- [ISACA — Risk IT Framework](https://www.isaca.org/resources/it-risk)
