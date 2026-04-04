# The Mathematics of IPv6 — 128-Bit Address Space, SLAAC, and Prefix Delegation

> *IPv6 expands the address space from 32 bits to 128 bits — a jump so vast it defies human intuition. The math involves astronomical comparisons, EUI-64 bit manipulation, and hierarchical prefix delegation that makes address planning a tree-partitioning problem.*

---

## 1. The Address Space — 128-Bit Magnitude

### The Total

$$N_{total} = 2^{128} = 340,282,366,920,938,463,463,374,607,431,768,211,456$$

That's $\sim 3.4 \times 10^{38}$ addresses.

### Magnitude Comparisons

| Comparison | Value | IPv6 Addresses Per... |
|:---|:---:|:---:|
| World population | $8 \times 10^9$ | $4.25 \times 10^{28}$ per person |
| Grains of sand on Earth | $\sim 7.5 \times 10^{18}$ | $4.5 \times 10^{19}$ per grain |
| Stars in observable universe | $\sim 10^{24}$ | $3.4 \times 10^{14}$ per star |
| Atoms in a human body | $\sim 7 \times 10^{27}$ | $4.9 \times 10^{10}$ per atom |
| Nanoseconds since Big Bang | $\sim 4.3 \times 10^{26}$ | $7.9 \times 10^{11}$ per nanosecond |

### Comparison with IPv4

$$\frac{2^{128}}{2^{32}} = 2^{96} = 7.9 \times 10^{28}$$

IPv6 has $\sim 79$ billion billion billion times more addresses than IPv4.

### Why Not Just 64-Bit?

$$2^{64} = 1.8 \times 10^{19}$$

Still enormous, but the 128-bit design dedicates the upper 64 bits to network prefix and lower 64 bits to interface identifier, enabling stateless autoconfiguration.

---

## 2. The /64 Boundary — Subnet Design

### The Standard Allocation

Every IPv6 subnet is a /64:

$$\text{Hosts per /64} = 2^{64} = 18,446,744,073,709,551,616$$

**18.4 quintillion hosts per subnet.** Even at LAN scale, this is inexhaustible — the design eliminates subnet sizing as an engineering concern.

### Typical Allocation Hierarchy

| Entity | Prefix | Subnets Available |
|:---|:---:|:---:|
| RIR to ISP | /32 | $2^{32}$ = 4,294,967,296 /64s |
| ISP to customer (residential) | /48 | $2^{16}$ = 65,536 /64s |
| ISP to customer (small biz) | /56 | $2^{8}$ = 256 /64s |
| Single subnet | /64 | 1 subnet, $2^{64}$ hosts |

### How Many Subnets in a /48?

$$S = 2^{(64 - 48)} = 2^{16} = 65,536 \text{ subnets}$$

For a home network, 65,536 subnets is absurdly generous. For an enterprise campus:

| Campus Size | Subnets Needed | /48 Utilization |
|:---|:---:|:---:|
| Small office (5 VLANs) | 5 | 0.008% |
| Medium campus (200 VLANs) | 200 | 0.3% |
| Large enterprise (2,000 VLANs) | 2,000 | 3.1% |
| Mega campus (10,000 VLANs) | 10,000 | 15.3% |

---

## 3. SLAAC — EUI-64 Bit Manipulation

### The Algorithm

Stateless Address Autoconfiguration (SLAAC) derives a 64-bit Interface Identifier from the 48-bit MAC address:

1. Split MAC into two 24-bit halves: OUI and device ID
2. Insert `FF:FE` between them (48-bit → 64-bit)
3. Flip the 7th bit (Universal/Local bit)

### Step-by-Step Example

MAC address: `00:1A:2B:3C:4D:5E`

```
Step 1: Split           00:1A:2B | 3C:4D:5E
Step 2: Insert FF:FE    00:1A:2B:FF:FE:3C:4D:5E
Step 3: Flip bit 7      02:1A:2B:FF:FE:3C:4D:5E
```

Binary of first byte:
```
00000000  (original)
00000010  (bit 7 flipped = Universal → Local)
= 0x02
```

**Result:** Interface ID = `021A:2BFF:FE3C:4D5E`

With prefix `2001:db8:1:1::/64`, the full address becomes:

$$\text{2001:db8:1:1:021A:2BFF:FE3C:4D5E}$$

### Privacy Addresses (RFC 8981)

EUI-64 exposes the MAC address (and thus the device) in every packet. Privacy extensions generate random interface IDs:

$$IID_{random} = \text{PRNG}(64 \text{ bits}) \quad \text{with bit 6 = 0 (local scope)}$$

