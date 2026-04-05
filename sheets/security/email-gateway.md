# Email Security Gateway (Cisco ESA / Secure Email)

Dedicated appliance for inbound and outbound email security: anti-spam, anti-malware, DLP, encryption, DKIM/DMARC/SPF verification, content filtering, and message tracking. Processes mail through a multi-stage pipeline from reception to delivery.

## Architecture

### Mail Pipeline Overview

```
Internet          ESA                                    Internal Mail
(Sender) -------> Listener -----> Work Queue ---------> (Exchange/O365)
                    |                 |
                    v                 v
              HAT / RAT         Processing:
              (connection        1. CASE Anti-Spam
               filtering)       2. Anti-Virus (Sophos/McAfee)
                                3. AMP (Advanced Malware Protection)
                                4. Graymail Detection
                                5. Content Filters
                                6. DLP
                                7. Outbreak Filters
                                8. Message Filters (pre-policy)
```

### Deployment Models

| Model | Description |
|-------|-------------|
| MX gateway | ESA is MX record, sits in front of mail server |
| Transparent | Inline, no MX change (rare) |
| Cloud (CES) | Cisco-hosted Secure Email Cloud Gateway |
| Hybrid | On-prem ESA + Cloud Email Security |
| Dual-appliance | Separate inbound/outbound ESAs |

### Listeners

```
! Incoming listener — receives mail from internet
! Public listener on interface Data1
!   - Accepts mail for domains listed in RAT
!   - Applies HAT policies based on sender reputation

! Outgoing listener — receives mail from internal servers
! Private listener on interface Data2
!   - Accepts mail from internal relay hosts
!   - Applies outbound policies (DLP, encryption)

! CLI: listenerconfig
esa> listenerconfig
Choose the operation:
- NEW     — create a new listener
- EDIT    — edit a listener
- DELETE  — delete a listener
- SETUP   — configure listener defaults
```

## Connection Filtering (HAT / RAT)

### Host Access Table (HAT)

The HAT determines how the ESA handles connections from sending hosts:

| Policy | Action | Use Case |
|--------|--------|----------|
| ACCEPT | Accept connection, apply mail policy | Default for unknown senders |
| REJECT | Reject with 5xx SMTP response | Known bad senders |
| RELAY | Accept and relay (no RAT check) | Internal hosts, partner relays |
| TCPREFUSE | Drop TCP connection silently | Block without SMTP response |
| CONTINUE | Continue to next rule | Chain HAT rules |

### HAT Configuration

```
! Add sender group to HAT
esa> listenerconfig > edit > hostaccess

! HAT order (top-down, first match):
! 1. ALLOWLIST — trusted senders (ACCEPT + skip anti-spam)
! 2. BLOCKLIST — known bad senders (REJECT)
! 3. SUSPECTLIST — suspicious senders (throttle)
! 4. UNKNOWNLIST — default (ACCEPT with full scanning)

! Sender group with SenderBase reputation
! SenderBase Reputation Score (SBRS): -10.0 to +10.0
!   -10 to -3.0 → BLOCKLIST
!   -3.0 to -1.0 → SUSPECTLIST (throttle)
!   -1.0 to +3.0 → UNKNOWNLIST (full scan)
!   +3.0 to +10.0 → ALLOWLIST (reduced scan)
```

### Recipient Access Table (RAT)

```
! Define which domains the ESA accepts mail for
esa> listenerconfig > edit > rcptaccess

! RAT entries:
!   example.com     — ACCEPT
!   test.example.com — ACCEPT
!   ALL             — REJECT (default, reject mail for unknown domains)

! LDAP-based recipient verification (reject invalid recipients at SMTP)
esa> ldapconfig
! Configure LDAP acceptance query to verify recipients exist in directory
```

## Anti-Spam

### CASE (Context Adaptive Scanning Engine)

```
! CASE combines multiple techniques:
! 1. SenderBase/Talos reputation scoring
! 2. Content analysis (heuristic rules)
! 3. URL reputation and categorization
! 4. Message structure analysis
! 5. Adaptive rules (auto-updated from Talos)

! Anti-spam verdict:
! - Positive → spam (quarantine or drop)
! - Suspect → possible spam (tag or quarantine)
! - Negative → clean (deliver)

! Configure anti-spam on mail policy
esa> policyconfig > antispam

! Positive spam action: quarantine (default) | drop | deliver + tag
! Suspect spam action: deliver + tag [SUSPECTED SPAM] (default)
! Spam threshold: adjustable per policy (default: moderate)
```

