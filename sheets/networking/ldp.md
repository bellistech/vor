# LDP (Label Distribution Protocol)

Signaling protocol (RFC 5036) used in MPLS networks to distribute label bindings between Label Switching Routers (LSRs), mapping FEC (Forwarding Equivalence Class) entries to MPLS labels so that IP prefixes can be forwarded along Label Switched Paths (LSPs). LDP establishes sessions over TCP port 646, discovers neighbors via UDP hello messages, and builds the Label Information Base (LIB) that drives MPLS forwarding.

---

## LDP Message Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Version (1)                  |       PDU Length              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       LDP Identifier                          |
|                       (6 bytes: LSR-ID + Label Space)         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|U| Message Type (15 bits)      |       Message Length          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Message ID (32 bits)                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Mandatory Parameters                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Optional Parameters                     |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

U bit:  0 = known message type, 1 = unknown (forward if U=1)

Message Types:
  0x0100  Notification
  0x0200  Hello
  0x0201  Initialization
  0x0202  KeepAlive
  0x0300  Address
  0x0301  Address Withdraw
  0x0400  Label Mapping
  0x0401  Label Request
  0x0402  Label Withdraw
  0x0403  Label Release
  0x0404  Label Abort Request

# LDP uses TCP port 646 for sessions, UDP port 646 for discovery
# LDP Identifier = LSR-ID (4 bytes, typically loopback) + Label Space (2 bytes)
```

## LDP Session Establishment

```
Router A (10.0.0.1)                       Router B (10.0.0.2)
  |                                          |
  |  UDP Hello (224.0.0.2:646)               |
  |  Transport: 10.0.0.1                     |
  |  ────────────────────────────────────>   |
  |                                          |
  |  UDP Hello (224.0.0.2:646)               |
  |  Transport: 10.0.0.2                     |
  |  <────────────────────────────────────   |
  |                                          |
  |  [Hello Adjacency Formed]                |
  |  Higher IP (10.0.0.2) opens TCP to 646   |
  |                                          |
  |  TCP SYN ──────────────────────────────> |
  |  TCP SYN-ACK <────────────────────────── |
  |  TCP ACK ──────────────────────────────> |
  |                                          |
  |  Initialization (proposed params)        |
  |  ────────────────────────────────────>   |
  |                                          |
  |  Initialization (proposed params)        |
  |  <────────────────────────────────────   |
  |                                          |
  |  KeepAlive (session confirmed)           |
  |  ────────────────────────────────────>   |
  |                                          |
  |  KeepAlive (session confirmed)           |
  |  <────────────────────────────────────   |
  |                                          |
  |  [LDP Session UP — exchange bindings]    |
  |                                          |
  |  Label Mapping (FEC → Label)             |
  |  ←──────────────────────────────────→    |

# TCP session initiator: router with the HIGHER transport address
# Hello hold time: default 15s (link), 45s (targeted)
# Keepalive interval: default 60s, hold timer: 180s
# Session negotiates: label advertisement mode, keepalive timer,
#   max PDU length, label space ID
```

## Label Advertisement Modes

```bash
# LDP supports two label advertisement modes:

# 1. Downstream Unsolicited (DU) — DEFAULT
#    - LSR advertises label bindings to peers without being asked
#    - Binding sent as soon as FEC is known (IGP route exists)
#    - Peers get bindings proactively
#    - Used by: Cisco IOS (default), Juniper (default), FRR

# 2. Downstream On Demand (DoD)
#    - LSR only sends a binding when explicitly requested
#    - Peer sends Label Request, LSR replies with Label Mapping
#    - Used in: ATM-based MPLS, frame-relay MPLS, some DoD deployments

# Label retention modes:

# 1. Liberal Label Retention (default)
#    - Keep ALL received bindings, even from non-best-next-hop peers
#    - Faster convergence (backup label already in LIB)
#    - Higher memory usage (stores all labels)

# 2. Conservative Label Retention
#    - Only keep bindings from the best next-hop peer for each FEC
#    - Lower memory usage
#    - Slower convergence (must request new label on topology change)

