# Spanning Tree Protocol — ELI5

> Spanning Tree Protocol (STP) is the "everyone agree on a path or there will be a stampede" referee for switched Ethernet. Without it, plug three switches into a triangle and a single broadcast frame loops forever, melting the network in a few seconds.

## Prerequisites

- Read `ramp-up/ip-eli5` first if you have not already, so you know what an IP address is and what a "subnet" means.
- It also helps to have heard the word "Ethernet" before, but if you have not, do not worry — we will explain it below in plain English.
- You should know what a "broadcast" is in human life: a broadcast is when one person yells something so that *everyone* in the room hears it. The same idea applies to networks. A "broadcast frame" is a little Ethernet message that says "everybody in this neighborhood, listen up."
- That is all. The rest of this sheet starts from zero.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## A Quick Picture Before We Begin

Imagine you and four friends are in a circle of rooms, and each room has a doorway to two other rooms. You whisper a secret to one friend. That friend whispers it to the next two friends. Each of those whisperers in turn whispers to the friends in their adjacent rooms. Within a few seconds, the same secret is being whispered to the same person from two different directions, and they whisper it back, and the original person hears it again, and whispers it forward again. The secret never dies. Everyone is whispering the same secret to everyone else, forever, louder and louder, until nobody can hear anything else.

That is what an Ethernet broadcast does in a network with a loop. It does not die. There is no clock that says "this whisper has been going around for 5 seconds, time to stop." There is no rule that says "we already heard this one, ignore it." Every switch faithfully forwards every broadcast frame out every other port. The frames duplicate at every hop. Within seconds you have a screaming howl of broadcasts and the network is dead.

The job of Spanning Tree Protocol is to walk into that circle of rooms and **close one of the doorways**. Now whispers can still travel — A can whisper to B, B can whisper to C, C can whisper to D. But the loop is broken. A whisper that arrives at D cannot be whispered back to A by closing the loop, because the loop-closing door is shut.

If somebody opens the closed door (a link comes up), STP closes a different door instead — the topology may change, but there is always exactly one shut door (one blocked port) per loop. That is the whole game.

## What Even Is STP — and why does Ethernet need a referee

### A switch is just a smart power strip for network cables

Imagine a power strip on the wall. You plug a lamp into it. You plug a TV into it. You plug a phone charger into it. Each thing you plug in works. The power strip does not care which thing is which — it just gives them all electricity.

A network **switch** is sort of like that, but for network cables instead of power. You plug a computer into one of its ports. You plug a printer into another port. You plug a server into another port. Each device can now send little messages (we call them **frames**) to each other through the switch. The switch listens for who is plugged in where, remembers it, and forwards messages from one port to another so they reach the right place.

That is the whole job of a switch: receive a frame on one port, look at the destination MAC address on that frame (a MAC address is the unique hardware nameplate of every network card on Earth), and forward the frame out the right port.

If a switch does not yet know which port the destination lives on, it does the dumb-but-effective thing: it **floods** the frame out every port except the one it came in on. The destination will eventually answer, the switch will see which port the answer came back on, and from then on it will know.

### Now connect two switches

You can plug a switch into another switch with a cable. This is called a **trunk** or just an "uplink" between switches. Now devices on switch A can talk to devices on switch B and the two switches together act like one big switch.

Why would anyone do this? Because real buildings have more devices than fit on one switch. A switch might have 24 ports. A building might have 500 computers. So you connect lots of switches together. This is normal.

### Now connect three switches in a triangle

Here is where the trouble starts.

Pretend you have three switches: **A**, **B**, and **C**. You connect:
- A to B
- B to C
- C to A

You did this because you wanted **redundancy**. If the cable between A and B breaks, traffic can still flow A → C → B. Cool. Smart. Resilient.

But there is a hidden monster in this design.

Suppose your laptop sends a single broadcast frame. A broadcast frame in Ethernet has a special destination MAC address: `ff:ff:ff:ff:ff:ff`. The rule for switches is: **when a switch receives a broadcast, it must forward it out every other port**.

So your laptop, plugged into switch A, sends one broadcast.

- Switch A receives the broadcast and forwards it out two ports: toward B and toward C.
- Switch B receives the broadcast (from A) and forwards it out two ports: back to A and onward to C.
- Switch C receives the broadcast (from A) and forwards it out two ports: back to A and onward to B.
- Switch C also receives the broadcast that B forwarded, so C forwards *that* one too — back to A and back to B.
- Switch B also receives the broadcast that C forwarded, so B forwards *that* one too.
- And on, and on, and on.

There is **no TTL** on Ethernet frames. There is **no hop counter**. The broadcast does not die. It will loop around the triangle forever, multiplying every time it passes through a switch.

In about two seconds, the entire link saturates with copies of one frame. Every device on every switch is screaming with broadcast traffic. CPUs spike to 100%. Web pages stop loading. VoIP phones go silent. The MAC address tables go insane because the same source MAC keeps showing up on every port (the switches keep "learning" and "unlearning" where the source is, and writing the result to flash storage on some platforms, eventually wearing the flash chips out). Ten seconds in, the network is dead. Welcome to a **broadcast storm**.

This is not a hypothetical. This has happened, accidentally and intentionally, in basically every datacenter on Earth. It is one of the oldest and most dangerous failure modes in networking.

### Enter the referee

In 1985, **Radia Perlman** wrote a beautiful little algorithm — and a beautiful little poem about it (*The Algorhyme*) — that solves this problem. Her algorithm finds, automatically, a **loop-free path** through any switched topology. It works no matter how many switches you have, no matter how they are wired together, and it heals automatically if a link breaks.

Her algorithm became **IEEE 802.1D**, the Spanning Tree Protocol, in 1990. Every Ethernet switch on Earth runs some flavor of it.

The whole job of STP is: **detect loops, block enough ports to break the loops, and unblock those ports if the active path fails.** That is it. That is the entire purpose. It does nothing else.

## The Broadcast Storm Problem

Let's draw the disaster.

```
        SwitchA
        /     \
       /       \
      /         \
   SwitchB ----- SwitchC

Laptop is plugged into SwitchA.
Laptop sends ONE broadcast.

t=0     A receives broadcast on port to laptop.
t=1ms   A floods broadcast out toward B and toward C.
t=2ms   B receives from A, floods toward C and BACK toward A.
t=2ms   C receives from A, floods toward B and BACK toward A.
t=3ms   A receives the copy from B, floods toward laptop and toward C.
t=3ms   A receives the copy from C, floods toward laptop and toward B.
t=3ms   B receives the copy from C, floods toward A.
t=3ms   C receives the copy from B, floods toward A.
t=4ms   The frames have multiplied. Now there are 8 in flight.
t=5ms   16 in flight.
t=10ms  Hundreds in flight.
t=2s    The links are 100% saturated.
        Every CPU on every switch is at 100%.
        Every host is being pummeled with broadcasts.
        ARP tables are flapping.
        MAC tables are flapping.
        The network is dead.
```

A real broadcast storm in a real datacenter killed thousands of dollars of business per minute, every minute, for as long as the loop existed. Storms have made hospitals stop. Storms have made 911 dispatch stop. Storms have made stock exchanges stop. They are silent and they are immediate.

### Why is there no TTL on Ethernet?

Because Ethernet was designed in the 1970s for **one cable in one office** with no loops. There were no switches yet, just shared coax. There was nothing to loop around. Ethernet has no hop counter, no time-to-live, no time-of-flight check, nothing. It is a **dumb broadcast medium** at heart, and the brilliant move of switches is that they pretend to be a dumb broadcast medium while actually being clever underneath.

But that pretense breaks if you create a physical loop. The frames will circulate forever because Ethernet has no built-in mechanism to stop them.

So you need an out-of-band mechanism — a *referee* — that says "this is the agreed-upon non-loopy path" and disables the rest. That referee is STP.

## The Solution: a Loop-Free Tree

A **tree** in graph theory is a connected graph with no cycles. Imagine a real tree: it has a trunk, the trunk splits into branches, branches split into smaller branches, smaller branches into twigs, twigs into leaves. There is exactly one path from the root of the tree to any leaf. There are no loops.

If the network is shaped like a tree, frames cannot loop. Because there are no loops.

A network can be wired any way you like (a "graph") but STP overlays a logical tree on top of that physical graph. STP picks one switch to be the **Root Bridge** (the trunk of the tree). Every other switch picks the single best port toward the root and calls it its **Root Port**. Every link between switches picks one switch to be the **Designated Switch** for that link (the one allowed to forward toward the leaves). Every other port — every port that would create a loop — gets put into **BLOCKING** state. A blocked port does not forward traffic. It just listens for control messages.

The result: a tree. One path from any switch to any other switch. No loops. Broadcasts can no longer storm because there is only one path the broadcast can take, and that path has no cycle.

### What "spanning" means

The "spanning" part of "spanning tree" means the tree must reach (span) every switch. Every switch must be in the tree. No switch is left out. Even if the tree includes only one path through a redundant pair of cables, both switches are reachable.

### What happens when a link fails

The clever part is what happens when a working link in the tree breaks. Imagine the tree has decided that A → B is the active path and C is a backup. Some idiot trips on the cable between A and B and unplugs it. STP notices (within 30-50 seconds in classic STP, milliseconds in RSTP) and **unblocks** the C port. The tree heals. Traffic flows again, now via C.

The network self-heals. The administrator does not have to log in and fix anything. STP did it. You may not even notice (with RSTP, you certainly will not notice for a moment).

### Drawing the spanning tree

Same physical triangle as before:

```
        SwitchA  <-- Root Bridge
        /     \
       /       \
   SwitchB     SwitchC

      [BLOCKED]
   SwitchB ------ SwitchC
```

The link between B and C still exists physically — the cable is still plugged in — but STP has chosen one of the two ports on that link and **blocked** it. Frames cannot pass through that port. The result is a tree:

```
        A
       / \
      B   C
```

No loops. Tree. Storm impossible.

If A → B fails, STP detects it, unblocks the B-to-C port, and reshapes the tree:

```
        A
        |
        C
        |
        B
```

Same tree shape (line). Different active links. No loops. Storm still impossible.

That is STP, in one drawing. The rest of this sheet is just the bookkeeping.

## Bridge Priority and Root Election

Before we can have a root, we need an election. STP elects exactly one Root Bridge per network.

The election uses something called the **Bridge ID**. The Bridge ID is a 64-bit number made of two parts:

- **Priority** (16 bits, default value `32768`)
- **Bridge MAC address** (48 bits, the actual hardware MAC of the switch)

Every switch starts up assuming **it is** the root. It announces itself with messages that include its own Bridge ID. Switches compare Bridge IDs. The switch with the **numerically lowest** Bridge ID wins. That switch is the Root Bridge. Every other switch backs down.

### Lowest priority wins

The priority is checked first. If two switches have different priorities, the lower one wins. So if you set switch X's priority to `4096` and every other switch has the default `32768`, switch X always wins because `4096 < 32768`.

### Tie? Lowest MAC wins

If priorities are tied (and they usually are, because everyone uses defaults), the Bridge MAC address is compared and the **numerically lowest MAC wins**. MAC addresses are 48-bit numbers (usually written as six hex pairs like `00:1c:42:fa:b1:7c`). So `00:1c:42:fa:b1:7c` beats `00:1c:42:fa:b1:7d`.

This is why the "default" Root Bridge in a network with all-default priorities tends to be **the oldest switch** — older switches were manufactured first and have lower-numbered MAC addresses. This is almost always the wrong switch to be the root, because the oldest switch is usually the slowest and least central. So in practice, network administrators **manually set** the priority of the switch they want to be the root.

### How to deliberately make a switch the root

```
SwitchA(config)# spanning-tree vlan 1 priority 4096
```

Or even simpler:

```
SwitchA(config)# spanning-tree vlan 1 root primary
```

