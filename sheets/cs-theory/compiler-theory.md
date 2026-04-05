# Compiler Theory (Lexing, Parsing, Optimization, and Code Generation)

A practitioner's reference for compiler construction -- from source text to machine code, covering every major phase of translation.

## Compilation Phases

```
Source Code
    |
    v
[1. Lexical Analysis]  -- tokens (via DFA / maximal munch)
    |
    v
[2. Syntax Analysis]   -- parse tree / AST (via LL, LR, etc.)
    |
    v
[3. Semantic Analysis]  -- type checking, symbol tables, scope resolution
    |
    v
[4. IR Generation]     -- three-address code, SSA form
    |
    v
[5. Optimization]      -- constant folding, DCE, LICM, ...
    |
    v
[6. Code Generation]   -- instruction selection, register allocation, scheduling
    |
    v
Machine Code / Assembly
```

| Phase | Input | Output | Key Formalism |
|-------|-------|--------|---------------|
| Lexical Analysis | Character stream | Token stream | Regular expressions, DFA |
| Syntax Analysis | Token stream | AST / Parse tree | Context-free grammars |
| Semantic Analysis | AST | Annotated AST | Attribute grammars, type systems |
| IR Generation | Annotated AST | IR (TAC, SSA) | Three-address code |
| Optimization | IR | Optimized IR | Data flow analysis, lattices |
| Code Generation | Optimized IR | Machine code | Graph coloring, tree matching |

## Lexical Analysis

### Regular Expressions to DFA

```
RE -> NFA (Thompson's construction)
NFA -> DFA (subset construction)
DFA -> minimized DFA (Hopcroft's algorithm)
```

### Token Specification

```
Keyword:     if | else | while | return | ...
Identifier:  [a-zA-Z_][a-zA-Z0-9_]*
Integer:     [0-9]+
Float:       [0-9]+\.[0-9]+ ([eE][+-]?[0-9]+)?
String:      "([^"\\]|\\.)*"
Operator:    \+ | \- | \* | / | == | != | <= | >= | ...
Whitespace:  [ \t\n\r]+   (skip)
Comment:     //[^\n]*      (skip)
```

### Maximal Munch Rule

```
Always match the longest possible token.

Input: "iffy"
  NOT:  keyword "if" + identifier "fy"
  YES:  identifier "iffy"

Input: "3.14"
  NOT:  integer "3" + dot "." + integer "14"
  YES:  float "3.14"
```

## Parsing

### Grammar Classification

| Parser | Grammar Class | Direction | Lookahead | Complexity |
|--------|--------------|-----------|-----------|------------|
| Recursive Descent | LL(1) | Top-down | 1 token | O(n) |
| LL(k) | LL(k) | Top-down | k tokens | O(n) |
| LR(0) | LR(0) | Bottom-up | 0 tokens | O(n) |
| SLR(1) | SLR(1) | Bottom-up | 1 token | O(n) |
| LALR(1) | LALR(1) | Bottom-up | 1 token | O(n) |
| LR(1) | LR(1) | Bottom-up | 1 token | O(n) |
| Earley | Any CFG | Chart | - | O(n^3) general, O(n) for LR |
| GLR | Any CFG | Bottom-up | - | O(n^3) worst case |

### FIRST and FOLLOW Sets

```
FIRST(alpha) = set of terminals that can begin strings derived from alpha
  - If alpha =>* epsilon, then epsilon in FIRST(alpha)

FOLLOW(A) = set of terminals that can appear immediately after A
  - If A is the start symbol, $ in FOLLOW(A)

Rules for FIRST:
  FIRST(terminal)   = { terminal }
  FIRST(epsilon)    = { epsilon }
  FIRST(A -> Y1 Y2 ... Yk):
    Add FIRST(Y1) - {epsilon}
    If epsilon in FIRST(Y1), add FIRST(Y2) - {epsilon}
    ...continue until Yi does not derive epsilon

Rules for FOLLOW:
  For each production A -> alpha B beta:
    Add FIRST(beta) - {epsilon} to FOLLOW(B)
    If epsilon in FIRST(beta) or beta is empty:
      Add FOLLOW(A) to FOLLOW(B)
```

