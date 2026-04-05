# The Mathematics of Operating Systems -- Scheduling, Memory, Deadlock, and the Foundations of Multiprogramming

> *An operating system is the most complex piece of software most programmers will ever depend on, yet its core mechanisms rest on elegant mathematical structures -- queueing theory, graph algorithms, optimal replacement policies, and resource allocation invariants that can be stated and proved with precision.*

---

## 1. Belady's Anomaly (FIFO Page Replacement)

### The Problem

Prove that FIFO page replacement can produce more page faults when given more physical frames -- a counterintuitive result known as Belady's anomaly.

### The Formula

Consider a reference string $\omega = r_1, r_2, \ldots, r_T$ and let $F_k(t)$ denote the set of pages in memory under FIFO with $k$ frames after processing reference $r_t$. Belady's anomaly states:

$$\exists \, \omega, \, k : \text{faults}(k+1, \omega) > \text{faults}(k, \omega)$$

A page replacement algorithm is a **stack algorithm** if for all reference strings and all times $t$:

$$S_k(t) \subseteq S_{k+1}(t)$$

where $S_k(t)$ is the set of pages in memory with $k$ frames at time $t$. Stack algorithms (LRU, OPT) satisfy this **inclusion property** and therefore never exhibit Belady's anomaly. FIFO is not a stack algorithm.

### Worked Example

Reference string: $1, 2, 3, 4, 1, 2, 5, 1, 2, 3, 4, 5$

**3 frames (FIFO):**

| Step | Ref | Frame 1 | Frame 2 | Frame 3 | Fault? |
|------|-----|---------|---------|---------|--------|
| 1    | 1   | 1       | -       | -       | Yes    |
| 2    | 2   | 1       | 2       | -       | Yes    |
| 3    | 3   | 1       | 2       | 3       | Yes    |
| 4    | 4   | 4       | 2       | 3       | Yes    |
| 5    | 1   | 4       | 1       | 3       | Yes    |
| 6    | 2   | 4       | 1       | 2       | Yes    |
| 7    | 5   | 5       | 1       | 2       | Yes    |
| 8    | 1   | 5       | 1       | 2       | No     |
| 9    | 2   | 5       | 1       | 2       | No     |
| 10   | 3   | 5       | 3       | 2       | Yes    |
| 11   | 4   | 5       | 3       | 4       | Yes    |
| 12   | 5   | 5       | 3       | 4       | No     |

Total faults: **9**

**4 frames (FIFO):**

| Step | Ref | Frame 1 | Frame 2 | Frame 3 | Frame 4 | Fault? |
|------|-----|---------|---------|---------|---------|--------|
| 1    | 1   | 1       | -       | -       | -       | Yes    |
| 2    | 2   | 1       | 2       | -       | -       | Yes    |
| 3    | 3   | 1       | 2       | 3       | -       | Yes    |
| 4    | 4   | 1       | 2       | 3       | 4       | Yes    |
| 5    | 1   | 1       | 2       | 3       | 4       | No     |
| 6    | 2   | 1       | 2       | 3       | 4       | No     |
| 7    | 5   | 5       | 2       | 3       | 4       | Yes    |
| 8    | 1   | 5       | 1       | 3       | 4       | Yes    |
| 9    | 2   | 5       | 1       | 2       | 4       | Yes    |
| 10   | 3   | 5       | 1       | 2       | 3       | Yes    |
| 11   | 4   | 4       | 1       | 2       | 3       | Yes    |
| 12   | 5   | 4       | 5       | 2       | 3       | Yes    |

Total faults: **10**

More frames (4 > 3) yet more faults (10 > 9). The anomaly arises because FIFO eviction order depends on arrival time, not usage recency, so adding a frame changes the entire eviction sequence rather than simply retaining one additional page.

### Why It Matters

Belady's anomaly demonstrates that not all "reasonable-sounding" replacement policies have monotonic performance. It motivates the use of stack algorithms (LRU, OPT) which provably avoid this pathology. The inclusion property $S_k(t) \subseteq S_{k+1}(t)$ is the key structural invariant.

---

