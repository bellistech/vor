# Architecture Models & Enterprise Infrastructure Security (Security+ SY0-701 Objectives 3.1 + 3.2)

> Cloud vs on-prem vs IaC vs ICS/SCADA trade-offs, then the enterprise plumbing: zones, appliances, fail-open vs fail-closed, 802.1X, the four firewall types, and TLS-vs-IPSec tunneling — Domain 3's compare-and-contrast core.

## Architecture & Infrastructure Concepts (3.1)

```
# Cloud
#   Responsibility matrix — who secures what. Provider always owns the
#     physical layer; you always own your data and identities. The line
#     moves with the model:
#     IaaS (Infrastructure as a Service): you patch guest OS + up
#     PaaS (Platform as a Service): you own code + data
#     SaaS (Software as a Service): you own data, config, users
#   Hybrid considerations — cloud + on-prem: identity federation, data
#     in transit between sites, inconsistent controls, latency.
#   Third-party vendors — SaaS integrations expand attack surface;
#     vendor assessment applies (secplus-third-party-compliance).

# IaC (Infrastructure as Code)
#   Infrastructure defined in versioned templates (Terraform, CloudFormation).
#   Security wins: repeatable baselines, code review + static scanning of
#   infra, drift detection. Risks: secrets in templates, one bad template
#   deployed everywhere at once.

# Serverless
#   Functions run on provider-managed runtimes (Lambda-style). No OS to
#   patch (provider's job); your risks concentrate in code, dependencies,
#   permissions (over-privileged function roles), and event injection.

# Microservices
#   Many small services over APIs vs one monolith. Wins: isolation, small
#   blast radius, independent patching. Risks: huge east-west traffic to
#   authenticate (mTLS/service mesh), sprawling API attack surface.

# Network infrastructure
#   Physical isolation / air-gapped — no network connection at all.
#     Strongest separation; updates arrive by removable media (which
#     becomes the vector — Stuxnet lesson).
#   Logical segmentation — VLANs (Virtual Local Area Networks), subnets,
#     firewall zones separating traffic on shared hardware.
#   SDN (Software-Defined Networking) — control plane separated from
#     data plane, network programmable via controller APIs. Enables
#     micro-segmentation; the controller itself is a crown-jewel target.

# On-premises — you own everything: full control, full responsibility,
#   capex-heavy, your ceiling on scalability and power/compute.
# Centralized vs decentralized — one authority/point of control (simpler
#   policy, single point of failure) vs distributed (resilient, harder
#   to keep consistent).

# Containerization — shared host kernel, isolated userspaces. Light and
#   fast; kernel escape affects every container; image supply chain and
#   registry hygiene matter (sbom, supply-chain-security).
# Virtualization — full guest OSes on a hypervisor. Stronger isolation
#   than containers; VM escape and VM sprawl are the risks.

# IoT (Internet of Things) — cameras, sensors, wearables. Weak/default
#   creds, unpatchable firmware, long life. Segment them away.
# ICS (Industrial Control Systems) / SCADA (Supervisory Control and
#   Data Acquisition) — factories, power, water. Availability trumps
#   everything; devices are fragile under scanning; patch windows rare;
#   protocols (Modbus) lack auth. Isolate + monitor passively.
# RTOS (Real-Time Operating System) — deterministic timing (medical,
#   automotive, avionics). A security scan that adds latency can be a
#   safety incident; harden at build time, isolate at runtime.
# Embedded systems — fixed-function device firmware; constrained CPU/RAM
#   often can't run agents; long support gaps; vendor-dependent patching.
# High availability — design goal of surviving component failure
#   (clustering, load balancing, redundancy) → secplus-data-resilience.
```

## Consideration Trade-off Table (3.1)

