# The Mathematics of NFS -- Network Latency, Cache Coherence, and Delegation Models

> *NFS performance is bounded by network round-trip time and server processing latency. The mathematics cover the cost model for remote I/O operations, the close-to-open cache consistency protocol, delegation state machines for reducing server load, and the throughput equations that govern parallel NFS (pNFS) layouts.*

---

## 1. NFS Latency Model (Network + Server)

### Single Operation Latency

The latency of an NFS operation is the sum of network and server components:

$$T_{\text{op}} = T_{\text{RTT}} + T_{\text{server}} + T_{\text{data}}$$

where:
- $T_{\text{RTT}}$ = network round-trip time (propagation + serialization)
- $T_{\text{server}}$ = server processing time (lookup, disk I/O, journaling)
- $T_{\text{data}}$ = data transfer time = $\frac{\text{payload size}}{\text{bandwidth}}$

### Read Latency Breakdown

$$T_{\text{read}}(n) = T_{\text{RTT}} + T_{\text{lookup}} + T_{\text{disk}} + \frac{n}{B}$$

where $n$ is bytes read and $B$ is the effective bandwidth.

| Component | LAN (1Gbps) | WAN (100Mbps) | Cross-Region |
|:---|:---:|:---:|:---:|
| $T_{\text{RTT}}$ | 0.2 ms | 20 ms | 80 ms |
| $T_{\text{lookup}}$ | 0.1 ms | 0.1 ms | 0.1 ms |
| $T_{\text{disk}}$ (SSD) | 0.1 ms | 0.1 ms | 0.1 ms |
| $T_{\text{data}}$ (1 MB) | 8 ms | 80 ms | 80 ms |
| **Total** | **8.4 ms** | **100.2 ms** | **160.2 ms** |

### Metadata Operation Cost

Metadata operations (stat, readdir, open, close) are RTT-dominated because the payload is small:

$$T_{\text{metadata}} \approx T_{\text{RTT}} + T_{\text{server}} \quad (\text{since } T_{\text{data}} \approx 0)$$

A directory listing with 1000 entries on a WAN link:

$$T_{\text{readdir}} = \lceil \frac{1000}{\text{entries per RPC}} \rceil \times T_{\text{RTT}} = \lceil \frac{1000}{170} \rceil \times 20\text{ms} = 6 \times 20 = 120\text{ms}$$

---

## 2. Sequential I/O Throughput (Pipeline Analysis)

### Without Read-Ahead

Sequential read of a file of size $F$ with buffer size $r$ (rsize):

$$T_{\text{sequential}} = \frac{F}{r} \times (T_{\text{RTT}} + \frac{r}{B})$$

$$\text{Throughput} = \frac{F}{T_{\text{sequential}}} = \frac{r}{T_{\text{RTT}} + r/B}$$

### With Read-Ahead (Pipelining)

NFS clients pipeline read requests to hide latency:

$$\text{Throughput}_{\text{pipeline}} = \min\left(B, \frac{w \times r}{T_{\text{RTT}}}\right)$$

where $w$ is the pipeline window depth (number of outstanding requests).

| $T_{\text{RTT}}$ | rsize | Window ($w$) | Throughput | Max (1Gbps) |
|:---:|:---:|:---:|:---:|:---:|
| 0.2 ms | 1 MB | 4 | 20 GB/s | 125 MB/s |
| 20 ms | 1 MB | 4 | 200 MB/s | 12.5 MB/s |
| 20 ms | 1 MB | 16 | 800 MB/s | 12.5 MB/s |
| 80 ms | 1 MB | 16 | 200 MB/s | 12.5 MB/s |

On LAN, the bottleneck is bandwidth. On WAN, the bottleneck is RTT unless the window is large enough.

### Bandwidth-Delay Product

The optimal window size:

$$w_{\text{opt}} = \lceil \frac{B \times T_{\text{RTT}}}{r} \rceil$$

For 1 Gbps and 20ms RTT with 1 MB rsize:

$$w_{\text{opt}} = \lceil \frac{125\text{MB/s} \times 0.020\text{s}}{1\text{MB}} \rceil = \lceil 2.5 \rceil = 3$$

---

## 3. Cache Coherence -- Close-to-Open Consistency

### The CTO Protocol

