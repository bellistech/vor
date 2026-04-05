# BATTLE PLAN — Certification Gap Analysis & Sheet Creation

> **Project**: `/Users/govan/tmp/projects/cheat_sheet/`
> **Current Inventory**: 549 sheets / 549 details / 59 categories
> **Date**: 2026-04-05
> **Goal**: Audit 11 elite certifications against existing content, identify every gap, fill them all.

## Quick Reference

```bash
# Build & verify
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
go build -o ./cs ./cmd/cs/
go test ./... -count=1 -race
./cs -l | tail -n +3 | wc -l   # count sheets

# Sheet format: sheets/<category>/<topic>.md
# Detail format: detail/<category>/<topic>.md
# No code changes needed — go:embed auto-discovers
```

## Sheet Format Reference

```markdown
# Title (Subtitle)

One-liner description.

## Section Name
### Subsection
[content with code blocks, tables, diagrams]

## Tips
- Practical advice bullets

## See Also
- related-sheet-1, related-sheet-2

## References
- [Official Doc](url)
```

## Certifications Covered

| # | Cert | Vendor | Level | Estimated New Sheets |
|---|------|--------|-------|---------------------|
| 1 | CCNP Data Center | Cisco | Professional | ~18 |
| 2 | CCNP Enterprise | Cisco | Professional | ~15 |
| 3 | CCIE Enterprise Infrastructure | Cisco | Expert | ~12 |
| 4 | CCIE Service Provider | Cisco | Expert | ~14 |
| 5 | CCIE Security | Cisco | Expert | ~16 |
| 6 | CCIE Automation | Cisco | Expert | ~10 |
| 7 | JNCIE-SP | Juniper | Expert | ~12 |
| 8 | JNCIE-SEC | Juniper | Expert | ~10 |
| 9 | CompTIA Linux+ | CompTIA | Professional | ~8 |
| 10 | CISSP | ISC2 | Expert | ~14 |
| 11 | C\|RAGE | EC-Council | Professional | ~8 |
| **Total** | | | | **~137 new sheets** |

---

# GAP ANALYSIS BY CERTIFICATION

## 1. CCNP Data Center (350-601 DCCOR + 300-620/625/630/635)

### Existing Coverage
| Topic | Sheet | Status |
|-------|-------|--------|
| VLANs | `vlan` | HAVE |
| STP | `stp` | HAVE |
| LACP | `lacp` | HAVE |
| BGP | `bgp` | HAVE |
| OSPF | `ospf` | HAVE |
| VXLAN | `vxlan` | HAVE |
| IS-IS | `is-is` | HAVE |
| MPLS | `mpls` | HAVE |
| Docker/Containers | `docker`, `containerd` | HAVE |
| Kubernetes | `kubernetes` | HAVE |
| DHCP | `dhcp` | HAVE |
| SNMP | `snmp` | HAVE |
| QoS | `cos-qos` | HAVE |

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| DC-1 | `cisco-nexus` | `network-os` | NX-OS architecture, VDC, vPC, FabricPath, OTV |
| DC-2 | `fibre-channel` | `networking` | FC layers, zoning, VSAN, FLOGI/PLOGI, FCID |
| DC-3 | `fcoe` | `networking` | FCoE architecture, DCB, VN-Port, FIP, CNA |
| DC-4 | `private-vlans` | `networking` | Primary/isolated/community, promiscuous ports |
| DC-5 | `fhrp` | `networking` | HSRP, VRRP, GLBP — all three protocols |
| DC-6 | `cisco-aci` | `networking` | ACI fabric, APIC, EPG, contracts, bridge domains |
| DC-7 | `cisco-ucs` | `infrastructure` | UCS architecture, service profiles, pools, policies |
| DC-8 | `san-storage` | `storage` | SAN concepts, iSCSI, NVMe-oF, SCSI |
| DC-9 | `span-erspan` | `networking` | SPAN, RSPAN, ERSPAN, port mirroring |
| DC-10 | `vpc` | `networking` | Cisco vPC (NOT cloud VPC), peer-link, peer-keepalive |
| DC-11 | `fabric-multicast` | `networking` | PIM, IGMP snooping in DC fabric, multicast routing |
| DC-12 | `dc-automation` | `config-mgmt` | PowerOn Auto Provisioning, DCNM, NX-API |
| DC-13 | `roce` | `networking` | RoCE v1/v2, RDMA, iWARP, PFC, ECN |
| DC-14 | `eigrp` | `networking` | DUAL algorithm, feasible successor, stub routing |
| DC-15 | `copp` | `security` | Control Plane Policing, CoPP classes, rate-limiting |
| DC-16 | `nxos-security` | `security` | AAA, RBAC, first-hop security, DHCP snooping, DAI |
| DC-17 | `data-center-design` | `networking` | Spine-leaf, Clos, 3-tier, east-west vs north-south |
| DC-18 | `network-programmability` | `networking` | YANG models, NETCONF, RESTCONF, gNMI basics |

