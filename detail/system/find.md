# The Mathematics of find — Directory Traversal, Inode Costs & Optimization

> *find is a tree traversal algorithm with I/O costs at every node. Understanding the filesystem as a tree structure — and the cost of each inode lookup — transforms find from a blunt tool into a precision instrument.*

---

## 1. Directory Tree as a Graph

### The Filesystem Model

A filesystem is a **rooted tree** (ignoring hard links and symlinks):

- **Nodes** = inodes (files and directories)
- **Edges** = directory entries (dentries)
- **Root** = the starting path argument to find

### Key Metrics

$$N = \text{total nodes (files + directories)}$$
$$D = \text{total directories}$$
$$F = \text{total files} = N - D$$
$$d = \text{maximum depth}$$
$$b = \text{average branching factor (entries per directory)}$$

For a balanced tree: $N = \frac{b^{d+1} - 1}{b - 1}$

### Worked Example

A project with 10,000 files, average 20 entries per directory:

$$D \approx \frac{N}{b} = \frac{10000}{20} = 500 \text{ directories}$$

$$d \approx \log_b(N) = \log_{20}(10000) = \frac{\log(10000)}{\log(20)} \approx 3.1 \text{ levels}$$

---

## 2. Traversal Strategies — BFS vs DFS

### find's Default: DFS (Depth-First Search)

find uses **depth-first traversal** by default, which determines the order results appear:

| Strategy | Memory | Order | find flag |
|:---|:---:|:---|:---|
| DFS (pre-order) | $O(d)$ | Parent before children | Default |
| DFS (depth-limited) | $O(d)$ | Parent before children | `-maxdepth` |
| BFS (breadth-first) | $O(b^d)$ | Level by level | Not native |

DFS memory usage scales with depth, not total nodes:

$$M_{DFS} = O(d) = O(\log_b N)$$

For 10 million files with branching factor 100: $M_{DFS} = O(\log_{100} 10^7) = O(3.5)$, just a few stack frames.

### -depth Flag (Post-Order DFS)

