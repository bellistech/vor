# CRDT — Conflict-free Replicated Data Types

Data structures that replicate across nodes and merge without coordination — the math behind real-time collaboration, offline-first apps, and partition-tolerant distributed systems.

## Setup

CRDTs are data structures designed for replication across multiple nodes (or peers, or browsers, or devices) where each replica can be updated independently and concurrently — with no central coordinator — and where the divergent replicas can later be merged back together with a mathematical guarantee of convergence to the same state.

The property they provide is called Strong Eventual Consistency (SEC):

```text
SEC: any two replicas that have received the same set of updates
     have equivalent state.
```

Notice what SEC does NOT promise:

- It does NOT promise that all replicas see updates in the same order.
- It does NOT promise that updates are seen at the same time.
- It does NOT promise linearizability or sequential consistency.
- It does NOT promise that an external observer will see a consistent timeline.

What SEC DOES promise:

- Once two replicas have observed the same set of updates (regardless of arrival order), their visible state is bit-for-bit equivalent.
- Convergence is deterministic — there is no need for tie-breaking decisions, no need for a coordinator, no need for a central clock.
- Replicas can stay disconnected and accept writes locally; reconciliation happens whenever they reconnect.

The math: a CRDT is a data type whose merge function forms a join-semilattice over its state space. Every update produces a new state that is greater-or-equal in the partial order. Merging two states computes the least upper bound (the join). Because the join is commutative, associative, and idempotent, no matter what order updates arrive, no matter how many times duplicates arrive, the result is the same.

```text
Replica A:  ∅ → {a} → {a,b} → {a,b,c}
Replica B:  ∅ → {x} → {x,y}

Sync:       both reach {a,b,c,x,y}
            order doesn't matter; idempotency means duplicates are harmless
            associativity means batching is harmless
            commutativity means message ordering is harmless
```

```text
Lifecycle of a CRDT-backed write:

   client ── local update ──▶ replica
                                │
                                ▼
                        update local state
                                │
                                ▼
                  publish state delta or operation
                                │
              ┌─────────────────┼─────────────────┐
              ▼                 ▼                 ▼
          replica2          replica3          replica4
              │                 │                 │
              ▼                 ▼                 ▼
        merge(local, recv) — commutative, associative, idempotent
```

This sheet exists so you never have to leave the terminal to remember the math, the semantics of OR-Set vs 2P-Set, the difference between state-based and op-based CRDTs, or which library to use for which problem.

## Why CRDTs

CRDTs occupy a corner of the consistency-availability tradeoff space that traditional consensus protocols cannot reach.

vs CP databases (Paxos, Raft, ZooKeeper, etcd):

- Paxos/Raft provide strong consistency by funneling all writes through a leader and requiring quorum agreement before acknowledging.
- This means: writes block during partitions; latency is at least one network round trip to the leader; offline writes are impossible (the client cannot reach the leader, period).
- CRDTs trade that strictness for: partition tolerance (writes never block), low latency (every write is local), offline edits (replicas accept writes while disconnected and reconcile later), peer-to-peer topologies (no leader needed).
- The cost: you cannot enforce arbitrary invariants like "balance ≥ 0" without additional machinery (escrow, reservations, bounded counters with coordination at the boundary).

vs Last-Writer-Wins with manual conflict resolution:

- LWW is what naive systems do: store a timestamp with every value; the higher timestamp wins.
- Problem 1: clock skew. Two replicas with skewed clocks can have the "wrong" write win.
- Problem 2: lost data. The losing write is silently discarded — there is no notification to the user, no merge, no record.
- Problem 3: no provable convergence for complex data. LWW on a register works; LWW on a list does not (concurrent inserts get arbitrarily reordered/lost).
- CRDTs replace LWW with mathematically defined merge semantics that are provably convergent and that preserve concurrent intent (e.g., concurrent inserts in a sequence CRDT both appear).

When to reach for CRDTs:

- Real-time collaborative editing (Google Docs, Notion, Linear, Figma — many of these now use CRDT-flavored algorithms internally).
- Offline-first mobile apps where the device must keep working without a network.
- Geo-distributed systems where wide-area latency makes strong consistency painful.
- Multi-master databases (Riak, AntidoteDB, Redis Enterprise CRDTs).
- Peer-to-peer systems with no central server (libp2p, Earthstar, IPFS-pinned data).

When not to use CRDTs (covered in detail later): hard invariants, single-source-of-truth scenarios, strict schema enforcement at write time.

## Mathematical Foundations

The math underneath CRDTs is order theory and abstract algebra. The relevant structure is a join-semilattice.

Definitions:

- A partial order ≤ over a set S is a relation that is reflexive (x ≤ x), antisymmetric (x ≤ y and y ≤ x implies x = y), and transitive (x ≤ y and y ≤ z implies x ≤ z).
- A join (least upper bound) of two elements x, y is the smallest element z such that x ≤ z and y ≤ z. We write z = x ⊔ y.
- A join-semilattice is a partially ordered set where every two elements have a join.

Properties of the join operation that make CRDTs work:

```text
Commutativity:  x ⊔ y = y ⊔ x
Associativity:  (x ⊔ y) ⊔ z = x ⊔ (y ⊔ z)
Idempotency:    x ⊔ x = x
```

Why these three matter individually:

- Commutativity → message ordering does not matter. A receives B's update then C's update, or C's then B's: same result.
- Associativity → batching does not matter. Merging `(s1 ⊔ s2) ⊔ s3` equals `s1 ⊔ (s2 ⊔ s3)`. You can group merges however you like.
- Idempotency → duplicate delivery does not matter. If a network re-sends the same state, merging it again is harmless. This is the key property that lets CRDTs work over unreliable transports without dedup machinery.

Monotonic merge: every state transition moves the state up in the partial order. State only grows. This is why CRDTs are sometimes called "monotonic data structures."

```text
                   ●  s_final = s1 ⊔ s2 ⊔ s3
                  /|\
                 / | \
                ●  ●  ●     intermediate joins
                 \ | /
                  \|/
                   ●  s_initial (often the empty state ⊥)
```

Bottom element ⊥: the initial state, the identity for the join operation. For a G-Counter, ⊥ is the all-zeros vector. For a G-Set, ⊥ is the empty set.

Pseudocode for the abstract join-semilattice:

```text
type State = ...
fn merge(a: State, b: State) -> State:
    # must satisfy:
    #   merge(a, b) == merge(b, a)               (commutative)
    #   merge(merge(a, b), c) == merge(a, merge(b, c))   (associative)
    #   merge(a, a) == a                          (idempotent)
    ...
```

Any data type whose state forms a join-semilattice and whose merge function is the join operation is automatically a CvRDT (state-based CRDT). This is the whole trick.

## Strong Eventual Consistency Theorem

The foundational result, due to Shapiro, Preguiça, Baquero, and Zawirski (INRIA 2011):

Theorem (informal): For replicas of a CRDT (state-based or op-based) under reliable eventual delivery, any two replicas that have received the same set of updates have equivalent state.

For state-based CRDTs (CvRDT) the conditions are:

- The state space forms a join-semilattice.
- The merge function is the join operation of that semilattice.
- The local update function is monotonic (every update produces a state ≥ the previous state in the partial order).

For op-based CRDTs (CmRDT) the conditions are:

- Operations are delivered to every replica exactly once.
- Operations are delivered in causal order (or, equivalently, operations that are concurrent commute).
- The effect function (apply-operation-to-state) commutes for concurrent operations.

In either case, given the same set of updates, the resulting state is the same. This is the Strong Eventual Consistency property: "strong" because convergence is automatic and deterministic (no tie-breaker needed), "eventual" because there is no real-time bound on when convergence occurs.

```text
SEC vs Strong Consistency (Linearizability):

   Strong consistency:        all replicas appear to see the same total
                              order of operations at every moment;
                              requires consensus (Paxos/Raft);
                              writes block during partitions.

   Strong Eventual Consistency: replicas may diverge transiently, but
                              once they have the same set of updates
                              they have the same state; no consensus
                              needed; writes never block; partition
                              tolerance comes for free.
```

The proof sketch: because the join is commutative, associative, and idempotent, the result of folding the join over a set of states is independent of the order or multiplicity of states. Therefore two replicas with the same set of updates compute the same fold, hence the same final state.

