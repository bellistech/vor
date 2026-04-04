# The Mathematics of Cilium -- eBPF Packet Processing and Network Policy Evaluation

> *Cilium's eBPF datapath processes packets through a pipeline of BPF programs attached to kernel hooks, where network policy evaluation reduces to Boolean predicate matching over packet tuples, L7 protocol parsing applies finite automata to payload bytes, and Hubble flow aggregation uses streaming algorithms to maintain per-connection state at line rate.*

---

## 1. eBPF Packet Classification (Tuple Matching)

### The Problem

Cilium attaches eBPF programs to TC (traffic control) hooks on each pod's veth interface. Each packet is classified against a set of network policy rules by matching its header tuple against policy selectors. The classification must happen in O(1) amortized time to sustain line-rate throughput.

### The Formula

Packet tuple: $t = (\text{src\_ip}, \text{dst\_ip}, \text{src\_port}, \text{dst\_port}, \text{proto}, \text{identity})$

Policy rule set $R = \{r_1, \ldots, r_n\}$ where each $r_i$ is a predicate over tuples.

Policy decision:

$$\text{verdict}(t) = \begin{cases} \text{ALLOW} & \exists r \in R_{\text{allow}} : r(t) = \text{true} \\ \text{DROP} & \text{otherwise (default deny)} \end{cases}$$

BPF map lookup complexity: $O(1)$ per hash map lookup, $O(\log n)$ per LPM trie lookup.

Total per-packet cost:

$$C_{\text{packet}} = C_{\text{identity\_lookup}} + C_{\text{policy\_eval}} + C_{\text{conntrack}}$$

$$= O(1) + O(|R_{\text{applicable}}|) + O(1)$$

### Worked Examples

Cluster with 500 pods, 50 CiliumNetworkPolicies, average 3 rules each = 150 total rules.

Per-pod applicable rules (after identity filtering): ~5 rules on average.

Per-packet evaluation: 1 identity map lookup + 5 rule checks + 1 conntrack lookup.

At 10 Gbps with 64-byte packets: $\frac{10 \times 10^9}{(64 + 20) \times 8} = 14.88$ million packets/second.

Per-packet budget: $\frac{1}{14.88 \times 10^6} = 67.2$ ns.

BPF map lookup: ~50 ns. 5 rule evaluations: ~100 ns. Conntrack: ~50 ns.

Total: ~200 ns -- requires conntrack fast-path (established connections skip policy eval):

$$C_{\text{established}} = C_{\text{conntrack}} = 50 \text{ ns} \quad (\text{well within budget})$$

---

## 2. Identity-Based Security (Label Algebra)

### The Problem

Cilium assigns numeric security identities to groups of pods based on their Kubernetes labels. Network policies reference these identities rather than IP addresses, decoupling security from network topology. The identity assignment must be consistent across the cluster.

### The Formula

Label set function: $L : \text{Pod} \to 2^{\text{Labels}}$

Identity assignment: $I : 2^{\text{Labels}} \to \mathbb{N}$

$$I(L(p_1)) = I(L(p_2)) \iff L(p_1) = L(p_2)$$

Policy selector match:

$$\text{match}(p, s) = s \subseteq L(p)$$

where $s$ is the selector label set.

Number of distinct identities:

$$|I| \leq |\{L(p) : p \in \text{Pods}\}|$$

Identity-based policy evaluation:

$$\text{allowed}(\text{src}, \text{dst}) = \exists r \in R : I(L(\text{src})) \in r.\text{fromIdentities} \wedge r.\text{ports} \ni (\text{dst\_port}, \text{proto})$$

### Worked Examples

Cluster with 500 pods, 20 unique label combinations:

$$|I| = 20 \text{ identities} \quad (\text{vs. 500 IP-based rules})$$

Policy: "allow app=frontend to app=api on port 8080."

Frontend pods: 10 pods, all identity $I_1$.
API pods: 5 pods, all identity $I_2$.

IP-based rules needed: $10 \times 5 = 50$ allow rules.
Identity-based rules: 1 rule ($I_1 \to I_2$:8080).

Rule reduction: $\frac{50}{1} = 50\times$.

When frontend scales to 100 pods: still 1 identity-based rule (vs. 500 IP rules).

