# The Internals of Git — Content-Addressable Storage, DAGs, and Merge Algorithms

> *Git is a content-addressable filesystem with a version control system built on top. Every object (blob, tree, commit, tag) is identified by its SHA-1 hash. Commits form a directed acyclic graph (DAG). Merge algorithms use three-way comparison with recursive common ancestor resolution. Pack files use delta compression with a sliding window to achieve remarkable storage efficiency.*

---

## 1. Content-Addressable Storage

### The Object Model

Every piece of data in Git is stored as an **object** identified by its SHA-1 hash:

$$\text{hash} = \text{SHA-1}(\text{type} + \text{ } + \text{size} + \text{\textbackslash 0} + \text{content})$$

### Four Object Types

| Type | Content | Example Hash |
|:-----|:--------|:-------------|
| **Blob** | File content (no filename, no metadata) | `af5626b...` |
| **Tree** | Directory listing (mode, type, hash, name) | `4b825dc...` |
| **Commit** | Tree pointer, parent(s), author, message | `a1b2c3d...` |
| **Tag** | Points to commit with GPG signature | `e4f5g6h...` |

### Blob = Pure Content

Two files with identical content produce the **same blob hash**, regardless of filename or location:

```
echo "hello" | git hash-object --stdin
# ce013625030ba8dba906f756967f9e9ca394464a

# Same content in any file, any directory → same hash
```

### Tree = Directory Snapshot

```
100644 blob a1b2c3d4...  README.md
100644 blob e5f6g7h8...  main.go
040000 tree i9j0k1l2...  internal/
```

Mode values:
| Mode | Meaning |
|:-----|:--------|
| `100644` | Regular file |
| `100755` | Executable file |
| `120000` | Symbolic link |
| `040000` | Subdirectory (tree) |

### Commit Object

```
tree 4b825dc642cb6eb9a060e54bf899d15006d2ea7e
parent a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0
author Alice <alice@example.com> 1706000000 -0500
committer Alice <alice@example.com> 1706000000 -0500

Initial commit
```

A commit points to:
- Exactly **one tree** (the root directory snapshot)
- Zero or more **parents** (0 for initial, 1 for normal, 2+ for merges)

---

## 2. The Commit DAG

### Directed Acyclic Graph

Commits form a **DAG** — each commit points to its parent(s):

```
A ← B ← C ← D        (linear history)
         ↑
         └── E ← F    (branch)

Merge:
A ← B ← C ← D ← G    (G has two parents: D and F)
         ↑          ↗
         └── E ← F
```

### DAG Properties

| Property | Consequence |
|:---------|:-----------|
| Directed | Each commit points to its parent(s), not children |
| Acyclic | No commit can be its own ancestor |
| Immutable | Changing any object changes its hash → new object |
| Content-addressed | Same content → same hash → automatic deduplication |

### Reachability

A commit $C$ is **reachable** from commit $H$ (HEAD) if there exists a path from $H$ to $C$ following parent edges.

$$\text{reachable}(C, H) \iff \exists \text{ path } H \to \ldots \to C$$

Unreachable commits are garbage collected (after `gc.reflogExpire`, default 90 days).

### Branch = Pointer to Commit

A branch is just a **file** containing a commit hash:

```
.git/refs/heads/main    → a1b2c3d4e5f6...
.git/refs/heads/feature → e5f6g7h8i9j0...
.git/HEAD               → ref: refs/heads/main
```

HEAD is a symbolic reference — it points to a branch, which points to a commit.

---

## 3. The Three Trees

### Working Directory, Staging Area, Repository

```
Working Directory        Staging Area (Index)        Repository (.git)
  (files on disk)         (.git/index)               (object database)
        │                      │                          │
        ├── git add ──────────►│                          │
        │                      ├── git commit ───────────►│
        │◄── git checkout ─────┤◄── git reset ────────────┤
        │                      │                          │
```

### The Index File

