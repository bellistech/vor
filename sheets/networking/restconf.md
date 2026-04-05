# RESTCONF (REST-Based Network Configuration)

RFC 8040 protocol mapping YANG models to RESTful HTTP APIs — GET/POST/PUT/PATCH/DELETE on YANG-modeled resources with JSON or XML encoding, query parameters, and SSE notifications.

## RESTCONF Architecture

### Resource hierarchy

```bash
# RESTCONF root resource (discovered via /.well-known/host-meta)
# Typical: https://<device>/restconf

# Resource types:
# {+restconf}                          → API root
# {+restconf}/data                     → datastore resource (config + state)
# {+restconf}/data/<path>              → data resource (specific node)
# {+restconf}/operations               → RPC/action operations
# {+restconf}/operations/<rpc-name>    → specific RPC operation
# {+restconf}/yang-library-version     → YANG module set version
```

### Discover RESTCONF root

```bash
# Get RESTCONF root path
curl -s -k https://10.0.0.1/.well-known/host-meta \
  -u admin:cisco123
# Returns:
# <XRD>
#   <Link rel="restconf" href="/restconf"/>
# </XRD>
```

## HTTP Methods (CRUD)

### GET — read data

```bash
# Get entire running datastore
curl -s -k https://10.0.0.1/restconf/data \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Get specific resource (IETF interfaces)
curl -s -k https://10.0.0.1/restconf/data/ietf-interfaces:interfaces \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Get single interface
curl -s -k https://10.0.0.1/restconf/data/ietf-interfaces:interfaces/interface=GigabitEthernet0%2F0%2F0 \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Get single leaf
curl -s -k https://10.0.0.1/restconf/data/ietf-interfaces:interfaces/interface=GigabitEthernet0%2F0%2F0/description \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Get with XML encoding
curl -s -k https://10.0.0.1/restconf/data/ietf-interfaces:interfaces \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+xml"
```

### POST — create new resource

```bash
# Create a new loopback interface
curl -s -k -X POST \
  https://10.0.0.1/restconf/data/ietf-interfaces:interfaces \
  -u admin:cisco123 \
  -H "Content-Type: application/yang-data+json" \
  -H "Accept: application/yang-data+json" \
  -d '{
    "ietf-interfaces:interface": {
      "name": "Loopback99",
      "type": "iana-if-type:softwareLoopback",
      "enabled": true,
      "description": "Test loopback"
    }
  }'
# Returns: 201 Created (success) or 409 Conflict (already exists)
```

### PUT — create or replace resource

```bash
# Replace entire interface config (creates if not exists)
curl -s -k -X PUT \
  https://10.0.0.1/restconf/data/ietf-interfaces:interfaces/interface=Loopback99 \
  -u admin:cisco123 \
  -H "Content-Type: application/yang-data+json" \
  -H "Accept: application/yang-data+json" \
  -d '{
    "ietf-interfaces:interface": {
      "name": "Loopback99",
      "type": "iana-if-type:softwareLoopback",
      "enabled": true,
      "description": "Replaced config"
    }
  }'
# Returns: 201 Created (new) or 204 No Content (replaced)
```

### PATCH — merge into existing resource

```bash
# Update description only (merge — does not touch other fields)
curl -s -k -X PATCH \
  https://10.0.0.1/restconf/data/ietf-interfaces:interfaces/interface=Loopback99 \
  -u admin:cisco123 \
  -H "Content-Type: application/yang-data+json" \
  -H "Accept: application/yang-data+json" \
  -d '{
    "ietf-interfaces:interface": {
      "name": "Loopback99",
      "description": "Updated description only"
    }
  }'
# Returns: 200 OK or 204 No Content
```

### DELETE — remove resource

```bash
# Delete an interface
curl -s -k -X DELETE \
  https://10.0.0.1/restconf/data/ietf-interfaces:interfaces/interface=Loopback99 \
  -u admin:cisco123
# Returns: 204 No Content (success) or 404 Not Found
```

## Resource URI Construction

### Path encoding rules

```bash
# Module prefix required for top-level container
/restconf/data/ietf-interfaces:interfaces

# Child nodes do not repeat the module prefix (same module)
/restconf/data/ietf-interfaces:interfaces/interface=eth0

# List key encoding: =<key-value>
/restconf/data/ietf-interfaces:interfaces/interface=GigabitEthernet0%2F0%2F0
#                                                                ^^^ / encoded as %2F

# Multiple keys: comma-separated
/restconf/data/openconfig-network-instance:network-instances/network-instance=default/protocols/protocol=BGP,bgp

# Different module child (augmented node): prefix required
/restconf/data/ietf-interfaces:interfaces/interface=eth0/ietf-ip:ipv4

# URL-encode special characters
# /  → %2F
# :  → %3A
# =  → %3D (in values only)
# ,  → %2C (in values only)
# space → %20
```

