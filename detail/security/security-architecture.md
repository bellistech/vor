# Security Architecture — Theory and Deep Dive

> *Security architecture is the discipline of designing systems that are resilient to attack by construction, not just by configuration. From the layered matrices of SABSA to the mathematical rigor of zero trust and the hardware guarantees of trusted computing, security architecture provides the structural foundation upon which all operational security controls are built.*

---

## 1. SABSA Framework — Layers and Matrices

### The SABSA Matrix

SABSA defines a 6 x 6 matrix crossing architectural layers with interrogatives:

| Layer | What (Assets) | Why (Motivation) | How (Process) | Who (People) | Where (Location) | When (Time) |
|:---|:---|:---|:---|:---|:---|:---|
| Contextual | Business assets | Business risk | Business processes | Stakeholders | Business geography | Business timescales |
| Conceptual | Security domains | Security policy | Security services | Roles/trust | Security domains | Security lifecycle |
| Logical | Information model | Security rules | Logical mechanisms | Entity model | Domain topology | Processing sequence |
| Physical | Data structures | Security standards | Physical mechanisms | User interface | Platform architecture | Execution schedule |
| Component | Data fields | Control objectives | Component specs | Identity credentials | Network addresses | Clock intervals |
| Operational | Operational data | Operational metrics | Operations procedures | User administration | Operational locations | Operational schedules |

### SABSA Service Management Model

Each cell in the matrix produces a security deliverable. The business-driven approach ensures that every technical control traces back to a business requirement:

$$\text{Control justification} = f(\text{business risk}, \text{risk appetite}, \text{control cost})$$

The key innovation of SABSA is that security requirements flow top-down from business context, not bottom-up from technology capabilities. This prevents the common failure of deploying technology without business justification.

### Attribute Profiling

SABSA uses attribute profiles to bridge business requirements and technical controls:

1. **Business attributes**: what the business needs (privacy, trust, availability)
2. **Architectural attributes**: properties the architecture must exhibit (confidentiality, integrity, resilience)
3. **Measurable attributes**: quantifiable metrics (encryption strength, uptime percentage, MTTR)

$$\text{Business Attribute} \xrightarrow{\text{decompose}} \text{Architectural Attributes} \xrightarrow{\text{implement}} \text{Technical Controls}$$

---

## 2. Security Pattern Language

### Foundational Security Patterns

Security patterns are reusable solutions to recurring security design problems. Key patterns:

**Reference Monitor Pattern:**

All access requests pass through a tamper-resistant, always-invoked, verifiable enforcement point:

```
Subject → [Reference Monitor] → Object
              │
              ├── Complete Mediation (every access checked)
              ├── Tamper-proof (cannot be bypassed or modified)
              └── Verifiable (small enough to be formally verified)
```

The ideal reference monitor is small (enabling verification), always invoked (no bypass), and tamper-resistant. In practice, the Trusted Computing Base (TCB) approximates this.

**Secure Channel Pattern:**

```
Endpoint A ←→ [Authenticated + Encrypted Channel] ←→ Endpoint B
```

Properties:
- **Confidentiality**: eavesdropper cannot read content
- **Integrity**: modification is detected
- **Authentication**: endpoints verified (mutual preferred)
- **Freshness**: replay attacks prevented (nonces/timestamps)

**Checkpoint Pattern (Policy Enforcement Point):**

```
Untrusted Zone → [Checkpoint/PEP] → Trusted Zone
                      │
                      ├── Identity verification
                      ├── Policy evaluation
                      ├── Logging/audit
                      └── Traffic inspection
```

### Pattern Composition

Complex security architectures compose multiple patterns:

$$\text{Architecture} = \text{Pattern}_1 \circ \text{Pattern}_2 \circ \ldots \circ \text{Pattern}_n$$

Example — zero trust service access:

```
User Device → [Device Health Check] → [Identity Provider (MFA)]
    → [Policy Engine] → [PEP/Gateway] → [mTLS Channel] → Service
```

This composes: checkpoint, authentication, policy decision, secure channel, and reference monitor patterns.

---

