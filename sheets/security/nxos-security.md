# NX-OS Security (AAA, RBAC, and First-Hop Security)

> Quick-reference commands and configurations for securing Cisco Nexus switches running NX-OS.

---

## AAA — Authentication, Authorization, and Accounting

### Enable AAA Globally

```
feature tacacs+
feature radius

aaa authentication login default group tacacs+ local
aaa authentication login console local
aaa authorization commands default group tacacs+ local
aaa accounting default group tacacs+
```

### TACACS+ Server Configuration

```
tacacs-server host 10.1.1.50 key 0 T4c4csK3y!
tacacs-server host 10.1.1.51 key 0 T4c4csK3y!
tacacs-server timeout 10
tacacs-server deadtime 5

aaa group server tacacs+ TACACS_GROUP
  server 10.1.1.50
  server 10.1.1.51
  source-interface mgmt0
  use-vrf management
```

### RADIUS Server Configuration

```
radius-server host 10.1.1.60 key 0 R4d1usK3y! auth-port 1812 acct-port 1813
radius-server host 10.1.1.61 key 0 R4d1usK3y! auth-port 1812 acct-port 1813
radius-server timeout 5
radius-server retransmit 2

aaa group server radius RADIUS_GROUP
  server 10.1.1.60
  server 10.1.1.61
  source-interface mgmt0
  use-vrf management
```

### Local User Authentication (Fallback)

```
username admin password 5 $1$xyz$hash role network-admin
username backup-admin password 5 $1$abc$hash role network-admin
username monitor password 5 $1$def$hash role network-operator
```

### AAA Accounting

```
aaa accounting default group tacacs+
aaa accounting commands default group tacacs+

! Verify accounting records
show aaa accounting
show tacacs-server
show radius-server
```

### Verify AAA

```
show aaa authentication
show aaa authorization
show aaa accounting
show aaa groups
show tacacs-server
show radius-server
test aaa server tacacs+ 10.1.1.50 username admin password Cisco123
test aaa server radius 10.1.1.60 username admin password Cisco123
```

---

## RBAC — Role-Based Access Control

### Built-in Roles

| Role              | Privileges                                      |
|-------------------|--------------------------------------------------|
| network-admin     | Full read-write to entire device                 |
| network-operator  | Read-only access to show commands                |
| vdc-admin         | Full access within a VDC (Nexus 7000)            |
| vdc-operator      | Read-only within a VDC (Nexus 7000)              |

### Create a Custom Role

```
role name SEC_ADMIN
  description Security administration role
  rule 1 permit command show running-config
  rule 2 permit command show startup-config
  rule 3 permit command configure terminal
  rule 4 permit command interface *
  rule 5 permit command access-list *
  rule 6 permit command show access-list *
  rule 7 deny command reload
  rule 8 deny command write erase
  rule 10 permit read-write feature aclmgr
  rule 11 permit read-write feature dhcp-snoop
  rule 12 permit read-write feature arp-inspection
```

### Feature-Based RBAC Rules

```
role name ROUTING_ADMIN
  rule 1 permit read-write feature ospf
  rule 2 permit read-write feature bgp
  rule 3 permit read-write feature eigrp
  rule 4 permit read feature interface
  rule 5 deny command reload

role name VLAN_ADMIN
  rule 1 permit read-write feature vlan_mgr
  rule 2 permit read-write feature interface
  rule 3 permit command show vlan *
```

### Assign Roles to Users

```
username sec-ops password 5 $1$ghi$hash role SEC_ADMIN
username rtr-ops password 5 $1$jkl$hash role ROUTING_ADMIN
```

### Verify RBAC

```
show role
show role name SEC_ADMIN
show user-account
show users
```

---

## DHCP Snooping

### Enable and Configure

```
feature dhcp
ip dhcp snooping
ip dhcp snooping vlan 10,20,30

! Trust uplinks and DHCP server ports
interface Ethernet1/1
  ip dhcp snooping trust

interface Ethernet1/2
  ip dhcp snooping trust

! All access ports are untrusted by default — no extra config needed

! Rate-limit DHCP packets on untrusted ports
interface Ethernet1/10
  ip dhcp snooping limit rate 15
```

### DHCP Snooping Binding Table

```
show ip dhcp snooping
show ip dhcp snooping binding
show ip dhcp snooping statistics

! Persist the binding table across reboots
ip dhcp snooping database bootflash:dhcp_snoop_db
```

### Verify DHCP Snooping

```
show ip dhcp snooping
show ip dhcp snooping binding
show ip dhcp snooping binding vlan 10
show ip dhcp snooping statistics
show ip dhcp snooping database
```

---

## Dynamic ARP Inspection (DAI)

### Enable DAI

