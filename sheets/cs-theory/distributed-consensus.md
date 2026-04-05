# Distributed Consensus (Agreement in Faulty Systems)

A complete reference for consensus protocols, impossibility results, fault tolerance models, logical clocks, and state machine replication -- the formal backbone of reliable distributed systems.

## The Consensus Problem

```
Definition: n processes must agree on a single value despite failures.

Three requirements:
  Agreement     — no two correct processes decide differently
  Validity      — the decided value was proposed by some process
  Termination   — every correct process eventually decides

Failure models:
  Crash failure    — process stops permanently, no corrupted messages
  Omission failure — process fails to send or receive messages
  Byzantine failure — process can behave arbitrarily (lie, delay, collude)

  Crash < Omission < Byzantine   (increasing severity)

Synchrony models:
  Synchronous    — known bounds on message delay and step time
  Asynchronous   — no timing assumptions whatsoever
  Partially sync — eventually synchronous (unknown GST)
```

## FLP Impossibility Theorem -- Fischer, Lynch, Paterson, 1985

```
Theorem: No deterministic algorithm can solve consensus in an
         asynchronous system if even one process may crash.

Key insight: you cannot distinguish a crashed process from a slow one
             in an asynchronous network.

Implications:
  - Does NOT mean consensus is unsolvable in practice
  - Practical systems circumvent FLP via:
      1. Randomization (probabilistic termination)
      2. Failure detectors (partial synchrony assumptions)
      3. Timeouts (partially synchronous model)
  - Every practical consensus protocol assumes partial synchrony

Failure detector approach (Chandra & Toueg, 1996):
  - Unreliable failure detector with properties:
      Completeness — every crashed process is eventually suspected
      Accuracy     — some correct process is never suspected (eventually)
  - Weakest failure detector for consensus: Omega (leader oracle)
```

## Lamport Clocks and Vector Clocks

### Lamport Logical Clocks -- Lamport, 1978

```
Each process maintains a counter C(i):
  1. Before each event, increment: C(i) := C(i) + 1
  2. Send message m with timestamp C(i)
  3. On receive: C(j) := max(C(j), timestamp(m)) + 1

Property:
  If a -> b  (a happens-before b)  then  C(a) < C(b)

Converse does NOT hold:
  C(a) < C(b)  does NOT imply  a -> b
  (concurrent events may have ordered timestamps)
```

### Vector Clocks -- Fidge/Mattern, 1988

```
Each process i maintains a vector V[1..n]:
  1. Before each event: V[i] := V[i] + 1
  2. Send message with entire vector V
  3. On receive from j: V[k] := max(V[k], V_msg[k]) for all k
                         then V[i] := V[i] + 1

Characterization theorem:
  a -> b   iff   V(a) < V(b)        (componentwise comparison)
  a || b   iff   V(a) incomparable   (neither dominates)

  V(a) <= V(b)  means  V(a)[k] <= V(b)[k] for all k
  V(a) < V(b)   means  V(a) <= V(b) and V(a) != V(b)

Space cost: O(n) integers per message (n = number of processes)
```

## Total Order Broadcast

```
Also called atomic broadcast. All correct processes deliver the
same messages in the same order.

Properties:
  Validity      — if a correct process broadcasts m, all correct
                   processes eventually deliver m
  Agreement     — if a correct process delivers m, all correct
                   processes eventually deliver m
  Total Order   — if processes p and q both deliver m1 and m2,
                   they deliver them in the same order
  Integrity     — each message is delivered at most once, and only
                   if it was previously broadcast

Equivalence theorem (Chandra & Toueg):
  Total order broadcast is equivalent to consensus.
  (Each can be reduced to the other)
```

## Quorum Systems

```
A quorum system Q over a set of n processes is a collection of
subsets (quorums) such that any two quorums intersect.

Majority quorum:  every subset of size > n/2
  Intersection guarantee: |Q1 ∩ Q2| >= 1

Read/write quorums:
  Qr + Qw > n     (read quorum + write quorum must exceed n)
  Qw + Qw > n     (two write quorums must overlap)

  Common choice: Qr = Qw = floor(n/2) + 1  (majority)

Grid quorum (Cheung et al.):
  Arrange n processes in sqrt(n) x sqrt(n) grid
  Quorum = one full row + one element from each other row
  Size: O(sqrt(n)), but lower availability

Availability vs. load tradeoff:
  Majority — high availability, load = 1/2
  Grid     — lower availability, load = O(1/sqrt(n))
```

## Paxos -- Lamport, 1998 (written 1989)

