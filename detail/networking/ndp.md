# The Mathematics of NDP — Solicited-Node Multicast, Reachability Timers, and Cache Scaling

> *Neighbor Discovery Protocol replaces ARP's broadcast-flood model with targeted multicast and a state machine governed by randomized timers. The math spans probability (collision in solicited-node groups), queueing theory (cache sizing), and Markov-style state transitions (NUD).*

---

## 1. Solicited-Node Multicast — Collision Probability (Combinatorics)

### The Optimization

NDP sends Neighbor Solicitations not to broadcast (like ARP) but to a solicited-node multicast group derived from the last 24 bits of the target address:

$$\text{ff02::1:ff} \| \text{addr}[104:127]$$

Each host only processes NS messages for addresses sharing its last 24 bits. On a subnet with $n$ hosts, the expected number of hosts in a given solicited-node group:

$$E[\text{group size}] = \frac{n}{2^{24}} = \frac{n}{16{,}777{,}216}$$

### Practical Impact

| Hosts on /64 ($n$) | Expected Group Size | Hosts Processing Each NS |
|:---:|:---:|:---:|
| 100 | $5.96 \times 10^{-6}$ | 1 (the target) |
| 10,000 | $5.96 \times 10^{-4}$ | 1 (the target) |
| 1,000,000 | 0.0596 | 1 (the target) |
| $2^{24}$ (~16.7M) | 1.0 | ~2 (target + 1 collision) |

### Comparison with ARP

ARP broadcasts to all $n$ hosts on the subnet. NDP multicast targets $\approx 1$ host:

$$\text{Efficiency ratio} = \frac{n}{E[\text{group size}]} = \frac{n}{n / 2^{24}} = 2^{24}$$

NDP is $\sim 16.7$ million times more efficient than ARP at the link layer for large subnets. Even on a 10,000-host subnet, ARP interrupts all 10,000 hosts; NDP interrupts only the target.

### Birthday Problem — When Do Collisions Matter?

The probability that at least two hosts share a solicited-node group (analogous to birthday problem with $2^{24}$ "days"):

$$P(\text{collision}) \approx 1 - e^{-n^2 / (2 \times 2^{24})}$$

| Hosts ($n$) | $P(\text{any collision})$ |
|:---:|:---:|
| 100 | 0.03% |
| 1,000 | 2.9% |
| 5,000 | 52% |
| 10,000 | 95% |

Collisions are harmless (extra hosts just drop the NS after checking the target field), but the math shows the scaling boundary.

---

## 2. NUD State Machine — Markov Chain Model (Discrete Math)

### State Transitions

NUD defines a state machine for each neighbor cache entry with transition probabilities governed by traffic and timers:

$$S \in \{\text{INCOMPLETE}, \text{REACHABLE}, \text{STALE}, \text{DELAY}, \text{PROBE}, \text{FAILED}\}$$

### Transition Matrix (Simplified)

| From | To | Trigger |
|:---|:---|:---|
| INCOMPLETE | REACHABLE | NA received |
| INCOMPLETE | FAILED | No NA after $N$ retransmits |
| REACHABLE | STALE | ReachableTime expired |
| STALE | DELAY | Traffic sent to neighbor |
| DELAY | PROBE | 5s delay timer expired, no upper-layer confirmation |
| DELAY | REACHABLE | Upper-layer confirmation received |
| PROBE | REACHABLE | NA received |
| PROBE | FAILED | No NA after $N$ unicast probes |

### Time in REACHABLE State

The ReachableTime is randomized between 0.5x and 1.5x the base value to desynchronize NUD across hosts:

$$T_{reachable} \sim \text{Uniform}\left(\frac{T_{base}}{2}, \frac{3 \cdot T_{base}}{2}\right)$$

With default $T_{base} = 30{,}000\text{ ms}$:

$$T_{reachable} \in [15{,}000, 45{,}000] \text{ ms}$$

$$E[T_{reachable}] = T_{base} = 30{,}000 \text{ ms}$$

### Maximum Time to Detect Unreachability

From STALE state through DELAY and PROBE to FAILED:

$$T_{detect} = T_{delay} + N_{probes} \times T_{retrans}$$

With defaults ($T_{delay} = 5\text{s}$, $N_{probes} = 3$, $T_{retrans} = 1\text{s}$):

$$T_{detect} = 5 + 3 \times 1 = 8 \text{ seconds}$$

Total from last confirmed reachability (worst case):

$$T_{total} = T_{reachable,max} + T_{detect} = 45 + 8 = 53 \text{ seconds}$$

---

## 3. Cache Sizing — Queueing and Memory (Applied Math)

### Memory Per Entry

Each neighbor cache entry stores:

| Field | Size |
|:---|:---:|
| IPv6 address | 16 bytes |
| MAC address | 6 bytes |
| State + flags | 4 bytes |
| Timestamps | 16 bytes |
| Device pointer | 8 bytes |
| Linked list / hash | 16 bytes |
| Overhead (alignment) | ~6 bytes |
| **Total** | **~72 bytes** |

### Cache Memory Requirements

$$M = n_{entries} \times 72 \text{ bytes}$$

| gc_thresh3 (max entries) | Memory |
|:---:|:---:|
| 1,024 (default) | 72 KB |
| 8,192 | 576 KB |
| 65,536 | 4.6 MB |
| 262,144 | 18.4 MB |

