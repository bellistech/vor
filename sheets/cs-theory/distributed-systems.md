# Distributed Systems

Theorems, consistency models, replication, consensus, partitioning, time, failure handling — the full theoretical foundation behind every modern database, queue, KV store, and service mesh, with concrete reference systems for each pattern.

## Setup

A distributed system is a collection of independent nodes that appears to its users as a single coherent system, communicating over an unreliable network. The defining property is **partial failure** — some node is always failed, slow, or partitioned, and the system must still make progress. Everything in this sheet flows from that constraint.

The eight fallacies (Deutsch/Gosling, 1994) are the canonical mistakes:

| # | Fallacy | Reality |
|---|---------|---------|
| 1 | The network is reliable | Packets drop, links flap, switches reboot |
| 2 | Latency is zero | RTT is 0.1–300ms+, queueing adds tail |
| 3 | Bandwidth is infinite | TCP congestion, link saturation, BDP |
| 4 | The network is secure | MITM, replay, eavesdrop — assume hostile |
| 5 | Topology doesn't change | Routes flap, nodes scale up/down |
| 6 | One administrator | Many ops teams, many policies |
| 7 | Transport cost is zero | Egress fees, marshalling, syscalls |
| 8 | The network is homogeneous | Mixed protocols, MTU, jitter |

Three fundamental tradeoffs dominate distributed design:

| Tradeoff | Pick One |
|----------|----------|
| Consistency vs Availability | CAP under partition |
| Latency vs Consistency | PACELC normal-ops |
| Throughput vs Latency | Batching vs streaming |

```bash
# A distributed system in one definition (Lamport):
# "A distributed system is one in which the failure of a computer you
#  didn't even know existed can render your own computer unusable."
```

## Failure Models — fail-stop, fail-silent, Byzantine, omission, timing, FLP impossibility

A failure model defines what behaviours a faulty node is permitted to exhibit. Stronger models (Byzantine) require more replicas and more rounds; weaker models (fail-stop) admit simpler protocols. Choose the weakest model that matches your threat surface.

| Model | Behaviour | Detection | Example |
|-------|-----------|-----------|---------|
| **Fail-stop** | Halts, others reliably notice | Trivial | Idealised crash |
| **Fail-silent (crash)** | Halts, no announcement | Timeout-based | Real crashes |
| **Omission** | Drops messages | Hard | Lossy network |
| **Timing** | Slow / late | Hard | GC pause, paging |
| **Byzantine** | Arbitrary, including malicious | Very hard | Compromised, bug |
| **Fail-recovery** | Crashes then recovers w/ stable storage | Medium | Most real DBs |

Replica counts to tolerate `f` failures:

| Model | Replicas needed | Examples |
|-------|-----------------|----------|
| Crash-stop consensus | `2f+1` | Raft, Paxos, ZAB |
| Byzantine consensus | `3f+1` | PBFT, Tendermint, HotStuff |
| Crash-stop atomic broadcast | `2f+1` | Kafka ISR with `min.insync.replicas` |

**FLP Impossibility** (Fischer, Lynch, Paterson, 1985): in a *purely asynchronous* system with even a single crash failure, no deterministic consensus algorithm can guarantee both safety and liveness. Real systems escape FLP using:

| Workaround | Mechanism |
|------------|-----------|
| Partial synchrony (DLS 1988) | Bound after GST (Global Stabilization Time) |
| Failure detectors (CT 1996) | `◇P`, `◇S` (eventually perfect/strong) |
| Randomization (Ben-Or 1983) | Coin flip — termination w.p. 1 |
| Timing assumptions | Heartbeat / lease |

```bash
# FLP in one sentence: no asynchronous algorithm can both ensure that
# decisions are reached and that they always halt — because a single
# slow process is indistinguishable from a crashed one.
```

## Network Models — sync vs async vs partial-sync, msg-passing vs shared-mem

The network model bounds how long a message takes and whether nodes share clocks. Real systems are **partially synchronous** (Dwork-Lynch-Stockmeyer): there exists an unknown global stabilization time GST after which messages arrive within unknown but bounded delay Δ.

| Model | Message delay | Clock | Decidable? |
|-------|---------------|-------|------------|
| **Synchronous** | Bounded known Δ | Bounded skew | Trivially |
| **Partially synchronous** | Bounded after GST | Bounded after GST | Yes (Paxos) |
| **Asynchronous** | Unbounded | None | FLP says no for consensus |

Communication paradigms:

| Paradigm | Primitive | Examples |
|----------|-----------|----------|
| **Message passing** | send / receive | TCP, UDP, gRPC, AMQP |
| **RPC** | Call across process | gRPC, Thrift, JSON-RPC |
| **Shared memory** | Read / write registers | DSM, Linda, NUMA, RDMA |
| **Pub/Sub** | Topic, subscribe | Kafka, NATS, MQTT |
| **Stream** | Ordered log | Kafka, Pulsar, Kinesis |

```bash
# Async: useful proofs of impossibility (FLP, CAP)
# Partial-sync: where real protocols live (Raft, Paxos)
# Sync: hardware lockstep, in-memory, or single switch fabrics
```

## CAP Theorem — Brewer's clarification, the "during partition pick C or A"

CAP (Brewer 2000, proved Gilbert-Lynch 2002): when a network partition occurs, a system can guarantee at most **one** of:

| | Definition |
|---|---|
| **C — Consistency** | Linearizability (every read sees the most recent write) |
| **A — Availability** | Every non-failing node returns a non-error response |
| **P — Partition tolerance** | System continues to operate despite arbitrary message loss |

P is non-negotiable in a distributed system — partitions happen, so you must build for them. CAP reduces to: **during a partition, choose C or A**.

| Class | Behaviour during partition | Examples |
|-------|---------------------------|----------|
| **CP** | Reject writes (or reads) on minority side | etcd, ZooKeeper, HBase, Spanner, MongoDB (default) |
| **AP** | Serve possibly-stale reads on both sides | Cassandra, Dynamo, Riak, CouchDB |
| **CA** | Fiction unless single-node — abandon | Single-node Postgres, classic 2PC clusters |

**Brewer's 12-year clarification** (2012): "2 of 3" is misleading — in normal operation a system can offer both C and A; the choice is only forced *during* a partition. Modern systems are mode-aware: full ACID under healthy conditions, degraded mode (read-only, last-known, sloppy quorum) under partition.

```bash
# The most-quoted misunderstanding:
# "CAP says you can pick 2 of 3."
# Correct restatement:
# "When a partition happens, you must trade C for A or vice versa.
#  Outside partitions, both are achievable."
```

## PACELC — Partition then C/A Else L/C; Cassandra PA/EL, MongoDB PA/EC, Spanner PC/EC

PACELC (Abadi 2010) extends CAP: **if Partition then choose C or A, Else (normal ops) choose L (latency) or C (consistency)**. PACELC names the *normal-case* tradeoff CAP ignores.

| System | Partition mode | Normal mode | Class |
|--------|----------------|-------------|-------|
| **Cassandra** | A (sloppy quorum) | L (tunable, often eventual) | **PA/EL** |
| **DynamoDB** | A | L (eventual default) | **PA/EL** |
| **Riak** | A | L | **PA/EL** |
| **MongoDB** | A (default RC) | C (linearizable available) | **PA/EC** |
| **CouchDB** | A | L | **PA/EL** |
| **BigTable** | C | C | **PC/EC** |
| **HBase** | C | C | **PC/EC** |
| **etcd** | C | C | **PC/EC** |
| **ZooKeeper** | C | C | **PC/EC** |
| **Spanner** | C | C (TrueTime) | **PC/EC** |
| **CockroachDB** | C | C | **PC/EC** |
| **FoundationDB** | C | C | **PC/EC** |
| **VoltDB** | C | C | **PC/EC** |

```bash
# PACELC mnemonic:
#   IF Partition  → CHOOSE C or A   (CAP)
#   ELSE          → CHOOSE L or C   (latency vs consistency)
#
# Most "NoSQL" systems default to PA/EL, prioritising both availability
# under partition and latency under normal ops.
# Strongly consistent systems pay both round-trips and unavailability.
```

## Consistency Models Hierarchy — strict / linearizable / sequential / causal / RYW + monotonic-read / eventual

A consistency model defines the contract between the storage system and the application: which orderings of reads and writes are observable. Stronger models forbid more anomalies but cost more in latency and availability.

The **Bailis hierarchy** (top = strongest, bottom = weakest):

```
                    strict serializability
                            │
                       linearizable
                            │
                       sequential
                       ╱        ╲
                  causal+        per-key linearizable
                     │
        ┌────────────┼─────────────┐
        │            │             │
   read-your-     monotonic     monotonic
     writes         reads         writes      writes-follow-reads
        └────────────┴────────────┴────────────┘
                            │
                        eventual
```

| Model | Guarantees | Cost | Example |
|-------|------------|------|---------|
| **Strict serializable** | Linearizable + serializable | High | Spanner, FaunaDB |
| **Linearizable** | Single most-recent value globally | Sync majority | etcd, ZK |
| **Sequential** | Same total order on all replicas, agrees with each client's program order | Async possible | DSM models |
| **Causal+** | Happens-before preserved + convergence | Single-DC writes | COPS, Riak |
| **Read-your-writes** | Client sees own writes | Sticky session | App-level |
| **Monotonic reads** | Reads don't go backward | Same replica | App-level |
| **Eventual** | Will converge if writes stop | Anything goes | Dynamo |

```bash
# The cardinal rule:
#   Pick the *weakest* model your application can tolerate.
#   Stronger = simpler reasoning, slower, less available.
#   Weaker = harder to code, faster, more available.
```

## Linearizability — single-point-in-time abstraction; etcd, Spanner

Linearizability (Herlihy-Wing 1990): each operation appears to take effect **atomically at some single point in time between its invocation and response**, and the resulting total order respects real-time precedence (if op A returned before op B started, A precedes B).

| Property | Linearizability provides |
|----------|--------------------------|
| Recency | Reads return the latest committed write |
| Total order | All clients agree on order of ops |
| Real-time | Wall-clock-respecting |
| Composable | Linearizable objects compose |

Cost: requires a sync majority round-trip. Cannot be wait-free for both readers and writers in a partition (CAP).

| System | How it achieves linearizability |
|--------|---------------------------------|
| **etcd** | Raft consensus per write, read with `linearizable=true` (read-index) |
| **ZooKeeper** | ZAB consensus, `sync()` before read for linearizable read |
| **Spanner** | Paxos + TrueTime for external consistency |
| **CockroachDB** | Raft per-range + HLC + uncertainty intervals |
| **FoundationDB** | Resolver determines strict serializable order |

