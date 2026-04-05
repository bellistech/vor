# JunOS BNG -- Deep Dive

> Beyond the cheat sheet: BNG architecture internals, dynamic profile state machines,
> RADIUS interaction models, address assignment strategies, per-subscriber CoS design,
> subscriber scaling on MX platforms, redundancy mechanisms, disaggregated BNG (CUPS),
> and a thorough comparison with Cisco IOS-XR BNG.

## Prerequisites

- Familiarity with PPPoE and IPoE subscriber session concepts.
- Understanding of RADIUS authentication, authorization, and accounting flows.
- Comfort with Junos CLI configuration and dynamic profiles.
- Review the companion sheet: `sheets/juniper/junos-bng.md`.
- Understanding of CoS fundamentals (forwarding classes, schedulers, shapers).

## 1. JunOS BNG vs IOS-XR BNG -- Implementation Comparison

Junos (MX Series) and IOS-XR (ASR9000/NCS5500) are the two dominant BNG platforms
in service provider networks. Their architectural approaches differ significantly.

### Subscriber Session Model

```
JunOS (MX Series):
  - Subscriber = dynamic interface (pp0.x for PPPoE, demux0.x for IPoE)
  - Each subscriber gets a unique logical interface (IFL)
  - Dynamic profiles generate the IFL at session creation
  - Subscriber state is managed by the authd process
  - Interface hierarchy: physical -> VLAN -> subscriber IFL

IOS-XR (ASR9000):
  - Subscriber = session within a subscriber-interface (sub-if)
  - Uses "dynamic template" (equivalent to dynamic profile)
  - Subscriber state managed by iedge process
  - Interface hierarchy: physical -> sub-interface -> access-interface
  - Supports both per-session and per-VLAN subscriber models
```

### Configuration Paradigm

| Aspect                     | JunOS (MX)                          | IOS-XR (ASR9000)                      |
|----------------------------|-------------------------------------|---------------------------------------|
| Subscriber interface       | `pp0` (PPPoE), `demux0` (IPoE)     | `Bundle-Ether0.100` (sub-interface)   |
| Profile mechanism          | Dynamic profiles                    | Dynamic templates                     |
| Profile variables          | `$junos-*` predefined vars          | Attribute-based substitution          |
| Service stacking           | Multiple dynamic profiles stacked   | Service policy chaining               |
| AAA binding                | Access profile per interface        | AAA method list per sub-interface     |
| CoS binding                | Traffic-control-profile per IFL     | Policy-map on sub-interface           |
| Address assignment         | Local pool or RADIUS Framed-IP      | Local pool, RADIUS, DHCP proxy        |
| VLAN auto-config           | `auto-configure vlan-ranges`        | `subscriber ambiguity` + control policy |
| Session trigger            | PPPoE PADI/DHCP Discover on VLAN    | Control policy event triggers         |

### Control Plane Differences

```
JunOS Control Plane:
  authd          -- central subscriber authentication daemon
  jpppd          -- PPP protocol handling (LCP, CHAP, IPCP)
  jdhcpd         -- DHCP server/relay process
  dfcd           -- dynamic flow capture (subscriber creation)
  smid           -- subscriber management infrastructure daemon
  cosd           -- class of service daemon (per-sub queuing)

IOS-XR Control Plane:
  iedge          -- subscriber session manager
  pppoe_ma       -- PPPoE protocol handler
  dhcpd          -- DHCP process
  aaa            -- authentication/authorization/accounting
  pbr/qos        -- policy-based routing and QoS
  spa_ctrl       -- shared port adapter control
```

### Key Architectural Differences

**1. Interface Abstraction:**
Junos creates a first-class logical interface (IFL) for every subscriber. This means
each subscriber appears in `show interfaces`, has its own counters, and can be
individually targeted by firewall filters and CoS. IOS-XR uses a more lightweight
session abstraction within a subscriber-interface, which scales better in raw numbers
but offers less per-subscriber visibility in the interface table.

**2. Service Activation:**
Junos uses stacked dynamic profiles -- a base profile creates the subscriber session,
and additional service profiles are layered on top via RADIUS VSA
`Juniper-Service-Activate`. IOS-XR uses a control-policy framework where events
(session-start, CoA, timer expiry) trigger actions (activate-service, set-timer,
authenticate). The IOS-XR model is more event-driven; the Junos model is more
declarative.

**3. QoS Architecture:**
Junos implements hierarchical CoS (H-CoS) with per-subscriber queuing at the hardware
level using MPC line cards. Each subscriber gets dedicated hardware queues. IOS-XR uses
a policy-map hierarchy (parent shaper -> child policy per class) which is conceptually
similar but implemented through different hardware (Memory Buffer Processor on Typhoon/
Tomahawk NPUs vs MBP on Memory-type line cards in Junos).

**4. Redundancy:**
Junos subscriber redundancy uses subscriber replication between BNG pairs via the
unified-edge framework. IOS-XR uses Session Redundancy (SergR) with a dedicated
redundancy group model. Both achieve stateful failover, but the configuration
models differ significantly.

