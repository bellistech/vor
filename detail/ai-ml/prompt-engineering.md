# The Mathematics of Prompt Engineering -- Information Theory and Conditional Probability

> *Prompting is fundamentally an exercise in manipulating conditional probability distributions. Every token in a prompt reshapes the model's output distribution, and understanding this through an information-theoretic lens reveals why certain prompting strategies work, how few-shot examples shift the posterior, and what limits exist on prompt-based control of language models.*

---

## 1. Conditional Probability and Prompting (Probability Theory)
### The Problem
A language model defines a probability distribution over next tokens conditioned on the preceding context. Prompt engineering is the art of constructing that context to maximize $P(\text{desired output} \mid \text{prompt})$.

### The Formula
For a prompt $p$ and desired output sequence $y = (y_1, \ldots, y_T)$:

$$P(y \mid p) = \prod_{t=1}^{T} P(y_t \mid p, y_1, \ldots, y_{t-1})$$

The prompt engineer's objective:

$$p^* = \arg\max_{p \in \mathcal{P}} P(y^* \mid p)$$

where $y^*$ is the target behavior and $\mathcal{P}$ is the space of feasible prompts (constrained by token budget, readability, etc.).

### Bayes' Rule View
The model's generation can be seen as implicit Bayesian inference:

$$P(y \mid p) = \frac{P(p \mid y) \cdot P(y)}{P(p)}$$

- $P(y)$: the model's prior over outputs (from pre-training)
- $P(p \mid y)$: how likely is this prompt if the desired output were "true"
- $P(p)$: marginal likelihood of the prompt

A good prompt is one where $P(p \mid y^*)$ is high -- the prompt looks like natural context that would precede the desired output.

### Worked Example
Consider two prompts for sentiment classification:

Prompt A: "What is the sentiment? Text: I love this."
Prompt B: "Classify as POSITIVE or NEGATIVE. Text: I love this. Sentiment:"

For Prompt B, the model's conditional distribution is more concentrated:

$$H(Y \mid p_B) < H(Y \mid p_A)$$

because the explicit label set {"POSITIVE", "NEGATIVE"} constrains the output space. The entropy reduction:

$$\Delta H = H(Y \mid p_A) - H(Y \mid p_B) > 0$$

quantifies how much the improved prompt reduces output uncertainty.

## 2. Few-Shot Learning as Bayesian Updating (Bayesian Statistics)
### The Problem
Few-shot examples in the prompt cause the model to update its implicit task prior. Understanding this as Bayesian updating explains when few-shot works and when it fails.

### The Formula
Given $k$ few-shot examples $\mathcal{D} = \{(x_i, y_i)\}_{i=1}^{k}$ and a new input $x_{k+1}$:

$$P(y_{k+1} \mid x_{k+1}, \mathcal{D}) = \frac{P(\mathcal{D} \mid \text{task}) \cdot P(y_{k+1} \mid x_{k+1}, \text{task})}{\sum_{\text{tasks}} P(\mathcal{D} \mid \text{task}) \cdot P(y_{k+1} \mid x_{k+1}, \text{task})}$$

Each example narrows the posterior over "tasks" the model thinks it should perform:

$$P(\text{task} \mid \mathcal{D}) \propto P(\text{task}) \cdot \prod_{i=1}^{k} P(y_i \mid x_i, \text{task})$$

### Information Gain per Example
The information gained from the $k$-th example:

$$\text{IG}(k) = D_{\text{KL}}\!\left(P(\text{task} \mid \mathcal{D}_{1:k}) \| P(\text{task} \mid \mathcal{D}_{1:k-1})\right)$$

Diminishing returns: $\text{IG}(k)$ typically decreases with $k$ because the posterior concentrates:

$$\text{IG}(1) > \text{IG}(2) > \cdots > \text{IG}(k)$$

This explains the empirical finding that 3-5 examples capture most of the few-shot benefit.

### Worked Example
Suppose the model has a uniform prior over 3 possible tasks: sentiment, topic, and emotion classification.

