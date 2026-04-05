# Secure SDLC

> Integrating security practices into every phase of software development — from requirements gathering through design, coding, testing, deployment, and maintenance.

## Secure SDLC Phases

```
Phase              Security Activities
─────              ───────────────────
Requirements       Security requirements, abuse cases, risk assessment
                   compliance requirements, data classification

Design             Threat modeling, security architecture review,
                   secure design patterns, attack surface analysis

Implementation     Secure coding standards, SAST, code review,
                   dependency scanning (SCA), secrets detection

Testing            DAST, IAST, penetration testing, fuzzing,
                   security regression tests, SBOM generation

Deployment         Hardening, configuration review, secrets mgmt,
                   container scanning, infrastructure as code review

Maintenance        Vulnerability management, patching, monitoring,
                   incident response, security debt tracking
```

## Threat Modeling

### STRIDE

```
Threat                  Property Violated    Example
──────                  ─────────────────    ───────
Spoofing                Authentication       Forged credentials
Tampering               Integrity            Modified request
Repudiation             Non-repudiation      Denied action without logs
Information Disclosure  Confidentiality      Data leak via error msg
Denial of Service       Availability         Resource exhaustion
Elevation of Privilege  Authorization        Admin via SQLi
```

### STRIDE Process

```
1. Decompose the application
   - Identify entry points, assets, trust boundaries
   - Create data flow diagrams (DFDs)

2. Enumerate threats
   - Apply STRIDE to each DFD element:
     External Entity  → Spoofing, Repudiation
     Data Flow        → Tampering, Information Disclosure, DoS
     Data Store       → Tampering, Information Disclosure, Repudiation
     Process          → All STRIDE categories
     Trust Boundary   → Elevation of Privilege

3. Rate threats (DREAD or risk matrix)
4. Mitigate or accept each threat
5. Document and validate mitigations
```

### DREAD Risk Scoring

```
Factor               Range   Description
──────               ─────   ───────────
Damage Potential     1-10    How bad if exploited?
Reproducibility      1-10    How easy to reproduce?
Exploitability       1-10    How easy to exploit?
Affected Users       1-10    How many users impacted?
Discoverability      1-10    How easy to find?

Risk = (D + R + E + A + D) / 5
  High:    7-10
  Medium:  4-6
  Low:     1-3
```

### PASTA (Process for Attack Simulation and Threat Analysis)

```
Stage 1: Define business objectives
Stage 2: Define technical scope
Stage 3: Application decomposition (DFD, use cases)
Stage 4: Threat analysis (threat intel, attack libraries)
Stage 5: Vulnerability analysis (scan results, known CVEs)
Stage 6: Attack modeling (attack trees, attack patterns)
Stage 7: Risk/impact analysis (business impact, countermeasures)
```

### Attack Trees

```
# Root: Steal user credentials
# ├── Phishing
# │   ├── Email (spear phishing)
# │   └── Fake login page
# ├── Exploit application
# │   ├── SQL injection → dump users table
# │   └── XSS → steal session cookie
# ├── Brute force
# │   ├── Password spray
# │   └── Credential stuffing
# └── Insider threat
#     ├── Database admin exports
#     └── Social engineering helpdesk
```

## OWASP Top 10 (2021)

```
#   Category                          Key Mitigations
──  ────────                          ───────────────
A01 Broken Access Control             Deny by default, RBAC, validate
                                      server-side, disable directory listing
A02 Cryptographic Failures            TLS everywhere, strong algorithms,
                                      no hardcoded secrets, key rotation
A03 Injection                         Parameterized queries, input validation,
                                      output encoding, ORMs
A04 Insecure Design                   Threat modeling, secure design patterns,
                                      paved road libraries
A05 Security Misconfiguration         Hardening, minimal install, review
                                      defaults, automated config scanning
A06 Vulnerable Components             SCA, SBOM, patch management,
                                      dependency pinning, Dependabot
A07 Auth & Identity Failures          MFA, rate limiting, secure password
                                      storage (bcrypt/argon2), session mgmt
A08 Software & Data Integrity         Signed artifacts, CI/CD integrity,
                                      SBOM verification, code signing
A09 Security Logging & Monitoring     Centralized logging, alerting,
                                      audit trails, SIEM integration
A10 Server-Side Request Forgery       URL validation, allowlists, network
                                      segmentation, disable redirects
```

