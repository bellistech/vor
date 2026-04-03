# The Mathematics of PKI — Trust as a Directed Graph

> *Public Key Infrastructure is a system of trust relationships modeled as directed acyclic graphs, validated by cryptographic signature chains, and audited through Merkle tree transparency logs.*

---

## 1. Certificate Chain as Directed Graph

### Graph Model

A PKI certificate chain is a **directed acyclic graph** (DAG) where:

- **Vertices** $V$: certificates (Root CA, Intermediate CA, Leaf)
- **Edges** $E$: "signed by" relationships
- **Root** $r$: self-signed trust anchor ($r$ signs $r$)

$$G = (V, E) \quad \text{where} \quad (C_i, C_j) \in E \iff C_j.\text{key signed } C_i$$

### Chain Depth

$$d(C) = \text{length of path from } C \text{ to root}$$

| Certificate | Depth | Example |
|:---|:---:|:---|
| Root CA | 0 | DigiCert Global Root G2 |
| Intermediate CA | 1 | DigiCert SHA2 Extended Validation |
| Leaf (end-entity) | 2 | www.example.com |
| Cross-signed intermediate | varies | Let's Encrypt R3 (cross-signed by IdenTrust) |

### Maximum Path Length Constraint

The `pathLenConstraint` in Basic Constraints limits chain depth:

$$\text{pathLen}(C_i) \geq d(\text{leaf}) - d(C_i) - 1$$

If a CA has `pathLen=0`, it can only sign end-entity certificates (no sub-CAs).

---

## 2. Path Validation Algorithm (RFC 5280 Section 6)

### Formal Algorithm

Given a prospective chain $[C_0, C_1, \ldots, C_n]$ where $C_0$ is the leaf and $C_n$ is the trust anchor:

```
VALIDATE(chain):
  For i = n down to 0:
    1. Signature:     Verify(C_i.sig, C_{i+1}.pubkey) == true   [i < n]
    2. Validity:      C_i.notBefore <= now <= C_i.notAfter
    3. Name chain:    C_i.issuer == C_{i+1}.subject              [i < n]
    4. Revocation:    C_i not in CRL and OCSP(C_i) != "revoked"
    5. Key usage:     C_i.keyUsage includes required bits
    6. Constraints:   if i > 0: C_i.isCA == true
    7. Path length:   C_i.pathLen >= (remaining intermediates)
    8. Name constraints: C_i.subject within permitted subtrees
    9. Policy:        C_i.policies intersect required policies
  Return VALID if all checks pass
```

### Validation Complexity

$$T_{validate} = \sum_{i=0}^{n} \left[ T_{sig}(C_i) + T_{revocation}(C_i) \right]$$

| Operation | RSA-2048 | RSA-4096 | ECDSA-P256 |
|:---|:---:|:---:|:---:|
| Signature verify | 0.15 ms | 0.50 ms | 0.30 ms |
| CRL download | 50-500 ms | 50-500 ms | 50-500 ms |
| OCSP query | 20-200 ms | 20-200 ms | 20-200 ms |
| OCSP stapling | 0 ms (cached) | 0 ms (cached) | 0 ms (cached) |

Revocation checking dominates — OCSP stapling eliminates the network round-trip.

---

## 3. CRL vs OCSP — Revocation Checking

### CRL (Certificate Revocation List)

A CRL is a signed list of revoked serial numbers. Size grows linearly:

$$\text{CRL size} \approx 20 + 40n \text{ bytes}$$

Where $n$ = number of revoked certificates.

| Revoked Certs | CRL Size | Download Time (1 Mbps) |
|:---:|:---:|:---:|
| 100 | ~4 KB | 32 ms |
| 10,000 | ~400 KB | 3.2 s |
| 1,000,000 | ~40 MB | 320 s |

### OCSP (Online Certificate Status Protocol)

OCSP returns status for a single certificate:

$$\text{OCSP response size} \approx 500 \text{ bytes (fixed)}$$

### Comparison

| Property | CRL | OCSP | OCSP Stapling |
|:---|:---:|:---:|:---:|
| Freshness | Hours (cache) | Real-time | Seconds |
| Privacy | Client downloads all | CA sees queries | CA sees nothing |
| Size | $O(n)$ | $O(1)$ | $O(1)$ |
| Latency | High (first load) | Medium (per-cert) | Zero (pre-fetched) |
| Failure mode | Soft-fail (accept) | Soft-fail | Must-staple possible |

---

## 4. Certificate Transparency — Merkle Tree Audit

### The Problem

A CA could issue a fraudulent certificate. Certificate Transparency (CT, RFC 6962) makes all certificates publicly visible via append-only logs.

### Merkle Tree Structure

CT logs store certificates in a **Merkle hash tree**:

$$H(\text{leaf}) = \text{SHA-256}(0x00 \| \text{certificate data})$$
$$H(\text{node}) = \text{SHA-256}(0x01 \| H(\text{left}) \| H(\text{right}))$$

The **Merkle root** commits to all entries:

$$\text{root} = H(H(H(L_0, L_1), H(L_2, L_3)), H(H(L_4, L_5), H(L_6, L_7)))$$

### Proof Sizes

