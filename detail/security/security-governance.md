# Security Governance — Theory and Frameworks

> *Security governance establishes the strategic direction, oversight mechanisms, and accountability structures that ensure an organization's security program aligns with business objectives. Without governance, security becomes a collection of disconnected technical controls rather than a coherent risk management program.*

---

## 1. Governance vs Management vs Operations

### The Three Tiers

Security functions at three distinct levels, each with different stakeholders, timeframes, and concerns:

| Aspect | Governance | Management | Operations |
|:---|:---|:---|:---|
| **Question** | What should we do? | How do we do it? | Are we doing it? |
| **Focus** | Direction, oversight | Planning, execution | Daily activity |
| **Timeframe** | Strategic (1-5 years) | Tactical (months) | Operational (daily) |
| **Stakeholders** | Board, executives | Managers, architects | Analysts, engineers |
| **Output** | Policies, strategy | Standards, projects | Alerts, patches, logs |
| **Accountability** | Board/CEO | CISO/CIO | Team leads |
| **Metrics** | Risk posture, ROI | Project status, KPIs | SLAs, ticket counts |

### Governance Principles

Five principles underpin effective security governance:

1. **Accountability** — Clear ownership of security decisions and outcomes
2. **Transparency** — Visibility into security posture for all stakeholders
3. **Proportionality** — Security controls proportional to risk
4. **Integration** — Security embedded in business processes, not bolted on
5. **Alignment** — Security strategy supports business strategy

### The Principal-Agent Problem

In security governance, the **board** (principal) delegates security to the **CISO** (agent). Information asymmetry creates risk:

$$risk_{governance} = f(\text{information asymmetry}, \text{incentive misalignment})$$

The board cannot directly verify security effectiveness. This drives the need for:
- Independent audits (external verification)
- Metrics and reporting (reduce information asymmetry)
- Aligned incentives (CISO compensation tied to risk outcomes, not just compliance)
- Separation of duties (prevent self-assessment)

---

## 2. Corporate Governance Integration

### Security in the Corporate Governance Model

```
Shareholders
    ↓ (elect)
Board of Directors
    ↓ (oversees)
├── Audit Committee ← Security risk reporting
├── Risk Committee ← Enterprise risk including cyber
└── Compensation Committee ← CISO/security team incentives
    ↓ (delegates to)
Executive Management
    ↓
CISO ← Dual reporting: operational + risk
```

### Board Responsibilities for Security

Modern boards have **fiduciary duties** that include cybersecurity oversight:

| Duty | Security Implication |
|:---|:---|
| Duty of Care | Informed security decisions, adequate investment |
| Duty of Loyalty | No conflicts in security vendor selection |
| Duty of Obedience | Regulatory compliance (GDPR, HIPAA, etc.) |

### SEC Cybersecurity Disclosure Requirements (US)

Organizations must disclose:
- Material cybersecurity incidents (within 4 business days)
- Cybersecurity risk management processes
- Board oversight of cybersecurity risk
- Management's role in cybersecurity

This elevates security governance from a technical concern to a **boardroom agenda item**.

---

## 3. Security Governance Frameworks

### COBIT (Control Objectives for Information Technology)

COBIT provides a comprehensive governance framework bridging business requirements with IT capabilities:

**Governance objectives:**

| Domain | Focus |
|:---|:---|
| EDM01 | Ensure governance framework setting and maintenance |
| EDM02 | Ensure benefits delivery |
| EDM03 | Ensure risk optimization |
| EDM04 | Ensure resource optimization |
| EDM05 | Ensure stakeholder engagement |

**Management objectives (security-relevant):**

| Domain | Objective |
|:---|:---|
| APO12 | Managed risk |
| APO13 | Managed security |
| DSS05 | Managed security services |
| MEA01 | Managed performance and conformance monitoring |

COBIT uses a **capability maturity model** (0-5) for each process, enabling benchmarking and improvement tracking.

### ISO 27014 — Governance of Information Security

ISO 27014 defines six governance principles:

