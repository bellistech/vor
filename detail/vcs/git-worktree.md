# The Mathematics of Git Worktree -- Parallel Branch Models and Object Sharing

> *Git worktrees create parallel working directories that share a single object database. The mathematics involve set-theoretic models of shared vs per-worktree state, disk space analysis comparing worktrees to full clones, and the combinatorial constraints on branch assignment across concurrent working trees.*

---

## 1. Worktree as Parallel State Machine (Set Theory)

### Repository State Decomposition

A git repository $R$ can be decomposed into shared state $S$ and per-worktree state $W_i$:

$$R = S \cup \{W_1, W_2, \ldots, W_n\}$$

### Shared State

The shared state is the object database plus refs:

$$S = \{\text{objects}, \text{refs}, \text{config}, \text{hooks}, \text{packed-refs}\}$$

### Per-Worktree State

Each worktree $W_i$ maintains:

$$W_i = \{\text{HEAD}_i, \text{index}_i, \text{working tree}_i, \text{reflog}_i, \text{merge/rebase state}_i\}$$

### Independence Property

Worktrees are independent in their mutable state:

$$\forall i \neq j: W_i \cap W_j = \emptyset$$

This means staging a file in worktree $W_1$ has no effect on the index of $W_2$.

---

## 2. Branch Assignment Constraint (Injection)

### The One-Branch-One-Worktree Rule

Git enforces an injective mapping from worktrees to branches:

$$f: \{W_1, \ldots, W_n\} \hookrightarrow \text{Branches}$$

$$\forall i \neq j: f(W_i) \neq f(W_j) \quad \text{(when both are on named branches)}$$

### Available Branches

Given $b$ total branches and $n$ worktrees, the number of valid assignments:

$$\text{Assignments} = \frac{b!}{(b - n)!} = P(b, n)$$

For 20 branches and 3 worktrees:

$$P(20, 3) = 20 \times 19 \times 18 = 6840 \text{ valid assignments}$$

### Detached HEAD Exception

Detached HEAD worktrees are exempt from the injection constraint:

$$f(W_i) = \text{detached}(c) \implies W_i \text{ does not consume a branch slot}$$

Multiple worktrees can be detached at the same commit.

---

## 3. Disk Space Analysis (Clone vs Worktree)

### Full Clone Cost

Each full clone $C_k$ stores its own object database:

$$\text{Space}_{\text{clones}}(n) = n \times (|O| + |W_{\text{avg}}|)$$

where $|O|$ is the compressed object database size and $|W_{\text{avg}}|$ is the average working tree size.

### Worktree Cost

Worktrees share the object database:

$$\text{Space}_{\text{worktrees}}(n) = |O| + n \times (|W_{\text{avg}}| + |I|)$$

where $|I|$ is the index file size (typically small, a few MB).

### Savings

$$\text{Savings} = \text{Space}_{\text{clones}}(n) - \text{Space}_{\text{worktrees}}(n) = (n - 1) \times |O|$$

$$\text{Savings \%} = \frac{(n-1) \times |O|}{n \times (|O| + |W_{\text{avg}}|)} \times 100$$

### Worked Example

For a repository with 500MB object database and 200MB average working tree:

| Worktrees | Clone Space | Worktree Space | Savings |
|:---:|:---:|:---:|:---:|
| 1 | 700 MB | 700 MB | 0 MB (0%) |
| 2 | 1400 MB | 900 MB | 500 MB (36%) |
| 3 | 2100 MB | 1100 MB | 1000 MB (48%) |
| 5 | 3500 MB | 1500 MB | 2000 MB (57%) |
| 10 | 7000 MB | 2500 MB | 4500 MB (64%) |

As $n \to \infty$:

$$\lim_{n \to \infty} \text{Savings \%} = \frac{|O|}{|O| + |W_{\text{avg}}|} \times 100 = \frac{500}{700} \times 100 = 71.4\%$$

---

## 4. Object Sharing Model (DAG Reachability)

### Object Reachability Across Worktrees

All worktrees share the same object DAG $G = (V, E)$ where vertices are git objects:

$$V = \text{Blobs} \cup \text{Trees} \cup \text{Commits} \cup \text{Tags}$$

Each worktree $W_i$ with HEAD at commit $c_i$ can access:

$$\text{Reachable}(c_i) = \{v \in V : \exists \text{ path } c_i \leadsto v \text{ in } G\}$$

### Shared Object Fraction

The fraction of objects shared between two worktrees on branches $c_1$ and $c_2$:

$$\text{Shared}(c_1, c_2) = \frac{|\text{Reachable}(c_1) \cap \text{Reachable}(c_2)|}{|\text{Reachable}(c_1) \cup \text{Reachable}(c_2)|}$$

For branches that recently diverged (few commits apart), this ratio approaches 1. For branches that diverged long ago with many file changes, it decreases but typically stays above 0.7 due to shared history.

### Fetch Propagation

A `git fetch` in any worktree adds objects to the shared database:

$$\text{fetch}(W_i) \implies S' = S \cup \Delta O$$

$$\forall j: \text{Reachable from } W_j \text{ includes } \Delta O$$

This means fetching once updates all worktrees.

---

## 5. Worktree Creation Cost (Time Complexity)