`find -depth` processes children before parents. Essential for `find -delete` (can't delete a directory before its contents).

---

## 3. I/O Cost Model

### Per-Node System Call Cost

Each node visited by find requires system calls:

| Operation | System Call | Cost (SSD) | Cost (HDD) |
|:---|:---|:---:|:---:|
| Read directory | `getdents64()` | 5-50 us | 2-10 ms |
| Stat file | `lstat()` | 1-10 us | 0.5-5 ms |
| Check permission | `access()` | 1-5 us | 0.5-2 ms |

### Total I/O Cost

$$T_{find} = D \times T_{readdir} + N \times T_{stat}$$

**Example:** 10,000 files, 500 directories, on HDD:

$$T_{find} = 500 \times 5ms + 10000 \times 2ms = 2.5s + 20s = 22.5s$$

On SSD: $T_{find} = 500 \times 25\mu s + 10000 \times 5\mu s = 12.5ms + 50ms = 62.5ms$

The **360x speedup** from SSD is almost entirely due to eliminating seek time.

### Inode Cache (dcache/icache) Effect

The kernel caches inode and dentry lookups. On repeated find runs:

$$T_{cached} \approx N \times T_{memory\_lookup} \approx N \times 0.1\mu s$$

For 10,000 files: $T_{cached} \approx 1ms$. Cold cache to warm cache is a 1000x+ improvement.

---

## 4. -prune Optimization — Subtree Elimination

### The Pruning Formula

`-prune` eliminates entire subtrees from traversal:

$$N_{visited} = N_{total} - \sum_{p \in pruned} size(subtree_p)$$

$$savings = \frac{\sum size(subtree_p)}{N_{total}} \times 100\%$$

### Worked Example

Repository with 50,000 files. `node_modules/` contains 35,000 files:

```bash
find . -path ./node_modules -prune -o -name "*.js" -print
```

$$N_{visited} = 50000 - 35000 = 15000$$

$$savings = \frac{35000}{50000} = 70\%$$

Without prune: 50,000 stat() calls. With prune: 15,000 stat() calls + 1 prune check.

### Multiple Prune Targets

```bash
find . \( -path ./node_modules -o -path ./.git -o -path ./vendor \) -prune -o -name "*.go" -print
```

If these three directories contain 40,000 of 50,000 files:

$$speedup = \frac{50000}{10000} = 5\times$$

---

## 5. Expression Evaluation — Short-Circuit Logic

### find's Expression Model

find expressions are evaluated left-to-right with **short-circuit** evaluation:

- `-a` (AND, implicit): if left is false, skip right
- `-o` (OR): if left is true, skip right

### Optimization Rule

Place the **cheapest and most selective** test first:

$$T_{optimized} = T_{test1} + P(test1) \times T_{test2}$$

$$T_{unoptimized} = T_{test2} + P(test2) \times T_{test1}$$

**Example:** Finding `.log` files larger than 100 MB:

- `-name "*.log"` — cheap (string comparison), matches 5% of files
- `-size +100M` — expensive (stat() required), matches 1% of files

Optimized order: `-name "*.log" -size +100M`

$$T = N \times T_{name} + 0.05N \times T_{stat} = N \times 0.1\mu s + 0.05N \times 5\mu s$$

Unoptimized: `-size +100M -name "*.log"`

$$T = N \times T_{stat} + 0.01N \times T_{name} = N \times 5\mu s + 0.01N \times 0.1\mu s$$

**Ratio:** The optimized version is ~20x cheaper when stat() dominates.

---

## 6. -exec vs -exec + vs xargs — Process Spawning Cost

### Fork/Exec Cost Model

| Method | Processes Spawned | Total Cost |
|:---|:---:|:---|
| `-exec cmd {} \;` | $N_{matches}$ | $N \times T_{fork+exec}$ |
| `-exec cmd {} +` | $\lceil N / ARG\_MAX \rceil$ | $\lceil N/batch \rceil \times T_{fork+exec}$ |
| `\| xargs` | $\lceil N / ARG\_MAX \rceil$ | Same as `+`, extra pipe overhead |
| `-delete` | 0 | Inline `unlinkat()` syscall |

Where $T_{fork+exec} \approx 1-5ms$ and $ARG\_MAX \approx 2,097,152$ bytes on Linux.

### Worked Example

Delete 10,000 `.tmp` files:

| Method | Processes | Time |
|:---|:---:|:---:|
| `-exec rm {} \;` | 10,000 | $10000 \times 3ms = 30s$ |
| `-exec rm {} +` | ~5 | $5 \times 3ms = 15ms$ + rm time |
| `-delete` | 0 | Pure syscall, ~100ms |

The `-delete` flag is **300x faster** than `-exec rm {} \;` because it avoids 10,000 fork+exec cycles.

---

## 7. Time-Based Predicates — Clock Arithmetic

### -mtime, -atime, -ctime

These use **24-hour blocks** measured from now:

$$-mtime\ n \implies n \times 24h \leq age < (n+1) \times 24h$$

$$-mtime\ +n \implies age > (n+1) \times 24h$$

$$-mtime\ -n \implies age < n \times 24h$$

### -newer and -newerXY

`-newer ref` matches files modified after `ref`'s modification time:

$$mtime(file) > mtime(ref)$$

`-newermt "2024-01-01"` matches files modified after the given date (parsed to epoch timestamp).

### -mmin (Minute Precision)

$$-mmin\ n \implies n \times 60s \leq age < (n+1) \times 60s$$

**Common pitfall:** `-mtime 0` means "modified in the last 24 hours," not "modified today."

---

## 8. Filesystem-Specific Performance

### ext4 Directory Hashing

ext4 uses **HTree** (B-tree indexed by filename hash) for directory lookups:

$$T_{lookup} = O(\log_b n) \text{ where } b \approx 200 \text{ (HTree branching factor)}$$

For a directory with 100,000 entries: $\log_{200}(100000) \approx 2.2$ disk reads.

### XFS vs ext4 for Large Directories

| Filesystem | Directory Index | Lookup Cost | Readdir Cost |
|:---|:---|:---:|:---:|
| ext4 | HTree (hash) | $O(\log n)$ | $O(n)$ |
| XFS | B+tree | $O(\log n)$ | $O(n)$, sorted |
| tmpfs | Hash table | $O(1)$ amortized | $O(n)$ |

### Inode Size Impact

Each `lstat()` reads an inode (typically 256 bytes on ext4). On HDD with 4K sectors:

$$reads = \lceil \frac{256}{4096} \rceil = 1 \text{ sector per inode}$$

But inodes are often adjacent — readahead can amortize: $T_{readahead} \approx T_{seek} + N_{inodes} \times T_{transfer}$.

---

## 9. Summary of find Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Tree size | $b^{d+1}-1) / (b-1)$ | Geometric series |
| DFS memory | $O(\log_b N)$ | Logarithmic |
| I/O cost | $D \times T_{readdir} + N \times T_{stat}$ | Linear scan |
| Prune savings | $\sum subtree\_size / N_{total}$ | Subtree elimination |
| Short-circuit | $T_1 + P_1 \times T_2$ | Conditional probability |
| Exec batching | $\lceil N / ARG\_MAX \rceil$ | Ceiling division |
| Time predicates | $n \times 24h \leq age < (n+1) \times 24h$ | Interval arithmetic |

---

*find walks a tree, and every optimization — pruning, expression ordering, exec batching — reduces the number of nodes visited or the cost per node. Think of it as graph search with I/O weights on every edge.*
