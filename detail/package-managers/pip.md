# The Mathematics of pip — Python Package Manager Internals

> *pip resolves Python package dependencies from PyPI. The math covers dependency resolution (backtracking), wheel vs source distribution performance, virtual environment isolation, and the resolver's search space.*

---

## 1. Dependency Resolution — Backtracking Solver

### The Model

Since pip 20.3, pip uses a **backtracking resolver** that tries version combinations and backtracks on conflicts.

### Resolution Algorithm

$$\text{For each unresolved package } P:$$

$$\text{Try } v = \max(\text{compatible versions})$$

$$\text{If conflict: backtrack and try } v_{next}$$

### Search Space

$$\text{Worst Case} = \prod_{i=1}^{n} |V_i| \quad (\text{all version combinations})$$

Where $|V_i|$ = available versions of package $i$.

### Worked Example

*"5 packages, each with 20 compatible versions."*

$$\text{Search Space} = 20^5 = 3,200,000 \text{ combinations}$$

In practice, the resolver prunes aggressively using version constraints, typically exploring <100 candidates.

### Resolution Performance

| Scenario | Packages | Versions to Check | Time |
|:---|:---:|:---:|:---:|
| Simple (no conflicts) | 20 | 20 | <1s |
| Moderate conflicts | 50 | 200 | 2-10s |
| Heavy conflicts | 100 | 5,000 | 30-300s |
| Conflicting (unsolvable) | Any | All | Timeout |

### Version Specifiers

| Specifier | Meaning | Example |
|:---|:---|:---|
| `==1.2.3` | Exact match | Only 1.2.3 |
| `>=1.2,<2.0` | Range | 1.2.0 to 1.99.99 |
| `~=1.2.3` | Compatible release | >=1.2.3, ==1.2.* |
| `!=1.2.3` | Exclusion | Anything but 1.2.3 |
| `>=1.2.3` | Minimum | 1.2.3 or higher |

---

## 2. Wheel vs Source Distribution

### The Model

Wheels (.whl) are pre-compiled binary distributions. Source distributions (sdist) require compilation.

### Installation Time

$$T_{wheel} = T_{download} + T_{extract}$$

$$T_{sdist} = T_{download} + T_{extract} + T_{build} + T_{compile}$$

### Speedup from Wheels

$$\text{Speedup} = \frac{T_{sdist}}{T_{wheel}}$$

| Package | sdist Install | Wheel Install | Speedup |
|:---|:---:|:---:|:---:|
| requests | 2s | 1s | 2x |
| numpy | 120s | 3s | 40x |
| pandas | 300s | 5s | 60x |
| cryptography | 60s | 2s | 30x |
| scipy | 600s | 8s | 75x |

### Wheel Compatibility Tags

$$\text{Wheel Tag} = \{Python\}-\{ABI\}-\{Platform\}$$

$$\text{Compatible if:} \quad \text{tag} \in \text{sys.tags()}$$

| Tag | Meaning |
|:---|:---|
| `py3-none-any` | Pure Python 3, any platform |
| `cp312-cp312-manylinux_2_17_x86_64` | CPython 3.12, Linux x86_64 |
| `cp312-cp312-macosx_14_0_arm64` | CPython 3.12, macOS ARM |

### PyPI Wheel Coverage

$$\text{Packages with wheels} \approx 85\% \text{ of top 1000 packages}$$

---

## 3. Virtual Environment Math

### The Model

Virtual environments provide isolated Python installations with independent package sets.

### Space Usage

$$\text{Venv Size} = \text{Python Runtime (~50 MiB)} + \sum \text{Installed Packages}$$

### Symlink vs Copy

$$\text{Symlink venv} = 5-10 \text{ MiB (pointers to system Python)}$$

$$\text{Copy venv} = 50+ \text{ MiB (full Python copy)}$$

### Worked Example

| Project | Direct Deps | Transitive Deps | Venv Size |
|:---|:---:|:---:|:---:|
| Flask API | 5 | 15 | 30 MiB |
| Django project | 10 | 40 | 80 MiB |
| Data science (numpy, pandas, scipy) | 5 | 30 | 500 MiB |
| ML (torch, transformers) | 10 | 100 | 5-15 GiB |

