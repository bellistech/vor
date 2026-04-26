# IP Phone Provisioning — Deep Dive

Bootstrap protocol theory, trust-chain analysis, and the threat model around zero-touch IP-phone deployment.

## Setup

The bootstrap problem for IP phones is fundamental: a phone arriving at a customer site has no knowledge of its target environment. It does not know its phone number, its SIP credentials, the IP of its registrar, the URL of its provisioning server, the cipher suite to negotiate, or even its own VLAN ID. It has only what its manufacturer baked in: a factory firmware, a manufacturer-installed certificate (MIC), and a default behavior of "ask the network for instructions". From this minimal state, the phone must somehow arrive at a fully configured, registered, ready-to-make-calls state without human intervention.

Zero-touch provisioning solves this via a chain of trust anchored in three pieces:
1. **DHCP**: the local network tells the phone "your provisioning server is at this URL".
2. **The MIC**: the phone uses its factory cert to authenticate the provisioning server's TLS cert during the initial fetch.
3. **The provisioning server**: the server, having authenticated the phone (via the phone's MAC, MIC, or both), delivers per-MAC configuration that includes SIP credentials, registrar info, codec preferences, and tenant-specific parameters.

This chain has a chicken-and-egg quality: the phone needs the provisioning URL to know where to go, but the provisioning URL is delivered via DHCP (which is unauthenticated), and the phone needs TLS to fetch the config securely (which requires time-of-day for cert validation, which requires NTP, which is fetched via DHCP). The protocols layer in such a way that early bootstrap steps are weakly authenticated and later steps establish stronger trust. The threat model must account for this layering: a rogue DHCP server can hijack the entire bootstrap, but the MIC limits what the rogue server can do (it cannot forge a TLS cert for the legitimate provisioning service unless it controls the public CA chain).

The phone's first-boot behavior is deterministic: power on → DHCP discover → DHCP offer (with options) → DHCP request → DHCP ack → fetch firmware (if option specifies) → fetch config (if option specifies) → register with SIP server. Each step has well-defined failure modes; each step's protocol details matter for both reliability and security.

The trust establishment is one-way at first (the phone trusts the network), then becomes mutual (the provisioning server authenticates the phone via MAC/serial/MIC). The phone's MIC — a per-device certificate baked into the factory image — enables the server to verify "this is genuinely a Polycom/Yealink/Snom device with serial X". Subsequent provisioning rotations replace the MIC with a tenant-controlled cert (Locally Significant Certificate, LSC, or equivalent), giving the tenant cryptographic authority over which devices register at the tenant's PBX.

This document covers the theory: how trust is established, how the protocols compose, what the threat model is, what the failure modes are, and how to debug. It complements the cheatsheet by explaining the *why* of every protocol detail.

## Trust Establishment

The phone arrives knowing essentially nothing — only its serial number, its MAC address, its firmware version, and its MIC. The network it is connected to is the only knowledge source.

The DHCP server provides the phone with: an IP address (DHCP-assigned), DNS servers (option 6), domain name (option 15), the gateway (option 3), NTP server (option 42), and — crucially — the provisioning URL via a vendor-specific option. Common vendor option codes:
- **Cisco**: option 150 (TFTP server IP), option 66 (TFTP server name).
- **Polycom**: option 160 (URL).
- **Yealink**: option 66 (URL or hostname; deprecated for option 43 with sub-options).
- **Grandstream**: option 43.
- **Snom**: option 66.
- **Avaya**: option 242.

The vendor-specific option points at a HTTPS or HTTP URL. The phone fetches its config from this URL. If the URL is HTTPS, the phone must validate the server's TLS certificate.

The MIC (manufacturer-installed certificate) is the trust anchor for the initial TLS validation. The MIC is signed by a per-vendor CA hierarchy:
- **Polycom**: Polycom Manufacturing CA → Polycom Device CA → device cert.
- **Yealink**: Yealink CA → device cert.
- **Cisco**: Cisco Manufacturing CA (CMCA) → MIC.

The provisioning server's TLS cert is signed by either:
1. A public CA (DigiCert, Let's Encrypt), in which case the phone validates via the public CA roots in its truststore.
2. A vendor-managed CA, in which case the phone validates via the vendor CA in its truststore.
3. A tenant-managed CA, in which case the phone must have the tenant CA pre-installed (which requires a prior provisioning round).

The trust establishment is a chain: phone trusts MIC → MIC trusts vendor CA → vendor CA trusts provisioning server's cert (if vendor-issued) or public CA (if public-issued). This chain is the foundation of the phishing-resistance argument: a rogue DHCP server can point the phone at a rogue provisioning URL, but the rogue provisioning server cannot forge a TLS cert that chains to the legitimate CA, so the phone refuses the TLS connection.

After the first successful provisioning, the tenant can install its own LSC (locally significant certificate) on the phone. Subsequent provisioning rounds use the LSC for mutual TLS, replacing the MIC-based trust with tenant-controlled trust. The LSC is the cryptographic credential that binds the phone to the tenant's PBX.

## MIC (Manufacturer-Installed Certificate)

The MIC is a per-device X.509 certificate baked into the phone's firmware at the factory. It contains:
- Subject: typically `CN=<vendor>-<model>-<serial>` or similar.
- Issuer: the vendor's manufacturing CA.
- Public key: the device's public key (matching a private key stored in secure storage on the device — tamper-resistant).
- Validity: typically 10-30 years (longer than the phone's expected lifetime).
- Extensions: vendor-specific OIDs identifying the device class.

The MIC's purpose is twofold:
1. **Server-to-phone authentication**: the phone uses the MIC to authenticate inbound TLS connections from the provisioning server. The server presents its TLS cert; the phone validates the chain to a CA the phone trusts (either the vendor CA or a public CA). The MIC is not directly used here, but the phone's vendor truststore is.
2. **Phone-to-server authentication**: in mutual TLS deployments, the phone presents its MIC to the server. The server validates the MIC chain to the vendor CA, confirming the phone is genuine.

The phishing-resistance argument:
- Without MIC: a rogue DHCP server could point the phone at a rogue provisioning server with a self-signed cert. If the phone disabled cert validation (some legacy phones do, especially on first boot), the phone would fetch config from the attacker.
- With MIC and proper validation: the rogue provisioning server cannot forge a cert that chains to a CA the phone trusts. TLS handshake fails.

The MIC is *not* a panacea. Attacks on MIC-based trust:
- **Vendor CA compromise**: if the vendor CA is breached (rare but not unprecedented), an attacker can issue rogue certs that the phone will trust.
- **MIC extraction**: physical access to the phone allows extraction of the MIC private key (if the phone lacks tamper-resistant key storage). The attacker can then impersonate the phone to the legitimate provisioning server.
- **Pre-MIC firmware**: very old phones may lack MICs; they accept any TLS cert with weak validation.
- **Trust-On-First-Use (TOFU)**: some phones, on first boot with an empty truststore, accept the first cert presented and pin it. A rogue server during first boot can plant its cert.

Modern best practice: phones with HSM-backed key storage (most enterprise-class phones), per-device unique keys (no shared private keys), and short MIC validity (renewable via the vendor's PKI infrastructure).

## DHCP Option Threat Model

DHCP is unauthenticated. The DHCP protocol (RFC 2131) provides no mechanism to verify that a DHCP offer comes from a legitimate server. Any device on the same broadcast domain can answer DHCP DISCOVER messages.

The threat: a rogue DHCP server on the local segment can hijack a phone's bootstrap by:
1. Answering DHCP DISCOVER faster than the legitimate DHCP server.
2. Providing a different DNS server (pointing to a rogue resolver).
3. Providing a different provisioning URL (pointing to a rogue config server).
4. Providing a different NTP server (causing time skew that breaks cert validation).

If the phone's TLS cert validation is robust (MIC-based), the rogue config server cannot forge certs and TLS fails. But the phone may still leak information: the rogue server sees the phone's MAC, model, firmware version, and request patterns. This is a reconnaissance vector.

Some phones, faced with a TLS failure, fall back to HTTP. This is catastrophic: the rogue server can serve HTTP config without TLS validation. Fallback-to-HTTP must be disabled in any production deployment.

DHCP-snooping defense at the switch layer mitigates the threat:
1. **DHCP snooping**: the switch tracks which ports are connected to legitimate DHCP servers and blocks DHCP OFFER/ACK from non-trusted ports.
2. **Port security**: limits MAC addresses per port, preventing MAC-flooding attacks that could overwhelm the switch.
3. **DHCP option-82**: the switch inserts the relay-agent information option (option 82) into DHCP requests, allowing the DHCP server to verify the source port.

Per-MAC ACLs at the switch can pin a MAC to a specific port, preventing MAC-spoofing from a rogue device. This is a manual configuration; not zero-touch.

For wireless deployments (less common for IP phones, but increasing), 802.1X authentication (with EAP-TLS) provides per-device authentication at the network layer, eliminating the rogue-DHCP threat by requiring devices to authenticate before DHCP runs.

The defensive pattern: DHCP snooping + port security + 802.1X = layered defense. Each layer defends against one threat: DHCP snooping against rogue DHCP server, port security against MAC flooding, 802.1X against rogue device.

## The Redirector Service Chain

Vendors offer cloud-based "redirector" services that provide first-touch routing for new phones. The phone, on first boot, contacts the vendor's redirector using its MIC for authentication. The redirector looks up the phone's serial in its database; if the phone has been pre-claimed by a tenant (via the tenant's account at the vendor), the redirector returns the tenant's provisioning URL.

Major vendor redirectors:
- **Polycom RPRM** (RealPresence Resource Manager) — successor: Polycom ZTP (Zero-Touch Provisioning) at zerotouch.polycom.com.
- **Yealink RPS** (Redirection and Provisioning Service) — at api.yealink.com or rps.yealink.com.
- **Snom RPS** — at provisioning.snom.com.
- **Grandstream GDMS** (Grandstream Device Management System) — at gdms.cloud.
- **Cisco Webex Calling provisioning** — for Cisco MPP phones, a similar redirector is at activation.webex.com.

The flow:
1. Phone boots, sees no local provisioning URL (or sees a default vendor URL).
2. Phone contacts vendor redirector via HTTPS (with MIC mTLS).
3. Redirector validates MIC, looks up serial in its DB.
4. Redirector returns: tenant's provisioning URL, possibly tenant cert pin.
5. Phone fetches config from tenant URL, with tenant cert pinning.

Tenant binding: the tenant administrator pre-registers the phone's MAC or serial number at the vendor's redirector portal. The portal binds the device to the tenant's account. Without this binding, the redirector returns an error (and the phone fails to provision).

The trust model for the redirector:
- Vendor (e.g., Polycom) controls the redirector.
- Tenant (e.g., a customer) trusts the vendor to correctly bind devices.
- Phone trusts the vendor (via MIC + vendor CA in truststore).

The threat: vendor redirector compromise. If the vendor's redirector is breached, an attacker could redirect phones to a rogue server. Mitigations: vendor uses defense-in-depth; tenant uses cert pinning at the phone (so even a redirector hijack can't deliver a valid cert).

## Configuration File Encryption

Provisioning config files often contain SIP credentials in cleartext. If transported over plain HTTP, an attacker on the wire can extract them. TLS encrypts the transport; configuration file encryption adds a second layer.

Vendor schemes:
- **Polycom**: AES-256-CBC encryption of the config XML. The encryption key is per-device, derived from the phone's MAC and a tenant secret. The phone, knowing its MAC and the tenant secret (delivered via a separate channel), can decrypt.
- **Yealink**: AES-128-CBC. Similar key derivation.
- **Cisco**: in MPP firmware, configuration files are signed and optionally encrypted using device-specific keys derived during provisioning enrollment.

The per-MAC encryption key delivery is the critical design point. If the encryption key is delivered alongside the encrypted config, an attacker with access to one can derive the other. Vendor schemes typically deliver the key via an out-of-band channel:
- Initial provisioning: the key is delivered over TLS (so eavesdropping requires breaking TLS).
- Subsequent provisioning: the key is delivered encrypted with the device's MIC public key (so only the device can decrypt).

Configuration file encryption defends against:
- Compromise of the provisioning server's filesystem (encrypted files at rest are useless without the per-device keys).
- Eavesdropping on TLS-disabled deployments (rare, but exists in legacy environments).
- Lateral movement: if an attacker gains access to one phone's config, they cannot trivially read another phone's config without the other phone's key.

Configuration file encryption does *not* defend against:
- Compromise of the device itself (the device has the decryption key).
- Compromise of the master tenant secret (if the tenant secret is leaked, all per-device keys derivable).
- Compromise of the encryption-key-derivation algorithm (if the algorithm is reversible without the secret, all configs are vulnerable).

## Authentication Methods

The provisioning server can authenticate the phone via several mechanisms:

**HTTP Basic over TLS**: the phone sends username/password in the HTTPS request. The username is typically the MAC (e.g., `001565AABBCC`); the password is a tenant-shared secret or per-device password. TLS encrypts the credentials in transit.

The risk: if TLS is misconfigured (e.g., self-signed cert accepted), credentials are visible to a MITM attacker. Best practice: enforce TLS with cert validation; never fall back to HTTP.

**mTLS via device cert**: the phone presents its MIC (or LSC) as a TLS client cert. The server validates the cert chain to a trusted CA. No password required.

mTLS is stronger than HTTP Basic because:
- Credentials cannot be replayed (the cert is tied to the device's private key).
- Compromise of the server's password database doesn't compromise the auth (no passwords stored).
- Each device has unique credentials (no shared secrets).

mTLS is operationally heavier:
- Cert lifecycle management (issuance, renewal, revocation).
- Cert rollout to phones (requires initial provisioning round to install LSC).

**SIP credentials in config**: a common pattern is to embed SIP REGISTER credentials (username, password) in the config file. The phone uses these to register with the SIP server.

The risk: if the config file is fetched over HTTP without TLS, SIP credentials are visible on the wire. This is the source of many SIP-credential-leak incidents — phones provisioned over HTTP, attacker captures the config, attacker registers as the phone, attacker makes calls or eavesdrops.

The mitigation: always use HTTPS with cert validation; encrypt the config file at rest.

## The "Time Set" Bootstrap Order

A subtle but critical bootstrap problem: TLS cert validation requires accurate time-of-day. If the phone's clock is wrong (e.g., far in the past), a valid cert will appear "not yet valid" or "expired".

The phone's clock at boot is unreliable:
- Some phones have a real-time clock (RTC) with battery backup; they retain time across reboots.
- Some phones lose time on power-off; they boot with the clock at epoch (1970-01-01).
- Some phones use NTP-only timekeeping; they require NTP sync before they have time.

The chicken-and-egg problem: the phone needs NTP to set the clock; NTP is fetched from a server (DNS-resolved); DNS is fetched via DHCP; the phone needs to validate TLS for the provisioning fetch, which requires the clock to be set.

Vendor solutions:
1. **Weak HTTP for first time-sync**: the phone fetches NTP server config via plain HTTP (no TLS), syncs clock, then upgrades to HTTPS for config fetch. This opens a window for MITM during the HTTP step.
2. **DHCP option for NTP**: option 42 carries the NTP server; the phone uses this without HTTP. Time sync via NTP doesn't require accurate clock.
3. **OCSP-Must-Staple deferral**: some phones defer cert validation until after NTP sync, then validate.
4. **Pre-installed time**: some phones ship with a "time floor" (e.g., the manufacturing date) below which all certs are presumed valid; this prevents the "not yet valid" failure mode.
5. **Time-tolerant cert validation**: some phones validate certs with a generous clock skew (e.g., 30 days), accepting certs that are "expired but recently valid".

The robust design: NTP early, before any TLS. NTP itself is unauthenticated by default (NTPv4 doesn't require auth), but the threat surface is small (an NTP attacker can skew time but not exfiltrate data). Some deployments use NTS (Network Time Security, RFC 8915) for authenticated NTP, eliminating this threat.

Production deployments: ensure DHCP option 42 is set to a trusted NTP server; ensure the phone's time-floor is set to a recent date; enable OCSP-Must-Staple for cert validation.

## Firmware Trust Chain

Firmware on IP phones is signed by the vendor. The phone, on receiving a firmware update, validates the signature before installing. The validation chain:
- Firmware image has an embedded signature (typically RSA or ECDSA over a SHA-256 or SHA-384 digest of the image).
- Signature is by the vendor's firmware-signing key.
- Vendor's public key is baked into the phone's bootloader (Read-Only).

The "phone refuses unsigned" rule: most phones reject unsigned firmware. This prevents arbitrary code execution on the phone — an attacker who controls the provisioning server cannot push a malicious firmware unless they also have the vendor's signing key.

Rare-but-real "factory unlock" backdoor: some phones have a debug/factory mode that allows unsigned firmware loading, typically requiring physical access (e.g., holding a button during boot) and a vendor-controlled cryptographic challenge-response. This is used by the vendor for development and warranty repair. Attackers with physical access and the unlock procedure can install rooted firmware.

Firmware trust attacks:
- **Vendor signing-key compromise**: rare, catastrophic. Allows arbitrary firmware to be signed.
- **Firmware downgrade**: pushing an old (signed) firmware that has known vulnerabilities. Some phones enforce monotonic-version upgrade (rejecting older signed firmware); others don't.
- **Bootloader vulnerabilities**: bugs in the firmware-validation code can allow signature bypass. Vendor patches via firmware updates.

The defensive pattern: keep firmware up-to-date; enforce monotonic-version upgrade where possible; use hardware-backed key storage so even firmware compromise doesn't leak the device's private key.

## LLDP-MED Auto-Provisioning

LLDP-MED (Link Layer Discovery Protocol — Media Endpoint Discovery, ANSI/TIA-1057) is a switch-to-phone protocol that delivers VLAN, PoE, and location info without requiring DHCP options.

The flow:
1. Phone boots, transmits LLDP-MED packets identifying itself as a VoIP endpoint.
2. Switch responds with LLDP-MED packets containing:
   - **Voice VLAN ID**: the VLAN to which the phone should tag traffic (typically VLAN 100-200 for voice).
   - **Voice priority (802.1p)**: the QoS priority for voice traffic (typically 5).
   - **PoE class**: the power class allocated to the phone.
   - **Civic location** (optional): physical address for E911.
   - **Coordinate location** (optional): GPS lat/lon for E911.
3. Phone reboots (or re-tags), now using the Voice VLAN.
4. On the Voice VLAN, the phone gets a different DHCP scope (with different DNS, NTP, provisioning URLs — typically locked-down for voice traffic).

The Voice VLAN segregation is a security measure: voice traffic is isolated from data traffic, limiting the impact of a compromise on either side. It also enables QoS prioritization at the network layer.

LLDP-MED is industry-standard but has vendor-specific extensions:
- **Cisco CDP** (Cisco Discovery Protocol): a Cisco-only predecessor to LLDP-MED, still widely used in Cisco environments.
- **Polycom LLDP-MED extensions**: vendor-specific TLVs for additional info.

LLDP-MED is unauthenticated. A rogue switch (or a switch with MAC spoofing on a port) can deliver fake LLDP-MED, redirecting the phone to an attacker-controlled VLAN. Mitigations: physical port security; 802.1X for switch-to-switch authentication; LLDP-MED only on trusted switch ports.

## PoE Class Negotiation (802.3af/at/bt)

Power-over-Ethernet (PoE) delivers electrical power over the Ethernet cable, eliminating the need for a separate power supply for IP phones.

PoE standards:
- **802.3af** (PoE, 2003): up to 15.4W at the source, 12.95W at the device. Class 0-3.
- **802.3at** (PoE+, 2009): up to 30W at the source, 25.5W at the device. Class 4 added.
- **802.3bt** (PoE++ / 4PPoE, 2018): up to 90W at the source, 71.3W at the device. Classes 5-8 added.

Class negotiation:
1. Switch detects a PoE-compatible device via a low-voltage probe (the "detection signature").
2. Switch performs classification: applies a voltage and measures current. The current draw maps to a class (0-8).
3. Switch allocates power for that class.
4. Switch enables PoE; phone boots.
5. Optional: LLDP-MED extends the negotiation, allowing the phone to request a different power level dynamically (e.g., during a video call).

Class budgets (typical):
- Class 0: 12.95W (default)
- Class 1: 3.84W (low-power devices)
- Class 2: 6.49W
- Class 3: 12.95W
- Class 4: 25.5W (PoE+ devices)
- Class 5-8: PoE++ classes, up to 71.3W.

The PoE-budget exhaustion failure mode: a switch has a fixed PoE power budget (e.g., 360W for a 24-port switch with class 3 average). If too many phones connect (or phones request high power), the switch may refuse to power additional devices. The phones simply don't boot; debugging requires checking switch PoE allocation.

Defensive design: ensure switch PoE budget exceeds expected device count × class; use PoE budgeting features (most enterprise switches support per-port priority); reserve power for critical devices.

## Provisioning Cycle Phases

The full provisioning cycle for a typical IP phone:

1. **Boot**: power-on; bootloader initializes; firmware loads; phone starts network stack.
2. **DHCP**: send DHCP DISCOVER, receive DHCP OFFER, send DHCP REQUEST, receive DHCP ACK. Phone now has IP, DNS, gateway, NTP, provisioning URL.
3. **Time sync (optional, before TLS)**: send NTP request to NTP server; sync clock.
4. **Fetch config URL**: phone constructs the URL using the DHCP-provided URL plus its MAC (e.g., `https://prov.example.com/{mac}.cfg`). HTTPS request to the URL.
5. **Fetch firmware (if config specifies)**: config file may direct phone to fetch a specific firmware version. Phone downloads firmware, validates signature, flashes, reboots.
6. **Re-boot (after firmware)**: phone restarts; goes through DHCP and time-sync again.
7. **Fetch per-MAC config**: now on the new firmware, phone fetches its full config (SIP credentials, dial plans, codec lists, feature buttons).
8. **Register**: phone sends SIP REGISTER to its registrar. Registrar challenges with 401, phone re-sends with credentials, registrar responds 200 OK. Phone is registered.
9. **Subscribe (optional)**: phone subscribes to BLF (busy-lamp-field) events, voicemail-message-waiting events, etc.
10. **Idle**: phone is ready for inbound/outbound calls.

Each phase has well-defined failure modes (covered next). The phone typically retries on failure with exponential backoff; some phones have a "factory reset" recovery if all retries fail.

## Common Provisioning Failure Trees

**DHCP Option missing → no provisioning URL**: phone boots with DHCP succeeding (it has IP, DNS, gateway) but no provisioning URL. Phone enters "manual config" mode or retries DHCP. Diagnosis: check DHCP server config; ensure vendor-specific option (66, 150, 160, 43) is set.

**DNS resolution fails → URL not reachable**: phone has provisioning URL but cannot resolve the hostname. Diagnosis: check DNS server reachability from phone's VLAN; check DNS server has the right zone records; check phone is using DHCP-provided DNS, not a hardcoded one.

**Cert expired/untrusted → TLS handshake fails**: TLS handshake fails because the provisioning server's cert is expired, self-signed, or signed by a CA the phone doesn't trust. Diagnosis: check cert validity; check phone's truststore; check cert chain reaches a trusted root.

**Time wrong → cert validation fails**: phone's clock is set to epoch (or a date before the cert's "not before"); cert appears not-yet-valid. Diagnosis: check NTP option in DHCP; check NTP server reachability; check phone's RTC battery (if applicable).

**HTTP 404 → wrong filename pattern**: phone constructs URL with wrong template (e.g., `{MAC}.cfg` vs `{mac}.cfg` vs `00:15:65:AA:BB:CC.xml`). Diagnosis: check vendor's expected file-naming pattern; check provisioning server's URL routing; confirm MAC format (uppercase vs lowercase, colons vs none).

**HTTP 401 → auth wrong**: phone sends auth (HTTP Basic or mTLS) and server rejects. Diagnosis: check phone's credentials in config; check server's auth configuration; for mTLS, check cert chain validity.

**Firmware mismatch → version too old/new for config schema**: phone has firmware version A but config file is in schema for firmware version B. Phone may parse partial, ignore unknown fields, or fail to register. Diagnosis: align firmware version and config schema; deploy firmware first, then config.

The diagnostic flow:
1. Capture phone-to-provisioning-server traffic (tcpdump on phone's switch port).
2. Look at TLS handshake: check cipher, cert, expiry, chain.
3. Look at HTTP status code: 200 means success, 4xx means client error, 5xx means server error.
4. Look at config file content: check syntax, schema, version compatibility.
5. Check phone's logs (most phones expose a syslog stream): boot messages, DHCP outcome, TLS errors.

## Vendor-Specific Schema Drift

Config file schemas evolve with firmware versions. Common schema drift patterns:

**New fields in newer firmware**: a new firmware version adds support for a feature (e.g., HD Voice for video). The config schema gains a new XML element. Old firmware ignores unknown elements; new firmware uses them.

**Deprecated fields**: a feature is removed. The schema marks the field deprecated. Old firmware uses the field; new firmware ignores or warns.

**Renamed fields**: a field is renamed for consistency. The schema may support both names for a transition period; eventually one is removed.

**Restructured sections**: the XML structure changes (e.g., feature configs move from `<feature>` to `<features><feature>...</feature></features>`).

The "deploy firmware then config" rule: when upgrading both firmware and config, deploy firmware first. The phone reboots into new firmware, then fetches the new-schema config. Reverse order risks the new-schema config being parsed by old firmware that doesn't understand new fields, leaving the phone in a partial state.

The "version-pinned config" pattern: the provisioning server serves different config files based on the phone's firmware version (sent in User-Agent or query string). Each firmware version gets a config tailored to its schema.

Vendor practices vary:
- **Polycom**: explicit schema version in config; phone parses based on version.
- **Yealink**: schema is firmware-version-implicit; deploy firmware first, then config.
- **Cisco**: rigorous version control; CUCM and phone firmware are paired and certified.

## Bulk Deployment Theory

Provisioning a single phone is straightforward; provisioning 100 or 1000 is operations-intensive. Bulk deployment requires:

1. **Pre-import MAC list**: collect all phone MACs (often from packing slips or vendor portal). Import to vendor's RPS or to local provisioning server.
2. **Per-MAC config generation**: a CI pipeline generates a config file for each MAC, populating per-device fields (extension, name, voicemail PIN, BLF buttons). Generated files go to the provisioning server.
3. **Reachability verification before shipping**: ensure DNS, DHCP, NTP, provisioning server are all reachable from the target VLAN. A pilot phone tests the bootstrap path end-to-end.
4. **Staged rollout**: deploy phones in batches (e.g., 10 per day) so any issues are caught before all phones are affected.
5. **Drift detection**: monitor phone registration status; alert on phones that fail to register within an expected window.

The CI pipeline for config generation is usually in a templating language (Jinja, Handlebars). Inputs:
- MAC list (CSV).
- Tenant config (SIP server, codec policy, feature flags).
- Per-user config (extension, name, voicemail).

Outputs:
- One config file per MAC.
- A manifest of (MAC, extension, location) for audit.

The shipping process:
- Phones ship from the warehouse with the vendor-default firmware.
- Customer plugs in the phone; the bootstrap chain (DHCP → vendor RPS → tenant provisioning server → SIP register) runs end-to-end.
- Phone is operational without IT intervention.

Failure modes for bulk deployments:
- A single misconfigured phone in the batch fails; logs reveal the issue; fix is applied to that one phone.
- A schema or DNS issue affects all phones; pilot deployment should catch this before bulk rollout.
- PoE budget exhaustion: too many phones on one switch; some don't power up. Mitigation: distribute across switches; size PoE budget appropriately.

## The Cisco CTL/ITL Architecture

Cisco's UC system uses Certificate Trust List (CTL) and Initial Trust List (ITL) to manage trust at scale. CTL is for cluster-wide trust (which CUCM nodes are authentic); ITL is for per-phone trust (which CUCM is the phone's owner).

The CTL contains the cluster's signing certs. It is signed by a "CTL client" (originally a USB-key-based device, now software). When the CTL is updated, all phones must accept the new CTL — typically via a manual confirmation or scheduled re-trust.

The ITL contains the phone's identity, the cluster's CallManager certs, and the cluster's TFTP cert. The ITL is signed by the cluster's TVS (Trust Verification Service). The phone validates the ITL signature using the embedded cluster public key.

The "ITL mismatch" error is the canonical "phone won't register" failure for Cisco UC. Causes:
- Cluster's certs were rotated, but the phone has the old ITL pinned.
- Phone was provisioned by one cluster, then moved to another without updating the ITL.
- The TFTP server returned an ITL with a different signing key than the phone expects.

Resolution: delete the phone's ITL (factory reset) and let it re-bootstrap with the new cluster's ITL. Or, distribute the new cluster's CTL to phones first, allowing the trust transition.

CTL vs ITL distinction:
- CTL: which CUCM cluster nodes are trusted (server identity).
- ITL: which CUCM cluster owns this phone (mutual identity).

In modern Cisco deployments (CUCM 12.5+), CTL/ITL is being replaced by SIP-OAuth (next section).

## SIP-OAuth

SIP-OAuth (Cisco-specific, introduced in CUCM 12.5) is an OAuth-token-based authentication for SIP REGISTER. It replaces the username/password and the LSC/MIC-based mTLS for phone-to-CUCM auth.

The flow:
1. Phone boots, fetches CUCM cert and OAuth issuer info.
2. Phone presents its LSC (or MIC) to the OAuth issuer (the cluster's Identity Service).
3. Identity Service validates the cert and issues an OAuth access token (short-lived, e.g., 1 hour).
4. Phone uses the access token in SIP REGISTER's Authorization header.
5. CUCM validates the token via the Identity Service.
6. Phone is registered.

Token revocation: the Identity Service maintains a revocation list. Tokens can be revoked instantly (e.g., when a phone is decommissioned). The phone fails to register on next refresh.

Advantages over CTL/ITL:
- Tokens are short-lived; compromise window is small.
- Revocation is centralized and instant.
- Tokens are JSON Web Tokens (JWT), industry-standard format.
- No per-phone certs to manage.

Disadvantages:
- Requires CUCM 12.5+ and compatible phone firmware.
- Adds dependency on the Identity Service (a new failure point).
- Token refresh logic must be reliable (or phones fail registration on token expiry).

## Threat Model

What provisioning protocols protect:
- **Tenant isolation**: each tenant's phones get only the tenant's config; cross-tenant info leakage is prevented.
- **Credential confidentiality**: SIP credentials, voicemail PINs, conference passwords are not exposed in transit (via TLS) or at rest (via encrypted config).
- **Configuration integrity**: signed configs prevent man-in-the-middle modification.

What provisioning protocols don't protect:
- **Physical attack on phone**: an attacker with physical access can extract the MIC private key (if no HSM), inspect the phone's storage, or replace the firmware.
- **Local network compromise**: an attacker on the voice VLAN (e.g., from a compromised printer that's on the same VLAN) can sniff TLS traffic if cert validation is weak, or perform ARP spoofing.
- **Vendor compromise**: if the vendor's RPS is breached, all customers' phones could be redirected to a rogue server.
- **Insider threats**: a tenant administrator with access to the provisioning server can extract all SIP credentials.

The threat model layering:
- **Layer 1 (Network)**: VLAN segregation, firewall rules, 802.1X.
- **Layer 2 (Protocol)**: TLS for transport, mTLS for mutual auth, signed firmware for integrity.
- **Layer 3 (Operational)**: encrypted configs at rest, audit logging, principle of least privilege for admin access.

Each layer addresses different threats; together they provide defense-in-depth.

## Mitigation: Network Segmentation

Voice VLAN segregation is the primary network-layer defense. The voice VLAN is a separate broadcast domain:
- Voice VLAN traffic does not reach data VLAN devices.
- Voice VLAN devices cannot initiate connections to arbitrary internet endpoints (firewall rules block).
- Voice VLAN devices cannot reach administrative interfaces of network equipment.

Firewall rules typical for voice VLAN:
- Allow: phones → SIP server (5060/5061), phones → provisioning server (HTTPS), phones → NTP, phones → DNS.
- Deny: phones → internet (except whitelisted vendor RPS).
- Deny: phones → other internal subnets (printers, workstations, servers).

The "no internet access for IoT" pattern extends to phones. Phones don't need general internet access; they need only specific service endpoints. Restricting to a whitelist eliminates large attack surfaces:
- Botnet recruitment (compromised phone joins a DDoS network) is impossible without outbound internet.
- Data exfiltration is bounded to whitelisted endpoints.
- Lateral movement to other internal systems is blocked.

VLAN segregation is operationally complex (requires switch configuration, DHCP server scopes per VLAN, firewall rule maintenance) but yields strong security. It is industry best practice for any deployment beyond a few phones.

## E911 / Kari's Law

E911 (Enhanced 911) is the regulatory requirement that emergency calls (911 in the US, 112 in EU, 999 in UK) deliver location info to the dispatcher. For PSTN landlines, location is implicit (the line is bound to a physical address). For VoIP, location must be explicitly conveyed.

**Kari's Law** (US, 2018) requires that businesses with multi-line phone systems allow direct-dialing 911 without requiring an outside-line prefix (e.g., dialing 9 first). Historically, many PBXes required "9-911" to dial out; Kari's Law mandates that "911" alone must work.

**RAY BAUM's Act §506** (US, 2020) extends Kari's Law: emergency calls must convey "dispatchable location" — a specific address, including building, floor, room — sufficient for first responders to find the caller.

The provisioning challenge: each phone must have its location pre-provisioned. The location may include:
- Civic location (street address, building, floor, room).
- Coordinate location (GPS lat/lon, less common for indoor phones).

Delivery mechanisms:
- **LLDP-MED Civic Location TLV**: switch advertises the location based on switch port. The phone uses this for the emergency call.
- **HELD (HTTP-Enabled Location Delivery, RFC 5985)**: phone queries an HTTP server for its location, identified by IP or other context.
- **Manual provisioning**: location is in the phone's config file, statically.

The "direct dial 911 without prefix" mandate requires the dial plan to recognize "911" (and 112, 999, etc.) regardless of preceding digits. The PBX's call routing must:
- Detect emergency calls.
- Route to PSAN (PSAP routing service).
- Convey location via SIP geolocation headers (RFC 6442) or out-of-band SIP fields.

Compliance failure consequences:
- FCC fines (up to $10,000 per violation in the US).
- Liability if a 911 call fails to deliver location and someone is harmed.
- Reputational damage and customer loss.

Best practice: validate every phone's location during provisioning; test the 911 path quarterly; maintain a per-phone location database with audit trails.

## Audit Logging

The provisioning server must log every fetch:
- Per-MAC + per-IP + timestamp + outcome (200/401/404/500).
- Cert presented (for mTLS).
- Config version served.
- File integrity hash (so the served file's authenticity can be verified later).

These logs feed into central SIEM (Security Information and Event Management) systems. Detection rules:
- **Repeated 401**: brute-force credential attempts.
- **Repeated 404 for unknown MACs**: phone scanning, possibly attackers probing for misconfigured phones.
- **Single MAC fetching from multiple IPs**: phone has been physically moved (legitimate) or MAC has been spoofed (attack).
- **Config fetched but no subsequent SIP REGISTER**: phone bricked or misconfigured; investigate.

Retention: provisioning logs should be retained for at least 1 year (regulatory minimum in many jurisdictions). For E911 compliance, location-relevant logs may need longer retention (7 years in some US states).

The audit log is also a forensic resource: when a phone is suspected of compromise, the log shows when it was provisioned, what config it received, and whether anomalies preceded the suspicion.

Log integrity: provisioning logs should be append-only (write-once-read-many storage) or signed (each log entry signed with a key separate from the provisioning server's). This prevents an attacker who compromises the provisioning server from rewriting history.

The combined operational picture — provisioning server logs, SIP REGISTER logs from the SIP server, network flow logs from the switches — gives a complete view of phone activity. Cross-correlation across these data sources reveals patterns no single source could.

## Phone Storage Hierarchy

A typical IP phone has multiple persistent storage tiers, each with different security properties:

**ROM / OTP (One-Time Programmable)**: the immutable bootloader. Contains:
- The first-stage boot code.
- The vendor's public key for firmware signature validation.
- The phone's serial number and MAC.
- Hardware-locked at manufacture; cannot be modified post-factory.

**Secure storage (HSM or TrustZone)**: tamper-resistant storage for:
- The MIC private key.
- Per-device keys for config decryption.
- The LSC private key (after enrollment).
- Anti-rollback counters (preventing firmware downgrade).

**Read/write flash**: the firmware image, the config file, the bootlog, the user-modifiable settings.

**RAM**: ephemeral state — call state, transient credentials, current TLS session.

The threat model layering by storage tier:
- Compromise of RAM: requires runtime attack (RCE); mitigated by memory protection.
- Compromise of flash: requires physical access or RCE escalation; mitigated by encrypted-at-rest config.
- Compromise of secure storage: requires hardware attack (probing, decapping); mitigated by tamper-detection.
- Compromise of ROM: impossible without re-fabrication of the chip.

## Boot Sequence Detail

The boot sequence is more complex than the high-level overview. A typical Polycom or Yealink phone:

1. **Power-on reset**: the SoC starts at a fixed reset vector.
2. **First-stage bootloader (ROM)**: validates the second-stage bootloader's signature using the vendor public key in ROM.
3. **Second-stage bootloader**: validates the firmware image's signature; loads firmware into RAM.
4. **Firmware initialization**: hardware drivers initialize (display, network, audio).
5. **Network discovery**: LLDP-MED frame is sent on the link; if a switch responds with Voice VLAN, the phone re-tags its network interface.
6. **DHCP DISCOVER**: on the (now-Voice) VLAN, the phone broadcasts DISCOVER.
7. **DHCP OFFER, REQUEST, ACK**: the phone obtains its IP and DHCP options.
8. **DNS resolution**: the phone resolves the provisioning server hostname.
9. **NTP sync**: if NTP option is provided, the phone syncs clock.
10. **TLS handshake**: connect to the provisioning server, validate cert.
11. **HTTPS GET**: fetch the config file (typically `<MAC>.cfg` or `<MAC>.xml`).
12. **Config parse**: parse the config file; validate schema.
13. **Firmware check**: if the config specifies a firmware version different from the running version, fetch firmware.
14. **Firmware install**: validate firmware signature, write to flash, reboot.
15. **Per-feature config fetch**: some phones fetch additional files (per-line config, per-button config, dial plan, ringtones).
16. **SIP registration**: send REGISTER to each configured line's registrar.
17. **Subscriptions**: SUBSCRIBE for BLF, voicemail, presence.
18. **Idle**: phone displays the home screen.

Each step has timeout and retry logic. A typical phone retries failed steps with exponential backoff (1s, 2s, 4s, 8s, ...) up to a maximum (e.g., 5 minutes between retries). Some steps have hard timeouts after which the phone displays an error screen and stops trying.

## DHCP Vendor-Class Identifier (Option 60)

DHCP option 60 is the Vendor-Class Identifier. The phone includes its vendor identifier (e.g., "PolycomVVX-VVX_400-UA/5.5.0.0000") in DHCP DISCOVER. The DHCP server can use option 60 to:
- Match the request against a vendor-specific scope.
- Return vendor-specific options in the OFFER.
- Apply per-vendor IP-address pools (e.g., voice phones get one subnet, data devices another).

The format: a string identifying vendor, model, and firmware. Examples:
- `PolycomVVX-VVX_400-UA/5.5.0.0000`
- `yealink T46G`
- `SnomD735/8.9.3.81`
- `Cisco CP-7841`
- `Grandstream GXP1625`

DHCP server configuration (ISC dhcpd):
```
class "voice-phones" {
    match if option vendor-class-identifier ~~ "Polycom" or option vendor-class-identifier ~~ "yealink";
    option tftp-server-name "https://prov.example.com/";
}

subnet 10.20.0.0 netmask 255.255.255.0 {
    pool {
        allow members of "voice-phones";
        range 10.20.0.10 10.20.0.200;
    }
}
```

Option 60 matching enables zero-configuration deployment: a network administrator sets up the DHCP server once, and any new phone with a recognized vendor-class is automatically routed to the correct provisioning URL.

## TFTP Legacy and the Move to HTTPS

Historically, IP phones used TFTP (Trivial File Transfer Protocol, RFC 1350) for provisioning. TFTP is:
- UDP-based (no TCP overhead).
- No authentication.
- No encryption.
- Simple to implement.

The TFTP-based provisioning model:
1. DHCP option 66 or 150 specifies TFTP server.
2. Phone fetches `<MAC>.cfg` via TFTP.
3. Phone applies config, registers.

The security problem: TFTP is plaintext. SIP credentials in the config are visible to anyone on the wire. A passive attacker captures every config file fetched.

The historical mitigation: deploy phones on a physically separate "voice network" with no internet access. This made eavesdropping require physical access to the voice cabling — a high bar.

The modern reality: voice and data networks are increasingly converged (VLAN-isolated but on the same physical switch). Physical segregation is rare. TFTP is dangerous in any modern environment.

The migration: most vendors now default to HTTPS provisioning. Polycom UC Software 5.x+, Yealink firmware 80+, Cisco MPP firmware all default to HTTPS. TFTP is supported only as a legacy fallback.

The DHCP option mapping:
- Option 66: a hostname or URL. If a URL with `https://`, HTTPS is used; if a bare hostname, TFTP is used.
- Option 150: a list of TFTP server IPs (Cisco-specific).
- Option 160: a URL (Polycom-specific).
- Option 43: vendor-specific suboptions; used for various vendor URL formats.

For HTTPS migration: change DHCP options to specify HTTPS URLs; ensure the provisioning server has a valid TLS cert; ensure phones are running firmware that supports HTTPS.

## Cert Pinning vs Cert Validation

Standard TLS validation: the phone validates the server's cert chain against the system truststore. Any cert signed by a trusted CA is accepted.

Cert pinning: the phone is configured to accept ONLY a specific cert (or a specific public key) for a specific server. Even if a CA in the truststore signs a different cert for that server, the phone rejects.

Cert pinning is stronger than validation:
- Defends against rogue CAs (a compromised CA cannot issue a cert that the phone will accept).
- Defends against MITM with valid certs (an attacker with a CA-signed cert for the wrong server cannot impersonate).

Cert pinning is operationally heavier:
- The pin must be updated when the server's cert is renewed.
- A pin mismatch causes a hard failure (no graceful degradation).

Common pinning strategies:
- **Pin the leaf cert**: tied to a specific cert; renews require coordinated update of pin and cert.
- **Pin the intermediate cert**: tied to the issuing CA; renews require only updating the leaf, not the pin.
- **Pin the public key**: tied to a key pair; allows cert renewal without changing the pin (as long as the same key pair is reused).

Public-key pinning (with key reuse across cert renewals) is the most operational. The cert can be renewed annually without changing the pin.

For phone provisioning, cert pinning is most relevant for the redirector (vendor RPS). Phones often have the vendor RPS's public key pinned in firmware, ensuring no other server can impersonate the RPS even with a valid CA-signed cert.

## XML Configuration Schema Examples

Polycom XML config snippet:
```xml
<polycomConfig>
  <reg.1.address>1234</reg.1.address>
  <reg.1.auth.userId>1234</reg.1.auth.userId>
  <reg.1.auth.password>secret</reg.1.auth.password>
  <reg.1.outboundProxy.address>sip.example.com</reg.1.outboundProxy.address>
  <reg.1.outboundProxy.port>5061</reg.1.outboundProxy.port>
  <reg.1.outboundProxy.transport>TLSv1.2</reg.1.outboundProxy.transport>
  <reg.1.label>Alice</reg.1.label>
  <reg.1.displayName>Alice Smith</reg.1.displayName>
  <voIpProt.SIP.regBackoffSec>10</voIpProt.SIP.regBackoffSec>
</polycomConfig>
```

Yealink XML config snippet:
```xml
<linelist>
  <line>
    <label>Alice</label>
    <displayname>Alice Smith</displayname>
    <username>1234</username>
    <password>secret</password>
    <register_name>1234</register_name>
    <sip_server_host>sip.example.com</sip_server_host>
    <sip_server_port>5061</sip_server_port>
    <transport>TLS</transport>
  </line>
</linelist>
```

Cisco MPP XML config snippet:
```xml
<flat-profile>
  <Display_Name_1_>Alice Smith</Display_Name_1_>
  <User_ID_1_>1234</User_ID_1_>
  <Password_1_>secret</Password_1_>
  <Auth_ID_1_>1234</Auth_ID_1_>
  <Proxy_1_>sip.example.com:5061</Proxy_1_>
  <SIP_Transport_1_>TLS</SIP_Transport_1_>
</flat-profile>
```

Each vendor has its own schema, but the core fields are similar: line label, display name, SIP credentials, registrar address, transport. Tools that generate configs typically have per-vendor templates and translate from a common internal model.

## SIP REGISTER Authentication Flow

After provisioning, the phone registers via SIP REGISTER. The flow:

1. Phone sends REGISTER:
```
REGISTER sip:sip.example.com SIP/2.0
Via: SIP/2.0/TLS 192.0.2.5:51234;branch=z9hG4bK-abc
From: <sip:1234@sip.example.com>;tag=foo
To: <sip:1234@sip.example.com>
Call-ID: register-call-id-123
CSeq: 1 REGISTER
Contact: <sip:1234@192.0.2.5:51234;transport=tls>
Expires: 3600
User-Agent: Polycom VVX 400 5.5.0.0000
Content-Length: 0
```

2. Server responds 401 Unauthorized:
```
SIP/2.0 401 Unauthorized
Via: SIP/2.0/TLS 192.0.2.5:51234;branch=z9hG4bK-abc
From: <sip:1234@sip.example.com>;tag=foo
To: <sip:1234@sip.example.com>;tag=server-tag
Call-ID: register-call-id-123
CSeq: 1 REGISTER
WWW-Authenticate: Digest realm="sip.example.com", nonce="abc123", algorithm=SHA-256, qop="auth"
Content-Length: 0
```

3. Phone re-sends REGISTER with credentials:
```
REGISTER sip:sip.example.com SIP/2.0
Via: SIP/2.0/TLS 192.0.2.5:51234;branch=z9hG4bK-def
From: <sip:1234@sip.example.com>;tag=foo
To: <sip:1234@sip.example.com>
Call-ID: register-call-id-123
CSeq: 2 REGISTER
Contact: <sip:1234@192.0.2.5:51234;transport=tls>
Expires: 3600
Authorization: Digest username="1234", realm="sip.example.com", nonce="abc123",
  uri="sip:sip.example.com", response="...", algorithm=SHA-256, qop=auth, nc=00000001, cnonce="..."
Content-Length: 0
```

4. Server responds 200 OK:
```
SIP/2.0 200 OK
Via: SIP/2.0/TLS 192.0.2.5:51234;branch=z9hG4bK-def
From: <sip:1234@sip.example.com>;tag=foo
To: <sip:1234@sip.example.com>;tag=server-tag
Call-ID: register-call-id-123
CSeq: 2 REGISTER
Contact: <sip:1234@192.0.2.5:51234;transport=tls>;expires=3600
Date: Thu, 25 Apr 2026 12:00:00 GMT
Content-Length: 0
```

The phone is now registered. The Contact binding tells the server "calls for sip:1234@sip.example.com should be routed to the Contact URI". The Expires=3600 tells the server the binding lasts 1 hour; the phone re-registers before then.

## Re-Registration Strategy

The phone must re-register before its registration expires, or the binding will be removed. Strategies:

**Conservative refresh**: re-register at 50% of the expiry interval. For Expires=3600s, re-register at 1800s (30 min). Provides a 30-minute buffer if the re-registration fails.

**Aggressive refresh**: re-register at 90% of the expiry. For Expires=3600s, re-register at 3240s (54 min). Less overhead, but small failure window.

**Adaptive refresh**: re-register at 50% normally; if a refresh fails, halve the interval and retry. Increases robustness in unstable network conditions.

The Expires value can be:
- **Set by phone**: phone proposes Expires; server can shorten (in 200 OK Contact;expires=) but not lengthen.
- **Set by server**: server's Min-Expires header (RFC 3261 §10.2.8) sets the minimum; phone must use at least this value.

Common production values: 600-3600 seconds (10 min - 1 hour). Shorter values produce more REGISTER traffic but detect failures faster; longer values reduce traffic but extend stale-binding windows.

## Provisioning Server High Availability

Production provisioning servers must be HA:

**Active-passive failover**: two servers, one active. If active fails, passive takes over. Failover is detected via VRRP (Virtual Router Redundancy Protocol) or via DNS health checks.

**Active-active load balancing**: multiple servers behind a load balancer. Each request goes to any healthy server. Configs must be replicated across servers (typically via shared filesystem or DB).

**Geo-redundant**: servers in multiple data centers. DNS-based routing (e.g., GeoDNS) directs phones to the nearest healthy server.

The phone's role in HA:
- Phones cache the most recent successful config; if the provisioning server is down, the phone uses the cached config.
- Phones retry with exponential backoff; if multiple servers are configured, phones round-robin or prioritize.
- DHCP option can list multiple provisioning URLs; phones try each in order.

The cache typically lives in the phone's persistent storage. On boot, the phone first tries to fetch fresh config; if that fails, it loads cached config and proceeds (registering with the cached SIP credentials).

## Telnet, SSH, Web UI Access

IP phones typically have a management interface. The threat: this interface, if exposed, allows attacker access to the phone's config (and credentials).

**Telnet**: plaintext, bad. Disabled by default in modern phones.
**SSH**: encrypted but still requires auth. Disabled by default; enabled only for diagnostics.
**Web UI (HTTP)**: plaintext, exposes the config password. Avoid.
**Web UI (HTTPS)**: encrypted; the standard management interface for most phones.

Best practice:
- Disable Telnet and unencrypted HTTP entirely.
- Enable HTTPS web UI on a non-standard port (e.g., 8443).
- Set the admin password via provisioning (not the default).
- Restrict web UI access to a management VLAN or specific IPs.

The phone's admin password is often set in the provisioning config:
```xml
<device.set>
  <device.auth.localAdminPassword>strong-password-here</device.auth.localAdminPassword>
</device.set>
```

A common provisioning failure: the admin password is the vendor default (admin/admin or similar), and an attacker on the voice VLAN can log in and extract SIP credentials.

## DECT-over-IP Provisioning (Yealink W-Series, Polycom VVX D-Series)

DECT (Digital Enhanced Cordless Telecommunications) is a wireless protocol for cordless phones. DECT phones connect to a "base station" (or "multicell") that handles SIP signaling.

DECT base stations are provisioned similarly to wired IP phones: DHCP option, HTTPS config fetch, signed firmware. The DECT handsets themselves don't connect to the SIP network directly; they connect to the base station via DECT, which proxies to SIP.

Per-handset config (e.g., extension, name, ringtones) is delivered to the base station, which pushes it to the handset over DECT. The base station's provisioning config includes per-handset sections.

DECT-specific considerations:
- DECT encryption (DSAA, DSC) protects the wireless link. Modern phones use DSAA2 or DECT-NG (next-generation cipher).
- Handset registration with the base station requires the user to enter a PIN. This is a one-time pairing; no per-handset SIP credentials.
- Multicell deployments (multiple base stations covering a campus) require synchronization between bases for handover.

## SBC (Session Border Controller) Interaction

An SBC sits between the IP phone and the SIP server, providing:
- NAT traversal (the SBC is the public-facing endpoint).
- Topology hiding (internal SIP infrastructure is not exposed).
- Media anchoring (RTP flows through the SBC for monitoring).
- Security (DDoS mitigation, fraud detection).

The phone is provisioned with the SBC as its registrar (rather than the actual SIP server). The SBC forwards REGISTER to the internal SIP server. The phone is unaware of the SBC's mediation.

SBC implications for provisioning:
- The phone's Outbound Proxy is the SBC.
- The SBC performs SIP-ALG functions (rewriting Via, Contact, SDP).
- TLS is typically terminated at the SBC; internal SIP can be UDP/TCP.

For multi-site deployments, multiple SBCs are common: one per region, with DNS-based routing or per-tenant assignment.

## Multi-Tenant Provisioning Considerations

In hosted PBX deployments, one provisioning server serves multiple tenants. Tenant isolation is critical:

**Per-tenant URLs**: each tenant has its own URL prefix (e.g., `https://prov.example.com/<tenant-id>/<MAC>.cfg`). The provisioning server validates the tenant-id against the requesting phone's identity (MAC, MIC).

**Per-tenant cert authorities**: each tenant has its own CA, issuing LSCs to that tenant's phones. Cross-tenant LSC use is rejected.

**Per-tenant config templates**: each tenant has its own config template, with tenant-specific defaults (codecs, dial plans, branding).

**Tenant data segregation**: at-rest config files are stored in per-tenant directories with restrictive permissions. Database queries filter by tenant.

**Audit**: per-tenant audit logs; tenant administrators can see only their own tenant's logs.

The threat: a phone provisioning request with a forged tenant-id (e.g., tenant A's phone tries to fetch tenant B's config). Mitigations: validate the (tenant, MAC) pairing against a database; require the phone's MIC to match a tenant binding.

## References

- **RFC 2131** — Dynamic Host Configuration Protocol (DHCP) (https://www.rfc-editor.org/rfc/rfc2131.html)
- **RFC 2132** — DHCP Options and BOOTP Vendor Extensions
- **RFC 5246** — TLS 1.2 (obsoleted by RFC 8446)
- **RFC 5280** — Internet X.509 Public Key Infrastructure Certificate
- **RFC 5985** — HTTP-Enabled Location Delivery (HELD)
- **RFC 6442** — Location Conveyance for SIP
- **RFC 7616** — HTTP Digest Access Authentication
- **RFC 8446** — TLS 1.3
- **RFC 8915** — Network Time Security for NTP
- **ANSI/TIA-1057** — LLDP-MED specification
- **IEEE 802.3af / 802.3at / 802.3bt** — Power-over-Ethernet standards
- **47 CFR §9.10** — Kari's Law and RAY BAUM's Act regulations
- **NIST SP 800-52** — TLS Implementation Guidelines
- **NIST SP 800-90A** — Random number generation guidelines (for nonce generation in cert validation)
- **Polycom Provisioning Guide** — Polycom UC Software Administrator's Guide (current edition)
- **Yealink IP Phone Provisioning Guide** — Yealink documentation
- **Cisco Unified Communications Manager Security Guide** — Cisco Press, current edition
- **Cisco MPP Multiplatform Phone Administration Guide**
- **Snom Provisioning Guide** — Snom documentation
- **Grandstream Configuration Tool Manual** — Grandstream documentation
- ITU-T E.164 — Numbering plan for the PSTN
- ATIS-1000074 — SHAKEN (relevant for caller-ID assertion in modern provisioning)
- man pages: `dhcpcd(8)`, `dhclient(8)`, `tftpd(8)`, `tftp(1)`, `tcpdump(1)`, `openssl-s_client(1)`
