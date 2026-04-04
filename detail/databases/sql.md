# The Mathematics of SQL — Query Processing and Relational Algebra

> *SQL is grounded in relational algebra and set theory. The math covers join algorithms, cardinality estimation, index selectivity, query plan costs, and normalization theory.*

---

## 1. Relational Algebra — The Foundation

### Core Operations and Complexity

| Operation | SQL | Algebra | Time Complexity |
|:---|:---|:---|:---:|
| Selection | `WHERE` | $\sigma_{predicate}(R)$ | O(n) scan, O(log n) index |
| Projection | `SELECT cols` | $\pi_{cols}(R)$ | O(n) |
| Cross Product | `CROSS JOIN` | $R \times S$ | O(n × m) |
| Equi-Join | `JOIN ON` | $R \bowtie_{cond} S$ | O(n × m) to O(n + m) |
| Union | `UNION` | $R \cup S$ | O(n + m) |
| Intersection | `INTERSECT` | $R \cap S$ | O(n + m) |
| Difference | `EXCEPT` | $R - S$ | O(n + m) |

### Cardinality of Operations

$$|R \times S| = |R| \times |S|$$

$$|\sigma_{A=v}(R)| = \frac{|R|}{V(A, R)} \quad (\text{uniform distribution assumption})$$

Where $V(A, R)$ = number of distinct values of attribute $A$ in $R$.

$$|R \bowtie_{A=B} S| = \frac{|R| \times |S|}{\max(V(A,R), V(B,S))}$$

### Worked Example

*"Orders table (1M rows, 100K distinct customer_ids) JOIN Customers table (100K rows)."*

$$|O \bowtie C| = \frac{1,000,000 \times 100,000}{\max(100,000, 100,000)} = 1,000,000$$

Makes sense: each order matches exactly one customer.

---

## 2. Join Algorithm Costs

### Nested Loop Join

$$\text{Cost}_{NL} = |R| \times |S| \quad (\text{naive, no index})$$

$$\text{Cost}_{NL+index} = |R| \times (T_{index\_lookup}) = |R| \times O(\log |S|)$$

### Block Nested Loop Join

$$\text{Cost}_{BNL} = \lceil \frac{|R|}{B} \rceil \times |S| + |R|$$

Where $B$ = number of rows fitting in the join buffer.

### Sort-Merge Join

$$\text{Cost}_{SM} = \text{Sort}(R) + \text{Sort}(S) + |R| + |S|$$

$$= O(|R| \log |R| + |S| \log |S| + |R| + |S|)$$

### Hash Join

$$\text{Cost}_{Hash} = 3 \times (|R| + |S|) \quad (\text{build + probe, with partitioning})$$

### Algorithm Selection Guide

| $|R|$ | $|S|$ | Best Algorithm | Cost |
|:---:|:---:|:---|:---:|
| 10 | 1M | Index NL (index on S) | 10 × log(1M) = 200 |
| 10K | 1M | Hash Join | 3 × 1.01M = 3.03M |
| 1M | 1M | Hash Join or Sort-Merge | ~3M or ~40M |
| 1M | 1M | Nested Loop (no index) | 10^12 (never) |

---

## 3. Index Selectivity

### The Model

Selectivity determines whether an index is useful. Low selectivity = few matches = index wins.

### Selectivity Formula

$$\text{Selectivity} = \frac{|\sigma_{pred}(R)|}{|R|}$$

### Predicate Selectivity Estimates

| Predicate | Selectivity | Example |
|:---|:---:|:---|
| `col = value` | $\frac{1}{V(col)}$ | PK lookup = $\frac{1}{n}$ |
| `col > value` | $\frac{\max - value}{\max - \min}$ | Range query |
| `col BETWEEN a AND b` | $\frac{b - a}{\max - \min}$ | Range scan |
| `col IN (v1, v2, ..., vk)` | $\frac{k}{V(col)}$ | Multi-value lookup |
| `col LIKE 'abc%'` | $\frac{1}{V(\text{prefix})}$ | Prefix search |
| `col IS NULL` | $\frac{\text{null\_count}}{|R|}$ | Null check |

### Combined Selectivity (Independence Assumption)

$$S_{A \land B} = S_A \times S_B$$

$$S_{A \lor B} = S_A + S_B - S_A \times S_B$$

