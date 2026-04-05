# The Architecture of Supply Chain Trust -- Provenance, Attestation, and Systemic Risk

> *Supply chain security is fundamentally a trust propagation problem. Every software artifact inherits the security posture of every tool, library, build system, and human that touched it. The challenge is establishing verifiable trust chains in systems with thousands of transitive dependencies.*

---

## 1. Supply Chain Threat Landscape

### The Attack Surface Explosion

Modern software has an enormous transitive dependency tree:

$$\text{Total Dependencies} = \sum_{i=0}^{d} \text{direct}_i \times \text{avg\_transitive\_per\_direct}$$

A typical Node.js application with 50 direct dependencies often has 500-1,500 transitive dependencies. Each dependency is a potential attack vector.

**Attack surface quantification**:

$$\text{Attack Surface} = \sum_{i=1}^{n} P(\text{compromise}_i) \times \text{Impact}_i$$

where $n$ = total components in the supply chain and $P(\text{compromise}_i)$ is the probability of component $i$ being compromised.

### The Trust Chain Problem

Software trust is transitive but non-symmetric:

$$A \xrightarrow{\text{trusts}} B \xrightarrow{\text{trusts}} C \;\not\Rightarrow\; C \xrightarrow{\text{trusts}} A$$

If you trust library $B$ and $B$ depends on $C$, you implicitly trust $C$ even though you have no direct relationship with $C$'s maintainer, build system, or security practices.

**Trust degradation over hops**:

$$\text{Trust}(A, C) = \prod_{i=1}^{h} \text{Trust}(e_i) \leq \min_{i} \text{Trust}(e_i)$$

Trust across $h$ hops is bounded by the weakest link.

### Supply Chain Attack Economics

```text
Attacker ROI Analysis:
  Traditional attack:  1 target, moderate effort, limited impact
  Supply chain attack: 1 compromise, cascading impact across thousands

  Cost to attacker:
    Package typosquatting:        ~$0 (register package name)
    Dependency confusion:         ~$0 (register internal name on public registry)
    Maintainer social engineering: weeks-months of effort (xz Utils: 2+ years)
    Build system compromise:      high skill, but enormous payoff (SolarWinds)

  Expected value for attacker:
    E[value] = P(success) x N(victims) x Value(per_victim)

    SolarWinds: P(success)=1, N=18,000, Value=$millions each
    npm typosquat: P(success)=0.3, N=hundreds, Value=$thousands each
```

---

## 2. SBOM Generation and Consumption

### SBOM Data Model

An SBOM captures the component graph of a software artifact:

$$\text{SBOM} = (C, D, M)$$

where $C$ = set of components, $D$ = dependency relationships, $M$ = metadata.

Each component $c_i \in C$ contains:

| Field | Description | Example |
|:---|:---|:---|
| name | Component name | `requests` |
| version | Exact version | `2.31.0` |
| purl | Package URL | `pkg:pypi/requests@2.31.0` |
| cpe | Common Platform Enumeration | `cpe:2.3:a:python:requests:2.31.0:*:*:*:*:*:*:*` |
| licenses | SPDX license identifiers | `Apache-2.0` |
| hashes | Cryptographic digests | SHA-256, SHA-512 |
| supplier | Organization that supplied | `Kenneth Reitz` |
| externalRefs | Links to VEX, security advisories | Advisory URLs |

### CycloneDX vs SPDX

```text
Feature              CycloneDX                    SPDX
Origin               OWASP                        Linux Foundation
Primary focus        Security/risk                 License compliance
Formats              JSON, XML, Protobuf           JSON, RDF, YAML, tag-value
VEX support          Native (embedded)             Separate document
Dependency graph     Full DAG support              Relationship types
Vulnerability data   Integrated via VEX            External linkage
Adoption             Security tooling, DevSecOps   Legal/compliance, government
Spec complexity      Moderate                      Higher (more flexible)
```

### VEX (Vulnerability Exploitability eXchange)

VEX provides context on whether a vulnerability in a component actually affects the product:

$$\text{VEX Status} \in \{\text{not\_affected}, \text{affected}, \text{fixed}, \text{under\_investigation}\}$$

