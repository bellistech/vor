# Linux Boot Process

Complete boot sequence from firmware to login prompt -- UEFI, GRUB2, initramfs, kernel init, and systemd targets.

## Boot Sequence Overview

```
┌──────────────┐
│  1. Firmware  │  BIOS (legacy) or UEFI
│  POST + init  │  Hardware initialization, find boot device
└──────┬───────┘
       ▼
┌──────────────┐
│  2. Bootloader│  GRUB2 (or systemd-boot, syslinux)
│  Load kernel  │  Read config, present menu, load vmlinuz + initrd
└──────┬───────┘
       ▼
┌──────────────┐
│  3. Kernel    │  Decompress, hardware detection, mount initramfs
│  Early init   │  Run /init from initramfs
└──────┬───────┘
       ▼
┌──────────────┐
│  4. initramfs │  Load storage drivers, find real root filesystem
│  Root pivot   │  Assemble RAID/LVM, decrypt LUKS, mount root
└──────┬───────┘
       ▼
┌──────────────┐
│  5. systemd   │  PID 1 — mount filesystems, start services
│  (or SysVinit)│  Reach target (multi-user, graphical)
└──────┬───────┘
       ▼
┌──────────────┐
│  6. Login     │  getty (TTY) or display manager (GDM, SDDM)
└──────────────┘
```

## BIOS vs UEFI Boot

### BIOS (Legacy) boot

```
BIOS boot sequence:
  1. Power on → POST (Power-On Self-Test)
  2. BIOS reads first 512 bytes of boot device (MBR)
  3. MBR contains:
     - Bootstrap code (446 bytes) → stage 1 bootloader
     - Partition table (64 bytes) → 4 primary partitions
     - Boot signature (2 bytes) → 0x55AA
  4. Stage 1 loads stage 1.5 (from MBR gap or boot partition)
  5. Stage 1.5 loads stage 2 (GRUB2 from /boot/grub2/)
  6. GRUB2 loads kernel + initrd

Limitations:
  - Max disk size: 2 TB (MBR 32-bit LBA)
  - Max 4 primary partitions
  - No built-in security (no Secure Boot)
  - BIOS runs in 16-bit real mode
```

### UEFI boot

```
UEFI boot sequence:
  1. Power on → UEFI firmware initializes hardware
  2. UEFI reads GPT (GUID Partition Table) — supports disks > 2 TB
  3. UEFI looks for EFI System Partition (ESP):
     - FAT32 filesystem, typically /boot/efi/ (mounted)
     - Contains .efi bootloader binaries
  4. UEFI loads bootloader: /EFI/fedora/shimx64.efi (or grubx64.efi)
  5. Bootloader loads kernel + initrd
  6. Kernel boots

ESP layout:
  /boot/efi/
  └── EFI/
      ├── BOOT/
      │   └── BOOTX64.EFI            ← fallback bootloader
      ├── fedora/
      │   ├── shimx64.efi            ← Secure Boot shim
      │   ├── grubx64.efi            ← GRUB2 EFI binary
      │   └── grub.cfg               ← GRUB config (or pointer)
      └── Microsoft/
          └── Boot/
              └── bootmgfw.efi       ← Windows bootloader (dual-boot)
```

### UEFI boot manager

```bash
# List UEFI boot entries
efibootmgr -v

# Sample output:
# BootCurrent: 0001
# BootOrder: 0001,0000,0003
# Boot0000* Windows Boot Manager  HD(1,GPT,...)/EFI/Microsoft/Boot/bootmgfw.efi
# Boot0001* Fedora                HD(1,GPT,...)/EFI/fedora/shimx64.efi
# Boot0003* USB Drive             PciRoot(0x0)/USB(...)

# Set boot order
sudo efibootmgr -o 0001,0000

# Add new boot entry
sudo efibootmgr -c -d /dev/sda -p 1 -l '\EFI\custom\bootx64.efi' -L "Custom Boot"

# Delete boot entry
sudo efibootmgr -b 0003 -B

# Set next boot only
sudo efibootmgr -n 0000
```

