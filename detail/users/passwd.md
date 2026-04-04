# The Mathematics of passwd — Password Hashing, Entropy & Authentication Cost

> *passwd is a cryptographic gate: it converts human-memorable strings into computationally expensive hashes. The security of every account depends on the entropy of the password and the cost of the hash function — both are quantifiable.*

---

## 1. Password Hash Functions

### Hash Algorithm Selection

`/etc/shadow` stores password hashes with an algorithm identifier:

| ID | Algorithm | Output Size | Default Rounds | Status |
|:---:|:---|:---:|:---:|:---|
| $1$ | MD5 | 128 bit | 1000 | Deprecated |
| $5$ | SHA-256 | 256 bit | 5000 | Acceptable |
| $6$ | SHA-512 | 512 bit | 5000 | Recommended |
| $y$ | yescrypt | Variable | Cost 5 | Modern default |
| $2b$ | bcrypt | 184 bit | 12 ($2^{12}$ rounds) | Strong |

### Hash Computation Cost

$$T_{hash} = rounds \times T_{algorithm\_iteration}$$

| Algorithm | Rounds | Time per Hash | Hashes/sec (CPU) |
|:---|:---:|:---:|:---:|
| MD5 ($1$) | 1000 | 0.5 ms | 2,000 |
| SHA-512 ($6$) | 5000 | 2.5 ms | 400 |
| SHA-512 ($6$) | 100000 | 50 ms | 20 |
| bcrypt ($2b$) | $2^{12}$ | 250 ms | 4 |
| yescrypt ($y$) | cost 5 | 50 ms | 20 |

### GPU Acceleration Resistance

| Algorithm | CPU Hash/s | GPU Hash/s | GPU Speedup |
|:---|:---:|:---:|:---:|
| MD5crypt | 2,000 | 10,000,000 | 5,000x |
| SHA-512crypt | 400 | 200,000 | 500x |
| bcrypt | 4 | 100 | 25x |
| yescrypt | 20 | 50 | 2.5x |

Memory-hard algorithms (yescrypt, Argon2) resist GPU attacks because GPUs have limited per-thread memory.

---

## 2. Password Entropy — Strength Quantification

### Entropy Formula

$$H = \log_2(N^L) = L \times \log_2(N)$$

Where:
- $H$ = entropy (bits)
- $N$ = size of character set
- $L$ = password length

### Character Set Sizes

| Characters | $N$ | Entropy/char |
|:---|:---:|:---:|
| Digits only | 10 | 3.32 bits |
| Lowercase | 26 | 4.70 bits |
| Lower + upper | 52 | 5.70 bits |
| Alphanumeric | 62 | 5.95 bits |
| Printable ASCII | 95 | 6.57 bits |

### Entropy Examples

| Password Type | Length | Charset | Entropy |
|:---|:---:|:---:|:---:|
| PIN | 4 | 10 | 13.3 bits |
| Simple word | 6 | 26 | 28.2 bits |
| Mixed case + digits | 8 | 62 | 47.6 bits |
| Passphrase (4 words) | ~20 | ~7776 (diceware) | 51.7 bits |
| Full ASCII random | 12 | 95 | 78.8 bits |
| Random alphanumeric | 16 | 62 | 95.3 bits |

### Minimum Entropy Recommendations

| Use Case | Min Entropy | Min Equivalent |
|:---|:---:|:---|
| Low-value account | 30 bits | 6 lowercase chars |
| Standard user | 50 bits | 8 mixed alphanumeric |
| Admin/root | 70 bits | 11 mixed alphanumeric |
| Service account (key) | 128 bits | 22 alphanumeric |

---

## 3. Brute Force Time — The Security Equation

### Time to Crack

$$T_{crack} = \frac{2^H}{hash\_rate}$$

For a password with entropy $H$ and hash rate $R$:

$$T_{crack\_avg} = \frac{2^{H-1}}{R} \text{ (expected, on average half the space)}$$

### Worked Examples (SHA-512, 5000 rounds)

Single modern GPU: ~200,000 hashes/s

| Entropy | Keyspace | Time (1 GPU) | Time (1000 GPUs) |
|:---:|:---:|:---:|:---:|
| 30 bits | $10^9$ | 1.5 hours | 5 seconds |
| 40 bits | $10^{12}$ | 64 days | 1.5 hours |
| 50 bits | $10^{15}$ | 178 years | 65 days |
| 60 bits | $10^{18}$ | 182,000 years | 182 years |
| 80 bits | $10^{24}$ | $10^{13}$ years | $10^{10}$ years |

### With bcrypt (cost 12)

Single GPU: ~100 hashes/s

$$T_{30bit} = \frac{2^{30}}{100} = 124 \text{ days (vs 1.5 hours with SHA-512)}$$

bcrypt provides ~2000x more resistance per hash for the same password.

---

## 4. Password Aging — Time-Based Policy