This is the result that makes CRDTs feel almost too good to be true. The catch is that not every data type can be made into a CRDT — you have to design the operations and state to fit the semilattice structure. The rest of this sheet is a tour of the data types that have been figured out.

## CRDT Taxonomy

Two main flavors, with a third hybrid:

State-based CRDTs (CvRDT — Convergent Replicated Data Type):

- Replicas exchange their full state.
- Merge function is the join operation of a semilattice.
- Idempotent by construction (merging the same state twice has no effect).
- No requirement on message ordering — works over any reliable broadcast (or even unreliable with retries).
- Bandwidth-heavy: every message carries the entire state.

Op-based CRDTs (CmRDT — Commutative Replicated Data Type):

- Replicas exchange operations (the deltas of intent, like "increment", "add x", "delete tag-7").
- Operations must commute for any pair of concurrent operations.
- OR operations must be delivered in causal order (so non-commuting ops never appear concurrently).
- Bandwidth-light: only the operation flies on the wire.
- Requires reliable causal broadcast at the messaging layer (vector clocks, Lamport clocks, or similar).

Delta-state CRDTs (δ-CvRDT — Delta CRDT):

- Hybrid: state-based semantics but only ship "delta-states" (the chunk of state that changed since last sync).
- Smaller payloads than CvRDT, but still works over reliable-eventual delivery (no causal broadcast needed).
- Requires a delta-merge function that composes deltas correctly.

```text
                   ┌────────────────────────────────────────────┐
                   │  Choose by transport:                      │
                   │                                            │
                   │   reliable eventual + retry → CvRDT/δCRDT  │
                   │   reliable causal broadcast → CmRDT        │
                   │                                            │
                   │  Choose by bandwidth:                      │
                   │                                            │
                   │   small ops, big state → CmRDT             │
                   │   small state, complex ops → CvRDT/δ       │
                   └────────────────────────────────────────────┘
```

## State-based vs Op-based Tradeoffs

```text
                       │ State-based    │ Op-based       │ Delta-state
  ─────────────────────┼────────────────┼────────────────┼──────────────
  Payload size         │ full state     │ single op      │ small delta
  Idempotency          │ free (join)    │ needs at-most- │ free
                       │                │ once delivery  │
  Causal delivery      │ not needed     │ required       │ not needed
  Convergence          │ guaranteed by  │ guaranteed by  │ guaranteed by
                       │ semilattice    │ commutativity  │ semilattice
  Bandwidth (typical)  │ heavy          │ light          │ medium
  CPU at merge         │ medium-heavy   │ light          │ medium
  Best for             │ small state,   │ rich-history   │ medium state,
                       │ unreliable net │ + reliable bus │ best general
```

State-based example flow (CvRDT G-Counter):

```text
Replica A: state = {A:5}
Replica B: state = {A:5, B:3}    # B has heard from A and added own
Replica C: state = {C:2}

Network triggers gossip:
  A sends {A:5} to C
  C merges: {A:5, C:2}
  B sends {A:5, B:3} to C
  C merges: {A:5, B:3, C:2}

Even if A's first message arrives twice at C:
  C state stays {A:5, C:2} → {A:5, C:2}    (idempotent)
```

Op-based example flow (CmRDT G-Counter):

```text
Replica A: applies inc()  → broadcasts op "+1 from A"
Replica B: applies inc()  → broadcasts op "+1 from B"
Replica C: receives both ops in either order

If C receives "B+1" then "A+1": value 0 → 1 → 2
If C receives "A+1" then "B+1": value 0 → 1 → 2

Same final value because + is commutative.
But: causal broadcast required so reset/inc don't reorder.
```

Both strongly converge given the right delivery guarantees. Pick the model that matches your transport.

## Delta-state CRDTs

Delta-state CRDTs (Almeida et al, "Delta State Replicated Data Types") solve the bandwidth problem of state-based CRDTs without losing the simplicity.

The idea: instead of shipping the entire state on every sync, ship only the delta — a small piece of state representing the recent changes — and rely on the same join semantics to merge.

A δ-CRDT has:

- A normal state space and join (the underlying CvRDT).
- A delta-state space (a sub-lattice or join-compatible subset).
- A delta-mutator: each local operation produces a delta.
- A delta-merge: applying a delta into a state is a join.

```text
Replica A:  state = S_A
            local op produces δ_A
            new state = S_A ⊔ δ_A
            send δ_A on the wire (small)

Replica B:  state = S_B
            receive δ_A
            new state = S_B ⊔ δ_A
```

The trick is that replicas track which deltas they have already shipped to which peers, and only send the unsent deltas. This requires a "delta interval" log — a buffer of recent deltas keyed by sequence number.

Pseudocode:

```text
struct DeltaCvRDT:
    state: S
    delta_buffer: Map[seq, S_delta]
    next_seq: u64

fn local_update(d_op):
    delta = compute_delta(state, d_op)
    state = merge(state, delta)
    delta_buffer[next_seq] = delta
    next_seq += 1

fn sync_to(peer):
    last_acked = peer_acks[peer]
    bundle = merge_all(delta_buffer[last_acked..next_seq])
    send(peer, bundle, next_seq)

fn on_receive(remote_bundle):
    state = merge(state, remote_bundle)
```

