# Cisco ACI — Policy-Driven Data Center Fabric

> *ACI inverts the traditional networking paradigm: instead of configuring individual devices with CLIs and hoping the aggregate behavior matches intent, you declare the desired end state in a centralized policy model, and the fabric compiles it into forwarding rules. The network becomes a function of application requirements rather than the other way around.*

---

## 1. The Evolution from Device-Centric to Policy-Centric Networking

Traditional data center networking treats each switch as an independent entity. An engineer configures VLANs, ACLs, spanning tree priorities, HSRP groups, and routing protocols on every device. The "network" is an emergent property of hundreds of individual configurations that must remain consistent. This model has fundamental scaling problems.

### The Configuration Explosion

Consider a data center with $N$ leaf switches and $S$ spine switches. In a traditional design, each leaf might need:

- VLAN definitions (replicated on every switch)
- Spanning tree root priority tuning
- HSRP/VRRP configuration per VLAN per switch pair
- ACLs per interface or VLAN (replicated and kept in sync)
- Routing protocol adjacencies

The number of individual configuration items grows as $O(N \times V)$ where $V$ is the number of VLANs. With 100 leafs and 500 VLANs, that is 50,000 configuration elements to keep synchronized manually.

### ACI's Answer: Declare Once, Apply Everywhere

ACI replaces this with a single policy repository on the APIC cluster. You define objects (tenants, VRFs, bridge domains, EPGs, contracts) once. The APIC renders these into concrete configuration and pushes it to exactly the switches that need it. A policy change on APIC propagates to all affected leaf and spine switches within seconds.

The mental model shifts from "which box do I configure?" to "what should my application's network look like?" This is the meaning of Application Centric Infrastructure.

### Imperative vs Declarative

| Imperative (Traditional) | Declarative (ACI) |
|---|---|
| `interface Vlan10` | EPG: "WebTier" in BD: "Web-BD" |
| `ip address 10.10.10.1 255.255.255.0` | Subnet: 10.10.10.1/24 on BD |
| `standby 10 ip 10.10.10.1` | Anycast gateway (automatic) |
| `access-list 100 permit tcp any host 10.10.10.5 eq 443` | Contract with filter tcp/443 |
| (repeat on every switch pair) | (defined once, pushed by APIC) |

---

## 2. Spine-Leaf Fabric Architecture

### Why Spine-Leaf?

The spine-leaf (or Clos) topology was originally described by Charles Clos in 1953 for telephone switching networks. It provides non-blocking, predictable performance with uniform latency. Every endpoint is exactly two hops from every other endpoint (through one spine), eliminating the variable-depth tree problem of traditional three-tier designs.

### Capacity Planning

In a spine-leaf fabric with $L$ leaf switches and $S$ spine switches, the total bisection bandwidth is:

$$B_{bisection} = L \times S \times b$$

Where $b$ is the bandwidth of each leaf-to-spine uplink. If every leaf has one link to every spine (which ACI requires), the oversubscription ratio at the spine tier is:

$$Oversubscription = \frac{\text{Total server-facing bandwidth per leaf}}{\text{Total uplink bandwidth per leaf}} = \frac{D \times d}{S \times b}$$

Where $D$ is the number of downlink ports and $d$ is the downlink speed. For a leaf with 48x 25G downlinks and 6x 100G uplinks:

$$Oversubscription = \frac{48 \times 25}{6 \times 100} = \frac{1200}{600} = 2:1$$

### Scaling Limits

Adding more spines increases bisection bandwidth linearly. Adding more leafs increases the number of endpoints but does not change per-leaf bandwidth. The maximum fabric size is bounded by the number of ports on each spine switch. With 64-port spine switches:

- Maximum leaf switches: 64 per pod
- Maximum spine switches: limited by physical space and APIC capacity
- Endpoints per fabric: up to 400,000+ (depending on APIC model)

### Underlay vs Overlay

