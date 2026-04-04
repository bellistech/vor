# The Mathematics of SSH — Secure Shell Protocol Engineering

> *SSH (RFC 4253) combines key exchange, public-key authentication, symmetric encryption, and channel multiplexing into a layered protocol. Every connection negotiates fresh cryptographic material through precise mathematical operations.*

---

## 1. Key Exchange — Curve25519 Diffie-Hellman

### The Default: curve25519-sha256

SSH uses Curve25519 (RFC 8731) as the preferred key exchange. The curve operates over $\mathbb{F}_p$ where:

$$p = 2^{255} - 19$$

### Exchange Protocol

1. Client generates ephemeral secret $a$, computes $A = a \cdot G$
2. Server generates ephemeral secret $b$, computes $B = b \cdot G$
3. Shared secret: $K = a \cdot B = b \cdot A = ab \cdot G$
4. Exchange hash: $H = \text{SHA-256}(\text{V\_C} \| \text{V\_S} \| \text{I\_C} \| \text{I\_S} \| K_S \| A \| B \| K)$

Where $V_C, V_S$ are version strings, $I_C, I_S$ are key exchange init messages, and $K_S$ is the server's host key.

### Session ID and Key Derivation

Session ID = first exchange hash $H$. All session keys derived via:

$$\text{key} = \text{HASH}(K \| H \| X \| \text{session\_id})$$

Where $X$ is a single character identifying the key purpose:

| $X$ | Derived Key |
|:---:|:---|
| 'A' | Client-to-server IV |
| 'B' | Server-to-client IV |
| 'C' | Client-to-server encryption key |
| 'D' | Server-to-client encryption key |
| 'E' | Client-to-server MAC key |
| 'F' | Server-to-client MAC key |

---

## 2. Host Key Algorithms

### Key Types and Security

| Algorithm | Key Size | Security Level | Signature Size |
|:---|:---:|:---:|:---:|
| ssh-ed25519 | 256 bits | 128-bit | 64 bytes |
| ecdsa-sha2-nistp256 | 256 bits | 128-bit | 64 bytes |
| ecdsa-sha2-nistp384 | 384 bits | 192-bit | 96 bytes |
| rsa-sha2-256 | 3072+ bits | 128-bit | 384 bytes |
| rsa-sha2-512 | 3072+ bits | 128-bit | 384 bytes |
| ssh-dss (DSA) | 1024 bits | 80-bit | 40 bytes |

### Ed25519 Signature (RFC 8032)

Given private scalar $a$, public point $A = aG$, message $M$:

1. $r = \text{SHA-512}(\text{prefix} \| M) \pmod{\ell}$ (deterministic nonce)
2. $R = rG$
3. $S = r + \text{SHA-512}(R \| A \| M) \cdot a \pmod{\ell}$

Signature: $(R, S)$ — 64 bytes total.

Verification: $SG \stackrel{?}{=} R + \text{SHA-512}(R \| A \| M) \cdot A$

The group order: $\ell = 2^{252} + 27742317777372353535851937790883648493$

---

## 3. Symmetric Encryption — AEAD Ciphers

### Cipher Comparison

| Cipher | Key | IV/Nonce | Block | Mode |
|:---|:---:|:---:|:---:|:---|
| chacha20-poly1305@openssh.com | 256 bits | 64 bits | stream | AEAD |
| aes256-gcm@openssh.com | 256 bits | 12 bytes | 128 bits | AEAD |
| aes128-gcm@openssh.com | 128 bits | 12 bytes | 128 bits | AEAD |
| aes256-ctr | 256 bits | 16 bytes | 128 bits | CTR + separate MAC |
| aes128-ctr | 128 bits | 16 bytes | 128 bits | CTR + separate MAC |

### ChaCha20-Poly1305 in SSH

SSH uses a modified ChaCha20-Poly1305 with **two keys**:

- $K_1$: encrypts the packet payload
- $K_2$: encrypts the 4-byte packet length (separately)

$$\text{encrypted\_length} = \text{ChaCha20}_{K_2}(\text{length})$$
$$\text{encrypted\_payload} = \text{ChaCha20}_{K_1}(\text{payload})$$
$$\text{tag} = \text{Poly1305}_{r,s}(\text{encrypted\_length} \| \text{encrypted\_payload})$$

This prevents length-based traffic analysis — the length is encrypted before the payload is decrypted.

---

## 4. MAC Algorithms (Non-AEAD Modes)

### HMAC Construction

$$\text{HMAC}(K, m) = H\bigl((K \oplus \text{opad}) \| H((K \oplus \text{ipad}) \| m)\bigr)$$

Where $\text{opad} = \text{0x5C}\ldots$ and $\text{ipad} = \text{0x36}\ldots$

### MAC Security

| MAC | Output | Security (forgery) |
|:---|:---:|:---:|
| hmac-sha2-256 | 256 bits | $2^{-256}$ per attempt |
| hmac-sha2-512 | 512 bits | $2^{-512}$ per attempt |
| hmac-sha2-256-etm@openssh.com | 256 bits | $2^{-256}$ (encrypt-then-MAC) |
| umac-128-etm@openssh.com | 128 bits | $2^{-128}$ (faster) |

**Encrypt-then-MAC (ETM)** is preferred over encrypt-and-MAC because:
- MAC covers ciphertext, preventing padding oracle attacks
- Verification before decryption (fail-fast)

---

## 5. Channel Multiplexing

### Channel Model

SSH multiplexes multiple logical channels over a single TCP connection:

$$\text{Connection} \rightarrow \{C_0, C_1, \ldots, C_n\}$$

