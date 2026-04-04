# The Mathematics of SOPS — Envelope Encryption and Secret Sharing

> *SOPS implements envelope encryption where a random data key encrypts secrets and is itself encrypted by multiple master keys. Combined with Shamir's Secret Sharing, SOPS provides M-of-N threshold access control with information-theoretic security guarantees.*

---

## 1. Envelope Encryption (Symmetric + Asymmetric Hybrid)

### The Two-Layer Model

SOPS uses a data encryption key (DEK) encrypted by one or more key encryption keys (KEKs):

$$\text{Ciphertext}_i = E_{DEK}(\text{value}_i)$$
$$\text{EncryptedDEK}_j = E_{KEK_j}(DEK)$$

### Why Envelope Encryption

Direct encryption with $n$ master keys and $m$ values:

$$\text{Direct: } n \times m \text{ encryption operations}$$
$$\text{Envelope: } m + n \text{ encryption operations}$$

| Values | Master Keys | Direct Ops | Envelope Ops | Speedup |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 3 | 30 | 13 | 2.3x |
| 100 | 5 | 500 | 105 | 4.8x |
| 1000 | 10 | 10,000 | 1,010 | 9.9x |

### Key Hierarchy

$$\text{Master Key (KEK)} \xrightarrow{\text{wraps}} \text{Data Key (DEK)} \xrightarrow{\text{encrypts}} \text{Values}$$

Decryption requires ANY ONE master key:

$$DEK = D_{KEK_j}(\text{EncryptedDEK}_j) \quad \text{for any valid } j$$

$$\text{value}_i = D_{DEK}(\text{Ciphertext}_i)$$

---

## 2. AES-256-GCM Encryption (Symmetric Cipher)

### The Encryption Function

SOPS uses AES-256 in GCM mode for value encryption:

$$C = \text{AES-GCM}_{DEK}(IV, \text{plaintext}, \text{AAD})$$

Where:
- $DEK$ = 256-bit data encryption key
- $IV$ = 96-bit initialization vector (unique per value)
- $AAD$ = additional authenticated data (the YAML key path)

### Security Properties

$$\text{Key space} = 2^{256} \approx 1.16 \times 10^{77}$$

Brute force at $10^{18}$ operations/second:

$$T_{brute} = \frac{2^{256}}{10^{18}} \approx 3.67 \times 10^{51} \text{ years}$$

### GCM Authentication Tag

$$\text{Tag} = \text{GHASH}_{H}(C \| \text{AAD}) \oplus E_K(J_0)$$

Tag provides integrity verification:

$$P(\text{forgery}) = 2^{-128} \approx 2.94 \times 10^{-39}$$

### IV Collision Risk (Birthday Problem)

GCM IVs are 96 bits. Collision probability after $q$ encryptions:

$$P(\text{collision}) \approx \frac{q^2}{2^{97}}$$

For $q = 10^6$ values:

$$P(\text{collision}) \approx \frac{10^{12}}{1.58 \times 10^{29}} \approx 6.3 \times 10^{-18}$$

Safe for up to $2^{48} \approx 2.8 \times 10^{14}$ values per key.

---

## 3. Shamir's Secret Sharing (Threshold Cryptography)

### The Polynomial Construction

SOPS supports Shamir threshold for key groups. Split a secret $S$ into $n$ shares with threshold $k$:

$$f(x) = S + a_1 x + a_2 x^2 + \cdots + a_{k-1} x^{k-1} \pmod{p}$$

Where $a_1, \ldots, a_{k-1}$ are random coefficients and $p$ is a prime.

Share $i$: $(i, f(i))$

### Reconstruction via Lagrange Interpolation

Given $k$ shares $\{(x_1, y_1), \ldots, (x_k, y_k)\}$:

$$S = f(0) = \sum_{i=1}^{k} y_i \prod_{j \neq i} \frac{-x_j}{x_i - x_j} \pmod{p}$$

### Security Guarantee

With fewer than $k$ shares, the secret is information-theoretically secure:

$$H(S | \text{any } k-1 \text{ shares}) = H(S)$$

No computational power can recover $S$ from $k-1$ shares.

### SOPS Key Group Example

With `shamir_threshold: 2` and 3 key groups:

$$\binom{3}{2} = 3 \text{ possible decryption combinations}$$

| Configuration | Shares | Threshold | Combinations |
|:---:|:---:|:---:|:---:|
| 2-of-3 | 3 | 2 | 3 |
| 3-of-5 | 5 | 3 | 10 |
| 2-of-5 | 5 | 2 | 10 |
| 4-of-7 | 7 | 4 | 35 |

---

## 4. Selective Encryption (Tree Traversal)

### The Encryption Scope

SOPS encrypts values but preserves keys. For a YAML tree $T$:

$$\text{Encrypt}(T) = \text{map}(\text{encrypt\_value}, \text{leaves}(T))$$

