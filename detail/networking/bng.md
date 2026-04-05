# The Architecture of BNG — Service Provider Subscriber Edge

> *The Broadband Network Gateway is where the anonymous packet stream becomes a named, authenticated, policy-controlled subscriber session. Every bit of revenue-generating traffic crosses this boundary.*

---

## 1. BNG in Service Provider Architecture

### The Access Network Stack

A service provider broadband network has three distinct planes, and the BNG sits at the critical junction between access and core:

```
                    ┌─────────────────────┐
                    │     Core Network     │
                    │   (MPLS/SR, P/PE)    │
                    └──────────┬──────────┘
                               │
                    ┌──────────┴──────────┐
                    │        BNG          │ ◄── Session termination
                    │  (Subscriber Edge)  │     Authentication/Authorization
                    └──────────┬──────────┘     Per-subscriber policy
                               │
                    ┌──────────┴──────────┐
                    │   Aggregation       │
                    │  (L2/L3 switches)   │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                │                │
         ┌────┴────┐     ┌────┴────┐     ┌────┴────┐
         │  DSLAM  │     │   OLT   │     │ CMTS/   │
         │         │     │         │     │ OLT     │
         └────┬────┘     └────┬────┘     └────┬────┘
              │                │                │
         [ DSL CPE ]     [ ONT/ONU ]     [ Cable  ]
                                         [ Modem  ]
```

### BNG vs Traditional BRAS

The terms BNG and BRAS (Broadband Remote Access Server) are often used interchangeably, but they represent an architectural evolution:

| Aspect | Traditional BRAS | Modern BNG |
|--------|-----------------|------------|
| Era | 2000-2010 | 2010-present |
| Session types | PPPoE/PPPoA only | PPPoE + IPoE + L2TP |
| Scale | 8K-32K sessions | 64K-256K+ sessions |
| Policy model | Static per-port | Dynamic per-subscriber |
| QoS | Port-based shaping | Hierarchical per-subscriber QoS |
| IPv6 | Bolt-on or absent | Native dual-stack |
| Service activation | Reprovisioning required | RADIUS-driven dynamic templates |
| Redundancy | Cold standby | Stateful session recovery |
| Control plane | Monolithic | Distributed (some vendors support disaggregated BNG) |

The key shift: BRAS was a "dial-up termination box with Ethernet." BNG is a "subscriber-aware policy enforcement point" that integrates AAA, QoS, ACL, NAT, and service activation into a unified per-subscriber model.

---

## 2. PPPoE Protocol State Machine

### Discovery Phase (Layer 2)

PPPoE discovery uses Ethernet frames with EtherType `0x8863`. The four-message exchange identifies a willing Access Concentrator (AC) and establishes a session:

```
State: IDLE
  │
  ├── CPE sends PADI (PPPoE Active Discovery Initiation)
  │   - Destination: ff:ff:ff:ff:ff:ff (broadcast)
  │   - Contains: Service-Name tag (requested ISP), Host-Uniq (correlation)
  │   - Sent on all interfaces; multiple ACs may respond
  │
  ▼
State: PADI_SENT
  │
  ├── AC sends PADO (PPPoE Active Discovery Offer)
  │   - Destination: CPE MAC (unicast)
  │   - Contains: AC-Name, Service-Name (confirmed), AC-Cookie (anti-DoS)
  │   - Multiple ACs may respond; CPE picks one
  │
  ▼
State: PADO_RECEIVED (CPE selects AC)
  │
  ├── CPE sends PADR (PPPoE Active Discovery Request)
  │   - Destination: selected AC MAC (unicast)
  │   - Contains: Service-Name, Host-Uniq, AC-Cookie (echoed back)
  │   - Commits to specific AC
  │
  ▼
State: PADR_SENT
  │
  ├── AC sends PADS (PPPoE Active Discovery Session-Confirmation)
  │   - Contains: Session-ID (16-bit, nonzero, unique per AC-MAC + CPE-MAC pair)
  │   - Session is now established at Layer 2
  │
  ▼
State: SESSION_ACTIVE (EtherType switches to 0x8864 for session data)
```

### Session Phase (Layer 3 Negotiation)

