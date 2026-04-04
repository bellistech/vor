# The Mathematics of SAML — Signature Verification, Time Windows, and Trust Chains

> *SAML 2.0 security depends on XML digital signatures, strict temporal bounds, and X.509 certificate chains. The mathematics of canonicalization uniqueness, clock skew tolerance, and signature wrapping attack surfaces quantify the protocol's security guarantees and failure modes.*

---

## 1. XML Signature Verification (Cryptography)

### The Problem

SAML assertions are signed using XML DSIG with enveloped signatures. The SP must verify that the signature covers the correct assertion and has not been tampered with. The verification involves canonicalization, digest computation, and RSA/ECDSA signature validation.

### The Formula

For RSA-SHA256 signature verification, the SP computes:

$$\text{digest} = \text{SHA256}(\text{C14N}(\text{SignedInfo}))$$

$$\text{valid} = \text{RSA-Verify}(PK_{IdP}, \text{digest}, \sigma)$$

Where $\text{C14N}$ is Exclusive XML Canonicalization (exc-c14n). The digest of the referenced element:

$$d_{ref} = \text{SHA256}(\text{C14N}(\text{Assertion} \setminus \text{Signature}))$$

Verification succeeds iff:

$$d_{ref} = d_{expected} \quad \land \quad \text{RSA-Verify}(PK_{IdP}, \text{SHA256}(\text{C14N}(\text{SignedInfo})), \sigma) = \text{true}$$

### Worked Examples

Computational cost of signature verification per assertion:

| Operation | RSA-2048 | RSA-4096 | ECDSA P-256 |
|:---|:---:|:---:|:---:|
| C14N (1 KB assertion) | 0.05 ms | 0.05 ms | 0.05 ms |
| SHA-256 digest | 0.002 ms | 0.002 ms | 0.002 ms |
| Signature verify | 0.06 ms | 0.18 ms | 0.12 ms |
| **Total per assertion** | **0.11 ms** | **0.23 ms** | **0.17 ms** |
| Max verifications/sec | 9,000 | 4,300 | 5,800 |

RSA-2048 provides the highest throughput. RSA-4096 is 2x slower for marginal security improvement. ECDSA P-256 offers equivalent security to RSA-3072 at better performance.

---

## 2. Assertion Time Window (Temporal Logic)

### The Problem

SAML assertions include `NotBefore` and `NotOnOrAfter` conditions. The SP must account for clock skew between itself and the IdP while keeping the acceptance window narrow enough to limit replay attacks.

### The Formula

The assertion is valid at SP time $t_{SP}$ with configured skew tolerance $\epsilon$ if:

$$\text{NotBefore} - \epsilon \leq t_{SP} \leq \text{NotOnOrAfter} + \epsilon$$

The effective acceptance window:

$$W_{effective} = (\text{NotOnOrAfter} - \text{NotBefore}) + 2\epsilon$$

Replay attack window (time during which a captured assertion can be reused):

$$W_{replay} = W_{effective} - (t_{SP} - t_{capture})$$

Probability of successful replay given uniform capture time:

$$P_{replay} = \frac{W_{effective}}{T_{session}} \quad \text{(if no one-time-use enforcement)}$$

### Worked Examples

IdP sets validity window of 5 minutes ($\text{NotOnOrAfter} - \text{NotBefore} = 300\text{s}$):

| Skew Tolerance ($\epsilon$) | Effective Window | Replay Window (mid-capture) | Replays/day at 1 attempt/min |
|:---:|:---:|:---:|:---:|
| 0 s | 300 s | 150 s | 0.10 |
| 30 s | 360 s | 180 s | 0.13 |
| 120 s | 540 s | 270 s | 0.19 |
| 300 s | 900 s | 450 s | 0.31 |

With one-time assertion ID enforcement (recommended), $P_{replay} = 0$ regardless of window size. Without it, keeping $\epsilon \leq 30\text{s}$ is critical.

---

## 3. XML Wrapping Attack Surface (Combinatorics)

### The Problem

XML Signature Wrapping (XSW) attacks exploit the gap between which element is signed and which element the application processes. An attacker can move the signed assertion to a non-processed location and inject a forged assertion where the application expects it.

### The Formula

For a SAML Response with $n$ possible insertion points in the XML tree and $m$ reference resolution strategies, the attack surface is:

$$A_{XSW} = n \cdot m \cdot p$$

where $p$ is the number of distinct XPath evaluation behaviors across XML parsers. In practice:

$$A_{XSW} \approx 2(d + 1) \cdot k$$

where $d$ is the XML tree depth and $k$ is the number of ID-reference resolution methods (getElementById, XPath `//[@ID]`, schema-validated ID).

Known XSW variants:

$$|\text{XSW variants}| = 8 \quad \text{(XSW1 through XSW8, documented by Somorovsky et al.)}$$

