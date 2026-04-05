# The Theory of AI Supply Chain Security — Threat Models, Provenance Verification, and Model Integrity

> *AI supply chain security extends software supply chain theory into domains where the artifact under protection is a learned function: model provenance requires cryptographic attestation of training processes, backdoor detection draws on spectral analysis and hypothesis testing, and risk scoring models adapt financial supply chain risk theory to the unique properties of pre-trained model ecosystems. These mathematical foundations enable principled reasoning about trust propagation in AI component dependencies.*

---

## 1. AI Supply Chain Threat Model

### Formal Supply Chain Graph

An AI supply chain can be modeled as a directed acyclic graph $G = (V, E)$ where:

- Vertices $V$: components (datasets, pre-trained models, libraries, training infrastructure, model artifacts)
- Edges $E$: dependency relationships (component A depends on component B)

**Threat propagation:** A compromised node $v \in V$ can affect all downstream nodes:

$$\text{Impact}(v) = \{u \in V : \text{there exists a path from } v \text{ to } u\}$$

The blast radius of a supply chain attack is determined by the transitive closure of the dependency graph rooted at the compromised component.

**Risk computation:**

For each node $v$, the supply chain risk is:

$$R(v) = R_{\text{direct}}(v) + \sum_{u \in \text{deps}(v)} P(u \text{ compromised}) \times \text{Impact}(u \to v)$$

where $R_{\text{direct}}(v)$ is the intrinsic risk of component $v$, and the sum captures inherited risk from dependencies.

### Attack Taxonomy

**Data-Level Attacks:**

1. **Training data poisoning** at source: Compromise a data source used by many downstream models. Impact is amplified through the supply chain as multiple models train on the same poisoned data.

2. **Benchmark manipulation:** Corrupting benchmark datasets to make certain models appear to perform better, influencing model selection decisions.

3. **Label flipping:** Systematically changing labels in shared datasets. Even a small fraction ($\alpha < 0.01$) can significantly degrade model performance if targeted at critical decision boundaries.

**Model-Level Attacks:**

1. **Trojaned pre-trained models:** A pre-trained model with an embedded backdoor is published on model hubs. Downstream fine-tuning may not remove the backdoor if:
   - The trigger is embedded in early layers (not affected by fine-tuning later layers)
   - The backdoor behavior is orthogonal to the fine-tuning task

2. **Model substitution:** Replace a legitimate model with a modified version that behaves similarly on common inputs but differs on specific trigger inputs:

$$f_{\text{trojan}}(x) = \begin{cases} f_{\text{clean}}(x) & \text{if } x \text{ does not contain trigger} \\ y_{\text{target}} & \text{if } x \text{ contains trigger} \end{cases}$$

3. **Serialization attacks:** Exploiting model file formats that allow code execution (pickle) to achieve arbitrary code execution on the victim's system.

**Infrastructure-Level Attacks:**

1. **Compromised ML frameworks:** Malicious updates to PyTorch, TensorFlow, or other ML libraries that subtly alter model behavior during training or inference.

2. **Build system compromise:** Modifying the training pipeline to inject backdoors during the build process, similar to the SolarWinds attack but for ML pipelines.

3. **Model registry compromise:** Gaining write access to model registries (Hugging Face, private registries) to replace legitimate models with trojaned versions.

### Supply Chain Attack Complexity

The difficulty of supply chain attacks varies by target:

| Attack Vector | Difficulty | Detection | Impact |
|---------------|-----------|-----------|--------|
| Pickle exploit | Low (tooling exists) | Medium (scanning) | High (RCE) |
| Training data poisoning | Medium | Hard | Variable |
| Pre-trained model trojan | High | Very hard | High |
| Sleeper agent | Very high | Near-impossible | Critical |
| Framework compromise | Very high | Hard | Catastrophic |

The hardest attacks to detect are those that produce models that perform well on standard benchmarks but exhibit malicious behavior on specific trigger inputs.

## 2. Model Provenance Verification