**5. Scale Comparison (approximate per chassis):**

| Metric                    | MX2020 (Junos)    | ASR9922 (IOS-XR)   |
|---------------------------|-------------------|---------------------|
| Max subscribers           | 512K-1M           | 512K-1M             |
| Max PPPoE sessions        | 256K              | 256K                |
| Max IPoE sessions         | 512K              | 512K                |
| Max subscriber VLANs      | 128K              | 128K                |
| Max H-CoS queues          | 256K              | 256K                |
| Max dynamic profiles      | 1024              | 1024                |
| Max RADIUS transactions/s | 20K               | 15K                 |

These numbers are line-card dependent and vary by software version.

## 2. Dynamic Profile Architecture

Dynamic profiles are the central abstraction in Junos BNG. They are configuration
templates instantiated at subscriber login time, with variables resolved from RADIUS
attributes, DHCP options, or system-derived values.

### Profile Lifecycle

```
1. TRIGGER
   - PPPoE PADI arrives on auto-configure interface
   - DHCP Discover arrives on DHCP-enabled interface
   - dfcd (dynamic flow capture daemon) catches the packet

2. PROFILE SELECTION
   - auto-configure stanza maps the trigger to a dynamic profile name
   - RADIUS can override with Juniper-Switching-Filter or
     Juniper-Local-User-Name pointing to a different profile

3. VARIABLE RESOLUTION
   - $junos-interface-unit = system-assigned unit number (monotonically increasing)
   - $junos-underlying-interface = physical/VLAN interface that received the trigger
   - $junos-subscriber-ip-address = from RADIUS Framed-IP-Address or local pool
   - $junos-cos-traffic-control-profile = from RADIUS Juniper-Cos-* VSA
   - $junos-input-filter / $junos-output-filter = from RADIUS VSAs

4. INTERFACE CREATION
   - smid creates the logical interface (pp0.x or demux0.x)
   - cosd applies CoS bindings
   - firewall module installs per-subscriber filters
   - routing adds the subscriber route to the RIB

5. SESSION ACTIVE
   - Accounting Start sent to RADIUS
   - Interim updates on configured interval
   - Session persists until logout, timeout, or administrative clear

6. TEARDOWN
   - PPPoE PADT, DHCP Release, RADIUS Disconnect-Request, or admin clear
   - Accounting Stop sent to RADIUS
   - Interface destroyed, routes withdrawn, CoS/filter state removed
   - Pool address returned
```

### Profile Composition (Service Stacking)

Junos supports stacking multiple dynamic profiles on a single subscriber session.
This enables a modular service design.

```
Layer 0: Auto-configure       -- selects the base profile
Layer 1: Base subscriber       -- creates IFL, assigns IP, sets default CoS
Layer 2: Service profile(s)    -- activated via RADIUS or CoA
Layer 3: Additional services   -- e.g., multicast, VoIP prioritization

Example stack for a 100M residential subscriber with VoIP:

  BASE-RESIDENTIAL        -- creates demux0.x, assigns IP from POOL-RES
  SERVICE-100M            -- applies TCP-100M shaping
  SERVICE-VOIP            -- adds EF classification for RTP, priority queue boost

Activation sequence in RADIUS Access-Accept:
  Juniper-Service-Activate = "SERVICE-100M"
  Juniper-Service-Activate += "SERVICE-VOIP"

Mid-session upgrade via CoA:
  Juniper-Service-Deactivate = "SERVICE-100M"
  Juniper-Service-Activate = "SERVICE-1G"
```

### Internal Data Structures

```
Profile storage:
  - Profiles stored in the configuration database
  - At commit time, profiles are compiled into internal templates
  - Templates are indexed by name for O(1) lookup at subscriber login

Per-subscriber state:
  - Managed by smid (subscriber management infrastructure daemon)
  - Stored in shared memory segment accessible by authd, cosd, dfcd
  - State includes: session ID, interface name, IP assignment, active services,
    accounting counters, CoS profile reference, filter references

Interface state:
  - Each subscriber IFL has a kernel IFL entry
  - IFL entries consume memory proportional to the number of families configured
  - pp0 IFLs also include PPP state (LCP, authentication, IPCP/IPv6CP states)
```

## 3. Junos Subscriber State Machine

The subscriber session in Junos follows a well-defined state machine managed by
authd and smid.

### PPPoE State Machine

