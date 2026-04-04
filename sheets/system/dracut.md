# dracut (Initramfs Generator)

dracut is the standard initramfs (initial RAM filesystem) generator for Linux that assembles kernel modules, binaries, and scripts into a compressed archive loaded at boot, supporting modular composition, network boot, LVM, LUKS encryption, and emergency shell debugging.

## Basic Usage

### Building Initramfs

```bash
# Rebuild initramfs for current kernel
dracut --force

# Rebuild for a specific kernel version
dracut --force /boot/initramfs-6.8.0.img 6.8.0

# Rebuild with verbose output
dracut --force --verbose

# Rebuild all initramfs images
dracut --regenerate-all

# Build with only specific modules
dracut --force --add "lvm crypt"

# Build without specific modules
dracut --force --omit "plymouth network"

# Build a hostonly initramfs (minimal, for this machine only)
dracut --force --hostonly

# Build a generic initramfs (works on any hardware)
dracut --force --no-hostonly

# Print what would be included (dry run)
dracut --force --print-cmdline
```

### Inspecting Initramfs

```bash
# List contents of initramfs
lsinitrd /boot/initramfs-$(uname -r).img

# List only files (no module info)
lsinitrd /boot/initramfs-$(uname -r).img -f

# Show included modules
lsinitrd /boot/initramfs-$(uname -r).img -m

# Extract a specific file
lsinitrd /boot/initramfs-$(uname -r).img /etc/cmdline.d/

# Show kernel modules included
lsinitrd /boot/initramfs-$(uname -r).img --kmoddir

# Extract initramfs to a directory
mkdir /tmp/initrd-contents
cd /tmp/initrd-contents
zcat /boot/initramfs-$(uname -r).img | cpio -idmv
# Or for xz-compressed
xzcat /boot/initramfs-$(uname -r).img | cpio -idmv
```

## Configuration

### dracut.conf

```bash
# /etc/dracut.conf.d/custom.conf

# Include additional modules
add_dracutmodules+=" lvm crypt network "

# Omit modules
omit_dracutmodules+=" plymouth brltty "

# Add kernel modules
add_drivers+=" ahci nvme "

# Force include kernel modules
force_drivers+=" vfio vfio-pci "

# Install additional files
install_items+=" /etc/crypttab /usr/local/bin/unlock.sh "

# Set hostonly mode
hostonly="yes"
hostonly_cmdline="yes"

# Compress with specific algorithm
compress="xz"
# Options: gzip, bzip2, xz, lzma, lz4, zstd, cat (no compression)

# Set kernel command line defaults
kernel_cmdline="rd.lvm.vg=vg0 rd.luks.uuid=abc-123"

# Include firmware
fw_dir+=" /lib/firmware/updates "

# Set early microcode loading
early_microcode="yes"
```

### Drop-in Configuration

```bash
# List configuration priority
# /etc/dracut.conf                     - main config
# /etc/dracut.conf.d/*.conf            - drop-in overrides (alphabetical)
# /usr/lib/dracut/dracut.conf.d/*.conf - package defaults

# Example: /etc/dracut.conf.d/10-network.conf
add_dracutmodules+=" network-manager "
kernel_cmdline+=" ip=dhcp rd.neednet=1 "

# Example: /etc/dracut.conf.d/20-crypt.conf
add_dracutmodules+=" crypt "
kernel_cmdline+=" rd.luks=1 "
```

## Modules

### Core Modules

```bash
# List all available dracut modules
dracut --list-modules

# Key modules:
# base           - core functionality (init, udev, systemd)
# systemd        - systemd in initramfs
# kernel-modules - hardware detection modules
# rootfs-block   - block device root mounting
# fs-lib         - filesystem utilities

# Storage modules:
# lvm            - LVM logical volume support
# crypt          - LUKS encryption (dm-crypt)
# mdraid         - software RAID (md)
# multipath      - device-mapper multipath
# iscsi          - iSCSI initiator
# nfs            - NFS root filesystem
# nbd            - network block device

# Network modules:
# network        - legacy network scripts
# network-manager - NetworkManager in initramfs
# ifcfg          - interface configuration

# Security modules:
# tpm2-tss       - TPM 2.0 integration
# fido2          - FIDO2 token unlocking
# pkcs11         - PKCS#11 smart card

# Debug modules:
# debug          - additional debugging tools
# rescue         - rescue shell support
```

## Kernel Command Line Parameters

### Boot Debugging

```bash
# Break into emergency shell at various stages
rd.break                    # break before pivot_root
rd.break=pre-udev           # before udev starts
rd.break=pre-trigger        # before udev trigger
rd.break=pre-mount          # before root mount
rd.break=mount              # after root mount
rd.break=pre-pivot          # before switch_root
rd.break=cleanup            # before cleanup

# Enable shell on failure
rd.shell                    # drop to shell on failure
rd.debug                    # maximum debug output

# Kernel debug output
rd.info                     # informational messages
rd.memdebug=3               # memory debug level
systemd.log_level=debug     # systemd verbose logging
systemd.log_target=console  # log to console
```

### Root Device Parameters

