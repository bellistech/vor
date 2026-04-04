# The Mathematics of Cargo — Rust Package Manager Internals

> *Cargo handles Rust dependency resolution, compilation, and registry interaction. The math covers SemVer resolution, the dependency solver (SAT-based), compilation unit parallelism, and incremental build performance.*

---

## 1. Semantic Versioning — Compatibility Math

### The Model

Cargo uses SemVer (Semantic Versioning) with default compatibility rules.

### Version Format

$$\text{Version} = \text{MAJOR}.\text{MINOR}.\text{PATCH}$$

### Cargo's Default Compatibility (Caret ^)

$$\hat{} X.Y.Z \quad \text{matches} \quad [X.Y.Z, (X+1).0.0)$$

$$\hat{} 0.Y.Z \quad \text{matches} \quad [0.Y.Z, 0.(Y+1).0)$$

$$\hat{} 0.0.Z \quad \text{matches} \quad [0.0.Z, 0.0.(Z+1))$$

### Worked Examples

| Requirement | Minimum | Maximum (exclusive) | Matches |
|:---|:---:|:---:|:---|
| `^1.2.3` | 1.2.3 | 2.0.0 | 1.2.3, 1.3.0, 1.99.99 |
| `^0.2.3` | 0.2.3 | 0.3.0 | 0.2.3, 0.2.99 |
| `^0.0.3` | 0.0.3 | 0.0.4 | Only 0.0.3 |
| `~1.2.3` | 1.2.3 | 1.3.0 | 1.2.3, 1.2.99 |
| `>=1.2, <1.5` | 1.2.0 | 1.5.0 | Range |
| `=1.2.3` | 1.2.3 | 1.2.3 | Exact only |
| `*` | 0.0.0 | infinity | Any version |

---

## 2. Dependency Resolution — The Solver

### The Model

Cargo uses a **version-aware SAT solver** to find a compatible set of crate versions.

### Resolution Problem

Given dependency constraints $C_1, C_2, \ldots, C_n$:

$$\text{Find } \{v_1, v_2, \ldots, v_n\} \text{ such that } \forall i: v_i \in C_i \text{ and all mutual constraints satisfied}$$

### Complexity

$$\text{Worst case: NP-complete (general SAT)}$$

$$\text{Practical: } O(P \times V) \text{ where } P = \text{packages}, V = \text{avg versions per package}$$

### crates.io Statistics

| Metric | Value |
|:---|:---:|
| Total crates | ~140,000 |
| Total versions | ~1,000,000 |
| Avg versions per crate | ~7 |
| Max dependency depth | ~20-30 |

### Feature Resolution

Features multiply the resolution space:

$$\text{Feature Combinations} = 2^{|\text{features}|}$$

But Cargo uses additive feature unification — features are unioned, not independently solved:

$$\text{Active Features}(P) = \bigcup_{\text{all dependents}} \text{Requested Features}$$

---

## 3. Compilation Parallelism — Build Units

### The Model

Cargo compiles crates in parallel based on the dependency DAG. Independent crates compile simultaneously.

### Parallelism Formula

$$\text{Parallel Units} = \text{Width of DAG at Current Level}$$

$$T_{build} = \text{Critical Path Length} \times T_{avg\_crate}$$

### Critical Path

$$\text{Critical Path} = \text{Longest dependency chain (by compile time)}$$

### Worked Example

*"Project depends on A, B, C. A depends on D. B depends on D, E. C is independent."*

```
Level 0: D, E, C  (3 parallel)
Level 1: A, B     (2 parallel)
Level 2: Project  (1)
```

$$\text{Total levels} = 3$$

$$T_{serial} = T_D + T_E + T_C + T_A + T_B + T_{project}$$

$$T_{parallel} = \max(T_D, T_E, T_C) + \max(T_A, T_B) + T_{project}$$

### Jobs Flag

$$\text{Effective Parallelism} = \min(\text{-j flag}, \text{CPU cores}, \text{DAG width})$$

| CPU Cores | DAG Width | Effective Parallelism | Speedup vs Serial |
|:---:|:---:|:---:|:---:|
| 4 | 2 | 2 | ~2x |
| 8 | 10 | 8 | ~6-7x |
| 16 | 50 | 16 | ~10-12x |
| 32 | 100 | 32 | ~15-20x |

---

