# Advanced JunOS MPLS (JNCIE-SP)

Comprehensive MPLS cheatsheet covering LDP, RSVP-TE, FRR, SR-MPLS, LSP hierarchy, admin groups, OAM, and label operations on Juniper platforms.

## LDP Configuration

### Basic LDP

```
protocols {
    ldp {
        interface ge-0/0/0.0;
        interface ge-0/0/1.0;
        interface lo0.0;
    }
}
protocols {
    mpls {
        interface ge-0/0/0.0;
        interface ge-0/0/1.0;
    }
}
```

### Targeted LDP (tLDP)

Used for remote LDP sessions (e.g., L2VPN pseudowires, VPLS).

```
protocols {
    ldp {
        targeted-hello {
            hello-interval 5;
            hold-time 15;
        }
        /* Explicit targeted neighbor */
        neighbor 10.0.0.5;
    }
}
```

### LDP Session Protection

Maintains LDP session during link failures using targeted hello.

```
protocols {
    ldp {
        session-protection timeout 120;
    }
}
```

### LDP-IGP Synchronization

Prevents traffic blackhole during LDP convergence.

```
protocols {
    ospf {
        area 0.0.0.0 {
            interface ge-0/0/0.0 {
                ldp-synchronization;
            }
        }
    }
}
/* Or globally */
protocols {
    ldp {
        igp-synchronization holddown-interval 30;
    }
}

/* IS-IS variant */
protocols {
    isis {
        interface ge-0/0/0.0 {
            ldp-synchronization;
        }
    }
}
```

### LDP Track IGP Metric

```
protocols {
    ldp {
        track-igp-metric;
    }
}
```

## RSVP-TE Configuration

### Basic RSVP-TE LSP

```
protocols {
    rsvp {
        interface ge-0/0/0.0;
        interface ge-0/0/1.0;
    }
    mpls {
        label-switched-path to-PE2 {
            to 10.0.0.2;
        }
    }
}
```

### Explicit Path (Strict and Loose Hops)

```
mpls {
    path via-core1 {
        10.1.1.1 strict;   /* must traverse this exact hop */
        10.2.2.2 loose;    /* any path to reach this hop */
        10.3.3.3 strict;
    }
    label-switched-path to-PE2 {
        to 10.0.0.2;
        primary via-core1;
    }
}
```

### Bandwidth Reservation

```
mpls {
    label-switched-path to-PE2 {
        to 10.0.0.2;
        bandwidth 100m;   /* 100 Mbps reservation */
    }
}
protocols {
    rsvp {
        interface ge-0/0/0.0 {
            bandwidth 1g;              /* subscribable bandwidth */
            subscription 80;           /* allow 80% subscription */
        }
    }
}
```

### Auto-Bandwidth

Dynamically adjusts bandwidth reservation based on measured traffic.

```
mpls {
    label-switched-path to-PE2 {
        to 10.0.0.2;
        auto-bandwidth {
            adjust-interval 300;         /* measure every 5 min */
            adjust-threshold 10;         /* re-signal if >10% change */
            minimum-bandwidth 10m;
            maximum-bandwidth 500m;
            adjust-threshold-overflow-limit 3;
        }
    }
}
```

### Make-Before-Break (MBB)

Enabled by default in JunOS for RSVP-TE re-optimization. Old LSP torn down only after new LSP is established.

```
mpls {
    label-switched-path to-PE2 {
        to 10.0.0.2;
        optimize-timer 300;   /* re-optimize every 5 min */
        /* MBB is default behavior during re-signaling */
    }
}
```

### Adaptive RSVP-TE

Allows in-place modification of RSVP-TE LSPs without full re-signaling.

```
mpls {
    label-switched-path to-PE2 {
        to 10.0.0.2;
        adaptive;
    }
}
```

### TE Metric

```
protocols {
    mpls {
        label-switched-path to-PE2 {
            to 10.0.0.2;
            metric-based-computation;
        }
    }
}

/* Set TE metric on interface */
protocols {
    mpls {
        interface ge-0/0/0.0 {
            admin-group [ gold silver ];
        }
    }
}
interfaces {
    ge-0/0/0 {
        unit 0 {
            family mpls;
        }
    }
}
protocols {
    ospf {
        area 0.0.0.0 {
            interface ge-0/0/0.0 {
                te-metric 100;
            }
        }
    }
}
```

