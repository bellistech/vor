# The Mathematics of BIND — Authoritative DNS Server Internals

> *BIND (Berkeley Internet Name Domain) is the reference DNS server. The math covers zone file structure, query processing performance, DNSSEC signature overhead, and cache sizing.*

---

## 1. Zone File Structure — Record Count and Size

### The Model

A DNS zone contains resource records (RRs) organized by domain name. Zone size determines memory usage and transfer time.

### Zone Size Formula

$$\text{Zone Size} = \sum_{i=1}^{n} (\text{Name}_i + \text{TTL} + \text{Class} + \text{Type} + \text{RData}_i)$$

### Record Sizes (Wire Format)

| Record Type | RData Size | Typical Entry |
|:---|:---:|:---|
| A | 4 bytes | IPv4 address |
| AAAA | 16 bytes | IPv6 address |
| CNAME | Variable (name) | Alias target |
| MX | 2 + name bytes | Mail exchanger |
| TXT | Variable | Text string |
| NS | Variable (name) | Name server |
| SOA | ~30 bytes + names | Zone authority |
| SRV | 6 + name bytes | Service discovery |
| PTR | Variable (name) | Reverse lookup |

### Zone Size Estimation

$$\text{Zone Wire Size} \approx \text{Records} \times 50 \text{ bytes (avg including names)}$$

| Records | Zone File (text) | Wire Format | Transfer Time (1 Mbps) |
|:---:|:---:|:---:|:---:|
| 100 | 5 KiB | 5 KiB | <1 sec |
| 10,000 | 500 KiB | 500 KiB | 4 sec |
| 1,000,000 | 50 MiB | 50 MiB | 400 sec |
| 10,000,000 | 500 MiB | 500 MiB | 67 min |

---

## 2. Query Processing — Performance Model

### The Model

BIND processes queries through a pipeline: receive, parse, lookup, respond.

### Query Latency

$$T_{query} = T_{receive} + T_{parse} + T_{lookup} + T_{serialize} + T_{send}$$

### Lookup Complexity

BIND uses a red-black tree for zone data:

$$T_{lookup} = O(\log n) \quad \text{where } n = \text{names in zone}$$

### Query Throughput

$$\text{QPS} = \frac{\text{Worker Threads}}{T_{avg\_query}}$$

| Worker Threads | Avg Query Time | Max QPS |
|:---:|:---:|:---:|
| 1 | 0.01 ms | 100,000 |
| 4 | 0.01 ms | 400,000 |
| 8 | 0.01 ms | 800,000 |
| 16 | 0.01 ms | 1,600,000 |

Real-world QPS is lower due to cache misses, DNSSEC validation, and network overhead.

### Authoritative vs Recursive Performance

| Mode | Typical QPS | Bottleneck |
|:---|:---:|:---|
| Authoritative (cached) | 100K-1M | CPU |
| Recursive (cached) | 50K-500K | CPU |
| Recursive (cache miss) | 1K-10K | Upstream latency |

---

## 3. DNSSEC — Signature Math

### The Model

DNSSEC signs zone data with cryptographic keys. Signatures add size and CPU overhead.

### Signature Size

$$\text{RRSIG Size} \approx 85 + \text{Signature Bytes}$$

| Algorithm | Signature Size | RRSIG Total | Verify Speed |
|:---|:---:|:---:|:---:|
| RSA-2048 | 256 bytes | ~340 bytes | ~50K/s |
| RSA-4096 | 512 bytes | ~600 bytes | ~10K/s |
| ECDSA P-256 | 64 bytes | ~150 bytes | ~100K/s |
| Ed25519 | 64 bytes | ~150 bytes | ~200K/s |

### Zone Size with DNSSEC

$$\text{Signed Zone} = \text{Unsigned Zone} + \text{RRSIGs} + \text{NSEC/NSEC3 records} + \text{DNSKEY}$$

$$\text{Expansion Factor} \approx 3-10\times \text{(depending on algorithm and NSEC type)}$$

| Records | Unsigned | Signed (RSA-2048) | Signed (ECDSA) |
|:---:|:---:|:---:|:---:|
| 1,000 | 50 KiB | 400 KiB | 200 KiB |
| 100,000 | 5 MiB | 40 MiB | 20 MiB |
| 1,000,000 | 50 MiB | 400 MiB | 200 MiB |

### NSEC3 Hash Iterations

$$T_{nsec3\_hash} = \text{Iterations} \times T_{SHA1}$$

