# curl Deep Dive — Theory & Internals

> *curl is a data transfer tool that speaks dozens of protocols. Understanding its internals means understanding connection timing decomposition, HTTP/2 multiplexing math, retry backoff strategies, and the bandwidth-limiting token bucket that governs --limit-rate.*

---

## 1. Connection Timing Decomposition

### The `-w` Timing Variables

curl's `--write-out` exposes each phase of a transfer:

$$T_{total} = T_{DNS} + T_{connect} + T_{TLS} + T_{TTFB} + T_{transfer}$$

Formally:

$$T_{total} = \text{time\_total}$$
$$T_{DNS} = \text{time\_namelookup}$$
$$T_{connect} = \text{time\_connect} - \text{time\_namelookup}$$
$$T_{TLS} = \text{time\_appconnect} - \text{time\_connect}$$
$$T_{TTFB} = \text{time\_starttransfer} - \text{time\_appconnect}$$
$$T_{transfer} = \text{time\_total} - \text{time\_starttransfer}$$

### Worked Example

```
time_namelookup:  0.012
time_connect:     0.045
time_appconnect:  0.112
time_starttransfer: 0.156
time_total:       0.892
```

| Phase | Duration | % of Total |
|:---|:---:|:---:|
| DNS resolution | 12 ms | 1.3% |
| TCP connect | 33 ms | 3.7% |
| TLS handshake | 67 ms | 7.5% |
| Server processing (TTFB) | 44 ms | 4.9% |
| Data transfer | 736 ms | 82.5% |
| **Total** | **892 ms** | **100%** |

### Bottleneck Identification

$$\text{Bottleneck} = \arg\max(T_{DNS}, T_{connect}, T_{TLS}, T_{TTFB}, T_{transfer})$$

Common patterns:
- $T_{DNS} >> 100$ ms → DNS resolution problem
- $T_{connect} >> RTT$ → network path issue
- $T_{TLS} >> 2 \times RTT$ → TLS overhead or cipher negotiation
- $T_{TTFB} >> 200$ ms → server processing bottleneck
- $T_{transfer} >> \text{expected}$ → bandwidth constraint

---

## 2. Parallel Transfers — Connection Pooling

### HTTP/1.1 Pipeline Limitations

With $C$ connections and $R$ resources:

$$T_{HTTP/1.1} = \frac{R}{C} \times (T_{TTFB} + T_{transfer\_avg})$$

Default: curl opens 1 connection. With `--parallel` (HTTP/2):

### HTTP/2 Multiplexing

$$T_{HTTP/2} = T_{connect} + T_{TLS} + \max_{i}(T_{transfer_i})$$

All streams share one connection. Transfer time is bounded by the largest resource.

### Parallel Download Comparison

Downloading 10 files, each 1 MB, server TTFB = 50 ms:

| Method | Connections | Formula | Time (100 Mbps) |
|:---|:---:|:---|:---:|
| Sequential | 1 | $10 \times (50 + 80)$ ms | 1,300 ms |
| Parallel (6 conn) | 6 | $\lceil 10/6 \rceil \times 130$ ms | 260 ms |
| HTTP/2 multiplex | 1 | $50 + 80$ ms | 130 ms |

HTTP/2 multiplexing is 10x faster than sequential for many small files.

---

## 3. Rate Limiting — Token Bucket Model

### `--limit-rate` Implementation

curl uses a token bucket to enforce bandwidth limits:

$$\text{Tokens}(t) = \min(B, \text{tokens}_{prev} + R \times \Delta t)$$

Where:
- $R$ = rate limit (bytes/sec)
- $B$ = burst size (buffer = $R$ bytes by default)

### Transfer Time Under Rate Limit

$$T = \frac{S}{R}$$

| File Size | Rate Limit | Transfer Time |
|:---:|:---:|:---:|
| 10 MB | 1 MB/s | 10 sec |
| 100 MB | 500 KB/s | 200 sec |
| 1 GB | 10 MB/s | 100 sec |

### Interaction with TCP Window

If rate limit < link capacity, TCP window self-limits:

$$W_{steady} \approx R \times RTT$$

