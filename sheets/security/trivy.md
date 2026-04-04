# Trivy (Comprehensive Security Scanner)

Trivy is an all-in-one security scanner that detects vulnerabilities in container images, filesystems, Git repositories, and IaC configurations, generates SBOMs in CycloneDX and SPDX formats, finds exposed secrets, and integrates with Kubernetes via the Trivy Operator for continuous cluster scanning.

## Installation

### Install Trivy

```bash
# Install via Homebrew
brew install trivy

# Install via apt (Debian/Ubuntu)
sudo apt-get install wget apt-transport-https gnupg lsb-release
wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/trivy.gpg > /dev/null
echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb $(lsb_release -sc) main" | sudo tee /etc/apt/sources.list.d/trivy.list
sudo apt-get update && sudo apt-get install trivy

# Install via RPM (Fedora/RHEL)
sudo rpm -ivh https://github.com/aquasecurity/trivy/releases/download/v0.50.0/trivy_0.50.0_Linux-64bit.rpm

# Run as Docker container
docker run aquasec/trivy:latest image alpine:3.19

# Verify and update database
trivy image --download-db-only
trivy --version
```

## Image Scanning

### Basic Image Scan

```bash
# Scan a container image
trivy image alpine:3.19

# Scan with severity filter
trivy image --severity CRITICAL,HIGH nginx:latest

# Scan and fail on critical vulnerabilities
trivy image --exit-code 1 --severity CRITICAL myapp:latest

# Scan local Docker image (not from registry)
trivy image --input myimage.tar

# Scan with specific output format
trivy image --format json --output results.json alpine:3.19
trivy image --format table alpine:3.19
trivy image --format sarif --output results.sarif alpine:3.19

# Scan with specific vulnerability types
trivy image --vuln-type os,library alpine:3.19

# Ignore unfixed vulnerabilities
trivy image --ignore-unfixed nginx:latest

# Scan specific layers only
trivy image --list-all-pkgs alpine:3.19
```

### Registry Authentication

```bash
# Scan private registry image
TRIVY_USERNAME=user TRIVY_PASSWORD=pass trivy image registry.example.com/myapp:latest

# Use Docker config for auth
trivy image --docker-config ~/.docker registry.example.com/myapp:latest

# AWS ECR
trivy image $(aws sts get-caller-identity --query Account --output text).dkr.ecr.us-east-1.amazonaws.com/myapp:latest

# Google Artifact Registry
GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json trivy image us-docker.pkg.dev/project/repo/image:tag
```

## Filesystem and Repository Scanning

### Filesystem Scan

```bash
# Scan current directory
trivy fs .

# Scan specific directory
trivy fs /path/to/project

# Scan with severity filter
trivy fs --severity HIGH,CRITICAL --exit-code 1 .

# Scan only specific file types
trivy fs --scanners vuln .
trivy fs --scanners secret .
trivy fs --scanners misconfig .
```

### Git Repository Scan

```bash
# Scan a remote repository
trivy repo https://github.com/org/repo

# Scan specific branch
trivy repo --branch develop https://github.com/org/repo

# Scan specific commit
trivy repo --commit abc123 https://github.com/org/repo

# Scan with all scanners
trivy repo --scanners vuln,secret,misconfig https://github.com/org/repo
```

## IaC Scanning

### Terraform and CloudFormation

```bash
# Scan Terraform files
trivy config ./terraform/

# Scan CloudFormation templates
trivy config ./cloudformation/

# Scan Kubernetes manifests
trivy config ./k8s-manifests/

# Scan Dockerfiles
trivy config --file-patterns "Dockerfile:dockerfile" .

# Scan with specific severity
trivy config --severity HIGH,CRITICAL ./terraform/

# Custom policy directory
trivy config --policy ./custom-policies ./terraform/

# Skip specific checks
trivy config --skip-policy-update --skip-dirs node_modules ./

# Output as JSON
trivy config --format json --output iac-results.json ./terraform/
```

### Helm Charts

```bash
# Scan Helm chart directory
trivy config ./charts/myapp/

# Scan rendered Helm templates
helm template myapp ./charts/myapp/ | trivy config -
```

## Secret Scanning

### Detect Secrets

```bash
# Scan for secrets in filesystem
trivy fs --scanners secret .

# Scan image for embedded secrets
trivy image --scanners secret myapp:latest

# Scan repository for secrets
trivy repo --scanners secret https://github.com/org/repo

# Custom secret rules
trivy fs --scanners secret --secret-config trivy-secret.yaml .
```

