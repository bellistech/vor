# NETCONF — Network Configuration Protocol Architecture

> *NETCONF (RFC 6241) is a transaction-oriented network management protocol that provides mechanisms to install, manipulate, and delete configuration on network devices. Built on a four-layer architecture (transport, messages, operations, content), it introduces the datastore abstraction that separates intended configuration from operational state. Its candidate datastore, confirmed commit, and lock semantics provide the configuration lifecycle guarantees that SNMP Set and CLI lack.*

---

## 1. RFC 6241 Architecture

### Four-Layer Model

NETCONF separates concerns into four distinct layers:

| Layer | Scope | Protocol Elements |
|:---|:---|:---|
| **Content** | Configuration and state data | YANG-modeled XML documents |
| **Operations** | Configuration manipulation primitives | `get`, `get-config`, `edit-config`, `copy-config`, etc. |
| **Messages** | RPC request-response framing | `<rpc>`, `<rpc-reply>`, `<notification>` |
| **Transport** | Reliable, ordered, authenticated delivery | SSH (mandatory), TLS (optional) |

This layering is strict: each layer depends only on the layer below it. The content layer is entirely defined by YANG models — NETCONF itself is content-agnostic.

### Session Model

NETCONF operates over persistent sessions. Each session has:

- **Session ID** — unique integer assigned by the server in `<hello>`
- **Capabilities** — negotiated feature set
- **Datastore locks** — exclusive access grants held by this session
- **Pending changes** — uncommitted edits in the candidate datastore

Sessions are long-lived and stateful, in contrast to RESTCONF's stateless HTTP model. This statefulness enables the lock-edit-commit workflow that provides transactional guarantees.

---

## 2. Datastore Model

### Datastore Types

NETCONF defines three configuration datastores:

| Datastore | Purpose | Persistence | Capability Required |
|:---|:---|:---|:---|
| `<running>` | Currently active configuration | Volatile (lost on reboot unless saved) | Always available |
| `<candidate>` | Staging area for uncommitted changes | Session-scoped | `:candidate` |
| `<startup>` | Configuration loaded at boot | Non-volatile | `:startup` |

The datastore model provides a clear separation between "what I want to change" (candidate), "what is active now" (running), and "what will load on reboot" (startup).

### Datastore Relationships

```
                    ┌──────────┐
      edit-config → │ candidate│ ← discard-changes (revert to running)
                    └────┬─────┘
                         │ commit
                         ▼
                    ┌──────────┐
                    │ running  │ ← edit-config (if :writable-running)
                    └────┬─────┘
                         │ copy-config
                         ▼
                    ┌──────────┐
                    │ startup  │
                    └──────────┘
```

When the `:candidate` capability is absent, the device supports only the `<running>` datastore. Edit-config operations apply directly to running — there is no staging or atomic commit.

### NMDA (Network Management Datastore Architecture — RFC 8342)

RFC 8342 extends the original model with additional datastores:

| Datastore | Purpose |
|:---|:---|
| `<intended>` | Validated configuration that the system should be operating with |
| `<operational>` | Complete view of system state (config + derived + system-created) |

NMDA addresses the reality that running config and actual device state can diverge (e.g., interface oper-status down despite admin-status up). The operational datastore merges configuration and state into a single unified view.

---

## 3. Confirmed Commit

### Mechanism

Confirmed commit (`:confirmed-commit:1.1`) adds a safety net to configuration changes:

```
1. Client sends: <commit><confirmed/><confirm-timeout>300</confirm-timeout></commit>
2. Server applies candidate to running
3. Timer starts (300 seconds)
4. Client verifies changes work correctly
5a. Client sends: <commit/>          → changes are permanent
5b. Timer expires without confirm    → server reverts running to pre-commit state
```

This is the network equivalent of a database transaction with auto-rollback. It prevents configuration lockouts — if a change breaks management connectivity, the device automatically reverts.

### Persist ID

The `persist` parameter allows a confirmed commit to survive session disconnection:

```xml
<rpc message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <commit>
    <confirmed/>
    <confirm-timeout>600</confirm-timeout>
    <persist>my-change-12345</persist>
  </commit>
</rpc>
```

A different session can then confirm using the persist ID:

```xml
<rpc message-id="2" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <commit>
    <persist-id>my-change-12345</persist-id>
  </commit>
</rpc>
```

This is useful when the confirming system is different from the editing system (e.g., a monitoring system that verifies reachability before confirming).

---

## 4. edit-config Operations

### Operation Semantics

The `operation` attribute on XML elements controls how each piece of configuration is applied:

| Operation | If Target Exists | If Target Does Not Exist |
|:---|:---|:---|
| `merge` (default) | Merge new values into existing | Create new |
| `replace` | Delete existing, create from scratch | Create new |
| `create` | Error (`data-exists`) | Create new |
| `delete` | Delete | Error (`data-missing`) |
| `remove` | Delete | No-op (silent success) |

### default-operation Parameter

The `default-operation` parameter in the `<edit-config>` RPC sets the default for elements without an explicit `operation` attribute:

| Value | Behavior |
|:---|:---|
| `merge` | Default — merge incoming data with existing |
| `replace` | Replace entire target datastore with supplied config |
| `none` | No default — every element must have explicit `operation` attribute |

Using `none` is the safest option: it prevents accidental overwrites by requiring explicit intent for every element. This is recommended for automation tools that construct partial configs.

### Error Handling Options

The `error-option` parameter controls behavior when an error occurs mid-edit:

| Value | Behavior | Capability Required |
|:---|:---|:---|
| `stop-on-error` (default) | Stop processing, leave partial changes | Always |
| `continue-on-error` | Process remaining operations, report all errors | Always |
| `rollback-on-error` | Revert all changes from this edit-config | `:rollback-on-error` |

`rollback-on-error` provides atomicity: either all operations succeed or none are applied. This is essential for automation where partial config application can leave devices in broken states.

---

## 5. NETCONF 1.1 Chunked Framing (RFC 6242)

### Evolution from 1.0

NETCONF 1.0 used the `]]>]]>` marker to delimit messages. This created a vulnerability: if the configuration data itself contained the sequence `]]>]]>`, the parser would incorrectly interpret it as a message boundary.

NETCONF 1.1 (negotiated via the `urn:ietf:params:netconf:base:1.1` capability) replaces this with chunked framing:

```
\n#<length>\n<data>\n##\n

Example:
\n#128\n
<rpc xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="1">
  <get-config><source><running/></source></get-config>
</rpc>
\n##\n
```

The length prefix is a decimal byte count. Multiple chunks can compose a single message, allowing large configurations to be sent in manageable pieces.

### Framing Negotiation

The framing mode is determined by the capabilities exchanged in `<hello>`:

| Client | Server | Framing Used |
|:---|:---|:---|
| base:1.0 only | base:1.0 only | End-of-message `]]>]]>` |
| base:1.1 only | base:1.1 only | Chunked |
| Both | Both | Chunked (1.1 preferred) |
| base:1.0 | base:1.1 | Session fails |

---

## 6. YANG Integration

### Content Layer Binding

NETCONF is content-agnostic — it treats configuration as opaque XML. YANG provides the schema that gives meaning to that XML:

```
YANG module → defines data model (containers, lists, leaves, types)
    ↓
XML instance → NETCONF carries this as <config> or <data>
    ↓
Device implementation → maps YANG model to device internals
```

### Schema Discovery

Devices advertise their supported YANG modules through two mechanisms:

**1. Capability URIs in Hello:**
```
urn:ietf:params:xml:ns:yang:ietf-interfaces?module=ietf-interfaces&revision=2018-02-20
```

**2. YANG Library (RFC 7895 / RFC 8525):**
```xml
<get>
  <filter type="subtree">
    <yang-library xmlns="urn:ietf:params:xml:ns:yang:ietf-yang-library"/>
  </filter>
</get>
```

**3. get-schema RPC (RFC 6022):**
```xml
<get-schema xmlns="urn:ietf:params:xml:ns:yang:ietf-netconf-monitoring">
  <identifier>ietf-interfaces</identifier>
  <version>2018-02-20</version>
  <format>yang</format>
</get-schema>
```

### Namespace Mapping

YANG modules map to XML namespaces. Every YANG module has a `namespace` statement that becomes the XML namespace URI:

```yang
module ietf-interfaces {
    namespace "urn:ietf:params:xml:ns:yang:ietf-interfaces";
    prefix if;
    ...
}
```

