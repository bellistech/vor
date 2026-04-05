# JunOS L2VPN (JNCIE-SP)

Comprehensive Layer 2 VPN cheatsheet covering l2circuit pseudowires, VPLS (LDP and BGP signaled), H-VPLS, pseudowire redundancy, MAC management, storm control, VPLS with IRB, and EVPN-VPLS interop on Juniper platforms.

## L2circuit (Pseudowire)

### Basic Point-to-Point L2circuit

```
interfaces {
    ge-0/0/2 {
        encapsulation ethernet-ccc;
        unit 0 {
            family ccc;
        }
    }
}
protocols {
    l2circuit {
        neighbor 10.0.0.2 {        /* remote PE loopback */
            interface ge-0/0/2.0 {
                virtual-circuit-id 100;
            }
        }
    }
}
```

### L2circuit with VLAN Encapsulation

```
interfaces {
    ge-0/0/2 {
        flexible-vlan-tagging;
        encapsulation flexible-ethernet-services;
        unit 100 {
            encapsulation vlan-ccc;
            vlan-id 100;
        }
    }
}
protocols {
    l2circuit {
        neighbor 10.0.0.2 {
            interface ge-0/0/2.100 {
                virtual-circuit-id 100;
                encapsulation-type ethernet-vlan;
            }
        }
    }
}
```

### Local Cross-Connect (Local Switching)

No MPLS transport needed. Connects two local interfaces.

```
protocols {
    l2circuit {
        local-switching {
            interface ge-0/0/2.100 {
                end-interface {
                    interface ge-0/0/3.200;
                }
            }
        }
    }
}
```

### Pseudowire Redundancy

Active/standby pseudowire for resiliency.

```
protocols {
    l2circuit {
        neighbor 10.0.0.2 {
            interface ge-0/0/2.100 {
                virtual-circuit-id 100;
                backup-neighbor 10.0.0.3 {
                    virtual-circuit-id 100;
                    standby;
                }
            }
        }
    }
}
```

### Encapsulation Types

| Encapsulation Type       | Interface Config             | Use Case                  |
|--------------------------|------------------------------|---------------------------|
| `ethernet-ccc`           | Port-based L2circuit         | Entire port as pseudowire |
| `vlan-ccc`               | VLAN-based L2circuit         | Single VLAN as pseudowire |
| `ethernet`               | VPLS ethernet                | VPLS member port          |
| `flexible-ethernet-services` | Multi-service interface  | Mixed L2/L3 on same port  |

## VPLS — LDP Signaling (Kompella/Martini)

### Basic VPLS with LDP Signaling

```
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ge-0/0/2.100;
        protocols {
            vpls {
                vpls-id 100;
                neighbor 10.0.0.2;
                neighbor 10.0.0.3;
                encapsulation-type ethernet;
            }
        }
    }
}
```

### VPLS Interface Configuration

```
interfaces {
    ge-0/0/2 {
        flexible-vlan-tagging;
        encapsulation flexible-ethernet-services;
        unit 100 {
            encapsulation vlan-vpls;
            vlan-id 100;
            family vpls;
        }
    }
}
```

### VPLS with Site ID (Full Mesh Control)

```
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ge-0/0/2.100;
        protocols {
            vpls {
                vpls-id 100;
                neighbor 10.0.0.2;
                neighbor 10.0.0.3;
                site CE1 {
                    site-identifier 1;
                    interface ge-0/0/2.100;
                }
            }
        }
    }
}
```

## VPLS — BGP Signaling (Kompella)

### BGP-Signaled VPLS

```
routing-instances {
    VPLS-CUSTB {
        instance-type vpls;
        interface ge-0/0/2.200;
        route-distinguisher 10.0.0.1:200;
        vrf-target target:65000:200;
        protocols {
            vpls {
                site CE1 {
                    site-identifier 1;
                    interface ge-0/0/2.200;
                }
            }
        }
    }
}
protocols {
    bgp {
        group IBGP-PE {
            family l2vpn {
                signaling;
            }
        }
    }
}
```

### BGP VPLS with Site Range (Multi-Homing)

```
routing-instances {
    VPLS-CUSTB {
        instance-type vpls;
        interface ge-0/0/2.200;
        route-distinguisher 10.0.0.1:200;
        vrf-target target:65000:200;
        protocols {
            vpls {
                site CE1 {
                    site-identifier 1;
                    site-range 10;       /* maximum number of sites */
                    interface ge-0/0/2.200;
                }
            }
        }
    }
}
```

## H-VPLS (Hierarchical VPLS)

### Spoke PE (MTU) to Hub PE

Spoke PE connects to hub PE via pseudowire, reducing full-mesh requirement.

```
/* Spoke PE (MTU device) */
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ge-0/0/2.100;
        protocols {
            vpls {
                vpls-id 100;
                neighbor 10.0.0.1 {
                    /* Connect to hub PE */
                }
            }
        }
    }
}

/* Hub PE */
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ge-0/0/2.100;
        protocols {
            vpls {
                vpls-id 100;
                neighbor 10.0.0.2;     /* full mesh with other hub PEs */
                neighbor 10.0.0.3;
                mesh-group CORE {
                    neighbor 10.0.0.2;
                    neighbor 10.0.0.3;
                }
                mesh-group SPOKE {
                    neighbor 10.0.0.10;  /* spoke PE */
                    neighbor 10.0.0.11;  /* spoke PE */
                }
            }
        }
    }
}
```

## MAC Learning and Management

### MAC Limiting