```
                          PADI received
                               |
                               v
                     +-------------------+
                     |    INIT/IDLE      |
                     +-------------------+
                               |
                          Send PADO
                               |
                               v
                     +-------------------+
                     |   PADO_SENT       |
                     +-------------------+
                               |
                          PADR received
                               |
                               v
                     +-------------------+
                     |   SESSION_START   |  -- Send PADS, assign Session-ID
                     +-------------------+
                               |
                          LCP negotiation
                               |
                               v
                     +-------------------+
                     |   LCP_NEGOTIATING |  -- MRU, magic number, auth method
                     +-------------------+
                               |
                          LCP Open
                               |
                               v
                     +-------------------+
                     |   AUTHENTICATING  |  -- CHAP challenge/response or PAP
                     +-------------------+     RADIUS Access-Request sent
                               |
                     RADIUS Access-Accept
                               |
                               v
                     +-------------------+
                     |   NCP_NEGOTIATING |  -- IPCP (IPv4), IPv6CP
                     +-------------------+     Framed-IP or pool assignment
                               |
                          IPCP Open
                               |
                               v
                     +-------------------+
                     |   SESSION_ACTIVE  |  -- Acct-Start sent
                     +-------------------+     Traffic flowing
                               |
              PADT / timeout / admin clear / CoA disconnect
                               |
                               v
                     +-------------------+
                     |   TERMINATING     |  -- Acct-Stop sent
                     +-------------------+     IFL destroyed
                               |
                               v
                     +-------------------+
                     |   TERMINATED      |  -- Resources freed
                     +-------------------+
```

### IPoE State Machine

```
                     DHCP Discover received
                               |
                               v
                     +-------------------+
                     |   INIT            |  -- Client identification (MAC, Option 82)
                     +-------------------+
                               |
                     RADIUS Access-Request (MAC as username)
                               |
                               v
                     +-------------------+
                     |   AUTHENTICATING  |  -- Waiting for RADIUS response
                     +-------------------+
                               |
                     RADIUS Access-Accept (Framed-IP or pool)
                               |
                               v
                     +-------------------+
                     |   DHCP_OFFERING   |  -- DHCP Offer sent to client
                     +-------------------+
                               |
                          DHCP Request received
                               |
                               v
                     +-------------------+
                     |   DHCP_BINDING    |  -- DHCP Ack sent, IFL created
                     +-------------------+     demux0.x created
                               |
                          Acct-Start sent
                               |
                               v
                     +-------------------+
                     |   SESSION_ACTIVE  |  -- Traffic flowing
                     +-------------------+     Lease timer running
                               |
              DHCP Release / lease expiry / admin clear / CoA disconnect
                               |
                               v
                     +-------------------+
                     |   TERMINATING     |  -- Acct-Stop sent, IFL destroyed
                     +-------------------+

IPoE authentication modes:
  - MAC-based: client MAC address used as username
  - Option 82: circuit-id/remote-id used for identification
  - No-auth: DHCP only, no RADIUS (address from local pool)
```

### State Transition Timers

| Timer                      | Default   | Purpose                                     |
|----------------------------|-----------|---------------------------------------------|
| PPPoE PADI timeout         | 30s       | Time to wait for PADR after PADO            |
| LCP negotiation timeout    | 30s       | Max time for LCP to reach Open              |
| Authentication timeout     | 30s       | Max time for RADIUS response                |
| IPCP negotiation timeout   | 30s       | Max time for IPCP to reach Open             |
| Session-Timeout (RADIUS)   | varies    | Max session duration (from Access-Accept)    |
| Idle-Timeout (RADIUS)      | varies    | Inactivity timeout                           |
| DHCP lease time            | 86400s    | Standard DHCP lease (from pool config)       |
| Accounting interim interval| 600s      | Interim-Update frequency                     |
| RADIUS retry timeout       | 5s        | Per-attempt RADIUS timeout                   |
| RADIUS max retries         | 3         | Attempts before declaring RADIUS unreachable |

## 4. RADIUS Interaction Model

### Authentication Flow

```
BNG                                          RADIUS Server
 |                                                |
 |  Access-Request                                |
 |  - User-Name = "user@example.com"              |
 |  - User-Password or CHAP-Password             |
 |  - NAS-IP-Address = 192.168.1.1               |
 |  - NAS-Port = <interface index>                |
 |  - NAS-Port-Type = Ethernet (15)              |
 |  - NAS-Port-Id = "ge-1/0/0:100"              |
 |  - Calling-Station-Id = <client MAC>           |
 |  - Called-Station-Id = <service name>          |
 |  - Framed-Protocol = PPP (1) [PPPoE only]     |
 |  - Acct-Session-Id = <unique session ID>       |
 |----------------------------------------------->|
 |                                                |
 |  Access-Accept                                 |
 |  - Framed-IP-Address = 10.200.1.50            |
 |  - Framed-IP-Netmask = 255.255.255.255        |
 |  - Session-Timeout = 86400                    |
 |  - Idle-Timeout = 3600                        |
 |  - Framed-Pool = "POOL-100M" [alternative]    |
 |  - Juniper-Cos-Traffic-Control-Profile = TCP-100M |
 |  - Juniper-Ingress-Policy-Name = FILTER-IN    |
 |  - Juniper-Egress-Policy-Name = FILTER-OUT    |
 |  - Juniper-Service-Activate = "SVC-VOIP"      |
 |  - Juniper-Primary-Dns = 8.8.8.8             |
 |  - Juniper-Secondary-Dns = 8.8.4.4           |
 |<-----------------------------------------------|
 |                                                |
 |  [Session established, subscriber active]      |
```