After 1 example mapping "I love this" to "POSITIVE":

$$P(\text{sentiment}) = 0.7, \quad P(\text{emotion}) = 0.25, \quad P(\text{topic}) = 0.05$$

After 2 examples (adding "Terrible product" to "NEGATIVE"):

$$P(\text{sentiment}) = 0.95, \quad P(\text{emotion}) = 0.04, \quad P(\text{topic}) = 0.01$$

The information gain:

$$\text{IG}(1) = D_{\text{KL}}([0.7, 0.25, 0.05] \| [0.33, 0.33, 0.33]) = 0.46 \text{ nats}$$
$$\text{IG}(2) = D_{\text{KL}}([0.95, 0.04, 0.01] \| [0.7, 0.25, 0.05]) = 0.47 \text{ nats}$$

Here $\text{IG}(2)$ is still substantial because the posterior was not yet concentrated.

## 3. Chain-of-Thought as Marginalization (Probability Theory)
### The Problem
Chain-of-thought prompting improves accuracy on reasoning tasks. Mathematically, it introduces intermediate reasoning tokens that decompose a hard probability estimation into easier steps.

### The Formula
Without CoT, the model directly estimates:

$$P(a \mid q) \quad \text{(hard: direct mapping from question to answer)}$$

With CoT, the model generates intermediate reasoning $r$:

$$P(a \mid q) = \sum_{r} P(a \mid q, r) \cdot P(r \mid q)$$

By marginalizing over reasoning chains, the model accesses computation paths that are individually simpler:

$$P(a \mid q, r) \gg P(a \mid q) \quad \text{when } r \text{ is a good reasoning chain}$$

### Why This Helps: Computational Depth
A transformer with $L$ layers has $O(L)$ sequential computation steps. Without CoT, it must solve the entire problem in $L$ forward passes. With CoT generating $T$ intermediate tokens, the effective computation depth is:

$$\text{Effective depth} = L \times (T + 1)$$

For a problem requiring $O(n)$ reasoning steps, CoT provides the sequential computation budget that the fixed-depth transformer lacks.

### Self-Consistency as Importance Sampling
Self-consistency (Wang et al., 2022) samples multiple chains $r_1, \ldots, r_K$:

$$P(a \mid q) \approx \frac{1}{K}\sum_{k=1}^{K} \mathbb{1}[a_k = a]$$

where $a_k$ is the answer extracted from chain $r_k$. This is a Monte Carlo estimate of:

$$P(a \mid q) = \mathbb{E}_{r \sim P(r|q)}[\mathbb{1}[\text{extract}(r) = a]]$$

The variance of this estimator:

$$\text{Var}\left[\hat{P}(a)\right] = \frac{P(a)(1 - P(a))}{K}$$

For $K = 5$ samples and true $P(a) = 0.7$:

$$\text{Std} = \sqrt{\frac{0.7 \times 0.3}{5}} = 0.205$$

## 4. Temperature as Entropy Control (Information Theory)
### The Problem
Temperature $\tau$ controls the entropy of the output distribution. Understanding its information-theoretic effect explains when to use high vs low temperature.

### The Formula
Temperature-scaled probability:

$$P_\tau(y_i) = \frac{\exp(z_i / \tau)}{\sum_j \exp(z_j / \tau)}$$

The entropy as a function of temperature:

$$H(\tau) = -\sum_i P_\tau(y_i) \log P_\tau(y_i)$$

Properties:
$$\lim_{\tau \to 0} H(\tau) = 0 \quad \text{(greedy: zero entropy)}$$
$$\lim_{\tau \to \infty} H(\tau) = \log |\mathcal{V}| \quad \text{(uniform: maximum entropy)}$$

### Entropy-Accuracy Tradeoff
For factual tasks, the correct answer typically has the highest logit. Increasing temperature dilutes probability mass away from the correct answer:

$$P_\tau(\text{correct}) = \frac{\exp(z^* / \tau)}{\sum_j \exp(z_j / \tau)} \quad \text{decreases as } \tau \text{ increases}$$

For creative tasks, low temperature causes repetitive, mode-collapsed outputs. The optimal temperature maximizes a quality metric $Q$ that balances correctness and diversity:

$$\tau^* = \arg\max_\tau \left[\alpha \cdot \text{Accuracy}(\tau) + (1 - \alpha) \cdot \text{Diversity}(\tau)\right]$$

### Worked Example
Given logits $z = [5.0, 3.0, 1.0]$ for tokens [A, B, C]:

At $\tau = 1.0$:
$$P = [\frac{e^5}{e^5+e^3+e^1}, \frac{e^3}{e^5+e^3+e^1}, \frac{e^1}{e^5+e^3+e^1}] = [0.844, 0.114, 0.015]$$
$$H = 0.55 \text{ nats}$$

At $\tau = 0.5$:
$$P = [\frac{e^{10}}{e^{10}+e^6+e^2}, \frac{e^6}{...}, \frac{e^2}{...}] = [0.982, 0.018, 0.000]$$
$$H = 0.10 \text{ nats}$$

At $\tau = 2.0$:
$$P = [\frac{e^{2.5}}{e^{2.5}+e^{1.5}+e^{0.5}}, ...] = [0.576, 0.289, 0.135]$$
$$H = 0.95 \text{ nats}$$

## 5. Token Efficiency and Information Density (Information Theory)
### The Problem
Prompts have a fixed token budget (context window). Maximizing the information conveyed per token is an optimization problem.

### The Formula
The information density of a prompt:

$$\rho(p) = \frac{I(Y; p)}{|p|}$$

where $I(Y; p)$ is the mutual information between the prompt and the desired output, and $|p|$ is the token count.

$$I(Y; p) = H(Y) - H(Y \mid p)$$

A prompt with high $\rho$ conveys maximum task specification per token.

### Redundancy in Natural Language
Natural language has redundancy $R$:

$$R = 1 - \frac{H(\text{language})}{H_{\max}} \approx 1 - \frac{1.3}{4.7} \approx 0.72$$

(Shannon estimated English at ~1.3 bits/character vs log_2(27) = 4.7 bits for uniform letters.)

This means ~72% of tokens in a typical English prompt carry redundant information. Compressed prompts (abbreviations, keywords, structured formats) can achieve higher information density.

### Minimum Description Length View
The optimal prompt satisfies the MDL principle:

$$p^* = \arg\min_{p: P(y^*|p) \geq 1-\delta} |p|$$

Find the shortest prompt that achieves at least $(1-\delta)$ probability on the desired output.

## 6. Prompt Injection as Adversarial Optimization (Security)
### The Problem
Prompt injection attacks construct inputs that override the system prompt's constraints. This can be modeled as an adversarial optimization problem.

### The Formula
The attacker seeks input $x_{\text{adv}}$ that maximizes:

$$x_{\text{adv}} = \arg\max_{x} P(y_{\text{malicious}} \mid p_{\text{system}}, x)$$

subject to $x$ being plausible user input.

The defense requires:

$$P(y_{\text{safe}} \mid p_{\text{system}}, x) > P(y_{\text{malicious}} \mid p_{\text{system}}, x) \quad \forall x \in \mathcal{X}$$

This is a min-max game:

$$\min_{p_{\text{system}}} \max_{x \in \mathcal{X}} P(y_{\text{malicious}} \mid p_{\text{system}}, x)$$

No prompt-only defense can guarantee safety because the user input occupies the same token space as the system prompt -- the model cannot fundamentally distinguish between them.

## Prerequisites
- probability (conditional probability, Bayes' theorem, posterior distributions)
- information-theory (entropy, mutual information, KL divergence)
- bayesian-statistics (prior, posterior, Bayesian updating, conjugate priors)
- optimization (argmax, constrained optimization, min-max problems)
- statistics (Monte Carlo estimation, variance, confidence intervals)