### encrypted_suffix Filter

With `encrypted_suffix: _secret`, only matching keys are encrypted:

$$\text{Encrypt}(k, v) = \begin{cases}
E_{DEK}(v) & \text{if } k \text{ ends with } \texttt{\_secret} \\
v & \text{otherwise}
\end{cases}$$

### Tree Size and Encryption Cost

For a YAML file with $n$ leaf nodes and $m$ matching the filter:

$$T_{encrypt} = m \times T_{AES} + T_{DEK\_wrap} \times |KEKs|$$

$$T_{AES} \approx 0.001\text{ms per value}$$
$$T_{DEK\_wrap} \approx 1\text{ms (age)} \text{ to } 50\text{ms (KMS API call)}$$

| File Size | Leaf Nodes | Encrypted | Encrypt Time (age) | Encrypt Time (KMS) |
|:---:|:---:|:---:|:---:|:---:|
| Small | 20 | 5 | ~2ms | ~55ms |
| Medium | 100 | 30 | ~5ms | ~55ms |
| Large | 500 | 100 | ~10ms | ~60ms |

KMS dominates because of the API call, regardless of file size.

---

## 5. Key Rotation Mathematics (Cryptographic Hygiene)

### Rotation Model

Key rotation in SOPS has two operations:

1. **Data key rotation**: Generate new DEK, re-encrypt all values
2. **Master key rotation**: Re-wrap DEK with new KEK

### Data Key Rotation Complexity

$$T_{rotate} = T_{generate\_DEK} + n \times T_{decrypt} + n \times T_{encrypt} + |KEKs| \times T_{wrap}$$

### Exposure Window

If a key is compromised at time $t_c$ and rotated every $T_r$:

$$\text{Exposure window} = T_r$$

$$\text{Compromised secrets} = \text{secrets created in } [t_c - T_r, t_c]$$

### Rotation Schedule Optimization

Balance security vs. operational cost:

$$\text{Risk}(T_r) = \lambda \times T_r \times V_{secrets}$$

$$\text{Cost}(T_r) = \frac{C_{rotation}}{T_r}$$

$$T_r^{opt} = \sqrt{\frac{C_{rotation}}{\lambda \times V_{secrets}}}$$

Where $\lambda$ = compromise probability rate, $V$ = value of secrets.

---

## 6. MAC and Integrity (Message Authentication)

### SOPS MAC Computation

SOPS computes a MAC over all encrypted values to detect tampering:

$$\text{MAC} = \text{HMAC-SHA256}\left(DEK, \text{concat}(\text{all encrypted values})\right)$$

### Tamper Detection

Any modification to any encrypted value changes the MAC:

$$\text{Valid} \iff \text{MAC}_{stored} = \text{MAC}_{computed}$$

### Collision Resistance

HMAC-SHA256 provides:

$$P(\text{collision}) = 2^{-256}$$
$$P(\text{forgery without key}) = 2^{-256}$$

---

## 7. Diff-Friendly Encryption (Information Preservation)

### Structural Preservation

SOPS preserves YAML/JSON structure:

$$\text{keys}(\text{Encrypt}(T)) = \text{keys}(T)$$

This means git diffs show which keys changed:

$$\Delta(\text{Encrypt}(T_1), \text{Encrypt}(T_2)) \supseteq \Delta(\text{keys}(T_1), \text{keys}(T_2))$$

### Diff Entropy

A single value change produces one changed line in the diff, not a full file change:

$$|\text{diff}| = O(\text{changed values})$$

Compared to full-file encryption:

$$|\text{diff}_{full\_encryption}| = O(|\text{file}|)$$

### Review Efficiency

Code review effort:

$$\text{Effort}_{sops} = O(|\text{changes}|)$$
$$\text{Effort}_{sealed} = O(|\text{file}|) \text{ (opaque blob)}$$

---

*SOPS transforms secret management into a mathematically sound problem: AES-GCM provides authenticated encryption, envelope encryption separates key management from data protection, and Shamir's Secret Sharing enables threshold access control. The structural preservation of YAML keys makes encrypted secrets reviewable and diffable.*

## Prerequisites

- Symmetric encryption (AES, GCM mode)
- Asymmetric encryption (RSA, elliptic curves)
- Polynomial interpolation (Lagrange)
- Hash-based message authentication codes (HMAC)

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Encrypt single value | $O(|v|)$ | $O(|v|)$ |
| Encrypt file | $O(n \times |v|)$ | $O(n \times |v|)$ |
| DEK wrap (age) | $O(1)$ ~1ms | $O(1)$ |
| DEK wrap (KMS) | $O(1)$ ~50ms | $O(1)$ |
| Shamir split (k-of-n) | $O(n \times k)$ | $O(n)$ shares |
| Shamir reconstruct | $O(k^2)$ | $O(k)$ |
| Key rotation | $O(n + |KEKs|)$ | $O(n)$ |
