# mDNS and DNS-SD -- Protocol Internals and Architecture

> *Multicast DNS repurposes the unicast DNS wire format for link-local name resolution, while DNS-Based Service Discovery layers a browsing and registration protocol on top of standard DNS record types. Together they form the backbone of zero-configuration networking.*

---

## 1. mDNS Packet Format Differences from Unicast DNS

### Wire Format Reuse

mDNS reuses the standard DNS message format defined in RFC 1035. The header, question, answer, authority, and additional sections are identical in structure. However, several fields are reinterpreted.

### Header Differences

| Field | Unicast DNS | mDNS (RFC 6762) |
|:---|:---|:---|
| QR bit | 0 = query, 1 = response | Same semantics |
| Opcode | Standard query (0) | Must be 0; others ignored |
| AA bit | Set by authoritative server | Always set in responses (responder is authoritative for its own records) |
| TC bit | Truncation indicator | Same; mDNS supports known-answer suppression to reduce truncation |
| RD bit | Recursion desired | Should be 0 (no recursion in mDNS) |
| RA bit | Recursion available | Should be 0 |
| Response code | NOERROR, NXDOMAIN, etc. | Should be 0; non-zero codes are silently ignored |
| ID field | Transaction matching | Should be 0 in multicast queries; non-zero only for legacy unicast |

### The QU Bit (Question Unicast-Response)

The most significant bit of the QCLASS field is repurposed as the QU flag:

```
Standard DNS QCLASS: 16 bits = class (e.g., 1 = IN)
mDNS QCLASS:        bit 15 = QU flag, bits 0-14 = class

QU=0: multicast response requested (normal mDNS query)
QU=1: unicast response requested (used during probing and initial queries)
```

When QU=1, the responder sends a unicast reply directly to the querier's source address and port, in addition to sending a multicast response. This reduces unnecessary multicast traffic during startup.

### The Cache-Flush Bit

The most significant bit of the RRCLASS field in resource records is repurposed:

```
Standard DNS RRCLASS: 16 bits = class
mDNS RRCLASS:        bit 15 = cache-flush flag, bits 0-14 = class

cache-flush=1: receivers must flush all cached records with the same name
               and type, replacing them with this record
cache-flush=0: this record is additive (does not flush existing entries)
```

The cache-flush bit is critical for ownership changes. When a host takes over a name, it announces with cache-flush=1 so all peers discard stale records from the previous owner.

### Source Port and Destination

Unicast DNS uses source port = ephemeral, destination port = 53. mDNS uses:
- Source port: 5353 (for standard queries and responses)
- Destination: 224.0.0.251:5353 (IPv4) or [ff02::fb]:5353 (IPv6)
- Legacy unicast queries may use an ephemeral source port; responses to those go back unicast

### TTL Enforcement

All mDNS packets must be sent with IP TTL = 255. Receivers must verify TTL = 255 and silently discard any packet with a lower TTL. This ensures packets originated on the local link, preventing off-link injection attacks.

---

## 2. Conflict Resolution Algorithm

### Overview

mDNS names must be unique per link. The conflict resolution mechanism has three phases: probing, announcing, and defending.

### Phase 1: Probing

When a host wants to claim a name (e.g., `myhost.local`):

1. Send three probe queries at 250ms intervals.
2. Each probe is a standard mDNS query with the QU bit set.
3. The probe includes the proposed records in the Authority section (not the Answer section), so other hosts can compare.
4. Wait 250ms after each probe for conflicting responses.

If no conflict is detected after all three probes (750ms total), the name is claimed.

### Phase 2: Announcing

After successful probing:

1. Send two unsolicited multicast announcements, 1 second apart.
2. Announcements carry the cache-flush bit set, flushing stale records on all peers.
3. The record TTL should be the intended TTL (typically 120 seconds for address records).

### Phase 3: Defending

Once a name is claimed, the host must defend it:

1. If a probe is received from another host for the same name, compare records.
2. Comparison is lexicographic on the rdata (raw bytes, class then type then rdata).
3. The host with the lexicographically later rdata wins.
4. The loser must choose a new name.

### Tiebreaking Rules

The simultaneous probe tiebreaker compares proposed records:

1. Sort all proposed records by class, then type, then rdata.
2. Compare the sorted sets element by element.
3. The set with the lexicographically later element at the first point of difference wins.
4. If one set is a proper prefix of the other, the longer set wins.

### Name Collision Behavior

When a host loses a conflict:

1. It must choose a new name, typically by appending a number: `myhost (2).local`.
2. It must re-probe with the new name (three probes, 250ms apart).
3. After 15 consecutive conflicts, the host should wait 5 seconds before the next attempt to prevent storms.

---

## 3. DNS-SD Service Instance Naming

### Three-Level Naming Hierarchy

DNS-SD defines a structured naming convention:

```
<Instance>.<Service>.<Domain>

Instance: human-readable UTF-8 string, max 63 bytes
          Examples: "Steve's Printer", "Living Room Speaker"

Service:  _application-protocol._transport-protocol
          Examples: _http._tcp, _ipp._tcp, _ssh._tcp

Domain:   the DNS domain (usually "local" for mDNS)
```

### Instance Name Characteristics

Instance names are designed to be user-facing:
- UTF-8 encoded, may contain spaces, punctuation, and international characters.
- Maximum 63 bytes (DNS label limit).
- Must be unique within the same service type on the same link.
- The user or administrator chooses the name, not software.

### Service Type Format

Service types follow a strict format:
- Must begin with underscore: `_http`
- Application protocol name: 1-15 characters from [a-z0-9-]
- Transport protocol: `_tcp` or `_udp`
- Registered in the IANA service name registry

### The Enumeration Meta-Query

To discover what service types are available on a network:

```
_services._dns-sd._udp.<domain> PTR ?
```

This returns PTR records pointing to each registered service type. For example:

```
_services._dns-sd._udp.local PTR _http._tcp.local
_services._dns-sd._udp.local PTR _ipp._tcp.local
_services._dns-sd._udp.local PTR _ssh._tcp.local
```

This two-step process (enumerate types, then browse instances within a type) keeps browsing efficient.

---

## 4. TXT Record Key-Value Encoding

### Wire Format

A DNS TXT record contains one or more strings, each prefixed by a single length byte:

```
TXT RDATA: [len1][string1][len2][string2]...[lenN][stringN]

Each string: key=value
Length byte: 0-255 (max string length = 255 bytes)
```

### Key Rules (RFC 6763, Section 6)

- Keys are ASCII printable characters, excluding `=` (0x3D).
- Keys are case-insensitive (receivers should compare case-insensitively).
- Recommended maximum key length: 9 characters (for efficiency).
- A key must appear at most once in a TXT record.

### Value Encoding

Three forms of TXT key-value pairs:

```
key=value     Normal key-value pair. Value is opaque binary data
              (often UTF-8 text). Example: "pdl=application/postscript"

key=          Key with empty value. Key exists, value is zero-length.
              Different from key being absent.

key           Boolean attribute. Presence means "true".
              Absence means "false" or "default".
```

### Size Constraints

- Each individual TXT string: max 255 bytes (including key, `=`, and value).
- Total TXT record: should fit in a single DNS message. RFC 6763 recommends keeping the total under 1300 bytes to avoid truncation.
- A TXT record must contain at least one string. If no metadata is needed, use a single empty string (length byte 0).

### The txtvers Key

RFC 6763 recommends that every DNS-SD TXT record include a `txtvers` key indicating the version of the TXT record format:

```
txtvers=1
```

This allows future schema evolution. Receivers that encounter an unknown `txtvers` can choose to ignore the TXT record or use a fallback.

---

## 5. Continuous Querying

### The Problem with One-Shot Queries

Traditional DNS resolves a name with a single query-response exchange. DNS-SD, however, needs to track services that appear and disappear dynamically.

### Querying at Increasing Intervals

mDNS continuous querying uses an exponential backoff:

1. First query at time T.
2. Second query at T + 1 second.
3. Subsequent queries at 2s, 4s, 8s, 16s, 32s, 60s intervals.
4. After reaching 60 seconds, continue querying every 60 minutes (3600 seconds) at most.

This balances responsiveness (quick initial discovery) with network efficiency (reduced long-term traffic).

### Known-Answer Suppression

