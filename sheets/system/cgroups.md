# Cgroups (Control Groups)

Linux control groups partition processes into hierarchical groups to enforce resource limits on CPU, memory, I/O, and device access, forming the backbone of container resource isolation in Docker, Kubernetes, and systemd.

## Cgroups v1 vs v2

### v1 (Legacy)

```bash
# v1 uses separate hierarchies per controller
ls /sys/fs/cgroup/
# cpu/  memory/  blkio/  devices/  freezer/  net_cls/  pids/

# Check which version is active
stat -fc %T /sys/fs/cgroup/
# tmpfs = v1, cgroup2fs = v2

# Mount a v1 controller
mount -t cgroup -o cpu none /sys/fs/cgroup/cpu
```

### v2 (Unified)

```bash
# v2 uses a single unified hierarchy
ls /sys/fs/cgroup/
# cgroup.controllers  cgroup.subtree_control  system.slice/  user.slice/

# Check available controllers
cat /sys/fs/cgroup/cgroup.controllers
# cpu io memory pids rdma misc

# Enable controllers for children
echo "+cpu +memory +io +pids" > /sys/fs/cgroup/cgroup.subtree_control
```

### Migration Check

```bash
# Determine current cgroup version
grep cgroup /proc/filesystems
# nodev  cgroup
# nodev  cgroup2

# Force v2 on boot (GRUB)
# Add to GRUB_CMDLINE_LINUX in /etc/default/grub:
# systemd.unified_cgroup_hierarchy=1
```

## CPU Controller

```bash
# --- cgroups v1 ---
# Create a CPU cgroup
mkdir /sys/fs/cgroup/cpu/myapp

# Limit to 50% of one CPU (quota/period)
echo 50000 > /sys/fs/cgroup/cpu/myapp/cpu.cfs_quota_us    # 50ms
echo 100000 > /sys/fs/cgroup/cpu/myapp/cpu.cfs_period_us  # 100ms period

# Pin relative weight (shares, default 1024)
echo 512 > /sys/fs/cgroup/cpu/myapp/cpu.shares

# Add a process
echo $PID > /sys/fs/cgroup/cpu/myapp/cgroup.procs

# --- cgroups v2 ---
# Limit to 150% CPU (1.5 cores)
echo "150000 100000" > /sys/fs/cgroup/myapp/cpu.max

# Set CPU weight (1-10000, default 100)
echo 50 > /sys/fs/cgroup/myapp/cpu.weight
```

## Memory Controller

```bash
# --- cgroups v2 ---
# Set hard memory limit (500MB)
echo 524288000 > /sys/fs/cgroup/myapp/memory.max

# Set soft limit (triggers reclaim at this point)
echo 419430400 > /sys/fs/cgroup/myapp/memory.high

# Set minimum guaranteed memory
echo 209715200 > /sys/fs/cgroup/myapp/memory.min

# Disable swap for the cgroup
echo 0 > /sys/fs/cgroup/myapp/memory.swap.max

# Check current usage
cat /sys/fs/cgroup/myapp/memory.current
cat /sys/fs/cgroup/myapp/memory.stat

# --- cgroups v1 ---
echo 524288000 > /sys/fs/cgroup/memory/myapp/memory.limit_in_bytes
echo 0 > /sys/fs/cgroup/memory/myapp/memory.swappiness
cat /sys/fs/cgroup/memory/myapp/memory.usage_in_bytes
```

## I/O Controller

```bash
# Find device major:minor numbers
lsblk -o NAME,MAJ:MIN
# sda  8:0

# --- cgroups v2 ---
# Set read/write BPS limits (device 8:0, 50MB/s read, 10MB/s write)
echo "8:0 rbps=52428800 wbps=10485760" > /sys/fs/cgroup/myapp/io.max

# Set IOPS limits
echo "8:0 riops=1000 wiops=500" > /sys/fs/cgroup/myapp/io.max

# Set proportional weight (1-10000, default 100)
echo "8:0 100" > /sys/fs/cgroup/myapp/io.weight

# Check IO stats
cat /sys/fs/cgroup/myapp/io.stat

# --- cgroups v1 ---
echo "8:0 52428800" > /sys/fs/cgroup/blkio/myapp/blkio.throttle.read_bps_device
```

