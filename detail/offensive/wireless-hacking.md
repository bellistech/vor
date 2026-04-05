# Wireless Hacking -- Deep Dive

> For authorized security testing, red team exercises, and educational study only. This document provides in-depth technical analysis of wireless security mechanisms and their weaknesses, aligned with CEH v13 Module 16.

---

## Prerequisites

- Understanding of networking fundamentals (OSI model layers 1-2)
- Familiarity with Linux command line and wireless tools (aircrack-ng suite)
- Basic knowledge of cryptographic concepts (symmetric/asymmetric encryption, hashing)
- Wireless adapter capable of monitor mode and packet injection
- Suggested prior reading: `sheets/offensive/wireless-hacking.md` for command quick-reference

---

## 1. WPA2 4-Way Handshake Protocol Analysis

The WPA2 4-way handshake occurs after a client (supplicant) and access point (authenticator) have completed open system authentication and association. Its purpose is to mutually prove knowledge of the Pairwise Master Key (PMK) without transmitting it, and to derive fresh session keys.

### Key Hierarchy

The IEEE 802.11i key hierarchy derives all session keys from a root secret:

```
PSK or 802.1X MSK
       |
       v
PMK (Pairwise Master Key) -- 256 bits
       |
       v
PTK (Pairwise Transient Key) -- derived via PRF
       |
       +---> KCK (Key Confirmation Key)     -- 128 bits, EAPOL MIC
       +---> KEK (Key Encryption Key)       -- 128 bits, encrypts GTK delivery
       +---> TK  (Temporal Key)             -- 128 bits, data encryption (AES-CCMP)
```

### The Four Messages

**Message 1 (AP to Client):**
- AP generates a random ANonce (Authenticator Nonce)
- Sends ANonce in plaintext EAPOL-Key frame
- No MIC (client cannot verify yet)

**Message 2 (Client to AP):**
- Client generates its own SNonce (Supplicant Nonce)
- Client now has all inputs to derive PTK:
  - `PTK = PRF-X(PMK, "Pairwise key expansion", Min(AA,SPA) || Max(AA,SPA) || Min(ANonce,SNonce) || Max(ANonce,SNonce))`
  - Where AA = Authenticator MAC, SPA = Supplicant MAC
- Sends SNonce + MIC (computed with KCK portion of PTK)
- Includes RSN Information Element (cipher suite capabilities)

**Message 3 (AP to Client):**
- AP derives PTK independently using same inputs
- Verifies MIC from Message 2 (proves client knows PMK)
- Sends encrypted GTK (Group Temporal Key) using KEK
- Includes MIC for integrity
- Includes RSN IE from Beacon (client compares to detect downgrade)

**Message 4 (Client to AP):**
- Client sends acknowledgment with MIC
- AP installs PTK after receiving this message
- Client installs PTK after sending this message

### Why Capturing the Handshake Enables Offline Attack

An attacker who captures all four EAPOL frames has:
- ANonce and SNonce (transmitted in cleartext)
- AA and SPA (MAC addresses, visible in frame headers)
- MIC value from Message 2 (or Message 3)

The attacker can then:
1. Guess a passphrase
2. Derive PMK = PBKDF2(HMAC-SHA1, passphrase, SSID, 4096, 256)
3. Derive PTK from PMK + nonces + MACs
4. Compute MIC using KCK from derived PTK
5. Compare computed MIC to captured MIC
6. Match means correct passphrase

### KRACK (Key Reinstallation Attack)

CVE-2017-13077 through CVE-2017-13082. Discovered by Mathy Vanhoef in 2017.

The vulnerability exploits the retransmission of Message 3. If an attacker replays Message 3, the client reinstalls the PTK and resets the nonce counter. With CCMP (AES-CTR), nonce reuse allows XOR of two ciphertexts to recover plaintext. With TKIP, nonce reuse allows MIC key recovery and packet injection.

The attack requires a man-in-the-middle position (channel-based MitM) to block Message 4 and trigger retransmission.

---

## 2. PMKID Attack Mathematics

Discovered by Jens "atom" Steube (hashcat author) in 2018. This attack targets the RSN PMKID present in the first EAPOL message from the AP, requiring no connected clients and no completed handshake.

### PMKID Derivation

```
PMK = PBKDF2(HMAC-SHA1, Passphrase, SSID, 4096, 256)

PMKID = HMAC-SHA1-128(PMK, "PMK Name" || AA || SPA)
```

