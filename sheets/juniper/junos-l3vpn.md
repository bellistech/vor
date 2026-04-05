# JunOS L3VPN (JNCIE-SP)

Comprehensive Layer 3 VPN cheatsheet covering VRF configuration, PE-CE routing, MP-BGP VPNv4/VPNv6, inter-AS options, hub-and-spoke, carrier-of-carriers, 6VPE, and verification on Juniper platforms.

## Basic L3VPN Configuration

### VRF Routing Instance

```
routing-instances {
    CUSTOMER-A {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vrf-table-label;
    }
}
```

### Separate Import/Export Route Targets

```
routing-instances {
    CUSTOMER-A {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target {
            import target:65000:100;
            import target:65000:999;   /* shared services RT */
            export target:65000:100;
        }
        vrf-table-label;
    }
}
```

### Auto Route Target

```
routing-instances {
    CUSTOMER-A {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target {
            auto;
            /* JunOS derives RT from RD automatically */
        }
        vrf-table-label;
    }
}
```

## PE-CE Routing Protocols

### Static PE-CE

```
routing-instances {
    CUSTOMER-A {
        routing-options {
            static {
                route 192.168.10.0/24 next-hop 10.1.1.2;
            }
        }
    }
}
```

### OSPF PE-CE

```
routing-instances {
    CUSTOMER-A {
        protocols {
            ospf {
                domain-id 0.0.0.100;   /* for multi-PE OSPF loop prevention */
                area 0.0.0.0 {
                    interface ge-0/0/2.100;
                }
            }
        }
    }
}
```

### BGP PE-CE