`.git/index` is a **binary file** containing:
- Cached file metadata (mtime, size, mode)
- SHA-1 hash of staged blob
- File path

This is why `git status` is fast — it compares file metadata with the index, only hashing files that might have changed.

### The Reset Matrix

| Command | Working Dir | Index | HEAD |
|:--------|:---:|:---:|:---:|
| `git reset --soft` | Unchanged | Unchanged | Moved |
| `git reset --mixed` (default) | Unchanged | Reset | Moved |
| `git reset --hard` | Reset | Reset | Moved |

---

## 4. Merge Algorithms

### Three-Way Merge

Given two branches diverging from a common ancestor:

```
         Base (common ancestor)
        /    \
    Ours      Theirs
```

For each line/hunk, the three-way merge decides:

| Base | Ours | Theirs | Result |
|:-----|:-----|:-------|:-------|
| A | A | A | A (no change) |
| A | B | A | B (ours changed) |
| A | A | C | C (theirs changed) |
| A | B | C | **CONFLICT** (both changed differently) |
| A | B | B | B (both changed the same way) |

### Recursive Merge Strategy (Default)

When two branches have **multiple common ancestors** (criss-cross merge), Git uses the **recursive** strategy:

1. Find all common ancestors
2. If multiple, **merge the ancestors** recursively to create a virtual ancestor
3. Use the virtual ancestor as the base for three-way merge

```
    A ← B ← C ← D (ours)
    ↑↗      ↑↗
    E ← F ← G ← H (theirs)

Common ancestors: C-G merge → virtual base → three-way merge D and H
```

### Merge Strategies

| Strategy | Command | Use Case |
|:---------|:--------|:---------|
| Recursive | `git merge` (default) | Standard merge |
| Ort | `git merge` (Git 2.34+, default) | Faster recursive |
| Octopus | `git merge A B C` | Merge multiple branches |
| Ours | `git merge -s ours` | Discard other branch's changes |
| Subtree | `git merge -s subtree` | Merge at different tree path |

### Fast-Forward Merge

When the current branch is an ancestor of the merged branch:

```
A ← B ← C (main)
              ← D ← E (feature)

git merge feature → just move main pointer to E
```

No merge commit needed. To force a merge commit: `git merge --no-ff`.

---

## 5. Pack Files — Delta Compression

### Loose vs Packed Objects

| Storage | Format | When |
|:--------|:-------|:-----|
| Loose | Individual zlib-compressed files in `.git/objects/` | After each operation |
| Packed | Delta-compressed in `.git/objects/pack/` | After `git gc` or `git push` |

### Pack File Structure

```
Pack file (.pack):
┌──────────┐
│ Header    │  "PACK" + version + object count
├──────────┤
│ Object 1  │  type + size + data (zlib compressed)
│ Object 2  │  type + size + delta base ref + delta data
│ ...       │
├──────────┤
│ Checksum  │  SHA-1 of entire pack
└──────────┘

Index file (.idx):
  Sorted hash → offset mapping for O(log n) lookup
```

### Delta Compression

Similar objects are stored as **deltas** — instructions to reconstruct one object from another:

$$\text{stored\_size}(\text{delta}) = \text{size}(\text{copy instructions} + \text{insert instructions})$$

Delta instructions:
- **Copy:** "copy bytes [offset, length] from base object"
- **Insert:** "insert these literal bytes"

### Sliding Window Algorithm

Git's delta compression uses a **sliding window** (default: 10 objects):

1. Sort objects by type and size
2. For each object, try to delta-compress against the previous N objects in the window
3. Keep the smallest representation (full or delta)

```bash
git repack -a -d --window=250 --depth=250    # aggressive repacking
```

| Parameter | Default | Meaning |
|:----------|:--------|:--------|
| `--window` | 10 | Number of objects to try as delta bases |
| `--depth` | 50 | Maximum delta chain length |

### Worked Example: Compression Ratio

A 1MB file modified slightly (10 bytes changed):

