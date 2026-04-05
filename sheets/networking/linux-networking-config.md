# Linux Networking Configuration

Network interface management: NetworkManager, ip-route2, bonding, teaming, bridges, VLANs, VRFs, namespaces, firewalld, nftables.

## NetworkManager

### nmcli Connection Management

```bash
# List all connections
nmcli connection show

# Show active connections
nmcli connection show --active

# Show connection details
nmcli connection show "Wired connection 1"

# Show device status
nmcli device status

# Show device details
nmcli device show eth0
```

### Create Connections

```bash
# Static IP
nmcli connection add con-name eth0-static \
  type ethernet ifname eth0 \
  ipv4.method manual \
  ipv4.addresses 192.168.1.10/24 \
  ipv4.gateway 192.168.1.1 \
  ipv4.dns "8.8.8.8 8.8.4.4" \
  ipv6.method disabled

# DHCP
nmcli connection add con-name eth0-dhcp \
  type ethernet ifname eth0 \
  ipv4.method auto

# Modify existing
nmcli connection modify eth0-static \
  ipv4.addresses 192.168.1.20/24 \
  +ipv4.dns 1.1.1.1

# Bring up/down
nmcli connection up eth0-static
nmcli connection down eth0-static

# Delete
nmcli connection delete eth0-static

# Reload from disk
nmcli connection reload
```

### Connection Profiles

```bash
# Profile directory
ls /etc/NetworkManager/system-connections/

# Key file format (RHEL 9+)
cat /etc/NetworkManager/system-connections/eth0-static.nmconnection
# [connection]
# id=eth0-static
# type=ethernet
# interface-name=eth0
# [ipv4]
# method=manual
# address1=192.168.1.10/24,192.168.1.1
# dns=8.8.8.8;8.8.4.4;
# [ipv6]
# method=disabled

# nmtui — text-based UI
nmtui
# Options: Edit, Activate, Set hostname
```

## ip-route2 Suite

### ip link (Interface Management)

```bash
# Show all interfaces
ip link show

# Show specific interface
ip link show dev eth0

# Bring up/down
ip link set eth0 up
ip link set eth0 down

# Set MTU
ip link set eth0 mtu 9000

# Set MAC address
ip link set eth0 address 00:11:22:33:44:55

# Show statistics
ip -s link show eth0
```

### ip addr (Address Management)

```bash
# Show all addresses
ip addr show

# Show specific interface
ip addr show dev eth0

# Add address
ip addr add 192.168.1.10/24 dev eth0

# Add secondary address
ip addr add 192.168.1.11/24 dev eth0 label eth0:1

# Delete address
ip addr del 192.168.1.10/24 dev eth0

# Flush all addresses on interface
ip addr flush dev eth0
```

### ip route (Routing)

```bash
# Show routing table
ip route show

# Show specific table
ip route show table main
ip route show table local

# Add default route
ip route add default via 192.168.1.1

# Add static route
ip route add 10.0.0.0/8 via 192.168.1.254

# Add route via interface
ip route add 172.16.0.0/16 dev eth1

# Add route with metric
ip route add 10.0.0.0/8 via 192.168.1.254 metric 100

# Delete route
ip route del 10.0.0.0/8 via 192.168.1.254

# Replace route (add or modify)
ip route replace 10.0.0.0/8 via 192.168.1.254

# Show route for destination
ip route get 10.1.2.3

# Cache flush
ip route flush cache
```

### ip rule (Policy Routing)

```bash
# Show rules
ip rule show

# Add rule: traffic from source uses table 100
ip rule add from 10.0.0.0/8 table 100

# Add rule: traffic to destination uses table 200
ip rule add to 172.16.0.0/12 table 200

# Add rule with priority
ip rule add from 192.168.1.0/24 table 100 priority 1000

# Add rule by fwmark
ip rule add fwmark 1 table 100

# Delete rule
ip rule del from 10.0.0.0/8 table 100

# Add routes to custom table
ip route add default via 10.0.0.1 table 100
ip route add 10.0.0.0/24 dev eth1 table 100
```

