# The Mathematics of Red Team Ops — Adversary Modeling and Detection Theory

> *Red team operations are adversary simulations governed by detection theory, graph-based attack modeling, and information-theoretic covert channels. The mathematics of C2 beaconing analysis, kill chain probability, evasion as a signal-detection problem, and OPSEC as information leakage quantification provide the theoretical foundation for both offense and defense.*

---

## 1. Kill Chain Probability (Sequential Success)

### Cyber Kill Chain Model

The Lockheed Martin kill chain defines seven sequential stages. Overall success requires all stages:

$$P(\text{compromise}) = \prod_{i=1}^{7} P(\text{stage}_i)$$

| Stage | Description | Typical $P$ (Mature Org) | Typical $P$ (Immature Org) |
|:---|:---:|:---:|:---:|
| Reconnaissance | Target enumeration | 0.95 | 0.99 |
| Weaponization | Payload creation | 0.90 | 0.95 |
| Delivery | Payload transmission | 0.40 | 0.80 |
| Exploitation | Code execution | 0.50 | 0.85 |
| Installation | Persistence | 0.60 | 0.90 |
| C2 | Command channel | 0.50 | 0.85 |
| Actions | Objective completion | 0.70 | 0.90 |

Composite probability:

$$P_{\text{mature}} = 0.95 \times 0.90 \times 0.40 \times 0.50 \times 0.60 \times 0.50 \times 0.70 = 0.036$$

$$P_{\text{immature}} = 0.99 \times 0.95 \times 0.80 \times 0.85 \times 0.90 \times 0.85 \times 0.90 = 0.418$$

### Defender Advantage

A defender needs to break only one stage to prevent compromise:

$$P(\text{defense}) = 1 - P(\text{compromise}) = 1 - \prod_{i=1}^{7} P(\text{stage}_i)$$

Each additional defensive layer provides multiplicative protection.

---

## 2. C2 Beaconing Analysis (Time Series Detection)

### Beacon Interval Statistics

A C2 beacon with interval $\mu$ and jitter $j$ produces callback times:

$$t_i = t_{i-1} + \mu \times (1 + U(-j, j))$$

Where $U(-j, j)$ is a uniform random variable in $[-j, j]$.

### Detection via Periodicity

The autocorrelation function reveals hidden periodicity:

$$R(\tau) = \frac{1}{N} \sum_{i=1}^{N-\tau} (t_i - \bar{t})(t_{i+\tau} - \bar{t})$$

Peak at $\tau = \mu$ indicates beaconing even with jitter.

### Jitter Effectiveness

| Jitter % | Detection Difficulty | Autocorrelation Peak |
|:---|:---:|:---:|
| 0% | Trivial | Sharp spike at $\mu$ |
| 10% | Easy | Clear peak |
| 25% | Moderate | Visible peak |
| 50% | Hard | Broad, reduced peak |
| 75%+ | Very hard | Near noise floor |

### Entropy-Based Detection

Shannon entropy of inter-arrival times:

$$H = -\sum_{i} p_i \log_2(p_i)$$

| Traffic Type | Entropy (bits) | Pattern |
|:---|:---:|:---:|
| Fixed beacon | 0 | Single interval |
| Low jitter beacon | 2-4 | Clustered around $\mu$ |
| High jitter beacon | 5-7 | Spread but bounded |
| Legitimate browsing | 8-10 | High variability |
| Random/encrypted | ~12 | Maximum entropy |

Beacons typically have lower entropy than legitimate traffic, enabling statistical detection.

---

## 3. Evasion as Signal Detection (Neyman-Pearson)

### EDR Detection Model

Endpoint Detection and Response operates as a binary classifier:

$$\text{Decision} = \begin{cases} H_1 \text{ (malicious)} & \text{if } L(\mathbf{x}) > \tau \\ H_0 \text{ (benign)} & \text{if } L(\mathbf{x}) \leq \tau \end{cases}$$

Where $L(\mathbf{x})$ is the likelihood ratio of observed features and $\tau$ is the detection threshold.

### ROC Curve Trade-offs

| Metric | Formula | Defender Wants |
|:---|:---:|:---:|
| True Positive Rate | $P(\text{alert} \mid \text{attack})$ | High |
| False Positive Rate | $P(\text{alert} \mid \text{benign})$ | Low |
| Precision | $\frac{TP}{TP + FP}$ | High |
| Recall | $\frac{TP}{TP + FN}$ | High |

