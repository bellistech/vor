# The Mathematics of Nix — Functional Package Management Theory

> *Nix treats packages as pure functions of their inputs — the output is determined entirely by the input hash. This functional model enables atomic upgrades, rollbacks, and reproducible builds through content-addressable storage, closure computation, and a purely functional language.*

---

## 1. Package as Pure Function

### The Core Axiom

$$\text{package} = f(\text{inputs}) \rightarrow \text{/nix/store/}\langle\text{hash}\rangle\text{-}\langle\text{name}\rangle$$

Where:
- $f$ = build recipe (derivation)
- inputs = source code + dependencies + build tools + build script
- hash = cryptographic digest of ALL inputs

### The Hash Function

$$\text{hash} = \text{SHA-256}(\text{name} \| \text{system} \| \text{builder} \| \text{args} \| \text{env} \| \text{input\_hashes})$$

$$\text{store\_path} = \text{/nix/store/} \| \text{base32}(\text{hash}[0:160]) \| \text{-} \| \text{name}$$

### The Purity Guarantee

If any input changes — even a single build flag — the hash changes:

$$\text{inputs}_1 \neq \text{inputs}_2 \implies \text{hash}_1 \neq \text{hash}_2 \implies \text{path}_1 \neq \text{path}_2$$

Two builds with identical inputs produce identical outputs:

$$\text{inputs}_1 = \text{inputs}_2 \implies \text{output}_1 = \text{output}_2$$

This is **referential transparency** — the hallmark of pure functional programming.

### Worked Example

```
/nix/store/w9yy7gk3kiy1f3rvfwbh4rpc4al9j4pc-nginx-1.25.3
                |                                |
                SHA-256 of all inputs             name + version
```

Changing the OpenSSL dependency from 3.1 to 3.2 produces an entirely new path — both versions coexist.

---

## 2. Content-Addressable Store (Deduplication)

### The Problem

Multiple packages may share identical build outputs. Content addressing enables automatic deduplication.

### Store Size

$$S_{store} = \sum_{p \in \text{installed}} S_p - S_{dedup}$$

### Deduplication Rate

$$\text{Dedup ratio} = \frac{S_{logical} - S_{physical}}{S_{logical}}$$

### Hard Link Optimization

Within a single package, identical files are hard-linked:

$$S_{actual}(p) = S_{unique\_content}(p) + |\text{unique files}| \times S_{inode}$$

### Typical Store Sizes

| System Profile | Packages | Logical Size | Physical Size | Dedup |
|:---|:---:|:---:|:---:|:---:|
| Minimal server | 200 | 2 GB | 1.5 GB | 25% |
| Desktop | 1,500 | 15 GB | 10 GB | 33% |
| Dev workstation | 3,000 | 30 GB | 18 GB | 40% |
| NixOS full | 5,000+ | 50 GB | 28 GB | 44% |

---

## 3. Closure Computation (Transitive Dependencies)

### The Problem

A package's **closure** is the set of all transitive dependencies. This determines what must be present for the package to work.

### The Closure Function

$$\text{closure}(p) = \{p\} \cup \bigcup_{d \in \text{deps}(p)} \text{closure}(d)$$

This is a recursive definition — compute the fixed point by following all dependency edges.

### Closure Size

$$S_{closure}(p) = \sum_{q \in \text{closure}(p)} S_q$$

### Worked Examples

| Package | Direct Deps | Closure Size (packages) | Closure Size (disk) |
|:---|:---:|:---:|:---:|
| bash | 3 | 15 | 80 MB |
| nginx | 8 | 45 | 200 MB |
| python3 | 5 | 60 | 350 MB |
| ghc (Haskell) | 12 | 200+ | 2 GB+ |

### Why Closure Matters

To copy a package to another machine, you must copy its entire closure:

$$S_{transfer}(p) = \sum_{q \in \text{closure}(p) \setminus \text{already\_present}} S_q$$

### Closure Minimization

$$S_{minimal} = S_{closure}(\text{runtime\_deps only, exclude build deps})$$

A derivation with `buildInputs = [gcc]` won't include gcc in the runtime closure — only in the build closure.

---

## 4. Garbage Collection (Reachability)

### The Problem

The Nix store grows with every build. GC reclaims unreachable store paths.

### GC Roots

$$\text{Roots} = \text{profiles} \cup \text{gcroots} \cup \text{running\_builds}$$

### Reachability

$$\text{Reachable} = \bigcup_{r \in \text{Roots}} \text{closure}(r)$$

$$\text{Garbage} = \text{Store} \setminus \text{Reachable}$$

### GC Savings

$$S_{freed} = \sum_{p \in \text{Garbage}} S_p$$

### Worked Example

After upgrading Firefox from v120 to v121:

- v120 remains in store (reachable from previous profile generation)
- After `nix-collect-garbage -d` (delete old generations):

$$\text{Newly unreachable} = \text{closure}(v120) \setminus \text{closure}(v121)$$