### Worked Examples

| XML Depth ($d$) | ID Methods ($k$) | Attack Surface ($A_{XSW}$) | Mitigated by strict validation |
|:---:|:---:|:---:|:---:|
| 3 | 2 | 16 | Yes |
| 5 | 3 | 36 | Yes |
| 8 | 3 | 54 | Yes |

Mitigation requires validating that the signature's Reference URI resolves to the exact assertion element being processed -- not a separate element with the same ID attribute.

---

## 4. Certificate Chain Validation (PKI Mathematics)

### The Problem

SAML metadata contains X.509 certificates for signature verification. The SP must validate the certificate chain from the IdP's signing certificate up to a trusted root. Chain validation involves checking signatures, validity periods, and revocation status.

### The Formula

For a certificate chain of length $l$ (end-entity to root), total validation cost:

$$C_{chain} = \sum_{i=1}^{l} (C_{sig_i} + C_{revocation_i})$$

Certificate validity overlap during rotation. Old cert expires at $t_1$, new cert valid from $t_0$:

$$W_{overlap} = t_1 - t_0$$

Minimum safe overlap to avoid downtime with metadata refresh interval $R$ and propagation delay $\Delta$:

$$W_{min} = R + \Delta + \epsilon_{safety}$$

### Worked Examples

With metadata refresh $R = 24\text{h}$, propagation delay $\Delta = 4\text{h}$, safety margin $\epsilon = 24\text{h}$:

| Scenario | $W_{min}$ | Recommended Overlap |
|:---|:---:|:---:|
| Single IdP, single SP | 52 h | 7 days |
| Federation (100 SPs) | 52 h | 14 days |
| Cross-org federation | 52 h | 30 days |

Longer overlaps accommodate slow metadata refresh in large federations. A 30-day overlap is standard practice for multi-organization deployments.

---

## 5. Assertion Entropy and ID Uniqueness (Probability)

### The Problem

SAML assertion IDs must be globally unique to enable one-time-use enforcement and prevent replay attacks. The SP maintains a cache of seen IDs within the validity window.

### The Formula

For random hex IDs of length $n$ characters (each character = 4 bits of entropy):

$$\text{Entropy} = 4n \text{ bits}$$

Birthday paradox -- probability of collision after $k$ assertions:

$$P_{collision} \approx 1 - e^{-k^2 / 2^{4n+1}}$$

For target collision probability $P_{target}$:

$$k_{max} = \sqrt{2^{4n+1} \cdot \ln\left(\frac{1}{1 - P_{target}}\right)}$$

### Worked Examples

SAML spec requires IDs starting with `_` followed by hex (typical implementations use 32-40 hex chars):

| ID Length ($n$ hex chars) | Entropy (bits) | $k_{max}$ at $P < 10^{-9}$ |
|:---:|:---:|:---:|
| 16 | 64 | 6,074 |
| 24 | 96 | 397 million |
| 32 | 128 | $2.6 \times 10^{13}$ |
| 40 | 160 | $1.7 \times 10^{18}$ |

With 32 hex characters (128-bit entropy), collision is practically impossible even at billions of assertions. Most implementations use 40+ hex characters, providing ample margin.

---

## 6. Deflate Compression Ratio (Information Theory)

### The Problem

The HTTP-Redirect binding deflates the AuthnRequest XML before base64-encoding it for the URL query string. URL length limits (typically 2,048 bytes in older browsers, 8,192 in modern ones) constrain the maximum uncompressed message size.

### The Formula

For XML with repetitive namespace declarations and tag names, the DEFLATE compression ratio:

$$r = \frac{|M_{compressed}|}{|M_{original}|}$$

After base64 encoding (4/3 expansion) and URL encoding (~1.5x for special chars):

$$|URL_{param}| = |M_{original}| \cdot r \cdot \frac{4}{3} \cdot e_{url}$$

Maximum original message size for URL limit $L$:

$$|M_{max}| = \frac{L}{r \cdot \frac{4}{3} \cdot e_{url}}$$

### Worked Examples

Typical SAML XML compression ratio $r \approx 0.30$, URL encoding expansion $e_{url} \approx 1.3$:

| URL Limit ($L$) | Max Original XML | Typical AuthnRequest | Fits? |
|:---:|:---:|:---:|:---:|
| 2,048 bytes | 3,938 bytes | 800-1,500 bytes | Yes |
| 4,096 bytes | 7,877 bytes | 800-1,500 bytes | Yes |
| 2,048 bytes | 3,938 bytes | 5,000 bytes (with extensions) | No |

Simple AuthnRequests fit comfortably. Responses with full assertions (5-20 KB) always require HTTP-POST binding.

---

## Prerequisites

- rsa-cryptography, sha-256, xml-canonicalization, x509-certificates, probability, information-theory
