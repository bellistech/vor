# GRUB2 (Grand Unified Bootloader)

> Boot loader for Linux and other operating systems; handles kernel selection, boot parameters, and chainloading.

## Configuration

### /etc/default/grub

```bash
# Default boot entry (0-indexed, or saved with GRUB_DEFAULT=saved)
GRUB_DEFAULT=0
GRUB_DEFAULT=saved                       # Use last booted entry

# Timeout before auto-boot (seconds)
GRUB_TIMEOUT=5
GRUB_TIMEOUT_STYLE=menu                  # menu, countdown, hidden

# Kernel command line parameters
GRUB_CMDLINE_LINUX_DEFAULT="quiet splash"   # Normal boot
GRUB_CMDLINE_LINUX=""                       # Applied to ALL entries (including recovery)

# Console settings
GRUB_TERMINAL=console                    # Text-only (no graphical)
GRUB_GFXMODE=1920x1080x32               # Graphical resolution
GRUB_GFXPAYLOAD_LINUX=keep              # Keep resolution for kernel

# Disable OS prober (security, speed)
GRUB_DISABLE_OS_PROBER=true

# Enable submenu for old kernels
GRUB_DISABLE_SUBMENU=false

# Disable recovery entries
GRUB_DISABLE_RECOVERY=false
```

### Apply Configuration Changes

```bash
# Regenerate grub.cfg (Debian/Ubuntu)
sudo update-grub

# Regenerate grub.cfg (RHEL/Fedora/Arch)
sudo grub-mkconfig -o /boot/grub/grub.cfg

# UEFI systems may use
sudo grub-mkconfig -o /boot/efi/EFI/<distro>/grub.cfg
```

## Installation

### grub-install

```bash
# BIOS/MBR install
sudo grub-install /dev/sda

# UEFI install
sudo grub-install --target=x86_64-efi --efi-directory=/boot/efi --bootloader-id=GRUB

# Reinstall after boot repair
sudo grub-install --recheck /dev/sda

# Force install (override safety checks)
sudo grub-install --force /dev/sda

# Install to removable media (USB)
sudo grub-install --target=x86_64-efi --efi-directory=/mnt/usb/efi --removable
```

## Kernel Parameters

### Common Boot Parameters

```bash
# Display and graphics
quiet                   # Suppress most boot messages
splash                  # Show splash screen
nomodeset               # Disable kernel mode setting (GPU fallback)
vga=normal              # Force VGA text mode

# Init and runlevel
init=/bin/bash           # Boot to root shell (bypass init)
single                   # Single-user mode (SysVinit)
systemd.unit=rescue.target      # Rescue mode (systemd)
systemd.unit=emergency.target   # Emergency mode (minimal)
systemd.unit=multi-user.target  # Text-mode multi-user

# Filesystem
root=/dev/sda2           # Root partition
ro                       # Mount root read-only (default)
rw                       # Mount root read-write

# Debug and troubleshooting
debug                    # Verbose kernel messages
loglevel=7               # Max kernel log verbosity (0-7)
earlyprintk=vga          # Early boot messages on screen
nokaslr                  # Disable kernel address space randomization
nosmp                    # Disable SMP (single CPU)
noapic                   # Disable APIC (interrupt controller issues)
acpi=off                 # Disable ACPI (power management issues)
pci=noacpi               # PCI without ACPI routing
iommu=soft               # Software IOMMU (virtualization issues)
```

### Editing Parameters at Boot

```
1. At GRUB menu, highlight entry and press 'e' to edit
2. Find the line starting with 'linux' or 'linuxefi'
3. Add/modify parameters at end of that line
4. Press Ctrl+X or F10 to boot with changes
5. Changes are temporary (one-time only)
```

## Menu Entry Syntax

### Custom Entry in /etc/grub.d/40_custom

```bash
#!/bin/sh
exec tail -n +3 $0

menuentry "My Custom Kernel" {
    set root='hd0,msdos1'
    linux /vmlinuz-custom root=/dev/sda2 ro quiet
    initrd /initramfs-custom.img
}

menuentry "Memory Test (memtest86+)" {
    linux16 /boot/memtest86+.bin
}
```

### Submenu

```bash
submenu "Advanced Options" {
    menuentry "Kernel 6.1" {
        set root='hd0,gpt2'
        linux /vmlinuz-6.1.0 root=/dev/sda2 ro
        initrd /initrd.img-6.1.0
    }
    menuentry "Kernel 5.15" {
        set root='hd0,gpt2'
        linux /vmlinuz-5.15.0 root=/dev/sda2 ro
        initrd /initrd.img-5.15.0
    }
}
```

