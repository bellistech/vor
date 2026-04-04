# Social Engineering (Human-Layer Attack Vectors)

The art and science of manipulating human psychology to bypass security controls, encompassing phishing (spear, whaling, vishing, smishing), pretexting, baiting, tailgating, business email compromise, and watering hole attacks, with defense through awareness training, simulation platforms, and organizational controls.

## Attack Vector Taxonomy
### Classification of Social Engineering Attacks
```
Category 1: Phishing (Electronic Deception)
  Phishing          Mass email campaigns, generic lures
  Spear Phishing    Targeted at specific individuals/roles
  Whaling           Targeting C-suite executives
  Clone Phishing    Duplicate of legitimate email with malicious payload
  Vishing           Voice phishing (phone calls)
  Smishing          SMS/text message phishing
  Angler Phishing   Via social media (fake support accounts)
  QR Phishing       Malicious QR codes (quishing)
  SEO Poisoning     Malicious results in search engines

Category 2: Pretexting (Fabricated Scenarios)
  Impersonation     Posing as IT support, vendor, executive
  Authority         Claiming to be law enforcement, auditor
  Urgency           "Your account will be closed in 24 hours"
  Reciprocity       Offering something to get information
  Help desk attacks Calling help desk as "locked out employee"

Category 3: Physical (In-Person Attacks)
  Tailgating        Following authorized person through door
  Piggybacking      With knowledge/consent of the authorized person
  Dumpster diving   Searching trash for sensitive documents
  Shoulder surfing  Observing screens/keyboards
  USB drops         Leaving malicious USB drives in parking lots

Category 4: Technical-Human Hybrid
  Watering hole     Compromise sites frequented by targets
  BEC               Business Email Compromise (account takeover
                    or domain spoofing for wire fraud)
  Callback phishing Lure victim into calling attacker
  MFA fatigue       Repeated push notifications until approved
  Consent phishing  OAuth app requesting excessive permissions
```

## Phishing Attack Anatomy
### Kill Chain
```
1. Reconnaissance
   - OSINT on target (LinkedIn, social media, company website)
   - Identify organizational structure and key personnel
   - Determine email format (first.last@company.com)
   - Find technology stack (job postings, DNS records)

2. Weaponization
   - Register lookalike domain (typosquatting)
   - Clone legitimate email template
   - Craft payload (credential harvester, malware dropper)
   - Set up phishing infrastructure (redirect, hosting)

3. Delivery
   - Send phishing email / SMS / voice call
   - Bypass email security (SPF/DKIM/DMARC evasion)
   - Time delivery for maximum impact (Monday morning, end of quarter)

4. Exploitation
   - Victim clicks link or opens attachment
   - Credential entry on fake login page
   - Malware execution on victim system
   - OAuth token harvest

5. Action on Objectives
   - Account takeover
   - Lateral movement using stolen credentials
   - Wire transfer fraud (BEC)
   - Data exfiltration
   - Ransomware deployment
```

## Business Email Compromise (BEC)
### Attack Patterns
```
BEC Type 1: CEO Fraud
  Attacker impersonates CEO → emails CFO/Finance
  Request: "Wire $250K to this account for an acquisition"
  Characteristics: Urgency, secrecy ("keep this confidential"),
  authority ("I need this done today")

BEC Type 2: Account Compromise
  Attacker compromises employee email account
  Uses legitimate mailbox to send invoices to contacts
  Changes payment details to attacker-controlled accounts

BEC Type 3: Vendor Email Compromise
  Attacker compromises vendor/supplier email
  Sends fake invoices with modified bank details
  Targets accounts payable departments

BEC Type 4: Attorney Impersonation
  Attacker impersonates legal counsel
  Requests urgent confidential wire transfer
  Typically targets end-of-day / end-of-week

BEC Type 5: Data Theft
  Targets HR/payroll for W-2 forms, employee PII
  Tax season timing (January-April)
  Often impersonates executive requesting "all employee W-2s"

Defense Indicators:
  - Unusual urgency in payment requests
  - Changes to wire transfer instructions
  - Requests to bypass normal approval processes
  - Domain lookalikes (company-inc.com vs company.com)
  - Reply-to address differs from display name
```

