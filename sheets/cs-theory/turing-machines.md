# Turing Machines & Computability (Foundations of Computation)

A complete reference for Turing machines, decidability, and the theoretical limits of what can be computed — the bedrock of computer science.

## Turing Machine Definition

### The 7-Tuple

```
A Turing machine is a 7-tuple M = (Q, Sigma, Gamma, delta, q0, q_accept, q_reject)

  Q        — finite set of states
  Sigma    — input alphabet (does not include blank symbol)
  Gamma    — tape alphabet (Sigma is a subset of Gamma, includes blank _)
  delta    — transition function: Q x Gamma -> Q x Gamma x {L, R}
  q0       — start state (in Q)
  q_accept — accept state
  q_reject — reject state (q_reject != q_accept)
```

### Tape, Head, and Execution

```
Infinite tape (one direction or both):

  ... | _ | _ | a | b | b | a | _ | _ | ...
                    ^
                  head

Each step:
  1. Read symbol under head
  2. Look up (current_state, symbol) in delta
  3. Write new symbol, move head L or R, transition to new state
  4. Halt if state is q_accept or q_reject
```

### Transition Function Example

```
State  Read  Write  Move  Next State
-----  ----  -----  ----  ----------
q0     a     X      R     q1
q0     _     _      R     q_accept
q1     a     a      R     q1
q1     b     Y      L     q2
q2     a     a      L     q2
q2     X     X      R     q0
```

## Universal Turing Machine (UTM)

```
A UTM U takes as input:
  <M, w> = encoded description of TM M + input string w

U simulates M on w step by step:
  - Decodes M's transition table
  - Maintains simulated tape, head position, and state
  - Accepts iff M accepts, rejects iff M rejects

Key insight: One machine can simulate ANY other machine.
This is the theoretical basis for general-purpose computers.
```

## Church-Turing Thesis

```
"Any function that is effectively computable (by an algorithm)
 is computable by a Turing machine."

NOT a theorem — a thesis (cannot be formally proved).
Supported by equivalence of all known computational models:

  Turing machines  <=>  Lambda calculus (Church)
  Turing machines  <=>  Recursive functions (Godel)
  Turing machines  <=>  Post systems
  Turing machines  <=>  Register machines
  Turing machines  <=>  RAM model
  Turing machines  <=>  Modern programming languages
```

## The Halting Problem

### Statement

```
HALT = { <M, w> : M is a TM and M halts on input w }

Theorem: HALT is undecidable (no TM can decide it).
```

### Proof Sketch (Diagonalization)

```
Assume for contradiction that H decides HALT.
  H(<M, w>) = accept  if M halts on w
  H(<M, w>) = reject  if M loops on w

Construct D: on input <M>
  1. Run H(<M, <M>>)
  2. If H accepts (M halts on <M>), then LOOP forever
  3. If H rejects (M loops on <M>), then HALT and accept

Run D on <D>:
  - If D halts on <D>, then H says "halts", so D loops. Contradiction.
  - If D loops on <D>, then H says "loops", so D halts. Contradiction.

Therefore H cannot exist. HALT is undecidable.
```

## Decidable vs Recognizable Languages

```
Classification of languages:

  Decidable (Recursive)
  |  TM halts on ALL inputs (always accepts or rejects)
  |  Examples: { a^n b^n : n >= 0 }, every CFL, PRIME
  |
  Recognizable (Recursively Enumerable, RE)
  |  TM halts and accepts for strings IN the language
  |  May loop forever for strings NOT in the language
  |  Examples: HALT, A_TM = { <M,w> : M accepts w }
  |
  Co-Recognizable (co-RE)
  |  Complement is recognizable
  |
  Neither Recognizable nor Co-Recognizable
     Examples: EQ_TM = { <M1,M2> : L(M1) = L(M2) }

Key theorem:
  L is decidable  <=>  L is both recognizable AND co-recognizable
```

## Reductions

### Mapping Reductions

```
A <=m B  means "A reduces to B"

  There exists computable function f such that:
    w in A  <=>  f(w) in B

If A <=m B and B is decidable, then A is decidable.
If A <=m B and A is undecidable, then B is undecidable.

Common technique: reduce HALT or A_TM to a new problem
to prove the new problem is undecidable.
```

### Classic Undecidable Problems (via reduction from HALT)

