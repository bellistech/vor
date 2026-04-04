# mtr Deep Dive — Theory & Internals

> *mtr (My Traceroute) combines traceroute and ping into a continuously-updating path analysis tool. The math covers TTL-based hop discovery, statistical analysis of per-hop latency, loss detection methodology, and the difference between forward-path and reverse-path problems.*

---

## 1. TTL-Based Path Discovery

### How It Works

mtr sends packets with incrementing TTL values. Each router decrements TTL and returns ICMP Time Exceeded when TTL reaches 0:

$$\text{Hop } n \text{ responds when: } TTL_{sent} = n$$

### Discovery Time

$$T_{discovery} = H \times T_{probe}$$

Where $H$ = number of hops and $T_{probe}$ = time between probes.

For a 15-hop path at 1 probe/sec: $T_{full\_discovery} = 15$ seconds.

### Packet Loss at Each Hop

mtr sends multiple probes per hop over time. The loss percentage at hop $i$:

$$L_i = \frac{P_{lost,i}}{P_{sent,i}} \times 100\%$$

---

## 2. Latency Statistics — Per-Hop Analysis

### The Metrics

For each hop, mtr computes:

$$\text{Avg} = \frac{1}{N}\sum_{j=1}^{N} RTT_{i,j}$$

$$\text{Best} = \min_j(RTT_{i,j})$$

$$\text{Worst} = \max_j(RTT_{i,j})$$

$$\text{StDev} = \sqrt{\frac{1}{N-1}\sum_{j=1}^{N}(RTT_{i,j} - \text{Avg})^2}$$

### Interpreting Per-Hop Latency

The **per-hop latency** (propagation + processing at that hop):

$$\Delta_{i} = \text{Avg}_i - \text{Avg}_{i-1}$$

| $\Delta_i$ | Interpretation |
|:---:|:---|
| 0-2 ms | Local link (same LAN/DC) |
| 2-10 ms | Metropolitan distance |
| 10-40 ms | Cross-country |
| 40-80 ms | Transatlantic/transpacific |
| 80-150 ms | Satellite or congested path |
| Negative | ICMP deprioritization (not real) |

### Negative Delta — The Rate Limiting Artifact

$$\Delta_i < 0 \quad \text{does NOT mean the hop is faster}$$

Some routers rate-limit ICMP generation, making their response appear slower than the next hop. The rule:

$$\text{If } L_i > 0 \text{ but } L_{i+1} = 0 \text{: loss is at hop } i$$

$$\text{If } L_i > 0 \text{ and } L_{i+1} > 0 \text{ (same %}): \text{ loss is at hop } i$$

$$\text{If } L_i > 0 \text{ but } L_{i+1} < L_i\text{: ICMP rate limiting, not real loss}$$

---

## 3. Loss Pattern Analysis

### Forward vs Return Path

mtr only sees the round trip. Loss could be:

$$L_{measured} = L_{forward} + L_{return} - L_{forward} \times L_{return}$$

For small loss rates: $L_{measured} \approx L_{forward} + L_{return}$.

### Identifying the Lossy Hop

**Rule:** Real loss persists at all subsequent hops. ICMP rate limiting appears only at the specific hop.

| Pattern | Diagnosis |
|:---|:---|
| 5% loss at hop 4, 5% at hops 5-10 | Real loss at hop 4 |
| 5% loss at hop 4, 0% at hops 5-10 | ICMP rate limiting at hop 4 |
| 0% everywhere except 10% at final hop | Destination or last-mile issue |
| Increasing loss from hop 6 onward | Congestion starting at hop 6 |

### Statistical Significance

For $N$ probes, the confidence interval on loss rate:

$$L \pm z \times \sqrt{\frac{L(1-L)}{N}}$$

Where $z = 1.96$ for 95% confidence.

| Probes ($N$) | Measured Loss | 95% CI |
|:---:|:---:|:---:|
| 10 | 10% | 0% - 29% |
| 50 | 10% | 2% - 18% |
| 100 | 10% | 4% - 16% |
| 500 | 10% | 7% - 13% |

