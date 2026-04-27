# OSPF (Open Shortest Path First, RFC 2328 / 5340)

Link-state interior gateway protocol that floods Link-State Advertisements (LSAs) inside an area, builds an identical link-state database (LSDB) on every router, and runs Dijkstra (SPF) to compute shortest paths.

## Quick Reference

OSPFv2 = IPv4 (RFC 2328). OSPFv3 = IPv6 + IPv4 address families (RFC 5340 / 5838). Each router floods its Type 1 Router LSA inside an area; ABRs translate Type 1/2 into Type 3 summaries between areas; ASBRs inject Type 5/7 externals. Dijkstra runs per area on the LSDB. Timers default to hello 10s / dead 40s on broadcast and point-to-point, hello 30s / dead 120s on NBMA.

```
# IOS — minimum viable OSPF
router ospf 1
 router-id 10.0.0.1
 network 10.0.0.0 0.0.0.255 area 0

# Junos — minimum viable OSPF
set protocols ospf area 0.0.0.0 interface ge-0/0/0

# FRR — minimum viable OSPF
vtysh -c "configure terminal" -c "router ospf" \
       -c "ospf router-id 10.0.0.1" \
       -c "network 10.0.0.0/24 area 0"

# Inspect adjacencies
show ip ospf neighbor                   # IOS / NX-OS / IOS-XR
show ospf neighbor                      # Junos
vtysh -c "show ip ospf neighbor"        # FRR

# Dump the LSDB
show ip ospf database
show ospf database extensive
vtysh -c "show ip ospf database"
```

## OSPF in 60 Seconds

OSPF runs directly on IP protocol number 89 — no TCP, no UDP. Routers discover neighbors with Hello packets sent to AllSPFRouters (224.0.0.5) and AllDRouters (224.0.0.6). They negotiate adjacencies through a strict state machine (Down → Init → 2-Way → ExStart → Exchange → Loading → Full), exchange the LSDB via Database Description (DBD), Link-State Request (LSR), Link-State Update (LSU), and Link-State Acknowledgement (LSAck) packets, then flood any new or changed LSAs to all neighbors in the same area. Once the LSDB is identical everywhere in an area, every router runs Dijkstra and produces the same shortest-path tree, then installs OSPF routes into the RIB. ABRs link non-zero areas to the backbone (area 0); ASBRs inject external routes from other protocols.

```
+----+ Hellos +----+    DBD/LSR/LSU/LSAck    +----+
| R1 |<------>| R2 |<----------------------->| R3 |
+----+        +----+                         +----+
   |            |                               |
   |  Each builds identical LSDB, runs SPF      |
   |  Result: same shortest-path tree on all    |
   v            v                               v
 RIB add      RIB add                        RIB add
```

## OSPFv2 vs OSPFv3

OSPFv2 carries IPv4 prefixes only and runs over IPv4. OSPFv3 was rewritten for IPv6 and uses link-local addresses for adjacencies, supports multiple instances per link via Instance ID, and uses a richer LSA scheme. RFC 5838 lets OSPFv3 carry IPv4 too via address families. Authentication moved out of the protocol in OSPFv3 (use IPsec per RFC 4552 or the new built-in auth trailer in RFC 7166).

| Feature | OSPFv2 (RFC 2328) | OSPFv3 (RFC 5340) |
|---------|-------------------|-------------------|
| Address family | IPv4 only | IPv6 native, IPv4 via AF (RFC 5838) |
| Transport | IP proto 89 over IPv4 | IP proto 89 over IPv6 |
| Adjacency address | Interface IPv4 | Interface link-local IPv6 (fe80::/10) |
| Auth | Built-in (null/simple/MD5/HMAC-SHA) | IPsec (RFC 4552) or trailer (RFC 7166) |
| Multiple instances per link | No | Yes, via Instance ID byte |
| Router LSA contains prefixes | Yes (in body) | No — Intra-Area Prefix LSA (Type 9) |
| Network LSA contains prefixes | Yes | No — separate Type 9 |
| LSA flooding scope encoded | Implicit by type | Explicit S1/S2 bits in LSA header |
| Multicast addresses | 224.0.0.5 / 224.0.0.6 | ff02::5 / ff02::6 |

```
# IOS-XR / IOS — both protocols on the same interface
router ospf 1
 area 0
  interface Gi0/0
!
router ospfv3 1
 address-family ipv6 unicast
  area 0
   interface Gi0/0
```

## Areas and the Backbone

Area 0 is the backbone. Every non-zero area must connect to area 0 via at least one ABR. An ABR sits in area 0 and at least one other area, and it translates Type 1/2 LSAs into Type 3 Summary LSAs as they cross the area boundary. Type 5 externals flood throughout the AS except where blocked by stub/NSSA rules. Where direct connectivity to area 0 is impossible, a Virtual Link traverses a transit area to repair the topology — but virtual links are an emergency tool, not a design.

```
                +-------- Area 0 (backbone) --------+
                |                                   |
              [ABR1]----[R-backbone]-----[ABR2]
              /                                   \
       Area 1 (regular)                       Area 2 (stub)
        |    |                                |    |
       [R][R]                                [R][R]
                                              |
                                    no Type 5 LSAs here
```

### Area Types and What They Filter

| Area type | Type 3 in | Type 4 in | Type 5 in | Type 7 in | Default route injected |
|-----------|-----------|-----------|-----------|-----------|------------------------|
| Normal (regular) | Yes | Yes | Yes | No | No |
| Stub | Yes | No | No | No | Yes (by ABR) |
| Totally Stubby | No | No | No | No | Yes (by ABR) |
| NSSA | Yes | No | No | Yes (translated to 5 by ABR) | Optional |
| Totally NSSA | No | No | No | Yes | Yes (by ABR) |
| Backbone (area 0) | n/a | n/a | n/a | n/a | n/a |

```
# IOS — declare an area as stub / totally stubby / NSSA
router ospf 1
 area 1 stub                            ! stub area
 area 2 stub no-summary                 ! totally stubby
 area 3 nssa                            ! NSSA
 area 4 nssa no-summary                 ! totally NSSA
 area 4 nssa default-information-originate

# Junos
set protocols ospf area 0.0.0.1 stub
set protocols ospf area 0.0.0.2 stub no-summaries
set protocols ospf area 0.0.0.3 nssa
set protocols ospf area 0.0.0.4 nssa no-summaries default-lsa default-metric 10

# FRR
router ospf
 area 0.0.0.1 stub
 area 0.0.0.2 stub no-summary
 area 0.0.0.3 nssa
 area 0.0.0.4 nssa no-summary
```

## LSA Types

OSPFv2 uses these LSAs (RFC 2328 §A.4 with extensions):

| Type | Name | Scope | Originator | Carries |
|------|------|-------|------------|---------|
| 1 | Router LSA | Area | Every router | Local links + costs |
| 2 | Network LSA | Area | DR on broadcast/NBMA | Routers attached to that segment |
| 3 | Summary LSA | Area | ABR | Inter-area prefixes |
| 4 | ASBR Summary | Area | ABR | Path to ASBR router-id |
| 5 | AS External | AS | ASBR | External prefixes (E1/E2) |
| 6 | Multicast OSPF | — | — | Deprecated (RFC 1584) |
| 7 | NSSA External | NSSA | ASBR in NSSA | Externals translatable to Type 5 |
| 8 | External Attributes | — | — | Rare, BGP attributes |
| 9 | Opaque (link-local) | Link | Any | TE / SR per RFC 5250 |
| 10 | Opaque (area) | Area | Any | TE per RFC 3630 |
| 11 | Opaque (AS) | AS | Any | Domain-wide opaque data |

OSPFv3 renumbers LSAs because the function code is split out (RFC 5340 §4.4):

| Function code | Name | Scope | Equivalent OSPFv2 |
|---------------|------|-------|-------------------|
| 0x2001 | Router LSA | Area | Type 1 |
| 0x2002 | Network LSA | Link | Type 2 |
| 0x2003 | Inter-Area-Prefix LSA | Area | Type 3 |
| 0x2004 | Inter-Area-Router LSA | Area | Type 4 |
| 0x4005 | AS-External LSA | AS | Type 5 |
| 0x2006 | Group-Membership | Area | Type 6 (deprecated) |
| 0x2007 | Type-7 LSA | NSSA | Type 7 |
| 0x0008 | Link LSA | Link | new — link-local addrs |
| 0x2009 | Intra-Area-Prefix LSA | Area | new — prefixes per Router/Network LSA |

```
# IOS — show one LSA in detail
show ip ospf database router 10.0.0.1 self-originate
show ip ospf database network adv-router 10.0.0.2
show ip ospf database external 0.0.0.0
show ip ospf database opaque-area

# Junos — same idea
show ospf database router lsa-id 10.0.0.1 detail
show ospf database opaque-area extensive

# FRR
vtysh -c "show ip ospf database router"
vtysh -c "show ip ospf database external 0.0.0.0"
```

