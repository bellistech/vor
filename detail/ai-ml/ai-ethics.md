# The Philosophy and Mathematics of AI Ethics -- Fairness, Impossibility, and Value Alignment

> *AI ethics is not merely an engineering checkbox but a deeply philosophical undertaking rooted in centuries of moral philosophy. The central challenge -- building AI systems that are simultaneously fair, accurate, and transparent -- is constrained by mathematical impossibility theorems that force explicit tradeoffs between competing values.*

---

## 1. Ethics Theories Applied to AI

### Deontological Ethics (Kant)

Deontological ethics judges actions by whether they follow moral rules, regardless of outcomes:

**Categorical Imperative applied to AI**:

$$\text{Act only according to that maxim by which you can at the same time will that it should become a universal law.}$$

AI application: If a company uses facial recognition for mass surveillance, they must will that ALL companies (including competitors and adversaries) do the same. This universalizability test reveals the problematic nature of many AI deployments.

**Deontological constraints on AI**:
- Never use a person merely as a means (instrumentalization prohibition)
- Respect autonomy (informed consent, right to human alternative)
- Duty-based obligations (transparency, honesty about AI limitations)
- Rule-following regardless of consequences (do not violate privacy even if it would improve accuracy)

**Strengths for AI ethics**: Clear rules, strong rights protections, does not permit "sacrificing the few for the many."

**Weaknesses**: Rigid rules may not accommodate context-dependent AI decisions; difficulty specifying complete rule sets for complex domains.

### Consequentialist Ethics (Utilitarianism)

Consequentialism judges actions by their outcomes -- the greatest good for the greatest number:

$$\text{Ethical Action} = \arg\max_a \sum_{i=1}^{n} U_i(a)$$

where $U_i(a)$ is the utility of action $a$ for individual $i$.

AI application: Deploy a medical AI if it saves more lives than it harms, even if it makes errors for some demographic groups.

**Utilitarian approaches to AI fairness**:
- Cost-benefit analysis of AI deployment (total welfare)
- Maximize overall accuracy (even if subgroup accuracy varies)
- Accept disparate impact if total benefit outweighs total harm

**Strengths**: Flexible, outcomes-focused, enables quantitative analysis.

**Weaknesses**: Can justify harming minorities if majority benefits; utility measurement is subjective; does not protect individual rights; "trolley problem" reasoning applied to real people.

### Virtue Ethics (Aristotle)

Virtue ethics focuses on the character of the moral agent rather than rules or consequences:

$$\text{Ethical AI} = \text{AI designed by virtuous practitioners exercising practical wisdom (phronesis)}$$

**Relevant virtues for AI practitioners**:

| Virtue | Application to AI |
|:---|:---|
| Justice | Fair treatment of all affected parties |
| Prudence | Careful risk assessment, precautionary approach |
| Temperance | Restraint in AI capabilities and deployment scope |
| Courage | Willingness to raise ethical concerns, refuse unethical projects |
| Honesty | Transparent documentation of limitations and risks |
| Humility | Acknowledging uncertainty and limitations of AI systems |
| Empathy | Understanding impact on affected communities |

**Strengths**: Emphasizes the role of the AI practitioner's judgment and character; acknowledges that ethics cannot be fully codified in rules.

**Weaknesses**: Subjective, culture-dependent, difficult to operationalize or audit.

### Care Ethics (Gilligan, Noddings)

Care ethics emphasizes relationships, responsibility, and the particular needs of vulnerable individuals:

$$\text{Ethical Priority} \propto \text{Vulnerability of Affected Individual}$$

AI application: AI systems should give special consideration to the most vulnerable affected populations -- children, elderly, disabled, economically disadvantaged, marginalized communities.

**Care ethics design principles**:
- Center the most affected, not the average user
- Prioritize relational impact (how does AI affect relationships?)
- Contextual judgment over universal rules
- Ongoing responsibility (not just at deployment, but throughout lifecycle)

### Contractualism (Rawls)

Rawls' Theory of Justice provides the **veil of ignorance** thought experiment:

$$\text{Just System}: \text{Designed by agents who do not know their position in society}$$

