# The Mathematics of iptables — Chain Traversal, Rule Matching, and State Tracking

> *iptables implements a packet filter as a directed graph of chains, where each packet traverses a sequence of rules matched in O(n) linear time. The math governs rule ordering optimization, connection tracking state table sizing, and the throughput cost of deep rule sets.*

---

## 1. Packet Traversal — Chains as a Directed Graph

### The Structure

iptables has 5 built-in chains forming a directed acyclic graph (DAG):

```
PREROUTING → routing decision → FORWARD → POSTROUTING
                ↓                              ↑
              INPUT → local process → OUTPUT ──┘
```

### Tables and Chain Interactions

Each chain is evaluated across tables in a fixed order:

| Chain | Table Order | Max Rule Sets Evaluated |
|:---|:---|:---:|
| PREROUTING | raw → mangle → nat | 3 |
| INPUT | mangle → filter → nat | 3 |
| FORWARD | mangle → filter | 2 |
| OUTPUT | raw → mangle → nat → filter | 4 |
| POSTROUTING | mangle → nat | 2 |

### Total Rules Evaluated per Packet

For a forwarded packet:

$$R_{total} = R_{PREROUTING} + R_{FORWARD} + R_{POSTROUTING}$$

$$= (R_{raw} + R_{mangle} + R_{nat}) + (R_{mangle} + R_{filter}) + (R_{mangle} + R_{nat})$$

For a locally-destined packet:

$$R_{total} = R_{PREROUTING} + R_{INPUT}$$

---

## 2. Rule Matching Complexity — Linear Search

### The Problem

iptables evaluates rules sequentially. For $n$ rules in a chain, each packet requires:

$$O(n) \text{ comparisons (worst case: no match until last rule or default policy)}$$

### Average Case

If the matching rule is uniformly distributed:

$$E[\text{comparisons}] = \frac{n + 1}{2}$$

But traffic is not uniform — most packets match early rules (established connections):

$$E[\text{comparisons}] = \sum_{i=1}^{n} i \times p_i$$

Where $p_i$ = probability of matching rule $i$.

### Worked Example: Optimized Rule Order

| Rule # | Match | Traffic Share | Comparisons Contributed |
|:---:|:---|:---:|:---:|
| 1 | ESTABLISHED,RELATED | 90% | $1 \times 0.90 = 0.90$ |
| 2 | TCP dport 443 | 5% | $2 \times 0.05 = 0.10$ |
| 3 | TCP dport 80 | 3% | $3 \times 0.03 = 0.09$ |
| 4 | TCP dport 22 | 1% | $4 \times 0.01 = 0.04$ |
| 5 | DROP (default) | 1% | $5 \times 0.01 = 0.05$ |
| | | **Total:** | **1.18 avg comparisons** |

**Unoptimized** (ESTABLISHED last):

$$E = 5 \times 0.90 + \ldots = 4.5 + \ldots \approx 4.7 \text{ avg comparisons}$$

**Optimization factor: 4x fewer comparisons** just by reordering.

### Rule Ordering Principle

$$\text{Optimal order: sort by } \frac{p_i}{\text{cost}_i} \text{ descending (highest hit-rate, lowest cost first)}$$

---

## 3. Connection Tracking (conntrack) State Table

### The Problem

Stateful firewalling requires tracking every connection. How large does the conntrack table need to be?

### Table Size Formula

$$C_{max} = N_{hosts} \times S_{avg}$$

Where $S_{avg}$ = average concurrent sessions per host.

### Default Limits

$$\text{nf\_conntrack\_max} = \frac{RAM_{MB}}{16} \times 1024$$

(Approximately. Kernel auto-tunes based on available memory.)

### Memory per Entry

Each conntrack entry: ~300 bytes (kernel version dependent).

$$\text{Memory} = C_{max} \times 300 \text{ bytes}$$

| Connections | Memory | Default Fits (4 GB RAM) |
|:---:|:---:|:---:|
| 65,536 | 19 MB | Yes |
| 262,144 | 75 MB | Yes (default) |
| 1,000,000 | 286 MB | Yes (tuned) |
| 10,000,000 | 2.86 GB | Tight |

### Hash Table Sizing

The conntrack hash table has $B$ buckets. Optimal:

$$B = \frac{C_{max}}{8}$$

Average chain length = $C_{max} / B = 8$. Lookup time: $O(8) \approx O(1)$.

**If $B$ is too small** (long chains): $O(C/B)$ — degrades to linear search per bucket.

---

## 4. Throughput Impact — Cost per Rule

