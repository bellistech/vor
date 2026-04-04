# The Mathematics of etcd — Raft Consensus, MVCC, and Cluster Quorum

> *etcd provides strong consistency through Raft consensus and MVCC storage. The math covers Raft leader election timing, log replication latency, quorum requirements, MVCC revision space, compaction cost, and watch notification complexity.*

---

## 1. Raft Consensus (Distributed Systems)

### Quorum Requirement

For a cluster of $n$ nodes, the quorum (majority) is:

$$Q = \left\lfloor \frac{n}{2} \right\rfloor + 1$$

### Fault Tolerance

$$F = n - Q = \left\lfloor \frac{n - 1}{2} \right\rfloor$$

| Cluster Size ($n$) | Quorum ($Q$) | Fault Tolerance ($F$) | Even Size Waste |
|:---:|:---:|:---:|:---:|
| 1 | 1 | 0 | - |
| 2 | 2 | 0 | 1 node wasted |
| 3 | 2 | 1 | - |
| 4 | 3 | 1 | 1 node wasted |
| 5 | 3 | 2 | - |
| 7 | 4 | 3 | - |

Even-sized clusters tolerate the same failures as $n-1$ but cost an extra node.

### Leader Election Timing

The election timeout is randomized in $[T_{\min}, T_{\max}]$:

$$P(\text{split vote}) \approx \frac{1}{n} \times \left(\frac{T_{\text{heartbeat}}}{T_{\max} - T_{\min}}\right)^{n-1}$$

Recommended: $T_{\min} = 10 \times T_{\text{RTT}}$, $T_{\max} = 2 \times T_{\min}$.

| RTT | $T_{\min}$ | $T_{\max}$ | Expected Election Time |
|:---:|:---:|:---:|:---:|
| 1 ms (same DC) | 10 ms | 20 ms | ~15 ms |
| 10 ms (cross AZ) | 100 ms | 200 ms | ~150 ms |
| 50 ms (cross region) | 500 ms | 1000 ms | ~750 ms |

---

## 2. Write Latency (Replication)

### Commit Latency

A write is committed when a quorum acknowledges the log entry:

$$T_{\text{commit}} = T_{\text{leader\_append}} + \text{Quantile}_{Q-1}(T_{\text{replication}})$$

This is the $(Q-1)$-th fastest replication time (wait for quorum, not all nodes).

### Worked Examples

For a 3-node cluster ($Q = 2$), we wait for the 1st (fastest) follower:

$$T_{\text{commit}} = T_{\text{local}} + \min(T_{\text{follower}_1}, T_{\text{follower}_2})$$

For a 5-node cluster ($Q = 3$), we wait for the 2nd fastest follower:

$$T_{\text{commit}} = T_{\text{local}} + \text{2nd smallest}(T_{f_1}, T_{f_2}, T_{f_3}, T_{f_4})$$

| Cluster | Follower RTTs | Commit Latency |
|:---:|:---|:---:|
| 3-node (same DC) | 1ms, 1ms | ~1.5 ms |
| 3-node (cross AZ) | 2ms, 5ms | ~2.5 ms |
| 5-node (cross AZ) | 1ms, 2ms, 5ms, 10ms | ~2.5 ms |
| 5-node (cross region) | 1ms, 2ms, 50ms, 100ms | ~2.5 ms |

### Write Throughput

$$\lambda_{\text{max}} = \frac{\text{Pipeline Depth}}{T_{\text{commit}}}$$

With Raft batching (pipeline depth $P$):

| Commit Latency | Pipeline | Max Writes/sec |
|:---:|:---:|:---:|
| 1 ms | 1 | 1,000 |
| 1 ms | 100 | 100,000 |
| 5 ms | 100 | 20,000 |
| 50 ms | 100 | 2,000 |

---

## 3. MVCC and Revision Space (Versioning)

### Revision Model

Each write creates a new revision. The revision space grows monotonically:

$$R_{\text{current}} = R_0 + W_{\text{total}}$$

where $W_{\text{total}}$ = total number of write operations since creation.

### Storage per Revision

$$\text{MVCC storage} = \sum_{r=R_{\text{compacted}}}^{R_{\text{current}}} S_r$$

where $S_r$ = size of changes at revision $r$.

### Compaction Savings

$$\text{Reclaimable} = \sum_{r=R_{\text{oldest}}}^{R_{\text{compact\_target}}} S_r$$