## 2. CCNP Enterprise (350-401 ENCOR + 300-4xx)

### Existing Coverage
| Topic | Sheet | Status |
|-------|-------|--------|
| BGP | `bgp` | HAVE |
| OSPF | `ospf` | HAVE |
| STP | `stp` | HAVE |
| VLAN | `vlan` | HAVE |
| LACP | `lacp` | HAVE |
| IPsec | `ipsec` | HAVE |
| GRE | part of networking | PARTIAL |
| SNMP | `snmp` | HAVE |
| WiFi | `wireless-hacking` | PARTIAL (offensive only) |
| QoS | `cos-qos` | HAVE |
| NAT | `nat` | HAVE |
| DHCP | `dhcp` | HAVE |
| ACL | `acl` | HAVE (file-level, need network ACL) |

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| EN-1 | `dmvpn` | `networking` | DMVPN phases, mGRE, NHRP, spoke-to-spoke |
| EN-2 | `sd-wan` | (exists) | VERIFY — may need expansion |
| EN-3 | `sd-access` | `networking` | SD-Access, LISP, VXLAN fabric, DNA Center |
| EN-4 | `ip-sla` | `networking` | IP SLA probes, tracking, PBR integration |
| EN-5 | `netflow-ipfix` | `monitoring` | NetFlow v5/v9, IPFIX, flexible NetFlow |
| EN-6 | `cisco-wireless` | `networking` | WLC, CAPWAP, 802.11ax, RF, RRM, FlexConnect |
| EN-7 | `dot1x` | `security` | 802.1X/EAP, MAB, RADIUS, NAC fundamentals |
| EN-8 | `macsec` | `security` | MACsec (802.1AE), MKA, hop-by-hop encryption |
| EN-9 | `network-acl` | `networking` | Standard/extended/named ACLs, wildcard masks |
| EN-10 | `gre-tunnels` | `networking` | GRE, mGRE, tunnel keepalives, recursive routing |
| EN-11 | `pbr` | `networking` | Policy-Based Routing, route-maps, match/set |
| EN-12 | `cisco-dna-center` | `networking` | DNA Center / Catalyst Center, assurance, automation |
| EN-13 | `eem` | `config-mgmt` | Embedded Event Manager, applets, Tcl policies |
| EN-14 | `cisco-ios-xr` | `network-os` | IOS XR architecture, commit model, admin plane |
| EN-15 | `flexvpn` | `networking` | FlexVPN, IKEv2, smart defaults, SVTI |

## 3. CCIE Enterprise Infrastructure

### Existing Coverage
Most L2/L3 protocols covered. CCIE goes deeper on:

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| CI-1 | `multicast-routing` | `networking` | PIM-SM, PIM-SSM, BSR, Auto-RP, MSDP, RPF |
| CI-2 | `mpls-te` | `networking` | MPLS TE, RSVP-TE, FRR, explicit paths, CBTS |
| CI-3 | `mpls-vpn` | `networking` | L3VPN (VPNv4/v6), L2VPN (VPLS, VPWS), VRF-Lite |
| CI-4 | `lisp` | `networking` | LISP protocol, EID/RLOC, map-server, PxTR |
| CI-5 | `isis-advanced` | `networking` | IS-IS multi-topology, wide metrics, TLVs, BFD |
| CI-6 | `bgp-advanced` | `networking` | BGP confederations, ORF, add-path, PIC, convergence |
| CI-7 | `ospf-advanced` | `networking` | OSPF area types deep dive, LSA types, SPF tuning |
| CI-8 | `network-services` | `networking` | WCCP, NTP advanced, DHCP relay, DNS proxy |
| CI-9 | `ipv6-advanced` | `networking` | IPv6 transition (6PE, 6VPE, NAT64, DS-Lite, MAP) |
| CI-10 | `vrf` | `networking` | VRF-Lite, VRF leaking, multi-VRF CE, import/export |
| CI-11 | `network-security-infra` | `security` | Zone-based firewall, uRPF, CoPP, iACL, ZBFW |
| CI-12 | `qos-advanced` | `networking` | MQC, CBWFQ, LLQ, WRED, shaping vs policing deep dive |