## OSINT for Target Profiling
### Reconnaissance Techniques
```bash
# LinkedIn reconnaissance
# - Identify employees, roles, reporting structure
# - Technology stack from job postings
# - Recent hires (less security-aware)
# - Company events and announcements

# Email address enumeration
# theHarvester — gather emails, subdomains, hosts
pip install theHarvester
theHarvester -d target.com -b google,linkedin,dnsdumpster

# Email format verification
# Check if email exists without sending
pip install verify-email
python3 -c "
from verify_email import verify_email
print(verify_email('first.last@target.com'))
"

# Social media OSINT
# Maltego — visual link analysis for relationships
# Recon-ng — web reconnaissance framework
pip install recon-ng
recon-ng
# [recon-ng][default] > marketplace search contacts
# [recon-ng][default] > modules load recon/contacts-contacts/...

# Domain reconnaissance for phishing infrastructure
# Check for typosquatting domains
pip install dnstwist
dnstwist -r target.com         # generate and check domain permutations
dnstwist -r --tld-dict target.com   # check across TLDs

# OSINT Framework — comprehensive resource directory
# https://osintframework.com/
```

## Phishing Simulation Platforms
### GoPhish
```bash
# GoPhish — open source phishing simulation
# Download from https://github.com/gophish/gophish/releases
wget https://github.com/gophish/gophish/releases/download/v0.12.1/gophish-v0.12.1-linux-64bit.zip
unzip gophish-v0.12.1-linux-64bit.zip
cd gophish

# Configure config.json
cat <<'EOF' > config.json
{
  "admin_server": {
    "listen_url": "0.0.0.0:3333",
    "use_tls": true,
    "cert_path": "gophish_admin.crt",
    "key_path": "gophish_admin.key"
  },
  "phish_server": {
    "listen_url": "0.0.0.0:8080",
    "use_tls": false
  },
  "db_name": "sqlite3",
  "db_path": "gophish.db",
  "migrations_prefix": "db/db_"
}
EOF

# Launch GoPhish
./gophish
# Default creds in terminal output — change immediately

# GoPhish workflow:
# 1. Create Sending Profile (SMTP settings)
# 2. Create Email Template (phishing email HTML)
# 3. Create Landing Page (credential capture page)
# 4. Create User Group (import target list CSV)
# 5. Launch Campaign (schedule, track results)
# 6. View Results (opened, clicked, submitted data)

# GoPhish API for automation
curl -k https://localhost:3333/api/campaigns/ \
  -H "Authorization: Bearer API_KEY" | jq .
```

### Commercial Platforms and Metrics
```
Platforms: KnowBe4 (10K+ templates, PAB), Proofpoint (risk scoring),
           Cofense (simulation + IR triage)

Key Metrics:
  Click rate:    Industry baseline 15-25%
  Report rate:   Target >70% of simulated phish reported
  Repeat clicks: Track users failing multiple simulations
  Time to report: Target <5 minutes
```

## Defense Measures
### Technical Controls
```bash
# Email authentication (SPF, DKIM, DMARC)
# SPF record — authorize sending servers
# dig +short TXT target.com
# "v=spf1 include:_spf.google.com -all"

# DKIM — cryptographic email signing
# Configured at mail server; public key in DNS TXT record

# DMARC — policy for SPF/DKIM failures
cat <<'EOF'
DMARC record (_dmarc.target.com TXT):
  v=DMARC1; p=reject; rua=mailto:dmarc@target.com;
  ruf=mailto:dmarc-forensic@target.com; pct=100;

Policies:
  p=none     Monitor only (start here)
  p=quarantine   Send to spam
  p=reject   Block delivery (goal)

Ramp-up: none (2 weeks) → quarantine (4 weeks) → reject
EOF

# Additional technical defenses
cat <<'EOF'
1. Email gateway / secure email gateway (SEG)
   - URL rewriting and time-of-click analysis
   - Attachment sandboxing
   - Impersonation detection (display name spoofing)
   - YARA rules for phishing indicators

2. Multi-factor authentication (MFA)
   - Phishing-resistant: FIDO2/WebAuthn, hardware tokens
   - Moderate: TOTP, push notifications (with number matching)
   - Weak: SMS OTP (SIM swapping risk)

3. Conditional access policies
   - Block sign-ins from unusual locations
   - Require MFA for sensitive operations
   - Device compliance requirements

4. Browser isolation
   - Render web content in isolated container
   - Prevent credential entry on unknown domains
   - Strip active content from email links
EOF
```

