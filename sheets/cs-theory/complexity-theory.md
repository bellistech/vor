# Complexity Theory (Classes, Reductions, and the P vs NP Question)

A practitioner's reference for computational complexity classes, polynomial reductions, and the landscape of tractable vs intractable problems.

## Core Complexity Classes

### Definitions

```
P       = problems solvable in polynomial time by a deterministic TM
NP      = problems verifiable in polynomial time (or solvable by a nondeterministic TM)
co-NP   = complements of NP problems (e.g., "no satisfying assignment exists")
NP-hard = at least as hard as every problem in NP (via poly reduction)
NP-complete = NP-hard AND in NP (the hardest problems still in NP)
```

| Class | Deterministic? | Time Bound | Key Example |
|---|---|---|---|
| P | Yes | O(n^k) | Shortest path, sorting, matching |
| NP | No (verify: yes) | O(n^k) verify | SAT, Clique, TSP (decision) |
| co-NP | No (verify complement) | O(n^k) verify | Tautology, primality (before AKS) |
| NP-complete | No | O(n^k) verify | 3-SAT, Vertex Cover, Subset Sum |
| NP-hard | Not necessarily in NP | Unbounded | Halting problem, optimal TSP |
| PSPACE | Yes | Poly space | QBF (TQBF), Generalized Chess |
| EXPTIME | Yes | O(2^(n^k)) | Go (generalized), complete info games |
| BPP | Randomized | O(n^k) w/ bounded error | Primality (Miller-Rabin) |

### Complexity Class Hierarchy (ASCII)

```
                    EXPTIME
                   /       \
                PSPACE
               /      \
            NP        co-NP
           /    \    /
          P      NP ∩ co-NP
          |
        L (log space)

  Known strict inclusions:
    P ⊆ NP ⊆ PSPACE ⊆ EXPTIME

  Open questions:
    P =? NP
    NP =? co-NP
    NP =? PSPACE

  By time/space hierarchy theorems:
    P ≠ EXPTIME  (strict)
    L ≠ PSPACE   (strict)
```

## Polynomial Reductions

### Definition

```
Language A reduces to language B (A ≤_p B) if there exists
a polynomial-time computable function f such that:

    x ∈ A  ⟺  f(x) ∈ B

Consequence: If B ∈ P, then A ∈ P
Contrapositive: If A ∉ P, then B ∉ P
```

### Reduction Chain (Karp Reductions)

```
SAT → 3-SAT → Clique → Vertex Cover → Hamiltonian Cycle → TSP

Each arrow means: left ≤_p right
If ANY of these is in P, they ALL are (and P = NP).
```

## Cook-Levin Theorem

### Statement

```
SAT is NP-complete.

That is:
  1. SAT ∈ NP              (a satisfying assignment is a poly-time certificate)
  2. For all L ∈ NP: L ≤_p SAT  (any NP computation can be encoded as a Boolean formula)
```

### Proof Sketch

```
Given: Nondeterministic TM M deciding language L in time p(n)

Construct Boolean formula φ encoding M's computation tableau:
  - Variables: x_{i,j,s} = "cell (i,j) contains symbol s"
                           where i = time step, j = tape position
  - Clauses enforce:
    1. Valid initial configuration (input encoded)
    2. Valid transitions (local consistency via transition function windows)
    3. Accepting state reached
    4. At most one symbol per cell per time step

Size of φ: O(p(n)^2 * |Γ|) — polynomial in n
Construction time: polynomial in n

Therefore: x ∈ L  ⟺  φ_x is satisfiable
```

### Independently Proven By

- **Stephen Cook** (1971) — "The Complexity of Theorem-Proving Procedures"
- **Leonid Levin** (1973) — Independent discovery in the Soviet Union

## Karp's 21 NP-Complete Problems (1972)

Richard Karp showed these are all NP-complete via reductions from SAT:

```
 1. SAT                    12. Chromatic Number
 2. 0-1 Integer Programming 13. Clique Cover
 3. Clique                  14. Exact Cover
 4. Set Packing             15. Hitting Set
 5. Vertex Cover            16. Steiner Tree
 6. Set Cover               17. 3-Dimensional Matching
 7. Feedback Node Set       18. Knapsack
 8. Feedback Arc Set        19. Job Sequencing
 9. Directed Hamiltonian    20. Partition
10. Undirected Hamiltonian  21. Max Cut
11. Graph Coloring
```

