# Secure SDLC — Theory, Methodologies, and Maturity

> *Security cannot be bolted on after development. The cost of fixing a vulnerability increases exponentially as it progresses through the SDLC — from 1x at requirements to 30x in production. Integrating security at every phase transforms it from a gate to a continuous practice.*

---

## 1. Security Integration at Each SDLC Phase

### Requirements Phase

Security requirements define what the system must enforce:

**Functional security requirements:**
- Authentication: "The system shall support MFA for all privileged accounts"
- Authorization: "Users shall only access records within their organizational unit"
- Audit: "All data modifications shall be logged with user, timestamp, and old/new values"

**Non-functional security requirements:**
- "Passwords shall be stored using Argon2id with minimum 64MB memory cost"
- "TLS 1.3 shall be required for all external communications"
- "Session tokens shall expire after 15 minutes of inactivity"

**Abuse cases** (misuse cases) complement use cases:

| Use Case | Abuse Case | Mitigation |
|:---|:---|:---|
| User logs in | Attacker brute-forces credentials | Rate limiting, account lockout, MFA |
| User uploads file | Attacker uploads malicious payload | File type validation, virus scanning, sandboxed processing |
| User views report | Attacker accesses another user's report | Server-side authorization, IDOR prevention |

### Design Phase

**Secure design principles** (Saltzer & Schroeder, 1975):

1. **Economy of mechanism:** Keep security mechanisms simple
2. **Fail-safe defaults:** Deny by default
3. **Complete mediation:** Check every access (no caching of authorization decisions without expiry)
4. **Open design:** Security should not depend on secrecy of the mechanism
5. **Separation of privilege:** Require multiple conditions for access
6. **Least privilege:** Minimum necessary permissions
7. **Least common mechanism:** Minimize shared mechanisms
8. **Psychological acceptability:** Security must not make the system unusable

**Attack surface analysis** quantifies exposure:

$$\text{Attack Surface} = \sum_{i} w_i \times \text{count}(e_i)$$

Where $e_i$ are entry/exit points (network ports, API endpoints, file parsers, admin interfaces) and $w_i$ are weights based on privilege level and trust boundary crossing.

### Implementation Phase

**Taint analysis** (the theoretical basis of SAST):

Data from untrusted sources is "tainted." Taint propagates through assignments and operations. If tainted data reaches a security-sensitive sink without passing through a sanitizer, a vulnerability exists.

$$\text{Source} \xrightarrow{\text{propagation}} \text{Sink} \implies \text{vulnerability}$$
$$\text{Source} \xrightarrow{\text{propagation}} \text{Sanitizer} \xrightarrow{\text{clean}} \text{Sink} \implies \text{safe}$$

### Testing Phase

Security testing validates that security requirements are met and no unintended vulnerabilities exist. Three complementary approaches:

| Approach | Finds | Misses |
|:---|:---|:---|
| SAST | Code-level flaws (injection patterns, weak crypto) | Runtime behavior, config issues |
| DAST | Runtime vulnerabilities (XSS, auth bypass) | Internal code issues, logic flaws |
| IAST | Real data flow vulnerabilities | Only exercised code paths |

### Deployment Phase

Hardening checklist:
- Remove default accounts and sample data
- Disable unnecessary services and ports
- Apply CIS benchmark configurations
- Verify TLS certificates and cipher suites
- Confirm secrets are in vault, not config files
- Container image scanning (no known CVEs above threshold)

### Maintenance Phase

**Security debt** accumulates when:
- Vulnerabilities are deferred ("will fix next sprint")
- Dependencies are not updated
- Security configurations drift from baseline
- Logging/monitoring gaps are not addressed

---

## 2. Threat Modeling Methodologies — Comparison

### STRIDE vs PASTA vs VAST vs LINDDUN

