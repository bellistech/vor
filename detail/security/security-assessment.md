# Security Assessment — Theory, Scoring Mathematics, and Maturity

> *Security assessment is the systematic process of measuring an organization's security posture against threats and vulnerabilities. The science behind vulnerability scoring, risk-based prioritization, and assessment methodology determines whether limited remediation resources are applied where they matter most.*

---

## 1. Vulnerability Assessment Theory

### Vulnerability Lifecycle

A vulnerability progresses through stages with distinct risk profiles:

```
Discovery → Disclosure → Patch Available → Patch Applied → Verified
    │            │              │                │            │
    ▼            ▼              ▼                ▼            ▼
 Zero-day   Public exploit   Race to patch   Remediated   Confirmed
 (unknown    (max risk)      (decreasing     (residual    (closed)
  to vendor)                  risk window)    risk only)
```

**Risk window** = time between public disclosure and patch application:

$$\text{Risk Window} = T_{patch\_applied} - T_{disclosure}$$

Industry average risk windows:
- Critical vulnerabilities: 60-150 days (should be <14 days)
- High vulnerabilities: 90-180 days (should be <30 days)
- The longer the window, the higher the probability of exploitation

### Vulnerability Categories

| Category | Definition | Example |
|:---|:---|:---|
| Design flaw | Fundamental architectural weakness | Insecure protocol design |
| Implementation bug | Coding error introducing vulnerability | Buffer overflow, SQL injection |
| Configuration error | Misconfigured secure software | Default credentials, open S3 bucket |
| Missing patch | Known fix not applied | Unpatched Apache Struts |
| Zero-day | Unknown vulnerability, no patch exists | Novel exploit |

### Scan Accuracy Metrics

$$\text{True Positive (TP)} = \text{scanner reports vuln that actually exists}$$
$$\text{False Positive (FP)} = \text{scanner reports vuln that does not exist}$$
$$\text{True Negative (TN)} = \text{scanner correctly reports no vuln}$$
$$\text{False Negative (FN)} = \text{scanner misses actual vuln}$$

**Precision** (positive predictive value):
$$\text{Precision} = \frac{TP}{TP + FP}$$

**Recall** (sensitivity, detection rate):
$$\text{Recall} = \frac{TP}{TP + FN}$$

**F1 Score** (harmonic mean of precision and recall):
$$F_1 = 2 \times \frac{\text{Precision} \times \text{Recall}}{\text{Precision} + \text{Recall}}$$

| Scanner Type | Typical Precision | Typical Recall | Notes |
|:---|:---|:---|:---|
| Unauthenticated network scan | 60-70% | 40-60% | Many FP from banner grabbing |
| Credentialed scan | 85-95% | 80-90% | Reads actual installed versions |
| Agent-based | 95-99% | 90-98% | Real-time, OS-level visibility |

### False Positive Analysis

Common causes of false positives:
1. **Banner-based detection:** Scanner reads version string, vendor backports security fix without changing version
2. **Signature mismatch:** Generic pattern matches benign code
3. **Context ignorance:** Vulnerability exists but compensating control neutralizes it
4. **Environment assumption:** Scanner assumes default configuration

Cost of false positives:
$$\text{FP Cost} = N_{FP} \times T_{triage} \times C_{analyst\_hour}$$

At 500 FP per scan, 15 minutes triage each, $75/hour analyst cost:
$$500 \times 0.25 \times \$75 = \$9,375 \text{ per scan cycle in wasted triage time}$$

---

## 2. CVSS v3.1 Scoring Mathematics

### Base Score Calculation

**Step 1:** Calculate Impact Sub Score (ISS):

$$\text{ISS} = 1 - [(1 - C) \times (1 - I) \times (1 - A)]$$

Where $C$, $I$, $A$ are the confidentiality, integrity, and availability impact values.

**Step 2:** Calculate Impact:

If Scope is Unchanged:
$$\text{Impact} = 6.42 \times \text{ISS}$$

If Scope is Changed:
$$\text{Impact} = 7.52 \times [\text{ISS} - 0.029] - 3.25 \times [\text{ISS} - 0.02]^{15}$$

**Step 3:** Calculate Exploitability:

$$\text{Exploitability} = 8.22 \times AV \times AC \times PR \times UI$$

**Step 4:** Calculate Base Score:

If Impact $\leq 0$: Base Score $= 0$