# Typical combination: Downstream Unsolicited + Liberal Retention
# This gives fastest convergence at the cost of more LIB entries
```

## FEC-to-Label Binding

```bash
# FEC (Forwarding Equivalence Class) types in LDP:
#   Type 1: Wildcard FEC — applies to all FECs
#   Type 2: Prefix FEC — IP prefix (e.g., 10.0.0.0/24)
#   Type 3: Host Address FEC — /32 host route

# Label ranges:
#   0       — Explicit NULL (IPv4)
#   1       — Router Alert
#   2       — Explicit NULL (IPv6)
#   3       — Implicit NULL (penultimate hop popping / PHP)
#   4-15    — Reserved
#   16+     — Dynamic labels allocated by LDP

# Penultimate Hop Popping (PHP):
# The second-to-last LSR pops the label and sends unlabeled IP
# This avoids double lookup (label + IP) on the egress LSR
# LDP signals PHP by advertising Implicit NULL (label 3) for connected prefixes

# Label binding example:
# Router 10.0.0.1 advertises:
#   FEC 10.0.0.1/32 → Label 3 (Implicit NULL / PHP)
#   FEC 192.168.1.0/24 → Label 24001
# Meaning: "To reach 10.0.0.1/32 through me, pop the label (PHP)"
#          "To reach 192.168.1.0/24 through me, use label 24001"
```

## Targeted LDP Sessions

```bash
# Targeted LDP sessions are used for:
#   - Remote LDP neighbors (not directly connected)
#   - LDP over RSVP-TE tunnels
#   - Pseudowires (L2VPN)
#   - LDP session protection (backup targeted session)

# Normal LDP: discovers neighbors via link-local multicast UDP Hellos
# Targeted LDP: sends unicast UDP Hellos to a specific remote IP

# FRR (vtysh) — configure targeted LDP neighbor
configure terminal
mpls ldp
 neighbor 10.0.0.5 targeted
 discovery targeted-hello accept

# Cisco IOS — targeted session
mpls ldp neighbor 10.0.0.5 targeted
mpls ldp discovery targeted-hello accept

# Juniper JunOS — targeted session
set protocols ldp targeted-hello 10.0.0.5

# Targeted hellos have longer hold time (default 45s vs 15s for link)
# They can traverse multiple hops (routed, not link-local)
# Used to maintain LDP sessions over RSVP-TE tunnels (LDPoRSVP)
```

## LDP-IGP Synchronization

```bash
# Problem: when LDP session goes down but IGP adjacency stays up,
# traffic is blackholed (IGP forwards to MPLS-enabled next-hop
# but no label exists). LDP-IGP sync prevents this.

# Mechanism:
# 1. When LDP session goes down, IGP advertises max metric for the link
# 2. Traffic reroutes around the link via IGP
# 3. When LDP session re-establishes, IGP restores normal metric
# 4. Traffic returns to the link

# FRR — LDP-IGP sync with OSPF
configure terminal
mpls ldp
 address-family ipv4
  interface eth0
   igp sync holddown 60

router ospf
 mpls ldp-sync

# Cisco IOS — LDP-IGP sync with OSPF
router ospf 1
 mpls ldp sync

# Juniper JunOS
set protocols ospf area 0 interface ge-0/0/0 ldp-synchronization

# IS-IS also supports LDP-IGP sync
# The holddown timer prevents flapping (default varies by vendor)

# Verify sync state
# FRR: show mpls ldp igp-sync
# Cisco: show mpls ldp igp sync
# Juniper: show ldp synchronization
```

## FRR (Free Range Routing) LDP Configuration

```bash
# FRR provides LDP via the ldpd daemon

# Enable ldpd in /etc/frr/daemons
# ldpd=yes

# Basic LDP configuration (vtysh)
configure terminal
mpls ldp
 router-id 10.0.0.1
 address-family ipv4
  discovery transport-address 10.0.0.1
  interface eth0
  interface eth1
  neighbor 10.0.0.2 password s3cret

# Show LDP neighbors
show mpls ldp neighbor
show mpls ldp neighbor detail

# Show LDP bindings (FEC → label mappings)
show mpls ldp binding
show mpls ldp binding 10.0.0.2/32

# Show MPLS forwarding table (LFIB)
show mpls table

# Show LDP discovery (hello adjacencies)
show mpls ldp discovery

