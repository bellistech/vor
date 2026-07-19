# Hardening, Mitigation & Secure Baselines (Security+ SY0-701 Objectives 2.5 + 4.1)

> Mitigation techniques (segmentation, allow lists, patching, least privilege, decommissioning), hardening every target class (workstations to ICS/SCADA to IoT), secure baselines, mobile/wireless security (MDM, BYOD/COPE/CYOD, WPA3/SAE/802.1X), and application security — with practical Linux/Windows commands and exam-cue mappings.

## Objective Map

```
# 2.5 — Given a scenario, apply mitigation techniques to secure the enterprise
#   Segmentation, access control (ACL, permissions), application allow list,
#   isolation, patching, encryption, monitoring, least privilege,
#   configuration enforcement, decommissioning, hardening techniques
# 4.1 — Given a scenario, apply common security techniques to computing resources
#   Secure baselines, hardening targets, wireless devices (site survey,
#   heat map), mobile solutions (MDM, BYOD/COPE/CYOD, connection methods),
#   wireless security settings (WPA3, AAA/RADIUS, crypto/auth protocols),
#   application security, sandboxing, monitoring
# Domain weights: Domain 2 (Threats/Vulns/Mitigations) = 22%,
#                 Domain 4 (Security Operations) = 28% — the biggest domain
```

## Mitigation Techniques (2.5)

```
# Segmentation — split the network so a compromise cannot spread laterally
#   VLANs (Virtual Local Area Networks), subnets, firewalled zones,
#   microsegmentation (per-workload rules, the zero-trust flavor)
#   Exam cue: "prevent lateral movement", "separate the SCADA network",
#   "legacy system that cannot be patched" → put it on its own segment
#
# Access control
#   ACL (Access Control List) — ordered permit/deny rules on routers,
#     firewalls, and file systems; evaluated top-down, first match wins,
#     implicit deny at the end
#   Permissions — file/share/database rights (NTFS, POSIX rwx);
#     misconfigured permissions = the classic insider/audit finding
#
# Application allow list (whitelisting) — ONLY listed executables run;
#   everything else is blocked by default (deny-by-default for software)
#   vs block list (blacklisting) — listed apps blocked, everything else runs
#   Allow list = stronger, higher admin cost; the exam's "most secure" pick
#   Tools: AppLocker / WDAC (Windows Defender Application Control),
#          fapolicyd (Linux)
#
# Isolation — remove a system from general connectivity entirely
#   Air gap (physical isolation), quarantine VLAN, sandbox detonation
#   Exam cue: "infected host — FIRST step per IR" → isolate/contain it
#
# Patching — apply vendor fixes on a defined cadence; test in staging
#   first; emergency/out-of-band patches for actively exploited CVEs
#   (Common Vulnerabilities and Exposures); unpatchable → compensating
#   controls (segmentation + monitoring)
#
# Encryption — render data unreadable without the key
#   At rest: FDE (Full Disk Encryption — BitLocker, LUKS), file/volume/
#   database/record-level. In transit: TLS (Transport Layer Security),
#   IPSec, SSH. Mitigates: stolen laptop/drive, eavesdropping
#
# Monitoring — you cannot mitigate what you cannot see
#   Log collection → SIEM (Security Information and Event Management),
#   alerting, file integrity monitoring (FIM), NetFlow, EDR telemetry
#
# Least privilege — every identity gets the MINIMUM rights needed to do
#   its job, nothing more; enforced via RBAC, JIT (just-in-time) elevation,
#   periodic access reviews/recertification
#   Exam cue: "user has more access than required" → least privilege
#
# Configuration enforcement — automatically detect and revert drift from
#   the approved baseline: Group Policy (GPO), SCCM/Intune, Ansible/
#   Puppet/Chef, posture checks at connect time (NAC health checks)
#
# Decommissioning — formal end-of-life removal of a system:
#   1. Remove from inventory/CMDB and monitoring
#   2. Revoke certificates, credentials, firewall rules, DNS entries
#   3. Sanitize media (wipe per NIST SP 800-88: clear/purge/destroy)
#   4. Certificate of destruction if a third party destroys media
#   Exam cue: "old server still reachable with valid creds" →
#   decommissioning process failure
```

## Hardening Techniques (2.5 continued)

