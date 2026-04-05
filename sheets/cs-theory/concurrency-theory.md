# Concurrency Theory (Foundations of Parallel and Distributed Computation)

A complete reference for process calculi, synchronization primitives, correctness conditions, and memory models — the formal backbone of concurrent and distributed systems.

## Process Calculi

### CSP (Communicating Sequential Processes) — Hoare, 1978

```
Core idea: processes communicate via named channels, no shared state.

Syntax:
  STOP             — deadlocked process (does nothing)
  SKIP             — successful termination
  a -> P           — prefix: perform event a, then behave as P
  P [] Q           — external (deterministic) choice
  P |~| Q          — internal (nondeterministic) choice
  P ||| Q          — interleave (independent parallel)
  P [| A |] Q      — alphabetized parallel (sync on set A)
  P \ A            — hiding (internalize events in A)
  P ; Q            — sequential composition

Channel communication:
  c!v -> P         — send value v on channel c, then P
  c?x -> P(x)      — receive value into x on channel c, then P(x)

Traces:  sequences of visible events a process can perform
         traces(a -> b -> STOP) = { <>, <a>, <a,b> }

Refinement:  P refines Q  iff  traces(P) is a subset of traces(Q)
             (and failures/divergences for full models)
```

### CCS (Calculus of Communicating Systems) — Milner, 1980

```
Syntax:
  0                — nil process
  a.P              — action prefix
  P + Q            — choice (summation)
  P | Q            — parallel composition
  P \ a            — restriction (hide channel a)
  P[b/a]           — relabeling

Complementary actions:
  a and a-bar synchronize (handshake) to produce internal action tau

Key relation:  bisimulation equivalence (~)
  P ~ Q iff each can simulate the other step-for-step
```

### Pi-Calculus — Milner, 1992

```
Extension of CCS: channel names can be passed as messages (mobility).

Syntax:
  0                        — nil
  x<y>.P                   — send name y on channel x, then P
  x(z).P                   — receive a name on channel x, bind to z, then P
  (new x) P                — create fresh channel x, scoped to P
  P | Q                    — parallel
  !P                       — replication (infinite copies of P)

Key feature: scope extrusion — a private name can escape its scope
  (new x)(x-bar<x>.0 | x(y).y-bar<v>.0) --> name x sent outside
```

## Actor Model — Hewitt, 1973

```
Fundamental unit: the actor (autonomous computational agent)

Each actor has:
  - A mailbox (message queue)
  - Internal state (private, no sharing)
  - A behavior function

On receiving a message, an actor can:
  1. Send messages to known actors
  2. Create new actors
  3. Change its own behavior for the next message

Properties:
  - No shared state            — all interaction via messages
  - Asynchronous messaging     — send is non-blocking
  - Location transparency      — actors addressed by name, not location
  - Inherent concurrency       — actors run independently

Comparison to CSP:
  CSP:   synchronous channels, process-oriented, algebraic laws
  Actor: async mailboxes, object-oriented, operational semantics
```

## Petri Nets — Carl Adam Petri, 1962

```
A Petri net is a bipartite directed graph N = (P, T, F, M0):

  P   — set of places (drawn as circles)
  T   — set of transitions (drawn as bars/rectangles)
  F   — flow relation: (P x T) union (T x P) -> {0,1,...}
  M0  — initial marking: P -> {0,1,2,...}  (tokens in each place)

Firing rule:
  Transition t is enabled at marking M iff:
    for all p in pre(t):  M(p) >= F(p,t)

  Firing t at M produces M':
    M'(p) = M(p) - F(p,t) + F(t,p)

Reachability:
  M' is reachable from M if there exists a firing sequence
  M = M0 -> M1 -> ... -> Mk = M'

Key properties:
  Boundedness  — places never exceed k tokens (k-bounded)
  Liveness     — every transition can eventually fire
  Deadlock-free — some transition is always enabled
  Reversibility — M0 is reachable from every reachable marking
```

## Mutual Exclusion

### Dijkstra's Solution (1965)

```
First correct software solution for N processes.
Uses shared turn variable and flags.
Proved: mutual exclusion + no deadlock (but no bounded waiting for N > 2).
```

