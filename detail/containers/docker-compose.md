# The Mathematics of Docker Compose — Multi-Container Orchestration

> *Docker Compose is a declarative tool for defining multi-container applications. Its internals involve dependency graph resolution, network topology construction, and resource allocation across service replicas.*

---

## 1. Service Dependency Graph (DAG Resolution)

### The Problem

Compose must start services in dependency order. Services form a **directed acyclic graph** (DAG) via `depends_on`.

### Topological Sort

Given services $S = \{s_1, s_2, \ldots, s_n\}$ and dependencies $D \subseteq S \times S$, Compose performs a topological sort:

$$\text{start\_order} = \text{toposort}(S, D)$$

$$\text{Time complexity} = O(|S| + |D|)$$

### Parallelism from the DAG

Services at the same depth level can start in parallel:

$$\text{Parallelism} = \max_{l \in \text{levels}} |\{s : \text{depth}(s) = l\}|$$

### Worked Example

```
web → api → database
web → cache
worker → api
```

| Depth | Services | Started In Parallel |
|:---:|:---|:---:|
| 0 | database, cache | 2 |
| 1 | api | 1 |
| 2 | web, worker | 2 |

$$T_{startup} = \sum_{l=0}^{L} \max_{s \in \text{level}(l)} T_{ready}(s)$$

Total startup time is the sum of the slowest service at each level, not the sum of all services.

### Cycle Detection

If services form a cycle, Compose rejects the file:

$$\exists \text{ path } s_i \rightarrow s_j \rightarrow \cdots \rightarrow s_i \implies \text{ERROR}$$

Cycle detection runs in $O(|S| + |D|)$ using DFS with coloring.

---

## 2. Network Topology (Bridge Per Project)

### The Problem

Each Compose project creates an isolated bridge network. How do services discover each other?

### DNS Resolution Model

Every service registers with an embedded DNS server:

$$\text{DNS}: \text{service\_name} \rightarrow \{IP_1, IP_2, \ldots, IP_r\}$$

Where $r$ = number of replicas. Round-robin load balancing:

$$\text{target}(request_k) = IP_{(k \bmod r) + 1}$$

### Network Isolation

$$\text{Reachable}(s) = \{t : \exists \text{ shared network between } s \text{ and } t\}$$

For project with $N$ networks:

$$\text{Networks} = \{n_1, n_2, \ldots, n_N\}$$

$$\text{Reachable}(s) = \bigcup_{n \in \text{networks}(s)} \text{members}(n)$$

### Worked Example: Frontend/Backend Isolation

```yaml
networks:
  frontend:  # web, api
  backend:   # api, database
```

| Service | Networks | Can Reach |
|:---|:---|:---|
| web | frontend | api |
| api | frontend, backend | web, database |
| database | backend | api |
| web → database | none shared | BLOCKED |

---

## 3. Replica Scaling (Resource Multiplication)

### The Problem

`docker compose up --scale service=N` creates $N$ instances. What are the resource implications?

### Resource Formula

$$R_{total}(service) = N \times R_{per\_replica}$$

For CPU and memory limits:

$$\text{CPU}_{total} = N \times \text{cpu\_limit}$$
$$\text{MEM}_{total} = N \times \text{mem\_limit}$$

### Port Conflict Resolution

With host-mapped ports, scaling beyond 1 replica causes conflicts:

$$\text{Max replicas with host port} = 1$$

With port ranges:

$$\text{Max replicas} = p_{end} - p_{start} + 1$$

### Capacity Planning Table

| Service | Replicas | CPU Limit | Memory Limit | Total CPU | Total Memory |
|:---|:---:|:---:|:---:|:---:|:---:|
| api | 3 | 0.5 | 256 MB | 1.5 | 768 MB |
| worker | 5 | 1.0 | 512 MB | 5.0 | 2.5 GB |
| cache | 1 | 0.25 | 128 MB | 0.25 | 128 MB |
| **Total** | **9** | | | **6.75** | **3.4 GB** |

---

## 4. Volume Mount Performance

### The Problem

Compose supports bind mounts, named volumes, and tmpfs. Performance varies dramatically.

### I/O Latency Model

$$T_{IO} = T_{syscall} + T_{filesystem} + T_{transport}$$

| Mount Type | Transport Overhead | Use Case |
|:---|:---:|:---|
| Named volume | 0 (native) | Database storage |
| Bind mount (Linux) | 0 (native) | Source code sharing |
| Bind mount (macOS) | 5-50x slower | Development only |
| tmpfs | 0 (RAM-backed) | Secrets, temp files |