### Worked Example

*"Table with 1M rows. WHERE status = 'active' (80%) AND region = 'US' (30%)."*

$$S_{combined} = 0.80 \times 0.30 = 0.24$$

$$\text{Expected rows} = 1,000,000 \times 0.24 = 240,000$$

Index useful? 24% selectivity is borderline — likely a full scan on HDD, possibly index scan on SSD.

---

## 4. Normalization Theory — Functional Dependencies

### Normal Forms

| Form | Requirement | Eliminates |
|:---|:---|:---|
| 1NF | Atomic values, no repeating groups | Nested data |
| 2NF | 1NF + no partial key dependencies | Partial dependencies |
| 3NF | 2NF + no transitive dependencies | Transitive dependencies |
| BCNF | Every determinant is a candidate key | All non-trivial FDs |

### Decomposition — Lossless Join

For a relation $R(A, B, C)$ decomposed into $R_1(A, B)$ and $R_2(A, C)$:

$$R_1 \bowtie R_2 = R \quad \text{(lossless)} \iff A \rightarrow B \text{ or } A \rightarrow C$$

### Functional Dependency Closure

$$A^+ = \{B : A \rightarrow B \text{ can be derived from FD set}\}$$

Algorithm: Repeatedly add attributes reachable via FDs until no new attributes are added. $O(|FD|^2)$.

---

## 5. Aggregation and Window Functions

### GROUP BY Complexity

$$\text{Hash Aggregation:} \quad O(n) \text{ time}, O(g) \text{ memory}$$

$$\text{Sort Aggregation:} \quad O(n \log n) \text{ time}, O(1) \text{ memory}$$

Where $g$ = number of groups.

### Window Function Cost

$$\text{Cost}_{window} = O(n \log n) \text{ (sort)} + O(n \times w) \text{ (frame evaluation)}$$

Where $w$ = window frame size.

| Window Frame | Cost per Row | Total for 1M Rows |
|:---|:---:|:---:|
| `ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT` | O(1) running | O(n) |
| `ROWS BETWEEN 10 PRECEDING AND 10 FOLLOWING` | O(21) | O(21n) |
| `RANGE BETWEEN INTERVAL '1 day'` | O(log n) | O(n log n) |

---

## 6. Query Optimization — The Search Space

### Join Order Problem

For $n$ tables, the number of possible join orders:

$$\text{Bushy plans} = \frac{(2(n-1))!}{(n-1)!}$$

$$\text{Left-deep plans} = n!$$

| Tables | Left-Deep Plans | Bushy Plans |
|:---:|:---:|:---:|
| 2 | 2 | 2 |
| 3 | 6 | 12 |
| 5 | 120 | 1,680 |
| 8 | 40,320 | 17,297,280 |
| 10 | 3,628,800 | ~1.76 × 10^10 |

**This is why query optimization is NP-hard** and planners use dynamic programming or heuristics.

### Dynamic Programming Cost

$$T_{DP} = O(2^n) \quad (\text{enumerate all subsets of tables})$$

Practical for $n \leq 12-15$ tables. Beyond that, heuristic/greedy planners are used.

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{|R| \times |S|}{V(A)}$ | Ratio | Join cardinality |
| $|R| \times O(\log |S|)$ | Log-linear | Index nested loop |
| $\frac{1}{V(col)}$ | Reciprocal | Selectivity |
| $S_A \times S_B$ | Probability | Combined selectivity |
| $n!$ and $(2n)!/(n!)$ | Factorial | Join order search space |
| $O(n \log n)$ | Linearithmic | Sort-based operations |

---

*Every `EXPLAIN`, `ANALYZE TABLE`, and query plan visualization reflects these cost calculations — SQL is a declarative language, but underneath it's an optimization problem that query planners solve millions of times per second.*

## Prerequisites

- Relational algebra fundamentals (sets, joins, projections)
- B-tree index structures
- Basic statistics (cardinality, selectivity)
- Algorithm complexity (Big-O notation for scan vs seek)

## Complexity

- **Beginner:** Reading EXPLAIN output, understanding scan types
- **Intermediate:** Cost model parameters, join algorithm selection, index coverage analysis
- **Advanced:** Join order combinatorics, histogram-based selectivity estimation, multi-column index prefix optimization
