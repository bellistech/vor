# DMVPN --- Dynamic Multipoint VPN Architecture and Phases

> *DMVPN solves the fundamental scalability problem of site-to-site VPNs: connecting N sites with traditional IPsec requires N(N-1)/2 tunnels and O(N^2) configuration lines. DMVPN collapses this to N spoke configurations and one hub configuration by combining multipoint GRE tunneling with the Next Hop Resolution Protocol. Understanding how mGRE, NHRP, IPsec, and routing interact across DMVPN's three phases is essential for designing networks that scale from ten sites to thousands.*

---

## 1. The Problem DMVPN Solves

### Traditional IPsec VPN Scaling

In a classic point-to-point IPsec deployment, every pair of sites needs a dedicated tunnel. For a network with N sites requiring full-mesh connectivity:

```
Tunnels required:  T = N(N-1) / 2

Sites   Tunnels   Config Lines (approx)
  5       10         200
 10       45         900
 50     1225       24,500
100     4950       99,000
500   124,750    2,495,000
```

Each tunnel requires a crypto map entry, an ACL, a peer definition, and a tunnel interface (or virtual tunnel interface). This is unmanageable beyond a few dozen sites.

Even a hub-and-spoke model with static GRE tunnels requires N-1 point-to-point tunnel interfaces on the hub, each with a unique IP subnet and static configuration. Adding a new spoke means configuring both the spoke and the hub.

### What DMVPN Changes

DMVPN introduces a single multipoint GRE interface on the hub that dynamically accepts connections from any spoke. Spokes register their public (NBMA) addresses with the hub via NHRP, eliminating static tunnel definitions. The hub configuration is fixed regardless of the number of spokes.

```
Configuration complexity:

Traditional P2P:   O(N^2) for full mesh, O(N) on hub for hub-and-spoke
DMVPN:             O(1) on hub, O(1) per spoke (independent of N)
```

Adding a new spoke requires configuring only the spoke itself. The hub learns about it automatically through NHRP registration.

---

## 2. Architectural Components

### 2.1 Multipoint GRE (mGRE)

Standard GRE (RFC 2784) creates a point-to-point tunnel between two endpoints. Each tunnel interface has exactly one tunnel destination. Multipoint GRE removes this limitation: a single tunnel interface can send and receive encapsulated packets to and from any number of remote endpoints.

**How mGRE works internally:**

The router maintains a mapping table that associates tunnel overlay addresses with NBMA (underlay) addresses. When a packet arrives at the tunnel interface destined for overlay address 10.255.0.3, the router looks up the corresponding NBMA address (say, 203.0.113.20) and builds the outer GRE/IP header with that destination. This lookup is performed by NHRP.

**GRE header format in DMVPN context:**

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|C| |K|S|    Reserved0    | Ver |       Protocol Type           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     Key (if K bit set)                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                  Sequence Number (if S bit set)               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+

Key field: Used in dual-cloud DMVPN to differentiate which tunnel
           interface should process inbound GRE packets when multiple
           tunnels share the same source address.
Protocol Type: 0x0800 for IPv4 payload, 0x86DD for IPv6.
```

The Key bit (K) is critical in dual-cloud designs. When two DMVPN tunnels (Tunnel0 and Tunnel1) share the same physical source interface, the router uses the GRE key value to demultiplex incoming packets to the correct tunnel. Without the key, all GRE packets would arrive at the first matching tunnel interface.

**mGRE vs point-to-point GRE:**

```
Attribute              Point-to-Point GRE       Multipoint GRE
────────────────────────────────────────────────────────────────────
tunnel destination     Static, one peer         Dynamic, many peers
IP addressing          /30 subnet per tunnel    /24+ shared subnet
Scalability            1 interface per peer     1 interface for all
Spoke-to-spoke         Requires hub relay       Direct tunnels possible
Config on hub          Grows with N spokes      Fixed regardless of N
```

### 2.2 NHRP (Next Hop Resolution Protocol)

NHRP (RFC 2332) is the control-plane protocol that makes mGRE work. It provides the mapping between overlay (tunnel) addresses and underlay (NBMA) addresses --- the same function that ARP performs on a LAN, but across a non-broadcast multi-access network.

**NHRP terminology:**

- **NBMA (Non-Broadcast Multi-Access):** The underlay transport network (the internet, an MPLS network). "Non-broadcast" means you cannot send broadcast/multicast natively --- each destination needs an explicit address.
- **NHS (Next Hop Server):** The NHRP server, always running on the hub. Spokes register with the NHS and query it for address resolution.
- **NHC (Next Hop Client):** The NHRP client, running on every spoke. Sends registration and resolution requests.
- **NBMA address:** The public/transport IP address of a tunnel endpoint.
- **Protocol address:** The tunnel overlay IP address.

**NHRP message types and their roles:**

1. **Registration Request/Reply:** Spoke informs the hub of its tunnel-to-NBMA mapping. The hub stores this in its NHRP cache and responds with a reply. Registrations are periodic (default hold time 600 seconds, re-registration at hold_time/3).

2. **Resolution Request/Reply:** A node asks "What is the NBMA address for tunnel IP X?" The NHS looks up its cache and either replies directly or forwards the request to the target spoke. In Phase 2, this is what triggers spoke-to-spoke tunnel creation.

3. **Purge Request/Reply:** Removes a stale mapping from caches. Sent when a tunnel endpoint changes its NBMA address (e.g., a spoke gets a new public IP from DHCP).

4. **Redirect:** Introduced for Phase 3. The hub sends this to a spoke to indicate that a shorter path exists. This is the trigger for NHRP shortcut installation.

5. **Error Indication:** Reports failures (e.g., destination unreachable, administratively prohibited).

**NHRP cache structure:**

```
Hub NHRP Cache:
Target         NBMA Address      Type       Flags   Expire
10.255.0.2     203.0.113.10      dynamic    reg     540s
10.255.0.3     203.0.113.20      dynamic    reg     580s
10.255.0.4     203.0.113.30      dynamic    reg     320s
10.255.0.5     198.18.1.100      dynamic    reg     600s

