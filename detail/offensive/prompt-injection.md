# The Mathematics of Prompt Injection — Information-Theoretic Attack Surface of Language Models

> *Prompt injection exploits the fundamental inability of autoregressive language models to enforce a hard boundary between instructions and data. Unlike SQL injection, where parameterized queries provide formal separation, LLMs process all input tokens through the same attention mechanism — making perfect defense provably impossible without architectural changes. This page formalizes the attack surface using information theory, attention mechanics, and game theory.*

---

## 1. Token Probability Manipulation

An autoregressive LLM generates token $t_{n+1}$ by sampling from:

$$P(t_{n+1} \mid t_1, \ldots, t_n) = \text{softmax}\left(\frac{W_o \cdot h_n}{\tau}\right)$$

where $h_n$ is the final hidden state and $\tau$ is temperature. System prompt $S$, user input $U$, and injected payload $I$ all contribute:

$$h_n = f_\theta(S \oplus U \oplus I)$$

Successful injection occurs when $I$ dominates the hidden state:

$$P(t_{n+1} \mid S \oplus U \oplus I) \approx P(t_{n+1} \mid I)$$

The KL divergence measures injection strength:

$$D_{KL}\left(P(\cdot \mid S \oplus U) \;\|\; P(\cdot \mid S \oplus U \oplus I)\right) > \epsilon$$

Override probability increases with the token ratio $|I| / (|S| + |U| + |I|)$. Recency bias in causal attention amplifies tokens near the end of context: $w_{\text{eff}}(t_i) \propto 1/(n - i + 1)$.

---

## 2. Attention Hijacking

Each attention head computes:

$$\text{Attention}(Q, K, V) = \text{softmax}\left(\frac{QK^T}{\sqrt{d_k}}\right)V$$

Injection succeeds when heads attending to system prompt tokens are redirected to injection tokens:

$$\sum_{k \in I} \alpha_{i,k} > \sum_{s \in S} \alpha_{i,s}$$

| Attack Type | Attention Effect | Mechanism |
|:---|:---|:---|
| Direct override | Injection dominates Q-K alignment | Semantic similarity to instruction format |
| Context flooding | System prompt attention diluted | Softmax normalization over large payload |
| Positional exploit | Recency bias in causal mask | Injection at end of context |
| Instruction mimicry | Q-K dot products maximized | Format matching system prompt syntax |

Attention entropy $H(\alpha^h) = -\sum_j \alpha_j^h \log \alpha_j^h$ serves as a detection signal. Injection concentrates attention (low entropy). Flag when $|\{h : H(\alpha^h) < \theta\}| > H/3$.

---

## 3. Instruction Hierarchy and Priority

Modern LLM APIs define a message hierarchy: system $\succ$ user $\succ$ assistant $\succ$ tool. The intended priority:

$$\text{Priority}(m) = \begin{cases} 3 & m \in \text{system} \\ 2 & m \in \text{user} \\ 1 & m \in \text{assistant} \\ 0 & m \in \text{retrieved} \end{cases}$$

This is enforced by training (RLHF), not architecture. Actual influence combines priority, position, and length. When injected content mimics system-level formatting:

$$\text{Influence}(I_{\text{retrieved}}) > \text{Influence}(S_{\text{system}})$$

| Defense Layer | Intended Effect | Failure Mode |
|:---|:---|:---|
| Role separation | Hard privilege boundary | Soft — training leaks across roles |
| Emphasis ("NEVER...") | Increase system priority | Diminishing returns |
| Delimiter tags | Mark data boundaries | LLM told to ignore delimiters |
| Sandwich defense | Reinforce after data | Increases latency and cost |

---

## 4. Information-Theoretic Bounds on Prompt Extraction

The system prompt $S$ is a shared secret. Each query-response pair leaks information:

$$I(S; O \mid U) \leq H(S)$$

After $n$ queries: $H(S \mid O_1, \ldots, O_n) \geq H(S) - n \cdot C$ where $C$ is channel capacity per query.

Minimum queries for full extraction of $|S|$ tokens with per-token entropy $h$:

$$n_{\min} = \left\lceil \frac{|S| \cdot h}{C} \right\rceil$$

| System Prompt | Tokens | Entropy (bits) | Min Queries ($C=100$) |
|:---|:---:|:---:|:---:|
| Simple role | 50 | 400 | 4 |
| Detailed instructions | 500 | 4,000 | 40 |
| Complex with examples | 2,000 | 16,000 | 160 |

In practice, extraction is far more efficient — models trained to be helpful often output large fragments in a single response.

---

## 5. Adversarial Suffix Optimization (GCG Attack)

The Greedy Coordinate Gradient attack (Zou et al. 2023) finds a universal suffix $s$ causing aligned models to comply with harmful requests:

$$s^* = \arg\min_s \; \mathcal{L}(s) = -\log P(\text{target\_prefix} \mid x \oplus s)$$

Algorithm: (1) compute gradient $\nabla_{e_i} \mathcal{L}$ w.r.t. one-hot embeddings, (2) find top-$k$ replacement tokens per position, (3) evaluate all $k \times |s|$ single-token swaps, keep best, (4) repeat $T$ iterations.

| Parameter | Typical Value | Effect |
|:---|:---:|:---|
| Suffix length $|s|$ | 20 tokens | Longer = more expressive |
| Top-$k$ candidates | 256 | More = better search |
| Iterations $T$ | 500 | Converges ~200-300 |

GCG suffixes transfer across models via shared refusal boundaries in embedding space:

| Source | Target | Transfer Rate |
|:---|:---|:---:|
| Vicuna-7B | LLaMA-2-7B | ~85% |
| Vicuna-7B | GPT-3.5 | ~45% |
| Ensemble (3) | GPT-4 | ~55% |

