# Vulnerability Types (Security+ SY0-701 Objective 2.3)

> Explain various types of vulnerabilities: application, OS, web-based (SQLi/XSS), hardware, virtualization, cloud-specific, supply chain, cryptographic, misconfiguration, mobile, and zero-day — with canonical mitigations and exam-cue keyword mappings.

## Objective Map

```
# SY0-701 Objective 2.3 — "Explain various types of vulnerabilities"
# Domain 2 (Threats, Vulnerabilities, and Mitigations) = 22% of the exam
#
# Sub-topics this sheet covers (all of them are fair game):
#   Application      — memory injection, buffer overflow, race conditions
#                      (TOC/TOU), malicious update
#   Operating system — OS-based vulnerabilities
#   Web-based        — SQL injection (SQLi), cross-site scripting (XSS)
#   Hardware         — firmware, end-of-life (EOL), legacy
#   Virtualization   — VM escape, resource reuse
#   Cloud-specific   — misconfigured storage, insecure APIs, shared tenancy
#   Supply chain     — service provider, hardware provider, software provider
#   Cryptographic    — weak algorithms, key management, downgrade exposure
#   Misconfiguration — defaults, open permissions, unneeded services
#   Mobile device    — side loading, jailbreaking/rooting
#   Zero-day         — unknown to the vendor, no patch exists
#
# How the exam words it: "Which type of VULNERABILITY..." (2.3) vs
# "Which type of ATTACK / indicator..." (2.4 → cs secplus-attack-indicators).
# 2.3 questions describe a WEAKNESS; 2.4 questions describe ACTIVITY.
```

## Application Vulnerabilities

```
# Memory injection
# - Attacker-supplied code/data is written into a running process's
#   memory space and executed with that process's privileges
# - Umbrella term: includes DLL (Dynamic Link Library) injection on
#   Windows — malicious library loaded into a legitimate process
# - Exam cue: "malicious code inserted into the memory of a running
#   process" / "DLL injection" → memory injection
# - Mitigation: DEP (Data Execution Prevention), ASLR (Address Space
#   Layout Randomization), code signing, EDR (Endpoint Detection and
#   Response) watching for cross-process memory writes

# Buffer overflow
# - Program writes MORE data into a fixed-size buffer than it can hold;
#   overflow overwrites adjacent memory (saved return address, function
#   pointers) → crash or attacker-controlled execution
# - Stack overflow (return address) vs heap overflow (heap metadata) —
#   SY0-701 only needs the umbrella concept
# - Classic unsafe C functions: strcpy(), gets(), sprintf() (no bounds
#   checking) — safe variants: strncpy(), fgets(), snprintf()
# - Exam cue: "input larger than the allocated buffer", "adjacent
#   memory overwritten", "0x41414141 / AAAA in a crash dump" → overflow
# - Mitigation: bounds checking / input validation, memory-safe
#   languages (Rust, Go, Java, C#), stack canaries, ASLR + DEP, patching

# Race condition — TOC/TOU (also written TOCTOU)
# - Two processes/threads access a shared resource in an unexpected
#   order; outcome depends on timing
# - TOC (Time-Of-Check): moment the program VALIDATES a resource
#   (e.g. "does the user have permission to this file?")
# - TOU (Time-Of-Use): moment the program actually USES the resource
# - The vulnerability = the GAP between check and use; attacker swaps
#   the resource (e.g. replaces file with a symlink to /etc/shadow)
#   in that window
# - Exam cue: "state changed between the time it was checked and the
#   time it was used", "two processes racing", "symlink swapped after
#   the permission check" → race condition / TOC-TOU
# - Mitigation: atomic operations, file locking, mutexes/semaphores,
#   re-check at use time, operate on file descriptors not path names

# Malicious update
# - A software update mechanism is abused to deliver malware — either
#   the vendor's update server is compromised or the client accepts
#   unsigned/unverified updates
# - Real-world anchor the exam alludes to: SolarWinds Orion (trojanized
#   signed update pushed to ~18,000 customers) — overlaps supply chain
# - Exam cue: "users installed a legitimate-looking update that
#   contained malware", "compromised update server" → malicious update
# - Mitigation: signed updates with signature VERIFICATION enforced,
#   official update channels only, hash validation, staged rollouts
```

## Operating System (OS-Based) Vulnerabilities

