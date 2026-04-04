# PCI DSS (Payment Card Industry Data Security Standard)

The global security standard for all entities that store, process, or transmit cardholder data, comprising 12 requirements organized under 6 control objectives, with PCI DSS v4.0 introducing customized validation approaches and mandatory compliance by March 31, 2025.

## PCI DSS v4.0 — The 12 Requirements
### Organized by 6 Goals
```
Goal 1: Build and Maintain a Secure Network and Systems
  Req 1:  Install and maintain network security controls
          (formerly "firewall configuration")
  Req 2:  Apply secure configurations to all system components
          (eliminate vendor defaults)

Goal 2: Protect Account Data
  Req 3:  Protect stored account data (encryption, truncation,
          masking, hashing, tokenization)
  Req 4:  Protect cardholder data with strong cryptography
          during transmission over open, public networks

Goal 3: Maintain a Vulnerability Management Program
  Req 5:  Protect all systems and networks from malicious
          software (anti-malware, anti-phishing)
  Req 6:  Develop and maintain secure systems and software
          (patch management, secure SDLC)

Goal 4: Implement Strong Access Control Measures
  Req 7:  Restrict access to system components and cardholder
          data by business need to know
  Req 8:  Identify users and authenticate access to system
          components (MFA required for CDE access)
  Req 9:  Restrict physical access to cardholder data

Goal 5: Regularly Monitor and Test Networks
  Req 10: Log and monitor all access to system components
          and cardholder data
  Req 11: Test security of systems and networks regularly
          (ASV scans, penetration tests, IDS/IPS)

Goal 6: Maintain an Information Security Policy
  Req 12: Support information security with organizational
          policies and programs (security awareness, incident
          response, risk assessment)
```

## Scoping
### Cardholder Data Environment (CDE)
```bash
# Scoping categories for PCI DSS assessment
Scope Categories:

1. CDE Systems (always in scope):
   - Systems that store, process, or transmit CHD/SAD
   - Systems directly attached to CDE network segment
   - Payment terminals, POS systems, payment applications

2. Connected-to / Security-Impacting Systems (in scope):
   - Systems that connect to or can access the CDE
   - Systems providing security services to CDE (DNS, NTP, AAA)
   - Systems that can impact CDE security configuration
   - Network devices routing/switching CDE traffic

3. Out-of-Scope Systems:
   - Systems with NO connectivity to CDE
   - Systems on fully isolated network segments
   - Must be validated through network segmentation testing

Cardholder Data Elements:
  PAN (Primary Account Number): Always considered cardholder data
  Cardholder Name:   Protected when stored with PAN
  Service Code:      Protected when stored with PAN
  Expiration Date:   Protected when stored with PAN

Sensitive Authentication Data (SAD) — NEVER store after authorization:
  Full Track Data:   Magnetic stripe / chip equivalent
  CAV2/CVC2/CVV2:   Card verification values
  PIN / PIN Block:   Personal identification numbers
```

### Network Segmentation
```bash
# Network segmentation reduces PCI scope
Segmentation Methods:
  - Firewalls with deny-all default rules
  - VLANs with ACLs (VLANs alone are NOT sufficient)
  - Internal network firewalls / microsegmentation
  - Air-gapped networks for highest security

Segmentation Testing Requirements (Req 11.4.5):
  - Penetration test must validate segmentation every 6 months
    (or after any change to segmentation controls)
  - Test must confirm out-of-scope systems cannot access CDE
  - Must test from all out-of-scope network segments

Segmentation Benefits:
  - Reduces number of systems subject to PCI requirements
  - Reduces assessment scope and cost
  - Limits blast radius of a compromise
  - Simplifies compliance maintenance
```

## Self-Assessment Questionnaires (SAQ)
### SAQ Types
```
SAQ A — Card-not-present merchants (e-commerce)
  - All payment processing fully outsourced
  - No electronic CHD storage, processing, or transmission
  - Only iframe/redirect (no direct form handling)
  - 22 requirements

SAQ A-EP — E-commerce merchants with partial outsourcing
  - Website controls redirect to third-party processor
  - Website could impact transaction security
  - No CHD storage
  - 191 requirements

SAQ B — Imprint or standalone dial-out terminals
  - No electronic CHD storage
  - Standalone terminal, no internet connection
  - 41 requirements

SAQ B-IP — Standalone IP-connected PTS terminals
  - PTS-approved terminals connected via IP
  - No electronic CHD storage
  - 82 requirements

SAQ C — Payment application systems connected to internet
  - Payment application on internet-connected system
  - No electronic CHD storage
  - 160 requirements

SAQ C-VT — Virtual terminal on isolated computer
  - Web-based virtual terminal only
  - No electronic CHD storage
  - 79 requirements

SAQ D — All others
  - Full assessment for merchants (SAQ D-Merchant)
  - Full assessment for service providers (SAQ D-SP)
  - 329 requirements

SAQ P2PE — Hardware payment terminals in a validated P2PE solution
  - Uses PCI-listed P2PE solution
  - No electronic CHD storage
  - 33 requirements
```

