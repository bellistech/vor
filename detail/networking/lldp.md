# LLDP Deep Dive -- Frame Encoding, Topology Discovery & Security

> *LLDP is the simplest useful Layer 2 discovery protocol: no handshake, no state machine, no acknowledgment. A device periodically multicasts a single frame describing itself, and every directly connected neighbor listens. The elegance lies in the TLV encoding that makes the protocol infinitely extensible while keeping mandatory overhead under 30 bytes.*

---

## 1. LLDPDU Frame Format

### Ethernet Encapsulation

LLDP frames use a dedicated EtherType and a reserved multicast destination address. They are never forwarded by 802.1D-compliant bridges.

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|        Destination MAC: 01:80:C2:00:00:0E  (6 octets)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|              Source MAC  (6 octets)                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         EtherType = 0x88CC    |         LLDPDU ...            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   ... TLV sequence ...                         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           End of LLDPDU TLV (0x0000)        |       FCS       |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Three multicast destination addresses are defined by IEEE 802.1AB:

| Address             | Scope                        | Use Case                |
|:--------------------|:-----------------------------|:------------------------|
| 01:80:C2:00:00:0E   | Nearest Bridge               | Standard LLDP           |
| 01:80:C2:00:00:03   | Nearest Non-TPMR Bridge      | Provider bridges        |
| 01:80:C2:00:00:00   | Nearest Customer Bridge      | Customer bridges (C-VLAN)|

Each address defines a different forwarding scope. The nearest-bridge address (0E) is the most commonly used and is consumed by the first bridge in the path.

---

## 2. TLV Encoding Details

### TLV Header Structure

Every TLV begins with a 16-bit header packed into two octets:

```
Bits:  15 14 13 12 11 10 9  8  7  6  5  4  3  2  1  0
      +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+
      |    Type (7 bits)     |      Length (9 bits)       |
      +--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+--+

Type:   0-127 — only 0-8 and 127 are defined by the standard
Length: 0-511 — length of the Value field in octets (not including header)
```

### Encoding Rules

- The first TLV must be Chassis ID (type 1).
- The second TLV must be Port ID (type 2).
- The third TLV must be TTL (type 3).
- The last TLV must be End of LLDPDU (type 0, length 0).
- Optional TLVs (types 4-8) may appear in any order after TTL.
- Multiple instances of optional TLVs are allowed for types 4-8.
- Type 127 (organizationally-specific) may appear zero or more times.

### Mandatory TLV Encoding

**Chassis ID (Type 1)**

```
Subtype  Description            Example Value
──────────────────────────────────────────────────
1        Chassis component      Entitiy MIB value
2        Interface alias        "ge-0/0/0"
3        Port component         Entity MIB value
4        MAC address            00:11:22:33:44:55 (6 octets)
5        Network address        Family(1) + IPv4 (4 octets)
6        Interface name          "eth0"
7        Locally assigned        Free-form string
```

The most common subtype is 4 (MAC address), where the value is exactly 7 octets: 1 octet subtype + 6 octets MAC.

**Port ID (Type 2)**

```
Subtype  Description            Example Value
──────────────────────────────────────────────────
1        Interface alias        "GigabitEthernet0/1"
2        Port component         Entity MIB value
3        MAC address            Port MAC (6 octets)
4        Network address        Family(1) + IP
5        Interface name         "eth0"
6        Agent circuit ID       DHCP option 82 style
7        Locally assigned       Free-form string
```

**TTL (Type 3)**

Fixed 2-octet unsigned integer. Value 0 signals the receiver to immediately purge all information associated with this MSAP (MAC Service Access Point) identifier.

$$TTL = tx\text{-}interval \times tx\text{-}hold$$

Default: $30 \times 4 = 120$ seconds. Maximum: 65535 seconds (~18 hours).

**End of LLDPDU (Type 0)**

Always encoded as exactly 2 octets: 0x0000. Type = 0, Length = 0, no value field.

### Minimum Valid LLDPDU Size

```
Chassis ID TLV:  2 (header) + 7 (subtype + MAC)   =  9 octets
Port ID TLV:     2 (header) + 7 (subtype + MAC)   =  9 octets
TTL TLV:         2 (header) + 2 (uint16)           =  4 octets
End TLV:         2 (header)                        =  2 octets
                                              Total: 24 octets

With Ethernet header (14) and FCS (4):
Minimum LLDP frame = 14 + 24 + 4 = 42 octets

Note: Ethernet minimum frame is 64 octets, so padding is applied.
Actual minimum LLDP frame on the wire = 64 octets.
```

---

## 3. LLDP-MED Network Policy TLV

