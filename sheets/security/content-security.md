# Content Security (DLP, Sandboxing, CDR, URL Filtering)

Protecting data at rest, in motion, and in use through content inspection, data loss prevention, sandboxing, and content disarm across email, web, and cloud channels.

## Content Security Architecture

### Traffic Flow Through Content Security Stack

```
Inbound/Outbound Traffic
         |
         v
+------------------+
| URL Filtering    |  Block known-bad categories
+------------------+
         |
         v
+------------------+
| Anti-Malware     |  Signature + heuristic scan
+------------------+
         |
         v
+------------------+
| Sandbox Analysis |  Detonate unknown files
+------------------+
         |
         v
+------------------+
| DLP Engine       |  Inspect for sensitive data
+------------------+
         |
         v
+------------------+
| CDR (Content     |  Strip active content,
|  Disarm & Recon) |  rebuild clean files
+------------------+
         |
         v
[Delivered / Blocked / Quarantined]
```

### Content Security Channels

| Channel | Technology | Inspection Point |
|---------|-----------|-----------------|
| Email inbound | Cisco Secure Email (ESA), Microsoft Defender | MX relay, cloud gateway |
| Email outbound | ESA DLP, M365 DLP | Outbound MTA |
| Web traffic | Cisco Secure Web Appliance (WSA), Umbrella | Proxy, DNS |
| Cloud SaaS | CASB (Netskope, Microsoft Defender for Cloud Apps) | API, inline proxy |
| Endpoint | Cisco Secure Endpoint, CrowdStrike | Host agent |
| Network | Cisco Secure Firewall (FTD), Stealthwatch | Inline, TAP/SPAN |

## Data Loss Prevention (DLP)

### DLP Policy Components

```
DLP Policy
  |
  +-- Condition (WHAT to detect)
  |     +-- Content type (PII, PCI, PHI, IP)
  |     +-- Detection method (regex, fingerprint, EDM, ML)
  |     +-- Confidence level (low, medium, high)
  |
  +-- Context (WHERE / WHO / WHEN)
  |     +-- Source / destination user or group
  |     +-- Channel (email, web, cloud, endpoint)
  |     +-- Direction (inbound, outbound, internal)
  |
  +-- Action (WHAT to do)
        +-- Allow, block, quarantine, encrypt
        +-- Notify user, notify admin
        +-- Log, audit, incident creation
```

### DLP Detection Methods

| Method | Description | Accuracy | Performance |
|--------|-------------|----------|-------------|
| Regex pattern | Regular expression matching | Medium | Fast |
| Keyword / dictionary | Word or phrase lists | Low-Medium | Fast |
| Exact Data Match (EDM) | Hash of actual data records | Very High | Medium |
| Document fingerprint | Structural hash of document templates | High | Medium |
| ML / statistical | Machine learning classifiers | High | Slow |
| OCR + pattern | Image text extraction + regex | Medium | Very Slow |

### Common Regex Patterns for Sensitive Data

#### Credit Card Numbers (PCI DSS)

```regex
# Visa
\b4[0-9]{12}(?:[0-9]{3})?\b

# Mastercard
\b5[1-5][0-9]{14}\b
\b2(?:2[2-9][1-9]|2[3-9][0-9]|[3-6][0-9]{2}|7[01][0-9]|720)[0-9]{12}\b

# American Express
\b3[47][0-9]{13}\b

# Generic (Luhn-validatable)
\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13}|6(?:011|5[0-9]{2})[0-9]{12})\b
```

#### Social Security Numbers (PII)

```regex
# SSN (with dashes)
\b(?!000|666|9\d{2})\d{3}-(?!00)\d{2}-(?!0000)\d{4}\b

# SSN (no dashes)
\b(?!000|666|9\d{2})\d{3}(?!00)\d{2}(?!0000)\d{4}\b

# SSN (with spaces)
\b(?!000|666|9\d{2})\d{3}\s(?!00)\d{2}\s(?!0000)\d{4}\b
```

#### Protected Health Information (PHI / HIPAA)

```regex
# Medical Record Number (common format)
\bMRN[:\s#]*[0-9]{6,10}\b

# ICD-10 codes
\b[A-TV-Z][0-9][0-9AB]\.?[0-9A-TV-Z]{0,4}\b

# DEA Number
\b[ABFM][A-Z][0-9]{7}\b

# NPI (National Provider Identifier)
\b[12][0-9]{9}\b
```

#### Email Addresses and Phone Numbers

