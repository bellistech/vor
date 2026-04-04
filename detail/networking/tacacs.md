# The Mathematics of TACACS+ — Encryption Mechanics, Session Analysis, and Security Comparison

> *TACACS+ secures network device administration through full-body packet encryption and separated AAA services. The mathematical foundations of its encryption scheme, session handling, and security trade-offs reveal both its strengths and its known limitations.*

---

## 1. Packet Encryption — MD5 Pad Generation (Cryptography)

### Pseudo-Pad Construction

TACACS+ encrypts the packet body by XORing it with a pseudo-random pad generated from MD5 hashes:

$$\text{pad}_1 = \text{MD5}(S \| K \| V \| N)$$

$$\text{pad}_n = \text{MD5}(S \| K \| V \| N \| \text{pad}_{n-1}), \quad n > 1$$

Where:
- $S$ = session ID (4 bytes, random)
- $K$ = shared secret key
- $V$ = version byte (1 byte)
- $N$ = sequence number (1 byte)
- $\|$ = concatenation

The total pad is:

$$\text{pad} = \text{pad}_1 \| \text{pad}_2 \| \cdots \| \text{pad}_{\lceil L/16 \rceil}$$

Truncated to $L$ bytes (body length). Each MD5 output is 16 bytes (128 bits).

### Number of MD5 Operations

For a body of length $L$ bytes:

$$n_{MD5} = \left\lceil \frac{L}{16} \right\rceil$$

| Body Size | MD5 Operations | Pad Bytes Generated |
|:---:|:---:|:---:|
| 16 B | 1 | 16 |
| 32 B | 2 | 32 |
| 64 B | 4 | 64 |
| 128 B | 8 | 128 |
| 256 B | 16 | 256 |
| 1024 B | 64 | 1024 |

### Encryption/Decryption Operation

$$\text{encrypted\_body} = \text{body} \oplus \text{pad}[0:L]$$

$$\text{body} = \text{encrypted\_body} \oplus \text{pad}[0:L]$$

This is a symmetric stream cipher. The XOR operation is its own inverse: $A \oplus B \oplus B = A$.

---

## 2. Session ID Collision Probability (Birthday Problem)

### Collision Analysis

The session ID is a 32-bit random value. The probability of at least one collision among $n$ concurrent sessions follows the birthday problem approximation:

$$P(\text{collision}) \approx 1 - e^{-\frac{n(n-1)}{2 \times 2^{32}}}$$

For small $n$ relative to $2^{32}$:

$$P(\text{collision}) \approx \frac{n^2}{2 \times 2^{32}} = \frac{n^2}{2^{33}}$$

| Concurrent Sessions | Collision Probability |
|:---:|:---:|
| 100 | 0.0000012 (1.2 in a million) |
| 1,000 | 0.00012 |
| 10,000 | 0.012 (1.2%) |
| 50,000 | 0.25 (25%) |
| 65,536 ($2^{16}$) | 0.39 (39%) |

### Security Implication

A session ID collision with the same shared secret and sequence number produces the same pad. An attacker who observes two sessions with the same ID can XOR the ciphertexts:

$$C_1 \oplus C_2 = (P_1 \oplus \text{pad}) \oplus (P_2 \oplus \text{pad}) = P_1 \oplus P_2$$

This leaks the XOR of the two plaintexts, which is exploitable with known-plaintext attacks.

---

## 3. Key Space and Brute Force Analysis

### Shared Secret Entropy

The shared secret is typically an ASCII string. For a key of length $k$ characters from an alphabet of size $a$:

$$\text{Key space} = a^k$$

$$\text{Entropy} = k \times \log_2(a) \text{ bits}$$

| Key Type | Alphabet ($a$) | Length ($k$) | Entropy |
|:---|:---:|:---:|:---:|
| Lowercase only | 26 | 8 | 37.6 bits |
| Alphanumeric | 62 | 8 | 47.6 bits |
| Full printable ASCII | 95 | 8 | 52.6 bits |
| Alphanumeric | 62 | 16 | 95.3 bits |
| Full printable ASCII | 95 | 16 | 105.1 bits |
| Full printable ASCII | 95 | 24 | 157.7 bits |

### Offline Attack Cost

An attacker who captures TACACS+ packets can attempt offline key recovery. For each candidate key, compute one MD5 hash and check against known plaintext fields:

$$\text{Time} = \frac{a^k}{R_{MD5}}$$

At $R_{MD5} = 10^{10}$ hashes/second (GPU cluster):

| Key Entropy | Key Space | Time to Exhaust |
|:---:|:---:|:---:|
| 37.6 bits | $2.1 \times 10^{11}$ | 21 seconds |
| 47.6 bits | $2.2 \times 10^{14}$ | 6.1 hours |
| 52.6 bits | $6.6 \times 10^{15}$ | 7.7 days |
| 95.3 bits | $4.8 \times 10^{28}$ | 1.5 billion years |

This demonstrates why TACACS+ shared secrets must be at least 16 characters with mixed character sets.

---

## 4. TACACS+ vs RADIUS Encryption — Quantitative Comparison

### Coverage Analysis

TACACS+ encrypts the entire body; RADIUS encrypts only the User-Password attribute:

$$\text{Coverage}_{TACACS+} = \frac{L_{body}}{L_{total}} = \frac{L_{total} - 12}{L_{total}}$$