```
ip arp inspection vlan 10,20,30

! Trust uplinks (skip ARP validation on these)
interface Ethernet1/1
  ip arp inspection trust

interface Ethernet1/2
  ip arp inspection trust
```

### ARP ACLs for Static Hosts

```
arp access-list STATIC_ARP
  permit ip host 10.10.10.5 mac host 0000.1111.2222
  permit ip host 10.10.10.6 mac host 0000.3333.4444

ip arp inspection filter STATIC_ARP vlan 10
```

### DAI Validation

```
ip arp inspection validate src-mac dst-mac ip

! Rate-limit ARP on untrusted ports (packets per second)
interface Ethernet1/10
  ip arp inspection limit rate 30
```

### Verify DAI

```
show ip arp inspection
show ip arp inspection vlan 10
show ip arp inspection statistics
show ip arp inspection interfaces
show ip arp inspection log
```

---

## IP Source Guard (IPSG)

### Enable IPSG

```
! Requires DHCP snooping to be active on the VLAN
interface Ethernet1/10
  ip verify source dhcp-snooping-vlan

! For static bindings (no DHCP)
ip source binding 10.10.10.5 0000.1111.2222 vlan 10 interface Ethernet1/10
```

### Verify IPSG

```
show ip verify source
show ip verify source interface Ethernet1/10
show ip dhcp snooping binding
```

---

## Port Security

### Configure Port Security

```
interface Ethernet1/10
  switchport
  switchport mode access
  switchport port-security
  switchport port-security maximum 3
  switchport port-security violation restrict
  switchport port-security mac-address sticky
  switchport port-security aging time 120
  switchport port-security aging type inactivity
```

### Violation Modes

| Mode       | Action                                                     |
|------------|-------------------------------------------------------------|
| shutdown   | Err-disables the port (default)                            |
| restrict   | Drops violating frames, sends SNMP trap, increments counter |
| protect    | Silently drops violating frames                            |

### Verify Port Security

```
show port-security
show port-security interface Ethernet1/10
show port-security address
```

---

## Storm Control

### Configure Storm Control

```
interface Ethernet1/10
  storm-control broadcast level 10.00
  storm-control multicast level 10.00
  storm-control unicast level 5.00
  storm-control action trap
  storm-control action shutdown
```

### Verify Storm Control

```
show storm-control
show storm-control broadcast
show storm-control interface Ethernet1/10
```

---

## First-Hop Security for IPv6

### RA Guard

```
feature dhcp

ipv6 nd raguard policy HOST_POLICY
  device-role host

ipv6 nd raguard policy ROUTER_POLICY
  device-role router

interface Ethernet1/10
  ipv6 nd raguard attach-policy HOST_POLICY

interface Ethernet1/1
  ipv6 nd raguard attach-policy ROUTER_POLICY
```

### DHCPv6 Guard

```
ipv6 dhcp guard policy SERVER_POLICY
  device-role server

ipv6 dhcp guard policy CLIENT_POLICY
  device-role client

interface Ethernet1/1
  ipv6 dhcp guard attach-policy SERVER_POLICY

interface Ethernet1/10
  ipv6 dhcp guard attach-policy CLIENT_POLICY
```

### Verify IPv6 First-Hop Security

```
show ipv6 nd raguard policy HOST_POLICY
show ipv6 dhcp guard policy SERVER_POLICY
show ipv6 snooping policies
```

---

## NX-OS Hardening

### SSH Key-Based Authentication

```
feature ssh

ssh key rsa 2048
username admin sshkey ssh-rsa AAAAB3...== stevie@bellis.tech

! Disable password authentication for SSH
no ssh password-auth enable

ssh login-gracetime 60
ssh login-attempts 3
```

### Password Policies

```
password strength-check
username admin password-expiry max-lifetime 90
username admin password-expiry warn-interval 14

! Set password minimum length
security password-strength min-length 12
```

### Login Banner

```
banner motd #
*************************************************************
*  UNAUTHORIZED ACCESS PROHIBITED                          *
*  All sessions are logged and monitored.                  *
*  Disconnect immediately if you are not authorized.       *
*************************************************************
#
```

### Management ACL

```
ip access-list MGMT_ACL
  10 permit tcp 10.0.0.0/8 any eq 22
  20 permit tcp 10.0.0.0/8 any eq 443
  30 permit udp 10.0.0.0/8 any eq 161
  40 deny ip any any log

line vty
  access-class MGMT_ACL in
```

### Disable Unnecessary Services

```
no feature telnet
no feature nxapi       ! unless required
no ip domain-lookup
```

---

## MACsec on Nexus

### Enable MACsec