```bash
# Root device specification
root=/dev/mapper/vg0-root
root=UUID=abc-123-def
root=LABEL=rootfs
root=/dev/sda2

# Root filesystem options
rootfstype=ext4
rootflags=rw,noatime

# LVM parameters
rd.lvm=1                         # enable LVM
rd.lvm.vg=vg0                    # activate volume group
rd.lvm.lv=vg0/root               # activate specific LV
rd.lvm.conf=0                    # skip lvm.conf

# LUKS parameters
rd.luks=1                        # enable LUKS
rd.luks.uuid=<uuid>              # unlock specific UUID
rd.luks.name=<uuid>=cryptroot    # name the device
rd.luks.key=/keyfile:UUID=<uuid> # keyfile on device
rd.luks.options=discard           # pass options to cryptsetup
rd.luks.timeout=60                # unlock timeout

# MD RAID parameters
rd.md=1                          # enable MD RAID
rd.md.uuid=<uuid>                # assemble specific array
rd.md.conf=0                     # skip mdadm.conf

# Network root (NFS/iSCSI)
ip=dhcp                          # DHCP for network
ip=192.168.1.10::192.168.1.1:255.255.255.0:host:eth0:none
rd.neednet=1                     # require network
root=nfs:192.168.1.1:/exports/root
```

## Emergency Shell Debugging

### Common Debug Workflow

```bash
# 1. Boot with rd.break to get a shell
# At GRUB: press 'e', append to linux line:
#   rd.break rd.shell

# 2. In emergency shell, inspect the environment
cat /proc/cmdline
ls /dev/mapper/
lvm pvs
lvm vgs
lvm lvs
cryptsetup status cryptroot

# 3. Manually mount root if needed
mount -o rw /dev/mapper/vg0-root /sysroot

# 4. Chroot into the real root
chroot /sysroot

# 5. Fix issues (e.g., rebuild initramfs, fix fstab)
dracut --force
vi /etc/fstab

# 6. Exit and continue boot
exit
exit
```

## Custom Modules

### Module Structure

```bash
# Create a custom module
mkdir -p /usr/lib/dracut/modules.d/99mymodule

# module-setup.sh (required)
cat > /usr/lib/dracut/modules.d/99mymodule/module-setup.sh << 'EOF'
#!/bin/bash

check() {
    # Return 0 to include, 1 to skip
    require_binaries my-tool || return 1
    return 0
}

depends() {
    # List module dependencies
    echo "base"
}

install() {
    # Install files into initramfs
    inst_binary /usr/local/bin/my-tool
    inst_simple /etc/my-tool.conf
    inst_hook pre-mount 50 "$moddir/my-hook.sh"
}

installkernel() {
    # Install kernel modules
    instmods my-driver
}
EOF

# my-hook.sh (runs at pre-mount)
cat > /usr/lib/dracut/modules.d/99mymodule/my-hook.sh << 'EOF'
#!/bin/bash
type my-tool > /dev/null 2>&1 && my-tool --init
EOF

chmod +x /usr/lib/dracut/modules.d/99mymodule/*.sh
```

### Hook Points

```bash
# Available hook points (in execution order):
# cmdline          - parse kernel command line
# pre-udev         - before udev daemon starts
# pre-trigger      - before udev trigger
# initqueue        - main init queue (wait for devices)
# pre-mount        - before root filesystem mount
# mount            - mount root filesystem
# pre-pivot        - before switch to real root
# cleanup          - final cleanup

# Hook priority: lower number = runs earlier
# inst_hook <hookpoint> <priority> <script>
# inst_hook pre-mount 50 "$moddir/my-hook.sh"
```

## Tips

- Always use `--force` when rebuilding to overwrite the existing initramfs image
- Use `--hostonly` for production systems to minimize initramfs size and boot time
- Use `--no-hostonly` for rescue images or when moving disks between machines
- Test initramfs changes by booting with `rd.break` before committing to production
- Keep a known-good initramfs backup before rebuilding: `cp initramfs.img initramfs.img.bak`
- Use `lsinitrd -m` to verify that required modules (crypt, lvm, network) are included
- Drop-in configs in `/etc/dracut.conf.d/` are cleaner than editing the main `/etc/dracut.conf`
- When LUKS unlock fails at boot, check that `rd.luks.uuid` matches `cryptsetup luksUUID /dev/sdX`
- Use `rd.debug` on the kernel command line to get maximum diagnostic output in journalctl
- After kernel updates, verify the new initramfs exists and contains the right modules
- Use `zstd` compression for the best balance of initramfs size and decompression speed

## See Also

- grub, systemd, lvm, luks, cryptsetup, kernel, mkinitcpio

## References

- [dracut Manual Page](https://www.man7.org/linux/man-pages/man8/dracut.8.html)
- [dracut Kernel Command Line](https://www.man7.org/linux/man-pages/man7/dracut.cmdline.7.html)
- [dracut Modules Guide](https://github.com/dracut-ng/dracut-ng/wiki)
- [Fedora dracut Documentation](https://docs.fedoraproject.org/en-US/fedora/latest/system-administrators-guide/kernel-module-driver-configuration/Working_with_the_GRUB_2_Boot_Loader/)
