# SDN — ELI5

> SDN is when one big brain at the airport tells every airplane where to fly, instead of every pilot guessing.

## Prerequisites

- `ramp-up/ip-eli5` — what an IP address is, what a packet is, what a router does.
- `ramp-up/bgp-eli5` — how routers talk to each other and decide which way to send a packet.

If you have not read those two sheets, go read them first. This sheet uses words like "packet," "router," "switch," and "BGP" without slowing down to redefine each one. Each word is also in the **Vocabulary** table at the bottom of this sheet. If a word feels weird, jump down there, read the one-line definition, and come back.

A reminder about how to read the code blocks: if you see a `$` at the start of a line, that means "type the rest of the line into your terminal." You do not type the `$` itself. Lines without a `$` are what your computer prints back at you. We call that "output."

## What Even Is SDN

### One sentence

SDN stands for **Software Defined Networking**. It is the idea that one program — running somewhere on a server — should be in charge of telling every router and every switch in your network what to do. The routers and switches stop being clever. They become dumb forwarders. The brain lives in the program.

### Air traffic control

Imagine an airport with a hundred airplanes. The old way: every pilot decides where to fly, when to take off, when to land, what altitude to climb to, which runway to use. The pilots talk to each other a little bit, but mostly they just guess and hope. Sometimes two airplanes try to use the same runway at the same time. Sometimes a plane lands at the wrong airport because the pilot misheard.

The new way: one big building in the middle of the airport, the **air traffic control tower**, has a person with a radio. That person can see every plane on a giant screen. That person tells each plane exactly when to take off, exactly which runway to use, exactly which altitude to climb to, and exactly when to land. The pilots still fly the planes — they still pull the stick — but they do not decide. They obey.

That is SDN. The control tower is the **controller**. The pilots are the **switches** and **routers**. The radio commands are the **southbound API**. The screen is the **network state database**.

### A second analogy: the orchestra

Imagine a hundred-piece orchestra without a conductor. Every player reads the same sheet music, but they all start at slightly different times. The violins are a beat ahead. The cellos are dragging. The percussion missed a count. It still kind of sounds like music if you squint, but it is sloppy.

Now put a conductor in front of the orchestra. The conductor does not play any instrument. The conductor's only job is to wave the baton at the right moments. Now the violins enter precisely on beat one. The cellos come in on beat three. The percussion lands on the downbeat. Same musicians, same instruments, same sheet music, but now it sounds like one piece of music instead of a hundred.

That is SDN again. The musicians are switches. The sheet music is the routing protocol. The conductor is the controller. Without the conductor, every device does its own thing and you get distributed (sometimes inconsistent) behavior. With the conductor, every device is coordinated.

### A third analogy: the pizza chain

Old-school networking is like every pizza shop in a chain inventing its own menu. Each shop guesses what people want. Each shop has different prices. Each shop calls a Margherita something different. The chain is technically a chain, but it is a chain only on the sign out front.

SDN is like one corporate office that owns the menu, the prices, the recipes, and the branding. Each shop just cooks. The shop does not invent. The shop does not decide. The shop receives instructions and executes. If corporate decides tomorrow that all shops should add a new pizza, every shop has it the same day.

That is what SDN does for networks. Corporate is the controller. The shops are the switches. The menu is the network policy.

### Why this is different

For thirty years, every router and every switch had its own brain. Every router ran its own copy of OSPF, its own copy of BGP, its own little piece of spanning-tree, its own ACL list, its own QoS policy. The vendor put a CPU inside every box, and every box thought hard about every packet.

This worked, but it was a mess. If you wanted to change one thing across a hundred routers, you had to log into each one. If two routers disagreed about the topology, you got a routing loop. If you wanted to do something clever, like "send all the video traffic to a different path during business hours," you had to write hairy scripts that poked at each router with SSH and hoped nothing broke.

SDN says: stop. Take the brain out of the routers. Put one big brain on a server. Let the server tell the routers what to do. The routers become forwarders. The server becomes the network.

## The Pre-SDN World

### Every router for itself

Picture a city where every traffic light has its own little brain. Each light has a tiny computer inside. The tiny computer looks at the cars in front of it, looks at the cars behind it, listens to the lights on the next block, and decides on its own when to turn green and when to turn red.

That is how networks worked before SDN. Every router was a little brain. Every router ran software like:

- **OSPF** — to learn the shape of the network from its neighbors.
- **BGP** — to talk to routers in other companies.
- **STP** — to stop loops on switches.
- **ACL** — to decide which packets to drop.
- **QoS** — to decide which packets are important.

When you bought a router from Cisco, you got a Cisco-shaped brain. When you bought a router from Juniper, you got a Juniper-shaped brain. Each brain spoke its vendor's CLI. Each brain disagreed about which way to send packets in subtle ways.

```
            +------+        +------+        +------+
            |Brain1|<------>|Brain2|<------>|Brain3|
            |+CLI  |  OSPF  |+CLI  |  OSPF  |+CLI  |
            +------+        +------+        +------+
              | |             | |             | |
              v v             v v             v v
            packets         packets         packets
```

Every box thinks. Every box has its own opinion. Every box runs the same protocols but with slightly different bugs.

### What broke about this

Three things broke about this world.

**Configuration drift.** You logged into Router A and made a change. You logged into Router B and made the same change. You forgot Router C. Now Router C disagrees with Router A and Router B. Traffic does weird things. Nobody knows why.

**Slow change.** You wanted to add a new VLAN to a hundred switches. You wrote a script. The script SSH'd into each switch and pasted in commands. The script broke halfway through because Switch 47 had a typo in its hostname. You spent two days fixing it.

**Stuck in vendor land.** You bought Cisco. Now everything has to be Cisco-flavored. You wanted to try a Juniper feature, but it interoperated badly with the Cisco BGP implementation. You were locked in.

### The 2008 paper

In 2008 a paper came out of Stanford called **OpenFlow: Enabling Innovation in Campus Networks**. The paper said: what if every switch had a standard, open protocol that let an outside controller program its forwarding table? Then anybody could write any controller they wanted. Any switch from any vendor would obey. The brain could go anywhere.

That paper started SDN. The protocol it described was **OpenFlow**.

### Why the paper hit so hard

Networking research had a problem. If you wanted to try a new routing protocol, you had to convince Cisco or Juniper to implement it in their proprietary firmware. They almost never did. Researchers built simulators or wrote papers without ever testing on real hardware. The whole field had calcified.

OpenFlow was a bargain: vendors agreed to implement a tiny standardized protocol that let outside controllers program flow tables. In exchange, researchers got to actually run their experiments on real switches in real campus networks. Stanford built a working campus deployment. Then Berkeley. Then Princeton. Then Google found out about it.

Google's adoption was the moment SDN went from research to real. Google rewrote its WAN — the network connecting its data centers, called **B4** — using OpenFlow-style centralized control. Google reported in 2013 that B4 ran at 95%+ utilization while traditional WANs ran at 30-40%. That paper, "B4: Experience with a Globally-Deployed Software Defined WAN," sold SDN to the industry. Every cloud, every telco, every big enterprise wanted what Google had.

### The pre-SDN-on-purpose world

There were earlier attempts to centralize control. **4D**, **Ethane**, **RCP**, **PCE-PCEP** — academic projects in the early 2000s all argued for separating control from data. **ForCES** (RFC 3654) was an IETF effort. **NETCONF** (RFC 4741, later 6241) standardized programmatic config. None of those broke through. OpenFlow did, partly because the timing was right (commodity x86 servers were finally cheap and fast enough to be controllers) and partly because the bargain with vendors was minimal.

The lesson here is that SDN as an idea predated OpenFlow by at least a decade. OpenFlow was just the moment the idea got cheap enough and standard enough to ship.

## The SDN Promise

### Three big promises

SDN as a movement made three promises.

**Promise 1: separation of concerns.** The control plane (the deciding) is separate from the data plane (the forwarding). The control plane lives on a controller. The data plane lives on the switches. They talk over a well-defined protocol. You can replace either side independently.

**Promise 2: open and programmable.** The controller exposes an API. You can write any program you want. You can write a program that says "during business hours, send video traffic over the fast path; after hours, send it over the cheap path." That program runs on the controller and the controller pushes flow rules to the switches.

**Promise 3: vendor neutrality.** Buy switches from anybody. As long as they speak the southbound protocol, your controller drives them. Cisco, Arista, Juniper, whitebox — all the same to you.

### Did the promises come true

Mostly. Promise 1 came true. Almost every modern data center separates control from data — sometimes via OpenFlow, sometimes via BGP-EVPN, sometimes via P4Runtime, sometimes via Cilium and eBPF. Promise 2 came true in pieces. The Kubernetes ecosystem in particular delivered a lot of programmable networking. Promise 3 came partly true. You can mix vendors, but every vendor adds proprietary extensions, and the controllers tend to be locked to one ecosystem.

The big lesson: pure OpenFlow did not take over the world. SDN as an idea did. The idea is everywhere now. We just stopped calling it "SDN" and started calling it whatever the specific implementation is.

### A scoreboard of the promises

Looking back from 2026, here is how each promise actually played out.