That second command sets the priority to whatever value (lower than every other switch's current priority) is needed to win. It is a Cisco macro, not a real command — when you run it, the switch quietly sets `priority 4096` (or `8192` if `4096` is already taken).

You can also pre-stage a backup root:

```
SwitchB(config)# spanning-tree vlan 1 root secondary
```

That sets the priority to `16384`, which beats default `32768` but loses to a primary at `4096`. So if A dies, B becomes root automatically.

### Bridge Priority must be a multiple of 4096

Modern STP (the post-2004 version) uses something called **Extended System ID** which steals 12 bits out of the priority field to encode the VLAN ID. That leaves only 4 bits for the actual priority. So legal priority values are: `0, 4096, 8192, 12288, 16384, 20480, 24576, 28672, 32768, 36864, 40960, 45056, 49152, 53248, 57344, 61440`. Just multiples of 4096, from 0 to 61440.

If you try to set `priority 1234`, the switch will reject it.

### A picture of the election

```
   Priority  MAC                    Bridge ID
   --------  ---                    ---------
   A: 32768  00:1c:42:aa:aa:aa  ->  32768.00:1c:42:aa:aa:aa
   B: 32768  00:1c:42:bb:bb:bb  ->  32768.00:1c:42:bb:bb:bb
   C: 32768  00:1c:42:cc:cc:cc  ->  32768.00:1c:42:cc:cc:cc

A wins because aa:aa:aa is numerically lowest.
```

Now lower A's priority manually:

```
   Priority  MAC                    Bridge ID
   --------  ---                    ---------
   B: 4096   00:1c:42:bb:bb:bb  ->  4096.00:1c:42:bb:bb:bb
   A: 32768  00:1c:42:aa:aa:aa  ->  32768.00:1c:42:aa:aa:aa
   C: 32768  00:1c:42:cc:cc:cc  ->  32768.00:1c:42:cc:cc:cc

B wins now. Priority 4096 < 32768.
```

That is the election.

## BPDU (Bridge Protocol Data Unit) Format

Switches communicate with each other about the spanning tree using a special little message called a **BPDU**. BPDUs are sent every two seconds (by default) on every active port. They go to a multicast MAC address: `01:80:c2:00:00:00`. Hosts ignore BPDUs because hosts are not switches and do not care.

A Configuration BPDU (the most common kind) carries this information:

| Field                  | Bytes | Meaning                                                |
|------------------------|-------|--------------------------------------------------------|
| Protocol Identifier    | 2     | always 0x0000                                          |
| Protocol Version       | 1     | 0x00=STP, 0x02=RSTP, 0x03=MSTP                         |
| BPDU Type              | 1     | 0x00=config, 0x80=TCN                                  |
| Flags                  | 1     | TC, TCA, Proposal, Agreement, Port Role bits           |
| Root Bridge ID         | 8     | Priority + MAC of the believed root                    |
| Root Path Cost         | 4     | Cost from sender to root                               |
| Sender Bridge ID       | 8     | Priority + MAC of the sender                           |
| Sender Port ID         | 2     | Port priority + port number                            |
| Message Age            | 2     | How long since the root sent this (in 1/256 sec units) |
| Max Age                | 2     | When to discard this BPDU (default 20s)                |
| Hello Time             | 2     | Send interval (default 2s)                             |
| Forward Delay          | 2     | State transition timer (default 15s)                   |
| Version 1 Length       | 1     | RSTP-only, set to 0x00                                 |

Total: 35 bytes for an STP/RSTP config BPDU. MSTP adds more after that.

### How BPDUs flood

The Root Bridge sends BPDUs out every port every 2 seconds. Other switches **do not generate** BPDUs of their own (mostly — non-root switches generate BPDUs when forwarding/relaying, but the Root Bridge is the original source). They receive a BPDU on their Root Port, update the Root Path Cost field (adding their port's path cost), and forward the modified BPDU out their Designated Ports.

This means the cost field grows as the BPDU travels away from the root, naturally encoding "distance from root."

### Path cost values

The IEEE-recommended path cost depends on link speed:

| Speed     | Old (short) cost | New (long) cost |
|-----------|------------------|-----------------|
| 10 Mbps   | 100              | 2000000         |
| 100 Mbps  | 19               | 200000          |
| 1 Gbps    | 4                | 20000           |
| 10 Gbps   | 2                | 2000            |
| 100 Gbps  | (n/a)            | 200             |
| 1 Tbps    | (n/a)            | 20              |

Older STP used the "short" 16-bit cost field. Modern STP/RSTP/MSTP supports 32-bit "long" path cost. You enable long cost on Cisco with:

```
SwitchA(config)# spanning-tree pathcost method long
```

Use long path cost in any modern data center, otherwise 100G and 10G look the same to STP (both are cost 2 in short cost) and you cannot distinguish them.

## Configuration BPDU vs TCN BPDU

There are two kinds of BPDU you will see in classic STP:

### Configuration BPDU

The normal kind. Sent by the root every Hello (default 2 seconds). Contains the full state of the tree (root ID, path cost, sender ID, port ID, timers). This is what propagates the tree out to every switch.

### TCN BPDU (Topology Change Notification)

Tiny BPDU (4 bytes). Sent **upstream toward the root** when a switch detects a topology change (a link came up, a link went down, a port changed state). The TCN says, in effect, "hey root, something changed down here."

The flow:
1. Switch X notices a topology change (e.g., its Root Port went down).
2. Switch X sends a TCN BPDU **out its Root Port** (toward the root).
3. The next switch upstream receives the TCN, sets the **TCA flag** in its config BPDUs back to X (acknowledging), and forwards the TCN further upstream.
4. Eventually the TCN reaches the root.
5. The root sets the **TC flag** in its outgoing config BPDUs for the next `Forward Delay + Max Age` (default 35 seconds).
6. Every switch receiving a config BPDU with the TC flag flushes its MAC address table (using a shorter aging time, 15 seconds, instead of the usual 300).

The point: a topology change might mean MAC addresses live behind different ports now, so we need to relearn the topology. We flush the MAC tables.

In RSTP this is done differently and faster — RSTP eliminated TCN as a separate frame type; topology changes are signaled within the regular BPDU flags and propagate immediately, not via a separate notification chain.

## Port Roles

Every port on every switch participating in STP has a **role**. The role tells you what the port is *for* in the tree.

### Root Port (RP)

The port on a non-root switch that has the lowest path cost back to the Root Bridge. Every non-root switch has exactly one Root Port. That is the port pointing "up" toward the root.

The Root Bridge itself has no Root Port (it *is* the root).

### Designated Port (DP)

For every link (segment), exactly one switch must be the "Designated Switch" — the switch that is allowed to forward traffic onto that segment toward the leaves. The port on that designated switch facing the segment is the Designated Port.

The Root Bridge's ports are *all* designated (because the root is the closest thing to itself, so it always wins).

### Alternate Port (Classic STP calls this "Blocking")

A port that is **not** the Root Port but receives BPDUs from the same root via a different path. It is blocked. If the Root Port goes down, an Alternate Port can quickly take over (in RSTP, in milliseconds; in classic STP, after 30-50 seconds of timer expiration).

### Backup Port

A port that is on the same switch as a Designated Port for the same segment. Rare — it only appears when a single switch has two ports plugged into the same shared segment (e.g., two ports plugged into the same hub). Almost never seen in modern networks.

### Disabled

A port that is administratively shut down or has no link. Not part of STP at all.

### A picture of port roles

```
                       SwitchA (Root)
                        /        \
                       DP         DP   <-- A's ports are all Designated
                       |          |
                       RP         RP   <-- B and C see A as the path to root
                     SwitchB    SwitchC
                        \        /
                         DP    ALT     <-- One side of the B-C link
                          \    /            is Designated, the other is Alternate
                           \  /              (blocked).
                          (one cable)
```

That is the entire STP tree as a picture. Every port has a role. Every link has exactly one Designated Port and (depending on what is on the other end) either a Root Port (toward root) or an Alternate Port (blocked).

## Port States in Classic STP

In classic 802.1D (1990 / 1998), every port that is up moves through five **states** as the tree is built. The transition takes 30-50 seconds in the worst case. This is why classic STP is slow.

### 1. Disabled

Port is administratively down. Not in the tree at all.

### 2. Blocking

Port is up but does not forward traffic. Does not learn MAC addresses. Just listens for BPDUs to know if it should become a Root Port or Designated Port. This is the steady state for an Alternate Port.

### 3. Listening (Forward Delay seconds, default 15)

Port has decided it might become a forwarding port. It listens to BPDUs and starts participating in the election but does not yet forward data and does not learn MAC addresses. Lasts 15 seconds by default.

### 4. Learning (Forward Delay seconds, default 15)

Port still does not forward data, but it does start learning MAC addresses (so when it does forward, the MAC table will be primed). Lasts 15 seconds by default.

### 5. Forwarding

Port forwards data normally. This is the steady state for Root Ports and Designated Ports.

### Why 30-50 seconds?

The walkthrough:
- Time 0: link comes up. Port enters Blocking.
- Time 0 to ~20: Port stays Blocking until Max Age expires (20 seconds default) on stale BPDUs and the port realizes nothing better is on the other end.
- Time 20 to 35: Port enters Listening for 15 seconds (Forward Delay).
- Time 35 to 50: Port enters Learning for 15 seconds.
- Time 50: Port enters Forwarding.

So a freshly plugged-in port takes 30 seconds (skipping the Max Age phase if no stale BPDUs are in play) or 50 seconds (full storm-avoidance dance) to start passing traffic.

Imagine plugging your laptop into a wall jack and waiting **50 seconds** before it can talk to the network. Yeah, that is bad. Users hated it. Engineers hated it. So we got RSTP.

## RSTP (802.1w) — the Faster Spanning Tree

**RSTP** (Rapid Spanning Tree Protocol) was published as IEEE 802.1w in 2001. It uses the same underlying ideas as STP but converges in **less than one second** in most cases.

### Three states instead of five

RSTP collapses the port states down to three:
- **Discarding** — equivalent to Disabled + Blocking + Listening (does not forward, does not learn).
- **Learning** — does not forward, does learn.
- **Forwarding** — forwards and learns.

The state machine is simpler.

### Proposals and Agreements

The big speed-up: when a link comes up between two RSTP switches, they immediately exchange a **Proposal/Agreement handshake**:
1. The switch closer to the root sends a Proposal: "I want this link to be Designated, and you should be Root for me."
2. The other switch checks: am I OK with this? It then **syncs** its other ports (sets them to Discarding briefly, to make sure no loop sneaks through during the handshake).
3. The other switch sends back an Agreement: "Yes, I agree."
4. Both ports immediately go to Forwarding.

This handshake takes **milliseconds**, not seconds. There is no 15+15 second wait. The price is that other ports on the receiving switch briefly go Discarding during the sync — but those are mostly edge ports anyway, and edge ports are exempt from the sync (see below).

### Edge Ports

RSTP introduced the concept of an **edge port** — a port connected to a host (laptop, server, printer) and not to another switch. Edge ports skip all the BPDU dance entirely and go straight to Forwarding the instant they come up.

You configure edge ports manually (or via heuristic — RSTP can detect "I never heard a BPDU on this port" and assume it's edge). On Cisco:

```
SwitchA(config-if)# spanning-tree portfast
```

(Yes, in Cisco-land, "PortFast" predates RSTP — it was a Cisco extension to classic STP — but with RSTP the concept became standard as "edge port".)

### Point-to-Point vs Shared

RSTP needs to know whether a link is point-to-point (full-duplex) or shared (half-duplex hub). Point-to-point links can use the fast Proposal/Agreement handshake. Shared links cannot — they fall back to classic timers. In modern networks, every link is full-duplex by default, so this is automatic.

### Rapid convergence example

```
Plug a fresh switch C into existing tree A-B.

t=0      C's port to B comes up. B's port to C comes up.
t=10ms   B sends Proposal: "I'm root via A. I want to be Designated."
t=20ms   C agrees. Sends Agreement.
t=25ms   Both ports forwarding.
```

25 milliseconds. Compare to 50 seconds in classic STP. This is why everybody uses RSTP today.

## MSTP (802.1s) — multiple regions, multiple trees

The problem RSTP does not solve: in a network with VLANs (multiple Layer 2 broadcast domains running on the same physical switches), a single tree means a single set of blocked links. Imagine a triangle of switches with link A-B blocked. Every VLAN's traffic is forced through A-C-B, never A-B. Half your bandwidth is wasted.

You'd like to be able to say: "VLAN 10's tree blocks A-B, but VLAN 20's tree blocks A-C." That way, VLAN 10 traffic uses A-C and VLAN 20 traffic uses A-B — both links are utilized, total bandwidth doubles.

**MSTP** (Multiple Spanning Tree Protocol, IEEE 802.1s, 2002, later folded into 802.1Q-2005) does exactly this.

### Regions

MSTP groups switches into **regions**. A region is a set of switches that share:
- The same configuration name
- The same revision number
- The same VLAN-to-instance mapping

If any of these three things is different on two switches, they are in **different regions** even if they are physically connected.

Inside a region, MSTP runs its own tree per **MST Instance** (MSTI). Each MSTI has its own tree, with its own root, blocking different links.

Between regions (and to the outside world), MSTP runs the **Common and Internal Spanning Tree** (CIST), which looks like RSTP from outside the region.

### Configuration name and revision

Every switch in a region must have:

```
SwitchA(config)# spanning-tree mode mst
SwitchA(config)# spanning-tree mst configuration
SwitchA(config-mst)# name PROD
SwitchA(config-mst)# revision 1
SwitchA(config-mst)# instance 1 vlan 10,20
SwitchA(config-mst)# instance 2 vlan 30,40
```

If two switches have different `name`, `revision`, or `instance ... vlan ...` lines, they are in different regions and MSTP between them collapses to a single CIST.

### Why "digest"?

MSTP computes a hash (MD5) of the VLAN-to-instance mapping table and includes it in BPDUs. This is the **MST Configuration Digest**. Switches compare digests; if they match, they agree on the mapping. If digests differ, the switches are in different regions — even if everything *looks* the same. Common bug: VLAN 100 mapped to instance 1 on switch A, mapped to instance 2 on switch B → different digests → different regions → MSTP fails between them.

### Picture of MSTP regions

```
Region "PROD", revision 1:
  +-----+         +-----+
  |  A  |---------|  B  |
  +-----+         +-----+
     |               |
  +-----+         +-----+
  |  C  |---------|  D  |
  +-----+         +-----+

Inside Region PROD:
  Instance 1 (VLAN 10,20):  Root=A.  Blocks B-D.
  Instance 2 (VLAN 30,40):  Root=B.  Blocks A-C.

Result: VLAN 10 traffic uses A-B-D and A-C-D.
        VLAN 30 traffic uses B-D and B-A-C.
        Both links fully utilized.
```

## PVST+ / Rapid PVST+

**PVST+** (Per-VLAN Spanning Tree Plus) is Cisco's pre-MSTP solution to the same problem: run *one entire spanning tree per VLAN*. With 100 VLANs, you have 100 spanning trees, each with its own root, its own BPDUs, its own blocking decisions.

This works. It works really well, actually, because every VLAN can have a different root and you get full per-VLAN flexibility. The downside: **CPU**. Every BPDU has to be sent and processed per-VLAN. With 1000 VLANs and 50 switches, your switch CPUs are busy.

**Rapid PVST+** is exactly the same but using RSTP timers and state machine instead of classic STP. Most modern Cisco shops use Rapid PVST+ until they outgrow it, then move to MSTP for scale.

### When to pick which

| Scale                    | Recommended                    |
|--------------------------|--------------------------------|
| <50 switches, <50 VLANs  | Rapid PVST+ (default on Cisco) |
| Multi-vendor environment | MSTP (only standard option)    |
| 100+ VLANs, large fabric | MSTP (less BPDU overhead)      |
| Single site, all Cisco   | Rapid PVST+ (simpler)          |

### A comparison table

| Feature                  | Classic STP | RSTP   | PVST+    | Rapid PVST+ | MSTP   |
|--------------------------|-------------|--------|----------|-------------|--------|
| IEEE standard            | 802.1D-1998 | 802.1w | Cisco    | Cisco       | 802.1s |
| Convergence              | 30-50s      | <1s    | 30-50s   | <1s         | <1s    |
| Trees per network        | 1           | 1      | 1 / VLAN | 1 / VLAN    | 1 / MSTI |
| Vendor-portable          | Yes         | Yes    | No (Cisco) | No (Cisco) | Yes    |
| BPDU CPU cost            | Low         | Low    | High     | High        | Low    |
| Year                     | 1990        | 2001   | 2002     | 2003        | 2002   |

## BPDU Guard, Root Guard, Loop Guard, BPDU Filter

These are the safety harness for STP. Without them, a single misconfigured port can break your tree.

### BPDU Guard

If a port that is supposed to be an edge port (PortFast) receives a BPDU, **err-disable** the port. Reasoning: if a host port suddenly receives a BPDU, somebody has plugged a switch into a port that should only have a host on it — possibly accidentally, possibly maliciously. Shut the port down before the rogue switch can mess up the tree.

```
SwitchA(config-if)# spanning-tree bpduguard enable
```

When triggered, the port goes into **err-disabled** state and stays there until you clear it (or err-disable recovery kicks in).

```
%SPANTREE-2-BLOCK_BPDUGUARD: Received BPDU on port GigabitEthernet0/5 with BPDU Guard enabled. Disabling port.
%PM-4-ERR_DISABLE: bpduguard error detected on Gi0/5, putting Gi0/5 in err-disable state
```

### Root Guard

If a port that is supposed to be facing **away** from the root (a Designated Port) starts receiving a *superior* BPDU (a BPDU that claims a better root), Root Guard puts the port in **root-inconsistent** state. The port stays up, keeps sending BPDUs, but does not forward data. Reasoning: if some downstream switch suddenly claims to be the root, you do not want it to actually become the root and reshape your topology — it is probably misconfigured (or a rogue switch was plugged in).

```
SwitchA(config-if)# spanning-tree guard root
```

```
%SPANTREE-2-ROOTGUARD_BLOCK: Root guard blocking port GigabitEthernet0/5
```

When the superior BPDUs stop arriving, the port automatically returns to forwarding.

### Loop Guard

If a Root Port or Alternate Port stops receiving BPDUs (which would normally make STP think the link is dead and unblock the port), Loop Guard instead puts the port in **loop-inconsistent** state. Reasoning: a port stopping BPDU reception could mean the link is dead, OR it could mean unidirectional failure where traffic flows one way but not the other (e.g., a fiber transmit died but receive still works). In the unidirectional case, simply unblocking the port creates a loop. Loop Guard says "if I stopped hearing BPDUs, I'd rather stay blocked than risk a loop."

```
SwitchA(config-if)# spanning-tree guard loop
```

```
%SPANTREE-2-LOOPGUARD_BLOCK: Loop guard blocking port GigabitEthernet0/5
```

### BPDU Filter

The opposite of BPDU Guard. BPDU Filter says "do not send BPDUs on this port and do not act on received BPDUs." Used in special cases (service provider edges where customer switches should not see provider BPDUs). **Dangerous** — easy to misuse and cause loops. Do not use casually.

```
SwitchA(config-if)# spanning-tree bpdufilter enable
```

### Together: PortFast + BPDU Guard

The two go hand in hand. PortFast skips the listening/learning states for host ports. BPDU Guard ensures that if anyone plugs a *switch* into a host port, it gets shut down. The combo is essentially mandatory on every access port:

```
SwitchA(config-if)# switchport mode access
SwitchA(config-if)# spanning-tree portfast
SwitchA(config-if)# spanning-tree bpduguard enable
```

You can apply this globally:

```
SwitchA(config)# spanning-tree portfast default
SwitchA(config)# spanning-tree portfast bpduguard default
```

That makes every newly configured access port get PortFast + BPDU Guard automatically.

## PortFast / Edge Port

PortFast is the original "skip the wait" feature, predating RSTP by years. With PortFast on a port:

- The port does **not** go through Listening and Learning states.
- The port goes directly Blocking → Forwarding when link comes up.
- The port does **not** generate TCN BPDUs when it transitions states (because, if it's just a host coming and going, the rest of the tree doesn't need to know).

In RSTP/MSTP this is called an **edge port**. Cisco kept the name "PortFast" for backwards compatibility.

```
SwitchA(config-if)# spanning-tree portfast
%Warning: portfast should only be enabled on ports connected to a single
host. Connecting hubs, concentrators, switches, bridges, etc.. to this
interface when portfast is enabled, can cause temporary bridging loops.
Use with CAUTION
```

That warning is real. PortFast on a switch-to-switch port WILL eventually cause a loop. Pair with BPDU Guard.

## UplinkFast / BackboneFast

These are **legacy** Cisco extensions to classic STP that accelerated convergence. They are obsolete because RSTP does the same thing better. Mentioned here for historical context — you may see them in old configs.

### UplinkFast

Detects the loss of a Root Port and immediately fails over to an Alternate Port (skipping the listening/learning wait). Roughly equivalent to RSTP's behavior for Alternate Ports.

```
SwitchA(config)# spanning-tree uplinkfast
```

### BackboneFast

Speeds up reconvergence after **indirect** link failures (a link two switches away goes down). It uses a special protocol called RLQ (Root Link Query) to short-circuit the Max Age timer.

```
SwitchA(config)# spanning-tree backbonefast
```

If you are running RSTP, do not enable these. They do nothing useful and may interfere.

## Why Spine-Leaf Doesn't Use STP

Modern data centers use **spine-leaf fabrics** (read `ramp-up/spine-leaf-eli5` for the deep dive). Spine-leaf is not switched at Layer 2 — it is **routed at Layer 3** between every leaf and every spine. There are no Ethernet broadcast domains spanning the whole fabric, just point-to-point Layer 3 links.

Why? Because:
- **ECMP** (Equal-Cost Multi-Path routing) lets you use *every* uplink simultaneously. STP forces you to block all but one. ECMP wastes none.
- **Faster failure response** — IP routing protocols (BGP, OSPF) reconverge in <1 second, similar to RSTP, but without the loop danger.
- **No broadcast storms possible** — Layer 3 IP packets have a TTL. Loops self-limit.

So in spine-leaf, STP runs only inside individual leaf switches' downstream-facing access ports (where end hosts connect). The fabric itself is loop-free *by design*, not by referee.

This is why "STP is dead in modern data centers" is half true. STP is dead in the fabric. STP is alive and well at the **access layer** (the edge where end hosts plug in). Every wall jack in every office still has STP enabled.

## Common Errors

Verbatim error strings you will see when STP misbehaves. Memorize the patterns.

### 1. BPDU Guard tripped

```
%SPANTREE-2-BLOCK_BPDUGUARD: Received BPDU on port GigabitEthernet0/5 with BPDU Guard enabled. Disabling port.
%PM-4-ERR_DISABLE: bpduguard error detected on Gi0/5, putting Gi0/5 in err-disable state
%LINK-3-UPDOWN: Interface GigabitEthernet0/5, changed state to down
```

Cause: someone plugged a switch into a host-port (PortFast + BPDU Guard configured port).

Fix: unplug the rogue switch, then `clear errdisable interface gi0/5` or `shutdown` / `no shutdown`.

### 2. Topology change generated

```
%SPANTREE-6-PORT_STATE: Port Gi0/5 instance 0 moving from forwarding to blocking
%SPANTREE-6-PORTSTATE: vlan 10 port Gi0/5 state changed from forwarding to learning
STP topology change generated by port Gi0/5
```

Cause: a port went up or down, or moved between states. Not necessarily an error — just informational. But if you see this *constantly*, something is flapping.

### 3. Wrong mode

```
spanning-tree mode rapid-pvst
```

Cisco command to set the global STP mode to Rapid PVST+. Or:

```
spanning-tree mode mst
spanning-tree mode pvst
```

### 4. Root Guard tripped

```
%SPANTREE-2-ROOTGUARD_BLOCK: Root guard blocking port GigabitEthernet0/5 on VLAN 10.
%SPANTREE-2-ROOTGUARD_UNBLOCK: Root guard unblocking port GigabitEthernet0/5 on VLAN 10.
```

Cause: a downstream switch sent a BPDU claiming to be a better root than the real root. Root Guard kept it from taking over.

Fix: find the rogue switch and either remove it or fix its priority.

### 5. Loop Guard tripped

```
%SPANTREE-2-LOOPGUARD_BLOCK: Loop guard blocking port GigabitEthernet0/5 on VLAN 10.
%SPANTREE-2-LOOPGUARD_UNBLOCK: Loop guard unblocking port GigabitEthernet0/5 on VLAN 10.
```

Cause: a port stopped receiving BPDUs but is still up. Possibly a unidirectional failure on a fiber.

Fix: replace the fiber, or check for one-way SFP/transceiver failure.

### 6. STP convergence timeout

```
%STP-3-CONVERGENCE_TIMEOUT: STP did not converge within the expected time on VLAN 100
```

Cause: BPDUs are not flowing somewhere they should. Check cabling, check `shutdown` interfaces, check VLAN propagation.

### 7. BPDUs from another VLAN

```
%SPANTREE-7-RECV_OTHER: Received BPDU for VLAN 30 on port Gi0/5 (VLAN 10).
```

Cause: a port is configured as access VLAN 10 but receiving BPDUs tagged for VLAN 30. Native VLAN mismatch.

Fix: check the trunk's allowed-VLAN list and native VLAN configuration on both ends.

### 8. Superior BPDU on Designated port

```
%SPANTREE-2-RECV_1Q_NON_TRUNK: Received 802.1Q BPDU on non-trunk Gi0/5 VLAN1
%SPANTREE-2-UNBLOCK_CONSIST_PORT: Unblocking port Gi0/5 VLAN0010
%SPANTREE-2-RECV_PVID_ERR: Received BPDU with inconsistent peer vlan id 30 on Gi0/5 VLAN0010
Superior BPDU received on port that should be designated.
```

Cause: a port that should be Designated (toward leaves) suddenly got a BPDU claiming a better root from below.

Fix: enable Root Guard on the port to prevent the takeover.

### 9. Root inconsistent

```
SwitchA# show spanning-tree inconsistentports
Name                 Interface            Inconsistency
-------------------- -------------------- ------------------
VLAN0010             GigabitEthernet0/5   Root Inconsistent
Number of inconsistent ports (segments) in the system : 1
```

Cause: Root Guard caught a superior BPDU. The port is up but blocked.

Fix: same as Root Guard tripped — find and fix the rogue switch.

### 10. MSTP region mismatch

```
%MST-4-REGION_MISMATCH: MST region mismatch with Gi0/5: name=PROD-1 expected=PROD revision=2 expected=1
%MST-4-REGION_DIGEST_MISMATCH: MST region digest mismatch with Gi0/5
```

Cause: the switch on the other end of `Gi0/5` has a different MSTP region (different name, revision, or VLAN-to-instance mapping). MSTP cannot run across the link in MST mode; it falls back to CIST.

Fix: align the region name, revision, and `instance N vlan X,Y,Z` lines across all switches in the region.

### 11. Type-inconsistent

```
%SPANTREE-2-INCONSISTENCY: Type-Inconsistent BPDU received on port Gi0/5 from another bridge.
```

Cause: a switch on the other side is in a different mode (e.g., one is PVST+, the other MSTP, and they disagree on how to encode VLAN information in BPDUs).

Fix: use compatible STP modes. Cisco can interop PVST+ ↔ MSTP via the "PVST+ TLV" extension, but it requires specific configuration.

### 12. PVID mismatch (CDP-detected)

```
%CDP-4-NATIVE_VLAN_MISMATCH: Native VLAN mismatch discovered on GigabitEthernet0/5 (10), with SwitchB GigabitEthernet0/5 (20).
```

Cause: the native VLAN on a trunk is different on the two ends. STP BPDUs travel on the native VLAN, so this can break STP.

Fix: align native VLAN on both ends.

## Hands-On

Paste-ready commands. Use whichever applies to your platform.

### Cisco IOS / IOS-XE / NX-OS

```
SwitchA# show spanning-tree

SwitchA# show spanning-tree summary
Switch is in rapid-pvst mode
Root bridge for: VLAN0001
Extended system ID                    is enabled
Portfast Default                      is disabled
PortFast BPDU Guard Default           is disabled
Portfast BPDU Filter Default          is disabled
Loopguard Default                     is disabled
EtherChannel misconfig guard          is enabled
UplinkFast                            is disabled
BackboneFast                          is disabled
Configured Pathcost method used is short

Name                   Blocking Listening Learning Forwarding STP Active
---------------------- -------- --------- -------- ---------- ----------
VLAN0001                     0         0        0          5          5

SwitchA# show spanning-tree root
                                        Root    Hello Max Fwd
Vlan                   Root ID          Cost    Time  Age Dly  Root Port
---------------- -------------------- --------- ----- --- --- ---------
VLAN0001         32769 0050.7966.6800     19      2   20  15  Gi0/1

SwitchA# show spanning-tree bridge
                                                   Hello  Max  Fwd
Vlan                         Bridge ID              Time   Age  Dly  Protocol
---------------- --------------------------------- -----  ---  ---  --------
VLAN0001         32769 (priority 32768 sys-id-ext 1)
                       0050.7966.6900               2      20   15   rstp

SwitchA# show spanning-tree vlan 10
VLAN0010
  Spanning tree enabled protocol rstp
  Root ID    Priority    24586
             Address     0050.7966.6800
             Cost        4
             Port        1 (GigabitEthernet0/1)
             Hello Time   2 sec  Max Age 20 sec  Forward Delay 15 sec

  Bridge ID  Priority    32778  (priority 32768 sys-id-ext 10)
             Address     0050.7966.6900
             Hello Time   2 sec  Max Age 20 sec  Forward Delay 15 sec
             Aging Time  300 sec

Interface           Role Sts Cost      Prio.Nbr Type
------------------- ---- --- --------- -------- ----------
Gi0/1               Root FWD 4         128.1    P2p
Gi0/2               Desg FWD 4         128.2    P2p
Gi0/3               Altn BLK 4         128.3    P2p

SwitchA# show spanning-tree interface Gi0/1 detail
 Port 1 (GigabitEthernet0/1) of VLAN0010 is root forwarding
   Port path cost 4, Port priority 128, Port Identifier 128.1.
   Designated root has priority 24586, address 0050.7966.6800
   Designated bridge has priority 24586, address 0050.7966.6800
   Designated port id is 128.1, designated path cost 0
   Timers: message age 2, forward delay 0, hold 0
   Number of transitions to forwarding state: 1
   Link type is point-to-point by default
   BPDU: sent 1, received 102

SwitchA# show spanning-tree mst

##### MST0    vlans mapped:   1-9,11-19,21-4094
Bridge        address 0050.7966.6900  priority      32768 (32768 sysid 0)
Root          this switch for the CIST
Operational   hello time 2 , forward delay 15, max age 20, txholdcount 6
Configured    hello time 2 , forward delay 15, max age 20, max hops    20

Interface        Role Sts Cost      Prio.Nbr Type
---------------- ---- --- --------- -------- --------------------------------
Gi0/1            Desg FWD 20000     128.1    P2p
Gi0/2            Desg FWD 20000     128.2    P2p

SwitchA# show spanning-tree mst configuration
Name      [PROD]
Revision  1     Instances configured 3

Instance  Vlans mapped
--------  ---------------------------------------------------------------------
0         1-9,11-19,21-29,31-4094
1         10,20
2         30
-------------------------------------------------------------------------------

SwitchA# show errdisable detect
ErrDisable Reason            Detection    Mode
-----------------            ---------    ----
all                          Enabled      port
arp-inspection               Enabled      port
bpduguard                    Enabled      port
dtp-flap                     Enabled      port
gbic-invalid                 Enabled      port
inline-power                 Enabled      port
link-flap                    Enabled      port
loopback                     Enabled      port
pagp-flap                    Enabled      port
psp                          Enabled      port
security-violation           Enabled      port
udld                         Enabled      port

SwitchA# show errdisable recovery
ErrDisable Reason            Timer Status
-----------------            --------------
bpduguard                    Disabled
udld                         Disabled
arp-inspection               Disabled
link-flap                    Disabled

Timer interval: 300 seconds
Interfaces that will be enabled at the next timeout:

SwitchA# show interface status err-disabled
Port      Name               Status       Reason               Err-disabled Vlans
Gi0/5                        err-disabled bpduguard

SwitchA# debug spanning-tree events
Spanning Tree event debugging is on
%SPANTREE-7-PORT_STATE: Port Gi0/5 vlan 1 moving from learning to forwarding
%SPANTREE-7-RECV_BPDU: Received BPDU on port Gi0/5 vlan 1, root 24586/0050.7966.6800
```

### Linux bridge with `mstpd`

```
$ sudo apt install mstpd ifupdown2

$ sudo ip link add name br0 type bridge
$ sudo ip link set br0 up
$ sudo ip link set eth1 master br0
$ sudo ip link set eth2 master br0

$ sudo ip link set br0 type bridge stp_state 1     # Enable STP

$ sudo brctl showstp br0
br0
 bridge id              8000.000c2987a17e
 designated root        8000.000c2987a17e
 root port                 0                    path cost                  0
 max age                  20.00                 bridge max age            20.00
 hello time                2.00                 bridge hello time          2.00
 forward delay            15.00                 bridge forward delay      15.00
 ageing time             300.00
 hello timer               0.62                 tcn timer                  0.00
 topology change timer     0.00                 gc timer                  29.99
 flags

eth1 (1)
 port id                8001                    state                forwarding
 designated root        8000.000c2987a17e       path cost                  4
 designated bridge      8000.000c2987a17e       message age timer          0.00
 designated port        8001                    forward delay timer        0.00
 designated cost           0                    hold timer                 0.62
 flags

eth2 (2)
 port id                8002                    state                  blocking
 designated root        8000.000c2987a17e       path cost                  4
 designated bridge      8000.000c2987a17e       message age timer          0.00
 designated port        8002                    forward delay timer        0.00
 designated cost           0                    hold timer                 0.00
 flags

$ sudo ip link show type bridge_slave
4: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 master br0 state forwarding ...
5: eth2: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 master br0 state blocking ...

$ sudo systemctl start mstpd
$ sudo mstpctl showtree br0 0
bridge        br0
admin / oper bridge id   8.000.00:0c:29:87:a1:7e / 8.000.00:0c:29:87:a1:7e
admin / oper external root path cost   20000 / 20000
regional root  8.000.00:0c:29:87:a1:7e
internal root path cost   0
hello time   2 / 2     forward delay  15 / 15  max age  20 / 20  tx hold count   6 / 6
max hops    20 / 20
time since topology change   33s
topology change count   1
topology change   no
topology change port   None
last topology change port   None

$ sudo mstpctl setforcevers br0 rstp
$ sudo mstpctl settreeprio br0 0 8

$ sudo sysctl net.bridge.bridge-nf-call-iptables
net.bridge.bridge-nf-call-iptables = 1

$ sudo sysctl -w net.bridge.bridge-nf-filter-vlan-tagged=0
```

### Open vSwitch (OVS)

```
$ sudo ovs-vsctl add-br br0
$ sudo ovs-vsctl add-port br0 eth1
$ sudo ovs-vsctl add-port br0 eth2

$ sudo ovs-vsctl set Bridge br0 stp_enable=true
$ sudo ovs-vsctl set Bridge br0 rstp_enable=true       # RSTP instead of classic STP

$ sudo ovs-appctl rstp/show br0
---- br0 ----
Root ID:
   stp-priority    32768
   stp-system-id   00:0c:29:87:a1:7e
   root-path-cost  0
   root-port       n/a
   stp-hello-time  2s
   stp-max-age     20s
   stp-fwd-delay   15s

Bridge ID:
   stp-priority    32768
   stp-system-id   00:0c:29:87:a1:7e

Interface            Role       State        Cost  Pri.Nbr  Type
---------- ---------- ----------- ----- ------- ----
eth1       designated forwarding  20000 128.1   point-to-point
eth2       designated forwarding  20000 128.2   point-to-point

$ sudo ovs-appctl bridge/dump-flows br0

$ sudo ovs-vsctl set Bridge br0 other_config:stp-priority=8
$ sudo ovs-vsctl set Bridge br0 other_config:stp-hello-time=2

# Lower the priority so this bridge becomes root.
```

### Sniffing BPDUs

```
$ sudo tcpdump -i any -n stp
tcpdump: data link type LINUX_SLL2
tcpdump: verbose output suppressed, use -v[v]... for full protocol decode
listening on any, link-type LINUX_SLL2 (Linux cooked v2), snapshot length 262144 bytes
12:42:13.012  In eth1 STP 802.1d, Config, Flags [none], bridge-id 8000.00:0c:29:87:a1:7e.8001, length 35
        message-age 0.00s, max-age 20.00s, hello-time 2.00s, forwarding-delay 15.00s
        root-id 8000.00:0c:29:87:a1:7e, root-pathcost 0
12:42:15.012  In eth1 STP 802.1d, Config, Flags [none], bridge-id 8000.00:0c:29:87:a1:7e.8001, length 35
        message-age 0.00s, max-age 20.00s, hello-time 2.00s, forwarding-delay 15.00s
        root-id 8000.00:0c:29:87:a1:7e, root-pathcost 0

$ sudo tcpdump -i any -n 'ether proto 0x88cc or stp'
# 0x88cc is LLDP. Combined: see neighbor discovery + STP at the same time.

$ sudo tcpdump -i any -n -e ether host 01:80:c2:00:00:00
# All BPDUs and CFM frames go to this multicast MAC.
```

### Bridge utility (legacy `brctl`)

```
$ sudo brctl show
bridge name     bridge id               STP enabled     interfaces
br0             8000.000c2987a17e       yes             eth1
                                                        eth2

$ sudo brctl showstp br0
$ sudo brctl setbridgeprio br0 0
$ sudo brctl setfd br0 15        # forward delay
$ sudo brctl setmaxage br0 20    # max age
$ sudo brctl sethello br0 2      # hello timer
$ sudo brctl stp br0 on
$ sudo brctl stp br0 off         # WARNING: now you can loop yourself
```

### `bridge` command (modern replacement)

```
$ bridge link show
2: eth1@if12: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 master br0 state forwarding priority 32 cost 100
3: eth2@if13: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 master br0 state blocking priority 32 cost 100

$ bridge fdb show br br0
00:0c:29:87:a1:7e dev br0 vlan 1 master br0 permanent
01:80:c2:00:00:00 dev eth1 self permanent
33:33:00:00:00:01 dev br0 self permanent

$ bridge vlan show
port    vlan-id
br0     1 PVID Egress Untagged
eth1    1 PVID Egress Untagged
eth2    1 PVID Egress Untagged
```

### Trigger a topology change manually

```
SwitchA# config terminal
SwitchA(config)# interface gi0/1
SwitchA(config-if)# shutdown
%LINEPROTO-5-UPDOWN: Line protocol on Interface GigabitEthernet0/1, changed state to down
%SPANTREE-6-PORT_STATE: Port Gi0/1 instance 0 moving from forwarding to blocking
%SPANTREE-6-PORT_STATE: Port Gi0/2 instance 0 moving from blocking to forwarding   # alternate becomes root!
SwitchA(config-if)# no shutdown
```

### NetworkManager

```
$ nmcli connection add type bridge ifname br0
$ nmcli connection modify br0 bridge.stp yes
$ nmcli connection modify br0 bridge.priority 4096
$ nmcli connection modify br0 bridge.forward-delay 15
$ nmcli connection modify br0 bridge.hello-time 2
$ nmcli connection modify br0 bridge.max-age 20
$ nmcli connection up br0
```

### Netplan (Ubuntu)

```yaml
network:
  version: 2
  bridges:
    br0:
      interfaces: [eth1, eth2]
      parameters:
        stp: true
        forward-delay: 15
        hello-time: 2
        max-age: 20
        priority: 4096
      addresses: [192.168.1.10/24]
```

### ifupdown2 (Cumulus, Debian)

```
auto br0
iface br0
    bridge-ports eth1 eth2
    bridge-stp on
    bridge-bridgeprio 4096
    bridge-fd 15
    bridge-hello 2
    bridge-maxage 20
    address 192.168.1.10/24
```

## A Detailed Walkthrough: Building the Tree from Scratch

Let's run through what actually happens, second by second, when you power on a brand-new network of three switches connected in a triangle.

```
   +-----+
   |  A  |  MAC = 00:00:00:00:00:01
   +--+--+
      |
   +--+--+      +-----+
   |  B  |------|  C  |
   +-----+      +-----+
   MAC = 00:00:00:00:00:02   MAC = 00:00:00:00:00:03

(Triangle: A-B, B-C, C-A. All cables plugged in. All three switches powered on simultaneously.)
```

### Second 0: Boot

All three switches power up at the same instant. Every switch starts up assuming it is the Root Bridge. Every switch starts sending Configuration BPDUs out every port.

Switch A's BPDUs say: "Root = A. Cost = 0. Sender = A."
Switch B's BPDUs say: "Root = B. Cost = 0. Sender = B."
Switch C's BPDUs say: "Root = C. Cost = 0. Sender = C."

### Second 0-2: Initial BPDU Exchange

Switch A receives BPDUs from B (claiming root B) and from C (claiming root C). Switch A compares them to its own claimed root (A). A's MAC is `00:00:00:00:00:01` which is numerically less than B's `00:00:00:00:00:02` and C's `00:00:00:00:00:03`. So A keeps believing A is root and ignores the inferior BPDUs from B and C.

Switch B receives BPDUs from A (claiming root A) and from C (claiming root C). B compares root IDs:
- Its own (B's) root claim: priority 32768 + MAC `02` = `32768.02`
- A's root claim: priority 32768 + MAC `01` = `32768.01` (lower! superior!)
- C's root claim: priority 32768 + MAC `03` = `32768.03` (higher, inferior)

B realizes A's BPDU is superior. B accepts that A is the real root. B updates its understanding: Root = A. B records the path through which it heard A's BPDU.

Switch C does the same: it receives BPDUs from A and B, sees A's is superior to its own and to B's, accepts A as root.

By second 2, all three switches agree: **A is root**.

### Second 2-4: Determining Root Ports

B has two paths to A:
- Direct via the A-B link (cost = 4 if 1G).
- Indirect via C: A-C-B (cost = 4 + 4 = 8).

B picks the shortest: the A-B link. That port becomes B's Root Port.

C similarly picks the direct A-C link as its Root Port.

### Second 4-6: Determining Designated Ports

For every link, exactly one switch is Designated.
- A-B link: Both A and B might claim Designated. A is root (cost 0), so A wins. A's port toward B is Designated. B's port toward A is Root Port.
- A-C link: Same logic. A is Designated. C's port is Root Port.
- B-C link: Neither end is root. Both are tied on cost back to root (both have cost 4 to reach A directly). Tie-break by Bridge ID. B's MAC `02` < C's MAC `03`, so B wins. B's port toward C is Designated. C's port toward B is **Alternate** (which means BLOCKED in classic STP).

### Second 6-50: State Transitions

In classic STP, after deciding roles, each port goes through state transitions:

| Time | Port              | State                       |
|------|-------------------|-----------------------------|
| 6    | A's port to B     | Listening                   |
| 6    | A's port to C     | Listening                   |
| 6    | B's port to A     | Listening                   |
| 6    | B's port to C     | Listening                   |
| 6    | C's port to A     | Listening                   |
| 6    | C's port to B     | Listening (will go Blocking)|
| 21   | All Listening     | → Learning (15s elapsed)    |
| 36   | All Learning      | → Forwarding (15s elapsed)  |
| 36   | C's port to B     | Blocking (Alternate)        |

Total time: ~36 seconds before the network is fully usable. With unlucky timing relative to Max Age (20s), this can balloon to ~50 seconds.

### Second 50: Steady State

```
   +-----+
   |  A  |  Root
   +--+--+
   D-P  D-P            (D-P = Designated Port)
   |     |
   R-P   R-P
   +-+--+ +--+--+
   | B  | |  C  |
   +-+--+ +-----+
     D-P    Alt-BLK     (Alt-BLK = Alternate Blocking)
       \    /
        \  /
       (cable still plugged in,
        but C-side port is blocked)
```

Network is forwarding. A broadcast from a host on B will go: host → B → A (via Root Port up). Then A floods to C (via Designated Port). Then C forwards to its host. The frame travels A-B and A-C only. The B-C link is silent (blocked).

If the A-B link breaks, STP reconverges. Within ~30s (classic) or <1s (RSTP), C's blocked port unblocks and the new tree is:

```
   +-----+
   |  A  |  Root
   +--+--+
        \
         \
          R-P
        +-----+
        |  C  |
        +-----+
         D-P
           \
            \
             R-P
           +-----+
           |  B  |
           +-----+
```

Same root. Different shape. Still a tree. No loops.

## A Long-Form Tour of RSTP's Speed Tricks

Why is RSTP so much faster than classic STP? Three reasons.

### Trick 1: Skip the wait if you have a backup

In classic STP, when a Root Port goes down, the switch waits for Max Age (20s) before deciding the path is dead, then transitions through Listening (15s) and Learning (15s) on the new port. That's 50 seconds.

In RSTP, switches **track Alternate Ports explicitly** as roles. When the Root Port goes down, the switch already knows which port is Alternate (it's been listening to BPDUs there all along) and can immediately promote the Alternate to Root, in milliseconds.

### Trick 2: Edge ports skip the dance entirely

If a port is connected to a host (not a switch), there is no loop risk. RSTP knows this and lets edge ports go straight to Forwarding without listening or learning. Saves 30 seconds on every host port plug-in.

### Trick 3: Proposal/Agreement handshake

When two RSTP switches connect, they exchange a Proposal/Agreement handshake that lets them both go Forwarding immediately. The "sync" step ensures no loop sneaks through during the handshake.

The handshake:

```
RSTP Switch A                         RSTP Switch B
     |                                      |
     |---- Proposal (Designated, Root=A) -->|
     |                                      |  B blocks all non-edge ports
     |                                      |  (sync), waits for them to be sure
     |                                      |  they don't form a loop.
     |                                      |
     |<---- Agreement (Root Port confirmed)|
     |                                      |
     |   ports now Forwarding               |
     |                                      |  unblocks downstream ports
     |                                      |  with their own proposals.
```

This handshake propagates outward from the root, switch by switch, in milliseconds.

### Trick 4: Direct topology change

In classic STP, a topology change is signaled via a separate TCN BPDU that must travel up to the root, get acknowledged, then flooded back down. In RSTP, topology changes are signaled directly via flags in the regular BPDU, which floods immediately throughout the network. MAC tables are flushed (using a fast aging time) within a second.

## More on MSTP — Worked Example

Imagine you have four switches A, B, C, D in a square:

```
   A----B
   |    |
   D----C
```

You have 100 VLANs. With Rapid PVST+, you'd run 100 trees — too much CPU. With single-instance RSTP, all VLANs share one tree with one blocked link, wasting half your bandwidth.

MSTP solution: define two MSTIs.
- MSTI 1: VLANs 1-50 → Root = A
- MSTI 2: VLANs 51-100 → Root = C

```
SwitchA(config)# spanning-tree mode mst
SwitchA(config)# spanning-tree mst configuration
SwitchA(config-mst)# name PROD
SwitchA(config-mst)# revision 1
SwitchA(config-mst)# instance 1 vlan 1-50
SwitchA(config-mst)# instance 2 vlan 51-100
SwitchA(config-mst)# exit
SwitchA(config)# spanning-tree mst 1 root primary
```

Apply identical config (with `instance 2 root primary` instead of `1`) on switch C. Apply the same `name PROD revision 1` and instance/VLAN mapping on B and D.

Result:

- For VLAN 1-50 (MSTI 1): tree rooted at A. Probably blocks D-C link. Traffic flows A-B-C and A-D.
- For VLAN 51-100 (MSTI 2): tree rooted at C. Probably blocks A-B link. Traffic flows C-B and C-D-A.

Both A-B and D-C links are utilized! Bandwidth doubled compared to single-instance RSTP. CPU cost: 2 trees, not 100.

That is the whole point of MSTP.

## Bridge ID Anatomy

Let's dissect a Bridge ID more carefully because it confuses everyone the first time.

Modern STP (post-2004) uses what's called **MAC reduction** or **Extended System ID**. The original 16-bit Priority field is split:

```
+-------------------+----------------+
| 4-bit priority    | 12-bit VLAN ID |
+-------------------+----------------+
       (default       (e.g., for VLAN 10,
        priority=8)    this is 0x00A=10)
```

Combined with the 48-bit MAC, the full 64-bit Bridge ID for VLAN 10 with default priority on a switch with MAC `00:1c:42:aa:bb:cc` looks like:

```
Priority field = 8 (4 bits) << 12 | 10 (VLAN ID, 12 bits)
              = 0x800A
              = 32768 + 10
              = 32778

Bridge ID = 32778 + 00:1c:42:aa:bb:cc
         = 0x800A001c42aabbcc
```

When you set `spanning-tree vlan 10 priority 4096`:
- `4096 / 4096 = 1` (the priority value, 1 << 12 = 0x1000 = 4096).
- The full priority field becomes `4096 + 10 = 4106 = 0x100A`.
- Bridge ID becomes `0x100A001c42aabbcc`.

Lower number wins, so this switch wins the election for VLAN 10.

This is why priority must be a multiple of 4096: only 4 bits are usable, so values are `0, 4096, 8192, ..., 61440`.

## A Note on Hardware: When STP Lives in ASICs vs CPU

In a modern switch, the data plane (forwarding actual frames) lives in an ASIC chip (Broadcom Trident, NVIDIA Spectrum, Cisco Silicon One). The control plane (running STP, computing trees, parsing BPDUs) lives in the switch CPU.

When a topology change happens:
- BPDU arrives → ASIC traps it to CPU → CPU runs STP state machine → CPU updates ASIC's per-port forwarding state (forwarding/blocking).

This is fast, but not instantaneous. PVST+ with 1000 VLANs means 1000 trees of BPDUs arriving every 2 seconds = 500 BPDU/s arriving for the CPU to parse. At scale, this CPU pressure is why MSTP becomes attractive.

Some switches accelerate STP further by letting the ASIC handle simple BPDU forwarding/dropping without CPU intervention, with the CPU only intervening on topology change. This is a vendor-specific optimization.

## Common Confusions

### 1. STP vs RSTP vs MSTP — what's the actual difference

- **STP** (802.1D, 1990): single tree, slow (30-50s).
- **RSTP** (802.1w, 2001): single tree, fast (<1s).
- **MSTP** (802.1s, 2002): multiple trees per region, fast.
- **PVST+** / **Rapid PVST+** (Cisco): one tree per VLAN, slow / fast respectively.

You almost never want classic STP today. Use RSTP at minimum.

### 2. PVST+ vs MSTP — when to pick which

PVST+ (and Rapid PVST+) is simpler conceptually: one tree per VLAN. It works great up to a few hundred VLANs. Beyond that, BPDU CPU cost on switches becomes a problem.

MSTP groups VLANs into instances (multiple VLANs share one tree), so 1000 VLANs in 4 instances is only 4 trees. MSTP is also the only IEEE-standard option for multi-vendor environments.

Pick MSTP if: multi-vendor, large scale (>500 VLANs), you want explicit control over which links each VLAN-group uses.

Pick Rapid PVST+ if: all-Cisco, smaller scale, simpler mental model preferred.

### 3. What does "STP region" mean

A region is a set of switches that all agree on:
- The MSTP configuration name (`name PROD`)
- The MSTP revision number (`revision 1`)
- The VLAN-to-instance mapping table (`instance 1 vlan 10,20`)

Within a region, MSTP runs IST + multiple MSTIs. Between regions, MSTP runs a single CIST that looks like RSTP from outside.

### 4. Root Bridge election — what wins

Lowest **Priority** first. If priority is tied, lowest **MAC**. The combined `Priority.MAC` value is called the **Bridge ID** and the bridge with the numerically lowest Bridge ID wins.

### 5. Forwarding vs Listening — what's the difference

Listening: port has decided it might forward, but is double-checking. Does NOT forward data, does NOT learn MACs. Lasts 15s by default.

Forwarding: port forwards data and learns MACs.

In RSTP the Listening state is gone — it's collapsed into Discarding (not forwarding, not learning).

### 6. Why disabling STP "fixes" something is dangerous

If you have a legitimate loop in your physical wiring, disabling STP will indeed make traffic flow through the redundant link — and then a single broadcast will loop forever and kill the network. Disabling STP is **never** the right answer to a connectivity problem. Find out *why* STP is blocking the port (probably for a good reason), fix the misconfiguration, do not disable the safety net.

The only exception: if the network is a pure point-to-point chain with no loops at all (rare), STP is unnecessary. But even then, leaving it on costs nothing.

### 7. PortFast and BPDU Guard go together

PortFast skips listening/learning for host ports. BPDU Guard ensures host ports stay host ports — if a switch BPDU arrives, the port is shut down. Always pair them. Setting PortFast without BPDU Guard is a loop waiting to happen the first time a user plugs a cheap home switch into a wall jack.

### 8. BPDU Guard vs BPDU Filter

BPDU Guard: receive a BPDU on this port → err-disable the port. Loud, safe.

BPDU Filter: drop received BPDUs and don't send BPDUs on this port. Silent, dangerous. The port stays up no matter what arrives. Easy to create loops with BPDU Filter.

Use BPDU Guard 99% of the time. Use BPDU Filter only in service-provider edge scenarios where you really must hide BPDUs from the customer.

### 9. Root Guard vs Loop Guard

Root Guard: protects against superior BPDUs arriving on a Designated port (downstream switch claims to be root). Puts port in **root-inconsistent** state.

Loop Guard: protects against BPDUs *stopping* on Root Port or Alternate Port (suggests unidirectional failure). Puts port in **loop-inconsistent** state.

Root Guard goes on Designated ports facing leaves. Loop Guard goes on Root Ports and Alternate Ports facing the root.

### 10. PVST+ TLV interaction with MSTP

In a mixed environment (some switches PVST+, some MSTP), MSTP encodes PVST+-compatible information in a special **PVST+ TLV** added to BPDUs, so the PVST+ side sees what looks like a per-VLAN tree. This is fragile — get all switches into one mode if possible.

### 11. Topology change flooding

When a topology change happens, the root sets the TC flag in BPDUs for `Forward Delay + Max Age` (default 35s). Every receiving switch flushes its MAC address table (using a fast aging time, 15s, instead of 300s). The point: the topology changed, so MAC locations may have changed, so we need to relearn.

In RSTP, topology changes are signaled directly in the regular BPDU, not via the separate TCN BPDU — RSTP is faster about this whole dance.

### 12. What happens when an edge port gets a BPDU

If the edge port (PortFast) does not have BPDU Guard:
- The port loses its edge status. It re-enters the normal STP state machine. It probably blocks (because the BPDU may indicate a loop). 30-50 seconds wasted.

If the edge port has BPDU Guard:
- The port goes err-disabled. Stays down until manually cleared (or err-disable recovery clears it after 5 minutes).

### 13. What is "err-disabled"

A Cisco port state. The port is down and will not come back up automatically. Caused by a violation (BPDU Guard, port-security, link-flap, UDLD, etc.). Recovery requires:

```
SwitchA(config)# errdisable recovery cause bpduguard
SwitchA(config)# errdisable recovery interval 300

# Or manually:
SwitchA# clear errdisable interface Gi0/5
SwitchA(config)# interface Gi0/5
SwitchA(config-if)# shutdown
SwitchA(config-if)# no shutdown
```

### 14. Cisco's Rapid PVST+ vs MSTP — interop

Rapid PVST+ is Cisco-proprietary. MSTP is IEEE standard. If you have a mixed-vendor network, use MSTP. If everything is Cisco, either works (though MSTP scales better at high VLAN counts).

### 15. How 802.1Q tags interact with STP

In PVST+, BPDUs are sent on each VLAN separately, **untagged on the native VLAN** and **tagged with each non-native VLAN**. In MSTP, BPDUs are sent only on the IST/CIST instance and are **untagged** (they carry MSTI information internally). In MSTP, all VLANs in an instance share that instance's tree — there are no per-VLAN tagged BPDUs.

This is a subtle but important point if you are sniffing BPDUs.

### 16. STP doesn't work over wireless mesh

Wireless mesh networks (Wi-Fi mesh, batman-adv) use entirely different protocols (HWMP, BATMAN, OLSR). STP would not work because RF links are shared, half-duplex, and have unpredictable latency. Don't try.

### 17. Can you run two Root Bridges?

No. STP elects exactly one root per spanning-tree instance. With Per-VLAN Spanning Tree (PVST+ or Rapid PVST+), each VLAN has its own root, so you can have different roots for different VLANs — but for any single VLAN there is one and only one root.

If two switches are simultaneously claiming to be root (because of partition or misconfiguration), the network has split-brained and is not a valid spanning tree.

### 18. Is STP still used in modern data centers — yes at access layer

The fabric (spine-leaf) is L3 routed. STP is not used there. But the access layer (where end hosts plug in) is still L2, still has the broadcast-storm risk, and absolutely still uses STP. Every wall jack in your office runs STP. Every server-facing switch port runs STP.

In summary: STP is dead in the fabric, alive at the edge.

### 19. What is "L2 fabric" vs spanning tree

A L2 fabric is a Layer-2 network that uses an alternative loop-prevention mechanism — typically **TRILL** (RFC 6325, IS-IS for Layer 2), **SPB** (Shortest Path Bridging, IEEE 802.1aq, 2012), or vendor-proprietary protocols like Cisco **FabricPath**. These all replace STP with an IS-IS-based shortest-path computation that allows multipath load balancing (no blocked links) while still preventing loops via TTL-like mechanisms.

L2 fabrics are rarely deployed in 2026 because spine-leaf L3 + EVPN/VXLAN has won. But they are the answer to "can we have a loop-free L2 with full multipath?" — yes, but the industry largely chose L3 instead.

## Vocabulary

| Term | Plain English |
|------|---------------|
| STP | Spanning Tree Protocol. The referee that prevents loops in switched Ethernet. |
| IEEE 802.1D | Original STP standard, 1990. Revised in 1998 and 2004. |
| IEEE 802.1w | Rapid STP (RSTP), 2001. Sub-second convergence. |
| IEEE 802.1s | Multiple STP (MSTP), 2002. Multiple trees per region. |
| IEEE 802.1Q | VLAN tagging, 1998. Adds 4-byte tag to Ethernet frames. |
| IEEE 802.1Q-2014 | Consolidated bridging spec; folded in 802.1D, 802.1w, 802.1s. |
| IEEE 802.1AX | Link Aggregation Control Protocol (LACP), 2008. |
| IEEE 802.1ad | QinQ / provider bridging, 2005. Stacked VLAN tags. |
| IEEE 802.1ah | Provider Backbone Bridging (PBB / MAC-in-MAC), 2008. |
| IEEE 802.1ag | Connectivity Fault Management (CFM), 2007. L2 ping/traceroute. |
| IEEE 802.1aq | Shortest Path Bridging (SPB), 2012. IS-IS-based L2 multipath. |
| BPDU | Bridge Protocol Data Unit. The little control message switches exchange to build the tree. |
| Configuration BPDU | The standard kind of BPDU. Carries root ID, path cost, sender ID, timers. |
| TCN BPDU | Topology Change Notification BPDU. 4-byte upstream-only message announcing a topology change. |
| Topology Change | A link or port state changed. Triggers MAC table flush across the network. |
| TC flag | Topology Change flag. Set in config BPDUs by the root for `Forward Delay + Max Age` after a topology change. |
| TCA flag | Topology Change Acknowledgment flag. Used in classic STP between switches relaying TCN BPDUs. |
| Root Bridge | The switch with the lowest Bridge ID. Becomes the root of the spanning tree. |
| Bridge ID | 16-bit Priority + 48-bit MAC = 64-bit identifier. |
| Bridge Priority | 16-bit priority field, default 32768, must be a multiple of 4096 (when Extended System ID is enabled). |
| System MAC | The 48-bit MAC address of the switch's bridge. |
| Extended System ID | 12-bit field carved out of the priority for VLAN ID. Modern STP uses this. |
| MAC reduction | The use of Extended System ID to encode VLAN ID in the priority field. |
| Root Path Cost | Cumulative path cost from a switch to the root. |
| Path Cost | Per-link cost based on link speed. |
| Sender Bridge ID | The Bridge ID of the switch that just sent this BPDU. |
| Sender Port ID | The port priority + port number from which the BPDU was sent. |
| Message Age | How long since the root sent this BPDU (in 1/256 sec units). |
| Max Age | When to discard a BPDU. Default 20s. |
| Hello Time | How often BPDUs are sent. Default 2s. |
| Forward Delay | Per-state timer. Default 15s. |
| Root Port (RP) | A non-root switch's port with the lowest cost path to the root. Forwards. |
| Designated Port (DP) | The single port on each link that is allowed to forward toward the leaves. |
| Alternate Port | A blocked port that has another path back to the root. Backup to Root Port. |
| Backup Port | A blocked port on the same switch as a Designated Port (rare). |
| Disabled Port | An administratively shut-down port. Not in STP at all. |
| Edge Port | A port connected to a host (not a switch). Skips listening/learning. Same as PortFast. |
| Point-to-point link | Full-duplex link between two switches. Allows fast Proposal/Agreement. |
| Shared link | Half-duplex link (hub). Falls back to classic timers. |
| Half-duplex link | A link where only one side can transmit at a time. |
| Proposal | An RSTP request to make a link forwarding. Sent by the side closer to root. |
| Agreement | An RSTP response confirming the Proposal. |
| Sync | The brief step where a switch puts other ports into Discarding before agreeing. |
| Port Priority | 8-bit priority field for a port, used as tiebreaker. Default 128. |
| Port ID | Port priority + port number combined. |
| IEEE-recommended path cost (short) | 100 / 19 / 4 / 2 for 10M / 100M / 1G / 10G. |
| IEEE-recommended path cost (long) | 2000000 / 200000 / 20000 / 2000 / 200 / 20 for 10M / 100M / 1G / 10G / 100G / 1T. |
| Long path cost | 32-bit cost field. Required for distinguishing 10G+ links. |
| Short path cost | Legacy 16-bit cost field. |
| BPDU Guard | Err-disables a port if a BPDU is received on a PortFast port. |
| BPDU Filter | Drops sent and received BPDUs on a port. Dangerous; used at SP edges. |
| Root Guard | Puts a port in root-inconsistent state if a superior BPDU arrives on a Designated port. |
| Loop Guard | Puts a port in loop-inconsistent state if BPDUs stop arriving on a Root or Alternate port. |
| BPDU Skew Detection | Cisco feature that warns when BPDUs are too late. |
| UDLD | Unidirectional Link Detection. Detects fiber transmit failures via L2 keepalives. |
| DPP | Dispute mechanism. Detects role-conflict via TLV in RSTP/MSTP BPDUs. |
| PortFast | Cisco feature that skips listening/learning on edge ports. Now standard as "edge port." |
| UplinkFast | Legacy Cisco feature that fast-fails over Root Port to Alternate. Obsoleted by RSTP. |
| BackboneFast | Legacy Cisco feature that speeds up indirect-failure response via RLQ. Obsoleted by RSTP. |
| RLQ | Root Link Query. BackboneFast's special protocol message. |
| MSTP region | A set of switches sharing the same MSTP config name, revision, and VLAN-instance mapping. |
| MSTP digest | MD5 hash of the VLAN-to-instance mapping. Switches with different digests are in different regions. |
| MSTP revision number | Integer used to version the MSTP region's configuration. |
| IST (Internal Spanning Tree) | The spanning tree for instance 0 inside an MSTP region. |
| MSTI | MST Instance. A separate spanning tree (1 to 4094) within an MSTP region. |
| MST Configuration Name | Identifies the MSTP region. Free-form string. |
| CIST | Common and Internal Spanning Tree. The tree visible outside MSTP regions. Looks like RSTP. |
| CST | Common Spanning Tree. The legacy single tree if MSTP is not used. |
| Bridge Priority (16-bit) | Default 32768. With MAC reduction, only 4 bits are usable (multiples of 4096). |
| Designated Bridge | The switch whose Designated Port is on a given segment. |
| Internal Path Cost | Path cost inside an MSTP region. |
| External Path Cost | Path cost between MSTP regions. |
| Port Path Cost | Per-port-configurable cost. Defaults from link speed. |
| Link Aggregation | Bundling multiple physical links into one logical link. STP sees the bundle as a single link. |
| MLAG / MC-LAG | Multi-Chassis Link Aggregation. Two switches act as one for LAG purposes. STP sees them as one switch. |
| vPC | Cisco's MLAG implementation on Nexus. |
| vPC peer-link | The link between the two vPC peer switches that synchronizes state. |
| vPC keepalive | Out-of-band heartbeat between vPC peers. |
| EtherChannel | Cisco's term for link aggregation. |
| LACP | Link Aggregation Control Protocol (IEEE 802.1AX-2008). |
| LACPDU | LACP Data Unit. The control message LACP uses. |
| Slow Protocols | A class of L2 protocols including LACP and OAM. |
| Marker | Slow Protocols message used in LACP for graceful failover. |
| Marker Response | The reply to a Marker. |
| EAPOL | Extensible Authentication Protocol over LAN. Used for 802.1X. |
| ARP | Address Resolution Protocol. Maps IP to MAC. Sent as Ethernet broadcast. |
| Gratuitous ARP | An ARP "announcement" sent by a host to itself. Used in failover. |
| Broadcast Storm | When broadcast frames loop and multiply, saturating links. |
| Multicast Storm | Same idea, with multicast frames. |
| MAC table | A switch's table of MAC-to-port mappings. |
| MAC table overflow | When more MACs are seen than the table can hold; switch becomes a hub. |
| MAC flapping | When a single MAC keeps appearing on different ports rapidly. Sign of a loop. |
| MAC move detection | Feature that flags rapid MAC moves. |
| ELS | Ethernet Link State (NX-OS feature). Tracks link state for protocols. |
| ETS | Enhanced Transmission Selection (802.1Qaz). Bandwidth allocation per traffic class. |
| 802.1Qbb | Priority Flow Control (PFC). Pause specific traffic classes. |
| 802.1Qaz | DCBX + ETS. Data Center Bridging Capability Exchange. |
| EVB | Edge Virtual Bridging. Virtual switch / hypervisor integration. |
| VEPA | Virtual Ethernet Port Aggregator. EVB feature for hypervisor traffic to flow through external switch. |
| VN-Tag | Cisco proprietary tag for virtual NIC identification. |
| FabricPath | Cisco proprietary L2 multipath, IS-IS-based. |
| TRILL | Transparent Interconnection of Lots of Links (RFC 6325). IETF L2 multipath. |
| SPB | Shortest Path Bridging (802.1aq). IEEE L2 multipath. |
| SPBM | SPB MAC mode. Encapsulates with MAC-in-MAC (PBB). |
| SPBV | SPB VLAN mode. Encapsulates with QinQ. |
| L2GP | Layer 2 Gateway Protocol. Cisco-proprietary STP variant. |
| Root Inconsistent | Port state when Root Guard is active. Up but not forwarding. |
| Type-Inconsistent | Port state when STP modes mismatch (e.g., PVST+ vs MSTP). |
| MSTP-region-incompatible | Two switches think they're in different MSTP regions. |
| ovs-vsctl | Open vSwitch's main CLI tool. |
| mstpd | Linux user-space STP/RSTP/MSTP daemon. |
| mstpctl | CLI tool to control mstpd. |
| ifupdown2 | Modern Debian/Cumulus interface management. Supports STP options inline. |
| netplan | Ubuntu's YAML-based network config. Supports STP via parameters. |
| NetworkManager | Desktop Linux network manager. Supports bridges with STP via nmcli. |
| brctl | Legacy Linux bridge CLI tool (replaced by `bridge` and `ip`). |
| bridge link | Modern Linux command to show bridge port info. |
| bridge fdb | Modern Linux command to show forwarding database (MAC table). |
| bridge vlan | Modern Linux command to show per-port VLAN info. |
| Network namespace bridge | A Linux network namespace's own bridge instance, isolated from the host's. |
| Container networking | Containers (Docker, Kubernetes) use bridges; STP is usually OFF by default. |
| Docker bridge | Default Docker network type. STP off by default. |
| Cumulus Linux | NVIDIA's Linux-based switch OS. Uses mstpd for STP. |
| NVIDIA SONiC | Open-source switch OS. STP via FRR mstpd. |
| FRR | Free Range Routing. Routing daemon suite that includes mstpd. |
| OpenSwitch | An open-source switch OS (less common today). |
| OpenWrt | Linux for home routers. Bridges with optional STP. |
| batman-adv | Wireless mesh protocol. Different from STP, lives at L2.5. |
| TSN | Time-Sensitive Networking. Set of IEEE 802.1 standards for deterministic latency. |
| 802.1Qci | Per-Stream Filtering and Policing. TSN feature. |
| Frame | An Ethernet packet (the L2 PDU). |
| MAC address | 48-bit hardware address on every Ethernet NIC. |
| Multicast | A frame addressed to a group, like `01:80:c2:00:00:00` (BPDU). |
| Flooding | Forwarding a frame out every port except the one it came in on. Default behavior for unknown unicast and broadcasts. |
| Hub | A dumb device that floods every frame out every port. Replaced by switches in 1990s. |
| Switch | A bridge with many ports. Learns MAC addresses, forwards intelligently. |
| Bridge | The L2 device that became the modern switch. STP comes from "bridge protocol." |
| Trunk port | A port carrying multiple VLANs (typically 802.1Q-tagged). |
| Access port | A port belonging to a single VLAN (typically untagged). |
| Native VLAN | The untagged VLAN on a trunk. Default 1. |
| VLAN | Virtual LAN. Logical broadcast domain on shared physical switches. |
| VLAN ID | 12-bit identifier (1-4094). |
| 802.1Q tag | 4-byte tag inserted into Ethernet frame to identify VLAN. |
| QinQ | Stacking two 802.1Q tags (provider over customer). 802.1ad. |
| Provider Bridge | A switch that supports QinQ for service providers. |
| Customer VLAN | The "inner" tag in QinQ. |
| Service VLAN | The "outer" tag in QinQ. |
| Port-channel | Cisco's term for a logical link aggregation interface. |
| LAG | Link Aggregation Group. Generic term. |
| LACP active | LACP mode that initiates negotiation. |
| LACP passive | LACP mode that responds to negotiation but does not initiate. |
| Static LAG | Aggregation without LACP. Less safe. |
| Convergence | The process of the network agreeing on a stable topology. |
| Convergence time | How long it takes after a change for the network to be fully converged. |
| Sub-second convergence | Convergence in < 1 second. RSTP achieves this. |
| Topology change flooding | The process of broadcasting topology change notifications. |
| Aging time | How long an idle MAC address stays in the table before being evicted. Default 300s. |
| MAC table flush | Deleting all dynamic MAC entries. Triggered by topology changes. |
| Ethernet broadcast | A frame addressed to `ff:ff:ff:ff:ff:ff`. Floods to every port. |
| Multicast MAC | A MAC starting with an odd first octet. `01:80:c2:00:00:00` is the BPDU multicast. |
| BPDU multicast address | `01:80:c2:00:00:00`. Used by all STP/RSTP/MSTP BPDUs. |
| LLC header | Logical Link Control header. STP BPDUs use LLC encapsulation. |
| SNAP header | Subnetwork Access Protocol header. Used for some BPDU encapsulation variants. |
| STP timer (Hello) | Default 2 seconds. How often BPDUs sent. |
| STP timer (Forward Delay) | Default 15 seconds. Per-state timer in classic STP. |
| STP timer (Max Age) | Default 20 seconds. When BPDUs expire if not refreshed. |
| Tx Hold Count | Maximum BPDUs sent per Hello. Default 6. |
| Max Hops | MSTP TTL-equivalent for BPDUs. Default 20. |
| Inferior BPDU | A BPDU claiming a worse root than what's currently believed. |
| Superior BPDU | A BPDU claiming a better root than what's currently believed. |
| Root Bridge MAC | The MAC of the elected root. |
| Root Bridge ID | Priority + MAC of the elected root. |
| Designated Bridge ID | Priority + MAC of the designated bridge for a segment. |
| Designated Port ID | Port priority + port number on the designated bridge. |
| Switch fabric | The internal forwarding plane of a switch. |
| Crossbar fabric | A type of switch fabric using crossbar switches. |
| ASIC | Application-Specific Integrated Circuit. The forwarding chip in a switch. |
| Fabric ASIC | The ASIC that implements the switch's data plane. |
| Trident / Tomahawk | Broadcom's switch ASIC families. |
| Spectrum / Mellanox | NVIDIA's (formerly Mellanox's) switch ASIC families. |
| Cisco Silicon One | Cisco's modern switch ASIC family. |
| L2 fabric | A Layer-2 network using TRILL/SPB/FabricPath instead of STP. |
| L3 fabric | A Layer-3 routed network (no STP, ECMP-routed). |
| Spine-Leaf | A two-tier L3 fabric topology common in modern data centers. |
| ECMP | Equal-Cost Multi-Path. IP routing's answer to multipath. |
| EVPN | Ethernet VPN. BGP-based L2 overlay. Often paired with VXLAN underlay. |
| VXLAN | Virtual eXtensible LAN. UDP-encapsulated L2 over L3. |
| VTEP | VXLAN Tunnel End Point. The encap/decap device. |
| Underlay | The IP-routed transport network beneath an overlay. |
| Overlay | A virtual network running on top of an underlay. |
| The Algorhyme | Radia Perlman's poem about her spanning-tree algorithm (1985). |
| Radia Perlman | Inventor of STP. "Mother of the Internet." |
| 802.1D-2004 | The current STP/RSTP standard (when MSTP was folded in). |
| 802.1Q-2018 | Modern consolidated bridging spec. Includes STP/RSTP/MSTP/802.1Q. |
| Cisco STP Cookbook | A 2005 book by Cisco engineers on real-world STP. |
| Russ White | Networking author. Wrote about STP in *The Art of Network Architecture*. |
| Ethan Banks | Networking podcaster (Packet Pushers). Co-author with Russ on networking books. |

## Real-World Anti-Patterns

Things people actually do that cause outages.

### 1. Plugging a "spare" cable between two switches "just in case"

You have a working network with two switches connected by one cable. You think "let me plug another cable between them for redundancy." You plug it in. You did NOT enable LACP or any link aggregation. You just plugged a second cable in.

Now there's a loop. STP will detect it and block one of the two cables. From a forwarding standpoint nothing useful happened. From a maintenance standpoint, you now have an unused cable that everyone forgets exists.

The right answer: configure both ports as part of a port-channel (Cisco) / LAG (generic) with LACP. Then they form a single logical link, both cables forward, no loop, full bandwidth.

### 2. Disabling STP because "it was blocking my port"

Yes, it was blocking your port. For a reason. The reason is that you have a loop. If you turn STP off, the loop will instantly become a broadcast storm. Do not do this. Find out why STP is blocking. Fix that.

### 3. Using a hub at a desk

Modern devices generally don't use hubs, but occasionally you'll find a forgotten hub. A hub broadcasts every frame out every port and operates at half-duplex. STP can run across hubs but the timers fall back to slow ones (no Proposal/Agreement). Worse, half-duplex means **collisions**, which are the original Ethernet failure mode and which can absolutely happen on a hub.

The right answer: replace the hub with a switch. Hubs have no business existing in 2026.

### 4. Misnumbered VLAN-to-Instance maps in MSTP

Switch A says: `instance 1 vlan 10,20`. Switch B says: `instance 1 vlan 10,30`.

Because the VLAN-to-instance mapping is different, the MSTP digest differs. The two switches are now in different MSTP regions even though `name` and `revision` match. MSTP between them collapses to CIST. Some VLANs may not converge correctly.

Always commit the same config to all switches in the region.

### 5. Skipping the revision bump when you change the config

You change `instance 1 vlan 10,20` to `instance 1 vlan 10,20,30` on all switches but you don't bump the revision number. Now STP is partially confused: the digests match, but the configs differ. (Actually, since the digests are computed from the actual mapping, the digests will differ if the mappings differ, so bumping `revision` is more about human-bookkeeping than protocol correctness — but you should do it anyway.)

### 6. Forgetting BPDU Guard on access ports

A user plugs a Linksys home router into a wall jack, with the router's WAN port left dangling and its LAN ports connected to other things in the office. Now there's a loop and a duplicate DHCP server. Without BPDU Guard, STP may eventually figure it out (and block the user's port) — or may not, if the rogue switch is misbehaving. With BPDU Guard, the user's port goes err-disabled within milliseconds and the problem is contained.

### 7. Native VLAN mismatch on a trunk

Trunk port on switch A has native VLAN 1. Trunk port on switch B has native VLAN 99. Cisco switches will detect this via CDP and complain:

```
%CDP-4-NATIVE_VLAN_MISMATCH: Native VLAN mismatch discovered on GigabitEthernet0/24 (1), with SwitchB GigabitEthernet0/24 (99).
```

But traffic on the native VLAN of one side will leak into the other side's native VLAN. This is sometimes a security concern (VLAN-hopping attack vector) and always a debugging nightmare. Match native VLANs on both ends.

### 8. Using VLAN 1 for everything

VLAN 1 is the default native VLAN, the default VLAN for un-configured ports, the default VLAN for management traffic, and the default VLAN for a lot of control-plane chatter (CDP, VTP, DTP, PAgP). Using it for production data is asking for cross-contamination. Move user traffic to a real VLAN (e.g., 10, 20).

### 9. PVST+ ↔ MSTP without explicit interop

Cisco supports cross-mode operation but only with specific config (the PVST+ TLV in MSTP BPDUs). If you just plug a PVST+ switch into an MSTP switch and hope for the best, you may find some VLANs converge and some don't. Pick a mode network-wide.

### 10. Ignoring `show spanning-tree inconsistentports`

This command is gold. If anything is in a "wrong" state — root-inconsistent, loop-inconsistent, type-inconsistent, region-mismatched — it shows up here. Run it routinely. If it's empty, you're good. If it's not, you have a problem to investigate.

```
SwitchA# show spanning-tree inconsistentports
Name                 Interface            Inconsistency
-------------------- -------------------- ------------------
VLAN0010             GigabitEthernet0/5   Root Inconsistent
VLAN0020             GigabitEthernet0/6   Loop Inconsistent
Number of inconsistent ports (segments) in the system : 2
```

## Cisco-Specific STP Features You Should Know

These are not all standard. Some are Cisco extensions. They are useful enough that you should know them by name.

### EtherChannel Misconfig Guard

Detects when one side of a link bundle has channeling enabled and the other doesn't. Without this, you'd silently form half a port-channel and half not, leading to flapping forwarding.

```
SwitchA(config)# spanning-tree etherchannel guard misconfig
```

### UDLD (Unidirectional Link Detection)

Detects when fiber transmit fails but receive works (a one-way link). Without UDLD, the receiving side keeps assuming the link is fine. With UDLD, the link is err-disabled.

```
SwitchA(config)# udld enable                 # Enables UDLD globally in normal mode.
SwitchA(config)# udld aggressive             # More paranoid; err-disables on any UDLD timeout.
SwitchA(config-if)# udld port aggressive
```

UDLD is layered on top of STP — it does not replace it but complements it. Loop Guard and UDLD together protect against unidirectional failures.

### Dispute Mechanism (DPP)

RSTP/MSTP includes a "dispute" mechanism: if a switch sees a BPDU on a port indicating that the neighbor thinks the local port is Designated, but locally the port has been designated, the protocol disputes — both sides put the port in Discarding to break a potential loop. This is a behind-the-scenes self-protection.

### Backbone STP

A Cisco term for the spanning tree at the data center core. Usually refers to the MSTP region encompassing the core switches.

### Pseudo-Information (Pseudo-Info)

Cisco's term for STP state in a vPC peer-group. Two vPC peers share STP state via the peer-link, presenting themselves as a single bridge to STP.

## Vendor Comparisons

| Vendor   | Default Mode   | Recommended Mode  | Notes                                    |
|----------|----------------|-------------------|------------------------------------------|
| Cisco    | PVST+          | Rapid PVST+ or MSTP | Cisco-only environments default to RPVST+. |
| Juniper  | RSTP           | RSTP or MSTP      | Junos calls MSTP "MSTP" too.             |
| Arista   | MSTP           | MSTP              | Arista defaults to MSTP across all VLANs. |
| Aruba    | MSTP           | MSTP              | HPE/Aruba defaults aligned with IEEE.    |
| Extreme  | MSTP           | MSTP              | Long history of multi-vendor MSTP.       |
| Cumulus  | MSTP (mstpd)   | MSTP              | Linux-based; mstpd does the heavy lift.  |
| OVS      | none           | RSTP              | Must explicitly enable.                  |
| Mikrotik | MSTP           | MSTP              | RouterOS supports STP/RSTP/MSTP.         |

If you connect a Cisco switch to a Juniper switch, set both sides to MSTP for clean interop. PVST+ → Juniper requires special tunneling.

## Try This

Hands-on experiments. Each has a setup, a thing to do, and a thing to observe.

### 1. Build a 3-switch loop in Mininet/OVS

Install Mininet:

```
$ sudo apt install mininet openvswitch-switch
```

Create a triangle topology:

```python
# triangle.py
from mininet.topo import Topo
from mininet.net import Mininet
from mininet.node import OVSSwitch
from mininet.cli import CLI

class TriangleTopo(Topo):
    def build(self):
        s1 = self.addSwitch('s1', cls=OVSSwitch)
        s2 = self.addSwitch('s2', cls=OVSSwitch)
        s3 = self.addSwitch('s3', cls=OVSSwitch)
        h1 = self.addHost('h1')
        h2 = self.addHost('h2')
        self.addLink(h1, s1)
        self.addLink(h2, s2)
        self.addLink(s1, s2)
        self.addLink(s2, s3)
        self.addLink(s3, s1)

topos = {'triangle': TriangleTopo}
```

```
$ sudo mn --custom triangle.py --topo triangle --switch ovsk
*** Creating network
*** Adding hosts:
h1 h2
*** Adding switches:
s1 s2 s3
*** Adding links:
(h1, s1) (h2, s2) (s1, s2) (s2, s3) (s3, s1)
*** Starting controller
*** Starting 3 switches
s1 s2 s3 ...
```

### 2. Watch a broadcast storm without STP

In the Mininet CLI:

```
mininet> sh ovs-vsctl set Bridge s1 stp_enable=false
mininet> sh ovs-vsctl set Bridge s2 stp_enable=false
mininet> sh ovs-vsctl set Bridge s3 stp_enable=false
mininet> h1 ping h2
```

Watch htop in another terminal. CPU goes to 100% within 2 seconds. Ping latencies skyrocket. Welcome to a broadcast storm.

Stop the storm:

```
mininet> sh ovs-vsctl set Bridge s1 stp_enable=true
mininet> sh ovs-vsctl set Bridge s2 stp_enable=true
mininet> sh ovs-vsctl set Bridge s3 stp_enable=true
```

Within 30-50 seconds (classic STP), the storm subsides as the tree converges and one link goes blocking.

### 3. Enable RSTP and observe convergence

```
mininet> sh ovs-vsctl set Bridge s1 rstp_enable=true stp_enable=false
mininet> sh ovs-vsctl set Bridge s2 rstp_enable=true stp_enable=false
mininet> sh ovs-vsctl set Bridge s3 rstp_enable=true stp_enable=false
mininet> sh ovs-appctl rstp/show s1
```

Now break a link:

```
mininet> link s1 s2 down
mininet> h1 ping h2
```

Pings resume in <1 second. Bring it back:

```
mininet> link s1 s2 up
```

Same pattern: <1s convergence.

### 4. Change priority to elect a different root

```
mininet> sh ovs-vsctl set Bridge s1 other_config:rstp-priority=4096
mininet> sh ovs-appctl rstp/show s1
```

s1 is now root. Verify with:

```
mininet> sh ovs-appctl rstp/show s2
mininet> sh ovs-appctl rstp/show s3
```

Both s2 and s3 should report s1 as the root.

### 5. Plug a hub between two switches (simulated)

OVS doesn't have hubs, but you can simulate by enabling broadcast flooding aggressively or by using a real Linux bridge in legacy mode. Watch how STP behaves on shared media (it falls back to slower timers).

### 6. Induce a topology change and capture TCN BPDUs

In one terminal:

```
$ sudo tcpdump -i s1-eth1 -n stp -v
```

In Mininet:

```
mininet> link s1 s2 down
mininet> link s1 s2 up
```

Watch tcpdump for BPDUs with the TC flag set. Note that in RSTP these are inline in the regular config BPDU, not separate TCN frames.

### 7. Configure MSTP regions on Linux mstpd

```
$ sudo apt install mstpd
$ sudo systemctl start mstpd
$ sudo mstpctl setforcevers br0 mstp
$ sudo mstpctl setmstpconfigid br0 PROD 1
$ sudo mstpctl createmsti br0 1
$ sudo mstpctl setvid2msti br0 10 1
$ sudo mstpctl setvid2msti br0 20 1
$ sudo mstpctl showmstconfid br0
```

### 8. Trigger BPDU Guard

On a Cisco switch:

```
SwitchA(config)# interface gi0/5
SwitchA(config-if)# spanning-tree portfast
SwitchA(config-if)# spanning-tree bpduguard enable
SwitchA(config-if)# end
```

Now plug another switch into Gi0/5. Watch:

```
%SPANTREE-2-BLOCK_BPDUGUARD: Received BPDU on port GigabitEthernet0/5 with BPDU Guard enabled. Disabling port.
%PM-4-ERR_DISABLE: bpduguard error detected on Gi0/5, putting Gi0/5 in err-disable state
```

Recover:

```
SwitchA# clear errdisable interface Gi0/5
SwitchA# config terminal
SwitchA(config)# interface Gi0/5
SwitchA(config-if)# shutdown
SwitchA(config-if)# no shutdown
```

### 9. Monitor MAC table flushing on topology change

```
SwitchA# show mac address-table dynamic | count
17 dynamic entries

SwitchA# show mac address-table aging-time
Vlan    Aging Time
----    ----------
 1       300

# Trigger a topology change.

SwitchA# show mac address-table aging-time
Vlan    Aging Time
----    ----------
 1       15        # <-- temporarily reduced to 15s for the topology-change interval

SwitchA# show mac address-table dynamic | count
2 dynamic entries
```

### 10. Verify Root Guard

On a Designated port (toward leaves):

```
SwitchA(config)# interface gi0/5
SwitchA(config-if)# spanning-tree guard root
```

Now misconfigure a downstream switch with priority 0 (claiming to be root). Watch:

```
%SPANTREE-2-ROOTGUARD_BLOCK: Root guard blocking port Gi0/5

SwitchA# show spanning-tree inconsistentports
Name                 Interface            Inconsistency
-------------------- -------------------- ---------------
VLAN0010             Gi0/5                Root Inconsistent
```

Fix the downstream switch's priority. Watch:

```
%SPANTREE-2-ROOTGUARD_UNBLOCK: Root guard unblocking port Gi0/5
```

### 11. Compare classic STP and RSTP convergence times empirically

Use a stopwatch (or a script that pings and logs). Switch your testbed between modes:

```
SwitchA(config)# spanning-tree mode pvst
# Test: how long after `shutdown` of root link before pings resume?

SwitchA(config)# spanning-tree mode rapid-pvst
# Test again.
```

You should see ~30s vs <1s difference.

## A Long-Form FAQ

Things people ask about STP all the time.

### Q: How do I know which switch is the root in my network?

```
SwitchA# show spanning-tree root
                                        Root    Hello Max Fwd
Vlan                   Root ID          Cost    Time  Age Dly  Root Port
---------------- -------------------- --------- ----- --- --- ---------
VLAN0001         32769 0050.7966.6800     19      2   20  15  Gi0/1
```

The "Root ID" column tells you. The MAC `0050.7966.6800` is the MAC of the root bridge.

If the local switch IS the root, the output says so:

```
SwitchA# show spanning-tree
VLAN0001
  Spanning tree enabled protocol rstp
  Root ID    Priority    32769
             Address     0050.7966.6900
             This bridge is the root
             ...
```

### Q: How do I make a specific switch the root?

```
SwitchA(config)# spanning-tree vlan 1 root primary
```

This is a Cisco macro. It sets the priority to whatever value (lower than every current priority in the network) is needed to win. Usually `4096` or `8192`.

### Q: Should I touch the timers (Hello, Max Age, Forward Delay)?

99% of the time: NO. The defaults work everywhere. The defaults are tuned for tens of switches and links across geographic spans. The only reason to change them is if you have very large diameter networks (spanning more than 7 switches between any two points) or if you need to interoperate with extremely old gear. Changing timers without understanding them is a great way to break STP convergence.

### Q: What happens if two roots get elected simultaneously?

Two switches simultaneously claim to be root → BPDUs from each propagate → switches between them eventually receive both root claims → use the lowest-bridge-ID rule → one root "wins" at each switch → eventually consistent. There is no permanent two-root state in a single connected network. You may see brief flapping during convergence.

If the network is partitioned (split brain), each partition will independently elect its own root. When the partition heals, the lower-priority winner takes over both halves.

### Q: Do I need STP if I use LACP / port-channel?

LACP bundles multiple physical links into one logical link. STP sees the bundle as one link. You still run STP on top. STP and LACP are complementary, not alternatives.

### Q: Can I run STP and LACP at the same time?

Yes. They are layered. LACP is L2 (slow protocols, sent on the same MAC `01:80:c2:00:00:02`), STP is L2 (BPDUs on `01:80:c2:00:00:00`). They coexist and are designed to.

### Q: Does STP run over Wi-Fi?

No. APs and Wi-Fi clients do not run STP. APs are typically connected via Ethernet to the wired network where STP runs. The Wi-Fi side of an AP is a different beast (uses 802.11 mechanisms, not bridging).

If you mesh Wi-Fi APs (mesh networks), they use protocols like 802.11s HWMP or BATMAN, not STP.

### Q: What about VPLS / EVPN over MPLS?

These are L2 VPN technologies that can extend Ethernet over a service provider network. Inside the SP, MPLS forwarding handles loop avoidance via TTL and label stacks. At the edge (CE) where the customer connects, STP runs locally on the customer's switches. The SP usually does not run STP across its core.

### Q: I see BPDUs on my management network. Should I worry?

Yes and no. BPDUs are normal on any L2 link between two STP-enabled switches. If your management network is a flat L2 segment with multiple switches, you should expect BPDUs. If you see BPDUs from a switch you don't own (a rogue), worry. If you see BPDUs that suddenly include a new bridge ID, investigate.

### Q: What if I have one switch with 20 ports — is STP doing anything?

Yes. STP is enabled by default. If you only have one switch with no loops, STP is mostly idle (sending BPDUs out every port every 2 seconds, but no decisions are needed). If you plug a second switch into a port that already has another switch behind it through some indirect path, STP catches the loop.

### Q: What's the difference between a "bridge" and a "switch"?

Historically, a "bridge" was a 2-port or few-port device that bridged two LAN segments together. A "switch" was a many-port device with hardware acceleration. In practice, the words are now interchangeable. STP standards still use "bridge" because that's what they were called in 1985. Every modern switch is a multi-port bridge.

### Q: Can I use STP with stretched L2 between data centers?

You can, but you should not. Stretched L2 between DCs causes split-brain risk if the inter-DC link fails. Use VXLAN/EVPN or other L2 VPN with explicit per-DC fault domains instead. STP across a WAN is asking for trouble.

### Q: Why is my switch sending so many TCN BPDUs?

A common cause: a flapping host port (e.g., a user's laptop power-cycling, or a desk phone with bad cabling). Every up/down generates a topology change and (without PortFast) a TCN. Solutions:
- Enable PortFast on all access ports.
- Fix the flapping cable / device.

### Q: How do I see a complete history of STP events?

```
SwitchA# show logging | include SPANTREE
%SPANTREE-6-PORT_STATE: Port Gi0/1 instance 0 moving from forwarding to blocking
%SPANTREE-6-PORT_STATE: Port Gi0/1 instance 0 moving from blocking to listening
... (and so on)

SwitchA# show spanning-tree detail | include from
        Number of topology changes 7 last change occurred 0:30:14 ago
                from GigabitEthernet0/1
```

The "from" line tells you which port last triggered a topology change.

### Q: Should I ever run multiple STPs simultaneously?

Per VLAN, yes (with PVST+) — but those are different *trees*, all under the same protocol family. You should not mix protocol families (e.g., classic STP and RSTP) on the same network if you can avoid it. Pick one. RSTP is backward-compatible with STP, so a mixed environment will fall back to classic STP timers.

### Q: How big can an STP network get?

The maximum diameter (longest path between any two switches) is 7 switches by default (constrained by the timers — Max Age = 20, Hello = 2, BPDUs lose 1 unit of Message Age per hop). You can extend this by tweaking timers, but rarely needed in practice. Modern designs use L3 fabrics for large scale anyway.

### Q: What's "STP suppression"?

Cisco term for a feature that suppresses BPDU sending on certain access ports (usually paired with VTP pruning and other optimizations). Rarely used.

### Q: What's the difference between a "BPDU" and a "Hello packet"?

In OSPF and other routing protocols, "Hello packet" is the keepalive. In STP, the equivalent is the Configuration BPDU sent every Hello Time (default 2s). They serve similar roles (keepalive + state announcement) but operate at different layers (BPDU is L2, OSPF Hello is L3).

### Q: I disabled STP and the network seems fine — why bother re-enabling it?

You don't currently have a loop. Sometime later, a colleague or contractor or you-yourself will plug an extra cable or connect a misconfigured switch. The instant a loop forms, the network dies. STP is insurance. The cost of running it is essentially zero. Always run it.

### Q: How do I troubleshoot a slow-converging RSTP network?

1. Check `show spanning-tree summary` for mode (should be `rapid-pvst` or `mst`, not classic `pvst` or `stp`).
2. Check `show spanning-tree interface gi0/1 detail` for "Link type" — should be `point-to-point`. If it says `shared`, force point-to-point: `spanning-tree link-type point-to-point`.
3. Check that all neighbors are also running RSTP/MSTP (not classic STP — interop falls back to slow timers).
4. Check `show spanning-tree detail` for "Number of topology changes" — high count suggests something is flapping.

### Q: My port shows as "blocking" but I'm pretty sure I have only one path. What gives?

Check `show spanning-tree interface gi0/1 detail`. The output will say *why* the port is blocking. Common reasons:
- It's an Alternate Port (you do have a redundant path you didn't realize).
- BPDU Guard or Root Guard tripped (different output, says "root inconsistent" or "err-disabled").
- The port has lower priority than another path (you can manually set port priority to flip it).

### Q: How can I verify the path BPDUs travel?

```
SwitchA# show spanning-tree detail | include from
        Number of topology changes 3 last change occurred 0:00:42 ago
                from GigabitEthernet0/1
```

And on the upstream switch:

```
SwitchUpstream# show spanning-tree detail
... ind. of which port the BPDUs go to ...
```

Combined with `tcpdump` you can trace BPDU paths.

### Q: Can I run STP only on some ports of a switch?

You can disable STP per-VLAN (`no spanning-tree vlan 100`) or per-port-direction (BPDU Filter), but not really "per port" cleanly. Avoid mixing STP-enabled and STP-disabled segments in the same broadcast domain.

### Q: What's "STP toolkit" mean?

Generic term for the collection of features on top of basic STP: PortFast, BPDU Guard, BPDU Filter, Root Guard, Loop Guard, UDLD, EtherChannel Misconfig Guard. Usually applied as a default template on access ports.

### Q: Is there a "best practices" config?

Yes:
- `spanning-tree mode rapid-pvst` (or `mst` for large networks).
- `spanning-tree portfast default` on access ports (enables PortFast everywhere).
- `spanning-tree portfast bpduguard default` (enables BPDU Guard with PortFast).
- `spanning-tree extend system-id` (enabled by default on modern Cisco).
- `spanning-tree pathcost method long` (modern cost values).
- Set explicit `spanning-tree vlan X root primary` and `root secondary` on chosen switches.
- Enable `udld enable` and `udld aggressive` on fiber links.
- Enable `errdisable recovery cause bpduguard` and `errdisable recovery interval 300` so BPDU Guard violations recover automatically after 5 minutes.

## Where to Go Next

- `ramp-up/spine-leaf-eli5` — modern data center fabric design, no STP in the fabric.
- `ramp-up/ip-eli5` — what IP and routing are.
- `networking/ethernet` — deeper Ethernet protocol details.
- `networking/lacp` — link aggregation and how it interacts with STP.
- `networking/lldp` — neighbor discovery, often used alongside STP.
- `networking/bridge` — Linux bridges in detail.
- `networking/private-vlans` — extending VLAN concepts.
- `networking/q-in-q` — stacked VLAN tags for service provider edges.
- `networking/macvlan` — alternate L2 isolation in Linux.
- `networking/data-center-design` — putting all the pieces together.
- `networking/cisco-aci` — Cisco's modern data center overlay system.

## See Also

- `networking/ethernet`
- `networking/lacp`
- `networking/lldp`
- `networking/cisco-aci`
- `networking/data-center-design`
- `networking/bridge`
- `networking/private-vlans`
- `networking/q-in-q`
- `networking/macvlan`
- `ramp-up/ip-eli5`
- `ramp-up/spine-leaf-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- IEEE 802.1D-2004, *IEEE Standard for Local and Metropolitan Area Networks: Media Access Control (MAC) Bridges* (the consolidated STP/RSTP standard).
- IEEE 802.1w-2001, *Rapid Reconfiguration of Spanning Tree* (the RSTP amendment, later folded into 802.1D-2004).
- IEEE 802.1s-2002, *Multiple Spanning Trees* (the MSTP amendment, later folded into 802.1Q-2005).
- IEEE 802.1Q-2018, *Bridges and Bridged Networks* (the modern consolidated bridging spec, supersedes 802.1D and 802.1s).
- Radia Perlman, *An Algorithm for Distributed Computation of a Spanning Tree in an Extended LAN* (1985 ACM SIGCOMM paper — the original).
- Radia Perlman, *The Algorhyme* (her famous spanning-tree poem):
  > *I think that I shall never see / A graph more lovely than a tree. / A tree whose crucial property / Is loop-free connectivity. / A tree which must be sure to span / So packets can reach every LAN. / First the Root must be selected / By ID it is elected. / Least cost paths from Root are traced / In the tree these paths are placed. / A mesh is made by folks like me / Then bridges find a spanning tree.*
- Cisco, *Cisco STP Cookbook* (Cisco Press, 2005) — the practical real-world handbook.
- Russ White and Ethan Banks, *The Art of Network Architecture* (Cisco Press, 2014), Chapter 15 ("Layer 2 Topologies").
- RFC 6325, *Routing Bridges (RBridges): Base Protocol Specification* (TRILL — IS-IS for L2).
- IEEE 802.1aq-2012, *Shortest Path Bridging* (SPB).
- Linux `mstpd` man pages: `mstpctl(8)`, `mstpd(8)`.
- Open vSwitch documentation, *RSTP Configuration*: https://docs.openvswitch.org/en/latest/howto/rstp/
- Cisco IOS Configuration Guide, *Spanning Tree Protocol*.
- Juniper Junos Configuration Guide, *Configuring Spanning Tree Protocols*.
- Arista EOS Configuration Guide, *Spanning Tree Protocol*.
