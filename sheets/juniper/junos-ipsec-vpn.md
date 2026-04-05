# JunOS IPsec VPN

SRX IPsec VPN: route-based (st0 tunnel interface, preferred) or policy-based. IKE phase 1 establishes secure channel; IPsec phase 2 negotiates data encryption. Supports site-to-site, hub-and-spoke, AutoVPN, ADVPN, and Group VPNv2.

## Route-Based VPN (st0 Interface)

### Complete site-to-site route-based VPN
```
# IKE Phase 1 — proposal
set security ike proposal IKE-PROP authentication-method pre-shared-keys
set security ike proposal IKE-PROP dh-group group14
set security ike proposal IKE-PROP authentication-algorithm sha-256
set security ike proposal IKE-PROP encryption-algorithm aes-256-cbc
set security ike proposal IKE-PROP lifetime-seconds 28800

# IKE Phase 1 — policy
set security ike policy IKE-POL mode main
set security ike policy IKE-POL proposals IKE-PROP
set security ike policy IKE-POL pre-shared-key ascii-text "S3cur3K3y!"

# IKE Phase 1 — gateway
set security ike gateway IKE-GW ike-policy IKE-POL
set security ike gateway IKE-GW address 198.51.100.1
set security ike gateway IKE-GW external-interface ge-0/0/1.0
set security ike gateway IKE-GW version v2-only

# IPsec Phase 2 — proposal
set security ipsec proposal IPSEC-PROP protocol esp
set security ipsec proposal IPSEC-PROP authentication-algorithm hmac-sha-256-128
set security ipsec proposal IPSEC-PROP encryption-algorithm aes-256-cbc
set security ipsec proposal IPSEC-PROP lifetime-seconds 3600

# IPsec Phase 2 — policy
set security ipsec policy IPSEC-POL perfect-forward-secrecy keys group14
set security ipsec policy IPSEC-POL proposals IPSEC-PROP

# IPsec VPN
set security ipsec vpn SITE-B ike gateway IKE-GW
set security ipsec vpn SITE-B ike ipsec-policy IPSEC-POL
set security ipsec vpn SITE-B bind-interface st0.0
set security ipsec vpn SITE-B establish-tunnels immediately

# Tunnel interface
set interfaces st0 unit 0 family inet address 10.255.0.1/30
set interfaces st0 unit 0 family inet mtu 1400

# Routing over tunnel
set routing-options static route 172.16.0.0/16 next-hop 10.255.0.2

# Security zone for tunnel
set security zones security-zone vpn interfaces st0.0
set security zones security-zone vpn host-inbound-traffic system-services ike

# Security policy (trust → vpn)
set security policies from-zone trust to-zone vpn policy TO-VPN match source-address any
set security policies from-zone trust to-zone vpn policy TO-VPN match destination-address any
set security policies from-zone trust to-zone vpn policy TO-VPN match application any
set security policies from-zone trust to-zone vpn policy TO-VPN then permit
```

## Policy-Based VPN

```
# Same IKE/IPsec config as above, but NO st0 interface, NO bind-interface

# Policy-based VPN uses proxy-id (traffic selectors embedded in policy)
set security ipsec vpn SITE-B-POLICY ike gateway IKE-GW
set security ipsec vpn SITE-B-POLICY ike ipsec-policy IPSEC-POL
set security ipsec vpn SITE-B-POLICY establish-tunnels immediately

# Security policy triggers the VPN
set security policies from-zone trust to-zone untrust policy VPN-POLICY match source-address LOCAL-NET
set security policies from-zone trust to-zone untrust policy VPN-POLICY match destination-address REMOTE-NET
set security policies from-zone trust to-zone untrust policy VPN-POLICY match application any
set security policies from-zone trust to-zone untrust policy VPN-POLICY then permit tunnel ipsec-vpn SITE-B-POLICY
set security policies from-zone trust to-zone untrust policy VPN-POLICY then permit tunnel pair-policy VPN-RETURN

# Reverse policy
set security policies from-zone untrust to-zone trust policy VPN-RETURN match source-address REMOTE-NET
set security policies from-zone untrust to-zone trust policy VPN-RETURN match destination-address LOCAL-NET
set security policies from-zone untrust to-zone trust policy VPN-RETURN match application any
set security policies from-zone untrust to-zone trust policy VPN-RETURN then permit tunnel ipsec-vpn SITE-B-POLICY
set security policies from-zone untrust to-zone trust policy VPN-RETURN then permit tunnel pair-policy VPN-POLICY
```

## IKE Phase 1 Configuration

