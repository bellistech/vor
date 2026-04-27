# CRDTs — Deep Dive

The why and the algebra behind Conflict-free Replicated Data Types — the data structures that converge without coordination.

## Setup

Strong Eventual Consistency (SEC) is the property that any two replicas which have received the same set of updates have equivalent observable state. Marc Shapiro and his INRIA colleagues formalized this in the 2011 paper "Conflict-free Replicated Data Types," giving birth to a family of distributed data structures that converge automatically — without locks, without coordination, without consensus.

The premise is radical. Distributed systems traditionally rely on consensus (Paxos, Raft) to agree on order: which write happened first, which value won. Consensus requires majority quorums and synchronous communication windows. CRDTs sidestep this entirely. Replicas merge their state via a mathematical operation (a join in a semilattice), and convergence is guaranteed by the algebra alone.

CRDTs are the technical foundation of local-first software, collaborative editors (Yjs, Automerge, Fluid Framework), offline-first mobile apps, peer-to-peer systems, and many distributed databases (Riak, Redis Active-Active, AntidoteDB). They enable seamless multi-master replication, where every replica can accept writes and they all converge once they exchange state.

This deep dive covers the algebraic foundations, the convergence proofs, the design space of specific CRDTs (counters, sets, registers, sequences, maps), the optimization techniques (delta-state, tombstone GC), and the practical limitations.

## Mathematical Foundations

A **partial order** (S, ≤) is a relation that is reflexive (x ≤ x), antisymmetric (x ≤ y ∧ y ≤ x ⟹ x = y), and transitive (x ≤ y ∧ y ≤ z ⟹ x ≤ z). Not every pair must be comparable.

A **semilattice** is a partial order where every pair (x, y) has a **least upper bound** (LUB), denoted x ⊔ y. The LUB is the smallest element greater than or equal to both x and y.

The LUB operation must be:
- **Commutative:** x ⊔ y = y ⊔ x
- **Associative:** (x ⊔ y) ⊔ z = x ⊔ (y ⊔ z)
- **Idempotent:** x ⊔ x = x

These three properties (CAI) are the magic. They ensure that the order in which updates are merged does not matter (commutative), grouping does not matter (associative), and applying the same update twice has no extra effect (idempotent).

For CRDTs, the state of each replica is an element of a semilattice. Local updates produce a new element strictly greater than the current one. Merging two replicas' states is the LUB operation. Convergence follows from CAI.

Examples of semilattices:
- (ℕ, ≤, max) — natural numbers under max
- (𝒫(S), ⊆, ∪) — power set under union
- (ℕ ∪ {⊥}, ⊑, max) where ⊥ is bottom — natural numbers with a bottom element
- Pointwise lifting: if (S, ⊑, ⊔) is a semilattice, then so is (S^I, ⊑, ⊔) where (f ⊔ g)(i) = f(i) ⊔ g(i)
- Cartesian product: if S₁ and S₂ are semilattices, so is S₁ × S₂

This compositionality is critical. Complex CRDTs are built by composing simple semilattices.

## The SEC Theorem

**Theorem (Strong Eventual Consistency):** Let R be a set of replicas, each holding a state in semilattice (S, ⊑, ⊔). Suppose all replicas receive the same set U of updates (in any order, possibly with duplicates). After all updates are applied and merged, all replicas have equivalent state.

**Proof:** Let r₁, r₂ ∈ R be two replicas. Let s₁ = the state of r₁ after applying all updates in U (in some order with some merges), and similarly s₂ for r₂. We show s₁ = s₂.

Each update u ∈ U corresponds to an element u ∈ S. Since updates and merges are CAI-respecting:
- Applying u twice ≡ applying u once (idempotence)
- Order of u₁ then u₂ ≡ u₂ then u₁ (commutativity)
- Grouping ≡ another grouping (associativity)

Therefore s₁ = ⨆_{u ∈ U} u = s₂. The states match. QED.

The SEC theorem is the foundational guarantee. It says that if you have a CRDT and you ensure all updates eventually reach all replicas, convergence is automatic. No coordination needed. No consensus.

The catch: "eventually reach all replicas" is an assumption about the network layer, not a property the CRDT enforces. CRDTs make convergence trivial **given** that updates propagate; they do not solve the propagation problem.

## State-based vs Op-based

There are two main flavors of CRDTs:

**State-based (CvRDT — Convergent):** Each replica holds a state. To synchronize, replicas send their entire state to each other and merge via LUB. Updates are local mutations that move the state up the semilattice.

CvRDTs are simple to reason about: any pair of replicas can converge by exchanging full states. They tolerate any network: messages can be lost, duplicated, reordered, delayed arbitrarily. Each merge is idempotent, so duplicates are harmless. The cost is bandwidth — exchanging full states is expensive for large CRDTs.

**Operation-based (CmRDT — Commutative):** Each replica holds a state. Updates are operations broadcast to all replicas. Each replica applies the operation locally. As long as operations commute (after appropriate causal ordering), all replicas converge.

CmRDTs are more bandwidth-efficient — only operations are sent, not full states. The cost is the network requirement: operations must be delivered exactly once (or with idempotent effects) and in causal order. The transport layer must implement reliable causal broadcast.

**Equivalence of expressive power:** Any CvRDT can be expressed as a CmRDT and vice versa. The choice is implementation-driven, not capability-driven. Most modern CRDT libraries (Yjs, Automerge) are CmRDT-flavored with delta-state optimizations.

## Op-Based Requires Causal Delivery

CmRDTs assume operations are delivered in causal order. Why?

Consider an OR-Set (Observed-Remove Set):
- Operation 1: add("apple", tag="t1") at replica A
- Operation 2: remove("apple") at replica A (sees tag="t1", removes it)

If replica B receives op 2 before op 1, it sees a remove of an unknown tag. The semantics break down: should B accept the remove anyway? Should B buffer it until op 1 arrives?

The standard solution is to require op 2 to be delivered after op 1 at every replica. This is **causal delivery**: if op 1 → op 2 (op 2 was generated with knowledge of op 1), then every replica must apply op 1 before op 2.

The formal definition: a **causal broadcast** layer ensures that for any two operations a, b where a "happened before" b (in Lamport's sense), every replica applies a before b. Operations not causally related can be applied in any order.

Causal delivery is sufficient for most CmRDTs. It is implemented via vector clocks: each operation carries the vector clock of its origin replica at generation time. The receiver buffers the operation until its own vector clock dominates the operation's clock (excluding the sender's component, which can advance by 1).

## Causal Delivery Algorithms

**Birman-Schiper-Stephenson Protocol (BSS, 1991):** The classical causal-broadcast algorithm. Each process maintains a vector clock V[1..N]. To send a message m, the sender increments V[me] and tags m with V. On receipt, the receiver buffers m until V_msg[i] = V_recv[i] + 1 for the sender i, and V_msg[j] ≤ V_recv[j] for all other j. When this condition holds, m is delivered and V_recv updates.

**ISIS (Birman 1985):** An extension supporting groups of processes and ordered/atomic broadcast. Used in early distributed systems. Influential but heavyweight.

**Lightweight CmRDT delivery:** In practice, Yjs and Automerge use a simplified per-replica clock and check causal dependencies inline during operation application. This is more efficient than full vector clocks but assumes a specific operation structure.

The cost of causal delivery: per-message metadata grows with N (number of replicas). Vector clocks scale poorly to thousands of replicas. Approaches like dotted version vectors (Almeida et al.) reduce metadata by tracking dots (replica-id, counter) pairs instead of full vectors.

## δ-State CRDTs

State-based CRDTs ship full states; this is wasteful when the change is small. Delta-state CRDTs (Almeida, Shoker, Baquero 2015) ship only the **delta**: the increment that changed since the last sync.

The key insight: a delta is itself a state in the semilattice. Merging the delta with the receiver's state gives the same result as merging the full state — assuming the delta covers all changes since last sync.

Implementation: each replica tracks the most recent state it has shipped to each other replica. When a local update happens, the replica computes a delta (a small state encoding only the new change). The delta is shipped instead of the full state.

Delta-state CRDTs combine the simplicity of CvRDTs with the bandwidth efficiency of CmRDTs. They are increasingly adopted in modern systems.

## Counter Algebra

**G-Counter (Grow-Only Counter):** A vector C[1..N] of per-replica counts. Each replica i can only increment C[i]. The value of the counter is sum(C[1..N]). The merge is pointwise max: (C ⊔ C')[i] = max(C[i], C'[i]).

Why pointwise max? Because each C[i] is monotonic at replica i (only i increments it). Two replicas may have different views of C[i] at any moment, but the larger one is the more recent. Taking max gives the latest known increment per replica.

The state space (ℕ^N, ⊑, ⊔) is a semilattice (Cartesian product of N max-semilattices). G-Counter satisfies CAI.

**Limitation of G-Counter:** Cannot decrement. Useful for monotonic metrics (page views, requests served), not for general counters.

**PN-Counter (Positive-Negative Counter):** A pair (G+, G-) of two G-Counters. Increments add to G+, decrements add to G-. Value = G+.sum - G-.sum.

