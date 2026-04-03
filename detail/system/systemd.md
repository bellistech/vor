# The Mathematics of systemd — Dependency Graphs, Boot Ordering & Resource Control

> *systemd models the boot process as a directed acyclic graph. Every unit is a node, every dependency an edge, and boot ordering is a topological sort problem from graph theory.*

---

## 1. Boot Ordering as Topological Sort

### The Dependency Graph

systemd units form a **DAG** (Directed Acyclic Graph) where edges represent ordering constraints:

- `After=B` means A's node has an incoming edge from B
- `Before=A` means B has an outgoing edge to A
- `Requires=B` means A depends on B (both ordering + activation)

### Topological Sort Algorithm

Boot ordering is computed via **Kahn's algorithm** (BFS-based topological sort):

1. Compute in-degree for every unit: $in(u) = |\{v : (v, u) \in E\}|$
2. Enqueue all units with $in(u) = 0$
3. While queue non-empty: dequeue $u$, start $u$, for each $(u, v) \in E$: decrement $in(v)$; if $in(v) = 0$, enqueue $v$

**Complexity:** $O(V + E)$ where $V$ = number of units, $E$ = number of dependency edges.

### Worked Example

Units: `network.target`, `sshd.service`, `nginx.service`, `app.service`

```
network.target → sshd.service
network.target → nginx.service
nginx.service  → app.service
```

In-degrees: network=0, sshd=1, nginx=1, app=1

| Step | Queue | Start | Updates |
|:---:|:---|:---|:---|
| 0 | [network] | network.target | sshd→0, nginx→0 |
| 1 | [sshd, nginx] | sshd.service | — |
| 2 | [nginx] | nginx.service | app→0 |
| 3 | [app] | app.service | — |

**Parallelism:** Steps 1-2 show sshd and nginx can start **simultaneously** — this is how systemd achieves parallel boot.

### Maximum Parallelism

The **critical path** determines minimum boot time:

$$T_{boot} \geq \max_{p \in paths} \sum_{u \in p} T_{start}(u)$$

The longest path through the DAG is the theoretical minimum boot time regardless of parallelism.

### Cycle Detection

systemd refuses to boot with dependency cycles. Detection uses DFS with three-color marking:
- White: unvisited
- Gray: in current DFS path
- Black: fully explored

A back-edge (gray → gray) indicates a cycle. Cost: $O(V + E)$.

---

## 2. Socket Activation — Connection Handoff

### The Model

Socket activation decouples listening from serving:

1. systemd creates socket $S$ and calls `bind()` + `listen()`
2. Connection arrives → systemd starts service unit
3. File descriptor passed via `sd_listen_fds()` (fd 3+)

### Queue Sizing

The listen backlog determines how many connections queue before the service starts:

$$max\_pending = backlog\_size$$

If the service takes $T_{start}$ seconds and connections arrive at rate $\lambda$:

$$expected\_queued = \lambda \times T_{start}$$

To avoid drops: $backlog \geq \lambda \times T_{start}$

**Example:** Service starts in 2 seconds, 100 connections/second:

$$backlog \geq 100 \times 2 = 200$$

### Activation Latency

$$T_{first\_request} = T_{socket\_create} + T_{service\_start} + T_{fd\_handoff} + T_{request\_process}$$

Typically: $\approx 0 + 500ms + 0.1ms + T_{app}$

---

## 3. Cgroup Resource Distribution — CPU Weight

### CPU Weight Shares

systemd uses cgroup v2 CPU weight (replacing cgroup v1 shares):

$$CPU_i = \frac{weight_i}{\sum_j weight_j} \times available\_CPU$$

Weight range: 1-10000, default: 100.

### Worked Example

Three services with weights 100, 200, 500 competing for 4 CPUs:

$$total = 100 + 200 + 500 = 800$$

| Service | Weight | CPU Share | CPU Cores |
|:---:|:---:|:---:|:---:|
| A | 100 | 12.5% | 0.50 |
| B | 200 | 25.0% | 1.00 |
| C | 500 | 62.5% | 2.50 |

**When not contending:** Each service can use up to its `CPUQuota` limit or all available CPU.

### CPU Quota (Hard Limit)

$$CPUQuota = \frac{allowed\_time}{period} \times 100\%$$

`CPUQuota=200%` means 2 full CPU cores. Enforced by CFS bandwidth throttling:

$$throttle\_when: \frac{runtime}{period} \geq \frac{quota}{period}$$

Default period: 100ms. At 200% quota, the cgroup gets 200ms of CPU time per 100ms wall time (across 2+ cores).

