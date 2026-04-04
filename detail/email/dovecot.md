# The Mathematics of Dovecot -- Storage Geometry and Index Structures

> *An IMAP server is a database engine in disguise: it indexes millions of messages, enforces storage quotas as linear constraints, and serves concurrent clients through lock-free data structures.*

---

## 1. Mailbox Storage Formats (Space and I/O Complexity)

### The Problem

Dovecot supports multiple storage formats -- Maildir, mbox, sdbox, mdbox -- each with different performance characteristics. The choice affects I/O operations per message access, disk space overhead, and directory entry scaling. Understanding the complexity of each format lets administrators choose correctly for their workload.

### The Formula

For a mailbox with $n$ messages, each of average size $s$ bytes:

**Maildir** (one file per message):
$$\text{Files} = n, \quad \text{Inodes} = n + 3, \quad \text{Readdir cost} = O(n)$$
$$\text{Disk} = n \cdot (s + b_{fs})$$

where $b_{fs}$ is the filesystem block overhead (typically 4096-byte alignment). Wasted space per message:

$$w = b_{fs} - (s \mod b_{fs}) \quad \text{when } s \mod b_{fs} \neq 0$$

**mbox** (single file per folder):
$$\text{Files} = 1, \quad \text{Inodes} = 1, \quad \text{Seek cost} = O(\log n) \text{ with index}$$
$$\text{Disk} = \sum_{i=1}^{n} (s_i + h_i) \quad \text{(no block waste per message)}$$

where $h_i$ is the "From " separator line overhead (~60 bytes).

**mdbox** (multi-dbox, multiple messages per file):
$$\text{Files} = \lceil n / m_{max} \rceil, \quad \text{Disk} = n \cdot (s + 24) \quad \text{(24-byte dbox header)}$$

where $m_{max}$ is the maximum messages per file (configurable via `mdbox_rotate_size`).

### Worked Examples

**Example: 100,000 messages, average 10 KB each, ext4 filesystem (4 KB blocks).**

Maildir:
- Files: 100,000
- Inodes: 100,003
- Wasted space: each 10 KB message uses 3 blocks (12 KB), waste = $100{,}000 \times 2048 = 195$ MB
- Total disk: $100{,}000 \times 12{,}288 = 1.17$ GB
- Readdir: 100,000 entries (slow on ext3, acceptable on ext4 with dir_index)

mdbox (2 MB rotation):
- Messages per file: $\lfloor 2{,}000{,}000 / 10{,}240 \rfloor = 195$
- Files: $\lceil 100{,}000 / 195 \rceil = 513$
- No per-message block waste
- Total disk: $100{,}000 \times 10{,}264 = 978$ MB

Savings: mdbox uses 16% less disk and 99.5% fewer files.

## 2. Quota Enforcement (Linear Constraints)

### The Problem

Dovecot quotas enforce per-user storage limits across multiple mailbox folders. The quota system tracks both message count and storage bytes. Quota rules define per-folder allowances that must sum to within the global limit. This is a linear constraint satisfaction problem.

### The Formula

Let $Q$ be the global quota limit (bytes), and $q_i$ be the additional allowance for folder $i$. The effective limit for folder $i$:

$$\text{limit}_i = Q + q_i$$

The global constraint requires:

$$\sum_{j=1}^{k} \text{used}_j \leq Q + \sum_{j \in \text{bonus}} q_j$$

Wait -- Dovecot quota rules work differently. The `+` prefix means "add to global limit," so:

$$Q_{effective} = Q_{base} + \sum_{i \in \text{bonus\_folders}} q_i$$

The constraint is:

$$\sum_{j=1}^{k} \text{used}_j \leq Q_{effective}$$

With `quota_grace` fraction $g$, the hard limit becomes:

$$Q_{hard} = Q_{effective} \cdot (1 + g)$$

### Worked Examples

**Example: Quota configuration analysis.**

```
quota_rule = *:storage=1G
quota_rule2 = Trash:storage=+100M
quota_rule3 = INBOX:storage=+200M
quota_grace = 10%%
```

Effective quota: $Q_{effective} = 1024 + 100 + 200 = 1324$ MB

With grace: $Q_{hard} = 1324 \times 1.10 = 1456.4$ MB

A user with 1300 MB total usage is within quota. At 1400 MB, they are over the effective limit but within grace. At 1457 MB, delivery is rejected with "552 Mailbox full."

## 3. Full-Text Search Indexing (Inverted Index)

### The Problem

Dovecot FTS builds an inverted index mapping terms to message UIDs. For $n$ messages with vocabulary size $V$ and average $t$ unique terms per message, the index must support fast boolean queries (AND, OR, NOT) across the corpus.

### The Formula

Inverted index size estimate:

$$I = V \cdot (h + \bar{p} \cdot b_{uid})$$

