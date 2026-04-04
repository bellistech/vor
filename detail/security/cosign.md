# The Mathematics of Cosign -- Digital Signature Verification and Trust Chains

> *Container image signing with cosign relies on elliptic curve digital signatures (ECDSA) over SHA-256 digests, keyless flows that bind ephemeral certificates to OIDC identities via Fulcio, and Merkle tree inclusion proofs in the Rekor transparency log to provide tamper-evident audit trails.*

---

## 1. ECDSA Signature Scheme (Elliptic Curve Cryptography)

### The Problem

Cosign signs container image digests using ECDSA on the P-256 curve. The signer must produce a signature that anyone with the public key can verify, but no one without the private key can forge.

### The Formula

Key generation on curve $E$ over field $\mathbb{F}_p$ with generator point $G$ of order $n$:

$$d \xleftarrow{R} [1, n-1], \quad Q = dG$$

where $d$ is the private key and $Q$ is the public key.

Signing message $m$ with hash function $H$:

$$e = H(m), \quad z = \text{leftmost } \lceil \log_2 n \rceil \text{ bits of } e$$

$$k \xleftarrow{R} [1, n-1], \quad (x_1, y_1) = kG$$

$$r = x_1 \bmod n, \quad s = k^{-1}(z + rd) \bmod n$$

Signature is $(r, s)$.

Verification:

$$u_1 = zs^{-1} \bmod n, \quad u_2 = rs^{-1} \bmod n$$

$$P = u_1 G + u_2 Q$$

$$\text{valid} \iff P_x \bmod n = r$$

### Worked Examples

For P-256: $p \approx 2^{256}$, $n \approx 2^{256}$, key size = 32 bytes.

Image digest: `sha256:a3ed95caeb02ffe68cdd9fd84406680ae93d633cb16422d00e8a7c22955b46d4`

$$e = H(\texttt{sha256:a3ed95...}) \quad (\text{256-bit hash of the digest bytes})$$

Signature output: $(r, s)$ each 32 bytes, total signature = 64 bytes (DER-encoded ~71 bytes).

Security level: breaking ECDSA P-256 requires $O(2^{128})$ operations (birthday bound on discrete log).

---

## 2. Certificate Transparency (Merkle Trees)

### The Problem

Rekor stores every signing event in an append-only transparency log. Inclusion proofs allow anyone to verify that a specific signature was recorded without downloading the entire log.

### The Formula

Merkle tree with $n$ leaves. Each leaf $L_i$:

$$h_i = H(0x00 \| L_i)$$

Internal nodes:

$$h_{\text{parent}} = H(0x01 \| h_{\text{left}} \| h_{\text{right}})$$

Root hash:

$$R = h_{\text{root}}$$

Inclusion proof for leaf $L_i$ at index $i$ in tree of size $n$ requires:

$$\text{proof\_length} = \lceil \log_2 n \rceil$$

Verification: reconstruct path from $h_i$ to root using $\lceil \log_2 n \rceil$ sibling hashes.

### Worked Examples

Rekor log with $n = 10{,}000{,}000$ entries:

$$\text{proof\_length} = \lceil \log_2 10{,}000{,}000 \rceil = 24 \text{ hashes}$$

Each hash = 32 bytes (SHA-256):

$$\text{proof\_size} = 24 \times 32 = 768 \text{ bytes}$$

Verification cost: 24 hash computations (microseconds), regardless of log size.

Consistency proof between tree sizes $m$ and $n$ ($m < n$):

$$\text{proof\_length} \leq \lceil \log_2 n \rceil + 1$$

---

## 3. Keyless Signing (OIDC Certificate Binding)

### The Problem

Keyless signing eliminates long-lived keys by binding ephemeral signing certificates to OIDC identity tokens. The certificate lifetime must be short enough that key compromise is impractical, yet the signature must remain verifiable long after the certificate expires (via the transparency log timestamp).

### The Formula

Certificate validity window:

$$\Delta t_{\text{cert}} = t_{\text{expiry}} - t_{\text{issue}} \quad (\text{typically 10 minutes})$$

Signature is valid if:

$$t_{\text{issue}} \leq t_{\text{sign}} \leq t_{\text{expiry}} \quad \wedge \quad t_{\text{sign}} \in \text{Rekor}$$

Probability of key compromise during window $\Delta t$ given attack rate $\lambda$ (attempts/second):

$$P(\text{compromise}) = 1 - e^{-\lambda \cdot p_{\text{success}} \cdot \Delta t}$$

For ECDSA P-256: $p_{\text{success}} \approx 2^{-128}$ per attempt.

$$P(\text{compromise}) = 1 - e^{-\lambda \cdot 2^{-128} \cdot 600} \approx 0$$

### Worked Examples

Traditional long-lived key (1 year = $3.15 \times 10^7$ seconds):

$$P = 1 - e^{-10^{12} \times 2^{-128} \times 3.15 \times 10^7} \approx 9.3 \times 10^{-80}}$$

Ephemeral key (10 minutes = 600 seconds):