| Consideration | The question it asks | Model that scores best / worst |
|---|---|---|
| Availability | Does it stay up through failures? | Cloud multi-region best; single on-prem server worst |
| Resilience | How fast does it recover from damage? | IaC-rebuilt cloud best; hand-built legacy worst |
| Cost | Capex vs opex, and at what scale? | Cloud = opex, low entry; on-prem cheaper at steady large scale |
| Responsiveness | Latency to users/processes | Edge/on-prem best for local, RTOS for deterministic |
| Scalability | Can it grow (and shrink) on demand? | Serverless/cloud auto-scale best; physical worst |
| Ease of deployment | Time from decision to running | Serverless/containers best; ICS worst |
| Risk transference | Can responsibility shift contractually? | Cloud/MSP shifts some risk; air-gap keeps it all in-house |
| Ease of recovery | Restore after incident | IaC + snapshots best; embedded/ICS worst |
| Patch availability | Are fixes even published? | Mainstream OS best; EOL/embedded/IoT worst |
| Inability to patch | What if you never can? | ICS/RTOS/medical — plan compensating controls instead |
| Power | Who provides/backs it? | Cloud provider's problem vs your generators+UPS |
| Compute | Capacity ceiling | Cloud elastic; embedded fixed forever |

## Infrastructure Considerations (3.2)

```
# Device placement — what sits where: screened subnet for public-facing
#   services, sensors at chokepoints, jump server bridging admin zone.
# Security zones — trust boundaries: internet (untrusted) / screened
#   subnet / internal / restricted-management. Traffic between zones
#   crosses a policy enforcement point.
# Attack surface — every exposed port/service/interface; placement
#   decisions grow or shrink it.
# Connectivity — redundant links, out-of-band management network.

# Failure modes (memorize):
#   Fail-open   — device fails → traffic FLOWS uninspected.
#                 Availability preserved, security lost.
#   Fail-closed — device fails → traffic BLOCKED.
#                 Security preserved, availability lost.
#   Exam logic: safety/availability-critical (fire doors, hospital nets)
#   → fail-open; security-critical (firewall on classified net)
#   → fail-closed. The question tells you which value wins.

# Device attributes:
#   Active vs passive — active devices sit in the traffic path and ACT
#     (block/modify); passive devices observe and alert only.
#   Inline vs tap/monitor — inline = physically in the path (can block,
#     adds latency, its failure matters → fail-open/closed choice);
#     tap/monitor = copy of traffic via TAP (Test Access Point) or SPAN
#     (Switched Port Analyzer) port — zero traffic impact, zero blocking.
#   IPS must be inline+active. IDS can be passive on a tap.

# Network appliances:
#   Jump server — hardened intermediary; admins hop through it into the
#     secure zone. All privileged access funnels (and logs) through one
#     audited point. Exam cue: "administer servers in a screened subnet".
#   Proxy server — intermediary for client traffic. Forward proxy =
#     outbound control/filtering for users; reverse proxy = inbound
#     shield in front of servers (TLS termination, WAF pairing).
#   IPS (Intrusion Prevention System) — inline, blocks in real time.
#   IDS (Intrusion Detection System) — detects + alerts, does not block.
#   Load balancer — spreads traffic across servers; health checks;
#     also absorbs some DoS and hides backend topology.
#   Sensors — collection points feeding SIEM/NetFlow/IDS.

# Port security:
#   802.1X — port-based network access control: supplicant (client) /
#     authenticator (switch or AP) / authentication server (RADIUS —
#     Remote Authentication Dial-In User Service). No auth, no LAN.
#   EAP (Extensible Authentication Protocol) — the auth framework
#     carried by 802.1X: EAP-TLS (mutual certs, strongest), PEAP
#     (Protected EAP — TLS tunnel protecting inner creds).
```

## Firewall Types (3.2)

| Type | What it does | Layer | Exam cue |
|---|---|---|---|
| Layer 4 firewall | Filters on IP/port/protocol (stateful) | Transport | "Blocks by port and address" |
| Layer 7 firewall | Understands application payloads | Application | "Inspects HTTP methods/content" |
| NGFW (Next-Generation Firewall) | L7 + application awareness + IPS + user identity + TLS inspection | 3–7 | "Application-aware, deep packet inspection" |
| WAF (Web Application Firewall) | Protects web apps specifically — SQLi/XSS filtering | 7 (HTTP) | "In front of the web server, blocks injection" |
| UTM (Unified Threat Management) | All-in-one box: firewall + IPS + AV + content filter + VPN | Multi | "Single appliance for a branch office, many functions" |

```
# WAF vs NGFW trap: attack named is SQLi/XSS against YOUR web app → WAF.
# General perimeter control with app awareness → NGFW. Small office
# wanting one box that does everything → UTM (trade-off: single point
# of failure, jack of all trades).
```

## Secure Communication / Access (3.2)