Sample Type 1 (Router) LSA from IOS:

```
            OSPF Router with ID (10.0.0.1) (Process ID 1)
                  Router Link States (Area 0)
LS age: 412
Options: (No TOS-capability, DC)
LS Type: Router Links
Link State ID: 10.0.0.1
Advertising Router: 10.0.0.1
LS Seq Number: 80000019
Checksum: 0xA12C
Length: 60
Number of Links: 2
  Link connected to: a Transit Network
   (Link ID) Designated Router address: 10.10.0.2
   (Link Data) Router Interface address: 10.10.0.1
    Number of MTID metrics: 0
     TOS 0 Metrics: 1
  Link connected to: a Stub Network
   (Link ID) Network/subnet number: 10.99.0.0
   (Link Data) Network Mask: 255.255.255.0
    Number of MTID metrics: 0
     TOS 0 Metrics: 10
```

## Adjacency States

```
Down --Hello received--> Init --2-way Hello--> 2-Way
                                                |
                                  (DR/BDR election done)
                                                v
2-Way --DBD start--> ExStart --DBD master/slave--> Exchange
                                                     |
                                       LSR/LSU exchange
                                                     v
Exchange --done--> Loading --LSDB synced--> Full
```

| State | What's happening | Common stall reason |
|-------|------------------|---------------------|
| Down | No Hellos received | Layer 1/2 issue, ACL blocking proto 89, multicast filtered |
| Attempt | NBMA only — sending unicast Hellos | Wrong neighbor address, no L3 path |
| Init | One-way Hello — peer not seeing me | One-way ARP, asymmetric ACL, mismatched subnet |
| 2-Way | Both see each other; on broadcast networks DR/BDR election now happens | Stuck if all priorities are 0, expected on DROther↔DROther pairs |
| ExStart | Negotiating master/slave for DBD | MTU mismatch (most common), duplicate router-ids |
| Exchange | DBD packets flowing | LSAck loss, ACL flapping |
| Loading | LSR/LSU in flight to fill in missing LSAs | LSU drops from QoS or rate limits |
| Full | LSDB synchronized with this neighbor | — |

```
R1#show ip ospf neighbor
Neighbor ID     Pri   State           Dead Time   Address         Interface
10.0.0.2          1   FULL/DR         00:00:35    10.10.0.2       GigabitEthernet0/0
10.0.0.3          1   FULL/BDR        00:00:32    10.10.0.3       GigabitEthernet0/0
10.0.0.4          1   2-WAY/DROTHER   00:00:33    10.10.0.4       GigabitEthernet0/0
```

The 2-WAY/DROTHER row is normal: DROthers do not form full adjacencies with each other — only with the DR and BDR.

## Network Types

```
+------------------+------------------+------------------+
| Network type     | DR/BDR election? | Default H/D (s)  |
+------------------+------------------+------------------+
| Broadcast        | Yes              | 10 / 40          |
| Point-to-Point   | No               | 10 / 40          |
| Point-to-Multi   | No               | 30 / 120         |
| NBMA             | Yes              | 30 / 120         |
| Loopback         | n/a (always /32) | n/a              |
+------------------+------------------+------------------+
```

```
# IOS — override the network type
interface Gi0/0
 ip ospf network point-to-point
 ip ospf network non-broadcast
 ip ospf network point-to-multipoint

# Junos
set protocols ospf area 0.0.0.0 interface ge-0/0/0 interface-type p2p
set protocols ospf area 0.0.0.0 interface ge-0/0/0 interface-type nbma

# FRR
interface eth0
 ip ospf network point-to-point
 ip ospf network non-broadcast
```

Loopbacks are always advertised as /32 even when the underlying mask is shorter — to fix this on IOS use `ip ospf network point-to-point` on the loopback so the actual mask is used. This trips up newcomers building lab fabrics with /24 loopbacks.

## DR/BDR Election

On broadcast and NBMA networks, OSPF elects a Designated Router (DR) and a Backup Designated Router (BDR) so flooding scales as O(N) rather than O(N²). Only DR↔DROther and BDR↔DROther pairs go to FULL; DROther↔DROther pairs stop at 2-WAY. The DR generates the Type 2 Network LSA listing every router on the segment.

Election rules:

1. The router with the highest interface priority wins. Default is 1, range 0–255.
2. Ties are broken by the highest router-id.
3. Priority 0 means the router never becomes DR or BDR.
4. The election is non-preemptive — a higher-priority router that comes up later does not displace the existing DR.

```
# IOS — set priority and force a DR
interface Gi0/0
 ip ospf priority 200            ! likely DR
interface Gi0/1
 ip ospf priority 0              ! never DR

# Junos
set protocols ospf area 0.0.0.0 interface ge-0/0/0 priority 200

# FRR
interface eth0
 ip ospf priority 200
```

```
# DR/BDR flooding logic on a broadcast segment
                    [ DR ]
                  /   |   \
                 /    |    \
            [DROther][BDR][DROther]
                 \    |    /
                  \   |   /
              224.0.0.5 (AllSPFRouters) used by DR/BDR to flood
              224.0.0.6 (AllDRouters)   used by DROthers to send
```

DROther floods to AllDRouters (224.0.0.6); only DR and BDR listen there. The DR re-floods on AllSPFRouters (224.0.0.5) so every router on the segment sees the LSA.

On point-to-point and point-to-multipoint there is no DR. Stretching a hub-and-spoke topology over Frame Relay/DMVPN is the classic place where forgetting DR rules makes everything fall over — fix it with `network point-to-multipoint` or static neighbor statements.

## SPF Algorithm

Each router runs Dijkstra against its area's LSDB to build a shortest-path tree rooted at itself. Algorithm complexity is O((N + E) log N) using a Fibonacci heap, O(N²) using an array — for typical IGP topologies (a few hundred nodes, sparse graph) it's microseconds to a few milliseconds. iSPF (incremental SPF) recomputes only the affected branches when one LSA changes; partial SPF skips rebuilding when only a Type 3/5 changes (it just relabels prefixes, not topology).

```
# Worked example — 4-router triangle plus stub
                cost 1
        R1 ----------- R2
         | \            |
   cost 5|  \cost 3     |cost 2
         |   \          |
        R3----\---------R4
                cost 4
        |
       LAN 10.99/24 (cost 10 stub)

R1 root SPF:
  R1 (0)
   ├─ R2 (1)        via R1→R2
   │   └─ R4 (3)    via R1→R2→R4
   ├─ R4 (3)        chosen via R2 (3 < 5+? compare paths)
   ├─ R3 (5)        via R1→R3 direct
Stub LAN: R3→10.99/24 cost 10 → from R1: 5+10 = 15
```

```
# IOS — SPF throttling (timers in ms)
router ospf 1
 timers throttle spf 50 200 5000     ! initial-delay min-hold max-wait
 timers throttle lsa 0 200 5000

# Show SPF stats
show ip ospf statistics
show ip ospf | include SPF

! Last SPF run was 0.000124s ago, ran for 1284 microseconds
```

```
# Junos
set protocols ospf spf-options delay 50 holddown 200 rapid-runs 3
show ospf statistics

# FRR
router ospf
 timers throttle spf 50 200 5000
 timers throttle lsa all 0 200 5000
```

## Authentication

OSPFv2 has built-in authentication; OSPFv3 historically delegated to IPsec but RFC 7166 added a native authentication trailer. Always pick the strongest your peer supports; HMAC-SHA-256 (RFC 5709) is the modern minimum for OSPFv2.

```
# IOS — keychain with HMAC-SHA-256
key chain OSPF-KC
 key 1
  key-string MyOSPFSecret
  cryptographic-algorithm hmac-sha-256
  send-lifetime 00:00:00 Jan 1 2024 infinite
  accept-lifetime 00:00:00 Jan 1 2024 infinite

interface Gi0/0
 ip ospf authentication key-chain OSPF-KC

# Older MD5 form (still common)
interface Gi0/0
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MyMD5Secret

router ospf 1
 area 0 authentication message-digest
```

```
# Junos — area-level MD5
set protocols ospf area 0.0.0.0 authentication-type md5
set protocols ospf area 0.0.0.0 interface ge-0/0/0 authentication md5 1 key MyMD5Secret

# FRR
interface eth0
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MyMD5Secret
```

```
# OSPFv3 — IPsec ESP (RFC 4552) on Junos
set protocols ospf3 area 0.0.0.0 interface ge-0/0/0 ipsec-sa OSPF3-SA
set security ipsec security-association OSPF3-SA mode transport \
    manual direction bidirectional protocol esp authentication algorithm \
    hmac-sha1-96 key ascii-text MyOSPFv3Secret
```

## Convergence Tuning

