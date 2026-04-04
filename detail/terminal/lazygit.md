# The Mathematics of lazygit -- Graph Theory and DAG Operations in Git

> *lazygit provides a visual interface to a directed acyclic graph (DAG) of commits, where branches are pointers, merges create multi-parent nodes, and interactive rebase restructures the graph while preserving content hashes through cryptographic commit identity.*

---

## 1. Commit Graph as DAG (Graph Theory)

### The Problem

Git history forms a directed acyclic graph $G = (V, E)$ where each vertex $v \in V$ is a commit and each directed edge $(v, u) \in E$ points from child to parent. Lazygit must render, traverse, and restructure this graph interactively.

### The Formula

A commit is a node with $0$ to $n$ parents:

$$\text{parents}(v) = \{u : (v, u) \in E\}$$

$$|\text{parents}(v)| = \begin{cases} 0 & \text{root commit} \\ 1 & \text{normal commit} \\ 2+ & \text{merge commit} \end{cases}$$

The commit graph is a DAG (no cycles):

$$\nexists\; v_1, v_2, \ldots, v_k : (v_i, v_{i+1}) \in E \;\forall i \text{ and } v_k = v_1$$

### Worked Examples

Linear history: $A \to B \to C \to D$ (4 nodes, 3 edges, max path length 3)

Feature branch merged: $A \to B \to D$ and $A \to C \to D$ where $D$ is a merge commit with $|\text{parents}(D)| = 2$.

---

## 2. Branch Visualization (Graph Layout)

### The Problem

Render a DAG as a 2D terminal layout with commit nodes in time order and branch lines showing parentage, minimizing edge crossings.

### The Formula

The Sugiyama framework for layered graph drawing:

1. **Layer assignment**: topological sort gives $y$-coordinate:

$$y(v) = \max_{u \in \text{parents}(v)} y(u) + 1$$

2. **Crossing minimization**: minimize $\sum_{(u,v), (u',v')} \text{cross}(u,v,u',v')$ which is NP-hard in general. Lazygit uses a heuristic: assign branch columns greedily.

3. **Column assignment**: each branch gets a column $x$:

$$x(\text{branch}_i) = \min\{c : c \text{ not occupied at layers where branch}_i \text{ is active}\}$$

The number of columns needed:

$$W = \max_{\text{layer}\; l} |\{\text{branches active at } l\}|$$

---

## 3. Interactive Rebase (DAG Rewriting)

### The Problem

Given a sequence of commits $C_1, C_2, \ldots, C_n$ on a branch, apply operations (pick, squash, fixup, drop, reorder) to produce a new sequence $C_1', C_2', \ldots, C_m'$ where $m \leq n$.

### The Formula

Each commit $C_i$ produces a patch (diff) $\delta_i$. Rebase replays patches on a new base $B$:

$$C_i' = \text{apply}(C_{i-1}', \delta_{\pi(i)})$$

where $\pi$ is the permutation/selection function from the rebase todo list.

For squash: $\delta_{squashed} = \delta_i \circ \delta_{i+1}$ (composition of patches)

