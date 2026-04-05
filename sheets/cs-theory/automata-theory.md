# Automata Theory (Formal Languages, Grammars, and Computation)

A practitioner's reference for the hierarchy of formal languages — from finite automata and regular expressions to Turing machines and undecidability.

## The Chomsky Hierarchy

```
Type 0: Recursively Enumerable (Turing Machine)
  |
  |  -- unrestricted grammars, alpha -> beta
  |
Type 1: Context-Sensitive (Linear Bounded Automaton)
  |
  |  -- alpha A beta -> alpha gamma beta, |gamma| >= 1
  |
Type 2: Context-Free (Pushdown Automaton)
  |
  |  -- A -> gamma, single nonterminal on LHS
  |
Type 3: Regular (Finite Automaton)
       -- A -> aB or A -> a (right-linear)
```

| Type | Language Class | Grammar | Automaton | Closed Under |
|------|---------------|---------|-----------|-------------|
| 3 | Regular | Right-linear | DFA / NFA | Union, intersection, complement, concat, star |
| 2 | Context-Free | Context-free | PDA (nondeterministic) | Union, concat, star (NOT intersection, complement) |
| 1 | Context-Sensitive | Context-sensitive | LBA | Union, intersection, complement, concat, star |
| 0 | Recursively Enumerable | Unrestricted | Turing Machine | Union, intersection, concat, star (NOT complement) |

## Regular Languages and Finite Automata

### DFA (Deterministic Finite Automaton)

```
5-tuple: M = (Q, Sigma, delta, q0, F)

Q      = finite set of states
Sigma  = finite input alphabet
delta  = transition function  Q x Sigma -> Q
q0     = start state, q0 in Q
F      = set of accept states, F subset of Q
```

Example DFA accepting strings over {0,1} with an even number of 1s:

```
        0           0
     +-----+     +-----+
     |     v     |     v
---> (q0)---1--->(q1)
     accept       reject
      ^           |
      +-----1-----+
```

### NFA (Nondeterministic Finite Automaton)

```
5-tuple: M = (Q, Sigma, delta, q0, F)

delta: Q x (Sigma union {epsilon}) -> P(Q)    (power set of Q)
```

- NFAs allow epsilon-transitions and multiple transitions on the same symbol.
- Every NFA has an equivalent DFA (subset construction, worst case 2^n states).
- Every DFA is trivially an NFA.

### DFA Minimization

```
1. Remove unreachable states
2. Table-filling (Myhill-Nerode):
   - Mark all pairs (p, q) where p in F, q not in F (or vice versa)
   - Repeat: mark (p, q) if for some symbol a,
     (delta(p,a), delta(q,a)) is already marked
   - Unmarked pairs are equivalent -- merge them
3. Result is the unique minimal DFA
```

### Regular Expressions

```
Base cases:         epsilon, empty set, a (for a in Sigma)
Induction:          R1 | R2    (union)
                    R1 R2      (concatenation)
                    R*         (Kleene star)

Kleene's Theorem:   RE <=> DFA <=> NFA
```

### Pumping Lemma for Regular Languages

```
If L is regular, then there exists p >= 1 such that
for every w in L with |w| >= p,
w = xyz where:
  1. |y| > 0
  2. |xy| <= p
  3. for all i >= 0, xy^i z in L
```

Usage: proof by contradiction that a language is NOT regular.

Example: L = { a^n b^n | n >= 0 } is not regular.

```
Assume regular. Let p be the pumping length.
Take w = a^p b^p, |w| = 2p >= p.
Then w = xyz with |xy| <= p, so y = a^k for some k >= 1.
Pump down: xy^0 z = a^(p-k) b^p, but p-k != p. Contradiction.
```

## Context-Free Languages and Pushdown Automata

### Context-Free Grammar (CFG)

```
4-tuple: G = (V, Sigma, R, S)

V      = finite set of variables (nonterminals)
Sigma  = finite set of terminals
R      = finite set of rules  V -> (V union Sigma)*
S      = start variable, S in V
```

