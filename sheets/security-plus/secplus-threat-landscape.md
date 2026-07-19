# Threat Actors, Vectors & Social Engineering (Security+ SY0-701 Objectives 2.1 + 2.2)

> Who attacks (nation-state to unskilled), why they attack (money to ideology), and every road in (email, USB, supply chain, and the human being at the keyboard) — with the disambiguation tables the exam builds scenario questions from.

## Threat Actors (2.1)

```
# The exam gives you a scenario and asks "which threat actor is MOST likely
# responsible?" Match on funding + sophistication + motivation keywords.

# Nation-state
#   Government-sponsored. Effectively unlimited funding, highest
#   sophistication. Runs APTs (Advanced Persistent Threats) — long-dwell,
#   stealthy, multi-stage campaigns. Motivations: espionage, war,
#   data exfiltration, service disruption of critical infrastructure.
#   Exam cue: "well-funded", "sophisticated", "government", "APT",
#   "attacked a power grid / defense contractor".

# Unskilled attacker (formerly "script kiddie")
#   Low skill, low funding. Uses pre-built tools, public exploit scripts,
#   downloaded kits — cannot write their own. Motivation: chaos, bragging
#   rights, curiosity. Often noisy and easily detected.
#   Exam cue: "used a freely available tool", "no advanced knowledge".

# Hacktivist
#   Ideologically motivated — philosophical/political beliefs. Targets
#   organizations they oppose. Typical actions: website defacement,
#   DDoS (Distributed Denial-of-Service), doxxing, data leaks to press.
#   Moderate skill, low-to-moderate funding.
#   Exam cue: "defaced website with a political message", "activist group".

# Insider threat
#   Employee, contractor, or partner with legitimate access. Can be
#   malicious (revenge after a poor review, financial gain, sabotage
#   before quitting) or unintentional (well-meaning but careless).
#   Dangerous because they start INSIDE the perimeter with valid creds.
#   Exam cue: "recently terminated employee", "user with legitimate
#   access", "disgruntled". Mitigations: least privilege, offboarding,
#   user behavior analytics, separation of duties.

# Organized crime
#   Professional criminal enterprise. Well-funded, skilled, hierarchical
#   (developers, mules, launderers). Motivation is almost always
#   FINANCIAL GAIN: ransomware, banking trojans, card theft, BEC
#   (Business Email Compromise), extortion, blackmail.
#   Exam cue: "ransomware demanding payment", "for-profit", "syndicate".

# Shadow IT
#   Not an outside attacker — internal users/departments deploying
#   unsanctioned hardware, software, or cloud services (personal Dropbox,
#   unapproved SaaS, rogue Wi-Fi AP). Creates unmanaged attack surface.
#   Exam cue: "department signed up for a cloud app without IT approval".
```

## Actor Attribute & Motivation Matrix

| Actor | Internal/External | Funding | Sophistication | Primary motivations |
|---|---|---|---|---|
| Nation-state | External | Massive | Very high (APT) | Espionage, war, disruption, exfiltration |
| Organized crime | External | High | High | Financial gain, blackmail, extortion |
| Hacktivist | External | Low–moderate | Moderate | Philosophical/political beliefs, disruption |
| Insider threat | Internal | Varies (has access instead) | Varies | Revenge, financial gain, ethics (whistleblower), accident |
| Unskilled attacker | External | Minimal | Low (borrowed tools) | Chaos, reputation, curiosity |
| Shadow IT | Internal | Departmental budget | N/A (not hostile) | Convenience, working around IT |

```
# Motivation vocabulary the exam uses verbatim (2.1):
# Data exfiltration     — stealing data out of the org
# Espionage             — spying for state or competitor advantage
# Service disruption    — taking systems/services down
# Blackmail             — "pay or we leak/encrypt/expose"
# Financial gain        — direct profit (fraud, ransom, theft, resale)
# Philosophical/political beliefs — hacktivism
# Ethical               — white-hat/grey-hat; whistleblowing insiders
# Revenge               — fired/disgruntled employee striking back
# Disruption/chaos      — no profit motive, just damage or notoriety
# War                   — nation-state attacks accompanying armed conflict
```