### Measurement Model

Each rule evaluation costs $T_{rule}$ nanoseconds:

$$T_{packet} = T_{base} + n \times T_{rule}$$

Typical values: $T_{base} \approx 500$ ns, $T_{rule} \approx 50$-$100$ ns.

### Packets per Second

$$PPS = \frac{10^9}{T_{packet}} = \frac{10^9}{T_{base} + n \times T_{rule}}$$

| Rules ($n$) | $T_{packet}$ (ns) | PPS (millions) | Throughput at 1500B |
|:---:|:---:|:---:|:---:|
| 10 | 1,000 | 1.0 | 12 Gbps |
| 100 | 5,500 | 0.18 | 2.2 Gbps |
| 1,000 | 50,500 | 0.020 | 240 Mbps |
| 10,000 | 500,500 | 0.002 | 24 Mbps |

**10,000 rules drops throughput by 500x.** This is why ipset and nftables sets exist.

---

## 5. ipset — O(1) Matching with Hash Sets

### The Improvement

Instead of $n$ rules matching individual IPs:

```
-A INPUT -s 1.2.3.4 -j DROP
-A INPUT -s 5.6.7.8 -j DROP
... (10,000 rules)
```

Use one ipset rule:

```
-A INPUT -m set --match-set blocklist src -j DROP
```

### Complexity Comparison

| Method | Matching Complexity | 10,000 entries |
|:---|:---:|:---:|
| Individual rules | $O(n)$ | 10,000 comparisons |
| ipset (hash:ip) | $O(1)$ | 1 hash lookup |
| ipset (hash:net) | $O(1)$ | 1 hash lookup + prefix match |

### Memory for ipset

Hash set with $n$ entries and load factor $\alpha = 0.75$:

$$\text{Buckets} = \lceil n / \alpha \rceil$$

$$\text{Memory} \approx n \times 64 \text{ bytes (per entry)}$$

| Entries | Rules Method (mem) | ipset Method (mem) | Speedup |
|:---:|:---:|:---:|:---:|
| 100 | ~50 KB | ~7 KB | 100x |
| 10,000 | ~5 MB | ~640 KB | 10,000x |
| 1,000,000 | ~500 MB | ~64 MB | 1,000,000x |

---

## 6. NAT — State Table and Port Allocation

### SNAT/Masquerade Port Mapping

Available ports per NAT IP:

$$P_{available} = 65,535 - 1,024 = 64,511$$

With $K$ public IPs:

$$C_{max\_NAT} = K \times 64,511 \times 2 \quad \text{(TCP + UDP independently)}$$

### NAT Lookup Complexity

NAT uses the conntrack table for reverse mapping:

$$T_{NAT} = T_{conntrack\_lookup} = O(1) \quad \text{(hash table)}$$

---

## 7. Rate Limiting — Token Bucket Math

### The `-m limit` Module

Uses a token bucket model:

$$\text{Tokens}(t) = \min(\text{burst}, \text{tokens}_{prev} + \text{rate} \times \Delta t)$$

Packet is allowed if tokens > 0; one token consumed per packet.

### Parameters

| Option | Meaning | Default |
|:---|:---|:---:|
| `--limit` | Token refill rate | 3/hour |
| `--limit-burst` | Bucket size (max tokens) | 5 |

### Worked Example

`-m limit --limit 10/sec --limit-burst 20`:

- Sustained rate: 10 packets/sec
- Burst capacity: 20 packets
- Time to refill after full burst: $20 / 10 = 2$ sec

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $O(n)$ per chain | Linear search | Rule matching complexity |
| $\sum i \times p_i$ | Weighted average | Expected comparisons |
| $C \times 300$ bytes | Linear | Conntrack memory |
| $10^9 / (T_{base} + n \times T_{rule})$ | Inverse | Packets per second |
| $O(1)$ hash lookup | Constant | ipset matching |
| $K \times 64,511 \times 2$ | Product | NAT capacity |
| $\min(\text{burst}, \text{tokens} + \text{rate} \times \Delta t)$ | Token bucket | Rate limiting |

## Prerequisites

- set theory, chain evaluation, token bucket algorithms, packet matching

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Linear rule match | O(n) | O(n) |
| Connection tracking lookup | O(1) avg | O(connections) |

---

*Every iptables rule you add is a tax on every packet. The difference between a 10-rule firewall and a 10,000-rule firewall is the difference between line-rate forwarding and a bottleneck — which is why the first rule should always be ESTABLISHED,RELATED.*
