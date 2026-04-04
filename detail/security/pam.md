# The Mathematics of PAM — Pluggable Authentication Modules

> *PAM is a sequential rule evaluation engine: each authentication request traverses an ordered stack of modules, and the final decision is a Boolean function of individual module results combined through control flags. The security of the system depends on stack ordering, module interaction, and the probability of each authentication factor.*

---

## 1. PAM Stack — Sequential Evaluation

### Stack Model

A PAM stack is an ordered list of (control, module) pairs:

$$\text{Stack} = [(c_1, m_1), (c_2, m_2), \ldots, (c_n, m_n)]$$

Each module returns one of:

| Return Code | Meaning |
|:---|:---|
| PAM_SUCCESS | Module succeeded |
| PAM_AUTH_ERR | Authentication failed |
| PAM_USER_UNKNOWN | User not found |
| PAM_MAXTRIES | Max attempts exceeded |
| PAM_IGNORE | Module not applicable |

### Control Flags

| Control | On Success | On Failure | Effect |
|:---|:---|:---|:---|
| required | Continue | Continue (but remember fail) | Must pass, but evaluates all |
| requisite | Continue | Return FAIL immediately | Must pass, fail-fast |
| sufficient | Return SUCCESS (if no prior required fail) | Continue | Can short-circuit success |
| optional | Continue | Continue | Only matters if sole module |

### Decision Function

$$\text{Result} = \begin{cases} \text{SUCCESS} & \text{if no required/requisite failed AND (at least one succeeded)} \\ \text{FAIL} & \text{if any required/requisite failed} \end{cases}$$

---

## 2. Multi-Factor Authentication — Probability

### Security of Stacked Factors

With $n$ independent authentication factors, each with bypass probability $p_i$:

$$P(\text{bypass all}) = \prod_{i=1}^{n} p_i$$

### Worked Example: Two-Factor Auth

| Factor | Type | $P(\text{bypass})$ |
|:---|:---|:---:|
| Password (pam_unix) | Knowledge | $10^{-4}$ (strong password) |
| TOTP (pam_oath) | Possession | $10^{-6}$ (30s window, 6 digits) |

$$P(\text{bypass 2FA}) = 10^{-4} \times 10^{-6} = 10^{-10}$$

Single factor: 1 in 10,000. Two factors: 1 in 10 billion.

### TOTP Mathematics

Time-based One-Time Password (RFC 6238):

$$\text{TOTP}(K, T) = \text{Truncate}(\text{HMAC-SHA1}(K, \lfloor T/30 \rfloor))$$

The truncated output is 6 digits:

$$\text{OTP} \in [0, 999999] \quad P(\text{guess}) = \frac{1}{10^6} = 10^{-6}$$

With a 30-second window and 1-step tolerance (90 seconds effective):

$$P(\text{brute force in window}) = \frac{\text{attempts} \times 3}{10^6}$$

At 10 attempts/second: $P = \frac{10 \times 90 \times 3}{10^6} = 0.0027 = 0.27\%$

Rate limiting to 3 attempts makes this negligible.

---

## 3. PAM Module Types

### Four Module Interfaces

| Type | Purpose | When Invoked |
|:---|:---|:---|
| auth | Verify identity | Login, sudo, su |
| account | Authorization/access control | After auth succeeds |
| password | Credential management | passwd, password change |
| session | Setup/teardown | Login shell, SSH session |

### Evaluation Independence

Each type has its own stack — they evaluate independently:

$$\text{Login} = \text{auth}() \land \text{account}() \land \text{session}()$$
$$\text{Password change} = \text{auth}() \land \text{password}()$$

### Stack Depth by Service

| Service | auth Stack | account Stack | Typical Total |
|:---|:---:|:---:|:---:|
| login | 3-5 modules | 2-3 modules | 10-15 |
| sshd | 3-5 modules | 2-3 modules | 10-15 |
| sudo | 2-3 modules | 1-2 modules | 5-8 |
| su | 2-3 modules | 1-2 modules | 5-8 |

---

## 4. Common Stack Patterns

### Pattern 1: Password + TOTP (2FA)

```
auth required   pam_unix.so
auth required   pam_oath.so usersfile=/etc/oath/users.oath
```

Both must succeed: $P(\text{auth}) = P(\text{password}) \times P(\text{TOTP})$

### Pattern 2: Password OR Smartcard

```
auth sufficient pam_pkcs11.so
auth required   pam_unix.so
```

Either can succeed: $P(\text{auth}) = 1 - (1-P(\text{card})) \times (1-P(\text{password}))$

### Pattern 3: Failover Authentication

```
auth [success=2 default=ignore] pam_unix.so
auth [success=1 default=ignore] pam_ldap.so
auth requisite                   pam_deny.so
auth required                    pam_permit.so
```

This uses **jump syntax** for complex flow control, equivalent to:

$$\text{Auth} = \text{pam\_unix} \lor \text{pam\_ldap}$$

### Stack Vulnerability: Order Matters

