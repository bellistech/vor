# The Mathematics of Traceroute — TTL Mechanics, Probe Timing & Path Inference

> *Traceroute transforms the simple TTL decrement into a network tomography tool. Each incremented TTL is a binary search probe into the topology, and the timing data encodes link latencies, queue depths, and routing asymmetries that statistical analysis can decode.*

---

## 1. TTL Increment Mathematics (Linear Probing)

### The Problem

Traceroute sends packets with increasing TTL (1, 2, 3, ...). How many probes are needed, and what is the total time?

### The Formula

Total probes for a path of $H$ hops with $Q$ probes per hop:

$$N_{probes} = H \times Q$$

Total time (best case, all replies received):

$$T_{best} = \sum_{h=1}^{H} Q \times RTT_h$$

If probes within a hop are sent in parallel:

$$T_{parallel} = \sum_{h=1}^{H} RTT_h + (Q - 1) \times \delta$$

Where $\delta$ is the inter-probe delay (typically 0 for modern traceroute).

### Worked Examples

| Path Length $H$ | Probes/Hop $Q$ | Total Probes | Avg RTT/Hop | Time (sequential) |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 3 | 30 | 20 ms | 600 ms |
| 15 | 3 | 45 | 30 ms | 1.35 s |
| 20 | 3 | 60 | 40 ms | 2.4 s |
| 30 | 1 | 30 | 50 ms | 1.5 s |
| 30 | 3 | 90 | 50 ms | 4.5 s |

With timeouts for non-responding hops ($T_{timeout}$ = 5 seconds default):

$$T_{worst} = H \times Q \times T_{timeout}$$

For 30 hops, 3 probes, 5s timeout: $T_{worst} = 450$ seconds. This is why `-q 1 -w 2` dramatically speeds up traceroute.

---

## 2. Per-Hop Latency Estimation (Difference Method)

### The Problem

Traceroute reports RTT to each hop, not the link latency between hops. How do we extract link latencies?

### The Formula

The link latency between hop $h$ and hop $h+1$:

$$L_{h \to h+1} = RTT_{h+1} - RTT_h$$

But this is unreliable because:
1. ICMP generation delay varies per router
2. Return paths may differ per hop
3. Queuing delays fluctuate

### Better Estimation (Median-Based)

Using multiple probes, take the minimum RTT per hop to reduce noise:

$$L_{h \to h+1} = \min(RTT_{h+1}) - \min(RTT_h)$$

### Worked Example

```
Hop  RTT1    RTT2    RTT3    Min    Link Latency (min-based)
1    1.2     1.3     1.1     1.1    —
2    5.4     5.8     5.2     5.2    5.2 - 1.1 = 4.1 ms
3    12.1    150.3   11.8    11.8   11.8 - 5.2 = 6.6 ms
4    20.5    19.8    20.1    19.8   19.8 - 11.8 = 8.0 ms
```

Note hop 3, probe 2 shows 150.3 ms — an outlier from ICMP generation delay. The minimum (11.8 ms) gives a much better estimate.

### Negative Link Latencies

$$L_{h \to h+1} < 0 \implies \text{asymmetric routing or ICMP generation artifact}$$

Negative link latencies are common and indicate that the ICMP reply from hop $h$ took a longer return path than the reply from hop $h+1$. This is not an error; it is information about routing asymmetry.

---

## 3. ECMP and Multipath Detection (Flow Hashing)

### The Problem

ECMP routers hash packet headers to select next-hop links. Classic traceroute varies the source port per probe, causing probes to follow different ECMP paths. How many probes are needed to discover all paths?

### The Formula (Coupon Collector Problem)

If there are $k$ equal-cost paths and probes are uniformly distributed, the expected number of probes to discover all paths:

$$E[N] = k \times H_k = k \sum_{i=1}^{k} \frac{1}{i}$$

Where $H_k$ is the $k$-th harmonic number.

### Worked Examples

| ECMP Paths $k$ | $E[N]$ Probes | $E[N]$ / $k$ | P(all found in $2k$ probes) |
|:---:|:---:|:---:|:---:|
| 2 | 3 | 1.5 | 75% |
| 3 | 5.5 | 1.83 | 74% |
| 4 | 8.3 | 2.08 | 73% |
| 8 | 21.7 | 2.72 | 68% |
| 16 | 54.1 | 3.38 | 61% |

### Paris Traceroute Approach

Paris traceroute keeps the flow hash constant (fixed source port, dest port, protocol), so all probes follow the same ECMP path. The MDA (Multipath Discovery Algorithm) systematically varies the flow key to enumerate all paths:

$$\text{MDA confidence} = 1 - \left(\frac{k-1}{k}\right)^n$$

Where $n$ is probes sent and $k$ is the number of paths. To reach 95% confidence of finding all $k$ paths:

$$n \geq k \times (\ln k + \ln \frac{1}{1 - 0.95})$$

| Paths $k$ | Probes for 95% | Probes for 99% |
|:---:|:---:|:---:|
| 2 | 6 | 9 |
| 4 | 15 | 22 |
| 8 | 37 | 51 |
| 16 | 85 | 113 |

---

## 4. AS Path Inference (Topology Mapping)

### The Problem

Given the IP addresses of intermediate routers, can we infer the Autonomous System (AS) path? How accurate is this?

### The Method

1. Map each router IP to an AS number via BGP route origin:
   $$IP_h \xrightarrow{\text{BGP RIB}} AS_h$$

