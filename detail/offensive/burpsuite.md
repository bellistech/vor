# The Mathematics of Burp Suite — Web Application Security Testing

> *Burp Suite is a web security testing platform that intercepts, modifies, and replays HTTP requests. Its effectiveness comes from systematic parameter fuzzing, payload permutation, and automated scanning that covers the combinatorial space of web attack vectors.*

---

## 1. Proxy Interception — Request/Response Analysis

### HTTP Request Decomposition

Every intercepted request is a tuple:

$$R = (\text{method}, \text{URL}, \text{headers}, \text{cookies}, \text{body})$$

### Parameter Injection Points

$$\text{Injection points} = |P_{URL}| + |P_{headers}| + |P_{cookies}| + |P_{body}|$$

| Component | Parameter Count (typical form) | Example |
|:---|:---:|:---|
| URL path segments | 2-5 | `/api/v1/users/123` |
| Query parameters | 3-10 | `?id=1&sort=name&page=2` |
| Cookie values | 2-8 | `session=abc; lang=en` |
| POST body fields | 5-20 | Form data or JSON keys |
| Custom headers | 1-5 | `X-Auth-Token: xxx` |

### Total Attack Surface per Request

$$A_{request} = \sum_{p \in \text{params}} |\text{payloads}(p)|$$

With 15 parameters and 100 payloads each: $A = 1{,}500$ test requests per endpoint.

---

## 2. Intruder — Automated Fuzzing

### Attack Types

| Type | Parameters | Requests | Use Case |
|:---|:---|:---:|:---|
| Sniper | 1 at a time | $\sum |P_i|$ | Test each parameter independently |
| Battering Ram | Same payload in all | $|P_1|$ | Same value everywhere |
| Pitchfork | Parallel lists | $\max(|P_i|)$ | Correlated parameters |
| Cluster Bomb | All combinations | $\prod |P_i|$ | Brute force / credential stuffing |

### Request Count Formulas

**Sniper** with $n$ parameters, each tested with $m$ payloads:

$$R_{sniper} = n \times m$$

**Cluster Bomb** with $n$ parameters of sizes $m_1, m_2, \ldots, m_n$:

$$R_{cluster} = \prod_{i=1}^{n} m_i$$

### Worked Example: Login Brute Force

Cluster bomb with username list (1000) and password list (10000):

$$R = 1000 \times 10{,}000 = 10{,}000{,}000 \text{ requests}$$

At 100 requests/second: $\frac{10^7}{100} = 100{,}000 \text{ seconds} = 27.8 \text{ hours}$

At 500 requests/second (Pro): $\frac{10^7}{500} = 20{,}000 \text{ seconds} = 5.6 \text{ hours}$

### Payload Processing Pipeline

$$\text{Final payload} = \text{Encode}(\text{Process}(\text{Match/Replace}(\text{raw payload})))$$

Each processing step can multiply request count:

| Processing | Multiplier | Example |
|:---|:---:|:---|
| No processing | 1x | Raw payloads |
| URL encoding | 1x | `<script>` → `%3Cscript%3E` |
| Double encoding | 2x | `%253Cscript%253E` |
| Case variation | $2^n$ per alpha char | `Script`, `SCRIPT`, `sCrIpT` |

---

## 3. Scanner — Automated Vulnerability Detection

### Scan Coverage

$$\text{Coverage} = \frac{|\text{endpoints scanned}|}{|\text{total endpoints}|} \times \frac{|\text{vuln classes tested}|}{|\text{total vuln classes}|}$$

### Vulnerability Classes Tested

| Class | Payloads per Parameter | Detection Method |
|:---|:---:|:---|
| SQL Injection | 50-200 | Error messages, time delays |
| XSS (Reflected) | 100-300 | Payload reflection in response |
| XSS (Stored) | 50-100 | Payload in subsequent responses |
| Command Injection | 30-80 | Time delays, DNS callbacks |
| Path Traversal | 20-50 | Known file content in response |
| SSRF | 10-30 | Collaborator callbacks |
| XXE | 10-20 | Collaborator callbacks, errors |
| SSTI | 20-40 | Math expression evaluation |

### Total Scan Requests

$$R_{scan} = |\text{endpoints}| \times |\text{params/endpoint}| \times \sum_{c \in \text{classes}} |\text{payloads}_c|$$

| Application Size | Endpoints | Params | Total Requests | Scan Time (100 req/s) |
|:---|:---:|:---:|:---:|:---:|
| Small (10 pages) | 30 | 5 | ~75,000 | 12.5 min |
| Medium (100 pages) | 300 | 8 | ~1.2M | 3.3 hours |
| Large (1000 pages) | 3,000 | 10 | ~15M | 41.7 hours |

---

## 4. Collaborator — Out-of-Band Detection

### The Problem

Some vulnerabilities don't produce visible responses (blind injection). Collaborator detects out-of-band interactions:

$$\text{Alert if: target server} \xrightarrow{\text{DNS/HTTP/SMTP}} \text{Collaborator server}$$

### Callback Types