If Scope is Unchanged:
$$\text{Base} = \text{Roundup}(\min[(\text{Impact} + \text{Exploitability}), 10])$$

If Scope is Changed:
$$\text{Base} = \text{Roundup}(\min[1.08 \times (\text{Impact} + \text{Exploitability}), 10])$$

The Roundup function rounds to the nearest tenth, rounding 0.05 up.

### Worked Example

**CVE: Remote Code Execution via unauthenticated network request**

| Metric | Value | Score |
|:---|:---|:---|
| AV | Network | 0.85 |
| AC | Low | 0.77 |
| PR | None | 0.85 |
| UI | None | 0.85 |
| S | Changed | — |
| C | High | 0.56 |
| I | High | 0.56 |
| A | High | 0.56 |

$$\text{ISS} = 1 - [(1 - 0.56)(1 - 0.56)(1 - 0.56)]$$
$$= 1 - [0.44 \times 0.44 \times 0.44]$$
$$= 1 - 0.085184 = 0.914816$$

$$\text{Impact (Changed)} = 7.52 \times [0.914816 - 0.029] - 3.25 \times [0.914816 - 0.02]^{15}$$
$$= 7.52 \times 0.885816 - 3.25 \times 0.894816^{15}$$
$$= 6.661 - 3.25 \times 0.1876$$
$$= 6.661 - 0.610 = 6.051$$

$$\text{Exploitability} = 8.22 \times 0.85 \times 0.77 \times 0.85 \times 0.85$$
$$= 8.22 \times 0.473 = 3.887$$

$$\text{Base} = \text{Roundup}(\min[1.08 \times (6.051 + 3.887), 10])$$
$$= \text{Roundup}(\min[10.734, 10]) = 10.0$$

### Temporal Score

Temporal metrics adjust the base score based on current exploit state:

$$\text{Temporal} = \text{Roundup}(\text{Base} \times E \times RL \times RC)$$

| Metric | Values | Weight |
|:---|:---|:---|
| Exploit Code Maturity (E) | Not Defined=1.0, Unproven=0.91, PoC=0.94, Functional=0.97, High=1.0 |
| Remediation Level (RL) | Not Defined=1.0, Official Fix=0.95, Temp Fix=0.96, Workaround=0.97, Unavailable=1.0 |
| Report Confidence (RC) | Not Defined=1.0, Unknown=0.92, Reasonable=0.96, Confirmed=1.0 |

### Environmental Score

Organizations modify scores based on their specific context:
- Modified Base Metrics (adjust AV, AC, PR, UI, S, C, I, A for local context)
- Security Requirements (CR, IR, AR): Low=0.5, Medium=1.0, High=1.5

This allows a vulnerability rated 9.8 globally to be scored 6.5 locally if compensating controls exist.

---

## 3. Risk-Based Vulnerability Prioritization

### EPSS (Exploit Prediction Scoring System)

EPSS provides the probability that a vulnerability will be exploited in the wild within 30 days:

$$P(\text{exploit within 30 days} | \text{CVE features})$$

**Features used by the EPSS model:**
- Vulnerability age, vendor, product type
- CVSS base metrics
- Exploit database presence (Metasploit, ExploitDB)
- Social media mentions and threat intelligence
- Historical exploitation patterns

**EPSS vs CVSS prioritization:**

| Approach | Question Answered | Typical Coverage |
|:---|:---|:---|
| CVSS-only (Critical/High) | "How bad could it be?" | Remediates ~40% of all CVEs |
| EPSS top 10% | "How likely to be exploited?" | Catches ~80% of exploited CVEs |
| Combined (CVSS >= 7 AND EPSS >= 10%) | Intersection | Highest ROI |

The CVSS-only approach remediates many vulnerabilities that are never actually exploited, while missing some medium-severity CVEs that are actively exploited.

### SSVC (Stakeholder-Specific Vulnerability Categorization)

Developed by CERT/CC, SSVC uses decision trees instead of numeric scores:

**Decision points:**

| Point | Values | Meaning |
|:---|:---|:---|
| Exploitation | None / PoC / Active | Current exploitation status |
| Automatable | No / Yes | Can exploitation be automated? |
| Technical Impact | Partial / Total | Scope of compromise |
| Mission Prevalence | Minimal / Support / Essential | How critical is the affected system? |
| Public Well-Being | Minimal / Material / Irreversible | Impact on public safety |

