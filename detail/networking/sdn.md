# Software-Defined Networking — Deep Dive

> *SDN is the architectural decoupling of the control plane from the data plane. The control plane becomes a centralized, programmatic, software-defined entity; the data plane becomes a population of obedient forwarders driven by a southbound protocol. Every interesting property of SDN — agility, programmability, scale, telemetry, intent — falls out of that single split.*

---

## SDN definition + history

### The core thesis

For three decades, networking devices shipped as vertically integrated appliances: vendor-supplied silicon, vendor-supplied operating system, vendor-supplied control logic baked into firmware. Every router or switch ran its own copy of OSPF, IS-IS, BGP, STP, RSVP-TE, and so on. Cooperation between devices was achieved through distributed protocols whose only common interface was the wire. The behaviour of the network as a system was an *emergent property* of the local decisions of hundreds of independent agents.

SDN inverts that. The thesis is:

```
control plane         := centralized, programmable, software
data plane            := commoditized, fast, dumb forwarders
southbound API        := the protocol that wires the two together
northbound API        := the surface applications use to express intent
```

Once that split is real, three powerful consequences follow:

1. **Global view.** A central controller can observe and manipulate the entire network as a single object. Distributed protocols can only converge toward consistency; a controller can *enforce* it.
2. **Programmability.** Network behaviour becomes a function of code rather than a function of vendor firmware. You can compile policy to flow rules. You can write applications.
3. **Innovation rate decouples from hardware refresh cycles.** The data plane upgrades on silicon time scales (years). The control plane upgrades on software time scales (days).

### Brief history

- **2003-2007 — Ethane (Stanford).** Martin Casado, Nick McKeown, Scott Shenker. A clean-slate enterprise architecture with a centralized policy controller and dumb switches. Casado's PhD thesis laid the groundwork.
- **2008 — OpenFlow 1.0.** First public specification of a standard southbound protocol for installing flow rules into a switch. McKeown et al., "OpenFlow: Enabling Innovation in Campus Networks," ACM SIGCOMM CCR.
- **2011 — Open Networking Foundation (ONF).** Industry consortium founded by Deutsche Telekom, Facebook, Google, Microsoft, Verizon, and Yahoo to standardize OpenFlow and related SDN protocols.
- **2012-2013 — Google B4.** First production hyperscale SDN deployment: Google's inter-datacenter WAN, replacing traditional MPLS-TE with a centralized traffic engineering solver. SIGCOMM 2013 paper "B4: Experience with a Globally-Deployed Software Defined WAN."
- **2013 — OpenDaylight Project.** Linux Foundation umbrella for an open-source SDN controller. Initial members included Cisco, IBM, Juniper, NEC, Red Hat.
- **2014 — ONOS.** ON.Lab releases the Open Network Operating System, a distributed SDN controller targeting service-provider scale.
- **2014-2016 — P4.** Pat Bosshart et al., "Programming Protocol-Independent Packet Processors." A high-level language for describing the parser, match-action pipeline, and deparser of a programmable data plane. ACM CCR 2014.
- **2016-present — SD-WAN.** Vendor-led SDN for the branch-office WAN. Commercializes the controller idea but keeps the abstraction proprietary.
- **2018-present — SRv6 + intent.** IETF Segment Routing (RFC 8402), intent-based networking (Apstra, Cisco DNA, Juniper Apstra), closed-loop telemetry (gNMI, INT).

### What "control plane" and "data plane" mean precisely

- **Data plane** (a.k.a. forwarding plane). The per-packet machinery: parse headers, look up tables, rewrite headers, queue, schedule, transmit. Cycle budget is nanoseconds. Implemented in ASIC, NPU, FPGA, or fast-path software (DPDK/XDP). Decisions are deterministic functions of the packet and the table state.
- **Control plane.** The per-flow / per-prefix / per-policy machinery: run routing protocols, compute spanning trees, accept management commands, build forwarding tables. Cycle budget is milliseconds-to-seconds. Historically implemented in the box's CPU running a network OS (IOS, Junos, NX-OS).

In a classical router these two planes share silicon and a chassis but communicate through a narrow internal bus. In SDN they communicate through a *network*, using a standardized protocol. The control plane can therefore live anywhere — including a different building.

### What SDN is **not**

- It is not "network automation." Automation drives existing CLI/NETCONF interfaces with scripts; it does not move the control plane. SDN moves the control plane.
- It is not merely "a controller." A pile of Ansible playbooks talking to a vendor box is not SDN even if you call the box "the controller."
- It is not strictly OpenFlow. OpenFlow is one southbound protocol; many SDN deployments use NETCONF, gNMI, BGP-LS, P4Runtime, or vendor APIs.
- It is not the same as NFV (Network Functions Virtualization). NFV virtualizes *boxes* (firewalls, load balancers); SDN virtualizes the *control plane*. They compose well, but they are orthogonal.

---

## OpenFlow protocol

OpenFlow is the canonical southbound protocol. Even where production deployments use other protocols, OpenFlow's data model — flow tables, match-action, pipelines — is the lingua franca for SDN reasoning.

### Version evolution

| Version | Year | Highlights | Match fields |
|:---:|:---:|:---|:---:|
| 1.0 | 2009 | Single flow table, 12-tuple match, basic actions | 12 |
| 1.1 | 2011 | Multiple flow tables, group tables, MPLS | 15 |
| 1.2 | 2011 | Extensible match (OXM), IPv6 | ~30 |
| 1.3 | 2012 | Meter tables, per-flow counters, IPv6 ext headers | 40 |
| 1.4 | 2013 | Optical port descriptions, bundle messages | 41 |
| 1.5 | 2014 | Egress tables, packet type aware pipeline | 45+ |

**1.3 is the de-facto deployed standard.** Most controllers, switches, and reference stacks target 1.3 (specifically 1.3.5 with errata). Versions 1.4 and 1.5 added features but did not see broad adoption; many vendors stopped at 1.3.

### Flow table structure

A flow table entry has six fields:

```
+-----------+----------+--------------+----------+---------+----------+
|  Match    | Priority | Counters     | Instr.   | Timeout | Cookie   |
| (fields)  | (uint16) | (pkt, byte)  | (list)   | (idle,  | (opaque  |
|           |          |              |          |  hard)  |  uint64) |
+-----------+----------+--------------+----------+---------+----------+
```

- **Match.** A set of (header_field, value, mask) triples. Wildcarded with the mask.
- **Priority.** Higher priority wins on ties. 16-bit unsigned.
- **Counters.** Packet count, byte count, duration. Updated by hardware.
- **Instructions.** What to do on match: write actions, apply actions, goto-table, write metadata, meter, clear actions.
- **Timeouts.** Idle (drop entry after N seconds with no hits) and hard (drop entry after N seconds regardless).
- **Cookie.** Opaque controller-supplied identifier for rule classification.

Pipeline of N tables (1.1+) means a packet enters table 0, executes its instructions (which may include `goto-table k` for some k > current), and proceeds through the pipeline until either dropped, sent to a port, or sent to controller.

### The 12-tuple match (1.0)

OpenFlow 1.0 fixed the match to exactly 12 fields:

| # | Field | Bits | Notes |
|:---:|:---|:---:|:---|
| 1 | Ingress port | 16 | Switch-local port number |
| 2 | Ethernet source | 48 | MAC src |
| 3 | Ethernet dest | 48 | MAC dst |
| 4 | Ethernet type | 16 | 0x0800=IPv4, 0x86dd=IPv6, 0x0806=ARP |
| 5 | VLAN ID | 12 | 802.1Q VID |
| 6 | VLAN PCP | 3 | 802.1p priority |
| 7 | IP source | 32 | IPv4 src |
| 8 | IP dest | 32 | IPv4 dst |
| 9 | IP protocol | 8 | 6=TCP, 17=UDP, 1=ICMP |
| 10 | IP ToS | 8 | DSCP/ECN |
| 11 | TCP/UDP src port | 16 | L4 src |
| 12 | TCP/UDP dst port | 16 | L4 dst |

### The 40+-tuple match (1.3+)

OpenFlow 1.2 introduced the OpenFlow Extensible Match (OXM), a TLV format that lets the spec keep adding fields without breaking the wire format. By 1.3 the OXM registry holds 40+ fields:

```
PHY:    in_port, in_phy_port, metadata, tunnel_id
L2:     eth_src, eth_dst, eth_type, vlan_vid, vlan_pcp
L2.5:   mpls_label, mpls_tc, mpls_bos, pbb_isid
L3:     ipv4_src, ipv4_dst, ipv6_src, ipv6_dst, ipv6_flabel, ipv6_exthdr
        ip_proto, ip_dscp, ip_ecn, arp_op, arp_spa, arp_tpa,
        arp_sha, arp_tha
L4:     tcp_src, tcp_dst, udp_src, udp_dst, sctp_src, sctp_dst
ICMP:   icmpv4_type, icmpv4_code, icmpv6_type, icmpv6_code
ND:     ipv6_nd_target, ipv6_nd_sll, ipv6_nd_tll
```

A single OXM TLV is `[oxm_class:16 | field:7 | hasmask:1 | length:8 | value | mask?]`.

### Priority math and wildcards

Two flows match the same packet iff their match sets both subsume it. The packet hits the entry with the **highest priority**. Ties are implementation-defined (controllers must avoid them).

Priority field is uint16 → 65,536 levels. Common conventions:

| Priority | Use |
|:---:|:---|
| 0 | Default / table-miss / drop-all |
| 1-99 | Static defaults (LLDP punt, DHCP punt) |
| 100-999 | Reactive flows installed by controller |
| 1000-9999 | Service-policy rules |
| 10000-49999 | Application-installed micro-flows |
| 50000-65534 | Security exceptions (deny-list) |
| 65535 | Reserved |

A flow with all match fields wildcarded and priority 0 = the **table-miss flow**. By 1.3 spec the table-miss MUST be installed explicitly; unmatched packets are dropped silently otherwise.

Wildcard subsumption math: an entry $E_1$ is **strictly more specific** than $E_2$ iff every match field that $E_2$ wildcards $E_1$ also wildcards or specifies, and at least one wildcard in $E_2$ is specified in $E_1$. The ordering is partial; controllers usually enforce total order via priority.

### flow-mod, packet-in, packet-out messages

Three messages dominate any OpenFlow trace:

```
flow-mod        controller -> switch
                "Install / modify / delete this flow entry"

packet-in       switch -> controller
                "I have a packet that didn't match any flow (or was sent to me explicitly).
                 Here is the buffered packet (or its first 128 bytes), the ingress port,
                 the reason, and a cookie."

packet-out      controller -> switch
                "Inject this packet (referenced by buffer_id or carried inline) into the
                 pipeline, optionally specifying actions."
```

The reactive controller pattern:

```
1. Switch receives an unknown frame.
2. Table-miss flow sends packet to controller as packet-in.
3. Controller computes path, installs flow entries on every switch in the path.
4. Controller sends packet-out to the original switch to forward the original packet.
5. Subsequent packets of that flow hit the installed entries directly in hardware.
```

Reactive is simple but pays a controller round-trip on the first packet of every new flow. Hyperscalers don't tolerate that latency, so they install **proactive** flows — pre-compute the entire forwarding table and push it before any packet arrives.

### flow-mod commands

```
ADD          install new entry
MODIFY       update existing matching entries
MODIFY_STRICT match exactly (priority + match must match)
DELETE       remove matching entries
DELETE_STRICT exact match required
```

### Group tables

Groups exist for one-to-many forwarding: multicast, ECMP, fast failover.

| Type | Behavior |
|:---|:---|
| ALL | Execute every action bucket (multicast / mirror) |
| SELECT | Execute one bucket selected by hash / weight (ECMP) |
| INDIRECT | Single bucket; useful as a level of indirection |
| FAST_FAILOVER | Execute first bucket whose watch-port is up |

Fast failover lets the data plane react to a port-down event in microseconds without controller involvement.

### Meter tables

A meter rate-limits or remarks traffic to which it is bound. Each meter has bands (rate, burst, type=DROP|REMARK). The data plane measures the flow's rate; if it exceeds a band, the band's action is applied.

---

## Northbound / southbound APIs

The naming convention mirrors a stack diagram with the controller in the middle:

```
       +----------------------------+
       |  Network applications       |  e.g. routing app, firewall app,
       |  (Java / Python / Go etc.)  |       intent compiler, telemetry app
       +----------------------------+
                      ^
                      |  Northbound API (NBI):
                      |    - REST  (HTTPS + JSON)
                      |    - gRPC  (HTTP/2 + protobuf)
                      |    - GraphQL
                      |    - language-specific bindings
                      |
       +----------------------------+
       |       SDN controller        |  e.g. ONOS, OpenDaylight, Ryu, NOX,
       |  (Java / Python / Go etc.)  |       Faucet, Floodlight
       +----------------------------+
                      ^
                      |  Southbound API (SBI):
                      |    - OpenFlow  (1.0 / 1.3 / 1.5)
                      |    - NETCONF + YANG  (RFC 6241)
                      |    - gNMI / gNOI  (gRPC + protobuf)
                      |    - P4Runtime  (gRPC + protobuf)
                      |    - BGP-LS, PCEP, OVSDB, SNMP, REST, vendor-CLI
                      |
       +----------------------------+
       |   Data plane forwarders     |  switches, routers, vSwitches, smart
       +----------------------------+        NICs, programmable ASICs
```

### Northbound

The northbound API expresses the network *as an application sees it*. There is no single standard. Common shapes:

- **REST + JSON.** The default. URL paths model resources; verbs (GET/POST/PUT/DELETE) model operations. Easy to consume from any language.
- **gRPC + protobuf.** Lower latency, streaming, strongly typed. Used where the application is itself a service rather than a CLI.
- **GraphQL.** Federated query for multi-controller deployments (one query → multiple controllers).
- **Language SDKs.** ONOS ships a Java API; ODL has Karaf bundles; Ryu is itself a Python library.
- **Intent-based.** A higher-level NBI where the application says *what* it wants, not *how*. Examples: ONOS Intent Framework, Cisco DNA-C intent API, Apstra.

A typical NBI request and response (REST):

```http
POST /onos/v1/flows/of:0000000000000001 HTTP/1.1
Content-Type: application/json

{
  "priority": 40000,
  "timeout": 0,
  "isPermanent": true,
  "deviceId": "of:0000000000000001",
  "treatment": { "instructions": [{"type": "OUTPUT", "port": "2"}] },
  "selector": { "criteria": [
    {"type": "ETH_TYPE", "ethType": "0x0800"},
    {"type": "IPV4_DST", "ip": "10.0.0.42/32"}
  ]}
}
```

### Southbound

The southbound is the boundary where the controller meets the silicon. Choice of SBI is a meaningful architectural decision:

| Protocol | Best at | Limitations |
|:---|:---|:---|
| OpenFlow | Match-action programming, real-time flow installs | Ossifies on a fixed pipeline; vendor support uneven |
| NETCONF / YANG | Config-style management of vendor boxes | Higher latency; not packet-level; transaction semantics weak in practice |
| gNMI | Streaming telemetry + config; gRPC fast and clean | Newer; less ubiquitous |
| P4Runtime | Pipeline-agnostic, target-agnostic forwarding install | Requires P4-compiled target |
| BGP-LS | Topology export from existing IGPs | Read-only; no install |
| PCEP | Path computation requests / replies; MPLS-TE | Single-purpose |
| OVSDB | Open vSwitch configuration | OVS-specific |

Real controllers speak several southbound protocols simultaneously — OpenFlow to white-box switches, NETCONF to vendor routers, gNMI for telemetry, BGP-LS to learn topology from a service provider IGP.

### Interface semantics

A clean SDN architecture treats the SBI as the *contract* of the data plane and the NBI as the *contract* of the controller. The controller's job is to translate between them. Every architectural decision (which abstractions to expose northbound, which to push southbound) reduces to compiler-design questions: where does the boundary sit, how rich is each side, who pays which cost.

---

## Controller architectures

A controller is the binary that holds the centralized state and runs the algorithms. Three architectural shapes dominate.

### Centralized

One controller process, one machine, one source of truth. Examples: NOX (the original OpenFlow controller, ~2008), Floodlight, the inner core of Ryu.

