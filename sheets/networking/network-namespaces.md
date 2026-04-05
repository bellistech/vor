# Network Namespaces

Linux kernel feature (CLONE_NEWNET) that provides complete network stack isolation -- separate interfaces, routing tables, iptables rules, and sockets per namespace.

## Core Concepts

```
Each network namespace gets its own:
  - Network interfaces (lo, eth0, veth*, etc.)
  - Routing tables (ip route)
  - Firewall rules (iptables/nftables)
  - Sockets and port bindings
  - /proc/net and /sys/class/net views
  - ARP/NDP neighbor tables

Default namespace = "init_net" (PID 1's namespace)
Named namespaces are bind-mounted to /run/netns/<name>
```

## ip netns Commands

```bash
# Create a named network namespace
ip netns add red

# List all named namespaces
ip netns list

# Delete a namespace
ip netns del red

# Execute a command inside a namespace
ip netns exec red ip link show
ip netns exec red bash    # interactive shell in namespace

# Identify the namespace of a process
ip netns identify <pid>

# List PIDs in a namespace
ip netns pids red

# Monitor namespace creation/deletion events
ip netns monitor

# Set a namespace's NETNS ID (for cross-namespace references)
ip netns set red 100
```

## Veth Pairs (Virtual Ethernet)

```bash
# Create a veth pair -- two endpoints connected like a pipe
ip link add veth-red type veth peer name veth-blue

# Move each end into a different namespace
ip link set veth-red netns red
ip link set veth-blue netns blue

# Configure addresses and bring up
ip netns exec red ip addr add 10.0.0.1/24 dev veth-red
ip netns exec red ip link set veth-red up
ip netns exec red ip link set lo up

ip netns exec blue ip addr add 10.0.0.2/24 dev veth-blue
ip netns exec blue ip link set veth-blue up
ip netns exec blue ip link set lo up

# Test connectivity
ip netns exec red ping 10.0.0.2
```

## Bridge for Multi-Namespace Connectivity

```bash
# Create a bridge in the default namespace
ip link add br0 type bridge
ip link set br0 up
ip addr add 10.0.0.254/24 dev br0

# For each namespace, create a veth pair with one end on the bridge
ip link add veth-red type veth peer name veth-red-br
ip link set veth-red netns red
ip link set veth-red-br master br0
ip link set veth-red-br up
ip netns exec red ip addr add 10.0.0.1/24 dev veth-red
ip netns exec red ip link set veth-red up

ip link add veth-blue type veth peer name veth-blue-br
ip link set veth-blue netns blue
ip link set veth-blue-br master br0
ip link set veth-blue-br up
ip netns exec blue ip addr add 10.0.0.2/24 dev veth-blue
ip netns exec blue ip link set veth-blue up

# All namespaces can now reach each other via the bridge
ip netns exec red ping 10.0.0.2
ip netns exec blue ping 10.0.0.1
```

## Macvlan and Ipvlan

```bash
# macvlan -- each namespace gets its own MAC on the physical NIC
ip link add macvlan0 link eth0 type macvlan mode bridge
ip link set macvlan0 netns red
ip netns exec red ip addr add 192.168.1.50/24 dev macvlan0
ip netns exec red ip link set macvlan0 up

# macvlan modes: private, vepa, bridge, passthru, source

# ipvlan -- all endpoints share the parent MAC (L3 routing)
ip link add ipvlan0 link eth0 type ipvlan mode l3
ip link set ipvlan0 netns blue
ip netns exec blue ip addr add 192.168.1.51/24 dev ipvlan0
ip netns exec blue ip link set ipvlan0 up

# ipvlan modes: l2 (switching), l3 (routing), l3s (routing + iptables)
```

## Namespace Persistence and Bind Mounts

