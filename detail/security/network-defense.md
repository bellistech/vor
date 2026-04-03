# The Mathematics of Network Defense — Segmentation, Monitoring, and Response

> *Network defense is topology engineering: segmentation reduces blast radius polynomially, monitoring coverage determines detection probability, and traffic analysis uses statistical models to distinguish normal from malicious flows.*

---

## 1. Network Segmentation — Graph Partitioning

### Flat Network Risk

In an unsegmented network with $N$ hosts, every host can reach every other:

$$\text{Attack paths} = N \times (N - 1) = N^2 - N$$

### Segmented Network

With $k$ segments of size $n_i$:

$$\text{Intra-segment paths} = \sum_{i=1}^{k} n_i(n_i - 1)$$

$$\text{Cross-segment paths} = \text{controlled by firewall rules}$$

| Hosts | Segments | Intra-Segment Paths | Reduction from Flat |
|:---:|:---:|:---:|:---:|
| 100 | 1 (flat) | 9,900 | 0% |
| 100 | 5 ($n=20$) | 1,900 | 80.8% |
| 100 | 10 ($n=10$) | 900 | 90.9% |
| 100 | 25 ($n=4$) | 300 | 97.0% |
| 100 | 100 ($n=1$) | 0 | 100% (zero trust) |

### Optimal Segment Size

Minimizing intra-segment paths with $k$ equal segments of size $N/k$:

$$\text{Paths} = k \times \frac{N}{k} \times \left(\frac{N}{k} - 1\right) = N\left(\frac{N}{k} - 1\right)$$

Doubling $k$ roughly halves the attack surface.

---

## 2. Traffic Baseline — Flow Analysis

### NetFlow/IPFIX Metrics

Each flow record captures:

$$\text{Flow} = (\text{src IP}, \text{dst IP}, \text{src port}, \text{dst port}, \text{proto}, \text{bytes}, \text{packets}, \text{duration})$$

### Normal Traffic Distribution

| Metric | Normal Range | Alert Threshold |
|:---|:---|:---|
| Bytes per flow | $\mu \pm 3\sigma$ | > $\mu + 5\sigma$ |
| Flows per host per hour | 100-10,000 | > 50,000 |
| Unique destinations per host | 10-500 | > 5,000 (scan) |
| DNS queries per host per hour | 50-2,000 | > 10,000 (tunnel) |
| Connection duration | 1s-300s | > 3600s (C2 beacon) |

### Traffic Volume Anomaly

$$Z = \frac{x - \mu}{\sigma}$$

| Z-score | $P(\text{normal traffic})$ | Action |
|:---:|:---:|:---|
| $|Z| < 2$ | 95.4% | Normal |
| $2 \leq |Z| < 3$ | 4.3% | Investigate |
| $3 \leq |Z| < 4$ | 0.26% | Alert |
| $|Z| \geq 4$ | 0.006% | Critical alert |

---

## 3. DNS Security — Detection Mathematics

### DNS Tunneling Detection

DNS tunnels encode data in subdomain labels:

$$\text{Bandwidth} = \frac{|\text{subdomain}| \times 8}{T_{query}} \text{ bits/second}$$

Maximum subdomain: 253 characters. At 1 query/second: ~2 Kbps.

### Entropy of DNS Queries

Normal DNS queries have low entropy (human-readable):

$$H_{normal} \approx 2.5-3.5 \text{ bits/char}$$

DNS tunnel queries have high entropy (encoded binary):

$$H_{tunnel} \approx 4.5-5.5 \text{ bits/char}$$

### DGA (Domain Generation Algorithm) Detection

DGA domains have high character entropy and abnormal n-gram distributions:

$$\text{DGA score} = w_1 \times H_{chars} + w_2 \times (1 - P_{bigram}) + w_3 \times \text{length}$$

| Domain Type | Entropy | Bigram Score | DGA Score |
|:---|:---:|:---:|:---:|
| google.com | 2.8 | 0.85 | 0.15 (benign) |
| amazon.com | 2.8 | 0.82 | 0.18 (benign) |
| xkcd7fp2q.com | 4.2 | 0.12 | 0.87 (DGA) |
| a8f3kq2m1.net | 4.5 | 0.08 | 0.92 (DGA) |

---

## 4. Intrusion Kill Chain — Detection Points

### Lockheed Martin Kill Chain

| Phase | Detection Opportunity | $P(\text{detect})$ |
|:---:|:---|:---:|
| 1. Reconnaissance | DNS/port scan detection | 0.30 |
| 2. Weaponization | Threat intel (IOC match) | 0.10 |
| 3. Delivery | Email/web gateway | 0.60 |
| 4. Exploitation | IDS/endpoint detection | 0.40 |
| 5. Installation | AV/EDR | 0.50 |
| 6. C2 | Network monitoring | 0.35 |
| 7. Actions on Objective | DLP/anomaly detection | 0.25 |

