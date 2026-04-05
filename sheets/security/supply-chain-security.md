# Supply Chain Security (SCRM, SBOM, Third-Party Risk)

Practical reference for software and hardware supply chain risk management including SBOM generation, vendor risk assessment, supply chain attack prevention, and regulatory compliance.

## Supply Chain Risk Management (SCRM)

### SCRM Framework

```text
SCRM Lifecycle:
  1. Identify     -- map supply chain, inventory suppliers and dependencies
  2. Assess       -- evaluate risk for each supplier/component
  3. Mitigate     -- apply controls proportional to risk level
  4. Monitor      -- continuous surveillance for changes and threats
  5. Respond      -- incident response for supply chain compromises
  6. Recover      -- restore operations, switch to alternate suppliers

Risk Categories:
  Cybersecurity risk     -- software vulnerabilities, malware injection, backdoors
  Operational risk       -- supplier outage, single-source dependency, capacity issues
  Compliance risk        -- supplier non-compliance with regulations
  Geopolitical risk      -- export controls, sanctions, regional instability
  Financial risk         -- supplier bankruptcy, acquisition, financial instability
  Reputational risk      -- supplier breach damages your brand
  Quality risk           -- counterfeit components, substandard materials
```

### Supplier Tiering

```text
Tier Classification:
  Tier 1 (Critical):    Direct suppliers of critical components/services
                        Single-source or limited alternatives
                        Access to sensitive data or production systems
                        Example: cloud provider, IAM vendor, CI/CD platform

  Tier 2 (Important):   Multiple alternatives available
                        Moderate data access or system integration
                        Example: monitoring tools, log aggregation, CDN

  Tier 3 (Standard):    Commodity services, easily replaceable
                        Minimal data access, no system integration
                        Example: office supplies, general SaaS tools

Assessment Frequency:
  Tier 1:  Annual comprehensive + continuous monitoring
  Tier 2:  Annual questionnaire + periodic review
  Tier 3:  Initial assessment + biennial review
```

## SBOM (Software Bill of Materials)

### CycloneDX Format

```bash
# Generate CycloneDX SBOM with cdxgen
npm install -g @cyclonedx/cdxgen
cdxgen -o sbom.json              # auto-detect project type
cdxgen -o sbom.xml -t xml        # XML format
cdxgen -t go -o sbom.json .      # specify project type

# Generate CycloneDX for Go projects
go install github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@latest
cyclonedx-gomod mod -json -output sbom.json

# Generate CycloneDX for Python
pip install cyclonedx-bom
cyclonedx-py requirements -i requirements.txt -o sbom.json --format json

# Generate CycloneDX for container images
syft alpine:latest -o cyclonedx-json > sbom.json
```

### SPDX Format

```bash
# Generate SPDX SBOM with syft (Anchore)
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
syft . -o spdx-json > sbom.spdx.json       # source directory
syft alpine:3.18 -o spdx-json > sbom.json   # container image
syft dir:/app -o spdx-json > sbom.json       # filesystem path

# SPDX with Microsoft sbom-tool
sbom-tool generate -b . -bc . -pn myproject -pv 1.0.0 -ps myorg -nsb https://myorg.com

# Validate SPDX document
pip install spdx-tools
pyspdx-validate sbom.spdx.json
```

### SBOM Consumption and Analysis

```bash
# Vulnerability scanning with grype (uses SBOM as input)
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
grype sbom:sbom.json                        # scan SBOM for CVEs
grype sbom:sbom.json -o json > vulns.json   # JSON output
grype sbom:sbom.json --fail-on critical     # fail CI on critical vulns

# SBOM diff between versions
syft . -o spdx-json > sbom-v1.json
# ... deploy new version ...
syft . -o spdx-json > sbom-v2.json
diff <(jq '.packages[].name' sbom-v1.json | sort) \
     <(jq '.packages[].name' sbom-v2.json | sort)

# Dependency-Track (OWASP) -- SBOM management platform
# Upload SBOM via API:
curl -X POST https://deptrack.example.com/api/v1/bom \
  -H "X-Api-Key: $DEPTRACK_API_KEY" \
  -H "Content-Type: multipart/form-data" \
  -F "project=$PROJECT_UUID" \
  -F "bom=@sbom.json"

# SBOM fields to verify
# Components:  name, version, purl (package URL), licenses, hashes
# Dependencies: dependency tree (who depends on what)
# Metadata:    tool, timestamp, supplier, author
```

### Package URL (purl) Specification

