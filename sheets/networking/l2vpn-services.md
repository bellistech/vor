# L2VPN Services

Layer 2 VPN technologies that extend Ethernet segments across an MPLS or IP backbone, providing point-to-point (VPWS), multipoint (VPLS), and next-generation (EVPN) connectivity while preserving customer MAC addressing and VLAN semantics.

## Concepts

### L2VPN Taxonomy

| Service | Type | Signaling | Use Case |
|:---|:---|:---|:---|
| VPWS | Point-to-point | LDP (RFC 4447) or BGP (RFC 6624) | Dedicated pseudowire between two sites |
| VPLS | Multipoint | LDP (RFC 4762) or BGP (RFC 4761) | Multi-site LAN extension |
| H-VPLS | Multipoint (hierarchical) | LDP + spoke pseudowires | Scaled VPLS with hub-spoke |
| EVPN | Multipoint | BGP (RFC 7432) | Next-gen, replaces VPLS |

### Pseudowire Fundamentals

- **Pseudowire (PW):** Emulated Layer 2 circuit across a Packet Switched Network (PSN)
- **PW encapsulation:** Customer L2 frame wrapped in PW header + tunnel header
- **VC label:** Inner MPLS label identifying the pseudowire instance
- **Transport label:** Outer MPLS label providing LSP transport between PEs
- **VC ID:** Locally significant identifier; must match on both ends of VPWS PW
- **Control word (CW):** Optional 4-byte header for sequencing, padding, and protocol identification

### Label Stack for L2VPN

```
+-------------------+-------------------+-------------------+
| Transport Label   | VC Label (PW)     | Customer L2 Frame |
| (LDP/RSVP/BGP-LU)| (service label)   | (Ethernet + data) |
+-------------------+-------------------+-------------------+
        Outer               Inner
```

## VPWS (Virtual Private Wire Service)

### Cisco IOS — xconnect

```
! Point-to-point pseudowire between two PEs
! PE1 config
interface GigabitEthernet0/1
 ! No IP address — pure L2 attachment
 xconnect 10.0.0.2 100 encapsulation mpls
 ! 10.0.0.2 = remote PE loopback
 ! 100 = VC ID (must match remote side)

! PE2 config
interface GigabitEthernet0/1
 xconnect 10.0.0.1 100 encapsulation mpls
```

### Cisco IOS — Pseudowire Class

```
! Define pseudowire parameters
pseudowire-class PW-MPLS
 encapsulation mpls
 ! Use specific tunnel for transport
 preferred-path interface Tunnel0
 ! Enable control word
 control-word
 ! Sequencing
 sequencing both

! Apply pseudowire class
interface GigabitEthernet0/1
 xconnect 10.0.0.2 100 pw-class PW-MPLS
```

### Cisco IOS-XR — VPWS

```
l2vpn
 xconnect group VPWS-GROUP
  p2p VPWS-1
   interface GigabitEthernet0/0/0/1
   neighbor ipv4 10.0.0.2 pw-id 100
    mpls static label local 5001 remote 5002
    ! Or use LDP signaling (default)
   !
  !
 !
```

### Cisco IOS-XR — EVPN-VPWS

```
l2vpn
 xconnect group EVPN-VPWS
  p2p EVPN-VPWS-1
   interface GigabitEthernet0/0/0/1
   neighbor evpn evi 100 target 1 source 2
   !
  !
 !
!
evpn
 evi 100
  bgp
   route-target 65000:100
  !
 !
```

### Juniper JunOS — VPWS (l2circuit)

```
interfaces {
    ge-0/0/1 {
        encapsulation ethernet-ccc;
        unit 0 {
            family ccc;
        }
    }
}

protocols {
    l2circuit {
        neighbor 10.0.0.2 {
            interface ge-0/0/1.0 {
                virtual-circuit-id 100;
                encapsulation-type ethernet;
                # Control word
                control-word;
                # Pseudowire redundancy
                backup-neighbor 10.0.0.3 {
                    virtual-circuit-id 100;
                    standby;
                }
            }
        }
    }
}
```

### Pseudowire Redundancy