```
routing-instances {
    CUSTOMER-A {
        protocols {
            bgp {
                group CE-PEERS {
                    type external;
                    peer-as 64512;
                    neighbor 10.1.1.2 {
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

### IS-IS PE-CE

```
routing-instances {
    CUSTOMER-A {
        protocols {
            isis {
                interface ge-0/0/2.100 {
                    point-to-point;
                }
            }
        }
    }
}
```

### RIP PE-CE

```
routing-instances {
    CUSTOMER-A {
        protocols {
            rip {
                group CE-GROUP {
                    neighbor ge-0/0/2.100;
                }
            }
        }
    }
}
```

## MP-BGP VPNv4/VPNv6 Configuration

### PE-PE iBGP with VPNv4

```
protocols {
    bgp {
        group IBGP-PE {
            type internal;
            local-address 10.0.0.1;
            family inet-vpn {
                unicast;
            }
            neighbor 10.0.0.2;
            neighbor 10.0.0.3;
        }
    }
}
```

### VPNv6 (IPv6 VPN)

```
protocols {
    bgp {
        group IBGP-PE {
            type internal;
            local-address 10.0.0.1;
            family inet6-vpn {
                unicast;
            }
            neighbor 10.0.0.2;
        }
    }
}
routing-instances {
    CUSTOMER-A {
        interface ge-0/0/2.100;
        instance-type vrf;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vrf-table-label;
        protocols {
            bgp {
                group CE-V6 {
                    type external;
                    peer-as 64512;
                    neighbor 2001:db8::2 {
                        family inet6 {
                            unicast;
                        }
                    }
                }
            }
        }
    }
}
```

### Route Reflector for VPNv4

```
protocols {
    bgp {
        group IBGP-RR-CLIENTS {
            type internal;
            local-address 10.0.0.10;
            cluster 10.0.0.10;
            family inet-vpn {
                unicast;
            }
            neighbor 10.0.0.1;
            neighbor 10.0.0.2;
            neighbor 10.0.0.3;
        }
    }
}
```

## VRF Table Policies

### Export Policy (Control What Gets Advertised)

```
policy-options {
    policy-statement VRF-EXPORT-CUSTA {
        term DIRECT {
            from protocol direct;
            then {
                community add RT-CUSTA;
                accept;
            }
        }
        term STATIC {
            from protocol static;
            then {
                community add RT-CUSTA;
                accept;
            }
        }
        term DEFAULT {
            then reject;
        }
    }
    community RT-CUSTA members target:65000:100;
}
routing-instances {
    CUSTOMER-A {
        vrf-export VRF-EXPORT-CUSTA;
    }
}
```

### Import Policy (Control What Gets Accepted)

```
policy-options {
    policy-statement VRF-IMPORT-CUSTA {
        term ACCEPT-OWN {
            from community RT-CUSTA;
            then accept;
        }
        term ACCEPT-SHARED {
            from community RT-SHARED;
            then accept;
        }
        term DEFAULT {
            then reject;
        }
    }
    community RT-CUSTA members target:65000:100;
    community RT-SHARED members target:65000:999;
}
routing-instances {
    CUSTOMER-A {
        vrf-import VRF-IMPORT-CUSTA;
    }
}
```

## Inter-AS L3VPN

### Option A (Back-to-Back VRF)

Each ASBR has a VRF per VPN. PE-CE-like peering between ASBRs.

```
/* ASBR1 in AS 65000 */
routing-instances {
    CUSTOMER-A {
        instance-type vrf;
        interface ge-0/0/5.100;   /* inter-AS link */
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vrf-table-label;
        protocols {
            bgp {
                group INTER-AS {
                    type external;
                    peer-as 65001;
                    neighbor 172.16.0.2 {
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

### Option B (ASBR-to-ASBR MP-eBGP VPNv4)

ASBRs exchange VPNv4 routes directly. No per-VPN VRF needed on ASBR.

```
/* ASBR1 */
protocols {
    bgp {
        group INTER-AS-VPNV4 {
            type external;
            multihop;
            local-address 172.16.0.1;
            peer-as 65001;
            neighbor 172.16.0.2 {
                family inet-vpn {
                    unicast;
                }
            }
        }
    }
}
/* ASBR must re-advertise VPNv4 routes — next-hop-self */
protocols {
    bgp {
        group IBGP-PE {
            type internal;
            local-address 10.0.0.1;
            family inet-vpn {
                unicast;
            }
            neighbor 10.0.0.5 {
                export NHS;
            }
        }
    }
}
policy-options {
    policy-statement NHS {
        then {
            next-hop self;
        }
    }
}
```

### Option C (Multi-hop MP-eBGP Between RRs)

RRs in different ASes exchange VPNv4. ASBRs only exchange labeled IPv4 (for next-hop resolution).

```
/* RR in AS 65000 */
protocols {
    bgp {
        group INTER-AS-RR {
            type external;
            multihop {
                ttl 255;
                no-nexthop-change;
            }
            peer-as 65001;
            local-address 10.0.0.10;
            neighbor 10.1.0.10 {
                family inet-vpn {
                    unicast;
                }
            }
        }
    }
}

/* ASBR exchanges labeled unicast for next-hop reachability */
protocols {
    bgp {
        group ASBR-LABELED {
            type external;
            peer-as 65001;
            neighbor 172.16.0.2 {
                family inet {
                    labeled-unicast;
                }
            }
        }
    }
}
```

## Hub-and-Spoke L3VPN

Spoke sites can only communicate via the hub site.

```
/* Hub PE */
routing-instances {
    HUB-CUSTA {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target {
            import target:65000:101;   /* import spoke RT */
            export target:65000:100;   /* export hub RT */
        }
        vrf-table-label;
    }
}

/* Spoke PE */
routing-instances {
    SPOKE-CUSTA {
        instance-type vrf;
        interface ge-0/0/3.200;
        route-distinguisher 10.0.0.2:100;
        vrf-target {
            import target:65000:100;   /* import hub RT only */
            export target:65000:101;   /* export spoke RT */
        }
        vrf-table-label;
        routing-options {
            static {
                route 0.0.0.0/0 next-table HUB-CUSTA.inet.0;
                /* or default via hub PE-CE */
            }
        }
    }
}
```

## Shared Services VRF

Allow VPN customers to access shared resources (DNS, NTP, etc.).

```
routing-instances {
    SHARED-SERVICES {
        instance-type vrf;
        interface ge-0/0/4.0;
        route-distinguisher 10.0.0.1:999;
        vrf-target {
            import target:65000:100;   /* import customer A */
            import target:65000:200;   /* import customer B */
            export target:65000:999;   /* export shared RT */
        }
        vrf-table-label;
    }
}
/* Customer VRFs import target:65000:999 to receive shared routes */
```

## Carrier-of-Carriers (CoC)

Service provider provides MPLS VPN transport to a carrier customer.

```
/* Carrier PE — VRF for carrier customer */
routing-instances {
    CARRIER-CUSTOMER {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:500;
        vrf-target target:65000:500;
        vrf-table-label;
        protocols {
            bgp {
                group CARRIER-CE {
                    type external;
                    peer-as 64600;
                    neighbor 10.2.2.2 {
                        family inet {
                            labeled-unicast;  /* labeled routes for CoC */
                        }
                    }
                }
            }
            mpls {
                interface ge-0/0/2.100;
            }
        }
    }
}
```

## 6VPE (IPv6 VPN over IPv4 MPLS Core)

Carry IPv6 VPN traffic over an IPv4-only MPLS backbone.

```
protocols {
    bgp {
        group IBGP-PE {
            type internal;
            local-address 10.0.0.1;
            family inet6-vpn {
                unicast;
            }
            neighbor 10.0.0.2;
        }
    }
}
routing-instances {
    CUST-A-V6 {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:600;
        vrf-target target:65000:600;
        vrf-table-label;
        protocols {
            bgp {
                group CE-V6 {
                    type external;
                    peer-as 64512;
                    neighbor 2001:db8:1::2 {
                        family inet6 {
                            unicast;
                        }
                    }
                }
            }
        }
    }
}
```

## Verification Commands

### VRF and Routing Instance

```bash
# List all routing instances
show route instance

# Show VRF detail
show route instance CUSTOMER-A detail

# Show VRF routing table
show route table CUSTOMER-A.inet.0

# Show VRF routing table with detail
show route table CUSTOMER-A.inet.0 detail

# Show active routes only
show route table CUSTOMER-A.inet.0 active-path

# VRF interface associations
show route instance CUSTOMER-A | match interface
```

### MP-BGP VPNv4/VPNv6

```bash
# BGP VPNv4 summary
show bgp summary group IBGP-PE

# VPNv4 received routes
show route receive-protocol bgp 10.0.0.2 table bgp.l3vpn.0

# VPNv4 advertised routes
show route advertising-protocol bgp 10.0.0.2 table bgp.l3vpn.0

# BGP VPNv4 route detail
show route table bgp.l3vpn.0 detail

# VPNv6 routes
show route table bgp.l3vpn-inet6.0
```

### Route Target and RD

```bash
# Show routes with specific community
show route table bgp.l3vpn.0 community target:65000:100

# Show route-distinguisher info
show route table bgp.l3vpn.0 match-prefix "10.0.0.1:100:*"

# Verify RT import/export on VRF
show route instance CUSTOMER-A detail | match "target"
```

### PE-CE Protocol Verification

```bash
# OSPF in VRF
show ospf neighbor instance CUSTOMER-A
show ospf route instance CUSTOMER-A

# BGP in VRF
show bgp summary instance CUSTOMER-A
show bgp neighbor instance CUSTOMER-A

# Static routes in VRF
show route table CUSTOMER-A.inet.0 protocol static
```

### Label Verification

```bash
# VRF label binding
show route table mpls.0 label 299776

# VPN label in VPNv4 route
show route table bgp.l3vpn.0 detail | match "Label"

# Label forwarding table
show route forwarding-table family mpls
```

### Inter-AS Verification

```bash
# Option B: ASBR VPNv4 session
show bgp summary group INTER-AS-VPNV4

# Option C: labeled unicast
show route table inet.3

# Option C: RR VPNv4
show bgp summary group INTER-AS-RR
```

## See Also

- junos-mpls-advanced
- junos-l2vpn
- junos-evpn-vxlan
- junos-routing-fundamentals

## References

- RFC 4364 — BGP/MPLS IP Virtual Private Networks (VPNs)
- RFC 4659 — BGP-MPLS IP VPN Extension for IPv6 VPN (6VPE)
- RFC 4684 — Constrained Route Distribution for BGP/MPLS VPN
- RFC 3107 — Carrying Label Information in BGP-4
- Juniper TechLibrary: Layer 3 VPN Configuration Guide
- Juniper Day One: Configuring Layer 3 VPNs
