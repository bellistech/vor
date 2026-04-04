# The Mathematics of DHCPv6 — Lease State Machines, Prefix Delegation Combinatorics, and Timer Analysis

> *DHCPv6 manages distributed state across clients, servers, and relays using deterministic timers, exponential backoff retransmission, and hierarchical prefix delegation. The math covers finite state machines, combinatorial prefix partitioning, and probabilistic analysis of rapid commit conflicts.*

---

## 1. Retransmission — Exponential Backoff (Exponential Functions)

### The Formula (RFC 8415, Section 15)

DHCPv6 retransmits messages using a randomized exponential backoff:

$$RT = 2 \times RT_{prev} + \text{RAND} \times RT_{prev}$$

Where RAND is uniform on $[-0.1, +0.1]$, giving:

$$RT \in [1.8 \times RT_{prev}, 2.2 \times RT_{prev}]$$

### Initial Retransmission Time

$$RT_1 = IRT + \text{RAND} \times IRT$$

Where IRT (Initial Retransmission Time) varies by message type:

| Message | IRT | MRT (Max RT) | MRC (Max Retransmit Count) |
|:---|:---:|:---:|:---:|
| Solicit | 1s | 3600s | 0 (unlimited) |
| Request | 1s | 30s | 10 |
| Renew | 10s | 600s | 0 (unlimited) |
| Rebind | 10s | 600s | 0 (unlimited) |
| Information-Request | 1s | 3600s | 0 (unlimited) |
| Release | 1s | N/A | 4 |
| Decline | 1s | N/A | 4 |

### Time to Reach MRT

Starting from $RT_1 \approx IRT$, the number of retransmissions to reach MRT:

$$n = \left\lceil \log_2 \frac{MRT}{IRT} \right\rceil$$

For Solicit ($IRT = 1\text{s}$, $MRT = 3600\text{s}$):

$$n = \lceil \log_2(3600) \rceil = \lceil 11.81 \rceil = 12 \text{ retransmissions}$$

Total elapsed time before reaching MRT:

$$T_{total} = \sum_{i=0}^{n-1} 2^i \times IRT = IRT \times (2^n - 1)$$

$$T_{total} = 1 \times (2^{12} - 1) = 4{,}095 \text{ seconds} \approx 68 \text{ minutes}$$

After reaching MRT, all subsequent retransmissions use $RT = MRT \pm 10\%$.

---

## 2. Prefix Delegation — Hierarchical Partitioning (Combinatorics)

### The Delegation Tree

An ISP with allocation $/A$ delegating prefixes of length $/D$ can serve:

$$C = 2^{(D - A)} \text{ customers}$$

### Capacity Planning Table

| ISP Allocation ($/A$) | Delegated Prefix ($/D$) | Customers ($C$) | /64 Subnets per Customer |
|:---:|:---:|:---:|:---:|
| /32 | /48 | $2^{16}$ = 65,536 | $2^{16}$ = 65,536 |
| /32 | /56 | $2^{24}$ = 16,777,216 | $2^{8}$ = 256 |
| /32 | /60 | $2^{28}$ = 268,435,456 | $2^{4}$ = 16 |
| /24 | /48 | $2^{24}$ = 16,777,216 | $2^{16}$ = 65,536 |
| /29 | /48 | $2^{19}$ = 524,288 | $2^{16}$ = 65,536 |

### Customer Subnet Capacity

Each customer with delegated prefix $/D$ can create:

$$S_{customer} = 2^{(64 - D)} \text{ subnets}$$

With a /56 delegation: $S = 2^{8} = 256$ subnets (enough for any home or small business).

### Multi-Tier Delegation

ISP → Regional POP → Customer:

```
/32 ISP allocation
 ├── /40 Region (256 regions from /32)
 │    ├── /48 Enterprise customer (256 per region)
 │    └── /56 Residential customer (65,536 per region)
 └── /40 Region
      └── ...
```

Bits consumed at each level: $\text{level\_bits} = \text{child\_prefix} - \text{parent\_prefix}$

$$\text{Total capacity} = \prod_{i} 2^{\text{level\_bits}_i}$$

---

## 3. Lease Lifecycle — Timer Mathematics (Linear/Exponential)

### Lease Timers

Each DHCPv6 address has two key timers:

$$T_1 = \text{Renew timer (default: } 0.5 \times \text{preferred\_lifetime)}$$
$$T_2 = \text{Rebind timer (default: } 0.8 \times \text{preferred\_lifetime)}$$

### Timeline

$$0 \xrightarrow{\text{use}} T_1 \xrightarrow{\text{renew (unicast)}} T_2 \xrightarrow{\text{rebind (multicast)}} T_{pref} \xrightarrow{\text{deprecated}} T_{valid} \xrightarrow{\text{expired}}$$

### Renewal Probability

If the server is reachable with probability $p$ per attempt and renewal starts at $T_1$ with retransmissions every $RT$:

$$\text{Attempts before } T_2 = \frac{T_2 - T_1}{E[RT]}$$

With $T_1 = 1800\text{s}$, $T_2 = 2880\text{s}$, $E[RT] \approx 20\text{s}$ (Renew MRT = 600s, starting at IRT = 10s):

