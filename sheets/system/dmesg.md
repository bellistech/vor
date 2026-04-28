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

## All Flags Reference

```bash
dmesg [options]

# Output control
-w, --follow            tail -f equivalent (live append, no buffer-clear)
-W, --follow-new        only follow NEW messages (don't dump existing)
-T, --ctime             human-readable wall-clock timestamps
-d, --show-delta        seconds elapsed since previous message
-e, --reltime           relative time format (e.g. "[Wed Apr 28 10:00] +0.0s")
-H, --human             color, paged, smart timestamps
-L, --color=always|auto|never  color output
-J, --json              JSON output (one object per record; util-linux 2.32+)
--noescape              don't escape unprintable bytes
--nopager               disable the auto-pager (otherwise dmesg pages on TTY)

# Filtering
-l, --level <list>      filter by priority: emerg,alert,crit,err,warn,notice,info,debug
-f, --facility <list>   filter by facility: kern,user,mail,daemon,auth,syslog,lpr,
                                            news,uucp,cron,authpriv,ftp,local0-7
-k, --kernel            same as -f kern (most common)
-u, --userspace         same as -f user (and others)
--since "1 hour ago"    relative time filter (uses dateparse)
--until "10:00:00"      complementary upper bound

# Buffer management (root required)
-c, --read-clear        read AND zero the buffer
-C, --clear             zero without reading
-n, --console-level <N> set console log level (1=emerg only, 7=debug)
--console-on            re-enable kernel→console messages
--console-off           silence kernel→console (still in buffer)
-D                      disable kernel-to-console (alias for --console-off)
-E                      enable kernel-to-console (alias for --console-on)

# Other
-x, --decode            decode facility AND level to text prefix
-s, --buffer-size N     set the size of the buffer to read (default 16K, max 16M)
--time-format raw|ctime|reltime|delta|notime|iso  custom timestamp formatting
--force-prefix          always prefix lines with timestamp/facility/level
-V                      version
```

## Log Level + Facility Tables

```
LEVEL (severity)    NUMBER  WHEN  TO USE
─────────────────  ──────  ──────────────────────────────────────────────
emerg               0       system unusable (kernel BUG, fs corruption)
alert               1       action must be taken immediately
crit                2       critical condition (hardware failure)
err                 3       error condition (driver error, OOM kill)
warn                4       warning (non-fatal)
notice              5       normal but significant
info                6       informational (driver init, link up)
debug               7       debug (verbose driver chatter)

FACILITY            COMMON SOURCES
─────────────────  ──────────────────────────────────────────────────────
kern                kernel itself (most dmesg lines)
user                user-space programs that wrote to /dev/kmsg or syslog
mail                MTA / sendmail / postfix
daemon              non-privileged daemons
auth                login / sshd authentication
syslog              syslog facility itself
lpr                 printer subsystem
news                NNTP (legacy)
uucp                UUCP (legacy)
cron                cron / atd
authpriv            secure authentication (often filtered out of /var/log/messages)
ftp                 FTP daemon
local0..local7      site-specific
```

## journalctl -k Equivalents

When systemd is present, `journalctl -k` reads the same source PLUS persists across reboots:

| dmesg | journalctl equivalent |
|-------|----------------------|
| `dmesg -T` | `journalctl -k -o short-iso` |
| `dmesg -Tw` | `journalctl -k -f` |
| `dmesg -l err,warn` | `journalctl -k -p warning` |
| `dmesg -f kern` | (always — `-k` filters to kernel already) |
| `dmesg --since="1 hour ago"` | `journalctl -k --since="1 hour ago"` |
| (current boot) | `journalctl -k -b` |
| (previous boot) | `journalctl -k -b -1` |
| (list reboots) | `journalctl --list-boots` |

The big advantage of `journalctl -k`: persistent across reboots (provided `/var/log/journal/` exists). The big disadvantage: requires systemd and a running journald.

## /dev/kmsg — Reading the Source