Spoke NHRP Cache:
Target         NBMA Address      Type       Flags   Expire
10.255.0.1     198.51.100.1      static     -       -          (hub, configured)
10.255.0.3     203.0.113.20      dynamic    res     280s       (resolved spoke)
```

Static entries never expire. Dynamic entries from registration or resolution expire after the NHRP hold time. The spoke must re-register before the hub's cache entry expires, or the hub will drop it.

**NHRP authentication:**

NHRP authentication is not cryptographic --- it is a plaintext string match (like OSPF simple password). It prevents accidental misconfiguration but does not protect against active attacks. IPsec provides the actual security layer.

```
ip nhrp authentication SECRETKEY    ! Max 8 characters
```

All nodes in the same DMVPN network must share the same authentication string.

### 2.3 IPsec Protection

DMVPN relies on IPsec for confidentiality and integrity of the GRE-encapsulated traffic. The integration is done through an IPsec profile rather than a crypto map, which is critical because crypto maps require static peer definitions --- exactly what DMVPN eliminates.

**Why IPsec profiles (not crypto maps):**

A crypto map binds an ACL (defining interesting traffic) to a specific peer IP and a transform set. This is static. An IPsec profile is a template: it specifies the transform set and key management parameters but has no ACL and no peer address. When a GRE packet needs to be sent to a new NBMA address, the router dynamically creates an IPsec SA using the profile as a template.

**IPsec transport mode rationale:**

DMVPN always uses IPsec transport mode, not tunnel mode. The reason is that mGRE already adds an outer IP header (the NBMA source and destination). Using IPsec tunnel mode would add yet another IP header, wasting 20 bytes per packet and complicating PMTUD. With transport mode, ESP encrypts the GRE header and payload while the outer IP header remains untouched for routing.

**Packet encapsulation order:**

```
Original packet:
[IP Header][TCP/UDP][Payload]

After GRE encapsulation:
[Outer IP (NBMA src/dst)][GRE Header][Original IP][TCP/UDP][Payload]

After IPsec transport mode:
[Outer IP (NBMA src/dst)][ESP Header][GRE Header][Original IP][TCP/UDP][Payload][ESP Trailer][ESP Auth]

Total overhead:
  GRE:   24 bytes (with key)
  ESP:   ~50-73 bytes (depends on algorithm, padding)
  Total: ~74-97 bytes
  Effective MTU from 1500: ~1400-1426 bytes
```

**IKE negotiation in DMVPN:**

When a spoke registers with the hub and tunnel protection is configured, the spoke initiates an IKE negotiation with the hub's NBMA address. The resulting IPsec SA protects all traffic between the spoke and hub.

When a spoke-to-spoke tunnel forms (Phase 2 or 3), a new IKE negotiation occurs between the two spokes' NBMA addresses. This happens automatically after NHRP resolution provides the peer's NBMA address.

**IKEv1 vs IKEv2 in DMVPN:**

```
Feature                    IKEv1                IKEv2
────────────────────────────────────────────────────────────────
Messages to establish SA   6-9 (main/aggressive) 4 (initial exchange)
NAT-T detection            Separate (RFC 3947)   Built-in
EAP authentication         Not supported          Supported
Fragmentation handling     Vendor-specific         RFC 7383
Multiple SAs per IKE SA    No                     Yes (CREATE_CHILD_SA)
Dead Peer Detection        Separate (RFC 3706)    Built-in liveness checks
Suite B support            Limited                Full
```

IKEv2 is strongly recommended for new DMVPN deployments. It reduces handshake latency, handles NAT traversal natively, and supports more robust authentication methods.

### 2.4 Routing Protocol Role

The routing protocol distributes overlay network reachability across the DMVPN cloud. Every spoke learns which subnets are behind which other spoke (or behind the hub). The critical interaction is between the routing protocol's next-hop behavior and NHRP's resolution mechanism.

**The next-hop problem:**

Consider Spoke A wanting to reach network 172.16.1.0/24 behind Spoke B. The routing protocol at the hub learns this prefix from Spoke B with a next-hop of 10.255.0.3 (Spoke B's tunnel IP). When the hub advertises this prefix to Spoke A, what happens to the next-hop field?

- **If the hub preserves the next-hop (10.255.0.3):** Spoke A's routing table says "172.16.1.0/24 via 10.255.0.3". Spoke A does not have an NHRP mapping for 10.255.0.3 in Phase 2, so it sends an NHRP Resolution Request. After resolution, traffic flows directly spoke-to-spoke. This is the Phase 2 trigger mechanism.

- **If the hub rewrites the next-hop to itself (10.255.0.1):** Spoke A's routing table says "172.16.1.0/24 via 10.255.0.1". Spoke A already has an NHRP mapping for the hub. No resolution is triggered. All traffic goes through the hub. Phase 2 is broken.

This is why Phase 2 imposes strict constraints on routing protocol behavior, and why Phase 3 was developed to eliminate these constraints.

---

## 3. DMVPN Phases in Depth

### 3.1 Phase 1 --- Hub-and-Spoke

Phase 1 is the simplest DMVPN deployment. All traffic between spokes transits through the hub. Spokes may use point-to-point GRE or mGRE, but there is no spoke-to-spoke direct communication.

**Data-plane flow:**

```
Spoke A (10.255.0.2) --> Hub (10.255.0.1) --> Spoke B (10.255.0.3)