## 3. Zero Trust Maturity Model

### CISA Zero Trust Maturity Model

The Cybersecurity and Infrastructure Security Agency (CISA) defines five pillars with three maturity levels:

| Pillar | Traditional | Advanced | Optimal |
|:---|:---|:---|:---|
| **Identity** | Password-based, limited MFA | MFA everywhere, some risk-based auth | Continuous validation, phishing-resistant MFA, identity governance |
| **Devices** | Limited visibility, compliance checking | Device inventory, managed devices only | Real-time device health assessment, automated remediation |
| **Network** | Perimeter-based, VPN | Microsegmentation, encrypted traffic | Per-workload microsegment, full encryption, SDP |
| **Application** | On-prem, limited integration | Cloud-aware, some API security | Full ABAC, continuous authorization, workload identity |
| **Data** | Manual classification, perimeter DLP | Automated classification, cloud DLP | Real-time data-centric protection, rights management, automated response |

Cross-cutting capabilities across all pillars:
- **Visibility and Analytics**: logging, SIEM, UEBA
- **Automation and Orchestration**: SOAR, policy-as-code
- **Governance**: continuous compliance monitoring

### Zero Trust Decision Algorithm

For each access request, the Policy Engine evaluates:

$$\text{Decision} = f(\text{Identity}, \text{Device}, \text{Network}, \text{Application}, \text{Data}, \text{Context})$$

Where Context includes:
- Time of access (business hours vs off-hours)
- Geolocation (normal location vs anomalous)
- Behavioral baseline (typical patterns vs deviation)
- Threat intelligence (known-bad indicators)
- Risk score (composite of all signals)

The decision is not binary — it can be:
- **Allow**: full access granted
- **Allow with conditions**: step-up MFA, reduced permissions, session recording
- **Deny**: access refused, alert generated
- **Quarantine**: access suspended pending investigation

### Trust Score Model

$$T_{\text{score}} = w_1 \cdot I + w_2 \cdot D + w_3 \cdot N + w_4 \cdot B + w_5 \cdot C$$

Where:
- $I$ = identity confidence (authentication strength, credential age)
- $D$ = device health (patch level, compliance, management status)
- $N$ = network trustworthiness (corporate managed, known VPN, public wifi)
- $B$ = behavioral score (deviation from baseline)
- $C$ = context score (time, location, threat intelligence)
- $w_i$ = weights reflecting organizational risk priorities

Access is granted when $T_{\text{score}} \geq \theta_{\text{resource}}$, where $\theta$ is the trust threshold for the requested resource (more sensitive resources require higher scores).

---

## 4. Cloud Security Reference Architecture

### CSA (Cloud Security Alliance) Reference Architecture

```
Cloud Security Architecture Layers:

Governance and Risk Layer
├── Cloud governance framework
├── Risk management (shared responsibility)
├── Compliance mapping (FedRAMP, SOC 2, ISO 27017)
└── Audit and assurance

Identity and Access Layer
├── Cloud IAM (centralized identity)
├── Federation and SSO (SAML, OIDC)
├── Privileged access management (PAM)
├── Service accounts and workload identity
└── Temporary credentials (STS, assumed roles)

Infrastructure Security Layer
├── Virtual network isolation (VPC, VNet)
├── Microsegmentation (security groups, NSGs)
├── Encryption (in transit, at rest, in use)
├── Key management (KMS, customer-managed keys)
└── Hardware security (CloudHSM, confidential VMs)

Application Security Layer
├── Secure CI/CD pipeline
├── Container/serverless security
├── API gateway and WAF
├── Secrets management
└── Dependency scanning

Data Security Layer
├── Data classification and discovery
├── Encryption and tokenization
├── DLP and rights management
├── Backup and DR
└── Data residency and sovereignty

Operations Security Layer
├── Cloud-native monitoring (CloudWatch, Monitor, Operations)
├── CSPM (Cloud Security Posture Management)
├── CWPP (Cloud Workload Protection Platform)
├── CNAPP (Cloud-Native Application Protection Platform)
└── Incident response (cloud-specific playbooks)
```

