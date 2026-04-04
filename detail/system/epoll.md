# The Mathematics of epoll — I/O Multiplexing Scalability

> *epoll achieves O(1) event notification by maintaining kernel-side state. The math covers scalability analysis, ready-list amortization, thundering herd probability, and comparison with select/poll complexity.*

---

## 1. Scalability: select/poll vs epoll

### The Problem

With $n$ monitored file descriptors and $k$ ready fds per call, what is the per-call cost of each mechanism?

### The Formula

**select/poll** — copies the entire interest set to/from kernel on every call:

$$T_{\text{select}} = O(n) + O(k)$$

**epoll** — interest set lives in the kernel; only ready events are returned:

$$T_{\text{epoll\_wait}} = O(k)$$

$$T_{\text{epoll\_ctl}} = O(\log n)$$

For a server processing $R$ requests/sec with $n$ total connections:

$$\text{select cost/sec} = R \times O(n) = O(Rn)$$

$$\text{epoll cost/sec} = R \times O(k) + C \times O(\log n)$$

where $C$ is the rate of connection changes (add/remove) and $k$ is the number of ready fds per epoll_wait call.

### Worked Examples

| Connections ($n$) | Ready/call ($k$) | Changes/sec ($C$) | select cost | epoll cost | Speedup |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 100 | 10 | 50 | O(100) | O(10) | 10x |
| 10,000 | 10 | 100 | O(10,000) | O(10) | 1,000x |
| 100,000 | 50 | 500 | O(100,000) | O(50) | 2,000x |
| 1,000,000 | 100 | 1,000 | O(1,000,000) | O(100) | 10,000x |

---

## 2. Red-Black Tree (Interest Set)

### The Problem

epoll uses a red-black tree to store monitored fds. What are the costs of add, modify, delete, and lookup operations?

### The Formula

For a red-black tree with $n$ nodes:

$$T_{\text{insert}} = O(\log n)$$

$$T_{\text{delete}} = O(\log n)$$

$$T_{\text{lookup}} = O(\log n)$$

$$T_{\text{rebalance}} = O(1) \text{ amortized (at most 3 rotations)}$$

Memory per entry:

$$M_{\text{entry}} = \text{sizeof(struct epitem)} \approx 128 \text{ bytes}$$

Total memory for the interest set:

$$M_{\text{tree}} = n \times M_{\text{entry}}$$

### Worked Examples

| Connections ($n$) | Tree Depth ($\log_2 n$) | Memory (interest set) | epoll_ctl latency |
|:---:|:---:|:---:|:---:|
| 1,000 | 10 | 125 KB | ~1 us |
| 10,000 | 14 | 1.2 MB | ~1.5 us |
| 100,000 | 17 | 12.2 MB | ~2 us |
| 1,000,000 | 20 | 122 MB | ~2.5 us |

---

## 3. Ready List and Callback Cost

### The Problem

When an fd becomes ready, the kernel appends it to the ready list via a callback. epoll_wait drains this list. What is the cost model?

### The Formula

The callback is triggered by the device driver or socket layer:

$$T_{\text{callback}} = O(1) \text{ (linked list append)}$$

epoll_wait copies at most $\text{maxevents}$ entries to userspace:

$$T_{\text{wait}} = O(\min(k, \text{maxevents}))$$

For edge-triggered mode, callbacks fire only on state transitions:

$$\text{Callbacks/sec (ET)} = \text{Events/sec}$$

For level-triggered mode, callbacks fire on every epoll_wait while condition holds:

$$\text{Callbacks/sec (LT)} = \text{Events/sec} \times \bar{n}_{\text{polls}}$$

where $\bar{n}_{\text{polls}}$ is the average number of epoll_wait calls before the condition is cleared.

### Worked Examples

| Mode | Events/sec | Avg polls before drain | Callbacks/sec | Overhead vs ET |
|:---:|:---:|:---:|:---:|:---:|
| Edge-triggered | 50,000 | 1 | 50,000 | 1x |
| Level-triggered | 50,000 | 1 | 50,000 | 1x |
| Level-triggered | 50,000 | 3 | 150,000 | 3x |
| Level-triggered | 50,000 | 5 | 250,000 | 5x |

In practice, the LT overhead is small unless the application frequently calls epoll_wait without draining data.

