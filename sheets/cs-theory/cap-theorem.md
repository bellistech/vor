# CAP Theorem and Consistency Models

A practitioner's reference for distributed consistency theory -- from Brewer's CAP conjecture and its proof to consistency model spectra, CRDTs, quorum protocols, anti-entropy mechanisms, and the distinction between consistency and consensus.

## The CAP Theorem

### Statement (Brewer, 2000; Gilbert-Lynch, 2002)

```
In an asynchronous network subject to partitions, a distributed system
cannot simultaneously provide all three of:

  C -- Consistency    Every read receives the most recent write (linearizability)
  A -- Availability   Every non-failing node returns a response to every request
  P -- Partition      The system continues to operate despite network partitions
       Tolerance

"Pick two" is misleading.  Partitions are not optional in real networks.
The actual choice is: when a partition occurs, choose C or A.

  During partition:
    CP system  -->  sacrifices availability (refuses some requests)
    AP system  -->  sacrifices consistency (serves stale/divergent data)
    CA system  -->  impossible in the presence of network partitions
```

### Gilbert-Lynch Proof Intuition

```
Model: asynchronous network, at least two nodes (n1, n2), messages can be lost.

Assume a partition separating n1 and n2.

  1. Client writes value v1 to n1
  2. n1 cannot communicate with n2 (partition)
  3. Client reads from n2

If consistent (C):  n2 must return v1, but it never received the write
                    --> n2 must block/refuse --> not available (not A)

If available (A):   n2 must respond --> returns stale value
                    --> not consistent (not C)

Contradiction: cannot have both C and A during a partition.
QED (by contradiction under partition assumption)
```

### Real-World System Classification

```
CP systems (sacrifice availability during partitions):
  HBase, MongoDB (w:majority), ZooKeeper, etcd, Consul, Spanner,
  CockroachDB, FoundationDB, Redis (Sentinel w/ WAIT)

AP systems (sacrifice consistency during partitions):
  Cassandra (default), DynamoDB, Riak, CouchDB, Voldemort,
  Aerospike (AP mode), DNS, Eureka

Nuances:
  - Most systems are configurable along the C-A spectrum
  - Cassandra with QUORUM reads/writes behaves more CP-like
  - MongoDB with w:1 behaves more AP-like
  - "CP" or "AP" is a simplification; real behavior depends on config
```

## PACELC Extension (Abadi, 2012)

```
If Partition:
  choose Availability or Consistency          (PA or PC)
Else (normal operation):
  choose Latency or Consistency               (EL or EC)

Full classification:  PA/EL, PA/EC, PC/EL, PC/EC

System          | P?  | A/C | E?  | L/C
----------------|-----|-----|-----|-----
DynamoDB        | PA  |     | EL  |       (favor availability and latency)
Cassandra       | PA  |     | EL  |       (tunable per query)
Riak            | PA  |     | EL  |
MongoDB         | PC  |     | EC  |       (strong consistency, higher latency)
HBase           | PC  |     | EC  |
Spanner         | PC  |     | EC  |       (TrueTime enables external consistency)
PNUTS (Yahoo)   | PC  |     | EL  |       (timeline consistency)
Cosmos DB       | PA  |     | EL  |       (5 tunable consistency levels)

Key insight: even without partitions, there is a latency-consistency tradeoff.
PACELC captures the "normal case" that CAP ignores.
```

## Consistency Models Spectrum

### From Strongest to Weakest

```
Strict Consistency (Linearizability)
  |   Every operation appears to take effect at a single instant between
  |   its invocation and response.  Real-time order is preserved.
  |   Requires: coordination (consensus or single leader)
  |
Sequential Consistency
  |   All processes see the same interleaving of operations.
  |   The interleaving respects per-process order but NOT real-time order.
  |   Weaker than linearizability: no real-time constraint.
  |
Causal Consistency
  |   Operations causally related are seen in the same order by all.
  |   Concurrent (causally independent) operations may be seen in any order.
  |   Requires: causal dependency tracking (vector clocks, version vectors)
  |
Read-Your-Writes
  |   A process always sees its own prior writes.
  |   Does NOT guarantee other processes see those writes.
  |
Monotonic Reads
  |   If a process reads value v, subsequent reads never return older values.
  |
Monotonic Writes
  |   Writes from a single process are applied in order everywhere.
  |
Eventual Consistency
      If no new writes occur, all replicas eventually converge.
      No bound on convergence time.  Weakest useful guarantee.
```

### Session Guarantees (Terry et al., 1994)

```
Four guarantees a client session can request:

  1. Read Your Writes    After writing x, session reads reflect x
  2. Monotonic Reads     Successive reads in session never go backward
  3. Writes Follow Reads Writes are ordered after causally preceding reads
  4. Monotonic Writes    Writes within session are applied in order

These are composable: a system can provide any subset.
All four together approximate causal consistency for a single session.
```

