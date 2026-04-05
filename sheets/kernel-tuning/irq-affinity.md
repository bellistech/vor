# IRQ Affinity & Interrupt Tuning (Linux Network Stack)

Pin interrupts to CPUs, steer packets to the right core, and eliminate jitter from the data path.

---

## IRQ Affinity

Set which CPUs handle a specific interrupt via bitmask or CPU list.

```bash
# Show current affinity for IRQ 42
cat /proc/irq/42/smp_affinity        # hex bitmask (e.g., f = CPUs 0-3)
cat /proc/irq/42/smp_affinity_list   # CPU list   (e.g., 0-3)

# Pin IRQ 42 to CPU 2 only
echo 4 > /proc/irq/42/smp_affinity          # bitmask: bit 2 = 0x4
echo 2 > /proc/irq/42/smp_affinity_list     # CPU list form

# Pin IRQ 42 to CPUs 0,2,4
echo 15 > /proc/irq/42/smp_affinity         # 0x15 = bits 0,2,4
echo 0,2,4 > /proc/irq/42/smp_affinity_list

# List all IRQs and their CPU affinity
for irq in /proc/irq/*/smp_affinity_list; do
  echo "IRQ $(basename $(dirname $irq)): $(cat $irq)"
done

# Show IRQ counts per CPU
cat /proc/interrupts | head -1; grep eth /proc/interrupts
```

## irqbalance

The userspace daemon that auto-distributes IRQs across CPUs. Often needs tuning or disabling for latency-sensitive workloads.

```bash
# Check if irqbalance is running
systemctl status irqbalance

# Stop irqbalance (take manual control)
systemctl stop irqbalance
systemctl disable irqbalance

# Run irqbalance in one-shot mode (set once, then exit)
irqbalance --oneshot

# Ban CPUs 0-3 from irqbalance (reserve for application)
# /etc/sysconfig/irqbalance or /etc/default/irqbalance
IRQBALANCE_BANNED_CPULIST=0-3

# Use a policy script to customize placement
IRQBALANCE_ARGS="--policyscript=/etc/irqbalance/policy.sh"

# Policy script receives IRQ info on stdin, outputs:
#   ban=true     — skip this IRQ
#   balance_level=none|package|cache|core
#   numa_node=N  — prefer this NUMA node
cat > /etc/irqbalance/policy.sh << 'POLICY'
#!/bin/bash
read TYPE NUMBER CPULIST
# Ban storage IRQs from rebalancing
if grep -q nvme /proc/irq/$NUMBER/actions 2>/dev/null; then
  echo "ban=true"
fi
POLICY
chmod +x /etc/irqbalance/policy.sh

# Debug irqbalance decisions
irqbalance --foreground --debug
```

## RPS (Receive Packet Steering)

Software-based receive steering for NICs without multi-queue or with fewer queues than CPUs.

```bash
# Enable RPS on eth0 queue 0, steering to CPUs 0-7
echo ff > /sys/class/net/eth0/queues/rx-0/rps_cpus

# Set RPS flow table size (must be power of 2, per-CPU)
echo 32768 > /sys/class/net/eth0/queues/rx-0/rps_flow_cnt

# Global flow table entries (sum of all per-queue entries)
echo 32768 > /proc/sys/net/core/rps_sock_flow_entries

# Enable RPS on all RX queues of eth0
for rxq in /sys/class/net/eth0/queues/rx-*/rps_cpus; do
  echo ff > "$rxq"
done
```

## RFS (Receive Flow Steering)

Steers packets to the CPU where the application is running — builds on RPS.

```bash
# Set global flow table (should be >= active connections)
echo 32768 > /proc/sys/net/core/rps_sock_flow_entries

# Set per-queue flow count (global / number_of_rx_queues)
QUEUES=$(ls -d /sys/class/net/eth0/queues/rx-* | wc -l)
PER_Q=$((32768 / QUEUES))
for rxq in /sys/class/net/eth0/queues/rx-*/rps_flow_cnt; do
  echo $PER_Q > "$rxq"
done
```

## XPS (Transmit Packet Steering)

Map TX queues to CPUs so each core transmits on a dedicated queue, avoiding lock contention.