### Multi-Cloud Security Challenges

| Challenge | Single Cloud | Multi-Cloud |
|:---|:---|:---|
| Identity | Single IAM | Identity federation across providers |
| Networking | Single VPC model | Cross-cloud connectivity, consistent policy |
| Encryption | Single KMS | Key management across providers |
| Monitoring | Native tools | Unified visibility across clouds |
| Compliance | Single audit scope | Compliance per provider + aggregate |
| Skills | One platform expertise | Multiple platform expertise |

Multi-cloud security requires an abstraction layer (CSPM/CNAPP) that normalizes security controls across providers.

---

## 5. Hardware Root of Trust

### Chain of Trust

The security of a system ultimately rests on a hardware root of trust — the foundational component assumed to be secure:

```
Hardware Root of Trust (TPM/HSM/TEE)
    │
    ├── Firmware (UEFI Secure Boot verifies firmware signatures)
    │
    ├── Bootloader (measured by firmware, verified by TPM)
    │
    ├── OS Kernel (measured by bootloader, verified by TPM)
    │
    ├── Kernel Modules/Drivers (signed, loaded by kernel)
    │
    ├── Services and Daemons (launched by verified OS)
    │
    └── Applications (running in verified environment)
```

Each layer measures the next, creating an unbroken chain from hardware to application. If any link is compromised, all subsequent measurements are untrusted.

### Remote Attestation Protocol

Remote attestation allows a verifier to confirm the integrity of a remote platform:

```
1. Verifier → Platform: "Prove your configuration"
2. Platform: TPM generates Quote (signed PCR values)
3. Platform → Verifier: Quote + Event Log
4. Verifier:
   a. Verify Quote signature (TPM's attestation key)
   b. Replay Event Log to compute expected PCR values
   c. Compare expected vs actual PCR values
   d. Check known-good database for acceptable configurations
5. Decision: Allow or deny based on platform integrity
```

The attestation is only as trustworthy as:
- The TPM hardware (assumed secure)
- The known-good database (must be maintained)
- The freshness of the attestation (replay protection via nonce)

### Secure Boot vs Measured Boot

| Aspect | Secure Boot | Measured Boot |
|:---|:---|:---|
| Action on tamper | Refuse to boot | Boot, but record measurements |
| Enforcement | Preventive (stop bad code) | Detective (detect and report) |
| Flexibility | Strict — only signed code runs | Flexible — any code can run |
| Attestation | Not inherent | Yes — remote verification |
| Standard | UEFI Secure Boot | TCG Measured Boot |
| Use case | Consumer devices, locked platforms | Servers, cloud instances |

Best practice: use both — Secure Boot prevents known-bad, Measured Boot provides attestation evidence.

---

## 6. Formal Methods in Security Architecture

### Security Properties — Formal Definitions

**Safety property** — something bad never happens:

$$\forall t : \neg \text{Bad}(t)$$

Examples: "classified data is never sent to an uncleared recipient," "no buffer overflow occurs."

**Liveness property** — something good eventually happens:

$$\forall t : \exists t' > t : \text{Good}(t')$$

Examples: "every access request receives a response," "revoked certificates are eventually distributed."

Most security properties are safety properties. Availability is a liveness property.

### Attack Surface Analysis

The attack surface is the set of points where an attacker can attempt to interact with the system:

$$\text{Attack Surface} = \sum_{i} w_i \cdot |\text{entry points}_i|$$

Where $w_i$ is the risk weight of each entry point type:

| Entry Point Type | Weight (Relative Risk) |
|:---|:---:|
| Network-facing services (public) | High |
| API endpoints (authenticated) | Medium-High |
| File upload/processing | Medium-High |
| User input fields | Medium |
| Administrative interfaces | High |
| Internal service-to-service | Low-Medium |
| Physical interfaces (USB, serial) | Medium |

Attack surface reduction strategies:
- Minimize exposed services (close unused ports)
- Reduce code complexity (fewer features = fewer bugs)
- Remove default accounts and credentials
- Disable unnecessary protocols and features
- Microsegment to limit lateral movement

### Threat-Driven Architecture