### LL(1) Parse Table Construction

```
For each production A -> alpha:
  For each terminal a in FIRST(alpha):
    M[A, a] = A -> alpha
  If epsilon in FIRST(alpha):
    For each terminal b in FOLLOW(A):
      M[A, b] = A -> alpha
```

### LR Parsing (Shift-Reduce)

```
Stack-based parsing:
  SHIFT:  push next input token onto stack
  REDUCE: pop RHS of production, push LHS
  ACCEPT: input consumed, start symbol on stack
  ERROR:  no valid action

LR(0) item: A -> alpha . beta
  dot position marks how much of the RHS has been seen

LALR(1): merge LR(1) states with same core (LR(0) items)
  Same power as SLR in practice, fewer states than LR(1)
```

### Recursive Descent

```
// For grammar rule:  E -> T (('+' | '-') T)*
func parseE() AST {
    node := parseT()
    for tok == PLUS || tok == MINUS {
        op := tok
        advance()
        right := parseT()
        node = BinaryOp(op, node, right)
    }
    return node
}
```

### Operator Precedence (Pratt Parsing)

```
Precedence levels (low to high):
  1: = (assignment, right-assoc)
  2: || (logical or)
  3: && (logical and)
  4: == != (equality)
  5: < > <= >= (comparison)
  6: + - (additive)
  7: * / % (multiplicative)
  8: ! - (unary prefix)
  9: () [] . -> (postfix / call / member)
```

## Abstract Syntax Trees

```
Source: x = a + b * c

Parse Tree (CST):              AST:
  Stmt                         Assign
  /  |  \                      /    \
 x   =   Expr                 x     Add
         / | \                       / \
       Expr + Term                  a  Mul
        |     / | \                    / \
        a    b  *  c                  b   c

CST: preserves all grammar structure (parentheses, semicolons)
AST: retains only semantic content (operators, operands, structure)
```

## Semantic Analysis

### Symbol Tables

```
Scope: nested hash tables (stack of scopes)
  Global -> Function -> Block -> ...

Lookup: search from innermost scope outward
Insert: always into current (innermost) scope

struct Symbol {
    name     string
    type     Type
    scope    int
    offset   int    // stack offset or register
    mutable  bool
}
```

### Type Checking

```
Type rules (judgments):
  Gamma |- e : T    ("in environment Gamma, expression e has type T")

  Gamma |- n : int                           (integer literal)
  Gamma |- true : bool                       (boolean literal)
  Gamma(x) = T => Gamma |- x : T            (variable lookup)

  Gamma |- e1 : int, Gamma |- e2 : int
  ----------------------------------         (arithmetic)
  Gamma |- e1 + e2 : int

  Gamma |- e1 : T, Gamma |- e2 : T
  ----------------------------------         (comparison)
  Gamma |- e1 == e2 : bool

  Gamma |- e : T1, T1 <: T2
  --------------------------                 (subsumption)
  Gamma |- e : T2
```

## Intermediate Representations

### Three-Address Code (TAC)

```
Source: a = b * c + d * e

TAC:
  t1 = b * c
  t2 = d * e
  t3 = t1 + t2
  a  = t3

Forms: x = y op z | x = op y | x = y | goto L
       if x relop y goto L | param x | call p, n | return x
```

### Static Single Assignment (SSA)

```
Original:              SSA form:
  x = 1                  x1 = 1
  x = 2                  x2 = 2
  y = x                  y1 = x2

At control-flow merge points, insert phi-functions:

  if (cond)              if (cond)
    x = 1                  x1 = 1
  else                   else
    x = 2                  x2 = 2
  y = x                  x3 = phi(x1, x2)
                          y1 = x3

Phi-functions placed at dominance frontiers.
```