```
! Cisco IOS — Redundant pseudowire
interface GigabitEthernet0/1
 xconnect 10.0.0.2 100 encapsulation mpls
  backup peer 10.0.0.3 100
  backup delay 0 0
  ! 0 0 = failover/fallback delay in seconds
```

```
! IOS-XR — PW redundancy with preferred path
l2vpn
 xconnect group REDUNDANT
  p2p PW-REDUNDANT
   interface GigabitEthernet0/0/0/1
   neighbor ipv4 10.0.0.2 pw-id 100
    pw-class MPLS-PW
    backup neighbor 10.0.0.3 pw-id 100
     pw-class MPLS-PW
    !
   !
  !
 !
```

## VPLS (Virtual Private LAN Service)

### Cisco IOS — LDP-signaled VPLS (RFC 4762)

```
! Define the VPLS VFI (Virtual Forwarding Instance)
l2vpn vfi context VPLS-CUST-A
 vpn id 100
 ! Full-mesh pseudowires to all PEs in the VPLS
 member 10.0.0.2 encapsulation mpls
 member 10.0.0.3 encapsulation mpls
 member 10.0.0.4 encapsulation mpls

! Bind VFI to a bridge domain
bridge-domain 100
 member vfi VPLS-CUST-A
 member GigabitEthernet0/1 service-instance 100
```

### Cisco IOS — BGP-signaled VPLS (RFC 4761)

```
! BGP auto-discovery eliminates manual PW mesh config
l2vpn vfi context VPLS-BGP
 vpn id 200
 autodiscovery bgp signaling bgp
  ve-id 1
  ve-range 10
  route-target export 65000:200
  route-target import 65000:200

! BGP address-family for VPLS
router bgp 65000
 address-family l2vpn vpls
  neighbor 10.0.0.2 activate
  neighbor 10.0.0.2 send-community extended
```

### Cisco IOS-XR — VPLS

```
l2vpn
 bridge group VPLS-BG
  bridge-domain VPLS-BD
   ! Attachment circuit
   interface GigabitEthernet0/0/0/1.100
   !
   ! VFI for VPLS
   vfi VPLS-VFI
    neighbor 10.0.0.2 pw-id 100
    neighbor 10.0.0.3 pw-id 100
    neighbor 10.0.0.4 pw-id 100
    ! PW class for MTU, control word, etc.
    pw-class STANDARD-PW
   !
  !
 !
```

### Juniper JunOS — VPLS

```
routing-instances {
    VPLS-CUST-A {
        instance-type vpls;
        interface ge-0/0/1.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        protocols {
            vpls {
                # LDP-signaled VPLS
                site-range 10;
                site CE1 {
                    site-identifier 1;
                    interface ge-0/0/1.100;
                }
                # Or explicit neighbors for LDP VPLS
                neighbor 10.0.0.2;
                neighbor 10.0.0.3;
            }
        }
    }
}
```

### Split Horizon

- **Rule:** Frames received on a PW must not be forwarded to another PW in the same VPLS
- **Purpose:** Prevents loops in the full-mesh PW topology
- **Implementation:** PW interfaces are in the same split-horizon group
- **Exception:** Hub-and-spoke (H-VPLS) spoke PWs are NOT in split-horizon group on the hub PE

```
! IOS-XR — Split-horizon group assignment
l2vpn
 bridge group BG
  bridge-domain BD
   vfi VFI
    neighbor 10.0.0.2 pw-id 100
    neighbor 10.0.0.3 pw-id 100
    ! All VFI members are automatically in the same
    ! split-horizon group
   !
  !
 !
```

### MAC Learning and Flooding

- **MAC learning:** PE learns source MACs from received frames on ACs and PWs
- **Unknown unicast flooding:** Frames with unknown destination MAC flooded to all ACs and PWs (except ingress, per split-horizon)
- **BUM traffic:** Broadcast, Unknown unicast, Multicast — all flooded
- **MAC aging:** Learned entries expire after aging timer (default 300s typically)
- **MAC table limit:** Per-bridge-domain limit to prevent table overflow

```
! IOS-XR — MAC table controls
l2vpn
 bridge group BG
  bridge-domain BD
   mac
    limit
     maximum 10000
     action flood
     notification both
    !
    aging
     time 300
    !
   !
  !
 !
```

