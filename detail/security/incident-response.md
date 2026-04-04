# The Mathematics of Incident Response — Time, Containment, and Cost

> *Incident response is a time-critical optimization problem: minimize the product of dwell time and blast radius. Every hour of uncontained breach increases damage exponentially, while detection and containment follow measurable probability curves.*

---

## 1. Incident Timeline Metrics

### Key Time Intervals

$$\text{Total breach duration} = T_{dwell} = T_{detect} + T_{contain} + T_{eradicate}$$

| Metric | Symbol | Definition |
|:---|:---:|:---|
| Mean Time to Detect | MTTD | Discovery of breach |
| Mean Time to Contain | MTTC | Isolation of threat |
| Mean Time to Eradicate | MTTE | Full removal |
| Mean Time to Recover | MTTR | Return to normal ops |
| Dwell Time | $T_{dwell}$ | Total attacker presence |

### Industry Benchmarks (2024)

| Metric | Median | Top 10% | Bottom 10% |
|:---|:---:|:---:|:---:|
| MTTD | 204 days | 14 days | 400+ days |
| MTTC | 73 days | 7 days | 200+ days |
| Total dwell | 277 days | 21 days | 600+ days |

### Cost as a Function of Dwell Time

$$C(t) = C_{base} + C_{daily} \times t + C_{records} \times R(t)$$

Where:
- $C_{base}$ = fixed incident costs (forensics, legal, notification)
- $C_{daily}$ = per-day operational cost
- $R(t)$ = records compromised (grows with time)

| Dwell Time | Records Exposed | Total Cost (est.) |
|:---:|:---:|:---:|
| 1 day | 1,000 | $50K |
| 30 days | 100,000 | $500K |
| 200 days | 10,000,000 | $4.5M |
| 365 days | 50,000,000 | $9.4M |

---

## 2. Detection Probability

### Detection Sources

| Source | $P(\text{detect within 24h})$ | $P(\text{detect within 30d})$ |
|:---|:---:|:---:|
| SIEM/SOC | 0.15 | 0.60 |
| EDR alert | 0.25 | 0.70 |
| User report | 0.05 | 0.20 |
| Third-party notification | 0.02 | 0.15 |
| Network anomaly | 0.10 | 0.45 |
| Threat hunting | 0.03 | 0.30 |

### Combined Detection Probability

With $n$ independent detection sources:

$$P(\text{detect}) = 1 - \prod_{i=1}^{n} (1 - P_i)$$

Using all 6 sources:

$$P(\text{detect 24h}) = 1 - (0.85)(0.75)(0.95)(0.98)(0.90)(0.97) = 1 - 0.497 = 0.503$$

$$P(\text{detect 30d}) = 1 - (0.40)(0.30)(0.80)(0.85)(0.55)(0.70) = 1 - 0.025 = 0.975$$

Multiple detection sources make long-term evasion very difficult.

---

## 3. Containment Strategies — Blast Radius

### Blast Radius Model

$$B(t) = |N(C_0, t)| = \text{reachable nodes from initial compromise } C_0 \text{ at time } t$$

Without network segmentation (flat network):

$$B(t) = \min(N_{total}, B_0 \times e^{\lambda t})$$

With segmentation:

