# udev (Dynamic Device Manager)

udev is the Linux kernel's device manager that handles device node creation in `/dev`, executes rules when devices are added or removed, provides persistent device naming via symlinks, and supports custom actions triggered by device events for hotplug management.

## udevadm Commands

### Device Information

```bash
# Show all attributes for a device
udevadm info --query=all --name=/dev/sda

# Show device attributes by path
udevadm info --attribute-walk --name=/dev/sda

# Show only the device path
udevadm info --query=path --name=/dev/ttyUSB0

# Show symlinks for a device
udevadm info --query=symlink --name=/dev/sda1

# Show properties (environment) for a device
udevadm info --query=property --name=/dev/sda

# Walk up the device chain (parent attributes)
udevadm info --attribute-walk --path=/sys/class/net/eth0
```

### Monitoring Events

```bash
# Monitor all udev events (kernel + udev)
udevadm monitor

# Monitor kernel events only
udevadm monitor --kernel

# Monitor udev events only (after rule processing)
udevadm monitor --udev

# Monitor with properties shown
udevadm monitor --property

# Monitor specific subsystem
udevadm monitor --subsystem-match=usb

# Monitor with tag filter
udevadm monitor --tag-match=systemd
```

### Triggering and Testing

```bash
# Trigger events for all devices
udevadm trigger

# Trigger for specific subsystem
udevadm trigger --subsystem-match=net

# Trigger for specific action
udevadm trigger --action=change --subsystem-match=block

# Trigger specific device
udevadm trigger --name-match=/dev/sda

# Wait for all pending events to complete
udevadm settle

# Wait with timeout
udevadm settle --timeout=30

# Test rules for a device (dry run)
udevadm test /sys/class/net/eth0

# Test with specific action
udevadm test --action=add /sys/class/block/sda

# Reload rules without reboot
udevadm control --reload-rules
udevadm trigger
```

## Rules Syntax

### Rule File Location

```bash
# System rules (distribution/package managed)
/usr/lib/udev/rules.d/

# Local/admin rules (override system rules)
/etc/udev/rules.d/

# Rule files are processed in lexical order
# Naming convention: NN-description.rules
# Lower numbers run first (e.g., 10-local.rules before 99-custom.rules)
```

### Match Keys

```bash
# KERNEL         - match kernel device name (e.g., "sd*", "ttyUSB*")
# SUBSYSTEM      - match subsystem (e.g., "block", "net", "usb", "input")
# DRIVER         - match driver name
# ATTR{file}     - match sysfs attribute value
# ATTRS{file}    - match sysfs attribute on parent devices
# ACTION         - match event action ("add", "remove", "change", "bind")
# ENV{key}       - match device property
# TAG            - match device tag
# TEST           - test file existence
# PROGRAM        - run external program, match exit code
# RESULT         - match PROGRAM output
# KERNELS        - match parent kernel name
# SUBSYSTEMS     - match parent subsystem
# DRIVERS        - match parent driver
```

### Assignment Keys

```bash
# NAME           - set network interface name
# SYMLINK        - create symlink in /dev
# OWNER          - set device node owner
# GROUP          - set device node group
# MODE           - set device node permissions
# RUN            - run command on event
# LABEL          - label for GOTO
# GOTO           - jump to LABEL
# IMPORT{type}   - import properties (program, file, db, cmdline, parent)
# ENV{key}       - set device property
# TAG            - set/unset device tag
# OPTIONS        - additional options (e.g., "last_rule", "static_node=")
```

### Basic Rule Examples

```bash
# /etc/udev/rules.d/99-usb-serial.rules

# Create persistent symlink for USB serial adapter by vendor/product
SUBSYSTEM=="tty", ATTRS{idVendor}=="0403", ATTRS{idProduct}=="6001", \
  SYMLINK+="ttyFTDI", MODE="0666"

# Name USB device by serial number
SUBSYSTEM=="tty", ATTRS{serial}=="A50285BI", SYMLINK+="gps"

# Set permissions on specific device
KERNEL=="video0", MODE="0666", GROUP="video"

# Run script when USB drive is inserted
ACTION=="add", SUBSYSTEM=="block", KERNEL=="sd[b-z]1", \
  RUN+="/usr/local/bin/automount.sh %k"

# Run script when USB drive is removed
ACTION=="remove", SUBSYSTEM=="block", KERNEL=="sd[b-z]1", \
  RUN+="/usr/local/bin/autounmount.sh %k"
```

## Persistent Device Naming

### Disk Naming

```bash
# View persistent disk names
ls -la /dev/disk/by-id/
ls -la /dev/disk/by-uuid/
ls -la /dev/disk/by-path/
ls -la /dev/disk/by-label/
ls -la /dev/disk/by-partlabel/
ls -la /dev/disk/by-partuuid/

# Find device by UUID
blkid /dev/sda1
udevadm info --query=property --name=/dev/sda1 | grep ID_FS_UUID
```

### Custom Persistent Names

```bash
# /etc/udev/rules.d/70-persistent-usb.rules

# Name by USB port location (always same port = same name)
SUBSYSTEM=="tty", KERNELS=="1-1.2:1.0", SYMLINK+="usb-port1"

# Name by device attributes
SUBSYSTEM=="tty", ATTRS{idVendor}=="2341", ATTRS{idProduct}=="0043", \
  SYMLINK+="arduino"

# Multiple devices differentiated by serial
SUBSYSTEM=="tty", ATTRS{idVendor}=="0403", ATTRS{serial}=="ABC123", \
  SYMLINK+="sensor-temp"
SUBSYSTEM=="tty", ATTRS{idVendor}=="0403", ATTRS{serial}=="DEF456", \
  SYMLINK+="sensor-pressure"
```

