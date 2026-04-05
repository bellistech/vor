# Security Architecture

> Design and implement security controls across all layers of an enterprise — from hardware root of trust through application security — using frameworks, zero trust principles, and defense in depth.

## Security Architecture Frameworks

### SABSA (Sherwood Applied Business Security Architecture)

```
# Business-driven security architecture framework
# Six layers aligned with the Zachman Framework:

# Contextual (Business View) — WHY
# - Business requirements, risk appetite, drivers
# - Stakeholder: business owner
# - Question: Why do we need security?

# Conceptual (Architect's View) — WHAT
# - Security concepts, principles, policies
# - Stakeholder: chief architect
# - Question: What security services are needed?

# Logical (Designer's View) — HOW
# - Logical security services and mechanisms
# - Stakeholder: security designer
# - Question: How will security be implemented logically?

# Physical (Builder's View) — WITH WHAT
# - Physical security mechanisms and products
# - Stakeholder: security engineer
# - Question: What specific technologies will be used?

# Component (Tradesman's View) — DETAIL
# - Detailed product configurations, rules, settings
# - Stakeholder: security administrator
# - Question: How are components configured?

# Operational (Service Manager's View) — WHEN/WHO
# - Security operations, monitoring, maintenance
# - Stakeholder: security operations
# - Question: How is security managed day-to-day?
```

### TOGAF Security Architecture

```
# TOGAF ADM (Architecture Development Method) with security overlay
# Security embedded in every phase:

# Phase A: Architecture Vision
# - Security vision aligned with business strategy
# - Identify key security stakeholders

# Phase B: Business Architecture
# - Map security requirements to business processes
# - Identify sensitive data flows

# Phase C: Information Systems Architecture
# - Data security architecture (classification, encryption)
# - Application security patterns (AuthN, AuthZ, input validation)

# Phase D: Technology Architecture
# - Network security zones and boundaries
# - Infrastructure security controls
# - Hardware security (TPM, HSM)

# Phase E: Opportunities and Solutions
# - Security solution evaluation
# - Build vs buy vs outsource for security

# Phase F-H: Migration, Implementation, Change Management
# - Security testing gates
# - Operational security procedures
# - Security change management
```

### Zachman Framework (Security Perspective)

```
# Zachman's 6×6 matrix applied to security:
#
#               What      How       Where     Who       When      Why
# Scope         Assets    Processes Locations  People    Events    Motivation
# Business      Data      Functions Networks   Roles     Schedules Objectives
# System        Models    Workflows Topology   Privileges Triggers  Rules
# Technology    Tables    Programs  Protocols  Identities Cycles    Controls
# Components    Fields    Modules   Addresses  Credentials Intervals Policies
# Operations    Instances Services  Connections Sessions  Timestamps Audits
```

## Defense in Depth

```
# Layer 1: Policies, Procedures, and Awareness
# - Security policies, acceptable use, training
# - Foundation for all other layers

# Layer 2: Physical Security
# - Fences, locks, guards, cameras, mantraps
# - Environmental controls (fire, flood, HVAC)

# Layer 3: Perimeter Security
# - Firewalls, DMZ, WAF, IDS/IPS
# - DDoS protection, content filtering

# Layer 4: Network Security
# - Segmentation (VLANs, microsegmentation)
# - Network monitoring, flow analysis
# - Wireless security (WPA3, 802.1X)

# Layer 5: Host Security
# - OS hardening, patching, EDR
# - Host-based firewall, host IDS
# - Secure boot, integrity monitoring

# Layer 6: Application Security
# - Secure coding practices, input validation
# - Authentication, authorization, session management
# - SAST/DAST, code review

# Layer 7: Data Security
# - Encryption (at rest, in transit, in use)
# - DLP, access controls, masking
# - Backup and recovery

# Principle: failure of any single layer does not compromise
# the system — each layer provides independent protection
# An attacker must defeat ALL layers, not just one
```

## Security Zones and Boundaries

```
# Zone architecture:
#
# Internet (Untrusted)
#     │
#     ├── External Firewall
#     │
# DMZ (Semi-trusted)
#     ├── Web servers, reverse proxies, WAF
#     ├── Mail gateway, DNS (external)
#     ├── VPN concentrator
#     │
#     ├── Internal Firewall
#     │
# Internal Network (Trusted)
#     ├── User workstations
#     ├── Internal servers
#     │
#     ├── Application Firewall / Microsegment
#     │
# Restricted Zone (High Security)
#     ├── Database servers
#     ├── Critical infrastructure
#     ├── HSM / key management
#     │
# Management Zone
#     ├── Admin workstations (jump boxes)
#     ├── SIEM, log collectors
#     ├── Patch/config management
#     └── Out-of-band management (IPMI/iLO/iDRAC)

# Rules:
# Traffic flows from lower trust to higher trust require inspection
# No direct access from untrusted to restricted zone
# Management zone is isolated — out-of-band access only
# Microsegmentation: workload-level isolation within zones
```

