# JunOS Multicast — Deep Dive Theory and Analysis

> In-depth exploration of JunOS multicast implementation: forwarding architecture, PIM state machines, RPF mechanics (inet.2), multicast in routing instances, mVPN implementation details, and troubleshooting methodology. For JNCIE-SP level understanding.

## 1. JunOS Multicast Forwarding Architecture

### 1.1 Multicast Routing Tables

JunOS uses dedicated routing tables for multicast:

| Table     | Purpose                                                          |
|-----------|------------------------------------------------------------------|
| `inet.0`  | Unicast routes; used for RPF by default when inet.2 is empty     |
| `inet.1`  | Multicast forwarding cache; contains (S,G) and (*,G) entries     |
| `inet.2`  | Multicast RPF table; when populated, used instead of inet.0      |

**inet.1 (Forwarding Cache):**
- Contains active multicast forwarding entries
- Populated by PIM when joins are received and sources are active
- Each entry maps (S,G) or (*,G) to an incoming interface (RPF) and outgoing interface list (OIL)
- Entries have timers and are removed when PIM state is pruned

**inet.2 (RPF Table):**
- Dedicated table for multicast RPF lookups
- By default, empty — RPF falls back to inet.0
- Can be populated via `rib-group`, static routes, or multicast-specific protocol imports
- Critical for scenarios where unicast and multicast topologies diverge

### 1.2 RPF Check Mechanics

Reverse Path Forwarding is the fundamental multicast loop prevention mechanism:

1. Multicast packet arrives on interface X
2. Router looks up the source address in the RPF table (inet.2, then inet.0)
3. RPF lookup returns the expected incoming interface for that source
4. **If X matches the RPF interface:** Packet passes RPF check, forwarded to OIL
5. **If X does not match:** Packet fails RPF check, dropped silently

**Why inet.2 exists:** In asymmetric routing scenarios, the unicast best path (inet.0) may differ from the multicast best path. inet.2 allows multicast to use a different RPF topology without affecting unicast forwarding.

### 1.3 Multicast Forwarding in the PFE

JunOS multicast forwarding on the PFE:

1. **First packet (no cache entry):** Punted to RE for PIM processing (software forwarding)
2. **PIM creates state:** RE installs (S,G) or (*,G) entry in inet.1
3. **Subsequent packets:** PFE forwards at hardware speed using inet.1 cache
4. **OIL replication:** PFE replicates packets to all interfaces in the OIL using hardware multicast replication engine

**Performance implication:** First-packet latency is higher (RE processing). Steady-state forwarding is at line rate. This is why multicast convergence after topology changes involves a brief period of RE-based forwarding.

### 1.4 Multicast Forwarding Next-Hops

JunOS uses special next-hop types for multicast:

| Next-Hop Type   | Description                                          |
|-----------------|------------------------------------------------------|
| `mcst`          | Multicast composite next-hop (list of OIL interfaces)|
| `mdsc`          | Multicast discard (no receivers)                     |
| `mgrp`          | Multicast group next-hop                             |

The composite next-hop (`mcst`) contains pointers to all outgoing interfaces. When the OIL changes (join/prune), the composite next-hop is updated and pushed to the PFE.

## 2. PIM Implementation in JunOS

### 2.1 PIM-SM State Machine

PIM-SM maintains per-(S,G) and per-(*,G) state on each router:

**(*,G) State (Shared Tree / RPT):**
```
States: NoInfo, Joined, Pruned
          NoInfo
           / \
    Join  /   \ Prune
         v     v
      Joined  Pruned
         \     /
    Prune \   / Join
           v v
          NoInfo
```

- **NoInfo:** No downstream receivers, no upstream join
- **Joined:** Downstream receivers exist, upstream (*,G) Join sent toward RP
- **Pruned:** Explicitly pruned (used in dense mode or RPT prune in SM)

**( S,G) State (Source Tree / SPT):**
```
States: NoInfo, Joined, Pruned
```

- Tracks source-specific state
- SPT switchover: When data rate exceeds threshold, last-hop router joins (S,G) SPT directly to source, prunes (S,G) from RPT

### 2.2 SPT Switchover

JunOS SPT switchover behavior:

1. Receiver joins (*,G) — traffic flows via RP (RPT)
2. First-hop router (near source) encapsulates in Register to RP
3. RP de-encapsulates, forwards down RPT to receivers
4. RP joins (S,G) SPT toward source
5. **Last-hop router:** After receiving data on RPT, switches to (S,G) SPT:
   - Sends (S,G) Join toward source
   - Sends (S,G,rpt) Prune toward RP
   - Traffic now flows directly from source via SPT

**SPT threshold in JunOS:**

```
protocols {
    pim {
        spt-threshold {
            infinity;    /* never switch to SPT — stay on RPT */
            /* default: switch immediately on first packet */
        }
    }
}
```

Default: immediate switchover (threshold = 0). Setting `infinity` keeps traffic on the RPT permanently (useful for low-bandwidth multicast in hub-and-spoke).

### 2.3 PIM Assert

When two PIM routers share a multi-access segment and both forward the same multicast group, duplicate packets occur. PIM Assert resolves this:

1. Both routers detect duplicate multicast traffic on the shared segment
2. Each sends a PIM Assert message containing:
   - RPT bit (whether the route is RPT or SPT)
   - Metric preference (route preference to the source/RP)
   - Metric (IGP metric to the source/RP)
3. **Assert winner:** Router with:
   - SPT preferred over RPT
   - Then lowest metric preference
   - Then lowest metric
   - Then highest IP address (tiebreaker)
4. Assert loser stops forwarding on that interface

### 2.4 PIM DR Election

On multi-access segments, the Designated Router (DR) is responsible for:
- Sending PIM Register messages to RP (for sources)
- Sending (*,G) Joins toward RP (for receivers)

**Election:**
1. Highest `priority` value wins (default 1, range 0-4294967295)
2. If tie: highest IP address wins
3. DR sends periodic Hello messages advertising its priority

```
protocols {
    pim {
        interface ge-0/0/2.0 {
            priority 200;    /* higher = more preferred */
        }
    }
}
```

## 3. RPF in JunOS (inet.2 Table)

### 3.1 RPF Lookup Precedence

JunOS RPF lookup order:
1. Check `inet.2` for the source address
2. If `inet.2` has no matching route, fall back to `inet.0`
3. The matching route's next-hop determines the expected RPF interface

### 3.2 Populating inet.2

**Method 1: rib-group (recommended)**
Copy routes from inet.0 to inet.2:

```
routing-options {
    interface-routes {
        rib-group inet MCAST-RIB;
    }
    rib-groups {
        MCAST-RIB {
            import-rib [ inet.0 inet.2 ];
            import-policy MCAST-RPF-FILTER;
        }
    }
}
```

**Method 2: Static routes in inet.2**
For specific RPF overrides:

```
routing-options {
    rib inet.2 {
        static {
            route 10.10.0.0/16 next-hop 10.0.0.1;
        }
    }
}
```

**Method 3: MBGP (Multicast BGP)**
BGP with `family inet multicast` populates inet.2:

```
protocols {
    bgp {
        group MBGP {
            family inet {
                multicast;
            }
        }
    }
}
```

### 3.3 RPF Policy for Asymmetric Routing

When the unicast path and multicast path differ (e.g., due to traffic engineering), RPF policy forces multicast to use a specific interface:

```
routing-options {
    multicast {
        rpf-check-policy RPF-OVERRIDE;
    }
}
policy-options {
    policy-statement RPF-OVERRIDE {
        term SOURCE-OVERRIDE {
            from {
                source-address-filter 10.10.0.0/16 orlonger;
            }
            then {
                rpf-check-nexthop ge-0/0/1.0;
                accept;
            }
        }
    }
}
```

### 3.4 Static RPF

For sources behind a specific interface that may not be in the routing table:

```
routing-options {
    multicast {
        scope-policy SCOPE-FILTER;
        flow-map STATIC-RPF {
            source 10.10.10.0/24;
            group 239.1.1.0/24;
            action {
                interface ge-0/0/1.0;
            }
        }
    }
}
```

## 4. Multicast in Routing Instances

### 4.1 VRF Multicast Tables