Packet from 192.168.1.10 to 172.16.1.10:

Step 1: Spoke A encapsulates in GRE to hub NBMA (198.51.100.1)
Step 2: Hub receives, decapsulates GRE
Step 3: Hub routes internally: 172.16.1.0/24 via 10.255.0.3
Step 4: Hub encapsulates in GRE to Spoke B NBMA (203.0.113.20)
Step 5: Spoke B receives, decapsulates, delivers to 172.16.1.10
```

**Control-plane:** NHRP is used only for spoke-to-hub registration. No resolution requests are needed because all traffic goes to/from the hub.

**When Phase 1 is appropriate:**
- Security policy requires all traffic to pass through a central firewall/IDS
- Compliance mandates (e.g., PCI-DSS) require traffic inspection at a known point
- Very small deployments where bandwidth through the hub is not a constraint
- Transitional step before moving to Phase 2 or 3

**Limitations:**
- Hub is a bandwidth bottleneck (all spoke-to-spoke traffic is doubled)
- Hub is a single point of failure (without redundancy)
- Latency increases for geographically distant spokes that must transit through the hub
- Hub CPU load from encryption/decryption of all inter-spoke traffic

### 3.2 Phase 2 --- Spoke-to-Spoke with NHRP Resolution

Phase 2 adds direct spoke-to-spoke tunnels. After an initial period of routing through the hub, NHRP resolution creates a direct path. However, Phase 2 imposes significant constraints on the routing configuration.

**The Phase 2 mechanism step by step:**

```
1. Spoke A wants to reach 172.16.1.0/24 (behind Spoke B)
2. Spoke A looks up routing table: 172.16.1.0/24 via 10.255.0.3 (Spoke B)
   --- This next-hop is critical. It MUST be Spoke B, not the hub.
3. Spoke A checks NHRP cache for 10.255.0.3 -> no entry
4. CEF punts the packet to process-switching
5. Spoke A sends NHRP Resolution Request for 10.255.0.3 to its NHS (hub)
6. Hub receives request, looks up 10.255.0.3 in its NHRP cache
7. Hub finds mapping: 10.255.0.3 -> 203.0.113.20
8. Hub forwards the Resolution Request to Spoke B
   (or may reply authoritatively from its own cache)
9. Spoke B receives request, generates Resolution Reply
10. Resolution Reply reaches Spoke A: 10.255.0.3 -> 203.0.113.20
11. Spoke A installs NHRP mapping in its cache
12. IKE negotiation begins between Spoke A (203.0.113.10) and Spoke B (203.0.113.20)
13. IPsec SAs established
14. Direct GRE+IPsec tunnel between spokes is operational
15. CEF installs adjacency, subsequent packets use the direct path
```

**Routing constraints in Phase 2:**

The entire mechanism depends on the routing table having the remote spoke as the next-hop. This means:

1. **EIGRP:** The hub must NOT use next-hop-self. By default, EIGRP changes the next-hop to the advertising router when the route is re-advertised on the same interface. This must be disabled:
   ```
   interface Tunnel0
    no ip next-hop-self eigrp <AS>
   ```

2. **EIGRP split-horizon:** Must be disabled on the hub tunnel. By default, routes learned on an interface are not re-advertised out the same interface. Since all spokes connect to the same Tunnel0, the hub would not advertise Spoke B's routes back to Spoke A:
   ```
   interface Tunnel0
    no ip split-horizon eigrp <AS>
   ```

3. **OSPF:** The network type must preserve next-hop. `point-to-multipoint` preserves the originator as the next-hop. `broadcast` uses the DR as the next-hop (the hub), which breaks Phase 2 unless additional tuning is applied.

4. **Summarization:** Cannot be performed at the hub. If the hub summarizes 172.16.0.0/16, the next-hop for this aggregate is the hub itself, and spoke-to-spoke resolution never triggers.

5. **Default routes:** Cannot be originated from the hub for the same reason. Spokes would use the default route via the hub, and NHRP resolution would never be triggered.

These constraints are the primary reason Phase 3 was developed.

### 3.3 Phase 3 --- NHRP Redirect and Shortcuts

Phase 3 fundamentally changes the spoke-to-spoke trigger mechanism. Instead of relying on the routing table's next-hop field, Phase 3 uses NHRP protocol messages to signal shortcut availability. This removes all routing constraints from Phase 2.

**The Phase 3 mechanism step by step:**

```
1. Spoke A wants to reach 172.16.1.0/24 (behind Spoke B)
2. Spoke A looks up routing table: 172.16.1.0/24 via 10.255.0.1 (hub)
   --- Next-hop IS the hub. This is fine in Phase 3.