Real-world: many production CRDT systems (Riak's later versions, Akka Distributed Data) use delta-style propagation under the hood while retaining state-based correctness.

## Counters

The simplest CRDTs.

### G-Counter (Grow-only Counter)

A counter that only goes up.

State: a vector of per-replica integer counters.

```text
type GCounter = Map[replica_id, u64]

fn inc(self, replica_id):
    self[replica_id] += 1

fn value(self) -> u64:
    return sum(self.values())

fn merge(a: GCounter, b: GCounter) -> GCounter:
    # pointwise max
    result = {}
    for k in keys(a) ∪ keys(b):
        result[k] = max(a.get(k, 0), b.get(k, 0))
    return result
```

Why pointwise max is a join:

- Each replica's slot is monotonically growing (a replica only increments its own slot).
- max is commutative, associative, idempotent.
- A vector with pointwise ≤ as the partial order forms a semilattice with pointwise max as the join.

Worked example:

```text
Replica A: inc, inc, inc           → {A:3}
Replica B: inc, inc, inc, inc, inc → {B:5}
Replica C: inc, inc                → {C:2}

A ⊔ B = {A:3, B:5}                  value = 8
(A ⊔ B) ⊔ C = {A:3, B:5, C:2}       value = 10

A ⊔ (B ⊔ C) = A ⊔ {B:5, C:2}
            = {A:3, B:5, C:2}        value = 10  ✓
```

### PN-Counter (Positive-Negative Counter)

A counter that goes up and down.

State: two G-Counters, one for increments and one for decrements.

```text
type PNCounter = (P: GCounter, N: GCounter)

fn inc(self, replica_id):  self.P.inc(replica_id)
fn dec(self, replica_id):  self.N.inc(replica_id)

fn value(self) -> i64:
    return self.P.value() - self.N.value()

fn merge(a, b) -> PNCounter:
    return (merge(a.P, b.P), merge(a.N, b.N))
```

Why two counters? Because subtraction is not monotonic — if you allowed direct decrement, the merge would have to know "did A decrement to 5 because it had 6 and decremented once, or because it had 10 and decremented 5 times?" Splitting into two grow-only counters and subtracting at read time keeps the semilattice intact.

```text
Replica A: P={A:3}, N={A:1}        value = 2
Replica B: P={B:2}, N={B:0}        value = 2

A ⊔ B: P={A:3, B:2}, N={A:1, B:0}  value = 4
```

## Sets

Sets are richer than counters and have several variants depending on how you want concurrent add/remove to resolve.

### G-Set (Grow-only Set)

The simplest set.

```text
type GSet[E] = Set[E]

fn add(self, e):  self.insert(e)
fn contains(self, e) -> bool:  return e in self

fn merge(a, b) -> GSet:
    return a ∪ b
```

Union is commutative, associative, idempotent. Done.

Limitation: cannot remove anything.

### 2P-Set (Two-Phase Set)

Adds and removes, with the rule that once removed an element cannot be re-added.

```text
type TwoPSet[E] = (A: GSet[E], R: GSet[E])

fn add(self, e):     self.A.add(e)
fn remove(self, e):  self.R.add(e)
fn contains(self, e) -> bool:
    return e in self.A and e not in self.R

fn merge(x, y) -> TwoPSet:
    return (merge(x.A, y.A), merge(x.R, y.R))
```

R is a tombstone set. The "two-phase" name refers to the lifecycle of an element: added once, optionally removed once, then frozen as removed forever.

Gotcha: if you intend to support re-adding elements (a typical contact list, todo list, friend list), 2P-Set is wrong — once you remove something it stays removed even if a concurrent or later add says otherwise.

### LWW-Element-Set (Last-Writer-Wins Element Set)

Each add or remove is timestamped; the most recent operation wins.

```text
type LwwElementSet[E] = (
    A: Map[E, timestamp],
    R: Map[E, timestamp]
)

fn add(self, e, ts):
    if ts > self.A.get(e, -∞):
        self.A[e] = ts

fn remove(self, e, ts):
    if ts > self.R.get(e, -∞):
        self.R[e] = ts

fn contains(self, e) -> bool:
    a_ts = self.A.get(e, -∞)
    r_ts = self.R.get(e, -∞)
    return a_ts > r_ts        # or >= depending on bias

fn merge(x, y) -> LwwElementSet:
    A = pointwise_max(x.A, y.A)
    R = pointwise_max(x.R, y.R)
    return (A, R)
```

Tie-break on equal timestamps: pick a fixed bias (add-wins or remove-wins) and stick with it. Use a (timestamp, replica-id) tuple for total order.

Gotcha: LWW requires comparable timestamps across replicas. Wall-clock timestamps drift; use Lamport clocks or hybrid logical clocks (HLC) for safety.

### OR-Set (Observed-Remove Set)

The canonical "concurrent add/remove" set. Supports re-adding elements after removal. The semantics: a remove only removes the add-tags it has observed; concurrent adds are preserved.

```text
type ORSet[E] = (
    elements: Map[E, Set[Tag]],   # element -> live tags
    tombstones: Set[Tag]           # removed tags
)

fn add(self, e):
    tag = unique_tag()             # e.g., (replica_id, counter)
    self.elements[e].add(tag)
    return tag

fn remove(self, e):
    for tag in self.elements[e]:
        self.tombstones.add(tag)
    self.elements[e].clear()       # locally; merge restores from peers

fn contains(self, e) -> bool:
    return any(tag not in self.tombstones for tag in self.elements[e])

fn merge(x, y) -> ORSet:
    elements = {}
    for e in keys(x.elements) ∪ keys(y.elements):
        elements[e] = (x.elements.get(e, ∅) ∪ y.elements.get(e, ∅))
    tombstones = x.tombstones ∪ y.tombstones
    # filter tombstoned tags
    for e, tags in elements.items():
        elements[e] = tags - tombstones
    return (elements, tombstones)
```

The unique tag ensures that if Replica A adds x, Replica B never sees that add, and Replica B issues a remove(x), B's remove only removes the tags B knew about — A's add tag survives the merge, and x persists.

Add-wins vs remove-wins:

- Add-Wins (AW-Set / OR-Set): on a concurrent add and remove of the same element, add wins (the element appears in the merged state). This is the OR-Set as described above.
- Remove-Wins (RW-Set): on a concurrent add and remove, remove wins (the element does not appear). Implemented by marking the element itself with a remove-timestamp that overrides untimed adds.

Yjs, Automerge, Riak's set type all default to add-wins (it's the more intuitive choice for human collaboration).

## Registers

A register holds a single value.

### LWW-Register (Last-Writer-Wins Register)

```text
type LWWRegister[T] = (value: T, ts: timestamp)

fn write(self, v, ts):
    if ts > self.ts:                # or (ts, replica) > (self.ts, self.replica)
        self.value = v
        self.ts = ts

fn read(self) -> T:
    return self.value

fn merge(a, b) -> LWWRegister:
    if a.ts > b.ts:
        return a
    elif b.ts > a.ts:
        return b
    else:
        return tiebreak(a, b)        # by replica id, lexicographic, etc.
```

Used pervasively in distributed databases (Cassandra cells, DynamoDB items with last-write-wins) and as the "primitive value" type inside larger CRDTs (Automerge uses LWW for primitive scalars; Yjs uses LWW for the value of a Y.Map entry).

The clock skew problem is real here. Use Lamport clocks (a per-replica counter) or hybrid logical clocks (HLC, a (physical, logical) tuple) to ensure monotonicity across replicas.

### MV-Register (Multi-Value Register)

When concurrent writes happen, keep both. The application reads all concurrent values and resolves at the application layer.

```text
type MVRegister[T] = Set[(value: T, vector_clock: VC)]

fn write(self, v, my_replica, my_vc):
    # drop entries causally dominated by my_vc
    self = { (val, vc) for (val, vc) in self if vc not_dominated_by my_vc }
    self.add((v, my_vc))

fn read(self) -> Set[T]:
    return {val for (val, vc) in self}

fn merge(a, b) -> MVRegister:
    # union, then drop dominated entries
    s = a ∪ b
    return { (v, vc) for (v, vc) in s if not exists (v', vc') in s where vc < vc' }
```

When read returns multiple values, the application has to decide what to do. Riak Last-Write-Wins-disabled buckets behave like this; the client must merge siblings.

## Maps / Dictionaries

A map of keys to CRDT values.

### OR-Map

Keys behave like an OR-Set; values are themselves CRDTs.

```text
type ORMap[K, V_CRDT] = Map[K, (tags: Set[Tag], value: V_CRDT)]

fn put(self, k, v_op):
    if k not in self:
        self[k] = (tags: {unique_tag()}, value: V_CRDT.empty())
    apply_op(self[k].value, v_op)

fn remove(self, k):
    # tombstone tags
    self[k].tags.tombstone()

fn merge(a, b) -> ORMap:
    result = {}
    for k in keys(a) ∪ keys(b):
        tags = merge_tags(a.get(k).tags, b.get(k).tags)
        if tags is empty (all tombstoned):
            continue              # or keep as deleted
        value = merge(a.get(k).value, b.get(k).value)
        result[k] = (tags, value)
    return result
```

The key observation: nested CRDTs compose. An OR-Map of OR-Sets is itself a CRDT. An OR-Map of OR-Maps of LWW-Registers is itself a CRDT. This compositional property is what gives Automerge and Yjs their richness — JSON-shaped data falls out naturally.

### LWW-Map

A simpler variant: each key has a LWW-Register value.

```text
type LWWMap[K, V] = Map[K, LWWRegister[V]]
```

Used when you don't need rich nested CRDT semantics — for example, a profile object where each field is a scalar.

### The "remove all" challenge

Operations like "clear the map" or "delete all entries with prefix X" are tricky because they must be expressed as CRDT operations, which usually means tombstoning every key — and that interacts with concurrent additions (a concurrent add to a key being cleared: should it survive or not?).

The propagating-tombstone-vs-causal-stability tradeoff: keeping tombstones forever bloats metadata; reclaiming them too early risks resurrecting deleted data when a delayed message finally arrives. Causal-stability detection (tracking when all replicas have certainly seen a remove) is the principled solution.

```text
                  +-------------------+
                  |  pending remove   |   tag still tombstoned everywhere
                  |  (not yet stable) |   metadata not yet reclaimable
                  +-------------------+
                            │
                            ▼
                  +-------------------+
                  | causally stable   |   all replicas certainly have it
                  | (all peers ack'd) |   safe to GC the tombstone
                  +-------------------+
```

## Sequence / List CRDTs

The hardest CRDT family. The challenge: model an ordered sequence (a string of characters, a list of items) such that concurrent inserts and deletes converge to a sensible order.

### WOOT (Without Operational Transform)

Each character has a unique identifier (replica_id, sequence_number) plus a pair of "previous" and "next" character ids representing the position context at insertion time.

```text
char W = (id, prev_id, next_id, value, visible)
```

Inserts find the right position by comparing prev/next against the current state; concurrent inserts at the same position get a deterministic order based on id comparison.

Pros: provably correct, simple semantics.
Cons: O(n) per insert in the naive implementation; tombstones grow forever.

### Logoot

Each position is a fractional identifier: a sequence of (digit, replica_id) pairs treated as an arbitrary-precision rational number.

```text
pos("a") = [(5, A1)]
pos("b") = [(7, A1)]
pos("c") = [(6, A1)]              # inserted between a and b
pos("d") = [(6, A1), (5, B1)]     # inserted between c and b — needs more digits
```

