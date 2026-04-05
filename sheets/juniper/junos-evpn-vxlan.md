# JunOS EVPN-VXLAN (JNCIE-SP)

Comprehensive EVPN-VXLAN cheatsheet covering IP fabric underlay, EVPN overlay with route types 1-5, VXLAN encapsulation, symmetric/asymmetric IRB, anycast gateway, ESI multihoming, BUM handling, ERB vs CRB architectures, and verification on Juniper platforms.

## IP Fabric Underlay

### eBGP Underlay (IP Fabric)

```
/* Spine */
routing-options {
    router-id 10.255.0.1;
    autonomous-system 65000;
    forwarding-table {
        export ECMP-POLICY;
    }
}
protocols {
    bgp {
        group UNDERLAY {
            type external;
            export EXPORT-LOOPBACK;
            multipath multiple-as;
            neighbor 10.1.0.1 {
                peer-as 65001;    /* leaf-1 */
            }
            neighbor 10.1.0.3 {
                peer-as 65002;    /* leaf-2 */
            }
        }
    }
}
policy-options {
    policy-statement EXPORT-LOOPBACK {
        term LOOPBACK {
            from {
                protocol direct;
                interface lo0.0;
            }
            then accept;
        }
    }
    policy-statement ECMP-POLICY {
        then {
            load-balance per-packet;
        }
    }
}
```

### OSPF Underlay

```
protocols {
    ospf {
        area 0.0.0.0 {
            interface lo0.0 {
                passive;
            }
            interface ge-0/0/0.0 {
                interface-type p2p;
            }
            interface ge-0/0/1.0 {
                interface-type p2p;
            }
        }
    }
}
```

### IS-IS Underlay

```
protocols {
    isis {
        interface lo0.0 {
            passive;
        }
        interface ge-0/0/0.0 {
            point-to-point;
        }
        interface ge-0/0/1.0 {
            point-to-point;
        }
    }
}
```

## EVPN Overlay — BGP Configuration

### iBGP Overlay with Route Reflector

```
/* Leaf (VTEP) */
protocols {
    bgp {
        group EVPN-OVERLAY {
            type internal;
            local-address 10.255.1.1;
            family evpn {
                signaling;
            }
            neighbor 10.255.0.1;   /* spine RR */
            neighbor 10.255.0.2;   /* spine RR */
        }
    }
}

/* Spine (Route Reflector) */
protocols {
    bgp {
        group EVPN-OVERLAY-CLIENTS {
            type internal;
            local-address 10.255.0.1;
            cluster 10.255.0.1;
            family evpn {
                signaling;
            }
            neighbor 10.255.1.1;   /* leaf-1 */
            neighbor 10.255.1.2;   /* leaf-2 */
            neighbor 10.255.1.3;   /* leaf-3 */
        }
    }
}
```

### eBGP Overlay

```
protocols {
    bgp {
        group EVPN-OVERLAY {
            type external;
            multihop {
                ttl 2;
            }
            local-address 10.255.1.1;
            family evpn {
                signaling;
            }
            peer-as 65000;
            neighbor 10.255.0.1;
            neighbor 10.255.0.2;
        }
    }
}
```

## EVPN Route Types

### Route Type Summary

| Type | Name                    | Purpose                                       |
|------|-------------------------|-----------------------------------------------|
| 1    | Ethernet Auto-Discovery | ESI multihoming, fast convergence, aliasing    |
| 2    | MAC/IP Advertisement    | MAC and MAC+IP binding advertisement           |
| 3    | Inclusive Multicast      | BUM traffic handling, PMSI tunnel binding      |
| 4    | Ethernet Segment        | DF election for ESI multihoming                |
| 5    | IP Prefix               | L3 routing (inter-VNI, external routes)        |

## EVPN Instance Configuration

### Basic EVPN-VXLAN Instance (MAC-VRF)