```bash
# Live tail of the kernel ring buffer (the same source dmesg reads)
sudo cat /dev/kmsg

# Each line: <facility>,<level>,<sequence>,<timestamp_us>;<message>
# Example:
#   6,1234,5678901234,-;usb 1-2: new high-speed USB device

# Write a message into the buffer (root only — from userspace, for marking)
echo "deploy: starting v3.2.1" | sudo tee /dev/kmsg
# Now visible in `dmesg` with facility=user, level=info
```

## Common Hardware Diagnostic Recipes

### Recipe 1 — Disk health

```bash
# All disk-layer events since boot
dmesg -T -l err,warn | grep -iE "(sd|nvme|ata|scsi|i/o error)"

# What does this disk see right now?
dmesg -Tw | grep -i nvme0n1   # in another terminal, run something on nvme0n1

# Common alarming lines:
#   "I/O error, dev sda, sector 12345"           — bad block / dying disk
#   "ata1: hard resetting link"                  — SATA link instability
#   "nvme nvme0: I/O 1234 QID 5 timeout"         — NVMe controller hang
#   "Buffer I/O error on dev sda1"               — fs/disk mismatch
#   "ext4-fs (sda1): error -5"                   — fs ran into a hardware issue
```

### Recipe 2 — USB plug/unplug debugging

```bash
# Live view as you plug something in
dmesg -Twi
# (then in another terminal)
# plug in a USB device — see "new high-speed USB device" lines

# Common patterns:
#   "device descriptor read/64, error -71"  → flaky cable or hub
#   "USB disconnect, device number 4"       → cable yanked
#   "device not accepting address X"        → BIOS/firmware bug, try other port
```

### Recipe 3 — OOM killer victims

```bash
# Find the most recent OOM kill
dmesg -T | grep -i "killed process"
# [Mon Apr 28 09:14:22 2026] Out of memory: Killed process 12345 (postgres) ...

# Full OOM report (memory map, eligible processes, scoring)
dmesg -T | grep -A 50 "invoked oom-killer"
# Look at:
#   gfp_mask=...                  what kind of allocation triggered it
#   order=N                       2^N pages requested
#   oom_score_adj=N               adjustment if any
#   pid     uid     tgid    total_vm  rss   pgtables_bytes  swapents oom_score_adj  name
#   ...                                                                              ← scoreboard

# Per-process oom_score (sort to find next likely victim)
for p in /proc/[0-9]*; do
  printf "%6d %s %s\n" \
    "$(cat $p/oom_score 2>/dev/null)" \
    "$(cat $p/oom_score_adj 2>/dev/null)" \
    "$(cat $p/comm 2>/dev/null)"
done | sort -rn | head -10
```

### Recipe 4 — Network interface flapping

```bash
# Watch link-state changes in real time
dmesg -Tw | grep -iE "(link|eth|eno|enp|wlan|bond)"

# Common lines:
#   "eth0: link up, 1000Mbps, full-duplex"
#   "eth0: link down"
#   "eth0: NIC Link is Down"
#   "eth0: tx hang detected, resetting"           — driver-level reset
#   "br0: port 2(eth0) entered blocking state"    — STP recalculating

# Kernel network counters
ip -s link show eth0
```

### Recipe 5 — Kernel panic / soft lockup

```bash
# Panics get written to dmesg, but the box may also reboot before you
# can read them. Two approaches:

# 1. Persistent journal — works across reboots
journalctl -k -p err --since="1 day ago"
journalctl -k -b -1                  # previous boot's kernel log

# 2. kdump — saves a kernel core dump to /var/crash/
ls /var/crash/

# Common patterns:
#   "BUG: soft lockup - CPU#N stuck for Xs!"
#   "watchdog: BUG: hard LOCKUP"
#   "Kernel panic - not syncing: ..."
#   "RIP: 0010:[<ffffffff...>]"           — instruction pointer in the offending function
#   "Call Trace:"                          — followed by the stack trace
```

## Kernel Taint Flags

If something is wrong with your kernel, dmesg will say so:

```bash
dmesg | grep -i taint
# Possible taints (from include/linux/kernel.h):
#   P  proprietary module loaded (e.g. NVIDIA)
#   F  module force-loaded (insmod -f)
#   S  SMP with kernel built without SMP
#   R  module force-unloaded
#   M  machine check exception
#   B  bad page reference
#   U  user-supplied taint (echo > /proc/sys/kernel/tainted)
#   D  kernel oops occurred
#   A  ACPI table override
#   W  warning issued (BUG_ON elided)
#   C  staging driver
#   I  workaround for firmware bug
#   O  out-of-tree module
#   E  unsigned module loaded
#   K  kernel live-patched
#   X  auxiliary taint defined for distros
```

If you see `D` (oops) the kernel has hit a known buggy code path. If you see `W` (warning), it tried to continue. Check the original message; "tainted" by itself doesn't break anything, but bug reports against tainted kernels are usually closed by upstream.

## Common Errors and Fixes

```bash
# "dmesg: read kernel buffer failed: Operation not permitted"
$ dmesg
# Cause: kernel.dmesg_restrict=1 (enabled on most modern distros).
$ cat /proc/sys/kernel/dmesg_restrict
1
# Fix: sudo dmesg, or sudo sysctl -w kernel.dmesg_restrict=0 (until reboot).

# Garbled output / no color
# Cause: pager doesn't support ANSI escapes.
PAGER='less -R' dmesg
# Or just bypass the pager:
dmesg --nopager | head

# Timestamps say "[    0.000000]" — what does that mean?
# Cause: kernel ring buffer's native timestamp is monotonic seconds since
#        boot, not wall-clock. Use -T to convert.
dmesg -T

# Ring buffer is full of old messages, can't see new ones
# Fix: clear and start fresh (root):
sudo dmesg -C
# Or just use -W (follow new only) instead of -w (follow all)

# `dmesg -w` exits silently after a while
# Cause: SIGPIPE from a downstream pager, or terminal closed.
# Fix: dmesg -w | less +F   (less stays open even when source EOFs)
```

## Tips

- `dmesg -T` timestamps are approximations (wall-clock calculated from boot time + monotonic offset) — they can drift slightly.
- The ring buffer has a fixed size (typically 1MB on modern kernels, configurable via `CONFIG_LOG_BUF_SHIFT`); old messages are overwritten. Use `journalctl -k` for persistent kernel logs.
- `dmesg` requires root on many modern distros (controlled by `kernel.dmesg_restrict` sysctl).
- `-w` (follow mode) is essential when diagnosing hardware issues in real time (plug in a USB device, watch output).
- `dmesg -c` both displays AND clears the buffer — useful for isolating messages from a specific action.
- On systems with systemd, `journalctl -k` is an alternative that supports the same time filtering as journalctl AND persists across reboots.
- `dmesg -J` (JSON, util-linux 2.32+) is much easier to parse than the text format for log-aggregation pipelines.
- The current ring buffer size is at `/proc/sys/kernel/printk_ratelimit` and related; `dmesg -s` only changes how much you READ at once.
- To write a marker into the buffer: `echo "marker" | sudo tee /dev/kmsg` — useful for "I'm running this test now, here's the boundary".
- The "BUG_ON" macro emits a warning + taint W; the "BUG()" macro emits an oops + taint D. The former is recoverable; the latter usually isn't.

## See Also

- system/journalctl, system/sysctl, system/systemd, troubleshooting/linux-errors

## References

- [man dmesg(1)](https://man7.org/linux/man-pages/man1/dmesg.1.html)
- [man syslog(2)](https://man7.org/linux/man-pages/man2/syslog.2.html)
- [man kmsg(4) — /dev/kmsg](https://man7.org/linux/man-pages/man4/kmsg.4.html)
- [Kernel Log Buffer Documentation](https://www.kernel.org/doc/html/latest/core-api/printk-basics.html)
- [Kernel printk Documentation](https://www.kernel.org/doc/html/latest/admin-guide/serial-console.html)
- [Arch Wiki — Kernel Messages](https://wiki.archlinux.org/title/Syslog)
- [Red Hat — Inspecting Kernel Ring Buffer](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_monitoring_and_updating_the_kernel/getting-started-with-kernel-logging_managing-monitoring-and-updating-the-kernel)
- [Ubuntu Manpage — dmesg](https://manpages.ubuntu.com/manpages/noble/man1/dmesg.1.html)