Consider:
```
auth sufficient pam_permit.so    # INSECURE: grants access to everyone
auth required   pam_unix.so
```

The `sufficient` pam_permit.so succeeds for ALL users, short-circuiting the password check. Module ordering is critical.

---

## 5. pam_unix — Password Hashing

### Hash Algorithms

| Algorithm | Hash ID | Iterations | Salt | Output | Security |
|:---|:---:|:---:|:---:|:---:|:---|
| DES (legacy) | (none) | 25 | 2 chars | 13 chars | Broken |
| MD5 | $1$ | 1,000 | 8 chars | 22 chars | Weak |
| SHA-256 | $5$ | 5,000 (default) | 16 chars | 43 chars | Moderate |
| SHA-512 | $6$ | 5,000 (default) | 16 chars | 86 chars | Moderate |
| yescrypt | $y$ | tunable | 16 chars | variable | Strong |
| bcrypt | $2b$ | $2^n$ | 22 chars | 31 chars | Strong |

### Cost Factor Comparison

$$T_{hash} = \text{iterations} \times T_{base}$$

| Algorithm | Iterations | Hash Time (CPU) | GPU Cracking Rate |
|:---|:---:|:---:|:---:|
| MD5 (1000 rounds) | 1,000 | 0.002 ms | 10B/s |
| SHA-512 (5000) | 5,000 | 0.01 ms | 500M/s |
| SHA-512 (500000) | 500,000 | 1 ms | 5M/s |
| bcrypt (cost 12) | 4,096 | 250 ms | 25K/s |
| yescrypt | tunable | 100-500 ms | 1-10K/s |

### Memory-Hard Functions

yescrypt and bcrypt are **memory-hard**: GPU parallelism is limited by memory bandwidth, not compute:

$$\text{GPU speedup} = \frac{\text{GPU cores}}{\text{memory bandwidth limit}} \ll \text{GPU cores}$$

---

## 6. Account Controls — Access Restrictions

### pam_access: Source-Based Control

$$\text{access}(u, s) = \begin{cases} \text{allow} & \text{if } (u, s) \in \text{allow list} \\ \text{deny} & \text{otherwise} \end{cases}$$

Where $u$ = user/group and $s$ = source (host, IP, tty).

### pam_time: Time-Based Control

$$\text{access}(u, t) = \begin{cases} \text{allow} & \text{if } t \in \text{allowed hours for } u \\ \text{deny} & \text{otherwise} \end{cases}$$

### pam_limits: Resource Limits

| Resource | Soft Limit | Hard Limit | Purpose |
|:---|:---:|:---:|:---|
| nproc | 1024 | 4096 | Fork bomb prevention |
| nofile | 1024 | 65536 | File descriptor limit |
| maxlogins | 3 | 5 | Concurrent session limit |
| maxsyslogins | 10 | 20 | Total system logins |

Fork bomb mitigation: nproc limit $n$ stops the exponential growth $2^k$ at $k = \lceil \log_2(n) \rceil$:

$$\text{Max processes from fork bomb} = n \text{ (instead of } 2^k \to \infty \text{)}$$

---

## 7. PAM Configuration Security Audit

### Stack Completeness Check

For each service file, verify:

$$\text{Secure} \iff \begin{cases} \text{auth stack terminates (has required or requisite)} \\ \text{No pam\_permit without guard} \\ \text{pam\_deny exists as fallback} \\ \text{account module present (not auth-only)} \end{cases}$$

### Common Misconfigurations

| Misconfiguration | Risk | Detection |
|:---|:---|:---|
| Missing `required` in auth | Auth bypass possible | No required/requisite in stack |
| `sufficient pam_permit` without guard | Universal access | pam_permit as first sufficient |
| No `pam_deny` fallback | Implicit allow | Last rule not deny or required |
| LDAP as only auth source | Single point of failure | No local fallback |

### Configuration Entropy

$$H_{config} = \log_2(|\text{possible configurations}|)$$

With 10 modules, 4 control flags, 4 service types:

$$|\text{configs}| = (4 \times 10)^4 \times 10! = \text{enormous}$$

This is why PAM misconfiguration is a top-10 Linux security issue — the configuration space is vast and most configurations are insecure.

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| Stack evaluation | Sequential Boolean | Authentication decision |
| $\prod p_i$ | Probability product | Multi-factor security |
| TOTP $1/10^6$ | Uniform probability | One-time password strength |
| Control flags | Branching logic | Stack flow control |
| Hash iterations | Linear cost | Password stretching |
| nproc $\lceil \log_2 n \rceil$ | Logarithmic bound | Fork bomb limit |
| Access $(u,s)$ predicate | Boolean function | Source-based control |

## Prerequisites

- Boolean logic, stack evaluation, set theory

---

*PAM is the gatekeeper function for every authentication decision on a Linux system — its stack-based evaluation model means the difference between a secure system and an open door is often a single line in a configuration file, evaluated in microseconds at every login attempt.*