```
# VPN (Virtual Private Network) — encrypted tunnel over untrusted nets.
#   Site-to-site: gateway↔gateway, always-on (IPSec typical).
#   Remote access: user→gateway (TLS or IPSec client).
#   Full tunnel = all traffic via HQ (inspected, slower).
#   Split tunnel = only corporate traffic tunneled (fast, less visibility).

# Tunneling protocols:
#   TLS (Transport Layer Security) VPN — runs over TCP/443, passes
#     through almost any firewall/NAT; per-app or clientless portal.
#   IPSec (Internet Protocol Security):
#     AH  (Authentication Header)          — integrity/auth, NO encryption
#     ESP (Encapsulating Security Payload) — encryption + integrity (the
#                                            one actually used)
#     Transport mode — payload encrypted, original IP header kept
#       (host-to-host)
#     Tunnel mode    — entire packet encrypted inside new header
#       (site-to-site gateways)
#     IKE (Internet Key Exchange) negotiates the SAs (security
#       associations).
#   Exam: "authentication only, no confidentiality" → AH;
#         "site-to-site" → ESP tunnel mode; "works from hotel Wi-Fi
#         that blocks everything but 443" → TLS VPN.

# SD-WAN (Software-Defined Wide Area Network) — policy-driven routing
#   across MPLS/broadband/LTE per application; encrypted overlays;
#   branch traffic can go direct-to-cloud instead of hairpinning HQ.
# SASE (Secure Access Service Edge) — SD-WAN + cloud-delivered security
#   stack (SWG — Secure Web Gateway, CASB — Cloud Access Security
#   Broker, ZTNA — Zero Trust Network Access, FWaaS) as one service at
#   the edge, near the user. Exam cue: "converged networking + security
#   as a cloud service for remote workforce" → SASE.

# Selection of effective controls — match control to data flow, threat,
#   and the availability-vs-security trade the scenario states; layered
#   (defense in depth), fail-safe defaults, least privilege.
```

## Exam Cues (3.1 + 3.2)

```
# "no physical connection to any network"        → air-gapped
# "who patches the hypervisor in IaaS?"          → cloud provider
# "who secures the data in SaaS?"                → the customer, always
# "factory controllers, cannot be patched"       → ICS/SCADA → isolate +
#                                                  compensating controls
# "deterministic timing required"                → RTOS
# "define infrastructure in versioned templates" → IaC
# "controller programs the network"              → SDN
# "firewall died, traffic must keep flowing"     → fail-open
# "block everything if the device fails"         → fail-closed
# "copy of traffic, cannot block"                → passive / tap-monitor
# "admins reach secure zone via one host"        → jump server
# "switch asks RADIUS before enabling port"      → 802.1X
# "mutual certificate authentication on Wi-Fi"   → EAP-TLS
# "one branch appliance doing FW+AV+IPS+VPN"     → UTM
# "stop SQLi hitting the web app"                → WAF
# "integrity without encryption"                 → IPSec AH
# "encrypt whole packet between gateways"        → IPSec ESP tunnel mode
# "cloud-delivered security at the edge"         → SASE
```

## See Also

security-plus-sy0-701, secplus-security-concepts, secplus-hardening, secplus-data-resilience, secplus-monitoring-defense, secplus-iam, security-architecture, cloud-security, firewall-design, ids-ips, waf, network-access-control, dot1x, remote-access-vpn, site-to-site-vpn, tls, zero-trust, network-defense

## References

- [CompTIA Security+ SY0-701 Exam Objectives](https://www.comptia.org/certifications/security)
- [NIST SP 800-207 — Zero Trust Architecture](https://csrc.nist.gov/pubs/sp/800/207/final)
- [NIST SP 800-82 Rev. 3 — Guide to Operational Technology (OT) Security](https://csrc.nist.gov/pubs/sp/800/82/r3/final)
- [NIST SP 800-125 — Guide to Security for Full Virtualization Technologies](https://csrc.nist.gov/pubs/sp/800/125/final)
- [RFC 4301 — Security Architecture for the Internet Protocol (IPSec)](https://www.rfc-editor.org/rfc/rfc4301)
- [RFC 8446 — TLS 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [IEEE 802.1X — Port-Based Network Access Control](https://standards.ieee.org/ieee/802.1X/7345/)
- [Cloud Security Alliance — Shared Responsibility Model](https://cloudsecurityalliance.org/)
