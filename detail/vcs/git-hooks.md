# The Mathematics of Git Hooks -- Pipeline Composition and CI Gate Probability

> *Git hooks form a sequential pipeline of validation gates. Each hook is a predicate function that must return true (exit 0) for the operation to proceed. The mathematics of hook execution involve ordered pipeline composition, conditional probability of pipeline success, and the cost-benefit analysis of shifting checks left in the development workflow.*

---

## 1. Hook Execution Model (Pipeline Algebra)

### Hooks as Predicate Functions

Each hook $h_i$ is a function mapping the repository state $S$ to a boolean:

$$h_i : S \to \{0, 1\}$$

where $0$ = pass (exit code 0) and $1$ = fail (exit code non-zero).

### Pipeline Composition

A git operation (e.g., commit) executes hooks in a fixed order. The operation succeeds only if all hooks pass:

$$\text{commit}(S) = h_{\text{pre-commit}}(S) \wedge h_{\text{prepare-commit-msg}}(S) \wedge h_{\text{commit-msg}}(S)$$

For push:

$$\text{push}(S) = h_{\text{pre-push}}(S) \wedge h_{\text{pre-receive}}(S) \wedge h_{\text{update}}(S)$$

### Short-Circuit Evaluation

Git evaluates hooks left to right with short-circuit semantics:

$$\text{Pipeline}(h_1, h_2, \ldots, h_n) = \begin{cases} \text{FAIL at } h_k & \text{if } h_k(S) = 1 \text{ and } \forall j < k: h_j(S) = 0 \\ \text{PASS} & \text{if } \forall i: h_i(S) = 0 \end{cases}$$

The expected number of hooks executed before failure:

$$E[\text{hooks executed}] = \sum_{k=1}^{n} k \cdot P(\text{fail at } h_k) + n \cdot P(\text{all pass})$$

---

## 2. Hook Ordering and Dependency Graph (Partial Orders)

### Client-Side Hook Execution Order

The commit lifecycle defines a total order:

```
pre-commit → prepare-commit-msg → commit-msg → post-commit
```

| Phase | Hook | Can Abort | Typical Duration |
|:---:|:---|:---:|:---:|
| 1 | pre-commit | Yes | 1-30s |
| 2 | prepare-commit-msg | Yes | <100ms |
| 3 | commit-msg | Yes | <100ms |
| 4 | post-commit | No | <1s |

### Push Hook Chain

```
client: pre-push → [network] → server: pre-receive → update (per ref) → post-receive
```

| Hook | Runs On | Can Abort | Scope |
|:---|:---:|:---:|:---|
| pre-push | Client | Yes | All refs in push |
| pre-receive | Server | Yes | All refs (single invocation) |
| update | Server | Yes | Per ref (can reject individual refs) |
| post-receive | Server | No | All refs |
| post-update | Server | No | All refs (legacy) |

### Partial Order Constraints

Hooks within a phase are independent and could theoretically run in parallel:

$$h_{\text{lint}} \parallel h_{\text{format}} \parallel h_{\text{secrets}} \quad \text{(all in pre-commit phase)}$$

Tools like `pre-commit` and `lefthook` exploit this with `parallel: true`, reducing wall-clock time:

$$T_{\text{sequential}} = \sum_{i=1}^{n} t_i \qquad T_{\text{parallel}} = \max_{i=1}^{n} t_i$$

Speedup factor:

$$\text{Speedup} = \frac{\sum t_i}{\max t_i}$$

---

## 3. CI Gate Probability (Defect Detection)

### Single Gate Pass Rate

Let $p_i$ be the probability that a given change passes hook $h_i$:

$$P(\text{pipeline passes}) = \prod_{i=1}^{n} p_i$$

For a pipeline with 4 hooks each passing 95% of the time:

$$P(\text{all pass}) = 0.95^4 = 0.8145$$

### Defect Detection Rate

If hook $h_i$ catches defect class $D_i$ with sensitivity $s_i$:

$$P(\text{defect escapes}) = \prod_{i=1}^{n} (1 - s_i)$$

| Hook | Defect Class | Sensitivity $s_i$ | Escape Rate |
|:---|:---|:---:|:---:|
| pre-commit (lint) | Style violations | 0.99 | 0.01 |
| pre-commit (secrets) | Credential leaks | 0.85 | 0.15 |
| commit-msg | Bad commit messages | 0.95 | 0.05 |
| pre-push (tests) | Regressions | 0.90 | 0.10 |

Combined escape rate for a defect that all hooks could catch:

$$P(\text{escape}) = 0.01 \times 0.15 \times 0.05 \times 0.10 = 7.5 \times 10^{-6}$$

### Layered Defense (Swiss Cheese Model)

Adding redundant checks across stages:

$$P(\text{defect reaches prod}) = \prod_{j=1}^{m} (1 - d_j)$$

where $d_j$ is the detection probability at layer $j$:

| Layer | Detection Rate $d_j$ | Cumulative Escape |
|:---:|:---|:---:|
| 1 | pre-commit hooks: 0.85 | 0.15 |
| 2 | pre-push tests: 0.90 | 0.015 |
| 3 | CI pipeline: 0.95 | 0.00075 |
| 4 | Code review: 0.80 | 0.00015 |
| 5 | Staging tests: 0.70 | 0.000045 |

---

## 4. Shift-Left Economics (Cost Analysis)

