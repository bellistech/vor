# Network Automation — Deep Dive

> *Network automation is the discipline of compiling operator intent into device configuration and observing the resulting runtime state, end-to-end, programmatically. Its mathematical foundations are model-theoretic (YANG schemas as constrained type systems), protocol-theoretic (NETCONF / RESTCONF / gNMI as datastore-manipulation algebras), and control-theoretic (closed-loop drift detection and reconvergence). This deep dive treats the protocol stack, the data-modelling language, the telemetry math, and the operational pipelines that bind them as a single coherent system.*

---

## 1. NETCONF protocol

### 1.1 RFC 6241 — protocol layering

NETCONF (Network Configuration Protocol) is layered:

```
+------------------------------+   Layer 4: Content   (YANG-defined config + state data)
|         Content              |
+------------------------------+   Layer 3: Operations (get, get-config, edit-config, ...)
|         Operations           |
+------------------------------+   Layer 2: Messages   (rpc, rpc-reply, notification)
|          Messages            |
+------------------------------+   Layer 1: Transport  (SSH/TLS/BEEP/SOAP — RFC 6242 mandates SSH)
|          Transport           |
+------------------------------+
```

The transport is **stateful and session-oriented**. Each session carries:

- a `session-id` (uint32, assigned by server),
- a `<hello>` capabilities exchange,
- a stream of `<rpc>` / `<rpc-reply>` messages,
- optional `<notification>` messages (RFC 5277, asynchronous),
- a terminating `<close-session>` or transport tear-down.

NETCONF over SSH (RFC 6242) opens subsystem `netconf` on TCP/830. Framing has two modes:

- **End-of-message marker** (legacy): each message ends with the literal sequence `]]>]]>`.
- **Chunked framing** (RFC 6242 §4.2, mandatory if both ends advertise `:base:1.1`): each chunk is `\n#<chunk-size>\n<bytes>` with a terminating `\n##\n`.

ASCII math for chunked framing overhead:

```
overhead_bytes = 3 + ceil(log10(chunk_size)) + 4    // \n# digits \n + final \n##\n
overhead_ratio = overhead_bytes / payload_bytes
```

For a 64 KiB chunk: `overhead = 3 + 5 + 4 = 12 bytes`, ratio ≈ `0.018%`. Framing is essentially free.

### 1.2 Capability exchange (`<hello>`)

The first message both sides emit is `<hello>`. It enumerates URIs the speaker implements:

```
urn:ietf:params:netconf:base:1.0
urn:ietf:params:netconf:base:1.1
urn:ietf:params:netconf:capability:writable-running:1.0
urn:ietf:params:netconf:capability:candidate:1.0
urn:ietf:params:netconf:capability:confirmed-commit:1.1
urn:ietf:params:netconf:capability:rollback-on-error:1.0
urn:ietf:params:netconf:capability:validate:1.1
urn:ietf:params:netconf:capability:startup:1.0
urn:ietf:params:netconf:capability:url:1.0?scheme=file,ftp,sftp
urn:ietf:params:netconf:capability:xpath:1.0
urn:ietf:params:netconf:capability:notification:1.0
urn:ietf:params:netconf:capability:interleave:1.0
urn:ietf:params:netconf:capability:with-defaults:1.0?basic-mode=explicit&also-supported=report-all,trim
http://cisco.com/ns/yang/cisco-ia?module=cisco-ia&revision=...
http://openconfig.net/yang/interfaces?module=openconfig-interfaces&revision=...
```

Vendor-specific module URIs follow the IETF base set. Both peers compute the **intersection** and use only mutually supported features.

### 1.3 Datastore semantics

A NETCONF server exposes one or more named datastores:

| Datastore | Persistence | Mutability | Required? |
|:---|:---|:---|:---:|
| `running` | Live, applied | Direct write only if `:writable-running:` advertised | yes |
| `candidate` | Volatile staging | Edit freely; commit promotes to `running` | if `:candidate:` |
| `startup` | Persisted boot config | Copy from running | if `:startup:` |
| `intended` (NMDA) | Post-template, pre-validated | Read-only | NMDA only |
| `operational` (NMDA) | Live system state | Read-only | NMDA only |

NMDA (RFC 8342, Network Management Datastore Architecture) replaces the old conflation of state and config with a clean separation. The relevant set inclusions are:

```
intended  ⊆  validated(running ∪ defaults)
operational ⊇ applied(intended) ∪ system-generated state
```

Reading `operational` returns *what is actually in the system* including learned routes, ARP cache, system clock — not just the operator-supplied config.

### 1.4 Edit-config and merge semantics

`<edit-config>` carries an `<operation>` attribute on subtree elements:

| Operation | Effect |
|:---|:---|
| `merge` (default) | Existing leaves keep their values unless explicitly set |
| `replace` | The named subtree is *replaced wholesale* — anything not in the request is deleted |
| `create` | Fail if the node already exists |
| `delete` | Fail if the node does not exist |
| `remove` | Delete if present, no-op otherwise |

The `default-operation` attribute on `<edit-config>` itself sets the implicit operation: `merge` (default), `replace`, or `none` (only explicit ops apply). The `error-option` controls failure behaviour: `stop-on-error` (default), `continue-on-error`, or `rollback-on-error` (requires `:rollback-on-error:`).

### 1.5 Commit / rollback timing

The candidate→running transition is governed by `<commit>`:

```
<commit>
  <confirmed/>
  <confirm-timeout>120</confirm-timeout>   <!-- seconds, default 600 -->
  <persist>commit-id-foo</persist>          <!-- survives session loss -->
</commit>
```

The state machine:

```
candidate ──commit──► running (provisional)
   ▲                        │
   │                        │ T < confirm-timeout AND <commit> received
   │                        ▼
   │                    running (confirmed, candidate cleared)
   │
   └── T ≥ confirm-timeout ── revert running to pre-commit snapshot
```

Mathematically the confirmed-commit timer is a *guarded promotion*:

```
running(T) =
  pre_commit_snapshot                        if T < commit_time
  candidate_at_commit_time                   if commit_time ≤ T < commit_time + confirm_timeout
                                              AND no second <commit> received
  candidate_at_commit_time (latched)         if a confirming <commit> arrived in window
  pre_commit_snapshot (rolled back)          if confirm_timeout expired with no confirmation
```

The `<persist>` token detaches the confirmation from the original session; any session can issue `<commit><persist-id>token</persist-id></commit>` before timeout. This is the standard pattern for out-of-band rescue when an operator's connection is severed by their own change (the canonical "commit confirmed, rolled back, can SSH again" recovery).

---

## 2. YANG data model semantics

YANG (RFC 7950 for YANG 1.1; RFC 6020 for YANG 1.0) is a strongly typed, hierarchical schema language that compiles into XML, JSON, and protobuf. It is the schema that NETCONF, RESTCONF, and gNMI all share.

### 2.1 Tree types

A YANG module produces a *schema tree* whose nodes are one of:

| Statement | Carries data? | Multiplicity | Identifier |
|:---|:---:|:---|:---|
| `container` | no (organizational) | 1 | name |
| `leaf` | yes (single value) | 1 | name |
| `leaf-list` | yes (ordered set of values) | 0..N | name |
| `list` | yes (ordered/unordered set of containers) | 0..N | one or more `key` leaves |
| `choice`/`case` | branching, only one case present | XOR | name |
| `anydata`/`anyxml` | opaque | 1 | name |
| `rpc` | invocation node | 1 | name |
| `notification` | event node | 1 | name |
| `action` (1.1) | invocation bound to a node | 1 | name |

