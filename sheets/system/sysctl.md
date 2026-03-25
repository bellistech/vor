# sysctl (kernel parameters)

Read and modify kernel parameters at runtime.

## Reading Parameters

### Get Values

```bash
# Get a specific parameter
sysctl net.ipv4.ip_forward

# Just the value (no key)
sysctl -n net.ipv4.ip_forward

# All parameters
sysctl -a

# Search parameters by pattern
sysctl -a | grep tcp_keepalive

# All network parameters
sysctl net.ipv4
```

## Setting Parameters

### Runtime Changes

```bash
# Set a value (lost on reboot)
sysctl -w net.ipv4.ip_forward=1

# Set multiple values
sysctl -w net.core.somaxconn=65535 -w net.ipv4.tcp_max_syn_backlog=65535
```

### Persistent Changes

```bash
# Add to /etc/sysctl.conf or create a file in /etc/sysctl.d/
echo "net.ipv4.ip_forward = 1" >> /etc/sysctl.d/99-custom.conf

# Reload all config files
sysctl -p

# Reload a specific file
sysctl -p /etc/sysctl.d/99-custom.conf

# Reload all files in /etc/sysctl.d/
sysctl --system
```

## Common Tunables

### Networking

```bash
# Enable IP forwarding (for routers/containers)
sysctl -w net.ipv4.ip_forward=1

# Increase connection backlog
sysctl -w net.core.somaxconn=65535

# Increase max open connections tracking
sysctl -w net.netfilter.nf_conntrack_max=262144

# TCP keepalive (seconds)
sysctl -w net.ipv4.tcp_keepalive_time=600

# Allow reuse of TIME_WAIT sockets
sysctl -w net.ipv4.tcp_tw_reuse=1

# Increase local port range
sysctl -w net.ipv4.ip_local_port_range="1024 65535"

# Increase network buffer sizes
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216
```

### Virtual Memory

```bash
# Swappiness (0-100, lower = prefer RAM)
sysctl -w vm.swappiness=10

# Dirty page writeback (percentage of RAM)
sysctl -w vm.dirty_ratio=20
sysctl -w vm.dirty_background_ratio=5

# Overcommit memory (0=heuristic, 1=always, 2=never)
sysctl -w vm.overcommit_memory=0

# Max memory map areas (increase for many mmap'd files)
sysctl -w vm.max_map_count=262144
```

### Kernel

```bash
# Max open file descriptors system-wide
sysctl -w fs.file-max=2097152

# Max inotify watchers (for file monitoring tools)
sysctl -w fs.inotify.max_user_watches=524288

# Core dump pattern
sysctl -w kernel.core_pattern=/tmp/core-%e-%p

# Restrict dmesg to root
sysctl -w kernel.dmesg_restrict=1

# SysRq key (0=disable, 1=enable all)
sysctl -w kernel.sysrq=1
```

## Tips

- Changes via `sysctl -w` are lost on reboot. Always also add them to `/etc/sysctl.d/` for persistence.
- Use `/etc/sysctl.d/99-custom.conf` (not `/etc/sysctl.conf` directly) so your changes survive package upgrades.
- Files in `/etc/sysctl.d/` are loaded in lexicographic order -- higher numbers override lower.
- `vm.max_map_count=262144` is required by Elasticsearch -- the default 65530 causes it to fail.
- `fs.inotify.max_user_watches` too low causes "no space left on device" errors from tools like webpack, VSCode, or inotifywait.
- `sysctl --system` reloads from all standard locations (`/etc/sysctl.d/`, `/run/sysctl.d/`, `/usr/lib/sysctl.d/`).
