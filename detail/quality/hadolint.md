# The Mathematics of Hadolint -- Image Layer Optimization and Security Surface

> *Every Dockerfile instruction creates a layer, every unpinned dependency is a nondeterministic variable, and every unnecessary package expands the attack surface. Hadolint enforces the mathematics of minimal, reproducible, and secure container images.*

---

## 1. Layer Size Optimization (Cache-Aware Minimization)

### The Problem

Docker images are stacks of layers. Each `RUN`, `COPY`, and `ADD` instruction creates a new layer. Files deleted in a later layer still exist in the earlier layer, consuming space. Hadolint's DL3009 (clean up apt lists) and layer-combining rules address this.

### The Formula

The image size is the sum of all layers, not the final filesystem size:

$$S_{image} = \sum_{i=1}^{n} S_{layer_i}$$

If layer $j$ adds files of size $A$ and layer $k > j$ deletes them:

$$S_{image} = \ldots + A + \ldots + 0 + \ldots = \ldots + A + \ldots$$

The deleted files still contribute $A$ to the image. Combining into a single layer:

$$S_{combined} = A - A = 0$$

### Worked Examples

A Dockerfile with separate install and cleanup:

```
Layer 1: apt-get update           (+30 MB lists)
Layer 2: apt-get install curl     (+15 MB)
Layer 3: rm -rf /var/lib/apt/lists/*  (+0 MB, but lists still in Layer 1)
```

$$S_{image} = 30 + 15 + 0 = 45 \text{ MB}$$

Combined into one `RUN`:

```
Layer 1: apt-get update && apt-get install curl && rm -rf /var/lib/apt/lists/*
```

$$S_{image} = 15 \text{ MB}$$

$$\text{Savings} = 1 - \frac{15}{45} = 66.7\%$$

For a typical application image with 5 package installations:

$$S_{uncombined} = 5 \times (L_{lists} + L_{pkg}) = 5 \times (30 + 15) = 225 \text{ MB}$$
$$S_{combined} = 5 \times L_{pkg} = 5 \times 15 = 75 \text{ MB}$$

---

## 2. Dependency Pinning and Reproducibility (Version Entropy)

### The Problem

Unpinned dependencies (`apt-get install curl`) resolve to whatever version is current at build time. DL3008 flags this because builds become nondeterministic. The version space grows with the number of unpinned packages.

### The Formula

For $n$ unpinned packages, each with $v_i$ available versions, the number of possible build configurations:

$$C = \prod_{i=1}^{n} v_i$$

The probability that two builds on different days produce identical images:

$$P(\text{identical}) = \prod_{i=1}^{n} P(\text{same version}_i) = \prod_{i=1}^{n} \frac{1}{1 + r_i \cdot \Delta t}$$

where $r_i$ is the release rate (versions per day) and $\Delta t$ is the time between builds.

### Worked Examples

A Dockerfile installs 10 packages, each averaging 2 new versions per month ($r = 0.067$/day). Builds are 30 days apart:

$$P(\text{same}_i) = \frac{1}{1 + 0.067 \times 30} = \frac{1}{3.0} = 0.333$$

$$P(\text{identical}) = 0.333^{10} = 0.0000169 \approx 0.002\%$$

There is a $99.998\%$ chance the builds differ. With pinned versions:

$$P(\text{identical}) = 1.0 \quad (100\%)$$

---

## 3. Attack Surface Quantification (CVE Exposure)

### The Problem

Every installed package adds potential vulnerabilities. DL3015 (use `--no-install-recommends`) and package pinning reduce the attack surface. The expected number of CVEs is proportional to the number of installed packages.

### The Formula

Expected CVEs in an image with $n$ packages, each with an average CVE rate of $\lambda$ CVEs per package per year:

$$E[\text{CVEs}] = n \times \lambda$$

With `--no-install-recommends`, the package count drops from $n_{full}$ to $n_{minimal}$:

$$\text{CVE Reduction} = \frac{n_{full} - n_{minimal}}{n_{full}} \times 100\%$$

The time-to-exploit window for an unpinned package:

$$T_{exposed} = T_{CVE\_publish} - T_{image\_rebuild}$$

If $T_{exposed} > 0$, the image is vulnerable. Pinned versions make this deterministic.

### Worked Examples

Installing python3 on Debian:
- With recommends: 45 packages installed
- Without recommends: 12 packages installed

$$\text{CVE Reduction} = \frac{45 - 12}{45} = 73.3\%$$

At $\lambda = 0.5$ CVEs/package/year:

$$E[\text{CVEs}]_{full} = 45 \times 0.5 = 22.5 \text{ CVEs/year}$$
$$E[\text{CVEs}]_{minimal} = 12 \times 0.5 = 6 \text{ CVEs/year}$$

Absolute reduction: $16.5$ fewer expected CVEs per year.

---

## 4. Signal Propagation and Process Models (DL3025)

### The Problem

