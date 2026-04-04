# The Mathematics of ARP — Cache Dynamics & Broadcast Storm Analysis

> *ARP is a deceptively simple protocol whose mathematics reveal the hidden costs of flat networks: broadcast domains scale quadratically, cache eviction follows queueing theory, and ARP storms exhibit the same exponential growth as network meltdowns.*

---

## 1. ARP Cache Timeout and Reachability (Stochastic Timers)

### The Problem

Linux randomizes ARP cache timeouts to prevent synchronized expiration across all hosts. How does this randomization work, and what is the expected cache hit rate?

### The Formula

The actual reachable time is uniformly distributed:

$$T_{reachable} \sim U\left[\frac{T_{base}}{2}, \frac{3 \cdot T_{base}}{2}\right]$$

Where $T_{base}$ = `base_reachable_time_ms` (default 30,000 ms).

Expected value:

$$E[T_{reachable}] = T_{base} = 30 \text{s}$$

Variance:

$$Var[T_{reachable}] = \frac{(3T_{base}/2 - T_{base}/2)^2}{12} = \frac{T_{base}^2}{12}$$

Standard deviation:

$$\sigma = \frac{T_{base}}{\sqrt{12}} \approx 0.289 \cdot T_{base} \approx 8.66 \text{s}$$

### Cache Hit Probability

If a host communicates with a peer at average interval $\tau$, the probability the cache is still REACHABLE:

$$P(hit) = P(\tau < T_{reachable})$$

For $T_{base} = 30$s, the cache is valid between 15s and 45s:

| Communication Interval $\tau$ | $P(hit)$ |
|:---:|:---:|
| 5s (frequent) | 1.0 |
| 15s | 1.0 |
| 20s | 0.833 |
| 30s | 0.50 |
| 40s | 0.167 |
| 45s | 0.0 |
| 60s | 0.0 (but STALE still works) |

Note: STALE entries are still used immediately while revalidation happens in the background (NUD - Neighbor Unreachability Detection), so the practical impact of a cache miss is minimal for active flows.

---

## 2. ARP Broadcast Storm Probability (Exponential Cascades)

### The Problem

An ARP storm occurs when broadcast traffic triggers a cascade of ARP requests. What conditions cause storms, and how fast do they grow?

### The Formula

In a network with $N$ hosts and a broadcast loop (e.g., STP failure), each broadcast frame is replicated. The number of frames after $k$ loop iterations with $L$ loop paths:

$$F(k) = F_0 \times L^k$$

Where $F_0$ is the initial broadcast frame count.

### Worked Examples

| Initial Frames $F_0$ | Loop Paths $L$ | After 5 iterations | After 10 iterations |
|:---:|:---:|:---:|:---:|
| 1 | 2 | 32 | 1,024 |
| 1 | 3 | 243 | 59,049 |
| 10 | 2 | 320 | 10,240 |
| 10 | 3 | 2,430 | 590,490 |

### ARP-Induced Broadcast Load

On a flat network with $N$ hosts, the worst-case ARP broadcast rate (all caches expire simultaneously):

$$R_{ARP} = N \times (N-1) \times \frac{1}{T_{base}}$$

This is the "ARP storm" scenario. With randomized timers, the expected rate is spread over $[T_{base}/2, 3T_{base}/2]$:

$$R_{ARP}^{avg} = \frac{N \times (N-1)}{T_{base}} \text{ requests/sec (amortized)}$$

| Hosts $N$ | Pairs $N(N-1)$ | ARP req/s ($T_{base}=30$s) | Ethernet % (at 1 Gbps) |
|:---:|:---:|:---:|:---:|
| 50 | 2,450 | 81.7 | ~0.003% |
| 200 | 39,800 | 1,327 | ~0.05% |
| 500 | 249,500 | 8,317 | ~0.3% |
| 1000 | 999,000 | 33,300 | ~1.2% |
| 5000 | 24,995,000 | 833,167 | ~30% |

At 5000 hosts on a flat L2 network, ARP alone consumes significant bandwidth and every host's CPU must process every request.

---

## 3. ARP Table Size Calculations (Memory Bounds)

### The Problem

The Linux kernel limits ARP table entries via `gc_thresh1/2/3`. How should these be sized, and what is the memory cost?

### The Formula

Each ARP entry in the Linux neighbor table occupies approximately:

$$S_{entry} \approx 256 \text{ bytes (struct neighbour + hash overhead)}$$

Total memory for $N$ entries:

$$M = N \times S_{entry}$$

### Worked Examples

