# Network Namespaces -- Kernel Internals, Veth Mechanics, and Virtual Network Lab Design

> *A network namespace is a complete copy of the networking stack. Understanding how the kernel implements that copy -- and how packets traverse the boundary -- is the key to building anything from containers to multi-router test topologies.*

---

## 1. Kernel Implementation: struct net

### The Data Structure

Every network namespace in Linux is represented by a single `struct net` instance,
defined in `include/net/net_namespace.h`. This structure holds pointers to all
per-namespace networking state:

```c
struct net {
    refcount_t              passive;        // reference count
    spinlock_t              rules_mod_lock; // routing rules lock
    unsigned int            hash_mix;       // hash randomization seed

    struct ns_common        ns;             // generic namespace metadata
    struct list_head        list;           // linked list of all net namespaces
    struct list_head        exit_list;      // cleanup callbacks

    struct net_device       *loopback_dev;  // per-ns loopback
    struct hlist_head       *dev_name_head; // device hash by name
    struct hlist_head       *dev_index_head;// device hash by ifindex

    struct fib_rules_ops    *rules_ops[AF_MAX]; // routing rules per family
    struct net_generic      *gen;           // subsystem private data
    struct netns_ipv4       ipv4;           // IPv4 state (routes, iptables, sysctls)
    struct netns_ipv6       ipv6;           // IPv6 state
    struct netns_nf         nf;             // netfilter state
    struct netns_xt         xt;             // xtables state
    struct netns_ct         ct;             // conntrack state
    // ... more subsystem-specific state
};
```

The init namespace (`init_net`) is a statically allocated global. All other
namespaces are dynamically allocated via `copy_net_ns()` during `clone()` or
`unshare()` with `CLONE_NEWNET`.

### Per-Namespace Subsystem State

Each networking subsystem registers initialization and cleanup functions through
`pernet_operations`:

```c
struct pernet_operations {
    struct list_head list;
    int (*init)(struct net *net);    // called when namespace is created
    void (*exit)(struct net *net);   // called when namespace is destroyed
    void (*exit_batch)(struct list_head *net_exit_list); // batch cleanup
    int *id;                        // subsystem ID for net_generic
    size_t size;                    // private data size
};

register_pernet_subsys(&my_ops);    // register with the namespace framework
```

When a new namespace is created, the kernel iterates all registered
`pernet_operations` and calls each `init()` function. This creates fresh
routing tables, netfilter chains, conntrack tables, and socket hash tables
for the new namespace. The cost of namespace creation is therefore
proportional to the number of registered subsystems.

### Namespace Lifecycle

```
clone(CLONE_NEWNET) / unshare(CLONE_NEWNET)
       |
       v
  alloc_net()          -- allocate struct net
       |
       v
  setup_net()          -- call all pernet_operations.init()
       |                  (creates loopback, routing tables, iptables, etc.)
       v
  [namespace active]   -- processes can be assigned to it
       |                  network devices can be moved into it
       |                  sockets are bound to it
       v
  cleanup_net()        -- triggered when last reference drops
       |                  (no processes, no bind mounts, no open fds)
       v
  call all pernet_operations.exit()
       |
       v
  free_net()           -- memory reclaimed
```

Key points about lifecycle:

- A namespace stays alive as long as any of these hold: a process is in it,
  a bind mount exists (e.g., /run/netns/name), or a file descriptor to
  /proc/pid/ns/net is open.
- `ip netns add` creates a namespace and immediately bind-mounts it. The
  namespace persists even with zero processes.
- `ip netns del` removes the bind mount. If no other references exist, the
  kernel triggers cleanup.
- Cleanup is deferred to a workqueue (`net_cleanup_work`) to avoid blocking
  the caller.

## 2. Veth Pair Internals

### Architecture

A veth (virtual Ethernet) pair consists of two `net_device` structures, each
holding a pointer to the other. The driver is implemented in
`drivers/net/veth.c`:

```c
struct veth_priv {
    struct net_device __rcu *peer;   // pointer to the other end
    atomic64_t              dropped; // packets dropped
    struct bpf_prog         *_xdp_prog; // optional XDP program
    struct xdp_mem_info     xdp_mem;
    // ...
};
```

### Packet Traversal Path

When a packet is transmitted on one end of a veth pair, it is received on
the other end. The path through the kernel:

```
Application in Namespace A
       |
  send() / write()
       |
       v
  socket layer (sock_sendmsg)
       |
       v
  IP layer (ip_output -> ip_finish_output)
       |
       v
  neighbor/ARP resolution (neigh_output)
       |
       v
  dev_queue_xmit(skb, veth-a)
       |
       v
  veth_xmit(skb)                    [drivers/net/veth.c]
       |
       +-- skb->dev = peer (veth-b)  // redirect to peer device
       +-- skb reset MAC header
       +-- if XDP program on peer:
       |       run XDP (XDP_PASS / XDP_DROP / XDP_TX / XDP_REDIRECT)
       |
       v
  netif_rx(skb)                      // inject into peer's receive path
       |                              // NOTE: this crosses namespace boundary
       v
  __netif_receive_skb(skb)           // in Namespace B's context
       |
       v
  ip_rcv() -> ip_rcv_finish()        // Namespace B's routing table
       |
       v
  local delivery or forwarding       // Namespace B's iptables
       |
       v
  socket receive buffer               // Application in Namespace B
```

Critical implementation details:

- **No actual copy occurs** between namespaces. The `skb` (socket buffer) is
  simply re-tagged with the peer device and injected into the receive path.
  This makes veth pairs extremely efficient.

- The `netif_rx()` call puts the skb on the peer CPU's backlog. If the
  peer's backlog is full, the packet is dropped (visible in `ethtool -S`
  as `peer_iflink` drops).

- GRO (Generic Receive Offload) is supported on veth devices since kernel 4.19,
  improving throughput for TCP workloads.

- XDP programs can be attached to veth devices (since kernel 4.19 for native
  mode). XDP runs before the packet enters the IP stack, enabling fast-path
  packet processing at the namespace boundary.

### Veth Performance Characteristics

Latency overhead per veth crossing (measured on modern hardware):

```
Direct loopback:          ~10-15 us round-trip
Single veth pair:         ~12-18 us round-trip  (+1-3 us per crossing)
Bridge + two veth pairs:  ~18-30 us round-trip  (+5-10 us for bridge FDB lookup)
```

Throughput (single TCP stream, iperf3, modern kernel):

```
Loopback:                 ~40-60 Gbps
Veth pair (no bridge):    ~20-40 Gbps
Veth + bridge:            ~15-25 Gbps
Veth + iptables NAT:      ~8-15 Gbps  (conntrack overhead)
```

The primary throughput bottleneck is softirq processing. Each veth crossing
involves a full NAPI poll cycle on the receiving side. With multi-queue veth
(kernel 5.14+), throughput scales with CPU cores.

## 3. Routing Between Namespaces

### IP Forwarding

For one namespace to route packets to another through an intermediary (router)
namespace, IP forwarding must be enabled in the router namespace:

```bash
ip netns exec router sysctl -w net.ipv4.ip_forward=1
```

This sets the `IPV4_DEVCONF_FORWARDING` flag in the namespace's `netns_ipv4`
structure, causing `ip_forward()` to accept packets not destined for local
addresses.

### Policy Routing

Each namespace maintains independent routing policy databases (RPDB). This
allows sophisticated routing decisions per namespace:

```bash
# Add a custom routing table in the router namespace
ip netns exec router ip route add 10.1.0.0/24 dev veth-a-r table 100
ip netns exec router ip route add 10.2.0.0/24 dev veth-b-r table 100

# Policy rule: packets from 10.1.0.0/24 use table 100
ip netns exec router ip rule add from 10.1.0.0/24 lookup 100

# Asymmetric routing: different paths for different source ranges
ip netns exec router ip rule add from 10.2.0.0/24 lookup 200
ip netns exec router ip route add default via 10.99.0.1 table 200
```