To prevent redundant responses, a querier includes known answers in the Answer section of its query:

1. The querier lists all cached records that answer the question.
2. A responder that sees its record in the known-answer list suppresses its response.
3. If the known-answer section is too large for one packet, the TC (truncated) bit is set, and additional known answers follow in subsequent packets within 500ms.

This is critical for large networks with many services. Without it, every query would trigger responses from every service instance.

### Record Refresh at 80% TTL

To keep records from expiring, mDNS queriers refresh cached records:

1. At 80% of the TTL, send a query for the record.
2. At 85%, 90%, and 95% of the TTL, send additional queries if no response received.
3. If no response after all four attempts, the record is expired from cache.

This ensures records remain cached as long as the service is alive, without requiring the service to send unsolicited announcements.

---

## 6. Service Subtypes

### Motivation

A single service type like `_http._tcp` may encompass many different kinds of HTTP services (web servers, REST APIs, admin panels). Subtypes allow finer-grained browsing.

### Subtype PTR Records

A service instance can register under one or more subtypes:

```
_subtype._sub._http._tcp.local PTR "My Admin Panel._http._tcp.local"
```

The `_sub` label is the fixed delimiter. The subtype name precedes it.

### Browsing Subtypes

To browse only a specific subtype:

```
Query: _printer._sub._http._tcp.local PTR ?
```

This returns only HTTP services that registered the `_printer` subtype, filtering out all other HTTP services.

### Registration with Subtypes (Avahi Example)

```xml
<service-group>
  <name>Admin Panel</name>
  <service>
    <type>_http._tcp</type>
    <subtype>_admin._sub._http._tcp</subtype>
    <port>8443</port>
  </service>
</service-group>
```

The service is discoverable both as `_http._tcp` (general browse) and as `_admin._sub._http._tcp` (subtype browse).

---

## 7. Wide-Area DNS-SD

### Beyond Link-Local

Standard mDNS/DNS-SD is limited to a single network link. Wide-area DNS-SD (described in RFC 6763, Section 11) extends service discovery across network boundaries using unicast DNS.

### How It Works

1. Services register their PTR, SRV, and TXT records in a conventional DNS zone via DNS UPDATE (RFC 2136).
2. Clients browse by querying the DNS zone instead of sending multicast queries.
3. The browse domain is discovered via:
   - DHCP option 119 (domain search list)
   - Manual configuration
   - `b._dns-sd._udp.<domain>` and `lb._dns-sd._udp.<domain>` PTR lookups

### Domain Enumeration

DNS-SD defines special PTR queries for discovering browse and registration domains:

```
b._dns-sd._udp.<domain>   PTR <browse-domain>     # recommended browse domain
db._dns-sd._udp.<domain>  PTR <browse-domain>     # default browse domain
r._dns-sd._udp.<domain>   PTR <reg-domain>        # recommended registration domain
dr._dns-sd._udp.<domain>  PTR <reg-domain>        # default registration domain
lb._dns-sd._udp.<domain>  PTR <legacy-browse>     # legacy browse domain
```

### Practical Limitations

Wide-area DNS-SD sees limited real-world deployment:
- Requires DNS UPDATE infrastructure (dynamic DNS zones).
- Security of dynamic updates is complex (TSIG keys, update policies).
- Long-lived DNS records do not reflect real-time service availability.
- Most environments use mDNS for the local link and other mechanisms (consul, etcd, Kubernetes service discovery) for cross-network discovery.

---

## 8. Avahi Daemon Architecture

### Process Model

The Avahi stack consists of several cooperating components:

```
avahi-daemon          Core mDNS/DNS-SD responder and resolver.
                      Listens on UDP 5353, manages the local record database,
                      handles probing, announcing, conflict resolution,
                      and continuous querying. Runs as a system service.

avahi-dnsconfd        Configures the system's unicast DNS resolver based on
                      DNS server information discovered via mDNS. Watches
                      for _dns-sd._udp services and updates resolv.conf.

avahi-autoipd         IPv4 link-local address autoconfiguration (RFC 3927).
                      Assigns a 169.254.x.x address if no DHCP server is found.
                      Coordinates with avahi-daemon for .local name registration.
```

