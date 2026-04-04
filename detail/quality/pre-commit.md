# The Mathematics of pre-commit -- Hook Scheduling and Defect Economics

> *A pre-commit hook is an automated quality gate positioned at the cheapest possible point in the defect lifecycle. The mathematics of hook orchestration involves pipeline scheduling, environment caching, and the economics of shifting defect detection left.*

---

## 1. Shift-Left Economics (Cost of Defects)

### The Problem

Defects caught later in the development lifecycle cost exponentially more to fix. Pre-commit hooks shift detection to the earliest possible moment -- before the commit enters the repository. The economic argument for hooks is the cost differential.

### The Formula

The cost to fix a defect discovered at stage $s$:

$$C(s) = C_0 \times k^s$$

where $C_0$ is the base cost (developer minutes to fix at coding time), $k$ is the escalation factor (typically 3-10x), and $s$ is the stage index.

| Stage $s$ | Phase | $C(s)$ at $k=5$ |
|:---:|:---|:---:|
| 0 | Pre-commit (local) | $C_0$ |
| 1 | CI pipeline | $5 C_0$ |
| 2 | Code review | $25 C_0$ |
| 3 | QA/staging | $125 C_0$ |
| 4 | Production | $625 C_0$ |

### Worked Examples

A trailing whitespace issue takes 10 seconds to fix locally ($C_0 = 10$s).

Caught at pre-commit: $C = 10$ s (auto-fixed).

Caught in CI: $C = 5 \times 10 = 50$ s (wait for CI, read log, fix, push).

Caught in code review: $C = 25 \times 10 = 250$ s (reviewer comments, context switch, fix, re-review).

For a team producing 20 fixable issues per day:

$$\text{Annual savings (pre-commit vs. CI)} = 20 \times 250 \times (5 - 1) \times 40 = 200{,}000 \text{ s} \approx 55 \text{ hours/year}$$

---

## 2. Hook Execution Pipeline (Parallel Scheduling)

### The Problem

pre-commit runs hooks sequentially by default, but hooks operating on different file types could theoretically run in parallel. Understanding the critical path determines the total hook execution time.

### The Formula

For $n$ hooks with execution times $t_1, t_2, \ldots, t_n$ running sequentially:

$$T_{sequential} = \sum_{i=1}^{n} t_i$$

With parallel execution on non-overlapping file sets, the time is bounded by the critical path:

$$T_{parallel} = \max_{p \in \text{paths}} \sum_{i \in p} t_i$$

The speedup from parallelization:

$$\text{Speedup} = \frac{T_{sequential}}{T_{parallel}}$$

### Worked Examples

Five hooks:
- `trailing-whitespace` (0.3s, all files)
- `black` (1.2s, Python files)
- `eslint` (2.0s, JS files)
- `go-fmt` (0.5s, Go files)
- `shellcheck` (0.8s, Shell files)

**Sequential:**
$$T_{sequential} = 0.3 + 1.2 + 2.0 + 0.5 + 0.8 = 4.8 \text{ s}$$

**Parallel** (trailing-whitespace must run first, then language hooks can run simultaneously):
$$T_{parallel} = 0.3 + \max(1.2, 2.0, 0.5, 0.8) = 0.3 + 2.0 = 2.3 \text{ s}$$

$$\text{Speedup} = \frac{4.8}{2.3} = 2.1\times$$

Note: pre-commit currently runs hooks sequentially. The parallel analysis shows the theoretical limit. The `require_serial` flag exists for hooks that must not overlap.

---

## 3. Environment Cache Hit Rates (Amortized Setup)

### The Problem

pre-commit creates isolated environments for each hook (virtualenvs, node_modules, etc.). First run is expensive; subsequent runs hit the cache. The amortized cost depends on the cache lifetime.

### The Formula

For a hook environment with setup cost $S$ and per-run cost $R$, the amortized cost over $n$ runs before cache invalidation:

$$C_{amortized} = \frac{S}{n} + R$$

Cache invalidation occurs when `rev` changes in `.pre-commit-config.yaml`. With autoupdate frequency $f$ (updates per month) and $d$ commits per day:

$$n = \frac{30 \times d}{f}$$

### Worked Examples

A Python hook environment: $S = 15$ s (pip install), $R = 1.2$ s (lint run). Team averages 10 commits/day, monthly autoupdate ($f = 1$):

$$n = \frac{30 \times 10}{1} = 300 \text{ runs between invalidations}$$

$$C_{amortized} = \frac{15}{300} + 1.2 = 0.05 + 1.2 = 1.25 \text{ s}$$

The setup cost is effectively invisible. But with weekly autoupdate ($f = 4$):

$$n = \frac{30 \times 10}{4} = 75$$

$$C_{amortized} = \frac{15}{75} + 1.2 = 0.20 + 1.2 = 1.40 \text{ s}$$

