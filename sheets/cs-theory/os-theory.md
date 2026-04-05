# Operating System Theory (Processes, Memory, Scheduling, Deadlock)

A complete reference for operating system internals — process management, CPU scheduling, memory management, virtual memory, deadlock, file systems, I/O scheduling, and inter-process communication.

## Process Management

### Process States

```
                  admit              dispatch
  New ----------> Ready ----------> Running
                    ^                  |  |
                    |    interrupt     |  |
                    +------------------+  |
                    |                     |
                    |   I/O complete      | I/O or event wait
                    +--- Waiting <--------+
                                          |
                                          | exit
                                          v
                                      Terminated

Five-state model:
  New        — process created, not yet admitted
  Ready      — waiting to be assigned to CPU
  Running    — instructions being executed
  Waiting    — waiting for I/O or event
  Terminated — finished execution
```

### Process Control Block (PCB)

```
PCB contents:
  Process ID (PID)          — unique identifier
  Process state             — new/ready/running/waiting/terminated
  Program counter           — address of next instruction
  CPU registers             — contents of all process-centric registers
  CPU scheduling info       — priority, queue pointers
  Memory management info    — page tables, segment tables, base/limit registers
  Accounting info           — CPU time used, time limits, job/process numbers
  I/O status info           — list of open files, allocated I/O devices
```

### Context Switch

```
Context switch steps:
  1. Save state of currently running process (registers, PC) into its PCB
  2. Update process state (running -> ready or waiting)
  3. Move PCB to appropriate queue
  4. Select next process from ready queue (scheduler)
  5. Update state of selected process (ready -> running)
  6. Restore saved state from new process's PCB
  7. Flush TLB and pipeline (architecture-dependent)

Cost: pure overhead — system does no useful work during switch
Typical time: 1-10 microseconds (hardware-dependent)
```

## CPU Scheduling Algorithms

### First-Come, First-Served (FCFS)

```
Non-preemptive. Processes served in arrival order.

Example:
  Process   Burst Time   Arrival
  P1        24           0
  P2        3            0
  P3        3            0

Gantt chart:  |--- P1 (24) ---|-- P2 (3) --|-- P3 (3) --|
Wait times:   P1=0, P2=24, P3=27
Average wait: (0 + 24 + 27) / 3 = 17.0

Convoy effect: short processes stuck behind long one
```

### Shortest Job First (SJF)

```
Non-preemptive. Select process with smallest burst time.
Provably optimal for minimizing average waiting time (non-preemptive case).

Example:
  Process   Burst Time
  P1        6
  P2        8
  P3        7
  P4        3

Order: P4(3), P1(6), P3(7), P2(8)
Wait:  P4=0, P1=3, P3=9, P2=16
Avg:   (0 + 3 + 9 + 16) / 4 = 7.0

Problem: requires knowing burst time in advance
Approximation: exponential averaging
  tau_(n+1) = alpha * t_n + (1 - alpha) * tau_n
  alpha = 0 => recent history ignored
  alpha = 1 => only last burst matters
```

### Shortest Remaining Time First (SRTF)

```
Preemptive version of SJF.
When new process arrives, compare its burst to remaining time of current process.
Preempt if new process has shorter remaining time.

Optimal for minimizing average waiting time (preemptive case).
Higher overhead due to frequent context switches.
```

### Round Robin (RR)

```
Preemptive. Each process gets a time quantum q.
After q time units, process is preempted and placed at end of ready queue.

Performance depends on q:
  q too large => degenerates to FCFS
  q too small => too many context switches (overhead dominates)
  Rule of thumb: 80% of CPU bursts should be shorter than q

Example (q = 4):
  Process   Burst
  P1        24
  P2        3
  P3        3

  |P1(4)|P2(3)|P3(3)|P1(4)|P1(4)|P1(4)|P1(4)|P1(4)|
  Wait: P1=6, P2=4, P3=7   Avg = 5.67

Higher average turnaround than SJF, but better response time.
```