```
# Any weakness in the OS kernel, services, or bundled components
# - Unpatched OS (missing security updates) — the #1 practical cause
# - Default configurations and default accounts left enabled
# - Excessive service exposure (every listening service = attack surface)
# - Local privilege escalation flaws (user → SYSTEM/root)
# - Exam cue: "server missed several monthly patch cycles",
#   "vulnerability in the kernel" → OS-based vulnerability
# - Mitigation: patch management (test → deploy on cadence), hardening
#   baselines (CIS Benchmarks, DISA STIGs), least functionality
#   (remove/disable unneeded roles and services), endpoint protection
# - Related sheets: cs secplus-hardening, cs hardening-linux,
#   cs cis-benchmarks
```

## Web-Based Vulnerabilities

```
# SQL injection (SQLi)
# - Untrusted input concatenated into a SQL (Structured Query Language)
#   statement changes the query's logic
# - Signature exam string:  ' OR 1=1--
#   (closes the string literal, ORs a tautology so WHERE is always true,
#    -- comments out the rest of the query)
# - Impact: authentication bypass, data exfiltration, data tampering,
#   sometimes RCE (Remote Code Execution) via DB features
# - Exam cue: apostrophes/quotes in input fields, "OR 1=1", UNION SELECT
#   in web logs, "database error displayed after entering a quote" → SQLi
# - CANONICAL mitigation (in exam-preference order):
#   1. Parameterized queries / prepared statements (query structure is
#      compiled first; input can only ever be data, never code)
#   2. Stored procedures (when they don't build dynamic SQL internally)
#   3. Input validation (allow-list) — defense in depth, not sufficient alone
#   4. WAF (Web Application Firewall) — compensating control, virtual patch
#   5. Least-privilege DB account (app account can't DROP TABLE)
# - "Sanitize input" is the distractor; "parameterized queries" is the
#   BEST answer when both appear

# Cross-site scripting (XSS)
# - Attacker's SCRIPT executes in the VICTIM'S BROWSER in the context of
#   a trusted site (steals session cookies, rewrites the page, redirects)
# - Payload tell: <script>...</script>, onerror=, javascript: URIs
# - Three flavors (know the distinctions):
#   Stored (persistent) XSS  — payload saved server-side (comment field,
#     profile, forum post); EVERY visitor who views it is hit.
#     Cue: "all users who viewed the page were affected"
#   Reflected (non-persistent) XSS — payload lives in a crafted URL/
#     request and is echoed back immediately; victim must click the link.
#     Cue: "user clicked an emailed link containing a script in the
#     query string"
#   DOM-based XSS — the vulnerability is in CLIENT-SIDE JavaScript
#     (e.g. document.write(location.hash)); payload may never reach the
#     server. Cue: "server logs show nothing; flaw is in client script"
# - CANONICAL mitigations:
#   1. Output encoding / escaping (HTML-entity-encode untrusted data at
#      the point of output: < becomes &lt;)
#   2. Input validation (allow-list)
#   3. CSP (Content Security Policy) header — restricts which script
#      sources the browser may execute; blocks inline script
#   4. HttpOnly cookie flag — script cannot read the session cookie
#      (limits impact, doesn't prevent XSS itself)
# - Discriminator vs CSRF: XSS = attacker's code RUNS in the browser;
#   CSRF = attacker rides the victim's EXISTING authenticated session
#   without running script (CSRF is objective 2.4 — forgery)
```

## Hardware Vulnerabilities

