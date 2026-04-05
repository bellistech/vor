# Boolean Algebra and Logic

A practitioner's reference for Boolean algebra, propositional and predicate logic, canonical forms, logic gates, and the satisfiability problem.

## Boolean Algebra Axioms

### Huntington Axioms

```
Let (B, +, *, ', 0, 1) be a Boolean algebra:

Identity:        a + 0 = a              a * 1 = a
Complement:      a + a' = 1             a * a' = 0
Commutativity:   a + b = b + a          a * b = b * a
Associativity:   (a+b)+c = a+(b+c)      (a*b)*c = a*(b*c)
Distributivity:  a*(b+c) = a*b + a*c    a+(b*c) = (a+b)*(a+c)

Derived laws:
Idempotency:     a + a = a              a * a = a
Domination:      a + 1 = 1             a * 0 = 0
Involution:      (a')' = a
Absorption:      a + a*b = a            a * (a+b) = a
Consensus:       a*b + a'*c + b*c = a*b + a'*c
```

### De Morgan's Laws

```
(a + b)' = a' * b'
(a * b)' = a' + b'

Generalized:
(x_1 + x_2 + ... + x_n)' = x_1' * x_2' * ... * x_n'
(x_1 * x_2 * ... * x_n)' = x_1' + x_2' + ... + x_n'
```

### Duality Principle

```
Every Boolean identity remains valid when:
  + <-> *    and    0 <-> 1

Example:
  a + 0 = a   (identity for +)
  a * 1 = a   (identity for *, the dual)
```

## Truth Tables

### Basic Operations

```
AND (conjunction):       OR (disjunction):        NOT (negation):
 A | B | A*B              A | B | A+B              A | A'
 0 | 0 |  0               0 | 0 |  0              0 |  1
 0 | 1 |  0               0 | 1 |  1              1 |  0
 1 | 0 |  0               1 | 0 |  1
 1 | 1 |  1               1 | 1 |  1

XOR (exclusive or):      NAND:                    XNOR (equivalence):
 A | B | A^B              A | B | (A*B)'           A | B | A<->B
 0 | 0 |  0               0 | 0 |  1               0 | 0 |  1
 0 | 1 |  1               0 | 1 |  1               0 | 1 |  0
 1 | 0 |  1               1 | 0 |  1               1 | 0 |  0
 1 | 1 |  0               1 | 1 |  0               1 | 1 |  1
```

### Number of Boolean Functions

```
For n variables: 2^(2^n) distinct Boolean functions

n=1:  4 functions    (identity, complement, constant 0, constant 1)
n=2:  16 functions   (AND, OR, XOR, NAND, NOR, XNOR, implication, ...)
n=3:  256 functions
n=4:  65536 functions
```

## Canonical Forms

### Sum of Products (SOP / DNF)

```
Disjunctive Normal Form: OR of AND terms (minterms)

Each minterm includes every variable (complemented or not).

Example for f(A,B,C) = 1 when ABC in {001, 011, 111}:
  f = A'B'C + A'BC + ABC
  f = sum(m1, m3, m7)
  f = Sum m(1, 3, 7)

Minterm numbering: binary encoding of variable values
  m0 = A'B'C', m1 = A'B'C, ..., m7 = ABC
```

### Product of Sums (POS / CNF)

```
Conjunctive Normal Form: AND of OR terms (maxterms)

Each maxterm includes every variable (complemented or not).

Example for f(A,B,C) = 0 when ABC in {000, 010, 100, 101, 110}:
  f = (A+B+C)(A+B'+C)(A'+B+C)(A'+B+C')(A'+B'+C)
  f = Prod M(0, 2, 4, 5, 6)

Maxterm M_i is the complement of minterm m_i:
  M_i = m_i'
```

### Conversion Between Forms

```
f = Sum m(1, 3, 7)   over 3 variables (8 minterms total)
f = Prod M(0, 2, 4, 5, 6)

Rule: the minterm indices and maxterm indices are complementary sets.
```

## Karnaugh Maps

### 2-Variable K-Map

```
        B=0   B=1
  A=0 | m0  | m1  |
  A=1 | m2  | m3  |

Group adjacent 1-cells (differ by one variable) to simplify.
```

### 3-Variable K-Map

```
          BC=00  BC=01  BC=11  BC=10
  A=0   |  m0  |  m1  |  m3  |  m2  |
  A=1   |  m4  |  m5  |  m7  |  m6  |

Note: columns use Gray code order (00, 01, 11, 10).
```

### 4-Variable K-Map

```
            CD=00  CD=01  CD=11  CD=10
  AB=00   |  m0  |  m1  |  m3  |  m2  |
  AB=01   |  m4  |  m5  |  m7  |  m6  |
  AB=11   | m12  | m13  | m15  | m14  |
  AB=10   |  m8  |  m9  | m11  | m10  |

Grouping rules:
  - Groups must be rectangular, sizes are powers of 2 (1, 2, 4, 8, 16)
  - Map wraps around (top-bottom, left-right)
  - Each group eliminates variables that change within the group
  - Larger groups yield simpler terms
  - Every 1-cell must be covered; don't-cares (X) can be included optionally
```

