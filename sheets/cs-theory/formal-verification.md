# Formal Verification (Proofs, Model Checking, and Correctness Guarantees)

A practitioner's reference for mathematically proving that software and hardware systems satisfy their specifications.

## Hoare Logic

### Hoare Triples

```
{P} C {Q}

P = precondition  (what must hold before execution)
C = command/program
Q = postcondition (what must hold after execution)
```

**Partial correctness:** If P holds and C terminates, then Q holds.
**Total correctness:** [P] C [Q] -- additionally guarantees termination.

### Core Inference Rules

```
Assignment:    {Q[x/E]} x := E {Q}

Sequencing:    {P} C1 {R},  {R} C2 {Q}
               ────────────────────────
                    {P} C1; C2 {Q}

Conditional:   {P ∧ B} C1 {Q},  {P ∧ ¬B} C2 {Q}
               ────────────────────────────────────
                    {P} if B then C1 else C2 {Q}

While:         {P ∧ B} C {P}
               ─────────────────────────
               {P} while B do C {P ∧ ¬B}
               (P is the loop invariant)

Consequence:   P' → P,  {P} C {Q},  Q → Q'
               ──────────────────────────────
                       {P'} C {Q'}
```

### Weakest Precondition

```
wp(x := E, Q)              = Q[x/E]
wp(C1; C2, Q)              = wp(C1, wp(C2, Q))
wp(if B then C1 else C2, Q) = (B → wp(C1, Q)) ∧ (¬B → wp(C2, Q))
```

The weakest precondition `wp(C, Q)` is the least restrictive condition on the initial state that guarantees Q after executing C.

### Loop Invariants

| Property | Requirement |
|----------|-------------|
| Initialization | Invariant holds before loop entry |
| Maintenance | If invariant holds before iteration, it holds after |
| Termination | Invariant + loop exit condition implies postcondition |
| Variant | Decreasing integer expression for total correctness |

## Model Checking

### Overview

```
Model checking = exhaustive state-space exploration
Input:  System model M (finite-state) + Property φ (temporal logic formula)
Output: M ⊨ φ  or  counterexample trace
```

### Kripke Structures

```
M = (S, S₀, R, L)

S   = finite set of states
S₀  ⊆ S   = initial states
R   ⊆ S×S  = transition relation (total: ∀s ∃s'. (s,s') ∈ R)
L   : S → 2^AP  = labeling function (AP = atomic propositions)
```

### Temporal Logics

| Operator | CTL (branching) | LTL (linear) | Meaning |
|----------|----------------|--------------|---------|
| Always | AG φ | G φ (□ φ) | φ holds on all future states |
| Eventually | AF φ | F φ (◇ φ) | φ holds at some future state |
| Next | AX φ | X φ (○ φ) | φ holds in the next state |
| Until | A[φ U ψ] | φ U ψ | φ holds until ψ becomes true |
| Exists path | EG, EF, EX | (implicit universal) | Some path satisfies property |

**CTL:** path quantifiers (A, E) + temporal operators -- branching time.
**LTL:** no path quantifiers -- all properties hold over all paths.

### State Explosion Problem

```
Components:  n concurrent processes, each with k states
State space: k^n  (exponential in number of components)

Mitigations:
  - Symbolic model checking (BDDs / SAT-based BMC)
  - Partial order reduction
  - Symmetry reduction
  - Abstraction / CEGAR
  - Compositional verification
```

## Theorem Proving (Interactive Proof Assistants)

| System | Logic | Language | Notable Uses |
|--------|-------|----------|-------------|
| Coq | Calculus of Inductive Constructions | Gallina/Ltac | CompCert, Four Color Theorem |
| Isabelle/HOL | Higher-Order Logic | Isar | seL4 microkernel, Archive of Formal Proofs |
| Lean | Dependent Type Theory | Lean 4 | Mathlib, Liquid Tensor Experiment |
| Agda | Martin-Lof Type Theory | Agda | HoTT, verified algorithms |
| ACL2 | First-Order + Induction | Lisp-like | AMD processor verification |

### Curry-Howard Correspondence

```
Propositions  ↔  Types
Proofs        ↔  Programs
Proof checking ↔  Type checking
```

## Abstract Interpretation

```
Concrete domain (C, ⊆)  ←→  Abstract domain (A, ⊑)
         α (abstraction)  →
         ←  γ (concretization)

Galois connection: α(c) ⊑ a  ⟺  c ⊆ γ(a)
```

| Abstract Domain | Tracks | Precision | Cost |
|----------------|--------|-----------|------|
| Sign | +, -, 0 | Very low | O(n) |
| Interval | [lo, hi] | Low | O(n) |
| Octagon | ±x ± y ≤ c | Medium | O(n^3) |
| Polyhedra | Linear constraints | High | Exponential |

**Key tool:** Astree (used by Airbus for flight control software -- zero false alarms on production code).

## TLA+ (Temporal Logic of Actions)

```
Specification = Init ∧ □[Next]_vars ∧ Liveness

Init    = initial state predicate
Next    = disjunction of possible actions (state transitions)
□[A]_v  = always: either A occurs or v is unchanged (stuttering)
vars    = tuple of all state variables
```