This addresses the false positive problem in vulnerability scanning. A library may contain a vulnerable function, but if that function is never called in the consuming application, the vulnerability is "not affected."

**VEX reduces noise**:

$$\text{Actionable Vulns} = \text{Total CVEs in SBOM} - \text{VEX(not\_affected)} - \text{VEX(fixed)}$$

In practice, VEX can reduce actionable vulnerabilities by 60-80% compared to raw SBOM scanning.

---

## 3. Supply Chain Attack Taxonomy

### Taxonomy by Kill Chain Phase

```text
Phase 1: Initial Compromise (Entry Point)
  ├── Source compromise
  │   ├── Maintainer account takeover
  │   ├── Social engineering of maintainer (xz Utils)
  │   └── Malicious contribution (Trojan Source)
  ├── Build compromise
  │   ├── CI/CD pipeline injection
  │   ├── Build tool tampering
  │   └── Compiler manipulation (Thompson attack)
  ├── Distribution compromise
  │   ├── Package registry poisoning
  │   ├── Update server compromise
  │   └── Mirror/CDN tampering
  └── Dependency compromise
      ├── Typosquatting
      ├── Dependency confusion
      ├── Star-jacking
      └── Package hijacking

Phase 2: Payload (What Malicious Code Does)
  ├── Data exfiltration (env vars, secrets, source code)
  ├── Backdoor installation (persistent access)
  ├── Cryptocurrency mining
  ├── Ransomware deployment
  ├── Credential harvesting
  └── Lateral movement enabling

Phase 3: Persistence and Propagation
  ├── Self-replicating (infect other packages)
  ├── Version pinning evasion (affects all versions)
  ├── Build-time only (not in source, only in artifact)
  └── Time-delayed activation (sleeper code)
```

### The xz Utils Attack Pattern (Long-game Social Engineering)

The xz Utils compromise (CVE-2024-3094) represents a new class of supply chain attack -- the patient, long-term maintainer trust exploitation:

```text
Timeline:
  2021:       Attacker begins contributing helpful patches
  2022:       Attacker becomes trusted contributor
  2023 Q1:    Attacker becomes co-maintainer
  2023 Q3:    Attacker pressures original maintainer (burnout exploitation)
  2024 Feb:   Malicious code inserted via test fixtures (obfuscated)
  2024 Mar:   Backdoor in xz 5.6.0/5.6.1 releases
  2024 Mar 29: Discovered by Andres Freund (500ms SSH latency anomaly)

Key Lessons:
  - Long-term social engineering bypasses all technical controls
  - Open source maintainer burnout is an exploitable vulnerability
  - Binary test fixtures can hide malicious code from code review
  - Build-time injection can differ from source-visible code
  - Performance anomalies (latency) can reveal backdoors
```

---

## 4. Software Provenance (SLSA and Sigstore)

### SLSA Provenance Model

SLSA (Supply-chain Levels for Software Artifacts) defines a provenance attestation format:

$$\text{Provenance} = \text{Builder} + \text{Source} + \text{Build Config} + \text{Materials} + \text{Output}$$

The provenance document answers three critical questions:
1. **Who** built it? (Builder identity)
2. **What** source was used? (Git commit, branch, repo)
3. **How** was it built? (Build configuration, parameters)

### Trust Boundaries in SLSA

```text
SLSA Level 1: Documentation
  Trust boundary: None (provenance exists but is self-attested)
  Threat model: Protects against accidental errors, not malicious actors
  Verification: Provenance document exists and is parseable

SLSA Level 2: Build Service
  Trust boundary: Build service (separate from developer)
  Threat model: Developer cannot falsify provenance
  Verification: Provenance signed by build service

SLSA Level 3: Hardened Build
  Trust boundary: Hardened, isolated build platform
  Threat model: Compromised build config cannot affect other builds
  Verification: Non-falsifiable provenance from trusted builder

  Requirements:
    - Ephemeral, isolated build environments
    - Build service generates provenance (not user-provided)
    - Provenance cannot be modified after generation
    - Build inputs are fully declared and verifiable
```

### Sigstore Architecture

Sigstore provides keyless signing -- no long-lived keys to manage:

$$\text{Sigstore} = \text{Fulcio} + \text{Rekor} + \text{Cosign}$$