$$B(t) = \min(|S_0|, B_0 \times e^{\lambda' t}) \quad \text{where } |S_0| \ll N_{total}$$

$\lambda' < \lambda$ because each segment boundary requires additional exploit effort.

### Segmentation Impact

| Network Type | Max Blast Radius | Containment Time |
|:---|:---:|:---:|
| Flat (no segmentation) | All hosts | Hours to full compromise |
| VLANs (basic) | Segment (~100 hosts) | Days to escape |
| Micro-segmented | 1-5 hosts | Requires specific lateral exploit |
| Zero trust | 1 host (initially) | Each hop requires new auth |

### Containment Decision Matrix

| Action | Speed | Disruption | Completeness |
|:---|:---:|:---:|:---:|
| Network isolation | Minutes | High | High |
| Account disable | Minutes | Medium | Medium |
| Endpoint quarantine | Minutes | Low (per host) | Medium |
| Firewall rules | Hours | Low | Partial |
| Full shutdown | Immediate | Maximum | Complete |

---

## 4. MITRE ATT&CK Coverage

### Technique Coverage Score

$$\text{Coverage} = \frac{|\text{Techniques with detection}|}{|\text{Total ATT&CK techniques}|}$$

| Category | Total Techniques | Typical Detection | Coverage |
|:---|:---:|:---:|:---:|
| Initial Access | 9 | 6 | 67% |
| Execution | 12 | 9 | 75% |
| Persistence | 19 | 14 | 74% |
| Privilege Escalation | 13 | 9 | 69% |
| Defense Evasion | 42 | 15 | 36% |
| Credential Access | 17 | 12 | 71% |
| Discovery | 31 | 8 | 26% |
| Lateral Movement | 9 | 7 | 78% |
| Collection | 17 | 5 | 29% |
| Exfiltration | 9 | 5 | 56% |

**Defense Evasion** and **Discovery** are the hardest to detect — attackers specifically optimize these phases.

---

## 5. Severity Classification

### Impact Score

$$I = \frac{C + I + A}{3} \times \text{scope}$$

Where $C, I, A \in [0, 10]$ (confidentiality, integrity, availability) and scope is a multiplier:

| Scope | Multiplier | Example |
|:---|:---:|:---|
| Single system | 1.0 | One compromised workstation |
| Department | 2.0 | Shared drive encrypted |
| Business unit | 3.0 | Database breach |
| Organization-wide | 5.0 | Domain controller compromised |
| Cross-organization | 7.0 | Supply chain attack |

### Severity Levels

| Level | Impact Score | Response Time | Escalation |
|:---|:---:|:---:|:---|
| P1 (Critical) | > 30 | < 15 min | CISO + Legal + Exec |
| P2 (High) | 20-30 | < 1 hour | IR Lead + Management |
| P3 (Medium) | 10-20 | < 4 hours | IR Team |
| P4 (Low) | < 10 | < 24 hours | Analyst |

---

## 6. Evidence Collection — Chain of Custody

### Evidence Volatility Order (RFC 3227)

Collect most volatile evidence first:

| Priority | Source | Volatility | Collection Time |
|:---:|:---|:---|:---:|
| 1 | CPU registers, cache | Nanoseconds | Impractical |
| 2 | Memory (RAM) | Seconds (power loss) | 1-30 min |
| 3 | Network state | Seconds-minutes | 1-5 min |
| 4 | Running processes | Minutes | 1 min |
| 5 | Disk (filesystem) | Persistent | 30-120 min |
| 6 | Remote logs | Days (rotation) | Minutes |
| 7 | Archived data | Months-years | Hours |

### Evidence Integrity

Each evidence item requires cryptographic verification:

$$\text{Integrity}(E) = \text{SHA-256}(E) \text{ at collection time}$$

$$\text{Chain verified} \iff H(E_{current}) = H(E_{collected})$$

If the hash changes, the evidence is tainted and inadmissible.

---

## 7. Post-Incident Metrics

### Lessons Learned Quantification

$$\text{Improvement} = \frac{T_{dwell,before} - T_{dwell,after}}{T_{dwell,before}} \times 100\%$$

### Incident Recurrence

$$P(\text{repeat within 1 year}) = 1 - e^{-\lambda \times 365}$$

Where $\lambda$ is the incident rate per day.

| Incidents/Year | $\lambda$ | $P(\text{recurrence in 90 days})$ |
|:---:|:---:|:---:|
| 1 | 0.0027 | 21.7% |
| 4 | 0.011 | 62.8% |
| 12 | 0.033 | 94.8% |

### Cost-Benefit of IR Investment

$$\text{ROI}_{IR} = \frac{C_{breach} \times P(\text{breach}) \times \text{risk reduction}}{C_{IR}}$$

| IR Investment | Annual Cost | Risk Reduction | ROI (at $4.5M breach, 20% chance) |
|:---|:---:|:---:|:---:|
| SOC (24/7) | $1M | 40% | $\frac{4.5M \times 0.2 \times 0.4}{1M} = 36\%$ |
| EDR + SIEM | $200K | 30% | $\frac{4.5M \times 0.2 \times 0.3}{200K} = 135\%$ |
| IR retainer | $100K | 20% | $\frac{4.5M \times 0.2 \times 0.2}{100K} = 180\%$ |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $C_{base} + C_{daily} \times t$ | Linear cost | Breach cost model |
| $1 - \prod(1 - P_i)$ | Independence complement | Detection probability |
| $B_0 \times e^{\lambda t}$ | Exponential growth | Blast radius |
| $C + I + A$ / 3 | Weighted average | Severity score |
| $1 - e^{-\lambda t}$ | Exponential CDF | Recurrence probability |
| ROI ratio | Cost-benefit | IR investment |

## Prerequisites

- timeline analysis, exponential cost growth, probability, forensic chain of custody

---

*Incident response is a race against an exponential clock — every hour of dwell time multiplies the blast radius and cost. The mathematics prove that investment in detection speed (reducing MTTD) has far greater ROI than investment in prevention alone.*
