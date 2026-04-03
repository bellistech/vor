# The Mathematics of nmap — Scan Timing, Host Discovery, and Port State Probability

> *nmap is a network scanner built on statistical probing — it sends carefully crafted packets and infers host/port state from responses (or lack thereof). The math covers scan timing algorithms, parallelism tuning, OS fingerprint matching, and the combinatorial explosion of scanning large networks.*

---

## 1. Scan Space — Combinatorial Scale

### The Problem

Scanning a network involves probing every (host, port) combination:

$$\text{Total probes} = N_{hosts} \times N_{ports}$$

### Scale Examples

| Target | Hosts | Ports | Total Probes |
|:---|:---:|:---:|:---:|
| Single host, top 1000 | 1 | 1,000 | 1,000 |
| /24 subnet, top 1000 | 254 | 1,000 | 254,000 |
| /16 subnet, top 1000 | 65,534 | 1,000 | 65,534,000 |
| /24 subnet, all ports | 254 | 65,535 | 16,645,890 |
| /16 subnet, all ports | 65,534 | 65,535 | 4,294,705,290 |

A /16 all-port scan is ~4.3 billion probes — this is why timing optimization matters.

---

## 2. Timing Templates — Probe Rate Math

### The `-T` Templates

| Template | Name | Parallelism | Probe Timeout | Probe Rate |
|:---:|:---|:---:|:---:|:---:|
| T0 | Paranoid | Serial | 5 min | ~1 probe/5 min |
| T1 | Sneaky | Serial | 15 sec | ~1 probe/15 sec |
| T2 | Polite | Serial | 1 sec | ~10 probes/sec |
| T3 | Normal | Parallel | 1 sec | ~100-1000 probes/sec |
| T4 | Aggressive | High parallel | 500 ms | ~1000-10000 probes/sec |
| T5 | Insane | Max parallel | 300 ms | ~10000+ probes/sec |

### Scan Duration

$$T_{scan} = \frac{N_{probes}}{R_{probes/sec}} + N_{hosts} \times T_{host\_overhead}$$

| Target | Probes | T3 (1000/s) | T4 (5000/s) | T5 (10000/s) |
|:---|:---:|:---:|:---:|:---:|
| 1 host, 1000 ports | 1,000 | 1 sec | 0.2 sec | 0.1 sec |
| /24, 1000 ports | 254,000 | 4.2 min | 51 sec | 25 sec |
| /24, all ports | 16.6M | 4.6 hours | 55 min | 28 min |
| /16, 1000 ports | 65.5M | 18.2 hours | 3.6 hours | 1.8 hours |

---

## 3. SYN Scan — State Inference

### TCP SYN Scan (`-sS`)

$$\text{Port state} = \begin{cases} \text{open} & \text{if SYN-ACK received} \\ \text{closed} & \text{if RST received} \\ \text{filtered} & \text{if no response (timeout)} \end{cases}$$

### Response Probability Model

For each probe, the probability of each state:

$$P(\text{open}) = P(\text{service listening}) \times P(\text{not filtered})$$

$$P(\text{closed}) = (1 - P(\text{service listening})) \times P(\text{not filtered})$$

$$P(\text{filtered}) = P(\text{firewall drops SYN})$$

### Retransmission Logic

nmap retransmits on timeout:

$$R_{max} = \begin{cases} 10 & \text{T0-T2} \\ 6 & \text{T3} \\ 2 & \text{T4-T5} \end{cases}$$

**Filtered port cost:**

$$T_{filtered} = \sum_{i=0}^{R_{max}} T_{timeout} \times 2^i \approx T_{timeout} \times 2^{R_{max}+1}$$

This is why filtered ports dramatically slow down scans.

---

## 4. Host Discovery — Alive Detection

### Default Discovery Probes

nmap sends multiple probe types to maximize detection:

| Probe | Packet | Response Expected |
|:---|:---|:---|
| ICMP Echo | Ping | ICMP Echo Reply |
| TCP SYN to 443 | SYN | SYN-ACK or RST |
| TCP ACK to 80 | ACK | RST |
| ICMP Timestamp | Timestamp request | Timestamp reply |

