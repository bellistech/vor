# Cisco vPC — Virtual Port Channel Architecture and Failure Analysis

> *vPC eliminates the fundamental constraint of Spanning Tree Protocol — that only one uplink can be active — by presenting two physical Nexus switches as a single logical Layer 2 device to downstream equipment. Understanding the control plane mechanics, failure domain boundaries, and consistency enforcement model is essential for designing resilient data center fabrics.*

---

## Prerequisites

- Spanning Tree Protocol (STP) fundamentals: root bridge election, port roles, convergence
- IEEE 802.3ad Link Aggregation (LACP): PDU exchange, system priority, port priority
- HSRP/VRRP: virtual IP, active/standby election, preempt behavior
- Layer 2 forwarding: MAC learning, flooding, ARP resolution
- Nexus NX-OS basics: VDC, VRF, port-channel configuration

---

## 1. vPC Architecture Overview

### The Problem vPC Solves

Traditional Spanning Tree blocks redundant links to prevent loops. In a classic triangle topology with two aggregation switches and one access switch, STP blocks one uplink, leaving 50% of bandwidth unused. EtherChannel solves this within a single chassis, but cannot span two independent switches.

vPC extends the port-channel abstraction across two physical switches (called **vPC peers**). A downstream device forms a single port-channel with member links terminating on two different Nexus switches. Both links forward simultaneously — no STP blocking, full bandwidth utilization.

### The Three Planes of vPC

vPC relies on three distinct communication channels between peers, each serving a specific purpose:

**1. Peer-Keepalive Link (Heartbeat Plane)**

The keepalive is a lightweight Layer 3 heartbeat between peers. It carries periodic UDP packets (every 1 second by default, with a 5-second timeout) on port 3200. Its sole purpose is to detect whether the remote peer is alive when the peer-link is down. It does NOT carry data traffic or control plane synchronization.

The keepalive should run over a dedicated VRF — typically the management VRF or a purpose-built VRF. Running it over the peer-link creates a circular dependency: the mechanism designed to detect peer-link failure would itself fail when the peer-link fails.

Keepalive parameters:
- **Interval:** 1 second (configurable 400ms-10s)
- **Timeout:** 5 seconds (configurable 3-20s)
- **Hold-timeout:** 3 seconds (time to wait before declaring peer dead after timeout)
- **Transport:** UDP port 3200

**2. Peer-Link (Data + Control Plane)**

The peer-link is a Layer 2 trunk port-channel between the two peers. It carries:

- **Cisco Fabric Services (CFS) messages:** MAC table sync, IGMP snooping sync, ARP sync, HSRP/VRRP state, STP BPDUs, consistency check parameters
- **Orphan traffic:** Frames from devices single-homed to one peer that need to reach a VLAN on the other peer
- **BUM traffic:** Broadcast, Unknown unicast, and Multicast frames that arrive on a vPC member port and must be flooded to the other peer's locally connected hosts
- **Redirected traffic:** When a peer receives traffic on a vPC port but the destination MAC is learned on the other peer's local port

The peer-link should be built from dedicated interfaces — minimum two 10G links in a port-channel. It must trunk all VLANs that participate in vPC. STP port type should be set to `network`.

**3. CFS (Cisco Fabric Services) Protocol**

CFS is the control-plane synchronization protocol that runs over the peer-link. It is responsible for:

- MAC address table synchronization (both peers learn all MACs)
- IGMP snooping state replication
- ARP table synchronization (when `ip arp synchronize` is enabled)
- HSRP/VRRP active/standby state awareness
- STP topology information exchange
- Consistency parameter verification

CFS uses a reliable transport — messages are acknowledged and retransmitted if lost. The synchronization is continuous, not periodic.

---

## 2. vPC Domain and Role Election

### Domain Identity

A vPC domain is identified by a numeric domain ID (1-1000). Both peers must be configured with the same domain ID. When the domain forms, NX-OS derives a **system MAC address** from the domain ID. This system MAC is used as the LACP system ID presented to downstream devices, making both peers appear as a single LACP partner.

The system MAC format:
```
00:23:04:ee:be:<domain-id-derived>
```

Both peers advertise this same system MAC in LACP PDUs. The downstream device sees one LACP system and forms a single port-channel across the two physical links.

### Role Election Process

