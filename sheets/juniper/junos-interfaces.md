# JunOS Interfaces (JNCIA-Junos Exam Prep)

Quick reference for Juniper interface naming, types, configuration, and monitoring.

## Interface Naming Convention

```
type-FPC/PIC/Port[.unit]

FPC  = Flexible PIC Concentrator (line card / slot number)
PIC  = Physical Interface Card (sub-slot within the FPC)
Port = Port number on the PIC

Examples:
  ge-0/0/0      Gigabit Ethernet, FPC 0, PIC 0, Port 0
  xe-0/0/1      10G Ethernet, FPC 0, PIC 0, Port 1
  et-0/0/0      40G/100G Ethernet, FPC 0, PIC 0, Port 0
  ge-0/0/0.0    Logical unit 0 on ge-0/0/0
  ge-0/0/0.100  Logical unit 100 (e.g., VLAN 100 subinterface)
```

## Interface Types

| Prefix | Type                              | Notes                              |
|--------|-----------------------------------|------------------------------------|
| `ge`   | Gigabit Ethernet (1G)             | Most common access/uplink          |
| `xe`   | 10-Gigabit Ethernet               | 10G SFP+                           |
| `et`   | 40G / 100G Ethernet               | QSFP+ / QSFP28                    |
| `em`   | Management Ethernet               | Out-of-band mgmt (some platforms)  |
| `fxp`  | Internal / Management             | fxp0 = OOB mgmt, fxp1 = internal  |
| `lo`   | Loopback                          | Always up, used for router-id      |
| `ae`   | Aggregated Ethernet (LAG)         | Bundle of physical links via LACP  |
| `irb`  | Integrated Routing and Bridging   | L3 interface for VLANs (like SVI)  |
| `vlan` | VLAN interface                    | Legacy L3 VLAN (older EX)          |
| `st0`  | Secure Tunnel                     | IPsec VPN tunnel interface         |
| `gr`   | GRE Tunnel                        | Generic Routing Encapsulation      |

## Physical vs Logical Interfaces

```
Physical interface:  ge-0/0/0        (the hardware port itself)
Logical interface:   ge-0/0/0.0      (unit 0 -- where you assign addresses)

Physical properties:  speed, duplex, mtu, link-mode
Logical properties:   family, address, vlan-id, description
```

- Every physical interface needs at least one unit (logical interface) to pass traffic.
- Unit 0 is the default; use other unit numbers for VLAN subinterfaces.

## Units (Subinterfaces)

```
set interfaces ge-0/0/0 vlan-tagging
set interfaces ge-0/0/0 unit 10 vlan-id 10
set interfaces ge-0/0/0 unit 10 family inet address 10.0.10.1/24
set interfaces ge-0/0/0 unit 20 vlan-id 20
set interfaces ge-0/0/0 unit 20 family inet address 10.0.20.1/24
set interfaces ge-0/0/0 unit 20 family inet6 address 2001:db8:20::1/64
```

Each unit can carry its own family and address configuration independently.

## Family Types

| Family               | Protocol / Use                        |
|----------------------|---------------------------------------|
| `inet`               | IPv4                                  |
| `inet6`              | IPv6                                  |
| `mpls`               | MPLS label switching                  |
| `iso`                | IS-IS CLNS                            |
| `ethernet-switching` | Layer 2 switching (EX/QFX)            |
| `bridge`             | Bridge domain (MX)                    |
| `ccc`                | Circuit cross-connect (L2 VPN)        |

## Address Configuration

```
# Basic IPv4
set interfaces ge-0/0/0 unit 0 family inet address 10.0.1.1/24

# IPv4 + IPv6 dual stack
set interfaces ge-0/0/0 unit 0 family inet address 10.0.1.1/24
set interfaces ge-0/0/0 unit 0 family inet6 address 2001:db8:1::1/64

# Secondary address
set interfaces ge-0/0/0 unit 0 family inet address 10.0.1.2/24

# DHCP client
set interfaces ge-0/0/0 unit 0 family inet dhcp
```

## Loopback Interface (lo0)

```
set interfaces lo0 unit 0 family inet address 192.168.1.1/32
set interfaces lo0 unit 0 family inet6 address 2001:db8::1/128

# Typical uses:
#   - Router ID for OSPF/BGP
#   - Management access (always reachable if any path exists)
#   - Source address for routing protocols
#   - Filter/policy anchor point
```

- lo0 is always up -- not tied to any physical link state.
- Best practice: assign a /32 (IPv4) or /128 (IPv6).

## Management Interfaces