```text
purl format: pkg:<type>/<namespace>/<name>@<version>?<qualifiers>#<subpath>

Examples:
  pkg:golang/github.com/gin-gonic/gin@v1.9.1
  pkg:npm/%40angular/core@16.2.0
  pkg:pypi/requests@2.31.0
  pkg:maven/org.apache.logging.log4j/log4j-core@2.20.0
  pkg:deb/debian/curl@7.88.1-10+deb12u5
  pkg:docker/alpine@3.18?repository_url=docker.io
  pkg:github/actions/checkout@v4
```

## Vendor Risk Assessment

### Assessment Questionnaire (SIG Lite)

```text
SIG (Standardized Information Gathering) Questionnaire Domains:
  1.  Risk Management       -- governance, risk appetite, risk assessment
  2.  Security Policy        -- documented policies, review frequency
  3.  Organization           -- security roles, responsibilities, CISO
  4.  Asset Management       -- inventory, classification, handling
  5.  Human Resources        -- background checks, training, termination
  6.  Physical Security      -- data center, access controls, environmental
  7.  Operations Management  -- change management, capacity, malware protection
  8.  Access Control         -- authentication, authorization, privileged access
  9.  Application Security   -- SDLC, code review, testing
  10. Incident Management    -- detection, response, notification, lessons learned
  11. Business Continuity    -- BCP, DRP, RTO/RPO, testing
  12. Compliance             -- regulatory, audit, contractual obligations
  13. Network Security       -- segmentation, monitoring, encryption
  14. Privacy                -- data handling, consent, cross-border transfers
  15. Cloud Security         -- shared responsibility, configuration, monitoring
  16. Threat Management      -- vulnerability management, penetration testing
  17. Server Security        -- hardening, patching, logging
  18. Mobile / BYOD          -- MDM, app security, remote wipe

Alternative Questionnaires:
  CAIQ (CSA)              -- Cloud Controls Matrix questionnaire
  VSAQ (Google)           -- Vendor Security Assessment Questionnaire
  HECVAT                  -- Higher Education vendor assessment
  Custom risk assessment  -- tailored to organization-specific requirements
```

### Due Diligence Checklist

```text
Pre-Engagement:
  [ ] SOC 2 Type II report (or SOC 1 if financial data)
  [ ] ISO 27001 certification
  [ ] Penetration test results (within last 12 months)
  [ ] Insurance coverage (cyber liability, E&O)
  [ ] Financial stability assessment (D&B, credit reports)
  [ ] References from similar-sized clients
  [ ] Data processing agreement (DPA) for personal data
  [ ] Subprocessor list and notification process
  [ ] Incident notification SLA (72 hours maximum for breach)
  [ ] Right to audit clause in contract
  [ ] Business continuity and disaster recovery plans
  [ ] Data location and residency commitments
  [ ] Exit/transition plan and data return provisions
  [ ] Regulatory compliance attestations (HIPAA BAA, PCI AOC)
```

## Third-Party Risk Management (TPRM)

### TPRM Lifecycle

```text
Phase 1: Planning and Scoping
  - Define risk appetite and tolerance for third-party risk
  - Establish TPRM policy and governance structure
  - Define roles: TPRM team, business owners, legal, procurement

Phase 2: Due Diligence and Selection
  - Inherent risk assessment (data sensitivity, criticality, access level)
  - Security questionnaire (SIG, CAIQ, custom)
  - Evidence review (SOC 2, ISO 27001, pen test reports)
  - Residual risk rating and risk acceptance decision

Phase 3: Contracting
  - Security requirements in contract (SLAs, incident notification)
  - Data processing agreement (DPA) for personal data
  - Right to audit clause
  - Insurance requirements
  - Termination and data return provisions

Phase 4: Onboarding
  - Access provisioning (least privilege, dedicated accounts)
  - Integration security review (API security, data flows)
  - Security configuration verification

Phase 5: Ongoing Monitoring
  - Continuous risk monitoring (SecurityScorecard, BitSight, UpGuard)
  - Annual reassessment (questionnaire + evidence review)
  - Trigger-based reassessment (breach, M&A, material change)
  - Performance monitoring against SLAs

Phase 6: Offboarding
  - Access revocation (immediate, verify completeness)
  - Data return or destruction (with certification)
  - Key/credential rotation for shared secrets
  - Contract termination documentation
```

### Continuous Monitoring Tools