### Purpose

LLDP-MED (Media Endpoint Discovery), defined by TIA-1057, extends LLDP for VoIP and media endpoints. The Network Policy TLV is the most operationally significant extension -- it tells an IP phone which VLAN to use and how to mark traffic.

### Network Policy TLV Format (LLDP-MED Type 2)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
| App Type (8)  |U|T| VLAN ID (12 bits) | L2 Pri(3)|DSCP (6)  |X|
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

App Type:   1=Voice, 2=Voice Signaling, 3=Guest Voice,
            4=Guest Voice Signaling, 5=Softphone Voice,
            6=Video Conferencing, 7=Streaming Video,
            8=Video Signaling
U flag:     Unknown policy (1 = policy not defined)
T flag:     Tagged (1 = VLAN tagged, 0 = untagged/priority)
VLAN ID:    802.1Q VLAN identifier (0-4094)
L2 Priority: 802.1p CoS value (0-7)
DSCP:       DiffServ Code Point (0-63)
```

### VoIP VLAN Assignment Sequence

```
Phase 1: Discovery
  Phone ──LLDP (no MED)──> Switch
  Phone advertises: Chassis ID, Port ID, TTL

Phase 2: Switch Identifies MED Endpoint
  Switch ──LLDP + LLDP-MED──> Phone
  Switch sends: Network Policy TLV (VLAN 100, CoS 5, DSCP 46/EF)

Phase 3: Phone Reconfigures
  Phone applies VLAN tag 100 to voice traffic
  Phone marks with 802.1p = 5, DSCP = 46 (EF)

Phase 4: Operational
  Phone ──tagged VLAN 100──> Switch (voice traffic)
  Phone ──untagged──> Switch (data, if PC connected behind phone)
```

### LLDP-MED Device Classes

| Class | Description             | Example              |
|:------|:------------------------|:---------------------|
| I     | Generic Endpoint        | IP communicator      |
| II    | Media Endpoint          | IP phone, video unit |
| III   | Communication Device    | IP PBX, media gateway|
| N/A   | Network Connectivity    | Switch, router, AP   |

Class II and III devices must support all mandatory LLDP-MED TLVs. Class I devices may support a subset.

---

## 4. Topology Discovery Algorithms

### Building a Neighbor Table

Each LLDP-capable device maintains a local Management Information Base (MIB) populated from received LLDPDUs. The key identifier is the MSAP (MAC Service Access Point), composed of:

$$MSAP = (Chassis\ ID_{TLV},\ Port\ ID_{TLV})$$

Each unique MSAP represents one remote port on one remote chassis. The receiving device stores all TLVs associated with that MSAP and refreshes on each received LLDPDU.

### Aging and Purging

When no LLDPDU is received for a given MSAP within the TTL period, the entry is purged:

$$t_{purge} = t_{last\_received} + TTL_{value}$$

If the remote device sends a TTL of 0 (shutdown LLDPDU), the entry is immediately purged regardless of when the last frame was received.

### Network-Wide Topology Construction

An NMS constructs the full network topology by polling every device's LLDP remote table (via SNMP or CLI):

```
For each device D in network:
    For each entry E in D.lldpRemTable:
        Link = (D.chassisId, E.localPortId) <---> (E.remChassisId, E.remPortId)
        Add Link to topology graph

Result: undirected graph where
    Vertices = unique chassis IDs
    Edges    = discovered LLDP adjacencies
