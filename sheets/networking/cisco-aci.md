# Cisco ACI (Application Centric Infrastructure)

Policy-driven SDN fabric for data center networks using a spine-leaf topology managed centrally by the APIC cluster through a declarative object model.

## Fabric Architecture

### Spine-Leaf Topology

```
                    ┌──────────┐   ┌──────────┐   ┌──────────┐
                    │ Spine  1 │   │ Spine  2 │   │ Spine  3 │
                    │ (N9K-C)  │   │ (N9K-C)  │   │ (N9K-C)  │
                    └──┬──┬──┬─┘   └─┬──┬──┬──┘   └─┬──┬──┬──┘
                       │  │  │       │  │  │         │  │  │
              ┌────────┘  │  └───┐   │  │  │   ┌─────┘  │  └────────┐
              │           │      │   │  │  │   │        │           │
           ┌──┴──┐   ┌────┴──┐  └───┴──┴──┴───┘   ┌────┴──┐   ┌───┴──┐
           │Leaf1│   │Leaf 2 │      (full mesh)    │Leaf 3 │   │Leaf 4│
           │(TOR)│   │ (TOR) │                     │ (TOR) │   │(TOR) │
           └──┬──┘   └───┬───┘                     └───┬───┘   └──┬───┘
              │          │                              │          │
         ┌────┴───┐  ┌───┴───┐                     ┌───┴───┐  ┌───┴───┐
         │Servers │  │Servers│                     │Servers│  │Servers│
         └────────┘  └───────┘                     └───────┘  └───────┘
```

- Every leaf connects to every spine (no leaf-to-leaf or spine-to-spine links)
- Consistent hop count: any endpoint to any other is always 2 hops through spine
- Spines run IS-IS for underlay, VXLAN/EVPN for overlay
- Leaf switches are Nexus 9300/9400 series; spines are Nexus 9500 series

### APIC Cluster

```
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │ APIC  1 │◄──►│ APIC  2 │◄──►│ APIC  3 │
   │(Active) │    │(Active) │    │(Active) │
   └────┬────┘    └────┬────┘    └────┬────┘
        │              │              │
   ┌────┴──────────────┴──────────────┴────┐
   │       Leaf Switches (Out-of-Band)     │
   └───────────────────────────────────────┘
```

- Minimum 3 APICs for production (odd number for quorum)
- All APICs are active; workload is distributed (no standby)
- APIC connects to leaf switches (never to spines)
- Sharded database (Consul-like) replicated across cluster
- Losing one APIC: cluster still has quorum; losing two: read-only mode
- APIC manages policy; data plane continues if all APICs go down

## ACI Object Model

### Hierarchy

```
Fabric (Universe)
 └── Tenant
      ├── VRF (Private Network / Context)
      │    └── Bridge Domain (BD)
      │         ├── Subnet (default gateway)
      │         └── Endpoint Group (EPG)
      │              ├── Static Bindings (port, VLAN)
      │              ├── VMM Domain bindings
      │              └── Contracts (provided/consumed)
      ├── Application Profile
      │    └── EPG (grouped here for app context)
      ├── Contract
      │    ├── Subject
      │    │    └── Filter (L3/L4 rules)
      │    └── Subject (additional)
      ├── L3Out (external routed network)
      │    └── External EPG (l3extInstP)
      └── L2Out (external bridged network)
```

### Key Objects

| Object | DN Prefix | Description |
|---|---|---|
| Tenant | `uni/tn-{name}` | Top-level policy container; isolation boundary |
| VRF | `uni/tn-{name}/ctx-{name}` | Layer 3 routing domain (one routing table) |
| BD | `uni/tn-{name}/BD-{name}` | Layer 2 broadcast domain; replaces VLAN |
| Subnet | `uni/tn-{name}/BD-{name}/subnet-[ip]` | Default gateway for the BD |
| App Profile | `uni/tn-{name}/ap-{name}` | Logical grouping of EPGs for an application |
| EPG | `uni/tn-{name}/ap-{name}/epg-{name}` | Set of endpoints with same policy |
| Contract | `uni/tn-{name}/brc-{name}` | Defines allowed traffic between EPGs |
| Filter | `uni/tn-{name}/flt-{name}` | ACL-like match criteria (proto/port) |
| L3Out | `uni/tn-{name}/out-{name}` | External L3 connectivity (BGP/OSPF/static) |