```
                     +--------------+
                     |  Controller  |
                     +-------+------+
                             |
          +------+-----------+----------+------+
          |      |                      |      |
        +-+-+  +-+-+                  +-+-+  +-+-+
        | S |  | S |       ...        | S |  | S |
        +---+  +---+                  +---+  +---+
```

- **Pros.** Simple. Strongly consistent state by construction. Easy to reason about.
- **Cons.** Single point of failure. Doesn't scale beyond ~hundreds of switches before flow-mod throughput becomes the bottleneck.
- **When.** Lab, small DC, edge / branch SDN where switch counts are bounded.

### Distributed

Multiple controller instances cooperate over a consensus protocol, typically Raft, Paxos, or ZooKeeper. Examples: ONOS, OpenDaylight clustering, Faucet with multiple controllers.

```
        +-------+    Raft / ZK    +-------+    Raft / ZK   +-------+
        | C1    |<--------------->| C2    |<-------------->| C3    |
        +---+---+                 +---+---+                +---+---+
            |                         |                        |
            |                         |                        |
            |  switches partitioned by mastership / shardkey  |
            v                         v                        v
        +---+---+   ...          +---+---+    ...         +---+---+
        |  S1   |                |  Sk   |                |  Sn   |
        +-------+                +-------+                +-------+
```

Mastership semantics — at any moment, exactly one controller is the *master* for a given switch; the others are *standbys*. The master is the only writer. A failover is an atomic re-election handled by the consensus protocol.

State sharding is essential for scale. A naive replicated log forces every controller to process every flow event; in a sharded cluster each controller owns a partition (e.g. switch ID mod N).

- **Pros.** No single point of failure. Horizontal scale to thousands of switches.
- **Cons.** Distributed-system complexity: consistency models, split-brain risk, failover latency, rolling upgrade strategy.
- **When.** Production deployments at any meaningful scale.

### Hierarchical

Two-tier: a fleet of *regional* controllers each manage their slice of the network, and a *global* controller stitches them together. Used in WAN SDN where regions are geographically distant and a single Raft log across continents would be impractical.

```
                    +-------------+
                    |   Global    |
                    | controller  |
                    +------+------+
                           |
       +-------------------+-------------------+
       |                   |                   |
+------+------+    +-------+-----+    +--------+-----+
| Regional A  |    | Regional B  |    | Regional C   |
| controller  |    | controller  |    | controller   |
+------+------+    +-------+-----+    +--------+-----+
       |                   |                   |
       v                   v                   v
   switches           switches           switches
```

Google B4 and Microsoft SWAN use this pattern. The global controller does inter-region traffic engineering; the regional controllers do intra-region forwarding.

### Failure modes

- **Split-brain.** Two controllers each believe they are master for the same switch. Dual writes corrupt the FIB. Mitigation: quorum-based mastership, fencing tokens, generation IDs on every flow-mod.
- **Controller failover timing.** From master loss to new master enacting writes: typically 1-5 seconds in production. During the gap, switches keep forwarding using their cached flow tables but cannot install new flows.
- **Switch fail-secure vs fail-standalone.** When a switch loses controller contact, OpenFlow defines two modes:
    - *fail-secure.* Drop unmatched packets. Existing flows continue to forward.
    - *fail-standalone.* Fall back to a local L2 learning bridge.
  Production deployments typically use fail-secure with long flow-entry timeouts so the data plane survives a controller outage of minutes-to-hours.
- **Slow controller.** flow-mod throughput backs up; reactive flows starve; new connections time out. A common production lesson: **always go proactive**; never depend on per-flow controller round-trips.
- **Network partition between controllers.** Quorum loss → no writes possible → degraded mode.

### Failover timing math

Detection latency $T_d$ (typically 1-3 BFD intervals if BFD is between controller and switch):

$$T_d = k \cdot T_{BFD}, \quad k = 3$$

Election latency $T_e$ (Raft):

$$T_e \approx T_{election\_timeout} + T_{round\_trip}$$

where election timeout is randomized in some range (Raft default ~150-300 ms) and the leader candidate needs a majority vote.

State load latency $T_l$ (new master pulls / verifies switch state):

$$T_l \approx N_{sw\_per\_master} \cdot T_{flow\_stats}$$

Total recovery:

$$T_{recovery} = T_d + T_e + T_l$$

For a 100 ms BFD interval, 200 ms election timeout, 1000 switches at 1 ms each, total recovery is roughly $300 + 200 + 1000 = 1500$ ms. Hyperscale deployments push this under 500 ms by aggressive BFD intervals (10-50 ms) and pre-staged state on standby controllers.

---

## P4 programmable data plane

OpenFlow assumes the silicon already understands fixed protocols (Ethernet, IPv4, MPLS, etc.). P4 says *the silicon shouldn't bake in any protocols*; instead, the application defines parsers, tables, and pipeline stages.

### Language semantics

A P4 program is organized into:

- **Headers** — typed bit-vector definitions of protocol fields.
- **Parser** — a state machine that walks bytes from the wire into typed headers.
- **Tables** — match-action tables. Match keys are defined declaratively; actions are user-defined functions.
- **Actions** — read / write headers, drop, set egress port, increment counters, push/pop labels, hash, recirculate.
- **Controls** — user code that orchestrates table lookups (apply tables in sequence, conditionally).
- **Deparser** — emits headers back to the wire in deterministic order.

A minimal P4 (P4_16) example:

```c
#include <core.p4>
#include <v1model.p4>

header ethernet_t {
    bit<48> dstAddr;
    bit<48> srcAddr;
    bit<16> etherType;
}

header ipv4_t {
    bit<4>  version;
    bit<4>  ihl;
    bit<8>  diffserv;
    bit<16> totalLen;
    bit<16> identification;
    bit<3>  flags;
    bit<13> fragOffset;
    bit<8>  ttl;
    bit<8>  protocol;
    bit<16> hdrChecksum;
    bit<32> srcAddr;
    bit<32> dstAddr;
}

struct headers {
    ethernet_t ethernet;
    ipv4_t     ipv4;
}

parser MyParser(packet_in pkt, out headers hdr,
                inout standard_metadata_t std_meta) {
    state start { transition parse_eth; }
    state parse_eth {
        pkt.extract(hdr.ethernet);
        transition select(hdr.ethernet.etherType) {
            0x0800: parse_ipv4;
            default: accept;
        }
    }
    state parse_ipv4 {
        pkt.extract(hdr.ipv4);
        transition accept;
    }
}

control MyIngress(inout headers hdr,
                  inout standard_metadata_t std_meta) {
    action drop() { mark_to_drop(std_meta); }
    action ipv4_forward(bit<9> port, bit<48> dmac) {
        std_meta.egress_spec = port;
        hdr.ethernet.dstAddr = dmac;
        hdr.ipv4.ttl = hdr.ipv4.ttl - 1;
    }

    table ipv4_lpm {
        key = { hdr.ipv4.dstAddr : lpm; }
        actions = { ipv4_forward; drop; NoAction; }
        size = 1024;
        default_action = drop();
    }

    apply {
        if (hdr.ipv4.isValid()) ipv4_lpm.apply();
    }
}

control MyDeparser(packet_out pkt, in headers hdr) {
    apply {
        pkt.emit(hdr.ethernet);
        pkt.emit(hdr.ipv4);
    }
}

V1Switch(MyParser(), verifyChecksum(),
         MyIngress(), MyEgress(),
         computeChecksum(), MyDeparser()) main;
```

This is a complete IPv4 LPM forwarder in ~80 lines. The P4 compiler maps it onto the target's pipeline (BMv2 software switch, Tofino ASIC, FPGA, smart NIC).

### PISA (Protocol-Independent Switch Architecture)

PISA is the abstract machine that P4 targets. Conceptually:

```
+-----------+   +-----------+   +-----+-----+-----+-----+   +-------------+
|  Parser   |-->|  Match-   |-->|  M  |  M  |  M  |  M  |-->|  Deparser   |
| (state    |   |  Action   |   |  A  |  A  |  A  |  A  |   |             |
|  machine) |   |  Stage 0  |   |     |     |     |     |   |             |
+-----------+   +-----------+   +-----+-----+-----+-----+   +-------------+
                                  Stage 1...N (configurable
                                  TCAM + SRAM + ALU per stage)
```

Each stage is a few-nanosecond pipelined unit with:

- a TCAM bank for ternary / LPM lookup
- an SRAM bank for exact / hash lookup
- a configurable ALU bank for header rewrites and stateful operations
- counters / meters / registers for stateful programs

Crucially the *protocol* is not baked in; the parser and tables are programmable. A PISA switch can be told to parse and forward GTP-U for a 5G UPF, or RoCEv2 for an HPC fabric, or some custom in-network telemetry header.

### Tofino + Tofino 2 capabilities

Intel's Tofino (originally Barefoot Networks) is the first widely deployed P4 ASIC.

| Generation | Bandwidth | Pipelines | Lookup capacity / pipeline |
|:---:|:---:|:---:|:---|
| Tofino 1 | 6.5 Tbps | 4 | ~12 stages, ~10M LPM, ~64M exact |
| Tofino 2 | 12.8 Tbps | 4-8 | ~20 stages, larger SRAM/TCAM banks |

Tofino 2 added per-pipeline egress, more stages, and more flexible parser.

### In-network telemetry (INT)

INT lets the data plane embed measurement data into production packets so collectors can see hop-by-hop latency, queue depth, and path without the controller polling.

Two flavors:

- **INT-MD (metadata).** Each switch on the path inserts a metadata stack into a tunnelled outer header. Last hop strips it and exports.
- **INT-XD / INT-MX.** Switches export telemetry separately; the packet itself carries only an instruction header.

What gets recorded per hop:

```
- switch_id            (which device handled the packet)
- ingress_port         (which port it came in on)
- egress_port          (which port it went out on)
- ingress_timestamp    (ns precision)
- egress_timestamp     (ns precision)
- queue_id, queue_depth, queue_drops
- latency = egress - ingress
```

A 4-hop traversal therefore produces a stack of 4 telemetry blocks attached to the packet itself. At line rate. Without polling. INT closes the loop on closed-loop SDN.

---

## ONOS / OpenDaylight

The two open-source production SDN controllers. Both Java, both OSGi-based, both target large carrier and DC deployments.

### ONOS (Open Network Operating System)

Architectural primitives:

- **Distributed core.** ONOS runs as a cluster (3, 5, or 7 nodes typical) with sharded mastership. Built on Atomix (Raft) for strong consistency.
- **Network graph.** A versioned, queryable model of the topology: devices, ports, links, hosts.
- **Flow rule subsystem.** A *desired-state* subsystem. Apps install rules into ONOS; ONOS reconciles with the data plane through the SBI provider, retrying on failure.
- **Intent framework.** Apps express *what* they want (point-to-point, multipoint-to-single, host-to-host, path constraints). The intent compiler translates intents into flow rules. Recompilation happens on topology changes.
- **Apps (sub-projects).** Forwarding, VPLS, BGP speaker, segment routing, ACL, OpenStack Neutron driver, SD-Fabric, Trellis.

#### Intent-rule compilation pipeline

```
HostToHostIntent("h1", "h2")
        |
        v
PathIntent: ordered list of links (h1 -> S1 -> S3 -> S5 -> h2)
        |
        v
LinkCollectionIntent: per-link forwarding directives
        |
        v
FlowRuleIntent: concrete flow-mod messages, switch-by-switch
        |
        v
Driver / SBI provider: actual OpenFlow / P4Runtime / NETCONF wire messages
```

When a link in the chosen path goes down, ONOS recompiles only the affected intents. If no alternate path exists the intent enters FAILED state; otherwise it transitions through INSTALLING and back to INSTALLED with the new path's flow rules.

#### Flow rule subsystem

The reconciler runs continuously. Three rule states:

| State | Meaning |
|:---|:---|
| PENDING_ADD | Controller wants this rule; SBI not yet acked |
| ADDED | SBI confirms install |
| PENDING_REMOVE | Controller wants this rule deleted |
| REMOVED | SBI confirms delete |

Periodic reconciliation asks each device for its current flow stats and compares against the desired state. Drifted rules (rules that the controller did not install but appeared on the device) are deleted; missing rules are reinstalled.

### OpenDaylight (ODL)

Architectural primitives:

- **MD-SAL (Model-Driven Service Abstraction Layer).** YANG-defined data trees as the single canonical store. Services subscribe to data-tree changes.
- **YANG everywhere.** Every API, north or south, is YANG-modelled. A YANG schema generates the binding code, the REST/RESTCONF API, and the data store schema.
- **NETCONF, OpenFlow, BGP-LS, PCEP, OVSDB plugins.** Each southbound protocol is a Karaf bundle that translates wire-protocol semantics into MD-SAL data-tree operations.
- **AAA.** Pluggable auth, often integrated with Keystone for OpenStack deployments.

ODL's data-driven philosophy contrasts with ONOS's intent-driven philosophy. They are not incompatible — both can model both — but the centerpiece differs.

### Policy compilation

Both controllers translate higher-level policy into low-level flow rules. The compilation cost is a real constraint: for a fabric with $S$ switches, $H$ hosts, $L$ links, naive intent recompilation has cost roughly:

$$T_{compile} = O(I \cdot \text{shortest-path}(S, L)) = O(I \cdot (S \log S + L))$$

with $I$ = number of intents impacted by the topology change. For $I = 10^4$, $S = 10^3$, $L = 10^4$, that's on the order of $10^4 \cdot 10^4 = 10^8$ operations — seconds to tens of seconds. ONOS uses incremental recompilation (only intents whose path actually broke are recomputed) to cut this to milliseconds in typical cases.

---

## SDN at scale — Google B4

The B4 paper (SIGCOMM 2013) is the canonical proof that SDN works at hyperscale. Google's inter-datacenter WAN connects ~30 sites worldwide with ~Tbps of long-haul capacity.

### What B4 replaces

Traditional WAN traffic engineering uses MPLS-RSVP-TE: tunnels are signalled hop by hop, bandwidth is reserved by RSVP, paths are chosen by CSPF. The result is *online distributed* traffic engineering — every router runs CSPF, every router negotiates reservations.

B4 replaces this with a *centralized offline TE solver*. A controller knows all link capacities, all flow demands, all priorities. It solves an optimization problem to map demands to paths. It pushes the resulting label stacks into the routers.

### Architecture

```
+-------------------+
|  Global TE        |
|  controller       |
|  (optimization)   |
+---------+---------+
          |  intents per region
          v
+-------------------+
|  Site controller  |  one per site; talks OpenFlow to local switches
|  (gateway)        |
+---------+---------+
          |  flow-mods
          v
+-------------------+
|  WAN switches     |  custom merchant-silicon white boxes
|  (OpenFlow)       |
+-------------------+
```

### TE solver math

Given:

- $V$ = set of sites (vertices)
- $E$ = set of links (edges)
- $D$ = set of (src, dst, demand_bps, priority) flow demands
- $c_{ij}$ = capacity of link $(i, j)$

Minimize the max link utilization (or maximize total carried demand subject to fairness):

$$\min \quad \alpha$$

$$\text{subject to:} \quad \sum_{p \ni (i,j)} f_p \leq \alpha \cdot c_{ij} \quad \forall (i,j) \in E$$

$$\sum_{p \in P_d} f_p = \text{served}(d) \quad \forall d \in D$$

$$\text{served}(d) \leq \text{demand}(d)$$

$$f_p \geq 0$$

This is a multi-commodity flow LP. Naive LP complexity is roughly $O(V^2 E)$ per simplex pivot times number of pivots; in practice B4 uses bandwidth-allocation heuristics (max-min fairness with priority levels) instead of solving the full LP.

### Numbers from the paper

- B4 achieves close to 100% link utilization on bottleneck links.
- Traditional WANs typically run at 30-40% utilization to leave headroom for failures.
- B4's centralized TE delivers $\approx 2\text{-}3\times$ the carried traffic on the same fibre.

### Failure recovery

B4 splits failure response into two timescales:

- **Fast.** Local data-plane fast reroute (FRR) reroutes around a failure within tens of milliseconds without touching the controller. Pre-installed backup label stacks on every switch.
- **Slow.** The TE solver re-runs to reflect the new topology; it pushes a new optimal mapping in seconds. While the new map is being computed, FRR carries the load.

Sub-second TE recompute is achievable because the optimization problem is constrained: only ~30 sites, ~hundreds of demands, ~hundreds of links. The state space is small enough for a centralized solver, even given the LP cost above.