Both G+ and G- are independently monotonic. The merge is pointwise on each. The pair (G+, G-) lives in (ℕ^N × ℕ^N, ⊑, ⊔) — also a semilattice.

PN-Counter handles bidirectional changes but requires twice the storage. It cannot enforce "counter ≥ 0" — concurrent decrements may overshoot.

**Bounded counters** (Balegas et al. 2015): To enforce bounds, allocate "tokens" to replicas via a separate consensus mechanism. Replicas can decrement up to their token budget without coordination. Only token reallocation requires consensus. This hybridizes CRDTs with consensus for bounded resources.

## Set Algebra

**G-Set (Grow-Only Set):** A set of elements that only grows. Add adds an element; remove is unsupported. Merge is union: S ⊔ S' = S ∪ S'.

The state space (𝒫(E), ⊆, ∪) is a semilattice. G-Set is the simplest CRDT.

**Limitation:** Cannot remove elements. Useful for append-only logs, immutable references.

**2P-Set (Two-Phase Set):** A pair (Added, Removed) of G-Sets. Element x ∈ effective set iff x ∈ Added ∧ x ∉ Removed. Once x is removed, it stays removed (no re-add).

The state space is (𝒫(E)² , (⊆, ⊆), (∪, ∪)). 2P-Set satisfies CAI.

**Limitation:** Re-adding a removed element is impossible. The "tombstone" of removal is permanent.

**OR-Set (Observed-Remove Set):** Each add of element x creates a unique tag (replica-id, counter). The state is a map from elements to sets of tags. Add adds a (element, tag) pair. Remove only removes tags that the remover has observed.

Effective set: x is in the set iff there exists a tag for x that has not been removed.

The semantics: re-adding a removed element produces a fresh tag, which is not in the remove set. So re-add works.

The merge is component-wise: union of (element, tag) pairs and union of removed tags. Both are G-Sets internally.

OR-Set is the most commonly used set CRDT in practice. Its main cost is metadata: each element carries a set of tags, growing with the number of add operations.

**Add-Wins vs Remove-Wins:** OR-Set is "add-wins" — concurrent add and remove resolve to add. Remove-wins is the dual: concurrent operations resolve to remove. Both are valid; the choice depends on application semantics.

## OR-Set Causal Length Optimization

Almeida et al. (2018, "The Conflict-Free Replicated Data Type Ricciardo") proposed an optimized OR-Set using **causal length**: instead of per-element tag sets, each element has a count. The count goes up on add and down on remove. The element is in the set iff count > 0.

Crucially, the count is interpreted modulo causal history. Two replicas may have different counts due to concurrent operations, but the merge correctly determines the element's effective presence.

This reduces metadata significantly: O(N) per element instead of O(adds). The trade-off is some loss of expressiveness (cannot distinguish individual add events) but the convergence and add-wins semantics are preserved.

## Register Algebra

A register holds a single value. Two main flavors:

**LWW-Register (Last-Writer-Wins):** Each value is tagged with a timestamp. The register's value is the value with the highest timestamp. Merge: keep the (value, timestamp) with higher timestamp.

LWW is simple but loses concurrent writes. If A writes "x" at time 10 and B writes "y" at time 10, the system picks one (using replica-id as tiebreaker). The other write is silently discarded.

LWW assumes synchronized clocks. With clock skew, "happens-before" is approximated by physical time, which can be wrong.

**MV-Register (Multi-Value Register):** Each value is tagged with a vector clock. The register holds a set of values whose vector clocks are concurrent (incomparable). When reads happen, the application sees all concurrent values and resolves them.

MV preserves all concurrent writes. The application must define merge logic. For example, in shopping carts, MV merges cart items by union; in a stock-price register, MV might pick the maximum.

MV-Register is used in Riak, Cassandra (with last-write-wins disabled), and many NoSQL stores.

**LWW-Element-Set:** A set CRDT where each element has add-timestamp and remove-timestamp. Element is in the set iff add-timestamp > remove-timestamp. Concurrent add and remove resolve by timestamp comparison; ties broken by replica ID. Simpler than OR-Set but lossy for concurrent operations on the same element. Used in some Cassandra schema designs.

**MV-Map (Multi-Value Map):** A map where each key maps to an MV-Register. Concurrent writes to the same key produce a set of concurrent values, which the application resolves on read. Useful for shopping carts, document tags, anywhere conflict resolution is application-specific.

**Trade-off:** LWW is simple but loses data; MV preserves data but requires application-level merging. The choice is application-specific.

## Map Algebra

**OR-Map (Observed-Remove Map):** A map where keys form an OR-Set, and each value is itself a CRDT. Adding a (key, value) is like adding to an OR-Set, with the value being a nested CRDT.

Map operations:
- put(k, v): adds (k, tag) to the OR-Set, sets value[k] = v
- update(k, op): applies op to value[k] (if k exists)
- delete(k): removes all tags for k

Concurrent put(k, v1) and update(k, op) on different replicas: both are visible. The map shows k → merge(v1, v1.applied(op)). The value CRDT determines how to merge.

Recursive composition: an OR-Map can hold OR-Maps as values. Automerge and Yjs support arbitrary JSON tree CRDTs by recursive composition.

**Limitation:** "Add-wins" for keys. If A adds a key and B deletes it concurrently, the key persists. This is usually desired but should be confirmed for the application.

**OR-Map causal-stability cleanup:** Like OR-Set, OR-Map accumulates tags. Periodic causal-stability sweeps can collapse tag sets to a single representative tag once all replicas have observed the entry. Yjs's Y.Map and Automerge's Map both implement variants of this with explicit tombstone tables and stability frontiers.

## Sequence CRDTs

Sequences (ordered lists, used in collaborative text editing) are the most challenging CRDT family. They must preserve insertion order across concurrent operations and handle insertions at arbitrary positions.

**WOOT (Without Operational Transform, Oster et al. 2006):** Each character has a unique ID and references its predecessor and successor. Insert(c, prev, next) creates a new character between prev and next. Concurrent inserts at the same position are ordered by ID.

WOOT is correct but inefficient: storing predecessor/successor for every character creates O(n) lookup time, and tombstones (deleted characters) accumulate.

**Logoot (Weiss et al. 2009):** Each character has a fractional position ID — an arbitrary-precision rational number. Insert at position p between positions p1 and p2 picks a fresh ID strictly between them.

Logoot eliminates tombstones but suffers from "position blowup": rapid insertions at the same point require ever-longer IDs, growing without bound. Worst case, IDs grow to N bytes for N concurrent inserts.

**RGA (Replicated Growable Array, Roh et al. 2009):** Operation-based. Each character has an ID (replica, counter). Insert references the predecessor's ID. Delete sets a tombstone. Merge: insert the new character after its referenced predecessor; deletes mark tombstones.

RGA achieves O(log n) insertion (with appropriate data structures) and is the basis for most modern sequence CRDTs.

**Yjs YText (2015):** Optimized RGA-flavor. Uses doubly-linked list with structural sharing. Stores characters in compressed "items" that can split and merge. Adopts garbage collection for stable tombstones.

YText is one of the highest-performing sequence CRDTs. Yjs is widely deployed in Notion, Evernote, and many real-time collaborative apps.

**Diamond Types (Joseph Gentle, 2022):** A new sequence CRDT with the "eg-walker" algorithm. Achieves O(1) amortized insert and read by reorganizing operations into a more efficient internal structure. Aims to be the fastest text CRDT.

## RGA Algorithm

The RGA structure: a doubly-linked list of items. Each item has a unique ID (replica, counter), content, predecessor pointer, and a tombstone flag.

**Insert(c, predID):**
1. Generate a new ID (me, my_counter++).
2. Create a new item with content c, ID, predecessor = predID.
3. Find the position: traverse from predID, skipping items with greater ID (those are concurrent inserts at the same position; they go after the new one in the resolved order).
4. Splice in the new item.

**Delete(id):**
1. Find the item with matching ID.
2. Set its tombstone flag.

**Merge:** When receiving a remote insert, follow steps 2-4 of Insert (creating the new item with the remote ID). When receiving a remote delete, set the tombstone of the matching item.

The order of operations within an item's predecessor must be deterministic. The standard rule: among items sharing a predecessor, order by ID (replica, counter). All replicas see the same total order.

**Garbage Collection:** Once all replicas have seen a delete, the item can be physically removed. This requires causal stability tracking — knowing when an operation is "stable" across all replicas.

## Yjs Internal Format and YText Algorithm

Yjs encodes operations as a binary stream. Each "item" has:
- ID: 8 bytes (clientID + clock)
- Origin: previous item's ID (for ordering)
- Right origin: next item's ID (for redundancy)
- Content: variable-length
- Parent: containing structure (for nested CRDTs)
- Parent-sub: key (for Map entries)

Items can be **structurally shared**: contiguous insertions by the same client form a single item. This drastically reduces overhead for typical typing patterns. When items split (due to interleaved concurrent inserts), Yjs splits the structure dynamically.

Yjs's binary format is compact and supports efficient sync: replicas exchange "state vectors" (per-client clocks) to identify missing updates, then exchange only the missing portion.

**The YText core insertion algorithm:**

