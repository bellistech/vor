# The Internals of Go — Scheduler, GC, and Channels

> *Go's runtime is a sophisticated piece of systems software. The goroutine scheduler uses a work-stealing algorithm (GMP model), the garbage collector uses tricolor concurrent marking with a write barrier, and channels are implemented with lock-protected ring buffers and wait queues.*

---

## 1. The Goroutine Scheduler — GMP Model

### Three Entities

| Entity | Symbol | What It Is | Count |
|:-------|:------:|:-----------|:------|
| Goroutine | **G** | Lightweight thread (2KB initial stack) | Millions possible |
| OS Thread | **M** | Kernel thread (1-8MB stack) | Bounded by `GOMAXPROCS` active |
| Processor | **P** | Logical CPU context with run queue | Exactly `GOMAXPROCS` |

### The Invariant

An M can only execute a G when it holds a P. The relationship:

$$\text{Active goroutines} \leq |P| = \text{GOMAXPROCS}$$

Each P maintains a **local run queue** (LRQ) of up to 256 Gs. There is also a **global run queue** (GRQ) as overflow.

### Scheduling Algorithm

```
1. Check local run queue (LRQ) of current P
2. If empty, check global run queue (GRQ)
3. If empty, check network poller
4. If empty, STEAL from another P's LRQ (take half)
5. If nothing to steal, park the M
```

### Work Stealing

When P's LRQ is empty, it steals from a random other P:

$$\text{stolen} = \lfloor \text{victim.LRQ.len} / 2 \rfloor$$

This ensures load balancing with $O(1)$ amortized scheduling cost. The steal attempt cycles through all Ps before parking.

### Goroutine State Machine

```
         ┌──────────┐
    ┌───►│ Runnable  │◄───────────┐
    │    └─────┬─────┘            │
    │          │ scheduled        │ unblocked
    │          ▼                  │
    │    ┌──────────┐       ┌─────┴─────┐
    │    │ Running  ├──────►│  Waiting   │
    │    └─────┬─────┘ I/O, │  (blocked) │
    │          │      chan,  └───────────┘
    │          │      lock
    │          ▼
    │    ┌──────────┐
    └────┤  Preempt │  (async preemption via signal)
         └──────────┘
```

### Preemption

Go 1.14+ uses **asynchronous preemption**: the runtime sends a signal (`SIGURG`) to preempt long-running goroutines that don't hit safe points. Before 1.14, goroutines could only be preempted at function calls (cooperative).

### Stack Growth

Goroutines start with a **2KB stack** that grows dynamically (up to 1GB default). Growth uses **copy-on-grow**: allocate larger stack, copy contents, update all pointers.

$$\text{new\_size} = 2 \times \text{old\_size}$$

Stack shrink happens during GC if stack is less than 25% utilized.

---

## 2. Garbage Collector — Tricolor Concurrent Mark-Sweep

### Tricolor Abstraction

Every object is colored:

| Color | Meaning |
|:------|:--------|
| **White** | Not yet seen (candidates for collection) |
| **Grey** | Seen but children not yet scanned |
| **Black** | Scanned, all children are grey or black |

### The Algorithm

```
Phase 1: Mark Setup     (STW — stop the world, ~10-30 us)
  - Enable write barrier
  - Turn all roots grey

Phase 2: Concurrent Mark (runs alongside mutators)
  - Pick grey object → scan its pointers
  - Children become grey, object becomes black
  - Repeat until no grey objects remain

Phase 3: Mark Termination (STW — ~10-30 us)
  - Drain remaining work
  - Disable write barrier

Phase 4: Concurrent Sweep (runs alongside mutators)
  - Free white (unreachable) objects
```

### The Tricolor Invariant

**No black object may point to a white object.** If violated, the white object would be collected while still reachable.

The **write barrier** maintains this invariant. When a pointer write occurs during marking:

```go
// Dijkstra write barrier (simplified)
func writePointer(slot *unsafe.Pointer, ptr unsafe.Pointer) {
    shade(ptr)   // mark the new pointee grey
    *slot = ptr
}
```

Go uses a **hybrid write barrier** (Go 1.17+): combination of Dijkstra insertion barrier and Yuasa deletion barrier, eliminating the need to rescan stacks.

### GC Pacer Formula

The pacer decides when to trigger GC:

$$\text{GC\_trigger} = \text{live\_heap} \times \left(1 + \frac{\text{GOGC}}{100}\right)$$

Where:
- $\text{live\_heap}$ = bytes of live (reachable) data after last GC
- $\text{GOGC}$ = GC percentage (default: 100)

### Worked Examples

| Live Heap | GOGC | Trigger Point | Heap Growth Allowed |
|:---:|:---:|:---:|:---:|
| 10 MB | 100 | 20 MB | 10 MB (100%) |
| 10 MB | 50 | 15 MB | 5 MB (50%) |
| 10 MB | 200 | 30 MB | 20 MB (200%) |
| 100 MB | 100 | 200 MB | 100 MB |

### GC CPU Target

The GC aims to use **25% of CPU** during the mark phase. With `GOMAXPROCS=4`, one P is dedicated to GC. If the mutator allocates faster than GC can mark, the pacer engages **mark assist**: goroutines that allocate must help mark.

$$\text{assist\_ratio} = \frac{\text{scan\_work\_remaining}}{\text{heap\_distance\_to\_goal}}$$

