# dmesg (kernel ring buffer)

Display and control the kernel ring buffer messages.

## Basic Usage

### View Kernel Messages

```bash
# All messages
dmesg

# Follow (like tail -f)
dmesg -w

# Human-readable timestamps
dmesg -T

# Follow with human timestamps
dmesg -Tw
```

## Filtering by Level

### Log Levels

```bash
# Only errors and above
dmesg -l err

# Warnings and errors
dmesg -l warn,err

# Emergency, alert, critical
dmesg -l emerg,alert,crit

# Available levels: emerg, alert, crit, err, warn, notice, info, debug
```

## Filtering by Facility

### Facility Types

```bash
# Kernel messages only
dmesg -f kern

# Daemon messages
dmesg -f daemon

# Multiple facilities
dmesg -f kern,user,daemon
```

## Output Control

### Formatting

```bash
# Color output
dmesg -L

# No pager (raw output)
dmesg --nopager

# Show facility and level prefix
dmesg -x

# Decode facility and level to readable text
dmesg -x -T
```

## Since Boot

### Recent Boot Messages

```bash
# Only since last boot (clears ring buffer counter)
dmesg -c    # read AND clear (requires root)

# Clear without reading
dmesg -C

# Messages with timestamp since boot
dmesg -T
```

## Common Patterns

### Hardware and Driver Issues

```bash
# USB events
dmesg -Tw | grep -i usb

# Disk errors
dmesg -T -l err | grep -i 'sd\|nvme\|ata'

# Out of memory killer
dmesg -T | grep -i "oom\|out of memory"

# Filesystem errors
dmesg -T | grep -i "ext4\|xfs\|btrfs"

# Network interface changes
dmesg -T | grep -i "link\|eth\|eno\|enp"
```

### Kernel Panics and Oops

```bash
# Kernel oops messages
dmesg | grep -i "oops\|panic\|bug\|call trace"

# Segfaults
dmesg -T | grep segfault
```

## Tips

- `dmesg -T` timestamps are approximations (wall-clock calculated from boot time + monotonic offset) -- they can drift slightly.
- The ring buffer has a fixed size (typically 256KB-1MB); old messages are overwritten. Use `journalctl -k` for persistent kernel logs.
- `dmesg` requires root on many modern distros (controlled by `kernel.dmesg_restrict` sysctl).
- `-w` (follow mode) is essential when diagnosing hardware issues in real time (plug in a USB device, watch output).
- `dmesg -c` both displays AND clears the buffer -- useful for isolating messages from a specific action.
- On systems with systemd, `journalctl -k` is an alternative that supports the same time filtering as journalctl.