```yang
container interfaces {
  list interface {
    key "name";
    leaf name { type string; }
    leaf description { type string; }
    leaf-list ip-address { type inet:ip-address; ordered-by user; }
    container state {
      config false;             // operational only
      leaf oper-status { type enumeration { enum up; enum down; } }
      leaf last-change { type yang:date-and-time; }
    }
  }
}
```

`config true` (default) marks configurable nodes, `config false` marks read-only state. The classification is *recursive*: a `config false` container makes every descendant `config false`.

### 2.2 Constraint statements

YANG constraints are declarative and enforced by the server (and tooling such as `yanglint`, `pyang -p`, `libyang`):

| Statement | Scope | Semantics | Example |
|:---|:---|:---|:---|
| `must "expr"` | instance | XPath 1.0 boolean over data | `must "../mtu >= 576"` |
| `when "expr"` | instance | node only exists if expr true | `when "../type = 'ethernet'"` |
| `range` | type | numeric bounds (incl. unions) | `range "0..4095 \| 4096..16777215"` |
| `length` | string | byte/char count bounds | `length "1..253"` |
| `pattern` | string | XSD regex (RFC 7950 §9.4.6) | `pattern '[A-Za-z0-9_\\-]+'` |
| `mandatory true` | leaf/choice | absence is invalid | applied at validation |
| `min-elements`/`max-elements` | list/leaf-list | cardinality | `min-elements 1` |
| `unique` | list | uniqueness over leaf set | `unique "vlan-id"` |
| `default` | leaf | server-supplied value | `default 1500` |

`when` differs subtly from `must`: `when` *gates the existence* of the node (a `when`-failed node is *not* in the instance tree), whereas `must` is a post-hoc validity assertion (the node exists, but the validator rejects it). This matters for the `with-defaults` algorithm and for tooling that walks the schema looking for live nodes.

### 2.3 Statement-level vs instance-level constraint scope

The distinction is critical:

- **Statement-level** constraints (`range`, `length`, `pattern`, `min-elements`, `max-elements`, `mandatory`) are evaluated **per-leaf** with no awareness of other tree nodes. They are bounded-time and embarrassingly parallelizable.
- **Instance-level** constraints (`must`, `when`, `unique`) are evaluated against the **full data instance** because XPath traverses the tree. Their evaluation cost is at least O(n) per expression and frequently O(n × m) when the predicate cross-references multiple lists.

Practical consequence: a 50-MB candidate datastore with thousands of `must` constraints can take seconds to validate. Vendors compile XPath into specialised matchers (Cisco IOS-XR uses an internal evaluator; libyang does similar) but worst-case complexity remains tied to instance size, not just schema size.

### 2.4 Augments and deviations

`augment` adds nodes from one module into another's tree without modifying the target module's source:

```yang
augment "/if:interfaces/if:interface" {
  when "if:type = 'ianaift:ethernetCsmacd'";
  leaf speed { type uint64; units "bits/second"; }
}
```

Augments are how OpenConfig vendors layer hardware-specific extensions onto a common base. They are also how IETF separates *type-of-thing* modules (`ietf-interfaces`) from *role-of-thing* modules (`ietf-ip` augmenting interface).

`deviation` declares that a server *does not* fully implement a module:

```yang
deviation /if:interfaces/if:interface/if:type {
  deviate replace { type my:vendor-iftype; }
}
deviation /if:interfaces/if:interface/if:enabled {
  deviate not-supported;
}
```

The `deviate` keywords are `add`, `replace`, `delete`, `not-supported`. Deviations are the official mechanism for vendor partial-implementation claims and they appear in capability advertisements as `deviation-modules`.

### 2.5 Identityref and bit types

`identity` declares a hierarchical, openly-extensible enumeration:

```yang
identity transport-protocol;
identity tcp { base transport-protocol; }
identity udp { base transport-protocol; }
identity quic { base transport-protocol; }   // can be added in a later module

leaf transport { type identityref { base transport-protocol; } }
```

`bits` encodes a fixed set of named flags packed into an integer:

```yang
typedef tcp-flags {
  type bits {
    bit fin { position 0; }
    bit syn { position 1; }
    bit rst { position 2; }
    bit psh { position 3; }
    bit ack { position 4; }
    bit urg { position 5; }
  }
}
```

---

## 3. YANG → JSON / XML encoding

### 3.1 RFC 6020 — XML encoding (NETCONF native)

XML follows the NETCONF wire format. Each YANG container is an XML element; lists become repeated elements; leaves are text-content elements. The module's namespace becomes the XML namespace:

```yang
module example { namespace "urn:example:net"; prefix ex; ... }
```

```xml
<interfaces xmlns="urn:example:net">
  <interface>
    <name>eth0</name>
    <description>uplink</description>
    <enabled>true</enabled>
  </interface>
</interfaces>
```

Boolean leaves use the literals `true` / `false` (NOT `1`/`0`). Empty leaves serialize as `<leaf/>`. List entries appear in document order; the `key` leaf must precede sibling content within each entry per canonical XML.

### 3.2 RFC 7951 — JSON encoding

JSON encoding for YANG normalises namespacing into module-prefixed JSON keys:

- Top-level (or namespace-changing) members are `"<module-name>:<node-name>"`.
- Children inherit the parent's namespace unless they cross modules.
- Lists become JSON arrays of objects.
- Leaves become JSON values; numeric types ≤32-bit can be JSON numbers, but **64-bit ints serialize as JSON strings** to avoid IEEE-754 precision loss.
- `empty` leaves serialise as `[null]` (a single-element array with null).
- `bits` serialise as a single space-separated string: `"syn ack"`.

Example mapping:

```json
{
  "ietf-interfaces:interfaces": {
    "interface": [
      {
        "name": "eth0",
        "description": "uplink",
        "enabled": true,
        "ietf-ip:ipv4": {
          "address": [
            { "ip": "10.0.0.1", "prefix-length": 24 }
          ]
        }
      }
    ]
  }
}
```

The `ietf-ip:` prefix appears because `ipv4` is augmented in by a different module.

### 3.3 with-defaults algorithm (RFC 6243 / RFC 8040 §4.8.9)

Default values complicate "what does the server actually return". Modes:

| Mode | Behaviour |
|:---|:---|
| `report-all` | Every leaf with a default is emitted, whether explicitly set or not |
| `trim` | Leaves equal to the schema default are *omitted* |
| `explicit` | Only explicitly-set leaves are emitted (matches the configuration intent) |
| `report-all-tagged` | Like `report-all` but defaults are tagged with `wd:default="true"` |

The algorithm for `report-all-tagged` is:

```
for leaf L in schema_tree:
    if L has default D:
        if L is set explicitly to V:
            emit(L, V)
        else:
            emit(L, D, attr={"wd:default": "true"})
    else:
        if L is set:
            emit(L, value(L))
```

Capability advertisement: `?basic-mode=explicit&also-supported=report-all,trim` informs the client which modes the server accepts.

### 3.4 Default value resolution rules

When a leaf has a `default`, the value applies only if **no ancestor `when`-statement is false** and the leaf is reachable in the instance tree. Defaults do NOT apply inside `case` statements unless that case is selected. This is why `when`/`choice`/`case` and `default` interact non-trivially — and why YANG validators must walk `when` predicates *before* applying defaults.

---

## 4. RESTCONF protocol

### 4.1 RFC 8040 — HTTP mapping

RESTCONF provides RESTful HTTP access to NETCONF datastores using the same YANG modules. Resource paths are derived deterministically from YANG schema:

| YANG schema fragment | RESTCONF URI |
|:---|:---|
| `/interfaces/interface=eth0` | `/restconf/data/ietf-interfaces:interfaces/interface=eth0` |
| `/interfaces/interface=eth0/description` | `.../interface=eth0/description` |
| `/interfaces/interface=eth0/ipv4/address=10.0.0.1` | `.../interface=eth0/ietf-ip:ipv4/address=10.0.0.1` |

URI escaping rules (RFC 8040 §3.5.3) require commas in list keys to be percent-encoded as `%2C`, and `/` inside a key value as `%2F`.

### 4.2 HTTP methods

```
GET     /restconf/data/<path>           -- read configured + state
GET     /restconf/data/<path>?content=config   -- config only
GET     /restconf/data/<path>?content=nonconfig -- state only
POST    /restconf/data/<path>           -- create child resource
PUT     /restconf/data/<path>           -- replace
PATCH   /restconf/data/<path>           -- merge (or YANG Patch with media type application/yang-patch+json)
DELETE  /restconf/data/<path>           -- remove
HEAD                                    -- metadata only
OPTIONS                                 -- discover allowed methods
```

Datastore endpoints (RFC 8527 NMDA):

```
/restconf/ds/ietf-datastores:running
/restconf/ds/ietf-datastores:candidate
/restconf/ds/ietf-datastores:operational
/restconf/ds/ietf-datastores:intended
```

### 4.3 Query parameters

| Parameter | Semantics | Example |
|:---|:---|:---|
| `depth` | Limit tree depth: `1..65535` or `unbounded` | `?depth=3` |
| `fields` | Return only listed leaves: comma-separated paths | `?fields=name;state/oper-status` |
| `content` | `config`, `nonconfig`, `all` | `?content=nonconfig` |
| `with-defaults` | `report-all`, `trim`, `explicit`, `report-all-tagged` | `?with-defaults=trim` |
| `filter` | XPath subtree filter (NETCONF-style) | `?filter=interface[name='eth0']` |
| `start-time`/`stop-time` | Replay window for streams | `?start-time=2026-04-27T00:00:00Z` |
| `insert`/`point` | Ordered-list insertion control | `?insert=before&point=...` |

The `fields` parameter has its own mini-grammar (RFC 8040 §4.8.3):

```
fields      = node-selector *( ";" node-selector )
node-selector = path *( "(" sub-fields ")" )
sub-fields  = node-selector *( ";" node-selector )
path        = api-identifier *( "/" api-identifier )
```

Example: `fields=interface(name;state(oper-status;last-change))` returns only `name` plus the two named state leaves of every interface.

### 4.4 Response media types

```
application/yang-data+json    -- RFC 7951 JSON
application/yang-data+xml     -- RFC 6020 XML
application/yang-patch+json   -- RFC 8072 ordered patch
```

The `Accept` header negotiates encoding:

```http
GET /restconf/data/ietf-interfaces:interfaces HTTP/1.1
Host: device.example.com
Accept: application/yang-data+json
Authorization: Basic YWRtaW46YWRtaW4=
```

### 4.5 with-defaults algorithm under RESTCONF

Identical to NETCONF: the `?with-defaults=` query parameter is a direct mapping of the NETCONF `<with-defaults>` element. Servers MUST advertise their default mode in `/restconf/yang-library-version` capabilities. Clients that omit the parameter receive the server's `basic-mode`.

---

## 5. gNMI

### 5.1 gRPC + Protobuf foundation

gNMI is a gRPC service defined in `gnmi.proto`. Transport is HTTP/2 (typically TCP/9339), encoding is Protobuf, authentication is mTLS or username/password embedded in gRPC metadata.

```protobuf
service gNMI {
  rpc Capabilities(CapabilityRequest) returns (CapabilityResponse);
  rpc Get(GetRequest) returns (GetResponse);
  rpc Set(SetRequest) returns (SetResponse);
  rpc Subscribe(stream SubscribeRequest) returns (stream SubscribeResponse);
}

message Path {
  string origin = 2;            // e.g. "openconfig"
  repeated PathElem elem = 3;
}

message PathElem {
  string name = 1;
  map<string, string> key = 2;  // list keys
}

message TypedValue {
  oneof value {
    string string_val = 1;
    int64 int_val = 2;
    uint64 uint_val = 3;
    bool bool_val = 4;
    bytes bytes_val = 5;
    float float_val = 6;        // deprecated
    Decimal64 decimal_val = 7;
    ScalarArray leaflist_val = 8;
    bytes any_val = 9;          // protobuf.Any
    bytes json_val = 10;        // legacy
    bytes json_ietf_val = 11;   // RFC 7951
    bytes ascii_val = 12;
    bytes proto_bytes = 13;
  }
}
```

### 5.2 Subscription modes (the cardinal product)

`SubscriptionList.mode` selects the *outer* mode:

| Outer mode | Semantics |
|:---|:---|
| `STREAM` | Long-lived stream; per-path subscription mode applies |
| `ONCE` | Server returns a single sync response and closes |
| `POLL` | Server returns a snapshot every time the client sends a `Poll` message |

`Subscription.mode` selects the *inner* mode (only meaningful when outer = `STREAM`):

| Inner mode | Semantics |
|:---|:---|
| `SAMPLE` | Periodic snapshot every `sample_interval` ns |
| `ON_CHANGE` | Push only when leaf value changes (server-side dedup) |
| `TARGET_DEFINED` | Server picks the optimal mode per leaf |

The full cross-product:

```
ONCE     × {SAMPLE, ON_CHANGE, TARGET_DEFINED}    -- inner ignored, single shot
POLL     × {SAMPLE, ON_CHANGE, TARGET_DEFINED}    -- inner ignored, snapshot per Poll
STREAM   × SAMPLE                                  -- periodic
STREAM   × ON_CHANGE                               -- event-driven
STREAM   × TARGET_DEFINED                          -- server's choice per leaf
```

### 5.3 Sample interval math

`sample_interval` is in nanoseconds (uint64). For `SAMPLE` mode the server emits an update tuple at every `sample_interval` boundary regardless of value change.

```
updates_per_second_per_leaf = 10^9 / sample_interval_ns
```

For 5-second sampling: `10^9 / 5_000_000_000 = 0.2 updates/sec`.

`heartbeat_interval` (also nanoseconds) applies *primarily* to `ON_CHANGE`: even when nothing changes, the server emits a synthetic update at the heartbeat boundary so the client can distinguish "nothing changed" from "stream is broken".

```
heartbeat_overhead_bps = (payload_bytes * 8) / (heartbeat_interval_ns / 10^9)
```

For ON_CHANGE on a leaf with `heartbeat_interval = 60s` and 200-byte payload: `1600 / 60 = 26.7 bps` of synthetic traffic per leaf.

`suppress_redundant=true` (a per-Subscription field) tells the server: even in SAMPLE mode, skip emitting an update if the value is unchanged from the previous sample. Combined with `heartbeat_interval`, this collapses bandwidth for rarely-changing counters.

### 5.4 Set transactionality

`Set` carries any combination of `delete[]`, `replace[]`, `update[]` lists. The server MUST treat the entire request as atomic: either all paths apply or none do. Order within the request is `delete → replace → update`. This means clients can express idempotent set-deltas without compound transactions.

```protobuf
message SetRequest {
  Path prefix = 1;
  repeated Path delete = 2;
  repeated Update replace = 3;
  repeated Update update = 4;
  repeated google.protobuf.Any extension = 5;
}
```

### 5.5 Origin and path encoding

`origin` disambiguates which schema model a path uses. Standard values:

```
""            -- legacy / default
"openconfig"  -- OpenConfig YANG models
"rfc7951"     -- IETF YANG models, JSON-encoded paths
"cli"         -- vendor CLI text (rare, deprecated)
```

