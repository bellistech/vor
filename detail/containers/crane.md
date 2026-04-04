# The Mathematics of crane — Image Content Addressing and Layer Deduplication

> *Container images are content-addressed Merkle DAGs where each layer is identified by its cryptographic digest. crane operates on this mathematical structure, where copy operations are reference transfers, mutations create new nodes in the DAG, and flattening is a union of filesystem layers with overlay semantics.*

---

## 1. Content Addressing (Cryptographic Hash Functions)

### The Problem

Every container image layer, config, and manifest is addressed by its SHA-256 digest. This creates an immutable, verifiable content store where the address is derived from the content itself.

### The Formula

For content $C$, the content address:

$$\text{addr}(C) = \text{sha256}(C) \in \{0,1\}^{256}$$

Collision resistance (birthday bound):

$$P(\text{collision}) \approx \frac{n^2}{2^{257}}$$

where $n$ is the number of distinct objects.

Manifest digest chains content to layers:

$$\text{digest}(M) = \text{sha256}(M), \quad M = \{L_1.\text{digest}, L_2.\text{digest}, \ldots, L_k.\text{digest}, \text{config.digest}\}$$

### Worked Examples

Image with 3 layers:

| Object | Size | SHA-256 (truncated) |
|:---|:---:|:---|
| Layer 1 | 5.2 MB | sha256:a3ed95... |
| Layer 2 | 12.8 MB | sha256:7b4e56... |
| Layer 3 | 0.3 MB | sha256:e1c75a... |
| Config | 2.1 KB | sha256:93d5a2... |
| Manifest | 0.7 KB | sha256:df7042... |

Changing a single byte in Layer 3 produces an entirely new digest, which cascades to a new manifest digest.

For $n = 10^{12}$ objects in a global registry:

$$P(\text{collision}) \approx \frac{(10^{12})^2}{2^{257}} \approx 4.3 \times 10^{-54}}$$

---

## 2. Layer Deduplication (Set Theory)

### The Problem

When copying images between registries, layers that already exist at the destination need not be transferred. This is a set difference operation on layer digests.

### The Formula

Source image layers: $S = \{l_1, l_2, \ldots, l_m\}$ (by digest).

Destination existing layers: $D$.

Layers to transfer:

$$T = S \setminus D$$

Bandwidth saved:

$$\text{saved} = \sum_{l \in S \cap D} \text{size}(l)$$

Transfer efficiency:

$$\eta = 1 - \frac{\sum_{l \in T} \text{size}(l)}{\sum_{l \in S} \text{size}(l)}$$

### Worked Examples

Source: 4 layers [10MB, 25MB, 3MB, 1MB]. Destination already has layers 1 and 2 (same base image):

$$T = \{l_3, l_4\}, \quad |T| = 2$$

$$\text{saved} = 10 + 25 = 35 \text{ MB}$$

$$\eta = 1 - \frac{3 + 1}{10 + 25 + 3 + 1} = 1 - \frac{4}{39} = 0.897 = 89.7\%$$

---

## 3. Image Flattening (Union Filesystem Algebra)

### The Problem

`crane flatten` merges multiple layers into one. Container layers use overlay filesystem semantics where upper layers override lower layers, and whiteout files indicate deletions.

### The Formula

For layers $L_1, L_2, \ldots, L_n$ (bottom to top), the flattened filesystem $F$:

$$F = L_n \cup_{\text{ov}} (L_{n-1} \cup_{\text{ov}} (\cdots \cup_{\text{ov}} L_1))$$

where overlay union $\cup_{\text{ov}}$ is:

$$A \cup_{\text{ov}} B = \{f \in A \mid f \notin \text{whiteout}(B)\} \cup B$$

Final file count:

$$|F| = \left| \bigcup_{i=1}^{n} \text{files}(L_i) \right| - \left| \bigcup_{i=1}^{n} \text{whiteouts}(L_i) \right|$$

### Worked Examples

Layer 1: {/bin/sh, /etc/passwd, /app/old}
Layer 2: {/app/new, .wh.app/old} (whiteout deletes /app/old)
Layer 3: {/etc/passwd} (overwrites)

