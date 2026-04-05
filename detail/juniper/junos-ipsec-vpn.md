# JunOS IPsec VPN — Architecture, Protocol Internals, and Scaling Analysis

> *JunOS IPsec VPN on SRX operates within the flow-based security framework, binding encrypted tunnels to either virtual tunnel interfaces (route-based) or security policies (policy-based). Understanding the architectural differences, IKE exchange mechanics, AutoVPN/ADVPN dynamic tunnel lifecycle, and VPN failover mechanisms is critical for JNCIE-SEC design scenarios.*

---

## 1. Route-Based vs Policy-Based Architecture

### Route-Based VPN (st0)

Route-based VPN creates a virtual tunnel interface (st0.x) that acts as the IPsec tunnel endpoint. All traffic routed to the st0 interface is encrypted and sent through the tunnel.

```
                    Routing Table
                    ┌──────────────────────┐
                    │ 172.16.0.0/16 → st0.0│
                    │ 10.0.0.0/8 → ge-0/0/0│
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │    st0.0 (tunnel)     │
                    │  ┌─────────────────┐  │
                    │  │ IPsec encrypt   │  │
                    │  │ ESP encapsulate │  │
                    │  └────────┬────────┘  │
                    └───────────┼───────────┘
                    ┌───────────▼───────────┐
                    │  ge-0/0/1.0 (physical)│
                    │  → Peer 198.51.100.1  │
                    └───────────────────────┘
```

Key properties:

1. **Traffic selection is routing-based** — anything routed to st0.x enters the tunnel. This means dynamic routing protocols (OSPF, BGP, IS-IS) can run over the tunnel, and traffic selection changes with route table changes.

2. **One st0 unit per tunnel** (point-to-point mode) or **one st0 unit for multiple tunnels** (multipoint mode for hub-and-spoke/ADVPN).

3. **Security zone assignment** — st0.x is placed in a security zone, enabling zone-based policies for tunnel traffic. This provides granular control over what traverses the VPN.

4. **NAT compatibility** — traffic is routed to st0 naturally, so NAT exclusion rules simply avoid NATing traffic destined for the tunnel (by matching the route).

5. **Redundancy** — multiple st0 interfaces can provide tunnel redundancy with routing failover (e.g., two tunnels to different peers, OSPF detects failure and shifts routes).

### Policy-Based VPN

Policy-based VPN triggers IPsec directly from a security policy action:

```
Security Policy Match                 IPsec Processing
┌────────────────────────┐           ┌──────────────────┐
│ src: 10.0.0.0/8        │ ────→    │ Encrypt with SA   │
│ dst: 172.16.0.0/12     │           │ matching this     │
│ then: permit + tunnel  │           │ proxy-id pair     │
│   ipsec-vpn SITE-B     │           └──────────────────┘
└────────────────────────┘
```

Key properties:

1. **Traffic selection is policy-based** — only traffic matching the security policy enters the tunnel. The match criteria (src/dst address, application) become the proxy-id (traffic selector) in the IKE negotiation.

2. **One SA pair per policy match** — if you have three policies directing traffic to the same VPN, you get three separate IPsec SA pairs. This can cause scaling issues.

3. **No st0 interface** — no tunnel interface in any zone. The VPN traffic is treated as transit traffic between the source and destination zones in the policy.

4. **No dynamic routing over tunnel** — because there is no tunnel interface, routing protocols cannot run over the VPN.

5. **Pair-policy required** — each policy needs a matching reverse policy (pair-policy) for bidirectional traffic.

### When to Use Which

```
Route-based (preferred for):        Policy-based (use only for):
├── Dynamic routing over VPN        ├── Interop with legacy devices
├── Hub-and-spoke / ADVPN           │   that only support proxy-id
├── Multiple traffic flows in one   ├── Simple single-subnet pairs
│   tunnel                          └── Specific compliance
├── NAT + VPN combination           │   requirements
├── Multicast over VPN
├── Chassis cluster + VPN
└── AutoVPN with dynamic spokes
```

---

## 2. IKEv1/IKEv2 Exchange in JunOS

### IKEv1 Main Mode (6 Messages)