```
Overlay (VXLAN + MP-BGP EVPN):
  Tenant traffic, EPG segmentation, policy enforcement
  ┌─────────────────────────────────────────────────┐
  │  VXLAN Header (VNI=16M segments)                │
  │  ┌─────────────────────────────────────────┐    │
  │  │  Original Ethernet Frame (tenant data)  │    │
  │  └─────────────────────────────────────────┘    │
  └─────────────────────────────────────────────────┘

Underlay (IS-IS + COOP):
  Infrastructure reachability between VTEPs (leaf/spine TEP addresses)
  ┌─────────────────────────────────────────────────┐
  │  Outer IP Header (src/dst TEP)                  │
  │  Outer UDP Header (port 4789)                   │
  │  [Overlay payload above]                        │
  └─────────────────────────────────────────────────┘
```

ACI uses IS-IS as its underlay routing protocol. Every leaf and spine forms an IS-IS adjacency with its neighbors on the infra VLAN. The IS-IS routes carry only the TEP (Tunnel Endpoint) addresses of fabric nodes, not tenant routes. Tenant routes are distributed via MP-BGP EVPN between leaf switches, with spine switches acting as route reflectors.

COOP (Council of Oracles Protocol) is an ACI-proprietary protocol that runs between leaf switches and spines. Leaf switches report learned endpoints (IP/MAC bindings) to spines via COOP. Spines maintain a global endpoint database and answer queries from other leafs, enabling hardware-proxy mode to avoid ARP flooding.

---

## 3. The ACI Object Model in Depth

### The Management Information Tree (MIT)

ACI's entire configuration is stored as a tree of Managed Objects (MOs) in the Management Information Tree. Every object has a Distinguished Name (DN) that describes its position in the tree. The DN is a path from the root (`uni` for policy universe, `topology` for physical topology).

```
uni (Policy Universe)
 ├── tn-{tenant}                          # fvTenant
 │    ├── ctx-{vrf}                       # fvCtx (VRF)
 │    ├── BD-{bd}                         # fvBD (Bridge Domain)
 │    │    ├── subnet-[{ip}]              # fvSubnet
 │    │    └── rsctx                      # relationship to VRF
 │    ├── ap-{app}                        # fvAp (Application Profile)
 │    │    └── epg-{epg}                  # fvAEPg (EPG)
 │    │         ├── rsbd                  # relationship to BD
 │    │         ├── rsprov-{contract}     # provided contract
 │    │         ├── rscons-{contract}     # consumed contract
 │    │         └── rspathAtt             # static path binding
 │    ├── brc-{contract}                  # vzBrCp (Contract)
 │    │    └── subj-{subject}             # vzSubj (Subject)
 │    │         └── rsSubjFiltAtt         # relationship to filter
 │    ├── flt-{filter}                    # vzFilter
 │    │    └── e-{entry}                  # vzEntry (match rule)
 │    └── out-{l3out}                     # l3extOut
 │         └── instP-{ext-epg}           # l3extInstP
 │
 topology (Physical)
  └── pod-{id}
       └── node-{id}
            ├── sys                       # system info
            │    ├── phys-[eth1/1]        # physical interface
            │    └── bgp                  # BGP instance
            └── health                    # health scores
```

### Object Relationships

ACI uses explicit relationship objects (prefix `Rs` for "relation source") to link objects. This is different from traditional networking where relationships are implicit. For example, an EPG does not "know" its BD by having a VLAN number; instead, it has an `fvRsBd` object pointing to the BD's DN.

This indirection is powerful: you can change which BD an EPG belongs to by modifying one relationship object, without touching the EPG or BD definitions themselves. All dependent configuration (gateway IP, routing, etc.) automatically adjusts.

### Class Hierarchy

Every managed object belongs to a class. Classes form an inheritance hierarchy. Key base classes:

| Class | Description | Key Subclasses |
|---|---|---|
| `polUni` | Policy universe root | All policy objects |
| `fvTenant` | Tenant container | Contains VRF, BD, AP, contracts |
| `fvCtx` | VRF / routing context | One routing table per VRF |
| `fvBD` | Bridge domain | L2 flooding domain |
| `fvAEPg` | Endpoint group | Policy enforcement boundary |
| `vzBrCp` | Binary contract | Connects two EPGs |
| `vzFilter` | Traffic filter | L3/L4 match criteria |
| `l3extOut` | L3 external connection | BGP/OSPF/static to outside |
| `fabricNode` | Physical switch | Leaf or spine |
| `faultInst` | Fault instance | Alarms and health events |

---

## 4. Endpoint Groups: The Core Abstraction

### What Problem EPGs Solve

In traditional networking, security policy is tied to network topology. A firewall rule says "allow 10.10.10.0/24 to 10.10.20.0/24 on port 443." If you move a server from one subnet to another, you must rewrite firewall rules, update ACLs, and hope nothing breaks. The server's identity is its IP address.

EPGs decouple identity from topology. A server belongs to an EPG because of a policy assignment (static binding, VMM integration, or attribute match), not because of its IP address. You can move a server to a different leaf switch, different subnet, even a different pod, and its EPG membership (and therefore its security policy) follows it.

### EPG Classification Methods

**Static Path Binding:** The most explicit method. You bind a leaf/port/VLAN combination to an EPG. Any traffic arriving on that port with that VLAN encapsulation belongs to that EPG. This is the method used for bare-metal servers and network appliances.

**VMM Domain:** The APIC integrates with hypervisor managers (vCenter, SCVMM, OpenStack) and container orchestrators (Kubernetes). When a VM is placed in a port group that corresponds to an EPG, or a pod is annotated with an EPG reference, the endpoint is automatically classified. This is the method used for virtualized and containerized workloads.

**IP/MAC Attribute (Microsegmentation):** A uSeg EPG defines match criteria based on endpoint attributes: IP address, MAC address, VM name, operating system, or custom attributes. Endpoints matching the criteria are pulled from their base EPG into the uSeg EPG, receiving different policy. This enables zero-trust microsegmentation without changing the physical or virtual network.

### Intra-EPG Isolation

By default, endpoints within the same EPG can communicate freely. This models the traditional behavior where servers in the same VLAN can talk to each other. However, ACI allows you to enforce intra-EPG isolation, blocking all traffic between endpoints in the same EPG. This is useful for:

- Multi-tenant hosting (each customer's server in the same EPG but isolated)
- PCI-DSS environments requiring microsegmentation
- Desktop VDI where users should not see each other's traffic

---

## 5. Contracts: The Policy Enforcement Engine

### The Whitelist Model

ACI implements a default-deny model between EPGs. Two EPGs cannot communicate unless a contract explicitly permits the traffic. This is the opposite of traditional networking, where two VLANs on the same switch can communicate by default (routing permitting).

The whitelist model has profound implications for security posture. In a traditional network, a new application deployed in an existing VLAN immediately has network access to everything in that VLAN and anything reachable via routing. In ACI, a new EPG has zero connectivity until contracts are defined. Security is the default state.

### Contract Compilation

When you create a contract and associate it with EPGs, the APIC does not simply push an ACL to a switch. Instead, it performs a multi-step compilation:

1. **Policy resolution:** APIC resolves the contract's subjects and filters into concrete match criteria.
2. **Scope evaluation:** Based on the contract scope (application-profile, VRF, tenant, global), APIC determines which leaf switches need the rules.
3. **Zoning rule generation:** APIC compiles the contract into zoning rules. Each EPG is assigned a pcTag (policy class tag, a 16-bit identifier within the VRF). The zoning rule says "allow traffic from pcTag A to pcTag B matching filter F."
4. **Hardware programming:** The zoning rule is pushed to the leaf switch's TCAM. The leaf applies it in hardware at line rate.

The result is that contract enforcement happens in the data plane at ASIC speed, not in software. There is no performance penalty for having hundreds of contracts.

### pcTag and Class ID

Every EPG receives a pcTag (also called class ID) that is unique within its VRF scope. When a packet enters a leaf switch, the leaf looks up the source endpoint's EPG and stamps the packet with the source pcTag. At the destination leaf, the destination endpoint's pcTag is looked up. The leaf then checks the zoning rule table for a match on (src_pcTag, dst_pcTag, filter). If a match exists, the packet is forwarded; otherwise, it is dropped.

This is why ACI does not need traditional ACLs. The pcTag mechanism is more efficient because it operates on EPG identity rather than IP addresses, and the zoning rule table is much smaller than an equivalent ACL (one rule per EPG pair vs. one rule per IP pair).

### Taboo Contracts and Deny Rules

Standard contracts are permits (whitelist entries). ACI also supports taboo contracts, which are explicit deny rules. A taboo contract attached to an EPG blocks specific traffic even if another contract would permit it. Taboo contracts take precedence over regular contracts.

Use cases for taboo contracts:

- Blocking a specific port across all EPGs (e.g., deny SMBv1 globally)
- Emergency containment (block a compromised EPG's lateral movement)
- Compliance (ensure certain traffic never traverses the fabric)

---

## 6. Bridge Domains: Reimagining Layer 2

### The Problem with Traditional VLANs

VLANs were invented in 1995 to segment broadcast domains. They work, but they have limitations that become painful at data center scale:

- **4094 VLAN limit:** The 12-bit VLAN ID field in 802.1Q allows only 4094 VLANs. Large multi-tenant data centers exhaust this quickly.
- **Spanning tree:** VLANs require STP to prevent loops, which blocks redundant links and wastes bandwidth.
- **Flooding:** Broadcast, unknown unicast, and multicast (BUM) traffic floods the entire VLAN, consuming bandwidth on every port.
- **Gateway locality:** HSRP/VRRP provides a virtual gateway, but only one router is active. Traffic from servers connected to the standby router must hairpin.

### How Bridge Domains Fix These Problems

An ACI Bridge Domain is a logical broadcast domain that uses VXLAN instead of VLANs across the fabric. This provides:

- **16 million segments:** VXLAN uses a 24-bit VNI (VXLAN Network Identifier), allowing over 16 million unique segments.
- **No spanning tree:** The spine-leaf topology is loop-free by design. VXLAN encapsulation means the fabric never runs STP between leaf switches.
- **Controlled flooding:** BD settings control whether BUM traffic is flooded, proxied, or dropped. In hardware-proxy mode, ARP requests are intercepted by the leaf and answered from the COOP database, eliminating most broadcast traffic.
- **Distributed anycast gateway:** Every leaf switch that has endpoints in a BD hosts the same gateway IP with the same MAC address. Traffic is always routed locally; there is no active/standby hairpinning.

### Hardware Proxy vs Flood Mode

**Hardware Proxy (Default):** When endpoint A sends an ARP request for endpoint B, the local leaf intercepts it. The leaf queries the spine's COOP database for B's IP-to-MAC binding. If found, the leaf fabricates an ARP reply and sends it back to A. The ARP request never leaves the local leaf. This dramatically reduces broadcast traffic and is the recommended mode for BDs with more than a few hundred endpoints.

**Flood Mode:** ARP requests are flooded across the fabric to all leaf switches with ports in the BD. This behaves like a traditional VLAN. It is simpler but does not scale. Use flood mode only when endpoints require it (e.g., legacy appliances that rely on broadcast for discovery).

### Multiple Subnets per BD

Unlike traditional VLANs where one VLAN equals one subnet, an ACI Bridge Domain can host multiple subnets. This is because the BD is a Layer 2 domain and subnets are Layer 3 constructs attached to it. Multiple subnets share the same L2 flooding behavior and the same set of EPGs.

This is useful for subnet migration scenarios. You can add a new subnet to an existing BD, migrate servers gradually, and remove the old subnet without changing the BD or its EPGs.

---

## 7. L3Out: Connecting to the Outside World

### The Boundary Problem

An ACI fabric is a closed system. Endpoints inside the fabric communicate via EPGs and contracts. But real applications need to reach external networks: the internet, campus LANs, WAN links, partner networks. L3Out is the mechanism that bridges the ACI policy model with traditional routing.

### How L3Out Works

An L3Out creates a routed connection between the ACI fabric and an external router. It consists of:

1. **Logical Node Profile:** Identifies which leaf switch(es) will peer with the external router.
2. **Logical Interface Profile:** Configures the interface (routed port, SVI, or sub-interface) with IP addressing.
3. **Routing Protocol:** BGP, OSPF, EIGRP, or static routes for exchanging prefixes with the external router.
4. **External EPG (l3extInstP):** Classifies external traffic into an EPG so that contracts can be applied. External EPG subnets define which external prefixes are classified into this EPG.

### External EPG Subnet Scope Flags

External EPG subnets have scope flags that control their behavior:

| Flag | Meaning |
|---|---|
| `export-rtctrl` | Advertise this subnet to the external router |
| `import-security` | Use this subnet for contract enforcement (classify incoming traffic) |
| `shared-rtctrl` | Share this route with other VRFs (inter-VRF route leaking) |
| `shared-security` | Allow this subnet to be used for contract enforcement across VRFs |
| `aggregate-export` | Summarize (aggregate) when advertising |
| `aggregate-import` | Summarize when importing for classification |

Getting these flags right is one of the most common sources of L3Out misconfiguration. A subnet with only `export-rtctrl` will be advertised but incoming traffic will not match any contract (no `import-security`), resulting in drops.

### Transit Routing

ACI can function as a transit network, routing traffic between two L3Outs. This requires:

1. Both L3Outs in the same VRF (or route-leaking between VRFs)
2. Transit subnets classified on both External EPGs
3. Contracts between the two External EPGs
4. `import-security` and `export-rtctrl` flags on the appropriate subnets

This transforms ACI from an endpoint-hosting fabric into a core routing platform.

---

## 8. VMM Integration Architecture

### The Integration Model

ACI's VMM (Virtual Machine Manager) integration is not just a port group manager. It creates a bidirectional link between the hypervisor/container platform and the fabric:

**APIC to Hypervisor:** APIC pushes port groups (VMware) or network definitions (OpenStack/K8s) to the virtualization platform. When an EPG is associated with a VMM domain, the corresponding port group appears in vCenter automatically.

**Hypervisor to APIC:** The hypervisor reports endpoint information (VM name, MAC, IP, host, port group) back to APIC. This provides real-time endpoint inventory and enables features like VM mobility tracking, microsegmentation by VM attribute, and health monitoring.

### OpFlex: The Southbound Protocol

ACI uses OpFlex as the southbound protocol between APIC and leaf switches. OpFlex is a declarative protocol: APIC sends policy declarations ("EPG Web has pcTag 1234, contract permits tcp/443 from pcTag 5678") rather than imperative commands ("program this ACL entry"). The leaf switch resolves the policy locally based on which endpoints are present.

For VMM integration, the OpFlex agent runs on the hypervisor (vSwitch) or container host. It receives policy from APIC and programs the local virtual switch accordingly. This extends ACI policy enforcement down to the virtual switch, providing microsegmentation between VMs on the same host.

### Kubernetes CNI Architecture

The ACI Kubernetes integration uses the `aci-containers-controller` and `aci-containers-host-agent`:

```
┌──────────── Kubernetes Cluster ────────────┐
│                                            │
│  ┌─────────────────────────────────────┐   │
│  │ aci-containers-controller (1x)      │   │
│  │  - Watches K8s API (pods, services) │   │
│  │  - Maps namespaces → EPGs           │   │
│  │  - Maps NetworkPolicy → Contracts   │   │
│  │  - Reports to APIC                  │   │
│  └─────────────────────────────────────┘   │
│                                            │
│  ┌────────────── Each Node ─────────────┐  │
│  │ aci-containers-host-agent            │  │
│  │  - Receives policy via OpFlex        │  │
│  │  - Programs OVS (Open vSwitch)       │  │
│  │  - Assigns VXLAN VNIDs to pods       │  │
│  │  - Reports endpoints to controller   │  │
│  └──────────────────────────────────────┘  │
│                                            │
└────────────────────────────────────────────┘
         │                    ▲
         ▼                    │
┌────── ACI Fabric ──────────────────────────┐
│  APIC receives pod endpoints               │
│  Applies contracts per namespace/EPG       │
│  Provides load balancing for Services      │
└────────────────────────────────────────────┘
```

This architecture means that Kubernetes network policies are not just enforced at the OVS level (as with Calico or Cilium) but are also enforced in the ACI fabric hardware. A pod-to-pod flow between different nodes traverses both the OVS policy and the leaf switch zoning rules.

---

## 9. Multi-Site and Multi-Pod Architectures

### Multi-Pod: Extending a Single Domain

Multi-Pod allows a single ACI fabric (one APIC cluster, one policy domain) to span multiple physical locations connected by an Inter-Pod Network (IPN). The IPN is a standard IP network (routers, not ACI switches) that carries VXLAN-encapsulated traffic between pods.

Requirements for the IPN:

- OSPF reachability between pods (for TEP route distribution)
- DHCP relay for initial pod discovery
- MTU of at least 9150 bytes (VXLAN overhead + original frame)
- PIM Bidir or head-end replication for multicast-based BUM traffic

Multi-Pod is transparent to tenants. An EPG that spans two pods works identically to one in a single pod. The APIC cluster distributes policy to all pods, and the spines in each pod act as gateway nodes for inter-pod traffic.

### Multi-Site: Independent Failure Domains

Multi-Site connects two or more independent ACI fabrics, each with its own APIC cluster. The Nexus Dashboard Orchestrator (NDO, formerly MSO) provides a unified policy layer above the individual sites.

Key differences from Multi-Pod:

| Aspect | Multi-Pod | Multi-Site |
|---|---|---|
| APIC cluster | Single (shared) | One per site (independent) |
| Failure domain | Single (APIC outage affects all pods) | Independent (one site survives other's failure) |
| Policy sync | Automatic (same cluster) | NDO pushes templates to each site |
| Intersite traffic | VXLAN over IPN (same VNI space) | VXLAN with intersite header (translated at border leaf) |
| Control plane | IS-IS + COOP across all pods | BGP EVPN between sites |
| Use case | Campus / multi-building | Geo-distributed data centers |

### Stretched vs Local Objects

In Multi-Site, NDO introduces the concept of stretched and local objects:

- **Stretched tenant:** Exists on multiple sites, same configuration
- **Stretched EPG:** Endpoints on multiple sites share the same EPG; cross-site communication is seamless
- **Stretched BD:** L2 domain extends across sites (use cautiously; cross-site L2 adds complexity)
- **Local EPG/BD:** Exists only on one site, not synchronized

The general recommendation is to stretch L3 (VRFs and contracts) but keep L2 (BDs) local unless active-active workloads require it. Stretching L2 across a WAN introduces latency-sensitive ARP/BUM traffic and increases the blast radius of broadcast storms.

---

## 10. ACI Policy Model vs Traditional Networking

### Paradigm Comparison

Traditional networking follows a device-centric, imperative paradigm:

1. Engineer identifies which switches need configuration
2. Engineer writes CLI commands for each switch
3. Engineer applies commands in a maintenance window
4. If something breaks, engineer troubleshoots per-device

ACI follows an application-centric, declarative paradigm:

1. Engineer defines the desired application topology (EPGs, contracts)
2. APIC compiles this into per-device configuration
3. APIC pushes to all affected switches atomically
4. If something breaks, engineer examines the policy model and health scores

### What ACI Eliminates

| Traditional Task | ACI Equivalent |
|---|---|
| VLAN trunking configuration | Automatic (EPG-to-leaf binding handles encapsulation) |
| Spanning tree tuning | Not needed (loop-free spine-leaf) |
| HSRP/VRRP configuration | Not needed (distributed anycast gateway) |
| ACL management per switch | Contracts (defined once, compiled to hardware) |
| DHCP relay per SVI | BD-level DHCP relay (configured once) |
| Monitoring IP SLA | Health scores and fault correlation (built in) |

### What ACI Introduces

ACI is not simpler than traditional networking; it shifts complexity:

- **Object model learning curve:** Engineers must learn the tenant/VRF/BD/EPG/contract hierarchy
- **Contract troubleshooting:** "Show zoning-rule" replaces "show access-list" but requires understanding pcTags
- **API-first operations:** Bulk changes require REST API or Ansible, not CLI scripts
- **Health score interpretation:** Faults roll up into health scores that require understanding the aggregation model
- **Upgrade complexity:** Fabric-wide upgrades must be coordinated (APIC first, then switches)

---

## 11. The APIC REST API Architecture

### REST Principles in ACI

The APIC REST API maps directly to the Management Information Tree. Every managed object has a URL based on its DN:

```
https://apic/api/mo/{dn}.json        # Managed object by DN
https://apic/api/class/{class}.json   # All objects of a class
```

The API supports:

- **GET:** Read objects and their properties
- **POST:** Create or update objects (idempotent; posting the same object twice is safe)
- **DELETE:** Remove objects
- **WebSocket subscriptions:** Real-time event streaming for monitoring

### Query Filters and Options

The API supports query parameters that control the response:

| Parameter | Purpose | Example |
|---|---|---|
| `query-target` | Scope of query | `self`, `children`, `subtree` |
| `target-subtree-class` | Filter subtree by class | `fvAEPg`, `fvBD` |
| `query-target-filter` | Boolean filter expression | `eq(fvTenant.name,"Prod")` |
| `rsp-subtree` | Include children in response | `full`, `children`, `no` |
| `rsp-subtree-class` | Filter response subtree | `fvRsBd,fvRsProv` |
| `rsp-subtree-include` | Include related info | `faults`, `health`, `count` |
| `order-by` | Sort results | `fvTenant.name\|asc` |
| `page` and `page-size` | Pagination | `page=0&page-size=100` |

### Idempotency and Desired State

A critical property of the ACI API is idempotency. POSTing the same JSON to the same URL multiple times has the same effect as posting it once. This makes automation safe: if a script fails halfway through and is re-run, it will not create duplicate objects or corrupt state.

This also means the API is "desired state" driven. You POST the final desired configuration, and APIC calculates the diff. If the object already exists with different properties, APIC updates only the changed properties. If it does not exist, APIC creates it.

### Error Handling

API errors return structured JSON with error codes:

```json
{
  "imdata": [{
    "error": {
      "attributes": {
        "code": "102",
        "text": "configured object ((Dn0)) not found"
      }
    }
  }]
}
```

Common error codes:

| Code | Meaning |
|---|---|
| 102 | Object not found (invalid DN) |
| 103 | Property validation failed (e.g., invalid VLAN range) |
| 120 | Authentication required (token expired) |
| 121 | Authorization failed (RBAC) |
| 122 | Invalid request (malformed JSON) |
| 400 | Timeout (APIC overloaded or unreachable) |

---

## 12. Fabric Discovery, COOP, and Endpoint Learning

### The Discovery Protocol Stack

When a new switch is connected to an ACI fabric, a multi-protocol discovery sequence executes:

**Phase 1 — LLDP (Link Layer Discovery Protocol):** The new switch and its neighbor exchange LLDP frames. This establishes physical connectivity and identifies the neighbor's node ID.

**Phase 2 — Infra VLAN and DHCP:** The new switch sends a DHCP request on the infra VLAN (default 3967). The APIC acts as DHCP server and assigns a TEP address from the TEP pool. The switch also receives the APIC's IP address.

**Phase 3 — Firmware Download:** If the switch firmware does not match the APIC's configured version, the APIC pushes the correct firmware via TFTP/HTTP. The switch reboots with the new image.

**Phase 4 — Policy Download:** Once firmware matches, the APIC pushes the full policy (tenants, EPGs, contracts) relevant to that switch via OpFlex. The switch programs its TCAM with zoning rules.

**Phase 5 — IS-IS Adjacency:** The switch forms IS-IS adjacencies with its neighbors on the infra VLAN. This establishes underlay reachability for VXLAN tunnels.

**Phase 6 — COOP Registration:** The switch registers with the spine's COOP database. From this point, the switch can learn and report endpoints.

### Endpoint Learning

ACI learns endpoints through data plane inspection:

1. A frame arrives at a leaf switch port
2. The leaf examines the source MAC and source IP
3. The leaf records the endpoint: (MAC, IP, EPG/pcTag, interface, VTEP)
4. The leaf reports this endpoint to the spine via COOP
5. Other leaf switches can query the spine's COOP database to resolve unknown destinations

This is how hardware-proxy ARP works: instead of flooding an ARP request, the leaf queries COOP for the target IP's MAC address and fabricates the ARP reply locally.

### Endpoint Aging and Bounce

Endpoints have aging timers. If no traffic is seen from an endpoint for the aging period (default 300 seconds for remote endpoints, 900 seconds for local endpoints), the endpoint is removed from the table. This prevents stale entries from accumulating.

Endpoint bounce detection prevents flapping. If an endpoint's location changes rapidly (e.g., due to a loop or misconfiguration), the fabric dampens the updates. The endpoint is flagged as "bouncing" and eventually quarantined until the issue is resolved.

---

## Prerequisites

- Solid understanding of VLANs, IP routing, and ARP
- Familiarity with VXLAN encapsulation and EVPN control plane concepts
- Basic knowledge of spine-leaf (Clos) fabric topology
- Understanding of Layer 2 vs Layer 3 forwarding and broadcast domains
- Familiarity with REST APIs and JSON data structures
- Basic understanding of IS-IS and BGP routing protocols
- Exposure to Cisco NX-OS command line (helpful for leaf/spine troubleshooting)

## References

- [Cisco ACI Fundamentals Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/apic/all/aci-fundamentals/cisco-aci-fundamentals.html)
- [APIC REST API Configuration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/apic/all/apic-rest-api-configuration-guide/cisco-apic-rest-api-configuration-guide.html)
- [ACI Policy Model Whitepaper](https://www.cisco.com/c/en/us/solutions/collateral/data-center-virtualization/application-centric-infrastructure/white-paper-c11-731960.html)
- [OpFlex Protocol Specification](https://tools.ietf.org/html/draft-smith-opflex-03)
- [RFC 7348 — VXLAN: A Framework for Overlaying Virtualized Layer 2 Networks over Layer 3 Networks](https://www.rfc-editor.org/rfc/rfc7348)
- [RFC 7432 — BGP MPLS-Based Ethernet VPN](https://www.rfc-editor.org/rfc/rfc7432)
- [Cisco ACI Multi-Site Architecture Whitepaper](https://www.cisco.com/c/en/us/solutions/collateral/data-center-virtualization/application-centric-infrastructure/white-paper-c11-739609.html)
- [Cisco ACI and Kubernetes Integration Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/containers/cisco-aci-and-kubernetes-integration.html)
- [Charles Clos, "A Study of Non-Blocking Switching Networks," Bell System Technical Journal, 1953](https://ieeexplore.ieee.org/document/6770468)
- [Cisco ACI Troubleshooting Guide](https://www.cisco.com/c/en/us/td/docs/dcn/aci/apic/all/aci-troubleshooting/cisco-aci-troubleshooting-guide.html)
