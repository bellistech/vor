# OOM-Killer (Out-of-Memory Killer)

The Linux OOM killer is the kernel's last-resort mechanism for recovering from memory exhaustion, selecting and terminating processes based on a scoring algorithm that balances memory usage, process importance, and administrator overrides.

## OOM Score Basics

```bash
# View OOM score of a process (0-2000, higher = more likely to be killed)
cat /proc/$PID/oom_score

# View OOM score adjustment (-1000 to 1000)
cat /proc/$PID/oom_score_adj

# View all processes sorted by OOM score
for pid in /proc/[0-9]*/; do
  score=$(cat "${pid}oom_score" 2>/dev/null)
  adj=$(cat "${pid}oom_score_adj" 2>/dev/null)
  name=$(cat "${pid}comm" 2>/dev/null)
  [ -n "$score" ] && echo "$score $adj $name $(basename $pid)"
done | sort -rn | head -20

# One-liner: top OOM candidates
ps -eo pid,comm,%mem --sort=-%mem | head -20
```

## Adjusting OOM Score

```bash
# Protect a critical process from OOM killer
echo -1000 > /proc/$PID/oom_score_adj
# -1000 = OOM_SCORE_ADJ_MIN (never kill)

# Make a process the first OOM victim
echo 1000 > /proc/$PID/oom_score_adj

# Moderate adjustment (slightly favor killing)
echo 500 > /proc/$PID/oom_score_adj

# Moderate protection (slightly protect)
echo -500 > /proc/$PID/oom_score_adj

# Protect from OOM via systemd service
# [Service]
# OOMScoreAdjust=-900

# From command line for a running service
systemctl set-property myapp.service OOMScoreAdjust=-500

# Apply adjustment at process start
nice -n 0 choom -n -500 -- /usr/bin/myapp

# Check adjustment was applied
choom -p $PID
```

## Memory Overcommit Control

```bash
# Check current overcommit mode
cat /proc/sys/vm/overcommit_memory
# 0 = Heuristic (default) — kernel guesses if allocation is safe
# 1 = Always overcommit — never refuse malloc (dangerous)
# 2 = Never overcommit — strict accounting

# Set overcommit mode
echo 0 > /proc/sys/vm/overcommit_memory
# Persistent:
echo "vm.overcommit_memory=0" >> /etc/sysctl.d/99-oom.conf

# Check overcommit ratio (used with mode 2)
cat /proc/sys/vm/overcommit_ratio
# 50 (default: 50% of RAM)

# Check overcommit kbytes (alternative to ratio)
cat /proc/sys/vm/overcommit_kbytes
# 0 (0 means use overcommit_ratio instead)

# View commit limit and current committed memory
cat /proc/meminfo | grep -i commit
# CommitLimit:     12288000 kB   (max allowed commitment)
# Committed_AS:     8192000 kB   (currently committed)

# CommitLimit = (overcommit_ratio / 100) * RAM + Swap
# With 16GB RAM, 4GB swap, 50% ratio:
# CommitLimit = 0.5 * 16 + 4 = 12 GB
```

## Cgroup OOM Control

### Cgroups v2

```bash
# Set memory limit (triggers cgroup-level OOM)
echo 512M > /sys/fs/cgroup/myapp/memory.max

# Monitor OOM events
cat /sys/fs/cgroup/myapp/memory.events
# low 0
# high 12
# max 3
# oom 1
# oom_kill 1
# oom_group_kill 0

# Enable cgroup-wide OOM kill (kill all processes in cgroup)
echo 1 > /sys/fs/cgroup/myapp/memory.oom.group

# Set OOM priority for a cgroup (lower = less likely to be killed)
# This is done via memory.low and memory.min protections
echo 256M > /sys/fs/cgroup/myapp/memory.min    # Guaranteed minimum
echo 384M > /sys/fs/cgroup/myapp/memory.low    # Best-effort minimum

# Watch for OOM kills in real time
dmesg -w | grep -i "oom\|killed"
```

### Cgroups v1

```bash
# OOM control in v1
cat /sys/fs/cgroup/memory/myapp/memory.oom_control
# oom_kill_disable 0
# under_oom 0

# Disable OOM killer for this cgroup (pauses processes instead)
echo 1 > /sys/fs/cgroup/memory/myapp/memory.oom_control

# WARNING: Disabling OOM in v1 can deadlock the system
# Paused processes hold locks that other processes need
```

## Docker and Kubernetes OOM

```bash
# Docker: set memory limit (OOM kills container if exceeded)
docker run -m 512m myimage

# Docker: check if container was OOM killed
docker inspect --format '{{.State.OOMKilled}}' container_id
# true

# Docker events showing OOM
docker events --filter event=oom

# Kubernetes: resource limits trigger cgroup OOM
# spec:
#   containers:
#   - resources:
#       limits:
#         memory: "512Mi"    # Hard limit -> memory.max
#       requests:
#         memory: "256Mi"    # Used for scheduling + memory.min

# Kubernetes OOM killed pods
kubectl get pods | grep OOMKilled
kubectl describe pod mypod | grep -A5 "Last State"
#   Last State:   Terminated
#     Reason:     OOMKilled
#     Exit Code:  137

# Kubernetes QoS classes affect OOM priority
# Guaranteed (requests == limits)  -> oom_score_adj = -997
# Burstable  (requests < limits)   -> oom_score_adj = 2..999
# BestEffort (no requests/limits)  -> oom_score_adj = 1000
```

## Earlyoom (Userspace OOM Prevention)