---

## 3. L7 Protocol Parsing (Finite Automata)

### The Problem

Cilium's L7 policies parse HTTP, gRPC, Kafka, and DNS payloads to enforce application-layer access control. The parser must extract method, path, and headers from the byte stream using a finite automaton that runs within the eBPF or Envoy proxy context.

### The Formula

HTTP request parser as DFA $A = (Q, \Sigma, \delta, q_0, F)$:

- States $Q$: method, URI, version, header-name, header-value, body
- Alphabet $\Sigma$: byte values $[0, 255]$
- Transitions $\delta$: RFC 7230 grammar

L7 policy match:

$$\text{match}_{\text{L7}}(r, \text{method}, \text{path}, \text{headers}) = (r.\text{method} = \text{method}) \wedge (\text{path} \in r.\text{path\_regex}) \wedge (r.\text{headers} \subseteq \text{headers})$$

Regex matching complexity for path pattern:

$$C_{\text{regex}} = O(|\text{path}| \times |\text{NFA states}|)$$

### Worked Examples

L7 policy rule: `GET /api/v1/users/[0-9]+` with header `Authorization: Bearer .*`.

Request: `GET /api/v1/users/42 HTTP/1.1\r\nAuthorization: Bearer eyJ...`

NFA for path regex `\/api\/v1\/users\/[0-9]+`: 22 states.

Path length: 18 bytes. Matching cost: $O(18 \times 22) = O(396)$ operations.

At 100,000 requests/second: $396 \times 100{,}000 = 39.6 \times 10^6$ operations/second.

On a 3 GHz core: $\frac{39.6 \times 10^6}{3 \times 10^9} = 1.32\%$ CPU utilization for L7 parsing alone.

Comparison: L3/L4 policy (tuple match only) costs ~5 operations per packet vs 400+ for L7.

---

## 4. Hubble Flow Aggregation (Streaming Statistics)

### The Problem

Hubble observes every packet decision (allow/drop) and aggregates flows for visualization and alerting. With millions of packets per second, Hubble must maintain per-flow state using memory-efficient streaming data structures.

### The Formula

Flow key: $k = (\text{src\_id}, \text{dst\_id}, \text{dst\_port}, \text{proto}, \text{verdict})$

Flow state: $s_k = (\text{packet\_count}, \text{byte\_count}, \text{first\_seen}, \text{last\_seen})$

Memory per flow entry:

$$M_{\text{flow}} = |k| + |s_k| \approx 32 + 32 = 64 \text{ bytes}$$

Total memory for $n$ concurrent flows:

$$M_{\text{total}} = n \times M_{\text{flow}}$$

Event loss rate when ingest exceeds drain:

$$P(\text{loss}) = \max\left(0, 1 - \frac{\mu_{\text{drain}}}{\lambda_{\text{ingest}}}\right)$$

### Worked Examples

Cluster with 2,000 pods, average 50 concurrent connections per pod:

$$n = 2{,}000 \times 50 = 100{,}000 \text{ flows}$$

$$M_{\text{total}} = 100{,}000 \times 64 = 6.4 \text{ MB}$$

Hubble ring buffer: 16 MB, event size 256 bytes:

$$N_{\text{events}} = \frac{16 \times 10^6}{256} = 62{,}500 \text{ events}$$

At 50,000 flow events/second with 30,000 drain rate:

$$P(\text{loss}) = 1 - \frac{30{,}000}{50{,}000} = 0.40 = 40\% \text{ event loss}$$

Solution: increase Hubble relay replicas or enable sampling at rate $\frac{1}{k}$ where $k = \lceil\frac{50{,}000}{30{,}000}\rceil = 2$.

---

## 5. Cluster Mesh Routing (Distributed Consensus)

### The Problem

Cilium Cluster Mesh connects multiple Kubernetes clusters, sharing service endpoints and identity information via etcd. Global services must route requests to the optimal cluster based on latency, load, and affinity policies.

### The Formula

Service $s$ with endpoints across $c$ clusters: $E_s = \{e_1^{(1)}, \ldots, e_{n_1}^{(1)}, \ldots, e_1^{(c)}, \ldots, e_{n_c}^{(c)}\}$