1. **Establish organization-wide information security** — not just IT
2. **Adopt a risk-based approach** — proportional to threats and value
3. **Set the direction of investment decisions** — prioritize security spending
4. **Ensure conformance with internal and external requirements** — compliance
5. **Foster a security-positive environment** — culture and awareness
6. **Review performance in relation to business outcomes** — not just technical metrics

**Governance processes:**

$$\text{Evaluate} \to \text{Direct} \to \text{Monitor} \to \text{Communicate} \to \text{Assure}$$

### NIST Cybersecurity Framework (CSF) 2.0

CSF 2.0 adds **Govern** as a sixth core function:

| Function | Purpose |
|:---|:---|
| **Govern** (new) | Establish and monitor risk management strategy, expectations, and policy |
| Identify | Asset management, risk assessment, supply chain |
| Protect | Access control, training, data security |
| Detect | Anomalies, continuous monitoring |
| Respond | Response planning, communications, mitigation |
| Recover | Recovery planning, improvements, communications |

The Govern function includes:
- GV.OC — Organizational Context
- GV.RM — Risk Management Strategy
- GV.RR — Roles, Responsibilities, and Authorities
- GV.PO — Policy
- GV.OV — Oversight
- GV.SC — Cybersecurity Supply Chain Risk Management

### Framework Comparison

| Aspect | COBIT | ISO 27014 | NIST CSF |
|:---|:---|:---|:---|
| Scope | IT governance (broad) | InfoSec governance | Cybersecurity |
| Audience | Enterprise governance | Security leaders | All organizations |
| Maturity model | Yes (0-5) | No (conformance) | Yes (tiers 1-4) |
| Certification | No | Via ISO 27001 | No |
| Prescriptiveness | High | Medium | Low (flexible) |
| Cost | Licensed | Standard (paid) | Free |
| Best for | Large enterprises | ISMS governance | Flexible adoption |

---

## 4. Policy Hierarchy

### Policy Architecture

The policy hierarchy creates a layered framework where each level adds specificity:

```
                    ┌──────────┐
                    │  POLICY  │  WHY and WHAT (mandatory)
                    │          │  Approved by: Board/Executive
                    └────┬─────┘  Review: Annual
                         │
                ┌────────┴────────┐
                │    STANDARD     │  WHAT specifically (mandatory)
                │                 │  Approved by: CISO/Committee
                └────────┬────────┘  Review: Annual
                         │
            ┌────────────┴────────────┐
            │       GUIDELINE         │  HOW (recommended)
            │                         │  Approved by: Security team
            └────────────┬────────────┘  Review: As needed
                         │
        ┌────────────────┴────────────────┐
        │           PROCEDURE             │  HOW step-by-step (mandatory)
        │                                 │  Approved by: Team lead
        └─────────────────────────────────┘  Review: As needed
```

### Characteristics of Each Level

**Policy:**
- Technology-neutral ("data must be encrypted at rest")
- Enduring (survives technology changes)
- Concise (2-5 pages)
- Mandatory for all in scope
- Requires executive approval for changes
- Written in business language

**Standard:**
- Technology-specific ("AES-256 for data at rest, TLS 1.3 for transit")
- Changes with technology landscape
- Detailed (5-20 pages)
- Mandatory for all implementations
- CISO approval for changes
- Written in technical language

**Guideline:**
- Advisory ("consider using full-disk encryption for laptops")
- Flexible, allows alternatives
- Recommended best practices
- Not auditable as requirements
- Security team maintains

**Procedure:**
- Step-by-step instructions
- Specific to systems and tools
- Most frequently updated
- Enables consistent execution
- Team-level ownership

### Policy Writing Principles

Effective policies share these characteristics:

| Principle | Correct | Incorrect |
|:---|:---|:---|
| Technology-neutral | "Strong authentication required" | "Must use RSA-2048" |
| Measurable | "Review access quarterly" | "Review access regularly" |
| Enforceable | "MFA required for remote access" | "MFA encouraged" |
| Clear scope | "All employees and contractors" | "Users" |
| Exception process | "Exceptions require CISO approval" | No mention |
| Consequence stated | "Violations subject to disciplinary action" | No consequence |