| Aspect | STRIDE | PASTA | VAST | LINDDUN |
|:---|:---|:---|:---|:---|
| Focus | Technical threats | Risk-centric | Scalable/agile | Privacy threats |
| Approach | Per-element analysis | 7-stage process | Visual + automated | Privacy-specific |
| Output | Threat list per component | Risk-rated attack scenarios | Threat models per agile story | Privacy threat trees |
| Effort | Medium | High | Low per iteration | Medium |
| Best for | Application/system design | Business-critical systems | DevOps/agile teams | GDPR compliance |

### STRIDE in Depth

STRIDE maps to security properties and standard mitigations:

| Threat | Property | Standard Mitigation |
|:---|:---|:---|
| Spoofing | Authentication | Cryptographic authentication, MFA, certificate pinning |
| Tampering | Integrity | Digital signatures, MACs, input validation, checksums |
| Repudiation | Non-repudiation | Audit logs, digital signatures, timestamps, secure log storage |
| Info Disclosure | Confidentiality | Encryption (at rest + in transit), access control, data classification |
| Denial of Service | Availability | Rate limiting, redundancy, CDN, input size limits |
| Elevation of Privilege | Authorization | Least privilege, sandboxing, input validation, RBAC |

### Attack Trees — Formal Structure

An attack tree $T$ is a rooted tree where:
- Root node = attacker's goal
- Leaf nodes = atomic attack steps
- Internal nodes = decomposition (AND/OR)

**Quantitative analysis** assigns metrics to leaf nodes:

| Metric | Meaning |
|:---|:---|
| Cost ($) | Resources needed |
| Time (days) | Duration |
| Skill (1-10) | Technical expertise required |
| Detection (%) | Probability of detection |

For OR nodes: $\text{metric}_{parent} = \min(\text{metrics}_{children})$ (attacker picks easiest path)

For AND nodes: $\text{metric}_{parent} = \sum(\text{metrics}_{children})$ (all steps required)

The **minimum cost attack path** is the leaf-to-root path through OR nodes minimizing cost.

---

## 3. OWASP ASVS (Application Security Verification Standard)

### Verification Levels

| Level | Target | Effort |
|:---|:---|:---|
| L1 | All applications (minimum baseline) | Automated tools + basic manual |
| L2 | Applications handling sensitive data | Thorough testing + design review |
| L3 | Critical applications (healthcare, financial, military) | Deep code review + architecture analysis |

### ASVS Categories (v4.0)

| # | Category | L1 | L2 | L3 |
|:---:|:---|:---:|:---:|:---:|
| V1 | Architecture, Design, Threat Modeling | 5 | 13 | 17 |
| V2 | Authentication | 14 | 22 | 28 |
| V3 | Session Management | 7 | 11 | 14 |
| V4 | Access Control | 8 | 14 | 16 |
| V5 | Validation, Sanitization, Encoding | 12 | 22 | 26 |
| V6 | Stored Cryptography | 4 | 8 | 10 |
| V7 | Error Handling and Logging | 5 | 9 | 12 |
| V8 | Data Protection | 6 | 10 | 14 |
| V9 | Communications | 3 | 5 | 6 |
| V10 | Malicious Code | 2 | 4 | 6 |
| V11 | Business Logic | 4 | 8 | 10 |
| V12 | File and Resources | 5 | 8 | 10 |
| V13 | API and Web Service | 6 | 12 | 16 |
| V14 | Configuration | 5 | 8 | 10 |

Total requirements increase from ~100 (L1) to ~200+ (L3).

---

## 4. SAST vs DAST Theory

### SAST — Taint Analysis and Abstract Interpretation

**Taint analysis** tracks untrusted data through the program:

1. **Source identification:** Mark inputs as tainted (HTTP parameters, file reads, database results from user data)
2. **Propagation rules:** Taint flows through assignments, concatenations, format strings
3. **Sanitizer identification:** Specific functions remove taint (parameterized query, HTML encoder)
4. **Sink detection:** Security-sensitive operations (SQL query, HTML output, file path, OS command)

**Abstract interpretation** approximates program behavior:
- Over-approximation: considers all possible execution paths (sound but imprecise — false positives)
- Under-approximation: considers subset of paths (precise but incomplete — false negatives)

**Data flow analysis types:**