### Accounting Flow

```
BNG                                          RADIUS Server
 |                                                |
 |  Accounting-Request (Start)                    |
 |  - Acct-Status-Type = Start (1)               |
 |  - Acct-Session-Id = <session ID>             |
 |  - User-Name = "user@example.com"              |
 |  - NAS-IP-Address = 192.168.1.1               |
 |  - Framed-IP-Address = 10.200.1.50            |
 |  - Acct-Session-Time = 0                       |
 |----------------------------------------------->|
 |  Accounting-Response                           |
 |<-----------------------------------------------|
 |                                                |
 |  ... time passes (interim interval) ...        |
 |                                                |
 |  Accounting-Request (Interim-Update)           |
 |  - Acct-Status-Type = Interim-Update (3)      |
 |  - Acct-Input-Octets = 1073741824             |
 |  - Acct-Output-Octets = 5368709120            |
 |  - Acct-Input-Packets = 892451                |
 |  - Acct-Output-Packets = 4215789              |
 |  - Acct-Session-Time = 600                    |
 |----------------------------------------------->|
 |  Accounting-Response                           |
 |<-----------------------------------------------|
 |                                                |
 |  ... session ends ...                          |
 |                                                |
 |  Accounting-Request (Stop)                     |
 |  - Acct-Status-Type = Stop (2)                |
 |  - Acct-Terminate-Cause = User-Request (1)    |
 |  - Acct-Session-Time = 28800                  |
 |  - Acct-Input-Octets = <total>                |
 |  - Acct-Output-Octets = <total>               |
 |----------------------------------------------->|
 |  Accounting-Response                           |
 |<-----------------------------------------------|
```

### Change of Authorization (CoA) Flow

```
RADIUS Server                                BNG
 |                                                |
 |  CoA-Request                                   |
 |  - Acct-Session-Id = <session ID>              |
 |  - Juniper-Cos-Traffic-Control-Profile = TCP-1G|
 |  - Juniper-Service-Deactivate = "SVC-100M"    |
 |  - Juniper-Service-Activate = "SVC-1G"        |
 |----------------------------------------------->|
 |                                                |
 |  [BNG applies changes to active session]       |
 |  [Updates CoS profile, swaps service profiles] |
 |                                                |
 |  CoA-ACK                                       |
 |<-----------------------------------------------|

CoA Use Cases:
  - Speed tier upgrade/downgrade (change traffic-control-profile)
  - Service activation (add VoIP, IPTV multicast, VPN)
  - Service deactivation
  - Firewall filter change (parental controls, content filtering)
  - Session timeout modification
  - Redirect URL injection (captive portal, notification)

Disconnect-Request (DM):
  - RADIUS sends Disconnect-Request with session identifier
  - BNG terminates the session, sends Accounting-Stop
  - Used for: prepaid balance exhausted, admin disconnect, abuse response
```

### RADIUS Failover Behavior

```
Primary RADIUS server:     10.0.0.100
Secondary RADIUS server:   10.0.0.101

Failover logic:
  1. Send Access-Request to primary (10.0.0.100)
  2. Wait for timeout (default 5 seconds)
  3. Retry up to max-retries (default 3)
  4. If all retries exhausted, mark primary as dead
  5. Send Access-Request to secondary (10.0.0.101)
  6. Dead-time timer starts for primary (default 300 seconds)
  7. After dead-time, primary is re-probed

Failure modes:
  - RADIUS timeout during PPPoE auth: PPPoE session torn down
  - RADIUS timeout during DHCP auth: DHCP Offer not sent, client retries
  - Accounting server unreachable: buffered locally (if configured)
  - CoA listener unreachable: RADIUS server retries independently
```

## 5. Address Assignment Strategies

### Strategy Comparison

| Strategy                | Source        | Scale    | Use Case                        |
|-------------------------|---------------|----------|---------------------------------|
| Local pool              | BNG config    | Medium   | Small-to-mid ISP, deterministic |
| RADIUS Framed-IP        | RADIUS server | High     | Static IP, enterprise subs      |
| RADIUS Framed-Pool      | RADIUS->local | High     | Pool selection by RADIUS         |
| DHCP relay to external  | External DHCP | High     | Centralized IPAM                 |
| Linked pools            | BNG config    | High     | Pool chaining for overflow       |

### Local Pool Design

```
Design considerations:
  - Pool size must accommodate peak concurrent subscribers + headroom (20%)
  - Avoid overlapping pools across BNG nodes (partition address space)
  - Use /16 or larger blocks per pool for operational simplicity
  - Configure multiple ranges within a pool for phased provisioning

Pool hierarchy:
  Pool RESIDENTIAL
    Range HOME-1:      10.100.0.2   - 10.100.63.254     (~16K addresses)
    Range HOME-2:      10.100.64.2  - 10.100.127.254    (~16K addresses)
    Range HOME-3:      10.100.128.2 - 10.100.191.254    (~16K addresses)
    Range OVERFLOW:    10.100.192.2 - 10.100.255.254    (~16K addresses)
    DHCP attributes:   router 10.100.0.1, DNS, lease 86400

Linked pools (overflow):
  Pool RESIDENTIAL links-to RESIDENTIAL-OVERFLOW
  Pool RESIDENTIAL-OVERFLOW network 10.101.0.0/16

When RESIDENTIAL is exhausted, assignments spill into RESIDENTIAL-OVERFLOW.
```