### Priority Scheduling

```
Each process assigned a priority. CPU allocated to highest-priority process.
Can be preemptive or non-preemptive.

Problem: starvation — low-priority processes may never execute
Solution: aging — gradually increase priority of waiting processes

Internal priority: set by OS (time limits, memory, I/O usage)
External priority: set by user/admin (importance, payment)
```

### Multi-Level Feedback Queue (MLFQ)

```
Multiple queues with different priorities and scheduling algorithms.

Rules:
  1. If priority(A) > priority(B), A runs
  2. If priority(A) = priority(B), A and B run in RR
  3. New processes start at highest priority
  4. If a process uses its entire time slice, move it down one level
  5. If a process gives up CPU before time slice expires, stay at same level
  6. Periodically boost all processes to top queue (prevent starvation)

Typical configuration:
  Queue 0 (highest): RR with q = 8ms
  Queue 1:           RR with q = 16ms
  Queue 2 (lowest):  FCFS

Approximates SJF without requiring burst time knowledge.
```

### Completely Fair Scheduler (CFS)

```
Linux default scheduler (since 2.6.23).

Key concept: virtual runtime (vruntime)
  vruntime += delta_exec * (NICE_0_WEIGHT / weight)

  Processes with lower weight accumulate vruntime faster.
  Scheduler always picks process with smallest vruntime.

Data structure: red-black tree keyed by vruntime
  - Leftmost node = smallest vruntime = next to run
  - O(log n) insert/delete
  - O(1) pick-next (cached leftmost)

Target latency: total period in which every runnable task runs at least once
  min_granularity: minimum time slice (prevents thrashing with many tasks)

  time_slice = target_latency * (weight / total_weight)
```

## Memory Management

### Paging

```
Physical memory divided into fixed-size frames.
Logical memory divided into same-size pages.
Page size typically 4 KB (some systems support 2 MB or 1 GB huge pages).

Address translation:
  Logical address = (page_number, page_offset)
  page_number     = address / page_size
  page_offset     = address % page_size

  Physical address = frame_table[page_number] * page_size + page_offset

No external fragmentation.
Internal fragmentation: on average, half a page per process.
```

### Multi-Level Page Tables

```
Problem: flat page table for 32-bit address space with 4 KB pages = 2^20 entries.
         At 4 bytes each = 4 MB per process.

Solution: hierarchical page tables.

Two-level (x86-32):
  10 bits: page directory index
  10 bits: page table index
  12 bits: page offset

  CR3 -> Page Directory -> Page Table -> Physical Frame

Four-level (x86-64, 48-bit virtual):
  9 bits: PML4 index
  9 bits: PDP index
  9 bits: PD index
  9 bits: PT index
  12 bits: page offset

Only allocate page table entries that are actually used.
```

### Translation Lookaside Buffer (TLB)

```
Hardware cache for page table entries.
Fully associative or set-associative.
Typical: 64-1024 entries, hit time < 1 cycle.

Effective Access Time (EAT):
  EAT = hit_rate * (TLB_time + memory_time) + (1 - hit_rate) * (TLB_time + n * memory_time)
  where n = number of page table levels + 1

Example (single-level, TLB hit rate = 98%, memory = 100ns, TLB = 1ns):
  EAT = 0.98 * (1 + 100) + 0.02 * (1 + 200)
      = 0.98 * 101 + 0.02 * 201
      = 98.98 + 4.02
      = 103 ns

  Without TLB: 200 ns. TLB cuts access time nearly in half.
```

### Segmentation

```
Logical address space divided into variable-size segments:
  code, data, stack, heap, etc.

Logical address = (segment_number, offset)
Segment table maps segment_number -> (base, limit)

  if offset < limit:
      physical_address = base + offset
  else:
      segmentation fault

Advantage: matches programmer's view of memory.
Disadvantage: external fragmentation (variable-size allocation).

Can be combined with paging (segmented paging): each segment paged independently.
```

### Page Replacement Algorithms

