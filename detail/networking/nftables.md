# The Mathematics of nftables — Sets, Maps, and Verdict Optimization

> *nftables replaces iptables with a virtual machine approach to packet classification. The key mathematical advantage: native set-based matching replaces O(n) linear rule chains with O(1) hash lookups, while verdict maps enable single-lookup decisions that eliminate entire rule chains.*

---

## 1. Architecture — Kernel VM vs Linear Chains

### The nftables Advantage

iptables: one kernel module per match type (xt_tcp, xt_udp, xt_conntrack, ...).
nftables: a single kernel VM interprets bytecode expressions.

### Rule Evaluation Model

nftables rules are compiled to bytecode instructions:

$$T_{rule} = \sum_{i=1}^{I} T_{instruction_i}$$

Where $I$ = number of bytecode instructions per rule. Typical: 5-15 instructions.

| Operation | Instructions | Approx Cost |
|:---|:---:|:---:|
| Load header field | 1 | ~10 ns |
| Compare (eq/neq) | 1 | ~5 ns |
| Set lookup | 1 | ~20 ns |
| Verdict (accept/drop) | 1 | ~5 ns |
| Conntrack match | 2-3 | ~30 ns |

---

## 2. Sets — O(1) Native Matching

### Types of Sets

| Set Type | Data Structure | Lookup Complexity | Use Case |
|:---|:---|:---:|:---|
| hash | Hash table | $O(1)$ | IP addresses, ports |
| rbtree (interval) | Red-black tree | $O(\log n)$ | IP ranges, port ranges |
| bitmap | Bit array | $O(1)$ | Dense port ranges |
| concat | Multi-field hash | $O(1)$ | IP+port combinations |

### Comparison with iptables

For $n$ blocked IPs:

| Approach | Complexity | 10,000 IPs | 100,000 IPs |
|:---|:---:|:---:|:---:|
| iptables rules | $O(n)$ | 10,000 checks | 100,000 checks |
| iptables + ipset | $O(1)$ | 1 check | 1 check |
| nftables set (hash) | $O(1)$ | 1 check | 1 check |
| nftables set (rbtree) | $O(\log n)$ | 14 checks | 17 checks |

### Hash Set Memory

$$M_{set} = B \times S_B + n \times S_E$$

Where $B$ = buckets, $S_B$ = bucket overhead, $n$ = elements, $S_E$ = per-element size.

| Elements | Memory (hash:ipv4) | Memory (hash:ipv4+port) |
|:---:|:---:|:---:|
| 1,000 | ~80 KB | ~120 KB |
| 10,000 | ~800 KB | ~1.2 MB |
| 100,000 | ~8 MB | ~12 MB |
| 1,000,000 | ~80 MB | ~120 MB |

---

## 3. Verdict Maps — Single-Lookup Decisions

### The Problem

Traditional approach — 5 rules for 5 services:

```
tcp dport 22 accept
tcp dport 80 accept
tcp dport 443 accept
tcp dport 8080 accept
tcp dport 8443 accept
```

5 rules = up to 5 evaluations.

### Verdict Map Solution

One rule with a verdict map:

```
tcp dport vmap { 22: accept, 80: accept, 443: accept, 8080: accept, 8443: accept }
```

1 hash lookup = 1 evaluation, regardless of map size.

### Scaling

| Ports | Rules (traditional) | Verdict Map | Speedup |
|:---:|:---:|:---:|:---:|
| 5 | 5 evaluations | 1 lookup | 5x |
| 50 | 50 evaluations | 1 lookup | 50x |
| 500 | 500 evaluations | 1 lookup | 500x |

### Named Maps — Dynamic Data

Maps can map values to values (not just verdicts):

```
tcp dport map @port_to_mark → set packet mark
```

$$\text{Lookup time} = O(1) \text{ regardless of map size}$$

---

## 4. Chain Types and Priorities — Numeric Ordering

### Priority Values

Chains are evaluated in priority order (lower number = earlier):

