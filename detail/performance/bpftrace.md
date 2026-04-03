# The Mathematics of bpftrace — Probe Overhead, Histogram Algebra & Aggregation Maps

> *bpftrace is a high-level language that compiles to eBPF programs. Its power lies in aggregation primitives — histograms, quantiles, and counts — computed in kernel space with O(1) per-event cost and O(N) reporting cost.*

---

## 1. Probe Types and Overhead

### Probe Mechanisms

| Probe Type | Mechanism | Overhead per Hit | Max Rate |
|:---|:---|:---:|:---:|
| kprobe | INT3 breakpoint | 50-100 ns | ~10M/s |
| kretprobe | Trampoline | 100-200 ns | ~5M/s |
| uprobe | INT3 in userspace | 1-5 us | ~1M/s |
| uretprobe | Trampoline | 2-10 us | ~500K/s |
| tracepoint | Static, no-op when disabled | 20-50 ns | ~20M/s |
| USDT | Static, NOP-sled when disabled | 50-200 ns | ~5M/s |
| profile | Timer interrupt | ~100 ns | Per-CPU |
| interval | Timer (one CPU) | ~100 ns | Per-timer |

### Total Overhead Model

$$overhead = event\_rate \times T_{probe} \times n_{CPUs\_affected}$$

**Example:** kprobe on `tcp_sendmsg`, 100K calls/s:

$$overhead = 100000 \times 75ns = 7.5ms/s = 0.75\%$$

### Overhead Budget

$$\%overhead = \frac{T_{probe\_total}}{T_{wall}} \times 100$$

| Overhead | Acceptable? | Use Case |
|:---:|:---|:---|
| < 1% | Production safe | Long-running monitoring |
| 1-5% | Development/staging | Debugging sessions |
| 5-20% | Careful in production | Short targeted traces |
| > 20% | Avoid in production | Benchmark only |

---

## 2. Histograms — Log-Linear Bucketing

### Power-of-2 Histogram (@hist)

bpftrace's `hist()` uses power-of-2 buckets:

$$bucket(x) = \begin{cases} 0 & \text{if } x = 0 \\ \lfloor \log_2(x) \rfloor & \text{otherwise} \end{cases}$$

Bucket ranges: $[2^k, 2^{k+1})$ for each $k$.

| Bucket | Range | Width |
|:---:|:---|:---:|
| 0 | [1, 2) | 1 |
| 1 | [2, 4) | 2 |
| 2 | [4, 8) | 4 |
| 3 | [8, 16) | 8 |
| ... | ... | ... |
| 20 | [1M, 2M) | 1M |

### Resolution and Error

The maximum relative error per bucket:

$$error = \frac{bucket\_width}{bucket\_start} = \frac{2^k}{2^k} = 100\%$$

This is coarse — a value of 5 and 7 both fall in the [4,8) bucket. For I/O latency analysis this is usually sufficient (orders of magnitude matter more than precision).

### Space Complexity

$$buckets = \lceil \log_2(max\_value) \rceil + 1$$

For nanosecond latencies up to 10 seconds ($10^{10}$ ns):

$$buckets = \lceil \log_2(10^{10}) \rceil + 1 = 34$$

$$memory = 34 \times 8 \text{ bytes} = 272 \text{ bytes per histogram}$$

### Per-CPU Histograms

bpftrace uses per-CPU maps to avoid contention:

$$memory_{total} = buckets \times 8 \times n_{CPUs} \times n\_keys$$

---

## 3. Linear Histogram (@lhist)

### Bucket Structure

$$bucket(x) = \lfloor \frac{x - min}{step} \rfloor$$

$$n\_buckets = \frac{max - min}{step} + 2 \text{ (plus overflow and underflow)}$$

### Resolution vs Space Tradeoff

**Example:** Latency histogram, 0-1000 us, step 10 us:

$$n\_buckets = \frac{1000}{10} + 2 = 102$$

$$memory = 102 \times 8 = 816 \text{ bytes}$$

With step 1 us:

$$n\_buckets = 1002, \quad memory = 8016 \text{ bytes}$$

### When to Use lhist vs hist

| Scenario | Best Choice | Why |
|:---|:---|:---|
| I/O latency (us to s range) | `hist()` | Log scale spans orders of magnitude |
| Packet size (64-1500 bytes) | `lhist(0, 1600, 100)` | Linear range, uniform resolution |
| Temperature (20-100 C) | `lhist(20, 100, 1)` | Narrow linear range |
| Queue depth (0-128) | `lhist(0, 128, 1)` | Small integer range |

---

## 4. Aggregation Functions — In-Kernel Statistics

### Available Aggregations