### System Tenants

| Tenant | Purpose |
|---|---|
| `common` | Shared policies (VRFs, BDs, contracts) inherited by all tenants |
| `infra` | Infrastructure policies (VLAN pools, domains, interface policies) |
| `mgmt` | In-band and out-of-band management |
| User tenants | Custom tenants for workloads |

## Endpoint Groups and Microsegmentation

### EPG Basics

```
# EPG = collection of endpoints sharing the same security policy
# Endpoints can be: physical servers, VMs, containers, IP addresses
#
# EPG membership by:
#   - Static port binding (leaf/port + VLAN)
#   - VMM domain (auto from vCenter/K8s)
#   - IP/MAC attribute (microsegmentation)

# Static binding example (APIC CLI):
apic1# configure
apic1(config)# tenant Production
apic1(config-tenant)# application MyApp
apic1(config-application)# epg WebServers
apic1(config-epg)# bridge-domain WebBD
apic1(config-epg)# contract-consumer WebToApp
apic1(config-epg)# contract-provider ExternalAccess
```

### Microsegmentation (uSeg EPG)

```
# Microsegmentation allows finer grouping within a base EPG
# Attributes for classification:
#   - IP address / subnet
#   - MAC address
#   - VM name (from VMM)
#   - Custom VM attribute (tag, OS type, etc.)
#   - DNS name

# Example: isolate PCI servers within a general EPG
# via APIC GUI or REST:
# Tenants > Production > uSeg EPGs > PCI-Servers
#   Match Type: IP
#   IP: 10.10.50.0/24
#   Precedence: set higher than base EPG
```

### EPG-to-EPG Communication

```
                     Contract
  ┌────────┐     ┌──────────────┐     ┌────────┐
  │ EPG A  │────►│  Subject(s)  │◄────│ EPG B  │
  │(consume│     │  Filter(s)   │     │(provide│
  │   r)   │     │  tcp/443     │     │   r)   │
  └────────┘     └──────────────┘     └────────┘

  - Without a contract, EPGs cannot communicate (whitelist model)
  - Intra-EPG traffic is allowed by default (configurable)
  - vzAny: apply contract to all EPGs in a VRF at once
```

## Contracts, Filters, and Subjects

### Contract Structure

```
Contract: "WebToApp"
 ├── Subject: "HTTPS-Traffic"
 │    ├── Filter: "tcp-443"
 │    │    └── Entry: ether=IP, proto=TCP, dPort=443
 │    └── Filter: "tcp-8443"
 │         └── Entry: ether=IP, proto=TCP, dPort=8443
 └── Subject: "Health-Checks"
      └── Filter: "icmp-allow"
           └── Entry: ether=IP, proto=ICMP
```

### Creating a Contract via REST API

```bash
# Create a filter
curl -sk -X POST https://apic1/api/mo/uni/tn-Production.json \
  -H "Content-Type: application/json" \
  -b cookie.txt \
  -d '{
    "vzFilter": {
      "attributes": { "name": "tcp-443" },
      "children": [{
        "vzEntry": {
          "attributes": {
            "name": "https",
            "etherT": "ip",
            "prot": "tcp",
            "dFromPort": "443",
            "dToPort": "443"
          }
        }
      }]
    }
  }'

# Create a contract with subject referencing the filter
curl -sk -X POST https://apic1/api/mo/uni/tn-Production.json \
  -H "Content-Type: application/json" \
  -b cookie.txt \
  -d '{
    "vzBrCp": {
      "attributes": { "name": "WebToApp", "scope": "tenant" },
      "children": [{
        "vzSubj": {
          "attributes": { "name": "https-traffic" },
          "children": [{
            "vzRsSubjFiltAtt": {
              "attributes": { "tnVzFilterName": "tcp-443" }
            }
          }]
        }
      }]
    }
  }'
```