---

## 5. Due Care vs Due Diligence

### Legal Foundation

These legal concepts underpin security governance liability:

**Due care** — Taking **reasonable measures** to protect assets and stakeholders. Actions demonstrate the organization is behaving responsibly.

$$\text{due care} = \text{acting as a "reasonable person" would under similar circumstances}$$

**Due diligence** — The process of **investigating and verifying** that due care measures are effective. Continuous monitoring and improvement.

$$\text{due diligence} = \text{verifying that due care is actually working}$$

### Practical Application

| Concept | Example |
|:---|:---|
| Due care | Implementing firewall rules, encrypting data, training employees |
| Due diligence | Penetration testing the firewall, auditing encryption, measuring training effectiveness |
| Neither | Knowing about a vulnerability and not patching it |

### Legal Liability Implications

```
Breach occurs
    ↓
Did the organization exercise due care?
├── YES → Was due diligence performed?
│         ├── YES → Defensible position (reduced liability)
│         └── NO  → Negligence risk (should have verified)
└── NO  → Negligence likely (failed to act reasonably)
```

### Negligence Test

A plaintiff must prove all four elements:

1. **Duty** — Organization had a duty to protect (contractual, regulatory, or common law)
2. **Breach** — Organization failed to meet that duty (no due care)
3. **Causation** — The breach caused the harm
4. **Damages** — Actual harm occurred

Due care and due diligence directly address element 2 (breach).

---

## 6. Security Governance in Regulated Industries

### Industry-Specific Requirements

| Industry | Primary Regulations | Governance Impact |
|:---|:---|:---|
| Financial | SOX, GLBA, PCI DSS, DORA | Board attestation, audit committees, data protection officer |
| Healthcare | HIPAA, HITECH | Privacy officer, security officer, risk analysis, BAAs |
| Government | FISMA, FedRAMP, CMMC | ATO process, continuous monitoring, ISSO role |
| Critical Infrastructure | NERC CIP, TSA directives | Physical + cyber governance, mandatory reporting |
| Telecom | CPNI rules, GDPR | Data protection, lawful intercept, mandatory breach notification |
| Energy | NERC CIP, IEC 62443 | OT/IT governance separation, safety integration |

### Regulated Governance Differences

| Aspect | Unregulated | Regulated |
|:---|:---|:---|
| Policy review | Best practice (annual) | Mandatory (defined cadence) |
| Risk assessment | Voluntary | Required (documented) |
| Audit | Internal sufficient | External required |
| Incident reporting | Voluntary | Mandatory (defined timeline) |
| Board involvement | Recommended | Required (documented) |
| Training | Best practice | Mandatory (tracked) |
| Evidence retention | Varies | Specified (3-7 years) |

### Cross-Regulation Governance

Organizations subject to multiple regulations need a **unified governance framework** that maps common controls across regulations:

$$\text{Unified control set} = \bigcup_{i=1}^{n} \text{Regulation}_i \text{ controls}$$

One control can satisfy multiple requirements:

$$\text{control}_{encryption} \to \{PCI\_3.4, HIPAA\_164.312(a)(2)(iv), GDPR\_Art.32\}$$

This reduces **control sprawl** and **audit fatigue**.

---

## 7. Maturity Model Assessment

### Assessment Methodology

Maturity assessment evaluates each security domain against the maturity scale:

$$\text{Overall maturity} = \frac{\sum_{i=1}^{n} w_i \times maturity_i}{\sum_{i=1}^{n} w_i}$$

Where $w_i$ is the weight (importance) of domain $i$.

### Assessment Domains

| Domain | Level 1 Indicators | Level 3 Indicators | Level 5 Indicators |
|:---|:---|:---|:---|
| Policy | No formal policy | Comprehensive, reviewed annually | Adaptive, real-time updates |
| Risk mgmt | Reactive | Formal methodology, regular assessment | Predictive analytics, AI-assisted |
| IAM | Shared accounts | RBAC, MFA, quarterly reviews | Zero trust, continuous verification |
| Vulnerability mgmt | Ad hoc scanning | Regular scanning, SLA-based patching | Risk-based prioritization, auto-patching |
| Incident response | No plan | Documented plan, annual testing | Automated response, threat intel integration |
| Awareness | No training | Annual training, phishing sims | Continuous micro-learning, behavioral analytics |