### SenderBase / Talos Reputation

| Score Range | Classification | Typical Action |
|-------------|---------------|----------------|
| +7.0 to +10.0 | Trusted | Bypass anti-spam |
| +3.0 to +7.0 | Good | Reduced scanning |
| -1.0 to +3.0 | Neutral | Full scanning |
| -3.0 to -1.0 | Poor | Throttle + full scan |
| -10.0 to -3.0 | Bad | Block at connection |

```
! Check reputation for a sender
esa> senderbaseconfig

! Enable SenderBase lookups
esa> senderbaseconfig > setup
! Enable IP Reputation Service: yes
! Enable IP Reputation lookups for incoming: yes
```

## Anti-Malware

### Engine Configuration

```
! Sophos Anti-Virus
esa> antivirusconfig
! Enable Sophos: yes
! Action on virus: drop / clean / quarantine
! Action on unscannable: quarantine

! McAfee Anti-Virus (if licensed)
esa> antivirusconfig
! Enable McAfee: yes
! Both engines can run simultaneously (multi-layer scanning)

! AMP (Advanced Malware Protection)
esa> ampconfig
! Enable File Reputation: yes (cloud lookup for file hash)
! Enable File Analysis: yes (sandbox detonation for unknown files)
! Action on malicious: drop
! Action on unscannable: deliver with warning

! File reputation check flow:
! 1. Calculate SHA-256 of attachment
! 2. Query AMP cloud for reputation verdict
! 3. Verdict: Clean / Malicious / Unknown
! 4. If Unknown + File Analysis enabled → send to sandbox
! 5. Retrospective alert if verdict changes later
```

### Outbreak Filters

```
! Adaptive rules for zero-day threats
esa> outbreakconfig
! Enable Outbreak Filters: yes
! Threat level threshold: 3 (1-5, lower = more aggressive)
! Maximum quarantine period: 4 hours
! URL rewriting: enabled (rewrites URLs to Cisco Security Proxy)

! Outbreak Filters quarantine messages matching threat patterns
! Recheck quarantined messages as new rules arrive
! Auto-release when threat level drops
```

## Content Filters

### Content Filter Configuration

```
! Content filters apply per-mail-policy (incoming or outgoing)
esa> policyconfig > filters

! Example: Block executable attachments
! Condition: Attachment file type = exe, bat, cmd, scr, vbs
! Action: Drop message, notify admin

! Example: Encrypt messages with SSN
! Condition: Body contains pattern \d{3}-\d{2}-\d{4}
! Action: Encrypt (using Cisco Registered Envelope)

! Example: Add disclaimer to outbound
! Condition: All messages
! Action: Add footer text

! Example: Strip large attachments
! Condition: Attachment size > 25MB
! Action: Strip attachment, notify sender

! Filter conditions:
! - Subject, body, header, envelope (regex or smart identifier)
! - Attachment: type, name, size, content
! - Message size
! - Sender/recipient (group membership, LDAP)
! - Reputation score
! - True file type (beyond extension)

! Filter actions:
! - Drop, bounce, quarantine, deliver
! - Tag subject, add header, strip header
! - Encrypt, redirect, BCC copy
! - Notify sender/recipient/admin
! - Strip/replace attachment
```

### Message Filters vs Content Filters

| Feature | Message Filters | Content Filters |
|---------|----------------|-----------------|
| Configuration | CLI only (filterconfig) | GUI and CLI |
| Processing order | Before mail policies | After mail policies |
| Scope | All messages (global) | Per-policy |
| Syntax | Proprietary scripting language | Condition/Action GUI |
| Power | Full programmatic control | Simpler, policy-based |
| Best for | Complex routing, header manipulation | Content-based actions |

### Message Filter Example

```
! CLI message filter syntax
esa> filters

! Drop messages with spoofed From header
spoof_filter:
  if (header("From") == "@example\\.com$" AND
      recv-listener == "IncomingMail")
  {
    drop();
    log-entry("Spoofed internal sender blocked");
  }

! Redirect messages to compliance
compliance_redirect:
  if (header("Subject") == "(?i)confidential")
  {
    bcc("compliance@example.com");
  }
```