### Cost of Defect by Stage

The cost to fix a defect increases exponentially as it progresses through stages:

$$C_{\text{fix}}(stage) \approx C_0 \cdot k^{stage}$$

where $C_0$ is the base cost and $k \approx 5\text{-}10$.

| Stage | Multiplier | Example Cost | Detection Method |
|:---:|:---:|:---:|:---|
| 0 (pre-commit) | $1 \times$ | $0.10 | Automated hook |
| 1 (pre-push) | $5 \times$ | $0.50 | Local tests |
| 2 (CI) | $25 \times$ | $2.50 | Pipeline |
| 3 (Code review) | $50 \times$ | $5.00 | Human review |
| 4 (Staging) | $100 \times$ | $10.00 | QA testing |
| 5 (Production) | $500 \times$ | $50.00 | Customer report |

### Expected Cost per Commit

$$E[\text{cost}] = \sum_{s=0}^{5} P(\text{detected at stage } s) \cdot C_{\text{fix}}(s)$$

Without pre-commit hooks (defects caught at CI):

$$E[\text{cost}] = 0.80 \times 2.50 + 0.15 \times 5.00 + 0.05 \times 50.00 = 2.00 + 0.75 + 2.50 = \$5.25$$

With pre-commit hooks:

$$E[\text{cost}] = 0.85 \times 0.10 + 0.10 \times 2.50 + 0.04 \times 5.00 + 0.01 \times 50.00 = 0.085 + 0.25 + 0.20 + 0.50 = \$1.035$$

---

## 5. Hook Execution Time Budget (Latency)

### Developer Tolerance Model

Developer patience follows a decay function:

$$P(\text{bypass}) = 1 - e^{-\lambda t}$$

where $t$ is hook execution time in seconds and $\lambda$ is the impatience factor.

| Hook Duration | $P(\text{bypass with --no-verify})$ | Developer Reaction |
|:---:|:---:|:---|
| 1s | 0.02 | Acceptable |
| 5s | 0.10 | Noticeable |
| 15s | 0.30 | Annoying |
| 30s | 0.55 | Frequently skipped |
| 60s | 0.80 | Almost always skipped |

### Optimal Time Budget

The optimal hook time $t^*$ minimizes total cost (hook overhead + escaped defects):

$$\text{Total Cost} = \underbrace{N \cdot t}_{\text{developer time}} + \underbrace{N \cdot P(\text{bypass}(t)) \cdot C_{\text{escape}}}_{\text{escaped defect cost}}$$

Taking the derivative and setting to zero gives the optimal hook execution time as a function of team size $N$, bypass probability, and escape cost.

For most teams: $t^* \approx 5\text{-}10$ seconds.

---

## 6. Hook Composition Patterns (Functional)

### Map-Filter-Reduce over Staged Files

Pre-commit hooks typically follow a functional pattern:

$$\text{result} = \text{reduce}(\wedge, \text{map}(h, \text{filter}(\text{glob}, \text{staged})))$$

```
staged_files → filter(*.go) → map(gofmt) → reduce(AND) → pass/fail
```

### Hook Multiplexing

When $m$ hooks run in the pre-commit phase with parallel execution:

$$T_{\text{wall}} = \max(t_1, t_2, \ldots, t_m) + T_{\text{overhead}}$$

$$T_{\text{overhead}} \approx m \times T_{\text{fork}} \approx m \times 5\text{ms}$$

For 6 parallel hooks averaging 2s each with max 8s:

$$T_{\text{sequential}} = 6 \times 2 = 12\text{s} \qquad T_{\text{parallel}} = 8 + 0.03 = 8.03\text{s}$$

---

## 7. Server-Side Hook Security (Trust Boundaries)

### Client vs Server Trust Model

| Property | Client Hooks | Server Hooks |
|:---|:---:|:---:|
| Bypassable | Yes (`--no-verify`) | No |
| Controlled by | Developer | Repository admin |
| Trust level | Advisory | Enforced |
| Scope | Local repo | All pushes |

### Required server-side checks

Any policy that must be enforced (not just encouraged) belongs in server-side hooks:

$$\text{Enforced Policy} \implies \text{pre-receive or update hook}$$

$$\text{Developer Convenience} \implies \text{pre-commit hook}$$

The security boundary is the network: anything running on the client is advisory.

---

## Prerequisites

- Boolean algebra, predicate logic, pipeline composition
- Probability theory (independent events, conditional probability)
- Cost-benefit analysis, expected value calculations
- Graph theory (partial orders, DAGs)

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Hook dispatch (per hook) | $O(1)$ | $O(1)$ |
| Staged file enumeration | $O(n)$ files | $O(n)$ |
| Pattern matching (glob filter) | $O(n \cdot m)$ patterns | $O(n)$ |
| Parallel hook execution | $O(\max(t_i))$ wall | $O(m)$ processes |
| Sequential pipeline | $O(\sum t_i)$ wall | $O(1)$ |

---

*The mathematics of git hooks reduce to pipeline reliability engineering. Each hook is a gate with a pass probability, and the pipeline's value comes from the multiplicative reduction in defect escape rate. The key insight is economic: a $0.10 fix at pre-commit beats a $50.00 fix in production by 500x. The constraint is human patience -- hooks must run fast enough that developers don't bypass them, making the optimal time budget approximately 5-10 seconds.*
