# MACsec — IEEE 802.1AE Media Access Control Security

> MACsec is the IEEE standard for Layer 2 hop-by-hop encryption. It provides
> confidentiality, data integrity, and data origin authentication for every
> Ethernet frame traversing a point-to-point link. Unlike IPsec (Layer 3) or
> TLS (Layer 4+), MACsec operates below the network layer, making it transparent
> to all higher-layer protocols and capable of protecting control plane traffic
> (ARP, DHCP, OSPF hellos) that upper-layer encryption cannot reach.

---

## 1. Historical Context and Motivation

The original Ethernet specification assumed a trusted physical medium. Any device
connected to the wire could read, inject, or modify frames with impunity. As campus
networks grew and switch ports became accessible to untrusted devices, several
attack vectors became practical:

- **Eavesdropping** — passive capture of cleartext frames on shared or tapped media.
- **Frame injection** — crafting rogue ARP, DHCP, or STP frames.
- **Man-in-the-middle** — inserting a transparent bridge between two legitimate peers.
- **Replay attacks** — capturing and retransmitting valid frames.

IEEE 802.1AE was ratified in 2006 (revised 2018) to address these threats at the
link layer itself. It was designed to complement 802.1X port-based access control:
802.1X authenticates the peer, and MACsec encrypts the resulting session.

The key insight of MACsec is that Layer 2 encryption, despite being hop-by-hop,
provides a security property that end-to-end encryption cannot: **protection of
the network control plane**. A MACsec-secured link prevents rogue BPDU injection,
ARP spoofing, DHCP starvation, and VLAN hopping at the hardware level.

---

## 2. MACsec Architecture

### 2.1 The SecY Entity

Every MACsec-capable port instantiates a **SecY** (Security Entity). The SecY is the
enforcement point: it intercepts outgoing frames, applies the SecTAG and encryption,
and on ingress, validates the ICV, checks the packet number for replay, and decrypts.

The SecY operates in one of three modes:

| Mode              | Outbound                    | Inbound                          |
|-------------------|-----------------------------|----------------------------------|
| Encrypt + Auth    | Encrypt payload, append ICV | Verify ICV, decrypt payload      |
| Auth Only         | Append ICV (no encryption)  | Verify ICV only                  |
| Bypass            | Pass through unmodified     | Pass through unmodified          |

Auth-only mode (integrity without confidentiality) is rarely used in practice but
is defined in the standard for scenarios where regulatory or debugging requirements
demand plaintext visibility.

### 2.2 Secure Channels and Secure Associations

A **Secure Channel (SC)** is a unidirectional construct identified by a **Secure
Channel Identifier (SCI)**, which is the concatenation of the source MAC address
and a 16-bit port number.

Each SC contains up to four **Secure Associations (SAs)**, indexed by a 2-bit
**Association Number (AN)** from 0 to 3. Only one SA is active for transmission at
any time, but multiple SAs may be active for reception during key rollover.

The SA holds:
- The **SAK** (Secure Association Key) — the symmetric key for GCM-AES.
- The **PN** (Packet Number) — a monotonically incrementing counter.
- The **lowest acceptable PN** — for replay protection on the receive side.

This four-SA design enables **hitless key rollover**: a new SAK is installed in the
next AN slot, the transmitter switches to the new AN, and the receiver accepts both
old and new AN until the old SA is explicitly retired.

### 2.3 Frame Processing Pipeline

```
Outbound:
  1. Original frame arrives at SecY.
  2. SecY inserts SecTAG after source MAC address.
  3. SecY encrypts the payload (everything after SecTAG) using SAK + PN as IV.
  4. SecY appends 16-byte ICV (GCM authentication tag).
  5. Frame is transmitted.

Inbound:
  1. Frame arrives with EtherType 0x88E5 (MACsec).
  2. SecY extracts AN from SecTAG, selects the corresponding SA.
  3. SecY checks PN against replay window.
  4. SecY verifies ICV (GCM authentication tag).
  5. SecY decrypts payload.
  6. SecY strips SecTAG and ICV, delivers original frame to upper layers.
```