```bash
# Map TX queue 0 to CPU 0, queue 1 to CPU 1, etc.
echo 1  > /sys/class/net/eth0/queues/tx-0/xps_cpus
echo 2  > /sys/class/net/eth0/queues/tx-1/xps_cpus
echo 4  > /sys/class/net/eth0/queues/tx-2/xps_cpus
echo 8  > /sys/class/net/eth0/queues/tx-3/xps_cpus

# Automate: 1:1 CPU-to-TX-queue mapping
i=0
for txq in /sys/class/net/eth0/queues/tx-*/xps_cpus; do
  echo $((1 << i)) > "$txq"
  ((i++))
done

# NUMA-aware XPS: map TX queues only to local-node CPUs
LOCAL_CPUS=$(cat /sys/class/net/eth0/device/local_cpulist)
for txq in /sys/class/net/eth0/queues/tx-*/xps_cpus; do
  echo "$LOCAL_CPUS" > "${txq%xps_cpus}xps_rxqs"  2>/dev/null
  cat /sys/devices/system/node/node0/cpumap > "$txq"
done
```

## NAPI & Busy Polling

Reduce interrupt overhead by polling the NIC in software. Trades CPU cycles for lower latency.

```bash
# Enable busy polling globally (microseconds to poll before sleeping)
echo 50 > /proc/sys/net/core/busy_read    # poll up to 50us on read
echo 50 > /proc/sys/net/core/busy_poll    # poll up to 50us on poll/select

# Per-socket busy polling (in application code)
# setsockopt(fd, SOL_SOCKET, SO_BUSY_POLL, &usec, sizeof(usec));

# NAPI: defer hard IRQs and use GRO flush timeout
echo 2     > /sys/class/net/eth0/napi_defer_hard_irqs
echo 200000 > /sys/class/net/eth0/gro_flush_timeout   # 200us in nanoseconds

# Show current NAPI settings
for iface in eth0; do
  echo "=== $iface ==="
  echo "napi_defer_hard_irqs: $(cat /sys/class/net/$iface/napi_defer_hard_irqs 2>/dev/null || echo N/A)"
  echo "gro_flush_timeout:    $(cat /sys/class/net/$iface/gro_flush_timeout 2>/dev/null || echo N/A)"
done

# Disable GRO if causing latency spikes
ethtool -K eth0 gro off
```

## CPU Isolation

Reserve CPUs exclusively for application threads. Kernel and IRQs stay off isolated cores.

```bash
# Boot parameters (add to GRUB_CMDLINE_LINUX)
# isolcpus=4-7               # remove CPUs 4-7 from general scheduler
# irqaffinity=0-3            # confine IRQs to CPUs 0-3
# nohz_full=4-7              # disable timer tick on isolated CPUs
# rcu_nocbs=4-7              # offload RCU callbacks from isolated CPUs

# Apply via GRUB
cat >> /etc/default/grub << 'EOF'
GRUB_CMDLINE_LINUX="isolcpus=4-7 irqaffinity=0-3 nohz_full=4-7 rcu_nocbs=4-7"
EOF
grub-mkconfig -o /boot/grub/grub.cfg

# Verify isolation at runtime
cat /sys/devices/system/cpu/isolated
cat /proc/cmdline | tr ' ' '\n' | grep -E 'isolcpus|irqaffinity|nohz_full|rcu_nocbs'

# Move all movable IRQs off isolated CPUs
for irq in /proc/irq/*/smp_affinity_list; do
  echo "0-3" > "$irq" 2>/dev/null
done

# Use cset for dynamic CPU shielding (no reboot)
cset shield --cpu=4-7 --kthread=on
cset shield --exec -- ./myapp --threads=4
```

## MSI/MSI-X — Multi-Queue Interrupts

Modern NICs use MSI-X to provide one interrupt vector per queue, enabling true per-CPU parallelism.

```bash
# Check MSI-X capability and vector count
lspci -vvv -s $(ethtool -i eth0 | awk '/bus-info/{print $2}') | grep -i msi

# Show number of combined queues
ethtool -l eth0

# Set queue count to match CPU count (or NUMA-local CPUs)
ethtool -L eth0 combined 8

# View IRQ-to-queue mapping
grep eth0 /proc/interrupts

# Map each queue's IRQ to its own CPU
IRQS=($(grep eth0 /proc/interrupts | awk '{print $1}' | tr -d ':'))
for i in "${!IRQS[@]}"; do
  echo $i > /proc/irq/${IRQS[$i]}/smp_affinity_list
done
```

## Practical: Full NIC Tuning Script