### D-Bus Interface

avahi-daemon exposes its functionality via D-Bus, which client applications use for service browsing and registration:

```
Bus name: org.freedesktop.Avahi
Object path: /

Key interfaces:
  org.freedesktop.Avahi.Server
    - GetHostName()
    - GetDomainName()
    - GetState()
    - EntryGroupNew() -> object path for registration
    - ServiceBrowserNew() -> object path for browsing
    - ServiceResolverNew() -> object path for resolving

  org.freedesktop.Avahi.EntryGroup
    - AddService(interface, protocol, flags, name, type, domain, host, port, txt[])
    - Commit()
    - Reset()
    - GetState()

  org.freedesktop.Avahi.ServiceBrowser
    - Signals: ItemNew, ItemRemove, Failure, AllForNow
```

### Client Libraries

Applications interact with avahi-daemon through client libraries rather than the D-Bus API directly:

- `libavahi-client` -- C library wrapping D-Bus calls. Used by most Linux desktop applications.
- `libavahi-core` -- embeds the full mDNS stack in-process (no daemon dependency). Used by embedded systems or applications that need standalone operation.
- `libavahi-compat-libdns_sd` -- compatibility shim that provides the Apple Bonjour API (dns_sd.h) on top of Avahi. Allows Bonjour-native applications to run on Linux without modification.
- `libavahi-glib`, `libavahi-qt` -- main-loop integration for GLib and Qt applications.

### Name Service Switch (NSS) Module

Avahi provides `nss-mdns` (libnss_mdns), a glibc NSS module that integrates .local name resolution into the standard `getaddrinfo()` / `gethostbyname()` path:

```
/etc/nsswitch.conf:
hosts: files mdns4_minimal [NOTFOUND=return] dns

mdns4_minimal  -- resolves .local names only, IPv4 only, returns NOTFOUND
                  for non-.local names (so the resolver falls through to dns)
mdns4          -- resolves .local via mDNS, IPv4 only
mdns_minimal   -- resolves .local via mDNS, IPv4 and IPv6
mdns           -- resolves all names via mDNS (not recommended)
```

The `_minimal` variants are strongly recommended because they prevent .local queries from leaking to unicast DNS, which would cause delays or incorrect results.

### Record Database and Caching

avahi-daemon maintains two record stores:

1. **Local records**: Records the host owns and defends (its hostname A/AAAA, registered services). Stored in memory, populated from static service files in `/etc/avahi/services/` and runtime registrations via D-Bus.

2. **Cache**: Records learned from other hosts on the network. Governed by mDNS TTL rules, with the 80%/85%/90%/95% refresh schedule described in the continuous querying section. Cache entries are per-interface and per-protocol (IPv4 and IPv6 caches are separate).

### Reflector Mode

When `enable-reflector=yes` is set in avahi-daemon.conf, the daemon forwards mDNS traffic between network interfaces. This bridges service discovery across VLANs or subnets:

- Incoming multicast queries on one interface are re-sent as multicast on all other interfaces.
- Responses are similarly forwarded.
- The reflector rewrites the interface index in SRV records so that resolved addresses are reachable from the querying subnet.
- This is simpler than wide-area DNS-SD but has scaling limitations (all multicast traffic is duplicated to all interfaces).

---

## References

- [RFC 6762 -- Multicast DNS](https://www.rfc-editor.org/rfc/rfc6762)
- [RFC 6763 -- DNS-Based Service Discovery](https://www.rfc-editor.org/rfc/rfc6763)
- [RFC 2136 -- Dynamic Updates in the Domain Name System (DNS UPDATE)](https://www.rfc-editor.org/rfc/rfc2136)
- [RFC 3927 -- Dynamic Configuration of IPv4 Link-Local Addresses](https://www.rfc-editor.org/rfc/rfc3927)
- [IANA Service Name and Transport Protocol Port Number Registry](https://www.iana.org/assignments/service-names-port-numbers)
- [Avahi Project -- avahi.org](https://avahi.org/)
- [Apple Bonjour -- developer.apple.com/bonjour](https://developer.apple.com/bonjour/)
- [nss-mdns -- GitHub](https://github.com/avahi/nss-mdns)