### Clone vs Worktree Add

| Operation | Network I/O | Disk I/O | Time |
|:---|:---:|:---:|:---:|
| `git clone` | $O(\|O\|)$ | $O(\|O\| + \|W\|)$ | Minutes |
| `git worktree add` | $0$ | $O(\|W\|)$ | Seconds |
| `git worktree add` (detached) | $0$ | $O(\|W\|)$ | Seconds |
| `git worktree remove` | $0$ | $O(\|W\|)$ | Seconds |

### Checkout Cost

Creating a worktree performs a checkout, which requires:

1. Reading the tree object for the target commit: $O(T)$ where $T$ = number of tree entries
2. Creating working tree files: $O(F)$ where $F$ = number of files
3. Building the index: $O(F \log F)$ for sorted index entries

$$T_{\text{worktree add}} = O(T + F \log F)$$

For a repository with 10,000 files:

$$T_{\text{worktree add}} \approx 10{,}000 \text{ file creates} + \text{index build} \approx 2\text{-}5 \text{ seconds}$$

$$T_{\text{clone}} \approx \text{network transfer} + 10{,}000 \text{ file creates} \approx 30\text{-}300 \text{ seconds}$$

---

## 6. Concurrent Operations (Locking)

### Index Lock Contention

Each worktree has its own `.git/worktrees/<name>/index.lock`:

$$\text{Lock}(W_i) \cap \text{Lock}(W_j) = \emptyset \quad \text{for } i \neq j$$

Worktrees never contend on index locks, enabling true parallel git operations.

### Ref Lock Contention

Ref updates use shared locks in the main `.git/refs/`:

$$P(\text{contention}) = 1 - \left(1 - \frac{1}{b}\right)^{n-1}$$

where $b$ is the number of branches and $n$ is the number of concurrent committers. For 100 branches and 3 concurrent worktrees:

$$P(\text{contention}) = 1 - \left(\frac{99}{100}\right)^2 = 0.0199 \approx 2\%$$

In practice, ref contention is negligible because each worktree typically works on its own branch.

### Object Database Concurrency

The object database supports concurrent reads. Writes use atomic file creation (write to temp, rename):

$$\text{Write}(o) = \text{write}(\text{tmp}) \to \text{rename}(\text{tmp}, \text{objects}/\text{hash}(o))$$

Rename is atomic on POSIX, so concurrent object writes are safe.

---

## 7. Worktree Lifecycle (State Machine)

### States

A worktree transitions through:

```
Created → Active → (Locked) → Prunable → Removed
```

$$W : \{\text{created}\} \xrightarrow{\text{add}} \{\text{active}\} \xrightarrow{\text{lock}} \{\text{locked}\} \xrightarrow{\text{unlock}} \{\text{active}\}$$

$$W : \{\text{active}\} \xrightarrow{\text{rm dir}} \{\text{prunable}\} \xrightarrow{\text{prune}} \{\text{removed}\}$$

### Stale Worktree Detection

A worktree is stale when its directory no longer exists:

$$\text{stale}(W_i) \iff \neg\text{exists}(\text{path}(W_i)) \wedge \neg\text{locked}(W_i)$$

`git worktree prune` removes metadata for stale worktrees:

$$\text{prune}: \{W_i : \text{stale}(W_i)\} \to \emptyset$$

---

## 8. Optimal Worktree Count (Resource Model)

### Memory Constraint

Each worktree consumes memory when active (editor buffers, build caches, running processes):

$$M_{\text{total}} = \sum_{i=1}^{n} M(W_i) \leq M_{\text{available}}$$

### Practical Limit

| Resource | Per Worktree | System Limit | Max Worktrees |
|:---|:---:|:---:|:---:|
| Working tree disk | 200 MB | 50 GB free | 250 |
| Editor memory | 500 MB | 16 GB RAM | 32 |
| Build cache | 1 GB | 50 GB free | 50 |
| Open file descriptors | 100 | 65,536 | 655 |
| **Practical** | **~2 GB** | **~16 GB** | **~8** |

The practical limit is cognitive, not technical: most developers work effectively with 2-4 concurrent worktrees.

---

## Prerequisites

- Set theory (injections, intersections, unions)
- Graph theory (DAG reachability, shared ancestors)
- Combinatorics (permutations for branch assignment)
- Asymptotic analysis (disk and time complexity)

## Complexity

| Operation | Time | Space |
|:---|:---:|:---:|
| `worktree add` | $O(F \log F)$ files | $O(F)$ working tree |
| `worktree remove` | $O(F)$ file deletes | $O(1)$ metadata |
| `worktree list` | $O(n)$ worktrees | $O(n)$ |
| `worktree prune` | $O(n)$ staleness checks | $O(1)$ |
| Object sharing (all worktrees) | $O(1)$ amortized | $O(1)$ marginal |
| Branch constraint check | $O(n)$ worktrees | $O(n)$ |

---

*The mathematics of git worktrees reduce to the economics of shared state. By factoring a repository into a large shared component (object database) and small per-instance components (index, HEAD), worktrees achieve near-zero marginal cost for parallel branch work. The key constraint is the injection from worktrees to branches -- each named branch can only be active in one worktree -- which prevents the state confusion that would arise from concurrent modifications to the same branch.*