## Trusted Computing

### TPM (Trusted Platform Module)

```
# Hardware chip providing cryptographic functions
# Tamper-resistant, bound to the motherboard

# Capabilities:
# - Secure key generation and storage (RSA, ECC)
# - Platform integrity measurement (PCR registers)
# - Sealed storage (decrypt only if platform is in known state)
# - Remote attestation (prove platform configuration to remote party)
# - Random number generation (hardware RNG)

# Measured Boot:
# BIOS → Bootloader → OS Kernel → Drivers → Applications
#   ↓         ↓           ↓          ↓          ↓
# PCR[0]   PCR[4]      PCR[8]     PCR[12]   PCR[14]
# Each stage measures the next into a PCR (Platform Configuration Register)
# PCR values are extended, not overwritten:
#   PCR_new = SHA-256(PCR_old || measurement)
# If any component is tampered, PCR values change → sealed keys won't unseal

# TPM versions:
# TPM 1.2: SHA-1, RSA only, limited PCRs
# TPM 2.0: SHA-256, RSA + ECC, more flexible, required by Windows 11

# Use cases:
# Full disk encryption (BitLocker + TPM)
# Secure boot chain verification
# Certificate-based device identity
# Hardware-backed key storage
```

### HSM (Hardware Security Module)

```
# Dedicated cryptographic hardware appliance
# FIPS 140-2/140-3 validated (levels 1–4)

# FIPS 140 levels:
# Level 1: basic requirements, no physical security
# Level 2: tamper-evident seals, role-based auth
# Level 3: tamper-responsive (zeroize on tamper), identity-based auth
# Level 4: complete physical protection envelope, environmental attacks

# Capabilities:
# - Key generation (RSA, ECC, AES)
# - Key storage (keys never leave the HSM in plaintext)
# - Cryptographic operations (sign, verify, encrypt, decrypt)
# - Random number generation
# - Certificate authority key protection
# - Code signing key protection

# Use cases:
# PKI root CA key protection
# Database TDE master key storage
# Payment processing (PCI HSM for PIN handling)
# DNSSEC key signing
# Code signing for software distribution
# Cloud KMS backed by HSM (AWS CloudHSM, Azure Dedicated HSM)
```

### TEE (Trusted Execution Environment)

```
# Isolated execution environment within the main processor
# Protects code and data from the OS, hypervisor, and other applications

# Intel SGX (Software Guard Extensions)
# - Creates hardware-encrypted memory regions (enclaves)
# - Even kernel/hypervisor cannot access enclave memory
# - Remote attestation proves code running in enclave
# - Enclave page cache (EPC) limits enclave size

# ARM TrustZone
# - Two execution environments: Normal World and Secure World
# - Hardware-enforced isolation at CPU, memory, peripheral level
# - Secure World runs trusted OS and trusted applications
# - Used in: mobile payments, DRM, biometric storage

# AMD SEV (Secure Encrypted Virtualization)
# - Encrypts VM memory with per-VM keys
# - Hypervisor cannot read VM memory in plaintext
# - SEV-ES: encrypts register state
# - SEV-SNP: adds integrity protection

# Confidential Computing:
# Process data while it remains encrypted in memory
# Eliminates the need to trust the infrastructure operator
# Key for multi-tenant cloud and regulated workloads
```

## Cloud Security Architecture

```
# Shared Responsibility Model:
#
# IaaS:
#   Customer: data, apps, OS, middleware, runtime
#   Provider: virtualization, servers, storage, networking, physical
#
# PaaS:
#   Customer: data, apps
#   Provider: runtime, middleware, OS, infrastructure
#
# SaaS:
#   Customer: data (classification, access control)
#   Provider: everything else

# Cloud security reference architecture:
#
# Identity Layer
# ├── Cloud IAM (roles, policies, MFA)
# ├── Federation (SAML, OIDC)
# └── Privileged access management
#
# Network Layer
# ├── VPC / Virtual Network isolation
# ├── Security groups / NACLs
# ├── WAF and DDoS protection
# ├── VPN / private connectivity
# └── DNS security
#
# Compute Layer
# ├── Instance hardening (CIS benchmarks)
# ├── Container security (image scanning, runtime protection)
# ├── Serverless security (function permissions, input validation)
# └── Confidential computing (TEE)
#
# Data Layer
# ├── Encryption at rest (KMS, customer-managed keys)
# ├── Encryption in transit (TLS, VPN)
# ├── Data classification and DLP
# └── Backup and disaster recovery
#
# Monitoring Layer
# ├── Cloud audit logs (CloudTrail, Activity Log, Audit Logs)
# ├── SIEM integration
# ├── CSPM (Cloud Security Posture Management)
# └── Threat detection (GuardDuty, Sentinel, SCC)
```

