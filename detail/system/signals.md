# The Mathematics of Signals -- Delivery Ordering, Queuing Theory, and Bitmask Operations

> *Unix signals encode process events as integers in a finite set, delivered through*
> *bitmask-based pending/blocked state machines. Real-time signal extensions add FIFO*
> *queuing with bounded capacity, turning signal handling into a study of queue theory and bitwise algebra.*

---

## 1. Signal Mask Bitwise Operations (Boolean Algebra)

### The Problem

The kernel tracks which signals are blocked, pending, ignored, and caught using 64-bit bitmasks. Every signal operation reduces to bitwise logic.

### The Formula

For signal number $s$ (1-64), the corresponding bit position is $s - 1$:

$$\text{bit}(s) = 1 \ll (s - 1)$$

Mask operations:

$$\text{block}(M, s) = M \; | \; \text{bit}(s)$$

$$\text{unblock}(M, s) = M \; \& \; \lnot\text{bit}(s)$$

$$\text{is\_set}(M, s) = (M \; \& \; \text{bit}(s)) \neq 0$$

$$\text{pending\_deliverable} = \text{SigPnd} \; \& \; \lnot\text{SigBlk}$$

### Worked Examples

Decoding `/proc/$PID/status` signal masks:

```
SigBlk: 0000000000000004
SigIgn: 0000000000001000
SigCgt: 0000000180004002
```

Blocked mask `0x4` = bit 2 set:

$$0x4 = 0\text{b}100 \implies \text{signal } 3 \text{ (SIGQUIT) blocked}$$

Caught mask `0x180004002`:

| Bit Position | Signal | Name |
|-------------|--------|------|
| 1 | 2 | SIGINT |
| 14 | 15 | SIGTERM |
| 17 | 18 | SIGCONT |
| 31 | 32 | SIGRTMIN |
| 32 | 33 | SIGRTMIN+1 |

Combined operations:

$$\text{deliverable} = \text{SigPnd} \; \& \; \lnot\text{SigBlk} \; \& \; \lnot\text{SigIgn}$$

If `SigPnd = 0x8002` (SIGINT + SIGKILL pending), `SigBlk = 0x0002` (SIGINT blocked):

$$\text{deliverable} = 0x8002 \; \& \; \lnot 0x0002 = 0x8002 \; \& \; 0x\text{FFFD} = 0x8000$$

Only SIGKILL (bit 8) is deliverable (it cannot actually be blocked, but this illustrates the logic).

## 2. Signal Delivery Ordering (Priority Queue)

### The Problem

When multiple signals are pending simultaneously, in what order does the kernel deliver them?

### The Formula

For standard signals (1-31), the delivery order follows the lowest-numbered signal first:

$$\text{next\_signal} = \text{ffs}(\text{pending\_deliverable})$$

where $\text{ffs}$ (find first set) returns the position of the lowest set bit.

For real-time signals (32-64), they are delivered in FIFO order per signal number, with lower-numbered signals having priority:

$$\text{priority}(s) = \begin{cases}
s & \text{for standard signals (lower = higher priority)} \\
s & \text{for RT signals (lower = higher priority)} \\
\end{cases}$$

Standard signals always have priority over real-time signals:

$$\text{priority}(\text{standard}) > \text{priority}(\text{RT})$$

### Worked Examples

Pending signals: SIGTERM(15), SIGINT(2), SIGUSR1(10), SIGRTMIN(32), SIGRTMIN+1(33)

Delivery order:

| Order | Signal | Number | Type |
|-------|--------|--------|------|
| 1st | SIGINT | 2 | Standard |
| 2nd | SIGUSR1 | 10 | Standard |
| 3rd | SIGTERM | 15 | Standard |
| 4th | SIGRTMIN | 32 | Real-time |
| 5th | SIGRTMIN+1 | 33 | Real-time |

## 3. Standard Signal Coalescing (Set Semantics)

### The Problem

Standard signals (1-31) are not queued. If the same signal is sent multiple times before delivery, only one instance is recorded. How much information is lost?

### The Formula

For standard signal $s$ sent $n$ times while blocked:

$$\text{delivered}(s, n) = \min(n, 1) = \begin{cases}
0 & \text{if } n = 0 \\
1 & \text{if } n \geq 1
\end{cases}$$

Information loss:

$$\text{loss}(s, n) = n - \text{delivered}(s, n) = n - 1 \quad (\text{for } n \geq 1)$$

### Worked Examples

A monitoring system sends SIGUSR1 to report events:

| Events Occurred | SIGUSR1 Sent | SIGUSR1 Delivered | Events Lost |
|----------------|-------------|-------------------|-------------|
| 1 | 1 | 1 | 0 |
| 5 | 5 | 1 | 4 |
| 100 | 100 | 1 | 99 |
| 1,000 | 1,000 | 1 | 999 |

This is why standard signals should not be used for event counting. Use real-time signals or other IPC.

## 4. Real-Time Signal Queue (Bounded FIFO)

### The Problem

Real-time signals (SIGRTMIN to SIGRTMAX) are queued with associated data values. The queue has a bounded capacity. What happens at capacity?

### The Formula

Queue capacity per user:

$$Q_{\text{max}} = \text{RLIMIT\_SIGPENDING}$$

Default value (from `/proc/sys/kernel/rtsig-max` or ulimit):

$$Q_{\text{default}} = \frac{\text{total\_ram\_pages}}{2} \quad \text{(capped at a sysctl limit)}$$