This matters when a device exposes both OpenConfig and vendor-native schemas: the same logical leaf appears at different paths in different origins.

---

## 6. Streaming telemetry math

### 6.1 Cardinality explosion

The fundamental scaling formula:

```
total_streams = N_devices × M_metrics × P_paths_per_metric
```

For a 1000-device fabric, 50 metrics, average 24 interfaces (paths) per metric:

```
streams = 1000 × 50 × 24 = 1_200_000
```

This is the active-stream count the collector must hold open. Each stream consumes:

- one HTTP/2 stream ID,
- per-stream flow-control window,
- a sequence-number cursor for dedup.

Multiplexing 1.2M streams over a single gRPC channel exhausts the HTTP/2 stream-id space (2^31 / 2 = ~1B per direction, but practical settings cap at `MAX_CONCURRENT_STREAMS = 100` by default — `gNMI` clients tune this up to 1000+ via `SETTINGS`).

### 6.2 Sample rate vs storage

Bytes per second per stream:

```
bps_per_stream = sample_rate_hz × payload_bytes
```

At 5-second sampling, 200-byte payload (typical interface counters):

```
bps_per_stream = 0.2 × 200 = 40 bytes/sec = 320 bps
```

For 1.2M streams:

```
aggregate_bps = 1_200_000 × 320 = 384_000_000 bps ≈ 384 Mbps
```

Storage cost over a 7-day retention:

```
total_bytes = aggregate_bps / 8 × 86400 × 7
            = 48_000_000 × 604_800
            = 29_030_400_000_000 bytes
            ≈ 29 TB raw
```

Compression (gRPC-internal gzip is ~6×; downstream Prometheus TSDB ~10×) reduces this to ~3 TB.

### 6.3 Downsampling pyramid

The standard four-tier retention pyramid:

```
Tier      Resolution    Retention    Bytes/leaf/year
raw       5 sec         24 hours     ~6.3 GB    (5s × 17_280 samples × 200B)
1-min     60 sec        7 days       ~12 MB     (60s × 10_080 × 200B)
5-min     300 sec       30 days      ~28 MB
1-hour    3600 sec      1 year       ~14 MB
```

Total per-leaf-year footprint after downsampling: ~6.4 GB (raw dominates because of the 24-hour retention; tune that to taste).

The downsampler is a streaming aggregation:

```
for each window W of duration interval:
    counter_max(W)   = max(samples in W)
    counter_min(W)   = min(samples in W)
    counter_avg(W)   = sum(samples in W) / |samples in W|
    counter_last(W)  = last sample in W
    rate(W)          = (counter_last(W) - counter_first(W)) / |W|
```

Counters MUST use `last - first` over the window, never `avg`, because counters are monotonic — the average of an increasing series is meaningless for rate computation.

### 6.4 Subscription QoS budget

Per-collector budget formula:

```
collector_capacity_bps = NIC_bps × 0.7      // leave headroom
max_streams = collector_capacity_bps / avg_bps_per_stream
```

For a 10 GbE collector running at 70%: `7_000_000_000 / 320 = 21_875_000` streams. The bottleneck is rarely the wire; it is decode CPU (protobuf parse) and the downstream TSDB ingest rate.

---

## 7. OpenConfig models

### 7.1 Common-base + augments pattern

OpenConfig models are organised as a *base layer* and a *vendor augment layer*:

```
oc-if  (openconfig-interfaces)
   ├── interfaces/interface/{name, type, description, enabled, ...}
   └── augmented by:
       ├── oc-eth   (openconfig-if-ethernet)    -- Ethernet-specific leaves
       ├── oc-vlan  (openconfig-vlan)           -- VLAN config
       ├── oc-aggr  (openconfig-if-aggregate)   -- LAG-specific
       └── vendor-specific extensions (Cisco, Arista, Juniper)
```

The common base lets a script that targets `oc-if` work across vendors; vendor-specific behaviour lives in clearly-named augments that a multi-vendor automation can detect and skip or specialise on.

### 7.2 Semantic vs syntactic match

Semantic match: two devices report the same logical quantity. Syntactic match: two devices use literally the same YANG path. OpenConfig aims for both, but vendor implementations diverge. Common failure modes:

- Device A reports interface admin-state under `openconfig-interfaces:interfaces/interface/config/enabled` (boolean).
- Device B reports it under the same path *but as a string* `"UP"` / `"DOWN"` (vendor deviation: `deviate replace { type string; }`).

The mitigation is a *normalisation layer* that maps each device's response into a canonical model — the unifying value-add of NAPALM and Nornir-Netbox-Driver style abstraction.

### 7.3 OpenConfig vs IETF coverage

| Domain | OpenConfig | IETF YANG |
|:---|:---|:---|
| Interfaces | `openconfig-interfaces` (rich) | `ietf-interfaces` (minimal) |
| Routing | `openconfig-network-instance` | `ietf-routing` + protocol modules |
| BGP | `openconfig-bgp` | `ietf-bgp` (later, less coverage) |
| MPLS | `openconfig-mpls` | scattered IETF modules |
| Telemetry | `openconfig-telemetry` | none (gNMI is the de-facto) |
| AAA | `openconfig-aaa` | scattered |

Vendor convergence is strongest on OpenConfig because it was designed with multi-vendor interop as the explicit goal. IETF modules tend to lag in adoption.

---

## 8. Idempotency math

### 8.1 Declarative model

A declarative system computes:

```
diff = desired_state - current_state          // set difference, per-leaf
new_state = apply(current_state, diff)
```

The fixed point is reached when `diff = ∅`. Convergence:

```
state_k+1 = apply(state_k, desired - state_k)
state_{k+1} = state_k  ⟺  desired = state_k       (fixed point)
```

For idempotent `apply`:

```
apply(apply(s, d), d) = apply(s, d)
```

This is the formal definition of idempotency: applying `d` repeatedly produces the same state as applying once.

### 8.2 Imperative model

An imperative system sequences side-effects:

```
state_n = op_n(op_{n-1}(... op_1(state_0)))
```

Order matters because operations are *non-commutative*:

```
op_a ∘ op_b ≠ op_b ∘ op_a    in general
```

Example: removing an IP address before the routing protocol that uses it succeeds; the reverse order leaves an orphaned route reference. This is why imperative tools (raw CLI, expect-style scripts) need explicit ordering and rollback paths whereas declarative tools (Terraform, NSO, Crossplane) compute the dependency DAG automatically.

### 8.3 Drift detection

Drift is the per-leaf delta between *desired* and *operational*:

```
drift = { (path, desired[path], operational[path])
          for each path in desired
          if desired[path] ≠ operational[path] }

|drift| = number of mismatched leaves
drift_severity = sum(weight(leaf) for leaf in drift)
```

`weight(leaf)` is policy-defined: a wrong description is weight 1, a wrong BGP neighbour AS is weight 100. The reconvergence loop:

```
while True:
    operational = collect_telemetry()
    drift = diff(desired, operational)
    if drift:
        if drift_severity > threshold:
            page_human()
        else:
            apply(diff)
    sleep(interval)
```

---

## 9. Source of Truth

### 9.1 NetBox / Nautobot data model

The Source of Truth (SoT) is a database of *intended* configuration used to derive device config. Core entities:

| Entity | Cardinality vs Device | Role |
|:---|:---|:---|
| Site | Sites contain devices | Geography |
| Tenant | Multi-tenancy boundary | RBAC + isolation |
| Device | One per chassis | Physical inventory |
| Interface | Many per device | Port inventory |
| IP Address | Many per interface | L3 binding |
| Prefix | Aggregates for IPAM | Allocation pool |
| VLAN | Scoped to site or global | L2 segmentation |
| Tag | Many-to-many label | Cross-cutting concern |
| Config Context | Hierarchical key-value | Render-time data |