## Microservices Security

```
# Service-to-service authentication
# - Mutual TLS (mTLS) — each service has its own certificate
# - Service mesh (Istio, Linkerd) — transparent mTLS
# - JWT tokens — signed tokens with service identity and claims
# - SPIFFE/SPIRE — universal identity framework for workloads

# API gateway security
# - Centralized authentication and authorization
# - Rate limiting and throttling
# - Input validation and schema enforcement
# - Request/response transformation
# - Logging and monitoring

# Container security
# - Minimal base images (distroless, scratch, Alpine)
# - Image scanning (Trivy, Grype) in CI/CD pipeline
# - No root in containers (runAsNonRoot)
# - Read-only filesystem
# - Seccomp and AppArmor/SELinux profiles
# - Network policies (Kubernetes NetworkPolicy)
# - Pod security standards (restricted, baseline, privileged)

# Secrets management
# - Never in environment variables or container images
# - Use secrets manager (Vault, AWS Secrets Manager)
# - Rotate credentials automatically
# - Short-lived credentials (OIDC federation, IAM roles)
```

## API Security Architecture

```
# Authentication
# - API keys (simple, low security — for server-to-server)
# - OAuth 2.0 + OIDC (for user-delegated access)
# - JWT bearer tokens (stateless, signed claims)
# - Mutual TLS (certificate-based, highest security)

# Authorization
# - RBAC (role-based) — roles mapped to API scopes
# - ABAC (attribute-based) — fine-grained contextual policies
# - Scope-based (OAuth scopes) — limit what tokens can do

# Input validation
# - Schema validation (OpenAPI/JSON Schema)
# - Parameter type checking and bounds
# - Request size limits
# - Content-type enforcement

# Rate limiting
# - Per-client rate limits (token bucket, sliding window)
# - Global rate limits (circuit breaker)
# - Quota management (daily/monthly limits)

# API security headers
# - CORS (Cross-Origin Resource Sharing) — restrict origins
# - Content-Security-Policy — prevent XSS
# - X-Content-Type-Options: nosniff
# - Strict-Transport-Security (HSTS)

# API versioning for security
# - Deprecate insecure API versions with timeline
# - Force TLS 1.2+ (no fallback to older versions)
# - Remove deprecated authentication methods
```

## Zero Trust Architecture (NIST SP 800-207)

```
# Core principle: "Never trust, always verify"
# No implicit trust based on network location

# Zero Trust tenets:
# 1. All data sources and computing services are resources
# 2. All communication is secured regardless of network location
# 3. Access to resources is granted on a per-session basis
# 4. Access is determined by dynamic policy (identity, device,
#    behavior, environment)
# 5. Enterprise monitors and measures integrity/security posture
#    of ALL assets
# 6. Authentication and authorization are dynamic and strictly
#    enforced before access
# 7. Enterprise collects information about assets, network,
#    communications and uses it to improve security posture

# Zero Trust components:
#
# Policy Engine (PE)
# ├── Makes access decisions based on policy
# ├── Inputs: identity, device health, threat intel, context
# └── Output: grant/deny with conditions
#
# Policy Administrator (PA)
# ├── Executes PE decisions
# ├── Creates/revokes session credentials
# └── Configures data plane enforcement points
#
# Policy Enforcement Point (PEP)
# ├── Enables/terminates connections to resources
# ├── Inline proxy or gateway
# └── Enforces PE/PA decisions in real-time

# Implementation approaches:
# Identity-centric: IAM + strong MFA + ABAC policies
# Network-centric: microsegmentation + software-defined perimeter
# Combined: both identity AND network controls (recommended)

# Maturity levels:
# Traditional:  perimeter security, implicit trust inside
# Advanced:     identity-aware, some microsegmentation
# Optimal:      full zero trust — per-request verification,
#               continuous monitoring, automated response
```

## Secure Design Principles

```
# Least Privilege
# - Grant minimum permissions required for the task
# - Revoke permissions when no longer needed
# - Use just-in-time (JIT) access for privileged operations
# - Implement time-bounded access (expires after X hours)

# Separation of Duties (SoD)
# - No single person can complete a critical transaction alone
# - Example: developer cannot deploy to production without approval
# - Enforced through dual controls, multi-party authorization

# Defense in Depth
# - Multiple independent security controls at each layer
# - Failure of one control does not compromise the system
# - Layers: physical, network, host, application, data

# Fail Secure (Fail Closed)
# - On failure, system denies access by default
# - Firewall rule: default deny, explicit allow
# - Authentication failure: deny access (do not fall back to anonymous)
# - Exception: life safety systems may fail open (fire doors)

# Economy of Mechanism
# - Keep security mechanisms simple and small
# - Simpler systems are easier to verify and audit
# - Fewer components = smaller attack surface

# Complete Mediation
# - Every access attempt is checked against authorization policy
# - No caching of access decisions that could become stale
# - Reference monitor concept: all access passes through a single point

# Open Design
# - Security does not depend on secrecy of implementation
# - Kerckhoffs's principle: system security relies on key secrecy only
# - Open to peer review and scrutiny

# Least Common Mechanism
# - Minimize shared mechanisms between subjects
# - Shared resources create covert channels and coupling
# - Isolate processes, use separate memory spaces

# Psychological Acceptability
# - Security mechanisms must not make resources harder to access
#   than if the mechanisms were not there
# - Users will circumvent security that is too burdensome
```

