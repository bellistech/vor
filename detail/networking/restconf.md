# RESTCONF — REST-Based Network Configuration Architecture

> *RESTCONF (RFC 8040) maps YANG data models to a RESTful HTTP API, providing a lightweight alternative to NETCONF for network device configuration and state retrieval. It leverages standard HTTP methods, JSON/XML encoding, and query parameters to provide a familiar REST interface over YANG-modeled data. While it lacks NETCONF's candidate datastore and confirmed commit, its simplicity and HTTP tooling compatibility make it the preferred choice for web-integrated automation workflows.*

---

## 1. Resource URI Construction

### URI Structure

RESTCONF defines a strict URI hierarchy that mirrors the YANG data model:

```
https://{host}:{port}/{+restconf}/{resource-type}/{path}?{query}

Components:
  {+restconf}    → API root (discovered via /.well-known/host-meta)
  {resource-type} → data | operations | yang-library-version
  {path}          → YANG node path with keys
  {query}         → depth, fields, content, filter, with-defaults
```

### Path Construction Rules

The path component follows specific rules for mapping YANG nodes to URI segments:

**1. Module prefix on top-level nodes:**
```
/restconf/data/ietf-interfaces:interfaces
               ^^^^^^^^^^^^^^^^ module prefix required
```

**2. No prefix for same-module children:**
```
/restconf/data/ietf-interfaces:interfaces/interface=eth0
                                          ^^^^^^^^^ no prefix (same module)
```

**3. Module prefix on augmented nodes:**
```
/restconf/data/ietf-interfaces:interfaces/interface=eth0/ietf-ip:ipv4
                                                         ^^^^^^^ different module (augment)
```

**4. List instance keys:**
```
YANG: list interface { key "name"; ... }
URI:  /interfaces/interface=Loopback0
                           ^ key encoded after =
```

**5. Multiple keys (comma-separated):**
```
YANG: list protocol { key "identifier name"; ... }
URI:  /protocols/protocol=BGP,bgp
                          ^^^^^^^^ key1,key2
```

**6. URL encoding for special characters:**

| Character | Encoded | Example |
|:---|:---|:---|
| `/` | `%2F` | `GigabitEthernet0%2F0%2F0` |
| `:` | `%3A` | rarely needed in path values |
| ` ` | `%20` | `interface=my%20port` |
| `=` | `%3D` | only in key values |
| `,` | `%2C` | only in key values |

### Identifying the Correct Path

Finding the correct RESTCONF path requires knowing the YANG module structure:

```
1. Discover supported modules:
   GET /restconf/data/ietf-yang-library:yang-library/module-set

2. Get module schema:
   GET /restconf/data/ietf-netconf-monitoring:netconf-state/schemas

3. Use pyang to visualize:
   pyang -f tree ietf-interfaces.yang
   → module: ietf-interfaces
       +--rw interfaces
          +--rw interface* [name]
             +--rw name          string
             +--rw description?  string
             +--rw type          identityref
             +--rw enabled?      boolean
```

---

## 2. YANG Module-to-API Mapping

### Data Model to REST Mapping

YANG constructs map to RESTCONF resources as follows:

| YANG Construct | RESTCONF Representation | HTTP Methods |
|:---|:---|:---|
| `container` | URI path segment, JSON object | GET, PUT, PATCH, DELETE |
| `list` | Collection at parent URI, instance at `/list=key` | GET (collection), POST (create), PUT/PATCH/DELETE (instance) |
| `leaf` | URI path segment, JSON primitive | GET, PUT, PATCH, DELETE |
| `leaf-list` | URI path segment, JSON array | GET, POST, PUT, PATCH, DELETE |
| `rpc` | Under `/restconf/operations/` | POST only |
| `action` | Under data resource path | POST only |
| `notification` | Under `/restconf/streams/` | GET (SSE) |

### JSON Encoding Rules (RFC 7951)

YANG data in JSON follows specific encoding rules:

**Module-qualified names:**
```json
{
  "ietf-interfaces:interfaces": {
    "interface": [
      {
        "name": "eth0",
        "ietf-ip:ipv4": {
          "address": [{"ip": "10.0.0.1", "prefix-length": 24}]
        }
      }
    ]
  }
}
```

The top-level node uses `module:name` format. Child nodes from the same module omit the prefix. Child nodes from a different module (augmented) must include the module prefix.

**Type Mappings:**