### Shadow File Fields

| Field | Meaning | Default |
|:---|:---|:---:|
| sp_lstchg | Last change (days since epoch) | Set on change |
| sp_min | Min days between changes | 0 |
| sp_max | Max days before must change | 99999 |
| sp_warn | Warning days before expiry | 7 |
| sp_inact | Grace period after expiry | -1 (disabled) |
| sp_expire | Account expiration date | -1 (never) |

### Password Lifecycle Timeline

$$T_0 = sp\_lstchg \text{ (password set)}$$

$$T_{can\_change} = T_0 + sp\_min$$

$$T_{warn} = T_0 + sp\_max - sp\_warn$$

$$T_{expired} = T_0 + sp\_max$$

$$T_{locked} = T_0 + sp\_max + sp\_inact$$

### Compliance Calculations

For a 90-day password policy with 14-day warning and 7-day grace:

$$T_{max} = 90 \text{ days}$$
$$T_{warn\_start} = 90 - 14 = 76 \text{ days after change}$$
$$T_{lockout} = 90 + 7 = 97 \text{ days after change}$$

$$password\_changes\_per\_year = \lceil 365 / sp\_max \rceil = \lceil 365/90 \rceil = 5$$

---

## 5. Salt — Uniqueness Guarantee

### Salt Purpose

$$hash(password, salt_1) \neq hash(password, salt_2) \quad \forall salt_1 \neq salt_2$$

### Salt Space

| Algorithm | Salt Size | Unique Salts |
|:---|:---:|:---:|
| MD5crypt | 8 chars (48 bits) | $2^{48} \approx 2.8 \times 10^{14}$ |
| SHA-512crypt | 16 chars (96 bits) | $2^{96} \approx 7.9 \times 10^{28}$ |
| bcrypt | 22 chars (128 bits) | $2^{128} \approx 3.4 \times 10^{38}$ |

### Rainbow Table Prevention

Without salt: precompute hash for every password → $O(1)$ lookup.

$$rainbow\_table\_size = |password\_space| \times hash\_size$$

With salt: must precompute for every password-salt pair:

$$rainbow\_table\_size = |password\_space| \times |salt\_space| \times hash\_size$$

For SHA-512: $|salt\_space| = 2^{96}$. Rainbow table would require $\approx 10^{40}$ entries — physically impossible.

---

## 6. PAM Integration — Authentication Pipeline

### PAM Stack Execution

passwd invokes PAM's password management stack:

$$result = \prod_{module \in stack} module(action, control)$$

| Control | Meaning | On Failure |
|:---|:---|:---|
| required | Must pass, continue checking | Fail (but continue) |
| requisite | Must pass, stop on failure | Fail immediately |
| sufficient | Pass = done, fail = continue | Continue |
| optional | Result ignored unless only module | Continue |

### Password Quality Check

pam_pwquality checks:

$$quality = f(length, classes, dictionary, similarity)$$

$$classes = |\{upper, lower, digit, special\} \cap chars(password)|$$

$$score = \begin{cases} fail & \text{if } length < minlen \\ fail & \text{if } classes < minclass \\ fail & \text{if } dictionary\_match = true \\ pass & \text{otherwise} \end{cases}$$

---

## 7. Account Recovery and Lockout

### Failed Attempt Lockout (pam_tally2/pam_faillock)

$$locked = (consecutive\_failures \geq deny) \land (T_{since\_last} < unlock\_time)$$

Default: deny=5 attempts, unlock_time=600 seconds.

### Lockout Impact

$$P(legitimate\_lockout) = P(\text{user forgets}) \times P(\text{tries } \geq deny)$$

$$support\_cost = P(lockout) \times N_{users} \times T_{reset}$$

With 1000 users, 5% lockout rate per month, 10 minutes per reset:

$$support\_cost = 0.05 \times 1000 \times 10 = 500 \text{ minutes/month}$$

---

## 8. Summary of passwd Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Entropy | $L \times \log_2(N)$ | Information theory |
| Crack time | $2^H / hash\_rate$ | Computational cost |
| Hash cost | $rounds \times T_{iteration}$ | Tunable work factor |
| Salt space | $2^{salt\_bits}$ | Rainbow table resistance |
| Password expiry | $lastchange + max\_days$ | Date arithmetic |
| Changes/year | $\lceil 365 / max\_days \rceil$ | Policy |
| GPU resistance | $CPU\_rate / GPU\_rate$ | Algorithm property |
| Lockout rate | $P(forget) \times P(tries \geq deny)$ | Probability |

## Prerequisites

- information theory (entropy), cryptographic hash functions, password aging, brute force complexity, key derivation functions

---

*passwd is where cryptography meets user experience. The hash function's cost parameter determines how long an attacker waits — and how long your users wait. The art is finding the sweet spot where authentication takes milliseconds but cracking takes millennia.*
