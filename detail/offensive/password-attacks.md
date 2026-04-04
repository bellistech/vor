# The Mathematics of Password Attacks — Keyspace, Entropy, and Cracking Economics

> *Password security is fundamentally a math problem: keyspace size determines brute-force time, hash function cost determines cracking throughput, and rainbow tables trade space for time. Every password policy and hash choice has a quantifiable security margin.*

---

## 1. Keyspace Formulas

### Total Keyspace

For a password of length $L$ drawn from a character set of size $C$:

$$K = C^L$$

### Character Set Sizes

| Charset | Size ($C$) | Example |
|:---|:---:|:---|
| Digits only | 10 | 0-9 |
| Lowercase | 26 | a-z |
| Uppercase | 26 | A-Z |
| Mixed case | 52 | a-zA-Z |
| Alphanumeric | 62 | a-zA-Z0-9 |
| Printable ASCII | 95 | All keyboard characters |
| Full extended ASCII | 128 | + special chars |

### Keyspace by Password Length

| Length | Digits ($10^L$) | Lower ($26^L$) | Mixed+Digits ($62^L$) | Full ASCII ($95^L$) |
|:---:|:---:|:---:|:---:|:---:|
| 4 | 10,000 | 456,976 | $1.5 \times 10^7$ | $8.1 \times 10^7$ |
| 6 | $10^6$ | $3.1 \times 10^8$ | $5.7 \times 10^{10}$ | $7.4 \times 10^{11}$ |
| 8 | $10^8$ | $2.1 \times 10^{11}$ | $2.2 \times 10^{14}$ | $6.6 \times 10^{15}$ |
| 10 | $10^{10}$ | $1.4 \times 10^{14}$ | $8.4 \times 10^{17}$ | $6.0 \times 10^{19}$ |
| 12 | $10^{12}$ | $9.5 \times 10^{16}$ | $3.2 \times 10^{21}$ | $5.4 \times 10^{23}$ |
| 16 | $10^{16}$ | $4.4 \times 10^{22}$ | $4.8 \times 10^{28}$ | $4.4 \times 10^{31}$ |

---

## 2. Password Entropy

### Shannon Entropy

$$H = L \times \log_2(C) = \log_2(C^L) = \log_2(K)$$

| Password Type | $C$ | $L$ | Entropy ($H$) |
|:---|:---:|:---:|:---:|
| PIN (4 digits) | 10 | 4 | 13.3 bits |
| Common password | ~1,000 words | 1 | 10 bits |
| 8-char lowercase | 26 | 8 | 37.6 bits |
| 8-char full ASCII | 95 | 8 | 52.6 bits |
| 12-char alphanumeric | 62 | 12 | 71.5 bits |
| 4-word diceware | 7,776 | 4 | 51.7 bits |
| 6-word diceware | 7,776 | 6 | 77.5 bits |
| 8-word diceware | 7,776 | 8 | 103.4 bits |

### NIST SP 800-63B Recommendation

Minimum 8 characters, no composition rules, check against breach lists.

Effective entropy depends on **user behavior**, not theoretical keyspace:

$$H_{effective} \ll H_{theoretical}$$

Because users choose passwords from a tiny subset of the keyspace (dictionary words, patterns, personal info).

---

## 3. Time-to-Crack Tables

### Cracking Rate by Hash Algorithm

$$T_{crack} = \frac{K}{2 \times R_{hash}} \quad \text{(expected time, half the keyspace)}$$

GPU cracking rates (RTX 4090, single GPU):

| Algorithm | Rate (hashes/sec) | Notes |
|:---|:---:|:---|
| MD5 | $1.6 \times 10^{11}$ | Trivial |
| SHA-1 | $2.4 \times 10^{10}$ | Fast |
| SHA-256 | $1.0 \times 10^{10}$ | Fast |
| NTLM | $1.4 \times 10^{11}$ | Windows passwords |
| bcrypt (cost 10) | $3.2 \times 10^4$ | Memory-hard |
| bcrypt (cost 12) | $8 \times 10^3$ | Slower |
| scrypt (default) | $1.5 \times 10^3$ | Memory-hard |
| Argon2id (default) | $5 \times 10^2$ | Memory-hard |