### Admin Groups (Link Coloring)

```
mpls {
    admin-groups {
        gold 0;
        silver 1;
        bronze 2;
        exclude-group 3;
    }
    interface ge-0/0/0.0 {
        admin-group [ gold silver ];
    }
    label-switched-path to-PE2 {
        to 10.0.0.2;
        admin-group {
            include-any [ gold silver ];
            exclude [ exclude-group ];
        }
    }
}
```

### PCEP (Path Computation Element Protocol)

```
protocols {
    pcep {
        pce pce1 {
            local-address 10.0.0.1;
            destination-ipv4-address 10.0.0.100;
            destination-port 4189;
            pce-type active stateful;
            lsp-provisioning;
            spring-capability;
        }
    }
}
```

## Fast Reroute (FRR)

### Facility Backup (Bypass LSP)

Protects using a pre-established bypass tunnel around the protected link/node.

```
protocols {
    mpls {
        label-switched-path to-PE2 {
            to 10.0.0.2;
            fast-reroute;              /* enable node-link protection */
        }
    }
    rsvp {
        interface ge-0/0/0.0 {
            link-protection;           /* facility backup */
        }
    }
}
```

### Node-Link Protection

```
protocols {
    rsvp {
        interface ge-0/0/0.0 {
            link-protection {
                node-link-protection;  /* protect against node failure too */
            }
        }
    }
}
```

### One-to-One Backup (Detour LSP)

Each protected LSP gets its own detour path.

```
protocols {
    mpls {
        label-switched-path to-PE2 {
            to 10.0.0.2;
            fast-reroute {
                detour;
            }
        }
    }
}
```

## LSP Hierarchy

Nest LSPs inside other LSPs (hierarchical LSPs).

```
mpls {
    label-switched-path inner-lsp {
        to 10.0.0.5;
        lsp-attributes {
            /* inner LSP is tunneled through outer */
        }
    }
    label-switched-path outer-lsp {
        to 10.0.0.3;
    }
}
```

## MPLS Label Operations

### Label Stack Operations

| Operation | Description                               | When Used                        |
|-----------|-------------------------------------------|----------------------------------|
| **Push**  | Add label to top of stack                 | Ingress LER                      |
| **Swap**  | Replace top label with new label          | Transit LSR                      |
| **Pop**   | Remove top label                          | Egress LER or penultimate hop    |

### Penultimate Hop Popping (PHP)

- Default behavior in JunOS: the penultimate router pops the label
- Egress LER advertises **implicit-null** (label 3) for PHP
- Reduces load on egress LER (no label lookup required)
- JunOS signals PHP by default; disable with `no-penultimate-hop-popping` if needed

```
mpls {
    label-switched-path to-PE2 {
        to 10.0.0.2;
        no-penultimate-hop-popping;   /* force ultimate hop popping */
    }
}
```

## SPRING / SR-MPLS on JunOS

### Segment Routing with OSPF

```
routing-options {
    source-packet-routing {
        srgb start-label 16000 index-range 8000;
    }
}
protocols {
    ospf {
        source-packet-routing {
            node-segment {
                ipv4-index 100;   /* SID = SRGB base + index = 16100 */
            }
        }
        area 0.0.0.0 {
            interface ge-0/0/0.0 {
                segment-routing;
            }
        }
    }
}
protocols {
    mpls {
        interface ge-0/0/0.0;
    }
}
```

### Segment Routing with IS-IS

```
routing-options {
    source-packet-routing {
        srgb start-label 16000 index-range 8000;
    }
}
protocols {
    isis {
        source-packet-routing {
            node-segment {
                ipv4-index 200;
            }
            srgb start-label 16000 index-range 8000;
        }
        interface ge-0/0/0.0 {
            point-to-point;
        }
    }
}
```

### SR-TE (Segment Routing Traffic Engineering)