```
# Hardening = reduce the attack surface of a single host
#
# Encryption            — FDE so a stolen/recycled drive leaks nothing
# Endpoint protection   — install EPP/EDR (Endpoint Protection Platform /
#                         Endpoint Detection and Response), anti-malware
# Host-based firewall   — per-host packet filter (Windows Defender
#                         Firewall, ufw/firewalld/nftables); blocks even
#                         same-subnet lateral traffic that a perimeter
#                         firewall never sees
# HIPS                  — Host-based Intrusion Prevention System: watches
#                         host behavior/traffic and BLOCKS inline
#                         (HIDS = detect-only; the P means prevent)
# Disable ports/protocols — close every listening service you don't need;
#                         kill Telnet(23), FTP(21), SMBv1, TLS 1.0/1.1
# Default password changes — change EVERY factory credential before
#                         deployment (routers, cameras, printers, IoT);
#                         Mirai botnet = the canonical consequence
# Removal of unnecessary software — every package is attack surface +
#                         patch burden; uninstall, don't just disable
```

```bash
# Linux hardening quick pass
ss -tulpn                                   # what is actually listening
systemctl list-unit-files --state=enabled   # what starts at boot
systemctl disable --now telnet.socket vsftpd cups   # kill unneeded services
sudo apt purge telnetd rsh-server           # remove, not just stop
sudo ufw default deny incoming && sudo ufw allow 22/tcp && sudo ufw enable
sudo passwd -l root                          # lock direct root login
grep -E '^PermitRootLogin|^PasswordAuthentication' /etc/ssh/sshd_config
#   PermitRootLogin no / PasswordAuthentication no (keys only)
sudo cryptsetup luksFormat /dev/sdb1        # LUKS full-disk/volume encryption
sudo aideinit                                # AIDE file integrity baseline
sudo unattended-upgrades --dry-run          # automatic security patching
```

```powershell
# Windows hardening quick pass
Get-NetTCPConnection -State Listen                    # listening ports
Get-Service | Where-Object {$_.StartType -eq 'Automatic'}
Set-Service -Name Telnet -StartupType Disabled
Disable-WindowsOptionalFeature -Online -FeatureName SMB1Protocol  # kill SMBv1
Set-NetFirewallProfile -Profile Domain,Public,Private -Enabled True
manage-bde -on C: -RecoveryPassword                   # BitLocker FDE
gpupdate /force                                       # re-apply GPO baseline
Get-AppLockerPolicy -Effective                        # allow-list policy
secedit /export /cfg C:\baseline.inf                  # export local sec policy
```

## Secure Baselines (4.1)

```
# Baseline = the documented, approved secure configuration for a device
# class (registry keys, services, policies, package set, firewall rules)
#
# ESTABLISH — start from an authoritative benchmark, then tailor:
#   CIS Benchmarks (Center for Internet Security), DISA STIGs (Security
#   Technical Implementation Guides), vendor guides (Microsoft Security
#   Baselines, NIST SP 800-123 server hardening); document deviations
# DEPLOY   — push via automation, never by hand:
#   GPO/Intune (Windows), Ansible/Puppet/Chef (Linux), golden images /
#   templates (cloud AMIs), IaC (Infrastructure as Code)
# MAINTAIN — continuously verify and correct drift:
#   configuration-compliance scans (SCAP — Security Content Automation
#   Protocol, OpenSCAP, Nessus policy compliance), re-baseline after
#   major patches, change management for any baseline modification
#
# Exam cue: "ensure all new servers are configured identically and
# securely" → deploy from a secure baseline / golden image
# Exam cue: "settings changed after deployment" → configuration
# enforcement + drift detection (maintain phase)
```

```bash
# OpenSCAP baseline scan against a CIS profile
sudo oscap xccdf eval \
  --profile xccdf_org.ssgproject.content_profile_cis \
  --report /tmp/report.html \
  /usr/share/xml/scap/ssg/content/ssg-ubuntu2204-ds.xml
```

## Hardening Targets (4.1) — Target → Specific Steps