## PID Controller

```bash
# Limit number of processes (fork bomb protection)
echo 100 > /sys/fs/cgroup/myapp/pids.max

# Check current count
cat /sys/fs/cgroup/myapp/pids.current
```

## Systemd Integration

```bash
# View cgroup tree via systemd
systemd-cgls

# Show resource usage per slice
systemd-cgtop

# Set CPU limit on a service
systemctl set-property myapp.service CPUQuota=50%

# Set memory limit on a service
systemctl set-property myapp.service MemoryMax=512M

# In unit file [Service] section:
# CPUQuota=200%         # 2 cores max
# MemoryMax=1G
# MemoryHigh=768M
# IODeviceWeight=/dev/sda 200
# TasksMax=100
# Delegate=yes          # allow sub-cgroups

# Create a transient scope
systemd-run --scope --slice=myslice -p MemoryMax=256M ./myapp

# Check cgroup of a running service
systemctl show myapp.service -p ControlGroup
```

## Managing Cgroups with cgroupfs

```bash
# Create a cgroup (v2)
mkdir /sys/fs/cgroup/myapp

# Add current shell to the cgroup
echo $$ > /sys/fs/cgroup/myapp/cgroup.procs

# List processes in a cgroup
cat /sys/fs/cgroup/myapp/cgroup.procs

# Remove a cgroup (must be empty)
rmdir /sys/fs/cgroup/myapp

# Check which cgroup a process belongs to
cat /proc/$PID/cgroup
# 0::/user.slice/user-1000.slice/session-1.scope

# List all controllers for a cgroup
cat /sys/fs/cgroup/myapp/cgroup.controllers
```

## Docker and Container Integration

```bash
# Run container with CPU limit (1.5 cores)
docker run --cpus=1.5 myimage

# Run container with memory limit
docker run -m 512m --memory-swap 1g myimage

# Run container with IO limits
docker run --device-read-bps /dev/sda:50mb myimage

# Run container with PID limit
docker run --pids-limit 100 myimage

# Inspect container cgroup
docker inspect --format '{{.HostConfig.CgroupParent}}' container_id

# Kubernetes resource limits map to cgroups
# resources:
#   limits:
#     cpu: "2"           -> cpu.max = 200000 100000
#     memory: "1Gi"      -> memory.max = 1073741824
#   requests:
#     cpu: "500m"         -> cpu.weight (proportional)
#     memory: "256Mi"     -> memory.min = 268435456
```

## Monitoring and Debugging

```bash
# Read memory pressure (v2 PSI)
cat /sys/fs/cgroup/myapp/memory.pressure
# some avg10=0.00 avg60=0.00 avg300=0.00 total=0
# full avg10=0.00 avg60=0.00 avg300=0.00 total=0

# Read CPU pressure
cat /sys/fs/cgroup/myapp/cpu.pressure

# Read IO pressure
cat /sys/fs/cgroup/myapp/io.pressure

# Watch for OOM events
cat /sys/fs/cgroup/myapp/memory.events
# low 0
# high 0
# max 0
# oom 0
# oom_kill 0

# Detailed memory breakdown
cat /sys/fs/cgroup/myapp/memory.stat | head -20
```

## Field Notes: Cap the Slice Instead of Running a Watchdog Script

A watcher script only observes. A cgroup limit refuses the fork. These notes come from one real incident and the cleanup after it.

### Why this section exists

- **2026-11-04, the freeze.** A dev host (14 GB RAM, 34 GB swap) froze solid: journal stopped, keyboard LEDs dead, power cycle needed. An automated shell session had defined a helper function named `g` on a host where `alias g='git'` existed. Bash expanded the alias inside the definition, so the function became `git(){ git grep ...; }`: it called itself, and every level forked a pipeline.
- **The numbers.** About 8,600 forks in the first 40 seconds, then a plateau at 17,320 tasks. That number was systemd's `DefaultTasksMax` (15% of `kernel.threads-max`) on the session scope, the only limit in the way, and it was far too high to save the box.
- **Why nothing stopped it.** The user slice had no useful task limit (the stock `TasksMax=33%` is higher still), no memory limit, `vm.overcommit_memory=1`, `kernel.panic=0`, `systemd-oomd` inactive, and 34 GB of swap to thrash through.
- **First response, a watcher script.** It sampled `ps` once a minute and dumped diagnostics above 2,000 tasks. It ran for five days: 6,885 samples, peak 1,448, never fired.
- **2026-11-09, the real fix.** A systemd drop-in capped the user slice (`TasksMax=4096`, `MemoryMax=12G`).
- **2026-11-10, the script retired.** It was redundant, and as a never-ending background job it blocked the session that started it from restarting.