## DLP (Data Loss Prevention)

```
! DLP uses predefined and custom policies
esa> dlpconfig

! Predefined DLP policies:
! - PCI-DSS (credit card numbers)
! - HIPAA (health information)
! - GLBA (financial data)
! - PII (SSN, passport, driver license)
! - SOX (financial compliance)

! DLP policy actions:
! - Deliver (log only)
! - Quarantine for review
! - Encrypt and deliver
! - Drop
! - Add disclaimer/header

! Custom DLP policy:
! Define classifiers (regex, smart identifiers, dictionaries)
! Set severity scale: Low (1-3), Medium (4-6), High (7-9), Critical (10)
! Assign actions per severity level

! Apply DLP to outgoing mail policies only
esa> policyconfig > outgoing > edit > DLP
```

## Email Authentication (DKIM / DMARC / SPF)

### SPF Verification

```
! Enable SPF verification on incoming mail policy
esa> spfconfig
! SPF verification: enabled
! Action on hard fail (-all): quarantine or tag subject
! Action on soft fail (~all): tag header
! Action on none/neutral: no action

! SPF checks the envelope sender (MAIL FROM) against DNS TXT record
! Example DNS: example.com IN TXT "v=spf1 ip4:203.0.113.0/24 include:_spf.google.com -all"
```

### DKIM Verification and Signing

```
! DKIM verification (incoming)
esa> dkimconfig > verification
! Enable verification: yes
! Action on failure: tag header (Authentication-Results)

! DKIM signing (outgoing)
esa> dkimconfig > signing
! Create signing profile:
!   Selector: sel1
!   Domain: example.com
!   Key size: 2048-bit RSA
!   Headers to sign: From, To, Subject, Date, MIME-Version
!   Canonicalization: relaxed/relaxed
!   Body length: entire body

! Export public key for DNS TXT record:
! sel1._domainkey.example.com IN TXT "v=DKIM1; k=rsa; p=<base64-public-key>"
```

### DMARC Verification

```
! Enable DMARC verification
esa> dmarcconfig
! DMARC verification: enabled
! Action on reject policy: quarantine (override sender's reject to quarantine)
! Action on quarantine policy: quarantine
! Aggregate report sending: enabled (send to rua= address)
! Forensic report sending: disabled (privacy concern)

! DMARC checks:
! 1. SPF alignment (envelope sender domain matches From header domain)
! 2. DKIM alignment (DKIM d= domain matches From header domain)
! 3. At least one must pass AND align
```

## Encryption

### Encryption Methods

| Method | Description | Use Case |
|--------|-------------|----------|
| TLS (opportunistic) | Encrypt SMTP transport if peer supports | Default for all connections |
| TLS (required) | Mandatory TLS, reject if unavailable | Partner domains |
| S/MIME | Per-message encryption + signing | End-to-end, certificate-based |
| PGP | Per-message encryption (OpenPGP) | End-to-end, key-based |
| Cisco Registered Envelope (CRES) | Envelope encryption via cloud portal | Policy-based outbound encryption |

### TLS Configuration

```
! TLS for outgoing (per-destination)
esa> destconfig
! Domain: partner.com
! TLS: required (verify)
! Certificate verification: yes

! TLS for incoming (on listener)
esa> listenerconfig > edit > TLS
! TLS: preferred (accept TLS if offered, fall back to plain)
! or: required (reject non-TLS connections)

! View TLS connections
esa> tlsverify partner.com

! Check TLS status
esa> tls
```

### Cisco Registered Envelope Service (CRES)

```
! Enable envelope encryption
esa> encryptionconfig
! Encryption profile: CRES_Profile
! Key server: res.cisco.com
! Subject: [SECURE] prepended
! Envelope format: HTML attachment
!
! Recipient opens encrypted email:
! 1. Receives HTML envelope attachment
! 2. Opens in browser → redirected to Cisco Registered Envelope portal
! 3. Registers / logs in
! 4. Reads decrypted message in browser

! Trigger encryption via content filter:
! Condition: Subject contains [encrypt] OR DLP match
! Action: Encrypt with profile CRES_Profile
```

