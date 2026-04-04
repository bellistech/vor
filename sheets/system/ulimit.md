# Ulimit (Resource Limits)

Per-process resource limits enforced by the kernel control maximum file descriptors, process counts, memory locks, core dumps, and stack sizes, preventing runaway processes from exhausting system resources.

## Viewing Current Limits

```bash
# Show all soft limits
ulimit -a

# Show all hard limits
ulimit -aH

# Show specific limits
ulimit -n    # Max open files (nofile)
ulimit -u    # Max user processes (nproc)
ulimit -l    # Max locked memory in KB (memlock)
ulimit -c    # Max core file size (core)
ulimit -s    # Max stack size in KB (stack)
ulimit -v    # Max virtual memory in KB (as)
ulimit -m    # Max resident set size (rss, advisory)
ulimit -f    # Max file size in blocks (fsize)
ulimit -t    # Max CPU time in seconds (cpu)
ulimit -q    # Max POSIX message queue bytes (msgqueue)
ulimit -e    # Max scheduling priority (nice)
ulimit -i    # Max pending signals (sigpending)

# View limits of a running process
cat /proc/$PID/limits
# Limit                     Soft Limit     Hard Limit     Units
# Max open files            1024           1048576        files
# Max processes             63432          63432          processes
# Max locked memory         8388608        8388608        bytes
```

## Setting Limits

```bash
# Set soft limit (any user can lower, raise up to hard limit)
ulimit -n 65536

# Set hard limit (only root can raise)
ulimit -Hn 1048576

# Set both soft and hard
ulimit -Sn 65536
ulimit -Hn 1048576

# Set unlimited (where applicable)
ulimit -c unlimited    # Unlimited core dumps
ulimit -l unlimited    # Unlimited locked memory

# Set in current shell and subprocesses
ulimit -n 65536
exec myapp  # Inherits the limit

# Per-process limits with prlimit (no shell restart needed)
prlimit --pid $PID --nofile=65536:1048576  # soft:hard
prlimit --pid $PID --nproc=4096
prlimit --pid $PID --memlock=unlimited

# Query a process's limits with prlimit
prlimit --pid $PID --nofile
prlimit --pid $PID --output=RESOURCE,SOFT,HARD
```

## /etc/security/limits.conf

```bash
# Persistent limits via PAM (login sessions)
# Format: <domain> <type> <item> <value>

# /etc/security/limits.conf
*               soft    nofile          65536
*               hard    nofile          1048576
*               soft    nproc           4096
*               hard    nproc           63432
root            soft    nofile          1048576
root            hard    nofile          1048576
@developers     soft    nproc           8192
nginx           soft    nofile          131072
nginx           hard    nofile          131072
*               soft    memlock         unlimited
*               hard    memlock         unlimited
*               soft    core            unlimited

# Drop-in directory (preferred on modern distros)
# /etc/security/limits.d/99-custom.conf
myapp           soft    nofile          524288
myapp           hard    nofile          524288
myapp           soft    memlock         unlimited

# Domain types:
# username      Specific user
# @groupname    All users in group
# *             All users (except root in some configs)
# root          Root user specifically
```

## Systemd Service Limits

```bash
# In unit file [Service] section
[Service]
LimitNOFILE=1048576        # Max open files
LimitNPROC=63432           # Max processes
LimitMEMLOCK=infinity      # Locked memory
LimitCORE=infinity         # Core dump size
LimitSTACK=8388608         # Stack size (bytes)
LimitAS=infinity           # Address space
LimitFSIZE=infinity        # File size
LimitCPU=infinity          # CPU time
LimitSIGPENDING=63432      # Pending signals

# Check current limits of a systemd service
systemctl show myapp.service | grep ^Limit

# Override limits for existing service (drop-in)
# /etc/systemd/system/myapp.service.d/limits.conf
[Service]
LimitNOFILE=524288

# Reload after changes
systemctl daemon-reload
systemctl restart myapp

# System-wide defaults for all services
# /etc/systemd/system.conf
[Manager]
DefaultLimitNOFILE=65536:1048576
DefaultLimitNPROC=63432
DefaultLimitMEMLOCK=infinity
```

## Common Limit Types in Detail

### nofile (Max Open Files)

```bash
# Check system-wide maximum
cat /proc/sys/fs/file-max
# 9223372036854775807

# Check currently open files system-wide
cat /proc/sys/fs/file-nr
# 12416  0  9223372036854775807
# (allocated  free  maximum)

# Count open files for a process
ls /proc/$PID/fd | wc -l
# Or more accurately:
ls -la /proc/$PID/fd/ 2>/dev/null | wc -l

# Find processes with most open files
for pid in /proc/[0-9]*/; do
  count=$(ls "$pid/fd" 2>/dev/null | wc -l)
  [ "$count" -gt 100 ] && echo "$count $(cat $pid/cmdline 2>/dev/null | tr '\0' ' ')"
done | sort -rn | head -20
```