## Logic Gates

### Gate Summary

```
Gate     | Symbol | Function    | Transistors (CMOS)
---------|--------|-------------|-------------------
NOT      |  >o    | A'          | 2
AND      |  D     | A * B       | 6
OR       |  )>    | A + B       | 6
NAND     |  D>o   | (A * B)'    | 4
NOR      |  )>o   | (A + B)'    | 4
XOR      |  =)>   | A ^ B       | 8-12
XNOR     |  =)>o  | (A ^ B)'   | 8-12
Buffer   |  >     | A           | 2
```

### Functional Completeness

```
A set of gates is functionally complete if any Boolean function
can be expressed using only gates from that set.

Complete sets:
  {AND, OR, NOT}     -- standard basis
  {AND, NOT}         -- since A+B = (A'*B')'
  {OR, NOT}          -- since A*B = (A'+B')'
  {NAND}             -- alone is complete (universal gate)
  {NOR}              -- alone is complete (universal gate)

NAND constructions:
  NOT A    = A NAND A
  A AND B  = (A NAND B) NAND (A NAND B)
  A OR B   = (A NAND A) NAND (B NAND B)
```

## Propositional Logic

### Syntax and Connectives

```
Connectives (precedence high to low):
  1. NOT      (negation)         ~p, !p, -p
  2. AND      (conjunction)      p /\ q, p & q
  3. OR       (disjunction)      p \/ q, p | q
  4. ->       (implication)      p -> q
  5. <->      (biconditional)    p <-> q

Implication truth table:
  p -> q  is equivalent to  ~p \/ q

  p | q | p -> q
  T | T |   T
  T | F |   F
  F | T |   T      (vacuously true)
  F | F |   T      (vacuously true)
```

### Key Tautologies

```
Law of excluded middle:       p \/ ~p
Double negation:              ~~p <-> p
Contrapositive:               (p -> q) <-> (~q -> ~p)
Material implication:         (p -> q) <-> (~p \/ q)
Exportation:                  ((p /\ q) -> r) <-> (p -> (q -> r))
Modus ponens:                 ((p -> q) /\ p) -> q
Modus tollens:                ((p -> q) /\ ~q) -> ~p
Hypothetical syllogism:       ((p -> q) /\ (q -> r)) -> (p -> r)
Disjunctive syllogism:        ((p \/ q) /\ ~p) -> q
```

### Satisfiability and Validity

```
Satisfiable:    true under at least one assignment
Unsatisfiable:  false under all assignments (contradiction)
Valid:          true under all assignments (tautology)

Relationships:
  - phi is valid  iff  ~phi is unsatisfiable
  - phi is satisfiable  iff  ~phi is not valid
```

## Predicate Logic (First-Order Logic)

### Quantifiers

```
Universal:    forall x. P(x)    "for all x, P(x) holds"
Existential:  exists x. P(x)   "there exists an x such that P(x)"

Negation of quantifiers (De Morgan for quantifiers):
  ~(forall x. P(x))  <->  exists x. ~P(x)
  ~(exists x. P(x))  <->  forall x. ~P(x)

Quantifier order matters:
  forall x. exists y. Loves(x, y)   -- everyone loves someone
  exists y. forall x. Loves(x, y)   -- someone is loved by everyone
```

### Free and Bound Variables

```
In  forall x. (P(x) /\ Q(x, y)):
  x is bound (captured by forall)
  y is free

A sentence (closed formula) has no free variables.
Substitution: P(x)[t/x] replaces free occurrences of x with t.
Capture-avoiding substitution required when t contains bound variables.
```

## Natural Deduction

### Core Rules

```
Introduction rules:          Elimination rules:

/\-intro:  A    B            /\-elim:   A /\ B     A /\ B
           ------                       ------      ------
           A /\ B                         A            B

\/-intro:    A       B       \/-elim:   A \/ B   [A]..C   [B]..C
           -----  -----                 --------------------------
           A\/B   A\/B                            C

->-intro:  [A]               ->-elim (MP):  A -> B    A
            :                               ----------
            B                                   B
           -----
           A -> B

~-intro:   [A]               ~-elim:   ~~A
            :                          ----
           bot                           A
           -----
            ~A
```

## Resolution

### Clausal Form and Resolution Rule

```
Resolution operates on clauses (disjunctions of literals).

Resolution rule:
  {A, p}   {B, ~p}
  -----------------
      {A, B}

where A, B are (possibly empty) sets of literals.

Example:
  {p, q}   {~p, r}
  -----------------
       {q, r}

To prove phi is unsatisfiable:
  1. Convert phi to CNF
  2. Apply resolution repeatedly
  3. If the empty clause {} is derived, phi is unsatisfiable
```

### Completeness

```
Resolution is refutation-complete for propositional logic:
  If a set of clauses is unsatisfiable, resolution will derive
  the empty clause in finitely many steps.

Note: Resolution is NOT complete for proving validity directly.
It proves validity of phi by refuting ~phi.
```

