# Database Theory (Relational Foundations, Normalization, and Transactions)

A practitioner's reference for the theoretical foundations of databases -- from the relational model and normalization to transactions, concurrency, recovery, and distributed consistency.

## The Relational Model

### Core Concepts

```
Relation:   A named set of tuples (rows) with a fixed schema
Tuple:      An ordered list of attribute values (a single row)
Attribute:  A named column with a declared domain
Domain:     The set of permissible values for an attribute
Schema:     R(A1: D1, A2: D2, ..., An: Dn)

A relation r(R) is a subset of D1 x D2 x ... x Dn

Key types:
  Superkey:      A set of attributes that uniquely identifies tuples
  Candidate key: A minimal superkey (no proper subset is a superkey)
  Primary key:   The chosen candidate key for the relation
  Foreign key:   Attributes referencing a primary key in another relation

Integrity constraints:
  Entity integrity:       Primary key attributes cannot be NULL
  Referential integrity:  Foreign key values must exist in the referenced relation
  Domain constraint:      Attribute values must belong to their declared domain
```

### Codd's 12 Rules (1985)

```
Rule 0:  Foundation — must use relational facilities exclusively for management
Rule 1:  Information — all data represented as values in tables
Rule 2:  Guaranteed access — every datum accessible by table + key + column
Rule 3:  Systematic NULL — NULLs for missing/inapplicable data (distinct from "")
Rule 4:  Active catalog — database description stored in relations (system tables)
Rule 5:  Comprehensive sublanguage — at least one language for DDL, DML, DCL, TCL
Rule 6:  View updating — all theoretically updatable views are updatable
Rule 7:  Set-level ops — insert, update, delete apply to sets, not just single rows
Rule 8:  Physical independence — apps unaffected by storage/access method changes
Rule 9:  Logical independence — apps unaffected by schema changes preserving info
Rule 10: Integrity independence — constraints defined in catalog, not in apps
Rule 11: Distribution independence — apps unaffected by data distribution changes
Rule 12: Nonsubversion — no low-level bypass of relational integrity constraints
```

## Relational Algebra

### Fundamental Operations

```
Selection (sigma):     sigma_{condition}(R)
  Returns tuples satisfying the condition
  Example: sigma_{age > 30}(Employee)

Projection (pi):       pi_{A1, A2, ...}(R)
  Returns specified attributes, eliminating duplicates
  Example: pi_{name, salary}(Employee)

Union:                 R union S
  Tuples in R or S (requires union-compatible schemas)

Set Difference:        R - S
  Tuples in R but not in S

Cartesian Product:     R x S
  All combinations of tuples from R and S

Rename (rho):          rho_{S(B1, B2, ...)}(R)
  Renames relation and/or attributes
```

### Derived Operations

```
Natural Join:          R |><| S
  Equi-join on all common attributes, removing duplicates
  R |><| S = pi_{...}(sigma_{R.A=S.A}(R x S))

Theta Join:            R |><|_{theta} S
  R x S filtered by condition theta

Semijoin:              R |><  S
  Tuples of R that join with some tuple in S
  R |><  S = pi_{attrs(R)}(R |><| S)

Antijoin:              R |>  S
  Tuples of R that do NOT join with any tuple in S

Outer Joins:
  Left:   R =|><| S   (preserves all tuples of R)
  Right:  R |><|= S   (preserves all tuples of S)
  Full:   R =|><|= S  (preserves all tuples of both)

Division:              R / S
  Tuples in pi_{R-S}(R) that are associated with every tuple in S
  "Which suppliers supply ALL parts?"
  R / S = pi_{R-S}(R) - pi_{R-S}((pi_{R-S}(R) x S) - R)

Aggregation:           G_{gamma} F(A)(R)
  Group by G, apply aggregate F (SUM, COUNT, AVG, MIN, MAX) to A
```

## Relational Calculus

### Tuple Relational Calculus (TRC)

```
{ t | P(t) }
  Set of tuples t satisfying predicate P

Example — employees earning > 50000:
  { t | Employee(t) AND t.salary > 50000 }

Example — names of employees in Sales:
  { t.name | Employee(t) AND t.dept = 'Sales' }

Quantifiers:
  Existential:  EXISTS t IN R (P(t))
  Universal:    FORALL t IN R (P(t))
```

### Domain Relational Calculus (DRC)

```
{ <x1, x2, ...> | P(x1, x2, ...) }
  Variables range over domains (values), not tuples

Example — names of employees earning > 50000:
  { <n> | EXISTS s (<n, s> IN Employee AND s > 50000) }
```

### Codd's Theorem