```
# Firmware
# - Software embedded in hardware (BIOS/UEFI — Unified Extensible
#   Firmware Interface, drive controllers, NIC firmware, BMC — Baseboard
#   Management Controller)
# - Runs below the OS → compromise persists across OS reinstalls and is
#   invisible to most endpoint tools
# - Exam cue: "malware survives a full OS reinstall", "UEFI implant" →
#   firmware vulnerability / firmware rootkit
# - Mitigation: firmware updates from the vendor, UEFI Secure Boot,
#   measured boot / attestation (TPM — Trusted Platform Module),
#   hardware root of trust

# End-of-life (EOL)
# - Vendor NO LONGER SELLS or SUPPORTS the product — no more security
#   patches EVER; every new CVE (Common Vulnerabilities and Exposures)
#   is permanent
# - EOL vs EOSL nuance: end-of-life = no longer sold/updated;
#   end-of-service-life = even extended/paid support has ended
# - Exam cue: "operating system no longer receives patches",
#   "Windows Server 2008 still in production" → EOL vulnerability
# - Mitigation: replace/upgrade (best), or COMPENSATING CONTROLS while
#   migrating: network segmentation/isolation, strict firewall rules,
#   enhanced monitoring, virtual patching via IPS

# Legacy
# - Old technology still in use because something depends on it (often
#   ICS/SCADA, medical devices, mainframe apps); may or may not be EOL,
#   but cannot run modern controls (no agent support, weak/absent
#   crypto, hardcoded credentials)
# - Exam cue: "system cannot be patched because the vendor application
#   only runs on the old OS" → legacy; answer is almost always
#   SEGMENTATION / isolation as the compensating control
```

## Virtualization Vulnerabilities

```
# VM (Virtual Machine) escape
# - Attacker breaks OUT of a guest VM into the HYPERVISOR (or host),
#   from which ALL other guest VMs on that host can be compromised
# - Severity: critical — collapses the core isolation guarantee
# - Exam cue: "attacker in one VM accessed the hypervisor / other
#   tenants' VMs on the same host" → VM escape
# - Mitigation: patch the hypervisor promptly, minimize guest-host
#   integrations (shared folders, guest tools), host hardening,
#   separate sensitive workloads onto dedicated hosts

# Resource reuse
# - Hypervisor reallocates resources (memory pages, storage blocks)
#   from one VM to another WITHOUT scrubbing → new tenant reads the
#   previous tenant's residual data
# - Exam cue: "newly provisioned VM's disk contained fragments of
#   another customer's data" → resource reuse
# - Mitigation: zeroize/sanitize memory and storage before
#   reallocation; provider-side hygiene; encrypt data at rest so
#   residue is ciphertext

# VM sprawl (adjacent concept, appears as a distractor)
# - Unmanaged proliferation of VMs → unpatched, untracked instances
# - Mitigation: inventory, lifecycle policy, template baselines
```

## Cloud-Specific Vulnerabilities

```
# Misconfigured storage
# - The classic: object storage bucket (e.g. AWS S3) left PUBLIC —
#   world-readable data breach with zero "hacking" required
# - Exam cue: "researcher found company data in a publicly accessible
#   cloud bucket" → misconfiguration (cloud storage)
# - Mitigation: block-public-access settings, least-privilege bucket
#   policies, CSPM (Cloud Security Posture Management) scanning,
#   encryption at rest, audit with the shared responsibility model in
#   mind (customer configures; provider supplies the controls)

# Insecure APIs (Application Programming Interfaces)
# - Cloud is API-driven; weak authN/authZ, no rate limiting, verbose
#   errors, or keys leaked in code/repos expose the management plane
# - Exam cue: "API key committed to a public repository", "unauthenticated
#   API endpoint returned customer records" → insecure API
# - Mitigation: strong auth (tokens, short-lived credentials), TLS
#   (Transport Layer Security) everywhere, rate limiting/throttling,
#   input validation, API gateway, secret scanning + rotation

# Shared tenancy
# - Multiple customers on the same physical hardware; side-channel
#   attacks (cache timing, Spectre/Meltdown-class CPU flaws) or
#   hypervisor bugs can leak data ACROSS tenants
# - Exam cue: "concern that another customer on the same physical host
#   could access data" → shared tenancy risk
# - Mitigation: dedicated hosts/instances for sensitive workloads,
#   encryption, provider patch hygiene; accept-vs-avoid is a RISK
#   decision (ties to cs secplus-governance-risk)
```

## Supply Chain Vulnerabilities