2. Construct the AS path by deduplicating consecutive AS numbers:
   $$\text{AS Path} = \text{unique}(AS_1, AS_2, \ldots, AS_H)$$

### Accuracy Challenges

| Issue | Cause | Frequency |
|:---|:---|:---:|
| Third-party addresses | Router uses peering link IP from neighbor AS | ~15-20% of hops |
| IXP addresses | IXP-assigned IPs don't map to either AS | ~5% |
| Unresponsive hops | Cannot map `* * *` to any AS | ~10-30% |
| Sibling ASes | Same organization, different AS numbers | ~5% |
| Anycast | Same IP announced from multiple locations | Variable |

### AS Path vs BGP Path

Traceroute AS path $P_{tr}$ vs BGP AS path $P_{bgp}$:

$$\text{Match rate} = \frac{|P_{tr} \cap P_{bgp}|}{|P_{bgp}|} \approx 70-85\%$$

The mismatch comes from asymmetric routing, third-party addresses, and the fact that traceroute shows the data-plane path while BGP shows the control-plane path.

---

## 5. Probe Timing and Rate Analysis (Bandwidth)

### The Problem

How much bandwidth does traceroute consume, and can it trigger rate limiting?

### The Formula

Traceroute probe size (UDP mode):

$$S_{probe} = 14_{Eth} + 20_{IP} + 8_{UDP} + P_{payload} = 42 + P$$

Default payload is typically 32-60 bytes:

$$S_{default} \approx 74 \text{ bytes}$$

Bandwidth:

$$B = \frac{H \times Q \times S_{probe}}{T_{total}} \times 8$$

### Worked Examples

| Hops | Probes/Hop | Probe Size | Duration | Bandwidth |
|:---:|:---:|:---:|:---:|:---:|
| 30 | 3 | 74 B | 5s | 10.6 kbps |
| 30 | 1 | 74 B | 2s | 8.9 kbps |
| 30 | 100 | 74 B | 10s | 1.8 Mbps |

Traceroute bandwidth is negligible for normal use. However, some routers rate-limit ICMP TTL Exceeded generation to ~1 per second, which is the real bottleneck.

### ICMP Rate Limit Impact on Accuracy

If a router rate-limits ICMP to $R$ messages per second and receives $P$ probes per second from various traceroutes:

$$P(\text{response}) = \min\left(1, \frac{R}{P}\right)$$

A busy transit router receiving probes from thousands of traceroutes may show significant loss even though data traffic is unaffected.

---

## 6. Geolocation from Latency (Speed of Light Bound)

### The Problem

Can we estimate geographic distance from RTT?

### The Formula

Light travels at $c = 299,792$ km/s in vacuum. In fiber optic cable, the speed is approximately $2/3 \times c$ due to the refractive index:

$$v_{fiber} \approx 200,000 \text{ km/s}$$

Maximum distance (one-way) from half-RTT:

$$D_{max} = \frac{RTT}{2} \times v_{fiber}$$

### Worked Examples

| RTT | One-Way Latency | Max Distance | Realistic Distance |
|:---:|:---:|:---:|:---:|
| 1 ms | 0.5 ms | 100 km | ~50 km (same metro) |
| 5 ms | 2.5 ms | 500 km | ~200 km |
| 20 ms | 10 ms | 2,000 km | ~1,000 km |
| 60 ms | 30 ms | 6,000 km | ~3,000 km (coast to coast US) |
| 80 ms | 40 ms | 8,000 km | ~4,000 km (transatlantic) |
| 150 ms | 75 ms | 15,000 km | ~8,000 km (US to Asia) |

The realistic distance is typically 40-60% of the speed-of-light maximum because:
1. Fiber paths are not straight lines (follow roads, railways, coastlines)
2. Router processing adds 50-200 microseconds per hop
3. Queuing delays add variable latency

### Constraint-Based Geolocation

$$D(A, B) \leq \frac{RTT(A,B)}{2} \times v_{fiber}$$

This gives an upper bound. With measurements from multiple vantage points, the intersection of distance circles constrains the target location.

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $H \times Q$ | Multiplication | Total probes |
| $\min(RTT_{h+1}) - \min(RTT_h)$ | Differencing | Link latency |
| $k \times H_k$ | Harmonic series | ECMP discovery |
| $1 - ((k-1)/k)^n$ | Complement probability | Path coverage |
| $RTT/2 \times v_{fiber}$ | Distance = rate $\times$ time | Geolocation |

## Prerequisites

- harmonic numbers, coupon collector problem, speed of light, AS topology

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Standard traceroute ($H$ hops, $Q$ probes) | $O(H \times Q \times RTT)$ | $O(H)$ |
| Paris traceroute (single path) | $O(H \times Q \times RTT)$ | $O(H)$ |
| MDA multipath discovery ($k$ paths) | $O(H \times k \times \ln k \times RTT)$ | $O(H \times k)$ |
| IP-to-AS mapping | $O(\log N)$ BGP table lookup | $O(N)$ RIB |
| Geolocation constraint solving | $O(V^2)$ for $V$ vantage points | $O(V)$ |

---

*Traceroute is network tomography with imperfect sensors. Each RTT sample is contaminated by ICMP generation delay, return path asymmetry, and queuing noise. The art is in the statistics: use minimums to cut through noise, use Paris mode to control ECMP, and never trust a single probe.*