## 2. LRU Optimality Proof Sketch

### The Problem

Show that among all online page replacement algorithms with $k$ frames, LRU is $k$-competitive with the optimal offline algorithm OPT.

### The Formula

Let $\text{LRU}_k(\omega)$ and $\text{OPT}_k(\omega)$ denote the number of page faults on reference string $\omega$ with $k$ frames. Sleator and Tarjan (1985) proved:

$$\text{LRU}_k(\omega) \le k \cdot \text{OPT}_{\lceil k/2 \rceil}(\omega)$$

More precisely, for any online deterministic algorithm $A$:

$$\text{competitive ratio}(A) \ge k$$

LRU achieves this bound, making it optimally competitive among deterministic online algorithms.

### Proof Sketch

**Partition argument.** Divide the reference string into **phases**, where each phase consists of references to exactly $k$ distinct pages. Within a single phase:

- LRU faults at most $k$ times (once per distinct page in the phase).
- OPT with $\lceil k/2 \rceil$ frames faults at least once per phase (by a pigeonhole argument: $k$ distinct pages cannot all fit in $\lceil k/2 \rceil$ frames).

Summing over all phases:

$$\text{LRU}_k(\omega) \le k \cdot (\text{number of phases}) \le k \cdot \text{OPT}_{\lceil k/2 \rceil}(\omega)$$

**Lower bound.** An adversary can force any deterministic online algorithm to fault on every request by always requesting the page not in memory. With $k$ frames and $k+1$ pages, the adversary causes $T$ faults in $T$ requests, while OPT faults at most $\lceil T/k \rceil$ times (by keeping the right pages). This gives a competitive ratio of at least $k$.

### Why It Matters

This result from competitive analysis shows that LRU is the best we can do without future knowledge. Randomized algorithms (e.g., RANDOM-MARK) can achieve $O(\log k)$ competitive ratio, explaining why randomized approaches are sometimes preferred in practice.

---

## 3. Banker's Algorithm Worked Example

### The Problem

Given a system with $n$ processes and $m$ resource types, determine whether a particular resource request can be safely granted without risking deadlock.

### The Formula

**Safety condition.** A state is **safe** if there exists a sequence $\langle P_{\pi(1)}, P_{\pi(2)}, \ldots, P_{\pi(n)} \rangle$ such that for each $P_{\pi(i)}$:

$$\text{Need}_{\pi(i)} \le \text{Available} + \sum_{j=1}^{i-1} \text{Allocation}_{\pi(j)}$$

This means each process can finish with the currently available resources plus those released by all previously completed processes.

### Worked Example

System: 5 processes ($P_0$--$P_4$), 3 resource types ($A$, $B$, $C$). Total resources: $(10, 5, 7)$.

**Current state:**

| Process | Allocation | Max   | Need  |
|---------|-----------|-------|-------|
| $P_0$   | (0,1,0)   | (7,5,3) | (7,4,3) |
| $P_1$   | (2,0,0)   | (3,2,2) | (1,2,2) |
| $P_2$   | (3,0,2)   | (9,0,2) | (6,0,0) |
| $P_3$   | (2,1,1)   | (2,2,2) | (0,1,1) |
| $P_4$   | (0,0,2)   | (4,3,3) | (4,3,1) |

$\text{Available} = (10,5,7) - (7,2,5) = (3,3,2)$

**Safety check:**

Step 1. $\text{Work} = (3,3,2)$. Find $P_i$ with $\text{Need}_i \le \text{Work}$.
- $P_1$: $(1,2,2) \le (3,3,2)$? Yes.
- Execute $P_1$: $\text{Work} = (3,3,2) + (2,0,0) = (5,3,2)$.

Step 2. $\text{Work} = (5,3,2)$.
- $P_3$: $(0,1,1) \le (5,3,2)$? Yes.
- Execute $P_3$: $\text{Work} = (5,3,2) + (2,1,1) = (7,4,3)$.

Step 3. $\text{Work} = (7,4,3)$.
- $P_4$: $(4,3,1) \le (7,4,3)$? Yes.
- Execute $P_4$: $\text{Work} = (7,4,3) + (0,0,2) = (7,4,5)$.