## Mail Policies

### Incoming Mail Policy

```
! Default incoming policy applies to all recipients
! Custom policies match on recipient (envelope or header)

esa> policyconfig > incoming

! Policy structure:
! 1. Anti-Spam: CASE settings, thresholds, actions
! 2. Anti-Virus: Sophos/McAfee settings
! 3. AMP: File reputation + file analysis
! 4. Graymail: Marketing, social, bulk detection
! 5. Content Filters: ordered list of filters
! 6. Outbreak Filters: threat level, quarantine time
! 7. Advanced: message modification, DKIM/SPF/DMARC

! Match by recipient:
! - Specific address: user@example.com
! - Domain: example.com
! - LDAP group: members of "Executives" group
```

### Outgoing Mail Policy

```
! Applied to messages from internal senders going outbound

esa> policyconfig > outgoing

! Policy structure:
! 1. Anti-Spam: minimal (internal senders)
! 2. Anti-Virus: scan outbound for compromised endpoints
! 3. DLP: primary enforcement point
! 4. Content Filters: disclaimers, attachment control
! 5. Encryption: trigger-based or policy-based
! 6. DKIM signing: sign outbound with domain key
```

## LDAP Integration

```
! LDAP server configuration
esa> ldapconfig

! LDAP queries:
! 1. Acceptance query — verify recipient exists (reject at SMTP)
! 2. Routing query — determine delivery destination
! 3. Masquerade query — rewrite envelope/header addresses
! 4. Group query — check group membership for policy matching
! 5. SMTP authentication query — authenticate outbound senders

! Example Active Directory configuration:
! Server: ldaps://dc1.corp.example.com:636
! Base DN: dc=corp,dc=example,dc=com
! Bind DN: cn=esa-service,ou=ServiceAccounts,dc=corp,dc=example,dc=com
! Acceptance query: (|(mail={a})(proxyAddresses=smtp:{a}))
! Group query: (&(objectClass=group)(member={dn}))

! Test LDAP query
esa> ldaptest
! Select query → enter test address → verify results
```

## Quarantine Management

```
! Centralized quarantines on ESA (or SMA for multi-appliance)

! View quarantine status
esa> quarantineconfig

! Quarantine types:
! 1. Spam Quarantine — end-user accessible via web portal
! 2. Policy Quarantine — admin-managed (DLP, content filter)
! 3. Virus Quarantine — messages with detected malware
! 4. Outbreak Quarantine — messages held by Outbreak Filters
! 5. Unclassified Quarantine — fallback

! Spam quarantine settings:
! - Retention: 14 days (default)
! - End-User Quarantine access: enabled
! - Digest notifications: daily at 08:00
! - Release: user can release from digest or portal

! CLI quarantine commands
esa> quarantineconfig
esa> quarantine              ! manage quarantined messages
esa> spamdigest              ! configure digest notifications
```

## Message Tracking

```
! Search for messages by sender, recipient, subject, message ID

! GUI: Monitor > Message Tracking
! Search criteria: sender, recipient, subject, date range, message ID
! Results show: injection, processing steps, delivery status

! CLI message tracking
esa> findevent
! Search by: envelope sender, envelope recipient, message ID
! Time range: last 1h, 8h, 24h, 3d, custom

! Track message through pipeline:
! MID (Message ID) — unique per message
! ICID (Injection Connection ID) — incoming SMTP session
! DCID (Delivery Connection ID) — outgoing SMTP session

! Log entries follow: MID → ICID → processing → DCID
! Example log flow:
! Info: New SMTP ICID 12345 from 203.0.113.10
! Info: MID 67890 — From: sender@ext.com, To: user@example.com
! Info: MID 67890 — anti-spam: negative
! Info: MID 67890 — antivirus: clean
! Info: MID 67890 — AMP: file reputation clean
! Info: MID 67890 — Content filter: no match
! Info: MID 67890 — delivered to 10.0.0.5 (DCID 54321)
```

## CLI and GUI Management

### Common CLI Commands