| Write Rate | Duration | Avg Write Size | Uncompacted Size |
|:---:|:---:|:---:|:---:|
| 100/s | 1 hour | 500 B | 180 MB |
| 100/s | 24 hours | 500 B | 4.3 GB |
| 1,000/s | 1 hour | 200 B | 720 MB |
| 1,000/s | 24 hours | 200 B | 17.3 GB |

---

## 4. Watch Scalability (Event Streaming)

### Watch Model

Each watch registers interest in a key or prefix. The server maintains:

$$M_{\text{watch}} = W \times (K_{\text{avg}} + O_{\text{watch}})$$

where $W$ = number of active watches, $O_{\text{watch}}$ = per-watch overhead (~200 bytes).

### Event Fan-Out

When a key changes, all matching watches receive the event:

$$\text{Fan-out} = |\{w : w.\text{prefix} \sqsubseteq \text{key}\}|$$

$$\text{Notification cost} = F \times S_{\text{event}}$$

| Active Watches | Avg Fan-Out | Events/sec | Notification Bandwidth |
|:---:|:---:|:---:|:---:|
| 100 | 1 | 100 | 50 KB/s |
| 10,000 | 5 | 1,000 | 2.5 MB/s |
| 100,000 | 10 | 10,000 | 50 MB/s |

### Watch Memory (Kubernetes Scale)

| K8s Pods | Watches (est.) | Watch Memory |
|:---:|:---:|:---:|
| 1,000 | ~5,000 | ~2 MB |
| 10,000 | ~50,000 | ~20 MB |
| 100,000 | ~500,000 | ~200 MB |

---

## 5. Lease Management (TTL)

### Lease Renewal

A lease with TTL $T$ must be renewed before expiry:

$$T_{\text{renew\_interval}} < T - T_{\text{max\_latency}}$$

$$\text{Safety margin} = T - T_{\text{renew\_interval}}$$

### Lease Count and Overhead

$$\text{Lease memory} = L \times (O_{\text{lease}} + K_L \times P)$$

where $L$ = number of leases, $K_L$ = keys per lease, $P$ = pointer size.

| Leases | Keys/Lease | TTL | Renewals/sec |
|:---:|:---:|:---:|:---:|
| 100 | 1 | 10s | 10 |
| 1,000 | 5 | 30s | 33 |
| 10,000 | 1 | 15s | 667 |
| 100,000 | 1 | 60s | 1,667 |

### Lease Expiry Storm

If all leases have the same TTL $T$ and are created at the same time:

$$\text{Simultaneous expirations} = L \quad \text{(all at } t_0 + T\text{)}$$

Stagger lease creation or use jittered TTLs:

$$T_i = T + \text{Uniform}(-J, J)$$

---

## 6. Disk I/O — WAL and Snapshots (Storage)

### WAL Write Amplification

$$\text{WAL bytes/write} = H_{\text{entry}} + K + V + \text{CRC}$$

$$\text{WAL throughput} = \lambda \times (H + K_{\text{avg}} + V_{\text{avg}} + 4)$$

### fsync Latency Impact

$$T_{\text{write}} = T_{\text{raft}} + T_{\text{fsync}}$$

| Disk Type | fsync Latency | Max Writes/sec |
|:---:|:---:|:---:|
| NVMe SSD | 0.1-0.5 ms | 10,000+ |
| SATA SSD | 0.5-2 ms | 2,000-5,000 |
| HDD | 5-15 ms | 200-500 |
| Network (EBS gp3) | 1-5 ms | 1,000-3,000 |

### Snapshot Sizing

$$S_{\text{snapshot}} \approx N_{\text{keys}} \times (K_{\text{avg}} + V_{\text{avg}} + O)$$

| Keys | Avg Key | Avg Value | Snapshot Size |
|:---:|:---:|:---:|:---:|
| 10,000 | 50 B | 200 B | ~2.5 MB |
| 100,000 | 100 B | 500 B | ~60 MB |
| 1,000,000 | 100 B | 1 KB | ~1.1 GB |

---

## Prerequisites

distributed-systems, consensus-algorithms, data-structures, kubernetes

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| Put (committed) | O(log n) + RTT | O(K + V) per revision |
| Get (single key) | O(log n) B-tree | O(V) |
| Get (prefix scan) | O(log n + k) | O(k * V) |
| Watch (register) | O(1) | O(K) per watch |
| Watch (notify) | O(F) fan-out | O(F * V) |
| Compact | O(R) revisions | O(1) |
| Defrag | O(D) data size | O(D) temp copy |
| Snapshot save | O(D) | O(D) |