### Memory Limit (Go 1.19+)

`GOMEMLIMIT` sets a soft heap limit. When set, the GC will run more frequently to stay under the limit:

$$\text{effective\_GOGC} = \min\left(\text{GOGC}, \frac{\text{GOMEMLIMIT} - \text{non\_heap}}{\text{live\_heap}} - 1\right) \times 100$$

---

## 3. Channel Implementation

### Buffered Channels — Ring Buffer

A buffered channel `make(chan T, n)` is backed by a **circular buffer**:

```
struct hchan {
    qcount   uint   // current number of elements
    dataqsiz uint   // buffer capacity (n)
    buf      *T     // ring buffer of size n
    sendx    uint   // send index (write position)
    recvx    uint   // receive index (read position)
    lock     mutex  // protects all fields
    recvq    waitq  // list of waiting receivers
    sendq    waitq  // list of waiting senders
}
```

### Send Operation

```
chan <- value:
  1. Acquire lock
  2. If a receiver is waiting in recvq:
     - Copy value directly to receiver's stack (no buffer touch)
     - Wake receiver
  3. Else if buffer not full (qcount < dataqsiz):
     - Copy value to buf[sendx]
     - sendx = (sendx + 1) % dataqsiz
     - qcount++
  4. Else (buffer full):
     - Create sudog, enqueue in sendq
     - Park goroutine (gopark)
  5. Release lock
```

### Unbuffered Channels

Unbuffered channels (`make(chan T)`) have `dataqsiz = 0`. Every send blocks until a receiver arrives (and vice versa). The value is copied **directly from sender's stack to receiver's stack** — the channel is just a rendezvous point.

### Select Statement

`select` on multiple channels:
1. Lock all channels (in address order to prevent deadlock)
2. Check each case for readiness
3. If one ready: execute it
4. If multiple ready: **pseudo-random choice** (uniform)
5. If none ready: enqueue current G in all channels' wait queues, park
6. When woken: dequeue from all other channels

---

## 4. Interface Implementation — Fat Pointers

An interface value is a pair:

$$\text{interface} = (\text{type pointer}, \text{data pointer})$$

```
struct iface {           struct eface {          // empty interface (any)
    tab  *itab              _type *_type
    data unsafe.Pointer     data  unsafe.Pointer
}                        }
```

The `itab` contains method dispatch table (like a vtable). Method lookup is $O(1)$ after the first call — itabs are cached in a hash table keyed by `(interface type, concrete type)`.

### Interface Satisfaction

Go uses **structural typing**: a type satisfies an interface if it implements all methods. No explicit `implements` declaration. This is checked at compile time for static types, at runtime for type assertions.

---

## 5. Memory Allocator — mcache/mcentral/mheap

### Three-Level Hierarchy

```
Per-P mcache ──► mcentral (per size class, locked) ──► mheap (global, locked)
```

Objects are classified into **68 size classes** (8B to 32KB). Objects > 32KB are allocated directly from the heap.

| Size | Allocation Path | Lock Contention |
|:-----|:---------------|:----------------|
| Tiny (< 16B, no pointers) | mcache tiny allocator | None (per-P) |
| Small (16B - 32KB) | mcache → mcentral → mheap | Minimal (per-P fast path) |
| Large (> 32KB) | mheap directly | Global lock |

### Span-Based Allocation

Memory is managed in **spans** (multiples of 8KB pages). Each span holds objects of one size class:

$$\text{objects\_per\_span} = \lfloor \text{span\_pages} \times 8192 / \text{size\_class} \rfloor$$

---

## 6. Escape Analysis

The compiler decides whether a variable lives on the **stack** (cheap) or **heap** (requires GC):

```
go build -gcflags="-m" ./...
```

### Common Escape Reasons

| Pattern | Escapes? | Reason |
|:--------|:---------|:-------|
| Return pointer to local | Yes | Outlives stack frame |
| Store in interface | Yes | Type not known at compile time |
| Closure captures variable | Yes (usually) | Closure outlives scope |
| Slice backing array grows | Yes | `append` may reallocate |
| Fixed-size local array | No | Size known, doesn't escape |

---

## 7. Summary of Key Formulas

| Concept | Formula | Domain |
|:--------|:--------|:-------|
| GC trigger | $\text{live} \times (1 + \text{GOGC}/100)$ | Memory management |
| Work stealing | $\lfloor \text{victim.len} / 2 \rfloor$ | Scheduling |
| Stack growth | $\text{new} = 2 \times \text{old}$ | Stack management |
| Ring buffer index | $(i + 1) \mod n$ | Channel implementation |
| Active goroutines | $\leq \text{GOMAXPROCS}$ | Concurrency bound |
| Size classes | 68 classes, 8B to 32KB | Memory allocator |
| GC CPU target | 25% of available CPU | GC pacer |

---

*The Go runtime is not magic — it's a carefully engineered scheduler, allocator, and garbage collector that trades some raw performance for safety and simplicity. Knowing these internals is what separates writing Go from understanding Go.*

## Prerequisites

- Concurrency concepts (goroutines, threads, mutexes, channels)
- Garbage collection basics (mark-and-sweep, generational GC)
- Memory layout (stack vs heap, pointer indirection, escape analysis)
- Interface-based polymorphism and structural typing
