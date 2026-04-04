# The Mathematics of HashiCorp Vault — Secrets Management and Cryptographic Operations

> *Vault is a secrets engine built on Shamir's Secret Sharing for unsealing, AES-GCM for encryption at rest, and a lease-based access model with time-bounded tokens. The mathematics span polynomial interpolation, authenticated encryption, and token entropy.*

---

## 1. Shamir's Secret Sharing — Unseal Process

### The Algorithm

Vault's unseal mechanism uses Shamir's Secret Sharing (SSS) over a finite field $GF(p)$:

1. Choose a random polynomial of degree $k-1$:

$$f(x) = a_0 + a_1 x + a_2 x^2 + \cdots + a_{k-1} x^{k-1} \pmod{p}$$

Where $a_0$ = master key (the secret).

2. Generate $n$ shares: $(i, f(i))$ for $i = 1, 2, \ldots, n$

3. Any $k$ shares can reconstruct $a_0$ via Lagrange interpolation.

### Lagrange Interpolation

Given $k$ shares $(x_1, y_1), \ldots, (x_k, y_k)$:

$$a_0 = f(0) = \sum_{i=1}^{k} y_i \prod_{j \neq i} \frac{x_j}{x_j - x_i} \pmod{p}$$

### Worked Example

$(3, 5)$ scheme: 3 of 5 shares needed.

Secret: $a_0 = 42$. Random coefficients: $a_1 = 7, a_2 = 3$.

$$f(x) = 42 + 7x + 3x^2$$

Shares:
| $x$ | $f(x)$ |
|:---:|:---:|
| 1 | 52 |
| 2 | 68 |
| 3 | 90 |
| 4 | 118 |
| 5 | 152 |

Reconstruct from shares (1,52), (3,90), (5,152):

$$f(0) = 52 \times \frac{3 \times 5}{(3-1)(5-1)} + 90 \times \frac{1 \times 5}{(1-3)(5-3)} + 152 \times \frac{1 \times 3}{(1-5)(3-5)}$$

$$= 52 \times \frac{15}{8} + 90 \times \frac{5}{-4} + 152 \times \frac{3}{8} = 97.5 - 112.5 + 57 = 42 \checkmark$$

### Security Properties

| Property | Guarantee |
|:---|:---|
| $k-1$ shares | Zero information about the secret |
| $k$ shares | Complete reconstruction |
| Any $k$ of $n$ | Same result (all combinations work) |

Quorum combinations: $\binom{n}{k}$

| Scheme | Quorums | Typical Use |
|:---:|:---:|:---|
| (3, 5) | 10 | Small team |
| (5, 10) | 252 | Enterprise |
| (3, 7) | 35 | Balanced |

---

## 2. Encryption at Rest — Barrier Encryption

### Encryption Layer

Vault encrypts all data before writing to the storage backend:

$$C = \text{AES-256-GCM}(K_{barrier}, \text{plaintext})$$

The barrier key $K_{barrier}$ is encrypted with the master key:

$$K_{barrier}^{enc} = \text{AES-256-GCM}(K_{master}, K_{barrier})$$

The master key is split via Shamir's SSS.

### Key Hierarchy

```
Unseal Key Shares (Shamir)
    |
    v (Lagrange interpolation)
Master Key (256 bits)
    |
    v (decrypt)
Barrier Key (256 bits)
    |
    v (encrypt/decrypt all data)
Storage Backend (Consul, etcd, etc.)
```

### Encryption Overhead

$$\text{Overhead per entry} = 12\text{ (nonce)} + 16\text{ (GCM tag)} + 4\text{ (version)} = 32 \text{ bytes}$$

| Secret Size | Encrypted Size | Overhead % |
|:---:|:---:|:---:|
| 32 bytes (password) | 64 bytes | 100% |
| 256 bytes (API key) | 288 bytes | 12.5% |
| 4096 bytes (cert) | 4128 bytes | 0.8% |

---

## 3. Transit Secrets Engine — Encryption as a Service

### Key Versioning

Transit supports key rotation with versioned keys:

$$\text{Ciphertext} = v{:}n{:}\text{Base64}(\text{AES-GCM}(K_n, \text{plaintext}))$$

Where $n$ is the key version.

### Convergent Encryption

For deduplication (deterministic encryption):

$$C = \text{AES-SIV}(K, \text{plaintext})$$

Same plaintext always produces same ciphertext (sacrificing IND-CPA security for deduplication).

### Minimum Encryption Version

$$\text{decrypt allowed} \iff n \geq n_{min\_decrypt}$$
$$\text{encrypt uses} \quad n = n_{latest}$$

This allows key rotation without re-encrypting old data immediately:

| Config | Old Ciphertext (v1) | New Ciphertext |
|:---|:---|:---|
| min_decrypt=1 | Decryptable | Uses latest version |
| min_decrypt=2 | Rejected | Uses latest version |

### Rewrap Operation

$$C_{new} = \text{Encrypt}(K_{new}, \text{Decrypt}(K_{old}, C_{old}))$$

Plaintext never leaves Vault during rewrap — the operation is atomic within the HSM/barrier.

---

## 4. Token Entropy and TTL

### Token Format

Vault tokens are high-entropy random strings:

$$H_{token} = \log_2(|\text{charset}|^{|\text{length}|})$$

