# Security Assessment

> Systematic evaluation of security posture through vulnerability scanning, penetration testing, and compliance validation to identify weaknesses and verify defenses.

## Assessment Types Comparison

```
Type                  Goal                    Scope        Exploitation?
────                  ────                    ─────        ─────────────
Vulnerability Scan    Find known vulns        Broad        No
Vulnerability         Scan + analyze +        Broad        No
  Assessment          prioritize
Penetration Test      Prove exploitability    Targeted     Yes
Red Team              Simulate real attacker  Full org     Yes (stealth)
Purple Team           Collaborative           Full org     Yes (cooperative)
                      attack/defend
Bug Bounty            Crowdsource findings    Defined      Yes (rules)
```

## Vulnerability Assessment

### Vulnerability Management Lifecycle

```
┌──────────┐   ┌───────────┐   ┌───────────┐   ┌──────────┐
│ Discover  │──>│ Prioritize│──>│ Remediate  │──>│  Verify  │
│           │   │           │   │            │   │          │
│ Scan      │   │ CVSS +    │   │ Patch      │   │ Rescan   │
│ Enumerate │   │ Context   │   │ Mitigate   │   │ Validate │
│ Inventory │   │ EPSS      │   │ Accept     │   │ Close    │
└──────────┘   └───────────┘   └───────────┘   └──────────┘
      ▲                                              │
      └──────────────── Continuous ──────────────────┘
```

### Vulnerability Scanners

```
Scanner         Type            License       Best For
───────         ────            ───────       ────────
Nessus          Network/host    Commercial    Enterprise scanning
OpenVAS/GVM     Network/host    Open source   Budget-conscious
Qualys          Cloud-based     Commercial    Large enterprise
Rapid7 InsightVM Network/host   Commercial    Integration-heavy
Tenable.io      Cloud-based     Commercial    Cloud + hybrid

# Nessus (command line via API)
# Create scan
curl -X POST https://nessus:8834/scans \
  -H "X-ApiKeys: accessKey=...; secretKey=..." \
  -d '{"uuid":"template-uuid","settings":{
    "name":"Weekly Scan",
    "targets":"10.0.0.0/24",
    "enabled":true
  }}'

# Launch scan
curl -X POST https://nessus:8834/scans/{scan_id}/launch

# Export results
curl -X POST https://nessus:8834/scans/{scan_id}/export \
  -d '{"format":"csv"}'

# OpenVAS (GVM - Greenbone Vulnerability Management)
# Create target
gvm-cli socket --xml '<create_target>
  <name>Internal Network</name>
  <hosts>10.0.0.0/24</hosts>
</create_target>'

# Create and start task
gvm-cli socket --xml '<create_task>
  <name>Weekly Internal Scan</name>
  <target id="target-uuid"/>
  <config id="full-and-fast-uuid"/>
</create_task>'
```

### Scan Types

```
Type              Approach              Accuracy    Impact
────              ────────              ────────    ──────
Unauthenticated   External network      Lower       Minimal
                  perspective           (surface    (no creds)
                                        only)
Authenticated     Logs into target,     Higher      Low
(credentialed)    checks installed      (sees       (read-only
                  software, configs     inside)     access)
Agent-based       Software on host      Highest     Lowest
                  reports continuously  (real-time) (local check)

# Credentialed scan advantages
# - Detects missing patches accurately
# - Reads local configuration files
# - Enumerates installed software
# - Checks file permissions
# - 10x fewer false positives vs unauthenticated
```

## CVSS Scoring

### CVSS v3.1 Base Score Components

```
Metric Group     Metric                    Values
────────────     ──────                    ──────
Attack Vector    Network/Adjacent/         N=0.85 A=0.62
(AV)             Local/Physical            L=0.55 P=0.20

Attack           Low/High                  L=0.77 H=0.44
Complexity (AC)

Privileges       None/Low/High             N=0.85 L=0.62/0.68
Required (PR)                              H=0.27/0.50
                                           (Changed scope)

User             None/Required             N=0.85 R=0.62
Interaction (UI)

Scope (S)        Unchanged/Changed         Affects PR and
                                           impact calculation

Confidentiality  None/Low/High             N=0 L=0.22 H=0.56
Impact (C)

Integrity        None/Low/High             N=0 L=0.22 H=0.56
Impact (I)

Availability     None/Low/High             N=0 L=0.22 H=0.56
Impact (A)
```

### CVSS Severity Ratings