```
Initiator                                  Responder
    │                                          │
    │──── SA proposal (encryption, auth,  ────→│  Message 1
    │     hash, DH group, lifetime)            │
    │                                          │
    │←─── SA selection (chosen proposal)  ─────│  Message 2
    │                                          │
    │──── KE (DH public value) +          ────→│  Message 3
    │     Nonce                                │
    │                                          │
    │←─── KE (DH public value) +          ─────│  Message 4
    │     Nonce                                │
    │     [Keys derived: SKEYID, SKEYID_d,     │
    │      SKEYID_a, SKEYID_e]                 │
    │                                          │
    │──── ID + Auth hash (encrypted)      ────→│  Message 5
    │                                          │
    │←─── ID + Auth hash (encrypted)      ─────│  Message 6
    │                                          │
    │     IKE SA established                   │
```

Main mode protects identities (messages 5-6 are encrypted). Requires known peer IP.

### IKEv1 Aggressive Mode (3 Messages)

```
Initiator                                  Responder
    │                                          │
    │──── SA + KE + Nonce + ID           ────→│  Message 1
    │                                          │
    │←─── SA + KE + Nonce + ID + Auth    ─────│  Message 2
    │                                          │
    │──── Auth hash                       ────→│  Message 3
    │                                          │
    │     IKE SA established                   │
```

Aggressive mode exposes identities in cleartext (message 1 ID is unencrypted). Used when the initiator has a dynamic IP and the responder must identify it by ID (hostname, email, DN).

### IKEv2 Exchange (4 Messages for IKE_SA_INIT + IKE_AUTH)

```
Initiator                                  Responder
    │                                          │
    │──── IKE_SA_INIT:                    ────→│  Message 1
    │     SA, KE, Nonce                        │
    │                                          │
    │←─── IKE_SA_INIT:                    ─────│  Message 2
    │     SA, KE, Nonce                        │
    │     [Keys derived from shared secret]    │
    │                                          │
    │──── IKE_AUTH: (encrypted)           ────→│  Message 3
    │     ID, Auth, SA, TSi, TSr               │
    │                                          │
    │←─── IKE_AUTH: (encrypted)           ─────│  Message 4
    │     ID, Auth, SA, TSi, TSr               │
    │                                          │
    │     IKE SA + first Child SA established  │
```

IKEv2 advantages over IKEv1:

1. **Fewer messages** — 4 messages establish IKE SA + first child SA (vs 6+3=9 in IKEv1)
2. **Built-in NAT-T** — NAT detection in IKE_SA_INIT, automatic UDP encapsulation
3. **EAP support** — extensible authentication (RADIUS, certificates, etc.)
4. **MOBIKE (RFC 4555)** — VPN survives IP address changes (mobile clients)
5. **Multiple child SAs** — CREATE_CHILD_SA exchange for additional SAs without re-authenticating
6. **Traffic selectors** — narrowing negotiation for flexible traffic selection

### JunOS IKE Implementation Details

JunOS uses the `kmd` daemon (key management daemon) for IKE negotiation. The kmd runs on the RE and communicates with the PFE to install SA state:

```
RE (kmd daemon)                          PFE (data plane)
├── IKE negotiation                      ├── ESP encryption/decryption
├── SA lifetime management               ├── Anti-replay window
├── DPD probes (IKEv1) or               ├── SA selector matching
│   liveness checks (IKEv2)             ├── Tunnel statistics
├── Rekey initiation                     └── Fragment reassembly
└── Certificate validation
```

---

## 3. AutoVPN Dynamic Tunnel Creation

### Architecture

AutoVPN allows the hub to accept VPN connections from spokes without pre-configuring each spoke individually. The hub uses a dynamic gateway with group IKE identity.

```
Hub                          Spoke 1            Spoke 2           Spoke N
┌──────────┐                ┌─────┐            ┌─────┐           ┌─────┐
│ AutoVPN  │ ←── IKE ────  │     │            │     │           │     │
│ gateway  │ ←── IKE ──────────────────────── │     │           │     │
│ (dynamic)│ ←── IKE ────────────────────────────────────────── │     │
│          │                │     │            │     │           │     │
│ st0.0    │                │st0.0│            │st0.0│           │st0.0│
│(multipt) │                │     │            │     │           │     │
└──────────┘                └─────┘            └─────┘           └─────┘
```

### Tunnel Lifecycle