| Target | Key hardening steps |
|---|---|
| Mobile devices | MDM enrollment, screen lock + biometrics, full-device encryption, remote wipe, patch OS, disable sideloading, containerize work data |
| Workstations | Baseline image, EDR, host firewall, FDE, application allow list, remove admin rights, automatic patching, disable USB mass storage if policy requires |
| Switches | Change default creds, disable unused ports, port security (MAC limiting), 802.1X, separate management VLAN, disable Telnet/HTTP → SSH/HTTPS only, DHCP snooping, dynamic ARP inspection, BPDU guard |
| Routers | Change default creds, SSHv2 only, ACLs on management plane, disable unused services (CDP on edge, HTTP server, finger), control-plane policing, secure routing protocol auth (OSPF/BGP MD5/SHA), NTP auth, log to central syslog |
| Cloud infrastructure | Least-privilege IAM, MFA on all accounts, no root/owner API keys, encrypt storage buckets + block public access, security groups deny-by-default, enable CloudTrail-style audit logs, CSPM (Cloud Security Posture Management) scanning |
| Servers | Minimal install (no GUI), role-based package set, host firewall, FIM, disable unneeded services/ports, TLS everywhere, dedicated service accounts (non-interactive), centralized logging, timely patching |
| ICS/SCADA | Segment/air-gap from IT network (Purdue model), unidirectional gateways (data diodes), no direct internet, vendor-approved patches in maintenance windows, protocol-aware monitoring (Modbus/DNP3), physical access control |
| Embedded systems | Change default creds, disable debug interfaces (JTAG/UART), signed firmware updates, network segmentation, document unpatchable constraints + compensating controls |
| RTOS (Real-Time Operating System) | Minimal attack surface by design, watchdog timers, signed code only, isolate on dedicated network — availability/timing outweighs patching cadence; compensate with segmentation |
| IoT devices | Change default password FIRST, separate IoT VLAN/SSID, disable UPnP, firmware updates, block outbound except needed, inventory every device |

```
# Common thread for the "can't patch it" targets (ICS/SCADA, embedded,
# RTOS, IoT, legacy): SEGMENT + MONITOR + RESTRICT ACCESS =
# compensating controls. The exam loves this pattern.
```

## Wireless Installation Considerations (4.1)

```
# Site survey — walk the facility with analyzer to map RF coverage,
#   interference sources (microwaves, cordless phones, neighbor APs),
#   and optimal AP (access point) placement BEFORE deployment
# Heat map — visual overlay of signal strength on a floor plan; output
#   of a site survey; identifies dead zones and bleed-over past the
#   building perimeter (signal in the parking lot = war-driving risk)
# Channel planning — non-overlapping channels (2.4 GHz: 1/6/11);
#   5/6 GHz has more room; avoid co-channel interference
# AP placement — central to coverage area, minimize exterior bleed,
#   physically secure the AP (locked ceiling enclosure)
# Exam cue: "before installing new WAPs, determine placement" → site survey
# Exam cue: "visualize coverage/dead spots"                  → heat map
```

## Mobile Solutions (4.1)

```
# MDM (Mobile Device Management) — central console to enforce policy on
# enrolled devices: passcode/biometric requirements, encryption, app
# allow/block lists, remote lock/wipe, geofencing/geolocation, patch
# level enforcement, jailbreak/root detection, containerization
# (separate encrypted work profile — the BYOD privacy answer)
# UEM (Unified Endpoint Management) = MDM grown up to cover laptops too
# MAM (Mobile Application Management) = manage only the apps/work
# container, not the whole device — pairs with BYOD
```

| Model | Who owns it | Who controls it | Pros | Cons / exam angle |
|---|---|---|---|---|
| BYOD (Bring Your Own Device) | Employee | Shared (MDM/MAM on personal device) | Lowest hardware cost, user satisfaction | Weakest control; privacy conflicts; mixed personal/work data; onboarding/offboarding wipes are contentious |
| COPE (Corporate-Owned, Personally Enabled) | Company | Company (personal use allowed) | Full control + user convenience | Highest cost; company liable for device; personal data on corp asset |
| CYOD (Choose Your Own Device) | Company | Company (user picks from approved list) | Control of COPE + some user choice; limited support matrix | Cost; less flexible than BYOD |
| COBO (Corporate-Owned, Business Only) | Company | Company (no personal use) | Maximum lockdown (kiosk, field devices) | Users carry two devices; not in 701 acronym list but appears as distractor |

```
# Connection methods — risks and controls:
# Cellular (4G/5G) — carrier-encrypted air interface but treat as
#   untrusted transport: use VPN for corp traffic; rogue base station
#   (IMSI catcher/Stingray) risk; tethering can bypass corp perimeter
# Wi-Fi — evil twin / rogue AP risk; enforce WPA3 or WPA2-Enterprise,
#   forbid open hotspots without VPN, disable auto-join
# Bluetooth — keep non-discoverable, unpair unused devices, patch
#   (BlueBorne); attacks: bluejacking (unsolicited messages),
#   bluesnarfing (data THEFT via Bluetooth) — snarf = steal
```