```
routing-options {
    source-packet-routing {
        segment-list sl-via-P1 {
            hop1 label 16001;
            hop2 label 16002;
        }
        source-routing-path sr-te-to-PE2 {
            to 10.0.0.2;
            primary {
                sl-via-P1;
            }
        }
    }
}
```

### TI-LFA (Topology Independent Loop Free Alternate)

```
protocols {
    isis {
        backup-spf-options {
            use-post-convergence-lfa;
            use-source-packet-routing;
        }
    }
}
```

## MPLS OAM

### LSP Ping

```bash
# Basic LSP ping
ping mpls ldp 10.0.0.2/32

# RSVP-TE LSP ping
ping mpls rsvp to-PE2

# With count and size
ping mpls ldp 10.0.0.2/32 count 5 size 1400
```

### LSP Traceroute

```bash
# LDP LSP traceroute
traceroute mpls ldp 10.0.0.2/32

# RSVP-TE LSP traceroute
traceroute mpls rsvp to-PE2
```

### BFD for MPLS

```
protocols {
    rsvp {
        interface ge-0/0/0.0 {
            link-protection;
        }
    }
    mpls {
        label-switched-path to-PE2 {
            to 10.0.0.2;
            oam {
                bfd-liveness-detection {
                    minimum-interval 100;   /* 100ms */
                    multiplier 3;
                    failure-action {
                        make-before-break;
                    }
                }
            }
        }
    }
}
```

## Verification Commands

### LDP Verification

```bash
# LDP neighbor status
show ldp neighbor

# LDP session details
show ldp session detail

# LDP database (label bindings)
show ldp database

# LDP interface status
show ldp interface

# LDP traffic statistics
show ldp statistics

# LDP-IGP sync status
show ldp synchronization
```

### RSVP-TE Verification

```bash
# RSVP neighbor status
show rsvp neighbor

# RSVP interface status
show rsvp interface

# RSVP sessions (LSPs)
show rsvp session
show rsvp session detail
show rsvp session ingress
show rsvp session transit
show rsvp session egress

# RSVP session by name
show rsvp session name to-PE2
```

### MPLS / LSP Verification

```bash
# All LSPs
show mpls lsp
show mpls lsp detail
show mpls lsp extensive

# Ingress LSPs
show mpls lsp ingress

# Transit LSPs
show mpls lsp transit

# MPLS interface
show mpls interface

# MPLS label table
show route table mpls.0

# CSPF computation result
show mpls lsp name to-PE2 detail | match "Computed ERO"

# Auto-bandwidth
show mpls lsp name to-PE2 autobandwidth

# FRR status
show rsvp session detail | match "Protection"
show mpls lsp name to-PE2 detail | match "bypass"
```

### Segment Routing Verification

```bash
# SRGB allocation
show spring-traffic-engineering srgb

# Node SIDs
show ospf overview | match segment
show isis adjacency detail | match segment

# SR-TE paths
show spring-traffic-engineering lsp

# SR routes in routing table
show route table inet.3 protocol spring-te
```

### MPLS Monitoring

```bash
# Monitor RSVP events
monitor traffic interface ge-0/0/0.0 matching "rsvp"

# LSP statistics
show mpls lsp statistics

# RSVP error counters
show rsvp statistics

# Label operations per interface
show mpls lsp ingress statistics
```

## Quick Reference: Label Values

| Label Value | Meaning                       |
|-------------|-------------------------------|
| 0           | IPv4 Explicit Null            |
| 1           | Router Alert                  |
| 2           | IPv6 Explicit Null            |
| 3           | Implicit Null (PHP signal)    |
| 4-15        | Reserved                      |
| 16+         | Dynamic/static label range    |

## See Also

- junos-l3vpn
- junos-l2vpn
- junos-evpn-vxlan
- junos-routing-fundamentals

## References

- RFC 5036 — LDP Specification
- RFC 3209 — RSVP-TE Extensions
- RFC 4090 — FRR Extensions to RSVP-TE
- RFC 8402 — Segment Routing Architecture
- RFC 8667 — IS-IS Extensions for Segment Routing
- RFC 8665 — OSPF Extensions for Segment Routing
- Juniper TechLibrary: MPLS Configuration Guide
- Juniper Day One: MPLS for Enterprise Engineers