```regex
# Email
\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b

# US Phone (various formats)
\b(?:\+?1[-.\s]?)?\(?[2-9]\d{2}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b

# International Phone
\+[1-9]\d{1,14}\b
```

### DLP Policy Severity Levels

| Severity | Example | Typical Action |
|----------|---------|---------------|
| Critical | 100+ SSNs in one file, source code repo | Block + encrypt + notify CISO |
| High | Single credit card + CVV | Block + quarantine + incident |
| Medium | Single SSN, employee ID | Warn user + log + notify manager |
| Low | Email address, phone number | Log only, user coaching |
| Informational | Keyword match without context | Log, no action |

## Content Inspection Techniques

### File Type Detection

```
# Magic byte signatures (first bytes of file)
PDF:    %PDF (25 50 44 46)
DOCX:   PK.. (50 4B 03 04) — ZIP container
XLSX:   PK.. (50 4B 03 04) — ZIP container
EXE:    MZ   (4D 5A)
ELF:    .ELF (7F 45 4C 46)
JPEG:   ....  (FF D8 FF)
PNG:    .PNG (89 50 4E 47)
GIF:    GIF8 (47 49 46 38)
GZIP:   ..   (1F 8B)
RAR:    Rar! (52 61 72 21)
7Z:     7z.. (37 7A BC AF)
```

### Content Inspection Pipeline

```
File Received
     |
     v
[Magic byte check] --> True type != declared extension? --> Flag
     |
     v
[Archive extraction] --> Recursive unpack (ZIP, RAR, 7z, tar.gz)
     |                    Nested depth limit (e.g., 10 levels)
     |                    Total size limit (e.g., 500 MB)
     v
[Text extraction] --> Office docs (OOXML), PDF (text layer + OCR)
     |                 HTML (tag stripping), email body + attachments
     v
[Pattern matching] --> Regex engine (PCRE/RE2)
     |                  Dictionary lookup
     |                  Exact data match (hash table)
     v
[ML classification] --> Trained models (NLP for unstructured data)
     |
     v
[Policy evaluation] --> Match conditions --> Apply actions
```

## Content Disarm and Reconstruct (CDR)

### CDR Process

```
Original File (e.g., .docx)
     |
     v
[Parse file structure] --> Identify embedded objects
     |
     v
[Remove active content]
     +-- Macros (VBA)
     +-- Embedded OLE objects
     +-- JavaScript (in PDF)
     +-- ActiveX controls
     +-- External links / references
     +-- Embedded executables
     +-- DDE fields
     v
[Reconstruct clean file]
     |
     v
Clean File (same format, safe content)
```

### CDR vs Traditional AV

| Feature | CDR | Antivirus |
|---------|-----|-----------|
| Zero-day protection | Yes (removes all active content) | No (signature-based) |
| False positives | Low (structural, not heuristic) | Medium-High |
| File fidelity | May lose macros/formatting | Original preserved |
| Performance | Medium | Fast |
| Evasion resistant | High (no signature needed) | Low (obfuscation bypasses) |

## Sandboxing (Cisco Threat Grid / Secure Malware Analytics)

### Sandbox Analysis Workflow

```
Suspicious File / URL
        |
        v
+------------------+
| Static Analysis  |  PE headers, strings, imports, entropy
+------------------+
        |
        v
+------------------+
| Dynamic Analysis |  Execute in instrumented VM
+------------------+  Monitor: file I/O, registry, network,
        |              process creation, API calls
        v
+------------------+
| Behavioral Score |  Threat score 0-100
+------------------+  Behavioral indicators mapped to MITRE ATT&CK
        |
        v
[Verdict: Clean / Suspicious / Malicious]
```

### Cisco Threat Grid API

```bash
# Submit a file for analysis
curl -X POST "https://panacea.threatgrid.com/api/v2/samples" \
  -H "Authorization: Bearer $API_KEY" \
  -F "sample=@malware.exe" \
  -F "vm=win10" \
  -F "playbook=default"

# Check submission status
curl "https://panacea.threatgrid.com/api/v2/samples/$SAMPLE_ID" \
  -H "Authorization: Bearer $API_KEY"

# Get threat score
curl "https://panacea.threatgrid.com/api/v2/samples/$SAMPLE_ID/threat" \
  -H "Authorization: Bearer $API_KEY"

# Get behavioral indicators
curl "https://panacea.threatgrid.com/api/v2/samples/$SAMPLE_ID/analysis/behaviors" \
  -H "Authorization: Bearer $API_KEY"

# Search for IOCs
curl "https://panacea.threatgrid.com/api/v2/search/submissions?q=domain:evil.com" \
  -H "Authorization: Bearer $API_KEY"
```

