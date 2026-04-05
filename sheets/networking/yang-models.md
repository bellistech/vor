# YANG Data Models (Network Data Modeling Language)

RFC 7950 data modeling language for NETCONF/RESTCONF/gNMI — defines configuration and state schemas with types, constraints, and structure for network device APIs.

## YANG Module Structure

### Basic module skeleton

```yang
module example-system {
    yang-version 1.1;                         // YANG 1.1 (RFC 7950)
    namespace "urn:example:system";           // unique XML namespace
    prefix sys;                               // short prefix for self-reference

    import ietf-yang-types {                  // import external types
        prefix yang;
    }

    import ietf-inet-types {                  // IP address types
        prefix inet;
    }

    organization "Example Corp";
    contact "support@example.com";
    description "System configuration model";

    revision 2026-01-15 {                     // version tracking
        description "Initial revision";
    }

    // ... containers, lists, leaves, etc.
}
```

### Submodule

```yang
submodule example-system-dns {
    yang-version 1.1;
    belongs-to example-system {               // parent module
        prefix sys;
    }

    import ietf-inet-types {
        prefix inet;
    }

    description "DNS configuration submodule";

    container dns {
        leaf domain-name {
            type string;
        }
        leaf-list search {
            type inet:domain-name;
            ordered-by user;
        }
        list server {
            key "address";
            leaf address {
                type inet:ip-address;
            }
        }
    }
}

// In main module:
// include example-system-dns;
```

## Core Language Constructs

### container — grouping node (no value)

```yang
container system {
    description "System configuration";
    container ntp {
        leaf enabled {
            type boolean;
            default true;
        }
    }
}
// XML: <system><ntp><enabled>true</enabled></ntp></system>
// JSON: {"system": {"ntp": {"enabled": true}}}
```

### list — keyed collection of entries

```yang
list interface {
    key "name";                               // unique key (one or more leaves)
    description "Network interface list";

    leaf name {
        type string;
    }
    leaf description {
        type string;
    }
    leaf enabled {
        type boolean;
        default true;
    }
    leaf mtu {
        type uint16 {
            range "68..9192";                 // value constraint
        }
        default 1500;
    }
}
// Multiple keys:
// list route { key "prefix next-hop"; ... }
```

### leaf — single typed value

```yang
leaf hostname {
    type string {
        length "1..64";                       // string length constraint
        pattern '[a-zA-Z][a-zA-Z0-9._-]*';   // regex constraint
    }
    mandatory true;                           // must be present
    description "Device hostname";
}

leaf idle-timeout {
    type uint32;
    units "seconds";                          // informational unit
    default 300;
}
```

### leaf-list — ordered collection of scalar values

```yang
leaf-list dns-server {
    type inet:ip-address;
    ordered-by user;                          // user-controlled order
    description "DNS server addresses";
}
// XML: <dns-server>8.8.8.8</dns-server>
//      <dns-server>8.8.4.4</dns-server>
// JSON: {"dns-server": ["8.8.8.8", "8.8.4.4"]}
```

### choice — mutually exclusive branches

```yang
choice address-type {
    mandatory true;
    case ipv4 {
        leaf ipv4-address {
            type inet:ipv4-address;
        }
        leaf ipv4-prefix-length {
            type uint8 { range "0..32"; }
        }
    }
    case ipv6 {
        leaf ipv6-address {
            type inet:ipv6-address;
        }
        leaf ipv6-prefix-length {
            type uint8 { range "0..128"; }
        }
    }
}
// Only one case can be active at a time
// Choice node itself does not appear in data
```

### grouping / uses — reusable schema fragments

```yang
grouping address-group {
    leaf address {
        type inet:ip-address;
    }
    leaf prefix-length {
        type uint8;
    }
}

container primary {
    uses address-group;                       // expands inline
}
container secondary {
    uses address-group;
}
// Both containers get address + prefix-length leaves
```

### uses with refine

```yang
container management {
    uses address-group {
        refine address {
            mandatory true;                   // override in this context
        }
        refine prefix-length {
            default 24;
        }
    }
}
```