Each channel has independent:
- **Window size** $W$: flow control (bytes)
- **Maximum packet size** $P_{max}$: per-message limit

### Flow Control

$$\text{Sender can transmit} = W_{remote} - \text{bytes\_in\_flight}$$

When the receiver processes data, it sends `SSH_MSG_CHANNEL_WINDOW_ADJUST`:

$$W_{new} = W_{old} + \Delta W$$

### Channel Types

| Type | Use | Typical Window |
|:---|:---|:---:|
| session | Shell, exec, subsystem | 2 MB |
| direct-tcpip | Local port forward | 64 KB |
| forwarded-tcpip | Remote port forward | 64 KB |
| x11 | X11 forwarding | 64 KB |

### Multiplexing Overhead

Per-channel overhead per packet:

$$\text{overhead} = 4(\text{channel ID}) + 4(\text{data length}) + \text{MAC length} + \text{padding}$$

Minimum overhead: ~28 bytes per packet (with AES-GCM).

---

## 6. SSH Key Entropy

### Key Generation

| Key Type | Private Key Entropy | Bits of Security |
|:---|:---:|:---:|
| ed25519 | 256 bits (32 bytes random) | 128-bit |
| ecdsa-nistp256 | 256 bits | 128-bit |
| rsa-4096 | Two 2048-bit primes | 128-bit |
| rsa-2048 | Two 1024-bit primes | 112-bit |

### Passphrase Protection (bcrypt-pbkdf)

OpenSSH encrypts private keys with bcrypt-pbkdf:

$$\text{derived\_key} = \text{bcrypt-pbkdf}(\text{passphrase}, \text{salt}, \text{rounds})$$

Default rounds: 16. Each round cost:

$$T_{total} = \text{rounds} \times T_{bcrypt} \approx 16 \times 0.1s = 1.6s$$

Brute force against passphrase-protected key:

$$T_{crack} = \frac{|\text{keyspace}|}{2} \times T_{total} \times \frac{1}{\text{parallelism}}$$

| Passphrase | Keyspace | Time (1 GPU) |
|:---|:---:|:---:|
| 4 random words (diceware) | $7776^4 = 3.7 \times 10^{15}$ | ~18,700 years |
| 8 lowercase chars | $26^8 = 2.1 \times 10^{11}$ | ~1,050 years |
| 6 mixed chars | $62^6 = 5.7 \times 10^{10}$ | ~285 years |
| "password123" | dictionary | seconds |

---

## 7. Agent Forwarding Attack Surface

### How Agent Forwarding Works

```
Laptop (agent) <--- unix socket ---> Server A <--- forwarded socket ---> Server B
```

The SSH agent holds private keys in memory. When forwarding is enabled:

1. Server A creates a Unix socket
2. Any process on Server A with access to that socket can request signatures
3. The agent on the laptop performs the signing operation

### Attack Model

If Server A is compromised, the attacker can:

$$P(\text{lateral movement}) = \begin{cases} 1 & \text{if agent forwarding enabled and attacker has socket access} \\ 0 & \text{if agent forwarding disabled or ProxyJump used} \end{cases}$$

### ProxyJump vs Agent Forwarding

| Method | Key Exposure | Attack Surface |
|:---|:---|:---|
| Agent forwarding | Key usable from any hop | All intermediate servers |
| ProxyJump (-J) | Key never leaves laptop | Zero (TCP forwarding only) |
| ProxyCommand | Key never leaves laptop | Zero (stdin/stdout pipe) |

**ProxyJump** tunnels SSH through intermediate hosts without exposing the agent:

$$\text{Laptop} \xrightarrow{\text{TCP tunnel}} \text{Bastion} \xrightarrow{\text{TCP tunnel}} \text{Target}$$

Authentication occurs end-to-end — the bastion only sees encrypted traffic.

---

## 8. SSH Fingerprint Mathematics

### Fingerprint Formats

| Format | Algorithm | Length | Example |
|:---|:---|:---:|:---|
| MD5 (legacy) | MD5(pubkey) | 128 bits | `d4:15:a3:...` (hex pairs) |
| SHA256 (current) | SHA-256(pubkey) | 256 bits | `SHA256:jE4r...` (base64) |

### Collision Probability (Birthday Bound)

For SHA-256 fingerprints among $k$ keys:

$$P(\text{collision}) \approx \frac{k^2}{2^{257}}$$

| Keys | Collision Probability |
|:---:|:---:|
| $10^6$ | $4.3 \times 10^{-66}$ |
| $10^{12}$ | $4.3 \times 10^{-54}$ |
| $10^{18}$ | $4.3 \times 10^{-42}$ |

Random art visualization (OpenSSH `VisualHostKey`) maps the fingerprint to a 17x9 grid using a random walk — humans detect visual differences more reliably than hex strings.

---

## 9. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Curve25519 DH | Elliptic curve scalar multiply | Key exchange |
| Ed25519 | Schnorr signature on Edwards curve | Host/user authentication |
| HMAC | Keyed hash composition | Message authentication |
| bcrypt-pbkdf | Iterated hash (cost function) | Passphrase protection |
| Channel windows | Sliding window flow control | Multiplexing |
| SHA-256 fingerprint | Collision-resistant hash | Key verification |
| Agent forwarding | Trust delegation chain | (Anti-pattern) |

## Prerequisites

- modular arithmetic, key exchange algorithms, hash functions, finite fields

---

*Every `ssh user@host` invocation executes this entire cryptographic stack — key exchange, authentication, encryption, and channel setup — in under 500ms, establishing a mathematically verified secure tunnel.*
