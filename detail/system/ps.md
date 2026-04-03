# The Mathematics of ps — Process State Machines, Memory Accounting & CPU Calculation

> *ps is a snapshot of the kernel's process table. Every column — %CPU, %MEM, RSS, VSZ — is a computed value with a precise formula, and understanding those formulas is the difference between reading ps output and understanding it.*

---

## 1. Process States — The State Machine

### Linux Process State Diagram

A process transitions through states in a **finite state machine**:

```
TASK_NEW → TASK_RUNNING ⇄ TASK_INTERRUPTIBLE
              ↓               ↓
        TASK_STOPPED    TASK_UNINTERRUPTIBLE
              ↓               ↓
         TASK_TRACED    EXIT_ZOMBIE → EXIT_DEAD
```

### ps State Codes

| Code | Kernel State | Meaning | Counts in Load Avg? |
|:---:|:---|:---|:---:|
| R | TASK_RUNNING | On CPU or in run queue | Yes |
| S | TASK_INTERRUPTIBLE | Sleeping, can be signaled | No |
| D | TASK_UNINTERRUPTIBLE | Sleeping, cannot be interrupted | Yes |
| T | TASK_STOPPED | Stopped by signal (SIGSTOP) | No |
| t | TASK_TRACED | Stopped by debugger (ptrace) | No |
| Z | EXIT_ZOMBIE | Terminated, waiting for parent wait() | No |
| I | TASK_IDLE | Kernel idle thread (kernel 4.14+) | No |

### State Distribution on a Healthy System

For a server with $N$ processes:

$$N = N_R + N_S + N_D + N_T + N_Z + N_I$$

Typical: $N_S \gg N_R > N_D > N_Z \approx N_T \approx 0$

**Red flags:** $N_D > 10$ (stuck I/O), $N_Z > 5$ (parent not reaping).

---

## 2. %CPU Calculation

### The Formula

ps computes %CPU as total CPU time divided by elapsed time:

$$\%CPU = \frac{cputime_{user} + cputime_{system}}{wall\_time} \times 100$$

Where:

$$wall\_time = now - start\_time$$

$$cputime = \text{from } /proc/[pid]/stat \text{ fields: utime + stime (in clock ticks)}$$

### Clock Tick Conversion

$$cputime\_seconds = \frac{utime + stime}{CLK\_TCK}$$

Where $CLK\_TCK = 100$ on most Linux systems (see `getconf CLK_TCK`).

### Multi-Core Interpretation

%CPU can exceed 100% for multi-threaded processes:

$$\%CPU_{max} = n_{threads\_on\_cpu} \times 100\%$$

A 4-threaded process fully utilizing 4 cores: $\%CPU = 400\%$.

### Instantaneous vs Average

`ps` shows **lifetime average**. `top` shows **per-interval** CPU:

$$\%CPU_{ps} = \frac{total\_cputime}{total\_walltime} \times 100$$

$$\%CPU_{top} = \frac{\Delta cputime}{\Delta walltime} \times 100$$

A process using 100% CPU for 1 second then idle for 99 seconds:
- `ps`: $\%CPU = 1\%$
- `top` (1s interval during burst): $\%CPU = 100\%$

---

## 3. Memory Columns — VSZ, RSS, %MEM

### Virtual Size (VSZ)

$$VSZ = \sum_{vma} (vma_{end} - vma_{start})$$

Total virtual address space mapped. Includes:
- Code (text segment)
- Data (heap, stack, BSS)
- Shared libraries
- Memory-mapped files
- Unused mapped regions

**VSZ is largely meaningless** for memory pressure analysis. A process can map terabytes without using physical RAM.

### Resident Set Size (RSS)

$$RSS = \text{physical pages currently in RAM} \times page\_size$$

$$RSS \leq VSZ$$

RSS excludes swapped-out pages and includes shared pages (counted per-process).

### Shared Memory Problem

If library `libc.so` is 2 MB and mapped by 100 processes:

$$\sum RSS_{all} = 100 \times 2MB = 200MB \text{ (overcounted)}$$

$$actual\_physical = 2MB \text{ (shared, counted once)}$$

This is why $\sum RSS > physical\_RAM$ is common and not an error.

### PSS (Proportional Set Size)

$$PSS = private\_pages + \sum_{shared} \frac{shared\_pages_i}{n\_sharers_i}$$

PSS divides shared pages by the number of processes sharing them:

$$\sum PSS_{all} \approx total\_physical\_used$$

Available in `/proc/[pid]/smaps_rollup`, not directly in ps.

### %MEM Calculation

$$\%MEM = \frac{RSS}{total\_physical\_RAM} \times 100$$

---

## 4. Process Hierarchy — ppid and Tree Structure

### Process Tree as Rooted Tree

