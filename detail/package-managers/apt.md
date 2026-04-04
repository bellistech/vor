# The Mathematics of APT — Debian Package Management Internals

> *APT (Advanced Package Tool) resolves dependency graphs, manages repository indices, and handles package verification. The math covers dependency resolution (SAT solving), download optimization, and cache management.*

---

## 1. Dependency Resolution — Boolean Satisfiability

### The Model

Package dependencies form a directed graph. Installing a package requires satisfying all its dependencies without conflicts. This is a **Boolean satisfiability (SAT)** problem.

### Dependency Types

$$\text{Depends: } A \Rightarrow B \quad (\text{A requires B})$$

$$\text{Pre-Depends: } A \Rightarrow B \quad (\text{B must be configured before A unpacks})$$

$$\text{Conflicts: } A \Rightarrow \neg B \quad (\text{A and B cannot coexist})$$

$$\text{Recommends: } A \rightarrow B \quad (\text{weak dependency, optional})$$

$$\text{Breaks: } A \Rightarrow \neg B_{version} \quad (\text{A breaks specific versions of B})$$

### SAT Complexity

$$\text{General dependency resolution is NP-complete}$$

In practice, APT uses heuristics (not full SAT solver) because:
- Dependency graphs are sparse (low connectivity)
- Most packages have 3-10 dependencies
- Virtual packages and alternatives are bounded

### Dependency Graph Size

| Distribution | Packages | Dependency Edges | Avg Dependencies/Package |
|:---|:---:|:---:|:---:|
| Debian stable | ~60,000 | ~300,000 | ~5 |
| Ubuntu LTS | ~80,000 | ~400,000 | ~5 |
| Debian testing | ~65,000 | ~350,000 | ~5.4 |

### Transitive Closure

$$\text{Total packages needed} = |\text{TransitiveClosure}(P)|$$

$$|\text{TC}(P)| = |P| + \sum_{d \in \text{Deps}(P)} |\text{TC}(d)|$$

| Package | Direct Deps | Transitive Deps | Total Installed |
|:---|:---:|:---:|:---:|
| `curl` | 5 | 15 | 20 |
| `nginx` | 10 | 40 | 50 |
| `postgresql` | 15 | 60 | 75 |
| `gnome-desktop` | 50 | 500 | 550 |

---

## 2. Repository Index — Download and Storage

### Index Size

$$\text{Packages.gz Size} \approx \text{Package Count} \times 500 \text{ bytes (compressed)}$$

$$\text{Packages Size (uncompressed)} \approx \text{Package Count} \times 2 \text{ KiB}$$

| Repository | Packages | Packages.gz | Packages (uncompressed) |
|:---|:---:|:---:|:---:|
| main (amd64) | 30,000 | 15 MiB | 60 MiB |
| universe | 45,000 | 22 MiB | 90 MiB |
| Total (all components) | 80,000 | 40 MiB | 160 MiB |

### apt update Bandwidth

$$\text{Update BW} = \sum_{\text{sources}} (\text{Packages.gz} + \text{Release} + \text{Translation})$$

With InRelease (combined Release+signature):

$$\text{Typical update} \approx 20-60 \text{ MiB (first time)} \quad | \quad 1-5 \text{ MiB (incremental/pdiff)}$$

### Incremental Updates (PDiff)

$$\text{PDiff Size} \approx \text{Changed Entries} \times 500 \text{ bytes}$$

$$\text{Worth using if:} \quad \text{PDiff Size} < \text{Full Packages.gz Size}$$

---

## 3. Package Verification — Cryptographic Chain

### The Model

APT verifies packages through a chain of trust:

$$\text{GPG Key} \rightarrow \text{Release (signed)} \rightarrow \text{Packages (hashed)} \rightarrow \text{.deb (hashed)}$$

### Hash Verification

$$\text{Integrity} = \text{SHA256}(\text{downloaded .deb}) = \text{SHA256 in Packages index}$$

### Verification Costs

| Operation | Time | Purpose |
|:---|:---:|:---|
| GPG signature verify | ~10 ms | Authenticate Release file |
| SHA256 of Packages index | ~50 ms | Verify index integrity |
| SHA256 per .deb | ~1-100 ms | Verify package integrity |
| MD5 (legacy, deprecated) | ~0.5-50 ms | Backward compatibility |