Frames that fail ICV verification or replay checks are silently dropped and counted
in error statistics (`InPktsNotValid`, `InPktsLate`).

---

## 3. The SecTAG in Detail

The MACsec Security Tag (SecTAG) is inserted between the source MAC address and the
original EtherType/payload. Its presence is indicated by EtherType 0x88E5.

### 3.1 Field Breakdown

```
Byte 0:     TCI (Tag Control Information) + AN
              Bit 7:   V  (Version, must be 0)
              Bit 6:   ES (End Station — set if the SecY is an end station)
              Bit 5:   SC (SCI present in SecTAG)
              Bit 4:   SCB (Single Copy Broadcast)
              Bit 3:   E  (Encryption — 1 = payload encrypted)
              Bit 2:   C  (Changed Text — 1 = original frame modified)
              Bits 1-0: AN (Association Number, 0-3)

Byte 1:     SL (Short Length)
              If the original frame is <= 48 bytes, SL holds the original
              payload length. Otherwise SL = 0.

Bytes 2-5:  Packet Number (PN) — 32-bit unsigned integer.
              With XPN, the upper 32 bits are implicit (not in the header)
              and reconstructed by the receiver from its local counter state.

Bytes 6-13: SCI (optional, only present if TCI.SC = 1)
              Bytes 6-11: Source MAC address
              Bytes 12-13: Port Identifier
```

### 3.2 When is the SCI Present?

The SCI is explicitly included when:
- The link is point-to-multipoint (e.g., a shared medium scenario).
- The receiving SecY needs the SCI to select the correct SC.

On dedicated point-to-point links (the common case), the SCI is implicitly derived
from the source MAC and port, so TCI.SC = 0 and the SecTAG is only 8 bytes
(saving 8 bytes of overhead).

### 3.3 Total Overhead Calculation

```
Minimum overhead (no explicit SCI):
  SecTAG:  8 bytes
  ICV:    16 bytes
  Total:  24 bytes

Maximum overhead (explicit SCI):
  SecTAG:  8 bytes
  SCI:     8 bytes
  ICV:    16 bytes
  Total:  32 bytes
```

This overhead must be accounted for in MTU planning. On a standard 1500-byte Ethernet
link, the MACsec-encapsulated frame can be up to 1532 bytes. Intermediate infrastructure
(if any, though MACsec is hop-by-hop) must support this size.

---

## 4. Cipher Suites

MACsec uses AES in Galois/Counter Mode (GCM), which provides both encryption and
authentication in a single pass. GCM is defined in NIST SP 800-38D.

### 4.1 Standard Cipher Suites

**GCM-AES-128** (default):
- 128-bit SAK, 128-bit GCM authentication tag (ICV).
- 32-bit Packet Number used as part of the 96-bit IV.
- Widely supported on all MACsec-capable hardware.
- Sufficient for most campus and data center deployments.

**GCM-AES-256**:
- 256-bit SAK, same 128-bit ICV.
- Recommended for environments requiring post-quantum resistance margins or
  compliance with policies mandating 256-bit encryption (e.g., CNSA Suite).
- Hardware support is near-universal on modern ASICs.

### 4.2 Extended Packet Numbering (XPN) Cipher Suites

The 32-bit PN has a maximum value of approximately 4.3 billion. On a 100 Gbps link
sending minimum-size (64-byte) frames, PN exhaustion occurs in roughly:

```
  4.3 x 10^9 frames / (148.8 x 10^6 frames/sec) = ~29 seconds
```

At 400 Gbps, exhaustion happens in under 8 seconds. Each PN exhaustion forces a
SAK rekey, which involves MKA negotiation and a brief window of potential traffic
disruption.