## Consistency vs Consensus

```
Consistency:  Agreement on the VALUES that reads return
              (a property of the data store / replication protocol)
              Examples: linearizability, sequential, causal, eventual

Consensus:    Agreement on a single DECISION among distributed processes
              (a protocol for reaching agreement)
              Examples: Paxos, Raft, Viewstamped Replication, PBFT

Relationship:
  Consensus can be used to IMPLEMENT strong consistency (linearizability).
  But consistency can be achieved without consensus (e.g., CRDTs for eventual).

  Linearizability = consensus equivalence (Herlihy & Wing, 1990):
    A linearizable register is equivalent in power to consensus.
    Both are impossible in asynchronous systems with failures (FLP).
```

## CRDTs (Conflict-Free Replicated Data Types)

### Overview (Shapiro et al., 2011)

```
CRDTs are data structures that can be replicated across nodes,
updated independently and concurrently, and always converge to
a consistent state WITHOUT coordination.

Two formulations:
  CvRDT (state-based):  Ship full state, merge via join-semilattice
  CmRDT (op-based):     Ship operations, require causal delivery

Convergence guarantee:
  All replicas that have received the same SET of updates
  are in the same state, regardless of ORDER of delivery.
```

### Common CRDT Types

```
G-Counter (Grow-only Counter)
  Structure:   vector of counts, one entry per node
  Increment:   node i increments its own entry
  Value:       sum of all entries
  Merge:       pointwise max
  Example:     {A:3, B:2, C:5} merged with {A:1, B:4, C:5} = {A:3, B:4, C:5}
               value = 12

PN-Counter (Positive-Negative Counter)
  Structure:   two G-Counters: P (increments) and N (decrements)
  Increment:   add to P
  Decrement:   add to N
  Value:       sum(P) - sum(N)
  Merge:       merge P and N independently (pointwise max each)

LWW-Register (Last-Writer-Wins Register)
  Structure:   (value, timestamp) pair
  Update:      set value with current timestamp
  Merge:       keep the entry with the higher timestamp
  Caveat:      requires synchronized clocks; concurrent writes: one is silently lost

OR-Set (Observed-Remove Set)
  Structure:   set of (element, unique-tag) pairs
  Add:         insert (element, new-unique-tag)
  Remove:      remove all (element, *) pairs currently observed
  Merge:       union of all pairs
  Property:    add wins over concurrent remove (add-wins semantics)
               Solves the add-remove concurrency problem of naive sets
```

## Dynamo-Style Quorum Protocols

### Quorum Configuration

```
N = number of replicas for each key
W = number of replicas that must acknowledge a write
R = number of replicas that must respond to a read

Consistency guarantee:
  R + W > N   -->  read and write quorums overlap
                   at least one node in the read set has the latest write
                   enables "read repair" to return most recent value

Common configurations:
  N=3, W=2, R=2  -->  strong read consistency (overlap = 1)
  N=3, W=3, R=1  -->  fast reads, slow writes
  N=3, W=1, R=3  -->  fast writes, slow reads
  N=3, W=1, R=1  -->  no overlap, eventual consistency only
```

### Sloppy Quorum and Hinted Handoff

```
Problem: strict quorum requires specific N nodes to be reachable.
         During partitions, writes may fail even with W < N.

Sloppy quorum:
  Write to ANY W healthy nodes (not necessarily the N "home" nodes).
  Increases availability at the cost of consistency.

Hinted handoff:
  If a home node is unreachable, write to a substitute node with a "hint."
  The hint records the intended recipient.
  When the home node recovers, the substitute forwards the data.
  Provides availability during transient failures.
```

## Anti-Entropy Mechanisms

### Merkle Trees (Hash Trees)

```
Purpose: efficiently detect differences between replicas.

Structure:
  Leaf nodes:   hash of each key-value pair (or key range)
  Internal:     hash of concatenated child hashes
  Root:         single hash representing entire dataset

Comparison protocol:
  1. Exchange root hashes
  2. If roots match --> replicas are identical, done
  3. If roots differ --> recursively compare children
  4. Descend to leaves to find exactly which keys differ
  5. Synchronize only the differing keys

Complexity: O(log n) hash comparisons to find differences
            (vs O(n) for full dataset comparison)
```

### Gossip Protocols (Epidemic Protocols)