## 4. CCIE Service Provider

### Existing Coverage
BGP, OSPF, IS-IS, MPLS, VXLAN, ECMP, BFD, RPKI, segment-routing exist.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| SP-1 | `carrier-ethernet` | `networking` | MEF services (E-Line/E-LAN/E-Tree), OAM, CFM |
| SP-2 | `unified-mpls` | `networking` | Unified MPLS/Seamless MPLS, inter-AS, ABR |
| SP-3 | `l2vpn-services` | `networking` | VPLS, VPWS, H-VPLS, MAC learning, PWE3 |
| SP-4 | `evpn-advanced` | `networking` | EVPN types 1-5, all-active, single-active, ESI |
| SP-5 | `mvpn` | `networking` | mVPN profiles, MDT, P-tunnel, data MDT |
| SP-6 | `srv6` | `networking` | SRv6, SRv6 TE, uSID, network programming |
| SP-7 | `bng` | `networking` | BNG/BRAS, PPPoE, IPoE, subscriber management |
| SP-8 | `cgnat` | `networking` | Carrier-Grade NAT (NAT444), DS-Lite, MAP-T/MAP-E |
| SP-9 | `sp-multicast` | `networking` | Multicast in SP (mLDP, P2MP TE, mVPN) |
| SP-10 | `peering-transit` | `networking` | IX peering, transit, PNI, route servers, IRR |
| SP-11 | `sp-qos` | `networking` | SP QoS models, DiffServ, H-QoS, traffic engineering |
| SP-12 | `te-rsvp` | `networking` | RSVP-TE, bandwidth reservation, FRR facility/1:1 |
| SP-13 | `g8032-erp` | `networking` | G.8032 Ethernet Ring Protection, RPL, ERPS |
| SP-14 | `q-in-q` | `networking` | 802.1ad Q-in-Q, selective QinQ, S-VLAN/C-VLAN |

## 5. CCIE Security

### Existing Coverage
TLS, IPsec, PKI, firewall (iptables/nftables), IDS/IPS, WAF, SELinux, AppArmor, fail2ban, seccomp, capabilities, Vault, zero-trust, SIEM, threat-modeling, etc.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| SE-1 | `cisco-ftd` | `security` | Firepower Threat Defense, FMC, Snort rules, AMP |
| SE-2 | `cisco-ise` | `security` | ISE architecture, profiling, posture, BYOD, pxGrid |
| SE-3 | `trustsec` | `security` | Cisco TrustSec, SGT, SGACL, SXP, inline tagging |
| SE-4 | `site-to-site-vpn` | `security` | IKEv1/IKEv2, crypto maps, VTI, GETVPN, FlexVPN |
| SE-5 | `remote-access-vpn` | `security` | AnyConnect, SSL VPN, clientless, split-tunnel |
| SE-6 | `email-gateway` | `security` | Cisco ESA/Cloud Email Security, anti-spam, DLP |
| SE-7 | `web-security-proxy` | `security` | Cisco WSA/SWG, URL filtering, HTTPS inspection |
| SE-8 | `network-access-control` | `security` | NAC framework, posture assessment, remediation |
| SE-9 | `cisco-umbrella` | `security` | DNS security, SIG, CASB, cloud-delivered firewall |
| SE-10 | `firewall-design` | `security` | Firewall architectures, DMZ, screened subnet, micro-seg |
| SE-11 | `crypto-protocols` | `security` | ESP/AH, IKE, ISAKMP, Diffie-Hellman groups, PFS |
| SE-12 | `endpoint-security` | `security` | AMP, EDR/XDR, host-based firewall, HIPS |
| SE-13 | `cloud-security` | `security` | CASB, CSPM, CWPP, cloud firewall, shared responsibility |
| SE-14 | `security-operations` | `security` | SOC tiers, incident response lifecycle, playbooks |
| SE-15 | `content-security` | `security` | DLP, content filtering, sandboxing, file reputation |
| SE-16 | `identity-management` | `security` | Identity governance, PAM, MFA, SSO architecture |

