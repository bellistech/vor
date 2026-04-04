# The Mathematics of DNF — Fedora/RHEL Package Manager Internals

> *DNF (Dandified YUM) uses libsolv for SAT-based dependency resolution and librepo for repository management. The math covers the libsolv solver, delta RPM savings, module streams, and transaction costs.*

---

## 1. libsolv — SAT-Based Dependency Resolution

### The Model

DNF uses libsolv, which translates package dependencies into a Boolean satisfiability problem and solves it with a DPLL-based SAT solver.

### SAT Translation

$$\text{Package } A \text{ depends on } B \Rightarrow (A \Rightarrow B) \equiv (\neg A \lor B)$$

$$\text{Package } A \text{ conflicts with } B \Rightarrow (A \Rightarrow \neg B) \equiv (\neg A \lor \neg B)$$

$$\text{Package } A \text{ obsoletes } B \Rightarrow (A \Rightarrow \neg B) \land (B \Rightarrow A_{new})$$

### Solver Complexity

$$\text{Worst case: NP-complete}$$

$$\text{Practical (libsolv): } O(P \times R) \text{ where } P = \text{packages}, R = \text{rules generated}$$

### Repository Statistics

| Distribution | Packages (x86_64) | Provides | Requires |
|:---|:---:|:---:|:---:|
| Fedora | ~70,000 | ~500,000 | ~300,000 |
| RHEL/CentOS | ~30,000 | ~200,000 | ~150,000 |
| EPEL | ~15,000 | ~100,000 | ~60,000 |

### Resolution Time

$$T_{resolve} = T_{load\_repos} + T_{generate\_rules} + T_{solve}$$

| Component | Time | Notes |
|:---|:---:|:---|
| Load repo metadata | 0.5-2s | From SQLite cache |
| Generate SAT rules | 0.1-0.5s | Per transaction |
| Solve | 0.01-1s | Depends on conflict count |
| Total | 0.6-3.5s | Typical install |

---

## 2. RPM Package Structure

### RPM Header Math

$$\text{RPM File} = \text{Lead (96 bytes)} + \text{Signature} + \text{Header} + \text{Payload (cpio.gz)}$$

### Payload Compression

$$\text{Installed Size} \approx 2-5 \times \text{RPM Size}$$

| Compression | RPM Size vs Installed | Speed |
|:---|:---:|:---:|
| gzip (default) | 30-50% | Fast |
| xz | 20-35% | Slow compress, fast decompress |
| zstd | 25-40% | Fast compress and decompress |
| lzma | 20-35% | Slowest |

### RPM Verification

$$\text{Verification} = \text{GPG Signature} + \text{SHA256 Header} + \text{MD5 Payload}$$

$$T_{verify} = T_{gpg} + T_{sha256}(\text{header}) + T_{md5}(\text{payload})$$

---

## 3. Delta RPMs (drpm) — Bandwidth Savings

### The Model

Delta RPMs contain only the binary diff between two versions.

### Delta Size Formula

$$\text{Delta Size} \approx \text{Changed Bytes} + \text{Delta Header}$$

$$\text{Savings} = 1 - \frac{\text{Delta Size}}{\text{Full RPM Size}}$$

### Worked Examples

| Package | Full RPM | Delta RPM | Savings | Apply Time |
|:---|:---:|:---:|:---:|:---:|
| kernel (minor update) | 60 MiB | 15 MiB | 75% | 30s |
| glibc (minor) | 5 MiB | 1 MiB | 80% | 5s |
| firefox (major) | 100 MiB | 80 MiB | 20% | 60s |
| systemd (minor) | 10 MiB | 3 MiB | 70% | 10s |

### When Deltas Are Not Worth It

$$\text{Delta NOT useful if:} \quad \frac{\text{Delta Size}}{\text{Full Size}} > 0.5 \quad \text{AND} \quad T_{apply} > T_{download\_full}$$

$$T_{apply} = \frac{\text{Installed Size}}{\text{CPU Reconstruct Speed}}$$

---

## 4. Module Streams — Parallel Versions

### The Model

Modularity allows multiple versions of the same software to exist in a repository.

### Stream Selection

