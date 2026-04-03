# The Mathematics of strace — ptrace Overhead, Syscall Costs & seccomp-BPF

> *strace makes the invisible visible — but at a cost. Every system call now requires two context switches through ptrace, and understanding that overhead is the key to using strace without destroying the thing you're measuring.*

---

## 1. ptrace Mechanics — The Double-Stop Model

### How strace Works

strace uses the `ptrace(2)` system call to intercept every syscall of the traced process:

1. Tracee enters syscall → kernel stops tracee, notifies strace
2. strace reads syscall number and arguments → resumes tracee
3. Tracee completes syscall → kernel stops tracee again
4. strace reads return value → resumes tracee

Each syscall requires **2 stops** and **4 context switches** (tracee→kernel→strace, strace→kernel→tracee, repeat).

### Context Switch Cost

$$T_{ptrace\_overhead} = 4 \times T_{context\_switch} + 2 \times T_{waitpid} + T_{read\_regs}$$

| Component | Typical Cost | Notes |
|:---|:---:|:---|
| Context switch | 1-5 us | Depends on cache state |
| waitpid() notification | 1-3 us | Signal delivery |
| PTRACE_GETREGS | 0.5-1 us | Read register file |
| PTRACE_PEEKDATA | 0.5-1 us per word | Read tracee memory |
| **Total per syscall** | **8-30 us** | Added to every syscall |

### Slowdown Factor

$$slowdown = \frac{T_{syscall} + T_{ptrace\_overhead}}{T_{syscall}}$$

| Syscall | Native Cost | With strace | Slowdown |
|:---|:---:|:---:|:---:|
| getpid() | 0.1 us | 15 us | 150x |
| read(4KB) | 0.5 us | 15.5 us | 31x |
| write(4KB) | 1 us | 16 us | 16x |
| read(1MB, disk) | 500 us | 515 us | 1.03x |
| connect() | 100 us | 115 us | 1.15x |

**Key insight:** strace's overhead is **constant per syscall**. Fast syscalls (getpid, gettimeofday) are devastated. Slow syscalls (disk I/O, network) are barely affected.

---

## 2. Syscall Rate and Total Overhead

### Application Classification

$$total\_overhead = syscall\_rate \times T_{ptrace\_overhead}$$

| Application Type | Syscall Rate | Overhead at 15 us/call | Impact |
|:---|:---:|:---:|:---|
| CPU-bound (compression) | 100/s | 1.5 ms/s | Negligible (0.15%) |
| I/O-moderate (web server) | 10,000/s | 150 ms/s | Significant (15%) |
| I/O-heavy (database) | 100,000/s | 1.5 s/s | Catastrophic (150%) |
| Extreme (Redis) | 500,000/s | 7.5 s/s | Unusable |

### The Heisenberg Problem

strace changes timing enough to mask or create race conditions:

$$P(race\_visible) = f(T_{window} / T_{operation})$$

Adding 15 us between every syscall changes all timing relationships. This is why strace can "fix" race conditions — it serializes everything.

---

## 3. seccomp-BPF — Kernel-Side Filtering

### The Filter Model

seccomp-BPF runs a **BPF program** in the kernel to filter syscalls before they execute. This is what tools like `strace --seccomp-bpf` use for acceleration.

### BPF Instruction Cost

$$T_{filter} = N_{instructions} \times T_{instruction}$$

Where $T_{instruction} \approx 1-4ns$ (in-kernel, no context switch).

A typical seccomp filter: 10-50 instructions → 10-200 ns total.

### Comparison: ptrace vs seccomp-BPF

| Method | Per-Syscall Cost | Mechanism |
|:---|:---:|:---|
| ptrace (full) | 8-30 us | 4 context switches, userspace |
| seccomp-BPF (notify) | 2-5 us | Kernel filter + selective notify |
| seccomp-BPF (kill/allow) | 10-200 ns | Pure kernel, no userspace |

### seccomp-BPF Filter Matching

The BPF program evaluates syscall data as a **decision tree**:

```
if (syscall_nr == __NR_write) → ALLOW
if (syscall_nr == __NR_open)  → TRACE (notify strace)
default                        → ALLOW
```

With `--seccomp-bpf`, strace only gets notified for syscalls it cares about:

$$overhead = N_{filtered} \times T_{ptrace} + (N_{total} - N_{filtered}) \times T_{BPF}$$

**Example:** Tracing only `open()` calls in a program making 100,000 syscalls/s, 500 of which are open():

$$overhead = 500 \times 15\mu s + 99500 \times 0.1\mu s = 7.5ms + 9.95ms = 17.45ms/s$$

vs full ptrace: $100000 \times 15\mu s = 1.5s/s$. That's an **86x improvement**.

---

## 4. String and Buffer Reading Cost

### PTRACE_PEEKDATA Mechanics