Packages shared between v120 and v121 are NOT collected.

---

## 5. Generations (Atomic Rollback Model)

### The Problem

Nix profiles maintain generations — snapshots of the installed package set. Rollback is instant.

### Generation Model

$$G_k = \{p_1, p_2, \ldots, p_n\} \quad \text{(set of installed packages at generation } k\text{)}$$

$$G_{k+1} = (G_k \setminus \text{removed}) \cup \text{added}$$

### Rollback

$$\text{rollback}() = G_{current} \leftarrow G_{current - 1}$$

This is a symlink switch — $O(1)$ time:

$$T_{rollback} = T_{symlink} \approx 1\text{ ms}$$

### Generation Diff

$$\text{Added}(k \rightarrow k+1) = G_{k+1} \setminus G_k$$
$$\text{Removed}(k \rightarrow k+1) = G_k \setminus G_{k+1}$$
$$\text{Unchanged} = G_k \cap G_{k+1}$$

### Storage Cost of Generations

$$S_{generations} = S_{closure}(G_{latest}) + \sum_{k < \text{latest}} S_{\text{closure}(G_k) \setminus \text{closure}(G_{latest})}$$

Each old generation only costs the delta in unique packages.

---

## 6. Nix Language Evaluation (Lazy Evaluation)

### The Problem

The Nix expression language is purely functional and lazily evaluated. Understanding evaluation cost helps with large Nixpkgs evaluations.

### Evaluation Model

$$\text{eval}(expr) = \begin{cases}
\text{value} & \text{if forced} \\
\text{thunk} & \text{if not yet needed}
\end{cases}$$

### Nixpkgs Evaluation Cost

$$T_{eval} = O(|\text{evaluated attributes}|)$$

Full Nixpkgs: ~80,000 packages. Evaluating all:

$$T_{eval\_all} \approx 30\text{-}120\text{s}$$

Evaluating a single package (lazy):

$$T_{eval\_one} \approx 0.5\text{-}5\text{s}$$

### Fixed-Point Evaluation (Overlays)

Nixpkgs uses a fixed-point combinator for package overrides:

$$\text{pkgs} = \text{fix}(\text{self}: \text{import nixpkgs} \{\ \text{overlays} = [\ldots]; \})$$

$$\text{fix}(f) = \text{let } x = f(x) \text{ in } x$$

This allows packages to reference the final set (including overrides) — lazy evaluation prevents infinite recursion.

---

## 7. Binary Cache (Substitution)

### The Problem

Building from source is slow. Binary caches provide pre-built outputs keyed by the same store path hash.

### Cache Hit Decision

$$\text{action}(p) = \begin{cases}
\text{substitute (download)} & \text{if } p \in \text{cache} \\
\text{build from source} & \text{otherwise}
\end{cases}$$

### Download vs Build Time

$$\text{Speedup} = \frac{T_{build}}{T_{download}} = \frac{T_{build}}{S_p / BW}$$

| Package | Build Time | Size | Download (100Mbps) | Speedup |
|:---|:---:|:---:|:---:|:---:|
| bash | 30s | 5 MB | 0.4s | 75x |
| python3 | 300s | 50 MB | 4s | 75x |
| linux kernel | 3600s | 150 MB | 12s | 300x |
| chromium | 7200s | 500 MB | 40s | 180x |

### Cache Hit Rate

$$\text{Hit rate} = \frac{|\text{closure}(p) \cap \text{cache}|}{|\text{closure}(p)|}$$

Official cache (cache.nixos.org) hit rate for standard configurations: ~99%.

Custom overlays or unfree packages: hit rate drops to 80-95%.

---

## 8. Summary of Functions by Type

| Formula | Math Type | Domain |
|:---|:---|:---|
| $f(\text{inputs}) \rightarrow \text{hash-name}$ | Pure function | Package model |
| $\text{SHA-256}(\text{all inputs})$ | Cryptographic hash | Content addressing |
| $\{p\} \cup \bigcup \text{closure}(d)$ | Recursive set union | Closure computation |
| $\text{Store} \setminus \text{Reachable}$ | Set difference | Garbage collection |
| $\text{fix}(f) = f(\text{fix}(f))$ | Fixed-point combinator | Package overrides |
| $G_k \setminus G_{k+1}$ | Set difference | Generation diff |

---

*Nix applies the lambda calculus to package management — packages are pure functions, the store is content-addressed, and upgrades are atomic generation switches. This isn't just theory — it's why NixOS can roll back an entire operating system in milliseconds.*

## Prerequisites

- Functional programming concepts (pure functions, immutability)
- Linux package management basics
- Content-addressable storage concepts (hashing)
- Basic understanding of build systems

## Complexity

- Beginner: nix-env package install/remove, nix-shell ad-hoc environments
- Intermediate: flakes, devShells, NixOS configuration, Home Manager
- Advanced: derivation authoring, closure computation, overlays, cross-compilation, store optimization