```
Relational Algebra = Safe Tuple Relational Calculus = Safe Domain Relational Calculus

A calculus expression is "safe" if it is guaranteed to produce a finite result.
SQL is based on TRC (SELECT-FROM-WHERE maps to {t | P(t)}).
```

## Normalization

### Functional Dependencies

```
X -> Y means: if two tuples agree on X, they agree on Y.

Trivial:     X -> Y where Y is a subset of X
Full:        X -> Y where no proper subset of X determines Y
Partial:     X -> Y where some proper subset of X determines Y
Transitive:  X -> Y -> Z (X -> Z via Y, Y does not determine X)
```

### Armstrong's Axioms (Sound and Complete)

```
Reflexivity:   If Y is a subset of X, then X -> Y
Augmentation:  If X -> Y, then XZ -> YZ
Transitivity:  If X -> Y and Y -> Z, then X -> Z

Derived rules:
  Union:          If X -> Y and X -> Z, then X -> YZ
  Decomposition:  If X -> YZ, then X -> Y and X -> Z
  Pseudotransitivity: If X -> Y and WY -> Z, then WX -> Z

Closure of X under F:  X+ = { A | X -> A is derivable from F }
  Algorithm: Start with X+ = X, repeatedly add A if some V -> A
             in F has V in X+, until stable.

Candidate keys: X is a candidate key iff X+ = all attributes
                and no proper subset of X has this property.
```

### Normal Forms

```
1NF:  All attributes are atomic (no repeating groups or nested relations)

2NF:  1NF + no partial dependencies on any candidate key
      (every non-prime attribute is fully dependent on every candidate key)

3NF:  2NF + no transitive dependencies
      For every X -> A: X is a superkey OR A is a prime attribute

BCNF: For every nontrivial X -> A: X is a superkey
      Strictly stronger than 3NF (eliminates all redundancy from FDs)
      Note: BCNF decomposition may not preserve all FDs

4NF:  BCNF + no nontrivial multivalued dependencies
      X ->> Y: for every pair of tuples agreeing on X,
      the Y-values can be "swapped" and still be in the relation

5NF:  4NF + no nontrivial join dependencies
      The relation cannot be losslessly decomposed further
      Also called Project-Join Normal Form (PJNF)

Decomposition properties:
  Lossless join:       R = pi_{R1}(R) |><| pi_{R2}(R) (no spurious tuples)
  Dependency preservation: All FDs checkable on individual decomposed relations
```

### Decomposition Summary

| Normal Form | Eliminates | Always Lossless | Always FD-Preserving |
|---|---|---|---|
| 2NF | Partial dependencies | Yes | Yes |
| 3NF | Transitive dependencies | Yes | Yes |
| BCNF | All FD-based redundancy | Yes | Not always |
| 4NF | Multivalued dependencies | Yes | N/A |
| 5NF | Join dependencies | Yes | N/A |

## ACID Properties

```
Atomicity:     All operations in a transaction succeed or none do
Consistency:   A transaction transforms the database from one valid state to another
Isolation:     Concurrent transactions appear to execute serially
Durability:    Once committed, changes survive any subsequent failure

Implementation:
  Atomicity   -> undo log (rollback on abort)
  Consistency -> constraints + application logic
  Isolation   -> concurrency control (locks, MVCC)
  Durability  -> redo log (WAL) + checkpointing
```

## Transaction Isolation Levels

```
Level                | Dirty Read | Non-repeatable Read | Phantom Read
---------------------|------------|---------------------|-------------
READ UNCOMMITTED     | Possible   | Possible            | Possible
READ COMMITTED       | Prevented  | Possible            | Possible
REPEATABLE READ      | Prevented  | Prevented           | Possible
SERIALIZABLE         | Prevented  | Prevented           | Prevented

Anomalies:
  Dirty read:           T1 reads data written by uncommitted T2
  Non-repeatable read:  T1 reads same row twice, gets different values
  Phantom read:         T1 re-executes a range query, gets new rows

Additional anomalies (Berenson et al., 1995):
  Write skew:           Two transactions read overlapping data and write
                        disjoint data, violating a constraint
  Lost update:          Two transactions read, then write the same item
```

## Concurrency Control

### Two-Phase Locking (2PL)

```
Growing phase:    Acquire locks, never release
Shrinking phase:  Release locks, never acquire

Guarantees conflict-serializability.

Variants:
  Basic 2PL:       Locks released after shrinking begins
  Strict 2PL:      Exclusive locks held until commit (prevents cascading aborts)
  Rigorous 2PL:    ALL locks held until commit

Lock types:        S (shared/read), X (exclusive/write)
Compatibility:     S-S = compatible, S-X = conflict, X-X = conflict

Deadlock handling:
  Prevention:   Wait-die, wound-wait (timestamp-based)
  Detection:    Waits-for graph (cycle = deadlock), victim selection
  Timeout:      Abort after waiting too long
```