| Iterations | Hash Time | QPS Impact |
|:---:|:---:|:---:|
| 0 | ~0.001 ms | Minimal |
| 10 | ~0.01 ms | Low |
| 100 | ~0.1 ms | Moderate |
| 1000 | ~1 ms | Severe |

**RFC 9276 recommends 0 iterations** for NSEC3.

---

## 4. Zone Transfers — AXFR and IXFR

### AXFR (Full Transfer)

$$T_{AXFR} = \frac{\text{Zone Size}}{\text{Network BW}} + T_{setup}$$

### IXFR (Incremental Transfer)

$$T_{IXFR} = \frac{\text{Changed Records} \times \text{Avg Record Size}}{\text{Network BW}}$$

$$\text{Savings} = 1 - \frac{T_{IXFR}}{T_{AXFR}}$$

### Worked Example

*"Zone with 1M records (50 MiB), 1000 changes since last transfer."*

$$T_{AXFR} = \frac{50 \text{ MiB}}{100 \text{ Mbps}} = 4 \text{ sec}$$

$$T_{IXFR} = \frac{1000 \times 50}{100 \text{ Mbps}} = 0.004 \text{ sec}$$

$$\text{Savings} = 99.9\%$$

### NOTIFY Propagation

$$T_{propagation} = T_{NOTIFY} + T_{IXFR}$$

$$\text{Max Staleness} = \text{SOA Refresh Interval (if NOTIFY fails)}$$

---

## 5. Cache Sizing — Recursive Resolver

### The Model

BIND's recursive resolver caches answers up to their TTL.

### Cache Size Formula

$$\text{Cache Entries} = \text{Unique Queries} \times P(\text{TTL not expired})$$

$$\text{Cache Memory} = \text{Entries} \times \text{Avg Entry Size (150-300 bytes)}$$

### Cache Hit Ratio

$$\text{Hit Ratio} = 1 - \frac{\text{Cache Misses}}{\text{Total Queries}}$$

$$\text{Effective Latency} = \text{Hit Ratio} \times T_{cache} + (1 - \text{Hit Ratio}) \times T_{recursive}$$

Where $T_{cache} \approx 0.01$ ms, $T_{recursive} \approx 20-200$ ms.

| Hit Ratio | Effective Latency | Upstream Load |
|:---:|:---:|:---:|
| 50% | 10-100 ms | 50% of queries |
| 80% | 4-40 ms | 20% of queries |
| 90% | 2-20 ms | 10% of queries |
| 95% | 1-10 ms | 5% of queries |

### Memory Configuration

$$\text{max-cache-size} = \text{Available RAM} \times 0.5 \quad (\text{rule of thumb})$$

| Cache Size | Approximate Entries | Suitable For |
|:---:|:---:|:---|
| 64 MiB | ~300K | Small office |
| 256 MiB | ~1.2M | Medium ISP |
| 1 GiB | ~5M | Large ISP |
| 4 GiB | ~20M | Major resolver |

---

## 6. Rate Limiting (RRL)

### The Model

Response Rate Limiting prevents DNS amplification attacks.

### RRL Formula

$$\text{Allowed Rate} = \frac{\text{responses-per-second}}{\text{window}}$$

$$\text{Slip Rate} = \frac{1}{\text{slip value}} \quad (\text{fraction of suppressed responses sent as truncated})$$

### Amplification Factor

$$\text{Amplification} = \frac{\text{Response Size}}{\text{Query Size}}$$

| Query Type | Query Size | Response Size | Amplification |
|:---|:---:|:---:|:---:|
| A record | ~30 bytes | ~50 bytes | 1.7x |
| ANY (abused) | ~30 bytes | ~3000 bytes | 100x |
| DNSSEC ANY | ~30 bytes | ~5000 bytes | 167x |
| TXT (large) | ~30 bytes | ~500 bytes | 17x |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $O(\log n)$ | Logarithmic | Zone lookup |
| $\frac{\text{Threads}}{T_{query}}$ | Rate equation | QPS capacity |
| $3-10\times$ expansion | Multiplication | DNSSEC zone inflation |
| $\frac{\text{Zone Size}}{\text{BW}}$ | Rate equation | Transfer time |
| $\text{Hit} \times T_c + (1-\text{Hit}) \times T_r$ | Weighted average | Effective latency |
| $\frac{\text{Response}}{\text{Query}}$ | Ratio | Amplification factor |

---

*Every `named-checkzone`, `rndc reload`, and `dig` query runs through BIND's red-black tree lookup and caching layer — a DNS server that has been the reference implementation since 1988.*
