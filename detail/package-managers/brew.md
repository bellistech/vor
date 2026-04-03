# The Mathematics of Homebrew — macOS Package Manager Internals

> *Homebrew manages packages on macOS via formulae (source recipes) and casks (binary apps). The math covers dependency resolution, bottle (binary) distribution, and the tap system architecture.*

---

## 1. Formula Dependency Graph

### The Model

Each formula declares dependencies. Homebrew resolves these into a directed acyclic graph (DAG) and installs in topological order.

### Dependency Types

$$\text{depends\_on:} \quad A \Rightarrow B \quad (\text{runtime})$$

$$\text{uses\_from\_macos:} \quad A \rightarrow B_{system} \quad (\text{use system version if available})$$

$$\text{build dependency:} \quad A \Rightarrow_{build} B \quad (\text{only needed for compilation})$$

### Topological Sort

$$\text{Install Order} = \text{TopologicalSort}(\text{DAG})$$

$$T_{topo} = O(V + E) \quad \text{where } V = \text{packages}, E = \text{dependency edges}$$

### Homebrew-core Statistics

| Metric | Value |
|:---|:---:|
| Formulae | ~6,500 |
| Casks | ~5,000 |
| Avg dependencies per formula | ~3-5 |
| Max dependency chain | ~15-20 deep |

---

## 2. Bottles — Binary Package Distribution

### The Model

Bottles are pre-compiled binary packages, avoiding source compilation.

### Build vs Bottle Time

$$\text{Speedup} = \frac{T_{source\_compile}}{T_{bottle\_download}}$$

| Formula | Source Compile | Bottle Download | Speedup |
|:---|:---:|:---:|:---:|
| openssl | 3 min | 5 sec | 36x |
| python | 10 min | 8 sec | 75x |
| gcc | 60 min | 15 sec | 240x |
| llvm | 120 min | 20 sec | 360x |

### Bottle Availability

$$\text{Bottle available if:} \quad (\text{macOS version}, \text{CPU arch}) \in \text{Built Bottles}$$

| Platform | Bottle Tag |
|:---|:---|
| macOS 14 (Sonoma) ARM | `arm64_sonoma` |
| macOS 14 (Sonoma) Intel | `sonoma` |
| macOS 13 (Ventura) ARM | `arm64_ventura` |
| Linux x86_64 | `x86_64_linux` |

### Bottle Size vs Installed Size

$$\text{Bottle} \approx \frac{\text{Installed Size}}{2-4} \quad (\text{gzip compressed tar})$$

| Formula | Installed Size | Bottle Size | Ratio |
|:---|:---:|:---:|:---:|
| node | 90 MiB | 25 MiB | 3.6x |
| python@3.12 | 120 MiB | 35 MiB | 3.4x |
| ffmpeg | 200 MiB | 60 MiB | 3.3x |
| gcc | 800 MiB | 200 MiB | 4.0x |

---

## 3. Cellar and Linking — Installation Math

### The Model

Packages install to versioned directories in the Cellar, then symlink into the prefix.

### Path Structure

```
/opt/homebrew/Cellar/openssl@3/3.2.1/
    bin/openssl    -> /opt/homebrew/bin/openssl (symlink)
    lib/libssl.a   -> /opt/homebrew/lib/libssl.a (symlink)
    include/       -> /opt/homebrew/include/openssl (symlink)
```

### Symlink Count

$$\text{Symlinks per Formula} = \text{bin files} + \text{lib files} + \text{include dirs} + \text{share files}$$

| Formula | Symlinks Created | Link Time |
|:---|:---:|:---:|
| Typical CLI tool | 1-5 | <0.1s |
| Library (openssl) | 20-50 | ~0.2s |
| Large package (gcc) | 100+ | ~0.5s |

### Disk Usage

$$\text{Total Brew Disk} = \sum_{\text{formulae}} \text{Installed Size} + \text{Cache (bottles)} + \text{Repo Clone}$$

| Component | Typical Size |
|:---|:---:|
| homebrew-core repo | ~300 MiB |
| Installed formulae (avg user) | 2-10 GiB |
| Bottle cache | 0-5 GiB |
| homebrew-cask repo | ~200 MiB |

---

## 4. Update and Upgrade Costs

### Update Frequency

$$\text{Formulae Updated/Week} \approx 200-500 \quad (\text{homebrew-core})$$

$$\text{Git Pull Size} = \text{Commits Since Last Update} \times \text{Avg Commit Size}$$

### Upgrade Calculation

$$\text{Outdated} = \{f : f.\text{installed\_version} < f.\text{latest\_version}\}$$

$$T_{upgrade} = \sum_{f \in \text{Outdated}} (T_{download_f} + T_{install_f} + T_{link_f})$$

### Cascade Upgrades

When a dependency upgrades, all dependents must be rebuilt (if compiled from source) or re-linked:

$$\text{Affected Packages} = \text{ReverseDependencies}(P)$$

$$|\text{RevDeps}(\text{openssl})| \approx 200+ \text{ packages}$$

---

## 5. Cleanup and Cache Math

### Cache Growth

$$\text{Cache Size} = \sum_{\text{installed versions}} \text{Bottle Size}$$

$$\text{Reclaimable} = \sum_{\text{old versions}} \text{Bottle Size}$$

### Cleanup Formula

`brew cleanup` removes:
- Old versions of installed formulae
- Downloads older than 120 days

$$\text{Freed Space} = \text{Old Version Bottles} + \text{Stale Downloads}$$

| Months Since Cleanup | Typical Cache | After Cleanup |
|:---:|:---:|:---:|
| 1 | 500 MiB | 200 MiB |
| 3 | 2 GiB | 400 MiB |
| 6 | 5 GiB | 500 MiB |
| 12 | 10+ GiB | 600 MiB |

---

## 6. Tap System — Repository Math

### The Model

Taps are Git repositories containing formulae/casks.

### Tap Size

$$\text{Tap Size} = \text{Formulae Count} \times \text{Avg Formula Size (3 KiB)} + \text{Git History}$$

| Tap | Formulae | Repo Size |
|:---|:---:|:---:|
| homebrew/core | ~6,500 | ~300 MiB |
| homebrew/cask | ~5,000 | ~200 MiB |
| Custom tap | 10-100 | 1-10 MiB |

### Formula Resolution Order

$$\text{Search Order:} \quad \text{homebrew/core} \rightarrow \text{taps (alphabetical)} \rightarrow \text{casks}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| Topological sort O(V+E) | Graph algorithm | Install order |
| $\frac{T_{compile}}{T_{download}}$ | Ratio | Bottle speedup |
| $\sum \text{Installed Sizes}$ | Summation | Disk usage |
| $\text{ReverseDeps}(P)$ | Graph traversal | Cascade upgrades |
| $\text{Old Versions} \times \text{Size}$ | Product | Cache cleanup |
| $\text{Formulae} \times 3\text{K}$ | Linear | Tap size estimate |

---

*Every `brew install`, `brew upgrade`, and `brew cleanup` navigates these dependency graphs and binary distribution channels — a package manager that made source-based package management practical on macOS by adding binary bottles on top.*
