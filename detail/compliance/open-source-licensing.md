# The Mathematics of Open Source Licensing — Compatibility Graphs, Dependency Risk, and Compliance Optimization

> *Open source licensing compliance is a constraint satisfaction problem over a directed acyclic graph of dependencies: license compatibility forms a partial order, transitive copyleft obligations propagate through the dependency tree, the probability of a licensing conflict grows combinatorially with dependency count, and optimal license selection can be modeled as an integer programming problem minimizing obligation while maximizing ecosystem compatibility.*

---

## 1. License Compatibility as Partial Order (Order Theory)
### The Problem
When combining software components under different licenses, compatibility is not symmetric or transitive in the general case. The compatibility relation forms a partial order that determines which license combinations are valid in a derivative work.

### The Formula
Define a set of licenses $\mathcal{L} = \{l_1, l_2, \ldots, l_n\}$ and a compatibility relation $\preceq$ where $l_i \preceq l_j$ means code under $l_i$ can be incorporated into a work licensed under $l_j$.

The resulting license of a combined work must satisfy:

$$l_{result} = \sup\{l_1, l_2, \ldots, l_k\} \quad \text{in } (\mathcal{L}, \preceq)$$

If the supremum does not exist (incompatible licenses), the combination is forbidden:

$$\text{Compatible}(l_1, \ldots, l_k) \iff \exists l \in \mathcal{L} : \forall i, \; l_i \preceq l$$

The copyleft strength function $\sigma: \mathcal{L} \to [0, 1]$ assigns:

$$\sigma(l) = \begin{cases} 0 & \text{permissive (MIT, BSD)} \\ 0.3 & \text{file-level copyleft (MPL-2.0)} \\ 0.5 & \text{library copyleft (LGPL)} \\ 0.8 & \text{strong copyleft (GPL)} \\ 1.0 & \text{network copyleft (AGPL)} \end{cases}$$

### Worked Examples
**Example**: A project wants to combine components under MIT, Apache-2.0, and LGPL-2.1.

In the partial order: $\text{MIT} \preceq \text{Apache-2.0} \preceq \text{LGPL-2.1}$

$$l_{result} = \sup\{\text{MIT}, \text{Apache-2.0}, \text{LGPL-2.1}\} = \text{LGPL-2.1}$$

The combined work must be distributed under LGPL-2.1 (or a compatible stronger copyleft).

Now add a GPL-2.0-only component:

$$\sup\{\text{LGPL-2.1}, \text{GPL-2.0-only}\} = \text{GPL-2.0-only}$$

But Apache-2.0 is incompatible with GPL-2.0-only (patent clause conflict). No supremum exists:

$$\text{Compatible}(\text{MIT}, \text{Apache-2.0}, \text{LGPL-2.1}, \text{GPL-2.0-only}) = \text{false}$$

The combination is forbidden. Solution: replace the GPL-2.0-only component with a GPL-2.0-or-later component, which allows upgrading to GPL-3.0 (compatible with Apache-2.0).

## 2. Dependency Tree Risk Propagation (Graph Theory)
### The Problem
Modern software projects have deep dependency trees. A single copyleft license deep in the tree can propagate obligations to the entire project. The probability of a licensing conflict grows with the number of transitive dependencies.

### The Formula
Model the dependency graph as a DAG $G = (V, E)$ where each node $v$ has license $l(v)$.

The effective license obligation at the root $r$:

$$L(r) = \sup_{v \in \text{Reach}(r)} \{l(v) : \sigma(l(v)) > \tau(v, r)\}$$

where $\tau(v, r)$ is the isolation threshold (e.g., dynamic linking reduces copyleft propagation for LGPL).

The probability of at least one licensing conflict in a tree with $n$ independent dependencies, each with probability $p$ of being copyleft-incompatible:

$$P(\text{conflict}) = 1 - (1 - p)^n$$

For a dependency tree with depth $d$ and branching factor $b$:

$$n_{total} = \frac{b^{d+1} - 1}{b - 1}$$

$$P(\text{conflict}) = 1 - (1 - p)^{\frac{b^{d+1} - 1}{b - 1}}$$

### Worked Examples
**Example**: A Node.js project has 1,200 transitive dependencies (typical for a medium React app). The probability that any single dependency has an incompatible license is $p = 0.005$ (0.5%).

$$P(\text{conflict}) = 1 - (1 - 0.005)^{1{,}200} = 1 - 0.995^{1{,}200}$$

$$= 1 - e^{1{,}200 \ln(0.995)} = 1 - e^{-6.03} = 1 - 0.00241 = 0.9976$$

