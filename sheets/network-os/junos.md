# JunOS (Juniper Networks Operating System)

CLI for configuring and managing Juniper routers, switches, and firewalls -- commit-based configuration, routing, security policies, and firewall filters.

## CLI Modes

### Mode Navigation

```
Operational:      user@router>           # show, ping, traceroute, request
Configuration:    user@router#           # set, delete, show, commit
Shell:            user@router%           # FreeBSD shell (rarely needed)

user@router> configure                   # operational -> configuration
user@router> configure exclusive         # lock config for exclusive editing
user@router# run show route             # run operational cmd from config mode
user@router# edit interfaces ge-0/0/0   # enter a config hierarchy level
user@router# top                        # jump to top of hierarchy
user@router# up                         # go up one level
user@router# exit                       # leave config mode
```

## Show Commands (Operational Mode)

### Interfaces, Chassis, Routing

```
show interfaces terse                    # brief status (like IOS "show ip int brief")
show interfaces ge-0/0/0 extensive       # detailed stats, errors, counters
show chassis hardware                    # installed hardware, serial numbers
show chassis alarms                      # active alarms
show chassis routing-engine              # CPU, memory, uptime

show route                               # full routing table
show route 10.0.1.0/24                   # routes matching prefix
show route protocol ospf                 # only OSPF routes
show route summary                       # route count per protocol
show ospf neighbor                       # OSPF adjacencies
show ospf database                       # OSPF LSDB
show bgp summary                         # BGP peer status
show configuration                       # full active configuration
show configuration | display set         # show as set commands (easy to copy)
show log messages | match error          # filter syslog output
```

## Configuration

### Basic Syntax

```
set system host-name R1
set interfaces ge-0/0/0 unit 0 family inet address 10.0.1.1/24
delete interfaces ge-0/0/0 unit 0 family inet address 10.0.1.1/24
deactivate interfaces ge-0/0/0           # disable without removing
activate interfaces ge-0/0/0             # re-enable
wildcard delete interfaces ge-0/0/[0-3]  # wildcard delete

# apply-groups: reusable config templates
set groups STANDARD system syslog host 10.0.1.50
set apply-groups STANDARD

show                                     # candidate config
show | compare                           # diff candidate vs active
```

## Commit Model

```
commit                                   # apply config, make it active
commit check                             # validate only, don't apply
commit confirmed 5                       # auto-rollback in 5 min unless confirmed
commit                                   # confirm within timeout to keep changes
commit comment "added OSPF"              # commit with log message
commit and-quit                          # commit and exit config mode

rollback 0                               # discard candidate changes
rollback 1                               # revert to previous commit
show | compare rollback 1                # diff current vs rollback 1
commit                                   # must commit after rollback
```

## Interfaces

```
set interfaces ge-0/0/0 description "Uplink to Core"
set interfaces ge-0/0/0 unit 0 family inet address 10.0.1.1/24
set interfaces lo0 unit 0 family inet address 1.1.1.1/32
set interfaces ge-0/0/0 disable          # disable interface
delete interfaces ge-0/0/0 disable       # enable interface

# VLAN tagging (trunk)
set interfaces ge-0/0/1 vlan-tagging
set interfaces ge-0/0/1 unit 10 vlan-id 10
set interfaces ge-0/0/1 unit 10 family inet address 10.10.10.1/24

# Aggregated Ethernet (LAG)
set chassis aggregated-devices ethernet device-count 2
set interfaces ge-0/0/0 ether-options 802.3ad ae0
set interfaces ge-0/0/1 ether-options 802.3ad ae0
set interfaces ae0 aggregated-ether-options lacp active
set interfaces ae0 unit 0 family inet address 10.0.1.1/24
```

## Routing

### Static, OSPF, BGP

```
set routing-options static route 192.168.2.0/24 next-hop 10.0.0.2
set routing-options static route 0.0.0.0/0 next-hop 10.0.0.1
set routing-options router-id 1.1.1.1

set protocols ospf area 0.0.0.0 interface ge-0/0/0.0
set protocols ospf area 0.0.0.0 interface lo0.0 passive
set protocols ospf export DEFAULT-ROUTE

set routing-options autonomous-system 65001
set protocols bgp group EBGP type external
set protocols bgp group EBGP neighbor 203.0.113.1 peer-as 65002
set protocols bgp group EBGP export ADVERTISE-ROUTES
```

### Routing Policy and Instances

```
set policy-options prefix-list MY-NETS 198.51.100.0/24
set policy-options policy-statement ADVERTISE term 1 from prefix-list MY-NETS
set policy-options policy-statement ADVERTISE term 1 then accept
set policy-options policy-statement ADVERTISE term DEFAULT then reject

set routing-instances CUSTOMER-A instance-type virtual-router
set routing-instances CUSTOMER-A interface ge-0/0/2.0
```

## Firewall Filters

