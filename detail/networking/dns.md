# The Mathematics of DNS — Recursion Trees, Caching, and DNSSEC Chains

> *DNS is a distributed hierarchical database with mathematically predictable lookup costs, cache decay curves, and cryptographic validation chains. Its efficiency comes from exponential fan-out at each tree level combined with aggressive TTL-governed caching.*

---

## 1. The Hierarchy — Tree Depth and Lookup Cost

### The Structure

DNS names form a tree of maximum depth $D$. A fully qualified domain name like `mail.example.com.` has:

$$D = \text{number of labels} = 4 \quad (\text{root}, \text{com}, \text{example}, \text{mail})$$

### Worst-Case Recursive Resolution

A recursive resolver, starting with no cache, must query each level:

$$Q_{max} = D + 1$$

The "+1" accounts for possible CNAME indirection or referral at each level.

### Worked Example: `www.subdomain.example.co.uk.`

| Step | Query | Server | Response |
|:---:|:---|:---|:---|
| 1 | `.` (root) | Root server | Referral to `.uk` |
| 2 | `.uk` | UK TLD | Referral to `.co.uk` |
| 3 | `.co.uk` | CO.UK registry | Referral to `example.co.uk` |
| 4 | `example.co.uk` | Auth NS | Referral to `subdomain` |
| 5 | `www.subdomain.example.co.uk` | Auth NS | Answer: A record |

5 queries, each adding ~1 RTT. At 20 ms average per query: **100 ms total**.

### Maximum Name Length

DNS names are limited to 253 characters, with each label max 63 characters:

$$L_{max} = 253 \text{ octets (total FQDN)}$$
$$L_{label} = 63 \text{ octets (per label)}$$
$$D_{max} = \lfloor 253 / 2 \rfloor = 126 \text{ labels (alternating 1-char labels + dots)}$$

In practice, depth rarely exceeds 5-6 levels.

---

## 2. TTL Decay and Cache Hit Ratio

### The Model

After a record is cached with TTL $T$ seconds, it decays linearly:

$$TTL_{remaining}(t) = T - t \quad \text{for } 0 \leq t \leq T$$

At $t = T$, the cache entry expires and must be re-fetched.

### Cache Hit Probability

If queries for a name arrive at rate $\lambda$ (queries/second) and the TTL is $T$ seconds:

$$P_{hit} = 1 - \frac{1}{\lambda \times T}$$

This assumes the first query after expiry is a miss, and all subsequent queries within the TTL window are hits.

### Worked Examples

| Query Rate ($\lambda$) | TTL ($T$) | Queries per TTL Window | $P_{hit}$ |
|:---:|:---:|:---:|:---:|
| 1/sec | 300 sec (5 min) | 300 | 99.7% |
| 1/sec | 60 sec (1 min) | 60 | 98.3% |
| 0.1/sec | 300 sec | 30 | 96.7% |
| 0.1/sec | 60 sec | 6 | 83.3% |
| 0.01/sec | 60 sec | 0.6 | 0% (mostly misses) |

### Optimal TTL Selection

Short TTL = faster failover, more queries. Long TTL = fewer queries, stale data risk.

**Query load on authoritative server:**

$$Q_{auth} = \frac{N_{resolvers}}{T}$$

Where $N_{resolvers}$ = number of caching resolvers querying the name.

| Resolvers | TTL = 60s | TTL = 300s | TTL = 3600s |
|:---:|:---:|:---:|:---:|
| 1,000 | 16.7 qps | 3.3 qps | 0.28 qps |
| 10,000 | 167 qps | 33 qps | 2.8 qps |
| 100,000 | 1,667 qps | 333 qps | 28 qps |

---

## 3. DNS Message Size and Truncation

### The Limits

| Transport | Max Size | Notes |
|:---|:---:|:---|
| UDP (classic) | 512 bytes | RFC 1035 |
| UDP + EDNS0 | 4,096 bytes (typical) | RFC 6891, up to 65,535 |
| TCP | 65,535 bytes | RFC 7766 |

### Record Packing

A DNS response with $N$ records, each of average size $S$ bytes:

$$\text{Response size} = H + \sum_{i=1}^{N} S_i$$

Where $H = 12$ bytes (header) + question section.

| Record Type | Typical Size | 10 Records |
|:---|:---:|:---:|
| A (IPv4) | 16 bytes | 160 bytes |
| AAAA (IPv6) | 28 bytes | 280 bytes |
| MX | 30 bytes | 300 bytes |
| TXT (SPF) | 50-200 bytes | 500-2,000 bytes |
| DNSKEY (DNSSEC) | 200-300 bytes | 2,000-3,000 bytes |