## Network Interface Naming

### systemd Predictable Names

```bash
# View current network names and their udev paths
udevadm info --path=/sys/class/net/enp0s3

# Name schemes (systemd v197+):
# en/wl/ww          - Ethernet/WLAN/WWAN prefix
# o<index>           - on-board device index
# s<slot>[f<func>]   - PCI hotplug slot
# p<bus>s<slot>      - PCI geographic location
# x<MAC>             - MAC address

# Examples:
# eno1               - on-board Ethernet #1
# enp3s0             - PCI bus 3 slot 0
# wlp2s0             - wireless PCI bus 2 slot 0
# enx78e7d1ea46da    - MAC-based name
```

### Custom Network Naming

```bash
# /etc/udev/rules.d/70-net-names.rules

# Rename by MAC address
SUBSYSTEM=="net", ACTION=="add", ATTR{address}=="00:11:22:33:44:55", \
  NAME="lan0"

# Rename by PCI path
SUBSYSTEM=="net", ACTION=="add", KERNELS=="0000:03:00.0", NAME="mgmt0"

# Rename by driver
SUBSYSTEM=="net", ACTION=="add", DRIVERS=="virtio_net", \
  ATTR{dev_id}=="0x0", NAME="vnet0"
```

## Advanced Rules

### Using PROGRAM and IMPORT

```bash
# /etc/udev/rules.d/80-custom.rules

# Run a program and use its output
SUBSYSTEM=="block", KERNEL=="sd*", \
  PROGRAM="/usr/local/bin/disk-label.sh %k", \
  SYMLINK+="disk-%c"

# Import properties from a program
SUBSYSTEM=="usb", IMPORT{program}="/usr/local/bin/usb-classify.sh"
ENV{USB_CLASS}=="storage", SYMLINK+="storage/%k"

# Import from a file
SUBSYSTEM=="net", IMPORT{file}="/etc/udev/net-settings.conf"

# Use string substitutions
# %k = kernel name,  %n = kernel number
# %p = devpath,      %b = bus id
# %M = major number, %m = minor number
# %c = PROGRAM output
# $env{key} = environment variable
```

### Power Management Rules

```bash
# /etc/udev/rules.d/90-power.rules

# Enable autosuspend for USB devices
ACTION=="add", SUBSYSTEM=="usb", TEST=="power/autosuspend", \
  ATTR{power/autosuspend}="2"

# Disable autosuspend for specific device
ACTION=="add", SUBSYSTEM=="usb", \
  ATTRS{idVendor}=="046d", ATTRS{idProduct}=="c52b", \
  ATTR{power/autosuspend}="-1"

# Set disk APM on battery
ACTION=="change", SUBSYSTEM=="power_supply", \
  ATTR{type}=="Mains", ATTR{online}=="0", \
  RUN+="/usr/sbin/hdparm -B 128 /dev/sda"
```

## Debugging

### Troubleshooting Workflow

```bash
# 1. Find the device's sysfs path
udevadm info --query=path --name=/dev/sda

# 2. Walk the attribute chain for matching
udevadm info --attribute-walk --name=/dev/sda

# 3. Test rules (dry run, verbose)
udevadm test /sys/class/block/sda 2>&1

# 4. Monitor events in real time
udevadm monitor --property --subsystem-match=block

# 5. Reload and trigger
udevadm control --reload-rules
udevadm trigger --subsystem-match=block

# 6. Check udev log for errors
journalctl -u systemd-udevd -f

# 7. Increase udev log verbosity
udevadm control --log-priority=debug
# Reset after debugging
udevadm control --log-priority=info
```

## Tips

- Always use `udevadm info --attribute-walk` to discover matchable attributes before writing rules
- Use `ATTRS{}` (with S) to match parent device attributes; without S only matches the device itself
- Use `+=` for SYMLINK and RUN to append rather than replace when multiple rules apply
- Test rules with `udevadm test` before reloading to catch syntax errors without side effects
- Name rule files with a numeric prefix (e.g., `99-`) so they run after system rules
- Use `SYMLINK` instead of `NAME` for block devices; renaming block devices can break boot
- Network interface renaming must happen early; use prefix `70-` or lower for net name rules
- The `%k` substitution in RUN commands expands to the kernel device name at rule evaluation time
- Use `GOTO` and `LABEL` to short-circuit rule evaluation for performance in large rule sets
- After writing rules, always run `udevadm control --reload-rules && udevadm trigger` to apply
- Set `udevadm control --log-priority=debug` temporarily when rules fail to match
- Use `ENV{SYSTEMD_WANTS}` to start a systemd unit when a specific device appears

## See Also

- systemd, dbus, lsblk, blkid, lsusb, lspci

## References

- [udev Manual Page](https://www.freedesktop.org/software/systemd/man/udev.html)
- [udevadm Manual Page](https://www.freedesktop.org/software/systemd/man/udevadm.html)
- [Writing udev Rules (Arch Wiki)](https://wiki.archlinux.org/title/udev)
- [systemd.link — Network Interface Naming](https://www.freedesktop.org/software/systemd/man/systemd.link.html)