```
# Weakness inherited from any THIRD PARTY in your product/service chain.
# The exam splits it three ways — match the actor:
#
# Service provider  — MSP (Managed Service Provider), cloud provider,
#   outsourced IT with privileged access to your environment.
#   Anchor incident style: attacker compromises the MSP's remote-admin
#   tool and pivots into all of its customers (Kaseya-style).
#   Cue: "attacker gained access through the company's managed IT vendor"
#
# Hardware provider — counterfeit or tampered equipment, implants added
#   in transit/manufacture, compromised firmware pre-installed.
#   Cue: "counterfeit network switches", "device arrived with malicious
#   firmware"
#
# Software provider — compromised vendor code or update pipeline
#   (SolarWinds), malicious/typosquatted packages in open-source
#   ecosystems, vulnerable bundled libraries (Log4j/Log4Shell exposure
#   through software you merely INCLUDED).
#   Cue: "trojanized update from a legitimate vendor", "vulnerable
#   open-source dependency"
#
# Mitigations (exam answers):
# - Vendor risk assessment / due diligence, right-to-audit clauses
# - SBOM (Software Bill of Materials) — inventory of components inside
#   software so you can find the vulnerable library fast
# - Trusted/authorized suppliers only; verify hashes and signatures
# - Contractual security requirements, continuous vendor monitoring
# - Deep dive: cs supply-chain-security, cs sbom,
#   cs secplus-third-party-compliance
```

## Cryptographic Vulnerabilities

```
# Weak / deprecated algorithms
# - Broken or under-strength primitives still in service:
#     Hashes:  MD5 (Message Digest 5), SHA-1 — collisions demonstrated
#     Ciphers: DES (Data Encryption Standard, 56-bit), RC4
#     Protocols: SSL (Secure Sockets Layer) 2/3, TLS 1.0/1.1, WEP
#       (Wired Equivalent Privacy)
#     RSA keys < 2048 bits
# - Exam cue: any of the above names in a scenario → weak-algorithm
#   vulnerability; answer = migrate to AES (Advanced Encryption
#   Standard), SHA-256+, TLS 1.2/1.3

# Improper key management
# - Hardcoded keys in source, keys stored beside the data they protect,
#   no rotation, no revocation, shared keys, weak key generation
#   (low-entropy RNG — Random Number Generator)
# - Exam cue: "private key found in the code repository", "same key
#   used for years" → key management vulnerability
# - Mitigation: HSM (Hardware Security Module) / KMS (Key Management
#   Service) / vault storage, rotation policy, separation of keys from
#   data, key escrow procedures (see cs secplus-cryptography, cs vault)

# Downgrade exposure
# - Systems that still ACCEPT weak protocol versions/ciphers can be
#   forced down to them by an on-path attacker (e.g. POODLE forcing
#   SSL 3.0), then broken
# - The vulnerability (2.3) is SUPPORTING the weak option; the
#   downgrade ATTACK itself is objective 2.4
# - Exam cue: "server still supports TLS 1.0 for backward compatibility"
#   → downgrade exposure
# - Mitigation: disable legacy versions/ciphers server-side, TLS 1.3
#   (removes renegotiation tricks), TLS_FALLBACK_SCSV, HSTS (HTTP
#   Strict Transport Security) against SSL stripping
```

## Misconfiguration

```
# Any security-relevant setting left wrong — consistently among the
# most common real-world root causes (and a favorite "BEST answer")
# - Default credentials still set (admin/admin)
# - Open/excessive permissions (world-writable shares, ANY-ANY firewall
#   rules)
# - Unneeded services/ports enabled; directory listing enabled
# - Verbose error messages leaking stack traces, paths, versions
# - Cloud storage exposure (see cloud-specific above — same family)
# - Exam cue: "device still uses the manufacturer's default password",
#   "port left open after a change" → misconfiguration
# - Mitigation: secure baselines + hardening guides (CIS), configuration
#   management with drift detection, change management, regular
#   configuration reviews/scans (cs secplus-hardening)
```

## Mobile Device Vulnerabilities

```
# Side loading
# - Installing apps from OUTSIDE the official app store (unknown-source
#   APKs on Android, third-party stores) — bypasses store vetting/signing
# - Exam cue: "user installed an app from a website instead of the
#   official store" → side loading
# - Mitigation: MDM (Mobile Device Management) policy blocking unknown
#   sources, allow-listed app catalogs, user training

# Jailbreaking (iOS) / rooting (Android)
# - Removing manufacturer OS restrictions to gain root/admin — disables
#   sandboxing, code-signing enforcement, and often breaks update paths
# - A jailbroken device can no longer be trusted to enforce policy
# - Exam cue: "device modified to remove vendor restrictions" →
#   jailbreaking/rooting
# - Mitigation: MDM with jailbreak/root DETECTION + conditional access
#   (quarantine/deny corporate resources), attestation checks
```

## Zero-Day

