# NX-OS Security — Defense in Depth for Data Center Switches

> Cisco NX-OS provides a layered security architecture purpose-built for data center
> environments. From AAA frameworks that centralize identity management, through RBAC
> that enforces least-privilege operations, to first-hop security features that protect
> the access layer from L2/L3 spoofing attacks — NX-OS treats security as a stack of
> interlocking controls rather than a single perimeter. This document examines each
> layer in depth: the theory behind it, the failure modes it prevents, and how the
> features compose into a coherent defense-in-depth posture.

---

## 1. AAA Framework — Authentication, Authorization, and Accounting

### 1.1 The AAA Model

AAA is the foundational security framework for network device access. It answers
three questions in sequence:

- **Authentication** — "Who are you?" Validates the identity of a user or process
  attempting to access the device.
- **Authorization** — "What can you do?" Determines the set of commands, features,
  and configurations the authenticated identity is permitted to invoke.
- **Accounting** — "What did you do?" Records session activity, command history,
  and resource consumption for audit trails and forensic analysis.

NX-OS implements AAA as a modular pipeline. Each phase can independently reference
different server groups, fall back to local databases, or chain multiple methods.
This separation is critical: authentication can use RADIUS (which is strong at
credential validation) while authorization uses TACACS+ (which provides per-command
authorization granularity that RADIUS lacks).

### 1.2 TACACS+ vs. RADIUS

| Property               | TACACS+                          | RADIUS                          |
|------------------------|-----------------------------------|---------------------------------|
| Transport              | TCP port 49                      | UDP ports 1812/1813             |
| Encryption             | Full packet body encrypted       | Only password field encrypted   |
| AAA separation         | Independent A, A, A phases       | Combined authentication+authz   |
| Command authorization  | Per-command granularity           | Not natively supported          |
| Accounting             | Start/stop/interim records       | Start/stop records              |
| Multiprotocol          | No (device admin only)           | Yes (802.1X, VPN, wireless)     |

TACACS+ is the preferred protocol for device administration on NX-OS because it
encrypts the entire packet body (not just the password), supports per-command
authorization (critical for RBAC enforcement via external policy), and cleanly
separates all three AAA functions. RADIUS remains essential for 802.1X network
access control and is commonly used when the same infrastructure serves both
device admin and network access authentication.

### 1.3 Local Authentication as Fallback

Remote AAA servers introduce a single point of failure. If both TACACS+ servers
are unreachable — due to a network partition, server crash, or control plane
issue — administrators are locked out of the device at the exact moment they
most need access (during an outage).

NX-OS solves this with method lists. The configuration
`aaa authentication login default group tacacs+ local` means: try the TACACS+
group first; if every server in the group is unreachable (timeout or connection
refused), fall back to the local user database. This is not the same as "if
authentication fails" — a rejected credential from TACACS+ does not trigger
fallback. Only server unreachability does.

Best practice: always maintain at least two local accounts — one `network-admin`
for emergency access and one `network-operator` for read-only triage — with
strong passwords rotated on a schedule independent of the AAA server.

### 1.4 Accounting and Audit Trails

Accounting records are non-negotiable for compliance frameworks (PCI-DSS
Requirement 10, SOC 2 CC6.1, NIST AC-2). NX-OS sends accounting records to
the configured server group for every command executed, session opened, and
session closed. Key record types:

- **Start** — User authenticated and session began.
- **Stop** — Session ended (includes duration, bytes transferred).
- **Interim** — Periodic update during long-lived sessions.
- **Command** — Individual command executed (TACACS+ only).

Command accounting via TACACS+ creates a complete audit trail of every
configuration change: who ran the command, when, from which source IP, and
in which VRF context.

### 1.5 AAA Server Deadtime and Failover

When an AAA server fails to respond, NX-OS marks it as dead for a configurable
`deadtime` period (default 0 minutes — meaning it retries every time). Setting
`tacacs-server deadtime 5` means the switch skips the dead server for 5 minutes
before probing it again. This dramatically reduces login latency during server
outages because the switch immediately contacts the next server in the group
rather than waiting for a timeout on the dead one.

The interaction between `timeout`, `deadtime`, and method-list fallback is
subtle. If timeout is 10 seconds and there are 2 servers, a login attempt
against a fully dead AAA infrastructure takes 20 seconds before local fallback
engages. Reducing timeout to 5 seconds and enabling deadtime cuts this to
5 seconds on the first failure and near-instant on subsequent attempts.

