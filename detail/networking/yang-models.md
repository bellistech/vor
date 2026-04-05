# YANG — Data Modeling Language for Network Management

> *YANG (RFC 7950) is a data modeling language designed to model configuration and state data, RPCs, actions, and notifications for the NETCONF, RESTCONF, and gNMI management protocols. It defines a hierarchical tree of typed nodes with constraints, enabling schema-driven validation of network device APIs. Understanding YANG's composition mechanisms (grouping, augment, deviation), its constraint system (must, when, leafref), and the ecosystem of standard models (OpenConfig, IETF, vendor-native) is fundamental to modern model-driven network automation.*

---

## 1. YANG as a Data Modeling Language

### Design Goals

YANG was designed with specific goals that distinguish it from general-purpose schema languages like XML Schema or JSON Schema:

| Goal | How YANG Achieves It |
|:---|:---|
| Human readability | C-like syntax, not XML |
| Network domain focus | Built-in types for IP, MAC, ASN |
| Configuration vs state | `config true/false` semantics |
| Constraint expression | `must`, `when`, `leafref`, `unique` |
| Extensibility | `augment`, `deviation`, `identity` |
| Reusability | `grouping`, `uses`, `typedef` |
| Protocol independence | Maps to NETCONF XML, RESTCONF JSON, gNMI protobuf |

### YANG vs Other Schema Languages

| Feature | YANG | XML Schema (XSD) | JSON Schema | Protobuf |
|:---|:---|:---|:---|:---|
| Domain | Network mgmt | General XML | General JSON | RPC/serialization |
| Config/state distinction | Native | No | No | No |
| Cross-node constraints | `must`, `when`, `leafref` | Limited (keyref) | `$ref`, no xpath | No |
| Extensibility | `augment` | Extension | `additionalProperties` | `extend` |
| Human readability | High | Low (verbose XML) | Medium | High |
| RPC modeling | `rpc`, `action` | No | No | `service` |
| Notification modeling | `notification` | No | No | No |

---

## 2. YANG Tree Structure Semantics

### Node Categories

YANG defines four categories of schema nodes:

| Category | YANG Nodes | Data Representation |
|:---|:---|:---|
| Data definition | `container`, `leaf`, `leaf-list`, `list`, `anydata`, `anyxml` | Appear in instance data |
| Schema-only | `choice`, `case` | Structure only, invisible in data |
| Type definition | `typedef`, `identity` | Define types, no data representation |
| Reuse mechanisms | `grouping`, `uses`, `augment` | Expand into data definition nodes |

### Tree Symbols (pyang)

```
+--rw  — read-write configuration node
+--ro  — read-only state/operational node
+--x   — RPC or action
+--n   — notification

?      — optional (not mandatory)
*      — list or leaf-list (multiple entries)
[key]  — list key
(choice) — choice node (invisible in data)
```

### Ordering Semantics

YANG supports two ordering modes for lists and leaf-lists:

| Mode | Behavior | Default For |
|:---|:---|:---|
| `ordered-by system` | Server determines order (typically insertion order or sorted) | `list`, `leaf-list` |
| `ordered-by user` | Client controls order, server preserves it | Must be explicitly declared |

User-ordered lists are critical for ACLs, route-maps, and prefix-lists where order determines match behavior.

### Presence Containers

```yang
container logging {
    presence "Enables logging";               // presence container
    leaf level { type enumeration { enum debug; enum info; } }
}
```

A **presence container** has meaning by its existence alone. If the `logging` container exists in the config, logging is enabled, even if all its children use defaults. Without the `presence` statement, a container is a **non-presence container** — it exists purely for structural grouping and is implied when any child exists.

---

## 3. Constraint Expressions

### must Statement

The `must` statement defines an XPath 1.0 expression that must evaluate to true for the data to be valid:

```yang
leaf bandwidth {
    type uint32;
    must ". <= ../max-bandwidth" {
        error-message "Bandwidth exceeds maximum";
        error-app-tag "bandwidth-exceeded";
    }
}
```

The XPath context node (`.`) is the node containing the `must` statement. The expression can reference sibling nodes (`../sibling`), ancestor nodes (`../../ancestor`), and even nodes in other parts of the tree using absolute paths.

### Common XPath Patterns in must