### Cisco native model paths (IOS-XE)

```bash
# Hostname
/restconf/data/Cisco-IOS-XE-native:native/hostname

# BGP config
/restconf/data/Cisco-IOS-XE-native:native/router/Cisco-IOS-XE-bgp:bgp

# Interface config
/restconf/data/Cisco-IOS-XE-native:native/interface/GigabitEthernet=0%2F0%2F0

# ACL
/restconf/data/Cisco-IOS-XE-native:native/ip/access-list

# OSPF
/restconf/data/Cisco-IOS-XE-native:native/router/Cisco-IOS-XE-ospf:router-ospf
```

### OpenConfig model paths

```bash
# Interfaces
/restconf/data/openconfig-interfaces:interfaces

# BGP neighbors
/restconf/data/openconfig-network-instance:network-instances/network-instance=default/protocols/protocol=BGP,bgp/bgp/neighbors

# System hostname
/restconf/data/openconfig-system:system/config/hostname

# LLDP
/restconf/data/openconfig-lldp:lldp
```

## Query Parameters

### depth — limit response depth

```bash
# Return only top-level containers (depth=1)
curl -s -k "https://10.0.0.1/restconf/data?depth=1" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Return up to 3 levels deep
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?depth=3" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# depth=unbounded — return everything (default)
```

### fields — select specific leaves

```bash
# Return only name and oper-status for each interface
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?fields=interface(name;oper-status)" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Multiple fields
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?fields=interface(name;enabled;description)" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"
```

### content — filter config vs state

```bash
# Config data only
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?content=config" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Non-config (state/operational) data only
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?content=nonconfig" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# All data (default)
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?content=all" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"
```

### with-defaults — control default value reporting

```bash
# Report all values including defaults
curl -s -k "https://10.0.0.1/restconf/data/ietf-interfaces:interfaces?with-defaults=report-all" \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"

# Modes: report-all, trim, explicit, report-all-tagged
```

## RESTCONF Operations (RPC)

### Invoke YANG RPC

```bash
# Restart device (Cisco IOS-XE)
curl -s -k -X POST \
  https://10.0.0.1/restconf/operations/cisco-ia:save-config \
  -u admin:cisco123 \
  -H "Content-Type: application/yang-data+json" \
  -H "Accept: application/yang-data+json"

# Custom RPC with input
curl -s -k -X POST \
  https://10.0.0.1/restconf/operations/ietf-routing:fib-route \
  -u admin:cisco123 \
  -H "Content-Type: application/yang-data+json" \
  -d '{
    "input": {
      "destination-address": "10.0.0.0/24"
    }
  }'
```

## RESTCONF Notifications (SSE)

### Subscribe to event stream

```bash
# Subscribe to NETCONF event stream via Server-Sent Events
curl -s -k -N \
  https://10.0.0.1/restconf/streams/NETCONF \
  -u admin:cisco123 \
  -H "Accept: text/event-stream"
# Keeps connection open, receives events as:
# data: <notification xmlns="...">...</notification>
# id: <event-id>

# Discover available streams
curl -s -k \
  https://10.0.0.1/restconf/data/ietf-restconf-monitoring:restconf-state/streams \
  -u admin:cisco123 \
  -H "Accept: application/yang-data+json"
```

## Python requests Examples

### GET with requests

```python
import requests
import json
from urllib3.exceptions import InsecureRequestWarning
requests.packages.urllib3.disable_warnings(InsecureRequestWarning)

BASE_URL = "https://10.0.0.1/restconf"
HEADERS = {
    "Accept": "application/yang-data+json",
    "Content-Type": "application/yang-data+json",
}
AUTH = ("admin", "cisco123")

# Get all interfaces
response = requests.get(
    f"{BASE_URL}/data/ietf-interfaces:interfaces",
    headers=HEADERS,
    auth=AUTH,
    verify=False,
)
interfaces = response.json()
for intf in interfaces.get("ietf-interfaces:interfaces", {}).get("interface", []):
    print(f"{intf['name']}: {intf.get('oper-status', 'unknown')}")
```

### POST/PUT/PATCH/DELETE with requests