### IPv6 Address Assignment Models

```
Model 1: SLAAC + DHCPv6-PD (most common residential)
  - Router Advertisement provides /64 prefix on the access link
  - Subscriber CPE derives address via SLAAC (EUI-64 or privacy extensions)
  - DHCPv6-PD delegates a /56 or /48 to the CPE for LAN subnets
  - BNG installs /128 host route for the link address
  - BNG installs /56 (or /48) route for the delegated prefix

Model 2: DHCPv6 IA-NA + DHCPv6-PD
  - DHCPv6 assigns a /128 address via IA-NA (non-temporary)
  - DHCPv6 delegates a /56 via IA-PD for LAN
  - More deterministic than SLAAC, preferred for business subscribers

Model 3: DHCPv6-PD Only
  - CPE uses link-local only on the access link
  - DHCPv6-PD provides /56 or /48 for all addressing
  - Subscriber self-assigns from delegated block
  - Simplest model but less visibility into CPE WAN address

Prefix sizing:
  /48 per subscriber = 65536 /64 LAN subnets (enterprise)
  /56 per subscriber = 256 /64 LAN subnets (residential, recommended)
  /60 per subscriber = 16 /64 LAN subnets (constrained)
  /64 per subscriber = 1 LAN subnet only (not recommended, breaks PD)
```

## 6. Per-Subscriber CoS in JunOS

### Hierarchical CoS Architecture

```
Junos BNG implements three-level hierarchical CoS:

Level 1: Physical port
  - Aggregate shaping at the physical interface rate
  - All subscribers on the port share this bandwidth
  - Configured via traffic-control-profile on the physical interface

Level 2: Subscriber session
  - Per-subscriber shaping (e.g., 50M, 100M, 1G per subscriber)
  - Each subscriber has dedicated hardware queues (on supported MPCs)
  - Configured via traffic-control-profile on the subscriber IFL

Level 3: Per-class within subscriber
  - Within each subscriber's shaped rate, traffic is classified into queues
  - Scheduler map defines bandwidth allocation per forwarding class
  - Typical: BE (remainder), EF (30%, strict-high), AF (20%), NC (5%)

Hardware requirements:
  - MPC7E, MPC8E, MPC9E, MPC10E, MPC11E support per-subscriber queuing
  - Memory Buffer Processor (MBP) on the line card manages queue state
  - Each subscriber can have 4 or 8 queues depending on configuration
  - Queue memory is partitioned across subscribers dynamically
```

### Traffic Control Profile Internals

```
A traffic-control-profile bundles:
  1. Shaping rate       -- aggregate output rate for the subscriber
  2. Guaranteed rate    -- minimum bandwidth guarantee (CIR)
  3. Scheduler map      -- per-class bandwidth allocation within the shaped rate
  4. Burst size         -- token bucket burst allowance (default: 15ms of shaping rate)
  5. Delay buffer rate  -- maximum burst absorption before tail drop

Shaping calculation:
  shaping-rate 100m
  burst-size = 100,000,000 * 0.015 = 1,500,000 bits = ~183 KB (default 15ms burst)

  The token bucket refills at 100 Mbps.
  Bursts up to 183 KB are absorbed without drops.
  Sustained rate above 100 Mbps is shaped (delayed, then dropped if buffer full).

Guaranteed rate interaction:
  guaranteed-rate 20m
  When the port is congested, the subscriber is guaranteed at least 20 Mbps.
  Above 20 Mbps up to the shaping-rate of 100 Mbps, bandwidth is best-effort
  and subject to scheduler priority and weight among other subscribers.
```

### Classifier and Rewrite Rules

```
Ingress classification (subscriber -> BNG):
  - DSCP classifier maps incoming packet DSCP to forwarding class + loss priority
  - Applied per-subscriber via dynamic profile variable
  - Typical mapping:
      DSCP EF (46)  -> expedited-forwarding, low loss
      DSCP AF41 (34) -> assured-forwarding, low loss
      DSCP 0 (BE)   -> best-effort, low loss

Egress rewrite (BNG -> subscriber):
  - Rewrites DSCP/802.1p bits on packets leaving toward subscriber
  - Ensures downstream traffic carries correct CoS markings
  - Applied per-subscriber via dynamic profile

Per-subscriber filter interaction:
  - Firewall filter runs before CoS classification (input direction)
  - Filter can set forwarding-class and loss-priority explicitly
  - This overrides the classifier for matched traffic
  - Common use: force all P2P traffic to best-effort regardless of DSCP
```

### CoS Scaling Considerations