## 6. CCIE Automation (DevNet Expert)

### Existing Coverage
Ansible, Terraform, Python, Go, Git, REST API, gRPC, YAML, JSON, Protobuf, Docker, K8s exist.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| AU-1 | `cisco-nso` | `config-mgmt` | NSO architecture, NEDs, YANG, service models |
| AU-2 | `pyats` | `testing` | pyATS/Genie, testbed, parsers, test scripts |
| AU-3 | `gnmi-gnoi` | `monitoring` | gNMI streaming telemetry, gNOI operations |
| AU-4 | `netconf` | `networking` | NETCONF protocol, operations, capabilities, subtree |
| AU-5 | `restconf` | `networking` | RESTCONF protocol, YANG-to-REST mapping |
| AU-6 | `yang-models` | `networking` | YANG language, containers, lists, augment, deviation |
| AU-7 | `nornir` | `config-mgmt` | Python automation framework, inventory, tasks |
| AU-8 | `network-ci-cd` | `ci-cd` | Network CI/CD pipelines, batfish, intent validation |
| AU-9 | `model-driven-telemetry` | `monitoring` | MDT, dial-in/dial-out, GPB/JSON encoding |
| AU-10 | `napalm` | `config-mgmt` | NAPALM library, getters, config merge/replace |

## 7. JNCIE-SP (Juniper Service Provider Expert)

### Existing Coverage
BGP, OSPF, IS-IS, MPLS, LDP, RSVP, segment-routing, BFD, RPKI + 8 JNCIA sheets.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| JS-1 | `junos-mpls-advanced` | `juniper` | MPLS TE, CSPF, LSP protection, inter-area TE |
| JS-2 | `junos-l3vpn` | `juniper` | L3VPN, VRF, PE-CE routing, inter-AS options |
| JS-3 | `junos-l2vpn` | `juniper` | VPLS, VPWS, CCC, TCC, learning domains |
| JS-4 | `junos-evpn-vxlan` | `juniper` | EVPN-VXLAN on Junos, ERB, CRB, ESI-LAG |
| JS-5 | `junos-multicast` | `juniper` | PIM, IGMP, MSDP, anycast RP on Junos |
| JS-6 | `junos-class-of-service` | `juniper` | CoS, schedulers, BA classifiers, rewrite rules |
| JS-7 | `junos-high-availability` | `juniper` | GRES, NSR, ISSU, BFD, VRRP on Junos |
| JS-8 | `junos-segment-routing` | `juniper` | SR-MPLS, TI-LFA, Flex-Algorithm on Junos |
| JS-9 | `junos-bgp-advanced` | `juniper` | BGP add-path, ORF, graceful restart, dampening |
| JS-10 | `junos-isis-advanced` | `juniper` | IS-IS multi-topology, overload, authentication |
| JS-11 | `junos-bng` | `juniper` | Subscriber management, PPPoE, DHCP local-server |
| JS-12 | `junos-nat` | `juniper` | Source/destination NAT, NAT pools, persistent NAT |

## 8. JNCIE-SEC (Juniper Security Expert)