### Contract Scope

| Scope | Traffic Allowed Between |
|---|---|
| `application-profile` | EPGs in the same App Profile only |
| `context` (VRF) | EPGs in the same VRF |
| `tenant` | EPGs in the same Tenant |
| `global` | EPGs across any Tenant |

### Provider vs Consumer

```
# Provider: the EPG that OFFERS the service (e.g., web server listening)
# Consumer: the EPG that INITIATES the connection (e.g., client)
#
# Direction matters for:
#   - Service graph insertion (PBR)
#   - QoS marking
#   - Statistics and health score
#
# A single EPG can be both provider and consumer of different contracts
```

## Bridge Domains vs VLANs

### Comparison

| Feature | Traditional VLAN | ACI Bridge Domain |
|---|---|---|
| Scope | Single switch / trunk | Entire fabric (VXLAN stretched) |
| Broadcast | Flooded everywhere | Controlled (ARP flooding or unicast) |
| Gateway | Per-switch HSRP/VRRP | Distributed anycast gateway |
| Limit | 4094 VLANs | 16M VNID (VXLAN segment IDs) |
| Multi-tenancy | VRF-lite hacks | Native tenant isolation |
| Subnets | One subnet per VLAN | Multiple subnets per BD |

### BD Settings

```
# Key BD knobs:
#   L2 Unknown Unicast: flood | hardware-proxy (default: proxy)
#   ARP Flooding: yes | no (default: no, use unicast ARP via COOP)
#   Unicast Routing: enabled (default) — needed for L3 forwarding
#   Multi-Destination Flooding: bd-flood | encap-flood | drop
#   Limit IP Learning to Subnet: yes — prevents rogue IPs

# Typical server BD:
#   - Unicast routing: enabled
#   - ARP flooding: disabled (hardware proxy for scale)
#   - L2 unknown unicast: hardware proxy
#   - Subnet: 10.10.10.1/24 (scope: public, shared as needed)
```

### VLAN to EPG Mapping

```
Physical Domain
 └── VLAN Pool: 100-199 (static allocation)
      └── Attachable Entity Profile (AEP)
           └── Interface Policy Group
                └── Leaf Interface Profile
                     └── Leaf/Port → EPG static binding

# VLANs in ACI are local significance only (leaf-facing)
# VXLAN VNID is used across the fabric
# Access VLAN → EPG → BD → VRF mapping:
#   Port VLAN 100 → epg-Web → BD-Web → VRF-Prod
```

## L3Out for External Connectivity

### L3Out Components

```
L3Out: "External-Internet"
 ├── Logical Node Profile
 │    └── Node: leaf-101 (router-id 10.0.0.101)
 │         └── Logical Interface Profile
 │              └── Interface: eth1/49 (SVI or routed)
 │                   └── IP: 192.168.1.1/30
 ├── External EPG (l3extInstP): "Internet"
 │    ├── Subnet: 0.0.0.0/0 (external classification)
 │    └── Contracts: consume "WebAccess"
 └── BGP Peer: 192.168.1.2 (remote-as 65000)
```

### L3Out via REST API

```bash
# Query existing L3Outs
curl -sk https://apic1/api/class/l3extOut.json \
  -b cookie.txt | python3 -m json.tool

# Common L3Out routing protocols:
#   - BGP (most common for external peering)
#   - OSPF (for campus/WAN integration)
#   - EIGRP (legacy Cisco campus)
#   - Static routes (simple uplinks)
```

### Transit Routing and Shared L3Out

```
# Shared L3Out (in tenant "common"):
#   1. Create L3Out in tenant "common"
#   2. External EPG subnet scope: "shared-rtctrl", "shared-security"
#   3. BD subnet in user tenant: "shared" scope
#   4. Contract between external EPG and user EPG with "global" scope
#
# This lets multiple tenants share one physical uplink
```

## VMM Integration

