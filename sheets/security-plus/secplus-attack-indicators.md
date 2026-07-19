# Indicators of Malicious Activity (Security+ SY0-701 Objective 2.4)

> Malware families, network/application/cryptographic/password attacks, and — the actual skill 2.4 tests — reading an indicator (impossible travel, out-of-cycle logs, account lockout) and naming the attack behind it.

## Malware Disambiguation (the must-know table)

| Malware | Defining trait | Tell-tale indicator |
|---|---|---|
| Ransomware | Encrypts data, demands payment | Ransom note, mass file renames/extensions, spiking disk I/O |
| Trojan | Malicious code disguised as legitimate software | "Free" utility installed, then odd outbound connections |
| Worm | Self-replicates across the network — no user action needed | Same infection appearing on many hosts fast; network saturation |
| Virus | Attaches to files/programs; needs a user to run the host file | Infected executables/documents; spreads on open/execute |
| Spyware | Covertly collects user data/activity | Browser redirects, unknown toolbars, data leaving the host |
| Bloatware | Unwanted pre-installed software (not strictly malicious) | New machine slow, unremovable vendor apps |
| Keylogger | Records keystrokes (software or hardware dongle) | Credentials stolen despite no phishing; unknown USB device inline |
| Logic bomb | Dormant code that triggers on a condition/date | Damage at a specific time or after a specific event (e.g. admin's account disabled) |
| Rootkit | Hides itself and other malware at OS/kernel level | AV can't see the process; scans from a boot disk find what the live OS can't |
| RAT (Remote Access Trojan) | Gives attacker interactive remote control | Off-hours mouse movement, remote-desktop-like traffic |

```
# Classic exam traps:
# Worm vs virus — worm needs NO user interaction and self-propagates;
#   virus needs a host file and a human to execute it.
# Logic bomb — the keyword is a TRIGGER CONDITION ("script ran the day
#   after the developer was terminated").
# Rootkit — the keyword is CONCEALMENT ("tools show nothing, but network
#   traffic proves compromise"); detection often requires booting from
#   trusted media or comparing hashes offline.
# Bloatware — annoying, capacity-wasting, expands attack surface; the
#   answer when nothing is overtly hostile but "came pre-installed".
```

## Physical Attacks (2.4)

```
# Brute force (physical)  — literally forcing entry: crowbar, door ram,
#   smashing a lock. Indicator: damaged locks/doors/racks.
# RFID (Radio Frequency Identification) cloning — copying a badge/fob
#   with a handheld reader. Indicator: badge use at odd hours from an
#   employee who is provably elsewhere; two simultaneous uses of one
#   badge in different buildings.
# Environmental — attacking the facility conditions: cutting power,
#   disabling HVAC (Heating, Ventilation, Air Conditioning) so servers
#   overheat, triggering water/fire systems.
```

## Network Attacks (2.4)

```
# DDoS (Distributed Denial-of-Service)
#   Amplified — small spoofed query → huge response (DNS, NTP, memcached
#     amplification). Amplification factor makes tiny botnets loud.
#   Reflected — spoof the VICTIM's IP as the source, so third-party
#     servers "reply" to the victim. (Amplified attacks are almost
#     always reflected too — the exam may ask for either trait.)
#   Indicators: resource consumption spikes, resource inaccessibility,
#   massive inbound from many sources, one UDP service dominating.

# DNS (Domain Name System) attacks
#   DNS poisoning/spoofing — forged records cached by a resolver; users
#     silently redirected to attacker infrastructure.
#     Indicator: correct URL, wrong site/certificate warnings.
#   DNS hijacking — registrar/zone records altered.
#   Domain generation / DNS tunneling — huge volumes of random-looking
#     subdomain lookups = C2 (command and control) or exfil over DNS.
#   Mitigation: DNSSEC (DNS Security Extensions), monitor query logs.

# Wireless
#   Evil twin — rogue AP with a legitimate SSID; users associate, all
#     traffic is intercepted. Indicator: duplicate SSID, users report
#     captive portals asking for creds.
#   Deauthentication — spoofed 802.11 deauth frames knock clients off,
#     often to force reconnection through an evil twin or capture WPA
#     handshakes. Mitigation: 802.11w protected management frames.

# On-path (formerly man-in-the-middle, MITM)
#   Attacker relays/alters traffic between two parties: ARP (Address
#   Resolution Protocol) poisoning on the LAN, rogue gateway, SSL strip.
#   Indicators: duplicate ARP replies, gateway MAC changes, unexpected
#   certificate errors. Mitigations: dynamic ARP inspection, TLS
#   everywhere with HSTS (HTTP Strict Transport Security).

# Credential replay
#   Captured authentication material (hash, token, cookie) re-sent to
#   authenticate as the victim — pass-the-hash, session-cookie reuse.
#   Indicators: valid logins from wrong hosts/geos; same session token
#   from two IPs. Mitigations: MFA, token binding, short session
#   lifetimes, Kerberos armoring/NTLM (New Technology LAN Manager)
#   deprecation.

# Malicious code
#   Umbrella term: scripts, macros, injected shellcode observed in
#   traffic or on hosts — the indicator is the payload itself.
```

## Application Attacks (2.4)

```
# Injection — untrusted input executed as code/commands: SQLi
#   (Structured Query Language injection), command injection, LDAP
#   injection. Indicator: quotes/semicolons/UNION in logs, database
#   errors returned to clients. Fix: parameterized queries, input
#   validation. (Vuln-side detail: secplus-vulnerabilities.)
# Buffer overflow — input exceeds a fixed-size buffer, overwriting
#   adjacent memory to crash or hijack execution. Indicators: service
#   crashes, unusually long inputs in logs. Mitigations: ASLR (Address
#   Space Layout Randomization), DEP (Data Execution Prevention),
#   memory-safe languages, patching.
# Replay — capture a valid request (payment, auth) and resend it.
#   Mitigations: nonces, timestamps, sequence numbers, TLS.
# Privilege escalation — going from low to high privilege.
#   Vertical: user → admin/root. Horizontal: user A → user B's access.
#   Indicators: standard account suddenly performing admin actions.
# Forgery
#   CSRF (Cross-Site Request Forgery) — victim's browser is tricked
#     into sending an authenticated request the user never intended.
#     Fix: anti-CSRF tokens, SameSite cookies.
#   SSRF (Server-Side Request Forgery) — the SERVER is tricked into
#     making internal requests (cloud metadata endpoint 169.254.169.254
#     is the classic target).
# Directory traversal — ../../ sequences escaping the web root:
#   GET /view?file=../../../../etc/passwd
#   Indicators: %2e%2e%2f or ../ patterns in web logs.
#   Fix: canonicalize + validate paths, run with least privilege.
```

## Cryptographic Attacks (2.4)

```
# Downgrade — force endpoints to negotiate a weaker protocol/cipher
#   (TLS 1.3 → TLS 1.0, POODLE-style). Indicator: sessions using
#   protocols your policy disables. Fix: disable legacy versions,
#   TLS_FALLBACK_SCSV, HSTS.
# Collision — two different inputs → same hash. Breaks integrity and
#   signatures. MD5 (Message Digest 5) and SHA-1 have practical
#   collisions — that is WHY they're banned for signatures.
# Birthday — the probability math behind collisions: with ~sqrt(2^n)
#   attempts a collision in an n-bit hash becomes likely (~50% at 2^(n/2)).
#   Mitigation for both: longer, modern hashes (SHA-256+).
```

## Password Attacks (2.4)

| Attack | Pattern | Give-away indicator |
|---|---|---|
| Brute force | Every combination against one account | One account, thousands of rapid failures → account lockout |
| Dictionary | Wordlist against one account | Failures using common words/leaked lists |
| Spraying | ONE common password against MANY accounts | One failure per account across hundreds of accounts, then pause (dodges lockout) |
| Credential stuffing | Leaked user:pass pairs replayed elsewhere | Valid-looking logins for many users from one source |

```
# Spraying vs brute force is a guaranteed question:
#   brute force = many passwords, one account (lockouts fire)
#   spraying    = one password, many accounts (lockouts do NOT fire —
#                 that's the point). Indicator: low-and-slow failed
#                 logins spread across the whole directory.
# Offline variants attack stolen hash databases — defense is salting +
#   key stretching (bcrypt/PBKDF2) → secplus-cryptography.
# Defenses: MFA, lockout policy, banned-password lists, monitoring.
```

## Indicator → Attack Mapping (the 2.4 core skill)

| Indicator | What it suggests |
|---|---|
| Account lockout | Brute-force/dictionary attack against that account |
| Concurrent session usage | Stolen credentials/session — same user active from two places |
| Blocked content | Controls firing — someone probing filtered categories/exfil paths |
| Impossible travel | Login from NY then Moscow 10 min later → compromised credentials |
| Resource consumption | DDoS, cryptomining malware, worm propagation |
| Resource inaccessibility | DoS in progress, or ransomware encrypted the share |
| Out-of-cycle logging | Activity at times nothing should run — attacker working off-hours |
| Published/documented | Your data/creds appear on paste sites or the dark web — breach confirmed |
| Missing logs | Log tampering/deletion — attackers covering tracks (classic rootkit/insider move) |

```
# Segmentation-of-evidence tip for scenario questions: the answer is the
# attack whose indicator is QUOTED, not the scariest-sounding option.
# "Logs are missing for the window in question" → tampering/cover-up,
# not DDoS. "CPU pegged on all web servers, site unreachable" → DDoS,
# not rootkit.
```

## Exam Cues (2.4)

```
# "files encrypted + payment demand"          → ransomware
# "spread with no user interaction"           → worm
# "triggered after developer termination"     → logic bomb
# "AV shows clean but C2 traffic persists"    → rootkit
# "unknown device between keyboard and PC"    → hardware keylogger
# "duplicate SSID in the lobby"               → evil twin
# "clients dropped from Wi-Fi repeatedly"     → deauthentication
# "gateway MAC address changed"               → on-path / ARP poisoning
# "small DNS queries, huge responses to victim" → amplified/reflected DDoS
# "../ in URL"                                → directory traversal
# "'; DROP TABLE"                             → SQL injection
# "one password tried on every account"       → password spraying
# "login NY + Moscow same hour"               → impossible travel (stolen creds)
# "TLS 1.0 negotiated despite policy"         → downgrade attack
# "two files, same MD5"                       → collision
# "badge used while owner on vacation"        → RFID cloning
```

## See Also

security-plus-sy0-701, secplus-threat-landscape, secplus-vulnerabilities, secplus-hardening, secplus-monitoring-defense, secplus-incident-response, secplus-cryptography, ids-ips, siem, log-analysis, mitre-attack, owasp-injection, owasp-auth, network-defense, endpoint-security

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [MITRE ATT&CK Enterprise Matrix](https://attack.mitre.org/matrices/enterprise/)
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP Cross-Site Request Forgery Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)
- [CISA — Understanding and Responding to Distributed Denial-of-Service Attacks](https://www.cisa.gov/resources-tools/resources/understanding-and-responding-distributed-denial-service-attacks)
- [NIST SP 800-63B — Digital Identity Guidelines (throttling & secrets)](https://pages.nist.gov/800-63-3/sp800-63b.html)
- [US-CERT — Ransomware guidance (StopRansomware)](https://www.cisa.gov/stopransomware)