**Inclusion proof** (prove a certificate is in the log): $O(\log_2 n)$ hashes

| Log Entries ($n$) | Proof Size (hashes) | Proof Size (bytes) |
|:---:|:---:|:---:|
| $2^{20}$ (1M) | 20 | 640 |
| $2^{30}$ (1B) | 30 | 960 |
| $2^{40}$ (1T) | 40 | 1,280 |

**Consistency proof** (prove log $n$ is a prefix of log $m$): also $O(\log_2 n)$.

### Signed Certificate Timestamp (SCT)

When a certificate is submitted to a CT log, the log returns an SCT:

$$\text{SCT} = \text{Sign}_{log\_key}(\text{timestamp} \| \text{cert\_hash})$$

TLS servers embed SCTs in certificates or stapled OCSP responses. Browsers require 2-3 SCTs from independent logs.

---

## 5. Certificate Signing — RSA and ECDSA

### RSA Signature

$$\text{sig} = H(C)^d \pmod{n}$$

Verification:

$$H(C) \stackrel{?}{=} \text{sig}^e \pmod{n}$$

### ECDSA Signature (RFC 6979)

Given private key $d$, message hash $z = H(m)$, curve order $n$, generator $G$:

1. Choose random $k \in [1, n-1]$
2. Compute $(x_1, y_1) = k \cdot G$
3. $r = x_1 \pmod{n}$ (if $r = 0$, restart)
4. $s = k^{-1}(z + rd) \pmod{n}$ (if $s = 0$, restart)

Signature: $(r, s)$

Verification:
1. $w = s^{-1} \pmod{n}$
2. $u_1 = zw \pmod{n}$, $u_2 = rw \pmod{n}$
3. $(x_1, y_1) = u_1 G + u_2 Q$ where $Q = dG$ is the public key
4. Valid iff $r \equiv x_1 \pmod{n}$

### Signature Sizes

| Algorithm | Signature Size | Certificate Overhead |
|:---|:---:|:---:|
| RSA-2048 | 256 bytes | ~1,200 bytes total |
| RSA-4096 | 512 bytes | ~1,700 bytes total |
| ECDSA-P256 | 64 bytes | ~500 bytes total |
| Ed25519 | 64 bytes | ~400 bytes total |

---

## 6. Key Lifecycle Mathematics

### Certificate Validity Period

The optimal validity period balances security against operational cost:

$$P(\text{compromise during validity}) = 1 - (1 - p_{daily})^{T_{days}}$$

Where $p_{daily}$ is the daily probability of key compromise and $T_{days}$ is the validity period.

| Validity | $p_{daily} = 10^{-6}$ | $p_{daily} = 10^{-5}$ |
|:---:|:---:|:---:|
| 90 days | 0.009% | 0.09% |
| 1 year | 0.037% | 0.36% |
| 2 years | 0.073% | 0.73% |
| 5 years | 0.182% | 1.82% |

Let's Encrypt uses 90-day certificates. CAs are moving to shorter validity periods.

### Key Ceremony Quorum

Root CA key ceremonies use **Shamir's Secret Sharing**:

$$f(x) = a_0 + a_1 x + a_2 x^2 + \cdots + a_{k-1} x^{k-1} \pmod{p}$$

Where $a_0$ is the secret (root key material), and $k$ shares are needed to reconstruct.

A $(k, n)$ threshold scheme requires $k$ of $n$ shareholders:

$$\binom{n}{k} = \frac{n!}{k!(n-k)!} \text{ possible quorums}$$

| Scheme | Shareholders | Required | Possible Quorums |
|:---:|:---:|:---:|:---:|
| (3, 5) | 5 | 3 | 10 |
| (5, 8) | 8 | 5 | 56 |
| (3, 7) | 7 | 3 | 35 |

---

## 7. Trust Store Economics

### Scale of the PKI Ecosystem

| Trust Store | Root CAs | Intermediate CAs | Active Leaf Certs |
|:---|:---:|:---:|:---:|
| Mozilla NSS | ~150 | ~3,000 | ~400M |
| Chrome Root Store | ~150 | ~3,000 | ~400M |
| Apple | ~170 | ~3,500 | ~400M |

### Certificate Issuance Rate

Let's Encrypt alone issues ~3 million certificates per day. The CT logs grow by:

$$\text{Growth} \approx 5 \times 10^6 \text{ entries/day} \approx 1.8 \times 10^9 \text{ entries/year}$$

Merkle tree depth at 10 billion entries: $\lceil \log_2(10^{10}) \rceil = 34$ levels.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Certificate chain | Directed acyclic graph | Trust path validation |
| Path validation | Sequential algorithm | RFC 5280 Section 6 |
| CRL size | Linear growth $O(n)$ | Revocation list scaling |
| Merkle tree | Binary hash tree | Certificate Transparency |
| Inclusion proof | Logarithmic $O(\log n)$ | CT audit |
| Shamir sharing | Polynomial interpolation | Key ceremony quorum |
| Validity risk | Geometric probability | Certificate lifetime |

---

*Every browser on the planet executes this graph traversal and cryptographic verification hundreds of times per day — the invisible PKI backbone that makes internet trust possible.*