# Show LDP interface status
show mpls ldp interface

# Show LDP address bindings (advertised addresses)
show mpls ldp address

# Debug LDP
debug mpls ldp messages recv
debug mpls ldp messages sent
debug mpls ldp errors
```

## Cisco IOS LDP Configuration

```bash
# Global LDP configuration
mpls ldp router-id Loopback0 force
mpls label protocol ldp
mpls label range 100 199999

# Enable LDP on interfaces
interface GigabitEthernet0/0
 mpls ip

# LDP authentication (MD5)
mpls ldp neighbor 10.0.0.2 password s3cret

# LDP session protection (targeted backup session)
mpls ldp session protection duration 120

# Show commands
show mpls ldp neighbor
show mpls ldp neighbor detail
show mpls ldp bindings
show mpls ldp bindings 10.0.0.1 32
show mpls forwarding-table
show mpls ldp discovery
show mpls ldp parameters
show mpls ldp igp sync

# Filter label advertisements
# Only advertise labels for loopback prefixes
mpls ldp label
 allocate global prefix-list LOOPBACKS
ip prefix-list LOOPBACKS permit 10.0.0.0/24 ge 32 le 32
```

## Juniper JunOS LDP Configuration

```bash
# LDP on Juniper
set protocols ldp interface ge-0/0/0.0
set protocols ldp interface ge-0/0/1.0
set protocols ldp interface lo0.0

# LDP transport address (loopback)
set protocols ldp transport-address router-id

# LDP authentication
set protocols ldp session 10.0.0.2 authentication-key s3cret

# LDP session protection
set protocols ldp session-protection timeout 120

# Deaggregate labels (per-prefix label instead of per-next-hop)
set protocols ldp deaggregate

# LDP tracking (follow IGP metric for fastest convergence)
set protocols ldp track-igp-metric

# Show commands
# show ldp neighbor
# show ldp session
# show ldp database
# show ldp route
# show ldp interface
# show route table mpls.0
# show ldp synchronization
```

## LDP vs RSVP-TE vs Segment Routing

```
Feature              LDP                RSVP-TE             Segment Routing
───────────────────  ─────────────────  ─────────────────   ─────────────────
Label allocation     Per-prefix         Per-tunnel (LSP)    Per-prefix/adj (SID)
Traffic engineering  No                 Yes (ERO)           Yes (SID list)
Bandwidth reserv.    No                 Yes                 No (relies on IGP)
Session state        Per-neighbor       Per-LSP (N^2)       None (stateless)
Failure recovery     IGP convergence    Fast-reroute (FRR)  TI-LFA
Control plane        LDP (TCP 646)      RSVP (IP proto 46)  IGP extensions
Scalability          Good               Poor at scale       Excellent
Setup complexity     Low                High                Medium
PHP support          Implicit NULL      Explicit config      Penultimate SID

# Modern trend: Segment Routing (SR-MPLS, SRv6) replacing both LDP and RSVP-TE
# Migration path: LDP → SR-MPLS with LDP interop → pure SR
# RFC 8661 defines SR-LDP interworking
```

## LDP Graceful Restart

```bash
# LDP Graceful Restart (RFC 3478) preserves MPLS forwarding
# during LDP session restart (control plane failure)

# Two roles:
# 1. Restarting router — restarts LDP, keeps MPLS forwarding table
# 2. Helper router — maintains label bindings during restart

# Sequence:
# 1. Control plane fails, LDP session drops
# 2. Helper detects session loss but keeps forwarding entries
# 3. Restarting router re-establishes LDP session
# 4. FT Session TLV in Initialization indicates GR capability
# 5. Label bindings are re-exchanged
# 6. Stale bindings cleaned up after recovery timer

# FRR — enable graceful restart
configure terminal
mpls ldp
 graceful-restart
 graceful-restart reconnect-timeout 120
 graceful-restart recovery-time 120

# Cisco IOS
mpls ldp graceful-restart
mpls ldp graceful-restart timers forwarding-holding 120
mpls ldp graceful-restart timers neighbor-liveness 120

# Juniper JunOS
set protocols ldp graceful-restart
set protocols ldp graceful-restart reconnect-time 120
set protocols ldp graceful-restart recovery-time 120

