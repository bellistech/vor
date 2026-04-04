# The Mathematics of Jupyter — Computational Reproducibility and Kernel Architecture

> *Jupyter's architecture embodies fundamental concepts from distributed systems and computational theory: the notebook execution model raises questions of determinism and reproducibility formalized through directed acyclic graphs, the kernel protocol implements a message-passing concurrency model, and the rendering pipeline applies document transformation theory to produce multiple output formats from a single source.*

---

## 1. Execution Order and Reproducibility (Graph Theory)
### The Problem
Jupyter notebooks allow out-of-order cell execution, creating hidden state dependencies. Modeling cell dependencies as a directed graph reveals whether a notebook is reproducible (topologically sortable) or contains cycles from mutation.

### The Formula
A notebook with $n$ cells defines a dependency graph $G = (V, E)$ where:
- $V = \{c_1, c_2, \ldots, c_n\}$ (cells)
- $(c_i, c_j) \in E$ if $c_j$ reads a variable that $c_i$ writes

The notebook is reproducible (top-to-bottom execution yields same results) if and only if:

$$\forall (c_i, c_j) \in E: i < j$$

The number of possible execution orderings is bounded by the number of topological sorts:

$$|\text{TopSort}(G)| \leq \frac{n!}{\prod_{v \in V} |\text{desc}(v)|!}$$

A reproducibility score:

$$R = 1 - \frac{|\{(c_i, c_j) \in E : i > j\}|}{|E|}$$

Where $R = 1$ means fully reproducible (all dependencies flow downward).

### Worked Examples
**Example**: A notebook with 5 cells:
- Cell 1: `import pandas as pd` (no deps)
- Cell 2: `df = pd.read_csv('data.csv')` (depends on cell 1)
- Cell 3: `summary = df.describe()` (depends on cell 2)
- Cell 4: `df['new_col'] = df['a'] * 2` (depends on cell 2, mutates df)
- Cell 5: `print(summary)` (depends on cell 3)

If executed 1, 2, 4, 3, 5: cell 3 sees the mutated `df` (with `new_col`), but `summary` was computed before mutation if executed 1, 2, 3, 4, 5.

Dependency edges: (1,2), (2,3), (2,4), (3,5), (4,3) [implicit via mutation].

Back-edge (4,3) makes $R = 1 - 1/5 = 0.8$. The notebook has a hidden dependency.

## 2. Kernel Message Protocol (Concurrency Theory)
### The Problem
The Jupyter kernel protocol uses ZeroMQ sockets for asynchronous message passing between the frontend and kernel. Understanding the message flow helps debug issues like hanging kernels and race conditions.

### The Formula
The protocol uses five channels modeled as typed message queues:

$$\text{Channels} = \{\text{shell}, \text{iopub}, \text{stdin}, \text{control}, \text{heartbeat}\}$$

Message throughput on channel $c$ with queue depth $d_c$:

$$T_c = \frac{d_c}{\bar{t}_{process} + \bar{t}_{serialize} + \bar{t}_{network}}$$

For $n$ concurrent users on JupyterHub, total message rate:

$$M_{total} = \sum_{i=1}^{n} \lambda_i$$

Where $\lambda_i$ is the message rate for user $i$. Using a Poisson model for cell executions:

$$P(\text{queue overflow}) = 1 - \sum_{k=0}^{Q} \frac{(\lambda \bar{t})^k e^{-\lambda \bar{t}}}{k!}$$

Where $Q$ is the queue capacity.

### Worked Examples
**Example**: JupyterHub with 50 concurrent users, each executing cells at rate $\lambda = 0.1$/second (one cell every 10 seconds). Average kernel processing time: 2 seconds.

Total message rate: $M = 50 \times 0.1 = 5$ messages/second.

Server utilization (single kernel per user, so no contention):
Each user: $\rho = \lambda \times \bar{t} = 0.1 \times 2 = 0.2$ (20% busy).

But for shared resources (disk I/O, memory):
Total demand: $5 \times 2 = 10$ kernel-seconds per second across the hub.

With 50 kernel slots: $\rho_{hub} = 10/50 = 0.2$ (20% hub utilization).

## 3. Notebook Document Transformation (Category Theory)
### The Problem
nbconvert transforms notebooks into HTML, PDF, slides, and scripts. This pipeline can be modeled as a series of functorial transformations between document categories.

### The Formula
The conversion pipeline as a composition of morphisms:

$$\text{output} = F_n \circ F_{n-1} \circ \cdots \circ F_1(\text{notebook})$$

Where each $F_i$ is a transformation:
- $F_1$: Preprocessor (execute cells, clear output, tag filtering)
- $F_2$: Exporter (cell to target format conversion)
- $F_3$: Postprocessor (cleanup, embedding, optimization)

Information loss function for format conversion:

$$L(\text{source} \to \text{target}) = H(\text{source}) - H(\text{target})$$

Where $H$ is the information entropy of the document.

For idempotent operations (running twice gives same result):

$$F \circ F = F$$

### Worked Examples
**Example**: Converting a notebook with 20 code cells, 10 markdown cells, 15 output cells (including 5 images).

To HTML: $L = 0$ (lossless, all content preserved)
To PDF via LaTeX: $L > 0$ (interactive widgets become static, JavaScript lost)
To Python script: $L = H_{markdown} + H_{outputs}$ (all non-code content stripped)
To slides: $L = H_{non-tagged}$ (only cells tagged for slides survive)

Information content estimate:
- Code: 20 cells, ~50 lines average = 1000 lines
- Markdown: 10 cells, ~20 lines = 200 lines
- Outputs: 15 cells including images

Script conversion preserves ~1000/1200 lines of text content (83%) but loses all visual output.

