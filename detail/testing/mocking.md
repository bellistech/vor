# The Mathematics of Mocking — Test Double Theory and Coupling Analysis

> *When we replace real dependencies with test doubles, we create a divergent model. This explores the formal taxonomy, the coupling mathematics that determine when mocks help versus when they create false confidence, and the probability theory behind mock-reality divergence.*

---

## 1. Test Double Taxonomy Formalization (Type Theory)

### The Problem

The terms "mock", "stub", "spy", and "fake" are used loosely and often interchangeably. A formal taxonomy helps us reason about what guarantees each type provides and what it sacrifices.

### The Formula

Define a dependency $D$ with interface $I = \{m_1, m_2, \ldots, m_k\}$ (set of methods). A test double $D'$ is any implementation of $I$ used in place of $D$ during testing.

**Behavioral fidelity** of a test double:

$$F(D') = \frac{|\{(i, o) : D'(i) = D(i), i \in I_{test}\}|}{|I_{test}|}$$

where $I_{test}$ is the set of inputs exercised during testing.

Each test double type has a characteristic fidelity profile:

| Type | $F(D')$ for tested inputs | $F(D')$ for untested inputs | Verification |
|------|---------------------------|----------------------------|-------------|
| Dummy | Undefined (never called) | Undefined | None |
| Stub | 1.0 (hardcoded correct answers) | 0 (no implementation) | None |
| Spy | $\leq 1.0$ (may delegate) | $\leq 1.0$ | Post-hoc call recording |
| Mock | 1.0 (programmed answers) | 0 | Pre-programmed expectations |
| Fake | $\approx 1.0$ (simplified but correct) | $> 0$ (has real logic) | None (tested independently) |

The key insight: only fakes provide non-zero fidelity for untested inputs.

### Worked Examples

**Example**: An `EmailSender` interface with `Send(to, subject, body) error`.

- **Stub**: Always returns `nil`. Fidelity = 1.0 for "email succeeds" tests, 0.0 for "what happens when the SMTP server is down?"
- **Fake**: In-memory mailbox that stores messages. Fidelity $\approx 1.0$ for all input combinations, but does not test actual SMTP behavior.
- **Mock** (gomock): `EXPECT().Send("alice@test.com", gomock.Any(), gomock.Any()).Return(nil)`. Fidelity = 1.0 for exactly `alice@test.com`, 0.0 for any other recipient.

## 2. Coupling Coefficient Metrics (Graph Theory)

### The Problem

Mocking introduces coupling between tests and implementation. If the implementation changes method signatures or call patterns, mock-heavy tests break even when behavior is preserved. How do we quantify this coupling?

### The Formula

**Afferent coupling** $C_a(M)$: number of test files that depend on mock $M$.

**Efferent coupling** $C_e(T)$: number of mocks used by test $T$.

**Instability** of a test:

$$I(T) = \frac{C_e(T)}{C_a(T) + C_e(T)}$$

where $I = 0$ means maximally stable (no outgoing dependencies) and $I = 1$ means maximally unstable.

**Mock coupling ratio** for a test suite:

$$R_{mock} = \frac{\sum_{T} C_e(T)}{|T| \cdot |D|}$$

where $|D|$ is the total number of dependencies. $R_{mock} \to 1$ means every test mocks every dependency (extremely coupled).

**Brittleness index**: probability that a non-behavioral change breaks at least one test:

$$B = 1 - \prod_{T} (1 - p_T)$$

where $p_T$ is the probability that test $T$ breaks due to a refactoring. For mock-heavy tests, $p_T$ is significantly higher than for tests using fakes or real implementations.

### Worked Examples

**Example**: A `UserService` with 3 dependencies (Store, EmailSender, Logger). Test suite has 10 tests.

Scenario A — heavy mocking: each test mocks all 3 dependencies.
$$R_{mock} = \frac{10 \times 3}{10 \times 3} = 1.0$$

Scenario B — minimal mocking: 5 tests mock only Store, 3 mock only EmailSender, 2 use all real.
$$R_{mock} = \frac{5 \times 1 + 3 \times 1 + 2 \times 0}{10 \times 3} = \frac{8}{30} = 0.27$$

Scenario B is 3.7x less coupled and will survive more refactorings.

## 3. Contract Testing Mathematics (Specification Theory)

### The Problem

When producer $P$ and consumer $C$ communicate through an interface, mocks in $C$'s tests may drift from $P$'s actual behavior. Contract testing ensures the mock and the real implementation satisfy the same specification.

### The Formula