### Peterson's Algorithm (1981)

```
Two-process mutual exclusion with 3 shared variables:

  flag[0] = false, flag[1] = false, turn = 0

  Process i (i = 0 or 1, j = 1 - i):
    flag[i] = true
    turn = j                    // yield to the other
    while (flag[j] && turn == j)
      ; // busy wait
    // critical section
    flag[i] = false

  Guarantees:
    - Mutual exclusion        (at most one in CS)
    - Progress                (no deadlock)
    - Bounded waiting         (at most one bypass)

  Caveat: requires sequential consistency — breaks on modern CPUs
          without memory barriers.
```

### Lamport's Bakery Algorithm (1974)

```
N-process mutual exclusion, no atomic hardware.

  choosing[i] = false, number[i] = 0   for all i

  Process i:
    choosing[i] = true
    number[i] = 1 + max(number[0], ..., number[N-1])
    choosing[i] = false
    for j = 0 to N-1:
      while choosing[j]                // wait until j picks a number
        ;
      while number[j] != 0 && (number[j], j) < (number[i], i)
        ;                              // wait if j has priority
    // critical section
    number[i] = 0

  Properties:
    - Mutual exclusion   (proved via ticket ordering)
    - No deadlock
    - First-come-first-served (FCFS)
    - Tolerates non-atomic reads/writes
```

## Synchronization Primitives

### Semaphores — Dijkstra, 1965

```
Integer variable S with two atomic operations:

  P(S) / wait(S):     while S <= 0: block;  S = S - 1
  V(S) / signal(S):   S = S + 1;  wake one blocked process

Binary semaphore:   S in {0, 1}  (acts as mutex)
Counting semaphore: S in {0, 1, ..., N}  (resource pool of N)

Classic problems solved with semaphores:
  - Producer-consumer (bounded buffer)
  - Readers-writers
  - Dining philosophers
```

### Monitors — Hoare, 1974 / Hansen, 1973

```
High-level synchronization construct:
  - Encapsulated shared data
  - Procedures that operate on the data
  - Implicit mutual exclusion (only one thread active in monitor)
  - Condition variables for signaling:

  condition c;
  c.wait()    — release monitor lock, block, re-acquire on wakeup
  c.signal()  — wake one waiting thread

Hoare semantics:  signaler blocks, signaled thread runs immediately
Mesa semantics:   signaler continues, signaled thread re-checks condition
  (Mesa is what Java, pthreads, and most real systems use)
```

## Deadlock

### Coffman Conditions (1971)

```
Deadlock occurs iff ALL four conditions hold simultaneously:

  1. Mutual exclusion     — resources are non-sharable
  2. Hold and wait        — process holds resources while requesting more
  3. No preemption        — resources cannot be forcibly taken
  4. Circular wait        — cycle in the resource-allocation graph

  Break ANY one condition to prevent deadlock.
```

### Strategies

```
Prevention:    design system to violate at least one Coffman condition
  - Ordered resource acquisition (breaks circular wait)
  - Request all resources at once (breaks hold-and-wait)

Avoidance:     dynamically check if granting a request is safe
  - Banker's algorithm (Dijkstra) — check if system remains in safe state

Detection:     allow deadlocks, then detect and recover
  - Periodically check for cycles in wait-for graph
  - Recovery: kill a process, preempt a resource, rollback
```

## Livelock and Starvation

```
Livelock:
  Processes continuously change state in response to each other
  but make no progress. Unlike deadlock, processes are NOT blocked.
  Example: two people in a corridor, both stepping aside in the same
  direction repeatedly.

Starvation:
  A process is perpetually denied access to a resource it needs,
  even though the system is not deadlocked.
  Cause: unfair scheduling (e.g., priority inversion without
  priority inheritance).
```

## Consistency and Correctness Conditions

### Linearizability — Herlihy & Wing, 1990

```
Each operation appears to take effect at a single instant (linearization
point) between its invocation and response.

  Formally: execution history H is linearizable iff there exists a
  sequential history S such that:
    1. S is a permutation of completed operations in H
    2. S respects the real-time order of H
    3. S is valid according to the sequential specification

  Composable:  if each object is individually linearizable,
               the whole system is linearizable.

  Linearizability = "atomic" consistency for concurrent objects.
```