```
When a page fault occurs and no free frames exist, evict a page.

FIFO (First-In, First-Out):
  Replace the oldest page in memory.
  Simple but suffers from Belady's anomaly.

LRU (Least Recently Used):
  Replace the page that has not been used for the longest time.
  Good performance but expensive to implement exactly.
  Approximations: clock algorithm, reference bits.

Clock Algorithm (Second Chance):
  Pages arranged in circular buffer with reference bit.
  On replacement:
    1. Check reference bit of page at clock hand
    2. If bit = 1, clear it, advance hand, repeat
    3. If bit = 0, replace this page

Optimal (Belady's Algorithm / OPT):
  Replace the page that will not be used for the longest time in the future.
  Not implementable (requires future knowledge).
  Used as a benchmark for comparison.
```

### Belady's Anomaly

```
FIFO can produce more page faults with more frames.

Classic example with reference string: 1 2 3 4 1 2 5 1 2 3 4 5

  3 frames: 9 page faults
  4 frames: 10 page faults (!)

Stack algorithms (LRU, OPT) do not exhibit this anomaly.
A page replacement algorithm is a stack algorithm if:
  S_n(t) is a subset of S_(n+1)(t) for all t
  where S_k(t) = set of pages in memory with k frames at time t
```

### Thrashing and Working Set

```
Thrashing: process spends more time paging than executing.
Occurs when sum of working set sizes > available physical memory.

Working set model:
  W(t, Delta) = set of pages referenced in interval [t - Delta, t]
  |W(t, Delta)| = working set size

  If total demand = sum of |W_i| for all processes i
  and total demand > available frames
  then thrashing occurs.

Solution: suspend processes until demand fits in memory.

Page Fault Frequency (PFF):
  Set upper and lower bounds on acceptable fault rate.
  If rate > upper bound: allocate more frames.
  If rate < lower bound: reclaim frames.
```

## Virtual Memory

### Demand Paging

```
Load pages only when needed (lazy loading).
Initially: no pages in memory, every reference causes a page fault.

Page fault handling:
  1. Check internal table (in PCB) — is reference valid?
  2. If invalid, terminate process. If valid but not in memory, continue.
  3. Find a free frame (or evict via replacement algorithm).
  4. Schedule disk read of needed page into free frame.
  5. Update page table entry (set valid bit).
  6. Restart the instruction that caused the fault.

Performance:
  Effective access time = (1 - p) * memory_access + p * page_fault_time
  where p = page fault rate

  If memory_access = 200 ns, page_fault_time = 8 ms:
  For < 10% degradation: p < 0.0000025 (less than 1 fault per 400,000 accesses)
```

### Copy-on-Write (COW)

```
After fork(), parent and child share same physical pages.
Pages marked read-only. On write attempt:
  1. Page fault triggered
  2. OS copies the page to a new frame
  3. Update page table of writing process to point to new frame
  4. Mark both copies as writable

Benefit: fork() is nearly free if child calls exec() immediately.
Used extensively in Unix/Linux. Enables efficient process creation.
```

### Memory-Mapped Files

```
Map file contents directly into process virtual address space.
  mmap(addr, length, prot, flags, fd, offset)

File I/O becomes memory read/write — no explicit read()/write() system calls.
Pages loaded on demand (page faults bring in file data).
Multiple processes can map the same file (shared memory via file backing).

Advantages:
  - Simpler programming model
  - Kernel can optimize I/O (read-ahead, lazy writeback)
  - Zero-copy sharing between processes
```

## Deadlock

### Coffman Conditions

```
All four must hold simultaneously for deadlock to occur:

  1. Mutual exclusion    — at least one resource held in non-sharable mode
  2. Hold and wait       — process holds resource(s) while waiting for others
  3. No preemption       — resources cannot be forcibly taken from a process
  4. Circular wait       — circular chain of processes, each waiting for
                           a resource held by the next process in the chain

Break any one condition to prevent deadlock.
```

