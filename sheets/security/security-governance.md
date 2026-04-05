# Security Governance

Security governance frameworks, policies, organizational structure, risk management, compliance, metrics, maturity assessment.

## Security Governance Framework

### Governance Structure

```
Board of Directors / Executive Leadership
    ↓ (sets risk appetite, approves policy)
Security Steering Committee
    ↓ (cross-functional oversight)
CISO / Head of Security
    ↓ (strategy, budget, reporting)
├── Security Operations (SOC)
├── Security Architecture
├── Security Engineering
├── Governance, Risk & Compliance (GRC)
├── Identity & Access Management
├── Application Security
└── Security Awareness & Training
```

### Governance vs Management vs Operations

```
GOVERNANCE  — What should we do? (Direction, oversight, accountability)
  ↓
MANAGEMENT  — How do we do it? (Planning, organizing, resourcing)
  ↓
OPERATIONS  — Do it. (Daily execution, monitoring, response)

# Governance: "We must protect customer data per GDPR"
# Management: "Implement encryption, access controls, DLP by Q2"
# Operations: "Deploy AES-256 on database, configure DLP rules, monitor alerts"
```

## Organizational Security Structure

### CISO Role and Reporting

```
# CISO reporting options (each has trade-offs):

# Reports to CEO — strongest independence
CEO → CISO
  Pro: Direct board access, budget authority
  Con: Rare, CEO has many direct reports

# Reports to CIO — most common
CEO → CIO → CISO
  Pro: Close to IT operations
  Con: Potential conflict of interest (security vs speed)

# Reports to CRO — risk-focused
CEO → CRO → CISO
  Pro: Aligned with enterprise risk
  Con: May be distant from technology

# Reports to General Counsel — compliance-focused
CEO → GC → CISO
  Pro: Legal and regulatory alignment
  Con: May underweight technical security

# Key CISO responsibilities:
# - Security strategy and roadmap
# - Risk management program
# - Policy development and enforcement
# - Security architecture approval
# - Incident response oversight
# - Regulatory compliance
# - Security awareness program
# - Budget management
# - Board-level reporting
# - Third-party risk management
```

### Security Steering Committee

```
# Composition:
# - CISO (chair)
# - CIO / CTO
# - CFO representative
# - Legal / General Counsel
# - HR representative
# - Business unit leaders
# - Compliance officer
# - Risk management

# Responsibilities:
# - Review and approve security policies
# - Prioritize security investments
# - Review risk assessment results
# - Approve risk treatment plans
# - Monitor security program effectiveness
# - Ensure regulatory compliance alignment
# - Review major security incidents
# - Approve security exceptions

# Cadence: Quarterly minimum, monthly preferred
```

## Security Policies

### Policy Hierarchy

```
POLICY (mandatory, approved by executives)
  "All data must be classified according to sensitivity"
  ↓
STANDARD (mandatory, specific requirements)
  "Passwords must be minimum 14 characters with MFA"
  ↓
GUIDELINE (recommended, best practices)
  "Consider using a password manager for unique passwords"
  ↓
PROCEDURE (step-by-step instructions)
  "To reset MFA: 1. Navigate to... 2. Click... 3. Verify..."
```

### Core Security Policies

```
# Information Security Policy (master policy)
# - Scope: all employees, contractors, third parties
# - Data classification scheme
# - Acceptable use requirements
# - Incident reporting obligations
# - Compliance requirements
# - Enforcement and sanctions

# Acceptable Use Policy (AUP)
# - Authorized use of company systems
# - Personal use limitations
# - Prohibited activities
# - Monitoring notice
# - Social media guidelines
# - BYOD requirements

# Data Classification Policy
# PUBLIC        — freely shareable (marketing materials)
# INTERNAL      — internal only, low impact if disclosed
# CONFIDENTIAL  — restricted access, significant impact
# RESTRICTED    — highly sensitive (PII, financial, health)

# Access Control Policy
# - Least privilege principle
# - Need-to-know basis
# - Role-based access control (RBAC)
# - Privileged access management
# - Access review cadence (quarterly/annual)
# - Separation of duties
# - Termination procedures

# Incident Response Policy
# - Incident classification (P1-P4)
# - Notification requirements (internal, regulatory, public)
# - Escalation matrix
# - Evidence preservation
# - Communication plan
# - Post-incident review requirements

# Change Management Policy
# - Change classification (standard, normal, emergency)
# - CAB approval requirements
# - Testing requirements
# - Rollback procedures
# - Change freeze windows

# Business Continuity Policy
# - BIA requirements
# - RTO/RPO definitions per system tier
# - DR testing cadence
# - Crisis communication plan
# - Succession planning
```

