# The Mathematics of CockroachDB — Raft Consensus, Range Distribution, and Multi-Region Latency

> *CockroachDB distributes SQL across ranges replicated via Raft. The math covers consensus quorum latency, range split heuristics, multi-region write amplification, and survivability guarantees under partition models.*

---

## 1. Raft Consensus Latency (Distributed Systems)

### The Problem

Every write must be committed by a Raft quorum. The commit latency equals the time for a majority of replicas to acknowledge, which depends on network round-trip times between replicas.

### The Formula

For $n$ replicas, a quorum requires $\lfloor n/2 \rfloor + 1$ acknowledgments. The write latency is the $(\lfloor n/2 \rfloor + 1)$-th order statistic of replica response times:

$$T_{\text{write}} = T_{\text{leader}} + T_{(\lfloor n/2 \rfloor + 1)}(\{d_1, d_2, \ldots, d_{n-1}\})$$

where $T_{(k)}$ is the $k$-th smallest RTT to followers and $T_{\text{leader}}$ is local processing time.

For replicas with independent RTTs drawn from distribution $F(t)$, the expected quorum latency:

$$E[T_{(k)}] = k \cdot \binom{n-1}{k} \int_0^\infty t \cdot f(t) \cdot F(t)^{k-1} \cdot (1-F(t))^{n-1-k} \, dt$$

### Worked Examples

3-node cluster, replication factor 3, quorum = 2. Leader needs 1 follower ACK:

| Topology | Follower RTTs | Quorum Latency | Total Write |
|:---:|:---:|:---:|:---:|
| Same rack | 0.1ms, 0.1ms | 0.1ms | ~1ms |
| Same region | 1ms, 2ms | 1ms | ~3ms |
| Cross-region (US) | 30ms, 70ms | 30ms | ~35ms |
| Global (US/EU/AP) | 80ms, 150ms | 80ms | ~85ms |

5-node cluster, replication factor 5, quorum = 3. Leader needs 2 follower ACKs:

$$T_{\text{write}} = T_{\text{leader}} + T_{(2)}(\{1, 2, 30, 70\}) = 1 + 2 = 3 \text{ ms}$$

## 2. Range Distribution and Splits (Data Partitioning)

### The Problem

CockroachDB splits the keyspace into ranges (default 512 MB). The number of ranges determines parallelism but also overhead from Raft groups, heartbeats, and leaseholder management.

### The Formula

Total ranges for a table of size $S$ with range size $R$:

$$N_{\text{ranges}} = \left\lceil \frac{S}{R} \right\rceil$$

Raft heartbeat overhead per range (leader to each follower every 200ms by default):

$$\text{Heartbeat messages/s} = N_{\text{ranges}} \cdot (n - 1) \cdot \frac{1}{0.2} = 5 \cdot N_{\text{ranges}} \cdot (n-1)$$

Total Raft messages (heartbeats + proposals) per second:

$$M_{\text{total}} = N_{\text{ranges}} \cdot \left( 5(n-1) + \frac{W}{N_{\text{ranges}}} \cdot (n-1) \right)$$

where $W$ is total writes/second across all ranges.

### Worked Examples

| Table Size | Range Size | Ranges | RF=3 Heartbeats/s | RF=5 Heartbeats/s |
|:---:|:---:|:---:|:---:|:---:|
| 10 GB | 512 MB | 20 | 200 | 400 |
| 100 GB | 512 MB | 200 | 2,000 | 4,000 |
| 1 TB | 512 MB | 2,000 | 20,000 | 40,000 |
| 10 TB | 512 MB | 20,000 | 200,000 | 400,000 |

At 20,000 ranges with RF=3: 200,000 heartbeat messages/s, which is manageable but non-trivial.

## 3. Multi-Region Write Amplification (Replication Theory)

### The Problem

In REGIONAL BY ROW tables, writes to non-local regions require cross-region Raft consensus. The write amplification depends on the locality of the leaseholder.

### The Formula

Write amplification factor for a single row write:

$$WA = 1 + (n - 1) \cdot \frac{\bar{d}_{\text{cross}}}{\bar{d}_{\text{local}}}$$

where $\bar{d}_{\text{cross}}$ is the average cross-region data transfer and $\bar{d}_{\text{local}}$ is local.

For GLOBAL tables, every write goes through a non-blocking transaction protocol:

$$T_{\text{global\_write}} = T_{\text{commit}} + T_{\text{clock\_skew\_wait}}$$

$$T_{\text{clock\_skew\_wait}} = \max(0, \epsilon - T_{\text{elapsed}})$$

where $\epsilon$ is the max clock offset (default 500ms, reduced with NTP/PTP).

### Worked Examples

