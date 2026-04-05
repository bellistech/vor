# Cisco IOS XR (Service Provider Network Operating System)

Microkernel-based network OS for carrier-grade routing platforms with process isolation, commit-based configuration, and in-service upgrades.

## Architecture Overview

### Process Model

```
+--------------------------------------------------+
|              IOS XR Process Architecture          |
+--------------------------------------------------+
| Management Plane                                  |
|   CLI, XML, NETCONF, gRPC, SNMP agents           |
+--------------------------------------------------+
| Control Plane (per-process restart)               |
|   BGP, OSPF, IS-IS, MPLS, LDP, RSVP, PIM        |
|   Each runs as independent process                |
+--------------------------------------------------+
| Data Plane                                        |
|   CEF/FIB distributed to line cards               |
|   Forwarding survives control plane restarts      |
+--------------------------------------------------+
| Kernel: QNX Neutrino (classic) / Linux (eXR)     |
|   Memory protection, process isolation, IPC       |
+--------------------------------------------------+
```

### Key Architecture Differences from IOS

| Feature | IOS | IOS XR |
|:---|:---|:---|
| Kernel | Monolithic | Microkernel (QNX or Linux) |
| Process model | Single process | Multi-process, isolated |
| Config model | Immediate apply | Commit-based |
| Software updates | Full image reload | SMU, package-based, ISSU |
| Failure recovery | Full reload | Process restart |
| HA model | SSO/NSR | NSR + process restart |
| Management | CLI/SNMP | CLI/XML/NETCONF/gRPC |

## Configuration Commit Model

### Enter Configuration Mode

```
! Standard configuration mode
configure terminal

! Exclusive configuration mode (locks out other users)
configure exclusive

! Shorthand
conf t
```

### Commit Operations

```
! Apply pending changes
commit

! Commit with comment
commit comment "Added BGP neighbor 10.0.0.1"

! Commit with automatic rollback timer (confirmed commit)
commit confirmed 120
! If not confirmed within 120 seconds, config rolls back automatically

! Confirm a pending confirmed commit
commit

! Commit best-effort (apply what can be applied, skip errors)
commit best-effort

! Commit and immediately replace running with candidate
commit replace
! WARNING: this replaces the ENTIRE running config with the candidate
```

### View Pending Changes

```
! Show uncommitted changes
show configuration

! Show changes as diff
show commit changes diff

! Show running config vs candidate
show configuration merge

! Show configuration history
show configuration history

! Show commit list
show configuration commit list
```

### Discard and Rollback

```
! Discard all uncommitted changes
abort

! Discard specific changes and stay in config mode
clear

! Rollback to a previous commit
rollback configuration to <commit-id>

! Rollback last N commits
rollback configuration last 1

! Show available rollback points
show configuration commit list

! Compare current config with a rollback point
show configuration rollback changes <commit-id>
```

### Configuration Replace

```
! Load a full config file and replace running config
load <url>
commit replace

! Example: load from TFTP
load tftp://10.0.0.5/router-config.cfg
show configuration
commit replace
```

## Admin Mode

### Enter Admin Configuration

```
! Admin mode for system-level operations
admin

! Admin configuration mode
admin configure terminal

! Show admin running config
admin show running-config

! Exit admin mode
exit
```

### Admin-Level Operations

```
! Install operations (admin mode)
admin install add source tftp://10.0.0.5/ <package.rpm>
admin install activate <package>
admin install commit

! Reload from admin mode
admin reload location all

! Hardware operations
admin show platform
admin show inventory
```

## Interface Naming and Configuration

### Interface Naming Convention

```
! Format: type rack/slot/module/port
! Single RSP systems: rack=0, slot=0
! Multi-chassis: rack varies

! Physical interfaces
interface GigabitEthernet0/0/0/0
interface TenGigE0/1/0/0
interface HundredGigE0/0/0/0
interface Bundle-Ether1
interface Loopback0

! Sub-interfaces
interface GigabitEthernet0/0/0/0.100
interface Bundle-Ether1.200

! Management
interface MgmtEth0/RSP0/CPU0/0
```

### Basic Interface Configuration

```
interface GigabitEthernet0/0/0/0
 description UPLINK-TO-PE2
 ipv4 address 10.0.0.1 255.255.255.252
 ipv6 address 2001:db8::1/64
 no shutdown
!
commit
```

