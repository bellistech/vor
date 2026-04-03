# The Mathematics of kill — Signal Delivery, Process Termination & Graceful Shutdown

> *kill is signal delivery, and signals are the kernel's interrupt mechanism for processes. Each signal has a number, a default action, and a delivery cost — and the ordering of signals in a shutdown sequence is a protocol with timing constraints.*

---

## 1. Signal Number Space

### Standard Signals (POSIX)

Signals occupy a fixed numeric range:

$$signal\_number \in [1, 64] \text{ on Linux}$$

| Range | Type | Count |
|:---|:---|:---:|
| 1-31 | Standard signals | 31 |
| 32-33 | Reserved (NPTL) | 2 |
| 34-64 | Real-time signals | 31 |

### Key Signals for Process Control

| Signal | Number | Default Action | Catchable? | Purpose |
|:---|:---:|:---|:---:|:---|
| SIGHUP | 1 | Terminate | Yes | Hangup / reload config |
| SIGINT | 2 | Terminate | Yes | Ctrl+C |
| SIGQUIT | 3 | Core dump | Yes | Ctrl+\\ |
| SIGKILL | 9 | Terminate | **No** | Unconditional kill |
| SIGTERM | 15 | Terminate | Yes | Graceful termination |
| SIGSTOP | 19 | Stop | **No** | Unconditional pause |
| SIGCONT | 18 | Continue | Yes | Resume stopped process |
| SIGUSR1 | 10 | Terminate | Yes | User-defined |
| SIGUSR2 | 12 | Terminate | Yes | User-defined |

### Uncatchable Signals

Only two signals cannot be caught, blocked, or ignored:

$$uncatchable = \{SIGKILL(9), SIGSTOP(19)\}$$

$$\forall s \notin uncatchable: sigaction(s, handler) \text{ is valid}$$

---

## 2. Signal Delivery Mechanics

### Delivery Cost

$$T_{signal} = T_{send} + T_{delivery} + T_{handler}$$

| Component | Cost | Notes |
|:---|:---:|:---|
| kill() syscall | 0.5-2 us | Send signal to kernel |
| Delivery to process | 1-5 us | Kernel interrupts target |
| Handler execution | 0.1 us - seconds | Depends on handler code |
| **SIGKILL delivery** | **1-10 us** | No handler, kernel terminates |

### Signal Queue Model

Standard signals (1-31) are **not queued** — only one pending per signal number:

$$pending(sig) \in \{0, 1\}$$

If a signal is sent twice before delivery, the second is lost.

Real-time signals (34-64) **are queued**:

$$pending(rt\_sig) \in [0, RLIMIT\_SIGPENDING]$$

Default `RLIMIT_SIGPENDING`: ~3800 per user.

---

## 3. Graceful Shutdown Protocol — SIGTERM then SIGKILL

### The Standard Pattern

```
kill -TERM $pid    # Ask nicely
sleep $timeout     # Wait for graceful exit
kill -KILL $pid    # Force if still alive
```

### Timeout Sizing

$$timeout \geq T_{graceful\_shutdown}$$

| Application | Typical Graceful Time | Recommended Timeout |
|:---|:---:|:---:|
| Simple daemon | 0.1-1 s | 5 s |
| Web server (drain connections) | 1-30 s | 30 s |
| Database (flush buffers) | 5-60 s | 90 s |
| Java application (JVM shutdown) | 5-30 s | 60 s |

### Data Loss Risk

$$P(data\_loss) = \begin{cases} P(unflushed\_data) & \text{if SIGKILL used} \\ \approx 0 & \text{if SIGTERM handler completes} \end{cases}$$

For a database with write-ahead log:

$$data\_at\_risk = write\_rate \times T_{since\_last\_fsync}$$

With 1 GB/s writes and 1-second fsync interval: up to 1 GB at risk from SIGKILL.

---

## 4. Signal Propagation — Process Groups and Sessions

### Process Group Signals

`kill -SIGNAL -$pgid` sends to all processes in a group:

$$recipients = \{p : pgid(p) = pgid\}$$

$$T_{group\_signal} = |recipients| \times T_{per\_signal}$$