When two peers discover each other via the keepalive and peer-link, they negotiate roles:

1. **Lower role priority wins primary** (default: 32667, range: 1-65535)
2. If priorities tie, **lower system MAC wins primary** (the real chassis MAC, not the vPC system MAC)
3. Role election happens once at initial formation; it is **sticky** — a failed primary that recovers does NOT preempt the current primary

The primary/secondary distinction matters in specific failure scenarios:

| Scenario | Primary | Secondary |
|:---|:---|:---|
| Normal operation | Forwards on all vPCs | Forwards on all vPCs |
| Peer-link fails, keepalive UP | Continues forwarding | **Suspends all vPC ports** |
| Both peer-link and keepalive fail | Continues forwarding | Continues forwarding (**split-brain**) |
| Type-1 consistency mismatch | Continues forwarding | **Suspends affected vPC** |

The secondary is always the one that yields. This asymmetry is deliberate — it prevents both switches from independently deciding to forward, which would cause loops.

### System Priority

The vPC system priority determines which peer's LACP system ID takes precedence. The peer with the lower system priority controls port-channel membership decisions. In practice, align the system priority with the role priority so the primary switch also controls LACP.

---

## 3. vPC Member Port-Channels

### How Member Port-Channels Work

A vPC member port-channel is a regular port-channel on each peer that is bound to a vPC number using the `vpc <id>` command. The vPC number is a local identifier — it does NOT need to match the port-channel number (though aligning them is a common convention for readability).

When a downstream device sends LACP PDUs, both peers respond with the same system MAC (derived from the vPC domain). The downstream device sees a single LACP partner and bundles all links into one port-channel.

Traffic distribution uses standard port-channel hashing on the downstream device. If the hash sends a frame to a link on peer A, peer A forwards it locally. If the destination is on peer B, the frame crosses the peer-link. The vPC loop-avoidance rule ensures frames received on the peer-link are never forwarded back out a vPC member port — this prevents loops without STP.

### The vPC Loop-Avoidance Rule

This is the most critical forwarding rule in vPC:

**A frame received on the peer-link is NEVER forwarded out a vPC member port.**

