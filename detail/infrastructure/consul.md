# The Mathematics of Consul — Consensus, Gossip, and Failure Detection

> *Consul's distributed systems foundations rest on two distinct protocols: Raft consensus for strong consistency of the catalog and KV store, and SWIM-based gossip for scalable failure detection and membership. Each has rigorous mathematical properties that determine cluster sizing, failure tolerance, and convergence guarantees.*

---

## 1. Raft Consensus (Distributed Systems)

### The Problem

Consul servers use Raft to maintain a consistent, replicated log of all service registrations, KV writes, and intentions. The protocol guarantees linearizable reads and writes as long as a majority (quorum) of servers are available.

### The Formula

For a cluster of $N$ servers, the quorum size is:

$$Q = \left\lfloor \frac{N}{2} \right\rfloor + 1$$

Maximum tolerable failures before losing consensus:

$$F_{max} = N - Q = \left\lfloor \frac{N - 1}{2} \right\rfloor$$

### Worked Examples

| Servers ($N$) | Quorum ($Q$) | Fault Tolerance ($F_{max}$) | Notes |
|:---:|:---:|:---:|:---|
| 1 | 1 | 0 | Dev only, no redundancy |
| 3 | 2 | 1 | Minimum production |
| 5 | 3 | 2 | Recommended production |
| 7 | 4 | 3 | Large-scale, rare |

### Why Not Even Numbers

For $N = 4$: $Q = 3$, $F_{max} = 1$. Same fault tolerance as $N = 3$ but with higher coordination overhead:

$$\text{Commit latency} \propto \text{median}(RTT_{Q-1}) \quad \text{(wait for quorum ACKs)}$$

With $N = 4$, you wait for 2 of 3 followers. With $N = 3$, you wait for 1 of 2. More servers, same tolerance, higher latency.

---

## 2. Gossip Protocol Convergence (Epidemiology)

### The Problem

Consul uses a SWIM-variant gossip protocol (via Serf) for membership and failure detection. Each node periodically picks a random peer and exchanges state. We need to know how quickly information propagates to all nodes.

### The Formula

For $N$ nodes with gossip fanout $k$ (number of peers contacted per round), the expected rounds to reach all nodes:

$$R = \frac{\ln(N)}{\ln(k + 1)}$$

More precisely, the probability that all $N$ nodes have received the information after $r$ rounds:

$$P(\text{all informed after } r) \approx \left(1 - \left(1 - \frac{k}{N}\right)^r\right)^N$$

### Worked Examples

With default Consul gossip fanout $k = 3$:

| Cluster Size ($N$) | Rounds to Full Propagation | Time at 1s interval |
|:---:|:---:|:---:|
| 10 | $\lceil \ln(10)/\ln(4) \rceil = 2$ | ~2s |
| 100 | $\lceil \ln(100)/\ln(4) \rceil = 4$ | ~4s |
| 1,000 | $\lceil \ln(1000)/\ln(4) \rceil = 5$ | ~5s |
| 10,000 | $\lceil \ln(10000)/\ln(4) \rceil = 7$ | ~7s |

Key property: gossip scales logarithmically. Doubling the cluster adds roughly one more round.

---

## 3. Failure Detection (Probability Theory)

### The Problem

Consul's SWIM protocol detects node failures through probe-and-relay. A node sends a direct ping, and if it fails, asks $k$ indirect probes via other nodes. We need to calculate the false-positive rate (marking a healthy node as failed) to tune timeouts.

### The Formula

Probability of false positive (healthy node incorrectly marked failed) with direct probe timeout $t_d$ and $k$ indirect probes:

$$P(\text{false positive}) = P(\text{direct timeout}) \cdot P(\text{all } k \text{ indirect timeout})$$

If packet loss probability is $p$ (independent):

$$P(\text{false positive}) = p \cdot p^k = p^{k+1}$$

For round-trip (probe + ack), both packets must succeed, so loss probability per probe is:

$$p_{probe} = 1 - (1 - p_{loss})^2$$

$$P(\text{false positive}) = p_{probe}^{k+1}$$

### Worked Examples

With 1% packet loss ($p_{loss} = 0.01$), $k = 3$ indirect probes:

$$p_{probe} = 1 - (1 - 0.01)^2 = 1 - 0.9801 = 0.0199$$

$$P(\text{false positive}) = 0.0199^4 \approx 1.57 \times 10^{-7}$$