## UEFI Secure Boot

### Chain of trust

```
UEFI Secure Boot chain:
  1. UEFI firmware contains Microsoft's root CA certificates (PK, KEK, db)
  2. Firmware loads shim (shimx64.efi) — signed by Microsoft
  3. Shim loads GRUB2 (grubx64.efi) — signed by distro vendor
  4. GRUB2 loads kernel (vmlinuz) — signed by distro vendor
  5. Kernel loads modules — must be signed

Certificate databases:
  PK  (Platform Key)    — single key, firmware owner (OEM)
  KEK (Key Exchange Key) — keys that can modify db/dbx (Microsoft, OEM)
  db  (Signature DB)     — trusted signing certificates (Microsoft, distro)
  dbx (Forbidden DB)     — revoked/blacklisted signatures
```

### MOK (Machine Owner Key)

```bash
# MOK allows users to enroll custom keys (for custom kernels, DKMS modules)

# Generate MOK
openssl req -new -x509 -newkey rsa:2048 -keyout MOK.priv -outform DER \
  -out MOK.der -nodes -days 36500 -subj "/CN=My MOK/"

# Enroll MOK (requires reboot + physical presence)
sudo mokutil --import MOK.der
# Enter password → reboot → MokManager prompts → enroll key

# List enrolled MOKs
mokutil --list-enrolled

# Sign a kernel module
sudo /usr/src/kernels/$(uname -r)/scripts/sign-file sha256 \
  MOK.priv MOK.der /path/to/module.ko

# Check Secure Boot status
mokutil --sb-state
# SecureBoot enabled / disabled

# Disable Secure Boot validation (for debugging)
sudo mokutil --disable-validation
```

### Shim

```bash
# Shim is the first-stage UEFI bootloader signed by Microsoft
# It loads GRUB2 and validates its signature

# Shim location
ls -la /boot/efi/EFI/*/shim*.efi

# Shim validates against:
#   1. Distro vendor key (embedded in shim)
#   2. MOK (Machine Owner Keys, enrolled by user)
#   3. UEFI db (firmware signature database)
```

## GRUB2

### Configuration

```bash
# GRUB2 config files
/boot/grub2/grub.cfg                        # BIOS systems (DO NOT edit directly)
/boot/efi/EFI/<distro>/grub.cfg             # UEFI systems
/etc/default/grub                           # user configuration (edit this)
/etc/grub.d/                                # menu entry scripts

# Regenerate grub.cfg
sudo grub2-mkconfig -o /boot/grub2/grub.cfg          # BIOS (RHEL/CentOS)
sudo grub2-mkconfig -o /boot/efi/EFI/fedora/grub.cfg # UEFI (Fedora)
sudo update-grub                                      # Debian/Ubuntu shortcut
```

### /etc/default/grub

```bash
GRUB_TIMEOUT=5                               # menu timeout (seconds)
GRUB_DEFAULT=saved                           # default entry (0, saved, "title")
GRUB_DISABLE_SUBMENU=true                    # flat menu (no submenus)
GRUB_CMDLINE_LINUX="rhgb quiet"              # kernel parameters for all entries
GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"    # Debian: normal boot only
GRUB_DISABLE_RECOVERY="false"                # show recovery entries
GRUB_ENABLE_BLSCFG=true                      # Boot Loader Spec entries (RHEL 9)
```

### Kernel parameters

```bash
# Add kernel parameters permanently
# Edit /etc/default/grub → GRUB_CMDLINE_LINUX="..."
sudo grub2-mkconfig -o /boot/grub2/grub.cfg

# Common kernel parameters
quiet                            # suppress most boot messages
rhgb                             # Red Hat graphical boot
splash                           # show splash screen
rd.break                         # break into initramfs shell (before root mount)
init=/bin/bash                   # skip init system, drop to shell
systemd.unit=rescue.target       # boot to rescue mode
systemd.unit=emergency.target    # boot to emergency mode
systemd.unit=multi-user.target   # boot to multi-user (no GUI)
single / s / 1                   # single-user mode (legacy)
enforcing=0                      # SELinux permissive mode
selinux=0                        # disable SELinux
net.ifnames=0 biosdevname=0      # use legacy NIC naming (eth0)
nomodeset                        # disable kernel mode setting (GPU issues)
mem=4G                           # limit usable memory
console=ttyS0,115200             # serial console
rd.lvm.lv=vg/root               # LVM root volume
rd.luks.uuid=<UUID>              # LUKS encrypted root
```