```bash
#!/bin/bash
# nic-irq-tune.sh — Pin NIC IRQs 1:1 to CPUs, enable RFS, configure NAPI
set -euo pipefail

IFACE="${1:?Usage: $0 <interface>}"
NUMA_NODE=$(cat /sys/class/net/$IFACE/device/numa_node 2>/dev/null || echo 0)
[ "$NUMA_NODE" -lt 0 ] && NUMA_NODE=0
LOCAL_CPUS=$(cat /sys/devices/system/node/node${NUMA_NODE}/cpulist)

echo "Interface: $IFACE  NUMA node: $NUMA_NODE  Local CPUs: $LOCAL_CPUS"

# Stop irqbalance for this interface
systemctl stop irqbalance 2>/dev/null || true

# Pin each IRQ to a NUMA-local CPU (round-robin)
IFS=',' read -ra CPU_ARRAY <<< "$(echo $LOCAL_CPUS | sed 's/-/,/g')"
# Expand ranges
CPUS=()
for c in $(seq $(echo $LOCAL_CPUS | tr ',-' ' ' | head -1) \
              $(echo $LOCAL_CPUS | tr ',-' ' ' | tail -1)); do
  CPUS+=($c)
done

IRQS=($(grep "$IFACE" /proc/interrupts | awk '{print $1}' | tr -d ':'))
for i in "${!IRQS[@]}"; do
  CPU=${CPUS[$((i % ${#CPUS[@]}))]}
  echo "$CPU" > /proc/irq/${IRQS[$i]}/smp_affinity_list
  echo "  IRQ ${IRQS[$i]} -> CPU $CPU"
done

# Enable RFS
RX_QUEUES=$(ls -d /sys/class/net/$IFACE/queues/rx-* 2>/dev/null | wc -l)
FLOW_ENTRIES=32768
echo $FLOW_ENTRIES > /proc/sys/net/core/rps_sock_flow_entries
PER_Q=$((FLOW_ENTRIES / RX_QUEUES))
for rxq in /sys/class/net/$IFACE/queues/rx-*/rps_flow_cnt; do
  echo $PER_Q > "$rxq"
done

# XPS: 1:1 TX queue to CPU
i=0
for txq in /sys/class/net/$IFACE/queues/tx-*/xps_cpus; do
  CPU=${CPUS[$((i % ${#CPUS[@]}))]}
  echo $((1 << CPU)) > "$txq"
  ((i++))
done

# NAPI tuning
echo 2      > /sys/class/net/$IFACE/napi_defer_hard_irqs 2>/dev/null || true
echo 200000 > /sys/class/net/$IFACE/gro_flush_timeout 2>/dev/null || true

echo "Done. Verify with: grep $IFACE /proc/interrupts"
```

## Tips

- Always check NUMA topology first: `numactl --hardware`. Crossing NUMA boundaries for IRQs adds 40-100ns per packet.
- Use `watch -n1 'cat /proc/interrupts | grep eth'` to verify IRQ distribution is balanced.
- Disable irqbalance on latency-sensitive systems. It rebalances every 10s and can move IRQs mid-flow.
- Set NIC queue count to match NUMA-local CPU count, not total CPU count.
- Busy polling is powerful but burns CPU. Only enable it on cores you can afford to dedicate.
- `napi_defer_hard_irqs=2` with `gro_flush_timeout=200000` is a good starting point for 10G+ NICs.
- For DPDK/AF_XDP workloads, isolate CPUs and skip kernel steering entirely.
- `perf stat -e irq:irq_handler_entry -a sleep 1` shows interrupt rate per second.

## See Also

- `sheets/kernel-tuning/cpu-pinning.md` — CPU affinity, cgroups, taskset
- `sheets/kernel-tuning/network-stack.md` — Socket buffers, TCP tuning, congestion control
- `sheets/kernel-tuning/numa.md` — NUMA topology, memory policy, numactl

## References

- [Linux kernel: SMP IRQ affinity](https://www.kernel.org/doc/Documentation/IRQ-affinity.txt)
- [Linux kernel: Scaling in the networking stack](https://www.kernel.org/doc/Documentation/networking/scaling.rst)
- [Red Hat: Network Performance Tuning](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/)
- [irqbalance(1) man page](https://linux.die.net/man/1/irqbalance)
- [Cloudflare: How to receive a million packets per second](https://blog.cloudflare.com/how-to-receive-a-million-packets/)
- [DPDK: NIC Performance Optimization](https://doc.dpdk.org/guides/nics/index.html)
