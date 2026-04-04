# The Mathematics of Ulimit -- File Descriptor Sizing, Process Limits, and Stack Growth

> *Resource limits define the hard boundaries of process capability through integer*
> *thresholds enforced at the kernel level. Each limit translates to a capacity planning*
> *problem: how many file descriptors, threads, or bytes can a workload safely consume?*

---

## 1. File Descriptor Table Sizing (Capacity Planning)

### The Problem

A server handling $C$ concurrent connections needs at least $C$ file descriptors, plus overhead for standard I/O, log files, and internal sockets. What should `nofile` be set to?

### The Formula

$$\text{FD}_{\text{required}} = \text{FD}_{\text{base}} + \text{FD}_{\text{connections}} + \text{FD}_{\text{internal}} + \text{FD}_{\text{margin}}$$

Where:
- $\text{FD}_{\text{base}} = 3$ (stdin, stdout, stderr)
- $\text{FD}_{\text{connections}} = C$ (one per client socket)
- $\text{FD}_{\text{internal}}$ = log files + config files + IPC sockets + timers + epoll FDs
- $\text{FD}_{\text{margin}} = 0.2 \times \text{FD}_{\text{total}}$ (20% safety margin)

### Worked Examples

**Nginx reverse proxy** serving 10,000 concurrent clients:

| Component | FDs | Calculation |
|-----------|-----|-------------|
| Base (stdin/out/err) | 3 | Fixed |
| Client connections | 10,000 | 1 per client |
| Upstream connections | 10,000 | 1 per proxied request |
| Log files | 4 | access + error + 2 vhosts |
| Listen sockets | 2 | HTTP + HTTPS |
| Epoll instances | 1 | Per worker |
| Timer FDs | 1 | Internal |
| **Subtotal** | **20,011** | |
| Safety margin (20%) | 4,002 | |
| **Total required** | **24,013** | |
| **Recommended nofile** | **32,768** | Round up to power of 2 |

**PostgreSQL database** with 500 connections:

| Component | FDs | Calculation |
|-----------|-----|-------------|
| Base | 3 | Fixed |
| Client connections | 500 | 1 per connection |
| Data files | 2,000 | ~4 per table (heap + 3 indexes) |
| WAL segments | 10 | Active WAL files |
| Temp files | 50 | Sort/hash operations |
| Shared libraries | 30 | .so files |
| **Subtotal** | **2,593** | |
| Safety margin (20%) | 519 | |
| **Total required** | **3,112** | |
| **Recommended nofile** | **4,096** | |

## 2. File Descriptor Memory Cost (Memory Accounting)

### The Problem

Each open file descriptor consumes kernel memory. What is the total memory cost for high FD counts?

### The Formula

Per-FD kernel memory:

$$\text{mem}_{\text{per\_fd}} = \text{sizeof(struct file)} + \text{sizeof(fd\_entry)} \approx 256 + 8 = 264 \text{ bytes}$$

For socket FDs, add socket buffer memory:

$$\text{mem}_{\text{socket\_fd}} = \text{mem}_{\text{per\_fd}} + \text{rcvbuf} + \text{sndbuf}$$

Total memory for $n$ FDs:

$$\text{mem}_{\text{total}} = n \times \text{mem}_{\text{per\_fd}}$$

### Worked Examples

| Open FDs | FD Table Memory | With Socket Buffers (128KB each) |
|---------|----------------|----------------------------------|
| 1,024 | 264 KB | 264 KB + 256 MB |
| 65,536 | 16.5 MB | 16.5 MB + 16 GB |
| 1,048,576 | 264 MB | 264 MB + 256 TB (impractical) |

This is why high-connection servers use small socket buffers:

$$\text{mem}_{\text{1M\_conns}} = 1{,}048{,}576 \times (264 + 4{,}096 + 4{,}096) = 8.4 \text{ GB}$$

With 4KB send + 4KB receive buffers per socket.

## 3. Process/Thread Limit Calculations (nproc)

### The Problem

The `nproc` limit caps threads per user. A multi-threaded application needs to calculate total thread consumption including runtime overhead.

### The Formula

$$\text{threads}_{\text{total}} = \text{workers} + \text{runtime\_threads} + \text{overhead}$$

For Go:

$$\text{threads}_{\text{go}} = \max(\text{GOMAXPROCS}, \text{goroutines\_in\_syscall}) + \text{GC\_threads} + \text{sysmon}$$

For Java:

$$\text{threads}_{\text{java}} = \text{app\_threads} + \text{GC\_threads} + \text{JIT\_threads} + \text{internal}$$

### Worked Examples

**Go HTTP server** with 50,000 goroutines:

| Component | Threads | Notes |
|-----------|---------|-------|
| GOMAXPROCS | 8 | CPU-bound goroutine runners |
| Goroutines in syscall | ~100 | Each gets an OS thread |
| GC threads | 8 | Matches GOMAXPROCS |
| sysmon + others | 3 | Runtime overhead |
| **Total OS threads** | **~119** | |

Go is efficient: 50,000 goroutines on ~119 OS threads.

**Java application server** with 200 request handler threads:

| Component | Threads | Notes |
|-----------|---------|-------|
| Request handlers | 200 | Thread-per-request model |
| G1 GC threads | 8 | ParallelGCThreads |
| JIT compiler threads | 4 | C1 + C2 compilers |
| Finalizer thread | 1 | Reference processing |
| Signal dispatcher | 1 | JVM internal |
| Common fork-join pool | 8 | Parallel streams |
| Timer/scheduler | 2 | ScheduledExecutorService |
| **Total OS threads** | **~224** | |

**Recommended nproc settings:**