**GCM-AES-XPN-128** and **GCM-AES-XPN-256** extend the PN to 64 bits:
- The upper 32 bits are maintained locally by the SecY (not transmitted in the header).
- The receiver reconstructs the full 64-bit PN using its own counter state and a
  sliding window algorithm.
- Maximum frames before rekey: ~1.8 x 10^19 — effectively infinite.
- Defined in IEEE 802.1AEbw-2013 (incorporated into 802.1AE-2018).

XPN cipher suites do **not** add any extra overhead to the frame. The upper 32 bits
are implicit, so the SecTAG remains the same size.

### 4.3 IV Construction

The 96-bit GCM IV (also called the nonce) is constructed as:

```
Standard (32-bit PN):
  IV = SCI (64 bits) || PN (32 bits)

XPN (64-bit PN):
  IV = (SCI XOR upper-32-of-PN padded to 64 bits) || lower-32-of-PN
```

The SCI component ensures uniqueness across different Secure Channels. The PN
component ensures uniqueness within a channel. The combination guarantees that
no IV is ever reused with the same key — a critical requirement for GCM security.

---

## 5. MKA — MACsec Key Agreement Protocol

MKA, defined in IEEE 802.1X-2010 (clause 11), is the control plane protocol that
establishes MACsec sessions. It runs as EAPoL (Extensible Authentication Protocol
over LAN) frames with EtherType 0x888E.

### 5.1 MKA PDU (MKPDU) Structure

MKPDUs carry parameter sets as TLVs (Type-Length-Value):

| Parameter Set                  | Purpose                                        |
|-------------------------------|------------------------------------------------|
| Basic Parameter Set            | CKN, MKA version, key server priority, SCI     |
| Live/Potential Peer List       | Track known peers for liveness detection        |
| MACsec SAK Use                 | Announce currently installed SAK, AN, PN        |
| Distributed SAK                | Carry the new SAK (wrapped with KEK)            |
| Distributed CAK                | For group CAK distribution (rare)               |
| ICV Indicator                  | Marks the end of MKPDU, carries ICV             |

### 5.2 Key Server Election

When two peers establish an MKA session, one must become the **Key Server**
(responsible for generating and distributing the SAK). Election rules:

1. The peer with the **lowest key server priority** value wins.
2. If priorities are equal, the peer with the **highest SCI** wins.
3. The key server priority is configured per interface (range 0-255, default varies by platform).

The Key Server generates a random SAK, wraps it with the KEK derived from the CAK,
and distributes it inside a Distributed SAK parameter set in an MKPDU.

### 5.3 Liveness and Timers

MKA maintains peer liveness through periodic MKPDU exchange:

- **MKA Hello interval**: 2 seconds (default). Each peer sends an MKPDU every 2 seconds.
- **MKA Lifetime**: 6 seconds. If no MKPDU is received from a peer within 6 seconds
  (3 missed hellos), the peer is declared dead and the MKA session is torn down.
- **SAK Rekey interval**: Configurable. The Key Server generates a new SAK either on
  a timer or when the PN approaches exhaustion (PN threshold).

These timers are intentionally aggressive to ensure rapid detection of link failures
and fast convergence. On a port-channel, each member link runs its own independent
MKA session.

### 5.4 MKA Session Establishment Sequence

```
Time 0s:  Peer A sends MKPDU with CKN, priority, Live Peer List (empty)
Time 0s:  Peer B sends MKPDU with CKN, priority, Live Peer List (empty)

Time 2s:  Peer A receives B's MKPDU, verifies CKN matches its CAK
          A adds B to its Live Peer List
          A sends MKPDU with B in Live Peer List

Time 2s:  Peer B receives A's MKPDU, verifies CKN
          B adds A to its Live Peer List
          B determines it is (or is not) the Key Server

Time 4s:  Key Server generates SAK, wraps with KEK
          Sends MKPDU with Distributed SAK parameter set

Time 4s:  Non-Key-Server installs SAK, sends MKPDU confirming SAK Use

Time 6s:  Both peers have SAK installed. MACsec data plane is active.
          Encrypted traffic begins flowing.
```