NFS uses close-to-open (CTO) consistency: a client re-validates cached data when a file is opened:

$$\text{OPEN}(f) \implies \text{GETATTR}(f) \implies \begin{cases} \text{use cache} & \text{if } \text{mtime}_{\text{server}} = \text{mtime}_{\text{cached}} \\ \text{invalidate} & \text{if } \text{mtime}_{\text{server}} \neq \text{mtime}_{\text{cached}} \end{cases}$$

### Cache Hit Rate

Let $p_w$ be the probability that file $f$ was modified between close and next open:

$$P(\text{cache hit}) = 1 - p_w$$

For a file modified on average every $M$ seconds, accessed every $A$ seconds:

$$p_w = 1 - e^{-A/M}$$

| Access Interval $A$ | Modification Interval $M$ | $P(\text{cache hit})$ |
|:---:|:---:|:---:|
| 1s | 60s | 98.3% |
| 1s | 10s | 90.5% |
| 10s | 60s | 84.7% |
| 10s | 10s | 36.8% |
| 60s | 60s | 36.8% |

### Attribute Cache Timing

NFS clients cache file attributes for a bounded time:

$$T_{\text{cache}} \in [\text{acregmin}, \text{acregmax}] \quad \text{(default: [3s, 60s])}$$

The cache timeout adapts: files that change frequently get shorter cache times.

$$T_{\text{attr}} = \min\left(\text{acregmax}, \max\left(\text{acregmin}, \frac{t_{\text{now}} - t_{\text{mtime}}}{10}\right)\right)$$

The divisor of 10 means: cache the attribute for 1/10th of the file's age since last modification.

---

## 4. NFSv4 Delegations (State Optimization)

### Delegation Model

A delegation grants a client exclusive or shared access, eliminating server round-trips:

$$\text{Delegation}(f, C) = \begin{cases} \text{READ} & C \text{ can cache reads without revalidation} \\ \text{WRITE} & C \text{ can cache reads and writes, flush on recall} \end{cases}$$

### Delegation Benefit

Without delegation (every operation requires server round-trip):

$$T_{\text{no\_deleg}}(n) = n \times T_{\text{RTT}}$$

With delegation (operations are local, no server round-trip):

$$T_{\text{deleg}}(n) = T_{\text{RTT}} + n \times T_{\text{local}} + T_{\text{RTT}}$$

where the first RTT is delegation grant and the last is delegation return.

Speedup for $n$ operations:

$$\text{Speedup} = \frac{n \times T_{\text{RTT}}}{2 \times T_{\text{RTT}} + n \times T_{\text{local}}}$$

For 100 operations on LAN (RTT=0.2ms, local=0.001ms):

$$\text{Speedup} = \frac{100 \times 0.2}{2 \times 0.2 + 100 \times 0.001} = \frac{20}{0.5} = 40\times$$

### Delegation Recall Cost

When client $C_2$ accesses a delegated file held by $C_1$:

$$T_{\text{recall}} = T_{\text{RTT}}(S \to C_1) + T_{\text{flush}}(C_1) + T_{\text{RTT}}(C_1 \to S) + T_{\text{op}}(C_2)$$

This makes delegations expensive to recall -- they are most beneficial for files accessed by a single client.

### Delegation Efficiency

Let $p_c$ be the probability of contention (another client accessing the delegated file):

$$E[\text{benefit}] = (1 - p_c) \times \text{Speedup} - p_c \times \text{Recall Cost}$$

Delegations are net-positive when:

$$p_c < \frac{\text{Speedup}}{\text{Speedup} + \text{Recall Cost}}$$

---

## 5. pNFS Layout Mathematics (Parallel I/O)

### Striped Layout

pNFS (NFSv4.1) stripes files across $k$ data servers:

$$\text{Throughput}_{\text{pNFS}} = \min\left(k \times B_{\text{per\_server}}, B_{\text{client}}\right)$$

### Stripe Width Optimization

For a file of size $F$ striped across $k$ servers with stripe unit $s$:

$$T_{\text{read}}(F) = \frac{F}{k \times s} \times \left(T_{\text{RTT}} + \frac{s}{B}\right)$$

The optimal stripe unit maximizes throughput:

$$s_{\text{opt}} = B \times T_{\text{RTT}}$$

