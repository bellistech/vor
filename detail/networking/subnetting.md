# The Mathematics of Subnetting — VLSM, Binary Prefix Algebra, and Address Planning

> *Subnetting is applied binary arithmetic under constraint: partition a fixed-size address space into variable-sized blocks that align to powers of 2, minimize waste, and permit hierarchical aggregation. It is the most practically important math in networking.*

---

## 1. The Fundamental Formulas

### Addresses in a Subnet

$$A = 2^{(32 - n)}$$

### Usable Hosts

$$H = 2^{(32 - n)} - 2$$

The $-2$ accounts for the network address (all host bits 0) and broadcast address (all host bits 1).

### Subnet Mask from Prefix Length

$$M = 2^{32} - 2^{(32-n)}$$

Or equivalently: $n$ ones followed by $(32-n)$ zeros in binary.

### Quick Reference Table

| Prefix | Binary Mask | Decimal Mask | Addresses | Usable Hosts |
|:---:|:---|:---|:---:|:---:|
| /20 | 11111111.11111111.11110000.00000000 | 255.255.240.0 | 4,096 | 4,094 |
| /21 | 11111111.11111111.11111000.00000000 | 255.255.248.0 | 2,048 | 2,046 |
| /22 | 11111111.11111111.11111100.00000000 | 255.255.252.0 | 1,024 | 1,022 |
| /23 | 11111111.11111111.11111110.00000000 | 255.255.254.0 | 512 | 510 |
| /24 | 11111111.11111111.11111111.00000000 | 255.255.255.0 | 256 | 254 |
| /25 | 11111111.11111111.11111111.10000000 | 255.255.255.128 | 128 | 126 |
| /26 | 11111111.11111111.11111111.11000000 | 255.255.255.192 | 64 | 62 |
| /27 | 11111111.11111111.11111111.11100000 | 255.255.255.224 | 32 | 30 |
| /28 | 11111111.11111111.11111111.11110000 | 255.255.255.240 | 16 | 14 |
| /29 | 11111111.11111111.11111111.11111000 | 255.255.255.248 | 8 | 6 |
| /30 | 11111111.11111111.11111111.11111100 | 255.255.255.252 | 4 | 2 |
| /31 | 11111111.11111111.11111111.11111110 | 255.255.255.254 | 2 | 2 (P2P) |
| /32 | 11111111.11111111.11111111.11111111 | 255.255.255.255 | 1 | 1 (host) |

---

## 2. Finding Network, Broadcast, and Range

### The Three Operations

Given an IP address and prefix length:

$$\text{Network} = \text{IP} \;\&\; \text{Mask}$$

$$\text{Broadcast} = \text{Network} \;|\; (\sim\text{Mask})$$

$$\text{First Host} = \text{Network} + 1$$

$$\text{Last Host} = \text{Broadcast} - 1$$

### Worked Example: 172.16.45.130/21

**Step 1:** Convert prefix to mask. /21 = 3rd octet has $21 - 16 = 5$ network bits:

$$2^8 - 2^{(8-5)} = 256 - 8 = 248$$

Mask: 255.255.248.0

**Step 2:** Network address (AND):

$$45 \;\&\; 248 = \text{?}$$

$$45 = 00101101$$
$$248 = 11111000$$
$$\text{AND} = 00101000 = 40$$

Network: 172.16.40.0

**Step 3:** Broadcast (OR with wildcard 0.0.7.255):

$$172.16.40.0 \;|\; 0.0.7.255 = 172.16.47.255$$

**Step 4:** Host range: 172.16.40.1 to 172.16.47.254

**Step 5:** Usable hosts: $2^{11} - 2 = 2,046$

---

## 3. VLSM Allocation Algorithm

### The Problem

Given a parent block and a list of subnet requirements (each with a minimum host count), allocate subnets with minimal address waste.

### The Algorithm

1. **Calculate prefix:** For each requirement of $H$ hosts: $n = 32 - \lceil \log_2(H + 2) \rceil$
2. **Sort** requirements by size (largest first — most constrained)
3. **Allocate** sequentially: each subnet starts at the next properly aligned address

### Alignment Rule

A subnet of size $2^k$ must start at an address divisible by $2^k$:

$$\text{Start address} \mod 2^k = 0$$

### Worked Example

Parent: 10.10.0.0/22 (1,024 addresses). Requirements:

| Subnet | Hosts Needed | H+2 | Next $2^k$ | Prefix | Allocated Block |
|:---|:---:|:---:|:---:|:---:|:---|
| Engineering | 200 | 202 | 256 | /24 | 10.10.0.0/24 |
| Sales | 100 | 102 | 128 | /25 | 10.10.1.0/25 |
| HR | 50 | 52 | 64 | /26 | 10.10.1.128/26 |
| Server VLAN | 25 | 27 | 32 | /27 | 10.10.1.192/27 |
| Management | 10 | 12 | 16 | /28 | 10.10.1.224/28 |
| WAN link 1 | 2 | 4 | 4 | /30 | 10.10.1.240/30 |
| WAN link 2 | 2 | 4 | 4 | /30 | 10.10.1.244/30 |

**Utilization:**

$$U = \frac{200 + 100 + 50 + 25 + 10 + 2 + 2}{1024} = \frac{389}{1024} = 38\% \text{ (hosts)}$$