Once PPPoE session is active, PPP runs inside the PPPoE tunnel:

1. **LCP (Link Control Protocol):** Negotiates MRU, authentication method (CHAP/PAP), magic number (loop detection)
2. **Authentication:** CHAP challenge-response or PAP cleartext (BNG proxies to RADIUS)
3. **IPCP/IPv6CP:** Negotiates IPv4 address (or IPv6 prefix via SLAAC/DHCPv6-PD)

### Session Teardown

Either side can send PADT (PPPoE Active Discovery Terminate):
- CPE logout, link failure, idle timeout, admin disconnect (CoA), session timeout
- PADT is a single frame; no acknowledgment required
- BNG sends RADIUS Accounting-Stop with Acct-Terminate-Cause

### MTU Implications

PPPoE adds an 8-byte header (6-byte PPPoE + 2-byte PPP protocol field) inside the Ethernet frame:

$$\text{PPPoE MTU} = \text{Ethernet MTU} - 8 = 1500 - 8 = 1492 \text{ bytes}$$

This means TCP MSS must be clamped:

$$\text{TCP MSS} = \text{PPPoE MTU} - 20\text{ (IP)} - 20\text{ (TCP)} = 1452 \text{ bytes}$$

With baby jumbo frames (1508 Ethernet payload), full 1500 MTU is restored inside PPPoE. This requires end-to-end support across the access network.

---

## 3. IPoE Session Lifecycle

### DHCP-Triggered Session Creation

IPoE eliminates PPP overhead. The subscriber session is triggered by DHCP:

```
CPE ──> Access Node ──> Aggregation ──> BNG ──> RADIUS
         (inserts          (L2/L3          (creates      (authenticates
          Option 82)        transport)      session)      subscriber)
```

**Session identification** in IPoE relies on one or more of:
- **DHCP Option 82** (Relay Agent Information): Circuit-ID (port/VLAN on access node) + Remote-ID (access node identifier)
- **Source MAC address**
- **VLAN tag(s)** (outer S-VLAN for service, inner C-VLAN for customer)

### IPoE vs PPPoE Trade-offs

| Factor | PPPoE | IPoE |
|--------|-------|------|
| Authentication | Strong (PAP/CHAP via PPP) | Weak (MAC/Option 82/802.1X) |
| Session awareness | Explicit (PPP state machine) | Implicit (DHCP lease) |
| MTU overhead | 8 bytes (1492 MTU) | None (1500 MTU) |
| CPE complexity | PPPoE client required | Standard DHCP |
| IPv6 | IPv6CP or DHCPv6 over PPP | Native SLAAC/DHCPv6 |
| Roaming/mobility | Session must renegotiate | DHCP rebind possible |
| Troubleshooting | PPPoE state visible | Session state inferred from DHCP |

### The Trend

Most greenfield FTTH deployments use IPoE. PPPoE remains dominant in DSL markets (Germany, France, Japan, Australia) due to legacy CPE, strong authentication requirements, or regulatory reasons.

---

## 4. RADIUS-Based Policy Enforcement

### The Policy Pipeline

When a subscriber authenticates, RADIUS returns attributes that the BNG translates into forwarding-plane policy:

```
RADIUS Access-Accept
    │
    ├── Framed-IP-Address / Framed-Pool ──> IP assignment
    ├── Framed-IPv6-Prefix ──────────────> IPv6 /64 assignment
    ├── Delegated-IPv6-Prefix ───────────> IA_PD (e.g., /48 or /56)
    ├── Cisco-AVPair sub-qos-policy ─────> QoS template activation
    ├── Cisco-AVPair ip:inacl/outacl ────> ACL application
    ├── Cisco-AVPair subscriber:sa ──────> Service activation (stacked)
    ├── Cisco-AVPair lcp:interface-config > VRF, MTU, other interface config
    ├── Session-Timeout ─────────────────> Maximum session duration
    ├── Idle-Timeout ────────────────────> Inactivity disconnect timer
    └── Acct-Interim-Interval ───────────> Accounting update frequency
```

### Service Activation Model

Modern BNG uses **dynamic templates** activated via RADIUS. Multiple services can be stacked on a single session:

```
Subscriber Session
    ├── Base template: PPP-DEFAULT (LCP, auth, IP)
    ├── Service 1: INTERNET-100M (QoS shaping, policing)
    ├── Service 2: VOIP-SERVICE (priority queuing)
    └── Service 3: IPTV-MULTICAST (IGMP proxy, multicast QoS)
```

Services can be added or removed mid-session via CoA without disrupting the subscriber's connectivity.

### CoA Use Cases

| Operation | RADIUS Attributes | Effect |
|-----------|------------------|--------|
| Speed upgrade | sub-qos-policy-in/out change | New QoS applied instantly |
| Service add | subscriber:sa=NEW-SERVICE | Template stacked |
| Service remove | subscriber:sa=-OLD-SERVICE | Template removed |
| Redirect (walled garden) | ip:inacl=REDIRECT-ACL | HTTP redirect for captive portal |
| Disconnect | Disconnect-Request | Session terminated |
| IP change | Framed-IP-Address change | Requires session restart |

---

## 5. Subscriber Scaling

### Session Density Calculations

BNG platforms are rated by concurrent subscriber sessions. Key metrics:

$$\text{Sessions per line card} = \frac{\text{Line card memory for session state}}{\text{Memory per session}}$$

Typical memory per subscriber session:

| Component | Memory |
|-----------|--------|
| Session control block | 2-4 KB |
| QoS policy instance | 1-2 KB |
| ACL instance | 0.5-1 KB |
| Accounting state | 0.5 KB |
| IPv4 + IPv6 state | 1-2 KB |
| **Total per session** | **5-10 KB** |

For a line card with 4 GB subscriber memory:

$$\text{Max sessions} = \frac{4 \times 10^9}{8 \times 10^3} = 500{,}000 \text{ sessions}$$

In practice, 64K-128K sessions per line card is typical due to TCAM, forwarding ASIC, and QoS hardware constraints.

### RADIUS Transaction Rate

Each subscriber session generates RADIUS transactions:

$$\text{RADIUS TPS} = \frac{N_{sessions}}{T_{avg\_session}} + \frac{N_{active}}{T_{interim}}$$

Where:
- $N_{sessions}/T_{avg\_session}$ = session setup/teardown rate (auth + acct start/stop)
- $N_{active}/T_{interim}$ = interim accounting updates

**Example:** 100K active sessions, 4-hour average session, 15-minute interim interval:

$$\text{Setup rate} = \frac{100{,}000}{4 \times 3600} \approx 7 \text{ sessions/sec}$$

$$\text{Interim rate} = \frac{100{,}000}{15 \times 60} \approx 111 \text{ updates/sec}$$

$$\text{Total RADIUS TPS} \approx 7 \times 4 + 111 \approx 139 \text{ TPS}$$

(Factor of 4 for auth request/response + acct start + acct stop per session lifecycle)

### Control Plane Protection

The BNG control plane is vulnerable to subscriber-generated protocol storms:

| Attack Vector | Protection |
|---------------|-----------|
| PPPoE PADI flood | Per-VLAN PADI rate limiting in BBA group |
| DHCP Discover flood | DHCP rate limiting, Option 82 validation |
| ARP storm | ARP rate limiting, proxy ARP |
| IGMP flood | IGMP rate limiting per subscriber |
| CPE reboot storm | Session throttle (delay between session teardown and re-establishment) |

Rate limiting formula for PADI:

$$\text{Max PADI rate} = \frac{\text{BNG control CPU capacity (sessions/sec)}}{\text{Safety factor (typically 2-4)}}$$

---

## 6. BNG Redundancy

### Session Recovery (Stateful Switchover)

For chassis-level redundancy (active/standby RSP/RP):

1. **Session state checkpointing:** Active RP replicates session state to standby RP
2. **Checkpointed state:** Session ID, IP address, QoS policy, accounting counters, RADIUS session attributes
3. **Not checkpointed:** In-flight packets, partial PPP negotiation, RADIUS transactions in progress

Recovery time:

$$T_{recovery} = T_{detect} + T_{switchover} + T_{replay}$$