There is a 99.8% probability of at least one licensing conflict. This demonstrates why automated scanning is essential for modern projects.

For a Go project with 80 dependencies ($p = 0.005$):

$$P(\text{conflict}) = 1 - 0.995^{80} = 1 - 0.670 = 0.330$$

A 33% chance — still significant enough to require scanning.

## 3. Attribution Compliance Cost (Combinatorial Optimization)
### The Problem
Permissive licenses require attribution (copyright notice + license text). As dependency count grows, generating and maintaining accurate attribution documents becomes a combinatorial problem with cost implications.

### The Formula
The cost of attribution compliance for $n$ dependencies:

$$C_{attr} = \sum_{i=1}^{n} c_i \cdot d_i$$

where $c_i$ is the per-dependency compliance cost and $d_i$ is a difficulty multiplier based on:

$$d_i = \begin{cases} 1 & \text{SPDX-identified, standard license} \\ 2 & \text{custom license text, manual review} \\ 5 & \text{no license file, contact author required} \\ 10 & \text{license ambiguity, legal review needed} \end{cases}$$

The information entropy of the license distribution measures compliance complexity:

$$H(\mathcal{L}) = -\sum_{l \in \mathcal{L}} \frac{n_l}{N} \log_2 \frac{n_l}{N}$$

where $n_l$ is the count of dependencies under license $l$ and $N$ is total dependencies. Higher entropy means more diverse licensing and higher compliance overhead.

### Worked Examples
**Example**: A project has 200 dependencies with license distribution:

| License | Count | Proportion |
|---------|-------|------------|
| MIT | 120 | 0.60 |
| Apache-2.0 | 40 | 0.20 |
| BSD-3-Clause | 20 | 0.10 |
| ISC | 10 | 0.05 |
| Other/Custom | 10 | 0.05 |

$$H = -(0.60 \log_2 0.60 + 0.20 \log_2 0.20 + 0.10 \log_2 0.10 + 0.05 \log_2 0.05 + 0.05 \log_2 0.05)$$
$$= -(0.60 \times -0.737 + 0.20 \times -2.322 + 0.10 \times -3.322 + 0.05 \times -4.322 + 0.05 \times -4.322)$$
$$= -(- 0.442 - 0.464 - 0.332 - 0.216 - 0.216) = 1.670 \text{ bits}$$

Compare to a project with 200 MIT-only dependencies: $H = 0$ bits (minimal compliance complexity).

The 10 custom-license dependencies at $d_i = 5$ dominate compliance cost despite being only 5% of the total.

## 4. Copyleft Boundary Analysis (Set Theory)
### The Problem
For weak copyleft licenses (LGPL, MPL-2.0), the boundary between what must be disclosed and what can remain proprietary depends on the coupling between components. Formally defining this boundary prevents both over-disclosure and non-compliance.

### The Formula
Define the copyleft boundary function $B: \mathcal{C} \to \{0, 1\}$ for each code component $c$:

$$B(c) = \begin{cases} 1 & \text{if } c \text{ is a modification of copyleft component} \\ 1 & \text{if } c \text{ is within the copyleft scope (GPL: derivative work)} \\ 0 & \text{if } c \text{ is outside the copyleft boundary} \end{cases}$$

For MPL-2.0, scope is file-level:

$$B_{MPL}(c) = \begin{cases} 1 & \text{if } c \in F_{MPL} \text{ (modified MPL files)} \\ 0 & \text{otherwise} \end{cases}$$

For LGPL, scope is the library boundary:

$$B_{LGPL}(c) = \begin{cases} 1 & \text{if } c \in L \cup \Delta L \text{ (library + modifications)} \\ 0 & \text{if } c \in A \setminus L \text{ (application using library)} \end{cases}$$

provided the linking mechanism allows substitution ($c$ dynamically links to $L$).

The "derivative work" determination for GPL uses the coupling metric:

$$\kappa(c, L) = \frac{|I(c) \cap API(L)|}{|I(c)|}$$

where $I(c)$ is the set of interfaces used by $c$ and $API(L)$ is the API surface of library $L$. Higher coupling strengthens the derivative work argument.

### Worked Examples
**Example**: An application uses an LGPL-2.1 library via dynamic linking. The app calls 15 of the library's 200 public API functions.

$$\kappa = \frac{15}{200} = 0.075$$

Low coupling (7.5%). Combined with dynamic linking, this clearly falls outside the copyleft boundary. The application can be proprietary.