## Secure Coding Practices

### Input Validation

```python
# WRONG — trusting user input
query = f"SELECT * FROM users WHERE id = {user_input}"

# RIGHT — parameterized query
cursor.execute("SELECT * FROM users WHERE id = %s", (user_input,))

# Input validation strategy
# 1. Validate type (integer, string, email, UUID)
# 2. Validate length (min/max)
# 3. Validate range (0-100, date range)
# 4. Validate format (regex for structured data)
# 5. Allowlist over denylist
# 6. Canonicalize before validation (Unicode, path traversal)
```

### Output Encoding

```
Context          Encoding Method
───────          ───────────────
HTML body        HTML entity encoding (&lt; &gt; &amp; &quot;)
HTML attribute   HTML attribute encoding + quote attributes
JavaScript       JavaScript hex encoding (\xNN)
URL parameter    Percent encoding (%20, %3C)
CSS              CSS hex encoding (\00003C)
SQL              Parameterized queries (not encoding)
JSON             JSON.stringify (not raw concatenation)
```

### Secrets Management

```bash
# WRONG — hardcoded secrets
API_KEY = "sk-abc123def456"
DB_PASSWORD = "hunter2"

# RIGHT — environment variables or vault
API_KEY = os.environ["API_KEY"]

# RIGHT — secrets manager
aws secretsmanager get-secret-value --secret-id prod/api-key
vault kv get -field=password secret/database

# Detection — pre-commit hooks
# .pre-commit-config.yaml
# - repo: https://github.com/Yelp/detect-secrets
#   hooks:
#     - id: detect-secrets
#       args: ['--baseline', '.secrets.baseline']

# git-secrets (AWS)
git secrets --install
git secrets --register-aws
git secrets --scan
```

## Security Testing Tools

### SAST (Static Application Security Testing)

```
Tool            Languages              Type
────            ─────────              ────
Semgrep         Many (rule-based)      Pattern matching
CodeQL          Many (GitHub)          Semantic analysis
SonarQube       25+ languages          Multi-purpose
Bandit          Python                 AST analysis
gosec           Go                     Go-specific
Brakeman        Ruby on Rails          Framework-specific
SpotBugs        Java                   Bytecode analysis
ESLint+plugins  JavaScript/TypeScript  Linting + security rules

# Run Semgrep
semgrep scan --config=auto .
semgrep scan --config=p/owasp-top-ten .

# Run gosec
gosec ./...

# Run Bandit
bandit -r ./src -f json -o bandit-report.json
```

### DAST (Dynamic Application Security Testing)

```
Tool              Type              Notes
────              ────              ─────
OWASP ZAP         Open source       Active/passive scanning
Burp Suite        Commercial        Manual + automated
Nuclei            Template-based    Community templates
Nikto             Web server        Legacy but useful
Arachni           Open source       Modern crawler

# OWASP ZAP (command line)
zap-cli quick-scan --self-contained \
  --start-options "-config api.disablekey=true" \
  http://localhost:8080

# ZAP Docker
docker run -t ghcr.io/zaproxy/zaproxy:stable \
  zap-baseline.py -t http://target:8080

# Nuclei
nuclei -u http://target:8080 -t cves/ -t vulnerabilities/
```

### IAST (Interactive Application Security Testing)

```
# Instruments the running application
# Monitors data flow during normal testing
# Lower false positive rate than SAST/DAST alone

Tools: Contrast Security, Seeker, Hdiv

# Advantages over SAST/DAST
# - Sees actual runtime data flow (true taint tracking)
# - Identifies exact vulnerable line of code
# - No separate scan needed — runs during QA testing
# - Lower false positive rate

# Limitations
# - Requires instrumentation agent in runtime
# - Language-specific agents
# - Performance overhead (5-10%)
# - Only finds issues in exercised code paths
```

### SCA (Software Composition Analysis)

