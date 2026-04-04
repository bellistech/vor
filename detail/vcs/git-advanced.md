# The Mathematics of Git -- Content-Addressable Storage and Graph Algorithms

> *Git is not a version control system; it is a content-addressable filesystem with a version control user interface bolted on top.*

---

## 1. Binary Search in Bisect (Algorithm Analysis)

### The Problem

`git bisect` finds the commit that introduced a bug by binary search over the commit history. For a linear history of $n$ commits, this is straightforward, but Git histories are DAGs, requiring a generalized binary search.

### The Formula

For a linear history of $n$ commits, bisect requires at most:

$$k = \lceil \log_2(n) \rceil$$

tests to identify the first bad commit. For a DAG with $n$ commits and $b$ branches, the optimal strategy minimizes the worst-case number of tests:

$$k_{\text{DAG}} = \lceil \log_2(|S|) \rceil$$

where $S$ is the set of candidate commits between the known good and bad refs. Git selects the commit $c$ that minimizes:

$$\max(|\text{ancestors}(c) \cap S|, |S| - |\text{ancestors}(c) \cap S|)$$

This is equivalent to finding the median of the candidate set by ancestry count, splitting $S$ as evenly as possible.

### Worked Examples

Linear history with $n = 1{,}000$ commits between good and bad: $k = \lceil \log_2(1000) \rceil = 10$ tests.

If each test takes 30 seconds (compile + run): total bisect time = $10 \times 30 = 300$ seconds = 5 minutes. A linear search would take $500 \times 30 = 15{,}000$ seconds = 4.2 hours on average.

DAG with a branch that merged back in: 800 commits on main, 200 on a feature branch merged at commit 900. $|S| = 1{,}000$, so $k = 10$ tests. Git selects the commit at the ancestry midpoint, which may be on either branch.

---

## 2. SHA-1 Object Identity (Cryptographic Hashing)

### The Problem

Git identifies every object (blob, tree, commit, tag) by the SHA-1 hash of its content. This creates a content-addressable store where identical content always maps to the same key, and any modification produces a completely different key.

### The Formula

The hash of a Git object is:

$$\text{SHA-1}(\texttt{type} \| \texttt{ } \| \text{len} \| \texttt{\textbackslash 0} \| \text{content})$$

The birthday paradox gives the collision probability after $n$ objects:

$$P(\text{collision}) \approx 1 - e^{-n^2 / 2^{161}}$$

The expected number of objects before a 50% collision probability:

$$n_{50\%} = \sqrt{\frac{\pi}{2} \cdot 2^{160}} \approx 1.71 \times 2^{80} \approx 2.07 \times 10^{24}$$

### Worked Examples

The Linux kernel repository has approximately $10^7$ objects. Collision probability:

$$P \approx \frac{(10^7)^2}{2^{161}} = \frac{10^{14}}{2.9 \times 10^{48}} \approx 3.4 \times 10^{-35}$$

Even if every human on Earth ran a Linux-sized repository: $8 \times 10^9 \times 10^7 = 8 \times 10^{16}$ objects:

$$P \approx \frac{(8 \times 10^{16})^2}{2^{161}} \approx 2.2 \times 10^{-15}$$

Still negligible. Git is migrating to SHA-256 ($2^{256}$) for defense against targeted collision attacks, not random collisions.

---

## 3. Merge-Base and Lowest Common Ancestor (Graph Theory)

### The Problem

Merging two branches requires finding their merge base -- the lowest common ancestor (LCA) in the commit DAG. This determines the diff3 three-way merge inputs.

### The Formula

For a DAG $G = (V, E)$ and two commits $a, b \in V$, the set of common ancestors is:

$$\text{CA}(a, b) = \text{ancestors}(a) \cap \text{ancestors}(b)$$

The LCA is the maximal element(s) of CA by the reachability partial order:

$$\text{LCA}(a, b) = \{c \in \text{CA}(a, b) : \nexists c' \in \text{CA}(a, b), c \neq c', c \in \text{ancestors}(c')\}$$

There may be multiple LCAs (criss-cross merges). Git handles this with recursive merge: it merges the LCAs to create a virtual merge base.

The algorithm runs in $O(|V| + |E|)$ using simultaneous BFS from both commits, aided by the commit-graph file which provides generation numbers for $O(1)$ reachability filtering.

### Worked Examples

Commit graph:
```
A---B---C---F (main)
     \     /
      D---E (feature)
```

$\text{ancestors}(F) = \{A, B, C, D, E, F\}$, $\text{ancestors}(E) = \{A, B, D, E\}$.

$\text{CA}(F, E) = \{A, B\}$. LCA = $\{B\}$ since $A \in \text{ancestors}(B)$.

Criss-cross example:
```
A---B---D---F (main)
     \ / \ /
      X   Y
     / \ / \
A---C---E---G (feature)
```

$\text{LCA}(F, G) = \{D, E\}$ -- two merge bases. Git creates virtual commit $M = \text{merge}(D, E)$ and uses $M$ as the merge base for merging $F$ and $G$.

---

## 4. Packfile Delta Compression (Information Theory)

### The Problem

Git stores objects efficiently using delta compression in packfiles. Similar objects (different versions of the same file) are stored as a base object plus a delta, dramatically reducing storage.

### The Formula

The delta size for transforming base $B$ into target $T$ is bounded by the edit distance:

$$|\delta(B, T)| \leq |T| - \text{LCS}(B, T) + O(\log |B|)$$

where LCS is the longest common subsequence. The compression ratio is:

$$r = \frac{|\delta(B, T)|}{|T|}$$

Git selects base objects using a heuristic: objects are sorted by type, filename, and size, then a sliding window of $w$ objects (default $w = 10$) is searched for the best base:

$$\text{base}(T) = \arg\min_{B \in \text{window}} |\delta(B, T)|$$

Total packfile size for $n$ objects with average raw size $\bar{s}$ and compression ratio $r$:

$$S_{\text{pack}} \approx n \cdot \bar{s} \cdot r + S_{\text{base objects}}$$

### Worked Examples

A file `app.js` (50 KiB) with 100 versions, each differing by 500 bytes on average.

Raw storage: $100 \times 50 = 5{,}000$ KiB.

Delta storage: 1 base (50 KiB) + 99 deltas ($\approx 500$ bytes + overhead each $\approx 600$ bytes): $50 + 99 \times 0.6 \approx 109$ KiB.

Compression ratio: $109/5000 = 0.022$ or $46\times$ reduction. With zlib on top (typically $2\text{-}3\times$ further): final size $\approx 40$ KiB for 100 versions of a 50 KiB file.

---

## 5. Reflog as Write-Ahead Log (Database Theory)

### The Problem

The reflog provides crash recovery and undo capability by recording every ref update. It functions like a write-ahead log (WAL) in databases, ensuring that no reachable state is lost even after destructive operations.

### The Formula

The reflog for ref $r$ is an append-only sequence of entries:

$$\text{reflog}(r) = [(t_0, h_0, h_0'), (t_1, h_1, h_1'), \ldots, (t_n, h_n, h_n')]$$

where $t_i$ is the timestamp, $h_i$ is the old value, and $h_i'$ is the new value. The state of ref $r$ at any time $t$ can be recovered:

$$r(t) = h_j' \quad \text{where } j = \max\{i : t_i \leq t\}$$

The recovery window $W$ before an object becomes unreachable and eligible for garbage collection:

$$W = \min(T_{\text{reflog\_expire}}, T_{\text{gc\_prune}})$$

Default: $T_{\text{reflog\_expire}} = 90$ days for reachable entries, $30$ days for unreachable. An object is safe from `gc` as long as any reflog entry references it.

### Worked Examples

Developer runs `git reset --hard HEAD~5` at $t = T$, discarding 5 commits. The reflog records:

$(T, \text{abc1234}, \text{def5678})$ -- HEAD moved from abc1234 to def5678.

Recovery at $T + 1\text{h}$: `git reset --hard HEAD@{1}` restores abc1234. The 5 "lost" commits remain in the object store.

Recovery window: commits remain recoverable for 30 days (unreachable reflog expiry). After `git reflog expire --expire=30.days.ago` and `git gc --prune=now`, the objects are permanently deleted.

Storage cost: each reflog entry is $\approx 200$ bytes. With 100 ref updates/day for 90 days: $100 \times 90 \times 200 = 1.8$ MiB -- negligible.

---

## 6. Three-Way Merge Correctness (Set Theory)

### The Problem

Git's merge algorithm compares two branches against their common ancestor to determine which changes to accept. The three-way merge must correctly handle concurrent modifications to the same file.

### The Formula

Given base version $B$, and two derived versions $L$ (left/ours) and $R$ (right/theirs), define the change sets:

$$\Delta_L = L \setminus B, \quad \Delta_R = R \setminus B$$

The merge result $M$ is:

$$M = B \cup \Delta_L \cup \Delta_R \quad \text{if } \Delta_L \cap \Delta_R = \emptyset$$

When changes overlap ($\Delta_L \cap \Delta_R \neq \emptyset$), a conflict occurs if $L|_{\text{overlap}} \neq R|_{\text{overlap}}$. The probability of conflict for two developers making $k_L$ and $k_R$ independent line changes in a file of $n$ lines:

$$P(\text{conflict}) = 1 - \frac{\binom{n}{k_L} \cdot \binom{n - k_L}{k_R}}{\binom{n}{k_L} \cdot \binom{n}{k_R}} = 1 - \prod_{i=0}^{k_R - 1} \frac{n - k_L - i}{n - i}$$

### Worked Examples

File with $n = 500$ lines. Developer A changes $k_L = 10$ lines, developer B changes $k_R = 5$ lines.

$$P(\text{conflict}) = 1 - \frac{490}{500} \cdot \frac{489}{499} \cdot \frac{488}{498} \cdot \frac{487}{497} \cdot \frac{486}{496}$$

$$\approx 1 - 0.980 \times 0.980 \times 0.980 \times 0.980 \times 0.980 \approx 1 - 0.904 = 0.096$$

Approximately 9.6% chance of conflict. With $k_L = k_R = 50$ lines each: $P \approx 1 - (0.9)^{50} \approx 1 - 0.0052 = 99.5\%$ -- almost certain conflict, which is why large PRs cause merge pain.

---

## Prerequisites

- Binary search and algorithm complexity analysis
- Cryptographic hash functions (SHA-1, SHA-256, birthday paradox)
- Graph theory (DAGs, lowest common ancestor, BFS/DFS)
- Edit distance and longest common subsequence
- Information theory (entropy, compression ratios)
- Database theory (write-ahead logging, crash recovery)
- Partial orders (maximal elements, lattice operations)
- Set theory (union, intersection, difference, conflict detection)
