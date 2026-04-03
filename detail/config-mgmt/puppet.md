# The Mathematics of Puppet — Configuration Management Theory

> *Puppet declares desired state in a catalog — a directed acyclic graph of resources. The compiler resolves classes, the agent applies resources in topological order, and the report processor tracks convergence metrics. Its mathematical core is graph theory and constraint solving.*

---

## 1. Catalog Compilation (Resource Graph)

### The Problem

Puppet compiles manifests into a **catalog** — a DAG of resources with dependency edges. The compiler resolves classes, evaluates conditionals, and builds the graph.

### The Catalog Graph

$$G = (R, E)$$

Where:
- $R$ = set of resources (file, package, service, user, ...)
- $E$ = dependency edges (require, before, notify, subscribe)

### Graph Properties

$$|R| = \text{total managed resources}$$
$$|E| = \text{total dependency relationships}$$
$$\text{depth}(G) = \text{longest path in DAG}$$
$$\text{width}(G) = \max_l |\{r : \text{depth}(r) = l\}|$$

### Parallelism Potential

$$\text{max\_parallel} = \text{width}(G)$$

Resources at the same depth level have no dependencies and can run concurrently.

### Worked Example

```puppet
Package['nginx'] -> File['/etc/nginx/nginx.conf'] ~> Service['nginx']
Package['openssl'] -> File['/etc/nginx/ssl.conf'] ~> Service['nginx']
```

| Depth | Resources | Parallel |
|:---:|:---|:---:|
| 0 | Package[nginx], Package[openssl] | 2 |
| 1 | File[nginx.conf], File[ssl.conf] | 2 |
| 2 | Service[nginx] | 1 |

$$T_{apply} = \sum_{l=0}^{2} \max_{r \in \text{level}(l)} T(r)$$

---

## 2. Resource Abstraction Layer (Type/Provider Model)

### The Problem

Puppet separates **what** (resource type) from **how** (provider). This is a mathematical abstraction layer.

### The Abstraction

$$\text{Resource} = (\text{type}, \text{title}, \text{attributes})$$

$$\text{Provider}: \text{Resource} \rightarrow \text{System Commands}$$

### Provider Selection

$$\text{provider}(r) = \begin{cases}
\text{specified} & \text{if } r.\text{provider} \text{ is set} \\
\text{default}(\text{type}, \text{os}) & \text{otherwise}
\end{cases}$$

| Type | Linux Provider | macOS Provider | Windows Provider |
|:---|:---|:---|:---|
| package | apt/yum/dnf | brew | chocolatey |
| service | systemd | launchd | windows |
| user | useradd | dscl | ADSI |

### State Comparison

$$\text{changes}(r) = \{a \in \text{attributes}(r) : \text{current}(a) \neq \text{desired}(a)\}$$

$$\text{in\_sync}(r) \iff |\text{changes}(r)| = 0$$

---

## 3. Agent Run Timing (Pull Model)

### The Problem

The Puppet agent runs periodically, pulling catalogs from the server. Timing affects convergence speed and server load.

### Run Interval

$$T_{run} = \text{runinterval} + \text{random}(0, \text{splay\_limit})$$

Default: $\text{runinterval} = 1800\text{s}$ (30 min), $\text{splay\_limit} = \text{runinterval}$.

### Convergence Latency

For a change committed at time $t_0$:

$$T_{convergence} \in [T_{compile}, \text{runinterval} + \text{splay\_limit} + T_{compile} + T_{apply}]$$

$$T_{worst\_case} = 30 + 30 + T_{compile} + T_{apply} \approx 62 \text{ min}$$

### Server Load Distribution

With splay enabled, $N$ agents distribute evenly:

$$\text{Rate}_{avg} = \frac{N}{\text{runinterval}}$$

| Nodes | Interval | Avg Rate | Peak Rate (no splay) |
|:---:|:---:|:---:|:---:|
| 500 | 30 min | 16.7/min | 500/min (thundering herd) |
| 2,000 | 30 min | 66.7/min | 2,000/min |
| 10,000 | 30 min | 333/min | 10,000/min |

### Compiler Performance

$$T_{compile} = O(|C| \times |R|)$$

Where $|C|$ = number of classes, $|R|$ = resources per class. Typical compile times:

| Catalog Size | Resources | Compile Time |
|:---:|:---:|:---:|
| Small | 50-200 | 0.5-2 s |
| Medium | 200-1,000 | 2-8 s |
| Large | 1,000-5,000 | 8-30 s |
| Massive | 5,000+ | 30-120 s |

---

## 4. Hiera Data Lookup (Hierarchical Resolution)

### The Problem

Hiera resolves data through a hierarchy of YAML files. The lookup strategy determines which values win.

