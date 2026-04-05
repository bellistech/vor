# JunOS Multicast (JNCIE-SP)

Comprehensive multicast cheatsheet covering PIM-SM/SSM/DM/BiDir, RP configuration (static, auto-RP, BSR, anycast), IGMP/MLD, IGMP snooping, multicast in routing instances, mLDP, multicast VPN (draft-rosen, NG-mVPN), policies, and verification on Juniper platforms.

## PIM Sparse Mode (PIM-SM)

### Basic PIM-SM Configuration

```
protocols {
    pim {
        rp {
            static {
                address 10.0.0.10;
            }
        }
        interface ge-0/0/0.0 {
            mode sparse;
        }
        interface ge-0/0/1.0 {
            mode sparse;
        }
        interface lo0.0 {
            mode sparse;
        }
    }
}
```

### PIM-SM with Interface-Specific Options

```
protocols {
    pim {
        interface ge-0/0/0.0 {
            mode sparse;
            hello-interval 30;
            priority 100;          /* DR election priority */
        }
    }
}
```

## PIM Source-Specific Multicast (SSM)

### PIM-SSM Configuration

```
protocols {
    pim {
        interface ge-0/0/0.0 {
            mode sparse-dense;     /* or sparse with SSM range */
        }
    }
}
routing-options {
    multicast {
        ssm-groups {
            232.0.0.0/8;           /* default SSM range */
        }
    }
}
/* IGMPv3 required for SSM */
protocols {
    igmp {
        interface ge-0/0/2.0 {
            version 3;
        }
    }
}
```

## PIM Dense Mode (PIM-DM)

### Basic PIM-DM

```
protocols {
    pim {
        dense-groups {
            224.0.0.0/4;
        }
        interface ge-0/0/0.0 {
            mode dense;
        }
        interface ge-0/0/1.0 {
            mode dense;
        }
    }
}
```

## PIM Bidirectional (BiDir)

### BiDir PIM Configuration

```
protocols {
    pim {
        rp {
            static {
                address 10.0.0.10 {
                    bidirectional;
                }
            }
        }
        interface ge-0/0/0.0 {
            mode sparse;
        }
    }
}
```

## Rendezvous Point (RP) Configuration

### Static RP

```
protocols {
    pim {
        rp {
            static {
                address 10.0.0.10;
                /* RP for specific group range */
                address 10.0.0.11 {
                    group-ranges {
                        239.0.0.0/8;
                    }
                }
            }
        }
    }
}
```

### Auto-RP

```
/* RP Candidate */
protocols {
    pim {
        rp {
            auto-rp {
                announce {
                    scope 16;
                    group-ranges {
                        224.0.0.0/4;
                    }
                    holdtime 60;
                }
            }
        }
    }
}

/* Auto-RP Mapping Agent */
protocols {
    pim {
        rp {
            auto-rp {
                mapping {
                    scope 16;
                    holdtime 60;
                }
            }
        }
    }
}

/* Auto-RP Listener */
protocols {
    pim {
        rp {
            auto-rp {
                discovery;
            }
        }
    }
}
```

### Bootstrap Router (BSR)

```
/* BSR Candidate */
protocols {
    pim {
        rp {
            bootstrap-import BSR-POLICY;
            rp-candidate {
                interface lo0.0;
                priority 200;
                group-ranges {
                    224.0.0.0/4;
                }
                holdtime 150;
            }
            bsr-candidate {
                interface lo0.0;
                priority 200;
                hash-mask-length 30;
            }
        }
    }
}
```

### Anycast RP

Multiple RPs share the same IP address for redundancy and load distribution.

```
/* RP1 — configure loopback with anycast address */
interfaces {
    lo0 {
        unit 0 {
            family inet {
                address 10.0.0.1/32;        /* unique address */
                address 10.0.0.100/32;       /* anycast RP address */
            }
        }
    }
}
protocols {
    pim {
        rp {
            static {
                address 10.0.0.100;          /* anycast RP */
            }
            local {
                address 10.0.0.100;          /* declare self as RP */
            }
        }
    }
}

/* MSDP between anycast RP peers for source synchronization */
protocols {
    msdp {
        peer 10.0.0.2 {                     /* other RP's unique address */
            local-address 10.0.0.1;
            active-source-limit {
                maximum 5000;
            }
        }
    }
}

/* Alternative: Anycast RP with PIM (RFC 4610) — no MSDP needed */
protocols {
    pim {
        rp {
            static {
                address 10.0.0.100;
            }
            local {
                address 10.0.0.100;
            }
            rp-set {
                address 10.0.0.1;            /* RP1 unique address */
                address 10.0.0.2;            /* RP2 unique address */
            }
        }
    }
}
```

## IGMP Configuration

### IGMPv2/v3

