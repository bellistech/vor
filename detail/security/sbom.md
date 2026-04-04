# The Mathematics of SBOMs -- Dependency Graphs and Supply Chain Completeness

> *Software Bills of Materials encode dependency relationships as directed acyclic graphs, where completeness metrics quantify coverage against NTIA minimum elements, vulnerability exposure scales with transitive closure size, and VEX documents apply Boolean filtering to reduce false positive rates in automated scanning.*

---

## 1. Dependency Graph Structure (Graph Theory)

### The Problem

An SBOM represents software dependencies as a directed acyclic graph (DAG). Understanding the structure of this graph -- its depth, branching factor, and total node count -- determines the scope of vulnerability exposure and the computational cost of analysis.

### The Formula

Dependency DAG $G = (V, E)$ where $V$ = components and $E$ = dependency edges.

In-degree (dependents) and out-degree (dependencies) of node $v$:

$$\text{deg}^+(v) = |\{u : (v, u) \in E\}|, \quad \text{deg}^-(v) = |\{u : (u, v) \in E\}|$$

Transitive closure (all reachable dependencies from root $r$):

$$T(r) = \{v \in V : \exists \text{ path } r \to v \text{ in } G\}$$

DAG depth:

$$d(G) = \max_{v \in V} \min_{p \in \text{paths}(r, v)} |p|$$

Total component count for tree-like DAGs with branching factor $b$:

$$|V| \approx \frac{b^{d+1} - 1}{b - 1}$$

### Worked Examples

Go project SBOM: 12 direct dependencies, average branching factor $b = 4$, depth $d = 4$.

$$|V| \approx \frac{4^5 - 1}{3} = \frac{1023}{3} = 341 \text{ components}$$

Direct: 12. Transitive: $341 - 12 - 1 = 328$.

Transitive amplification: $\frac{328}{12} = 27.3\times$ more hidden dependencies than declared.

npm project: $b = 12$, $d = 7$:

$$|V| \approx \frac{12^8 - 1}{11} = \frac{429{,}981{,}695}{11} \approx 39{,}089{,}245$$

(With deduplication, real npm projects typically have 500-2000 unique packages due to shared transitive deps.)

---

## 2. NTIA Completeness Score (Set Coverage)

### The Problem

The NTIA minimum elements define 7 required fields for each SBOM component. Completeness is measured as the fraction of components that satisfy all required fields, and the fraction of fields populated across all components.

### The Formula

Required fields set $F = \{f_1, \ldots, f_7\}$ (supplier, name, version, unique ID, relationships, author, timestamp).

For component $c$, populated fields:

$$\text{pop}(c) = \{f \in F : f(c) \neq \emptyset\}$$

Component-level completeness:

$$\text{complete}(c) = \begin{cases} 1 & |\text{pop}(c)| = |F| \\ 0 & \text{otherwise} \end{cases}$$

SBOM completeness score:

$$C_{\text{component}} = \frac{\sum_{c \in V} \text{complete}(c)}{|V|}$$

Field-level completeness:

$$C_{\text{field}}(f) = \frac{|\{c \in V : f(c) \neq \emptyset\}|}{|V|}$$

Overall quality score (sbomqs-style):

$$Q = \frac{1}{|F|} \sum_{f \in F} C_{\text{field}}(f)$$

### Worked Examples

SBOM with 200 components:

| Field | Populated | Coverage |
|:---|:---:|:---:|
| Supplier Name | 150 | 75.0% |
| Component Name | 200 | 100.0% |
| Version | 195 | 97.5% |
| Unique ID (PURL) | 180 | 90.0% |
| Relationships | 160 | 80.0% |
| Author | 200 | 100.0% |
| Timestamp | 200 | 100.0% |

$$Q = \frac{0.75 + 1.0 + 0.975 + 0.90 + 0.80 + 1.0 + 1.0}{7} = \frac{6.425}{7} = 91.8\%$$

Components with all 7 fields: 140 out of 200.

$$C_{\text{component}} = \frac{140}{200} = 70\%$$

---

## 3. Vulnerability Exposure Surface (Combinatorics)

### The Problem

The total vulnerability exposure of an application is the union of vulnerabilities affecting all components in the transitive closure. The expected exposure grows with component count, and shared dependencies create overlapping vulnerability sets.

### The Formula

For component set $V$ with vulnerability sets $\text{vuln}(c_i)$:

$$\text{Exposure} = \left|\bigcup_{c \in V} \text{vuln}(c)\right|$$

By inclusion-exclusion:

$$|\text{Exposure}| = \sum_{i}|\text{vuln}(c_i)| - \sum_{i<j}|\text{vuln}(c_i) \cap \text{vuln}(c_j)| + \cdots$$

Expected exposure with independent vulnerability probability $p$ per component and $N$ total known CVEs:

$$E[|\text{Exposure}|] = N \cdot \left(1 - (1-p)^{|V|}\right)$$

### Worked Examples

$|V| = 341$ components, $N = 250{,}000$ CVEs in database, $p = 0.00002$ (probability any CVE affects any component).

$$E[|\text{Exposure}|] = 250{,}000 \times (1 - (1 - 0.00002)^{341})$$

$$= 250{,}000 \times (1 - 0.99998^{341})$$

$$= 250{,}000 \times (1 - 0.99320)$$

$$= 250{,}000 \times 0.00680 = 1{,}700 \text{ CVEs}$$

With only direct dependencies ($|V| = 12$):