### Secret Configuration

```yaml
# trivy-secret.yaml
rules:
  - id: custom-api-key
    category: general
    title: Custom API Key
    severity: HIGH
    regex: 'MYAPP_API_KEY\s*=\s*["\']?([A-Za-z0-9]{32})["\']?'
    allow-rules:
      - id: skip-tests
        regex: '_test\.go$'
```

## SBOM Generation

### CycloneDX and SPDX

```bash
# Generate CycloneDX SBOM
trivy image --format cyclonedx --output sbom.cdx.json alpine:3.19

# Generate SPDX SBOM
trivy image --format spdx-json --output sbom.spdx.json alpine:3.19

# Generate SBOM for filesystem
trivy fs --format cyclonedx --output sbom.cdx.json .

# Scan an existing SBOM for vulnerabilities
trivy sbom sbom.cdx.json
trivy sbom sbom.spdx.json

# Generate SBOM with all packages listed
trivy image --format cyclonedx --list-all-pkgs --output sbom.cdx.json myapp:latest
```

## Trivy Operator (Kubernetes)

### Install Trivy Operator

```bash
# Install via Helm
helm repo add aqua https://aquasecurity.github.io/helm-charts/
helm repo update

helm install trivy-operator aqua/trivy-operator \
  --namespace trivy-system \
  --create-namespace \
  --set trivy.ignoreUnfixed=true

# View vulnerability reports
kubectl get vulnerabilityreports -A -o wide

# View config audit reports
kubectl get configauditreports -A

# View exposed secrets reports
kubectl get exposedsecretreports -A

# Detailed vulnerability report for a workload
kubectl get vulnerabilityreport -n default \
  -l trivy-operator.resource.name=myapp \
  -o json | jq '.items[].report.vulnerabilities[] | select(.severity=="CRITICAL")'
```

## Ignoring Vulnerabilities

### .trivyignore

```bash
# .trivyignore file
# Ignore specific CVEs
CVE-2023-44487
CVE-2023-39325

# Ignore with expiration
CVE-2024-1234  # will-not-fix, expires: 2025-12-31

# Ignore by package
pkg:golang/stdlib@1.21.0
```

### Inline Ignore

```bash
# Ignore by policy in CI
trivy image --ignorefile .trivyignore myapp:latest

# Ignore specific vulnerability types
trivy image --ignore-policy policy.rego myapp:latest
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Trivy vulnerability scan
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: myapp:${{ github.sha }}
    format: sarif
    output: trivy-results.sarif
    severity: CRITICAL,HIGH
    exit-code: 1

- name: Upload Trivy scan results
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: trivy-results.sarif
```

### GitLab CI

```yaml
trivy-scan:
  stage: security
  image: aquasec/trivy:latest
  script:
    - trivy image --exit-code 1 --severity CRITICAL,HIGH
        --format json --output trivy-report.json
        ${CI_REGISTRY_IMAGE}:${CI_COMMIT_SHA}
  artifacts:
    reports:
      container_scanning: trivy-report.json
```

## Tips

- Use `--severity CRITICAL,HIGH` in CI gates to avoid blocking on low-risk findings
- Use `--ignore-unfixed` to suppress vulnerabilities with no available patch
- Generate SBOMs with `--format cyclonedx` for supply chain compliance requirements
- Use `.trivyignore` for accepted risks rather than disabling entire vulnerability classes
- Run `trivy image --download-db-only` as a pre-step in CI to cache the vulnerability database
- Use `--scanners vuln,secret,misconfig` to run all scanner types in a single pass
- The Trivy Operator continuously scans Kubernetes workloads; check reports with `kubectl get vulnerabilityreports`
- Use SARIF output format for integration with GitHub Advanced Security code scanning
- Pin Trivy versions in CI to avoid unexpected behavior from database or scanner updates
- Use `trivy sbom` to scan pre-generated SBOMs, enabling offline vulnerability assessment

## See Also

- container-security, docker, falco, grype, syft, opa

## References

- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
- [Trivy GitHub Repository](https://github.com/aquasecurity/trivy)
- [Trivy Operator](https://aquasecurity.github.io/trivy-operator/)
- [Trivy SBOM Guide](https://aquasecurity.github.io/trivy/latest/docs/supply-chain/sbom/)
- [CycloneDX Specification](https://cyclonedx.org/specification/overview/)