3. Spoke A sends packet to hub via established spoke-hub tunnel
4. Hub receives packet, performs a routing lookup
5. Hub determines: outgoing interface is Tunnel0, destination spoke is Spoke B
6. Hub forwards the packet to Spoke B (traffic flows, no delay)
7. Hub simultaneously sends an NHRP Redirect message to Spoke A:
   "A shorter path exists for this destination"
8. Spoke A receives the Redirect
9. Spoke A sends an NHRP Resolution Request for the destination
   (sent via the hub to Spoke B)
10. Spoke B responds with an NHRP Resolution Reply
11. Spoke A installs an NHRP shortcut route in its CEF table:
    172.16.1.0/24 -> next-hop 10.255.0.3 via Tunnel0, NBMA 203.0.113.20
12. IKE/IPsec negotiation creates spoke-to-spoke SAs
13. Subsequent packets use the direct shortcut path
14. If no traffic flows for the duration of the NHRP hold time,
    the shortcut expires and traffic reverts to going through the hub
```

**Key difference from Phase 2:** The routing protocol is not involved in triggering spoke-to-spoke tunnels. The hub can use next-hop-self, summarize routes, originate defaults --- none of this matters because NHRP Redirect operates at the data-plane forwarding level, not the control-plane routing level.

**NHRP shortcut route installation:**

When a spoke receives a Resolution Reply, it does not modify the routing table directly. Instead, it installs a shortcut entry in the CEF (Cisco Express Forwarding) table. This shortcut overrides the routing table's next-hop for the specific prefix. The routing table still shows the hub as the next-hop, but CEF forwards directly to the spoke.

```
Spoke A routing table:
  172.16.0.0/16 via 10.255.0.1 [90/...] (EIGRP summary from hub)

Spoke A CEF table (after shortcut):
  172.16.1.0/24 -> next-hop 10.255.0.3 via Tunnel0 [NHRP shortcut]
  172.16.0.0/16 -> next-hop 10.255.0.1 via Tunnel0 [routing table]

CEF longest-match wins: traffic to 172.16.1.0/24 uses the shortcut.
Traffic to 172.16.2.0/24 still goes through the hub (until its own shortcut forms).
```

**NHRP Redirect behavior on the hub:**

The hub sends an NHRP Redirect only when:
- The incoming interface and outgoing interface are the same (Tunnel0 to Tunnel0)
- `ip nhrp redirect` is configured on the tunnel interface
- The packet is not generated by the hub itself

This mirrors ICMP Redirect behavior: "you sent me a packet, but I'm just forwarding it back out the same interface to another host. Talk to that host directly."

**NHRP shortcut behavior on the spoke:**

The spoke processes NHRP Redirects and installs shortcuts only when:
- `ip nhrp shortcut` is configured on the tunnel interface
- The spoke can successfully resolve the destination via NHRP
- An IPsec SA can be established with the remote spoke

**Shortcut expiration and renewal:**

Shortcuts are tied to the NHRP hold timer. If no data traffic triggers a refresh before the timer expires, the shortcut is removed and traffic reverts to the hub path. If traffic continues, the NHRP entry is refreshed automatically. This means idle spoke-to-spoke tunnels are automatically cleaned up, reducing state on both spokes and saving IPsec SA resources.

---

## 4. NHRP Protocol Deep Dive

### 4.1 Registration Mechanics

NHRP registration is the heartbeat of DMVPN. Without successful registration, a spoke cannot participate in the overlay network.

**Registration timing:**

```
Event                          Timer
─────────────────────────────────────────────────
Initial registration           At tunnel interface up
Re-registration interval       hold_time / 3 (default: 200s)
Hold time (cache lifetime)     600 seconds (default)
Registration timeout           Retransmit after 7s, 7s, 7s...
Max registration retries       Infinite (keeps trying)
```

The spoke sends a Registration Request to the NHS. If no reply arrives within 7 seconds, the spoke retransmits. There is no exponential backoff --- the spoke retransmits at a constant interval until it receives a reply or the tunnel interface goes down.

**Registration request contents:**

```
Source Protocol Address:     10.255.0.2  (spoke tunnel IP)
Source NBMA Address:         203.0.113.10  (spoke public IP)
Requested Hold Time:         600 seconds
Flags:                       unique (this mapping should be exclusive)
Authentication Extension:    SECRETKEY (if configured)
```

**Registration failure causes:**

1. Network-id mismatch between spoke and hub
2. Authentication string mismatch
3. Firewall blocking GRE (IP protocol 47) or NHRP (encapsulated within GRE)
4. Hub tunnel interface down or misconfigured
5. Spoke tunnel source interface down or unreachable
6. NAT device not passing GRE (requires NAT-T or GRE over IPsec first)

### 4.2 Resolution Mechanics

NHRP resolution maps an overlay address to an NBMA address. The resolution process differs between Phase 2 and Phase 3.

**Phase 2 resolution flow:**

The trigger is the CEF lookup failing to find an NHRP mapping for the routing table's next-hop. The spoke sends a Resolution Request to its NHS. The hub either replies from its cache (authoritative reply) or forwards the request to the target spoke (non-authoritative, spoke replies directly).

**Phase 3 resolution flow:**

The trigger is an NHRP Redirect from the hub. The spoke sends a Resolution Request for the specific destination prefix (not just the next-hop). The resolution may return a mapping for the exact spoke or for a more specific prefix if the spoke behind the destination has registered specific routes.

**Resolution reply flags:**

```
Flag                Meaning
──────────────────────────────────────────────────────
authoritative       Reply came from the NHS (hub) cache
router              Responder is a router (not an end host)
unique              Only one NBMA address maps to this protocol address
NAT                 Responder is behind NAT
```

### 4.3 Redirect and Shortcut Interaction

The NHRP Redirect/Shortcut mechanism in Phase 3 is a two-step process:

**Step 1 --- Redirect (hub-initiated):**

The hub detects that it is forwarding a packet from Tunnel0 back to Tunnel0 (spoke-to-spoke via hub). It sends an NHRP Redirect to the source spoke containing the destination protocol address. The redirect does not contain the NBMA address --- it merely signals that a shortcut is possible.

**Step 2 --- Resolution (spoke-initiated):**

Upon receiving the Redirect, the spoke sends a Resolution Request for the destination. This is identical to Phase 2 resolution except the trigger is different (Redirect vs routing next-hop). The reply contains the NBMA address, and the shortcut is installed.

**Rate limiting:**

The hub does not send a Redirect for every packet. It rate-limits Redirects per destination to avoid flooding the control plane during bulk transfers. The default rate is typically one Redirect per destination per 15 seconds.

### 4.4 NHRP and NAT

NHRP has limited NAT support. If a spoke is behind a NAT device, the NBMA address in its Registration Request is the private address. The hub sees the packet arriving from the NAT's public address. This mismatch causes resolution to fail because other spokes cannot reach the private NBMA address.

**Solutions:**

1. **NAT extension (no-nat):** Configure `ip nhrp registration no-unique` on the spoke. The hub uses the outer IP source address from the registration packet (the NAT public address) instead of the NBMA address inside the NHRP payload.

2. **IPsec NAT-T (UDP 4500):** When IPsec detects NAT, it encapsulates ESP in UDP port 4500. This works for hub-and-spoke (Phase 1) but complicates spoke-to-spoke because both spokes may be behind different NAT devices.

3. **Spoke behind NAT, hub on public IP:** This is the most common scenario and works well. The spoke initiates all connections (NHRP registration, IKE negotiation), so NAT state is maintained. Spoke-to-spoke tunnels may fail if both spokes are behind NAT (double NAT).

---

## 5. Routing Protocol Considerations

### 5.1 EIGRP over DMVPN

EIGRP is the most natural fit for DMVPN because it is a Cisco proprietary protocol commonly deployed alongside other Cisco technologies. However, several default behaviors conflict with DMVPN.

**Split-horizon:**

EIGRP split-horizon prevents routes learned on an interface from being advertised back out the same interface. Since all spokes connect to the hub's Tunnel0, a route learned from Spoke B would not be advertised to Spoke A. This must be disabled on the hub:

```
interface Tunnel0
 no ip split-horizon eigrp 100