### ip neigh (ARP/NDP)

```bash
# Show ARP table
ip neigh show

# Add static ARP entry
ip neigh add 192.168.1.1 lladdr 00:11:22:33:44:55 dev eth0

# Delete entry
ip neigh del 192.168.1.1 dev eth0

# Flush
ip neigh flush dev eth0
```

## Bonding

### Create Bond with nmcli

```bash
# Create bond interface
nmcli connection add con-name bond0 type bond \
  ifname bond0 \
  bond.options "mode=802.3ad,miimon=100,lacp_rate=fast,xmit_hash_policy=layer3+4"

# Add slave interfaces
nmcli connection add con-name bond0-port1 type ethernet \
  ifname eth0 master bond0
nmcli connection add con-name bond0-port2 type ethernet \
  ifname eth1 master bond0

# Assign IP to bond
nmcli connection modify bond0 \
  ipv4.method manual \
  ipv4.addresses 192.168.1.10/24 \
  ipv4.gateway 192.168.1.1

# Bring up
nmcli connection up bond0
```

### Bonding Modes

```bash
# Mode 0: balance-rr   — Round-robin, requires switch config
# Mode 1: active-backup — One active, others standby (no switch config)
# Mode 2: balance-xor  — XOR hash of MAC addresses
# Mode 3: broadcast    — All slaves transmit same frame
# Mode 4: 802.3ad      — LACP (requires switch LACP support)
# Mode 5: balance-tlb  — Adaptive transmit load balancing
# Mode 6: balance-alb  — Adaptive load balancing (tx + rx)

# Mode 4 (LACP) hash policies:
#   layer2         — src/dst MAC
#   layer2+3       — src/dst MAC + src/dst IP
#   layer3+4       — src/dst IP + src/dst port
#   encap2+3       — inner src/dst MAC + IP (for tunnels)
#   encap3+4       — inner src/dst IP + port (for tunnels)
```

### Bond Status

```bash
# View bond info
cat /proc/net/bonding/bond0

# View via ip
ip link show bond0

# View slaves
cat /sys/class/net/bond0/bonding/slaves
```

## Teaming

### Create Team with nmcli

```bash
# Create team with runner config
nmcli connection add con-name team0 type team \
  ifname team0 \
  team.config '{"runner": {"name": "lacp", "active": true, "fast_rate": true, "tx_hash": ["eth", "ipv4", "ipv6"]}}'

# Add ports
nmcli connection add con-name team0-port1 type ethernet \
  ifname eth0 master team0
nmcli connection add con-name team0-port2 type ethernet \
  ifname eth1 master team0

# Assign IP
nmcli connection modify team0 \
  ipv4.method manual \
  ipv4.addresses 192.168.1.10/24 \
  ipv4.gateway 192.168.1.1

nmcli connection up team0
```

### Team Runners

```bash
# roundrobin    — simple round-robin (no switch config)
# activebackup  — one active port, failover
# lacp          — 802.3ad LACP (requires switch)
# loadbalance   — active Tx load balancing (hash-based)
# broadcast     — all ports transmit
# random        — random port selection

# Runner configs (JSON)
# Active-backup with link watch:
'{"runner": {"name": "activebackup"}, "link_watch": {"name": "ethtool"}}'

# LACP:
'{"runner": {"name": "lacp", "active": true, "fast_rate": true}}'

# Loadbalance with hash:
'{"runner": {"name": "loadbalance", "tx_hash": ["eth", "ipv4"]}}'
```

### teamdctl

```bash
# Show team state
teamdctl team0 state

# Show config
teamdctl team0 config dump

# Show specific port
teamdctl team0 port config dump eth0
```

## Bridge

### Create Bridge