### Optimal gc_thresh3 for Subnet Size

Rule of thumb: set gc_thresh3 to at least 2x the expected active hosts:

$$\text{gc\_thresh3} \geq 2n_{active}$$

For a data center /64 with 4,000 active hosts:

$$\text{gc\_thresh3} \geq 8{,}000$$

---

## 4. DAD Probability — False Negatives (Probability Theory)

### The Risk

DAD sends $k$ NS probes and waits for NA. If the probe is lost (link error rate $p$), all $k$ probes must be lost for a false negative:

$$P(\text{false negative}) = p^k$$

| Link Error Rate ($p$) | $k = 1$ (default) | $k = 3$ | $k = 5$ |
|:---:|:---:|:---:|:---:|
| 1% | 0.01 | $10^{-6}$ | $10^{-10}$ |
| 5% | 0.05 | $1.25 \times 10^{-4}$ | $3.1 \times 10^{-7}$ |
| 10% | 0.10 | $10^{-3}$ | $10^{-5}$ |

### DAD Completion Time

$$T_{DAD} = k \times T_{retrans}$$

With default $k = 1$ and $T_{retrans} = 1\text{s}$: $T_{DAD} = 1\text{s}$.

With $k = 3$: $T_{DAD} = 3\text{s}$ (slower boot, but far more reliable).

### Optimistic DAD (RFC 4429)

Optimistic DAD allows using the address immediately (with restrictions) while DAD runs in parallel, reducing perceived latency to:

$$T_{optimistic} \approx 0 \text{ (address usable immediately, DAD runs concurrently)}$$

---

## 5. Retransmission — Exponential Backoff (Exponential Functions)

### NS Retransmission in INCOMPLETE State

When resolving a new address, NS probes are sent with retransmission timer $T_{retrans}$ (default 1 second). The total wait before declaring FAILED:

$$T_{total} = \sum_{i=1}^{N_{mcast}} T_{retrans} = N_{mcast} \times T_{retrans}$$

Default $N_{mcast} = 3$ (multicast solicitations): $T_{total} = 3\text{s}$.

### Randomized Retransmission (RFC 4861, Section 7.3.3)

To avoid synchronization, actual retransmission timers should be jittered:

$$T_{actual} = T_{retrans} \times \text{Uniform}(0.5, 1.5)$$

Expected total resolution time with jitter and $N_{mcast}$ probes:

$$E[T_{total}] = N_{mcast} \times T_{retrans}$$

Variance:

$$\text{Var}(T_{total}) = N_{mcast} \times \frac{T_{retrans}^2}{12}$$

---

## 6. RA Interval — Desynchronization (Uniform Distribution)

### Periodic RA Timing

Routers send unsolicited RAs at intervals drawn uniformly between MinRtrAdvInterval and MaxRtrAdvInterval:

$$T_{RA} \sim \text{Uniform}(\text{MinRtrAdvInterval}, \text{MaxRtrAdvInterval})$$

Defaults: MinRtrAdvInterval = 200s, MaxRtrAdvInterval = 600s.

$$E[T_{RA}] = \frac{200 + 600}{2} = 400 \text{ seconds}$$

### Host Discovery Latency

A host booting without sending RS must wait for a periodic RA. Expected wait:

$$E[T_{wait}] = \frac{E[T_{RA}]}{2} = 200 \text{ seconds}$$

With RS (immediate solicitation), the router responds within $\sim 0.5\text{s}$ (MAX_RA_DELAY_TIME), reducing discovery to sub-second.

### Multiple Routers — First RA Arrival

With $r$ routers each sending independent RAs, the expected time to first RA:

$$E[T_{first}] = \frac{E[T_{RA}]}{r}$$

With 2 routers: $E[T_{first}] = 200\text{s}$. With 4 routers: $E[T_{first}] = 100\text{s}$.

---

## 7. Summary of Formulas

| Formula | Domain | Application |
|:---|:---|:---|
| $n / 2^{24}$ | Probability | Solicited-node group size |
| $1 - e^{-n^2/(2 \times 2^{24})}$ | Birthday problem | Group collision probability |
| $T_{base}/2$ to $3T_{base}/2$ | Uniform distribution | Randomized ReachableTime |
| $T_{delay} + N \times T_{retrans}$ | Linear | Unreachability detection time |
| $p^k$ | Probability | DAD false negative rate |
| $n \times 72$ bytes | Linear | Cache memory requirement |

---

*NDP's mathematical design is deliberately conservative: randomized timers prevent thundering herds, solicited-node multicast eliminates broadcast storms, and the NUD state machine guarantees bounded detection time. The protocol trades a few seconds of detection latency for a system that scales to millions of hosts per subnet without the pathologies that plague ARP.*

## Prerequisites

- probability and combinatorics, uniform distributions, basic queueing theory

## Complexity

- **Beginner:** Understand solicited-node multicast addressing and why NDP replaced ARP for better scaling.
- **Intermediate:** Analyze NUD state transitions and timer tuning for different network environments (campus vs data center).
- **Advanced:** Model cache sizing under adversarial conditions (NDP exhaustion attacks) and evaluate SEND's cryptographic overhead vs RA Guard.
