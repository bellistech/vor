# SAST & DAST (Static and Dynamic Application Security Testing)

SAST analyzes source code and binaries for vulnerabilities without execution, while DAST probes running applications for exploitable flaws, together forming the core of shift-left application security when integrated with secrets scanning, dependency scanning, and IAST in CI/CD pipelines.

## SAST - Static Analysis

### Semgrep

```bash
# Install semgrep
pip install semgrep
brew install semgrep

# Scan with default rulesets
semgrep scan --config auto .

# Scan with specific rule packs
semgrep scan --config p/owasp-top-ten .
semgrep scan --config p/golang .
semgrep scan --config p/python .
semgrep scan --config p/javascript .

# Scan specific files
semgrep scan --config auto src/

# Output formats
semgrep scan --config auto --json -o results.json .
semgrep scan --config auto --sarif -o results.sarif .

# Custom rules
semgrep scan --config ./custom-rules/ .

# Severity filtering
semgrep scan --config auto --severity ERROR .

# Exclude paths
semgrep scan --config auto --exclude tests/ --exclude vendor/ .

# Dry run (show rules without scanning)
semgrep scan --config auto --dry-run .
```

### Custom Semgrep Rules

```yaml
# .semgrep/custom-rules.yaml
rules:
  - id: no-hardcoded-secrets
    patterns:
      - pattern: |
          $KEY = "..."
      - metavariable-regex:
          metavariable: $KEY
          regex: (password|secret|api_key|token)
    message: "Hardcoded secret detected in $KEY"
    languages: [python, javascript, go]
    severity: ERROR

  - id: sql-injection
    patterns:
      - pattern: |
          db.Query(fmt.Sprintf("... %s ...", $INPUT))
    message: "Possible SQL injection via string formatting"
    languages: [go]
    severity: ERROR
    metadata:
      cwe: "CWE-89"
      owasp: "A03:2021"
```

### CodeQL (GitHub)

```bash
# Install CodeQL CLI
gh extension install github/gh-codeql

# Create CodeQL database
codeql database create myapp-db --language=go --source-root=.
codeql database create myapp-db --language=python --source-root=.

# Run analysis
codeql database analyze myapp-db \
  --format=sarif-latest \
  --output=results.sarif \
  codeql/go-queries:codeql-suites/go-security-and-quality.qls

# Run specific queries
codeql database analyze myapp-db \
  --format=csv \
  --output=results.csv \
  codeql/go-queries:Security/CWE-089/SqlInjection.ql
```

### CodeQL GitHub Actions

```yaml
- name: Initialize CodeQL
  uses: github/codeql-action/init@v3
  with:
    languages: go, javascript

- name: Build
  run: make build

- name: Perform CodeQL Analysis
  uses: github/codeql-action/analyze@v3
  with:
    category: "/language:go"
```

### SonarQube

```bash
# Run SonarQube scanner
docker run --rm \
  -e SONAR_HOST_URL=http://sonarqube:9000 \
  -e SONAR_TOKEN=$SONAR_TOKEN \
  -v $(pwd):/usr/src \
  sonarsource/sonar-scanner-cli \
  -Dsonar.projectKey=myapp \
  -Dsonar.sources=. \
  -Dsonar.exclusions="**/vendor/**,**/test/**"

# SonarQube quality gate check
curl -s -u "$SONAR_TOKEN:" \
  "http://sonarqube:9000/api/qualitygates/project_status?projectKey=myapp" \
  | jq '.projectStatus.status'
```

## DAST - Dynamic Analysis

### OWASP ZAP

```bash
# Run ZAP in Docker
docker run --rm -t zaproxy/zap-stable zap-baseline.py \
  -t https://target.example.com

# Full scan
docker run --rm -t zaproxy/zap-stable zap-full-scan.py \
  -t https://target.example.com \
  -r report.html

# API scan (OpenAPI)
docker run --rm -t zaproxy/zap-stable zap-api-scan.py \
  -t https://target.example.com/openapi.json \
  -f openapi \
  -r api-report.html

# GraphQL scan
docker run --rm -t zaproxy/zap-stable zap-api-scan.py \
  -t https://target.example.com/graphql \
  -f graphql

# ZAP with authentication
docker run --rm -t zaproxy/zap-stable zap-full-scan.py \
  -t https://target.example.com \
  -z "-config auth.method=form \
      -config auth.loginurl=https://target.example.com/login \
      -config auth.username=test \
      -config auth.password=test"

# Output formats
docker run --rm -t zaproxy/zap-stable zap-baseline.py \
  -t https://target.example.com \
  -J report.json \
  -w report.md \
  -r report.html
```