## Wireless Security Settings (4.1)

```
# WPA3 (Wi-Fi Protected Access 3) — current standard
#   SAE (Simultaneous Authentication of Equals) replaces the WPA2 PSK
#   4-way handshake: password-authenticated key exchange (Dragonfly),
#   resistant to offline dictionary attacks, provides forward secrecy
#   Personal: SAE. Enterprise: 802.1X + optional 192-bit mode (CNSA suite)
#   Encryption: AES-GCMP/CCMP; Enhanced Open (OWE) encrypts open networks
#
# WPA2 — AES-CCMP; PSK handshake capturable → offline dictionary attack
#   (the reason WPA3/SAE exists); WPA/TKIP and WEP = broken, never answer
#
# AAA / RADIUS (Remote Authentication Dial-In User Service)
#   Authentication, Authorization, Accounting via central server;
#   the back end of WPA-Enterprise: AP = RADIUS client/authenticator,
#   user = supplicant, RADIUS = authentication server (802.1X roles)
#   UDP 1812 (auth) / 1813 (accounting); shared secret with the AP
#   TACACS+ = the device-administration AAA cousin (TCP 49, Cisco)
#
# Authentication protocols (EAP — Extensible Authentication Protocol):
#   EAP-TLS  — certificate on server AND client; strongest; cert mgmt cost
#   PEAP     — TLS tunnel w/ server cert, then inner auth (MSCHAPv2)
#   EAP-TTLS — like PEAP, more flexible inner methods
#   EAP-FAST — Cisco, PAC instead of server cert
#   Exam: "mutual certificate authentication" → EAP-TLS
```