1. **Spoke initiates IKE** — sends IKE_SA_INIT to hub's public IP
2. **Hub matches dynamic gateway** — spoke's IKE identity matches the group pattern
3. **IKE_AUTH completes** — hub creates IKE SA for this spoke
4. **Child SA created** — traffic selectors negotiated (spoke's local/remote subnets)
5. **st0.0 learns next-hop** — hub's st0.0 (multipoint) adds spoke as a next-hop tunnel binding
6. **Routes installed** — traffic selectors or dynamic routing populate routes to spoke via st0.0

### Dynamic vs Static Configuration

```
Static (per-peer):                    Dynamic (AutoVPN):
├── N gateways for N spokes           ├── 1 gateway for all spokes
├── N st0 units                       ├── 1 st0 unit (multipoint)
├── N VPN configs                     ├── 1 VPN config
├── N sets of routes                  ├── Routes auto-created
└── Config grows linearly             └── Config is constant
```

---

## 4. ADVPN Shortcut Switching

### The Problem ADVPN Solves

In hub-and-spoke VPN, all spoke-to-spoke traffic transits the hub:

```
Spoke A → Hub → Spoke B     (inefficient for A↔B traffic)
```

ADVPN (Auto Discovery VPN) creates direct spoke-to-spoke tunnels on demand:

```
Spoke A → Hub → Spoke B     (initial packets, triggers shortcut)
Spoke A ─────→ Spoke B      (shortcut tunnel, direct)
```

### Shortcut Creation Process

```
1. Spoke A sends traffic to Spoke B's subnet
   └─ Traffic routed to st0.0 → hub tunnel

2. Hub receives packet, forwards to Spoke B
   └─ Hub detects this is spoke-to-spoke traffic
   └─ Hub sends NHRP redirect to Spoke A with Spoke B's public IP

3. Spoke A receives NHRP redirect
   └─ Spoke A initiates IKE directly to Spoke B

4. Spoke A ↔ Spoke B IKE + IPsec negotiation completes
   └─ Direct tunnel established

5. Routing updated
   └─ Spoke A installs route to Spoke B's subnet via direct tunnel
   └─ Traffic bypasses hub

6. Idle timeout
   └─ If no traffic for idle-threshold seconds, shortcut torn down
   └─ Traffic reverts to hub-and-spoke path
```

### NHRP (Next Hop Resolution Protocol)

ADVPN uses NHRP for address resolution — mapping spoke subnets to spoke public IPs:

```
NHRP Server (Hub):
├── Maintains mapping: spoke-subnet → spoke-public-IP
├── Responds to resolution requests
├── Sends redirect notifications for shortcut opportunities
└── Distributes routes (or works with dynamic routing)

NHRP Client (Spoke):
├── Registers its subnet → public-IP mapping with hub
├── Sends resolution requests for unknown destinations
├── Acts on redirect notifications (creates shortcuts)
└── Reports shortcut establishment/teardown
```

---

## 5. Group VPN Architecture

### Concept

Group VPN (GVPNv2) differs fundamentally from traditional point-to-point IPsec:

```
Traditional IPsec:                    Group VPN:
├── Tunnel mode (new IP header)       ├── Transport mode (original header)
├── Point-to-point SAs                ├── Group SA (shared by all members)
├── N*(N-1)/2 tunnels for full mesh   ├── 1 SA set for entire group
├── Hides original IP headers         ├── Preserves original IP headers
└── Each pair negotiates separately   └── Key server distributes keys
```

### Architecture Components

```
Key Server (KS)
├── Generates group SA (TEK — Traffic Encryption Key)
├── Distributes TEK to all members via GDOI (RFC 6407)
├── Manages rekey (unicast or multicast rekey)
├── Defines encryption policy (which traffic to encrypt)
└── Anti-replay: time-based (not sequence-number-based)

Group Member (GM)
├── Registers with KS via IKE Phase 1
├── Receives TEK and encryption policy via GDOI
├── Encrypts matching traffic with group SA
├── Decrypts traffic from any other group member
└── All members share the same SA (symmetric)
```

### Why Preserve Original Headers

Group VPN uses transport mode, keeping original source and destination IP headers visible:

- Network monitoring tools see real source/destination IPs
- QoS policies based on IP addresses still function
- Routing is based on original destination (no tunnel routing needed)
- Firewall policies in the middle can still match on original IPs
- Multicast works without GRE encapsulation

### Rekey Process

```
1. TEK lifetime approaching expiration
2. KS generates new TEK
3. KS sends rekey message to all members
   ├── Unicast rekey: individual message to each member
   └── Multicast rekey: single message to group address
4. Members install new TEK
5. Overlap period: old and new TEK both valid
6. Old TEK expires
```

---

## 6. VPN Scaling Considerations

### Tunnel Scaling by Platform

```
Platform        Max IKE SAs    Max IPsec SAs    Max VPN Tunnels
SRX300          256            512              256
SRX320          256            512              256
SRX340          2K             2K               1K
SRX345          2K             4K               2K
SRX1500         6K             6K               3K
SRX4100         10K            20K              10K
SRX4200         10K            20K              10K
SRX4600         15K            30K              15K
SRX5400         20K            40K              20K
SRX5600         20K            40K              20K
SRX5800         20K            40K              20K
```

### Throughput with IPsec

IPsec encryption significantly reduces throughput compared to firewall-only:

- **Software crypto** — RE-based, very slow, used only when no hardware offload
- **Hardware crypto** — NPU/SPU-based, line-rate for common ciphers (AES-128/256-CBC, AES-GCM)
- **AES-GCM** — significantly faster than AES-CBC+HMAC on hardware-accelerated platforms (combined mode)

Typical throughput reduction with IPsec:

```
AES-256-GCM:     60-80% of firewall-only throughput
AES-256-CBC+SHA: 40-60% of firewall-only throughput
3DES-CBC+SHA:    20-40% of firewall-only throughput
```

### IKE Rekey Storms

In large-scale VPN deployments (hundreds of tunnels), simultaneous rekey can cause CPU spikes on the RE (kmd daemon):

- Stagger SA lifetimes with random jitter (JunOS applies some jitter automatically)
- Use IKEv2 (more efficient rekeying than IKEv1)
- Monitor `show system processes extensive | grep kmd` during rekey windows
- Consider longer lifetimes for stable tunnels to reduce rekey frequency

---

## 7. VPN Failover Mechanisms

### Chassis Cluster VPN Failover

```
Normal operation:
  Node 0 (primary RG1) → reth0 (active) → IKE/IPsec SAs active
  Node 1 (secondary)   → reth0 (standby) → IKE/IPsec SAs synced

Failover trigger:
  Node 0 interface failure → RG1 failover to Node 1

Failover sequence:
  1. RG1 priority recalculated → Node 1 becomes primary
  2. reth0 active member switches to Node 1's physical interface
  3. Gratuitous ARP sent for reth0 IP
  4. Synced IKE SA activated on Node 1
  5. Synced IPsec SAs activated on Node 1
  6. Traffic resumes through Node 1

Recovery time:
  ├── Sub-second for pre-synced SAs
  ├── 1-3 seconds including GARP convergence
  └── Full IKE re-negotiation if SA sync failed (~2-5 seconds for IKEv2)
```

### Dual-Tunnel Redundancy (Without Chassis Cluster)

```
# Two tunnels to different peers (or same peer, different paths)
set interfaces st0 unit 0 family inet address 10.255.0.1/30    # tunnel 1
set interfaces st0 unit 1 family inet address 10.255.1.1/30    # tunnel 2

# OSPF over both tunnels — detects failure and reroutes
set protocols ospf area 0 interface st0.0 metric 10
set protocols ospf area 0 interface st0.1 metric 20

# Or BGP with local-preference
set protocols bgp group VPN-PRIMARY neighbor 10.255.0.2 local-preference 200
set protocols bgp group VPN-BACKUP neighbor 10.255.1.2 local-preference 100
```

### VPN Monitoring Failover

```
# ICMP-based monitoring detects tunnel blackhole
# (IKE SA up but IPsec data path broken)
set security ipsec vpn SITE-B vpn-monitor source-interface ge-0/0/0.0
set security ipsec vpn SITE-B vpn-monitor destination-ip 10.255.0.2
set security ipsec vpn SITE-B vpn-monitor optimized

# On monitoring failure:
# 1. IPsec SA torn down
# 2. IKE SA torn down
# 3. Routes via st0.x withdrawn
# 4. Routing converges to backup path
# 5. IKE re-negotiation attempted
```

### DPD vs VPN Monitoring

```
DPD (Dead Peer Detection):
├── IKE-level check — verifies IKE peer is alive
├── Does NOT verify IPsec data path
├── Probe: IKE informational message
├── Failure action: tear down IKE SA
└── Good for: detecting peer device failure

VPN Monitoring:
├── Data-plane check — verifies end-to-end IPsec path
├── Verifies actual encrypted data delivery
├── Probe: ICMP ping through the tunnel
├── Failure action: tear down IPsec + IKE SAs
└── Good for: detecting path failures, blackholes, MTU issues
```

Both should be enabled together for comprehensive failure detection.