| Protocol | Latency | Reliability | Data Exfil Capacity |
|:---|:---:|:---:|:---:|
| DNS | 1-5 seconds | 95%+ (hard to block) | 253 chars/query |
| HTTP | 1-10 seconds | 80% (may be firewalled) | Unlimited |
| SMTP | 5-30 seconds | 60% (often blocked) | Unlimited |

### Blind Detection Confidence

$$P(\text{true positive}) = P(\text{callback received} \mid \text{vuln exists}) \times P(\text{vuln exists} \mid \text{callback})$$

If callback is received: very high confidence (~95%) because the target had to resolve an attacker-controlled domain.

If no callback: vulnerability may still exist (egress filtering):

$$P(\text{false negative}) = P(\text{vuln exists}) \times P(\text{callback blocked})$$

---

## 5. Repeater — Manual Testing

### Iteration Efficiency

Repeater allows rapid hypothesis testing:

$$T_{test} = T_{modify} + T_{send} + T_{analyze}$$

| Testing Phase | Manual (curl) | Repeater | Speedup |
|:---|:---:|:---:|:---:|
| Modify request | 30 seconds | 5 seconds | 6x |
| Send + view response | 10 seconds | 2 seconds | 5x |
| Compare responses | 60 seconds | 10 seconds (Comparer) | 6x |
| Full test iteration | ~100 seconds | ~17 seconds | 6x |

### Response Analysis Techniques

| Technique | What to Compare | Indicates |
|:---|:---|:---|
| Status code diff | 200 vs 500 | Error-based detection |
| Response length diff | $\Delta > 100$ bytes | Content injection |
| Response time diff | $\Delta > 3$ seconds | Time-based blind |
| Header diff | New headers | Info disclosure |
| Body keyword search | Error messages, stack traces | Vulnerability confirmation |

---

## 6. Sequencer — Entropy Analysis

### Token Randomness Testing

Sequencer collects samples of session tokens and tests randomness:

$$H_{observed} = -\sum_{i} p_i \log_2(p_i) \text{ per bit position}$$

### Statistical Tests

| Test | What It Measures | Good Token | Bad Token |
|:---|:---|:---|:---|
| Character frequency | Uniform distribution | Chi-square $p > 0.01$ | $p < 0.01$ |
| Bit-level | Each bit is 50/50 | $|p - 0.5| < 0.05$ | $|p - 0.5| > 0.1$ |
| Serial correlation | Adjacent bits independent | $r < 0.1$ | $r > 0.3$ |
| FIPS 140-2 monobit | Roughly equal 0s and 1s | Pass | Fail |

### Token Predictability

If Sequencer finds low entropy:

$$\text{Brute force space} = 2^{H_{effective}} \ll 2^{H_{theoretical}}$$

A 128-bit token with only 32 bits of effective entropy:

$$\text{Brute force} = 2^{32} = 4.3 \times 10^9 \text{ (feasible in seconds)}$$

---

## 7. Extension Ecosystem

### BApp Store Categories

| Category | Extensions | Purpose |
|:---|:---:|:---|
| Scanner enhancement | ~30 | Additional vulnerability checks |
| Authentication | ~15 | Token handling, OAuth, JWT |
| Logging/reporting | ~20 | Export findings |
| Payload generation | ~25 | Custom wordlists, encoding |
| Automation | ~15 | Macro recording, scripting |

### Custom Extension Performance

| Language | Request Processing Speed | Use Case |
|:---|:---:|:---|
| Java (native) | ~10,000 req/s | High-performance scanning |
| Python (Jython) | ~1,000 req/s | Rapid prototyping |
| Ruby (JRuby) | ~1,000 req/s | Scripting |

---

## 8. Summary of Functions by Type

| Concept | Math Type | Application |
|:---|:---|:---|
| $n \times m$ (Sniper) | Linear | Parameter fuzzing |
| $\prod m_i$ (Cluster Bomb) | Combinatorial explosion | Brute force |
| Scan coverage ratio | Set coverage | Testing completeness |
| Out-of-band callback | Conditional probability | Blind detection |
| Shannon entropy $H$ | Information theory | Token analysis |
| $2^{H_{effective}}$ | Exponential (reduced) | Token predictability |
| Response diff $\Delta$ | Comparison | Vulnerability identification |

---

*Burp Suite transforms web security testing from manual experimentation into systematic coverage — the Intruder's combinatorial attack types, Scanner's automated payload library, and Collaborator's out-of-band detection together cover the full attack surface of a web application.*

## Prerequisites

- HTTP request/response interception (proxy model)
- Web application attack surface (parameters, headers, cookies)
- Combinatorial testing (Intruder attack types: sniper, battering ram, pitchfork, cluster bomb)

## Complexity

- **Beginner:** Proxy interception, Repeater manual testing, Target scope, HTTP history browsing
- **Intermediate:** Intruder attack types, Scanner configuration, match/replace rules, session handling, macro recording
- **Advanced:** Intruder payload combinatorics, Scanner crawl graph coverage, Collaborator out-of-band detection timing, extension API (Montoya) for custom checks