```

**Next-hop-self:**

By default, when EIGRP re-advertises a route on the interface it was learned on, it changes the next-hop to itself. On the hub, this means routes from Spoke B are advertised to Spoke A with the hub as the next-hop. In Phase 2, this prevents NHRP resolution. In Phase 3, this is acceptable because NHRP Redirect handles it.

```
! Phase 2 requirement:
interface Tunnel0
 no ip next-hop-self eigrp 100

! Phase 3: default behavior (next-hop-self) is fine
```

**Stub routing:**

In large DMVPN networks, EIGRP stub routing on spokes reduces query scope and improves convergence:

```
! Spoke configuration
router eigrp 100
 eigrp stub connected summary
```

Stub spokes do not receive queries from the hub, reducing convergence time from seconds to milliseconds for spoke failures. The hub knows that the stub spoke only has connected and summary routes, so it does not need to ask it about alternative paths.

**EIGRP bandwidth and delay on tunnel:**

The tunnel interface defaults to bandwidth 9 Kbps and delay 500000 usec (satellite link). These values affect metric calculation and should be adjusted:

```
interface Tunnel0
 bandwidth 1000000          ! 1 Gbps (or actual WAN bandwidth)
 delay 1000                 ! 100 usec (adjust for latency)
```

### 5.2 OSPF over DMVPN

OSPF is more complex over DMVPN because its network type determines DR election, adjacency behavior, and next-hop semantics.

**Network type selection:**

```
Network Type           DR/BDR    Adjacency          Next-Hop      Best Phase
─────────────────────────────────────────────────────────────────────────────
point-to-point         No        Direct              Preserved     Phase 1 only
broadcast              Yes       Via DR/BDR          DR sets it    Phase 3
point-to-multipoint    No        Direct (hellos)     Preserved     Phase 2 or 3
non-broadcast (NBMA)   Yes       Manual neighbor     DR sets it    Not recommended
```

**broadcast network type for Phase 3:**

This is the most common choice. The hub is the DR (highest priority), and spokes set priority 0 to never become DR or BDR. The hub (as DR) may set the next-hop to itself, which is fine in Phase 3.

```
! Hub
interface Tunnel0
 ip ospf network broadcast
 ip ospf priority 100
 ip ospf hello-interval 10
 ip ospf dead-interval 40

! Spoke
interface Tunnel0
 ip ospf network broadcast
 ip ospf priority 0
 ip ospf hello-interval 10
 ip ospf dead-interval 40
