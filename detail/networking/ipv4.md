# The Mathematics of IPv4 — Binary Subnetting, CIDR, and Address Exhaustion

> *IPv4 is a 32-bit addressing system governed by binary arithmetic, powers of 2, and Boolean logic. Subnet design is combinatorics under constraint: partitioning a fixed address space to minimize waste while maintaining routability.*

---

## 1. The Address Space — 32-Bit Magnitude

### The Total

$$N_{total} = 2^{32} = 4,294,967,296 \text{ addresses}$$

After reservations (RFC 5735), the usable space is significantly smaller:

| Block | Size | Purpose | Addresses |
|:---|:---|:---|:---:|
| 10.0.0.0/8 | $2^{24}$ | Private (RFC 1918) | 16,777,216 |
| 172.16.0.0/12 | $2^{20}$ | Private (RFC 1918) | 1,048,576 |
| 192.168.0.0/16 | $2^{16}$ | Private (RFC 1918) | 65,536 |
| 127.0.0.0/8 | $2^{24}$ | Loopback | 16,777,216 |
| 224.0.0.0/4 | $2^{28}$ | Multicast | 268,435,456 |
| 240.0.0.0/4 | $2^{28}$ | Reserved/Future | 268,435,456 |

Total reserved: ~592 million. Usable unicast: ~3.7 billion, serving ~8 billion humans.

### Addresses per Person

$$\frac{3.7 \times 10^9}{8 \times 10^9} \approx 0.46 \text{ addresses/person}$$

This is why NAT and IPv6 exist.

---

## 2. Subnet Math — The Core Binary Formulas

### Hosts per Subnet

$$H = 2^{(32 - n)} - 2$$

Where $n$ = prefix length. The "$-2$" accounts for network address and broadcast address.

### Subnets from a Block

Given a parent prefix $/p$ split into subnets of size $/n$:

$$S = 2^{(n - p)}$$

### Worked Examples

| Prefix | Host Bits | Total Addresses | Usable Hosts | Subnet Mask |
|:---:|:---:|:---:|:---:|:---|
| /24 | 8 | 256 | 254 | 255.255.255.0 |
| /25 | 7 | 128 | 126 | 255.255.255.128 |
| /26 | 6 | 64 | 62 | 255.255.255.192 |
| /27 | 5 | 32 | 30 | 255.255.255.224 |
| /28 | 4 | 16 | 14 | 255.255.255.240 |
| /29 | 3 | 8 | 6 | 255.255.255.248 |
| /30 | 2 | 4 | 2 | 255.255.255.252 |
| /31 | 1 | 2 | 2 (RFC 3021) | 255.255.255.254 |
| /32 | 0 | 1 | 1 (host route) | 255.255.255.255 |

### Subnet Mask as Binary AND

The network address is computed by bitwise AND:

$$\text{Network} = \text{IP} \;\&\; \text{Mask}$$

**Example:** 192.168.1.130/26

```
IP:   11000000.10101000.00000001.10000010
Mask: 11111111.11111111.11111111.11000000
AND:  11000000.10101000.00000001.10000000 = 192.168.1.128
```

Broadcast = network OR (NOT mask):

$$\text{Broadcast} = \text{Network} \;|\; (\sim\text{Mask})$$

$$= 192.168.1.128 \;|\; 0.0.0.63 = 192.168.1.191$$

---

## 3. VLSM — Variable Length Subnet Masking

### The Problem

Given a network block (e.g., 10.0.0.0/24), allocate subnets of varying sizes with minimum waste.

### The Algorithm

1. Sort requirements from largest to smallest
2. Allocate each subnet at the next aligned boundary
3. Prefix length: $n = 32 - \lceil \log_2(H + 2) \rceil$

### Worked Example

Allocate from 10.0.0.0/24 for: 100 hosts, 50 hosts, 25 hosts, 2 hosts (point-to-point).

| Requirement | +2 (net+bcast) | Next power of 2 | Prefix | Subnet | Range |
|:---:|:---:|:---:|:---:|:---|:---|
| 100 hosts | 102 | 128 ($2^7$) | /25 | 10.0.0.0/25 | .0 - .127 |
| 50 hosts | 52 | 64 ($2^6$) | /26 | 10.0.0.128/26 | .128 - .191 |
| 25 hosts | 27 | 32 ($2^5$) | /27 | 10.0.0.192/27 | .192 - .223 |
| 2 hosts | 4 | 4 ($2^2$) | /30 | 10.0.0.224/30 | .224 - .227 |

**Waste calculation:**

$$W = 256 - (128 + 64 + 32 + 4) = 256 - 228 = 28 \text{ addresses (10.9\% waste)}$$

Without VLSM (all /25): $4 \times 128 = 512$ addresses needed — doesn't even fit in a /24.

---

## 4. CIDR Aggregation (Supernetting)

### The Rule

Two prefixes can be aggregated if and only if:

1. They are the same size ($/n$)
2. They are contiguous
3. The first prefix's network address has bit $(32 - n)$ equal to 0

$$\text{Aggregable:} \quad P_1/n + P_2/n \rightarrow P_1/(n-1) \quad \text{iff } P_1 \;\&\; 2^{(32-n)} = 0$$