```
routing-instances {
    VNI-1000 {
        instance-type mac-vrf;
        protocols {
            evpn {
                encapsulation vxlan;
                default-gateway do-not-advertise;
            }
        }
        vtep-source-interface lo0.0;
        service-type vlan-based;
        interface ge-0/0/2.100;
        route-distinguisher 10.255.1.1:1000;
        vrf-target target:65000:1000;
        bridge-domains {
            bd-1000 {
                vlan-id 100;
                interface ge-0/0/2.100;
                vxlan {
                    vni 1000;
                }
            }
        }
    }
}
```

### EVPN with Virtual Switch

```
routing-instances {
    EVPN-VS {
        instance-type virtual-switch;
        interface ge-0/0/2.100;
        interface ge-0/0/2.200;
        route-distinguisher 10.255.1.1:1;
        vrf-target target:65000:1;
        vtep-source-interface lo0.0;
        protocols {
            evpn {
                encapsulation vxlan;
                multicast-mode ingress-replication;
                extended-vlan-list [ 100 200 ];
            }
        }
        bridge-domains {
            bd-100 {
                vlan-id 100;
                interface ge-0/0/2.100;
                vxlan {
                    vni 1000;
                }
            }
            bd-200 {
                vlan-id 200;
                interface ge-0/0/2.200;
                vxlan {
                    vni 2000;
                }
            }
        }
    }
}
```

## VXLAN Encapsulation

### VTEP Source Interface

```
routing-instances {
    VNI-1000 {
        vtep-source-interface lo0.0;
    }
}
/* lo0.0 must be reachable via the underlay */
interfaces {
    lo0 {
        unit 0 {
            family inet {
                address 10.255.1.1/32;
            }
        }
    }
}
```

### VXLAN Header Structure

```
Outer Ethernet | Outer IP | Outer UDP (dst 4789) | VXLAN Header | Inner Ethernet | Payload
                                                    |
                                                    +-- VNI (24-bit): 1000
```

## IRB — Integrated Routing and Bridging

### Asymmetric IRB

Traffic is routed at the ingress VTEP and bridged at the egress VTEP. Both source and destination VNIs must exist on both VTEPs.

```
/* Both leaf switches need both VNIs configured */
interfaces {
    irb {
        unit 100 {
            family inet {
                address 192.168.100.1/24;
            }
            mac 00:00:5e:00:01:01;   /* anycast MAC */
        }
        unit 200 {
            family inet {
                address 192.168.200.1/24;
            }
            mac 00:00:5e:00:01:01;
        }
    }
}
routing-instances {
    EVPN-VS {
        bridge-domains {
            bd-100 {
                vlan-id 100;
                routing-interface irb.100;
                vxlan { vni 1000; }
            }
            bd-200 {
                vlan-id 200;
                routing-interface irb.200;
                vxlan { vni 2000; }
            }
        }
    }
}
```

### Symmetric IRB

Traffic is routed at both ingress and egress VTEPs using a shared L3 VNI. Only the local VNIs need to exist on each VTEP.

```
/* L3 VNI for inter-VNI routing */
routing-instances {
    VRF-TENANT-A {
        instance-type vrf;
        interface irb.100;
        interface irb.200;
        interface lo0.1;
        route-distinguisher 10.255.1.1:5000;
        vrf-target target:65000:5000;
        vrf-table-label;
    }
}
/* Map L3 VNI to VRF */
routing-instances {
    EVPN-VS {
        bridge-domains {
            bd-l3vni {
                vlan-id 999;
                routing-interface irb.999;
                vxlan { vni 5000; }       /* L3 VNI */
            }
        }
    }
}
interfaces {
    irb {
        unit 999 {
            family inet {
                address 10.99.99.1/32;    /* dummy, not used for traffic */
            }
        }
    }
}
```

## Distributed Anycast Gateway

All leaf switches share the same gateway IP and MAC for each subnet.