**Minimum useful probes: ~100** for statistically meaningful loss measurement.

---

## 4. Jitter Analysis

### Standard Deviation as Jitter Proxy

$$\text{Jitter}_i = \text{StDev}_i$$

### Per-Hop Jitter Contribution

$$J_{link,i} = \sqrt{\text{StDev}_i^2 - \text{StDev}_{i-1}^2}$$

(Assuming independence — variance is additive.)

### Jitter Quality Thresholds

| StDev | Quality | Suitable For |
|:---:|:---|:---|
| < 1 ms | Excellent | VoIP, trading |
| 1-5 ms | Good | Video, gaming |
| 5-20 ms | Acceptable | Web, bulk transfer |
| 20-50 ms | Poor | Only bulk/email |
| > 50 ms | Bad | Significant buffering needed |

---

## 5. MPLS Label Detection

### How mtr Detects MPLS

If routers include MPLS label information in ICMP extensions (RFC 4950):

$$\text{mtr output: } \text{hop 4: [MPLS: Lbl 24001 TC 0 S 1 TTL 1]}$$

### Label Stack Depth

Each additional MPLS label in the TTL-exceeded response indicates another layer of encapsulation:

$$D_{stack} = N_{labels\_shown}$$

---

## 6. AS Path Mapping (`-z` flag)

### IP-to-ASN Resolution

mtr can show the AS number for each hop using Team Cymru DNS:

$$\text{ASN}(IP) = \text{DNS lookup: } d.c.b.a.\text{origin.asn.cymru.com}$$

### Path AS Boundaries

| Hop | IP | ASN | Interpretation |
|:---:|:---|:---:|:---|
| 1 | 192.168.1.1 | Private | Your gateway |
| 2 | 10.0.0.1 | Private | ISP internal |
| 3 | 203.0.113.1 | AS64500 | ISP edge |
| 4 | 198.51.100.1 | AS64501 | Transit provider |
| 5 | 198.51.100.5 | AS64501 | Transit internal |
| 6 | 203.0.113.50 | AS64502 | Destination ISP |
| 7 | 93.184.216.34 | AS64502 | Destination |

Each AS transition is a peering or transit relationship. Latency jumps at AS boundaries often indicate geographic distance.

---

## 7. Probe Mode Comparison

### ICMP vs UDP vs TCP

| Mode | Flag | Response Source | Firewall Friendly |
|:---|:---:|:---|:---:|
| ICMP | default | ICMP Echo Reply | Often blocked |
| UDP | `--udp` | ICMP Port Unreachable | Usually works |
| TCP SYN | `--tcp -P 443` | SYN-ACK (final hop) | Best for web servers |

### Packet Size Impact

$$T_{transmission} = \frac{S_{packet} \times 8}{BW_{link}}$$

| Packet Size | At 1 Mbps | At 100 Mbps |
|:---:|:---:|:---:|
| 64 B (default) | 0.5 ms | 0.005 ms |
| 1,400 B | 11.2 ms | 0.112 ms |

On slow links, large mtr packets add measurable serialization delay.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $\text{Avg}_i - \text{Avg}_{i-1}$ | Difference | Per-hop latency |
| $P_{lost}/P_{sent} \times 100$ | Percentage | Hop loss rate |
| $L \pm z\sqrt{L(1-L)/N}$ | Confidence interval | Loss significance |
| $\sqrt{V_i^2 - V_{i-1}^2}$ | Variance decomposition | Per-link jitter |
| $H \times T_{probe}$ | Product | Path discovery time |

## Prerequisites

- statistics (mean, standard deviation), ICMP protocol, TTL mechanics

---

*mtr is the networking equivalent of a CT scan — it shows you not just that something is wrong, but exactly where in the path the problem lives. The statistics it computes per hop transform raw RTT samples into actionable diagnostics.*