If the application instead modifies the library source and statically links:

$$B_{LGPL}(app) = 1 \quad \text{(static linking triggers disclosure obligation)}$$

The application must either switch to dynamic linking or provide object files allowing relinking with a modified library.

## 5. License Selection Optimization (Integer Programming)
### The Problem
When publishing a new open source project, the license choice affects adoption, contribution, and compatibility with the ecosystem. This can be modeled as an optimization problem maximizing compatibility with target ecosystems while meeting business constraints.

### The Formula
Define binary decision variables $x_l \in \{0, 1\}$ for each candidate license $l \in \mathcal{L}$, where $x_l = 1$ means license $l$ is selected (choose exactly one):

$$\max \sum_{l \in \mathcal{L}} x_l \cdot \left(\alpha \cdot \text{compat}(l) + \beta \cdot \text{adopt}(l) + \gamma \cdot \text{protect}(l)\right)$$

Subject to:

$$\sum_{l \in \mathcal{L}} x_l = 1 \quad \text{(exactly one license)}$$
$$x_l \cdot \sigma(l) \leq \sigma_{max} \quad \forall l \quad \text{(copyleft strength limit)}$$
$$x_l \cdot (1 - \text{patent}(l)) = 0 \quad \text{if patent protection required}$$

Where:
- $\text{compat}(l)$ = fraction of ecosystem dependencies compatible with $l$
- $\text{adopt}(l)$ = historical adoption rate (proxy for community willingness)
- $\text{protect}(l)$ = intellectual property protection score

### Worked Examples
**Example**: A company wants to open-source a library. Requirements: patent protection, maximum ecosystem compatibility, moderate IP protection. Weights: $\alpha = 0.5$, $\beta = 0.3$, $\gamma = 0.2$.

| License | compat | adopt | protect | patent | Score |
|---------|--------|-------|---------|--------|-------|
| MIT | 0.95 | 0.90 | 0.10 | no | N/A (fails patent req) |
| Apache-2.0 | 0.85 | 0.70 | 0.30 | yes | $0.5(0.85) + 0.3(0.70) + 0.2(0.30) = 0.695$ |
| MPL-2.0 | 0.80 | 0.40 | 0.50 | yes | $0.5(0.80) + 0.3(0.40) + 0.2(0.50) = 0.620$ |
| LGPL-3.0 | 0.60 | 0.30 | 0.60 | yes | $0.5(0.60) + 0.3(0.30) + 0.2(0.60) = 0.510$ |
| GPL-3.0 | 0.40 | 0.25 | 0.80 | yes | $0.5(0.40) + 0.3(0.25) + 0.2(0.80) = 0.435$ |

Optimal selection: **Apache-2.0** with score 0.695.

## 6. SBOM Completeness and Vulnerability Exposure (Probabilistic Coverage)
### The Problem
A Software Bill of Materials (SBOM) must capture all dependencies for license compliance. Incomplete SBOMs leave unknown licenses and vulnerabilities. The probability of missing a critical dependency grows with the gap between actual and documented components.

### The Formula
Let $D$ be the total number of actual dependencies and $S$ be the number captured in the SBOM. The coverage ratio:

$$C = \frac{S}{D}$$

If each undocumented dependency has independent probability $q$ of containing a license violation, the probability of at least one undetected violation:

$$P(\text{violation}) = 1 - (1 - q)^{D - S} = 1 - (1 - q)^{D(1 - C)}$$

The expected number of undetected violations:

$$E[V] = (D - S) \cdot q = D(1 - C) \cdot q$$

### Worked Examples
**Example**: A project has $D = 500$ actual dependencies. The SBOM captures $S = 450$ ($C = 0.90$). Each undocumented dependency has a 3% chance of containing a copyleft license ($q = 0.03$).

$$P(\text{violation}) = 1 - (1 - 0.03)^{50} = 1 - 0.97^{50} = 1 - 0.218 = 0.782$$

78.2% chance of at least one undetected copyleft dependency. Expected violations: $50 \times 0.03 = 1.5$.

At 99% coverage ($S = 495$, 5 undocumented):

$$P(\text{violation}) = 1 - 0.97^5 = 1 - 0.859 = 0.141$$

14.1% risk. At 100% SBOM coverage, $P(\text{violation}) = 0$. The steep reduction from 78% to 14% between 90% and 99% coverage demonstrates the non-linear return on SBOM completeness investment.

## Prerequisites
- order-theory, graph-theory, set-theory, combinatorics, information-theory, integer-programming, probability-theory, optimization, coverage-analysis