```
A_TM   = { <M,w> : M accepts w }           — undecidable
HALT   = { <M,w> : M halts on w }          — undecidable
E_TM   = { <M>   : L(M) = empty }          — undecidable
EQ_TM  = { <M,N> : L(M) = L(N) }           — undecidable
ALL_TM = { <M>   : L(M) = Sigma* }         — undecidable (not even RE)
```

## Rice's Theorem

```
Rice's Theorem:
  Every non-trivial property of the language of a TM
  is undecidable.

"Non-trivial" = some TMs have the property, some do not.

Examples of undecidable questions about TM languages:
  - Does L(M) contain the empty string?
  - Is L(M) finite?
  - Is L(M) regular?
  - Is L(M) = Sigma*?
  - Does L(M) contain "hello"?

NOT covered by Rice's theorem (properties of the machine, not language):
  - Does M have exactly 5 states?
  - Does M halt within 100 steps on empty input?
```

## Multi-Tape Turing Machines

```
A k-tape TM has k independent tapes, each with its own head.

  Tape 1:  | a | b | a | _ | ...    (input tape)
                ^
  Tape 2:  | _ | X | Y | _ | ...    (work tape)
                    ^
  Tape 3:  | _ | _ | _ | _ | ...    (output tape)
            ^

Transition: delta: Q x Gamma^k -> Q x Gamma^k x {L,R,S}^k

Theorem: Every k-tape TM has an equivalent single-tape TM.
  Simulation cost: O(t^2) steps for t steps of multi-tape TM.
  Multi-tape does NOT increase computational power.
```

## Nondeterministic Turing Machines (NTM)

```
delta: Q x Gamma -> P(Q x Gamma x {L, R})

At each step, multiple transitions are possible.
NTM accepts if ANY branch of computation reaches q_accept.

  Computation tree:

              q0
             / | \
           q1  q2  q3
          / \     / \
        q4  q5  q6  q_accept  <-- NTM accepts

Theorem: Every NTM has an equivalent deterministic TM.
  Simulation: BFS over computation tree.
  Cost: exponential blowup in time, but same language class.

NTMs recognize exactly the same languages as deterministic TMs.
The question P =? NP asks about EFFICIENCY, not computability.
```

## Key Figures

```
Alan Turing (1912-1954)
  - Defined the Turing machine (1936)
  - Proved the halting problem undecidable
  - Designed the Universal Turing Machine
  - Bletchley Park codebreaker (Enigma)
  - "On Computable Numbers, with an Application to the Entscheidungsproblem"

Alonzo Church (1903-1995)
  - Created lambda calculus (1936)
  - Independently proved Entscheidungsproblem undecidable
  - Church-Turing thesis bears his name
  - PhD advisor to Turing at Princeton

Kurt Godel (1906-1978)
  - Incompleteness theorems (1931) — precursor to undecidability
  - Godel numbering — encoding formal systems as numbers
  - Showed no consistent formal system can prove all true statements
  - Recursive functions — one of several equivalent models
```

## Tips

- The halting problem is the canonical undecidable problem — most undecidability proofs reduce from it
- Rice's theorem is a powerful shortcut: if you are asked whether a property of TM languages is decidable and the property is non-trivial, the answer is no
- Multi-tape and nondeterministic TMs do not add computational power — they recognize the same class of languages
- Decidability is about the existence of an algorithm, not its efficiency — that is the domain of complexity theory
- The Church-Turing thesis is evidence-based, not proved — every known model of computation has been shown equivalent to TMs
- Diagonalization is the key proof technique: construct an object that differs from every item in a list along the diagonal
- A language is decidable if and only if both it and its complement are recognizable

## See Also

- `detail/cs-theory/turing-machines.md` — formal definitions, full halting problem proof, Godel connections
- `sheets/cs-theory/complexity-theory.md` — P, NP, NP-completeness, space complexity
- `sheets/cs-theory/automata-theory.md` — DFA, NFA, PDA, regular and context-free languages
- `sheets/cs-theory/automata-theory.md` — Chomsky hierarchy, grammars

## References

- "Introduction to the Theory of Computation" by Michael Sipser (Cengage, 3rd edition)
- Turing, A. M. "On Computable Numbers, with an Application to the Entscheidungsproblem" (1936)
- Church, A. "An Unsolvable Problem of Elementary Number Theory" (1936)
- Hopcroft, Motwani, Ullman. "Introduction to Automata Theory, Languages, and Computation" (Pearson, 3rd edition)
- "Computational Complexity: A Modern Approach" by Arora and Barak (Cambridge, 2009)