```
Score Range    Severity     Action Timeline
───────────    ────────     ───────────────
0.0            None         Informational
0.1 - 3.9     Low          Fix within 180 days
4.0 - 6.9     Medium       Fix within 90 days
7.0 - 8.9     High         Fix within 30 days
9.0 - 10.0    Critical     Fix within 14 days (or immediately)
```

### Example CVSS Vectors

```
# Remote code execution, no auth required
CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H  → 10.0 (Critical)

# Local privilege escalation, requires low priv
CVSS:3.1/AV:L/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:H  → 7.8 (High)

# XSS requiring user interaction
CVSS:3.1/AV:N/AC:L/PR:N/UI:R/S:C/C:L/I:L/A:N   → 6.1 (Medium)

# Information disclosure, adjacent network
CVSS:3.1/AV:A/AC:H/PR:N/UI:N/S:U/C:L/I:N/A:N   → 2.6 (Low)
```

### CVSS v4.0 Changes

```
# Key differences from v3.1
# - Attack Requirements (AT) replaces Attack Complexity
# - Supplemental metrics: Safety, Automatable, Recovery, etc.
# - Threat metric group replaces Temporal
# - Provider Urgency added
# - Modified base metrics for environmental context
# - Improved naming convention

# v4.0 vector format
CVSS:4.0/AV:N/AC:L/AT:N/PR:N/UI:N/VC:H/VI:H/VA:H/SC:N/SI:N/SA:N

# New impact split: Vulnerable system (VC/VI/VA) vs
#                   Subsequent system (SC/SI/SA)
```

## Penetration Testing

### Engagement Types

```
Type          Knowledge              Simulates
────          ─────────              ─────────
Black Box     No prior knowledge     External attacker
              (target IP/URL only)   (realistic)

White Box     Full knowledge         Insider threat
              (source code, arch,    (thorough)
              credentials)

Gray Box      Partial knowledge      Compromised user /
              (user creds, some      partner with some
              architecture docs)     access
```

### Penetration Testing Methodology (PTES)

```
Phase                Activities
─────                ──────────
1. Pre-engagement    Scope, rules of engagement, NDA, emergency contacts,
                     legal authorization, communication plan

2. Intelligence      OSINT, DNS recon, WHOIS, social engineering recon,
   Gathering         technology fingerprinting, employee enumeration

3. Threat Modeling   Identify high-value targets, attack vectors,
                     threat actors, business logic risks

4. Vulnerability     Automated scanning, manual testing, service
   Analysis          enumeration, configuration review

5. Exploitation      Attempt exploitation of confirmed vulns,
                     proof of concept, credential attacks,
                     web app attacks, social engineering

6. Post-              Privilege escalation, lateral movement,
   Exploitation      data exfiltration, persistence, pivoting,
                     accessing high-value targets

7. Reporting         Executive summary, technical findings,
                     risk ratings, remediation guidance,
                     evidence/screenshots, retesting offer
```

### Common Tools by Phase

```
Phase              Tools
─────              ─────
Recon              nmap, Shodan, theHarvester, Maltego, Recon-ng,
                   Amass, subfinder, httpx

Scanning           Nessus, OpenVAS, Nikto, Nuclei, testssl.sh,
                   masscan, rustscan

Web App            Burp Suite, OWASP ZAP, SQLMap, ffuf, gobuster,
                   wfuzz, Postman, httpie

Exploitation       Metasploit, Cobalt Strike, Sliver, exploit-db,
                   searchsploit, BeEF

Post-Exploit       Mimikatz, BloodHound, PowerView, Rubeus,
                   Impacket, CrackMapExec, Covenant

Password           Hashcat, John the Ripper, Hydra, CeWL,
                   Responder, ntlmrelayx

Pivoting           Chisel, ligolo-ng, sshuttle, proxychains,
                   socat, SSH tunnels
```

### Rules of Engagement

```
# Must be documented and signed BEFORE testing begins

Scope Definition:
  In scope:    IP ranges, domains, applications, services
  Out of scope: production databases, third-party services,
                specific hosts, social engineering (unless agreed)

Authorization:
  - Written authorization from asset owner
  - Legal review and sign-off
  - Time window for testing
  - Emergency stop procedure
  - Point of contact (24/7 during test)

Constraints:
  - No denial of service (unless specifically authorized)
  - No data destruction
  - No modification of production data
  - Sensitive data handling procedures
  - Notification before high-risk activities

Evidence Handling:
  - Encrypt all findings and evidence
  - Secure deletion after engagement
  - Data retention period
  - PII handling procedures
```

## Compliance Scanning

### CIS Benchmarks