### Existing Coverage
8 JNCIA sheets + general firewall/IDS/VPN sheets.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| JC-1 | `junos-srx` | `juniper` | SRX architecture, zones, policies, flow-based |
| JC-2 | `junos-ipsec-vpn` | `juniper` | IPsec on SRX, IKE, proposals, traffic selectors |
| JC-3 | `junos-utm` | `juniper` | UTM, antivirus, web filtering, anti-spam on SRX |
| JC-4 | `junos-ids-ips` | `juniper` | IDP on SRX, custom signatures, sensor modes |
| JC-5 | `junos-nat-security` | `juniper` | NAT in security context, policy NAT, interface NAT |
| JC-6 | `junos-ha-security` | `juniper` | Chassis cluster, RG, RTO, failover on SRX |
| JC-7 | `junos-advanced-security` | `juniper` | AppSecure, AppTrack, AppFW, AppQoS, SSL proxy |
| JC-8 | `junos-security-policies` | `juniper` | Security policy hierarchy, global, zone-pair |
| JC-9 | `junos-screens` | `juniper` | Screen options, DoS protection, flood thresholds |
| JC-10 | `junos-sky-atp` | `juniper` | Sky ATP / Juniper ATP Cloud, threat intelligence |

## 9. CompTIA Linux+ (XK0-005)

### Existing Coverage
Extensive Linux coverage already: systemd, cgroups, namespaces, iptables/nftables, SELinux, AppArmor, PAM, LVM, RAID, filesystem, services, etc.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| LX-1 | `sssd` | `auth` | SSSD, realm, AD/LDAP integration, offline auth |
| LX-2 | `polkit` | `security` | PolicyKit, pkexec, polkit rules, D-Bus auth |
| LX-3 | `selinux-advanced` | `security` | SELinux deep-dive: policy modules, booleans, contexts |
| LX-4 | `linux-boot-process` | `system` | BIOS/UEFI, GRUB2, initramfs, kernel params, targets |
| LX-5 | `linux-storage-management` | `storage` | Device mapper, multipath, iSCSI initiator, LIO |
| LX-6 | `linux-networking-config` | `networking` | NetworkManager, nmcli, ip-route2, bonding, teaming |
| LX-7 | `linux-troubleshooting` | `system` | Systematic troubleshooting, log analysis, recovery |
| LX-8 | `linux-automation-scripting` | `shell` | Cron/at, systemd timers, expect, heredocs, functions |

## 10. CISSP (ISC2)

### Existing Coverage
PKI, TLS, cryptography, IAM, OAuth, OIDC, SAML, compliance frameworks (NIST, FEDRAMP, SOC2, PCI-DSS, HIPAA, GDPR, ISO27001), zero-trust, threat-modeling, incident-response.

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| CP-1 | `security-governance` | `security` | Security governance, policies, standards, guidelines |
| CP-2 | `risk-management` | `security` | Risk assessment, quantitative/qualitative, frameworks |
| CP-3 | `bcp-drp` | `security` | BCP/DRP, BIA, RPO/RTO, DR strategies, testing |
| CP-4 | `security-models` | `security` | Bell-LaPadula, Biba, Clark-Wilson, Brewer-Nash |
| CP-5 | `asset-security` | `security` | Data classification, handling, retention, destruction |
| CP-6 | `security-architecture` | `security` | Security architecture frameworks, SABSA, TOGAF-sec |
| CP-7 | `access-control-models` | `security` | DAC, MAC, RBAC, ABAC, rule-based, temporal |
| CP-8 | `sdlc-security` | `security` | Secure SDLC, OWASP, SAST/DAST/IAST, DevSecOps |
| CP-9 | `physical-security` | `security` | Perimeter, surveillance, environmental, fire suppression |
| CP-10 | `security-assessment` | `security` | Pen testing types, vulnerability mgmt lifecycle |
| CP-11 | `forensics-investigation` | `security` | Digital forensics process, chain of custody, evidence |
| CP-12 | `security-awareness` | `security` | Training programs, phishing simulation, metrics |
| CP-13 | `supply-chain-security` | `security` | SBOM, vendor risk, third-party assessment, SCRM |
| CP-14 | `privacy-regulations` | `compliance` | Privacy principles, GDPR deep-dive, CCPA, cross-border |

## 11. C|RAGE — Certified Responsible AI Governance & Ethics (EC-Council)

### Existing Coverage
| Topic | Sheet | Status |
|-------|-------|--------|
| LLM Fundamentals | `llm-fundamentals` | HAVE |
| Prompt Injection | `prompt-injection` | HAVE |
| Prompt Engineering | `prompt-engineering` | HAVE |
| RAG | `rag` | HAVE |
| Transformers | `transformers` | HAVE |
| Threat Modeling | `threat-modeling` | HAVE |
| NIST | `nist` | HAVE |
| ISO 27001 | `iso27001` | HAVE |
| GDPR | `gdpr` | HAVE |
| Incident Response | `incident-response` | HAVE |
| BCP/DRP | `bcp-drp` | PLANNED (CISSP wave) |
| Risk Management | `risk-management` | PLANNED (CISSP wave) |

