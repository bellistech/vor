# The Mathematics of GDPR — Privacy Metrics, Anonymization Theory, and Risk Quantification

> *Behind the legal text of the GDPR lies a rich mathematical framework: differential privacy provides formal guarantees for anonymization, information theory quantifies re-identification risk, Bayesian models assess breach severity, and decision theory optimizes the balance between data utility and privacy protection. These mathematical tools transform legal obligations into measurable, auditable properties.*

---

## 1. Differential Privacy (Information Theory)
### The Problem
GDPR's Recital 26 states that anonymization must be irreversible. Differential privacy provides a mathematical guarantee that the inclusion or exclusion of any individual's data does not significantly affect query results.

### The Formula
A randomized mechanism $\mathcal{M}$ satisfies $(\epsilon, \delta)$-differential privacy if for all datasets $D_1, D_2$ differing in one record, and all subsets $S$ of outputs:

$$P[\mathcal{M}(D_1) \in S] \leq e^{\epsilon} \cdot P[\mathcal{M}(D_2) \in S] + \delta$$

For the Laplace mechanism on a function $f$ with sensitivity $\Delta f$:

$$\mathcal{M}(D) = f(D) + \text{Lap}\left(\frac{\Delta f}{\epsilon}\right)$$

Global sensitivity:

$$\Delta f = \max_{D_1, D_2} |f(D_1) - f(D_2)|$$

### Worked Examples
**Example**: Publishing average salary from a dataset of 1,000 employees. Salaries range from 30K to 200K.

Global sensitivity for the mean: $\Delta f = \frac{200{,}000 - 30{,}000}{1{,}000} = 170$

For $\epsilon = 0.1$ (strong privacy):

$$\text{Noise scale} = \frac{170}{0.1} = 1{,}700$$

True mean: 75,000. Published result: $75{,}000 \pm \text{Lap}(1700)$.

The standard deviation of Laplace noise is $\sqrt{2} \times 1700 \approx 2{,}404$.

95% of the time, the noise is within $\pm 2 \times 2{,}404 = \pm 4{,}808$, so accuracy is reasonable for this population size.

## 2. K-Anonymity and L-Diversity (Set Theory)
### The Problem
When releasing datasets, quasi-identifiers (age, zip code, gender) can be combined to re-identify individuals. K-anonymity ensures each combination of quasi-identifiers appears at least $k$ times.

### The Formula
A dataset satisfies $k$-anonymity if every equivalence class of quasi-identifier values contains at least $k$ records:

$$\forall q \in Q: |[q]_{\sim}| \geq k$$

$l$-diversity extends this by requiring each equivalence class to contain at least $l$ distinct values of the sensitive attribute:

$$\forall q \in Q: |\{s : (q, s) \in D\}| \geq l$$

$t$-closeness further requires that the distribution of the sensitive attribute in each class is within distance $t$ of the overall distribution:

$$\forall q \in Q: d(P_{[q]}, P_D) \leq t$$

Using Earth Mover's Distance:

$$EMD(P, Q) = \frac{1}{m-1} \sum_{i=1}^{m} \left| \sum_{j=1}^{i} (p_j - q_j) \right|$$

### Worked Examples
**Example**: A medical dataset with quasi-identifiers (age, zip):

| Age | Zip | Disease |
|-----|-----|---------|
| 25 | 10001 | Flu |
| 27 | 10001 | Diabetes |
| 26 | 10001 | Flu |

This group has $k = 3$ (three records share similar quasi-identifiers).

For $l$-diversity: diseases = {Flu, Diabetes}, so $l = 2$. The group is 2-diverse.

If we need 3-diversity but only have 2 distinct diseases, we must generalize further (e.g., merge with adjacent zip code groups).

## 3. Re-identification Risk Assessment (Bayesian Inference)
### The Problem
GDPR controllers must assess whether data is truly anonymized or merely pseudonymized. Bayesian analysis quantifies the probability of successfully re-identifying an individual.

### The Formula
Prior probability of re-identification for individual $i$ in dataset of $N$ records:

$$P(i) = \frac{1}{N}$$

After observing quasi-identifiers $q$ that match $k$ records:

$$P(i \mid q) = \frac{P(q \mid i) \cdot P(i)}{P(q)} = \frac{1}{k}$$

Prosecutor risk (targeted attack on known individual):

$$R_{prosecutor} = \max_{q} \frac{1}{|[q]_{\sim}|}$$

Journalist risk (any individual in the dataset):

$$R_{journalist} = \frac{1}{N} \sum_{q \in Q} \frac{|[q]_{\sim}|}{|[q]_{\sim}|} \cdot \mathbf{1}_{|[q]_{\sim}| < k}$$

Marketer risk (overall expected success rate):

$$R_{marketer} = \frac{1}{N} \sum_{i=1}^{N} \frac{1}{|[q_i]_{\sim}|}$$

### Worked Examples
**Example**: Dataset of 10,000 individuals. After generalization:
- 200 equivalence classes
- Smallest class has 5 records, largest has 150