### Policy Lifecycle

```
1. DRAFT      — Author creates based on requirements
2. REVIEW     — Stakeholders review and comment
3. APPROVE    — Executive/committee approval
4. PUBLISH    — Distribute, train, acknowledge
5. ENFORCE    — Monitor compliance, handle violations
6. REVIEW     — Annual review cycle (minimum)
7. UPDATE     — Revise based on changes
   ↓
   Back to step 2

# Policy metadata to track:
# - Version number
# - Effective date
# - Review date (next scheduled review)
# - Owner (responsible executive)
# - Author (writer)
# - Approver (signing authority)
# - Classification
# - Scope
# - Related policies/standards
# - Change history
```

## Security Awareness Program

### Program Elements

```
# Training Types:
# - New hire orientation (security module)
# - Annual security awareness training (all employees)
# - Role-based training (developers, admins, executives)
# - Phishing simulations (monthly/quarterly)
# - Incident response tabletop exercises (annual)
# - Compliance-specific training (PCI, HIPAA, GDPR)

# Topics to Cover:
# - Phishing and social engineering
# - Password hygiene and MFA
# - Data handling and classification
# - Physical security
# - Remote work security
# - Incident reporting
# - Insider threat awareness
# - Mobile device security
# - Clean desk policy
# - Secure software development (for developers)

# Measurement:
# - Training completion rates
# - Phishing simulation click rates (target: <5%)
# - Incident self-reporting rates
# - Security knowledge assessment scores
# - Policy acknowledgment rates
# - Time to complete training
```

## Risk Appetite and Tolerance

### Risk Appetite Framework

```
# Risk Appetite: Amount of risk the organization is willing to accept
#                to achieve objectives (set by board)
#
# Risk Tolerance: Acceptable variation from risk appetite
#                 (operational boundaries)
#
# Risk Capacity: Maximum risk the organization can absorb
#                before failure

# Risk Appetite Statement Example:
# "We accept moderate risk in pursuit of market growth.
#  We have zero appetite for:
#  - Regulatory non-compliance
#  - Customer data breaches
#  - Operational downtime >4 hours
#  We accept calculated risk for:
#  - New technology adoption (moderate)
#  - Market expansion (moderate-high)
#  - Innovation initiatives (high)"

# Risk Tolerance Levels:
# LOW    — minimal deviation from appetite (regulated areas)
# MEDIUM — some deviation acceptable with monitoring
# HIGH   — significant deviation acceptable (innovation)

# Quantitative example:
# Risk appetite: Annual loss expectation (ALE) < $5M
# Risk tolerance: ALE between $5M-$7M triggers review
# Risk capacity: ALE > $10M threatens business viability
```

## Security Metrics and KPIs

### Operational Metrics

```
# Vulnerability Management
# - Mean time to patch critical vulnerabilities (target: <72 hours)
# - Percentage of systems patched within SLA
# - Number of open critical/high vulnerabilities
# - Vulnerability scan coverage percentage

# Incident Response
# - Mean time to detect (MTTD) — target: <24 hours
# - Mean time to respond (MTTR) — target: <4 hours
# - Mean time to contain (MTTC) — target: <8 hours
# - Number of incidents by severity
# - Incident recurrence rate

# Access Management
# - Percentage of accounts with MFA enabled
# - Number of orphaned accounts (target: 0)
# - Average time to provision/deprovision access
# - Privileged account ratio (target: <5% of total)
# - Access review completion rate

# Security Awareness
# - Phishing simulation click rate (target: <5%)
# - Training completion rate (target: >95%)
# - Security incident self-report rate (higher is better)

# Compliance
# - Audit findings (open/closed ratio)
# - Policy exception count and trend
# - Control effectiveness scores
# - Third-party risk assessment completion rate
```