### Bundle (LAG) Configuration

```
interface Bundle-Ether1
 description LAG-TO-CORE
 ipv4 address 10.0.1.1 255.255.255.252
 lacp mode active
 bundle minimum-active links 2
!
interface GigabitEthernet0/0/0/0
 bundle id 1 mode active
!
interface GigabitEthernet0/0/0/1
 bundle id 1 mode active
!
commit
```

### BVI (Bridge Virtual Interface)

```
! Layer 2 bridge domain with BVI for IRB
interface BVI1
 ipv4 address 192.168.1.1 255.255.255.0
!
l2vpn
 bridge group OFFICE
  bridge-domain BD1
   interface GigabitEthernet0/0/0/2
   !
   interface GigabitEthernet0/0/0/3
   !
   routed interface BVI1
  !
 !
!
commit
```

## VRF Configuration

### Define a VRF

```
vrf CUSTOMER-A
 address-family ipv4 unicast
  import route-target
   65000:100
  !
  export route-target
   65000:100
  !
 !
 address-family ipv6 unicast
  import route-target
   65000:100
  !
  export route-target
   65000:100
  !
 !
!
commit
```

### Assign Interface to VRF

```
interface GigabitEthernet0/0/0/5
 vrf CUSTOMER-A
 ipv4 address 172.16.0.1 255.255.255.252
 no shutdown
!
commit
```

### VRF-Aware Routing

```
router bgp 65000
 vrf CUSTOMER-A
  rd 65000:100
  address-family ipv4 unicast
   redistribute connected
  !
  neighbor 172.16.0.2
   remote-as 65001
   address-family ipv4 unicast
    route-policy CUST-A-IN in
    route-policy CUST-A-OUT out
   !
  !
 !
!
commit
```

## Routing Protocol Configuration

### OSPF

```
router ospf 1
 router-id 10.255.0.1
 area 0
  interface Loopback0
   passive enable
  !
  interface GigabitEthernet0/0/0/0
   network point-to-point
   cost 10
  !
  interface GigabitEthernet0/0/0/1
   network point-to-point
   cost 100
  !
 !
 area 1
  interface GigabitEthernet0/0/0/2
   authentication message-digest
   message-digest-key 1 md5 encrypted <hash>
  !
 !
!
commit
```

### IS-IS

```
router isis CORE
 is-type level-2-only
 net 49.0001.0100.0000.0001.00
 address-family ipv4 unicast
  metric-style wide
  mpls traffic-eng level-2-only
  mpls traffic-eng router-id Loopback0
 !
 address-family ipv6 unicast
  metric-style wide
 !
 interface Loopback0
  passive
  address-family ipv4 unicast
  !
  address-family ipv6 unicast
  !
 !
 interface GigabitEthernet0/0/0/0
  point-to-point
  address-family ipv4 unicast
   metric 10
  !
 !
!
commit
```

### BGP

```
router bgp 65000
 bgp router-id 10.255.0.1
 bgp log neighbor changes detail
 address-family ipv4 unicast
 !
 address-family vpnv4 unicast
 !
 address-family ipv6 unicast
 !
 neighbor-group IBGP-PEERS
  remote-as 65000
  update-source Loopback0
  address-family ipv4 unicast
   next-hop-self
  !
  address-family vpnv4 unicast
  !
 !
 neighbor 10.255.0.2
  use neighbor-group IBGP-PEERS
  description PE2
 !
 neighbor 10.255.0.3
  use neighbor-group IBGP-PEERS
  description PE3
 !
!
commit
```

### MPLS LDP

```
mpls ldp
 router-id 10.255.0.1
 address-family ipv4
 !
 interface GigabitEthernet0/0/0/0
 !
 interface GigabitEthernet0/0/0/1
 !
!
commit
```

### Route Policy (RPL)

```
! IOS XR uses Route Policy Language (RPL) instead of route-maps
route-policy CUST-A-IN
 if destination in prefix-set CUST-A-PREFIXES then
  set local-preference 200
  pass
 else
  drop
 endif
end-policy

prefix-set CUST-A-PREFIXES
 172.16.0.0/16 le 24,
 192.168.0.0/16 le 24
end-set

! Apply to BGP neighbor
router bgp 65000
 neighbor 10.0.0.2
  address-family ipv4 unicast
   route-policy CUST-A-IN in
   route-policy PERMIT-ALL out
  !
 !
!
commit
```