$$\text{squash}(C_i, C_{i+1}) = \text{apply}(C_{i-1}', \delta_i \circ \delta_{i+1})$$

For fixup (squash without message):

$$\text{fixup}(C_i, C_{i+1}) = (\text{tree of squash}, \text{message of } C_i)$$

The SHA changes because the parent pointer changes:

$$\text{SHA}(C_i') = H(\text{tree}_i', \text{parent}_{i-1}', \text{message}_i, \text{author}_i, \text{timestamp})$$

---

## 4. Cherry-Pick (Patch Application)

### The Problem

Apply a commit's changes from one branch to another without merging the entire branch history.

### The Formula

Cherry-picking commit $C$ with parent $P$ onto branch $B$:

$$\delta_C = \text{diff}(P, C)$$
$$C' = \text{apply}(B, \delta_C)$$

The three-way merge for cherry-pick:

$$\text{result} = \text{merge3}(\text{base}=P, \text{ours}=B, \text{theirs}=C)$$

Conflict occurs when both $B$ and $C$ modify the same region relative to $P$:

$$\text{conflict} \iff \exists\; \text{region } r : \delta_{P \to B}(r) \neq \emptyset \land \delta_{P \to C}(r) \neq \emptyset \land \delta_{P \to B}(r) \neq \delta_{P \to C}(r)$$

---

## 5. Bisect (Binary Search on DAG)

### The Problem

Find the commit that introduced a bug. Given $n$ commits in topological order, identify the first "bad" commit using minimum tests.

### The Formula

On a linear history, bisect is binary search:

$$T_{bisect} = \lceil \log_2 n \rceil$$

On a DAG with merge commits, the bisection point maximizes information gain. For candidate set $C$ of potentially bad commits:

$$\text{midpoint} = \arg\min_{v \in C} \left| |\text{ancestors}(v) \cap C| - \frac{|C|}{2} \right|$$

This minimizes the worst-case remaining candidates:

$$|C_{next}| \leq \frac{|C|}{2} + 1$$

### Worked Examples

Linear history with 128 commits: $\lceil \log_2 128 \rceil = 7$ tests needed.

DAG with merges (effective length 100 after deduplication): $\lceil \log_2 100 \rceil = 7$ tests.

---

## 6. Merge Conflict Resolution (Three-Way Merge)

### The Problem

Combine changes from two branches with a common ancestor, detecting and presenting conflicting modifications.

### The Formula

Three-way merge of files $A$ (base), $B$ (ours), $C$ (theirs):

$$M(i) = \begin{cases} B(i) & \text{if } A(i) = C(i) \neq B(i) \\ C(i) & \text{if } A(i) = B(i) \neq C(i) \\ B(i) = C(i) & \text{if } B(i) = C(i) \neq A(i) \\ A(i) & \text{if } A(i) = B(i) = C(i) \\ \text{CONFLICT} & \text{if } A(i) \neq B(i) \neq C(i) \neq A(i) \end{cases}$$

Conflict rate estimation for random edits of length $n$ with edit probability $p$:

$$P(\text{conflict at line } i) = p_B \cdot p_C$$

$$E[\text{conflicts}] = n \cdot p_B \cdot p_C$$

---

## 7. Content-Addressable Storage (Cryptographic Hashing)

### The Problem

Git identifies every object (blob, tree, commit) by its SHA-1 hash. Any graph operation that changes content or parentage produces new hashes.

### The Formula

$$\text{SHA}_{commit} = \text{SHA-1}(\text{"commit " + len + "\0"} \| \text{tree} \| \text{parents} \| \text{author} \| \text{message})$$

$$\text{SHA}_{blob} = \text{SHA-1}(\text{"blob " + len + "\0"} \| \text{content})$$

The collision probability for $n$ objects:

$$P(\text{collision}) \approx 1 - e^{-n^2 / (2 \times 2^{160})} \approx \frac{n^2}{2^{161}}$$

For $n = 10^9$ objects: $P \approx 10^{18} / 10^{48} \approx 10^{-30}$.

Rebase changes parent hashes, cascading through all descendants:

$$\text{changed\_commits} = |\text{descendants}(C_{rebase\_start})| = n - i + 1$$

---

## 8. Stash as Merge Commit (Hidden DAG Nodes)

### The Problem

Git stash stores working directory state as hidden commits not on any branch.

### The Formula

A stash entry is a merge commit $S$ with two (or three) parents:

$$\text{parents}(S) = (HEAD, \text{index\_commit}, [\text{untracked\_commit}])$$

$$S = \text{merge}(HEAD, I, [U])$$

The stash reflog is a stack (LIFO):

$$\text{stash}@\{0\} = S_n, \quad \text{stash}@\{1\} = S_{n-1}, \quad \ldots$$

Pop operation: apply $\delta_{HEAD \to S}$ to current working tree and remove $S_n$ from the reflog.

---

## Prerequisites

- graph theory, directed acyclic graphs, topological sort, binary search, three-way merge algorithms, cryptographic hash functions, patch algebra

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Topological sort | $O(V + E)$ | $O(V)$ |
| Graph rendering | $O(V \times W)$ | $O(V \times W)$ |
| Bisect | $O(\log V)$ tests | $O(V)$ |
| Rebase (n commits) | $O(n \times \text{merge})$ | $O(n)$ |
| Cherry-pick | $O(\text{merge})$ | $O(\text{diff size})$ |
| Merge (3-way) | $O(n)$ per file | $O(n)$ |

---

*Every lazygit operation is a graph transformation: staging builds trees, committing adds nodes, branching creates pointers, merging joins paths, rebasing rewrites ancestry, and the content-addressable store ensures every transformation is traceable through cryptographic identity.*