DL3025 enforces exec form (`CMD ["app"]`) over shell form (`CMD app`). Shell form wraps the process in `/bin/sh -c`, which intercepts signals (SIGTERM, SIGINT). This affects container shutdown behavior and graceful termination.

### The Formula

Signal delivery probability to the application process:

$$P(\text{signal reaches app}) = \begin{cases} 1 & \text{exec form (PID 1 = app)} \\ P(\text{sh forwards}) & \text{shell form (PID 1 = sh)} \end{cases}$$

In practice, many `sh` implementations do not forward SIGTERM:

$$P(\text{sh forwards SIGTERM}) \approx 0$$

The result: Kubernetes sends SIGTERM, waits `terminationGracePeriodSeconds` (default 30s), then sends SIGKILL. The graceful shutdown window is wasted:

$$T_{graceful} = \begin{cases} 30 \text{s (usable)} & \text{exec form} \\ 0 \text{s (wasted, SIGKILL after timeout)} & \text{shell form} \end{cases}$$

### Worked Examples

A web server needs 5 seconds to drain connections:

**Exec form:** SIGTERM received at $t=0$, drain completes at $t=5$, exit at $t=5$. Zero dropped requests.

**Shell form:** SIGTERM received by sh at $t=0$, not forwarded. App continues serving until SIGKILL at $t=30$. All in-flight requests at $t=30$ are dropped.

$$\text{Requests dropped}_{shell} = \text{RPS} \times T_{inflight} \approx 100 \times 0.5 = 50$$

---

## 5. Trusted Registry Enforcement (Supply Chain)

### The Problem

Hadolint's `trustedRegistries` config restricts which registries images can be pulled from. This is a supply chain security control. The risk of pulling from an untrusted registry is proportional to the number of registries allowed.

### The Formula

The probability of a supply chain attack via image substitution:

$$P(\text{attack}) = 1 - \prod_{i=1}^{R} (1 - p_i)$$

where $R$ is the number of allowed registries and $p_i$ is the compromise probability of registry $i$.

With trusted-only registries ($R_t \subset R_{all}$):

$$P(\text{attack})_{trusted} \leq P(\text{attack})_{all}$$

The risk reduction:

$$\Delta P = P(\text{attack})_{all} - P(\text{attack})_{trusted}$$

### Worked Examples

Unrestricted (5 registries, each with $p = 0.001$ annual compromise probability):

$$P(\text{attack})_{all} = 1 - (1 - 0.001)^5 = 1 - 0.995 = 0.005$$

Restricted to 2 trusted registries:

$$P(\text{attack})_{trusted} = 1 - (1 - 0.001)^2 = 1 - 0.998 = 0.002$$

$$\Delta P = 0.005 - 0.002 = 0.003 \quad (60\% \text{ reduction})$$

---

## 6. Build Cache Efficiency (Instruction Ordering)

### The Problem

Docker caches layers and invalidates the cache from the first changed instruction onward. Instruction ordering determines cache hit rates. Hadolint rules indirectly improve cache efficiency by encouraging patterns that maximize cache reuse.

### The Formula

For a Dockerfile with $n$ instructions, if instruction $k$ changes, layers $k$ through $n$ are rebuilt:

$$\text{Cache miss cost} = \sum_{i=k}^{n} T_{build_i}$$

The expected cache miss cost given that each instruction changes with probability $p_i$:

$$E[\text{rebuild}] = \sum_{k=1}^{n} p_k \times \sum_{i=k}^{n} T_{build_i}$$

Optimal ordering: place rarely-changing instructions first (low $p$) and frequently-changing ones last (high $p$).

### Worked Examples

A 5-instruction Dockerfile:

| Instruction | $T_{build}$ | $p$ (change freq) |
|:---|:---:|:---:|
| FROM ubuntu:22.04 | 0s | 0.01 |
| COPY requirements.txt | 1s | 0.10 |
| RUN pip install | 30s | 0.10 |
| COPY . . | 2s | 0.90 |
| CMD ["python", "app.py"] | 0s | 0.05 |

**Optimal order (as shown):**
$$E[\text{rebuild}] = 0.01 \times 33 + 0.10 \times 33 + 0.10 \times 32 + 0.90 \times 2 + 0.05 \times 0 = 0.33 + 3.3 + 3.2 + 1.8 + 0 = 8.63 \text{ s}$$

**Worst order (COPY . . before pip install):**
$$E[\text{rebuild}] = 0.01 \times 33 + 0.90 \times 33 + 0.10 \times 32 + 0.10 \times 2 + 0.05 \times 0 = 0.33 + 29.7 + 3.2 + 0.2 + 0 = 33.43 \text{ s}$$

$$\text{Speedup} = \frac{33.43}{8.63} = 3.9\times$$

---

## Prerequisites

- Docker layer model (union filesystem, copy-on-write)
- Unix process signals (SIGTERM, SIGINT, SIGKILL, PID 1)
- Software supply chain security (image provenance, registry trust)
- Combinatorics (Cartesian product of version spaces)
- Expected value and probability
- Cache invalidation theory (LRU, ordered invalidation)
