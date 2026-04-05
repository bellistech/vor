# FHRP — First Hop Redundancy Protocol Theory and Comparison

> First Hop Redundancy Protocols (FHRP) solve a fundamental problem in IP networking: hosts
> typically configure a single default gateway, and if that gateway fails, all traffic from the
> subnet is blackholed. FHRP protocols — HSRP, VRRP, and GLBP — create a virtual gateway
> shared between two or more routers, providing seamless failover (and in GLBP's case, native
> load balancing) without any host-side changes.

## 1. The Problem FHRP Solves

IP hosts learn their default gateway in one of three ways: static configuration, DHCP, or
router advertisements (IPv6 SLAAC). In all cases, the host sends traffic destined for remote
networks to a single next-hop IP address. If the router behind that IP goes offline, the host
has no mechanism to discover a new gateway — it continues sending packets into a black hole
until manual intervention or a timeout-based protocol (like ICMP Router Discovery) kicks in.

FHRP protocols address this by:

1. Assigning a **virtual IP address (VIP)** shared by two or more routers.
2. Generating a **virtual MAC address** so that hosts' ARP caches map the VIP to a stable MAC.
3. Running an **election protocol** to determine which router actively forwards traffic.
4. Detecting failures via **hello/advertisement timers** and promoting a standby/backup router.
5. Optionally tracking **upstream interface or route availability** to trigger proactive failover.

The host never changes its gateway configuration. The routers negotiate among themselves.

## 2. HSRP — Hot Standby Router Protocol

### 2.1 Overview

HSRP is Cisco's proprietary FHRP, first introduced in the early 1990s. It is the most widely
deployed FHRP in Cisco-only environments and has mature integration with NX-OS features like
vPC.

Two versions exist:

| Aspect              | HSRP v1                  | HSRP v2                  |
|---------------------|--------------------------|--------------------------|
| Group range         | 0-255                    | 0-4095                   |
| Multicast address   | 224.0.0.2 (all routers)  | 224.0.0.102 (dedicated)  |
| Virtual MAC format  | 0000.0c07.acXX           | 0000.0c9f.fXXX           |
| Timer granularity   | Seconds only             | Milliseconds supported   |
| IPv6 support        | No                       | Yes                      |
| Transport           | UDP 1985                 | UDP 1985                 |

HSRP v1 shares the 224.0.0.2 multicast address with other protocols, which can cause issues.
HSRP v2 uses a dedicated multicast group and should always be preferred.

### 2.2 HSRP State Machine

HSRP routers transition through six states:

```
                              +-----------+
                              |   Init    |
                              | (Disabled)|
                              +-----+-----+
                                    |
                          Interface comes up
                                    |
                                    v
                              +-----------+
                              |   Learn   |
                              | (No VIP)  |
                              +-----+-----+
                                    |
                         Learns VIP from Active
                          or has VIP configured
                                    |
                                    v
                              +-----------+
                              |  Listen   |
                              | (Passive) |
                              +-----+-----+
                                    |
                          Active + Standby timers
                          expire (or higher priority)
                                    |
                                    v
                              +-----------+
                              |   Speak   |
                              |(Contending)|
                              +-----+-----+
                                    |
                     +-------- Election --------+
                     |                          |
                     v                          v
              +-----------+              +-----------+
              |  Standby  |              |  Active   |
              |  (Backup) |              |(Forwarding)|
              +-----+-----+              +-----+-----+
                    |                          |
                    |    Active fails/resigns  |
                    +------------------------->|
                    |                          |
                    |   Preempt (higher pri)   |
                    |<-------------------------+
```

**Init:** Interface is down or HSRP is not yet configured. No participation.

**Learn:** The router does not know the virtual IP address and is waiting to hear from the
Active router. This state is only reached when the VIP is not statically configured.

**Listen:** The router knows the VIP but is neither Active nor Standby. It monitors hellos
from both Active and Standby routers. If both fail, it moves to Speak.

