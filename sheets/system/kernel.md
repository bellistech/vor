# Kernel (Linux Kernel Management and Tuning)

Manage kernel modules, tune runtime parameters, configure boot options, and optimize system performance at the kernel level.

## Kernel Information

### Version and Build

```bash
# Show kernel version
uname -r
# 6.8.0-45-generic

# Full system info (kernel, hostname, arch, OS)
uname -a

# Detailed kernel build info
cat /proc/version

# Show kernel command line used at boot
cat /proc/cmdline
# BOOT_IMAGE=/vmlinuz-6.8.0-45-generic root=UUID=... ro quiet splash

# List installed kernels (Debian/Ubuntu)
dpkg --list | grep linux-image

# List installed kernels (RHEL/Fedora)
rpm -qa | grep kernel-core
```

## Kernel Modules

### Listing and Information

```bash
# List all loaded modules
lsmod

# Show info about a module (description, parameters, dependencies)
modinfo e1000e

# Show module parameters currently in use
systool -v -m e1000e

# List available parameters for a module
modinfo -p e1000e

# Check if a specific module is loaded
lsmod | grep br_netfilter
```

### Loading and Unloading

```bash
# Load a module
sudo modprobe br_netfilter

# Load with parameters
sudo modprobe bonding mode=4 miimon=100

# Unload a module (fails if in use)
sudo modprobe -r br_netfilter

# Force unload (dangerous — can crash system)
sudo rmmod -f module_name
```

### Persist Modules Across Reboot

```bash
# Load module at boot — create a .conf file in /etc/modules-load.d/
echo "br_netfilter" | sudo tee /etc/modules-load.d/br_netfilter.conf

# Load multiple modules at boot
cat <<'EOF' | sudo tee /etc/modules-load.d/kubernetes.conf
br_netfilter
overlay
ip_vs
ip_vs_rr
ip_vs_wrr
ip_vs_sh
EOF

# Set module parameters at boot — create a .conf file in /etc/modprobe.d/
echo "options bonding mode=4 miimon=100" | sudo tee /etc/modprobe.d/bonding.conf

# Blacklist a module (prevent loading)
echo "blacklist nouveau" | sudo tee /etc/modprobe.d/blacklist-nouveau.conf

# Blacklist AND prevent alias loading
cat <<'EOF' | sudo tee /etc/modprobe.d/blacklist-nouveau.conf
blacklist nouveau
options nouveau modeset=0
EOF

# Rebuild initramfs after blacklisting (Debian/Ubuntu)
sudo update-initramfs -u

# Rebuild initramfs after blacklisting (RHEL/Fedora)
sudo dracut --force
```

## Sysctl — Runtime Kernel Parameters

### Reading Parameters

```bash
# Show all current parameters
sysctl -a

# Read a specific parameter
sysctl net.ipv4.ip_forward
# net.ipv4.ip_forward = 0

# Read from /proc/sys directly
cat /proc/sys/net/ipv4/ip_forward
```

### Setting Parameters (Runtime — Lost on Reboot)

```bash
# Enable IP forwarding
sudo sysctl -w net.ipv4.ip_forward=1

# Write directly to /proc/sys
echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward

# Apply multiple at once
sudo sysctl -w net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1
```

### Persist Sysctl Across Reboot

```bash
# Create a .conf file in /etc/sysctl.d/ (loaded at boot in alphabetical order)
cat <<'EOF' | sudo tee /etc/sysctl.d/99-custom.conf
# Network forwarding
net.ipv4.ip_forward = 1
net.ipv6.conf.all.forwarding = 1

# Disable ICMP redirects
net.ipv4.conf.all.send_redirects = 0
net.ipv4.conf.all.accept_redirects = 0

# SYN flood protection
net.ipv4.tcp_syncookies = 1
net.ipv4.tcp_max_syn_backlog = 4096
EOF

# Apply immediately without reboot
sudo sysctl --system
# or apply a specific file
sudo sysctl -p /etc/sysctl.d/99-custom.conf

# Files are loaded in this order (later overrides earlier):
#   /usr/lib/sysctl.d/*.conf      — vendor defaults
#   /run/sysctl.d/*.conf          — runtime overrides
#   /etc/sysctl.d/*.conf          — admin overrides
#   /etc/sysctl.conf              — legacy (still works)
```