| Locality Mode | Write Location | Leaseholder | Write Latency |
|:---:|:---:|:---:|:---:|
| REGIONAL BY TABLE (us-east1) | us-east1 | us-east1 | ~3ms |
| REGIONAL BY TABLE (us-east1) | eu-west1 | us-east1 | ~80ms |
| REGIONAL BY ROW | Local region | Local | ~3ms |
| REGIONAL BY ROW | Remote region | Remote | ~80ms |
| GLOBAL | Any | Primary | ~600ms |

## 4. Survivability Analysis (Fault Tolerance)

### The Problem

CockroachDB offers ZONE and REGION failure survivability. The probability of data unavailability depends on the failure model and replication topology.

### The Formula

With $r$ replicas across $z$ zones, ZONE failure survivability requires:

$$r \geq 3 \quad \text{and} \quad z \geq 3$$

Probability of data loss (all replicas of a range lost) with independent zone failure probability $p_z$:

$$P_{\text{data\_loss}} = N_{\text{ranges}} \cdot p_z^{\lceil r/z \rceil \cdot z}$$

For REGION failure survivability with $R$ regions:

$$P_{\text{unavailable}} = \binom{R}{2} \cdot p_R^2 \cdot (1 - p_R)^{R-2}$$

where two simultaneous region failures are needed to lose quorum.

### Worked Examples

With 5 replicas across 3 regions, annual region failure probability $p_R = 0.01$:

$$P_{\text{unavailable}} = \binom{3}{2} \cdot 0.01^2 \cdot 0.99 = 3 \times 0.0001 \times 0.99 = 0.000297$$

| Replicas | Regions | Survive Zone | Survive Region | Annual Unavailability |
|:---:|:---:|:---:|:---:|:---:|
| 3 | 1 (3 zones) | Yes | No | ~0.03% |
| 5 | 3 | Yes | Yes | ~0.03% |
| 5 | 5 | Yes | Yes | ~0.001% |
| 7 | 3 | Yes | Yes | ~0.00003% |

## 5. Transaction Contention (Queueing Theory)

### The Problem

Under serializable isolation, concurrent transactions on the same key range may conflict. The retry rate depends on contention density and transaction duration.

### The Formula

For $\lambda$ transactions/second touching the same key, with average transaction duration $D$:

$$P_{\text{conflict}} = 1 - e^{-\lambda \cdot D}$$

The effective throughput with retries (assuming exponential backoff):

$$\text{Throughput} = \frac{\lambda}{1 + \lambda \cdot D} \approx \frac{1}{D} \quad \text{(at saturation)}$$

The abort rate:

$$\text{Abort rate} = \lambda \cdot P_{\text{conflict}} = \lambda \cdot (1 - e^{-\lambda D})$$

### Worked Examples

| TXN Rate ($\lambda$) | TXN Duration ($D$) | Conflict Prob | Effective Throughput |
|:---:|:---:|:---:|:---:|
| 10/s | 5ms | 4.9% | 9.5/s |
| 100/s | 5ms | 39.3% | 66.7/s |
| 1000/s | 5ms | 99.3% | 200/s (bottleneck) |
| 100/s | 50ms | 99.3% | 20/s (bottleneck) |
| 10/s | 50ms | 39.3% | 6.7/s |

Key insight: keep transaction duration short and distribute writes across ranges.

## 6. Follower Read Staleness (Consistency Theory)

### The Problem

Follower reads trade freshness for latency. The closed timestamp mechanism propagates a "safe to read" timestamp from the leaseholder.

### The Formula

The staleness window:

$$\Delta_{\text{stale}} = \text{closed\_ts\_interval} + \text{propagation\_delay}$$

Read latency from nearest replica vs. leaseholder:

$$T_{\text{follower}} = d_{\text{nearest}}$$

$$T_{\text{leaseholder}} = d_{\text{leaseholder}}$$

$$\text{Savings} = d_{\text{leaseholder}} - d_{\text{nearest}}$$

### Worked Examples

Default closed timestamp interval = 3s. Reader in eu-west1, leaseholder in us-east1:

| Read Type | Latency | Staleness |
|:---:|:---:|:---:|
| Leaseholder (us-east1) | 80ms | 0 |
| Follower (eu-west1 local) | 1ms | ~3-4s |
| AS OF SYSTEM TIME '-10s' | 1ms | 10s |

Savings: $80 - 1 = 79$ ms per read, at the cost of up to 4 seconds staleness.

## Prerequisites

- Raft consensus protocol and quorum mechanics
- Distributed hash tables and key-range partitioning
- Network latency models and order statistics
- Serializable isolation and optimistic concurrency control
- Clock synchronization (NTP, PTP) and hybrid logical clocks
- Probability of correlated and independent failures
