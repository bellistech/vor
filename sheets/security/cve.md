# CVE (Common Vulnerabilities and Exposures)

Standardized system for identifying, scoring, and tracking publicly disclosed security vulnerabilities using unique identifiers, CVSS severity scores, and lifecycle management across the software supply chain.

## CVE Identifier Format

### Structure

```
CVE-YYYY-NNNNN

CVE-2024-21626       # runc container escape
CVE-2023-44487       # HTTP/2 Rapid Reset
CVE-2021-44228       # Log4Shell
CVE-2021-3156        # sudo Baron Samedit
CVE-2017-5754        # Meltdown

# YYYY = year of assignment (not disclosure)
# NNNNN = sequential number (can be 4-7+ digits)
# Assigned by CNAs (CVE Numbering Authorities)
```

### CNA Hierarchy

```
Top-Level CNA:  MITRE Corporation (root CNA)
Sub-CNAs:       Red Hat, Google, Microsoft, Apache, GitHub, etc.
                Each CNA assigns CVEs within their scope

# Request a CVE:
# - Report to vendor (they may be a CNA)
# - Or submit to MITRE via https://cveform.mitre.org/
# - Or use GitHub's private vulnerability reporting (CNA for repos)
```

## CVSS v3.1 Scoring

### Base Score Components

```
Attack Vector (AV):
  Network (N)    = 0.85   # Remotely exploitable
  Adjacent (A)   = 0.62   # Local network/Bluetooth
  Local (L)      = 0.55   # Requires local access
  Physical (P)   = 0.20   # Requires physical access

Attack Complexity (AC):
  Low (L)        = 0.77   # No special conditions
  High (H)       = 0.44   # Requires specific config/timing

Privileges Required (PR):
  None (N)       = 0.85   # No authentication needed
  Low (L)        = 0.62   # Basic user access
  High (H)       = 0.27   # Admin/privileged access

User Interaction (UI):
  None (N)       = 0.85   # No user action needed
  Required (R)   = 0.62   # User must click/interact
```

### Impact Metrics

```
Confidentiality (C):
  High (H)       # Total information disclosure
  Low (L)        # Limited information disclosure
  None (N)       # No confidentiality impact

Integrity (I):
  High (H)       # Total data modification
  Low (L)        # Limited data modification
  None (N)       # No integrity impact

Availability (A):
  High (H)       # Total service disruption
  Low (L)        # Reduced performance
  None (N)       # No availability impact

Scope (S):
  Changed (C)    # Impacts resources beyond the vulnerable component
  Unchanged (U)  # Impact limited to vulnerable component
```

### Severity Ratings

```
Score Range    Rating       Action Timeline
0.0            None         No action required
0.1 - 3.9     Low          Schedule for next patch cycle
4.0 - 6.9     Medium       Patch within 30 days
7.0 - 8.9     High         Patch within 7 days
9.0 - 10.0    Critical     Patch immediately / emergency change

# Vector string example:
CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H    # Score: 10.0 (Log4Shell)
CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H    # Score: 9.8
CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H    # Score: 7.8
```

## Vulnerability Lifecycle

### Stages

```
1. Discovery     # Researcher/attacker finds the vulnerability
2. Reporting     # Reported to vendor (responsible disclosure)
3. CVE Assignment # CNA assigns CVE ID
4. Triage        # Vendor assesses severity and impact
5. Patch Dev     # Vendor develops fix
6. Disclosure    # Coordinated public disclosure + advisory
7. Patch Release # Fix available for users
8. Remediation   # Users apply patches
9. Post-mortem   # Lessons learned, detection rules updated

# Embargo period: typically 90 days (Google Project Zero standard)
# Zero-day: exploited before patch available
```

### Advisory Sources

```bash
# National Vulnerability Database (NVD)
https://nvd.nist.gov/vuln/search

# MITRE CVE List
https://cve.mitre.org/cve/search_cve_list.html

# Vendor advisories
https://access.redhat.com/security/cve/
https://ubuntu.com/security/cves
https://security.gentoo.org/
https://www.debian.org/security/
https://msrc.microsoft.com/update-guide/

# GitHub Advisory Database
https://github.com/advisories

# CISA Known Exploited Vulnerabilities (KEV)
https://www.cisa.gov/known-exploited-vulnerabilities-catalog
```

## Scanning Tools

### OS-Level Scanning