$$F = \{/bin/sh, /etc/passwd_{\text{v3}}, /app/new\}$$

$$|F| = 3, \quad \text{total unique paths across layers} = 5, \quad \text{whiteouts} = 1, \quad \text{overwrites} = 1$$

---

## 4. Rebase as Graph Surgery (Graph Theory)

### The Problem

`crane rebase` replaces the base layers of an image while preserving application layers. This is a graph surgery operation on the layer DAG.

### The Formula

Original image layers: $O = [B_1, B_2, \ldots, B_j, A_1, A_2, \ldots, A_k]$

Old base: $B_{\text{old}} = [B_1, \ldots, B_j]$

New base: $B_{\text{new}} = [B'_1, \ldots, B'_m]$

Rebased image:

$$R = B_{\text{new}} \| [A_1, A_2, \ldots, A_k]$$

$$|R| = m + k$$

This is valid when application layers $A_i$ do not depend on specific paths in the base layers that changed.

### Worked Examples

Original: ubuntu:22.04 (3 layers) + app (2 layers) = 5 layers total.

Rebase onto ubuntu:24.04 (4 layers):

$$R = [B'_1, B'_2, B'_3, B'_4, A_1, A_2] = 6 \text{ layers}$$

Size change: if old base = 78MB, new base = 82MB, app layers = 15MB:

$$\Delta S = (82 + 15) - (78 + 15) = 4 \text{ MB increase}$$

---

## 5. Multi-Architecture Manifest Index (Combinatorics)

### The Problem

A multi-arch image is a manifest index pointing to platform-specific manifests. The index is a product of OS and architecture combinations.

### The Formula

For OS set $O$ and architecture set $A$ with optional variant set $V$:

$$N_{\text{platforms}} = \sum_{(o, a) \in O \times A} |V(o, a)|$$

Manifest index size:

$$S_{\text{index}} = H + N_{\text{platforms}} \cdot E$$

where $H$ is the index header size and $E$ is per-entry size (~100 bytes).

Total storage for a multi-arch image:

$$S_{\text{total}} = S_{\text{index}} + \sum_{p \in \text{platforms}} S_{\text{image}}(p)$$

### Worked Examples

Alpine supports: linux/{amd64, arm64, arm/v6, arm/v7, 386, ppc64le, s390x} = 7 platforms:

$$N_{\text{platforms}} = 7$$

$$S_{\text{index}} = 100 + 7 \times 100 = 800 \text{ bytes}$$

If average image size is 3.5 MB per platform:

$$S_{\text{total}} = 0.8 \text{ KB} + 7 \times 3.5 \text{ MB} = 24.5 \text{ MB}$$

---

## 6. Registry Transfer Optimization (Network Theory)

### The Problem

`crane copy` performs server-side blob mounts when source and destination are in the same registry. This avoids data transfer through the client.

### The Formula

Cross-registry copy bandwidth:

$$B_{\text{cross}} = \frac{S_{\text{total}}}{B_{\text{up}}} + \frac{S_{\text{total}}}{B_{\text{down}}} = S_{\text{total}} \left(\frac{1}{B_{\text{up}}} + \frac{1}{B_{\text{down}}}\right)$$

Same-registry mount (server-side):

$$B_{\text{mount}} = N_{\text{layers}} \cdot C_{\text{api}} \approx 0$$

where $C_{\text{api}}$ is the cost of a single API call (negligible).

Speedup factor:

$$\text{speedup} = \frac{B_{\text{cross}}}{B_{\text{mount}}} \to \infty$$

### Worked Examples

Image: 150 MB, upload 10 MB/s, download 50 MB/s:

$$T_{\text{cross}} = \frac{150}{50} + \frac{150}{10} = 3 + 15 = 18\text{s}$$

Same-registry mount: 5 API calls at ~50ms each:

$$T_{\text{mount}} = 5 \times 0.05 = 0.25\text{s}$$

$$\text{speedup} = \frac{18}{0.25} = 72\times$$

---

## Prerequisites

- cryptographic-hashing, set-theory, graph-theory, combinatorics, network-theory, overlay-filesystems