Total convergence time is typically 4-6 seconds from link-up to encrypted traffic.

---

## 6. Key Hierarchy and Derivation

### 6.1 Connectivity Association Key (CAK)

The CAK is the root of the MACsec key hierarchy. It is never used directly for
encryption. Instead, it serves as input to key derivation functions (KDFs) that
produce the operational keys.

**PSK (Pre-Shared Key) Mode:**
The administrator configures a CAK (128 or 256 bits, represented as 32 or 64 hex
characters) and a CKN (up to 64 hex characters) on both ends of the link.

**EAP Mode (802.1X):**
The CAK is derived from the MSK (Master Session Key) produced by the EAP method
(e.g., EAP-TLS, EAP-FAST). The first 16 bytes of the MSK become the CAK (for
128-bit) and the first 16 bytes of the MSK become the CKN. For 256-bit CAK, the
first 32 bytes of the MSK are used.

### 6.2 Derived Keys

From the CAK, two keys are derived using the KDF defined in 802.1X-2010:

**KEK (Key Encrypting Key):**
- Derived as: KEK = KDF(CAK, label="IEEE8021 KEK", context, length)
- Used to AES-KeyWrap the SAK for distribution in MKPDUs.
- Length matches the CAK length (128 or 256 bits).

**ICK (ICV Key):**
- Derived as: ICK = KDF(CAK, label="IEEE8021 ICK", context, length)
- Used to compute the ICV (CMAC) over each MKPDU for integrity verification.
- Ensures that MKPDUs cannot be tampered with by an attacker who does not possess the CAK.

### 6.3 Secure Association Key (SAK)

The SAK is a randomly generated key (128 or 256 bits, matching the chosen cipher suite)
produced by the Key Server. It is the actual key used by GCM-AES to encrypt and
authenticate Ethernet frames.

- The SAK is wrapped (encrypted) with the KEK using AES Key Wrap (RFC 3394).
- The wrapped SAK is distributed inside an MKPDU.
- Only peers possessing the correct KEK (derived from the matching CAK) can unwrap the SAK.
- SAK lifetime is tied to PN exhaustion or a configured rekey interval.

---

## 7. PSK vs 802.1X-Derived Key: Detailed Comparison

### 7.1 Pre-Shared Key (PSK) Deployments

PSK is the dominant mode for **infrastructure links**: switch-to-switch,
switch-to-router, and data center fabric interconnects.

Advantages:
- No dependency on external authentication infrastructure (RADIUS, PKI).
- Deterministic — the session will establish as long as CKN/CAK match.
- Simpler to troubleshoot (fewer moving parts).

Disadvantages:
- Key distribution is manual and error-prone.
- Key rotation requires configuration changes on both ends.
- Scalability is limited: N links require N distinct CKN/CAK pairs.

Best practices for PSK:
- Use unique CKN/CAK per link (never reuse across links).
- Store keys in a secrets manager; never in version-controlled config files.
- Implement fallback keychains for hitless key rotation.
- Use 256-bit CAKs for all new deployments.

### 7.2 802.1X-Derived Key Deployments

EAP-derived MACsec is used primarily for **access ports** in campus networks,
where endpoints (laptops, IP phones, printers) authenticate via 802.1X and the
RADIUS server signals MACsec policy in the authorization response.

The RADIUS server (e.g., Cisco ISE) includes attributes in the Access-Accept:
- `Tunnel-Type = VLAN`
- `Tunnel-Medium-Type = 802`
- `Tunnel-Private-Group-ID = <vlan>`
- `cisco-av-pair = linksec-policy=must-secure` (or `should-secure`)

The MSK from the EAP exchange becomes the CAK. The supplicant and authenticator
both derive KEK and ICK, elect a Key Server, and establish the MACsec session.

