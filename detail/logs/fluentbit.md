# The Mathematics of Fluent Bit — Pipeline Throughput, Buffering, and Backpressure

> *Fluent Bit processes millions of log records through a plugin pipeline. The math covers throughput bounds, memory and disk buffer sizing, regex parsing cost, backpressure dynamics, and chunk management for reliable log delivery.*

---

## 1. Pipeline Throughput (Queueing Theory)

### Single-Pipeline Model

The pipeline processes records sequentially through stages:

$$T_{\text{record}} = T_{\text{input}} + T_{\text{parse}} + \sum_{i=1}^{f} T_{\text{filter}_i} + T_{\text{output}}$$

$$\lambda_{\text{max}} = \frac{1}{T_{\text{record}}}$$

### Stage Latency Breakdown

| Stage | Typical Latency | Bottleneck |
|:---|:---:|:---|
| Tail read | 1-5 us/record | Disk I/O |
| JSON parse | 2-10 us/record | CPU |
| Regex parse | 5-50 us/record | CPU (regex complexity) |
| Grep filter | 1-5 us/record | CPU |
| Modify filter | 0.5-2 us/record | CPU |
| Kubernetes filter | 10-50 us/record | API cache lookup |
| Lua filter | 5-100 us/record | Lua interpreter |
| ES output (batched) | 0.1-1 us/record | Network (amortized) |

### Throughput Examples

| Pipeline Stages | Per-Record Cost | Max Records/sec |
|:---:|:---:|:---:|
| tail + json + stdout | 5 us | 200,000 |
| tail + regex + grep + ES | 30 us | 33,333 |
| tail + regex + k8s + lua + ES | 100 us | 10,000 |
| tail + json + modify + loki | 15 us | 66,667 |

---

## 2. Memory Buffer Sizing (Capacity)

### Chunk Model

Fluent Bit stores records in chunks. Each chunk holds records up to a size limit:

$$C_{\text{size}} = 2 \text{ MiB (default)}$$

$$\text{Chunks in memory} = \left\lceil \frac{\lambda \times T_{\text{flush}} \times R_{\text{avg}}}{C_{\text{size}}} \right\rceil$$

where $R_{\text{avg}}$ = average record size, $T_{\text{flush}}$ = flush interval.

### Memory Usage

$$M_{\text{total}} = N_{\text{chunks}} \times C_{\text{size}} + M_{\text{overhead}}$$

| Records/sec | Avg Record | Flush Interval | Chunks | Memory |
|:---:|:---:|:---:|:---:|:---:|
| 1,000 | 500 B | 5s | 2 | 4 MiB |
| 10,000 | 500 B | 5s | 13 | 26 MiB |
| 10,000 | 2 KB | 5s | 49 | 98 MiB |
| 100,000 | 500 B | 5s | 122 | 244 MiB |

### Mem_Buf_Limit Effect

When memory reaches `Mem_Buf_Limit`, the input pauses:

$$T_{\text{pause}} = \frac{M_{\text{buf\_limit}}}{\lambda_{\text{in}} \times R_{\text{avg}}} - \frac{M_{\text{buf\_limit}}}{\lambda_{\text{out}} \times R_{\text{avg}}}$$

If $\lambda_{\text{out}} = 0$ (output down):

$$T_{\text{until\_pause}} = \frac{M_{\text{buf\_limit}}}{\lambda_{\text{in}} \times R_{\text{avg}}}$$

| Mem_Buf_Limit | Records/sec | Avg Record | Time Until Pause |
|:---:|:---:|:---:|:---:|
| 10 MB | 1,000 | 500 B | 20 sec |
| 10 MB | 10,000 | 500 B | 2 sec |
| 50 MB | 10,000 | 500 B | 10 sec |
| 100 MB | 10,000 | 1 KB | 10 sec |

---

## 3. Filesystem Buffer Sizing (Disk)

### Disk Buffer Capacity

$$D_{\text{buffer}} = \lambda \times R_{\text{avg}} \times T_{\text{outage}}$$

| Records/sec | Avg Record | Outage Duration | Disk Required |
|:---:|:---:|:---:|:---:|
| 1,000 | 500 B | 1 hour | 1.8 GB |
| 10,000 | 500 B | 1 hour | 18 GB |
| 10,000 | 1 KB | 4 hours | 144 GB |
| 100,000 | 500 B | 30 min | 3 GB |

