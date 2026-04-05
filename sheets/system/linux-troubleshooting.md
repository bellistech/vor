# Linux Troubleshooting

Systematic troubleshooting: boot failures, filesystem, network, services, performance, logs, SELinux.

## Systematic Methodology

### Troubleshooting Process

```bash
# 1. IDENTIFY — What is the symptom?
#    - What changed? When did it start?
#    - Can you reproduce it?
#    - What is the impact scope?

# 2. ANALYZE — Gather data
#    - Check logs (journalctl, /var/log)
#    - Check resource usage (top, free, df)
#    - Check recent changes (rpm -qa --last, git log, /etc history)

# 3. HYPOTHESIZE — Form a theory
#    - Start with most likely/simplest cause
#    - Check one variable at a time

# 4. TEST — Implement fix
#    - Make one change at a time
#    - Have a rollback plan

# 5. VERIFY — Confirm resolution
#    - Reproduce original test
#    - Check for side effects

# 6. DOCUMENT — Record findings
#    - Root cause, fix, prevention
```

## Boot Failures

### GRUB Rescue

```bash
# GRUB prompt appears when config is broken
# At grub> prompt:
ls                          # list partitions
ls (hd0,msdos1)/           # browse partition
set root=(hd0,msdos1)
set prefix=(hd0,msdos1)/boot/grub2
insmod normal
normal

# If grub rescue> prompt (minimal shell):
set prefix=(hd0,msdos1)/boot/grub2
insmod normal
normal

# Reinstall GRUB from rescue media
chroot /mnt/sysroot
grub2-install /dev/sda
grub2-mkconfig -o /boot/grub2/grub.cfg
```

### Emergency and Rescue Targets

```bash
# Boot into emergency target (minimal, no mounts)
# At GRUB: append systemd.unit=emergency.target to kernel line
# Or press 'e' at GRUB, add to linux line:
systemd.unit=emergency.target

# Boot into rescue target (single-user, mounts filesystems)
systemd.unit=rescue.target

# Break into initramfs (before root mount)
# Append to kernel line:
rd.break
# This drops to initramfs shell before root pivot
# Root FS is at /sysroot (read-only)
mount -o remount,rw /sysroot
chroot /sysroot
# Make changes...
touch /.autorelabel    # if SELinux changes needed
exit
exit

# Reset root password via rd.break
mount -o remount,rw /sysroot
chroot /sysroot
passwd root
touch /.autorelabel
exit; exit
```

### Boot Analysis

```bash
# Show boot time breakdown
systemd-analyze

# Show per-unit boot time
systemd-analyze blame

# Show critical chain
systemd-analyze critical-chain

# Plot boot chart
systemd-analyze plot > boot.svg

# Show failed units
systemctl --failed

# Check boot logs
journalctl -b       # current boot
journalctl -b -1    # previous boot
journalctl -b --priority=err  # errors only
```

## Filesystem Issues

### Filesystem Check and Repair

```bash
# Check ext4 (must be unmounted!)
umount /dev/sdb1
fsck -y /dev/sdb1

# Force check even if clean
fsck -f /dev/sdb1

# Check XFS (must be unmounted)
xfs_repair /dev/sdb1

# Dry run (no changes)
xfs_repair -n /dev/sdb1

# If XFS log is dirty, clear it (data loss risk)
xfs_repair -L /dev/sdb1

# Remount read-only for emergency
mount -o remount,ro /

# Check filesystem while mounted (read-only check)
# ext4 only:
tune2fs -l /dev/sdb1 | grep "Filesystem state"

# Force fsck on next boot
touch /forcefsck
# Or for systemd:
systemctl enable systemd-fsck-root.service
```

### Recover Space from Deleted Files

```bash
# Find processes holding deleted files
lsof +L1

# Find large deleted files still open
lsof +L1 | awk '$7 > 1000000'

# Truncate (reclaim space without killing process)
: > /proc/<PID>/fd/<FD>

# Or identify and restart the service
```

## Network Troubleshooting

### Layer-by-Layer Approach