---

## 2. RBAC — Role-Based Access Control

### 2.1 The Principle of Least Privilege

RBAC on NX-OS implements the principle of least privilege at the CLI and API
level. Rather than granting blanket access to every command, administrators
receive only the permissions required for their operational responsibilities.
This limits the blast radius of compromised credentials and reduces the risk
of accidental misconfiguration.

### 2.2 Role Architecture

An NX-OS role consists of:

- **Rules** — Ordered entries (evaluated top-down, first match wins) that
  permit or deny access to commands, features, or feature groups.
- **VLAN Policy** — Optional restriction limiting the role to a subset of VLANs.
- **Interface Policy** — Optional restriction limiting the role to a subset of
  interfaces.

Rule types:

| Rule Type      | Scope                                           |
|----------------|--------------------------------------------------|
| Command rule   | Matches a specific CLI command or wildcard       |
| Feature rule   | Matches all commands within an NX-OS feature     |
| Feature-group  | Matches all commands across a group of features  |
| Read/Read-Write| Determines whether the match grants show-only or full access |

### 2.3 Built-in Roles

NX-OS ships with predefined roles:

- **network-admin** — Unrestricted. Can execute any command on the device.
  Equivalent to privilege level 15 on IOS. Should be reserved for break-glass
  emergency access.
- **network-operator** — Read-only. Can execute `show` commands but cannot
  make configuration changes. Ideal for NOC monitoring accounts.
- **vdc-admin / vdc-operator** — Scoped versions of the above for Virtual
  Device Contexts on Nexus 7000 series. These roles cannot affect other VDCs.

### 2.4 User-Defined Roles

Custom roles are where RBAC becomes powerful. Consider a tiered operational
model:

- **L1 NOC** — `show` commands only, no configuration, no reload. Uses the
  `network-operator` role or a clone with further restrictions.
- **L2 Network Engineer** — Can modify interfaces, VLANs, and ACLs, but
  cannot modify routing protocols, reload the device, or erase the config.
- **Security Admin** — Can modify ACLs, DHCP snooping, DAI, and port security,
  but cannot touch routing or VLAN configurations.
- **Routing Admin** — Can modify OSPF, BGP, EIGRP, and static routes, but
  cannot modify security features or VLANs.

This separation means that a compromised "Routing Admin" account cannot disable
DHCP snooping or modify ACLs — the attacker's lateral movement is constrained
by the role boundary even after credential theft.

### 2.5 Feature-Based Rules

Feature-based rules are more maintainable than command-based rules. When Cisco
adds new commands to a feature in a software upgrade, feature-based rules
automatically cover them. Command-based rules require manual updates. The
tradeoff is granularity: command rules can distinguish between `show ip route`
and `clear ip route`, while feature rules grant blanket access to everything
under the `routing` feature.

### 2.6 RBAC and TACACS+ Integration

When TACACS+ is configured for command authorization, NX-OS sends each command
to the TACACS+ server before execution. The server's authorization policy can
supplement or override the local RBAC role. This creates a two-layer
authorization model:

1. Local RBAC role determines the baseline permissions.
2. TACACS+ command authorization can further restrict (or, controversially,
   expand) those permissions.

The interaction can be surprising: if the local role permits a command but
TACACS+ denies it, the command is denied. If the local role denies a command,
NX-OS never sends it to TACACS+ — it is denied locally before the server is
consulted.

---

## 3. DHCP Snooping

### 3.1 The Attack: DHCP Spoofing

In an unsecured Layer 2 network, any host can respond to DHCP Discover messages
and pose as a DHCP server. A rogue DHCP server can assign clients a default
gateway that routes traffic through the attacker's machine (man-in-the-middle),
assign DNS servers that resolve names to malicious IPs (DNS hijacking), or
simply assign invalid addresses to cause denial of service.

### 3.2 How DHCP Snooping Works

DHCP snooping classifies switch ports into two categories:

- **Trusted ports** — Connected to legitimate DHCP servers or upstream network
  infrastructure. DHCP server messages (Offer, Ack, Nak) are permitted.
- **Untrusted ports** — Connected to end hosts. DHCP server messages received
  on these ports are dropped. Only DHCP client messages (Discover, Request,
  Release, Decline) are permitted.

As the switch observes DHCP transactions on untrusted ports, it builds a
**binding table** that maps IP addresses to MAC addresses, VLANs, and
interfaces. This binding table becomes the authoritative source of truth for
two downstream features: DAI and IPSG.