```
Insert(content, leftItemID, rightItemID, clientID, clock):
  newItem = Item(id=(clientID, clock), origin=leftItemID, rightOrigin=rightItemID, content=content)
  // Walk from leftItem.right toward rightItem to find the correct insertion point.
  cursor = leftItem.right
  while cursor != rightItem:
    if cursor.origin == leftItemID:
      // Concurrent insert at the same origin. Resolve by clientID order.
      if cursor.id.clientID > newItem.id.clientID:
        cursor = cursor.right
        continue
      else:
        break
    if cursor.origin happens-before newItem.origin:
      // cursor's origin was inserted later in the conflict chain; skip it.
      cursor = cursor.right
      continue
    break
  splice newItem before cursor
```

The algorithm guarantees that all replicas, given the same set of operations, splice each item at the same position regardless of receipt order. Determinism comes from the clientID tiebreaker and the origin walk.

The garbage collection of deleted struct chains: once all replicas have acknowledged a delete, Yjs removes the item from memory. Stability is tracked via "delete sets" — explicit records of deletes that can be pruned.

## Automerge JSON CRDT

Automerge is a CRDT library that represents arbitrary JSON documents. Every JSON value (object, array, string, number) is a CRDT.

- **Maps (objects):** OR-Map of keys to values.
- **Arrays:** RGA-style sequence of values.
- **Strings:** RGA of characters (treated as a sequence type).
- **Primitives (numbers, booleans):** LWW-Register.

Each operation in Automerge is a (clientID, opCounter, action, target, value) tuple. Operations form a DAG of causal dependencies. Replicas exchange operation sets and merge by adding missing ops and recomputing the materialized view.

**Columnar storage format:** Automerge stores the operation history in a column-oriented format. Each "column" holds one attribute of the operations (clientIDs, counters, actions, etc.). Compression is excellent because consecutive operations often share clientID.

The columnar format reduces history size dramatically — a 1000-character document might require only a few KB. Without compression, operation logs can balloon to MBs.

**Sync protocol:** Replicas exchange Bloom filters of operation hashes. Missing operations are identified probabilistically and exchanged. Syncing two replicas typically requires 2-3 rounds.

## Tombstone Garbage Collection

Tombstones (deletion markers) cannot be removed eagerly. If a deleted element's tombstone is gone, a stale write of that element from a slow replica might re-insert it.

**Causal Stability:** A delete is **stable** when every replica has seen it. After stability, any subsequent operation will already know about the delete (by causal delivery), so the tombstone is redundant.

**Stable Frontier:** Each replica tracks the "frontier" — the set of operations that all replicas have seen. Operations in the frontier can have their tombstones removed.

Computing the stable frontier requires explicit "ack" messages or out-of-band tracking. In practice, this is often done via heartbeats: each replica reports its current vector clock periodically; when a particular operation is below all replicas' clocks, it is stable.

**Approximate stability:** Some systems use timeouts (assume stability after T seconds without conflicting writes). This is unsafe in the worst case but pragmatic for many workloads.

Tombstone GC is a major engineering concern in long-running CRDT systems. Without GC, state grows unboundedly. With GC, complex synchronization is required.

## Tombstone GC Algorithm — Pseudocode

```
// Per-replica state
my_clock: Map[ReplicaID -> int]
peer_clocks: Map[ReplicaID -> Map[ReplicaID -> int]]  // last-known clock of each peer
tombstones: List[(opID, vectorClock)]

// Periodic exchange — every T seconds
broadcast_clock():
  send(my_clock) to all peers

on_receive_clock(peer, clk):
  peer_clocks[peer] = clk

// Compute stability frontier
compute_stable_frontier() -> VectorClock:
  // The minimum clock across all replicas (including self)
  frontier = my_clock.copy()
  for peer, clk in peer_clocks:
    for replica, count in clk:
      frontier[replica] = min(frontier[replica], count)
  return frontier

// Run GC
gc_step():
  frontier = compute_stable_frontier()
  for (opID, opClock) in tombstones:
    // An op is stable if its clock is dominated by the frontier
    if opClock <= frontier:  // pointwise
      // Safe to remove: no replica can later refer to this op as concurrent
      remove tombstone(opID)
      remove all metadata associated with opID
```

**Correctness sketch:** When `opClock <= frontier`, every replica has acknowledged seeing op (because frontier is the min of all replicas' clocks). Any subsequent op generated at any replica will have a clock dominating opClock — so any future "concurrent" reference is impossible. The tombstone is genuinely no longer needed.

**Safety boundary:** If a peer is silently offline (not reporting its clock), `frontier` stalls. This is the safe fallback: GC pauses rather than risk premature collection. Real systems include timeouts: if a peer is unresponsive for >T_dead, it is considered crashed and removed from the frontier computation. This is a liveness/safety trade-off — premature peer removal can cause divergence on rejoin.

**Implementation in Yjs:** Yjs maintains a `deleteSet` mapping each clientID to ranges of deleted clocks. When state vectors are exchanged, deletions older than the minimum clock for each client are eligible for memory reclamation. Yjs runs this opportunistically during sync.

## Causal-Stability Proof

**Theorem (Causal Stability):** Let op be an operation with vector clock V_op generated at replica r. Op is causally stable iff for every replica r', the local clock V_r'[r] ≥ V_op[r] (and similarly for all other components).

**Proof sketch (one direction — stability ⟹ no concurrent ops):**

Suppose op is stable. We show that no future op op' generated at any replica r' can be concurrent with op.

Consider any future op'. By causal delivery, when r' generates op', it has already applied all ops in V_r'. Since stability means V_r' dominates V_op (component-wise), r' has already applied op. Therefore op happens-before op', not concurrent.

So once stable, op is in the causal past of every future operation.

**Proof sketch (other direction — no concurrent ⟹ stable):**

Suppose no future op can be concurrent with op. We show V_r'[k] ≥ V_op[k] for every component k at every replica r'.

By contradiction: suppose V_r'[k] < V_op[k] for some k at some r'. Then r' has not yet seen op_k (the kth operation from replica k that op depended on). r' could generate an op' concurrent with op_k — and transitively concurrent with op. Contradicts assumption.

Hence stability is equivalent to "fully replicated and acknowledged."

**Corollary:** Tombstones can be safely garbage-collected once stable, because the "concurrent add" scenario that the tombstone was protecting against is impossible by the theorem.

This proof is the formal justification for tombstone GC. It connects the abstract "stability frontier" to a concrete safety property.

## CRDT Limitations

**Cannot enforce uniqueness:** "Username must be globally unique" cannot be expressed as a CRDT. Two replicas can each accept the same username, and the merge cannot reject one. Uniqueness requires consensus.

**Cannot enforce arbitrary cross-document constraints:** "Sum of all account balances must equal X" cannot be enforced via CRDTs. Each account is independent; the sum is an emergent property.

**Convergence ≠ correctness:** CRDTs guarantee that all replicas reach the same state. They do not guarantee the state is "correct" by application semantics. A user might want "last write wins" but get "concurrent values preserved." The CRDT cannot read the user's mind.

**Bandwidth/storage trade-offs:** CRDTs typically have larger metadata than non-replicated structures. OR-Set carries tag sets per element. RGA carries IDs per character. This metadata is essential for convergence but increases storage.

**Recovery from divergence:** If a replica's state somehow becomes corrupted (e.g., disk error), CRDT merge cannot fix it — corruption is "below" the LUB level. Application-level checksums or external integrity checks are needed.

**Latency tolerance:** CRDTs assume eventually-delivered updates. If a replica is offline for a year, its updates may conflict with a year of progress. Merging is correct but may produce surprising results.

## CRDT vs OT (Operational Transform)

Operational Transform (OT) was the original real-time collaborative editing technique, used in Google Docs, Etherpad, and similar systems. OT works by transforming concurrent operations against each other to produce equivalent linearized operations.

For example, if A inserts "hello" at position 5 and B simultaneously inserts "world" at position 5, OT must transform B's operation to insert at position 10 (since A's insert pushed positions 5-9 to 10-14).

OT's challenge: the transformation function must be carefully designed to commute. Designing a correct transformation function for arbitrary operations (especially undo) is famously difficult. Many OT papers contain bugs.

CRDTs eliminate the need for transformation. Operations carry enough metadata (IDs, vector clocks) that they commute naturally. CRDTs are easier to prove correct.

OT requires a central server (or careful peer coordination) to ensure transformation chains are consistent. CRDTs work peer-to-peer.