The process tree is a **rooted tree** with PID 1 (init/systemd) as root:

$$parent(p) = ppid(p) \quad \forall p \neq 1$$

### Tree Metrics

$$depth(p) = \begin{cases} 0 & \text{if } p = 1 \\ 1 + depth(parent(p)) & \text{otherwise} \end{cases}$$

$$subtree\_size(p) = 1 + \sum_{c \in children(p)} subtree\_size(c)$$

Typical depth for a service: $depth \approx 3-5$ (init → systemd → service-manager → service → worker).

### Orphan and Zombie Counting

Orphans are reparented to PID 1 (or a subreaper):

$$orphaned = \{p : parent(p)_{original} \neq parent(p)_{current}\}$$

$$zombies = \{p : state(p) = Z\}$$

Zombie accumulation rate: if parent never calls `wait()`:

$$N_{zombies}(t) = \int_0^t fork\_rate(s)\ ds$$

---

## 5. Priority and Scheduling — PRI, NI, CLS

### Priority Mapping

$$PRI_{ps} = 80 + nice \text{ (for SCHED\_OTHER)}$$

| nice | PRI (ps) | Kernel priority | Meaning |
|:---:|:---:|:---:|:---|
| -20 | 60 | 100 | Highest normal priority |
| 0 | 80 | 120 | Default |
| 19 | 99 | 139 | Lowest normal priority |

### Real-Time Priorities

$$PRI_{rt} = -1 - rt\_priority \text{ (shown negative in ps)}$$

Real-time range: 1-99 (kernel priority 0-98), always preempts normal tasks.

### Scheduling Classes

| Class (CLS) | Policy | Priority Range | Use Case |
|:---|:---|:---:|:---|
| TS | SCHED_OTHER | nice -20 to 19 | Default, CFS |
| FF | SCHED_FIFO | rt 1-99 | Real-time, no preemption by same priority |
| RR | SCHED_RR | rt 1-99 | Real-time, round-robin at same priority |
| B | SCHED_BATCH | nice -20 to 19 | Batch processing, longer timeslice |
| IDL | SCHED_IDLE | 0 | Only runs when CPU idle |
| DL | SCHED_DEADLINE | N/A | Deadline scheduling (EDF) |

---

## 6. Time Columns — ELAPSED, TIME, STIME

### Elapsed Time

$$ELAPSED = now - start\_time$$

### CPU TIME

$$TIME = utime + stime \text{ (converted from clock ticks to HH:MM:SS)}$$

### CPU Efficiency

$$efficiency = \frac{TIME}{ELAPSED}$$

| Efficiency | Interpretation |
|:---:|:---|
| $\approx 0$ | Mostly sleeping (I/O-bound, idle) |
| $\approx 1$ | Fully utilizing one core |
| $> 1$ | Multi-threaded, using multiple cores |
| $= n_{cores}$ | Fully utilizing all cores |

---

## 7. Custom Format — Output Column Sizing

### Column Width Calculation

ps auto-sizes columns based on data:

$$width_i = \max(header\_len_i, \max_{p} value\_len(p, i))$$

$$total\_width = \sum_i width_i + (n_{columns} - 1) \times separator\_width$$

### Efficient Formats for Scripting

| Need | Format | Columns |
|:---|:---|:---:|
| Memory hogs | `ps -eo pid,rss,comm --sort=-rss` | 3 |
| CPU hogs | `ps -eo pid,pcpu,comm --sort=-pcpu` | 3 |
| Zombie hunt | `ps -eo pid,ppid,state,comm \| grep Z` | 4 |
| Thread count | `ps -eo pid,nlwp,comm --sort=-nlwp` | 3 |

### Thread Counting

$$nlwp = \text{number of lightweight processes (threads)}$$

Total system threads: $\sum_{p} nlwp(p)$

Thread limit: `kernel.threads-max` (default: $\frac{RAM\_pages}{4}$).

---

## 8. Summary of ps Mathematics

| Column | Formula | Notes |
|:---|:---|:---|
| %CPU | $(utime + stime) / walltime \times 100$ | Lifetime average |
| %MEM | $RSS / total\_RAM \times 100$ | Instantaneous |
| VSZ | $\sum vma\_sizes$ | Virtual, not physical |
| RSS | $resident\_pages \times page\_size$ | Overcounts shared |
| PSS | $private + shared/n\_sharers$ | True proportional |
| PRI | $80 + nice$ | Normal scheduling |
| TIME | $(utime + stime) / CLK\_TCK$ | Total CPU time |
| ELAPSED | $now - start\_time$ | Wall clock |

---

*ps reads /proc — the kernel's window into its own soul. Every number is a counter or a computation, and they all tell the same story: how much of the machine's finite resources this process has consumed.*