| Knob | Default | Tunable | Effect |
|------|---------|---------|--------|
| Hello interval | 10s broadcast/p2p | 1–65535s | How often Hellos are sent |
| Dead interval | 4×Hello | Any | Time without Hello before neighbor down |
| LSA arrival | 1000ms | 0–600000ms | Min interval between same-LSA arrivals |
| LSA group pacing | 240s | 10–1800s | How often LSAs are batched for refresh |
| Retransmit interval | 5s | 1–8192s | LSA retransmit cadence on unACKed packets |
| Transmit delay | 1s | 0–8192s | Estimated link latency added to LSA age |
| SPF throttle | varies | initial / hold / max | Delays SPF after triggers |

```
# IOS — tight convergence on a P2P link with BFD
interface Gi0/0
 ip ospf network point-to-point
 ip ospf hello-interval 3
 ip ospf dead-interval 12
 ip ospf bfd

router ospf 1
 timers throttle spf 50 200 5000
 timers throttle lsa 0 200 5000
 timers lsa arrival 100
 bfd all-interfaces

bfd-template single-hop FAST
 interval min-tx 50 min-rx 50 multiplier 3
```

In practice, lean on BFD instead of dropping Hello below 3 seconds — cheap CPUs, BGP, EIGRP, IS-IS, and OSPF can all share a single BFD session per link.

## Cisco IOS / IOS-XR / NX-OS

```
! IOS — full minimal config
hostname R1
ip multicast-routing
!
router ospf 1
 router-id 10.0.0.1
 auto-cost reference-bandwidth 100000
 passive-interface default
 no passive-interface GigabitEthernet0/0
 no passive-interface GigabitEthernet0/1
 area 0 authentication message-digest
 network 10.10.0.0 0.0.0.3 area 0
 network 10.10.0.4 0.0.0.3 area 0
 network 10.99.0.0 0.0.0.255 area 1
 area 1 stub no-summary
 default-information originate always metric 1 metric-type 1
!
interface GigabitEthernet0/0
 description to-R2
 ip address 10.10.0.1 255.255.255.252
 ip ospf network point-to-point
 ip ospf hello-interval 3
 ip ospf dead-interval 12
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MyMD5Secret
 ip ospf bfd
!
interface Loopback0
 ip address 10.0.0.1 255.255.255.255
 ip ospf network point-to-point
 ip ospf 1 area 0
```

```
! IOS-XR — same idea, hierarchical syntax
router ospf 1
 router-id 10.0.0.1
 auto-cost reference-bandwidth 100000
 area 0
  authentication message-digest
  interface Loopback0
   passive enable
  interface GigabitEthernet0/0/0/0
   network point-to-point
   hello-interval 3
   dead-interval 12
   authentication message-digest
   message-digest-key 1 md5 MyMD5Secret
   bfd fast-detect
 area 1
  stub no-summary
  interface GigabitEthernet0/0/0/1
```

```
! NX-OS — feature-gated
feature ospf
feature bfd
!
router ospf 1
 router-id 10.0.0.1
 auto-cost reference-bandwidth 100000 Gbps
 timers throttle spf 50 200 5000
 default-information originate always
!
interface Ethernet1/1
 ip router ospf 1 area 0.0.0.0
 ip ospf network point-to-point
 ip ospf hello-interval 3
 ip ospf dead-interval 12
 ip ospf bfd
```

### Cisco show / debug commands

```
show ip ospf
show ip ospf 1
show ip ospf neighbor [detail]
show ip ospf interface [brief]
show ip ospf database
show ip ospf database router self-originate
show ip ospf database external
show ip ospf border-routers
show ip ospf statistics [detail]
show ip ospf rib
show ip route ospf
show ip ospf events
show ip ospf flood-list

debug ip ospf adj
debug ip ospf events
debug ip ospf hello
debug ip ospf packet
debug ip ospf lsa-generation
debug ip ospf flood
```

Sample `show ip ospf interface brief`:

```
R1#show ip ospf interface brief
Interface    PID   Area            IP Address/Mask    Cost  State Nbrs F/C
Lo0          1     0               10.0.0.1/32        1     LOOP  0/0
Gi0/0        1     0               10.10.0.1/30       1     P2P   1/1
Gi0/1        1     1               10.99.0.1/24       1     DR    2/2
```

## Juniper Junos

```
# Junos — full config
set routing-options router-id 10.0.0.1
set protocols ospf reference-bandwidth 100g
set protocols ospf area 0.0.0.0 interface lo0.0 passive
set protocols ospf area 0.0.0.0 interface ge-0/0/0 interface-type p2p
set protocols ospf area 0.0.0.0 interface ge-0/0/0 hello-interval 3
set protocols ospf area 0.0.0.0 interface ge-0/0/0 dead-interval 12
set protocols ospf area 0.0.0.0 interface ge-0/0/0 bfd-liveness-detection minimum-interval 50 multiplier 3
set protocols ospf area 0.0.0.0 authentication-type md5
set protocols ospf area 0.0.0.0 interface ge-0/0/0 authentication md5 1 key MyMD5Secret
set protocols ospf area 0.0.0.1 stub no-summaries
set protocols ospf area 0.0.0.1 interface ge-0/0/1
set protocols ospf default-lsa default-metric 1 metric-type 1 always
```

### Junos show commands

```
show ospf overview
show ospf neighbor [detail]
show ospf interface [extensive]
show ospf database [extensive]
show ospf database router lsa-id 10.0.0.1 detail
show ospf database external advertising-router 10.0.0.7
show ospf statistics
show ospf log
show route protocol ospf
show route protocol ospf extensive
```

Sample `show ospf neighbor extensive`:

```
labroot@R1> show ospf neighbor extensive
Address          Interface              State     ID               Pri  Dead
10.10.0.2        ge-0/0/0.0             Full      10.0.0.2         128    35
  Area 0.0.0.0, opt 0x52, DR 10.10.0.2, BDR 0.0.0.0
  Up 02:14:11, adjacent 02:14:09
  Topology default (ID 0) -> Bidirectional
```

## FRR / Quagga

```
! /etc/frr/frr.conf
hostname R1
log syslog informational
!
interface lo
 ip ospf area 0
!
interface eth0
 ip ospf area 0
 ip ospf network point-to-point
 ip ospf hello-interval 3
 ip ospf dead-interval 12
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MyMD5Secret
 ip ospf bfd
!
router ospf
 ospf router-id 10.0.0.1
 auto-cost reference-bandwidth 100000
 passive-interface default
 no passive-interface eth0
 area 0 authentication message-digest
 area 1 stub no-summary
 default-information originate always metric 1 metric-type 1
 timers throttle spf 50 200 5000
 timers throttle lsa all 0 200 5000
!
ip prefix-list ALLOW seq 5 permit 10.0.0.0/8 le 24
route-map STATIC-IN permit 10
 match ip address prefix-list ALLOW
!
router ospf
 redistribute static route-map STATIC-IN
```

### FRR commands

```
vtysh
R1# show ip ospf
R1# show ip ospf neighbor
R1# show ip ospf interface
R1# show ip ospf database
R1# show ip ospf database router self-originate
R1# show ip ospf border-routers
R1# show ip ospf route
R1# show ip route ospf
R1# show running-config ospfd
R1# show ip ospf json                  # FRR ≥ 7.0 emits JSON

# Or one-shot from the shell
vtysh -c "show ip ospf neighbor json"
vtysh -c "show ip ospf database router self-originate"
```

## OSPF on Linux (FRR daemon)

```
# /etc/frr/daemons (Debian/Ubuntu)
zebra=yes
ospfd=yes
ospf6d=yes
bfdd=yes

# Enable + start
systemctl enable --now frr

# Status
systemctl status frr
journalctl -u frr -f

# Live config edits via vtysh
sudo vtysh
configure terminal
router ospf
 ospf router-id 10.0.0.1
 network 10.0.0.0/24 area 0
end
write memory
```

```
# Verify the kernel installed OSPF routes
ip route show proto ospf
# 10.0.1.0/24 via 10.10.0.2 dev eth0  proto ospf  metric 20
# 10.0.2.0/24 via 10.10.0.2 dev eth0  proto ospf  metric 30

# Check the OSPF process directly
vtysh -c "show ip ospf neighbor"
vtysh -c "show ip ospf database"
```

## OSPFv3 (IPv6) Quick Config

```
# IOS-XR OSPFv3
router ospfv3 1
 router-id 10.0.0.1
 area 0
  interface Loopback0
  interface GigabitEthernet0/0/0/0
   network point-to-point

# Junos
set protocols ospf3 area 0.0.0.0 interface lo0.0 passive
set protocols ospf3 area 0.0.0.0 interface ge-0/0/0 interface-type p2p

# FRR (separate daemon ospf6d)
router ospf6
 ospf6 router-id 10.0.0.1
 interface eth0 area 0.0.0.0
 interface lo area 0.0.0.0

# Show
show ipv6 ospf neighbor
show ipv6 ospf database
show ipv6 route ospf
```

## Sample Configurations

### 1. Two-router /30 backbone