```
# A vulnerability UNKNOWN TO THE VENDOR (or known but unpatched) that
# attackers discover/exploit first — defenders have had "zero days" to
# prepare; NO PATCH EXISTS at time of exploitation
# - Signature-based tools (traditional AV, signature IDS) MISS zero-days
#   by definition — no signature exists yet
# - Exam cue: "no patch available", "previously unknown vulnerability",
#   "signature-based tools did not detect it" → zero-day
# - Mitigation (nothing prevents; these reduce blast radius):
#   defense in depth, behavior/anomaly-based detection (EDR, heuristic
#   IPS), application allow-listing, segmentation, least privilege,
#   threat intelligence feeds, compensating controls until patch ships
# - Once the vendor patches it, it is no longer a zero-day — unpatched
#   systems then have a KNOWN vulnerability (patch management problem)
```

## Vulnerability Class → Example → Mitigation Table

| Class | Canonical example | Primary mitigation |
|---|---|---|
| Memory injection | DLL injection into a running process | DEP + ASLR, code signing, EDR |
| Buffer overflow | `strcpy()` past a fixed buffer, return address overwritten | Input/bounds validation, memory-safe language, canaries |
| Race condition (TOC/TOU) | File swapped between permission check and open | Atomic ops, locking, re-check at use |
| Malicious update | Trojanized vendor update (SolarWinds-style) | Signed updates + verification, official channels |
| OS-based | Unpatched kernel/service flaw | Patch management, hardening baselines |
| SQLi | `' OR 1=1--` in a login field | Parameterized queries / prepared statements |
| Stored XSS | Script saved in a comment, hits every viewer | Output encoding, input validation, CSP |
| Reflected XSS | Script in an emailed link's query string | Output encoding, CSP |
| DOM XSS | Client-side JS writes `location.hash` to page | Safe DOM APIs, output encoding, CSP |
| Firmware | UEFI implant surviving OS reinstall | Firmware updates, Secure Boot, TPM attestation |
| End-of-life | OS past vendor support, no patches ever | Replace; segment + monitor as compensating control |
| Legacy | SCADA box that cannot be patched | Network segmentation / isolation |
| VM escape | Guest breaks into hypervisor | Hypervisor patching, minimal guest-host integration |
| Resource reuse | New VM reads prior tenant's residual disk data | Sanitize before reallocation, encrypt at rest |
| Misconfigured storage | Public cloud bucket leaks data | Block public access, least privilege, CSPM |
| Insecure API | API key leaked in public repo | Strong auth, rate limiting, secret rotation |
| Shared tenancy | Cross-tenant side channel on same host | Dedicated hosts, encryption |
| Supply chain (software) | Vulnerable open-source dependency (Log4j) | SBOM, dependency scanning, vendor assessment |
| Weak crypto | MD5 / DES / SSL 3.0 / WEP still enabled | Migrate to AES, SHA-256, TLS 1.2/1.3 |
| Key management | Private key hardcoded in source | HSM/vault, rotation, separation from data |
| Downgrade exposure | Server still accepts TLS 1.0 | Disable legacy versions/ciphers |
| Misconfiguration | Default credentials on network device | Secure baselines, config management, reviews |
| Side loading | App installed from unknown source | MDM blocking unknown sources |
| Jailbreaking/rooting | Vendor restrictions removed from device | MDM root detection + conditional access |
| Zero-day | Exploited flaw with no patch available | Defense in depth, behavior-based detection, allow-listing |

## Exam Cues — Keyword → Answer Reflex Table