### augment — extend another module's schema

```yang
// In module ietf-ip:
augment "/if:interfaces/if:interface" {       // add nodes to ietf-interfaces
    container ipv4 {
        leaf enabled {
            type boolean;
            default true;
        }
        list address {
            key "ip";
            leaf ip {
                type inet:ipv4-address;
            }
            leaf prefix-length {
                type uint8 { range "0..32"; }
                mandatory true;
            }
        }
    }
}
```

### deviation — vendor-specific schema modifications

```yang
// Vendor deviates from standard model
deviation "/if:interfaces/if:interface/if:mtu" {
    deviate replace {
        type uint16 {
            range "64..9216";                 // vendor-specific range
        }
    }
}

deviation "/if:interfaces/if:interface/if:link-up-down-trap-enable" {
    deviate not-supported;                    // feature not implemented
}

// Deviation types:
// deviate not-supported  — node does not exist on this platform
// deviate add            — add properties (default, must, etc.)
// deviate replace        — replace type, default, etc.
// deviate delete         — remove properties
```

### typedef — named type definition

```yang
typedef percent {
    type uint8 {
        range "0..100";
    }
    description "Percentage value";
}

typedef vlan-id {
    type uint16 {
        range "1..4094";
    }
}

leaf cpu-usage {
    type percent;
}
leaf native-vlan {
    type vlan-id;
}
```

### identity — extensible enumeration

```yang
identity transport-protocol {
    description "Base identity for transport protocols";
}
identity tcp {
    base transport-protocol;
}
identity udp {
    base transport-protocol;
}
identity sctp {
    base transport-protocol;
}

leaf protocol {
    type identityref {
        base transport-protocol;             // accepts tcp, udp, sctp
    }
}
// Other modules can define new identities derived from transport-protocol
```

### anyxml / anydata

```yang
anyxml raw-config {
    description "Opaque XML configuration blob";
}

anydata operational-data {
    description "Arbitrary YANG-modeled data (YANG 1.1)";
}
```

## YANG Types

### Built-in types

```yang
// Numeric
leaf port {          type uint16; }           // 0..65535
leaf metric {        type int32; }            // -2^31..2^31-1
leaf bandwidth {     type uint64; }           // 0..2^64-1
leaf temperature {   type decimal64 { fraction-digits 2; } }  // e.g., 23.50

// String
leaf name {          type string { length "1..255"; } }
leaf pattern-ex {    type string { pattern '[0-9a-fA-F]+'; } }

// Boolean
leaf enabled {       type boolean; }          // true or false

// Enumeration
leaf oper-status {
    type enumeration {
        enum up { value 1; }
        enum down { value 2; }
        enum testing { value 3; }
    }
}

// Bits (flag set)
leaf permissions {
    type bits {
        bit read { position 0; }
        bit write { position 1; }
        bit execute { position 2; }
    }
}

// Binary
leaf ssh-key {       type binary; }           // base64-encoded

// Empty (presence/absence)
leaf is-default {    type empty; }

// Union (multiple types)
leaf address-or-name {
    type union {
        type inet:ip-address;
        type inet:domain-name;
    }
}
```

### Common derived types (ietf-yang-types)

```yang
import ietf-yang-types { prefix yang; }

leaf mac {           type yang:mac-address; }       // xx:xx:xx:xx:xx:xx
leaf uuid {          type yang:uuid; }              // RFC 4122
leaf counter {       type yang:counter64; }         // monotonic counter
leaf gauge {         type yang:gauge32; }           // up-down gauge
leaf timestamp {     type yang:date-and-time; }     // RFC 3339
leaf hex {           type yang:hex-string; }        // hex-encoded bytes
leaf phys-addr {     type yang:phys-address; }      // physical address
```

### Common derived types (ietf-inet-types)