```bash
# L1 — Physical
ip link show eth0                # check link state (UP/DOWN)
ethtool eth0                     # speed, duplex, link detected

# L2 — Data Link
ip neigh show                    # ARP table
arping -I eth0 192.168.1.1      # ARP ping

# L3 — Network
ip addr show                     # check IP config
ip route show                    # check routes
ping -c 3 192.168.1.1           # test gateway
ping -c 3 8.8.8.8               # test internet
traceroute 8.8.8.8              # trace path
mtr 8.8.8.8                     # continuous traceroute

# L4 — Transport
ss -tlnp                         # listening TCP ports
ss -ulnp                         # listening UDP ports
ss -s                            # socket statistics
ss -tnp state established        # established connections

# L7 — Application
curl -v http://example.com       # HTTP test
curl -Ik https://example.com     # HTTPS headers
dig example.com                  # DNS test
nslookup example.com             # DNS test (simple)
host example.com                 # DNS test (simple)
```

### DNS Troubleshooting

```bash
# Check resolution
dig example.com
dig @8.8.8.8 example.com        # query specific server
dig +trace example.com           # trace delegation
dig -x 8.8.8.8                  # reverse lookup

# Check local config
cat /etc/resolv.conf
cat /etc/nsswitch.conf
resolvectl status                # systemd-resolved
resolvectl query example.com

# Flush DNS cache
resolvectl flush-caches
systemd-resolve --flush-caches   # older systems
```

### Packet Capture

```bash
# tcpdump — capture on interface
tcpdump -i eth0 -n              # all traffic, no DNS resolution
tcpdump -i eth0 host 10.0.0.1   # specific host
tcpdump -i eth0 port 80          # specific port
tcpdump -i eth0 'tcp port 443 and host 10.0.0.1'
tcpdump -i eth0 -w /tmp/capture.pcap   # write to file
tcpdump -r /tmp/capture.pcap           # read file
tcpdump -i eth0 -c 100 -n             # capture 100 packets

# ss deep-dive
ss -tnp dst 10.0.0.0/8           # connections to 10.x network
ss -tnp sport = :80               # connections from port 80
ss -tnpi                          # show internal TCP info (cwnd, rtt)
```

## Service Troubleshooting

### systemctl

```bash
# Check service status
systemctl status nginx.service

# Show detailed properties
systemctl show nginx.service

# Check if enabled
systemctl is-enabled nginx.service

# List failed units
systemctl --failed

# Show dependencies
systemctl list-dependencies nginx.service

# Show reverse dependencies (what depends on this)
systemctl list-dependencies --reverse nginx.service

# Reload daemon after unit file changes
systemctl daemon-reload

# Mask (completely prevent starting)
systemctl mask nginx.service
systemctl unmask nginx.service
```

### journalctl

```bash
# Logs for specific unit
journalctl -u nginx.service

# Follow live
journalctl -u nginx.service -f

# Since last boot
journalctl -u nginx.service -b

# Since specific time
journalctl -u nginx.service --since "2024-01-01 10:00" --until "2024-01-01 12:00"

# Priority filter
journalctl -p err                # errors and above
journalctl -p warning            # warnings and above

# JSON output (for parsing)
journalctl -u nginx -o json-pretty

# Kernel messages
journalctl -k

# Disk usage
journalctl --disk-usage

# Vacuum (reclaim space)
journalctl --vacuum-size=500M
journalctl --vacuum-time=7d
```

## Performance Issues

### CPU

```bash
# Real-time process viewer
top
htop

# Load average meaning
# load avg: 1min 5min 15min
# Value = number of processes in runnable + uninterruptible state
# Compare to number of CPUs: nproc
uptime

# Per-CPU usage
mpstat -P ALL 1

# Process CPU usage
pidstat -u 1

# Top CPU consumers
ps aux --sort=-%cpu | head -20
```

### Memory

```bash
# Memory overview
free -h

# Detailed memory info
cat /proc/meminfo

# Key fields:
#   MemTotal — total physical RAM
#   MemFree — completely unused
#   MemAvailable — available for allocation (includes reclaimable cache)
#   Buffers — raw device cache
#   Cached — page cache
#   SwapTotal/SwapFree — swap usage

# Per-process memory
ps aux --sort=-%mem | head -20
pmap -x <PID>

# OOM killer logs
journalctl -k | grep -i "oom\|killed process"
dmesg | grep -i "oom\|killed process"

# OOM score (higher = more likely to be killed)
cat /proc/<PID>/oom_score
cat /proc/<PID>/oom_score_adj    # manual adjustment (-1000 to 1000)
```

