# The Mathematics of HTTP — Multiplexing, Head-of-Line Blocking & Connection Economics

> *HTTP's evolution from 1.1 to 3 is a study in queuing theory and protocol overhead. Each version attacks the same problem: how to maximize throughput across an unreliable network while minimizing latency. The math reveals exactly why multiplexing wins and when it doesn't.*

---

## 1. Connection Economics — HTTP/1.1 (Queueing Theory)

### The Problem

HTTP/1.1 uses persistent connections but processes requests sequentially (one response at a time per connection). Browsers open multiple connections to parallelize. What is the optimal number of connections?

### The Formula

Page load time with $k$ connections and $n$ resources:

$$T_{page} = \left\lceil \frac{n}{k} \right\rceil \times \bar{T}_{response} + T_{connect}$$

Where:
- $n$ = number of resources on the page
- $k$ = parallel connections (browser limit: typically 6)
- $\bar{T}_{response}$ = average response time per resource
- $T_{connect}$ = TCP + TLS setup cost (amortized across requests)

### Worked Examples

| Resources $n$ | Connections $k$ | Avg Response $\bar{T}$ | Serial Rounds | Load Time |
|:---:|:---:|:---:|:---:|:---:|
| 50 | 1 | 100 ms | 50 | 5.0 s |
| 50 | 6 | 100 ms | 9 | 0.9 s |
| 100 | 6 | 50 ms | 17 | 0.85 s |
| 100 | 6 | 200 ms | 17 | 3.4 s |

### Connection Setup Cost

$$T_{setup} = T_{TCP} + T_{TLS}$$

$$T_{TCP} = 1.5 \times RTT \quad \text{(SYN, SYN-ACK, ACK)}$$

$$T_{TLS_{1.2}} = 2 \times RTT, \quad T_{TLS_{1.3}} = 1 \times RTT$$

Total for new HTTPS connection (TLS 1.3):

$$T_{setup} = 1.5 \times RTT + 1 \times RTT = 2.5 \times RTT$$

For 6 parallel connections at 50 ms RTT: $6 \times 125$ ms = 750 ms just for setup (though these happen in parallel, so it is 125 ms wall-clock if concurrent).

### Keep-Alive Savings

Without keep-alive, each request pays $T_{setup}$. With keep-alive:

$$\text{Saved} = (n - k) \times T_{setup}$$

For 100 requests with 6 connections at 50 ms RTT:

$$\text{Saved} = 94 \times 125 \text{ ms} = 11.75 \text{ seconds}$$

---

## 2. HTTP/2 Multiplexing (Stream Concurrency)

### The Problem

HTTP/2 multiplexes all requests as streams over a single TCP connection. How does this compare to HTTP/1.1's multi-connection approach?

### The Formula

HTTP/2 page load time:

$$T_{page}^{H2} = T_{setup} + \max(T_{stream_1}, T_{stream_2}, \ldots, T_{stream_n})$$

All $n$ streams start nearly simultaneously. The total time is dominated by the slowest stream (largest resource) plus framing overhead.

Effective throughput:

$$\Theta_{H2} = \frac{\sum_{i=1}^{n} S_i}{T_{page}^{H2}}$$

Where $S_i$ is the size of resource $i$.

### Frame Overhead

Each HTTP/2 frame has a 9-byte header:

$$\text{overhead} = \frac{9}{S_{frame}} \times 100\%$$

| Frame Payload | Frame Total | Overhead |
|:---:|:---:|:---:|
| 100 bytes | 109 bytes | 9.0% |
| 1,000 bytes | 1,009 bytes | 0.9% |
| 16,384 bytes (max default) | 16,393 bytes | 0.055% |

### Stream Concurrency Limit

HTTP/2 `SETTINGS_MAX_CONCURRENT_STREAMS` defaults to 100 (server-configurable). If a page has $n > S_{max}$ resources:

$$\text{Rounds} = \left\lceil \frac{n}{S_{max}} \right\rceil$$

In practice, $S_{max} = 100$ is rarely the bottleneck. Bandwidth and server processing are.