```
protocols {
    igmp {
        interface ge-0/0/2.0 {
            version 3;                /* IGMPv3 for SSM */
            immediate-leave;          /* fast leave for point-to-point */
        }
        interface ge-0/0/3.0 {
            version 2;
            static {
                group 239.1.1.1;      /* static IGMP join */
                group 239.1.1.2 {
                    source 10.10.10.1; /* SSM static join */
                }
            }
        }
    }
}
```

### IGMP Query Configuration

```
protocols {
    igmp {
        query-interval 125;
        query-response-interval 10;
        query-last-member-interval 1;
        robust-count 2;
    }
}
```

## MLD (Multicast Listener Discovery — IPv6)

```
protocols {
    mld {
        interface ge-0/0/2.0 {
            version 2;
            immediate-leave;
        }
    }
}
```

## IGMP Snooping

### Basic IGMP Snooping

```
protocols {
    igmp-snooping {
        vlan VLAN100 {
            interface ge-0/0/0.0;
            interface ge-0/0/1.0;
            interface ge-0/0/2.0 {
                multicast-router-interface;   /* static mrouter port */
            }
        }
    }
}
/* OR in EVPN/VPLS context */
routing-instances {
    EVPN-VS {
        bridge-domains {
            bd-100 {
                protocols {
                    igmp-snooping;
                }
            }
        }
    }
}
```

## Multicast in Routing Instances

### PIM in VRF

```
routing-instances {
    VRF-CUSTA {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vrf-table-label;
        protocols {
            pim {
                rp {
                    static {
                        address 192.168.1.1;
                    }
                }
                interface ge-0/0/2.100 {
                    mode sparse;
                }
            }
        }
    }
}
```

### IGMP in VRF

```
routing-instances {
    VRF-CUSTA {
        protocols {
            igmp {
                interface ge-0/0/2.100 {
                    version 3;
                }
            }
        }
    }
}
```

## mLDP (Multipoint LDP)

### mLDP Configuration

```
protocols {
    ldp {
        p2mp;                    /* enable mLDP */
        interface ge-0/0/0.0;
        interface ge-0/0/1.0;
    }
}
routing-options {
    multicast {
        interface-type ldp;
    }
}
```

## Multicast VPN (mVPN)

### Draft-Rosen (PIM/GRE mVPN)

Classic mVPN using PIM in the provider core with GRE tunnels.

```
routing-instances {
    VRF-CUSTA {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vrf-table-label;
        protocols {
            pim {
                rp {
                    static {
                        address 192.168.1.1;    /* customer RP */
                    }
                }
                interface ge-0/0/2.100 {
                    mode sparse;
                }
                vpn-group-address 239.1.1.1;    /* provider multicast group */
            }
        }
        multicast {
            /* Enable multicast for the VRF */
        }
    }
}
/* Provider PIM must be running in default instance */
protocols {
    pim {
        rp {
            static {
                address 10.0.0.10;              /* provider RP */
            }
        }
        interface ge-0/0/0.0 {
            mode sparse;
        }
    }
}
```

### NG-mVPN (Next-Generation mVPN / BGP-Based)

Uses BGP for mVPN signaling instead of PIM in the core.

```
routing-instances {
    VRF-CUSTA {
        instance-type vrf;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:100;
        vrf-target target:65000:100;
        vrf-table-label;
        provider-tunnel {
            ingress-replication {
                label-switched-path {
                    /* Use RSVP-TE or mLDP for P-tunnel */
                }
            }
            /* OR */
            ldp-p2mp;
            /* OR */
            rsvp-te {
                label-switched-path-template {
                    default-template;
                }
            }
            selective {
                group 239.0.0.0/8 {
                    source 0.0.0.0/0 {
                        ldp-p2mp;
                    }
                }
            }
        }
        protocols {
            pim {
                rp {
                    static {
                        address 192.168.1.1;
                    }
                }
                interface ge-0/0/2.100 {
                    mode sparse;
                }
            }
            mvpn;
        }
    }
}
/* BGP family for mVPN */
protocols {
    bgp {
        group IBGP-PE {
            family inet-mvpn {
                signaling;
            }
        }
    }
}
```

### NG-mVPN Provider Tunnel Types

| Tunnel Type          | Configuration              | Use Case                    |
|---------------------|----------------------------|-----------------------------|
| Ingress Replication | `ingress-replication`      | Small scale, no PIM in core |
| mLDP P2MP           | `ldp-p2mp`                 | LDP-based core, scalable    |
| RSVP-TE P2MP        | `rsvp-te`                  | TE-enabled core             |
| PIM/GRE             | `pim-asm` or `pim-ssm`     | Legacy/draft-rosen compat   |

## PIM Join/Prune Policies

### Filter Multicast Groups

```
protocols {
    pim {
        import PIM-JOIN-FILTER;
    }
}
policy-options {
    policy-statement PIM-JOIN-FILTER {
        term ALLOW-SSM {
            from {
                route-filter 232.0.0.0/8 orlonger;
            }
            then accept;
        }
        term ALLOW-SPECIFIC {
            from {
                route-filter 239.1.0.0/16 orlonger;
            }
            then accept;
        }
        term DENY-REST {
            then reject;
        }
    }
}
```