```
Roles:
  Proposer  — proposes values
  Acceptor  — votes on proposals, provides durability
  Learner   — learns the decided value

Phase 1: Prepare
  1a. Proposer selects ballot number n, sends Prepare(n) to acceptors
  1b. Acceptor receives Prepare(n):
      - If n > highest promised ballot:
          Promise not to accept anything below n
          Reply with (n_accepted, v_accepted) if any
      - Else: ignore or send Nack

Phase 2: Accept
  2a. Proposer receives promises from majority:
      - If any acceptor reported an accepted value:
          Use the value from the highest-numbered accepted ballot
      - Else: proposer is free to choose any value
      - Send Accept(n, v) to acceptors
  2b. Acceptor receives Accept(n, v):
      - If n >= highest promised ballot:
          Accept the proposal, write (n, v) to stable storage
          Notify learners
      - Else: ignore

Safety invariant:
  If a value v is chosen, then every higher-numbered ballot that
  is accepted also has value v.

Liveness:
  Not guaranteed — dueling proposers can livelock.
  Fix: elect a distinguished proposer (leader).

Quorum requirement: majority of acceptors (2f + 1 tolerates f crashes)
```

## Multi-Paxos

```
Optimization for deciding a sequence of values (log entries):

  1. Elect a stable leader (proposer)
  2. Leader runs Phase 1 once for the entire log
  3. For each new entry, leader runs only Phase 2
     (skips Prepare, goes straight to Accept)
  4. If leader fails, new leader runs Phase 1 to reclaim leadership

Benefits:
  - Amortizes Phase 1 over many consensus instances
  - Reduces message complexity from 4 to 2 per entry (steady state)
  - Practical systems: Chubby, Spanner, many databases

Slots and gaps:
  - Each log position is an independent Paxos instance
  - Gaps can occur; filled lazily or via no-op proposals
```

## Raft -- Ongaro & Ousterhout, 2014

```
Designed for understandability. Equivalent to Multi-Paxos in power.

Three subproblems:
  1. Leader election
  2. Log replication
  3. Safety

Server states: Leader, Follower, Candidate

Terms:
  Monotonically increasing logical clock
  Each term has at most one leader
  Term acts as a logical clock to detect stale leaders

Leader Election:
  1. Follower times out (no heartbeat from leader)
  2. Becomes Candidate, increments term, votes for self
  3. Sends RequestVote to all servers
  4. Wins if receives majority of votes
  5. Each server votes for at most one candidate per term
  6. Election restriction: candidate's log must be at least as
     up-to-date as voter's log (prevents stale leaders)

Log Replication:
  1. Client sends command to leader
  2. Leader appends entry to local log
  3. Leader sends AppendEntries RPC to followers
  4. Followers append entry, acknowledge
  5. Leader commits entry when majority acknowledge
  6. Leader notifies followers of committed entries

Safety guarantee (Log Matching Property):
  If two logs contain an entry with the same index and term,
  then all preceding entries are identical.

  Maintained by: AppendEntries consistency check
  (each RPC includes index and term of entry immediately preceding
   new entries; follower rejects if it doesn't match)

Committed entry guarantee:
  If an entry is committed in a given term, it will be present
  in the logs of all leaders for higher terms.
  (Proven via: election restriction + majority overlap)
```

## Viewstamped Replication (VR) -- Oki & Liskov, 1988

```
Precursor to Paxos (published earlier, less well-known).

Key concepts:
  View        — identifies the current leader (primary)
  View change — reconfiguration when primary fails
  Log         — ordered sequence of operations

Normal operation:
  1. Client sends request to primary
  2. Primary assigns op-number, sends Prepare to backups
  3. Backups log operation, send PrepareOK
  4. Primary commits when f+1 (including self) have prepared
  5. Primary sends Commit to backups

View change:
  1. Backup suspects primary failure (timeout)
  2. Sends StartViewChange to all replicas
  3. On f+1 StartViewChange: send DoViewChange (with log) to new primary
  4. New primary selects log with highest view-number
  5. Installs new view, sends StartView to all backups
```

## Zab (ZooKeeper Atomic Broadcast) -- Junqueira, Reed, Serafini, 2011

```
Powers Apache ZooKeeper's replicated state machine.

Guarantees:
  - Total order of all state changes
  - Reliable delivery
  - Primary order: if primary broadcasts a before b, a is delivered first

Phases:
  1. Discovery  — followers find the most up-to-date leader
  2. Synchronization — leader syncs its history with followers
  3. Broadcast  — leader proposes, followers acknowledge, leader commits

Difference from Paxos:
  - Preserves FIFO ordering of leader's proposals
  - Transaction IDs: (epoch, counter) pairs
  - Epoch = leader's term; incremented on each leader change
  - Simpler recovery: leader's history is authoritative
```

## Byzantine Fault Tolerance

### Byzantine Generals Problem -- Lamport, Shostak, Pease, 1982

```
n generals must agree on attack or retreat.
Up to f generals may be traitors (send conflicting messages).

Impossibility result:
  Consensus impossible if n <= 3f
  (need n >= 3f + 1 to tolerate f Byzantine faults)

  With 3 generals and 1 traitor: impossible to distinguish which
  is the traitor — each honest general may see consistent lies.

With authentication (signed messages):
  Consensus possible for any n >= f + 1 (but f + 1 rounds needed)
```

### PBFT (Practical Byzantine Fault Tolerance) -- Castro & Liskov, 1999