```bash
# Linearizable quorum read (etcd-style)
ETCDCTL_API=3 etcdctl --consistency=l get foo   # default; goes via leader

# Serializable (potentially stale) read
ETCDCTL_API=3 etcdctl --consistency=s get foo   # any follower, faster
```

## Sequential Consistency — Lamport's definition; weaker than linearizable

Sequential consistency (Lamport 1979): the result of any execution is the same as if the operations of all processors were executed in **some sequential order**, and the operations of each individual processor appear in this sequence in the order specified by its program.

Key difference vs linearizability: sequential allows the global order to **violate real-time** as long as program order is preserved. If client A finishes write `x=1` before client B starts read, sequential consistency permits B to read the older value, linearizable does not.

| Property | Linearizable | Sequential |
|----------|--------------|------------|
| Per-process program order | Yes | Yes |
| Real-time order | Yes | **No** |
| Total order | Yes | Yes |
| Composable | Yes | **No** |

```bash
# Example: P1 writes x=1 then x=2; P2 reads x then reads x.
# Sequential ALLOWS:   P2 sees 2 then 1   (no, violates P1 program order)
# Sequential ALLOWS:   P2 sees 1 then 2   (ok)
# Sequential ALLOWS:   P2 sees 0 then 1   (ok, even after P1 done)
# Linearizable FORBIDS: any read after P1 returns must see ≥ last value
```

## Causal Consistency — happens-before preserved