```python
# Create interface (POST)
new_intf = {
    "ietf-interfaces:interface": {
        "name": "Loopback99",
        "type": "iana-if-type:softwareLoopback",
        "enabled": True,
        "description": "Created via RESTCONF",
    }
}
r = requests.post(
    f"{BASE_URL}/data/ietf-interfaces:interfaces",
    headers=HEADERS, auth=AUTH, verify=False,
    json=new_intf,
)
print(f"Create: {r.status_code}")             # 201 = created

# Update description (PATCH)
patch_data = {
    "ietf-interfaces:interface": {
        "name": "Loopback99",
        "description": "Updated via RESTCONF",
    }
}
r = requests.patch(
    f"{BASE_URL}/data/ietf-interfaces:interfaces/interface=Loopback99",
    headers=HEADERS, auth=AUTH, verify=False,
    json=patch_data,
)
print(f"Patch: {r.status_code}")              # 200 or 204

# Replace (PUT)
replace_data = {
    "ietf-interfaces:interface": {
        "name": "Loopback99",
        "type": "iana-if-type:softwareLoopback",
        "enabled": False,
        "description": "Replaced via RESTCONF",
    }
}
r = requests.put(
    f"{BASE_URL}/data/ietf-interfaces:interfaces/interface=Loopback99",
    headers=HEADERS, auth=AUTH, verify=False,
    json=replace_data,
)
print(f"Replace: {r.status_code}")            # 204

# Delete (DELETE)
r = requests.delete(
    f"{BASE_URL}/data/ietf-interfaces:interfaces/interface=Loopback99",
    headers=HEADERS, auth=AUTH, verify=False,
)
print(f"Delete: {r.status_code}")             # 204
```

### Error handling

```python
r = requests.patch(
    f"{BASE_URL}/data/ietf-interfaces:interfaces/interface=Loopback99",
    headers=HEADERS, auth=AUTH, verify=False,
    json=patch_data,
)
if r.status_code >= 400:
    error = r.json()
    errors = error.get("ietf-restconf:errors", {}).get("error", [])
    for e in errors:
        print(f"Type: {e.get('error-type')}")
        print(f"Tag: {e.get('error-tag')}")
        print(f"Message: {e.get('error-message')}")
```

## Device Configuration

### IOS-XE RESTCONF

```bash
# Enable RESTCONF (requires NETCONF-YANG first)
restconf
ip http secure-server                        # HTTPS required
ip http authentication local                 # local auth
ip http secure-port 443                      # default HTTPS port

# Verify
show platform software yang-management process
```

### NX-OS RESTCONF

```bash
feature restconf                             # enable RESTCONF
# NX-OS uses DME (Data Management Engine) model
# Default port: 443

# Verify
show feature | include restconf
```

### Authentication

```bash
# Basic auth (most common)
curl -u admin:cisco123 ...

# Token-based (if supported)
# 1. Get token
curl -s -k -X POST https://10.0.0.1/api/v1/auth/token \
  -d '{"username":"admin","password":"cisco123"}'
# 2. Use token
curl -s -k -H "Authorization: Bearer <token>" ...
```

## Content-Type Headers

### JSON vs XML

```bash
# JSON encoding (preferred for automation)
-H "Accept: application/yang-data+json"
-H "Content-Type: application/yang-data+json"

# XML encoding
-H "Accept: application/yang-data+xml"
-H "Content-Type: application/yang-data+xml"

# RESTCONF also accepts (legacy):
-H "Accept: application/vnd.yang.data+json"
```

## HTTP Status Codes

### RESTCONF response codes

```bash
# Success
# 200 OK           — GET with body, PATCH with body
# 201 Created      — POST/PUT created new resource
# 204 No Content   — PUT replaced, PATCH merged, DELETE removed

# Client error
# 400 Bad Request   — malformed request body
# 401 Unauthorized  — authentication failure
# 403 Forbidden     — insufficient privileges
# 404 Not Found     — resource does not exist
# 405 Method Not Allowed — HTTP method not valid for resource
# 409 Conflict      — POST on existing resource (use PUT)

# Server error
# 500 Internal Server Error — device-side failure
```

## See Also

- netconf
- yang-models
- gnmi-gnoi
- pyats

## References

- RFC 8040 — RESTCONF Protocol: https://datatracker.ietf.org/doc/html/rfc8040
- RFC 8072 — YANG Patch: https://datatracker.ietf.org/doc/html/rfc8072
- RFC 8071 — NETCONF Call Home / RESTCONF Call Home: https://datatracker.ietf.org/doc/html/rfc8071
- RFC 7951 — JSON Encoding of YANG Data: https://datatracker.ietf.org/doc/html/rfc7951
- Cisco RESTCONF guide: https://developer.cisco.com/docs/ios-xe/restconf/
