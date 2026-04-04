# wget Deep Dive — Theory & Internals

> *wget is a non-interactive HTTP/FTP downloader optimized for reliability — it handles retries, recursion, and resume natively. The math covers recursive download modeling (web crawling as graph traversal), bandwidth throttling, mirror sizing estimation, and retry timing.*

---

## 1. Recursive Download — Web Crawling as Graph Theory

### The Model

A website is a directed graph:
- Nodes = pages/resources
- Edges = hyperlinks

`wget -r` performs a breadth-first traversal of this graph.

### Pages Downloaded

With depth limit $D$ and average $L$ links per page:

$$N_{pages} \leq \sum_{d=0}^{D} L^d = \frac{L^{D+1} - 1}{L - 1}$$

### Growth by Depth

| Depth | Links/Page = 10 | Links/Page = 50 | Links/Page = 100 |
|:---:|:---:|:---:|:---:|
| 1 | 11 | 51 | 101 |
| 2 | 111 | 2,551 | 10,101 |
| 3 | 1,111 | 127,551 | 1,010,101 |
| 4 | 11,111 | 6,377,551 | 101,010,101 |
| 5 | 111,111 | ~3.2 billion | ~10 billion |

**This is exponential growth.** Depth 5 with 50 links/page = 3.2 billion pages. This is why `wget -r` needs `-l <depth>` limits.

### Practical Crawl Estimation

Real websites have significant link overlap (many pages link to the same resources):

$$N_{unique} \approx N_{total} \times (1 - R_{overlap})$$

Typical overlap: 60-80%. So $N_{unique} \approx 0.2 \times N_{total}$.

---

## 2. Download Speed and Bandwidth Limiting

### `--limit-rate` Implementation

wget uses a sleep-between-reads approach:

$$T_{sleep} = \frac{S_{chunk}}{R_{limit}} - T_{read}$$

### Effective Transfer Time

$$T_{transfer} = \frac{S_{file}}{R_{limit}} \quad \text{(when rate-limited)}$$

$$T_{transfer} = \frac{S_{file}}{R_{link}} \quad \text{(when link-limited)}$$

### Wait Between Pages (`-w`)

For recursive downloads, `--wait=N` adds delay between page requests:

$$T_{total} = N_{pages} \times (T_{download} + T_{wait})$$

| Pages | Avg Download | Wait | Total Time |
|:---:|:---:|:---:|:---:|
| 100 | 0.5 sec | 0 sec | 50 sec |
| 100 | 0.5 sec | 1 sec | 150 sec |
| 100 | 0.5 sec | 5 sec | 550 sec |
| 1,000 | 0.2 sec | 2 sec | 2,200 sec |

The `--random-wait` option multiplies the wait by a random factor in $[0.5, 1.5]$:

$$T_{wait\_actual} = T_{wait} \times U(0.5, 1.5)$$

---

## 3. Resume and Retry — Reliability Math

### `-c` (Continue/Resume)

wget uses HTTP Range headers to resume:

$$\text{Range: bytes=}B_{completed}\text{-}$$

### Retry Timing

With `--tries=N` and `--waitretry=T`:

$$T_{wait}(n) = \min(n, T_{max}) \text{ seconds}$$

This is **linear** backoff (not exponential like curl):

| Retry | Wait | Cumulative |
|:---:|:---:|:---:|
| 1 | 1 sec | 1 sec |
| 2 | 2 sec | 3 sec |
| 3 | 3 sec | 6 sec |
| 5 | 5 sec | 15 sec |
| 10 | 10 sec | 55 sec |

### Download Success Probability

With per-attempt success probability $p$ and $N$ retries:

$$P_{success} = 1 - (1-p)^N$$

| Per-Attempt Success | 3 Retries | 5 Retries | 10 Retries | 20 Retries |
|:---:|:---:|:---:|:---:|:---:|
| 50% | 87.5% | 96.9% | 99.9% | 100.0% |
| 80% | 99.2% | 99.97% | ~100% | ~100% |
| 95% | 99.99% | ~100% | ~100% | ~100% |

