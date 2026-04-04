# The Mathematics of Parquet — Columnar Compression and Query Optimization

> *Parquet's columnar layout exploits a fundamental property of real-world data: values within a column are far more homogeneous than values within a row. This homogeneity, measurable as lower Shannon entropy per column versus per row, enables compression ratios and query performance that row-oriented formats cannot achieve. The Dremel encoding for nested data uses repetition and definition levels to flatten arbitrary tree structures into flat column arrays.*

---

## 1. Columnar vs Row-Oriented Storage (Information Theory)

### The Problem

Why does storing data by column compress better than storing by row? Can we quantify the compression advantage mathematically?

### The Formula

Shannon entropy measures the information content of a data stream. For a column $C$ with values drawn from alphabet $\mathcal{A}$ with probability distribution $p$:

$$H(C) = -\sum_{a \in \mathcal{A}} p(a) \log_2 p(a)$$

For a row containing $k$ columns with different types and distributions:

$$H(\text{row}) = \sum_{i=1}^{k} H(C_i)$$

The key insight: column-wise entropy is equal in total, but each individual column stream has lower entropy than a mixed row stream, enabling specialized encoders:

$$H_{\text{dict}}(C_i) \leq H(C_i) \quad \text{when } |\mathcal{A}_i| \ll |C_i|$$

Dictionary encoding replaces values with indices, achieving:

$$B_{\text{dict}}(C) = |\mathcal{A}| \cdot \bar{w} + |C| \cdot \lceil \log_2 |\mathcal{A}| \rceil$$

Where $\bar{w}$ is average value width and $|C|$ is the number of values.

### Worked Examples

**Column of 1M country codes (200 unique values, avg 3 bytes each):**

Without dictionary: $1{,}000{,}000 \times 3 = 3{,}000{,}000$ bytes = 3 MB

With dictionary:
$$B_{\text{dict}} = 200 \times 3 + 1{,}000{,}000 \times \lceil \log_2 200 \rceil / 8$$
$$= 600 + 1{,}000{,}000 \times 8 / 8 = 600 + 1{,}000{,}000 = 1{,}000{,}600 \text{ bytes} \approx 1 \text{ MB}$$

Compression ratio: $3{,}000{,}000 / 1{,}000{,}600 \approx 3.0\times$

With RLE on sorted data (10 countries dominate, 5000 runs):
$$B_{\text{RLE}} = 5{,}000 \times (8 + 4) = 60{,}000 \text{ bytes} = 60 \text{ KB}$$

Overall: $3 \text{ MB} \to 60 \text{ KB} = 50\times$ compression.

---

## 2. Predicate Pushdown (Statistics-Based Skipping)

### The Problem

Parquet stores min/max statistics per column chunk and per page. A query predicate can skip entire row groups if the predicate is unsatisfiable given the statistics. What is the expected speedup?

### The Formula

For a predicate $P: v \in [a, b]$ on a column with global range $[L, U]$ split into $G$ row groups, each with local range $[\min_g, \max_g]$:

A row group $g$ is skipped iff:

$$\max_g < a \quad \text{or} \quad \min_g > b$$

If data is uniformly distributed and the column is unsorted, the probability of skipping a row group:

$$P(\text{skip}) = P(\max_g < a) + P(\min_g > b)$$

For a uniform distribution over $[L, U]$ with $n$ values per row group:

$$P(\max_g < a) = \left(\frac{a - L}{U - L}\right)^n$$

For sorted data, the expected fraction of row groups matching a range query $[a, b]$:

$$f_{\text{match}} = \frac{b - a}{U - L} + \frac{2}{G}$$

The $\frac{2}{G}$ term accounts for the two boundary row groups that partially match.

### Worked Examples

**Unsorted timestamp column, 10 row groups, each with 100K rows, querying 1 day out of 365:**

$$P(\text{skip per group}) \approx 1 - \frac{1}{365} = 99.7\%$$

But with random data, each group likely spans the full range:
$$P(\max_g < a) = \left(\frac{1}{365}\right)^{100{,}000} \approx 0$$

Unsorted data: virtually no skipping. This is why sorting matters.

**Sorted timestamp column, 10 row groups, querying 1 day of 365:**

$$f_{\text{match}} = \frac{1}{365} + \frac{2}{10} = 0.003 + 0.2 = 0.203$$

About 2-3 of 10 row groups read. Speedup: $\frac{10}{2.03} \approx 4.9\times$

**With 100 row groups (smaller groups):**

$$f_{\text{match}} = \frac{1}{365} + \frac{2}{100} = 0.023$$

About 2-3 of 100 row groups. Speedup: $\frac{100}{2.3} \approx 43\times$

---

## 3. Column Pruning (I/O Reduction)

### The Problem

If a query only needs $k$ of $n$ columns, Parquet reads only those $k$ column chunks. What is the I/O reduction?

### The Formula

Let $s_i$ be the compressed size of column $i$. Total file size:

$$S_{\text{total}} = \sum_{i=1}^{n} s_i + M$$

Where $M$ is metadata (footer, offsets). I/O for a query needing columns $Q \subseteq [1, n]$:

$$S_{\text{query}} = \sum_{i \in Q} s_i + M$$

Pruning ratio:

$$\text{pruning} = 1 - \frac{S_{\text{query}}}{S_{\text{total}}} = 1 - \frac{\sum_{i \in Q} s_i + M}{\sum_{i=1}^{n} s_i + M}$$

For uniform column sizes and $M \ll S_{\text{total}}$:

$$\text{pruning} \approx 1 - \frac{|Q|}{n} = \frac{n - |Q|}{n}$$

### Worked Examples

**50-column table, query reads 3 columns, uniform sizes:**

$$\text{pruning} = 1 - \frac{3}{50} = 94\%$$

Only 6% of the file is read. On a 10 GB file, that is 600 MB vs 10 GB.

**Non-uniform: 3 queried columns are 500 MB out of 10 GB total:**

$$\text{pruning} = 1 - \frac{500}{10{,}000} = 95\%$$

Combined with predicate pushdown (5x from sorting), effective I/O:

$$\text{I/O} = \frac{500}{5} = 100 \text{ MB out of 10 GB} = 1\%$$

---

## 4. Dremel Encoding (Nested Type Flattening)

### The Problem

Parquet must encode arbitrarily nested data (repeated and optional fields) into flat columnar arrays. The Dremel paper introduced repetition and definition levels to achieve this losslessly.

### The Formula

For a field at schema path $p$ with maximum repetition level $r_{\max}$ and maximum definition level $d_{\max}$:

$$r_{\max}(p) = \text{number of } \texttt{repeated} \text{ ancestors (including self)}$$
$$d_{\max}(p) = \text{number of optional or repeated ancestors (including self)}$$

A value at path $p$ is encoded as the triple $(r, d, v)$:

- $r$ = repetition level: which repeated field started a new entry (0 = new record)
- $d$ = definition level: how many levels are actually defined (non-null)
- $v$ = value (only present when $d = d_{\max}$)

Space per value:

$$B_{\text{Dremel}} = \lceil \log_2(r_{\max} + 1) \rceil + \lceil \log_2(d_{\max} + 1) \rceil + B(v) \cdot [d = d_{\max}]$$

### Worked Examples

**Schema: `repeated group A { optional string B }`**

$r_{\max}(B) = 1$ (one repeated ancestor), $d_{\max}(B) = 2$ (A is repeated=1, B is optional=1).

| Record | A.B values | r levels | d levels |
|--------|-----------|----------|----------|
| rec 1  | ["x", null, "y"] | [0, 1, 1] | [2, 1, 2] |
| rec 2  | []        | [0]      | [0]      |
| rec 3  | ["z"]     | [0]      | [2]      |

Flat column: values=["x", "y", "z"], r=[0,1,1,0,0], d=[2,1,2,0,2]

Overhead for levels (1M records, avg 3 nested values each):
$$B_{\text{levels}} = 3{,}000{,}000 \times (1 + 2) / 8 = 1{,}125{,}000 \text{ bytes} \approx 1.1 \text{ MB}$$

With RLE on levels (many repeated patterns): typically compresses to 10-50 KB.

---

## 5. Bloom Filter Analysis (False Positive Trade-offs)

### The Problem

Parquet Bloom filters enable equality predicate pushdown on high-cardinality columns. What is the optimal filter size for a target false positive rate?

### The Formula

For a Bloom filter with $m$ bits, $k$ hash functions, and $n$ inserted elements, the false positive probability:

$$p_{\text{fp}} = \left(1 - e^{-kn/m}\right)^k$$

Optimal number of hash functions:

$$k_{\text{opt}} = \frac{m}{n} \ln 2$$

Required bits per element for target false positive rate $p$:

$$\frac{m}{n} = -\frac{\ln p}{(\ln 2)^2} \approx -1.44 \log_2 p$$

### Worked Examples

**1M unique user_ids, target 1% false positive rate:**

$$\frac{m}{n} = -\frac{\ln 0.01}{(\ln 2)^2} = \frac{4.605}{0.480} = 9.59 \text{ bits/element}$$

$$m = 9.59 \times 1{,}000{,}000 = 9{,}590{,}000 \text{ bits} \approx 1.2 \text{ MB}$$

$$k_{\text{opt}} = 9.59 \times \ln 2 = 6.64 \approx 7 \text{ hash functions}$$

For 0.1% FP rate: $m/n = 14.4$ bits, total $\approx 1.8$ MB, $k = 10$.

**Cost-benefit:** A 1.2 MB Bloom filter avoids reading a 500 MB column chunk when a single user_id is queried. Break-even if more than $\frac{1.2}{500} = 0.24\%$ of queries use equality predicates on user_id.

---

## Prerequisites

- information-theory, compression, bloom-filters, encoding-theory, avro, protobuf