This rule prevents loops: if peer A receives a broadcast on vPC port 10, it floods to the peer-link. Peer B receives it on the peer-link but does NOT flood it back out its vPC port 10 member link (because the downstream device already received the frame from peer A's link). Peer B only floods it to its locally connected non-vPC ports (orphan ports) and local SVIs.

Consequence: if all member links on one peer fail for a given vPC, traffic from the downstream device arrives at the other peer. But return traffic destined to that downstream device cannot cross the peer-link and exit a vPC member port. The downstream device must rehash to reach the surviving peer's links directly.

---

## 4. Consistency Checks — The Type-1/Type-2 Model

### Why Consistency Checks Exist

Both vPC peers must present a consistent Layer 2 environment to the downstream device. If one peer has MTU 9216 and the other has MTU 1500 on the same vPC, jumbo frames would be silently dropped on one path. If one peer trunks VLAN 100 and the other doesn't, hosts in VLAN 100 would experience intermittent connectivity.

vPC uses CFS to exchange configuration parameters between peers and flag mismatches.

### Type-1: Mandatory Consistency Parameters

Type-1 parameters are considered critical enough that a mismatch **suspends the vPC** (on the secondary peer, if graceful consistency check is enabled). These include:

| Parameter | Why It's Mandatory |
|:---|:---|
| LACP mode | Both peers must use the same negotiation mode |
| Switchport mode (access/trunk) | Mismatched modes cause frame format errors |
| Allowed VLANs on trunk | Missing VLANs cause connectivity loss |
| STP mode (RPVST+/MST) | Mismatched modes cause STP failures |
| STP port type | Affects BPDU handling and convergence |
| MTU | Mismatched MTU causes silent drops |
| Speed/duplex | Mismatched speeds cause physical layer issues |
| Storm control | Inconsistent policing causes asymmetric drops |

When a Type-1 mismatch is detected:
1. CFS notifies both peers of the mismatch
2. With `graceful consistency-check` (default), only the secondary suspends its vPC leg
3. Without graceful check, BOTH peers suspend — total outage
4. Fixing the mismatch automatically restores the vPC

### Type-2: Advisory Consistency Parameters

Type-2 parameters are logged as warnings but do NOT suspend the vPC. They include STP cost/priority, BPDU filter/guard, DHCP snooping, IGMP snooping settings, and ACL/QoS policies.

While Type-2 mismatches don't cause immediate outage, they can cause subtle problems: asymmetric STP costs may cause suboptimal forwarding; mismatched DHCP snooping can drop DHCP requests on one path.

### Global vs. Per-Interface Checks

Consistency checks operate at two levels:

- **Global:** Parameters that apply to the entire vPC domain (STP mode, system priority)
- **Per-vPC interface:** Parameters specific to each vPC member (VLANs, MTU, switchport mode)

A global mismatch suspends ALL vPCs. A per-interface mismatch suspends only the affected vPC.

---

## 5. Failure Scenario Analysis

### Failure Matrix

Understanding vPC behavior requires analyzing the state of three components: member links, peer-link, and keepalive. The matrix below covers every combination:

| Member Links | Peer-Link | Keepalive | Result |
|:---|:---|:---|:---|
| All UP | UP | UP | Normal operation |
| Partial failure on one peer | UP | UP | Traffic rehashes; peer-link carries overflow |
| All links on one peer fail | UP | UP | All traffic via surviving peer; peer-link carries return traffic |
| UP | DOWN | UP | Secondary suspends all vPCs |
| UP | DOWN | DOWN | **Split-brain** — both peers forward independently |
| UP | UP | DOWN | No immediate impact; fix urgently |
| Peer A reloads | N/A | N/A | Peer B continues; A recovers via delay restore |
| Both peers reload | N/A | N/A | auto-recovery elects one peer after timer |

### Peer-Link Failure (Keepalive UP) — Deep Dive

This is the most operationally significant failure scenario. When the peer-link fails:

1. CFS synchronization stops immediately
2. Both peers detect peer-link down
3. Both peers check keepalive status — keepalive is UP
4. Secondary detects that the primary is alive (via keepalive)
5. **Secondary suspends ALL its vPC member ports**
6. Downstream devices detect link failure on secondary's ports
7. LACP on downstream devices reconverges to use only primary's ports
8. Traffic flows exclusively through the primary peer

The secondary suspends because if both peers forwarded independently without CFS synchronization, MAC tables would diverge, BUM traffic would be duplicated, and STP would see two independent bridges — causing loops.

After the peer-link is restored:
1. CFS re-synchronizes MAC tables, ARP, IGMP state
2. Secondary un-suspends its vPC member ports
3. Downstream LACP reconverges to use both peers
4. Normal dual-active forwarding resumes

### Split-Brain (Peer-Link + Keepalive Both Down) — Deep Dive

This is the worst-case scenario. Neither peer can determine if the other is alive:

1. Primary assumes secondary is dead — continues forwarding
2. Secondary assumes primary is dead — continues forwarding
3. Both peers advertise the same vPC system MAC via LACP
4. Downstream device sees both links as valid (same LACP system)
5. Both peers independently learn MACs and forward traffic
6. **Result:** Duplicate frames, MAC table instability, potential loops

Mitigation strategies:
- **Auto-recovery:** After the reload-delay timer, one peer (based on lower MAC) disables its vPCs
- **Physical diversity:** Route peer-link and keepalive over different physical paths
- **Dual keepalive paths:** Use both mgmt and a routed interface for keepalive redundancy
- **Monitoring:** Alert on keepalive failure immediately; treat as P1

### Simultaneous Peer Reload

When both peers lose power and recover:

1. Both peers boot and initialize
2. Neither has an active keepalive or peer-link partner initially
3. Without auto-recovery, both peers wait indefinitely for the other
4. **With auto-recovery:** After `reload-delay` timer (default 240s), the peer with the lower system MAC assumes primary and brings up vPCs unilaterally
5. When the other peer completes boot, it joins as secondary and normal vPC forms

---

## 6. Orphan Ports — The Overlooked Risk

### Definition and Problem

An orphan port is any port on a vPC peer that is NOT part of a vPC member port-channel. This includes:

- Single-homed servers connected to only one peer
- Management ports, monitoring taps, or out-of-band connections
- Non-vPC port-channels (standard EtherChannels within a single chassis)

The risk: when the peer-link fails and the secondary suspends its vPC ports, orphan ports on the secondary remain UP. Hosts on those orphan ports can still reach the secondary's local SVIs, but they cannot reach anything beyond the secondary because:

1. The peer-link is down (no path to primary's networks)
2. vPC member ports are suspended (no path to downstream switches)
3. The host is effectively isolated

### Orphan Port Suspend

The `vpc orphan-ports suspend` command on an interface tells NX-OS to shut that port when the secondary suspends its vPCs (peer-link failure). This forces the orphan host to detect a link failure and fail over via an alternate path (if one exists at Layer 3).

This feature is only useful if the orphan device has an alternative path — a management network, a routed connection to another switch, or a secondary NIC. If the device is truly single-homed with no alternative, suspending it merely makes the failure explicit rather than leaving the host in a half-connected state.

---

## 7. Peer-Gateway and ARP Synchronization

### The Peer-Gateway Problem

In a standard HSRP deployment with vPC, both peers share a virtual IP and virtual MAC. The HSRP active peer responds to ARP requests with the virtual MAC. All hosts send traffic to the virtual MAC, and whichever peer receives it (via the vPC hash) routes it.

The problem arises with certain devices (notably NetApp storage controllers, F5 load balancers, and some older servers) that source packets with the **physical router MAC** of one peer as the destination, rather than the HSRP virtual MAC. This happens when these devices cache the MAC from a unicast ARP reply or ICMP redirect.

Without peer-gateway: if such a packet hashes to the wrong peer (the one whose physical MAC is NOT the destination), that peer drops the packet because the destination MAC is not its own.

### How Peer-Gateway Works

With `peer-gateway` enabled, each peer programs the other peer's physical router MAC into its forwarding table as a local MAC. When peer A receives a frame destined to peer B's router MAC, peer A routes it locally instead of bridging it across the peer-link.

This requires ARP synchronization to work correctly. Without `ip arp synchronize`, peer A might not have ARP entries for hosts that were learned by peer B. The packet would be routed to the correct SVI but then require an ARP resolution, adding latency or causing drops.

With both `peer-gateway` and `ip arp synchronize`:
1. Both peers have identical ARP tables (via CFS sync)
2. Both peers accept traffic destined to either peer's router MAC
3. Both peers can route traffic for any host in the VLAN
4. No unnecessary traffic crosses the peer-link

### Peer-Gateway with Layer 3 Routing

Peer-gateway has an interaction with routing protocols. If both peers are running OSPF or BGP and advertising the same subnets, packets may arrive at either peer via Layer 3 routing and then need to be routed to a local VLAN. With peer-gateway, the routed packet can be delivered regardless of which peer's MAC the next-hop ARP resolves to.

---

## 8. Delay Restore Timers

### The Convergence Race

When a vPC peer reloads, there is a window between:
- **Port-channel UP:** Physical links come up, LACP forms, vPC member ports activate
- **Routing protocol convergence:** OSPF/BGP adjacencies form, routes are learned, FIB is populated

If vPC member ports come up before routing converges, the recovering peer attracts traffic (via LACP hash) but cannot route it — traffic is blackholed.

### Timer Hierarchy

vPC provides three independent delay restore timers:

1. **`delay restore`** (default 30s, recommended 60-120s): Delays vPC member port-channel activation after peer adjacency forms. This is the primary timer.

2. **`delay restore interface-vlan`** (default 10s, recommended 30-45s): Delays SVI activation. Even after vPC ports are up, SVIs remain down until this timer expires. This gives routing protocols time to converge over the newly-active SVIs.

3. **`delay restore orphan-port`** (default 0s): Delays orphan port activation. Useful if orphan-connected devices depend on routing being available.

The timers start when the peer adjacency forms (peer-link + keepalive both UP). They do not start at system boot.

### Tuning Guidelines

The delay restore timer should be longer than the worst-case routing convergence time for the platform:

| Protocol | Typical Convergence | Recommended delay restore |
|:---|:---|:---|
| OSPF (default timers) | 30-40s | 60s |
| OSPF (BFD-assisted) | 3-5s | 30s |
| BGP (default timers) | 60-180s | 120s |
| BGP (BFD-assisted) | 3-5s | 30s |
| Static routes | Immediate | 30s (for ARP/MAC learning) |

---

## 9. vPC with HSRP/VRRP

### Alignment Principle

In a vPC domain running HSRP, the vPC primary should also be the HSRP active router. This alignment ensures that:

1. The primary peer is the "preferred" forwarder for both Layer 2 (vPC) and Layer 3 (HSRP)
2. When the primary fails, the secondary takes over both vPC primary role (sticky) and HSRP active role
3. Traffic patterns are predictable and debuggable

To achieve alignment:
- Set vPC `role priority` lower on the intended primary
- Set HSRP `priority` higher on the same switch
- Enable HSRP `preempt` so the primary reclaims active after recovery

### HSRP Behavior During vPC Failures

| Failure | HSRP Active | HSRP Standby | Effect |
|:---|:---|:---|:---|
| Primary member links fail | Primary (still) | Secondary | Traffic via secondary vPC links, routed by primary via peer-link |
| Secondary member links fail | Primary | Secondary (still) | No change — primary still forwards |
| Peer-link fails (keepalive UP) | Primary | Secondary suspends vPCs | All traffic via primary; secondary's HSRP standby still up but unreachable for L2 |
| Primary reloads | Switches to secondary | N/A | Secondary becomes HSRP active; vPC auto-recovery if configured |

### The Forwarding Asymmetry Problem

Without peer-gateway, traffic destined to the HSRP virtual MAC works correctly. But return traffic from the routed destination may arrive at either peer (based on upstream routing). If it arrives at the standby peer, that peer must bridge it across the peer-link to reach the active peer for routing, then back across the peer-link to reach the vPC member port — a "tromboning" pattern that wastes peer-link bandwidth.

Peer-gateway eliminates this by letting the standby peer route directly. Combined with `ip arp synchronize`, the standby has the necessary ARP entries and routes the packet locally.

---

## 10. vPC with FEX (Fabric Extender)

### Dual-Homed FEX Topology

A dual-homed FEX connects to both vPC peers via a vPC fabric port-channel. This is the recommended topology because:

1. If one parent switch fails, the FEX remains operational via the other parent
2. Host-facing ports on the FEX continue forwarding through the surviving parent
3. The FEX appears as a remote line card to both parents simultaneously

The fabric port-channel between the FEX and each parent must be configured as a vPC with matching `vpc <id>` on both parents. The FEX ID must be the same on both parents.

### Single-Homed FEX Topology

A single-homed FEX connects to only one vPC peer. If that peer fails, the FEX and all devices connected to it lose connectivity. This topology is used when:

- Physical cabling constraints prevent dual-homing
- The connected devices are non-critical and don't require high availability
- Cost constraints limit the number of uplink ports

Single-homed FEX ports are effectively orphan ports from a vPC perspective. Consider configuring `vpc orphan-ports suspend` on the fabric port-channel if the FEX hosts have alternate paths.

### FEX Host vPC (Enhanced vPC)

In an enhanced vPC topology, a host connected to a dual-homed FEX can itself form a port-channel (host vPC) with links going to the FEX connected to different parent switches. This provides end-to-end active-active forwarding from the host through the FEX to the aggregation layer.

---

## 11. vPC Peer-Switch — STP Optimization

### The STP Convergence Problem

In a standard vPC domain, each peer has its own STP bridge ID. When one peer reloads, downstream switches detect STP topology change — the root bridge (if it was the reloaded peer) is lost, a new root is elected, and STP reconverges. This can take seconds (RPVST+) and briefly disrupts traffic.

### How Peer-Switch Solves It

With `peer-switch` enabled, both peers use the same STP bridge ID (derived from the vPC domain). Downstream switches see a single STP bridge. When one peer reloads:

1. The surviving peer continues sending BPDUs with the same bridge ID
2. Downstream switches see no STP topology change
3. No reconvergence, no traffic disruption

Requirements:
- STP mode must be RPVST+ or MST (not legacy PVST+)
- Both peers must be configured as STP root for the relevant VLANs
- Both peers must have `peer-switch` enabled
- Bridge priority should be set identically on both peers

### Interaction with STP Bridge Assurance

Bridge Assurance sends BPDUs bidirectionally on point-to-point links. With peer-switch, both peers send BPDUs with the same bridge ID. If one peer fails, the surviving peer's BPDUs keep Bridge Assurance satisfied on all peer-link and vPC member ports.

---

## 12. vPC Auto-Recovery

### The Simultaneous Failure Problem

Auto-recovery addresses the scenario where both peers reload simultaneously (e.g., data center power event). Without auto-recovery:

1. Both peers boot and initialize NX-OS
2. Peer-link and keepalive are not yet established (both peers are still booting)
3. Neither peer can determine if the other is alive
4. Both peers keep vPC member ports down indefinitely, waiting for peer adjacency
5. **All connected devices are down** even though both switches are physically operational

### Auto-Recovery Mechanism

With auto-recovery enabled:

1. Both peers boot and start the `reload-delay` timer (default 240s)
2. During the timer window, both peers attempt to form peer adjacency normally
3. If the timer expires and no peer adjacency has formed:
   - The peer with the lower operational system MAC assumes primary role
   - It brings up vPC member ports unilaterally
   - It operates in "auto-recovery" mode — a degraded state where vPC functions without a peer
4. When the other peer completes boot and establishes keepalive/peer-link:
   - Normal vPC formation occurs
   - The auto-recovery peer may remain primary (roles are sticky)
   - CFS synchronization begins, and full dual-active forwarding resumes

### Tuning the Reload-Delay

The reload-delay should be long enough for a normal boot to complete and form peer adjacency. If set too short, one peer may enter auto-recovery while the other is still booting, causing a brief period of single-peer operation followed by reconvergence.

Recommended: 240-360 seconds, depending on platform boot time.

---

## 13. Design Best Practices

### Physical Design

1. **Peer-link:** Minimum 2x10G (or 2x40G/100G), dedicated interfaces, no shared fabric modules. Use interfaces from different line cards/ASICs for resilience.

2. **Peer-keepalive:** Dedicated management VRF. Use a direct cable or a dedicated out-of-band network. Never route keepalive over the peer-link or over the vPC domain's production network.

3. **Diversity:** Peer-link and keepalive should traverse different physical paths, different cable trays, and ideally different patch panels.

4. **Cabling:** Label every vPC member link with the vPC ID, local port-channel number, and remote device. Documentation is the difference between a 5-minute fix and a 2-hour outage.

### Logical Design

1. **Role priority alignment:** Set the intended primary's vPC role priority lower than the secondary's. Align with HSRP priority on the same switch.

2. **Domain per pair:** Each pair of vPC peers gets its own domain ID. Never reuse domain IDs across different switch pairs.

3. **VLAN pruning:** Only allow necessary VLANs on vPC member trunks. Over-permissive trunk allowed lists increase the blast radius of broadcast storms.

4. **STP root:** Both vPC peers should be STP root (primary and secondary) for all VLANs in the domain. No downstream switch should ever become root.

5. **Consistency first:** Before activating a new vPC, verify consistency parameters. A Type-1 mismatch on activation will immediately suspend the secondary's leg.

### Operational Practices

1. **Upgrade procedure:** Always upgrade one peer at a time. Upgrade the secondary first (it's the one that yields). After the secondary is stable, upgrade the primary. vPC ISSU (In-Service Software Upgrade) is supported on some platforms but has strict prerequisites.

2. **Pre-change validation:** Before any vPC change, run `show vpc consistency-parameters global` and `show vpc consistency-parameters vpc <id>` on both peers. Save the output.

3. **Monitoring:** Alert on: peer-keepalive failure, peer-link utilization above 30%, vPC member port suspension, Type-1 consistency mismatch, role change.

4. **Documentation:** Maintain a vPC topology diagram showing: domain IDs, vPC numbers, port-channel numbers, member interfaces, peer-link interfaces, keepalive interfaces, HSRP VIPs, and role priorities.

---

## 14. Troubleshooting Methodology

### Step 1: Establish Baseline

```
show vpc                              ! Overall status, peer adjacency, role
show vpc role                         ! Primary/secondary, role priority, system MAC
show vpc brief                        ! Compact summary of all vPC member status
```

### Step 2: Check Control Plane

```
show vpc peer-keepalive               ! Keepalive status, last received timestamp
show cfs peers                        ! CFS peer status (should show peer switch)
show cfs application                  ! Which applications are using CFS
```

### Step 3: Validate Consistency

```
show vpc consistency-parameters global     ! Global Type-1 and Type-2 parameters
show vpc consistency-parameters vpc <id>   ! Per-vPC parameters
show vpc consistency-parameters vlans      ! VLAN-level consistency
```

### Step 4: Inspect Member Ports

```
show port-channel summary                 ! All port-channels, member status (P/D/I/s)
show lacp counters                        ! LACP PDU counts (should be incrementing)
show lacp neighbor                        ! Remote LACP system ID and port info
show interface port-channel <id>          ! Detailed counters, errors, drops
```

### Step 5: Verify Forwarding

```
show mac address-table                    ! MAC entries — should appear on both peers
show ip arp vrf <vrf>                     ! ARP entries — should be synchronized
show ip arp synchronize vpc               ! ARP sync status and counters
show forwarding adjacency                 ! FIB entries for routed traffic
```

### Step 6: Check STP Interaction

```
show spanning-tree vlan <id>              ! STP state for specific VLAN
show spanning-tree summary                ! Bridge IDs — should match with peer-switch
show spanning-tree inconsistentports      ! Ports in STP inconsistent state
```

### Common Diagnostic Patterns

**Pattern: vPC member shows "suspended"**
1. Check `show vpc consistency-parameters vpc <id>` — look for Type-1 mismatch
2. Compare configuration on both peers for that vPC (VLANs, MTU, switchport mode)
3. Fix the mismatch — vPC restores automatically

**Pattern: MAC flapping between peers**
1. Indicates possible split-brain — check `show vpc role` on both peers
2. If both show "primary," you have a dual-active condition
3. Check peer-link and keepalive status
4. If keepalive is up but peer-link is down, secondary should have suspended — investigate why it didn't

**Pattern: Traffic blackholing after peer reload**
1. Check if delay restore timer was sufficient for routing convergence
2. Verify `show ip route` has all expected routes before vPC ports come up
3. Increase `delay restore` timer if routing wasn't converged

**Pattern: Asymmetric traffic flows (excessive peer-link utilization)**
1. Check if `peer-gateway` is enabled — without it, traffic may trombone
2. Check if `ip arp synchronize` is enabled — without it, ARP misses cause peer-link forwarding
3. Verify downstream device hashing — if most flows hash to one peer, rebalance port-channel members

---

## 15. vPC Scalability and Platform Limits

### Nexus Platform Limits

Scalability varies by platform. Key parameters to verify before deployment:

| Parameter | Typical Limit |
|:---|:---|
| vPC member port-channels per domain | 528 (N9K), 256 (N7K), 256 (N5K) |
| VLANs per vPC domain | 4094 |
| MAC addresses (combined table) | 128K-256K depending on TCAM |
| Port-channel members per channel | 32 (N9K), 16 (N7K/N5K) |
| FEX per vPC pair | 24-48 depending on model |

Always consult the NX-OS Verified Scalability Guide for the specific platform and software version.

### Performance Considerations

The peer-link carries control traffic (CFS), orphan traffic, and BUM traffic. In a well-designed vPC domain:

- Peer-link utilization should be below 30% steady-state
- High peer-link utilization indicates: too many orphan ports, asymmetric hashing, or insufficient vPC member bandwidth
- CFS overhead is typically negligible (kilobits) compared to data traffic

---

## References

- Cisco Nexus 9000 Series NX-OS vPC Configuration Guide — https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/93x/vxlan/configuration/guide/b-cisco-nexus-9000-series-nx-os-vxlan-configuration-guide-93x.html
- Cisco vPC Best Practices Design Guide (CVD) — https://www.cisco.com/c/en/us/products/collateral/switches/nexus-5000-series-switches/design_guide_c07-625857.html
- Cisco Nexus 7000 vPC Operations — https://www.cisco.com/c/en/us/support/docs/switches/nexus-7000-series-switches/200different-vpc-operations.html
- Cisco NX-OS Verified Scalability Guide — https://www.cisco.com/c/en/us/support/switches/nexus-9000-series-switches/products-installation-and-configuration-guides-list.html
- IEEE 802.1AX — Link Aggregation standard
- RFC 7348 — Virtual eXtensible Local Area Network (VXLAN), relevant for vPC with VXLAN fabrics
- Cisco Live Presentations: BRKDCN-2095 — vPC Deep Dive (annually updated)