**Promise 1, separation of concerns.** Big win. Every modern fabric has a clear control / data split. Even traditional Cisco/Juniper devices now expose programmatic APIs (NETCONF, gNMI) that allow external controllers to drive them. The fight is over the **degree** of separation, not whether to separate.

**Promise 2, programmability.** Partial win. Public clouds and Kubernetes delivered on this hugely. Programmable Tofino-class hardware briefly delivered. Traditional enterprise switching delivered much less — vendors still gate "advanced" features behind licenses and proprietary CLI.

**Promise 3, vendor neutrality.** Mostly false advertising. Every controller talks "OpenFlow" but with vendor-specific extensions for hardware quirks. Every "open" YANG model has vendor-specific augmentations. You can mix vendors more easily than 2010, but you cannot truly swap vendors at zero cost.

A bonus promise that nobody said out loud at the start: **observability**. Modern SDN platforms expose telemetry that older networks could not dream of. Streaming gNMI, eBPF flow visibility, service-mesh trace headers, BGP-LS topology export. The ability to actually see what your network is doing is, in retrospect, the biggest practical SDN win.

## Forwarding Plane vs Control Plane

This is the most important idea in SDN. We will say it three different ways and then test it with shell commands.

### The two planes, in plain English

Every router has two jobs. Job one is **figuring out where packets should go**. Job two is **actually moving packets**. These are different jobs.

**Job one** is slow and thoughtful. It involves talking to neighbors, learning routes, computing shortest paths, applying policy. This is the **control plane**. It runs on the router's CPU. A modern router CPU is maybe an x86 chip or an ARM chip. Slow by network standards.

**Job two** is fast and dumb. It involves looking at a packet's destination address, looking up the address in a table, and shipping the packet out the right port. This is the **data plane**, also called the **forwarding plane**. It runs on the router's ASIC, which is a custom chip that can do this lookup billions of times per second.

```
   +----------------------+
   |   CONTROL PLANE      |   <- slow CPU, runs OSPF/BGP, builds tables
   | (the thinker)        |
   +----------+-----------+
              |
              | tables get pushed down
              v
   +----------------------+
   |   DATA / FWD PLANE   |   <- fast ASIC, looks up dest, ships packet
   | (the doer)           |
   +----------------------+
```

### The cooking analogy

The control plane is the chef who plans the menu. The chef thinks hard. The chef looks at the ingredients, decides what dishes go together, writes recipes. That takes a while.

The data plane is the line cooks. The line cooks do not plan menus. They look at the ticket, grab the right ingredients, throw them on the grill, plate the food. They do this very fast. They never stop to think about whether the menu is good.

Pre-SDN: every restaurant had its own chef and its own line cooks. SDN says: one big chef in a big building plans menus for every restaurant. The line cooks at each restaurant just follow the menu they got.

### The third plane nobody talks about

Sometimes you'll hear about the **management plane**. This is a third plane — separate from control and data — that handles things like SNMP polling, syslog, software upgrades, license management, and operator login. The management plane is even slower than the control plane. It is the part of the router you SSH into to type configuration commands and read counters.

A complete picture:

```
   +----------------------+
   |   MANAGEMENT PLANE   |   <- humans, scripts, monitoring
   |   (SSH, SNMP, syslog)|
   +----------+-----------+
              |
              v
   +----------------------+
   |   CONTROL PLANE      |   <- routing protocols, table compute
   |   (OSPF/BGP/etc)     |
   +----------+-----------+
              |
              v
   +----------------------+
   |   DATA PLANE         |   <- forwarding silicon, line rate
   |   (ASIC, NPU, kernel)|
   +----------------------+
```

Different SDN movements rearrange these. Classic OpenFlow lifts the control plane out of the box. NETCONF/gNMI-style "SDN" lifts the management plane out and standardizes it. Cilium with eBPF replaces the control plane on every node with a kernel-resident eBPF program plus a Kubernetes-resident operator. Service meshes lift management and control out of the network entirely and put them at the application layer.

### Test it on Linux

Linux itself does the control-plane / data-plane split. The data plane is in the kernel. The control plane is whatever userspace daemon is talking to it. Try this:

```
$ ip route show
default via 192.168.1.1 dev eth0 proto dhcp
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.42
```

That table is the data plane's forwarding table. The kernel looks at every packet, walks that table, and ships the packet. Who put the routes in that table? On a normal laptop, **dhclient** put the default route. On a server running BGP, **FRRouting** would put hundreds of routes in. The thing putting routes in is the control plane.

Now look at the kernel's view directly:

```
$ sudo ip -d route show
default via 192.168.1.1 dev eth0 proto dhcp metric 100
```

The `proto dhcp` tag tells you which control-plane program installed that route. That is SDN-style separation, just on a single host.

## The OpenFlow Protocol

### The original SDN protocol

OpenFlow was the first standardized **southbound API** — that is, the protocol the controller uses to push instructions down to the switches. Stanford published the first version in 2009. The Open Networking Foundation (ONF) took over the standardization later. Versions went 1.0, 1.1, 1.2, 1.3, 1.4, 1.5.

Most production OpenFlow deployments today use OpenFlow 1.3. It is the version most controllers and most switches actually implement.

### Match-action: the heart of OpenFlow

OpenFlow is built around a single idea: a **flow table**. A flow table is a list of entries. Each entry has two parts:

- **Match** — a description of which packets this entry applies to.
- **Action** — what to do with packets that match.

That is the whole model. Packets come in. The switch walks the flow table. The first matching entry wins. The actions in that entry get applied. The packet leaves.

```
+----------+-----------+-----------+--------+----------+
| priority |   match   |  actions  | counts | timeout  |
+----------+-----------+-----------+--------+----------+
|   100    | dst=10.1  | out:port3 |  4912  |   60s    |
|    90    | tcp.dst=22| drop      |  17    |   inf    |
|    50    | (any)     | controller|  0     |   inf    |
+----------+-----------+-----------+--------+----------+
```

Match fields can include source MAC, destination MAC, VLAN tag, source IP, destination IP, IP protocol, source TCP/UDP port, destination TCP/UDP port, and more. The action can be "send out port 3," "drop the packet," "rewrite a header field," "send the packet to the controller," "push a VLAN tag," "decrement TTL," and several others.

### The pipeline

OpenFlow 1.3 introduced a multi-table pipeline. A packet enters table 0. Table 0 may match it and apply actions and stop. Or table 0 may say "go to table 1." Table 1 does its match. And so on, up to table 255 or wherever the switch's limits are.

```
   pkt -> [Table 0] -> [Table 1] -> [Table 2] -> [Table 3] -> egress
            |             |             |             |
            v             v             v             v
         counters      counters      counters      counters
```

This is how a real switch implements complicated logic. Table 0 might do MAC learning. Table 1 might do VLAN classification. Table 2 might do ACLs. Table 3 might do destination lookup. Each table is small and focused. The pipeline glues them together.

### Packet-in / packet-out

The really interesting OpenFlow feature is **packet-in**. When a packet arrives that does not match any flow entry — or matches a "send to controller" action — the switch wraps the packet up and sends it to the controller. The controller looks at it, decides what to do, and either:

- Pushes a new flow entry down so future packets like this one are handled in hardware, or
- Sends the packet back as a **packet-out** with explicit instructions where to forward it once.

That is the cycle. New traffic hits the controller, the controller learns, the controller programs the switch, future traffic stays in the fast path.

### A picture of the match-action pipeline

```
   ingress packet
         |
         v
   +-----------+    miss    +------------+    miss   +------------+
   | Table 0   | ---------> | Table 1    | --------> | Table 2    | ---> drop
   | (L2 MAC)  |            | (L3 route) |           | (ACL)      |
   +-----+-----+            +------+-----+           +------+-----+
         | hit                     | hit                    | hit
         v                         v                        v
     actions:                  actions:                  actions:
     - learn src MAC           - decrement TTL           - permit/deny
     - lookup dst MAC          - rewrite next-hop MAC    - mark QoS
     - emit on port            - emit on port            - count
         |                         |                        |
         +-------------------------+------------------------+
                                   |
                                   v
                              egress packet
```

Each row in each table is a (priority, match, action, counter) tuple. Each table is a focused stage. The pipeline glues them. This is the same picture P4 generalized — P4 lets you describe **any** pipeline of any depth with any tables, instead of OpenFlow's fixed pipeline you only configure.

### What "stateless" means here

A flow table is technically stateless from the packet's point of view. The same packet hitting the same flow table in two switches will get the same actions. State lives in two places: the **table contents** (which the controller installs) and the **counters** (which the switch updates). Counters are read-only from the controller's perspective; the controller polls them, the controller does not write them.

This separation matters because it means flow tables are conceptually a function of (controller decisions) plus (current packet). They are not a function of "how many packets came before." For stateful behavior — like NAT, where you need to remember a flow's mapping — OpenFlow either pushes state into the controller (via packet-in) or punts to vendor extensions.

## OpenFlow Versions (1.0 → 1.5)

A quick tour of how OpenFlow evolved.

### 1.0 (December 2009)