```
# Out-of-band management (platform-dependent):
#   fxp0  -- MX, PTX, older SRX
#   em0   -- vSRX, some EX
#   me0   -- newer EX, QFX
#   re0:mgmt-0 / re0:mgmt-1 -- dual RE platforms

set interfaces fxp0 unit 0 family inet address 192.168.100.1/24

# fxp0 lives in a separate routing instance by default on some platforms
```

## Interface Properties

```
# Description
set interfaces ge-0/0/0 description "Uplink to core-sw1"

# MTU (physical-level)
set interfaces ge-0/0/0 mtu 9192

# Speed and duplex
set interfaces ge-0/0/0 speed 1g
set interfaces ge-0/0/0 link-mode full-duplex

# Disable / enable
set interfaces ge-0/0/0 disable
delete interfaces ge-0/0/0 disable

# Gigether options
set interfaces ge-0/0/0 gigether-options no-auto-negotiation
```

## Aggregated Interfaces (ae / LAG)

```
# 1. Set chassis aggregated-devices count
set chassis aggregated-devices ethernet device-count 5

# 2. Create ae interface
set interfaces ae0 description "LAG to distribution switch"
set interfaces ae0 aggregated-ether-options lacp active
set interfaces ae0 aggregated-ether-options minimum-links 2
set interfaces ae0 unit 0 family inet address 10.0.0.1/30

# 3. Assign member links
set interfaces ge-0/0/1 gigether-options 802.3ad ae0
set interfaces ge-0/0/2 gigether-options 802.3ad ae0
set interfaces ge-0/0/3 gigether-options 802.3ad ae0
```

- `minimum-links` defines the threshold below which the ae goes down.
- LACP modes: `active` (preferred) or `passive`.

## IRB Interfaces (Integrated Routing and Bridging)

```
# IRB = Layer 3 gateway for a VLAN (equivalent to Cisco SVI)

set interfaces irb unit 100 family inet address 10.0.100.1/24
set interfaces irb unit 100 description "Gateway for VLAN 100"

# Associate with a VLAN
set vlans VLAN100 vlan-id 100
set vlans VLAN100 l3-interface irb.100
```

- The IRB unit number typically matches the VLAN ID (convention, not required).

## Interface Monitoring

```
# Quick status of all interfaces (up/down, addresses)
show interfaces terse

# Detailed single-interface info (counters, errors, speed, MAC)
show interfaces ge-0/0/0 extensive

# Optics / transceiver diagnostics (SFP power, temperature, voltage)
show interfaces diagnostics optics ge-0/0/0

# Traffic statistics
show interfaces ge-0/0/0 statistics

# Interface summary by status
show interfaces terse | match "up|down"

# Clear interface counters
clear interfaces statistics ge-0/0/0
```

## Configuration Groups for Interfaces

```
# Define a group with standard interface settings
set groups STANDARD-INTF interfaces <*> mtu 9192
set groups STANDARD-INTF interfaces <*> unit <*> family inet mtu 1500

# Apply globally
set apply-groups STANDARD-INTF

# Apply to a specific interface
set interfaces ge-0/0/0 apply-groups STANDARD-INTF

# Verify what the group contributes
show interfaces ge-0/0/0 | display inheritance
```

- `<*>` is a wildcard that matches any interface or unit name.

## Interface Ranges

```
# Define a named range
set interfaces interface-range ACCESS-PORTS member "ge-0/0/[0-23]"
set interfaces interface-range ACCESS-PORTS description "Access ports"
set interfaces interface-range ACCESS-PORTS unit 0 family ethernet-switching
set interfaces interface-range ACCESS-PORTS unit 0 family ethernet-switching vlan members VLAN100

# Ranges accept:
#   member "ge-0/0/[0-23]"     bracket range
#   member-range ge-0/0/0 to ge-0/0/23
```

- Changes to an interface-range apply to all members simultaneously.

## Tips

- Always commit with `commit check` first to validate interface config before applying.
- Use `show interfaces terse` as the first troubleshooting step -- it shows link state and protocol state at a glance.
- `show configuration interfaces | display set` converts hierarchical config to set commands for easy copy-paste.
- On EX switches, access ports use `family ethernet-switching`; routed ports use `family inet`.
- If an interface shows "Administratively down," look for `disable` in the config.
- IRB unit numbers do not have to match VLAN IDs, but keeping them aligned is standard practice.
- Remember: physical properties (speed, mtu, duplex) go on the physical interface; addresses and families go on the unit.

## See Also

- JunOS Routing Protocols
- JunOS Firewall Filters
- JunOS VLANs and Switching
- JunOS Class of Service (CoS)

## References

- Juniper TechLibrary: Interfaces Configuration Guide
- Juniper JNCIA-Junos Study Guide (JN0-105)
- Juniper Day One: Configuring Junos Basics
- RFC 3635 -- Definitions of Managed Objects for the Ethernet-like Interface Types