When multicast is enabled in a VRF, JunOS creates:
- `<vrf>.inet.1` — VRF multicast forwarding cache
- `<vrf>.inet.2` — VRF multicast RPF table (if populated)
- PIM runs independently within the VRF

### 4.2 PE-CE Multicast Interactions

**Source behind CE:**
1. CE runs PIM with PE (PE is DR or not)
2. Source sends multicast, first-hop PE Register-encapsulates to customer RP
3. Customer RP is reachable via the VRF routing table
4. RP can be in the same VRF on another PE, or behind another CE

**Receiver behind CE:**
1. CE sends IGMP Report to PE
2. PE sends PIM (*,G) Join toward customer RP within the VRF
3. Join is forwarded across MPLS backbone via mVPN

### 4.3 Provider Multicast vs Customer Multicast

Two independent multicast domains:
- **Provider multicast:** PIM in the global routing instance, provider RP, used for draft-rosen P-tunnel or NG-mVPN signaling
- **Customer multicast:** PIM within the VRF, customer RP, carries actual customer multicast traffic

These must be interconnected via mVPN (draft-rosen or NG-mVPN) to transport customer multicast across the provider MPLS backbone.

## 5. mVPN Implementation

### 5.1 Draft-Rosen (PIM/GRE)

**Architecture:**
- Provider core runs PIM-SM in the global instance
- Each mVPN creates a provider multicast group (one per VRF)
- PE encapsulates customer multicast in GRE, with outer destination = provider multicast group
- Provider PIM distributes these GRE-encapsulated packets via the provider multicast tree

**Advantages:**
- Simple to understand (just PIM + GRE)
- No BGP changes required

**Disadvantages:**
- Requires PIM in the provider core (additional protocol complexity)
- GRE encapsulation adds overhead
- Provider multicast group per VRF consumes provider multicast state
- Scaling limited by provider PIM state

### 5.2 NG-mVPN (BGP-Based, RFC 6513/6514)

**Architecture:**
- BGP carries mVPN signaling (C-multicast routes: Type 6 and Type 7)
- Provider tunnel (P-tunnel) can be: ingress replication, mLDP P2MP, RSVP-TE P2MP, or PIM/GRE
- No PIM required in the provider core (unless PIM P-tunnel chosen)
- BGP auto-discovery for mVPN membership

**mVPN BGP Route Types:**

| Type | Name                        | Purpose                                    |
|------|-----------------------------|--------------------------------------------|
| 1    | Intra-AS I-PMSI Auto-Discovery | Discover PEs in the mVPN                |
| 2    | Inter-AS I-PMSI Auto-Discovery | Inter-AS mVPN discovery                 |
| 3    | S-PMSI Auto-Discovery      | Selective tunnel binding for (S,G)         |
| 4    | Leaf Auto-Discovery         | Receiver PE joining selective tunnel       |
| 5    | Source Active               | Replaces MSDP SA (source active notification)|
| 6    | Shared Tree Join (C-*,G)   | Customer RPT join across provider backbone |
| 7    | Source Tree Join (C-S,G)   | Customer SPT join across provider backbone |

### 5.3 P-Tunnel Types Analysis

**Ingress Replication:**
- PE replicates packet to each receiver PE individually (unicast MPLS)
- No multicast in the core at all
- Simple but does not scale for high-bandwidth streams with many receivers
- Best for: low-rate, few-receiver mVPN flows

**mLDP P2MP:**
- LDP-based P2MP tree in the provider core
- No RSVP-TE required
- Automatically built when receiver PEs join
- Scales well for moderate-to-large deployments
- Requires LDP with `p2mp` extension

**RSVP-TE P2MP:**
- Traffic-engineered P2MP tree with bandwidth guarantees
- Most resource-intensive to set up (RSVP state per tree)
- Best for: high-bandwidth, QoS-sensitive multicast

### 5.4 Inclusive vs Selective P-Tunnels

**Inclusive (I-PMSI):**
- Default P-tunnel that carries ALL multicast traffic for a VRF
- All PEs in the mVPN join the I-PMSI
- Simple but wasteful: PEs without interested receivers still receive traffic

**Selective (S-PMSI):**
- Per-(S,G) or per-(*,G) P-tunnel
- Only PEs with interested receivers join
- Reduces unnecessary traffic replication
- Configured with thresholds: traffic switches from I-PMSI to S-PMSI when rate exceeds threshold