Rotated every $T_{preferred}$ (typically 24 hours). This defeats tracking across networks.

---

## 4. Neighbor Discovery — DAD Probability

### The Problem

Duplicate Address Detection (DAD) checks if a generated address is already in use. With random IIDs on a /64, what's the collision probability?

### Birthday Problem Application

$$P_{collision} = 1 - \prod_{i=0}^{n-1}\left(1 - \frac{i}{2^{64}}\right) \approx \frac{n^2}{2 \times 2^{64}}$$

| Hosts on Subnet ($n$) | $P_{collision}$ |
|:---:|:---:|
| 1,000 | $2.7 \times 10^{-14}$ |
| 1,000,000 | $2.7 \times 10^{-8}$ |
| $10^9$ (1 billion) | $2.7 \times 10^{-2}$ (2.7%) |
| $2^{32}$ (4.3 billion) | $\sim 50\%$ |

At realistic LAN sizes (< 10,000 hosts), collision probability is essentially zero — $< 10^{-12}$.

---

## 5. Prefix Delegation Math

### The Problem

ISPs must delegate prefixes to customers from their allocation. How many customers can a /32 serve?

### The Formula

$$C = 2^{(D - A)}$$

Where $D$ = delegated prefix length, $A$ = ISP allocation prefix length.

### Capacity Planning

| ISP Allocation | Customer Prefix | Customers Served |
|:---:|:---:|:---:|
| /32 | /48 each | $2^{16}$ = 65,536 |
| /32 | /56 each | $2^{24}$ = 16,777,216 |
| /32 | /64 each | $2^{32}$ = 4,294,967,296 |
| /24 | /48 each | $2^{24}$ = 16,777,216 |

### Hierarchical Delegation Tree

A /32 ISP allocating /48s to regional POPs, then /56s to customers:

```
/32 ISP
 ├── /40 Region A (256 × /48 enterprise, or 65,536 × /56 residential)
 ├── /40 Region B
 ├── ...
 └── /40 Region H (up to 256 regions)
```

Each level consumes bits: $40 - 32 = 8$ bits = 256 regions.

---

## 6. Header Simplification — Efficiency Math

### IPv4 vs IPv6 Header Comparison

| Feature | IPv4 | IPv6 |
|:---|:---:|:---:|
| Header size (min) | 20 bytes | 40 bytes |
| Header size (with options) | 20-60 bytes | 40 (fixed) + extension headers |
| Fields | 14 | 8 |
| Checksum | Yes | No (removed!) |
| Fragmentation fields | In base header | Extension header only |

### Processing Cost

IPv4 routers must: recompute header checksum on every hop (TTL decrement changes it).

$$\text{IPv4 per-hop ops} = \text{lookup} + \text{TTL decrement} + \text{checksum recalc}$$

$$\text{IPv6 per-hop ops} = \text{lookup} + \text{hop limit decrement}$$

**Checksum removal saves:** At 1 billion packets/second on a core router, eliminating the 16-bit ones' complement sum saves ~1 billion arithmetic operations per second.

---

## 7. Address Representation — Compression Rules

### The Math of Abbreviation

IPv6 addresses are 128 bits = 32 hex digits = 8 groups of 4 hex digits.

**Compression rules:**
1. Leading zeros in each group can be omitted
2. One sequence of consecutive all-zero groups can be replaced with `::`

**Example:** `2001:0db8:0000:0000:0000:0000:0000:0001`

Step 1 (strip leading zeros): `2001:db8:0:0:0:0:0:1`

Step 2 (collapse zeros): `2001:db8::1`

### Possible Representations

An address with $k$ consecutive zero groups has:

$$R = k + 1 \text{ valid representations (:: can replace 1 to } k \text{ groups, or none)}$$

But only one placement of `::` is allowed, and the longest run should be chosen (RFC 5952).

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $2^{128} \approx 3.4 \times 10^{38}$ | Exponent | Total address space |
| $2^{64}$ per /64 | Exponent | Hosts per subnet |
| $2^{(D-A)}$ | Exponent | Prefix delegation capacity |
| $n^2 / (2 \times 2^{64})$ | Birthday problem | DAD collision probability |
| EUI-64: insert FFFE + flip bit 7 | Bit manipulation | SLAAC address generation |
| $2^{(64-p)}$ | Exponent | Subnets per allocation |

## Prerequisites

- binary arithmetic, hexadecimal notation, powers of two, bitwise operations

---

*IPv6's 128-bit address space is so large that if you allocated a million addresses per nanosecond, it would take $10^{19}$ years to exhaust — roughly a billion times the age of the universe. The math isn't just big; it's designed to make address scarcity permanently impossible.*