```
Per-subscriber queue memory consumption:
  - Each queue consumes ~2 KB of MBP SRAM for scheduling state
  - 4-queue model: 8 KB per subscriber
  - 8-queue model: 16 KB per subscriber

MPC queue capacity (approximate):
  MPC7E:   64K subscriber queues (4Q model = 16K subscribers per MPC)
  MPC8E:   128K subscriber queues
  MPC9E:   256K subscriber queues
  MPC10E:  128K subscriber queues (cost-optimized)
  MPC11E:  512K subscriber queues

Design rule:
  Total queues needed = subscribers_per_MPC * queues_per_subscriber
  Always provision 20% headroom for queue memory

When queue memory is exhausted:
  - New subscribers fall back to port-level queuing (no per-sub shaping)
  - Existing subscribers retain their dedicated queues
  - Alarm raised: "CoS memory exhausted on FPC X"
```

## 7. Subscriber Scaling on MX Platforms

### Platform Density Comparison

| Platform | Max Subscribers | Max PPPoE  | Max IPoE   | Line Cards     | Slots  |
|----------|-----------------|------------|------------|----------------|--------|
| MX240    | 64K             | 32K        | 64K        | MPC7E-11E      | 3      |
| MX480    | 128K            | 64K        | 128K       | MPC7E-11E      | 6      |
| MX960    | 256K            | 128K       | 256K       | MPC7E-11E      | 12     |
| MX2010   | 512K            | 256K       | 512K       | MPC7E-11E      | 10     |
| MX2020   | 1M              | 512K       | 1M         | MPC7E-11E      | 20     |
| MX10003  | 128K            | 64K        | 128K       | Fixed (3 slots)| 3      |
| MX10008  | 256K            | 128K       | 256K       | LC1101/2101    | 8      |
| MX10016  | 512K            | 256K       | 512K       | LC1101/2101    | 16     |

Note: These are approximate maximums. Actual capacity depends on line card type,
features enabled (H-CoS, IPv6, filters), and Junos version.

### Scaling Bottlenecks

```
1. Control Plane (RE)
   - authd processes RADIUS transactions serially per thread
   - Multi-threaded authd can handle ~10K-20K authentications/second
   - Burst login scenarios (power outage recovery) can overwhelm authd
   - Mitigation: PPPoE rate limiters, DHCP rate limiters, staggered retry timers

2. Data Plane (MPC)
   - Memory: each subscriber IFL consumes ~4 KB of MPC forwarding table memory
   - FIB: each subscriber creates a /32 host route in the FIB
   - ACL: per-subscriber firewall filters consume TCAM entries
   - CoS: per-subscriber queues consume MBP SRAM (see Section 6)

3. RADIUS Infrastructure
   - BNG generates 3+ RADIUS transactions per subscriber session
     (Access-Request, Acct-Start, Acct-Stop, plus interim updates)
   - 100K subscribers with 10-min interim = ~167 accounting packets/second steady state
   - Login storm of 50K subscribers in 5 minutes = ~333 auth packets/second burst
   - RADIUS server must handle the burst or subscribers fail to authenticate

4. Address Pool Exhaustion
   - Monitor pool utilization: show network-access address-assignment pool <name> usage
   - Alert at 80% utilization, emergency expansion at 90%
   - Linked pools provide automatic overflow
```

### Login Storm Mitigation

```
# Limit PPPoE session setup rate (per interface)
set protocols pppoe max-sessions-per-interface 8000
set protocols pppoe service-name-table default-service service-name any

# Limit DHCP rate
set system services dhcp-local-server dhcpv4 group DHCP-GROUP overrides process-inform-rate 100
set system services dhcp-local-server dhcpv4 group DHCP-GROUP overrides client-discover-match

# RADIUS rate limiting
set access radius-server 10.0.0.100 max-outstanding-requests 2000

# Stagger subscriber retry timers (PPPoE)
# Subscribers that fail auth will retry at randomized intervals
# instead of all retrying simultaneously
```

## 8. Redundancy Mechanisms

### Dual-RE GRES/NSR

```
Graceful Routing Engine Switchover (GRES):
  - Primary RE replicates kernel state to backup RE
  - On RE failure, backup assumes control
  - Interfaces remain up during switchover
  - Subscriber sessions are NOT preserved by GRES alone

Nonstop Active Routing (NSR):
  - Replicates routing protocol state (BGP, OSPF, IS-IS)
  - Combined with GRES, routing peers do not detect the switchover
  - Subscriber sessions still require subscriber replication

Nonstop Bridging (NSB):
  - Replicates L2 state (MAC tables, VLAN)
  - Required for IPoE subscribers on bridged access interfaces

Configuration:
  set chassis redundancy graceful-switchover
  set routing-options nonstop-routing
  set protocols layer2-control nonstop-bridging
```

### Subscriber Replication (Inter-Chassis)