### 3.3 Binding Table Mechanics

Each entry in the DHCP snooping binding table contains:

| Field       | Source                                    |
|-------------|-------------------------------------------|
| MAC address | DHCP client hardware address field        |
| IP address  | Assigned IP from DHCP Ack                 |
| VLAN        | VLAN of the receiving interface           |
| Interface   | Physical port where the DHCP exchange occurred |
| Lease time  | DHCP lease duration                       |

The binding table is stored in memory by default. On a switch reload, all
entries are lost, and clients must re-DHCP to rebuild them. During this
reconvergence window, DAI and IPSG will drop legitimate traffic because they
have no bindings to validate against. Persisting the table to bootflash
(`ip dhcp snooping database bootflash:dhcp_snoop_db`) eliminates this window.

### 3.4 Rate Limiting

Untrusted ports should have DHCP rate limiting enabled to prevent DHCP
starvation attacks, where an attacker floods DHCP Discover messages to
exhaust the server's address pool. The rate limit is specified in packets per
second. A typical access port serving a single host needs at most 15 pps;
a port connected to a hypervisor with many VMs might need 100 pps.

When the rate is exceeded, the port is err-disabled by default. This is
aggressive — a single burst can take down a port. Consider combining rate
limiting with an automatic `errdisable recovery cause dhcpsnoop` timer to
bring ports back online after a cooling period.

---

## 4. Dynamic ARP Inspection (DAI)

### 4.1 The Attack: ARP Spoofing / ARP Poisoning

ARP operates on trust. When a host broadcasts "Who has 10.10.10.1?", any host
on the segment can respond "10.10.10.1 is at AA:BB:CC:DD:EE:FF" — even if that
MAC address belongs to the attacker, not the real owner of the IP. The
requesting host caches this poisoned mapping and sends subsequent traffic to the
attacker. This enables man-in-the-middle attacks, session hijacking, and
credential interception.

ARP spoofing is trivially easy to execute (tools like `arpspoof`, `ettercap`,
and `bettercap` automate it) and devastating in flat Layer 2 networks common
in data centers.

### 4.2 How DAI Works

DAI intercepts all ARP packets on untrusted ports and validates them against
the DHCP snooping binding table (or static ARP ACLs for hosts with static IPs).
The validation checks:

- **Source MAC** — The Ethernet source MAC must match the sender hardware address
  in the ARP payload.
- **Destination MAC** — For ARP replies, the Ethernet destination MAC must match
  the target hardware address in the ARP payload.
- **IP address** — The sender IP in the ARP payload must exist in the DHCP
  snooping binding table and be associated with the correct MAC and interface.

If any validation fails, the ARP packet is dropped, an optional log entry is
generated, and the violation counter increments. Unlike port security, DAI does
not err-disable the port by default — it silently drops the offending ARP
while allowing other traffic to continue.

### 4.3 ARP ACLs for Static Hosts

Not every host uses DHCP. Servers, network appliances, and infrastructure
devices often have static IP assignments. These hosts have no DHCP snooping
binding table entry. Without intervention, DAI would drop all their ARP traffic.

ARP ACLs solve this by providing explicit permit entries for static
IP-to-MAC mappings. The ACL is applied as a filter on specific VLANs. DAI
checks the ARP ACL first; if no match is found, it falls back to the DHCP
snooping binding table.

### 4.4 DAI Rate Limiting

ARP-based attacks often generate hundreds of ARP packets per second. DAI
rate limiting protects the switch's CPU by capping the number of ARP packets
processed per second on untrusted ports. The default rate on NX-OS is 15 pps
per port. Exceeding the rate causes the port to be err-disabled.

This default is appropriate for access ports serving single hosts or small
groups. Ports connected to virtualization hosts or trunks carrying multiple
VLANs may need higher limits. The key principle: set the rate to the expected
peak ARP rate plus a safety margin, never to the theoretical maximum.

---

## 5. IP Source Guard (IPSG)

### 5.1 The Attack: IP Spoofing at Layer 2

Even with DHCP snooping and DAI in place, an attacker can spoof IP addresses
in non-ARP traffic (TCP, UDP, ICMP). DHCP snooping prevents rogue DHCP servers;
DAI prevents ARP spoofing; but neither validates the source IP of data-plane
packets. An attacker could configure a static IP on their NIC and send traffic
with any source address.