### Gaps — NEW sheets needed
| # | Sheet Name | Category | Topics |
|---|-----------|----------|--------|
| CR-1 | `ai-governance` | `ai-ml` | AI governance frameworks, operating models, roles, decision rights, AI charters |
| CR-2 | `ai-ethics` | `ai-ml` | AI ethics principles, fairness, transparency, accountability, bias mitigation |
| CR-3 | `ai-risk-management` | `ai-ml` | AI-specific risks, NIST AI RMF, AI threat landscape, adversarial attacks |
| CR-4 | `ai-compliance` | `compliance` | EU AI Act, AI regulatory landscape, audit readiness, continuous compliance |
| CR-5 | `ai-security-architecture` | `ai-ml` | Secure AI design, model protection, pipeline security, runtime security |
| CR-6 | `ai-privacy-trust` | `ai-ml` | Privacy-enhancing tech, differential privacy, federated learning, explainability |
| CR-7 | `ai-testing-assurance` | `ai-ml` | AI testing strategies, model validation, bias testing, robustness, AI auditing |
| CR-8 | `ai-supply-chain` | `ai-ml` | Third-party AI risk, vendor due diligence, model provenance, SBOM for AI |

---

# EXECUTION PLAN

## Wave Structure

Each wave uses 4-5 parallel agents. Each agent creates exactly 2 files:
- `sheets/<category>/<name>.md` — the cheatsheet
- `detail/<category>/<name>.md` — the deep-dive theory page

**After each wave**: `go build -o ./cs ./cmd/cs/ && go test ./... -count=1 -race`

## Wave 1 — Data Center Foundations (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `cisco-nexus` | network-os |
| B | `fibre-channel` | networking |
| C | `fcoe` | networking |
| D | `private-vlans` | networking |
| E | `fhrp` | networking |

## Wave 2 — Data Center Advanced (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `cisco-aci` | networking |
| B | `cisco-ucs` | infrastructure |
| C | `san-storage` | storage |
| D | `span-erspan` | networking |
| E | `vpc` (Cisco vPC) | networking |

## Wave 3 — Data Center + Enterprise Networking (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `fabric-multicast` | networking |
| B | `dc-automation` | config-mgmt |
| C | `roce` | networking |
| D | `eigrp` | networking |
| E | `copp` | security |

## Wave 4 — Data Center + Enterprise (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `nxos-security` | security |
| B | `data-center-design` | networking |
| C | `network-programmability` | networking |
| D | `dmvpn` | networking |
| E | `sd-access` | networking |

## Wave 5 — Enterprise Networking (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `ip-sla` | networking |
| B | `netflow-ipfix` | monitoring |
| C | `cisco-wireless` | networking |
| D | `dot1x` | security |
| E | `macsec` | security |

## Wave 6 — Enterprise Networking (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `network-acl` | networking |
| B | `gre-tunnels` | networking |
| C | `pbr` | networking |
| D | `cisco-dna-center` | networking |
| E | `eem` | config-mgmt |

## Wave 7 — Enterprise + CCIE (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `cisco-ios-xr` | network-os |
| B | `flexvpn` | networking |
| C | `multicast-routing` | networking |
| D | `mpls-te` | networking |
| E | `mpls-vpn` | networking |

## Wave 8 — CCIE Enterprise (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `lisp` | networking |
| B | `isis-advanced` | networking |
| C | `bgp-advanced` | networking |
| D | `ospf-advanced` | networking |
| E | `network-services` | networking |

## Wave 9 — CCIE Enterprise + SP (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `ipv6-advanced` | networking |
| B | `vrf` | networking |
| C | `network-security-infra` | security |
| D | `qos-advanced` | networking |
| E | `carrier-ethernet` | networking |