```
# Center for Internet Security hardening guidelines

# Categories
CIS Benchmark          Target
─────────────          ──────
CIS Ubuntu 22.04       Linux server hardening
CIS RHEL 9             Linux server hardening
CIS Windows Server 2022 Windows hardening
CIS Docker             Container runtime
CIS Kubernetes         K8s cluster
CIS AWS Foundations    Cloud configuration
CIS Azure Foundations  Cloud configuration
CIS GCP Foundations    Cloud configuration

# Scanning tools
# CIS-CAT Pro (official CIS tool)
./CIS-CAT.sh -b benchmarks/CIS_Ubuntu_22.04_v1.0.0.xml \
  -r /path/to/report

# OpenSCAP (open source)
oscap xccdf eval \
  --profile xccdf_org.ssgproject.content_profile_cis \
  --results results.xml \
  --report report.html \
  /usr/share/xml/scap/ssg/content/ssg-ubuntu2204-ds.xml

# InSpec (Chef/Progress)
inspec exec https://github.com/dev-sec/linux-baseline \
  -t ssh://user@target --reporter html:report.html

# Scoring
# Level 1: Essential, minimal performance impact
# Level 2: Defense in depth, may reduce functionality
# Scored:   Pass/fail contributes to benchmark score
# Unscored: Recommendation only, no score impact
```

## Red Team vs Pen Test vs Vuln Assessment

```
Aspect         Vuln Assessment  Pen Test         Red Team
──────         ───────────────  ────────         ────────
Objective      Find vulns       Prove exploit    Test detection
                                                 & response
Duration       Hours-days       Days-weeks       Weeks-months
Scope          Broad            Targeted         Full org
Stealth        None             Low              High (evade
                                                 defenders)
Social eng.    No               Sometimes        Yes
Physical       No               Sometimes        Yes
Rules          Scan only        Limited exploit   Full adversary
                                                 emulation
Defenders      Aware            Aware            Unaware
                                                 (except mgmt)
Reporting      Vuln list +      Exploit chains   Narrative of
               remediation      + evidence       attack campaign
Cost           $               $$               $$$
```

## Bug Bounty Programs

```
# Structured vulnerability disclosure with rewards

Program Types:
  Private:  Invite-only researchers
  Public:   Open to all researchers
  VDP:      Vulnerability Disclosure Policy (no rewards, safe harbor)

Platforms:
  HackerOne, Bugcrowd, Intigriti, Synack (managed)

# Typical reward ranges
Severity     Bounty Range
────────     ────────────
Critical     $5,000 - $100,000+
High         $2,000 - $20,000
Medium       $500 - $5,000
Low          $100 - $1,000
Info         $0 - $100

# Program elements
- Scope definition (domains, apps, APIs)
- Out of scope exclusions
- Safe harbor legal protection for researchers
- Response SLAs (initial response, triage, fix)
- Reward table by severity
- Responsible disclosure requirements
- Duplicate handling policy
```

## Assessment Reporting

```
# Report Structure

1. Executive Summary (1-2 pages)
   - Overall risk rating
   - Key findings count by severity
   - Business impact summary
   - Top 3 recommendations

2. Methodology
   - Testing approach and tools
   - Scope and limitations
   - Testing timeline

3. Findings (per finding)
   - Title and severity (CVSS)
   - Description
   - Affected systems/endpoints
   - Evidence (screenshots, request/response)
   - Business impact
   - Remediation recommendation
   - References (CVE, CWE, OWASP)

4. Risk Summary Matrix
   ┌────────────┬─────────────────────────────┐
   │ Severity   │ Count │ Remediated │ Open  │
   ├────────────┼───────┼────────────┼───────┤
   │ Critical   │   2   │     1      │   1   │
   │ High       │   5   │     3      │   2   │
   │ Medium     │  12   │     8      │   4   │
   │ Low        │  23   │    15      │   8   │
   │ Info       │   8   │     —      │   —   │
   └────────────┴───────┴────────────┴───────┘

5. Appendices
   - Full scan results
   - Tool configurations
   - Remediation verification results
```

## See Also

- vulnerability-scanning
- sast-dast
- cve
- cis-benchmarks
- incident-response
- mitre-attack

## References

- NIST SP 800-115: Technical Guide to Information Security Testing and Assessment
- PTES: Penetration Testing Execution Standard (http://www.pentest-standard.org)
- OWASP Testing Guide v4.2
- CVSS v3.1 Specification: FIRST (https://www.first.org/cvss/)
- CVSS v4.0 Specification: FIRST
- CIS Benchmarks: Center for Internet Security
- PCI DSS: Requirement 11 (Vulnerability Scanning and Penetration Testing)
