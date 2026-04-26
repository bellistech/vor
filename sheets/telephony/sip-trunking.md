# SIP Trunking

PBX-to-ITSP signaling for PSTN connectivity over IP. Replaces PRI/T1.

## Setup

A SIP trunk is a virtual telephony "line" between your PBX and an ITSP (Internet Telephony Service Provider) carried over IP using the SIP signaling protocol (RFC 3261) plus RTP for media. It's the cloud-era replacement for legacy TDM connectivity:

- **Old world:** PRI (Primary Rate Interface, 23B+D = 23 voice channels + 1 signaling) on a T1 line, or BRI (2B+D) over ISDN — purchased from an ILEC, terminated on a PRI gateway, $300-1000/month per span.
- **New world:** SIP trunk over your existing internet uplink, terminated directly on the IP-PBX (Asterisk/FreeSWITCH/3CX/Cisco CUBE/Avaya SBCE), $5-25/month per channel + per-minute rates, instantly scalable.

A trunk carries N "channels" (concurrent call sessions). 20 channels = 20 simultaneous calls; the 21st caller gets busy. Bursting (above ratecard for an extra fee) is sometimes available.

**Cost structure** has three pieces:
- **Per-channel monthly fee** ($1-5/channel/month) — capacity reservation.
- **DID monthly fee** ($0.50-1/DID/month) — per inbound number.
- **Per-minute rate** — outbound (typically $0.005-0.015/min US/CA, much higher international) and sometimes inbound (often free for US/CA local, paid for toll-free).

Some ITSPs offer **flat-rate plans** (unlimited US local for $25-40/channel/month) — economical above ~3000 minutes/channel/month.

```
+------------+        SIP/UDP 5060               +-----------+
|   IP-PBX   |  <----- INVITE/200/ACK ---->      |   ITSP    |---> PSTN
|  Asterisk  |  <----- RTP audio --------->      |   SBC     |
+------------+                                   +-----------+
```

Why SIP trunks beat PRI: instant scaling (no T1 cross-connect lead time), lower cost, geographic redundancy (route to multiple POPs), unified comms (SMS/MMS/video over the same account), no physical hardware (no PRI gateway to depreciate), encryption (TLS+SRTP) available.

Why PRI still exists: legacy systems, regulatory in some markets, hard-real-time guarantees, no jitter from internet pathing.

## Authentication Models

Three flavors, in order of increasing security:

### IP-Based Authentication (ACL / whitelist)

The ITSP whitelists the public IP of your PBX. No credentials. Anyone reaching the ITSP edge from that source IP is presumed to be your trunk.

- **Pros:** Simple. No registration. Works behind static-NAT (so long as outbound source IP is fixed).
- **Cons:** Requires static public IP. Anyone who spoofs/hijacks your IP can ride your trunk. ACL on ITSP side and on your PBX (`permit/deny` lines in Asterisk) is mandatory.
- **Configuration:** ITSP gives you `sip.itsp.com:5060` as their endpoint, and you tell them your public IP `203.0.113.5`. INVITEs go from your IP straight to theirs without REGISTER.

### Credential-Based Registration

The PBX sends `REGISTER` to the ITSP every N seconds (typically 60-3600s) with a username + password. Once REGISTER 200 OK is returned, the trunk is "up". Inbound calls follow the registration binding back to your PBX.

- **Pros:** Works with dynamic IP (residential broadband). No coordination of IP changes.
- **Cons:** Credentials can leak. Subject to brute-force (mitigate via fail2ban + rate-limit).
- **Auth:** SIP Digest authentication (MD5 challenge-response on REGISTER and on INVITE if `auth-int` is required).

### TLS + Mutual Certificate Authentication

`sips:` URIs over TLS on port 5061 (signaling) and SRTP for media. Optional client certificate authenticates the PBX cryptographically, eliminating password leak risk.

- **Pros:** Strongest security. Encryption end-to-end (PBX↔ITSP). Mandatory for HIPAA / PCI-adjacent voice flows.
- **Cons:** Cert provisioning + rotation. CPU cost (handshake + AES on every packet). Some ITSPs charge premium for TLS trunks.
- **Configuration in Asterisk:** `transport=tls`, `cert_file=/etc/asterisk/keys/pbx.pem`, `ca_list_file=/etc/ssl/certs/ca-certificates.crt`, `verify_server=yes`.

## ITSP Catalog

Major providers and their distinguishing features:

| ITSP | Strength | Notes |
|------|----------|-------|
| **Twilio Elastic SIP Trunking** | Programmable, global, robust API | Per-minute pricing, no monthly minimums. Strong dev experience. |
| **Vonage Business / Vonage API** | Global reach, UCaaS | Formerly Nexmo. Has both SIP trunking and Communications API. |
| **Bandwidth.com** | Direct-to-CLEC, US carrier-grade | Owns last-mile, used by many cloud comms providers as upstream. |
| **SignalWire** | OpenSIPS/FreeSWITCH founders | Cheap, programmable, LAML/cXML compatible with Twilio. |
| **Telnyx** | Programmable, low-cost, MPLS backbone | Strong portal, granular routing controls, public PoP IPs. |
| **Flowroute** | Dev-focused, US/Canada | HD voice support, RFC 4733 DTMF. Now part of Sinch. |
| **VoIP.ms** | Hobbyist / SMB friendly | Per-minute pricing, very cheap. Manual portal. |
| **Skyetel** | Low-cost wholesale, US | Aimed at MSPs / resellers. |
| **Anveo** | International origination/termination | Cheap international, less polished UI. |
| **BroadVoice** | Hosted PBX + trunking | Bundled UCaaS. |
| **Plivo** | Programmable voice + SMS | Twilio-style API, lower cost. |
| **RingCentral** | UCaaS, hosted PBX | Less common as raw SIP trunking; mostly turnkey. |

**Consumer vs business tier:** Consumer VoIP (Vonage Home, MagicJack, Ooma) gives you a single phone number and an ATA box; you can't terminate a SIP trunk on a PBX. Business SIP trunking (the providers above) gives you an ITSP account with N channels, M DIDs, and SIP credentials/whitelisting. Pricing is often 5-10x more expensive on the consumer tier per equivalent feature.

```
Consumer VoIP:    1 phone, 1 number, $20-40/month, RJ11 ATA out
Business SIP:     N channels, M DIDs, $1-5/ch + DID + per-minute, sip:trunk@itsp.com
```

## E.164 Format

ITU-T E.164 defines the international public telecommunication numbering plan. Format:

```
+CC NSN
```

- `+` — international prefix indicator (literal `+` character).
- `CC` — country code (1-3 digits): `1` = NANP, `7` = Russia/Kazakhstan, `33` = France, `44` = UK, `49` = Germany, `81` = Japan, `86` = China, `91` = India, etc.
- `NSN` — national significant number (up to 12 digits, typically 7-10).

Examples:
```
+14155551212        US, San Francisco
+442071838750       UK, London
+33142868200        France, Paris
+861012345678       China, Beijing
+819012345678       Japan, mobile
+919876543210       India, mobile
```

**Always normalize to E.164 internally.** Every dialed number, every CLID, every DID, every CDR record stores the canonical `+CC...` form. Only at the final user-facing display layer do you reformat to local conventions (`(415) 555-1212` for US, `020 7183 8750` for UK).

Why? Because PSTN routing and ITSP APIs uniformly accept E.164. Storing `415-555-1212` vs `(415) 555-1212` vs `4155551212` creates a normalization nightmare. Pick `+14155551212` and stick to it.

```python
# E.164 normalization (Python, libphonenumber)
import phonenumbers
n = phonenumbers.parse("(415) 555-1212", "US")
canonical = phonenumbers.format_number(n, phonenumbers.PhoneNumberFormat.E164)
# canonical == "+14155551212"
```

Rules:
- No spaces, no hyphens, no parentheses.
- Always include `+` and country code, even for domestic calls.
- Maximum 15 digits after the `+` (E.164 spec).

## NANP

The North American Numbering Plan covers the US, Canada, and 23 Caribbean countries. Country code `+1`. Format: 10 digits as **NXX-NXX-XXXX**.

```
+1 NPA NXX XXXX
   |   |   |
   |   |   +-- Subscriber (4 digits)
   |   +------ Exchange / Central Office (3 digits, NXX where N=2-9, X=0-9)
   +---------- Area Code (NPA = Numbering Plan Area, 3 digits, N=2-9)
```

`N` = digit 2-9. `X` = digit 0-9. So area codes never start with 0 or 1, and exchanges never start with 0 or 1 either.

**Toll-free codes** (caller pays nothing; dialed party pays):
- 800, 833, 844, 855, 866, 877, 888 (in roll-out order, 1967 onward).
- 822, 880, 881, 882, 887, 889 reserved for future use.

**Special area codes:**
- N11 services: 211 (community), 311 (non-emergency municipal), 411 (directory), 511 (traffic), 611 (carrier repair), 711 (TRS for hearing-impaired), 811 (utilities locator), 911 (emergency).
- 555-01XX = fictional / for-use-in-fiction (movies, TV).
- 950, 958, 976 = legacy carrier-access / pay-per-call.
- 900 = pay-per-call premium (used by adult/psychic services historically; heavily regulated since the 90s).

**Geographic vs non-geographic NPAs:** Most NPAs are tied to a region (415 = San Francisco). Some are non-geographic (500-series for personal communications, 700 for inter-exchange carriers).

**Overlay codes:** When an NPA exhausts, an overlay is added covering the same geography (e.g., NYC: 212/646/332). Forces 10-digit dialing inside the metro.

## International Dialing