### Detection Probability

$$P(\text{alive}) = 1 - \prod_{i=1}^{P} (1 - P(\text{response}_i))$$

If each probe has independent 70% chance of being blocked:

$$P(\text{alive}) = 1 - 0.3^4 = 1 - 0.0081 = 99.19\%$$

Four independent probes make detection highly reliable even with aggressive filtering.

### Host Discovery Speed

$$T_{discovery} = \frac{N_{hosts}}{R_{parallel}} \times T_{round}$$

For a /24 with T4 timing: $254 / 100 \times 0.5 \approx 1.3$ seconds.

---

## 5. OS Fingerprinting — Matching Probability

### The Method

nmap sends 16 specifically crafted probes and compares responses against a database of ~6,000 known fingerprints.

### Matching Score

$$\text{Score} = \frac{\text{matching fields}}{\text{total fields}} \times 100\%$$

| Score | Confidence | Output |
|:---:|:---|:---|
| 100% | Exact match | "OS: Linux 5.15" |
| 90-99% | High confidence | "OS: Linux 5.x (95%)" |
| 80-89% | Moderate | "Aggressive guess: Linux 5.x (85%)" |
| < 80% | Low | "No exact OS match" |

### Required Conditions

OS detection needs at least one open port and one closed port:

$$\text{OS detect viable} = (\exists \text{ open port}) \wedge (\exists \text{ closed port})$$

This is because some fingerprint tests specifically probe open and closed port behaviors.

---

## 6. Service Version Detection — Banner Analysis

### Probe Sequence

nmap sends increasingly specific probes until a match:

$$\text{Probes tried} = \min(N_{probes}, \text{first match index})$$

### Version Intensity (`--version-intensity 0-9`)

| Intensity | Probes Sent | Coverage | Speed |
|:---:|:---:|:---:|:---:|
| 0 | Banner grab only | ~50% | Very fast |
| 2 | Common probes | ~70% | Fast |
| 5 (default) | Moderate set | ~85% | Normal |
| 7 | Extended set | ~95% | Slow |
| 9 | All probes | ~99% | Very slow |

### Time per Port

$$T_{version} = N_{probes} \times (T_{send} + T_{wait\_response})$$

Typical: 1-5 seconds per open port at default intensity.

---

## 7. NSE Script Scanning — Execution Model

### Script Categories and Execution Time

$$T_{scripts} = \sum_{s \in \text{active}} T_s(p)$$

Where $T_s(p)$ = time for script $s$ on port $p$.

| Category | Typical Scripts | Time per Host |
|:---|:---:|:---:|
| default | ~50 | 5-30 sec |
| safe | ~80 | 10-60 sec |
| vuln | ~100 | 30-300 sec |
| brute | ~30 | 60-3600 sec |
| all | ~600 | Minutes to hours |

### Parallelism

Scripts run in parallel with configurable concurrency:

$$T_{total} = \frac{\sum T_s}{P_{parallel}} + T_{sequential}$$

Some scripts have dependencies and must run sequentially.

---

## 8. Output Data Sizing

### Results Volume

$$S_{output} = N_{hosts} \times (H_{overhead} + N_{open} \times S_{per\_port})$$

| Format | Per-Host Overhead | Per-Port | 254 hosts, 10 open ports |
|:---|:---:|:---:|:---:|
| Normal (-oN) | ~200 B | ~50 B | 178 KB |
| XML (-oX) | ~500 B | ~200 B | 635 KB |
| Grepable (-oG) | ~100 B | ~30 B | 102 KB |

---

## 9. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $N_{hosts} \times N_{ports}$ | Product | Total scan space |
| $N_{probes} / R_{rate}$ | Division | Scan duration |
| $T_{timeout} \times 2^{R}$ | Exponential | Filtered port cost |
| $1 - \prod(1 - P_i)$ | Complement probability | Host detection rate |
| $\text{matching}/\text{total}$ | Ratio | OS fingerprint score |

---

*nmap is applied combinatorics meets network forensics — every scan is a search through the (host, port, service) state space, optimized by parallelism, guided by heuristics, and bounded by the timeout math that makes filtered ports the most expensive thing to scan.*