Modern collaborative editors (Yjs, Automerge, Notion's internal stack) use CRDTs. OT is in maintenance mode in legacy systems.

## Network Sync Protocols

**Yjs over WebSocket:** y-websocket provides a server that brokers Yjs document syncing. Each client connects, fetches the document state vector, exchanges missing operations, and stays connected to receive real-time updates. The y-websocket server can be a thin relay; it does not store state long-term.

**Automerge sync via Bloom filters:** When two replicas sync, each computes a Bloom filter of operation hashes. Each sends the filter to the other. The receiver identifies operations the sender lacks (false positives are tolerable; missed operations are not). The receiver sends the missing operations.

**libp2p pubsub:** A peer-to-peer message-routing layer. Replicas subscribe to a "topic" (the document ID) and publish operations. Others receive and merge. Used in IPFS-based collaborative apps.

**Offline-first reconciliation:** When a replica returns from offline, it has potentially many local changes and many missed remote changes. The sync protocol must efficiently identify the gap and exchange only the missing operations. State vectors and Bloom filters are common tools.

Sync protocols are an active area of research. The challenges include scalability (many concurrent writers), security (trustless peers), bandwidth (low-bandwidth connections), and battery (mobile devices).

## Local-First Software

Ink & Switch's "Local-first software" manifesto (Kleppmann et al. 2019) articulates a vision where:

1. **No spinners:** Software responds instantly because all data is local.
2. **Works offline:** No network required for routine use.
3. **Data is yours:** Users control their data, not cloud providers.
4. **Multi-device sync:** Devices automatically reconcile via CRDTs or similar.
5. **Long-lived:** Data outlives any company or service.

CRDTs are the technical enabler. They allow multi-device, multi-user collaboration without a central server, with automatic convergence.

Examples of local-first apps: Notion (partial), Obsidian (with sync plugins), Logseq, Anytype, Kinopio. The ecosystem is growing.

The challenges for local-first: scaling to large datasets, handling untrusted peers, building user-friendly conflict resolution, and matching the polish of cloud apps. CRDTs solve the convergence problem; the rest is engineering.

## Theoretical Connections

**CALM Theorem (Hellerstein, Alvaro 2010):** A program produces consistent results without coordination if and only if it is monotonic. Monotonic programs operate over semilattice-like structures. CRDTs are the data-structure embodiment of this principle.

**Bloom Language:** A declarative language by Hellerstein for writing CALM-conforming programs. Programs are sets of monotonically growing rules. Compiles to distributed implementations.

**Dotted Version Vectors (Almeida et al. 2010):** A compact alternative to vector clocks. Tracks operations as (replica, counter) "dots" plus a vector of summarized counts. Reduces metadata for systems with many transient clients.

**Pure Op-Based CRDTs (Baquero et al. 2014):** A CRDT design where operations require no preparation (no state read before broadcast). Simpler causal-broadcast requirements.

**Mixed CRDTs:** Hybrid systems combining CRDTs with consensus for constraints that pure CRDTs cannot express. AntidoteDB and Cure are research prototypes; Riak's bounded counters are a production example.

## Implementation Notes

Building a CRDT library requires careful attention:

- **Verify CAI properties.** Use property-based testing (QuickCheck-style). Random sequences of operations should converge regardless of merge order.
- **Test with adversarial schedulers.** Design tests that introduce maximum reordering.
- **Stress-test with many replicas.** Pairwise convergence is easy; N-replica convergence with N=100 reveals subtle bugs.
- **Profile metadata growth.** Many CRDTs have hidden costs (tombstones, version vectors).
- **Plan for GC.** Without garbage collection, state grows linearly with operations.

Mature CRDT libraries: Yjs (TypeScript), Automerge (JavaScript/Rust), Riak DT (Erlang), AntidoteDB (Erlang). Each has trade-offs in performance, correctness guarantees, and feature set.

## Performance Tuning

CRDTs can be fast or slow depending on implementation:

- **Yjs:** Microseconds per operation. Hand-tuned RGA with structural sharing.
- **Automerge:** Slower (milliseconds) due to JavaScript overhead and richer JSON model. The Automerge-Rs Rust port is faster.
- **Diamond Types:** Optimized for raw text editing. Aims to be the fastest text CRDT.
- **Riak DT:** Slower (latencies in tens of milliseconds) due to Erlang VM and per-key persistence.

Optimizations:
- Use binary encoding (CBOR, MessagePack) instead of JSON for state.
- Batch operations and merge once per batch.
- Cache materialized views (the user-visible state) and update incrementally.
- Use copy-on-write for immutable structures.
- Garbage-collect aggressively (with safety checks).

## Practical Use Cases

**Collaborative document editing:** Yjs and Automerge dominate. Notion, Logseq, Obsidian, Anytype use CRDTs.

**Multi-master databases:** Riak, Redis Active-Active, Cosmos DB (multi-master mode), AntidoteDB. CRDTs enable writes at any datacenter.

**Offline-first mobile apps:** RxDB, Apollo Client (with delta-state extensions). Apps that work offline and sync when online.

**Edge computing:** Cloudflare Workers, Fly.io, Deno Deploy. Edge replicas of state need CRDT-style convergence.

**P2P apps:** IPFS-based collaboration. Web3-style apps with no central server.

**Game state synchronization:** Some multiplayer games use CRDT-flavored state replication for player data, inventories.

**Configuration management:** Etcd-style stores with eventual consistency for less-critical config.

CRDTs are increasingly mainstream. The next decade will likely see them in OS-level filesystems, IDE shared workspaces, and data-mesh architectures.

## Looking Forward

Active research in CRDTs:

- **Rich-text CRDTs:** Beyond plain text — formatted text with overlapping styles is hard. Peritext (Ink & Switch, 2022) is a recent design.
- **Schema evolution:** How does a CRDT handle field renames, type changes? Active research.
- **Encryption-friendly CRDTs:** End-to-end encrypted collaboration. Yjs has experimental support; more mature solutions are coming.
- **Verified CRDTs:** Formal proofs of convergence using Coq, Isabelle. The Liquid Haskell port of Yjs. CRDTs are popular targets for formal verification.
- **Adversarial CRDTs:** Resistance to malicious peers. Default CRDTs assume honest replicas.
- **Operational efficiency:** Reducing metadata, improving compression, faster merges.

CRDTs are a young field — only ~15 years old since the seminal Shapiro paper. The fundamentals are settled, but the practical engineering, the scaling, and the adversarial extensions are evolving rapidly.

## Worked Example: G-Counter Convergence Trace

Two replicas A and B start with G-Counter [0, 0]. The vector position 0 is for replica A; position 1 is for replica B.

| step | event                       | State_A     | State_B     | observed value |
|------|-----------------------------|-------------|-------------|----------------|
| 0    | initial                     | [0, 0]      | [0, 0]      | 0              |
| 1    | A increments locally        | [1, 0]      | [0, 0]      | A=1, B=0       |
| 2    | B increments twice locally  | [1, 0]      | [0, 2]      | A=1, B=2       |
| 3    | A and B exchange states     | [1, 2]      | [1, 2]      | 3              |
| 4    | A increments again          | [2, 2]      | [1, 2]      | A=4, B=3       |
| 5    | B disconnects, restarts from snapshot [1, 2] (loses RAM but keeps disk) | [2, 2] | [1, 2] | A=4, B=3 |
| 6    | B increments after recovery | [2, 2]      | [1, 3]      | A=4, B=4       |
| 7    | exchange                    | [2, 3]      | [2, 3]      | 5              |

The pointwise-max LUB ensures that even after B's crash and re-increment, the merge correctly accumulates all updates. B's state [1, 2] was a valid lower-bound element of the semilattice; merging [1, 3] (B's post-recovery state) with [2, 2] (A's state) gives [2, 3] — correct.

This worked example shows: idempotence (B's repeated state isn't double-counted), commutativity (order of merges doesn't matter), associativity (multi-step merges produce the same result).

## Worked Example: OR-Set ABA Scenario

Consider the classic ABA problem: a value is added, removed, then re-added. Does the OR-Set handle it correctly?

| step | event                                                | State_A                         | State_B                         |
|------|------------------------------------------------------|---------------------------------|---------------------------------|
| 0    | initial                                              | {}                              | {}                              |
| 1    | A adds "apple" with tag (A, 1)                       | {apple: {(A,1)}}                | {}                              |
| 2    | B receives op, applies                               | {apple: {(A,1)}}                | {apple: {(A,1)}}                |
| 3    | A removes "apple"; remove-set = {(A,1)}              | {apple: {}, removed: {(A,1)}}   | {apple: {(A,1)}}                |
| 4    | B receives remove                                    | {apple: {}, removed: {(A,1)}}   | {apple: {}, removed: {(A,1)}}   |
| 5    | A re-adds "apple" with tag (A, 2)                    | {apple: {(A,2)}, removed: {(A,1)}} | {apple: {}, removed: {(A,1)}}   |
| 6    | B receives re-add                                    | {apple: {(A,2)}, removed: {(A,1)}} | {apple: {(A,2)}, removed: {(A,1)}} |

The element "apple" is correctly present after re-add because the new tag (A, 2) is not in the remove-set. This is what distinguishes OR-Set from 2P-Set: tombstones are *per-tag*, not *per-element*, allowing re-add.

**The harder ABA case — concurrent re-add with remove:**

| step | event                                                            | State_A                              | State_B                              |
|------|------------------------------------------------------------------|--------------------------------------|--------------------------------------|
| 0    | initial after step 4 above                                       | {apple: {}, removed: {(A,1)}}        | {apple: {}, removed: {(A,1)}}        |
| 1    | A locally re-adds "apple" with (A, 2)                            | {apple: {(A,2)}, removed: {(A,1)}}   | {apple: {}, removed: {(A,1)}}        |
| 2    | B locally adds "apple" with (B, 1) (didn't see A's re-add)       | {apple: {(A,2)}, removed: {(A,1)}}   | {apple: {(B,1)}, removed: {(A,1)}}   |
| 3    | A and B exchange                                                 | {apple: {(A,2),(B,1)}, removed: {(A,1)}} | {apple: {(A,2),(B,1)}, removed: {(A,1)}} |

Both tags survive. "apple" is present, supported by two tags. Future removes target individual tags, not the element. This is the add-wins property: concurrent adds accumulate.

**The pathological case — concurrent remove of one tag while another adds:**

| step | event                                                                 | State_A                              | State_B                              |
|------|-----------------------------------------------------------------------|--------------------------------------|--------------------------------------|
| 0    | shared state                                                          | {apple: {(A,1)}}                     | {apple: {(A,1)}}                     |
| 1    | A removes (A, 1) — remove-set adds (A, 1)                             | {apple: {}, removed: {(A,1)}}        | {apple: {(A,1)}}                     |
| 2    | B concurrently re-adds with new tag (B, 1)                            | {apple: {}, removed: {(A,1)}}        | {apple: {(A,1),(B,1)}}               |
| 3    | merge                                                                 | {apple: {(B,1)}, removed: {(A,1)}}   | {apple: {(B,1)}, removed: {(A,1)}}   |

The element survives via (B,1). A's intent to remove (A,1) is honored — that specific tag is gone — but the concurrent add by B adds a fresh tag that A's remove never observed. Add-wins by design.

## Worked Example: RGA Insertion-Deletion-Resolution

Document is initially "AB" with characters at positions (A: id=(R1,1), B: id=(R1,2)). Two replicas R1 and R2.

| step | event                                                         | R1's view                                   | R2's view                                   |
|------|---------------------------------------------------------------|---------------------------------------------|---------------------------------------------|
| 0    | initial                                                       | A(R1,1) → B(R1,2)                           | A(R1,1) → B(R1,2)                           |
| 1    | R1 inserts "C" between A and B; new id=(R1,3), pred=(R1,1)    | A(R1,1) → C(R1,3) → B(R1,2)                 | A(R1,1) → B(R1,2)                           |
| 2    | R2 inserts "X" between A and B (concurrent); new id=(R2,1), pred=(R1,1) | A(R1,1) → C(R1,3) → B(R1,2) | A(R1,1) → X(R2,1) → B(R1,2)                 |
| 3    | R1 receives X (predID=(R1,1)). Items sharing pred (R1,1): {C, X}. Order by id: (R1,3) < (R2,1)? Compare clientID: R1=1, R2=2 → (R1,3) < (R2,1). So C before X. | A → C → X → B | (unchanged)                  |
| 4    | R2 receives C (predID=(R1,1)). Same logic: C before X.        | A → C → X → B                               | A → C → X → B                               |
| 5    | R1 deletes C; tombstone on id=(R1,3)                          | A → C[†] → X → B                            | A → C → X → B                               |
| 6    | R2 deletes X; tombstone on id=(R2,1)                          | A → C[†] → X → B                            | A → C → X[†] → B                            |
| 7    | exchange tombstones                                           | A → C[†] → X[†] → B                         | A → C[†] → X[†] → B                         |
| 8    | rendered output (skip tombstones)                             | "AB"                                        | "AB"                                        |

Both replicas converge to "AB" — the original document, with C and X both inserted then deleted. The tombstones for C and X persist until causal stability allows GC. Once stable (all replicas have observed both deletes), the items are removable from memory.

**Performance note:** Yjs would compress the contiguous tombstones [†][†] into a single deletion-range entry in its delete set, reducing the per-tombstone overhead from ~32 bytes to ~8 bytes amortized.

## Worked Example: Yjs Document Sync via State Vectors

Two clients C1 and C2 are editing a Yjs document via a y-websocket server.

**Step 1:** C1 has state vector {C1: 5, C2: 3}, meaning it has seen 5 ops from C1 and 3 from C2.
**Step 2:** C2 has state vector {C1: 4, C2: 6}, having seen 4 ops from C1 and 6 from C2.

**Step 3:** C1 sends its state vector to C2 via the server. C2 sees that C1 is missing C2's op #4, #5, #6 and has C1's op #5 (which C2 hasn't seen).

**Step 4:** C2 sends C1 the missing C2 ops (#4, #5, #6). C1 sends C2 the missing C1 op (#5). Each batches as a binary update.

**Step 5:** Both apply the missing ops. C1's state: {C1: 5, C2: 6}. C2's state: {C1: 5, C2: 6}. Convergence achieved.

The state-vector sync protocol minimizes bandwidth — only missing ops are exchanged, not full state.

## Worked Example: Automerge Sync via Bloom Filter

A and B want to sync via Bloom filters of operation hashes.

**Step 1:** A computes BloomA from the hashes of all its ops.
**Step 2:** B computes BloomB. Each Bloom is ~few KB.
**Step 3:** A sends BloomA to B. B identifies which of its ops are likely missing from A by checking each B-op against BloomA. False positives cause B to think A already has an op when it doesn't; false negatives don't occur.
**Step 4:** B sends those ops to A. A applies them.
**Step 5:** Often, a second round catches false-positive misses.

Total bandwidth: O(log(N)) Bloom + O(missing ops). For small differences, sync is very cheap.

## Performance Comparison Across CRDT Types

| CRDT          | State size                    | Op cost     | Merge cost      | Use case                      |
|---------------|-------------------------------|-------------|-----------------|-------------------------------|
| G-Counter     | O(N) per replica              | O(1)        | O(N)            | Monotonic counter             |
| PN-Counter    | O(N) per replica              | O(1)        | O(N)            | Counter                       |
| G-Set         | O(items)                      | O(1)        | O(items)        | Append-only set               |
| 2P-Set        | O(items)                      | O(1)        | O(items)        | Permanent removal             |
| OR-Set        | O(items × adds-per-item)      | O(1)        | O(metadata)     | General set                   |
| LWW-Register  | O(value)                      | O(1)        | O(1)            | Single value                  |
| MV-Register   | O(concurrent-values)          | O(1)        | O(concurrent)   | Single value with conflicts   |
| RGA           | O(items)                      | O(log items)| O(ops)          | Ordered list                  |
| Yjs YText     | O(items, compressed)          | O(log items)| O(ops)          | Collaborative text            |
| Automerge JSON| O(history, compressed)        | O(history)  | O(missing-ops)  | JSON tree                     |

For high-throughput systems: G-Counter and LWW are cheapest. For collaborative editing: Yjs YText is currently the fastest. For complex JSON: Automerge offers richer semantics at higher cost.

## Performance Numbers — Yjs vs Automerge

Concrete benchmark results (from Yjs author's published numbers and Diamond Types' comparison suite):

**Document size after 100K random edits (typing pattern, 5 clients interleaved):**
- Yjs: ~250 KB binary log (compressed structure-shared items).
- Automerge (v2): ~720 KB columnar binary.
- Automerge (v1): ~3.5 MB JSON-encoded.
- Diamond Types: ~180 KB (most compact for text).

**Apply 1000 remote ops:**
- Yjs: ~1.0 ms (1 µs per op).
- Automerge-rs: ~3.5 ms (3.5 µs per op).
- Automerge JS: ~12 ms.
- Diamond Types: ~0.6 ms.

**Materialize entire document of 1M characters:**
- Yjs: ~25 ms.
- Automerge: ~120 ms.
- Diamond Types: ~15 ms.

**Per-character byte overhead (steady state, no GC):**
- Yjs: ~5-10 bytes per character (after structural sharing of contiguous typing).
- Automerge: ~25-40 bytes per character (richer metadata).
- Diamond Types: ~3-5 bytes per character.

**Sync latency (LAN, 10ms RTT):**
- Yjs state-vector sync, 100-op delta: ~12 ms total (one round trip + small payload).
- Automerge Bloom-filter sync, 100-op delta: ~25 ms total (typically two rounds).

These numbers shift over time as libraries optimize; consult the CRDT-benchmarks GitHub repo for current results.

## Tracking Causality with Vector Clocks

Each replica maintains a vector clock V[1..N]. To send op, increment V[me] and tag op with V. On receipt, deliver op when:
- V_op[sender] = V_recv[sender] + 1 (this is the next op from sender), AND
- V_op[other] ≤ V_recv[other] for all other replicas (we've seen all causally-preceding ops).

If conditions fail, buffer the op and wait.

**Example:**
- Replica 1 sends op_1 with V=[1,0,0].
- Replica 1 sends op_2 with V=[2,0,0].
- Replica 2 receives op_2 first. Buffers (V_recv[1] = 0, but op_2 needs sender clock = 2).
- Replica 2 receives op_1 with V=[1,0,0]. Conditions met: V_recv[1]=0, op_1 sender clock = 1 = 0+1. Delivers. V_recv = [1,0,0].
- Replica 2 re-checks buffer. op_2 (sender clock 2) now matches V_recv[1]+1 = 2. Delivers. V_recv = [2,0,0].

Vector clocks ensure causal delivery but require O(N) metadata per message.

## Causal Stability and Tombstone GC

The naive OR-Set has unbounded metadata growth (tombstones never removed). Causal stability solves this.

**Definition:** An op X is **causally stable** at replica R if every replica has acknowledged seeing X. Equivalently: all replicas have a vector clock where X is in the past.

**Implementation:**
- Each replica periodically broadcasts its vector clock.
- A replica computes the minimum clock across all replicas.
- Any op with clock ≤ minimum is stable everywhere.

**GC trigger:** Once an op is stable, downstream tombstones from before the op can be removed. The remove op is no longer needed because:
1. All replicas have already applied it.
2. No new "concurrent" add can appear (all future ops are causally after).

So the (added, removed) tag pair can be collapsed: removed effectively deletes the added entry. Memory is reclaimed.

**Edge case:** Replicas that go offline and miss the stability broadcast must catch up before stability advances. Otherwise, GC is paused. This trade-off (responsiveness vs offline tolerance) is configurable.

**Adversarial replicas:** A misbehaving replica can stall stability indefinitely by never advancing its clock. Real systems detect and remove such replicas (or use heartbeat timeouts).

## CRDTs as Lattices: Composition

If (S₁, ⊑₁, ⊔₁) and (S₂, ⊑₂, ⊔₂) are semilattices, then so is (S₁ × S₂, ⊑, ⊔) where (a₁, a₂) ⊑ (b₁, b₂) iff a₁ ⊑₁ b₁ and a₂ ⊑₂ b₂, and (a₁, a₂) ⊔ (b₁, b₂) = (a₁ ⊔₁ b₁, a₂ ⊔₂ b₂).

This compositionality is why complex CRDTs work:
- PN-Counter = G-Counter × G-Counter (positive and negative).
- 2P-Set = G-Set × G-Set (added and removed).
- OR-Map = OR-Set × (Key → Value-CRDT).
- Document = Map[Key, ...]<br/>Each value can itself be a CRDT.

The "Map[K, V]" pattern is critical: a map where keys form a CRDT (often OR-Set semantics) and values are CRDTs. Recursive composition creates arbitrary nested structures.

## Garbage Collection Strategies

**Tombstone-based GC:**
- Track removal as a tombstone.
- Only remove tombstones when causally stable.
- Memory cost: O(removed × N) until stability.

**Counter-based (causal length):**
- Replace tag sets with counters.
- Counters advance on add/remove.
- Smaller metadata but lossy: cannot distinguish individual events.

**Tabular GC:**
- Store add/remove operations in a table.
- GC compacts the table when stable.
- More flexible but requires schema awareness.

**Snapshot + log:**
- Take periodic state snapshots.
- Discard ops before the snapshot timestamp once all replicas have it.
- Familiar pattern from databases.

## Real-World Latencies

Yjs benchmark (2023):
- Insert one character: ~10 µs.
- Apply remote update of 1000 ops: ~1 ms.
- Compute diff between two state vectors: < 1 ms.

Automerge benchmark:
- Insert one character: ~50 µs.
- Apply 1000-op update: ~10 ms.
- JSON document with 100 keys: 100 µs per access.

Riak DT (Erlang):
- Update with consensus: ~10-50 ms (network-bound).
- Read: ~1-5 ms.

Mobile peer-to-peer (libp2p over WebRTC):
- Sync 100-op delta: ~100 ms (network RTT-bound).

CRDT operations are typically faster than network latency, so the bottleneck is communication, not computation.

## Engineering Patterns: Schema Evolution

When the data schema changes, CRDT replicas must agree. Strategies:

**Forward-compatible additions:** Adding a new key to an OR-Map is safe. Existing replicas ignore unknown keys; new replicas use them.

**Backward-incompatible changes:** Renaming or restructuring requires migration. Options:
1. **Dual schema:** Both schemas active. New ops go to new schema; reads merge old and new. Eventually deprecate old.
2. **Migration op:** A special op converts old to new. All replicas must apply it.
3. **Versioned CRDT:** Each op tagged with schema version. Code handles each version.

Yjs and Automerge are working on versioning support but it's not fully solved.

## Engineering Patterns: Encrypted CRDTs

End-to-end encryption with CRDTs is hard because servers cannot decrypt and merge. Approaches:

**Op-encrypted with shared key:** All clients share a key. Server stores encrypted ops; cannot read them. Clients decrypt and merge locally. Sync via op exchange.

**Hybrid with metadata-only encryption:** Encrypt only sensitive fields. Server can sync structure but not content.

**Functional Encryption:** Theoretically possible but no production systems exist.

Yjs has experimental e2e encrypted sync. Hocuspocus (Yjs server) supports server-side encryption (less strong).

## CRDT vs Eventually-Consistent KV Stores

Many systems claim "eventually consistent." How do they relate to CRDTs?

**Last-Writer-Wins (LWW):** Many systems (Cassandra, default Cosmos DB) use LWW for conflicts. This is a CRDT (LWW-Register) but a lossy one. Older writes are overwritten.

**Multi-Value Registers:** Some systems (Riak, Cosmos DB strong-eventual) preserve all concurrent values. The application reconciles. This is a generic MV-Register.

**Custom CRDTs:** Riak supports counters, sets, maps, registers natively. Cassandra has counters and (limited) sets.

**Pure CRDT databases:** AntidoteDB, Cure, Lasp. Use rich CRDT semantics natively.

The trend is toward richer CRDT support in mainstream databases. Cosmos DB, Riak, and others now offer multiple CRDT types.

## Distributed Systems Theorems Relevant to CRDTs

**CAP Theorem:** Under partition, must choose Consistency or Availability. CRDTs choose Availability — they always accept writes locally and reconcile later.

**PACELC:** Even when not partitioned, choose between Latency and Consistency. CRDTs prioritize Latency (local writes are fast).

**FLP Impossibility:** No deterministic asynchronous consensus. CRDTs avoid consensus entirely; convergence is automatic via algebra.

**CALM Theorem:** Monotonic = consistent without coordination. CRDTs are the canonical monotonic data structures.

**CRDTs sit at one extreme of the design space:** maximum availability, maximum latency tolerance, no coordination. The trade-off is constraints on what semantics you can express.

## When to Choose CRDTs

**Use CRDTs when:**
- Multiple writers across the network.
- Offline operation is required.
- Eventual convergence is acceptable.
- Simple data semantics (counters, sets, registers, sequences) suffice.
- Network partitions are common.

**Don't use CRDTs when:**
- Need to enforce uniqueness or non-local invariants.
- Need linearizable reads.
- Need to abort conflicting transactions atomically.
- Data semantics require complex constraints.

For these cases, use Paxos/Raft (consensus), Spanner (consistent global timestamps), or 2PC (atomic transactions).

Many real systems are hybrid: CRDTs for high-throughput data, consensus for critical metadata. AntidoteDB is one example; "transactional CRDTs" (like SwiftCloud and Cure) are another research direction.

## Future of CRDTs

**Bounded CRDTs:** Enforce hard limits on counters or set sizes. Combines CRDTs with token-passing consensus.

**Temporal CRDTs:** Time-travel queries on the CRDT log. "What was the document at time T?"

**Content-addressed CRDTs:** Use immutable hashes for identifiers. Combines with Merkle DAGs for verification.

**Local-first cloud:** Cloud services that respect local-first principles. Backup, search, AI on top of CRDT documents.

**Verifiable CRDTs:** Cryptographic proofs of correct merge. Useful for trustless contexts.

**Probabilistic CRDTs:** Bloom-filter sets, HyperLogLog sets. Lossy but space-efficient.

The next decade will likely see CRDTs in operating systems, end-to-end-encrypted collaboration, and decentralized applications.

## Detailed Comparison: Yjs vs Automerge

Both libraries implement CRDTs for collaborative apps. They differ:

**Yjs:**
- Optimized for real-time collaborative editing.
- Binary internal format, very compact.
- Fast: microsecond-scale operations.
- Plain text, rich text, JSON, XML support.
- Mature, used in Notion, Logseq, etc.
- TypeScript, with WebAssembly for performance.

**Automerge:**
- General-purpose JSON CRDT.
- Richer JSON model.
- Slower than Yjs (millisecond-scale operations).
- Strong cryptographic verification options.
- Newer, smaller ecosystem.
- TypeScript with optional Rust core.

**Trade-offs:**
- Yjs: speed-optimized for editing.
- Automerge: model-rich, cryptographic options.

Both work, both are correct, both have happy users. The choice often comes down to specific use case — editing performance vs JSON modeling fidelity.

## CRDT Storage Patterns

**Append-only log:** Each operation is appended. Replay produces current state. Yjs and Automerge use this internally.

**Materialized view:** Maintain current state alongside log. Reads are fast; writes are reconciliation.

**Snapshots:** Periodic snapshots of materialized state. Discard log entries older than snapshot.

**Differential storage:** Store only the difference between successive snapshots. Compact but slower to read.

**Hybrid approach:** Recent operations in log; older operations compressed into snapshot. Periodic compaction.

The choice affects:
- Read latency (materialized view fastest).
- Storage size (differential most compact).
- Recovery time (snapshots fastest).
- Network bandwidth (log smallest for sync).

## Conflict Resolution Patterns

While CRDTs converge, the converged state may not match user expectations. Patterns for handling conflict:

**Last-Writer-Wins (LWW):** Simple, but loses concurrent writes. Use with caution.

**Multi-Value:** Preserve concurrent writes. Application reconciles. Used in Riak, Cassandra (with proper config).

**Custom merge:** Define application-specific merge logic. E.g., shopping cart: union of items.

**User intervention:** Prompt user to resolve conflicts. Used in Git-like systems.

**Hybrid (CRDT + consensus):** Most operations via CRDT, critical ones via consensus.

Real apps use mixes. Notion combines CRDTs with backend authority for some operations.

## CRDT Integration with Existing Databases

Many databases now support CRDTs natively or via extensions:

**Riak:** Native counter, set, map, register CRDTs.

**Cosmos DB (Microsoft):** Multi-master with custom merge logic. CRDTs underneath.

**PostgreSQL with extensions:** Some extensions (e.g., BDR) provide CRDT-like multi-master.

**Cassandra:** Counter type uses CRDT semantics.

**Redis:** Active-active replication uses CRDTs for built-in types (counters, sets).

**SQLite:** No native CRDT, but libraries (e.g., sqlite-crdt) wrap it.

**MongoDB:** No native CRDT, but multi-master replication can use CRDT-like patterns at app level.

The trend: CRDTs are becoming first-class citizens in databases.

## Performance Engineering for CRDTs

Optimizations specific to CRDT implementations:

**Compaction:** Periodic GC of stable state.

**Compression:** Binary format with delta encoding.

**Lazy evaluation:** Compute materialized view only when read.

**Parallel merge:** Multi-threaded merge for large states.

**Memory-mapped storage:** mmap for large CRDTs.

**Index structures:** B-trees or skip-lists for fast lookup within CRDTs.

**Batching:** Apply many ops in batch, single materialization.

These engineering details determine the difference between research-quality and production-ready CRDT libraries.

## Worked Example: G-Counter Convergence Trace

Three replicas A, B, C, starting from `{}`. Each replica maintains a per-replica-id increment count.

```text
T=0:   A: {}             B: {}             C: {}                       value=0
T=1:   A: {A:1}                                            (A.inc())   value=1
T=2:                     B: {B:5}                          (B.inc x5)  value=5
T=3:                                       C: {C:2}        (C.inc x2)  value=2

# A and B sync (e.g., gossip exchange):
T=4:   A merges B's state: max({A:1}, {B:5}) per key = {A:1, B:5}     value=6
       B merges A's state: max({B:5}, {A:1}) per key = {A:1, B:5}     value=6

# A and B both increment after sync:
T=5:   A: {A:2, B:5}     B: {A:1, B:6}                                 A.value=7, B.value=7
T=6:                                       C still: {C:2}              value=2

# C learns about A's state (via gossip):
T=7:                                       C merges {A:2, B:5}, {C:2}
                                          = {A:2, B:5, C:2}            value=9

# Final convergence (everyone gossips):
T=8:   A: {A:2, B:6, C:2}    B: {A:2, B:6, C:2}    C: {A:2, B:6, C:2}  value=10

# Note: at T=5 A had {A:2,B:5} and B had {A:1,B:6}. After they sync
# (in either order, multiple times), they converge to {A:2,B:6}. Order doesn't
# matter; idempotent under repeat.
```

The merge function is associative + commutative + idempotent — these are exactly the requirements for SEC by Shapiro's theorem.

## Worked Example: OR-Set ABA Scenario

Two replicas A, B. Sequence of operations that demonstrates why OR-Set's tags are necessary.

```text
T=0:  A: {}           B: {}
T=1:  A.add(x)        // A creates tag t1 for x; A: {(x, {t1})}
T=2:  A.remove(x)     // A removes all observed tags for x; A: {(x, {}), tombstones: {t1}}
T=3:  A.add(x)        // A creates new tag t2 for x; A: {(x, {t2}), tombstones: {t1}}

# Now A and B sync (B has only seen up to T=1's state):
T=4:  B receives A's state through T=3
      B.add at this point: B: {(x, {t2}), tombstones: {t1}} — x is in the set
      The "ABA" sequence (add-remove-add) preserved x because t2 ≠ t1

# Without tags (a naive 2P-Set with element-level tombstone):
T=2:  A.remove(x)     // permanent tombstone for x
T=3:  A.add(x)        // ignored! x is in the tombstone set
      Result: x is NOT in the set after re-add — incorrect for use cases
      where re-add is meaningful (e.g., adding back a friend you unfriended).
```

OR-Set's per-add unique tag is what enables re-adding the same element after removal — the tag of the new add wasn't observed at the time of removal, so it survives.

## Worked Example: RGA Insertion-Deletion-Resolution

Two replicas editing collaboratively. RGA (Replicated Growable Array) uses per-character unique IDs:

```text
Initial state: empty document.

Replica A: insert('H', after=∅, id=(A,1))
           insert('i', after=(A,1), id=(A,2))
           Document A: H[A,1] - i[A,2]

(no sync yet)

Replica B: insert('B', after=∅, id=(B,1))
           insert('y', after=(B,1), id=(B,2))
           Document B: B[B,1] - y[B,2]

Sync A→B and B→A.

Conflict: both replicas have insertions referencing 'after=∅' (the start).

RGA tie-break: when two characters reference the same predecessor, sort
by their ID's replica component (lexicographic). A < B, so A's chars come first:

Document (after merge): H[A,1] - i[A,2] - B[B,1] - y[B,2]
Result text: "HiBy"

# Alternatively, B inserts at start with B[B,1] after empty, A inserts H[A,1]:
# - Both have predecessor=∅
# - Tie-break: A < B → A's H first → "HiBy"
```

Now insert + delete trace:

```text
A: insert('!', after=(A,2), id=(A,3))   →  H i ! (with explicit id chain)
A: delete (A,2)  → soft-delete via tombstone (mark id (A,2) as deleted)
   The 'i' character is not removed from the data structure — its tombstone is
   set so that future operations referencing 'after=(A,2)' still resolve.

B (concurrent, hasn't seen A's delete): insert('?', after=(A,2), id=(B,3))
   B's insert references the tombstoned 'i'.
   On merge:
     H[A,1] - i[A,2,tombstoned] - !{B,3 child of i} - !{A,3 child of i}
   Tie-break siblings of (A,2): id-sorted A < B, so A's child first:
   "H!?" (with i tombstoned and rendered invisible)
```

## Yjs Internal Format Details

Yjs serializes operations in a packed binary format (Y.encodeStateAsUpdate).

```text
[update bytes header]
  client_id        varint
  clock            varint    (sequence number for this client's struct)
  struct_count     varint    (number of structs in this update)

For each struct:
  origin_left      (optional) ID = (client_id, clock)   varint pair
  origin_right     (optional) ID = (client_id, clock)
  parent_info       compressed reference to parent shared type
  content_type      tag byte (string, embed, format, deleted)
  content_payload   length-prefixed bytes

[delete set]
  per-client deletion ranges as (clock_start, length) pairs
```

The columnar packing means a 100-character text edit + 50 deletions can compress to ~150 bytes (vs. naive JSON serialization ~10kB).

Y.encodeStateVector returns a state vector summarizing what each peer has seen:

```text
{client_id_1: clock_1, client_id_2: clock_2, ...}
```

When two peers sync via Yjs y-websocket-server, they exchange state vectors first, then the receiver computes "structs you have but I don't" and ships only those.

## Automerge Columnar Storage

Automerge groups changes into columns by attribute (one column for action types, one for actor IDs, one for sequence numbers, one for property keys). This achieves compression via:

- RLE (run-length encoding): repeated values like "all 100 ops by actor=alice" stored once
- Delta encoding: sequence numbers stored as deltas from previous
- Dictionary encoding: actor IDs replaced with small integer indices into a header table

Result: a 10,000-op history can fit in <100KB. Automerge 2.x's binary format is roughly 100x more compact than the JSON-based 1.x format.

## Causal-Stability Garbage Collection (full algorithm)

The goal: when can a tombstone be reclaimed? Answer: when ALL replicas have observed the operation that caused it.

```text
algorithm GC_tombstones(replica R):
  causal_min := min(version[a] for a in known_actors) for each actor's clock at R
  # The "causal frontier" — the lowest clock value any replica has seen

  for each tombstone t in R.tombstones:
    if t.creation_clock < causal_min[t.actor]:
      # All replicas have observed t's creation; safe to reclaim
      remove(t)
```

Practical issue: computing `causal_min` requires querying every replica's current state. In practice this is done out-of-band via gossip protocols or admin coordination. Yjs and Automerge punt on this — they keep tombstones forever, accepting the metadata growth (mitigated by columnar compression).

## When CRDTs Are Wrong

| Use case | Why CRDT fails | Right tool |
|---|---|---|
| Bank account balance ≥ 0 invariant | Concurrent debits can both succeed → negative balance | Paxos / Raft + 2PC |
| Username uniqueness | Two replicas can both register "alice" | Centralized registry / Paxos |
| Inventory: "only 1 item left" | Two buyers both purchase the last item | Lock + transaction |
| Global ordering of events (audit log) | CRDTs don't agree on total order | Raft replicated log |
| Strict access control | Permission revocation racing with use | Centralized auth check |
| Settlement / financial reconciliation | Final-amount disagreement intolerable | Paxos + manual reconciliation |
| Schema-validated data with strict invariants | Concurrent edits can violate schema | Operational Transform with validation, or central validation |

For all of the above, you need a single source of truth and synchronous coordination — exactly what CRDTs by design avoid.

## References

- Shapiro, M., Preguiça, N., Baquero, C., Zawirski, M. (2011). "Conflict-Free Replicated Data Types." INRIA.
- Shapiro, M., Preguiça, N., Baquero, C., Zawirski, M. (2011). "A comprehensive study of Convergent and Commutative Replicated Data Types." INRIA Tech Report.
- Almeida, P.S., Shoker, A., Baquero, C. (2018). "Delta state replicated data types." Journal of Parallel and Distributed Computing.
- Baquero, C., Almeida, P.S., Shoker, A. (2014). "Pure Operation-Based Replicated Data Types." arXiv.
- Almeida, P.S., Baquero, C., Gonçalves, R., Preguiça, N., Fonte, V. (2014). "Scalable and Accurate Causality Tracking for Eventually Consistent Stores." DAIS.
- Oster, G., Urso, P., Molli, P., Imine, A. (2006). "Data consistency for P2P collaborative editing." CSCW.
- Weiss, S., Urso, P., Molli, P. (2009). "Logoot: A Scalable Optimistic Replication Algorithm for Collaborative Editing on P2P Networks." ICDCS.
- Roh, H.G., Jeon, M., Kim, J.S., Lee, J. (2009). "Replicated Abstract Data Types: Building Blocks for Collaborative Applications." JPDC.
- Kleppmann, M., Beresford, A.R. (2017). "A Conflict-Free Replicated JSON Datatype." IEEE Transactions on Parallel and Distributed Systems.
- Kleppmann, M., Wiggins, A., van Hardenberg, P., McGranaghan, M. (2019). "Local-first software: You own your data, in spite of the cloud." Onward!
- Birman, K., Schiper, A., Stephenson, P. (1991). "Lightweight Causal and Atomic Group Multicast." TOCS.
- Hellerstein, J.M., Alvaro, P. (2010). "Keeping CALM: When Distributed Consistency Is Easy." CACM.
- Balegas, V., Duarte, S., Ferreira, C., Rodrigues, R., Preguiça, N., Najafzadeh, M., Shapiro, M. (2015). "Putting Consistency Back into Eventual Consistency." EuroSys.
- Litt, G., Mehrotra, S., van Hardenberg, P. (2022). "Peritext: A CRDT for Collaborative Rich Text Editing." Ink & Switch.
- Lamport, L. (1978). "Time, Clocks, and the Ordering of Events in a Distributed System." CACM.
- Yjs documentation: https://yjs.dev
- Automerge documentation: https://automerge.org
- CRDT.tech: https://crdt.tech
- Ink & Switch publications: https://www.inkandswitch.com

## Worked Examples (Extended)

### Example 1: G-Counter Trace

3 replicas {A, B, C}. Each holds a vector of per-replica counters.

**Initial**: A = [0, 0, 0], B = [0, 0, 0], C = [0, 0, 0]. Value = sum = 0.

**Step 1**: A increments locally. A = [1, 0, 0]. Value at A = 1.

**Step 2**: B increments twice locally. B = [0, 2, 0]. Value at B = 2.

**Step 3**: A and B exchange state.
- A receives [0, 2, 0]. Merges: max([1,0,0], [0,2,0]) = [1, 2, 0]. Value = 3.
- B receives [1, 0, 0]. Merges: max([0,2,0], [1,0,0]) = [1, 2, 0]. Value = 3.

**Step 4**: C increments once. C = [0, 0, 1]. Value at C = 1.

**Step 5**: All exchange.
- All three converge to [1, 2, 1]. Value = 4.

**Property**: Merge is commutative (order doesn't matter), associative (grouping doesn't matter), idempotent (re-applying the same state is a no-op). These three properties are the **semilattice** that makes G-Counter a CRDT.

### Example 2: OR-Set with the ABA Problem

Observed-Remove Set tracks tagged adds. Each `add(x)` creates a new tag; `remove(x)` removes ALL tags for x that the remover has seen.

**Step 1**: A adds "apple" → A = {("apple", tag-A1)}.
**Step 2**: A propagates to B → B = {("apple", tag-A1)}.
**Step 3**: B removes "apple" → B = {} (removed tag-A1).
**Step 4**: A re-adds "apple" → A = {("apple", tag-A2)}.
**Step 5**: A propagates to B → B sees tag-A2 was NOT in its remove set, accepts → B = {("apple", tag-A2)}.

The new add survives because tag-A2 is fresh — even though "apple" was removed, only tag-A1 was removed. The "ABA problem" is solved by always tagging fresh adds.

### Example 3: RGA (Replicated Growable Array) Walkthrough

Used by collaborative text editors. Each character is uniquely identified by `(replica_id, sequence_number)`.

**Initial**: empty document. Tree: virtual root.

**A inserts "H"** → tree: root → (A, 1, "H").
**A inserts "i" after "H"** → root → (A, 1, "H") → (A, 2, "i").
**B (concurrently with A's "i") inserts "!" after "H"** → tree from B's perspective: root → (A, 1, "H") → (B, 1, "!").

When A and B sync:

- A sees `(B, 1, "!")` was inserted after `(A, 1, "H")`.
- A's tree now has `(A, 1, "H")` with TWO children: `(A, 2, "i")` and `(B, 1, "!")`.
- Conflict resolution: order siblings by `(timestamp, replica_id)` descending. Suppose A=1, B=2. Then `(B, 1)` > `(A, 2)` lexicographically, so `(B, 1, "!")` comes first.
- Final tree: `H!i`.

Both replicas converge on `H!i`. Order is determined by the IDs, not by physical clock — strong eventual consistency without coordination.

### Example 4: Yjs Internal Format

Yjs uses **double-linked lists** of items, each with:

```
struct Item {
  ID: (clientId, clock)        // 16 bytes
  origin: ID | null            // left neighbor at insert time
  rightOrigin: ID | null       // right neighbor at insert time
  parent: TypeRef | null
  parentSub: string | null     // for map keys
  content: Content             // the actual character/value
  deleted: bool                // tombstone
}
```

Garbage collection: tombstoned items are merged into "delete sets" — compact run-length encodings of deleted IDs. After sync, peers exchange delete sets along with insertions, allowing GC of tombstones whose deletes everyone has seen.

Wire format: a sequence of `(structId, structSize, structPayload)` triplets, then the delete-set vector. Optimized: most updates are <100 bytes for a single keystroke.

### Example 5: Automerge Columnar Encoding

Automerge stores ops in a **columnar format**: instead of storing ops as records, it stores each FIELD across ops in a separate column.

```
ops = [
  { id: 1, action: "set", key: "x", value: 5 },
  { id: 2, action: "set", key: "y", value: 7 },
  { id: 3, action: "del", key: "x", value: null },
]

// Column-store representation:
ids:    [1, 2, 3]
actions: ["set", "set", "del"]
keys:    ["x", "y", "x"]     // note: "x" repeats — RLE compresses
values:  [5, 7, null]
```

Each column is run-length-encoded and zip-compressed. For a typical document, this achieves 10-20× compression over per-op JSON. Read perf: column scans are CPU-friendly (fits in L1 cache, no pointer chasing).

### Example 6: Causal-Stability Garbage Collection

Tombstones (deleted items) accumulate forever in naive CRDTs. Causal-stability GC: an item is "causally stable" once every replica has seen all operations that could affect it. Once stable, tombstones can be deleted safely.

Algorithm:
1. Each replica tracks a vector clock of "latest operation seen from each peer."
2. Periodically, replicas exchange clocks.
3. Compute `min_clock = min(clocks)` element-wise.
4. Any operation with `(replica_id, seq)` where `seq < min_clock[replica_id]` is causally stable.
5. Tombstones for stable operations can be GC'd.

Cost: requires all replicas to participate in the clock exchange. If a replica is offline indefinitely, GC cannot proceed (or you accept that replica's data may be lost).

## Performance Comparison

| Library | Lang | Insert/sec (avg op) | Wire size (1KB doc, 100 ops) | Notes |
|---------|------|--------------------:|-----------------------------:|-------|
| Yjs | JS/TS | ~100K | ~3 KB | reference impl, fastest in JS |
| Automerge 2.x | Rust + WASM | ~30K | ~1.5 KB (columnar compressed) | Rust core, JS bindings |
| diamond-types | Rust | ~500K | ~2 KB | research-grade, fastest known RGA |
| EGwalker | Rust (research) | ~1M | ~2 KB | event graph walker, alpha |
| Loro | Rust | ~50K | ~2 KB | compact wire, multi-language |
| collabs | TS | ~50K | ~3 KB | extensible framework |

For most apps, Yjs (JS) or Automerge (when you need rich document types like Markdown) are production-grade choices. For high-frequency updates (>10K op/s sustained), drop to Rust-backed diamond-types or EGwalker.