$$U_{allocated} = \frac{256 + 128 + 64 + 32 + 16 + 4 + 4}{1024} = \frac{504}{1024} = 49.2\% \text{ (addresses)}$$

Remaining: $1024 - 504 = 520$ addresses for future growth.

---

## 4. Supernetting (Aggregation) Conditions

### The Three Conditions

Two or more prefixes can be aggregated into one if:

1. **Contiguous**: Address ranges are adjacent with no gaps
2. **Same size**: All prefixes have the same prefix length
3. **Aligned**: The first address is divisible by the aggregate block size

### The Math

$2^k$ contiguous $/$n prefixes aggregate to a single $/($n - k$)$ if:

$$\text{First network address} \mod 2^{(32 - n + k)} = 0$$

### Worked Example: Which Sets Aggregate?

| Set | Prefixes | Contiguous? | Aligned? | Aggregate |
|:---|:---|:---:|:---:|:---|
| A | 192.168.0.0/24 + .1.0/24 | Yes | Yes (0 mod 512=0) | 192.168.0.0/23 |
| B | 192.168.1.0/24 + .2.0/24 | Yes | No (256 mod 512=256) | Cannot aggregate |
| C | 10.0.0.0/24 thru 10.0.3.0/24 | Yes | Yes (0 mod 1024=0) | 10.0.0.0/22 |
| D | 10.0.1.0/24 thru 10.0.4.0/24 | Yes | No (not power-of-2 aligned) | Cannot fully aggregate |

**Set D detail:** 10.0.1.0 and 10.0.2.0 can pair → 10.0.2.0/23 (wait, 1 and 2 can't pair since 1 mod 2=1). Partial: 10.0.2.0/24 + 10.0.3.0/24 → 10.0.2.0/23. Result: three routes instead of one.

---

## 5. Subnet Utilization and Waste Analysis

### The Waste Formula

For a requirement of $H$ hosts:

$$W = 2^{\lceil \log_2(H+2) \rceil} - (H + 2)$$

$$W\% = \frac{W}{2^{\lceil \log_2(H+2) \rceil}} \times 100$$

### Waste by Requirement Size

| Hosts Needed | Block Allocated | Waste | Waste % |
|:---:|:---:|:---:|:---:|
| 1 | /30 (4) | 1 | 25% |
| 2 | /30 (4) | 0 | 0% |
| 14 | /28 (16) | 0 | 0% |
| 15 | /27 (32) | 15 | 47% |
| 30 | /27 (32) | 0 | 0% |
| 31 | /26 (64) | 31 | 48% |
| 100 | /25 (128) | 26 | 20% |
| 200 | /24 (256) | 54 | 21% |
| 254 | /24 (256) | 0 | 0% |
| 255 | /23 (512) | 255 | 50% |

**Worst case:** Requirements of $2^k + 1$ waste nearly 50% of the allocated block.

### Optimization Strategy

When waste > 25%, consider splitting into two subnets with a routing summary:

$$H = 255 \rightarrow /24 (254 \text{ hosts}) + /30 (2 \text{ hosts, for the extra 1})$$

Though this adds routing complexity, it may be worthwhile in address-constrained environments.

---

## 6. The Powers of 2 — Essential Mental Math

### Quick Conversions

| Power | Value | Common Usage |
|:---:|:---:|:---|
| $2^0$ | 1 | /32 host route |
| $2^1$ | 2 | /31 point-to-point |
| $2^2$ | 4 | /30 point-to-point (classic) |
| $2^3$ | 8 | /29 small subnet |
| $2^4$ | 16 | /28 |
| $2^5$ | 32 | /27 |
| $2^6$ | 64 | /26 |
| $2^7$ | 128 | /25 |
| $2^8$ | 256 | /24 (the "standard" subnet) |
| $2^{10}$ | 1,024 | /22 |
| $2^{12}$ | 4,096 | /20 |
| $2^{16}$ | 65,536 | /16 |
| $2^{24}$ | 16,777,216 | /8 |

### The Doubling/Halving Shortcut

Each prefix bit doubles or halves the subnet:
- /24 → /25 = halve the addresses (256 → 128)
- /24 → /23 = double the addresses (256 → 512)

$$\text{Moving } k \text{ bits: multiply or divide by } 2^k$$

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $2^{(32-n)} - 2$ | Exponent | Usable hosts |
| $\text{IP} \;\&\; \text{Mask}$ | Binary AND | Network address |
| $\text{Net} \;|\; \sim\text{Mask}$ | Binary OR | Broadcast address |
| $32 - \lceil\log_2(H+2)\rceil$ | Logarithm/ceiling | Optimal prefix length |
| $\text{Start} \mod 2^k = 0$ | Modular arithmetic | Alignment check |
| $2^{\lceil\log_2(H+2)\rceil} - (H+2)$ | Waste calculation | Utilization analysis |
| $2^{(n-p)}$ | Exponent | Subnet count |

## Prerequisites

- binary arithmetic, powers of two, logarithms, bitwise AND/OR

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Longest prefix match | O(W) | O(n * W) |
| CIDR aggregation | O(n log n) | O(n) |

---

*Subnetting is the most frequently tested networking skill because it combines binary arithmetic, logarithmic thinking, and constrained optimization into problems that reveal whether you truly understand how IP addressing works at the bit level.*