```
First practical BFT protocol for asynchronous networks.
Tolerates f faults with n = 3f + 1 replicas.

Normal operation (3-phase):
  1. Pre-prepare — primary assigns sequence number, broadcasts
  2. Prepare     — replicas echo, wait for 2f matching prepares
  3. Commit      — replicas broadcast commit, wait for 2f+1 commits

View change:
  - Replicas suspect faulty primary
  - Send ViewChange with prepared certificates
  - New primary collects 2f+1 ViewChange messages
  - Computes new-view message, starts new view

Message complexity: O(n^2) per consensus instance
Latency: 3 message delays (normal case)

Comparison: crash fault protocols need 2f+1, BFT needs 3f+1
```

## State Machine Replication (SMR)

```
Schneider's framework (1990):

  A deterministic state machine + consensus protocol =
  a fault-tolerant replicated service.

Requirements:
  1. All replicas start in the same initial state
  2. All replicas execute the same commands
  3. All replicas execute commands in the same order
  4. State machine is deterministic

Total order broadcast provides requirement 3.
Consensus provides total order broadcast.

Therefore: consensus is the fundamental building block of SMR.

Linearizability: clients observe behavior indistinguishable from
a single correct server processing commands sequentially.
```

## Leader-Based vs. Leaderless

```
Leader-based (Paxos, Raft, VR, Zab):
  + Simpler protocol (leader serializes decisions)
  + Lower message complexity in steady state
  + Clear responsibility for ordering
  - Leader is single point of bottleneck
  - Leader election adds latency on failure
  - Uneven load distribution

Leaderless (EPaxos, Mencius):
  + No single bottleneck
  + Better geographic distribution
  + Any replica can propose
  - Higher message complexity
  - More complex conflict resolution
  - Harder to implement correctly

EPaxos (Egalitarian Paxos) — Moraru, Andersen, Kaminsky, 2013:
  - Any replica can propose
  - Fast path (1 round) when commands don't conflict
  - Slow path (2 rounds) when commands conflict
  - Optimal commit latency in wide-area deployments
```

## Key Figures

```
Leslie Lamport     — Paxos, Lamport clocks, happens-before relation,
                     bakery algorithm, TLA+, Byzantine generals
                     (co-author), LaTeX
Barbara Liskov     — Viewstamped Replication, PBFT (with Castro),
                     Liskov Substitution Principle, CLU language
Diego Ongaro       — Raft (with Ousterhout), designed for
                     understandability
Michael Fischer    — FLP impossibility (with Lynch & Paterson)
Nancy Lynch        — FLP impossibility, distributed algorithms textbook,
                     I/O automata
Michael Paterson   — FLP impossibility (with Fischer & Lynch)
Miguel Castro      — PBFT (with Liskov), practical Byzantine fault
                     tolerance
Tushar Chandra     — Failure detectors, unreliable failure detectors
                     (with Toueg)
Sam Toueg          — Failure detectors (with Chandra), reduction of
                     broadcast to consensus
Fred Schneider     — State machine replication framework
```

## See Also

- Concurrency Theory
- Distributed Systems
- Complexity Theory
- Operating Systems
- Networking

## References

```
Fischer, M., Lynch, N., and Paterson, M. "Impossibility of Distributed
  Consensus with One Faulty Process." JACM, 1985.
Lamport, L. "The Part-Time Parliament." ACM TOCS, 1998.
Lamport, L. "Paxos Made Simple." ACM SIGACT News, 2001.
Lamport, L. "Time, Clocks, and the Ordering of Events in a Distributed
  System." CACM, 1978.
Lamport, L., Shostak, R., Pease, M. "The Byzantine Generals Problem."
  ACM TOPLAS, 1982.
Ongaro, D. and Ousterhout, J. "In Search of an Understandable Consensus
  Algorithm." USENIX ATC, 2014.
Castro, M. and Liskov, B. "Practical Byzantine Fault Tolerance."
  OSDI, 1999.
Oki, B. and Liskov, B. "Viewstamped Replication: A New Primary Copy
  Method to Support Highly-Available Distributed Systems." PODC, 1988.
Junqueira, F., Reed, B., Serafini, M. "Zab: High-performance Broadcast
  for Primary-Backup Systems." DSN, 2011.
Chandra, T. and Toueg, S. "Unreliable Failure Detectors for Reliable
  Distributed Systems." JACM, 1996.
Schneider, F. "Implementing Fault-Tolerant Services Using the State
  Machine Approach: A Tutorial." ACM Computing Surveys, 1990.
Fidge, C. "Timestamps in Message-Passing Systems That Preserve the
  Partial Ordering." Australian Computer Science Conf., 1988.
Mattern, F. "Virtual Time and Global States of Distributed Systems."
  Parallel and Distributed Algorithms, 1988.
Moraru, I., Andersen, D., Kaminsky, M. "There Is More Consensus in
  Egalitarian Parliaments." SOSP, 2013.
```
