# Paxos — Deep Dive

The why and the proofs behind the consensus algorithm that runs the world's most important distributed systems.

## Setup

In 1989, Leslie Lamport submitted a paper titled "The Part-Time Parliament" to ACM Transactions on Computer Systems. The paper described, through the metaphor of a fictional Greek island called Paxos, a consensus protocol of remarkable subtlety. Reviewers found it impenetrable. The paper was rejected. Lamport withdrew it, frustrated that distributed-systems researchers could not see past the parable.

The paper sat unpublished for nine years. By 1998, distributed databases needed a rigorous consensus protocol; Lamport's parable was unearthed and finally published. By then, Lamport himself realized the metaphor had backfired. In 2001 he wrote "Paxos Made Simple" — a five-page note stripped of Greek references — and the algorithm finally entered the canon.

Paxos is the protocol Google's Chubby uses to elect masters. It is the protocol Megastore uses to commit cross-datacenter writes. It is the foundation of Spanner's commit phase. ZooKeeper's ZAB is a Paxos variant. etcd uses Raft, which Diego Ongaro explicitly described as "Paxos repackaged for understandability." Whenever a system says "we tolerate F failures with 2F+1 replicas via consensus," the algorithm beneath that claim is some descendant of Paxos.

This deep dive covers the WHY: why the protocol works, why each phase is necessary, why the safety proofs hold, why FLP impossibility forces certain compromises, and why subtle variants (Multi-Paxos, Fast Paxos, Flexible Paxos, EPaxos) exist.

## The Consensus Problem

Formally, a consensus protocol involves a set of N processes (often called acceptors or replicas), each of which proposes a value. The protocol must produce a single chosen value satisfying three properties:

**Agreement (Safety):** No two correct processes decide on different values. Once a value is chosen, no other value can ever be chosen.

**Validity:** The chosen value must have been proposed by some process. The system cannot conjure a value out of thin air.

**Termination (Liveness):** Every correct process eventually decides. The protocol must not run forever.

These three properties seem innocuous, but they conflict in the presence of failures. A protocol that is safe under all conditions can be forced to never terminate; a protocol that always terminates can be tricked into violating safety. The art of consensus is to maximize liveness without ever sacrificing safety.

In practice, "no two values decided" matters far more than "decide quickly." A bank that fails to commit a transaction can retry; a bank that commits a transaction twice has a serious problem. So Paxos and its descendants are designed to be safe always, and to be live whenever the network is well-behaved enough.

## FLP Impossibility Theorem

In 1985, Fischer, Lynch, and Paterson published "Impossibility of Distributed Consensus with One Faulty Process." The theorem is short to state and devastating in scope: in a purely asynchronous system, where messages can be arbitrarily delayed and a single process can crash silently, no deterministic algorithm can guarantee consensus.

The proof proceeds by adversarial scheduling. Define a configuration of the system as the combined state of all processes plus the set of in-flight messages. A configuration is **bivalent** if it can lead to either decision value 0 or decision value 1 (depending on subsequent message ordering). It is **univalent** if only one outcome is possible.

The first step shows that there exists at least one bivalent initial configuration. If every initial configuration were univalent, you could derive a contradiction by considering inputs that vary by a single process — flipping one input cannot flip the decision (or some process is decisive, which contradicts fault tolerance).

The second step shows that from any bivalent configuration, there is some sequence of message deliveries that leads to another bivalent configuration. The adversary, controlling the network, can always find such a sequence — by delaying the message that would force univalence and processing some other message first.

Iterating, the adversary keeps the system bivalent forever. No process ever decides. Liveness fails.

The theorem applies to a strong model: deterministic processes, asynchronous messages, fail-stop failures only (no Byzantine adversary), reliable delivery (messages arrive eventually). It is not a result about adversarial networks; it is a result about pure asynchrony alone. Any algorithm hoping to terminate must assume something more.

## Partial Synchrony

Dwork, Lynch, and Stockmeyer (1988) introduced the partial synchrony model: there exists some Global Stabilization Time (GST) after which all messages arrive within bound Δ, but GST is unknown to the algorithm. Before GST, the network is fully asynchronous.

Under partial synchrony, consensus is solvable. The trick is that algorithms can be designed to be safe always (regardless of network conditions) and live eventually (once the network stabilizes). Paxos is exactly such an algorithm. During asynchronous periods, Paxos may not make progress — proposals may be rejected or interleaved — but it never decides incorrectly. After stabilization, a leader can drive proposals to completion.

This separation of safety from liveness is a recurring theme in distributed-systems design. Safety constraints must be unconditional invariants; liveness is best-effort under benign network behavior. The CAP theorem, the FLP theorem, and the various "consistency under partition" results all reflect this division.

## Basic Paxos Protocol

Paxos has three roles: proposers (suggest values), acceptors (vote on values), and learners (discover the chosen value). In practice, processes play multiple roles. We assume N acceptors, of which a strict majority (⌊N/2⌋+1) is required for any quorum.

A proposal is a pair (n, v) where n is a globally unique, monotonically increasing proposal number and v is the proposed value. Proposal numbers are typically constructed as (round, proposer-id) pairs ordered lexicographically — guaranteeing uniqueness while allowing ordering.

**Phase 1: Prepare/Promise**

A proposer chooses a proposal number n and sends Prepare(n) to a majority of acceptors.

When an acceptor receives Prepare(n):
- If n > any prior proposal number it has promised, it responds with Promise(n, prior_n, prior_v) where (prior_n, prior_v) is the highest-numbered proposal it has previously accepted (or ⊥ if none).
- The acceptor commits to never accept any proposal with number less than n in the future.
- If n ≤ a previously promised number, the acceptor ignores or NACKs the request.

**Phase 2: Accept/Accepted**

If the proposer receives Promise from a majority, it can proceed. It must choose a value v for the Accept message:
- If any Promise contained a (prior_n, prior_v) ≠ ⊥, the proposer must use the value with the highest prior_n.
- Otherwise, the proposer is free to use any value (typically its own input).

The proposer sends Accept(n, v) to a majority. Each acceptor that has not promised something newer than n responds with Accepted(n, v) and stores (n, v) durably.

When a majority of acceptors return Accepted(n, v), the value v is **chosen**. Learners can be informed via the Accepted messages.

The protocol seems mysteriously elaborate. Why two phases? Why force the proposer to adopt a value seen in Phase 1? The answer is in the safety proof.

## Safety Proofs — Expanded

The proof centers on showing that once a value is chosen, no different value can be chosen later. Lamport breaks the proof into a hierarchy of properties. Each property is a refinement of the previous, narrowing toward an implementable invariant.

**P1: An acceptor must accept the first proposal it receives.**

This is the trivial base case. Without P1, no proposal would ever be accepted (acceptors could refuse all). With P1, at least the first round can succeed.

But P1 alone is unsafe. Two proposers could send proposals to disjoint majorities, and both proposals could be accepted by their respective majorities. We need a stronger property.

**P2: If a proposal with value v is chosen, every higher-numbered proposal that is chosen has the same value v.**

P2 is too vague to implement. Lamport refines it:

**P2a: If a proposal with value v is chosen, every higher-numbered proposal accepted by any acceptor has value v.**

P2a implies P2 because a proposal cannot be chosen unless some acceptor accepts it.

**P2b: If a proposal with value v is chosen, every higher-numbered proposal issued by any proposer has value v.**

P2b implies P2a. P2b is the form Paxos directly enforces, via Phase 1.

**P2c: For any (n, v) issued by a proposer, there is a majority Q such that either no acceptor in Q has accepted any proposal with number < n, OR v is the value of the highest-numbered proposal accepted by acceptors in Q with number < n.**

P2c is what Phase 1 provides. The proposer learns from a majority's Promises whether any prior proposals exist; if so, it must adopt the highest-numbered such value.

**The Majority Overlap Property:** Any two majorities of N acceptors intersect in at least one acceptor. This is the foundation. If proposal (n₁, v₁) was chosen by majority Q₁, and a later proposer with number n₂ > n₁ contacts majority Q₂, then Q₁ ∩ Q₂ contains at least one acceptor that accepted (n₁, v₁). That acceptor will report (n₁, v₁) in its Promise, and the new proposer is forced to adopt v₁. Thus v₂ = v₁.

The induction step extends this: even if the chosen value's existence is not yet visible to all acceptors, the majority overlap forces any future proposer to discover it.

**Inductive Argument — Step by Step:**

*Base case (n = the smallest proposal number that was chosen).* Suppose proposal (n_min, v_chosen) was chosen by a majority Q_chosen. By definition, every acceptor in Q_chosen has durably stored (n_min, v_chosen). No proposer has issued a proposal with smaller number that competes (n_min is the smallest chosen). So P2 holds trivially for proposals at n < n_min.

*Inductive hypothesis.* Assume P2 holds for all proposals with number in {n_min, n_min+1, ..., n-1}: if any such proposal was chosen with value v, then all higher-numbered chosen proposals have value v.

*Inductive step.* Consider a new proposer P attempting to issue proposal (n, v_new). P sends Prepare(n) to a majority Q'. Each acceptor in Q' responds Promise(n, prior_n, prior_v) durably committing to refuse all proposals with number < n.

Two cases:

**Case 1: All Promises return ⊥ (no prior accepted value).** This means every acceptor in Q' has accepted no proposal with number < n. Combined with majority overlap, Q' ∩ Q_chosen must contain at least one acceptor a*. Since a* ∈ Q_chosen, it accepted (n_min, v_chosen). But Promise from a* returned ⊥, contradiction. So this case is impossible if a value was chosen at any number < n.

Therefore: if all Promises are ⊥, no value has been chosen at any smaller number. P is free to propose any value. P2c holds trivially.

**Case 2: Some Promise returns (prior_n, prior_v).** Let n* = max(prior_n) across all Promises in Q'. Let v* = the corresponding prior_v. By the protocol, P must propose (n, v*).

We claim v* = v_chosen if any value was chosen at smaller number. By inductive hypothesis, all chosen values at numbers in [n_min, n-1] equal v_chosen. The Promise (n*, v*) reflects an *accepted* proposal, not necessarily a *chosen* one. But by majority overlap: any chosen proposal at number n_c was accepted by majority Q_c. Q' ∩ Q_c is non-empty; some acceptor a* ∈ Q' ∩ Q_c reports (n_c, v_chosen) in its Promise. Among all Promises, the highest-numbered (n*, v*) has n* ≥ n_c. By inductive hypothesis applied to n*, v* = v_chosen.

