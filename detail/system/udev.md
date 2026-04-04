# The Mathematics of udev — Rule Matching and Device Enumeration

> *udev processes device events through a rule evaluation engine that is fundamentally a pattern matching automaton over sysfs attribute trees. Device enumeration is a tree traversal, rule ordering is a priority queue, and persistent naming is a bijection from device attributes to filesystem paths.*

---

## 1. Rule Matching as Predicate Evaluation (Predicate Logic)

### The Problem

Each udev rule is a conjunction of match predicates. When a device event occurs, the rule engine evaluates every rule file in lexical order, applying actions from all matching rules.

### The Formula

A rule $r$ matches device $d$ when all match keys are satisfied:

$$\text{match}(r, d) = \bigwedge_{i=1}^{k} P_i(d)$$

where $P_i$ are predicates like:

$$P_{\text{KERNEL}}(d) = \text{glob}(d.\text{name}, r.\text{KERNEL})$$

$$P_{\text{ATTR}}(d) = d.\text{sysfs}[r.\text{attr\_key}] = r.\text{attr\_value}$$

$$P_{\text{ATTRS}}(d) = \exists a \in \text{ancestors}(d) : a.\text{sysfs}[r.\text{attr\_key}] = r.\text{attr\_value}$$

The total rules evaluated per event:

$$W(e) = \sum_{f \in F} |R_f|$$

where $F$ is the set of rule files and $|R_f|$ is the rule count in file $f$.

### Worked Examples

Rule: `SUBSYSTEM=="tty", ATTRS{idVendor}=="0403", ATTRS{idProduct}=="6001", SYMLINK+="ftdi"`

Device `/dev/ttyUSB0` with parent chain:

| Level | SUBSYSTEM | idVendor | idProduct |
|:---|:---|:---|:---|
| ttyUSB0 | tty | - | - |
| 1-2:1.0 | usb | - | - |
| 1-2 | usb | 0403 | 6001 |

$$\text{match} = (\text{tty} == \text{tty}) \wedge (0403 == 0403) \wedge (6001 == 6001) = \text{true}$$

Note: `ATTRS` matches across different ancestor levels simultaneously.

---

## 2. Device Tree Enumeration (Tree Traversal)

### The Problem

The sysfs filesystem is a tree rooted at `/sys`. Device enumeration walks this tree to discover all devices matching a subsystem or attribute.

### The Formula

The sysfs device tree $T = (V, E)$ where:

$$V = \{d_{\text{root}}, d_1, d_2, \ldots, d_n\}$$

$$E = \{(d_i, d_j) \mid d_i = \text{parent}(d_j)\}$$

`udevadm info --attribute-walk` performs a root-ward traversal:

$$\text{walk}(d) = [d, \text{parent}(d), \text{parent}^2(d), \ldots, d_{\text{root}}]$$

The depth of device $d$:

$$\text{depth}(d) = |\text{walk}(d)| - 1$$

### Worked Examples

Path: `/sys/devices/pci0000:00/0000:00:14.0/usb1/1-2/1-2:1.0/ttyUSB0/tty/ttyUSB0`

Walk produces 8 nodes (depth = 7):

$$[\text{ttyUSB0}, \text{tty}, \text{ttyUSB0}, \text{1-2:1.0}, \text{1-2}, \text{usb1}, \text{0000:00:14.0}, \text{pci0000:00}]$$

Each node exposes different attributes for matching rules.

---

## 3. Rule File Ordering (Priority Queue)

### The Problem

Rule files are processed in lexical order by filename. Within a file, rules execute top to bottom. The numeric prefix determines priority.

### The Formula

Given rule files $F = \{f_1, f_2, \ldots, f_m\}$ with names $n_i$:

$$\text{order}(F) = \text{sort}(F, \leq_{\text{lex}})$$

Effective rule sequence:

$$R_{\text{total}} = R_{f_{\sigma(1)}} \| R_{f_{\sigma(2)}} \| \cdots \| R_{f_{\sigma(m)}}$$

where $\sigma$ is the lexicographic permutation and $\|$ denotes concatenation.

Override semantics: `/etc/udev/rules.d/` files shadow `/usr/lib/udev/rules.d/` files with the same name.

$$R_{\text{effective}}(n) = \begin{cases}
R_{\text{etc}}(n) & \text{if } n \in F_{\text{etc}} \\
R_{\text{lib}}(n) & \text{otherwise}
\end{cases}$$

