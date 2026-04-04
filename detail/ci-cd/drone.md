# The Mathematics of Drone — Pipeline Scheduling and Matrix Combinatorics

> *Drone pipelines are sequential step chains with conditional branching, matrix builds expand into a combinatorial product space, and runner capacity planning is a bin-packing problem. The underlying mathematics spans queueing theory, combinatorics, and graph scheduling.*

---

## 1. Pipeline Execution as Sequential Graphs (Graph Theory)

### The Problem

Each Drone pipeline executes steps sequentially within a stage. The total execution time is the sum of all step durations, unlike DAG-based systems where parallelism reduces critical path length.

### The Formula

For a pipeline with $n$ steps, each with duration $d_i$:

$$T_{\text{pipeline}} = \sum_{i=1}^{n} d_i + \sum_{i=1}^{n} \delta_i$$

where $\delta_i$ is the container startup overhead for step $i$.

With $k$ parallel pipelines (multi-pipeline YAML):

$$T_{\text{total}} = \max_{j=1}^{k} T_{\text{pipeline}_j}$$

### Worked Examples

Pipeline: clone (5s) -> test (120s) -> build (60s) -> push (30s) -> deploy (15s).

Container overhead $\delta = 3s$ per step:

$$T = (5 + 120 + 60 + 30 + 15) + 5 \times 3 = 230 + 15 = 245\text{s}$$

---

## 2. Matrix Build Combinatorics (Combinatorics)

### The Problem

Matrix builds create one pipeline instance per combination of matrix variables. The total number of builds is the Cartesian product of all variable value sets.

### The Formula

For $m$ matrix variables with $|V_1|, |V_2|, \ldots, |V_m|$ values each:

$$N_{\text{builds}} = \prod_{i=1}^{m} |V_i|$$

With `include` entries adding $a$ explicit combinations and `exclude` removing $e$:

$$N_{\text{effective}} = \prod_{i=1}^{m} |V_i| + a - e$$

Total CI time with $r$ runner slots:

$$T_{\text{matrix}} = \left\lceil \frac{N_{\text{effective}}}{r} \right\rceil \cdot T_{\text{pipeline}}$$

### Worked Examples

Matrix: GO_VERSION=[1.21, 1.22], GOOS=[linux, darwin], GOARCH=[amd64, arm64]:

$$N = 2 \times 2 \times 2 = 8 \text{ pipeline instances}$$

Excluding darwin/arm64 ($e = 2$, one per Go version):

$$N_{\text{effective}} = 8 - 2 = 6$$

With 4 runner slots and $T_{\text{pipeline}} = 245s$:

$$T_{\text{matrix}} = \lceil 6/4 \rceil \times 245 = 2 \times 245 = 490\text{s}$$

---

## 3. Runner Capacity Planning (Queueing Theory)

### The Problem

Drone runners process build requests from a queue. Each runner has a configurable capacity (concurrent pipelines). This is modeled as an M/M/c queue.

### The Formula

For arrival rate $\lambda$ (builds/hour), service rate $\mu$ (builds/hour per slot), and $c$ total runner slots:

Traffic intensity:

$$\rho = \frac{\lambda}{c \cdot \mu}$$

Erlang C probability (all servers busy):

$$C(c, \lambda/\mu) = \frac{\frac{(\lambda/\mu)^c}{c!} \cdot \frac{1}{1 - \rho}}{\sum_{k=0}^{c-1} \frac{(\lambda/\mu)^k}{k!} + \frac{(\lambda/\mu)^c}{c!} \cdot \frac{1}{1 - \rho}}$$

Average wait time in queue:

$$W_q = \frac{C(c, \lambda/\mu)}{c \cdot \mu - \lambda}$$

### Worked Examples

$\lambda = 30$ builds/hour, $\mu = 10$ builds/hour per slot, $c = 4$ slots:

$$\rho = \frac{30}{4 \times 10} = 0.75$$

Average queue wait:

$$W_q = \frac{C(4, 3)}{4 \times 10 - 30} = \frac{C(4, 3)}{10}$$

Computing $C(4, 3) \approx 0.509$:

$$W_q \approx \frac{0.509}{10} = 0.051 \text{ hours} \approx 3.1 \text{ minutes}$$

---

## 4. Secret Entropy and Security (Information Theory)

### The Problem

Drone secrets must have sufficient entropy to resist brute-force attacks. The security of the system depends on the minimum entropy among all secrets.

### The Formula

For a secret of length $L$ drawn from alphabet of size $|\Sigma|$:

$$H = L \cdot \log_2 |\Sigma|$$

Time to brute-force at rate $r$ attempts/second:

$$T_{\text{brute}} = \frac{2^H}{2 \cdot r}$$

For the RPC shared secret between server and runners:

$$H_{\min} \geq 128 \text{ bits (recommended)}$$

### Worked Examples

Alphanumeric secret ($|\Sigma| = 62$), length $L = 32$:

$$H = 32 \times \log_2(62) = 32 \times 5.95 = 190.4 \text{ bits}$$

At $r = 10^9$ attempts/second:

$$T_{\text{brute}} = \frac{2^{190.4}}{2 \times 10^9} \approx 10^{48} \text{ seconds}$$

---

## 5. Conditional Step Evaluation (Boolean Logic)

### The Problem

Drone step conditions (`when` clauses) combine branch, event, status, and target predicates using implicit conjunction. The evaluation determines which steps execute.

### The Formula

Step $s$ executes when:

$$\text{exec}(s) = \bigwedge_{p \in \text{when}(s)} p(\text{context})$$

where each predicate $p$ is:

$$p_{\text{branch}}(ctx) = ctx.\text{branch} \in \text{branches}(s)$$

$$p_{\text{event}}(ctx) = ctx.\text{event} \in \text{events}(s)$$

$$p_{\text{status}}(ctx) = ctx.\text{status} \in \text{statuses}(s)$$

With `exclude` lists, predicates become:

$$p_{\text{event}}(ctx) = ctx.\text{event} \notin \text{exclude}(s)$$

### Worked Examples

Step with `when: { branch: [main], event: [push, tag], status: [success] }`:

For context (branch=main, event=push, status=success):

$$\text{exec} = (main \in \{main\}) \wedge (push \in \{push, tag\}) \wedge (success \in \{success\}) = \text{true}$$

For context (branch=develop, event=push, status=success):

$$\text{exec} = (develop \in \{main\}) \wedge \ldots = \text{false}$$

---

## 6. Cron Scheduling Density (Number Theory)

### The Problem

Cron expressions define periodic build triggers. Multiple cron jobs with different periods create a pattern of build density over time.

### The Formula

For cron jobs with periods $p_1, p_2, \ldots, p_k$ (in minutes), the combined build rate:

$$\lambda_{\text{cron}} = \sum_{i=1}^{k} \frac{1}{p_i} \text{ builds/minute}$$

The LCM determines when all jobs coincide:

$$T_{\text{coincide}} = \text{lcm}(p_1, p_2, \ldots, p_k)$$

Builds within one LCM period:

$$N_{\text{lcm}} = \sum_{i=1}^{k} \frac{T_{\text{coincide}}}{p_i}$$

### Worked Examples

Three cron jobs: every 60 min, every 360 min, every 1440 min:

$$\lambda = \frac{1}{60} + \frac{1}{360} + \frac{1}{1440} = 0.01667 + 0.00278 + 0.00069 = 0.02014 \text{ builds/min}$$

$$T_{\text{coincide}} = \text{lcm}(60, 360, 1440) = 1440 \text{ min} = 24 \text{ hours}$$

$$N_{\text{lcm}} = 24 + 4 + 1 = 29 \text{ builds per day}$$

---

## Prerequisites

- queueing-theory, combinatorics, boolean-logic, information-theory, graph-theory