```
Architecture:
  BNG-A (Active)  <---- subscriber replication link ----> BNG-B (Standby)

  - Active BNG handles all subscriber sessions
  - Subscriber state replicated to standby in real-time
  - On active failure, standby assumes all sessions
  - Access network (DSLAM/OLT) must support dual-homing or fast failover

Replicated state:
  - PPPoE session state (Session-ID, MAC, VLAN mapping)
  - IPoE binding (MAC, IP, VLAN, lease state)
  - IP address assignments
  - Active dynamic profiles and service profiles
  - CoS profile bindings
  - Firewall filter state
  - Accounting counters (for accurate billing on failover)
  - RADIUS session IDs (so Acct-Stop uses correct session ID)

Not replicated:
  - In-flight packets (momentary traffic loss during failover)
  - PPP LCP/NCP state (re-negotiated transparently)
  - RADIUS pending transactions (retried by new active)

Failover time: typically 1-3 seconds for subscriber traffic restoration
```

### ISSU for BNG

```
In-Service Software Upgrade preserves subscriber sessions during Junos upgrade.

Requirements:
  - Dual RE chassis (MX240, MX480, MX960, MX2010, MX2020)
  - GRES + NSR enabled
  - ISSU-compatible software versions (same major version family)
  - Sufficient memory on both REs

ISSU process:
  1. Upload new Junos image to backup RE
  2. request system software in-service-upgrade <image>
  3. Backup RE boots with new image
  4. Backup RE synchronizes state from primary
  5. Switchover: backup becomes new primary
  6. Old primary reboots with new image
  7. Old primary rejoins as new backup

Subscriber impact during ISSU:
  - Sub-second traffic interruption during RE switchover
  - No subscriber re-authentication required
  - Accounting sessions continue with same session IDs
  - CoS and filter state preserved through switchover

Limitations:
  - Not supported across major version boundaries
  - Some line card upgrades require Unified ISSU (separate process)
  - FPC-level ISSU (MPC firmware update) causes brief per-FPC outage
```

### Redundancy Design Patterns

```
Pattern 1: Dual-RE Single Chassis
  - Protects against RE failure only
  - GRES + NSR + subscriber replication (intra-chassis)
  - Simple, cost-effective, single point of failure at chassis level

Pattern 2: Dual-Chassis Active/Standby
  - Two MX chassis, one active, one standby
  - Inter-chassis subscriber replication
  - Access network dual-homed (LAG or LACP to both BNGs)
  - VRRP or MC-LAG for gateway redundancy
  - Full chassis protection at 2x hardware cost

Pattern 3: Dual-Chassis Active/Active
  - Both BNG chassis actively serving subscribers
  - Subscribers distributed across both (e.g., odd/even VLAN)
  - Each chassis is standby for the other's subscribers
  - Better resource utilization than active/standby
  - More complex RADIUS and access network design

Pattern 4: Geographic Redundancy
  - BNG nodes in different sites
  - Access rings dual-homed to both sites
  - RADIUS profile determines primary vs backup BNG
  - Protects against site failure
  - Highest complexity, highest availability
```

## 9. CUPS / Disaggregated BNG

### Traditional vs Disaggregated BNG

```
Traditional (Integrated) BNG:
  +------------------------------------------+
  |              MX Series BNG               |
  |  +----------+  +----------+  +--------+  |
  |  | Control  |  |  User    |  | Line   |  |
  |  |  Plane   |  |  Plane   |  | Cards  |  |
  |  | (RE)     |  | (PFE)    |  | (MPC)  |  |
  |  +----------+  +----------+  +--------+  |
  +------------------------------------------+
  - Control and user plane in same chassis
  - Scale by adding line cards or chassis
  - Well-proven, operationally mature

CUPS (Control and User Plane Separation) BNG:
  +-------------------+        +-------------------+
  |   Control Plane   |        |    User Plane     |
  |   (Virtualized)   | <----> |   (MX / vMX)      |
  |                   |  PFCP  |                   |
  |  - RADIUS client  |        |  - Forwarding     |
  |  - DHCP server    |        |  - QoS            |
  |  - PPPoE control  |        |  - ACL            |
  |  - Session mgmt   |        |  - Accounting     |
  +-------------------+        +-------------------+

  PFCP = Packet Forwarding Control Protocol (3GPP-derived, similar to GTP-C/U split)
```

### CUPS Architecture Benefits

```
1. Independent Scaling
   - Scale control plane (VM instances) independently from user plane (hardware)
   - Add forwarding capacity without proportional control plane growth
   - Control plane can run on COTS servers in the cloud or data center

2. Centralized Control
   - Single control plane instance manages multiple user plane nodes
   - Unified subscriber database and policy
   - Simplified operations: one place to configure AAA, pools, profiles

3. Cost Optimization
   - User plane can be simpler (and cheaper) forwarding-only hardware
   - Control plane virtualized on standard compute
   - Mix of user plane hardware: MX for high-density, vMX for remote/edge

4. Agile Service Deployment
   - New services deployed by updating control plane software only
   - User plane firmware updates less frequent
   - CI/CD for control plane functions

5. Placement Flexibility
   - User plane close to subscribers (latency-sensitive forwarding)
   - Control plane centralized (operational simplicity)
   - Supports distributed edge architectures
```