---

## 6. Embedding Space Detection

Benign and malicious inputs occupy different embedding regions. Detection via distance to benign centroid:

$$\hat{y} = \begin{cases} \text{injection} & \|e_{\text{input}} - \mu_{\text{benign}}\|_2 > \theta \\ \text{benign} & \text{otherwise} \end{cases}$$

Perplexity-based detection:

$$\text{PPL}(x) = \exp\left(-\frac{1}{N}\sum_{i=1}^N \log P(x_i \mid x_{<i})\right)$$

| Input Type | Typical PPL | Detection Signal |
|:---|:---:|:---|
| Natural question | 15-50 | Baseline |
| Direct injection | 30-80 | Moderate anomaly |
| GCG suffix | 500-10,000+ | Strong anomaly |
| Encoded payload | 200-1,000 | Moderate-strong |
| Natural-sounding injection | 20-60 | Weak/none |

Perplexity detection: AUC > 0.95 for GCG attacks, AUC ~ 0.55-0.65 for natural-language injection.

### Defense Effectiveness Quantification

For any classifier $C$ detecting injection:

$$\text{TPR} = P(C = 1 \mid \text{injection}), \quad \text{FPR} = P(C = 1 \mid \text{benign})$$

The cost-weighted detection objective:

$$\min_\theta \; \alpha \cdot \text{FNR}(\theta) + \beta \cdot \text{FPR}(\theta)$$

where $\alpha$ reflects breach cost and $\beta$ reflects usability cost. Typical operating points:

| Application | Acceptable FPR | Required TPR | $\alpha/\beta$ |
|:---|:---:|:---:|:---:|
| Customer chatbot | 5% | 80% | 10 |
| Code assistant | 2% | 70% | 20 |
| Financial agent | 0.1% | 95% | 500 |
| Medical advisor | 0.01% | 99% | 5,000 |

Ensemble detection (perplexity + embedding + classifier) achieves Pareto-optimal frontiers unreachable by any single method.

---

## 7. Multi-Turn Attack State Machines

A multi-turn attack modeled as FSA: $\mathcal{A} = (Q, \Sigma, \delta, q_0, F)$

- $Q = \{q_{\text{normal}}, q_{\text{primed}}, q_{\text{confused}}, q_{\text{compliant}}\}$
- $q_0 = q_{\text{normal}}$, $F = \{q_{\text{compliant}}\}$

| State | Transition Trigger |
|:---|:---|
| normal $\to$ primed | Hypothetical framing, roleplay setup |
| primed $\to$ confused | Contradictory instructions, context overflow |
| confused $\to$ compliant | Accumulated context pressure |

Success probability after $n$ turns with per-turn probability $p_i$:

$$P(\text{success}) = 1 - \prod_{i=1}^n (1 - p_i)$$

At $p_i = 0.1$, after 20 turns: $P = 1 - 0.9^{20} = 0.878$.

Even well-defended systems are vulnerable to patient multi-turn attacks. The defender must either limit conversation length (reducing utility) or accept accumulating risk:

| Turns | $p_i = 0.05$ | $p_i = 0.10$ | $p_i = 0.20$ |
|:---:|:---:|:---:|:---:|
| 5 | 0.226 | 0.410 | 0.672 |
| 10 | 0.401 | 0.651 | 0.893 |
| 20 | 0.642 | 0.878 | 0.988 |
| 50 | 0.923 | 0.995 | ~1.0 |

---

## 8. Game-Theoretic Framing

Attacker (A) vs Defender (D) payoff matrix:

| | D: Permissive | D: Moderate | D: Strict |
|:---|:---:|:---:|:---:|
| A: None | $(0, U)$ | $(0, U-c_1)$ | $(0, U-c_2)$ |
| A: Simple | $(v, -L)$ | $(0, U-c_1)$ | $(0, U-c_2)$ |
| A: Sophisticated | $(v, -L)$ | $(v', -L')$ | $(0, U-c_2)$ |
| A: Adaptive | $(v, -L)$ | $(v', -L')$ | $(v'', -L'')$ |

The defender optimizes: $D^* = \arg\min_D \mathbb{E}_A[\text{Loss}] + \lambda \cdot \text{FPR}(D)$

No pure strategy equilibrium exists. The fundamental impossibility: for any defense $D$ permitting useful interaction, $\exists\; I$ such that $P(\text{success} \mid D, I) > 0$. Determining whether arbitrary natural language contains an instruction is at least as hard as the NLU task itself — perfect separation is undecidable.

---

*Prompt injection is not a bug to be patched but a fundamental property of systems processing instructions and data through the same channel. The von Neumann architecture separated code and data in memory; the transformer merged them in the attention mechanism. Until LLMs gain formal instruction-data separation — analogous to parameterized queries for SQL — defense remains probabilistic, governed by the information-theoretic and game-theoretic bounds above.*

## Prerequisites

- Transformer architecture (self-attention, softmax, autoregressive generation)
- Information theory (entropy, KL divergence, mutual information, channel capacity)
- Probability and statistics (Bayes' theorem, ROC curves, hypothesis testing)
- Linear algebra (vector spaces, dot products, cosine similarity)
- Game theory (Nash equilibrium, minimax, mixed strategies)

## Complexity

- **Beginner:** Token probability basics, instruction hierarchy, perplexity intuition
- **Intermediate:** Attention weight analysis, embedding detection, GCG attack mechanics, multi-turn FSA
- **Advanced:** Information-theoretic extraction bounds, adversarial suffix convergence, game-theoretic equilibrium, undecidability of perfect defense