```yang
import ietf-inet-types { prefix inet; }

leaf ip {            type inet:ip-address; }        // IPv4 or IPv6
leaf ipv4 {          type inet:ipv4-address; }      // dotted quad
leaf ipv6 {          type inet:ipv6-address; }      // colon-hex
leaf prefix {        type inet:ip-prefix; }         // CIDR notation
leaf domain {        type inet:domain-name; }       // DNS name
leaf uri {           type inet:uri; }               // RFC 3986
leaf port {          type inet:port-number; }       // 0..65535
leaf as {            type inet:as-number; }         // 0..4294967295
```

## XPath Expressions in YANG

### must statement (constraint)

```yang
list interface {
    key "name";
    leaf name { type string; }
    leaf mtu {
        type uint16;
        must ". >= 68 and . <= 9192" {
            error-message "MTU must be 68..9192";
        }
    }
    leaf ipv4-address {
        type inet:ipv4-address;
        must "../enabled = 'true'" {
            error-message "Interface must be enabled to set IP";
        }
    }
}
```

### when statement (conditional presence)

```yang
container ipv4 {
    when "../type = 'ethernet'" {
        description "IPv4 only on Ethernet interfaces";
    }
    leaf address {
        type inet:ipv4-address;
    }
}

leaf secondary-address {
    when "../primary-address" {
        description "Secondary requires primary";
    }
    type inet:ipv4-address;
}
```

### leafref (cross-reference)

```yang
list interface {
    key "name";
    leaf name { type string; }
}

container routing {
    leaf egress-interface {
        type leafref {
            path "/interface/name";           // must reference existing interface
        }
    }
}
```

## OpenConfig vs IETF vs Native Models

### Model families

```bash
# OpenConfig — vendor-neutral, declarative, semantic versioning
#   - Maintained by OpenConfig working group (Google, Facebook, etc.)
#   - Focus: multi-vendor interoperability
#   - Naming: openconfig-interfaces, openconfig-bgp, openconfig-system
#   - Uses augment heavily, models state separately from config

# IETF — standards-track, RFC-published
#   - Maintained by IETF YANG working groups
#   - Focus: standards compliance
#   - Naming: ietf-interfaces, ietf-routing, ietf-system
#   - More conservative evolution, formal review process

# Native (Vendor) — platform-specific, full feature coverage
#   - Cisco: Cisco-IOS-XE-native, Cisco-IOS-XR-*, Cisco-NX-OS-*
#   - Juniper: junos-conf-*, junos-state-*
#   - Arista: arista-*, openconfig with augments
#   - Full feature coverage, but not portable across vendors
```

### Model selection strategy

```bash
# Prefer OpenConfig for:
#   - Multi-vendor environments
#   - Standardized automation pipelines
#   - When sufficient feature coverage exists

# Prefer IETF for:
#   - Standards compliance requirements
#   - Features modeled in IETF but not OpenConfig
#   - L3VPN, L2VPN, MPLS models

# Prefer Native for:
#   - Platform-specific features (VPC, FabricPath, etc.)
#   - When OpenConfig/IETF lacks the needed knobs
#   - Full configuration fidelity
```

## pyang Tool

### View YANG tree

```bash
# Install pyang
pip install pyang

# Print tree representation
pyang -f tree ietf-interfaces.yang
# Output:
# module: ietf-interfaces
#   +--rw interfaces
#      +--rw interface* [name]
#         +--rw name          string
#         +--rw description?  string
#         +--rw type          identityref
#         +--rw enabled?      boolean
#         +--ro if-index      int32
#         +--ro oper-status   enumeration

# Tree with augments from another module
pyang -f tree ietf-interfaces.yang ietf-ip.yang

# Tree for specific path
pyang -f tree --tree-path /interfaces/interface ietf-interfaces.yang

# Limit tree depth
pyang -f tree --tree-depth 3 openconfig-bgp.yang
```

### Validate YANG module

```bash
# Validate syntax
pyang --lint ietf-interfaces.yang

# Validate with dependencies
pyang -p /path/to/yang/models/ --lint my-module.yang

# Check strict RFC compliance
pyang --ietf my-module.yang
```

### Generate output formats