### 9.2 IPAM cardinality math

A `/24` IPv4 prefix provides:

```
hosts_per_prefix = 2^(32 - prefix_length) - 2     // -2 for network and broadcast
                 = 2^8 - 2 = 254
```

Subtract the gateway address and any reservations:

```
allocatable = hosts_per_prefix - reserved_count
```

For a NetBox `/24` with a `.1` gateway and 5 reserved addresses: `254 - 1 - 5 = 248` allocatable. Utilisation:

```
utilisation% = allocated_count / hosts_per_prefix × 100
```

NetBox's hierarchy lets you compute aggregate utilisation:

```
utilisation(parent) = sum(allocated leaves under parent) / sum(host capacity of leaves)
```

Whether parent prefixes count toward host capacity depends on the `is_pool` flag — a *container* prefix is the sum of its children only; a *pool* prefix counts the prefix itself.

### 9.3 VLAN scope semantics

NetBox VLANs have a `scope` field that defines uniqueness:

| Scope | Uniqueness boundary |
|:---|:---|
| Global (no group) | VLAN ID unique across entire NetBox |
| Site | VLAN ID unique within site |
| Cluster Group | Unique within VM cluster group |
| Region (custom) | Unique within region |

The constraint is enforced as `UNIQUE(scope, vid)`. Cross-site reuse is intentional — VLAN 10 in datacenter A is unrelated to VLAN 10 in datacenter B unless the topology bridges them via VXLAN/EVPN.

### 9.4 Tag inheritance rules

Tags on a `Device` do **not** automatically propagate to its `Interface` children. NetBox treats tags as property sets attached to a specific object class. Render templates must walk the parent chain explicitly:

```python
device_tags = set(device.tags.all())
interface_tags = set(interface.tags.all())
effective_tags = device_tags | interface_tags     # explicit union
```

Nautobot's *Relationships* feature provides an alternative inheritance model with declared semantics, but for plain tag-based filtering you compute inheritance at render time.

---

## 10. CI/CD for network

### 10.1 Batfish symbolic execution

Batfish parses configurations from many vendors into a vendor-agnostic **Vendor-Independent Model** (VIM), then computes:

- the data plane (FIBs) implied by the configurations,
- routing protocol convergence via simulated route-redistribution,
- reachability between any two endpoints via symbolic execution.

Symbolic execution uses **Binary Decision Diagrams (BDDs)** to represent packet headers as logical formulas over their bits:

```
header_bits = src_ip(32) + dst_ip(32) + src_port(16) + dst_port(16) + proto(8) + tcp_flags(8)
            = 112 bits
```

BDD complexity is bounded by the number of internal nodes, which in the worst case is `O(2^V)` for `V` variables but is dramatically reduced by:

- variable ordering (good ordering → polynomial; bad ordering → exponential),
- structural decomposition (per-prefix, per-ACL),
- canonical form caching across many evaluations.

Batfish queries:

```
reachability(src=10.0.0.0/8, dst=192.168.0.0/16, proto=tcp, dst_port=22)
            → set of header tuples that succeed,
              set of header tuples that fail (with offending node),
              ACL/route entries responsible.
```

The complexity bound `O(2^V)` worst case is mitigated in practice because most networks have *structurally regular* policies — large prefix groups behave identically. Batfish exploits this regularity automatically.

### 10.2 Containerlab topology compilation

Containerlab consumes a YAML topology file and compiles it into a Linux network namespace + container graph:

```yaml
name: lab1
topology:
  nodes:
    spine1: { kind: nokia_srlinux, image: ghcr.io/nokia/srlinux:23.10 }
    leaf1:  { kind: arista_ceos, image: ceos:4.31.0F }
    leaf2:  { kind: arista_ceos, image: ceos:4.31.0F }
  links:
    - endpoints: [ spine1:e1-1, leaf1:eth1 ]
    - endpoints: [ spine1:e1-2, leaf2:eth1 ]
```

The graph compilation pipeline:

```
parse YAML → validate against schema →
  resolve image tags → docker pull (parallel) →
  create per-node netns + veth pairs →
  apply node-startup config → mark "ready" →
  emit `clab inspect` graph
```

Each link compiles to one `veth` pair joining two namespaces. Node count `N` and link count `L` map directly to `2L` veth interfaces and `N` containers.

### 10.3 Suzieq rolling state collection

Suzieq is a multi-vendor state collector that polls devices and writes Parquet to a normalised schema. The rolling collection model:

```
poll_interval = 60s        // typical
per_device_collect_time = 2 to 5 seconds
parallel_workers = 16
worker_capacity = 60s × 16 / avg_collect_time = ~256 devices per minute
```

Schema is designed for diff-friendly time-series query: each row has `(timestamp, namespace, hostname, table, ...)` and Suzieq queries are SQL over Parquet via `pandas`. This means CI tests can ask "did any BGP neighbour change state in the last hour" as a single SQL query.

### 10.4 Network CI pipeline shape

```
git push
  ↓
lint (yamllint, ansible-lint, pyang, yanglint)
  ↓
unit tests (jinja render, hierarchy diff)
  ↓
syntactic validation (Batfish parse)
  ↓
semantic validation (Batfish reachability queries)
  ↓
integration test (Containerlab spin-up → Suzieq snapshot → assert)
  ↓
canary deploy (1 device)
  ↓
soak (telemetry watch for N minutes)
  ↓
fleet rollout (batched)
```

Each stage has a measurable budget. Failure at any stage halts promotion and records the offending diff.

---

## 11. Intent-based networking

### 11.1 Pipeline

```
intent      ──compile──►  policy
policy      ──compile──►  device-config
device-config ──apply──►  running state
running-state ──telemetry──►  observed
observed    ──diff──►     drift
drift       ──reconcile──► policy adjustment (closed loop)
```

The compiler has the same structure as a multi-pass code compiler: front-end (parse intent), middle-end (optimise/normalise), back-end (vendor-specific config emit).

### 11.2 Intent language example (Cisco NSO YDK / IBN style)

```yang
service site-vpn {
  list site {
    key "name";
    leaf name { type string; }
    leaf-list pe-router { type leafref { path "/dev:devices/dev:device/dev:name"; } }
    container reachability {
      list permitted {
        key "to-site";
        leaf to-site { type leafref { path "../../../name"; } }
      }
    }
  }
}
```

A higher-level *intent* such as "Site A may reach Site B and Site C, no others" compiles into the corresponding L3VPN VRF imports/exports + RT lists at every PE.

### 11.3 Closed-loop reconvergence math

Reconvergence cycle time:

```
T_loop = T_telemetry + T_diff + T_decide + T_compile + T_apply + T_propagate
```

Typical values (production fabric):

```
T_telemetry  =  5s     // gNMI sample interval
T_diff       =  1s     // controller diff engine
T_decide     =  2s     // policy engine + rate limit
T_compile    =  3s     // intent → device-config
T_apply      = 10s     // gNMI Set + commit
T_propagate  =  5s     // protocol convergence (BGP, OSPF)
T_loop       = 26s
```

Drift that lasts less than `T_loop` is invisible; drift that exceeds it is detectable. Tuning the loop is a balance between fabric chatter (low T_loop → constant churn) and operational latency.

### 11.4 Graph-coloring conflict detection

