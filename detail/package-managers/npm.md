# The Mathematics of npm — Node Package Manager Internals

> *npm manages JavaScript dependencies with a nested node_modules tree. The math covers SemVer resolution, dependency deduplication, install performance, and the node_modules disk space problem.*

---

## 1. Dependency Tree — The node_modules Problem

### The Model

npm resolves dependencies into a tree. Without deduplication, this tree grows exponentially.

### Nested vs Flat

**npm v2 (nested):**

$$\text{Total Packages} = \prod_{\text{depth levels}} \text{Dependencies per Level}$$

**npm v3+ (flat/deduped):**

$$\text{Total Packages} = |\text{Unique (name, version) pairs}|$$

### Worked Example

*"Package A depends on B@1 and C@1. Both B and C depend on D@1."*

| Strategy | Copies of D | node_modules entries |
|:---|:---:|:---:|
| Nested (npm v2) | 2 (one under B, one under C) | 5 |
| Flat (npm v3+) | 1 (hoisted to top) | 4 |

### Real-World Dependency Counts

| Project Type | Direct Deps | Transitive Deps | node_modules Size |
|:---|:---:|:---:|:---:|
| Simple CLI | 5 | 50 | 20 MiB |
| Express API | 15 | 200 | 50 MiB |
| React app (CRA) | 30 | 1,500 | 300 MiB |
| Next.js app | 20 | 800 | 200 MiB |
| Enterprise monorepo | 100 | 5,000 | 1-3 GiB |

---

## 2. SemVer Resolution

### Version Ranges

| Syntax | Meaning | Range |
|:---|:---|:---|
| `^1.2.3` | Compatible | [1.2.3, 2.0.0) |
| `~1.2.3` | Patch-level | [1.2.3, 1.3.0) |
| `>=1.2.3 <2.0.0` | Explicit range | As stated |
| `1.2.x` | Any patch | [1.2.0, 1.3.0) |
| `*` | Any version | [0.0.0, inf) |
| `1.2.3` | Exact | Only 1.2.3 |

### Resolution Algorithm

$$\text{For each package } P \text{ with constraint } C:$$

$$\text{Selected Version} = \max(v : v \in \text{Registry}(P) \land v \in C)$$

npm selects the **highest version** satisfying all constraints (maximal satisfying version).

### Conflict Resolution

When two packages need different versions of the same dependency:

$$\text{If } A \text{ needs } D@^1.0 \text{ and } B \text{ needs } D@^2.0:$$

$$\text{npm installs both: } D@1.x \text{ (nested under A)} + D@2.x \text{ (hoisted or nested under B)}$$

---

## 3. Install Performance — Package Count Impact

### Install Time Model

$$T_{install} = T_{resolve} + T_{download} + T_{extract} + T_{link} + T_{lifecycle\_scripts}$$

### Component Breakdown

| Component | Complexity | Typical Time (1000 packages) |
|:---|:---:|:---:|
| Resolution | O(P × V) | 2-5s |
| Download | O(Total Size / BW) | 10-30s |
| Extract | O(P × Avg Size) | 5-15s |
| Linking | O(P) | 1-3s |
| Lifecycle scripts | O(Scripts) | 5-60s |

### Lockfile Impact

$$T_{with\_lock} = T_{download} + T_{extract} \quad (\text{skip resolution})$$

$$T_{without\_lock} = T_{resolve} + T_{download} + T_{extract}$$

$$\text{Savings} = \frac{T_{resolve}}{T_{without\_lock}} \approx 20-40\%$$

### npm ci vs npm install

| Command | Reads | Deletes node_modules | Reproducible |
|:---|:---:|:---:|:---:|
| `npm install` | package.json | No | No |
| `npm ci` | package-lock.json | Yes | Yes |

$$T_{ci} < T_{install} \quad (\text{no resolution, no dedup calculation})$$

---

## 4. node_modules Disk Math

### File Count Explosion

$$\text{Files in node\_modules} = \sum_{\text{packages}} \text{Files per Package}$$

| Project | Packages | Files | Inodes Used |
|:---|:---:|:---:|:---:|
| Simple | 200 | 20,000 | 20,000 |
| Medium | 1,000 | 100,000 | 100,000 |
| Large (CRA) | 1,500 | 150,000 | 150,000 |
| Enterprise | 5,000 | 500,000 | 500,000 |

### Path Length Problem (Windows)

$$\text{Max Path} = 260 \text{ chars (Windows legacy)}$$

$$\text{Nested path} = \texttt{node\_modules/A/node\_modules/B/node\_modules/C/...}$$

Each nesting adds ~25 characters. At depth 10: ~250 characters.

### Deduplication Savings

$$\text{Dedup Ratio} = 1 - \frac{\text{Deduped Size}}{\text{Non-Deduped Size}}$$

| Before Dedup | After Dedup | Savings |
|:---:|:---:|:---:|
| 500 MiB | 300 MiB | 40% |
| 1 GiB | 400 MiB | 60% |
| 3 GiB | 800 MiB | 73% |

---

## 5. Registry and Network

### Registry Metadata

$$\text{Package Metadata} = \text{JSON with all versions, dependencies, dist info}$$

$$\text{Avg Metadata Size} = 5-50 \text{ KiB per package}$$

### Tarball Download

$$\text{Tarball Size} = \text{Source + Dependencies} \times \text{gzip ratio}$$

$$\text{Total Download} = \sum_{\text{packages}} \text{Tarball Size}_i$$

### npm Cache

$$\text{Cache Size} = \sum_{\text{ever installed}} \text{Tarball Size}$$

$$\text{Cache Location:} \quad \texttt{\~{}/.npm/\_cacache/}$$

| Usage Duration | Typical Cache Size |
|:---:|:---:|
| 1 month | 500 MiB - 2 GiB |
| 6 months | 2-10 GiB |
| 1 year | 5-20 GiB |

---

## 6. pnpm and Alternative Strategies

### Content-Addressable Storage

pnpm uses a global store with hard links:

$$\text{Disk Usage (pnpm)} = \text{Global Store} + \text{Hard Links (0 bytes each)}$$

$$\text{Disk Usage (npm)} = \sum_{\text{projects}} \text{node\_modules}_i$$

### Comparison

| Strategy | 10 Projects, Same Deps | Disk Usage |
|:---|:---:|:---:|
| npm (copies) | 10 × 300 MiB | 3 GiB |
| pnpm (hard links) | 300 MiB + links | 300 MiB |
| yarn PnP (no node_modules) | zip cache | 100 MiB |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\max(v : v \in C)$ | Optimization | Version selection |
| $\prod \text{deps/level}$ | Exponential | Nested tree growth |
| $|\text{unique (name,ver)}|$ | Set cardinality | Deduped package count |
| $T_{resolve} + T_{download} + T_{extract}$ | Sum | Install time |
| $1 - \frac{\text{deduped}}{\text{full}}$ | Ratio | Dedup savings |
| $\sum \text{files per package}$ | Summation | Inode usage |

---

*Every `npm install`, `package-lock.json`, and `node_modules/` directory reflects these resolution algorithms — a package manager handling the JavaScript ecosystem's uniquely deep dependency trees with deduplication, caching, and lockfiles.*