```
feature macsec

key chain MACSEC_KC macsec
  key 01
    key-octet-string 7 <hex-key> cryptographic-algorithm aes-256-cmac
    lifetime 00:00:00 Jan 01 2026 infinite

macsec policy MACSEC_POL
  cipher-suite GCM-AES-256
  security-policy must-secure
  sak-expiry-time 120
  include-icv-indicator
  key-server-priority 0

interface Ethernet1/49
  macsec keychain MACSEC_KC policy MACSEC_POL
```

### Verify MACsec

```
show macsec policy
show macsec mka session
show macsec mka summary
show macsec mka statistics interface Ethernet1/49
show macsec secy statistics interface Ethernet1/49
```

---

## Keychain Management

### Create and Apply Keychains

```
key chain OSPF_KC
  key 1
    key-string 7 <encrypted-key>
    accept-lifetime 00:00:00 Jan 01 2026 infinite
    send-lifetime 00:00:00 Jan 01 2026 infinite
    cryptographic-algorithm HMAC-SHA-256

router ospf 1
  area 0.0.0.0 authentication message-digest
  interface Ethernet1/1
    authentication key-chain OSPF_KC
```

### Verify Keychains

```
show key chain
show key chain OSPF_KC
```

---

## System-Level Security

### NX-API HTTPS-Only

```
feature nxapi
nxapi http port 80     ! disable or redirect
nxapi https port 443
no nxapi http

! Restrict NX-API access
nxapi use-vrf management
nxapi sandbox           ! disable in production
no nxapi sandbox
```

### NX-OS Integrity Verification

```
show system integrity all
show system integrity filesystem bootflash:

! Verify running image
show install all impact nxos bootflash:nxos.10.4.1.F.bin
show version image bootflash:nxos.10.4.1.F.bin
```

### CoPP — Control Plane Policing

```
show copp status
show policy-map interface control-plane

! Apply stricter CoPP profile
copp profile strict
```

### Console and VTY Hardening

```
line console
  exec-timeout 5
  login authentication console-auth

line vty
  session-limit 5
  exec-timeout 10
  login authentication default
  transport input ssh
  transport output ssh
```

---

## Tips

- Always configure a local fallback user when relying on remote AAA servers
  (`aaa authentication login default group tacacs+ local`).
- Set `deadtime` on TACACS+/RADIUS servers so the switch stops retrying
  unreachable servers and falls back faster.
- DHCP snooping must be active before DAI or IPSG can function; they rely
  on the snooping binding table.
- Persist the DHCP snooping binding table to bootflash so it survives reboots.
- Use `ip arp inspection validate src-mac dst-mac ip` to catch ARP spoofing
  that manipulates both L2 and L3 headers.
- Port security `sticky` MACs survive reboots only if you `copy running
  startup` — they are written to the running config.
- Apply storm-control on all access ports; a single broadcast storm can
  saturate an entire VLAN.
- RA Guard `device-role host` blocks all router advertisements on access ports
  — essential for preventing IPv6 MITM attacks.
- MACsec `must-secure` policy drops all non-MACsec traffic; use `should-secure`
  during migration to allow fallback to cleartext.
- Disable NX-API sandbox in production — it exposes a web-based CLI on the
  management interface.
- Run `show system integrity all` after every ISSU or image upgrade to confirm
  the running image matches Cisco's signed hash.
- Set CoPP to `strict` profile to harden the supervisor against control-plane
  floods (ARP, ICMP, BGP, OSPF, LACP).
- Avoid `network-admin` for day-to-day operations; create purpose-specific
  RBAC roles with least-privilege rules.

---

## See Also

- `cs show nxos-acl` — NX-OS access control lists
- `cs show nxos-aaa` — detailed AAA troubleshooting
- `cs show tacacs` — TACACS+ protocol deep dive
- `cs show radius` — RADIUS protocol reference
- `cs show macsec` — MACsec and MKA reference
- `cs show copp` — CoPP profiles and tuning
- `cs show ipv6-firsthop` — IPv6 first-hop security suite

---

## References

- Cisco NX-OS Security Configuration Guide, Release 10.x
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x.html
- Cisco NX-OS RBAC Configuration Guide
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-user-accounts-and-rbac.html
- Cisco NX-OS DHCP Snooping and DAI Configuration
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-dhcp.html
- Cisco NX-OS MACsec Configuration Guide
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-macsec.html
- Cisco NX-OS CoPP Configuration Guide
  https://www.cisco.com/c/en/us/td/docs/dcn/nx-os/nexus9000/104x/configuration/security/cisco-nexus-9000-nx-os-security-configuration-guide-104x/m-configuring-copp.html
- NIST SP 800-53 Rev. 5 — Security and Privacy Controls
  https://csrc.nist.gov/publications/detail/sp/800-53/rev-5/final