```text
External Risk Rating Platforms:
  SecurityScorecard      -- A-F rating, 10 risk factors, breach probability
  BitSight               -- 250-900 score, industry benchmarking
  UpGuard                -- vendor risk profiles, data leak detection
  RiskRecon (Mastercard) -- asset discovery, issue prioritization
  Panorays               -- automated questionnaires + external scanning
  Black Kite             -- technical, financial, compliance risk scoring

What They Monitor:
  - SSL/TLS configuration and certificate management
  - DNS health and DNSSEC deployment
  - Email security (SPF, DKIM, DMARC)
  - Open ports and services exposed to internet
  - Patching cadence and known vulnerabilities
  - Data leak detection (dark web, paste sites, code repos)
  - IP reputation and botnet membership
  - Web application security headers
```

## Supply Chain Attack Types

### Notable Attacks

```text
Attack                  Year    Vector                          Impact
SolarWinds (SUNBURST)   2020    Build system compromise          18,000 orgs, USG agencies
Codecov                 2021    CI script manipulation           Env vars exfiltrated
Kaseya (REvil)          2021    MSP software supply chain        1,500+ businesses ransomed
ua-parser-js            2021    npm package hijack               Crypto miner injected
Log4Shell               2021    Ubiquitous library vulnerability Millions of applications
3CX                     2023    Desktop app build compromise     600,000+ organizations
PyTorch nightly         2022    Dependency confusion             torchtriton package hijacked
xz Utils (CVE-2024-3094) 2024  Long-term maintainer compromise  SSH backdoor in compression lib
```

### Attack Taxonomy

```text
Build System Attacks:
  - Compromise CI/CD pipeline (inject malicious steps)
  - Modify build scripts or Makefiles
  - Tamper with build environment (compiler, SDK)
  - Compromise signing keys

Package/Dependency Attacks:
  - Typosquatting: "reqeusts" instead of "requests"
  - Dependency confusion: private package name on public registry
  - Maintainer account compromise: hijack legitimate package
  - Star-jacking: transfer stars to malicious fork
  - Protestware: maintainer intentionally adds harmful code

Source Code Attacks:
  - Compromise source control (GitHub account takeover)
  - Malicious pull request with obfuscated code
  - Long-term social engineering of maintainer trust (xz Utils pattern)
  - Unicode bidirectional override (Trojan Source)

Distribution Attacks:
  - Compromise update server
  - Man-in-the-middle update downloads
  - Malicious mirror/CDN
  - Tamper with package signatures
```

## Software Supply Chain Controls

### SLSA Framework (Supply-chain Levels for Software Artifacts)

```text
SLSA Levels:
  Level 0:  No guarantees (most software today)
  Level 1:  Build process documented, provenance generated
  Level 2:  Build service used, signed provenance
  Level 3:  Hardened build platform, non-falsifiable provenance
  Level 4:  Two-person review, hermetic/reproducible builds (future)

Provenance Requirements:
  Level 1:  Provenance exists and identifies the builder
  Level 2:  Provenance is signed by the build service
  Level 3:  Provenance is non-falsifiable (build service is hardened)

Implementation:
  - Use hosted CI/CD (GitHub Actions, Cloud Build, Tekton)
  - Generate SLSA provenance attestation
  - Sign with Sigstore (cosign, fulcio, rekor)
  - Verify provenance before deployment
```

### Sigstore (Signing and Verification)

```bash
# Install cosign (Sigstore)
go install github.com/sigstore/cosign/v2/cmd/cosign@latest

# Sign a container image (keyless, uses OIDC identity)
cosign sign --yes myregistry.io/myimage:v1.0.0

# Verify container image signature
cosign verify --certificate-identity=user@example.com \
  --certificate-oidc-issuer=https://accounts.google.com \
  myregistry.io/myimage:v1.0.0

# Sign a blob (binary, SBOM, etc.)
cosign sign-blob --yes --bundle artifact.sig.bundle artifact.tar.gz

# Verify a signed blob
cosign verify-blob --bundle artifact.sig.bundle artifact.tar.gz

# Attach SBOM to container image
cosign attach sbom --sbom sbom.json myregistry.io/myimage:v1.0.0

# Generate SLSA provenance attestation
cosign attest --predicate provenance.json --type slsaprovenance \
  myregistry.io/myimage:v1.0.0

# Verify attestation
cosign verify-attestation --type slsaprovenance \
  --certificate-identity=builder@ci.example.com \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  myregistry.io/myimage:v1.0.0
```

### Package Manager Security