| Type | Direction | Tracks | Example |
|:---|:---|:---|:---|
| Forward | Source → Sink | Where tainted data goes | Taint analysis |
| Backward | Sink → Source | What feeds into a sink | Demand-driven taint |
| Interprocedural | Across functions | Full call chain | Context-sensitive |
| Intraprocedural | Within function | Single function | Fast but incomplete |

**Symbolic execution** explores all paths by treating inputs as symbolic variables:
- Builds a constraint system for each path
- Uses SMT solver (Z3) to determine feasibility
- Sound but suffers from path explosion: $O(2^n)$ paths for $n$ branches

### DAST — Runtime Analysis

**Crawling:** Discovers application structure (pages, forms, APIs).

**Active scanning:** Sends crafted payloads and analyzes responses:
- SQL injection: `' OR 1=1--`, `" OR ""="`, time-based blind
- XSS: `<script>alert(1)</script>`, event handlers, SVG payloads
- Path traversal: `../../etc/passwd`, `....//....//etc/passwd`
- Command injection: `; id`, `| whoami`, `` `id` ``

**Response analysis:**
- Error-based: Stack traces, SQL errors in response
- Time-based: Response delay indicates successful injection
- Differential: Compare response with normal vs malicious input
- Out-of-band: DNS/HTTP callbacks from injected payloads

**Limitations:**
- Cannot see source code (black box)
- Coverage depends on crawling effectiveness
- Authentication/authorization testing requires configuration
- Cannot find logic flaws without specific test cases

---

## 5. Fuzzing Theory

### Coverage-Guided Fuzzing

**Concept:** Mutate inputs; if a mutation triggers new code coverage, keep it as a seed for further mutation.

**Algorithm (simplified AFL approach):**

```
seeds = initial_corpus
while time_remaining:
    input = pick_random(seeds)
    mutated = mutate(input)  # bit flip, insert, delete, havoc
    coverage_before = get_coverage()
    run_target(mutated)
    coverage_after = get_coverage()
    if coverage_after > coverage_before:
        seeds.add(mutated)  # new coverage = interesting input
    if crash_detected():
        save_crash(mutated)
```

**Coverage metrics:**
- Edge coverage: unique source→destination basic block transitions
- Path coverage: unique sequences of edges
- Edge frequency: hit count buckets (1, 2, 4, 8, 16, 32, 64, 128+)

**Key fuzzers:**
- AFL/AFL++: Binary instrumentation, deterministic + havoc mutations
- libFuzzer: In-process, LLVM-based, coverage-guided
- Honggfuzz: Multi-process, hardware-based coverage (Intel PT)
- go-fuzz / native Go fuzzing: Go-specific corpus-based fuzzing

### Grammar-Based Fuzzing

For structured inputs (parsers, protocols, file formats):

**Grammar definition** describes valid input structure:
```
<html> ::= <tag> <content> </tag>
<tag>  ::= "<" <name> <attrs> ">"
<name> ::= [a-zA-Z]+
<attrs> ::= (<attr>)*
<attr> ::= <name> "=" "\"" <value> "\""
```

**Mutation strategies:**
- Subtree replacement: swap grammar subtrees
- Recursive expansion: deeply nest rules
- Boundary values: max length strings, empty fields
- Type confusion: integer where string expected

### Fuzzing Effectiveness

**Theoretical coverage bound:** For a program with $P$ reachable paths, random fuzzing discovers paths at rate:

$$E[\text{new paths}] \approx P \cdot (1 - (1 - 1/P)^n)$$

After $n$ test cases, approaching $P$ logarithmically. Coverage-guided fuzzing accelerates this by preferentially exploring uncovered regions.

---

## 6. DevSecOps Maturity Model

### Maturity Levels

| Level | Name | Characteristics |
|:---|:---|:---|
| 0 | Ad Hoc | No security integration; manual, reactive |
| 1 | Basic | SAST/DAST in CI; manual reviews; basic training |
| 2 | Managed | Automated gates; SCA + SBOM; security champions; metrics |
| 3 | Optimized | Shift-left complete; continuous monitoring; threat modeling standard; security debt tracked |
| 4 | Innovative | Risk-adaptive pipelines; ML-assisted triage; supply chain verification; proactive threat hunting |