Normal forms:

```
Chomsky Normal Form (CNF):    A -> BC  or  A -> a  or  S -> epsilon
Greibach Normal Form (GNF):   A -> a alpha   (a terminal, alpha in V*)
```

### Pushdown Automaton (PDA)

```
6-tuple: M = (Q, Sigma, Gamma, delta, q0, F)

Gamma  = stack alphabet
delta  : Q x (Sigma union {epsilon}) x (Gamma union {epsilon})
            -> P(Q x (Gamma union {epsilon}))
```

- PDAs are nondeterministic (deterministic PDAs recognize a proper subset of CFLs).
- A language is context-free if and only if some PDA recognizes it.

### Pumping Lemma for Context-Free Languages

```
If L is context-free, then there exists p >= 1 such that
for every w in L with |w| >= p,
w = uvxyz where:
  1. |vy| > 0
  2. |vxy| <= p
  3. for all i >= 0, uv^i xy^i z in L
```

Example: L = { a^n b^n c^n | n >= 0 } is not context-free.

## Context-Sensitive Languages and LBAs

### Context-Sensitive Grammar

```
Productions: alpha A beta -> alpha gamma beta
where A is a nonterminal, alpha/beta/gamma are strings,
|gamma| >= 1 (non-contracting).
```

### Linear Bounded Automaton (LBA)

A Turing machine restricted to use only the tape cells holding the input. Recognizes exactly the context-sensitive languages.

Open problem: does DLBA = NLBA? (deterministic vs nondeterministic LBA equivalence)

## Turing Machines

```
7-tuple: M = (Q, Sigma, Gamma, delta, q0, q_accept, q_reject)

delta : Q x Gamma -> Q x Gamma x {L, R}

Sigma       = input alphabet (subset of Gamma, not including blank)
Gamma       = tape alphabet (includes blank symbol)
q_accept    != q_reject
```

Key results:

| Problem | Status |
|---------|--------|
| Acceptance (A_TM) | Recognizable, undecidable |
| Halting (HALT_TM) | Recognizable, undecidable |
| Emptiness (E_TM) | Co-recognizable, undecidable |
| Equivalence (EQ_TM) | Neither recognizable nor co-recognizable |
| A_DFA, E_DFA, EQ_DFA | Decidable |
| A_CFG, E_CFG | Decidable |
| EQ_CFG | Undecidable |

## Key Figures

| Name | Contribution |
|------|-------------|
| Noam Chomsky | Chomsky hierarchy of formal grammars (1956) |
| Michael Rabin | NFA and nondeterminism, Rabin-Scott theorem (1959) |
| Dana Scott | DFA-NFA equivalence with Rabin, domain theory |
| Stephen Kleene | Regular expressions, Kleene's theorem, Kleene star (1956) |
| Alan Turing | Turing machine, halting problem, Church-Turing thesis (1936) |
| Emil Post | Post correspondence problem, production systems |

## Tips

- To prove a language is NOT regular: use the pumping lemma or Myhill-Nerode.
- To prove a language is NOT context-free: use the CFL pumping lemma or Ogden's lemma.
- Closure properties are your friend: if L1 intersect L2 would need to be regular but cannot be, L1 is not regular.
- DFA minimization always produces the unique minimal automaton for a regular language.
- Every regular language is context-free, but not vice versa. The hierarchy is strict.

## See Also

- Computational Complexity
- Decidability and Computability
- Formal Verification
- Compiler Design (lexing and parsing phases)

## References

- Sipser, M. *Introduction to the Theory of Computation*, 3rd ed. (2012)
- Hopcroft, J., Motwani, R., Ullman, J. *Introduction to Automata Theory, Languages, and Computation*, 3rd ed. (2006)
- Chomsky, N. "Three models for the description of language." IRE Trans. on Information Theory (1956)
- Rabin, M. & Scott, D. "Finite automata and their decision problems." IBM J. Research (1959)
- Kleene, S. "Representation of events in nerve nets and finite automata." Automata Studies (1956)