The red team's goal is to operate in the region where:

$$P(\text{detect} \mid \text{attack}) \approx P(\text{false alarm} \mid \text{benign})$$

Making malicious activity statistically indistinguishable from benign behavior.

### Feature Space Evasion

EDR features typically include:

$$\mathbf{x} = (x_{\text{process}}, x_{\text{network}}, x_{\text{file}}, x_{\text{registry}}, x_{\text{memory}})$$

Evasion requires matching the benign distribution for each feature:

$$P(\mathbf{x}_{\text{attack}}) \approx P(\mathbf{x}_{\text{benign}}) \quad \forall \text{ features}$$

Living-off-the-land inherently achieves this because the process features match legitimate system tools.

---

## 4. MITRE ATT&CK Coverage (Graph Coverage Problem)

### ATT&CK Matrix as Bipartite Graph

The ATT&CK matrix forms a bipartite graph $G = (T, D, E)$:

$$T = \text{techniques}, \quad D = \text{data sources/detections}, \quad E = \text{detection relationships}$$

### Detection Coverage

$$C = \frac{|\{t \in T \mid \exists d \in D : (t, d) \in E \wedge d \text{ is enabled}\}|}{|T|}$$

| Maturity Level | Coverage | Techniques Detected |
|:---|:---:|:---:|
| Level 1 (Basic) | 15-25% | Commodity malware |
| Level 2 (Managed) | 40-60% | Known TTPs |
| Level 3 (Optimized) | 70-85% | Advanced techniques |
| Level 4 (Adaptive) | 85-95% | Novel combinations |

### Red Team Technique Selection

Optimal technique selection minimizes detection probability:

$$\min_{T' \subseteq T} \sum_{t \in T'} P(\text{detect}(t)) \quad \text{subject to} \quad T' \text{ achieves objective}$$

This is a constrained optimization over the ATT&CK technique graph.

---

## 5. Lateral Movement Graph Theory (Network Traversal)

### Trust Graph

An Active Directory environment defines a trust graph $G = (H, E, W)$:

$$H = \text{hosts}, \quad E = \text{trust/credential relationships}$$

Edge weight $W(e)$ represents the cost/risk of traversal:

$$W(e) = f(\text{detection\_risk}, \text{credential\_type}, \text{network\_distance})$$

### Credential Propagation

If host $h_i$ has cached credentials for user $u$ who has access to host $h_j$:

$$h_i \xrightarrow{u} h_j$$

The reachable set from initial foothold $h_0$ with credential set $\mathcal{C}$:

$$\text{Reach}(h_0, \mathcal{C}) = \{h \mid \exists \text{ path from } h_0 \text{ using creds in } \mathcal{C}\}$$

### Kerberoasting Economics

Time to crack a Kerberos TGS ticket with password entropy $H$:

$$T_{\text{crack}} = \frac{2^H}{R_{\text{hash}}}$$

| Password Entropy | Hashcat Speed (RTX 4090) | Time to Crack |
|:---|:---:|:---:|
| 20 bits | $7 \times 10^9$ H/s | Instant |
| 30 bits | $7 \times 10^9$ H/s | 0.15 seconds |
| 40 bits | $7 \times 10^9$ H/s | 157 seconds |
| 50 bits | $7 \times 10^9$ H/s | 44 hours |
| 60 bits | $7 \times 10^9$ H/s | 5.2 years |

Service accounts with weak passwords ($H < 40$ bits) are cracked within minutes.

---

## 6. Covert Channel Capacity (Information Theory)

### DNS Exfiltration Bandwidth

DNS covert channel capacity:

$$B_{\text{DNS}} = \frac{L_{\text{label}} \times N_{\text{labels}} \times E_{\text{encoding}}}{T_{\text{interval}}}$$

| Parameter | Value | Notes |
|:---|:---:|:---:|
| Max label length | 63 chars | DNS specification |
| Max domain length | 253 chars | Total FQDN |
| Usable per query | ~180 chars | After overhead |
| Base32 efficiency | 5 bits/char | Encoding overhead |
| Query interval | 1-10 seconds | OPSEC constraint |

$$B_{\text{max}} = \frac{180 \times 5}{8 \times 1} = 112.5 \text{ bytes/second}$$

### HTTPS Covert Channel

$$B_{\text{HTTPS}} = \frac{P_{\text{payload}} \times R_{\text{requests}}}{1 + O_{\text{TLS}}}$$