### Why the watcher was the wrong tool

- It sampled every 60 seconds; the storm made 8,600 processes in 40 seconds.
- It needed to fork `ps`, `sort`, `awk` to take a sample. On a host out of tasks, the watcher cannot fork either.
- It limited nothing. The best case was a diagnostics file written just before the freeze.
- It was one more long-running process to remember to restart after every reboot.

The kernel checks `pids.max` inside `fork()` itself, so there is no sampling gap and nothing to keep running.

### The alias trap, reproduced safely

```bash
# FUNCNEST stops the recursion at 8 levels; timeout is a second belt.
timeout 10 bash -c 'shopt -s expand_aliases
alias g="git"
FUNCNEST=8
g(){ git --version; }
g'
# environment: line 4: git: maximum function nesting level exceeded (8)

# What bash actually defined:
bash -c 'shopt -s expand_aliases
alias g="git"
g(){ git --version; }
declare -F'
# declare -f git        <- the function is named git, not g
```

```bash
# Fix 1: call the binary with `command`, so the function cannot call itself
g(){ command git --version; }

# Fix 2: drop the alias before defining the function
unalias g 2>/dev/null; g(){ git --version; }

# Fix 3: do not name functions after aliases; list what is defined first
alias | cut -d= -f1

# Seatbelt for scripts and agent shells: cap nesting and wall time
FUNCNEST=64
timeout 120 some-command
```

Non-interactive bash expands aliases only with `shopt -s expand_aliases`. Tools that replay a captured interactive environment into a batch shell turn it on, which is how an interactive convenience alias reached an unattended script.

### The built-in replacement: cap the user slice

```bash
# Persistent drop-in for one user (UID 1000)
sudo mkdir -p /etc/systemd/system/user-1000.slice.d
printf '[Slice]\nTasksMax=4096\nMemoryMax=12G\n' |
  sudo tee /etc/systemd/system/user-1000.slice.d/50-caps.conf
sudo systemctl daemon-reload

# Alternative: one command, applies now, persists in /etc/systemd/system.control/
sudo systemctl set-property user-1000.slice TasksMax=4096 MemoryMax=12G

# Every user at once: use the template directory
#   /etc/systemd/system/user-.slice.d/50-caps.conf
# The distro default lives in /usr/lib/systemd/system/user-.slice.d/10-defaults.conf (TasksMax=33%)
```

Pick one of the two. `system.control/` outranks `/etc/systemd/system/`, so a leftover `set-property` file silently overrides later edits to the hand-written drop-in. List what is in force with `systemctl show user-1000.slice -p DropInPaths`.

```bash
# Verify the limit is live
systemctl show user-1000.slice -p TasksMax -p TasksCurrent -p MemoryMax -p MemoryCurrent
cat /sys/fs/cgroup/user.slice/user-1000.slice/pids.max      # 4096
cat /sys/fs/cgroup/user.slice/user-1000.slice/memory.max    # 12884901888

# Verify YOUR shell is under it (tmux, ssh, and `systemd-run --user` scopes all should be)
cat /proc/$$/cgroup
# 0::/user.slice/user-1000.slice/user@1000.service/tmux-spawn-<id>.scope
```

Limits are hierarchical and the tightest ancestor wins. A child scope can show `pids.max` 17320 while the parent slice holds it to 4096.

What the cap does not cover: anything outside `user.slice`. System services, cron jobs run by a system cron daemon, and Docker or LXD containers sit under `system.slice` or their own slices and need their own `TasksMax=` / `--pids-limit`.

### Built-in tripwire and high-water mark (replaces the log file)