| Concept | Description |
|---------|-------------|
| State predicate | Boolean formula over state variables |
| Action | Boolean formula over primed (next) and unprimed (current) variables |
| Stuttering | Steps where no variable changes (enables refinement) |
| Fairness | WF_v(A) = weak fairness, SF_v(A) = strong fairness |

**TLC:** explicit-state model checker for TLA+. Used at AWS (DynamoDB, S3, EBS), Azure (Cosmos DB), Elasticsearch.

## Separation Logic

```
emp           = empty heap
x ↦ v        = x points to value v (and nothing else)
P * Q         = separating conjunction (P and Q hold on disjoint heap regions)
P -* Q        = magic wand (if P is added to current heap, Q holds)
```

### Frame Rule

```
  {P} C {Q}
─────────────────   (C does not modify free variables of R)
{P * R} C {Q * R}
```

Enables local reasoning: verify a command against only the memory it touches; the rest of the heap (R) is automatically preserved.

## SAT/SMT Solvers

### SAT (Boolean Satisfiability)

```
Input:  propositional formula in CNF
Output: satisfying assignment or UNSAT

Core algorithm: CDCL (Conflict-Driven Clause Learning)
  1. Unit propagation
  2. Decision (pick variable + polarity)
  3. Conflict analysis → learn clause
  4. Backjump (non-chronological backtracking)
```

### SMT (Satisfiability Modulo Theories)

```
SMT = SAT + domain-specific theory solvers

Theories:  Linear arithmetic (LIA, LRA)
           Arrays
           Bit vectors
           Uninterpreted functions
           Strings
           Floating point (IEEE 754)
```

**Z3** (Microsoft Research): dominant SMT solver, used in Dafny, KLEE, SAGE, Boogie.

## Property-Based Testing vs Formal Methods Spectrum

```
Manual testing → Example-based tests → PBT → Fuzzing → Static analysis → Model checking → Theorem proving
   ←── increasing automation ──→        ←── increasing assurance ──→
   ←── lower cost ──→                   ←── higher cost ──→
```

| Technique | Soundness | Completeness | Automation |
|-----------|-----------|-------------|------------|
| Unit tests | No | No | High |
| Property-based testing (QuickCheck) | No | No | High |
| Fuzzing (AFL, libFuzzer) | No | No | High |
| Static analysis (abstract interp.) | Sound (over-approx) | No | High |
| Model checking | Sound + Complete (finite) | Yes (finite) | High |
| Theorem proving | Sound + Complete | Yes | Low (interactive) |

## Key Figures

| Name | Contribution | Year |
|------|-------------|------|
| Tony Hoare | Hoare logic, axiomatic semantics | 1969 |
| Robert Floyd | Floyd-Hoare logic precursor, flowchart verification | 1967 |
| Edsger Dijkstra | Weakest precondition calculus, guarded commands | 1975 |
| Edmund Clarke | Model checking (CTL) | 1981 |
| E. Allen Emerson | Model checking (CTL), temporal logic | 1981 |
| Joseph Sifakis | Model checking (process algebras) | 1982 |
| Leslie Lamport | TLA+, temporal logic of actions | 1994 |
| John Reynolds | Separation logic | 2002 |
| Peter O'Hearn | Separation logic, Infer (Facebook) | 2002 |
| Patrick Cousot | Abstract interpretation | 1977 |
| Leonardo de Moura | Z3 SMT solver, Lean theorem prover | 2008 |

Clarke, Emerson, and Sifakis received the 2007 Turing Award for model checking.

## Tips

- Start with the cheapest technique that gives useful guarantees for your domain
- Model checking excels at concurrency bugs (deadlocks, races, livelocks)
- Theorem proving is necessary when the state space is infinite or parametric
- Separation logic is the foundation of tools like Infer (used at Meta on every diff)
- TLA+ is practical for distributed systems design -- write the spec before the code
- Loop invariants are the hardest part of Hoare logic proofs; work backward from the postcondition
- SAT/SMT solvers underpin most automated verification tools; learn to encode problems into them

## See Also

- `detail/cs-theory/formal-verification.md` -- Hoare logic proof rules, LTL/CTL semantics, TLA+ specs, CEGAR
- `sheets/cs-theory/computability-theory.md` -- decidability, halting problem, Rice's theorem
- `sheets/cs-theory/complexity-theory.md` -- complexity classes, SAT is NP-complete
- `sheets/cs-theory/type-theory.md` -- Curry-Howard, dependent types, proof assistants
- `sheets/cs-theory/automata-theory.md` -- finite automata, Buchi automata (for LTL model checking)

## References

- Hoare, "An Axiomatic Basis for Computer Programming" (CACM, 1969)
- Clarke, Emerson, Sistla, "Automatic Verification of Finite-State Concurrent Systems Using Temporal Logic" (TOPLAS, 1986)
- Lamport, "The Temporal Logic of Actions" (TOPLAS, 1994)
- Reynolds, "Separation Logic: A Logic for Shared Mutable Data Structures" (LICS, 2002)
- "Principles of Model Checking" by Baier and Katoen (MIT Press, 2008)
- "Software Foundations" by Pierce et al. (online, 2024) -- Coq-based formal verification course
- de Moura and Bjorner, "Z3: An Efficient SMT Solver" (TACAS, 2008)
- Cousot and Cousot, "Abstract Interpretation: A Unified Lattice Model" (POPL, 1977)