At `--limit-rate 1M` and RTT = 50 ms: $W = 1,000,000 \times 0.05 = 50,000$ bytes.

---

## 4. Retry Math — Backoff Strategy

### `--retry` Mechanism

curl retries with exponential backoff:

$$T_{wait}(n) = \min(T_{base} \times 2^{n-1}, T_{max})$$

Default: $T_{base} = 1$ sec, $T_{max} = 600$ sec.

| Retry # | Wait Time | Cumulative |
|:---:|:---:|:---:|
| 1 | 1 sec | 1 sec |
| 2 | 2 sec | 3 sec |
| 3 | 4 sec | 7 sec |
| 4 | 8 sec | 15 sec |
| 5 | 16 sec | 31 sec |
| 10 | 512 sec | 1,023 sec |

### `--retry-max-time` Constraint

$$\text{Actual retries} = \max(n) \text{ where } \sum_{i=1}^{n} T_{wait}(i) + T_{attempt}(i) \leq T_{max\_time}$$

---

## 5. Resume Capability — Range Request Math

### `--continue-at` / `-C -`

curl uses HTTP Range headers:

$$\text{Range: bytes=}B_{received}\text{-}$$

### Transfer Efficiency with Interruptions

If a download is interrupted $K$ times, each resumption incurs one RTT + TLS overhead:

$$T_{total} = T_{transfer} + K \times (T_{connect} + T_{TLS})$$

**Overhead ratio:**

$$O = \frac{K \times T_{overhead}}{T_{transfer}}$$

| File Size | Interruptions | Overhead per Resume | Total Overhead |
|:---:|:---:|:---:|:---:|
| 100 MB | 2 | 200 ms | 400 ms / ~10 sec transfer = 4% |
| 1 GB | 5 | 200 ms | 1 sec / ~100 sec transfer = 1% |
| 10 GB | 10 | 200 ms | 2 sec / ~1000 sec transfer = 0.2% |

---

## 6. Cookie and Header Overhead

### Per-Request Overhead

$$S_{request} = L_{method} + L_{URL} + \sum_{i} L_{header_i} + L_{cookies}$$

### Cookie Growth Impact

| Cookies | Avg Size | Cookie Header | Requests (1000) | Total Overhead |
|:---:|:---:|:---:|:---:|:---:|
| 5 | 50 B | 250 B | 1,000 | 250 KB |
| 20 | 100 B | 2,000 B | 1,000 | 2 MB |
| 50 | 200 B | 10,000 B | 1,000 | 10 MB |

With HTTP/2 header compression (HPACK), repeated headers are sent as indices:

$$S_{compressed} \approx S_{first} + (N-1) \times S_{index}$$

Where $S_{index} \approx 2$-$3$ bytes per previously-seen header.

---

## 7. DNS Caching — Resolution Frequency

### `--dns-cache-time`

Default: 60 seconds. DNS lookups saved:

$$\text{Lookups saved} = N_{requests} - \lceil \frac{T_{total}}{T_{cache}} \rceil$$

| Requests | Duration | Cache TTL | Lookups (no cache) | Lookups (cached) |
|:---:|:---:|:---:|:---:|:---:|
| 100 | 60 sec | 60 sec | 100 | 1 |
| 1,000 | 300 sec | 60 sec | 1,000 | 5 |
| 10,000 | 600 sec | 60 sec | 10,000 | 10 |

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $T_{DNS} + T_{connect} + T_{TLS} + T_{TTFB} + T_{transfer}$ | Summation | Total transfer time |
| $R/C \times T_{avg}$ | Division | HTTP/1.1 parallel downloads |
| $\min(B, \text{tokens} + R \times \Delta t)$ | Token bucket | Rate limiting |
| $T_{base} \times 2^{n-1}$ | Exponential backoff | Retry timing |
| $T_{transfer} + K \times T_{overhead}$ | Linear | Resume overhead |

---

*curl's timing output is the best free diagnostic tool in networking — it decomposes every HTTP transfer into its constituent phases, letting you identify whether the bottleneck is DNS, TLS, the server, or the network. The math is in the measurement.*
