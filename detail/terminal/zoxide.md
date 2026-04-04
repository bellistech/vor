# The Mathematics of zoxide -- Frecency Scoring and Directory Ranking

> *zoxide solves a ranked retrieval problem: given a short query and a database of previously visited directories, score each candidate by a frecency function that blends visit frequency with temporal recency, returning the highest-scoring match in sublinear time.*

---

## 1. Frecency Model (Time-Weighted Frequency)

### The Problem

A user visits $N$ distinct directories over time. Each directory $d_i$ has a visit history $\{t_1, t_2, \ldots, t_{k_i}\}$. Given a query $q$, rank all matching directories by how likely the user wants to visit them now.

Pure frequency fails because abandoned directories accumulate stale counts. Pure recency fails because a one-time visit would outrank daily destinations.

### The Formula

zoxide computes a frecency score for each directory:

$$S(d) = \sum_{j=1}^{k} w(t_{now} - t_j)$$

where $w(\Delta t)$ is a time-decay weight function:

$$w(\Delta t) = \begin{cases} 4 & \text{if } \Delta t < 1\text{ hour} \\ 2 & \text{if } \Delta t < 1\text{ day} \\ 1 & \text{if } \Delta t < 1\text{ week} \\ 0.5 & \text{if } \Delta t \geq 1\text{ week} \end{cases}$$

### Worked Examples

Directory `/home/user/projects/api` visited 3 times: 30 min ago, 5 hours ago, 3 days ago.

$$S = w(30\text{m}) + w(5\text{h}) + w(3\text{d}) = 4 + 2 + 1 = 7$$

Directory `/home/user/old-project` visited 10 times, all 2 weeks ago:

$$S = 10 \times w(2\text{w}) = 10 \times 0.5 = 5$$

The recently active directory (score 7) outranks the historically frequent one (score 5).

---

## 2. Aging and Normalization (Score Decay)

### The Problem

Without bounds, scores grow without limit as the database accumulates visits. zoxide caps total score via an aging mechanism.

### The Formula

When the sum of all scores exceeds a threshold $M$ (default $M = 10{,}000$):

$$S'(d_i) = \alpha \cdot S(d_i) \quad \forall d_i$$

where $\alpha = 0.9$ is the decay factor. Entries falling below a minimum threshold $\epsilon = 1$ are pruned:

$$\text{if } S'(d_i) < \epsilon, \text{ remove } d_i$$

### Convergence

After repeated aging events, the steady-state total score converges to:

$$S_{total}^{*} = \frac{r}{1 - \alpha} \cdot M$$

where $r$ is the rate of new score accumulation between aging events. The geometric decay ensures the database remains bounded:

$$S_{total}(n) = M \cdot \alpha^n + \frac{r(1 - \alpha^n)}{1 - \alpha}$$

### Worked Examples

Database has 500 entries totaling $S_{total} = 10{,}200 > M$.

After aging: $S_{total}' = 0.9 \times 10{,}200 = 9{,}180$

An entry with $S = 1.05$ becomes $S' = 0.945 < 1$, so it is pruned.

---

## 3. Query Matching (Subsequence Search)

### The Problem

Given query keywords $q_1, q_2, \ldots, q_k$ and a directory path $p = c_1/c_2/\ldots/c_n$ (split by `/`), determine if $p$ matches and compute match quality.

### The Formula

zoxide splits both query and path into components. The match requires each query keyword to appear as a substring of some path component, in order:

$$\text{match}(q, p) = \exists\, i_1 < i_2 < \cdots < i_k \text{ such that } q_j \subseteq c_{i_j} \;\forall\, j$$

where $\subseteq$ denotes substring containment:

$$q_j \subseteq c_i \iff \exists\, s : c_i[s:s+|q_j|] = q_j$$

The last keyword must match the last path component:

$$q_k \subseteq c_n$$

This constraint dramatically reduces false positives.

### Worked Examples

Query: `z foo bar`

- `/home/user/foo/project/bar` -- MATCH ($q_1$=foo in component 3, $q_2$=bar in component 5, last component matches)
- `/home/user/bar/foo` -- NO MATCH (bar appears before foo, order violated)
- `/home/user/foobar` -- NO MATCH (bar is not in the last component separately)