## Install Operations (Software Management)

### Package Management

```
! Show installed packages
show install active summary

! Show available packages
show install repository

! Add package from remote source
install add source tftp://10.0.0.5/packages/ <package.rpm>

! Activate a package
install activate <package-name>

! Commit installed packages (makes them persistent across reload)
install commit

! Deactivate a package
install deactivate <package-name>

! Remove a package
install remove <package-name>
```

### SMU (Software Maintenance Update)

```
! Add SMU
install add source tftp://10.0.0.5/smus/ <smu.rpm>

! Activate SMU
install activate <smu-name>

! Verify SMU
show install active summary
show install log

! Commit SMU
install commit

! Rollback SMU
install rollback to <install-id>
```

### ISSU (In-Service Software Upgrade)

```
! Check ISSU readiness
show issu

! Prepare ISSU
install extract <package>

! Execute ISSU (platform-dependent)
install activate issu <package>

! Monitor ISSU progress
show issu
show install log
```

## AAA Configuration

### TACACS+

```
aaa group server tacacs+ TAC-SERVERS
 server-private 10.0.0.10 port 49
  key 7 <encrypted-key>
 !
 server-private 10.0.0.11 port 49
  key 7 <encrypted-key>
 !
!

aaa authentication login default group TAC-SERVERS local
aaa authorization exec default group TAC-SERVERS local
aaa authorization commands default group TAC-SERVERS local
aaa accounting exec default start-stop group TAC-SERVERS
aaa accounting commands default start-stop group TAC-SERVERS

commit
```

### Local Users

```
username admin
 group root-lr
 group cisco-support
 secret 10 $6$<hash>
!
username operator
 group operator
 secret 10 $6$<hash>
!
commit
```

### Task-Based Authorization (XR-Specific)

```
! IOS XR uses task-based authorization instead of privilege levels
! Built-in groups: root-lr, root-system, cisco-support, operator, sysadmin

! Custom task group
taskgroup MONITORING
 task read bgp
 task read ospf
 task read interface
 task read logging
!
usergroup MONITOR-USERS
 taskgroup MONITORING
!
username monitor
 group MONITOR-USERS
!
commit
```

## XML/NETCONF/gRPC

### NETCONF Configuration

```
! Enable NETCONF agent
netconf-yang agent
 ssh
!
ssh server netconf port 830
ssh server v2

commit

! Test NETCONF from client
ssh admin@10.0.0.1 -p 830 -s netconf
```

### gRPC Configuration

```
! Enable gRPC server
grpc
 port 57400
 no-tls
 address-family dual
!
commit

! Model-Driven Telemetry (MDT) via gRPC
telemetry model-driven
 sensor-group INTF-COUNTERS
  sensor-path Cisco-IOS-XR-infra-statsd-oper:infra-statistics/interfaces/interface/latest/generic-counters
 !
 subscription INTF-SUB
  sensor-group-id INTF-COUNTERS sample-interval 30000
  destination-id COLLECTOR
 !
 destination-group COLLECTOR
  address-family ipv4 10.0.0.100 port 57500
   encoding self-describing-gpb
   protocol grpc no-tls
  !
 !
!
commit
```

### XML Agent

```
! Enable XML agent
xml agent tty
!
xml agent ssl
 iteration off
!
commit
```

## Show Commands (Operational)

### Platform and Hardware

```
! System information
show version
show platform
show inventory
show redundancy

! RSP/LC status
show platform vm
show controllers card-manager inventory

! Environment
show environment temperatures
show environment power
show environment fans
```

### Interface Status

```
! Interface summary
show ip interface brief
show ipv4 interface brief
show ipv6 interface brief

! Detailed interface
show interface GigabitEthernet0/0/0/0
show interface GigabitEthernet0/0/0/0 accounting

! Bundle status
show bundle Bundle-Ether1
show lacp Bundle-Ether1
```

### Routing

```
! Routing table
show route
show route ipv4
show route ipv6
show route vrf CUSTOMER-A

! BGP
show bgp summary
show bgp ipv4 unicast summary
show bgp vpnv4 unicast summary
show bgp neighbors 10.255.0.2
show bgp ipv4 unicast 10.0.0.0/8

! OSPF
show ospf neighbor
show ospf database
show ospf interface

! IS-IS
show isis neighbors
show isis database
show isis interface
show isis route

! MPLS
show mpls ldp neighbor
show mpls ldp bindings
show mpls forwarding
show mpls label table
```