| Servers ($k$) | Per-Server BW | Aggregate BW | 1 GB Read Time |
|:---:|:---:|:---:|:---:|
| 1 | 125 MB/s | 125 MB/s | 8.0s |
| 4 | 125 MB/s | 500 MB/s | 2.0s |
| 8 | 125 MB/s | 1000 MB/s | 1.0s |
| 16 | 125 MB/s | 2000 MB/s | 0.5s |

### Layout Types

| Layout | Description | Use Case |
|:---|:---|:---|
| Files | Whole file on one server | Small files |
| Blocks | Block-level striping | SAN integration |
| Objects | Object-level striping | Object storage |
| Flex Files | File-level with mirrors | NAS clusters |

---

## 6. Write Semantics (sync vs async)

### Sync Write Cost

With `sync` exports, every write commits to stable storage before acknowledging:

$$T_{\text{sync\_write}} = T_{\text{RTT}} + T_{\text{journal}} + T_{\text{disk\_flush}}$$

$$T_{\text{journal}} \approx 1\text{-}5\text{ms (SSD)}, \quad 5\text{-}15\text{ms (HDD)}$$

### Async Write Cost

With `async` exports, writes are acknowledged before committing:

$$T_{\text{async\_write}} = T_{\text{RTT}} + T_{\text{memcpy}}$$

$$T_{\text{memcpy}} \approx 0.01\text{ms}$$

### Risk Window

The async risk window is the time between acknowledgment and disk commit:

$$T_{\text{risk}} = T_{\text{commit\_interval}} \approx 5\text{-}30\text{s}$$

$$P(\text{data loss}) = P(\text{crash in } T_{\text{risk}}) \approx \frac{T_{\text{risk}}}{\text{MTBF}}$$

For a server with 99.99% uptime (MTBF = 8760 hours):

$$P(\text{data loss per write}) = \frac{30\text{s}}{8760 \times 3600\text{s}} = 9.5 \times 10^{-7}$$

---

## 7. NFS vs Local I/O Comparison

### IOPS Comparison

| Operation | Local SSD | NFS LAN | NFS WAN | Ratio (LAN) |
|:---|:---:|:---:|:---:|:---:|
| Random read (4K) | 100,000 | 5,000 | 50 | 20x slower |
| Sequential read (1M) | 3,000 | 950 | 12 | 3x slower |
| Metadata (stat) | 500,000 | 5,000 | 50 | 100x slower |
| readdir (100 entries) | 200,000 | 3,000 | 40 | 67x slower |

### When NFS Makes Sense

NFS throughput approaches local performance when:

$$\frac{T_{\text{data}}}{T_{\text{RTT}}} \gg 1$$

This occurs with large sequential I/O on fast networks:

$$\text{I/O size} \gg B \times T_{\text{RTT}} = 125\text{MB/s} \times 0.2\text{ms} = 25\text{KB}$$

---

## Prerequisites

- Network latency models (RTT, bandwidth-delay product)
- Probability (cache hit rates, Poisson process for modifications)
- Queueing theory (pipeline depth, outstanding requests)
- State machines (delegation lifecycle, lease management)

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Single NFS read | $O(T_{\text{RTT}} + n/B)$ | $O(n)$ buffer |
| Sequential read (pipelined) | $O(F/\min(B, wR/T_{\text{RTT}}))$ | $O(w \times r)$ |
| Metadata operation | $O(T_{\text{RTT}})$ | $O(1)$ |
| Delegation grant | $O(T_{\text{RTT}})$ | $O(1)$ per file |
| Delegation recall | $O(2 \times T_{\text{RTT}} + T_{\text{flush}})$ | $O(\text{dirty data})$ |
| pNFS read ($k$ servers) | $O(F / (k \times B))$ | $O(k \times r)$ |

---

*NFS performance mathematics are dominated by a single variable: network round-trip time. Every operation that requires server contact pays the RTT tax. The entire evolution of NFS -- from v3's stateless protocol to v4's delegations to v4.1's parallel layouts -- is an engineering effort to minimize the number of round trips per unit of useful work. Caching and delegations are local optimizations (reduce trips per operation). pNFS is a parallel optimization (increase bandwidth per trip). The fundamental tradeoff is consistency vs performance: stronger consistency requires more round trips.*