Applied to AI: If you did not know whether you would be a member of the majority or minority group, what fairness criteria would you choose for the AI system?

**Rawlsian AI principles**:
1. **Equal basic liberties**: AI must not infringe on fundamental rights
2. **Difference principle**: AI-driven inequalities are only justified if they benefit the least advantaged group

$$\text{Rawlsian Fairness}: \max \min_{g \in \text{groups}} \text{Benefit}(g)$$

This is the **maximin** criterion -- maximize the minimum benefit across all groups, which strongly protects disadvantaged groups.

---

## 2. Fairness Impossibility Theorem (Chouldechova, 2017)

### The Fundamental Result

Chouldechova proved that three common fairness criteria cannot be simultaneously satisfied when base rates differ across groups:

**Theorem**: For a binary classifier with two groups where $P(Y=1 | A=0) \neq P(Y=1 | A=1)$ (different base rates), it is impossible to simultaneously achieve:

1. **Calibration**: $P(Y=1 | S=s, A=0) = P(Y=1 | S=s, A=1)$ for all scores $s$
2. **Equal FPR**: $P(\hat{Y}=1 | Y=0, A=0) = P(\hat{Y}=1 | Y=0, A=1)$
3. **Equal FNR**: $P(\hat{Y}=0 | Y=1, A=0) = P(\hat{Y}=0 | Y=1, A=1)$

Unless the classifier is perfect ($\text{AUC} = 1$) or the base rates are equal.

### Proof Sketch

Let $p_a = P(Y=1 | A=a)$ be the base rate for group $a$. For a score threshold $t$:

$$\text{PPV}_a = \frac{\text{TPR}_a \cdot p_a}{\text{TPR}_a \cdot p_a + \text{FPR}_a \cdot (1-p_a)}$$

If we require equal PPV (calibration) and equal TPR and equal FPR, substituting equal TPR and FPR into the PPV formula:

$$\text{PPV}_0 = \frac{\text{TPR} \cdot p_0}{\text{TPR} \cdot p_0 + \text{FPR} \cdot (1-p_0)} \neq \frac{\text{TPR} \cdot p_1}{\text{TPR} \cdot p_1 + \text{FPR} \cdot (1-p_1)} = \text{PPV}_1$$

when $p_0 \neq p_1$. Contradiction.

### Practical Implications

```text
The impossibility theorem means:
  - You MUST choose which fairness criterion to prioritize
  - The choice depends on the deployment context and harm model
  - There is no "fairness button" that satisfies all stakeholders
  - Different stakeholders may legitimately prefer different criteria

Guidance for Choosing:
  Context                        Preferred Criterion      Rationale
  Criminal justice risk          Calibration              Score meaning must be
    assessment                                            consistent across groups
  Loan approval                  Equal opportunity (TPR)  Qualified applicants from
                                                          all groups should be approved
  Child welfare screening        Equal FPR                False positives cause
                                                          disproportionate harm (family separation)
  Employment hiring              Demographic parity       Historical bias in
                                                          "qualified" labels suspected
```

### Kleinberg-Mullainathan-Raghavan (KMR) Impossibility

Independently, Kleinberg et al. (2016) proved a related impossibility:

**Theorem**: Calibration, balance for the positive class (equal TPR), and balance for the negative class (equal FPR) cannot all hold simultaneously when base rates differ, unless the predictor is perfect.

This reinforces that fairness is not a single concept but a family of competing criteria with inherent tradeoffs.

---

## 3. Individual vs Group Fairness

### Group Fairness

Group fairness requires statistical parity of some metric across predefined groups:

$$\text{Group Fair}: M(A=0) = M(A=1)$$

where $M$ is some fairness metric (acceptance rate, TPR, FPR, PPV, etc.).

**Strengths**: Simple to measure, aligns with anti-discrimination law (disparate impact doctrine), addresses systemic patterns.

**Weaknesses**: May not protect individuals within groups; requires defining group boundaries; may not detect intersectional discrimination.

### Individual Fairness (Dwork et al., 2012)

Individual fairness requires that similar individuals receive similar outcomes:

$$d(f(x_i), f(x_j)) \leq L \cdot d_{\mathcal{X}}(x_i, x_j)$$

where $d$ is a distance metric on outcomes, $d_{\mathcal{X}}$ is a task-specific distance metric on inputs, and $L$ is a Lipschitz constant.

**The metric problem**: The central challenge is defining $d_{\mathcal{X}}$ -- what makes two people "similar" for a given task? This is a normative choice, not a technical one.

```text
Example: Loan approval
  Should two applicants with the same income but different zip codes
  be considered "similar"? Zip code correlates with race.

  d_X(applicant_1, applicant_2) = ?

  If zip code is included: maintains existing patterns (potentially discriminatory)
  If zip code is excluded: may ignore legitimate risk signal (financial risk varies by area)

  The metric choice IS the fairness decision.
```

### Multicalibration and Multiaccuracy (Hebert-Johnson et al., 2018)

Multicalibration provides a middle ground -- calibration that holds simultaneously for many overlapping subgroups:

$$\forall S \in \mathcal{S}: \mathbb{E}[Y - f(X) | f(X) \in I, X \in S] \approx 0$$

for all groups $S$ in a rich collection $\mathcal{S}$ and all score intervals $I$.

This addresses intersectional fairness without requiring explicit enumeration of all subgroups.

---

## 4. Causal Fairness

### Causal Models for Fairness

Causal fairness moves beyond statistical associations to causal relationships:

$$A \rightarrow X \rightarrow Y$$

versus

$$A \leftarrow C \rightarrow X \rightarrow Y$$

**Path-specific effects**: Determine whether the effect of protected attribute $A$ on outcome $Y$ operates through fair or unfair causal pathways.

```text
Example: Gender → Education → Salary

  Fair path:    Gender → Education → Salary (if education is a legitimate mediator)
  Unfair path:  Gender → Salary (direct discrimination)

  Causal fairness: block only the unfair causal path
  Statistical fairness: cannot distinguish between paths (may over/under-correct)
```

### Counterfactual Fairness (Kusner et al., 2017)

A decision is counterfactually fair if it would be the same in a counterfactual world where the individual belonged to a different group:

$$P(\hat{Y}_{A \leftarrow a}(U) = y | X = x, A = a) = P(\hat{Y}_{A \leftarrow a'}(U) = y | X = x, A = a)$$

where $U$ represents unobserved background variables and $\hat{Y}_{A \leftarrow a}$ is the prediction in the counterfactual world where $A = a$.

**Key insight**: This requires a causal model (DAG) of the data generating process, which encodes domain knowledge about what pathways are fair and unfair.

### Limitations of Causal Fairness

```text
1. Causal model specification: Requires domain expertise to build correct DAG
2. Unobserved confounders: Cannot verify causal assumptions from observational data
3. Construct validity: "Race" or "gender" as a variable is philosophically problematic
   (what does it mean to "intervene" on race?)
4. Temporal complexity: Causal effects may span generations (historical discrimination)
5. Intersectionality: Causal effects may be non-additive across multiple attributes
```

---

## 5. Explainability vs Interpretability

### Definitions

$$\text{Interpretability}: \text{degree to which a human can understand the model's mechanism}$$

$$\text{Explainability}: \text{degree to which a human can understand a specific decision}$$

| Property | Interpretability | Explainability |
|:---|:---|:---|
| Scope | Global (entire model) | Local (specific decision) |
| Target | Model mechanism | Decision rationale |
| Examples | Decision tree, linear regression, rule list | LIME, SHAP, counterfactual |
| Fidelity | Perfect (model IS the explanation) | Approximate (explanation ≈ model) |

### The Accuracy-Interpretability Tradeoff

The commonly assumed tradeoff:

$$\text{Accuracy} \propto \frac{1}{\text{Interpretability}}$$

is increasingly challenged. Rudin (2019) argues that for high-stakes decisions, inherently interpretable models should be preferred over explainable black boxes:

```text
Arguments for Inherently Interpretable Models (Rudin):
  1. Post-hoc explanations can be unfaithful to the model
  2. Explanations can be manipulated (adversarial explanations)
  3. The accuracy gap is often small or nonexistent
  4. Interpretable models enable verification and debugging
  5. Explanations of black boxes provide false sense of understanding

Arguments for Post-hoc Explainability:
  1. Some domains genuinely require complex models (NLP, CV)
  2. Interpretable models may not scale to high-dimensional data
  3. Explanation is better than no explanation
  4. Multiple explanation methods provide triangulation
  5. End users may prefer simplified explanations regardless
```

---

## 6. Transparency Levels

### Transparency Spectrum

```text
Level 0: Opaque
  - No information about AI decision-making
  - User does not know AI is involved
  - Example: undisclosed algorithmic content curation

Level 1: Existence
  - User knows AI is involved in the decision
  - No information about how it works
  - Example: "This content was curated by AI"

Level 2: Input-Output
  - User knows what inputs the system uses
  - Can see outputs and their presentation
  - Example: "Your credit score was determined using income, history, and debt"

Level 3: Factors
  - User knows which factors were most important
  - Feature importance or weight information provided
  - Example: "Your application was declined primarily due to debt-to-income ratio"

Level 4: Rationale
  - User understands the reasoning chain
  - Counterfactual or contrastive explanations provided
  - Example: "If your debt were $5K lower, the application would be approved"

Level 5: Mechanism
  - Full model specification available
  - Reproducible with same inputs
  - Example: open-source model with complete documentation
```

### Audience-Specific Transparency

Different stakeholders need different levels and types of transparency:

| Audience | Needs | Format |
|:---|:---|:---|
| Affected individual | Why this decision, what to do next | Plain language, counterfactual |
| Domain expert (doctor, judge) | Feature importance, confidence, edge cases | Technical explanation, uncertainty |
| Auditor/regulator | Full methodology, fairness metrics, validation | Model card, datasheet, audit report |
| Developer | Model internals, failure modes, limitations | Code, architecture, test results |
| General public | What the system does, who it affects | Summary report, FAQ |

---

## 7. Accountability Frameworks

### Accountability in Practice

$$\text{Accountability} = \text{Answerability} + \text{Liability} + \text{Enforceability}$$

**Answerability**: Obligation to inform and explain decisions to affected parties.

**Liability**: Legal responsibility for harms caused by AI systems. Current frameworks:

```text
Product Liability:
  - Strict liability: manufacturer liable regardless of fault
  - Negligence: liable if failed reasonable standard of care
  - Application to AI: EU product liability directive extends to software/AI

Algorithmic Liability:
  - Who is liable when an AI system causes harm?
    - Developer: built the model, chose the architecture
    - Deployer: chose to deploy in this context
    - Operator: configured and operated the system
    - Data provider: supplied biased training data

  - Answer depends on jurisdiction and specific harm
  - EU AI Act assigns obligations to "providers" and "deployers"
  - Increasing trend toward shared/proportionate liability
```

**Enforceability**: Mechanisms to impose consequences:
- Regulatory fines (EU AI Act: up to 7% global revenue for prohibited practices)
- Civil litigation (affected individuals suing for damages)
- Market consequences (reputation, customer trust)
- Professional consequences (individual accountability)

### Algorithmic Accountability in Practice

```text
NYC Local Law 144 (2023) -- Automated Employment Decision Tools:
  Requirements:
    - Annual bias audit by independent auditor
    - Publish audit results on website
    - Notify candidates that AEDT is being used
    - Allow candidates to request alternative process

  Metrics Required:
    - Selection rate for each demographic group
    - Impact ratio (disparate impact analysis)
    - Intersectional analysis (race x gender)

  Significance:
    - First US law requiring AI bias audits
    - Narrow scope (hiring tools only) but precedent-setting
    - Enforcement by NYC DCWP (fines: $500-$1,500 per violation)
```

---

## 8. Ethical AI in Practice

### Common Ethical Failures

```text
Failure Mode               Example                     Root Cause
Proxy discrimination        Zip code proxies for race    Unexamined feature correlations
Automation bias              Radiologist accepts AI error Over-trust in AI outputs
Feedback loops               Predictive policing bias     Self-reinforcing data cycles
Consent laundering           "Agree to Terms" for AI use  Meaningless consent mechanisms
Ethics washing               AI ethics board with no      Governance theater
                             authority
Fairwashing                  Cherry-picked fairness       Selective metric reporting
                             metric looks good
Performative transparency    Model card with no useful    Documentation without substance
                             information
Poverty of imagination       "We didn't think of that     Homogeneous team, no stakeholder
                             use case"                    consultation
```

### Organizational Ethics Maturity

```text
Level 1: Reactive
  "We fix ethical issues when they become PR problems"
  - No proactive ethics process
  - Ethics discussed only after incidents
  - No dedicated ethics resources

Level 2: Compliance
  "We check the regulatory boxes"
  - Ethics reduced to legal compliance
  - Minimum viable ethics (bias audit as checkbox)
  - Ethics as legal risk mitigation

Level 3: Principled
  "We have ethics principles and follow them"
  - Published AI ethics principles
  - Ethics review process for new AI projects
  - Training and awareness for AI practitioners
  - Ethics board with advisory authority

Level 4: Embedded
  "Ethics is part of how we build AI"
  - Ethics integrated into development lifecycle
  - Diverse teams with ethicist participation
  - Stakeholder consultation standard practice
  - Proactive fairness testing and monitoring
  - Ethics board with binding authority

Level 5: Transformative
  "We advance the field of AI ethics"
  - Original research in AI ethics
  - Open-source fairness tools and methodologies
  - Industry leadership in responsible AI standards
  - Active community engagement and transparency
  - Ethics as innovation driver
```

---

## 9. The Value Alignment Problem

### Specification, Robustness, and Assurance

The value alignment problem has three components (Amodei et al., 2016):

$$\text{Alignment} = \text{Specification} \times \text{Robustness} \times \text{Assurance}$$

1. **Specification**: Did we correctly specify what we want the AI to do?
   - Reward hacking: AI optimizes the metric but not the intent
   - Goodhart's Law: "When a measure becomes a target, it ceases to be a good measure"
   - Example: AI trained to maximize engagement discovers that outrage maximizes clicks

2. **Robustness**: Does the AI behave as specified even in novel situations?
   - Distributional shift: AI encounters inputs unlike training data
   - Adversarial inputs: deliberately crafted inputs that cause failures
   - Edge cases: rare but consequential scenarios

3. **Assurance**: Can we verify that the AI is aligned?
   - Interpretability: can we understand what the AI is doing?
   - Monitoring: can we detect misalignment in deployment?
   - Correction: can we fix misalignment when detected?

### The Alignment Tax

$$\text{Alignment Tax} = \text{Cost}(\text{Aligned System}) - \text{Cost}(\text{Unaligned System})$$

The alignment tax includes:
- Development cost: bias testing, fairness constraints, documentation
- Performance cost: fairness constraints may reduce overall accuracy
- Speed cost: ethics review, stakeholder consultation add time
- Opportunity cost: prohibited use cases, restricted deployment

The goal of AI ethics research is to minimize the alignment tax -- making it cheap and easy to build fair, transparent, and safe AI systems so that there is no competitive disadvantage to doing the right thing.

---

## See Also

- AI Governance
- Privacy Regulations
- Security Awareness

## References

- Chouldechova, A. (2017): Fair Prediction with Disparate Impact: A Study of Bias in Recidivism Prediction Instruments
- Kleinberg, J., Mullainathan, S., Raghavan, M. (2016): Inherent Trade-Offs in the Fair Determination of Risk Scores
- Dwork, C. et al. (2012): Fairness Through Awareness
- Kusner, M. et al. (2017): Counterfactual Fairness
- Rudin, C. (2019): Stop Explaining Black Box Machine Learning Models for High Stakes Decisions
- Rawls, J. (1971): *A Theory of Justice*
- Hebert-Johnson, U. et al. (2018): Multicalibration: Calibration for the (Computationally-Identifiable) Masses
- Amodei, D. et al. (2016): Concrete Problems in AI Safety
- Solove, D. (2006): A Taxonomy of Privacy
- Nissenbaum, H. (2004): Privacy as Contextual Integrity