### Multiversion Concurrency Control (MVCC)

```
Each write creates a new version of the data item.
Readers see a consistent snapshot without acquiring locks.

Implementations:
  PostgreSQL:  Tuple versioning with xmin/xmax transaction IDs
  MySQL/InnoDB: Undo log for old versions, read views
  Oracle:      Undo tablespace for consistent reads

Snapshot Isolation (SI):
  Each transaction reads from a snapshot taken at start time.
  Write-write conflicts detected at commit (first committer wins).
  SI prevents dirty reads, non-repeatable reads, and phantoms
  but allows write skew (not fully serializable).
```

### Other Methods

```
Timestamp Ordering (TO):
  Each transaction gets a timestamp. Operations on data items
  must follow timestamp order. Reject and restart if violated.
  Thomas's write rule: ignore "too late" writes instead of aborting.

Optimistic Concurrency Control (OCC):
  Three phases: Read (tentative), Validate (check conflicts), Write.
  Best when conflicts are rare.

Serializable Snapshot Isolation (SSI):
  Extends SI to detect read-write conflicts (dangerous structures).
  Used by PostgreSQL 9.1+ for true SERIALIZABLE isolation.
```

## Serializability

```
A schedule is serializable if its effect equals some serial schedule.

Conflict serializability:
  Two operations conflict if they access the same data item
  and at least one is a write.

  Precedence graph (conflict graph):
    Nodes = transactions
    Edge Ti -> Tj if Ti has a conflicting operation before Tj
    Schedule is conflict-serializable iff the graph is acyclic.

View serializability:
  Schedule S is view-equivalent to serial schedule S' if:
    1. Same initial reads (first read of each item)
    2. Same dependent reads (reads-from relationships)
    3. Same final writes (last write of each item)

  View serializability is strictly weaker than conflict serializability.
  Testing view serializability is NP-complete.
```

## Recovery

### Write-Ahead Logging (WAL)

```
WAL Protocol:
  1. Log record written to stable storage BEFORE the data page
  2. All log records for a transaction written before COMMIT
  3. Log records: <T, item, old_value, new_value>

Policies:
  STEAL:    Uncommitted changes can be flushed to disk (needs UNDO)
  NO-STEAL: Uncommitted changes never flushed (no UNDO needed)
  FORCE:    All changes flushed at commit (no REDO needed)
  NO-FORCE: Changes may not be flushed at commit (needs REDO)

  Most systems use STEAL/NO-FORCE (requires both UNDO and REDO).
```

### ARIES Recovery Algorithm

```
Three phases:
  1. Analysis:  Scan log forward from last checkpoint
                Determine dirty pages and active transactions
                Build dirty page table (DPT) and active transaction table (ATT)

  2. Redo:      Scan log forward from earliest recLSN in DPT
                Repeat history: redo ALL logged actions (including aborts)
                Brings database to exact pre-crash state

  3. Undo:      Scan log backward
                Undo all uncommitted transactions (those in ATT)
                Write CLRs (Compensation Log Records) for each undo

Key concepts:
  LSN:       Log Sequence Number (monotonically increasing)
  pageLSN:   LSN of last log record applied to page
  recLSN:    LSN of first log record that dirtied a page
  CLR:       Compensation Log Record (undo of an undo is a no-op)
  Fuzzy checkpoint: Checkpoint without halting transactions
```

## Query Optimization

### Cost Models

```
Cost factors:
  I/O cost:  Number of disk page reads/writes (dominant factor)
  CPU cost:  Comparison operations, hash computations
  Memory:    Available buffer pool pages

Statistics maintained:
  n_r:       Number of tuples in relation r
  b_r:       Number of disk blocks for relation r
  l_r:       Size of a tuple in bytes
  f_r:       Blocking factor (tuples per block)
  V(A, r):   Number of distinct values for attribute A in r
  Histograms: Equi-width, equi-depth, or compressed for value distribution
```

### Join Algorithms

| Algorithm | Cost (I/O) | Best When |
|---|---|---|
| Nested Loop | b_r * b_s | Small inner relation |
| Block Nested Loop | b_r * ceil(b_s / (M-2)) + b_s | Limited memory |
| Index Nested Loop | b_r * c per lookup | Index on join attr |
| Sort-Merge | sort(R) + sort(S) + merge | Both sortable, equality join |
| Hash Join | 3 * (b_r + b_s) | Equality join, enough memory |
| Grace Hash Join | 3 * (b_r + b_s) | Large relations, equality join |

