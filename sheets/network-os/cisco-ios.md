# Cisco IOS (Internetwork Operating System)

CLI for configuring and managing Cisco routers and switches -- routing, switching, ACLs, NAT, and device administration.

## CLI Modes

### Mode Navigation

```
User EXEC:          Router>              # limited show commands
Privileged EXEC:    Router#              # full show, debug, copy, reload
Global Config:      Router(config)#      # system-wide settings
Interface Config:   Router(config-if)#   # per-interface settings
Line Config:        Router(config-line)# # console/vty line settings
Router Config:      Router(config-router)# # routing protocol config

Router> enable                           # user -> privileged
Router# configure terminal              # privileged -> global config
Router(config)# interface g0/1          # global -> interface config
Router(config)# line vty 0 4           # global -> line config
Router(config)# router ospf 1          # global -> router config
Router(config-if)# exit                 # back one level
Router(config-if)# end                  # back to privileged (from anywhere)
```

## Show Commands

### Device and Interfaces

```
Router# show version                     # IOS version, uptime, hardware
Router# show running-config              # current active configuration
Router# show startup-config              # saved configuration (NVRAM)
Router# show running-config | section interface  # filter output
Router# show running-config | include hostname   # grep-like filter
Router# show logging                     # syslog buffer
Router# show interfaces                  # all interfaces, detailed stats
Router# show ip interface brief          # summary: IP, status, protocol
Router# show interfaces status           # switch: speed, duplex, vlan
Router# show interfaces trunk            # trunk port details
```

### Routing, Switching, Neighbors

```
Router# show ip route                    # full routing table
Router# show ip route ospf               # only OSPF-learned routes
Router# show ip protocols                # active routing protocols
Router# show arp                         # ARP table (IP to MAC)
Switch# show vlan brief                  # VLAN IDs, names, assigned ports
Switch# show mac address-table           # MAC address to port mappings
Switch# show spanning-tree               # STP topology and port states
Switch# show cdp neighbors               # directly connected Cisco devices
Switch# show cdp neighbors detail        # includes IP addresses, platform
```

## Interface Configuration

### Basic Setup

```
Router(config)# interface GigabitEthernet0/1
Router(config-if)# description Uplink to Core
Router(config-if)# ip address 10.0.1.1 255.255.255.0
Router(config-if)# no shutdown                     # enable the interface
Router(config-if)# shutdown                        # disable the interface
Router(config-if)# speed 1000                      # 10/100/1000/auto
Router(config-if)# duplex full                     # half/full/auto
```

### Switch Port Modes

```
Switch(config-if)# switchport mode access           # single VLAN port
Switch(config-if)# switchport access vlan 10
Switch(config-if)# switchport mode trunk            # carries multiple VLANs
Switch(config-if)# switchport trunk allowed vlan 10,20,30
Switch(config-if)# switchport trunk native vlan 99
Switch(config-if)# switchport port-security
Switch(config-if)# switchport port-security maximum 2
Switch(config-if)# switchport port-security violation shutdown
```

### Port-Channel (EtherChannel)

```
Switch(config)# interface range g0/1 - 2
Switch(config-if-range)# channel-group 1 mode active    # LACP
Switch(config)# interface port-channel 1
Switch(config-if)# switchport mode trunk
```

## Routing

### Static Routes

```
Router(config)# ip route 192.168.2.0 255.255.255.0 10.0.0.2       # next-hop IP
Router(config)# ip route 192.168.2.0 255.255.255.0 g0/1           # exit interface
Router(config)# ip route 0.0.0.0 0.0.0.0 10.0.0.1                 # default route
```

### OSPF

```
Router(config)# router ospf 1
Router(config-router)# router-id 1.1.1.1
Router(config-router)# network 10.0.1.0 0.0.0.255 area 0          # wildcard mask
Router(config-router)# passive-interface g0/2                      # don't send hellos
Router(config-router)# default-information originate               # advertise default

Router# show ip ospf neighbor             # adjacency table
Router# show ip ospf interface brief      # OSPF-enabled interfaces
Router# show ip ospf database             # link-state database
```

### EIGRP

```
Router(config)# router eigrp 100
Router(config-router)# network 10.0.0.0 0.0.255.255
Router(config-router)# no auto-summary
Router(config-router)# passive-interface default
Router(config-router)# no passive-interface g0/0
```

### BGP Basics

```
Router(config)# router bgp 65001
Router(config-router)# neighbor 203.0.113.1 remote-as 65002
Router(config-router)# network 198.51.100.0 mask 255.255.255.0

Router# show ip bgp summary               # peer status
Router# show ip bgp                        # BGP table
```

## VLANs

```
Switch(config)# vlan 10
Switch(config-vlan)# name Engineering

# Inter-VLAN routing (router-on-a-stick)
Router(config)# interface g0/1.10
Router(config-subif)# encapsulation dot1Q 10
Router(config-subif)# ip address 10.10.10.1 255.255.255.0

# VTP
Switch(config)# vtp mode server           # server | client | transparent
Switch(config)# vtp domain CORP
```

## Access Control Lists

### Standard ACL (source only, 1-99)

```
Router(config)# access-list 10 permit 192.168.1.0 0.0.0.255
Router(config)# access-list 10 deny any               # implicit deny at end anyway
Router(config-if)# ip access-group 10 in               # apply inbound
```