In NETCONF XML:
```xml
<interfaces xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
  <interface>
    <name>eth0</name>
  </interface>
</interfaces>
```

---

## 7. NETCONF vs RESTCONF vs gNMI Comparison

### Protocol Matrix

| Feature | NETCONF | RESTCONF | gNMI |
|:---|:---|:---|:---|
| **RFC/Spec** | RFC 6241 | RFC 8040 | OpenConfig spec |
| **Transport** | SSH | HTTPS | gRPC (HTTP/2) |
| **Encoding** | XML | JSON or XML | Protobuf |
| **Session** | Stateful | Stateless | Stateful (streams) |
| **Candidate DS** | Yes | No | No |
| **Lock** | Yes | No | No |
| **Confirmed Commit** | Yes | No | No |
| **Streaming Telemetry** | Notifications (RFC 5277) | SSE | Subscribe (native) |
| **Transaction** | Lock + candidate + commit | Per-request | Atomic Set |
| **Partial Config** | edit-config with operation attr | PATCH | Update (merge) |
| **Full Replace** | copy-config or replace operation | PUT | Replace |
| **Validation** | validate RPC | Server-side only | Server-side only |
| **Call-Home** | RFC 8071 | Not standardized | Not standardized |
| **Tooling** | ncclient, NAPALM | curl, requests, Postman | gnmic, pygnmi |

### Decision Framework

Use **NETCONF** when:
- You need transactional guarantees (lock → edit → validate → commit)
- Confirmed commit with auto-rollback is required
- The device supports candidate datastore and you want staging
- You need call-home for devices behind NAT/firewall

Use **RESTCONF** when:
- HTTP/REST integration is simpler for your toolchain
- You want quick ad-hoc queries (curl from terminal)
- Your team is comfortable with REST APIs
- No candidate/commit workflow is needed

Use **gNMI** when:
- Streaming telemetry is the primary use case
- High-frequency counter collection is needed
- Your environment is OpenConfig-standardized
- You want binary-efficient transport

---

## 8. Error Handling

### Error Response Structure

```xml
<rpc-reply message-id="1" xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <rpc-error>
    <error-type>application</error-type>
    <error-tag>data-exists</error-tag>
    <error-severity>error</error-severity>
    <error-path>/interfaces/interface[name='Loopback99']</error-path>
    <error-message>Data already exists</error-message>
    <error-info>
      <bad-element>interface</bad-element>
    </error-info>
  </rpc-error>
</rpc-reply>
```

### Error Types

| error-type | Layer |
|:---|:---|
| `transport` | SSH/TLS layer failure |
| `rpc` | Malformed RPC message |
| `protocol` | NETCONF protocol violation |
| `application` | Device/model-level error |

### Common Error Tags

| error-tag | Meaning |
|:---|:---|
| `in-use` | Resource locked by another session |
| `invalid-value` | Value does not match YANG type/range |
| `too-big` | Request exceeds device capacity |
| `missing-attribute` | Required XML attribute missing |
| `bad-attribute` | Invalid XML attribute |
| `unknown-element` | Element not in YANG model |
| `access-denied` | Insufficient privileges |
| `lock-denied` | Lock held by another session |
| `data-exists` | create operation on existing data |
| `data-missing` | delete operation on non-existent data |
| `operation-not-supported` | Operation not available |
| `operation-failed` | General operation failure |

---

## See Also

- restconf
- yang-models
- gnmi-gnoi
- pyats

## References

- RFC 6241 — NETCONF Protocol: https://datatracker.ietf.org/doc/html/rfc6241
- RFC 6242 — NETCONF over SSH: https://datatracker.ietf.org/doc/html/rfc6242
- RFC 6020 — YANG: https://datatracker.ietf.org/doc/html/rfc6020
- RFC 8342 — NMDA: https://datatracker.ietf.org/doc/html/rfc8342
- RFC 5277 — NETCONF Notifications: https://datatracker.ietf.org/doc/html/rfc5277
- RFC 8071 — NETCONF Call Home: https://datatracker.ietf.org/doc/html/rfc8071
- RFC 6022 — YANG Module for NETCONF Monitoring: https://datatracker.ietf.org/doc/html/rfc6022