### Worked Example: 8-Character Password (Full ASCII)

Keyspace: $95^8 = 6.63 \times 10^{15}$

| Hash | GPU Rate | Crack Time (avg) |
|:---|:---:|:---:|
| MD5 | $1.6 \times 10^{11}$/s | 5.8 hours |
| SHA-256 | $10^{10}$/s | 3.8 days |
| bcrypt (10) | $3.2 \times 10^4$/s | 3,282 years |
| Argon2id | $500$/s | 210 million years |

**The hash algorithm matters more than password complexity.**

---

## 4. Rainbow Tables — Space-Time Tradeoff

### Hellman's Tradeoff

A rainbow table precomputes hash chains:

$$\text{Time} \times \text{Space} = \text{Keyspace}^2$$

For a table with $m$ chains of length $t$, covering keyspace $N$:

$$m \times t \geq N$$

Storage: $m$ start-endpoint pairs. Lookup: at most $t$ hash computations.

### Rainbow Table Sizes

| Keyspace | Chain Length | Chains | Table Size | Lookup Time |
|:---:|:---:|:---:|:---:|:---:|
| $2^{28}$ (LM hash) | 4,000 | 67,000 | 1 MB | < 1 sec |
| $2^{36}$ (6-char alphanum) | 10,000 | 6.9M | 100 MB | < 5 sec |
| $2^{48}$ (8-char lower) | 100,000 | 2.8B | 40 GB | < 30 sec |
| $2^{53}$ (8-char full) | 200,000 | 45B | 640 GB | < 2 min |

### Salt Defeats Rainbow Tables

A salt $s$ means the hash is $H(s \| p)$. Each unique salt requires a separate table:

$$\text{Total storage} = |\text{salts}| \times \text{table size}$$

With 128-bit random salts: $2^{128}$ tables needed — computationally impossible.

**Modern hash functions (bcrypt, Argon2) always use salts.** Rainbow tables are only viable against unsalted hashes (MD5, old NTLM).

---

## 5. Dictionary and Rule-Based Attacks

### Dictionary Attack Speed

$$T_{dict} = \frac{|D| \times |R|}{R_{hash}}$$

Where $|D|$ is dictionary size and $|R|$ is rule count (mutations per word).

| Dictionary | Words | Rules | Candidates | MD5 Time | bcrypt Time |
|:---|:---:|:---:|:---:|:---:|:---:|
| rockyou.txt | 14M | 1 | 14M | < 1 sec | 7 min |
| rockyou + best64 | 14M | 64 | 896M | 6 sec | 7.7 hours |
| rockyou + dive | 14M | 36K | 504B | 53 min | 500 years |
| CrackStation | 1.5B | 1 | 1.5B | 9 sec | 13 hours |

### Common Hashcat Rules

| Rule | Example | Multiplier |
|:---|:---|:---:|
| best64 | Toggle case, append digits | 64x |
| d3ad0ne | Extensive mutations | 34,000x |
| dive | Deep mutation rules | 36,000x |
| OneRuleToRuleThemAll | Optimized coverage | 52,000x |

### Password Pattern Distribution

From breach analysis (RockYou, LinkedIn, etc.):

| Pattern | Example | Frequency |
|:---|:---|:---:|
| Lowercase + digits | password123 | 35% |
| First capital + digits | Password1 | 25% |
| All lowercase | password | 20% |
| Mixed with trailing ! or # | P@ssword1! | 10% |
| Truly random | kX9$mQ2& | <5% |

95% of passwords follow predictable patterns — dictionary + rules cracks most passwords faster than brute force.

---

## 6. bcrypt Cost Factor Analysis

### Iteration Count

bcrypt uses a cost factor $c$ where iterations $= 2^c$:

$$T_{hash} = 2^c \times T_{base}$$

| Cost ($c$) | Iterations | Hash Time (CPU) | GPU Rate (RTX 4090) |
|:---:|:---:|:---:|:---:|
| 4 | 16 | 1 ms | $5 \times 10^5$ |
| 8 | 256 | 16 ms | $3.2 \times 10^4$ |
| 10 | 1,024 | 65 ms | $8 \times 10^3$ |
| 12 | 4,096 | 260 ms | $2 \times 10^3$ |
| 14 | 16,384 | 1 sec | $5 \times 10^2$ |
| 16 | 65,536 | 4 sec | $1.3 \times 10^2$ |

### Optimal Cost Factor

Target: hash time ~250ms (acceptable for login UX):

$$c = \lfloor \log_2(0.25 / T_{base}) \rfloor \approx 12 \text{ on modern hardware}$$

Each cost increase by 1 doubles both legitimate hash time AND attacker cost.

### Argon2id Parameters

Argon2id adds memory-hardness:

$$\text{Cost} = t \times m \times p$$

Where $t$ = iterations, $m$ = memory (KB), $p$ = parallelism.

| Config | Time | Memory | GPU Resistance |
|:---|:---:|:---:|:---|
| t=1, m=64MB, p=4 | 250 ms | 64 MB | High (memory-bound) |
| t=3, m=64MB, p=4 | 750 ms | 64 MB | Very high |
| t=1, m=256MB, p=4 | 500 ms | 256 MB | Extreme |

A GPU with 24 GB VRAM can only run $\lfloor 24576/64 \rfloor = 384$ parallel Argon2 instances at 64 MB — vs. millions of MD5 instances.

---

## 7. Credential Stuffing — Breach Reuse

### Reuse Statistics

$$P(\text{password reuse}) \approx 0.52 \text{ (52% of users reuse passwords)}$$

### Attack Efficiency

Given a breach database of $B$ credentials and a target site with $U$ users:

$$\text{Expected matches} = B \times \frac{U}{N_{internet}} \times P(\text{reuse}) \times P(\text{same password})$$

Simplified: if 1% of breach entries match target users, and 52% reuse passwords:

$$\text{Success rate} \approx 0.01 \times 0.52 = 0.52\%$$

At 1 million credential pairs: ~5,200 account takeovers.

### Rate Limiting Defense

$$\text{Effective rate} = \min(R_{attack}, R_{limit})$$

At rate limit of 10 attempts/IP/minute with 1000 rotating IPs:

$$R_{effective} = 10 \times 1000 = 10{,}000 \text{/min}$$

To try 1M credentials: $\frac{10^6}{10{,}000} = 100 \text{ minutes}$.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $C^L$ keyspace | Exponential | Brute force space |
| $L \times \log_2(C)$ | Logarithmic | Password entropy |
| $K / (2R)$ | Linear time | Expected crack time |
| $m \times t = N$ | Space-time tradeoff | Rainbow tables |
| $2^c$ iterations | Exponential cost | bcrypt difficulty |
| Salt $\times$ table size | Multiplication | Rainbow table defeat |
| $B \times P(\text{reuse})$ | Probability | Credential stuffing |

---

*Password security is not about making uncrackable passwords — it's about making cracking uneconomical. The right hash function (bcrypt/Argon2) with appropriate cost parameters makes brute force take centuries regardless of password choice, while salts make precomputation impossible.*

## Prerequisites

- Information entropy and keyspace calculation
- Cryptographic hash functions (MD5, SHA, bcrypt, Argon2)
- Time-memory trade-off (rainbow tables, precomputation)

## Complexity

- **Beginner:** Dictionary attacks, basic brute force, common wordlists, online vs offline attacks
- **Intermediate:** Rule-based attacks (hashcat rules), mask attacks, hash identification, credential stuffing, salted hash cracking
- **Advanced:** Keyspace exhaustion calculations, GPU throughput modeling, hash function cost parameter tuning, rainbow table size/coverage trade-offs