## Wave 10 — Service Provider (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `unified-mpls` | networking |
| B | `l2vpn-services` | networking |
| C | `evpn-advanced` | networking |
| D | `mvpn` | networking |
| E | `srv6` | networking |

## Wave 11 — Service Provider (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `bng` | networking |
| B | `cgnat` | networking |
| C | `sp-multicast` | networking |
| D | `peering-transit` | networking |
| E | `sp-qos` | networking |

## Wave 12 — Service Provider + Security (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `te-rsvp` | networking |
| B | `g8032-erp` | networking |
| C | `q-in-q` | networking |
| D | `cisco-ftd` | security |
| E | `cisco-ise` | security |

## Wave 13 — CCIE Security (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `trustsec` | security |
| B | `site-to-site-vpn` | security |
| C | `remote-access-vpn` | security |
| D | `email-gateway` | security |
| E | `web-security-proxy` | security |

## Wave 14 — CCIE Security (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `network-access-control` | security |
| B | `cisco-umbrella` | security |
| C | `firewall-design` | security |
| D | `crypto-protocols` | security |
| E | `endpoint-security` | security |

## Wave 15 — CCIE Security + Automation (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `cloud-security` | security |
| B | `security-operations` | security |
| C | `content-security` | security |
| D | `identity-management` | security |
| E | `cisco-nso` | config-mgmt |

## Wave 16 — Automation (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `pyats` | testing |
| B | `gnmi-gnoi` | monitoring |
| C | `netconf` | networking |
| D | `restconf` | networking |
| E | `yang-models` | networking |

## Wave 17 — Automation + Juniper SP (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `nornir` | config-mgmt |
| B | `network-ci-cd` | ci-cd |
| C | `model-driven-telemetry` | monitoring |
| D | `napalm` | config-mgmt |
| E | `junos-mpls-advanced` | juniper |

## Wave 18 — Juniper SP (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `junos-l3vpn` | juniper |
| B | `junos-l2vpn` | juniper |
| C | `junos-evpn-vxlan` | juniper |
| D | `junos-multicast` | juniper |
| E | `junos-class-of-service` | juniper |

## Wave 19 — Juniper SP + SEC (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `junos-high-availability` | juniper |
| B | `junos-segment-routing` | juniper |
| C | `junos-bgp-advanced` | juniper |
| D | `junos-isis-advanced` | juniper |
| E | `junos-bng` | juniper |

## Wave 20 — Juniper SEC (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `junos-nat` | juniper |
| B | `junos-srx` | juniper |
| C | `junos-ipsec-vpn` | juniper |
| D | `junos-utm` | juniper |
| E | `junos-ids-ips` | juniper |

## Wave 21 — Juniper SEC (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `junos-nat-security` | juniper |
| B | `junos-ha-security` | juniper |
| C | `junos-advanced-security` | juniper |
| D | `junos-security-policies` | juniper |
| E | `junos-screens` | juniper |

## Wave 22 — Juniper + Linux+ (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `junos-sky-atp` | juniper |
| B | `sssd` | auth |
| C | `polkit` | security |
| D | `selinux-advanced` | security |
| E | `linux-boot-process` | system |

## Wave 23 — Linux+ (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `linux-storage-management` | storage |
| B | `linux-networking-config` | networking |
| C | `linux-troubleshooting` | system |
| D | `linux-automation-scripting` | shell |
| E | `security-governance` | security |

## Wave 24 — CISSP (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `risk-management` | security |
| B | `bcp-drp` | security |
| C | `security-models` | security |
| D | `asset-security` | security |
| E | `security-architecture` | security |

## Wave 25 — CISSP (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `access-control-models` | security |
| B | `sdlc-security` | security |
| C | `physical-security` | security |
| D | `security-assessment` | security |
| E | `forensics-investigation` | security |

## Wave 26 — CISSP Final + C|RAGE (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `security-awareness` | security |
| B | `supply-chain-security` | security |
| C | `privacy-regulations` | compliance |
| D | `ai-governance` | ai-ml |
| E | `ai-ethics` | ai-ml |

## Wave 27 — C|RAGE (5 agents)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `ai-risk-management` | ai-ml |
| B | `ai-compliance` | compliance |
| C | `ai-security-architecture` | ai-ml |
| D | `ai-privacy-trust` | ai-ml |
| E | `ai-testing-assurance` | ai-ml |

