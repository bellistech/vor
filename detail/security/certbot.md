# The Mathematics of Certbot — Automated Certificate Lifecycle

> *Certbot automates the ACME protocol (RFC 8555) to obtain, renew, and manage TLS certificates from Let's Encrypt. The mathematics involve challenge-response proofs of domain control, certificate validity windows, and renewal scheduling to prevent expiration.*

---

## 1. ACME Protocol — Challenge Mathematics

### Challenge Types

ACME proves domain control through one of three challenge types:

| Challenge | Mechanism | Proof |
|:---|:---|:---|
| HTTP-01 | Place token at `/.well-known/acme-challenge/` | HTTP GET returns token |
| DNS-01 | Create `_acme-challenge` TXT record | DNS query returns hash |
| TLS-ALPN-01 | Serve self-signed cert with ALPN extension | TLS handshake proof |

### HTTP-01 Challenge

The authorization token is constructed as:

$$\text{keyAuth} = \text{token} \| \text{"."} \| \text{thumbprint}(JWK)$$

Where the JWK thumbprint (RFC 7638) is:

$$\text{thumbprint} = \text{Base64url}(\text{SHA-256}(\text{canonical\_JSON}(\text{public\_key})))$$

### DNS-01 Challenge

The TXT record value:

$$\text{TXT value} = \text{Base64url}(\text{SHA-256}(\text{keyAuth}))$$

This is a 43-character string (256-bit hash in Base64url).

### Challenge Security

$$P(\text{forged challenge}) = \frac{1}{2^{256}} \approx 8.6 \times 10^{-78}$$

An attacker cannot predict the keyAuth without the account private key.

---

## 2. Certificate Validity and Renewal Timing

### Let's Encrypt Certificate Lifetime

$$T_{validity} = 90 \text{ days}$$

### Renewal Window

Certbot's default renewal check:

$$T_{renew} = T_{expiry} - 30 \text{ days}$$

This gives a 30-day window for renewal attempts before expiration.

### Renewal Scheduling

Certbot timer runs twice daily with random jitter:

$$t_{check} = t_{base} + \text{Uniform}(0, 43200) \text{ seconds}$$

The jitter prevents all Certbot instances from hitting Let's Encrypt simultaneously.

### Failure Tolerance

With 30 days and 2 checks/day:

$$\text{Attempts before expiration} = 30 \times 2 = 60$$

If each attempt has $p = 0.95$ probability of success:

$$P(\text{all fail}) = (1 - p)^{60} = 0.05^{60} = 8.7 \times 10^{-79}$$

Even with $p = 0.5$ (50% success rate): $P(\text{all fail}) = 0.5^{60} = 8.7 \times 10^{-19}$

The system is extremely robust to transient failures.

---

## 3. Rate Limits — Queuing Theory

### Let's Encrypt Rate Limits

| Limit | Value | Window |
|:---|:---:|:---:|
| Certificates per domain | 50 | 7 days |
| Duplicate certificates | 5 | 7 days |
| Failed validations | 5 | 1 hour |
| New orders | 300 | 3 hours |
| Accounts per IP | 10 | 3 hours |

### Capacity Planning

For $n$ subdomains under one registered domain:

$$\text{Time to certify all} = \left\lceil \frac{n}{50} \right\rceil \times 7 \text{ days}$$

| Subdomains | Batches | Calendar Time |
|:---:|:---:|:---:|
| 10 | 1 | 1 day |
| 50 | 1 | 1 day |
| 100 | 2 | 14 days |
| 500 | 10 | 70 days |

### SAN Certificate Optimization

A single certificate can have up to 100 Subject Alternative Names:

$$\text{Certificates needed} = \left\lceil \frac{n_{domains}}{100} \right\rceil$$

This reduces the rate limit impact by 100x.

---

## 4. Key Management

### Key Sizes and Types

| Key Type | Default | Security | CSR Size |
|:---|:---:|:---:|:---:|
| RSA 2048 | Yes (legacy) | 112-bit | ~1.2 KB |
| RSA 4096 | Optional | 128-bit | ~1.7 KB |
| ECDSA P-256 | Preferred | 128-bit | ~0.5 KB |
| ECDSA P-384 | Optional | 192-bit | ~0.6 KB |

### ECDSA Bandwidth Savings

Certificate chain size (leaf + intermediate):

$$\text{RSA-2048 chain} \approx 3.5 \text{ KB}$$
$$\text{ECDSA-P256 chain} \approx 1.8 \text{ KB}$$

Savings per TLS handshake: ~1.7 KB. At 10,000 new connections/second:

$$\text{Bandwidth saved} = 1.7 \text{ KB} \times 10{,}000 \text{/s} = 17 \text{ MB/s} = 136 \text{ Mbps}$$

### Key Rotation

Each renewal generates a new private key (Certbot default). The probability of key compromise over $n$ validity periods:

$$P(\text{compromise in any period}) = 1 - (1 - p)^n$$

Where $p$ is the per-period compromise probability.

With 90-day certificates and $p = 10^{-6}$: after 10 years ($n = 40$):

$$P = 1 - (1 - 10^{-6})^{40} = 4 \times 10^{-5} = 0.004\%$$

---

## 5. OCSP Stapling with Certbot

### OCSP Response Caching

$$T_{cache} = \min(T_{nextUpdate} - T_{thisUpdate}, T_{max\_cache})$$

Typical OCSP response validity: 7 days. Refresh interval:

$$T_{refresh} = \frac{T_{cache}}{2} = 3.5 \text{ days}$$

### Stapling Bandwidth Savings

Without stapling: each client queries OCSP responder:

$$\text{OCSP queries} = N_{connections} \times (1 - P_{client\_cache\_hit})$$

With stapling: server fetches once, serves to all:

$$\text{OCSP queries} = \frac{1}{T_{refresh}} = \frac{1}{302{,}400 \text{ s}} \approx 0.000003 \text{/s}$$

For a server with 1000 connections/second: from 1000 queries/s to 0.000003/s — a $3.3 \times 10^{8}$ reduction.

---

## 6. Wildcard Certificates

### Wildcard Scope

A wildcard certificate `*.example.com` covers:

$$\text{Matches} = \{x.\text{example.com} : x \in \text{single label}\}$$

It does NOT match:
- `example.com` (bare domain)
- `sub.sub.example.com` (multi-level)

### Wildcard vs Individual Certificate Count

| Subdomains | Individual Certs | Wildcards Needed |
|:---:|:---:|:---:|
| 10 (single level) | 10 | 1 |
| 10 (two levels) | 10 | 2 (*.x.com + *.y.x.com) |
| 100 (single level) | 100 | 1 |
| 100 (mixed levels) | 100 | varies |

### DNS-01 Requirement

Wildcards require DNS-01 challenge (cannot use HTTP-01):

$$\text{Wildcard} \implies \text{DNS-01 only}$$

This requires DNS API access — a larger attack surface than file-based HTTP-01.

---

## 7. Certificate Transparency Compliance

### SCT Requirements

Let's Encrypt submits to multiple CT logs:

$$n_{SCTs} \geq 2 \text{ (from independent logs)}$$

### CT Log Growth from Let's Encrypt

Let's Encrypt issues ~3 million certificates/day:

$$\text{CT entries/year} = 3 \times 10^6 \times 365 \approx 1.1 \times 10^9$$

Merkle tree depth for 1 billion entries: $\lceil \log_2(10^9) \rceil = 30$ levels.

Inclusion proof size: $30 \times 32 \text{ bytes} = 960 \text{ bytes}$.

---

## 8. Deployment Automation

### Renewal Hook Timing

$$T_{total} = T_{renew} + T_{deploy} + T_{reload}$$

| Component | Duration | Failure Impact |
|:---|:---:|:---|
| ACME order + challenge | 5-30 seconds | Retry next check |
| Certificate download | 1-5 seconds | Retry next check |
| Deploy hook (file copy) | <1 second | Old cert still valid |
| Service reload (nginx) | 1-5 seconds | Brief connection drop |

### Monitoring Formula

$$\text{Days until expiration} = \frac{T_{notAfter} - T_{now}}{86400}$$

Alert thresholds:

| Days Remaining | Severity | Action |
|:---:|:---|:---|
| 30 | Info | Normal renewal window |
| 14 | Warning | Renewal should have succeeded |
| 7 | Critical | Investigate immediately |
| 1 | Emergency | Manual intervention required |

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| keyAuth = token.thumbprint | Cryptographic binding | Domain proof |
| SHA-256(keyAuth) | One-way hash | DNS-01 challenge value |
| $(1-p)^{60}$ | Geometric probability | Renewal robustness |
| $\lceil n/50 \rceil \times 7$ | Ceiling + linear | Rate limit planning |
| ECDSA size savings | Linear scaling | Bandwidth optimization |
| CT Merkle depth $\log_2 n$ | Logarithmic | Proof size |

---

*Certbot transforms the manual, error-prone process of certificate management into a mathematically robust automation — the 30-day renewal window with 60 retry attempts makes certificate expiration a virtually impossible failure mode.*