```
Purpose: disseminate state updates across all nodes.

Push gossip:
  Each round, a node selects a random peer and sends its updates.

Pull gossip:
  Each round, a node asks a random peer for any updates it is missing.

Push-pull:
  Combine both: exchange digests, then exchange missing updates.

Properties:
  Convergence: O(log n) rounds for n nodes (epidemic spread)
  Robustness:  tolerates node failures; no single point of failure
  Scalability: each node communicates with O(1) peers per round
  Consistency: probabilistic -- not guaranteed at any specific time

Uses: failure detection, membership, metadata propagation
Examples: Cassandra (gossip for membership), Amazon S2S, SWIM protocol
```

## Jepsen Testing

```
Jepsen (Kyle Kingsbury / Aphyr):
  A framework for testing distributed systems under network partitions,
  clock skew, process pauses, and other real-world failure modes.

Approach:
  1. Set up a distributed system on multiple nodes
  2. Run concurrent operations (reads, writes, CAS)
  3. Inject faults: network partitions, node kills, clock skew
  4. Record a history of operations (invocations and responses)
  5. Check if the history is linearizable (using Knossos checker)

Key findings (selected):
  System         | Issue Found
  ---------------|---------------------------------------------------
  MongoDB 3.4    | Stale reads under partition with w:majority
  Cassandra LWT  | Lost updates under partition
  etcd 3.1       | Stale reads during leader election
  Redis Sentinel | Split-brain data loss
  CockroachDB    | Serialization anomalies (fixed in later versions)
  PostgreSQL     | Serializable mode correct (gold standard)

Linearizability checker:
  Knossos uses the Wing-Gong algorithm (NP-complete in general)
  to verify if an observed history is linearizable.
  Practical for short histories; combinatorial explosion for long ones.
```

## Key Figures

| Name | Contribution | Year |
|---|---|---|
| Eric Brewer | CAP conjecture (keynote at PODC 2000) | 2000 |
| Seth Gilbert | Formal proof of CAP theorem (with Lynch) | 2002 |
| Nancy Lynch | CAP proof, FLP impossibility (with Fischer, Paterson) | 1985/2002 |
| Daniel Abadi | PACELC extension of CAP | 2012 |
| Marc Shapiro | CRDTs (conflict-free replicated data types) | 2011 |
| Werner Vogels | "Eventually Consistent" paper, Amazon CTO | 2008 |
| Leslie Lamport | Sequential consistency, Paxos, logical clocks | 1979 |
| Maurice Herlihy | Linearizability definition (with Wing) | 1990 |
| Kyle Kingsbury | Jepsen distributed systems testing | 2013 |
| Doug Terry | Session guarantees, Bayou system | 1994 |
| Giuseppe DeCandia | Dynamo paper (Amazon key-value store) | 2007 |

## Tips

- CAP is about partitions, not a free three-way tradeoff. In practice, focus on the latency-consistency tradeoff (PACELC) during normal operation.
- Linearizability is per-object; sequential consistency is per-system. A system can be linearizable for each key individually but not sequentially consistent across keys.
- CRDTs trade expressiveness for coordination-freedom. Not every data structure has a natural CRDT form; some require tombstones that grow without bound.
- R+W>N gives overlap, not linearizability. You still need read-repair or a coordinator to resolve conflicting versions.
- Eventual consistency without conflict resolution is useless. Always define a merge strategy (LWW, vector clocks, CRDTs, application-level).
- Jepsen results are a snapshot in time. Systems improve; always check the version tested.
- "Consistent" means different things in CAP (linearizable), ACID (integrity constraints), and everyday usage (agreement). Be precise.

## See Also

- database-theory
- graph-theory
- complexity-theory
- information-theory
- automata-theory

## References

- Brewer, E. "Towards Robust Distributed Systems" (2000), PODC Keynote
- Gilbert, S. & Lynch, N. "Brewer's Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web Services" (2002), SIGACT News
- Brewer, E. "CAP Twelve Years Later: How the 'Rules' Have Changed" (2012), IEEE Computer
- Abadi, D. "Consistency Tradeoffs in Modern Distributed Database System Design" (2012), IEEE Computer
- Herlihy, M. & Wing, J. "Linearizability: A Correctness Condition for Concurrent Objects" (1990), ACM TOPLAS
- Lamport, L. "How to Make a Multiprocessor Computer That Correctly Executes Multiprocess Programs" (1979), IEEE TC
- Shapiro, M., Preguica, N., Baquero, C. & Zawirski, M. "Conflict-Free Replicated Data Types" (2011), SSS
- DeCandia, G. et al. "Dynamo: Amazon's Highly Available Key-Value Store" (2007), SOSP
- Vogels, W. "Eventually Consistent" (2008), CACM
- Terry, D. et al. "Session Guarantees for Weakly Consistent Replicated Data" (1994), PDIS
- Kingsbury, K. "Jepsen: Distributed Systems Safety Research" (2013--present), jepsen.io
- Kleppmann, M. "Designing Data-Intensive Applications" (O'Reilly, 2017), Ch. 5, 7, 9