## Network Tuning

### TCP Stack

```bash
# Persist network tuning across reboot
cat <<'EOF' | sudo tee /etc/sysctl.d/60-network-tuning.conf
# --- TCP Buffer Sizes ---
# Min, default, max (bytes) for receive/send buffers
net.core.rmem_max = 16777216
net.core.wmem_max = 16777216
net.ipv4.tcp_rmem = 4096 87380 16777216
net.ipv4.tcp_wmem = 4096 65536 16777216

# --- TCP Performance ---
net.ipv4.tcp_window_scaling = 1
net.ipv4.tcp_timestamps = 1
net.ipv4.tcp_sack = 1
net.ipv4.tcp_no_metrics_save = 1
net.ipv4.tcp_moderate_rcvbuf = 1

# --- Connection Handling ---
net.core.somaxconn = 65535
net.core.netdev_max_backlog = 65536
net.ipv4.tcp_max_tw_buckets = 2000000
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 10
net.ipv4.tcp_slow_start_after_idle = 0

# --- Keepalive (detect dead connections faster) ---
net.ipv4.tcp_keepalive_time = 60
net.ipv4.tcp_keepalive_intvl = 10
net.ipv4.tcp_keepalive_probes = 6

# --- Congestion Control ---
# BBR requires kernel 4.9+ and fq qdisc
net.ipv4.tcp_congestion_control = bbr
net.core.default_qdisc = fq
EOF

sudo sysctl -p /etc/sysctl.d/60-network-tuning.conf
```

### Conntrack (Connection Tracking)

```bash
cat <<'EOF' | sudo tee /etc/sysctl.d/61-conntrack.conf
# Increase conntrack table size (default often 65536)
net.netfilter.nf_conntrack_max = 524288

# Reduce timeout for established connections (default 432000 = 5 days)
net.netfilter.nf_conntrack_tcp_timeout_established = 86400

# Hash table size (set in modprobe, not sysctl)
EOF

# Conntrack hash table size must be set as a module parameter
echo "options nf_conntrack hashsize=131072" | sudo tee /etc/modprobe.d/conntrack.conf

# Check current conntrack usage
cat /proc/sys/net/netfilter/nf_conntrack_count
cat /proc/sys/net/netfilter/nf_conntrack_max
conntrack -C
```

## Memory Tuning

### Swap and Cache

```bash
cat <<'EOF' | sudo tee /etc/sysctl.d/62-memory-tuning.conf
# --- Swappiness ---
# 0 = avoid swap, 100 = aggressively swap
# Desktop: 10-30, Server: 1-10, Database: 1
vm.swappiness = 10

# --- VFS Cache Pressure ---
# How aggressively kernel reclaims inode/dentry cache
# Lower = keep more filesystem metadata in memory
vm.vfs_cache_pressure = 50

# --- Dirty Page Writeback ---
# Percentage of RAM that can be dirty before background writeback starts
vm.dirty_background_ratio = 5
# Percentage of RAM that can be dirty before processes are forced to write
vm.dirty_ratio = 10
# Centiseconds between writeback runs
vm.dirty_writeback_centisecs = 500

# --- Overcommit ---
# 0 = heuristic (default), 1 = always allow, 2 = strict (use overcommit_ratio)
vm.overcommit_memory = 0
vm.overcommit_ratio = 50
EOF

sudo sysctl -p /etc/sysctl.d/62-memory-tuning.conf
```

### Hugepages

```bash
# Allocate 1024 hugepages (each 2MB = 2GB total) at runtime
sudo sysctl -w vm.nr_hugepages=1024

# Persist hugepages across reboot
echo "vm.nr_hugepages = 1024" | sudo tee /etc/sysctl.d/63-hugepages.conf

# Check hugepage status
grep -i huge /proc/meminfo
# HugePages_Total:    1024
# HugePages_Free:     1024
# Hugepagesize:       2048 kB

# 1GB hugepages must be set at boot (GRUB cmdline)
# hugepagesz=1G hugepages=4

# Mount hugetlbfs for applications to use
echo "hugetlbfs /dev/hugepages hugetlbfs defaults 0 0" | sudo tee -a /etc/fstab
sudo mount -a

# Transparent hugepages (THP) — disable for databases
echo never | sudo tee /sys/kernel/mm/transparent_hugepage/enabled
# Persist THP disable via systemd service or kernel cmdline:
# transparent_hugepage=never
```