### Worked Examples

Files: `10-local.rules`, `50-udev-default.rules`, `70-net.rules`, `99-custom.rules`.

Processing order:

$$[10\text{-local}, 50\text{-udev-default}, 70\text{-net}, 99\text{-custom}]$$

If `/etc/udev/rules.d/70-net.rules` exists, it completely replaces `/usr/lib/udev/rules.d/70-net.rules`.

---

## 4. Persistent Naming as Bijection (Set Theory)

### The Problem

Persistent device naming creates a bijection between immutable device attributes and stable filesystem paths, avoiding the non-deterministic kernel naming (sda, sdb...).

### The Formula

The persistent naming function:

$$f: \mathcal{A} \to \mathcal{P}$$

where $\mathcal{A}$ is the attribute space and $\mathcal{P}$ is the path space.

For `by-id`:
$$f_{\text{id}}(d) = \text{/dev/disk/by-id/} \| d.\text{bus} \text{-} d.\text{vendor} \text{\_} d.\text{model} \text{\_} d.\text{serial}$$

For `by-uuid`:
$$f_{\text{uuid}}(d) = \text{/dev/disk/by-uuid/} \| d.\text{fs\_uuid}$$

Bijection property (no collisions):

$$\forall d_1, d_2 \in D, \; d_1 \neq d_2 \implies f(d_1) \neq f(d_2)$$

### Worked Examples

Two identical USB drives inserted:

| Drive | Serial | UUID | by-id Path | by-uuid Path |
|:---|:---|:---|:---|:---|
| sdb1 | AAA111 | abc-123 | usb-SanDisk_AAA111-part1 | abc-123 |
| sdc1 | BBB222 | def-456 | usb-SanDisk_BBB222-part1 | def-456 |

Both $f_{\text{id}}$ and $f_{\text{uuid}}$ produce unique paths regardless of kernel enumeration order.

---

## 5. Glob Pattern Matching Complexity (Automata Theory)

### The Problem

udev KERNEL and other match keys use glob patterns (`*`, `?`, `[]`). Each match is a string comparison against a compiled pattern.

### The Formula

Glob-to-regex conversion:

$$\text{glob}(p) \to \text{regex}(p')$$

where `*` $\to$ `.*`, `?` $\to$ `.`, `[abc]` $\to$ `[abc]`.

Pattern matching complexity:

$$T(\text{glob}) = O(|p| \cdot |s|)$$

where $|p|$ is pattern length and $|s|$ is string length.

For $n$ rules with patterns and $m$ events:

$$T_{\text{total}} = O(m \cdot n \cdot |p_{\max}| \cdot |s_{\max}|)$$

### Worked Examples

Pattern `sd[a-z]` against device names:

| Device | Match | Operations |
|:---|:---:|:---:|
| sda | true | 3 comparisons |
| sdb1 | false | 4 comparisons (length mismatch at char 4) |
| nvme0n1 | false | 1 comparison (fails at 'n' vs 's') |

Early termination makes average case much faster than worst case.

---

## 6. Event Processing Throughput (Queueing Theory)

### The Problem

udev processes events from a netlink socket. Events can arrive in bursts (e.g., boot, USB hub insertion). The worker pool processes events with ordering constraints per device.

### The Formula

Event processing as M/M/c queue with $c$ worker threads:

$$\rho = \frac{\lambda}{c \cdot \mu}$$

where $\lambda$ = event arrival rate, $\mu$ = processing rate per worker.

Per-device ordering constraint reduces effective parallelism. For $d$ distinct devices in a burst of $n$ events:

$$c_{\text{effective}} = \min(c, d)$$

Burst processing time:

$$T_{\text{burst}} = \frac{n}{c_{\text{effective}} \cdot \mu} + \frac{n_{\text{settle}}}{\mu}$$

where $n_{\text{settle}}$ accounts for `udevadm settle` serialization.

### Worked Examples

Boot event burst: $n = 500$ events, $d = 200$ distinct devices, $c = 8$ workers, $\mu = 50$ events/s per worker:

$$c_{\text{effective}} = \min(8, 200) = 8$$

$$T_{\text{burst}} = \frac{500}{8 \times 50} = 1.25\text{s}$$

With `settle` adding 0.5s:

$$T_{\text{total}} = 1.25 + 0.5 = 1.75\text{s}$$

---

## Prerequisites

- predicate-logic, tree-traversal, set-theory, automata-theory, queueing-theory, bijection
