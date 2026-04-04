# The Mathematics of dracut — Initramfs Composition and Boot Dependency Resolution

> *dracut builds an initramfs by resolving a dependency graph of modules, where each module declares prerequisites and installs files into a union. The boot process itself is a state machine transitioning through hook points, with device discovery modeled as a convergence problem under timeout constraints.*

---

## 1. Module Dependency Resolution (Graph Theory)

### The Problem

Each dracut module declares dependencies on other modules via the `depends()` function. The inclusion set must be transitively closed over these dependencies.

### The Formula

Given module dependency graph $G = (M, D)$ where $D \subseteq M \times M$:

For requested module set $R \subseteq M$, the inclusion set is the transitive closure:

$$I = \text{TC}(R, D) = R \cup \bigcup_{m \in R} \text{TC}(\text{deps}(m), D)$$

The build fails if:

$$\exists m \in I : \text{check}(m) = \text{false}$$

Module count:

$$|I| \geq |R|, \quad |I| \leq |M|$$

### Worked Examples

Requested: $R = \{\text{crypt}\}$.

Dependency chain:
- crypt depends on {dm, base}
- dm depends on {base, udev-rules}
- base depends on {} (leaf)
- udev-rules depends on {base}

$$I = \{\text{crypt}, \text{dm}, \text{base}, \text{udev-rules}\}$$

$$|I| = 4, \quad |R| = 1$$

Transitive closure expanded the single request into 4 modules.

---

## 2. Initramfs Size Estimation (Combinatorics)

### The Problem

Initramfs size depends on included modules, kernel modules, and firmware. Hostonly mode minimizes size by including only hardware-specific drivers.

### The Formula

Total uncompressed size:

$$S_{\text{raw}} = S_{\text{base}} + \sum_{m \in I} S_m + \sum_{k \in K} S_k + \sum_{f \in F} S_f$$

where $I$ = dracut modules, $K$ = kernel modules, $F$ = firmware files.

Compressed size with ratio $\gamma$:

$$S_{\text{compressed}} = \gamma \cdot S_{\text{raw}}$$

Typical compression ratios:

| Algorithm | $\gamma$ | Decompress Speed |
|:---|:---:|:---:|
| gzip | 0.35 | 250 MB/s |
| xz | 0.25 | 150 MB/s |
| zstd | 0.30 | 500 MB/s |
| lz4 | 0.45 | 800 MB/s |

### Worked Examples

Hostonly initramfs: $S_{\text{base}} = 5$ MB, 4 modules at 1 MB each, 30 kernel modules at 100 KB each, 2 firmware blobs at 500 KB each:

$$S_{\text{raw}} = 5 + 4 + 3 + 1 = 13 \text{ MB}$$

With zstd ($\gamma = 0.30$):

$$S_{\text{compressed}} = 0.30 \times 13 = 3.9 \text{ MB}$$

Generic initramfs: same base + 20 modules + 500 kernel modules + 50 firmware:

$$S_{\text{raw}} = 5 + 20 + 50 + 25 = 100 \text{ MB}$$

$$S_{\text{compressed}} = 0.30 \times 100 = 30 \text{ MB}$$

---

## 3. Boot State Machine (Automata Theory)

### The Problem

The dracut boot process transitions through a series of hook points. Each transition requires all hooks at that stage to complete successfully before advancing.

### The Formula

The boot state machine $A = (Q, \Sigma, \delta, q_0, F)$:

$$Q = \{q_{\text{cmdline}}, q_{\text{pre-udev}}, q_{\text{pre-trigger}}, q_{\text{initqueue}}, q_{\text{pre-mount}}, q_{\text{mount}}, q_{\text{pre-pivot}}, q_{\text{cleanup}}, q_{\text{root}}, q_{\text{emergency}}\}$$

Transition function:

$$\delta(q_i) = \begin{cases}
q_{i+1} & \text{if all hooks at } q_i \text{ succeed} \\
q_{\text{emergency}} & \text{if any hook fails and } \text{rd.shell}=1 \\
q_{\text{halt}} & \text{if any hook fails and } \text{rd.shell}=0
\end{cases}$$

Total boot time through the state machine:

$$T_{\text{boot}} = \sum_{q \in \text{path}(q_0, q_{\text{root}})} \left( \sum_{h \in H(q)} T_h \right)$$

### Worked Examples

Normal boot path with hook durations:

| State | Hooks | Duration |
|:---|:---:|:---:|
| cmdline | 3 | 0.1s |
| pre-udev | 2 | 0.05s |
| pre-trigger | 1 | 0.02s |
| initqueue | 5 (device wait) | 2.0s |
| pre-mount | 2 | 0.1s |
| mount | 1 | 0.5s |
| pre-pivot | 2 | 0.1s |
| cleanup | 1 | 0.05s |

