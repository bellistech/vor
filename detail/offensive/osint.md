# The Mathematics of OSINT — Graph Theory and Information Correlation

> *Open-source intelligence is fundamentally a graph problem: entities (people, domains, IPs, organizations) form nodes, and relationships (owns, hosts, employs, links-to) form edges. The intelligence analyst's task is to traverse, correlate, and cluster this graph to extract actionable knowledge from the combinatorial space of publicly available data.*

---

## 1. Entity-Relationship Graphs (Knowledge Representation)

### OSINT Graph Model

An OSINT knowledge graph is a directed labeled multigraph:

$$G = (V, E, \lambda_V, \lambda_E)$$

where $V$ is the set of entities, $E \subseteq V \times V$ is relationships, $\lambda_V$ assigns entity types, and $\lambda_E$ assigns relationship types.

| Entity Type | Examples | Typical Sources |
|:---|:---|:---|
| Person | Names, handles, emails | LinkedIn, social media, WHOIS |
| Organization | Companies, groups | Business registries, LinkedIn |
| Domain | target.com, sub.target.com | DNS, CT logs, WHOIS |
| IP Address | IPv4, IPv6, ranges | Shodan, Censys, BGP |
| Credential | Email:password pairs | Breach databases |
| Document | PDFs, images, files | Google dorking, web crawling |

Graph density: $\text{Density} = |E| / n(n-1)$. OSINT graphs are sparse ($< 0.1$) but exhibit clustering around organizational boundaries.

---

## 2. Subdomain Enumeration — Set Coverage

### Source Coverage Analysis

Let $S_i$ be the set of subdomains discovered by source $i$:

$$\text{Total unique} = \left|\bigcup_{i=1}^{k} S_i\right|$$

Jaccard similarity between sources: $J(S_i, S_j) = |S_i \cap S_j| / |S_i \cup S_j|$

Low Jaccard means the sources are complementary.

| Source | Avg. Unique Contribution | Overlap |
|:---|:---:|:---:|
| Certificate Transparency | 60-80% of total | High base coverage |
| Passive DNS | 20-40% | Moderate overlap with CT |
| Search engine scraping | 10-25% | Low overlap |
| DNS brute force | 15-30% | Low (finds non-web hosts) |
| Web crawling | 5-15% | Finds app-specific subs |

### Diminishing Returns

Marginal discovery rate: $\Delta(k) \propto k^{-\alpha}$ with $\alpha \approx 1.5$.

After 5-6 sources, each additional source typically contributes $< 5\%$ new subdomains.

---

## 3. Certificate Transparency — Merkle Tree Verification

### CT Log Structure

CT logs are append-only Merkle hash trees:

$$H_{\text{leaf}} = \text{SHA-256}(0x00 \| \text{cert}_i)$$
$$H_{\text{node}} = \text{SHA-256}(0x01 \| H_{\text{left}} \| H_{\text{right}})$$

Inclusion proof for $n$ entries: $\lceil\log_2 n\rceil$ hashes.

| CT Log | Entries (approx.) | Tree Height |
|:---|:---:|:---:|
| Google Argon | $> 10^9$ | ~30 |
| Let's Encrypt Oak | $> 10^9$ | ~30 |

Domains using Let's Encrypt (90-day renewal) generate ~4 certificates per year per subdomain.

### Subdomain Discovery via CT

For a domain with $d$ subdomains, the probability of finding a specific subdomain in CT logs depends on whether certificates have been issued for it:

$$P(\text{sub in CT}) = 1 - e^{-c_d / N}$$

where $c_d$ is certificates issued and $N$ is total log size. For any subdomain with at least one certificate: $P \approx 1$.

---

## 4. Search Engine Dorking — Information Retrieval

### Query Precision and Recall

$$\text{Precision} = \frac{|\text{relevant results}|}{|\text{total results}|}, \quad \text{Recall} = \frac{|\text{relevant results}|}{|\text{total relevant docs}|}$$

| Operator | Precision | Recall | Best For |
|:---|:---:|:---:|:---|
| `site:` | High | Medium | Scoping to target domain |
| `filetype:` | High | Low | Finding specific file types |
| `intitle:` | Medium | Medium | Finding page types |
| Combined | Very High | Low | Targeted discovery |