ITU country codes (Recommendation E.164, regularly updated). Selected list:

| CC | Country |
|----|---------|
| 1 | NANP (US/CA/Caribbean) |
| 7 | Russia, Kazakhstan |
| 20 | Egypt |
| 27 | South Africa |
| 30 | Greece |
| 31 | Netherlands |
| 32 | Belgium |
| 33 | France |
| 34 | Spain |
| 36 | Hungary |
| 39 | Italy |
| 40 | Romania |
| 41 | Switzerland |
| 43 | Austria |
| 44 | UK |
| 45 | Denmark |
| 46 | Sweden |
| 47 | Norway |
| 48 | Poland |
| 49 | Germany |
| 51 | Peru |
| 52 | Mexico |
| 53 | Cuba |
| 54 | Argentina |
| 55 | Brazil |
| 56 | Chile |
| 57 | Colombia |
| 58 | Venezuela |
| 60 | Malaysia |
| 61 | Australia |
| 62 | Indonesia |
| 63 | Philippines |
| 64 | New Zealand |
| 65 | Singapore |
| 66 | Thailand |
| 81 | Japan |
| 82 | South Korea |
| 84 | Vietnam |
| 86 | China |
| 90 | Turkey |
| 91 | India |
| 92 | Pakistan |
| 93 | Afghanistan |
| 94 | Sri Lanka |
| 95 | Myanmar |
| 98 | Iran |
| 211 | South Sudan |
| 212 | Morocco |
| 213 | Algeria |
| 220 | Gambia |
| 234 | Nigeria |
| 254 | Kenya |
| 255 | Tanzania |
| 256 | Uganda |
| 351 | Portugal |
| 352 | Luxembourg |
| 353 | Ireland |
| 354 | Iceland |
| 358 | Finland |
| 380 | Ukraine |
| 420 | Czech Republic |
| 421 | Slovakia |
| 852 | Hong Kong |
| 853 | Macau |
| 855 | Cambodia |
| 880 | Bangladesh |
| 886 | Taiwan |
| 962 | Jordan |
| 965 | Kuwait |
| 966 | Saudi Arabia |
| 971 | UAE |
| 972 | Israel |

**National trunk prefix vs international prefix:**
- The **trunk prefix** is what locals dial within their own country before the area code (e.g., UK `0`, France `0`, Japan `0`, China `0`). When you dial the same number from outside, you DROP the trunk prefix.
- Example: London number `020 7183 8750` from inside UK; from outside, dial `+44 20 7183 8750` (drop the leading 0).
- The **international exit code** is what you dial to leave your country: NANP = `011`, most of Europe/Africa/most-of-world = `00`, Australia = `0011`, Japan/Korea = `010`, Mexico = `00`.

So a US person calls Paris by dialing `011 33 1 42 86 82 00`; a French person calls SF by dialing `00 1 415 555 1212`. The E.164 form `+33 1 42 86 82 00` is universal; the local exit-code form is regional.

PBX dialplans should:
1. Strip the user's exit code (`011` or `9011` etc.).
2. Prepend `+`.
3. Send to ITSP as `+CC...`.

## DID — Direct Inward Dial

A DID is a phone number that the ITSP routes directly to one of your trunk channels, bypassing a switchboard/auto-attendant. The PBX matches the DID in its dialplan and rings the corresponding extension or queue.

```
PSTN call to +14155551212
   |
   v
ITSP routes to trunk
   |
   v INVITE with To: <sip:+14155551212@pbx.example.com>
PBX dialplan match: 4155551212 -> ext 1023
   |
   v
SIP/PJSIP endpoint 1023 rings
```

Allocated by ITSP (you order the DID through their portal or API). Often grouped:
- **Local DID:** $0.50-1/month, in-area-code.
- **Toll-free DID:** $1-5/month + per-minute inbound.
- **International DID:** highly variable ($1-50/month depending on country).

**DID groups / blocks:** Sometimes purchased in blocks (10 sequential numbers) for departmental dialing.

**SIP DID handoff:** ITSP sends INVITE; the To URI's user portion or the Diversion header carries the DID. PBX matches via dialplan:

```
; Asterisk extensions.conf
[from-trunk]
exten => 4155551212,1,Goto(employees,1023,1)
exten => 4155551213,1,Goto(employees,1024,1)
exten => 4155551200,1,Goto(reception,1,1)
```

## DOD — Direct Outward Dial

DOD is the outbound mirror of DID: each extension can present its own caller-ID number when calling out, rather than the company main line.

```
ext 1023 dials 5551234
   |
   v
PBX outbound rule: From: <sip:+14155551212@pbx.example.com>
                   P-Asserted-Identity: <sip:+14155551212@itsp.com>
   |
   v INVITE to ITSP
ITSP places call on PSTN with calling number +14155551212
```

PBX must:
- Set `From:` header user-portion = the extension's DOD number (E.164).
- Set the display name to the extension's name (rendered as CNAM after CNAM dip).
- Match against ITSP-permitted CLI list (some ITSPs require you to register each DOD number to prevent spoofing).

Without DOD, every employee's outbound call shows the same main number. With DOD, each agent's caller-ID matches their direct dial inbound, enabling natural callback workflows.

## CLID Manipulation

Two distinct fields:
- **Caller ID Number (CLID number):** The actual digit string (E.164). Carried in `From:` user portion, `P-Asserted-Identity`, and ISDN/SS7 ANI.
- **Caller ID Name (CNAM):** A 15-character ASCII display name ("ACME CORP"). NOT carried in SIP signaling end-to-end. Resolved by **CNAM lookup** at the terminating carrier.

### CNAM Lookup

