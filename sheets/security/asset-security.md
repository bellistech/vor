# Asset Security

> Classify, protect, and manage data throughout its lifecycle — from creation through destruction — with proper ownership, handling, retention, and privacy controls.

## Data Classification

### Government / Military Classification

```
# Top Secret (TS)
# - Unauthorized disclosure could cause exceptionally grave damage
#   to national security
# - Examples: weapons design, intelligence sources, war plans
# - Requires highest level of clearance and need-to-know

# Secret (S)
# - Unauthorized disclosure could cause serious damage
#   to national security
# - Examples: military plans, diplomatic negotiations
# - Requires Secret clearance and need-to-know

# Confidential (C)
# - Unauthorized disclosure could cause damage
#   to national security
# - Examples: troop movements, technical manuals
# - Requires Confidential clearance and need-to-know

# Unclassified (U)
# - No damage to national security from disclosure
# - May still be sensitive (FOUO, SBU, CUI)
# - CUI (Controlled Unclassified Information) replaces many legacy markings

# Clearance hierarchy:
# Top Secret > Secret > Confidential > Unclassified
# Higher clearance grants access to that level AND all below
# BUT need-to-know still required (clearance alone is not sufficient)
```

### Commercial / Private Sector Classification

```
# Confidential (Restricted)
# - Most sensitive business data
# - Disclosure causes significant financial or legal harm
# - Examples: trade secrets, M&A plans, customer PII, source code
# - Access: named individuals only, encrypted, strict logging

# Internal (Private)
# - For internal use within the organization
# - Disclosure could cause moderate harm
# - Examples: internal policies, project plans, org charts
# - Access: all employees, not shared externally

# Public
# - Information intended for public access
# - No harm from disclosure (already published or approved)
# - Examples: marketing materials, press releases, public APIs
# - Access: unrestricted

# Some organizations add a fourth level:
# Highly Confidential > Confidential > Internal > Public

# Classification must be assigned by the data owner
# When in doubt, classify at the HIGHER level
# Reclassification requires data owner approval
```

## Data States

```
# Data at Rest — stored on persistent media
# Threats: physical theft, unauthorized access, backup exposure
# Controls:
#   Full disk encryption (LUKS, BitLocker, FileVault)
#   Database encryption (TDE — Transparent Data Encryption)
#   File-level encryption (GPG, age)
#   Access controls (filesystem permissions, ACLs)
#   Physical security (locked rooms, hardware security)

# Data in Transit — moving across a network
# Threats: eavesdropping, MITM, packet capture, replay attacks
# Controls:
#   TLS 1.3 for web traffic (HTTPS)
#   IPsec for network-level encryption
#   SSH/SCP/SFTP for secure file transfer
#   VPN for remote access
#   Certificate pinning for mobile apps
#   DNSSEC for DNS integrity

# Data in Use — actively being processed in memory
# Threats: memory scraping, cold boot attacks, side channels
# Controls:
#   Trusted Execution Environments (TEE — Intel SGX, ARM TrustZone)
#   Homomorphic encryption (compute on encrypted data)
#   Secure enclaves
#   Memory encryption (AMD SME/SEV)
#   Process isolation
#   Minimize plaintext exposure time
```

## Data Handling Procedures

```
# Labeling / Marking
# - All data must bear its classification marking
# - Physical media: labels, stamps, cover sheets
# - Digital: metadata tags, headers, footers, watermarks
# - Email: subject line prefix [CONFIDENTIAL], classification banner
# - Files: filename prefix or metadata property
# - Systems: banner at login, classification in page headers

# Storage
# - Classified data: encrypted storage, access-controlled locations
# - Physical: locked cabinets, safes, secure rooms (proportional to class)
# - Digital: encrypted volumes, access-controlled directories
# - Cloud: encryption with customer-managed keys (BYOK/HYOK)
# - Separation: different classification levels in separate containers

# Transmission
# - Confidential+: encrypted channel required (TLS, VPN, encrypted email)
# - Physical transport: tamper-evident packaging, courier, chain of custody
# - Fax: pre-arrange with recipient, confirm receipt
# - Email: encrypted (S/MIME, PGP), no confidential data in subject line
# - Never transmit classification credentials alongside classified data

# Destruction (see Media Sanitization section)
# - Must be irreversible and proportional to classification level
# - Destruction must be witnessed and documented for high classifications
# - Certificates of destruction for regulated data
```

## Data Lifecycle

