# The Mathematics of Jest -- Mock Theory and Test Isolation

> *Every mock is a lie you tell your test suite -- a controlled lie that isolates behavior from dependencies. The mathematics of mocking concerns information hiding, call graph partitioning, and the tradeoffs between test speed and fidelity.*

---

## 1. Test Isolation as Graph Partitioning (Dependency Graphs)

### The Problem

A unit test should test one unit. But code has dependencies, forming a directed acyclic graph (DAG). Mocking cuts edges in this graph to isolate the node under test. The question is: which edges to cut?

### The Formula

Given a call graph $G = (V, E)$ where $V$ is the set of modules and $E$ is the set of call edges, testing module $v$ requires mocking the set $M_v \subseteq \text{neighbors}(v)$:

$$\text{Isolation}(v) = \frac{|M_v|}{|\text{out-degree}(v)|}$$

Full isolation ($\text{Isolation} = 1$) means every dependency is mocked. Partial isolation means some real implementations remain.

### Worked Examples

Module `OrderService` depends on `Database`, `PaymentGateway`, `EmailSender`, and `Logger`:

$$|\text{out-degree}| = 4$$

Mocking `Database`, `PaymentGateway`, and `EmailSender` but keeping `Logger` real:

$$\text{Isolation} = \frac{3}{4} = 0.75$$

The total mocking burden across a test suite with $n$ modules:

$$M_{total} = \sum_{v \in V} |M_v|$$

For 50 modules averaging 3 mocks each: $M_{total} = 150$ mock configurations to maintain.

---

## 2. Mock Return Value Combinatorics (State Space)

### The Problem

A mock with $k$ methods, each returning one of $r$ possible values, creates a state space of test scenarios. Understanding this space helps you choose which scenarios to test.

### The Formula

The total state space for a mock object with $k$ methods and $r_i$ possible return values for method $i$:

$$S = \prod_{i=1}^{k} r_i$$

With `mockReturnValueOnce` chains of length $l$, the sequenced state space grows:

$$S_{sequenced} = \prod_{i=1}^{k} r_i^{l_i}$$

### Worked Examples

A mocked API client has 3 methods: `getUser` (returns user or null = 2), `getPosts` (returns array or throws = 2), `getComments` (returns array, empty array, or throws = 3):

$$S = 2 \times 2 \times 3 = 12 \text{ scenarios}$$

If `getUser` is chained with 3 sequential return values:

$$S_{sequenced} = 2^3 \times 2 \times 3 = 48 \text{ scenarios}$$

Covering all 48 is unnecessary. Boundary analysis reduces to:

$$S_{boundary} \approx \sum_{i=1}^{k} r_i = 2 + 2 + 3 = 7 \text{ key scenarios}$$

---

## 3. Snapshot Entropy (Change Detection)

### The Problem

Snapshot tests serialize component output and compare it to a stored reference. Large snapshots have high entropy -- small code changes produce large diffs, causing developers to blindly update snapshots without reviewing changes.

### The Formula

The information content of a snapshot with $n$ characters drawn from alphabet $\Sigma$:

$$H = -\sum_{c \in \Sigma} p(c) \log_2 p(c)$$

The probability of a "meaningful review" decreases with snapshot size:

$$P(\text{review}) \approx \frac{1}{1 + e^{\alpha(n - n_0)}}$$

This logistic decay models that developers review small snapshots carefully but rubber-stamp large ones. $n_0$ is the threshold (typically around 50 lines), and $\alpha$ controls the steepness.

### Worked Examples

A component snapshot of 20 lines:

$$P(\text{review}) = \frac{1}{1 + e^{0.1(20 - 50)}} = \frac{1}{1 + e^{-3}} = \frac{1}{1 + 0.05} \approx 0.95$$

A component snapshot of 200 lines:

$$P(\text{review}) = \frac{1}{1 + e^{0.1(200 - 50)}} = \frac{1}{1 + e^{15}} \approx \frac{1}{3{,}269{,}017} \approx 0$$

This is why Jest recommends inline snapshots for small outputs and targeted assertions for complex components.

---

## 4. Fake Timer Precision (Temporal Testing)

### The Problem

`jest.useFakeTimers()` replaces the system clock, allowing deterministic testing of time-dependent code. The precision of timer advancement determines whether race conditions and edge cases are caught.