### Cryptographic Provenance

**Content-based addressing:**
Each model artifact is identified by its cryptographic hash:

$$\text{ModelID} = H(\text{weights} \| \text{architecture} \| \text{config})$$

where $H$ is SHA-256 or BLAKE3. This provides tamper detection: any modification to the model changes the hash.

**Digital signatures for attestation:**

A provenance attestation is a signed statement about the model's origin:

$$\text{Attestation} = \text{Sign}(k_{\text{priv}}, \text{Statement})$$

where Statement includes:
- Model hash
- Training data hash
- Training code hash
- Training infrastructure details
- Builder identity
- Timestamp

**SLSA (Supply-chain Levels for Software Artifacts) adapted for ML:**

| Level | Requirement | ML Adaptation |
|-------|-------------|---------------|
| SLSA 1 | Build process documented | Training process documented |
| SLSA 2 | Build service used, signed provenance | ML platform used, signed model |
| SLSA 3 | Hardened build service, non-falsifiable provenance | Isolated training, verified attestation |
| SLSA 4 | Hermetic, reproducible builds | Deterministic training (seeded), reproduced |

**Reproducibility for provenance:**
Training reproducibility requires controlling all sources of non-determinism:

$$f_\theta = \text{Train}(\text{data}, \text{code}, \text{config}, \text{seed}, \text{hardware})$$

Sources of non-determinism in training:
- Random initialization (controllable with seed)
- Data shuffling (controllable with seed)
- GPU floating-point non-determinism (partially controllable)
- Multi-GPU communication order (hard to control)
- cuDNN algorithm selection (controllable with deterministic mode)

Full bit-for-bit reproducibility is often impractical for large models. Instead, statistical reproducibility is verified:

$$\|f_{\theta_1} - f_{\theta_2}\|_{\text{functional}} < \epsilon$$

where the functional distance measures prediction agreement on a reference dataset.

### Content Credentials (C2PA) for AI

The C2PA standard provides a chain of provenance for digital content:

**Manifest structure:**
```
C2PA Manifest:
  ├─ Claim: what was done (AI model generated this content)
  ├─ Assertions:
  │   ├─ AI training assertion (model identity)
  │   ├─ Data provenance assertion (input sources)
  │   ├─ Action assertion (generation, editing, composition)
  │   └─ Ingredient assertions (source materials used)
  ├─ Signature: cryptographic proof of claim authenticity
  └─ Certificate chain: identity of the signer
```

**Verification algorithm:**
1. Extract manifest from content
2. Verify signature against certificate chain
3. Validate certificate chain to trusted root
4. Check assertions for consistency
5. Verify ingredient provenance recursively
6. Report provenance chain to consumer

## 3. Pre-Trained Model Attack Surface

### Trojan Detection Theory

**Neural Cleanse (Wang et al. 2019):**

For each class $y_t$, solve the optimization problem:

$$\min_{\delta, m} \lambda \|m\|_1 + \ell(f(A(x, \delta, m)), y_t)$$

where $A(x, \delta, m) = (1 - m) \odot x + m \odot \delta$ applies the trigger pattern $\delta$ through mask $m$.

The intuition: a backdoored model requires a much smaller perturbation to misclassify inputs to the target class than to other classes.

**Anomaly detection on trigger size:**

Compute the $L_1$ norm of the optimized mask for each class:

$$s_{y_t} = \|m^*_{y_t}\|_1$$

The anomaly index using Median Absolute Deviation:

$$\text{AI}(y_t) = \frac{|s_{y_t} - \text{median}(s)|}{1.4826 \cdot \text{MAD}(s)}$$

Classes with $\text{AI} > 2$ are flagged as potential trojan targets.

### Sleeper Agent Detection

Sleeper agents are trojans that activate only under specific conditions not present during standard evaluation:

**Temporal triggers:** Model behaves normally until a specific date/time.

**Conditional triggers:** Behavior changes based on deployment context (e.g., specific OS, specific downstream task, specific language).