### Sandbox Evasion Indicators to Watch

| Technique | Detection | Countermeasure |
|-----------|-----------|---------------|
| Sleep timer (delay execution) | Patches Sleep API, accelerates clock | Bare-metal sandbox |
| VM detection (VMware tools, CPUID) | Checks for VM artifacts | Hardened VM images |
| User interaction required | Waits for mouse click | Automated interaction scripts |
| Environment checks | Checks hostname, username, MAC | Realistic environment data |
| Anti-debug | IsDebuggerPresent, timing checks | Transparent instrumentation |

## URL Filtering

### URL Filtering Categories

| Category Group | Examples |
|---------------|----------|
| Security threat | Malware, phishing, botnet, cryptomining |
| Adult content | Pornography, nudity, mature content |
| Bandwidth-heavy | Streaming media, P2P, file sharing |
| Productivity | Social networking, gaming, shopping |
| Legal liability | Gambling, weapons, drugs, hate speech |
| Business | News, finance, education, government |
| Uncategorized | Unknown, newly registered domains |

### Cisco Umbrella DNS-Layer Security

```bash
# Test if a domain is blocked
dig @208.67.222.222 malware-domain.com

# Umbrella Investigate API — domain risk score
curl -H "Authorization: Bearer $UMBRELLA_TOKEN" \
  "https://investigate.api.umbrella.com/domains/risk-score/example.com"

# Get domain categorization
curl -H "Authorization: Bearer $UMBRELLA_TOKEN" \
  "https://investigate.api.umbrella.com/domains/categorization/example.com"

# Check domain WHOIS
curl -H "Authorization: Bearer $UMBRELLA_TOKEN" \
  "https://investigate.api.umbrella.com/whois/example.com"
```

### URL Filtering Decision Flow

```
URL Request
    |
    v
[Local cache lookup] --> Hit? --> Apply cached policy
    |
    | Miss
    v
[Cloud lookup (Talos/Umbrella)] --> Categorized? --> Apply category policy
    |
    | Unknown
    v
[Dynamic analysis]
    +-- Newly registered? --> Higher risk score
    +-- Domain age < 30 days? --> Suspicious
    +-- DGA pattern? --> Block
    |
    v
[Default policy] --> Allow / Block / Warn
```

## Cloud Content Security

### Microsoft 365 DLP

```
# PowerShell: List DLP policies
Get-DlpCompliancePolicy

# Create a DLP policy
New-DlpCompliancePolicy -Name "PCI Protection" \
  -ExchangeLocation All \
  -SharePointLocation All \
  -OneDriveLocation All \
  -Mode Enable

# Create a DLP rule (credit card detection)
New-DlpComplianceRule -Name "Block CC External" \
  -Policy "PCI Protection" \
  -ContentContainsSensitiveInformation @{
    Name = "Credit Card Number";
    minCount = 1
  } \
  -BlockAccess $true \
  -NotifyUser "SiteAdmin"

# View DLP incidents
Get-DlpDetailReport -StartDate (Get-Date).AddDays(-7) -EndDate (Get-Date)

# Export DLP matches
Export-DlpPolicyMatchReport -PolicyName "PCI Protection" -OutputPath ./dlp_report.csv
```

### Google Workspace DLP

```
# Google Workspace DLP is configured via Admin Console
# API-based inspection via Cloud DLP API

# gcloud: Inspect text for sensitive data
gcloud dlp text inspect \
  --content="My SSN is 123-45-6789" \
  --info-types="US_SOCIAL_SECURITY_NUMBER" \
  --min-likelihood=LIKELY

# gcloud: Inspect a file
gcloud dlp content inspect \
  --file=report.csv \
  --info-types="CREDIT_CARD_NUMBER,US_SOCIAL_SECURITY_NUMBER,EMAIL_ADDRESS" \
  --min-likelihood=POSSIBLE

# gcloud: De-identify (redact) content
gcloud dlp text deidentify \
  --content="Call me at 555-123-4567" \
  --info-types="PHONE_NUMBER" \
  --deidentify-config='{"infoTypeTransformations":{"transformations":[{"primitiveTransformation":{"replaceConfig":{"newValue":{"stringValue":"[REDACTED]"}}}}]}}'
```

### CASB Integration for SaaS DLP