### macOS Bind Mount Penalty (VirtioFS)

$$T_{macOS} = T_{Linux} \times k \quad \text{where } k \in [2, 50]$$

For a `node_modules` directory with 50,000 files:

$$T_{npm\_install}^{Linux} \approx 10\text{s}$$
$$T_{npm\_install}^{macOS} \approx 60\text{s} \quad (k \approx 6)$$

### Mitigation: Sync Strategies

| Strategy | Speedup | Tradeoff |
|:---|:---:|:---|
| Named volume for deps | 5-10x | Manual sync needed |
| VirtioFS (default now) | 2-3x vs old gRPC-FUSE | Requires macOS 12.5+ |
| `:cached` flag (legacy) | 1.5-2x | Eventual consistency |

---

## 5. Health Check State Machine

### The Problem

Compose services with `healthcheck` transition through states that affect dependency readiness.

### State Transitions

$$\text{States} = \{\text{starting}, \text{healthy}, \text{unhealthy}\}$$

$$\text{starting} \xrightarrow{\text{check passes}} \text{healthy}$$
$$\text{starting} \xrightarrow{\text{retries exhausted}} \text{unhealthy}$$
$$\text{healthy} \xrightarrow{\text{check fails } \times \text{ retries}} \text{unhealthy}$$
$$\text{unhealthy} \xrightarrow{\text{check passes}} \text{healthy}$$

### Health Check Timing

$$T_{first\_healthy} = \text{start\_period} + (n - 1) \times \text{interval} + T_{check}$$

Where $n$ = the check attempt that first succeeds.

$$T_{declare\_unhealthy} = \text{start\_period} + \text{retries} \times \text{interval}$$

### Worked Example

```yaml
healthcheck:
  test: ["CMD", "pg_isready"]
  interval: 5s
  timeout: 3s
  retries: 3
  start_period: 10s
```

- Best case (passes immediately): $T = 10 + 0 + 0.1 = 10.1$ s
- Worst case (passes on last retry): $T = 10 + 2 \times 5 + 0.1 = 20.1$ s
- Failure (all retries fail): $T = 10 + 3 \times 5 = 25$ s → unhealthy

---

## 6. Compose File Merge (Override Resolution)

### The Problem

Multiple Compose files are merged using an override hierarchy. The merge follows a precedence model.

### Merge Precedence

$$\text{Final} = \text{Base} \triangleleft \text{Override}_1 \triangleleft \text{Override}_2 \triangleleft \cdots$$

Where $\triangleleft$ means "override with right side winning for scalar values":

$$\text{value}(key) = \begin{cases}
\text{Override}[key] & \text{if key exists in Override} \\
\text{Base}[key] & \text{otherwise}
\end{cases}$$

For lists (ports, volumes, environment):

$$\text{result} = \text{Base} \cup \text{Override} \quad \text{(append, no dedup)}$$

For mappings (labels, environment as map):

$$\text{result}[k] = \text{Override}[k] \text{ if exists, else Base}[k]$$

---

## 7. Build Context Transfer

### The Problem

The build context is sent to the Docker daemon. Its size directly impacts build initiation time.

### Transfer Time

$$T_{context} = \frac{S_{context}}{BW_{socket}} + T_{tar}$$

### .dockerignore Impact

$$S_{context} = S_{total} - S_{ignored}$$

| Project | Without .dockerignore | With .dockerignore | Savings |
|:---|:---:|:---:|:---:|
| Node.js app | 800 MB (node_modules) | 5 MB | 99.4% |
| Go project | 200 MB (vendor) | 15 MB | 92.5% |
| Python app | 150 MB (.venv) | 3 MB | 98% |

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $\text{toposort}(S, D)$ | Graph theory | Startup ordering |
| $k \bmod r$ | Modular arithmetic | DNS round-robin |
| $N \times R_{per\_replica}$ | Linear scaling | Resource planning |
| $T_{macOS} = k \times T_{Linux}$ | Multiplicative penalty | I/O performance |
| $\text{Base} \triangleleft \text{Override}$ | Merge algebra | File composition |
| $S_{total} - S_{ignored}$ | Set difference | Build context |

---

*Docker Compose turns a YAML file into a running distributed system — the dependency graph, network topology, and resource accounting happen automatically, but understanding the math lets you optimize startup time, resource usage, and I/O performance.*