```bash
# npm -- audit and lockfile integrity
npm audit                          # check for known vulnerabilities
npm audit fix                      # auto-fix where possible
npm audit signatures               # verify registry signatures
npm ci                             # install from lockfile only (CI use)
# package-lock.json MUST be committed and reviewed in PRs

# Go -- module verification
go mod verify                      # verify checksums match go.sum
GONOSUMCHECK=                      # never disable sum checking
GONOSUMDB=                         # never disable sum database
GOFLAGS=-mod=readonly              # prevent go.mod changes during build
go mod tidy                        # remove unused, add missing

# Python -- pip with hash checking
pip install --require-hashes -r requirements.txt
# requirements.txt with hashes:
# requests==2.31.0 \
#   --hash=sha256:942c5a758f98d790eaed1a29cb6eefc7f0d7c...

# Rust -- cargo audit
cargo install cargo-audit
cargo audit                        # check for known vulnerabilities
cargo audit fix                    # auto-fix where possible
cargo deny check                   # license and advisory checks

# Pin dependencies to exact versions in production
# Use lockfiles (package-lock.json, go.sum, Cargo.lock, poetry.lock)
# Review dependency updates in PRs (Dependabot, Renovate)
```

## Hardware Supply Chain

### Counterfeit and Tampering Controls

```text
Hardware Supply Chain Risks:
  Counterfeit components   -- fake ICs, recycled chips, remarked parts
  Firmware tampering       -- pre-installed malware, BIOS/UEFI rootkits
  Hardware implants        -- rogue chips on PCB, modified network equipment
  Refurbished as new       -- used equipment sold as new
  Gray market              -- diverted components outside authorized channels

Controls:
  Authorized resellers     -- buy only from manufacturer-authorized distributors
  Part authentication      -- X-ray inspection, decapsulation, electrical testing
  Tamper-evident packaging -- verify seals, custody chain documentation
  Firmware verification    -- hash verification against vendor-published values
  Trusted Platform Module  -- TPM for hardware root of trust, measured boot
  Secure boot              -- UEFI Secure Boot, verified boot chain
  Hardware attestation     -- remote attestation of platform integrity
```

## NIST C-SCRM (SP 800-161r1)

### Key Practices

```text
NIST SP 800-161r1 C-SCRM Practices:

Foundational:
  - Establish C-SCRM governance (policy, roles, authority)
  - Integrate C-SCRM into enterprise risk management
  - Define supply chain risk appetite and tolerance

Sustaining:
  - Supplier assessment and monitoring program
  - Supply chain incident response procedures
  - Supply chain threat intelligence integration
  - Training and awareness for acquisition personnel

Enhancing:
  - Automated supply chain risk monitoring
  - Formal supplier diversity and resilience planning
  - Advanced analytics for supply chain risk prediction
  - Industry information sharing (ISACs, ISAOs)

Mapping to NIST CSF:
  ID.SC-1:  Supply chain risk management processes are identified and agreed to
  ID.SC-2:  Suppliers and partners are assessed using formal assessments
  ID.SC-3:  Contracts with suppliers include security requirements
  ID.SC-4:  Suppliers and partners are routinely assessed
  ID.SC-5:  Response and recovery testing includes suppliers
```

## Contractual Controls

### Security Contract Clauses

```text
Essential Contract Security Clauses:
  1.  Data protection and classification requirements
  2.  Access control and least privilege enforcement
  3.  Encryption requirements (at rest, in transit, in use)
  4.  Security incident notification (timeline: 24-72 hours)
  5.  Breach liability and indemnification
  6.  Right to audit (annual, or upon material event)
  7.  Penetration testing requirements (annual minimum)
  8.  Background check requirements for personnel with access
  9.  Subcontractor/subprocessor approval and flow-down requirements
  10. Data return and destruction upon termination
  11. Business continuity and disaster recovery commitments
  12. Insurance requirements (cyber liability minimums)
  13. Compliance with applicable regulations (GDPR, HIPAA, PCI-DSS)
  14. Change notification for material security changes
  15. SLA for security patching (critical: 24-48 hrs, high: 7 days)
```

## See Also

- Security Awareness
- Privacy Regulations
- AI Governance

## References

- NIST SP 800-161r1: Cybersecurity Supply Chain Risk Management Practices
- Executive Order 14028: Improving the Nation's Cybersecurity
- SLSA Framework: https://slsa.dev
- Sigstore: https://sigstore.dev
- CycloneDX Specification: https://cyclonedx.org
- SPDX Specification: https://spdx.dev
- NTIA SBOM Minimum Elements
- SIG Questionnaire (Shared Assessments)