```bash
S=/sys/fs/cgroup/user.slice/user-1000.slice

# How many forks has the cap refused? 0 = never hit
cat $S/pids.events            # max 0

# Highest task count the slice has ever reached (kernel 6.1+)
cat $S/pids.peak              # 285

# Same for memory: peak bytes, and how often each limit was hit
cat $S/memory.peak            # 9563959296
cat $S/memory.events          # low 0  high 0  max 0  oom 0  oom_kill 0

# The kernel logs the first refusal per cgroup
journalctl -k -g "fork rejected"
# kernel: cgroup: fork rejected by pids controller in /user.slice/user-1000.slice/...

# Live view, per cgroup (interactive, q to quit)
systemd-cgtop user.slice
```

What a hit looks like from the offending shell:

```bash
# bash: fork: retry: Resource temporarily unavailable
# bash: fork: Resource temporarily unavailable          (errno EAGAIN)
```

The rest of the host keeps running, and root (outside the slice) can still log in and kill the offender:

```bash
# DESTRUCTIVE: kills every process the user owns in that slice, your own shell included
sudo systemctl kill --signal=KILL user-1000.slice

# Narrower: one scope, as the user (find its name with `systemctl --user list-units --type=scope`)
systemctl --user kill --signal=KILL tmux-spawn-<id>.scope
```

### Sizing the numbers

```bash
# Read the real peak after a normal busy week, then set the cap to a few multiples of it
cat /sys/fs/cgroup/user.slice/user-1000.slice/pids.peak
cat /sys/fs/cgroup/user.slice/user-1000.slice/memory.peak
```

On the incident host the peak was 285 tasks and 9.6 GB, so `TasksMax=4096` is about 14x headroom (2048 would still be roomy) and `MemoryMax=12G` leaves about 2 GB for the system.

### Gaps a task cap alone leaves open

```bash
# 1. Swap is unlimited by default: a hog hits MemoryMax, then spills into swap and thrashes
systemctl show user-1000.slice -p MemorySwapMax        # MemorySwapMax=infinity
#    Fix: add to the drop-in
#    MemorySwapMax=2G          (or 0 for no swap at all)

# 2. No soft limit: nothing slows a hog down before the hard kill
#    MemoryHigh=10G            (throttle and reclaim at 10G, kill at MemoryMax=12G)

# 3. Userspace OOM handling is off
systemctl is-active systemd-oomd                       # inactive
#    systemd-oomd acts on cgroups marked ManagedOOMMemoryPressure=kill / ManagedOOMSwap=kill

# 4. A frozen kernel stays frozen
sysctl kernel.panic vm.overcommit_memory
#    kernel.panic = 0          -> never reboots after a panic; set e.g. 10, plus a hardware watchdog
#    vm.overcommit_memory = 1  -> every allocation succeeds until the OOM killer runs
```

### Which limit does what

| Mechanism | Scope | Enforced where | Notes |
|---|---|---|---|
| `TasksMax=` / `pids.max` | a cgroup and all descendants | in `fork()`/`clone()` | counts threads too; hierarchical; root is not exempt |
| `MemoryMax=` / `memory.max` | a cgroup and all descendants | on allocation | OOM kill is confined to the cgroup |
| `MemoryHigh=` / `memory.high` | same | on allocation | throttles and reclaims, does not kill |
| `MemorySwapMax=` / `memory.swap.max` | same | on swap-out | default `max` (unlimited) |
| `ulimit -u` (`RLIMIT_NPROC`) | per real UID | in `fork()` | not hierarchical, root exempt, set per login via PAM |
| `kernel.pid_max`, `kernel.threads-max` | whole host | in `fork()` | the last ceiling; hitting it means everything is already stuck |
| `systemd-oomd` | marked cgroups | userspace, on PSI pressure | kills a whole cgroup before the kernel OOM killer has to |

### When a watcher script is still worth having

Only for diagnostics, never as the limit. A cap tells you that something forked too much; a dump tells you what. The retired script, kept as a recipe for a one-shot "who is forking" snapshot:

```bash
#!/bin/bash
# Samples task counts each minute. Groups by name and parent because the leak
# was thousands of separate processes, not threads in one process.
# Dumps once above 2000 tasks and exits.
L=/var/tmp/tasks.log
while :; do
  n=$(ps -eLo pid= | wc -l); p=$(ps -eo pid= | wc -l)
  printf '%s tasks=%s procs=%s top=%s\n' "$(date -u +%H:%M:%S)" "$n" "$p" \
    "$(ps -eo comm= | sort | uniq -c | sort -rn | head -4 | awk '{printf "%s:%s ",$1,$2}')" >> "$L"
  if [ "$n" -gt 2000 ]; then
    D=/var/tmp/tasks-dump-$(date -u +%H%M%S).txt
    { echo "tasks=$n procs=$p"
      echo "== by name";   ps -eo comm= | sort | uniq -c | sort -rn | head -20
      echo "== by parent"; ps -eo ppid= | sort -n | uniq -c | sort -rn | head -12 |
        while read -r c q; do printf '%s ppid=%s %s\n' "$c" "$q" "$(ps -o args= -p "$q" 2>/dev/null | cut -c1-140)"; done
      echo "== by cgroup"; for f in /proc/[0-9]*/cgroup; do cat "$f" 2>/dev/null; done |
        sort | uniq -c | sort -rn | head -8 | cut -c1-140
      echo "== oldest and newest args"; ps -eo pid,ppid,etimes,rss,args --sort=-etimes | tail -60 | cut -c1-180
    } > "$D"
    echo "TASKS=$n dump=$D"; exit 0
  fi
  sleep 60
done
```

If it must run, run it as a unit with its own small cap rather than as a background job tied to a login session:

```bash
systemd-run --user --unit=task-watch -p TasksMax=64 -p MemoryMax=64M /path/to/task-watch.sh
systemctl --user stop task-watch
```

### Writing notes like these: the heredoc trap

```bash
# BROKEN: the outer heredoc ends at the FIRST line that is exactly EOF,
# so everything after the inner EOF runs as shell commands (sudo lines included)
cat > notes.md <<'EOF'
sudo tee /etc/example.conf <<'EOF'
key=value
EOF
sudo systemctl daemon-reload
EOF

# FIXED: give the outer heredoc a delimiter that cannot appear in the text
cat > notes.md <<'NOTES_END_7f3a'
sudo tee /etc/example.conf <<'EOF'
key=value
EOF
sudo systemctl daemon-reload
NOTES_END_7f3a
```

This bit while this very section was being written: two `sudo systemctl` lines from the draft ran for real. Write documentation that contains shell with an editor, not through a heredoc.

## Tips

- Always check whether your system runs cgroups v1 or v2 before writing automation scripts
- Use `systemd-run --scope` for quick one-off resource limits without writing unit files
- Set `memory.high` before `memory.max` to get throttling instead of OOM kills
- `Delegate=yes` in systemd unit files lets containers manage their own sub-cgroups
- Monitor PSI (Pressure Stall Information) files for early warning of resource exhaustion
- PIDs controller is your best defense against fork bombs in multi-tenant systems
- `pids.events` and `pids.peak` are a free tripwire and high-water mark; no watcher script needed
- `MemoryMax=` without `MemorySwapMax=` lets a hog thrash through swap instead of being killed
- Docker `--cpus=1.5` translates to `cpu.max = 150000 100000` (quota/period in microseconds)
- Always set `memory.swap.max=0` in production cgroups to prevent unpredictable swap behavior
- Use `systemd-cgtop` like you would use `top`, but for cgroup-level resource monitoring
- Kubernetes `requests` map to cgroup minimums/weights, `limits` map to hard maximums
- The root cgroup cannot have resource limits; create child cgroups for enforcement
- Check `/proc/$PID/cgroup` to verify a process landed in the correct cgroup

## See Also

namespaces, oom-killer, ulimit, swap, proc-sys, systemd, bash, journalctl

## References

- [Linux Kernel cgroup v2 Documentation](https://docs.kernel.org/admin-guide/cgroup-v2.html)
- [Red Hat Resource Management Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_monitoring_and_updating_the_kernel/assembly_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel)
- [systemd Resource Control](https://www.freedesktop.org/software/systemd/man/systemd.resource-control.html)
- [Docker Resource Constraints](https://docs.docker.com/config/containers/resource_constraints/)
- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
