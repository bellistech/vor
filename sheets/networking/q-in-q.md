# Q-in-Q (802.1ad VLAN Stacking)

IEEE 802.1ad provider bridging that encapsulates customer VLAN tags (C-tag) inside a service provider VLAN tag (S-tag), enabling VLAN space multiplication and transparent L2 transport across provider networks.

## Concepts

### Frame Structure

```
Normal 802.1Q frame:
  DA | SA | 0x8100 | C-VLAN | EtherType | Payload | FCS

Q-in-Q (802.1ad) frame:
  DA | SA | 0x88A8 | S-VLAN | 0x8100 | C-VLAN | EtherType | Payload | FCS
         ^                          ^
         S-tag (outer)              C-tag (inner, preserved)
```

- **S-tag (Service tag):** Outer VLAN tag added by the provider, TPID 0x88A8
- **C-tag (Customer tag):** Inner VLAN tag from the customer, TPID 0x8100
- **S-VLAN:** Provider VLAN ID (12-bit, 1-4094)
- **C-VLAN:** Customer VLAN ID (12-bit, 1-4094), transparent to the provider

### TPID Values

| TPID | Standard | Usage |
|------|----------|-------|
| 0x8100 | IEEE 802.1Q | Standard VLAN tag (C-tag) |
| 0x88A8 | IEEE 802.1ad | Standard S-tag |
| 0x9100 | Non-standard | Legacy Q-in-Q (pre-802.1ad, Cisco/others) |
| 0x9200 | Non-standard | Some vendor implementations |

### Port-Based vs Selective Q-in-Q

- **Port-based:** All traffic on the port gets the same S-tag, regardless of C-tag
- **Selective:** Different S-tags assigned based on C-tag value (per-VLAN mapping)

## Port-Based Q-in-Q

### IOS (Catalyst)

```
! Set the global dot1q tunnel TPID (optional, default 0x8100 on older platforms)
vlan dot1q tag native

! Customer-facing port — tunnel mode
interface GigabitEthernet1/0/1
 switchport mode dot1q-tunnel
 switchport access vlan 100
 ! All customer traffic gets S-VLAN 100
 l2protocol-tunnel cdp
 l2protocol-tunnel stp
 l2protocol-tunnel vtp
 no cdp enable

! Provider-facing port — trunk carrying S-VLANs
interface GigabitEthernet1/0/24
 switchport mode trunk
 switchport trunk allowed vlan 100-200
```

### NX-OS

```
! Customer-facing port
interface Ethernet1/1
 switchport
 switchport mode dot1q-tunnel
 switchport access vlan 100

! Provider-facing trunk
interface Ethernet1/48
 switchport
 switchport mode trunk
 switchport trunk allowed vlan 100-200
```

### JunOS

```
interfaces {
    ge-0/0/0 {
        description "Customer-facing";
        flexible-vlan-tagging;
        native-vlan-id 1;
        encapsulation flexible-ethernet-services;
        unit 0 {
            vlan-id 100;
            input-vlan-map {
                push;
            }
            output-vlan-map {
                pop;
            }
        }
    }
    ge-0/0/1 {
        description "Provider-facing trunk";
        flexible-vlan-tagging;
        encapsulation flexible-ethernet-services;
        unit 100 {
            vlan-id 100;
        }
    }
}
```

## Selective Q-in-Q

### IOS-XE (ASR/ISR)

```
! Map specific customer VLANs to different S-VLANs
interface GigabitEthernet0/0/0
 service instance 10 ethernet
  encapsulation dot1q 10
  rewrite ingress tag push dot1q 100 symmetric
  bridge-domain 100
 !
 service instance 20 ethernet
  encapsulation dot1q 20
  rewrite ingress tag push dot1q 200 symmetric
  bridge-domain 200
 !
 service instance 30 ethernet
  encapsulation dot1q 30-40
  rewrite ingress tag push dot1q 300 symmetric
  bridge-domain 300
```

### IOS-XR

```
interface GigabitEthernet0/0/0/0
 description Customer-facing
!

interface GigabitEthernet0/0/0/0.10
 encapsulation dot1q 10
 rewrite ingress tag push dot1q 100 symmetric
!

interface GigabitEthernet0/0/0/0.20
 encapsulation dot1q 20
 rewrite ingress tag push dot1q 200 symmetric
!
```

### JunOS Selective

```
interfaces {
    ge-0/0/0 {
        flexible-vlan-tagging;
        encapsulation flexible-ethernet-services;
        unit 10 {
            vlan-id 10;
            input-vlan-map {
                push;
                vlan-id 100;
            }
            output-vlan-map {
                pop;
            }
        }
        unit 20 {
            vlan-id 20;
            input-vlan-map {
                push;
                vlan-id 200;
            }
            output-vlan-map {
                pop;
            }
        }
    }
}
```

## L2PT (Layer 2 Protocol Tunneling)

### Concepts

- Customer L2 protocols (STP, CDP, VTP, LLDP) must be tunneled transparently across the provider
- Without L2PT, the provider drops or processes these BPDUs
- L2PT rewrites the destination MAC to a provider-specific multicast (01:00:0C:CD:CD:D0) for tunneling
- At the far-end customer port, the MAC is rewritten back to the original protocol MAC

### IOS L2PT Configuration