### Total Verification Time

$$T_{verify} = T_{gpg} + T_{index\_hash} + \sum_{i=1}^{n} T_{deb\_hash_i}$$

For installing 50 packages averaging 5 MiB each:

$$T = 10 + 50 + 50 \times 5 = 310 \text{ ms}$$

---

## 4. Download Optimization

### Parallel Downloads

APT2 supports parallel downloads (default: 5 concurrent):

$$T_{download} = \frac{\sum \text{Package Sizes}}{\min(\text{Bandwidth}, n \times \text{Per-Connection BW})}$$

Where $n$ = parallel connections.

### Mirror Selection

$$T_{mirror} = T_{DNS} + T_{TCP} + T_{TLS} + T_{TTFB}$$

$$\text{Best mirror} = \arg\min_m T_{mirror_m}$$

### Cache Hit Rate

$$\text{Cache Hit} = \frac{\text{Packages in /var/cache/apt/archives/}}{\text{Total Packages to Install}}$$

$$\text{Download Needed} = \text{Total} - \text{Cached}$$

$$\text{Cache Size} = \sum_{\text{cached .debs}} \text{File Size}$$

| Cache Policy | Disk Usage | Re-download on Reinstall |
|:---|:---:|:---:|
| Keep all (default) | Growing | Never |
| `apt clean` | 0 | Always |
| `apt autoclean` | Current versions only | Only obsolete |

---

## 5. Disk Space Calculations

### Installation Size

$$\text{Installed Size} \approx 2-5 \times \text{.deb Download Size}$$

$$\text{Total Disk} = \text{Cache (downloads)} + \text{Installed Files} + \text{dpkg Database}$$

### dpkg Database Size

$$\text{/var/lib/dpkg/} \approx 50-200 \text{ MiB}$$

Contains: status file, available file, info/ directory with control files for each package.

$$\text{Per-Package DB Entry} \approx 2-10 \text{ KiB}$$

### Autoremove Savings

$$\text{Removable} = \text{Installed} - \text{Manually Installed} - \text{Dependencies of Manual}$$

---

## 6. Version Comparison Algorithm

### Debian Version Format

$$\text{Version} = [\text{epoch}:]\text{upstream\_version}[-\text{debian\_revision}]$$

### Comparison Algorithm

1. Compare epoch (integer, default 0)
2. Compare upstream version (character-by-character, digits as numbers)
3. Compare debian revision

### Version Ordering Examples

| Version A | Version B | Result | Reason |
|:---|:---|:---:|:---|
| 1.0 | 2.0 | A < B | Upstream version |
| 1:1.0 | 2.0 | A > B | Epoch 1 > epoch 0 |
| 1.0-1 | 1.0-2 | A < B | Debian revision |
| 1.0~beta1 | 1.0 | A < B | Tilde sorts before everything |
| 1.0+dfsg | 1.0 | A > B | Plus sorts after empty |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| SAT / NP-complete | Boolean satisfiability | Dependency resolution |
| $|\text{TC}(P)|$ | Graph transitive closure | Total dependencies |
| SHA256 chain | Cryptographic hash | Package verification |
| $\frac{\sum \text{Size}}{n \times \text{BW}}$ | Rate equation | Download time |
| $\text{epoch:upstream-revision}$ | Lexicographic | Version comparison |
| $\text{Packages} \times 500$ bytes | Linear | Index size |

---

*Every `apt install`, `apt update`, and `dpkg --configure` runs through these algorithms — a dependency solver that navigates a 60,000-node graph to find a consistent set of packages that satisfies all constraints.*

## Prerequisites

- Dependency graph concepts (directed acyclic graphs)
- Boolean satisfiability (SAT) problem basics
- Debian .deb package format and repository structure
- GPG signature verification concepts

## Complexity

- **Beginner:** Install/remove/upgrade packages, add repositories
- **Intermediate:** Pinning, version holds, dpkg low-level operations, offline installs
- **Advanced:** SAT-based dependency resolution internals, Packages.gz index parsing, delta compression, resolver conflict strategies