The original. One flow table. Twelve match fields (source MAC, dest MAC, ethertype, VLAN ID, VLAN priority, source IP, dest IP, IP protocol, IP ToS, source TCP/UDP port, dest TCP/UDP port, ingress port). Simple actions: forward, drop, modify-field. Most early demos used this version.

### 1.1 (February 2011)

Added the multi-table pipeline. Added group tables for handling multipath and link aggregation. Almost nobody implemented 1.1 — it was a transitional release.

### 1.2 (December 2011)

Added IPv6 match fields. Added extensible match (TLV-style) so vendors could add new match types without breaking the protocol. Cleaned up the pipeline semantics.

### 1.3 (April 2012)

The version everyone actually uses. Added meters (rate limiting), per-flow counters, IPv6 extension headers, and tunnel-ID matching for VXLAN/MPLS. **If you read about an OpenFlow deployment in production, it is almost certainly 1.3.**

### 1.4 (October 2013)

Added optical port support, eviction of flow entries, vacancy events. Saw very limited adoption.

### 1.5 (December 2014)

Added egress tables (so the pipeline can have stuff happen on the way out, not just on the way in), packet-type-aware pipeline, and scheduled bundles. Saw almost no adoption.

After 1.5, OpenFlow effectively stopped evolving. The world moved on to P4, which lets you describe the pipeline itself rather than just configuring a fixed one.

## Controllers (ONOS, OpenDaylight, Floodlight, Ryu, Faucet)

The controller is the software that runs the network. There are many. Here are the famous ones.

### ONOS (Open Network Operating System)

Born at ON.Lab, now hosted by ONF. Java. Distributed by design — runs as a cluster of controllers that share state. Strong telco focus. Used in CORD (Central Office Re-architected as a Datacenter). Heavyweight, polished, hard to learn.

### OpenDaylight (ODL)

The Linux Foundation's controller. Java. Also distributed. Built around a YANG-driven data store. Sprawling — has subprojects for OpenFlow, NETCONF, BGP-LS, P4Runtime, gNMI, almost everything. If you work with any vendor's "SDN" platform, ODL is probably under the hood somewhere.

### Floodlight

Older OpenFlow controller from Big Switch Networks. Java. Single-instance. Simple. Good for learning. Largely superseded by ONOS for new work.

### Ryu

Python. Single-instance. Tiny. Trivial to write apps for — you write a Python class, override a few methods, and you have an OpenFlow application. Very common in research and teaching. Made by NTT.

### Faucet

Python, built on Ryu. Production-focused. Configuration-driven instead of programming-driven — you write a YAML file describing what you want, Faucet pushes flow rules. Used in real campus networks (e.g. universities, university hospitals). The "production OpenFlow" controller.

### Why so many

OpenFlow is just a wire protocol. The controller decides what to do with it. Different communities had different needs, so different controllers evolved. The Java ones (ONOS, ODL, Floodlight) tend to live in big telcos. The Python ones (Ryu, Faucet) tend to live in universities and small deployments.

## The CAP Theorem and SDN

### The theorem in 30 seconds

