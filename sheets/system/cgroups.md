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

## Tips

- Always check whether your system runs cgroups v1 or v2 before writing automation scripts
- Use `systemd-run --scope` for quick one-off resource limits without writing unit files
- Set `memory.high` before `memory.max` to get throttling instead of OOM kills
- `Delegate=yes` in systemd unit files lets containers manage their own sub-cgroups
- Monitor PSI (Pressure Stall Information) files for early warning of resource exhaustion
- PIDs controller is your best defense against fork bombs in multi-tenant systems
- Docker `--cpus=1.5` translates to `cpu.max = 150000 100000` (quota/period in microseconds)
- Always set `memory.swap.max=0` in production cgroups to prevent unpredictable swap behavior
- Use `systemd-cgtop` like you would use `top`, but for cgroup-level resource monitoring
- Kubernetes `requests` map to cgroup minimums/weights, `limits` map to hard maximums
- The root cgroup cannot have resource limits; create child cgroups for enforcement
- Check `/proc/$PID/cgroup` to verify a process landed in the correct cgroup

## See Also

namespaces, oom-killer, ulimit, swap, proc-sys, systemd

## References

- [Linux Kernel cgroup v2 Documentation](https://docs.kernel.org/admin-guide/cgroup-v2.html)
- [Red Hat Resource Management Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_monitoring_and_updating_the_kernel/assembly_using-cgroups-v2-to-control-distribution-of-cpu-time-for-applications_managing-monitoring-and-updating-the-kernel)
- [systemd Resource Control](https://www.freedesktop.org/software/systemd/man/systemd.resource-control.html)
- [Docker Resource Constraints](https://docs.docker.com/config/containers/resource_constraints/)
- [Kubernetes Resource Management](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