```
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ge-0/0/2.100;
        bridge-domains {
            bd100 {
                interface ge-0/0/2.100 {
                    mac-limit 1000 {
                        action drop;     /* drop | log | shutdown | none */
                    }
                }
            }
        }
    }
}
/* Or at VPLS instance level */
routing-instances {
    VPLS-CUSTA {
        forwarding-options {
            family vpls {
                flood {
                    input STORM-CONTROL;
                }
            }
        }
        protocols {
            vpls {
                mac-table-size {
                    1000;
                }
                interface-mac-limit {
                    500;
                }
            }
        }
    }
}
```

### MAC Aging

```
routing-instances {
    VPLS-CUSTA {
        protocols {
            vpls {
                mac-table-aging-time 600;   /* seconds */
                no-mac-learning;            /* disable learning (optional) */
            }
        }
    }
}
```

## Storm Control

```
interfaces {
    ge-0/0/2 {
        unit 100 {
            family vpls {
                filter {
                    input STORM-CONTROL;
                }
            }
        }
    }
}
firewall {
    family vpls {
        filter STORM-CONTROL {
            term BROADCAST {
                from {
                    traffic-type broadcast;
                }
                then policer STORM-POLICER;
            }
            term MULTICAST {
                from {
                    traffic-type multicast;
                }
                then policer STORM-POLICER;
            }
            term UNKNOWN-UNICAST {
                from {
                    traffic-type unknown-unicast;
                }
                then policer STORM-POLICER;
            }
            term DEFAULT {
                then accept;
            }
        }
    }
    policer STORM-POLICER {
        if-exceeding {
            bandwidth-limit 10m;
            burst-size-limit 1m;
        }
        then discard;
    }
}
```

## VPLS Multihoming

### BGP Multihoming (Designated Forwarder Election)

```
/* PE1 — multihomed to same CE */
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ae0.100;
        route-distinguisher 10.0.0.1:200;
        vrf-target target:65000:200;
        protocols {
            vpls {
                site CE-MULTI {
                    site-identifier 1;
                    site-preference 100;   /* higher = preferred DF */
                    interface ae0.100;
                    multi-homing;
                }
            }
        }
    }
}
```

## VPLS with IRB (Integrated Routing and Bridging)

Provide L3 gateway within a VPLS domain.

```
routing-instances {
    VPLS-CUSTA {
        instance-type vpls;
        interface ge-0/0/2.100;
        routing-interface irb.100;    /* L3 gateway */
        protocols {
            vpls {
                vpls-id 100;
                neighbor 10.0.0.2;
            }
        }
    }
}
interfaces {
    irb {
        unit 100 {
            family inet {
                address 192.168.100.1/24;
            }
        }
    }
}
/* Route IRB into a VRF for L3VPN integration */
routing-instances {
    L3VPN-CUSTA {
        instance-type vrf;
        interface irb.100;
        route-distinguisher 10.0.0.1:101;
        vrf-target target:65000:101;
        vrf-table-label;
    }
}
```

## EVPN-VPLS Interop

Migration from VPLS to EVPN using the same instance.

```
routing-instances {
    EVPN-VPLS-CUSTA {
        instance-type virtual-switch;
        interface ge-0/0/2.100;
        route-distinguisher 10.0.0.1:300;
        vrf-target target:65000:300;
        bridge-domains {
            bd100 {
                vlan-id 100;
                interface ge-0/0/2.100;
            }
        }
        protocols {
            evpn {
                encapsulation mpls;
                extended-vlan-list 100;
            }
        }
    }
}
```

## Verification Commands

### L2circuit Verification

```bash
# L2circuit connections
show l2circuit connections

# L2circuit connections detail
show l2circuit connections detail

# Specific neighbor
show l2circuit connections neighbor 10.0.0.2

# L2circuit local switching
show l2circuit connections local-switching

# Connection status
show l2circuit connections interface ge-0/0/2.100
```

### VPLS Verification

```bash
# VPLS connections
show vpls connections

# VPLS connections detail
show vpls connections instance VPLS-CUSTA

# VPLS neighbors
show vpls connections instance VPLS-CUSTA | match "Neighbor"

# VPLS flood info
show vpls flood instance VPLS-CUSTA

# VPLS statistics
show vpls statistics instance VPLS-CUSTA
```

### MAC Table

```bash
# MAC table for VPLS instance
show vpls mac-table instance VPLS-CUSTA

# MAC table count
show vpls mac-table instance VPLS-CUSTA count

# Specific MAC lookup
show vpls mac-table instance VPLS-CUSTA address 00:11:22:33:44:55

# MAC table for bridge domain
show bridge mac-table

# MAC learning statistics
show vpls mac-table instance VPLS-CUSTA extensive
```

### BGP L2VPN

```bash
# BGP L2VPN summary
show bgp summary group IBGP-PE | match l2vpn

# L2VPN routes
show route table bgp.l2vpn.0

# L2VPN route detail
show route table bgp.l2vpn.0 detail
```

### MPLS Label Verification

```bash
# Labels for l2circuit
show l2circuit connections detail | match "label"

# Labels for VPLS
show vpls connections instance VPLS-CUSTA detail | match "label"

# MPLS forwarding for L2 labels
show route table mpls.0 label <label-value>
```

### Pseudowire Status

```bash
# PW status
show l2circuit connections detail | match "Status"

# Backup PW
show l2circuit connections detail | match -A 2 "backup"
```

## See Also

- junos-mpls-advanced
- junos-l3vpn
- junos-evpn-vxlan
- junos-routing-fundamentals

## References

- RFC 4761 — VPLS Using BGP for Auto-Discovery and Signaling
- RFC 4762 — VPLS Using LDP Signaling
- RFC 6624 — Layer 2 VPNs over Tunnels
- RFC 4447 — Pseudowire Setup and Maintenance Using LDP
- RFC 4448 — Encapsulation Methods for Transport of Ethernet over MPLS
- Juniper TechLibrary: Layer 2 VPN Configuration Guide
- Juniper TechLibrary: VPLS Configuration Guide