CAP says: in any distributed system, when the network partitions (some nodes can't talk to other nodes), you have to choose at most two of:

- **C**onsistency — every read sees the latest write.
- **A**vailability — every request gets a response.
- **P**artition tolerance — the system keeps working even when nodes can't reach each other.

Network partitions are not optional. Networks fail. So really CAP forces you to choose between **C** and **A** during a partition.

### Why this matters for SDN controllers

The promise of "one centralized brain" runs straight into CAP. If your network has one controller, that controller is a single point of failure. If your network has many controllers (a cluster), the controllers must agree about state. When the cluster partitions, the controllers must choose: refuse to make decisions (consistent, not available) or make decisions independently and hope they agree later (available, not consistent).

In practice, ONOS and ODL use Raft consensus to keep state consistent across cluster members. When a partition happens, only the partition with a majority can keep making decisions. The minority partition stops accepting new flow installs.

### The realistic answer

Most SDN deployments accept a small consistency hit for availability. They use eventual consistency where possible. Critical decisions (like "should this user get an IP address") are kept consistent. Bulk decisions (like flow stats reporting) are kept eventually consistent. Real production controllers have a lot of careful engineering around this. CAP is not theoretical here.

## Why Pure-OpenFlow Fell Short

Pure OpenFlow — meaning, every switch is dumb, one controller programs every flow — did not take over the world. Here is why.

### Reason 1: hardware mismatch

OpenFlow assumed every switch had a flexible TCAM that could match arbitrary fields. Real merchant silicon (the chips Broadcom and others sell to switch vendors) had fixed pipelines. You could not actually implement OpenFlow's full match/action model on the chip. Vendors implemented "OpenFlow profiles" — subsets of the protocol that mapped to their hardware. The promise of vendor-neutral programming evaporated.

### Reason 2: scale

Asking the controller about every new flow does not scale. If you have ten thousand new flows per second per switch, the controller has to handle a hundred thousand new flows per second per cluster of ten switches. That is hard. Solutions involved proactively installing flow rules so most traffic never hit the controller, but then you lost most of the dynamic-policy promise.

### Reason 3: the controller is also a single point of failure

If the controller goes down, the whole network goes down. Distributed controllers help, but they are hard to build. Telcos and cloud operators were not willing to bet their networks on early OpenFlow controllers.

### Reason 4: humans like distributed routing

The pre-SDN distributed routing protocols (OSPF, IS-IS, BGP) had been refined for thirty years. They worked. They self-healed. They scaled. Replacing them with a centralized controller that had to recompute paths for the entire network on every link failure was a tough sell.

### Reason 5: the SDN idea won, the SDN protocol did not

The big lesson: the **idea** of centralizing control and abstracting policy from hardware took over the world. The **specific protocol** OpenFlow did not. Modern SDN looks like:

- BGP-EVPN with a route reflector acting as the brain of a data-center fabric.
- Cilium with eBPF programming the Linux kernel forwarding plane on every node, controlled by Kubernetes.
- Service mesh sidecars enforcing policy with Envoy proxies, controlled by Istio.
- P4 programmable switches with P4Runtime as the southbound API.

All of those are SDN in spirit. None of them speak OpenFlow.

## Modern SDN Variants — Hybrid, BGP-as-SDN, P4, VXLAN+EVPN, Cilium/eBPF, Service Mesh

This section is the meat. Each of these is a real, in-production form of SDN.

### Hybrid SDN

The most common deployment. Routers still run distributed routing protocols (OSPF, BGP) for basic reachability. A controller layer sits on top to inject policy: traffic engineering, segment routing, ACLs, QoS. The controller talks to routers using NETCONF, RESTCONF, gNMI, or BGP-LS. The routers do not stop being smart; the controller layers on top.

Cisco DNA Center, Cisco APIC, Apstra, Arista CloudVision — all hybrid SDN.

### BGP as SDN

This is the cleanest, most boring, most successful form of SDN in modern data centers. The model: every leaf switch and every spine switch runs BGP. A small set of route reflectors aggregates the routing state. Operators inject policy via BGP communities and route maps. The "controller" is whatever automation tool generates the BGP config and pushes it via Ansible or a configuration management daemon.

The killer combination is **BGP-EVPN** — using MP-BGP with the EVPN address family to distribute MAC and IP reachability for a VXLAN overlay. Every modern data-center fabric uses some flavor of this. Cumulus pushed this hard. Cisco does it. Arista does it. Juniper does it. Whitebox vendors do it. **It is the dominant SDN in 2026.**

The "SDN" part is that you decoupled physical underlay from logical overlay, and you can move workloads around without renumbering. The protocol is BGP. The controller is your automation tool plus the route reflectors.

### P4

P4 is a programming language for the data plane. Instead of OpenFlow's "fixed pipeline you configure," P4 says "here is a language; describe the pipeline you want; we will compile it to whatever target chip you have." Targets include software (bmv2), FPGAs, and special programmable ASICs (Tofino, the Intel chip).

P4-16 is the language version (released 2016). **p4c** is the compiler. **P4Runtime** is the standardized control-plane API to load P4 programs and program their tables.

```
   P4 source
      |
      v
   p4c compiler
      |
      v
   target binary  ----+
                      |
   P4Runtime ---------+--->  switch
   (table writes,
    counter reads,
    config changes)
```

P4 is the most direct heir to the OpenFlow vision: full programmability of the data plane, with a clean control-plane API on top. It is mostly used by hyperscalers and research labs because programmable Tofino-class chips are not cheap. The Tofino program got cancelled by Intel in 2023, which threw the future of the hardware into doubt; software targets (bmv2) and other vendors carry on.

### VXLAN + EVPN

This pairs the most common overlay encapsulation (VXLAN, RFC 7348) with the most common control plane for it (EVPN, RFC 7432). The combination lets you build a multi-tenant overlay on top of a plain IP underlay.

VXLAN wraps Ethernet frames in UDP packets so they can travel across an IP network. Every switch that participates in the overlay is a **VTEP** (VXLAN Tunnel Endpoint). EVPN distributes which MACs and IPs live behind which VTEPs.

The "SDN" feeling is that you can spin up a new tenant network anywhere on the fabric, and the control plane (BGP-EVPN) figures out the plumbing automatically.

### Cilium / eBPF

Cilium is a Kubernetes networking plugin that programs Linux kernel data paths using **eBPF**. Every node in the cluster runs a Cilium agent. The agent compiles eBPF programs and loads them into the kernel. The eBPF programs implement service-to-service load balancing, network policy, NAT, encryption, observability — all at the kernel level, replacing iptables and the kube-proxy.

The "controller" in this model is the Kubernetes API server plus the Cilium operator. They distribute policy and identity. The "data plane" is every node's kernel running compiled eBPF. **Hubble** is the observability layer that taps the eBPF datapath for flow visibility.

Cilium 1.0 shipped in 2018; eBPF as a kernel feature became broadly stable around kernel 4.18 (2018). By 2024, Cilium became the default CNI in many large Kubernetes installs.

### Service mesh (Istio / Linkerd / Envoy)

A service mesh moves SDN even higher up the stack. Instead of programming switches, you program **sidecar proxies** running next to every application pod. The sidecar (usually Envoy) intercepts every connection the app makes. The control plane (Istio, Linkerd) configures the sidecars with routing rules, retry policies, circuit breakers, mTLS, and rate limits.

The sidecar is doing what a switch's flow table does, just at L7 (HTTP, gRPC) instead of L2/L3 (Ethernet, IP). Match: an HTTP request with `host: api.example.com` and path `/v2/foo`. Action: route to upstream cluster `api-v2`, retry on 5xx, add a trace header, enforce mTLS. It is the same match-action pattern, applied to application traffic.

```
                    +------------+
                    |  Istiod    |    <- control plane
                    |  (config)  |
                    +-----+------+
                          |
              +-----------+-----------+
              |           |           |
              v           v           v
   +----------+ +----------+ +----------+
   | sidecar  | | sidecar  | | sidecar  |   <- data plane (Envoy proxies)
   | + app A  | | + app B  | | + app C  |
   +----------+ +----------+ +----------+
        ^^           ^^           ^^
        ||  mTLS     ||  mTLS     ||  mTLS
        ++===========++===========++
```

```
   +----------+        +----------+
   |   App A  |        |   App B  |
   |  +-----+ |        | +-----+  |
   |  |sidec|<--mTLS-->|sidec|    |
   |  +-----+ |        | +-----+  |
   +----------+        +----------+
        ^                   ^
        |  policy           |
        +-----+   +---------+
              |   |
          +---+---+---+
          | control   |   <- Istio / Linkerd control plane
          | plane     |
          +-----------+
```

The sidecar is the data plane. The control plane configures it. That is SDN, just at the application layer instead of the packet layer.

## Intent-Based Networking

### What it means

Intent-Based Networking (IBN) is the next layer above plain SDN. Instead of telling the controller "install this flow rule on this switch," you tell it "Service A should be able to reach Service B with mTLS, and nothing else should be able to reach Service A." The system figures out the rules.

You declare intent. The system computes the configuration. The system pushes it down. The system continuously verifies that the network state matches the intent. If reality drifts, the system either alerts or self-heals.

### Real implementations

- **Cisco DNA Center** — campus and access intent.
- **Cisco APIC** — data-center intent for ACI fabrics.
- **Juniper Apstra** — data-center intent across vendors.
- **Arista CloudVision** — telemetry plus intent.
- **Kubernetes NetworkPolicy + Cilium** — intent at the application layer.

### How it relates to SDN

IBN is SDN with a higher-level API. The southbound is still NETCONF, gNMI, or whatever. The northbound is now declarative: YAML, JSON, sometimes a UI. The controller translates intent into device-level config.

### A small example of intent

A pre-SDN network engineer would write this to allow a web server to reach a database server:

```
ip access-list extended ALLOW_DB
 permit tcp host 10.1.1.10 host 10.2.2.20 eq 5432
 deny ip any host 10.2.2.20
interface GigabitEthernet0/1
 ip access-group ALLOW_DB in
```

A Kubernetes-NetworkPolicy intent would write the same idea this way:

```
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-web-to-db
spec:
  podSelector:
    matchLabels: { tier: db }
  ingress:
  - from:
    - podSelector:
        matchLabels: { tier: web }
    ports:
    - protocol: TCP
      port: 5432
```

The pre-SDN version names IP addresses and interfaces. If the database moves, you rewrite the rule. If a second web server appears, you rewrite the rule. If the network is renumbered, you rewrite a hundred rules.

The intent version names labels. The controller translates labels to IPs continuously. If the database moves, the controller updates the rule. If a second web server appears, the controller updates the rule. The human author wrote the intent once.

That is the IBN promise: write intent, let the system worry about plumbing.

## Network Disaggregation (whitebox + ONIE + SONiC/Cumulus/DENT)

### The disaggregation idea

Old networking: you bought a switch from Cisco. The switch came with Cisco's IOS. Cisco's IOS only ran on Cisco's hardware. You paid Cisco for both.

Disaggregation: the switch hardware (a "whitebox") and the switch operating system are separate products. You buy the hardware from one vendor and the OS from another. The hardware is built around merchant silicon (usually Broadcom). The OS is whatever you choose.

This is critical to SDN because it makes the data plane a commodity. The brain (the controller, the OS) is where the value lives.

### ONIE (Open Network Install Environment)

ONIE is a small Linux that ships in the boot ROM of a whitebox switch. When you power on the box, ONIE looks for a network OS image on the network (DHCP + TFTP or HTTP). It downloads the OS. It installs it. Reboot. The switch boots into your OS of choice. This is the network equivalent of PXE booting a server.

### SONiC

**SONiC** (Software for Open Networking in the Cloud) is the network OS Microsoft open-sourced in 2017. Linux-based. Containerized control-plane (every protocol — BGP, LLDP, SNMP, telemetry — runs in its own container). Uses **SAI** (Switch Abstraction Interface) to talk to whatever underlying ASIC (Broadcom, Mellanox, Marvell, Innovium). Runs in production at hyperscalers. Can run on dozens of whitebox SKUs.

### Cumulus Linux

Started as a startup, acquired by NVIDIA in 2020. Debian-based. Made BGP-EVPN-on-whitebox practical for everyone. Uses FRR for routing, Linux for forwarding-table abstraction. Loved by network engineers because you can SSH in and use normal Linux tools.

### DENT

A lightweight network OS for the network edge — small offices, retail, light-industrial. Built on standard Linux kernel forwarding (no separate forwarding daemon), uses switchdev to push forwarding state into the kernel, which pushes it to the ASIC. Linux Foundation project. Smaller scope than SONiC but much simpler.

## Common Errors

Verbatim error messages you will hit, and what they mean.

```
ovs-vsctl: unix:/usr/local/var/run/openvswitch/db.sock: database connection failed (No such file or directory)
```

The Open vSwitch database daemon (`ovsdb-server`) isn't running. Start it: `sudo systemctl start openvswitch-switch` or run `ovs-ctl start`.

```
ovs-ofctl: no support for protocol 'OpenFlow15'
```

You asked `ovs-ofctl` to use OpenFlow 1.5, but this build of OVS only supports up to 1.3 or 1.4. Pick a different protocol with `-O OpenFlow13`.

```
*** Error setting resource limits. Mininet's performance may be affected.
```

Mininet warning when ulimit-style settings can't be applied. Run mininet with sudo or as root, or accept the default limits.

```
2024-04-12 09:14:01,234 - ryu.controller.controller - ERROR - Address already in use
```

A Ryu app tried to bind to port 6633 or 6653 but another OpenFlow controller is already running. Kill the other one or pass `--ofp-tcp-listen-port 6634`.

```
faucet ERROR Config file /etc/faucet/faucet.yaml is invalid: missing required key 'dps'
```

Faucet's YAML must define `dps` (datapaths). Add a top-level `dps:` block listing each switch.

```
hubble: failed to dial: connection refused
```

`hubble` can't reach the Hubble relay. Make sure `hubble-relay` is running in the cluster: `kubectl -n kube-system get pods -l k8s-app=hubble-relay`.

```
cilium status timed out: context deadline exceeded
```

The Cilium agent on this node isn't responding. `kubectl -n kube-system logs ds/cilium` to see why.

```
gnmic: rpc error: code = Unavailable desc = connection error: desc = "transport: authentication handshake failed: x509: certificate signed by unknown authority"
```

You're using TLS for gNMI but don't trust the device's cert. Either provide a CA bundle (`--tls-ca`) or use `--skip-verify` for testing.

```
gobgp: Error: rpc error: code = Unavailable desc = connection error: dial tcp [::1]:50051: connect: connection refused
```

`gobgp` (the CLI) tried to reach `gobgpd` (the daemon) on localhost:50051 and the daemon isn't running. Start `gobgpd -f /etc/gobgp.conf`.

```
p4c: error: Could not find file: switch.p4
```

`p4c` (the P4 compiler) couldn't open the source. Use an absolute path or pass `-I` to add include directories.

```
bmv2 error: Cannot find action 'forward' in table 'ipv4_lpm'
```

The control-plane code referenced an action that isn't defined for the table in your P4 program. Check the P4 source's table definition.

```
vtysh: cannot connect to bgpd
```

The FRR `bgpd` daemon isn't running. `sudo systemctl start frr` or check `/etc/frr/daemons` to make sure `bgpd=yes`.

```
NETCONF: error: rpc-error: bad-element xml-tag-not-recognized "configure"
```

You sent an `<edit-config>` payload using the wrong YANG namespace. The device expects its vendor-specific namespace (e.g. Juniper's `junos-conf-root`).

```
openflow: BAD_REQUEST OFPBRC_BAD_VERSION
```

The controller's OpenFlow version doesn't match the switch's. Both sides must agree (1.0, 1.3, 1.4, etc.). Check controller config.

```
kubectl get pods -n kube-system -l k8s-app=cilium
NAME           READY   STATUS             RESTARTS   AGE
cilium-abc12   0/1     CrashLoopBackOff   7          20m
```

Cilium pod can't start. `kubectl describe pod cilium-abc12 -n kube-system` and check kernel version (eBPF requires 4.9+, ideally 4.19+) and kernel config.

```
istioctl analyze
Error [IST0101] (VirtualService default/foo) Referenced host not found: "bar.default.svc.cluster.local"
```

Your Istio `VirtualService` points at a host that doesn't exist as a `Service`. Either create the service or fix the host string.

## Hands-On

You will need a Linux machine. A laptop running Ubuntu, a VM, or a cloud instance all work. Most of these examples assume Ubuntu 22.04. Substitutions for other distros are minor.

### 1. Install Open vSwitch

```
$ sudo apt install -y openvswitch-switch
```

```
$ sudo ovs-vsctl --version
ovs-vsctl (Open vSwitch) 3.0.3
DB Schema 8.3.0
```

If the version line prints, OVS is installed and the daemon is running.

### 2. Create your first virtual bridge

```
$ sudo ovs-vsctl add-br br0
```

Now confirm:

```
$ sudo ovs-vsctl show
b29f8a1f-7a18-4e1d-9c7c-9cb4f6e3a3aa
    Bridge br0
        Port br0
            Interface br0
                type: internal
    ovs_version: "3.0.3"
```

You just made a virtual switch. It exists only in software. You can now plug interfaces into it.

### 3. Add ports

```
$ sudo ovs-vsctl add-port br0 veth0
$ sudo ovs-vsctl add-port br0 veth1
$ sudo ovs-vsctl list-ports br0
veth0
veth1
```

(You'll need to create the veth pairs first with `ip link add`. We'll do that in step 5.)

### 4. Set OpenFlow version

```
$ sudo ovs-vsctl set bridge br0 protocols=OpenFlow13
$ sudo ovs-vsctl get bridge br0 protocols
[OpenFlow13]
```

The bridge will now talk OpenFlow 1.3 to any controller.

### 5. Install a flow rule by hand

```
$ sudo ovs-ofctl -O OpenFlow13 add-flow br0 "priority=100,in_port=1,actions=output:2"
$ sudo ovs-ofctl -O OpenFlow13 dump-flows br0
 cookie=0x0, duration=4.821s, table=0, n_packets=0, n_bytes=0, priority=100,in_port=1 actions=output:2
```

You just programmed the flow table. Anything that arrives on port 1 will go out port 2.

### 6. Watch flows update in real time

```
$ watch -n 1 'sudo ovs-ofctl -O OpenFlow13 dump-flows br0'
```

Open another terminal and run traffic. The `n_packets` and `n_bytes` counters increment.

### 7. Install Mininet

```
$ sudo apt install -y mininet
```

```
$ sudo mn --version
2.3.0
```

Mininet creates whole networks of virtual hosts and switches in your laptop. Perfect for SDN experimentation.

### 8. Start a tiny Mininet topology

```
$ sudo mn --topo single,3 --controller none
*** Creating network
*** Adding controller
*** Adding hosts:
h1 h2 h3
*** Adding switches:
s1
*** Adding links:
(h1, s1) (h2, s1) (h3, s1)
*** Configuring hosts
h1 h2 h3
*** Starting controller
*** Starting 1 switches
s1 ...
*** Starting CLI:
mininet>
```

Try `pingall`:

```
mininet> pingall
*** Ping: testing ping reachability
h1 -> X X
h2 -> X X
h3 -> X X
*** Results: 100% dropped (0/6 received)
```

Pings fail because there is no controller and no flow rules. The switch drops everything. SDN at work — without a brain, the data plane does nothing.

### 9. Add a learning-switch controller manually

Quit Mininet. Then start the built-in OVS learning controller:

```
$ sudo mn --topo single,3 --controller ovsc
mininet> pingall
*** Ping: testing ping reachability
h1 -> h2 h3
h2 -> h1 h3
h3 -> h1 h2
*** Results: 0% dropped (6/6 received)
```

Now everything works. The OVS controller program installed flow rules to make the switch act like a normal learning switch.

### 10. Install Ryu

```
$ pip install ryu
$ ryu-manager --version
ryu-manager 4.34
```

### 11. Run the bundled simple-switch app

```
$ ryu-manager ryu.app.simple_switch_13
loading app ryu.app.simple_switch_13
loading app ryu.controller.ofp_handler
instantiating app ryu.app.simple_switch_13 of SimpleSwitch13
instantiating app ryu.controller.ofp_handler of OFPHandler
```

Leave that running. In another terminal:

```
$ sudo mn --topo single,3 --controller remote,ip=127.0.0.1,port=6653
mininet> pingall
*** Results: 0% dropped (6/6 received)
```

Your Ryu controller just programmed the switches.

### 12. Install Faucet

```
$ pip install faucet
$ which faucet
/usr/local/bin/faucet
```

Faucet wants a config file. Minimal example saved as `/etc/faucet/faucet.yaml`:

```
dps:
  s1:
    dp_id: 1
    interfaces:
      1:
        native_vlan: 100
      2:
        native_vlan: 100
      3:
        native_vlan: 100
vlans:
  100:
    description: "lab"
```

Then:

```
$ faucet --config-file=/etc/faucet/faucet.yaml
INFO     faucet.config_parser_util  config /etc/faucet/faucet.yaml is loaded.
INFO     faucet                     Beginning faucet event loop
```

Point Mininet at it (`--controller remote,ip=127.0.0.1,port=6653`) and your switches will be VLAN-aware.

### 13. Install minikube + Cilium

```
$ minikube start --cni=cilium
$ kubectl -n kube-system get ds/cilium
NAME     DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   AGE
cilium   1         1         1       1            1           2m
```

### 14. Look at Cilium endpoints

```
$ kubectl -n kube-system exec ds/cilium -- cilium endpoint list
ENDPOINT   POLICY (ingress)   POLICY (egress)   IDENTITY   LABELS                       IPV4         STATUS
345        Disabled           Disabled          1          reserved:host                              ready
1112       Disabled           Disabled          4          reserved:health              10.0.0.62    ready
```

Each running pod becomes a Cilium endpoint identified by labels. eBPF programs enforce policy on those identities.

### 15. Apply a NetworkPolicy

```
$ kubectl apply -f - <<'YAML'
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: deny-all
  namespace: default
spec:
  podSelector: {}
  policyTypes: [Ingress, Egress]
YAML
networkpolicy.networking.k8s.io/deny-all created
```

Now no pod in default can talk to any other pod. Cilium has compiled this into eBPF programs running on every node.

### 16. Watch packets with hubble

```
$ kubectl -n kube-system exec ds/cilium -- hubble observe --follow
TIMESTAMP             SOURCE              DESTINATION             TYPE    VERDICT  SUMMARY
Apr 12 09:18:42.331   default/curl-1234   default/web-7654:80     L3/L4   DROPPED  Policy denied
```

You can see the policy enforcement in real time. This is the "observability" promise of eBPF SDN.

### 17. Install gNMIc

```
$ bash -c "$(curl -sL https://get-gnmic.openconfig.net)"
$ gnmic version
version : 0.31.5
```

### 18. Subscribe to a device's interface counters

Assuming a router at 192.0.2.1 with gNMI enabled:

```
$ gnmic -a 192.0.2.1:57400 -u admin -p admin --skip-verify \
        subscribe --path "/interfaces/interface/state/counters" \
        --mode stream --stream-mode sample --sample-interval 5s
{
  "source": "192.0.2.1:57400",
  "subscription-name": "default-1714157912",
  "timestamp": 1714157915123456789,
  "time": "2024-04-26T20:18:35.123456789Z",
  "updates": [
    {
      "Path": "interfaces/interface[name=ge-0/0/0]/state/counters/in-octets",
      "values": { "interfaces/...": 17234587234 }
    }
  ]
}
```

This is streaming telemetry — much more efficient than SNMP polling.

### 19. Install FRR + use vtysh

```
$ sudo apt install -y frr
$ sudo systemctl enable --now frr
```

Then drop into the Cisco-flavored CLI:

```
$ sudo vtysh
Hello, this is FRRouting (version 8.4.4).
Copyright 1996-2005 Kunihiro Ishiguro, et al.
router1#
```

Configure BGP:

```
router1# configure terminal
router1(config)# router bgp 65001
router1(config-router)# bgp router-id 192.0.2.1
router1(config-router)# neighbor 192.0.2.2 remote-as 65002
router1(config-router)# end
router1# show bgp summary
IPv4 Unicast Summary:
BGP router identifier 192.0.2.1, local AS number 65001 vrf-id 0
Neighbor        AS    MsgRcvd MsgSent State/PfxRcd
192.0.2.2       65002 12      11      0
```

### 20. Install gobgp

```
$ go install github.com/osrg/gobgp/v3/cmd/gobgp@latest
$ go install github.com/osrg/gobgp/v3/cmd/gobgpd@latest
```

Start the daemon with a tiny config:

```
$ cat > gobgp.toml <<'TOML'
[global.config]
  as = 65010
  router-id = "10.0.0.1"
TOML
$ gobgpd -f gobgp.toml
```

Then in another shell:

```
$ gobgp neighbor
Peer    AS    Up/Down State       |#Received  Accepted
```

### 21. Install p4c (the P4 compiler)

```
$ sudo apt install -y p4lang-p4c
$ p4c --version
p4c 1.2.4.10
```

### 22. Compile a tiny P4 program

Save as `hello.p4`:

```
#include <core.p4>
#include <v1model.p4>

header ethernet_t {
    bit<48> dst;
    bit<48> src;
    bit<16> etype;
}
struct headers { ethernet_t eth; }
struct metadata {}

parser MyParser(packet_in p, out headers h, inout metadata m, inout standard_metadata_t s) {
    state start { p.extract(h.eth); transition accept; }
}
control MyVerify(inout headers h, inout metadata m) { apply {} }
control MyIngress(inout headers h, inout metadata m, inout standard_metadata_t s) {
    apply { s.egress_spec = 1; }
}
control MyEgress(inout headers h, inout metadata m, inout standard_metadata_t s) { apply {} }
control MyCompute(inout headers h, inout metadata m) { apply {} }
control MyDeparser(packet_out p, in headers h) { apply { p.emit(h.eth); } }

V1Switch(MyParser(), MyVerify(), MyIngress(), MyEgress(), MyCompute(), MyDeparser()) main;
```

Compile:

```
$ p4c -b bmv2 -o build hello.p4
$ ls build
hello.json  hello.p4i
```

You just compiled a (very dumb) data-plane program. With bmv2 you could now load it and feed it packets.

### 23. Load it into bmv2

```
$ simple_switch --thrift-port 9090 build/hello.json -i 0@veth0 -i 1@veth1 &
$ simple_switch_CLI --thrift-port 9090
RuntimeCmd: tables
mytable
```

### 24. Inspect Linux's own forwarding tables

```
$ ip -d link show type bridge
4: br0: <BROADCAST,MULTICAST,UP> mtu 1500 qdisc noqueue
    link/ether 02:42:cc:11:22:33 brd ff:ff:ff:ff:ff:ff
    bridge forward_delay 1500 hello_time 200 ...
```

```
$ bridge fdb show br br0
33:33:00:00:00:01 dev veth0 self permanent
01:80:c2:00:00:00 dev veth0 self permanent
```

That's the kernel's MAC forwarding table — the data plane on Linux when used as a software switch.

### 25. Install Istio (optional, takes a minute)

```
$ curl -L https://istio.io/downloadIstio | sh -
$ ./istio-1.21.0/bin/istioctl install --set profile=demo -y
```

```
$ kubectl -n istio-system get pods
NAME                                    READY   STATUS    RESTARTS   AGE
istiod-7c4b8c5d4-abc12                  1/1     Running   0          80s
istio-ingressgateway-89f6f7c5d-def34    1/1     Running   0          80s
```

### 26. Inject a sidecar

```
$ kubectl label namespace default istio-injection=enabled
$ kubectl run nginx --image=nginx
$ kubectl get pods nginx
NAME    READY   STATUS    RESTARTS   AGE
nginx   2/2     Running   0          12s
```

`2/2` — the app and the Envoy sidecar. The sidecar is the data plane. Istiod is the control plane.

### 27. Watch sidecar config from the control plane

```
$ istioctl proxy-config listeners nginx -o json | head
[
  {
    "name": "0.0.0.0_15006",
    "address": {
      "socketAddress": {
        "address": "0.0.0.0",
        "portValue": 15006
      }
    },
    ...
```

That JSON is the Envoy config Istiod pushed down. Every pod's sidecar got something like it.

### 28. Try a flow-trace through OVS

```
$ sudo ovs-appctl ofproto/trace br0 in_port=1,dl_type=0x0800,nw_dst=10.0.0.5
Flow: ip,in_port=1,nw_src=0.0.0.0,nw_dst=10.0.0.5,nw_proto=0,nw_tos=0,nw_ttl=0
bridge("br0")
-------------
 0. priority 100, in_port=1, actions=output:2
Final flow: unchanged
Megaflow: ...
Datapath actions: 2
```

This is how you debug an OpenFlow forwarding decision.

### 29. List Cilium policies in detail

```
$ kubectl exec -n kube-system ds/cilium -- cilium policy get
[
  {
    "name": "deny-all",
    "namespace": "default",
    "selector": {
      "match-labels": {
        "any:io.kubernetes.pod.namespace": "default"
      }
    },
    "ingress": [],
    "egress": [],
    ...
  }
]
```

### 30. Capture XDP traces

```
$ sudo bpftool prog list
4: cgroup_skb  tag 6deef7357e7b4530  gpl
5: xdp  name xdp_cilium_bpf  tag a45f...
```

```
$ sudo bpftool prog tracelog
```

You can attach to the running eBPF program and see live tracepoints.

### 31. Use OpenConfig YANG models with NETCONF

```
$ ssh -s netconf admin@192.0.2.1
<?xml version="1.0" encoding="UTF-8"?>
<hello xmlns="urn:ietf:params:xml:ns:netconf:base:1.0">
  <capabilities>
    <capability>urn:ietf:params:netconf:base:1.1</capability>
    <capability>http://openconfig.net/yang/interfaces?module=openconfig-interfaces</capability>
  </capabilities>
</hello>
]]>]]>
```

The `]]>]]>` is the NETCONF framing.

### 32. Push config via RESTCONF

```
$ curl -k -u admin:admin \
    -H 'Content-Type: application/yang-data+json' \
    -X PATCH \
    -d '{"openconfig-interfaces:interfaces":{"interface":[{"name":"ge-0/0/0","config":{"description":"new label"}}]}}' \
    https://192.0.2.1/restconf/data/openconfig-interfaces:interfaces
```

If the device returns 204 No Content, the change took.

## Common Confusions

Pairs of things that beginners confuse. Each pair has the difference in one or two sentences.

### SDN vs NFV

**SDN** moves the network control plane to software running on a server. **NFV** moves whole network functions (firewalls, load balancers, routers) off custom appliances and onto generic servers. They're complementary; you often use both together.

### Control plane vs data plane

The **control plane** decides where packets should go. The **data plane** moves the packets. Different jobs, different speeds, different hardware.

### Northbound vs southbound

The **southbound API** is how the controller talks to the network devices (OpenFlow, NETCONF, gNMI, P4Runtime). The **northbound API** is how applications talk to the controller (REST, GraphQL, gRPC).

### OpenFlow vs OVS

**OpenFlow** is a protocol. **OVS** (Open vSwitch) is one implementation of an OpenFlow-capable switch (and also a non-OpenFlow regular switch). OpenFlow is to OVS as TCP is to Linux's TCP stack.

### OVS vs Linux bridge

Both are Linux software switches. The **Linux bridge** is built into the kernel and is dumb (basic learning, basic VLAN). **OVS** is much more capable — OpenFlow, GRE, VXLAN, sophisticated flow tables. OVS is what Kubernetes/Openstack used for years until eBPF/Cilium showed up.

### Ryu vs Faucet

**Ryu** is an OpenFlow framework you write Python apps against. **Faucet** is a YAML-configured production controller built on top of Ryu. Ryu is "build your own controller." Faucet is "use the one that already works."

### ONOS vs OpenDaylight

Both are Java OpenFlow controllers with distributed clustering. **ONOS** has stricter latency goals (carrier focus). **OpenDaylight** is a broader platform with many subprojects. Most folks pick based on community fit, not feature lists.

### Mininet vs containerlab

**Mininet** uses Linux network namespaces to emulate a topology with hosts and OVS switches. **containerlab** uses Docker containers, often running real network OSes (FRR, Nokia SR Linux, Arista cEOS). containerlab is closer to real hardware behavior.

### VXLAN vs Geneve

Both are overlay encapsulations carrying Ethernet inside UDP. **VXLAN** has a fixed 8-byte header. **Geneve** has a variable-length header so you can add metadata (e.g. for service-function chaining). Geneve is newer but VXLAN is everywhere.

### EVPN vs OSPF

**OSPF** is an underlay routing protocol — it tells routers about IP reachability inside one autonomous system. **EVPN** is an overlay control plane — it distributes MAC and IP addresses for tenants riding on top of an underlay. Different jobs, can be used together.

### eBPF vs iptables

Both filter and route packets in the Linux kernel. **iptables** uses a fixed chain-of-rules model. **eBPF** lets you write small programs that the kernel JIT-compiles to native code at attach time. eBPF is faster, more flexible, and harder to use.

### XDP vs tc-bpf

Both load eBPF programs to handle packets. **XDP** runs at the lowest level — before the packet enters the network stack — for max speed. **tc-bpf** runs in the traffic-control layer, after the packet has been classified. XDP is faster but more limited.

### kube-proxy vs Cilium

**kube-proxy** uses iptables (or IPVS) to implement Kubernetes Service load balancing. **Cilium** replaces kube-proxy with eBPF programs that do the same job in the kernel datapath. Cilium is faster and more observable.

### Sidecar vs ambient mesh

**Sidecar** mode (Istio classic, Linkerd) runs an Envoy/proxy alongside every pod. **Ambient mesh** (Istio ambient, Cilium service mesh) puts the proxy at the node level so apps don't carry the per-pod cost.

### Intent vs config

**Configuration** says "set BGP local-as to 65001 on Router-A." **Intent** says "Service A should reach Service B with mTLS." The IBN system translates intent into config across many devices.

### YANG vs JSON Schema

Both describe data structures. **YANG** is the IETF/network industry's modeling language with a long history of describing router config. **JSON Schema** is the JSON world's modeling language. Network controllers usually expose YANG to the south and translate to JSON or REST for the north.

### gNMI vs NETCONF

Both are network management protocols. **NETCONF** is older, XML-based, RPC over SSH or TLS. **gNMI** is newer, gRPC/protobuf-based, optimized for streaming telemetry. New deployments prefer gNMI.

### SDN vs SD-WAN

**SDN** is general — any centralized-control network. **SD-WAN** is a specific product category that uses SDN ideas to manage WAN connectivity (MPLS, broadband, LTE) for branch offices. SD-WAN is one application of SDN.

### Whitebox vs brite-box

**Whitebox** is bare hardware from an ODM (e.g. Edgecore, Celestica) that you put any OS on. **Brite-box** is the same hardware sold by a brand-name vendor (e.g. Arista) with their OS on it. Sometimes also called "branded whitebox."

## Vocabulary

| Term | Plain English |
|------|---------------|
| **SDN** | Software Defined Networking. One brain on a server tells the routers/switches what to do. |
| **NFV** | Network Function Virtualization. Run firewalls/routers/etc as software on regular servers. |
| **control plane** | The deciding part. Slow, thoughtful, runs protocols, builds tables. |
| **data plane** | The doing part. Fast, dumb, moves packets through tables. |
| **forwarding plane** | Another name for data plane. |
| **southbound** | Direction from controller down to devices (OpenFlow, NETCONF, gNMI). |
| **northbound** | Direction from controller up to applications (REST, GraphQL). |
| **OpenFlow** | The original SDN southbound protocol. Match/action flow tables. |
| **OF1.0** | OpenFlow 1.0, 2009, single flow table. |
| **OF1.3** | OpenFlow 1.3, 2012, multi-table pipeline, the version most folks actually use. |
| **OF1.5** | OpenFlow 1.5, 2014, egress tables; little adoption. |
| **flow table** | A list of (match, action) entries inside a switch. |
| **flow entry** | One row in a flow table. |
| **match** | The criteria a flow entry uses to grab packets (port, MAC, IP, etc). |
| **action** | What the flow entry does to matched packets (forward, drop, rewrite). |
| **packet-in** | Switch sending a mystery packet to the controller for instructions. |
| **packet-out** | Controller injecting a packet for the switch to forward. |
| **OVS** | Open vSwitch. The most common software OpenFlow switch. |
| **ovs-vsctl** | CLI to configure OVS bridges and ports. |
| **ovs-ofctl** | CLI to manipulate OVS flow tables over OpenFlow. |
| **Mininet** | Linux-namespace network emulator. Hosts + OVS in software. |
| **containerlab** | Docker-container network emulator. Runs real network OSes. |
| **ONOS** | Java distributed SDN controller, telco-flavored. |
| **OpenDaylight** | Java SDN platform, broad. AKA ODL. |
| **Floodlight** | Older Java OpenFlow controller. |
| **Ryu** | Tiny Python OpenFlow controller framework. |
| **Faucet** | Production YAML-driven OpenFlow controller built on Ryu. |
| **BGP-LS** | BGP Link-State. Carries IGP topology info into BGP. RFC 7752 (2016). |
| **BGP-FlowSpec** | BGP extension to carry per-flow filtering rules. RFC 5575. |
| **MP-BGP** | Multi-Protocol BGP. The BGP that carries any address family. |
| **EVPN** | Ethernet VPN. BGP control plane for L2/L3 overlays. RFC 7432. |
| **VXLAN** | Encapsulation: Ethernet frame inside UDP, 24-bit VNI. RFC 7348. |
| **Geneve** | Newer overlay encapsulation with extensible header. RFC 8926. |
| **VTEP** | VXLAN Tunnel Endpoint. The thing that wraps and unwraps VXLAN. |
| **NVE** | Network Virtualization Edge. Generic name for VTEP-class function. |
| **RD** | Route Distinguisher. Makes per-VRF routes globally unique in BGP. |
| **RT** | Route Target. BGP community that controls VRF import/export. |
| **ESI** | Ethernet Segment Identifier. EVPN's way of naming a multi-homed link. |
| **MC-LAG** | Multi-Chassis Link Aggregation. Two switches looking like one. |
| **leaf-spine** | Modern data-center fabric: leaves talk to spines, spines never talk to spines. |
| **P4** | Programming language for data planes. P4-16 is the current version. |
| **p4c** | The P4 compiler. |
| **bmv2** | Behavioral Model v2 — the software P4 target. |
| **Tofino** | Intel/Barefoot's programmable switch ASIC. Discontinued 2023. |
| **P4Runtime** | gRPC API to control a P4 program from outside the switch. |
| **gNMI** | gRPC Network Management Interface. Streaming telemetry + config. |
| **gNOI** | gRPC Network Operations Interface. Operations like reboot, ping. |
| **OpenConfig** | Vendor-neutral YANG models, started by big network operators. |
| **YANG** | Data modeling language for network management. RFC 6020. |
| **NETCONF** | RPC-style network management over SSH or TLS. RFC 6241. |
| **RESTCONF** | NETCONF-style data access over HTTP/REST. RFC 8040. |
| **IBN** | Intent-Based Networking. Declare what you want, system figures out how. |
| **Cisco DNA** | Cisco's IBN product for campus/access. |
| **APIC** | Application Policy Infrastructure Controller. Cisco's ACI controller. |
| **ACI** | Application Centric Infrastructure. Cisco's data-center SDN. |
| **Apstra** | Juniper's vendor-neutral data-center IBN platform. |
| **CloudVision** | Arista's network management/telemetry platform. |
| **NSX** | VMware's network virtualization platform. |
| **Calico** | Kubernetes CNI focused on BGP-based networking. |
| **Cilium** | Kubernetes CNI built on eBPF. The dominant choice for modern clusters. |
| **hubble** | Cilium's observability tool. Reads eBPF flow data. |
| **kube-proxy** | Original Kubernetes Service load-balancer (iptables/IPVS). |
| **eBPF** | Extended Berkeley Packet Filter. Run small programs in the Linux kernel. |
| **XDP** | eXpress Data Path. eBPF hook before the network stack. |
| **network namespace** | Linux kernel feature that gives a process its own network stack. |
| **veth** | Virtual Ethernet pair. Two ends, one in each namespace. |
| **Istio** | Service mesh on Kubernetes. Envoy sidecars + Istiod control plane. |
| **Linkerd** | Service mesh. Lighter-weight than Istio. Rust data plane. |
| **Envoy** | The L7 proxy used as Istio's data plane. |
| **sidecar** | Helper container that runs next to the app and intercepts its traffic. |
| **mTLS** | Mutual TLS. Both sides authenticate. Service mesh standard. |
| **OPA** | Open Policy Agent. Policy engine often used with service meshes. |
| **ETSI MANO** | The telco standard model for NFV management and orchestration. |
| **ASIC** | Application-Specific Integrated Circuit. The custom chip in a switch. |
| **NPU** | Network Processing Unit. Programmable network chip, more flexible than ASIC. |
| **switch silicon** | Generic term for the chips inside switches. Broadcom, Mellanox, etc. |
| **SAI** | Switch Abstraction Interface. Standard API to switch silicon. |
| **SONiC** | Microsoft-originated open network OS. Linux + containers + SAI. |
| **Cumulus** | Network OS. Linux + FRR. Acquired by NVIDIA. |
| **DENT** | Lightweight Linux-kernel-forwarding network OS. |
| **ONIE** | Open Network Install Environment. PXE-style installer for whitebox switches. |
| **whitebox** | Bare-metal switch hardware. ODM, no OS, you bring one. |
| **brite-box** | Whitebox sold by a brand with their OS. |
| **STP** | Spanning Tree Protocol. Old loop avoidance for L2. |
| **OSPF** | Open Shortest Path First. Link-state IGP. RFC 2328. |
| **IS-IS** | Intermediate System to Intermediate System. Link-state IGP, popular with telcos. |
| **BGP** | Border Gateway Protocol. The Internet's routing protocol. RFC 4271. |
| **route reflector** | BGP peer that bounces routes between iBGP speakers. RFC 4456. |
| **iBGP** | Internal BGP. BGP between routers in the same AS. |
| **eBGP** | External BGP. BGP between routers in different ASes. |
| **AS** | Autonomous System. A network with one routing policy. |
| **ASN** | Autonomous System Number. Identifies an AS. |
| **VRF** | Virtual Routing and Forwarding. Multiple routing tables on one box. |
| **MPLS** | Multi-Protocol Label Switching. Label-based forwarding under IP. |
| **SRv6** | Segment Routing over IPv6. Source-routed paths in IPv6 packets. |
| **SR-MPLS** | Segment Routing over MPLS. Source-routed paths over MPLS labels. |
| **TE** | Traffic Engineering. Steering traffic on non-shortest paths. |
| **PCE** | Path Computation Element. Centralized brain that computes TE paths. |
| **PCEP** | PCE Communication Protocol. RFC 5440. |
| **TLV** | Type-Length-Value. Common encoding for extensible protocols. |
| **TCAM** | Ternary CAM. The chip memory that does flow-table lookups. |
| **CRC** | Cyclic Redundancy Check. The error-detection code in Ethernet frames. |
| **ECMP** | Equal-Cost Multi-Path. Hash flows across multiple equal paths. |
| **LACP** | Link Aggregation Control Protocol. Bundle physical links. |
| **LAG** | Link Aggregation Group. The bundle. |
| **CNI** | Container Network Interface. The plugin spec for Kubernetes networking. |
| **CRD** | Custom Resource Definition. Kubernetes way to add new object types. |
| **PoP** | Point of Presence. A physical site where the network meets the customer. |
| **CO** | Central Office. Telco's local exchange building. |
| **CORD** | Central Office Re-architected as Datacenter. ONOS's flagship use case. |
| **ICN** | Information Centric Networking. Different naming model; not really SDN. |
| **TLS** | Transport Layer Security. The encryption layer for network protocols. |
| **gRPC** | Google RPC over HTTP/2. The transport for gNMI, P4Runtime. |
| **protobuf** | Protocol Buffers. The serialization format under gRPC. |
| **JIT** | Just In Time compilation. eBPF programs are JITed to native code. |
| **CAP** | Consistency, Availability, Partition tolerance. The trade-off triangle. |
| **Raft** | Consensus protocol. Used by ONOS, ODL, etcd, Consul. |
| **etcd** | Distributed KV store using Raft. Backs Kubernetes and many controllers. |
| **bond** | Linux name for LAG. |
| **MLAG** | Multi-Chassis LAG. Same idea as MC-LAG. |
| **uplink** | The link from a leaf switch up to a spine switch. |
| **ToR** | Top of Rack. Old name for what we now call a leaf switch. |
| **CLOS** | The leaf-spine network shape. Named after Charles Clos. |
| **fabric** | A leaf-spine (or similar) network treated as one logical thing. |
| **overlay** | A virtual network on top of a physical network. |
| **underlay** | The physical IP network the overlay rides on. |
| **encapsulation** | Putting one packet inside another. VXLAN, Geneve, GRE. |
| **flow** | A stream of packets sharing the same 5-tuple (src/dst IP/port + proto). |
| **5-tuple** | (src IP, dst IP, src port, dst port, protocol) — uniquely names a flow. |
| **micro-segmentation** | Per-workload firewall rules. Enabled by SDN. |
| **east-west traffic** | Traffic between servers inside one data center. |
| **north-south traffic** | Traffic between the data center and the outside world. |

## Try This

1. Install OVS and create a single-bridge topology with three ports. Send packets with `ping` and watch flow counters increment with `ovs-ofctl dump-flows`.
2. Run Mininet's `--topo single,3` with `--controller none`. Watch every ping fail. Now switch to `--controller ovsc`. Watch every ping succeed. Explain to yourself in one sentence what changed.
3. Write a Ryu app that implements a "drop everything from h1" rule. Run it. Confirm h1's pings to h2 fail while h2's pings to h3 succeed.
4. Spin up a minikube cluster with Cilium. Apply a NetworkPolicy that allows traffic only between pods labeled `tier=app` and `tier=db`. Use `hubble observe` to watch the drops.
5. Set up two FRR instances peering BGP over a veth. Configure them as separate ASes. Use `vtysh` to see the BGP table and `ip route` to see the resulting kernel routes.
6. Compile the tiny P4 program in this sheet with `p4c`. Load it into bmv2 and feed it a packet. Observe the egress port.
7. Stand up Istio on minikube. Inject sidecars into a pair of httpbin pods. Use `istioctl proxy-config routes` to inspect what the sidecars know.
8. Subscribe to gNMI streaming telemetry from a virtualized router (e.g. Nokia SR Linux in containerlab). Watch counters update every 5 seconds.
9. Read your local Linux's MAC table (`bridge fdb show`), routing table (`ip route show`), and ARP cache (`ip neigh show`). For each, identify which control-plane component owns it.
10. Take any single switch (real or virtual) and write a one-page document mapping each of its tables (MAC, IP, ACL, QoS) to "who fills this in" and "who reads this." That mapping is the SDN architecture in miniature.

## Where to Go Next

Once this sheet feels comfortable, the natural next steps are:

- `ramp-up/kubernetes-eli5` — to ground the service-mesh and CNI material in Kubernetes basics.
- `ramp-up/ebpf-eli5` — to dig deeper into the kernel-programming side that powers Cilium and modern SDN.
- `networking/network-programmability` — for the practical day-to-day of configuring SDN platforms.
- `networking/evpn-advanced` — to learn the dominant production SDN of 2026.
- `networking/segment-routing` — the second dominant production SDN of 2026.
- `service-mesh/istio` — the deep dive on service-mesh SDN.

## See Also

- `ramp-up/ip-eli5` — the prerequisite on IP and packets.
- `ramp-up/bgp-eli5` — the prerequisite on BGP.
- `ramp-up/kubernetes-eli5` — Kubernetes basics for the CNI and service-mesh material.
- `ramp-up/ebpf-eli5` — eBPF foundations for Cilium.
- `ramp-up/linux-kernel-eli5` — kernel basics that underlie eBPF and software switching.
- `networking/cisco-aci` — Cisco's commercial data-center SDN.
- `networking/cisco-dna-center` — Cisco's commercial campus IBN.
- `networking/sd-wan` — SDN applied to wide-area connectivity.
- `networking/sd-access` — SDN applied to campus access.
- `networking/network-programmability` — practical NETCONF/RESTCONF/gNMI.
- `networking/netconf` — the NETCONF protocol in detail.
- `networking/restconf` — the RESTCONF protocol in detail.
- `networking/yang-models` — YANG data modeling in depth.
- `networking/evpn-advanced` — BGP-EVPN, the dominant production SDN.
- `networking/segment-routing` — source-routed traffic engineering.
- `service-mesh/istio` — the canonical service mesh.
- `service-mesh/cilium` — eBPF-native service mesh.
- `service-mesh/linkerd` — the lightweight alternative.
- `service-mesh/envoy` — the L7 proxy itself.

## References

- McKeown et al., "OpenFlow: Enabling Innovation in Campus Networks," ACM SIGCOMM CCR, 2008.
- Open Networking Foundation, OpenFlow Switch Specification 1.0 (2009), 1.3 (2012), 1.5 (2014).
- RFC 7348 — Virtual eXtensible Local Area Network (VXLAN).
- RFC 7432 — BGP MPLS-Based Ethernet VPN (EVPN).
- RFC 7752 — North-Bound Distribution of Link-State and TE Information using BGP (BGP-LS).
- RFC 5575 — Dissemination of Flow Specification Rules (BGP-FlowSpec).
- RFC 8040 — RESTCONF Protocol.
- RFC 6241 — Network Configuration Protocol (NETCONF).
- RFC 6020 — YANG Data Modeling Language.
- RFC 8926 — Geneve: Generic Network Virtualization Encapsulation.
- The P4 Language Consortium, "P4-16 Language Specification" (2016, current).
- OpenConfig — github.com/openconfig/public — vendor-neutral YANG models.
- gNMI Specification — github.com/openconfig/reference/tree/master/rpc/gnmi.
- ONOS — onosproject.org — Open Network Operating System.
- OpenDaylight — opendaylight.org.
- Faucet — faucet.nz.
- Ryu — github.com/faucetsdn/ryu.
- Open vSwitch — openvswitch.org.
- Cilium — cilium.io and the eBPF documentation at ebpf.io.
- Istio — istio.io.
- Linkerd — linkerd.io.
- Envoy — envoyproxy.io.
- SONiC — sonicfoundation.dev.
- ONIE — opencomputeproject.org / onie.org.
- Mininet — mininet.org.
- containerlab — containerlab.dev.
- FRRouting — frrouting.org.
- gobgp — github.com/osrg/gobgp.
- gnmic — gnmic.openconfig.net.
- Cisco DNA Center, Cisco APIC, Juniper Apstra, Arista CloudVision — vendor product documentation.
- "Network Programmability and Automation" by Edelman, Lowe, and Oswalt (O'Reilly).
- "BGP in the Data Center" by Dinesh Dutt (O'Reilly).
- "Linux Observability with BPF" by Calavera and Fontana (O'Reilly).