### Junos CUPS Implementation

```
Juniper's approach to disaggregated BNG:

Cloud-Native BNG (CN-BNG):
  - Control plane: containerized microservices on Kubernetes
  - User plane: MX Series or MX10K with BNG-UP software
  - Protocol: PFCP between control and user plane
  - Session management: cloud-native, stateless, horizontally scalable

Components:
  CN-BNG Controller (Control Plane):
    - Subscriber session manager
    - RADIUS client
    - DHCP server/relay
    - PPPoE control (PADI/PADO/PADR/PADS)
    - Policy engine
    - Address pool manager
    - REST API for OSS/BSS integration

  BNG-UP (User Plane):
    - Packet forwarding (subscriber routes)
    - Per-subscriber QoS (H-CoS)
    - Per-subscriber ACLs
    - Accounting (usage counters)
    - PFCP agent (receives rules from controller)

PFCP Session Rules:
  - PDR (Packet Detection Rule): match subscriber traffic
  - FAR (Forwarding Action Rule): forward, drop, redirect
  - QER (QoS Enforcement Rule): shaping rate, scheduler map
  - URR (Usage Reporting Rule): accounting counters
```

### Migration Path: Integrated to Disaggregated

```
Phase 1: Integrated BNG (current state)
  - Traditional MX BNG with all functions in one chassis
  - Well-understood, stable, proven at scale

Phase 2: Hybrid
  - Introduce CN-BNG controller alongside existing MX BNGs
  - Controller manages new subscriber sessions
  - Existing MX BNGs continue serving legacy subscribers
  - Gradual migration of subscribers to controller-managed sessions

Phase 3: Full CUPS
  - All subscriber sessions managed by CN-BNG controller
  - MX chassis run BNG-UP software only
  - Control plane fully virtualized and cloud-native
  - User plane hardware optimized for forwarding density

Migration considerations:
  - RADIUS infrastructure must support both integrated and CUPS BNG
  - OSS/BSS integration changes (API-based vs SNMP/CLI)
  - Monitoring and troubleshooting workflows change
  - Staff training on Kubernetes, microservices, PFCP
  - Rollback plan: user plane can fall back to integrated mode
```

## Key Takeaways

1. **Junos BNG creates a full logical interface per subscriber.** This provides
   excellent per-subscriber visibility and control but consumes more control plane
   resources than session-based models.

2. **Dynamic profiles are the core abstraction.** Everything from subscriber creation
   to QoS to firewall filters flows through dynamic profiles and their `$junos-*`
   variables.

3. **RADIUS drives the subscriber lifecycle.** Authentication, authorization (profile
   selection, QoS, filters), accounting, and mid-session changes (CoA) all flow
   through RADIUS.

4. **H-CoS gives real per-subscriber queuing.** Unlike many platforms that share
   queues across subscribers, Junos MPC line cards dedicate hardware queues per
   subscriber, enabling true per-subscriber QoS.

5. **Scaling is line-card bound.** Subscriber capacity, queue memory, TCAM for
   filters, and FIB entries are all constrained by MPC type. Design for the
   bottleneck, not the chassis maximum.

6. **Redundancy has multiple layers.** Dual-RE protects the control plane, inter-
   chassis replication protects against chassis failure, and ISSU minimizes planned
   downtime. Each adds complexity and cost.

7. **CUPS is the future direction.** Disaggregated BNG separates concerns, enables
   independent scaling, and aligns with cloud-native operations, but requires
   significant operational maturity.

## See Also

- JunOS Interfaces
- JunOS Routing Fundamentals
- JunOS Firewall Filters
- JunOS Class of Service (CoS)
- JunOS Architecture
- RADIUS and AAA
- PPPoE Protocol
- DHCP
- ISP Edge Architecture
- Cisco IOS-XR BNG

## References

- Juniper TechLibrary: Subscriber Management and Services Administration Guide
- Juniper TechLibrary: Broadband Subscriber Sessions Overview
- Juniper TechLibrary: Dynamic Profiles for Subscriber Access
- Juniper TechLibrary: Class of Service for Enhanced Subscriber Management
- Juniper TechLibrary: ANCP Overview
- Juniper TechLibrary: Subscriber Redundancy for Enhanced Subscriber Management
- Juniper TechLibrary: Cloud-Native Broadband Network Gateway
- RFC 2516 -- A Method for Transmitting PPP Over Ethernet (PPPoE)
- RFC 2865 -- Remote Authentication Dial In User Service (RADIUS)
- RFC 2866 -- RADIUS Accounting
- RFC 5176 -- Dynamic Authorization Extensions to RADIUS (CoA/DM)
- RFC 6320 -- Protocol for Access Node Control Mechanism in Broadband Networks (ANCP)
- RFC 8300 -- Network Service Header (NSH) -- context for service chaining in CUPS
- 3GPP TS 29.244 -- Interface between PFCP entities (CUPS protocol reference)