## Wave 28 — C|RAGE Final (1 agent)
| Agent | Sheet | Category |
|-------|-------|----------|
| A | `ai-supply-chain` | ai-ml |

---

# AGENT PROMPT TEMPLATE

Use this template when spawning each agent:

```
You are creating cheatsheet content for the `cs` CLI tool.
Project: /Users/govan/tmp/projects/cheat_sheet/

Create these 2 files:

1. sheets/<CATEGORY>/<NAME>.md — practical cheatsheet
   Format: # Title (Subtitle) → one-liner → ## Sections with code/tables/diagrams → ## Tips → ## See Also → ## References

2. detail/<CATEGORY>/<NAME>.md — theory deep-dive
   Format: # Title — Subtitle → blockquote → numbered ## sections with LaTeX/proofs → ## Prerequisites → ## References

Topic: <TOPIC DESCRIPTION>
Cover: <SPECIFIC SUBTOPICS>

Ensure mkdir -p for new category dirs. Write both files completely — no placeholders.
```

---

# VERIFICATION CHECKLIST

After ALL waves complete:

```bash
# 1. Build
export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH"
go build -o ./cs ./cmd/cs/

# 2. Test
go test ./... -count=1 -race

# 3. Count — should be ~686 (549 + 137)
./cs -l | tail -n +3 | wc -l

# 4. Spot check new categories
./cs cisco-nexus
./cs -d cisco-nexus
./cs fibre-channel
./cs cisco-ftd
./cs junos-srx
./cs bcp-drp

# 5. Search works
./cs search "MPLS"
./cs search "firewall"
./cs search "CISSP"

# 6. TUI shows new categories
./cs -i
```

---

# PROGRESS TRACKER

| Wave | Status | Sheets Created | Running Total |
|------|--------|---------------|---------------|
| 1 | PENDING | 0/5 | 549 |
| 2 | PENDING | 0/5 | 549 |
| 3 | PENDING | 0/5 | 549 |
| 4 | PENDING | 0/5 | 549 |
| 5 | PENDING | 0/5 | 549 |
| 6 | PENDING | 0/5 | 549 |
| 7 | PENDING | 0/5 | 549 |
| 8 | PENDING | 0/5 | 549 |
| 9 | PENDING | 0/5 | 549 |
| 10 | PENDING | 0/5 | 549 |
| 11 | PENDING | 0/5 | 549 |
| 12 | PENDING | 0/5 | 549 |
| 13 | PENDING | 0/5 | 549 |
| 14 | PENDING | 0/5 | 549 |
| 15 | PENDING | 0/5 | 549 |
| 16 | PENDING | 0/5 | 549 |
| 17 | PENDING | 0/5 | 549 |
| 18 | PENDING | 0/5 | 549 |
| 19 | PENDING | 0/5 | 549 |
| 20 | PENDING | 0/5 | 549 |
| 21 | PENDING | 0/5 | 549 |
| 22 | PENDING | 0/5 | 549 |
| 23 | PENDING | 0/5 | 549 |
| 24 | PENDING | 0/5 | 549 |
| 25 | PENDING | 0/5 | 549 |
| 26 | PENDING | 0/5 | 549 |
| 27 | PENDING | 0/5 | 549 |
| 28 | PENDING | 0/1 | 549 |
| **TOTAL** | | **0/137** | **549 → 686** |

---

# CONTEXT RESET RECOVERY

If context maxes out and session resets, the next session should:

1. Read this file: `docs/BATTLE-PLAN.md`
2. Check progress tracker above (update after each wave)
3. Check actual file count: `./cs -l | tail -n +3 | wc -l`
4. Resume from the first PENDING wave
5. Each wave is self-contained — no dependencies between waves
6. Build after each wave: `go build -o ./cs ./cmd/cs/`
7. Commit after every 2-3 waves

**Project path**: `/Users/govan/tmp/projects/cheat_sheet/`
**Build**: `export PATH="/usr/local/go/bin:$HOME/go/bin:$PATH" && go build -o ./cs ./cmd/cs/`
**Test**: `go test ./... -count=1 -race`
**Git email**: `stevie@bellis.tech`
