# The Mathematics of pytest -- Fixture Scope and Test Combinatorics

> *A test suite is a sampling problem: you want maximum defect coverage from minimum test execution. pytest's parametrize, fixture scoping, and marker system give you precise control over how the combinatorial space of inputs maps to actual test runs.*

---

## 1. Parametric Combinatorics (Test Explosion)

### The Problem

When a function accepts multiple parameters each with several valid values, the number of test cases grows as the Cartesian product. Exhaustive testing quickly becomes infeasible, so you need strategies to sample the space efficiently.

### The Formula

For $k$ parameters with $n_1, n_2, \ldots, n_k$ values respectively, the total number of test combinations is:

$$T_{exhaustive} = \prod_{i=1}^{k} n_i$$

With pytest's stacked `@pytest.mark.parametrize` decorators, each decorator multiplies the test count.

### Worked Examples

A login endpoint accepts `username` (3 types), `password` (4 types), and `mfa_token` (2 types):

$$T = 3 \times 4 \times 2 = 24 \text{ test cases}$$

Adding a fifth value to passwords:

$$T' = 3 \times 5 \times 2 = 30$$

The marginal cost of one new password variant is:

$$\Delta T = 30 - 24 = 6 \text{ additional tests}$$

In general, adding one value to parameter $j$:

$$\Delta T = \prod_{i \neq j} n_i$$

### Pairwise Reduction

Full Cartesian product is often overkill. Pairwise (2-way covering array) testing ensures every pair of parameter values appears in at least one test. The lower bound on the number of tests:

$$T_{pairwise} \geq \max_{i} n_i \times \max_{j \neq i} n_j$$

For the login example:

$$T_{pairwise} \geq 4 \times 3 = 12 \text{ tests (vs. 24 exhaustive)}$$

Savings ratio:

$$\text{Savings} = 1 - \frac{T_{pairwise}}{T_{exhaustive}} = 1 - \frac{12}{24} = 50\%$$

---

## 2. Fixture Scope Cost Model (Setup Amortization)

### The Problem

Expensive fixtures (database connections, Docker containers) should be created once and shared. pytest's scope hierarchy controls how many times a fixture is instantiated. Choosing the wrong scope wastes time or introduces shared-state bugs.

### The Formula

Let $S$ be the setup cost (seconds), $D$ the teardown cost, and $N$ the number of tests using the fixture. The total fixture overhead is:

$$C_{total} = I \times (S + D)$$

where $I$ is the number of instantiations, determined by scope:

| Scope | $I$ |
|:---|:---|
| `function` | $N$ |
| `class` | $\lceil N / \bar{c} \rceil$ (classes using it) |
| `module` | $\lceil N / \bar{m} \rceil$ (modules using it) |
| `session` | $1$ |

### Worked Examples

A PostgreSQL fixture takes $S = 2.0$s to start and $D = 0.5$s to stop. 200 tests use it.

**Function scope:**
$$C_{function} = 200 \times (2.0 + 0.5) = 500 \text{ s}$$

**Module scope (10 modules):**
$$C_{module} = 10 \times 2.5 = 25 \text{ s}$$

**Session scope:**
$$C_{session} = 1 \times 2.5 = 2.5 \text{ s}$$

Speedup from function to session scope:

$$\text{Speedup} = \frac{C_{function}}{C_{session}} = \frac{500}{2.5} = 200\times$$

The tradeoff: session-scoped fixtures share mutable state across all tests, risking order-dependent failures.

---

## 3. Parallel Execution Speedup (pytest-xdist)

### The Problem

Test suites grow linearly with codebase size, but developer patience does not. Distributing tests across $P$ workers reduces wall-clock time, but overhead from process spawning and I/O contention limits the speedup.

### The Formula

Applying Amdahl's Law to test parallelization:

$$\text{Speedup}(P) = \frac{1}{(1 - f) + \frac{f}{P}}$$

where $f$ is the fraction of test time that can run in parallel and $P$ is the number of workers. The sequential fraction $(1-f)$ includes fixture setup, collection, and reporting.

### Worked Examples

A 600-second test suite has 30 seconds of serial overhead (collection, session fixtures, reporting), so $f = \frac{570}{600} = 0.95$.

With 8 workers:

