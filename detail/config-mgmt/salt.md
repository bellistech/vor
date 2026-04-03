# The Mathematics of Salt — Configuration Management Theory

> *Salt (SaltStack) uses a high-speed ZeroMQ message bus for real-time infrastructure control. Its execution model combines pub/sub messaging, pillar-based data isolation, state dependency graphs, and a reactor system with event-driven automation.*

---

## 1. Transport Layer (ZeroMQ Pub/Sub)

### The Problem

Salt communicates with minions over ZeroMQ (or TCP). Understanding the message delivery model is essential for scaling.

### Pub/Sub Fan-Out

When the master publishes a command:

$$\text{Messages sent} = 1 \quad \text{(multicast via PUB socket)}$$
$$\text{Messages received} = N \quad \text{(one per subscribed minion)}$$

Unlike Ansible's SSH model ($N$ connections), Salt sends **one** message regardless of fleet size.

### Response Collection

$$T_{command} = T_{publish} + \max_{i \in \text{targeted}} T_{execute_i} + T_{return}$$

$$T_{timeout} = \text{configured timeout (default 5s)}$$

### Throughput Comparison

| Transport | 100 nodes | 1,000 nodes | 10,000 nodes |
|:---|:---:|:---:|:---:|
| Ansible (SSH) | 20s | 200s | 2000s |
| Salt (ZeroMQ) | 0.5s | 0.8s | 2s |
| Salt (TCP) | 0.5s | 1.0s | 3s |

The improvement: $O(1)$ publish vs $O(N)$ connection establishment.

### Multi-Master Failover

$$\text{Availability} = 1 - \prod_{i=1}^{M} (1 - A_i)$$

For 2 masters each at 99.9%:

$$A = 1 - (0.001)^2 = 1 - 0.000001 = 99.9999\%$$

---

## 2. Targeting (Compound Matchers as Set Operations)

### The Problem

Salt targets minions using grains, pillar data, PCRE, and compound expressions. These are set operations.

### Targeting Algebra

| Matcher | Syntax | Set Operation |
|:---|:---|:---|
| Glob | `'web*'` | $\{m : \text{name}(m) \sim \text{glob}\}$ |
| PCRE | `'E@web\d+'` | $\{m : \text{name}(m) \sim \text{regex}\}$ |
| Grain | `'G@os:Ubuntu'` | $\{m : m.\text{grains}[\text{os}] = \text{Ubuntu}\}$ |
| Pillar | `'I@role:webserver'` | $\{m : m.\text{pillar}[\text{role}] = \text{webserver}\}$ |
| Compound | `'G@os:Ubuntu and E@web\d+'` | Intersection: $G_{Ubuntu} \cap R_{web}$ |
| Compound | `'G@os:Ubuntu or G@os:Debian'` | Union: $G_{Ubuntu} \cup G_{Debian}$ |
| Compound | `'not G@os:Windows'` | Complement: $U \setminus G_{Windows}$ |

### Worked Example

Fleet: 500 minions

| Matcher | Expression | Result Set Size |
|:---|:---|:---:|
| All | `'*'` | 500 |
| Grain | `'G@os:Ubuntu'` | 320 |
| Compound | `'G@os:Ubuntu and G@role:web'` | 45 |
| Exclude | `'G@os:Ubuntu and not G@env:dev'` | 280 |

---

## 3. State Dependency Graph (Requisite System)

### The Problem

Salt states define dependencies through requisites: `require`, `watch`, `onchanges`, `onfail`. These form a DAG.

### Requisite Types

| Requisite | Meaning | Edge Type |
|:---|:---|:---|
| `require` | Must succeed first | Hard dependency |
| `watch` | Require + trigger on change | Change-reactive |
| `onchanges` | Run only if dependency changed | Conditional |
| `onfail` | Run only if dependency failed | Error handler |
| `prereq` | Run before, only if would change | Pre-flight |

### The State Graph

$$G = (S, E)$$
$$S = \text{state declarations}$$
$$E = \{(s_i, s_j, \text{type}) : s_j \text{ has requisite on } s_i\}$$

### Execution Order

$$\text{order} = \text{toposort}(G)$$

If topological sort fails (cycle detected):

$$\exists \text{ cycle } \implies \text{state run aborts}$$

### Worked Example

```yaml
nginx_pkg:
  pkg.installed:
    - name: nginx

nginx_conf:
  file.managed:
    - name: /etc/nginx/nginx.conf
    - require:
      - pkg: nginx_pkg

nginx_svc:
  service.running:
    - name: nginx
    - watch:
      - file: nginx_conf
```

| Depth | State | Depends On |
|:---:|:---|:---|
| 0 | nginx_pkg | (none) |
| 1 | nginx_conf | nginx_pkg |
| 2 | nginx_svc | nginx_conf (watch) |

