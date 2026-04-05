# Network Stack Tuning (Linux Kernel)

Tune the Linux TCP/IP stack for maximum throughput, minimum latency, or high connection counts using sysctl, ethtool, and BPF.

## TCP Buffer Tuning

```bash
# Core socket buffer maximums (applies to all protocols)
sysctl -w net.core.rmem_max=16777216      # 16 MB max receive buffer
sysctl -w net.core.wmem_max=16777216      # 16 MB max send buffer

# Default socket buffer sizes
sysctl -w net.core.rmem_default=1048576   # 1 MB default receive
sysctl -w net.core.wmem_default=1048576   # 1 MB default send

# TCP auto-tuning: min / default / max (bytes)
sysctl -w net.ipv4.tcp_rmem="4096 1048576 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 1048576 16777216"

# Enable TCP memory auto-tuning (on by default)
sysctl -w net.ipv4.tcp_moderate_rcvbuf=1

# Increase TCP memory limits: min / pressure / max (pages, 4 KB each)
sysctl -w net.ipv4.tcp_mem="1048576 1572864 2097152"

# Verify current TCP buffer settings
sysctl net.ipv4.tcp_rmem net.ipv4.tcp_wmem
sysctl net.core.rmem_max net.core.wmem_max
```

### Buffer Sizing Rule of Thumb

```bash
# Buffer >= BDP = Bandwidth * RTT
# 10G link, 10ms RTT: BDP = 1.25 GB/s * 0.01s = 12.5 MB
# So rmem_max/wmem_max >= 12,500,000

# For 100G, 20ms RTT: BDP = 12.5 GB/s * 0.02s = 250 MB
sysctl -w net.core.rmem_max=268435456    # 256 MB
sysctl -w net.core.wmem_max=268435456
sysctl -w net.ipv4.tcp_rmem="4096 1048576 268435456"
sysctl -w net.ipv4.tcp_wmem="4096 1048576 268435456"
```

## Congestion Control

```bash
# List available congestion control algorithms
sysctl net.ipv4.tcp_available_congestion_control

# Show current default
sysctl net.ipv4.tcp_congestion_control

# Set default to BBR (requires kernel 4.9+)
modprobe tcp_bbr
sysctl -w net.ipv4.tcp_congestion_control=bbr

# Set default to CUBIC (default on most distros)
sysctl -w net.ipv4.tcp_congestion_control=cubic

# Enable ECN (Explicit Congestion Notification)
sysctl -w net.ipv4.tcp_ecn=1           # Request ECN on outgoing SYN
sysctl -w net.ipv4.tcp_ecn=2           # Server-side ECN (respond if requested)

# Allow non-default algos for per-socket selection
sysctl -w net.ipv4.tcp_allowed_congestion_control="cubic bbr reno"
```

### Per-Socket Congestion Control

```bash
# Set per-socket via setsockopt (C)
# setsockopt(fd, IPPROTO_TCP, TCP_CONGESTION, "bbr", 3)

# Set via ip route for a destination
ip route change default via 10.0.0.1 congctl bbr

# Verify per-connection algo
ss -ti | grep -A1 "cubic\|bbr\|reno"
```

## Connection Tracking

```bash
# Check current connection tracking table size
sysctl net.netfilter.nf_conntrack_max
cat /proc/sys/net/netfilter/nf_conntrack_count   # current entries

# Increase max tracked connections (default often 65536)
sysctl -w net.netfilter.nf_conntrack_max=1048576

# Set hash table size (ideally conntrack_max / 4)
echo 262144 > /sys/module/nf_conntrack/parameters/hashsize

# Reduce timeout for established connections (default 432000 = 5 days)
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_established=86400

# Reduce TIME_WAIT tracking timeout
sysctl -w net.netfilter.nf_conntrack_tcp_timeout_time_wait=30

# Disable conntrack entirely for high-throughput servers
# (in iptables/nftables raw table)
iptables -t raw -A PREROUTING -p tcp --dport 80 -j NOTRACK
iptables -t raw -A OUTPUT -p tcp --sport 80 -j NOTRACK
```

## Socket Backlog

```bash
# Max pending connections in listen queue (default 4096 on newer kernels)
sysctl -w net.core.somaxconn=65535

# Max SYN_RECV queue length (half-open connections)
sysctl -w net.ipv4.tcp_max_syn_backlog=65535

# Enable SYN cookies (protection against SYN floods)
sysctl -w net.ipv4.tcp_syncookies=1

# Increase netdev backlog (packets queued for processing)
sysctl -w net.core.netdev_max_backlog=65536

# Verify application uses adequate listen backlog
ss -tlnp | awk '{print $3, $4, $5}'    # show Send-Q (listen backlog)
```

