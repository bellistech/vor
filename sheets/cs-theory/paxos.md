# Paxos

Distributed consensus protocol — N processes agree on a single value (or sequence of values) under partial synchrony, surviving up to f failures with 2f+1 acceptors.

## Setup

Paxos was first described by Leslie Lamport in *The Part-Time Parliament* (submitted 1989, published TOCS 1998). The original paper used an extended metaphor about parliamentarians on the Greek island of Paxos who wished to maintain consistent legislative records despite priests wandering in and out of the chamber. Reviewers found the metaphor obscure and the paper sat unpublished for nearly a decade. After repeated reader confusion Lamport published *Paxos Made Simple* (2001), a short note that strips out the metaphor and presents the protocol as a sequence of phases. The 2001 paper is canonical; *The Part-Time Parliament* is historical.

Why Paxos exists:

- **Consensus** — In a distributed system every replica must agree on the same value (or the same ordered sequence of values). Without consensus, replicas diverge and the system loses linearizability.
- **Partial synchrony** — Real networks deliver messages with unbounded but usually finite delay. Pure asynchrony (no timing assumptions) hits FLP impossibility (Fischer–Lynch–Paterson 1985); pure synchrony (bounded delay) is unrealistic. Paxos lives in the middle: safety holds always, liveness holds when the network is "well-behaved enough."
- **Failure-stop (fail-stop) processes** — Nodes either operate correctly or crash and stop. Paxos does *not* tolerate Byzantine (arbitrary/malicious) failures; for that you need PBFT, HotStuff, or similar.
- **The hard part is safety under asynchrony** — Liveness can be argued informally and tuned by retry/election timeouts; safety must hold in the presence of arbitrary message loss, duplication, reordering, and arbitrary delays.

The canonical contribution: a protocol that *never* lets two different values be chosen, even if the network is hostile, and that *eventually* makes progress when the network calms down and a single proposer leads.

## Problem Statement

Distributed consensus, formal definition:

- A set of N processes can each propose a value.
- The processes communicate via an asynchronous, unreliable, but non-Byzantine network.
- Up to f processes may crash (fail-stop).
- The protocol must satisfy:
  - **Validity** — Any decided value must have been proposed by some process.
  - **Agreement (Safety)** — No two correct processes decide on different values.
  - **Termination (Liveness)** — Every correct process eventually decides.

```text
Validity:    decided value ∈ proposed values
Agreement:   ∀ correct p, q: decided(p) = decided(q)  (if both decide)
Termination: ∀ correct p: eventually decides
```

FLP impossibility (Fischer, Lynch, Paterson 1985): in a pure asynchronous model with even one possible crash, no deterministic protocol can guarantee both Agreement and Termination. The proof uses bivalence: an adversarial scheduler can always force the protocol into a state where it could still decide either of two values, and then keep delaying messages indefinitely.

Paxos's response: prioritize safety, accept that liveness requires *eventual synchrony* — there must be some period during which messages are delivered in bounded time and a single proposer is allowed to make progress. Outside such windows, Paxos is allowed to stall (but never violate safety).

The single-decision variant (Basic Paxos) decides on one value. The sequence variant (Multi-Paxos) decides on a sequence of values, one per "slot" or "instance," and is the foundation of state-machine replication.

## Roles

Paxos splits responsibilities into three logical roles:

- **Proposer** — Initiates consensus by proposing a value. Drives the two-phase protocol. Multiple concurrent proposers are allowed but can cause livelock.
- **Acceptor** — Votes on proposals. Persistently records the highest proposal number it has *promised* to consider and the highest *accepted* (n, v) pair. Forms the durable layer of the protocol.
- **Learner** — Learns the chosen value once it has been decided. May be notified by acceptors, or may snoop on the Phase 2b traffic. Often just an application thread reading from the replicated log.

Important deployment notes:

- A single physical node typically plays *all three* roles. Don't picture three separate machines per role — picture one process per replica that internally exposes proposer, acceptor, and learner threads.
- The number of *acceptors* determines fault tolerance. With 2f+1 acceptors you tolerate f failures (because a majority of (2f+1) is f+1, and any two majorities intersect in at least 1 acceptor — see [Why Majorities](#why-majorities)).
- Common configurations: 3 acceptors tolerate 1 failure; 5 tolerate 2; 7 tolerate 3. Production systems rarely exceed 5 voting members because each additional voter adds a Phase 2 fsync to the critical path.
- Proposers and learners can scale horizontally without affecting the safety analysis.

```text
       ┌──────────┐
       │ Proposer │   (drives protocol)
       └────┬─────┘
            │ Prepare(n) / Accept(n,v)
            ▼
       ┌──────────┐
       │ Acceptor │   (votes, persists state)
       └────┬─────┘
            │ Promise / Accepted
            ▼
       ┌──────────┐
       │ Learner  │   (observes chosen value)
       └──────────┘
```

## Basic Paxos — Single Decision

Basic Paxos decides on a single value. The protocol has two phases, each with two sub-steps (a = request, b = response).

### Phase 1 — Prepare / Promise

- **Phase 1a (Prepare):** Proposer picks a proposal number `n` (globally unique, monotonically increasing — see [Proposal Number Space](#proposal-number-space)) and sends `Prepare(n)` to a majority of acceptors.
- **Phase 1b (Promise):** When acceptor A receives `Prepare(n)`:
  - If A has already promised some `n' >= n`: ignore (or send `Nack(n')` as an optimization).
  - Otherwise: persist `n` as A's new highest-promised number, then reply `Promise(n, last_accepted_n, last_accepted_value)`. The last-accepted fields are `null` if A has never accepted anything.

### Phase 2 — Accept / Accepted

- **Phase 2a (Accept):** Proposer collects Promise responses from a majority of acceptors. It picks the value to propose:
  - If any Promise carried a non-null `last_accepted_value`, take the value with the highest `last_accepted_n` among them.
  - Otherwise, the proposer is free to pick its own value.
- It then sends `Accept(n, v)` to a majority of acceptors.
- **Phase 2b (Accepted):** When acceptor A receives `Accept(n, v)`:
  - If A has promised some `n' > n`: ignore (or send Nack).
  - Otherwise: persist `(n, v)` as the latest accepted pair, then reply `Accepted(n, v)`. (A's promised number is also at least `n`.)

### Decision

When a majority of acceptors have accepted `(n, v)`, the value `v` is *chosen*. The proposer (or any acceptor that observes the majority) notifies learners.

```text
Proposer            Acceptor A   Acceptor B   Acceptor C
   │                    │            │             │
   │── Prepare(n) ─────▶│            │             │
   │── Prepare(n) ──────────────────▶│             │
   │── Prepare(n) ───────────────────────────────▶ │
   │                    │            │             │
   │◀─── Promise(n,—,—) │            │             │
   │◀────────────────── Promise(n,—,—)             │
   │◀───────────────────────────── Promise(n,—,—)  │
   │                    │            │             │
   │── Accept(n, v) ───▶│            │             │
   │── Accept(n, v) ────────────────▶│             │
   │── Accept(n, v) ─────────────────────────────▶ │
   │                    │            │             │
   │◀── Accepted(n,v) ──│            │             │
   │◀────────────────── Accepted(n,v)              │
   │◀───────────────────────────── Accepted(n,v)   │
   │
   │ (majority accepted ⇒ v is chosen)
```

Pseudocode for the proposer:

```text
function propose(my_value):
    n = next_proposal_number()
    send Prepare(n) to all acceptors
    wait for Promise responses from a majority Q1
    if Q1 timeout: backoff and retry with higher n
    
    # pick value
    promises_with_value = [p for p in Q1 if p.last_accepted_v is not null]
    if promises_with_value:
        v = promises_with_value.argmax(p.last_accepted_n).last_accepted_v
    else:
        v = my_value
    
    send Accept(n, v) to all acceptors
    wait for Accepted responses from a majority Q2
    if Q2 timeout: backoff and retry
    return v   # v is chosen
```

Pseudocode for the acceptor (durable state: `promised_n`, `accepted_n`, `accepted_v`):

```text
on Prepare(n):
    if n > promised_n:
        promised_n = n
        fsync()
        reply Promise(n, accepted_n, accepted_v)
    else:
        # ignore or reply Nack(promised_n)

on Accept(n, v):
    if n >= promised_n:
        promised_n = n
        accepted_n = n
        accepted_v = v
        fsync()
        reply Accepted(n, v)
    else:
        # ignore or reply Nack(promised_n)
```

Note: every state change must be persisted (fsync) *before* sending the reply. Skipping persistence is a classic safety violation — see [Common Pitfalls](#common-pitfalls).

## Proposal Number Space

Proposal numbers `n` must satisfy:

- **Globally unique** — No two proposers ever pick the same `n`.
- **Monotonically increasing per proposer** — Each proposer's sequence is strictly increasing.
- **Comparable across proposers** — Acceptors compare numbers from different proposers.

The classic encoding is a tuple `(round_count, server_id)` ordered lexicographically:

```text
n = (round, server_id)
(r1, s1) < (r2, s2)  iff  r1 < r2  or  (r1 == r2 and s1 < s2)
```

Each proposer increments its own `round` counter, persists it, and tags every proposal with its unique `server_id`. Two proposers that pick the same round value still produce different `n` because their `server_id` breaks the tie.

Production realities:

- Persist `round` to disk *before* using it. On restart, read the persisted value, increment by some safety margin (e.g. +1), and continue.
- Don't use wall-clock time as `round` — clock skew can cause ties or non-monotonicity. If you do, append `server_id` so ties are broken.
- A 64-bit integer is plenty: `round` in upper 48 bits, `server_id` in lower 16 bits.
- An acceptor that learns of a higher `n` (via Nack) can hint the proposer to skip ahead, reducing rounds wasted.

Example with three servers `{A=1, B=2, C=3}`:

```text
A's first proposal:  (1, 1)
B's first proposal:  (1, 2)        > (1, 1)
A's second proposal: (2, 1)        > (1, 2)
C sees (2,1) and bumps to (3,3): (3, 3) > (2, 1)
```

## Why Majorities

The safety argument hinges on **quorum intersection**:

> In a system of N acceptors, any two majorities intersect in at least one acceptor.

If `|M1| > N/2` and `|M2| > N/2`, then `|M1 ∩ M2| >= 1`. This is just the pigeonhole principle.

This single acceptor — the one in the intersection — carries the safety invariant forward. Suppose proposal `n1` is accepted by majority `M1` with value `v1`. A later proposer running with proposal `n2 > n1` needs Promises from a majority `M2`. At least one acceptor `a ∈ M1 ∩ M2` will report `last_accepted = (n1, v1)` to the new proposer, forcing it to propose `v1` (since it picks the highest last-accepted value from its Promise responses).

Therefore: **once a value is chosen, every subsequent successful proposal proposes the same value.** This is invariant P2 (see [Safety Invariants](#safety-invariants)).

Fault tolerance count:

- With N = 2f+1 acceptors, any majority has size f+1.
- Even if f acceptors crash, the remaining f+1 still form a majority — progress is possible.
- If f+1 or more crash, no majority exists and the system halts (safely — never decides wrong, just doesn't decide).

Examples:

```text
N=3 (f=1): majority=2; tolerate 1 failure
N=5 (f=2): majority=3; tolerate 2 failures
N=7 (f=3): majority=4; tolerate 3 failures
```

You cannot reduce the quorum below a majority and keep classic Paxos safe. Variants like Flexible Paxos relax this by *separately* sizing the Phase 1 and Phase 2 quorums; see [Flexible Paxos](#flexible-paxos).

## Worked Example — 3 Nodes

Three acceptors {A, B, C}, two proposers P1 and P2 racing.

P1 wants to propose value X. P2 wants to propose Y. Use proposal numbers `(round, id)` with P1=1, P2=2.

Step 1. P1 picks `n=(1,1)` and sends `Prepare(1,1)` to {A, B, C}.

```text
P1 → A: Prepare(1,1)
P1 → B: Prepare(1,1)
P1 → C: Prepare(1,1)
```

Step 2. A and B respond first; both have not promised anything, so they record `promised_n = (1,1)` and reply `Promise((1,1), null, null)`. C is slow (network delay).

```text
A → P1: Promise((1,1), null, null)
B → P1: Promise((1,1), null, null)
```

Step 3. P1 has a majority {A, B}. No Promise carried a value, so P1 chooses `v = X`. It sends `Accept((1,1), X)` to {A, B, C}.

```text
P1 → A: Accept((1,1), X)
P1 → B: Accept((1,1), X)
P1 → C: Accept((1,1), X)
```

Step 4. Meanwhile P2 picks `n=(1,2)` (note: P2's round=1, but id=2, so `(1,2) > (1,1)`). P2 sends `Prepare(1,2)` to {A, B, C}.

Suppose the messages arrive in this order at acceptor A:

1. `Accept((1,1), X)` from P1 → A persists `accepted_n=(1,1), accepted_v=X`, replies `Accepted`.
2. `Prepare((1,2))` from P2 → A's `promised_n` was (1,1); now (1,2) > (1,1), so A bumps `promised_n = (1,2)` and replies `Promise((1,2), (1,1), X)`.

At acceptor B suppose `Prepare((1,2))` from P2 arrives *before* `Accept((1,1), X)` from P1:

1. `Prepare((1,2))` from P2 → B bumps `promised_n=(1,2)`, replies `Promise((1,2), null, null)`.
2. `Accept((1,1), X)` from P1 → B sees `promised_n=(1,2) > (1,1)`, *rejects*; replies Nack or ignores.

At acceptor C suppose only `Prepare((1,2))` arrives:

1. C records `promised_n=(1,2)`, replies `Promise((1,2), null, null)`.

Step 5. P2 collects Promises from {A, B, C}. A's Promise carried `(1,1), X`. By the rule "pick value from highest last_accepted in the Promise set," P2 must propose `v = X` (not its preferred Y).

```text
P2's promises: A: ((1,1), X), B: (null, null), C: (null, null)
Highest last_accepted_n among non-null: (1,1) with value X
P2 must propose v = X
```

Step 6. P2 sends `Accept((1,2), X)`. All three acceptors accept (their `promised_n=(1,2)` matches `(1,2)`). The chosen value is X.

The crucial outcome: **even though P2 wanted Y, the protocol forced it to propose X because the chain of Promises showed that X had been (or might have been) chosen.** This is quorum intersection in action: A's overlap between the original majority {A, B} (where X was accepted) and the new Promise majority {A, B, C} carries the invariant.

Counterexample worth noting: if P2 ignored the rule and proposed Y anyway, you'd get split-brain — A and B might end up with `(1,1)→X` while {B, C} (a different majority) accept Y. Two values chosen. The whole protocol depends on the proposer obeying step 5.

## Safety Invariants

Lamport's *Paxos Made Simple* presents the protocol as a derivation from the safety invariants. The key invariants:

### P1 — Acceptor Constraint

> An acceptor accepts a proposal `(n, v)` only if it has not promised a higher proposal number `n' > n`.

This is enforced by the acceptor's own state machine: persist `promised_n`, only accept if `n >= promised_n`.

### P2 — Proposer Constraint (master)

> If a proposal with value `v` is chosen, then every higher-numbered proposal that is chosen has value `v`.

P2 is the safety property we want to prove. Lamport decomposes it into progressively stronger statements (each implies the next):

### P2a

> If a proposal with value `v` is chosen, then every higher-numbered proposal *accepted by any acceptor* has value `v`.

P2a → P2 (because chosen requires accepted by a majority).

### P2b

> If a proposal with value `v` is chosen, then every higher-numbered proposal *issued by any proposer* has value `v`.

P2b → P2a (because an accepted proposal was issued by some proposer).

### P2c

> For any `(n, v)` issued, there is a set `S` of a majority of acceptors such that either:
> (a) no acceptor in `S` has accepted any proposal numbered less than `n`, or
> (b) `v` is the value of the highest-numbered proposal accepted among acceptors in `S`.

P2c → P2b (by induction on proposal number).

The protocol satisfies P2c by construction: in Phase 1 the proposer collects Promises from majority `S`; in Phase 2 it picks `v` per the rule above. Quorum intersection guarantees that any chosen value's majority intersects with `S`, so the rule "pick highest last_accepted" preserves the invariant.

### Proof Sketch

By induction on proposal number `n`:

- Base case: smallest `n` chosen. Trivially, P2 holds — nothing is "higher-numbered yet."
- Inductive step: assume P2 holds for all numbers < n. Show that any `(n, v)` issued has v = (the chosen value, if any has been chosen below n). The proposer collects Promises from majority `S`. Let `(n', v')` be the chosen proposal with smallest `n' < n`. By quorum intersection, `S` contains an acceptor `a` from the majority that chose `v'`. So `a`'s Promise carried some `(n'', v'')` with `n'' >= n'`. Either `v'' = v'` (if `n'' = n'`) or by induction `v'' = v'` (if `n'' > n'`). Therefore the highest last_accepted in `S` is `v'`, and the proposer proposes `v = v'`. □

This is the heart of Paxos: a constructive protocol whose every step preserves an inductive invariant, with quorum intersection as the linchpin.

## Liveness

Basic Paxos can **livelock**:

```text
P1 → Prepare(1)
A, B → Promise(1)
P2 → Prepare(2)
A, B → Promise(2)              # now they've promised >1
P1 → Accept(1, X)              # rejected, promised_n=2
P1 → Prepare(3)
A, B → Promise(3)              # now they've promised >2
P2 → Accept(2, Y)              # rejected, promised_n=3
P2 → Prepare(4)
A, B → Promise(4)
P1 → Accept(3, X)              # rejected
P1 → Prepare(5)
... forever
```

Two proposers keep stealing each other's Promise from the acceptors. Neither completes Phase 2.

Solutions:

- **Stable leader / leader election.** Designate one proposer at a time. Other proposers either back off entirely or forward client requests to the leader. With a single proposer, no Phase 2 message is ever rejected.
- **Failure detector + exponential backoff.** A proposer that loses Phase 2 (Nack) waits a randomized backoff before retrying. Combined with a leader election protocol, this prevents thundering herd.
- **Eventual synchrony.** During synchronous periods, the leader is uncontested and progress happens. During asynchronous periods, no progress is required (FLP allows this).

Multi-Paxos (next section) makes leader election an explicit and useful optimization, not just a livelock workaround.

## Multi-Paxos — Sequence of Decisions

For state-machine replication, you don't decide one value — you decide a sequence (a *log* or *command stream*). Multi-Paxos models this as one Paxos *instance* per slot.

```text
Slot 0: Paxos instance for command 0
Slot 1: Paxos instance for command 1
Slot 2: Paxos instance for command 2
...
```

Naive implementation: run Basic Paxos for each slot. That's 2 RTTs per command, which is dreadful.

**Leader optimization.** A stable leader skips Phase 1 for subsequent slots:

- The leader runs Phase 1 *once*, with proposal number `n`, against all slots simultaneously (or against an unbounded range of future slots). Each acceptor's Promise covers slots from the leader's chosen offset onward.
- For each subsequent client command, the leader assigns the next slot index and sends Phase 2 only: `Accept(n, slot, v)`.
- That's 1 RTT per command in steady state.

Pseudocode for the Multi-Paxos leader:

```text
on become_leader():
    n = next_proposal_number()
    send Prepare(n) to all acceptors (covers all slots from current_slot onward)
    wait for Promise from majority Q
    
    # for any slot in Q's promises that has a last_accepted, propose that value
    for each slot s in Q.promises_with_value:
        v = highest last_accepted_value for slot s in Q
        send Accept(n, s, v) to all acceptors
    
    next_slot = max(observed slots) + 1

on client_command(cmd):
    s = next_slot
    next_slot += 1
    send Accept(n, s, cmd) to all acceptors
    wait for Accepted from majority
    notify learners that slot s = cmd
```

Acceptor state grows: `promised_n` (still scalar) plus `accepted[slot] = (n, v)` per slot.

Followers redirect Phase 1 (or any) message to the leader if they think a leader exists, similar to Raft. If no leader, a follower may try to become leader.

When the leader fails:

- Followers detect via heartbeat timeout.
- A new leader is elected (separate protocol — see [Leader Election](#leader-election)).
- The new leader runs Phase 1 with a higher `n` for all slots from its current view onward. Some slots may have been partially accepted by the old leader; the new leader's Phase 1 collects those last-accepted values and replays them as Phase 2 messages with the new `n`. This guarantees no in-flight slot is lost.

A practical Multi-Paxos optimization: the leader piggybacks the next slot's Phase 2 on the previous slot's Accepted response. With pipelining, throughput is bounded by network bandwidth, not RTT.

## Leader Election

Multi-Paxos delegates leader election to a separate sub-protocol. Common approaches:

- **Bully algorithm** — Highest-ID node alive becomes leader. Simple; works on small clusters.
- **Ranked priority + heartbeat timeout** — Each node has a priority. On timeout, the highest-priority alive node proposes itself.
- **Gossip / failure detector** — Nodes gossip about which leaders they've seen. Once a majority agrees, the new leader is recognized.
- **ZAB-style election** — As in ZooKeeper: nodes vote for the candidate with the highest committed transaction ID.
- **Raft's election** — Randomized timeout, `RequestVote` RPC, "win iff a majority of votes." Multi-Paxos can use Raft-style election as a black box.

Crucial property: the elected leader must use a proposal number *higher than any previously used*. Otherwise its Phase 1 will be rejected and another election will trigger.

A subtle correctness point: leader election in Paxos is an *optimization for liveness*, not a safety mechanism. Two simultaneous leaders are safe — the protocol's Phase 1/Phase 2 logic handles it. They just won't make progress. The "leader" abstraction is purely about reducing message count.

```text
# Sketch of a heartbeat-based election
loop:
    if I am leader:
        send Heartbeat to all
    else:
        if no Heartbeat for T seconds:
            n = next_proposal_number()  # higher than any seen
            send Prepare(n) for all slots from current_slot onward
            if majority Promise:
                I am leader; resume Phase 2 for any in-flight slots
            else:
                back off, retry
```

## Cheap Paxos

*Cheap Paxos* (Lamport & Massa 2004) reduces the number of message-handling acceptors in steady state.

Idea: out of N acceptors, only `f+1` need to be "active" at any time. The remaining `f` are *auxiliary* and only participate when one of the active ones fails.

Trade-off: in steady state, only f+1 messages per phase (vs 2f+1). When an active acceptor fails, an auxiliary takes over — but during the transition, latency increases.

Cheap Paxos is rarely deployed; the savings are modest, and the operational complexity of tracking "active" sets isn't worth it for most systems. It appears in textbooks more than in production.

## Fast Paxos

*Fast Paxos* (Lamport 2006) skips Phase 1 in fast cases, achieving 1 RTT in the absence of contention.

Mechanism:

- Clients (or any proposer) send their value directly in Phase 2, skipping Phase 1.
- Acceptors accept the first value they see for a given fast-round.
- If acceptors agree (no contention), 1 RTT and done.
- If they conflict, fall back to classic Paxos: a coordinator resolves the conflict in a slow round.

The catch: Fast Paxos requires a *larger quorum*. Specifically, fast quorums must intersect in more than one acceptor. The standard requirement is `3f+1` acceptors to tolerate `f` failures (instead of `2f+1`), with fast-quorum size `2f+1` and classic-quorum size `f+1`.

```text
Classic Paxos:  N=2f+1, quorum=f+1, intersection >= 1
Fast Paxos:     N=3f+1, fast_quorum=2f+1, classic_quorum=2f+1
                fast quorums intersect in (2f+1)+(2f+1)-(3f+1) = f+1 acceptors
                that's enough to detect and recover from conflicts
```

Used in: some research systems and a few production deployments where 1-RTT writes matter and contention is rare.

## Generalized Paxos

*Generalized Paxos* (Lamport 2005) recognizes that not every command needs to be totally ordered.

Idea: if two commands commute (e.g. two writes to different keys), they can be applied in either order without changing the state machine's behaviour. So they don't need a strict ordering — only conflicting commands do.

Acceptors accept *sequences* of commutative commands. Conflicts are detected at the proposer level; only conflicting commands trigger a slow-path round.

Throughput improves dramatically when most operations are independent (e.g. updates to different keys in a key-value store). The cost is complexity: defining "conflict" is application-specific.

Influence: EPaxos (see below) is the most widely-cited descendant of Generalized Paxos.

## Mencius

*Mencius* (Mao, Junqueira, Marzullo 2008) avoids the leader bottleneck by partitioning slots round-robin among proposers.

```text
Slot 0: owned by proposer 0
Slot 1: owned by proposer 1
Slot 2: owned by proposer 2
Slot 3: owned by proposer 0
Slot 4: owned by proposer 1
...
```

Each proposer can decide its own slots without coordinating with others (no leader contention). Throughput scales with the number of proposers.

Failure handling: if proposer P fails, its slots are filled with no-ops by other proposers (via a Paxos round). This keeps the log dense (no permanent gaps) but adds a small recovery cost.

Trade-offs:

- All proposers must operate at roughly the same rate; the slowest determines latency for downstream consumers.
- When one proposer is silent, others must propose no-ops on its behalf, increasing per-slot cost.
- Geographically distributed deployments often pick Mencius because every region has its own proposer, eliminating cross-region leader latency.

## Vertical Paxos / Reconfigurable Paxos

Membership changes (adding/removing acceptors) are subtle: while transitioning from membership `M_old` to `M_new`, two majorities (one from each) might choose different values.

*Vertical Paxos* (Lamport, Malkhi, Zhou 2009) and similar "reconfigurable Paxos" approaches handle this:

- Treat the configuration itself as a value chosen by Paxos: "the new configuration is C_new."
- Once C_new is chosen, all subsequent slots use C_new's quorums.
- During the transition, slots in flight under C_old must complete before C_new takes over (or use a joint-consensus rule where both old and new majorities must accept).

Practical pattern (used by Raft as well):

1. Leader proposes `Reconfigure(C_new)` as a regular log entry.
2. Once committed, the leader switches to using C_new's quorums for subsequent entries.
3. Removed members may safely shut down once they've seen the entry committed.

Vertical Paxos terminology: "vertical" refers to *layering* configurations — at any point in time, exactly one configuration is "active," and configuration changes are themselves Paxos decisions in a meta-instance.

## Flexible Paxos

*Flexible Paxos* (Howard, Malkhi, Spiegelman 2016) is a major theoretical relaxation.

Observation: classic Paxos requires that *both* Phase 1 quorums (Q1) and Phase 2 quorums (Q2) be majorities. The only thing safety actually requires is that **every Q1 intersects every Q2**:

```text
Classic Paxos:     |Q1| > N/2  AND  |Q2| > N/2
Flexible Paxos:    Q1 ∩ Q2 ≠ ∅  for all Q1, Q2
                   (no requirement that either be a majority)
```

This unlocks asymmetric quorums. For example, with 6 acceptors:

```text
Classic: |Q1|=4, |Q2|=4 (both majorities)
Flexible: |Q1|=2, |Q2|=5  → still intersect (because 2+5 > 6)
          or |Q1|=5, |Q2|=2
```

Why this matters:

- If Phase 1 (leader election) is rare and Phase 2 (steady-state writes) is common, you can size Q2 small for fast writes and Q1 large to compensate.
- Geo-distributed deployments can place Q2's small set in one DC for low write latency; Q1 must span DCs but only runs at leader change.

Flexible Paxos is the theoretical foundation for variants like *Paxos Quorum Leases* and influences modern systems that customize quorum shapes.

## EPaxos (Egalitarian Paxos)

*EPaxos* (Moraru, Andersen, Kaminsky 2013) eliminates the leader entirely.

Idea: every node can propose any command. Commands are only ordered when they *conflict* (e.g. two writes to the same key). Non-conflicting commands can be decided in 1 RTT.

Mechanism (sketch):

- Each command is proposed with its set of *dependencies* (commands it conflicts with that were observed at the proposer).
- Acceptors record commands and their dependencies, forming a partial-order DAG.
- A *fast path* (1 RTT) succeeds when a fast quorum agrees on the dependency set.
- A *slow path* (2 RTT, classic Paxos-style) handles disagreements.

Benefits:

- 1 RTT in the common case (no conflict).
- No leader bottleneck.
- Geo-friendly: each region's proposer handles local writes locally.

Costs:

- Significantly more complex than Multi-Paxos.
- Garbage collection of the dependency DAG is non-trivial.
- Conflict detection is application-specific (must define what "conflicts" means).

Used in: research prototypes, Tencent's PaxosStore variants, and some specialized geo-replicated systems.

## CASPaxos

*CASPaxos* (Rystsov 2018) is a minimalist single-decree variant: consensus on a *register* (one variable) via a `change(f)` operation, where `f` is a function `old_value -> new_value`.

API:

```text
change(register, f) -> new_value
    # atomic compare-and-swap-like update
```

Use case: state in a single register (e.g. a configuration value, a lease holder, a sequencer). Simpler than Multi-Paxos because there's no log — just a register that evolves.

CASPaxos is essentially Basic Paxos with a different mental model: instead of "decide value v," it's "atomically update register from old to f(old)." Each `change` is a fresh Paxos instance over the register's current state.

Used in: simple coordination primitives, lease management, leader election protocols themselves.

## Paxos vs Raft

Raft (Ongaro, Ousterhout 2014) was designed explicitly as "Paxos for understandability." It enforces stricter invariants that simplify the protocol:

| Aspect | Paxos | Raft |
|---|---|---|
| Leader role | Optional optimization | Required at all times |
| Log invariant | Slots can be filled out-of-order | Strict log-prefix: a follower's log is always a prefix of the leader's log up to commit index |
| Election | Separate sub-protocol | Built-in (`RequestVote` RPC, randomized timeouts) |
| Log replication | Per-slot Paxos instance | `AppendEntries` RPC with prev-log-index / prev-log-term match check |
| Membership change | Reconfigurable / Vertical Paxos | Joint consensus (C_old,new) or single-server change |
| Restrictions | Any node can propose | Only leader can propose |
| Complexity | Higher (more variants, fewer constraints) | Lower (more constraints, fewer variants) |
| Mathematical equivalence | Generalized framework | Specific instance of Multi-Paxos with extra invariants |

Both achieve consensus for state-machine replication. The choice is largely engineering taste:

- **Raft** is easier to implement correctly. The log-prefix invariant means recovery and replication are straightforward.
- **Paxos** is more flexible. Leader-less variants (EPaxos), commutative optimizations (Generalized Paxos), and asymmetric quorums (Flexible Paxos) are all natural extensions.

Quote from Lamport (paraphrased): "Raft is what you get when you specialize Paxos for the common case of state-machine replication."

## State Machine Replication

The canonical use of Multi-Paxos: replicate a state machine across N nodes.

```text
Client         Replica 1    Replica 2    Replica 3
   │              │ (leader) │             │
   │── cmd1 ─────▶│          │             │
   │              │── Accept(n,1,cmd1) ─────────▶
   │              │◀──────────────────── Accepted
   │              │ (committed slot 1)
   │              │ apply cmd1 to local state machine
   │              │ replicate decision to followers
   │◀── result ───│
```

Properties:

- All replicas apply the *same* sequence of commands → if commands are deterministic, all replicas reach the same state.
- A replica that falls behind catches up by reading the log (or a snapshot + tail of the log).
- Linearizability is achieved if the leader serves reads *after* committing them (or via lease-based optimistic reads).

The state machine must be deterministic. Common pitfalls:

- Random number generation (use a seed embedded in the command).
- Clock reads (use a logical clock embedded in the command, or quantize wall clock).
- Map/dict iteration order in some languages (sort before iterating).
- Floating-point operations (deterministic across compilers? usually yes for IEEE 754, but be careful).

Snapshots: when the log grows large, take a snapshot of the state machine and truncate the log up to the snapshot point. A replica catching up first installs the latest snapshot, then applies the log tail.

```text
log: [snapshot at slot 1000] [slot 1001] [slot 1002] ... [slot N]
```

## Implementation Considerations

### Durable Storage

Acceptor state that must survive crashes:

- `promised_n` — highest proposal number promised.
- `accepted_n`, `accepted_v` — last accepted (n, v) pair (per slot in Multi-Paxos).
- For Multi-Paxos, the entire log from the last snapshot.

Rules:

- **fsync at every Promise and Accept *before* sending the reply.** Skipping this is the #1 safety bug. After a crash, an acceptor must come back with the *exact* state it implied to others.
- Use a write-ahead log (WAL) for performance: append-only, sequentially written, fsync once per batch.
- Group commits: batch multiple acceptor responses into one fsync. This trades latency for throughput.

```text
on Accept(n, slot, v):
    if n >= promised_n:
        wal.append({type: "accept", n: n, slot: slot, v: v})
        wal.fsync()
        promised_n = n
        accepted[slot] = (n, v)
        send Accepted(n, slot, v)
```

### Log Truncation

The log grows unboundedly. Truncate by snapshotting:

1. Apply log up to slot K to the state machine.
2. Take a snapshot of the state machine.
3. Persist the snapshot.
4. Truncate the log: discard slots 0..K.

A crashed replica restores from the snapshot, then replays the log from K+1.

### Network

- Messages can be lost, duplicated, reordered. The protocol handles this — every message is idempotent if you use the same `n`.
- TCP gives in-order delivery within a connection but doesn't help across connections (e.g. retries after disconnect).
- Use sequence numbers within messages to discard old replies.

### Throughput vs Latency

Each Phase 2 round has a critical path: leader fsync → followers fsync → leader fsync (commit). Reducing fsyncs (via batching, NVRAM, group commits) is the primary throughput lever.

Latency is bounded by:

- Network RTT to a quorum of replicas.
- Disk fsync time on each replica.

For low-latency systems: replicate to nearby replicas; use NVMe with battery-backed write cache; batch commands aggressively.

## Common Pitfalls

### Forgetting to Persist State Before Responding

```bash
# WRONG — reply before fsync
on Accept(n, v):
    accepted_n = n; accepted_v = v
    send Accepted(n, v)
    fsync()    # too late
```

If the acceptor crashes between `send` and `fsync`, recovery loses `(n, v)`. The proposer thinks it has a majority's accept, but on restart the acceptor "forgets" — a different value can be chosen.

```bash
# RIGHT — fsync before reply
on Accept(n, v):
    accepted_n = n; accepted_v = v
    fsync()
    send Accepted(n, v)
```

### Reusing Proposal Numbers

```bash
# WRONG — round counter in memory only
proposer.round = 0
on propose(v):
    proposer.round += 1
    n = (proposer.round, server_id)
    ...
```

After a restart, `round` resets to 0. New proposals use already-seen `n` values, breaking uniqueness and safety.

```bash
# RIGHT — persist round before use
on propose(v):
    proposer.round += 1
    persist(proposer.round)  # fsync
    n = (proposer.round, server_id)
    ...
```

### Confusing Acceptor Count With Replica Count

A common confusion: "I have 3 replicas, so my majority is 2." That's only true if all 3 replicas are *acceptors*. If one is a non-voting learner (witness), your majority is still 2 of 3, but you've only got 2 voting members → the system can't survive *any* failure.

Be explicit: `acceptor_count = 2f + 1`. Learners and proposers don't count.

### Relying on Synchronous Network

```bash
# WRONG — assume timeout means failure
if no Promise within 100ms:
    treat acceptor as failed; remove from quorum
```

Network delays can exceed your timeout under load. The acceptor isn't dead, just slow. Removing it from quorum violates membership invariants.

```bash
# RIGHT — timeouts trigger retry, not membership change
if no Promise within timeout:
    backoff; retry with higher n
```

### Dueling Proposers After Partition Heal

When a network partition heals, two proposers may each think they're leader. Without a higher proposal number forcing one to back down, they livelock.

Mitigation: every leader transition must use a strictly higher `n`. The election protocol must track and persist the latest `n` seen.

## Worked Example — Multi-Paxos with 5 Nodes

Five replicas {R1, R2, R3, R4, R5}. Majority = 3. R1 is initially the leader.

### Step 1 — Leader Election

R1 picks `n=(10, 1)`, sends `Prepare((10,1))` to all (covers slots from current_slot onward).

```text
R1 → R2, R3, R4, R5: Prepare((10,1))
```

R2..R5 promise (none has promised higher), reply `Promise((10,1), {slot_data})`. Suppose all replies show no accepted slots above current_slot. R1 has majority {R1, R2, R3} (R1 votes for itself).

R1 declares itself leader. Phase 1 done; subsequent slots use proposal number `(10, 1)`.

### Step 2 — Steady State

Client sends `cmd_A` to R1. R1 assigns slot 100.

```text
R1 → R2, R3, R4, R5: Accept((10,1), 100, cmd_A)
R2, R3, R4 → R1: Accepted
R1: majority reached; commit slot 100 = cmd_A; apply to state machine
```

R5 was slow but eventually replies `Accepted` too. No retransmit needed; commit already happened with R1, R2, R3, R4.

R1 sends `Decision(slot 100, cmd_A)` to learners (or piggybacks on next message).

### Step 3 — More Slots, Pipelined

Client sends `cmd_B`, `cmd_C`. R1 assigns slots 101, 102.

```text
R1 → all: Accept((10,1), 101, cmd_B)
R1 → all: Accept((10,1), 102, cmd_C)
```

R1 doesn't wait for slot 101 to commit before sending slot 102 — pipelining. Followers receive both, fsync both, reply Accepted for both. R1 commits 101 then 102.

### Step 4 — Leader Failure

R1 crashes between sending `Accept((10,1), 103, cmd_D)` and receiving Accepted. R2 received it; R3, R4, R5 did not.

After heartbeat timeout, R3 starts election with `n = (11, 3)`.

```text
R3 → R2, R4, R5: Prepare((11, 3))
```

R2 promises and reports `accepted[103] = ((10,1), cmd_D)` (it had received and persisted that Accept).

R4, R5 promise; their `accepted[103]` is null.

R3 has majority {R3, R2, R4} (3 of 5).

### Step 5 — Recovery of In-Flight Slot

R3 sees that slot 103 has a non-null accepted value among Promise responses. Per the protocol: pick the value with the highest accepted_n. Only R2's was non-null, so R3 must propose `cmd_D` for slot 103.

```text
R3 → R2, R4, R5: Accept((11,3), 103, cmd_D)
all reply Accepted
slot 103 committed = cmd_D
```

The crashed leader's in-flight command is rescued. No information loss.

### Step 6 — Continued Operation

R3 is leader at proposal number `(11, 3)`. It assigns slots 104, 105, ... and runs Phase 2 only for each.

When R1 recovers, it sees a higher proposal number from R3's heartbeats and accepts being a follower. R1 catches up via state transfer (snapshot + log tail) and joins as a regular follower.

The crucial invariant preserved across this whole scenario: at no point are two different values committed for the same slot. Specifically, slot 103 might have looked like it could go either way (cmd_D or some new value) — but because R2 retained the accepted record and was in the new election quorum, the protocol forced cmd_D.

## Real-World Implementations

### Google Chubby

Chubby is Google's distributed lock service, the basis for many internal systems including BigTable's metadata service. It uses Multi-Paxos for replicating its small file-system-like namespace. Chubby's paper (Burrows 2006) is the first major industrial Paxos write-up.

Chubby cells are typically 5 replicas. Throughput is modest (a few thousand ops/sec) because the workload is small writes with strong consistency. Clients cache aggressively to reduce master load.

### Apache ZooKeeper

ZooKeeper uses ZAB (ZooKeeper Atomic Broadcast), a Paxos-flavored protocol with stronger ordering guarantees. ZAB is *not* literally Paxos but is in the same family — leader-based, majority quorums, log replication.

Differences from Multi-Paxos:

- ZAB uses a "primary order" rather than per-slot proposal numbers.
- ZAB has explicit phases for discovery, synchronization, broadcast.
- ZAB recovery is simpler because of stronger ordering invariants.

ZooKeeper is used by HBase, Kafka (pre-KRaft), Hadoop, Solr, and others.

### etcd

etcd uses Raft, not Paxos. But Raft is conceptually a Paxos derivative. etcd is the metadata store for Kubernetes and is one of the most widely-deployed Raft implementations in the world.

### Microsoft Service Fabric

Used by Azure for stateful services. Implements Multi-Paxos with custom optimizations (Vertical Paxos for reconfiguration, batching, etc.).

### Apache BookKeeper

A distributed log service used by Pulsar and others. Uses a Paxos-derived replication protocol within ledgers.

### OpenReplica / Various

Several open-source Paxos libraries exist for educational and embedded use: OpenReplica, libpaxos, Paxos for Go (multiple).

### MySQL Group Replication

Uses a variant called XCom, derived from Paxos. Provides synchronous multi-master replication for MySQL.

## Spanner's Paxos

Google Spanner uses Multi-Paxos for replication within a *Paxos group* (a tablet — a range of keys in a directory). Each Paxos group spans 5 replicas, typically across 3+ data centers.

Cross-group transactions use 2PC over Paxos groups:

- The 2PC coordinator is itself a Paxos group.
- Each participant is a Paxos group.
- 2PC's "prepare" and "commit" messages are themselves Paxos-replicated within each group.

The result: a 2PC transaction's prepare/commit phases each take 2 Paxos commit latencies, but every step is durable and survives any single-replica failure.

Spanner adds TrueTime — a synchronized clock with bounded uncertainty — to provide external consistency without coordinating clocks across regions. Paxos handles replication; TrueTime handles ordering.

## CockroachDB

CockroachDB shards data into ranges (~512MB each). Each range has its own Raft group (3 or 5 replicas).

Why Raft and not Paxos?

- Raft's strict log-prefix invariant simplifies range-leveraging optimizations.
- The Raft paper's clarity made it easier to onboard engineers.

Conceptually, CockroachDB's per-range Raft is interchangeable with per-range Multi-Paxos.

CockroachDB's transaction layer is more complex: distributed, MVCC-based, with a coordinator-less commit (using transaction records replicated via Raft). Cross-range transactions don't use 2PC in the classical sense — instead they use intent records and parallel commits.

## Apache Kafka

Pre-Kafka 3.3 ("Bridge Release"), Kafka relied on ZooKeeper (and thus ZAB) for cluster metadata: broker membership, topic configurations, partition leadership.

Kafka 3.3+ introduces KRaft (Kafka Raft), an embedded Raft-based metadata quorum. This eliminates the ZooKeeper dependency and simplifies operations.

Note: Kafka *partition replication* (the actual log data) does not use Paxos or Raft directly. It uses ISR (In-Sync Replicas) with a leader-elected-by-controller model. This is a separate replication mechanism, not a consensus protocol — leadership decisions are delegated to the metadata quorum (ZooKeeper or KRaft).

## Performance Characteristics

### Latency

- **Basic Paxos:** 2 RTTs (Phase 1 + Phase 2) plus disk fsync at each acceptor.
- **Multi-Paxos steady state:** 1 RTT (Phase 2 only) plus 1 fsync per replica.
- **Fast Paxos no-conflict:** 1 RTT in fast path.
- **EPaxos no-conflict:** 1 RTT.

For a 5-replica cluster with ~1ms inter-replica RTT and ~1ms fsync (NVMe):

```text
Multi-Paxos commit latency ≈ 1 ms (RTT) + 1 ms (fsync) ≈ 2 ms
```

For geo-replicated (5 regions, 50ms RTT):

```text
Multi-Paxos commit latency ≈ 50 ms (RTT) + 1 ms (fsync) ≈ 51 ms
(no useful improvement from faster disks; RTT dominates)
```

### Throughput

Throughput is bounded by:

- Leader's outbound bandwidth (every command goes to every replica).
- Disk throughput (each command requires an fsync).
- Network RTT (limits in-flight commands without pipelining).

Pipelining (sending Phase 2 of slot N+1 before slot N's Phase 2 completes) is the key technique. With sufficient pipeline depth, throughput approaches `bandwidth / message_size`.

Batching multiple client commands into one Paxos slot multiplies throughput: 1000 commands/slot × 10000 slots/sec = 10M commands/sec.

```text
Throughput ≈ batch_size × pipeline_depth × (1 / leader_fsync_time)
```

### Memory

- Every slot's accepted (n, v) is held in memory (or paged from disk).
- Truncate via snapshots to bound memory.
- Pending promises and accepts are held until quorum or timeout.

## Tuning

### Batching

Group multiple client commands into a single Paxos slot:

```text
on client_cmd(c):
    pending_batch.append(c)
    if pending_batch.size >= MAX_BATCH or batch_timer expired:
        slot = next_slot
        send Accept(n, slot, pending_batch)
        pending_batch = []
```

Trade-off: larger batches → higher throughput but higher individual latency.

### Pipelining

Allow multiple slots in flight simultaneously:

```text
loop:
    slot = next_slot
    next_slot += 1
    send Accept(n, slot, cmd) to all   # don't wait for previous slot
```

Followers fsync and reply per slot; the leader commits in slot order.

### Group Commit (fsync)

Batch fsyncs across slots:

```text
on receive Accept((n, slot, v)):
    log.append((n, slot, v))
    if pending_writes >= GROUP_SIZE or group_timer expired:
        log.fsync()  # one fsync covers many slots
        for each pending: send Accepted
```

### Persistence Tuning

- **WAL on NVMe** with battery-backed write cache approaches RAM speed (~10 µs per fsync).
- **Group commit** amortizes fsync cost across many ops (e.g. 1000 ops in one fsync = 10 ns/op fsync overhead).
- **Async durability** (relaxed durability) trades safety for speed — not recommended.

### Read Optimizations

Reads in a state-machine-replicated system:

- **Linearizable reads via Paxos:** every read is also a Paxos slot. Costs full RTT + fsync. Strongest consistency.
- **Read leases:** leader holds a lease for time T; during the lease, only the leader can commit, and reads from the leader are linearizable without consensus.
- **Stale reads:** read from any replica (potentially stale). Cheapest but only "eventual consistency."
- **Quorum reads:** read from a majority; return the most recent value seen. Consistent but expensive.

## Failure Modes

### Leader Failure

- Heartbeat timeout triggers election.
- New leader runs Phase 1 with higher `n` for all slots from current view onward.
- In-flight slots from old leader are recovered: any acceptor that accepted (under old `n`) reports it; new leader replays that value with new `n`.
- Total downtime ≈ election timeout + Phase 1 RTT.

### Network Partition

- Minority side: cannot form a majority, cannot decide. Stalls safely.
- Majority side: continues normally with elected leader.
- On heal: minority side catches up via state transfer + log replay. Old leader (on minority side) discovers higher `n` from new leader's heartbeats and steps down.

### Acceptor Crash

- Tolerated up to f failures.
- Crashed acceptor's Promise/Accepted state survives (it's persisted).
- On restart, the acceptor reads its persisted state and resumes participating.
- A long-down acceptor catches up via state transfer.

### Acceptor Returns

When an acceptor that's been down for a while comes back:

1. Read persisted `promised_n` and `accepted` log.
2. Receive Phase 1 / Phase 2 from current leader.
3. Detect that the leader's `n` is higher → bump `promised_n`, reply Promise.
4. Replay the log tail (any committed slots it missed) — typically via state transfer:
   - Receive snapshot + log tail.
   - Apply to state machine.
   - Resume normal operation.

### Both Phases Stalled (Permanent)

If more than f acceptors crash, no majority is possible. The system stalls indefinitely. Safety is preserved — no decisions are made — but liveness is lost until at least one crashed acceptor comes back.

This is acceptable: better to stall than to corrupt. Operators must intervene to restore replicas.

## Membership Changes

Adding or removing acceptors changes the quorum size, which threatens safety during the transition.

### The Naive Approach (Broken)

```bash
# WRONG — switch from C_old to C_new in one step
old config: {A, B, C}
new config: {A, B, C, D, E}
when admin says "add D and E": just start treating C_new as the config
```

Race condition: during the transition, two majorities can be formed without intersection — `{A, B}` from C_old (a majority of 3) and `{C, D, E}` from C_new (a majority of 5) — and they don't intersect. Two values can be chosen for the same slot.

### Joint Consensus (Raft)

Raft's solution: use a *joint configuration* `C_old,new` that requires majorities from *both* old and new configurations.

1. Leader proposes `Reconfigure(C_old,new)` as a log entry. Quorum: majority of C_old.
2. Once committed, all subsequent decisions require majority of C_old AND majority of C_new.
3. Leader proposes `Reconfigure(C_new)`. Quorum: majority of joint.
4. Once committed, only C_new's majority is required.

Step 2 ensures that any decision under the joint config is safe under both old and new — no split-brain possible.

### Vertical Paxos / Reconfigurable Paxos

Use Paxos itself to decide the configuration:

- `meta_paxos` decides "the active configuration is C_k."
- Each data Paxos instance is tagged with the configuration index `k` it operates under.
- When the configuration changes, in-flight instances under C_k complete first; new instances run under C_{k+1}.

### Cheap Paxos Reconfiguration

Cheap Paxos's "active vs auxiliary" distinction makes reconfiguration easier: swap an auxiliary into active set, or vice versa, without re-running consensus on the configuration itself. The config-change is implicit in the routing layer.

### Single-Server Change (Raft Optimization)

If you only add or remove one server at a time, joint consensus simplifies: any majority of C_old and any majority of C_new always overlap (because they differ by only one member). You can skip the joint phase. Most operational tools use this rule: "add one node, wait for it to catch up, then add the next."

## Comparison Table

| Protocol | Leader | Quorum | Steady-State Round | Fast Path | Used By |
|---|---|---|---|---|---|
| Basic Paxos | None | f+1 of 2f+1 | 2 RTT | — | Educational |
| Multi-Paxos | Stable | f+1 of 2f+1 | 1 RTT | — | Chubby, Spanner, Cassandra LWT |
| Raft | Required | f+1 of 2f+1 | 1 RTT | — | etcd, CockroachDB, Consul, TiKV |
| ZAB | Required | f+1 of 2f+1 | 1 RTT | — | ZooKeeper |
| VR (Viewstamped) | Stable | f+1 of 2f+1 | 1 RTT | — | Research, some MIT systems |
| Fast Paxos | None / Stable | 2f+1 of 3f+1 (fast) | 1 RTT (fast) | Yes | Research |
| EPaxos | None | f+1 (slow) / fast quorum | 1 RTT (no conflict) | Yes | Research, PaxosStore |
| Mencius | Round-robin | f+1 of 2f+1 | 1 RTT | Yes (no contention) | Research |
| Generalized Paxos | Optional | f+1 of 2f+1 | 1 RTT (commutative) | Yes | Research |
| Flexible Paxos | Stable | Q1 ∩ Q2 only | 1 RTT | Configurable | Research, custom systems |
| CASPaxos | None | f+1 of 2f+1 | 2 RTT | — | Specialized registers |

Notes:

- "Steady-state round" = phases needed once a leader is established.
- "Fast path" = optimization that skips a phase when conditions are met.
- "Used by" focuses on best-known production deployments.

## Common Errors

### "Paxos Requires a Reliable Network"

False. Paxos tolerates message loss, duplication, and reordering. It assumes a *fair-loss* network — i.e. messages eventually get through if retried — but does not assume bounded delay. The protocol's timeouts are for liveness only; safety holds under any network behaviour.

What Paxos *does* require: fail-stop processes (no Byzantine failures) and durable storage at acceptors.

### "Paxos Requires Synchronous Clocks"

False. Paxos has no clock dependency. Proposal numbers can be derived from any monotonic source (a counter, a hybrid logical clock, etc.). The leader-election layer may use timeouts (timer-based, not clock-based) but those are heuristics for liveness, not correctness.

Spanner uses synchronized clocks (TrueTime) for *external consistency*, but that's an additional property layered on top of Paxos, not a Paxos requirement.

### "Paxos Guarantees Liveness"

False. FLP impossibility: in a pure asynchronous model, no protocol can guarantee both safety and liveness with even one crash. Paxos sacrifices liveness in the worst case; safety is always maintained. Liveness is achieved only when the system is "well-behaved" (eventual synchrony — a long enough period of bounded delays and a stable leader).

### "A Single Paxos Instance Is Enough for State Machine Replication"

False. A single Paxos instance decides one value. State machine replication needs a *sequence* of decisions — one per command. Multi-Paxos runs one instance per slot.

### "Paxos Decides Multiple Values"

False (with caveat). Within a single instance, Paxos decides exactly one value. Multi-Paxos decides many values — one per instance — but each instance's logic is independent (sharing only the leader optimization).

### "Acceptors Need to Agree With Each Other"

False. Acceptors don't talk to each other. They only talk to proposers (and learners read from acceptors). All coordination flows through proposers. This is why a slow acceptor doesn't slow down others.

### "Majority Quorums Are Always Required"

False (with caveat). Classic Paxos requires majorities. Flexible Paxos relaxes this: any quorum sizes such that Q1 ∩ Q2 ≠ ∅ work. Some variants (Fast Paxos) require larger-than-majority quorums in fast paths.

## Common Gotchas

### Reusing Proposal Numbers After Restart

```bash
# WRONG — round counter only in memory
class Proposer:
    def __init__(self):
        self.round = 0
    def next_n(self):
        self.round += 1
        return (self.round, self.id)
```

After a crash and restart, `self.round = 0` again. Same proposal numbers are re-issued, breaking uniqueness.

```bash
# RIGHT — persist round before issuing
class Proposer:
    def next_n(self):
        new_round = self.persisted_round + 1
        persist_and_fsync(new_round)
        self.persisted_round = new_round
        return (new_round, self.id)
```

### Skipping Promise Persistence

```bash
# WRONG — promise without fsync
on Prepare(n):
    if n > self.promised_n:
        self.promised_n = n
        send Promise(n, ...)
        # later: write to disk
```

If the acceptor crashes after sending Promise but before persisting, it might "forget" the promise on restart and accept a lower-numbered proposal. This breaks the safety chain.

```bash
# RIGHT — fsync before reply
on Prepare(n):
    if n > self.promised_n:
        self.promised_n = n
        wal.append(("promised", n))
        wal.fsync()
        send Promise(n, ...)
```

### Using Wall-Clock Time as Proposal Number Without Uniqueness

```bash
# WRONG — wall-clock may collide
def next_n(self):
    return (time.time(), self.id)   # if two proposers query at same instant
```

Two proposers calling `time.time()` at the same nanosecond → tie, not strict ordering. Even with `(time, id)` tuple, NTP can move time backward → non-monotonic.

```bash
# RIGHT — counter-based
def next_n(self):
    return (self.persisted_counter + 1, self.id)
```

Or use a hybrid logical clock (HLC) that combines wall-clock with a counter for monotonicity.

### Letting Non-Leader Propose Without Phase 1 in Multi-Paxos

```bash
# WRONG — follower sends Accept directly
on client_cmd(cmd):
    slot = guess_next_slot()
    send Accept(my_n, slot, cmd) to all
```

If a follower bypasses the leader and sends Accept with its own `n`, two proposals can compete for the same slot. The leader's quorum may shift; safety can break under partition.

```bash
# RIGHT — followers redirect to leader
on client_cmd(cmd):
    if i_am_leader:
        slot = next_slot
        next_slot += 1
        send Accept(leader_n, slot, cmd)
    else:
        forward(cmd, leader)
```

### Stale Leader After Partition Heal

```bash
# WRONG — old leader keeps proposing after partition heal
on client_cmd(cmd):
    send Accept(old_n, slot, cmd)
```

If the old leader didn't notice the partition and a new leader was elected on the other side, the old leader is using a stale `n`. Its Accepts will be Nack'd, but it might pollute followers' state if they accept transiently.

```bash
# RIGHT — leader steps down on Nack with higher n
on Nack(higher_n):
    if higher_n > self.current_n:
        self.is_leader = False
        # back to follower; re-elect if needed
```

### Election Timeout Too Aggressive

```bash
# WRONG — tight timeout causes constant elections
ELECTION_TIMEOUT = 50 ms
```

Network jitter exceeds 50ms occasionally → false leader-down triggers → constant election churn → no progress.

```bash
# RIGHT — randomized timeout, large enough to absorb jitter
ELECTION_TIMEOUT = uniform(150, 300) ms   # Raft's default
```

Randomization prevents thundering-herd; the upper bound should be at least 5-10x the typical RTT.

### Snapshot During Active Round

```bash
# WRONG — snapshot mid-Paxos round
on snapshot_request:
    pause()
    snapshot = state_machine.copy()
    save(snapshot)
    truncate_log_up_to(current_slot)
    resume()
```

If a Paxos round is in progress at slot S and the snapshot truncates the log past S, the in-flight round loses its accepted record.

```bash
# RIGHT — coordinate snapshot with consensus state
on snapshot_request:
    safe_slot = highest committed slot whose log entry is no longer needed
    snapshot = state_machine.snapshot_at(safe_slot)
    save(snapshot)
    truncate_log_up_to(safe_slot)
```

Only truncate up to a *committed* slot; never truncate slots that are in flight.

### Confusing "Decided" With "Known to Be Decided"

```bash
# WRONG — proposer assumes decision is committed everywhere immediately
on majority_accepted(slot, v):
    return v  # client sees success
    # but: other replicas may not yet know
```

A value is decided once a majority of acceptors accepts. But other learners (and acceptors) may not know yet — the decision is *durable* but not *visible*. If you read from a non-leader, you might see stale data.

```bash
# RIGHT — propagate decision; reads must see latest
on majority_accepted(slot, v):
    broadcast Decision(slot, v)
    apply v to state machine
    return v to client
```

For linearizable reads, route through the leader (with lease) or via a consensus read.

### Forgetting That Acceptors Don't Talk to Each Other

```bash
# WRONG — peer-to-peer acceptor sync
on Accept(n, v):
    accept locally
    forward to other acceptors  # noise; not the protocol
```

Acceptors don't gossip among themselves. The proposer is the only entity that talks to all acceptors. Adding peer-to-peer forwarding doesn't improve safety; it creates redundant traffic and complexity.

```bash
# RIGHT — proposer fans out
on Accept(n, v):
    accept locally
    reply Accepted to proposer  # proposer aggregates
```

### Mixing Up Paxos Numbers in Multi-Paxos

In Multi-Paxos, the *same* proposal number `n` is reused across many slots within a single leader's term. Some implementations confuse this:

```bash
# WRONG — each slot has its own n
on client_cmd(cmd):
    slot = next_slot
    n = next_proposal_number()  # never reuse!
    send Accept(n, slot, cmd)
```

That's correct for Basic Paxos (per slot) but wasteful for Multi-Paxos. The leader optimization is *exactly* that the same `n` covers all slots once Phase 1 is done.

```bash
# RIGHT — leader's n covers many slots
on client_cmd(cmd):
    slot = next_slot
    send Accept(leader_n, slot, cmd)   # leader_n fixed for term
```

## Reading the Original Papers

### *The Part-Time Parliament* (Lamport 1989/1998)

The Greek-island metaphor — parliamentarians on Paxos who must agree on legislative records despite priests wandering in and out — was Lamport's attempt to make the protocol approachable. It backfired: reviewers found the metaphor distracting.

Reading guide:

- **Skim sections 1-2** (the metaphor setup) — feel free to skip.
- **Section 3 (the basic protocol)** — this is the meat. Translate Greek terms to standard:
  - "ledger" → log of decisions
  - "decree" → value
  - "ballot" → proposal (with number)
  - "priest" → acceptor / proposer / learner role
- **Sections 4-6 (multi-decree, optimization, etc.)** — equivalent to Multi-Paxos.

Key takeaways from the original:

- The protocol is a single algorithm, not a collection of variants.
- Safety is the dominant concern; liveness is "best effort."
- The math is correct but presented informally; the *Paxos Made Simple* version formalizes the proof.

### *Paxos Made Simple* (Lamport 2001)

A 13-page note. The canonical reference. Reading guide:

- **Section 2 (the consensus problem)** — the formal problem statement.
- **Section 2.1 (choosing a value)** — derivation of the protocol from invariants P1, P2, P2a, P2b, P2c.
- **Section 2.2 (the algorithm)** — the protocol in its final form.
- **Section 2.3 (learning a chosen value)** — how learners observe decisions.
- **Section 3 (implementing a state machine)** — Multi-Paxos.

The proof technique is the most enduring contribution: derive the protocol from the invariant you want to maintain. This style (invariant-driven derivation) is now standard in distributed-systems papers.

### *Paxos Made Live* (Chandra, Griesemer, Redstone 2007)

Google engineers' account of implementing Paxos for Chubby. Discusses real-world challenges:

- Disk corruption and how to handle it.
- Master leases for read optimization.
- Membership changes and dynamic configuration.
- Software engineering hazards (testing, debugging).

Required reading for anyone implementing Paxos in production.

### *Paxos Made Moderately Complex* (van Renesse & Altinbuken 2015)

A textbook treatment with detailed pseudocode for Multi-Paxos including reconfiguration. More implementable than the original but less "elegant."

### *Flexible Paxos* (Howard, Malkhi, Spiegelman 2016)

A short, clear paper showing that the majority requirement can be relaxed. Read after understanding classic Paxos.

### *In Search of an Understandable Consensus Algorithm* (Ongaro, Ousterhout 2014)

The Raft paper. Even if you stick with Paxos, reading this clarifies the design space — what tradeoffs Raft makes for understandability and how they translate to Paxos's flexibility.

### *Heidi Howard's PhD Thesis: Distributed Consensus Revised* (2019)

A unifying framework that shows Paxos, Raft, ZAB, VR, and others as variants of a single underlying algorithm. The "common substrate" perspective is invaluable for understanding why these protocols differ in surface but agree in essence.

## Idioms

- **"Paxos is what you build when you've outgrown 2PC."** Two-phase commit doesn't tolerate coordinator failure; Paxos does. Once you need fault-tolerant atomic commit, you move to consensus.

- **"Multi-Paxos is for log-structured replicated state machines."** Single-decree Paxos is rare in production; the value is in replicating an entire command log.

- **"If you don't need leader optimization, basic Paxos is enough."** For low-throughput scenarios (rare, important decisions like configuration changes), basic Paxos's 2 RTTs are fine and you avoid the leader-election machinery.

- **"Use Raft if you can — it's just easier to implement."** Most engineering teams should pick Raft over Paxos for new systems. The understandability advantage compounds over years of maintenance.

- **"Paxos requires only fail-stop, not synchrony."** Common misconception is that Paxos needs reliable networks or synchronized clocks — it doesn't.

- **"Quorum intersection is the only thing that matters for safety."** All of Paxos's variants are essentially: "what quorum shapes preserve the intersection property?" Once you internalize that, the variants stop feeling like a zoo.

- **"The leader is an optimization, not a correctness mechanism."** Two leaders don't break safety, just liveness.

- **"You haven't really implemented Paxos until you've debugged a partition heal."** The hardest bugs surface when split-brain heals and stale leaders compete.

- **"State must be persisted *before* the reply, not after."** The single most common safety bug.

- **"Acceptors don't gossip."** All coordination is via proposers.

- **"FLP says liveness is impossible in pure async; we live with eventual synchrony."** Don't try to defeat FLP — work around it with sensible timeouts and retries.

## See Also

- [distributed-consensus](../cs-theory/distributed-consensus.md) — broader survey of consensus protocols (Paxos, Raft, ZAB, PBFT)
- [cap-theorem](../cs-theory/cap-theorem.md) — fundamental tradeoff between consistency, availability, partition-tolerance

## References

- **Lamport, L.** *The Part-Time Parliament*. ACM Transactions on Computer Systems, 1998. Originally submitted 1989.
- **Lamport, L.** *Paxos Made Simple*. ACM SIGACT News, 2001. The canonical short version.
- **Lamport, L.** *Fast Paxos*. Distributed Computing, 2006. Skip-Phase-1 variant.
- **Lamport, L.** *Generalized Consensus and Paxos*. Microsoft Research Technical Report, 2005.
- **Lamport, L., Massa, M.** *Cheap Paxos*. International Conference on Dependable Systems and Networks (DSN), 2004.
- **Lamport, L., Malkhi, D., Zhou, L.** *Vertical Paxos and Primary-Backup Replication*. PODC, 2009.
- **van Renesse, R., Altinbuken, D.** *Paxos Made Moderately Complex*. ACM Computing Surveys, 2015.
- **Howard, H., Malkhi, D., Spiegelman, A.** *Flexible Paxos: Quorum Intersection Revisited*. arXiv:1608.06696, 2016.
- **Howard, H.** *Distributed Consensus Revised*. PhD Thesis, University of Cambridge, 2019.
- **Mao, Y., Junqueira, F., Marzullo, K.** *Mencius: Building Efficient Replicated State Machines for WANs*. OSDI, 2008.
- **Moraru, I., Andersen, D., Kaminsky, M.** *There Is More Consensus in Egalitarian Parliaments*. SOSP, 2013. EPaxos.
- **Rystsov, D.** *CASPaxos: Replicated State Machines without Logs*. arXiv:1802.07000, 2018.
- **Ongaro, D., Ousterhout, J.** *In Search of an Understandable Consensus Algorithm*. USENIX ATC, 2014. The Raft paper.
- **Chandra, T., Griesemer, R., Redstone, J.** *Paxos Made Live: An Engineering Perspective*. PODC, 2007. Google Chubby's implementation experience.
- **Burrows, M.** *The Chubby Lock Service for Loosely-Coupled Distributed Systems*. OSDI, 2006.
- **Junqueira, F., Reed, B., Serafini, M.** *ZAB: High-Performance Broadcast for Primary-Backup Systems*. DSN, 2011. ZooKeeper Atomic Broadcast.
- **Corbett, J., Dean, J., et al.** *Spanner: Google's Globally-Distributed Database*. OSDI, 2012.
- **Fischer, M., Lynch, N., Paterson, M.** *Impossibility of Distributed Consensus with One Faulty Process*. JACM, 1985. The FLP impossibility result.
- **Liskov, B., Cowling, J.** *Viewstamped Replication Revisited*. MIT Technical Report, 2012. VR — a Paxos contemporary with similar guarantees.
- Lamport's Paxos page: https://lamport.azurewebsites.net/pubs/pubs.html
- Heidi Howard's blog and thesis: https://hh360.user.srcf.net/
- The Raft Consensus Algorithm site: https://raft.github.io/