### Connected Routing vs Static Routes

When a veth endpoint is assigned an address, the kernel automatically creates
a connected route for that subnet. For cross-subnet routing:

```bash
# Namespace "client" (10.1.0.0/24) wants to reach "server" (10.2.0.0/24)
# via router namespace

# Option 1: Static routes
ip netns exec client ip route add 10.2.0.0/24 via 10.1.0.1

# Option 2: Default gateway
ip netns exec client ip route add default via 10.1.0.1

# Option 3: Source-based routing (multi-homed namespace)
ip netns exec client ip route add 10.2.0.0/24 via 10.1.0.1 src 10.1.0.2
```

## 4. Iptables Per-Namespace

### Independent Filter State

Each network namespace has completely independent netfilter state:

- Separate chain sets (INPUT, OUTPUT, FORWARD, plus custom chains)
- Separate conntrack table (`struct netns_ct`)
- Separate NAT rules
- Separate raw, mangle, and security tables

```bash
# Namespace-local firewall
ip netns exec red iptables -P INPUT DROP
ip netns exec red iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
ip netns exec red iptables -A INPUT -p tcp --dport 80 -j ACCEPT

# This has zero effect on the default namespace or any other namespace
iptables -L -n   # default namespace rules unchanged
```

### Conntrack Isolation

Connection tracking is per-namespace. This means:

- A connection in namespace A has no conntrack entry in namespace B
- NAT mappings are namespace-local
- Conntrack table size (`nf_conntrack_max`) is set per-namespace via sysctl
- Conntrack hash table parameters are shared globally (kernel compile-time)

```bash
# Per-namespace conntrack limits
ip netns exec red sysctl -w net.netfilter.nf_conntrack_max=65536

# View conntrack entries per namespace
ip netns exec red conntrack -L
ip netns exec red conntrack -C   # count
```

### FORWARD Chain in Router Namespaces

When a router namespace forwards packets between connected namespaces, the
FORWARD chain in that router namespace is traversed. This is where
inter-namespace firewall policy is enforced:

```bash
# Allow forwarding only specific traffic between namespaces
ip netns exec router iptables -P FORWARD DROP
ip netns exec router iptables -A FORWARD -i veth-a-r -o veth-b-r \
    -p tcp --dport 80 -j ACCEPT
ip netns exec router iptables -A FORWARD -i veth-b-r -o veth-a-r \
    -m state --state ESTABLISHED,RELATED -j ACCEPT
```

## 5. Network Namespaces and eBPF

### TC Programs Per-Namespace

Traffic Control (TC) eBPF programs can be attached to any interface, including
veth endpoints inside namespaces. Since each namespace has its own interfaces,
TC programs are inherently per-namespace:

```bash
# Attach a TC-BPF program to the veth inside namespace "red"
ip netns exec red tc qdisc add dev veth-red clsact
ip netns exec red tc filter add dev veth-red ingress \
    bpf da obj filter.o sec tc_ingress

# The program runs in the context of namespace "red"
# It sees packets arriving at veth-red before they enter the IP stack
```

TC-BPF at namespace boundaries enables:

- Per-namespace packet filtering without iptables overhead
- Traffic accounting and rate limiting
- Custom load balancing (Cilium uses this for Kubernetes services)
- Transparent proxying

### XDP Programs Per-Namespace

XDP (eXpress Data Path) programs can be attached to veth devices in native
mode (since kernel 4.19). XDP runs before `skb` allocation, providing the
lowest-latency programmable hook:

```bash
# Attach XDP to veth in namespace
ip netns exec red ip link set dev veth-red xdpgeneric obj xdp_drop.o sec xdp

# Native XDP on veth (kernel 5.9+)
ip netns exec red ip link set dev veth-red xdp obj xdp_prog.o sec xdp
```

Important caveats:

- XDP on veth operates on the **receiving** side. An XDP program on veth-red
  in namespace "red" processes packets arriving from the peer (veth-blue).