Step 4. $\text{Work} = (7,4,5)$.
- $P_0$: $(7,4,3) \le (7,4,5)$? Yes.
- Execute $P_0$: $\text{Work} = (7,4,5) + (0,1,0) = (7,5,5)$.

Step 5. $\text{Work} = (7,5,5)$.
- $P_2$: $(6,0,0) \le (7,5,5)$? Yes.
- Execute $P_2$: $\text{Work} = (7,5,5) + (3,0,2) = (10,5,7)$.

Safe sequence: $\langle P_1, P_3, P_4, P_0, P_2 \rangle$. The state is **safe**.

**Now suppose $P_1$ requests $(1,0,2)$:**

Check: $(1,0,2) \le \text{Need}_1 = (1,2,2)$? Yes.
Check: $(1,0,2) \le \text{Available} = (3,3,2)$? Yes.

Tentatively allocate:
- $\text{Available} = (3,3,2) - (1,0,2) = (2,3,0)$
- $\text{Allocation}_1 = (2,0,0) + (1,0,2) = (3,0,2)$
- $\text{Need}_1 = (1,2,2) - (1,0,2) = (0,2,0)$

Run safety algorithm on new state. Find safe sequence $\langle P_1, P_3, P_4, P_0, P_2 \rangle$. Safe. Grant the request.

### Complexity

The safety algorithm runs in $O(n^2 \cdot m)$ time: at most $n$ passes through the process list, each pass checking $n$ processes against $m$ resource types.

### Why It Matters

The banker's algorithm is the canonical deadlock avoidance algorithm. Its conservatism (it only grants requests that maintain safety) means it may deny requests that would not actually cause deadlock, but it guarantees the system never enters a deadlocked state.

---

## 4. CFS Scheduling Mathematics

### The Problem

Formalize the Completely Fair Scheduler's (CFS) approach to CPU time allocation and show how virtual runtime and the red-black tree data structure achieve proportional fairness.

### The Formula

Each task $i$ has a **weight** $w_i$ derived from its nice value:

$$w_i = \frac{1024}{1.25^{\text{nice}_i}}$$

(Approximation. Actual values from the `sched_prio_to_weight` table.)

The **virtual runtime** of task $i$ advances as:

$$\text{vruntime}_i \mathrel{+}= \Delta t_{\text{exec}} \cdot \frac{w_0}{w_i}$$

where $w_0 = 1024$ is the weight of a nice-0 task and $\Delta t_{\text{exec}}$ is wall-clock time spent running.

**Ideal fair share.** With $n$ runnable tasks, the ideal CPU share for task $i$ is:

$$\text{share}_i = \frac{w_i}{\sum_{j=1}^{n} w_j}$$

**Time slice.** CFS assigns:

$$\text{slice}_i = \text{target\_latency} \cdot \frac{w_i}{\sum_{j=1}^{n} w_j}$$

subject to $\text{slice}_i \ge \text{min\_granularity}$ (typically 0.75 ms).

The **target latency** (typically 6 ms for $\le 8$ tasks) is the period in which every runnable task should execute at least once.

### Data Structure: Red-Black Tree

Tasks are stored in a red-black tree keyed by `vruntime`:

- **Pick next:** always the leftmost node (smallest vruntime). $O(1)$ with a cached pointer.
- **Insert/delete:** $O(\log n)$ with at most 3 rotations for rebalancing.
- **Fairness invariant:** at any point, the difference in vruntime between any two runnable tasks is bounded by the target latency.

### Worked Example

Three tasks with nice values $-5$, $0$, $+5$:

$$w_{-5} \approx 3121, \quad w_0 = 1024, \quad w_{+5} \approx 335$$

$$\text{total weight} = 3121 + 1024 + 335 = 4480$$

$$\text{share}_{-5} = \frac{3121}{4480} \approx 69.7\%, \quad \text{share}_0 = \frac{1024}{4480} \approx 22.9\%, \quad \text{share}_{+5} = \frac{335}{4480} \approx 7.5\%$$