Architecture decisions driven by threat modeling (rather than feature requirements alone):

```
1. Identify assets worth protecting
2. Enumerate threat actors and their capabilities
3. Map attack vectors using STRIDE or MITRE ATT&CK
4. Design architecture to address specific threats:
   - Spoofing → authentication controls
   - Tampering → integrity controls
   - Repudiation → logging and non-repudiation
   - Information Disclosure → encryption and access control
   - Denial of Service → redundancy and rate limiting
   - Elevation of Privilege → least privilege and sandboxing
5. Validate architecture against threat model
6. Iterate as threats evolve
```

---

## 7. Security Architecture Evaluation

### Architecture Trade-off Analysis Method (ATAM)

ATAM evaluates architecture against quality attributes including security:

1. **Present architecture** — describe the system
2. **Identify architectural approaches** — security patterns used
3. **Generate quality attribute utility tree** — prioritize security scenarios
4. **Analyze architectural approaches** — find sensitivity points and tradeoffs
5. **Identify risks and non-risks** — document architectural decisions that affect security

### Security Metrics for Architecture Evaluation

| Metric | Measurement | Target |
|:---|:---|:---|
| Attack surface size | Exposed endpoints, services, APIs | Minimize |
| Defense depth | Number of independent security layers | $\geq$ 3 |
| Single points of failure | Components without redundancy | 0 for critical paths |
| Blast radius | Impact scope of a single compromised component | Minimize via segmentation |
| Time to detect (TTD) | Mean time from compromise to detection | $<$ 1 hour |
| Time to contain (TTC) | Mean time from detection to containment | $<$ 4 hours |
| Crypto agility score | Time to rotate/replace a cryptographic algorithm | $<$ 1 sprint |
| Zero trust coverage | Percentage of access requests with identity verification | 100% |

### Common Architecture Anti-Patterns

| Anti-Pattern | Risk | Remedy |
|:---|:---|:---|
| Perimeter-only security | Single point of failure | Defense in depth + zero trust |
| Shared credentials | No accountability, impossible revocation | Per-entity identity, secrets management |
| Implicit trust by network | Lateral movement after breach | Microsegmentation, zero trust |
| Security as afterthought | Expensive retrofitting, gaps | Security by design, threat modeling |
| Monolithic secrets | Single key compromise = total loss | Key hierarchy, envelope encryption |
| Direct database exposure | SQL injection, data exfiltration | API layer, parameterized queries |

---

## 8. Summary — Architecture Decision Framework

| Architecture Question | Framework/Approach | Key Decision |
|:---|:---|:---|
| How to align security with business? | SABSA attribute profiling | Top-down from business risk |
| What trust model to adopt? | NIST 800-207 zero trust | Never trust, always verify |
| Where to place controls? | Defense in depth layers | Independent controls at every layer |
| How to establish trust? | Hardware root of trust (TPM/HSM) | Anchor trust in tamper-resistant hardware |
| How to secure cloud workloads? | Shared responsibility + CNAPP | Customer owns data and identity |
| How to evaluate the architecture? | ATAM + security metrics | Measure attack surface, blast radius, TTD |
| How to handle future threats? | Crypto agility + threat-driven design | Design for algorithm replacement |

## Prerequisites

- security models, networking fundamentals, cryptography, cloud computing, systems engineering

## Complexity

| Operation | Time | Space |
|-----------|------|-------|
| SABSA attribute profiling | O(n × m) attributes × layers | O(n × m) |
| Zero trust policy evaluation | O(k) signals per request | O(1) per decision |
| Remote attestation (TPM) | O(n) PCR measurements | O(n) event log |
| Attack surface enumeration | O(n) services + endpoints | O(n) |
| Architecture evaluation (ATAM) | O(n × s) approaches × scenarios | O(n × s) |

---

*Security architecture is the discipline of making explicit, defensible decisions about how systems resist attack. Every architectural pattern, every trust boundary, and every cryptographic choice embodies a judgment about threats, costs, and acceptable risk. The frameworks and methods above transform these judgments from intuition into rigorous, auditable engineering practice.*