### Cumulative Detection Probability

$$P(\text{detect by phase } k) = 1 - \prod_{i=1}^{k} (1 - P_i)$$

| By Phase | Cumulative $P(\text{detect})$ |
|:---:|:---:|
| 1 (Recon) | 30.0% |
| 3 (Delivery) | 74.8% |
| 5 (Installation) | 90.4% |
| 7 (Actions) | 95.7% |

Defense in depth across all 7 phases gives 95.7% detection — but only if all monitoring is active and tuned.

---

## 5. Firewall Rule Optimization

### Rule Complexity

$$C_{firewall} = n_{rules} \times n_{objects} \times n_{zones}$$

| Environment | Rules | Objects | Zones | Complexity |
|:---|:---:|:---:|:---:|:---:|
| Small office | 50 | 20 | 3 | 3,000 |
| Enterprise | 5,000 | 500 | 20 | 50M |
| Data center | 50,000 | 5,000 | 50 | 12.5B |

### Rule Shadowing

Rule $r_j$ is shadowed by $r_i$ (where $i < j$) if:

$$\text{match}(r_i) \supseteq \text{match}(r_j) \quad \text{AND} \quad \text{action}(r_i) = \text{action}(r_j)$$

Shadowed rules are dead code — they never trigger. Typical firewall rule audits find 10-30% shadowed rules.

### Rule Hit Analysis

$$\text{Unused rule} \iff \text{hit count} = 0 \text{ over } T_{observation}$$

| Observation Period | Rules with 0 Hits | Action |
|:---:|:---:|:---|
| 7 days | 40-60% | Too short to conclude |
| 30 days | 20-40% | Review candidates |
| 90 days | 10-25% | Likely removable |
| 365 days | 5-15% | Remove or justify |

---

## 6. Encrypted Traffic Analysis (ETA)

### The Problem

With 80-90% of traffic encrypted (TLS), traditional deep packet inspection fails:

$$P(\text{content inspection}) = 1 - P(\text{encrypted}) \approx 0.1-0.2$$

### Metadata Analysis

Without decryption, analyze:

$$\text{Features} = (\text{flow size}, \text{duration}, \text{packet timing}, \text{TLS metadata})$$

| Feature | Normal HTTPS | C2 Traffic | Exfiltration |
|:---|:---|:---|:---|
| Flow duration | 1-60s | 60-3600s (beaconing) | 10-600s |
| Packet intervals | Variable | Periodic ($\sigma < 1$s) | Bursty |
| Upload/download ratio | 1:10 (download-heavy) | 1:1 | 10:1 (upload-heavy) |
| Certificate | Valid, known CA | Self-signed, short-lived | Valid |
| JA3 fingerprint | Browser-like | Unique/uncommon | Browser-like |

### JA3 Fingerprint

TLS client fingerprint from ClientHello:

$$\text{JA3} = \text{MD5}(\text{TLSVersion}, \text{Ciphers}, \text{Extensions}, \text{EllipticCurves}, \text{PointFormats})$$

Known malware JA3 hashes can be matched without decrypting traffic.

---

## 7. Network Monitoring Coverage

### Coverage Model

$$\text{Coverage} = \frac{|\text{monitored links}|}{|\text{total links}|}$$

| Monitoring Point | Typical Coverage | Blind Spots |
|:---|:---:|:---|
| Internet gateway only | 30-40% | East-west, inter-VLAN |
| Gateway + core switches | 60-70% | Intra-VLAN, local |
| Full TAP/SPAN deployment | 90-95% | Encrypted tunnels |
| Agent-based (per host) | 95-99% | Unmanaged devices |

### Sensor Placement Optimization

Given a network graph $G = (V, E)$ and budget for $k$ sensors:

$$\text{Maximize } |\{e \in E : e \text{ passes through a sensor}\}|$$

This is a **vertex cover / set cover** problem — NP-hard in general, but heuristics (greedy placement at highest-degree nodes) achieve >90% of optimal.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $N(N-1)$ paths | Quadratic | Flat network risk |
| $N(N/k - 1)$ | Linear reduction | Segmentation benefit |
| $Z = (x-\mu)/\sigma$ | Normal distribution | Traffic anomaly |
| Shannon entropy $H$ | Information theory | DNS tunnel detection |
| $1 - \prod(1-P_i)$ | Kill chain probability | Defense in depth |
| Rule shadowing $\supseteq$ | Set containment | Firewall optimization |
| Vertex cover | Graph theory | Sensor placement |

---

*Network defense is the art of topology — segmentation reduces blast radius, monitoring increases detection probability, and traffic analysis reveals attacks hiding in encrypted streams. The mathematics quantify each layer's contribution to overall security posture.*