```bash
# Named namespaces are automatically bind-mounted:
#   /run/netns/<name> -> /proc/self/ns/net
# This keeps the namespace alive even with no processes inside

# Manually bind-mount an unnamed namespace for persistence
touch /run/netns/custom
mount --bind /proc/<pid>/ns/net /run/netns/custom

# Verify the namespace file
ls -la /run/netns/
stat /proc/self/ns/net    # inode identifies the namespace
readlink /proc/<pid>/ns/net

# Named namespaces also get a mount namespace for /etc/resolv.conf:
#   /etc/netns/<name>/resolv.conf -> mounted as /etc/resolv.conf inside
mkdir -p /etc/netns/red
echo "nameserver 8.8.8.8" > /etc/netns/red/resolv.conf
ip netns exec red cat /etc/resolv.conf   # sees 8.8.8.8
```

## unshare and nsenter

```bash
# Create a new network namespace with unshare
unshare --net bash
# Now in a new namespace with only the loopback interface
ip link show   # only lo

# Combine with other namespaces
unshare --net --mount --pid --fork bash

# Enter an existing namespace by PID
nsenter --net=/proc/<pid>/ns/net bash

# Enter a named namespace (equivalent to ip netns exec)
nsenter --net=/run/netns/red bash

# Enter multiple namespace types at once
nsenter --net --mount --target <pid> bash

# The /proc/[pid]/ns/net file descriptor approach
ls -la /proc/$$/ns/net      # current process network namespace
readlink /proc/1/ns/net     # init's namespace (default)
```

## Per-Namespace Iptables

```bash
# Each namespace has its own iptables/nftables ruleset
ip netns exec red iptables -L -n
ip netns exec red iptables -A INPUT -p icmp -j DROP

# NAT for namespace internet access
# In the default namespace:
sysctl -w net.ipv4.ip_forward=1
iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE
iptables -A FORWARD -i br0 -o eth0 -j ACCEPT
iptables -A FORWARD -i eth0 -o br0 -m state --state RELATED,ESTABLISHED -j ACCEPT

# In the namespace, set default route
ip netns exec red ip route add default via 10.0.0.254
```

## Container Networking Model

```bash
# Docker, LXD, Podman, and Kubernetes all use network namespaces

# Find a Docker container's namespace
PID=$(docker inspect --format '{{.State.Pid}}' <container>)
nsenter --net=/proc/$PID/ns/net ip addr show

# Docker creates veth pairs: one in container ns, one on docker0 bridge
docker network inspect bridge

# LXD container namespace
lxc info <container> | grep Pid
nsenter --net=/proc/<pid>/ns/net ip route

# Kubernetes pod networking: each pod gets its own network namespace
# CNI plugins (Calico, Flannel, Cilium) configure the veth pairs
crictl inspect <container-id> | jq '.info.pid'
```

## Multi-Namespace Lab (Simulated Topology)

```bash
# Router-on-a-stick: two namespaces connected through a router namespace
# Topology: [ns-a] -- [router] -- [ns-b]

# Create namespaces
ip netns add ns-a
ip netns add ns-b
ip netns add router

# ns-a <-> router link
ip link add veth-a type veth peer name veth-a-r
ip link set veth-a netns ns-a
ip link set veth-a-r netns router

# ns-b <-> router link
ip link add veth-b type veth peer name veth-b-r
ip link set veth-b netns ns-b
ip link set veth-b-r netns router

# Configure addressing
ip netns exec ns-a ip addr add 10.1.0.2/24 dev veth-a
ip netns exec ns-a ip link set veth-a up
ip netns exec ns-a ip link set lo up
ip netns exec ns-a ip route add default via 10.1.0.1

ip netns exec ns-b ip addr add 10.2.0.2/24 dev veth-b
ip netns exec ns-b ip link set veth-b up
ip netns exec ns-b ip link set lo up
ip netns exec ns-b ip route add default via 10.2.0.1

ip netns exec router ip addr add 10.1.0.1/24 dev veth-a-r
ip netns exec router ip addr add 10.2.0.1/24 dev veth-b-r
ip netns exec router ip link set veth-a-r up
ip netns exec router ip link set veth-b-r up
ip netns exec router ip link set lo up
ip netns exec router sysctl -w net.ipv4.ip_forward=1

# Test end-to-end
ip netns exec ns-a ping 10.2.0.2
ip netns exec ns-a traceroute 10.2.0.2
```