### 5.2 How IPSG Works

IPSG programs hardware ACLs (TCAM entries) on each port that restrict source
IP addresses to those present in the DHCP snooping binding table. Any packet
arriving on an untrusted port with a source IP not in the binding table is
dropped in hardware — no CPU involvement, no performance impact.

IPSG operates in two modes:

- **IP filter** — Validates only the source IP address against the binding table.
- **IP + MAC filter** — Validates both the source IP and source MAC address
  against the binding table. This is stricter and prevents an attacker from
  using a legitimate IP with a spoofed MAC.

### 5.3 TCAM Considerations

IPSG consumes TCAM entries — one per binding per port. In environments with
thousands of hosts, this can exhaust the TCAM on access-layer switches. Nexus
9000 series switches with large TCAM carving profiles can handle this, but
it requires capacity planning. Run `show hardware access-list resource
utilization` to monitor TCAM consumption.

### 5.4 Static Bindings

For hosts with static IP addresses (no DHCP snooping entry), administrators
must create manual source bindings. These are functionally equivalent to
DHCP snooping binding table entries but are statically configured and persist
across reboots without needing database persistence.

---

## 6. Port Security

### 6.1 Purpose and Scope

Port security limits the number of MAC addresses learned on a switchport and
can restrict which specific MAC addresses are permitted. It is a blunt
instrument compared to 802.1X but provides a baseline defense against:

- **MAC flooding attacks** — An attacker sends frames with thousands of random
  source MACs to overflow the switch's MAC address table, causing the switch to
  flood all traffic to all ports (effectively turning it into a hub).
- **Unauthorized device connection** — A user plugs a personal switch or AP
  into an access port, creating an uncontrolled extension of the network.

### 6.2 Sticky MAC Addresses

Sticky learning dynamically learns the first N MAC addresses on a port and
writes them to the running configuration as static entries. This provides the
security of static MAC bindings without the operational overhead of manually
discovering and configuring every host's MAC address.

Important caveat: sticky MACs are written to running-config, not startup-config.
They persist across port flaps but are lost on device reload unless the
administrator explicitly saves the configuration. This is a common operational
surprise.

### 6.3 Violation Modes

The three violation modes represent different points on the security-vs-
availability tradeoff:

- **Shutdown** (default) — Maximum security. The port is err-disabled on the
  first violation. Requires manual intervention (or `errdisable recovery`) to
  restore. Appropriate for high-security environments where unauthorized access
  is a serious incident.
- **Restrict** — Balanced. Violating frames are dropped, SNMP traps are sent,
  syslog messages are generated, and violation counters increment. The port
  remains operational for legitimate traffic. Appropriate for environments where
  availability is critical but monitoring is active.
- **Protect** — Minimal disruption. Violating frames are silently dropped with
  no logging or notification. Appropriate only when the goal is preventing MAC
  flooding without operational overhead, and where monitoring violations is not
  a priority.

---

## 7. Storm Control

### 7.1 The Problem: Traffic Storms

A broadcast storm occurs when broadcast frames loop indefinitely through a
Layer 2 network (typically due to a spanning-tree failure or misconfiguration).
A single storm can consume 100% of link bandwidth across every port in the
VLAN, causing complete network outage.

Even without loops, excessive broadcast or multicast traffic from a
malfunctioning NIC, misconfigured application, or deliberate attack can
degrade network performance.

### 7.2 Mechanism

Storm control monitors incoming traffic on a per-port basis and compares the
broadcast, multicast, or unicast traffic rate against a configured threshold
(expressed as a percentage of link bandwidth or as packets per second). When
the rate exceeds the threshold, the switch takes one of several actions:

- **Drop excess traffic** (default) — Traffic above the threshold is dropped
  while traffic below continues to flow.
- **Trap** — Sends an SNMP trap to the management station.
- **Shutdown** — Err-disables the port.

### 7.3 Tuning Thresholds

The default thresholds are intentionally conservative. In practice:

- **Broadcast** — 10% is reasonable for access ports. Server ports with PXE
  boot or DHCP relay may need higher thresholds during specific operations.
- **Multicast** — Depends heavily on the environment. Networks with multicast
  routing (PIM, IGMP) may see legitimate multicast rates of 20-30%.
- **Unknown unicast** — 5% is typically sufficient. High unknown unicast rates
  usually indicate MAC table overflow or asymmetric routing.