```
interfaces {
    irb {
        unit 100 {
            family inet {
                address 192.168.100.1/24;
            }
            virtual-gateway-accept-data;
            virtual-gateway-address 192.168.100.1;
            virtual-gateway-v4-mac 00:00:5e:00:01:01;
        }
    }
}
/* OR using identical static MAC across all leaves */
interfaces {
    irb {
        unit 100 {
            family inet {
                address 192.168.100.1/24;
            }
            mac 00:00:5e:00:01:01;   /* same MAC on all leaves */
        }
    }
}
```

## ESI Multihoming (All-Active)

### Ethernet Segment Configuration

```
interfaces {
    ae0 {
        esi {
            00:11:22:33:44:55:66:77:88:99;
            all-active;
        }
        aggregated-ether-options {
            lacp {
                active;
                system-id 00:00:00:00:00:01;
                /* Same LACP system-id on both PEs */
            }
        }
        flexible-vlan-tagging;
        encapsulation flexible-ethernet-services;
        unit 100 {
            encapsulation vlan-bridge;
            vlan-id 100;
        }
    }
}
```

### ESI-LAG on Both PEs

```
/* PE1 */
interfaces {
    ae0 {
        esi {
            00:01:02:03:04:05:06:07:08:09;
            all-active;
        }
        aggregated-ether-options {
            lacp {
                active;
                system-id 00:00:00:00:00:42;
            }
        }
    }
}

/* PE2 — same ESI and LACP system-id */
interfaces {
    ae0 {
        esi {
            00:01:02:03:04:05:06:07:08:09;
            all-active;
        }
        aggregated-ether-options {
            lacp {
                active;
                system-id 00:00:00:00:00:42;
            }
        }
    }
}
```

## BUM Handling

### Ingress Replication

```
routing-instances {
    EVPN-VS {
        protocols {
            evpn {
                multicast-mode ingress-replication;
            }
        }
    }
}
```

### Assisted Replication

For scaling BUM in large fabrics, designated replicators handle BUM forwarding.

```
routing-instances {
    EVPN-VS {
        protocols {
            evpn {
                assisted-replication {
                    replicator;           /* this node is a replicator */
                    /* OR */
                    leaf;                 /* this node delegates replication */
                }
            }
        }
    }
}
```

## EVPN Type 5 — IP Prefix Routes

### Inter-VNI Routing with Type 5

```
routing-instances {
    VRF-TENANT-A {
        instance-type vrf;
        interface irb.100;
        interface irb.200;
        interface lo0.1;
        route-distinguisher 10.255.1.1:5000;
        vrf-target target:65000:5000;
        vrf-table-label;
        protocols {
            evpn {
                ip-prefix-routes {
                    advertise direct-nexthop;
                    encapsulation vxlan;
                    vni 5000;
                }
            }
        }
    }
}
```

### External Route Advertisement via Type 5

```
routing-instances {
    VRF-TENANT-A {
        protocols {
            evpn {
                ip-prefix-routes {
                    advertise direct-nexthop;
                    encapsulation vxlan;
                    vni 5000;
                }
            }
            bgp {
                group EXTERNAL {
                    type external;
                    peer-as 64512;
                    neighbor 10.100.0.2 {
                        family inet {
                            unicast;
                        }
                    }
                }
            }
        }
    }
}
```

## ERB vs CRB Architecture

### Edge-Routed Bridging (ERB)

All leaves perform routing. Distributed L3 gateway on every leaf.

```
/* Every leaf has IRB interfaces and VRF */
routing-instances {
    VRF-TENANT-A {
        instance-type vrf;
        interface irb.100;
        interface irb.200;
        route-distinguisher 10.255.1.1:5000;
        vrf-target target:65000:5000;
        vrf-table-label;
    }
}
```

### Centrally-Routed Bridging (CRB)

Only designated gateway leaves (or spines) perform routing. Other leaves do L2 only.