Typical values:
- $T_{detect}$: 1-3 seconds (BFD or hardware watchdog)
- $T_{switchover}$: 2-5 seconds (standby RP takes over)
- $T_{replay}$: 0 seconds (state already replicated)
- **Total: 3-8 seconds** (subscriber sees brief traffic loss, no re-authentication)

### Geographic Redundancy (Dual-Homed BNG)

Two BNGs serve the same subscriber population, typically via:

```
                    ┌─────────┐     ┌─────────┐
                    │  BNG-A  │     │  BNG-B  │
                    │ (Active)│     │(Standby)│
                    └────┬────┘     └────┬────┘
                         │               │
                    ┌────┴───────────────┴────┐
                    │   Aggregation Switch     │
                    │   (MC-LAG / dual-homed)  │
                    └─────────────────────────┘
```

**Approaches:**

| Method | How It Works | Session Impact |
|--------|-------------|---------------|
| MC-LAG + session sync | Both BNGs active, sessions synced via IPC | Subsecond failover, no re-auth |
| VRRP + cold standby | Standby BNG takes VIP on failure | All sessions re-establish (30-120 sec) |
| Access-node dual-homing | Access node has links to both BNGs | Controlled by access node LAG |
| Subscriber-initiated | CPE re-sends PADI/DHCP Discover | Full session re-establishment |

### Scaling Redundancy Overhead

Session synchronization has a cost:

$$\text{Sync bandwidth} = N_{sessions} \times S_{state} \times \frac{1}{T_{sync}}$$

Where:
- $N_{sessions}$ = number of active sessions
- $S_{state}$ = session state size (5-10 KB)
- $T_{sync}$ = sync interval (1-5 seconds)

**Example:** 100K sessions, 8 KB state, 2-second sync interval:

$$\text{Sync BW} = \frac{100{,}000 \times 8{,}000}{2} = 400 \text{ MB/s}$$

This is why BNG session sync typically uses incremental updates (only changed sessions), reducing bandwidth by 95-99%.

---

## 7. Disaggregated BNG (D-BNG)

### The Emerging Architecture

Traditional BNG is a monolithic chassis. Disaggregated BNG separates control and user planes:

```
┌──────────────────────┐
│  BNG Control Plane   │  ◄── Subscriber management, AAA, policy
│  (Virtual / Server)  │      CUPS model (TR-459)
└──────────┬───────────┘
           │ (PFCP or proprietary)
┌──────────┴───────────┐
│  BNG User Plane      │  ◄── Packet forwarding, QoS, NAT
│  (White-box / ASIC)  │      Bare-metal switch or SmartNIC
└──────────────────────┘
```

**Benefits:** Independent scaling of control and user planes, vendor flexibility, cloud-native control plane. **Challenges:** Immature ecosystem, inter-plane protocol standardization (BBF TR-459 CUPS), session state distribution.

---

## See Also

- bgp, mpls, radius, cgnat, ipv4, ipv6, sp-multicast

## References

- [RFC 2516 — A Method for Transmitting PPP Over Ethernet (PPPoE)](https://www.rfc-editor.org/rfc/rfc2516)
- [RFC 1661 — The Point-to-Point Protocol (PPP)](https://www.rfc-editor.org/rfc/rfc1661)
- [RFC 1332 — The PPP Internet Protocol Control Protocol (IPCP)](https://www.rfc-editor.org/rfc/rfc1332)
- [RFC 2865 — Remote Authentication Dial In User Service (RADIUS)](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 2866 — RADIUS Accounting](https://www.rfc-editor.org/rfc/rfc2866)
- [RFC 5176 — Dynamic Authorization Extensions to RADIUS](https://www.rfc-editor.org/rfc/rfc5176)
- [RFC 3046 — DHCP Relay Agent Information Option](https://www.rfc-editor.org/rfc/rfc3046)
- [BBF TR-459 — Control and User Plane Separation for BNG](https://www.broadband-forum.org/technical/download/TR-459.pdf)
- [BBF TR-101 — Migration to Ethernet-Based Broadband Aggregation](https://www.broadband-forum.org/technical/download/TR-101.pdf)
