# dig Deep Dive — Theory & Internals

> *dig (Domain Information Groper) is the canonical DNS diagnostic tool. Understanding its output means understanding DNS message structure, query timing analysis, DNSSEC validation chains, and the iterative vs recursive resolution paths that determine lookup performance.*

---

## 1. Query Timing — RTT Decomposition

### The `Query time` Field

dig reports the total round-trip time for the DNS query:

$$T_{query} = T_{network\_RTT} + T_{server\_processing}$$

### Factors Affecting Query Time

| Factor | Impact | Typical Range |
|:---|:---|:---:|
| Network RTT to resolver | Dominant for remote servers | 1-200 ms |
| Server cache status | Cache hit = fast, miss = recursive | 0-500 ms |
| DNSSEC validation | Additional crypto operations | 1-50 ms |
| TCP fallback (large response) | 3-way handshake overhead | +1 RTT |
| EDNS0 buffer size | Larger responses avoid truncation | -1 RTT if avoids TCP |

### Diagnostic Patterns

| Query Time | Likely Explanation |
|:---:|:---|
| < 1 ms | Local cache hit (resolver on localhost) |
| 1-5 ms | LAN resolver, cached |
| 5-30 ms | LAN resolver, recursive lookup |
| 30-100 ms | Remote resolver or uncached response |
| 100-500 ms | Slow authoritative chain or DNSSEC validation |
| > 1,000 ms | Timeout/retry occurred |

---

## 2. DNS Message Structure — Byte Counting

### Message Sections

$$S_{message} = H + Q + A + N + D$$

Where:
- $H$ = Header (12 bytes, always)
- $Q$ = Question section
- $A$ = Answer section
- $N$ = Authority section
- $D$ = Additional section

### Section Sizes

| Section | Per-Record Size | Purpose |
|:---|:---:|:---|
| Header | 12 B (fixed) | Flags, counts |
| Question | ~20-50 B | Query name + type + class |
| Answer | Varies by type | The actual records |
| Authority | ~50-100 B per NS | Authoritative nameservers |
| Additional | ~16-40 B per A/AAAA | Glue records |

### Record Size by Type

| Type | Typical Size | Formula |
|:---|:---:|:---|
| A | $L_{name} + 14$ | Name + type(2) + class(2) + TTL(4) + rdlen(2) + IP(4) |
| AAAA | $L_{name} + 26$ | Same + IPv6(16) |
| MX | $L_{name} + 14 + L_{exchange}$ | Preference(2) + exchange name |
| TXT | $L_{name} + 12 + L_{text}$ | Variable-length text |
| RRSIG | $L_{name} + 30 + L_{sig}$ | Signature data (~128-256 B) |

### Name Compression

DNS uses message compression — repeated domain names are replaced with 2-byte pointers:

$$S_{compressed} = 2 \text{ bytes (pointer)}$$

$$\text{Savings per repeated name} = L_{name} - 2$$

For `www.example.com` (17 bytes): saves 15 bytes per repetition.

---

## 3. Iterative vs Recursive — Query Count

### Recursive Query (`+recurse`, default)

$$Q_{client} = 1 \quad \text{(resolver does all the work)}$$

### Iterative Resolution (`+trace`)

dig's `+trace` shows the full resolution chain:

$$Q_{total} = 1 + D$$

Where $D$ = depth of the domain hierarchy.

### Worked Example: `+trace www.example.com`

| Step | Query To | Response | New Referral |
|:---:|:---|:---|:---|
| 1 | Root (.) | NS for .com | a.gtld-servers.net |
| 2 | .com TLD | NS for example.com | ns1.example.com |
| 3 | example.com | A record for www | 93.184.216.34 |

Total: 3 queries, each adding network RTT.

$$T_{trace} = \sum_{i=1}^{D} (RTT_i + T_{process_i})$$

---

## 4. DNSSEC Verification — `+dnssec` Output