```
# 1. Create / Collect
#    - Assign classification at creation
#    - Identify data owner
#    - Apply minimum necessary collection (data minimization)
#    - Document purpose for collection (especially PII)

# 2. Store
#    - Encrypt at rest per classification level
#    - Apply access controls (least privilege)
#    - Log all access
#    - Maintain backup copies per retention policy

# 3. Use / Process
#    - Access granted based on clearance + need-to-know
#    - Processing must maintain classification level
#    - Output classification = highest input classification
#    - Audit trail for all operations on sensitive data

# 4. Share / Distribute
#    - Verify recipient authorization before sharing
#    - Use approved channels for the classification level
#    - Apply DLP rules to prevent unauthorized sharing
#    - Third-party sharing requires data sharing agreement
#    - Cross-border transfer requires legal review

# 5. Archive
#    - Move to long-term storage per retention schedule
#    - Maintain encryption and access controls
#    - Ensure archived data remains retrievable
#    - Index for legal discovery / e-discovery compliance

# 6. Destroy / Dispose
#    - Destroy when retention period expires
#    - Use sanitization method appropriate to media and classification
#    - Document destruction (certificate of destruction)
#    - Cannot destroy data under legal hold
```

## Data Ownership Roles

```
# Data Owner (business executive / senior management)
# - Ultimately responsible for the data
# - Sets classification level
# - Defines access policies (who can access, under what conditions)
# - Approves access requests
# - Determines retention requirements
# - Decides on risk acceptance for the data

# Data Custodian (IT operations / technical staff)
# - Implements and maintains protections defined by owner
# - Manages storage, backups, encryption
# - Implements access controls per owner's policy
# - Monitors and reports on data protection status
# - Does NOT set policy — executes owner's decisions

# Data Steward (data governance / quality role)
# - Ensures data quality, accuracy, consistency
# - Manages metadata and data dictionaries
# - Coordinates between owners and custodians
# - Enforces data standards and naming conventions
# - Handles data lineage and provenance tracking

# Data Processor (GDPR term)
# - Processes personal data on behalf of the controller
# - Must follow controller's instructions
# - Cannot use data for own purposes
# - Must implement appropriate security measures
# - Examples: cloud provider, payroll service, analytics vendor

# Data Controller (GDPR term)
# - Determines purpose and means of processing personal data
# - Responsible for compliance with data protection laws
# - Must demonstrate lawful basis for processing
# - Responds to data subject access requests (DSARs)
# - Notifies authorities of breaches (72 hours under GDPR)

# Data Subject
# - The individual whose personal data is being processed
# - Has rights: access, rectification, erasure, portability, objection
```

## Data Retention Policies

```
# Retention policy components:
# - What data is retained (scope by type/classification)
# - How long it is retained (retention period)
# - Where it is stored during retention (storage location)
# - When and how it is destroyed (disposition)
# - Legal holds (override normal retention for litigation)

# Common regulatory retention periods:
# Tax records:              7 years (IRS)
# Employee records:         7 years after termination
# Medical records (HIPAA):  6 years from last date of service
# Financial records (SOX):  7 years
# PCI cardholder data:      per business need, minimize retention
# GDPR personal data:       only as long as purpose requires
# SEC broker-dealer:        3–6 years depending on record type
# Education records (FERPA): duration of enrollment + retention period

# Key principles:
# Keep data only as long as legally required or business-necessary
# Shorter retention = smaller attack surface
# Legal hold supersedes retention policy (do NOT destroy)
# Document retention decisions and exceptions
# Automate retention enforcement where possible
```

## Media Sanitization (NIST SP 800-88)

```
# Clear — logical overwrite
# - Overwrite with fixed patterns (zeros, ones, random)
# - Protects against: simple file recovery, undelete tools
# - Does NOT protect against: laboratory-level recovery
# - Use for: reuse within the same organization, same security level
# - Methods: single-pass overwrite, built-in secure erase command

# Purge — advanced overwrite or degaussing
# - Makes data unrecoverable by any known laboratory technique
# - Methods:
#   Block erase (flash/SSD)
#   Cryptographic erase (destroy encryption key, data unreadable)
#   Degaussing (magnetic media only — destroys magnetic patterns)
#   Firmware-level secure erase (ATA Secure Erase)
# - Use for: reuse outside the organization or lower security level
# - Cryptographic erase is the most practical for encrypted media

# Destroy — physical destruction
# - Renders media physically unusable
# - Methods:
#   Shredding (cross-cut, particle size per classification)
#   Disintegration (industrial disintegrator)
#   Incineration (licensed incinerator)
#   Pulverization (grinding to dust)
#   Melting (smelter)
# - Use for: highest classification levels, end-of-life media
# - Only option for damaged media that cannot be electronically sanitized

# Media type considerations:
# HDD (magnetic):  clear=overwrite, purge=degauss, destroy=shred
# SSD (flash):     clear=overwrite (unreliable), purge=crypto erase, destroy=shred
# Tape:            clear=overwrite, purge=degauss, destroy=incinerate
# Optical (CD/DVD): destroy only (shred/incinerate)
# Paper:           cross-cut shred (DIN 66399 level per classification)

# SSD caution: traditional overwrite is UNRELIABLE for SSDs
# due to wear leveling, over-provisioning, and bad block management
# Always use cryptographic erase or physical destruction for SSDs
```

## Data Loss Prevention (DLP)

