# The Mathematics of Skopeo — Content-Addressed Distribution and Transport Algebra

> *Skopeo operates on the mathematical properties of content-addressed storage, where images are merkle DAGs distributed across registry endpoints through transport morphisms that preserve digest integrity across format conversions and network boundaries.*

---

## 1. Content-Addressed Storage (Hash Theory)

### Digest Function

Every OCI/Docker blob is identified by its cryptographic digest:

$$d: \mathcal{B} \rightarrow \mathcal{D}$$

$$d(b) = \text{algo}:\text{hex}(H(b))$$

Where $H$ is typically SHA-256:

$$d(b) = \text{sha256}:\text{hex}(\text{SHA-256}(b))$$

### Integrity Properties

**Collision resistance:** For any two distinct blobs $b_1 \neq b_2$:

$$P(d(b_1) = d(b_2)) \leq 2^{-128} \quad \text{(birthday bound)}$$

For $n$ blobs in a registry, probability of any collision:

$$P(\text{collision}) \leq \frac{n^2}{2^{257}} \approx \frac{n^2}{2^{257}}$$

At $n = 10^{12}$ (1 trillion blobs):

$$P \leq \frac{10^{24}}{2^{257}} \approx 10^{-53}$$

### Verification on Copy

Skopeo verifies integrity during every copy:

$$\text{verify}(b, d_{\text{expected}}) = (d(b) \stackrel{?}{=} d_{\text{expected}})$$

If verification fails, the copy aborts. This provides end-to-end integrity without trusting intermediate transport.

---

## 2. Transport System as Category (Category Theory)

### Transport Morphisms

Skopeo's transports form a category $\textbf{Transport}$:

**Objects:** Image storage backends

$$\text{Ob}(\textbf{Transport}) = \{\text{docker}, \text{docker-archive}, \text{oci}, \text{oci-archive}, \text{dir}, \text{containers-storage}\}$$

**Morphisms:** Copy operations between transports

$$\text{copy}: T_{\text{src}} \rightarrow T_{\text{dst}}$$

### Composition

Copies compose transitively:

$$\text{copy}_{A \to C} = \text{copy}_{B \to C} \circ \text{copy}_{A \to B}$$

However, direct copy is preferred (fewer format conversions):

$$\text{cost}(\text{copy}_{A \to C}) \leq \text{cost}(\text{copy}_{A \to B}) + \text{cost}(\text{copy}_{B \to C})$$

### Transport Compatibility Matrix

Not all transport pairs preserve all features:

| Source \ Dest | docker | docker-archive | oci | dir |
|:---|:---:|:---:|:---:|:---:|
| **docker** | Full | Full | Convert | Full |
| **docker-archive** | Full | Full | Convert | Full |
| **oci** | Convert | Convert | Full | Full |
| **dir** | Full | Full | Full | Full |

Where "Convert" means manifest format translation is applied.

---

## 3. Manifest Format Algebra (Algebra)

### Format Conversion

Skopeo converts between manifest formats:

$$\phi: M_{\text{docker}} \leftrightarrow M_{\text{oci}}$$

Docker manifest v2s2:

$$M_D = (\text{schemaVersion}: 2, \text{mediaType}: \text{docker.manifest.v2}, \text{config}, \text{layers}[])$$

OCI manifest:

$$M_O = (\text{schemaVersion}: 2, \text{mediaType}: \text{oci.image.manifest.v1}, \text{config}, \text{layers}[])$$

The conversion $\phi$ preserves:
- Layer digests (content unchanged)
- Configuration content (re-serialized)
- Annotation mapping

### Media Type Mapping

| Docker v2 | OCI v1 |
|:---|:---|
| `docker.manifest.v2+json` | `oci.image.manifest.v1+json` |
| `docker.manifest.list.v2+json` | `oci.image.index.v1+json` |
| `docker.layer.v1.tar.gzip` | `oci.image.layer.v1.tar+gzip` |
| `docker.container.image.v1+json` | `oci.image.config.v1+json` |

### Invariant Under Conversion

The layer content digests are invariant under format conversion:

$$\forall l \in \text{layers}: d(l_{\text{docker}}) = d(l_{\text{oci}})$$

Only the manifest and config digests change (different JSON serialization).

---

## 4. Registry Sync as Graph Problem (Graph Theory)

### Mirror Topology

A registry mirroring setup is a directed graph:

$$G_{\text{mirror}} = (R, E)$$

Where $R$ = registries and $E$ = sync edges.

Common topologies:

**Hub-and-spoke:**
$$E = \{(r_{\text{source}}, r_i) : i = 1, \ldots, k\}$$
$$\text{sync cost} = k \times S_{\text{image}}$$

**Chain:**
$$E = \{(r_i, r_{i+1}) : i = 1, \ldots, k-1\}$$
$$\text{sync latency} = (k-1) \times T_{\text{copy}}$$