```

**point-to-multipoint for Phase 2:**

This type preserves the originator's router-id as the next-hop, which is required for Phase 2 NHRP resolution to trigger. However, it creates a /32 host route for each neighbor and generates more LSAs than broadcast mode.

**OSPF area design:**

For large DMVPN networks, placing the DMVPN cloud in a non-backbone area and using the hub as an ABR (Area Border Router) allows route summarization at the area boundary:

```
Hub:    Area 0 (backbone) + Area 1 (DMVPN)
Spokes: Area 1 only

! Hub
router ospf 1
 area 1 range 10.0.0.0 255.0.0.0         ! Summarize spoke networks into backbone
```

### 5.3 BGP over DMVPN

BGP is the best choice for very large DMVPN deployments (hundreds to thousands of spokes) because it does not flood routing updates to all peers --- the hub (as a route reflector) controls what each spoke receives.

**Hub as route reflector:**

```
router bgp 65000
 neighbor DMVPN-SPOKES peer-group
 neighbor DMVPN-SPOKES remote-as 65000
 neighbor DMVPN-SPOKES route-reflector-client
 neighbor DMVPN-SPOKES update-source Tunnel0
 ! Dynamic neighbors (no per-spoke config)
 bgp listen range 10.255.0.0/24 peer-group DMVPN-SPOKES
```

**Dynamic BGP neighbors:**

BGP `listen range` allows the hub to accept BGP sessions from any spoke in the tunnel subnet without explicit neighbor statements. This is the BGP equivalent of NHRP's `map multicast dynamic` --- zero per-spoke configuration on the hub.

**BGP next-hop behavior:**

iBGP does not change the next-hop by default when reflecting routes. This means Phase 2 works naturally with BGP. For Phase 3, `next-hop-self` can be configured if desired.

**BGP advantages for DMVPN:**

```
Feature                      EIGRP/OSPF        BGP
────────────────────────────────────────────────────────────
Control-plane state at hub   All routes         All routes
Control-plane state at spoke All routes         Only needed routes
Route filtering              Limited            Full policy control
Convergence time             Seconds            Seconds (with BFD)
Scale (tested deployments)   ~500 spokes        ~5000 spokes
Summarization                Protocol-specific  aggregate-address
```

### 5.4 BFD (Bidirectional Forwarding Detection)

BFD provides sub-second failure detection over DMVPN tunnels. Without BFD, routing protocol hello timers determine convergence time (OSPF default: 40 seconds dead interval, EIGRP default: 15 seconds hold time).

```
! Enable BFD on tunnel
interface Tunnel0
 bfd interval 500 min_rx 500 multiplier 3   ! 1.5 second detection

! Attach to routing protocol
router eigrp 100
 bfd interface Tunnel0

! Or for OSPF
router ospf 1
 bfd all-interfaces

! Or for BGP
router bgp 65000
 neighbor 10.255.0.2 fall-over bfd
```

BFD over DMVPN runs inside the GRE tunnel, so it detects both underlay failures and overlay issues.

---

## 6. Advanced Topics

### 6.1 Front-Door VRF (FVRF)

FVRF solves the recursive routing problem that occurs when the default route points through the DMVPN tunnel. Without FVRF, the router tries to reach the hub's NBMA address (198.51.100.1) using the default route, which goes through the tunnel, which needs to reach 198.51.100.1 --- a loop.

**The recursion problem:**

```
Without FVRF:
  ip route 0.0.0.0 0.0.0.0 10.255.0.1           (via tunnel)
  Tunnel0 destination: 198.51.100.1               (needs routing to reach)
  Route to 198.51.100.1: via default -> 10.255.0.1 -> Tunnel0 -> LOOP
```

**Traditional workaround (fragile):**

```
ip route 198.51.100.1 255.255.255.255 203.0.113.1  (static to hub NBMA via ISP gateway)
ip route 0.0.0.0 0.0.0.0 10.255.0.1                (default via tunnel)
```

This works but is fragile: if the hub's NBMA address changes, or if there are multiple NHS addresses, or if the ISP gateway changes, routes must be manually updated. With dual-hub designs, this becomes a maintenance burden.

**FVRF solution:**

Place the physical interface in a VRF. The tunnel's transport routing (reaching the NHS NBMA address) happens in that VRF, completely separated from the overlay routing in the global table.

```
! Transport VRF
ip vrf TRANSPORT
 rd 100:1

! Physical interface in VRF
interface GigabitEthernet0/0
 ip vrf forwarding TRANSPORT
 ip address 203.0.113.10 255.255.255.0

! Default route in transport VRF (to ISP)
ip route vrf TRANSPORT 0.0.0.0 0.0.0.0 203.0.113.1

! Tunnel uses transport VRF for outer IP routing
interface Tunnel0
 ip address 10.255.0.2 255.255.255.0
 tunnel source GigabitEthernet0/0
 tunnel vrf TRANSPORT                     ! This is the FVRF

! Default route in global table (via tunnel)
ip route 0.0.0.0 0.0.0.0 10.255.0.1      ! No recursion: tunnel uses TRANSPORT VRF
```

Now the two routing domains are completely isolated:
- **TRANSPORT VRF:** Knows how to reach NBMA addresses via the ISP. Does not know about overlay routes.
- **Global table:** Knows about overlay routes via the tunnel. Does not know about NBMA addresses. No recursion possible.

### 6.2 Per-Tunnel QoS Architecture

Standard interface-level QoS on the hub's Tunnel0 applies a single policy to all spoke traffic combined. This is insufficient because different spokes may have different bandwidth allocations (a headquarters spoke with 100 Mbps vs a branch with 10 Mbps).

**Per-tunnel QoS mechanism:**

The hub creates a virtual access interface for each spoke when it registers via NHRP. An NHRP group tag on the spoke's registration determines which QoS policy template is applied to that virtual interface.

```
Architecture:

Hub Tunnel0
  |
  +--- Virtual-Access1 (Spoke A, NHRP group: BRANCH-10M) -> QoS: 10 Mbps shape
  +--- Virtual-Access2 (Spoke B, NHRP group: BRANCH-50M) -> QoS: 50 Mbps shape
  +--- Virtual-Access3 (Spoke C, NHRP group: HQ-100M)    -> QoS: 100 Mbps shape
```

**Configuration:**

```
! Hub: define QoS policies
policy-map SHAPE-10M
 class class-default
  shape average 10000000
   service-policy BRANCH-QOS

policy-map SHAPE-50M
 class class-default
  shape average 50000000
   service-policy BRANCH-QOS

policy-map BRANCH-QOS
 class VOICE
  priority percent 20
 class VIDEO
  bandwidth percent 30
 class CRITICAL
  bandwidth percent 25
 class class-default
  fair-queue

! Hub: map NHRP groups to policies
ip nhrp map group BRANCH-10M service-policy output SHAPE-10M
ip nhrp map group BRANCH-50M service-policy output SHAPE-50M

! Spoke: set NHRP group during registration
interface Tunnel0
 ip nhrp group BRANCH-10M
```

**Why per-tunnel QoS matters:**

Without it, a high-bandwidth spoke can starve low-bandwidth spokes. Per-tunnel QoS ensures fair allocation and allows the hub to shape traffic to match each spoke's WAN link capacity, preventing drops at the WAN edge that are invisible to TCP congestion control.

### 6.3 Dual-Hub Design Considerations

Redundant hubs are essential for production DMVPN. Two architectures exist:

**Single-cloud (active/active or active/standby):**

Both hubs share the same tunnel subnet. Spokes register with both hubs (two NHS entries). The routing protocol determines which hub is preferred.

```
Advantages:
  - Simpler spoke configuration (one tunnel interface)
  - Faster failover (routing protocol convergence only)
  - Spoke-to-spoke tunnels survive hub failure (already established)

Disadvantages:
  - Both hubs must coordinate NHRP state
  - Routing protocol may oscillate between hubs
  - Asymmetric routing possible (ingress via Hub-1, egress via Hub-2)
```

**Dual-cloud (two independent DMVPN networks):**

Each hub owns a separate tunnel subnet and NHRP network-id. Spokes have two tunnel interfaces, one per cloud. Routing metrics determine primary/secondary cloud.

```
Advantages:
  - Complete isolation between clouds (fault domain separation)
  - No NHRP coordination between hubs
  - Clear primary/backup semantics via routing metrics

Disadvantages:
  - Double the tunnel interfaces on spokes
  - Double the IPsec SAs
  - Spoke-to-spoke tunnels are per-cloud (not shared)
  - More complex spoke configuration
```

**Failover mechanics:**

In single-cloud, if Hub-1 fails:
1. Spokes detect via routing protocol (EIGRP hold time, OSPF dead interval, or BFD)
2. Routes via Hub-1 are removed from the routing table
3. Routes via Hub-2 become active
4. Existing spoke-to-spoke shortcuts remain active (they do not depend on the hub after formation)
5. New spoke-to-spoke shortcuts use Hub-2 for NHRP resolution

In dual-cloud, if the primary cloud fails:
1. Spokes detect via routing protocol on Tunnel0
2. Tunnel1 routes (backup cloud) become preferred
3. Spoke-to-spoke tunnels must re-establish through the backup hub
4. Failback occurs when Tunnel0 routes become available again

### 6.4 DMVPN and IPv6

DMVPN supports IPv6 overlay over IPv4 underlay (most common), IPv6 overlay over IPv6 underlay, and dual-stack configurations.

**IPv6 overlay over IPv4 transport:**

```
interface Tunnel0
 ipv6 address 2001:db8:255::2/64
 tunnel source GigabitEthernet0/0            ! IPv4 source
 tunnel mode gre multipoint                  ! GRE carries IPv6
 ipv6 nhrp network-id 1
 ipv6 nhrp nhs 2001:db8:255::1 nbma 198.51.100.1 multicast
 ipv6 nhrp shortcut
 tunnel protection ipsec profile DMVPN-PROFILE