### IKEv1 vs IKEv2
```
# IKEv1 main mode (6 messages — identity protected)
set security ike gateway GW version v1-only
set security ike policy POL mode main

# IKEv1 aggressive mode (3 messages — identity exposed, needed for dynamic IP peers)
set security ike policy POL mode aggressive

# IKEv2 (default 4 messages — simpler, supports EAP, MOBIKE, multiple SAs)
set security ike gateway GW version v2-only

# Allow both
set security ike gateway GW version v1-v2
```

### Multiple proposals
```
set security ike proposal STRONG authentication-method pre-shared-keys
set security ike proposal STRONG dh-group group20
set security ike proposal STRONG authentication-algorithm sha-384
set security ike proposal STRONG encryption-algorithm aes-256-gcm

set security ike proposal COMPAT authentication-method pre-shared-keys
set security ike proposal COMPAT dh-group group14
set security ike proposal COMPAT authentication-algorithm sha-256
set security ike proposal COMPAT encryption-algorithm aes-256-cbc

set security ike policy IKE-POL proposals [ STRONG COMPAT ]
# Negotiates in order — tries STRONG first, falls back to COMPAT
```

### Gateway options
```
# Dead Peer Detection (DPD)
set security ike gateway GW dead-peer-detection interval 10
set security ike gateway GW dead-peer-detection threshold 5
# Sends DPD probe every 10 seconds; declares peer dead after 5 missed responses

# NAT-Traversal (NAT-T)
set security ike gateway GW nat-keepalive 20
# IKE auto-detects NAT and encapsulates ESP in UDP 4500
# nat-keepalive sends keepalive every 20 seconds to maintain NAT mapping

# General IKE gateway options
set security ike gateway GW local-identity hostname vpn.example.com
set security ike gateway GW remote-identity hostname remote.example.com
set security ike gateway GW fragmentation enable
```

## IPsec Phase 2 Configuration

### Proposals
```
# ESP with AES-GCM (combined mode — encryption + authentication)
set security ipsec proposal IPSEC-GCM protocol esp
set security ipsec proposal IPSEC-GCM encryption-algorithm aes-256-gcm
set security ipsec proposal IPSEC-GCM lifetime-seconds 3600
set security ipsec proposal IPSEC-GCM lifetime-kilobytes 1024000

# ESP with separate encryption and auth
set security ipsec proposal IPSEC-CBC protocol esp
set security ipsec proposal IPSEC-CBC encryption-algorithm aes-256-cbc
set security ipsec proposal IPSEC-CBC authentication-algorithm hmac-sha-256-128
set security ipsec proposal IPSEC-CBC lifetime-seconds 3600
```

### PFS (Perfect Forward Secrecy)
```
set security ipsec policy IPSEC-POL perfect-forward-secrecy keys group14
# Forces new DH exchange for each phase 2 rekey
# Slightly slower rekey but compromise of one SA key does not expose others
```

## Traffic Selectors

```
# Traffic selectors define which traffic enters the tunnel
# For route-based VPN, they create automatic routes
set security ipsec vpn SITE-B traffic-selector TS1 local-ip 10.0.0.0/8
set security ipsec vpn SITE-B traffic-selector TS1 remote-ip 172.16.0.0/12

# Multiple traffic selectors (each creates a child SA pair)
set security ipsec vpn SITE-B traffic-selector TS2 local-ip 192.168.1.0/24
set security ipsec vpn SITE-B traffic-selector TS2 remote-ip 192.168.2.0/24
```

## Hub-and-Spoke VPN

```
# Hub configuration — st0 in point-to-multipoint mode
set interfaces st0 unit 0 multipoint
set interfaces st0 unit 0 family inet address 10.255.0.1/24

# Hub — static routes to each spoke
set routing-options static route 172.16.1.0/24 next-hop 10.255.0.2
set routing-options static route 172.16.2.0/24 next-hop 10.255.0.3

# Or use dynamic routing (OSPF/BGP over st0)
set protocols ospf area 0.0.0.0 interface st0.0

# Spoke — single default or specific route via hub
set routing-options static route 0.0.0.0/0 next-hop 10.255.0.1

# Hub needs NHRP for dynamic spoke-to-spoke (see ADVPN)
set protocols nhrp tunnel st0.0 nhrp-server 10.255.0.1
```

## AutoVPN