### Gap Analysis

$$gap_i = target\_maturity_i - current\_maturity_i$$

$$investment\_priority = gap_i \times risk\_weight_i$$

Domains with the largest weighted gap get priority investment.

### Maturity Improvement Timeline

Moving one maturity level typically requires:

| Transition | Typical Timeline | Key Activities |
|:---|:---|:---|
| Level 1 → 2 | 6-12 months | Document policies, define roles, basic tooling |
| Level 2 → 3 | 12-18 months | Formalize processes, implement metrics, integrate security into SDLC |
| Level 3 → 4 | 18-24 months | Automate controls, advanced analytics, continuous monitoring |
| Level 4 → 5 | 24-36 months | Adaptive controls, predictive capabilities, innovation program |

---

## 8. Security Governance Metrics

### Metrics Hierarchy

Metrics should cascade from governance to operations:

```
GOVERNANCE METRICS (Board level)
├── Overall risk score trend
├── Regulatory compliance status
├── Security investment ROI
├── Major incident count and impact
└── Third-party risk posture
    ↓
MANAGEMENT METRICS (CISO level)
├── Program maturity scores by domain
├── Policy compliance rates
├── Risk treatment plan progress
├── Audit finding closure rates
└── Security budget utilization
    ↓
OPERATIONAL METRICS (Team level)
├── MTTD, MTTR, MTTC
├── Vulnerability remediation SLA compliance
├── Patching coverage and timeliness
├── Phishing simulation click rates
└── Alert-to-incident ratio
```

### Effective Metric Characteristics (SMART)

| Criterion | Description | Example |
|:---|:---|:---|
| Specific | Clearly defined, unambiguous | "Critical vulnerability patch time" not "patching speed" |
| Measurable | Quantifiable, consistent | Hours/days, not "fast" |
| Achievable | Realistic targets | <72 hours for critical, not "immediate" |
| Relevant | Tied to risk/business outcome | Patch time → breach risk reduction |
| Time-bound | Measured over defined period | Monthly, quarterly, annually |

### Leading vs Lagging Indicators

| Type | Description | Examples |
|:---|:---|:---|
| **Leading** | Predict future performance | Training completion rate, vulnerability scan coverage, security debt trend |
| **Lagging** | Measure past outcomes | Breach count, incident cost, compliance findings |

Leading indicators are more valuable for governance because they enable **proactive** decision-making:

$$\text{value}_{leading} > \text{value}_{lagging}$$

A lagging indicator (breach occurred) is too late. A leading indicator (critical patches overdue is increasing) enables prevention.

### Return on Security Investment (ROSI)

$$ROSI = \frac{ALE_{before} - ALE_{after} - \text{cost of control}}{\text{cost of control}} \times 100\%$$

Where:
- $ALE = SLE \times ARO$ (Annual Loss Expectancy = Single Loss Expectancy x Annual Rate of Occurrence)

**Caution:** ROSI is imprecise because ALE estimates are uncertain. Use ranges:

$$ROSI_{range} = [ROSI_{pessimistic}, ROSI_{optimistic}]$$

Focus on **relative** comparison between security investments rather than absolute ROSI values.

---

## References

- NIST Cybersecurity Framework (CSF) 2.0
- ISO/IEC 27001:2022 — ISMS Requirements
- ISO/IEC 27014:2020 — Governance of Information Security
- ISACA COBIT 2019 Framework
- NIST SP 800-53 Rev. 5 — Security and Privacy Controls
- NIST SP 800-100 — Information Security Handbook: A Guide for Managers
- SEC Final Rule: Cybersecurity Risk Management, Strategy, Governance, and Incident Disclosure (2023)
- ISC2 CISSP Common Body of Knowledge (Security and Risk Management domain)
- CMMI Institute — Capability Maturity Model Integration