$$\text{Attempts} \approx \frac{1080}{20} = 54 \text{ attempts}$$

Probability of failing all renewal attempts:

$$P(\text{all fail}) = (1-p)^{54}$$

| Server Availability ($p$) | $P(\text{renew fails})$ |
|:---:|:---:|
| 95% | $6.3 \times 10^{-71}$ |
| 80% | $1.4 \times 10^{-38}$ |
| 50% | $5.6 \times 10^{-17}$ |
| 10% | $2.4 \times 10^{-3}$ |

Even with 50% packet loss, renewal is virtually guaranteed.

---

## 4. Rapid Commit — Conflict Analysis (Probability)

### The Problem

With Rapid Commit, a Solicit triggers an immediate Reply from the server. If multiple servers respond, the client uses the first Reply and ignores others, but each server has already committed state.

### Conflict Probability with $k$ Servers

Each server receiving the Solicit independently commits an address. The client accepts exactly one:

$$\text{Wasted allocations} = k - 1$$

### Server Synchronization Window

If server $i$ responds after delay $d_i \sim \text{Uniform}(0, \delta_{max})$:

$$P(\text{client sees server } j \text{ first}) = \frac{1}{k}$$

The orphaned allocations are cleaned up after lease expiry. The cost of wasted addresses over time $T$ with Solicit rate $\lambda$:

$$\text{Wasted addresses} = \lambda \times T \times (k - 1)$$

For 1000 clients, 3 servers, $T = 3600\text{s}$: up to 2000 temporarily orphaned addresses per hour (reclaimed on lease expiry).

### Recommendation

Rapid Commit is safe only when $k = 1$ (single server). The RFC explicitly warns against using it with multiple servers.

---

## 5. DUID Uniqueness — Collision Probability (Information Theory)

### DUID-LLT Entropy

DUID-LLT contains: hardware type (2 bytes) + time (4 bytes) + MAC (6 bytes) = 12 bytes.

$$H(DUID\text{-}LLT) \leq 96 \text{ bits}$$

But effective entropy is lower:
- Hardware type: ~3 bits (few types in practice)
- Time: ~31 bits (seconds since 2000-01-01, wraps in 2136)
- MAC: ~46 bits (OUI is semi-predictable)

$$H_{effective} \approx 80 \text{ bits}$$

### Collision Probability

For $n$ clients with DUIDs drawn from effective space $2^{80}$:

$$P(\text{collision}) \approx \frac{n^2}{2 \times 2^{80}} = \frac{n^2}{2.42 \times 10^{24}}$$

| Clients ($n$) | $P(\text{collision})$ |
|:---:|:---:|
| $10^6$ | $4.1 \times 10^{-13}$ |
| $10^9$ | $4.1 \times 10^{-7}$ |
| $10^{12}$ | $0.41$ |

For any realistic deployment (< billion clients), DUID collisions are negligible.

---

## 6. Relay Encapsulation — Hop Count Analysis (Graph Theory)

### Maximum Relay Chain

DHCPv6 allows relay chaining (relay forwards to another relay). The hop-count field limits depth:

$$\text{hop-count} \leq 32$$

Each relay adds encapsulation overhead:

$$\text{Overhead per relay} = 34 \text{ bytes (Relay-Forward header + options)}$$

### Maximum Packet Size

With IPv6 minimum MTU of 1280 bytes:

$$\text{Max payload} = 1280 - 40_{IPv6} - 8_{UDP} = 1232 \text{ bytes}$$

Maximum relay depth before fragmentation:

$$d_{max} = \left\lfloor \frac{1232 - S_{client\_msg}}{34} \right\rfloor$$

For a 200-byte client message:

$$d_{max} = \left\lfloor \frac{1032}{34} \right\rfloor = 30 \text{ relay hops}$$

In practice, 2-3 relay hops is the maximum seen in real deployments.

---

## 7. Summary of Formulas

| Formula | Domain | Application |
|:---|:---|:---|
| $RT = 2 \times RT_{prev} \pm 10\%$ | Exponential backoff | Message retransmission |
| $\lceil \log_2(MRT/IRT) \rceil$ | Logarithm | Retransmissions to max timer |
| $2^{(D-A)}$ | Exponent | Prefix delegation capacity |
| $(1-p)^n$ | Geometric probability | Renewal failure probability |
| $n^2 / (2 \times 2^{80})$ | Birthday problem | DUID collision rate |
| $(1232 - S) / 34$ | Linear | Max relay chain depth |

---

*DHCPv6's retransmission algorithm ensures that a client will eventually reach any reachable server, with exponential backoff preventing network saturation. The prefix delegation math shows that even a single /32 allocation can serve millions of customers with hundreds of subnets each — the IPv6 address space makes scarcity-driven engineering permanently obsolete.*

## Prerequisites

- exponential functions and logarithms, combinatorics, basic probability theory

## Complexity

- **Beginner:** Understand the four-message exchange, stateful vs stateless modes, and how M/O flags control client behavior.
- **Intermediate:** Analyze retransmission timing, prefix delegation capacity planning, and relay encapsulation overhead.
- **Advanced:** Model rapid commit conflict rates in multi-server environments and design hierarchical prefix delegation schemes for ISP-scale deployments.