Advantages:
- Automatic key management — no manual key distribution.
- Per-session keys — each authentication produces a unique CAK.
- Scales to thousands of endpoints.

Disadvantages:
- Requires RADIUS infrastructure and PKI (for EAP-TLS).
- More complex troubleshooting path (RADIUS, EAP, MKA, MACsec).
- Supplicant software must support MACsec (not all do).

---

## 8. Security Policies: should-secure vs must-secure

### 8.1 must-secure

When the policy is `must-secure`, the SecY will **drop all traffic** if a MACsec
session cannot be established. This provides the strongest security guarantee:
no cleartext frames will ever traverse the link.

Use must-secure when:
- The link carries sensitive data and cleartext is never acceptable.
- Both ends are known to support MACsec.
- You have tested and validated the MACsec configuration.

### 8.2 should-secure

When the policy is `should-secure`, the SecY will **attempt** MACsec but fall back
to cleartext if the peer does not support it or MKA negotiation fails.

Use should-secure when:
- Migrating a network to MACsec incrementally.
- The peer may or may not support MACsec (mixed environment).
- You want to avoid link-down events due to MACsec misconfiguration.

### 8.3 Migration Strategy

A recommended approach for deploying MACsec on an existing network:

```
Phase 1: Enable MKA with should-secure on all links.
         Monitor: show mka sessions — verify sessions establish.
         Duration: 1-2 weeks in production.

Phase 2: Identify links where MKA did not establish.
         Investigate: hardware support, firmware version, configuration.
         Resolve issues.

Phase 3: Switch all validated links to must-secure.
         Monitor for drops: show macsec statistics — check InPktsNoSA.

Phase 4: Final audit — ensure no link is still in should-secure.
```

---

## 9. MACsec on WAN and Overlay Networks

### 9.1 The Hop-by-Hop Limitation

Standard MACsec encrypts at Layer 2 between directly connected peers. Every
intermediate switch must decrypt, inspect (for forwarding decisions), and re-encrypt.
This means:

- Every hop in the path must support MACsec.
- Every hop must participate in MKA and hold the SAK.
- The security perimeter is only as strong as the weakest hop.

For campus and data center **leaf-spine** topologies, this is acceptable because all
infrastructure is trusted and under the same administrative domain.

For **WAN** links traversing service provider networks, standard MACsec is not feasible
because the provider equipment is not under your control.

### 9.2 WAN MACsec

WAN MACsec is used on dedicated point-to-point circuits (dark fiber, DWDM wavelengths,
Metro Ethernet) where there is no intermediate L2 device between your routers.

Platforms: Cisco ASR 1000, Catalyst 8500, Juniper MX Series, Nokia 7750.

The key difference from LAN MACsec is that WAN MACsec often runs on **routed interfaces**
(not switchports). The router encrypts the Ethernet frame on egress and decrypts on
ingress, even though the interface is operating at Layer 3.

MTU is critical: the circuit must support at least 1532-byte frames (or jumbo frames
if the inner MTU is already 9000+).

### 9.3 CloudSec (Cisco Nexus 9000)

CloudSec extends MACsec protection over VXLAN fabrics spanning multiple sites
connected via IP transport (MPLS, internet, SD-WAN).

Key differences from standard MACsec:

| Aspect            | Standard MACsec              | CloudSec                           |
|-------------------|------------------------------|------------------------------------|
| Scope             | Hop-by-hop L2                | Site-to-site over L3 underlay      |
| Encrypted region  | Everything after SecTAG      | VXLAN payload only                 |
| Outer headers     | Encrypted                    | Cleartext (allows IP forwarding)   |
| Key management    | MKA                          | MKA (adapted for tunnel endpoints) |
| Platform          | Any MACsec-capable           | Nexus 9300-GX, 9400, 9500-GX      |

CloudSec inserts the SecTAG and ICV around the VXLAN-encapsulated payload, leaving
the outer Ethernet, IP, and UDP headers in cleartext. This allows intermediate
routers (which may be provider equipment) to forward the packet based on the outer
IP header without needing MACsec capability.

