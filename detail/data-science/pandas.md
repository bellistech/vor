# The Mathematics of pandas — Alignment, Aggregation, and Relational Algebra

> *pandas implements a computational model rooted in relational algebra, with automatic label-based alignment that distinguishes it from raw NumPy arrays. Understanding the mathematical foundations of its groupby-aggregate pattern, join algorithms, and resampling operations reveals why certain operations are efficient and how to reason about correctness when combining heterogeneous datasets.*

---

## 1. Relational Algebra of Joins (Set Theory)
### The Problem
pandas merge operations implement the same join semantics as relational databases. Understanding the set-theoretic foundations helps predict output cardinality and avoid unexpected row multiplication.

### The Formula
For two relations $R$ and $S$ with join key $k$:

**Inner join**: $R \bowtie S = \{(r, s) : r \in R, s \in S, r.k = s.k\}$

Output cardinality for a key with multiplicities $m_R(v)$ and $m_S(v)$:

$$|R \bowtie S| = \sum_{v \in K} m_R(v) \cdot m_S(v)$$

**Left join**: preserves all rows of $R$:

$$|R \bowtie_L S| = \sum_{v \in K_R} m_R(v) \cdot \max(m_S(v), 1)$$

**Cross join** (Cartesian product): $|R \times S| = |R| \cdot |S|$

### Worked Examples
**Example**: Orders table (1000 rows), Customers table (500 rows). Each customer has 0-5 orders. 50 customers have no orders.

Inner join on customer_id:
- 450 customers with orders, average 2.2 orders each
$$|R \bowtie S| = \sum_{v} m_R(v) \cdot 1 = 1000 \text{ (each order matches exactly 1 customer)}$$

Left join (keep all orders): Same as inner join (every order has a customer).

Right join (keep all customers): $1000 + 50 = 1050$ rows (50 customers with NaN order fields).

**Many-to-many danger**: If both tables have duplicates on the key:
- Table A: key=1 appears 3 times
- Table B: key=1 appears 4 times
- Join produces $3 \times 4 = 12$ rows for key=1 alone

## 2. GroupBy-Aggregate (Monoid Homomorphisms)
### The Problem
The groupby-aggregate pattern partitions data into groups and applies a reduction function. Mathematically, efficient aggregations are monoid homomorphisms that allow parallel and incremental computation.

### The Formula
A monoid $(M, \oplus, e)$ consists of a set $M$, an associative binary operation $\oplus$, and an identity element $e$.

An aggregation function $f: \text{List}[A] \to M$ is a monoid homomorphism if:

$$f(xs \mathbin\Vert ys) = f(xs) \oplus f(ys)$$

Common aggregations as monoids:

$$\text{sum}: (\mathbb{R}, +, 0) \quad \text{count}: (\mathbb{N}, +, 0)$$
$$\text{min}: (\mathbb{R} \cup \{\infty\}, \min, \infty) \quad \text{max}: (\mathbb{R} \cup \{-\infty\}, \max, -\infty)$$

Mean is not a monoid but decomposes into two:

$$\text{mean}(x) = \frac{\text{sum}(x)}{\text{count}(x)}$$

Variance via Welford's online algorithm:

$$M_{2,n} = M_{2,n-1} + (x_n - \bar{x}_{n-1})(x_n - \bar{x}_n)$$

$$\sigma^2_n = \frac{M_{2,n}}{n}$$

### Worked Examples
**Example**: Computing mean salary by department incrementally.

Department "Eng" arrives in two batches:
- Batch 1: [70K, 85K, 120K] -> sum=275K, count=3
- Batch 2: [95K, 110K] -> sum=205K, count=2

$$\text{mean} = \frac{275K + 205K}{3 + 2} = \frac{480K}{5} = 96K$$

This decomposition is why `groupby().agg(['sum', 'count'])` followed by manual division is equivalent to `groupby().mean()` but allows distributed computation.

## 3. Resampling and the Nyquist Criterion (Signal Processing)
### The Problem
When resampling time series data (e.g., daily to weekly), information is lost. The Nyquist-Shannon theorem defines the minimum sampling rate to faithfully represent the underlying signal.

### The Formula
A continuous signal $x(t)$ with maximum frequency $f_{max}$ can be perfectly reconstructed from samples taken at rate:

$$f_s \geq 2 f_{max}$$

When downsampling from rate $f_1$ to $f_2 < f_1$, the aggregation function matters. For a signal sampled at points $t_1, \ldots, t_n$ within a resample window:

$$\bar{x}_{window} = \frac{1}{n} \sum_{i=1}^{n} x(t_i) \quad \text{(mean aggregation)}$$

$$\hat{x}_{window} = x(t_n) \quad \text{(last value, for OHLC)}$$

Aliasing occurs when downsampling below Nyquist, creating spurious low-frequency patterns:

$$f_{alias} = |f_{signal} - k \cdot f_s|, \quad k = \text{nearest integer}$$

### Worked Examples
**Example**: Daily stock data resampled to monthly.

Daily frequency: $f_1 = 1/\text{day} \approx 252/\text{year}$ (trading days).

