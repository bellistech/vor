# The Mathematics of Concurrent Web Crawling -- Graph Traversal Under Bounded Parallelism

> *A concurrent web crawler is a parallel BFS on a directed graph with a semaphore-bounded frontier -- where correctness demands atomic visited-set operations and termination requires counting outstanding work to zero.*

---

## 1. Graph Model and Domain Filtering (Graph Theory)

### The Problem

Model the web as a directed graph and define the crawl as a reachability query restricted
to a single domain.

### The Formula

Let $G = (V, E)$ be a directed graph where $V$ is the set of all URLs and
$E = \{(u, v) \mid v \in \text{getUrls}(u)\}$.

Given a start vertex $s \in V$ and a domain predicate $D(v)$, the crawl result is:

$$R = \{v \in V \mid v \text{ is reachable from } s \text{ in } G_D\}$$

where $G_D = (V_D, E_D)$ is the subgraph induced by $V_D = \{v \in V \mid D(v)\}$.

The domain predicate extracts the hostname: $D(v) = (\text{hostname}(v) = \text{hostname}(s))$.

### Worked Examples

Start URL: `http://example.com/`. Graph edges:

| From | To | Same domain? |
|------|----|:---:|
| `/` | `/page1` | Yes |
| `/` | `/page2` | Yes |
| `/` | `http://other.com/x` | No (filtered) |
| `/page1` | `/` | Yes (cycle, already visited) |
| `/page1` | `/page3` | Yes |

Reachable set $R = \{/, /\text{page1}, /\text{page2}, /\text{page3}\}$, $|R| = 4$.

---

## 2. The Semaphore as a Counting Resource (Concurrency Theory)

### The Problem

Formalize the concurrency bound as a counting semaphore and prove that at most $N$
tasks execute the fetch section simultaneously.

### The Formula

A counting semaphore $S$ with initial value $N$ provides two operations:

$$\text{acquire}(S): \text{wait until } S > 0, \text{ then } S \gets S - 1$$

$$\text{release}(S): S \gets S + 1$$

**Invariant:** At any point in time, the number of tasks between acquire and release is:

$$\text{active} = N - S \le N$$

Since $S \ge 0$ by the wait condition, $\text{active} \le N$.

In Go, a buffered channel of capacity $N$ acts as a semaphore:
- `sem <- struct{}{}` is acquire (blocks when buffer full)
- `<-sem` is release

### Worked Examples

$N = 3$, 5 URLs discovered simultaneously:

| Time | Action | Channel buffer | Active fetches |
|------|--------|:-:|:-:|
| $t_0$ | URL1 acquires | `[_, _, _]` $\to$ `[x, _, _]` | 1 |
| $t_1$ | URL2 acquires | `[x, x, _]` | 2 |
| $t_2$ | URL3 acquires | `[x, x, x]` | 3 |
| $t_3$ | URL4 tries acquire | BLOCKS | 3 |
| $t_4$ | URL1 releases | `[x, x, _]` | 2 |
| $t_5$ | URL4 unblocks, acquires | `[x, x, x]` | 3 |

Maximum concurrent fetches never exceeds 3.

---

## 3. Atomic Visited-Set and the Check-Then-Act Race (Distributed Systems)

### The Problem

Why must the visited check and insert be atomic? What race condition occurs otherwise?

### The Formula

The visited set requires a compound atomic operation: **test-and-set**.

Non-atomic (racy) pattern:

$$\text{Thread A: if } v \notin \text{visited} \quad\text{(true)}$$
$$\text{Thread B: if } v \notin \text{visited} \quad\text{(true, interleaved)}$$
$$\text{Thread A: visited.add}(v); \text{spawn}(v)$$
$$\text{Thread B: visited.add}(v); \text{spawn}(v) \quad\text{DUPLICATE!}$$

Atomic pattern using `LoadOrStore` (Go) or equivalent:

$$\text{atomically: if } v \notin \text{visited}, \text{ then visited.add}(v), \text{ return } \texttt{new}$$

This guarantees exactly one thread gets `new = true` for each URL.

### Worked Examples

Using `sync.Map.LoadOrStore("http://example.com/page1", true)`:

- Thread A calls LoadOrStore: key absent, stores `true`, returns `loaded=false` -- spawns task
- Thread B calls LoadOrStore: key present, returns `loaded=true` -- skips

Result: exactly one task for `/page1`.

---

## 4. Termination Detection via Work Counting (Parallel Algorithms)

### The Problem

How does the crawler know all work is complete? Define a termination condition.

### The Formula

Maintain a counter $W$ (outstanding work units):

$$W_0 = 1 \quad\text{(the start URL)}$$

Before spawning a new task: $W \gets W + 1$ (`wg.Add(1)`)

When a task completes: $W \gets W - 1$ (`wg.Done()`)

**Termination condition:** $W = 0$

This is precisely the semantics of `sync.WaitGroup` in Go. The `wg.Wait()` call blocks
until $W$ reaches zero.

**Correctness:** $W = 0$ implies no task is running and no task will be spawned (since
spawning requires an active task to discover new URLs). Therefore all reachable URLs
have been processed.

### Worked Examples

Crawl of 4-page site:

| Event | $W$ |
|-------|:---:|
| Start: spawn root | 1 |
| Root discovers page1, page2: spawn both | 3 |
| Root completes | 2 |
| Page1 discovers page3: spawn | 3 |
| Page2 completes (no new URLs) | 2 |
| Page1 completes | 1 |
| Page3 completes (page1 already visited) | 0 |

$W = 0$ -- crawler terminates. Total pages: 4.

---

## 5. BFS vs DFS Traversal Order (Algorithm Design)

### The Problem

Does the concurrent crawler perform BFS or DFS? Does it matter for correctness?

### The Formula

With unbounded goroutines/tasks, the traversal order is **nondeterministic** -- it
depends on the scheduler. The Go implementation spawns goroutines eagerly (DFS-like
recursion), while Python's `asyncio.gather` waits for all children (BFS-like levels).

**Correctness is order-independent:** The result set $R$ depends only on reachability
in $G_D$, not on traversal order. Both BFS and DFS (and any interleaving) discover
the same set of vertices.

**Performance differs:** BFS discovers all pages at distance $d$ before $d+1$, giving
breadth-first latency. DFS may chase deep chains before exploring siblings. For web
crawling, BFS is typically preferred to stay "closer" to the start page.

The final result is sorted alphabetically regardless of traversal order.

### Worked Examples

For a balanced binary tree of depth 3 (7 pages):
- BFS order: root, L, R, LL, LR, RL, RR (level by level)
- DFS order: root, L, LL, LR, R, RL, RR (depth first)
- Concurrent: any interleaving, but sorted output is identical

---

## Prerequisites

- Graph traversal (BFS, DFS)
- Concurrency primitives (mutexes, semaphores, channels)
- URL parsing and hostname extraction
- Producer-consumer patterns

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Implement a single-threaded BFS crawler with a visited set. Add domain filtering. Verify cycle handling with circular link structures. |
| **Intermediate** | Add bounded concurrency using a semaphore. Use atomic visited-set operations to prevent duplicate work. Implement in multiple languages with their idiomatic concurrency primitives. |
| **Advanced** | Analyze throughput vs. concurrency level. Implement politeness delays (per-domain rate limiting). Handle DNS resolution, redirects, and error retry with exponential backoff. Profile memory usage as the visited set grows. |