## 4. Incremental Compilation — Cache Math

### The Model

Cargo tracks file changes and only recompiles affected crates.

### Rebuild Triggers

$$\text{Rebuild}(C) \iff \text{Source Changed}(C) \lor \text{Dep Changed}(C) \lor \text{Flags Changed}$$

### Incremental Build Time

$$T_{incremental} = \sum_{\text{changed crates}} T_{compile_i}$$

$$\text{Savings} = 1 - \frac{T_{incremental}}{T_{full}}$$

### Worked Example

*"100-crate project, 1 leaf crate changed."*

$$T_{full} = 100 \times 2\text{s (avg)} = 200\text{s}$$

$$T_{incremental} = 1 \times 2\text{s} + 1 \times 3\text{s (final binary link)} = 5\text{s}$$

$$\text{Savings} = 1 - \frac{5}{200} = 97.5\%$$

### Target Directory Size

$$\text{target/ Size} = \sum_{\text{profiles}} \sum_{\text{crates}} (\text{rlib} + \text{deps} + \text{incremental cache})$$

| Project Size | Debug target/ | Release target/ |
|:---|:---:|:---:|
| Small (10 deps) | 500 MiB | 200 MiB |
| Medium (50 deps) | 2 GiB | 800 MiB |
| Large (200 deps) | 10 GiB | 3 GiB |
| Very large (500 deps) | 30 GiB | 10 GiB |

---

## 5. Registry and Download

### crates.io Index

$$\text{Index Size} \approx 200 \text{ MiB (git clone)} \quad | \quad \text{Sparse index: on-demand}$$

### Sparse Registry (Default since Rust 1.70)

$$\text{Download per crate} = \text{Single HTTP request for crate metadata}$$

$$T_{resolve} = \text{Dependency Count} \times T_{http\_request}$$

vs Git index:

$$T_{resolve} = T_{git\_clone} + O(1) \text{ local lookups}$$

| Method | First Use | Subsequent | Network |
|:---|:---:|:---:|:---:|
| Git index | 30-60s clone | <1s | 200 MiB |
| Sparse index | 2-5s (parallel HTTP) | <1s (cached) | ~100 KiB |

### Crate Download Size

$$\text{Crate .crate} = \text{Source code compressed (gzip tar)}$$

$$\text{Avg crate size} \approx 50-200 \text{ KiB}$$

---

## 6. Build Profiles — Optimization Trade-offs

### Debug vs Release

| Aspect | Debug | Release |
|:---|:---:|:---:|
| Optimization level | 0 | 3 |
| Debug info | Full | None |
| Compile time | 1x | 2-5x |
| Binary size | 5-20x larger | 1x |
| Runtime speed | 1x | 5-100x |

### LTO (Link-Time Optimization)

$$T_{lto} = T_{compile} + T_{lto\_pass}$$

$$T_{lto\_pass} \approx 0.5-2 \times T_{compile}$$

| LTO Mode | Compile Overhead | Binary Improvement | Use Case |
|:---|:---:|:---:|:---|
| None | 0% | Baseline | Development |
| Thin | +20-50% | -10-20% size, +5-10% speed | Release |
| Fat | +100-200% | -20-30% size, +10-20% speed | Final release |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $[X.Y.Z, (X+1).0.0)$ | Range/interval | SemVer compatibility |
| NP-complete (SAT) | Satisfiability | Dependency resolution |
| $\text{DAG Critical Path}$ | Graph algorithm | Build parallelism |
| $1 - \frac{T_{incr}}{T_{full}}$ | Ratio | Incremental savings |
| $2^{|\text{features}|}$ | Exponential | Feature combinations |
| $\min(\text{cores}, \text{DAG width})$ | Min function | Effective parallelism |

---

*Every `cargo build`, `cargo update`, and `Cargo.lock` reflects these algorithms — a build system and package manager that solves version constraints, parallelizes compilation across the dependency DAG, and caches incremental results.*

## Prerequisites

- Rust compilation model (crates, editions, codegen units)
- SemVer dependency resolution
- DAG-based build parallelism (compilation units and linking)

## Complexity

- **Beginner:** New projects, build/run/test, add dependencies
- **Intermediate:** Features, workspaces, profiles (dev/release), cross-compilation, clippy/fmt
- **Advanced:** Dependency solver internals, incremental compilation, LTO optimization, build timing analysis