| Question keyword / phrasing | Answer |
|---|---|
| `' OR 1=1--`, quote breaks the page, UNION SELECT | SQL injection |
| `<script>` tag, cookie theft via browser, `onerror=` | XSS |
| "Every user who VIEWED the page was affected" | Stored (persistent) XSS |
| "User CLICKED a crafted link" with script in URL | Reflected XSS |
| "Flaw entirely in client-side JavaScript; server never sees payload" | DOM-based XSS |
| "Input longer than the buffer", adjacent memory overwritten | Buffer overflow |
| "Two processes racing between check and use", TOC/TOU window | Race condition |
| "Code injected into a running process's memory / DLL injection" | Memory injection |
| "Legitimate update contained malware" | Malicious update / supply chain (software) |
| "Malware persists after full OS reinstall" | Firmware vulnerability |
| "No longer supported, no patches available" | End-of-life |
| "Cannot patch — vendor app requires old OS" → best control? | Segmentation (legacy compensating control) |
| "Guest VM accessed the hypervisor / other tenants" | VM escape |
| "New VM contained another customer's leftover data" | Resource reuse |
| "Publicly accessible cloud bucket" | Misconfigured storage (cloud) |
| "API key in a public repository" | Insecure API / key management |
| "Another customer on the same physical host" | Shared tenancy |
| "Breach came through the managed service provider" | Supply chain — service provider |
| "Counterfeit or tampered equipment" | Supply chain — hardware provider |
| MD5, SHA-1, DES, RC4, WEP, SSL 3.0 named | Weak/deprecated algorithm |
| "Forced to negotiate an older, weaker protocol" | Downgrade |
| "Default password still enabled" | Misconfiguration |
| "App installed from outside the official store" | Side loading |
| "Restrictions removed to get root access on phone" | Jailbreaking / rooting |
| "Previously unknown, no patch, signatures missed it" | Zero-day |
| "BEST prevents SQLi" (validation vs parameterized both listed) | Parameterized queries |
| "BEST limits which scripts a browser will run" | Content Security Policy (CSP) |

## Distractor Traps

```
# SQLi vs XSS — SQLi targets the DATABASE via query manipulation; XSS
#   targets the USER'S BROWSER via script. Payload with quotes/SQL
#   keywords → SQLi; payload with <script>/HTML → XSS.
# XSS vs CSRF — XSS runs attacker code in the browser; CSRF forges a
#   REQUEST using the victim's existing session (no script needs to run).
# Zero-day vs unpatched — if a patch EXISTS and wasn't applied, that is
#   NOT a zero-day; it's a patch-management / OS-based vulnerability.
# EOL vs legacy — EOL = vendor support ended; legacy = old tech still
#   depended on (may still be supported). Both usually answered with
#   compensating controls, especially segmentation.
# VM escape vs resource reuse — escape is an ACTIVE breakout to the
#   hypervisor; resource reuse is PASSIVE residual-data leakage.
# Malicious update vs software supply chain — the same scenario can be
#   either; pick whichever term appears in the options (malicious update
#   is the 2.3 application-vulnerability framing).
# Misconfiguration vs vulnerability (flaw) — a wrong SETTING is
#   misconfiguration; a code DEFECT is an application/OS vulnerability.
# Side loading vs jailbreaking — side loading = installing from outside
#   the store; jailbreaking = removing OS restrictions entirely.
```

## See Also

security-plus-sy0-701, secplus-threat-landscape, secplus-attack-indicators, secplus-hardening, secplus-cryptography, secplus-asset-vuln-management, secplus-third-party-compliance, vulnerability-scanning, supply-chain-security, sbom, cve, owasp-injection, cloud-security, sast-dast, cis-benchmarks, threat-modeling

## References

- CompTIA Security+ SY0-701 Exam Objectives — https://www.comptia.org/certifications/security
- OWASP Top 10 — https://owasp.org/www-project-top-ten/
- OWASP SQL Injection Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html
- OWASP XSS Prevention Cheat Sheet — https://cheatsheetseries.owasp.org/cheatsheets/Cross_Site_Scripting_Prevention_Cheat_Sheet.html
- MITRE CWE Top 25 Most Dangerous Software Weaknesses — https://cwe.mitre.org/top25/
- CWE-787 Out-of-bounds Write / CWE-367 TOCTOU Race Condition — https://cwe.mitre.org/
- NIST SP 800-40 Rev. 4: Enterprise Patch Management Planning — https://csrc.nist.gov/pubs/sp/800/40/r4/final
- NIST SP 800-161 Rev. 1: Cybersecurity Supply Chain Risk Management — https://csrc.nist.gov/pubs/sp/800/161/r1/final
- NIST SP 800-125A: Security Recommendations for Server-based Hypervisor Platforms — https://csrc.nist.gov/pubs/sp/800/125/a/r1/final
- NIST National Vulnerability Database — https://nvd.nist.gov/
- CISA Known Exploited Vulnerabilities Catalog — https://www.cisa.gov/known-exploited-vulnerabilities-catalog
- Content Security Policy (MDN) — https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
