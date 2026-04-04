# Grype (Vulnerability Scanner)

Grype is an SBOM-first vulnerability scanner from Anchore that matches software packages and dependencies against known CVE databases, supporting container images, filesystems, and pre-generated SBOMs with configurable severity filtering, ignore rules, and multiple output formats.

## Installation

### Install Grype

```bash
# Install via Homebrew
brew install grype

# Install via curl (latest)
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin

# Install specific version
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin v0.84.0

# Run as Docker container
docker run --rm anchore/grype:latest alpine:3.19

# Verify installation
grype version
grype db status
```

## Image Scanning

### Basic Image Scan

```bash
# Scan a container image
grype alpine:3.19

# Scan by digest
grype myregistry.io/myapp@sha256:abc123...

# Scan local Docker image
grype docker:myapp:latest

# Scan image archive (tarball)
grype docker-archive:myimage.tar

# Scan OCI layout directory
grype oci-dir:./oci-layout/

# Pull from private registry
grype registry:myregistry.io/myapp:latest
```

### Severity Filtering

```bash
# Show only critical and high vulnerabilities
grype alpine:3.19 --only-fixed

# Fail on severity threshold (for CI gates)
grype alpine:3.19 --fail-on critical
grype alpine:3.19 --fail-on high

# Show only fixable vulnerabilities
grype alpine:3.19 --only-fixed

# Filter by severity in output
grype alpine:3.19 -o table | grep -E "Critical|High"
```

## SBOM-First Scanning

### Generate and Scan SBOMs

```bash
# Generate SBOM with syft, then scan with grype
syft myregistry.io/myapp:v1.0 -o syft-json > sbom.syft.json
grype sbom:sbom.syft.json

# CycloneDX SBOM
syft myregistry.io/myapp:v1.0 -o cyclonedx-json > sbom.cdx.json
grype sbom:sbom.cdx.json

# SPDX SBOM
syft myregistry.io/myapp:v1.0 -o spdx-json > sbom.spdx.json
grype sbom:sbom.spdx.json

# Scan filesystem-generated SBOM
syft dir:. -o syft-json > project-sbom.json
grype sbom:project-sbom.json

# Scan SBOM from stdin
syft myapp:latest -o syft-json | grype
```

## Filesystem and Directory Scanning

### Local Scanning

```bash
# Scan current directory
grype dir:.

# Scan specific directory
grype dir:/path/to/project

# Scan a specific file
grype file:/path/to/binary

# Scan with package type hints
grype dir:. --add-cpes-if-none
```

## Database Management

### Grype DB Commands

```bash
# Check database status
grype db status

# Update the vulnerability database
grype db update

# Download database without scanning
grype db check

# List available databases
grype db list

# Delete cached database
grype db delete

# Import database from file (air-gapped environments)
grype db import ./grype-db.tar.gz

# Use specific database
GRYPE_DB_CACHE_DIR=/custom/path grype alpine:3.19
```

## Output Formats

### Format Options

```bash
# Table output (default)
grype alpine:3.19 -o table

# JSON output
grype alpine:3.19 -o json > results.json

# CycloneDX output
grype alpine:3.19 -o cyclonedx > results.cdx.xml
grype alpine:3.19 -o cyclonedx-json > results.cdx.json

# SARIF output (GitHub code scanning)
grype alpine:3.19 -o sarif > results.sarif

# Template output (custom Go templates)
grype alpine:3.19 -o template -t ./custom-template.tmpl

# Write to file
grype alpine:3.19 -o json --file results.json
```

## Ignore Rules

### Configuration File

```yaml
# .grype.yaml
ignore:
  # Ignore specific CVEs
  - vulnerability: CVE-2023-44487
  - vulnerability: CVE-2023-39325

  # Ignore by package name
  - package:
      name: openssl
      version: "3.1.2"
      type: apk

  # Ignore by fix state
  - fix-state: wont-fix

  # Ignore by severity below threshold
  - vulnerability: CVE-2024-*
    package:
      type: gem

  # Ignore with reason (documentation)
  - vulnerability: CVE-2023-12345
    # Reason: not exploitable in our configuration

# Fail-on severity
fail-on-severity: critical

# Only show fixed vulnerabilities
only-fixed: true

# Add CPEs if none detected
add-cpes-if-none: true

# Database configuration
db:
  auto-update: true
  cache-dir: /tmp/grype-db
```