So P proposes (n, v_chosen). P2c holds. P2b follows: the proposed value matches any chosen value. P2a follows: any acceptance of (n, v_chosen) preserves the chosen value. P2 follows.

This completes the induction. No two different values can be chosen. QED.

The subtlety: P2c does not require that v* equals the *chosen* value at every smaller number — it requires that v* equals the value at the *highest-numbered accepted proposal seen in the majority*. The chain of reasoning that connects "highest accepted" to "chosen at any number" is via the inductive hypothesis. This is why Paxos's proof is delicate; missing the induction breaks the argument.

## Liveness

Pure Paxos has no built-in mechanism for terminating. Two competing proposers can starve each other:

1. Proposer A sends Prepare(1) to majority. All Promise.
2. Proposer B sends Prepare(2) to majority. All Promise (revoking A's promise).
3. Proposer A sends Accept(1, v_A). All NACK because they promised 2.
4. Proposer A retries with Prepare(3). All Promise (revoking B's promise).
5. Proposer B sends Accept(2, v_B). All NACK because they promised 3.
6. Proposer B retries with Prepare(4)...

This is the dueling-proposers livelock. The fix is to elect a stable leader and only let the leader propose. Leaders are elected via a separate mechanism (often Paxos itself, ironically), and they back off on conflict — for example, exponential backoff with randomization.

The Paxos protocol is safe regardless of how many proposers compete, but live only when contention is bounded. This matches the partial-synchrony bargain: safety always, liveness when conditions are good.

## Multi-Paxos Optimization

Real systems do not run a single instance of consensus; they run a sequence of instances, one per log entry. Re-running both phases of Paxos for every entry is wasteful. Multi-Paxos optimizes by amortizing Phase 1 across many entries.

Once a leader is elected and has completed Phase 1 for slot s, it stays the leader. For subsequent slots s+1, s+2, ..., the leader skips Phase 1 entirely — it already has Promises from a majority for proposals beyond any committed value. The leader can directly issue Accept(n, v_i) for each new value v_i.

This reduces the steady-state cost from 2 RTTs per decision to 1 RTT. Phase 1 only re-runs when the leader changes (typically due to failure or timeout).

Multi-Paxos is what most production systems implement. The "leader leases" pattern (Chubby, Spanner) extends Multi-Paxos with time bounds: a leader holds a lease for some duration, and during that lease no other leader can run Phase 1. Leases require a synchronized clock assumption (typically loose: NTP-synchronized, with safety bounded by clock skew).

## Cheap Paxos

Cheap Paxos (Lamport, Massa 2004) addresses cost: in a 2F+1-replica system, why must all 2F+1 be active? At steady state, F+1 are sufficient. Cheap Paxos identifies F "auxiliary" acceptors that remain dormant unless one of the F+1 primaries fails. The auxiliaries have minimal storage and CPU overhead, but they vote in Phase 1 when the configuration changes.

Throughput numbers from the Cheap Paxos paper: with 5 nodes (F=2), Cheap Paxos uses 3 active replicas in steady state, achieving roughly 1.4-1.6× the throughput of Basic Multi-Paxos with 5 active. The savings come from fewer messages per decision and reduced disk-I/O on the auxiliaries during normal operation.

This is useful in geographically distributed deployments where the cost of a fully active replica per data center is high. Cheap Paxos lets you have small "witness" nodes that activate only during failures.

## Fast Paxos

Lamport's Fast Paxos (2006) reduces best-case latency from 2 RTTs to 1 RTT, at the cost of larger quorum sizes. The idea: bypass the leader for "fast rounds." Proposers send their values directly to acceptors. If a majority happens to accept the same value, consensus is immediate.

The catch: with multiple proposers, conflicting values may be sent. The classical-Paxos quorum (majority) is not sufficient to ensure safety in fast rounds. Fast Paxos requires a "fast quorum" of size F+⌈(F+1)/2⌉ (roughly 2N/3 for N=3F+1). When fast rounds fail (due to conflict), the leader resolves via classic Paxos.

Throughput numbers reported in the Fast Paxos paper: in low-contention LAN deployments, Fast Paxos achieves roughly 2× the throughput of Multi-Paxos when conflict probability is below 5%; under high conflict (>30%), Fast Paxos drops to roughly 0.7× of Multi-Paxos due to the fallback overhead.

Fast Paxos is useful when contention is rare and round-trip latency dominates throughput. EPaxos (below) generalizes this idea further.

## Generalized Paxos

Generalized Paxos (Lamport 2005) observes that command ordering matters only when commands conflict. Two reads to the same key can run in any order; two writes to the same key must be ordered. Generalized Paxos accepts a partial order over commands rather than a total order. Conflicting commands are sequenced; non-conflicting commands run concurrently.

Empirical numbers from Lamport's analysis: for read-mostly workloads (95% reads, 5% writes), Generalized Paxos achieves roughly 3-5× the throughput of strict Multi-Paxos because reads can be batched and committed in parallel. For write-heavy workloads, the gain shrinks to under 1.2×.

This allows higher throughput because the "ordering bottleneck" is reduced. The trade-off is complexity: each replica must track the conflict graph, and reads of the state machine must check that conflicting commands are sequenced relative to one another.

## Mencius

Mencius (Mao, Junqueira, Marzullo 2008) is a leaderless multi-Paxos. Instead of a single leader who drives all slots, slot ownership rotates round-robin: replica i owns slots i, i+N, i+2N, etc. When a replica wants to propose at one of its slots, it does so directly. When a replica wants to use someone else's slot, it sends a "skip" command.

Throughput numbers from the Mencius paper, on a 5-replica WAN deployment with 50ms inter-replica RTT: Mencius achieves ~8000 commits/sec balanced across replicas, vs Multi-Paxos's ~3500 commits/sec bottlenecked at the leader. Under leader-skewed workloads, Multi-Paxos catches up; under uniform workloads, Mencius dominates.

Mencius distributes load across replicas and avoids leader bottlenecks. It tolerates failure by allowing other replicas to "take over" failed replicas' slots via classic Paxos. Mencius scales better under bursty workloads but adds complexity in failure recovery.

## Vertical Paxos and Reconfiguration

A persistent question: how does a Paxos cluster change membership? Add a new replica? Remove a failed one?

Naive reconfiguration is unsafe. Suppose the original cluster is {A, B, C} and we want to switch to {A, B, D}. If A and B accept the new configuration but C is still alive and has different state, two majorities now exist: {A, B} (under either config) and {A, C} or {B, C} (under old config). They could decide differently.

Vertical Paxos (Lamport, Malkhi, Zhou 2009) handles reconfiguration by treating the configuration itself as a Paxos-decided value. The system runs at a "configuration version," and switching versions requires a Paxos decision. During the switch, both configurations must agree on the new state.

Raft's "joint consensus" is the same idea: during reconfiguration, decisions require quorums in both old and new configurations. This guarantees safety across the transition.

## Flexible Paxos

For decades the Paxos community treated "majority" as a single concept. Heidi Howard's Flexible Paxos (2016) showed that majority is not strictly necessary; what is necessary is **quorum intersection between Phase 1 and Phase 2 quorums**.

Specifically: let Q1 be the set of Phase 1 quorums and Q2 be the set of Phase 2 quorums. The safety property requires that for every Q ∈ Q1 and every Q' ∈ Q2, Q ∩ Q' ≠ ∅. (And Q2 quorums must intersect each other.)

This generalization opens design space. With 5 acceptors, instead of requiring 3 (majority) for both phases, you could require 4 for Phase 1 and 2 for Phase 2 — guaranteed to intersect (4+2 > 5). Phase 2 latency drops because only 2 acceptors must respond. Phase 1 (rare in Multi-Paxos) requires more, but only on leader change.

Throughput numbers from Howard's evaluation: for a 5-acceptor cluster with grid-style quorums (4×2 split), Flexible Paxos achieves roughly 1.6-1.8× the steady-state throughput of standard Multi-Paxos because Phase 2 contacts fewer acceptors per decision. Recovery (Phase 1) is slower by ~1.5× but happens infrequently.

Flexible Paxos enables asymmetric replica deployments: place 4 acceptors in one datacenter (handling Phase 1 during failover) and 2 in remote sites (handling Phase 2 in steady state). The trade-off is recovery cost.

## EPaxos

Egalitarian Paxos (Moraru, Andersen, Kaminsky 2013) is a leaderless multi-Paxos that decides commands in 1 RTT in the common case. EPaxos uses dependency tracking: when a replica receives a command, it sends a PreAccept including the dependencies it has observed. If a fast quorum of replicas agrees on the dependencies, the command is committed in 1 RTT.

If dependencies disagree (because replicas saw concurrent commands in different orders), EPaxos falls back to a slow path with explicit ordering — 2 RTTs.

Throughput numbers from the EPaxos paper, 5-replica WAN deployment with inter-replica RTT of 80-160ms: under 0% conflict, EPaxos commits ~4× faster than Multi-Paxos at the closest replica (bypassing leader). Under 50% conflict, EPaxos drops to ~1.2× of Multi-Paxos. Median commit latency of EPaxos is the local-quorum RTT (~80ms) vs Multi-Paxos's leader-RTT (~160ms when leader is remote).

EPaxos achieves high throughput under low conflict and degrades gracefully under high conflict. It is one of the most-studied leaderless consensus protocols. Its complexity is its main barrier to adoption: the dependency-tracking logic is intricate and the failure-recovery proof is nontrivial.

## CASPaxos

CASPaxos (Trushkin 2018) is a minimalist Paxos variant that exposes a single CompareAndSwap (CAS) operation on a single register. It is essentially a single Paxos instance per key. Clients call CAS(old, new); the algorithm uses Paxos to decide the new value. State machine replication can be built on top of CASPaxos by treating the log as a sequence of CAS operations, but most users adopt CASPaxos for simpler use cases (configuration registers, locks).

CASPaxos's appeal is API simplicity. It exposes Paxos as a key-value store with linearizable CAS, which is enough to implement distributed locks, leader elections, and many coordination primitives.

## State Machine Replication via Paxos

The standard application of consensus is State Machine Replication (SMR). Define a deterministic state machine M; replicate M on N servers; have each server apply commands in the same order as decided by Paxos. If commands are deterministic and the order matches, all replicas reach identical state.

The Paxos log is a sequence of slots, each holding a command. Slot i is decided independently via a Paxos instance (or via Multi-Paxos amortization). Once slot i is decided, every replica applies command i to its local state machine.

The deterministic-state-machine assumption is critical. If commands have non-determinism (e.g., timestamps, random numbers, file I/O), replicas diverge. Real systems sandbox commands to be deterministic: pre-compute timestamps at the leader, use seeded random, abstract over external resources.

Holes in the log can arise — slot 5 may be decided before slot 4 if their proposals raced. The state machine must be applied in order, so slot 5 must wait for slot 4. This introduces latency on disorder. Multi-Paxos's stable leader minimizes disorder by serializing proposals.

Snapshot management is another concern. The log grows unboundedly. Real systems take periodic snapshots of state-machine state and discard log entries older than the latest snapshot. Snapshots must be replicated separately (a new replica catches up via snapshot + tail of log).

## Paxos vs Raft

Diego Ongaro's Raft (2014) was an explicit attempt to make Paxos understandable. Ongaro and Ousterhout argued that Paxos's reputation for incomprehensibility was not necessary and that an algorithmically equivalent protocol could be presented more intuitively.

Raft's main differences:

- **Strong leader:** Raft requires a leader at all times. The leader is the only proposer. Followers cannot accept conflicting proposals.
- **Log prefix invariant:** Raft enforces that the leader's log is a prefix of any subsequently elected leader's log. This is achieved via the election protocol: a candidate must have at least all committed entries to win.
- **Term numbers:** Raft uses monotonic terms, equivalent to Paxos's proposal numbers but with simpler structure.
- **Reconfiguration via joint consensus:** Raft formalizes reconfiguration as a two-phase process where both old and new configurations must agree.

Raft and Multi-Paxos are equivalent in safety guarantees and in their failure-tolerance bound (F failures with 2F+1 replicas). Raft is generally considered easier to implement and reason about, but its performance characteristics are similar.

In practice, Raft has won the popularity contest. etcd, Consul, CockroachDB, and most modern systems use Raft. Paxos persists in legacy systems (Chubby, ZooKeeper's ZAB, Spanner) and in research papers.

## Real-World Implementations

**Google Chubby:** Lock service using Multi-Paxos. Used internally for leader election in many Google services. Famously described in "The Chubby Lock Service for Loosely-Coupled Distributed Systems" (Burrows 2006). Chubby exposes a filesystem-like API; clients acquire locks by creating files, and Paxos ensures only one client holds a given lock. Chubby's published latency: median session establishment ~10ms, KeepAlive RTT ~2-5ms within a datacenter. Master leases held for tens of seconds. Failure-detection timeouts ~3-10 seconds.

**ZooKeeper:** Apache project, originally Yahoo. Uses ZAB (ZooKeeper Atomic Broadcast), a Paxos-flavored protocol. ZAB's design predates Multi-Paxos clarity and was developed somewhat independently. ZooKeeper is widely used in Hadoop, Kafka, and many distributed systems for coordination.

**etcd:** Raft-based key-value store. Used by Kubernetes for cluster state. CoreOS / Red Hat. etcd's Raft library is widely reused. etcd's published RTT for Raft commit on a 3-node SSD cluster: median ~5-10ms, p99 ~30ms. Throughput tops out around 30-50K writes/sec depending on payload and snapshot frequency.

**Spanner:** Google's globally distributed database. Uses Multi-Paxos within Paxos groups (a Paxos group is a shard, and each shard runs its own consensus). Cross-shard transactions use 2PC over Paxos groups. TrueTime provides global timestamps. Spanner's Paxos groups are typically 3-5 replicas each, deployed across geographically separated datacenters. A typical Spanner deployment has thousands of Paxos groups, one per shard, each independent. The published Spanner numbers: cross-continent commit median ~60-100ms, p99 ~300ms+. Within-region commit ~5-15ms.

**CockroachDB:** Open-source Spanner-inspired database. Uses Raft per range (range = shard). Transaction coordinator handles cross-range consistency. CockroachDB Raft RTT measured in production: same-region p50 ~3-8ms, p99 ~30ms; cross-region p50 ~80-150ms.

**Consul, Nomad:** HashiCorp products using Raft for service discovery and orchestration.

## Common Pitfalls

**Forgetting to persist state before responding:** When an acceptor sends Promise(n) or Accepted(n, v), it must durably write that state to disk before responding. If the acceptor crashes after responding but before persisting, it could later vote inconsistently. This is the most common implementation bug in Paxos.

**Reusing proposal numbers:** Proposal numbers must be globally unique and monotonically increasing. A common mistake is using time-based numbers without disambiguating by proposer ID, causing collisions when clocks drift.

**Unbounded log growth:** The log of decided commands grows indefinitely. Without snapshotting and log truncation, disk fills.

**Configuration changes during operation:** Adding or removing acceptors mid-protocol is unsafe without explicit reconfiguration. This was the source of bugs in early Multi-Paxos implementations.

**Confusing acceptors with proposers in failure analysis:** Paxos tolerates F failed acceptors with 2F+1 acceptors. Proposers are not constrained — any number can fail without affecting safety, only liveness.

**Phase 1 collisions with stale leaders:** A leader that was deposed but still alive can issue Accept messages, which acceptors will refuse. The leader must back off and re-run Phase 1.

**Quorum confusion in flexible quorum systems:** Flexible Paxos's intersection requirement is subtle. An incorrect quorum configuration breaks safety. Always verify the intersection property formally.

## The Configuration-Change Bug

Consider a Paxos cluster {A, B, C, D, E} reconfiguring to remove E. Naively:
1. A new configuration {A, B, C, D} is proposed via Paxos.
2. The configuration is committed.
3. E is removed.

But suppose during step 2, some replicas have committed and others have not. A replica that has committed treats {A, B, C, D} (3 = majority) as a valid quorum. A replica that has not yet committed treats {A, B, C, D, E} (3 = minority) — wait, 3 of 5 is also majority. Both quorums of size 3 exist, but they differ in which 3.

The bug: a quorum {A, B, C} could decide value v under the new config (A, B, C all committed it) while {C, D, E} could decide value v' under the old config (C, D, E all unaware of new config). C is in both, but C has committed the new config — yet D and E believe the old config and form a "majority of 5" by adding C. This requires C to have not yet seen the conflicting decision, which is possible during message delays.

The fix is joint consensus: during transition, decisions require both an old-config quorum AND a new-config quorum. C cannot be counted in two different majorities. After the transition completes, only the new config's quorum is required.

Raft formalizes this explicitly: a "joint configuration" C_old,new requires majorities in both C_old and C_new for any decision. After C_old,new is committed, the system transitions to C_new alone.

This subtlety is why naive reconfiguration breaks Paxos and why Raft's joint-consensus mechanism (or Vertical Paxos's equivalent) is essential.

## Implementation Notes

Implementing Paxos correctly is famously difficult. Lamport himself notes that "Paxos is hard." Some implementation guidelines that have emerged:

- **Use a battle-tested library.** etcd's Raft library, hashicorp/raft, or a similarly mature implementation. Roll your own only if you have months for testing.
- **Test with Jepsen.** Kyle Kingsbury's Jepsen test suite has revealed bugs in nearly every distributed-systems product. Run your implementation through Jepsen-style chaos testing.
- **Model-check critical paths.** TLA+ and PlusCal are the standard tools. Lamport himself uses TLA+ to specify Paxos. Real systems (e.g., Microsoft's TLA+ at Azure scale) have caught subtle bugs.
- **Persist before respond.** Every state-changing message must trigger a durable write before the response is sent.
- **Bound log growth.** Snapshot regularly. Coordinate with garbage collection of old log entries.
- **Idempotent commands.** Clients may retry; the state machine must handle duplicate commands. Use unique request IDs.

## Performance Tuning

In production, Paxos performance depends on:

- **Quorum size:** Smaller quorums mean faster decisions but worse failure tolerance.
- **Network topology:** Cross-datacenter replication adds RTT to every decision.
- **Disk write latency:** Persistence is on the critical path. Use fast SSDs or battery-backed cache.
- **Batching:** Batch many client requests into a single Paxos decision.
- **Pipelining:** Allow multiple decisions in flight, even if some are not yet committed.
- **Read leases:** Avoid running Paxos for reads. The leader can serve reads locally if it has a lease.

Spanner and CockroachDB combine all these tricks. They achieve millions of transactions per second across multiple datacenters by careful engineering atop Paxos/Raft.

## Theoretical Connections

Paxos sits at a particular point in the design space of consensus protocols. Other points include:

- **PBFT (Practical Byzantine Fault Tolerance, Castro and Liskov 1999):** Tolerates Byzantine (arbitrary) failures, not just crashes. Quorum size is 3F+1 to tolerate F Byzantine failures.
- **HotStuff (Yin et al. 2019):** Optimized BFT consensus with linear communication. Used in some blockchain systems.
- **Tendermint:** BFT consensus designed for blockchains.
- **Nakamoto consensus (Bitcoin):** Probabilistic consensus via proof-of-work. Different model entirely; not Paxos-style.

The CAP theorem is often invoked here. Under network partitions, a system must choose between consistency (no two replicas decide differently) and availability (every replica responds). Paxos chooses consistency: if a replica is in the minority partition, it cannot serve writes. Availability is sacrificed for safety.

For systems that prioritize availability, eventual-consistency designs (Dynamo, Cassandra, CRDTs) make different trade-offs. Paxos is for the consistency side of CAP.

## Looking Forward

Active research areas in consensus include:

- **Verified implementations:** IronFleet, Verdi, and similar projects use formal verification to prove Paxos/Raft implementations correct.
- **Geo-distributed consensus:** Reducing latency for transactions spanning continents. EPaxos, CALVIN, and recent proposals address this.
- **Byzantine consensus at scale:** HotStuff, Streamlet, and others target permissioned blockchain settings.
- **Hybrid logical clocks:** Combining physical and logical time for richer ordering guarantees.
- **Sharding and federation:** How do many small Paxos clusters interact to scale globally?

Paxos remains a 25+-year-old algorithm at the heart of modern distributed systems. Its proofs are among the most carefully studied in computer science. Whatever future consensus protocols emerge, they will all be measured against Paxos's safety guarantees — because Paxos showed it was possible to do consensus right, and everyone since has had to match that bar.

## Detailed Worked Example — 5-Acceptor Basic Paxos

To make the protocol concrete, let us walk through a complete Paxos round on a 5-acceptor cluster {A1, A2, A3, A4, A5}. Two proposers (P1 and P2) compete to set the value of a register.

**Step 1: P1 prepares.** P1 chooses proposal number n=10 and sends Prepare(10) to A1, A2, A3, A4, A5.

**Step 2: Acceptors promise.** A1, A2, A3, A4, A5 have never seen a higher proposal number. Each responds Promise(10, ⊥) (no prior accepted value). Each durably writes "promised 10" to disk before responding.

**Step 3: P1 receives a majority of promises.** P1 has heard from at least 3 acceptors (a majority). All Promises contained ⊥, so P1 is free to use any value. P1 chooses v_1 = "Apple."

**Step 4: P1 sends Accept.** P1 sends Accept(10, "Apple") to a majority. Suppose A1, A2, A3 are reachable. Each compares 10 with their last promise (also 10), which matches. Each accepts: durably writes (10, "Apple") and responds Accepted(10, "Apple").

**Step 5: Value chosen.** P1 has a majority Accepted. The value "Apple" is chosen. Learners are notified.

Now consider the failure case: P2 begins competing.

**Step 6: P2 prepares with a higher number.** P2 chooses n=20 and sends Prepare(20). Acceptors A1, A2, A3, A4, A5 respond. A1, A2, A3 have accepted (10, "Apple"). They respond Promise(20, 10, "Apple"). A4, A5 had only promised 10; they respond Promise(20, ⊥).

**Step 7: P2 must adopt "Apple".** P2 sees Promises with values. The highest is (10, "Apple"). P2 must propose "Apple" — even though P2 wanted "Banana." This is P2c in action: the safety property forces P2 to extend the prior decision rather than overwrite it.

**Step 8: P2 sends Accept(20, "Apple").** Acceptors update to (20, "Apple"). The value "Apple" is reaffirmed.

The chosen value remained "Apple" throughout. P2 lost its preferred value, but consistency was preserved.

If P1 had crashed after Phase 1 (before Phase 2), the chosen value would be undefined — no acceptor has yet recorded "Apple" as accepted. P2 would still be free to propose "Banana" because all Promises return ⊥. This is the boundary between "no value chosen" and "value chosen." Once a majority accepts, the value is locked in.

## Worked Example — Dueling Proposers (Livelock Trace)

To see how dueling proposers actually unfold, consider acceptors {A1, A2, A3} and two proposers P, Q. Time advances downward.

| t   | event                                          | A1 promised | A1 accepted | A2 promised | A2 accepted | A3 promised | A3 accepted |
|-----|------------------------------------------------|-------------|-------------|-------------|-------------|-------------|-------------|
| 0   | initial                                        | 0           | ⊥           | 0           | ⊥           | 0           | ⊥           |
| 1   | P sends Prepare(1) to all                      |             |             |             |             |             |             |
| 2   | A1, A2, A3 receive, respond Promise(1, ⊥)      | 1           | ⊥           | 1           | ⊥           | 1           | ⊥           |
| 3   | Q sends Prepare(2) to all                      |             |             |             |             |             |             |
| 4   | A1, A2, A3 receive, respond Promise(2, ⊥)      | 2           | ⊥           | 2           | ⊥           | 2           | ⊥           |
| 5   | P sends Accept(1, "X") to all                  |             |             |             |             |             |             |
| 6   | A1, A2, A3 NACK (promised 2 > 1)               | 2           | ⊥           | 2           | ⊥           | 2           | ⊥           |
| 7   | P retries: Prepare(3)                          |             |             |             |             |             |             |
| 8   | A1, A2, A3 respond Promise(3, ⊥)               | 3           | ⊥           | 3           | ⊥           | 3           | ⊥           |
| 9   | Q sends Accept(2, "Y") to all                  |             |             |             |             |             |             |
| 10  | A1, A2, A3 NACK (promised 3 > 2)               | 3           | ⊥           | 3           | ⊥           | 3           | ⊥           |
| 11  | Q retries: Prepare(4)                          |             |             |             |             |             |             |
| ... | (repeats forever)                              |             |             |             |             |             |             |

No accepted value is ever recorded. The protocol is safe (no value is chosen wrongly) but not live. Backoff and leader election break the cycle: if P backs off (e.g., randomized exponential delay), Q can complete Phase 2 between t=4 and a subsequent P retry.

## Worked Example — Leader Election Ladder

A practical leader-election protocol on top of Paxos: each candidate runs a Paxos instance for the role "current leader at epoch e." The candidate with the highest accepted leader-value at the highest epoch wins.

| t   | event                                                     | epoch | leader candidate | accepted |
|-----|-----------------------------------------------------------|-------|------------------|----------|
| 0   | initial; epoch 0; no leader                                | 0     | none             | ⊥        |
| 1   | Node A nominates itself: runs Paxos at epoch 1, value="A"  | 1     | A                | ⊥        |
| 2   | A reaches Phase 2 quorum; majority accepts ("A", epoch 1) | 1     | A                | ("A",1)  |
| 3   | A holds lease for 30s; sends heartbeats                    | 1     | A                | ("A",1)  |
| 4   | At t=20s A crashes silently                                | 1     | A (stale)        | ("A",1)  |
| 5   | At t=35s, lease expires; B detects no heartbeat            | 1     | A (stale)        | ("A",1)  |
| 6   | B nominates itself: Paxos at epoch 2, value="B"            | 2     | B (proposing)    | ("A",1)  |
| 7   | B's Phase 1 sees prior accepted ("A",1) — but it's epoch 1 < 2; so B is free to propose "B" | 2 | B | ("A",1) |
| 8   | B reaches Phase 2 quorum; accepted ("B", epoch 2)          | 2     | B                | ("B",2)  |
| 9   | B is leader; A's old promises are now obsolete             | 2     | B                | ("B",2)  |

The crucial design choice: each leadership "term" is a separate Paxos instance keyed by epoch. Phase 1 of epoch n+1 is independent of epoch n's outcome, so a stale leader cannot block election of a new one once the lease expires.

Lease management uses bounded clock skew. If A's clock is 5 seconds ahead and B's is 5 seconds behind, A's 30-second lease may appear to expire 10 seconds early at B. Spanner's TrueTime addresses this by exposing the uncertainty interval explicitly; algorithms wait through TT.now() to ensure no overlap.

## State Machine Replication: Worked Log Example

Consider a key-value store replicated via Paxos. Slots in the log correspond to commands.

Slot 0: PUT k1 v1
Slot 1: PUT k2 v2
Slot 2: GET k1
Slot 3: PUT k1 v3

Each slot is a Paxos instance. Slot 0 is decided first (Paxos converges on PUT k1 v1). Slot 1 is decided next, etc.

In Multi-Paxos, the leader holds Phase 1 promises across all future slots. To propose slot 2, it just sends Accept(20, GET k1) — no Prepare needed.

If the leader fails between proposing slot 1 and slot 2, the new leader must complete slot 1 if it was partially decided, then take over.

The state machine reads the log in order. Each command is applied deterministically. The result for the GET in slot 2 returns v1 (since slot 0 was PUT k1 v1 and no later slot has overwritten by slot 2).

## Failure Scenarios in Detail

**Scenario A: Single acceptor failure.** A1 dies. The cluster {A2, A3, A4, A5} still has 4 nodes; majority is 3. Decisions continue. When A1 recovers, it learns the missed decisions from peers and rejoins.

**Scenario B: Network partition.** {A1, A2} are partitioned from {A3, A4, A5}. Neither side has majority alone; {A1, A2} is 2 of 5, {A3, A4, A5} is 3 of 5. The 3-side can decide; the 2-side cannot. After partition heals, A1 and A2 catch up.

**Scenario C: Leader transition.** L1 was leader, dies. Acceptors timeout and trigger leader election. L2 wins. L2 runs Phase 1 to learn any pending decisions, then takes over Phase 2.

**Scenario D: Liveness violation.** Two proposers continuously preempt each other (dueling proposers). Without a leader-election mechanism, no progress is made. With backoff and leader election, progress resumes once one proposer dominates.

**Scenario E: Acceptor disk corruption.** A3's persistent state is lost. A3 is now equivalent to a new acceptor — it forgets prior promises. This violates safety: A3 might accept an old proposal that other acceptors have rejected. The cluster must detect this (e.g., via persistent identity tokens) and refuse to seat A3 until reinitialized.

**Scenario F: Acceptor restart with stale state.** A3 restarts with old data. If "promised 10" is on disk but the cluster is now at proposal 100, A3 accepts proposal 100 (10 < 100). No safety violation.

## Comparison Across Variants

| Variant         | RTTs/decision | Quorum size       | Best-case throughput vs Multi-Paxos | Best for                          |
|-----------------|---------------|-------------------|-------------------------------------|-----------------------------------|
| Basic Paxos     | 2             | Majority          | 0.5×                                | Single decision                   |
| Multi-Paxos     | 1 (steady)    | Majority          | 1.0× (baseline)                     | Log-style replication             |
| Cheap Paxos     | 1 (steady)    | F+1 active        | 1.4-1.6×                            | Cost-sensitive deployments        |
| Fast Paxos      | 1 (best)      | Larger fast quorum| 1.5-2.0× low contention             | Low-contention reads/writes       |
| Generalized     | 1 (commute)   | Majority          | 3-5× read-heavy                     | Commutative ops                   |
| Mencius         | 1 (steady)    | Majority          | 2-3× balanced load                  | Bursty writes                     |
| EPaxos          | 1 (low conf)  | Fast quorum       | 2-4× low conflict                   | Geo-distributed, low conflict     |
| CASPaxos        | 1-2           | Majority          | ~1×                                 | Single register                   |
| Flexible Paxos  | 1 (small Q2)  | Asymmetric        | 1.6-1.8× small Q2                   | Read-heavy or skewed              |
| Raft            | 1 (steady)    | Majority          | ~1× (equivalent to Multi-Paxos)     | When understandability matters    |

The "right" variant depends on workload:
- High write throughput: Multi-Paxos or Raft.
- Geo-distribution: EPaxos or Flexible Paxos.
- Cost-sensitive: Cheap Paxos.
- Conflict-heavy: stick with Multi-Paxos.

## Storage Engine Considerations

Paxos requires durable persistence for promised proposal numbers and accepted values. The storage layer matters:

**Synchronous fsync():** Each Paxos round requires at least one fsync per acceptor. With slow disks (HDD), this is the bottleneck — hundreds of operations per second per acceptor. Modern systems use batching (multiple decisions per fsync) and group commit to amortize.

**SSD vs HDD:** SSDs reduce per-fsync latency from ~10ms (HDD) to ~100µs. This dramatically improves throughput.

**Battery-backed RAM:** Some systems use NVRAM or battery-backed RAM to persist without fsync. Lower latency but higher cost and complexity.

**Write-ahead log (WAL):** Standard pattern. Each Paxos message is logged before being applied. On restart, the log is replayed.

**Snapshot + log truncation:** Periodic snapshots allow truncating old log entries. The snapshot interval trades recovery time for storage.

## Pipelining and Batching

Real Paxos implementations pipeline multiple decisions concurrently. Each slot in the log is a separate Paxos instance, but they share the leader's Phase 1 quorum.

**Pipeline depth:** Limited by acceptor memory and network buffer sizes. Typical depth: tens to hundreds of in-flight slots.

**Batching:** Each Paxos decision can include many client requests. The leader buffers requests, batches them into a single Accept, and decides them together. Throughput scales with batch size.

**Trade-off:** Batching adds latency (waiting to fill a batch) and reduces RPC overhead. Tuning depends on workload mix.

## Read Paths

Paxos as written serializes all operations through consensus. Reads are slow: a full Paxos round per read.

**Read leases:** The leader holds a lease — a time-bounded right to serve reads locally. Lease holders need not run Paxos for reads. The lease must be renewed before expiry (via Paxos heartbeats).

Lease validity assumes bounded clock skew. If a leader's clock is fast and the next leader's is slow, both might believe they hold the lease simultaneously. Spanner handles this via TrueTime (uncertainty intervals) and explicit "wait out" periods.

**Quorum reads:** An alternative — read from a quorum and return the highest-version value. No leader needed. Higher latency than lease reads but no clock assumption.

## Latency Distributions

In practice:
- Same-datacenter Paxos: ~1-2ms per decision.
- Cross-datacenter Paxos: ~10-100ms (limited by RTT).
- Cross-continent Paxos: ~150-300ms (limited by RTT).

Tail latencies can be much worse — slow disks, GC pauses, network jitter all contribute. Real systems aim for 99th-percentile commit latency targets.

## Membership Reconfiguration in Detail

A 5-node cluster wants to switch to a 7-node cluster (add 2 nodes).

**Naive approach (broken):** Replace config entry directly. Old config still believes it has 3 acceptors as majority. New nodes haven't synced. Disaster.

**Joint consensus approach (Raft):**
1. Propose C_old,new = {old_5} ∪ {new_7}. Decisions require quorum in both old (3 of 5) AND new (4 of 7).
2. Once C_old,new is committed, propose C_new = {new_7} alone.
3. Once C_new is committed, the transition is complete.

During the joint phase, both quorums are required. This guarantees that no decision can occur in only one configuration — preventing splits.

**Single-server changes (Raft optimized):** Add or remove one node at a time. The intersection of "majority of N" and "majority of N±1" is non-empty, so safety holds.

## Geo-Distributed Optimization

For globally distributed systems, naive Paxos has high latency (cross-continent RTTs). Optimizations:

**Local quorums (Cassandra-style):** Restrict each datacenter to a local quorum. Only critical operations (consistency violations) escalate to global Paxos.

**EPaxos for low-contention regions:** Use EPaxos to commit non-conflicting operations in 1 RTT to nearest replicas only.

**Multi-leader systems:** Each datacenter has its own leader for local operations. Cross-datacenter operations use traditional Paxos. This is the model Spanner uses (Paxos groups are sharded, each with its own leader).

**Lease-based local reads:** Reads served locally via leases avoid cross-datacenter coordination.

The art of geo-distribution is choosing what to coordinate globally vs locally. CRDT-style designs avoid coordination entirely; Paxos-style designs coordinate everything; real systems are hybrids.

## Verification and Formal Proofs

Paxos's safety has been formally proven in many frameworks:

**TLA+:** Lamport's own preferred formal-methods tool. The TLA+ specification of Paxos is publicly available and widely used as a teaching example.

**Coq:** Verdi (Wilcox et al.) proved Raft correct in Coq, including the safety properties and the joint-consensus reconfiguration.

**Isabelle/HOL:** Various Paxos verifications, including handling of reconfiguration and Byzantine variants.

**IronFleet:** Microsoft Research project — verified Multi-Paxos and a key-value store, with end-to-end safety proofs against the implementation code.

These verifications have caught real bugs in production systems. Even widely deployed Paxos libraries had subtle safety issues that formal methods exposed.

## Production Tuning

Lessons from large-scale Paxos deployments:

**Heartbeat intervals:** Too short and CPU is wasted; too long and failure detection is slow. Typical: 100ms heartbeats with 1-2 second timeout.

**Election timeout randomization:** Without randomization, multiple followers can become candidates simultaneously, splitting the vote. Randomize timeout per server (e.g., 1500-3000ms).

**Slow client detection:** Disconnect clients that don't ack heartbeats; they may be holding open expensive resources.

**Backpressure:** When the log grows faster than commits, throttle clients. Without backpressure, OOM is inevitable.

**Snapshot frequency:** Trade off CPU/IO for snapshot vs replay time on restart. Typical: snapshot every few thousand log entries.

**Log compaction:** Once a snapshot exists, log entries before it can be discarded. Free disk space.

**Crash testing:** Kill processes randomly; verify safety. Real-world: chaos engineering (Chaos Monkey style).

**Network testing:** Inject delays, drops, partitions. Jepsen-style tests.

## Common Bugs in Practice

**1. Stale promise persisted but not flushed.** Acceptor responds Promise(n) but the disk write was buffered. Crash. On restart, the promise is lost. New round may proceed with smaller n, breaking safety.

**2. Forgotten last-accepted in Promise.** Acceptor forgets to include its last accepted value in Promise. New leader proposes a fresh value. Different from chosen. Safety violation.

**3. Off-by-one in proposal numbers.** Proposal numbers must be strictly monotonic per proposer. Off-by-one bugs (reusing the same number) cause acceptors to accept conflicting values.

**4. Configuration change race.** Adding a member during an active Paxos round before joint consensus is established. The new member doesn't know about ongoing decisions and could disrupt.

**5. Network partition handling.** Some implementations treat "no response from N/2 nodes" as failure; others as a partition. The behavior matters for split-brain prevention.

**6. Term/proposal-number confusion.** Raft has terms, Paxos has proposal numbers. Confusing the two when porting algorithms is a common error.

## When NOT to Use Paxos

Paxos is overkill for:

**Single-master replication:** If only one writer exists, simpler protocols (primary-backup with quorum acks) suffice.

**Eventually-consistent stores:** Dynamo, Cassandra, Riak don't need Paxos. CRDTs or vector clocks handle their consistency model.

**Read-heavy workloads with lax consistency:** Often, weakly-consistent reads are fine. Paxos's stricter guarantees are unnecessary.

**Centralized systems:** A single-server database doesn't need consensus.

**Transient cluster state:** Service discovery (e.g., DNS) doesn't need strong consensus.

Use Paxos when:
- You need strong consistency across multiple replicas.
- Failure tolerance is important.
- The data is durable and consistent reads matter (financial transactions, configurations, locks).

## Paxos Variants Decision Tree

Choosing the right Paxos variant for your system:

**Question 1: Single decision or sequence of decisions?**
- Single: Basic Paxos.
- Sequence: Multi-Paxos.

**Question 2: Stable leader or rotating leader?**
- Stable: Multi-Paxos with leader election.
- Rotating: Mencius, EPaxos.

**Question 3: Geo-distributed?**
- Yes: EPaxos for low-conflict, Flexible Paxos for asymmetric topology, Spanner-style for global transactions.
- No: Multi-Paxos or Raft.

**Question 4: Conflict-heavy or conflict-light?**
- Conflict-heavy: Multi-Paxos (sequential).
- Conflict-light: EPaxos (1 RTT).

**Question 5: Cost-constrained?**
- Yes: Cheap Paxos (passive acceptors).
- No: Standard Multi-Paxos.

**Question 6: Implementation simplicity?**
- Simple: Raft.
- Maximum performance: Multi-Paxos with hand-tuning.

In practice, most production systems use Raft (or Multi-Paxos for legacy). Exotic variants are reserved for specialized workloads.

## Network Models and Their Impact

Paxos's behavior depends on network model assumptions:

**Synchronous:** Bounded message delivery. Strongest assumption. Allows time-based protocols. Real networks are not synchronous.

**Asynchronous:** No bounds on delivery. FLP applies — no deterministic consensus. Paxos requires partial synchrony.

**Partial synchronous:** Eventually-bounded delivery. Standard model for Paxos. Can guarantee progress eventually.

**Mostly synchronous:** Synchronous most of the time, with rare async periods. Real-world networks. Paxos handles gracefully.

**Lossy:** Messages can be dropped. Paxos handles via retry; no fundamental issue.

**Reordering:** Messages may arrive out of order. Paxos uses sequence numbers / proposal numbers, robust.

**Duplication:** Same message arrives multiple times. Paxos is idempotent in critical paths; safe.

**Byzantine:** Some replicas misbehave arbitrarily. Standard Paxos breaks. Use PBFT or related.

## Paxos in Cloud Native Systems

**Kubernetes etcd:** etcd uses Raft. Stores cluster state. Without etcd, Kubernetes can't make scheduling decisions. etcd's reliability is foundational to cloud-native infrastructure.

**Consul:** Service discovery and config store. Uses Raft. Provides strongly consistent KV store with TTL-based locking.

**Nomad:** Workload scheduler. Uses Raft for cluster coordination.

**Vault:** Secret management. Uses Raft (newer versions) for HA storage.

**ZooKeeper:** Older but still widely used. ZAB protocol (Paxos-flavored). Used by Hadoop, Kafka, HBase.

**Apache Kafka:** Uses ZooKeeper for broker coordination (transitioning to KRaft, a Raft variant).

**Apache Cassandra:** Uses Paxos (Lightweight Transactions) for compare-and-set operations. Most operations use eventual consistency.

These systems demonstrate Paxos / Raft's foundational role in modern infrastructure. Outages in etcd or ZooKeeper take down entire Kubernetes clusters or Hadoop installations.

## Why Did Raft Win Adoption?

Despite Paxos being older and conceptually equivalent, Raft has dominated newer system designs. Several reasons:

**Pedagogy:** Ongaro's paper explicitly targeted understandability. Examples, diagrams, and clear explanations made Raft easier to teach.

**Open Source momentum:** etcd, hashicorp/raft, and other libraries are mature, well-tested, and easily adopted.

**Strict log ordering:** Raft enforces strict log prefixes. This invariant simplifies reasoning and debugging.

**Joint consensus reconfiguration:** Explicit, separate from normal operation. Easier to verify correct.

**Visualizations:** The Raft website (raft.github.io) has interactive visualizations. Paxos has nothing comparable for free.

**Industry trend:** Companies want code their engineers can read. Raft's clearer model wins.

**Performance equivalence:** Raft and Multi-Paxos perform similarly. No performance cost to choosing Raft.

## Future of Distributed Consensus

Active research areas:

**Beyond Paxos/Raft:** Newer algorithms aim for better latency or throughput. EPaxos achieves 1-RTT in low-conflict cases. Mencius distributes leadership.

**Verified implementations:** IronFleet, Verdi, and others provide formally verified consensus libraries.

**Byzantine for permissioned networks:** PBFT, HotStuff, Tendermint for Byzantine consensus in known-membership networks.

**Quantum-resistant consensus:** Some research on consensus protocols resistant to quantum-computing attacks on cryptography.

**Geo-distributed transactions:** Spanner/CockroachDB approach combines Paxos groups with timestamps. Newer designs (e.g., Calvin) take different trade-offs.

**Reconfiguration in production:** Vertical Paxos and joint consensus are well-understood, but the engineering of reconfiguration in real systems remains tricky.

**Asymmetric topologies:** Flexible Paxos enables novel deployment patterns. Edge computing, IoT, and cellular networks have different requirements.

## Lamport's Wisdom

A Lamport quote: "Any protocol that has not been proven correct is incorrect." Paxos was proven correct in his original paper, and the proof has been re-verified many times. This rigor is the backbone of modern distributed systems.

Another Lamport observation: "Distributed systems are systems where the failure of a computer you didn't even know existed can render your own computer unusable." Paxos is the algorithm that makes such systems robust.

His message: take consensus seriously. Implement carefully. Test exhaustively. Use formal methods. The cost of getting Paxos wrong is data loss, downtime, and shaken trust. The reward of getting it right is decades of reliable operation.

Paxos's longevity — three decades and counting — testifies to the timelessness of well-designed algorithms. Even as hardware evolves, network speeds increase, and software stacks change, the algorithm remains relevant. Because consensus is a fundamental problem, the solution is fundamentally important.

## Detailed Trace: Multi-Paxos Phase 1 Recovery

A new leader takes over after the previous leader's crash. It must complete any in-progress Paxos rounds before driving new ones.

**Initial state:** Acceptors {A1, A2, A3, A4, A5}. Previous leader L1 was running Multi-Paxos. Slot 100 has been decided. Slot 101 was being decided when L1 crashed: A1 and A2 had accepted (proposal n=50, value="X"), but A3, A4, A5 had not yet voted on slot 101.

**Step 1:** L2 elected via leader-election protocol (e.g., a separate Paxos instance for "current leader" identity).

**Step 2:** L2 issues Prepare(60) for all slots ≥ 101. (Multi-Paxos: one Prepare batch per slot range.)

**Step 3:** Acceptors respond with Promise messages. For slot 101:
- A1: Promise(60, prior=(50, "X")) — already accepted.
- A2: Promise(60, prior=(50, "X")).
- A3, A4, A5: Promise(60, ⊥) — never accepted.

**Step 4:** L2 sees that at least one Promise has a prior value. By the protocol, L2 must propose "X" for slot 101.

**Step 5:** L2 sends Accept(60, "X") for slot 101.

**Step 6:** All 5 acceptors accept. Slot 101 is now decided as "X".

**Step 7:** L2 starts handling new client requests, assigning them to slots 102, 103, ....

The crucial point: L2 cannot decide a different value for slot 101, even though L1 had not yet "decided" it from the system's perspective. The act of A1 and A2 having accepted (50, "X") was sufficient to lock in "X" as the slot's value. L2 discovers this via Phase 1's Promise messages.

**What if A1 and A2 are now unreachable during recovery?** L2 issues Prepare(60). A3, A4, A5 respond with Promise(60, ⊥). L2 has a majority (3) of Promises, all with ⊥. L2 is free to choose any value. L2 picks "Y" and sends Accept(60, "Y") to A3, A4, A5. Slot 101 is now "Y".

But wait — A1 and A2 have "X". When they recover, they accept (60, "Y") since 60 > 50. State converges on "Y". The earlier accepted (50, "X") is lost.

**Was this safe?** Yes. The chosen value is whatever majority of acceptors have accepted in the same proposal number. Slot 101 was never "chosen" with value "X" because only 2 acceptors had it (not majority). The system had no commitment to "X". Choosing "Y" is correct.

**What if 3 acceptors had accepted "X"?** Then Phase 1 of any future round, contacting any majority, would include at least one of those 3 (by majority overlap). That acceptor's Promise would include (50, "X"). L2 would be forced to use "X". Safety preserved.

This is the practical incarnation of the safety proof. Quorum overlap ensures that any accepted-by-majority value is rediscovered in subsequent Prepares.

## Implementation Lessons from Production

**Google Chubby:** Chubby's original implementation discovered numerous edge cases not in the original Paxos paper. Memory leaks in Phase 1 retransmission, race conditions in master election, file-descriptor exhaustion, etc. The lesson: Paxos in pure form is academic; Paxos in production is a thousand engineering details on top.

**etcd's Raft:** Even Raft's clearer presentation didn't eliminate bugs. etcd has had subtle bugs around leader transfers, configuration changes, and snapshot installation. Each was fixed and the test suite expanded.

**Spanner:** Google's Spanner paper acknowledges that the implementation was significantly more complex than the abstract algorithm. TrueTime's clock-uncertainty bounds are critical engineering details.

**Lessons:**
1. Build formal models early. TLA+ specs catch bugs before code.
2. Test exhaustively with chaos engineering.
3. Plan for snapshots and log truncation from day one.
4. Reconfiguration is harder than it looks.
5. Tail latencies dominate user experience.

## Closing: The Paxos Way of Thinking

Paxos teaches a way of thinking about distributed systems that goes beyond the algorithm:

**Safety first, liveness second:** Many bugs come from prioritizing liveness (responding quickly) over safety (responding correctly). Paxos enforces the discipline.

**Quorums as the unit of truth:** No single replica is authoritative. Truth emerges from majority agreement.

**Persistence before response:** Every state change must be durably stored before being acknowledged. This eliminates a whole class of failure modes.

**Monotonic decision:** Once decided, never undecided. Reconfiguration must respect prior decisions.

**Bounded participation:** Any decision involves only the relevant quorum, not all replicas. This is what enables fault tolerance.

These principles, distilled from Paxos, apply to any distributed system. Even systems not using Paxos for consensus benefit from thinking in these terms.

The next time you design a distributed system, ask: what is the safety invariant? What is the smallest quorum that must agree? What state must persist? How does configuration change? Paxos provides answers to these questions in its specific domain. The questions themselves apply far more broadly.

## Performance Numbers from Real Systems

**Spanner Paxos groups (cross-continent):** Median commit latency ~10 ms within region, 60-100ms across regions. Tail (99th percentile) ~100 ms in-region, 300+ms across regions. Throughput depends on Paxos group, typically 10K txn/sec per group. A typical Spanner instance has 1000s of Paxos groups, achieving aggregate throughput in the millions of txn/sec.

**Chubby (single datacenter):** Median session establishment ~10 ms. KeepAlive RTT ~2-5ms. Master election typically ~5 seconds. Master holds lease for tens of seconds (default 30s).

**etcd Raft RTT (default Kubernetes deployment):** Read latency ~1 ms (with leases), write latency ~5-10 ms in-region SSD, p99 ~30ms. Throughput ~10K-30K ops/sec depending on payload size and snapshot frequency.

**ZooKeeper:** Similar order of magnitude as etcd. Tuned setups achieve 50K+ writes/sec with appropriate batching and dedicated SSD storage.

**CockroachDB Raft RTT:** Same-region p50 ~3-8ms, p99 ~30ms; cross-region p50 ~80-150ms. Per-range throughput 1-5K commits/sec; cluster scales horizontally with range count.

**Performance scales with:** disk speed (SSD vs HDD: 10-50× difference), network speed (1 Gbps vs 10 Gbps: 2-3× difference under throughput-bound load), leader stability (no leader changes vs frequent: 5-10× difference), batching factor (batch=1 vs batch=100: 10-50× difference). Tuning these parameters often yields 5-10× improvements without changing the algorithm.

## A Final Note on Trust

Paxos is the algorithm we trust to keep our most important distributed state correct. When you commit a payment in your bank, that commit goes through a Paxos-like consensus. When Kubernetes schedules your container, etcd's Raft made the decision. When Google saves your spreadsheet, Spanner's Paxos protected it.

The intellectual debt to Lamport — and to Paterson, Lynch, Fischer, Dwork, Stockmeyer, and the others who developed the theory — is enormous. They formalized what consensus means, proved what's possible, and gave us algorithms to achieve it.

Modern distributed systems work because we can rely on consensus as a primitive. We don't reinvent it for each system; we use proven libraries and protocols. Paxos and Raft are the foundations of that reliability.

This is the gift of theoretical computer science to engineering practice: rigorous, provable algorithms for problems we cannot afford to get wrong. Paxos exemplifies that gift.

## Further Reading and Study Path

For practitioners wanting to deeply understand Paxos:

**Phase 1 — Conceptual:**
1. Lamport's "Paxos Made Simple" — the canonical short introduction.
2. Diego Ongaro's Raft thesis — clearer explanations of related algorithms.
3. Maurice Herlihy's "Distributed Computing" textbook chapters on consensus.

**Phase 2 — Formal:**
4. Read the original "Part-Time Parliament" paper carefully.
5. Study the TLA+ specifications of Paxos and Multi-Paxos.
6. Work through the safety proof step by step.

**Phase 3 — Implementation:**
7. Study an open-source implementation (etcd's raft, hashicorp/raft).
8. Implement Paxos yourself in Go or Rust.
9. Run Jepsen-style tests against your implementation.

**Phase 4 — Variants:**
10. Read EPaxos, Flexible Paxos, Mencius papers.
11. Understand the trade-offs each variant addresses.
12. Pick a variant suitable for your use case.

**Phase 5 — Production:**
13. Read post-mortems of consensus failures (Cloudflare, AWS, GitHub all have public ones).
14. Study how Spanner, Chubby, ZooKeeper, etcd handle edge cases.
15. Build your own production-grade consensus system (with eyes wide open about the difficulty).

This study path takes years. There are no shortcuts. But the rewards — being able to design and build resilient distributed systems — are substantial.

The journey is its own reward: each stage deepens your understanding of distributed systems, of algorithm design, and of the deep constraints physics and mathematics impose on coordination at a distance.

## Worked Example: Basic Paxos with 5 Nodes

Setup: 5 acceptors (A1..A5), 2 proposers (P1, P2). The required majority is 3 of 5.

### Round 1 — single proposer succeeds

```text
T=0:    P1 picks proposal number n=10, value v="X"
T=10ms: P1 sends Prepare(10) to A1, A2, A3
T=15ms: A1, A2, A3 receive Prepare(10)
        Each acceptor: max_promised = none (first round); promise(10, no_prior_value)
T=25ms: P1 receives Promise(10, ⊥) from A1, A2, A3 — majority reached
T=25ms: P1 sees no prior accepted value → uses its own value v="X"
T=30ms: P1 sends Accept(10, "X") to A1, A2, A3
T=40ms: A1, A2, A3 each: max_promised==10, no higher promise → accept(10, "X")
T=50ms: P1 receives Accepted(10, "X") from A1, A2, A3 — majority reached
        Value "X" is now CHOSEN.
T=60ms: P1 broadcasts Decided("X") to learners.
```

### Round 2 — dueling proposers, second wins

```text
T=0:    P1 picks n=10, v="X"; P2 picks n=11, v="Y" (concurrent)
T=10ms: P1 sends Prepare(10) to A1, A2, A3
        P2 sends Prepare(11) to A3, A4, A5
T=15ms: A1, A2: max_promised := 10; promise(10, ⊥)
        A4, A5: max_promised := 11; promise(11, ⊥)
        A3: receives Prepare(10) first, max_promised := 10; then receives Prepare(11), 11 > 10 so max_promised := 11; promise(11, ⊥)
T=25ms: P1 receives Promise(10, ⊥) from A1, A2 — only 2 of needed 3
        P1 also receives a NACK from A3 (since A3 now promised 11)
        P1 fails to reach majority on Prepare → MUST restart with n > 11.
T=25ms: P2 receives Promise(11, ⊥) from A3, A4, A5 — majority reached
T=30ms: P2 sends Accept(11, "Y") to A3, A4, A5
T=40ms: A3, A4, A5: max_promised==11; accept(11, "Y")
T=50ms: P2 receives Accepted(11, "Y") from A3, A4, A5 — chosen.
T=55ms: P1, having failed, retries with n=12, but the safety invariant means:
        when P1 sends Prepare(12), it will see A3's accepted value "Y" in the promise,
        and Phase 2 P1's Accept(12, ?) MUST use "Y" — proving safety.
```

### Round 3 — leader fails after Phase 1

```text
T=0:    P1 picks n=20; v="X"
T=10ms: P1 sends Prepare(20) to A1..A5
T=20ms: All acceptors promise(20, ⊥)
T=21ms: P1 crashes before sending Accept.
T=2s:   P2 wakes up, picks n=21, v="Y"
T=2.01s: P2 sends Prepare(21) to A1..A5
T=2.02s: All promise(21, ⊥) — no Accept yet from P1, so no prior value to surface
T=2.03s: P2 sends Accept(21, "Y")
T=2.04s: All accept(21, "Y") — chosen.

# The crash before Phase 2 is benign. Liveness deferred to next leader,
# safety preserved (no value was ever chosen).
```

### Round 4 — leader fails BETWEEN sending Accept (split outcomes)

```text
T=0:    P1 picks n=30, v="X"
T=10ms: P1 sends Prepare(30) to A1..A5; all promise.
T=20ms: P1 sends Accept(30, "X") to A1..A5 — but only A1, A2 receive before P1 crashes.
T=21ms: A1, A2 accept(30, "X"); A3, A4, A5 never received Accept.
        Value "X" is NOT yet chosen (only 2 of 5 accepted; need majority).
T=2s:   P2 picks n=31, v="Y"
T=2.01s: P2 sends Prepare(31) to A1..A5
T=2.02s: A1: promise(31, accepted=(30,"X"))   ← surfaces P1's accepted value
        A2: promise(31, accepted=(30,"X"))
        A3, A4, A5: promise(31, ⊥)
T=2.03s: P2 receives 5 promises. Picks the highest-numbered accepted: (30, "X").
        Per safety invariant P2c: P2 MUST send Accept(31, "X"), not "Y".
T=2.04s: P2 sends Accept(31, "X") to A1..A5
T=2.05s: All accept(31, "X") — "X" is chosen.

# Even though P1 crashed mid-round, the value it was advancing ("X") is
# safely chosen by the next round. This is the deep safety property:
# any partially-completed proposal will dominate any future Accept that
# observes it via Promise's prior-accepted-value reporting.
```

## Performance Numbers from Real Deployments

| System | Median latency (RTT) | Throughput per leader | Notes |
|---|---:|---:|---|
| Google Chubby | ~10ms (intra-zone) | ~1k ops/s | Designed for coarse-grained locks |
| ZooKeeper (ZAB) | ~5ms (LAN) | ~50k ops/s | Single leader bottleneck |
| etcd v3 (Raft) | ~5ms (intra-zone) | ~10k ops/s | gRPC overhead noticeable |
| CockroachDB (Range Raft) | ~3ms (intra-zone) | ~1M ops/s aggregate (sharded) | Per-range Raft groups |
| Spanner (Multi-Paxos) | ~10ms (intra-DC) | ~1M ops/s aggregate | TrueTime adds tens of ms for external consistency |
| ScyllaDB (LWT/Paxos) | ~5ms median, ~50ms p99 | ~10k ops/s per partition | Quorum reads + Paxos for writes |

Cross-region adds 50-300ms RTT, which is why most production systems pin Paxos groups to a single region/zone.

## Configuration-Change Bug — Joint Consensus

The naive approach: "to add acceptor A6, broadcast 'config change' and start using {A1..A6} from now on."

The problem: in the brief window where some nodes have switched to the new config and others haven't, two majorities exist:

```text
Old config majority: {A1, A2, A3} of {A1..A5}  →  3 of 5 = majority
New config majority: {A4, A5, A6} of {A1..A6}  →  3 of 6 = NOT majority

But: {A1, A2, A3} could decide value X under old config
While: {A4, A5, A6} could decide value Y under new config
And neither overlaps with the other.

Two values chosen. Safety violated.
```

Raft's solution (joint consensus):

```text
Phase 1: append "configuration C(old, new)" entry — both old and new majorities required for any decision
Phase 2: once C(old, new) is committed, append "configuration C(new)" — only new majority required
```

During the joint phase, decisions must be approved by BOTH a majority of the old config AND a majority of the new config — this overlap requirement is the key safety property.

Cheap Paxos (Lamport 2001) and Vertical Paxos (Lamport 2009) explore lighter-weight reconfiguration with auxiliary acceptors.

## Failure Scenarios

| Scenario | Safety preserved? | Liveness impact |
|---|---|---|
| Single acceptor crashes | Yes | None (others form majority) |
| Minority acceptors crash | Yes | None |
| Majority acceptors crash | Yes | No progress until majority recovers |
| Network partition isolating minority | Yes | Minority side cannot decide; majority side continues |
| Network partition isolating majority | Yes | Majority side continues normally |
| Leader crashes after Promise but before Accept | Yes | New round needed; ~election timeout latency |
| Leader crashes during Accept (partial delivery) | Yes (next round surfaces accepted value) | New round needed |
| All acceptors crash with disk loss | NO | Catastrophic; safety lost |
| Acceptor lies (Byzantine) | NO | Paxos assumes fail-stop; use BFT-Paxos for adversarial |
| Two leaders concurrent | Yes (proposal numbers ensure ordering) | Repeated Phase 1 failures (livelock); needs leader stability |
| Clock skew between nodes | Yes (proposal numbers don't depend on clock) | None |
| Storage write reorder/loss | NO | Acceptors must fsync before responding |

## When NOT to Use Paxos

- Read-mostly with relaxed consistency: use eventual consistency / CRDTs / quorum reads
- Single-writer scenarios: use a leader+followers replication protocol (lighter-weight)
- Coordination at human timescales: use a simple database with locks
- Across many independent shards: use per-shard Paxos groups, not one global one
- When BFT is needed: use PBFT, HotStuff, or similar
- When throughput >>>> safety: consider reducing replication factor
- When latency is paramount and you trust the leader: leader-based Raft variants

## Decision Tree: Which Variant?

```text
Need replicated state machine?
├── Yes → Multi-Paxos or Raft
│   ├── Need geo-replication with low-latency reads? → Spanner-style (Paxos + TrueTime)
│   ├── Need leaderless throughput? → EPaxos
│   ├── Want simplicity over generality? → Raft
│   └── Existing Paxos infra? → Multi-Paxos
└── No → just consensus on a single value?
    ├── One-shot decision? → Basic Paxos
    ├── Compare-and-swap on a register? → CASPaxos
    └── Need 1 RTT in best case? → Fast Paxos (3f+1 acceptors)

Membership changes frequent?
├── Yes → Joint consensus (Raft) or Cheap Paxos / Vertical Paxos
└── No → Static membership (simpler)

Geo-distributed (>50ms RTT)?
├── Yes → consider Mencius, EPaxos, or sharded local Paxos
└── No → Multi-Paxos or Raft

Need BFT?
├── Yes → PBFT, Tendermint, HotStuff
└── No → Paxos / Raft
```

## Cloud-Native Systems Using Paxos / Paxos-Derived

| System | Protocol | Use |
|---|---|---|
| Google Chubby | Multi-Paxos | Distributed locks |
| Google Spanner | Multi-Paxos + TrueTime | Database replication |
| Apache ZooKeeper | ZAB (Paxos-flavored) | Configuration / locks |
| etcd | Raft | Kubernetes config store |
| CockroachDB | Range Raft | SQL DB replication |
| Yugabyte | Range Raft | SQL DB replication |
| FoundationDB | Custom (Paxos-influenced) | KV store + transactions |
| Apache BookKeeper | Custom (quorum ledger) | Kafka tier-2 storage |
| Apache Kafka (KRaft, post-3.3) | Raft | Replaces ZooKeeper |
| RethinkDB (defunct) | Raft | Replication |
| Riak | Riak Core (vector clocks, not Paxos) | KV store |
| Cassandra LWT (post-2.0) | Paxos | Compare-and-set |
| ScyllaDB LWT | Paxos | Compare-and-set |
| Apache Pulsar | ZooKeeper-based | Message broker |
| TiDB / TiKV | Multi-Raft | NewSQL |
| Vitess | etcd (Raft) | MySQL sharding metadata |

## References

- Lamport, L. (1998). "The Part-Time Parliament." ACM Transactions on Computer Systems.
- Lamport, L. (2001). "Paxos Made Simple." ACM SIGACT News.
- Lamport, L. (2006). "Fast Paxos." Distributed Computing.
- Lamport, L. (2005). "Generalized Consensus and Paxos." Microsoft Research Tech Report.
- Lamport, L., Massa, M. (2004). "Cheap Paxos." DSN.
- Lamport, L., Malkhi, D., Zhou, L. (2009). "Vertical Paxos and Primary-Backup Replication." PODC.
- Fischer, M.J., Lynch, N.A., Paterson, M.S. (1985). "Impossibility of Distributed Consensus with One Faulty Process." JACM.
- Dwork, C., Lynch, N., Stockmeyer, L. (1988). "Consensus in the Presence of Partial Synchrony." JACM.
- Howard, H., Mortier, R. (2020). "Paxos vs Raft: Have we reached consensus on distributed consensus?" PaPoC.
- Howard, H., Malkhi, D., Spiegelman, A. (2016). "Flexible Paxos: Quorum Intersection Revisited." OPODIS.
- Moraru, I., Andersen, D.G., Kaminsky, M. (2013). "There Is More Consensus in Egalitarian Parliaments." SOSP.
- Mao, Y., Junqueira, F.P., Marzullo, K. (2008). "Mencius: Building Efficient Replicated State Machines for WANs." OSDI.
- Ongaro, D., Ousterhout, J. (2014). "In Search of an Understandable Consensus Algorithm." USENIX ATC.
- Castro, M., Liskov, B. (1999). "Practical Byzantine Fault Tolerance." OSDI.
- Burrows, M. (2006). "The Chubby Lock Service for Loosely-Coupled Distributed Systems." OSDI.
- Hunt, P., Konar, M., Junqueira, F.P., Reed, B. (2010). "ZooKeeper: Wait-free coordination for Internet-scale systems." USENIX ATC.
- Corbett, J.C. et al. (2012). "Spanner: Google's Globally-Distributed Database." OSDI.
- Trushkin, D. (2018). "CASPaxos: Replicated State Machines without logs." arXiv.
- Yin, M., Malkhi, D., Reiter, M.K., Gueta, G.G., Abraham, I. (2019). "HotStuff: BFT Consensus in the Lens of Blockchain." PODC.
- Hawblitzel, C. et al. (2015). "IronFleet: Proving Practical Distributed Systems Correct." SOSP.
- Wilcox, J.R. et al. (2015). "Verdi: A Framework for Implementing and Formally Verifying Distributed Systems." PLDI.
- Kingsbury, K. "Jepsen: distributed-systems testing." https://jepsen.io

## Worked Examples (Extended)

### Example 1: Naive Paxos — Single Round, No Failures

5-node cluster {A, B, C, D, E}. Proposer A wants to propose value `X`.

- **Phase 1a (Prepare)**: A picks proposal number `n=1`, sends `Prepare(1)` to all 5 acceptors.
- **Phase 1b (Promise)**: All 5 respond `Promise(1, ⊥)` (none have accepted anything). A receives 5 ≥ majority (3 of 5).
- **Phase 2a (Accept)**: A sends `Accept(1, X)` to all 5.
- **Phase 2b (Accepted)**: All 5 respond `Accepted(1, X)`. Value `X` is chosen.
- **Phase 3 (Learn)**: A informs learners. Total: 5 × 4 = 20 messages, 4 RTTs (or 5 with learner notification).

### Example 2: Competing Proposers (the dueling-proposers problem)

Proposer A picks `n=1`, Proposer B picks `n=2`. Both run Phase 1 in parallel.

1. A sends `Prepare(1)`. Acceptors respond `Promise(1, ⊥)`.
2. B sends `Prepare(2)`. Acceptors see 2 > 1, respond `Promise(2, ⊥)`. They've now promised n=2.
3. A sends `Accept(1, X)`. Acceptors reject (1 < 2).
4. A retries with `n=3`. `Prepare(3)` arrives.
5. Acceptors respond `Promise(3, ⊥)`.
6. B sends `Accept(2, Y)`. Acceptors reject (2 < 3). B retries with n=4.

Without external coordination, two proposers can starve each other indefinitely. **Solution**: leader election (Multi-Paxos) — only one proposer is "active" at a time.

### Example 3: Recovery After a Value Was Already Chosen

Proposer A succeeded with `n=1, value=X`. Proposer B wakes up later, doesn't know `X` was chosen.

1. B sends `Prepare(2)`. Acceptors that already accepted `(1, X)` respond `Promise(2, (1, X))` — they include the most recent accepted value.
2. B receives Promises. Per the Paxos rules, B MUST propose `X` (the highest-numbered accepted value among Promises) — NOT its own preferred value.
3. B sends `Accept(2, X)`. Acceptors accept.
4. `X` is "re-chosen" with proposal n=2.

This is the **safety property** in action: once chosen, no different value can be chosen, even by a later proposer that didn't know about the original choice.

### Example 4: Network Partition

Cluster {A, B, C, D, E}. Partition splits into {A, B} and {C, D, E}.

- **Minority {A, B}**: Cannot reach majority (need 3 of 5). Any proposal stalls. Unavailable but consistent.
- **Majority {C, D, E}**: Reaches quorum. Proposals succeed. Available and consistent.
- **Healing**: A and B sync via the standard protocol — when they next see a higher proposal number from C/D/E, they update.

**Liveness** sacrificed on minority side; **safety** never violated.

### Example 5: Multi-Paxos Steady-State

After leader election, leader L runs Multi-Paxos for log slots 1, 2, 3, ...

- **Slot 1**: Phase 1 done during election (the leader's `Prepare(1)` covers ALL future slots — implicit). Phase 2: `Accept(1, op_1)`. Quorum responds. **1 RTT.**
- **Slot 2**: `Accept(1, op_2)`. **1 RTT.**
- **Slot 3**: `Accept(1, op_3)`. **1 RTT.**

Steady-state cost: 1 RTT per operation, no Phase 1 needed (skipped because the leader's `n=1` is still the highest seen). On leader failover, new leader runs Phase 1 with `n=2`; from then all operations use `n=2`.

## Performance Numbers (Real-World)

| System | Algorithm | Throughput | p99 Latency | Notes |
|--------|-----------|-----------:|------------:|-------|
| Google Chubby | Multi-Paxos | ~50K ops/sec | ~10 ms | LAN, 5-node ensemble |
| Apache ZooKeeper | Zab | ~20K writes/sec | ~5 ms | similar deployment |
| etcd | Raft | ~10K writes/sec | ~10 ms | 3-5 node cluster |
| Consul | Raft | ~5K writes/sec | ~10 ms | conservative defaults |
| Spanner | Multi-Paxos + TrueTime | ~10K ops/sec/group | ~10-100 ms | cross-region |
| HotStuff | BFT-Paxos chain | ~3K ops/sec | ~50 ms | tolerates 1/3 Byzantine |

Throughput scales linearly with batching (commits 100 ops in one Accept). Latency = max latency among the fastest majority — quorum is gated by the slowest member.

## Joint Consensus (Membership Changes)

Naive Paxos cannot safely change cluster membership. The classical solution: **joint consensus** — temporarily run TWO configurations, where any decision must be approved by both.

1. Start with `Cold = {A, B, C}`.
2. Propose `Cnew = {A, B, D}` via standard Paxos.
3. During transition, every operation must be quorum-approved in BOTH `Cold` AND `Cnew` simultaneously.
4. Once `Cnew` is confirmed, drop `Cold`. Run normal Paxos on `Cnew` only.

Cost: during the joint phase, latency = max(P99 in Cold, P99 in Cnew). Acceptable because membership changes are rare.

Raft uses **single-server reconfiguration** instead — change one server at a time, each step is a small enough delta that joint consensus isn't needed. Simpler, slower migrations.

## When to Use What — Decision Tree

```
Need consensus?
│
├─ Crash failures only (no Byzantine)?
│  ├─ Need understandability for ops team? → Raft
│  ├─ Need max throughput / battle-tested? → Multi-Paxos (etcd, ZK, Chubby)
│  └─ Need leaderless? → CASPaxos / EPaxos
│
└─ Byzantine failures possible (untrusted nodes)?
   ├─ Permissioned (known node identities)? → PBFT or HotStuff
   ├─ Permissionless (open membership)? → Nakamoto consensus (Bitcoin)
   └─ Need fast finality? → Tendermint, Algorand, HotStuff variants
```

For 95% of distributed-systems problems in practice, **use an existing library** (etcd, ZooKeeper, Consul). Implementation correctness is HARD — even Google's first Paxos implementation in Chubby had bugs found in production.
