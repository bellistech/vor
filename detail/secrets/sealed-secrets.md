# The Mathematics of Sealed Secrets — Asymmetric Encryption and Key Lifecycle

> *Sealed Secrets implements a public-key cryptosystem where RSA-OAEP encrypts each secret value independently, scoping binds ciphertext to namespace/name via authenticated encryption labels, and certificate rotation follows a forward-secrecy model with overlapping key windows.*

---

## 1. RSA-OAEP Encryption (Asymmetric Cryptography)

### The Sealing Operation

Each secret value is encrypted with RSA-OAEP:

$$C = \text{RSA-OAEP}_{PK}(\text{label}, M)$$

Where:
- $PK$ = controller's public key (2048 or 4096 bit RSA)
- $M$ = plaintext secret value
- $\text{label}$ = scope-dependent label (namespace/name)

### RSA Key Mathematics

$$n = p \times q \quad \text{(two large primes)}$$
$$\phi(n) = (p-1)(q-1)$$
$$e \times d \equiv 1 \pmod{\phi(n)}$$

Encryption: $C = M^e \bmod n$

Decryption: $M = C^d \bmod n$

### Key Sizes and Security

| Key Size | Security Level | Factoring Complexity | Status |
|:---:|:---:|:---:|:---|
| 1024 bit | 80 bit | $\sim 2^{80}$ ops | Deprecated |
| 2048 bit | 112 bit | $\sim 2^{112}$ ops | Standard |
| 4096 bit | 128 bit | $\sim 2^{128}$ ops | High security |

### OAEP Padding

OAEP (Optimal Asymmetric Encryption Padding) prevents chosen-ciphertext attacks:

$$\text{padded} = \text{OAEP}(M, \text{label}, \text{seed})$$

Maximum message size:

$$|M|_{max} = |n| - 2|H| - 2 = \frac{\text{key\_bits}}{8} - 2 \times 32 - 2$$

For 2048-bit RSA: $|M|_{max} = 256 - 66 = 190$ bytes.

### Hybrid Encryption for Large Values

For values exceeding RSA capacity, SOPS-style hybrid encryption is used:

$$K_{sym} \xleftarrow{\$} \{0,1\}^{256}$$
$$C_{key} = \text{RSA-OAEP}(K_{sym})$$
$$C_{value} = \text{AES-GCM}_{K_{sym}}(\text{value})$$

---

## 2. Scoping as Authenticated Labels (Binding Theory)

### The Scope Model

Scoping binds ciphertext to metadata, preventing relocation attacks:

$$\text{label}(\text{strict}) = \text{namespace} \| \text{name}$$
$$\text{label}(\text{namespace-wide}) = \text{namespace}$$
$$\text{label}(\text{cluster-wide}) = \epsilon \text{ (empty)}$$

### Security Implications

| Scope | Label Binding | Relocation Risk | Use Case |
|:---|:---|:---|:---|
| strict | namespace + name | None | Default, most secure |
| namespace-wide | namespace only | Can rename secret | Shared secrets in NS |
| cluster-wide | None | Full relocation | Cross-namespace sharing |

### Attack Prevention

Without scoping, an attacker with cluster write access could:

$$\text{Attack}: \text{copy SealedSecret to attacker's namespace} \to \text{controller decrypts} \to \text{attacker reads Secret}$$

With strict scoping:

$$D_{SK}(\text{label}_{attacker\_ns}, C) \neq M$$

Because $\text{label}_{attacker\_ns} \neq \text{label}_{original\_ns}$.

---

## 3. Certificate Rotation (Key Lifecycle)

### Rotation Timeline

The controller generates new keys every $T_r$ (default 30 days = 720 hours):

$$K_i \text{ valid from } t_i = i \times T_r$$

### Key Retention

Old keys are retained for decryption. At time $t$:

$$\text{Active keys} = \{K_0, K_1, \ldots, K_{\lfloor t/T_r \rfloor}\}$$

### Forward Secrecy Window

New sealing uses the latest key $K_{current}$:

$$\text{Seal}(M, t) = E_{PK_{current}}(M)$$

If $K_i$ is compromised:

$$\text{Compromised secrets} = \{\text{sealed with } K_i\}$$

Secrets sealed with other keys remain safe.

### Storage Growth

$$S_{keys}(t) = \left\lfloor \frac{t}{T_r} \right\rfloor \times S_{key\_pair}$$

Where $S_{key\_pair} \approx 3\text{KB}$ for 2048-bit RSA.

After 1 year (12 rotations): $S_{keys} \approx 36\text{KB}$.

### Re-Sealing Cost

When rotating, re-sealing all secrets:

$$T_{reseal} = N_{secrets} \times V_{values} \times T_{RSA\_encrypt}$$

| Secrets | Values/Secret | RSA Ops | Time (2048-bit) |
|:---:|:---:|:---:|:---:|
| 50 | 5 | 250 | ~50ms |
| 200 | 10 | 2,000 | ~400ms |
| 1000 | 20 | 20,000 | ~4s |

---

## 4. Controller Decryption Pipeline (Reconciliation Loop)

### Event-Driven Processing

The controller watches SealedSecret resources and decrypts on creation/update:

$$\text{Reconcile}(SS) = \begin{cases}
\text{Create Secret} & \text{if } \nexists \text{Secret}(SS.name) \\
\text{Update Secret} & \text{if } SS.\text{generation} > \text{Secret}.\text{generation} \\
\text{No-op} & \text{otherwise}
\end{cases}$$

### Decryption Latency

$$T_{decrypt} = T_{event} + T_{RSA} \times |values| + T_{API\_write}$$

Typical: $T_{event} \approx 10\text{ms}$, $T_{RSA} \approx 2\text{ms}$, $T_{API\_write} \approx 20\text{ms}$:

$$T_{decrypt} \approx 30\text{ms} + 2 \times |values|\text{ms}$$

### Throughput Bound

Controller processes SealedSecrets sequentially:

$$\text{Throughput} = \frac{1}{T_{decrypt}}$$

For 5-value secrets: $\text{Throughput} \approx \frac{1}{40\text{ms}} = 25$ secrets/second.

---

## 5. Backup and Recovery (Disaster Recovery)

### Key Loss Impact

If the controller's private keys are lost:

$$P(\text{recovery without backup}) = 0$$

All existing SealedSecrets become permanently undecryptable.

### Backup Requirements

$$\text{Minimum backup set} = \{K_{SK_i} : i = 0, 1, \ldots, \text{current}\}$$

### Recovery Time Objective

$$T_{recovery} = T_{restore\_keys} + T_{restart\_controller} + T_{reconcile\_all}$$

$$T_{recovery} \approx 30\text{s} + 60\text{s} + N_{secrets} \times 40\text{ms}$$

For 200 secrets: $T_{recovery} \approx 98\text{s}$.

### Multi-Cluster Key Management

Each cluster has its own key pair:

$$\text{Total keys} = |clusters| \times \lceil T / T_r \rceil$$

For 5 clusters over 1 year: $5 \times 12 = 60$ key pairs to manage.

---

## 6. Security Model (Threat Analysis)

### Trust Boundaries

$$\text{Trust}_{seal} = \text{anyone with public cert (non-sensitive)}$$
$$\text{Trust}_{unseal} = \text{only the controller with private key}$$

### Attack Surface

| Attack Vector | Mitigated By | Residual Risk |
|:---|:---|:---|
| Git repo read access | RSA encryption | None (ciphertext is safe) |
| Cluster namespace access | Scope binding | Cluster-wide scope weakens |
| Controller compromise | Key backup + rotation | Full compromise = all secrets |
| Stolen sealed YAML | RSA-OAEP security | Quantum computing (future) |
| Key backup theft | Encrypt backups | Depends on backup security |

### Quantum Computing Threat

RSA-2048 broken by Shor's algorithm with:

$$\text{Qubits needed} \approx 4n + 2 = 8194 \text{ for } n = 2048$$

Current state: ~1000 noisy qubits. Estimated timeline: 10-20+ years.

### Comparison: Sealed Secrets vs SOPS

| Property | Sealed Secrets | SOPS |
|:---|:---|:---|
| Encryption type | Asymmetric (RSA) | Hybrid (envelope) |
| Key management | Controller in cluster | External (age/KMS) |
| Decrypt location | Only in cluster | Anywhere with key |
| Multi-key support | Single controller key | Multiple backends |
| Key rotation | Automatic (30 days) | Manual |
| Git-diff friendly | Partial (base64 blobs) | Yes (key paths visible) |
| Platform lock-in | Kubernetes only | Any platform |

---

## 7. Encryption Performance (Computational Cost)

### RSA Operation Benchmarks

| Operation | 2048-bit | 4096-bit | Relative |
|:---|:---:|:---:|:---:|
| Key generation | ~100ms | ~1s | 10x |
| Encrypt (public) | ~0.2ms | ~0.5ms | 2.5x |
| Decrypt (private) | ~2ms | ~15ms | 7.5x |
| Sign | ~2ms | ~15ms | 7.5x |
| Verify | ~0.2ms | ~0.5ms | 2.5x |

### Sealing vs Unsealing Asymmetry

Sealing (encryption with public key) is fast:

$$T_{seal}(n) = n \times 0.2\text{ms} \approx 0$$

Unsealing (decryption with private key) dominates:

$$T_{unseal}(n) = n \times 2\text{ms}$$

This asymmetry is by design: many developers seal quickly, one controller unseals.

---

*Sealed Secrets reduces Kubernetes secret management to an asymmetric encryption problem: public keys enable anyone to seal, private keys restrict unsealing to the cluster controller. Scoping prevents relocation attacks through authenticated labels, and automatic key rotation limits the blast radius of key compromise.*

## Prerequisites

- RSA cryptosystem (key generation, encryption, decryption)
- OAEP padding scheme
- Public key infrastructure (certificates, X.509)
- Kubernetes Secret and controller reconciliation model

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Seal one value (RSA encrypt) | $O(1)$ ~0.2ms | $O(|n|)$ |
| Unseal one value (RSA decrypt) | $O(1)$ ~2ms | $O(|n|)$ |
| Seal entire secret | $O(v)$ values | $O(v \times |n|)$ |
| Certificate fetch | $O(1)$ API call | $O(|cert|)$ |
| Key rotation | $O(1)$ generation | $O(|key|)$ |
| Re-seal all secrets | $O(N \times v)$ | $O(N \times v \times |n|)$ |