## Common NP-Complete Problems

### Quick Reference

| Problem | Input | Question |
|---|---|---|
| 3-SAT | CNF formula, 3 literals/clause | Satisfying assignment? |
| Clique | Graph G, integer k | Clique of size k? |
| Vertex Cover | Graph G, integer k | Cover all edges with k vertices? |
| Independent Set | Graph G, integer k | k pairwise non-adjacent vertices? |
| Hamiltonian Cycle | Graph G | Cycle visiting every vertex exactly once? |
| Subset Sum | Set S of integers, target t | Subset summing to t? |
| TSP (decision) | Weighted graph, bound B | Tour of cost ≤ B? |
| Graph Coloring | Graph G, integer k | Proper k-coloring? |
| Partition | Set of integers | Split into two equal-sum subsets? |

### Relationships

```
Clique(G, k) ⟺ Independent Set(complement(G), k)
Vertex Cover(G, k) ⟺ Independent Set(G, n-k)
3-SAT ≤_p Clique ≤_p Vertex Cover ≤_p Hamiltonian Cycle ≤_p TSP
```

## P vs NP — Millennium Prize Problem

```
Question: Does P = NP?

Stated formally:
  Is every language whose membership proofs can be verified
  in polynomial time also decidable in polynomial time?

Prize: $1,000,000 (Clay Mathematics Institute, 2000)
Status: OPEN

Implications of P = NP:
  - Cryptography collapses (RSA, Diffie-Hellman broken)
  - Optimization becomes trivial (scheduling, logistics, design)
  - Mathematical proofs become automatically discoverable
  - Machine learning: optimal models found in poly time

Implications of P ≠ NP:
  - Confirms inherent computational hardness
  - Cryptography rests on solid foundations
  - Approximation algorithms remain essential
  - Heuristics (SAT solvers, metaheuristics) justified

Consensus: Most complexity theorists believe P ≠ NP.
```

## Key Figures

| Name | Contribution | Year |
|---|---|---|
| Alan Cobham | Defined class P, Cobham's thesis (poly-time = feasible) | 1965 |
| Stephen Cook | Cook-Levin theorem (SAT is NP-complete) | 1971 |
| Richard Karp | 21 NP-complete problems via reductions | 1972 |
| Leonid Levin | Independent proof of NP-completeness (USSR) | 1973 |
| Larry Stockmeyer | Polynomial hierarchy (PH) | 1976 |
| Michael Sipser | Time hierarchy theorem contributions | 1978 |

## Tips

- To prove a problem is NP-complete: (1) show it is in NP, (2) reduce a known NP-complete problem TO it
- The reduction direction matters: reduce FROM the known hard problem TO your target
- NP-hard does not mean "in NP" — the halting problem is NP-hard but undecidable
- Many NP-complete problems have efficient approximation algorithms (e.g., 2-approx for Vertex Cover)
- SAT solvers (DPLL, CDCL) are remarkably fast in practice despite worst-case exponential time
- co-NP ≠ NP would imply P ≠ NP, but the converse is unknown
- PSPACE = NPSPACE by Savitch's theorem — nondeterminism does not help for space

## See Also

- `detail/cs-theory/complexity-theory.md` — Cook-Levin proof, reduction examples, randomized classes
- `sheets/cs-theory/automata-theory.md` — decidability, halting problem, Rice's theorem
- `sheets/cs-theory/complexity-theory.md` — approximation algorithms for NP-hard problems
- `sheets/cs-theory/information-theory.md` — entropy, Kolmogorov complexity

## References

- "Introduction to the Theory of Computation" by Michael Sipser (3rd ed., 2012)
- "Computational Complexity: A Modern Approach" by Arora and Barak (Cambridge, 2009)
- Cook, "The Complexity of Theorem-Proving Procedures" (STOC, 1971)
- Karp, "Reducibility Among Combinatorial Problems" (1972)
- Clay Mathematics Institute — P vs NP Problem Statement (2000)