With target latency = 6 ms:

$$\text{slice}_{-5} = 6 \cdot 0.697 = 4.18 \text{ ms}, \quad \text{slice}_0 = 6 \cdot 0.229 = 1.37 \text{ ms}, \quad \text{slice}_{+5} = 6 \cdot 0.075 = 0.45 \text{ ms}$$

After running for its slice, each task has accumulated the same vruntime increment (approximately $6 / 3 = 2$ ms of virtual time), maintaining fairness in the virtual timeline.

### Why It Matters

CFS replaced the $O(1)$ scheduler in Linux 2.6.23 because its logarithmic operations are negligible in practice (typical systems have $n < 1000$ runnable tasks), while providing bounded latency and proportional fairness without the complex heuristics of MLFQ.

---

## 5. Multi-Level Page Table Walk Analysis

### The Problem

Analyze the cost of address translation through a multi-level page table hierarchy, quantifying the TLB's impact on performance.

### The Formula

For an $L$-level page table, a TLB miss requires $L + 1$ memory accesses: one per level of the page table plus one for the actual data.

**Effective Access Time (EAT):**

$$\text{EAT} = h \cdot (t_{\text{TLB}} + t_{\text{mem}}) + (1 - h) \cdot (t_{\text{TLB}} + (L + 1) \cdot t_{\text{mem}})$$

where:
- $h$ = TLB hit rate
- $t_{\text{TLB}}$ = TLB lookup time (typically $< 1$ ns, often overlapped with cache access)
- $t_{\text{mem}}$ = memory access time
- $L$ = number of page table levels

Simplifying with $t_{\text{TLB}} \approx 0$ (overlapped):

$$\text{EAT} = t_{\text{mem}} + (1 - h) \cdot L \cdot t_{\text{mem}} = t_{\text{mem}} \cdot (1 + (1 - h) \cdot L)$$

### Worked Example: x86-64 Four-Level Page Table

Parameters: $L = 4$, $t_{\text{mem}} = 100$ ns, $h = 0.99$.

$$\text{EAT} = 100 \cdot (1 + (1 - 0.99) \cdot 4) = 100 \cdot (1 + 0.04) = 104 \text{ ns}$$

Without TLB ($h = 0$):

$$\text{EAT} = 100 \cdot (1 + 4) = 500 \text{ ns}$$

The TLB reduces access time by a factor of $\approx 4.8\times$.

**Break-even analysis.** For the TLB to provide at least a $2\times$ speedup:

$$100 \cdot (1 + (1 - h) \cdot 4) \le \frac{500}{2}$$
$$1 + 4(1 - h) \le 2.5$$
$$1 - h \le 0.375$$
$$h \ge 0.625$$

Even a modest 63% TLB hit rate yields a $2\times$ improvement. In practice, TLB hit rates exceed 99% due to spatial and temporal locality.

### Why It Matters

The page table walk is the hidden cost of virtual memory. Modern processors use multi-level TLBs, page walk caches (also called "paging structure caches"), and hardware page table walkers to mitigate this cost. Understanding the multiplicative effect of each level motivates huge pages (2 MB or 1 GB), which reduce both TLB pressure and walk depth.

---

## 6. Demand Paging Performance Analysis

### The Problem

Derive the effective access time for a demand-paged virtual memory system and determine the maximum tolerable page fault rate for acceptable performance.

### The Formula

$$\text{EAT} = (1 - p) \cdot t_{\text{mem}} + p \cdot t_{\text{fault}}$$

where:
- $p$ = page fault probability
- $t_{\text{mem}}$ = memory access time (typically 100--200 ns)
- $t_{\text{fault}}$ = page fault service time (typically 2--10 ms)

The page fault service time includes:

1. Trap to OS ($\sim 1\,\mu\text{s}$)
2. Save process state ($\sim 1\,\mu\text{s}$)
3. Determine page is not in memory ($\sim 1\,\mu\text{s}$)
4. Disk seek + read ($\sim 2\text{--}8$ ms, dominates)
5. Update page table ($\sim 1\,\mu\text{s}$)
6. Restart instruction ($\sim 1\,\mu\text{s}$)