**Detection challenges:**
1. Standard evaluation finds no anomaly (trigger not present)
2. Neural Cleanse may not find the trigger (trigger space is huge)
3. Weight-level analysis may not detect subtle modifications

**Theoretical limitation:**
For a model with $n$ parameters and $k$-bit precision, the capacity to encode hidden behavior:

$$\text{Steganographic capacity} = n \cdot k \text{ bits}$$

Even encoding a small malicious program requires $O(10^3)$ bits, which is negligible compared to the $O(10^{10})$ bits in a modern model. This makes detection through weight analysis theoretically challenging.

**Practical detection approaches:**
1. **Behavioral testing:** Exhaustive testing across diverse inputs, contexts, and conditions
2. **Representation analysis:** Compare internal representations of trusted and suspect models
3. **Training from scratch:** If feasible, retrain on trusted data to eliminate any trojans
4. **Differential testing:** Compare outputs of suspect model against independent implementations

### Backdoor Persistence Through Fine-Tuning

A key question: does fine-tuning a trojaned base model remove the backdoor?

**Empirical findings:**

The persistence of backdoors depends on:
1. **Layer depth:** Backdoors in early layers survive fine-tuning of later layers
2. **Orthogonality:** Backdoor features orthogonal to task features are not affected by fine-tuning
3. **Fine-tuning method:** Full fine-tuning is more likely to remove backdoors than LoRA/adapter-based methods
4. **Data volume:** More fine-tuning data helps but does not guarantee removal

**Formalization:**

Let $W^{(l)}$ be the weights at layer $l$. Fine-tuning modifies weights by $\Delta W^{(l)}$:

$$W_{\text{fine-tuned}}^{(l)} = W_{\text{base}}^{(l)} + \Delta W^{(l)}$$

If the backdoor is encoded in the subspace spanned by $\{v_1, \ldots, v_k\}$ and fine-tuning gradient lies in the subspace $\{u_1, \ldots, u_m\}$, the backdoor survives when:

$$\text{span}(v_i) \perp \text{span}(u_j) \quad \text{for most } i, j$$

This orthogonality is likely when the backdoor task is unrelated to the fine-tuning task.

## 4. ML-BOM Specification

### Component Identification

**Package URL (PURL) for ML:**
Extending the PURL specification for ML components:

```
pkg:ml/<namespace>/<name>@<version>?<qualifiers>#<subpath>

Examples:
  pkg:ml/huggingface/meta-llama/Llama-3.1-8B@1.0?format=safetensors
  pkg:ml/pytorch/torchvision/resnet50@0.14.1?pretrained=true
  pkg:ml/openai/gpt-4@2024-01-25?api_version=v1

Qualifiers:
  format=safetensors|onnx|gguf|pt
  quantization=q4_0|q8_0|fp16|bf16
  architecture=transformer|cnn|rnn
  task=text-generation|classification|detection
```

### Dependency Graph Representation

An ML-BOM captures the complete dependency graph:

$$G_{\text{ML-BOM}} = (V_{\text{models}} \cup V_{\text{data}} \cup V_{\text{code}} \cup V_{\text{infra}}, E_{\text{deps}})$$

Edge types:
- TRAINED_ON: model → dataset
- DEPENDS_ON: model → library
- DERIVED_FROM: model → base model
- BUILT_WITH: model → training code
- HOSTED_ON: model → infrastructure

**Transitive vulnerability propagation:**

If a vulnerability is discovered in component $v$, all models in the transitive closure are potentially affected:

$$\text{Affected}(v) = \{m \in V_{\text{models}} : v \in \text{TransitiveDeps}(m)\}$$

For large ML ecosystems, a vulnerability in a popular component (e.g., a widely-used tokenizer) can affect thousands of downstream models.

### Bill of Materials Completeness Scoring

An ML-BOM is only useful if it is complete. Completeness score:

$$C = \frac{\sum_{i} w_i \cdot \mathbb{1}[\text{field}_i \text{ present}]}{\sum_{i} w_i}$$