**Speak:** The router sends hello messages and actively participates in the election for
Active and Standby roles. This is a transient state during initial negotiation.

**Standby:** The router is the next in line to become Active. It continues to monitor the
Active router's hellos. If the hold timer expires without receiving a hello from Active,
the Standby router transitions to Active.

**Active:** The router owns the virtual IP and virtual MAC. It forwards all traffic
addressed to the VIP, responds to ARP requests for the VIP, and sends periodic hellos.

### 2.3 Election Process

The Active router is elected based on:

1. **Highest priority** wins (default 100, range 0-255).
2. **Highest IP address** breaks ties.

The Standby router is the second-highest priority router. All other routers remain in Listen.

**Preemption** is disabled by default. Without preemption, if a higher-priority router comes
online after the Active is already established, it remains in Listen/Standby — it does NOT
take over. Enable preemption explicitly:

```
standby 1 preempt
standby 1 preempt delay minimum 30 reload 60
```

The `delay minimum` prevents flapping by waiting N seconds after the router detects it should
preempt. The `reload` delay waits after a router reload before preempting, allowing routing
protocols to converge first.

### 2.4 Timers

| Timer   | Default  | Purpose                                           |
|---------|----------|---------------------------------------------------|
| Hello   | 3 sec    | Interval between hello messages from Active/Standby|
| Hold    | 10 sec   | Time before declaring Active/Standby as failed     |

The hold timer must be at least 3x the hello timer. Both Active and Standby routers send
hellos. Routers in Listen state do NOT send hellos.

For sub-second failover:

```
standby 1 timers msec 200 msec 750
```

This achieves ~750ms failover detection. Be cautious with aggressive timers on congested
links or CPU-constrained platforms — false positives cause unnecessary transitions.

### 2.5 Authentication

HSRP supports two authentication methods:

**Plaintext** (insecure, visible in packet captures):
```
standby 1 authentication text MyPassword
```

**MD5** (recommended for production):
```
! Key-string method
standby 1 authentication md5 key-string S3cureK3y!

! Key-chain method (allows key rotation)
key chain HSRP-KEYS
 key 1
  key-string FirstKey123
  accept-lifetime 00:00:00 Jan 1 2026 00:00:00 Jul 1 2026
  send-lifetime 00:00:00 Jan 1 2026 00:00:00 Jun 1 2026
 key 2
  key-string SecondKey456
  accept-lifetime 00:00:00 Jun 1 2026 infinite
  send-lifetime 00:00:00 Jun 15 2026 infinite

standby 1 authentication md5 key-chain HSRP-KEYS
```

Mismatched authentication causes the router to ignore hellos from the peer, potentially
resulting in a dual-Active condition.

### 2.6 Interface Tracking

Static priority alone cannot react to upstream failures. HSRP integrates with Cisco's object
tracking to dynamically decrement priority when tracked objects go down:

```
track 1 interface GigabitEthernet0/1 line-protocol
track 2 ip route 0.0.0.0/0 reachability
track 3 ip sla 1 reachability

standby 1 track 1 decrement 20
standby 1 track 2 decrement 15
standby 1 track 3 decrement 30
```

When the tracked object goes down, the HSRP priority is decremented by the specified value.
If the resulting priority is lower than the Standby router's priority (and preemption is
enabled on the Standby), a failover occurs.

**Design rule:** Set the decrement value so that:

```
Active_priority - decrement < Standby_priority
```

Example: Active priority 110, Standby priority 100, decrement must be > 10.

## 3. VRRP — Virtual Router Redundancy Protocol

### 3.1 Overview

VRRP is the IETF standard alternative to HSRP. VRRPv2 (RFC 3768) supports IPv4 only. VRRPv3
(RFC 5798) supports both IPv4 and IPv6 in a unified framework.

Key differences from HSRP:

- **Open standard** — works across vendors (Cisco, Juniper, Arista, Nokia, etc.).
- **Preemption is enabled by default** — a higher-priority router always takes over.
- **Owner mode** — if a router's real IP equals the VIP, it becomes Master with priority 255.
- **Uses IP protocol 112** — not UDP. This affects firewall and ACL rules.
- **Only Master sends advertisements** — Backup routers are silent unless they detect Master failure.
- **Terminology:** Master (not Active), Backup (not Standby).

### 3.2 VRRP State Machine

VRRP has a simpler state machine than HSRP — only three states:

```
                    +------------+
                    | Initialize |
                    +------+-----+
                           |
              Interface comes up, VRRP enabled
                           |
            +--------------+--------------+
            |                             |
  Priority == 255                  Priority < 255
  (IP owner)                       (Non-owner)
            |                             |
            v                             v
      +-----------+                +-----------+
      |  Master   |<------------->|  Backup   |
      +-----------+  failover /   +-----------+
      | Forwards  |  preemption   | Listens   |
      | traffic   |               | for adverts|
      | Sends     |               | Master_Down|
      | adverts   |               | timer      |
      +-----------+               +-----------+
```

**Initialize:** VRRP is starting. If the router owns the VIP (real IP == VIP), it goes
directly to Master. Otherwise, it goes to Backup.

**Backup:** The router monitors for advertisements from the Master. It does NOT send any
advertisements. If the Master_Down_Interval expires (3x advertisement interval + skew time),
the Backup transitions to Master.

**Master:** The router owns the VIP, responds to ARP, and forwards traffic. It sends periodic
advertisements at the configured interval. If it receives an advertisement with a higher
priority, and preemption is configured (on by default), it transitions back to Backup.

### 3.3 Master_Down_Interval Calculation

The skew time ensures that higher-priority Backup routers detect Master failure faster:

```
Skew_Time = ((256 - Priority) / 256) * Master_Advert_Interval

Master_Down_Interval = (3 * Master_Advert_Interval) + Skew_Time
```

Example with default 1-second advertisements and priority 100:

```
Skew_Time = ((256 - 100) / 256) * 1 = 0.609 seconds
Master_Down_Interval = 3 + 0.609 = 3.609 seconds
```

A Backup with priority 150:

```
Skew_Time = ((256 - 150) / 256) * 1 = 0.414 seconds
Master_Down_Interval = 3 + 0.414 = 3.414 seconds
```

The higher-priority Backup detects failure faster and wins the election.

### 3.4 Owner Mode

If a router's real interface IP matches the configured VIP, it is the **IP Address Owner**.
The owner:

- Automatically gets priority 255 (highest possible, not configurable).
- Transitions directly from Initialize to Master, skipping Backup.
- Cannot be preempted by any other router (no one can have priority > 255).
- Responds to ICMP pings to the VIP even when in Backup state (unlike non-owners).

This is a unique VRRP feature not present in HSRP or GLBP. It is useful when migrating
from a standalone router to a redundant pair — the original router keeps its IP and
automatically becomes Master.

### 3.5 VRRPv3 vs VRRPv2

| Feature                | VRRPv2 (RFC 3768)        | VRRPv3 (RFC 5798)        |
|------------------------|--------------------------|--------------------------|
| IP version             | IPv4 only                | IPv4 and IPv6            |
| Timer granularity      | Seconds                  | Centiseconds (10ms)      |
| Authentication         | Plaintext (optional)     | Removed (by design)      |
| Sub-second failover    | No                       | Yes                      |
| Configuration (IOS-XE) | `vrrp N ip X.X.X.X`     | `vrrp N address-family`  |

VRRPv3 intentionally removed authentication. The RFC authors argued that plaintext auth
provides no real security, and MD5/SHA would add complexity without meaningful benefit since
VRRP operates on directly-connected links. If security is needed, use IPsec or MACsec at
the link layer.

### 3.6 Preemption Control