| YANG Type | JSON Type | Example |
|:---|:---|:---|
| `string` | string | `"hello"` |
| `int8/16/32` | number | `42` |
| `int64`, `uint64` | string | `"9223372036854775807"` |
| `boolean` | boolean | `true` |
| `empty` | `[null]` | `[null]` |
| `enumeration` | string | `"up"` |
| `bits` | string (space-separated) | `"read write"` |
| `identityref` | string (module:name) | `"iana-if-type:ethernetCsmacd"` |
| `union` | varies (first matching type) | depends on value |
| `decimal64` | number | `3.14` |
| `binary` | string (base64) | `"SGVsbG8="` |
| `instance-identifier` | string | `"/if:interfaces/if:interface[if:name='eth0']"` |

---

## 3. PATCH Semantics

### Plain PATCH (RFC 8040 Section 4.6.1)

A plain PATCH request performs a merge operation — the supplied data is merged into the target resource. Fields not present in the PATCH body are left unchanged.

```
Target state:     {"name": "eth0", "enabled": true, "description": "old"}
PATCH body:       {"name": "eth0", "description": "new"}
Result:           {"name": "eth0", "enabled": true, "description": "new"}
                                   ^^^^^^^^^^^^^^ preserved (not in PATCH)
```

This is equivalent to NETCONF `edit-config` with `default-operation="merge"`.

### YANG Patch (RFC 8072)

YANG Patch provides fine-grained control over individual operations within a single PATCH request, similar to NETCONF's per-element `operation` attribute:

```json
{
  "ietf-yang-patch:yang-patch": {
    "patch-id": "patch-1",
    "edit": [
      {
        "edit-id": "edit-1",
        "operation": "create",
        "target": "/interface=Loopback99",
        "value": {
          "interface": {
            "name": "Loopback99",
            "type": "iana-if-type:softwareLoopback"
          }
        }
      },
      {
        "edit-id": "edit-2",
        "operation": "merge",
        "target": "/interface=Loopback0",
        "value": {
          "interface": {
            "name": "Loopback0",
            "description": "Updated"
          }
        }
      },
      {
        "edit-id": "edit-3",
        "operation": "delete",
        "target": "/interface=Loopback50"
      }
    ]
  }
}
```

YANG Patch operations:

| Operation | Behavior |
|:---|:---|
| `create` | Create new, error if exists |
| `delete` | Delete, error if not exists |
| `insert` | Insert into ordered list (with point/where) |
| `merge` | Merge into existing or create |
| `move` | Reorder in ordered list |
| `replace` | Replace if exists, create if not |
| `remove` | Delete if exists, no-op if not |

YANG Patch uses content type: `application/yang-patch+json` or `application/yang-patch+xml`.

---

## 4. RESTCONF vs NETCONF Operation Mapping

### Operation Equivalence

| RESTCONF | NETCONF | Notes |
|:---|:---|:---|
| `GET /data` | `<get>` | Config + state |
| `GET /data?content=config` | `<get-config source="running">` | Config only |
| `POST /data/{parent}` | `<edit-config operation="create">` | Create child |
| `PUT /data/{resource}` | `<edit-config operation="replace">` | Replace or create |
| `PATCH /data/{resource}` | `<edit-config operation="merge">` | Merge (plain PATCH) |
| `DELETE /data/{resource}` | `<edit-config operation="delete">` | Delete |
| `POST /operations/{rpc}` | `<rpc><rpc-name>` | RPC invocation |

### Feature Gaps

RESTCONF intentionally omits several NETCONF features:

| Feature | NETCONF | RESTCONF | Rationale |
|:---|:---|:---|:---|
| Candidate datastore | Yes | No | HTTP is stateless — no session to hold candidate |
| Lock/Unlock | Yes | No | Stateless — no session-owned locks |
| Confirmed commit | Yes | No | Requires session state for rollback timer |
| Validation RPC | Yes | No | Server validates inline with each request |
| Rollback on error | Yes | No | Each request is atomic (single operation) |
| Call-home | Yes (RFC 8071) | Partial | Less common in practice |

These gaps mean RESTCONF is best suited for environments where:
- Individual atomic operations suffice (no multi-step transactions)
- Rollback is handled at the automation layer (e.g., Ansible rollback playbook)
- The simplicity of HTTP outweighs the loss of transaction semantics

---

## 5. Event Notifications over SSE

### Server-Sent Events (RFC 8040 Section 6)

RESTCONF uses Server-Sent Events (SSE) for event notification delivery, replacing NETCONF's `<notification>` messages:

```
Client                              Server
  |--- GET /restconf/streams/NETCONF →|    (Accept: text/event-stream)
  |                                    |
  |←-- HTTP 200 (chunked transfer) ---|
  |←-- data: <notification>...</>  ---|    (event 1)
  |←-- id: 1                       ---|
  |←--                              ---|
  |←-- data: <notification>...</>  ---|    (event 2)
  |←-- id: 2                       ---|
  |        ...keeps streaming...       |
```