---

## 4. Memory Limits — Cgroup v2 Memory Controller

### Memory Accounting

$$memory.current = RSS + cache + kernel\_stack + slab + sock + swap\_usage$$

### Limit Hierarchy

| Setting | Behavior | Formula |
|:---|:---|:---|
| `MemoryMin` | Guaranteed minimum (no reclaim below) | Hard floor |
| `MemoryLow` | Best-effort minimum (reclaim only under pressure) | Soft floor |
| `MemoryHigh` | Throttle point (heavy reclaim above) | Soft ceiling |
| `MemoryMax` | Hard limit (OOM kill above) | Hard ceiling |

### Reclaim Pressure Model

When system memory is under pressure, cgroups with usage above `MemoryLow` are reclaimed proportionally:

$$reclaim_i = \frac{usage_i - MemoryLow_i}{\sum_j (usage_j - MemoryLow_j)} \times total\_reclaim$$

### Worked Example

Three cgroups, system needs to reclaim 1 GB:

| Cgroup | Usage | MemoryLow | Excess | Reclaim Share |
|:---:|:---:|:---:|:---:|:---:|
| A | 4 GB | 2 GB | 2 GB | $\frac{2}{6} \times 1GB = 333MB$ |
| B | 3 GB | 1 GB | 2 GB | $\frac{2}{6} \times 1GB = 333MB$ |
| C | 5 GB | 3 GB | 2 GB | $\frac{2}{6} \times 1GB = 333MB$ |

---

## 5. Rate Limiting — Restart and Start Policies

### Restart Rate Limiting

systemd enforces restart limits to prevent crash loops:

$$allowed = StartLimitBurst \text{ starts within } StartLimitIntervalSec$$

Default: 5 starts within 10 seconds. If exceeded, unit enters **failed** state.

### Restart Timing

$$T_{restart} = RestartSec + T_{stop} + T_{start}$$

With `RestartSec=5s`, a crashing service restarts at most:

$$max\_restarts\_per\_interval = \left\lfloor \frac{StartLimitIntervalSec}{T_{restart}} \right\rfloor$$

**Example:** `StartLimitBurst=5`, `StartLimitIntervalSec=30`, `RestartSec=1s`, $T_{stop} = 0.5s$, $T_{start} = 0.5s$:

$$T_{restart} = 1 + 0.5 + 0.5 = 2s$$

$$max = \lfloor 30 / 2 \rfloor = 15 \text{ possible restarts, but burst limit caps at 5}$$

---

## 6. Journal Storage — Log Sizing

### Storage Budget

systemd-journald enforces storage limits:

$$SystemMaxUse = \frac{filesystem\_size \times 0.10}{8}$$

Capped at default 4 GB. With `SystemKeepFree`, the effective limit:

$$effective\_limit = \min(SystemMaxUse,\ filesystem\_size - SystemKeepFree)$$

### Rotation Rate

$$retention\_days = \frac{SystemMaxUse}{daily\_log\_rate}$$

**Example:** 500 MB journal limit, 50 MB/day log rate:

$$retention = \frac{500}{50} = 10 \text{ days}$$

---

## 7. Boot Analysis — Critical Chain

### Critical Path Calculation

`systemd-analyze critical-chain` computes the longest path through the boot DAG:

$$T_{critical} = \sum_{i \in critical\_path} T_{activate}(i)$$

Each unit's activation time: $T_{activate} = T_{after\_deps} + T_{self\_start}$

The critical chain is the **bottleneck** — optimizing any unit not on this path has zero effect on boot time.

### Blame Sorting

`systemd-analyze blame` lists units by $T_{self\_start}$ descending. But a slow unit only matters if it's on the critical path.

---

## 8. Summary of systemd Mathematics

| Domain | Formula | Type |
|:---|:---|:---|
| Boot ordering | Topological sort | Graph theory, $O(V+E)$ |
| Parallelism | Critical path / longest path | DAG analysis |
| Socket activation | $backlog \geq \lambda \times T_{start}$ | Queuing theory |
| CPU weight | $w_i / \Sigma w_j$ | Proportional share |
| CPU quota | $runtime / period$ | Rate limiting |
| Memory reclaim | $(usage - low) / \Sigma excess$ | Proportional distribution |
| Restart limits | $burst / interval$ | Token bucket |
| Journal sizing | $max\_use / daily\_rate$ | Capacity planning |

---

*systemd is a graph scheduler, a resource allocator, and a supervision engine — all running the same proportional-share and rate-limiting math the kernel uses internally.*
