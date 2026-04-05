# LISP (Locator/ID Separation Protocol)

Overlay routing protocol that decouples host identity (EID) from network location (RLOC), enabling seamless mobility, multihoming, and traffic engineering across IP networks.

## Architecture

### EID/RLOC Split

```
EID (Endpoint Identifier):  Host-facing address — stays constant regardless of location
RLOC (Routing Locator):     Infrastructure address — changes as the host moves

+----------+                                      +----------+
| Host A   |                                      | Host B   |
| EID:     |                                      | EID:     |
| 10.1.1.1 |                                      | 10.2.2.2 |
+----+-----+                                      +----+-----+
     |                                                  |
+----+-----+     LISP Encapsulated Tunnel          +----+-----+
| ITR      |===================================== | ETR      |
| RLOC:    |  Outer: RLOC-src -> RLOC-dst         | RLOC:    |
| 1.1.1.1  |  Inner: EID-src  -> EID-dst          | 2.2.2.2  |
+----------+                                      +----------+
```

- **EID space** is not routed in the core; only edge devices (xTRs) hold EID routes
- **RLOC space** is routed normally in the underlay (IGP/BGP)
- The mapping system binds EIDs to RLOCs dynamically

### LISP Data Plane Encapsulation

```
+------+--------+-----+------+--------+---------+
| Outer| Outer  | UDP | LISP | Inner  | Inner   |
| L2   | IP Hdr | 4341| Hdr  | IP Hdr | Payload |
+------+--------+-----+------+--------+---------+
       RLOC-src   ^         EID-src
       RLOC-dst   |         EID-dst
                   |
              LISP data port
```

- UDP destination port 4341 (data), 4342 (control)
- 36-byte overhead for IPv4-in-IPv4 (8 byte LISP header + 8 byte UDP + 20 byte outer IP)
- Instance ID field (24-bit) provides VRF-like segmentation (up to 16 million instances)

## LISP Roles

| Role | Full Name | Function |
|:-----|:----------|:---------|
| ITR | Ingress Tunnel Router | Encapsulates packets from EID source toward RLOC destination |
| ETR | Egress Tunnel Router | Decapsulates LISP packets and delivers to local EID hosts |
| xTR | Combined ITR/ETR | Most common deployment; both roles on one device |
| MR | Map-Resolver | Receives map-requests from ITRs, forwards to map-server or ETR |
| MS | Map-Server | Stores EID-to-RLOC registrations from ETRs, answers map-requests |
| PITR | Proxy ITR | Attracts non-LISP traffic for LISP EIDs via BGP, encapsulates toward ETR |
| PETR | Proxy ETR | Decapsulates LISP traffic on behalf of sites that cannot run LISP |
| MR/MS | Combined | Typically co-located for simplicity |

### Control Plane Message Flow

```
ETR ---[Map-Register]---> Map-Server (MS)
MS  ---[Map-Notify]-----> ETR  (acknowledgement)

ITR ---[Map-Request]----> Map-Resolver (MR)
MR  ---[Map-Request]----> ETR  (forwarded or answered from MS database)
ETR ---[Map-Reply]------> ITR  (EID-to-RLOC mapping returned)
```

## Map-Server / Map-Resolver Configuration (IOS-XE)

### Map-Server / Map-Resolver

```
! Enable LISP on a device acting as MS/MR
router lisp
 locator-set RLOC-SET
  IPv4-interface Loopback0 priority 1 weight 50
  exit-locator-set
 !
 service ipv4
  map-server
  map-resolver
  exit-service-ipv4
 !
 site SITE-A
  authentication-key SITE-A-KEY
  eid-record instance-id 0 10.1.0.0/16
  exit-site
 !
 site SITE-B
  authentication-key SITE-B-KEY
  eid-record instance-id 0 10.2.0.0/16
  exit-site
```

### xTR (ITR + ETR) Configuration

```
router lisp
 locator-set RLOC-SET
  IPv4-interface Loopback0 priority 1 weight 50
  exit-locator-set
 !
 service ipv4
  itr map-resolver 192.168.100.1
  itr
  etr map-server 192.168.100.1 key SITE-A-KEY
  etr
  exit-service-ipv4
 !
 instance-id 0
  service ipv4
   eid-table default
   database-mapping 10.1.0.0/16 locator-set RLOC-SET
   exit-service-ipv4
  exit-instance-id
```

### IOS-XR LISP Configuration

```
router lisp
 locator-set RLOC-SET
  ipv4-interface Loopback0 priority 1 weight 50
 !
 service ipv4
  itr map-resolver 192.168.100.1
  etr map-server 192.168.100.1 key SITE-KEY
  itr
  etr
 !
 eid-table default instance-id 0
  address-family ipv4 unicast
   database-mapping 10.1.0.0/16 locator-set RLOC-SET
```

## LISP Site Registration and Verification