Preemption is enabled by default in VRRP. To prevent flapping:

```
! VRRPv3 (IOS-XE)
vrrp 1 address-family ipv4
 preempt delay minimum 30       ! Wait 30 seconds before preempting

! Disable preemption entirely (unusual but valid)
vrrp 1 address-family ipv4
 no preempt
```

The preempt delay is critical in environments where the Master router reboots and needs time
for OSPF/BGP/EIGRP to converge before it should start forwarding traffic.

## 4. GLBP — Gateway Load Balancing Protocol

### 4.1 Overview

GLBP is Cisco's proprietary protocol that solves the load balancing limitation of HSRP and
VRRP. With HSRP/VRRP, only one router forwards traffic at a time (unless you configure
multiple groups). GLBP allows all routers in the group to forward traffic simultaneously
using a single virtual IP address.

### 4.2 Architecture: AVG and AVF

GLBP introduces two roles:

**AVG (Active Virtual Gateway):**
- One router is elected AVG per GLBP group (highest priority, then highest IP).
- The AVG answers all ARP requests for the virtual IP.
- Instead of always returning the same virtual MAC, the AVG distributes different virtual
  MACs to different ARP requesters, pointing them to different routers (AVFs).
- The AVG itself can also be an AVF simultaneously.

**AVF (Active Virtual Forwarder):**
- Each router in the group is assigned a virtual MAC and becomes an AVF.
- Up to 4 AVFs per GLBP group.
- Each AVF forwards traffic from hosts that learned its virtual MAC via ARP.
- AVF virtual MACs follow the format: `0007.b400.XXYY` where XX = group, YY = AVF number.

```
                        ARP Request: "Who has 10.0.1.1?"
                                    |
                                    v
                            +-------+-------+
                            |     AVG       |
                            | (10.0.1.2)    |
                            | Assigns VMACs |
                            +-------+-------+
                           /        |        \
                          /         |         \
                ARP Reply:    ARP Reply:    ARP Reply:
              MAC = AVF1     MAC = AVF2    MAC = AVF3
                 |               |              |
                 v               v              v
            +--------+     +--------+     +--------+
            | AVF 1  |     | AVF 2  |     | AVF 3  |
            |R1: .2  |     |R2: .3  |     |R3: .4  |
            |VMAC:01 |     |VMAC:02 |     |VMAC:03 |
            +--------+     +--------+     +--------+
                 ^               ^              ^
                 |               |              |
             Host A          Host B          Host C
          (got AVF1 MAC)  (got AVF2 MAC)  (got AVF3 MAC)
```

All three hosts use the same VIP (10.0.1.1) as their default gateway, but each was given a
different virtual MAC address, so their traffic is distributed across all three routers.

### 4.3 Load Balancing Methods

**Round-Robin (default):**
Each successive ARP request gets the next AVF's MAC. Host A gets AVF1, Host B gets AVF2,
Host C gets AVF3, Host D gets AVF1 again.

```
glbp 1 load-balancing round-robin
```

**Weighted:**
AVFs are assigned weights, and the AVG distributes ARP replies proportionally. A router
with weight 200 gets twice the traffic of a router with weight 100.

```
glbp 1 load-balancing weighted
glbp 1 weighting 200 lower 50 upper 100
```

The `lower` threshold defines when the AVF stops forwarding (weight drops below 50 due to
tracked objects going down). The `upper` threshold defines when the AVF resumes forwarding
(weight recovers above 100).

**Host-Dependent:**
A hash of the client's source MAC determines which AVF it is assigned to. The same client
always gets the same AVF, providing session persistence.

```
glbp 1 load-balancing host-dependent
```

### 4.4 Redirect and Timeout Timers

When an AVF fails, the AVG must handle the transition:

| Timer     | Default  | Purpose                                                |
|-----------|----------|--------------------------------------------------------|
| Redirect  | 600s     | How long AVG continues directing new ARP replies away  |
|           | (10 min) | from the failed AVF (using remaining AVFs)             |
| Timeout   | 14400s   | How long before the AVG removes the failed AVF entry   |
|           | (4 hrs)  | entirely from the GLBP table                           |

During the redirect period: New clients get directed to healthy AVFs. Old clients that
already have the failed AVF's MAC cached will have their traffic blackholed until their
ARP cache expires (or the AVG proxy-forwards on behalf of the failed AVF).

After redirect expires but before timeout: The AVG no longer redirects, but the old AVF
entry remains in the table in case it comes back.

After timeout: The AVF entry is completely removed.

### 4.5 AVF States

Each AVF goes through its own state machine:

```
+----------+     +---------+     +--------+     +--------+
| Disabled |---->| Initial |---->| Listen |---->| Active |
+----------+     +---------+     +--------+     +--------+
                                     ^               |
                                     |   AVF fails   |
                                     +---------------+
```

Additionally, there are two special states related to failover:

- **Active:** The AVF is forwarding traffic for its assigned virtual MAC.
- **Listen:** The AVF is monitoring the Active AVF and ready to take over.

## 5. FHRP with vPC (Cisco Nexus)

### 5.1 The Problem

Virtual Port-Channel (vPC) presents two physical Nexus switches as a single logical switch
to downstream devices. When combined with FHRP, special considerations arise:

- Both vPC peers have SVIs on the same VLAN.
- Both run HSRP (or VRRP) on those SVIs.
- Traffic from hosts can arrive at either peer via the port-channel.
- If traffic arrives at the Standby peer, it must be forwarded to the Active peer via the
  peer-link, adding latency and consuming peer-link bandwidth.

### 5.2 Peer-Gateway

The `peer-gateway` feature solves this by allowing each vPC peer to act as the gateway for
packets destined to the OTHER peer's router MAC:

```
vpc domain 10
 peer-gateway
```

Without peer-gateway:
```
Host --> vPC Standby Peer --> (peer-link) --> vPC Active Peer --> Upstream
```

With peer-gateway:
```
Host --> vPC Standby Peer --> Upstream  (Standby forwards directly)
```

The Standby peer recognizes packets destined to the Active peer's HSRP MAC and routes them
locally instead of hairpinning through the peer-link. This is critical for performance in
data center environments.

### 5.3 HSRP Fabric Peering

In NX-OS 7.0(3) and later, HSRP fabric peering enhances HSRP behavior in vPC environments:

```
interface Vlan100
 hsrp version 2
 hsrp 100
  fabricpath-peering
```

Benefits:
- HSRP hellos are exchanged over the vPC peer-link fabric channel instead of the SVI.
- Prevents HSRP state changes during SVI flaps that don't affect the peer-link.
- More stable HSRP operation in large-scale vPC deployments.

### 5.4 Best Practices for FHRP + vPC

1. **Always enable peer-gateway** — prevents unnecessary peer-link traffic.
2. **Use HSRP v2** — better vPC integration than VRRP on NX-OS.
3. **Set consistent priorities** — Primary vPC peer should be HSRP Active.
4. **Avoid GLBP with vPC** — GLBP's multi-AVF model conflicts with vPC's dual-peer design.
5. **Track the peer-link** — if the peer-link fails and the secondary loses its upstream,
   it should relinquish HSRP Active.
6. **Match HSRP Active with STP root** — align L2 and L3 forwarding paths.

```
! Recommended vPC + HSRP design
! Primary peer (lower vPC role priority):
vpc domain 10
 role priority 1000
 peer-gateway

interface Vlan100
 ip address 10.0.100.2/24
 hsrp version 2
 hsrp 100
  ip 10.0.100.1
  priority 110
  preempt

! Secondary peer:
vpc domain 10
 role priority 2000
 peer-gateway

interface Vlan100
 ip address 10.0.100.3/24
 hsrp version 2
 hsrp 100
  ip 10.0.100.1
  priority 100
  ! No preempt — let primary take over naturally
```

