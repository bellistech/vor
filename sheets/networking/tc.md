# tc (Traffic Control)

Linux kernel subsystem for managing packet scheduling, shaping, policing, and delay on network interfaces, using a hierarchy of qdiscs (queueing disciplines), classes, and filters attached to the egress (and with ingress qdiscs) path.

## tc Architecture

```
                    Ingress
Packets ─────────────┐
                     │
              ┌──────▼──────┐
              │  ingress     │  (policing only, no shaping)
              │  qdisc       │
              └──────┬──────┘
                     │
              ┌──────▼──────┐
              │  Routing     │
              │  Decision    │
              └──────┬──────┘
                     │
              ┌──────▼──────┐
              │  root qdisc  │  (default: pfifo_fast or fq_codel)
              │   1:0        │
              ├──────────────┤
              │ class 1:1    │──► filter ──► class 1:10 ──► qdisc
              │ class 1:2    │──► filter ──► class 1:20 ──► qdisc
              │ class 1:3    │──► filter ──► class 1:30 ──► qdisc
              └──────┬──────┘
                     │
                  Egress
                ─────────── Wire
```

## Viewing Current tc Configuration

```bash
# Show all qdiscs on all interfaces
tc qdisc show

# Show qdiscs on specific interface
tc qdisc show dev eth0

# Show classes
tc class show dev eth0

# Show filters
tc filter show dev eth0

# Show statistics
tc -s qdisc show dev eth0
tc -s class show dev eth0

# Show detailed (include invisible/default)
tc -d qdisc show dev eth0

# JSON output
tc -j -p qdisc show dev eth0
```

## HTB (Hierarchical Token Bucket)

```bash
# Classic bandwidth management qdisc
# Replace root qdisc with HTB
tc qdisc add dev eth0 root handle 1: htb default 30

# Root class: 100 Mbit total bandwidth
tc class add dev eth0 parent 1: classid 1:1 htb \
  rate 100mbit ceil 100mbit

# High priority class: guaranteed 50 Mbit, burst to 100 Mbit
tc class add dev eth0 parent 1:1 classid 1:10 htb \
  rate 50mbit ceil 100mbit prio 1

# Normal class: guaranteed 30 Mbit, burst to 80 Mbit
tc class add dev eth0 parent 1:1 classid 1:20 htb \
  rate 30mbit ceil 80mbit prio 2

# Best effort (default): guaranteed 20 Mbit
tc class add dev eth0 parent 1:1 classid 1:30 htb \
  rate 20mbit ceil 50mbit prio 3

# Attach leaf qdiscs (fq_codel for fair queuing)
tc qdisc add dev eth0 parent 1:10 handle 10: fq_codel
tc qdisc add dev eth0 parent 1:20 handle 20: fq_codel
tc qdisc add dev eth0 parent 1:30 handle 30: fq_codel

# Classify traffic with filters
# SSH → high priority
tc filter add dev eth0 parent 1: protocol ip prio 1 \
  u32 match ip dport 22 0xffff flowid 1:10

# HTTP/HTTPS → normal
tc filter add dev eth0 parent 1: protocol ip prio 2 \
  u32 match ip dport 80 0xffff flowid 1:20
tc filter add dev eth0 parent 1: protocol ip prio 2 \
  u32 match ip dport 443 0xffff flowid 1:20

# Everything else → default (1:30)
```

## TBF (Token Bucket Filter)

```bash
# Simple rate limiter — single queue, single rate
# Limit to 10 Mbit with 32 KB buffer and 10ms latency
tc qdisc add dev eth0 root tbf \
  rate 10mbit burst 32kbit latency 10ms

# With peak rate limiting
tc qdisc add dev eth0 root tbf \
  rate 10mbit burst 32kbit latency 10ms \
  peakrate 15mbit mtu 1600

# Replace existing root qdisc
tc qdisc replace dev eth0 root tbf \
  rate 1mbit burst 10kbit latency 50ms
```

## netem (Network Emulator)