| Packet Loss | Direct Only ($k=0$) | $k=1$ | $k=3$ | $k=5$ |
|:---:|:---:|:---:|:---:|:---:|
| 1% | $0.02$ | $3.96 \times 10^{-4}$ | $1.57 \times 10^{-7}$ | $6.2 \times 10^{-11}$ |
| 5% | $0.10$ | $9.75 \times 10^{-3}$ | $9.5 \times 10^{-5}$ | $9.3 \times 10^{-7}$ |
| 10% | $0.19$ | $3.61 \times 10^{-2}$ | $1.3 \times 10^{-3}$ | $4.7 \times 10^{-5}$ |

---

## 4. Anti-Entropy and Consistency (Information Theory)

### The Problem

Consul runs an anti-entropy process that periodically synchronizes each agent's local state with the catalog on the servers. The convergence time determines how long stale data can persist after a change.

### The Formula

Anti-entropy sync interval $I$ (default 60s). Maximum staleness for a single agent:

$$t_{stale}^{max} = I$$

For the full cluster of $C$ clients, expected time until all agents have synced at least once:

$$E[t_{all\_synced}] = I \cdot H_C = I \cdot \sum_{i=1}^{C} \frac{1}{i} \approx I \cdot (\ln C + \gamma)$$

where $H_C$ is the $C$th harmonic number and $\gamma \approx 0.5772$ is the Euler-Mascheroni constant.

### Worked Examples

With $I = 60\text{s}$:

| Clients ($C$) | $E[t_{all\_synced}]$ | Max Staleness |
|:---:|:---:|:---:|
| 10 | $60 \times 2.93 \approx 176\text{s}$ | 60s |
| 100 | $60 \times 5.19 \approx 311\text{s}$ | 60s |
| 1,000 | $60 \times 7.49 \approx 449\text{s}$ | 60s |

Note: individual staleness is bounded by $I$, but the coupon collector problem governs when *all* agents have synced.

---

## 5. KV Store Consistency Models (Distributed Computing)

### The Problem

Consul KV reads support three consistency modes: `default` (leader leasing), `consistent` (Raft verify), and `stale` (any server). Each has different latency and staleness trade-offs.

### The Formula

Read latency by mode:

$$T_{stale} \approx RTT_{client \to nearest}$$

$$T_{default} \approx RTT_{client \to leader} + T_{lease\_check}$$

$$T_{consistent} \approx RTT_{client \to leader} + T_{raft\_verify}$$

Where $T_{raft\_verify}$ requires a round of Raft heartbeats:

$$T_{raft\_verify} \approx \text{median}(RTT_{leader \to follower_i}) \quad \text{for } i \in Q - 1$$

### Staleness Bounds

| Mode | Max Staleness | Availability on Partition |
|:---|:---|:---|
| `stale` | $\leq T_{gossip\_propagation} + I$ | Available (reads from any server) |
| `default` | $\leq T_{leader\_lease}$ (typically 0) | Available if leader reachable |
| `consistent` | 0 (linearizable) | Unavailable without quorum |

---

## 6. Cluster Sizing and Performance (Capacity Planning)

### The Problem

Raft write throughput is limited by quorum latency. We need to model the maximum write rate as a function of cluster size and network latency.

### The Formula

Maximum write throughput (sequential commits):

$$W_{max} = \frac{1}{T_{commit}}$$

$$T_{commit} = T_{log\_append} + \text{median}_{Q-1}(RTT_{leader \to follower_i})$$

For pipelined Raft (Consul default):

$$W_{pipelined} = \frac{\text{pipeline\_depth}}{T_{commit}}$$

### Worked Examples

$T_{log\_append} = 1\text{ms}$, same-rack RTT = 0.5ms, cross-AZ RTT = 2ms:

| Topology | Commit Time | Writes/sec (pipeline=20) |
|:---|:---:|:---:|
| 3 servers, same rack | $1 + 0.5 = 1.5\text{ms}$ | $20/1.5 \approx 13,333$ |
| 5 servers, same AZ | $1 + 0.5 = 1.5\text{ms}$ | $\approx 13,333$ |
| 5 servers, 3 AZs | $1 + 2 = 3\text{ms}$ | $20/3 \approx 6,667$ |
| 5 servers, 2 regions | $1 + 50 = 51\text{ms}$ | $20/51 \approx 392$ |

Cross-region Raft is not recommended. Use WAN federation with independent Raft groups per datacenter.

## Prerequisites

- raft-consensus, gossip-protocols, distributed-systems, dns, networking, service-mesh