## Cryptographic Architecture

```
# Key hierarchy:
#
# Master Key (KEK) — stored in HSM, rarely used
#     │
#     ├── Key Encryption Keys — encrypt/decrypt DEKs
#     │
#     ├── Data Encryption Keys (DEKs) — encrypt actual data
#     │       ├── Storage DEK (AES-256)
#     │       ├── Transport DEK (TLS session keys)
#     │       └── Application DEK (column encryption)
#     │
#     └── Signing Keys — authentication and integrity
#             ├── CA signing key (root/intermediate)
#             ├── Code signing key
#             └── API signing key (HMAC)

# Algorithm selection (2024+ recommendations):
# Symmetric encryption:   AES-256-GCM (AEAD)
# Asymmetric encryption:  RSA-3072+ or ECDSA P-256/P-384
# Hashing:               SHA-256 or SHA-3-256
# Key exchange:           ECDHE (X25519 preferred)
# Password hashing:       Argon2id, bcrypt, scrypt
# Post-quantum (future):  CRYSTALS-Kyber (KEM), CRYSTALS-Dilithium (signature)

# Crypto agility — design systems to swap algorithms:
# - Abstract crypto behind interfaces
# - Version/tag algorithm in protocol headers
# - Plan for post-quantum migration
# - Inventory all crypto usage across the enterprise
```

## PKI Architecture

```
# Public Key Infrastructure hierarchy:
#
# Root CA (offline, HSM-protected)
# ├── Issuing CA 1 (server certificates)
# │   ├── web.example.com
# │   ├── api.example.com
# │   └── *.internal.example.com
# ├── Issuing CA 2 (client/user certificates)
# │   ├── alice@example.com
# │   └── bob@example.com
# └── Issuing CA 3 (code signing)
#     └── Software Publisher Certificate

# Certificate lifecycle:
# 1. Key generation (on endpoint or HSM)
# 2. CSR (Certificate Signing Request) creation
# 3. CA validation (DV/OV/EV)
# 4. Certificate issuance
# 5. Certificate deployment
# 6. Certificate monitoring (expiry alerts)
# 7. Certificate renewal/rotation
# 8. Certificate revocation (CRL, OCSP)

# Best practices:
# - Root CA always offline (air-gapped HSM)
# - Short certificate lifetimes (90 days for web, per Let's Encrypt)
# - Automate renewal (ACME protocol / certbot)
# - Monitor certificate transparency logs (CT)
# - Pin certificates only in controlled environments (mobile apps)
# - Cross-sign for trust chain redundancy
# - OCSP stapling over CRL for performance
```

## Tips

- Zero trust is a journey, not a product — start with identity and microsegmentation, then expand.
- Defense in depth means no single layer is trusted to be perfect; every layer assumes the layers above it have been breached.
- HSMs are non-negotiable for PKI root CA keys and payment processing — software key storage is insufficient.
- Fail secure is the default for security systems; fail open is the exception only for life safety.
- Cloud shared responsibility means you still own security of your data and identity, even in SaaS.
- Crypto agility is critical — design systems to swap algorithms without rewriting the application.

## See Also

- zero-trust, cloud-security, pki, cryptography, security-models, firewall-design, container-security

## References

- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/publications/detail/sp/800-207/final)
- [SABSA Institute — SABSA Framework](https://sabsa.org/)
- [TOGAF Security Architecture](https://pubs.opengroup.org/togaf-standard/)
- [NIST SP 800-160 Vol 1 — Systems Security Engineering](https://csrc.nist.gov/publications/detail/sp/800-160/vol-1/final)
- [NIST SP 800-53 Rev 5 — Security and Privacy Controls](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Saltzer & Schroeder — The Protection of Information in Computer Systems (1975)](https://web.mit.edu/Saltzer/www/publications/protection/)
- [TCG — Trusted Platform Module Specification](https://trustedcomputinggroup.org/resource/tpm-library-specification/)
- [CNCF — Cloud Native Security Whitepaper](https://www.cncf.io/blog/2020/11/18/announcing-the-cloud-native-security-white-paper/)