```
/* L2-only leaf — no VRF, no IRB */
routing-instances {
    EVPN-VS {
        instance-type virtual-switch;
        interface ge-0/0/2.100;
        route-distinguisher 10.255.1.3:1;
        vrf-target target:65000:1;
        vtep-source-interface lo0.0;
        protocols {
            evpn {
                encapsulation vxlan;
            }
        }
        bridge-domains {
            bd-100 {
                vlan-id 100;
                interface ge-0/0/2.100;
                vxlan { vni 1000; }
            }
        }
    }
}

/* Gateway leaf (or spine) — has IRB and VRF */
routing-instances {
    EVPN-VS {
        bridge-domains {
            bd-100 {
                vlan-id 100;
                routing-interface irb.100;
                vxlan { vni 1000; }
            }
        }
    }
    VRF-TENANT-A {
        instance-type vrf;
        interface irb.100;
        route-distinguisher 10.255.1.10:5000;
        vrf-target target:65000:5000;
        vrf-table-label;
    }
}
```

## Verification Commands

### EVPN Database

```bash
# EVPN database (all route types)
show evpn database

# Specific instance
show evpn database instance EVPN-VS

# Specific route type
show evpn database state-type 2     # MAC/IP routes
show evpn database state-type 5     # IP prefix routes

# EVPN route detail
show route table default-switch.evpn.0

# EVPN routes per type
show route table default-switch.evpn.0 match-prefix "1:*"   # Type 1
show route table default-switch.evpn.0 match-prefix "2:*"   # Type 2
show route table default-switch.evpn.0 match-prefix "3:*"   # Type 3
show route table default-switch.evpn.0 match-prefix "5:*"   # Type 5
```

### VXLAN Verification

```bash
# VXLAN tunnel endpoints (VTEPs)
show ethernet-switching vxlan-tunnel-end-point remote

# VXLAN source interface
show ethernet-switching vxlan-tunnel-end-point source

# VNI to VLAN mapping
show ethernet-switching table vlan-id 100

# VXLAN statistics
show ethernet-switching vxlan-tunnel-end-point remote summary
```

### ESI Multihoming

```bash
# Ethernet segment status
show evpn instance esi

# ESI detail
show evpn instance EVPN-VS extensive | match "ESI"

# Designated forwarder
show evpn instance EVPN-VS designated-forwarder

# All-active status
show interfaces ae0 | match "esi"
```

### MAC/IP Table

```bash
# MAC table
show ethernet-switching table

# MAC table for specific VLAN
show ethernet-switching table vlan-id 100

# MAC/IP bindings (from EVPN Type 2)
show evpn mac-table

# ARP table in VRF
show arp interface irb.100 instance VRF-TENANT-A
```

### IRB and L3 Routing

```bash
# IRB interface status
show interfaces irb

# VRF routing table
show route table VRF-TENANT-A.inet.0

# Type 5 IP prefix routes
show route table VRF-TENANT-A.evpn.0 match-prefix "5:*"

# Inter-VNI connectivity
ping routing-instance VRF-TENANT-A 192.168.200.10 source 192.168.100.1
```

### BGP EVPN

```bash
# BGP EVPN summary
show bgp summary group EVPN-OVERLAY

# EVPN received routes
show route receive-protocol bgp 10.255.0.1 table default-switch.evpn.0

# EVPN advertised routes
show route advertising-protocol bgp 10.255.0.1 table default-switch.evpn.0
```

## See Also

- junos-mpls-advanced
- junos-l3vpn
- junos-l2vpn
- junos-routing-fundamentals

## References

- RFC 7432 — BGP MPLS-Based Ethernet VPN
- RFC 8365 — A Framework for Ethernet-Tree (E-Tree) Service over EVPN
- RFC 9135 — Integrated Routing and Bridging in EVPN
- RFC 7348 — VXLAN: A Framework for Overlaying Virtualized Layer 2 Networks over Layer 3 Networks
- RFC 8584 — Framework for Ethernet VPN Designated Forwarder Election Extensibility
- Juniper TechLibrary: EVPN-VXLAN Configuration Guide
- Juniper Validated Design: Data Center EVPN-VXLAN Fabric
