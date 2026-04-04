# The Mathematics of Falco — Syscall Analysis and Behavioral Anomaly Detection

> *Falco processes a continuous stream of syscall events through a rule engine that is formally a complex event processing system. Detection rules are temporal predicates over event sequences, the eBPF probe operates as a kernel-space filter with bounded memory, and alert routing through Falcosidekick is a fan-out graph with priority-based filtering.*

---

## 1. Syscall Event Stream Processing (Stream Algebra)

### The Problem

Falco processes millions of syscalls per second per host. Each syscall is an event in a continuous stream. The rule engine must evaluate predicates over this stream with minimal latency.

### The Formula

Event stream: $E = (e_1, e_2, \ldots)$ where each $e_i = (\text{type}, \text{args}, t_i, \text{pid}, \text{container})$.

Rule $r$ with condition predicate $\phi_r$:

$$\text{alert}(e, r) = \begin{cases} 1 & \text{if } \phi_r(e) = \text{true} \wedge \text{priority}(r) \geq P_{\min} \\ 0 & \text{otherwise} \end{cases}$$

Total alerts per second:

$$A = \sum_{r \in R} \lambda \cdot P(\phi_r(e) = \text{true})$$

where $\lambda$ is the event rate.

Processing budget per event:

$$T_{\text{budget}} = \frac{1}{\lambda}$$

### Worked Examples

Host generating $\lambda = 500{,}000$ syscalls/second, 25 active rules:

$$T_{\text{budget}} = \frac{1}{500{,}000} = 2\mu\text{s per event}$$

Rule evaluation cost: $\sim 0.5\mu\text{s per rule}$

Total per event: $25 \times 0.5 = 12.5\mu\text{s}$

Required parallelism: $\lceil 12.5 / 2.0 \rceil = 7$ evaluation threads.

If 0.01% of events match the "Shell in Container" rule:

$$A_{\text{shell}} = 500{,}000 \times 0.0001 = 50 \text{ alerts/s}$$

---

## 2. eBPF Ring Buffer Sizing (Queueing Theory)

### The Problem

The eBPF probe captures events into a per-CPU ring buffer. If the consumer (userspace Falco) cannot keep up, events are dropped. Buffer sizing is a queueing theory problem.

### The Formula

Ring buffer as M/D/1 queue with arrival rate $\lambda$ and service rate $\mu$:

$$\rho = \frac{\lambda}{\mu}$$

Drop probability when buffer has capacity $B$ events:

$$P(\text{drop}) = \begin{cases} 0 & \text{if } \rho < 1 \\ \frac{\rho^B (1-\rho)}{1-\rho^{B+1}} & \text{if } \rho \approx 1 \end{cases}$$

For burst events, the buffer must absorb the burst:

$$B \geq \lambda_{\text{peak}} \cdot T_{\text{drain}}$$

where $T_{\text{drain}}$ is the time to process the burst backlog.

Memory required:

$$M = B \times S_{\text{event}} \times N_{\text{cpus}}$$

### Worked Examples

$\lambda_{\text{avg}} = 500$K/s, $\lambda_{\text{peak}} = 2$M/s, $\mu = 800$K/s, $S_{\text{event}} = 256$ bytes, $N_{\text{cpus}} = 8$:

Burst duration 100ms: events during burst = $2{,}000{,}000 \times 0.1 = 200{,}000$.

Drain during burst: $800{,}000 \times 0.1 = 80{,}000$.

Buffer needed: $B = 200{,}000 - 80{,}000 = 120{,}000$ events.

Memory: $120{,}000 \times 256 \times 8 = 245.8$ MB.

Default `syscall_buf_size_preset=4` allocates $\sim 8$ MB per CPU = $64$ MB total, handling:

$$B_{\text{default}} = \frac{8 \times 10^6}{256} = 31{,}250 \text{ events/cpu}$$

---

## 3. Rule Condition as Boolean Algebra (Logic)

### The Problem

Falco rule conditions are boolean expressions over event fields. Macros allow composition. The condition is compiled into an evaluation tree.

### The Formula

A condition is a boolean formula in conjunctive/disjunctive form:

$$\phi = \bigwedge_{i} \left(\bigvee_{j} L_{ij}\right)$$

where $L_{ij}$ are literal predicates on event fields.

Macro expansion inlines sub-formulas:

$$\phi[\text{macro}_k / \psi_k]$$

Evaluation complexity:

$$T(\phi) = \sum_{i} T(L_i) \text{ (worst case)}$$

With short-circuit evaluation:

$$T_{\text{avg}}(\phi) = T(L_1) + P(L_1) \cdot T(L_2) + P(L_1 \wedge L_2) \cdot T(L_3) + \cdots$$

### Worked Examples

Rule: `spawned_process and container and is_shell and not proc.pname in (cron, sshd)`

Expanding macros:

$$\phi = (\text{evt.type} \in \{\text{execve}\} \wedge \text{evt.dir} = \text{<}) \wedge (\text{container.id} \neq \text{host}) \wedge (\text{proc.name} \in \{\text{bash, sh, zsh}\}) \wedge (\text{proc.pname} \notin \{\text{cron, sshd}\})$$

Short-circuit evaluation probabilities:

| Predicate | $P(\text{true})$ | Cumulative $P$ | Cost |
|:---|:---:|:---:|:---:|
| evt.type in (execve) | 0.002 | 0.002 | 50ns |
| evt.dir = < | 0.50 | 0.001 | 20ns |
| container.id != host | 0.30 | 0.0003 | 30ns |
| proc.name in (bash...) | 0.05 | 0.000015 | 40ns |

$$T_{\text{avg}} = 50 + 0.002 \times 20 + 0.001 \times 30 + 0.0003 \times 40 = 50.08\text{ns}$$

Short-circuit makes evaluation nearly constant time since the first predicate filters 99.8%.

---

## 4. Alert Fan-Out through Falcosidekick (Graph Theory)

### The Problem

Falcosidekick routes alerts to multiple destinations based on priority thresholds. This is a directed graph from Falco to output sinks with priority-based edge filtering.

### The Formula

Fan-out graph $G = (V, E)$ where $V = \{\text{falco}\} \cup \text{Outputs}$ and edges have priority thresholds.

For alert with priority $p$, the destination set:

$$D(p) = \{o \in \text{Outputs} \mid P_{\min}(o) \leq p\}$$

Fan-out factor:

$$F(p) = |D(p)|$$

Total messages per second across all outputs:

$$M = \sum_{p \in \text{priorities}} A_p \cdot F(p)$$

### Worked Examples

Outputs: Slack ($P_{\min}$ = WARNING), PagerDuty ($P_{\min}$ = CRITICAL), Elasticsearch ($P_{\min}$ = DEBUG), Loki ($P_{\min}$ = INFO).

Priority levels: DEBUG=0, INFO=1, NOTICE=2, WARNING=3, ERROR=4, CRITICAL=5.

| Alert Priority | Fan-out $F(p)$ | Destinations |
|:---|:---:|:---|
| DEBUG | 1 | ES |
| INFO | 2 | ES, Loki |
| WARNING | 3 | ES, Loki, Slack |
| CRITICAL | 4 | ES, Loki, Slack, PD |

Alert rates: 100 DEBUG/s, 20 INFO/s, 5 WARNING/s, 0.1 CRITICAL/s:

$$M = 100(1) + 20(2) + 5(3) + 0.1(4) = 100 + 40 + 15 + 0.4 = 155.4 \text{ messages/s}$$

---

## 5. MITRE ATT&CK Coverage (Set Theory)

### The Problem

Falco rules map to MITRE ATT&CK techniques. Coverage analysis measures what fraction of the technique space is detected by the deployed rule set.

### The Formula

MITRE technique set for containers: $\mathcal{M} = \{T_1, T_2, \ldots, T_N\}$.

Rule-to-technique mapping: $\tau: R \to \mathcal{P}(\mathcal{M})$.

Coverage:

$$C = \frac{\left|\bigcup_{r \in R} \tau(r)\right|}{|\mathcal{M}|}$$

Detection depth (rules per technique):

$$d(T) = |\{r \in R \mid T \in \tau(r)\}|$$

Weighted coverage (accounting for technique prevalence $w_T$):

$$C_w = \frac{\sum_{T \in \bigcup \tau(r)} w_T}{\sum_{T \in \mathcal{M}} w_T}$$

### Worked Examples

Container ATT&CK matrix: 40 techniques. Default Falco rules cover 22 techniques:

$$C = \frac{22}{40} = 55\%$$

With custom rules adding 8 more techniques:

$$C' = \frac{30}{40} = 75\%$$

Detection depth for T1059 (Command and Scripting Interpreter):

$$d(T1059) = 4 \text{ rules (shell, python, curl, wget)}$$

---

## 6. False Positive Rate Estimation (Bayesian Statistics)

### The Problem

Rule tuning requires balancing detection rate (true positives) against false positive rate. Each rule's precision depends on how well the condition separates malicious from benign events.

### The Formula

For rule $r$ with detection rate (sensitivity) $\text{TPR}$ and false positive rate $\text{FPR}$:

$$\text{Precision} = \frac{\text{TPR} \cdot P(\text{attack})}{\text{TPR} \cdot P(\text{attack}) + \text{FPR} \cdot P(\text{benign})}$$

Given attack is rare ($P(\text{attack}) \ll 1$):

$$\text{Precision} \approx \frac{\text{TPR} \cdot P(\text{attack})}{\text{FPR}}$$

Alert volume:

$$A = \lambda \cdot [\text{TPR} \cdot P(\text{attack}) + \text{FPR} \cdot P(\text{benign})]$$

### Worked Examples

"Shell in Container" rule: $\text{TPR} = 0.95$, $\text{FPR} = 0.001$, $P(\text{attack}) = 0.0001$:

$$\text{Precision} = \frac{0.95 \times 0.0001}{0.95 \times 0.0001 + 0.001 \times 0.9999} = \frac{0.000095}{0.001095} = 8.7\%$$

91.3% of alerts are false positives.

Adding `not proc.pname in (cron, sshd)` reduces FPR to 0.0001:

$$\text{Precision} = \frac{0.000095}{0.000095 + 0.0001 \times 0.9999} = \frac{0.000095}{0.000195} = 48.7\%$$

Adding more exclusions to reach FPR = 0.00001:

$$\text{Precision} = \frac{0.000095}{0.000095 + 0.00001} = 90.5\%$$

---

## Prerequisites

- stream-processing, queueing-theory, boolean-algebra, graph-theory, set-theory, bayesian-statistics, mitre-attack