### Worked Example

$t_{\text{mem}} = 200$ ns, $t_{\text{fault}} = 8$ ms $= 8{,}000{,}000$ ns.

$$\text{EAT} = (1 - p) \cdot 200 + p \cdot 8{,}000{,}000 = 200 + 7{,}999{,}800 \cdot p$$

For less than 10% degradation ($\text{EAT} \le 220$ ns):

$$200 + 7{,}999{,}800 \cdot p \le 220$$
$$7{,}999{,}800 \cdot p \le 20$$
$$p \le 2.5 \times 10^{-6}$$

This means fewer than 1 page fault per 400,000 memory accesses. The extreme sensitivity to $p$ explains why page replacement algorithm quality matters enormously and why working set management is critical.

### With SSD Backing Store

If $t_{\text{fault}} = 200\,\mu\text{s} = 200{,}000$ ns (SSD random read):

$$\text{EAT} = 200 + 199{,}800 \cdot p$$

For 10% degradation: $p \le 1.0 \times 10^{-4}$, or 1 fault per 10,000 accesses. SSDs relax the page fault budget by roughly $40\times$.

### Why It Matters

This analysis reveals why the disk-memory speed gap makes demand paging performance so sensitive to fault rate. It drives the design of page replacement algorithms, prefetching strategies, and memory overcommit policies. The transition to SSDs has made aggressive page-out policies more viable.

---

## 7. The Dining Philosophers Problem and Solutions

### The Problem

Five philosophers sit around a table, each alternating between thinking and eating. Between each pair of adjacent philosophers is a single fork. Eating requires both adjacent forks. Design a protocol that prevents deadlock and starvation.

### Dijkstra's Semaphore Solution

Assign each fork a semaphore $f_i$ initialized to 1. Philosopher $i$ picks up fork $\min(i, (i+1) \bmod 5)$ first, then $\max(i, (i+1) \bmod 5)$.

```
semaphore fork[5] = {1, 1, 1, 1, 1};

philosopher(i):
    loop:
        think()
        low  = min(i, (i+1) % 5)
        high = max(i, (i+1) % 5)
        wait(fork[low])
        wait(fork[high])
        eat()
        signal(fork[high])
        signal(fork[low])
```

**Why it works.** The resource ordering (always acquire lower-numbered fork first) breaks the circular wait condition. At least one philosopher's acquisition order differs from the cycle, preventing deadlock.

**Proof sketch.** Suppose deadlock occurs. Then each philosopher $i$ holds one fork and waits for another. Since each acquires $\min(i, (i+1) \bmod 5)$ first, philosopher $4$ acquires fork $0$ before fork $4$. But philosopher $3$ acquires fork $3$ before fork $4$. For circular wait, we need $4 \to 0 \to 1 \to 2 \to 3 \to 4$, but philosopher $4$ holds fork $0$ (not fork $4$), breaking the cycle.

### Alternative: Limit Concurrency

Allow at most $n - 1 = 4$ philosophers to sit simultaneously:

```
semaphore seats = 4;
semaphore fork[5] = {1, 1, 1, 1, 1};

philosopher(i):
    loop:
        think()
        wait(seats)
        wait(fork[i])
        wait(fork[(i+1) % 5])
        eat()
        signal(fork[(i+1) % 5])
        signal(fork[i])
        signal(seats)
```

**Proof.** With at most 4 philosophers and 5 forks, by pigeonhole at least one philosopher can acquire both forks. That philosopher eats, releases forks, enabling progress. No deadlock is possible.

### Alternative: Chandy-Misra Solution

For the general case with arbitrary resource graphs. Each fork is **dirty** or **clean**. A philosopher holding a dirty fork must give it up when requested. This achieves both deadlock freedom and starvation freedom without a global ordering.

### Why It Matters

The Dining Philosophers problem, introduced by Dijkstra in 1965, is the canonical illustration of deadlock and resource contention. Its solutions map directly to real systems: resource ordering is used in database lock managers, concurrency limiting is used in connection pools, and the Chandy-Misra approach underlies distributed mutual exclusion.