### SSE Format

Each event is formatted as:

```
data: <notification xmlns="urn:ietf:params:xml:ns:netconf:notification:1.0">
data:   <eventTime>2026-01-15T10:30:00Z</eventTime>
data:   <interface-state-change xmlns="urn:ietf:params:xml:ns:yang:ietf-interfaces">
data:     <interface-name>GigabitEthernet0/0/0</interface-name>
data:     <oper-status>down</oper-status>
data:   </interface-state-change>
data: </notification>
id: 42

```

The `id` field enables reconnection — the client sends `Last-Event-ID: 42` on reconnect and the server resumes from that point.

### Limitations

SSE has significant limitations compared to gNMI Subscribe:

| Aspect | SSE (RESTCONF) | Subscribe (gNMI) |
|:---|:---|:---|
| Direction | Server → client only | Bidirectional |
| Subscription control | Limited (stream selection) | Per-path, per-mode |
| Sample interval | Not supported | Configurable per path |
| On-change | Stream-dependent | Native |
| Reconnection | Last-Event-ID | gRPC retry |
| Encoding | XML/JSON text | Protobuf binary |
| Performance | Low throughput | High throughput |

---

## 6. RESTCONF Performance Characteristics

### Latency Profile

| Operation | Typical Latency | Factors |
|:---|:---|:---|
| GET (single leaf) | 10-50 ms | Path depth, TLS overhead |
| GET (large subtree) | 100 ms - 5s | Data size, JSON serialization |
| PUT/PATCH (single resource) | 50-500 ms | Config compilation, commit |
| POST (create) | 50-500 ms | Schema validation + commit |
| DELETE | 50-200 ms | Dependency check + commit |

### Throughput Considerations

RESTCONF's HTTP/1.1 transport creates overhead compared to gNMI:

- **Connection per request** — unless HTTP keep-alive is used
- **Text encoding** — JSON/XML is 3-10x larger than protobuf
- **No streaming** — each GET is a full request-response cycle
- **No multiplexing** — HTTP/1.1 is sequential per connection (HTTP/2 would help, but few implementations support it)

For bulk operations, RESTCONF is significantly slower than NETCONF (which can batch multiple edits in one `edit-config`) or gNMI (which uses a single Set with multiple updates).

### Optimization Strategies

- Use `fields` query parameter to select only needed leaves
- Use `depth` to limit response nesting
- Use `content=config` when state data is not needed
- Batch related changes using YANG Patch instead of individual requests
- Use HTTP keep-alive to avoid TLS handshake per request

---

## 7. API Versioning

### YANG Module Revisions

RESTCONF does not have explicit API versioning. Instead, versioning is implicit through YANG module revisions:

```
Module: ietf-interfaces
Revision: 2018-02-20   ← determines available leaves and types
```

Clients discover the active revision via:
- YANG Library: `GET /restconf/data/ietf-yang-library:yang-library`
- Capabilities in root resource

### Handling Model Evolution

When YANG models change across device software versions:

| Change Type | Impact | Strategy |
|:---|:---|:---|
| New leaf added | Old clients unaffected | New clients can use it |
| Leaf deprecated | Still accessible | Clients should migrate |
| Leaf removed | Old clients break | Version-check in automation |
| New augment module | Old clients unaffected | New module prefix in paths |
| Type constraint tightened | Previously valid values rejected | Validate before sending |

### OpenConfig Versioning

OpenConfig uses semantic versioning in module revisions:

```
openconfig-interfaces 3.0.0
  → Major: breaking changes
  → Minor: backwards-compatible additions
  → Patch: bug fixes
```

Automation tools should check module versions before constructing paths, especially in multi-vendor environments where module support varies.

---

## See Also

- netconf
- yang-models
- gnmi-gnoi
- pyats

## References

- RFC 8040 — RESTCONF Protocol: https://datatracker.ietf.org/doc/html/rfc8040
- RFC 8072 — YANG Patch Media Type: https://datatracker.ietf.org/doc/html/rfc8072
- RFC 7951 — JSON Encoding of YANG Data: https://datatracker.ietf.org/doc/html/rfc7951
- RFC 7950 — YANG 1.1: https://datatracker.ietf.org/doc/html/rfc7950
- RFC 8525 — YANG Library: https://datatracker.ietf.org/doc/html/rfc8525
- RESTCONF API Explorer (Cisco): https://developer.cisco.com/docs/ios-xe/restconf/