Typically around 63,432 on modern systems.

Queue occupancy:

$$Q_{\text{current}} = \sum_{s=\text{SIGRTMIN}}^{\text{SIGRTMAX}} q_s$$

where $q_s$ is the number of queued instances of signal $s$.

Overflow condition:

$$\text{EAGAIN if } Q_{\text{current}} \geq Q_{\text{max}}$$

### Worked Examples

| Scenario | Queue Capacity | RT Signals Queued | Status |
|----------|---------------|-------------------|--------|
| Normal | 63,432 | 100 | OK |
| Moderate | 63,432 | 30,000 | OK |
| Heavy | 63,432 | 63,432 | Full |
| Overflow | 63,432 | 63,433 | EAGAIN |

Real-time signal with data (`sigqueue`):

```c
union sigval val;
val.sival_int = 42;
sigqueue(pid, SIGRTMIN, val);  // Queued with data
sigqueue(pid, SIGRTMIN, val);  // Second instance also queued
sigqueue(pid, SIGRTMIN, val);  // Third instance also queued
// All three delivered in FIFO order
```

## 5. Signal Delivery Latency (Timing)

### The Problem

How long does it take from `kill()` to the signal handler executing in the target process?

### The Formula

$$t_{\text{delivery}} = t_{\text{syscall}} + t_{\text{schedule}} + t_{\text{context\_switch}} + t_{\text{handler\_entry}}$$

For a blocked (sleeping) process:

$$t_{\text{delivery}} \approx t_{\text{wakeup}} + t_{\text{context\_switch}}$$

For a running process:

$$t_{\text{delivery}} \leq t_{\text{kernel\_return}} \quad \text{(checked at syscall/interrupt return)}$$

### Worked Examples

Typical latencies on modern x86_64:

| Scenario | Latency | Notes |
|----------|---------|-------|
| Target running on same CPU | 1-5 us | Checked at next kernel return |
| Target running on different CPU | 5-20 us | Requires IPI (inter-processor interrupt) |
| Target sleeping (interruptible) | 10-50 us | Wake + context switch |
| Target in uninterruptible sleep | Unbounded | Delivered when state changes |
| SIGKILL to stopped process | 10-100 us | Forced wakeup |
| SIGKILL across PID namespace | 20-100 us | Namespace traversal overhead |

## 6. Zombie Process Accumulation (Process Lifecycle)

### The Problem

When child processes exit, they become zombies until the parent calls `wait()`. Failure to handle SIGCHLD leads to zombie accumulation.

### The Formula

Zombie accumulation rate without SIGCHLD handling:

$$Z(t) = \sum_{i=0}^{t} \text{exits}(i) - \sum_{i=0}^{t} \text{waits}(i)$$

With SIGCHLD coalescing (standard signal), maximum missed reaps per signal delivery:

$$\text{missed\_per\_delivery} = \text{exits\_during\_handler} - 1$$

Correct reaping loop in handler:

```c
while (waitpid(-1, NULL, WNOHANG) > 0) { /* reap all */ }
```

### Worked Examples

Server forking 100 connections/second, each lasting 1 second:

| SIGCHLD Handling | Zombies After 1 Hour | PID Table Impact |
|-----------------|---------------------|-----------------|
| No handler | 360,000 | Exhausted |
| Handler with single wait() | 0 -- 359,999 | Variable (coalescing) |
| Handler with waitpid loop | 0 | Clean |
| SIG_IGN for SIGCHLD | 0 | Kernel auto-reaps |

## 7. Signal Safety and Re-entrancy (Concurrency)

### The Problem

Signal handlers interrupt normal execution at arbitrary points. What functions are safe to call?

### The Formula

Define the set of async-signal-safe functions $S_{\text{safe}}$:

$$f \in S_{\text{safe}} \iff f \text{ is reentrant} \lor f \text{ is atomic}$$

A signal handler $H$ is correct if:

$$\forall f \in H: f \in S_{\text{safe}}$$

### Worked Examples

| Function | Async-Signal-Safe | Reason |
|----------|------------------|--------|
| `write()` | Yes | Atomic syscall |
| `_exit()` | Yes | Atomic syscall |
| `signal()` | Yes | POSIX mandated |
| `printf()` | **No** | Uses internal locks |
| `malloc()` | **No** | Uses internal locks |
| `syslog()` | **No** | Uses stdio + locks |
| `exit()` | **No** | Runs atexit handlers |
| `strlen()` | Yes | Pure computation |

The safe pattern: set a `volatile sig_atomic_t` flag in the handler, check it in the main loop.

POSIX defines approximately 130 async-signal-safe functions (see `signal-safety(7)`).

## Prerequisites

boolean-algebra, queue-theory, process-lifecycle, concurrency-primitives, bitwise-operations

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| Send signal (kill syscall) | O(1) | O(1) |
| Signal mask update (sigprocmask) | O(1) | O(1) bitmask |
| Find lowest pending signal (ffs) | O(1) hw instruction | O(1) |
| RT signal enqueue (sigqueue) | O(1) | O(1) per queued signal |
| RT signal dequeue | O(1) | O(1) |
| waitpid reap | O(1) per child | O(1) |
| Signal delivery (blocked->running) | O(1) + context switch | O(stack frame) |
| Signal mask decode (all 64 bits) | O(64) = O(1) | O(1) |