| Priority | Name | Typical Use |
|:---:|:---|:---|
| -400 | conntrack (defrag) | Reassembly |
| -300 | raw | Conntrack bypass |
| -200 | mangle | Packet modification |
| -150 | dstnat | DNAT |
| -100 | (custom) | Early filtering |
| 0 | filter | Standard filtering |
| 100 | security | SELinux |
| 200 | srcnat | SNAT |
| 300 | (custom) | Late processing |

### Chain Evaluation Order

For hooks with $C$ chains at priorities $p_1 < p_2 < \ldots < p_C$:

$$\text{Evaluation order: chain}(p_1) \rightarrow \text{chain}(p_2) \rightarrow \ldots \rightarrow \text{chain}(p_C)$$

A DROP verdict at any chain stops evaluation — no further chains are processed.

---

## 5. Concatenated Sets — Multi-Dimensional Matching

### The Problem

Match on multiple fields simultaneously (e.g., source IP + destination port). Without concatenation:

$$R = |IP_{set}| \times |Port_{set}| \text{ rule combinations}$$

### With Concatenation

$$R = 1 \text{ rule with concatenated set lookup}$$

$$T_{lookup} = O(1) \quad \text{(single hash on composite key)}$$

### Example

Allow 1,000 specific (IP, port) pairs:

| Approach | Rules | Lookups per Packet |
|:---|:---:|:---:|
| Individual rules | 1,000 | up to 1,000 |
| IP set + port set | 2 | 2 |
| Concatenated set | 1 | 1 |

### Concat Key Hash

$$H = \text{hash}(field_1 \| field_2 \| \ldots \| field_k)$$

Supported concatenation width: up to 5 fields (src IP + dst IP + proto + src port + dst port = 5-tuple).

---

## 6. Atomic Ruleset Replacement

### The Problem

iptables applies rules one at a time. Between rule additions, the ruleset is inconsistent:

$$T_{inconsistent} = N_{rules} \times T_{rule\_add} \approx N \times 1 \text{ ms}$$

For 10,000 rules: ~10 seconds of inconsistency.

### nftables Atomic Commit

nftables loads the entire ruleset atomically:

$$T_{inconsistent} = 0 \text{ (single kernel transaction)}$$

### Ruleset Load Time

$$T_{load} = T_{compile} + T_{commit}$$

| Ruleset Size | iptables (sequential) | nftables (atomic) |
|:---:|:---:|:---:|
| 100 rules | 100 ms | 5 ms |
| 1,000 rules | 1 sec | 20 ms |
| 10,000 rules | 10 sec | 100 ms |
| 100,000 rules | 100 sec | 1 sec |

---

## 7. Conntrack Integration — Stateful Performance

### Flowtable Offload

nftables supports flowtable offload for established connections:

$$T_{flowtable} \approx T_{base} \quad \text{(bypasses entire nftables evaluation)}$$

### Throughput with Flowtable

| Path | Operations | Throughput |
|:---|:---|:---:|
| Full nftables eval | All chains + rules | ~1 Mpps |
| Flowtable (kernel) | Hash lookup only | ~5 Mpps |
| Flowtable (hardware) | NIC offload | Line rate |

### When to Use Flowtable

$$\text{Benefit} = (1 - P_{new}) \times (T_{full} - T_{flowtable})$$

Where $P_{new}$ = fraction of packets that are new connections. For typical web traffic ($P_{new} \approx 1\%$):

$$\text{Benefit} = 0.99 \times (T_{full} - T_{base}) \approx 0.99 \times T_{rules}$$

~99% of packets skip the entire ruleset.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $O(1)$ hash lookup | Constant time | Set matching |
| $O(\log n)$ tree lookup | Logarithmic | Interval/range matching |
| $O(n)$ linear (avoided) | Linear | Traditional rule chains |
| $\text{hash}(f_1 \| f_2 \| \ldots)$ | Hash composition | Concatenated sets |
| $T_{inconsistent} = 0$ | Atomic operation | Ruleset commit |
| $(1 - P_{new}) \times \Delta T$ | Probability weighting | Flowtable benefit |

---

*nftables is what happens when you rethink packet filtering as a data structure problem rather than a list-traversal problem. The shift from O(n) rule matching to O(1) set lookups is the same algorithmic insight that separates a linear search from a hash table — and at millions of packets per second, that difference is everything.*