```bash
# Scan dependencies for known vulnerabilities

# Trivy (comprehensive)
trivy fs --scanners vuln .
trivy image myapp:latest
trivy sbom --format spdx-json -o sbom.json .

# Grype (Anchore)
grype dir:. --output table
grype sbom:sbom.json

# npm audit
npm audit
npm audit fix
npm audit --production    # skip devDependencies

# pip-audit (Python)
pip-audit
pip-audit -r requirements.txt

# govulncheck (Go)
govulncheck ./...

# Dependabot / Renovate (automated PR)
# .github/dependabot.yml
# version: 2
# updates:
#   - package-ecosystem: "npm"
#     directory: "/"
#     schedule:
#       interval: "weekly"
```

## SBOM (Software Bill of Materials)

```bash
# Formats: SPDX, CycloneDX

# Generate SBOM with Syft
syft dir:. -o spdx-json > sbom.spdx.json
syft dir:. -o cyclonedx-json > sbom.cdx.json

# Generate from container image
syft myapp:latest -o spdx-json > sbom.spdx.json

# Verify SBOM against vulnerabilities
grype sbom:sbom.spdx.json

# CycloneDX for Go
cyclonedx-gomod mod -json -output sbom.cdx.json

# Required by:
# - US Executive Order 14028
# - NIST SSDF (Secure Software Development Framework)
# - EU Cyber Resilience Act
```

## DevSecOps Pipeline

```yaml
# CI/CD Security Gates
# ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
# │  Commit  │→ │  Build   │→ │  Test    │→ │  Deploy  │
# │          │  │          │  │          │  │          │
# │ Secrets  │  │ SAST     │  │ DAST     │  │ Config   │
# │ scan     │  │ SCA      │  │ IAST     │  │ scan     │
# │ Lint     │  │ License  │  │ Pen test │  │ Runtime  │
# │          │  │ SBOM     │  │ Fuzz     │  │ protect  │
# └──────────┘  └──────────┘  └──────────┘  └──────────┘

# GitHub Actions Example
# .github/workflows/security.yml
name: Security Pipeline
on: [push, pull_request]
jobs:
  sast:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: returntocorp/semgrep-action@v1
        with:
          config: >-
            p/owasp-top-ten
            p/golang

  sca:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: aquasecurity/trivy-action@master
        with:
          scan-type: fs
          scan-ref: .

  secrets:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: gitleaks/gitleaks-action@v2
```

## Security Code Review Checklist

```
Category            Check Items
────────            ───────────
Authentication      Password hashing (bcrypt/argon2), MFA, session mgmt,
                    account lockout, credential rotation
Authorization       Access control at every endpoint, IDOR checks,
                    server-side enforcement, least privilege
Input handling      All inputs validated, parameterized queries,
                    output encoding, file upload restrictions
Cryptography        No custom crypto, strong algorithms, proper key mgmt,
                    no ECB mode, authenticated encryption (GCM)
Error handling      No stack traces to users, generic error messages,
                    proper logging without sensitive data
Dependencies        Known vulnerabilities, pinned versions, license check
Configuration       No default credentials, debug disabled in prod,
                    security headers set (CSP, HSTS, X-Frame-Options)
Logging             Security events logged, no PII in logs, tamper-proof
                    storage, sufficient for forensics
```

## Security Champions Program

```
# Distributed security expertise across dev teams

Role Responsibilities:
- Participate in threat modeling sessions
- Review security-relevant PRs
- Triage security tool findings (reduce noise)
- Mentor team on secure coding
- Liaison between security team and dev team
- Track security debt for their service

Program Structure:
- 1 champion per 8-10 developers
- 10-20% time allocation for security activities
- Monthly champion community meetings
- Quarterly training and certification support
- Annual security champion recognition
```

## See Also

- sast-dast
- threat-modeling
- security-code-review
- sbom
- vulnerability-scanning
- container-security

## References

- OWASP Top 10 (2021): https://owasp.org/Top10/
- OWASP ASVS: Application Security Verification Standard
- OWASP SAMM: Software Assurance Maturity Model
- NIST SSDF: SP 800-218 (Secure Software Development Framework)
- Microsoft SDL: Security Development Lifecycle
- BSIMM: Building Security In Maturity Model