| Token Type | Length | Charset | Entropy |
|:---|:---:|:---:|:---:|
| Service token | 26 chars | Base62 (62) | 155 bits |
| Batch token | ~200 chars | Base64 | ~1200 bits |

### Brute Force Token Search

$$T_{crack} = \frac{2^{H-1}}{R_{attempts}}$$

At $10^{12}$ attempts/second against a service token:

$$T = \frac{2^{154}}{10^{12}} = 2.3 \times 10^{34} \text{ seconds} = 7.3 \times 10^{26} \text{ years}$$

Token brute force is computationally infeasible.

### Token TTL and Renewal

$$T_{effective} = T_{initial} + \sum_{i=1}^{n} T_{renewal,i} \quad \text{subject to} \quad T_{effective} \leq T_{max}$$

| TTL | Renewals (each +1hr) | Max TTL | Effective Lifetime |
|:---:|:---:|:---:|:---:|
| 1 hour | 0 | 24 hours | 1 hour |
| 1 hour | 7 | 24 hours | 8 hours |
| 1 hour | 23 | 24 hours | 24 hours (max) |
| 1 hour | 100 | 24 hours | 24 hours (capped) |

---

## 5. Dynamic Secrets — Lease-Based Access

### Lease Model

Each dynamic secret has a finite lease:

$$\text{Lease}(s) = (s, t_{issue}, T_{ttl}, T_{max})$$

$$\text{Valid}(s, t) \iff t_{issue} \leq t \leq t_{issue} + T_{effective}$$

### Credential Rotation

With $n$ applications and lease TTL $T$:

$$\text{Active credentials at any time} \leq n$$
$$\text{Credentials issued per day} = n \times \frac{86400}{T}$$

| Applications | TTL | Daily Issuances | Active at Any Time |
|:---:|:---:|:---:|:---:|
| 10 | 1 hour | 240 | 10 |
| 10 | 8 hours | 30 | 10 |
| 100 | 1 hour | 2,400 | 100 |
| 100 | 24 hours | 100 | 100 |

### Blast Radius Reduction

If a credential is compromised at time $t$:

$$\text{Exposure window} = T_{ttl} - (t - t_{issue})$$

$$E[\text{exposure}] = \frac{T_{ttl}}{2}$$

| Static Credential TTL | Dynamic Credential TTL | Exposure Reduction |
|:---:|:---:|:---:|
| Forever | 1 hour | $\frac{\infty}{0.5 \text{ hr}} = \infty$ |
| 90 days | 1 hour | $\frac{1080}{0.5} = 2160\times$ |
| 90 days | 8 hours | $\frac{1080}{4} = 270\times$ |

---

## 6. Policy Evaluation

### Policy Model

Vault policies are path-based ACLs:

$$\text{policy}(p, \text{op}) = \begin{cases} \text{allow} & \text{if } \exists r \in \text{rules} : r.\text{path matches } p \land \text{op} \in r.\text{capabilities} \\ \text{deny} & \text{otherwise (default deny)} \end{cases}$$

### Capabilities

| Capability | HTTP Verb | Vault Operation |
|:---|:---|:---|
| create | POST | Write new secret |
| read | GET | Read secret |
| update | PUT/POST | Modify existing |
| delete | DELETE | Soft delete |
| list | LIST | Enumerate paths |
| sudo | any | Privileged paths |
| deny | any | Explicit denial |

### Policy Composition

When a token has multiple policies:

$$\text{effective}(p, \text{op}) = \begin{cases} \text{deny} & \text{if any policy explicitly denies} \\ \text{allow} & \text{if any policy allows AND none deny} \\ \text{deny} & \text{otherwise} \end{cases}$$

Deny always wins — the most restrictive interpretation.

---

## 7. Audit Logging

### Audit Entry Structure

Each Vault operation generates an audit log entry:

$$S_{entry} \approx 500\text{-}2000 \text{ bytes (JSON)}$$

### HMAC-Protected Audit

Sensitive values in audit logs are HMAC'd:

$$\text{audit\_value} = \text{HMAC-SHA256}(K_{audit}, \text{secret\_value})$$

This allows log correlation (same secret produces same HMAC) without revealing the secret.

$$P(\text{reverse HMAC}) = \frac{1}{2^{256}} \text{ per guess}$$

### Audit Volume

$$V_{daily} = R_{ops} \times S_{entry} \times 86400$$

| Operations/sec | Entry Size | Daily Volume |
|:---:|:---:|:---:|
| 10 | 1 KB | 864 MB |
| 100 | 1 KB | 8.64 GB |
| 1,000 | 1 KB | 86.4 GB |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Shamir SSS | Polynomial interpolation | Unseal key sharing |
| $\binom{n}{k}$ quorums | Combinatorics | Key ceremony |
| AES-256-GCM | AEAD cipher | Barrier encryption |
| Token entropy $2^{155}$ | Information theory | Token security |
| Lease $T_{ttl}/2$ | Expected value | Exposure window |
| Policy deny-wins | Boolean logic (meet) | Access control |
| HMAC audit | Keyed hash | Log privacy |

## Prerequisites

- polynomial interpolation (Shamir's secret sharing), symmetric encryption, HMAC

---

*Vault transforms secret management from a static credential problem into a mathematically bounded lease system — Shamir's polynomial splitting protects the master key, AES-GCM protects data at rest, and time-bounded leases ensure that compromised credentials automatically expire.*