Set storm-control action to `trap` in addition to `shutdown` so that the
management system is notified before or as the port goes down.

---

## 8. First-Hop Security for IPv6

### 8.1 Why IPv6 Needs Special Protection

IPv6 replaces ARP with Neighbor Discovery Protocol (NDP), which uses ICMPv6
messages: Router Solicitation (RS), Router Advertisement (RA), Neighbor
Solicitation (NS), and Neighbor Advertisement (NA). NDP is inherently
unauthenticated, making it vulnerable to the same classes of attacks as ARP
but with additional attack surfaces:

- **Rogue RA attacks** — An attacker sends Router Advertisements, causing
  hosts to adopt the attacker as their default gateway (equivalent to DHCP
  spoofing in IPv4).
- **NDP spoofing** — Equivalent to ARP spoofing; an attacker responds to
  Neighbor Solicitations with its own link-layer address.
- **DHCPv6 spoofing** — A rogue DHCPv6 server assigns addresses and DNS
  servers controlled by the attacker.

### 8.2 RA Guard

RA Guard filters Router Advertisement and Router Redirect messages based on
a policy applied to switch ports. Ports are classified as:

- **Host ports** — RA and Router Redirect messages are dropped. This is the
  correct policy for all access ports connected to end hosts.
- **Router ports** — RA messages are permitted. Applied only to uplinks
  connected to legitimate routers.

RA Guard operates in the data plane (hardware-based on supported platforms)
and does not rely on any binding table. It is a simple but effective control
that blocks the most common IPv6 first-hop attack.

### 8.3 DHCPv6 Guard

DHCPv6 Guard filters DHCPv6 server messages (Advertise, Reply, Reconfigure,
Relay-Reply) on ports classified as client-facing. Only ports classified
as server-facing permit these messages. This prevents rogue DHCPv6 servers
from assigning addresses, DNS servers, or other configuration parameters
to IPv6 hosts.

### 8.4 IPv6 Snooping and Binding Table

NX-OS can build an IPv6 neighbor binding table by snooping NDP and DHCPv6
traffic, analogous to the DHCP snooping binding table for IPv4. This table
feeds IPv6 Source Guard and IPv6 Destination Guard, providing the same
source IP validation for IPv6 that IPSG provides for IPv4.

---

## 9. NX-OS System Hardening

### 9.1 SSH Hardening

Telnet transmits credentials in cleartext and must be disabled (`no feature
telnet`). SSH is the only acceptable remote access protocol. Hardening SSH:

- **Key-based authentication** — Eliminates password-based brute force. Each
  administrator uploads their public key; the switch authenticates using
  asymmetric cryptography.
- **Key size** — Minimum 2048-bit RSA or Ed25519. The default 1024-bit RSA
  key generated by some NX-OS versions is insufficient.
- **Login attempts** — `ssh login-attempts 3` locks out after 3 failed
  attempts, mitigating online brute-force attacks.
- **Grace time** — `ssh login-gracetime 60` closes idle unauthenticated SSH
  sessions after 60 seconds, preventing connection exhaustion attacks.

### 9.2 Password Policies

NX-OS supports password complexity enforcement:

- **Strength check** — `password strength-check` rejects passwords that do
  not meet minimum complexity requirements (length, character diversity).
- **Expiration** — `password-expiry max-lifetime 90` forces password rotation
  every 90 days.
- **Warning** — `password-expiry warn-interval 14` notifies users 14 days
  before expiration.

These controls are particularly important for local accounts that serve as
fallback when AAA servers are unreachable.

### 9.3 Management ACLs

A management ACL restricts which source IPs can reach the switch's management
plane. Applied to VTY lines, it ensures that only authorized management
stations (jump hosts, bastion hosts, automation servers) can establish SSH
sessions.

The ACL should use an explicit deny-all at the end with logging enabled.
Every denied connection attempt generates a syslog message, providing
visibility into unauthorized access attempts.

### 9.4 Login Banners

Login banners serve a legal function, not a technical one. In many jurisdictions,
unauthorized computer access is prosecutable only if the system displayed a
clear warning that access was restricted. The banner should state:

1. Access is restricted to authorized users only.
2. All activity is monitored and logged.
3. Unauthorized access will be prosecuted.

### 9.5 Disabling Unnecessary Services

Every enabled feature increases the attack surface. NX-OS is modular — features
are explicitly enabled with `feature <name>`. Disable everything not in use:

- `no feature telnet` — Always.
- `no feature nxapi` — Unless NX-API is actively used for automation.
- `no ip domain-lookup` — Prevents DNS resolution delays when mistyping
  commands (also prevents DNS-based information leakage from the mgmt VRF).
- `no cdp enable` — On ports facing untrusted networks (CDP reveals device
  model, software version, and IP addresses).
- `no lldp transmit` / `no lldp receive` — Same rationale as CDP.

---

## 10. MACsec on Nexus

### 10.1 What MACsec Provides

MACsec (IEEE 802.1AE) provides hop-by-hop Layer 2 encryption, integrity
checking, and replay protection between directly connected devices. Unlike
IPsec (Layer 3) or TLS (Layer 4+), MACsec encrypts everything above Layer 2
including the IP header, making traffic analysis impossible.

### 10.2 MKA — MACsec Key Agreement

MKA (IEEE 802.1X-2010) is the key management protocol for MACsec. It handles:

- **Peer discovery** — Devices on a link identify each other as MACsec-capable.
- **Key derivation** — The Connectivity Association Key (CAK) is pre-shared;
  MKA derives the Secure Association Key (SAK) from it.
- **Key rotation** — SAK is rotated periodically (configured via
  `sak-expiry-time`) to limit the impact of key compromise.
- **Key server election** — On point-to-point links, the device with the
  lower key-server-priority becomes the key server and distributes the SAK.

### 10.3 Security Policies

- **must-secure** — All traffic must be MACsec-encrypted. Unencrypted traffic
  is dropped. Use in production after verifying MACsec works on all links.
- **should-secure** — Prefer MACsec but allow cleartext fallback. Use during
  migration or when MACsec capability is uncertain on peer devices.

### 10.4 Cipher Suites

NX-OS supports GCM-AES-128 and GCM-AES-256. GCM-AES-256 provides a higher
security margin and should be preferred unless hardware limitations mandate
128-bit. Both provide authenticated encryption with associated data (AEAD) —
the integrity tag covers both the encrypted payload and the unencrypted
Layer 2 header.

### 10.5 Deployment Considerations

MACsec is hop-by-hop: every device on the path must support it. In a
spine-leaf topology, MACsec must be enabled on both the leaf uplink and the
spine downlink. If any intermediate device (e.g., a patch panel with active
components, a media converter) does not support MACsec, the encrypted frames
are dropped.

MACsec adds 32 bytes of overhead per frame (16-byte SecTAG + 16-byte ICV),
which may cause issues with jumbo frame configurations on links already
operating near the MTU limit.

---

## 11. Keychain Management

### 11.1 Purpose

Keychains provide a structured mechanism for managing authentication keys
used by routing protocols (OSPF, BGP, EIGRP, IS-IS) and other features.
They support key rollover through lifetime windows: each key has a
`send-lifetime` and `accept-lifetime`. By overlapping lifetimes during
rotation, the switch can transition from one key to the next without
dropping adjacencies.

### 11.2 Cryptographic Algorithms

NX-OS keychains support multiple algorithms:

| Algorithm        | Strength     | Use Case                                    |
|-----------------|--------------|----------------------------------------------|
| MD5             | Weak         | Legacy compatibility only (deprecated)       |
| HMAC-SHA-1      | Moderate     | Acceptable for existing deployments          |
| HMAC-SHA-256    | Strong       | Recommended for new deployments              |
| AES-128-CMAC    | Strong       | Required for MACsec CAK                     |

MD5 is cryptographically broken and should not be used for new deployments.
HMAC-SHA-256 is the recommended minimum for routing protocol authentication.

### 11.3 Key Rollover Strategy

A zero-downtime key rollover follows this sequence:

1. Add the new key to the keychain on all peers with an `accept-lifetime`
   starting now and `send-lifetime` starting in the future (e.g., 1 hour).
2. Wait for the configuration to propagate to all peers.
3. When the `send-lifetime` activates, devices begin sending with the new key
   while still accepting both old and new keys.
4. After all devices are sending with the new key, update the old key's
   `accept-lifetime` to expire (e.g., set end time to now + 1 hour).
5. After the old key's `accept-lifetime` expires, remove it from the keychain.

---

## 12. System-Level Security

### 12.1 NX-API Security

NX-API exposes the NX-OS CLI and NX-OS data structures via HTTP/HTTPS REST
endpoints. By default, it may listen on both HTTP (port 80) and HTTPS
(port 443). Securing NX-API:

- **Disable HTTP** — `no nxapi http` ensures all API traffic is encrypted.
- **Restrict to management VRF** — `nxapi use-vrf management` prevents data-
  plane access to the API.
- **Disable sandbox** — `no nxapi sandbox` removes the web-based CLI that
  could be used for interactive exploitation if credentials are compromised.
- **Certificate management** — Replace the self-signed default certificate
  with a CA-signed certificate to enable proper TLS validation by API clients.

### 12.2 NX-OS Image Integrity Verification

NX-OS supports cryptographic verification of the running image and filesystem
integrity. The `show system integrity all` command displays:

- **Image signing verification** — Confirms the running image was signed by
  Cisco and has not been modified.
- **Filesystem integrity** — Checks critical system files against known-good
  hashes.

This should be run after every image upgrade, ISSU, or whenever the device
exhibits unexpected behavior. A failed integrity check indicates either
corruption or compromise and warrants immediate investigation.

### 12.3 Control Plane Policing (CoPP)

The supervisor module's CPU processes control-plane traffic: routing protocol
hello packets, ARP requests, ICMP, SSH sessions, SNMP polls, and more. Without
protection, an attacker (or a network event like an ARP storm) can overwhelm
the CPU, causing routing adjacency flaps, management session drops, and
potentially a switch reload.

CoPP applies QoS policies to traffic destined for the supervisor CPU. NX-OS
ships with predefined CoPP profiles:

- **Lenient** — Minimal rate limiting. Suitable for lab environments.
- **Moderate** — Balanced. Default on most platforms.
- **Strict** — Aggressive rate limiting. Recommended for production. Limits
  ARP to the CPU to a few hundred pps, ICMP to a similar rate, and prioritizes
  routing protocol traffic.
- **Dense** — Optimized for high-density configurations with many SVI
  interfaces.

Apply `copp profile strict` in production. Monitor `show policy-map interface
control-plane` to verify that legitimate traffic is not being dropped. If
routing protocols experience intermittent flaps, check CoPP drop counters
before investigating other causes.

### 12.4 Console and VTY Hardening

- **Exec timeout** — `exec-timeout 5` on console, `exec-timeout 10` on VTY.
  Prevents abandoned sessions from remaining authenticated indefinitely.
- **Session limit** — `session-limit 5` on VTY limits concurrent SSH sessions,
  mitigating connection exhaustion attacks.
- **Transport restriction** — `transport input ssh` ensures only SSH is
  accepted on VTY lines. Without this, telnet may be accepted even if the
  `feature telnet` is disabled (platform-dependent behavior).

---

## Prerequisites

Before implementing the security features described in this document:

- NX-OS version 10.2(1) or later is recommended for full feature support
  (MACsec, IPv6 FHS, NX-API HTTPS-only).
- DHCP snooping must be enabled and operational before enabling DAI or IPSG,
  as both depend on the snooping binding table.
- TACACS+ or RADIUS servers must be deployed, tested, and reachable from the
  management VRF before configuring remote AAA.
- TCAM carving may be required for IPSG on Nexus 9000 platforms; verify
  available TCAM resources with `show hardware access-list resource utilization`.
- MACsec requires compatible optics and line cards; not all Nexus SKUs support
  MACsec in hardware.
- Local fallback accounts must exist before enabling remote AAA to prevent
  lockout during server outages.
- Backup the running configuration before enabling features that can err-disable
  ports (DAI, port security, storm control) to allow rapid rollback.

---

## References

- IEEE 802.1X-2010 — Port-Based Network Access Control (MKA)
- IEEE 802.1AE-2018 — Media Access Control Security (MACsec)
- RFC 2865 — Remote Authentication Dial In User Service (RADIUS)
- RFC 8907 — The TACACS+ Protocol
- RFC 4861 — Neighbor Discovery for IP version 6 (IPv6)
- RFC 6105 — IPv6 Router Advertisement Guard
- RFC 7610 — DHCPv6-Shield: Protecting Against Rogue DHCPv6 Servers
- NIST SP 800-53 Rev. 5 — Security and Privacy Controls for Information Systems
- NIST SP 800-123 — Guide to General Server Security
- Cisco NX-OS Security Configuration Guide, Release 10.x
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x.html
- Cisco Nexus 9000 MACsec Configuration Guide
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-macsec.html
- Cisco NX-OS CoPP Best Practices
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-copp.html