### Board-Level Reporting

```
# Executive Dashboard — Focus on risk posture, trend, and business impact
#
# Include:
# 1. Overall risk score (heat map or index)
# 2. Top 5 risks with treatment status
# 3. Key metric trends (quarter-over-quarter)
# 4. Major incident summary
# 5. Compliance status (regulatory deadlines)
# 6. Budget utilization and ROI
# 7. Benchmark comparison (industry peers)
#
# Reporting cadence:
# - Board: Quarterly
# - Executive committee: Monthly
# - Security steering committee: Monthly
# - Operations: Weekly/Daily
#
# Language: Business risk, not technical jargon
# "23% of critical systems have unpatched vulnerabilities
#  exposing the organization to potential data breach
#  with estimated financial impact of $2-5M"
# NOT: "47 CVEs remain unpatched across 12 servers"
```

## Regulatory Compliance Mapping

### Common Framework Mapping

```
# Requirement               NIST CSF    ISO 27001   SOC 2       PCI DSS
# ─────────────────────────────────────────────────────────────────────
# Access control             PR.AC       A.9         CC6.1       Req 7-8
# Encryption                 PR.DS       A.10        CC6.1       Req 3-4
# Logging/monitoring         DE.CM       A.12        CC7.2       Req 10
# Incident response          RS.RP       A.16        CC7.3       Req 12.10
# Vulnerability mgmt         ID.RA       A.12        CC7.1       Req 6.1
# Awareness training         PR.AT       A.7         CC1.4       Req 12.6
# Third-party management     ID.SC       A.15        CC9.2       Req 12.8
# Change management          PR.IP       A.12        CC8.1       Req 6.4
# Business continuity        PR.IP       A.17        A1.2        Req 12.10
# Data classification        ID.AM       A.8         CC6.1       Req 9

# Benefits of mapping:
# - Avoid duplicate controls (one control satisfies multiple frameworks)
# - Identify gaps (requirement not covered by any control)
# - Streamline audit preparation
# - Reduce compliance fatigue
```

## Security Program Maturity (CMM)

### Capability Maturity Model for Security

```
# Level 1: INITIAL (Ad hoc)
# - No formal security program
# - Reactive, individuals doing what they think is right
# - No documented policies
# - Hero-dependent security

# Level 2: MANAGED (Repeatable)
# - Basic policies exist
# - Some processes documented
# - Security roles defined
# - Inconsistent implementation across teams

# Level 3: DEFINED (Proactive)
# - Formal security program
# - Policies, standards, procedures documented
# - Security integrated into SDLC
# - Regular risk assessments
# - Awareness training program
# - Metrics being collected

# Level 4: QUANTITATIVELY MANAGED (Measured)
# - Metrics-driven decisions
# - Continuous monitoring
# - Automated compliance checks
# - Predictive risk analytics
# - Benchmarking against peers
# - Security integrated into business decisions

# Level 5: OPTIMIZING (Adaptive)
# - Continuous improvement culture
# - Advanced threat intelligence integration
# - Proactive threat hunting
# - Innovation in security controls
# - Industry-leading practices
# - Security as competitive advantage

# Most organizations: Level 2-3
# Target for regulated industries: Level 3-4
# World-class security programs: Level 4-5
```

## Information Security Management System (ISMS)

### ISO 27001 ISMS Structure