## Threat Vectors & Attack Surfaces (2.2)

```
# Vector = the path the attacker uses to reach you.
# Attack surface = the sum of all points where an attacker can try.
# Reduce attack surface: close ports, remove software, disable services,
# decommission properly, train humans.

# Message-based
#   Email — #1 vector: phishing, malicious attachments, malicious links.
#   SMS (Short Message Service) — smishing, malicious links to fake
#     login pages; also OTP-relay scams.
#   IM (Instant Messaging) — Slack/Teams/Discord phishing, malicious
#     file shares; trusted-channel assumption makes clicks more likely.

# Image-based
#   Malicious payloads embedded in image files (steganography, malformed
#   parsers, SVG with embedded script). Exam cue: "SVG attachment ran
#   script", "payload hidden inside an image".

# File-based
#   Malicious documents (macro-enabled Office files), PDFs, archives,
#   executables. Exam cue: "opened an invoice.docm and enabled macros".

# Voice call
#   Vishing (voice phishing) — caller impersonates help desk, bank, IRS;
#   also deepfake-audio CEO fraud. Exam cue: "caller claimed to be from
#   the service desk and asked for the user's password".

# Removable device
#   USB drop attacks — infected flash drives seeded in a parking lot;
#   malicious cables/chargers ("juice jacking"). Mitigation: disable
#   autorun, block USB mass storage via policy, user training.

# Vulnerable software
#   Client-based — an agent/application installed on the endpoint must be
#     patched on every host (bigger management burden, local exploit).
#   Agentless   — no installed client (web app, scanner without agent);
#     nothing to patch on the endpoint, but visibility is shallower.
#   Exam wants the client-based vs agentless distinction.

# Unsupported systems and applications
#   EOL (end-of-life) OS or app — no more security patches, ever.
#   Mitigations: isolate/segment, compensating controls, replace.

# Unsecure networks
#   Wireless  — open/WEP/WPS Wi-Fi, evil twin APs, deauth attacks.
#   Wired     — unauthenticated wall jacks (mitigate with 802.1X port
#               security), tapping, rogue devices.
#   Bluetooth — bluejacking (unsolicited messages), bluesnarfing (data
#               theft over BT), outdated pairing modes.

# Open service ports
#   Every listening service = attack surface. Enumerable with a port
#   scan. Mitigation: close/disable, firewall, service hardening.

# Default credentials
#   admin/admin on routers, cameras, IoT, printers. First thing botnets
#   try. Mitigation: change on deployment — a hardening baseline item.

# Supply chain
#   MSP (Managed Service Provider) — compromise one MSP, reach all its
#     customers (Kaseya-style). MSPs hold privileged remote access.
#   Vendor   — software vendor's update mechanism poisoned (SolarWinds-
#     style malicious update).
#   Supplier — tampered hardware/components upstream of you.
#   Exam cue: "compromise originated from a trusted third party's
#   software update" → supply chain / malicious update.
```

## Human Vectors / Social Engineering (2.2)

The exam's favorite question format: a one-line scenario, "which attack is this?" Match the keyword.

| Technique | Definition | Scenario keyword that gives it away |
|---|---|---|
| Phishing | Fraudulent email harvesting creds/delivering malware | Mass email, fake login link |
| Spear phishing | Phishing targeted at a specific person/group | "Tailored to the finance team" |
| Whaling | Spear phishing aimed at executives | "Sent to the CFO/CEO" |
| Vishing | Phishing by voice call | Phone call, caller ID spoofed |
| Smishing | Phishing by SMS text | Text message with a link |
| BEC (Business Email Compromise) | Attacker uses/spoofs a real business account to redirect payments | "Vendor emailed new wire instructions" |
| Pretexting | Invented scenario/identity to justify the ask | "Claimed to be an auditor needing records" |
| Impersonation | Pretending to be a trusted person (help desk, police, courier) | "Posed as IT support" |
| Brand impersonation | Faking a company's look/domain/branding at scale | "Email styled exactly like Microsoft" |
| Typosquatting | Registering look-alike misspelled domains | "gooogle.com", "micros0ft.com" |
| Watering hole | Compromise a site the target group already visits | "Industry forum served malware" |
| Misinformation/disinformation | False info spread accidentally (mis-) or deliberately (dis-) | "Coordinated false narrative campaign" |