$$P = 1 - e^{-10^{12} \times 2^{-128} \times 600} \approx 1.8 \times 10^{-84}}$$

Both are astronomically small for P-256, but the ephemeral approach also eliminates key storage, rotation, and revocation overhead.

---

## 4. OCI Artifact Addressing (Content-Addressable Storage)

### The Problem

Cosign attaches signatures and attestations as OCI artifacts referenced by the image digest. The addressing scheme must guarantee that a signature corresponds to exactly one image content.

### The Formula

Image digest (content address):

$$d = \text{SHA256}(\text{manifest}) \in \{0,1\}^{256}$$

Collision resistance:

$$P(\text{collision after } q \text{ images}) = 1 - \prod_{i=0}^{q-1}\left(1 - \frac{i}{2^{256}}\right) \approx \frac{q^2}{2^{257}}$$

Signature tag convention:

$$\text{sig\_tag} = \text{sha256-} d_{\text{hex}}[0..63] \texttt{.sig}$$

### Worked Examples

Probability of digest collision after $q = 10^{18}$ images (far beyond any registry):

$$P \approx \frac{(10^{18})^2}{2^{257}} = \frac{10^{36}}{2^{257}} \approx \frac{10^{36}}{2.3 \times 10^{77}} \approx 4.3 \times 10^{-42}$$

Tag mutation attack (why sign by digest, not tag):

- Tag `v1.0` can be reassigned to different digest
- Digest `sha256:abc...` is immutable
- Cosign verification by digest: cryptographically bound
- Cosign verification by tag: vulnerable to tag reassignment between sign and verify

---

## 5. Policy Evaluation (Predicate Logic)

### The Problem

Admission controllers evaluate image signatures against policies. A policy defines which identities, issuers, and key sources are trusted. The policy engine must evaluate conjunctive and disjunctive trust predicates.

### The Formula

Trust predicate for image $I$:

$$\text{trusted}(I) = \exists \sigma \in \text{sigs}(I) : \text{valid}(\sigma) \wedge \text{policy}(\sigma)$$

Policy evaluation with identity and issuer constraints:

$$\text{policy}(\sigma) = (\sigma.\text{identity} \in \text{AllowedIdentities}) \wedge (\sigma.\text{issuer} \in \text{AllowedIssuers})$$

For multiple authorities (disjunctive):

$$\text{trusted}(I) = \bigvee_{a \in \text{Authorities}} \exists \sigma : \text{verify}(\sigma, a.\text{key}) \wedge \text{policy}_a(\sigma)$$

For required attestations (conjunctive):

$$\text{admitted}(I) = \text{trusted}(I) \wedge \bigwedge_{t \in \text{RequiredTypes}} \exists A_t \in \text{attestations}(I) : \text{valid}(A_t)$$

### Worked Examples

Policy: require signature from GitHub Actions CI AND an SBOM attestation.

Image has: 1 keyless signature (GitHub Actions identity) + 1 CycloneDX attestation.

$$\text{trusted}(I) = \text{verify}(\sigma, \text{Fulcio}) \wedge (\sigma.\text{issuer} = \texttt{token.actions.githubusercontent.com})$$

$$= \text{true} \wedge \text{true} = \text{true}$$

$$\text{admitted}(I) = \text{true} \wedge \text{valid}(A_{\text{cyclonedx}}) = \text{true} \wedge \text{true} = \text{true}$$

Image admitted.

Image missing SBOM attestation:

$$\text{admitted}(I) = \text{true} \wedge \text{false} = \text{false}$$

Image denied.

---

## 6. Signature Freshness and Timestamp Authority (Time)

### The Problem

When a certificate expires, how do we know the signature was created while the certificate was valid? Signed timestamps from a Timestamp Authority (TSA) or Rekor log entries provide cryptographic proof of signing time.

### The Formula

Timestamp token from TSA:

$$T = \text{Sign}_{K_{\text{TSA}}}(H(\sigma) \| t_{\text{sign}})$$

Validity condition:

$$\text{fresh}(\sigma, T) = \text{Verify}(T, K_{\text{TSA}}^{\text{pub}}) \wedge (t_{\text{sign}} \in [t_{\text{issue}}, t_{\text{expiry}}])$$

Rekor-based timestamping (log entry inclusion):

$$\text{fresh}(\sigma) = \exists e \in \text{Rekor} : e.\text{body} = \sigma \wedge e.\text{timestamp} \in [t_{\text{issue}}, t_{\text{expiry}}]$$

### Worked Examples

Certificate issued at $t_0$, expires at $t_0 + 600s$.

Signature created at $t_0 + 120s$, Rekor entry timestamped at $t_0 + 121s$.

$$t_0 \leq t_0 + 121 \leq t_0 + 600 \quad \checkmark$$

Verification at $t_0 + 86400s$ (24 hours later): certificate expired, but Rekor proves the signature was created within the validity window.

Without Rekor or TSA: no proof of signing time, signature cannot be verified after certificate expiry.

---

## Prerequisites

- elliptic-curve-cryptography, hash-functions, merkle-trees, predicate-logic, probability, oidc, x509-certificates