Typical C2 over HTTPS: 10 KB per request at 1 request per 30 seconds = 333 bytes/second.

### Channel Detection Difficulty

| Channel | Bandwidth | Stealth | Detection Complexity |
|:---|:---:|:---:|:---:|
| DNS TXT | 100 B/s | Medium | DNS analytics |
| HTTPS | 300+ B/s | High | TLS inspection |
| ICMP | 50 B/s | Low | Payload analysis |
| DNS over HTTPS | 100 B/s | Very high | Encrypted DNS |
| Steganography | 10 B/s | Very high | Statistical analysis |

---

## 7. Persistence Survivability (Reliability Theory)

### Persistence Mechanism Reliability

Each persistence mechanism has a survival probability $p_i$ against defender cleanup:

$$P(\text{survive}) = 1 - \prod_{i=1}^{n} (1 - p_i)$$

With $n$ independent persistence mechanisms:

| Mechanisms | $p_i = 0.3$ each | $p_i = 0.5$ each | $p_i = 0.7$ each |
|:---|:---:|:---:|:---:|
| 1 | 0.30 | 0.50 | 0.70 |
| 2 | 0.51 | 0.75 | 0.91 |
| 3 | 0.66 | 0.875 | 0.97 |
| 5 | 0.83 | 0.97 | 0.998 |

### MTTR (Mean Time To Re-establish)

$$\text{MTTR} = T_{\text{detect\_loss}} + T_{\text{activate\_backup}} + T_{\text{verify\_access}}$$

Redundant persistence reduces MTTR by providing immediate fallback:

$$\text{MTTR}_{\text{redundant}} = T_{\text{detect\_loss}} + T_{\text{check\_backup}}$$

---

## 8. OPSEC as Information Leakage (Entropy Model)

### Information Leakage Model

Each operator action leaks information to defenders. Total leakage:

$$I_{\text{leaked}} = \sum_{a \in \text{actions}} H(a \mid \text{defender's prior knowledge})$$

### Attribution Difficulty

Attribution requires correlating multiple weak signals:

$$P(\text{attribution}) = P\left(\bigcap_{i=1}^{n} \text{signal}_i \text{ traced}\right) = \prod_{i=1}^{n} P(\text{trace}_i)$$

| Signal Type | Traceability | Mitigation |
|:---|:---:|:---:|
| Source IP | High ($p = 0.8$) | VPN + redirectors |
| Domain registration | Medium ($p = 0.5$) | Privacy services |
| TLS certificate | Medium ($p = 0.4$) | Let's Encrypt automation |
| Malware hash | High ($p = 0.9$) | Per-target compilation |
| C2 traffic pattern | Low ($p = 0.2$) | Domain fronting |
| Working hours | Medium ($p = 0.4$) | Automated scheduling |

With 6 independent signals and mitigations:

$$P(\text{attribution}) = 0.8 \times 0.5 \times 0.4 \times 0.9 \times 0.2 \times 0.4 = 0.0115$$

Each mitigation layer (redirectors, privacy registration, etc.) reduces attribution probability multiplicatively.

### Operational Tempo and Detection

Detection probability increases with operational tempo:

$$P(\text{detect}) = 1 - e^{-\lambda \times N_{\text{actions}} \times V_{\text{visibility}}}$$

Where $\lambda$ is the detection rate per visible action. Slower tempo ($N$ small) with low-visibility techniques ($V$ small) minimizes detection.

---

*Red team operations are fundamentally an adversarial game played in the detection-theory domain. The mathematics reveal that defender advantage comes from serial dependence in the kill chain (breaking any link stops the attack), while attacker advantage comes from parallel persistence (any surviving mechanism maintains access). OPSEC is information-theoretic — minimizing the mutual information between operator actions and defender observations. The equilibrium between offense and defense is determined by the relative costs of detection versus evasion at each stage.*

## Prerequisites

- Probability theory (conditional probability, Bayes' theorem, independence)
- Information theory (entropy, mutual information, channel capacity)
- Graph theory (directed graphs, reachability, shortest paths)

## Complexity

- **Beginner:** Computing kill chain probabilities, understanding beaconing detection via fixed intervals
- **Intermediate:** Analyzing lateral movement graphs, estimating covert channel bandwidth, calculating persistence redundancy
- **Advanced:** Applying Neyman-Pearson detection theory to evasion design, quantifying OPSEC information leakage, and optimizing technique selection against known detection coverage using constrained optimization