### Resource Allocation Graph

```
Directed graph:
  Process nodes: circles (P1, P2, ...)
  Resource nodes: rectangles (R1, R2, ...)
  Request edge:  Pi -> Rj (process Pi requests resource Rj)
  Assignment edge: Rj -> Pi (resource Rj assigned to process Pi)

If graph has no cycle: no deadlock.
If graph has cycle:
  - Single instance per resource type: deadlock exists.
  - Multiple instances: deadlock may or may not exist.
```

### Wait-For Graph

```
Simplified version of resource allocation graph.
Only process nodes. Edge Pi -> Pj means Pi is waiting for a resource held by Pj.

Deadlock exists if and only if the wait-for graph contains a cycle.

Maintained by the OS. Periodically check for cycles.
Cycle detection: O(n^2) where n = number of processes.
```

### Banker's Algorithm

```
Deadlock avoidance. Determines if granting a request leads to a safe state.

Data structures (n processes, m resource types):
  Available[m]    — available instances of each resource type
  Max[n][m]       — maximum demand of each process
  Allocation[n][m] — currently allocated to each process
  Need[n][m]      — remaining need: Need[i][j] = Max[i][j] - Allocation[i][j]

Safety algorithm:
  1. Work = Available; Finish[i] = false for all i
  2. Find i such that Finish[i] = false and Need[i] <= Work
  3. Work = Work + Allocation[i]; Finish[i] = true; go to step 2
  4. If all Finish[i] = true, system is in a safe state

Request algorithm:
  1. If Request[i] <= Need[i], go to 2. Else error.
  2. If Request[i] <= Available, go to 3. Else wait.
  3. Pretend to allocate: Available -= Request[i]; Allocation[i] += Request[i]; Need[i] -= Request[i]
  4. Run safety algorithm. If safe, grant. Else restore and wait.
```

## File Systems

### Inode-Based File Systems (Unix/ext)

```
Inode: fixed-size data structure storing file metadata.

  Inode contents:
    File type, permissions, owner, group
    File size, timestamps (atime, mtime, ctime)
    Link count (hard links)
    Direct block pointers (typically 12)
    Single indirect pointer   -> block of pointers
    Double indirect pointer   -> block of pointers to blocks of pointers
    Triple indirect pointer   -> three levels of indirection

  With 4 KB blocks and 4-byte pointers (1024 pointers per block):
    Direct:          12 * 4 KB = 48 KB
    Single indirect: 1024 * 4 KB = 4 MB
    Double indirect: 1024^2 * 4 KB = 4 GB
    Triple indirect: 1024^3 * 4 KB = 4 TB

  Directory: file containing list of (name, inode_number) pairs.
```

### Journaling File Systems

```
Problem: crash during multi-step file operation leaves inconsistent state.

Solution: write-ahead log (journal).
  1. Write intended changes to journal (log)
  2. Perform actual changes to file system
  3. Mark journal entry as complete

Recovery: replay incomplete journal entries on mount.

Modes:
  Journal (full):    log metadata + data (safest, slowest)
  Ordered (default): log metadata, write data before metadata commit
  Writeback:         log metadata only (fastest, data can be lost)

Examples: ext3, ext4, NTFS, XFS, JFS
```

### Log-Structured File Systems (LFS)

```
All writes are sequential (append to log).
Optimized for write-heavy workloads and SSDs.

Segments: large contiguous regions written sequentially.
Inode map: maps inode number to current inode location.
Garbage collection: reclaim space from obsolete segments.

Trade-off: fast writes, slower random reads (must consult inode map).
Examples: F2FS, NILFS2, original Sprite LFS
```

### B-Tree File Systems

```
Use B-tree (or B+ tree) variants for all metadata.
Single data structure for directories, extents, free space.

Advantages:
  - Efficient for large directories
  - Copy-on-write enables snapshots and checksums
  - Self-healing with redundant metadata

Examples: Btrfs, ZFS (uses modified B-tree / DMU), APFS, ReFS
```