### Per-Interface Join Filter

```
protocols {
    pim {
        interface ge-0/0/2.0 {
            mode sparse;
            accept-join-always-from policy JOIN-POLICY;
        }
    }
}
```

## RPF Policy

### Modify RPF Interface

```
routing-options {
    multicast {
        rpf-check-policy RPF-OVERRIDE;
    }
}
policy-options {
    policy-statement RPF-OVERRIDE {
        term REDIRECT {
            from {
                source-address-filter 10.10.0.0/16 orlonger;
            }
            then {
                rpf-check-nexthop ge-0/0/1.0;
            }
        }
    }
}
```

### Multicast RPF Table (inet.2)

```
/* Import multicast RPF routes into inet.2 */
routing-options {
    rib inet.2 {
        static {
            route 10.10.10.0/24 next-hop 10.0.0.1;
        }
    }
}
/* OR use rib-group to copy routes from inet.0 to inet.2 */
routing-options {
    interface-routes {
        rib-group inet MCAST-RIB;
    }
    rib-groups {
        MCAST-RIB {
            import-rib [ inet.0 inet.2 ];
        }
    }
}
```

## Multicast Flow Monitoring

```
protocols {
    pim {
        interface ge-0/0/0.0 {
            mode sparse;
        }
    }
}
forwarding-options {
    sampling {
        input {
            rate 1000;
            run-length 0;
        }
        family inet {
            output {
                flow-server 10.0.0.50 {
                    port 9995;
                    version 9;
                }
            }
        }
    }
}
```

## Verification Commands

### PIM Verification

```bash
# PIM neighbors
show pim neighbors

# PIM interfaces
show pim interfaces

# PIM join state (multicast routes)
show pim join extensive

# PIM source state
show pim source

# PIM RP status
show pim rps

# PIM statistics
show pim statistics

# PIM bootstrap status
show pim bootstrap

# PIM neighbor detail
show pim neighbors detail
```

### Multicast Route Table

```bash
# Multicast route table
show multicast route

# Multicast route extensive
show multicast route extensive

# Multicast route for specific group
show multicast route group 239.1.1.1

# Multicast forwarding cache
show multicast route active

# Multicast next-hops
show multicast next-hops

# Multicast RPF
show multicast rpf 10.10.10.1

# inet.1 (multicast forwarding cache)
show route table inet.1

# inet.2 (RPF table)
show route table inet.2
```

### IGMP Verification

```bash
# IGMP groups
show igmp group

# IGMP group detail
show igmp group detail

# IGMP interface
show igmp interface

# IGMP statistics
show igmp statistics

# IGMP snooping membership
show igmp-snooping membership

# IGMP snooping interfaces
show igmp-snooping interfaces
```

### mVPN Verification

```bash
# mVPN neighbor
show mvpn neighbor

# mVPN instance
show mvpn instance

# Provider tunnel
show mvpn provider-tunnel

# mVPN C-multicast routes
show mvpn c-multicast

# BGP inet-mvpn routes
show route table bgp.mvpn.0

# VRF multicast routes
show multicast route instance VRF-CUSTA
```

### mLDP Verification

```bash
# mLDP database
show ldp p2mp path

# mLDP session
show ldp session

# mLDP FEC
show ldp p2mp fec
```

### Multicast Monitoring

```bash
# Monitor PIM events
monitor start pim

# Multicast traffic statistics
show pim statistics

# Interface multicast counters
show interfaces ge-0/0/0 extensive | match "Multicast"

# Multicast scope policy hits
show firewall filter PIM-JOIN-FILTER
```

## Quick Reference: Multicast Address Ranges

| Range              | Purpose                                  |
|--------------------|------------------------------------------|
| 224.0.0.0/24       | Local link multicast (not forwarded)     |
| 224.0.1.0/24       | Internetwork control                     |
| 232.0.0.0/8        | Source-specific multicast (SSM)          |
| 233.0.0.0/8        | GLOP addressing                          |
| 239.0.0.0/8        | Administratively scoped (private)        |

## See Also

- junos-mpls-advanced
- junos-l3vpn
- junos-routing-fundamentals

## References

- RFC 7761 — PIM Sparse Mode (PIMv2, Revised)
- RFC 4607 — Source-Specific Multicast for IP
- RFC 5015 — Bidirectional PIM
- RFC 5332 — MPLS Multicast Encapsulations
- RFC 6513 — Multicast in MPLS/BGP IP VPNs
- RFC 6514 — BGP Encodings and Procedures for mVPN
- RFC 6388 — Label Distribution Protocol Extensions for P2MP and MP2MP LSPs
- Juniper TechLibrary: Multicast Configuration Guide
- Juniper TechLibrary: Multicast VPN Configuration Guide