```

This produces an accurate Layer 2 physical topology. Since LLDP frames are not forwarded, every discovered link is a direct physical (or logical) connection.

### Topology Completeness

For a network of $N$ devices each with $P$ ports, the maximum number of links:

$$L_{max} = \frac{N \times P}{2}$$

LLDP can only discover links where both endpoints run LLDP. Coverage percentage:

$$Coverage = \frac{L_{discovered}}{L_{actual}} \times 100\%$$

Gaps occur when endpoints do not run LLDP (printers, embedded devices, unmanaged switches).

---

## 5. LLDP vs CDP Feature Comparison

### Protocol Characteristics

| Feature                  | LLDP (IEEE 802.1AB)             | CDP (Cisco)                     |
|:-------------------------|:--------------------------------|:--------------------------------|
| Standard body            | IEEE (open)                     | Cisco (proprietary)             |
| First published          | 2005                            | ~1994                           |
| EtherType                | 0x88CC                          | LLC/SNAP                        |
| Multicast address        | 01:80:C2:00:00:0E               | 01:00:0C:CC:CC:CC               |
| Default transmit interval| 30 seconds                      | 60 seconds                      |
| Default hold time        | 120 seconds (30 * 4)            | 180 seconds (60 * 3)            |
| Frame format             | TLV sequence                    | TLV sequence                    |
| Extensibility            | Org-specific TLVs (type 127)    | Cisco-defined TLVs only         |
| VoIP support             | LLDP-MED (TIA-1057)             | Native CDP TLVs                 |
| PoE negotiation          | 802.3at/bt + LLDP-MED           | CDP power TLVs                  |
| Multi-vendor             | Yes                             | Cisco and compatible only       |
| SNMP MIB                 | IEEE8021-LLDP-MIB               | CISCO-CDP-MIB                   |
| IPv6 support             | Native (management address TLV) | CDP v2                          |

### CDP TLV Types (for comparison)

```
Type   CDP TLV Name              LLDP Equivalent
───────────────────────────────────────────────────────
0x0001 Device ID                 Chassis ID (type 1)
0x0002 Addresses                 Management Address (type 8)
0x0003 Port ID                   Port ID (type 2)
0x0004 Capabilities              System Capabilities (type 7)
0x0005 Software Version          System Description (type 6)
0x0006 Platform                  System Description (type 6)
0x000E TTL                       TTL (type 3)
0x000A Native VLAN               Org-specific (802.1, subtype 1)
0x0010 Power Available           LLDP-MED Extended Power
0x0011 VoIP VLAN Reply           LLDP-MED Network Policy
```

### Migration Considerations

Running both protocols simultaneously is the recommended migration path. Most modern network operating systems support coexistence:

- Cisco IOS/NX-OS: `lldp run` enables LLDP alongside CDP.
- Juniper Junos: LLDP enabled by default; CDP requires explicit configuration.
- Arista EOS: Both supported simultaneously.
- Linux (lldpd): CDP reception/transmission configurable via `lldpcli`.

Bandwidth overhead for both: approximately 2 extra frames per port per minute, negligible on any link speed.

---

## 6. OpenLLDP and Alternative Implementations

### Open-Source LLDP Implementations

| Project   | Language | Focus                              | Status      |
|:----------|:---------|:-----------------------------------|:------------|
| lldpd     | C        | Full LLDP + CDP + EDP + SONMP      | Active      |
| lldpad    | C        | LLDP + DCB (Data Center Bridging)  | Maintenance |
| OpenLLDP  | C        | Reference implementation           | Dormant     |
| systemd   | C        | networkd has basic LLDP reception   | Active      |

### lldpd Architecture

```
lldpd (daemon)
  |
  +-- Raw socket per interface (AF_PACKET, ETH_P_ALL)
  |     Filters: LLDP multicast, CDP multicast, EDP, SONMP
  |
  +-- TLV encoder/decoder
  |     Builds and parses LLDPDU frames
  |
  +-- Neighbor database (in-memory)
  |     Indexed by MSAP (chassis ID + port ID)
  |     Entries aged by TTL
  |
  +-- Unix domain socket (/var/run/lldpd.socket)
  |     lldpctl connects here for CLI queries
  |
  +-- SNMP AgentX sub-agent (optional)
        Exposes LLDP-MIB to snmpd
```

### systemd-networkd LLDP

systemd-networkd includes a minimal LLDP receiver (no transmitter by default):

```ini
# /etc/systemd/network/50-eth0.network
[Network]
LLDP=yes              # receive and decode LLDP
EmitLLDP=yes           # transmit LLDP (systemd 248+)