$$\text{Speedup}(8) = \frac{1}{0.05 + \frac{0.95}{8}} = \frac{1}{0.05 + 0.119} = \frac{1}{0.169} \approx 5.9\times$$

$$T_{parallel} = \frac{600}{5.9} \approx 102 \text{ s}$$

Theoretical maximum (infinite workers):

$$\text{Speedup}(\infty) = \frac{1}{1 - f} = \frac{1}{0.05} = 20\times$$

So the serial overhead caps the best possible time at $30$ seconds regardless of core count.

---

## 4. Coverage as Set Theory (pytest-cov)

### The Problem

Code coverage measures which statements are executed during testing. But 100% line coverage does not mean 100% of behaviors are tested. Understanding coverage as set membership clarifies its limits.

### The Formula

Let $L$ be the set of all executable lines, and $E$ the set of lines executed by the test suite:

$$\text{Coverage} = \frac{|E|}{|L|} \times 100\%$$

Branch coverage extends this. For $B$ total branches and $B_t$ branches taken:

$$\text{Branch Coverage} = \frac{|B_t|}{|B|} \times 100\%$$

Line coverage is a necessary but not sufficient condition for branch coverage:

$$\text{Branch Coverage} \leq \text{Line Coverage}$$

is false in general, but typically branch coverage $<$ line coverage because a single line can contain multiple branches (e.g., ternary operators, short-circuit evaluation).

### Worked Examples

A module has 400 lines, 20 branch points (each with true/false = 40 branches). Tests execute 380 lines and 30 branches:

$$\text{Line Coverage} = \frac{380}{400} = 95\%$$

$$\text{Branch Coverage} = \frac{30}{40} = 75\%$$

The 20 untaken branches represent code paths that might harbor bugs despite high line coverage. To reach 90% branch coverage, you need:

$$B_{needed} = 0.9 \times 40 - 30 = 6 \text{ additional branches}$$

---

## 5. Marker-Based Test Selection (Set Algebra)

### The Problem

pytest markers let you tag tests and select subsets with `-m` expressions. The selection language supports AND, OR, and NOT, forming a Boolean algebra over test sets.

### The Formula

Let $M_i$ denote the set of tests with marker $i$. The `-m` expression maps to set operations:

$$\text{-m "A and B"} \Rightarrow M_A \cap M_B$$

$$\text{-m "A or B"} \Rightarrow M_A \cup M_B$$

$$\text{-m "not A"} \Rightarrow U \setminus M_A$$

$$\text{-m "A and not B"} \Rightarrow M_A \setminus M_B$$

where $U$ is the universe of all collected tests.

### Worked Examples

Given $|U| = 500$, $|M_{slow}| = 80$, $|M_{integration}| = 120$, $|M_{slow} \cap M_{integration}| = 30$:

**Fast unit tests only:**
$$|\text{-m "not slow and not integration"}| = 500 - |M_{slow} \cup M_{integration}|$$
$$= 500 - (80 + 120 - 30) = 500 - 170 = 330$$

**CI smoke suite (integration but fast):**
$$|\text{-m "integration and not slow"}| = 120 - 30 = 90$$

---

## 6. Expected Failure Analysis (xfail)

### The Problem

`@pytest.mark.xfail` documents known failures. Tracking the ratio of expected failures to total tests quantifies technical debt in the test suite.

### The Formula

$$\text{Debt Ratio} = \frac{|X_{fail}|}{|U|} \times 100\%$$

The "surprise pass" rate (xfail tests that now pass, indicating fixed bugs):

$$\text{Surprise Rate} = \frac{|X_{pass}|}{|X_{fail}|} \times 100\%$$

### Worked Examples

A suite of 1000 tests has 45 xfail markers. After a refactor, 12 now pass unexpectedly:

$$\text{Debt Ratio} = \frac{45}{1000} = 4.5\%$$

$$\text{Surprise Rate} = \frac{12}{45} = 26.7\%$$

With `strict=True`, those 12 surprise passes become failures, forcing you to remove the xfail marker -- keeping the suite honest.

---

## Prerequisites

- Python functions and decorators
- Basic set theory (union, intersection, complement)
- Amdahl's Law and parallel speedup limits
- Combinatorics (Cartesian product, counting principles)
- Basic probability and sampling theory