Where:
- `PMK` = Pairwise Master Key (derived from passphrase and SSID)
- `"PMK Name"` = literal ASCII string
- `AA` = Authenticator (AP) MAC address
- `SPA` = Supplicant (Client) MAC address
- Result is truncated to 128 bits

### PMK Derivation via PBKDF2

The computational bottleneck is PBKDF2 with 4096 iterations of HMAC-SHA1:

```
PMK = PBKDF2(HMAC-SHA1, passphrase, SSID, 4096, 256)

Internally:
  U1 = HMAC-SHA1(passphrase, SSID || INT(1))
  U2 = HMAC-SHA1(passphrase, U1)
  ...
  U4096 = HMAC-SHA1(passphrase, U4095)
  Block1 = U1 XOR U2 XOR ... XOR U4096

  (Second block for bits 161-256)
  V1 = HMAC-SHA1(passphrase, SSID || INT(2))
  ...
  Block2 = V1 XOR V2 XOR ... XOR V4096

  PMK = Block1[0:160] || Block2[0:96]    (256 bits total)
```

Each passphrase guess requires 2 x 4096 = 8192 HMAC-SHA1 operations. This is the primary performance constraint and why GPU acceleration (hashcat) dramatically outperforms CPU cracking.

### Why PMKID Is Easier to Obtain

- The PMKID is included in the first EAPOL-Key message (robust security network PMKID key data)
- No client needs to be connected or deauthenticated
- A single association request to the AP is sufficient
- Works against WPA2-PSK networks with PMKID enabled (most Broadcom/Intel APs)

### Hashcat Attack

```bash
# Mode 22000 (unified WPA format, supports both handshake and PMKID)
hashcat -m 22000 -a 0 pmkid_hash wordlist.txt

# With rules
hashcat -m 22000 -a 0 pmkid_hash wordlist.txt -r rules/best64.rule

# Benchmark: modern GPU achieves ~500,000-1,000,000 PMKs/sec
```

---

## 3. SAE/Dragonfly Protocol and Dragonblood Vulnerabilities

WPA3-Personal replaces PSK with Simultaneous Authentication of Equals (SAE), based on the Dragonfly key exchange (RFC 7664). SAE provides forward secrecy and resistance to offline dictionary attacks.

### Dragonfly Key Exchange

SAE operates in two phases:

**Commit Phase:**
1. Both parties derive a Password Element (PE) -- a point on an elliptic curve derived from the password and both MAC addresses:
   - `PE = HashToElement(password, MAC_A, MAC_B)`
   - Uses a hunt-and-peck or hash-to-curve algorithm to map password to a curve point
2. Each party selects random values (private scalar `r` and mask `m`):
   - `scalar = (r + m) mod q`
   - `Element = -m * PE` (inverse of mask times PE)
3. Exchange `scalar` and `Element`

**Confirm Phase:**
1. Both parties compute the shared secret:
   - `ss = r * (scalar_peer * PE + Element_peer)`
   - This equals `r * r_peer * PE` if both are honest
2. Derive PMK from shared secret
3. Exchange confirm messages (HMAC proof of shared key)

### Security Properties

- **Forward secrecy:** Fresh random values per exchange; compromising the password later does not reveal past sessions
- **Offline dictionary resistance:** An attacker observing the exchange cannot test password guesses offline (unlike WPA2 handshake)
- **Mutual authentication:** Both parties prove knowledge of the password simultaneously

### Dragonblood Vulnerabilities (2019)

Discovered by Mathy Vanhoef and Eyal Ronen. Multiple classes of attacks:

**Timing Side-Channel (CVE-2019-9494):**
The hunt-and-peck algorithm to convert password to curve point iterates until a valid point is found. The number of iterations depends on the password, and implementations that do not run a constant number of iterations leak timing information. An attacker measuring SAE commit processing time can eliminate password candidates, reducing the search space to a practical brute-force.

**Cache-Based Side-Channel (CVE-2019-9494):**
The branch pattern in hash-to-element depends on the password. On shared hardware (VMs, cloud), cache-timing attacks (Flush+Reload, Prime+Probe) reveal which branches were taken, leaking the password partition.