### Worked Example

Can we aggregate 192.168.4.0/24 and 192.168.5.0/24?

- Same prefix length: /24 (yes)
- Contiguous: 4 and 5 are adjacent (yes)
- Third octet in binary: 00000100 and 00000101 — differ only in last bit (yes)
- 4 AND 1 = 0 (the lower address has the aggregation bit = 0) (yes)

**Result:** 192.168.4.0/23

### Multi-Level Aggregation

| Original Prefixes | Step 1 | Step 2 | Step 3 |
|:---|:---|:---|:---|
| 10.1.0.0/24 | 10.1.0.0/23 | 10.1.0.0/22 | 10.1.0.0/21 |
| 10.1.1.0/24 | | | |
| 10.1.2.0/24 | 10.1.2.0/23 | | |
| 10.1.3.0/24 | | | |
| 10.1.4.0/24 | 10.1.4.0/23 | 10.1.4.0/22 | |
| 10.1.5.0/24 | | | |
| 10.1.6.0/24 | 10.1.6.0/23 | | |
| 10.1.7.0/24 | | | |

8 prefixes reduced to 1 — $\log_2(8) = 3$ aggregation levels.

---

## 5. NAT State Table Sizing

### The Problem

NAT devices must track every active connection. How large does the state table grow?

### Port Space

Each NAT public IP provides:

$$\text{Ports} = 65,535 - 1,024 = 64,511 \text{ usable (above well-known)}$$

Per protocol (TCP/UDP independently), so effectively $\sim 129,022$ sessions per public IP.

### State Table Size

$$S = N_{internal} \times C_{avg}$$

Where $C_{avg}$ = average concurrent connections per internal host.

| Internal Hosts | Connections/Host | State Entries | Memory (~150 B/entry) |
|:---:|:---:|:---:|:---:|
| 100 | 50 | 5,000 | 750 KB |
| 1,000 | 100 | 100,000 | 15 MB |
| 10,000 | 200 | 2,000,000 | 300 MB |
| 100,000 | 200 | 20,000,000 | 3 GB |

### CGNAT Scaling

Carrier-Grade NAT (RFC 6888) recommends:

$$\text{Public IPs needed} = \frac{N_{subscribers} \times C_{avg}}{P_{usable}}$$

For 100,000 subscribers at 200 connections each:

$$\frac{100,000 \times 200}{64,511} \approx 310 \text{ public IPs}$$

---

## 6. Classful vs Classless — Historical Waste

### Classful Allocation

| Class | First Bits | Range | Networks | Hosts/Network | Total Addresses |
|:---|:---:|:---|:---:|:---:|:---:|
| A | 0 | 0-127 | 128 | 16,777,214 | $2^{31}$ |
| B | 10 | 128-191 | 16,384 | 65,534 | $2^{30}$ |
| C | 110 | 192-223 | 2,097,152 | 254 | $2^{29}$ |

**The waste problem:** An organization needing 300 hosts would get a Class B (65,534 hosts):

$$\text{Utilization} = \frac{300}{65,534} = 0.46\%$$

CIDR (/23 = 510 hosts) gives:

$$\text{Utilization} = \frac{300}{510} = 58.8\%$$

**Improvement: 128x better utilization.**

---

## 7. Fragmentation Math

### Maximum Transmission Unit

$$\text{Fragments} = \lceil \frac{L_{payload}}{MTU - 20} \rceil$$

Where 20 = IP header size (minimum).

| Payload | MTU 1500 | MTU 576 | MTU 296 |
|:---:|:---:|:---:|:---:|
| 1,480 B | 1 | 3 | 6 |
| 4,000 B | 3 | 8 | 15 |
| 65,515 B (max) | 45 | 118 | 237 |

**Fragment loss amplification:** If any fragment is lost, the entire datagram must be retransmitted:

$$P_{success} = (1 - p)^F$$

Where $p$ = per-packet loss rate, $F$ = number of fragments.

| Loss Rate | 1 Fragment | 3 Fragments | 10 Fragments |
|:---:|:---:|:---:|:---:|
| 0.1% | 99.9% | 99.7% | 99.0% |
| 1% | 99.0% | 97.0% | 90.4% |
| 5% | 95.0% | 85.7% | 59.9% |

This is why Path MTU Discovery (PMTUD) exists — avoiding fragmentation entirely.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $2^{32-n} - 2$ | Exponents | Usable hosts per subnet |
| $2^{n-p}$ | Exponents | Number of subnets |
| $\text{IP} \;\&\; \text{Mask}$ | Boolean AND | Network address |
| $32 - \lceil\log_2(H+2)\rceil$ | Logarithm | Optimal prefix length |
| $\lceil L / (MTU-20) \rceil$ | Ceiling division | Fragment count |
| $(1-p)^F$ | Exponential probability | Fragment loss |
| $N \times C / P$ | Rate ratio | CGNAT public IP requirement |

---

*Every packet on the internet carries a 32-bit source and destination address — and the binary math of subnetting, masking, and aggregation determines whether that packet reaches its destination or vanishes into a black hole.*