A **contract** $K$ is a set of input-output pairs: $K = \{(i_1, o_1), (i_2, o_2), \ldots, (i_n, o_n)\}$.

**Contract satisfaction**: implementation $D$ satisfies contract $K$ iff:

$$\forall (i, o) \in K : D(i) = o$$

**Mock-reality divergence** $\Delta$:

$$\Delta(D', D, K) = \frac{|\{(i, o) \in K : D'(i) \neq D(i)\}|}{|K|}$$

A contract test suite ensures $\Delta = 0$ by running $K$ against both $D$ (provider verification) and $D'$ (consumer stub verification).

**Divergence growth rate**: without contract tests, the probability of divergence after $n$ provider releases:

$$P(\Delta > 0 \text{ after } n \text{ releases}) = 1 - \prod_{j=1}^{n} (1 - p_j)$$

where $p_j$ is the probability of a breaking change in release $j$. For typical APIs with $p_j \approx 0.05$:

| Releases | $P(\Delta > 0)$ |
|----------|------------------|
| 5        | 22.6%            |
| 10       | 40.1%            |
| 20       | 64.2%            |
| 50       | 92.3%            |

### Worked Examples

**Example**: Consumer mocks an API that returns `{"user": {"id": "123"}}`. After release 15, the provider changes to `{"user": {"uuid": "123"}}`.

Without contract tests: $P(\text{undetected divergence}) \approx 53.7\%$ (based on $p_j = 0.05$, $n = 15$).

With contract tests: the shared contract $K$ includes the response schema, so the provider verification test fails at release 15. $\Delta$ is detected immediately.

## 4. Mock-Reality Divergence Probability (Information Theory)

### The Problem

Every mock encodes assumptions about the real dependency. Over time, these assumptions become stale. What is the expected information loss from using mocks instead of real implementations?

### The Formula

Model the real dependency's behavior as a probability distribution $P(O|I)$ over outputs given inputs. The mock's behavior is a simplified distribution $Q(O|I)$.

**KL divergence** (information loss from using mock $Q$ instead of real $P$):

$$D_{KL}(P \| Q) = \sum_{o \in O} P(o|i) \ln \frac{P(o|i)}{Q(o|i)}$$

For a stub that returns a constant $c$: $Q(o|i) = \mathbf{1}[o = c]$.

If the real system has $m$ possible outcomes with probability $p_k$ each:

$$D_{KL} = \sum_{k=1}^{m} p_k \ln \frac{p_k}{\mathbf{1}[o_k = c]} = -\ln p_c + H(P) + \sum_{k \neq c} p_k \cdot \infty$$

The KL divergence is **infinite** whenever the stub returns a value that has zero probability in the real system, or when the real system can return values the stub never produces. This is the formal reason why stubs that only model the "happy path" provide zero information about error handling.

**Practical approximation** — **behavioral coverage** of mock $D'$:

$$B_{cov}(D') = \frac{|\text{output classes modeled by } D'|}{|\text{output classes of real } D|}$$

For a dependency with 4 output classes (success, timeout, auth error, server error), a stub that only returns success has $B_{cov} = 0.25$.

### Worked Examples

**Example**: HTTP client mock for an external API.

Real API behavior distribution:
- 200 OK: 95%
- 429 Rate Limited: 3%
- 500 Server Error: 1.5%
- Timeout: 0.5%

Mock returning only 200: $B_{cov} = 1/4 = 25\%$.

Mock returning 200 and 500: $B_{cov} = 2/4 = 50\%$.

Mock with all four behaviors configured: $B_{cov} = 4/4 = 100\%$, but the probability weights still differ (the mock might exercise timeout 25% of the time vs 0.5% in reality).

## Prerequisites

- Interface/polymorphism concepts (Go interfaces, duck typing)
- Basic probability theory (Bernoulli trials, conditional probability)
- Information theory (KL divergence, entropy)
- Graph theory (coupling, dependency graphs)

## Complexity

| Analysis | Time Complexity | Space Complexity |
|----------|----------------|-----------------|
| Contract verification ($K$ pairs) | $O(|K| \cdot T_{run})$ | $O(|K|)$ |
| Coupling coefficient computation | $O(|T| \cdot |D|)$ | $O(|T| + |D|)$ |
| Divergence detection (per release) | $O(|K|)$ | $O(1)$ |
| Mock generation (reflection-based) | $O(|I| \cdot |M|)$ | $O(|I|)$ |

Where: $|T|$ = test count, $|D|$ = dependency count, $|K|$ = contract size, $|I|$ = interface method count, $|M|$ = method parameter count.