## Report on Compliance (ROC) vs SAQ
```
ROC (Report on Compliance):
  - Required for Level 1 merchants (>6M transactions/year)
  - Required for Level 1 service providers
  - Performed by Qualified Security Assessor (QSA)
  - Detailed evidence-based assessment
  - Submitted to acquirer/payment brand

SAQ (Self-Assessment Questionnaire):
  - For Level 2-4 merchants (varies by card brand)
  - Self-reported compliance status
  - May require ISA (Internal Security Assessor) validation
  - Signed by executive and submitted to acquirer

Merchant Levels (Visa):
  Level 1: >6 million transactions/year        → ROC required
  Level 2: 1-6 million transactions/year        → SAQ + quarterly ASV
  Level 3: 20,000-1 million e-commerce/year     → SAQ + quarterly ASV
  Level 4: <20,000 e-commerce or <1M other/year → SAQ + quarterly ASV
```

## PAN Storage and Protection (Requirement 3)
### Acceptable Methods
```bash
# PAN protection methods per PCI DSS v4.0
Rendering PAN unreadable:
  1. One-way hashes (SHA-256 with salt, bcrypt, Argon2)
     - Original PAN cannot be recovered
     - Suitable for comparison operations

  2. Truncation
     - Max display: first 6 and last 4 digits (BIN + last 4)
     - Cannot store truncated AND hashed versions of same PAN

  3. Index tokens / tokenization
     - Replace PAN with surrogate value
     - Token vault must meet all PCI DSS requirements
     - Format-preserving tokens pass Luhn check

  4. Strong cryptography with key management
     - AES-256 for encryption at rest
     - Key management per Requirement 3.6/3.7
     - Key custodians, split knowledge, dual control

Key Management Requirements (Req 3.6, 3.7):
  - Document and implement key management procedures
  - Minimum key strength: AES-128 or equivalent
  - Restrict access to cryptographic keys (fewest custodians)
  - Store keys securely (encrypted KEK, HSM, split components)
  - Rotate keys: crypto period defined, replace when weakened
  - Split knowledge and dual control for manual key operations
  - Prevent unauthorized substitution of keys
  - Retire/replace keys at end of crypto period
```

## Vulnerability Scanning and Pen Testing (Requirement 11)
### ASV Scanning
```bash
# ASV scan requirements
Approved Scanning Vendor (ASV) Scans — Req 11.3.2:
  Frequency:    Quarterly external vulnerability scans
  Scope:        All externally-facing systems in CDE scope
  Pass criteria: No vulnerabilities with CVSS >= 4.0
                 No component identified as a critical vulnerability
  Rescans:      Required until passing scan achieved
  Disputes:     Can dispute false positives with evidence

Internal Vulnerability Scanning — Req 11.3.1:
  Frequency:    Quarterly (at minimum)
  Scope:        All in-scope internal systems
  Resolution:   Rescan after remediation
  High-risk:    Must be resolved per risk-based approach

Penetration Testing — Req 11.4:
  External:     At least annually and after significant changes
  Internal:     At least annually and after significant changes
  Application:  Test all payment applications
  Segmentation: Every 6 months (service providers)
                Every 12 months (merchants)
  Methodology:  Industry-accepted (PTES, OWASP, NIST SP 800-115)
  Findings:     Remediate and retest all exploitable vulnerabilities
```

## Logging and Monitoring (Requirement 10)
### Required Log Events
```bash
Events that MUST be logged (Req 10.2):
  10.2.1  All individual user accesses to cardholder data
  10.2.1.1 All individual access to PAN with business justification
  10.2.1.2 All actions by anyone with admin access
  10.2.1.3 Access to all audit trails
  10.2.1.4 Invalid logical access attempts
  10.2.1.5 Changes to identification/authentication credentials
  10.2.1.6 Initialization/stopping/pausing of audit logs
  10.2.1.7 Creation and deletion of system-level objects
  10.2.2   All security events on all system components

Log entry content requirements (Req 10.3):
  - User identification
  - Type of event
  - Date and time
  - Success or failure indication
  - Origination of event
  - Identity or name of affected data/resource

Log retention:
  - 12 months total retention
  - Last 3 months immediately available for analysis
  - Time synchronization via NTP (Req 10.6)

File Integrity Monitoring (Req 10.5.1 / 11.5.2):
  - Deploy FIM on critical system files, configs, content files
  - Alert on unauthorized modification
  - Perform comparisons at least weekly
  - Tools: OSSEC, Tripwire, AIDE, Samhain
```