```
Loose: 1MB (base) + 1MB (modified) = 2MB
Packed: 1MB (base) + ~100 bytes (delta) ≈ 1MB

Compression ratio: 50% → ~99.99%
```

This is why `git clone` downloads far less data than the sum of all file versions.

---

## 6. Reflog — Safety Net

### What Reflog Tracks

Every HEAD movement is recorded:

```bash
git reflog
# a1b2c3d HEAD@{0}: commit: Add feature
# e5f6g7h HEAD@{1}: checkout: moving from main to feature
# i9j0k1l HEAD@{2}: commit: Fix bug
```

### Recovery

```bash
# Accidentally reset --hard? Find the commit:
git reflog
git reset --hard HEAD@{1}    # go back to previous state

# Accidentally deleted a branch?
git reflog | grep "branch-name"
git checkout -b recovered <hash>
```

### Reflog Expiry

| Setting | Default | Meaning |
|:--------|:--------|:--------|
| `gc.reflogExpire` | 90 days | Reachable entries |
| `gc.reflogExpireUnreachable` | 30 days | Unreachable entries |

After expiry, `git gc` can delete objects only referenced by expired reflog entries.

---

## 7. Transfer Protocols

### Smart Protocol (Default)

```
Client                              Server
  │                                    │
  ├── "want" hashes ──────────────────►│
  │                                    │
  │◄── "have" negotiation ────────────►│
  │    (find common commits)           │
  │                                    │
  │◄── Pack file (only new objects) ───┤
  │                                    │
```

### Negotiation Algorithm

1. Client sends all **want** hashes (refs it doesn't have)
2. Client sends **have** hashes (commits it already has)
3. Server finds **common ancestors**
4. Server sends pack file with objects reachable from wants but not from haves

$$\text{Transfer size} = \text{objects}(\text{want} \setminus \text{have})$$

### Protocol Versions

| Version | Transport | Features |
|:--------|:----------|:---------|
| v0 | SSH, HTTP | Original |
| v1 | SSH, HTTP | No functional change |
| v2 | SSH, HTTP | Ref advertisement filtering, partial clone |

---

## 8. Worktrees, Sparse Checkout, and Partial Clone

### Git Worktrees

Multiple working directories from one `.git`:

```bash
git worktree add ../feature-branch feature
# Creates a new directory with feature branch checked out
# Shares the same object database
```

### Sparse Checkout

Only check out a subset of files:

```bash
git sparse-checkout set src/ docs/
# Only src/ and docs/ appear in working directory
```

### Partial Clone

Clone without downloading all objects:

```bash
git clone --filter=blob:none <url>    # download blobs on demand
git clone --filter=tree:0 <url>       # download trees on demand
```

Blobs are fetched lazily when needed — dramatically reduces initial clone time for large repos.

---

## 9. Summary of Key Concepts

| Concept | Detail |
|:--------|:-------|
| Object model | 4 types: blob, tree, commit, tag (SHA-1 addressed) |
| Commit graph | DAG (directed acyclic graph) |
| Branches | Mutable pointers to commits (just files) |
| Index | Binary cache of staged file state |
| Three-way merge | Compare base/ours/theirs per hunk |
| Recursive merge | Merge common ancestors recursively |
| Pack files | Delta compression with sliding window |
| Reflog | 90-day safety net for all HEAD movements |
| Transfer | Send only objects not in common ancestor set |

---

*Git's design insight is that version control is a graph problem over content-addressed storage. Every commit is immutable (changing it changes its hash, which changes its children's hashes). Every branch is a pointer (cheap to create, rename, delete). Every merge is a graph operation (find common ancestor, three-way diff). The entire complexity of Git reduces to understanding these three concepts: hashes, DAGs, and three-way merge.*

## Prerequisites

- Directed acyclic graphs (DAGs) and graph traversal
- Content-addressable storage (SHA-1/SHA-256 hashing)
- Three-way merge algorithm (common ancestor, ours, theirs)
- Diff algorithms (Myers, patience, histogram)