strace reads syscall arguments (strings, buffers) from tracee memory using `PTRACE_PEEKDATA`, which reads one **word** (8 bytes on x86-64) per call:

$$reads = \lceil \frac{buffer\_size}{8} \rceil$$

$$T_{read\_buffer} = \lceil \frac{size}{8} \rceil \times T_{PEEKDATA}$$

### -s Flag (String Size Limit)

Default: `-s 32` (truncate strings at 32 bytes). For a 4096-byte buffer:

| -s value | Words read | Time | Output size |
|:---:|:---:|:---:|:---:|
| 32 | 4 | 2 us | Small |
| 256 | 32 | 16 us | Medium |
| 4096 | 512 | 256 us | Large |
| 65536 | 8192 | 4 ms | Huge |

**Rule:** Keep `-s` as small as possible. `-s 4096` on a high-throughput read() loop adds milliseconds per call.

### process_vm_readv() Optimization

Modern strace uses `process_vm_readv()` instead of PTRACE_PEEKDATA when available:

$$T_{readv} \approx 1\mu s + \frac{size}{memory\_bandwidth}$$

Single syscall regardless of size. Reading 4096 bytes: $1\mu s$ vs $256\mu s$ — a 256x improvement for large buffers.

---

## 5. Timing Precision (-T, -r, -t flags)

### Measurement Accuracy

strace timestamps use `clock_gettime(CLOCK_MONOTONIC)`:

$$resolution \approx 1\mu s \text{ (with TSC)}$$

But the **measurement overhead** is larger than the resolution:

$$accuracy = \pm T_{ptrace\_overhead}$$

A syscall measured at 5 us may actually take 0.1 us, with 4.9 us of ptrace overhead.

### -c Flag (Summary Statistics)

`strace -c` collects per-syscall statistics:

$$\bar{T}_{syscall} = \frac{\sum T_i}{count}$$

$$\% time = \frac{\sum T_{syscall\_type}}{\sum T_{all\_syscalls}} \times 100$$

**Warning:** These timings include ptrace overhead. Use `perf trace` for accurate syscall timing.

### Time Breakdown Formula

$$T_{total} = T_{userspace} + \sum_{i=1}^{N} (T_{syscall_i} + T_{ptrace_i})$$

strace only sees $T_{syscall_i}$ (approximately). $T_{userspace}$ between syscalls can be inferred:

$$T_{userspace_i} = timestamp_{i+1} - (timestamp_i + T_{syscall_i})$$

---

## 6. File Descriptor Tracking — State Machine

### fd Lifecycle

strace maintains an internal map of file descriptors to paths:

| Syscall | fd State Transition |
|:---|:---|
| open/openat → fd | $\emptyset \to path$ |
| dup/dup2(fd, newfd) | $newfd \to path(fd)$ |
| close(fd) | $path \to \emptyset$ |
| pipe([r, w]) | $\emptyset \to pipe_r, pipe_w$ |
| socket() → fd | $\emptyset \to socket$ |
| accept() → newfd | $\emptyset \to connected\_socket$ |

This state machine has $O(max\_fd)$ space and $O(1)$ lookup per operation.

---

## 7. Filtering Efficiency

### -e trace= Filter

$$T_{filtered} = \sum_{s \in S_{filter}} count(s) \times T_{full\_trace}$$

Where $S_{filter}$ is the set of traced syscall types.

### Common Filter Sets and Coverage

| Filter | Syscalls | Typical Coverage |
|:---|:---:|:---|
| `-e trace=file` | ~20 | open, stat, access, chmod... |
| `-e trace=network` | ~10 | socket, connect, bind, send... |
| `-e trace=process` | ~10 | fork, exec, wait, exit... |
| `-e trace=memory` | ~5 | mmap, munmap, brk, mprotect... |
| `-e trace=signal` | ~5 | kill, sigaction, sigprocmask... |

Tracing only network syscalls on a web server (10% of total syscalls):

$$overhead = 0.10 \times N \times T_{ptrace} + 0.90 \times N \times T_{BPF}$$

With seccomp-BPF: 90% of syscalls pass through the kernel filter at near-zero cost.

---

## 8. Summary of strace Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| ptrace overhead | $4 \times T_{ctx\_switch} + 2 \times T_{wait}$ | Constant per syscall |
| Slowdown factor | $(T_{native} + T_{overhead}) / T_{native}$ | Ratio |
| Total overhead | $syscall\_rate \times T_{overhead}$ | Linear in rate |
| Buffer reading | $\lceil size / 8 \rceil \times T_{peek}$ | Per-word I/O |
| seccomp speedup | $N_{filtered} \times T_{ptrace} / (N_{total} \times T_{ptrace})$ | Selective tracing |
| Timing accuracy | $\pm T_{ptrace\_overhead}$ | Measurement error |

---

*strace is the most powerful debugging tool you'll never want to use in production — because the observer effect is measured in microseconds per syscall, and at scale, microseconds become seconds.*