---

## 3. Head-of-Line Blocking Analysis (Probability)

### The Problem

HTTP/2 over TCP suffers from TCP-level head-of-line (HoL) blocking: if one TCP segment is lost, all streams stall until retransmission. How often does this happen?

### The Formula

Probability that at least one of $n$ concurrent streams is blocked by a single packet loss:

$$P(\text{any stream blocked}) = 1 - (1 - p)^{W}$$

Where:
- $p$ = packet loss rate
- $W$ = TCP congestion window size (in packets)

Given a loss affects a random position, the expected number of stalled streams:

$$E[\text{stalled streams}] = n \times P(\text{stream's packet lost})$$

For uniform data distribution across $n$ streams:

$$P(\text{specific stream hit}) = \frac{S_i / MSS}{W} \approx \frac{1}{n}$$

### Worked Examples

| Loss Rate $p$ | Window $W$ (pkts) | $P(\text{HoL block})$ | Streams Stalled (of 50) |
|:---:|:---:|:---:|:---:|
| 0.01% | 100 | 0.99% | 0.5 |
| 0.1% | 100 | 9.5% | 4.75 |
| 0.5% | 100 | 39.4% | 19.7 |
| 1.0% | 100 | 63.4% | 31.7 |
| 2.0% | 100 | 86.7% | 43.4 |
| 0.1% | 300 | 25.9% | 13.0 |

### HTTP/3 (QUIC) Solution

QUIC isolates streams at the transport layer. A lost packet only stalls the stream it belongs to:

$$P(\text{specific stream blocked}) = 1 - (1-p)^{w_i}$$

Where $w_i$ is the number of packets in flight for stream $i$ only. With 50 concurrent streams, $w_i \approx W/50$:

$$P(\text{stream blocked})_{QUIC} = 1 - (1-p)^{W/n}$$

At $p = 1\%$, $W = 100$, $n = 50$:
- HTTP/2 (TCP): 63.4% chance **all** streams stall
- HTTP/3 (QUIC): 1.98% chance **each** stream stalls independently

---

## 4. Connection Coalescing (HTTP/2 Optimization)

### The Problem

HTTP/2 allows reusing a connection to `origin-a.com` for `origin-b.com` if they share the same IP and TLS certificate. How much does this save?

### The Formula

Without coalescing, connections needed:

$$C_{no\_coal} = D \quad \text{(one per domain)}$$

With coalescing:

$$C_{coal} = |\{IP_i\}| \quad \text{(one per unique server IP)}$$

Savings:

$$\text{Saved connections} = D - C_{coal}$$

$$\text{Saved time} = (D - C_{coal}) \times T_{setup}$$

### Worked Examples

A typical CDN serves `cdn1.example.com`, `cdn2.example.com`, `static.example.com`, `api.example.com` — all resolving to the same IP with a wildcard cert:

| Domains | Unique IPs | Connections (no coal) | Connections (coal) | Saved |
|:---:|:---:|:---:|:---:|:---:|
| 4 | 1 | 4 | 1 | 3 |
| 10 | 2 | 10 | 2 | 8 |
| 20 | 3 | 20 | 3 | 17 |

At 50 ms RTT, saving 8 connections saves $8 \times 125$ ms = 1 second of setup latency.

---

## 5. HPACK Header Compression (Information Theory)

### The Problem

HTTP headers are highly repetitive across requests. HPACK uses a static table (61 common headers), a dynamic table, and Huffman coding. What compression ratios are achievable?

### The Formula

Compression ratio:

$$R = 1 - \frac{S_{compressed}}{S_{uncompressed}}$$

A typical HTTP/1.1 request has ~800 bytes of headers. On a warm HTTP/2 connection:

| Request # | Uncompressed | Compressed | Ratio |
|:---:|:---:|:---:|:---:|
| 1 (cold) | 800 B | 400 B | 50% |
| 2 | 800 B | 50 B | 93.8% |
| 3+ | 800 B | 20-30 B | 96-97% |

### Dynamic Table Sizing

The dynamic table has a maximum size (default 4,096 bytes, configurable via SETTINGS). Each entry occupies:

$$S_{entry} = |\text{name}| + |\text{value}| + 32$$

The 32-byte overhead accounts for the entry structure in the table.

Maximum entries in default table:

$$N_{entries} = \frac{4096}{|\text{avg name}| + |\text{avg value}| + 32}$$

For typical headers (name ~15 bytes, value ~30 bytes):

$$N_{entries} = \frac{4096}{15 + 30 + 32} \approx 53 \text{ entries}$$

### Bandwidth Savings Over Many Requests

For $n$ requests with average header size $H$:

$$\text{HTTP/1.1 overhead} = n \times H$$
$$\text{HTTP/2 overhead} \approx H + (n-1) \times H \times (1 - R_{warm})$$

For 100 requests, H = 800 bytes, $R_{warm}$ = 0.95:

- HTTP/1.1: $100 \times 800 = 80,000$ bytes
- HTTP/2: $800 + 99 \times 40 = 4,760$ bytes
- Savings: 94%

---

## 6. Keep-Alive Economics (Cost-Benefit Analysis)

### The Problem

Persistent connections save setup cost but consume server resources (memory, file descriptors). What is the optimal idle timeout?

### The Formula

Cost of a new connection:

$$C_{new} = T_{setup} + M_{state}$$

Cost of keeping a connection idle for time $t$:

$$C_{idle}(t) = t \times R_{mem} + FD_{cost}$$

Break-even idle time:

$$t_{break} = \frac{T_{setup}}{R_{mem}}$$

### Server Resource Consumption

Each idle HTTP/2 connection consumes:
- ~10 KB kernel memory (TCP buffers, minimal)
- 1 file descriptor
- ~50 KB application memory (HTTP/2 state, HPACK tables)

For a server with 10,000 idle connections:

$$M_{total} = 10,000 \times 60 \text{ KB} = 600 \text{ MB}$$

### Optimal Timeout

If request inter-arrival time follows exponential distribution with rate $\mu$:

$$P(\text{reuse within } t) = 1 - e^{-\mu t}$$

| $\mu$ (req/sec) | $P(\text{reuse in 30s})$ | $P(\text{reuse in 60s})$ | $P(\text{reuse in 120s})$ |
|:---:|:---:|:---:|:---:|
| 0.1 | 95.0% | 99.8% | ~100% |
| 0.01 | 25.9% | 45.1% | 69.9% |
| 0.001 | 2.96% | 5.82% | 11.3% |

For a client making 1 request every 100 seconds ($\mu = 0.01$), a 60-second timeout only gives 45% reuse probability. Not worth the server resources.

---

## 7. Summary of Formulas

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\lceil n/k \rceil \times \bar{T}$ | Ceiling division | HTTP/1.1 load time |
| $T_{TCP} + T_{TLS}$ | Additive latency | Connection setup |
| $9/S_{frame} \times 100\%$ | Ratio | HTTP/2 frame overhead |
| $1 - (1-p)^W$ | Complement probability | HoL blocking |
| $1 - S_c/S_u$ | Compression ratio | HPACK efficiency |
| $|name| + |value| + 32$ | Linear | HPACK entry size |
| $1 - e^{-\mu t}$ | Exponential CDF | Connection reuse |

## Prerequisites

- queueing theory, probability, exponential distribution, information theory

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| HTTP/1.1 request parsing | $O(H)$ header bytes | $O(H)$ |
| HTTP/2 HPACK decode | $O(H)$ per header | $O(T)$ dynamic table |
| HTTP/2 stream multiplex | $O(1)$ per frame | $O(S)$ per stream |
| HTTP/2 priority tree | $O(\log n)$ reweight | $O(n)$ stream tree |
| QUIC packet demux | $O(1)$ connection ID | $O(n)$ streams |

---

*HTTP's evolution is a mathematical journey from sequential queuing (1.1) to concurrent multiplexing (2) to loss-isolated streams (3). Each step solves one equation's bottleneck only to reveal the next. The fundamental tension between connection cost and parallelism drives every design decision.*