```
interface GigabitEthernet1/0/1
 switchport mode dot1q-tunnel
 switchport access vlan 100
 ! Tunnel specific protocols
 l2protocol-tunnel cdp
 l2protocol-tunnel stp
 l2protocol-tunnel vtp
 l2protocol-tunnel lldp
 ! Rate limit to prevent protocol storms (packets per second)
 l2protocol-tunnel shutdown-threshold 2000
 l2protocol-tunnel drop-threshold 1000
```

### IOS-XR L2PT

```
l2vpn
 bridge group BG1
  bridge-domain BD1
   interface GigabitEthernet0/0/0/0.10
    ! Protocol tunneling
   !
  !
 !
!
```

## MAC-Based Q-in-Q

- S-tag assigned based on source MAC address instead of port or C-VLAN
- Useful for shared-medium environments (e.g., wireless, shared Ethernet segment)
- Less common than port-based or selective Q-in-Q
- Implementation varies significantly by vendor

## MTU Considerations

### Frame Size Impact

```
Standard Ethernet frame:     1518 bytes (with FCS)
Single 802.1Q tag:           1522 bytes (+4 bytes)
Q-in-Q (S-tag + C-tag):     1526 bytes (+8 bytes from untagged)
                             1522 bytes (+4 bytes from single-tagged)
```

- Provider network must support at least **1526-byte frames** (or more for jumbo frames)
- If customer sends 1500-byte payload with C-tag: total frame = 1522 bytes + 4-byte S-tag = 1526 bytes
- Configure provider-facing interfaces with increased MTU:

```
! IOS
interface GigabitEthernet1/0/24
 system mtu 9216
 ! Or per-interface where supported
 mtu 9216

! IOS-XR
interface GigabitEthernet0/0/0/1
 mtu 9216

! JunOS
interfaces ge-0/0/1 {
    mtu 9216;
}
```

- **Baby giant:** Frames slightly larger than 1518 bytes (up to ~1600); some switches handle these automatically
- If MTU is not increased, Q-in-Q frames may be silently dropped

## Q-in-Q with MPLS

### Integration

- Q-in-Q can be used as the access technology feeding into MPLS L2VPN (pseudowire or VPLS)
- The S-tag identifies the service at the PE, and the PE pushes MPLS labels for transport
- C-tag is preserved end-to-end through the MPLS core

```
! IOS-XE: Q-in-Q into MPLS pseudowire
interface GigabitEthernet0/0/0
 service instance 100 ethernet
  encapsulation dot1q 100
  ! Match S-VLAN 100 (which already contains customer C-tags)
  xconnect 10.0.0.5 100 encapsulation mpls
```

## PBB (Provider Backbone Bridging) Comparison

| Feature | Q-in-Q (802.1ad) | PBB (802.1ah) |
|---------|-------------------|---------------|
| Encapsulation | S-tag + C-tag | B-tag + I-tag + C-tag |
| MAC learning | Customer MACs visible to provider | Customer MACs hidden (B-MAC only) |
| VLAN scale | 4094 S-VLANs x 4094 C-VLANs | 16M I-SIDs |
| Service identifier | S-VLAN (12-bit) | I-SID (24-bit) |
| Provider MAC table | Grows with customer MACs | Only backbone MACs |
| Complexity | Simple | More complex |
| Use case | Metro Ethernet access | Large-scale provider backbone |

## Verification

```
! IOS: show Q-in-Q tunnel ports
show dot1q-tunnel
show vlan dot1q tag native

! IOS: verify L2PT
show l2protocol-tunnel

! IOS-XE: show service instances
show ethernet service instance
show ethernet service instance id 10 interface GigabitEthernet0/0/0
show ethernet service instance detail

! IOS-XR
show l2vpn bridge-domain
show interfaces GigabitEthernet0/0/0/0.10

! NX-OS
show vlan dot1q-tunnel
show interface switchport

! JunOS
show interfaces ge-0/0/0 detail
show vlans
```

## Tips

- Always verify MTU end-to-end in the provider network; Q-in-Q adds 4 bytes to every frame.
- Use TPID 0x88A8 for standards-compliant deployments; 0x9100 is legacy and may cause interop issues.
- L2PT is essential when customers run STP; without it, customer STP topologies will break.
- Set L2PT rate limits to prevent a customer STP storm from overwhelming the provider control plane.
- Selective Q-in-Q is more flexible than port-based but requires more configuration; use port-based for simple deployments.
- In multi-vendor environments, verify TPID handling; some platforms default to 0x8100 for the outer tag.
- Q-in-Q does not provide MAC isolation; consider PBB (802.1ah) or EVPN-VXLAN for MAC scalability.
- Customer VLAN 1 (native) requires special attention; ensure native VLAN tagging behavior is consistent.
- When combining Q-in-Q with MPLS, the MPLS PE must be configured to match the S-tag, not the C-tag.

## See Also

- ethernet, vxlan, mpls, g8032-erp

## References

- [IEEE 802.1ad — Provider Bridges](https://standards.ieee.org/standard/802_1ad-2005.html)
- [IEEE 802.1Q — Bridges and Bridged Networks](https://standards.ieee.org/standard/802_1Q-2022.html)
- [IEEE 802.1ah — Provider Backbone Bridges](https://standards.ieee.org/standard/802_1ah-2008.html)
- [Cisco Q-in-Q Tunneling Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst3750x_3560x/software/release/15-0_1_se/configuration/guide/3750xcg/sw8021q.html)
- [Juniper Q-in-Q Tunneling Configuration](https://www.juniper.net/documentation/us/en/software/junos/multicast-l2/topics/topic-map/q-in-q-tunneling.html)