## H-VPLS (Hierarchical VPLS)

### Architecture

```
        +------+     Full Mesh PWs      +------+
        | PE1  |<======================>| PE2  |
        | (hub)|<======================>| (hub)|
        +------+                        +------+
         /    \                          /    \
   Spoke PWs  Spoke PWs          Spoke PWs  Spoke PWs
       /        \                    /        \
  +------+  +------+          +------+  +------+
  | MTU1 |  | MTU2 |          | MTU3 |  | MTU4 |
  +------+  +------+          +------+  +------+
```

- **PE (hub):** Participates in full-mesh core VPLS
- **MTU (Multi-Tenant Unit):** Connects to PE via spoke pseudowire
- **Spoke PW:** Point-to-point PW from MTU to PE; NOT in split-horizon group on the PE
- **Benefit:** MTUs do not need full VPLS mesh; reduces PW count from $O(N^2)$ to $O(N)$

### IOS-XR H-VPLS Config

```
l2vpn
 bridge group H-VPLS-BG
  bridge-domain H-VPLS-BD
   ! Core full-mesh VFI
   vfi CORE-VFI
    neighbor 10.0.0.2 pw-id 100
    neighbor 10.0.0.3 pw-id 100
   !
   ! Spoke pseudowires from MTUs (NOT in split-horizon)
   neighbor 10.0.1.1 pw-id 200
    ! Spoke PW — no split-horizon with VFI
   !
   neighbor 10.0.1.2 pw-id 201
   !
  !
 !
```

## Pseudowire Interworking

### Ethernet-VLAN Interworking

- One side sends tagged frames (802.1Q), other side sends untagged
- PE performs VLAN tag push/pop at the interworking point

```
! IOS — Ethernet-VLAN interworking
interface GigabitEthernet0/1
 ! Untagged (Ethernet mode) side
 xconnect 10.0.0.2 100 encapsulation mpls
  interworking ethernet

! Remote PE — tagged (VLAN mode) side
interface GigabitEthernet0/1.100
 encapsulation dot1Q 100
 xconnect 10.0.0.1 100 encapsulation mpls
  interworking vlan
```

### IP Interworking Mode

- Strips L2 header entirely; transports only the IP payload
- Used when endpoints have incompatible L2 encapsulations (e.g., Ethernet ↔ PPP, Ethernet ↔ Frame Relay)

```
! IOS — IP-mode interworking
interface GigabitEthernet0/1
 xconnect 10.0.0.2 100 encapsulation mpls
  interworking ip
```

## Control Word

### Format (4 bytes)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|0 0 0 0| Flags |FRG|  Length   |     Sequence Number           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

- **First nibble:** Always 0000 (distinguishes PW from IP payload behind label)
- **Flags:** Reserved (must be 0)
- **FRG:** Fragmentation bits (00=unfragmented, 01=first, 10=last, 11=middle)
- **Length:** Payload length (0 if >= 64 bytes)
- **Sequence Number:** For ordered delivery (0 if sequencing disabled)

### Why Control Word Matters

- Without CW, ECMP may hash on the first nibble after the label stack
- IPv4 first nibble = 0x4, IPv6 = 0x6; ECMP may misidentify PW payload as IP
- CW first nibble = 0x0, preventing false IP identification
- Required for interworking, recommended for all Ethernet PWs
- **Must match on both ends:** Both PEs must agree on CW usage or PW will not come up

```
! Enable control word (IOS-XR)
l2vpn
 pw-class CW-ENABLED
  encapsulation mpls
   control-word
  !
 !
```

## MPLS-TP (Transport Profile)

### Key Properties

- MPLS subset designed for transport (OTN/SONET replacement)
- No IP control plane required (can use static LSPs)
- Bidirectional LSPs (unlike standard MPLS unidirectional)
- Enhanced OAM: BFD for MPLS-TP, G-ACh (Generic Associated Channel)
- Linear protection: 1+1 and 1:1 with sub-50ms switchover
- No PHP (always label-based forwarding)
- No ECMP (deterministic forwarding path)

## Verification Commands

### Cisco IOS / IOS-XR