**Decision outcomes:**
- **Defer:** No action needed now
- **Scheduled:** Remediate within normal cycle
- **Out-of-cycle:** Remediate outside normal cycle, with urgency
- **Immediate:** Act now, escalate to leadership

### Combined Prioritization Framework

$$\text{Priority Score} = w_1 \times \text{CVSS} + w_2 \times \text{EPSS} + w_3 \times \text{Asset Criticality} + w_4 \times \text{Exposure}$$

Where:
- CVSS: normalized severity (0-1)
- EPSS: exploitation probability (0-1)
- Asset criticality: business importance (0-1)
- Exposure: internet-facing=1.0, internal=0.5, isolated=0.2

---

## 4. Penetration Testing Ethics and Legal Framework

### Legal Authorization

**Required documentation:**

| Document | Purpose | Content |
|:---|:---|:---|
| Scope of Work (SOW) | Define engagement boundaries | Systems, methods, timeline, deliverables |
| Rules of Engagement (ROE) | Operating constraints | Allowed/disallowed techniques, times, contacts |
| Authorization Letter | Legal permission | Signed by system owner, explicit permission to test |
| NDA | Protect findings | Confidentiality of vulnerabilities and data |
| Liability Waiver | Risk acceptance | Client accepts risk of testing (DoS, data corruption) |

### Legal Considerations

| Jurisdiction | Key Law | Implication |
|:---|:---|:---|
| US | Computer Fraud and Abuse Act (CFAA) | Unauthorized access is federal crime |
| US | State laws (vary) | Some states have stricter provisions |
| EU | Computer Misuse Directive | Similar to CFAA across EU member states |
| UK | Computer Misuse Act 1990 | Unauthorized access, modification, supply of tools |

**Critical requirements:**
1. Written authorization from the legal owner of every tested system
2. Cloud provider notification (AWS, Azure, GCP have testing policies)
3. Third-party systems explicitly excluded unless separately authorized
4. Data handling: PII encountered during testing must be protected
5. Evidence preservation: secure storage, defined retention, secure deletion

### Ethical Guidelines

- **Do no harm:** Testing should not cause outage, data loss, or business disruption
- **Minimal footprint:** Extract only enough data to prove the vulnerability
- **Immediate disclosure:** Report critical findings immediately, not just at engagement end
- **Data protection:** Never exfiltrate real PII/PHI; redact in reports
- **Stay in scope:** Do not test adjacent systems even if accessible
- **Professional conduct:** Do not leverage findings for personal gain

---

## 5. Assessment Frequency and Triggers

### Recommended Frequencies

| Assessment Type | Minimum Frequency | Regulatory Driver |
|:---|:---|:---|
| Vulnerability scan (external) | Quarterly | PCI DSS 11.3.1 |
| Vulnerability scan (internal) | Quarterly | PCI DSS 11.3.2 |
| Penetration test (external) | Annually | PCI DSS 11.4.1 |
| Penetration test (internal) | Annually | PCI DSS 11.4.2 |
| Application security test | Each major release | NIST SSDF |
| Red team exercise | Annually | TIBER-EU, CBEST |
| Configuration audit | Monthly | CIS Benchmarks |

### Event-Triggered Assessments

| Trigger | Assessment Required |
|:---|:---|
| Major infrastructure change | Vulnerability scan + config review |
| New application deployment | Full application security test |
| Significant code change | SAST/DAST + pen test |
| Merger/acquisition | Full security assessment of acquired assets |
| Security incident | Focused assessment of affected systems |
| New compliance requirement | Gap assessment against new standard |
| Vendor/supplier change | Third-party risk assessment |

---

## 6. Continuous Vulnerability Management

### Architecture

```
┌─────────────┐    ┌──────────────┐    ┌──────────────┐
│ Asset        │    │ Vulnerability │    │ Threat        │
│ Inventory   │───>│ Scanner       │<───│ Intelligence  │
│ (CMDB)      │    │              │    │ (feeds)       │
└─────────────┘    └──────┬───────┘    └──────────────┘
                          │
                    ┌─────▼──────┐
                    │ Correlation │
                    │ & Dedup     │
                    └─────┬──────┘
                          │
              ┌───────────▼───────────┐
              │ Prioritization Engine  │
              │ (CVSS + EPSS + context)│
              └───────────┬───────────┘
                          │
              ┌───────────▼───────────┐
              │ Remediation Workflow   │
              │ (ticketing, SLA, track)│
              └───────────┬───────────┘
                          │
              ┌───────────▼───────────┐
              │ Verification           │
              │ (rescan, validate)     │
              └───────────────────────┘
```