```
# Hub accepts dynamic spokes without pre-configuring each peer

# Hub — IKE gateway with dynamic peer
set security ike gateway AUTO-GW ike-policy IKE-POL
set security ike gateway AUTO-GW dynamic hostname SPOKE-DOMAIN
set security ike gateway AUTO-GW dynamic ike-user-type group-ike-id
set security ike gateway AUTO-GW external-interface ge-0/0/1.0
set security ike gateway AUTO-GW version v2-only

# Hub — VPN with traffic selector (auto-creates routes)
set security ipsec vpn AUTO-VPN ike gateway AUTO-GW
set security ipsec vpn AUTO-VPN ike ipsec-policy IPSEC-POL
set security ipsec vpn AUTO-VPN bind-interface st0.0
set security ipsec vpn AUTO-VPN traffic-selector TS1 local-ip 10.0.0.0/8
set security ipsec vpn AUTO-VPN traffic-selector TS1 remote-ip 0.0.0.0/0

# Hub — st0 multipoint
set interfaces st0 unit 0 multipoint
set interfaces st0 unit 0 family inet address 10.255.0.1/24

# Spoke — standard IKE gateway pointing to hub
set security ike gateway HUB-GW ike-policy IKE-POL
set security ike gateway HUB-GW address 203.0.113.1
set security ike gateway HUB-GW local-identity hostname spoke1.example.com
set security ike gateway HUB-GW external-interface ge-0/0/0.0
set security ike gateway HUB-GW version v2-only
```

## ADVPN (Auto Discovery VPN)

```
# Enables direct spoke-to-spoke tunnels (shortcut switching)
# Traffic initially goes through hub, then NHRP creates direct tunnel

# Hub — enable ADVPN
set security ipsec vpn HUB-VPN vpn-monitor
set security ipsec vpn HUB-VPN advpn suggester enable
set security ipsec vpn HUB-VPN advpn suggester disable-partner-network-identification

# Hub — NHRP
set protocols nhrp tunnel st0.0 nhrp-server 10.255.0.1
set protocols nhrp tunnel st0.0 shortcut-target 0.0.0.0/0

# Spoke — enable ADVPN
set security ipsec vpn SPOKE-VPN advpn partner enable
set security ipsec vpn SPOKE-VPN advpn partner idle-threshold 60
set security ipsec vpn SPOKE-VPN advpn partner connection-limit 10

# Spoke — NHRP
set protocols nhrp tunnel st0.0 nhrp-server 10.255.0.1
set protocols nhrp tunnel st0.0 shortcut-target 0.0.0.0/0
```

## Group VPNv2

```
# Group VPN — single SA shared by all group members
# Preserves original IP headers (no tunnel encapsulation overhead)
# Used for any-to-any encryption within a WAN

# Group Server (Key Server)
set security group-vpn server group GVPN ike-gateway MEMBER1
set security group-vpn server group GVPN ike-gateway MEMBER2
set security group-vpn server group GVPN group-sa-lifetime 3600
set security group-vpn server group GVPN anti-replay time-based
set security group-vpn server group GVPN server-address 10.0.0.1
set security group-vpn server group GVPN match-policy ENCRYPT match source-address 10.0.0.0/8
set security group-vpn server group GVPN match-policy ENCRYPT match destination-address 10.0.0.0/8
set security group-vpn server group GVPN match-policy ENCRYPT then aes-256-cbc sha-256

# Group Member
set security group-vpn member ike gateway KEY-SERVER address 10.0.0.1
set security group-vpn member ike gateway KEY-SERVER ike-policy GV-IKE-POL
set security group-vpn member ipsec vpn GVPN-MEMBER group 1
set security group-vpn member ipsec vpn GVPN-MEMBER ike-gateway KEY-SERVER
set security group-vpn member ipsec vpn GVPN-MEMBER group-vpn-external-interface ge-0/0/1.0
set security group-vpn member ipsec vpn GVPN-MEMBER recovery-probe
```

## Certificate-Based Authentication

```
# Load CA certificate
request security pki ca-certificate load ca-profile ROOT-CA filename /var/tmp/ca-cert.pem

# Generate local key pair
request security pki generate-key-pair certificate-id SRX1-CERT size 2048 type rsa

# Generate CSR
request security pki generate-certificate-request certificate-id SRX1-CERT domain-name vpn.example.com subject "CN=vpn.example.com,O=Example,C=US" filename /var/tmp/srx1.csr

# Load signed certificate
request security pki local-certificate load certificate-id SRX1-CERT filename /var/tmp/srx1-cert.pem

# IKE with certificates
set security ike proposal CERT-PROP authentication-method rsa-signatures
set security ike proposal CERT-PROP dh-group group14
set security ike proposal CERT-PROP authentication-algorithm sha-256
set security ike proposal CERT-PROP encryption-algorithm aes-256-cbc

set security ike policy CERT-POL proposals CERT-PROP
set security ike policy CERT-POL certificate local-certificate SRX1-CERT

set security ike gateway CERT-GW ike-policy CERT-POL
set security ike gateway CERT-GW address 198.51.100.1
set security ike gateway CERT-GW external-interface ge-0/0/1.0
set security ike gateway CERT-GW local-identity distinguished-name
set security ike gateway CERT-GW remote-identity distinguished-name
set security ike gateway CERT-GW version v2-only

# CRL checking
set security pki ca-profile ROOT-CA revocation-check crl
set security pki ca-profile ROOT-CA revocation-check crl url http://crl.example.com/root.crl
```