```bash
# Show pseudowire status (IOS)
show mpls l2transport vc
show mpls l2transport vc detail
show mpls l2transport vc <vc-id>

# Show pseudowire status (IOS-XR)
show l2vpn xconnect
show l2vpn xconnect detail
show l2vpn xconnect group <group-name>

# VPLS bridge domain info (IOS-XR)
show l2vpn bridge-domain
show l2vpn bridge-domain detail
show l2vpn bridge-domain bd-name <name> detail

# MAC address table
show l2vpn bridge-domain bd-name <name> mac address-table
show l2vpn forwarding bridge-domain <name> mac-address location <lc>

# Pseudowire neighbor discovery (BGP VPLS)
show bgp l2vpn vpls all

# PW class configuration
show l2vpn pw-class

# PW redundancy state
show l2vpn xconnect group <group> detail | include "state|backup"

# VPLS flooding and BUM stats
show l2vpn bridge-domain bd-name <name> detail | include "flood|storm"
```

### Juniper JunOS

```bash
# Show l2circuit (VPWS) status
show l2circuit connections

# Show VPLS connections
show vpls connections

# Show VPLS MAC table
show vpls mac-table instance <name>

# Show VPLS flooding info
show vpls flood instance <name>

# Show VPLS statistics
show vpls statistics instance <name>
```

## Tips

- Always enable control word on Ethernet pseudowires; without it, ECMP hashing may treat PW payload as IP and cause out-of-order delivery.
- VC IDs must match on both PEs for a VPWS pseudowire; a mismatch is the most common cause of PW down state.
- VPLS split-horizon prevents loops in the full-mesh PW core, but spoke PWs in H-VPLS must NOT be in the split-horizon group or traffic from MTUs will be dropped.
- MAC address table limits are critical in VPLS; without them, a MAC flood attack can exhaust PE memory.
- MTU must match on both ends of a pseudowire; a 1-byte mismatch will prevent the PW from coming up with LDP signaling (LDP advertises the MTU in the FEC element).
- BGP-signaled VPLS (RFC 4761) scales better than LDP-signaled (RFC 4762) because BGP auto-discovery eliminates manual neighbor configuration.
- For greenfield deployments, use EVPN instead of VPLS; EVPN provides all VPLS capabilities plus multihoming, MAC mobility, and ARP suppression.
- Pseudowire redundancy requires consistent VC IDs on primary and backup; use different remote PE addresses but the same VC ID.
- IP interworking mode strips the L2 header entirely; use only when endpoints have incompatible L2 types and L2 transparency is not required.
- MPLS-TP is designed for transport networks requiring deterministic paths and sub-50ms protection; it is not a replacement for standard MPLS in IP/MPLS networks.
- When troubleshooting PW status, check both local and remote status codes; common issues are MTU mismatch, VC type mismatch, and control word mismatch.

## See Also

- mpls, evpn-advanced, unified-mpls, vxlan, bgp, ethernet

## References

- [RFC 4447 — Pseudowire Setup and Maintenance Using LDP](https://www.rfc-editor.org/rfc/rfc4447)
- [RFC 4762 — Virtual Private LAN Service (VPLS) Using LDP Signaling](https://www.rfc-editor.org/rfc/rfc4762)
- [RFC 4761 — Virtual Private LAN Service (VPLS) Using BGP](https://www.rfc-editor.org/rfc/rfc4761)
- [RFC 7432 — BGP MPLS-Based Ethernet VPN (EVPN)](https://www.rfc-editor.org/rfc/rfc7432)
- [RFC 6718 — Pseudowire Redundancy](https://www.rfc-editor.org/rfc/rfc6718)
- [RFC 4448 — Encapsulation Methods for Transport of Ethernet over MPLS](https://www.rfc-editor.org/rfc/rfc4448)
- [RFC 6310 — Pseudowire OAM](https://www.rfc-editor.org/rfc/rfc6310)
- [RFC 5921 — MPLS Transport Profile (MPLS-TP) Framework](https://www.rfc-editor.org/rfc/rfc5921)
- [Cisco L2VPN Configuration Guide (IOS-XR)](https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/l2vpn/configuration/guide/l2vpn-cg.html)
- [Juniper VPLS Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/vpls/topics/topic-map/vpls-overview.html)