## Interrupt Coalescing

```bash
# Show current coalescing settings
ethtool -c eth0

# Set receive interrupt coalescing (batch interrupts)
ethtool -C eth0 rx-usecs 100           # delay 100us before interrupt
ethtool -C eth0 rx-frames 64           # or interrupt after 64 frames

# Set transmit coalescing
ethtool -C eth0 tx-usecs 100
ethtool -C eth0 tx-frames 64

# Enable adaptive coalescing (auto-tunes based on traffic)
ethtool -C eth0 adaptive-rx on adaptive-tx on

# Low-latency mode (disable coalescing)
ethtool -C eth0 rx-usecs 0 rx-frames 1
ethtool -C eth0 tx-usecs 0 tx-frames 1

# Show interrupt affinity (map IRQs to CPUs)
cat /proc/interrupts | grep eth0
cat /proc/irq/<IRQ>/smp_affinity_list

# Set IRQ affinity for NIC queues (pin to specific CPUs)
echo 2 > /proc/irq/<IRQ>/smp_affinity_list
```

## GRO/GSO/TSO Offloads

```bash
# Show all offload settings
ethtool -k eth0

# TCP Segmentation Offload (split large writes into MTU-sized segments)
ethtool -K eth0 tso on

# Generic Segmentation Offload (software TSO fallback)
ethtool -K eth0 gso on

# Generic Receive Offload (aggregate small packets into large ones)
ethtool -K eth0 gro on

# Large Receive Offload (more aggressive, can break forwarding)
ethtool -K eth0 lro off                # usually off for routers/bridges

# UDP Segmentation Offload (kernel 4.18+)
ethtool -K eth0 tx-udp-segmentation on

# Scatter-gather (prerequisite for TSO)
ethtool -K eth0 sg on

# Disable all offloads (debugging/troubleshooting)
ethtool -K eth0 tso off gso off gro off sg off

# Show offload statistics
ethtool -S eth0 | grep -i "offload\|gro\|tso\|gso"
```

## BPF Socket Tuning

```bash
# Attach BPF program to socket (conceptual — requires compiled BPF bytecode)
# setsockopt(fd, SOL_SOCKET, SO_ATTACH_BPF, &bpf_fd, sizeof(bpf_fd))

# BPF sockmap — redirect packets between sockets at kernel level
# Use bpftool to manage sockmap programs
bpftool prog list | grep sock
bpftool map list | grep sockmap

# Load a sockmap BPF program
bpftool prog load sockmap_prog.o /sys/fs/bpf/sockmap_prog

# Verify BPF socket programs attached
bpftool net show

# Show cgroup BPF programs (socket-level)
bpftool cgroup show /sys/fs/cgroup/ | grep sock

# Kernel bypass with AF_XDP (zero-copy packet I/O)
ethtool -K eth0 rxvlan off             # disable VLAN offload for XDP
ip link set eth0 xdp obj xdp_prog.o    # attach XDP program
```

## Network Queue Tuning

```bash
# Set transmit queue length (packets, default 1000)
ip link set eth0 txqueuelen 10000

# Network device processing budget (packets per softirq cycle)
sysctl -w net.core.netdev_budget=600           # default 300
sysctl -w net.core.netdev_budget_usecs=8000    # time budget in usecs

# Show current queue disciplines
tc qdisc show dev eth0

# Replace with multi-queue fair queueing (good for high-speed NICs)
tc qdisc replace dev eth0 root fq

# FQ with explicit pacing rate
tc qdisc replace dev eth0 root fq maxrate 10gbit

# Show queue statistics
tc -s qdisc show dev eth0

# Set RPS (Receive Packet Steering) across CPUs
echo "ff" > /sys/class/net/eth0/queues/rx-0/rps_cpus

# Set XPS (Transmit Packet Steering)
echo "1" > /sys/class/net/eth0/queues/tx-0/xps_cpus

# Set RFS (Receive Flow Steering) table size
sysctl -w net.core.rps_sock_flow_entries=32768
echo 2048 > /sys/class/net/eth0/queues/rx-0/rps_flow_cnt
```

## TIME_WAIT Tuning