Prosecutor risk: $R_{prosecutor} = 1/5 = 0.20$ (20%)

Marketer risk: $R_{marketer} = \frac{1}{10{,}000} \sum_{c=1}^{200} |c_j| \times \frac{1}{|c_j|} = \frac{200}{10{,}000} = 0.02$ (2%)

For GDPR compliance, the Article 29 Working Party suggests prosecutor risk should be below 0.09 (less than 9%).

## 4. Breach Severity Scoring (Multi-Factor Risk Model)
### The Problem
Article 33 requires notifying the supervisory authority unless the breach is "unlikely to result in a risk to the rights and freedoms of natural persons." A quantitative severity model informs this decision.

### The Formula
ENISA breach severity score:

$$S = DPC \times EI \times CB$$

Where:
- $DPC$ = Data Processing Context (1-4 scale based on data type)
- $EI$ = Ease of Identification (1-4 scale)
- $CB$ = Circumstances of Breach (1-4 scale based on malicious intent, volume)

Overall severity:

$$\text{Severity} = \begin{cases} \text{Low} & S \leq 2 \\ \text{Medium} & 2 < S \leq 3 \\ \text{High} & 3 < S \leq 4 \\ \text{Very High} & S > 4 \end{cases}$$

Alternatively, using the WP29 methodology:

$$S_{total} = \sum_{i=1}^{n} w_i \cdot f_i$$

Where $f_i$ are factors: volume, sensitivity, ease of identification, severity of consequences, and special characteristics of individuals.

### Worked Examples
**Example**: Breach of 5,000 health records (names + diagnoses), data posted publicly.

- $DPC = 4$ (special category health data)
- $EI = 4$ (names directly identify individuals)
- $CB = 4$ (malicious, large scale, public exposure)

$$S = \sqrt[3]{4 \times 4 \times 4} = \sqrt[3]{64} = 4 \implies \text{Very High}$$

Notification required to both DPA (within 72 hours) and all affected data subjects (without undue delay).

## 5. Data Minimization Optimization (Information Theory)
### The Problem
Article 5(1)(c) requires that personal data be "adequate, relevant and limited to what is necessary." Information theory helps quantify the minimum data needed for a given processing purpose.

### The Formula
Mutual information between collected data $X$ and processing purpose $Y$:

$$I(X; Y) = \sum_{x,y} p(x,y) \log \frac{p(x,y)}{p(x)p(y)}$$

Data minimization is achieved when we find the minimal sufficient statistic $T(X)$ such that:

$$I(T(X); Y) = I(X; Y) \quad \text{and} \quad H(T(X)) \leq H(X)$$

The data minimization ratio:

$$\eta = \frac{H(T(X))}{H(X)} = \frac{\text{Entropy of minimal data}}{\text{Entropy of collected data}}$$

Values closer to 0 indicate greater minimization.

### Worked Examples
**Example**: An e-commerce site collects full date of birth but only needs age verification (over 18).

- Full DOB entropy: $H(DOB) \approx \log_2(365.25 \times 80) \approx 14.8$ bits
- Age verification: $H(\text{over18}) = -0.8\log_2(0.8) - 0.2\log_2(0.2) \approx 0.72$ bits

$$\eta = \frac{0.72}{14.8} = 0.049$$

Collecting full DOB carries 20x more information than needed. A boolean "over 18" field satisfies data minimization.

## 6. Consent Validity Modeling (Temporal Logic)
### The Problem
GDPR consent must be freely given, specific, informed, and unambiguous. Consent validity can be modeled as a temporal property that must hold continuously.

### The Formula
Consent state as a function of time:

$$C(t) = \begin{cases} 1 & \text{if } \exists t_g \leq t : \text{granted}(t_g) \wedge \nexists t_w \in [t_g, t] : \text{withdrawn}(t_w) \\ & \wedge \neg \text{expired}(t_g, t) \wedge \text{valid\_notice}(t_g, t) \\ 0 & \text{otherwise} \end{cases}$$

Consent freshness (probability that consent still reflects user intent):

$$F(t) = e^{-\lambda(t - t_g)}$$

Where $\lambda$ is the decay rate dependent on context changes (policy updates, service changes).

### Worked Examples
**Example**: Consent granted at $t_g = 0$, privacy policy updated at $t_p = 6$ months. Decay rate $\lambda = 0.05$/month under stable conditions, $\lambda = 0.3$/month after policy change.

At month 5 (before policy change):
$$F(5) = e^{-0.05 \times 5} = e^{-0.25} = 0.778$$

At month 8 (2 months after policy change):
$$F(8) = e^{-0.05 \times 6} \times e^{-0.3 \times 2} = 0.741 \times 0.549 = 0.407$$

Consent freshness has dropped below 50%, suggesting re-consent should be sought.

## Prerequisites
- differential-privacy, information-theory, bayesian-inference, set-theory, optimization, temporal-logic, entropy