$$E = 250{,}000 \times (1 - 0.99998^{12}) = 250{,}000 \times 0.000240 = 60 \text{ CVEs}$$

Ratio: $\frac{1{,}700}{60} = 28.3\times$ more exposure from transitive dependencies.

---

## 4. VEX Filtering (Boolean Logic)

### The Problem

VEX (Vulnerability Exploitability eXchange) documents declare the exploitability status of vulnerabilities for specific products. Applying VEX to scan results reduces false positives by filtering out non-applicable findings using logical predicates.

### The Formula

VEX statement: $(v, p, s, j)$ where $v$ = vulnerability, $p$ = product, $s \in \{\text{not\_affected}, \text{affected}, \text{fixed}, \text{under\_investigation}\}$, $j$ = justification.

Filtered scan results:

$$R_{\text{filtered}} = R \setminus \{r \in R : \exists (v, p, s, j) \in \text{VEX} : r.v = v \wedge r.p \subseteq p \wedge s = \text{not\_affected}\}$$

False positive reduction:

$$\text{FPR} = \frac{|R| - |R_{\text{filtered}}|}{|R|}$$

Residual noise (remaining false positives):

$$\text{residual\_FP} = |R_{\text{filtered}}| - |\text{true\_positives}(R_{\text{filtered}})|$$

### Worked Examples

Scanner reports $|R| = 85$ findings.

VEX document contains 30 `not_affected` statements, 5 `fixed`, 3 `under_investigation`.

Matched against results: 28 of 85 findings match `not_affected` VEX entries.

$$|R_{\text{filtered}}| = 85 - 28 = 57$$

$$\text{FPR} = \frac{28}{85} = 32.9\%$$

Of 57 remaining, 45 are true positives, 12 are unaddressed false positives.

$$\text{residual\_FP} = 57 - 45 = 12$$

Signal-to-noise ratio improvement: $\frac{45}{85} = 52.9\% \to \frac{45}{57} = 78.9\%$

---

## 5. SPDX vs CycloneDX Expressiveness (Information Theory)

### The Problem

SPDX and CycloneDX encode overlapping but distinct information about software components. The expressiveness of each format can be compared by the number of distinct fields and relationship types they support, and the information density of generated SBOMs.

### The Formula

Information content of an SBOM:

$$H(\text{SBOM}) = -\sum_{f \in \text{fields}} P(f) \log_2 P(f)$$

where $P(f)$ is the probability that field $f$ provides non-trivial (non-default) information.

Field coverage ratio between formats:

$$\text{overlap} = \frac{|F_{\text{SPDX}} \cap F_{\text{CDX}}|}{|F_{\text{SPDX}} \cup F_{\text{CDX}}|}$$

Compression ratio (practical density):

$$\rho = \frac{\text{unique\_info\_bits}}{\text{total\_file\_size\_bits}}$$

### Worked Examples

SPDX 2.3 defines ~50 package fields, 11 relationship types.
CycloneDX 1.6 defines ~45 component fields, 25 dependency relationship types, plus VEX, services, formulation.

Estimated overlap: 35 common concepts out of 60 unique:

$$\text{overlap} = \frac{35}{60} = 58.3\%$$

For the same 200-component application:

| Format | File Size | Unique Fields Used | Density |
|:---|:---:|:---:|:---:|
| SPDX JSON | 2.1 MB | 28 fields | 13.3 bits/byte |
| CycloneDX JSON | 1.8 MB | 32 fields | 17.8 bits/byte |

CycloneDX achieves higher density due to nested component structure vs SPDX's flat package list with external relationships.

---

## 6. Executive Order Compliance (Decision Theory)

### The Problem

US Executive Order 14028 mandates SBOM delivery for federal software procurement. Organizations must decide the optimal SBOM generation frequency, format, and completeness level to minimize compliance cost while meeting minimum requirements.

### The Formula

Compliance cost function:

$$C(\text{freq}, \text{fields}, \text{format}) = c_{\text{gen}} \cdot \text{freq} + c_{\text{review}} \cdot |\text{fields}| + c_{\text{tool}} \cdot |\text{formats}|$$

Risk of non-compliance (penalty cost $P$, probability of audit $a$):

$$E[\text{penalty}] = a \cdot P \cdot (1 - C_{\text{component}})$$

Optimal generation frequency (minimize total cost):

$$\text{freq}^* = \arg\min_f \left(c_{\text{gen}} \cdot f + \frac{E[\text{new\_vulns}]}{f}\right)$$

Taking derivative and solving:

$$f^* = \sqrt{\frac{E[\text{new\_vulns}]}{c_{\text{gen}}}}$$

### Worked Examples

Generation cost: $c_{\text{gen}} = \$50$ per SBOM generation (CI time + storage).

New vulnerabilities affecting project: $E[\text{new\_vulns}] = 3$ per month.

Cost per undetected vulnerability: $\$5{,}000$.

$$f^* = \sqrt{\frac{3 \times 5{,}000}{50}} = \sqrt{300} \approx 17.3 \text{ per month}$$

Rounding: generate SBOM on every CI build (approximately daily) for cost-optimal compliance.

Annual cost: $17 \times 12 \times \$50 = \$10{,}200$ vs potential penalty of $3 \times 12 \times \$5{,}000 = \$180{,}000$.

---

## Prerequisites

- graph-theory, set-theory, combinatorics, boolean-logic, information-theory, decision-theory, probability