| Pattern | Purpose | Example |
|:---|:---|:---|
| `. >= N` | Range validation | `must ". >= 1"` |
| `../sibling = 'val'` | Sibling dependency | `must "../type = 'ethernet'"` |
| `count(../list) <= N` | List cardinality | `must "count(../server) <= 3"` |
| `not(../leaf)` | Mutual exclusion | `must "not(../ipv6-address)"` |
| `boolean(../leaf)` | Presence check | `must "boolean(../primary)"` |
| `contains(., 'str')` | String content | `must "contains(., '@')"` |
| `string-length(.) <= N` | String length | `must "string-length(.) <= 64"` |

### when Statement

The `when` statement conditionally includes a node in the schema tree:

```yang
leaf ipv4-address {
    when "../address-family = 'ipv4'";
    type inet:ipv4-address;
}
```

If the `when` expression evaluates to false, the node does not exist in the data tree — it cannot be set, and any existing value is ignored. This differs from `must`, which validates existing data rather than controlling presence.

### leafref Constraints

`leafref` creates a foreign-key-like reference between nodes:

```yang
list vrf {
    key "name";
    leaf name { type string; }
}

container routing {
    leaf vrf {
        type leafref {
            path "/vrf/name";
        }
        // Value must match an existing VRF name
    }
}
```

The `path` expression uses a restricted XPath syntax. With `require-instance true` (default), the referenced value must exist. With `require-instance false`, the reference is advisory only.

### unique Constraint

The `unique` statement enforces uniqueness across list entries for non-key leaves:

```yang
list server {
    key "name";
    unique "ip port";                         // IP+port combination must be unique
    leaf name { type string; }
    leaf ip { type inet:ip-address; }
    leaf port { type inet:port-number; }
}
```

---

## 4. Grouping/Augment Composition

### Composition Model

YANG's reuse system is based on textual substitution at compile time:

```
grouping definition          → template (not in data tree)
uses expansion               → inline substitution at usage point
augment from external module → cross-module extension
deviation                    → platform-specific override
```

### Grouping Scope

Groupings are purely schema constructs — they have no representation in instance data. When `uses` expands a grouping, the result is indistinguishable from manually defining the same nodes:

```yang
grouping endpoint {
    leaf address { type inet:ip-address; }
    leaf port { type inet:port-number; }
}

container source {
    uses endpoint;
    // Equivalent to:
    // leaf address { type inet:ip-address; }
    // leaf port { type inet:port-number; }
}
```

### Augment Composition Rules

Augmentation follows specific rules:

1. **Target must exist** — the augmented path must be valid in the target module
2. **No key modification** — cannot add keys to an existing list
3. **Conditional augment** — `when` expression can gate augment applicability
4. **Namespace preservation** — augmented nodes retain the augmenting module's namespace
5. **Multiple augments** — multiple modules can augment the same target

```yang
// Module A augments ietf-interfaces
augment "/if:interfaces/if:interface" {
    when "if:type = 'ethernet'";              // only for Ethernet interfaces
    container qos {
        leaf policy { type string; }
    }
}

// Module B also augments ietf-interfaces
augment "/if:interfaces/if:interface" {
    container acl {
        leaf ingress { type string; }
        leaf egress { type string; }
    }
}

// Both augments coexist — interface gets both qos and acl containers
```

---

## 5. Deviation Use Cases

### Why Deviations Exist

YANG modules define an idealized data model, but real devices may:
- Not implement all features (e.g., IPv6 on a legacy platform)
- Support different value ranges (e.g., MTU 64-9216 instead of 68-65535)
- Add vendor-specific constraints
- Deprecate features in specific software versions

Deviations declare these platform-specific differences without forking the base module.

### Deviation Patterns

**Not Supported — feature absent on platform:**
```yang
deviation "/sys:system/sys:ntp/sys:authentication" {
    deviate not-supported;
    description "NTP authentication not implemented on this platform";
}
```

**Replace — different type or constraint:**
```yang
deviation "/if:interfaces/if:interface/if:mtu" {
    deviate replace {
        type uint16 { range "64..9216"; }     // platform-specific range
    }
}
```

**Add — additional constraints:**
```yang
deviation "/if:interfaces/if:interface/if:name" {
    deviate add {
        must "re-match(., 'Ethernet[0-9]+/[0-9]+(/[0-9]+)?')" {
            error-message "Invalid interface name format";
        }
    }
}
```

**Delete — remove default or constraint:**
```yang
deviation "/if:interfaces/if:interface/if:enabled" {
    deviate delete {
        default true;                         // remove default value
    }
}
```