When two intents target overlapping resources (same VLAN, same prefix, same ACL slot), a graph-coloring algorithm decides which compiles first or whether they merge. Vertices = intents; edges = resource conflicts; colors = priority levels.

```
G = (V, E)
V = set of active intents
E = { (i, j) : intent_i.resources ∩ intent_j.resources ≠ ∅ }
```

The chromatic number `χ(G)` is the minimum priority levels required to schedule all intents non-conflictingly. Intents in the same color class compile in parallel; different classes serialise. Real systems use heuristic colorings (greedy by intent priority) because optimal graph coloring is NP-hard.

---

## 12. Ansible network connection internals

### 12.1 Persistent connections (`network_cli`)

Ansible's network modules use a persistent connection plugin that opens **one SSH session per host per play** and reuses it across all tasks. Alternatives:

| Plugin | Transport | Use |
|:---|:---|:---|
| `ansible.netcommon.network_cli` | SSH + paramiko | CLI-based devices |
| `ansible.netcommon.netconf` | SSH NETCONF subsystem | NETCONF-capable devices |
| `ansible.netcommon.httpapi` | HTTP / HTTPS | RESTCONF / NX-API / Junos REST |
| `ansible.netcommon.libssh` | libssh2 | faster CLI alternative |

The connection daemon (`ansible-connection`) lives for the duration of the play, multiplexing all task RPCs over a single transport.

### 12.2 Timeout cascade

Three layers, each must be larger than the next:

```
play_timeout    >    task_timeout    >    connect_timeout
```

Variables:

```yaml
# ansible.cfg or playbook vars
ansible_command_timeout: 30          # per-RPC
ansible_connect_timeout: 30          # initial SSH/NETCONF handshake
ansible_persistent_command_timeout: 30
ansible_persistent_connect_timeout: 30
```

A misordered cascade silently truncates: if `task_timeout = 60` but `connect_timeout = 90`, the task fails before the connection even completes its handshake on a slow device.

### 12.3 Parallelism math

```
forks = number of parallel workers (default 5)
serial = max hosts per batch (default = all)
effective_parallelism = min(forks, num_hosts)
batch_count = ceil(num_hosts / serial)
total_time = batch_count × max_task_time_per_host / effective_parallelism
```

For 1000 devices with `forks: 50, serial: 100, max_task_time_per_host: 30s`:

```
batch_count = ceil(1000 / 100) = 10
per_batch_workers = min(50, 100) = 50
per_batch_time = 100 × 30 / 50 = 60s
total = 10 × 60 = 600s
```

Add per-host retries:

```
effective_rate = forks × num_hosts / (1 + retry_factor)
retry_factor = avg_retries × retry_cost_ratio
```

If 5% of tasks retry once at 2× the original cost: `retry_factor = 0.05 × 2 = 0.1`. `effective_rate` drops by ~9%.

### 12.4 Strategy plugins

| Strategy | Behaviour |
|:---|:---|
| `linear` (default) | Lockstep: all hosts complete task `k` before any starts `k+1` |
| `free` | Hosts run independently; total time = slowest host's serial path |
| `host_pinned` | Per-host serial execution but parallelism across hosts |
| `debug` | Interactive debugging on failure |

For network plays, `linear` is preferred because the lockstep ensures e.g. both endpoints of an L2 trunk are configured before either attempts to come up.

---

## 13. NAPALM unified driver layer

### 13.1 Abstraction cost

NAPALM (Network Automation and Programmability Abstraction Layer with Multivendor) provides one Python API across IOS, IOS-XR, NX-OS, EOS, JunOS. Cost dimensions:

- per-vendor parsing (CLI scrape, JSON-RPC, NETCONF) is bespoke,
- the intersection of features across vendors is the **minimum supported subset**,
- vendor-specific richness is exposed via per-driver `cli()` escape hatch.

Common ops minimum subset:

| Method | Returns |
|:---|:---|
| `get_facts()` | hostname, vendor, model, OS version, uptime |
| `get_interfaces()` | per-interface admin/oper state, speed, MAC |
| `get_interfaces_counters()` | tx/rx packets, errors, discards |
| `get_interfaces_ip()` | per-interface IP/prefix (v4 + v6) |
| `get_arp_table()` | (interface, mac, ip, age) tuples |
| `get_mac_address_table()` | (vlan, mac, interface, type) tuples |
| `get_route_to(destination)` | longest-prefix lookup with protocol metadata |
| `get_bgp_neighbors()` | per-vrf, per-AF neighbour state |
| `get_lldp_neighbors_detail()` | LLDP TLVs, per-port |
| `get_environment()` | temp, power, fan, CPU, memory |

### 13.2 Configuration ops (commit pipeline)

```python
device.open()
device.load_merge_candidate(filename="diff.cfg")    # or load_replace_candidate
device.compare_config()                              # show what would change
if user_approves(diff):
    device.commit_config()
else:
    device.discard_config()
device.close()
```

The driver translates `load_merge_candidate` into vendor-native ops:

| Vendor | Mechanism |
|:---|:---|
| IOS | `tftp` upload + `copy tftp running` (no real candidate, simulated diff) |
| IOS-XR | Native commit + rollback |
| NX-OS | Checkpoint/rollback |
| JunOS | Candidate + commit confirmed |
| EOS | Session-based config |

For drivers without a real candidate (legacy IOS), NAPALM stages a snapshot, generates the diff against the running config, and runs the change in a single `configure terminal` block. Rollback uses the snapshot. This is *strictly* less safe than a vendor-native candidate because there is no atomic transaction.

### 13.3 Per-vendor parsing complexity

```
parse_complexity ∝ output_lines × parser_state_machine_states
```

For `show ip route` on a 10K-route device:

- IOS: ~25K lines (multi-line per route with NHs), regex-based parser, ~1–2s.
- IOS-XR: structured JSON via `show ... json`, ~50ms parse.
- NX-OS: structured JSON via `show ... | json native`, ~100ms.
- JunOS: structured XML via `show ... | display xml`, ~80ms.

Structured outputs are ~20× faster than CLI scraping.

---

## 14. YANG tooling

### 14.1 pyang transformation pipeline

`pyang` is the IETF reference parser. Pipeline:

```
.yang  ──parse──►  AST  ──validate──►  model  ──transform──►  output
```

Output formats:

| Format | Flag | Use |
|:---|:---|:---|
| Tree | `-f tree` | Human-readable schema |
| Sample XML | `-f sample-xml-skeleton` | NETCONF request scaffold |
| JSON Schema | `-f jsonxsl` | YANG → XSL → JSON instance gen |
| HTML | `-f jstree` | Browser tree |
| UML | `-f uml` | Diagrams |
| YANG | `-f yang` | Canonical reformat |

Validation pipeline:

```
1. lex + parse YANG into AST
2. resolve imports and includes
3. resolve groupings (uses) into expanded form
4. validate constraints (range, pattern, must syntax)
5. resolve augments and deviations
6. emit validated schema
```

### 14.2 yanglint validation rules

`yanglint` (libyang) validates *data instances* against schemas:

```
yanglint -f json -t config -p yang-models/ ietf-interfaces.yang ietf-ip.yang config.json
```

Validations:

- type: range, length, pattern, enum
- structural: missing mandatory leaves, mis-keyed list entries
- referential: leafref targets exist, instance-identifiers resolve
- constraints: must / when expressions evaluate true on the instance
- defaults: applied per `with-defaults` mode

Exit codes: 0 valid; 1 schema/data error.

### 14.3 libyang2 vs libyang

libyang1 was the original C implementation; libyang2 introduces:

- NMDA-aware datastores (operational, intended, ds-specific defaults),
- per-datastore validation policy,
- better extension hook API (custom XPath functions, custom types),
- improved memory layout for large datastores (10–100M leaves),
- pyang/clixon/sysrepo all migrated.

Migration concern: libyang2 is API-incompatible with libyang1. Tools that embed libyang must update their wrappers.

---

## 15. Worked examples

### 15.1 NETCONF candidate-commit-confirmed full sequence (60s rollback)

```
Client                                    Device
  |                                          |
  |--- TCP/830, SSH auth, hello ---→         |
  |←-- hello (capabilities) ---|             |
  |--- <lock target=candidate> ---→          |
  |←-- <ok/> ---|                            |
  |--- <discard-changes/> ---→               |  (clean candidate)
  |←-- <ok/> ---|                            |
  |--- <edit-config target=candidate>        |
  |       merge; ietf-interfaces             |
  |       set eth0/description "uplink"     |
  |       set eth0/enabled true             |
  |    </edit-config> ---→                   |
  |←-- <ok/> ---|                            |
  |--- <validate><source>candidate</source></validate> ---→ |
  |←-- <ok/> ---|                            |
  |--- <commit><confirmed/>                  |
  |          <confirm-timeout>60</confirm-timeout>          |
  |          <persist>commit-2026-04-27-001</persist>       |
  |    </commit> ---→                        |
  |←-- <ok/> ---|                            |  (provisional running)
  |                                          |
  |  ... operator validates the change ...   |
  |  ... if ok, send confirming commit ...   |
  |                                          |
  |--- <commit><persist-id>commit-2026-04-27-001</persist-id></commit> ---→ |
  |←-- <ok/> ---|                            |  (running latched)
  |                                          |
  |--- <unlock target=candidate> ---→        |
  |←-- <ok/> ---|                            |
  |--- <close-session/> ---→                 |
```

Failure path: if the operator's session dies after the initial `<commit><confirmed/>` and before the confirming commit, after 60 seconds the device automatically reverts. The operator can reconnect (via the now-restored config) and try again.

### 15.2 gNMI Subscribe SAMPLE 5s with ON_CHANGE fallback

```protobuf
SubscribeRequest {
  subscribe: SubscriptionList {
    prefix: { origin: "openconfig", elem: [ {name: "interfaces"} ] }
    subscription: [
      {
        path: { elem: [ {name: "interface", key: {"name": "Ethernet1"}}, {name: "state"}, {name: "counters"} ] }
        mode: SAMPLE
        sample_interval: 5_000_000_000        // 5s in ns
        suppress_redundant: false
        heartbeat_interval: 0
      },
      {
        path: { elem: [ {name: "interface", key: {"name": "Ethernet1"}}, {name: "state"}, {name: "oper-status"} ] }
        mode: ON_CHANGE
        heartbeat_interval: 60_000_000_000    // 60s
      }
    ]
    mode: STREAM
    encoding: JSON_IETF
    updates_only: false
  }
}
```

Server sends:

```
SubscribeResponse: Notification {
  timestamp: 1714176001_000_000_000
  prefix: /interfaces
  update: [
    { path: /interface[name=Ethernet1]/state/counters, val: { json_ietf_val: {...} } }
  ]
}
SubscribeResponse: Notification {
  timestamp: 1714176001_000_000_000
  prefix: /interfaces
  update: [
    { path: /interface[name=Ethernet1]/state/oper-status, val: { string_val: "UP" } }
  ]
}
SubscribeResponse: sync_response = true       // initial snapshot complete
SubscribeResponse: Notification { timestamp: 1714176006_..., update: [counters @ 5s] }
SubscribeResponse: Notification { timestamp: 1714176011_..., update: [counters @ 10s] }
... oper-status only emits on change OR every 60s heartbeat ...
SubscribeResponse: Notification { timestamp: 1714176061_..., update: [oper-status heartbeat] }
SubscribeResponse: Notification { timestamp: 1714176083_..., update: [oper-status DOWN — change!] }
```

### 15.3 Batfish reachability query: ACL hole detection

```python
import pybatfish.client.session as bfs
from pybatfish.question.question import load_questions

session = bfs.Session(host="batfish.local")
session.set_network("prod")
session.init_snapshot("snapshots/2026-04-27", name="2026-04-27")
load_questions()

# Question: from any host in 10.0.0.0/8 trying TCP/22 to any host in 192.168.0.0/16,
# which packet headers succeed (potential SSH exposure across zones)?
result = session.q.reachability(
    pathConstraints={"startLocation": "@enter(10.0.0.0/8)"},
    headers={"srcIps": "10.0.0.0/8", "dstIps": "192.168.0.0/16",
             "ipProtocols": ["tcp"], "dstPorts": "22"},
    actions=["SUCCESS"]
).answer().frame()

print(result.head())
# columns: Flow, Traces (per-hop list), TraceCount
```

Mathematically the query enumerates the BDD region of header bits that satisfy `srcIp ∈ 10/8 AND dstIp ∈ 192.168/16 AND proto=TCP AND dstPort=22 AND no-ACL-deny-on-path`. Any non-empty result is an ACL hole — SSH from the corporate WAN reaching the datacentre.

### 15.4 Ansible playbook timing math for 1000 devices

```yaml
- hosts: all_routers
  gather_facts: false
  serial: 100
  strategy: linear
  vars:
    ansible_command_timeout: 30
    ansible_persistent_command_timeout: 60
  tasks:
    - name: Push BGP policy
      cisco.ios.ios_config:
        src: "templates/bgp-policy.j2"
      register: result
    - name: Validate
      cisco.ios.ios_command:
        commands:
          - show ip bgp summary | json
      register: bgp_state
    - name: Save running to startup
      cisco.ios.ios_command:
        commands: write memory
```

Timing for `forks: 50`, 1000 devices, 3 tasks averaging 10s each:

```
batches = 1000 / 100 = 10
per_batch_workers = min(50, 100) = 50
serial_per_host_total = 3 × 10s = 30s
per_batch_time = (100 / 50) × 30s = 60s
total = 10 × 60 = 600s = 10 minutes
```

Add 5% retry overhead: `total ≈ 630s`.

If `strategy: free` were used instead, the per_batch_time becomes `max_per_host_total = 30s` (since each host runs independently), but the lockstep verification semantics are lost — task 2 might run on host A before task 1 finished on host B. For network changes that depend on lockstep convergence, stick with `linear`.

### 15.5 YANG augment vs deviation pattern decisions

Decision tree:

```
Is the target leaf semantically present, just with vendor-specific extension?
    YES → augment: add new leaf in your module under target node
    NO  → continue.

Does the device implement the leaf at all?
    NO  → deviation { deviate not-supported; }
    YES → continue.

Does the device implement the leaf with a different type / range?
    YES → deviation { deviate replace { type ...; } }
    NO  → augment for any *additional* leaves you need.
```

Worked example: vendor's interface module supports a `mtu` leaf typed as `uint16` with range `64..9216`. The standard model has `uint32` with range `0..65535`. Choices:

```yang
// Wrong — augment cannot replace existing leaf type:
augment "/if:interfaces/if:interface" {
  leaf mtu { type uint16 { range "64..9216"; } }
}

// Right — deviation replaces the type:
deviation "/if:interfaces/if:interface/if:mtu" {
  deviate replace { type uint16 { range "64..9216"; } }
}
```

Agents that consume the device's `yang-library` advertisement learn the deviation and adjust their generated requests accordingly. This is how OpenConfig multi-vendor automation tolerates per-vendor quirks without forking the model.

---

## 16. Practical operational notes

### 16.1 Authentication and AAA