```
# Plan-Do-Check-Act (PDCA) cycle:
#
# PLAN:
# - Define ISMS scope
# - Establish security policy
# - Conduct risk assessment
# - Select controls (Annex A)
# - Produce Statement of Applicability (SoA)
# - Create risk treatment plan
#
# DO:
# - Implement risk treatment plan
# - Implement selected controls
# - Define metrics and monitoring
# - Conduct awareness training
# - Manage ISMS operations
#
# CHECK:
# - Monitor and measure controls
# - Conduct internal audits
# - Management review
# - Evaluate performance against objectives
# - Track nonconformities
#
# ACT:
# - Address nonconformities
# - Implement corrective actions
# - Drive continual improvement
# - Update risk assessment
# - Revise ISMS as needed

# Key documents:
# - ISMS scope statement
# - Information security policy
# - Risk assessment methodology
# - Statement of Applicability (SoA)
# - Risk treatment plan
# - Security objectives
# - Competence evidence (training records)
# - Operational planning documents
# - Monitoring and measurement results
# - Internal audit program and results
# - Management review minutes
# - Nonconformity and corrective action records
```

## Security Charter

### Elements

```
# Security Charter (foundational document):
#
# 1. PURPOSE
#    "Establish the authority, scope, and objectives of
#     the information security program"
#
# 2. AUTHORITY
#    "The CISO has authority to establish security policies,
#     approve security architectures, and halt deployments
#     that pose unacceptable risk"
#
# 3. SCOPE
#    "All information systems, data, personnel, and third
#     parties that access organization resources"
#
# 4. OBJECTIVES
#    - Protect confidentiality, integrity, availability
#    - Ensure regulatory compliance
#    - Enable business operations securely
#    - Manage risk to acceptable levels
#
# 5. RESPONSIBILITIES
#    - Board: oversight, risk appetite, budget
#    - CISO: strategy, policy, program management
#    - IT: implementation, operations
#    - Business units: data ownership, risk acceptance
#    - All employees: policy compliance, incident reporting
#
# 6. ACCOUNTABILITY
#    - Violations → disciplinary action per HR policy
#    - Exceptions → formal risk acceptance by data owner
#    - Escalation → security steering committee
#
# 7. REVIEW
#    - Annual review by security steering committee
#    - Approved by CEO/Board

# Signed by: CEO, CISO, CIO (minimum)
```

## Third-Party Governance

### Vendor Risk Management

```
# Third-Party Risk Lifecycle:
#
# 1. IDENTIFICATION
#    - Inventory all third parties
#    - Classify by data access and criticality
#    - Tier vendors: Critical / High / Medium / Low
#
# 2. ASSESSMENT
#    - Security questionnaire (SIG, CAIQ)
#    - SOC 2 Type II report review
#    - Penetration test results
#    - Business continuity plans
#    - Insurance coverage verification
#    - On-site assessment (critical vendors)
#
# 3. CONTRACTING
#    - Security requirements in contract
#    - Right to audit clause
#    - Breach notification requirements (<72 hours)
#    - Data handling and destruction requirements
#    - SLAs for security incidents
#    - Subcontractor restrictions
#    - Liability and indemnification
#
# 4. MONITORING
#    - Continuous monitoring (security ratings)
#    - Annual reassessment
#    - Incident tracking
#    - SLA compliance
#    - Regulatory change monitoring
#
# 5. OFFBOARDING
#    - Access revocation
#    - Data return or destruction (certificate of destruction)
#    - Knowledge transfer
#    - Contract termination procedures

# Assessment frequency by tier:
# Critical: Annual assessment + continuous monitoring
# High: Annual assessment + quarterly review
# Medium: Annual questionnaire
# Low: Initial assessment + biennial review
```

## See Also

- risk-management
- compliance-frameworks
- nist-csf
- iso-27001
- incident-response

## References

- NIST SP 800-53 Rev. 5 — Security and Privacy Controls
- ISO/IEC 27001:2022 — Information Security Management Systems
- ISO/IEC 27014:2020 — Governance of Information Security
- ISACA COBIT 2019 — Governance and Management Framework
- NIST Cybersecurity Framework (CSF) 2.0
- SOC 2 Trust Services Criteria (AICPA)