### Lessons

- Centralized TE works at the WAN scale because the problem is small (sites count, not switches count).
- Fast reroute in the data plane is non-negotiable; controllers cannot meet failover SLAs alone.
- Hardware uniformity (Google's white boxes) drastically simplifies the controller.
- The economic case is overwhelming: doubling effective WAN capacity with no new fibre pays for the controller infrastructure many times over.

---

## Network virtualization

Network virtualization decouples logical L2/L3 topology from physical topology. Three encapsulations dominate.

### VXLAN (RFC 7348)

Virtual eXtensible LAN: an Ethernet-in-UDP encapsulation. The header carries a 24-bit VNI (VXLAN Network Identifier).

#### Header format

```
+-------------------------------------------------+
| Outer Ethernet (14 / 18 bytes)                 |
+-------------------------------------------------+
| Outer IP (20 bytes IPv4 / 40 IPv6)             |
+-------------------------------------------------+
| Outer UDP (8 bytes; src=hash, dst=4789)        |
+-------------------------------------------------+
| VXLAN (8 bytes)                                |
|  Flags(8) | Reserved(24) | VNI(24) | Resv(8)   |
+-------------------------------------------------+
| Inner Ethernet (14 / 18 bytes)                 |
+-------------------------------------------------+
| Inner payload                                  |
+-------------------------------------------------+
```

#### VNI math

$$N_{VNI} = 2^{24} = 16{,}777{,}216$$

vs $N_{VLAN} = 2^{12} = 4096$. Expansion factor $2^{12} = 4096\times$.

#### Overhead math

| Outer | Inner | UDP | VXLAN | Total tunnel overhead |
|:---:|:---:|:---:|:---:|:---:|
| Eth(14) + IPv4(20) | (none) | 8 | 8 | **50 bytes** |
| Eth(14) + IPv6(40) | (none) | 8 | 8 | **70 bytes** |

Effective MTU for the inner frame:

$$MTU_{inner} = MTU_{outer} - 50 \text{ (IPv4)} = 1500 - 50 = 1450$$

To preserve a 1500-byte inner MTU, the underlay must run at 1550 bytes (jumbo or MTU-discovery).

#### UDP source port

The outer UDP source port is computed as a hash of the inner 5-tuple. This drives ECMP across the underlay so that flows in the same VNI but different inner 5-tuples spread across multiple paths.

### Geneve (RFC 8926)

Generic Network Virtualization Encapsulation. Designed to obsolete both VXLAN and NVGRE with extensibility.

Header:

```
+-------------------+
| Outer Ethernet    |
+-------------------+
| Outer IP          |
+-------------------+
| Outer UDP (6081)  |
+-------------------+
| Geneve base (8 B) |   Ver | OptLen | O | C | Rsvd | ProtoType | VNI(24) | Rsvd
+-------------------+
| Variable TLVs     |   0..N option TLVs (for metadata, telemetry, ACLs)
+-------------------+
| Inner frame       |
+-------------------+
```

Geneve's signature feature: variable-length TLV options. A 24-bit VNI for tenancy plus arbitrary metadata for service chaining, telemetry, ACLs, you name it.

### NVGRE (RFC 7637)

Network Virtualization using GRE. Microsoft-led. Less commonly deployed today; VXLAN dominates.

```
+-------------------+
| Outer Ethernet    |
+-------------------+
| Outer IP          |
+-------------------+
| GRE (8 bytes)     |   C(0) | K(1) | S(0) | Reserved | Version(0) | ProtoType
|                   |   VSID(24) | FlowID(8)
+-------------------+
| Inner Ethernet    |
+-------------------+
| Inner payload     |
+-------------------+
```

VSID is the 24-bit tenant ID — same address space size as VXLAN. The 8-bit FlowID is meant for ECMP hashing the way VXLAN uses the UDP source port.

### Comparison

| Property | VXLAN | Geneve | NVGRE |
|:---|:---:|:---:|:---:|
| VNI / VSID width | 24 bits | 24 bits | 24 bits |
| Transport | UDP/4789 | UDP/6081 | GRE |
| Header size | 8 B fixed | 8 B base + variable | 8 B fixed |
| Extensibility | poor | excellent (TLVs) | poor |
| ECMP friendliness | excellent (UDP src) | excellent (UDP src) | mediocre (FlowID 8 b) |
| Deployment | dominant | growing (OVS, NSX) | declining |

---

## Service chaining (NFV)

Service chaining strings packets through an ordered sequence of network functions: firewall → IPS → NAT → load balancer → encryptor → wire. Two principal mechanisms.

### NSH (Network Service Header) RFC 8300

NSH is a service-aware shim header inserted between the underlay and the payload. Functions are addressed by Service Path ID + Service Index.

```
+----------------+
| Outer encap    |   Eth + IP + UDP/VXLAN-GPE or GRE
+----------------+
| NSH Base (4 B) |   Ver | OAM | UnAssigned | TTL | Length | MD-Type | NextProto
+----------------+
| NSH Service    |   Service Path ID (24 b) | Service Index (8 b)
+----------------+
| Context (var)  |   MD-Type 1: 4 32-bit context fields; MD-Type 2: variable TLVs
+----------------+
| Inner payload  |
+----------------+
```

- **Service Path ID** identifies *which chain* this packet is on (24 bits = 16M paths).
- **Service Index** decrements at each function. Reaches 0 = end of chain. (Packets with TTL/SI = 0 are dropped to prevent loops.)
- **Context headers** carry per-flow metadata between functions: classification result, customer ID, telemetry hooks.

A classifier inserts the NSH at chain ingress; a service function forwarder (SFF) routes between functions; a chain-egress strips the NSH and emits the original packet.

### Segment Routing IPv6 (SRv6)

SRv6 (RFC 8402, RFC 8754) uses IPv6 addresses themselves as segment identifiers. A packet's IPv6 routing extension header carries an ordered list of SIDs; the destination address is rewritten at each segment endpoint.

```
+--------+--------+--------+--------+--------+--------+
| outer src IPv6 | outer dst IPv6 (= active segment)  |
+--------+--------+--------+--------+--------+--------+
| Routing Hdr (RH; type 4)                            |
|  NextHdr | HdrExtLen | RoutingType=4 | SegmentsLeft |
|  LastEntry | Flags | Tag                            |
+----------------------------------------------------+
|  Segment List [0..N]: each is an IPv6 address      |
|    SID 0  (last segment)                           |
|    SID 1                                            |
|    ...                                              |
|    SID N  (first segment after dst)                 |
+----------------------------------------------------+
```

A SID encodes both *locator* (how to get to that segment endpoint) and *function* (what the endpoint should do). End.X (cross-connect to a specific link), End.DT4 (decapsulate and lookup in IPv4 table N), End.B6 (encapsulate with new SRH) — and many more.

For service chaining, define a SID per service function. The classifier pushes a segment list ordered by chain. Each function endpoint decrements SegmentsLeft, rewrites destination to the next SID, and forwards. End-of-list executes the final action (typically decap + L3 lookup).

### NSH vs SRv6 comparison

| Property | NSH | SRv6 |
|:---|:---|:---|
| Header location | Between transport and payload | IPv6 extension header |
| Identifier space | 24-bit Service Path ID | 128-bit SIDs (vast) |
| Native IP-routable? | No (needs a transport encap) | Yes |
| Hop-by-hop processing | Yes (every SFF) | Optional (only at SID endpoints) |
| Per-flow metadata | First-class (context headers) | Possible via TLVs |
| Industry trend | Stable, niche | Ascendant, dominant in SP |

### Service Function Path (SFP) composition

Choosing a chain has classifier and routing components:

```
1. Classifier sees packet, runs match logic (5-tuple, app signature, customer ID).
2. Classifier writes (SPI, SI) into NSH header (or pushes SR list).
3. Each SFF reads (SPI, SI), looks up an SFP table, forwards to the next service.
4. Service function processes, may modify metadata, decrements SI.
5. Last service: SI = 0 → SFF strips header and forwards inner packet to underlay.
```

The composition surface is the SFP table itself — adding a new chain means inserting a new (SPI → ordered list of SF endpoints) entry.

---

## SD-WAN

SD-WAN is SDN applied to the branch-office WAN problem. Vendor-led; typically not interoperable; commercially dominant since 2017.

### Architecture

```
                +------------------+
                |  Cloud-hosted    |
                |  controller +    |
                |  orchestrator    |
                +--------+---------+
                         |
        DC overlay     +---+---+   ...
        +-----+        |   |   |
        |  Hub|========|       |========  Branch n
        +-----+        |   ...
                       |   |
                    Branch 1, 2, ...
```

The controller runs in the vendor's cloud (or in a customer-managed VM). Branch *edge devices* (CPE) build IPsec or DTLS tunnels back to one or more *hub* sites. The control plane is a separate path; the data plane uses the public Internet, MPLS, LTE, or whatever transport(s) the branch has.

### Dynamic path selection by SLA

The CPE measures each available transport continuously: latency, jitter, loss. Each application has an SLA policy: VoIP needs <150 ms latency, <30 ms jitter, <1% loss; web traffic tolerates more.

```
For each packet:
    a = classify_application(pkt)            # DPI / signature
    candidates = transports_meeting_sla(a)   # list of (link, score)
    if candidates:
        link = best(candidates)
    else:
        link = best_available()              # graceful degradation
    encap_and_forward(pkt, link)
```

This is "policy-aware ECMP." A flow's ECMP target can change mid-flow if the current link breaches SLA, leading to *application-aware* traffic engineering at the branch edge.

### Vendor flavors

| Vendor | Product | Notes |
|:---|:---|:---|
| Cisco | Viptela (acquired 2017) | vManage / vSmart / vBond / vEdge |
| VMware | VeloCloud (acquired 2017) | Now part of Broadcom |
| HPE | Silver Peak / Aruba EdgeConnect | Acquired by HPE 2020 |
| Fortinet | FortiGate SD-WAN | Integrated with FortiGate firewalls |
| Versa | Versa SD-WAN | Often deployed by SPs |
| Palo Alto | Prisma SD-WAN (CloudGenix) | Identity-based policy focus |

The architecture is broadly similar across vendors; the differentiation is in the controller's policy engine, the supported underlay diversity, the orchestration UX, and the embedded security stack (firewall, secure web gateway, CASB).

### When SD-WAN, when classical WAN

- **SD-WAN.** Many small branches. Mixed transport (broadband + LTE + MPLS). High frequency of policy changes. Cloud-application-heavy traffic patterns.
- **Classical WAN (MPLS / DMVPN / IPsec).** Few sites with large, stable workloads. Predictable traffic patterns. Strong SLA commitments from a single carrier. Compliance requirements that constrain which transport can carry which data.

---

## Intent-based + closed-loop

Intent-based networking (IBN) is the next abstraction above flow rules. Apps say *what they want*; the controller translates to flow rules, then watches the network and re-converges if reality drifts.

### Pipeline

```
       +------------+
       |  Operator  |   "I want any device tagged 'guest' to reach only the
       |  / Apps    |    Internet, never internal subnets."
       +-----+------+
             |
             v
       +------------+
       |  Intent    |   POST /intent { from: "tag:guest", to: "any:Internet",
       |  layer     |                    deny: "subnet:internal" }
       +-----+------+
             |
             v
       +------------+
       |  Policy    |   Compile to per-device ACLs, per-flow VRF assignments,
       |  layer     |   per-vSwitch security group rules.
       +-----+------+
             |
             v
       +------------+
       |  Flow rule |   Concrete OpenFlow / NETCONF / gRPC messages on the wire.
       |  layer     |
       +-----+------+
             |
             v
       +------------+
       |  Telemetry |   gNMI subscriptions, INT, sFlow, NetFlow.
       |  feedback  |   "The network said it would do X; is it actually doing X?"
       +-----+------+
             |
             v
       +------------+
       |  Drift     |   Compare measured state to compiled policy. If they
       |  detector  |   disagree, re-compile and re-push.
       +------------+
```

### Intent specifications

A simple intent expressed in YAML:

```yaml
apiVersion: intent.networking/v1
kind: Connectivity
metadata:
  name: guest-internet-only
spec:
  source:
    selector: "tag=guest"
  destination:
    selector: "any:internet"
  constraints:
    bandwidth: "100mbps"
    latency: "< 50ms"
  exclude:
    - selector: "subnet:10.0.0.0/8"
    - selector: "subnet:172.16.0.0/12"
    - selector: "subnet:192.168.0.0/16"
  action: ALLOW
```

The compiler walks the topology, picks a path that meets the constraints, generates flow rules + ACLs + VRF leaks, pushes them, and registers a telemetry watch.

### Telemetry feedback loop

Three telemetry pipelines feed the closed loop:

- **gNMI streaming.** Subscribes to interface counters, queue depth, control-plane CPU, flow-cache stats. Sub-second granularity.
- **INT.** In-band telemetry stamps every packet (or sampled packets) with hop info.
- **sFlow / NetFlow / IPFIX.** Sampled flow records exported every few seconds.

The drift detector runs periodically. If `expected.flow_rules != observed.flow_rules`, the controller re-pushes. If `expected.SLA != observed.SLA`, the policy compiler re-runs (perhaps choosing a different path).

### Drift detection and re-converge

```python
def detect_drift(controller, devices):
    expected = controller.flow_rules
    observed = {d: d.dump_flows() for d in devices}
    drift = []
    for d in devices:
        missing = expected[d] - observed[d]
        extra   = observed[d] - expected[d]
        if missing or extra:
            drift.append((d, missing, extra))
    return drift

def reconverge(controller, drift):
    for d, missing, extra in drift:
        for rule in extra:
            controller.delete_flow(d, rule)
        for rule in missing:
            controller.install_flow(d, rule)
```

The reconverge loop runs continuously. In production it operates in seconds, not minutes — a fabric whose state is stale by 30 s is acceptable; one stale by 30 minutes is not.

---

## SDN security threat model

Centralization is a sword with two edges. The controller's global view is the same view an attacker would dream of having.

### Threat: controller compromise

If an attacker owns the controller, the attacker owns the network. Fully. Every flow rule, every VRF assignment, every ACL. They can install drop rules to deny service, install duplicate rules to mirror traffic to themselves, install loop rules to break the data plane.

Mitigations:

- **Defense in depth.** Network segmentation around the controller; admin access requires MFA + short-lived certs; no flat layer-2 to controller management interfaces.
- **Hardware roots of trust.** TPM attestation on controller VMs, secure boot.
- **Audit trail.** Every flow-mod logged with operator, app, intent, timestamp. Cryptographic chaining if the audit log is itself a high-value target.
- **Fail-secure data plane.** Switches retain last-known-good flow tables on controller loss; they refuse new flow installs from a controller whose certificate doesn't validate.
- **Quorum.** Multi-controller architectures require a majority to authorize state changes; an attacker with one controller cannot push.

### Threat: southbound DoS

If the controller can be DoSed via packet-in floods, the data plane stops getting new flow rules. Common attack: send a flood of unique flows to a switch with a default reactive flow-mod policy.

Mitigations:

- **Proactive flow installation.** Don't rely on reactive packet-in for normal traffic.
- **Rate-limit packet-in.** Each switch caps packet-in pps to the controller; over the cap, drop.
- **Per-MAC / per-port quotas.** A single misbehaving host cannot consume the entire packet-in budget.
- **Flow-mod throttling.** The controller itself rate-limits incoming northbound API calls.

### Threat: data plane misdirection

Attacker injects packets crafted to match overly-broad flow rules and divert traffic. Closely related: rogue switch reports false LLDP / BGP-LS topology to the controller, which compiles policy assuming false topology.

Mitigations:

- **Authenticated topology discovery.** LLDP-DS, BGP-LS with TCP-AO, signed link advertisements.
- **Strict flow rule precision.** Avoid wildcards in security-critical rules; favour exact-match.
- **Path verification.** INT or signed traceroute confirms packets traversed the path the controller intended.
- **Switch attestation.** Boot-time measurements of switch firmware checked against a trusted baseline.

### Threat: northbound API abuse

A compromised app (or developer credentials) installs malicious intents.

Mitigations:

- **App sandboxing.** Apps cannot install rules outside their authorized scope.
- **RBAC on intents.** Each app / user has rights to a subset of the intent space.
- **Signed intents.** Critical intents require a signed manifest reviewed by a separate operator.

### Threat: privacy and traffic interception

A controller that sees every flow header sees a great deal of metadata. INT amplifies this with per-packet hop traces.

Mitigations:

- **Minimize the metadata controllers see.** Sample, don't capture.
- **Encrypt control channels.** TLS between controllers, between controller and apps, between controller and switch (OpenFlow over TLS).
- **Compartmentalize.** Per-tenant controllers in multi-tenant deployments.

---

## Worked examples

### Example 1 — 4-switch tree with OpenFlow controller, flow installation order

Topology:

```
        Controller (10.0.0.1)
        /  |  |  \
       /   |  |   \
     S1---S2  S3---S4
      |    |   |    |
     h1   h2  h3   h4

Links:    S1-S2, S2-S3, S3-S4
Hosts:    h1@S1:1, h2@S2:1, h3@S3:1, h4@S4:1
```

A reactive controller learns. Trace of a single ping from h1 to h4 (assume empty FIBs initially):

```
1. h1 sends ARP for h4's MAC.
2. S1 has table-miss flow -> packet-in to controller.
3. Controller learns h1's MAC@S1:1.
4. Controller has no entry for h4 -> flood ARP to all edge ports.
5. h4 receives ARP, replies. ARP reply enters S4.
6. S4 has table-miss flow -> packet-in to controller.
7. Controller learns h4's MAC@S4:1, computes shortest path:
       S1 -> S2 -> S3 -> S4
8. Controller issues flow-mod messages:
       At S1: match (eth_dst=h4_mac) -> output S1:port-to-S2
       At S2: match (eth_dst=h4_mac) -> output S2:port-to-S3
       At S3: match (eth_dst=h4_mac) -> output S3:port-to-S4
       At S4: match (eth_dst=h4_mac) -> output S4:1 (h4 port)
       And reverse direction (eth_dst=h1_mac) -> back through path
9. Controller issues packet-out at S4 to deliver the original ARP reply to h1
   via the just-installed reverse path (or sends it directly).
10. ARP exchange completes.
11. h1 sends ICMP Echo Request. S1 hits the installed flow:
       eth_dst=h4_mac -> S1:port-to-S2  (forwards to S2 in hardware)
       At S2, S3, S4 same.
12. h4 receives Echo Request, sends Reply. Path is symmetric and pre-installed.
```

Key observations:

- The first packet of a new flow pays a controller round-trip ($T_{rtt} \approx 1\text{-}5\text{ ms}$ in lab, more in production).
- All subsequent packets forward at line rate.
- The flow rule timeouts (idle = 60 s, hard = 0) eventually free the entries when the conversation ends.

### flow-mod messages on the wire (textual sketch)

```
flow-mod table=0
  match: eth_dst=02:00:00:00:00:04
  priority: 40000
  instructions: apply_actions(output=2)   # port to next-hop switch
  cookie: 0x1234abcd
  idle_timeout: 60
  hard_timeout: 0
```

This is the controller's wire-level abstraction: a sequence of such messages, one per switch on the path.

### Example 2 — VXLAN encap byte-level walk-through

Inner Ethernet frame from host h1 to host h4 over VNI 50000:

```
Inner Ethernet (14 B):
  dst MAC = 02:00:00:00:00:04
  src MAC = 02:00:00:00:00:01
  ethertype = 0x0800 (IPv4)

Inner IPv4 (20 B):
  ver/ihl = 0x45
  ttl = 64
  proto = 6 (TCP)
  src = 10.0.0.1
  dst = 10.0.0.4

Inner TCP (20 B):
  sport = 51234
  dport = 80

Payload: 100 bytes
```

Total inner frame: 14 + 20 + 20 + 100 = 154 bytes.

VTEP A wraps:

```
Outer Ethernet (14 B):
  dst MAC = MAC(VTEP B)
  src MAC = MAC(VTEP A)
  ethertype = 0x0800

Outer IPv4 (20 B):
  ttl = 64
  proto = 17 (UDP)
  src = 192.168.0.1   (VTEP A loopback)
  dst = 192.168.0.4   (VTEP B loopback)

Outer UDP (8 B):
  src port = hash(inner_5tuple) -> 49152
  dst port = 4789
  length = 8 + 8 + 154 = 170

VXLAN header (8 B):
  flags = 0x08 (I bit set; VNI valid)
  reserved = 0
  VNI = 50000  (encoded in 24 bits = 0x00C350)
  reserved = 0

Inner Ethernet through payload (154 B as above)
```

Outer-to-inner total: 14 + 20 + 8 + 8 + 154 = **204 bytes** on the wire.

Overhead: 50 bytes (everything before inner Ethernet). If the underlay MTU is 1500, the largest inner frame VXLAN can carry without IP fragmentation is 1450 bytes.

To preserve a 1500-byte inner MTU, configure the underlay to 1550:

$$1500 + 50 = 1550 \quad \text{(IPv4 underlay)}$$
$$1500 + 70 = 1570 \quad \text{(IPv6 underlay)}$$

Most DC fabrics set 9100-byte jumbo frames on the underlay so this concern disappears.

VTEP B receives, decapsulates, looks up VNI 50000 → bridge domain BD-50000, performs MAC learning on inner src MAC, forwards inner frame to the port where h4 lives.

### Example 3 — segment routing label stack derivation

SR-MPLS topology:

```
         ___16002___
        /          \
    R1 ---- R2 --- R3 ---- R4
    SID:    SID:   SID:    SID:
    16001   16002  16003   16004
```

SRGB = [16000, 23999] uniformly. Prefix SIDs:
- R1 → 16001
- R2 → 16002
- R3 → 16003
- R4 → 16004

Headend R1 wants to send to R4 via R3 (skip R2 path). Build the segment list:

```
Top of stack
+------------+
|  16003     |   "go via R3"
+------------+
|  16004     |   "destination R4"
+------------+
Bottom of stack
```

Wire (MPLS label stack, top first):

```
| 16003 | 16004 | <payload> |
```

Forwarding:

1. **R1 -> R2.** R2 receives. Top label = 16003 = R3's prefix SID. R2's own SRGB matches. R2 PHPs (penultimate hop pop on local label) or forwards via SWAP, using 16003 as the LFIB key. R2 sends to R3.
2. **R2 -> R3.** R3 receives. Top label = 16003 = R3's own prefix SID. R3 pops it. Now top label is 16004.
3. **R3 -> R4.** R3 looks up 16004. R4 is the prefix SID owner; R3 forwards toward R4. R4 pops; reads underlying payload header.

### Adjacency SIDs

Adjacency SIDs are local labels (per-router, dynamically allocated) that identify a specific link rather than a prefix. Use them to force a path through a specific link:

```
R1 has Adj-SID 24001 = "the link R1->R2"
R2 has Adj-SID 24002 = "the link R2->R3 via direct"
```

Stack to force R1 -> (R1-R2 link) -> (R2-R3 direct link) -> R4:

```
| 24001 | 24002 | 16004 |
```

Even if shorter paths exist, the segment list dictates the literal hops.

### Example 4 — Google B4 TE step-by-step

Simplified scenario: 5 sites (A, B, C, D, E), full-mesh fibre, $c_{ij} = 100$ Gbps each.

Demands:
- $d_1 = (A, D, 80\text{ Gbps}, \text{prio 1})$
- $d_2 = (B, D, 60\text{ Gbps}, \text{prio 1})$
- $d_3 = (A, E, 50\text{ Gbps}, \text{prio 2})$
- $d_4 = (C, E, 40\text{ Gbps}, \text{prio 2})$

Naive shortest path: $d_1$ via $A\to D$, $d_2$ via $B\to D$. Both fit ($\leq 100$). $d_3$ via $A\to E$ (50 Gbps, fits). $d_4$ via $C\to E$ (40 Gbps, fits).

Total served: 230 Gbps. Bottleneck utilization: $\max(80/100, 60/100, 50/100, 40/100) = 80\%$.

Now assume the link $A\to D$ goes down. $d_1$ must reroute. Naive options:

- $A \to B \to D$: shares with $d_2$ on $B\to D$. Demand $d_1 + d_2 = 80 + 60 = 140 > 100$. Doesn't fit.
- $A \to C \to D$: free path. $d_1$ uses $A \to C$ (80) and $C \to D$ (80). Both fit.
- $A \to E \to D$: shares with $d_3$ on $A\to E$. $d_3 + d_1 = 50 + 80 = 130 > 100$. Doesn't fit.

B4 controller picks $A \to C \to D$ for $d_1$. Updates label stacks at A, C, D.

If we now push $d_5 = (A, B, 30, \text{prio 3})$ — there is no path that doesn't congest somewhere; the LP will reduce $d_5$ to fit (or accept a small overload, depending on policy). B4 uses **bandwidth functions** with priority-aware fairness rather than strict LP feasibility, so $d_5$ may be partially served (15 Gbps) while $d_1$/$d_2$ keep their full demand.

Step-by-step:

```
1. Topology event: A-D link down.
2. Controller updates topology graph (delete edge A-D).
3. TE solver re-runs over impacted demands (d_1).
4. Solver picks A->C->D as the new path for d_1.
5. Controller computes new label stacks for d_1 entry switch (A) and changes
   on every switch on the new path.
6. Push order:
     a. Install new path's flow entries at exit (D) first, then C, then A
        (install backwards along the path so a packet entering A only sees
        the new flow once every downstream hop is ready).
     b. Atomic-or-not? Some controllers use a 2-phase commit; others rely
        on prefix-SID idempotence + brief packet loss during the swap.
7. Once new path is fully installed, the old path's entries at A are deleted.
8. Telemetry verifies link utilizations match the planned values.
```

Failure recovery time depends on the slowest hop's flow-mod ack and the controller's batching strategy. Production B4 numbers: hundreds of milliseconds typical, sub-second worst case.

---

## When SDN, when not

### Strong fits

- **Hyperscale data center fabric.** Hundreds-to-thousands of switches with uniform hardware, predictable workload, large DevOps team. SDN gives uniform programmability and tight integration with orchestration (Kubernetes CNI, OpenStack Neutron).
- **Service-provider WAN traffic engineering (B4 / SWAN-style).** Small site count, expensive long-haul capacity, valuable utilization gains.
- **Multi-tenant cloud.** Per-tenant overlays, programmatic policy, automated tenant onboarding. Effectively impossible to do at scale without an SDN-style controller.
- **Telco NFV.** Service chaining, dynamic VNF placement, per-customer service profiles.
- **Campus / branch (SD-WAN).** Centralized policy across hundreds-to-thousands of branches, application-aware path selection.

### Weak fits

- **Tiny office (1-10 switches, 1 admin).** The operational overhead of running a controller exceeds the savings. A few well-configured boxes with classic IOS/Junos / OPNsense are fine. Use vendor-managed cloud SDN (Meraki, Aruba Central) only if it shifts the operational burden cleanly.
- **Retail edge with low-skill operators.** SDN demands a culture of automation, telemetry, and on-call expertise the retail edge typically does not have.
- **Networks with heavy non-IP traffic.** SDN tooling assumes IP. Telephony TDM, fibre channel, industrial control buses, and similar workloads are often better served by purpose-built control planes (though FCoE and similar can ride SDN-aware fabrics).
- **Networks where vendor opinionation is acceptable.** If the vendor solution meets the requirements, the cost of building / operating an SDN stack rarely pays back. SDN is for organizations that need network behaviour the vendor cannot or will not deliver.

### The classical-vs-SDN decision matrix

| Driver | Classical | SDN |
|:---|:---:|:---:|
| Small site count, stable workload | yes | maybe |
| Hundreds of identical sites, frequent policy change | no | yes |
| Programmatic policy required | no | yes |
| Strong vendor OS already meets needs | yes | maybe |
| Centralized telemetry / closed-loop required | no | yes |
| Sub-50 ms WAN failover required everywhere | classical FRR | SDN + data-plane FRR |
| Multi-tenant overlays | hard | yes |
| Compliance demands strict per-flow audit | yes (on a single box) | yes (controller log) |
| Operator team < 5 people | yes | no |
| Operator team > 50 people | maybe | yes |

### Common failure modes when SDN is mis-applied

- **Reactive flow installation at scale.** Controller becomes the bottleneck; new flows time out. Symptom: TCP three-way handshake failures during traffic spikes.
- **Single-controller deployment in production.** Controller restart = network outage. Always run a cluster.
- **Flow-mod throughput exceeded.** Controller saturates the southbound TLS sessions; flow installation latency climbs from ms to seconds. Mitigation: bulk flow-mods, sharding, asynchronous SBI.
- **Underestimated southbound state.** A 10k-switch fabric with 1k flows per switch is 10M flows. Controllers need to be designed for that working set.
- **No data-plane FRR.** Controller failures pause new installs but should not break existing forwarding. Configure long flow timeouts and pre-installed backup paths.

---

## See Also

- `networking/bgp` — distributed control-plane comparison
- `networking/network-programmability` — automation and programmability ecosystem (NETCONF, gNMI, P4Runtime)
- `networking/vxlan` — overlay encapsulation used by most SDN fabrics
- `networking/segment-routing` — source-routed control plane that pairs naturally with SDN
- `networking/geneve` — extensible overlay encapsulation for SDN service chaining
- `networking/sd-wan` — vendor SDN for the branch WAN
- `networking/sd-access` — vendor SDN for the campus
- `networking/cisco-aci` — vendor SDN for the data center
- `networking/netconf` — config-style southbound protocol
- `networking/yang-models` — schema language used by NETCONF / gNMI / RESTCONF
- `ramp-up/sdn-eli5` — narrative ramp-up for the same topic
- `ramp-up/spine-leaf-eli5` — the dominant SDN data-center fabric topology

## References

- ONF, *OpenFlow Switch Specification, Version 1.3.5*, 2015. https://opennetworking.org/wp-content/uploads/2014/10/openflow-switch-v1.3.5.pdf
- ONF, *OpenFlow Switch Specification, Version 1.5.1*, 2015.
- McKeown, Anderson, Balakrishnan, Parulkar, Peterson, Rexford, Shenker, Turner, "OpenFlow: Enabling Innovation in Campus Networks," *ACM SIGCOMM Computer Communication Review*, April 2008.
- Casado, Freedman, Pettit, Luo, McKeown, Shenker, "Ethane: Taking Control of the Enterprise," *ACM SIGCOMM 2007*.
- Casado, *Architectural Support for Security Management in Enterprise Networks* (PhD Thesis, Stanford), 2007.
- Jain, Kumar, Mandal, Ong, Poutievski, Singh, Venkata, Wanderer, Zhou, Zhu, Zolla, Hölzle, Stuart, Vahdat, "B4: Experience with a Globally-Deployed Software Defined WAN," *ACM SIGCOMM 2013*.
- Hong, Kandula, Mahajan, Zhang, Gill, Nanduri, Wattenhofer, "Achieving High Utilization with Software-Driven WAN," *ACM SIGCOMM 2013*. (Microsoft SWAN)
- Bosshart, Daly, Gibb, Izzard, McKeown, Rexford, Schlesinger, Talayco, Vahdat, Varghese, Walker, "P4: Programming Protocol-Independent Packet Processors," *ACM SIGCOMM CCR*, July 2014.
- The P4 Language Consortium, *P4_16 Language Specification, v1.2.4*, 2023.
- RFC 7348, *Virtual eXtensible Local Area Network (VXLAN): A Framework for Overlaying Virtualized Layer 2 Networks over Layer 3 Networks*, August 2014.
- RFC 7637, *NVGRE: Network Virtualization Using Generic Routing Encapsulation*, September 2015.
- RFC 8300, *Network Service Header (NSH)*, January 2018.
- RFC 8402, *Segment Routing Architecture*, July 2018.
- RFC 8754, *IPv6 Segment Routing Header (SRH)*, March 2020.
- RFC 8926, *Geneve: Generic Network Virtualization Encapsulation*, November 2020.
- RFC 6241, *Network Configuration Protocol (NETCONF)*, June 2011.
- gNMI specification: https://github.com/openconfig/reference/blob/master/rpc/gnmi/gnmi-specification.md
- ONOS Project documentation: https://wiki.onosproject.org/
- OpenDaylight Project documentation: https://docs.opendaylight.org/
- Open Networking Foundation (ONF): https://opennetworking.org/