### Disk I/O

```bash
# I/O statistics
iostat -xz 1

# Key columns:
#   %util   — device saturation (100% = fully busy)
#   await   — average I/O wait time (ms)
#   r_await — read wait time
#   w_await — write wait time
#   avgqu-sz — average queue length

# Per-process I/O
iotop
pidstat -d 1

# Disk space
df -h
df -i                # inode usage

# Find large files
du -sh /* 2>/dev/null | sort -rh | head -20
du -sh /var/log/* | sort -rh | head -20
```

### System-wide Performance

```bash
# vmstat (virtual memory stats)
vmstat 1
# Columns: procs, memory, swap, io, system, cpu
# Key: si/so (swap in/out), wa (I/O wait), st (steal)

# sar (historical data from sysstat)
sar -u 1 10           # CPU
sar -r 1 10           # memory
sar -b 1 10           # I/O
sar -n DEV 1 10       # network
sar -q 1 10           # load average
sar -f /var/log/sa/sa01  # historical data for day 01

# perf (performance counters)
perf top                      # live function profiling
perf record -g -p <PID> -- sleep 10   # record profile
perf report                   # analyze recording
perf stat -p <PID> -- sleep 5         # hardware counters
```

## Log Analysis

### Key Log Locations

```bash
# System logs
journalctl                  # systemd journal
/var/log/messages           # traditional syslog (RHEL)
/var/log/syslog             # traditional syslog (Debian)

# Authentication
/var/log/secure             # RHEL
/var/log/auth.log           # Debian
journalctl -u sshd

# Boot
/var/log/boot.log
journalctl -b

# Kernel
dmesg
journalctl -k

# Package management
/var/log/dnf.log
/var/log/apt/history.log

# Audit
/var/log/audit/audit.log
ausearch -m AVC             # SELinux denials
```

### Log Correlation

```bash
# Logs around a specific time
journalctl --since "2024-01-15 14:00" --until "2024-01-15 14:30"

# Multiple units at same time
journalctl -u nginx -u php-fpm --since "10 min ago"

# Grep across all logs
journalctl --no-pager | grep -i error

# Follow multiple sources
journalctl -f -u nginx -u php-fpm
```

## SELinux Troubleshooting

### Check and Diagnose

```bash
# Current mode
getenforce
sestatus

# Search for denials
ausearch -m AVC -ts recent
ausearch -m AVC --start today

# Human-readable analysis
sealert -a /var/log/audit/audit.log

# What would be denied (permissive mode audit)
semodule -l | grep permissive

# Check file contexts
ls -Z /var/www/html/
restorecon -Rv /var/www/html/

# Check process contexts
ps auxZ | grep nginx

# Check port contexts
semanage port -l | grep http

# Add port to context
semanage port -a -t http_port_t -p tcp 8080

# Check booleans
getsebool -a | grep httpd
setsebool -P httpd_can_network_connect on

# Generate policy module from denials
ausearch -m AVC -ts recent | audit2allow -M mypolicy
semodule -i mypolicy.pp
```

## Core Dumps

### coredumpctl

```bash
# List recent core dumps
coredumpctl list

# Show info about latest
coredumpctl info

# Show info for specific PID
coredumpctl info <PID>

# Debug with gdb
coredumpctl gdb <PID>

# Dump to file
coredumpctl dump <PID> -o /tmp/core.dump

# Configuration
cat /etc/systemd/coredump.conf
# Storage=external
# Compress=yes
# MaxUse=1G
# ProcessSizeMax=2G
```

## See Also

- dmesg
- systemd-units
- journalctl
- performance-tracing
- selinux

## References

- Red Hat System Administration Guide
- man journalctl, systemctl, ss, tcpdump, iostat, vmstat, sar
- Brendan Gregg: Systems Performance (USE Method)
- kernel.org: Documentation/admin-guide/