### VMware vCenter Integration

```
# VMM Domain setup:
#   1. Create VLAN Pool (dynamic allocation, e.g., 1000-1999)
#   2. Create VMM Domain (type: VMware)
#   3. Add vCenter controller credentials
#   4. Associate domain with AEP
#   5. Bind EPGs to VMM domain
#
# APIC pushes port groups to vCenter automatically
# VMs placed in port group → auto-classified into EPG
# APIC reads VM inventory from vCenter for endpoint tracking

# VMM Domain types:
#   - VMware (vDS integration)
#   - Microsoft (Hyper-V, SCVMM)
#   - OpenStack (Neutron plugin)
#   - Kubernetes / OpenShift (CNI plugin)
#   - Red Hat Virtualization
```

### Kubernetes Integration

```
# ACI CNI Plugin (acc-provision):
#   - Deploys opflex-agent and aci-containers on each node
#   - Pods get EPG membership via annotations/namespace mapping
#   - Network policies translated to ACI contracts
#   - Service graph for LoadBalancer services

# Namespace → EPG mapping:
#   namespace: production  → EPG: kube-production
#   namespace: staging     → EPG: kube-staging

# Pod annotation for EPG override:
# metadata:
#   annotations:
#     opflex.cisco.com/endpoint-group: '{"tenant":"K8s","app":"MyApp","epg":"WebTier"}'
```

## Multi-Site and Multi-Pod

### Multi-Pod

```
  ┌─────── Pod 1 ──────┐     IPN      ┌─────── Pod 2 ──────┐
  │ Spine ─── Spine     │◄───(OSPF)───►│ Spine ─── Spine     │
  │   │         │       │   VXLAN MP   │   │         │       │
  │ Leaf  ─── Leaf      │  BGP EVPN    │ Leaf  ─── Leaf      │
  │ APIC1   APIC2       │             │ APIC3                │
  └─────────────────────┘             └──────────────────────┘

# Multi-Pod: single APIC cluster, multiple pods
# IPN (Inter-Pod Network): standard IP routers connecting pods
# Same policy domain, same APIC cluster
# Use case: separate rooms, floors, or buildings on campus
```

### Multi-Site

```
  ┌──── Site A ────┐                  ┌──── Site B ────┐
  │ APIC Cluster A │◄── MSO/NDO ────►│ APIC Cluster B │
  │ (independent)  │   (orchestrator) │ (independent)  │
  │ Full fabric    │                  │ Full fabric    │
  └────────────────┘                  └────────────────┘

# Multi-Site: separate APIC clusters, unified by NDO
# NDO (Nexus Dashboard Orchestrator): stretches tenants/EPGs/contracts
# Each site is fully independent (survives WAN failure)
# Use case: geographically separated data centers
# Intersite traffic: VXLAN over WAN (needs DCI link)
```

## APIC REST API

### Authentication

```bash
# Login and get auth cookie
curl -sk -X POST https://apic1/api/aaaLogin.json \
  -d '{"aaaUser":{"attributes":{"name":"admin","pwd":"C1sco!23"}}}' \
  -c cookie.txt

# Token is valid for 300 seconds (default), refresh with:
curl -sk -X GET https://apic1/api/aaaRefresh.json -b cookie.txt
```

### Common API Queries

```bash
# List all tenants
curl -sk https://apic1/api/class/fvTenant.json -b cookie.txt

# Get specific tenant
curl -sk https://apic1/api/mo/uni/tn-Production.json -b cookie.txt

# Get all EPGs in a tenant
curl -sk "https://apic1/api/mo/uni/tn-Production.json?\
query-target=subtree&target-subtree-class=fvAEPg" -b cookie.txt

# Get all endpoints (fvCEp) in the fabric
curl -sk https://apic1/api/class/fvCEp.json -b cookie.txt

# Get fabric health score
curl -sk https://apic1/api/mo/topology/HDfabricOverallHealth5min-0.json \
  -b cookie.txt

# Get faults (severity critical)
curl -sk "https://apic1/api/class/faultInst.json?\
query-target-filter=eq(faultInst.severity,\"critical\")" -b cookie.txt
```