where $h$ is the per-term hash/pointer overhead (~32 bytes), $\bar{p}$ is the average posting list length (number of messages containing the term), and $b_{uid}$ is bytes per UID entry (4-8 bytes with delta encoding).

Average posting list length:

$$\bar{p} = \frac{n \cdot t}{V}$$

Query cost for a $k$-term AND query with posting lists of lengths $p_1 \leq p_2 \leq \cdots \leq p_k$:

$$T_{AND} = O\left(p_1 \cdot k\right) \quad \text{(iterate shortest list, probe others)}$$

With skip lists or galloping search:

$$T_{AND} = O\left(p_1 \cdot \log\left(\frac{p_k}{p_1}\right) \cdot k\right)$$

### Worked Examples

**Example: Index sizing for a 50,000-message mailbox.**

Parameters: $n = 50{,}000$ messages, $t = 200$ unique terms per message (after stemming/stopwords), $V = 100{,}000$ vocabulary size.

Average posting list: $\bar{p} = \frac{50{,}000 \times 200}{100{,}000} = 100$ messages per term.

Index size: $I = 100{,}000 \times (32 + 100 \times 6) = 100{,}000 \times 632 = 60.3$ MB.

That is approximately 6% of a 1 GB mailbox -- typical FTS overhead.

Search for "quarterly AND report": if "quarterly" appears in 500 messages and "report" in 2,000:

$$T_{AND} = O(500 \times \log(4) \times 2) = O(2{,}000) \text{ comparisons}$$

Sub-millisecond on modern hardware.

## 4. Replication Conflict Resolution (Vector Clocks)

### The Problem

Dovecot dsync replication must handle concurrent modifications on two replicas. When the same message is flagged on both sides between sync cycles, or different messages are expunged, dsync must detect and resolve conflicts. Dovecot uses per-mailbox GUIDs and modification sequences (modseq) as logical clocks.

### The Formula

Each replica maintains a modification sequence counter $m_A$, $m_B$. After sync, both replicas agree on a synchronized state $S$. A conflict exists when:

$$\Delta_A = \{changes \mid m > m_{sync}\} \neq \emptyset \;\land\; \Delta_B = \{changes \mid m > m_{sync}\} \neq \emptyset$$

and the changes overlap (same message UID). Resolution rules:

$$\text{resolve}(op_A, op_B) = \begin{cases} op_A & \text{if } \text{type}(op_A) = \text{expunge} \\ op_B & \text{if } \text{type}(op_B) = \text{expunge} \\ op_A \cup op_B & \text{if both are flag changes (merge)} \end{cases}$$

Expunge always wins (you cannot un-delete). Flag changes merge (union of all flags set on either side).

### Worked Examples

**Example: Flag conflict on two replicas.**

Initial state: message UID 42, flags = `\Seen`.

Replica A: user adds `\Flagged` -- flags become `\Seen \Flagged`, $m_A = 15$.
Replica B: user adds `\Answered` -- flags become `\Seen \Answered`, $m_B = 12$.

dsync detects overlap on UID 42. Resolution:

$$\text{flags} = \{\texttt{\textbackslash Seen}\} \cup \{\texttt{\textbackslash Flagged}\} \cup \{\texttt{\textbackslash Answered}\} = \{\texttt{\textbackslash Seen, \textbackslash Flagged, \textbackslash Answered}\}$$

Both replicas converge to the merged flag set.

## 5. Connection Concurrency (Little's Law)

### The Problem

Dovecot must size its process pool to handle concurrent IMAP connections. Each authenticated user holds a persistent IMAP connection (often idle via IDLE command). The number of active processes determines memory usage and responsiveness.

### The Formula

Little's Law relates concurrent connections $L$, arrival rate $\lambda$, and average session duration $W$:

$$L = \lambda \cdot W$$

Memory per Dovecot IMAP process: approximately 2-10 MB depending on mailbox size and index caching. Total memory:

$$M = L \cdot m_{proc}$$

For $L$ concurrent connections with `service_count = 0` (persistent processes):

$$\text{Processes} = L$$

### Worked Examples

**Example: Sizing for 5,000 users.**

Peak concurrent connections: 60% of users = 3,000. Average IMAP process memory: 5 MB.

$$M = 3{,}000 \times 5 \text{ MB} = 15 \text{ GB RAM}$$

With `imap_idle_notify_interval = 120s` and `imap_hibernate_timeout = 30s`, idle connections release their process, reducing active processes to ~500 during off-peak:

$$M_{idle} = 500 \times 5 \text{ MB} = 2.5 \text{ GB}$$

The hibernation feature provides a 6x memory reduction for idle-heavy workloads.

## Prerequisites

- Filesystem internals (inodes, block allocation, directory indexing)
- Inverted index data structures (posting lists, skip lists)
- Linear constraint satisfaction (quota modeling)
- Logical clocks and conflict resolution (vector clocks, CRDTs)
- Little's Law and capacity planning
- Probability and combinatorics for cache hit rate analysis