---

## 8. Microkernel vs. Monolithic Kernel: The Tanenbaum-Torvalds Debate

### The Problem

Analyze the fundamental architectural tradeoff between microkernels (minimal kernel with user-space servers) and monolithic kernels (all OS services in kernel space).

### Microkernel Architecture

The kernel provides only:
1. **Inter-process communication (IPC)** -- message passing
2. **Address space management** -- page tables, MMU configuration
3. **Thread scheduling** -- basic CPU multiplexing

Everything else runs as user-space servers: file systems, device drivers, network stacks, memory managers.

**Formal IPC cost model.** A microkernel system call that would be a single kernel function call in a monolithic kernel requires:

$$t_{\mu\text{kernel}} = t_{\text{syscall}} + t_{\text{marshal}} + t_{\text{ipc}} + t_{\text{context\_switch}} + t_{\text{unmarshal}} + t_{\text{server}} + t_{\text{return\_path}}$$

versus:

$$t_{\text{monolithic}} = t_{\text{syscall}} + t_{\text{function\_call}} + t_{\text{return}}$$

The IPC overhead is typically $2\text{--}10\times$ per operation. L4 microkernels reduced IPC to $\sim 1\,\mu\text{s}$ (Liedtke, 1993), demonstrating that the overhead is an engineering problem, not a fundamental one.

### The Debate (1992)

Andrew Tanenbaum argued:

1. **Modularity.** User-space servers can crash without bringing down the kernel. Fault isolation improves reliability.
2. **Portability.** Minimal hardware-dependent code in the kernel.
3. **Security.** Smaller trusted computing base (TCB). The kernel is small enough to formally verify (seL4 achieved this in 2009: 8,700 lines of C, machine-checked proof of functional correctness).
4. **Historical trajectory.** "Linux is obsolete" -- monolithic kernels are a step backward from the research direction.

Linus Torvalds argued:

1. **Performance.** IPC overhead for every system service interaction is unacceptable for a general-purpose OS.
2. **Pragmatism.** A working monolithic kernel today beats a theoretically superior microkernel that does not exist in production.
3. **Complexity migration.** Microkernels do not eliminate complexity; they move it to user space where debugging is harder.
4. **Optimization.** Tight coupling between subsystems enables optimizations (e.g., zero-copy, shared data structures) impossible across process boundaries.

### Quantitative Comparison

| Property | Microkernel | Monolithic |
|----------|------------|------------|
| Kernel size (LOC) | 10K--50K | 1M--30M |
| IPC overhead | $1\text{--}10\,\mu\text{s}$ | N/A (function call) |
| Driver fault isolation | Full (user space) | None (kernel panic) |
| Formal verification | Feasible (seL4) | Infeasible at scale |
| Context switches/syscall | $2\text{--}4\times$ more | Baseline |
| Real-world examples | QNX, seL4, Minix 3, Fuchsia | Linux, FreeBSD, Windows NT* |

*Windows NT is a hybrid: microkernel-inspired but with major subsystems (graphics, windowing) in kernel space for performance.

### Modern Resolution

The debate has largely converged on **hybrid approaches**:

- Linux uses **loadable kernel modules** (LKMs) for modularity without IPC overhead.
- Linux's **eBPF** provides safe, verified code execution in kernel space -- achieving some microkernel safety guarantees within a monolithic architecture.
- **Unikernels** (MirageOS, Unikernel Linux) collapse the distinction entirely by linking application and kernel into a single address space.
- **Fuchsia** (Google) uses the Zircon microkernel, betting that modern hardware makes IPC overhead negligible.

The theoretical ideal (small, verified kernel with fault-isolated services) remains correct; the engineering question is whether the IPC tax is acceptable for a given workload.

### Why It Matters

This debate shaped 30 years of OS design. Tanenbaum was right about the direction (modularity, verification, fault isolation) but wrong about the timeline. Torvalds was right about the pragmatics (Linux dominates servers, embedded, mobile, supercomputers). The synthesis -- monolithic kernels with microkernel-inspired isolation mechanisms -- is the current state of the art.