## Compensating Controls
```
When an entity cannot meet a requirement due to legitimate
technical or business constraints:

Requirements for a valid compensating control:
  1. Meet the intent and rigor of the original requirement
  2. Provide a similar level of defense
  3. Be above and beyond other PCI DSS requirements
  4. Be commensurate with the additional risk
  5. Document the compensating control worksheet:
     - Original requirement and constraint
     - Objective of the original control
     - Identified risk if control is not implemented
     - Definition of compensating control
     - Validation of compensating control

PCI DSS v4.0 Customized Approach (new):
  - Alternative to compensating controls
  - Define custom control to meet the stated objective
  - Requires independent assessment by QSA
  - Not available for all requirements
  - Derives from the "Customized Approach Objective"
    listed for each requirement in v4.0
```

## PCI DSS v4.0 Key Changes
```
Major changes from v3.2.1 to v4.0:

1. Customized Approach — validate with custom controls
2. Expanded MFA — required for ALL access into CDE
   (not just remote; Req 8.4.2)
3. Targeted risk analysis — entity defines frequency of
   recurring tasks (Req 12.3.1)
4. Enhanced authentication — MFA for all non-console admin
   access (Req 8.4.3 — future-dated to Mar 2025)
5. Automated technical mechanisms — for public-facing web
   apps (WAF or automated technical solution, Req 6.4.2)
6. Encrypted internal network — protect CHD on internal
   networks (Req 4.2.1)
7. Anti-phishing — mechanisms to detect and protect against
   phishing attacks (Req 5.4.1)
8. Payment page script management — monitor scripts on
   payment pages (Req 6.4.3, 11.6.1)
9. Disk-level encryption — no longer acceptable for removable
   media in some configurations (Req 3.5.1.2)
10. Service provider specific — more frequent segmentation
    testing, incident response, change detection
```

## Tips
- Start scoping by mapping every system that touches, stores, or could access cardholder data, then aggressively segment to minimize the CDE
- Use tokenization or point-to-point encryption (P2PE) to remove cardholder data from your environment entirely and dramatically reduce scope
- Never store sensitive authentication data (CVV, full track, PIN) after authorization, even if encrypted; there is no compliant way to retain it
- Schedule quarterly ASV scans well in advance of the deadline to allow time for remediation and rescans if vulnerabilities are found
- Implement file integrity monitoring on all systems in the CDE and ensure alerts are reviewed within 24 hours of detection
- Treat MFA as mandatory for all CDE access under v4.0, not just remote access, and plan implementation before enforcement dates
- Document compensating controls with the formal worksheet format and ensure the QSA reviews them; informal justifications will be rejected
- Synchronize all system clocks via NTP (Requirement 10.6) because log correlation depends on accurate, consistent timestamps
- Maintain audit logs for 12 full months with the most recent 3 months immediately accessible for forensic analysis
- Conduct annual penetration tests using an industry-accepted methodology and retest all exploitable findings before closing them
- Monitor payment page scripts (Requirement 6.4.3 in v4.0) to detect Magecart-style skimming attacks on e-commerce checkout pages

## See Also
- hipaa, gdpr, soc2, iso27001, nist

## References
- [PCI DSS v4.0 Standard (PDF)](https://docs-prv.pcisecuritystandards.org/PCI%20DSS/Standard/PCI-DSS-v4_0.pdf)
- [PCI SSC Document Library](https://www.pcisecuritystandards.org/document_library/)
- [PCI DSS Quick Reference Guide](https://www.pcisecuritystandards.org/documents/PCI_DSS_QRG_v4_0.pdf)
- [PCI SSC Approved Scanning Vendors](https://www.pcisecuritystandards.org/assessors_and_solutions/approved_scanning_vendors)
- [PCI DSS v4.0 Summary of Changes](https://docs-prv.pcisecuritystandards.org/PCI%20DSS/Standard/PCI-DSS-v3-2-1-to-v4-0-Summary-of-Changes-r2.pdf)
- [NIST SP 800-115 — Technical Guide to Security Testing](https://csrc.nist.gov/publications/detail/sp/800-115/final)