| Workload | Formula | Example Value |
|----------|---------|---------------|
| Single Go service | threads x 2 | 256 |
| Java app server | threads x 3 | 1,024 |
| Multi-service host | sum(services) x 2 | 4,096 |
| Container host | per-user max | 63,432 |

## 4. Stack Size and Recursion Depth (Stack Growth)

### The Problem

Given a stack size limit, how deep can recursion go before hitting a stack overflow?

### The Formula

$$\text{max\_depth} = \frac{\text{stack\_limit} - \text{stack\_base}}{\text{frame\_size}}$$

Stack frame size per function call:

$$\text{frame\_size} = \text{return\_addr} + \text{saved\_regs} + \text{local\_vars} + \text{alignment\_padding}$$

### Worked Examples

Typical function with 4 local `int64` variables on x86_64:

$$\text{frame\_size} = 8 + 48 + 32 + 8 = 96 \text{ bytes}$$

| Stack Limit | Base Overhead | Available | Frame Size | Max Recursion Depth |
|------------|--------------|-----------|-----------|-------------------|
| 8 MB | 64 KB | 8,128 KB | 96 B | 86,700 |
| 8 MB | 64 KB | 8,128 KB | 512 B | 16,250 |
| 8 MB | 64 KB | 8,128 KB | 4 KB | 2,032 |
| 16 MB | 64 KB | 16,320 KB | 96 B | 174,080 |
| 1 MB | 64 KB | 960 KB | 96 B | 10,240 |

Go goroutine stacks start at 8 KB and grow dynamically up to 1 GB:

$$\text{max\_depth}_{\text{go}} = \frac{1 \text{ GB}}{96 \text{ B}} \approx 11{,}184{,}810$$

## 5. Memory Lock Accounting (memlock)

### The Problem

Applications using `mlock()`, `mmap(MAP_LOCKED)`, or huge pages must account for locked memory against the `memlock` limit.

### The Formula

$$\text{locked}_{\text{total}} = \text{mlock\_pages} \times \text{page\_size} + \text{hugetlb\_pages} \times \text{hugepage\_size}$$

$$\text{mlock\_check}: \text{locked}_{\text{total}} \leq \text{RLIMIT\_MEMLOCK}$$

### Worked Examples

**eBPF application** with maps:

| Resource | Size | Locked Memory |
|----------|------|---------------|
| BPF map (hash, 10K entries) | 640 KB | 640 KB |
| BPF map (array, 1M entries) | 8 MB | 8 MB |
| BPF program (loaded) | 32 KB | 32 KB |
| Ring buffer | 256 KB | 256 KB |
| **Total** | | **8.9 MB** |

Default `memlock` of 64 KB would fail. Required: `memlock=unlimited` or at least 16 MB.

**DPDK application** with huge pages:

| Resource | Count | Size | Locked Memory |
|----------|-------|------|---------------|
| 2MB huge pages | 1,024 | 2 MB each | 2 GB |
| 1GB huge pages | 4 | 1 GB each | 4 GB |
| **Total** | | | **6 GB** |

## 6. CPU Time Limit (cpu)

### The Problem

The `cpu` ulimit sets maximum CPU seconds a process can consume. How does this interact with multi-threaded execution?

### The Formula

$$\text{CPU}_{\text{consumed}} = \sum_{t=1}^{T} \text{cpu\_time}(t)$$

where $T$ is the number of threads. CPU time is aggregate across all threads.

Time to hit limit with $n$ busy threads:

$$\text{wall\_time}_{\text{to\_limit}} = \frac{\text{RLIMIT\_CPU}}{n}$$

### Worked Examples

CPU limit of 3600 seconds (1 hour of CPU time):

| Threads | CPU Usage | Wall Time to Limit |
|---------|-----------|-------------------|
| 1 | 100% each | 60 min |
| 4 | 100% each | 15 min |
| 8 | 100% each | 7.5 min |
| 4 | 50% each | 30 min |
| 16 | 100% each | 3.75 min |

When `RLIMIT_CPU` soft limit is reached: `SIGXCPU` sent.
When hard limit is reached: `SIGKILL` sent.

## 7. Limit Inheritance and Interaction (Hierarchy)

### The Problem

Limits propagate through fork/exec and interact with cgroup limits. Which limit applies?

### The Formula

The effective limit for a process is:

$$\text{effective} = \min(\text{ulimit}, \text{cgroup\_limit}, \text{sysctl\_limit})$$

Inheritance on fork:

$$\text{child.rlimit} = \text{parent.rlimit} \quad \text{(copied at fork time)}$$

### Worked Examples

| Level | nofile | nproc | memlock |
|-------|--------|-------|---------|
| sysctl (fs.file-max) | 9.2 x 10^18 | -- | -- |
| sysctl (kernel.pid_max) | -- | 4,194,304 | -- |
| systemd DefaultLimit | 65,536 | 63,432 | 8 MB |
| Service LimitNOFILE | 1,048,576 | -- | unlimited |
| Container ulimit | 524,288 | 4,096 | unlimited |
| **Effective** | **524,288** | **4,096** | **unlimited** |

## Prerequisites

operating-systems, memory-management, process-model, linux-kernel, capacity-planning

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| ulimit query (getrlimit) | O(1) | O(1) |
| ulimit set (setrlimit) | O(1) | O(1) |
| FD allocation (open) | O(1) amortized | O(1) per FD |
| FD table resize | O(n) copy | O(n) new table |
| Limit check on fork | O(1) | O(1) |
| prlimit on remote process | O(1) | O(1) |
| /proc/PID/limits read | O(limits) ~16 | O(1) |
| mlock page accounting | O(pages) | O(1) per page |