The insert algorithm picks an identifier strictly between the left and right neighbors. If there's no integer in between (5 and 6), it appends another digit.

Pros: any element identified by a single position string, so reads are simple.
Cons: position-id explosion. With many concurrent inserts at the same spot, position ids grow without bound.

### Treedoc

Positions are paths in a binary tree. Each insert chooses left or right of the nearest neighbor; the path becomes the position id.

```text
                  root
                  / \
                 0   1
                /\   /\
               0 1  0 1
```

Pros: balanced inserts → log-depth ids.
Cons: requires periodic rebalancing or grows arbitrarily deep with adversarial patterns.

### LSEQ

A bounded-growth variant of Logoot that switches between dense and sparse identifier strategies per depth.

Pros: amortized bounded id length.
Cons: complex to implement correctly.

### RGA (Replicated Growable Array)

The most influential sequence CRDT — used by Yjs (in spirit), Automerge, and most modern systems.

Each character carries:

- A unique id (replica_id, lamport_timestamp).
- A "left origin" — the id of the character it was inserted to the right of.
- A visible/tombstoned flag.

```text
char Char = (id, origin_id, value, visible)
```

Insertion algorithm: find the origin character; walk right past any concurrent inserts that should come before this one (decided by id comparison); insert.

```text
       origin            new char
   ─── [c1] ────────── [c2 (origin=c1.id)] ─────
                         │
                         ▼
   if a concurrent insert c2' with origin=c1.id arrives:
       order c2 and c2' by id (lamport ts, then replica id)
```

Deletion: mark visible=false. The character stays in the structure as a tombstone (for future inserts to reference), but does not appear in the visible string.

RGA is O(n) traversal in the naive form; production implementations (Yjs, Diamond Types) use balanced trees or skip lists for O(log n) inserts.

### Yjs YText

Yjs is the production-grade RGA-flavored CRDT for collaborative text in browsers. It is the de facto standard for ProseMirror, TipTap, BlockNote, and is used inside Notion-clones, Linear, and many editor projects.

Key design choices:

- Each item has an id (clientID, clock).
- Items form a linked structure (left/right neighbors).
- Deletion uses a separate "DeleteSet" rather than per-item tombstones (better GC).
- Binary encoding: varint, run-length, delta encoding — compact wire format.
- Garbage collection: deleted items can be collapsed once causal-stability is detected.

```text
+-------+-------+-------+-------+
|  H    |  e    |  l    |  l    |  ...
| id=A1 | id=A2 | id=A3 | id=A4 |
+-------+-------+-------+-------+
       structure: doubly linked items in document order

DeleteSet: { (A1, 1), (A4, 2) }   # range [A1.clock+0..A1.clock+1) deleted
```

### Diamond Types

Rust-based, very fast (10-100x faster than Yjs in benchmarks). Uses the eg-walker (event graph walker) algorithm: replays the operation graph efficiently using b-tree-indexed positional state.

### Automerge text

Automerge embeds RGA-style sequences inside its broader JSON CRDT. Recent Automerge versions (2.x) use columnar storage for compactness — values, ids, ops are stored in column-oriented arrays which compress well.

## JSON CRDTs

The frontier of CRDT design: how do you make arbitrary JSON-shaped data into a CRDT?

### Automerge

Automerge is the canonical JSON CRDT library. State is a tree of:

- Objects (OR-Map of keys to values).
- Lists (RGA).
- Primitive values (LWW-Register).
- Counters (PN-Counter).
- Text (RGA optimized for character-level edits).

```text
{
  "title": "Notes",                // LWW string
  "tags": ["urgent", "personal"],  // RGA list
  "counts": { "views": 42 },       // OR-Map containing PN-Counter
  "body": <text-CRDT>              // RGA text
}
```

Each operation produces an op (set, insert, delete, increment) tagged with (actor, sequence). Operations form a DAG; merge is "union of operations, applied in causal order, with the appropriate CRDT semantics for each type."

```text
ops: [
  (A:1) set title "Notes"
  (A:2) insert tags[0] "urgent"
  (B:1) insert tags[0] "personal"     # concurrent with A:2
  (A:3) increment counts.views by 1
  (B:2) increment counts.views by 1   # concurrent with A:3
]

After merge:
  title = "Notes"
  tags  = ["urgent", "personal"]   or ["personal", "urgent"]   per RGA
  counts.views = 2                  (PN-Counter sums)
```

Tradeoffs:

- "Schema is implicit" — there's no enforced type. If A writes title as a string and B writes title as an object, the conflict is handled (LWW wins for primitives; for nested-vs-primitive, the merge picks one, often based on op order). This is more flexible than schema-validated systems but riskier.
- Rich history: every op is retained, so you get free undo, time-travel, blame, branch-and-merge.

### Conflict resolution semantics

Default Automerge semantics:

- Concurrent set on a primitive: LWW.
- Concurrent insert on a list: both insertions appear, ordered by RGA rules.
- Concurrent delete and modify on a value: depends on type; usually delete wins for the slot, modify wins for the surviving value.

Yjs is similar in spirit but is JS-first and optimized for editor integration. Automerge prioritizes data-structure correctness and history; Yjs prioritizes speed and binary compactness.

## Counter Variants

### Bounded Counter (capped)

A counter with a maximum (or minimum) bound. The challenge: enforcing the bound requires coordination, but CRDTs don't coordinate.

Riak's bounded counter solution: assign each replica a "reservation" of allowed increments. Replicas can spend their reservation locally without coordination. When they run low, they coordinate to redistribute reservations.

```text
Bound: total ≤ 100
Replicas A, B, C each reserve 33 (one slack)

A spends 30 locally       → no coordination needed
A wants 31st              → must request more reservation from B or C
                            (this is an explicit coordination round)
```

This is sometimes called "escrow" or "demarcation." It's a hybrid: CRDT semantics in the common case, consensus only when bounds threaten to be violated.

### Resettable Counter

A counter that can be reset to zero. Naively this breaks the semilattice (a reset is a downward move).

Solutions:

- "Logical reset" via tombstoning: tombstone all increment-tags up to a reset point, so the visible value drops. Adds metadata.
- Epoch-based: each reset starts a new epoch, and replicas only sum increments from the latest epoch.

The trick is making concurrent (reset, increment) converge sensibly. Most production systems opt for "reset wins" or "increment wins" with a documented bias.

## Threshold + Causal Tracking

Many CRDTs require tracking which operations a replica has seen — not just "did I see op X" but "what is the full causal history of operations I've seen up to now."

### Causal Context

A causal context is a compact summary of "operations I've observed":

```text
type CausalContext = Map[replica_id, max_sequence_number_seen]
```