### Lookup Strategies

**First (default)** — return the first match:

$$V(key) = \text{data}_{first\ level\ defining\ key}(key)$$

**Unique** — merge arrays, deduplicate:

$$V(key) = \text{unique}\left(\bigcup_{l \in \text{hierarchy}} \text{data}_l(key)\right)$$

**Hash** — deep merge hashes:

$$V(key) = \text{data}_1(key) \cup_{deep} \text{data}_2(key) \cup_{deep} \cdots$$

**Deep** — recursive deep merge:

$$V(key) = \text{deep\_merge}(\text{data}_N(key), \ldots, \text{data}_1(key))$$

### Worked Example

Hierarchy: `nodes/%{fqdn}.yaml` > `environments/%{environment}.yaml` > `common.yaml`

| Level | `ntp::servers` |
|:---|:---|
| nodes/web1.yaml | `[10.0.0.1]` |
| environments/prod.yaml | `[10.0.0.2, 10.0.0.3]` |
| common.yaml | `[pool.ntp.org]` |

| Strategy | Result |
|:---|:---|
| first | `[10.0.0.1]` |
| unique | `[10.0.0.1, 10.0.0.2, 10.0.0.3, pool.ntp.org]` |

---

## 5. Certificate Authority (PKI Math)

### The Problem

Puppet uses mutual TLS — every agent has a signed certificate. The CA manages the PKI lifecycle.

### Certificate Signing

$$\text{cert} = \text{sign}(\text{CSR}, K_{CA\_private})$$

$$\text{verify}(cert, K_{CA\_public}) = \text{true} \iff \text{signed by CA}$$

### Certificate Expiry Model

$$T_{valid} = T_{issued} + \text{ttl}$$

Default TTL: 5 years = 157,680,000 seconds.

### CRL Size Growth

$$|CRL| = \sum_{\text{revoked}} 1 = R$$

Each agent checks the CRL on every connection. CRL distribution time:

$$T_{CRL} = \frac{|CRL| \times S_{entry}}{BW}$$

For 1,000 revoked certs at ~100 bytes each: $T_{CRL} = 100\text{KB} / BW \approx \text{negligible}$.

---

## 6. Exported Resources (Cross-Node Coordination)

### The Problem

Exported resources let one node define resources that are applied on other nodes. This creates cross-node dependencies.

### The Export/Collect Model

Node A exports:

$$\text{exports}(A) = \{r : r \text{ has } @@\text{ prefix}\}$$

Node B collects:

$$\text{collected}(B) = \{r \in \bigcup_n \text{exports}(n) : r \text{ matches query}\}$$

### PuppetDB Query Cost

$$T_{query} = O(\log N + K)$$

Where $N$ = total exported resources, $K$ = matching results.

### Convergence Delay

Exported resources require **two agent runs** to take effect:

1. **Run 1:** Node A exports resource → stored in PuppetDB
2. **Run 2:** Node B collects resource → applies it

$$T_{cross\_node} = T_{export\_run} + T_{collect\_run} \leq 2 \times (\text{runinterval} + \text{splay})$$

Worst case: $2 \times 60 = 120$ minutes.

---

## 7. Report Metrics (Convergence Analytics)

### The Key Metrics

$$\text{Changed} = |\{r : r.\text{status} = \text{changed}\}|$$
$$\text{Failed} = |\{r : r.\text{status} = \text{failed}\}|$$
$$\text{Skipped} = |\{r : r.\text{status} = \text{skipped}\}|$$
$$\text{Total} = |\text{catalog}|$$

### Convergence Ratio

$$\text{Convergence\%} = \frac{\text{Total} - \text{Changed} - \text{Failed}}{\text{Total}} \times 100$$

A healthy infrastructure should be > 99% converged.

### Fleet Health

$$\text{Fleet convergence} = \frac{|\{n : \text{Changed}(n) = 0 \text{ AND } \text{Failed}(n) = 0\}|}{N} \times 100$$

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $G = (R, E)$ DAG | Graph theory | Catalog structure |
| $\text{current} \neq \text{desired}$ | State comparison | Resource sync |
| First/unique/hash/deep merge | Resolution strategies | Hiera lookup |
| $N / \text{runinterval}$ | Rate distribution | Agent scheduling |
| $2 \times \text{runinterval}$ | Delay multiplication | Exported resources |
| $(\text{Total} - \text{Changed}) / \text{Total}$ | Ratio | Convergence metric |

---

*Puppet's catalog is a graph — the compiler builds it, the agent topologically sorts it, the reporter measures convergence. Every 30-minute cycle, this graph-theoretic machinery ensures thousands of nodes match their declared state.*