```
CASB Deployment Models:

API-based (Out-of-Band)          Inline (Forward/Reverse Proxy)
+--------+     +--------+       +--------+     +--------+
|  SaaS  |<--->|  CASB  |       |  User  |---->|  CASB  |---->| SaaS |
|  App   | API |  Cloud |       |        | TLS |  Proxy |     |  App |
+--------+     +--------+       +--------+     +--------+     +------+
                                     |
Pros:                            Pros:
- No user impact                 - Real-time block
- Retroactive scan               - Inline DLP
- No cert deployment             - URL filtering
                                 - Threat prevention
Cons:                            Cons:
- Not real-time                  - TLS inspection (cert deploy)
- Cannot block (only alert)      - Latency impact
- API rate limits                - Single point of failure
```

## Content Classification

### Data Classification Levels

| Level | Label | Examples | Controls |
|-------|-------|----------|----------|
| 1 | Public | Marketing material, public docs | None required |
| 2 | Internal | Internal memos, org charts | Access control |
| 3 | Confidential | Financial reports, contracts | Encryption + DLP |
| 4 | Restricted | PII, PHI, PCI data, trade secrets | Encryption + DLP + audit + access review |
| 5 | Top Secret | M&A plans, security architecture | All above + need-to-know + air gap |

### Automated Classification Tools

| Tool | Method | Integration |
|------|--------|-------------|
| Microsoft Information Protection (MIP) | ML + regex + fingerprint | M365, Azure, endpoints |
| Google Cloud DLP | InfoType detectors + ML | Workspace, GCP |
| Cisco Umbrella DLP | Regex + dictionary | SIG, cloud proxy |
| Trellix (McAfee) DLP | Fingerprint + ML + regex | Email, web, endpoint |
| Symantec DLP | EDM + ML + fingerprint | Email, web, endpoint, cloud |

## DLP Incident Management

### Incident Workflow

```
Detection --> Triage --> Investigation --> Remediation --> Closure
    |            |            |                |              |
    v            v            v                v              v
Auto-alert   Assign       Review          Block/encrypt   Root cause
to SOC       analyst      content +       Revoke access   Policy update
             Set SLA      context         Coach user      Training
```

### Incident Severity Matrix

| Data Type | Volume | Direction | Severity |
|-----------|--------|-----------|----------|
| PCI (credit cards) | Any | External | Critical |
| PHI (health records) | > 500 records | External | Critical |
| PII (SSN, passport) | > 10 records | External | High |
| PII (SSN, passport) | 1-10 records | External | Medium |
| Source code / IP | Any | External | High |
| Internal documents | Bulk | External | Medium |
| Any sensitive | Any | Internal unauthorized | Low-Medium |

## Tips

- Start DLP in monitor-only mode for 2-4 weeks before enforcing to baseline false positive rates.
- Combine multiple detection methods (regex + EDM + ML) for highest accuracy and lowest false positives.
- Use CDR for high-risk file types (Office docs, PDFs) entering the organization — it stops zero-day document exploits without needing signatures.
- Tune sandbox analysis timeouts; sophisticated malware may sleep for 5+ minutes before activating.
- Deploy CASB in API mode first for visibility, then add inline proxy for enforcement once policies are proven.
- Always validate regex patterns against known test data before deploying to production DLP policies.
- Set recursive archive extraction depth limits (10 levels) and total size limits (500 MB) to prevent zip bomb attacks.
- Classify data at creation time whenever possible; retroactive classification is expensive and error-prone.
- Integrate DLP incident data into SIEM for correlation with other security events.
- Use Luhn algorithm validation after regex matching for credit card numbers to dramatically reduce false positives.

## See Also

- cisco-ftd, waf, siem, tls, pki, zero-trust, cisco-ise, cryptography

## References

- [Cisco Secure Email Gateway Administration Guide](https://www.cisco.com/c/en/us/td/docs/security/esa/esa15-0/user_guide/b_ESA_Admin_Guide_15-0.html)
- [Cisco Umbrella Documentation](https://docs.umbrella.com/)
- [Cisco Threat Grid (Secure Malware Analytics) API](https://www.cisco.com/c/en/us/td/docs/security/threat_grid/v2-api/b_threat-grid-api.html)
- [Microsoft Purview DLP Documentation](https://learn.microsoft.com/en-us/purview/dlp-learn-about-dlp)
- [Google Cloud DLP Documentation](https://cloud.google.com/sensitive-data-protection/docs)
- [NIST SP 800-53 SC-7 — Boundary Protection](https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final)
- [PCI DSS v4.0 Requirement 3 — Protect Stored Account Data](https://www.pcisecuritystandards.org/)
- [HIPAA Security Rule — 45 CFR 164.312](https://www.hhs.gov/hipaa/for-professionals/security/index.html)