## Optimization

### Constant Folding

```
Before:          After:
  x = 3 + 5       x = 8
  y = x * 2       y = 16
```

### Dead Code Elimination

```
Before:              After:
  x = a + b           y = c + d     // x removed (never used)
  y = c + d
```

### Common Subexpression Elimination (CSE)

```
Before:              After:
  t1 = a + b           t1 = a + b
  t2 = a + b           t2 = t1       // reuse t1
```

### Loop Invariant Code Motion (LICM)

```
Before:                    After:
  for i in 0..n:             t = x * y     // hoisted
    a[i] = x * y + i         for i in 0..n:
                                a[i] = t + i
```

### Strength Reduction

```
Before:              After:
  i * 4                i << 2
  i * 7                (i << 3) - i
```

### Loop Unrolling

```
Before:                    After:
  for i in 0..4:             body(0)
    body(i)                  body(1)
                             body(2)
                             body(3)
```

## Code Generation

### Instruction Selection (Tree Matching)

```
IR tree:         Possible instructions:
   +              ADD  r1, r2, r3
  / \             ADDI r1, r2, #imm
 r1  *            MUL  r1, r2, r3
    / \           MADD r1, r2, r3, r4  (fused multiply-add)
   r2  r3

Goal: tile the IR tree with machine instruction patterns
      minimizing cost (cycles, code size).
```

### Register Allocation via Graph Coloring

```
1. Build interference graph:
   - Node = virtual register (live range)
   - Edge = two registers live at the same point

2. Color the graph with K colors (K = physical registers):
   - Simplify: remove nodes with degree < K (push to stack)
   - Spill: if all nodes have degree >= K, pick one to spill
   - Select: pop stack, assign colors (registers)

If a node cannot be colored -> spill to memory (load/store).
```

### Instruction Scheduling

```
Goal: reorder instructions to minimize pipeline stalls

Dependency graph:
  - Data dependencies (RAW, WAR, WAW)
  - Control dependencies
  - Memory dependencies

List scheduling:
  1. Build dependency DAG
  2. Compute priorities (critical path length)
  3. Greedily schedule highest-priority ready instruction
```

## Key Figures

| Person | Contribution |
|--------|-------------|
| Alfred Aho | Co-author of the Dragon Book, lex/yacc foundations |
| Jeffrey Ullman | Co-author of Dragon Book, formal language theory |
| Ravi Sethi | Co-author of Dragon Book, compiler optimization |
| Andrew Appel | Modern Compiler Implementation series, SSA form |
| Donald Knuth | Invented LR parsing (1965), attribute grammars |
| Noam Chomsky | Chomsky hierarchy of formal grammars |
| Stephen Kleene | Regular expressions, Kleene's theorem |
| John Cocke | Pioneered optimizing compilation, RISC |
| Frances Allen | Control flow analysis, optimization foundations |
| Gregory Chaitin | Graph coloring register allocation |
| Vaughan Pratt | Pratt parsing (top-down operator precedence) |

## See Also

- automata-theory
- complexity-theory
- type-theory
- lambda-calculus

## References

- Aho, Lam, Sethi, Ullman. *Compilers: Principles, Techniques, and Tools* (Dragon Book), 2nd ed. Pearson, 2006.
- Appel. *Modern Compiler Implementation in ML/Java/C*. Cambridge University Press, 1998.
- Cooper, Torczon. *Engineering a Compiler*, 3rd ed. Morgan Kaufmann, 2022.
- Muchnick. *Advanced Compiler Design and Implementation*. Morgan Kaufmann, 1997.
- Knuth. "On the Translation of Languages from Left to Right." *Information and Control* 8(6), 1965.
- Cytron et al. "Efficiently Computing Static Single Assignment Form and the Control Dependence Graph." *TOPLAS* 13(4), 1991.
- Chaitin. "Register Allocation and Spilling via Graph Coloring." *SIGPLAN Notices* 17(6), 1982.