### Creating Objects via API

```bash
# Create a tenant
curl -sk -X POST https://apic1/api/mo/uni.json \
  -H "Content-Type: application/json" \
  -b cookie.txt \
  -d '{"fvTenant":{"attributes":{"name":"NewTenant","descr":"API created"}}}'

# Create VRF inside tenant
curl -sk -X POST https://apic1/api/mo/uni/tn-NewTenant.json \
  -H "Content-Type: application/json" \
  -b cookie.txt \
  -d '{"fvCtx":{"attributes":{"name":"Prod-VRF"}}}'

# Create BD with subnet
curl -sk -X POST https://apic1/api/mo/uni/tn-NewTenant.json \
  -H "Content-Type: application/json" \
  -b cookie.txt \
  -d '{
    "fvBD": {
      "attributes": {"name":"Web-BD","arpFlood":"no","unicastRoute":"yes"},
      "children": [
        {"fvRsCtx":{"attributes":{"tnFvCtxName":"Prod-VRF"}}},
        {"fvSubnet":{"attributes":{"ip":"10.10.10.1/24","scope":"public"}}}
      ]
    }
  }'
```

### WebSocket Subscriptions

```bash
# Subscribe to real-time events on a class
# 1. Open WebSocket: wss://apic1/socket{token}
# 2. Subscribe via REST:
curl -sk "https://apic1/api/class/fvCEp.json?subscription=yes" \
  -b cookie.txt
# Response includes subscriptionId — events pushed on WebSocket
```

## Common Show Commands

### acidiag (APIC Diagnostics)

```bash
# Check APIC cluster health
acidiag avread                   # cluster status, all APICs
acidiag fnvread                  # fabric node vector (all nodes)
acidiag rvread                   # replica leader info

# Verify cluster convergence
acidiag verifyapic               # check APIC health

# Show APIC cluster size and IDs
cat /data/data_admin/sam_exported.config
```

### moquery (Managed Object Query)

```bash
# Query all fabric nodes
moquery -c fabricNode
# Output: node-id, name, role (spine/leaf/controller), serial, model

# Query specific tenant
moquery -c fvTenant -f 'fv.Tenant.name=="Production"'

# Query all EPGs
moquery -c fvAEPg

# Query endpoints (MAC+IP) in a specific EPG
moquery -c fvCEp -f 'fv.CEp.dn=="/uni/tn-Prod/ap-App/epg-Web"' -x \
  'query-target=subtree'

# Query faults
moquery -c faultInst -f 'fault.Inst.severity=="critical"'

# Query interface status
moquery -c ethpmPhysIf -f 'ethpm.PhysIf.operSt=="up"'

# Query VXLAN tunnel endpoints
moquery -c tunnelIf
```

### NX-OS Style Commands on Leaf/Spine

```bash
# Show endpoints learned on a leaf
show endpoint                    # all endpoints
show endpoint ip 10.10.10.50     # specific IP
show endpoint mac 00:50:56:xx    # specific MAC
show endpoint vrf Prod:VRF1      # endpoints in VRF

# Show VXLAN info
show nve peers                   # VTEP peers
show nve vni                     # VNI to BD/EPG mapping
show nve interface nve1          # NVE interface status

# Fabric health
show fabric membership           # registered nodes
show isis adjacency              # IS-IS underlay neighbors
show bgp l2vpn evpn summary     # EVPN route summary

# Contract/Zoning
show zoning-rule                 # compiled contract rules on leaf
show zoning-filter               # filter entries
show system internal epm endpoint  # endpoint manager table
```

## Fabric Discovery and Registration

### Discovery Process