### OOM Killer

```bash
# Check OOM score of a process (higher = more likely to be killed)
cat /proc/$(pidof mysqld)/oom_score

# Adjust OOM score (-1000 to 1000, lower = less likely to kill)
echo -1000 | sudo tee /proc/$(pidof mysqld)/oom_adj  # legacy
echo -1000 | sudo tee /proc/$(pidof mysqld)/oom_score_adj  # modern

# Persist OOM protection via systemd unit
# [Service]
# OOMScoreAdjust=-1000

# Trigger OOM killer manually (emergency — kills a process)
echo f | sudo tee /proc/sysrq-trigger

# Panic on OOM instead of killing processes (use for debugging)
echo "vm.panic_on_oom = 1" | sudo tee /etc/sysctl.d/64-oom.conf
```

## I/O Tuning

### I/O Schedulers

```bash
# Check current scheduler for a device
cat /sys/block/sda/queue/scheduler
# [mq-deadline] kyber bfq none

# Change scheduler at runtime
echo kyber | sudo tee /sys/block/sda/queue/scheduler

# Persist via udev rule (survives reboot)
cat <<'EOF' | sudo tee /etc/udev/rules.d/60-io-scheduler.rules
# SSD/NVMe — none (passthrough) or kyber
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="0", ATTR{queue/scheduler}="none"
ACTION=="add|change", KERNEL=="nvme[0-9]*", ATTR{queue/scheduler}="none"

# HDD — mq-deadline or bfq
ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/rotational}=="1", ATTR{queue/scheduler}="mq-deadline"
EOF

sudo udevadm control --reload-rules && sudo udevadm trigger
```

### Readahead and Queue Depth

```bash
# Check readahead (in 512-byte sectors)
cat /sys/block/sda/queue/read_ahead_kb
# 128 (default)

# Increase readahead for sequential workloads (HDD)
echo 2048 | sudo tee /sys/block/sda/queue/read_ahead_kb

# Persist via udev
echo 'ACTION=="add|change", KERNEL=="sd[a-z]", ATTR{queue/read_ahead_kb}="2048"' | \
  sudo tee /etc/udev/rules.d/61-readahead.rules

# Adjust nr_requests (I/O queue depth)
echo 256 | sudo tee /sys/block/sda/queue/nr_requests

# Check and tune I/O stats
cat /sys/block/sda/stat
iostat -x 1
```

## CPU Tuning

### CPU Frequency Governors

```bash
# Check current governor
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_governor

# List available governors
cat /sys/devices/system/cpu/cpu0/cpufreq/scaling_available_governors
# performance powersave schedutil ondemand conservative

# Set governor at runtime
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor

# Persist via systemd tmpfiles (survives reboot)
echo 'w /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor - - - - performance' | \
  sudo tee /etc/tmpfiles.d/cpu-governor.conf

# Or persist via kernel cmdline (GRUB)
# intel_pstate=disable cpufreq.default_governor=performance
```

### CPU Isolation and Pinning

```bash
# Isolate CPUs from scheduler (boot parameter)
# Add to GRUB_CMDLINE_LINUX in /etc/default/grub:
# isolcpus=4-7 nohz_full=4-7 rcu_nocbs=4-7

# Pin a process to specific CPUs
taskset -c 4-7 ./my-realtime-app

# Pin an existing process
taskset -pc 4-7 $(pidof my-realtime-app)

# Set CPU affinity via systemd
# [Service]
# CPUAffinity=4 5 6 7

# Check NUMA topology
numactl --hardware
lscpu | grep NUMA

# Run process on specific NUMA node
numactl --cpunodebind=0 --membind=0 ./my-app
```

## Boot Parameters (GRUB)

### Editing Kernel Command Line