| Component | Role | Analogy |
|:---|:---|:---|
| Fulcio | Certificate Authority -- issues short-lived certs from OIDC identity | Let's Encrypt for code signing |
| Rekor | Transparency log -- immutable record of signing events | Certificate Transparency for artifacts |
| Cosign | Signing/verification tool -- creates and verifies signatures | GPG replacement |

**Keyless signing flow**:

$$\text{Developer} \xrightarrow{\text{OIDC}} \text{Fulcio} \xrightarrow{\text{cert}} \text{sign(artifact)} \xrightarrow{\text{record}} \text{Rekor}$$

1. Developer authenticates via OIDC (GitHub, Google, Microsoft identity)
2. Fulcio issues a short-lived certificate (10 minutes) binding OIDC identity to signing key
3. Cosign signs the artifact with the ephemeral key
4. Signing event is recorded in Rekor transparency log
5. Ephemeral key is discarded -- no key management burden

**Verification**:

$$\text{verify}(\text{artifact}, \text{signature}) = \text{check\_cert}(\text{Fulcio}) \wedge \text{check\_log}(\text{Rekor})$$

---

## 5. Hardware Root of Trust

### Trusted Platform Module (TPM)

The TPM provides a hardware-anchored trust foundation:

$$\text{Trust Chain}: \text{TPM} \rightarrow \text{UEFI Firmware} \rightarrow \text{Bootloader} \rightarrow \text{Kernel} \rightarrow \text{OS} \rightarrow \text{Application}$$

**Platform Configuration Registers (PCRs)**: Each boot stage extends the measurement chain:

$$\text{PCR}[n]_{\text{new}} = H(\text{PCR}[n]_{\text{old}} \| \text{measurement})$$

where $H$ is SHA-256. PCRs can only be extended, never set directly. This creates an unforgeable record of the boot sequence.

| PCR | Contents |
|:---|:---|
| 0 | BIOS/UEFI firmware |
| 1 | BIOS/UEFI configuration |
| 2-3 | Option ROMs, platform configuration |
| 4-5 | MBR, bootloader |
| 7 | Secure Boot policy |
| 8-9 | OS kernel, initrd |
| 14 | Shim/MOK (Linux Secure Boot) |

**Remote Attestation**:

A verifier can request a TPM quote (signed PCR values) to confirm a remote machine's software state matches expected values. This is the foundation of confidential computing and zero-trust architectures.

### Secure Boot Chain

```text
UEFI Secure Boot:
  PK  (Platform Key)     -- OEM root of trust (one per machine)
  KEK (Key Exchange Key) -- keys authorized to update db/dbx
  db  (Signature DB)     -- authorized bootloader/kernel signatures
  dbx (Forbidden DB)     -- revoked/blacklisted signatures

Boot verification:
  UEFI → verify(bootloader, db) → verify(kernel, db) → measured boot

  Any signature mismatch → boot halted

Linux with shim:
  UEFI → shim (Microsoft-signed) → GRUB (distro-signed) → kernel (distro-signed)
```

---

## 6. Vendor Concentration Risk

### Single-Point-of-Failure Analysis

$$\text{Concentration Risk} = \frac{\text{Critical functions served by vendor}}{\text{Total critical functions}} \times \text{Replaceability Factor}$$

where Replaceability Factor ranges from 0 (trivially replaceable) to 1 (irreplaceable).

**Concentration risk scenarios**:

| Scenario | Risk Level | Mitigation |
|:---|:---|:---|
| Single cloud provider for all workloads | Critical | Multi-cloud or cloud-agnostic design |
| Single CDN provider | High | Secondary CDN on standby |
| One authentication provider (IdP) | Critical | Federated identity, backup IdP |
| Single DNS provider | High | Secondary DNS (dual provider) |
| Monoculture OS across all servers | High | Heterogeneous OS strategy |

### Dependency Graph Analysis

For a software project with dependency graph $G = (V, E)$:

**Centrality analysis** identifies critical dependencies:

$$\text{Betweenness}(v) = \sum_{s \neq v \neq t} \frac{\sigma_{st}(v)}{\sigma_{st}}$$

where $\sigma_{st}$ = number of shortest paths from $s$ to $t$, and $\sigma_{st}(v)$ = those paths passing through $v$.