## 4. Resource Allocation for JupyterHub (Queueing Theory)
### The Problem
JupyterHub must allocate compute resources (CPU, memory, GPU) across concurrent users. Queueing theory models help size the infrastructure.

### The Formula
Modeling user requests as an M/M/c queue (Poisson arrivals, exponential service, $c$ servers):

Utilization: $\rho = \frac{\lambda}{c \mu}$

Erlang C formula (probability of waiting):

$$P_W = \frac{\frac{(c\rho)^c}{c!(1-\rho)}}{\sum_{k=0}^{c-1} \frac{(c\rho)^k}{k!} + \frac{(c\rho)^c}{c!(1-\rho)}}$$

Average wait time:

$$W_q = \frac{P_W}{c\mu(1-\rho)}$$

For memory allocation, the probability that total memory demand exceeds capacity $M$:

$$P\left(\sum_{i=1}^{n} X_i > M\right) \approx 1 - \Phi\left(\frac{M - n\mu_X}{\sigma_X \sqrt{n}}\right)$$

By the Central Limit Theorem, where $X_i$ is memory demand per user.

### Worked Examples
**Example**: JupyterHub for a data science class of 100 students.

Parameters:
- Arrival rate: $\lambda = 2$ server requests/minute during peak
- Average session duration: $1/\mu = 30$ minutes
- Each server pod: 4GB RAM, 2 CPU cores
- Available capacity: $c = 40$ concurrent pods

$$\rho = \frac{\lambda}{c \mu} = \frac{2}{40 \times (1/30)} = \frac{2}{1.333} = 1.5$$

Since $\rho > 1$, the queue is unstable during peak. Need $c > \lambda/\mu = 2 \times 30 = 60$ pods.

With $c = 70$ pods:
$$\rho = \frac{60}{70} = 0.857$$

Memory sizing: if average user needs 2GB ($\sigma = 1$GB), for 70 concurrent users:

$$P\left(\sum > 180\text{GB}\right) = 1 - \Phi\left(\frac{180 - 140}{1 \times \sqrt{70}}\right) = 1 - \Phi(4.78) \approx 0$$

Total 180GB is safe. But 150GB:

$$P\left(\sum > 150\text{GB}\right) = 1 - \Phi\left(\frac{150 - 140}{8.37}\right) = 1 - \Phi(1.19) = 0.117$$

11.7% chance of memory pressure — risky.

## 5. Computational Reproducibility Metrics (Information Theory)
### The Problem
Quantifying how reproducible a notebook is requires metrics that capture environment specification, execution determinism, and output stability.

### The Formula
Reproducibility entropy of a notebook:

$$H_{repro} = -\sum_{i=1}^{k} p_i \log_2 p_i$$

Where $p_i$ is the probability of source $i$ of non-determinism:
- Random seeds not set
- External data dependencies
- Time-dependent operations
- Environment-specific paths
- Network calls without caching

Execution determinism score:

$$D = \frac{|\{c : f(c) = f'(c)\}|}{n}$$

Where $f(c)$ and $f'(c)$ are outputs of cell $c$ across two independent runs.

Environment specification completeness:

$$S_{env} = \frac{|\text{pinned deps}|}{|\text{total deps}|}$$

### Worked Examples
**Example**: A notebook with 30 cells:
- 25 cells produce identical output across runs: $D = 25/30 = 0.833$
- 3 cells use `datetime.now()`: non-deterministic
- 2 cells use unseeded random: non-deterministic

Environment: 15 packages used, 12 pinned in requirements.txt: $S_{env} = 12/15 = 0.8$

Composite reproducibility score:

$$R_{composite} = w_D \cdot D + w_S \cdot S_{env} = 0.6 \times 0.833 + 0.4 \times 0.8 = 0.5 + 0.32 = 0.82$$

Improving to $R = 0.95$ requires: setting random seeds (+2 cells), replacing `datetime.now()` with fixed timestamps for testing (+3 cells), and pinning remaining 3 packages.

## 6. Widget Interaction Latency (Control Theory)
### The Problem
Interactive widgets create a feedback loop between user input and visual output. The perceived responsiveness depends on the round-trip latency through the kernel.

### The Formula
Total interaction latency:

$$\tau_{total} = \tau_{frontend} + \tau_{serialize} + \tau_{network} + \tau_{kernel} + \tau_{render}$$

For acceptable interactivity, following Nielsen's response time thresholds:

$$\tau_{total} < 100\text{ms} \implies \text{instantaneous feel}$$
$$\tau_{total} < 1\text{s} \implies \text{noticeable but acceptable}$$
$$\tau_{total} > 10\text{s} \implies \text{user loses focus}$$

For debounced slider widgets with update rate $f$:

$$f_{effective} = \min\left(f_{input}, \frac{1}{\tau_{total}}\right)$$

### Worked Examples
**Example**: A slider controlling a matplotlib plot with 10,000 data points.

- Frontend event: ~5ms
- Serialization: ~2ms
- Network (local): ~1ms
- Kernel computation: ~50ms (plot generation)
- Render: ~30ms

$$\tau_{total} = 5 + 2 + 1 + 50 + 30 = 88\text{ms}$$

Maximum smooth update rate: $1/0.088 \approx 11$ fps. Acceptable for interactive exploration.

For remote JupyterHub ($\tau_{network} = 100$ms):

$$\tau_{total} = 5 + 2 + 100 + 50 + 30 = 187\text{ms}$$

Maximum: $\approx 5$ fps. Noticeably laggy. Mitigation: debounce slider to 200ms intervals, precompute results.

## Prerequisites
- graph-theory, concurrency, queueing-theory, information-theory, control-theory, category-theory
