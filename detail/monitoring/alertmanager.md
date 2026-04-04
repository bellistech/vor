# The Mathematics of Alertmanager — Routing Trees, Grouping, and Notification Theory

> *Alertmanager transforms a stream of firing alerts into actionable notifications. The math covers routing tree traversal, group combinatorics, inhibition graph logic, silence interval coverage, and notification deduplication in HA clusters.*

---

## 1. Routing Tree Traversal (Tree Theory)

### Route Matching Model

The routing configuration forms a tree. Each alert traverses from root to leaf:

$$\text{match}(a, r) = \bigwedge_{(k,v) \in r.\text{matchers}} (a.\text{labels}[k] = v)$$

For regex matchers:

$$\text{match\_re}(a, r) = \bigwedge_{(k, p) \in r.\text{matchers}} (a.\text{labels}[k] \sim p)$$

### Traversal Complexity

For a routing tree with depth $d$ and branching factor $b$:

$$\text{Nodes} = \frac{b^{d+1} - 1}{b - 1}$$

$$\text{Worst-case matches per alert} = d \times b \quad \text{(check all siblings at each level)}$$

| Depth | Branching | Total Routes | Checks/Alert |
|:---:|:---:|:---:|:---:|
| 2 | 3 | 13 | 6 |
| 3 | 4 | 85 | 12 |
| 3 | 10 | 1,111 | 30 |
| 4 | 5 | 781 | 20 |

With `continue: true`, an alert may match multiple siblings, increasing notification fan-out.

---

## 2. Alert Grouping (Set Theory)

### Group Key

Alerts are grouped by a subset of labels $G = \{g_1, g_2, \ldots, g_k\}$:

$$\text{group\_key}(a) = (a.\text{labels}[g_1], a.\text{labels}[g_2], \ldots, a.\text{labels}[g_k])$$

### Number of Groups

$$|\text{Groups}| = \left|\left\{ \text{group\_key}(a) : a \in \text{Alerts} \right\}\right| \leq \prod_{i=1}^{k} |V_{g_i}|$$

### Notification Volume

$$N_{\text{notifications}} = |\text{Groups}| \times \left\lceil \frac{T_{\text{window}}}{T_{\text{repeat}}} \right\rceil$$

| Alerts | group_by labels | Distinct Groups | Repeat Interval | Notifications/day |
|:---:|:---:|:---:|:---:|:---:|
| 100 | alertname(10) | 10 | 4h | 60 |
| 100 | alertname(10), cluster(3) | 30 | 4h | 180 |
| 1,000 | alertname(50), instance(100) | 500 | 4h | 3,000 |
| 1,000 | alertname(50) | 50 | 12h | 100 |

### Grouping Reduces Notifications

Without grouping (each alert is its own group):

$$N_{\text{ungrouped}} = |\text{Alerts}| \times \left\lceil \frac{T}{T_{\text{repeat}}} \right\rceil$$

Grouping reduction factor:

$$R = \frac{|\text{Alerts}|}{|\text{Groups}|} \quad \text{(alerts per notification)}$$

---

## 3. Inhibition Logic (Graph Theory)

### Inhibition as Directed Graph

Each inhibition rule creates edges from source to target alerts:

$$\text{inhibits}(s, t) \iff \text{match}(s, R_{\text{source}}) \wedge \text{match}(t, R_{\text{target}}) \wedge \forall l \in E: s[l] = t[l]$$

where $E$ = set of `equal` labels.

### Inhibition Evaluation Complexity

For $n$ firing alerts and $r$ inhibition rules with $|E|$ equal labels:

$$O(n^2 \times r \times |E|)$$

### Inhibition Chain

Inhibition is NOT transitive. If A inhibits B and B inhibits C, A does NOT inhibit C.

$$\text{inhibited}(t) = \exists s \in \text{Firing}: \text{inhibits}(s, t) \wedge \neg \text{inhibited}(s)$$

| Firing Alerts | Inhibition Rules | Equal Labels | Comparisons |
|:---:|:---:|:---:|:---:|
| 10 | 2 | 2 | 400 |
| 100 | 3 | 2 | 60,000 |
| 1,000 | 5 | 3 | 15,000,000 |

---

## 4. Silence Coverage (Interval Mathematics)

### Silence as Time-Label Predicate

$$\text{silenced}(a, t) = \exists s \in S: \text{match}(a, s.\text{matchers}) \wedge s.t_{\text{start}} \leq t \leq s.t_{\text{end}}$$

### Coverage Ratio

$$\text{Coverage} = \frac{\sum_{s \in S} (s.t_{\text{end}} - s.t_{\text{start}})}{T_{\text{total}}} \quad \text{(with overlap dedup)}$$

For non-overlapping silences:

$$\text{Coverage} = \frac{\sum_{i=1}^{n} \Delta t_i}{T_{\text{total}}}$$

### Maintenance Window Planning

| Window Duration | Repeat (weekly) | Monthly Coverage |
|:---:|:---:|:---:|
| 1 hour | 1x/week | 0.6% |
| 4 hours | 1x/week | 2.4% |
| 4 hours | daily | 16.7% |
| 8 hours | daily | 33.3% |

---

## 5. HA Cluster Deduplication (Consensus)

### Gossip Protocol

Alertmanager instances share notification state via a gossip mesh (Memberlist):

$$\text{Convergence time} \approx O(\log n) \times T_{\text{gossip}}$$

where $n$ = cluster size, $T_{\text{gossip}}$ = gossip interval (~200ms).

### Deduplication Window

A notification is sent only once per group across the cluster. The deduplication relies on consistent hashing of the group key:

$$\text{owner}(\text{group}) = \text{hash}(\text{group\_key}) \mod n$$

### Split-Brain Notifications

If the cluster partitions into $k$ partitions:

$$\text{Max duplicate notifications} = k$$

| Cluster Size | Gossip Interval | Convergence | Partition Risk |
|:---:|:---:|:---:|:---:|
| 2 | 200ms | ~200ms | Low |
| 3 | 200ms | ~300ms | Very Low |
| 5 | 200ms | ~500ms | Minimal |

---

## 6. Notification Timing (Scheduling)

### Timeline of a New Alert Group

$$T_{\text{first\_notify}} = T_{\text{fire}} + T_{\text{group\_wait}}$$

$$T_{\text{update}} = T_{\text{first\_notify}} + T_{\text{group\_interval}}$$

$$T_{\text{re-notify}} = T_{\text{first\_notify}} + n \times T_{\text{repeat}}$$

### Total Notifications Over Duration

$$N = 1 + \left\lfloor \frac{T_{\text{duration}} - T_{\text{group\_wait}}}{T_{\text{repeat}}} \right\rfloor$$

| Duration | group_wait | repeat_interval | Notifications |
|:---:|:---:|:---:|:---:|
| 1 hour | 30s | 4h | 1 |
| 8 hours | 30s | 4h | 2 |
| 24 hours | 30s | 4h | 6 |
| 24 hours | 30s | 1h | 24 |
| 24 hours | 30s | 12h | 2 |

---

## Prerequisites

probability, set-theory, graph-theory, prometheus

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Route matching (per alert) | O(d * b) | O(d) stack |
| Grouping (all alerts) | O(n) hashing | O(g) groups |
| Inhibition check | O(n^2 * r) | O(n) alert set |
| Silence matching | O(n * s) | O(s) silences |
| Gossip convergence | O(log c) rounds | O(c) peer state |
| Notification dispatch | O(g) per cycle | O(g) pending |