### ZAP GitHub Actions

```yaml
- name: OWASP ZAP Baseline Scan
  uses: zaproxy/action-baseline@v0.12.0
  with:
    target: https://target.example.com
    rules_file_name: zap-rules.tsv
    cmd_options: '-a'

- name: ZAP Full Scan
  uses: zaproxy/action-full-scan@v0.10.0
  with:
    target: https://target.example.com
    allow_issue_writing: false
```

### Nuclei (Template-Based Scanner)

```bash
# Install nuclei
go install github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest

# Scan with all templates
nuclei -u https://target.example.com

# Scan with specific templates
nuclei -u https://target.example.com -t cves/
nuclei -u https://target.example.com -t exposures/
nuclei -u https://target.example.com -t misconfiguration/

# Severity filtering
nuclei -u https://target.example.com -severity critical,high

# Output formats
nuclei -u https://target.example.com -o results.txt
nuclei -u https://target.example.com -jsonl -o results.jsonl
nuclei -u https://target.example.com -sarif -o results.sarif

# Scan multiple targets
nuclei -l targets.txt -severity critical
```

## Secrets Scanning

### Gitleaks

```bash
# Install gitleaks
brew install gitleaks

# Scan repository
gitleaks detect -v

# Scan specific path
gitleaks detect --source /path/to/repo -v

# Scan git history
gitleaks detect --log-opts="--all" -v

# Generate baseline (suppress known findings)
gitleaks detect --baseline-path gitleaks-baseline.json -v

# Output formats
gitleaks detect -f json -r gitleaks-report.json
gitleaks detect -f sarif -r gitleaks-report.sarif

# Pre-commit hook
gitleaks protect --staged -v
```

### TruffleHog

```bash
# Install and scan git repository
brew install trufflehog
trufflehog git file://. --since-commit HEAD~50
trufflehog github --org myorg --token $GITHUB_TOKEN
trufflehog filesystem /path/to/scan --only-verified
```

## Dependency Scanning

### Language-Specific Auditing

```bash
# Go: govulncheck
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
govulncheck -mode=binary ./myapp

# Node.js
npm audit --json
npm audit fix

# Python
pip install pip-audit
pip-audit -r requirements.txt -f json -o audit.json
```

## CI/CD Pipeline Integration

### Shift-Left Pipeline

```yaml
# Complete security scanning pipeline
stages:
  - secrets     # Earliest: catch leaked credentials
  - sast        # Pre-build: analyze source code
  - build       # Build artifacts
  - dependency  # Post-build: scan dependencies
  - dast        # Post-deploy: scan running app

secrets-scan:
  stage: secrets
  script:
    - gitleaks detect --exit-code 1

sast-scan:
  stage: sast
  script:
    - semgrep scan --config auto --severity ERROR --error .

dependency-scan:
  stage: dependency
  script:
    - govulncheck ./...
    - grype dir:. --fail-on high

dast-scan:
  stage: dast
  script:
    - zap-baseline.py -t $STAGING_URL -I
  when: manual
```

## Tips

- Run secrets scanning as the first pipeline stage; leaked credentials are the highest-impact finding
- Use semgrep with `--config auto` for instant results without custom rule configuration
- Prefer SARIF output format across all tools for unified reporting in GitHub/GitLab code scanning
- SAST catches logic flaws in your code; DAST catches configuration and deployment issues at runtime
- Combine SAST and DAST for coverage: SAST has no false negatives for pattern matches, DAST finds real exploitability
- Use gitleaks as a pre-commit hook to prevent secrets from ever entering git history
- CodeQL excels at taint tracking (source-to-sink data flow analysis) for injection vulnerabilities
- Run DAST against staging environments, never production, to avoid data corruption or denial of service
- Dependency scanning (govulncheck, npm audit) catches vulnerabilities in code you did not write
- IAST provides the best signal-to-noise ratio but requires instrumented test environments
- Set quality gates in CI: fail builds on CRITICAL/HIGH SAST findings, alert on MEDIUM

## See Also

- vulnerability-scanning, container-security, trivy, grype, cosign, sbom, secrets

## References

- [Semgrep Documentation](https://semgrep.dev/docs/)
- [CodeQL Documentation](https://codeql.github.com/docs/)
- [OWASP ZAP](https://www.zaproxy.org/)
- [Gitleaks](https://github.com/gitleaks/gitleaks)
- [TruffleHog](https://github.com/trufflesecurity/trufflehog)
- [OWASP Testing Guide](https://owasp.org/www-project-web-security-testing-guide/)
- [govulncheck](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck)
- [Nuclei Scanner](https://github.com/projectdiscovery/nuclei)