### Drain Time After Outage

$$T_{\text{drain}} = \frac{D_{\text{buffered}}}{\lambda_{\text{out}} - \lambda_{\text{in}}}$$

If output can process 2x the input rate:

$$T_{\text{drain}} = \frac{D_{\text{buffered}}}{\lambda_{\text{in}}} = T_{\text{outage}}$$

---

## 4. Regex Parsing Complexity (Automata Theory)

### NFA vs DFA

Fluent Bit uses PCRE (backtracking NFA):

$$T_{\text{NFA}} = O(2^m) \quad \text{worst case (catastrophic backtracking)}$$

$$T_{\text{DFA}} = O(n) \quad \text{guaranteed linear}$$

where $m$ = pattern length, $n$ = input length.

### Practical Pattern Costs

| Pattern Type | Example | Cost/Record |
|:---|:---|:---:|
| Literal match | `error` | O(n) |
| Simple groups | `^(\S+) (\S+)` | O(n) |
| Alternation | `(error\|warn\|info)` | O(n * k) |
| Greedy quantifiers | `(.*)message(.*)` | O(n^2) |
| Nested quantifiers | `(a+)+b` | O(2^n) |

### Named Capture Groups

Each named group adds extraction overhead:

$$T_{\text{extract}} = T_{\text{match}} + k \times T_{\text{copy}}$$

where $k$ = number of capture groups.

---

## 5. Output Batching (Throughput Optimization)

### Batch Efficiency

$$\text{Effective throughput} = \frac{B_{\text{size}}}{T_{\text{batch\_send}}}$$

$$T_{\text{batch\_send}} = T_{\text{serialize}} + T_{\text{network}} + T_{\text{ack}}$$

### Network Overhead per Record

$$O_{\text{per\_record}} = \frac{H_{\text{request}}}{B_{\text{size}}} + H_{\text{record}}$$

| Batch Size | Request Header | Per-Record Overhead | Records/sec (10ms RTT) |
|:---:|:---:|:---:|:---:|
| 1 | 200 B | 200 B | 100 |
| 100 | 200 B | 2 B | 10,000 |
| 1,000 | 200 B | 0.2 B | 100,000 |
| 10,000 | 200 B | 0.02 B | 1,000,000 |

### Flush Interval vs Latency

$$\text{Max delivery latency} = T_{\text{flush}} + T_{\text{batch\_send}} + T_{\text{retry}}$$

| Flush Interval | Send Time | Max Latency (no retry) |
|:---:|:---:|:---:|
| 1s | 50ms | 1.05s |
| 5s | 50ms | 5.05s |
| 10s | 100ms | 10.1s |

---

## 6. Tail Input File Tracking (State Management)

### Offset Database

The SQLite DB file tracks read position per file:

$$\text{DB entries} = F_{\text{active}} + F_{\text{rotated}}$$

$$\text{DB size} \approx N_{\text{files}} \times 200 \text{ bytes}$$

### File Rotation Detection

Fluent Bit detects rotation via inode tracking:

$$\text{rotation detected} \iff \text{inode}(path) \neq \text{inode}_{\text{stored}}$$

### Catch-Up Rate After Restart

$$T_{\text{catchup}} = \frac{\sum_{f} (S_f - O_f)}{R_{\text{read}}}$$

where $S_f$ = current file size, $O_f$ = stored offset, $R_{\text{read}}$ = read throughput.

| Files | Avg Backlog | Read Speed | Catch-Up Time |
|:---:|:---:|:---:|:---:|
| 10 | 10 MB each | 100 MB/s | 1 sec |
| 100 | 100 MB each | 100 MB/s | 100 sec |
| 100 | 1 GB each | 200 MB/s | 500 sec |

---

## Prerequisites

queueing-theory, automata-theory, information-theory, rsyslog

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| JSON parse | O(n) input length | O(k) fields |
| Regex parse (simple) | O(n) | O(g) groups |
| Regex parse (backtrack) | O(2^m) worst | O(m) stack |
| Grep filter | O(n) per record | O(1) |
| Modify filter | O(1) per field | O(1) |
| Kubernetes filter | O(1) cache hit | O(p) pods cached |
| Chunk flush | O(c) chunk size | O(c) |
| Offset DB lookup | O(log f) files | O(f) entries |