```bash
# Edit GRUB defaults
sudo vim /etc/default/grub

# Common performance/tuning parameters:
GRUB_CMDLINE_LINUX="transparent_hugepage=never hugepagesz=2M hugepages=1024 isolcpus=4-7 nohz_full=4-7 rcu_nocbs=4-7 intel_iommu=on iommu=pt"

# Apply changes
sudo update-grub                    # Debian/Ubuntu
sudo grub2-mkconfig -o /boot/grub2/grub.cfg  # RHEL/Fedora

# One-time boot parameter test (at GRUB menu, press 'e', edit, Ctrl-X to boot)
```

### Common Boot Parameters

```bash
# Hugepages (1GB pages must be set here)
hugepagesz=1G hugepages=4

# CPU isolation for real-time workloads
isolcpus=4-7 nohz_full=4-7 rcu_nocbs=4-7

# IOMMU for PCI passthrough / VFIO
intel_iommu=on iommu=pt       # Intel
amd_iommu=on iommu=pt         # AMD

# Disable transparent hugepages (databases)
transparent_hugepage=never

# Kernel lockdown (security)
lockdown=confidentiality       # strictest
lockdown=integrity             # moderate

# Disable CPU mitigations (benchmarking ONLY — insecure)
mitigations=off

# Serial console (headless servers)
console=ttyS0,115200n8 console=tty0

# Memory limit (testing)
mem=4G

# Disable kernel module loading after boot
modules_disabled=1
```

## /proc and /sys Exploration

### Key /proc Files

```bash
# CPU info
cat /proc/cpuinfo | grep "model name" | head -1
nproc                      # number of CPUs

# Memory
cat /proc/meminfo
free -h

# Kernel parameters (same as sysctl -a)
ls /proc/sys/

# Process info
cat /proc/$(pidof nginx)/status    # memory, threads, capabilities
cat /proc/$(pidof nginx)/limits    # resource limits
cat /proc/$(pidof nginx)/fd        # open file descriptors
cat /proc/$(pidof nginx)/maps      # memory mappings

# System
cat /proc/uptime
cat /proc/loadavg
cat /proc/interrupts
cat /proc/softirqs
```

### Key /sys Paths

```bash
# Block devices (I/O scheduler, queue depth, rotational)
ls /sys/block/sda/queue/

# CPU (frequency, governor, topology)
ls /sys/devices/system/cpu/cpu0/cpufreq/

# Network devices
ls /sys/class/net/eth0/

# Power management
cat /sys/power/state

# Hugepages
ls /sys/kernel/mm/hugepages/

# NUMA
ls /sys/devices/system/node/
```

## Security Hardening

### Persist Security Parameters

```bash
cat <<'EOF' | sudo tee /etc/sysctl.d/90-security.conf
# --- Network Security ---
# Disable IP forwarding (unless routing)
net.ipv4.ip_forward = 0

# Ignore ICMP broadcasts (smurf attack prevention)
net.ipv4.icmp_echo_ignore_broadcasts = 1

# Ignore bogus ICMP errors
net.ipv4.icmp_ignore_bogus_error_responses = 1

# Reverse path filtering (anti-spoofing)
net.ipv4.conf.all.rp_filter = 1
net.ipv4.conf.default.rp_filter = 1

# Disable source routing
net.ipv4.conf.all.accept_source_route = 0
net.ipv6.conf.all.accept_source_route = 0

# Disable ICMP redirects
net.ipv4.conf.all.accept_redirects = 0
net.ipv4.conf.all.send_redirects = 0
net.ipv6.conf.all.accept_redirects = 0

# Log martian packets
net.ipv4.conf.all.log_martians = 1

# --- Kernel Security ---
# Restrict dmesg to root
kernel.dmesg_restrict = 1

# Restrict /proc/kallsyms
kernel.kptr_restrict = 2

# Restrict ptrace (0=all, 1=parent, 2=admin, 3=nobody)
kernel.yama.ptrace_scope = 2

# ASLR (0=off, 1=partial, 2=full)
kernel.randomize_va_space = 2

# Restrict unprivileged user namespaces
kernel.unprivileged_userns_clone = 0

# Restrict BPF
kernel.unprivileged_bpf_disabled = 1
net.core.bpf_jit_harden = 2

# Restrict perf events
kernel.perf_event_paranoid = 3

# SysRq (0=disable, 1=all, or bitmask)
# 176 = sync + remount-ro + reboot only
kernel.sysrq = 176

# --- Filesystem ---
# Restrict hardlinks and symlinks
fs.protected_hardlinks = 1
fs.protected_symlinks = 1
fs.protected_fifos = 2
fs.protected_regular = 2

# Increase file descriptor limits
fs.file-max = 2097152
fs.inotify.max_user_watches = 524288
EOF

sudo sysctl --system
```