```bash
# Generate XSD schema
pyang -f xsd ietf-interfaces.yang -o interfaces.xsd

# Generate UML diagram
pyang -f uml ietf-interfaces.yang -o interfaces.uml

# Generate DSDL schemas (RelaxNG + Schematron)
pyang -f dsdl ietf-interfaces.yang

# Generate sample XML
pyang -f sample-xml-skeleton ietf-interfaces.yang

# Generate YANG from tree (reverse)
pyang -f yang --yang-canonical ietf-interfaces.yang
```

### Search YANG paths

```bash
# List all paths
pyang -f paths ietf-interfaces.yang
# /interfaces
# /interfaces/interface
# /interfaces/interface/name
# /interfaces/interface/description
# ...

# List all paths with types
pyang -f paths --paths-type ietf-interfaces.yang
```

## YANG Catalog

### Browse available models

```bash
# YANG Catalog — searchable index of all published YANG modules
# https://yangcatalog.org

# Search via API
curl -s "https://yangcatalog.org/api/search/modules/ietf-interfaces"

# Download models from GitHub
git clone https://github.com/YangModels/yang
# Structure:
#   yang/standard/ietf/RFC/        — IETF published models
#   yang/vendor/cisco/xe/          — Cisco IOS-XE models
#   yang/vendor/cisco/xr/          — Cisco IOS-XR models
#   yang/vendor/juniper/           — Juniper models
#   yang/experimental/             — draft models
```

## yanglint Validation

### Validate data against model

```bash
# Install yanglint (part of libyang)
# apt install libyang2-tools (Debian/Ubuntu)
# brew install libyang (macOS)

# Validate YANG module
yanglint ietf-interfaces@2018-02-20.yang

# Validate XML instance data against model
yanglint ietf-interfaces.yang instance-data.xml

# Validate JSON instance data
yanglint -f json ietf-interfaces.yang instance-data.json

# Interactive mode
yanglint
> load ietf-interfaces
> data -t config instance.xml
> quit
```

## Revision Management

### Revision history

```yang
module example-module {
    // ...
    revision 2026-01-15 {
        description "Added IPv6 support";
        reference "Release 2.0";
    }
    revision 2025-06-01 {
        description "Added description leaf";
    }
    revision 2025-01-01 {
        description "Initial revision";
    }
    // Most recent revision first
}
```

### Import with revision

```yang
import ietf-interfaces {
    prefix if;
    revision-date 2018-02-20;                // pin to specific revision
}
// Without revision-date, any revision is accepted
```

## YANG 1.1 Features (RFC 7950)

### Key additions over YANG 1.0

```yang
// 1. action — RPC within a data node context
list interface {
    key "name";
    leaf name { type string; }
    action reset-counters {                   // action within list
        output {
            leaf result { type string; }
        }
    }
}

// 2. anydata — typed opaque data (replaces anyxml for YANG data)
anydata inline-config;

// 3. Notification in data nodes
list interface {
    key "name";
    leaf name { type string; }
    notification link-failure {               // notification within list
        leaf admin-status { type boolean; }
    }
}

// 4. Leaf-list defaults
leaf-list protocol {
    type string;
    default "ssh";
    default "https";
}

// 5. if-feature with boolean expressions
leaf advanced-metric {
    if-feature "advanced-routing and not legacy-mode";
    type uint32;
}

// 6. Identity derives from multiple bases
identity tls-1.3 {
    base tls;
    base modern-crypto;
}
```

## See Also

- netconf
- restconf
- gnmi-gnoi
- pyats

## References

- RFC 7950 — YANG 1.1: https://datatracker.ietf.org/doc/html/rfc7950
- RFC 6020 — YANG 1.0: https://datatracker.ietf.org/doc/html/rfc6020
- RFC 6991 — Common YANG Data Types: https://datatracker.ietf.org/doc/html/rfc6991
- YANG Catalog: https://yangcatalog.org/
- OpenConfig YANG models: https://github.com/openconfig/public
- YANG Models repository: https://github.com/YangModels/yang
- pyang documentation: https://github.com/mbj4668/pyang