### Multiple Venvs

$$\text{Total Disk} = n \times \text{Avg Venv Size}$$

| Projects | Avg Venv | Total Disk | With Shared Cache |
|:---:|:---:|:---:|:---:|
| 5 | 100 MiB | 500 MiB | 200 MiB |
| 10 | 200 MiB | 2 GiB | 500 MiB |
| 20 | 500 MiB | 10 GiB | 2 GiB |

---

## 4. PyPI Registry and Network

### Package Index

$$\text{PyPI Packages} \approx 500,000+$$

$$\text{Total Releases} \approx 5,000,000+$$

### Download Performance

$$T_{download} = \frac{\text{Package Size}}{\text{Bandwidth}} + T_{DNS} + T_{TLS}$$

### pip Cache

$$\text{Cache Location:} \quad \texttt{\~{}/.cache/pip/}$$

$$\text{Cache Hit} = \begin{cases} \text{Skip download} & \text{if exact version cached} \\ \text{Skip build} & \text{if wheel cached from previous sdist build} \end{cases}$$

### Cache Savings

$$\text{Install Time (cached)} = T_{extract} \text{ only}$$

$$\text{Speedup} = \frac{T_{download} + T_{build}}{T_{extract}} \approx 5-100\times$$

---

## 5. Requirements File Math

### Pinning Strategies

| Strategy | Reproducibility | Update Flexibility | Example |
|:---|:---:|:---:|:---|
| Unpinned | None | Full | `requests` |
| Minimum | Low | High | `requests>=2.28` |
| Compatible | Medium | Medium | `requests~=2.28.0` |
| Exact | High | None | `requests==2.28.1` |
| Hash-pinned | Highest | None | `requests==2.28.1 --hash=sha256:...` |

### Hash Verification

$$\text{Verified} = \text{SHA256}(\text{downloaded}) \in \text{Allowed Hashes}$$

$$T_{hash\_verify} = \frac{\text{File Size}}{\text{SHA256 Throughput}} \approx \frac{\text{File Size}}{1 \text{ GiB/s}}$$

---

## 6. Dependency Conflicts — Common Patterns

### Diamond Dependency Problem

```
Project -> A -> C>=1.0,<2.0
Project -> B -> C>=2.0,<3.0
```

$$\text{No solution exists:} \quad [1.0, 2.0) \cap [2.0, 3.0) = \emptyset$$

### Resolution Strategies

| Strategy | Result |
|:---|:---|
| pip (strict) | Error — cannot resolve |
| pip (force) | `--force-reinstall` — last version wins (broken) |
| Backtrack | Try alternative versions of A or B |

### Common Conflicts

| Package Pair | Typical Conflict | Solution |
|:---|:---|:---|
| boto3 + awscli | botocore version | Pin compatible versions |
| numpy + scipy | numpy version | Use compatible releases |
| protobuf versions | Google packages | Pin protobuf explicitly |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\prod |V_i|$ | Combinatorial | Resolution search space |
| $\frac{T_{sdist}}{T_{wheel}}$ | Ratio | Wheel speedup |
| $n \times \text{Avg Venv}$ | Linear | Total venv disk |
| SHA256 verification | Cryptographic | Package integrity |
| $[a, b) \cap [c, d)$ | Interval arithmetic | Conflict detection |
| Backtracking search | Tree search | Resolver algorithm |

---

*Every `pip install`, `pip freeze`, and `requirements.txt` runs through these algorithms — a dependency resolver that navigates Python's 500,000+ package ecosystem with backtracking search to find compatible version sets.*

## Prerequisites

- Python virtual environments (venv, site-packages isolation)
- SemVer and PEP 440 version specifiers
- Wheel vs source distribution formats

## Complexity

- **Beginner:** Install packages, create virtual environments, freeze requirements
- **Intermediate:** Editable installs, extras, constraints files, private indexes
- **Advanced:** Backtracking resolver internals, wheel binary compatibility (manylinux), build system hooks (PEP 517/518)