| gc_thresh3 (max entries) | Memory | Use Case |
|:---:|:---:|:---|
| 1,024 (default) | 256 KB | Small server, <1K peers |
| 4,096 | 1 MB | Busy server, multiple subnets |
| 16,384 | 4 MB | Hypervisor, many VMs |
| 65,536 | 16 MB | Large load balancer |
| 131,072 | 32 MB | Container host, /16 subnet |

### Garbage Collection Behavior

```
gc_thresh1 = 128   — GC runs softly above this (entries older than 5s removed)
gc_thresh2 = 512   — GC runs aggressively above this (entries older than 5s forced)
gc_thresh3 = 1024  — Hard limit, new entries rejected above this
```

When the table is full ($N \geq$ gc_thresh3), new ARP resolutions fail with `ENOBUFS`. The symptom is intermittent connectivity loss to new destinations while existing connections (with cached entries) continue to work.

---

## 4. ARP Spoofing Detection — Statistical Analysis

### The Problem

How can you statistically detect ARP spoofing by monitoring MAC-to-IP binding changes?

### The Formula

Define the "flip rate" for an IP address as the number of MAC address changes per unit time:

$$F_{ip} = \frac{\text{MAC changes for IP}}{\Delta t}$$

Normal flip rate (legitimate DHCP reassignment):

$$F_{normal} \leq \frac{1}{T_{lease}} \approx \frac{1}{86400} \approx 1.16 \times 10^{-5} \text{ /s}$$

ARP spoofing typically produces:

$$F_{spoof} \geq \frac{1}{T_{poison\_interval}} \approx \frac{1}{2} = 0.5 \text{ /s}$$

Detection threshold (anomaly if):

$$F_{ip} > k \times F_{normal}$$

Where $k$ is typically 10-100.

### Gratuitous ARP Rate Analysis

Normal GARP events (per host):
- Boot: 1 per boot cycle
- DHCP renewal: 1 per lease
- Failover: 1 per event (rare)

Expected GARP rate for $N$ hosts:

$$R_{GARP}^{normal} \approx \frac{N}{T_{avg\_uptime}} + \frac{N}{T_{lease}}$$

| Hosts | Avg Uptime | Lease | Normal GARP/hour |
|:---:|:---:|:---:|:---:|
| 100 | 30 days | 8 hours | 12.5 |
| 500 | 30 days | 8 hours | 62.7 |
| 100 | 30 days | 1 hour | 100.1 |

A sudden spike above this baseline signals either a failover event (check VRRP/HSRP) or a spoofing attack.

---

## 5. Proxy ARP Routing Cost (Path Analysis)

### The Problem

Proxy ARP makes a router appear as the direct L2 neighbor for remote hosts. What is the overhead compared to normal routing?

### The Formula

With proxy ARP enabled, the router must:
1. Receive and process ARP requests for all remote IPs: $O(N_{remote})$ per broadcast
2. Maintain ARP entries for all local hosts that queried: $O(N_{local})$ entries
3. Forward each packet with MAC rewrite: same as normal routing

The ARP overhead compared to standard routing:

$$\Delta_{overhead} = N_{local} \times R_{query} \times S_{ARP}$$

Where $R_{query}$ is the query rate per local host and $S_{ARP}$ = 42 bytes (28-byte ARP + 14-byte Ethernet header).

For 200 local hosts querying 50 remote destinations with cache timeout 30s:

$$\Delta = 200 \times \frac{50}{30} \times 42 = 14,000 \text{ bytes/sec} \approx 112 \text{ kbps}$$

Negligible bandwidth but measurable CPU on low-end routers processing broadcast interrupts.

---

## 6. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $U[T/2, 3T/2]$ | Uniform distribution | Cache timeout |
| $F_0 \times L^k$ | Exponential growth | Broadcast storms |
| $N(N-1)/T$ | Quadratic scaling | ARP broadcast rate |
| $N \times 256$ bytes | Linear scaling | Table memory |
| $\text{MAC changes}/\Delta t$ | Rate analysis | Spoofing detection |
| $N \times R \times S$ | Product (throughput) | Proxy ARP overhead |

## Prerequisites

- probability distributions, exponential growth, hash tables, broadcast domains

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| ARP cache lookup | $O(1)$ hash | $O(N)$ |
| ARP request processing | $O(1)$ per host | $O(1)$ |
| Broadcast delivery | $O(N)$ all hosts | $O(1)$ per frame |
| Cache garbage collection | $O(N)$ scan | $O(1)$ |
| Spoofing detection (arpwatch) | $O(1)$ per ARP packet | $O(N)$ binding table |

---

*ARP's simplicity is deceptive. On a flat /16 network with 65,534 potential hosts, the ARP table alone can exhaust default kernel limits, and a single broadcast storm can saturate gigabit links in seconds. The math tells you exactly when to stop stretching Layer 2 and start routing.*