# View received LLDP data
networkctl lldp
```

---

## 7. Security Considerations

### Threat Model

LLDP operates without any authentication or integrity mechanism. Every frame is trusted implicitly by the receiver.

**Attack: LLDP Spoofing**

An attacker connected to a switch port can forge LLDP frames to:

1. **Impersonate a switch** -- Claim to be a different chassis, causing NMS topology maps to show false links.
2. **VLAN hopping via LLDP-MED** -- A rogue device sends LLDP-MED capabilities, prompting the switch to assign it to a voice VLAN it should not access.
3. **Information leakage** -- A passive listener on a port receives system name, management IP, OS version, and capabilities of the connected switch, aiding reconnaissance.
4. **Denial of service** -- Flooding a switch with LLDP frames from thousands of spoofed MSAPs can exhaust the neighbor table.

### LLDP Neighbor Table Exhaustion

The maximum number of remote entries a switch can store is implementation-dependent. For a table limited to $M$ entries:

$$Flood\ rate = \frac{M}{TTL_{min}} \text{ unique MSAPs/second to maintain saturation}$$

With $M = 8192$ and $TTL_{min} = 120$s:

$$Rate = \frac{8192}{120} \approx 69 \text{ unique MSAPs/second}$$

This is trivially achievable with a single host, making table exhaustion a practical concern.

### Mitigations

| Mitigation                      | Mechanism                                                |
|:--------------------------------|:---------------------------------------------------------|
| 802.1X port authentication      | Only authenticated devices can send/receive on the port  |
| LLDP per-port disable           | Disable LLDP on untrusted/edge ports                    |
| LLDP admin status rx-only       | Receive LLDP but do not transmit (limits info leakage)   |
| Neighbor table limits           | Cap entries per port (most managed switches support this)|
| LLDP rate limiting              | Limit LLDP frames processed per second per port          |
| Network segmentation            | Isolate management plane from user-accessible ports      |
| SNMP trap on LLDP changes       | Alert on unexpected neighbor changes (topology mutation) |

### LLDP vs CDP Security

Neither protocol offers authentication. CDP has had specific CVEs (buffer overflows in CDP parsing on Cisco devices, e.g., CDPwn -- CVE-2020-3110, CVE-2020-3119). LLDP implementations have had fewer high-severity CVEs, partly due to simpler parsing and the 9-bit length field limiting TLV size to 511 bytes.

---

## 8. Bandwidth and Scaling Analysis

### Per-Port LLDP Overhead

With default settings (30s interval, ~100-byte typical LLDPDU):

$$Bandwidth_{per\_port} = \frac{100 \times 8}{30} \approx 27 \text{ bits/sec}$$

For a 48-port switch:

$$Bandwidth_{total} = 48 \times 27 \approx 1,296 \text{ bits/sec} \approx 1.3 \text{ kbps}$$

LLDP overhead is negligible even on 10 Mbps links.

### Neighbor Table Memory

Each LLDP neighbor entry stores approximately 500-2000 bytes depending on TLV count. For a large campus switch with 48 ports:

$$Memory_{max} = 48 \times 2000 = 96 \text{ KB}$$

For a data center spine switch with 64 ports and one neighbor per port:

$$Memory_{max} = 64 \times 2000 = 128 \text{ KB}$$

### NMS Polling Scalability

An NMS polling $N$ devices every $T$ seconds via SNMP:

$$SNMP\_polls/sec = \frac{N}{T}$$

For 5000 devices polled every 300 seconds:

$$Rate = \frac{5000}{300} \approx 17 \text{ polls/sec}$$

Each poll retrieves the lldpRemTable, which scales with the number of neighbors per device. For an average of 24 neighbors per device, with ~10 SNMP varbinds per neighbor:

$$Varbinds/sec = 17 \times 24 \times 10 = 4,080$$

Well within the capacity of modern NMS platforms.

---

## 9. Summary of Key Values

| Parameter           | Default       | Range           | Notes                          |
|:--------------------|:--------------|:----------------|:-------------------------------|
| Tx interval         | 30 s          | 1-3600 s        | msgTxInterval                  |
| Tx hold multiplier  | 4             | 1-100           | msgTxHold                      |
| TTL                 | 120 s         | 1-65535 s       | txInterval * txHold            |
| Reinit delay        | 2 s           | 1-10 s          | Delay before re-init after disable |
| Tx delay            | 2 s           | 1-8192 s        | Min delay between successive frames |
| Notification interval | 5 s         | 1-3600 s        | Min interval between SNMP traps |
| Max neighbors/port  | Impl-specific | Typically 4-128 | Limits table exhaustion        |

## Prerequisites

- Ethernet framing, 802.1Q VLANs, multicast addressing, SNMP MIBs, QoS (DSCP/CoS)

## Complexity

| Operation                     | Time         | Space               |
|:------------------------------|:-------------|:---------------------|
| TLV parsing (per frame)       | $O(n)$ TLVs | $O(1)$ per TLV       |
| Neighbor lookup by MSAP       | $O(1)$ hash  | $O(N)$ entries       |
| Neighbor aging scan           | $O(N)$       | $O(1)$               |
| Topology graph construction   | $O(D \times P)$ | $O(V + E)$ graph |
| NMS full topology poll        | $O(D)$ sequential | $O(D \times P)$ table data |

Where $n$ = TLV count per frame, $N$ = total neighbors, $D$ = devices, $P$ = ports per device, $V$ = vertices, $E$ = edges.

---

*LLDP succeeds not through complexity but through constraint. By limiting scope to single-hop, unidirectional, unauthenticated announcements, it avoids every protocol state machine pitfall and scales to networks of any size with near-zero overhead. The cost of that simplicity is trust: every LLDP frame is believed without question, making network segmentation and port security the only real defenses.*