| Setting | WPA2-PSK (Personal) | WPA3-SAE (Personal) | WPA2/WPA3-Enterprise (802.1X/EAP) |
|---|---|---|---|
| Authentication | Shared passphrase (4-way handshake) | Shared passphrase via SAE exchange | Per-user credentials/certs via RADIUS |
| Offline dictionary attack | Yes — capture handshake, crack offline | No — SAE resists offline guessing | No |
| Forward secrecy | No | Yes | Yes (TLS-based methods) |
| Per-user accountability | No (one shared key) | No | Yes (AAA accounting) |
| Revoke one user | Change key everywhere | Change key everywhere | Disable the account/cert |
| Infrastructure needed | None | WPA3-capable gear | RADIUS server + IdP/PKI |
| Best fit | Legacy small office | Home/small office | Enterprise (the exam's default answer) |

## Application Security (4.1)

```
# Input validation — treat ALL input as hostile; allow-list expected
#   format/length/type/range server-side (client-side is UX, not
#   security); primary defense against injection (SQLi, XSS, command
#   injection) and buffer overflow
# Secure cookies — Secure flag (HTTPS only), HttpOnly (no JavaScript
#   access → blunts XSS session theft), SameSite=Strict/Lax (CSRF
#   defense); Set-Cookie: session=abc; Secure; HttpOnly; SameSite=Strict
# Static code analysis (SAST) — scan SOURCE without executing; finds
#   injection sinks, hardcoded secrets, unsafe functions early in SDLC
#   (vs dynamic/DAST = test the RUNNING app; fuzzing = malformed input)
# Code signing — digitally sign binaries/scripts/firmware so consumers
#   verify integrity + publisher authenticity; unsigned/invalid
#   signature → do not execute (pairs with allow listing)
```

## Sandboxing and Monitoring (4.1)

```
# Sandboxing — run untrusted code in an isolated environment where it
#   cannot touch the real system: malware detonation appliances,
#   browser tab isolation, VMs/containers for testing, mobile app
#   sandboxes. Exam cue: "safely analyze suspicious attachment" → sandbox
#   Evasion note: modern malware sleeps/detects VMs to dodge sandboxes
#
# Monitoring (as a 4.1 technique) — instrument the hardened resource:
#   host logs → SIEM, EDR telemetry, FIM on critical files, performance
#   + availability baselines so anomalies stand out
#   Hardening without monitoring = you never learn the control failed
```

## Exam Cues — Keyword → Answer

| Question wording / keyword | Answer |
|---|---|
| "Prevent lateral movement" / "isolate legacy unpatchable system" | Segmentation |
| "Only approved applications can execute" | Application allow list |
| "Most secure software restriction approach" | Allow list (deny-by-default) beats block list |
| "Malware-infected host, immediate action" | Isolation / quarantine |
| "Stolen laptop, data unreadable" | Full-disk encryption |
| "User can access more than the job requires" | Least privilege |
| "Settings drift from approved config, auto-remediate" | Configuration enforcement |
| "Retired server still on network / disk resold with data" | Decommissioning + media sanitization (NIST SP 800-88) |
| "Blocks attacks on the host itself, inline" | HIPS (detect-only = HIDS) |
| "Camera/IoT compromised via factory credentials" | Change default passwords |
| "Reduce attack surface on a new build" | Remove unnecessary software, disable unused ports/protocols |
| "All new systems configured consistently/securely" | Secure baseline / golden image (establish→deploy→maintain) |
| "Industry consensus hardening checklist" | CIS Benchmarks / DISA STIGs |
| "Determine AP placement before install" / "find dead zones" | Site survey / heat map |
| "Enforce policy, remote wipe on phones" | MDM |
| "Employees use personal phones for work" | BYOD (+ MAM/containerization for privacy) |
| "Company buys device, personal use allowed" | COPE |
| "Users choose from company-approved device list" | CYOD |
| "Data stolen over Bluetooth" | Bluesnarfing (messages only = bluejacking) |
| "Wi-Fi resistant to offline dictionary attacks" | WPA3 SAE |
| "Per-user Wi-Fi auth with central server" | WPA-Enterprise = 802.1X + RADIUS |
| "Mutual certificate-based wireless auth" | EAP-TLS |
| "Prevent script access to session cookie" | HttpOnly flag |
| "Find flaws in source code before deployment" | Static code analysis (SAST) |
| "Verify software publisher and integrity" | Code signing |
| "Safely detonate suspicious file" | Sandboxing |
| "SCADA network protection" | Segmentation/air gap + restricted access, NOT internet-facing patching |

## Practice Drill

```
# Cover the right column of both tables above and quiz yourself.
# Then reverse: given the answer, produce two exam phrasings.
# Distractor discipline:
#   - HIDS vs HIPS → the P prevents; detection-only can't "block"
#   - Allow list vs block list → "most secure"/"default deny" = allow list
#   - Site survey (process) vs heat map (artifact of the process)
#   - MDM (device) vs MAM (apps only) vs UEM (all endpoints)
#   - SAE (personal WPA3) vs 802.1X/EAP (enterprise) — "per-user" = enterprise
#   - Decommissioning ≠ just powering off — creds, DNS, certs, media
```

## See Also

security-plus-sy0-701, secplus-architecture, secplus-monitoring-defense, secplus-iam, secplus-vulnerabilities, secplus-asset-vuln-management, hardening-linux, cis-benchmarks, endpoint-security, network-access-control, dot1x, sast-dast, sdlc-security, zero-trust, vulnerability-scanning, siem, ids-ips

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks)
- [DISA STIGs](https://public.cyber.mil/stigs/)
- [NIST SP 800-88 Rev.1 — Guidelines for Media Sanitization](https://csrc.nist.gov/pubs/sp/800/88/r1/final)
- [NIST SP 800-123 — Guide to General Server Security](https://csrc.nist.gov/pubs/sp/800/123/final)
- [NIST SP 800-82 Rev.3 — Guide to Operational Technology (OT) Security](https://csrc.nist.gov/pubs/sp/800/82/r3/final)
- [NIST SP 800-124 Rev.2 — Guidelines for Managing Mobile Device Security](https://csrc.nist.gov/pubs/sp/800/124/r2/final)
- [Wi-Fi Alliance — WPA3 Specification](https://www.wi-fi.org/discover-wi-fi/security)
- [RFC 2865 — RADIUS](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 5216 — EAP-TLS](https://www.rfc-editor.org/rfc/rfc5216)
- [Microsoft Security Baselines](https://learn.microsoft.com/en-us/windows/security/operating-system-security/device-management/windows-security-configuration-framework/windows-security-baselines)
- [OWASP Secure Headers / Session Management Cheat Sheets](https://cheatsheetseries.owasp.org/)