### The Chain dig Reveals

With `+dnssec`, dig requests DNSSEC records and shows:

$$\text{Validated chain:} \quad \text{Root} \xrightarrow{DS/DNSKEY} \text{TLD} \xrightarrow{DS/DNSKEY} \text{Zone} \xrightarrow{RRSIG} \text{Record}$$

### Additional Records per DNSSEC Query

| Without DNSSEC | With DNSSEC | Overhead |
|:---|:---|:---:|
| A record (~50 B) | A + RRSIG (~300 B) | 6x |
| NS records (~200 B) | NS + RRSIG + DS (~600 B) | 3x |
| Full response (~200 B) | Full + DNSSEC (~800 B) | 4x |

### AD Flag (Authenticated Data)

$$\text{AD} = \begin{cases} 1 & \text{if resolver validated the full DNSSEC chain} \\ 0 & \text{if unsigned or validation not performed} \end{cases}$$

---

## 5. EDNS0 — Buffer Size Negotiation

### The Problem

Classic DNS: 512-byte UDP limit. DNSSEC responses exceed this.

### EDNS0 OPT Record

dig includes an OPT record advertising its buffer size:

$$\text{EDNS buffer} = B \text{ bytes (typically 4096)}$$

### Response Truncation Logic

$$\text{If } S_{response} > \min(B_{client}, B_{server}):$$
$$\quad \text{Truncate (TC=1) → client retries over TCP}$$

### TCP Fallback Cost

$$T_{TCP} = T_{UDP} + T_{handshake} = T_{UDP} + 1.5 \times RTT$$

---

## 6. Batch Queries — Throughput Analysis

### Sequential Queries

$$T_{batch} = N \times T_{query}$$

### Parallel Queries (Multiple dig Processes)

$$T_{parallel} = \frac{N}{P} \times T_{query}$$

Where $P$ = parallel processes.

| Queries | Sequential (50 ms each) | Parallel (P=10) | Parallel (P=50) |
|:---:|:---:|:---:|:---:|
| 100 | 5 sec | 0.5 sec | 0.1 sec |
| 1,000 | 50 sec | 5 sec | 1 sec |
| 10,000 | 500 sec | 50 sec | 10 sec |

### Server Rate Limiting

Many resolvers rate-limit at $R_{max}$ queries/sec:

$$T_{rate\_limited} = \max\left(\frac{N}{P} \times T_{query}, \frac{N}{R_{max}}\right)$$

Common public resolver limits: 1,000-10,000 qps per source IP.

---

## 7. Output Parsing — Key Fields

### Answer Section Anatomy

```
;; ANSWER SECTION:
example.com.    86400   IN  A   93.184.216.34
```

| Field | Value | Meaning |
|:---|:---:|:---|
| Name | example.com. | Queried domain (FQDN) |
| TTL | 86400 | Seconds until cache expiry |
| Class | IN | Internet class |
| Type | A | IPv4 address record |
| RDATA | 93.184.216.34 | The answer |

### TTL Decay Observation

Repeated queries show TTL counting down:

$$TTL_{remaining} = TTL_{original} - (t_{now} - t_{cached})$$

Query at $t=0$: TTL=86400. Query at $t=100$: TTL=86300. This proves the resolver has it cached.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $T_{network} + T_{server}$ | Summation | Query time decomposition |
| $H + Q + A + N + D$ | Summation | Message size |
| $L_{name} - 2$ per repetition | Savings | Name compression |
| $1 + D$ queries | Linear | Iterative resolution depth |
| $T_{UDP} + 1.5 \times RTT$ | Summation | TCP fallback cost |
| $N / P \times T_{query}$ | Division | Parallel query throughput |

---

*dig is the stethoscope of DNS — every field in its output maps to a protocol behavior, and learning to read the timing, flags, and sections is the difference between guessing at DNS problems and diagnosing them precisely.*