### Deviation Discovery

Devices advertise deviations in NETCONF capabilities or YANG Library:

```
urn:ietf:params:xml:ns:yang:ietf-interfaces?module=ietf-interfaces
  &revision=2018-02-20
  &deviations=vendor-ietf-interfaces-devs
```

Automation tools should fetch deviations to understand the actual schema on each platform.

---

## 6. YANG Module Lifecycle

### Evolution Rules (RFC 7950 Section 11)

YANG imposes strict backwards-compatibility rules on module revisions:

| Allowed Changes | Not Allowed |
|:---|:---|
| Add new nodes | Remove existing nodes |
| Add new enum/bit values | Change meaning of existing nodes |
| Relax constraints (widen range) | Tighten constraints (narrow range) |
| Change description text | Change node type |
| Add optional leaves | Make optional nodes mandatory |
| Add new features | Remove features |

These rules ensure that clients written against an older revision continue to work with newer revisions. Breaking changes require a new module name (not just a new revision).

### Module Naming Conventions

| Source | Pattern | Example |
|:---|:---|:---|
| IETF | `ietf-<domain>` | `ietf-interfaces`, `ietf-routing` |
| OpenConfig | `openconfig-<domain>` | `openconfig-bgp`, `openconfig-interfaces` |
| Cisco IOS-XE | `Cisco-IOS-XE-<feature>` | `Cisco-IOS-XE-bgp`, `Cisco-IOS-XE-native` |
| Cisco IOS-XR | `Cisco-IOS-XR-<feature>` | `Cisco-IOS-XR-ifmgr-cfg` |
| Juniper | `junos-conf-<hierarchy>` | `junos-conf-interfaces` |

---

## 7. OpenConfig Design Principles

### Architecture

OpenConfig YANG models follow specific design principles:

**1. Config/state separation:**
```yang
container interfaces {
    list interface {
        key "name";
        container config {                    // intended configuration
            leaf name { type string; }
            leaf enabled { type boolean; }
        }
        container state {                     // operational state
            config false;
            leaf name { type string; }
            leaf enabled { type boolean; }
            leaf oper-status { type enumeration { ... } }
            container counters { ... }
        }
    }
}
```

Every configurable leaf appears in both `config` (read-write) and `state` (read-only) containers. The `state` container also includes derived/computed values. This design enables clear comparison between intended and actual state.

**2. Semantic versioning:**
```
openconfig-interfaces 3.1.0
  3 = major (breaking)
  1 = minor (backwards-compatible addition)
  0 = patch (bug fix)
```

**3. Vendor-neutral abstraction:**
OpenConfig models abstract common network features across vendors. Platform-specific knobs are handled through augmentation from vendor-specific modules.

**4. Path-based telemetry alignment:**
OpenConfig paths map directly to gNMI subscription paths, ensuring the data model and telemetry pipeline use the same schema.

---

## 8. IETF YANG Module Catalog

### Module Publication Process

```
Individual Draft → WG Adoption → WG Last Call → IETF Last Call → RFC
  ↓                  ↓              ↓               ↓             ↓
draft-smith-foo  draft-ietf-foo  reviewed       approved      published
  (experimental)   (WG item)     (mature)       (stable)      (standard)
```

### Key IETF YANG Modules

| Module | RFC | Domain |
|:---|:---|:---|
| `ietf-interfaces` | RFC 8343 | Interface configuration and state |
| `ietf-ip` | RFC 8344 | IPv4/IPv6 configuration |
| `ietf-routing` | RFC 8349 | Routing management |
| `ietf-ospf` | RFC 9129 | OSPF configuration |
| `ietf-bgp` | draft | BGP configuration |
| `ietf-system` | RFC 7317 | System management |
| `ietf-netconf-monitoring` | RFC 6022 | NETCONF session monitoring |
| `ietf-yang-library` | RFC 8525 | Module capability reporting |
| `ietf-yang-types` | RFC 6991 | Common data types |
| `ietf-inet-types` | RFC 6991 | IP/network data types |
| `ietf-l3vpn-svc` | RFC 8299 | L3VPN service model |
| `ietf-l2vpn-svc` | RFC 8466 | L2VPN service model |
| `ietf-te` | RFC 8776 | Traffic engineering |
| `ietf-access-control-list` | RFC 8519 | ACL management |

---