**Mesh (full replication):**
$$|E| = k(k-1)$$
$$\text{impractical for large } k$$

### Incremental Sync

Skopeo sync transfers only new/changed images:

$$\Delta_{\text{sync}} = \text{tags}(R_{\text{src}}) \setminus \text{tags}(R_{\text{dst}})$$

By digest comparison:

$$\text{skip}(t) \iff d(\text{manifest}(R_{\text{src}}, t)) = d(\text{manifest}(R_{\text{dst}}, t))$$

Transfer volume:

$$V_{\text{sync}} = \sum_{t \in \Delta_{\text{sync}}} \sum_{l \in \text{layers}(t)} \text{size}(l) \times \mathbb{1}[l \notin R_{\text{dst}}]$$

Layer-level dedup across tags further reduces $V_{\text{sync}}$.

---

## 5. Bandwidth Optimization (Information Theory)

### Skopeo Inspect vs Pull

Inspect downloads only the manifest and config:

$$S_{\text{inspect}} = S_{\text{manifest}} + S_{\text{config}} \approx 5\text{-}20\text{KB}$$

Full pull downloads all layers:

$$S_{\text{pull}} = S_{\text{manifest}} + S_{\text{config}} + \sum_{i=1}^{n} S_{\text{layer}_i} \approx 10\text{-}1000\text{MB}$$

Bandwidth savings:

$$\text{savings} = 1 - \frac{S_{\text{inspect}}}{S_{\text{pull}}} \approx 1 - \frac{0.02}{100} = 99.98\%$$

### Multi-Architecture Image Analysis

For an image index with $p$ platforms:

$$S_{\text{inspect\_all}} = S_{\text{index}} + \sum_{i=1}^{p} (S_{\text{manifest}_i} + S_{\text{config}_i})$$

$$S_{\text{inspect\_all}} \approx p \times 15\text{KB}$$

vs pulling all platforms:

$$S_{\text{pull\_all}} = \sum_{i=1}^{p} S_{\text{image}_i}$$

For 5 platforms with 100MB average: $S_{\text{pull\_all}} = 500\text{MB}$.

---

## 6. Tag Listing and Pagination (Combinatorics)

### Registry Tag Space

A repository with $n$ tags across $v$ versions and $p$ platforms:

$$|\text{tags}| = v \times p + |\text{aliases}|$$

Common patterns:

$$\text{tags} = \{v_i, v_i\text{-}p_j, \text{latest}\} \quad \text{for } i = 1, \ldots, v; \; j = 1, \ldots, p$$

### Pagination

The OCI Distribution API returns paginated tag lists:

$$\text{pages} = \lceil n / k \rceil$$

Where $k$ is the page size (default: 100 in most registries).

Total API calls for `list-tags`:

$$\text{calls} = 1 + \lceil (n - k) / k \rceil = \lceil n / k \rceil$$

For Docker Hub's `nginx` repository (~600 tags):

$$\text{calls} = \lceil 600 / 100 \rceil = 6$$

---

## 7. Deletion and Garbage Collection (Reference Counting)

### Blob Reference Graph

Registry blobs form a reference graph:

$$G_{\text{ref}} = (B, R)$$

Where $R$ = references from manifests/configs to blobs.

A blob is deletable when its reference count drops to zero:

$$\text{deletable}(b) \iff |\{m \in M : b \in \text{refs}(m)\}| = 0$$

### Garbage Collection Phases

**Mark phase:** Traverse all live manifests, mark referenced blobs:

$$\text{marked} = \bigcup_{m \in M_{\text{live}}} \text{refs}(m)$$

$$T_{\text{mark}} = O(|M| \times \bar{L})$$

Where $\bar{L}$ is the average number of layers per manifest.

**Sweep phase:** Delete unmarked blobs:

$$\text{deletable} = B \setminus \text{marked}$$

$$T_{\text{sweep}} = O(|B|)$$

### Space Recovery

After deleting a manifest with $n$ layers:

$$\text{recovered} = \sum_{l \in \text{layers}} \text{size}(l) \times \mathbb{1}[\text{refcount}(l) = 0]$$

Shared layers are not recovered until all referencing manifests are deleted.

---

## Prerequisites

hash-theory, category-theory, graph-theory, information-theory, reference-counting

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Inspect (manifest + config) | $O(1)$ + network RTT | $O(1)$ — KB range |
| Copy (single image) | $O(S)$ — S = image size | $O(S)$ — temporary |
| Copy (with dedup) | $O(S \times (1-h))$ — h = hit rate | $O(\Delta S)$ — new blobs |
| Sync (incremental) | $O(\|\Delta\| \times \bar{S})$ | $O(\|\Delta\| \times \bar{S})$ |
| List tags | $O(\lceil n/k \rceil)$ — API calls | $O(n)$ — tag list |
| Delete manifest | $O(1)$ — single API call | $O(1)$ |
| Digest verification | $O(S)$ — SHA-256 of blob | $O(1)$ |