```
! Verify site registrations on the MS
show lisp site
show lisp site detail
show lisp site name SITE-A

! Verify map-cache on the ITR
show lisp instance-id 0 ipv4 map-cache
show lisp instance-id 0 ipv4 map-cache detail

! Verify database on the ETR
show lisp instance-id 0 ipv4 database

! Verify RLOC reachability
show lisp instance-id 0 ipv4 server rloc members

! Debug map-request/map-reply flow
debug lisp control-plane all
```

### Map-Cache Output Interpretation

```
LISP IPv4 Mapping Cache, 3 entries

10.2.0.0/16, uptime: 00:05:23, expires: 23:54:37, via map-reply, complete
  Locator      Pri/Wgt  Source     State
  2.2.2.2       1/50    map-reply  up

0.0.0.0/0, uptime: 01:00:00, expires: never, via static-send-map-request
  Negative cache entry, action: send-map-request

10.1.0.0/16, uptime: 01:00:00, expires: never, via dynamic-EID
  Locator      Pri/Wgt  Source     State
  1.1.1.1       1/50    local      site-self, reachable
```

## LISP Mobility

### Host Mobility (IP Preservation)

```
1. Host moves from Site-A (ETR-A, RLOC 1.1.1.1) to Site-B (ETR-B, RLOC 2.2.2.2)
2. Host keeps its EID address (10.1.1.100)
3. ETR-B detects the host and sends Map-Register to MS with new RLOC binding
4. MS updates the mapping: 10.1.1.100/32 -> RLOC 2.2.2.2
5. MS sends SMR (Solicit Map-Request) to ITRs with cached old mapping
6. ITRs refresh map-cache with new RLOC
7. Traffic now flows to new location — no DNS change, no IP change
```

### Dynamic EID Configuration

```
router lisp
 instance-id 0
  service ipv4
   eid-table default
   ! Detect hosts dynamically via ARP/ND
   dynamic-eid DETECT-HOSTS
    database-mapping 10.1.0.0/16 locator-set RLOC-SET
    exit-dynamic-eid
   exit-service-ipv4
  exit-instance-id
!
interface GigabitEthernet1
 lisp mobility DETECT-HOSTS
```

### VM Mobility

```
! LISP enables VM mobility across L3 boundaries without stretching VLANs

Hypervisor-A (Site-A)              Hypervisor-B (Site-B)
+--------+                        +--------+
| VM     |  -- vMotion/live -->   | VM     |
| EID:   |     migration          | EID:   |
| 10.1.1.5                       | 10.1.1.5
+--------+                        +--------+
     |                                 |
  ETR-A (RLOC 1.1.1.1)           ETR-B (RLOC 2.2.2.2)

1. VM migrates, ETR-B detects ARP from VM
2. ETR-B registers EID 10.1.1.5/32 with MS
3. MS sends Map-Notify to ETR-A to withdraw old mapping
4. ITRs get SMR, refresh cache
5. Convergence: sub-second (map-register + SMR propagation)
```

## LISP Pub-Sub (RFC 9437)

```
! Pub-Sub replaces pull-based map-requests with push-based notifications
! ITRs subscribe to EID prefixes and receive updates proactively

router lisp
 service ipv4
  itr map-resolver 192.168.100.1
  itr
  ! Enable pub-sub on the ITR
  map-request itr-rlocs RLOC-SET
  exit-service-ipv4

! On the MS: pub-sub is enabled by default in modern IOS-XE (17.x+)
! Subscribers receive immediate Map-Notify on any mapping change

! Verify subscriptions
show lisp instance-id 0 ipv4 server subscription
show lisp instance-id 0 ipv4 publisher
```

Benefits over pull model:
- No SMR/map-request round trip delay on mobility events
- Mapping convergence drops from seconds to milliseconds
- Reduced control plane chatter during mass mobility events

## SD-Access Integration

```
LISP in SD-Access (Cisco SDA):
+-------------------------------------------------------------------+
| Fabric Control Plane = LISP Map-Server/Map-Resolver               |
| Fabric Data Plane    = VXLAN encapsulation (not LISP data plane)  |
| Fabric Border        = PITR/PETR toward external networks         |
| Fabric Edge          = xTR (ITR + ETR)                            |
+-------------------------------------------------------------------+

! In SD-Access, LISP provides the control plane only:
! - Host tracking via Map-Register/Map-Notify
! - EID-to-RLOC resolution via Map-Request/Map-Reply
! - Macro/micro segmentation via Instance-ID (VN) and SGT
!
! Data plane uses VXLAN-GPO (Group Policy Option) not LISP encap

Instance-ID mapping in SD-Access:
  Instance-ID = Virtual Network (VN) identifier
  L3 VN -> LISP instance-id -> VRF
  L2 VN -> LISP instance-id -> VLAN/bridge-domain
```

### SD-Access LISP Verification