- `XDP_TX` on a veth device sends the packet back out the same veth, which
  means it arrives at the peer namespace. This enables XDP-based ping-pong
  without entering the IP stack.
- `XDP_REDIRECT` from a veth can forward to another device (including a
  device in a different namespace if using `bpf_redirect_peer()` since
  kernel 5.10).

### bpf_redirect_peer()

The `bpf_redirect_peer()` helper (kernel 5.10+) is a game-changer for
container networking. It allows an XDP or TC program to redirect a packet
directly to the peer veth device, bypassing the normal transmit path:

```
Normal veth path:    TX softirq -> veth_xmit -> netif_rx -> RX softirq
bpf_redirect_peer(): Directly delivers to peer's receive path (skips TX softirq)
```

This reduces per-packet latency by ~1-2 microseconds and is used by Cilium
for high-performance Kubernetes pod networking.

### Namespace-Aware BPF Maps

BPF maps are global (not per-namespace), but programs can use namespace
awareness to implement per-namespace policy:

```c
// In BPF program: identify the namespace by net cookie
__u64 cookie = bpf_get_netns_cookie(ctx);

struct policy *p = bpf_map_lookup_elem(&ns_policy_map, &cookie);
if (p && p->action == DROP)
    return TC_ACT_SHOT;
```

The `bpf_get_netns_cookie()` helper returns a unique identifier for the
current network namespace, allowing a single BPF program to enforce
different policies per namespace.

## 6. Performance Overhead Analysis

### Namespace Creation Cost

Creating a network namespace involves allocating and initializing all
per-namespace subsystem state. Measured overhead:

```
Operation                          Time (approx.)
-------------------------------------------------
unshare(CLONE_NEWNET):             ~0.5-1.0 ms
ip netns add (unshare + bind):     ~1.5-3.0 ms
Full namespace + veth setup:       ~5-10 ms
Docker container start (net only): ~20-50 ms
```

### Memory Overhead

Each namespace consumes memory for its networking state:

```
Component                          Per-Namespace Memory
-------------------------------------------------------
struct net + subsystem state:      ~30-50 KB
Routing table (empty):             ~2-4 KB
Iptables (default chains):         ~8-12 KB
Conntrack table (default):         ~16 KB (hash table)
Loopback device:                   ~2 KB
Total baseline:                    ~60-80 KB

With activity:
  + routing entries:               ~128 bytes each
  + conntrack entries:             ~320 bytes each
  + socket hash entries:           ~64 bytes each
```

For 1000 namespaces (e.g., 1000 containers), baseline memory overhead is
approximately 60-80 MB. This is dominated by conntrack hash tables.

### Packet Processing Overhead

Comparison of packet processing paths:

```
Path                                Added Latency     Throughput Impact
----------------------------------------------------------------------
Loopback (same namespace):          baseline           baseline
Veth pair (two namespaces):         +1-3 us            -10-20%
Veth + bridge (three namespaces):   +5-10 us           -30-40%
Veth + iptables FORWARD:            +2-5 us            -15-25%
Veth + conntrack NAT:               +5-10 us           -40-50%
Veth + TC-BPF:                      +0.5-1 us          -5-10%
Veth + XDP (native):                +0.2-0.5 us        -2-5%
```

### Scaling Considerations

- The kernel's network namespace list is protected by a mutex
  (`net_mutex`). Creating/destroying many namespaces concurrently can
  serialize on this lock.
- Each namespace's routing table is independent, so routing lookups do not
  contend across namespaces.
- Netfilter conntrack uses per-CPU locks within a namespace but has no
  cross-namespace contention.
- The softirq budget is shared across all namespaces on a given CPU. A
  namespace with high packet rates can starve other namespaces on the same
  CPU.

## 7. Building a Virtual Network Lab

### Multi-Router Topology Walkthrough

This section builds a complete three-subnet topology with two routers,
a DNS server namespace, and NAT to the internet:

```
Topology:
                           [internet]
                               |
                          [gw] (NAT)
                         /           \
                   [router-a]     [router-b]
                      |               |
                  [subnet-a]      [subnet-b]
                  10.1.0.0/24     10.2.0.0/24
                   |     |         |     |
                [host-a1][host-a2][host-b1][host-b2]
```

#### Step 1: Create All Namespaces

```bash
for ns in gw router-a router-b host-a1 host-a2 host-b1 host-b2; do
    ip netns add $ns
    ip netns exec $ns ip link set lo up
done
```

#### Step 2: Create and Wire Veth Pairs

```bash
# gw <-> router-a (172.16.0.0/30)
ip link add veth-gw-a type veth peer name veth-a-gw
ip link set veth-gw-a netns gw
ip link set veth-a-gw netns router-a

# gw <-> router-b (172.16.0.4/30)
ip link add veth-gw-b type veth peer name veth-b-gw
ip link set veth-gw-b netns gw
ip link set veth-b-gw netns router-b

# router-a <-> subnet-a bridge
ip netns exec router-a ip link add br-a type bridge
ip netns exec router-a ip link set br-a up

# host-a1 <-> router-a
ip link add veth-a1 type veth peer name veth-a1-br
ip link set veth-a1 netns host-a1
ip link set veth-a1-br netns router-a
ip netns exec router-a ip link set veth-a1-br master br-a

# host-a2 <-> router-a
ip link add veth-a2 type veth peer name veth-a2-br
ip link set veth-a2 netns host-a2
ip link set veth-a2-br netns router-a
ip netns exec router-a ip link set veth-a2-br master br-a

# router-b <-> subnet-b bridge
ip netns exec router-b ip link add br-b type bridge
ip netns exec router-b ip link set br-b up

# host-b1 <-> router-b
ip link add veth-b1 type veth peer name veth-b1-br
ip link set veth-b1 netns host-b1
ip link set veth-b1-br netns router-b
ip netns exec router-b ip link set veth-b1-br master br-b

# host-b2 <-> router-b
ip link add veth-b2 type veth peer name veth-b2-br
ip link set veth-b2 netns host-b2
ip link set veth-b2-br netns router-b
ip netns exec router-b ip link set veth-b2-br master br-b
```

#### Step 3: Assign Addresses and Bring Up Interfaces

```bash
# Gateway namespace
ip netns exec gw ip addr add 172.16.0.1/30 dev veth-gw-a
ip netns exec gw ip addr add 172.16.0.5/30 dev veth-gw-b
ip netns exec gw ip link set veth-gw-a up
ip netns exec gw ip link set veth-gw-b up
ip netns exec gw sysctl -w net.ipv4.ip_forward=1

# Router A
ip netns exec router-a ip addr add 172.16.0.2/30 dev veth-a-gw
ip netns exec router-a ip addr add 10.1.0.1/24 dev br-a
ip netns exec router-a ip link set veth-a-gw up
ip netns exec router-a ip link set veth-a1-br up
ip netns exec router-a ip link set veth-a2-br up
ip netns exec router-a sysctl -w net.ipv4.ip_forward=1

# Router B
ip netns exec router-b ip addr add 172.16.0.6/30 dev veth-b-gw
ip netns exec router-b ip addr add 10.2.0.1/24 dev br-b
ip netns exec router-b ip link set veth-b-gw up
ip netns exec router-b ip link set veth-b1-br up
ip netns exec router-b ip link set veth-b2-br up
ip netns exec router-b sysctl -w net.ipv4.ip_forward=1

# Hosts
ip netns exec host-a1 ip addr add 10.1.0.11/24 dev veth-a1
ip netns exec host-a1 ip link set veth-a1 up
ip netns exec host-a1 ip route add default via 10.1.0.1

ip netns exec host-a2 ip addr add 10.1.0.12/24 dev veth-a2
ip netns exec host-a2 ip link set veth-a2 up
ip netns exec host-a2 ip route add default via 10.1.0.1

ip netns exec host-b1 ip addr add 10.2.0.11/24 dev veth-b1
ip netns exec host-b1 ip link set veth-b1 up
ip netns exec host-b1 ip route add default via 10.2.0.1

ip netns exec host-b2 ip addr add 10.2.0.12/24 dev veth-b2
ip netns exec host-b2 ip link set veth-b2 up
ip netns exec host-b2 ip route add default via 10.2.0.1
```