# Verify GR capability in neighbor output
# show mpls ldp neighbor detail (look for "Graceful Restart" capability)
```

## Troubleshooting LDP

```bash
# Common LDP issues and diagnostics:

# 1. No LDP neighbor — check transport address reachability
ping 10.0.0.2 source 10.0.0.1
# Ensure loopback is reachable via IGP before LDP can establish

# 2. Check UDP hello exchange
tcpdump -i eth0 -nn 'udp port 646'

# 3. Check TCP session establishment
tcpdump -i any -nn 'tcp port 646'

# 4. LDP session flapping — check keepalive timers
# show mpls ldp neighbor detail | grep -i keep

# 5. Missing label bindings — check FEC filter
# show mpls ldp binding
# Verify IGP has the route before LDP allocates a label

# 6. MPLS forwarding not working — check LFIB
# show mpls table
# show mpls forwarding-table

# 7. LDP-IGP sync issues
# show mpls ldp igp-sync

# 8. Authentication failure
# Check MD5 password matches on both sides
# tcpdump will show TCP RST if MD5 auth fails

# 9. Label space mismatch
# show mpls ldp parameters
# Ensure both sides negotiate compatible parameters

# 10. MTU issues with MPLS
# MPLS adds 4 bytes per label to the packet
# Ensure interface MTU accounts for label stack
# ip link set eth0 mtu 9000  # or at least 1504 for single label
```

---

## Tips

- LDP follows the IGP topology exactly. If the IGP has a route to a prefix, LDP will have a label binding for it. If the IGP route disappears, the LDP binding is withdrawn. Debug IGP first, then LDP.
- The higher IP address initiates the TCP session during LDP establishment. If you see TCP connection attempts from only one side, the other router may have a transport address mismatch or ACL blocking port 646.
- Always configure LDP-IGP synchronization. Without it, a window exists during convergence where IGP forwards traffic to a neighbor that has no LDP label, causing silent packet drops.
- Penultimate Hop Popping (PHP) is the default behavior. The egress LSR advertises Implicit NULL (label 3) so the penultimate router pops the label, avoiding a double lookup. Explicit NULL (label 0) is used when the egress needs to see the MPLS header (e.g., for QoS EXP bits).
- Use targeted LDP sessions for session protection. If the direct link fails but an alternate path exists, the targeted session (via loopback) keeps label bindings alive, reducing convergence time to IGP reconvergence only.
- Filter label advertisements to reduce LIB size. In large networks, advertise labels only for loopback prefixes (infrastructure /32s) rather than every IGP route. This dramatically reduces memory consumption.
- LDP MD5 authentication protects the TCP session from spoofed packets. Always enable it in production. A mismatch causes silent TCP RSTs that are easy to miss without packet captures.
- When migrating from LDP to Segment Routing, run both protocols simultaneously during transition. SR-MPLS and LDP can coexist, with SR taking priority when both have bindings for the same FEC.
- LDP graceful restart is essential for high-availability MPLS networks. Without it, a control plane restart causes all label bindings to be withdrawn, disrupting MPLS forwarding even if the data plane is unaffected.
- MPLS adds 4 bytes per label to the frame. On links with 1500-byte MTU, this can cause fragmentation or drops for maximum-size IP packets. Set MPLS MTU to at least 1504 or use jumbo frames on the MPLS core.

---

## See Also

- mpls, ospf, bgp, is-is, rsvp

## References

- [RFC 5036 — LDP Specification](https://www.rfc-editor.org/rfc/rfc5036)
- [RFC 5283 — LDP Extension for Inter-Area LSPs](https://www.rfc-editor.org/rfc/rfc5283)
- [RFC 3478 — Graceful Restart Mechanism for LDP](https://www.rfc-editor.org/rfc/rfc3478)
- [RFC 5443 — LDP IGP Synchronization](https://www.rfc-editor.org/rfc/rfc5443)
- [RFC 8661 — Segment Routing MPLS Interworking with LDP](https://www.rfc-editor.org/rfc/rfc8661)
- [FRR — LDP User Guide](https://docs.frrouting.org/en/latest/ldpd.html)
- [Juniper — LDP Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/mpls/topics/concept/ldp-overview.html)