Still negligible. Setup cost only dominates on the first run or in CI without caching.

---

## 4. File Matching Efficiency (Regex Filtering)

### The Problem

Each hook specifies a `files` regex and/or `types` filter. The efficiency of file matching determines how many files are passed to each hook. Over-broad matching wastes time; over-narrow matching misses files.

### The Formula

For a commit touching $N$ files and a hook with file pattern selectivity $\sigma$ (fraction of files matching):

$$F_{matched} = \sigma \times N$$

Hook execution time scales with matched files (for per-file hooks):

$$T_{hook} = T_{base} + F_{matched} \times T_{per\_file}$$

The optimal selectivity balances coverage and speed:

$$\sigma_{optimal} = \frac{F_{relevant}}{N}$$

where $F_{relevant}$ is the number of files the hook should actually check.

### Worked Examples

A commit touches 50 files. An ESLint hook matches `\.jsx?$|\.tsx?$`:

- 15 JS/TS files, 20 Python files, 10 Go files, 5 shell scripts
- $\sigma = 15/50 = 0.30$
- $F_{matched} = 15$

If the pattern were overly broad (`.*`):
- $\sigma = 1.0$, $F_{matched} = 50$
- ESLint processes 35 irrelevant files, erroring or wasting time

Time impact ($T_{per\_file} = 0.2$s):

$$T_{correct} = 0.5 + 15 \times 0.2 = 3.5 \text{ s}$$
$$T_{broad} = 0.5 + 50 \times 0.2 = 10.5 \text{ s}$$

$$\text{Overhead} = \frac{10.5 - 3.5}{3.5} = 200\%$$

---

## 5. Autoupdate and Version Drift (Dependency Freshness)

### The Problem

`pre-commit autoupdate` bumps hook versions. Stale hooks miss new rules and bug fixes. The "freshness" of the hook configuration degrades over time, accumulating a delta between current and latest versions.

### The Formula

The version drift for hook $i$ at time $t$ since last update:

$$\Delta v_i(t) = \int_0^t r_i(\tau) \, d\tau$$

where $r_i$ is the release rate (versions per unit time). For constant release rate:

$$\Delta v_i(t) = r_i \times t$$

The cumulative drift across $n$ hooks:

$$D(t) = \sum_{i=1}^{n} \Delta v_i(t) = t \sum_{i=1}^{n} r_i$$

### Worked Examples

Configuration with 8 hooks, average release rate of 0.5 versions/month each:

After 3 months without autoupdate:

$$D(3) = 3 \times 8 \times 0.5 = 12 \text{ version-months of drift}$$

Each version-month introduces roughly 2 new rules or bug fixes:

$$\text{Missed improvements} = 12 \times 2 = 24$$

With monthly autoupdate ($t_{max} = 1$):

$$D_{max} = 1 \times 8 \times 0.5 = 4 \text{ version-months}$$
$$\text{Missed improvements}_{max} = 8$$

Reduction: $1 - 8/24 = 66.7\%$ fewer missed improvements.

---

## 6. Stage Placement Optimization (Gate Theory)

### The Problem

Hooks can run at different stages: `commit`, `push`, `commit-msg`, `manual`. Fast hooks belong in `commit` (runs every commit); slow hooks belong in `push` (runs less often). The question is: which stage minimizes total developer wait time?

### The Formula

Developer wait time for a hook in stage $s$:

$$W(s) = f_s \times T_{hook}$$

where $f_s$ is the frequency of that stage trigger:

| Stage | $f$ (triggers/day) |
|:---|:---:|
| commit | 15-30 |
| push | 3-5 |
| manual | 0-1 |

Total daily wait across all hooks:

$$W_{total} = \sum_{h \in \text{hooks}} f_{stage(h)} \times T_h$$

### Worked Examples

Moving a 10-second test suite from `commit` to `push` stage:

$$W_{commit} = 20 \times 10 = 200 \text{ s/day}$$
$$W_{push} = 4 \times 10 = 40 \text{ s/day}$$
$$\text{Savings} = 200 - 40 = 160 \text{ s/day}$$

For a team of 5 developers over a year:

$$\text{Annual savings} = 160 \times 5 \times 250 = 200{,}000 \text{ s} \approx 55 \text{ hours}$$

The break-even: hooks under 0.5s are fine in `commit`. Hooks over 5s should move to `push`. Hooks over 30s should be `manual` or CI-only.

$$T_{threshold} = \frac{W_{acceptable}}{f_{commit}} = \frac{10}{20} = 0.5 \text{ s}$$

---

## Prerequisites

- Software engineering economics (cost of defects, shift-left testing)
- Pipeline scheduling (critical path, parallel execution)
- Amortization and cache theory
- Regular expressions (file matching patterns)
- Queuing theory (service rates, wait times)
- Basic calculus (integration for continuous rates)