DNSSEC records frequently push responses past 512 bytes, making EDNS0 mandatory.

---

## 4. DNSSEC Chain Validation

### The Problem

DNSSEC creates a chain of trust from the root zone to the queried record. Each link requires cryptographic verification.

### Chain Depth

$$V_{total} = \sum_{i=1}^{D} (S_i + K_i)$$

Where:
- $D$ = domain depth
- $S_i$ = RRSIG verifications at level $i$
- $K_i$ = DNSKEY/DS lookups at level $i$

### Worked Example: Validating `www.example.com`

| Step | Operation | Crypto Work |
|:---:|:---|:---|
| 1 | Fetch `.com` DS from root | Verify RRSIG with root DNSKEY |
| 2 | Fetch `.com` DNSKEY | Verify DNSKEY matches DS hash |
| 3 | Fetch `example.com` DS from `.com` | Verify RRSIG with `.com` DNSKEY |
| 4 | Fetch `example.com` DNSKEY | Verify DNSKEY matches DS hash |
| 5 | Fetch `www.example.com` A | Verify RRSIG with `example.com` DNSKEY |

Total: **5 signature verifications** + **2 DS hash checks** + **5 additional queries** (worst case).

### Crypto Cost per Validation

| Algorithm | Key Size | Sign (ms) | Verify (ms) |
|:---|:---:|:---:|:---:|
| RSA-2048 (algo 8) | 2048-bit | 2.0 | 0.05 |
| ECDSA P-256 (algo 13) | 256-bit | 0.1 | 0.15 |
| Ed25519 (algo 15) | 256-bit | 0.05 | 0.08 |

For a 5-level validation chain with RSA-2048: $5 \times 0.05 = 0.25$ ms verification time (CPU), negligible compared to network RTTs.

### NSEC/NSEC3 — Proving Non-Existence

NSEC3 uses iterated hashing to prevent zone enumeration:

$$H_{final} = \underbrace{H(H(\ldots H(}_{iterations}\text{name} \| \text{salt})\ldots))$$

| Iterations | Hash Operations | Time per Query |
|:---:|:---:|:---:|
| 0 | 1 | ~0.001 ms |
| 10 | 11 | ~0.01 ms |
| 100 | 101 | ~0.1 ms |
| 1,000 | 1,001 | ~1 ms |

RFC 9276 recommends 0 iterations (NSEC3 with salt provides sufficient protection).

---

## 5. Anycast and Server Selection Math

### The Problem

Root servers and large DNS providers use anycast — one IP address announced from multiple locations. The client reaches the "nearest" instance (by BGP metric).

### Root Server Capacity

13 root server identities (A through M), with actual instances:

$$N_{instances} = \sum_{i=A}^{M} n_i > 1,500 \text{ globally}$$

### Query Load Distribution

If $Q_{total}$ queries/sec are distributed across $N$ anycast instances:

$$Q_{per\_instance} = \frac{Q_{total}}{N} \quad \text{(ideal uniform)}$$

In practice, distribution is skewed by geography. The busiest instance might handle $3\times$ the average.

---

## 6. Zone Transfer Sizing

### Full Transfer (AXFR)

$$\text{Transfer size} = R \times S_{avg}$$

Where $R$ = total records, $S_{avg}$ = average record size.

### Incremental Transfer (IXFR)

$$\text{Transfer size} = \Delta R \times S_{avg}$$

Where $\Delta R$ = records changed since last serial.

| Zone Size | AXFR Size | Daily Changes (1%) | IXFR Size | Savings |
|:---:|:---:|:---:|:---:|:---:|
| 10,000 records | ~500 KB | 100 | ~5 KB | 99% |
| 100,000 records | ~5 MB | 1,000 | ~50 KB | 99% |
| 1,000,000 records | ~50 MB | 10,000 | ~500 KB | 99% |

SOA serial number format (YYYYMMDDnn) allows 100 changes per day before rolling over.

---

## 7. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $D + 1$ queries | Linear (tree depth) | Recursive resolution cost |
| $1 - 1/(\lambda T)$ | Probability | Cache hit ratio |
| $N_{resolvers}/T$ | Rate / inverse | Auth server query load |
| $12 + \sum S_i$ | Summation | Response size estimation |
| $\sum (S_i + K_i)$ | Chain summation | DNSSEC validation cost |
| $H^{n}(\text{name}\|\text{salt})$ | Iterated hash | NSEC3 computation |
| $\Delta R \times S_{avg}$ | Delta calculation | IXFR transfer sizing |

---

*DNS resolves billions of queries per day, and the math behind caching, TTLs, and DNSSEC chain validation determines whether your browser loads a page in 50 ms or 500 ms.*