```bash
# Trivy — comprehensive scanner
trivy image myapp:latest                          # Container image
trivy fs /path/to/project                         # Filesystem
trivy repo https://github.com/user/repo           # Git repo
trivy k8s --report=summary cluster                # Kubernetes cluster
trivy sbom --format cyclonedx -o sbom.json .      # Generate SBOM

# Grype — container/filesystem vulnerability scanner
grype myapp:latest                                 # Scan image
grype dir:/path/to/project                         # Scan directory
grype sbom:./sbom.json                             # Scan SBOM

# OpenSCAP — NIST compliance scanning
oscap oval eval --results results.xml \
  com.redhat.rhsa-RHEL8.xml

# Vuls — agentless vulnerability scanner
vuls scan                                          # Scan configured hosts
vuls report                                        # Generate report
```

### Dependency Scanning

```bash
# Go
govulncheck ./...                                  # Official Go vuln checker
go list -m -json all | nancy sleuth                # Nancy (Sonatype)

# Node.js
npm audit                                          # Built-in
npm audit fix                                      # Auto-fix
npx audit-ci --critical                            # CI gate

# Python
pip-audit                                          # PyPI vulnerabilities
safety check -r requirements.txt                   # Safety DB

# Rust
cargo audit                                        # RustSec advisory DB

# Java
mvn org.owasp:dependency-check-maven:check         # OWASP dep check
```

### CI/CD Integration

```yaml
# GitHub Actions — Trivy
- name: Run Trivy vulnerability scanner
  uses: aquasecurity/trivy-action@master
  with:
    image-ref: 'myapp:${{ github.sha }}'
    format: 'sarif'
    output: 'trivy-results.sarif'
    severity: 'CRITICAL,HIGH'
    exit-code: '1'                          # Fail pipeline on findings

# GitLab CI
dependency_scanning:
  stage: test
  image: registry.gitlab.com/gitlab-org/security-products/analyzers/gemnasium
  script:
    - /analyzer run
  artifacts:
    reports:
      dependency_scanning: gl-dependency-scanning-report.json
```

## Vulnerability Management

### Tracking and Prioritization

```bash
# EPSS — Exploit Prediction Scoring System
# Probability a CVE will be exploited in the next 30 days
# Complements CVSS severity with likelihood
curl "https://api.first.org/data/v1/epss?cve=CVE-2024-21626"

# Prioritization matrix:
# CVSS Critical + EPSS High + In CISA KEV = Patch immediately
# CVSS High + EPSS Low + Not in KEV = Schedule within sprint
# CVSS Medium + EPSS Low = Next patch cycle

# SSVC — Stakeholder-Specific Vulnerability Categorization
# Decision tree: Exploitation status, Technical Impact, Mission impact
# Outcomes: Defer, Scheduled, Out-of-cycle, Immediate
```

### Patch Verification

```bash
# Check if specific CVE is patched
rpm -q --changelog package | grep CVE-2024-XXXXX     # RHEL/Fedora
apt changelog package 2>/dev/null | grep CVE          # Debian/Ubuntu
trivy image --vuln-type os myapp:latest | grep CVE-2024

# Verify kernel patches
uname -r                                               # Current kernel
cat /proc/version
```

## Tips

- Combine CVSS scores with EPSS probability and CISA KEV status for effective prioritization
- Not all CVEs are equal: a critical-scoring CVE with no known exploit may be lower priority than a medium with active exploitation
- Use `govulncheck` for Go projects as it only reports vulnerabilities in code paths you actually call
- Subscribe to vendor security mailing lists for early advisory notifications before NVD updates
- Maintain a Software Bill of Materials (SBOM) to quickly identify exposure when new CVEs are published
- Set CI/CD gates to fail builds on critical and high CVEs but allow overrides with documented exceptions
- The 90-day disclosure deadline (Google Project Zero standard) pressures vendors to patch promptly
- CVSS Temporal and Environmental scores refine the Base score for your specific context but are rarely used
- Zero-day vulnerabilities (exploited before patch) require compensating controls until patches arrive
- Track CVE remediation SLAs: Critical < 24h, High < 7d, Medium < 30d, Low < 90d as industry baselines
- Container base image updates are the single most impactful action for reducing CVE counts
- Use vulnerability scanners in both CI pipelines and scheduled production scans for continuous coverage

## See Also

container-security, vulnerability-scanning, hardening-linux, forensics, incident-response

## References

- [MITRE CVE Program](https://www.cve.org/)
- [NVD — National Vulnerability Database](https://nvd.nist.gov/)
- [CVSS v3.1 Specification](https://www.first.org/cvss/v3.1/specification-document)
- [CISA Known Exploited Vulnerabilities](https://www.cisa.gov/known-exploited-vulnerabilities-catalog)
- [EPSS — Exploit Prediction Scoring System](https://www.first.org/epss/)
- [Trivy Documentation](https://aquasecurity.github.io/trivy/)
