# ip (iproute2)

Swiss army knife for Linux networking — manages addresses, links, routes, neighbors, namespaces, and more.

## Addresses (ip addr)

### Show addresses
```bash
ip addr                         # all interfaces with addresses
ip addr show dev eth0           # specific interface
ip -4 addr                      # IPv4 only
ip -6 addr                      # IPv6 only
ip addr show scope global       # only globally routable addresses
ip -br addr                     # brief format — easy to read
```

### Add / remove addresses
```bash
ip addr add 10.0.0.5/24 dev eth0
ip addr add 10.0.0.6/24 dev eth0 label eth0:1   # virtual interface label
ip addr del 10.0.0.5/24 dev eth0
ip addr flush dev eth0          # remove all addresses
```

## Links (ip link)

### Show link status
```bash
ip link                         # all interfaces
ip -br link                     # brief: name, state, MAC
ip link show dev eth0           # specific interface
ip -s link                      # with packet/byte stats
ip -s -s link show dev eth0     # detailed error stats
```

### Bring interfaces up/down
```bash
ip link set eth0 up
ip link set eth0 down
```

### Change link properties
```bash
ip link set eth0 mtu 9000                    # jumbo frames
ip link set eth0 address 00:11:22:33:44:55   # change MAC
ip link set eth0 promisc on                  # promiscuous mode
ip link set eth0 txqueuelen 10000            # transmit queue length
```

### Create virtual interfaces
```bash
ip link add veth0 type veth peer name veth1          # veth pair
ip link add br0 type bridge                          # bridge
ip link set eth0 master br0                          # add to bridge
ip link add bond0 type bond mode 802.3ad             # LACP bond
ip link add vlan100 link eth0 type vlan id 100       # VLAN
ip link add macvlan0 link eth0 type macvlan mode bridge  # macvlan
ip link delete veth0                                 # remove interface
```

## Routes (ip route)

### Show routes
```bash
ip route                        # main routing table
ip route show table all         # all routing tables
ip route get 8.8.8.8            # which route handles this destination
ip -6 route                     # IPv6 routes
```

### Add / remove routes
```bash
ip route add 10.10.0.0/16 via 10.0.0.1
ip route add 10.10.0.0/16 via 10.0.0.1 dev eth0
ip route add default via 10.0.0.1
ip route del 10.10.0.0/16
ip route replace 10.10.0.0/16 via 10.0.0.2   # add or update
```

### Advanced routing
```bash
ip route add 10.10.0.0/16 via 10.0.0.1 metric 100
ip route add blackhole 192.168.99.0/24               # silently drop
ip route add unreachable 192.168.99.0/24              # ICMP unreachable
ip route add prohibit 192.168.99.0/24                 # ICMP prohibited
```

### Cache and flush
```bash
ip route flush cache             # flush route cache
ip route flush table main        # flush all routes in main table
```

## Neighbors (ip neigh / ARP)

### Show ARP/NDP table
```bash
ip neigh                         # all neighbors
ip neigh show dev eth0           # specific interface
ip -s neigh                      # with stats
```

### Manage entries
```bash
ip neigh add 10.0.0.1 lladdr 00:11:22:33:44:55 dev eth0
ip neigh del 10.0.0.1 dev eth0
ip neigh flush dev eth0          # clear ARP cache for interface
ip neigh replace 10.0.0.1 lladdr 00:11:22:33:44:55 dev eth0
```

## Network Namespaces (ip netns)

### Manage namespaces
```bash
ip netns list
ip netns add myns
ip netns del myns
ip netns exec myns ip addr       # run command inside namespace
ip netns exec myns bash          # shell inside namespace
ip link set eth1 netns myns      # move interface into namespace
```

### Connect namespaces with veth
```bash
ip link add veth0 type veth peer name veth1
ip link set veth1 netns myns
ip addr add 10.200.0.1/24 dev veth0
ip netns exec myns ip addr add 10.200.0.2/24 dev veth1
ip link set veth0 up
ip netns exec myns ip link set veth1 up
```

## Policy Routing (ip rule)

### Show and manage rules
```bash
ip rule list
ip rule add from 192.168.1.0/24 table 100
ip rule add fwmark 1 table 100
ip rule del from 192.168.1.0/24 table 100
```

## Tunnel Interfaces

### GRE tunnel
```bash
ip tunnel add gre1 mode gre remote 203.0.113.1 local 198.51.100.1 ttl 255
ip addr add 10.99.0.1/30 dev gre1
ip link set gre1 up
```

### IPIP tunnel
```bash
ip tunnel add ipip1 mode ipip remote 203.0.113.1 local 198.51.100.1
```

### WireGuard (ip link)
```bash
ip link add wg0 type wireguard
```

## Monitor (ip monitor)

### Watch for changes in real time
```bash
ip monitor                       # all events
ip monitor route                 # route changes only
ip monitor neigh                 # ARP/NDP changes
ip monitor link                  # link state changes
ip monitor address               # address changes
```

## Tips

- `ip -br` (brief) is your friend for quick status checks
- `ip route get <dst>` tells you exactly which route and interface will be used
- Changes made with `ip` are not persistent across reboots — use `netplan`, `NetworkManager`, or `/etc/network/interfaces` for persistence
- `ip -j` outputs JSON — pipe to `jq` for scripting
- `ip -c` enables color output for readability
- `ip link set dev eth0 netns <ns>` moves the interface — it disappears from the host
- `ip route replace` is idempotent and safer than `add` in scripts
- `ip neigh flush` can disrupt active connections briefly while ARP re-resolves