| Protocol | Auth options |
|:---|:---|
| NETCONF | SSH key, SSH password, TACACS+ via RADIUS-like delegation |
| RESTCONF | HTTP Basic, mTLS, OAuth2 bearer (RFC 8040 §2.5) |
| gNMI | mTLS, gRPC metadata (basic auth), per-role authorisation via gNSI |

mTLS deployment requires:

- a per-collector client cert,
- a per-device server cert,
- a CA hierarchy that both sides trust,
- cert rotation cadence (≤90 days for compliance).

### 16.2 Rate limits and backpressure

Devices throttle session creation to avoid CPU exhaustion:

```
NETCONF-server: 10 sessions/min default
gNMI-server:    20 sessions/sec default, MAX_CONCURRENT_STREAMS = 100
RESTCONF:       200 req/sec per client, HTTP/1.1 keep-alive limit
```

Client-side backoff:

```
on_failure(reason):
    backoff = base * 2^attempt + jitter
    base    = 1s, max = 60s
    jitter  = uniform(0, base)
```

### 16.3 Telemetry collector scaling

Single-collector throughput limits (commodity 16 vCPU, 64 GiB):

```
gNMI Subscribe streams:    ~50K
gNMI updates/sec:          ~500K
NETCONF notifications/sec: ~5K (XML parse cost dominates)
RESTCONF GETs/sec:         ~2K (HTTP/1.1 connection limit)
```

Sharding strategies for scale beyond a single collector:

- by site (one collector per geo)
- by device class (one for spines, one for leaves)
- by metric domain (one for counters, one for routing)

### 16.4 Common pitfalls

- **Datastore confusion**: editing `running` directly on a device that supports `:candidate:` defeats the rollback mechanism. Always edit `candidate` and `<commit/>`.
- **Lock starvation**: NETCONF `<lock>` is mandatory before `<edit-config>` on `candidate` if multiple agents share the device. Releasing the lock on session close is automatic but a half-dead session may hold the lock until a transport timeout.
- **gNMI `update` vs `replace`**: `replace` deletes any leaf not in the request; `update` merges. Mixing them in a single `Set` is allowed but the order is fixed (delete → replace → update).
- **YANG `when` ambiguity**: a node guarded by `when` may not be in the instance even if mentioned in a request — the client must compute `when` predicates locally to avoid send/reject loops.
- **Default values not echoed back**: `with-defaults=trim` (the most efficient mode) hides leaves that match defaults, which can confuse diff engines that expect every leaf back. Pin a known mode in the client's request.

### 16.5 Source of Truth contract vs render

The SoT (NetBox/Nautobot) holds *intent*. Render code (Jinja, Python, GoTemplate) maps intent to device-config. Two failure modes:

- **SoT diverges from render**: render is broken; fix the template.
- **Render diverges from device**: device has been hand-modified; either revert or absorb the change back into SoT.

The reconciliation loop is:

```
SoT  ──render──►  intended_config
                       │
                       ▼
                   device_config (running)
                       │
                       ▼
                  diff(intended, running)
                       │
                  ┌────┴────┐
                  │         │
              empty?     non-empty
                  │         │
                ok      decide: absorb-into-SoT or push-from-SoT
```

CI tests render the SoT and compare against expected config snapshots; production reconciliation tests render the SoT and compare against the live device.

---

## 17. See Also

- `networking/restconf` — RESTCONF protocol details, URI grammar, query parameters.
- `networking/yang-models` — YANG language reference, types, statements.
- `networking/network-programmability` — broader survey of programmability APIs.
- `config-mgmt/ansible` — Ansible playbook structure, modules, strategies.
- `config-mgmt/napalm` — NAPALM driver matrix and getters.
- `monitoring/gnmi-gnoi` — gNMI/gNOI gRPC services in depth.
- `monitoring/model-driven-telemetry` — MDT subscription design and collector architecture.
- `ramp-up/network-automation-eli5` — narrative ramp-up of the same material.
- `ramp-up/ansible-eli5` — narrative ramp-up of Ansible.

---

## 18. References

- **RFC 6020** — YANG 1.0 (M. Bjorklund, 2010). Original schema language.
- **RFC 7950** — YANG 1.1 (M. Bjorklund, 2016). Adds `action`, `notification`-in-data, `anydata`, refined error reporting.
- **RFC 6241** — NETCONF Configuration Protocol (R. Enns et al., 2011). Operations, datastores, capability framework.
- **RFC 6242** — Using SSH for NETCONF (M. Wasserman, 2011). Subsystem `netconf`, chunked framing.
- **RFC 6243** — With-Defaults Capability (A. Bierman, 2011). `report-all`, `trim`, `explicit`, `report-all-tagged`.
- **RFC 5277** — NETCONF Event Notifications (S. Chisholm, H. Trevino, 2008).
- **RFC 8040** — RESTCONF Protocol (A. Bierman et al., 2017). HTTP mapping of NETCONF.
- **RFC 8072** — RESTCONF YANG Patch Media Type (J. Schoenwaelder, 2017).
- **RFC 7951** — JSON Encoding of YANG Data (L. Lhotka, 2016).
- **RFC 8345** — A YANG Data Model for Network Topologies (A. Clemm et al., 2018).
- **RFC 8342** — Network Management Datastore Architecture (NMDA) (M. Bjorklund et al., 2018). `intended`, `operational`, datastore separation.
- **RFC 8527** — RESTCONF Extensions to Support NMDA (M. Bjorklund et al., 2019).
- **RFC 8526** — NETCONF Extensions to Support NMDA (M. Bjorklund et al., 2019).
- **RFC 7895** — YANG Module Library (M. Bjorklund, 2016). `ietf-yang-library` module.
- **RFC 8525** — YANG Library (NMDA-aware) (M. Bjorklund et al., 2019).
- **gNMI Specification** — `github.com/openconfig/gnmi` — gnmi.proto, sub-spec for Subscribe, gNMI Path Conventions.
- **gNOI Specification** — `github.com/openconfig/gnoi` — Service definitions for OS, file, cert, factory_reset, system, healthz, etc.
- **OpenConfig YANG Models** — `github.com/openconfig/public` — interface, BGP, network-instance, telemetry, AAA modules.
- **IETF YANG Models** — `github.com/YangModels/yang/tree/main/standard/ietf` — `ietf-interfaces`, `ietf-ip`, `ietf-routing`, `ietf-bgp`, etc.
- **Edelman, Lowe, Oswalt** — *Network Programmability and Automation* (O'Reilly, 2nd ed, 2023). Canonical multi-tool overview.
- **NetBox Documentation** — `docs.netbox.dev` — IPAM, DCIM, plugins, ORM model.
- **Nautobot Documentation** — `docs.nautobot.com` — fork of NetBox with extended automation primitives, Jobs, Git data sources.
- **Batfish Documentation** — `batfish.org` — symbolic analysis, BDD-based reachability, Pybatfish API.
- **Containerlab Documentation** — `containerlab.dev` — multi-vendor topology compiler.
- **Suzieq Documentation** — `suzieq.readthedocs.io` — multi-vendor state collector, Parquet schema.
- **Ansible Network Collections** — `ansible-collections/ansible.netcommon`, `cisco.ios`, `cisco.iosxr`, `cisco.nxos`, `arista.eos`, `junipernetworks.junos`.
- **NAPALM Documentation** — `napalm.readthedocs.io` — driver matrix, getters, configuration ops.
- **pyang** — `github.com/mbj4668/pyang` — IETF reference YANG processor.
- **libyang** — `github.com/CESNET/libyang` — high-performance YANG parser, validator, NMDA support.
- **yanglint** — bundled with libyang — schema and instance validator CLI.