### Session Signals

SIGHUP propagation on terminal close:

```
Session leader dies → SIGHUP to foreground process group
```

$$propagation = \{p : sid(p) = session \land pgid(p) = foreground\_pgid\}$$

### Kill All User Processes

`kill -TERM -1` from root kills all processes (except PID 1):

$$victims = \{p : p \neq 1 \land uid(p) \text{ matches or caller is root}\}$$

Order of termination is **undefined** — processes die in arbitrary order, which can cause dependency issues.

---

## 5. Signal Handling Cost — Handler Overhead

### Context Switch for Signal Delivery

When a signal is delivered to a process in userspace:

1. Save current register state on signal stack
2. Set up signal frame (siginfo, ucontext)
3. Jump to handler function
4. Handler returns via `sigreturn()` syscall
5. Restore registers, resume normal execution

$$T_{signal\_roundtrip} = T_{save} + T_{handler} + T_{sigreturn} + T_{restore}$$

$$T_{overhead} \approx 2-5\mu s \text{ (excluding handler body)}$$

### Signal Storm Impact

If a process receives signals at rate $\lambda$:

$$CPU_{signal\_handling} = \lambda \times (T_{overhead} + T_{handler})$$

**Example:** 10,000 SIGCHLD/s (forking server reaping children):

$$CPU = 10000 \times (3\mu s + 1\mu s) = 40ms/s = 4\% \text{ of one core}$$

### SA_RESTART Flag

Without SA_RESTART, interrupted syscalls return EINTR:

$$P(EINTR) = P(\text{signal during syscall}) = \frac{T_{syscall}}{T_{between\_signals}} \approx \lambda \times T_{syscall}$$

With high signal rate and slow syscalls (e.g., read with 1s timeout):

$$P(EINTR) = 10000 \times 1 = \text{essentially always interrupted}$$

---

## 6. OOM Kill vs Manual Kill

### OOM Kill Signal

The OOM killer sends SIGKILL (uncatchable) to the selected victim:

$$P(killed\_by\_OOM) \propto oom\_score(p)$$

$$oom\_score = \frac{RSS + swap}{RAM + swap} \times 1000 + oom\_score\_adj$$

### Comparison

| Method | Signal | Catchable | Data Loss Risk | Cleanup |
|:---|:---:|:---:|:---:|:---|
| `kill -TERM` | 15 | Yes | Low | Handler runs |
| `kill -KILL` | 9 | No | High | No cleanup |
| OOM killer | 9 | No | High | No cleanup |
| `kill -HUP` | 1 | Yes | Low | Reload config |

---

## 7. Real-Time Signal Priority

### Delivery Order

When multiple signals are pending:

$$delivery\_order: standard\_signals < real\_time\_signals$$

Among real-time signals: lower number delivered first:

$$SIGRTMIN < SIGRTMIN+1 < ... < SIGRTMAX$$

### Real-Time Signal Queue Depth

$$max\_queued = RLIMIT\_SIGPENDING$$

$$memory\_per\_signal = sizeof(siginfo\_t) \approx 128 \text{ bytes}$$

$$total\_memory = max\_queued \times 128$$

At default 3842 signals: $\approx 480 KB$ per user.

---

## 8. Summary of kill Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Signal space | $[1, 64]$ on Linux | Integer range |
| Standard pending | $\{0, 1\}$ per signal | Binary (no queue) |
| RT pending | $[0, RLIMIT]$ per signal | Queue |
| Signal delivery | $T_{send} + T_{deliver} + T_{handler}$ | Latency sum |
| Group kill | $\|group\| \times T_{signal}$ | Linear |
| Shutdown timeout | $\geq T_{graceful}$ | Lower bound |
| Data at risk | $write\_rate \times T_{unflushed}$ | SIGKILL risk |
| Signal storm | $\lambda \times (T_{overhead} + T_{handler})$ | CPU overhead |

---

*kill is the syscall interface to the kernel's interrupt system for processes. SIGTERM is a request; SIGKILL is an order. The 15-then-9 pattern is a protocol as old as Unix itself, and the timeout between them is the measure of how much you trust the application to clean up after itself.*