### Query Plans

```
Query processing pipeline:
  SQL -> Parse -> Logical plan (relational algebra) -> Optimize -> Physical plan -> Execute

Logical optimization:
  Push selections down (reduce tuple count early)
  Push projections down (reduce tuple width early)
  Reorder joins (smallest intermediate results first)
  Eliminate redundant operations

Physical optimization:
  Choose access methods (sequential scan, index scan, index-only scan)
  Choose join algorithms based on cost estimates
  Choose sort vs hash for grouping/distinct
  Pipeline vs materialize intermediate results
```

## CAP Theorem and BASE

### CAP Theorem (Brewer, 2000; Gilbert & Lynch, 2002)

```
In a distributed data store, you can guarantee at most two of:
  Consistency:         Every read returns the most recent write
  Availability:        Every request receives a response
  Partition tolerance: System continues despite network partitions

Since network partitions are unavoidable, the real choice is:
  CP: Sacrifice availability during partitions (e.g., HBase, MongoDB)
  AP: Sacrifice consistency during partitions (e.g., Cassandra, DynamoDB)

PACELC (Abadi, 2012):
  If Partition -> choose A or C
  Else         -> choose Latency or Consistency
```

### BASE Semantics

```
Basically Available:  System guarantees availability (per CAP)
Soft state:           State may change over time without input (due to replication)
Eventually consistent: System converges to consistency given no new updates

Contrast with ACID:
  ACID: Strong consistency, pessimistic, single-node focus
  BASE: Weak consistency, optimistic, distributed focus
```

## Key Figures

| Name | Contribution | Year |
|---|---|---|
| Edgar F. Codd | Relational model, normalization theory, Codd's 12 rules | 1970 |
| Jim Gray | Transaction processing, ACID, isolation levels, 5-minute rule | 1976 |
| Michael Stonebraker | INGRES, Postgres, column stores, NewSQL (Turing Award 2014) | 1976 |
| C. J. Date | Relational theory, "An Introduction to Database Systems" | 1975 |
| Pat Helland | Idempotence, "Life Beyond Distributed Transactions" | 2007 |
| C. Mohan | ARIES recovery algorithm | 1992 |
| Raymond Boyce | BCNF (Boyce-Codd Normal Form), SQL co-designer | 1974 |
| Ronald Fagin | 4NF, 5NF, Fagin's theorem | 1977 |

## Tips

- Start normalization analysis by computing attribute closures to find all candidate keys before checking normal forms.
- BCNF decomposition is always lossless but may not preserve all functional dependencies; 3NF decomposition preserves both.
- Conflict serializability is the practical standard; view serializability is theoretically interesting but NP-complete to test.
- MVCC enables readers to never block writers, but watch for write skew under snapshot isolation.
- ARIES "repeats history" to simplify recovery logic; CLRs ensure idempotent undo.
- For join ordering with N relations, there are O(N!) orderings; dynamic programming (Selinger-style) makes this tractable.
- CAP is about partitions; if your system rarely partitions, focus on the latency-consistency tradeoff (PACELC).

## See Also

- complexity-theory
- graph-theory
- information-theory
- distributed-systems
- operating-systems

## References

- Codd, E. F. "A Relational Model of Data for Large Shared Data Banks" (1970), CACM
- Codd, E. F. "Is Your DBMS Really Relational?" and "Does Your DBMS Run By the Rules?" (1985), ComputerWorld
- Gray, J. & Reuter, A. "Transaction Processing: Concepts and Techniques" (Morgan Kaufmann, 1993)
- Ramakrishnan, R. & Gehrke, J. "Database Management Systems" (3rd ed., McGraw-Hill, 2003)
- Mohan, C. et al. "ARIES: A Transaction Recovery Method Supporting Fine-Granularity Locking" (1992), ACM TODS
- Bernstein, P. A. & Newcomer, E. "Principles of Transaction Processing" (2nd ed., Morgan Kaufmann, 2009)
- Garcia-Molina, H., Ullman, J. D. & Widom, J. "Database Systems: The Complete Book" (2nd ed., Pearson, 2008)
- Helland, P. "Life Beyond Distributed Transactions: An Apostate's Opinion" (2007), CIDR
- Gilbert, S. & Lynch, N. "Brewer's Conjecture and the Feasibility of Consistent, Available, Partition-Tolerant Web Services" (2002), SIGACT News