```bash
# Add 100ms fixed delay
tc qdisc add dev eth0 root netem delay 100ms

# Variable delay: 100ms ± 20ms (uniform distribution)
tc qdisc add dev eth0 root netem delay 100ms 20ms

# Normal distribution delay: 100ms ± 20ms with correlation 25%
tc qdisc add dev eth0 root netem delay 100ms 20ms 25% \
  distribution normal

# Packet loss: 5%
tc qdisc add dev eth0 root netem loss 5%

# Correlated loss: 5% with 25% correlation (bursty)
tc qdisc add dev eth0 root netem loss 5% 25%

# Gilbert-Elliott model for bursty loss
tc qdisc add dev eth0 root netem loss gemodel 1% 10% 70% 0.1%

# Packet duplication: 1%
tc qdisc add dev eth0 root netem duplicate 1%

# Packet reordering: 25% of packets delayed by 10ms
tc qdisc add dev eth0 root netem delay 10ms reorder 25% 50%

# Packet corruption: 0.1%
tc qdisc add dev eth0 root netem corrupt 0.1%

# Rate limiting with netem (combine with delay)
tc qdisc add dev eth0 root netem delay 50ms rate 10mbit

# Combine multiple impairments
tc qdisc add dev eth0 root netem \
  delay 100ms 20ms distribution normal \
  loss 2% 25% \
  duplicate 0.5% \
  corrupt 0.1%

# Change netem parameters at runtime
tc qdisc change dev eth0 root netem delay 200ms loss 10%

# Remove netem
tc qdisc del dev eth0 root
```

## fq_codel (Fair Queuing Controlled Delay)

```bash
# Modern default qdisc — fights bufferbloat
tc qdisc add dev eth0 root fq_codel

# With custom parameters
tc qdisc add dev eth0 root fq_codel \
  limit 10240 \          # Max packets in queue
  flows 1024 \           # Number of flow buckets
  target 5ms \           # Acceptable delay target
  interval 100ms \       # Measurement interval
  quantum 1514 \         # Bytes per round-robin turn
  ecn                    # Enable ECN marking

# Check fq_codel stats
tc -s qdisc show dev eth0
# Shows: packets, drops, overlimits, requeues, maxpacket
```

## CAKE (Common Applications Kept Enhanced)

```bash
# Modern comprehensive shaper — replacement for HTB+fq_codel
# Simple bandwidth limit
tc qdisc add dev eth0 root cake bandwidth 50mbit

# Home gateway (with NAT, per-host fairness)
tc qdisc add dev eth0 root cake bandwidth 50mbit nat wash

# With overhead compensation (DSL, 8 bytes ATM overhead)
tc qdisc add dev eth0 root cake bandwidth 50mbit \
  overhead 8 atm

# Dual-stack (IPv4 + IPv6)
tc qdisc add dev eth0 root cake bandwidth 50mbit \
  dual-srchost dual-dsthost

# With diffserv support (4-tier)
tc qdisc add dev eth0 root cake bandwidth 50mbit diffserv4

# Ingress shaping via IFB
ip link add ifb0 type ifb
ip link set ifb0 up
tc qdisc add dev eth0 handle ffff: ingress
tc filter add dev eth0 parent ffff: protocol all \
  u32 match u32 0 0 action mirred egress redirect dev ifb0
tc qdisc add dev ifb0 root cake bandwidth 45mbit wash
```

## Filters and Classifiers

```bash
# u32 classifier — match by IP/port/protocol
# Match destination port 80
tc filter add dev eth0 parent 1: protocol ip prio 1 \
  u32 match ip dport 80 0xffff flowid 1:10

# Match source IP
tc filter add dev eth0 parent 1: protocol ip prio 1 \
  u32 match ip src 10.0.0.0/8 flowid 1:10

# Match IP protocol (TCP=6, UDP=17, ICMP=1)
tc filter add dev eth0 parent 1: protocol ip prio 1 \
  u32 match ip protocol 6 0xff flowid 1:10

# Match DSCP/TOS
tc filter add dev eth0 parent 1: protocol ip prio 1 \
  u32 match ip tos 0x68 0xfc flowid 1:10

# flower classifier — modern alternative
tc filter add dev eth0 parent 1: protocol ip prio 1 flower \
  ip_proto tcp dst_port 22 \
  action skbedit priority 1 \
  flowid 1:10

# flower with VLAN
tc filter add dev eth0 parent 1: protocol 802.1Q prio 1 flower \
  vlan_id 100 vlan_prio 5 \
  flowid 1:10

# Classify by firewall mark (requires iptables/nftables)
iptables -t mangle -A OUTPUT -p tcp --dport 22 -j MARK --set-mark 1
tc filter add dev eth0 parent 1: protocol ip prio 1 \
  handle 1 fw flowid 1:10
```