## I/O Scheduling

### Disk Scheduling Algorithms

```
Goal: minimize seek time (head movement).

SCAN (Elevator):
  Head moves in one direction servicing requests, then reverses.
  Provides uniform wait time.

C-SCAN (Circular SCAN):
  Head moves in one direction only. After reaching end, jumps to beginning.
  More uniform wait time than SCAN.

LOOK / C-LOOK:
  Like SCAN/C-SCAN but head only goes as far as the last request
  in each direction (does not go to disk edge).

Note: with SSDs (no seek time), I/O scheduling is less critical.
Modern schedulers: noop (SSD), mq-deadline, BFQ, Kyber.
```

## Inter-Process Communication (IPC)

### Pipes

```
Unidirectional byte stream between processes.

Anonymous pipes: pipe(fd[2])
  fd[0] = read end, fd[1] = write end
  Parent-child communication only.

Named pipes (FIFOs): mkfifo(path)
  File-system visible. Any process can open.
  Unidirectional. Half-duplex.

Pipe capacity: typically 64 KB (Linux). Writes > PIPE_BUF (4 KB) not atomic.
```

### Message Queues

```
Structured messages (with type/priority) stored in kernel.

POSIX: mq_open, mq_send, mq_receive
System V: msgget, msgsnd, msgrcv

Advantages over pipes:
  - Message boundaries preserved
  - Priority-based delivery
  - Multiple readers/writers
  - Persistent (outlive creating process)
```

### Shared Memory

```
Fastest IPC — no kernel involvement after setup.

POSIX: shm_open, mmap
System V: shmget, shmat, shmdt

Requires synchronization (semaphores, mutexes).
  Producer writes to shared region, consumer reads.
  Must coordinate access to avoid races.

Typical use: high-throughput data exchange (multimedia, databases).
```

### Signals

```
Asynchronous notification sent to a process.

Common signals:
  SIGINT  (2)  — interrupt (Ctrl+C)
  SIGKILL (9)  — kill (cannot be caught)
  SIGSEGV (11) — segmentation fault
  SIGTERM (15) — termination request
  SIGCHLD (17) — child process status changed
  SIGSTOP (19) — stop process (cannot be caught)

Handling: signal(sig, handler) or sigaction(sig, &act, NULL)
Signals can interrupt system calls (EINTR).
```

## Key Figures

```
Edsger Dijkstra    — THE multiprogramming system, semaphores, dining philosophers,
                     banker's algorithm, structured programming
Andrew Tanenbaum   — MINIX, "Operating Systems: Design and Implementation",
                     microkernel advocacy, Tanenbaum-Torvalds debate
Abraham Silberschatz — "Operating System Concepts" (the dinosaur book),
                     standard OS textbook for decades
Dennis Ritchie     — co-creator of Unix, C programming language
Ken Thompson       — co-creator of Unix, B language, Plan 9, UTF-8
Linus Torvalds     — Linux kernel, monolithic kernel design, Git
Per Brinch Hansen  — concurrent programming, monitors, RC 4000 nucleus
Butler Lampson     — Alto OS, hints for computer system design
```

## See Also

- Turing Machines
- Algorithms
- Computer Networking
- Systems Programming
- Concurrency

## References

```
Silberschatz, Galvin, Gagne. "Operating System Concepts" (10th ed., 2018)
Tanenbaum, Bos. "Modern Operating Systems" (4th ed., 2014)
Arpaci-Dusseau, Arpaci-Dusseau. "Operating Systems: Three Easy Pieces" (2018, free online)
  https://pages.cs.wisc.edu/~remzi/OSTEP/
Dijkstra. "The Structure of the THE Multiprogramming System" (1968)
Ritchie, Thompson. "The UNIX Time-Sharing System" (1974)
Love. "Linux Kernel Development" (3rd ed., 2010)
Corbato et al. "An Experimental Time-Sharing System" (1962)
Linux kernel source: https://github.com/torvalds/linux
```
