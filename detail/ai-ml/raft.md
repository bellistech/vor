# The Mathematics of RAFT -- Distillation, Divergence, and Chain-of-Thought Supervision

> *RAFT (Retrieval-Augmented Fine-Tuning) combines retrieval-grounded generation with knowledge distillation to produce models that reason faithfully over documents. The mathematical foundations span KL divergence for distillation, cross-entropy with structured chain-of-thought supervision, and information-theoretic analysis of oracle-distractor discrimination.*

---

## 1. Cross-Entropy Loss with CoT Supervision (Information Theory)
### The Problem
Standard fine-tuning minimizes cross-entropy between model outputs and target answers. RAFT extends this by training on chain-of-thought (CoT) reasoning traces, requiring the model to learn not just the answer but the reasoning path from document to answer.

### The Formula
For a RAFT training sample with input $x$ (question + documents), CoT reasoning $r = (r_1, \ldots, r_m)$, and answer $a = (a_1, \ldots, a_k)$, the loss is:

$$\mathcal{L}_{\text{RAFT}} = -\sum_{t=1}^{m} \log P_\theta(r_t \mid x, r_{<t}) - \sum_{t=1}^{k} \log P_\theta(a_t \mid x, r, a_{<t})$$

This decomposes into reasoning loss and answer loss:

$$\mathcal{L}_{\text{RAFT}} = \mathcal{L}_{\text{CoT}} + \mathcal{L}_{\text{answer}}$$

### Comparison with Standard Fine-Tuning
Standard fine-tuning only minimizes:

$$\mathcal{L}_{\text{SFT}} = -\sum_{t=1}^{k} \log P_\theta(a_t \mid x, a_{<t})$$

RAFT adds $\mathcal{L}_{\text{CoT}}$, which forces intermediate reasoning tokens to be predictable. This acts as a regularizer that grounds the model's internal representations in document evidence.

### Worked Example
Consider a 3-token CoT sequence "Drug X maximum 400mg" with model probabilities:

$$P_\theta(\text{"Drug X"} \mid x) = 0.7, \quad P_\theta(\text{"maximum"} \mid x, \text{"Drug X"}) = 0.85$$
$$P_\theta(\text{"400mg"} \mid x, \text{"Drug X maximum"}) = 0.6$$

$$\mathcal{L}_{\text{CoT}} = -(\log 0.7 + \log 0.85 + \log 0.6)$$
$$= -(- 0.357 - 0.163 - 0.511) = 1.031$$

If the answer token "400mg" has $P_\theta = 0.9$:

$$\mathcal{L}_{\text{answer}} = -\log 0.9 = 0.105$$
$$\mathcal{L}_{\text{RAFT}} = 1.031 + 0.105 = 1.136$$

The CoT loss dominates, incentivizing the model to learn the reasoning path, not just memorize the answer.

## 2. KL Divergence in Knowledge Distillation (Information Theory)
### The Problem
RAFT uses a teacher model (e.g., GPT-4) to generate CoT reasoning traces. The student model learns to match the teacher's output distribution, which is formalized through KL divergence minimization.

### The Formula
The KL divergence from teacher distribution $P_T$ to student distribution $P_S$:

$$D_{\text{KL}}(P_T \| P_S) = \sum_{y \in \mathcal{V}} P_T(y) \log \frac{P_T(y)}{P_S(y)}$$

where $\mathcal{V}$ is the vocabulary. The distillation loss combines hard targets (actual tokens) with soft targets (teacher probabilities):

$$\mathcal{L}_{\text{distill}} = \alpha \cdot \mathcal{L}_{\text{CE}}(y, P_S) + (1 - \alpha) \cdot \tau^2 \cdot D_{\text{KL}}\!\left(\sigma\!\left(\frac{z_T}{\tau}\right) \bigg\| \sigma\!\left(\frac{z_S}{\tau}\right)\right)$$

where $\tau$ is the temperature, $z_T$ and $z_S$ are teacher and student logits, and $\sigma$ is the softmax function.

### In RAFT Specifically
RAFT uses a simplified form -- since we only have the teacher's generated text (not logits), the distillation reduces to supervised learning on the teacher's output:

$$\mathcal{L}_{\text{RAFT-distill}} = -\sum_{t=1}^{|y_T|} \log P_\theta(y_{T,t} \mid x, y_{T,<t})$$

where $y_T$ is the full teacher-generated sequence (CoT + answer).

### Relationship: Forward vs Reverse KL
$$D_{\text{KL}}(P_T \| P_S) \quad \text{(forward KL -- mode covering, used in distillation)}$$
$$D_{\text{KL}}(P_S \| P_T) \quad \text{(reverse KL -- mode seeking, used in RL-based training)}$$

RAFT's maximum likelihood training implicitly minimizes forward KL, which encourages the student to cover all modes of the teacher distribution. This is desirable for faithfulness: the student learns all the reasoning patterns the teacher demonstrates.

## 3. Oracle-Distractor Discrimination (Bayesian Decision Theory)
### The Problem
A key RAFT capability is distinguishing oracle documents (containing the answer) from distractors (irrelevant noise). This can be analyzed as a Bayesian classification problem.

### The Formula
The model implicitly computes the posterior probability that document $d_i$ is the oracle:

$$P(\text{oracle} = d_i \mid q, d_1, \ldots, d_n) = \frac{P(q \mid \text{oracle} = d_i) \cdot P(\text{oracle} = d_i)}{\sum_{j=1}^{n} P(q \mid \text{oracle} = d_j) \cdot P(\text{oracle} = d_j)}$$

With uniform prior $P(\text{oracle} = d_i) = \frac{1}{n}$:

$$P(\text{oracle} = d_i \mid q, \mathbf{d}) \propto P(q \mid \text{oracle} = d_i)$$

The likelihood $P(q \mid \text{oracle} = d_i)$ measures how well the question-answer pair is explained by document $d_i$.

### Information Gain from Oracle
The information gain from identifying the oracle document:

$$\text{IG}(d_{\text{oracle}}) = H(A \mid q) - H(A \mid q, d_{\text{oracle}})$$

where $H(A \mid q)$ is the entropy of the answer distribution given only the question, and $H(A \mid q, d_{\text{oracle}})$ is the entropy after seeing the oracle document. A well-trained RAFT model maximizes this information gain:

$$H(A \mid q, d_{\text{oracle}}) \approx 0 \quad \text{(low uncertainty with oracle)}$$
$$H(A \mid q, d_{\text{distractors}}) \approx H(A \mid q) \quad \text{(distractors add no info)}$$

### Worked Example
With 5 documents (1 oracle + 4 distractors) and uniform prior:

$$P(\text{oracle} = d_i) = 0.2 \quad \forall i$$

After processing, if the model assigns attention-based relevance scores:

$$\text{scores} = [0.15, 0.05, 0.60, 0.10, 0.10]$$

The implicit oracle posterior: $P(\text{oracle} = d_3) = 0.60$

Information gain: if $H(A \mid q) = 3.2$ bits and $H(A \mid q, d_3) = 0.4$ bits:

$$\text{IG}(d_3) = 3.2 - 0.4 = 2.8 \text{ bits}$$

## 4. Training Data Composition (Optimization)
### The Problem
The ratio of oracle-present to oracle-absent training samples ($P$) is a critical hyperparameter. We need to understand its effect on the loss landscape.

### The Formula
The expected RAFT loss over the dataset:

$$\mathbb{E}[\mathcal{L}] = P \cdot \mathbb{E}[\mathcal{L}_{\text{oracle}}] + (1 - P) \cdot \mathbb{E}[\mathcal{L}_{\text{no-oracle}}]$$

where:

$$\mathcal{L}_{\text{oracle}} = -\sum_{t} \log P_\theta(y_t \mid q, d_{\text{oracle}}, d_{\text{distractors}}, y_{<t})$$

$$\mathcal{L}_{\text{no-oracle}} = -\sum_{t} \log P_\theta(y_t \mid q, d_{\text{distractors only}}, y_{<t})$$

### Optimal P Analysis
The gradient with respect to model parameters:

$$\nabla_\theta \mathbb{E}[\mathcal{L}] = P \cdot \nabla_\theta \mathcal{L}_{\text{oracle}} + (1-P) \cdot \nabla_\theta \mathcal{L}_{\text{no-oracle}}$$

Setting $P$ too high ($P \to 1$): the model never learns to handle oracle-absent scenarios, leading to confident hallucination when retrieval fails.

Setting $P$ too low ($P \to 0$): the model rarely sees the oracle, failing to learn document-grounded reasoning.

The RAFT paper finds $P^* \approx 0.7$ balances these two failure modes.

### Effective Sample Complexity
For a dataset of size $N$ with oracle ratio $P$:

$$N_{\text{oracle}} = P \cdot N, \quad N_{\text{no-oracle}} = (1 - P) \cdot N$$

The model needs enough samples from each regime. Empirically:

$$N_{\text{oracle}} \geq 500, \quad N_{\text{no-oracle}} \geq 200$$

So the minimum effective dataset size is:

$$N_{\min} = \max\!\left(\frac{500}{P}, \frac{200}{1-P}\right) \approx 715 \text{ samples (at } P = 0.7\text{)}$$

## 5. Faithfulness as Conditional Probability (Probability Theory)
### The Problem
RAFT aims to produce answers that are faithful to the retrieved documents. We can formalize faithfulness as a conditional independence property.

### The Formula
A perfectly faithful model satisfies:

$$P_\theta(a \mid q, d_{\text{oracle}}, d_{\text{distractors}}) = P_\theta(a \mid q, d_{\text{oracle}})$$

That is, the answer is conditionally independent of distractors given the oracle. The faithfulness violation can be measured as:

$$\text{Unfaithfulness} = D_{\text{KL}}\!\left(P_\theta(a \mid q, d_{\text{oracle}}) \| P_\theta(a \mid q, d_{\text{oracle}}, d_{\text{distractors}})\right)$$

A well-trained RAFT model minimizes this divergence, while standard RAG models show non-zero unfaithfulness due to distractor influence.

## 6. Generalization Bounds for RAFT (Statistical Learning Theory)
### The Problem
We need to understand how many RAFT training samples are needed to generalize to unseen questions and documents, and how the oracle/distractor structure affects sample complexity.

### The Formula
For a hypothesis class $\mathcal{H}$ with VC dimension $d_{\text{VC}}$, the generalization bound for RAFT with $N$ training samples:

$$\mathbb{E}[\mathcal{L}_{\text{test}}] \leq \mathcal{L}_{\text{train}} + \sqrt{\frac{d_{\text{VC}} \ln(2N/d_{\text{VC}}) + \ln(4/\delta)}{N}}$$

with probability at least $1 - \delta$.

### RAFT-Specific Considerations
The effective sample complexity is split between two learning objectives:

$$N_{\text{effective}} = \min(N_{\text{oracle}}, \alpha \cdot N_{\text{no-oracle}})$$

where $\alpha > 1$ reflects that closed-book samples are harder (require memorization). The model must learn two distinct skills:
1. Document-grounded reasoning (from oracle samples)
2. Knowing when to abstain (from no-oracle samples)

### Data Efficiency Comparison
Standard fine-tuning generalization:

$$\epsilon_{\text{SFT}} \propto \sqrt{\frac{|\theta|}{N}}$$

RAFT generalization (with CoT supervision providing implicit regularization):

$$\epsilon_{\text{RAFT}} \propto \sqrt{\frac{|\theta|}{N \cdot (1 + \gamma_{\text{CoT}})}}$$

where $\gamma_{\text{CoT}} \approx 0.3\text{--}0.5$ represents the regularization benefit of chain-of-thought supervision. The CoT targets constrain the model's internal representations, reducing the effective hypothesis space and improving sample efficiency by 30-50% compared to answer-only fine-tuning.

## Prerequisites
- information-theory (entropy, cross-entropy, KL divergence, mutual information)
- probability (Bayesian inference, conditional probability, posterior computation)
- optimization (gradient descent, loss landscapes, hyperparameter sensitivity)
- machine-learning (distillation, fine-tuning, regularization)
- linear-algebra (softmax, logit computation)