```bash
# Allow reuse of TIME_WAIT sockets for new connections
sysctl -w net.ipv4.tcp_tw_reuse=1

# Reduce FIN timeout (default 60s)
sysctl -w net.ipv4.tcp_fin_timeout=15

# Count current TIME_WAIT sockets
ss -s | grep timewait
ss -ant | awk '/TIME-WAIT/ {count++} END {print count}'

# Max orphan sockets (connections with no owning process)
sysctl -w net.ipv4.tcp_max_orphans=65536

# Max TIME_WAIT buckets (limits memory usage)
sysctl -w net.ipv4.tcp_max_tw_buckets=1048576

# WARNING: Never use tcp_tw_recycle (removed in kernel 4.12)
# It breaks connections behind NAT
```

## Practical: 10G Network Profile

```bash
cat > /etc/sysctl.d/90-network-10g.conf << 'EOF'
# TCP buffers — 10G, ~10ms RTT, BDP ~12.5 MB
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.core.rmem_default = 1048576
net.core.wmem_default = 1048576
net.ipv4.tcp_rmem = 4096 1048576 16777216
net.ipv4.tcp_wmem = 4096 1048576 16777216

# Congestion control
net.ipv4.tcp_congestion_control = bbr

# Socket backlog
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 65535
net.core.netdev_max_backlog = 65536

# TIME_WAIT
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15

# Connection tracking (if needed — otherwise disable)
net.netfilter.nf_conntrack_max = 1048576
EOF
sysctl --system
```

## Practical: 25G/100G Network Profile

```bash
cat > /etc/sysctl.d/90-network-100g.conf << 'EOF'
# TCP buffers — 100G, ~10ms RTT, BDP ~125 MB
net.core.rmem_max = 268435456
net.core.wmem_max = 268435456
net.core.rmem_default = 16777216
net.core.wmem_default = 16777216
net.ipv4.tcp_rmem = 4096 16777216 268435456
net.ipv4.tcp_wmem = 4096 16777216 268435456
net.ipv4.tcp_mem = 4194304 8388608 16777216

# Congestion control
net.ipv4.tcp_congestion_control = bbr
net.ipv4.tcp_ecn = 1

# Backlog and queues
net.core.somaxconn = 65535
net.ipv4.tcp_max_syn_backlog = 131072
net.core.netdev_max_backlog = 250000
net.core.netdev_budget = 1000
net.core.netdev_budget_usecs = 10000

# TIME_WAIT
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 10
net.ipv4.tcp_max_tw_buckets = 2097152

# Enable window scaling and timestamps
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_timestamps = 1
net.ipv4.tcp_sack = 1
EOF
sysctl --system

# NIC offloads
ethtool -K eth0 tso on gso on gro on sg on
ethtool -G eth0 rx 8192 tx 8192                # ring buffer size
ethtool -C eth0 adaptive-rx on adaptive-tx on   # adaptive coalescing

# Queue discipline — FQ for pacing with BBR
tc qdisc replace dev eth0 root fq

# Jumbo frames (if network supports it)
ip link set eth0 mtu 9000
```

## Tips

- Always calculate BDP (bandwidth x RTT) before setting buffer sizes -- undersized buffers are the number one throughput killer
- BBR generally outperforms CUBIC on lossy or high-BDP paths; CUBIC is better on low-latency LANs
- Enable adaptive interrupt coalescing before manually tuning rx-usecs/tx-usecs
- Disable connection tracking entirely on high-throughput servers that do not need stateful firewalling
- Use `ss -tmi` to see per-socket buffer utilization, congestion window, and RTT in real time
- GRO should almost always be on; LRO should almost always be off on forwarding devices
- When using BBR, pair it with the `fq` qdisc for proper pacing
- Test changes under load with iperf3 or neper before deploying to production
- Persist settings in `/etc/sysctl.d/` files and ethtool commands in a networkd-dispatcher or udev rule

## See Also

- tcp
- sysctl
- ethtool
- iptables
- ss

## References

- `man 7 tcp` -- TCP protocol parameters
- `man 7 socket` -- Socket-level options
- `man 8 sysctl` -- Kernel parameter tuning
- `man 8 ethtool` -- NIC configuration
- `man 8 tc` -- Traffic control / queuing disciplines
- Linux kernel docs: `Documentation/networking/ip-sysctl.rst`
- RFC 7323 -- TCP Window Scaling and Timestamps
- RFC 8312 -- CUBIC Congestion Control
- RFC 9002 -- BBR Congestion Control
- Red Hat Performance Tuning Guide -- Network chapter