Monthly frequency: $f_2 = 12/\text{year}$.

Maximum representable frequency after resampling: $f_{max} = 6/\text{year}$ (bi-monthly cycles).

Any pattern with periodicity shorter than 2 months will be aliased or lost. A weekly seasonal pattern ($f = 52/\text{year}$) cannot be recovered from monthly data.

## 4. Label-Based Alignment (Abstract Algebra)
### The Problem
pandas automatically aligns data by index labels during arithmetic operations. This alignment is a key differentiator from NumPy and implements a mathematical operation on partial functions.

### The Formula
A pandas Series is a partial function $f: I \to V$ from index labels to values. Binary operations use outer alignment:

$$(f \oplus g)(i) = \begin{cases} f(i) \oplus g(i) & \text{if } i \in \text{dom}(f) \cap \text{dom}(g) \\ \text{NaN} & \text{if } i \in \text{dom}(f) \triangle \text{dom}(g) \end{cases}$$

Where $\triangle$ denotes the symmetric difference.

With `fill_value=0`:

$$(f \oplus g)(i) = \begin{cases} f(i) \oplus g(i) & \text{if } i \in \text{dom}(f) \cap \text{dom}(g) \\ f(i) \oplus 0 & \text{if } i \in \text{dom}(f) \setminus \text{dom}(g) \\ 0 \oplus g(i) & \text{if } i \in \text{dom}(g) \setminus \text{dom}(f) \end{cases}$$

### Worked Examples
**Example**: Two Series with partially overlapping indices:

$f = \{a: 1, b: 2, c: 3\}$, $g = \{b: 10, c: 20, d: 30\}$

$f + g = \{a: \text{NaN}, b: 12, c: 23, d: \text{NaN}\}$

$f.\text{add}(g, \text{fill\_value}=0) = \{a: 1, b: 12, c: 23, d: 30\}$

Domain of result: $\text{dom}(f) \cup \text{dom}(g) = \{a, b, c, d\}$

## 5. Rolling Window Statistics (Convolution)
### The Problem
Rolling calculations (moving average, rolling standard deviation) are discrete convolutions that apply a kernel function across a sliding window of observations.

### The Formula
A rolling mean with window size $w$ is a convolution with a uniform kernel:

$$\bar{x}_t = \frac{1}{w} \sum_{i=0}^{w-1} x_{t-i} = (x * k)_t$$

Where $k = [\frac{1}{w}, \frac{1}{w}, \ldots, \frac{1}{w}]$.

Exponentially weighted moving average:

$$EWMA_t = \alpha \cdot x_t + (1 - \alpha) \cdot EWMA_{t-1}$$

The effective window size for EWMA:

$$w_{eff} = \frac{2}{\alpha} - 1$$

Rolling variance (Welford's method for numerical stability):

$$\sigma^2_t = \frac{1}{w-1} \left[ \sum_{i=0}^{w-1} x_{t-i}^2 - w \cdot \bar{x}_t^2 \right]$$

### Worked Examples
**Example**: 7-day rolling average of daily revenue: [100, 120, 90, 110, 130, 95, 105].

$$\bar{x}_7 = \frac{100 + 120 + 90 + 110 + 130 + 95 + 105}{7} = \frac{750}{7} \approx 107.1$$

EWMA with $\alpha = 0.3$ (span=5.67):
$$EWMA_1 = 100$$
$$EWMA_2 = 0.3(120) + 0.7(100) = 106$$
$$EWMA_3 = 0.3(90) + 0.7(106) = 101.2$$
$$EWMA_4 = 0.3(110) + 0.7(101.2) = 103.84$$

The EWMA responds more quickly to recent changes than the simple rolling mean.

## 6. Memory Optimization via Categoricals (Information Theory)
### The Problem
String columns with low cardinality waste memory by storing full string data for each row. The `category` dtype uses a dictionary encoding, storing each unique string once and referencing by integer code.

### The Formula
Memory for string representation:

$$M_{string} = N \times \bar{L} \times b$$

Where $N$ is row count, $\bar{L}$ is average string length, $b$ is bytes per character.

Memory for categorical representation:

$$M_{category} = K \times \bar{L} \times b + N \times \lceil \log_2 K \rceil / 8$$

Where $K$ is the number of unique categories.

Compression ratio:

$$\rho = \frac{M_{category}}{M_{string}} = \frac{K \times \bar{L} \times b + N \times \lceil \log_2 K \rceil / 8}{N \times \bar{L} \times b}$$

### Worked Examples
**Example**: Column "country" with 10M rows, 200 unique countries, average name length 8 chars (UTF-8).

$$M_{string} = 10^7 \times 8 = 80 \text{ MB}$$

$$M_{category} = 200 \times 8 + 10^7 \times 1 = 1{,}600 + 10{,}000{,}000 \approx 10 \text{ MB}$$

$$\rho = \frac{10}{80} = 0.125 \quad \text{(87.5\% reduction)}$$

## Prerequisites
- relational-algebra, set-theory, abstract-algebra, signal-processing, convolution, information-theory, monoids