### Configuration Verification

```
! Show running config (entire)
show running-config

! Show running config for specific section
show running-config router bgp
show running-config interface GigabitEthernet0/0/0/0
show running-config router ospf

! Show config differences
show configuration commit changes last 1
show running-config | diff
```

### Process and System

```
! Process status
show processes
show processes cpu
show processes memory

! Logging
show logging
show logging last 50

! System clock
show clock

! Users
show users
```

## RSP/LC Separation

### Route Switch Processor (RSP) Management

```
! Show RSP status
show platform vm
show redundancy

! Switchover (HA)
redundancy switchover

! Reload specific RSP
hw-module location 0/RSP0/CPU0 reload

! Process restart (non-disruptive)
process restart bgp
process restart ospf
```

### Line Card Management

```
! Show line card status
show platform
admin show platform

! Reload a line card
hw-module location 0/1/CPU0 reload

! Shut down a line card
hw-module location 0/1/CPU0 shutdown
```

## Common Operational Tasks

### Save Configuration

```
! IOS XR auto-saves committed config — no "write memory" needed
! But you can copy for backup:
copy running-config tftp://10.0.0.5/backup/router-config.cfg

! Configuration replace from file
load tftp://10.0.0.5/router-config.cfg
commit replace
```

### Password Recovery

```
! Boot into ROMMON
! Set confreg to bypass config
confreg 0x142
reset

! After boot:
configure terminal
username admin secret <new-password>
commit
exit

! Reset boot register
config-register 0x102
admin reload location all
```

### Upgrade Procedure

```
! 1. Pre-checks
show install active summary
show platform
show redundancy

! 2. Copy new image
copy tftp://10.0.0.5/images/<new-image>.tar disk0:

! 3. Add and activate
install add source disk0: <new-image>.tar
install activate <package-name>

! 4. Verify
show install active summary
show version

! 5. Commit (makes persistent)
install commit
```

### Configuration Archive

```
! IOS XR maintains config commit history automatically
show configuration commit list

! Export specific commit
show configuration commit changes <commit-id>

! Rollback
rollback configuration to <commit-id>
commit
```

### Ping and Traceroute

```
! Basic
ping 10.0.0.2
traceroute 10.0.0.2

! VRF-aware
ping vrf CUSTOMER-A 172.16.0.2
traceroute vrf CUSTOMER-A 172.16.0.2

! Extended ping
ping 10.0.0.2 source 10.255.0.1 count 100 size 1500 df-bit
```

## Tips

- Always `commit` after configuration changes — uncommitted changes are lost on session exit
- Use `commit confirmed <seconds>` for risky changes to auto-rollback if you lose connectivity
- Use `show configuration` before committing to review pending changes
- `abort` discards all pending changes and exits configuration mode
- Interface names use full rack/slot/module/port format (e.g., `GigabitEthernet0/0/0/0`)
- Route policies (RPL) replace route-maps and prefix-lists from IOS
- Use `process restart <process>` instead of reload for non-disruptive recovery
- `install commit` is required after package activation to survive reloads
- Task-based authorization replaces IOS privilege levels — map users to task groups
- Configuration changes are atomic: a `commit` either fully succeeds or fully fails
- Use `show configuration commit list` and `rollback configuration` to undo mistakes
- gRPC and model-driven telemetry are first-class features on XR platforms

## See Also

- BGP
- OSPF
- IS-IS
- MPLS
- EEM
- NETCONF

## References

- Cisco IOS XR Configuration Fundamentals Guide — https://www.cisco.com/c/en/us/td/docs/iosxr/configuration-guide.html
- Cisco IOS XR System Management Configuration Guide — https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/system-management/configuration/guide.html
- Cisco IOS XR Routing Configuration Guide — https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/routing/configuration/guide.html
- Cisco IOS XR Install Operations — https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/install/configuration/guide.html
- Cisco IOS XR NETCONF/YANG Guide — https://www.cisco.com/c/en/us/td/docs/iosxr/programmability/netconf-yang.html
- RFC 6241 — Network Configuration Protocol (NETCONF)
- RFC 7950 — The YANG 1.1 Data Modeling Language