If `nginx_conf` changes, `watch` triggers a service restart — it's `require` + change detection.

---

## 4. Pillar Data Isolation

### The Problem

Pillar data is per-minion encrypted data. Each minion receives only its own pillar — no cross-minion data leakage.

### Isolation Property

$$\text{pillar}(m_i) \cap \text{pillar}(m_j) = \text{shared\_data} \quad \text{(only if top.sls targets both)}$$

$$\text{encrypted\_on\_wire}: \text{pillar}(m_i) \text{ encrypted with } K_{m_i}$$

### Pillar Compilation Cost

$$T_{pillar} = O(|\text{top.sls rules}| \times |\text{pillar sources}|)$$

For large pillars, this can be expensive:

| Minions | Pillar Size/Minion | Total Compile Time |
|:---:|:---:|:---:|
| 100 | 10 KB | 2s |
| 1,000 | 50 KB | 15s |
| 5,000 | 100 KB | 90s |

### Pillar Caching

$$T_{cached} = O(1) \quad \text{vs} \quad T_{fresh} = O(N)$$

Cached pillars reduce server load but introduce staleness risk.

---

## 5. Mine System (Minion Data Sharing)

### The Problem

Salt Mine allows minions to share specific data points with each other — a pull-based discovery mechanism.

### Mine Update Formula

$$\text{mine.interval} = I \text{ (default: 60 minutes)}$$

$$T_{staleness} \in [0, I]$$

### Data Volume

$$S_{mine} = N \times S_{per\_minion}$$

For 1,000 minions each sharing 5 KB of data:

$$S_{mine} = 1000 \times 5 = 5{,}000 \text{ KB} = 5 \text{ MB}$$

### Cross-Minion Configuration

Node A publishes its IP via mine. Node B collects all IPs to build config:

$$\text{backends} = \{m.\text{mine}[\text{ip}] : m \in \text{minions matching role:app}\}$$

This is the Salt equivalent of Chef's search and Puppet's exported resources.

---

## 6. Reactor System (Event-Driven Automation)

### The Problem

The Salt reactor watches the event bus and triggers actions based on event patterns. This is complex event processing.

### Event Matching

$$\text{match}(event, pattern) = \begin{cases}
\text{true} & \text{if } event.\text{tag} \sim pattern \\
\text{false} & \text{otherwise}
\end{cases}$$

### Reactor Pipeline

$$\text{Event} \xrightarrow{\text{bus}} \text{Reactor} \xrightarrow{\text{match}} \text{SLS template} \xrightarrow{\text{render}} \text{Action}$$

### Event Rate

$$\text{Events/sec} = \frac{N \times E_{per\_minion}}{I}$$

Where $E_{per\_minion}$ = events per minion per interval $I$.

| Source | Events/Minion/Min | 1,000 Minions |
|:---|:---:|:---:|
| State runs | 0.033 (every 30 min) | 33/min |
| Beacons (file watch) | 1 | 1,000/min |
| Beacons (process) | 0.1 | 100/min |
| Custom events | Variable | Variable |

### Reactor Capacity

$$\text{Max events/sec} \approx 500\text{-}1000 \quad \text{(single master)}$$

Beyond this, events queue and latency increases.

---

## 7. Highstate vs Orchestration

### Highstate (Per-Minion)

$$T_{highstate} = \max_{m \in \text{targeted}} T_{states}(m)$$

All minions apply states independently in parallel.

### Orchestration (Cross-Minion)

$$T_{orch} = \sum_{step=1}^{S} T_{step}$$

Orchestration steps are sequential — each step can target different minions:

$$\text{Step 1}: \text{target}(G_1) \rightarrow \text{state.apply}$$
$$\text{Step 2}: \text{target}(G_2) \rightarrow \text{state.apply}$$

### Comparison

| Mode | Parallelism | Ordering | Use Case |
|:---|:---|:---|:---|
| Highstate | All minions parallel | Per-minion DAG | Routine config |
| Orchestrate | Steps sequential, within-step parallel | Cross-minion | Deployments |

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $O(1)$ publish vs $O(N)$ SSH | Complexity | Transport |
| $G \cap R$, $G \cup G$, $U \setminus G$ | Set algebra | Targeting |
| $\text{toposort}(G)$ | Graph theory | State ordering |
| $\text{pillar}(m_i) \cap \text{pillar}(m_j)$ | Set isolation | Security |
| $1 - \prod(1 - A_i)$ | Probability | Multi-master HA |
| Events/sec = $N \times E / I$ | Rate computation | Reactor capacity |

---

*Salt's ZeroMQ transport makes it the fastest configuration management tool — $O(1)$ command distribution to any fleet size. The reactor system, pillar isolation, and requisite graph add the intelligence needed for production infrastructure.*