### Inline Ignore File

```bash
# .grype-ignore file (one CVE per line)
CVE-2023-44487
CVE-2023-39325
GHSA-xxxx-yyyy-zzzz

# Use custom ignore file
grype alpine:3.19 --config custom-grype.yaml
```

## CI/CD Integration

### GitHub Actions

```yaml
- name: Install Grype
  uses: anchore/scan-action/download-grype@v4

- name: Scan image
  run: |
    grype myapp:${{ github.sha }} \
      --fail-on high \
      -o sarif --file grype-results.sarif

- name: Upload SARIF
  uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: grype-results.sarif

# Using Anchore scan action directly
- name: Anchore Grype scan
  uses: anchore/scan-action@v4
  with:
    image: myapp:${{ github.sha }}
    fail-build: true
    severity-cutoff: high
    output-format: sarif
```

### GitLab CI

```yaml
grype-scan:
  stage: security
  image: anchore/grype:latest
  script:
    - grype ${CI_REGISTRY_IMAGE}:${CI_COMMIT_SHA}
        --fail-on critical
        -o json --file grype-report.json
  artifacts:
    paths:
      - grype-report.json
    when: always
```

## Syft Integration

### SBOM Generation with Syft

```bash
# Install syft
brew install syft

# Generate SBOM for image
syft myapp:latest -o syft-json > sbom.json

# Generate SBOM for directory
syft dir:. -o cyclonedx-json > sbom.cdx.json

# Generate SBOM for archive
syft docker-archive:myimage.tar -o spdx-json > sbom.spdx.json

# Scan generated SBOM
grype sbom:sbom.json

# Pipeline: generate and scan in one step
syft myapp:latest -o syft-json | grype --fail-on critical
```

## Grype vs Trivy Comparison

### Feature Comparison

```bash
# Grype: SBOM-first, Anchore ecosystem, syft-generated SBOMs
grype sbom:sbom.syft.json --fail-on critical

# Trivy: all-in-one, IaC scanning, secret detection, Kubernetes operator
trivy image --severity CRITICAL --exit-code 1 myapp:latest

# Grype: lightweight, focused on vulnerability scanning
# Trivy: broader scope (vuln + misconfig + secret + license)

# Both support: CycloneDX, SPDX, SARIF, JSON output
# Grype unique: syft-json native format, Go template output
# Trivy unique: built-in IaC scanning, secret scanning, K8s operator
```

## Tips

- Use SBOM-first scanning (`syft` + `grype sbom:`) for reproducible results across environments
- Set `--fail-on critical` in CI pipelines as a minimum quality gate
- Use `.grype.yaml` for persistent ignore rules with documented reasons for each exception
- Run `grype db update` as a CI pre-step and cache the database to speed up subsequent scans
- Prefer JSON or SARIF output for machine-readable results in automated pipelines
- Use `--only-fixed` to focus developer attention on vulnerabilities they can actually remediate
- Combine grype with cosign to scan images and then sign verified-clean artifacts
- For air-gapped environments, pre-download the database with `grype db import`
- Use CycloneDX output format when feeding results into dependency-track or similar platforms
- Compare grype and trivy results periodically; different databases catch different CVEs
- Pin grype versions in CI and update deliberately after testing

## See Also

- trivy, container-security, cosign, sbom, vulnerability-scanning, falco

## References

- [Grype Documentation](https://github.com/anchore/grype)
- [Syft SBOM Generator](https://github.com/anchore/syft)
- [Anchore Scan GitHub Action](https://github.com/anchore/scan-action)
- [Grype Database Sources](https://github.com/anchore/grype-db)
- [CycloneDX Specification](https://cyclonedx.org/specification/overview/)
- [SARIF Specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)