```bash
# With nmcli
nmcli connection add con-name br0 type bridge ifname br0
nmcli connection add con-name br0-port1 type ethernet ifname eth0 master br0
nmcli connection add con-name br0-port2 type ethernet ifname eth1 master br0
nmcli connection modify br0 ipv4.method manual ipv4.addresses 192.168.1.10/24

# With ip
ip link add br0 type bridge
ip link set eth0 master br0
ip link set eth1 master br0
ip addr add 192.168.1.10/24 dev br0
ip link set br0 up
```

### Bridge Management

```bash
# Show bridge info
bridge link show
bridge fdb show
bridge vlan show

# STP
ip link set br0 type bridge stp_state 1
bridge link set dev eth0 cost 100
bridge link set dev eth0 priority 32

# Show STP state
cat /sys/class/net/br0/bridge/stp_state
```

## VLAN

### Create VLANs

```bash
# With nmcli
nmcli connection add con-name vlan100 type vlan \
  ifname eth0.100 dev eth0 id 100
nmcli connection modify vlan100 \
  ipv4.method manual \
  ipv4.addresses 10.100.0.10/24

# With ip
ip link add link eth0 name eth0.100 type vlan id 100
ip addr add 10.100.0.10/24 dev eth0.100
ip link set eth0.100 up

# Show VLAN info
cat /proc/net/vlan/config
ip -d link show eth0.100
```

## VRF (Virtual Routing and Forwarding)

### Create and Use VRFs

```bash
# Create VRF
ip link add vrf-red type vrf table 10
ip link set vrf-red up

# Assign interface to VRF
ip link set eth1 master vrf-red

# Add routes in VRF table
ip route add default via 10.0.0.1 table 10

# Execute command in VRF context
ip vrf exec vrf-red ping 10.0.0.1
ip vrf exec vrf-red ip route show

# Show VRFs
ip vrf show

# Show VRF interfaces
ip link show master vrf-red

# Per-VRF sockets
ip vrf exec vrf-red ss -tlnp
```

## Network Namespaces

### Namespace Management

```bash
# Create namespace
ip netns add ns1

# List namespaces
ip netns list

# Execute command in namespace
ip netns exec ns1 ip addr show
ip netns exec ns1 bash

# Create veth pair connecting namespaces
ip link add veth0 type veth peer name veth1
ip link set veth1 netns ns1

# Configure addresses
ip addr add 10.0.0.1/24 dev veth0
ip link set veth0 up
ip netns exec ns1 ip addr add 10.0.0.2/24 dev veth1
ip netns exec ns1 ip link set veth1 up
ip netns exec ns1 ip link set lo up

# Delete namespace
ip netns del ns1
```

## firewalld

### Zone Management

```bash
# List zones
firewall-cmd --get-zones

# Show default zone
firewall-cmd --get-default-zone

# Set default zone
firewall-cmd --set-default-zone=internal

# List active zones
firewall-cmd --get-active-zones

# Show zone details
firewall-cmd --zone=public --list-all

# Assign interface to zone
firewall-cmd --zone=internal --change-interface=eth0 --permanent
```

### Rules

```bash
# Add service
firewall-cmd --zone=public --add-service=http --permanent
firewall-cmd --zone=public --add-service=https --permanent

# Add port
firewall-cmd --zone=public --add-port=8080/tcp --permanent

# Rich rules
firewall-cmd --zone=public --add-rich-rule='rule family="ipv4" source address="10.0.0.0/8" service name="ssh" accept' --permanent

# Remove service
firewall-cmd --zone=public --remove-service=http --permanent

# Reload
firewall-cmd --reload

# Port forwarding
firewall-cmd --zone=public --add-forward-port=port=80:proto=tcp:toport=8080 --permanent

# Masquerade (NAT)
firewall-cmd --zone=public --add-masquerade --permanent
```

## nftables

### Basic Configuration