---

## 4. Ranking with Tie-Breaking (Total Order)

### The Problem

Multiple directories may match a query. The ranking must produce a total order.

### The Formula

Candidates are sorted by the tuple:

$$(S(d), -\text{depth}(d), \text{lexicographic}(d))$$

Highest score wins. On ties, shallower paths are preferred (shorter paths are more likely targets). Final tie-break is lexicographic for determinism.

$$\text{rank}(d_i) < \text{rank}(d_j) \iff (S(d_i), -\text{depth}(d_i)) >_{lex} (S(d_j), -\text{depth}(d_j))$$

---

## 5. Database Efficiency (Amortized Complexity)

### Insert (cd Hook)

Every directory change triggers:

$$T_{insert} = O(\log N) \quad \text{(lookup + update in sorted store)}$$

### Query

For a query with $k$ keywords against $N$ database entries:

$$T_{query} = O(N \times L \times k)$$

where $L$ is the average path length. With prefix filtering:

$$T_{query}^{opt} = O(|matches| \times L \times k) \ll O(N \times L \times k)$$

### Aging

Aging traverses the entire database:

$$T_{age} = O(N)$$

But aging occurs only when $S_{total} > M$, which happens at most every $\lfloor M / r \rfloor$ insertions. Amortized cost per insert:

$$T_{age}^{amort} = O\left(\frac{N}{M/r}\right) = O\left(\frac{Nr}{M}\right)$$

---

## 6. Information-Theoretic Perspective

### Query Efficiency

Each keyword narrows the candidate set. The information gained per keyword:

$$I(q_j) = \log_2 \frac{|C_{j-1}|}{|C_j|}$$

where $C_j$ is the candidate set after $j$ keywords.

For typical development environments ($N \approx 500$ directories):

$$\text{Expected keystrokes to unique match} \approx \frac{\log_2 N}{\log_2 \sigma} \approx \frac{9}{4.7} \approx 2$$

where $\sigma$ is the effective alphabet size of path components. Two keywords almost always suffice.

### Comparison with Uniform Search

Without frecency, a user must type enough characters to uniquely identify a path:

$$k_{uniform} = \lceil \log_\sigma N \rceil$$

With frecency, the top-ranked result is usually correct with fewer characters because the scoring disambiguates:

$$k_{frecency} \leq k_{uniform} - \lfloor \log_\sigma(\text{score\_ratio}) \rfloor$$

---

## 7. Comparison with Alternative Algorithms

### Exponential Decay vs Step Function

An alternative decay model uses continuous exponential decay:

$$w(\Delta t) = e^{-\lambda \Delta t}$$

| Property | Step Function (zoxide) | Exponential Decay |
|:---|:---|:---|
| Computation | $O(1)$ lookup | $O(1)$ with $e^x$ |
| Tuning | 4 discrete weights | Single $\lambda$ parameter |
| Interpretability | Clear time buckets | Smooth but opaque |
| Score stability | Jumps at boundaries | Continuous updates |

zoxide's step function is computationally simpler and produces more predictable rankings.

### Bayesian Approach

A Bayesian model would compute:

$$P(d \mid q, \text{history}) \propto P(q \mid d) \cdot P(d \mid \text{history})$$

where $P(d \mid \text{history})$ is the frecency prior and $P(q \mid d)$ is the match likelihood. zoxide approximates this by using the frecency score as both prior and posterior.

---

## Prerequisites

- time series analysis, exponential decay, amortized analysis, subsequence matching, information theory, ranking algorithms

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Insert (cd hook) | $O(\log N)$ | $O(1)$ |
| Query (k keywords) | $O(N \times L \times k)$ | $O(N)$ |
| Aging | $O(N)$ amortized | $O(1)$ |
| Interactive select | $O(N \log N)$ | $O(N)$ |
| Import | $O(M)$ per source entry | $O(M)$ |

---

*Frecency is a pragmatic Bayesian prior -- it encodes the observation that human directory access follows a power-law distribution where recent behavior is a stronger predictor of future intent than cumulative history.*