```
! R1                                          ! R2
interface Gi0/0                               interface Gi0/0
 ip address 10.10.0.1 255.255.255.252          ip address 10.10.0.2 255.255.255.252
 ip ospf network point-to-point                ip ospf network point-to-point
 ip ospf 1 area 0                              ip ospf 1 area 0
!                                             !
router ospf 1                                 router ospf 1
 router-id 10.0.0.1                            router-id 10.0.0.2
```

After commit:

```
R1#show ip ospf neighbor
Neighbor ID  Pri  State    Dead Time   Address      Interface
10.0.0.2       0  FULL/-   00:00:39    10.10.0.2    GigabitEthernet0/0
```

### 2. Stub area to reduce LSDB size

```
! ABR (R1) and ASBR (Rext) somewhere upstream
router ospf 1
 router-id 10.0.0.1
 network 10.10.0.0 0.0.0.3 area 0
 network 10.99.0.0 0.0.0.255 area 1
 area 1 stub                              ! ABR injects 0/0 into area 1
!
! Internal router R2 in area 1
router ospf 1
 router-id 10.0.0.2
 network 10.99.0.0 0.0.0.255 area 1
 area 1 stub                              ! must match on every area-1 router
```

```
R2#show ip route ospf
O*IA 0.0.0.0/0 [110/2] via 10.99.0.1, 00:00:14, GigabitEthernet0/0
```

### 3. Multi-area with ABR summarization

```
! Area 1 has 10.1.0.0–10.1.15.255, summarize as 10.1.0.0/20 into area 0
router ospf 1
 area 1 range 10.1.0.0 255.255.240.0       ! IOS uses subnet mask
! IOS-XR
router ospf 1
 area 1
  range 10.1.0.0/20
! Junos
set protocols ospf area 0.0.0.1 area-range 10.1.0.0/20
! FRR
router ospf
 area 0.0.0.1 range 10.1.0.0/20
```

After SPF the backbone sees one Type 3 LSA covering the whole /20:

```
R3#show ip ospf database summary
            OSPF Router with ID (10.0.0.3) (Process ID 1)
                Summary Net Link States (Area 0)
LS age: 18
Options: (No TOS-capability, DC, Upward)
LS Type: Summary Links (Network)
Link State ID: 10.1.0.0 (summary Network Number)
Advertising Router: 10.0.0.1
LS Seq Number: 80000007
Network Mask: /20
        TOS: 0 Metric: 10
```

### 4. NSSA with external redistribution

```
! ASBR sits in NSSA area 2, redistributing static routes
router ospf 1
 router-id 10.0.0.5
 network 10.20.0.0 0.0.0.255 area 2
 area 2 nssa
 redistribute static subnets metric 20 metric-type 1

ip route 192.0.2.0 255.255.255.0 Null0
```