---

## 4. Thundering Herd Analysis

### The Problem

When $T$ threads share one epoll instance and a new connection arrives, how many threads wake up?

### The Formula

**Without EPOLLEXCLUSIVE:**

$$W = T \quad (\text{all threads wake})$$

Wasted wakeups:

$$W_{\text{wasted}} = T - 1$$

Wasted CPU:

$$\text{CPU}_{\text{wasted}} = (T-1) \times T_{\text{wakeup}}$$

**With EPOLLEXCLUSIVE (Linux 4.5+):**

$$W = 1 \quad (\text{at most one thread wakes})$$

**With SO_REUSEPORT:**

$$W = 1 \quad (\text{kernel distributes to one socket/thread})$$

### Worked Examples

| Threads ($T$) | Connections/sec | Without EXCLUSIVE (wakeups/sec) | With EXCLUSIVE | Reduction |
|:---:|:---:|:---:|:---:|:---:|
| 4 | 10,000 | 40,000 | 10,000 | 75% |
| 8 | 10,000 | 80,000 | 10,000 | 87.5% |
| 16 | 10,000 | 160,000 | 10,000 | 93.75% |
| 32 | 50,000 | 1,600,000 | 50,000 | 96.88% |

---

## 5. max_user_watches Sizing

### The Problem

The kernel limits the total number of epoll watches per user via `/proc/sys/fs/epoll/max_user_watches`. Each watch consumes kernel memory.

### The Formula

$$M_{\text{total}} = W_{\text{max}} \times M_{\text{watch}}$$

where $M_{\text{watch}} \approx 160$ bytes on 64-bit systems (including the epitem struct and socket callbacks).

Default value (calculated from available memory):

$$W_{\text{default}} = \frac{\text{Low Memory (bytes)}}{160 \times \frac{1}{25}} = \frac{\text{Low Memory} \times 25}{160}$$

On a system with 4% of RAM as low memory:

$$W_{\text{default}} \approx \frac{0.04 \times \text{RAM}}{6.4}$$

### Worked Examples

| RAM | Low Memory (est.) | Default max_user_watches | Memory Used at Max |
|:---:|:---:|:---:|:---:|
| 4 GB | 160 MB | ~1,000,000 | 153 MB |
| 16 GB | 640 MB | ~4,000,000 | 610 MB |
| 64 GB | 2.56 GB | ~16,000,000 | 2.4 GB |
| 128 GB | 5.12 GB | ~32,000,000 | 4.9 GB |

---

## 6. io_uring vs epoll: Syscall Amortization

### The Problem

epoll requires at least one syscall per batch of events. io_uring amortizes syscalls via shared ring buffers. When does io_uring outperform epoll?

### The Formula

epoll syscalls per $R$ I/O operations:

$$S_{\text{epoll}} = R + W_{\text{calls}}$$

where $W_{\text{calls}}$ is the number of epoll_wait calls. Each I/O also requires a read/write syscall.

io_uring syscalls per $R$ I/O operations (with SQ polling):

$$S_{\text{io\_uring}} = \left\lceil \frac{R}{B} \right\rceil$$

where $B$ is the submission queue depth (batch size). With kernel-side SQ polling:

$$S_{\text{io\_uring\_sqpoll}} \approx 0 \quad (\text{kernel thread polls the ring})$$

### Worked Examples

| I/O ops/sec ($R$) | epoll syscalls/sec | io_uring (batch 32) | io_uring (SQ poll) |
|:---:|:---:|:---:|:---:|
| 10,000 | ~20,000 | 313 | ~0 |
| 100,000 | ~200,000 | 3,125 | ~0 |
| 1,000,000 | ~2,000,000 | 31,250 | ~0 |

At high I/O rates, the syscall overhead of epoll becomes the bottleneck, and io_uring's ring buffer model wins decisively.

---

## Prerequisites

- Operating system I/O models (blocking, non-blocking, multiplexed, async)
- File descriptor semantics (open, close, dup, inheritance across fork/exec)
- Linux kernel data structures (red-black trees, linked lists, wait queues)
- Socket programming (TCP state machine, accept, read, write, shutdown)
- Concurrency primitives (mutexes, condition variables, atomic operations)
- Systems performance (context switch cost, syscall overhead, cache effects)