```bash
# Install earlyoom
apt install earlyoom    # Debian/Ubuntu
dnf install earlyoom    # Fedora/RHEL

# Start earlyoom
systemctl enable --now earlyoom

# Configuration: /etc/default/earlyoom
# EARLYOOM_ARGS="-m 5 -s 10 --prefer '(^|/)(java|python3)$' --avoid '(^|/)(sshd|systemd)$'"

# Flags:
# -m 5       Kill when available memory < 5%
# -s 10      Kill when swap < 10%
# -r 60      Check every 60 seconds (default: 1)
# --prefer   Regex: prefer killing matching processes
# --avoid    Regex: avoid killing matching processes
# -n         Use SIGTERM instead of SIGKILL (let process clean up)

# Run manually for testing
earlyoom -m 10 -s 20 --dryrun

# Check earlyoom logs
journalctl -u earlyoom -f
```

## systemd-oomd

```bash
# systemd-oomd (systemd 248+): cgroup-aware OOM daemon
systemctl enable --now systemd-oomd

# Configuration: /etc/systemd/oomd.conf
# [OOM]
# SwapUsedLimit=90%
# DefaultMemoryPressureLimit=60%
# DefaultMemoryPressureDurationUSec=30s

# Per-service OOM policy
# [Service]
# ManagedOOMSwap=kill           # Kill when swap pressure high
# ManagedOOMMemoryPressure=kill # Kill when memory pressure high
# ManagedOOMMemoryPressureLimit=80%

# Monitor systemd-oomd decisions
oomctl
journalctl -u systemd-oomd -f
```

## Monitoring and Debugging OOM

```bash
# Check if OOM has occurred
dmesg | grep -i "oom\|killed process\|out of memory"

# Typical OOM kill log
# Out of memory: Killed process 1234 (myapp) total-vm:8192000kB,
# anon-rss:4096000kB, file-rss:102400kB, shmem-rss:0kB,
# oom_score_adj:0

# Watch for OOM events
journalctl -k | grep -i oom

# Monitor memory pressure (cgroups v2 PSI)
cat /proc/pressure/memory
# some avg10=5.00 avg60=3.00 avg300=1.00 total=500000
# full avg10=1.00 avg60=0.50 avg300=0.20 total=100000

# Set up PSI-based alerting
# Write poll(2) on /proc/pressure/memory with threshold
# some 150000 1000000  (150ms stall in any 1s window)

# Memory usage tracking
watch -n 1 'free -h && echo "---" && cat /proc/meminfo | grep -E "MemAvail|Commit|Active\(anon\)"'

# Find memory-hungry processes
ps aux --sort=-%mem | head -10

# Check vm.panic_on_oom
cat /proc/sys/vm/panic_on_oom
# 0 = kill process (default)
# 1 = kernel panic on OOM
# 2 = kernel panic on OOM (even for cgroup OOM)
```

## OOM Prevention Strategies

```bash
# 1. Set memory limits on all services
systemctl set-property myapp.service MemoryMax=2G MemoryHigh=1536M

# 2. Use overcommit mode 2 for strict accounting
echo 2 > /proc/sys/vm/overcommit_memory
echo 80 > /proc/sys/vm/overcommit_ratio  # Allow 80% of RAM + swap

# 3. Reserve memory for critical processes
echo -900 > /proc/$(pgrep sshd)/oom_score_adj
echo -900 > /proc/$(pgrep systemd | head -1)/oom_score_adj

# 4. Monitor and alert on memory trends
cat /proc/meminfo | awk '/MemAvailable/{avail=$2} /MemTotal/{total=$2}
  END{pct=avail*100/total; if(pct<10) print "WARNING: "pct"% available"}'

# 5. Configure swap as a buffer
# Provides time between "running low" and "OOM"
swapon --show
# Ensure swap is at least 25% of RAM

# 6. Use earlyoom or systemd-oomd as proactive killers
# They kill before the kernel OOM triggers, with better selection
```

## Tips

- The OOM killer is a last resort; if it triggers, your system is already in a degraded state
- Set `oom_score_adj=-1000` on critical daemons like sshd and your init process
- Use `memory.high` (cgroups v2) to throttle processes before they hit `memory.max` and get OOM killed
- `memory.oom.group=1` kills all processes in a cgroup together, mimicking container-level OOM
- Docker exit code 137 means the container received SIGKILL, usually from OOM (128 + 9 = 137)
- Use `earlyoom` on desktop/development machines to prevent complete system freezes under memory pressure
- `systemd-oomd` is preferred over earlyoom on systemd-based servers because it is cgroup-aware
- Overcommit mode 2 prevents OOM entirely but causes `malloc()` to fail, which many programs handle poorly
- PSI (Pressure Stall Information) memory metrics are the best early warning system for impending OOM
- Kubernetes BestEffort pods (no resource limits) are always first to be OOM killed
- Check `dmesg` after any unexplained process death; the OOM log includes the full scoring breakdown
- The OOM killer prioritizes the process using the most memory that will free the most resources

## See Also

cgroups, swap, proc-sys, ulimit

## References

- [Linux Kernel OOM Documentation](https://docs.kernel.org/admin-guide/mm/concepts.html)
- [proc(5): oom_score_adj](https://man7.org/linux/man-pages/man5/proc.5.html)
- [systemd-oomd](https://www.freedesktop.org/software/systemd/man/systemd-oomd.service.html)
- [earlyoom GitHub](https://github.com/rfjakob/earlyoom)
- [Kubernetes OOM Behavior](https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/)
- [LWN.net: Taming the OOM Killer](https://lwn.net/Articles/317814/)