### Extended ACL (source + dest + protocol, 100-199)

```
Router(config)# access-list 100 permit tcp 192.168.1.0 0.0.0.255 any eq 80
Router(config)# access-list 100 permit tcp 192.168.1.0 0.0.0.255 any eq 443
Router(config)# access-list 100 deny ip any any log
Router(config-if)# ip access-group 100 out
```

### Named ACL

```
Router(config)# ip access-list extended WEB-TRAFFIC
Router(config-ext-nacl)# permit tcp any host 10.0.1.100 eq 80
Router(config-ext-nacl)# permit tcp any host 10.0.1.100 eq 443
Router(config-ext-nacl)# deny ip any any log
```

## NAT

### Static, Dynamic, PAT

```
# Static NAT
Router(config)# ip nat inside source static 192.168.1.10 203.0.113.10
Router(config-if)# ip nat inside              # on inside interface
Router(config-if)# ip nat outside             # on outside interface

# Dynamic NAT
Router(config)# ip nat pool MYPOOL 203.0.113.20 203.0.113.30 netmask 255.255.255.0
Router(config)# access-list 1 permit 192.168.1.0 0.0.0.255
Router(config)# ip nat inside source list 1 pool MYPOOL

# PAT (overload) -- most common
Router(config)# ip nat inside source list 1 interface g0/1 overload

Router# show ip nat translations            # active NAT table
Router# clear ip nat translation *           # flush NAT table
```

## Services

### DHCP, NTP, Logging

```
Router(config)# ip dhcp pool LAN
Router(dhcp-config)# network 192.168.1.0 255.255.255.0
Router(dhcp-config)# default-router 192.168.1.1
Router(dhcp-config)# dns-server 8.8.8.8 8.8.4.4
Router(dhcp-config)# lease 7
Router(config)# ip dhcp excluded-address 192.168.1.1 192.168.1.50

Router(config)# ntp server 216.239.35.0
Router(config)# logging buffered 16384 informational
Router(config)# logging host 10.0.1.50
Router(config)# service timestamps log datetime msec
```

## Device Security

### Passwords and SSH

```
Router(config)# hostname R1
Router(config)# enable secret MyEnablePass
Router(config)# service password-encryption
Router(config)# banner motd # Authorized Access Only #

# SSH setup
Router(config)# ip domain-name example.com
Router(config)# crypto key generate rsa modulus 2048
Router(config)# ip ssh version 2
Router(config)# username admin privilege 15 secret AdminPass

# Console
Router(config)# line console 0
Router(config-line)# login local
Router(config-line)# logging synchronous
Router(config-line)# exec-timeout 10 0

# VTY (remote access)
Router(config)# line vty 0 4
Router(config-line)# transport input ssh           # SSH only, no telnet
Router(config-line)# login local
Router(config-line)# exec-timeout 5 0
```

## Save and Manage Configuration

```
Router# write memory                        # save running to startup (wr)
Router# copy running-config startup-config  # same as above (explicit)
Router# copy running-config tftp:           # backup to TFTP server
Router# erase startup-config                # factory reset on next reload
Router# reload                              # reboot the device
```

## Troubleshooting

```
Router# ping 10.0.1.1                       # ICMP reachability
Router# ping 10.0.1.1 source 192.168.1.1    # specify source IP
Router# traceroute 10.0.1.1                  # trace path hop by hop
Router# show interfaces g0/1 | include errors  # check error counters
Router# debug ip ospf events                 # real-time debug (use carefully)
Router# undebug all                          # stop all debugging
Router# show tech-support                    # full diagnostic for TAC
Router# terminal monitor                     # see debug/log on VTY session
```

## Tips

- Always `write memory` after changes -- running-config is lost on reboot.
- Use `do show` from config mode to run privileged commands without exiting.
- Tab completion works everywhere -- use it to avoid typos.
- The `?` key shows available commands or arguments at any point.
- Wildcard masks are the inverse of subnet masks: 255.255.255.0 becomes 0.0.0.255.
- Apply standard ACLs close to the destination, extended ACLs close to the source.
- Use `passive-interface` on LAN-facing ports to suppress routing hellos.
- `show running-config | section` is better than `include` for multi-line blocks.
- Always set `ip ssh version 2` and disable telnet on VTY lines in production.
- Use `debug` sparingly on production -- it consumes CPU and can crash busy routers.

## See Also

- junos
- bind
- dnsmasq
- ssh-tunneling
- haproxy

## References

- [Cisco IOS Command Reference](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/fundamentals/command/cf_command_ref.html)
- [Cisco IOS Configuration Fundamentals](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/fundamentals/configuration/xe-16/fundamentals-xe-16-book.html)
- [Cisco IOS Interface Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/interface/configuration/xe-16/ir-xe-16-book.html)
- [Cisco IOS IP Routing Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/iproute_pi/configuration/xe-16/iri-xe-16-book.html)
- [Cisco IOS Security Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/security/config_library/xe-16/sec-xe-16-library.html)
- [Cisco IOS ACL Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_data_acl/configuration/xe-16/sec-data-acl-xe-16-book.html)
- [Cisco IOS Release Notes](https://www.cisco.com/c/en/us/support/ios-nx-os-software/ios-xe-17/products-release-notes-list.html)
- [Cisco Learning Network](https://learningnetwork.cisco.com/)