```
set firewall family inet filter PROTECT-RE term ALLOW-SSH from protocol tcp
set firewall family inet filter PROTECT-RE term ALLOW-SSH from destination-port 22
set firewall family inet filter PROTECT-RE term ALLOW-SSH then accept
set firewall family inet filter PROTECT-RE term ALLOW-ICMP from protocol icmp
set firewall family inet filter PROTECT-RE term ALLOW-ICMP then accept
set firewall family inet filter PROTECT-RE term ALLOW-OSPF from protocol ospf
set firewall family inet filter PROTECT-RE term ALLOW-OSPF then accept
set firewall family inet filter PROTECT-RE term DENY-ALL then discard
# also: then reject, then log, then count COUNTER-NAME

# Apply to interface or loopback (protect routing engine)
set interfaces lo0 unit 0 family inet filter input PROTECT-RE
```

## NAT

```
# Source NAT (PAT/overload equivalent)
set security nat source rule-set SNAT from zone trust
set security nat source rule-set SNAT to zone untrust
set security nat source rule-set SNAT rule PAT match source-address 192.168.1.0/24
set security nat source rule-set SNAT rule PAT then source-nat interface

# Destination NAT (port forwarding)
set security nat destination pool WEB address 192.168.1.100/32 port 80
set security nat destination rule-set DNAT from zone untrust
set security nat destination rule-set DNAT rule WEB match destination-port 80
set security nat destination rule-set DNAT rule WEB then destination-nat pool WEB

# Static NAT
set security nat static rule-set STATIC from zone untrust
set security nat static rule-set STATIC rule MAP1 match destination-address 203.0.113.10/32
set security nat static rule-set STATIC rule MAP1 then static-nat prefix 192.168.1.10/32
```

## Security Zones and Policies (SRX)

```
set security zones security-zone trust interfaces ge-0/0/0.0
set security zones security-zone untrust interfaces ge-0/0/1.0
set security zones security-zone trust host-inbound-traffic system-services ssh

set security policies from-zone trust to-zone untrust policy ALLOW-WEB match source-address any
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match destination-address any
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match application junos-http
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then permit
```

## System Configuration

```
set system host-name R1
set system domain-name example.com
set system name-server 8.8.8.8
set system ntp server 216.239.35.0
set system syslog host 10.0.1.50 any info
set system syslog file messages any notice
set system login user admin class super-user
set system services ssh protocol-version v2
delete system services telnet
```

## Operational Commands

```
request system reboot                    # reboot device
request system snapshot                  # backup flash
file list /var/log/                      # list files
file show /var/log/messages              # display file
monitor interface ge-0/0/0              # real-time stats (ESC+q to exit)
monitor traffic interface ge-0/0/0      # packet capture (tcpdump)
ping 10.0.1.1 count 5 rapid
traceroute 10.0.1.1
clear bgp neighbor 203.0.113.1          # reset BGP session
```

## IOS to JunOS Quick Reference

```
# IOS                           ->  JunOS
# show running-config           ->  show configuration
# show ip interface brief       ->  show interfaces terse
# show ip route                 ->  show route
# show ip ospf neighbor         ->  show ospf neighbor
# configure terminal            ->  configure
# write memory                  ->  commit (auto-saved)
# no shutdown                   ->  delete ... disable
# access-list                   ->  firewall filter
# implicit deny at ACL end      ->  NO implicit deny (add explicit term)
# changes apply immediately     ->  changes staged until "commit"
```

## Tips

- JunOS uses a candidate/active config model -- nothing takes effect until `commit`.
- Use `commit confirmed` in production -- if you lock yourself out, config auto-rolls back.
- `show | compare` before every commit to review exactly what will change.
- `rollback 1` followed by `commit` undoes the last commit -- keep this in muscle memory.
- Unlike IOS, JunOS has no implicit deny at the end of firewall filters -- always add an explicit final term.
- Use `show | display set` to see config in set-command format (easy to copy/paste).
- Operational commands work from config mode with the `run` prefix.
- Juniper uses `ge-` (1G), `xe-` (10G), `et-` (40G/100G) interface naming.
- Use `request system zeroize` to factory-reset -- this erases everything.

## References

- [Juniper Junos OS Documentation](https://www.juniper.net/documentation/)
- [Junos OS CLI User Guide](https://www.juniper.net/documentation/us/en/software/junos/cli/topics/topic-map/junos-cli-overview.html)
- [Junos OS Routing Protocols Library](https://www.juniper.net/documentation/us/en/software/junos/is-is/topics/topic-map/is-is-overview.html)
- [Junos OS Interfaces Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/interfaces-fundamentals/topics/topic-map/router-interfaces-overview.html)
- [Junos OS Routing Policy and Firewall Filters](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/topic-map/policy-overview.html)
- [Junos OS System Management Guide](https://www.juniper.net/documentation/us/en/software/junos/system-basics/topics/topic-map/system-management-overview.html)
- [Juniper TechLibrary](https://www.juniper.net/documentation/)
- [Juniper Learning Portal](https://learningportal.juniper.net/)