#### Step 4: Configure Routing

```bash
# Router A: route to subnet B via gateway
ip netns exec router-a ip route add 10.2.0.0/24 via 172.16.0.1

# Router B: route to subnet A via gateway
ip netns exec router-b ip route add 10.1.0.0/24 via 172.16.0.5

# Gateway: routes to both subnets
ip netns exec gw ip route add 10.1.0.0/24 via 172.16.0.2
ip netns exec gw ip route add 10.2.0.0/24 via 172.16.0.6
```

#### Step 5: NAT for Internet Access (Optional)

```bash
# Move a veth pair from default namespace into gw for internet access
ip link add veth-inet type veth peer name veth-inet-gw
ip link set veth-inet-gw netns gw
ip addr add 192.168.99.1/24 dev veth-inet
ip link set veth-inet up
ip netns exec gw ip addr add 192.168.99.2/24 dev veth-inet-gw
ip netns exec gw ip link set veth-inet-gw up
ip netns exec gw ip route add default via 192.168.99.1

# NAT in default namespace for gw's outbound traffic
iptables -t nat -A POSTROUTING -s 192.168.99.0/24 -o eth0 -j MASQUERADE
sysctl -w net.ipv4.ip_forward=1

# NAT in gw namespace for subnet traffic going to internet
ip netns exec gw iptables -t nat -A POSTROUTING -s 10.0.0.0/8 \
    -o veth-inet-gw -j MASQUERADE
```

#### Step 6: Verification

```bash
# End-to-end connectivity
ip netns exec host-a1 ping -c 2 10.2.0.11     # cross-subnet
ip netns exec host-a1 traceroute -n 10.2.0.11  # should show 10.1.0.1 -> 172.16.0.1 -> 10.2.0.1

# Verify isolation
ip netns exec host-a1 ss -tulnp  # only local sockets
ip netns exec host-b1 ss -tulnp  # completely independent

# Run a service
ip netns exec host-b1 python3 -m http.server 8080 &
ip netns exec host-a1 curl http://10.2.0.11:8080/
```

#### Cleanup

```bash
for ns in gw router-a router-b host-a1 host-a2 host-b1 host-b2; do
    ip netns del $ns
done
# Deleting namespaces automatically removes all veth pairs and bridges within
```

### Design Principles for Virtual Labs

1. **Address planning first.** Assign non-overlapping subnets to each segment
   before creating any interfaces. Use /30 for point-to-point router links.

2. **Router namespaces are the control points.** Firewalling, NAT, and policy
   routing belong in router namespaces, not in endpoint namespaces.

3. **Bridges for broadcast domains.** Each subnet that needs L2 connectivity
   (ARP, mDNS) gets its own bridge inside the router namespace.

4. **Name conventions matter.** Use `veth-<src>-<dst>` naming so you can
   identify both ends of a pair from either namespace.

5. **Test incrementally.** Verify each link with `ping` before adding the
   next. Debugging a fully-wired but broken topology is much harder than
   catching a misconfiguration early.

---

## References

- [include/net/net_namespace.h (kernel source)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/include/net/net_namespace.h)
- [drivers/net/veth.c (kernel source)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/drivers/net/veth.c)
- [BPF and XDP Reference Guide (Cilium)](https://docs.cilium.io/en/stable/bpf/)
- [bpf_redirect_peer() commit (kernel 5.10)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/commit/?id=9aa1206e8f48)
- [namespaces(7) man page](https://man7.org/linux/man-pages/man7/namespaces.7.html)
- [network_namespaces(7) man page](https://man7.org/linux/man-pages/man7/network_namespaces.7.html)
- [veth(4) man page](https://man7.org/linux/man-pages/man4/veth.4.html)