```
# DLP strategies by location:

# Network DLP (data in transit)
# - Inspect email, web, FTP, cloud uploads
# - Pattern matching: SSN, credit card numbers, classification markers
# - Block or quarantine policy-violating transfers
# - Deploy at network perimeter and internal boundaries

# Endpoint DLP (data at rest and in use)
# - Monitor file operations: copy, move, print, USB, screenshot
# - Control removable media (USB block or encrypt)
# - Clipboard monitoring and control
# - Application-level restrictions

# Cloud DLP (data in cloud services)
# - CASB (Cloud Access Security Broker) integration
# - Scan cloud storage for classified data
# - Enforce sharing policies in SaaS applications
# - API-based scanning of cloud repositories

# DLP detection methods:
# Exact data matching    — hash comparison of known sensitive data
# Pattern matching       — regex for structured data (SSN, CC#)
# Statistical analysis   — fingerprinting of unstructured documents
# Classification tags    — metadata-based policy enforcement
# Machine learning       — trained models for sensitive content detection
```

## Encryption for Data Protection

```
# Encryption at rest:
# Full Disk Encryption (FDE):  LUKS, BitLocker, FileVault
# Volume encryption:           VeraCrypt, dm-crypt
# File-level encryption:       GPG, age, EFS
# Database encryption:         TDE, column-level, application-level
# Cloud storage:               SSE-S3, SSE-KMS, BYOK, HYOK

# Encryption in transit:
# TLS 1.3:     web, API, email (STARTTLS)
# IPsec:       network-level (tunnel/transport mode)
# SSH:          remote access, file transfer
# WireGuard:   modern VPN
# Signal Protocol: end-to-end messaging

# Key management:
# Generate:    CSPRNG, sufficient key length
# Store:       HSM, KMS, secrets manager (never in code/config)
# Rotate:      Periodic rotation per policy (annually minimum)
# Distribute:  Secure channel, key wrapping, envelope encryption
# Revoke:      CRL, OCSP for certificates
# Destroy:     Cryptographic erasure, zeroize HSM

# Envelope encryption (for large data):
# 1. Generate Data Encryption Key (DEK)
# 2. Encrypt data with DEK (symmetric, fast)
# 3. Encrypt DEK with Key Encryption Key (KEK)
# 4. Store encrypted DEK alongside encrypted data
# 5. KEK stays in KMS/HSM (never leaves secure boundary)
```

## Privacy Considerations

```
# Privacy principles (OECD / GDPR aligned):
# 1. Lawfulness — legal basis for processing
# 2. Purpose limitation — collect only for specified purposes
# 3. Data minimization — collect only what is necessary
# 4. Accuracy — keep data accurate and up to date
# 5. Storage limitation — retain only as long as needed
# 6. Integrity and confidentiality — protect from unauthorized access
# 7. Accountability — demonstrate compliance

# PII (Personally Identifiable Information):
# Direct identifiers: name, SSN, email, biometrics, passport
# Quasi-identifiers: ZIP code, birth date, gender (combinable)
# Sensitive PII: health, financial, racial, biometric, children's data

# Privacy-enhancing technologies:
# Anonymization — irreversibly remove identifying information
# Pseudonymization — replace identifiers with tokens (reversible)
# Data masking — obscure portions of data (e.g., ***-**-1234)
# Tokenization — replace sensitive data with non-sensitive tokens
# Differential privacy — add noise to query results
# K-anonymity — ensure each record matches ≥ k other records
```

## Tips

- Classification must be assigned at creation, not retroactively — unclassified data already at risk.
- The data owner is always a business role, never IT; IT serves as custodian implementing the owner's decisions.
- SSD sanitization requires cryptographic erase or physical destruction — overwrite methods are unreliable for flash media.
- DLP is defense in depth — deploy at network, endpoint, and cloud layers for comprehensive coverage.
- Data retention should default to the shortest legally permissible period to minimize exposure.
- Privacy by design means building privacy controls into systems from the start, not bolting them on later.

## See Also

- security-models, security-governance, cryptography, pki, risk-management, identity-management

## References

- [NIST SP 800-88 Rev 1 — Guidelines for Media Sanitization](https://csrc.nist.gov/publications/detail/sp/800-88/rev-1/final)
- [NIST SP 800-60 Rev 1 — Guide for Mapping Types of Information to Security Categories](https://csrc.nist.gov/publications/detail/sp/800-60/vol-1-rev-1/final)
- [NIST SP 800-122 — Guide to Protecting the Confidentiality of PII](https://csrc.nist.gov/publications/detail/sp/800-122/final)
- [ISO/IEC 27001:2022 — Information Security Management](https://www.iso.org/standard/27001)
- [GDPR — General Data Protection Regulation (EU 2016/679)](https://gdpr-info.eu/)
- [FIPS 199 — Standards for Security Categorization](https://csrc.nist.gov/publications/detail/fips/199/final)
- [DIN 66399 — Destruction of Data Media](https://www.din.de/en/getting-involved/standards-committees/nis/publications/wdc-beuth:din21:147757tried)