## 6. Design Considerations

### 6.1 When to Use Each Protocol

**Use HSRP when:**
- Cisco-only environment (no interop needed).
- vPC / Nexus data center fabric — best integration.
- Need MD5 authentication on the FHRP.
- Well-understood by operations team (most Cisco engineers know HSRP).

**Use VRRP when:**
- Multi-vendor environment (mandatory — HSRP/GLBP won't interop).
- IPv6 is required (VRRPv3 has the cleanest dual-stack support).
- You want preemption on by default without remembering to configure it.
- Standards compliance is a requirement (government, carrier networks).

**Use GLBP when:**
- Cisco-only environment AND you need load balancing across gateways.
- You cannot or do not want to split hosts across multiple gateway IPs.
- The network has many hosts on a single subnet and one gateway router is a bottleneck.
- Not recommended with vPC.

### 6.2 Active-Active with MLAG/vPC

True active-active gateway forwarding (both peers forward locally without peer-link hairpin)
is achieved through:

| Platform     | Feature                       | FHRP Needed?            |
|-------------|-------------------------------|------------------------|
| Cisco NX-OS | vPC + peer-gateway + HSRP     | Yes (HSRP v2)          |
| Arista EOS  | VARP (Virtual ARP)            | No (replaces FHRP)     |
| Arista EOS  | MLAG + VRRP                   | Yes                    |
| Cumulus/NVUE | VRR (Virtual Router Redundancy)| No (replaces FHRP)    |
| Juniper     | MC-LAG + VRRP                 | Yes                    |

Arista's VARP and Cumulus's VRR are not traditional FHRPs — they configure the same IP AND
MAC on both peers, making both truly active. No election, no failover delay.

### 6.3 Convergence Time Comparison

| Protocol     | Default Failover | Tuned Failover    | Notes                     |
|-------------|-----------------|-------------------|---------------------------|
| HSRP v1     | ~10 seconds     | ~3 seconds (1/3)  | Seconds granularity only  |
| HSRP v2     | ~10 seconds     | ~750ms (200ms/750ms)| Millisecond timers      |
| VRRP v2     | ~3.6 seconds    | ~1 second (1/3)   | Already faster by default |
| VRRP v3     | ~3.6 seconds    | ~300ms            | Centisecond timers        |
| GLBP        | ~10 seconds     | ~750ms            | Same timer model as HSRP  |

Note: These are FHRP detection times. Total convergence also depends on ARP cache refresh
on hosts and routing protocol convergence upstream.

## 7. Troubleshooting Deep-Dive

### 7.1 Common HSRP Issues

**Dual-Active (split-brain):**
Both routers claim Active. Causes:
- Authentication mismatch — hellos are ignored.
- ACL or firewall blocking UDP 1985 or multicast 224.0.0.102.
- VLAN mismatch — routers on different VLANs.
- CoPP (Control Plane Policing) dropping HSRP packets under load.

Diagnosis:
```
show standby                      ! Both show Active state
show standby | include Auth       ! Check authentication config
show access-lists                 ! Look for rules blocking UDP 1985
show ip mroute 224.0.0.102        ! Verify multicast reachability
show policy-map interface control-plane   ! Check CoPP counters
debug standby packets             ! See what hellos are received
```

**Failover not happening:**
- Preemption not enabled on the Standby.
- Tracked object decrement too small (Active priority still higher after decrement).
- Timer mismatch — one side has different hold timer.

**Flapping:**
- Aggressive timers on congested or lossy links.
- CPU overload on one router causing delayed hello transmission.
- STP topology changes causing temporary L2 connectivity loss.

### 7.2 Common VRRP Issues

**Master not transitioning:**
- Priority 255 (owner mode) — cannot be preempted.
- Preemption disabled (unusual but possible with `no preempt`).

**Multicast not reaching Backup:**
- Firewall blocking IP protocol 112.
- IGMP snooping dropping 224.0.0.18 (though link-local multicast should be flooded).

**VRRPv2/v3 mismatch:**
- Both sides must run the same version. v2 and v3 are not compatible.
- On IOS-XE, `vrrp N ip X.X.X.X` is v2; `vrrp N address-family` is v3.

### 7.3 Common GLBP Issues

**AVF not receiving traffic:**
- AVG is not reachable — check with `show glbp`.
- Load-balancing method is host-dependent and hash maps all hosts to same AVF.
- Redirect timer expired for a recovered AVF — it needs a new AVG assignment.

**Uneven load distribution:**
- Round-robin only balances per ARP request, not per flow or per packet.
- Large ARP cache timeouts on hosts mean infrequent rebalancing.
- Weighted mode requires manual weight tuning.

### 7.4 Comprehensive Verification Procedure

```
! === Step 1: Verify FHRP state on all routers ===
show standby brief
! Expected: One Active, one Standby, matching VIP

show vrrp brief
! Expected: One Master, one or more Backup

show glbp brief
! Expected: One AVG, multiple AVF with Active state

! === Step 2: Verify virtual MAC in ARP tables ===
! On a host or switch:
show mac address-table | include 0000.0c9f    ! HSRP v2
show mac address-table | include 0000.5e00    ! VRRP
show mac address-table | include 0007.b400    ! GLBP

! === Step 3: Verify tracking objects ===
show track
show track brief
! All tracked objects should show "UP"

! === Step 4: Test failover ===
! On Active/Master router — shut upstream interface
interface GigabitEthernet0/1
 shutdown

! Immediately check:
show standby brief                ! Should show failover
show logging | include HSRP       ! Transition logs

! === Step 5: Verify traffic flow post-failover ===
! From host: ping VIP, traceroute to remote destination
ping 10.0.1.1
traceroute 10.0.0.1

! === Step 6: Restore and verify preemption ===
interface GigabitEthernet0/1
 no shutdown

! Wait for preempt delay + routing convergence
show standby brief                ! Original Active should reclaim
```

## 8. Security Considerations

### 8.1 Rogue Router Attacks

An attacker on the subnet can send HSRP/VRRP/GLBP messages with priority 255 and become
the Active/Master router, enabling man-in-the-middle attacks.

**Mitigations:**
- HSRP/GLBP: Use MD5 authentication.
- VRRP: Use MACsec (802.1AE) on the link since VRRPv3 has no authentication.
- All: Use DHCP snooping + Dynamic ARP Inspection to prevent ARP spoofing.
- All: Use port security or 802.1X to prevent unauthorized devices on the network.
- All: Use CoPP to rate-limit FHRP packets to the control plane.

### 8.2 Control Plane Protection

```
! CoPP policy for FHRP protocols
ip access-list extended ACL-FHRP
 permit udp any host 224.0.0.102 eq 1985     ! HSRP v2
 permit 112 any host 224.0.0.18               ! VRRP
 permit udp any host 224.0.0.102 eq 3222     ! GLBP

policy-map COPP-POLICY
 class FHRP-CLASS
  police rate 500 pps burst 100 packets
   conform-action transmit
   exceed-action drop
```

## 9. FHRP and IPv6 Details

### 9.1 HSRP v2 for IPv6

HSRP v2 supports IPv6 using link-local addresses and multicast group ff02::66:

```
interface GigabitEthernet0/0
 ipv6 address 2001:db8:1::2/64
 standby version 2
 standby 2 ipv6 autoconfig       ! Auto-generates link-local VIP
 standby 2 ipv6 2001:db8:1::1    ! Explicit global VIP
 standby 2 priority 110
 standby 2 preempt
```

### 9.2 VRRPv3 for IPv6

VRRPv3 natively supports IPv6. Multiple IPv6 addresses can be associated:

```
interface GigabitEthernet0/0
 ipv6 address 2001:db8:1::3/64
 vrrp 1 address-family ipv6
  address fe80::1 primary              ! Link-local VIP
  address 2001:db8:1::1                ! Global VIP (secondary)
  priority 110
  exit-vrrp
```

### 9.3 IPv6 Considerations

- IPv6 hosts typically learn their gateway via Router Advertisements (RA), not DHCP.
- FHRP must integrate with NDP (Neighbor Discovery Protocol) instead of ARP.
- The virtual link-local address is used as the next-hop in the routing table.
- HSRP v1 and VRRP v2 do NOT support IPv6.

## 10. Summary Comparison

```
+------------------+--------------------+--------------------+--------------------+
|                  | HSRP v2            | VRRPv3             | GLBP               |
+------------------+--------------------+--------------------+--------------------+
| Standard         | Cisco proprietary  | RFC 5798           | Cisco proprietary  |
| Roles            | Active / Standby   | Master / Backup    | AVG / AVF          |
| Election         | Priority > IP      | Priority > IP      | Priority > IP (AVG)|
| Default Priority | 100                | 100                | 100                |
| Priority Range   | 0-255              | 1-254 (255=owner)  | 1-255              |
| Preemption       | Off by default     | On by default      | Off by default     |
| Multicast (v4)   | 224.0.0.102        | 224.0.0.18         | 224.0.0.102        |
| Transport        | UDP 1985           | IP protocol 112    | UDP 3222           |
| Virtual MAC      | 0000.0c9f.fXXX     | 0000.5e00.01XX     | 0007.b400.XXYY     |
| Groups           | 0-4095             | 1-255              | 0-1023             |
| Hello / Hold     | 3s / 10s           | 1s / 3.6s          | 3s / 10s           |
| Load Balancing   | Multi-group only   | Multi-group only   | Native (RR/W/HD)   |
| Authentication   | Plaintext / MD5    | None (v3)          | Plaintext / MD5    |
| IPv6             | Yes                | Yes                | Yes                |
| vPC Integration  | Excellent          | Good               | Not recommended    |
+------------------+--------------------+--------------------+--------------------+
```

## Prerequisites

- Layer 2 switching and VLANs (SVIs, trunk ports)
- IP addressing and subnetting (default gateway concept)
- ARP and MAC address tables
- Spanning Tree Protocol (understanding L2 topology)
- Basic IOS/NX-OS CLI navigation
- Interface tracking and IP SLA (for advanced configurations)
- vPC / MLAG fundamentals (for data center deployments)

## References

- [RFC 5798 — Virtual Router Redundancy Protocol (VRRP) Version 3](https://datatracker.ietf.org/doc/html/rfc5798)
- [RFC 3768 — Virtual Router Redundancy Protocol (VRRP) Version 2](https://datatracker.ietf.org/doc/html/rfc3768)
- [Cisco HSRP Configuration Guide (IOS-XE 16)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_fhrp/configuration/xe-16/fhp-xe-16-book/fhp-hsrp.html)
- [Cisco GLBP Configuration Guide (IOS-XE 16)](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_fhrp/configuration/xe-16/fhp-xe-16-book/fhp-glbp.html)
- [Cisco NX-OS HSRP Configuration Guide (9.3x)](https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/unicast/configuration/guide/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-93x/b-cisco-nexus-9000-series-nx-os-unicast-routing-configuration-guide-93x_chapter_01110.html)
- [Cisco vPC Design and Configuration Guide](https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/design_guide_c07-625857.html)
- [Cisco HSRP Support for BFD](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp_fhrp/configuration/xe-16/fhp-xe-16-book/fhp-hsrp-bfd.html)
- [Juniper VRRP Configuration](https://www.juniper.net/documentation/us/en/software/junos/high-availability/topics/topic-map/vrrp.html)