The NSSA generates Type 7 LSAs inside area 2; the ABR translates them to Type 5 and floods them throughout the rest of the AS. Backbone routers see external prefix 192.0.2.0/24 with metric type E1 (cost adds path cost) or E2 (cost stays at originator's metric).

### 5. Authentication with MD5 keys

```
! Per-interface
interface Gi0/0
 ip ospf authentication message-digest
 ip ospf message-digest-key 1 md5 MySecret

! Or area-wide (cleaner)
router ospf 1
 area 0 authentication message-digest
```

To roll a key without dropping adjacencies, configure the new key under a new index on every router first, then remove the old one:

```
! Phase 1 — add key 2 everywhere
ip ospf message-digest-key 2 md5 NewSecret

! Phase 2 — remove key 1 once all neighbors send key 2
no ip ospf message-digest-key 1
```

### 6. Virtual link to repair a broken backbone

```
! ABR1 in area 0+1, ABR2 in area 1+2; area 2 has lost its area-0 connection
! Build a virtual link across area 1 between ABR1 (rid 10.0.0.1) and ABR2 (rid 10.0.0.2)
! Configure on both routers, naming the *other* router-id

! ABR1
router ospf 1
 area 1 virtual-link 10.0.0.2

! ABR2
router ospf 1
 area 1 virtual-link 10.0.0.1
```

### 7. Default-information-originate from an internet-facing router

```
ip route 0.0.0.0 0.0.0.0 198.51.100.1
router ospf 1
 default-information originate always metric 1 metric-type 1
```

`always` flags the route as advertised even if the static dies — pair with object tracking if you want the default to follow real reachability.

## Common Errors and Diagnostics

```
%OSPF-4-ERRRCV: Received invalid packet: mismatched area from 10.10.0.2 ...
  Fix: both ends of the link must agree on area. Check `show ip ospf interface`.

%OSPF-4-MISMATCH_HELLO: Mismatched hello parameters from 10.10.0.2 ...
  Fix: hello-interval and dead-interval must match exactly.

%OSPF-4-MISMATCH_AREA_TYPE: Received Hello packet declaring area as ...
  Fix: stub/NSSA flag must match on every router in the area.

%OSPF-5-ADJCHG: Process 1, Nbr 10.0.0.2 on Gi0/0 from EXSTART to DOWN, Neighbor Down: ...
  Fix: usually MTU mismatch. Check `show interface Gi0/0 | include MTU` on both ends.

%OSPF-4-DUP_RTRID_NBR: OSPF detected duplicate router-id 10.0.0.1 from 10.10.0.2 on int Gi0/0
  Fix: explicitly set router-id on every router; never let the box auto-pick.

%OSPF-4-CONFLICTING_LSAID: Detected router with conflicting LSA ID ...
  Fix: two routers are advertising the same Link State ID; usually summary or external collisions.

%OSPF-4-NOVALIDKEY: No key found, packet from 10.10.0.2 dropped
  Fix: keychain mismatch or expired key; check accept-lifetime windows.

%OSPF-4-FLOOD_WAR: OSPF process flooding too fast ...
  Fix: an LSA is changing too often (flapping link). Look at `show ip ospf flood-list`.
```

```
# Junos error log signatures
ospfd[12345]: KRT: error sending OSPF route: file exists
  Fix: route already in RIB from another protocol with better preference; use route-map or import policy.

rpd[12345]: bgp_recv: OSPF area mismatch from neighbor 10.10.0.2: my=0.0.0.0 peer=0.0.0.1
  Fix: same as IOS area mismatch.

# FRR signatures
ospfd: Packet[DD]: Neighbor 10.0.0.2 MTU 1492 is larger than [eth0]'s MTU 1500
  Fix: lower interface MTU on the high side or `ip ospf mtu-ignore` on both ends.
```

### "Stuck in ExStart" — the canonical MTU bug

ExStart is where the master/slave for DBD exchange is negotiated. If one router's interface MTU is bigger than the other's, the larger DBD packet is silently dropped, retransmits accumulate, and adjacency never advances.

```
R1#show ip ospf neighbor
Neighbor ID    Pri  State           Dead Time   Address      Interface
10.0.0.2         0  EXSTART/-       00:00:39    10.10.0.2    Gi0/0
R1#show interface Gi0/0 | include MTU
  MTU 9000 bytes, BW 1000000 Kbit/sec, DLY 10 usec
R2#show interface Gi0/0 | include MTU
  MTU 1500 bytes, BW 1000000 Kbit/sec, DLY 10 usec
```

Fix: align MTU. If you absolutely cannot, `ip ospf mtu-ignore` on both ends — but you've now hidden a real misconfiguration that will bite the data plane.

### "Stuck in Init" — one-way visibility

```
R1#show ip ospf neighbor
10.0.0.2  0  INIT/-  00:00:38  10.10.0.2  Gi0/0
```

R1 receives Hellos but R2 doesn't — usually one of:

- ACL on R1 is blocking egress to 224.0.0.5 / proto 89
- `ip multicast boundary` filter on the interface
- ARP entry mismatch (replace cabling, restart neighbor)
- Asymmetric VLAN — R1 sees R2 untagged, R2 sees R1 tagged

```
debug ip ospf hello
*Apr 27 12:00:01.234: OSPF: Send hello to 224.0.0.5 area 0 on Gi0/0 from 10.10.0.1
*Apr 27 12:00:01.567: OSPF: Rcv hello from 10.0.0.2 area 0 from Gi0/0 10.10.0.2
*Apr 27 12:00:01.567: OSPF: Hello packet from 10.10.0.2 has neighbor list:
                                  (empty — confirms one-way)
```

### "Stuck in 2-Way" — DR/BDR mismatch on broadcast

If three or more routers all have priority 0, no DR is elected and everyone sits at 2-WAY. Set priority on at least two routers:

```
interface Gi0/0
 ip ospf priority 100
```

### "Stuck in Loading" — DBD reordering or dropped LSU

```
debug ip ospf flood
debug ip ospf packet
```

Look for retransmits. Often caused by aggressive QoS shaping or microbursts dropping multicast.

## Troubleshooting Workflow

```
1. show ip ospf neighbor
   - Adjacency at FULL? Done.
   - INIT?      → Hello hits one-way. Check ACL, multicast, MTU, area.
   - 2-WAY?     → DROther↔DROther is normal; otherwise priority/DR issue.
   - EXSTART?   → MTU mismatch. Compare interface MTU on both sides.
   - EXCHANGE?  → DBD acks dropping. Check QoS, ACLs.
   - LOADING?   → LSU loss. Increase retransmit-interval or fix flapping link.
2. show ip ospf interface <iface>
   - Confirm hello-interval, dead-interval, area, network type, auth.
3. show ip ospf
   - Confirm router-id, ABR/ASBR status, area summary.
4. show ip ospf database
   - Each router has the same number of Type 1/2 LSAs in an area.
   - If LSDB drifts, flooding is broken — check link state, ACLs.
5. show ip route ospf
   - Confirm SPF computed and installed routes.
6. debug ip ospf adj
   - Verbose state-machine logs for the failing neighbor only — use ACL.
```

```
# Scope debug to one neighbor only
ip access-list extended OSPF-NBR-DEBUG
 permit ip host 10.10.0.2 host 10.10.0.1
 permit ip host 10.10.0.1 host 10.10.0.2
debug ip ospf adj 10
debug ip ospf hello
```

## OSPF and BFD

OSPF Hello can fall as low as 1s (subseconds via `dead-interval minimal hello-multiplier`), but tight Hellos chew CPU and don't help on multipoint media. BFD (RFC 5880) detects loss of liveness in tens of milliseconds with one shared session per link, and OSPF just registers a callback.

```
! IOS — global BFD plus OSPF hookup
bfd-template single-hop FAST
 interval min-tx 50 min-rx 50 multiplier 3

interface Gi0/0
 bfd template FAST
 ip ospf bfd

router ospf 1
 bfd all-interfaces
```

```
# Junos
set protocols bfd interface ge-0/0/0 minimum-interval 50 multiplier 3
set protocols ospf area 0.0.0.0 interface ge-0/0/0 bfd-liveness-detection minimum-interval 50 multiplier 3

# FRR
interface eth0
 ip ospf bfd
!
bfd
 peer 10.10.0.2 interface eth0
  detect-multiplier 3
  receive-interval 50
  transmit-interval 50
  no shutdown
```

```
R1#show bfd neighbor
NeighAddr               LD/RD     RH/RS     State     Int
10.10.0.2               1/1       Up        Up        Gi0/0
```

When the BFD session goes down, OSPF tears the adjacency in <200ms instead of waiting for the dead interval.

## OSPF and Traffic Engineering

OSPF carries TE info via opaque LSAs (Type 10, area-scoped, RFC 3630). Each link is annotated with admin group, max bandwidth, max reservable, unreserved bandwidth per priority, and TE metric. RSVP-TE consumes the TE database (TED) to compute constrained shortest paths (CSPF).

Segment Routing (RFC 8665) extends OSPF with SR-MPLS labels. Each router advertises an SR Global Block (SRGB) and per-prefix Node SID; per-link Adjacency SID. Flex-Algo (RFC 9350) layers user-defined metrics (latency, TE-metric, IGP-metric) onto the SPF computation, giving multiple co-existing topologies in a single OSPF instance.

```
! IOS-XR — OSPF + MPLS-TE + SR
mpls traffic-eng
router ospf 1
 mpls traffic-eng router-id Loopback0
 area 0
  mpls traffic-eng

segment-routing mpls
 global-block 16000 23999
router ospf 1
 segment-routing mpls
 segment-routing prefix-sid-map advertise-local
```

```
! Show TE database
show ip ospf mpls traffic-eng database
show mpls traffic-eng topology
```

## Worked Math Examples

### Cost from bandwidth

`cost = max(1, reference-bandwidth / interface-bandwidth)` (both in same units).

```
ref-bw = 100 Mbps (default)
  100 Mbps link  -> 100 / 100 = 1
  10  Mbps link  -> 100 / 10  = 10
  1   Gbps link  -> 100 / 1000 = 0 -> rounded to 1 (problem!)
  10  Gbps link  -> 100 / 10000 = 0 -> rounded to 1

ref-bw = 100 Gbps  (auto-cost reference-bandwidth 100000)
  100 Mbps link  -> 100000 / 100   = 1000
  1   Gbps link  -> 100000 / 1000  = 100
  10  Gbps link  -> 100000 / 10000 = 10
  100 Gbps link  -> 100000 / 100000 = 1

ref-bw = 1 Tbps   (auto-cost reference-bandwidth 1000000)
  10  Gbps link  -> 1000000 / 10000  = 100
  100 Gbps link  -> 1000000 / 100000 = 10
  1   Tbps link  -> 1000000 / 1000000 = 1
```

Verify with cs:

```
cs calc -- '100000 / 1000'      # 100
cs calc -- '100000 / 10000'     # 10
```

### LSDB size estimation

LSDB ≈ Σ (router-LSAs + network-LSAs + summary-LSAs + external-LSAs).

For a single area with N routers and L average links per router:

- Router LSAs: N (one per router), each with L link descriptors @ 12 bytes
  ≈ N × (24 + 12L) bytes
- Network LSAs: B (broadcast segments), average M routers attached, each adds 4 bytes
  ≈ B × (24 + 4M) bytes
- Type 3 summaries injected by ABRs: P prefixes × 28 bytes
- Type 5 externals: E prefixes × 36 bytes

Worked numbers for 200 routers, avg 4 links each, 50 transit nets, 1000 inter-area prefixes, 500 externals:

```
Router LSAs : 200 × (24 + 12 × 4)   = 200 × 72  = 14400 B
Network LSAs:  50 × (24 + 4 × 8)    =  50 × 56  =  2800 B
Type 3      : 1000 × 28             =          = 28000 B
Type 5      :  500 × 36             =          = 18000 B
                                        Total ≈ 63 KiB
```

### SPF runtime

For an area with N routers and E links, Dijkstra is O((N + E) log N) with a heap. Modern CPUs can chew through a 1000-router LSDB in 1–5 ms. iSPF (incremental) typically processes 10–100 nodes after a single LSA change.

### Convergence budget

```
Total convergence = LSA generation delay
                  + flooding latency (per hop ~ ms)
                  + LSA arrival pacing (default 1000 ms)
                  + SPF throttle initial-delay
                  + SPF runtime
                  + RIB/FIB install
```

Tight production targets:

```
LSA-arrival 100 ms
+ flooding (3 hops × ~1 ms)
+ SPF throttle 50 ms
+ SPF runtime ~5 ms
+ FIB install ~50 ms
≈ 210 ms end-to-end
```

For sub-200ms, drop LSA-arrival to 0 and rely on BFD for the trigger event.

```
cs calc -- '100 + 3 + 50 + 5 + 50'   # 208
```

### Hello loss detection

Default broadcast: dead-interval = 4 × hello = 40 s. With BFD min-tx 50 ms × multiplier 3, detection is 150 ms — a 266× improvement.

```
cs calc -- '40 * 1000 / (50 * 3)'   # 266.67
```

## Tips & Idioms

- Always set `router-id` explicitly. Auto-pick changes after reboots if interfaces re-order.
- `passive-interface default`, then opt-in with `no passive-interface` per uplink. Reduces attack surface and stops accidental adjacency formation on user-facing ports.
- On point-to-point links — including loopbacks and unnumbered fabrics — set `network point-to-point` to skip DR election and advertise the actual mask instead of /32.
- Bump `auto-cost reference-bandwidth` to match your fastest link (100000 = 100 Gbps). Otherwise 1 Gbps and 100 Gbps both cost 1 and SPF picks at random.
- Use `ip ospf mtu-ignore` only as a last resort — it hides a real misconfiguration. Fix the MTU instead.
- Stubs and totally-stubby areas are huge LSDB savings for branch sites — push them everywhere you don't need full visibility.
- Summarize at ABRs (`area X range`) and ASBRs (`summary-address`). Never inject a /32 into an inter-area Type 3 unless you mean it.
- Roll authentication keys with overlap (configure new key, wait, remove old) to avoid resetting adjacencies.
- For data-center fabrics, prefer BGP unnumbered (RFC 5549) or BGP-EVPN over OSPF unless your team already runs OSPF in production.
- For massive SP networks, consider IS-IS — same algorithm, simpler scaling for DC fabrics, no IP dependency for control plane.
- Pair OSPF with BFD on every transit link. Hello-interval 1 with sub-second dead is a power tool that costs CPU; BFD is purpose-built.
- Watch for `LSA arrival` and `SPF` rate limits when running tight convergence — over-aggressive throttles cause silent SPF storms.
- Wireshark filter `ospf` and the dissector decodes every LSA field. Capture on a span for any persistent ExStart problem and read the DBD MTU value.
- For NSSAs, set `no-redistribution` on the ABR so locally-redistributed routes don't leak back into Type 5.
- Use `area 0 authentication message-digest` rather than per-interface — fewer places to forget.
- Document virtual links wherever they exist; they age out of memory quickly and become invisible failure modes.

## Comparison: OSPF vs IS-IS vs EIGRP

| Feature | OSPF | IS-IS | EIGRP |
|---------|------|-------|-------|
| RFC / origin | RFC 2328 / 5340 | ISO 10589 / RFC 1142 | RFC 7868 (Cisco) |
| Algorithm | Dijkstra | Dijkstra | DUAL (diffusing update) |
| Layer / encapsulation | IP proto 89 | Directly on Layer 2 (CLNS) | IP proto 88 |
| Address family | IPv4 + IPv6 (separate) | IPv4 + IPv6 (Multi-Topology) | IPv4 + IPv6 |
| Areas | Two-level (backbone + non-zero) | Two-level (Level-1 + Level-2) | Flat (no areas) |
| Hello | Multicast 224.0.0.5 / ff02::5 | Multicast L2 only | Multicast 224.0.0.10 |
| Default Hello / Dead | 10 / 40 (broadcast) | 9 / 27 (point-to-point) | 5 / 15 |
| Authentication | MD5 / HMAC-SHA / IPsec | MD5 / HMAC-SHA | MD5 / HMAC-SHA |
| Convergence | Sub-second w/ BFD | Sub-second w/ BFD | DUAL FS in microseconds |
| Scalability sweet spot | 100–500 routers per area | 1000+ in a level | 100–300 routers |
| MPLS-TE support | Yes (RFC 3630 opaque LSA) | Yes (TLV 22, sub-TLVs) | No native |
| Segment Routing | RFC 8665 | RFC 8667 | No |
| LSP/LSA refresh | 30 min (default) | 15 min (default) | n/a |
| Vendor support | Universal | Universal (SP-heavy) | Cisco + FRR (post-2013) |
| Best for | Enterprise, mixed | SP backbones, large fabrics | Cisco-only enterprise |

## Comparison: OSPFv2 vs OSPFv3 (deeper)

| Aspect | OSPFv2 | OSPFv3 |
|--------|--------|--------|
| Header size | 24 B | 16 B (no auth fields) |
| Authentication | In-protocol | IPsec / RFC 7166 trailer |
| Per-link multiple instances | No | Yes, Instance ID |
| LSA types numbered | 1–11 | Function code + scope bits |
| Prefix info location | In Router/Network LSA | In Intra-Area-Prefix LSA |
| Address families | IPv4 only | v6 native, v4 via RFC 5838 |
| Stub / NSSA | Yes | Yes |
| Hello packet field | Network mask, options, hello | Interface ID, options, hello — no mask |
| Default ACL | Allow IP proto 89 | Allow IPv6 proto 89, link-local sourced |

## Programmatic / API Access

```
# IOS-XR — gNMI
gnmic -a r1:6030 -u admin -p admin --insecure get \
  --path '/network-instances/network-instance/protocols/protocol[name=ospf]/ospf/areas'

# Junos — NETCONF
ssh r1.junos netconf
<rpc><get-ospf-neighbor-information/></rpc>

# FRR — JSON output
vtysh -c 'show ip ospf neighbor json' | jq .

# FRR — northbound YANG (FRR ≥ 8)
gnmic -a r1:32767 --insecure get \
  --path '/frr-ospfd:ospf'
```

## OSPF in the Cloud / Virtualised Stacks

OSPF still pops up in NFV and overlay tunnel networks. Common gotchas:

- VPC peerings drop multicast — OSPF dies. Either use IPsec/GRE wrap or switch to BGP.
- Cloud Layer 2 (e.g., AWS Transit Gateway) is point-to-point — set `ip ospf network point-to-point`.
- VMware NSX-T edge nodes run OSPF over GRE; bump `ip ospf transmit-delay` if encap latency is high.
- VRF-aware OSPF: each VRF has its own LSDB and process; never bridge two VRFs through OSPF unintentionally.

```
! IOS — OSPF in a VRF
ip vrf TENANT-A
 rd 65000:1
 route-target both 65000:1
!
router ospf 100 vrf TENANT-A
 router-id 10.0.0.1
 network 10.10.0.0 0.0.0.255 area 0
```

## OSPF Packet Formats

OSPF uses five packet types, each with the same 24-byte (OSPFv2) or 16-byte (OSPFv3) common header.

### OSPFv2 Common Header (RFC 2328 §A.3.1)

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|   Version #   |     Type      |         Packet length         |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Router ID                            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                           Area ID                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Checksum            |             AuType            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Authentication                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                       Authentication                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

| Field | Bytes | Notes |
|-------|-------|-------|
| Version | 1 | 2 for OSPFv2 |
| Type | 1 | 1=Hello, 2=DBD, 3=LSR, 4=LSU, 5=LSAck |
| Packet length | 2 | Total bytes incl. header |
| Router ID | 4 | Originating router's ID |
| Area ID | 4 | The area this packet is in |
| Checksum | 2 | IP-style 1's complement, excluding auth fields |
| AuType | 2 | 0=Null, 1=Simple, 2=MD5/HMAC |
| Auth | 8 | Cleartext password (Simple) or sequence + key-id (MD5) |

### Hello Packet Body (Type 1)

```
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Network Mask                           |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         HelloInterval         |    Options    |    Rtr Pri    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     RouterDeadInterval                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                      Designated Router                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                   Backup Designated Router                    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                          Neighbor                             |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                              ...                              |
```

The Hello packet must agree on Network Mask, HelloInterval, RouterDeadInterval, Area ID, and Authentication parameters — any mismatch and the neighbor is rejected. The Options byte signals capabilities (E for external routes, N for NSSA, MC for multicast, DC for demand circuit, O for opaque LSAs).

### Database Description (Type 2)

```
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         Interface MTU         |    Options    |0|0|0|0|0|I|M|MS
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     DD sequence number                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                                                               |
|                  An LSA Header (20 bytes)                     |
|                              ...                              |
```

The Interface MTU field is what kills ExStart on mismatched MTU. Bits I (Init), M (More), MS (Master/Slave) drive the master/slave handshake. Master keeps incrementing the DD sequence number; slave echoes it.

### LSA Header (used inside DBD, LSU, LSAck)

```
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|            LS age             |    Options    |    LS type    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Link State ID                          |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     Advertising Router                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                     LS sequence number                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|         LS checksum           |             length            |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

LS age starts at 0 and is incremented by 1 every second; when it hits MaxAge (3600s) the LSA is considered to be aging out. Sequence number space is 0x80000001 to 0x7FFFFFFF — wraps after 31 bits.

## OSPF Wireshark Capture Analysis

```
# Capture OSPF only, on interface
sudo tcpdump -i eth0 -nn -vvv 'ip proto 89' -w /tmp/ospf.pcap

# Decode in Wireshark
wireshark /tmp/ospf.pcap

# Useful display filters
ospf
ospf.msg == 1                              # Hello only
ospf.msg == 2                              # DBD only
ospf.lsa.lsa_type == 1                     # Router LSAs
ospf.advrouter == 10.0.0.1                 # LSAs from a specific router
ospf.hello.network_mask == 255.255.255.252 # /30 hellos
ospf.iface_mtu == 1500                     # filter by MTU field

# Common pcap signatures
# - ExStart loop: repeated DBD with M=1 / I=1 every second, no LSAcks
# - Auth fail: log line "OSPF Hello packet from 10.10.0.2 has wrong authentication"
# - Bad checksum: "checksum_bad" expert note in Wireshark
```

```
# Sample tcpdump output (broadcast network, mid-flooding)
14:30:12.345678 IP 10.10.0.1 > 224.0.0.5: OSPFv2, Hello, length 48
        Router-ID 10.0.0.1, Area 0.0.0.0
        Options [External]
        Hello Timer 10s, Dead Timer 40s, Mask 255.255.255.0, Priority 1
        Designated Router 10.10.0.1
        Backup Designated Router 10.10.0.2
        Neighbor List: 10.0.0.2

14:30:12.456789 IP 10.10.0.1 > 224.0.0.5: OSPFv2, LS-Update, length 96
        Router-ID 10.0.0.1, Area 0.0.0.0
        Number of LSAs: 1
        LSA-type Router-LSA, LSA-Age 1s, Length 60
                Advertising Router 10.0.0.1, seq 0x80000019
                Number of Links 2
```

## Redistribution into OSPF — Metrics in Detail

OSPF distinguishes external Type 1 (E1) from external Type 2 (E2). E1 metrics add the internal cost to reach the ASBR; E2 metrics stay constant across the AS (the cost is only what the ASBR set). Default is E2 with metric 20.

```
! IOS — explicit E1 with route-map
route-map TAG-EXTERNAL permit 10
 match interface Loopback99
 set metric-type type-1
 set metric 100
 set tag 65000

router ospf 1
 redistribute static subnets route-map TAG-EXTERNAL

! Junos
set policy-options policy-statement TO-OSPF term 1 from protocol static
set policy-options policy-statement TO-OSPF term 1 then external-type 1
set policy-options policy-statement TO-OSPF term 1 then metric 100
set policy-options policy-statement TO-OSPF term 1 then accept
set protocols ospf export TO-OSPF

! FRR
route-map TO-OSPF permit 10
 match interface lo
 set metric-type type-1
 set metric 100
!
router ospf
 redistribute static route-map TO-OSPF
```

| External type | Cost computed by | Best when |
|---------------|-----------------|-----------|
| E1 | ASBR-set metric + path cost to ASBR | Multiple ASBRs, want the closest |
| E2 (default) | Just the ASBR-set metric | Pinning all routers to one egress |

```
R1#show ip route ospf
O E1 192.0.2.0/24 [110/120] via 10.10.0.2, 00:00:14, Gi0/0
O E2 198.51.100.0/24 [110/20] via 10.10.0.2, 00:00:14, Gi0/0
                              ^^^ 20 = ASBR set, never changes
```

### Mutual Redistribution Hazards

Redistributing OSPF ↔ EIGRP, OSPF ↔ BGP, or OSPF ↔ static causes route loops if not filtered. Always tag routes on the way out and reject them on the way back in:

```
route-map OUT-TO-BGP permit 10
 set tag 64512

route-map IN-FROM-BGP deny 10
 match tag 64512
route-map IN-FROM-BGP permit 20

router ospf 1
 redistribute bgp 64500 subnets route-map IN-FROM-BGP
router bgp 64500
 redistribute ospf 1 match internal external 1 external 2 route-map OUT-TO-BGP
```

## Graceful Restart and Non-Stop Forwarding

Graceful Restart (GR, RFC 3623 for OSPFv2 and RFC 5187 for OSPFv3) lets a router rebuild its OSPF process without flushing the FIB. Helper neighbors keep forwarding while the restarter resyncs. Non-Stop Forwarding (NSF) is the platform feature that preserves the FIB during restarts; GR is the protocol-level cooperation.

```
! IOS
router ospf 1
 nsf cisco helper                     ! act as helper for Cisco-style GR
 nsf ietf helper                      ! act as helper for IETF GR
 nsf ietf restart-interval 120

! Junos
set protocols ospf graceful-restart restart-duration 120
set protocols ospf graceful-restart helper-disable     ! disable helper

! FRR
router ospf
 graceful-restart
 graceful-restart helper enable
 graceful-restart restart-time 120
```

Verify a restart:

```
R1#show ip ospf nsf
NSF helper support enabled, hello strict checking is enabled, helper exit-criteria
R1#show ip ospf neighbor detail | include GR
  GR helper-status: NotHelping, GR period: 120 sec
```

## OSPF and ECMP

OSPF installs equal-cost multipath up to a platform limit. Cisco IOS-XE: 32 paths. Junos: 16 (default), up to 64 with `forwarding-options hyper-mode`. FRR: 64 (compile-time MULTIPATH_NUM).

```
! IOS
router ospf 1
 maximum-paths 16

! Junos — uses load-balance per packet by default; enable per-flow
set policy-options policy-statement LB-ECMP from protocol ospf
set policy-options policy-statement LB-ECMP then load-balance per-packet
set routing-options forwarding-table export LB-ECMP

! FRR
router ospf
 maximum-paths 16
```

```
R1#show ip route 10.99.0.0
Routing entry for 10.99.0.0/24
  Known via "ospf 1", distance 110, metric 20, type intra area
  Last update from 10.10.0.6 on Gi0/1, 00:01:34 ago
  Routing Descriptor Blocks:
  * 10.10.0.2, from 10.0.0.5, 00:01:34 ago, via Gi0/0
      Route metric is 20, traffic share count is 1
    10.10.0.6, from 10.0.0.5, 00:01:34 ago, via Gi0/1
      Route metric is 20, traffic share count is 1
```

## Demand Circuits and OSPF over Slow / Dial Links

OSPF Demand Circuit (RFC 1793) lets you suppress the periodic 30-minute LSA refresh on links that bill by usage or have high setup cost. The DC bit in the Options field is set, and LSAs carry the DoNotAge bit.

```
! IOS
interface Dialer1
 ip ospf demand-circuit
```

Don't enable on broadcast LANs — it's intended for ISDN, X.25, dial-on-demand, or cellular failover circuits. With DC, hellos are suppressed once adjacency is Full.

## Multi-Area Design Patterns

```
                 +------------------+
                 |     Area 0       |
                 |  Backbone Core   |
                 +--+------------+--+
                    |            |
              +-----+--+      +--+-----+
              |        |      |        |
       Area 1 (West)  Area 2 (East) Area 3 (DMZ as NSSA)
              |        |      |        |
             ABRs perform Type 3 summarization out of each area
             ASBR in Area 3 redistributes BGP defaults as Type 7
```

Patterns to memorize:

| Pattern | Use when |
|---------|----------|
| Single-area | <50 routers, no scaling concerns, fast labs |
| Two-area (0 + branch stub) | Spoke sites with no need for full LSDB |
| Hub-and-spoke (DR pinned to hub) | DMVPN, MPLS L3VPN PE-CE |
| Multi-area with summarization | Enterprise campus, regional offices |
| NSSA at edge | Sites that redistribute external (BGP, static) |
| Unnumbered fabric (FRR / SONiC) | Spine-leaf data center |
| Single-area with strict filtering | Read-only OSPF over partner peering |

### Hub-and-Spoke over DMVPN

```
! Hub
interface Tunnel0
 ip ospf network point-to-multipoint
 ip ospf priority 10
 ip ospf hello-interval 10
 ip ospf dead-interval 40

! Spoke
interface Tunnel0
 ip ospf network point-to-multipoint
 ip ospf priority 0
```

DR election is suppressed (`network point-to-multipoint`). Each spoke forms a /32 host route to the hub, and inter-spoke traffic transits the hub at L3 unless DMVPN phase 3 with NHRP redirects is configured.

## Capacity and Scaling Limits

| Limit | Value | Notes |
|-------|-------|-------|
| Max LSA size | 65535 B (theoretical) | Practical: keep under MTU minus IP header |
| Max LSAs per area | platform-bounded | Cisco ASR-9000: ~250k; FRR: tested to 100k |
| Max neighbors per interface | 1000 (FRR) | Realistically <50 broadcast, <500 P2P |
| Max areas per router | 100+ (FRR) | Cisco IOS soft limit ~30 |
| Max equal-cost paths | 16 (default Cisco), 64 (FRR), 16 (Junos) | platform-tunable |
| Refresh interval | 1800 s (30 min) | Per LSA, not per protocol |
| MaxAge | 3600 s | LSA flushed when reached |

The most common scaling failure is LSA flooding storm caused by a single flapping link. Use:

```
router ospf 1
 timers throttle lsa 100 1000 5000      ! generation throttle
 timers lsa arrival 100                  ! reject duplicates faster than 100ms
 max-lsa 12000                           ! safety circuit-breaker
```

When `max-lsa` is hit, OSPF generates a warning, then ignore-time, then SPF runs are paused. Saves the box from an OOM in a flooding storm.

## Operational Recipes

### Drain a router for maintenance

```
! IOS — set max-metric so traffic moves away gracefully
router ospf 1
 max-metric router-lsa on-startup 600 wait-for-bgp
 max-metric router-lsa                  ! permanently hold-off (drain)

! Junos
set protocols ospf overload timeout 600
```

When set, the router still answers as a router but sets every link cost to 0xFFFF (LSInfinity), so neighbors see paths through it as worst-possible. Reload, fix, then `no max-metric router-lsa` to re-enter service.

### Force re-flooding after an LSDB drift

```
! Cisco — least invasive: re-originate self LSAs
clear ip ospf process
! prompted: "Reset ALL OSPF processes? [no]: yes"

! Junos
clear ospf database purge

! FRR
clear ip ospf neighbor
```

### Move a network into / out of OSPF without an outage

```
! Add new network to a different area first
router ospf 1
 network 10.50.0.0 0.0.0.255 area 1

! Confirm full adjacency in area 1, then remove from area 0:
 no network 10.50.0.0 0.0.0.255 area 0
```

### Change router-id without resetting neighbors

You can't — router-id is a process-wide identity. Plan a maintenance window:

```
clear ip ospf process
```

### Audit for missing summaries

```
show ip ospf database summary | include "10.1\.|10.2\."
show ip route ospf | include /24
```

If you see the same /24 from two ABRs with different metrics, only the lowest is installed and the duplicate is wasted memory — collapse with a single ABR-level `area X range`.

## Diagnostics: Reading the LSDB Like a Pro

```
R1#show ip ospf database
            OSPF Router with ID (10.0.0.1) (Process ID 1)

                Router Link States (Area 0)
Link ID         ADV Router      Age         Seq#       Checksum  Link count
10.0.0.1        10.0.0.1        412         0x80000019 0xA12C    3
10.0.0.2        10.0.0.2        389         0x80000017 0xC4D1    3

                Net Link States (Area 0)
Link ID         ADV Router      Age         Seq#       Checksum
10.10.0.2       10.0.0.2        389         0x80000003 0x12AB

                Summary Net Link States (Area 0)
Link ID         ADV Router      Age         Seq#       Checksum
10.99.0.0       10.0.0.1        119         0x80000001 0x4567
```

What to look for:

- Same Link ID with two ADV Routers? Two ABRs are advertising the same prefix. Expected for redundant ABRs. Costs should be equal for ECMP, or different to prefer one path.
- Age ≥ 3500? LSA is about to age out — if you didn't trigger a deletion, the router that originated it died.
- Seq# wraps from 0x7FFFFFFF to 0x80000001? Expected; happens once every (sequence-space × refresh-interval) ≈ years.
- Checksum 0x0000? Bad LSA — indicates corruption. Trigger SPF or `clear ip ospf`.

## OSPF in the Lab — Reproducible Scenarios

### Scenario A — three-router triangle in containers

```bash
# Build with FRR + Linux netns (works without any vendor sim)
sudo ip netns add R1
sudo ip netns add R2
sudo ip netns add R3
sudo ip link add R1-R2 type veth peer name R2-R1
sudo ip link add R2-R3 type veth peer name R3-R2
sudo ip link add R3-R1 type veth peer name R1-R3
sudo ip link set R1-R2 netns R1
sudo ip link set R2-R1 netns R2
sudo ip link set R2-R3 netns R2
sudo ip link set R3-R2 netns R3
sudo ip link set R3-R1 netns R3
sudo ip link set R1-R3 netns R1
# Configure addresses, run frr in each ns, watch adjacencies
```

### Scenario B — broadcast LAN with three routers electing DR

```bash
sudo ip link add br0 type bridge
sudo ip link set br0 up
# Attach R1, R2, R3 veths into br0 — all three see each other
# Set R1 priority 100, R2 priority 50, R3 priority 0
# Verify R1 DR, R2 BDR, R3 stays DROther
vtysh -c 'show ip ospf neighbor'
```

### Scenario C — area mismatch reproducer

```
! On R1 set Gi0/0 to area 0
! On R2 set Gi0/0 to area 1
! Logs:
%OSPF-4-MISMATCH_AREA: Hello from 10.10.0.2 area 1 conflicts with our area 0
```

### Scenario D — MTU mismatch reproducer

```bash
# Different ifaces, same subnet
ip netns exec R1 ip link set Gi0/0 mtu 9000
ip netns exec R2 ip link set Gi0/0 mtu 1500
# Run OSPF — adjacency stalls in EXSTART
vtysh -c 'show ip ospf neighbor'
# 10.0.0.2  ExStart  ...
# Fix: ip link set mtu 1500 on R1, or `ip ospf mtu-ignore` on both
```

## OSPF Internals: How Flooding Really Works

Reliable flooding is per-link with explicit acknowledgement. When a router receives an LSA, it:

1. Validates the OSPF and IP headers, checksum, and authentication.
2. Looks up the LSA in the LSDB by (LS Type, Link State ID, Advertising Router).
3. If newer (higher seq, same seq with newer checksum, or older age below MaxAge), accepts it.
4. Floods the LSA out every interface in the same flooding scope except the one it arrived on (split-horizon).
5. Sends an LSAck back on the receiving interface (delayed ack, batched up to MinLSArrival).
6. Updates the LSDB and schedules SPF.

```
        LSU on Gi0/0
           |
           v
  +-----------------+
  | LSDB lookup     |
  +-----------------+
           |
   newer?-----------> install + flood out other ifaces
   same?  -----------> ack only, no flood
   older? -----------> reply with our copy via direct LSU
```

Key pacing parameters:

| Parameter | Default | Tunable IOS | Meaning |
|-----------|---------|-------------|---------|
| MinLSInterval | 5 s | timers throttle lsa | min between same-LSA generations |
| MinLSArrival | 1 s | timers lsa arrival | min between same-LSA acceptances |
| GroupPacing | 240 s | timers pacing flood | batch refresh interval |
| RxmtInterval | 5 s | ip ospf retransmit-interval | LSA retransmit cadence |
| TransmitDelay | 1 s | ip ospf transmit-delay | added to LSA age on send |

## Stub Router (Max-Metric) and ISPF Hooks

Stub-router advertisement (RFC 6987) lets a router withdraw transit usefulness without dropping adjacency. Set every interface metric to LSInfinity. Used for upgrades and graceful insertion.

```
router ospf 1
 max-metric router-lsa on-startup 600 wait-for-bgp summary-lsa external-lsa
```

iSPF (Incremental SPF, RFC 7682) avoids full Dijkstra when a single Type 1/2 changes. Significant on large fabrics:

```
router ospf 1
 ispf

! show
show ip ospf statistics
! "Incremental SPF runs"   23
! "Full SPF runs"           4
```

## OSPF and Multicast / MOSPF (deprecated)

MOSPF (RFC 1584, Type 6 LSAs) was OSPF's attempt at multicast routing. It's effectively dead — vendors removed it after 2010. Modern multicast on OSPF networks uses PIM-SM/SSM with OSPF as the unicast underlay.

## Performance Tips for Large Fabrics

- Move loopback addressing into a dedicated /24 per area; lets ABRs summarize cleanly.
- Set `ip ospf network point-to-point` everywhere there's no DR need (loopbacks, p2p Ethernet, tunnels).
- Disable unused address families — don't run OSPFv3 if you don't run IPv6.
- Pre-stage configs with key-rolls, summarization, and BFD on day zero — adding them later requires touch-ups across every router.
- Capture baseline `show ip ospf statistics` after stabilization so you can compare during incidents.
- For fabrics with >100 routers in one area, evaluate IS-IS instead — same algorithm, smaller LSDB on many platforms.

## See Also

- `networking/bgp` — exterior routing and inter-AS policy
- `networking/is-is` — alternative IGP for SP and DC fabrics
- `networking/eigrp` — Cisco DUAL-based IGP
- `networking/mpls` — MPLS forwarding plane that consumes OSPF-TE
- `networking/bfd` — sub-second failure detection paired with OSPF
- `networking/subnetting` — IP planning for OSPF area design
- `networking/ip` — IP fundamentals
- `networking/ecmp` — equal-cost multipath, OSPF's load-sharing knob

## References

- [RFC 2328 — OSPF Version 2](https://www.rfc-editor.org/rfc/rfc2328)
- [RFC 5340 — OSPF for IPv6](https://www.rfc-editor.org/rfc/rfc5340)
- [RFC 5709 — OSPFv2 HMAC-SHA Cryptographic Authentication](https://www.rfc-editor.org/rfc/rfc5709)
- [RFC 5838 — Support of Address Families in OSPFv3](https://www.rfc-editor.org/rfc/rfc5838)
- [RFC 4552 — Authentication/Confidentiality for OSPFv3](https://www.rfc-editor.org/rfc/rfc4552)
- [RFC 7166 — OSPFv3 Authentication Trailer](https://www.rfc-editor.org/rfc/rfc7166)
- [RFC 5250 — OSPF Opaque LSA](https://www.rfc-editor.org/rfc/rfc5250)
- [RFC 3630 — Traffic Engineering Extensions to OSPFv2](https://www.rfc-editor.org/rfc/rfc3630)
- [RFC 5187 — OSPFv3 Graceful Restart](https://www.rfc-editor.org/rfc/rfc5187)
- [RFC 5243 — OSPF Database Exchange Summary List Optimization](https://www.rfc-editor.org/rfc/rfc5243)
- [RFC 5340 — OSPFv3 (canonical)](https://www.rfc-editor.org/rfc/rfc5340)
- [RFC 6549 — OSPFv2 Multi-Instance Extensions](https://www.rfc-editor.org/rfc/rfc6549)
- [RFC 7503 — OSPFv3 Autoconfiguration](https://www.rfc-editor.org/rfc/rfc7503)
- [RFC 8665 — OSPF SR Extensions](https://www.rfc-editor.org/rfc/rfc8665)
- [RFC 9350 — IGP Flex-Algorithm](https://www.rfc-editor.org/rfc/rfc9350)
- [RFC 5880 — Bidirectional Forwarding Detection](https://www.rfc-editor.org/rfc/rfc5880)
- [FRRouting OSPF Documentation](https://docs.frrouting.org/en/latest/ospfd.html)
- [FRRouting OSPFv3 Documentation](https://docs.frrouting.org/en/latest/ospf6d.html)
- [BIRD Internet Routing Daemon — OSPF](https://bird.network.cz/?get_doc&v=20&f=bird-6.html)
- [Cisco OSPF Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_ospf/configuration/xe-16/iro-xe-16-book.html)
- [Cisco OSPFv3 Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_ospfv3/configuration/xe-16/ip6-route-ospfv3-xe-16-book.html)
- [Juniper OSPF Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/ospf/topics/topic-map/ospf-overview.html)
- [Arista EOS OSPF Configuration Guide](https://www.arista.com/en/um-eos/eos-open-shortest-path-first-version-2)
- [NX-OS OSPF Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/unicast/configuration/guide/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-93x.html)
- man frr-ospfd, man frr-ospf6d, man bird-ospf