$$T_{\text{boot}} = 0.1 + 0.05 + 0.02 + 2.0 + 0.1 + 0.5 + 0.1 + 0.05 = 2.92\text{s}$$

The initqueue (device wait) dominates boot time.

---

## 4. Device Discovery Convergence (Probability Theory)

### The Problem

During the initqueue phase, dracut waits for required devices to appear. Device enumeration is asynchronous, with a timeout $T_{\max}$. The question is: what is the probability all devices appear before timeout?

### The Formula

For $n$ required devices with independent appearance times $X_i \sim \text{Exp}(\lambda_i)$:

$$P(\text{all devices before } T) = \prod_{i=1}^{n} (1 - e^{-\lambda_i T})$$

The expected time for the last device:

$$E[\max(X_1, \ldots, X_n)] = \sum_{k=1}^{n} (-1)^{k+1} \binom{n}{k} \frac{1}{\sum_{j \in S_k} \lambda_j}$$

For identical rates $\lambda$:

$$E[\max(X_1, \ldots, X_n)] = \frac{H_n}{\lambda} = \frac{1}{\lambda} \sum_{k=1}^{n} \frac{1}{k}$$

### Worked Examples

3 block devices, each appearing with rate $\lambda = 2$/s (mean 0.5s):

$$E[\max] = \frac{H_3}{2} = \frac{1 + 0.5 + 0.333}{2} = \frac{1.833}{2} = 0.917\text{s}$$

Probability all appear within $T = 3$s:

$$P = (1 - e^{-2 \times 3})^3 = (1 - 0.0025)^3 = 0.9975^3 = 0.9925$$

With default timeout $T_{\max} = 30$s:

$$P = (1 - e^{-60})^3 \approx 1.0$$

---

## 5. Hook Priority Scheduling (Scheduling Theory)

### The Problem

Each hook point runs scripts in priority order (numeric prefix). Scripts at the same priority level may run concurrently. This is a priority-based scheduling problem.

### The Formula

For hook point $q$ with scripts $H(q) = \{(p_1, s_1), (p_2, s_2), \ldots\}$:

Execution groups by priority:

$$G_k = \{s \in H(q) \mid p(s) = k\}$$

Sequential group execution:

$$T(q) = \sum_{k \in \text{priorities}} \max_{s \in G_k} T(s)$$

If all scripts run sequentially (single-threaded):

$$T(q) = \sum_{s \in H(q)} T(s)$$

### Worked Examples

Pre-mount hooks: (10, crypto-unlock, 2.0s), (50, lvm-activate, 0.5s), (50, check-fs, 0.3s), (90, mount-root, 0.5s).

Concurrent within priority groups:

$$T = 2.0 + \max(0.5, 0.3) + 0.5 = 3.0\text{s}$$

Sequential execution:

$$T = 2.0 + 0.5 + 0.3 + 0.5 = 3.3\text{s}$$

Concurrency saves $0.3$s.

---

## 6. Compression Trade-offs (Information Theory)

### The Problem

Choosing initramfs compression involves a trade-off between compressed size (affects disk I/O at boot) and decompression speed (affects CPU time at boot).

### The Formula

Total load time:

$$T_{\text{load}} = \frac{S_{\text{compressed}}}{B_{\text{disk}}} + \frac{S_{\text{raw}}}{D_{\text{speed}}}$$

where $B_{\text{disk}}$ is disk read bandwidth and $D_{\text{speed}}$ is decompression throughput.

Optimal compression minimizes $T_{\text{load}}$:

$$\gamma^* = \arg\min_{\gamma} \left( \frac{\gamma \cdot S_{\text{raw}}}{B_{\text{disk}}} + \frac{S_{\text{raw}}}{D(\gamma)} \right)$$

### Worked Examples

$S_{\text{raw}} = 50$ MB, $B_{\text{disk}} = 500$ MB/s (SSD):

| Algorithm | $\gamma$ | Compressed | Disk I/O | Decompress | Total |
|:---|:---:|:---:|:---:|:---:|:---:|
| gzip | 0.35 | 17.5 MB | 35ms | 200ms | 235ms |
| xz | 0.25 | 12.5 MB | 25ms | 333ms | 358ms |
| zstd | 0.30 | 15.0 MB | 30ms | 100ms | 130ms |
| lz4 | 0.45 | 22.5 MB | 45ms | 62ms | 107ms |

On SSD, lz4 wins. On HDD ($B_{\text{disk}} = 100$ MB/s):

| Algorithm | Disk I/O | Decompress | Total |
|:---|:---:|:---:|:---:|
| gzip | 175ms | 200ms | 375ms |
| zstd | 150ms | 100ms | 250ms |
| lz4 | 225ms | 62ms | 287ms |

On HDD, zstd wins due to better compression reducing I/O time.

---

## Prerequisites

- graph-theory, automata-theory, probability-theory, scheduling-theory, information-theory, combinatorics
