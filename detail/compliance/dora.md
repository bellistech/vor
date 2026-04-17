# The Mathematics of DORA — Resilience Metrics, Incident Classification, and Third-Party Risk Quantification

> *DORA transforms operational resilience from a qualitative aspiration into a measurable, auditable engineering property. Beneath its legal articles lies a quantitative framework: recovery time objectives bounded by service-level algebra, incident classification driven by multi-criteria decision theory, concentration risk measured by information-theoretic entropy, and threat-led penetration testing validated by attack-surface geometry.*

---

## 1. Recovery Time and Recovery Point Objectives

### The Problem

DORA Art. 12 requires documented recovery time objectives (RTOs) and recovery point objectives (RPOs) per ICT-supported business function. These are not legal abstractions — they define the maximum tolerable data loss and downtime, which propagate into infrastructure sizing, backup frequency, and replication topology.

### The Formula

Let $T_f$ be the failure time, $T_d$ the detection time, $T_r$ the recovery time. Then:

$$\text{RTO} \geq T_d + T_{\text{response}} + T_{\text{restore}} + T_{\text{validate}}$$

The Recovery Point Objective bounds data loss:

$$\text{RPO} = \max(T_f - T_{\text{last valid backup}})$$

For synchronous replication $\text{RPO} = 0$; for snapshot-based backup every $\Delta t$, $\text{RPO} \leq \Delta t$.

Combined availability from RTO and mean time between failures (MTBF):

$$A = \frac{\text{MTBF}}{\text{MTBF} + \text{RTO}}$$

For DORA critical functions, target $A \geq 99.95\%$ implies, with $\text{MTBF} = 8760$ hours (1 year):

$$\text{RTO} \leq \frac{8760(1 - 0.9995)}{0.9995} \approx 4.38 \text{ hours}$$

### Worked Example

Core banking ledger with monthly failure probability $0.001$:

- $\text{MTBF} = \frac{1}{0.001 \text{ per month}} = 1000 \text{ months}$
- Target $A = 99.99\%$
- Required $\text{RTO} = \frac{1000(1-0.9999)}{0.9999} \approx 0.1$ months $\approx 72$ hours

With 4-hour RPO using snapshot replication every 4 hours plus transaction log shipping every 5 minutes, worst-case data loss is 5 minutes of transactions.

### Why It Matters

DORA audits test whether RTOs and RPOs are *achievable*, not just declared. A 4-hour RTO claim with daily backups is mathematically impossible — regulators will find this in 15 minutes of evidence review.

---

## 2. Incident Classification (Multi-Criteria Decision Analysis)

### The Problem

JC RTS 2023/83 requires classification of incidents as "major" or "non-major" using multiple weighted criteria. The regulation's primary/secondary threshold logic must be implemented deterministically, with zero ambiguity for the 4-hour reporting clock.

### The Formula

Let criteria set $C = \{c_1, c_2, \ldots, c_n\}$ with binary indicators $x_i \in \{0, 1\}$ for whether threshold $T_i$ is crossed, partitioned into primary $P$ and secondary $S$ sets.

An incident is classified major if:

$$\text{major}(I) = \begin{cases} 1 & \text{if } \sum_{i \in P} x_i \geq 2 \\ 1 & \text{if } \sum_{i \in P} x_i \geq 1 \land \sum_{j \in S} x_j \geq 2 \\ 0 & \text{otherwise} \end{cases}$$

Primary criteria typically include:

- $c_1$: Clients affected $\geq 10\%$ or absolute number above threshold
- $c_2$: Duration $\geq 24$ hours unavailability
- $c_3$: Geographic spread $\geq 2$ EU member states
- $c_4$: Data losses (confidentiality/integrity/availability breach)

Secondary criteria:

- $c_5$: Economic impact $\geq$ regulatory threshold
- $c_6$: Reputational impact (high media/regulator attention)
- $c_7$: Critical services affected
- $c_8$: Cross-border interdependencies

### Worked Example

A ransomware incident encrypts customer transaction records for 36 hours, affecting 8% of clients in FR and BE (2 member states), with estimated economic impact of €250,000 and significant media coverage.

| Criterion | Threshold | Observed | Crossed |
|-----------|-----------|----------|---------|
| $c_1$ clients | $\geq 10\%$ | 8% | 0 |
| $c_2$ duration | $\geq 24$ h | 36 h | 1 |
| $c_3$ geo | $\geq 2$ MS | 2 | 1 |
| $c_4$ data | integrity breach | yes | 1 |
| $c_5$ economic | $\geq$ €100k | €250k | 1 |
| $c_6$ reputation | high | yes | 1 |

Primary sum: $x_2 + x_3 + x_4 = 3 \geq 2 \Rightarrow$ **major**. Initial notification within 4 hours of classification.

### Why It Matters

Inconsistent classification creates legal and regulatory exposure. Implement the logic in code, version-control it, and test every production incident through the classifier offline before filing.

---

## 3. Third-Party Concentration Risk (Information Theory)

### The Problem

DORA Art. 29 mandates concentration risk assessment for ICT third parties. Measuring concentration requires more than "how many vendors" — it requires measuring how *unevenly* critical functions depend on them.

### The Formula

Let $p_i$ be the proportion of critical business functions supported by provider $i$, with $\sum_i p_i = 1$.

Shannon entropy of the vendor portfolio:

$$H = -\sum_{i=1}^{n} p_i \log_2 p_i$$

Maximum entropy $H_{\max} = \log_2 n$ occurs when dependencies are uniformly distributed.

Herfindahl-Hirschman Index (HHI):

$$\text{HHI} = \sum_{i=1}^{n} p_i^2$$

Normalized concentration score:

$$K = 1 - \frac{H}{H_{\max}}$$

