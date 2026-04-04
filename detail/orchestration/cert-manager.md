# The Mathematics of cert-manager — Certificate Lifecycle as Renewal Theory

> *Every certificate is a ticking clock. The art is in knowing when to wind it before it stops.*

---

## 1. Renewal Scheduling (Stochastic Processes)

### The Problem

A certificate has a finite validity window. Renewal must occur before expiry, but not so early that it wastes resources. With hundreds of certificates, renewal storms can overwhelm ACME providers. How do we model optimal renewal timing?

### The Formula

The renewal window is defined by the relationship between certificate duration and the `renewBefore` threshold. Let $T_d$ be the duration and $T_r$ be the renew-before offset:

$$t_{\text{renew}} = t_{\text{issue}} + T_d - T_r$$

The probability that a renewal attempt succeeds before expiry, given a per-attempt success rate $p$ and retry interval $\Delta t$:

$$P(\text{renewed}) = 1 - (1 - p)^{\lfloor T_r / \Delta t \rfloor}$$

For $n$ independent certificates, the expected number of concurrent renewals at any time $t$ follows a Poisson process with rate:

$$\lambda = \frac{n}{T_d}$$

### Worked Examples

**Example 1**: A certificate valid for 90 days with `renewBefore: 30d`. The controller retries every 1 hour with 95% success rate per attempt.

$$\text{Attempts} = \lfloor 30 \times 24 / 1 \rfloor = 720$$

$$P(\text{renewed}) = 1 - (1 - 0.95)^{720} = 1 - 0.05^{720} \approx 1.0$$

Effectively certain to renew. Even at 50% per-attempt success:

$$P = 1 - 0.5^{720} \approx 1.0$$

**Example 2**: An organization manages 500 certificates, all 90-day duration. Expected concurrent renewal rate:

$$\lambda = \frac{500}{90} \approx 5.56 \text{ renewals/day}$$

The probability of a renewal storm (more than 20 in one day):

$$P(X > 20) = 1 - \sum_{k=0}^{20} \frac{e^{-5.56} \cdot 5.56^k}{k!} \approx 0.00001$$

---

## 2. ACME Challenge Verification (Graph Reachability)

### The Problem

HTTP01 challenges require the ACME server to reach a specific path on the target domain. DNS01 challenges require a TXT record to propagate through the DNS hierarchy. Both are reachability problems in directed graphs.

### The Formula

For DNS propagation, model the DNS hierarchy as a directed acyclic graph $G = (V, E)$ where each nameserver is a vertex. The propagation delay through a path of $k$ hops, each with TTL $\tau_i$:

$$T_{\text{propagation}} = \sum_{i=1}^{k} \tau_i$$

The worst-case propagation across all paths from authoritative server $s$ to resolver $r$:

$$T_{\text{max}} = \max_{P \in \text{paths}(s, r)} \sum_{(u,v) \in P} \tau_{(u,v)}$$

### Worked Examples

**Example 1**: A DNS01 challenge traverses 3 nameserver hops with TTLs of 300s, 60s, and 30s:

$$T_{\text{propagation}} = 300 + 60 + 30 = 390 \text{ seconds}$$

cert-manager's default DNS01 check interval is 10s with a propagation timeout of 60s, which may be insufficient. Setting `--dns01-recursive-nameservers` to authoritative servers reduces $k$ to 1.

**Example 2**: For HTTP01, the challenge reachability depends on ingress controller routing. The path must satisfy:

$$\text{ACME Server} \xrightarrow{\text{TCP/443}} \text{LB} \xrightarrow{\text{route}} \text{Ingress} \xrightarrow{\text{path}} \text{cert-manager solver pod}$$

Each hop has a probability of misconfiguration $q_i$. The end-to-end success probability:

$$P(\text{reachable}) = \prod_{i=1}^{n} (1 - q_i)$$

With 4 hops each at 1% misconfiguration risk: $P = 0.99^4 \approx 0.961$.

---

## 3. Certificate Chain Trust (Lattice Theory)

### The Problem

X.509 certificates form a trust hierarchy. A leaf certificate is valid only if a chain of trust can be constructed to a trusted root. This is a partial order with lattice properties.

### The Formula

Define a trust relation $\preceq$ on certificates where $a \preceq b$ means $a$ is signed by $b$. The trust chain is a totally ordered subset (chain) in the partial order:

$$\text{leaf} \preceq \text{intermediate}_1 \preceq \cdots \preceq \text{intermediate}_k \preceq \text{root}$$

The chain is valid if and only if for each adjacent pair $(c_i, c_{i+1})$:

$$\text{Verify}(\text{sig}(c_i), \text{pubkey}(c_{i+1})) = \text{true} \land t \in [\text{notBefore}(c_i), \text{notAfter}(c_i)]$$

The total validity window of the chain is the intersection:

$$W_{\text{chain}} = \bigcap_{i=0}^{k+1} [\text{notBefore}(c_i), \text{notAfter}(c_i)]$$

### Worked Examples

**Example 1**: A chain with three certificates:
- Leaf: valid 2024-01-01 to 2024-04-01
- Intermediate: valid 2023-01-01 to 2028-01-01
- Root: valid 2020-01-01 to 2035-01-01

$$W_{\text{chain}} = [2024\text{-}01\text{-}01, \ 2024\text{-}04\text{-}01]$$

The chain validity is bounded by the shortest-lived certificate (the leaf).

**Example 2**: When a cross-signed intermediate expires, the chain breaks even if the leaf is fresh. If intermediate $c_1$ expires at $t_e$ and the leaf was issued at $t_e - 1\text{d}$:

$$W_{\text{chain}} = [t_e - 1\text{d}, \ t_e]$$

The chain becomes invalid in 24 hours despite the leaf having 89 days remaining.

---

## 4. Rate Limiting (Token Bucket Model)

### The Problem

Let's Encrypt enforces rate limits: 50 certificates per registered domain per week, 5 duplicate certificates per week. How many certificates can an organization provision within these constraints?

### The Formula

Model rate limits as a token bucket with capacity $B$ and refill rate $r$ tokens per unit time. The maximum burst is $B$, and sustained throughput is $r$:

$$\text{tokens}(t) = \min(B, \ \text{tokens}(t - 1) + r \cdot \Delta t - \text{consumed})$$

For Let's Encrypt with $B = 50$ per domain per week and $r = 50/7 \approx 7.14$ per day:

$$t_{\text{drain}} = \frac{B}{d - r}$$

where $d$ is the demand rate.

### Worked Examples

**Example 1**: An organization needs 200 subdomains under `example.com` provisioned. With a limit of 50/week:

$$t_{\text{provision}} = \lceil 200 / 50 \rceil = 4 \text{ weeks}$$

Using 4 registered domains with SANs splits the load:

$$t_{\text{provision}} = \lceil 200 / (50 \times 4) \rceil = 1 \text{ week}$$

**Example 2**: During a cluster migration, 50 certificates are requested simultaneously, exhausting the bucket. A failed renewal 3 days later finds:

$$\text{tokens}(3) = \min(50, \ 0 + 7.14 \times 3) = 21.43 \rightarrow 21 \text{ available}$$

---

## 5. Key Rotation (Information-Theoretic Security)

### The Problem

Private keys have a security half-life. The longer a key exists, the higher the probability of compromise through side-channel leaks, memory dumps, or backup exposure.

### The Formula

Model key compromise as an exponential distribution with rate $\mu$ (compromises per unit time). The survival probability (key remains secure) after time $t$:

$$S(t) = e^{-\mu t}$$

The expected time to compromise:

$$E[T] = \frac{1}{\mu}$$

For $n$ independent keys, the probability that at least one is compromised:

$$P(\text{any compromised}) = 1 - e^{-n\mu t}$$

### Worked Examples

**Example 1**: Assume $\mu = 0.001$ per day (mean time to compromise = 1000 days). For a 90-day certificate:

$$S(90) = e^{-0.001 \times 90} = e^{-0.09} \approx 0.914$$

An 8.6% chance of compromise over the certificate lifetime. With 30-day rotation:

$$S(30) = e^{-0.03} \approx 0.970$$

Reducing exposure by rotating keys with each renewal cuts risk by a factor of 3.

**Example 2**: An organization with 500 certificates, each rotated every 90 days:

$$P(\text{any compromised}) = 1 - e^{-500 \times 0.001 \times 90} = 1 - e^{-45} \approx 1.0$$

Near-certain compromise of at least one key. Reducing to 30-day rotation:

$$P = 1 - e^{-15} \approx 1.0$$

Still high, but the blast radius per compromised key is 3x smaller.

---

## Prerequisites

- X.509 certificate structure (subject, issuer, extensions, validity period)
- Public key cryptography (RSA, ECDSA, key pairs, digital signatures)
- ACME protocol flow (account registration, order, authorization, challenge, finalize)
- Kubernetes custom resource definitions and controller pattern
- DNS record types (A, CNAME, TXT) and resolution hierarchy
- Probability distributions (Poisson, exponential)
- Basic graph theory (directed acyclic graphs, reachability)