### nproc (Max User Processes)

```bash
# Check system-wide PID max
cat /proc/sys/kernel/pid_max
# 4194304

# Count processes per user
ps -eo user --no-headers | sort | uniq -c | sort -rn

# Set PID max
echo 4194304 > /proc/sys/kernel/pid_max
# Or in sysctl.conf:
# kernel.pid_max = 4194304

# threads-max (system-wide thread limit)
cat /proc/sys/kernel/threads-max
# 126864
```

### memlock (Locked Memory)

```bash
# Required for:
# - DPDK (huge pages)
# - eBPF map memory
# - Real-time applications
# - Cryptographic key protection

# Set unlimited for eBPF / DPDK workloads
ulimit -l unlimited

# Check locked memory for a process
grep VmLck /proc/$PID/status
# VmLck:    0 kB

# Locked pages with mlock
# Pages locked via mlock() count against this limit
```

### core (Core Dump Size)

```bash
# Enable core dumps
ulimit -c unlimited

# Set core dump pattern
echo "/var/coredumps/core.%e.%p.%t" > /proc/sys/kernel/core_pattern
# %e = executable name
# %p = PID
# %t = timestamp

# Pipe to a handler (systemd-coredump)
echo "|/usr/lib/systemd/systemd-coredump %P %u %g %s %t %c %h" \
  > /proc/sys/kernel/core_pattern

# Disable core dumps for security
ulimit -c 0
echo "* hard core 0" >> /etc/security/limits.conf
```

### stack (Stack Size)

```bash
# Default stack size (usually 8MB)
ulimit -s
# 8192

# Increase for deeply recursive programs
ulimit -s 16384    # 16MB

# Unlimited stack (grows until it hits address space limits)
ulimit -s unlimited

# Check thread stack size in a running process
cat /proc/$PID/maps | grep -c stack
```

## Docker and Container Limits

```bash
# Docker applies ulimits per container
docker run --ulimit nofile=65536:131072 myimage
docker run --ulimit nproc=4096 myimage
docker run --ulimit memlock=-1:-1 myimage  # unlimited

# docker-compose.yml
# services:
#   myapp:
#     ulimits:
#       nofile:
#         soft: 65536
#         hard: 131072
#       nproc: 4096
#       memlock:
#         soft: -1
#         hard: -1

# Default Docker daemon limits
# /etc/docker/daemon.json
# {
#   "default-ulimits": {
#     "nofile": { "Name": "nofile", "Soft": 65536, "Hard": 131072 }
#   }
# }
```

## Troubleshooting

```bash
# "Too many open files" error
# 1. Check current limit
ulimit -n
# 2. Check current usage
ls /proc/$PID/fd | wc -l
# 3. Find file descriptor leaks
ls -la /proc/$PID/fd | awk '{print $NF}' | sort | uniq -c | sort -rn

# "Resource temporarily unavailable" (EAGAIN) for nproc
# Check user's process count vs limit
ps -u $(whoami) --no-headers | wc -l
ulimit -u

# "Cannot allocate memory" for memlock
grep VmLck /proc/$PID/status
ulimit -l

# Limits not applying after /etc/security/limits.conf change
# Ensure PAM module is enabled:
grep pam_limits /etc/pam.d/common-session
# session required pam_limits.so
# Must log out and back in
```

## Tips

- The soft limit is the effective limit; the hard limit is the ceiling for the soft limit
- Non-root users can lower hard limits but never raise them; only root can increase hard limits
- `prlimit` can change limits of a running process without restarting it -- invaluable for production
- For systemd services, use `LimitNOFILE=` in the unit file rather than `/etc/security/limits.conf`
- Always set both soft and hard limits; setting only soft means the hard limit stays at the old default
- `/etc/security/limits.conf` only applies to PAM login sessions, not to systemd services or cron jobs
- Use `nofile=1048576` for high-connection servers (nginx, HAProxy, databases)
- Setting `memlock=unlimited` is required for eBPF programs and DPDK applications
- Core dumps can fill disks fast; use `core_pattern` to pipe to `systemd-coredump` with size limits
- The `nproc` limit counts threads, not just processes, since Linux threads are lightweight processes
- Check `/proc/$PID/limits` to verify a running process actually has the limits you intended
- Docker `--ulimit` flags override the daemon defaults per container

## See Also

cgroups, proc-sys, oom-killer, signals

## References

- [ulimit bash builtin](https://man7.org/linux/man-pages/man1/ulimit.1p.html)
- [prlimit(1) Man Page](https://man7.org/linux/man-pages/man1/prlimit.1.html)
- [limits.conf(5) Man Page](https://man7.org/linux/man-pages/man5/limits.conf.5.html)
- [systemd Resource Control](https://www.freedesktop.org/software/systemd/man/systemd.exec.html)
- [Linux Kernel Documentation: /proc/sys/fs](https://docs.kernel.org/admin-guide/sysctl/fs.html)