## Recovery

### Boot Repair from GRUB Shell

```bash
# If GRUB drops to grub> prompt
ls                           # List partitions
ls (hd0,gpt2)/               # Browse partition contents
set root=(hd0,gpt2)
linux /vmlinuz root=/dev/sda2 ro
initrd /initrd.img
boot

# If GRUB drops to grub rescue>
set prefix=(hd0,gpt2)/boot/grub
insmod normal
normal
```

### Reinstall from Live USB

```bash
# Mount target root
sudo mount /dev/sda2 /mnt
sudo mount /dev/sda1 /mnt/boot/efi    # UEFI only
sudo mount --bind /dev /mnt/dev
sudo mount --bind /proc /mnt/proc
sudo mount --bind /sys /mnt/sys

# Chroot and reinstall
sudo chroot /mnt
grub-install /dev/sda                  # BIOS
grub-install --target=x86_64-efi --efi-directory=/boot/efi   # UEFI
update-grub
exit

# Unmount
sudo umount -R /mnt
```

## Password Protection

### Set GRUB Password

```bash
# Generate password hash
grub-mkpasswd-pbkdf2
# Enter password, copy the hash

# Add to /etc/grub.d/40_custom
cat <<'GRUB'
set superusers="admin"
password_pbkdf2 admin grub.pbkdf2.sha512.10000.<hash>
GRUB

# Restrict specific entries (in menuentry)
menuentry "Secure Kernel" --users admin {
    ...
}

# Unrestricted entries (anyone can boot, only admin can edit)
menuentry "Normal Boot" --unrestricted {
    ...
}
```

```bash
# Regenerate config
sudo update-grub
```

## Chainloading

### Boot Another OS or Bootloader

```bash
# Chainload Windows (BIOS/MBR)
menuentry "Windows" {
    set root='hd0,msdos1'
    chainloader +1
}

# Chainload Windows (UEFI)
menuentry "Windows" {
    search --set=root --file /EFI/Microsoft/Boot/bootmgfw.efi
    chainloader /EFI/Microsoft/Boot/bootmgfw.efi
}
```

## UEFI vs BIOS

```
Feature          BIOS/MBR               UEFI/GPT
-------          --------               --------
Partition table  MBR (4 primary max)    GPT (128+ partitions)
Boot partition   First sector of disk   EFI System Partition (ESP)
GRUB target      i386-pc                x86_64-efi
Install command  grub-install /dev/sda  grub-install --target=x86_64-efi
Config location  /boot/grub/grub.cfg    /boot/efi/EFI/<distro>/grub.cfg
Max disk size    2 TB                   9.4 ZB
Secure Boot      Not supported          Supported
```

## Tips

- Always run `update-grub` (or `grub-mkconfig`) after editing `/etc/default/grub` or files in `/etc/grub.d/`.
- Never edit `/boot/grub/grub.cfg` directly; it is auto-generated and will be overwritten.
- Use `GRUB_DEFAULT=saved` with `grub-set-default <N>` or `grub-reboot <N>` to control the default entry.
- Keep at least one known-good kernel entry; do not remove old kernels until a new one is confirmed working.
- For headless servers, set `GRUB_TERMINAL=serial` and add `GRUB_SERIAL_COMMAND` for serial console access.
- If Secure Boot is enabled, GRUB must be signed with a trusted key (shim-signed on Ubuntu/Fedora).

## References

- [GNU GRUB Manual](https://www.gnu.org/software/grub/manual/grub/)
- [GRUB Command Reference](https://www.gnu.org/software/grub/manual/grub/grub.html#Commands)
- [GRUB Environment Variables](https://www.gnu.org/software/grub/manual/grub/grub.html#Environment)
- [Arch Wiki — GRUB](https://wiki.archlinux.org/title/GRUB)
- [Arch Wiki — GRUB Tips and Tricks](https://wiki.archlinux.org/title/GRUB/Tips_and_tricks)
- [Ubuntu GRUB2 Documentation](https://help.ubuntu.com/community/Grub2)
- [Red Hat — Configuring GRUB2 Bootloader](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_boot_loader/configuring-grub2_managing-boot-loader)
- [Kernel Boot Parameters](https://www.kernel.org/doc/html/latest/admin-guide/kernel-parameters.html)
- [Kernel EFI Stub Documentation](https://www.kernel.org/doc/html/latest/admin-guide/efi-stub.html)