### GRUB2 rescue

```bash
# If GRUB menu appears but fails to boot:
# Press 'e' at menu to edit entry
# Press 'c' for GRUB command line

# GRUB command line rescue
grub> ls                                     # list partitions
grub> ls (hd0,gpt2)/                        # list files on partition
grub> set root=(hd0,gpt2)                   # set root partition
grub> linux /vmlinuz-5.x root=/dev/sda2     # load kernel
grub> initrd /initramfs-5.x.img             # load initrd
grub> boot                                   # boot

# Reinstall GRUB (from rescue/live environment)
# BIOS:
sudo grub2-install /dev/sda
sudo grub2-mkconfig -o /boot/grub2/grub.cfg

# UEFI:
sudo dnf reinstall grub2-efi-x64 shim-x64   # reinstall GRUB EFI
sudo grub2-mkconfig -o /boot/efi/EFI/fedora/grub.cfg

# From chroot (after mounting root + /boot + /boot/efi):
mount /dev/sda2 /mnt
mount /dev/sda1 /mnt/boot
mount /dev/sda1 /mnt/boot/efi                # if separate ESP
mount --bind /dev /mnt/dev
mount --bind /proc /mnt/proc
mount --bind /sys /mnt/sys
chroot /mnt
grub2-install /dev/sda                       # BIOS
grub2-mkconfig -o /boot/grub2/grub.cfg
exit
```

### Set default boot entry

```bash
# List available entries
sudo grub2-editenv list
sudo awk -F\' '/menuentry / {print $2}' /boot/grub2/grub.cfg

# Set default by index
sudo grub2-set-default 0

# Set default by title
sudo grub2-set-default "Fedora (6.5.0-200.fc39.x86_64) 39 (Thirty Nine)"

# Set for next boot only
sudo grub2-reboot 0
```

## initramfs

### Purpose

```
initramfs (initial RAM filesystem):
  Temporary root filesystem loaded into memory during boot.
  Contains kernel modules and tools needed to mount the REAL root filesystem.

Why needed:
  - Root filesystem might be on LVM, RAID, iSCSI, NFS, LUKS
  - Kernel doesn't have those drivers built-in (they're modules)
  - initramfs provides the drivers to access root, then pivots to it

Contents:
  /init                 ← init script (systemd or custom shell script)
  /lib/modules/         ← kernel modules (storage, filesystem, crypto)
  /usr/lib/systemd/     ← systemd (if using systemd in initramfs)
  /usr/bin/, /usr/sbin/  ← tools (mount, lvm, mdadm, cryptsetup)
  /etc/                 ← minimal config (fstab, crypttab, etc.)
```

### dracut (RHEL/Fedora)

```bash
# Rebuild initramfs for current kernel
sudo dracut --force

# Rebuild for specific kernel
sudo dracut --force /boot/initramfs-5.14.0-362.el9.x86_64.img 5.14.0-362.el9.x86_64

# Rebuild with verbose output
sudo dracut --force --verbose

# Add specific modules
sudo dracut --force --add "lvm crypt mdraid"

# List contents of initramfs
lsinitrd /boot/initramfs-$(uname -r).img
lsinitrd /boot/initramfs-$(uname -r).img | grep -i nvme

# Extract initramfs
mkdir /tmp/initrd && cd /tmp/initrd
/usr/lib/dracut/skipcpio /boot/initramfs-$(uname -r).img | zcat | cpio -idmv

# dracut configuration
cat /etc/dracut.conf.d/*.conf
# add_dracutmodules+=" lvm crypt "
# add_drivers+=" nvme "
# hostonly="yes"                             # only include host-specific modules
# hostonly="no"                              # include all (rescue/portable)
```