### Metrics for Each Level

| Metric | L0 | L1 | L2 | L3 | L4 |
|:---|:---|:---|:---|:---|:---|
| Mean time to remediate critical | >90 days | <60 | <30 | <14 | <7 |
| % apps with threat models | 0% | 10% | 50% | 90% | 100% |
| False positive rate | N/A | 60%+ | <40% | <20% | <10% |
| Security test coverage | 0% | 30% | 60% | 80% | 95%+ |
| SBOM generation | None | Manual | Automated | Verified | Attested |

---

## 7. Security Debt

### Definition

Security debt is the accumulation of known security issues that are deferred rather than remediated:

$$\text{Security Debt} = \sum_{i} \text{risk}(v_i) \times \text{age}(v_i) \times \text{exposure}(v_i)$$

### Categories

| Category | Example | Impact |
|:---|:---|:---|
| Code debt | Deprecated crypto functions still in use | Direct vulnerability |
| Architecture debt | Missing rate limiting, no input validation layer | Systemic weakness |
| Dependency debt | Known CVEs in transitive dependencies | Supply chain risk |
| Configuration debt | Default passwords, debug endpoints in prod | Easy exploitation |
| Knowledge debt | No threat model, no security requirements | Unknown unknowns |

### Management Strategy

1. **Inventory:** Track all known security issues in a single backlog
2. **Classify:** Use CVSS + business context to prioritize
3. **Budget:** Allocate 20% of sprint capacity to security debt
4. **Trend:** Track debt growth rate vs reduction rate
5. **Threshold:** Define maximum acceptable debt level per service

---

## 8. Software Supply Chain Security

### Threat Landscape

| Attack Vector | Example | Mitigation |
|:---|:---|:---|
| Typosquatting | `lodas` instead of `lodash` | Namespace reservation, lockfiles |
| Dependency confusion | Private package name hijacked on public registry | Scoped registries, namespace claiming |
| Compromised maintainer | `event-stream` (2018) | Dependency review, lockfile auditing |
| Build system compromise | SolarWinds (2020) | Reproducible builds, build provenance |
| Registry compromise | Codecov (2021) | Artifact signing, checksum verification |

### SLSA Framework (Supply-chain Levels for Software Artifacts)

| Level | Requirements |
|:---|:---|
| SLSA 1 | Build process is documented |
| SLSA 2 | Build service generates provenance; version control |
| SLSA 3 | Hardened build platform; non-falsifiable provenance |
| SLSA 4 | Two-person review; hermetic, reproducible builds |

### Verification Chain

```
Developer signs commit (GPG/SSH)
    → CI verifies commit signature
    → Build runs in isolated environment
    → Build generates SBOM + provenance attestation
    → Artifacts signed (Sigstore/cosign)
    → Registry stores signatures alongside artifacts
    → Deployment verifies signature + attestation
    → Runtime policy enforces signed-only images
```

### Sigstore Verification

The transparency log (Rekor) provides:
- Public, append-only record of signing events
- Anyone can verify an artifact was signed by a specific identity
- Keyless signing via OIDC identity (no long-lived keys to manage)

Verification proves: **who** built **what**, **when**, on **which** build system.

---

## References

- Saltzer, Schroeder. "The Protection of Information in Computer Systems" (1975)
- McGraw, G. "Software Security: Building Security In" (2006)
- OWASP ASVS v4.0: Application Security Verification Standard
- OWASP SAMM v2.0: Software Assurance Maturity Model
- NIST SP 800-218: Secure Software Development Framework (SSDF)
- NIST SP 800-161r1: Supply Chain Risk Management
- SLSA Framework: https://slsa.dev
- Shostack, A. "Threat Modeling: Designing for Security" (2014)
- AFL Technical Whitepaper (Michal Zalewski)
- Sigstore: https://sigstore.dev