```
provider-tunnel {
    selective {
        group 239.0.0.0/8 {
            source 0.0.0.0/0 {
                threshold-rate 10;    /* kbps threshold for S-PMSI */
                ldp-p2mp;
            }
        }
    }
}
```

## 6. Multicast Troubleshooting Methodology

### 6.1 Systematic Approach

**Step 1: Verify receiver registration**
```
show igmp group                        # confirm IGMP membership
show igmp interface                    # confirm IGMP-enabled interface
```

**Step 2: Verify PIM state**
```
show pim join extensive                # check (*,G) and (S,G) state
show pim rps                           # verify RP is known
show pim neighbors                     # verify PIM adjacencies
```

**Step 3: Verify RPF**
```
show multicast rpf <source-ip>         # check RPF for source
show route table inet.2                # check multicast RPF table
show route <source-ip> table inet.0    # fallback RPF lookup
```

**Step 4: Verify forwarding**
```
show multicast route                   # check forwarding cache
show multicast route extensive         # detailed forwarding entries
show route table inet.1                # multicast forwarding table
```

**Step 5: Verify data plane**
```
show interfaces <intf> statistics      # multicast counters
monitor traffic interface <intf> matching "multicast"
```

### 6.2 Common Failure Modes

**RPF failure:**
- Symptom: PIM join sent but no data received
- Cause: Source reachable via different interface than expected
- Fix: Verify inet.2 or inet.0 has correct route to source; use RPF policy if needed

**RP unreachable:**
- Symptom: (*,G) state shows "RP not reachable"
- Cause: RP address not in routing table or not advertised
- Fix: Verify RP configuration, check routing to RP address

**DR election mismatch:**
- Symptom: IGMP reports processed but no PIM join sent
- Cause: Router is not DR on the receiver segment
- Fix: Check DR election (`show pim interfaces`), adjust priority

**MTU issues:**
- Symptom: Multicast works for small packets but fails for large
- Cause: Path MTU too small for multicast + encapsulation (especially mVPN/GRE)
- Fix: Ensure consistent MTU across all interfaces in the multicast path

**TTL expiry:**
- Symptom: Multicast reaches some hops but not all
- Cause: Multicast TTL decremented at each hop; initial TTL too low
- Fix: Check source application TTL settings; verify no unexpected hops

### 6.3 mVPN-Specific Troubleshooting

```bash
# Verify mVPN PE discovery
show mvpn neighbor instance VRF-CUSTA

# Verify P-tunnel
show mvpn provider-tunnel instance VRF-CUSTA

# Verify C-multicast routes
show route table VRF-CUSTA.mvpn.0

# Verify BGP mVPN routes
show bgp summary | match mvpn

# Verify customer multicast in VRF
show multicast route instance VRF-CUSTA

# Verify PIM in VRF
show pim join instance VRF-CUSTA
show pim rps instance VRF-CUSTA
```

### 6.4 Performance Monitoring

Key metrics to monitor:
- **Multicast routes:** `show multicast route count` — track total (S,G) and (*,G) entries
- **PIM state:** `show pim join count` — number of PIM join entries
- **IGMP groups:** `show igmp group count` — receiver-side group count
- **PFE multicast replication:** `show pfe statistics traffic` — hardware replication counters
- **RE CPU:** Excessive multicast control-plane activity can overload the RE

## See Also

- junos-mpls-advanced
- junos-l3vpn
- junos-routing-fundamentals

## References

- RFC 7761 — PIM Sparse Mode (PIMv2, Revised)
- RFC 4607 — Source-Specific Multicast for IP
- RFC 5015 — Bidirectional PIM
- RFC 3376 — IGMPv3
- RFC 6513 — Multicast in MPLS/BGP IP VPNs
- RFC 6514 — BGP Encodings and Procedures for Multicast in MPLS/BGP IP VPNs
- RFC 6388 — LDP Extensions for P2MP and MP2MP LSPs
- RFC 4610 — Anycast-RP Using PIM
- Juniper TechLibrary: Multicast Protocols Feature Guide
- Juniper TechLibrary: Multicast VPN Configuration Guide
