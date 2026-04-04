# The Mathematics of DHCP — Lease Timing & Address Pool Exhaustion

> *DHCP is a distributed resource allocation system governed by timers, probability, and finite pools. Every lease is a contract with an expiration clock, every pool is a pigeonhole problem, and every relay hop adds latency to a time-sensitive state machine.*

---

## 1. Lease Timing Algebra (Timer Arithmetic)

### The Problem

A DHCP lease has three critical timers: T1 (renewal), T2 (rebind), and lease expiry. How do these timers interact, and what happens when the server is unreachable?

### The Formula

$$T_1 = \frac{L}{2}, \quad T_2 = L \times 0.875, \quad T_{expire} = L$$

Where $L$ is the lease duration in seconds. These are defaults from RFC 2131; the server can override T1 and T2 via options 58 and 59.

### Worked Examples

| Lease Duration $L$ | T1 (Renew) | T2 (Rebind) | Renew Window | Rebind Window |
|:---:|:---:|:---:|:---:|:---:|
| 1 hour (3600s) | 30 min | 52.5 min | 22.5 min | 7.5 min |
| 8 hours (28800s) | 4 hours | 7 hours | 3 hours | 1 hour |
| 24 hours (86400s) | 12 hours | 21 hours | 9 hours | 3 hours |
| 7 days (604800s) | 3.5 days | 6.125 days | 2.625 days | 0.875 days |

The **renew window** is the time between T1 and T2 during which the client unicasts renewals to the original server:

$$W_{renew} = T_2 - T_1 = L \times 0.875 - \frac{L}{2} = 0.375L$$

The **rebind window** is the time between T2 and expiry during which the client broadcasts:

$$W_{rebind} = L - T_2 = L - 0.875L = 0.125L$$

### Retry Behavior During Renewal

The client retries at exponentially decreasing intervals during each window. In the renew window, retransmissions occur at $W_{renew}/2$, $W_{renew}/4$, ... with a minimum of 60 seconds.

Total retry attempts during renew:

$$n_{renew} = \lfloor \log_2\left(\frac{W_{renew}}{60}\right) \rfloor + 1$$

| Lease $L$ | $W_{renew}$ | Max Retries |
|:---:|:---:|:---:|
| 1 hour | 1350s | 5 |
| 8 hours | 10800s | 8 |
| 24 hours | 32400s | 10 |

---

## 2. Address Pool Exhaustion (Combinatorics)

### The Problem

Given a DHCP pool of $N$ addresses and $C$ clients arriving at rate $\lambda$ with average lease duration $L$, when does the pool exhaust?

### The Formula — Steady State Occupancy

At steady state, the expected number of occupied addresses is:

$$E[occupied] = \lambda \times L$$

Pool exhaustion occurs when:

$$\lambda \times L \geq N$$

Critical arrival rate:

$$\lambda_{max} = \frac{N}{L}$$

### Worked Examples

| Pool Size $N$ | Lease $L$ | Max Arrival Rate $\lambda_{max}$ | Max Clients/Hour |
|:---:|:---:|:---:|:---:|
| 100 | 1 hour | 100/3600 = 0.028/s | 100 |
| 100 | 8 hours | 100/28800 = 0.0035/s | 12.5 |
| 254 (/24) | 30 min | 254/1800 = 0.141/s | 508 |
| 254 (/24) | 24 hours | 254/86400 = 0.0029/s | 10.6 |
| 1022 (/22) | 4 hours | 1022/14400 = 0.071/s | 255.5 |

### Birthday Problem Analog — Collision Probability

When the pool is nearly full with $N$ total addresses and $k$ active leases, the probability that the next random address pick collides with an existing lease (if using random allocation):

$$P(collision) = \frac{k}{N}$$

Expected picks to find a free address when $k$ addresses are taken:

$$E[picks] = \frac{N}{N - k}$$

| Pool Utilization | $k/N$ | Expected Picks |
|:---:|:---:|:---:|
| 50% | 0.5 | 2 |
| 75% | 0.75 | 4 |
| 90% | 0.9 | 10 |
| 95% | 0.95 | 20 |
| 99% | 0.99 | 100 |

Most DHCP servers use sequential or bitmap allocation, not random, but this illustrates why high utilization degrades performance.

---

## 3. DORA Timing Analysis (State Machine Latency)

### The Problem

How long does the complete DORA exchange take, and how does relay agent forwarding affect timing?

### The Formula

Without relay:

$$T_{DORA} = 2 \times RTT_{client \leftrightarrow server}$$

With $h$ relay hops:

$$T_{DORA} = 2 \times \left(RTT_{client \leftrightarrow relay_1} + \sum_{i=1}^{h} RTT_{relay_i \leftrightarrow relay_{i+1}} + RTT_{relay_h \leftrightarrow server}\right) + \sum_{i=1}^{h} \delta_i$$

Where $\delta_i$ is the processing delay at each relay.

### Worked Examples

| Topology | Hops | Segment RTTs | Total DORA Time |
|:---|:---:|:---|:---:|
| Direct (same LAN) | 0 | 0.5 ms | ~1 ms |
| Single relay | 1 | 0.5 ms + 2 ms | ~5 ms + processing |
| Dual relay (campus) | 2 | 0.5 + 1 + 3 ms | ~9 ms + processing |
| WAN relay (branch) | 1 | 0.5 + 50 ms | ~101 ms + processing |

### Timeout and Retry

RFC 2131 specifies initial retransmit at 4 seconds, doubling with random jitter:

$$T_{retry}(n) = 2^{n+1} + rand(-1, 1) \text{ seconds}$$

| Attempt $n$ | Base Timeout | Cumulative Wait |
|:---:|:---:|:---:|
| 0 | 4s | 4s |
| 1 | 8s | 12s |
| 2 | 16s | 28s |
| 3 | 32s | 60s |
| 4 | 64s (capped) | 124s |

---

## 4. Option Encoding (Binary Arithmetic)

### The Problem

DHCP options are TLV (Type-Length-Value) encoded. How much space is available, and how are multi-byte values packed?

### The Formula

Each option occupies:

$$S_{option} = 1 + 1 + V = 2 + V \text{ bytes}$$

Where $V$ is the value length. The DHCP message fits in a single UDP datagram; minimum MTU is 576 bytes for IPv4, giving:

$$S_{options\_max} = MTU - 20_{IP} - 8_{UDP} - 236_{DHCP\_fixed} - 4_{magic\_cookie} = 576 - 268 = 308 \text{ bytes}$$

With Ethernet (MTU 1500):

$$S_{options\_max} = 1500 - 268 = 1232 \text{ bytes}$$

### IP Address Encoding

IP addresses are 4 bytes. Option 6 (DNS servers) with two servers:

$$\text{Option 6} = [06][08][08.08.08.08][08.08.04.04]$$

Total: $2 + 8 = 10$ bytes for two DNS servers.

### Subnet Mask Encoding (Option 1)

$$255.255.255.0 \rightarrow [01][04][FF.FF.FF.00]$$

### Option 82 Sub-option Structure

```
Option 82 (Relay Agent Info):
  [52][len]
    Sub-option 1 (Circuit ID): [01][cid_len][circuit_id_data]
    Sub-option 2 (Remote ID):  [02][rid_len][remote_id_data]
```

Total size: $2 + 2 + |CID| + 2 + |RID|$ bytes.

---

## 5. Relay Hop Count (Bounded Iteration)

### The Problem

The `hops` field in the DHCP header counts relay forwarding. RFC 2131 limits this to prevent infinite loops.

### The Formula

$$hops_{max} = 16 \text{ (default, configurable per relay)}$$

Each relay increments:

$$hops_{out} = hops_{in} + 1$$

If $hops_{out} > hops_{max}$, the relay drops the packet.

### Path Length and giaddr Selection

With multiple relay hops, only the first relay's `giaddr` is preserved. Each subsequent relay appends Option 82 sub-options but does not overwrite `giaddr`:

$$giaddr = IP_{relay_1}$$

The server selects the address pool based on `giaddr`. If the relay's IP is not within any configured subnet, the server has no pool to allocate from and silently drops the request.

---

## 6. Pool Sizing for Subnets (Network Planning)

### The Problem

How many DHCP-assignable addresses exist in a given subnet, and what safety margin should you maintain?

### The Formula

For a CIDR block of prefix length $p$:

$$N_{total} = 2^{32-p}$$

$$N_{usable} = 2^{32-p} - 2 \text{ (subtract network and broadcast)}$$

$$N_{DHCP} = N_{usable} - N_{static} - N_{reserved}$$

### Worked Examples

| Subnet | Prefix | Total | Usable | Static (est.) | DHCP Pool |
|:---|:---:|:---:|:---:|:---:|:---:|
| /24 | 24 | 256 | 254 | 10 | 244 |
| /23 | 23 | 512 | 510 | 20 | 490 |
| /22 | 22 | 1024 | 1022 | 30 | 992 |
| /25 | 25 | 128 | 126 | 5 | 121 |
| /26 | 26 | 64 | 62 | 3 | 59 |
| /28 | 28 | 16 | 14 | 2 | 12 |

### Utilization Threshold

Best practice is to alert when pool utilization exceeds 80%:

$$U = \frac{k}{N_{DHCP}}, \quad \text{alert if } U > 0.80$$

At 80% utilization, the expected time until exhaustion under steady arrival rate $\lambda$:

$$T_{exhaust} = \frac{N_{DHCP} - k}{\lambda} = \frac{0.2 \times N_{DHCP}}{\lambda}$$

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $T_1 = L/2, T_2 = 0.875L$ | Linear fractions | Lease timing |
| $\lambda_{max} = N/L$ | Capacity (Little's law) | Pool sizing |
| $N/(N-k)$ | Harmonic series | Collision avoidance |
| $2 \times RTT + \sum \delta_i$ | Additive latency | DORA timing |
| $2^{n+1} + rand(-1,1)$ | Exponential backoff | Retry timing |
| $MTU - 268$ | Subtraction | Option space |
| $2^{32-p} - 2$ | Power of 2 | Subnet sizing |

## Prerequisites

- subnetting, binary encoding, exponential backoff, queueing theory basics

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Lease lookup (by MAC/client-id) | $O(1)$ hash table | $O(N)$ |
| Pool scan (find free address) | $O(N)$ worst case, $O(1)$ bitmap | $O(N/8)$ bitmap |
| Option parsing (TLV walk) | $O(n)$ where $n$ = options bytes | $O(1)$ |
| Relay forwarding | $O(1)$ per hop | $O(1)$ |
| Lease file write (ISC DHCP) | $O(1)$ append | $O(N)$ file |

---

*DHCP is a timed resource allocator operating under scarcity constraints. Every pool is a finite set, every lease is a ticking clock, and the boundary between "enough addresses" and "exhaustion" is a simple inequality that too many network engineers ignore until the pager goes off.*