$$\text{Coverage}_{RADIUS} = \frac{L_{password}}{L_{total}}$$

For a typical authentication exchange:

| Protocol | Total Packet | Encrypted Portion | Coverage |
|:---|:---:|:---:|:---:|
| TACACS+ authen START | 48 B | 36 B (body) | 75.0% |
| TACACS+ author REQ | 128 B | 116 B (body) | 90.6% |
| TACACS+ acct REQ | 256 B | 244 B (body) | 95.3% |
| RADIUS Access-Request | 200 B | 16 B (password) | 8.0% |
| RADIUS Acct-Request | 300 B | 0 B | 0.0% |

### Information Leakage

In a RADIUS packet, the following are sent in cleartext:
- Username (User-Name attribute)
- NAS-IP-Address
- NAS-Port
- Service-Type
- All vendor-specific attributes

$$\text{Leaked}_{RADIUS} = L_{total} - L_{password} - L_{authenticator}$$

TACACS+ leaks only the 12-byte header (session ID, type, sequence number, flags, length).

---

## 5. AAA Transaction Timing Analysis

### Round-Trip Calculations

Each AAA function requires separate TCP exchanges. Total authentication + authorization time:

$$T_{total} = T_{TCP} + T_{authen} + T_{author} + T_{acct}$$

Where:
$$T_{TCP} = 1.5 \times RTT \text{ (three-way handshake)}$$

$$T_{authen} = n_{exchanges} \times RTT + T_{server}$$

$$T_{author} = RTT + T_{server}$$

For ASCII authentication (interactive, 3 exchanges):

$$T_{authen}^{ASCII} = 3 \times RTT + T_{server}$$

For PAP (single exchange):

$$T_{authen}^{PAP} = 1 \times RTT + T_{server}$$

| Component | PAP (1 exchange) | ASCII (3 exchanges) |
|:---|:---:|:---:|
| TCP handshake | 1.5 RTT | 1.5 RTT |
| Authentication | 1 RTT | 3 RTT |
| Authorization | 1 RTT | 1 RTT |
| Accounting start | 1 RTT | 1 RTT |
| **Total** | **4.5 RTT** | **6.5 RTT** |

With single-connection mode (TCP reuse), eliminate the handshake for subsequent sessions:

$$T_{subsequent} = T_{total} - T_{TCP} = T_{total} - 1.5 \times RTT$$

### Latency Examples

| RTT | PAP Total | ASCII Total | With Single-Connect (PAP) |
|:---:|:---:|:---:|:---:|
| 1 ms (LAN) | 4.5 ms | 6.5 ms | 3.0 ms |
| 10 ms (campus) | 45 ms | 65 ms | 30 ms |
| 50 ms (WAN) | 225 ms | 325 ms | 150 ms |
| 100 ms (intercont.) | 450 ms | 650 ms | 300 ms |

---

## 6. Scalability — Server Capacity Planning

### Connection Model

With $N$ network devices, each making $C$ AAA requests per minute:

$$\text{Requests/sec} = \frac{N \times C}{60}$$

Each TCP connection (without single-connection) requires server resources:

$$\text{Concurrent TCP} = \text{Requests/sec} \times T_{session}$$

| Devices ($N$) | Req/min ($C$) | Req/sec | Concurrent TCP (100ms sessions) |
|:---:|:---:|:---:|:---:|
| 100 | 10 | 16.7 | 1.7 |
| 500 | 10 | 83.3 | 8.3 |
| 1,000 | 20 | 333.3 | 33.3 |
| 5,000 | 20 | 1,666.7 | 166.7 |
| 10,000 | 30 | 5,000.0 | 500.0 |

### Single-Connection Optimization

With single-connection mode, each device maintains exactly one persistent TCP connection:

$$\text{TCP connections} = N$$

This is independent of request rate, reducing server overhead from $O(N \times C)$ to $O(N)$.

---

## 7. Command Authorization Complexity

### Permission Matrix

For $U$ users, $D$ devices, and $M$ commands, the full authorization matrix has:

$$\text{Rules} = U \times D \times M$$

| Users | Devices | Commands | Rules (no groups) | Rules (with roles) |
|:---:|:---:|:---:|:---:|:---:|
| 10 | 50 | 100 | 50,000 | 500 (10 roles) |
| 50 | 200 | 100 | 1,000,000 | 2,000 (20 roles) |
| 200 | 1,000 | 200 | 40,000,000 | 6,000 (30 roles) |

With role-based access control using $R$ roles:

$$\text{Rules} = (U \times R_{mapping}) + (R \times D_{groups} \times M)$$

This reduces from $O(U \times D \times M)$ to $O(U + R \times M)$, demonstrating why TACACS+ deployments always use group-based policies.

---

*The mathematics of TACACS+ reveal a protocol built for network device security in a pre-TLS era. Its MD5-based encryption is simple and fast but requires strong shared secrets to resist modern computational attacks. The separated AAA model adds latency but provides the granular per-command authorization that network operations teams depend on daily.*

## Prerequisites

- modular arithmetic and XOR operations, birthday problem probability, entropy and information theory

## Complexity

- **Beginner:** XOR encryption/decryption and MD5 pad generation
- **Intermediate:** Session collision probability and brute force timing analysis
- **Advanced:** Comparative security analysis and authorization matrix optimization