## DPD (Dead Peer Detection)

```
set security ike gateway GW dead-peer-detection interval 10
set security ike gateway GW dead-peer-detection threshold 5

# DPD modes (IKEv1):
#   optimized (default) — send DPD only when there is outbound traffic but no inbound
#   probe-idle-tunnel    — send DPD even on idle tunnels
#   always-send          — send DPD on every interval regardless of traffic

set security ike gateway GW dead-peer-detection always-send
```

## NAT-Traversal (NAT-T)

```
# Auto-detected during IKE negotiation
# When NAT is detected, ESP is encapsulated in UDP port 4500

# Keepalive to maintain NAT mapping
set security ike gateway GW nat-keepalive 20

# Force NAT-T even without NAT detection (rare)
set security ike gateway GW force-nat-t
```

## VPN Monitoring

```
# ICMP-based VPN monitoring
set security ipsec vpn SITE-B vpn-monitor source-interface ge-0/0/0.0
set security ipsec vpn SITE-B vpn-monitor destination-ip 10.255.0.2
set security ipsec vpn SITE-B vpn-monitor optimized

# If monitoring fails, tunnel is torn down and re-established
```

## HA with VPN (Chassis Cluster)

```
# VPN bound to reth interface (follows RG failover)
set security ike gateway GW external-interface reth0.0

# IPsec SA sync between nodes
set chassis cluster redundancy-group 1 node 0 priority 200
set chassis cluster redundancy-group 1 node 1 priority 100

# VPN tunnel failover:
# - IKE SA and IPsec SA synced to secondary node
# - On failover, secondary activates synced SAs
# - Brief traffic disruption during RG switchover + GARP
# - If SA sync fails, full IKE re-negotiation occurs
```

## Verification Commands

```
# IKE Phase 1
show security ike security-associations
show security ike security-associations detail
show security ike active-peer

# IPsec Phase 2
show security ipsec security-associations
show security ipsec security-associations detail
show security ipsec statistics

# VPN status
show security ipsec inactive-tunnels
show security ipsec next-hop-tunnels st0.0

# Tunnel interface
show interfaces st0.0
show interfaces st0.0 extensive

# AutoVPN / ADVPN
show security ipsec security-associations vpn-name AUTO-VPN
show security ike security-associations family inet
show security ipsec next-hop-tunnels st0.0

# Group VPN
show security group-vpn member ipsec security-associations
show security group-vpn server registered-members

# Certificate status
show security pki ca-certificate
show security pki local-certificate
show security pki crl

# Troubleshooting
show security ike traceoptions
show log kmd

# Clear and restart
clear security ike security-associations
clear security ipsec security-associations
request security ipsec sa-statistics clear
```

## Tips

- Route-based VPN is almost always preferred — supports dynamic routing, multicast, redundant tunnels, traffic engineering
- Policy-based VPN creates one SA pair per policy match — multiple policies = multiple SAs
- Always enable DPD to detect dead peers and trigger re-establishment
- NAT-T is auto-detected but verify with `show security ike sa detail` (look for NAT-T enabled)
- Traffic selectors on route-based VPN auto-create routes — no need for manual static routes when using them
- PFS adds security but increases rekey time — standard practice for JNCIE-SEC
- AutoVPN simplifies hub config but requires consistent PSK/cert management across all spokes
- ADVPN shortcut tunnels time out after idle-threshold seconds — tune based on traffic patterns
- Group VPNv2 preserves original IP headers (transport mode) — network monitoring/QoS still works
- VPN + chassis cluster: always use reth as external-interface, never a physical interface
- MTU on st0: set to 1400 for standard ESP or 1380 for ESP+NAT-T to avoid fragmentation

## See Also

- junos-srx, junos-nat, junos-high-availability, ipsec, junos-firewall-filters

## References

- [Juniper TechLibrary — Route-Based VPN](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/concept/ipsec-route-based-vpn-overview.html)
- [Juniper TechLibrary — Policy-Based VPN](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/concept/ipsec-policy-based-vpn-overview.html)
- [Juniper TechLibrary — AutoVPN](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/concept/auto-discovery-vpn-overview.html)
- [Juniper TechLibrary — ADVPN](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/concept/advpn-overview.html)
- [Juniper TechLibrary — Group VPNv2](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/concept/group-vpn-overview.html)
- [Juniper TechLibrary — Certificate-Based Authentication](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/concept/certificate-based-authentication-overview.html)
- [RFC 7296 — IKEv2](https://www.rfc-editor.org/rfc/rfc7296)
- [RFC 4301 — IPsec Architecture](https://www.rfc-editor.org/rfc/rfc4301)
- [RFC 3948 — UDP Encapsulation of ESP (NAT-T)](https://www.rfc-editor.org/rfc/rfc3948)