```
# Disambiguation traps the exam sets:
# Phishing vs BEC — BEC involves a REAL (compromised) or convincingly
#   spoofed business mailbox and usually targets MONEY MOVEMENT (wire
#   fraud, payroll diversion), not credential harvest.
# Pretexting vs impersonation — pretexting is the fabricated STORY
#   ("I'm calling because your invoice failed"); impersonation is the
#   fabricated IDENTITY. They usually travel together; answer what the
#   question emphasizes.
# Typosquatting vs brand impersonation — typosquatting is the DOMAIN
#   trick; brand impersonation is the LOOK/branding trick.
# Watering hole vs supply chain — watering hole poisons a site the
#   victims VISIT; supply chain poisons something the victims INSTALL.
# Misinformation vs disinformation — DISinformation is Deliberate.

# Why social engineering works — the pressure levers:
# authority (CEO/police), urgency ("account closes in 1 hour"),
# scarcity, intimidation, consensus ("everyone else complied"),
# familiarity/liking, trust. Naming the lever is sometimes the answer.
```

## Attack Surface Reduction Checklist

```
# Per-vector canonical mitigation (memorize the pairing):
# Email phishing        → secure email gateway + DMARC/DKIM/SPF + training
# Removable media       → USB device control policy, disable autorun
# Open ports            → close/disable services, firewall, scans to verify
# Default credentials   → change at deployment (baseline enforcement)
# Unsupported systems   → replace; else isolate + compensating controls
# Wireless              → WPA3, 802.1X, site survey for rogue APs
# Wired                 → 802.1X port security, disable unused ports
# Bluetooth             → non-discoverable mode, patch, disable if unused
# MSP/vendor/supplier   → vendor assessment, right-to-audit, monitoring
#                         (see secplus-third-party-compliance)
# Humans                → awareness program + simulated phishing
#                         (see secplus-security-awareness)
```

## Exam Cues (2.1 + 2.2)

```
# Keyword → answer:
# "well-funded, sophisticated, long-term persistence"   → nation-state/APT
# "ransomware, profit"                                  → organized crime
# "political message defacement"                        → hacktivist
# "recently fired admin"                                → insider threat
# "used downloaded tools, low skill"                    → unskilled attacker
# "unapproved SaaS used by marketing"                   → shadow IT
# "text with a link"                                    → smishing
# "phone call asking for password"                      → vishing
# "email to CFO about urgent wire"                      → whaling / BEC
# "fake domain one letter off"                          → typosquatting
# "malware on industry news site"                       → watering hole
# "compromise via software update"                      → supply chain
# "found USB in parking lot"                            → removable device
# "admin/admin still worked"                            → default credentials
# "deliberately Deceptive narrative"                    → disinformation
```

## See Also

security-plus-sy0-701, secplus-security-concepts, secplus-vulnerabilities, secplus-attack-indicators, secplus-hardening, secplus-security-awareness, secplus-third-party-compliance, threat-modeling, threat-hunting, mitre-attack, supply-chain-security, email-gateway, security-awareness, network-access-control

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [MITRE ATT&CK — Groups (threat actor profiles)](https://attack.mitre.org/groups/)
- [CISA — Nation-State Cyber Actors](https://www.cisa.gov/topics/cyber-threats-and-advisories/nation-state-cyber-actors)
- [NIST SP 800-30 Rev. 1 — Guide for Conducting Risk Assessments (threat sources)](https://csrc.nist.gov/pubs/sp/800/30/r1/final)
- [Verizon Data Breach Investigations Report](https://www.verizon.com/business/resources/reports/dbir/)
- [FBI IC3 — Business Email Compromise](https://www.ic3.gov/Home/BEC)
- [CISA — Insider Threat Mitigation](https://www.cisa.gov/topics/physical-security/insider-threat-mitigation)