## Namespace + Cgroups for Bandwidth Limiting

```bash
# Use tc (traffic control) inside or outside the namespace
# Rate limit traffic on the veth peer in the default namespace
tc qdisc add dev veth-red-br root tbf rate 1mbit burst 32kbit latency 400ms

# Or use HTB for more sophisticated shaping
tc qdisc add dev veth-red-br root handle 1: htb default 10
tc class add dev veth-red-br parent 1: classid 1:10 htb rate 10mbit ceil 10mbit

# Combine with cgroups v2 for per-process network accounting
# The net_cls cgroup (v1) tags packets with a classid
# net_prio cgroup sets per-interface priority
echo 1 > /sys/fs/cgroup/net_cls/red/net_cls.classid
```

## Troubleshooting

```bash
# Run diagnostic tools inside the namespace
ip netns exec red ip addr show
ip netns exec red ip route show
ip netns exec red ip neigh show
ip netns exec red ss -tulnp
ip netns exec red iptables -L -n -v
ip netns exec red tcpdump -i veth-red -nn

# Check namespace connectivity
ip netns exec red ping -c 3 10.0.0.254

# Verify veth peer relationships
ip netns exec red ethtool -S veth-red   # shows peer_ifindex
ip netns exec red cat /sys/class/net/veth-red/iflink

# Find which namespace an interface is in
ip -all netns exec ip link show | grep veth

# Debug DNS inside namespace
ip netns exec red cat /etc/resolv.conf
ip netns exec red dig example.com

# Verify namespace isolation
ip netns exec red ss -tulnp   # only shows namespace-local sockets
ip netns exec red cat /proc/net/tcp
```

## Tips

- Always bring up the loopback interface inside new namespaces (`ip link set lo up`)
- Use `/etc/netns/<name>/resolv.conf` for per-namespace DNS configuration
- Veth pairs are the standard plumbing for connecting namespaces; one end moves, one stays
- macvlan gives direct physical network access without a bridge, but the host cannot communicate with macvlan interfaces in bridge mode
- ipvlan is preferred over macvlan when the switch enforces MAC-per-port limits
- Named namespaces persist without running processes; unnamed ones die with their last process
- Use `ip -all netns exec <cmd>` to run a command across all named namespaces at once
- Namespace-scoped iptables rules do not leak to or from the default namespace
- For production container networking, prefer CNI plugins over hand-rolled veth setups
- Combine `unshare --net --user` to create network namespaces without root (user namespace mapping required)
- Each veth pair adds ~1-2 microseconds of latency per packet crossing

## See Also

- bridge (Linux software bridge for L2 forwarding)
- vlan (802.1Q VLAN tagging)
- tuntap (TUN/TAP virtual devices)
- macvlan (MAC-based virtual interfaces)

## References

- [ip-netns(8) man page](https://man7.org/linux/man-pages/man8/ip-netns.8.html)
- [namespaces(7) man page](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [network_namespaces(7) man page](https://man7.org/linux/man-pages/man7/network_namespaces.7.html)
- [veth(4) man page](https://man7.org/linux/man-pages/man4/veth.4.html)
- [unshare(1) man page](https://man7.org/linux/man-pages/man1/unshare.1.html)
- [nsenter(1) man page](https://man7.org/linux/man-pages/man1/nsenter.1.html)
- [Kernel Documentation: Network Namespaces](https://www.kernel.org/doc/html/latest/networking/netns.html)
- [Docker Network Drivers](https://docs.docker.com/network/)