Routing decision with local affinity weight $w_{\text{local}}$:

$$P(\text{route to cluster } j) = \frac{w_j \cdot |E_s^{(j)}|}{\sum_{i=1}^{c} w_i \cdot |E_s^{(i)}|}$$

where $w_j = w_{\text{local}}$ if $j$ = local cluster, else $w_j = 1$.

Expected cross-cluster latency:

$$\bar{L} = \sum_{j=1}^{c} P(j) \cdot L_j$$

Identity synchronization: each cluster maintains a shared etcd with $|I|$ identities. Sync overhead:

$$\text{sync\_ops} = c \cdot |I| \cdot f_{\text{update}} \quad \text{(operations per second)}$$

### Worked Examples

3 clusters with global service `api`: $|E^{(1)}| = 5$, $|E^{(2)}| = 3$, $|E^{(3)}| = 8$ endpoints.

Local affinity weight $w_{\text{local}} = 5$, request from cluster 1:

$$P(1) = \frac{5 \times 5}{5 \times 5 + 1 \times 3 + 1 \times 8} = \frac{25}{36} = 69.4\%$$

$$P(2) = \frac{3}{36} = 8.3\%, \quad P(3) = \frac{8}{36} = 22.2\%$$

Latencies: $L_1 = 1$ ms, $L_2 = 15$ ms, $L_3 = 8$ ms.

$$\bar{L} = 0.694(1) + 0.083(15) + 0.222(8) = 0.694 + 1.245 + 1.776 = 3.72 \text{ ms}$$

Without local affinity ($w_{\text{local}} = 1$):

$$P(1) = \frac{5}{16} = 31.25\%, \quad \bar{L} = 0.3125(1) + 0.1875(15) + 0.5(8) = 7.12 \text{ ms}$$

Local affinity reduces average latency by $\frac{7.12 - 3.72}{7.12} = 47.8\%$.

---

## 6. Bandwidth Management (Token Bucket and BBR)

### The Problem

Cilium's bandwidth manager enforces per-pod rate limits using token bucket algorithms in eBPF, and optionally enables BBR congestion control for fair bandwidth sharing. The token bucket parameters determine burst tolerance and sustained rate.

### The Formula

Token bucket with rate $r$ (tokens/second) and burst size $b$ (tokens):

$$\text{tokens}(t) = \min(b, \text{tokens}(t-1) + r \cdot \Delta t)$$

Packet of size $s$ is transmitted if:

$$\text{tokens}(t) \geq s \implies \text{transmit}, \quad \text{tokens}(t) \mathrel{-}= s$$

Maximum burst duration at rate $R_{\text{burst}} > r$:

$$T_{\text{burst}} = \frac{b}{R_{\text{burst}} - r}$$

BBR bandwidth estimation:

$$\text{BtlBw} = \max_{\text{window}} \frac{\text{delivered}}{\Delta t}$$

$$\text{RTprop} = \min_{\text{window}} \text{RTT}$$

$$\text{pacing\_rate} = \frac{\text{BtlBw} \times \text{RTprop}}{\text{RTT}}$$

### Worked Examples

Pod with egress limit: $r = 10$ Mbps, burst $b = 1$ MB.

Sustained throughput: 10 Mbps.

Burst at 100 Mbps:

$$T_{\text{burst}} = \frac{1 \times 10^6 \times 8}{(100 - 10) \times 10^6} = \frac{8 \times 10^6}{90 \times 10^6} = 88.9 \text{ ms}$$

After burst: rate drops to sustained 10 Mbps.

Data transferred during burst: $100 \text{ Mbps} \times 88.9 \text{ ms} = 8.89 \text{ Mb} = 1.11 \text{ MB}$.

BBR: measured $\text{BtlBw} = 1$ Gbps, $\text{RTprop} = 0.5$ ms, $\text{RTT} = 2$ ms:

$$\text{pacing\_rate} = \frac{1 \times 10^9 \times 0.5 \times 10^{-3}}{2 \times 10^{-3}} = 250 \text{ Mbps}$$

---

## Prerequisites

- boolean-algebra, finite-automata, graph-theory, streaming-algorithms, distributed-systems, queuing-theory, congestion-control