```
show lisp instance-id * ipv4 server summary
show lisp instance-id * ethernet server summary
show lisp site
show lisp session
```

## Multihoming and Traffic Engineering

### Priority/Weight Load Balancing

```
! Priority: lower = preferred (active/standby model)
! Weight: relative proportion of traffic (load balancing within same priority)

router lisp
 locator-set MULTIHOME
  IPv4-interface Loopback0 priority 1 weight 50    ! primary, 50% share
  IPv4-interface Loopback1 priority 1 weight 50    ! primary, 50% share
  IPv4-interface Loopback2 priority 2 weight 100   ! backup (higher priority number)
  exit-locator-set
```

| Priority | Weight | Behavior |
|:---------|:-------|:---------|
| 1, 1 | 50, 50 | Active-active, equal load sharing |
| 1, 1 | 75, 25 | Active-active, 75/25 split |
| 1, 2 | 100, 100 | Active-standby failover |
| 1, 1, 2 | 50, 50, 100 | Two active, one backup |

## Convergence Protocol

```
LISP Convergence Timeline:

Event: Link/site failure detected
  |
  +-- 0ms:     Failure detection (BFD/IGP)
  +-- ~50ms:   ETR withdraws registration (Map-Register with TTL=0)
  +-- ~100ms:  MS processes withdrawal
  +-- ~150ms:  MS sends SMR to subscribing ITRs (or Map-Notify via pub-sub)
  +-- ~200ms:  ITRs send Map-Request for updated mapping
  +-- ~250ms:  ITRs receive Map-Reply with new RLOC(s)
  +-- ~300ms:  Traffic converges to new path

Total convergence: 200-500ms typical (with pub-sub: <200ms)
```

### Map-Cache Timers

```
! Default TTL for map-cache entries
show lisp instance-id 0 ipv4 map-cache

! Adjust registration interval (default 60s)
router lisp
 service ipv4
  etr map-server 192.168.100.1 key KEY
  etr registration-interval 30       ! more frequent keepalives
  exit-service-ipv4

! Map-cache TTL is set by the ETR in Map-Reply (default 1440 minutes / 24 hours)
! Negative cache TTL (for non-existent EIDs): 15 minutes default
```

## Tips

- Always deploy at least two MS/MR nodes for redundancy; xTRs register with both.
- Use Instance-ID to segment traffic across tenants; it maps directly to VRF on the xTR.
- Set different priorities on RLOCs for active/standby multihoming rather than relying on IGP metrics.
- Deploy PITRs at the network border to attract traffic from non-LISP domains toward LISP EIDs.
- Monitor map-cache size on ITRs; large-scale deployments can accumulate thousands of entries.
- Use LISP pub-sub (IOS-XE 17.x+) instead of the legacy SMR pull model for sub-second mobility convergence.
- Ensure PMTUD works across the overlay; the LISP encapsulation adds 36-56 bytes of overhead.
- In SD-Access, LISP is control-plane only; do not confuse it with the VXLAN data plane.
- Keep authentication keys per-site on the MS; shared keys across sites weaken security.
- Use `show lisp site detail` on the MS to verify all ETRs are registering and their timestamps are current.
- For VM mobility, configure dynamic-EID on the xTR interfaces facing hypervisors so /32 host routes register automatically.
- Test failover by shutting an RLOC interface and verifying map-cache updates on remote ITRs within the expected convergence window.

## See Also

- bgp, ospf, is-is, vxlan, cisco-dna-center, segment-routing, mpls, gre

## References

- [RFC 6830 — The Locator/ID Separation Protocol (LISP)](https://www.rfc-editor.org/rfc/rfc6830)
- [RFC 6831 — The LISP Alt Logical Topology](https://www.rfc-editor.org/rfc/rfc6831)
- [RFC 6832 — Interworking between LISP and Non-LISP Sites](https://www.rfc-editor.org/rfc/rfc6832)
- [RFC 6833 — LISP Map-Server Interface](https://www.rfc-editor.org/rfc/rfc6833)
- [RFC 6834 — LISP Map-Versioning](https://www.rfc-editor.org/rfc/rfc6834)
- [RFC 9300 — The Locator/ID Separation Protocol (LISP) — Updated](https://www.rfc-editor.org/rfc/rfc9300)
- [RFC 9301 — LISP Control Plane](https://www.rfc-editor.org/rfc/rfc9301)
- [RFC 9437 — LISP Publish/Subscribe](https://www.rfc-editor.org/rfc/rfc9437)
- [RFC 9303 — LISP-SEC](https://www.rfc-editor.org/rfc/rfc9303)
- [Cisco LISP Configuration Guide (IOS-XE)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_lisp/configuration/xe-17/irl-xe-17-book.html)
- [Cisco SD-Access Design Guide](https://www.cisco.com/c/en/us/td/docs/solutions/CVD/Campus/cisco-sda-design-guide.html)