## SAT Problem

### Definition and Complexity

```
SAT (Boolean Satisfiability):
  Given a Boolean formula phi, is there an assignment that makes phi true?

Cook-Levin Theorem (Cook 1971, Levin 1973):
  SAT is NP-complete.
  - SAT is in NP (a satisfying assignment is a polynomial-time certificate)
  - Every problem in NP is polynomial-time reducible to SAT

Variants:
  k-SAT:  each clause has exactly k literals
  2-SAT:  polynomial time (O(n + m) via implication graphs)
  3-SAT:  NP-complete (even the restricted case)
  Horn-SAT: polynomial time (unit propagation)
  MAX-SAT: optimization version (NP-hard)
```

### DPLL Algorithm

```
Davis-Putnam-Logemann-Loveland (1962):
  Backtracking search for SAT

  DPLL(clauses, assignment):
    1. Unit propagation: if a clause has one literal, assign it true
    2. Pure literal elimination: if a literal appears only positive
       (or only negative), assign it to satisfy those clauses
    3. If all clauses satisfied, return SAT
    4. If any clause is empty, return UNSAT (backtrack)
    5. Choose a variable x, branch:
       - Try x = true:  DPLL(simplified clauses, assignment + {x=T})
       - Try x = false: DPLL(simplified clauses, assignment + {x=F})

Modern SAT solvers (CDCL) add:
  - Conflict-driven clause learning
  - Non-chronological backtracking
  - VSIDS variable selection heuristic
  - Watched literals for efficient propagation
```

## Binary Decision Diagrams (BDDs)

### Ordered BDDs (OBDDs)

```
A BDD represents a Boolean function as a rooted DAG:
  - Two terminal nodes: 0 and 1
  - Each internal node is labeled with a variable x_i
  - Each internal node has two children: low (x_i=0) and high (x_i=1)
  - Variables appear in a fixed order along every path

Reduced OBDD (ROBDD):
  - No redundant nodes (low child = high child)
  - No isomorphic subgraphs (shared via hash consing)
  - Canonical: two functions are equal iff their ROBDDs are identical

Operations:
  Apply(op, f, g)    O(|f| * |g|)    -- AND, OR, XOR of two BDDs
  Restrict(f, x=v)   O(|f|)          -- cofactor / Shannon expansion
  Exists(x, f)        O(|f|^2)        -- quantification
  SatCount(f)         O(|f|)          -- count satisfying assignments

Variable ordering is critical: can cause exponential blowup.
  - Multiplication: exponential for all orderings
  - Addition: polynomial with interleaved ordering
```

## Key Figures

| Name | Contribution | Year |
|---|---|---|
| George Boole | Founded Boolean algebra ("The Laws of Thought") | 1854 |
| Augustus De Morgan | De Morgan's laws, formal logic | 1847 |
| Claude Shannon | Boolean algebra for switching circuits (master's thesis) | 1937 |
| Stephen Cook | Cook-Levin theorem (SAT is NP-complete) | 1971 |
| Leonid Levin | Independent proof of NP-completeness | 1973 |
| Martin Davis, Hilary Putnam | DP algorithm for SAT | 1960 |
| Randal Bryant | Reduced ordered BDDs (ROBDDs) | 1986 |
| Willard Quine, Edward McCluskey | Quine-McCluskey minimization | 1956 |

## Tips

- NAND and NOR are universal gates; real circuits are often built entirely from NAND (cheaper in CMOS).
- K-maps work well for up to 4-5 variables; beyond that, use Quine-McCluskey or Espresso.
- 2-SAT is in P; 3-SAT is NP-complete. The jump from 2 to 3 is a fundamental complexity threshold.
- Variable ordering in BDDs matters enormously; a bad ordering can make an ROBDD exponentially larger.
- De Morgan's laws are the workhorse of circuit optimization: push negations inward to convert between AND-OR and NAND-NOR forms.
- In propositional logic, implication p -> q is false only when p is true and q is false. Vacuous truth trips up beginners.
- Modern SAT solvers (MiniSat, CaDiCaL, Kissat) solve industrial instances with millions of variables using CDCL.

## See Also

- complexity-theory
- automata-theory
- information-theory
- lambda-calculus
- turing-machines

## References

- Boole, G. "An Investigation of the Laws of Thought" (1854)
- Shannon, C. E. "A Symbolic Analysis of Relay and Switching Circuits" (1937), master's thesis, MIT
- Cook, S. A. "The Complexity of Theorem-Proving Procedures" (1971), STOC
- Bryant, R. E. "Graph-Based Algorithms for Boolean Function Manipulation" (1986), IEEE Trans. Computers
- Enderton, H. B. "A Mathematical Introduction to Logic" (2nd ed., Academic Press, 2001)
- Biere, A. et al. "Handbook of Satisfiability" (2nd ed., IOS Press, 2021)
- Knuth, D. E. "The Art of Computer Programming, Vol. 4A: Combinatorial Algorithms" (Addison-Wesley, 2011)