Where $K = 0$ is fully diversified, $K = 1$ is a single-vendor monoculture.

### Worked Example

Financial entity uses 5 providers with critical-function weights $p = (0.6, 0.2, 0.1, 0.05, 0.05)$.

$$H = -(0.6 \log_2 0.6 + 0.2 \log_2 0.2 + 0.1 \log_2 0.1 + 2 \cdot 0.05 \log_2 0.05)$$
$$H = -(0.6 \cdot -0.737 + 0.2 \cdot -2.322 + 0.1 \cdot -3.322 + 0.1 \cdot -4.322)$$
$$H \approx 0.442 + 0.464 + 0.332 + 0.432 \approx 1.67 \text{ bits}$$

$H_{\max} = \log_2 5 \approx 2.32$ bits.

$K = 1 - 1.67/2.32 \approx 0.28$.

HHI $= 0.36 + 0.04 + 0.01 + 0.0025 + 0.0025 = 0.415$ — substantial concentration.

### Why It Matters

If the largest provider fails, 60% of critical functions degrade simultaneously. DORA expects such concentration risk to be assessed, documented, and mitigated through diversification, exit strategies, or escrow arrangements.

---

## 4. Threat-Led Penetration Testing (Attack Surface Geometry)

### The Problem

TLPT under DORA (Art. 26) must cover "live production systems supporting critical or important functions." Quantifying coverage requires modeling attack surface as a reachability graph and measuring what fraction of high-value targets the red team actually traversed.

### The Formula

Model the production environment as a directed graph $G = (V, E)$ where $V$ is the set of assets and $E$ encodes connectivity plus privilege-escalation edges.

Let $T \subseteq V$ be the target assets (critical business functions). Let $P \subseteq V$ be the assets the red team demonstrably reached.

TLPT coverage:

$$\text{Coverage}_{\text{TLPT}} = \frac{|P \cap T|}{|T|}$$

Weighted by business criticality $w(v)$:

$$\text{Coverage}_{\text{weighted}} = \frac{\sum_{v \in P \cap T} w(v)}{\sum_{v \in T} w(v)}$$

Attack-path depth distribution $d(v)$ = shortest path from red team foothold to asset $v$. Expected time-to-compromise:

$$E[\text{TTC}] = \sum_{v \in T} P(\text{compromise}(v)) \cdot \tau(v)$$

Where $\tau(v)$ is the observed traversal time.

### Worked Example

Production target set $T$ = {ledger, payment gateway, customer portal, ID provider}, with weights $w = (0.4, 0.3, 0.2, 0.1)$ summing to 1.

Red team demonstrated compromise of {ledger, payment gateway, customer portal}.

$$\text{Coverage}_{\text{weighted}} = \frac{0.4 + 0.3 + 0.2}{1.0} = 0.9$$

Path depths: ledger via 4 hops in 8 hours; payment gateway via 3 hops in 4 hours; customer portal via 2 hops in 1 hour.

Median TTC across compromised targets: 4 hours. The ID provider was not reached — represents a genuine control success or a gap in the test plan, requiring explicit discussion in the closure report.

### Why It Matters

TLPT reports that merely say "tested successfully" without coverage metrics fail DORA oversight. Quantitative coverage + path depth + residual gaps are what lead overseers actually read.

---

## 5. Sub-Outsourcing Chain Risk Propagation

### The Problem

Art. 28 requires tracking sub-outsourcing chains. A provider may appear tier-1 but hide a tier-3 dependency on a single geographic region or a single upstream vendor that also serves multiple of your other tier-1 providers.

### The Formula

For provider $p$ at tier $t$ with sub-provider set $S_p$, define the propagated risk as:

$$R(p) = R_{\text{direct}}(p) + \sum_{s \in S_p} \alpha \cdot R(s)$$

Where $\alpha \in (0, 1)$ is a dampening factor reflecting contractual insulation (typically 0.3 to 0.7).

Shared-dependency detection: let $U_i$ be the union of all sub-outsourcers in the chain of tier-1 provider $i$. The shared-dependency set:

$$D_{\text{shared}} = \bigcap_{i \in \text{tier-1}} U_i$$

Non-empty $D_{\text{shared}}$ indicates hidden concentration.

### Worked Example

Bank uses tier-1 providers $P_1$ (cloud), $P_2$ (card issuing), $P_3$ (KYC).

- $P_1$'s sub-outsourcers: {DNS-X, CDN-Y, DC-Z}
- $P_2$'s sub-outsourcers: {DNS-X, PRINT-A}
- $P_3$'s sub-outsourcers: {DNS-X, KYC-DB-B}

$D_{\text{shared}} = \{\text{DNS-X}\}$ — a single sub-outsourcer supports three critical functions via three different tier-1 providers. DORA-required disclosure + exit strategy.

### Why It Matters

This is the most common DORA finding: regulators trace a specific single point of failure four tiers deep and ask the entity to remediate it. Without graph analysis, you will not find it before they do.

---

## 6. Synthesis — The Resilience Equation

Putting it together, DORA-compliant operational resilience is:

$$\text{Resilience} = f(\text{Detect}, \text{Respond}, \text{Recover}) \cdot g(\text{Concentration}, \text{Coverage}, \text{Chain depth})$$

- $\text{Detect}$: Mean time to detect (MTTD), bounded by monitoring coverage entropy
- $\text{Respond}$: Mean time to respond (MTTR), bounded by runbook coverage and staffing
- $\text{Recover}$: RTO $+$ RPO bounded by replication topology
- $g$: Concentration / dependency graph properties

Each term is measurable, auditable, and must be re-evaluated at least annually (DORA Art. 6.5). Translate regulation into metrics, metrics into monitoring, monitoring into evidence — that is the operational implementation of DORA.

---
