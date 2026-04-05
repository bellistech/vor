# CPU Scheduler Tuning (Linux)

Tune the Linux CPU scheduler for throughput, latency, or real-time workloads.

## CFS — Completely Fair Scheduler

### Core Parameters

```bash
# View current CFS tunables
sysctl kernel.sched_latency_ns
sysctl kernel.sched_min_granularity_ns
sysctl kernel.sched_wakeup_granularity_ns

# Reduce scheduling latency (default 6ms, lower = more responsive)
sysctl -w kernel.sched_latency_ns=4000000

# Minimum timeslice per task (default 0.75ms)
sysctl -w kernel.sched_min_granularity_ns=500000

# Wakeup preemption granularity (lower = faster preemption)
sysctl -w kernel.sched_wakeup_granularity_ns=500000
```

### Nice Values and Weights

```bash
# Run a process at lower priority (nice 10)
nice -n 10 ./my-batch-job

# Change running process priority
renice -n -5 -p $(pidof my-service)

# View scheduling info for a process
chrt -p $(pidof my-service)
cat /proc/$(pidof my-service)/sched | grep -E 'vruntime|nr_switches'
```

## SCHED_DEADLINE

### Deadline Scheduling

```bash
# Set deadline policy: runtime=10ms, deadline=30ms, period=30ms
chrt -d --sched-runtime 10000000 --sched-deadline 30000000 \
        --sched-period 30000000 0 ./realtime-task

# View deadline parameters of a running process
chrt -p $(pidof realtime-task)

# Deadline tasks always preempt CFS and RT tasks
# Admission control: kernel rejects if total utilization > ~95%
```

## CPU Affinity

### taskset

```bash
# Run process on CPUs 0-3 only
taskset -c 0-3 ./my-app

# Pin running process to CPU 2
taskset -cp 2 $(pidof my-app)

# Show current affinity
taskset -cp $(pidof my-app)
```

### cgroups cpuset

```bash
# Create a cpuset cgroup (cgroups v2)
mkdir -p /sys/fs/cgroup/myapp
echo "0-3" > /sys/fs/cgroup/myapp/cpuset.cpus
echo "0" > /sys/fs/cgroup/myapp/cpuset.mems

# Move a process into the cpuset
echo $PID > /sys/fs/cgroup/myapp/cgroup.procs
```

### Boot-Time Isolation

```bash
# Isolate CPUs 4-7 from general scheduling (kernel boot param)
# In /etc/default/grub:
GRUB_CMDLINE_LINUX="isolcpus=4-7"

# Then:
update-grub && reboot

# Verify isolated CPUs
cat /sys/devices/system/cpu/isolated
```

## Tickless Mode (nohz_full)

```bash
# Boot param: make CPUs 4-7 tickless (no timer interrupts when single task)
# In /etc/default/grub:
GRUB_CMDLINE_LINUX="nohz_full=4-7 rcu_nocbs=4-7"

# Verify tickless status
cat /sys/devices/system/cpu/nohz_full

# Combine with isolcpus for lowest latency
GRUB_CMDLINE_LINUX="isolcpus=4-7 nohz_full=4-7 rcu_nocbs=4-7"

# Check timer interrupts per CPU
watch -n1 'cat /proc/interrupts | head -3'
```

## CPU Frequency Governor

```bash
# View current governor
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# Set performance mode (max frequency, no scaling)
echo performance | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Set powersave mode (min frequency)
echo powersave | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Use schedutil (frequency tracks scheduler load)
echo schedutil | tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# List available governors
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_available_governors

# Set min/max frequency bounds (kHz)
echo 2000000 > /sys/devices/system/cpu/cpu0/cpufreq/scaling_min_freq
echo 3500000 > /sys/devices/system/cpu/cpu0/cpufreq/scaling_max_freq

# cpupower utility
cpupower frequency-info
cpupower frequency-set -g performance
```

## Real-Time Scheduling

### SCHED_FIFO and SCHED_RR

```bash
# Run with SCHED_FIFO at priority 50
chrt -f 50 ./realtime-app

# Run with SCHED_RR (round-robin among same-priority RT tasks)
chrt -r 50 ./realtime-app

# Change policy of running process
chrt -f -p 80 $(pidof realtime-app)

# View scheduling policy
chrt -p $(pidof realtime-app)
```

### RT Throttling

```bash
# RT tasks get 95% of CPU time by default (safety valve)
sysctl kernel.sched_rt_runtime_us    # default 950000 (950ms)
sysctl kernel.sched_rt_period_us     # default 1000000 (1s)

# Disable RT throttling (dangerous — RT bug = full lockup)
sysctl -w kernel.sched_rt_runtime_us=-1

# Give RT tasks 99% of CPU time
sysctl -w kernel.sched_rt_runtime_us=990000
```

## NUMA-Aware Scheduling

```bash
# Show NUMA topology
numactl --hardware
lscpu | grep NUMA

# Run process on NUMA node 0 CPUs and memory
numactl --cpunodebind=0 --membind=0 ./my-app

# Interleave memory across all nodes (good for large shared data)
numactl --interleave=all ./my-app

# View auto-NUMA balancing status
sysctl kernel.numa_balancing

# Enable auto-NUMA (kernel migrates pages to match access patterns)
sysctl -w kernel.numa_balancing=1

# Disable if workload is already pinned
sysctl -w kernel.numa_balancing=0

# Check NUMA stats
numastat -p $(pidof my-app)
```

## Persistent Configuration

```bash
# /etc/sysctl.d/99-scheduler.conf
cat <<'EOF' > /etc/sysctl.d/99-scheduler.conf
kernel.sched_latency_ns = 4000000
kernel.sched_min_granularity_ns = 500000
kernel.sched_wakeup_granularity_ns = 500000
kernel.sched_rt_runtime_us = 950000
kernel.numa_balancing = 1
EOF

sysctl -p /etc/sysctl.d/99-scheduler.conf
```

## Tips

- Always benchmark before and after tuning; use `perf sched` and `cyclictest` to measure
- `isolcpus` + `nohz_full` + `rcu_nocbs` is the holy trinity for latency-critical cores
- SCHED_DEADLINE is preferred over SCHED_FIFO for periodic real-time tasks
- Set CPU governor to `performance` before benchmarking to eliminate frequency scaling noise
- On NUMA systems, always check `numastat` to ensure memory locality matches CPU pinning
- CFS `sched_latency_ns` tuning is a throughput vs. latency tradeoff: lower values = more context switches

## See Also

- sysctl
- process-management
- cgroups
- perf

## References

- kernel.org: Documentation/scheduler/sched-design-CFS.rst
- kernel.org: Documentation/scheduler/sched-deadline.rst
- man 7 sched — Linux scheduling overview
- man 1 chrt — manipulate real-time attributes
- man 1 taskset — set CPU affinity
- man 8 numactl — NUMA policy control
- Red Hat Performance Tuning Guide — CPU scheduling