wget defaults to 20 retries — making it extremely persistent.

---

## 4. Mirror Sizing (`--mirror`)

### Estimating Mirror Size

$$S_{mirror} = N_{pages} \times S_{avg\_page} + N_{assets} \times S_{avg\_asset}$$

### Typical Web Page Composition

| Component | Count per Page | Avg Size | Total |
|:---|:---:|:---:|:---:|
| HTML | 1 | 50 KB | 50 KB |
| CSS | 3 | 20 KB | 60 KB |
| JavaScript | 5 | 50 KB | 250 KB |
| Images | 10 | 100 KB | 1,000 KB |
| Fonts | 2 | 50 KB | 100 KB |
| **Total** | **21** | | **1.46 MB** |

### Mirror Size Estimation

| Site Size (pages) | Assets/Page | Avg Total/Page | Mirror Size |
|:---:|:---:|:---:|:---:|
| 50 | 20 | 1.5 MB | 75 MB |
| 500 | 20 | 1.5 MB | 750 MB |
| 5,000 | 25 | 2 MB | 10 GB |
| 50,000 | 25 | 2 MB | 100 GB |

### Incremental Mirror (`-N` / timestamping)

Second run downloads only changed files:

$$S_{incremental} = S_{mirror} \times R_{change}$$

If 5% of pages change daily: incremental download = 5% of full mirror.

---

## 5. Connection Reuse and Performance

### HTTP/1.1 Keep-Alive

wget reuses connections for same-host resources:

$$T_{with\_keepalive} = T_{connect} + N \times T_{request}$$

$$T_{without\_keepalive} = N \times (T_{connect} + T_{request})$$

### Savings

| Requests | Connect Time | With Keep-Alive | Without | Savings |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 50 ms | 550 ms | 1,000 ms | 45% |
| 50 | 50 ms | 2,550 ms | 5,000 ms | 49% |
| 100 | 100 ms | 10,100 ms | 20,000 ms | 50% |

---

## 6. Robots.txt Compliance

### Crawl Budget

`Crawl-delay: N` in robots.txt:

$$R_{max} = \frac{1}{N} \text{ requests/sec}$$

| Crawl-delay | Max Rate | 1,000 pages |
|:---:|:---:|:---:|
| 1 | 1/sec | 1,000 sec (17 min) |
| 5 | 0.2/sec | 5,000 sec (83 min) |
| 10 | 0.1/sec | 10,000 sec (2.8 hours) |
| 60 | 0.017/sec | 60,000 sec (16.7 hours) |

### Disallow Pattern Matching

For $P$ paths and $D$ Disallow rules:

$$\text{Allowed paths} = P - \sum_{d \in D} |M_d|$$

Where $|M_d|$ = number of paths matching Disallow rule $d$.

---

## 7. Quota Management (`-Q`)

### Disk Quota

$$\text{Stop when: } \sum_{i=1}^{N} S_i \geq Q$$

Where $Q$ = quota (e.g., `--quota=500m`).

### Quota Accuracy

wget checks quota between file downloads, not mid-file:

$$S_{actual} \leq Q + S_{max\_file}$$

For a 500 MB quota with max file size of 50 MB: actual usage could reach 550 MB.

---

## 8. Summary of Formulas

| Formula | Math Type | Application |
|:---|:---|:---|
| $(L^{D+1} - 1)/(L-1)$ | Geometric series | Recursive page count |
| $N \times (T_{dl} + T_{wait})$ | Product | Throttled crawl time |
| $1 - (1-p)^N$ | Complement probability | Retry success rate |
| $N_{pages} \times S_{avg}$ | Product | Mirror size estimate |
| $T_{connect} + N \times T_{request}$ | Summation | Keep-alive performance |
| $1/\text{Crawl-delay}$ | Inverse | Max request rate |

## Prerequisites

- URL encoding, recursive graph traversal, exponential backoff

---

*wget is the tool you use when reliability matters more than speed — its retry logic, resume capability, and recursive mirroring make it the workhorse of automated downloads. The exponential growth of recursive crawling is why every wget tutorial warns you about `-r` without `-l`.*