### Organizational Controls
```
1. Training: Onboarding + monthly phish sims + quarterly refreshers
2. Reporting: Phish report button, no punishment, reward reporting
3. BEC defense: Dual auth for wires, out-of-band verification,
   verbal confirmation for executive requests
4. Physical: Visitor escorts, badge access, clean desk, shredding
```

## Incident Response for Social Engineering
### Response Playbook
```bash
cat <<'EOF'
Phase 1: Detection and Triage (0-30 minutes)
  - User reports phishing email
  - SOC analyst validates the report
  - Extract IOCs: sender, URLs, attachments, headers
  - Determine scope: how many recipients?
  - Classify severity (credential phish vs malware vs BEC)

Phase 2: Containment (30-60 minutes)
  - Block sender domain/IP at email gateway
  - Block phishing URLs at proxy/DNS
  - Quarantine all copies from all mailboxes
  - If credentials compromised: force password reset + revoke sessions
  - If malware delivered: isolate affected endpoints

Phase 3: Investigation (1-24 hours)
  - Identify all users who received the email
  - Determine who opened/clicked/submitted data
  - Check for lateral movement from compromised accounts
  - Analyze malware payload if applicable
  - Review logs for unauthorized access
  - Check for email forwarding rules added by attacker

Phase 4: Recovery (24-72 hours)
  - Reset compromised credentials
  - Remove persistence mechanisms
  - Re-image affected systems if malware involved
  - Verify no data exfiltration occurred
  - Monitor for follow-up attacks

Phase 5: Lessons Learned (1 week)
  - Document timeline and decisions
  - Update detection rules based on IOCs
  - Adjust training if specific weakness exploited
  - Report metrics to management
  - Update phishing simulation templates
EOF
```

## Tips
- Phishing-resistant MFA (FIDO2/WebAuthn hardware keys) eliminates credential phishing entirely; TOTP and push notifications can still be bypassed by real-time proxy attacks
- Track the phishing report rate, not just the click rate; a high report rate (above 70%) indicates a healthy security culture even if some users click
- Never punish employees for clicking simulated phishing; punishment drives under-reporting and creates a culture of fear rather than vigilance
- Implement DMARC at p=reject for your own domains to prevent attackers from spoofing your organization in attacks against partners and customers
- Use out-of-band verification (phone call to a known number) for any wire transfer changes, payment redirections, or urgent executive requests
- Conduct targeted spear-phishing simulations for high-risk roles (finance, HR, executives) separately from general awareness campaigns
- Deploy a phishing report button directly in the email client so that reporting is as easy as clicking; every barrier reduces reporting rates
- Monitor for typosquatting domains that mimic your organization using tools like dnstwist, and register common variants proactively
- Test your incident response process for social engineering specifically; a credential phish requires different containment than malware delivery
- Review email forwarding rules regularly since attackers commonly add auto-forwarding to maintain access after initial compromise
- Layer defenses: technical controls catch most attacks, awareness training catches what slips through, and process controls prevent high-impact actions like wire fraud

## See Also
- recon, password-attacks, web-attacks, lateral-movement, phishing-defense

## References
- [NIST SP 800-177 Rev. 1 — Email Authentication](https://csrc.nist.gov/publications/detail/sp/800-177/rev-1/final)
- [GoPhish — Open Source Phishing Framework](https://getgophish.com/)
- [MITRE ATT&CK — Initial Access (Phishing)](https://attack.mitre.org/techniques/T1566/)
- [FBI IC3 — Business Email Compromise](https://www.ic3.gov/Media/Y2023/PSA230609)
- [OWASP Social Engineering Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Social_Engineering_Prevention_Cheat_Sheet.html)
- [Anti-Phishing Working Group (APWG)](https://apwg.org/)
- [KnowBe4 Phishing Benchmarks](https://www.knowbe4.com/phishing-benchmarking-report)
- [dnstwist — Domain Permutation Tool](https://github.com/elceef/dnstwist)