### Dork Combinatorial Space

With $o$ operators and $v$ values per operator:

$$\text{Possible queries} = \sum_{k=1}^{o} \binom{o}{k} \times v^k$$

For 6 operators with 10 values: $1{,}111{,}110$ possible queries.

---

## 5. Breach Data Correlation — Set Intersection

### Credential Reuse Rate

$$R_{\text{reuse}} = \frac{|\{(e,p) \in B_i \cap B_j, i \neq j\}|}{|\bigcup B_i|}$$

### k-Anonymity for Password Checking (HIBP)

Send first 5 hex chars of SHA-1(password): 20-bit prefix.

$$|\text{Response set}| \approx \frac{6 \times 10^8}{2^{20}} \approx 600 \text{ hashes}$$

Information leaked: 20 bits out of 160. Server cannot identify which suffix the client checks.

---

## 6. Image OSINT — Geolocation Mathematics

### EXIF GPS Precision

| Decimal Places | Precision | Use Case |
|:---|:---:|:---|
| 1 (0.1) | 11.1 km | Country/region |
| 3 (0.001) | 111 m | Neighborhood |
| 5 (0.00001) | 1.1 m | Building level |

### Visual Geolocation (No EXIF)

$$P(\text{location} | \text{features}) \propto P(\text{features} | \text{location}) \times P(\text{location})$$

| Feature | Specificity | Example |
|:---|:---:|:---|
| Language on signs | Country | Cyrillic, Arabic, Hangul |
| Driving side | Hemisphere | Left (UK, Japan) vs Right |
| Architecture style | Region | Tudor, Mediterranean |
| License plates | Country/state | Format, color, size |

---

## 7. Network Topology Inference

### AS-Level Graph

$$|V_{AS}| \approx 75{,}000, \quad |E_{AS}| \approx 350{,}000$$

For a target with $h$ hosts, probability of finding at least one vulnerable version:

$$P(\text{vuln found}) = 1 - \prod_{i=1}^{h} (1 - p_i)$$

| Scanner | IPv4 Coverage | Scan Frequency |
|:---|:---:|:---:|
| Shodan | ~95% of routable | Weekly |
| Censys | ~100% of routable | Daily |
| ZoomEye | ~90% of routable | Weekly |

---

## 8. OSINT Completeness — Capture-Recapture Estimation

### Estimating Total Entities

From two overlapping sources:

$$\hat{N} = \frac{n_1 \times n_2}{n_{12}}, \quad \text{Var}(\hat{N}) = \frac{n_1 n_2 (n_1 - n_{12})(n_2 - n_{12})}{n_{12}^3}$$

### Worked Example

Source A: 150 subdomains. Source B: 200 subdomains. Overlap: 100.

$$\hat{N} = \frac{150 \times 200}{100} = 300, \quad \text{Undiscovered} \approx 300 - 250 = 50$$

| Metric | Value |
|:---|:---:|
| Source A unique | 50 |
| Source B unique | 100 |
| Overlap | 100 |
| Union | 250 |
| Estimated total | 300 |
| Estimated undiscovered | 50 |

---

*The fundamental insight of OSINT mathematics is that no single source provides complete coverage, but the overlap between sources enables statistical estimation of what remains undiscovered. The capture-recapture model transforms OSINT from a collection exercise into a quantifiable intelligence problem with measurable confidence bounds. The analyst who understands source complementarity allocates collection effort optimally across the exponential space of publicly available information.*

## Prerequisites

- Graph theory basics (nodes, edges, traversal, clustering)
- Set theory (union, intersection, Jaccard similarity)
- Basic probability and statistics (Bayes' theorem, confidence intervals)

## Complexity

- **Beginner:** Using individual OSINT tools (Shodan, crt.sh, theHarvester) and interpreting results
- **Intermediate:** Combining multiple sources, estimating coverage gaps, building entity-relationship graphs
- **Advanced:** Capture-recapture estimation, optimal source allocation, automated correlation and clustering