The terminating carrier (your phone's carrier) receives an inbound call with a CLID number. It performs a **CNAM dip** against the LERG (Local Exchange Routing Guide) or a third-party CNAM database, fetching the registered display name.

- Cost: ~$0.001-0.005 per CNAM dip (paid by the terminating carrier or by you if you do dips on inbound).
- LERG: NANP-wide database mapping number ranges to OCNs (Operating Company Numbers) and their CNAM registrations.
- CNAM registration: You pay to register the name `ACME CORP` for your DID range. Propagation is 7-30 days.

### CNAM-as-Paid-Feature

ITSPs often charge:
- $1-5/month for CNAM registration per DID.
- $0.001-0.005 per inbound CNAM dip if your trunk requests CNAM lookups on inbound calls.

**Caller ID spoofing:** Trivially easy until STIR/SHAKEN. Anyone could `From: <sip:+15551234567@example.com>` an arbitrary number. Robocallers exploited this for two decades.

## P-Asserted-Identity

Defined in RFC 3325. SIP header used inside a "trust domain" (typically between SP and customer's PBX, or between two SPs) to assert the verified caller-ID.

```
INVITE sip:+14155551212@itsp.com SIP/2.0
From: "Anonymous" <sip:anonymous@example.com>
P-Asserted-Identity: <sip:+14085554321@pbx.example.com>
P-Preferred-Identity: <sip:+14085554321@pbx.example.com>
Privacy: id
```

- **`From:` header** is set by the user-agent and is **untrusted** — UAs can put anything there.
- **`P-Asserted-Identity` (PAI)** is set by a trusted edge (SBC, ITSP) AFTER verifying the caller. It carries the SP's assertion of who the caller is.
- **`Privacy: id`** instructs downstream proxies to hide PAI from the destination (caller wants anonymity).

Inside trust domain: PAI is honored. At trust-domain boundary: PAI is stripped if `Privacy: id` is set, or asserted on if not.

ITSPs use PAI to authoritatively label calls between themselves; STIR/SHAKEN extends this with cryptographic proof.

## STIR/SHAKEN

**STIR** = Secure Telephony Identity Revisited (RFC 8224 / 8225 / 8226).  
**SHAKEN** = Signature-based Handling of Asserted information using toKENs (ATIS-1000074).

The answer to caller-ID spoofing. Mandated by FCC's TRACED Act (2019, enforced June 30 2021 in the US). Canada CRTC followed in November 2021.

### How it works

1. **Originating carrier** receives an outbound call. Verifies that the calling number is one its customer is authorized to use.
2. Originating carrier signs a JWT (PASSporT, RFC 8225) containing:
   - `orig` = the calling number.
   - `dest` = the called number.
   - `iat` = timestamp.
   - `attest` = A, B, or C (attestation level).
3. JWT is attached to the INVITE in an `Identity` header.
4. Call traverses interconnect.
5. **Terminating carrier** verifies the JWT signature against the originating carrier's certificate (fetched from their public CA URL, governed by STI-PA).
6. Verification result is conveyed to the called party (typically as a "Verified" or "Spam Likely" annotation on the caller-ID display).

### Attestation Levels

- **A — Full Attestation:** Originator authenticated the caller AND verified the caller is authorized to use the calling number. Strongest.
- **B — Partial Attestation:** Originator authenticated the caller but cannot verify number authorization (common for customers with ported numbers or PBX trunks where SP doesn't own the number).
- **C — Gateway Attestation:** Originator received the call from somewhere upstream and is just passing it through (e.g., from a foreign carrier, or from a non-IP trunk). Weakest.

```
A => "Verified"           strong trust
B => "Verified"           medium trust
C => (often) "Spam Likely" weak trust
none => "Spam Likely"     no STIR/SHAKEN
```

### Carrier Responsibility

- Originating carriers MUST sign all outbound calls.
- Terminating carriers MUST verify all inbound calls.
- Both MUST enroll with the **STI-PA** (Policy Administrator, run by iconectiv) and obtain a certificate from an **STI-CA**.
- **Robocall Mitigation Plan** must be filed with FCC.
- Non-compliant carriers may be blocked by other carriers.

For SIP trunking customers: most large ITSPs sign on your behalf, attestation is typically B (since you're a customer with a number registered through them, full attestation requires they verify ownership chain).

If you run your own SBC and direct interconnects, you may need to deploy STIR/SHAKEN signing yourself: see [JsRSA-based PASSporT signing libraries], `sippy/stir-shaken-as`, Sangoma SBC's built-in module, or Oracle Communications Session Border Controller's STIR/SHAKEN feature.

## SIP Trunk Provisioning

Step-by-step:

1. **Open ITSP account.** Provide business info (regulatory: 911, taxes). Often 1-3 days for KYC.
2. **Order channels and DIDs.** Select N channels, M DIDs (local area codes).
3. **Receive credentials and endpoints.** ITSP gives:
   - SIP server hostname (e.g., `sip.itsp.com`, often with SRV record for `_sip._udp.itsp.com`).
   - Port (5060 UDP, 5061 TLS).
   - Username + password (or you whitelist your IP).
   - IP whitelist on their side (give them your PBX public IP).
   - Outbound proxy / edge SBC FQDN.
4. **Configure PBX outbound:**
   - Define trunk endpoint pointing to ITSP.
   - Define dial pattern (e.g., `_NXXNXXXXXX` -> prepend `+1`, send via trunk).
   - Set From: header to be a verified DOD.
5. **Configure PBX inbound:**
   - Map each DID to extension/queue/IVR.
6. **Test inbound:** Call your DID from a cell phone. Should ring the right extension.
7. **Test outbound:** Have an extension dial out. Should reach destination with correct CLID.
8. **Verify CLID display:** Have someone on T-Mobile / Verizon / AT&T receive your call; check that the display shows your registered CNAM (after 7-30 days propagation).
9. **Verify STIR/SHAKEN:** Most US carrier endpoints will display "Verified" if the trunk is signed properly.
10. **Set up monitoring:** OPTIONS keepalive, CDR export to SIEM, fraud alerts.

## Outbound Proxy / Edge Server

The ITSP's edge is a Session Border Controller. Your PBX talks to this edge over SIP; the edge then routes internally to the ITSP's softswitch and out to PSTN.

- **Signaling path:** PBX -> ITSP edge SBC -> ITSP core -> PSTN gateway.
- **Media path:** Often the same as signaling (anchored on the edge SBC), but some ITSPs use **media-bypass** / **DRA** (Direct Routing Anchor) to optimize: media goes PBX -> PSTN gateway directly.
- **Geographic routing:** Big ITSPs have edges in multiple regions (e.g., US-East, US-West, EU-West, AP-South). Your PBX should connect to the closest one. SRV records typically prioritize geographically.
- **DNS SRV:** `_sip._udp.itsp.com 86400 IN SRV 10 50 5060 east.sbc.itsp.com.` — priority 10, weight 50.
- **Failover:** PBX should handle multiple SRV targets; on connection failure, try the next.

## Codec Negotiation on Trunks

PSTN-side codec selection is much more constrained than enterprise-internal SIP. ITSPs typically support:

- **G.711 μ-law (PCMU):** Default in NANP. 64 kbps, 8 kHz, no compression. Universal.
- **G.711 a-law (PCMA):** Default in EU. Same bitrate.
- **G.729a / G.729ab:** 8 kbps, narrowband. Used to save bandwidth on metered uplinks. Royalty-free since 2017.
- **iLBC:** 13.3 kbps narrowband. Some ITSPs.
- **G.722:** 64 kbps wideband (HD voice). Increasingly common on US carriers.
- **Opus:** 6-510 kbps, ultra-modern. Rare on PSTN trunks (PSTN gateway usually transcodes to G.711).

**Transcoding cost:**
- If PBX offers Opus and ITSP wants G.711, someone must transcode.
- Transcoding burns CPU (~5-10% per concurrent call on a modest server).
- Best practice: have PBX offer G.711 toward the trunk to avoid transcoding at the SP edge.
- For internal calls between IP phones, use Opus or G.722 for HD; transcode only at the trunk handoff.

**Negotiation:** SDP offer/answer (RFC 3264). Both sides list codecs in `m=audio` line in preference order; the answerer picks the first compatible one.

```
; Asterisk pjsip.conf
[itsp-trunk]
type=endpoint
allow=!all,ulaw,alaw,g729
disallow=all
```

## SBC — Session Border Controller

A specialized SIP proxy that sits between two SIP networks (typically: trusted enterprise PBX vs untrusted ITSP, or two ITSPs peering with each other). Provides:

- **Topology hiding:** Strips Via/Record-Route/Contact headers so the internal PBX topology isn't leaked.
- **NAT traversal:** Rewrites Contact, Via, RTP IPs to its own address.
- **Codec transcoding:** Bridges Opus<->G.711, etc.
- **Encryption termination:** TLS+SRTP on one side, plain SIP+RTP on the other (or the reverse).
- **Fraud detection:** Pattern-match outbound CLIs, destination prefixes, call rates.
- **DDoS protection:** Rate-limit malformed SIP, blacklist sources.
- **Routing:** Least-cost, geographic, time-of-day.
- **SIP normalization:** Header manipulation (insert P-Asserted-Identity, remove unsupported headers, fix SDP quirks).
- **STIR/SHAKEN:** Sign or verify Identity headers.
- **Lawful intercept:** CALEA, ETSI LI.

### Common SBCs

| Vendor / Product | Type | Notes |
|------------------|------|-------|
| **Sangoma SBC (Vega / NetBorder)** | Hardware/software | Solid mid-market. |
| **AudioCodes Mediant** | Hardware/software | Mediant 500/800/1000/2600/4000/9080. Strong in Skype-for-Business / Teams Direct Routing. |
| **Oracle Acme Packet (formerly Acme Packet)** | Hardware | Carrier-grade, expensive. Common in tier-1 SP networks. |
| **Cisco CUBE (Unified Border Element)** | IOS-XE on ISR/ASR | Cisco's SBC-on-router. |
| **Ribbon (formerly Sonus + GENBAND)** | Hardware/software | SBC 5xxx, 7xxx. |
| **Kamailio** | Open-source proxy | Highly programmable, scriptable in `kamailio.cfg` (KEMI in Lua/Python/Ruby/JS). High-volume. |
| **OpenSIPS** | Open-source proxy | Fork of OpenSER. Similar to Kamailio. |
| **FreeSWITCH** | Open-source softswitch | Can be configured as B2BUA / SBC. Built-in transcoding. |
| **drachtio** | Node.js framework | Programmable SBC-as-code. |
| **Asterisk** | Open-source PBX | Can be a B2BUA but not usually called an SBC; better: front Asterisk with Kamailio. |

### Topology

```
                     |       DMZ        |
+----------+         |   +-----------+  |    +-------+
|   PBX    |--SIP----|---|    SBC    |--|----|  ITSP |
| 10.0.1.5 |   plain |   | 192.0.2.7 |  | TLS|  edge |
+----------+         |   +-----------+  |    +-------+
                     |                  |
       internal              public
       trust domain          internet
```

Internal: TCP 5060 plain SIP, RTP plain. External: TLS 5061, SRTP. SBC bridges.

## Fraud Prevention

Toll fraud is when an attacker abuses your trunk to place expensive calls (typically to high-cost destinations like satellite phones, premium-rate numbers, or international with revenue-share agreements). A compromised PBX can rack up $50,000+ in hours.

### Layered Defenses

#### 1. Rate Limiting Per Source

Cap call attempts per second per source IP / extension / DID:

```
; OpenSIPS pike module
modparam("pike", "sampling_time_unit", 2)
modparam("pike", "reqs_density_per_unit", 30)  # 30 req / 2s = 15 req/s max
```

Asterisk: `chan_pjsip` has built-in registration throttling; use fail2ban for SIP-flood protection.

#### 2. Geographic Restrictions

Block dial patterns to high-risk countries by default:

```
; Asterisk extensions.conf
exten => _X.,1,GotoIf($["${DIAL_PATTERN}" = "+247" | ${DIAL_PATTERN}" = "+682"]?reject)
exten => _X.,n,Dial(PJSIP/${EXTEN}@itsp)
exten => _X.,n(reject),Hangup(21)  ; rejected
```

Common high-risk prefixes: +247 (Ascension Island), +682 (Cook Islands), +678 (Vanuatu), +880 (Bangladesh), +509 (Haiti), +252 (Somalia), +355 (Albania), +373 (Moldova), +371 (Latvia), +371 (Latvia), +252 (Somalia), 900 prefixes (premium-rate), satellite codes (881x, 882x, 883x).

#### 3. Time-of-Day Restrictions

Block international dialing outside business hours:

```
[time-restrict]
exten => _011.,1,GotoIfTime(09:00-18:00,mon-fri,*,*?allow,1)
exten => _011.,n,Hangup(21)
exten => _011.,n(allow),Goto(international,${EXTEN},1)
```

#### 4. Concurrent-Call Limits Per Trunk / Per Extension

```
; pjsip.conf
[ext-1023]
type=endpoint
max_audio_streams=1
device_state_busy_at=1     ; ext 1023 max 1 simultaneous outbound call
```

For trunks: cap below the channel count purchased to leave room for inbound.

#### 5. 911 + Emergency Exempt

ALWAYS allow emergency dialing regardless of any other restriction. Failure to do so is a regulatory and possibly criminal liability.

```
[emergency]
exten => _9911,1,NoCDR()                   ; some PBXs strip 9 prefix
exten => _9911,n,Dial(PJSIP/911@itsp,,T)   ; T=transfer allowed, no timer
exten => _911,1,Dial(PJSIP/911@itsp,,T)
```

Provide accurate **dispatchable location** (RAY BAUM's Act, Kari's Law in US; both effective 2021).

#### 6. International Dialing: Default-Off, Allowlist-In

The strongest control. By default, deny `_011.` (NANP exit code) or `+` outbound to anything beyond NANP. Allow specific extensions to dial out internationally only after explicit business need.

```
[outbound]
exten => _NXXNXXXXXX,1,Dial(PJSIP/${EXTEN}@itsp)         ; US/CA only
exten => _011.,1,GotoIfTime(${DB(intl_allow/${CHANNEL(callerid)})}?allow:deny)
exten => _011.,n(deny),Hangup(21)
exten => _011.,n(allow),Dial(PJSIP/+${EXTEN:3}@itsp)
```

#### 7. Toll Fraud Monitoring

Watch for:
- Unusual concurrent call counts (10x normal in 30s).
- Repeated calls to the same destination from many extensions (sequential probing).
- Calls outside business hours from extensions that never call after hours.
- Calls to high-cost destinations.
- Calls of unusual duration (e.g., 6-hour calls — usually a hijacked extension dialing a revenue-share line).

Alert via SNMP / webhook / email when thresholds breached.

#### 8. Lock Down SIP Auth

- Strong passwords (24-char random) on every extension.
- Bind extensions to trusted IPs/subnets where possible.
- Enable TLS for extension registration.
- Run `fail2ban` on the PBX to ban IPs after N failed REGISTER attempts.

#### 9. Disable Unused Features

- Disable `t` and `T` Dial flags on inbound from trunk (prevents call transfer to attacker-controlled destinations via DTMF).
- Disable trunk-to-trunk call routing unless explicitly needed.
- Disable FollowMe / mobile twinning by default.

## Concurrent Call Limit

Trunk capacity is measured in **channels** (also called concurrent call sessions). 20 channels = 20 simultaneous active calls. The 21st simultaneous attempt fails (typically `503 Service Unavailable` or `486 Busy Here`).

```
Channels:        20
Inbound active:  12
Outbound active: 7
Available:       1
```

**Sizing rule of thumb:** Erlang-B traffic engineering. For 100 employees with average occupancy 0.15 erlangs each (about 9 minutes of call per hour), at 1% blocking: ~22 channels needed.

```
employees * avg-erlangs-per-employee = total-erlangs
100       * 0.15                     = 15 erlangs

Erlang-B(15 erlangs, 1% blocking) ≈ 22 channels
```

**Burst handling:** Some ITSPs allow you to exceed your purchased channel count, charging an over-burst rate. Useful for unexpected spikes.

**Per-DID concurrency:** Some ITSPs cap concurrent calls per DID (e.g., max 5 simultaneous calls to a single DID). Important for inbound IVR / call center.

## Call Routing

Routing logic in the PBX determines which trunk a call uses based on the destination prefix.

### Prefix-Based Routing

```
[outbound]
exten => _NXXNXXXXXX,1,Dial(PJSIP/${EXTEN}@trunk-us-primary)
exten => _011XX.,1,Dial(PJSIP/+${EXTEN:3}@trunk-international)
exten => _0X.,1,Dial(PJSIP/+${EXTEN:1}@trunk-international)
```

### Least-Cost Routing (LCR)

Multiple ITSPs at different rates per destination. Look up cheapest carrier for the destination prefix.

```
+1...   -> ITSP-A ($0.005/min)
+44...  -> ITSP-B ($0.012/min) cheaper than ITSP-A's $0.025
+86...  -> ITSP-C ($0.040/min) cheaper than B's $0.055
```

Implementation: rate table indexed by prefix, queried before dialing. Pick lowest-rate carrier with available capacity.

### Failover

Primary trunk, secondary trunk. On primary failure (timeout, 503, 5xx), retry on secondary:

```
exten => _NXXNXXXXXX,1,Dial(PJSIP/${EXTEN}@trunk-primary,30,T)
exten => _NXXNXXXXXX,n,GotoIf($["${DIALSTATUS}"="ANSWER"]?end)
exten => _NXXNXXXXXX,n,Dial(PJSIP/${EXTEN}@trunk-secondary,30,T)
exten => _NXXNXXXXXX,n(end),Hangup()
```

### Priority / Weight (Kamailio dispatcher)

```
1 sip:trunk1.itsp.com 0 50 priority=10  weight=80
1 sip:trunk2.itsp.com 0 50 priority=10  weight=20
1 sip:backup.itsp2.com 0 50 priority=20 weight=100
```

Round-robin within priority 10 (80/20 split), failover to priority 20.

## Inbound Call Flow

```
PSTN caller dials +14155551212
   |
   v
LEC's switch routes call to ITSP via SS7/SIP
   |
   v
ITSP receives call, lookups DID-to-customer mapping
   |
   v INVITE sip:+14155551212@pbx.example.com
ITSP edge SBC sends INVITE to your PBX (via PBX's public IP / NAT)
   |
   v
PBX (SBC if present) accepts, replies 100 Trying, 180 Ringing
   |
   v
PBX dialplan match: DID 4155551212 -> ext 1023 (Sales)
   |
   v INVITE sip:1023@10.0.1.5
PBX rings IP phone at extension 1023
   |
   v 200 OK / ACK
Phone answers; PBX 200 OK back to ITSP; ITSP 200 OK back to PSTN
   |
   v RTP
Media flows: PSTN<->ITSP-PSTN-gw<->ITSP-edge<->PBX<->phone
```

## Outbound Call Flow

```
Phone at ext 1023 dials 9-1-415-555-1234
   |
   v INVITE sip:914155551234@10.0.1.5
PBX dialplan match _9NXXNXXXXXX
   |
   v
Strip "9" outside-line prefix -> 4155551234
   |
   v
Reformat to E.164 -> +14155551234
   |
   v
Set From: <sip:+14085554321@pbx.example.com> (DOD for ext 1023)
   |
   v
Set P-Asserted-Identity: <sip:+14085554321@itsp.com>
   |
   v INVITE sip:+14155551234@sip.itsp.com
PBX -> ITSP edge SBC
   |
   v
ITSP signs Identity header (STIR/SHAKEN attest=A)
   |
   v
ITSP routes to PSTN gateway via SS7
   |
   v
Destination phone rings
```

## SIP Trunk Configuration in Asterisk

PJSIP (modern; chan_sip is deprecated since Asterisk 17).

### `pjsip.conf`

```
;
; Transport for the trunk (TLS recommended)
;
[transport-tls]
type=transport
protocol=tls
bind=0.0.0.0:5061
cert_file=/etc/asterisk/keys/pbx.pem
priv_key_file=/etc/asterisk/keys/pbx.key
ca_list_file=/etc/ssl/certs/ca-certificates.crt
method=tlsv1_2
verify_client=no
verify_server=yes

[transport-udp]
type=transport
protocol=udp
bind=0.0.0.0:5060
external_media_address=203.0.113.5
external_signaling_address=203.0.113.5
local_net=10.0.0.0/8
local_net=192.168.0.0/16
local_net=172.16.0.0/12

;
; Auth credentials for ITSP
;
[itsp-auth]
type=auth
auth_type=userpass
username=YOURACCOUNT
password=SECRETPASSWORD

;
; Address-of-record (where to send INVITEs to ITSP)
;
[itsp-aor]
type=aor
contact=sip:sip.itsp.com:5060
qualify_frequency=30
qualify_timeout=3.0

;
; Endpoint definition
;
[itsp]
type=endpoint
context=from-trunk
transport=transport-udp
aors=itsp-aor
outbound_auth=itsp-auth
disallow=all
allow=ulaw,alaw,g729
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes
ice_support=no
trust_id_inbound=yes
trust_id_outbound=yes
send_pai=yes
send_rpid=no
from_user=YOURACCOUNT
from_domain=sip.itsp.com

;
; Identify-by-IP (no registration; ITSP whitelists our IP)
;
[itsp-identify]
type=identify
endpoint=itsp
match=64.8.0.0/16          ; ITSP's IP range
match=72.10.0.0/16

;
; OR registration if IP-whitelist not used
;
[itsp-registration]
type=registration
transport=transport-udp
outbound_auth=itsp-auth
server_uri=sip:sip.itsp.com
client_uri=sip:YOURACCOUNT@sip.itsp.com
contact_user=YOURACCOUNT
expiration=3600
retry_interval=60
forbidden_retry_interval=600
max_retries=10
```

### `extensions.conf`

```
[from-trunk]
;
; Inbound DID routing
;
exten => 4155551212,1,NoOp(Inbound to main line)
 same => n,Goto(from-trunk-internal,1023,1)

exten => 4155551200,1,NoOp(Inbound to reception)
 same => n,Goto(reception-ivr,start,1)

exten => _.,1,NoOp(Unmatched DID ${EXTEN})
 same => n,Hangup(404)

[from-trunk-internal]
exten => _10XX,1,NoOp(Forwarding inbound DID to ${EXTEN})
 same => n,Dial(PJSIP/${EXTEN},30,T)
 same => n,VoiceMail(${EXTEN}@default,u)
 same => n,Hangup()

[to-trunk]
;
; Outbound dial patterns
;
; US/CA 10-digit
exten => _NXXNXXXXXX,1,NoOp(Outbound US/CA to ${EXTEN})
 same => n,Set(CALLERID(num)=${DB(dod/${CHANNEL(endpoint)})})
 same => n,Dial(PJSIP/+1${EXTEN}@itsp,60,T)
 same => n,Hangup()

; US/CA 11-digit (with 1)
exten => _1NXXNXXXXXX,1,NoOp(Outbound US/CA 11-digit ${EXTEN})
 same => n,Dial(PJSIP/+${EXTEN}@itsp,60,T)

; Toll-free
exten => _1800NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)
exten => _1833NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)
exten => _1844NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)
exten => _1855NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)
exten => _1866NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)
exten => _1877NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)
exten => _1888NXXXXXX,1,Dial(PJSIP/+${EXTEN}@itsp,60,T)

; Emergency (always allowed)
exten => 911,1,Dial(PJSIP/911@itsp,60,T)
exten => 9911,1,Dial(PJSIP/911@itsp,60,T)

; International (gated)
exten => _011.,1,GotoIf($[${DB(intl_allow/${CHANNEL(endpoint)})}=1]?allow:deny)
 same => n(deny),Playback(international-not-permitted)
 same => n,Hangup(21)
 same => n(allow),Dial(PJSIP/+${EXTEN:3}@itsp,60,T)
```

### `pjsip.conf` extension definition

```
[1023]
type=endpoint
context=from-internal
transport=transport-tls
disallow=all
allow=opus,g722,ulaw
auth=1023-auth
aors=1023-aor
direct_media=no

[1023-auth]
type=auth
auth_type=userpass
username=1023
password=RANDOM24CHARSECRET

[1023-aor]
type=aor
max_contacts=2
qualify_frequency=60
remove_existing=yes
```

## SIP Trunk Configuration in FreeSWITCH

FreeSWITCH uses a `sofia` SIP module with profiles (typically `internal` for extensions, `external` for trunks).

### `sip_profiles/external/itsp.xml`

```
<include>
  <gateway name="itsp">
    <param name="username" value="YOURACCOUNT"/>
    <param name="password" value="SECRETPASSWORD"/>
    <param name="realm" value="sip.itsp.com"/>
    <param name="proxy" value="sip.itsp.com"/>
    <param name="register" value="true"/>
    <param name="register-transport" value="udp"/>
    <param name="expire-seconds" value="600"/>
    <param name="retry-seconds" value="30"/>
    <param name="ping" value="30"/>
    <param name="caller-id-in-from" value="false"/>
    <param name="codec-prefs" value="PCMU,PCMA,G729"/>
    <param name="from-user" value="YOURACCOUNT"/>
    <param name="from-domain" value="sip.itsp.com"/>
    <param name="extension-in-contact" value="true"/>
  </gateway>
</include>
```

### `dialplan/default.xml` — outbound

```
<extension name="outbound-us">
  <condition field="destination_number" expression="^(\d{10})$">
    <action application="set" data="effective_caller_id_number=${dod_number}"/>
    <action application="set" data="effective_caller_id_name=${dod_name}"/>
    <action application="bridge" data="sofia/gateway/itsp/+1$1"/>
  </condition>
</extension>

<extension name="outbound-international">
  <condition field="destination_number" expression="^011(\d+)$">
    <action application="bridge" data="sofia/gateway/itsp/+$1"/>
  </condition>
</extension>

<extension name="outbound-emergency">
  <condition field="destination_number" expression="^9?(911|112|999)$">
    <action application="set" data="effective_caller_id_number=${ELIN}"/>
    <action application="bridge" data="sofia/gateway/itsp/$1"/>
  </condition>
</extension>
```

### `dialplan/public.xml` — inbound

```
<extension name="public_did">
  <condition field="destination_number" expression="^(\+?14155551212|14155551212)$">
    <action application="set" data="domain_name=$${domain}"/>
    <action application="transfer" data="1023 XML default"/>
  </condition>
</extension>
```

## SIP Trunk Configuration in Kamailio

Kamailio is a high-performance SIP proxy. Often used in front of Asterisk/FreeSWITCH for load-balancing, ACL, and SBC duties.

### `kamailio.cfg` (extracts)

```
#!define WITH_AUTH
#!define WITH_DISPATCHER
#!define WITH_ACL

# Listen
listen=udp:eth0:5060
listen=tls:eth0:5061

# Modules
loadmodule "dispatcher.so"
loadmodule "permissions.so"
loadmodule "acc.so"
loadmodule "auth.so"
loadmodule "auth_db.so"

# Dispatcher (round-robin to ITSP IPs)
modparam("dispatcher", "list_file", "/etc/kamailio/dispatcher.list")
modparam("dispatcher", "ds_ping_method", "OPTIONS")
modparam("dispatcher", "ds_ping_interval", 30)
modparam("dispatcher", "ds_probing_mode", 1)
modparam("dispatcher", "ds_probing_threshold", 3)

# Permissions / ACL
modparam("permissions", "db_url", DBURL)
modparam("permissions", "trusted_table", "trusted")

# Accounting
modparam("acc", "log_flag", 1)
modparam("acc", "db_url", DBURL)

route {
    # IP-based ACL: only allow trunked sources
    if (!allow_trusted("$si", "$proto")) {
        sl_send_reply("403", "Forbidden");
        exit;
    }

    if (is_method("INVITE")) {
        # LCR / dispatcher
        if (!ds_select_dst("1", "4")) {  # set 1, alg 4=round-robin
            sl_send_reply("503", "No gateway available");
            exit;
        }
        t_on_failure("RTF_FAILOVER");
        t_relay();
        exit;
    }
}

failure_route[RTF_FAILOVER] {
    if (t_check_status("503|408|500")) {
        if (ds_next_dst()) {
            t_on_failure("RTF_FAILOVER");
            t_relay();
            exit;
        }
    }
}
```

### `dispatcher.list`

```
# setid destination flags priority attrs
1 sip:edge-east.itsp.com:5060 0 100 weight=50;duid=east
1 sip:edge-west.itsp.com:5060 0 100 weight=50;duid=west
1 sip:backup.itsp2.com:5060 0 50  weight=100;duid=backup
```

## Trunk Health Monitoring

### OPTIONS Keepalive

The de-facto SIP heartbeat. Send `OPTIONS sip:trunk@itsp` every 30s. ITSP replies 200 OK if up.

- Asterisk: `qualify_frequency=30` on AOR (default in PJSIP).
- FreeSWITCH: `<param name="ping" value="30"/>` on gateway.
- Kamailio: `dispatcher.ds_ping_interval=30`.
- OpenSIPS: similar via dispatcher.

### Registration State Monitoring

If using REGISTER, watch for:
- Periodic 200 OK on REGISTER.
- 401/407 (challenge) followed by 200 (auth success).
- Persistent 401/403 = creds wrong.
- Timeout = network issue.

Asterisk: `pjsip show registrations`. FreeSWITCH: `sofia status gateway itsp`. Kamailio: `kamcmd dispatcher.list`.

### Alarms

Alert via SNMP / email / PagerDuty when:
- OPTIONS fails 3x consecutively.
- REGISTER fails for >5 minutes.
- Inbound INVITEs drop to 0 over a 15-minute window during business hours.
- Outbound 503/5xx rate > 5%.

```
# nagios check
define service {
    service_description    SIP Trunk OPTIONS
    check_command          check_sip!sip.itsp.com!5060
    check_interval         1
    notification_options   w,c,r
}
```

## Calling Plans

| Plan | Description |
|------|-------------|
| **Pure per-minute** | $0.005-0.015/min outbound US, no monthly minimums. Inbound free or $0.001-0.005/min. Best for low-volume. |
| **Flat-rate** | $25-40/channel/month for unlimited US local + LD. Best above ~3000 min/channel/month. |
| **Bundle** | N minutes included, overage per-minute. Mid-volume. |
| **DID-only** | $1-2/DID/month, no channel fee, per-minute outbound. Best for inbound-heavy. |
| **Toll-free included** | $5-10/month for 1 toll-free DID + $0.02-0.05/min inbound. |
| **International add-on** | Country-specific rate cards. Often requires deposit. |

**Per-minute rounding policies:** Most ITSPs round up to the nearest 6 seconds (1/10 minute) or 60 seconds. Always check; the difference for a 30-second call billed as 60 seconds doubles the cost.

## CDR / Billing Reconciliation

Match your PBX CDR (Call Detail Records) to ITSP invoice. Discrepancies usually indicate:
- ITSP rounded up (60-second min, 6-second increments).
- Failed calls billed (some ITSPs bill failed calls; most don't).
- Unauthorized calls (toll fraud — investigate immediately).
- Surcharges (regulatory, 911, USF) added to invoice.

```
# CDR diff (PBX vs invoice)
SELECT cdr.dst, cdr.duration AS pbx_dur, inv.duration AS itsp_dur,
       (inv.duration - cdr.duration) AS delta_seconds,
       cdr.calldate
FROM cdr
JOIN itsp_invoice inv ON cdr.uniqueid = inv.call_id
WHERE ABS(inv.duration - cdr.duration) > 10  -- discrepancy >10s
ORDER BY cdr.calldate;
```

The "where did this $300 charge come from" diagnostic:
1. Filter invoice for high-cost destinations.
2. Cross-reference CDR for the source extension.
3. Check whether call pattern is normal (regular international caller) or anomalous (extension that has never dialed internationally).
4. If anomalous, suspect toll fraud — change the extension password immediately, audit logs.

## Number Porting

LNP — Local Number Portability. Move a number from one carrier to another.

### Process

1. **Customer gives losing carrier's account info** to gaining carrier (account number, PIN, billing address — must match exactly).
2. **Gaining carrier files a port request** through NPAC (Number Portability Administration Center) or ATIS LSR (Local Service Request).
3. **Losing carrier validates** ownership and either approves (FOC = Firm Order Commitment) or rejects.
4. **Port date scheduled** (typically T+5-10 business days, sometimes T+1 for "expedited").
5. **At port date / time:** Routing in NPAC updates. New carrier receives calls.
6. **Brief outage during cutover** (seconds to minutes).

### Gotchas

- **Losing carrier required service**: Number must still be active at losing carrier for port to succeed.
- **Account info must match exactly**: even one wrong digit on the BTN (Billing Telephone Number) causes rejection.
- **Outstanding balance**: Some losing carriers stall the port until balance is paid.
- **Non-portable area codes**: Some rural exchanges where number portability isn't supported (rare in US/CA but common internationally).
- **Toll-free porting** uses a separate process with the **RespOrg** (see below).
- **International porting** is country-by-country; usually slower (T+30 days).

### Port Date Coordination

For a working business, you cannot afford an outage. Plan:

1. Add gaining carrier as **secondary trunk** before port date.
2. Configure inbound dialplan to accept the DID via gaining carrier as well.
3. Port date arrives; calls start flowing via gaining carrier.
4. After 24-48h verification, remove DID from losing carrier.
5. Cancel losing carrier's billing.

## Toll-Free Numbers

Numbers in the 8xx codes (800/833/844/855/866/877/888) where the **called party pays**. Caller dials free.

### RespOrg

The RespOrg (Responsible Organization) controls a toll-free number's routing in **SMS/800** (the toll-free database, run by Somos). Only one RespOrg at a time; changing RespOrg is the toll-free equivalent of porting.

- RespOrg ID is a 5-character code (e.g., `B0001`, `T0123`).
- RespOrg controls routing data: termination carrier, percentage allocation, geographic routing, time-of-day routing.

### SMS/MMS-Enabled Toll-Free

Toll-free numbers can be SMS-enabled via the **Toll-Free Messaging Roaming Database** (Somos). Enables business-to-consumer messaging.

- Most ITSPs support toll-free SMS via API (Twilio, Bandwidth, Telnyx).
- Verification process required (TCR = The Campaign Registry, or Somos for toll-free).

### Reverse-Billing / Per-Minute Inbound

Toll-free DIDs typically charge YOU (the recipient) per minute for incoming calls (typical $0.02-0.05/min). Plan for this in your cost model.

## SMS/MMS Over SIP

### SIP MESSAGE Method

RFC 3428. A SIP request that carries a text payload, sent like an INVITE but with no media.

```
MESSAGE sip:+14155551212@itsp.com SIP/2.0
From: <sip:+14085554321@pbx.example.com>
To: <sip:+14155551212@itsp.com>
Content-Type: text/plain
Content-Length: 21

Hello, this is a test
```

### SMPP / HTTP API Alternatives

Few ITSPs use SIP MESSAGE for SMS in production. Most use:

- **SMPP (Short Message Peer-to-Peer)**: Telecom-grade protocol for SMS. Used between SMSCs. Some ITSPs offer SMPP gateway access for high-volume senders.
- **HTTPS REST API**: The norm. Twilio Programmable Messaging, Bandwidth Messaging API, Telnyx Messaging, etc.

```
curl -X POST https://api.twilio.com/2010-04-01/Accounts/$ACCT/Messages.json \
  -u "$ACCT:$AUTH" \
  -d "From=+14085554321" \
  -d "To=+14155551212" \
  -d "Body=Hello from API"
```

### MMS Over HTTPS API

Multimedia (image/audio/video) is virtually always sent via HTTPS API with a `MediaUrl` parameter. SIP MESSAGE rarely carries multimedia in commercial deployments.

### A2P 10DLC Compliance

For business-to-consumer messaging on US local numbers, you must register with **TCR (The Campaign Registry)**:
- Brand registration ($4 one-time).
- Campaign registration ($10/month + per-message surcharges).
- Failure to register: messages get filtered/blocked by carriers, or surcharged at "unregistered" rates ($0.005-0.02 extra per message).

## CNAM

Caller ID Name. 15-character ASCII string ("ACME CORP", "JONES J", "WIRELESS CALLER").

### LERG / CNAM Database

- **LERG (Local Exchange Routing Guide):** Telcordia/iconectiv-maintained database of NPA-NXX assignments to OCNs (Operating Company Numbers).
- **CNAM database:** Separate from LERG. Each LEC operates a CNAM database server queried by terminating carriers via SS7 TCAP.
- Third-party providers: CallerID.com, Truecaller, CallApp, Hiya — supplement / replace LEC CNAM.

### CNAM Dip Cost

When your phone receives a call, your terminating carrier dips the CNAM database to fetch the display name. This dip costs the terminating carrier ~$0.001-0.005. They may pass this cost on or absorb it.

If your trunk does its own CNAM dip on inbound, expect $0.001-0.005 per inbound call as a line item.

### Registering CNAM

You register your CNAM with your ITSP. They publish it to the LERG/CNAM databases. Propagation: 7-30 days. Rejected names: anything misleading, profane, or too long.

## STIR/SHAKEN Implementation

For ITSPs and large enterprises with direct interconnects, you must implement signing and verification.

### Signing (Originating Carrier)

1. Obtain certificate from STI-CA (e.g., iconectiv, Sectigo, Comodo). Requires SPC token from STI-PA.
2. Configure SBC / softswitch to sign every outbound INVITE:

```
# Sangoma SBC YAML (conceptual)
stir-shaken:
  enabled: true
  cert: /etc/sbc/stir-cert.pem
  key:  /etc/sbc/stir-key.pem
  attest: B    # default attestation level
  default_orig_id: +14085554321
```

3. JWT (PASSporT) attached to INVITE:

```
INVITE sip:+14155551212@far-end SIP/2.0
Identity: eyJhbGciOiJFUzI1NiIsInBwdCI6InNoYWtlbiIsInR5cCI6InBhc3Nwb3J0IiwieDV1Ijoi
          aHR0cHM6Ly9jZXJ0LmV4YW1wbGUuY29tL2NlcnQucGVtIn0.eyJhdHRlc3QiOiJBIiwiZGV
          zdCI6eyJ0biI6WyIrMTQxNTU1NTEyMTIiXX0sImlhdCI6MTcxNTAwMDAwMCwib3JpZyI6ey
          J0biI6IisxNDA4NTU1NDMyMSJ9LCJvcmlnaWQiOiJhYmMxMjMifQ.signature_bytes_he
          re;info=<https://cert.example.com/cert.pem>;alg=ES256;ppt=shaken
```

### Verification (Terminating Carrier)

1. Extract Identity header from INVITE.
2. Parse JWT, extract `x5u` (cert URL) from header.
3. Fetch cert from `x5u` (HTTPS). Verify cert chain to STI-CA root.
4. Verify JWT signature (ES256).
5. Verify `iat` is recent (within 60s, typically).
6. Verify `orig.tn` matches `From:` header.
7. Verify `dest.tn` matches `To:` header.
8. Mark call as Verified or Failed.
9. Optionally pass verification result downstream via `Identity-Status` or jCard or display annotation.

### Failed Verification → Spam

When verification fails or attestation is C, terminating carriers (especially T-Mobile, Verizon, AT&T) typically:
- Display "Spam Likely" or "Scam Likely" on the called party's screen.
- Route to enhanced spam filtering.
- Some carriers outright block.

This is why **B-attestation is now table stakes** for legitimate businesses. Calls with no signing or C-only signing get filtered.

## International Trunk Considerations

### Destination Rate Card

Every country/destination has a rate. Some "premium" destinations (satellite, mobile in expensive countries, revenue-share scams) cost $1-5/min.

- **Rate cards updated weekly.**
- Some ITSPs publish them as CSV; load into your LCR engine.
- Destination prefix specificity matters: `+88216` (Inmarsat) is distinct from `+881` (general satellite) — tens of dollars per minute difference.

### Robocaller Scrubbing

Foreign carriers in some countries route massive volumes of robocaller traffic. ITSPs apply "scrubbing":
- Block known bad source carriers.
- Apply STIR/SHAKEN for ingress where supported.
- Honeypot / trap-line analysis.

### Country-Specific Compliance

- **UK Ofcom:** GC8.4 — call traceability requirements; STIR/SHAKEN-equivalent on the way.
- **EU CRA (Cyber Resilience Act):** Indirect impact on telecom; expect more compliance overhead by 2027.
- **France Arcep:** Mandatory authentication for international calls displaying a French CLI (effective 2024).
- **Germany BNetzA:** CLI rules — international calls displaying German CLI must be blocked unless authenticated.
- **India TRAI:** Strict regulations on international call CLI (calls displaying Indian numbers from abroad are blocked unless explicitly whitelisted).

When dialing internationally with non-local CLI, expect calls to be blocked or stripped. Use a CLI registered in the destination country if you need to display a local number.

## Common Errors

Verbatim, with canonical fix.

```
REGISTER 401 Unauthorized
  -> Wrong credentials. Verify username/password match what ITSP gave you.
     Check realm matches (some ITSPs require auth_realm=sip.itsp.com explicitly).

REGISTER 403 Forbidden
  -> Wrong account, suspended account, or your IP is not whitelisted.
     Check ITSP portal for account status. If using IP-auth, verify your
     public IP matches the whitelist (look at ITSP's CDR for "from-IP").

REGISTER 408 Request Timeout
  -> No response from ITSP. Network/firewall. Check UDP 5060 outbound
     allowed; check NAT keepalive; check DNS resolves.

INVITE 503 Service Unavailable
  -> ITSP overloaded, your trunk over-subscribed (channels exhausted),
     or outbound rate-limit hit. Check ITSP status page; check your
     concurrent-call count vs purchased channels.

INVITE 480 Temporarily Unavailable
  -> Destination phone is off-hook, on Do-Not-Disturb, or unreachable.

INVITE 488 Not Acceptable Here
  -> SDP/codec mismatch. Your offer doesn't intersect with ITSP-supported
     codecs. Add G.711 (ulaw, alaw) to allow= list.

INVITE 408 Request Timeout
  -> Network or firewall dropping packets. Check tcpdump on trunk
     interface for outbound INVITE going out and any ICMP unreachable
     coming back. Check that ITSP isn't blacklisting your IP.

INVITE 603 Decline
  -> ITSP fraud-blocked the call. Often: international destination
     not enabled on account, or call pattern flagged as suspicious.
     Contact ITSP support to whitelist destination.

INVITE 487 Request Terminated
  -> Caller hung up before answer. Normal; not an error per se.

INVITE 404 Not Found
  -> Destination DID doesn't exist or is unassigned at terminating
     carrier. Verify number is portable / dialable.

BYE 481 Call/Transaction Does Not Exist
  -> Race: both sides hung up simultaneously, or dialog state lost
     (PBX restart). Usually benign.

Cannot send re-INVITE: dialog terminated
  -> Dialog timed out mid-call (often Session-Timer expiry without
     refresh). Set "timers=yes;timers_min_se=90;timers_sess_expires=1800"
     in PJSIP endpoint.

100 Trying received but never 180/183
  -> Far-end is processing but not yet ringing. If never advances,
     dest carrier is hung. Increase RING timeout; investigate dest.

INVITE 400 Bad Request
  -> Malformed SIP. Often from header normalization mismatch. Use
     sngrep to inspect; compare to a working call.

INVITE 415 Unsupported Media Type
  -> SDP m= line uses a codec the far-end can't parse (rare with
     modern codecs). Strip exotic codecs from offer.

INVITE 416 Unsupported URI Scheme
  -> You sent sips: but ITSP only supports sip:, or vice versa.

INVITE 420 Bad Extension
  -> Your Require: header lists an extension the ITSP doesn't support.
     Remove the Require: header or pick supported extensions.

INVITE 421 Extension Required
  -> ITSP demands a Supported: extension you didn't include.

INVITE 423 Interval Too Brief
  -> Min-Expires from server > your Expires. Bump REGISTER expiry.

INVITE 491 Request Pending
  -> Both sides sent INVITEs simultaneously. Glare. Retry with backoff.

INVITE 500 Server Internal Error
  -> ITSP softswitch crashed/error. Check ITSP status; failover.

INVITE 501 Not Implemented
  -> ITSP doesn't support the method. Check version.

INVITE 504 Server Time-out
  -> ITSP can't reach further upstream. Failover.

INVITE 606 Not Acceptable
  -> Far-end rejects all your media offers. Codec, encryption, or
     bandwidth mismatch.

OPTIONS 484 Address Incomplete
  -> You sent OPTIONS to a wildcard or partial URI. Set qualify
     target to a specific reachable URI.
```

## Common Gotchas

Twelve+ broken→fixed patterns.

### 1. Forgot to Whitelist PBX IP at ITSP

```
Symptom:  REGISTER 403 Forbidden, INVITE 403 Forbidden
Diagnose: Look at ITSP portal "from IP" field. Compare to your
          public IP (curl ifconfig.me from PBX).
Fix:      Add public IP to ITSP allowlist. If dynamic IP, switch to
          credential-based REGISTER.
```

### 2. Wrong Port (5060 vs 5061)

```
Symptom:  Connection refused, SSL handshake error.
Diagnose: ss -tuln on PBX, tcpdump on trunk interface.
Fix:      Use 5060 for plain UDP/TCP, 5061 for TLS. Confirm with ITSP
          which they want.
```

### 3. DNS Resolution Inside PBX

```
Symptom:  "Cannot resolve sip.itsp.com" in logs.
Diagnose: nslookup sip.itsp.com from PBX shell.
Fix:      Add ITSP-provided SRV records to your DNS, or use
          1.1.1.1/8.8.8.8 as resolver. Many ITSPs require:
            _sip._udp.sip.itsp.com. SRV 10 50 5060 east.sbc.itsp.com.
```

### 4. TLS Cert Not in Trust Store

```
Symptom:  TLS handshake fails, "unable to get local issuer certificate".
Diagnose: openssl s_client -connect sip.itsp.com:5061 -showcerts
Fix:      Add ITSP cert chain to /etc/ssl/certs/ca-certificates.crt
          (Debian) or /etc/pki/tls/certs/ca-bundle.crt (RHEL).
          Run update-ca-certificates / update-ca-trust.
```

### 5. Concurrent-Call Limit Hit During Demo

```
Symptom:  503 Service Unavailable on Nth simultaneous call.
Diagnose: Check active call count vs trunk channel count.
Fix:      Order more channels at ITSP, OR raise burst allowance,
          OR audit for stuck dialogs (zombie calls eating capacity)
          via "pjsip show channelstats".
```

### 6. International Dialing Not Enabled at ITSP

```
Symptom:  503 / 603 on outbound +CC calls (CC != 1).
Diagnose: Check ITSP account "International Dialing: Disabled".
Fix:      Enable in portal; usually requires deposit. Set per-country
          permissions.
```

### 7. Caller ID Rewritten Incorrectly

```
Symptom:  Outbound calls show "UNKNOWN" or empty CLID; or wrong number.
Diagnose: tcpdump From: header on outbound INVITE.
Fix:      Set From: <sip:+14085554321@pbx.example.com> (full E.164).
          Set send_pai=yes in PJSIP. Verify CALLERID(num) is set
          BEFORE Dial().
```

### 8. DID Not Provisioned

```
Symptom:  INVITE 404 Not Found on inbound to a specific number.
Diagnose: ITSP portal "DIDs" — is the number listed?
Fix:      Order DID. Wait for activation (immediate to 24h).
```

### 9. Codec Negotiation Failure

```
Symptom:  INVITE 488 Not Acceptable Here.
Diagnose: sngrep -> view INVITE -> compare m= line in offer with
          ITSP's supported codecs.
Fix:      Add ulaw, alaw to allow= list in pjsip endpoint.
          disallow=all; allow=!all,ulaw,alaw,g729.
```

### 10. SBC NAT Helper Rewriting Contact

```
Symptom:  Call connects but media is silent (one-way or no audio).
Diagnose: tcpdump RTP — is media going to a private IP?
Fix:      In PJSIP transport: external_media_address=PUBLIC_IP,
          external_signaling_address=PUBLIC_IP, local_net=10.0.0.0/8.
          rtp_symmetric=yes, force_rport=yes, rewrite_contact=yes.
          On firewall: disable SIP-ALG (it mangles SIP).
```

### 11. SIP-Aware NAT on Customer Firewall

```
Symptom:  Random one-way audio, registration drops, mangled headers.
Diagnose: Compare SIP packets at PBX vs at firewall WAN interface.
Fix:      Disable SIP-ALG on firewall (every consumer firewall has
          it on by default; it's almost always wrong).
              Cisco ASA: no inspect sip
              Sonicwall: VoIP -> Settings -> Enable consistent NAT (off)
              FortiGate: config system settings -> set sip-helper disable
          Use STUN or, better, an outbound-only VPN to the ITSP.
```

### 12. Mid-Call DTMF Lost

```
Symptom:  IVR not picking up button presses; user can't navigate.
Diagnose: Check what DTMF mode is negotiated.
Fix:      Use RFC 4733 (out-of-band, telephone-event payload type).
          In PJSIP: dtmf_mode=rfc4733.
          Both sides must agree. SIP INFO is fallback.
          In-band DTMF only works with G.711 and is fragile.
```

### 13. Session Timer Expiry

```
Symptom:  Calls drop at exactly 30 min or 1 hour.
Diagnose: Look for BYE with Reason: Q.850;cause=44 or session timer
          expired.
Fix:      Enable Session-Timer:
            timers=yes
            timers_min_se=90
            timers_sess_expires=1800
          Send re-INVITE every 1800/2 = 900s.
```

### 14. Asterisk T.38 / Fax Failure

```
Symptom:  Fax start tone but no completion.
Diagnose: Check t38pt_udptl=yes on endpoint and fax.
Fix:      Use T.38 with NSE re-INVITE; some ITSPs require
          t38_redundancy_count=5. Alternatively, use G.711 pass-through
          (lossless ulaw) for fax with ECM disabled.
```

### 15. STIR/SHAKEN Verification Fails Outbound

```
Symptom:  Calls land as "Spam Likely".
Diagnose: ITSP portal -> Outbound Quality -> attestation level.
Fix:      Register your DID in ITSP's known-good caller ID list (so
          they sign as A or B). Some ITSPs require you to provide
          documentation of number ownership.
```

## Diagnostic Recipes

### tcpdump on Trunk Interface

```
# Capture SIP signaling to/from ITSP IP
tcpdump -i eth0 -nn -s0 -w /tmp/sip.pcap host 64.8.10.20

# Filter for UDP/5060 only (signaling)
tcpdump -i eth0 -nn -s0 'udp port 5060'

# RTP (after dynamic port; usually 10000-20000)
tcpdump -i eth0 -nn -s0 'udp portrange 10000-20000'

# Read in Wireshark or sngrep
sngrep -I /tmp/sip.pcap
```

### sngrep — Real-Time SIP Flow Viewer

```
# Live capture, all SIP messages
sudo sngrep

# Filter to a single call leg
sudo sngrep -d eth0 'src host 64.8.10.20 or dst host 64.8.10.20'

# Inside sngrep:
#   Enter on a dialog -> see all SIP messages in flow diagram
#   F2 / F3 / F4 -> filter by extensions, methods, response code
#   F5 -> save to PCAP
```

### ITSP Web Portal CDR

Confirm signaling reached the ITSP. Their portal logs every INVITE they receive from you, with timestamps and final response codes. Compare against your PBX CDR.

If your PBX sent INVITE but ITSP has no record, packet didn't reach (firewall/NAT issue).
If ITSP has INVITE but your PBX shows none, response was lost coming back.

### Test Echo Test Number

Most ITSPs publish an echo test number. Examples:
- VoIP.ms: `*43`
- Twilio: `+1-800-921-2949` (echo + greeting)
- Telnyx: `+1-844-868-5359` (test extension echo)
- Generic: `*43` works on many Asterisk PBXs (built-in)

Tests bidirectional audio: you hear your own voice.

### Test Call from Cell to Your DID

The fastest end-to-end inbound test. Have someone on a different network (cell phone, friend's landline) dial your DID. Verify:
- Right extension rings.
- Caller ID displays correctly.
- Audio is two-way.
- No drops.

### `pjsip show endpoint`

```
*CLI> pjsip show endpoint itsp
   Endpoint:  itsp                                            Not in use
                Aor:  itsp-aor                                  1
              Contact:  itsp-aor/sip:sip.itsp.com:5060              Avail
   Transport:  transport-udp                udp
       Identify:  itsp-identify/itsp
```

### `sofia status`

```
freeswitch> sofia status
Name        Type   Data                                 State
=========================================================================================
external    profile sip:mod_sofia@203.0.113.5:5060      RUNNING (1)
itsp        gateway sip:YOURACCOUNT@sip.itsp.com        REGED
```

### `kamcmd dispatcher.list`

```
{
  "SET": {
    "ID": 1,
    "TARGETS": [
      { "DEST": { "URI": "sip:east.sbc.itsp.com:5060", "FLAGS": "AP", "PRIORITY": 100 } },
      { "DEST": { "URI": "sip:west.sbc.itsp.com:5060", "FLAGS": "AP", "PRIORITY": 100 } }
    ]
  }
}
```

`AP` = Active + Probing. Anything else = degraded.

## Cost Optimization

### Multi-ITSP LCR

Pick the cheapest ITSP per destination. Maintain rate tables; query before each outbound INVITE.

```
Destination prefix    ITSP-A      ITSP-B      ITSP-C
+1...                $0.005      $0.007      $0.006
+44...               $0.012      $0.025      $0.018
+86...               $0.040      $0.030      $0.055
+91...               $0.025      $0.020      $0.018
```

For each call, route via lowest-rate carrier with available capacity.

### Aggregator vs Direct ILEC

- **Aggregator (ITSP):** Buys minutes wholesale across many carriers, resells. Single contract, per-minute rate. Easy.
- **Direct ILEC peering:** Negotiate directly with carriers (e.g., Verizon Wholesale, AT&T Carrier Services). Lower per-minute (~$0.001-0.003/min) but $5-50K/month minimums + porting setup.

For most enterprises, aggregator is right. Above 1M minutes/month, direct peering may pay.

### Voice on Wholesale Rate Model

Tier 2/3 ITSPs purchase from Tier 1 ILECs and resell. Rate stack:

```
Tier 1 (AT&T)   ----- $0.001/min ----->   wholesale
                                            |
                                            v
Tier 2 (Bandwidth)  -- $0.003/min ---->   reseller
                                            |
                                            v
Tier 3 (Twilio)  ---- $0.0085/min --->   end-customer
                                            |
                                            v
                                       per-channel + per-minute markups
```

Each layer adds 50-200% markup. The per-minute price you pay reflects how many layers your traffic goes through.

## Compliance

### CALEA (US)

Communications Assistance for Law Enforcement Act, 47 USC 1001 et seq. Telecom carriers (including SIP trunking ITSPs) must support lawful intercept of voice + signaling.

Implications for SIP trunking:
- Your ITSP, not you, handles the law enforcement intercept process for trunk traffic.
- If you operate as a CLEC or VoIP provider yourself (offering trunks/numbers to your own customers), CALEA applies to you.

### GDPR (EU)

For calls to/from EU subjects:
- Process voice data lawfully (consent, contract, legitimate interest).
- Disclose recording at start of call ("This call is being recorded for quality purposes").
- Provide subject-access for recordings on request.
- Erase on request.
- Notify breaches within 72h.
- Data protection officer if processing systematic.

### Recording Disclosures

US states are split:
- **One-party consent (38 states + DC + federal):** Either party can record. Recording party doesn't need to disclose.
- **Two-party / all-party consent (12 states):** All parties must consent. Examples: California, Florida, Illinois, Maryland, Massachusetts, Montana, New Hampshire, Pennsylvania, Washington, plus stricter rules in Connecticut, Delaware, Oregon.

Best practice: always disclose recording. "This call may be recorded for training and quality purposes."

EU/UK: GDPR + ePrivacy. Always disclose; obtain explicit consent where the recording is used for marketing.

### Call Recording Retention

- **HIPAA:** 6 years from last touch.
- **PCI:** No requirement to record; if recorded, do not store CVV/full card number in audio.
- **FINRA / SEC:** 3 years (some) to 7 years (broker-dealers).
- **GDPR:** Storage limitation principle — keep only as long as necessary for stated purpose.

Encrypt recordings at rest. Restrict access. Log every access.

## Idioms

- "Normalize to E.164 internally" — every number stored as `+CC...` regardless of how it was dialed/displayed.
- "STIR/SHAKEN compliance is now table stakes" — don't expect calls to be "Verified" without B-attestation minimum; without it, expect "Spam Likely".
- "Always have a failover trunk" — single ITSP is single point of failure; budget at least secondary.
- "Monitor for fraud signatures" — anomalous concurrent count, off-hours international, repeated probing all should alert.
- "OPTIONS ping every 30s" — universal SIP keepalive cadence.
- "TLS+SRTP for any high-security trunk" — non-negotiable for PCI/HIPAA-adjacent flows.
- "Test outbound CLID after every config change" — easy to break the From: rewriting.
- "Always test 911 — but only with prior coordination with PSAP" — don't be the IT guy who triggered an unannounced ambulance dispatch.
- "Burn-in new trunks for 7 days before cutting over" — catch propagation issues, CNAM rejection, port glitches.
- "If the bill jumps, suspect fraud first, codec next, miscount last" — international fraud is the most common surprise charge.
- "Disable SIP-ALG on every customer firewall, every time" — it's almost always wrong, even when "working".
- "Sip + RTP separation is a feature" — the path the audio takes can differ from signaling; debug each independently.

## See Also

- asterisk
- freeswitch
- sip-protocol
- rtp-sdp
- ip-phone-provisioning
- tls

## References

- RFC 3261 — SIP: Session Initiation Protocol
- RFC 3262 — Reliability of Provisional Responses (PRACK)
- RFC 3263 — Locating SIP Servers (DNS SRV/NAPTR)
- RFC 3264 — Offer/Answer Model with SDP
- RFC 3265 — SIP-Specific Event Notification
- RFC 3325 — P-Asserted-Identity / P-Preferred-Identity / Privacy: id
- RFC 3428 — SIP MESSAGE Method
- RFC 3711 — SRTP (Secure RTP)
- RFC 4566 — SDP: Session Description Protocol
- RFC 4733 — RTP Telephone Events (DTMF over RTP)
- RFC 5359 — SIP Service Examples (Best Practices)
- RFC 5630 — Use of TLS in SIP
- RFC 6086 — SIP INFO Method
- RFC 6228 — 199 Early Dialog Terminated
- RFC 7033 — WebFinger (used in some discovery)
- RFC 8224 — Authenticated Identity Management in SIP (STIR)
- RFC 8225 — PASSporT: Personal Assertion Token (STIR)
- RFC 8226 — Secure Telephone Identity Credentials (STIR certificates)
- RFC 8588 — PASSporT Extension for SHAKEN
- ATIS-1000074 — SHAKEN Framework
- ATIS-1000080 — STI-CA / STI-PA Architecture
- ATIS-1000084 — Technical Report on SHAKEN Identity Header
- E.164 — ITU-T Recommendation, The international public telecommunication numbering plan
- E.123 — ITU-T Recommendation, Notation for national and international telephone numbers
- E.212 — ITU-T Recommendation, International identification plan for public networks
- FCC TRACED Act (2019) — Telephone Robocall Abuse Criminal Enforcement and Deterrence Act
- FCC 47 CFR Part 64 — Robocall mitigation, STIR/SHAKEN obligations
- CRTC Telecom Decision 2018-32 — Canada STIR/SHAKEN mandate
- Ofcom General Conditions — UK consumer protection telecom
- Twilio Elastic SIP Trunking docs — https://www.twilio.com/docs/sip-trunking
- Bandwidth.com Voice docs — https://dev.bandwidth.com/docs/voice/
- Telnyx Voice API docs — https://developers.telnyx.com/docs/voice/programmable-voice/
- Vonage Voice API — https://developer.vonage.com/en/voice/voice-api/overview
- Asterisk wiki — https://wiki.asterisk.org/
- FreeSWITCH wiki — https://freeswitch.org/confluence/
- Kamailio docs — https://www.kamailio.org/wiki/
- OpenSIPS docs — https://www.opensips.org/Documentation/
- IETF STIR WG — https://datatracker.ietf.org/wg/stir/
- ATIS — https://www.atis.org/
- Somos (RespOrg / SMS-800) — https://www.somos.com/
- iconectiv (LERG / STI-PA) — https://iconectiv.com/
- TCR (Campaign Registry, A2P 10DLC) — https://www.campaignregistry.com/