**Denial of Service (CVE-2019-9496):**
SAE commit processing requires expensive elliptic curve operations. An attacker sending forged commit frames forces the AP to perform these computations, exhausting CPU. Anti-clogging tokens (SAE's built-in DoS protection) can be bypassed.

**Downgrade to WPA2 (Transition Mode):**
WPA3 transition mode allows both WPA2 and WPA3 clients. An attacker can:
1. Set up a rogue AP advertising only WPA2
2. Block client probe responses from the real AP
3. Client falls back to WPA2 and performs vulnerable 4-way handshake
4. Attacker captures handshake for offline cracking

**Group Downgrade:**
SAE supports multiple elliptic curve groups. An attacker can forge commit frames requesting a weaker group (if the AP supports multiple groups), then exploit timing differences in the weaker group.

### Mitigations

- Hash-to-curve (RFC 9380) replaces hunt-and-peck, eliminating timing side-channels
- Constant-time implementations prevent cache attacks
- WPA3-only mode (no transition) prevents downgrade
- Anti-clogging token improvements limit DoS impact
- Implementations should support only strong groups (P-256, P-384)

---

## 4. RF Signal Propagation and Antenna Theory

Understanding radio frequency behavior is essential for wireless security assessments: signal coverage determines attack range, and antenna selection affects monitoring capability.

### Signal Propagation

**Free-Space Path Loss (FSPL):**
```
FSPL (dB) = 20 * log10(d) + 20 * log10(f) + 32.44

Where:
  d = distance in km
  f = frequency in MHz
```

At 2.4 GHz, signal loses approximately 6 dB per doubling of distance (inverse square law). Indoor propagation adds:
- Drywall: 3-5 dB attenuation
- Concrete wall: 10-15 dB attenuation
- Metal: 15-20+ dB attenuation
- Glass: 2-3 dB attenuation

**Multipath:** Signals reflect off surfaces, causing constructive/destructive interference. 802.11n/ac/ax MIMO exploits multipath to increase throughput.

**Fresnel Zone:** The ellipsoidal region around line-of-sight that must be clear for reliable communication. At 2.4 GHz over 100m, the first Fresnel zone radius is approximately 1.77m.

### Antenna Fundamentals

**dBi (decibels relative to isotropic):** Gain measurement comparing antenna to a theoretical isotropic (omnidirectional) radiator.

**dBm (decibels relative to milliwatt):** Absolute power measurement.

```
Common reference points:
  0 dBm   = 1 mW
  10 dBm  = 10 mW
  20 dBm  = 100 mW (typical WiFi AP max)
  27 dBm  = 500 mW (high-power AP)
  30 dBm  = 1 W

Rule of 3s and 10s:
  +3 dB = double the power
  +10 dB = 10x the power
  -3 dB = half the power
  -10 dB = 1/10 the power
```

**EIRP (Effective Isotropic Radiated Power):**
```
EIRP (dBm) = Transmit Power (dBm) + Antenna Gain (dBi) - Cable Loss (dB)

Example:
  20 dBm transmitter + 9 dBi antenna - 2 dB cable loss = 27 dBm EIRP
  Regulatory limits: FCC Part 15 allows 36 dBm EIRP (point-to-point) on 5 GHz
```

### Antenna Types for Wireless Testing

| Type | Gain | Beamwidth | Use Case |
|------|------|-----------|----------|
| Omnidirectional (dipole) | 2-9 dBi | 360 horizontal | General scanning, broad area monitoring |
| Directional (Yagi) | 10-18 dBi | 30-60 degrees | Long-range targeted capture |
| Parabolic dish | 18-30+ dBi | 5-15 degrees | Extreme range, point-to-point |
| Panel/patch | 8-14 dBi | 60-120 degrees | Sector coverage, indoor |

**Practical implications for penetration testing:**
- A 15 dBi Yagi can capture handshakes from >300m away
- Directional antennas can reach networks beyond the building perimeter
- Higher gain = narrower beam = requires aiming, but extends range significantly
- Receiver sensitivity matters as much as transmit power (typical: -70 to -90 dBm usable)

### Link Budget

```
Received Power = Tx Power + Tx Antenna Gain - Path Loss + Rx Antenna Gain - Cable Losses

Example (2.4 GHz, 100m, line-of-sight):
  FSPL = 20*log10(0.1) + 20*log10(2400) + 32.44 = -20 + 67.6 + 32.44 = 80 dB
  Rx Power = 20 + 5 - 80 + 9 - 1 = -47 dBm (excellent signal)
```

---

## 5. 802.1X/EAP Authentication Flow

802.1X provides port-based network access control and is the foundation of enterprise wireless security. It uses the Extensible Authentication Protocol (EAP) framework to support multiple authentication methods.

### Architecture (Three Parties)

```
Supplicant (Client)  <-->  Authenticator (AP/Switch)  <-->  Authentication Server (RADIUS)
      |                          |                              |
  EAP methods              EAP relay                     EAP processing
  802.1X state             802.1X port control           User database
  Certificate store        RADIUS client                 Certificate authority
```

The authenticator acts as a pass-through: it encapsulates EAP frames from the supplicant in RADIUS packets and forwards them to the authentication server.

### Authentication Flow

```
Supplicant                Authenticator               RADIUS Server
    |                          |                          |
    |--- EAPOL-Start --------->|                          |
    |                          |                          |
    |<-- EAP-Request/Identity -|                          |
    |                          |                          |
    |--- EAP-Response/Identity>|                          |
    |                          |--- Access-Request ------>|
    |                          |    (EAP-Response inside)  |
    |                          |                          |
    |                          |<-- Access-Challenge -----|
    |<-- EAP-Request (method) -|    (EAP-Request inside)  |
    |                          |                          |
    |  [Method-specific exchange: TLS handshake, etc.]    |
    |                          |                          |
    |                          |<-- Access-Accept --------|
    |<-- EAP-Success ----------|    (MSK included)        |
    |                          |                          |
    |--- EAPOL-Key (4-way) --->|  [WPA2 key derivation]  |
    |                          |                          |
    [Port authorized, traffic flows]
```

### EAP-TLS (Most Secure)

Both client and server present X.509 certificates for mutual authentication:

1. Server sends its certificate; client validates against trusted CA
2. Client sends its certificate; server validates against trusted CA or RADIUS attribute
3. TLS session established; MSK (Master Session Key) derived from TLS keying material
4. No password transmitted at any point

**Strengths:** Immune to credential theft, evil twin attacks (if client validates server cert), dictionary attacks. Provides per-session encryption keys.

**Weakness:** Certificate management complexity. Every client needs a unique certificate. Revocation (CRL/OCSP) must be maintained.

### PEAP (Protected EAP)

Server-side certificate only; inner authentication uses MSCHAPv2 (username/password):

1. TLS tunnel established using server certificate (client should validate)
2. Inside tunnel: EAP-MSCHAPv2 exchange (challenge-response based on password hash)
3. MSK derived from outer TLS and inner authentication

**Attack vector (evil twin):**
```
1. Attacker creates rogue AP with same SSID
2. Runs hostapd-mana as RADIUS server (accepts any credentials)
3. Client connects, skips server certificate validation (common misconfiguration)
4. Client performs MSCHAPv2 inside attacker's TLS tunnel
5. Attacker captures MSCHAPv2 challenge/response
6. Crack offline: NTHash can be recovered from MSCHAPv2 in ~24 hours (DES weakness)
```

```bash
# Cracking captured MSCHAPv2
# chapcrack decomposes MSCHAPv2 to DES (crack.sh service or hashcat)
chapcrack parse -C <challenge> -R <response>
hashcat -m 14000 des_hashes  # Brute-force 56-bit DES keys

# Or use asleap for dictionary attack
asleap -C <challenge> -R <response> -W wordlist.txt
```

### EAP-TTLS (Tunneled TLS)

Similar to PEAP but supports more inner authentication methods (PAP, CHAP, MSCHAPv2, EAP). Inner PAP sends password in cleartext inside the TLS tunnel, so server certificate validation is critical.

### Defense Recommendations for Enterprise Wireless

1. **EAP-TLS preferred:** Eliminates password-based attacks entirely
2. **Certificate pinning:** Configure supplicants to accept only the specific RADIUS server certificate, not just any certificate from a trusted CA
3. **Disable inner method fallback:** Do not allow PAP or other weak inner methods
4. **RADIUS server hardening:** Use strong shared secrets, enable accounting, log authentication events
5. **Client configuration profiles:** Deploy via MDM with certificate and server validation pre-configured; prevent users from accepting rogue certificates
6. **Network segmentation post-auth:** Use RADIUS VLAN assignment to place authenticated users in appropriate network segments based on role