## Building Kernels

### From Source

```bash
# Install build dependencies (Debian/Ubuntu)
sudo apt install build-essential libncurses-dev bison flex libssl-dev libelf-dev

# Download and extract
wget https://cdn.kernel.org/pub/linux/kernel/v6.x/linux-6.8.tar.xz
tar xf linux-6.8.tar.xz && cd linux-6.8

# Start from current config
cp /boot/config-$(uname -r) .config
make olddefconfig              # accept defaults for new options

# Interactive configuration
make menuconfig                # ncurses UI
make nconfig                   # newer ncurses UI

# Build
make -j$(nproc)
make modules -j$(nproc)

# Install
sudo make modules_install
sudo make install
sudo update-grub
```

### DKMS (Dynamic Kernel Module Support)

```bash
# Install DKMS
sudo apt install dkms          # Debian/Ubuntu
sudo dnf install dkms          # Fedora

# Register a module with DKMS
sudo dkms add -m module-name -v 1.0

# Build for current kernel
sudo dkms build -m module-name -v 1.0

# Install (auto-rebuilds on kernel updates)
sudo dkms install -m module-name -v 1.0

# Check DKMS status
dkms status
```

## Tips

- Always use `/etc/sysctl.d/XX-name.conf` files instead of editing `/etc/sysctl.conf` — they're modular and the number prefix controls load order
- Use `/etc/modules-load.d/` for persistent module loading, not `/etc/modules` (legacy)
- Test sysctl changes at runtime with `sysctl -w` before making them persistent
- After changing GRUB parameters, always run `update-grub` or `grub2-mkconfig`
- Rebuild initramfs after blacklisting modules: `update-initramfs -u` (Debian) or `dracut --force` (RHEL)
- Use `sysctl --system` to reload all conf files without rebooting
- For database servers: disable THP, set `vm.swappiness=1`, tune dirty page ratios
- For high-traffic network servers: increase `somaxconn`, `netdev_max_backlog`, and TCP buffer sizes
- CPU governor `performance` gives consistent latency; `schedutil` saves power with acceptable overhead
- Use `udev` rules to persist `/sys` changes — `tmpfiles.d` works too but udev is more flexible
- The `isolcpus` boot parameter is permanent per boot; use `cset` for dynamic CPU shielding

## References

- [Kernel Parameters Documentation](https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html)
- [Sysctl Documentation](https://www.kernel.org/doc/html/latest/admin-guide/sysctl/)
- [sysctl(8) Man Page](https://man7.org/linux/man-pages/man8/sysctl.8.html)
- [sysctl.conf(5) Man Page](https://man7.org/linux/man-pages/man5/sysctl.conf.5.html)
- [modprobe(8) Man Page](https://man7.org/linux/man-pages/man8/modprobe.8.html)
- [modules-load.d(5) Man Page](https://man7.org/linux/man-pages/man5/modules-load.d.5.html)
- [modprobe.d(5) Man Page](https://man7.org/linux/man-pages/man5/modprobe.d.5.html)
- [Arch Wiki — Sysctl](https://wiki.archlinux.org/title/Sysctl)
- [Arch Wiki — Kernel Modules](https://wiki.archlinux.org/title/Kernel_module)
- [Red Hat — Kernel Administration Guide](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_monitoring_and_updating_the_kernel/)
- [Red Hat — Performance Tuning](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/monitoring_and_managing_system_status_and_performance/)
- [Brendan Gregg — Linux Performance](https://www.brendangregg.com/linuxperf.html)