### mkinitramfs / update-initramfs (Debian/Ubuntu)

```bash
# Rebuild initramfs for current kernel
sudo update-initramfs -u

# Rebuild for specific kernel
sudo update-initramfs -u -k 6.1.0-23-amd64

# Rebuild all
sudo update-initramfs -u -k all

# Create new initramfs
sudo mkinitramfs -o /boot/initrd.img-$(uname -r) $(uname -r)

# List contents
lsinitramfs /boot/initrd.img-$(uname -r)

# Configuration
cat /etc/initramfs-tools/initramfs.conf
# MODULES=most                               # most (default) | dep | list
```

## Kernel Loading

### Kernel files

```bash
ls /boot/
# vmlinuz-6.5.0-200.fc39.x86_64             ← compressed kernel image
# initramfs-6.5.0-200.fc39.x86_64.img       ← initramfs image
# System.map-6.5.0-200.fc39.x86_64          ← kernel symbol table
# config-6.5.0-200.fc39.x86_64              ← kernel build configuration

# vmlinuz = compressed (gzip/lzma/zstd) kernel binary
# GRUB decompresses it into memory and jumps to entry point
```

### Kernel boot parameters at runtime

```bash
# View current kernel command line
cat /proc/cmdline
# BOOT_IMAGE=(hd0,gpt2)/vmlinuz-6.5.0 root=/dev/mapper/rl-root ro rhgb quiet

# View all kernel parameters and their descriptions
sysctl -a                                    # runtime parameters (different from boot params)
```

## systemd Boot Targets

### Target hierarchy

```bash
# List available targets
systemctl list-units --type=target

# Current default target
systemctl get-default

# Common targets
multi-user.target                            # text mode, networking, all services
graphical.target                             # multi-user + display manager (GUI)
rescue.target                                # single-user, minimal services, root password
emergency.target                             # root shell, no services, read-only root fs
```

### Change default target

```bash
# Set default boot target
sudo systemctl set-default multi-user.target
sudo systemctl set-default graphical.target

# Switch target at runtime (no reboot)
sudo systemctl isolate multi-user.target     # drop to text mode
sudo systemctl isolate graphical.target      # start GUI
sudo systemctl isolate rescue.target         # rescue mode
```

### Boot to specific target (one-time)

```bash
# At GRUB menu, press 'e' to edit, add to linux line:
systemd.unit=rescue.target                   # rescue mode
systemd.unit=emergency.target                # emergency mode
systemd.unit=multi-user.target               # text mode (skip GUI)

# Or from running system:
sudo systemctl rescue                        # switch to rescue
sudo systemctl emergency                     # switch to emergency
```

### Target dependencies

```bash
# View target dependencies (what starts for a target)
systemctl list-dependencies multi-user.target
systemctl list-dependencies graphical.target

# Dependency chain:
# graphical.target
# └── multi-user.target
#     └── basic.target
#         └── sysinit.target
#             └── local-fs.target
#                 └── local-fs-pre.target
```

## systemd-boot (Alternative Bootloader)

```bash
# systemd-boot (formerly gummiboot) — simpler UEFI-only bootloader
# Used by default on some Arch, Clear Linux, and Pop!_OS installations

# Install systemd-boot
sudo bootctl install                         # installs to ESP

# Configuration
cat /boot/efi/loader/loader.conf
# default  fedora.conf
# timeout  5
# console-mode max

# Boot entries
ls /boot/efi/loader/entries/
cat /boot/efi/loader/entries/fedora.conf
# title    Fedora Linux 39
# linux    /vmlinuz-6.5.0-200.fc39.x86_64
# initrd   /initramfs-6.5.0-200.fc39.x86_64.img
# options  root=/dev/mapper/rl-root ro rhgb quiet

# Update systemd-boot
sudo bootctl update
```

## Root Filesystem Mount