### 9.4 MACsec and VXLAN Interaction

When standard MACsec protects a VXLAN underlay link:

```
Original overlay frame:
  [Inner Eth | Inner IP | Inner Payload]

After VXLAN encapsulation:
  [Outer Eth | Outer IP | UDP:4789 | VXLAN Hdr | Inner Eth | Inner IP | Inner Payload]

After MACsec encryption (standard, on underlay link):
  [Outer Eth (clear) | SecTAG | {Outer IP | UDP:4789 | VXLAN Hdr | Inner frame} encrypted | ICV]
```

The entire VXLAN packet (including outer IP) is encrypted. This means MACsec must be
terminated at every hop — each spine switch decrypts, makes a forwarding decision on
the outer IP, and re-encrypts on the egress port.

With CloudSec, only the inner payload is encrypted, and the outer IP remains readable
for routing. This is the critical architectural difference.

---

## 10. Hardware Considerations

### 10.1 ASIC Requirements

MACsec encryption and decryption must happen at line rate. This requires dedicated
hardware in the port ASIC or PHY. Software-based MACsec is not practical for
production network equipment.

Modern ASICs (e.g., Memory and forwarding pipeline on Memory and Broadcom Memory
Memory) integrate MACsec engines directly into the MAC block, operating inline
with no added latency beyond the encryption pipeline (typically < 1 microsecond).

### 10.2 Platform-Specific Notes

**Cisco Catalyst 9000 Series:**
- UADP 2.0 and later ASICs support MACsec on all ports.
- Both 128-bit and 256-bit cipher suites supported.
- XPN supported on Catalyst 9500 and 9400 with appropriate supervisors.
- Configuration uses `mka policy` and `macsec` interface commands.

**Cisco Nexus 9000 Series:**
- Broadcom-based linecards (N9K-X97160YC-EX and later) support MACsec.
- CloudSec requires -GX or later linecards with dedicated crypto engines.
- NX-OS configuration uses `feature macsec`, `macsec policy`, and `macsec keychain`.
- Supports fallback keychains natively.

**Linux / Intel NICs:**
- Intel X710 and E810 NICs support MACsec offload.
- Linux kernel 4.6+ includes the `macsec` module for software-based MACsec.
- `ip macsec` (from iproute2) is the configuration tool.
- `wpa_supplicant` provides MKA for Linux-based endpoints.
- Performance is limited compared to switch ASICs; suitable for endpoints, not switches.

### 10.3 NIC MACsec Offload (Linux)

```bash
# Create a MACsec interface on top of eth0
ip link add link eth0 macsec0 type macsec sci 0x0011223344550001 \
    cipher gcm-aes-128 encrypt on

# Add a receive channel
ip macsec add macsec0 rx sci 0xAABBCCDDEEFF0001

# Add a receive SA
ip macsec add macsec0 rx sci 0xAABBCCDDEEFF0001 sa 0 \
    pn 1 on key 00 <32-hex-char-key>

# Add a transmit SA
ip macsec add macsec0 tx sa 0 pn 1 on key 00 <32-hex-char-key>

# Bring the interface up
ip link set macsec0 up

# Verify
ip macsec show
```

For production Linux deployments, use `wpa_supplicant` with MKA for automatic key
management rather than static key configuration.

---

## 11. MTU Planning

MACsec adds overhead that must be accommodated end-to-end.

### 11.1 Overhead Scenarios

```
Scenario 1: Point-to-point, no explicit SCI
  Overhead = 8 (SecTAG) + 16 (ICV) = 24 bytes
  Original MTU 1500 -> Required link MTU: 1524

Scenario 2: Point-to-point, explicit SCI
  Overhead = 8 (SecTAG) + 8 (SCI) + 16 (ICV) = 32 bytes
  Original MTU 1500 -> Required link MTU: 1532

Scenario 3: VXLAN + MACsec
  VXLAN overhead:   50 bytes (14 outer Eth + 20 IP + 8 UDP + 8 VXLAN)
  MACsec overhead:  32 bytes (worst case)
  Original MTU 1500 -> Required link MTU: 1582

Scenario 4: Jumbo frames + MACsec
  Original MTU 9000 -> Required link MTU: 9032
  Most switches support 9216 MTU, so this is fine.
```

