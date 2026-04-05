# Network ACLs — Access Control List Architecture and Processing

> *Access Control Lists are ordered rule sets evaluated as linear or TCAM-based classifiers. Their behavior is governed by first-match semantics, wildcard mask bit arithmetic, and hardware forwarding pipeline constraints. Understanding the processing model — from software sequential scan to TCAM ternary matching — determines both security posture and forwarding performance.*

---

## 1. ACL Classification Theory — Rule-Based Packet Classifiers

### The Packet Classification Problem

Given a packet with header fields $(f_1, f_2, \ldots, f_d)$ and an ordered rule set $R = \{r_1, r_2, \ldots, r_n\}$, find the first rule $r_i$ where all field predicates match:

$$\text{match}(r_i, P) = \bigwedge_{j=1}^{d} \text{field}_j(r_i) \ni f_j(P)$$

For a standard ACL, $d = 1$ (source IP only). For an extended ACL, $d$ can be up to 5 (source IP, destination IP, protocol, source port, destination port) or more with flags and options.

### First-Match Semantics

ACLs use a priority-based first-match model. Given rules ordered by sequence number $s_1 < s_2 < \ldots < s_n$:

$$\text{action}(P) = \text{action}(r_k) \quad \text{where } k = \min\{i : \text{match}(r_i, P) = \text{true}\}$$

If no rule matches, the implicit deny applies:

$$k = n + 1 \implies \text{action}(P) = \text{deny}$$

This means rule order directly affects both security and performance. A permit rule placed after a more general deny rule becomes unreachable — a dead rule.

### Dead Rule Detection

A rule $r_j$ is dead (unreachable) if there exists $r_i$ with $i < j$ such that:

$$\forall P : \text{match}(r_j, P) \implies \text{match}(r_i, P)$$

In practice, this means $r_i$'s predicates are a superset of (or equal to) $r_j$'s predicates. For example:

```
10 deny ip any any             ← matches all packets
20 permit tcp host 10.0.0.1 any eq 80  ← dead rule (never reached)
```

### Shadowed Rules

A rule $r_j$ is shadowed if for all packets it matches, a higher-priority rule with a different action already matched:

$$\forall P : \text{match}(r_j, P) \implies \exists r_i, i < j : \text{match}(r_i, P) \land \text{action}(r_i) \neq \text{action}(r_j)$$

Shadowed rules indicate policy conflicts and potential security gaps.

---

## 2. Wildcard Mask Arithmetic — Bitwise Match Semantics

### Definition

A wildcard mask $W$ applied to address $A$ defines a match set. For a candidate address $C$:

$$\text{match}(C, A, W) = \left( (C \oplus A) \,\mathbin{\&}\, \overline{W} \right) = 0$$

Where $\oplus$ is XOR, $\overline{W}$ is bitwise NOT of the wildcard mask, and $\&$ is bitwise AND.

Equivalently: bits where $W = 0$ must match exactly between $C$ and $A$; bits where $W = 1$ are ignored.

### Contiguous vs Non-Contiguous Masks

A contiguous wildcard mask has all 0-bits followed by all 1-bits (like an inverted subnet mask):

$$W_{\text{contiguous}} = \underbrace{00\ldots0}_{k}\underbrace{11\ldots1}_{32-k}$$

This matches a standard CIDR prefix of length $k$. The number of matched addresses:

$$|\text{match set}| = 2^{32-k}$$

Non-contiguous masks allow arbitrary bit patterns:

$$W = 0.0.0.254 \quad (00000000.00000000.00000000.11111110)$$

This matches addresses where all bits match except bit 1 of the last octet, yielding:

$$|\text{match set}| = 2^{\text{popcount}(W)}$$

Where $\text{popcount}(W)$ counts the number of 1-bits in $W$.

### Subnet Mask to Wildcard Conversion

For a subnet mask $S$:

$$W = \overline{S} = (2^{32} - 1) - S$$

Examples:

| Prefix Length | Subnet Mask | Wildcard Mask | Match Set Size |
|:---:|:---|:---|:---:|
| /32 | 255.255.255.255 | 0.0.0.0 | 1 |
| /24 | 255.255.255.0 | 0.0.0.255 | 256 |
| /16 | 255.255.0.0 | 0.0.255.255 | 65,536 |
| /8 | 255.0.0.0 | 0.255.255.255 | 16,777,216 |
| /0 | 0.0.0.0 | 255.255.255.255 | 4,294,967,296 |

### Wildcard Range Encoding

A wildcard pair $(A, W)$ can encode discontinuous ranges that CIDR cannot:

```
# Match 10.0.0.0, 10.0.0.2, 10.0.0.4, 10.0.0.6 (even addresses)
10.0.0.0  0.0.0.6    → W bits: 00000110 → matches where bit 0 = 0

# Match 10.0.0.0 and 10.0.1.0 only (two /24 networks differing in bit 8)
10.0.0.0  0.0.1.0    → W bits: 00000001.00000000 → 2 addresses matched
```

---

## 3. Software ACL Processing — Sequential Scan Complexity

### Linear Scan Model

In software (IOS process switching, Linux iptables), each packet is compared against rules sequentially:

$$T_{\text{classify}} = O(n) \quad \text{where } n = \text{number of ACE entries}$$

Average-case comparison count assuming uniform traffic distribution across rules:

$$E[\text{comparisons}] = \frac{\sum_{i=1}^{n} i \cdot p_i}{\sum_{i=1}^{n} p_i}$$

Where $p_i$ is the probability that rule $i$ is the first match.

### Hot-Rule Optimization

If rule $i$ matches fraction $f_i$ of traffic, placing rules in decreasing $f_i$ order minimizes average comparisons:

$$E_{\text{optimal}} = \sum_{i=1}^{n} i \cdot f_{\sigma(i)}$$

Where $\sigma$ is the permutation sorting $f_i$ in decreasing order. However, ACL semantics require order preservation — reordering may change policy behavior unless rules are independent (non-overlapping).

### Turbo ACLs (Cisco IOS)

Cisco's Turbo ACL feature compiles ACLs with 3+ entries into a trie-based lookup structure:

$$T_{\text{turbo}} = O(1) \quad \text{(constant-time lookup)}$$

The trie is indexed by packet header fields. Memory cost:

$$M_{\text{turbo}} \approx n \times f \times w$$

Where $n$ is the number of entries, $f$ is the number of fields, and $w$ is the word width. Turbo ACLs trade memory for speed — suitable for large ACLs on routers with sufficient DRAM.

---

## 4. Hardware ACL Processing — TCAM Architecture

### TCAM (Ternary Content-Addressable Memory)