$$\text{Active Stream} \in \{\text{stream}_1, \text{stream}_2, \ldots\} \quad (\text{mutually exclusive})$$

### Module Resolution

$$\text{Module Dependencies} = \text{Package Dependencies} + \text{Module Stream Dependencies}$$

$$\text{Conflicts:} \quad \text{Module } A:\text{stream1} \text{ conflicts with } A:\text{stream2}$$

### Worked Example

```
Module: postgresql
  Stream 12: postgresql-12.x, postgresql-server-12.x
  Stream 15: postgresql-15.x, postgresql-server-15.x
  Stream 16: postgresql-16.x, postgresql-server-16.x
```

$$\text{Installable} = \text{Exactly one stream active at a time}$$

---

## 5. Repository Metadata — Download and Caching

### Metadata Structure

| File | Contents | Size (Fedora) |
|:---|:---|:---:|
| repomd.xml | Index of metadata files | 3 KiB |
| primary.xml.gz | Package names, deps, sizes | 15-25 MiB |
| filelists.xml.gz | All file paths per package | 20-30 MiB |
| other.xml.gz | Changelogs | 10-20 MiB |
| comps.xml | Package groups | 500 KiB |
| modules.yaml | Module streams | 200 KiB |

### Total Metadata Download

$$\text{First sync} \approx 50-80 \text{ MiB per repo}$$

### Metadata Expiry

$$\text{Cache Valid} = \text{metadata\_expire (default: 48 hours for Fedora, never for RHEL)}$$

$$\text{Forced Refresh:} \quad \texttt{dnf clean metadata}$$

### SQLite Cache Size

$$\text{Cache Size} \approx 2-3 \times \text{Compressed Metadata}$$

| Repos | Compressed | SQLite Cache |
|:---:|:---:|:---:|
| 1 (Fedora updates) | 25 MiB | 60 MiB |
| 3 (Fedora full) | 60 MiB | 150 MiB |
| 5 (RHEL + EPEL) | 40 MiB | 100 MiB |

---

## 6. Transaction Performance

### Transaction Phases

$$T_{transaction} = T_{resolve} + T_{download} + T_{verify} + T_{install} + T_{scriptlets}$$

### RPM scriptlet Execution

Each package can run scripts at 4 phases:

$$\text{Scriptlets} = \text{pre-install} + \text{post-install} + \text{pre-uninstall} + \text{post-uninstall}$$

$$T_{scriptlets} = \sum_{i=1}^{n} T_{script_i}$$

### Worked Example

*"Installing 50 packages, total 500 MiB download, 100 Mbps connection."*

$$T_{download} = \frac{500 \times 8}{100} = 40\text{s}$$

$$T_{verify} = 50 \times 0.1\text{s} = 5\text{s}$$

$$T_{install} = 50 \times 0.5\text{s} = 25\text{s}$$

$$T_{scriptlets} = 50 \times 0.2\text{s} = 10\text{s}$$

$$T_{total} = 3 + 40 + 5 + 25 + 10 = 83\text{s}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $(\neg A \lor B)$ | Boolean logic | SAT dependency rules |
| $1 - \frac{\text{Delta}}{\text{Full}}$ | Ratio | Delta RPM savings |
| $\text{Stream}_1 \oplus \text{Stream}_2$ | Exclusive OR | Module selection |
| $\frac{\text{Size} \times 8}{\text{BW}}$ | Rate equation | Download time |
| $\sum T_{phase}$ | Addition | Transaction time |
| $2-3 \times \text{compressed}$ | Multiplication | Cache size |

---

*Every `dnf install`, `dnf module enable`, and `dnf history` runs through libsolv's SAT solver — the same mathematical framework used in formal verification and AI planning, applied to finding a consistent set of RPM packages.*

## Prerequisites

- RPM package format and repository metadata (repodata)
- Dependency graph and SAT solving concepts
- Module stream versioning (RHEL/CentOS modularity)

## Complexity

- **Beginner:** Install/remove/upgrade packages, repository management
- **Intermediate:** Module streams, history undo/rollback, group management, delta RPMs
- **Advanced:** libsolv SAT solver internals, repository metadata compression, transaction cost analysis