### 11.2 Practical Recommendations

- Set the physical interface MTU to at least `desired_L3_MTU + 32`.
- For VXLAN fabrics, the underlay MTU should be at least 1600 (many operators use 9216).
- Test with `ping -s <size> -M do` (Linux) or `ping -f -l <size>` (Windows) to verify
  end-to-end MTU after enabling MACsec.
- MACsec overhead is **not** counted in the Ethernet FCS — the FCS is recalculated
  over the entire MACsec-encapsulated frame.

---

## 12. Troubleshooting Deep Dive

### 12.1 MKA Session Will Not Establish

Root causes (in order of likelihood):

1. **CKN mismatch** — the CKN is the lookup key for the CAK. If CKNs do not match,
   peers will never recognize each other. Verify with `show mka sessions` (should show
   the CKN) and compare both sides character by character.

2. **CAK mismatch** — CKNs match but the actual key material differs. The MKPDU ICV
   will fail verification. Look for `show mka statistics` -> ICV verification failures.

3. **MKA policy mismatch** — different cipher suites, confidentiality offsets, or
   `include-icv-indicator` settings. Both ends must agree.

4. **Connectivity** — EAPoL frames (0x888E) must not be blocked by intermediate
   equipment. Verify with packet capture.

5. **Clock skew** — if using lifetime-based key activation, clocks must be synchronized.
   Use NTP.

### 12.2 MKA Session Flapping

Symptoms: session cycles between Secured and Init states.

Causes:
- **Physical layer issues** — CRC errors, optical power out of spec, bad cable. MACsec
  amplifies the impact of physical errors because any bit flip causes ICV failure.
- **MKA timer mismatch** — one side sends hellos every 2s, the other expects them more
  frequently. Standardize MKA policy on both ends.
- **Software bug** — check platform release notes for known MKA stability issues.
- **CPU overload** — MKA runs in the control plane (CPU). If the CPU is overloaded,
  MKPDU processing is delayed, and the peer declares timeout.

### 12.3 Traffic Drops After MACsec Enabled

Symptoms: MKA session is Secured but traffic is black-holed.

Causes:
- **MTU exceeded** — MACsec overhead pushes frames beyond the link MTU. Check for
  `OutPktsToolong` in `show macsec statistics`. Increase MTU.
- **Cipher suite mismatch** — one side encrypts with AES-256, the other expects AES-128.
  Both sides silently discard frames they cannot decrypt.
- **SA not installed** — the SAK was distributed but the receiving side has not installed
  the SA yet. Check `show macsec interface` for active SA count.
- **Port channel hash asymmetry** — frames arrive on a different member link than expected.
  Enable `replay-protection window-size` > 0.
- **Middlebox interference** — a TAP, inline IDS, or cable patch panel is modifying frame
  bytes, causing ICV failure. MACsec is incompatible with any device that alters frame
  content between the two SecY endpoints.

### 12.4 Key Diagnostic Commands

**IOS-XE full diagnostic sequence:**

```
! Step 1: Session overview
show mka sessions
! Expected: Status = Secured, CKN matches, Peer present

! Step 2: Detailed session state
show mka sessions detail
! Expected: Key Server identified, SAK AN and PN visible, TX/RX SC active

! Step 3: MKA statistics (error counters)
show mka statistics
! Watch: MKPDUs received/validated, ICV verification failures, SAK failures

! Step 4: MACsec data plane state
show macsec interface <intf>
! Expected: Cipher negotiated, TX SA installed, RX SA installed

! Step 5: Frame counters
show macsec statistics interface <intf>
! Expected: InPktsOK and OutPktsEncrypted incrementing
! Red flags: InPktsNotValid, InPktsNoSA, InPktsLate, OutPktsTooLong

! Step 6: Key chain verification
show key chain macsec
! Expected: Active key with matching CKN, correct algorithm
```