(This is essentially a vector clock, sometimes called a version vector when it's used for CRDT bookkeeping rather than message delivery.)

It's needed by:

- OR-Set: to know which add-tags I have observed (so my remove can target them).
- OR-Map: same idea for keys.
- Causal-broadcast layers underneath op-based CRDTs.

```text
Replica A's causal context after some ops:
   { A: 7, B: 3, C: 0 }   # I've seen 7 of A's ops, 3 of B's, 0 of C's

When A wants to remove element e, it tombstones every tag (replica, n)
of e where n ≤ A's-knowledge-of-replica's-progress.
```

### Without Causal Tracking

If you don't track causality, you can't tell "this remove is about an add I haven't yet seen" vs "I have everything; this remove targets something genuinely missing." The result: orphaned operations, lost adds, or incorrectly-resurrected deletes.

The classic OR-Set bug: a naive implementation that just compares element-existence (without tags) loses concurrent re-adds.

## Worked Example — G-Counter convergence

Three replicas A, B, C start at the empty state ⊥ = {}.

Step 1: each replica receives some local increments.

```text
A: inc, inc, inc          state = {A:3}
B: inc, inc, inc, inc, inc state = {B:5}
C: inc, inc                state = {C:2}
```

Step 2: A and B sync (A sends to B, B sends to A).

```text
A.merge(B's state)
  = {A:3} ⊔ {B:5}
  = {A:3, B:5}    (pointwise max; missing keys treated as 0)

B.merge(A's state) → same:  {A:3, B:5}

A's value = 3 + 5 = 8
B's value = 3 + 5 = 8
C's value = 2          (still hasn't synced)
```

Step 3: B syncs to C.

```text
C.merge(B's state)
  = {C:2} ⊔ {A:3, B:5}
  = {A:3, B:5, C:2}

C's value = 3 + 5 + 2 = 10
```

Step 4: order doesn't matter — replay step 2 and 3 in reverse and the final state is the same.

```text
If C synced to A first, then A synced to B:
  A.merge({C:2}) = {A:3, C:2}
  B.merge({A:3, C:2}) = {A:3, B:5, C:2}
  Same final {A:3, B:5, C:2}.
```

Step 5: idempotency — A receives B's state again.

```text
A.merge({A:3, B:5}) where A already has {A:3, B:5}
  = pointwise max: {A:3, B:5}
  = unchanged. ✓
```

Step 6: associativity — batch sync.

```text
batch = merge(B's state, C's state) = {B:5, C:2}
A.merge(batch) = {A:3, B:5, C:2}        ✓ same as one-at-a-time.
```

Convergence is automatic. No coordination, no leader, no consensus.

## Worked Example — OR-Set convergence

Two replicas A, B. Initial state: empty.

Step 1: A adds x, B adds x. Each generates a unique tag.

```text
A.add(x) → tag t1=(A,1)
   A.elements = { x: {t1} }

B.add(x) → tag t3=(B,1)
   B.elements = { x: {t3} }
```

Step 2: A also adds y.

```text
A.add(y) → tag t2=(A,2)
   A.elements = { x: {t1}, y: {t2} }
```

Step 3: B removes x. B has only seen its own t3, so it can only tombstone t3.

```text
B.remove(x):
   B.tombstones += {t3}
   B.elements = { x: {} }   # locally cleared
```

Step 4: A and B sync.

```text
merge(A, B):
   elements:
     x: {t1} ∪ {} = {t1}     # A's t1 is preserved
     y: {t2} ∪ ∅ = {t2}
   tombstones: ∅ ∪ {t3} = {t3}

   filter: drop tags in tombstones
     x: {t1} - {t3} = {t1}   # t1 not tombstoned, so x stays
     y: {t2} - {t3} = {t2}

   merged.contains(x) = true  ✓ (because t1 survived B's remove)
   merged.contains(y) = true  ✓
```

The remove only removed what B had observed (t3). A's add (t1) was concurrent and unobserved, so it survives.

Step 5: a later remove that observes everything.

```text
After sync, both A and B have:
   elements: { x: {t1}, y: {t2} }
   tombstones: {t3}

Now A.remove(x):
   A.tombstones += {t1}
   A.elements = { y: {t2} }

After next sync:
   tombstones = {t1, t3}
   x: {t1} - {t1, t3} = ∅   # x truly gone
   y: {t2} - {t1, t3} = {t2}
```

This is the "observed-remove" semantic: you remove what you've observed; concurrent unobserved adds survive.

## Worked Example — RGA text editing

Two replicas A, B start with an empty text CRDT.

Step 1: A inserts "hello" at the start (origin = ⊥ = beginning).

```text
A inserts characters with ids (A,1)..(A,5):
  ┌───┬───┬───┬───┬───┐
  │ h │ e │ l │ l │ o │   each has origin = previous, all (A, i)
  └───┴───┴───┴───┴───┘
   A1   A2   A3   A4   A5

A's visible string: "hello"
```

Step 2: concurrently, B inserts " world" at the end. But B's view of the document at this moment: B might or might not have seen A's hello, depending on sync timing.

Case A: B has already received A's "hello".

```text
B's state before insert: "hello"
B inserts " world" with origin = A5 (the 'o'):
  characters (B,1)..(B,6)

B's visible string: "hello world"
```

Case B: B has not received A's "hello"; B inserts at the start of its empty document.

```text
B's state before insert: ""
B inserts " world" at the start, origin = ⊥
  characters (B,1)..(B,6) all with origin ⊥

B's visible string: " world"
```

Step 3: A and B sync.

In case A: trivial — B already has hello, A receives world's ops, applies them in order.

```text
Final structure on both replicas:
  h e l l o ' ' w o r l d
  A1 A2 A3 A4 A5 B1 B2 B3 B4 B5 B6

Visible: "hello world"
```

In case B: both replicas have inserts with origin ⊥. RGA's tie-break rule: when multiple inserts share the same origin, order them by their own id (lamport, then replica). Suppose A < B as replica order.

```text
Both inserts have origin ⊥; arrange by id:
   A1 A2 A3 A4 A5 (hello, A's ids)
   B1 B2 B3 B4 B5 B6 (' world', B's ids)

Sorted at origin ⊥ position:
  if A's lamport timestamps < B's: hello first
    "hello world"
  if B's lamport timestamps < A's: world first
    " worldhello"
```

The deterministic tie-break ensures both replicas agree, even if a human would prefer a different order. (For real collaborative editing, you typically have causal sync that prevents this exact pathology — the remote replica only inserts after seeing local context.)

Step 4: deletions. A deletes "h".

```text
A1's visible flag → false (tombstone)

After sync, B applies the same flag-flip; both visible: "ello world"

Tombstones remain in the structure to anchor potential future inserts.
GC: once causally stable, tombstones can be collapsed.
```

## Causal Delivery

Op-based CRDTs require operations to arrive in causal order — that is, if operation B was created after seeing operation A, then A is delivered before B.

Why: many CRDT operations only commute when they are concurrent. Non-concurrent ops with happens-before relationships often need to be applied in causal order to preserve correct semantics.

Example: in RGA, inserting character X with origin Y is meaningful only after Y has been delivered. If X arrives first, you cannot resolve its origin.

Causal order is implemented via vector clocks: each op carries a vector clock summarizing its causal history. The receiving replica checks that all causal predecessors have already been applied; if not, it buffers the op.

```text
Replica A: op a1 with VC {A:1}
Replica A: op a2 with VC {A:2}
Replica B: receives a2 first

B checks a2.VC = {A:2}. Has B applied {A:1}? No → buffer a2.
B receives a1, VC {A:1}. Apply.
B re-checks buffered a2: now ready. Apply.
```

Infrastructure: Erlang/OTP gen_server, Akka cluster, Riak's distributed broadcast all provide causal multicast as a service to layers above.

## Reliable Causal Broadcast

The algorithm underneath. Each replica maintains:

- A vector clock VC.
- An outgoing op buffer.
- An incoming-buffer for ops awaiting causal predecessors.

```text
fn local_op():
    VC[self] += 1
    op = make_op(VC.copy())
    apply_locally(op)
    broadcast(op)

fn on_receive(op):
    # ready iff op.VC[op.origin] == VC[op.origin] + 1
    # and op.VC[k] ≤ VC[k] for all k != op.origin
    if causally_ready(op, VC):
        apply(op)
        VC[op.origin] = op.VC[op.origin]
        check_pending()           # see if buffered ops are now ready
    else:
        pending.add(op)

fn check_pending():
    progress = true
    while progress:
        progress = false
        for op in pending:
            if causally_ready(op, VC):
                apply(op)
                VC[op.origin] = op.VC[op.origin]
                pending.remove(op)
                progress = true
```

The missing-message buffer (pending) absorbs out-of-order delivery. Once the missing causal predecessor arrives, the buffered ops cascade into application.

```text
Vector-clock causal-readiness check:
  op.VC = { A:5, B:2, C:1 }     op was created on A
  my VC = { A:4, B:2, C:1 }

  op.VC[A] == my VC[A] + 1     ?   5 == 5  ✓
  op.VC[B] ≤ my VC[B]          ?   2 ≤ 2   ✓
  op.VC[C] ≤ my VC[C]          ?   1 ≤ 1   ✓
  → ready, apply.
```

## Garbage Collection

Tombstones and dead metadata accumulate. Without GC, CRDTs grow without bound.

The key concept is causal stability: an operation is causally stable when every replica is known to have observed it. Once stable, the metadata it generated (tombstones, version vectors, etc.) can be reclaimed.

Algorithms:

Anti-entropy with summary digest:

- Periodically, replicas exchange summary VCs.
- A replica computes the minimum VC across all replicas.
- Any operation with VC ≤ minimum-VC is causally stable.
- GC anything keyed by such operations.

```text
Replicas A, B, C have VCs:
  A: {A:10, B:7, C:5}
  B: {A:10, B:7, C:5}
  C: {A:10, B:7, C:5}

Min-VC = pointwise min = {A:10, B:7, C:5}.

Any op whose VC ≤ {A:10, B:7, C:5} is causally stable.
GC its tombstones.
```

Epidemic stable storage:

- Each replica gossips its progress.
- A bounded staleness window means metadata older than the window can be reclaimed.

The hard case: a replica disappears (crashed, partitioned for a long time). The minimum VC stalls, and metadata accumulates until the replica is declared dead. Most production systems have a manual or timeout-based "evict member" operation.

## Metadata Bloat

The dominant scaling problem for CRDTs.

Sources of metadata:

- OR-Set: every add carries a unique tag. After 1M adds, the tag set is large.
- RGA text: every character (including deleted ones) is permanent until causally-stable GC.
- Vector clocks: O(replicas) per op.
- Tombstones: every removed key/element/character.

Mitigations:

- Causal-stable GC (above).
- Compact encodings: Yjs uses varint, run-length, delta encoding.
- Columnar layouts: Automerge 2.x stores ops in columns by field for compression.
- Tag pooling: instead of a unique tag per add, use (replica_id, sequence) so all ops from a replica share a "name" and the sequence is implicit.
- Causal Length Sets (next section).

A real Yjs document with 1M character edits weighs on the order of a few hundred KB to a few MB after GC, depending on edit pattern. Without GC, the same document can be 10-100x larger.

## Causal Length Sets

Almeida 2018: an alternative to OR-Set with smaller metadata.

Idea: instead of tracking a set of tags per element, track a single "causal length" — the number of times this element has been added in the causal history. An add increments the length; a remove sets a "remove length" snapshot.

```text
type CLSet[E] = Map[E, (add_len: u64, rem_len: u64)]

contains(e) ⇔ add_len(e) > rem_len(e)
```

Adding an element increments add_len. Removing snapshots add_len into rem_len. Adding again increments add_len past rem_len, making the element present again.

Pros: O(1) metadata per element instead of O(adds).
Cons: subtle semantics for some concurrent patterns; less widely deployed than OR-Set.

## Pure-Operation CRDTs

Almeida et al's pure-op formulation (2018): separate the "prepare" phase (which derives the op from local state) from the "effect" phase (which applies the op to the state, given operations from any source).

Goal: eliminate redundant metadata by ensuring that the operation itself contains exactly what it needs to commute correctly.

```text
fn prepare(state, intent):
    # generate op based on local view
    return op

fn effect(state, op):
    # apply op; must commute with concurrent ops
    return state'
```

Pure-op CRDTs remove the need for per-op tags in many cases — the op's identity (replica, seq) plus its semantic context is enough.

## δ-state CRDTs

Already covered in detail above. Key reminders:

- Underlying state is still a semilattice; deltas are members of the same (or a related) lattice.
- Local update produces a delta; the new state is the join of the old state and the delta.
- Sync sends recent deltas, not the whole state.
- Smaller bandwidth than CvRDT, no causal-broadcast requirement like CmRDT.
- Used in Akka Distributed Data, Riak's later versions.

## Real-World Systems

A tour of where CRDTs live in production.

- Yjs (yjs.dev): browser-first, JS/TS, RGA-based, optimized binary encoding. Used by ProseMirror, TipTap, BlockNote, the Notion-clone open-source projects, Linear, Capacities. The most-deployed CRDT in the world by document count.
- Automerge (automerge.org): JSON-shaped, cross-platform (Rust core with JS/Swift/Kotlin bindings), rich history. Used by InkAndSwitch projects, PushPin, Pixelboard, several local-first apps.
- Riak (Basho, now MIT): distributed key-value DB with built-in CRDT types: counter, set, map, register. Production at scale. Use case: multi-region key-value with conflict-free merges.
- Cassandra: not a CRDT in the strict sense — uses LWW per cell with tombstones. Approximates CRDT semantics for simple workloads but does not handle concurrent inserts the way real CRDTs do.
- SoundCloud Roshi: an LWW-Element-Set on top of Redis/Memcache for time-series-style data. Used at SoundCloud for stream timelines.
- Redis Enterprise CRDTs: counters, sets, strings with conflict-free merge across geo-distributed Redis instances.
- AntidoteDB: academic CRDT-native database with transactional snapshots over a CRDT keyspace. Not widely deployed but a research reference.
- Earthstar, Willow, IPFS-pinned CRDTs: P2P/local-first projects using CRDTs as the storage substrate.
- Diamond Types: Rust-based, very fast text CRDT — used in research and starting to appear in production editors.

```text
                  ┌─────────────────────────────────────────┐
                  │ Pick by problem:                        │
                  │                                         │
                  │  collaborative text     → Yjs           │
                  │  collaborative JSON     → Automerge     │
                  │  geo-distributed KV     → Riak          │
                  │  geo-distributed Redis  → Redis Enterp. │
                  │  research / academic    → AntidoteDB    │
                  │  blazing-fast text      → Diamond Types │
                  └─────────────────────────────────────────┘
```

## Network Sync Patterns

How replicas actually exchange CRDT updates over the wire.

### Yjs over WebSocket (y-websocket-server)

A central WebSocket relay server. Clients connect, broadcast their local update messages to the relay, and receive others' updates.

```text
client A ──┐
client B ──┼──▶ y-websocket-server ──▶ broadcasts to all clients
client C ──┘     (no merge logic in the server)
```

The server is dumb — it just relays binary update blobs. The merge happens in each client. The server can also persist updates to a backing store (LevelDB, Postgres, S3) for late-joiners and history.

### Automerge sync protocol

A symmetric peer-to-peer sync using Bloom-filter-like "have" digests.

```text
A → B: "I have ops with these heads: [hash1, hash2]"
       (a have-digest summarizing my known ops)
B    : compute the diff — which ops do I have that A doesn't?
B → A: "Here are the ops you're missing: [op1, op2, op3]"
```

The protocol minimizes wasted bandwidth: only the missing ops are transmitted, identified via causal-history hashes.

### libp2p pubsub

CRDT updates broadcast over a P2P pubsub mesh (e.g., GossipSub). Useful for fully decentralized apps where there is no server at all.

### WebRTC peer-to-peer

Truly offline-first apps can use WebRTC for direct peer-to-peer sync. Requires a signalling server to discover peers; once connected, the data flows P2P with no server involvement.

```text
Peer discovery (signalling server) ───▶ WebRTC handshake
                                            │
                                            ▼
                                    direct peer-to-peer
                                    binary CRDT updates
```

## Offline-First Apps

The "local-first software" philosophy (Kleppmann et al, 2019): software where the user's data lives on their device and works without a network, with optional sync to other devices and the cloud.

CRDTs are the technical backbone of local-first software. The pattern:

```text
+--------------------------------------------+
| user action                                 |
|     │                                       |
|     ▼                                       |
| local CRDT update (synchronous, no spinner)│
|     │                                       |
|     ├──▶ persist to local storage           |
|     │       (IndexedDB, SQLite, etc.)       |
|     │                                       |
|     └──▶ enqueue for sync                   |
|             (best-effort background)         |
+--------------------------------------------+
                        │
                        ▼
                if online: sync to peers/server
                if offline: keep working; sync later
```

Key UX outcomes:

- No spinner. Every user action is local and instant.
- Offline is not an error — it's a normal state.
- Multi-device works: the same data flows to phone, laptop, tablet via sync.
- Conflicts resolve automatically; in the rare case of an unresolvable semantic conflict, the app can show "you both edited X — here are both versions" instead of "one of you lost."

## Conflict Resolution Patterns

When updates collide, what wins?

- Concurrent add/remove of the same set element → add-wins (OR-Set / AW-Set) or remove-wins (RW-Set), per design choice. Add-wins is the default in most libraries because it matches human intuition ("I added it; it should appear").
- Concurrent register update → LWW (last-writer-by-timestamp) or MV (preserve all values, application resolves). LWW is simpler; MV is safer for high-stakes values.
- Concurrent inserts at the same text position → tie-break by replica id (deterministic but order may surprise users; in practice, network sync typically prevents true exact-position concurrency).
- Concurrent map-key set with different types (string vs object) → LWW for the slot; the loser is silently overwritten. Strongly-typed CRDTs (vs Automerge's implicit schema) reject this at write time.
- Concurrent delete of a parent and modify of a child → Automerge: the modify is preserved in history; the parent reappears if the child has surviving content (configurable). Yjs: structural deletes win over content writes.

```text
                  ┌─ AW-Set / OR-Set     (add-wins)
                  │
  Set conflicts ──┤
                  │
                  └─ RW-Set              (remove-wins)


                  ┌─ LWW-Register        (timestamp wins)
                  │
  Register ───────┤
                  │
                  └─ MV-Register         (keep all; app resolves)
```

## Performance Characteristics

```text
                  │ Yjs                │ Automerge       │ Riak CRDTs
  ─────────────── │ ─────────────────  │ ───────────────  │ ─────────────
  Wire format     │ binary (varint,    │ binary (col-     │ erlang/protobuf
                  │ run-length, delta) │ umnar in 2.x)    │
  Insert latency  │ ~µs                │ ~10-100µs        │ ms-scale
                  │                    │                  │ (network)
  Memory/op       │ very compact       │ compact, history │ compact
                  │ (history GC'd)     │ retained         │
  Op-based?       │ yes (with state    │ yes              │ state-based
                  │ as fallback)       │                  │
  Bandwidth       │ very low           │ low              │ medium
  Causal ordering │ enforced via       │ enforced via     │ vector-clocked
                  │ hash-DAG           │ hash-DAG         │
  GC              │ document-level     │ history retained │ tombstone GC
                  │ (collapsing)       │ by default       │
```

A useful rule of thumb: for collaborative text, Yjs is the fastest production option. For rich JSON data with branch-and-merge semantics, Automerge. For multi-master KV with built-in CRDT primitives, Riak. For geo-distributed Redis, Redis Enterprise CRDTs.

## Limitations

CRDTs are a powerful tool but they have hard limits.

- Cannot enforce uniqueness: "every username is unique" is not expressible. Two replicas can concurrently claim the same username; merge will produce a state with both, and the app must resolve.
- Cannot enforce arbitrary cross-document invariants: "the sum of balances equals zero" cannot be maintained without coordination.
- Cannot enforce strict ordering: "operation X must happen before Y" is not enforceable across replicas (replicas can issue operations in any order locally).
- Cannot enforce schema evolution mid-flight: if a replica still uses an old schema and another replica has migrated, merging produces undefined behavior unless the schema migration is itself CRDT-encoded (which is hard).
- Convergent ≠ correct: CRDTs guarantee replicas converge, not that the converged state is what the user wanted. A flaw in CRDT design can converge to "the wrong answer" perfectly consistently.

If your problem requires hard invariants, you need consensus (Paxos/Raft) or a coordination layer above CRDTs (escrow, reservations, bounded counters).

## When NOT to Use CRDTs

- Strict invariants like bank account ≥ 0, inventory ≤ 100 — need consensus or escrow. Bounded counter is the closest CRDT but still requires coordination at the boundary.
- Single-source-of-truth (publishing decisions, election outcomes, config rollouts) — use a CP store (etcd, ZooKeeper, Postgres with strong isolation).
- Strict schema enforcement at write time — CRDTs lose schema enforcement when types diverge across replicas. Use a schema-validated database with CP semantics.
- Strong total ordering required (event sourcing where order matters end-to-end) — use Kafka or similar log with single-writer-per-partition.
- High contention on a small key — CRDTs work best with low-contention or commutative operations. A single key with thousands of concurrent writers and a strict invariant will not be well-served by a CRDT.

```text
   ┌────────────────────────────────────────────────┐
   │ Decision tree:                                  │
   │                                                 │
   │ "Need writes during partitions?"                │
   │   no  → use Paxos/Raft (etcd, ZooKeeper)        │
   │   yes → next                                    │
   │                                                 │
   │ "Need to enforce invariants like balance ≥ 0?"  │
   │   yes → escrow / bounded-counter / consensus    │
   │   no  → next                                    │
   │                                                 │
   │ "Need rich history (undo, branch, merge)?"      │
   │   yes → Automerge                               │
   │   no  → next                                    │
   │                                                 │
   │ "Collaborative text in a browser?"              │
   │   yes → Yjs                                     │
   │   no  → Riak / Redis CRDTs / custom CRDT        │
   └────────────────────────────────────────────────┘
```

## CRDT vs Operational Transform

Operational Transform (OT) was the original collaborative-editing algorithm — used by Google Docs, Etherpad, and many early collaborative editors.

OT idea:

- Each operation (insert, delete) is sent to a central server.
- The server applies operations in arrival order and "transforms" each operation against the operations that arrived before it but were created concurrently.
- The transform function rewrites positions: "you wanted to insert at position 5, but two characters were inserted before position 5 in the meantime — now insert at position 7."

```text
A: insert(5, "x")  →  server  ←  B: insert(3, "y")
                         │
                         ▼
                   transform A's op vs B's op:
                   "y" went in at 3, so A's "x" is now at 6, not 5
                         │
                         ▼
              broadcast transformed ops to all clients
```

OT works but has notorious correctness pitfalls: every transform function (insert vs insert, insert vs delete, delete vs delete, ...) must satisfy the "transformation property" — and many published OT algorithms had bugs that took years to find.

CRDT idea:

- No transformation. Each operation carries its own context (origin, tag, timestamp).
- Convergence comes from algebraic properties (commutativity, idempotency) of the merge function.
- No central server required — peer-to-peer just works.

CRDTs eliminate the central transformation server but at the cost of more complex per-op metadata and tombstones. Yjs and Automerge demonstrate that CRDTs can be production-grade fast.

Modern Google Docs is reported to use a CRDT-flavored algorithm internally (rather than the original OT). The industry has converged on CRDTs for new collaborative-editing systems.

## Common Errors

Why CRDT convergence fails in practice:

- Non-commutative custom operations: a custom op that depends on order. The fix is to redesign the op to be commutative or to enforce causal delivery at the messaging layer.
- Forgetting unique tags in OR-Set: if two adds of the same element produce the same tag (e.g., a hash of just the value), a remove tombstones both — re-add stops working. Fix: tag = (replica_id, monotonic_counter).
- Wall-clock timestamps with skew: LWW with system time can have one replica's clock 10s ahead, causing all of its writes to "win" anomalously. Fix: Lamport clocks or hybrid logical clocks.
- Causal-stable GC mixed with non-causally-stable operations: GCing a tombstone before all replicas have observed the corresponding remove can resurrect deleted data. Fix: prove causal stability before GC.
- Idempotency violations: a custom merge that double-applies. Fix: ensure merge(a, a) = a as a unit test.
- Mixing op-based and state-based code paths: shipping ops on one path and state on another can produce duplicate effects. Fix: pick one and stick with it; if hybrid, document the boundary.
- Letting the merge function depend on external mutable state: now merge is not deterministic. Fix: merge must be a pure function of its inputs.
- Sequence CRDT inserts without causal context: a remote insert references an origin you haven't seen, and you crash or apply incorrectly. Fix: causal broadcast at the messaging layer.

## Common Gotchas

Broken→fixed pairs.

### LWW with skewed clocks

```text
broken:
  Replica A's clock: 12:00:05
  Replica B's clock: 12:00:01     (4s behind)
  A.write("x", ts=12:00:05.000)
  B.write("y", ts=12:00:01.500)   (issued AFTER A's write in real time)
  merge: A wins (higher ts) — but the user's most recent intent was B's "y"!

fixed:
  Use Lamport clocks: each replica counter increments per local op,
  monotonic across the system. Or hybrid logical clocks (HLC) =
  (max(physical, last_seen_physical), logical) tuple, capped to bound drift.
```

### Naive set with adds and removes

```text
broken:
  type Set = (added: GSet, removed: GSet)   # this is a 2P-Set!
  add("x"); remove("x"); add("x")
  result: x is removed (because tombstone is permanent)

fixed:
  Use OR-Set (Observed-Remove). Each add gets a unique tag.
  Re-adds produce new tags and survive earlier removes.
```

### Counter without per-replica tracking

```text
broken:
  type Counter = u64
  inc(): self.value += 1
  merge(a, b): max(a, b)
  Replica A inc once, Replica B inc once: each at value=1.
  merge: value=1.   But total inc count was 2!

fixed:
  G-Counter with per-replica vector. merge = pointwise max.
  value = sum of vector. Now: {A:1, B:1} → value 2.
```

### Tombstone-free remove

```text
broken:
  type Set = GSet
  remove(e): self.elements.remove(e)
  Replica A: add(x).
  Replica B: doesn't see add yet.
  Replica A: remove(x), so A.elements = {}.
  merge: B.elements ⊔ A.elements = {x}    !  resurrected.

fixed:
  Tombstone removed elements (2P-Set or OR-Set with tag tombstones).
  The remove signal must persist in state to dominate concurrent adds.
```

### Non-idempotent merge

```text
broken:
  fn merge(a, b): return concat(a, b)
  Replica A: state = "X"
  Replica A receives B's state "Y" twice (network duplicate).
  After first merge: "XY"
  After second merge: "XYY"     ! duplicate.

fixed:
  Choose a merge that satisfies merge(a, a) = a.
  For lists, use a sequence CRDT with unique ids, not raw concat.
```

### Forgetting causal context

```text
broken:
  Op-based CRDT. Replica B issues remove(x) but has not seen
  Replica A's add(x) yet. Without causal context tracking, B's
  remove encodes "remove x" with no detail about which adds it observed.
  When A's add(x) finally reaches B, x reappears; B has no record
  that the previous remove targeted anything specific.

fixed:
  Carry a causal-context (vector clock or version vector) with the
  remove. A remove tombstones precisely the add-tags it observed.
  Concurrent adds (with later ids) survive.
```

### Implicit-schema CRDT corruption

```text
broken:
  Automerge document. App v1 stores `tags` as a list of strings.
  App v2 stores `tags` as a list of objects {name, color}.
  A v1 client and a v2 client edit concurrently. After merge,
  some entries are strings and some are objects. The app crashes
  on render.

fixed:
  Either: (a) version the schema and migrate on read; (b) use a
  CRDT library with explicit schema (e.g., a Yjs map with typed
  entries); (c) embed schema-version metadata and refuse to merge
  across incompatible versions.
```

### Sync protocol that doesn't preserve causal order

```text
broken:
  Op-based CRDT shipped over a non-causal pubsub topic.
  Operations arrive out of order. RGA inserts reference origins
  that haven't been delivered yet. Replica drops them or applies
  incorrectly.

fixed:
  Use a transport that preserves causal order (Erlang/OTP gen_server,
  Akka cluster), or implement vector-clock-based reorder buffering
  on top of the unreliable transport.
```

### Bonus: clock skew on a hybrid logical clock

```text
broken:
  HLC initialized with system time but never updated on receive.
  Receiving op with HLC > local HLC: ignore — local HLC stays low.
  Drift accumulates.

fixed:
  On every receive: HLC = max(local_HLC, received_HLC + 1).
  Bounds drift to network message latency.
```

### Bonus: merging state across schema-incompatible CRDT types

```text
broken:
  Replica A: counter type for key "k".
  Replica B: set type for key "k".
  merge(...): undefined.

fixed:
  Type tags in CRDT entries; merge refuses or chooses a winner
  by deterministic rule (e.g., LWW on type-change op).
```

## Implementation Notes

### Yjs binary encoding

Yjs ships updates in a compact binary format:

```text
- Varint integers everywhere (1-byte for small values).
- Run-length encoding of consecutive items from the same client.
- Reference compression: use (clientID, clock-delta) instead of
  full (clientID, clock) pairs.
- Delete-set as compressed ranges: (clientID, clock-start, length).
```

A typical document update is bytes-to-tens-of-bytes per logical op.

### Automerge columnar format

Automerge 2.x stores ops in a columnar layout:

```text
ops table:
  actor:     [A, A, B, A, B, ...]
  seq:       [1, 2, 1, 3, 2, ...]
  action:    [SET, INS, INS, DEL, INS, ...]
  obj:       [obj1, list1, list1, list1, list1, ...]
  key:       ["title", 0, 1, 0, 2, ...]
  value:     ["Notes", "h", "i", _, "!", ...]
```

Column-oriented compresses well — repeated actors, repeated actions, monotonic sequences all delta-encode tightly.

### Roshi LWW with bucket rotation

Roshi (SoundCloud) uses LWW-Element-Set with periodic bucket rotation: old buckets become read-only and are periodically GC'd. This bounds metadata growth at the cost of losing operations older than the retention window.

### Causal-stability detection

Production systems implement causal stability via:

- Periodic anti-entropy: replicas exchange VCs and compute a global minimum.
- Heartbeats: replicas claim "I'm alive at VC X."
- Timeout-based eviction: a replica that hasn't reported in N seconds is presumed dead, and the global minimum advances without it.

The tradeoff: aggressive eviction risks data loss if the missing replica returns; conservative eviction grows metadata indefinitely.

## Idioms

- "Use Yjs for collaborative text, period." — it's the most-deployed, most-optimized, most-tested CRDT in the world; don't roll your own.
- "Use Automerge for JSON sync with rich history." — when you need branch-and-merge of full document trees with time-travel.
- "Use Riak's CRDT types for distributed key-value when LWW isn't enough." — multi-master KV with mergeable counters and sets.
- "If you need invariants, you don't need CRDTs." — invariants imply coordination; CRDTs avoid coordination by construction. Use Paxos/Raft for the invariant part, CRDTs for the everything-else part.
- "OR-Set is almost always the set you want." — re-adds work, removes are observed, semantics match human intuition.
- "PN-Counter, not (G-Counter, dec)." — a single grow-only counter cannot decrement; always use the (P, N) split.
- "Lamport clocks, not wall-clock." — for any LWW you ship to production, use logical or hybrid clocks.
- "Causal broadcast or state-based, not naive op shipping." — op-based CRDTs require causal delivery; if your transport doesn't provide it, use state-based or delta-state.
- "Separate the CRDT from the transport." — keep CRDT semantics pure; layer messaging (WebSocket, libp2p, Kafka) underneath. Easier to reason about and test.

## See Also

- distributed-consensus
- cap-theorem
- database-theory

## References

- Shapiro, Preguiça, Baquero, Zawirski. "A Comprehensive Study of Convergent and Commutative Replicated Data Types." INRIA Research Report RR-7506, 2011. https://hal.inria.fr/inria-00555588/
- Shapiro, Preguiça, Baquero, Zawirski. "Conflict-free Replicated Data Types." SSS 2011 (Stabilization, Safety, and Security of Distributed Systems).
- Almeida, Shoker, Baquero. "Delta State Replicated Data Types." Journal of Parallel and Distributed Computing, 2018.
- Almeida. "Approaches to Conflict-free Replicated Data Types." 2023 survey.
- Roh, Jeon, Kim, Lee. "Replicated Abstract Data Types: Building Blocks for Collaborative Applications." JPDC 2011 — the RGA paper.
- Weiss, Urso, Molli. "Logoot: a Scalable Optimistic Replication Algorithm for Collaborative Editing on P2P Networks." ICDCS 2009.
- Preguiça, Marquès, Shapiro, Letia. "A Commutative Replicated Data Type for Cooperative Editing." ICDCS 2009 — Treedoc.
- Nédelec, Molli, Mostefaoui, Desmontils. "LSEQ: an Adaptive Structure for Sequences in Distributed Collaborative Editing." DocEng 2013.
- Oster, Urso, Molli, Imine. "Data Consistency for P2P Collaborative Editing." CSCW 2006 — WOOT.
- Kleppmann, Beresford. "A Conflict-Free Replicated JSON Datatype." IEEE TPDS 2017 — the Automerge paper.
- Kleppmann, Wiggins, van Hardenberg, McGranaghan. "Local-First Software: You Own Your Data, in spite of the Cloud." Onward! 2019.
- Yjs documentation and source: https://github.com/yjs/yjs and https://yjs.dev
- Automerge documentation and source: https://github.com/automerge/automerge and https://automerge.org
- Diamond Types: https://github.com/josephg/diamond-types
- Riak CRDT documentation: https://docs.riak.com/riak/kv/latest/developing/data-types/
- Redis Enterprise CRDTs: https://redis.com/redis-enterprise/technology/active-active-geo-distribution/
- AntidoteDB: https://www.antidotedb.eu
- Jepsen consistency models reference: https://jepsen.io/consistency
- Almeida. "Causal Length Sets." 2018 — alternative to OR-Set with bounded metadata.
- Baquero, Almeida, Cunha, Ferreira. "Composition in State-based Replicated Data Types." Bulletin of the EATCS, 2017.
- Lamport. "Time, Clocks, and the Ordering of Events in a Distributed System." CACM 1978 — the foundational vector-clock paper.
- Kulkarni, Demirbas, Madappa, Avva, Leone. "Logical Physical Clocks." OPODIS 2014 — hybrid logical clocks (HLC).