```

**Considerations:**
- NHRP for IPv6 uses the same message types but with IPv6 protocol addresses
- IPsec SAs are still established between IPv4 NBMA addresses
- Routing protocols (OSPFv3, EIGRP for IPv6, BGP with IPv6 address family) run over the tunnel

---

## 7. Troubleshooting Methodology

### 7.1 Systematic Verification Order

DMVPN troubleshooting should follow the protocol stack from bottom to top:

```
Layer    Check                              Command
─────────────────────────────────────────────────────────────────────
1. IP    Can spoke reach hub NBMA?          ping <hub-NBMA> (from FVRF if used)
2. GRE   Is tunnel interface up/up?         show interface Tunnel0
3. NHRP  Is spoke registered with hub?      show ip nhrp nhs detail
4. NHRP  Does hub have spoke mapping?       show ip nhrp brief (on hub)
5. IPsec Is IKE SA established?             show crypto ikev2 sa / show crypto isakmp sa
6. IPsec Is IPsec SA active?                show crypto ipsec sa
7. Route Are routing neighbors up?          show ip eigrp neighbors / show ip ospf neighbor
8. Route Are routes being learned?          show ip route
9. CEF   Are shortcuts installed?           show ip cef <prefix>
10. Data End-to-end connectivity            ping <remote-LAN> source <local-LAN>
```

### 7.2 NHRP-Specific Troubleshooting

**show ip nhrp output interpretation:**

```
Router# show ip nhrp
10.255.0.2/32 via 10.255.0.2
   Tunnel0 created 00:15:23, expire 00:04:37
   Type: dynamic, Flags: registered used nhop
   NBMA address: 203.0.113.10

Fields:
  expire:     Time until cache entry expires (re-registration refreshes)
  Type:       static (configured) or dynamic (learned via NHRP)
  Flags:
    registered  - This entry was learned via NHRP Registration
    used        - Traffic has been forwarded using this mapping
    nhop        - This is a next-hop entry (not a shortcut)
    router      - Remote end is a router
    rib         - Installed in routing table
    implicit    - Learned from NHRP packet source (not from a Resolution Reply)
```

**Common NHRP states and meanings:**

```
NHS State         Meaning                     Action
──────────────────────────────────────────────────────────────────
RE (registered)   Spoke registered with NHS    Normal operation
E  (expected)     Registration sent, no reply  Check connectivity to hub
R  (replied)      Hub replied to resolution    Normal (transient)
No entries        NHRP not running             Check nhrp config, tunnel state
```

### 7.3 IPsec Troubleshooting

**IKE Phase 1 (IKEv2 IKE_SA_INIT + IKE_AUTH):**

```
show crypto ikev2 sa
  If no SA: IKE negotiation never started -> check NHRP, GRE reachability
  If SA state INIT: Stuck in negotiation -> check proposal mismatch
  If SA state AUTH: Authentication failing -> check PSK or certificate
  If SA state READY: IKE established -> proceed to IPsec SA check
```

**IPsec SA (CHILD_SA):**

```
show crypto ipsec sa
  Check:
    - Packets encrypted/decrypted (should be incrementing)
    - Packets dropped (transform mismatch, replay failures)
    - SA lifetime remaining
    - Tunnel source and destination match expected NBMA addresses
```

### 7.4 Performance Troubleshooting

**Hub CPU concerns:**

The hub performs crypto operations for every spoke. At 100 spokes with 10 Mbps each, the hub processes 1 Gbps of encrypted traffic. Hardware crypto accelerators (ISR-G2 SEC-K9, ASR IPsec) are essential.

**NHRP cache size:**

Large deployments may exhaust NHRP cache. Monitor with:

```
show ip nhrp traffic
show ip nhrp summary
```

**Convergence measurement:**

```
! Measure spoke-to-spoke shortcut creation time
debug dmvpn condition peer nbma <spoke-B-NBMA>
debug nhrp packet
debug crypto ikev2

! Trigger traffic, measure time from:
!   NHRP Redirect received -> Resolution Reply received -> IPsec SA up
! Typical: 2-5 seconds (1 RTT for NHRP, 2 RTTs for IKEv2)
```

---

## Prerequisites

Before deploying DMVPN, ensure familiarity with:

- **GRE tunneling:** Point-to-point GRE configuration, tunnel interfaces, GRE keep-alives
- **IPsec fundamentals:** IKE phases (or IKEv2 exchanges), transform sets, crypto maps vs profiles, transport vs tunnel mode
- **IP routing:** At least one of EIGRP, OSPF, or BGP at a working level
- **VRF:** VRF-Lite concepts if using FVRF
- **QoS:** MQC (Modular QoS CLI) for per-tunnel QoS: class-map, policy-map, service-policy
- **Cisco IOS/IOS-XE CLI:** Interface configuration, routing protocol configuration, show/debug commands
- **IP addressing and subnetting:** Overlay and underlay addressing plans

---

## References

- RFC 2332 --- NBMA Next Hop Resolution Protocol (NHRP)
- RFC 2784 --- Generic Routing Encapsulation (GRE)
- RFC 7676 --- IPv6 Support for Generic Routing Encapsulation (GRE)
- RFC 4303 --- IP Encapsulating Security Payload (ESP)
- RFC 7296 --- Internet Key Exchange Protocol Version 2 (IKEv2)
- RFC 3706 --- A Traffic-Based Method of Detecting Dead IKE Peers
- RFC 7383 --- Internet Key Exchange Protocol Version 2 (IKEv2) Message Fragmentation
- Cisco DMVPN Design Guide --- https://www.cisco.com/c/en/us/td/docs/solutions/Enterprise/WAN_and_MAN/DMVPN_Design_Guide.html
- Cisco DMVPN Configuration Guide (IOS XE) --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_dmvpn/configuration/xe-16/sec-conn-dmvpn-xe-16-book.html
- Cisco NHRP Configuration Guide --- https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipaddr_nhrp/configuration/xe-16/nhrp-xe-16-book.html
- INE DMVPN Deep Dive --- https://ine.com/
- "DMVPN: The Definitive Guide" by Brad Edgeworth (Cisco Press)