A dependency with high betweenness centrality is a critical chokepoint -- if compromised, it affects many downstream packages. Log4j had extremely high betweenness centrality in the Java ecosystem.

---

## 7. Supply Chain Resilience Theory

### Resilience Framework

$$\text{Resilience} = \text{Resistance} + \text{Recovery} + \text{Adaptation}$$

- **Resistance**: ability to withstand supply chain disruptions (redundancy, diversity)
- **Recovery**: speed of restoring operations after disruption (alternatives, playbooks)
- **Adaptation**: ability to restructure supply chain based on lessons learned

### Defense in Depth for Supply Chain

```text
Layer 1: Prevention
  - Verified sources (signed packages, provenance checks)
  - Minimal dependencies (reduce attack surface)
  - Lockfile integrity (committed, reviewed in PRs)
  - Private registries (artifact proxy, no direct public access)
  - Dependency review automation (Dependabot, Renovate with security focus)

Layer 2: Detection
  - SBOM-based vulnerability scanning (continuous)
  - Build reproducibility verification
  - Behavioral analysis of dependencies (what syscalls, network calls)
  - Package diff review (what changed between versions)
  - Anomaly detection (unexpected new maintainer, sudden burst of releases)

Layer 3: Containment
  - Least privilege for dependencies (sandboxing, capabilities)
  - Network segmentation for build systems
  - Immutable build environments (ephemeral containers)
  - Package firewall (block known-malicious packages)

Layer 4: Recovery
  - Vendor-agnostic architecture (avoid lock-in)
  - Alternative supplier identification
  - Rollback capability (pin to known-good versions)
  - Incident response playbook for supply chain compromises
```

---

## 8. Regulatory Requirements (EO 14028 and Beyond)

### Executive Order 14028 (May 2021)

EO 14028 mandated supply chain security improvements for US federal software:

**Key requirements**:

1. **SBOM for all software** sold to the federal government
2. **Secure development practices** (NIST SSDF SP 800-218)
3. **Vulnerability disclosure programs** for software vendors
4. **Attestation** of conformance to secure development practices

**SBOM minimum elements** (NTIA):

| Element | Description |
|:---|:---|
| Supplier Name | Entity that created or distributes component |
| Component Name | Name assigned to the software unit |
| Version | Version identifier |
| Unique Identifier | Unique identification (purl, CPE) |
| Dependency Relationship | Upstream/downstream mapping |
| Author of SBOM Data | Entity that generated the SBOM |
| Timestamp | Date/time SBOM was generated |

### EU Cyber Resilience Act (CRA)

The CRA introduces mandatory cybersecurity requirements for products with digital elements in the EU market:

```text
Key Requirements:
  - Security by design and default
  - Vulnerability handling obligations (coordinated disclosure)
  - SBOM for products (including open source components)
  - Conformity assessment (self-assessment or third-party)
  - 24-hour vulnerability notification to ENISA
  - Free security updates for product lifetime (minimum 5 years)
  - CE marking for compliant products

Categories:
  Default:     Self-assessment (most products)
  Important:   Harmonized standards or third-party assessment
  Critical:    Third-party assessment required (e.g., firewalls, HSMs)
```

### Convergence of Regulations

$$\text{Global Trend}: \text{EO 14028} + \text{EU CRA} + \text{NIST SSDF} + \text{PCI-DSS v4} \rightarrow \text{Universal SBOM requirement}$$

Organizations that invest in SBOM generation, software provenance, and supply chain risk management now will be positioned for compliance across all emerging regulatory frameworks.

---

## See Also

- Security Awareness
- Privacy Regulations
- AI Governance

## References

- NIST SP 800-161r1: C-SCRM Practices for Systems and Organizations
- NIST SP 800-218: Secure Software Development Framework (SSDF)
- SLSA Specification: https://slsa.dev/spec
- Sigstore Documentation: https://docs.sigstore.dev
- NTIA SBOM Minimum Elements (2021)
- Executive Order 14028 (May 2021)
- EU Cyber Resilience Act (2024)
- Ohm, M. et al. (2020): Backstabber's Knife Collection: A Review of Open Source Software Supply Chain Attacks