```bash
# Mount process during boot:
# 1. initramfs /init (or systemd) runs
# 2. Loads storage drivers (SCSI, NVMe, USB, etc.)
# 3. Activates LVM (vgchange -ay), assembles RAID (mdadm)
# 4. Decrypts LUKS (cryptsetup luksOpen)
# 5. Mounts root filesystem (mount -o ro /dev/mapper/vg-root /sysroot)
# 6. pivot_root /sysroot (or switch_root)
# 7. systemd remounts root read-write later (remount-fs.service)

# Check current root mount
findmnt /
mount | grep " / "

# Root mount options in fstab
cat /etc/fstab | grep " / "
# /dev/mapper/rl-root  /  xfs  defaults  0  0
```

## Troubleshooting Boot Failures

### rd.break (initramfs breakpoint)

```bash
# At GRUB menu, press 'e', add to linux line:
rd.break

# Drops to initramfs shell BEFORE root is mounted
# Useful for: resetting root password, fixing fstab, fixing LVM

# Reset root password from rd.break:
switch_root:/# mount -o remount,rw /sysroot
switch_root:/# chroot /sysroot
sh-5.1# passwd root
sh-5.1# touch /.autorelabel                 # SELinux relabel
sh-5.1# exit
switch_root:/# exit
# System continues boot
```

### init=/bin/bash

```bash
# At GRUB menu, press 'e', replace 'ro' with 'rw init=/bin/bash'
# Boots directly to bash shell as PID 1 (no systemd)

bash-5.1# mount -o remount,rw /              # if not already rw
bash-5.1# passwd root
bash-5.1# touch /.autorelabel
bash-5.1# exec /sbin/init                    # or reboot: exec /sbin/reboot -f
```

### Emergency and rescue targets

```bash
# Emergency target: minimal — root shell, root fs read-only
# Add to kernel cmdline: systemd.unit=emergency.target
# Or: at GRUB, press 'e', add: emergency
# Must remount root: mount -o remount,rw /

# Rescue target: more services — networking may be available
# Add to kernel cmdline: systemd.unit=rescue.target
# Or: at GRUB, press 'e', add: rescue / single / s / 1
```

### Boot analysis

```bash
# Boot time analysis
systemd-analyze                              # total boot time
systemd-analyze blame                        # per-unit boot time (sorted)
systemd-analyze critical-chain               # critical path
systemd-analyze plot > boot.svg              # visual boot chart

# Find slow services
systemd-analyze blame | head -20

# Show what delayed boot
systemd-analyze critical-chain graphical.target

# Check for failed units
systemctl --failed
systemctl list-units --state=failed
```

### Common boot failures

```bash
# GRUB "error: unknown filesystem"
# → Reinstall GRUB (see GRUB rescue section above)

# "Kernel panic - not syncing: VFS: Unable to mount root fs"
# → Wrong root= parameter, missing storage drivers in initramfs
# Fix: boot from rescue media, rebuild initramfs with correct modules

# "Give root password for maintenance"
# → fstab error (bad mount point, wrong UUID)
# Fix: enter root password, edit /etc/fstab, reboot

# "A start job is running for..." (boot hangs)
# → Network wait or slow service
# Fix: check systemctl status, reduce timeouts, mask service

# Filesystem corruption
# Boot to rescue, run fsck:
sudo fsck /dev/sda2                          # ext4
sudo xfs_repair /dev/sda2                    # XFS (unmount first)

# UEFI boot entry lost
sudo efibootmgr -c -d /dev/sda -p 1 -l '\EFI\fedora\shimx64.efi' -L "Fedora"
```

## See Also

- grub
- systemd
- dracut
- kernel
- dmesg
- journalctl

## References

- Red Hat System Administrator's Guide: Configuring the Boot Loader (RHEL 9)
- Arch Wiki: Boot Process (https://wiki.archlinux.org/title/Arch_boot_process)
- man dracut(8), man dracut.conf(5)
- man grub2-mkconfig(8), man grub2-install(8)
- man systemd-analyze(1), man bootctl(1)
- man efibootmgr(8), man mokutil(1)
- UEFI Specification: https://uefi.org/specifications
