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

## Yjs Internal Format

Yjs encodes operations as a binary stream. Each "item" has:
- ID: 8 bytes (clientID + clock)
- Origin: previous item's ID (for ordering)
- Right origin: next item's ID (for redundancy)
- Content: variable-length
- Parent: containing structure (for nested CRDTs)
- Parent-sub: key (for Map entries)

Items can be **structurally shared**: contiguous insertions by the same client form a single item. This drastically reduces overhead for typical typing patterns. When items split (due to interleaved concurrent inserts), Yjs splits the structure dynamically.

Yjs's binary format is compact and supports efficient sync: replicas exchange "state vectors" (per-client clocks) to identify missing updates, then exchange only the missing portion.

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

## Worked Example: Two-Replica G-Counter Convergence

Two replicas A and B start with G-Counter [0, 0]. The vector position 0 is for replica A; position 1 is for replica B.

**Step 1:** A increments. State_A = [1, 0].
**Step 2:** B increments twice. State_B = [0, 2].
**Step 3:** A and B exchange states.
- A receives [0, 2], merges with [1, 0] via pointwise max. State_A = [max(1,0), max(0,2)] = [1, 2].
- B receives [1, 0], merges with [0, 2]. State_B = [max(0,1), max(2,0)] = [1, 2].

**Step 4:** Both replicas have State = [1, 2]. Counter value = 1 + 2 = 3.

**Step 5:** A increments again. State_A = [2, 2].
**Step 6:** B doesn't sync; instead it crashes and restarts from [1, 2].
**Step 7:** B increments. State_B = [1, 3].
**Step 8:** A and B exchange.
- A merges [1, 3] with [2, 2] = [2, 3].
- B merges [2, 2] with [1, 3] = [2, 3].

Convergence: both at [2, 3], total = 5. The pointwise-max LUB ensures that even after B's crash and re-increment, the merge correctly accumulates all updates.

This worked example shows: idempotence (B's repeated state isn't double-counted), commutativity (order of merges doesn't matter), associativity (multi-step merges produce the same result).

## Worked Example: OR-Set Add and Remove

Two replicas A and B operate on an OR-Set.

**Step 1:** A adds "apple" with tag (A, 1). State_A = {apple: {(A,1)}}.
**Step 2:** B receives the operation, applies it. State_B = {apple: {(A,1)}}.
**Step 3:** A removes "apple". A removes the tag (A,1) it has observed. State_A = {apple: {}} (empty tag set means absent).
**Step 4:** Concurrently, B adds "apple" again. B generates tag (B, 1). State_B = {apple: {(A,1), (B,1)}}.
**Step 5:** A and B exchange.
- A: merge {apple: {(A,1), (B,1)}} with {apple: {}}. The merge preserves all tags ever added but tracks removed tags. The remove operation removed (A,1) only — it had not observed (B,1). So the result tracks: added = {(A,1), (B,1)}, removed = {(A,1)}. Effective: {(B,1)}. State = {apple: present (via tag B,1)}.
- B: same logic. State = {apple: present}.

The "apple" element is present in the converged state — even though A intended to remove it. This is "add-wins" semantics: a concurrent add wins over a remove that didn't observe it.

This is a feature for many applications (collaborative editing: don't lose B's recent add) but might be wrong for others (deletion is intentional and should propagate).

## Worked Example: RGA Sequence Insert

Document is "AB" with characters at positions (A: id=(R1,1), B: id=(R1,2)). Two replicas A1 and A2.

**Step 1:** A1 inserts "C" between A and B, referencing predecessor id=(R1,1). New char id=(R1,3). Sequence: "ACB".
**Step 2:** A2 also inserts at the same location. A2 inserts "X" referencing predecessor id=(R1,1). New char id=(R2,1). Sequence: "AXB".
**Step 3:** A1 and A2 sync.
- A1 receives X. X has predecessor (R1,1). Among items at this position: C (id=(R1,3)) and X (id=(R2,1)). Order by id: (R1,3) < (R2,1) (assuming R1=1, R2=2 in tiebreaker). So order is C before X. Result: "ACXB".
- A2 receives C. Same reasoning. Result: "ACXB".

Both replicas converge to "ACXB". The insertion order at the conflict point is determined by ID ordering — deterministic.

This example shows how RGA handles concurrent inserts at the same position. Without unique IDs and ordering, replicas could see "ACXB" or "AXCB" depending on order of receipt — diverging.

## Worked Example: Yjs Document Sync

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