| Function | Operation | Update Cost | Report Cost | Space |
|:---|:---|:---:|:---:|:---:|
| `count()` | $n += 1$ | $O(1)$ | $O(1)$ | 8 bytes |
| `sum(x)` | $s += x$ | $O(1)$ | $O(1)$ | 8 bytes |
| `avg(x)` | $s += x; n += 1$ | $O(1)$ | $O(1)$ | 16 bytes |
| `min(x)` | $m = \min(m, x)$ | $O(1)$ | $O(1)$ | 8 bytes |
| `max(x)` | $m = \max(m, x)$ | $O(1)$ | $O(1)$ | 8 bytes |
| `hist(x)` | Bucket update | $O(1)$ | $O(B)$ | $B \times 8$ bytes |
| `lhist(x)` | Bucket update | $O(1)$ | $O(B)$ | $B \times 8$ bytes |

**Critical property:** All aggregations are $O(1)$ per event. The cost of running bpftrace is independent of the aggregation complexity — only the event rate matters.

### Keyed Aggregation Maps

`@latency[comm] = hist(nsecs - @start[tid])` creates a histogram per process name:

$$memory = n_{keys} \times (key\_size + buckets \times 8 \times n_{CPUs})$$

**Example:** 100 unique process names, 34-bucket histogram, 16 CPUs:

$$memory = 100 \times (64 + 34 \times 8 \times 16) = 100 \times 4,416 = 441.6 KB$$

---

## 5. Timing Probes — Latency Measurement

### Delta Pattern

```
kprobe:func { @start[tid] = nsecs; }
kretprobe:func { @latency = hist(nsecs - @start[tid]); delete(@start[tid]); }
```

### Measurement Accuracy

$$measured\_latency = T_{func} + T_{probe\_entry} + T_{probe\_exit}$$

$$error = T_{probe\_entry} + T_{probe\_exit} \approx 100-300 ns$$

For latencies $> 10\mu s$: error is $< 3\%$. For latencies $< 1\mu s$: error can be $> 30\%$ (probe overhead dominates).

### Timestamp Resolution

`nsecs` reads `bpf_ktime_get_ns()` — nanosecond resolution from the monotonic clock:

$$resolution = 1 ns$$

But actual accuracy limited by clock source (~1-10 ns jitter with TSC).

---

## 6. Filtering — Predicate Cost

### Filter Overhead

`/condition/` filters in bpftrace add per-event cost:

$$T_{filtered} = T_{probe} + T_{condition\_eval}$$

### Short-Circuit Evaluation

Multiple conditions: `/pid == 1234 && comm == "nginx"/`

$$T_{condition} = T_{first} + P(first\_true) \times T_{second}$$

If `pid == 1234` rejects 99.9% of events:

$$T_{avg} = 5ns + 0.001 \times 20ns = 5.02ns$$

Almost all overhead eliminated by early filter.

### Selectivity Impact

$$effective\_overhead = event\_rate \times (T_{probe} + T_{filter}) + match\_rate \times T_{action}$$

**Example:** Tracing `read()` (100K/s), filtering to PID 1234 (100/s):

$$overhead = 100000 \times 75ns + 100 \times 200ns = 7.5ms + 0.02ms = 7.52ms/s$$

The filter doesn't save probe entry cost, but saves action cost for 99.9% of events.

---

## 7. Stack Traces — Storage and Cost

### Stack Trace Collection

`kstack` or `ustack` collect stack traces:

$$T_{kstack} = 1-5\mu s \text{ (kernel stack walk)}$$

$$T_{ustack} = 5-50\mu s \text{ (userspace stack walk, DWARF unwinding)}$$

### Stack Trace Deduplication

bpftrace stores unique stacks in a stack map:

$$memory = n_{unique\_stacks} \times depth \times 8 \times 2 \text{ (hash + addresses)}$$

Default max depth: 127 frames. Typical unique stacks: 100-10,000.

**Example:** 1000 unique stacks, average depth 20:

$$memory = 1000 \times 20 \times 8 \times 2 = 320 KB$$

### Flame Graph Generation

bpftrace output can be folded for flame graphs:

$$data\_points = \sum_{stack} count(stack)$$

$$flame\_width(frame) = \frac{\sum count\_with\_frame}{total\_samples}$$

---

## 8. Summary of bpftrace Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Probe overhead | $event\_rate \times T_{probe}$ | Linear |
| Log histogram buckets | $\lceil \log_2(max) \rceil + 1$ | Logarithmic |
| Histogram error | 100% per bucket (power-of-2) | Resolution limit |
| Aggregation update | $O(1)$ per event | Constant |
| Keyed map memory | $n_{keys} \times (key + buckets \times 8 \times CPUs)$ | Linear |
| Measurement error | $T_{probe\_in} + T_{probe\_out}$ | Fixed offset |
| Filter selectivity | $rate \times T_{probe} + match \times T_{action}$ | Selective overhead |
| Stack memory | $n_{stacks} \times depth \times 16$ | Linear |

---

*bpftrace is a domain-specific language for kernel instrumentation. Its genius is that every aggregation — count, sum, histogram — runs in O(1) per event inside the kernel, and only the final report crosses the kernel-user boundary. This is what makes million-event-per-second tracing practical.*