**NX-OS full diagnostic sequence:**

```
! Step 1: Feature status
show feature | include macsec

! Step 2: Policy verification
show macsec policy

! Step 3: Session state
show macsec mka session
show macsec mka session detail interface <intf>

! Step 4: Statistics
show macsec mka statistics interface <intf>
show macsec secy statistics interface <intf>

! Step 5: Internal debug
show system internal macsec info
```

---

## 13. Comparison: MACsec vs IPsec vs TLS

| Property                | MACsec (802.1AE)        | IPsec (ESP/AH)           | TLS 1.3                  |
|------------------------|-------------------------|--------------------------|--------------------------|
| OSI Layer              | 2                       | 3                        | 4+ (session)             |
| Scope                  | Hop-by-hop              | End-to-end (tunnel/transport) | End-to-end (application) |
| Protects control plane | Yes (ARP, DHCP, STP)    | No                       | No                       |
| MTU impact             | +24-32 bytes            | +50-70 bytes (tunnel)    | +5-40 bytes (record)     |
| Key exchange           | MKA (EAPoL)             | IKEv2 (UDP 500/4500)     | TLS handshake            |
| Hardware acceleration  | Required (ASIC/PHY)     | Common (crypto offload)  | Common (AES-NI)          |
| Multicast support      | Native                  | Complex (GRE+IPsec)      | Not applicable           |
| Deployment complexity  | Low-Medium              | Medium-High              | Low (application-level)  |
| Typical use case       | LAN/DC fabric           | Site-to-site VPN, remote access | Web, API, email          |

---

## Prerequisites

Before deploying MACsec, ensure:

- **Hardware compatibility** — all ports in the MACsec domain must have hardware crypto support.
  Check platform datasheets and release notes.
- **Firmware/software version** — MACsec features (especially XPN and CloudSec) require
  minimum software versions. Verify against vendor compatibility matrices.
- **MTU headroom** — all links must support at least 24-32 bytes above the desired L3 MTU.
- **NTP synchronization** — required for key lifetime scheduling and consistent logging.
- **Key management plan** — decide PSK vs 802.1X, key rotation schedule, and fallback strategy
  before deployment.
- **RADIUS infrastructure (if 802.1X)** — ISE or FreeRADIUS with MACsec authorization profiles.
- **Change control** — MACsec misconfiguration on `must-secure` links causes immediate outage.
  Always test in lab and deploy during maintenance windows.

---

## References

- IEEE 802.1AE-2018 — Media Access Control (MAC) Security
  https://standards.ieee.org/standard/802_1AE-2018.html
- IEEE 802.1X-2010 — Port-Based Network Access Control
  https://standards.ieee.org/standard/802_1X-2010.html
- IEEE 802.1AEbw-2013 — Extended Packet Numbering
  https://standards.ieee.org/standard/802_1AEbw-2013.html
- NIST SP 800-38D — Recommendation for Block Cipher Modes of Operation: GCM
  https://csrc.nist.gov/publications/detail/sp/800-38d/final
- RFC 3394 — Advanced Encryption Standard (AES) Key Wrap Algorithm
  https://www.rfc-editor.org/rfc/rfc3394
- Cisco IOS-XE MACsec Configuration Guide
  https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/macsec/configuration/xe-17/macsec-xe-17-book.html
- Cisco Nexus 9000 NX-OS Security Configuration Guide — MACsec
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/103x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-103x/m-configuring-macsec.html
- Cisco CloudSec Encryption Design Guide
  https://www.cisco.com/c/en/us/products/collateral/switches/nexus-9000-series-switches/white-paper-c11-740512.html