### The Formula

For a function with $n$ scheduled callbacks at times $t_1, t_2, \ldots, t_n$, advancing by $\Delta t$ fires all callbacks where:

$$\{i \mid t_i \leq t_{current} + \Delta t\}$$

The minimum distinguishing advancement to test ordering between callbacks $j$ and $k$:

$$\Delta t_{min} = |t_j - t_k|$$

### Worked Examples

A debounce function with 300ms delay and a polling function at 1000ms intervals:

$$t_{debounce} = 300, \quad t_{poll} = 1000$$

To test that debounce fires before the first poll:

$$\Delta t = 300 \text{ ms fires debounce only}$$
$$\Delta t = 1000 \text{ ms fires both}$$

Minimum advancement to distinguish them:

$$\Delta t_{min} = |1000 - 300| = 700 \text{ ms}$$

Testing the edge case where debounce resets:

$$\text{Advance } 299\text{ms} \rightarrow \text{trigger input} \rightarrow \text{advance } 300\text{ms}$$

Total elapsed: $599$ms, but debounce fires at $299 + 300 = 599$ms from start, not $300$ms. The reset extends the effective delay.

---

## 5. Coverage Confidence Intervals (Statistical Testing)

### The Problem

Jest's `--coverage` reports a point estimate. But how confident should you be that the reported coverage reflects the actual defect-detection capability? Coverage is a proxy metric, not a guarantee.

### The Formula

Treating each line as a Bernoulli trial (covered or not), the confidence interval for true coverage $p$ given $n$ lines and $k$ covered:

$$\hat{p} = \frac{k}{n}, \quad \text{CI}_{95\%} = \hat{p} \pm 1.96\sqrt{\frac{\hat{p}(1-\hat{p})}{n}}$$

The defect detection probability for a bug uniformly distributed across lines:

$$P(\text{detect}) = \frac{|E|}{|L|} = \hat{p}$$

For $m$ independent bugs:

$$P(\text{detect all } m) = \hat{p}^m$$

### Worked Examples

A module with 500 lines and 450 covered ($\hat{p} = 0.90$):

$$\text{CI}_{95\%} = 0.90 \pm 1.96\sqrt{\frac{0.90 \times 0.10}{500}} = 0.90 \pm 0.026$$

So true coverage is between $87.4\%$ and $92.6\%$ with 95% confidence.

Probability of detecting all 3 independent bugs:

$$P(\text{detect all 3}) = 0.90^3 = 0.729$$

There is a $27.1\%$ chance of missing at least one bug despite 90% coverage.

---

## 6. Worker Parallelism and Memory (Jest Workers)

### The Problem

Jest spawns worker processes to run test files in parallel. Each worker loads the full Node.js runtime and module graph. Memory usage scales linearly with workers, while speedup follows diminishing returns.

### The Formula

Total memory consumption:

$$M_{total} = M_{main} + P \times M_{worker}$$

where $M_{main}$ is the coordinator overhead and $M_{worker}$ is per-worker memory (typically 100-300 MB for a React project).

The optimal worker count balances speedup against available memory:

$$P_{optimal} = \min\left(\left\lfloor\frac{M_{available} - M_{main}}{M_{worker}}\right\rfloor, \, \text{CPU cores}\right)$$

### Worked Examples

CI runner with 8 GB RAM, $M_{main} = 200$ MB, $M_{worker} = 350$ MB:

$$P_{optimal} = \min\left(\left\lfloor\frac{8000 - 200}{350}\right\rfloor, 4\right) = \min(22, 4) = 4$$

Memory-constrained (2 GB runner):

$$P_{optimal} = \min\left(\left\lfloor\frac{2000 - 200}{350}\right\rfloor, 4\right) = \min(5, 4) = 4$$

But actual usage: $200 + 4 \times 350 = 1600$ MB, leaving only 400 MB for OS and other processes. Safer choice: `--maxWorkers=2` using $200 + 700 = 900$ MB.

---

## Prerequisites

- JavaScript closures and the module system (CommonJS/ESM)
- Graph theory basics (DAG, in-degree, out-degree)
- Combinatorics (Cartesian product, counting)
- Information theory (entropy, information content)
- Basic statistics (confidence intervals, Bernoulli trials)
- Process memory model (heap, workers, IPC)