## 9. YANG Compilation and Validation

### Compilation Pipeline

```
YANG source files
    │
    ├─ 1. Lexical analysis (tokenization)
    │
    ├─ 2. Syntax validation (grammar rules)
    │
    ├─ 3. Semantic validation
    │     ├─ Type checking
    │     ├─ Reference resolution (leafref, identityref, path)
    │     ├─ Grouping expansion
    │     ├─ Augment application
    │     ├─ Deviation application
    │     └─ Import/include resolution
    │
    ├─ 4. Schema tree construction
    │     └─ Fully expanded, deviation-applied tree
    │
    └─ 5. Output generation
          ├─ Tree diagram (pyang -f tree)
          ├─ XSD schema (pyang -f xsd)
          ├─ Code stubs (pyangbind, ygot)
          └─ Documentation
```

### Validation Tools

| Tool | Language | Strengths |
|:---|:---|:---|
| `pyang` | Python | Tree visualization, linting, multiple output formats |
| `yanglint` | C (libyang) | Fast validation, data instance checking, interactive |
| `confd` | C | Full NETCONF/RESTCONF server, compilation to .fxs |
| `ydk-gen` | Python | Code generation from YANG |
| `goyang` | Go | YANG parser for Go toolchains |
| `libyang` | C | High-performance parsing library |

### Common Validation Errors

| Error | Cause | Fix |
|:---|:---|:---|
| `module not found` | Missing import dependency | Add module to search path (`-p`) |
| `node not found` | Augment targets non-existent path | Check target module revision |
| `type not found` | Missing typedef module | Import module defining the type |
| `duplicate node` | Same node name at same level | Rename or use different namespace |
| `circular dependency` | Module A imports B which imports A | Restructure with submodules |

---

## 10. YANG-to-Code Generation

### Code Generation Tools

Several tools generate programming language bindings from YANG models:

| Tool | Output Language | Approach |
|:---|:---|:---|
| `pyangbind` | Python | Python classes with type validation |
| `ygot` | Go | Go structs with YANG-aware serialization |
| `ydk-gen` | Python, Go, C++ | Full SDK with CRUD operations |
| `libnetconf2` | C | NETCONF client/server library |
| `sysrepo` | C | YANG-based datastore with C API |

### pyangbind Example

```bash
# Generate Python bindings
pyang --plugindir $(python -c 'import pyangbind; print(pyangbind.__path__[0])') \
  -f pybind ietf-interfaces.yang \
  -o ietf_interfaces.py
```

```python
# Use generated bindings
from ietf_interfaces import ietf_interfaces

model = ietf_interfaces()
intf = model.interfaces.interface.add("Loopback0")
intf.config.enabled = True
intf.config.description = "Management"

# Serialize to JSON
import pyangbind.lib.pybindJSON as pybindJSON
print(pybindJSON.dumps(model, mode="ietf"))
```

### ygot (Go)

```bash
# Generate Go structs
go install github.com/openconfig/ygot/generator@latest
generator -path=yang/ -output_file=oc.go \
  -package_name=oc \
  openconfig-interfaces.yang
```

```go
// Use generated structs
device := &oc.Device{}
intf, _ := device.NewInterface("Loopback0")
intf.Config.Description = ygot.String("Management")
intf.Config.Enabled = ygot.Bool(true)

// Serialize to JSON
json, _ := ygot.EmitJSON(device, &ygot.EmitJSONConfig{
    Format: ygot.RFC7951,
})
```

---

## See Also

- netconf
- restconf
- gnmi-gnoi
- pyats

## References

- RFC 7950 — YANG 1.1: https://datatracker.ietf.org/doc/html/rfc7950
- RFC 6020 — YANG 1.0: https://datatracker.ietf.org/doc/html/rfc6020
- RFC 6991 — Common YANG Data Types: https://datatracker.ietf.org/doc/html/rfc6991
- RFC 8340 — YANG Tree Diagrams: https://datatracker.ietf.org/doc/html/rfc8340
- RFC 8342 — NMDA: https://datatracker.ietf.org/doc/html/rfc8342
- RFC 8525 — YANG Library: https://datatracker.ietf.org/doc/html/rfc8525
- OpenConfig YANG models: https://github.com/openconfig/public
- YANG Catalog: https://yangcatalog.org/
- pyangbind: https://github.com/robshakir/pyangbind
- ygot: https://github.com/openconfig/ygot