```bash
# Show current ruleset
nft list ruleset

# Create table
nft add table inet filter

# Create chain
nft add chain inet filter input { type filter hook input priority 0 \; policy accept \; }
nft add chain inet filter forward { type filter hook forward priority 0 \; policy drop \; }
nft add chain inet filter output { type filter hook output priority 0 \; policy accept \; }

# Add rules
nft add rule inet filter input ct state established,related accept
nft add rule inet filter input iif lo accept
nft add rule inet filter input tcp dport 22 accept
nft add rule inet filter input tcp dport { 80, 443 } accept
nft add rule inet filter input ip saddr 10.0.0.0/8 tcp dport 8080 accept
nft add rule inet filter input counter drop

# Delete rule by handle
nft -a list chain inet filter input   # show handles
nft delete rule inet filter input handle 5

# Save/restore
nft list ruleset > /etc/nftables.conf
nft -f /etc/nftables.conf

# Flush
nft flush ruleset
```

## Sysctl Networking Tunables

### Common Tunables

```bash
# Enable IP forwarding
sysctl -w net.ipv4.ip_forward=1

# Persistent (add to /etc/sysctl.d/99-network.conf):
net.ipv4.ip_forward = 1
net.ipv6.conf.all.forwarding = 1

# TCP tuning
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216
net.core.netdev_max_backlog = 5000

# Connection tracking
net.netfilter.nf_conntrack_max = 1048576

# ARP
net.ipv4.conf.all.arp_announce = 2
net.ipv4.conf.all.arp_ignore = 1

# ICMP
net.ipv4.icmp_echo_ignore_broadcasts = 1

# Reverse path filtering
net.ipv4.conf.all.rp_filter = 1

# Apply
sysctl -p /etc/sysctl.d/99-network.conf
```

## DNS and Name Resolution

### Configuration Files

```bash
# /etc/hosts (static mappings, checked first by default)
127.0.0.1   localhost
192.168.1.10 server1.example.com server1

# /etc/resolv.conf
nameserver 8.8.8.8
nameserver 8.8.4.4
search example.com internal.example.com
options timeout:2 attempts:3

# /etc/nsswitch.conf (resolution order)
hosts: files dns myhostname

# Check resolution
getent hosts server1.example.com
```

## systemd-networkd

### Network Configuration

```bash
# /etc/systemd/network/10-eth0.network
[Match]
Name=eth0

[Network]
Address=192.168.1.10/24
Gateway=192.168.1.1
DNS=8.8.8.8
DNS=8.8.4.4

[Route]
Destination=10.0.0.0/8
Gateway=192.168.1.254

# Enable
systemctl enable --now systemd-networkd
systemctl enable --now systemd-resolved

# Link creation
# /etc/systemd/network/20-br0.netdev
[NetDev]
Name=br0
Kind=bridge

# /etc/systemd/network/21-br0.network
[Match]
Name=br0

[Network]
Address=192.168.1.10/24
Gateway=192.168.1.1
```

## Persistent Routing

### Methods

```bash
# nmcli (preferred on RHEL/Fedora)
nmcli connection modify eth0 +ipv4.routes "10.0.0.0/8 192.168.1.254 100"
nmcli connection up eth0

# Route file (legacy, RHEL 8)
# /etc/sysconfig/network-scripts/route-eth0
# 10.0.0.0/8 via 192.168.1.254
# 172.16.0.0/12 via 192.168.1.253

# systemd-networkd
# [Route] section in .network file

# ip route with save/restore (non-persistent, for testing)
ip route save > /tmp/routes.bin
ip route restore < /tmp/routes.bin
```

## See Also

- arp
- bgp
- firewall-zones
- nftables
- network-namespaces
- vxlan

## References

- man nmcli, ip, ip-link, ip-route, ip-rule, ip-netns
- man firewall-cmd, nft
- kernel.org: Documentation/networking/bonding.rst
- kernel.org: Documentation/networking/vrf.rst
- libteam.fedorahosted.org