### KPIs for Vulnerability Management

| KPI | Formula | Target |
|:---|:---|:---|
| Mean Time to Remediate (MTTR) | $\frac{\sum (T_{fixed} - T_{discovered})}{N_{vulns}}$ | Critical: <14 days |
| Scan Coverage | $\frac{\text{Assets scanned}}{\text{Total assets}} \times 100$ | >95% |
| Vulnerability Density | $\frac{\text{Open vulns}}{\text{Total assets}}$ | Decreasing trend |
| Remediation Rate | $\frac{\text{Vulns fixed}}{\text{Vulns discovered}} \times 100$ | >80% per cycle |
| Overdue Vulns | $\frac{\text{Vulns past SLA}}{\text{Total open vulns}} \times 100$ | <10% |
| Recurrence Rate | $\frac{\text{Vulns reintroduced}}{\text{Vulns fixed}} \times 100$ | <5% |

---

## 7. Assessment Maturity Model

### Maturity Levels

| Level | Name | Characteristics |
|:---|:---|:---|
| 1 | Initial | Ad hoc scanning, no process, no tracking |
| 2 | Repeatable | Scheduled scans, basic tracking, manual prioritization |
| 3 | Defined | Documented process, CVSS-based prioritization, SLAs defined |
| 4 | Managed | Risk-based prioritization (EPSS/SSVC), automated workflow, KPIs tracked |
| 5 | Optimizing | Predictive analytics, automated remediation, threat-informed, continuous |

### Level 1 to Level 5 Progression

**Level 1 → 2:** Implement scheduled scanning (quarterly minimum), create asset inventory, assign vulnerability ownership.

**Level 2 → 3:** Document vulnerability management policy, define SLAs by severity, implement ticketing integration, train teams on CVSS.

**Level 3 → 4:** Add EPSS/SSVC for prioritization, integrate threat intelligence feeds, automate scan-to-ticket workflow, establish KPI dashboards, implement credentialed scanning.

**Level 4 → 5:** Deploy agent-based continuous scanning, implement auto-remediation for low-risk patches, use ML for false positive reduction, integrate with SOAR for response automation, conduct tabletop exercises for zero-day scenarios.

---

## 8. CVSS v4.0 Deep Dive

### Structural Changes from v3.1

CVSS v4.0 introduces a more granular scoring system:

**New metric groups:**
- Base (mandatory): exploitability + impact
- Threat (optional, replaces Temporal): current threat context
- Environmental (optional): local adjustments
- Supplemental (optional): additional context (not affecting score)

**Key metric changes:**

| v3.1 | v4.0 | Change |
|:---|:---|:---|
| Attack Complexity (AC) | Attack Complexity (AC) + Attack Requirements (AT) | Split into two dimensions |
| Scope (S) | Removed | Impact now explicit per system |
| C/I/A Impact | Vulnerable System (VC/VI/VA) + Subsequent System (SC/SI/SA) | Explicit multi-system impact |
| Exploit Code Maturity | Removed from scoring | Moved to Supplemental |
| — | Automatable (new) | Can attack be automated at scale? |
| — | Recovery (new) | Can the system recover? |
| — | Safety (new) | Physical safety impact |
| — | Provider Urgency (new) | Vendor's urgency assessment |

**Nomenclature:**

| Score + Supplemental | Label |
|:---|:---|
| CVSS-B | Base only |
| CVSS-BT | Base + Threat |
| CVSS-BE | Base + Environmental |
| CVSS-BTE | Base + Threat + Environmental |

---

## References

- FIRST CVSS v3.1 Specification: https://www.first.org/cvss/v3.1/specification-document
- FIRST CVSS v4.0 Specification: https://www.first.org/cvss/v4.0/specification-document
- FIRST EPSS: Exploit Prediction Scoring System (https://www.first.org/epss/)
- CERT/CC SSVC: Stakeholder-Specific Vulnerability Categorization
- NIST SP 800-115: Technical Guide to Information Security Testing and Assessment
- NIST SP 800-40r4: Guide to Enterprise Patch Management Planning
- PTES: Penetration Testing Execution Standard
- PCI DSS v4.0: Requirements 6 and 11
- OWASP Testing Guide v4.2
- SANS Critical Security Controls (CIS Controls v8)