```
1. New leaf/spine physically connected to fabric
2. LLDP discovers neighbor (existing leaf or APIC)
3. Switch downloads firmware image via TFTP from APIC
4. APIC assigns node-id and TEP address (from TEP pool)
5. Infra VLAN (default 3967) used for discovery communication
6. IS-IS adjacency forms on infra VLAN
7. COOP (Council of Oracles Protocol) syncs endpoint tables
8. Node shows "active" in fabric membership

Timeline:
  Power on → LLDP (30s) → Image download (5-15m) → Registration (1-2m)
  Total: ~10-20 minutes for new leaf
```

### Node Registration Commands

```bash
# On APIC: check pending nodes
acidiag fnvread                  # look for "unknown" or "inactive"

# Via API: query unregistered nodes
curl -sk https://apic1/api/class/dhcpClient.json -b cookie.txt

# Approve node registration (GUI path):
# Fabric > Inventory > Fabric Membership > Nodes Pending Registration
# Assign: Node ID, Node Name, Pod ID
```

### TEP Pool and Infra VLAN

```
# TEP (Tunnel Endpoint) Pool:
#   - Assigned during fabric setup (e.g., 10.0.0.0/16)
#   - Each node gets a /32 from this pool
#   - Spines: also get anycast TEP for COOP
#   - Cannot change after initial setup without re-building fabric

# Infra VLAN:
#   - Default: 3967 (configurable at setup only)
#   - Carries: discovery, APIC-to-node communication, DHCP, TFTP
#   - Must be allowed on all inter-switch links
#   - Never use for tenant traffic
```

## Tips

- Always start with the object model hierarchy when planning: Tenant > VRF > BD > EPG; skipping levels causes confusion.
- Use the `common` tenant for shared services (DNS, NTP, AD) and export contracts from there rather than duplicating across tenants.
- Set ARP flooding to disabled (hardware proxy mode) for BDs with more than a few hundred endpoints; flooding does not scale.
- Use `vzAny` contracts for baseline policies (e.g., permit ICMP to all EPGs in a VRF) instead of creating pairwise contracts.
- When troubleshooting connectivity, check `show zoning-rule` on the leaf first; if no rule exists, the contract is misconfigured.
- Enable "Limit IP Learning to Subnet" on every BD to prevent IP hijacking and rogue endpoint issues.
- Back up APIC configuration regularly via Policy > Export; snapshots are also useful before major changes (Config Rollback).
- The APIC REST API uses a tree structure; always navigate from the distinguished name (DN) down for precision.
- Use Postman or Python `requests` against the REST API for bulk operations; the GUI is not designed for hundreds of objects.
- For multi-site deployments, stretch only what is necessary; over-stretching L2 domains across sites introduces complexity and risk.
- Keep firmware consistent across all nodes; mixed versions cause unpredictable behavior and unsupported states.
- Label every object with descriptions; ACI tenants with hundreds of undocumented EPGs become unmanageable within months.
- Monitor the APIC health dashboard daily; a health score below 90 usually indicates actionable faults.

## See Also

- vxlan, bgp, ospf, is-is, vlan, private-vlans, stp, segment-routing, sd-wan, snmp

## References

- [Cisco ACI Fundamentals Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/apic/all/aci-fundamentals/cisco-aci-fundamentals.html)
- [APIC REST API Configuration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/apic/all/apic-rest-api-configuration-guide/cisco-apic-rest-api-configuration-guide.html)
- [ACI Fabric Hardware Installation Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/apic/all/apic-installation-aci-fabric-hardware.html)
- [Cisco ACI and Kubernetes Integration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/containers/cisco-aci-and-kubernetes-integration.html)
- [Cisco ACI Multi-Site Configuration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/aci-multi-site/cisco-aci-multi-site-configuration-guide.html)
- [Cisco ACI Policy Model Whitepaper](https://www.cisco.com/c/en/us/solutions/collateral/data-center-virtualization/application-centric-infrastructure/white-paper-c11-731960.html)
- [Nexus Dashboard Orchestrator Documentation](https://www.cisco.com/c/en/us/support/cloud-systems-management/multi-site-orchestrator/series.html)
- [ACI Best Practices Quick Summary](https://www.cisco.com/c/en/us/td/docs/dcn/whitepapers/cisco-aci-best-practices-quick-summary.html)