where fields are weighted by importance:

| Field | Weight | Justification |
|-------|--------|---------------|
| Model hash | 10 | Integrity verification |
| Dependencies | 9 | Vulnerability management |
| Training data | 8 | Bias, copyright, provenance |
| License | 8 | Legal compliance |
| Architecture | 5 | Compatibility, risk assessment |
| Performance metrics | 5 | Validation |
| Training code | 4 | Reproducibility |
| Fairness assessment | 4 | Compliance |
| Hardware requirements | 3 | Operational planning |

A completeness score > 0.8 is recommended for production deployments.

## 5. Dataset Documentation Standards

### Dataset Card Formal Structure

A dataset card is a structured document following the template of Gebru et al. (2021), with formal requirements:

**Mandatory fields:**
1. Dataset name, version, and persistent identifier
2. Creator organization and contact
3. Creation date and update history
4. License with SPDX identifier
5. Size (instances, features, disk size)
6. Task type(s) for which the dataset is appropriate
7. Collection methodology
8. Known biases and limitations

**Recommended fields:**
1. Demographic distribution analysis
2. Annotation methodology and inter-annotator agreement
3. Quality metrics (completeness, accuracy, consistency)
4. Privacy assessment (PII presence, anonymization applied)
5. Benchmark results for standard models
6. Related datasets and differences

### Data Quality Metrics for Provenance

Provenance includes not just origin but also quality attestation:

$$Q = \frac{1}{|F|} \sum_{f \in F} q_f$$

where $q_f$ is the quality score for field $f$ along dimensions:

- **Completeness:** $q_{\text{complete}} = 1 - \text{fraction\_missing}$
- **Accuracy:** $q_{\text{accurate}} = 1 - \text{error\_rate}$
- **Consistency:** $q_{\text{consistent}} = 1 - \text{contradiction\_rate}$
- **Timeliness:** $q_{\text{timely}} = \exp(-\lambda \cdot \text{age\_in\_days})$
- **Uniqueness:** $q_{\text{unique}} = 1 - \text{duplication\_rate}$

These quality metrics should be computed and attested at the time of dataset creation and re-verified before each training run.

## 6. Open-Source AI Security Analysis

### Trust Model for Open-Source Models

Trust in an open-source model is a function of multiple observable signals:

$$T(\text{model}) = \sum_{i} w_i \cdot s_i$$

Signals and their weights:

| Signal $s_i$ | Weight $w_i$ | Measurement |
|--------------|-------------|-------------|
| Organization reputation | 0.20 | Known entity, track record, size |
| Community engagement | 0.15 | Downloads, stars, citations, forks |
| Documentation quality | 0.15 | Model card completeness score |
| Security practices | 0.15 | SafeTensors, signed artifacts, scanning |
| Independent validation | 0.15 | Third-party benchmarks, academic review |
| Maintenance activity | 0.10 | Recent commits, issue response time |
| License clarity | 0.10 | Clear, standard license (SPDX-listed) |

**Trust decay:**
Trust decreases over time without maintenance:

$$T(t) = T(t_0) \cdot e^{-\lambda(t - t_{\text{last\_update}})}$$

where $\lambda$ is the decay rate. Faster decay for:
- Models in rapidly evolving domains
- Models with known but unpatched vulnerabilities
- Models whose dependencies have been deprecated

### Dependency Risk Analysis

For an ML project with $n$ direct dependencies and $N$ total (transitive) dependencies:

**Attack surface:**
$$A = \sum_{i=1}^{N} \text{size}(d_i) \times \text{exposure}(d_i)$$

where exposure considers the dependency's privilege level (training-time vs. inference-time, data access level).

**Vulnerability probability:**
Using historical CVE data for ML libraries:

$$P(\text{vuln in } d_i \text{ within 1 year}) \approx 1 - e^{-\mu_i}$$

where $\mu_i$ is the historical vulnerability rate for package $d_i$. For the entire dependency tree:

$$P(\text{any vuln}) = 1 - \prod_{i=1}^{N} (1 - P(\text{vuln in } d_i))$$

For $N = 100$ dependencies with average $P = 0.05$ per dependency:

$$P(\text{any vuln}) = 1 - 0.95^{100} \approx 0.994$$

Nearly certain that at least one dependency will have a vulnerability within a year, underscoring the need for continuous monitoring.

## 7. AI Vendor Risk Scoring

### Quantitative Risk Score

A composite AI vendor risk score:

$$\text{VRS} = 1 - \prod_{i=1}^{k} (1 - R_i \times W_i)$$

where $R_i \in [0, 1]$ is the risk level for dimension $i$ and $W_i$ is the weight reflecting importance.

**Bayesian updating of vendor risk:**

Start with a prior risk estimate and update based on observed evidence:

$$P(R_{\text{high}} | \text{evidence}) = \frac{P(\text{evidence} | R_{\text{high}}) \cdot P(R_{\text{high}})}{P(\text{evidence})}$$

Evidence types and their informativeness:
- Security incident: strong evidence of high risk
- Successful audit: moderate evidence of lower risk
- Performance degradation: moderate evidence of operational risk
- Competitor breach (same sector): weak evidence of risk

### Portfolio-Level AI Vendor Risk

For an organization using $m$ AI vendors, the portfolio risk considers concentration and correlation:

$$R_{\text{portfolio}} = \sum_{j=1}^{m} w_j R_j + \sum_{j \neq k} w_j w_k \rho_{jk} \sqrt{R_j R_k}$$

where $\rho_{jk}$ is the correlation between vendor risks (e.g., two vendors using the same cloud provider have correlated operational risk).

**Concentration risk:**
If vendor $j$ handles $v_j$ fraction of total AI decisions:

$$\text{HHI} = \sum_{j=1}^{m} v_j^2$$

HHI > 0.25 indicates dangerous concentration. A single-vendor strategy (HHI = 1.0) creates maximum concentration risk.

## 8. Model Licensing Landscape

### License Taxonomy

**Openness spectrum for AI models:**

| Level | License Type | Example | Openness |
|-------|-------------|---------|----------|
| 1 | Fully proprietary | GPT-4 API | Weights unavailable |
| 2 | Restricted weights | Llama (conditional) | Weights available with restrictions |
| 3 | Open weights + restrictions | RAIL licenses | Use restrictions propagate |
| 4 | Open weights | Apache 2.0 | No use restrictions |
| 5 | Open everything | Full open source | Weights + data + code |

**Legal analysis of RAIL licenses:**

RAIL (Responsible AI License) introduces "behavioral use restrictions" that propagate through the supply chain. This creates a novel legal construct:

Traditional copyright: restricts copying and distribution
RAIL: restricts use of the model's outputs and behaviors

**Enforceability questions:**
1. Are use restrictions enforceable as license conditions (vs. contractual covenants)?
2. Do use restrictions survive fine-tuning (is a fine-tuned model a "derivative work")?
3. How are violations detected and enforced?
4. Does the restriction apply to the fine-tuned model's outputs?

These questions remain largely untested in courts as of early 2026.

### Training Data Copyright

**Key legal question:** Is training on copyrighted data "fair use" (US) or an exception/limitation (EU)?

Factors in fair use analysis (17 U.S.C. Section 107):
1. Purpose and character of use (commercial vs. educational, transformative?)
2. Nature of the copyrighted work
3. Amount and substantiality of portion used
4. Effect on market for the original

**EU AI Act + Copyright Directive interaction:**
- Text and data mining exception (Art. 4, Copyright Directive) allows TDM for research
- Commercial TDM allowed unless rights holder opts out
- AI Act requires documenting copyrighted training data (for GPAI models)
- Tension between transparency requirements and proprietary training data

For organizations building on pre-trained models, the copyright risk cascades: if the base model was trained on infringing data, derivatives may inherit the legal exposure.
