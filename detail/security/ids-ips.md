# The Mathematics of IDS/IPS — Detection Theory and Pattern Matching

> *Intrusion Detection Systems are fundamentally classification engines. Their effectiveness is governed by statistical detection theory — sensitivity, specificity, and the base rate fallacy that makes false positives the dominant operational challenge.*

---

## 1. Detection Rate Mathematics

### Confusion Matrix

Every detection event falls into one of four categories:

|  | Actually Malicious | Actually Benign |
|:---:|:---:|:---:|
| **Alert Fired** | True Positive (TP) | False Positive (FP) |
| **No Alert** | False Negative (FN) | True Negative (TN) |

### Core Metrics

$$\text{Sensitivity (TPR)} = \frac{TP}{TP + FN} = P(\text{alert} \mid \text{attack})$$

$$\text{Specificity (TNR)} = \frac{TN}{TN + FP} = P(\text{no alert} \mid \text{benign})$$

$$\text{False Positive Rate (FPR)} = \frac{FP}{FP + TN} = 1 - \text{Specificity}$$

$$\text{Precision (PPV)} = \frac{TP}{TP + FP} = P(\text{attack} \mid \text{alert})$$

### Worked Example

A Suricata deployment monitors 1 million events/day. Attack prevalence: 0.01% (100 attacks/day).

| Metric | Value |
|:---|:---:|
| Sensitivity | 95% (catches 95 of 100 attacks) |
| Specificity | 99.9% (0.1% false positive rate) |
| True Positives | 95 |
| False Positives | $999,900 \times 0.001 = 999.9 \approx 1000$ |
| Precision | $\frac{95}{95 + 1000} = 8.7\%$ |

**The base rate fallacy:** Even with 99.9% specificity, only 8.7% of alerts are real attacks. This is why SOC analysts suffer alert fatigue.

---

## 2. The Base Rate Problem

### Bayes' Theorem Applied to IDS

$$P(\text{attack} \mid \text{alert}) = \frac{P(\text{alert} \mid \text{attack}) \times P(\text{attack})}{P(\text{alert})}$$

Where:

$$P(\text{alert}) = P(\text{alert} \mid \text{attack}) \times P(\text{attack}) + P(\text{alert} \mid \text{benign}) \times P(\text{benign})$$

### Precision as a Function of Prevalence

$$\text{Precision} = \frac{\text{TPR} \times \pi}{\text{TPR} \times \pi + \text{FPR} \times (1 - \pi)}$$

Where $\pi$ is the attack prevalence.

| Prevalence ($\pi$) | TPR=95%, FPR=0.1% | TPR=95%, FPR=0.01% |
|:---:|:---:|:---:|
| 10% | 99.1% | 99.9% |
| 1% | 90.5% | 99.0% |
| 0.1% | 48.7% | 90.5% |
| 0.01% | 8.7% | 48.7% |
| 0.001% | 0.9% | 8.7% |

**Key insight:** Reducing FPR by 10x has the same effect as increasing prevalence by 10x.

---

## 3. False Positive Rate vs Rule Count

### The Rule Accumulation Problem

Each rule has an independent false positive probability $p_i$. With $n$ rules:

$$P(\text{at least one FP per event}) = 1 - \prod_{i=1}^{n} (1 - p_i)$$

If all rules have equal FP rate $p$:

$$P(\text{any FP}) = 1 - (1 - p)^n \approx np \quad \text{for small } p$$

### Worked Example

| Rules ($n$) | FP Rate per Rule ($p$) | FP per Event | FP per Million Events |
|:---:|:---:|:---:|:---:|
| 100 | $10^{-5}$ | 0.001 | 1,000 |
| 1,000 | $10^{-5}$ | 0.01 | 10,000 |
| 10,000 | $10^{-5}$ | 0.095 | 95,000 |
| 30,000 | $10^{-5}$ | 0.26 | 260,000 |

Suricata's ET Open ruleset has ~30,000 rules. Even with $p = 10^{-5}$, you get 260K false positives per million events.

### The Tuning Imperative

Disabling the noisiest 5% of rules often eliminates 80% of false positives (Pareto principle applied to IDS).

---

## 4. Suricata Multi-Pattern Matching — Aho-Corasick Algorithm

### The Problem

Suricata must match thousands of byte patterns against every packet. Naive approach:

$$T_{naive} = O(n \times m \times L)$$

Where $n$ = number of patterns, $m$ = average pattern length, $L$ = packet length.

### Aho-Corasick Solution

Aho-Corasick builds a **finite automaton** from all patterns simultaneously:

$$T_{AC} = O(L + z)$$

Where $L$ = input length and $z$ = number of matches. **Independent of pattern count.**

### Automaton Construction

1. **Goto function**: Trie of all patterns — $O(\sum m_i)$ construction
2. **Failure function**: Suffix links (like KMP) — $O(\sum m_i)$ construction
3. **Output function**: Which patterns match at each state

### Performance Comparison

| Patterns | Packet Size | Naive ($\mu$s) | Aho-Corasick ($\mu$s) | Speedup |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 1500 B | 150 | 3 | 50x |
| 1,000 | 1500 B | 1,500 | 3 | 500x |
| 10,000 | 1500 B | 15,000 | 3 | 5,000x |
| 30,000 | 1500 B | 45,000 | 3 | 15,000x |

### Memory Cost

The automaton uses significant memory:

$$\text{Memory} \approx |\Sigma| \times |Q| \times \text{pointer size}$$

Where $|\Sigma| = 256$ (byte alphabet) and $|Q|$ = number of states.

For 30,000 rules: ~200-500 MB of automaton memory.

---

## 5. Snort vs Suricata Architecture

### Processing Models

| Feature | Snort 2 | Snort 3 | Suricata |
|:---|:---|:---|:---|
| Threading | Single-threaded | Multi-threaded | Multi-threaded |
| Throughput | ~200 Mbps | ~1 Gbps | ~10 Gbps |
| Pattern matching | Aho-Corasick | Hyperscan | Aho-Corasick |
| Rule language | Snort rules | Snort rules | Suricata/Snort rules |

### Throughput Formula

$$\text{Max throughput} = \frac{\text{CPU cycles/sec} \times \text{cores}}{\text{cycles/packet}}$$

For Suricata on a 4-core 3 GHz machine:

$$\text{Max} = \frac{3 \times 10^9 \times 4}{10^4 \text{ cycles/pkt}} = 1.2 \times 10^6 \text{ pkt/s}$$

At average 500 bytes/packet: $1.2M \times 500 \times 8 = 4.8 \text{ Gbps}$

---

## 6. Detection Methodologies

### Signature-Based Detection

$$\text{Match}(P, S) = \begin{cases} 1 & \text{if pattern } S \text{ found in packet } P \\ 0 & \text{otherwise} \end{cases}$$

Strengths: High precision for known attacks, low FP rate per rule.
Weakness: Zero detection of novel attacks (zero-days).

### Anomaly-Based Detection

Model normal behavior as a distribution $N(\mu, \sigma)$. Alert when:

$$|x - \mu| > k\sigma$$

Where $k$ is the sensitivity threshold (typically $k = 2$ or $3$).

| Threshold ($k$) | Normal Traffic Flagged | Detection Sensitivity |
|:---:|:---:|:---:|
| 1 | 31.7% | Very high (too many FPs) |
| 2 | 4.6% | High |
| 3 | 0.3% | Medium |
| 4 | 0.006% | Low (may miss attacks) |

### Statistical Features for Anomaly Detection

| Feature | Normal Range | Attack Indicator |
|:---|:---|:---|
| Bytes per flow | $\mu \pm 2\sigma$ | Exfiltration: $\gg \mu$ |
| Connections per minute | $\mu \pm 2\sigma$ | Scan: $\gg \mu$ |
| DNS query length | 10-50 chars | Tunnel: >100 chars |
| Packet size entropy | High (varied) | Low (repetitive = DoS) |

---

## 7. ROC Curves — Optimizing Detection Threshold

### The ROC Curve

A Receiver Operating Characteristic curve plots TPR vs FPR as the detection threshold varies:

$$\text{AUC} = \int_0^1 \text{TPR}(\text{FPR}) \, d(\text{FPR})$$

| AUC | Interpretation |
|:---:|:---|
| 1.0 | Perfect classifier |
| 0.9-1.0 | Excellent |
| 0.8-0.9 | Good |
| 0.7-0.8 | Fair |
| 0.5 | Random guess (useless) |

### Threshold Selection

The optimal threshold depends on the **cost ratio**:

$$\text{Cost} = C_{FP} \times FP + C_{FN} \times FN$$

If missing an attack costs 100x more than a false positive ($C_{FN} = 100 \times C_{FP}$), optimize for high sensitivity even at the cost of more false positives.

---

## 8. IPS Latency Budget

### Inline Mode Constraints

An IPS must inspect packets before forwarding. Latency budget:

$$T_{inspect} < T_{budget}$$

| Network | Typical $T_{budget}$ | Consequence of Overrun |
|:---|:---:|:---|
| Data center | < 100 $\mu$s | Application timeout |
| Enterprise LAN | < 1 ms | User-perceptible delay |
| Internet edge | < 5 ms | Acceptable |
| Cloud WAF | < 10 ms | Acceptable |

### Fail-Open vs Fail-Closed

$$\text{Availability risk} = \begin{cases} \text{Low} & \text{fail-open (bypass on overload)} \\ \text{High} & \text{fail-closed (drop on overload)} \end{cases}$$

$$\text{Security risk} = \begin{cases} \text{High} & \text{fail-open (attacks pass through)} \\ \text{Low} & \text{fail-closed (no uninspected traffic)} \end{cases}$$

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Sensitivity/Specificity | Conditional probability | Detection quality |
| Bayes' theorem | Posterior probability | Precision (PPV) |
| $(1-p)^n$ accumulation | Binomial probability | FP rate vs rule count |
| Aho-Corasick | Finite automaton | Multi-pattern matching |
| $|x - \mu| > k\sigma$ | Statistical threshold | Anomaly detection |
| ROC / AUC | Integral | Threshold optimization |
| Cost function | Optimization | Alert prioritization |

## Prerequisites

- Bayes' theorem, pattern matching, statistical baselines, false positive rates

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Signature matching (Aho-Corasick) | O(n + m) | O(m) |
| Anomaly baseline update | O(n) | O(w) |
| Alert correlation | O(a log a) | O(a) |

---

*An IDS is only as good as its mathematics — a 99.9% accurate detector still drowns analysts in false positives when attack prevalence is low. Understanding the base rate is the difference between a useful tool and an expensive noise generator.*