```
! System status
esa> status         ! brief system status
esa> status detail  ! detailed queue and resource stats

! Mail queue
esa> workqueue      ! show work queue status
esa> deleterecipients ! remove messages from queue
esa> showrecipients   ! show messages in queue by recipient

! Configuration
esa> commit          ! commit pending changes (REQUIRED after any change)
esa> clear           ! discard uncommitted changes
esa> showconfig      ! display current configuration
esa> saveconfig      ! save configuration to file
esa> loadconfig      ! load configuration from file

! Logging
esa> tail mail_logs   ! tail the mail log in real time
esa> grep             ! search logs

! Network
esa> diagnostic > network  ! ping, traceroute, nslookup
esa> smtproutes           ! SMTP routing table
esa> dnsconfig            ! DNS server configuration

! Cluster management (centralized management of multiple ESAs)
esa> clusterconfig
! Modes: machine-level, group-level, cluster-level settings
```

### Key GUI Sections

| Section | Path | Purpose |
|---------|------|---------|
| Message Tracking | Monitor > Message Tracking | Find and trace messages |
| Mail Flow | Monitor > Mail Flow Summary | Volume, threats, TLS stats |
| System Status | Monitor > System Status | CPU, RAM, queue, connections |
| Incoming Policy | Mail Policies > Incoming | Anti-spam/AV/AMP per policy |
| Outgoing Policy | Mail Policies > Outgoing | DLP/encryption per policy |
| Content Filters | Mail Policies > Incoming/Outgoing Content Filters | Filter rules |
| HAT | Network > Listener > HAT | Sender group policies |
| Quarantine | Monitor > Quarantine | Manage quarantined messages |
| DKIM/SPF/DMARC | Mail Policies > Mail Flow Policies | Authentication settings |

## Clustering (Centralized Management)

```
! ESA cluster — manage multiple appliances from one interface
esa> clusterconfig

! Cluster hierarchy:
! Cluster level — settings shared by all machines
!   └── Group level — settings shared by a group of machines
!       └── Machine level — settings specific to one machine

! Join cluster
esa> clusterconfig > join

! Configuration inheritance:
! Machine overrides group overrides cluster
! HAT, RAT, SMTP routes, policies can be at any level
```

## Tips

- Always commit changes after CLI modifications; uncommitted changes are lost on timeout.
- Use HAT sender groups with SenderBase reputation to block spam at the connection level before it consumes scanning resources.
- Deploy DKIM signing on outbound and DMARC verification on inbound as a pair for complete email authentication.
- Enable TLS preferred on all listeners and TLS required for known partners to maximize encryption coverage.
- Use content filters for simple policy-based actions; use message filters only for complex logic that cannot be expressed in the GUI.
- Configure LDAP acceptance queries to reject invalid recipients at the SMTP conversation (saves processing and prevents backscatter).
- AMP retrospective verdicts can change days later; ensure the quarantine retention period is long enough to capture delayed verdicts.
- DLP applies only to outgoing mail policies; do not expect DLP on inbound.
- Monitor the work queue regularly; a growing queue indicates delivery issues or processing bottlenecks.
- Use Cisco SMA (Security Management Appliance) for centralized reporting and quarantine across multiple ESAs.

## See Also

- tls, pki, dns, cryptography, cisco-ise

## References

- [Cisco Secure Email Administrator Guide](https://www.cisco.com/c/en/us/td/docs/security/esa/esa15-0/user_guide/b_ESA_Admin_Guide_15-0.html)
- [Cisco Secure Email CLI Reference](https://www.cisco.com/c/en/us/td/docs/security/esa/esa15-0/cli_reference/b_CLI_Reference_Guide_15-0.html)
- [Cisco Talos Intelligence](https://talosintelligence.com/)
- [RFC 7208 — SPF (Sender Policy Framework)](https://www.rfc-editor.org/rfc/rfc7208)
- [RFC 6376 — DKIM (DomainKeys Identified Mail)](https://www.rfc-editor.org/rfc/rfc6376)
- [RFC 7489 — DMARC (Domain-based Message Authentication)](https://www.rfc-editor.org/rfc/rfc7489)
- [RFC 5321 — SMTP](https://www.rfc-editor.org/rfc/rfc5321)
- [RFC 8461 — MTA-STS (SMTP MTA Strict Transport Security)](https://www.rfc-editor.org/rfc/rfc8461)