Causal consistency: if event `a → b` (Lamport's happens-before), then every node sees `a` before `b`. Concurrent events may be seen in any order. Captured with vector clocks or version vectors.

Causality preservation does not require a global order — different replicas may see concurrent writes in different orders. **Causal+** (COPS, Eiger) adds *convergent conflict handling* so concurrent writes resolve identically everywhere.

| Anomaly | Forbidden by causal? |
|---------|----------------------|
| Read of write that depended on unseen prior write | Yes |
| Two concurrent writes seen in different orders on different replicas | No (causal) / Yes (causal+) |
| Stale read (no causal dependency violated) | No — allowed |

| System | Causal mechanism |
|--------|------------------|
| **COPS** | Causal+ via dependency tracking |
| **Riak** | Vector clocks per key |
| **Bayou** | Anti-entropy + dependency check |
| **AntidoteDB** | Causal+ across DCs |

```bash
# Hand-wave example
# Alice posts: "I quit my job"     -- write A
# Alice comments: "It was awful"   -- comment depends on post, A → B
# Causal: every reader sees A before B. Some readers may not see either yet.
# Eventually-only: a reader could see B without A, exposing dangling reply.
```

## Read-Your-Writes / Monotonic-Reads / Monotonic-Writes / Writes-Follow-Reads (4 session guarantees)

Bayou's four **session guarantees** (Terry et al. 1994) are client-centric — defined per session, not globally. Each fixes a specific anomaly common in eventually-consistent systems.

| Guarantee | Forbids | Common fix |
|-----------|---------|------------|
| **Read-your-writes (RYW)** | Read missing your own write | Sticky session, read-after-write to leader |
| **Monotonic reads (MR)** | Read going backward in time | Pin to one replica per session |
| **Monotonic writes (MW)** | Writes from same client reordered | Write-ahead per-client sequence number |
| **Writes-follow-reads (WFR)** | Writes appearing before the writes you read | Track read-set causal context |

| Pattern | Anomaly without it | Code fix |
|---------|---------------------|---------|
| User edits profile then refreshes | Sees old profile | RYW: route both to primary |
| User scrolls feed | Newer page than older | MR: pin replica |
| User likes A then likes B | Replica shows B, no A | MW: per-user write ordering |
| User reads thread, replies | Reply visible w/o thread | WFR: include thread version in reply |

```bash
# In code (sticky read-your-writes):
last_write_lsn = WRITE(user_id, profile)
# Subsequent reads include the LSN; replicas with lsn < last_write_lsn defer
READ(user_id, min_lsn=last_write_lsn)
```

## Eventual Consistency — Dynamo paper; quiescence assumption

Eventual consistency: if no new updates are made, all replicas eventually converge. The Dynamo paper (DeCandia et al. 2007) popularised it for high-availability stores: prioritise A and L, accept temporary divergence, repair via anti-entropy and read-repair.

| Aspect | Detail |
|--------|--------|
| **Liveness** | Convergence assumed when writes stop ("quiescence") |
| **Safety** | None — any read may be stale |
| **Conflict resolution** | LWW, application merge, or CRDT |
| **Anomalies** | All — RYW, MR, MW, lost updates without VV |

Eventual consistency without **causal+** can expose: stale reads, lost updates, sibling explosions, dangling references, write skew amplified by replication.

```bash
# Cassandra: tunable consistency
CONSISTENCY ONE      -- write to 1 replica, return; eventual
CONSISTENCY QUORUM   -- N/2+1, strong if R+W>N
CONSISTENCY ALL      -- all replicas, no AP under partition
```

## Strong Eventual Consistency — CRDT contract

Strong Eventual Consistency (SEC, Shapiro 2011): replicas that have received the same set of updates have **the same state**, regardless of the order in which they received them. SEC + safety is the contract CRDTs satisfy.

| | Eventual | Strong eventual (SEC) |
|---|----------|------------------------|
| Convergence | Yes (if quiescent) | Yes |
| Order-independence | No | **Yes** |
| Conflict resolution | App-level / LWW | Built into data type |
| Liveness even without quiescence | No | **Yes** |

CRDTs trade rich consistency for arithmetic: only operations forming a join-semilattice can be SEC. Counters, sets, registers, maps, sequences (with caveats) all have CRDT formulations.

```bash
# G-counter (grow-only): {nodeA: 3, nodeB: 7}
# merge by element-wise max: {nodeA: max(a1,a2), nodeB: max(b1,b2)}
# value = sum of all entries
```

## Convergent vs Commutative — CvRDT vs CmRDT

Two CRDT styles produce SEC by different mechanisms:

| | CvRDT (state-based) | CmRDT (op-based) |
|---|---------------------|-------------------|
| What replicates | Full state | Operations |
| Merge | Idempotent, commutative, associative `⊔` | Operations commute pairwise |
| Network | Anti-entropy / gossip | Reliable causal broadcast |
| Cost | High bandwidth (state) | High delivery requirement |
| Examples | G-Counter, OR-Set | Op-CRDT counters, treedoc |
| Used by | Riak, AntidoteDB | Automerge, Yjs |

Both yield identical results given identical sets of updates — duality of state and operation views.

```bash
# CvRDT G-Counter pseudocode
class GCounter:
    state: dict[NodeId, int]
    def inc(self, node): self.state[node] += 1
    def value(self): return sum(self.state.values())
    def merge(self, other):
        for k,v in other.state.items():
            self.state[k] = max(self.state.get(k,0), v)
```

## Lamport Timestamps

Lamport timestamps (1978): a logical clock `L` per process, incremented on local events; on send, attach `L`; on receive, set `L = max(L_local, L_msg) + 1`. Defines a partial happens-before relation `→`.

| Property | Holds? |
|----------|--------|
| `a → b ⇒ L(a) < L(b)` | Yes |
| `L(a) < L(b) ⇒ a → b` | **No** (insufficient for concurrency detection) |
| Total order via tie-break (process id) | Yes |

Use cases: total order broadcast, mutual exclusion, deterministic event ordering when concurrency detection isn't required.

```bash
# Lamport clock pseudocode (per process p)
L = 0
def local_event():     L += 1; return L
def send(msg):         L += 1; msg.ts = L; return msg
def recv(msg):         L = max(L, msg.ts) + 1
```

## Vector Clocks

Vector clock (Mattern, Fidge 1988): `VC = [c_1, c_2, ..., c_n]` per process. On local event: `VC[self]++`. On send: attach `VC`. On receive: pointwise `VC[i] = max(VC[i], msg.VC[i])`, then `VC[self]++`.

| Comparison | Definition |
|------------|------------|
| `VC_a ≤ VC_b` | `∀i: VC_a[i] ≤ VC_b[i]` |
| `VC_a < VC_b` | `VC_a ≤ VC_b ∧ ∃i: VC_a[i] < VC_b[i]` |
| `VC_a ‖ VC_b` (concurrent) | Neither `≤` |

Vector clocks **detect** concurrent updates exactly: `a → b ⇔ VC_a < VC_b`. Cost: O(N) space per event, where N is the number of processes.

```bash
# Three nodes example
A: [1,0,0] → A sends to B → B: [1,1,0]
C: [0,0,1]                                 # concurrent with both
# B and C are concurrent: [1,1,0] ‖ [0,0,1]
```

## Version Vectors

Version vectors are vector clocks specialised to **objects**: one entry per *replica that has updated this key*, not per process. Used in Dynamo-class systems to detect sibling versions.

| | Vector clock | Version vector |
|---|--------------|------------------|
| Granularity | Per process | Per object |
| Indexed by | Process id | Replica id |
| Used in | Process-level causality | Per-key replication |
| Examples | Lattices, distributed proofs | Dynamo, Riak, Voldemort |

Sibling resolution: when two versions have incomparable VVs, both are returned to the client (Riak siblings) for application merge.

```bash
# Riak example
PUT key=cart, value=[apple], context=null
# Replica A: VV={A:1}                     [apple]
# Replica B receives gossip: VV={A:1}     [apple]
PUT key=cart, value=[apple,milk], context=null  -> A: VV={A:2} [apple,milk]
PUT key=cart, value=[apple,bread], context=null -> B: VV={B:1} [apple,bread]   # concurrent
GET key=cart -> sibling [apple,milk] VV={A:2} AND sibling [apple,bread] VV={B:1}
```

## Dotted Version Vectors

Dotted Version Vectors (Almeida 2014, Riak 2.0+) fix a flaw in classic VVs: under high concurrency a write may incorrectly subsume a concurrent write because both modify the same VV entry. DVVs add a per-write **dot** `(node, counter)` so concurrent writes from the same node are kept distinct.

| | VV | DVV |
|---|----|-----|
| Per-write identity | No | Yes (dot) |
| Sibling explosion under concurrent writes from one client | Hides | **Detects correctly** |
| Used in | Older Dynamo derivatives | Riak 2+, AntidoteDB |

```bash
# DVV-set: each version has a (node,counter) dot
# After conflicting concurrent PUTs from same client:
ver1 = {value:"a", dot:(N1,5), VV:{N1:4}}
ver2 = {value:"b", dot:(N1,6), VV:{N1:4}}    # both concurrent, both kept
```

## Hybrid Logical Clocks (HLC) — CockroachDB

HLC (Kulkarni 2014) combines physical NTP time with a logical counter to provide **bounded divergence from wall clock** while preserving causality.

`HLC = (l, c)` where `l ≥ physical_time`, `c` is logical tiebreaker.

```
on local event:
    l' = max(l, pt())
    if l' == l: c += 1 else c = 0
on send: attach (l,c)
on recv (msg.l, msg.c):
    l' = max(l, msg.l, pt())
    if l' == l == msg.l: c = max(c, msg.c) + 1
    elif l' == l:        c += 1
    elif l' == msg.l:    c = msg.c + 1
    else:                c = 0
    l = l'
```

| Property | HLC |
|----------|-----|
| Captures causality | Yes |
| Close to physical time | Bounded by max NTP skew |
| Total order via tiebreak | Yes |
| Used in | CockroachDB, YugabyteDB, MongoDB ChangeStreams |

```bash
# CockroachDB tx tx_id with HLC timestamp
SELECT cluster_logical_timestamp();   -- HLC l.c
-- l = wall-clock micros, c = 32-bit logical counter
```

## Snapshot Isolation — write-skew anomaly

Snapshot Isolation (SI, Berenson 1995): each transaction reads from a consistent snapshot taken at start; commits only if its writeset doesn't conflict with any concurrent committed writeset (first-committer-wins).

| Anomaly | Allowed under SI? |
|---------|-------------------|
| Dirty read | No |
| Non-repeatable read | No |
| Phantom | No |
| Lost update | No |
| **Write skew** | **Yes** |
| Read skew | No |

**Write skew**: two txns read overlapping data, write disjoint rows that violate an invariant.

```bash
-- Constraint: at least one doctor on call.
T1: SELECT count(*) FROM doctors WHERE on_call;   -- 2
T1: UPDATE doctors SET on_call=false WHERE id=1;
T2: SELECT count(*) FROM doctors WHERE on_call;   -- 2 (own snapshot)
T2: UPDATE doctors SET on_call=false WHERE id=2;
COMMIT both -> 0 doctors on call. SI permits this.
```

**Serializable Snapshot Isolation (SSI)** (Cahill 2008, Postgres `SERIALIZABLE`) detects rw-antidependency cycles and aborts one txn — eliminates write skew at runtime cost.

## Strict Serializable — Spanner, FaunaDB

**Serializable** = txns equivalent to *some* serial order. **Linearizable** = single-object real-time order. **Strict serializable** = both: real-time order + serial equivalence.

| Model | Real-time? | Multi-object? |
|-------|------------|---------------|
| Linearizable | Yes | No (single object) |
| Serializable | No | Yes |
| **Strict serializable** | **Yes** | **Yes** |

| System | Mechanism |
|--------|-----------|
| **Spanner** | Paxos + TrueTime commit-wait |
| **FaunaDB** | Calvin determinism |
| **CockroachDB** | HLC + uncertainty intervals + linearizable mode (`SET CLUSTER SETTING kv.transaction.linearizable = true`) |
| **FoundationDB** | Resolver-based deterministic commit |

```bash
# Spanner commit wait (TrueTime)
acquire_locks()
TT.now() returns [earliest, latest]
commit_ts = TT.now().latest
wait until TT.now().earliest > commit_ts   # ensures external consistency
write at commit_ts
release_locks()
```

## Quorum Reads/Writes — R+W>N

Dynamo-style quorum: with N replicas per key, a write succeeds if W replicas ack and a read returns the latest of R replicas.

| Inequality | Property |
|------------|----------|
| `R + W > N` | Read sees most recent write (overlap) |
| `W > N/2` | Forbids concurrent W-quorums (write linearizable for single key) |
| `R + W ≤ N` | Eventual only |

| Config | Behaviour |
|--------|-----------|
| W=N, R=1 | Read-optimized (write all, read one) |
| W=1, R=N | Write-optimized |
| W=R=⌈N/2⌉+1 | Balanced "quorum" |
| W=R=ALL | Strongly consistent, no fault tolerance |
| W=R=ONE | Eventual, max availability |

**Sloppy quorum** (Dynamo): when target replicas are unreachable, write to N alternates with **hinted handoff** — the write *appears* to count toward W but isn't on a preference-list node. Improves availability, breaks `R+W>N` guarantee until handoffs complete.

```bash
# Cassandra: per-statement consistency
CREATE TABLE kv (k text PRIMARY KEY, v text) WITH replication={'class':'SimpleStrategy','replication_factor':3};
INSERT INTO kv (k,v) VALUES ('a','1') USING CONSISTENCY QUORUM;
SELECT * FROM kv WHERE k='a' USING CONSISTENCY QUORUM;
-- N=3, R=W=2  →  R+W=4 > N=3 ✓
```

## Replication Patterns — single-leader / multi-leader / leaderless

Three top-level replication topologies — each trades availability, latency, and complexity differently.

| Pattern | Writes accepted at | Conflict possible? | Examples |
|---------|---------------------|---------------------|----------|
| **Single-leader** | One leader | No (linearizable possible) | Postgres, MySQL, MongoDB primary, Kafka partition |
| **Multi-leader** | Many leaders | Yes — needs resolution | CRDB multi-region, BDR, CouchDB, Galera |
| **Leaderless** | Any replica | Yes | Cassandra, DynamoDB, Riak, Scylla |

Replication delivery modes within each pattern:

| Mode | Latency | Durability | Loss on failover |
|------|---------|------------|------------------|
| Sync | High | High | None |
| Semi-sync | Med | Med | Bounded |
| Async | Low | Low | Possible |
| Chain replication | Increasing per hop | High | None |

```bash
# Postgres mix: leader async to standby1, sync to standby2
ALTER SYSTEM SET synchronous_standby_names = 'standby2';   # at-most-one-sync
```

## Single-Leader Replication — sync vs async, semi-sync, failover, split-brain

Single-leader (primary-secondary) is the simplest: all writes go through leader, replicas apply WAL/binlog/oplog. Reads can come from leader (linearizable) or followers (potentially stale, monotonic-read achievable per session).

| Sub-mode | Behaviour | Loss on leader crash |
|----------|-----------|------------------------|
| **Async** | Leader acks before replication | Up to in-flight WAL |
| **Semi-sync** | Wait for ≥1 follower ack | Bounded; one ack = 0 loss if follower survives |
| **Sync (chain)** | Wait for *all* followers | None, but tail-bound throughput |
| **Quorum** | Wait for k of n | Tunable |

**Failover challenges**:

| Challenge | Mitigation |
|-----------|------------|
| **Split-brain** | Fencing tokens, STONITH, lease |
| **Lost updates on async failover** | Use semi-sync; tools like `pt-slave-restart` |
| **New leader picks wrong replica (stale)** | Promote replica with highest LSN |
| **Unfenced old leader keeps accepting** | Quorum-based fencing, generation numbers |

```bash
# Postgres failover with fencing (Patroni-style)
# 1. Etcd lease expires - leader lease lost
# 2. New leader elected via etcd CAS on /leader key
# 3. Old leader detects lost lease, refuses writes (fence)
# 4. Promotion bumps timeline ID; old must re-base
```

## Multi-Leader Replication — collaborative editing, conflict resolution (LWW/custom/CRDT)

Multi-leader: multiple nodes accept writes; replicate to peers; conflicts resolved client-side or via deterministic merge. Common in multi-region setups, mobile/offline editing, and database integration.

| Conflict-resolution policy | How |
|----------------------------|-----|
| **Last-Write-Wins (LWW)** | Each write tagged with timestamp; max wins. Lossy. |
| **Per-record version vector** | Detects concurrent → app merges (Riak) |
| **CRDT** | Deterministic merge; no conflicts by construction |
| **Custom callback** | App-defined function (CouchDB, BDR) |
| **Manual** | Surface siblings to user (Riak) |

| Topology | Description | Failure mode |
|----------|-------------|--------------|
| **Star** | Hub-and-spoke | Hub SPOF |
| **Ring** | Each replicates to next | Single break stops |
| **All-to-all** | Mesh | High overhead, write amplification |

```bash
# CouchDB conflict resolution (deterministic + user override)
GET /db/doc?conflicts=true
{
  "_id":"a","_rev":"2-abc",
  "_conflicts":["2-def"]
}
# App reads both revisions, writes merged value with _rev=2-abc
```

## Leaderless Replication — Cassandra, DynamoDB, Riak

Leaderless: any replica accepts writes; coordinator forwards to N preference-list nodes; success when W ack. Reads contact R replicas, return latest. Quorum overlap (`R+W>N`) gives consistency; sloppy quorum trades it for availability.

| System | Mechanism |
|--------|-----------|
| **Cassandra** | Token ring, per-statement R/W, hinted handoff, read-repair |
| **DynamoDB** | Vnode partitioning, sloppy quorum, per-region replication |
| **Riak** | Vector clocks per object, sibling resolution, MapReduce |
| **Voldemort** | Vnode + VV |
| **ScyllaDB** | Cassandra-compatible, shard-per-core |

| Repair mechanism | When |
|------------------|------|
| **Read-repair** | Inline on read mismatch |
| **Hinted handoff** | Coordinator buffers writes for offline replicas |
| **Anti-entropy / Merkle tree** | Periodic background reconciliation |
| **Active repair / `nodetool repair`** | Operator-triggered full sync |

```bash
# Cassandra: ring with N=3, RF=3, QUORUM, hinted handoff
nodetool status      # UN = up-normal, DN = down-normal
nodetool repair      # full anti-entropy; run weekly < gc_grace_seconds
nodetool tpstats     # check HintedHandoff queue length
```

## Partitioning / Sharding — hash, range, consistent hashing, rendezvous

Partitioning splits data across nodes for capacity and parallelism. The choice of partition function dictates rebalancing cost, hotspots, and range-query feasibility.

| Strategy | Lookup | Rebalance | Range scans | Hotspots | Examples |
|----------|--------|-----------|-------------|----------|----------|
| **Hash** | O(1) | Heavy | No | Mitigated | Memcached, naive |
| **Range** | O(log N) | Split/move | **Yes** | Per-key | HBase, Spanner, CRDB, MongoDB chunks |
| **Consistent hash** | O(log N) ring | **Light** (only neighbours) | No | Vnode-mitigated | Cassandra, Dynamo, Memcached-Ketama |
| **Rendezvous (HRW)** | O(N) per key | Light | No | None | Tahoe-LAFS, some load balancers |
| **Directory / lookup** | RPC | Trivial | Yes | None | GFS master |

| Pitfall | Fix |
|---------|-----|
| Mod-N hashing remaps everything on resize | Consistent hashing |
| Hot key | Salting, secondary partition, request coalescing |
| Hot range | Pre-split on creation; range-aware splitting |
| Cross-shard transactions | 2PC, deterministic (Calvin), or design out |

## Consistent Hashing — virtual nodes

Consistent hashing (Karger 1997): map both nodes and keys onto an `m`-bit ring (often SHA-1 / SHA-256 / Murmur). A key belongs to the first node clockwise from `hash(key)`. Adding / removing a node only remaps keys assigned to the leaving region — `O(K/N)` keys move on a single node change vs `O(K)` for mod-N.

| Plain ring problem | Fix |
|---------------------|-----|
| Uneven distribution (random placement) | **Virtual nodes**: each physical node holds many tokens |
| Heterogeneous nodes | Vnode count proportional to capacity |
| Cascading load on failed neighbour | Successor list with replication factor |
| Hotspot near a token | Bounded loads (Mirrokni 2018) |

```bash
# Cassandra defaults: num_tokens = 256 vnodes per host
# Adding a new node steals 1/(N+1) of each existing node's tokens
# vs flat ring where new node would steal from one neighbour only
```

## Rendezvous Hashing — HRW

Highest Random Weight (Thaler 1996): for each key, compute `hash(key, node_i)` for all nodes; pick top-K nodes by score. No ring data structure; perfectly even distribution; replication is "top-R scores".

| | Consistent hash | Rendezvous (HRW) |
|---|------------------|--------------------|
| Lookup cost | O(log N) ring | **O(N)** per key |
| Rebalance | Vnode aware | Each key recomputes; same property |
| Distribution evenness | Vnode-dependent | **Perfect in expectation** |
| Replication picks | Successor list | **Top-R scores** |

```bash
# HRW pseudocode for replication
def top_r(key, nodes, R):
    scored = [(hash(node + key), node) for node in nodes]
    scored.sort(reverse=True)
    return [n for _,n in scored[:R]]
```

## Distributed Transactions — 2PC

Two-Phase Commit (Gray 1978): a coordinator drives all participants through prepare → commit/abort. Atomicity guaranteed; **liveness compromised** when coordinator fails between phases.

```
Phase 1 (PREPARE):
  Coord -> Participants: PREPARE
  Each: write WAL, lock rows, vote YES or NO
  If any NO or timeout: abort

Phase 2 (COMMIT/ABORT):
  Coord: write decision to log
  Coord -> Participants: COMMIT or ABORT
  Each: apply, release locks
```

| Failure | Effect |
|---------|--------|
| Participant before vote | Coord aborts after timeout |
| Participant after vote=YES, before commit | **Blocked**, holds locks until coord recovers |
| Coord after collecting votes, before decision | Participants wait — *blocking protocol* |
| Coord after decision, before sending | Recovers via log; participants wait |

| Improvement | What it adds |
|-------------|--------------|
| Presumed-abort | Coord forgets aborts; participants assume abort if no record |
| Cooperative termination | Participants ask peers; needs peer addresses |
| 3PC | Pre-commit phase removes blocking under crash-stop |

```bash
# XA transactions in MySQL
XA START 'tx1';
UPDATE accounts SET bal=bal-100 WHERE id=1;
XA END   'tx1';
XA PREPARE 'tx1';   -- vote YES, lock survives crash
XA COMMIT  'tx1';   -- or XA ROLLBACK 'tx1'
```

## 3PC — adds pre-commit

Three-Phase Commit (Skeen 1981) inserts a **pre-commit** step between prepare and commit. Goal: any participant can recover decision from peers without waiting for coordinator, eliminating blocking under crash-stop with synchronous network.

```
Phase 1 (CAN-COMMIT): votes
Phase 2 (PRE-COMMIT): all participants ack readiness
Phase 3 (DO-COMMIT):  apply
```

| Property | 2PC | 3PC |
|----------|-----|-----|
| Blocking under coord crash | **Yes** | No (under sync net + crash-stop) |
| Round-trips | 2 | 3 |
| Tolerates network partition | No | **No** (assumed sync) |
| Used in production | Common | Rarely (assumptions too strong) |

In practice 3PC is theoretically interesting but rarely deployed: real networks aren't synchronous, so 3PC's blocking-freedom doesn't hold. Modern systems prefer Paxos/Raft commit (replicated decision log) over either 2PC or 3PC.

## Paxos / Raft — pointer to dedicated sheets

Paxos (Lamport 1998) and Raft (Ongaro 2014) are the two canonical consensus algorithms for crash-stop, partial-sync systems. Both achieve agreement on a totally ordered log of commands using `2f+1` replicas to tolerate `f` failures, in `O(1)` rounds in steady state.

| | Paxos | Raft |
|---|-------|------|
| Year | 1998 | 2014 |
| Roles | Proposer, Acceptor, Learner | Leader, Follower, Candidate |
| Optimisation | Multi-Paxos for log | Strong leader from day one |
| Pedagogical clarity | Low | **High** |
| Implementations | Chubby, Spanner, ZK ZAB | etcd, Consul, CockroachDB, TiKV, Nomad |
| Throughput in steady state | One-round per write | One-round per write |

```bash
# Both protocols documented in dedicated sheets:
cs distributed-consensus
cs paxos
cs raft
```

## Saga Pattern — compensating actions; orchestration vs choreography

A **saga** (Garcia-Molina 1987) replaces a long-running ACID transaction with a sequence of local txns, each with a compensating action. If step `n` fails, sagas run compensations `C_{n-1}, C_{n-2}, ..., C_1` in reverse order.

| | Orchestration | Choreography |
|---|----------------|----------------|
| Coordinator | Central orchestrator | None — events drive each step |
| Coupling | Tight to orchestrator | Tight to event schema |
| Visibility | High (one place) | Low — distributed across services |
| Failure handling | Centralised compensation | Each handler reacts |
| Examples | Camunda, Temporal | Pure Kafka pub/sub |

| Saga property | Note |
|----------------|------|
| Atomicity | **Not** atomic — partial states observable |
| Isolation | None — anti-pattern: pessimistic locks |
| Compensation | Must be idempotent + retriable |
| Semantic locking | Reservation pattern hides intermediate state |

```bash
# Order saga: 5 steps, 5 compensations
T1 reserve_credit  -> C1 release_credit
T2 reserve_inventory -> C2 release_inventory
T3 charge_payment    -> C3 refund
T4 ship_order        -> C4 cancel_shipment
T5 send_confirmation -> C5 send_cancellation
```

## TCC — Try-Confirm-Cancel

Try-Confirm-Cancel (Pat Helland's pattern) is a structured saga: each participant exposes three idempotent endpoints. The orchestrator drives Try on all, then either Confirm-all on success or Cancel-all on failure.

| Phase | Semantics | Failure |
|-------|-----------|---------|
| **Try** | Reserve resources, optimistic lock | Reservation timeout if no Confirm/Cancel |
| **Confirm** | Apply reservation to permanent state | Must succeed eventually (retry indefinitely) |
| **Cancel** | Release reservation | Must succeed eventually |

| | 2PC | TCC |
|---|-----|-----|
| Layer | DB / XA | Application |
| Locks | Held across phases (pessimistic) | Reservations (optimistic) |
| Network failures | Block | Timeout + cancel/retry |
| Heterogeneous services | Hard | **Natural** |

```bash
# TCC interface example (microservice, Spring/Cloud-Tencent style)
POST /tcc/balance/try    {tx, account, amount}     # reserve
POST /tcc/balance/confirm {tx}                      # commit reservation
POST /tcc/balance/cancel  {tx}                      # release reservation
```

## Leader Election — Bully, Ring, Raft randomized timer, ZAB

Many systems require a single leader at a time (single-writer, scheduling, locks). Election protocols differ in topology, message complexity, and assumptions.

| Algorithm | Topology | Messages | Assumptions | Examples |
|-----------|----------|----------|-------------|----------|
| **Bully** (Garcia-Molina) | Mesh | O(N²) worst | Synchronous, IDs | Classic |
| **Ring** | Logical ring | O(N²) | Sync, ring intact | Token rings |
| **Raft randomized timer** | Mesh | O(N) | Partial sync | Raft, etcd, Consul |
| **ZAB** | Mesh | Quorum | Partial sync | ZooKeeper |
| **Paxos election** | Mesh | Round-trips | Partial sync | Multi-Paxos |
| **Chubby lock service** | Centralised | Lease | Partial sync | Google Chubby |
| **Bakery / token** | Shared mem | Variable | Shared registers | DSM models |

**Split-brain prevention**:

| Mechanism | How |
|-----------|-----|
| Quorum (≥N/2+1) | Two halves can't both pass |
| Lease | Old leader times out |
| Fencing tokens | Monotonic ID checked at storage |
| STONITH | Power-cycle losing node |
| Generation / epoch | Increment per election; old epochs rejected |

```bash
# Raft election rule of thumb
# Election timeout: 150-300 ms randomized (avoid synchronized splits)
# Heartbeat: 50 ms typical
# leader_only_during_majority_partition_with_active_quorum
```

## Consensus — FLP impossibility, partial-sync workarounds

Consensus: agreement, validity (decided value was proposed), termination (every non-faulty process decides). FLP says termination is impossible in pure async with even one crash. All real consensus algorithms relax one assumption.

| Workaround | Algorithm | Example |
|------------|-----------|---------|
| **Partial synchrony + ◇P failure detector** | Paxos, Raft | etcd, Consul |
| **Probabilistic termination** | Ben-Or, randomized | Algorand, Avalanche |
| **Stronger model (sync)** | Synchronous consensus | Theoretical |
| **Trust assumption** | PBFT, HotStuff | Tendermint, Diem |

| Family | Tolerates | Replicas needed for f |
|--------|-----------|------------------------|
| Crash | f crash failures | 2f+1 |
| Byzantine | f arbitrary failures | 3f+1 |
| Crash + omission | f | 2f+1 |
| Hybrid | f crash + g Byzantine | depends |

```bash
# In one line:
#   Consensus is solvable in partial-sync with crash failures
#   using Paxos/Raft and a quorum of 2f+1 replicas.
```

## Failure Detection — heartbeats, Phi-accrual, gossip, hinted handoff

Failure detectors in async networks can only be **eventually** correct (Chandra-Toueg classification). Real systems combine timeouts with adaptive thresholds and gossip dissemination.

| Detector | Mechanism | Use |
|----------|-----------|-----|
| **Heartbeat fixed timeout** | Last-seen + Δ | Simple, Δ tuning fragile |
| **Phi-accrual** (Hayashibara) | Suspicion as continuous score from inter-arrival distribution | Cassandra, Akka |
| **SWIM** (Das) | Indirect probe via random peers | Hashicorp memberlist, Consul |
| **Gossip-based** | Disseminate liveness via O(log N) rounds | Cassandra, Riak |
| **Lease** | Time-bound grant; expiry = failure | Chubby, ZooKeeper, etcd |
| **Quorum-based** | Majority must agree | Raft cluster membership |

| Class | Properties |
|-------|------------|
| `P` (perfect) | Strong completeness + accuracy — needs sync |
| `S` (strong) | Strong completeness + eventual strong accuracy |
| `◇P` (eventually perfect) | Both eventual |
| `◇S` (eventually strong) | What Paxos/Raft need |

```bash
# Cassandra phi_convict_threshold (default 8)
# Higher = slower but fewer false positives during GC pauses
# Distribution of inter-arrival times -> phi(now) score
# convict when phi > threshold
nodetool gossipinfo  # raw heartbeat state per host
```

## Gossip Protocols — SWIM, plumtree, HyParView

Gossip protocols disseminate state across N nodes in O(log N) rounds with O(N) total messages, tolerating arbitrary failures and partitions probabilistically. Used for membership, anti-entropy, broadcast, aggregation.

| Protocol | Purpose | Property |
|----------|---------|----------|
| **Push gossip** | Broadcast | O(log N) rounds; latent fanout |
| **Pull gossip** | Anti-entropy | Same; receiver-driven |
| **Push-pull** | Both | Strictly better termination |
| **SWIM** (Das 2002) | Membership | Indirect probe; suspect/alive/confirm |
| **HyParView** | Overlay | Partial views, robust to churn |
| **Plumtree** | Bcast tree | Eager + lazy push for efficiency |
| **Cyclon** | Random peer sampling | Continuous shuffling |

```bash
# SWIM round
# 1. Pick random peer P; send PING
# 2. If no ACK in T1: pick K random peers, send PING-REQ(P)
# 3. If still no ACK: mark P SUSPECT, gossip
# 4. After T2: confirm DEAD if not refuted; gossip
```

## Anti-Entropy — Merkle trees, read-repair, Cassandra-style three-tier

Anti-entropy reconciles divergent replicas. Three-tier strategy in Dynamo-class systems:

| Tier | Mechanism | Cost | When |
|------|-----------|------|------|
| **Read-repair** | Mismatch detected at read; coordinator pushes latest | Per read | Inline |
| **Hinted handoff** | Coordinator buffers write for unreachable replica | Per write | On detect |
| **Merkle tree repair** | Compare hashes per range, sync mismatches | O(log N) per disagreement | Periodic / manual |

**Merkle tree** (Merkle 1979): binary hash tree over key ranges. Two replicas can compare root hashes; if equal — done; otherwise descend to differing subtrees. Bandwidth O(D · log N) where D is number of differing leaves.

```bash
# Cassandra: scheduled anti-entropy via nodetool repair
nodetool repair -pr keyspace1 table1   # primary range repair
nodetool repair -inc                    # incremental; tracks which sstables already repaired
# Repair must complete within gc_grace_seconds (10 days default)
# else tombstones may resurrect zombie data
```

## Idempotency — exactly-once myth, dedup-by-id, UUIDv7/KSUID/ULID

**Exactly-once delivery is impossible** over an unreliable network in finite time (Kafka FAQ, "two armies"). What real systems achieve: **effectively-once = at-least-once delivery + idempotent processing**.

| Approach | Mechanism |
|----------|-----------|
| **Dedup by message id** | Consumer keeps recent ids in cache/DB |
| **Idempotent operations** | `SET x = 5` is naturally idempotent vs `INCR x` |
| **Upsert with version check** | `UPDATE ... WHERE version = expected` |
| **Idempotency-Key header** | Stripe-style; server stores response keyed by client UUID |
| **Two-phase / reservation** | Try → confirm; safe to retry both phases |

| Identifier | Sortable | Length | Source |
|------------|----------|--------|--------|
| **UUIDv4** | No | 128b | Random |
| **UUIDv1** | Yes (MAC + time) | 128b | MAC, sometimes leaked |
| **UUIDv6** | Yes (time first) | 128b | Time-reordered v1 |
| **UUIDv7** | **Yes (Unix ms first)** | 128b | Unix ms + random |
| **ULID** | Yes (48b time + 80b random) | 26 chars b32 | 2016 spec |
| **KSUID** | Yes | 27 chars b62 | Segment |
| **Snowflake** | Yes | 64b | Twitter |
| **TSID** | Yes | 64b | Truncated UUIDv7 |

```bash
# Idempotency key in HTTP (Stripe pattern)
POST /v1/charges
Idempotency-Key: 7e1f9a4b-6c0e-4f2c-9a05-7c1f9e0a1234
# server stores 200 response keyed by ID for 24h; replays return same response
```

## Distributed Locks — Redlock debate, ZooKeeper ephemeral, etcd lease, fencing tokens

Distributed locks coordinate exclusive access across processes. Three families with different correctness guarantees:

| Implementation | Mechanism | Safety |
|----------------|-----------|--------|
| **Single Redis SETNX + TTL** | Lock = key, TTL prevents permanent lock | Unsafe under GC pause / clock skew |
| **Redlock** (5 Redis nodes) | Acquire on majority within validity window | **Disputed** — Kleppmann critique |
| **ZooKeeper ephemeral seq-node** | Lock = lowest seq node; ephemeral so death releases | Safe with fencing |
| **etcd lease** | KV with lease TTL; revoke on death; CAS to acquire | Safe with fencing |
| **Consul lock / sessions** | KV + session | Safe with fencing |
| **DB row lock** | `SELECT ... FOR UPDATE` | Single-node only |

**Fencing tokens** (Kleppmann): every successful acquire returns a monotonic token. Storage rejects writes with stale tokens. Without fencing, *any* lock service is unsafe under GC pause.

```
client A: lock(K) -> token=33
client A: GC pauses 60s
client A: lock expires at 30s
client B: lock(K) -> token=34
client B: writes data (token=34)
client A: resumes, writes (token=33)  -> storage rejects: stale token
```

```bash
# etcd lock with lease + fencing
LEASE_ID=$(etcdctl lease grant 30 | awk '/granted/ {print $3}')
etcdctl lock --lease=$LEASE_ID /lock/job1 -- ./run-job.sh
# /lock/job1 carries lease; on death it auto-deletes; fencing via revision
```

## Distributed Counter — G/PN-Counter, Cassandra counter

A distributed counter without coordination needs CRDT semantics — independent increments must merge associatively, commutatively, idempotently.

| Counter | Operations | Mechanism |
|---------|------------|-----------|
| **G-Counter** | Increment only | Per-replica counters; value = sum |
| **PN-Counter** | +/- | Two G-counters: `pos[i] - neg[i]` |
| **Cassandra counter** | `+/-` | Coordinator-mediated, *not* CRDT — read-modify-write with paxos under hood |
| **Bounded counter** | +/- with cap | Reservations |

```bash
# G-Counter merge across 3 replicas
node1: {n1:5, n2:3, n3:2}     -> 10
node2: {n1:5, n2:4, n3:2}     -> 11
node3: {n1:5, n2:3, n3:2}     -> 10
merge: {n1:5, n2:4, n3:2}     -> 11   (element-wise max)
```

```bash
# Cassandra counter (NOT CRDT; coordinator path)
CREATE TABLE views (page text PRIMARY KEY, n counter);
UPDATE views SET n = n + 1 WHERE page='home';
-- reads serial; not idempotent on retry; use idempotent retry framework
```

## Sequencing — Snowflake, UUID v1/v4/v6/v7, ULID, KSUID

Choosing an ID format is a tradeoff between sortability, randomness, length, dependence on coordination, and information leakage.

| Scheme | Sortable? | Coordination? | Bits | Notes |
|--------|-----------|---------------|------|-------|
| **Auto-increment** | Yes | Per-DB lock | 64 | Bottleneck at scale |
| **Snowflake (Twitter)** | Yes (ms+seq) | Per-worker id | 64 | 41b ms + 10b worker + 12b seq |
| **UUIDv1** | Time-ordered | None | 128 | Leaks MAC |
| **UUIDv4** | No | None | 128 | 122 random bits |
| **UUIDv6** | Yes | None | 128 | UUIDv1 reordered for sort |
| **UUIDv7** | Yes | None | 128 | Unix ms + random; **recommended** |
| **ULID** | Yes | None | 128 | 26 chars b32; lexicographic |
| **KSUID** | Yes | None | 160 | 4b time + 16b random |
| **NanoID** | No | None | Variable | URL-safe |
| **TSID** | Yes | Optional node id | 64 | Compact UUIDv7-like |

```bash
# Snowflake bit layout (64 bits)
# 1 sign | 41 ms timestamp | 10 worker | 12 sequence
# 2^41 ms = ~69 years; 2^10 = 1024 workers; 2^12 = 4096 ids/ms/worker
# Roughly 2^22 = 4M ids/sec/cluster
```

## Time — NTP drift, leap seconds, Spanner TrueTime

Wall-clock time on commodity hardware drifts. NTP-synced hosts typically maintain ~1–100 ms accuracy; PTP brings sub-microsecond on a LAN; GPS gives nanosecond on a co-located server.

| Source | Typical accuracy | Caveat |
|--------|------------------|--------|
| Local quartz | 10⁻⁶ relative — drifts ~1s/day | No external sync |
| NTP (public) | 5–100 ms | Asymmetric paths skew |
| NTP (LAN) | 1–10 ms | Stable |
| PTP / IEEE 1588 | < 1 µs | LAN, hardware support |
| GPS | 100 ns | Antenna, atomic-disciplined |
| **Spanner TrueTime** | 7 ms ε bound | Datacenter-engineered |

| Hazard | Effect |
|--------|--------|
| **NTP step** | `clock_gettime` jumps backward → broken `time.now()` ordering |
| **Leap second** | June/December insertions; many bugs (Linux 2012, Reddit) |
| **VM pause / migration** | Stops time for seconds |
| **Stratum cycle** | Two NTP servers point at each other |
| **Smearing** (Google) | Spread leap second over 24h to avoid 60-second |

**Spanner TrueTime** API exposes `TT.now()` returning `[earliest, latest]` interval. Spanner waits out the uncertainty (commit-wait) so commit timestamps are *guaranteed* to be after every prior committed transaction in real time.

```bash
# Linux clock APIs and their behaviour
clock_gettime(CLOCK_REALTIME)     # wall clock; can jump backward
clock_gettime(CLOCK_MONOTONIC)    # monotonic since boot; never backward
clock_gettime(CLOCK_MONOTONIC_RAW) # not slewed by NTP
clock_gettime(CLOCK_BOOTTIME)     # includes suspend time
```

## Service Discovery — DNS, etcd/Consul/ZooKeeper, eBPF service mesh

Service discovery answers "which IPs implement service X right now?" for dynamic environments.

| Mechanism | Push/Pull | TTL | Examples |
|-----------|-----------|-----|----------|
| **Static config** | n/a | n/a | hosts file |
| **DNS A/SRV** | Pull | TTL-bound staleness | Round-robin DNS, k8s ClusterDNS |
| **DNS-SD / mDNS** | Pull/multicast | Zeroconf | Avahi, Bonjour |
| **Consul / etcd / ZK** | Pull + watch | Lease | Hashicorp stack |
| **Eureka (Netflix)** | Heartbeat + pull | Heartbeat | OSS-Java |
| **Kubernetes Endpoints** | Push (watch) | Lease | k8s |
| **Service mesh (Envoy/Linkerd)** | xDS push | Continuous | Istio |
| **eBPF (Cilium, Tetragon)** | Kernel hooks | Live | Cilium ServiceMesh |

```bash
# Consul DNS interface
dig @127.0.0.1 -p 8600 web.service.consul
# Returns A records of healthy instances

# k8s headless service for endpoint enumeration
dig my-svc.default.svc.cluster.local
```

## Load Balancing — round-robin, weighted, least-conn, consistent hashing, power-of-two

Load balancers distribute requests across replicas to spread load and isolate failures.

| Algorithm | Property | Use |
|-----------|----------|-----|
| **Round-robin** | Stateless, even by request count | Equal-cost backends |
| **Weighted RR** | Weighted by capacity | Heterogeneous |
| **Least-connections** | Active conn count | Long-lived conns |
| **Least-time** (HAProxy) | Active + RTT | Mixed latency |
| **Hash by source IP** | Affinity | Sticky session |
| **Consistent hash** | Cache locality, minimal churn | Caches, stateful |
| **Power-of-two choices** | Pick 2 random, send to less loaded | **Near-optimal**, simple |
| **JSQ (join-shortest-queue)** | Pick least loaded | Optimal but expensive (O(N)) |
| **EWMA** (Envoy) | Exponentially-weighted moving avg of RTT | High concurrency |
| **Maglev** (Google) | Even consistent hash | Ingress |

**Power-of-two-choices** (Mitzenmacher 2001): random pick 2 backends, send to the less-loaded. Achieves O(log log N) max load vs O(log N / log log N) for pure RR. Used by Nginx `least_conn` variants, Envoy.

```bash
# Nginx
upstream backend {
    least_conn;          # connection-aware
    server b1:8080 weight=3;
    server b2:8080 weight=1;
    server b3:8080 backup;
    keepalive 64;
}
```

## Circuit Breaker — closed/open/half-open

Circuit breaker (Nygard, *Release It!*) prevents cascading failure: when downstream errors exceed a threshold, fail fast for a cool-down, then test cautiously.

| State | Behaviour | Transition |
|-------|-----------|------------|
| **Closed** | Pass requests; count failures | If failure rate > T → Open |
| **Open** | Fail-fast immediately; no calls | After cool-down → Half-open |
| **Half-open** | Allow N probe calls | All succeed → Closed; any fail → Open |

| Tunable | Typical |
|---------|---------|
| Failure threshold | 50% over rolling 10s window |
| Min sample | 20 requests |
| Open duration | 30s exponential backoff |
| Half-open probes | 5 |

```bash
# Hystrix / resilience4j / Polly all expose:
# .failureRateThreshold(50)
# .slidingWindow(SECONDS, 10)
# .minimumNumberOfCalls(20)
# .waitDurationInOpenState(30s)
# .permittedCallsInHalfOpenState(5)
```

## Retry Patterns — exponential backoff with jitter, AIMD, jitter types

Retries amplify load during outages. Always combine retries with backoff and jitter; never retry idempotent-only operations on non-idempotent endpoints.

| Backoff | Formula |
|---------|---------|
| **Constant** | `delay = c` |
| **Linear** | `delay = c * attempt` |
| **Exponential** | `delay = base * 2^attempt` |
| **Exponential capped** | `delay = min(cap, base * 2^attempt)` |

| Jitter type | Formula |
|-------------|---------|
| **None** | Bad — thundering herds |
| **Full jitter** | `delay = random(0, cap_at_attempt)` |
| **Equal jitter** | `delay = base + random(0, base)` |
| **Decorrelated jitter** | `delay = min(cap, random(base, prev_delay * 3))` |

**AIMD** (Additive-Increase Multiplicative-Decrease, used by TCP and adaptive rate limiters): increase rate by `+α` each ok, multiply by `β<1` on failure. Converges to fair sharing.

```bash
# AWS recommended: full jitter
backoff = min(cap, base * 2 ** attempt)
sleep   = random(0, backoff)
```

## Backpressure — flow control, reactive streams

Backpressure: a slow consumer signals upstream to slow down, preventing buffer overflow and OOM. Without it, queues grow unbounded and tail latency explodes (queueing law).

| Mechanism | Where |
|-----------|-------|
| **TCP windowing** | Transport |
| **HTTP/2 flow control** | App-transport |
| **gRPC unary/streaming credits** | App |
| **Reactive Streams `request(n)`** | RxJava, Project Reactor, Akka Streams |
| **Kafka consumer max.poll.records** | Pull-based naturally backpressured |
| **AMQP prefetch (`basic.qos`)** | RabbitMQ |
| **Token bucket / rate limit** | API gateway, Envoy |
| **Semaphore / bounded queue** | App-internal |

| Anti-pattern | Symptom |
|--------------|---------|
| Unbounded queue | OOM, p99 inflation |
| Drop tail | Latency-sensitive traffic dies |
| RED / fair queueing | Better fairness |

```bash
# Reactive Streams contract (1-line summary)
# Subscriber requests N; Publisher emits at most N until next request(N)
subscriber.request(64);   // never push beyond 64 unrequested
```

## Bulkhead — resource isolation

Bulkhead pattern (ship compartments): partition resources so that exhaustion in one workload cannot starve others. Implemented via thread pools, semaphores, connection pools, or processes.

| Variant | Mechanism |
|---------|-----------|
| **Thread pool per dependency** | Hystrix-style; isolates blocking calls |
| **Semaphore per dependency** | Lower overhead, no pool |
| **Connection pool per upstream** | DBCP, HikariCP per service |
| **OS process / container** | Kubernetes resource limits |
| **Per-tenant queue** | Multi-tenant API gateway |

| Without bulkhead | With bulkhead |
|------------------|---------------|
| Slow `recommendations` exhausts shared 200-thread pool → all endpoints time out | `recommendations` pool of 50 saturates; other endpoints unaffected |

```bash
# Resilience4j bulkhead
BulkheadConfig.custom()
    .maxConcurrentCalls(25)        # semaphore
    .maxWaitDuration(0ms)          # fail-fast if full
    .build();
```

## Sidecar — envoy, istio, linkerd

Sidecar pattern: deploy a helper container alongside the main app (same pod / VM), handling cross-cutting concerns transparently.

| Use | Sidecar |
|-----|---------|
| L7 proxy (mTLS, retries, LB) | Envoy, Linkerd-proxy |
| Logs / metrics | Fluentd, Vector, OTel collector |
| Service identity | SPIRE agent |
| Secrets injection | Vault Agent |
| Database proxy | Cloud SQL Auth Proxy, ProxySQL |
| Service mesh data plane | istio-proxy, linkerd-proxy |

| Pros | Cons |
|------|------|
| Language-agnostic | Resource overhead per pod |
| Centralised config | Operational complexity |
| Upgrade independent | Coordination on lifecycle |

```bash
# Kubernetes pod with sidecar
spec:
  containers:
    - name: app
      image: myapp:1.0
    - name: envoy
      image: envoyproxy/envoy:v1.31
      args: ["-c", "/etc/envoy/config.yaml"]
```

## Outbox Pattern — atomic write + CDC

Outbox: services need to update DB and publish an event atomically. Distributed txns across DB and broker are fragile. Solution: write event to an `outbox` table in the **same transaction** as the business write; a CDC reader (Debezium / poller) tails the table and publishes.

| Step | Detail |
|------|--------|
| 1 | App opens DB tx |
| 2 | Insert business row |
| 3 | Insert event row in `outbox` |
| 4 | Commit |
| 5 | Outbox publisher (CDC or polling) reads from log |
| 6 | Publishes to broker; marks delivered |

| Variant | Mechanism |
|---------|-----------|
| **Polling outbox** | `SELECT ... WHERE delivered=false LIMIT N FOR UPDATE SKIP LOCKED` |
| **CDC** | Tail Postgres WAL via Debezium / logical replication |
| **Listen/Notify** | Postgres `LISTEN outbox` for low-latency wake |

```bash
-- Outbox table
CREATE TABLE outbox (
    id BIGSERIAL PRIMARY KEY,
    aggregate_type TEXT, aggregate_id TEXT,
    type TEXT, payload JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    published BOOLEAN DEFAULT false
);
-- Application txn writes to both business table and outbox in one commit
```

## Event Sourcing

Event sourcing: persist every state change as an immutable event in an append-only log; current state is a fold over the event stream. Provides full audit, time-travel, and natural CQRS.

| | Traditional CRUD | Event Sourcing |
|---|------------------|------------------|
| State of record | Latest row | Event log |
| Update | UPDATE row | Append event |
| History | Lost / audit table | Inherent |
| Replay | Hard | Trivial |
| Query | Direct | Project to read model |
| Schema evolution | Migrate rows | Upcasters / versioned events |

| Concern | Pattern |
|---------|---------|
| Events = past tense | `OrderShipped`, not `ShipOrder` |
| Snapshots | Store state every N events to bound replay |
| Event versioning | Upcast in code, never modify history |
| Multiple aggregates | Eventual consistency, sagas |
| GDPR / right to erasure | Crypto-shredding (forget keys) |

```bash
# EventStoreDB / Marten / Axon style
StreamId: "order-7e4"
Events:
  v1: OrderCreated{customer:abc, items:[...]}
  v2: PaymentReceived{amount:99}
  v3: OrderShipped{tracking:ABC123}
  v4: DeliveryConfirmed{at:...}
```

## CQRS

Command-Query Responsibility Segregation: separate write model (commands → events) from read models (denormalised views). Often paired with event sourcing but doesn't require it.

| Side | Optimised for | Tech |
|------|---------------|------|
| **Command** | Validation, invariants | RDBMS, ES log |
| **Read** | Query patterns | Elasticsearch, Redis, materialised views |
| **Sync** | Eventual consistency | Events, CDC, projections |

| When CQRS earns its keep | When not |
|--------------------------|----------|
| Read patterns far exceed writes | CRUD admin tool |
| Multiple read shapes per aggregate | Single view |
| Independent scaling needed | Small system |

```bash
# Order service: write side
POST /orders   -> appends OrderCreated to event store
# Read side projector
on OrderCreated -> upsert into Postgres orders_view
# Query
GET /orders/by-customer/abc -> reads orders_view
```

## Streaming — Kafka log-structured, exactly-once via transactional writes

Streaming systems treat data as an unbounded ordered log. Kafka popularised the **log-structured** model: partitioned, append-only, replicated, retained.

| Concept | Kafka semantics |
|---------|-----------------|
| **Topic** | Logical stream |
| **Partition** | Ordered shard |
| **Offset** | Byte / record position |
| **ISR** | In-sync replicas (≥ `min.insync.replicas`) |
| **Producer ack** | `acks=0` / `1` / `all` |
| **Idempotent producer** | Per-PID + sequence dedup |
| **Transactions** | `transactional.id` spans multi-partition send |
| **Exactly-once (EOS)** | Idempotent producer + transactions + read-committed |
| **Compaction** | Retain last value per key forever |
| **Tiered storage** | Hot in broker, cold in S3 |

```bash
# Producer config for at-least-once durability
acks=all
enable.idempotence=true
max.in.flight.requests.per.connection=5
retries=Integer.MAX
```

```bash
# Producer-consumer EOS (Kafka Streams)
processing.guarantee=exactly_once_v2
isolation.level=read_committed
# Reads only events from committed transactions
```

## Read Repair / Hinted Handoff / Anti-Entropy

Three reconciliation primitives in eventually-consistent systems work together:

| Mechanism | Trigger | Cost | What it fixes |
|-----------|---------|------|---------------|
| **Read-repair** | On read mismatch among R replicas | Per read | Single divergent key |
| **Hinted handoff** | Coord detects offline replica during write | Per write | Recent writes to dead replica |
| **Active anti-entropy / Merkle repair** | Scheduled or operator | O(log N) per disagreement | All divergence including silent corruption |

| Combined | Coverage |
|----------|----------|
| Read-repair only | Hot data; cold data drifts forever |
| HH only | Recent writes; older drift unfixed |
| Anti-entropy only | Eventual but high latency |
| All three | Cassandra, Riak, Dynamo standard |

```bash
# Cassandra config knobs
read_repair_chance: 0.1            # background read-repair probability
dclocal_read_repair_chance: 0.1
hinted_handoff_enabled: true
max_hint_window_in_ms: 10800000    # 3h before drop hints
gc_grace_seconds: 864000           # 10d; repair must complete before
```

## Tail Latency — p99/p99.9, hedged requests, deadline propagation

In a request-fanout system, the **slowest** sub-call dominates user-perceived latency. With 100 sub-calls, a per-call p99 of 10 ms gives a p100-of-fanout near 50 ms. Tail-latency engineering is essential.

| Technique | What |
|-----------|------|
| **Hedged requests** (Dean) | Issue duplicate after p95 wait; cancel slower |
| **Tied requests** | Both servers know about each other; first to start cancels other |
| **Backup requests** | Send to two from start; cheaper for low-fanout |
| **Deadline propagation** | gRPC deadline shrinks as it travels; servers shed work past deadline |
| **Adaptive timeouts** | EWMA RTT + cancel after kσ |
| **Load shedding** | Reject excess at source instead of queueing |
| **Co-location of fanout** | Reduce per-call variability |
| **Cache warmth** | Avoid cold paths in tail |

```bash
# Google Search style hedged read (illustrative)
result = race(call(replicaA), delayed(p95_ms, call(replicaB)))
# Cuts p99 dramatically at ~5% extra load
```

```bash
# gRPC deadline propagation
call_options.deadline = now + 200ms
# every downstream call uses min(its-default, remaining-budget - safety)
```

## Observability — distributed tracing (OTel/Jaeger/Tempo), trace_id propagation

Three pillars: logs, metrics, traces. Distributed tracing reconstructs causal chains across services using a propagated `trace_id` and per-call `span_id`.

| Concept | Definition |
|---------|------------|
| **Trace** | Tree/DAG of spans, single trace_id |
| **Span** | One unit of work; parent_span_id links |
| **Baggage** | Key/value carried alongside trace |
| **Sampling** | Head-based vs tail-based |
| **OTel** | Vendor-neutral standard (OpenTelemetry) |

| Header | Standard |
|--------|----------|
| `traceparent: 00-<trace>-<span>-<flags>` | W3C Trace Context |
| `tracestate: vendor=...` | W3C Trace Context |
| `b3` / `X-B3-*` | Zipkin |
| `X-Datadog-Trace-Id` | Datadog |
| `uber-trace-id` | Jaeger legacy |

| Backend | Storage |
|---------|---------|
| Jaeger | Cassandra / Elasticsearch / OpenSearch |
| Tempo | Object store (S3, GCS) |
| Zipkin | Cassandra, ES, MySQL |
| Datadog APM | Hosted |
| Honeycomb | Hosted, columnar |
| AWS X-Ray | Hosted |

```bash
# Propagate trace context with curl
curl -H 'traceparent: 00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01' \
     https://api.example.com/checkout
```

## Common Errors — partition-vs-slow confusion, wall-clock ordering, exactly-once myth

| Error | Why wrong |
|-------|-----------|
| "Node is partitioned" | Often it's just slow — indistinguishable from outside |
| "We need exactly-once delivery" | Impossible; achieve effectively-once via idempotency |
| "Use wall-clock for ordering" | Skew; use logical clocks or HLC |
| "CAP says pick 2 of 3" | Misreading; choice is during partition |
| "Quorum guarantees consistency" | Only with `R+W>N` and *not* sloppy quorum |
| "Locks across services are safe" | Only with fencing tokens |
| "Eventual consistency = stale by some seconds" | Could be never if writes never quiesce |
| "Microservices solve coupling" | They move it to network; harder to debug |
| "Retry until success" | Storms outages; needs backoff + cap |
| "Master-slave failover is automatic" | Risk of split-brain without quorum |
| "We have ACID across services" | No — only sagas/TCC |

## Common Gotchas — 8+ broken→fixed pairs

```bash
# 1. Mod-N hashing rebalances everything on cluster resize.
# Broken
node = hash(key) % N
# Fixed: consistent hashing with vnodes
node = ring.lookup(hash(key))   # only K/N keys move on resize
```

```bash
# 2. Exactly-once delivery via "just retry until ok".
# Broken
while not ok: send(msg)             # duplicates downstream
# Fixed: idempotent processing
send(msg)                           # at-least-once
process(msg) only if msg.id not in seen_ids
```

```bash
# 3. Naive Redis lock under GC pause.
# Broken
SET lock:k unique EX 30 NX
do_work_60s()                       # lock expired at 30; another holds it
# Fixed: fencing token + storage-side check
token = etcd_lease_acquire("k", 30)
storage.write(data, fence=token)    # rejects stale tokens
```

```bash
# 4. Wall-clock for event ordering.
# Broken
ts = time.time()                    # NTP step can move backward
events.sort(key=ts)
# Fixed: HLC / Lamport clock
ts = hlc.now()
events.sort(key=ts)
```

```bash
# 5. Sloppy quorum mistaken for strong quorum.
# Broken: assumes R+W>N when sloppy
# Fixed: explicit strict quorum
INSERT ... USING CONSISTENCY QUORUM AND NOT SLOPPY  -- (conceptual)
# Or accept eventual and add anti-entropy
```

```bash
# 6. Microservice DB-per-service plus distributed txn.
# Broken
BEGIN
  service_a.update()
  service_b.update()
COMMIT                              -- 2PC across services, fragile
# Fixed: outbox + saga
write outbox event in same DB tx
saga orchestrator drives next steps
```

```bash
# 7. Read-after-write to async replica returns stale.
# Broken
WRITE to leader
READ from any replica               -- replica lag → old data
# Fixed: route reads-after-writes to leader, or use LSN
last_lsn = write_to_leader()
read_from_replica(min_lsn=last_lsn)
```

```bash
# 8. Vector clock unbounded growth across actors.
# Broken
VV grows by one entry per ever-active client    # millions of entries
# Fixed: dotted version vector + actor pruning
DVV per write; actor list pruned by tombstone after retention
```

```bash
# 9. Health check that only ICMP-pings.
# Broken
ping host  # OS up but app deadlocked
# Fixed: app-level health (ready vs live), with k8s readinessProbe
GET /health/ready -> 200 only when fully serving
```

```bash
# 10. Single-region active-passive DR with async replication.
# Broken
sync = async; failover loses last seconds of data
# Fixed: semi-sync min.insync=2, or quorum across regions
synchronous_standby_names = 'standby_in_other_az'  # at least one ack
```

```bash
# 11. Saga compensation that's not idempotent.
# Broken
def cancel_shipment(): mark_cancelled()  # called twice → error
# Fixed: idempotent
def cancel_shipment(tx_id):
    if already_cancelled(tx_id): return
    mark_cancelled(tx_id)
```

```bash
# 12. ID generated client-side as random UUIDv4 used as primary key.
# Broken
id = uuid4()                 # random → terrible B-tree locality
# Fixed: UUIDv7 / ULID
id = uuid7()                 # time-prefixed; sequential inserts
```

## Idioms — simplest consistency model, monotonic reads in user-facing

A handful of pragmatic rules of thumb that keep distributed systems sane:

| Idiom | Why |
|-------|-----|
| Pick the **simplest** consistency model your app can tolerate | Stronger costs latency and availability |
| Default to **monotonic reads** for any user-facing list/feed | Scrolling backwards is jarring |
| Make every API endpoint **idempotent** by accepting `Idempotency-Key` | Retries are inevitable |
| Stamp every message with a unique id and sender clock | Enables dedup, replay, debugging |
| Always pair retries with **backoff + jitter** | Avoid thundering herds |
| Set a **deadline** on every cross-service call | Prevents cascade |
| Use **fencing tokens** with all distributed locks | GC pauses and clock skew exist |
| Prefer **append-only** logs to in-place updates for audit-able state | Easier replay & debugging |
| Treat **clocks as advisory**, never authoritative for ordering | Use logical clocks |
| Never trust a **single node**'s view of the world | Quorum or gossip |
| Health check should be **deep** (touches deps) for serving readiness | ICMP/TCP up != app ready |
| Replication lag must be **observable** | Otherwise stale reads invisible |
| In multi-region, route writes to **closest leader**, reads to any | Latency vs consistency |
| Keep **schemas backward and forward compatible** | Rolling deploys cross versions |

## Reference Systems — Spanner, CockroachDB, Cassandra, DynamoDB, etcd, Consul, ZooKeeper, Kafka, Redis, ScyllaDB, FaunaDB, FoundationDB

Quick characterisation of major systems against the dimensions in this sheet.

| System | Class | Consistency | Replication | Partition | Consensus |
|--------|-------|-------------|-------------|-----------|-----------|
| **Spanner** | PC/EC | Strict serializable | Paxos per shard | Range | Paxos + TrueTime |
| **CockroachDB** | PC/EC | Serializable (default) | Raft per range | Range | Raft + HLC |
| **YugabyteDB** | PC/EC | Serializable | Raft per tablet | Hash/range | Raft + HLC |
| **TiDB** | PC/EC | Snapshot / serializable | Raft per region | Range | Raft + Percolator |
| **FoundationDB** | PC/EC | Strict serializable | Replicated logs | Range | Resolver + 2-phase commit |
| **FaunaDB** | PC/EC | Strict serializable | Calvin determinism | Range | Calvin |
| **VoltDB** | PC/EC | Serializable | K-safety | Hash | Single-thread per partition |
| **Cassandra** | PA/EL | Tunable (RYW with QUORUM/SERIAL) | Leaderless N copies | Consistent hash + vnodes | Per-partition Paxos for LWT |
| **ScyllaDB** | PA/EL | Same as Cassandra | Leaderless | Consistent hash | Same |
| **DynamoDB** | PA/EL | Tunable, default eventual; strong on demand | Leaderless | Consistent hash | None for normal; transactions use coordinator |
| **Riak KV** | PA/EL | Eventual w/ DVV; bucket-tunable | Leaderless | Consistent hash | None |
| **MongoDB** | PA/EC | Configurable read/write concerns up to linearizable | Single-leader replica set | Range / hash | Raft-like |
| **etcd** | PC/EC | Linearizable | Raft | Single-keyspace | Raft |
| **Consul** | PC/EC | Linearizable | Raft | Single | Raft |
| **ZooKeeper** | PC/EC | Sequential (linearizable w/ sync) | ZAB | Single | ZAB |
| **Kafka** | PC/EC per partition | Linearizable per partition | ISR | Topic partitions | KRaft (Raft) since 3.x |
| **Pulsar** | PC/EC | Linearizable per topic | BookKeeper ledgers | Topic | ZooKeeper / metadata via Raft |
| **NATS Jetstream** | PC/EC | Linearizable per stream | Raft | Stream | JetStream Raft |
| **Redis (single)** | CA-ish | Linearizable single-thread | Async replica | None | None |
| **Redis Cluster** | PA/EC default | Async; not linearizable | Single-leader per slot | 16384 hash slots | Failover via gossip |
| **Redis Enterprise** | tunable | tunable | sync optional | shards | RDM consensus |
| **Memcached** | none | None | None | Consistent hash client | None |
| **Aerospike** | PA/EC strong-on-master | Strong via Strong-Consistency mode | Single-leader per partition | Hash | Custom |
| **Elasticsearch** | PA/EC | Eventual; refresh interval | Primary-replica | Hash | Master via Raft (Zen2 / cluster coord) |
| **InfluxDB Cluster** | depends | Eventual | Series partitioning | Time + tag | Raft |
| **Couchbase** | PA/EC | Tunable XDCR | Single-master | Hash | None for KV; Raft for management |
| **CouchDB** | PA/EL | Eventual via MVCC + replication | Multi-master | Database | None |
| **Cosmos DB** | configurable 5 levels | Strong / Bounded staleness / Session / Consistent prefix / Eventual | Multi-region | Range / hash | Internal |

```bash
# Mental map: latency floor of strong consistency
# Within a DC:    1 RTT (~ 0.5 ms LAN)
# Cross-DC:       1 RTT (~ 5–80 ms)
# Cross-region:   1 RTT (~ 100–250 ms)
# Cross-continent:1 RTT (~ 150–300 ms)
# A linearizable write is bounded below by the slowest WAN RTT in your quorum.
```

```bash
# Characteristic-by-pattern matrix (cheat-sheet within a cheat-sheet)
# Want low-latency global writes:        AP system + CRDTs (Riak, AntidoteDB)
# Want strong global txns:                Spanner-class (CockroachDB, Spanner)
# Want OLTP single-region transactions:   Postgres / MySQL with semi-sync replica
# Want huge time-series throughput:       Cassandra / Scylla / InfluxDB
# Want ordered event log:                 Kafka / Pulsar / Redpanda
# Want ephemeral coordination:            etcd / ZK / Consul
# Want global edge cache:                 CDN + DNS-based geo-routing
```

## See Also

- distributed-consensus
- cap-theorem
- paxos
- crdt
- algorithm-patterns
- database-theory
- queueing-theory

## References

- Kleppmann, *Designing Data-Intensive Applications* (DDIA), O'Reilly 2017 — the canonical synthesis
- Tanenbaum & Van Steen, *Distributed Systems: Principles and Paradigms*, 3rd ed.
- Lynch, *Distributed Algorithms*, Morgan Kaufmann 1996
- Corbett et al., *Spanner: Google's Globally-Distributed Database*, OSDI 2012
- DeCandia et al., *Dynamo: Amazon's Highly Available Key-value Store*, SOSP 2007
- Ongaro & Ousterhout, *In Search of an Understandable Consensus Algorithm (Raft)*, USENIX ATC 2014
- Lamport, *The Part-Time Parliament (Paxos)*, ACM TOCS 1998
- Lamport, *Paxos Made Simple*, 2001
- Fischer, Lynch, Paterson, *Impossibility of Distributed Consensus with One Faulty Process*, JACM 1985
- Dwork, Lynch, Stockmeyer, *Consensus in the Presence of Partial Synchrony*, JACM 1988
- Chandra, Toueg, *Unreliable Failure Detectors for Reliable Distributed Systems*, JACM 1996
- Gilbert, Lynch, *Brewer's Conjecture and the Feasibility of CAP*, SIGACT 2002
- Brewer, *CAP Twelve Years Later: How the "Rules" Have Changed*, Computer 2012
- Abadi, *Consistency Tradeoffs in Modern Distributed Database System Design (PACELC)*, Computer 2012
- Bailis & Ghodsi, *Eventual Consistency Today: Limitations, Extensions, and Beyond*, ACM Queue 2013
- Shapiro et al., *Conflict-Free Replicated Data Types*, INRIA 2011
- Lamport, *Time, Clocks, and the Ordering of Events in a Distributed System*, CACM 1978
- Mattern, *Virtual Time and Global States*, 1988
- Almeida, *Dotted Version Vectors: Logical Clocks for Optimistic Replication*, 2010 / 2014
- Kulkarni, *Logical Physical Clocks (HLC)*, OPODIS 2014
- Karger et al., *Consistent Hashing and Random Trees*, STOC 1997
- Thaler & Ravishankar, *A Name-Based Mapping Scheme for Rendezvous*, 1996
- Garcia-Molina & Salem, *Sagas*, SIGMOD 1987
- Helland, *Life Beyond Distributed Transactions*, CIDR 2007
- Skeen, *Nonblocking Commit Protocols*, SIGMOD 1981
- Castro & Liskov, *Practical Byzantine Fault Tolerance (PBFT)*, OSDI 1999
- Yin et al., *HotStuff: BFT Consensus with Linearity and Responsiveness*, PODC 2019
- Das, Gupta, Motivala, *SWIM: Scalable Weakly-consistent Infection-style Membership Protocol*, DSN 2002
- Hayashibara et al., *The Phi Accrual Failure Detector*, 2004
- Berenson et al., *A Critique of ANSI SQL Isolation Levels*, SIGMOD 1995
- Cahill, Röhm, Fekete, *Serializable Isolation for Snapshot Databases*, SIGMOD 2008
- Mitzenmacher, *The Power of Two Random Choices: A Survey of Techniques and Results*, 2001
- Dean & Barroso, *The Tail at Scale*, CACM 2013
- Nygard, *Release It! Design and Deploy Production-Ready Software*, Pragmatic 2007/2018
- Kreps, *The Log: What every software engineer should know about real-time data's unifying abstraction*, 2013
- Burrows, *The Chubby Lock Service for Loosely-Coupled Distributed Systems*, OSDI 2006
- Hunt et al., *ZooKeeper: Wait-free coordination for Internet-scale systems*, USENIX ATC 2010
- Junqueira et al., *ZAB: High-performance broadcast for primary-backup systems*, DSN 2011
- Terry et al., *Session Guarantees for Weakly Consistent Replicated Data*, PDIS 1994
- Lloyd et al., *Don't Settle for Eventual: Scalable Causal Consistency for Wide-Area Storage with COPS*, SOSP 2011
- Kleppmann, *How to do distributed locking*, blog 2016 — Redlock critique
- Antirez (Sanfilippo), *Is Redlock safe?* response, blog 2016
- Helland, *Idempotence Is Not a Medical Condition*, ACM Queue 2012
- jepsen.io — testing reports for Cassandra, MongoDB, etcd, CockroachDB, Spanner, RabbitMQ, Kafka, Redis, et al.
- aphyr.com — Kyle Kingsbury's distributed-systems blog (Jepsen)
- Pat Helland, *Standing on Distributed Shoulders of Giants*, ACM Queue 2016
- Microsoft Research, *Calvin: Fast Distributed Transactions for Partitioned Database Systems*, SIGMOD 2012
- IETF RFC 9562 — *Universally Unique IDentifiers (UUIDs)* (v6/v7/v8 standardisation)
- IETF RFC 5905 — *Network Time Protocol Version 4*
- IEEE 1588 — *Precision Time Protocol*
- W3C Trace Context — *traceparent / tracestate* spec
- OpenTelemetry specification (opentelemetry.io)
- *The Raft consensus algorithm* — raft.github.io
- *Designing Data-Intensive Applications* references list — comprehensive bibliography