### Serializability

```
For transactions (not individual operations):
  An execution of concurrent transactions is serializable iff its
  outcome is equivalent to SOME serial (one-at-a-time) execution.

  Does NOT require respecting real-time order.

  Strict serializability = serializability + real-time order
                         = linearizability for transactions.
```

### Happens-Before Relation — Lamport, 1978

```
Partial order on events in a distributed system:

  a -> b ("a happens before b") iff:
    1. a and b are in the same process and a precedes b, OR
    2. a is a send and b is the matching receive, OR
    3. transitivity: a -> c and c -> b

  Concurrent events:  a || b  iff  NOT (a -> b) AND NOT (b -> a)

  Lamport clocks:  assign timestamps C(e) such that
    a -> b  implies  C(a) < C(b)
    (but NOT the converse — use vector clocks for that)
```

## Memory Models

```
Sequential Consistency (SC) — Lamport, 1979:
  All processors see the same total order of operations,
  and each processor's operations appear in program order.

Total Store Order (TSO) — x86, SPARC:
  Stores may be delayed in a store buffer (store-load reordering).
  A load may see the process's own buffered store before it is
  globally visible. Fences (MFENCE) restore SC.

Relaxed / Weak Ordering — ARM, POWER:
  Almost any reordering is allowed unless constrained by barriers.
  Loads, stores, and dependent operations may all be reordered.
  Explicit fence instructions (dmb, isync, lwsync) required.

Hierarchy (strongest to weakest):
  SC  >  TSO  >  PSO  >  Relaxed/Weak

C/C++ memory model (C11/C++11):
  memory_order_seq_cst   — SC for atomics
  memory_order_acquire   — no loads/stores moved before this load
  memory_order_release   — no loads/stores moved after this store
  memory_order_relaxed   — no ordering guarantees (only atomicity)
```

## Key Figures

```
Edsger Dijkstra    — mutual exclusion, semaphores, Banker's algorithm
Tony Hoare         — CSP, monitors, Hoare logic
Carl Hewitt        — actor model
Robin Milner       — CCS, pi-calculus, bisimulation
Leslie Lamport     — bakery algorithm, happens-before, Lamport clocks,
                     TLA+, Paxos, sequential consistency
Carl Adam Petri    — Petri nets
Maurice Herlihy    — linearizability, wait-free algorithms, consensus
                     hierarchy
Michael Fischer    — FLP impossibility result (with Lynch & Paterson)
```

## See Also

- Distributed Systems
- Operating Systems
- Turing Machines
- Complexity Theory
- Type Theory

## References

```
Hoare, C.A.R. "Communicating Sequential Processes." Prentice Hall, 1985.
Milner, R. "Communication and Concurrency." Prentice Hall, 1989.
Milner, R. "The Polyadic Pi-Calculus: A Tutorial." 1993.
Hewitt, C. et al. "A Universal Modular ACTOR Formalism." IJCAI, 1973.
Petri, C.A. "Kommunikation mit Automaten." PhD thesis, 1962.
Dijkstra, E.W. "Solution of a Problem in Concurrent Programming Control." CACM, 1965.
Peterson, G.L. "Myths About the Mutual Exclusion Problem." IPL, 1981.
Lamport, L. "A New Solution of Dijkstra's Concurrent Programming Problem." CACM, 1974.
Lamport, L. "Time, Clocks, and the Ordering of Events." CACM, 1978.
Lamport, L. "How to Make a Multiprocessor Computer That Correctly Executes
  Multiprocess Programs." IEEE TC, 1979.
Herlihy, M. & Wing, J. "Linearizability: A Correctness Condition for
  Concurrent Objects." TOPLAS, 1990.
Coffman, E.G. et al. "System Deadlocks." Computing Surveys, 1971.
Fischer, M., Lynch, N., Paterson, M. "Impossibility of Distributed Consensus
  with One Faulty Process." JACM, 1985.
Herlihy, M. "Wait-Free Synchronization." TOPLAS, 1991.
Adve, S. & Gharachorloo, K. "Shared Memory Consistency Models: A Tutorial."
  IEEE Computer, 1996.
```