TCAM stores entries with three states per bit: 0, 1, or X (don't care). Every entry is compared simultaneously in a single clock cycle:

$$T_{\text{TCAM}} = O(1) \quad \text{regardless of table size}$$

### TCAM Entry Structure

Each ACL entry maps to one or more TCAM entries:

| Field | Width (bits) | Source |
|:---|:---:|:---|
| Source IP | 32 | ACE source + wildcard |
| Destination IP | 32 | ACE destination + wildcard |
| Protocol | 8 | ACE protocol |
| Source Port | 16 | ACE source port |
| Destination Port | 16 | ACE destination port |
| TCP Flags | 8 | ACE established/syn/etc. |
| Interface ID | varies | Applied interface |
| **Total** | **~112+** | |

### TCAM Width and Depth

TCAM capacity is measured in entries (depth) and bits per entry (width):

$$\text{Capacity} = \text{depth} \times \text{width}$$

Typical platform TCAM sizes:

| Platform | TCAM Entries | Width | Notes |
|:---|:---:|:---:|:---|
| Catalyst 3750 | 3,000 | 144-bit | Shared ACL/QoS/routing |
| Catalyst 9300 | 12,000 | 160-bit | Dedicated ACL region |
| Nexus 9300 | 64,000 | 160-bit | Algorithmic TCAM |
| Nexus 9500 | 128,000+ | 160-bit | Distributed per-linecard |

### Port Range Expansion

A port range cannot be encoded as a single ternary pattern. The range must be expanded into multiple TCAM entries:

For a range $[a, b]$, the number of TCAM entries needed:

$$N_{\text{entries}} \leq 2 \times \lceil \log_2(\text{port\_space}) \rceil$$

For 16-bit ports, worst case is $2 \times 16 = 32$ entries per range.

Example: port range 1024-2048:

```
# 1024-2047 = 0000010000000000 to 0000011111111111
# Encodes as: 00000100XXXXXXXX (1 TCAM entry covering 1024-1279)
#             00000101XXXXXXXX (1 TCAM entry covering 1280-1535)
#             0000011XXXXXXXXX (1 TCAM entry covering 1536-2047)
#             0000100000000000 (1 TCAM entry for 2048 exactly)
# Total: 4 TCAM entries
```

### Object-Group Expansion

Object-groups are expanded at install time. An ACL referencing object-groups with $s$ source entries, $d$ destination entries, and $p$ port entries produces:

$$N_{\text{TCAM}} = s \times d \times p$$

Example: 10 sources, 5 destinations, 3 port groups = 150 TCAM entries from a single logical ACE.

---

## 5. ACL Direction and Interface Binding

### Ingress vs Egress Processing

On most platforms, ingress ACLs are processed before the routing decision; egress ACLs are processed after:

```
Ingress ACL → Routing Lookup → Egress ACL → Forwarding
```

### Performance Implications

| Direction | Processing Point | Impact |
|:---|:---|:---|
| Ingress (in) | Before routing table lookup | Drops traffic early, saves CPU/bandwidth |
| Egress (out) | After routing, before transmit | Packet already consumed routing resources |

Ingress filtering is preferred for deny rules — it discards unwanted traffic before consuming forwarding resources.

### Interface Count and TCAM Duplication

When the same ACL is applied to multiple interfaces, TCAM behavior varies:

- **Per-interface TCAM**: each interface gets its own copy — $N_{\text{total}} = N_{\text{entries}} \times I$
- **Shared TCAM with interface qualifier**: single copy with interface ID field — $N_{\text{total}} = N_{\text{entries}}$

Most modern platforms (Catalyst 9000, Nexus 9000) use shared TCAM with interface qualifiers.

---

## 6. Reflexive ACL State Machine

### Session Tracking Model

Reflexive ACLs create dynamic entries that track sessions. When an outbound packet matches a `reflect` clause, a temporary inbound entry is created:

$$\text{Reflected entry}: \text{swap}(src, dst) \land \text{swap}(sport, dport) \land \text{same protocol}$$

### State Lifecycle

```
1. Outbound packet matches reflect clause
2. Dynamic ACE created in inbound ACL (reversed 5-tuple)
3. Timer starts (default 300 seconds for TCP, configurable)
4. Return traffic matches dynamic ACE → permitted
5. TCP FIN/RST detected → entry removed after brief delay
6. Timeout expires → entry removed
```

### Limitations

- No true stateful inspection (doesn't track TCP sequence numbers)
- No application-layer inspection
- Cannot handle protocols with dynamically negotiated ports (FTP active mode, SIP)
- One reflected session = one TCAM entry (if hardware-accelerated)

---

## 7. Time-Based ACL Implementation

### Clock Dependency

Time-based ACLs depend on the device's system clock. NTP synchronization is critical:

$$\text{ACE active} \iff T_{\text{start}} \leq T_{\text{current}} \leq T_{\text{end}}$$

### Periodic vs Absolute Time Ranges

Periodic ranges recur on a schedule:

$$\text{periodic}: \text{day-of-week set} \times [T_{\text{start}}, T_{\text{end}}]$$

Absolute ranges define a single window:

$$\text{absolute}: [T_{\text{absolute\_start}}, T_{\text{absolute\_end}}]$$

### TCAM Impact

When a time range activates or deactivates, the platform must reprogram TCAM entries. This introduces a brief processing delay during the transition. On platforms with large ACLs, the reprogramming time:

$$T_{\text{reprogram}} \approx N_{\text{affected}} \times T_{\text{per\_entry}}$$

Where $T_{\text{per\_entry}}$ is typically 1-10 microseconds on modern ASICs.

---

## 8. IPv6 ACL Differences

### Implicit Permits

Unlike IPv4 ACLs, IPv6 ACLs include implicit permits at the end:

```
implicit permit icmpv6 any any nd-na
implicit permit icmpv6 any any nd-ns
implicit deny ipv6 any any
```

This ensures Neighbor Discovery Protocol (NDP) continues functioning even with restrictive ACLs.

### Address Representation

IPv6 ACLs use prefix notation instead of wildcard masks:

```
# IPv4: 10.0.0.0 0.0.0.255   (wildcard mask)
# IPv6: 2001:db8:1::/48       (prefix length)
```

### Header Complexity

IPv6 extension headers complicate ACL matching. The ACL engine must parse a variable-length extension header chain to reach upper-layer headers:

```
IPv6 Header → [Hop-by-Hop] → [Routing] → [Fragment] → [ESP/AH] → TCP/UDP
```

If a fragment header is present, the first fragment contains upper-layer headers but subsequent fragments do not — making port-based filtering impossible on non-initial fragments.

---

## 9. ACL Scalability Analysis

### Rule Count vs Performance

| Environment | Rules | Software ACL | TCAM ACL | Notes |
|:---|:---:|:---|:---|:---|
| Small office | 10-50 | < 1 ms | 1 clock | Negligible |
| Enterprise | 500-2,000 | 5-20 ms | 1 clock | TCAM preferred |
| Service provider | 5,000-50,000 | 50-500 ms | 1 clock | TCAM required |
| DDoS mitigation | 100,000+ | Infeasible | 1 clock | Requires aTCAM/algorithmic |

### TCAM Utilization Monitoring

Monitoring TCAM utilization is critical. When TCAM is exhausted, new ACL entries fail to install and traffic may be silently permitted or denied based on platform behavior:

$$\text{Utilization} = \frac{N_{\text{used}}}{N_{\text{total}}} \times 100\%$$

Recommended threshold: alert at 80% utilization, critical at 90%.

### ACL Merge Optimization

Two rules can be merged if they differ in exactly one field and together cover a contiguous range:

$$r_i = (A, W_i, \text{action}) \quad r_j = (A, W_j, \text{action}) \quad \rightarrow \quad r_k = (A, W_k, \text{action})$$

Where $W_k$ is the minimal wildcard covering both $W_i$ and $W_j$ without matching additional addresses.

---

## 10. Security Considerations

### Implicit Deny and Fail-Closed Design

The implicit deny makes ACLs fail-closed by default. An improperly configured ACL (missing permits) blocks all traffic rather than allowing all traffic. This is a deliberate security design choice.

### Anti-Spoofing with ACLs (BCP 38 / RFC 2827)

Ingress filtering prevents IP address spoofing:

```
# On customer-facing interface:
# Only permit packets with source IP in customer's assigned range
permit ip 203.0.113.0 0.0.0.255 any     ← customer's prefix
deny   ip any any log                    ← spoofed traffic
```

### ACL Bypass Vectors

Common ways ACLs can be bypassed or rendered ineffective:

1. **Fragmentation**: non-initial fragments lack L4 headers — extended ACL port matching fails
2. **IP options**: packets with IP options may bypass TCAM (punted to CPU)
3. **Tunneled traffic**: GRE/VXLAN/IPsec encapsulated packets hide inner headers
4. **IPv6 extension headers**: complex header chains can evade parsing
5. **TCAM exhaustion**: failed ACL installation may default to permit on some platforms

---

## Prerequisites

- binary arithmetic, bitwise operations (AND, OR, XOR, NOT), CIDR notation, IP addressing, TCP/UDP port numbers, OSI model layers 3-4, routing fundamentals

---

*An ACL is a classifier: an ordered set of predicates applied to packet header fields. Whether evaluated in software as a linear scan or in hardware as a parallel TCAM lookup, the fundamental abstraction is the same — first match wins, implicit deny closes. The wildcard mask is the ACL's most elegant primitive: a single 32-bit word that defines which address bits matter and which do not.*