## Ingress Policing

```bash
# Ingress qdisc for inbound traffic policing
tc qdisc add dev eth0 handle ffff: ingress

# Police inbound to 100 Mbit — drop excess
tc filter add dev eth0 parent ffff: protocol ip prio 1 \
  u32 match u32 0 0 \
  police rate 100mbit burst 256k drop \
  flowid :1

# Police specific traffic (e.g., limit inbound UDP)
tc filter add dev eth0 parent ffff: protocol ip prio 1 \
  u32 match ip protocol 17 0xff \
  police rate 10mbit burst 64k drop \
  flowid :1

# Remove ingress qdisc
tc qdisc del dev eth0 handle ffff: ingress
```

## Common Recipes

```bash
# Simulate a slow 3G connection
tc qdisc add dev eth0 root netem delay 200ms 50ms \
  loss 5% rate 768kbit

# Simulate a flaky WiFi link
tc qdisc add dev eth0 root netem delay 20ms 10ms \
  loss 3% 25% duplicate 0.5%

# Bandwidth limit for a Docker container (via veth)
tc qdisc add dev veth123abc root tbf \
  rate 10mbit burst 32kbit latency 10ms

# Reset all tc rules on an interface
tc qdisc del dev eth0 root 2>/dev/null
tc qdisc del dev eth0 ingress 2>/dev/null

# Persist tc rules across reboots (systemd service)
# /etc/systemd/system/tc-rules.service
# [Service]
# Type=oneshot
# ExecStart=/usr/local/bin/tc-setup.sh
# RemainAfterExit=yes
```

## Tips

- Use `tc -s` to show statistics; watch for drops and overlimits to detect misconfigurations
- Prefer CAKE over HTB+fq_codel for home gateways; it handles NAT, per-host fairness, and overhead in one qdisc
- Always set `burst` appropriately in TBF/HTB; too small causes excessive drops, too large allows bursts past the rate
- Use netem only for testing; it adds overhead and can interact badly with production qdiscs
- Chain netem with a shaper: `tc qdisc add dev eth0 root handle 1: netem delay 50ms` then add HTB as child
- For ingress shaping, redirect through an IFB device; the ingress qdisc can only police (drop), not shape (delay)
- Use `flower` classifier over `u32` for readability and when matching on L2/VLAN fields
- Set fq_codel `target` to half the expected RTT; the default 5ms works well for most internet traffic
- Monitor bufferbloat with `tc -s qdisc show` — look at `delay` in fq_codel stats
- Back up working tc configurations; a typo in `tc qdisc replace` can kill connectivity on remote servers
- Use `tc qdisc replace` instead of `del`+`add` to avoid a gap in traffic control
- Combine `iptables -j MARK` with tc `fw` filter for flexible classification by application or user

## See Also

- iptables, nftables, ip, ethtool, netns, bridge, cake

## References

- [Linux Advanced Routing & Traffic Control (LARTC)](https://lartc.org/)
- [tc(8) man page](https://man7.org/linux/man-pages/man8/tc.8.html)
- [tc-htb(8)](https://man7.org/linux/man-pages/man8/tc-htb.8.html)
- [tc-netem(8)](https://man7.org/linux/man-pages/man8/tc-netem.8.html)
- [tc-cake(8)](https://man7.org/linux/man-pages/man8/tc-cake.8.html)
- [tc-fq_codel(8)](https://man7.org/linux/man-pages/man8/tc-fq_codel.8.html)
- [Bufferbloat Project](https://www.bufferbloat.net/)
