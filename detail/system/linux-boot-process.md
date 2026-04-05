# Linux Boot Process — UEFI Firmware, Secure Boot Chain, Kernel Init, and systemd PID 1 Sequence

> *The Linux boot process is a carefully orchestrated handoff across firmware, bootloader, kernel, initramfs, and init system. UEFI firmware initializes hardware and locates the ESP, the bootloader loads the kernel and initramfs into memory, the kernel decompresses and performs early hardware initialization, initramfs provides the minimal environment to mount the real root filesystem, and systemd (PID 1) brings the system to a usable state through dependency-ordered unit activation. Each stage validates the next through Secure Boot's chain of trust.*

---

## 1. UEFI Boot Process (Firmware to ESP to Bootloader)

### UEFI Firmware Initialization

The UEFI firmware replaces the legacy BIOS with a sophisticated boot environment:

```
UEFI firmware boot phases (PI specification):

  Phase 1: SEC (Security)
    - CPU comes out of reset in real mode
    - SEC code is in firmware flash (SPI NOR)
    - Initializes temporary RAM (CAR — Cache as RAM)
    - Validates PEI volume integrity
    - Transfers control to PEI

  Phase 2: PEI (Pre-EFI Initialization)
    - Discovers and initializes main memory (DRAM training)
    - Loads PEI modules (PEIMs) from firmware volume
    - Creates HOB (Hand-Off Block) list describing hardware
    - Memory is now available
    - Transfers control to DXE

  Phase 3: DXE (Driver Execution Environment)
    - Loads and executes DXE drivers (storage, network, USB, GPU)
    - Installs UEFI protocols (block I/O, file system, etc.)
    - Builds UEFI system table
    - ~50-200 drivers loaded (hardware-dependent)

  Phase 4: BDS (Boot Device Selection)
    - Reads boot variables (BootOrder, Boot0001, etc.) from NVRAM
    - Attempts to load each boot option in order
    - First successful load → transfer control to bootloader

  Phase 5: TSL (Transient System Load)
    - Bootloader runs in UEFI environment
    - Has access to UEFI services (file I/O, memory allocation, etc.)
    - Loads OS kernel
    - Calls ExitBootServices() → firmware releases control of hardware

  Phase 6: RT (Runtime)
    - Only UEFI runtime services remain (time, variables, reset)
    - OS kernel has full control
```

### ESP (EFI System Partition)

```
ESP requirements:
  Filesystem:  FAT32 (required by UEFI spec)
  Size:        200-500 MB typical (260 MB minimum recommended)
  GPT type:    C12A7328-F81F-11D2-BA4B-00A0C93EC93B
  Mount point: /boot/efi (Linux convention)
  Flags:       boot, esp

Why FAT32:
  - UEFI firmware must read the filesystem WITHOUT any OS drivers
  - FAT32 is simple enough to implement in firmware
  - Every UEFI implementation includes a FAT32 driver
  - No UEFI firmware supports ext4, XFS, btrfs natively

ESP file layout (multi-boot example):
  /EFI/
  ├── BOOT/
  │   └── BOOTX64.EFI           ← removable media fallback path
  │                                (used when no NVRAM boot entry exists)
  ├── fedora/
  │   ├── shimx64.efi           ← Secure Boot shim (Microsoft-signed)
  │   ├── grubx64.efi           ← GRUB2 (Fedora-signed)
  │   ├── grub.cfg              ← GRUB config stub (points to /boot)
  │   └── fonts/                ← GRUB display fonts
  ├── ubuntu/
  │   ├── shimx64.efi
  │   ├── grubx64.efi
  │   └── grub.cfg
  └── Microsoft/
      └── Boot/
          ├── bootmgfw.efi      ← Windows Boot Manager
          ├── BCD                ← Boot Configuration Data
          └── memtest.efi

NVRAM boot variables (stored in firmware flash):
  BootOrder:  0001,0003,0000    (try Fedora, then Ubuntu, then Windows)
  Boot0000:   "Windows Boot Manager" → HD(1,GPT,...)/EFI/Microsoft/Boot/bootmgfw.efi
  Boot0001:   "Fedora"          → HD(1,GPT,...)/EFI/fedora/shimx64.efi
  Boot0003:   "ubuntu"          → HD(1,GPT,...)/EFI/ubuntu/shimx64.efi
```

## 2. Secure Boot Chain of Trust

### Certificate Hierarchy

```
Secure Boot trust chain:

  ┌─────────────────────────────────────────────────────────┐
  │  OEM (e.g., Dell, Lenovo, HP)                           │
  │  Platform Key (PK)                                      │
  │  └── Establishes firmware ownership                     │
  │      Only PK holder can modify KEK database             │
  └─────────────────────┬───────────────────────────────────┘
                        │ signs
  ┌─────────────────────▼───────────────────────────────────┐
  │  Key Exchange Key (KEK)                                  │
  │  ├── Microsoft KEK (Microsoft Corporation KEK CA 2011)  │
  │  └── OEM KEK (optional)                                 │
  │      KEK holders can modify db/dbx                      │
  └─────────────────────┬───────────────────────────────────┘
                        │ authorizes modifications to
  ┌─────────────────────▼───────────────────────────────────┐
  │  Signature Database (db)                                 │
  │  ├── Microsoft Windows Production PCA 2011               │
  │  │   └── Signs: Windows bootloader, drivers              │
  │  ├── Microsoft Corporation UEFI CA 2011                  │
  │  │   └── Signs: third-party bootloaders (shim, etc.)     │
  │  └── Custom certificates (enrolled by user via MOK)      │
  ├──────────────────────────────────────────────────────────┤
  │  Forbidden Database (dbx)                                │
  │  └── Revoked signatures, vulnerable bootloaders          │
  │      (updated periodically by Microsoft via Windows      │
  │       Update or Linux fwupd)                             │
  └──────────────────────────────────────────────────────────┘
```

### Shim Validation Flow

```
Secure Boot validation with shim:

  UEFI firmware
    │
    ├── Load shimx64.efi from ESP
    │   ├── Check signature against db (UEFI Signature Database)
    │   │   └── Signed by "Microsoft Corporation UEFI CA 2011" → PASS
    │   └── shimx64.efi runs
    │
    ├── shim loads grubx64.efi
    │   ├── Check signature against:
    │   │   1. Distro vendor key (embedded in shim binary)
    │   │   2. MOK (Machine Owner Keys, enrolled by user)
    │   │   3. UEFI db
    │   │   └── Signed by "Red Hat Secure Boot CA" → PASS
    │   └── grubx64.efi runs
    │
    ├── GRUB loads vmlinuz
    │   ├── GRUB calls shim's verification protocol
    │   │   └── Signed by distro vendor → PASS
    │   └── Kernel starts
    │
    └── Kernel loads modules
        ├── Each .ko checked against:
        │   1. Built-in kernel signing key
        │   2. MOK keys (if enrolled)
        │   └── Unsigned module → REFUSED (if lockdown enforced)
        └── All modules loaded

MOK enrollment (user-interactive):
  1. mokutil --import MOK.der → queues key for enrollment
  2. Reboot → shim detects pending enrollment
  3. MokManager runs (blue/text screen)
  4. User enters password set during --import
  5. Key added to MOK database (stored in UEFI variable)
  6. Subsequent boots: shim trusts MOK-signed binaries
```

### Secure Boot and Kernel Lockdown

```
When Secure Boot is active, the kernel enables lockdown mode:

  Lockdown = integrity mode (default when Secure Boot on):
    Blocked operations:
    ├── /dev/mem and /dev/kmem access (no raw memory reads)
    ├── /dev/port access
    ├── kexec of unsigned kernels
    ├── Hibernation to unsigned swap
    ├── Custom ACPI tables (via acpi_rsdp=, initrd ACPI override)
    ├── ioperm / iopl (direct port I/O)
    ├── bpf_read() of kernel memory
    └── unsigned module loading

  Lockdown = confidentiality mode (optional, stricter):
    Everything in integrity mode, plus:
    ├── /proc/kcore access
    ├── perf counters
    └── Kernel tracing (ftrace, kprobes)

  Check lockdown status:
    cat /sys/kernel/security/lockdown
    # [none] integrity confidentiality
```

## 3. GRUB2 Stage Loading

### GRUB2 on BIOS Systems

```
BIOS/MBR GRUB2 stages:

  Stage 1 (boot.img, 440 bytes):
    - Installed in MBR bootstrap area
    - Only job: load stage 1.5
    - Contains disk address of stage 1.5
    - 440 bytes is too small for any filesystem code

  Stage 1.5 (core.img, ~32 KB):
    - Installed in "MBR gap" (sectors 1-2047, between MBR and first partition)
    - Or: installed in BIOS Boot Partition (GPT systems without ESP)
    - Contains: minimal filesystem driver (ext4, XFS, etc.) + GRUB core
    - Can read /boot/grub2/ directory
    - Loads stage 2 modules

  Stage 2 (normal.mod + modules):
    - /boot/grub2/ directory on disk
    - normal.mod: command parser, menu system
    - Modules loaded on demand: linux.mod, ext2.mod, gfxterm.mod, etc.
    - Reads /boot/grub2/grub.cfg for menu configuration
    - Presents menu, loads selected kernel + initrd
```

### GRUB2 on UEFI Systems

```
UEFI GRUB2 (single-stage):
  - grubx64.efi is a single UEFI application (~1-3 MB)
  - Contains GRUB core + essential modules (compiled in)
  - Uses UEFI firmware services for disk I/O (no raw disk access)
  - Reads grub.cfg from ESP or from /boot partition
  - No MBR gap or BIOS Boot Partition needed

GRUB2 kernel loading sequence:
  1. Parse grub.cfg → build menu
  2. User selects entry (or timeout selects default)
  3. Execute commands:
     linux /vmlinuz-6.5.0 root=/dev/mapper/rl-root ro quiet
     initrd /initramfs-6.5.0.img
     boot
  4. GRUB allocates memory for kernel + initrd (via UEFI AllocatePages)
  5. Loads vmlinuz at allocated address
  6. Loads initramfs at separate allocated address
  7. Sets up boot parameters (command line, initrd location, etc.)
  8. Calls ExitBootServices() (UEFI releases hardware control)
  9. Jumps to kernel entry point
```

## 4. initramfs Purpose and Content

### Why initramfs Exists

```
The chicken-and-egg problem:

  Kernel needs storage drivers to mount root filesystem
  Storage drivers are kernel modules (.ko files)
  Kernel modules are stored on the root filesystem
  But root filesystem isn't mounted yet!

Solution: initramfs (initial RAM filesystem)
  A compressed archive loaded into RAM alongside the kernel
  Contains just enough to find and mount the real root filesystem

Without initramfs, you'd need:
  - Every possible storage driver compiled into the kernel (vmlinuz)
  - Every possible filesystem driver compiled in
  - LVM, RAID, LUKS, multipath, iSCSI support compiled in
  - This would make the kernel enormous and inflexible

With initramfs:
  - Kernel is minimal (generic)
  - initramfs is machine-specific (dracut hostonly mode)
  - Contains only the modules needed for THIS machine's root device
```

### initramfs Internal Structure

```
initramfs contents (dracut/systemd-based):

  /init → /usr/lib/systemd/systemd          ← PID 1 inside initramfs
  /etc/
  ├── fstab                                  ← minimal (root entry only)
  ├── crypttab                               ← LUKS device mappings
  ├── lvm/lvm.conf                           ← LVM configuration
  └── modprobe.d/                            ← module loading config
  /usr/
  ├── lib/
  │   ├── modules/6.5.0/                     ← kernel modules
  │   │   ├── kernel/drivers/scsi/           ← SCSI drivers
  │   │   ├── kernel/drivers/nvme/           ← NVMe drivers
  │   │   ├── kernel/drivers/md/             ← RAID drivers
  │   │   ├── kernel/drivers/block/          ← block device drivers
  │   │   ├── kernel/fs/ext4/               ← filesystem drivers
  │   │   ├── kernel/fs/xfs/                ← XFS driver
  │   │   └── kernel/crypto/                ← crypto for LUKS
  │   ├── systemd/
  │   │   ├── systemd                        ← systemd binary
  │   │   └── system/
  │   │       ├── initrd.target              ← initramfs boot target
  │   │       ├── initrd-root-fs.target      ← root mounted
  │   │       ├── sysroot.mount             ← mount real root
  │   │       └── initrd-switch-root.service ← pivot to real root
  │   └── udev/                              ← udev rules + helpers
  ├── bin/
  │   ├── mount, umount
  │   ├── lvm                                ← LVM tools
  │   ├── mdadm                              ← RAID assembly
  │   ├── cryptsetup                         ← LUKS decryption
  │   └── systemctl
  └── sbin/
      └── modprobe, insmod                   ← module loading

initramfs boot flow (with systemd):
  1. Kernel unpacks initramfs into rootfs (tmpfs at /)
  2. Kernel executes /init (→ systemd)
  3. systemd starts initrd.target:
     a. Load kernel modules (udev triggers)
     b. Start udevd (device detection)
     c. Activate LVM: lvm vgchange -ay
     d. Assemble RAID: mdadm --assemble
     e. Decrypt LUKS: cryptsetup luksOpen
     f. Mount root: mount /dev/mapper/root /sysroot
  4. initrd-root-fs.target reached (root is mounted)
  5. switch_root /sysroot /usr/lib/systemd/systemd
     → PID 1 re-executes itself from real root filesystem
     → initramfs tmpfs is freed
```

### switch_root / pivot_root

```
Transition from initramfs to real root:

  switch_root (used by modern systems):
    1. Delete everything in initramfs (free memory)
    2. mount --move /sysroot /
    3. chroot to new root
    4. exec new /sbin/init (PID 1 replaces itself)

  pivot_root (older method):
    1. pivot_root /sysroot /sysroot/initrd
    2. Old root mounted at /initrd
    3. chroot to new root
    4. umount /initrd and rmdir /initrd

  switch_root is preferred because:
    - Atomically frees initramfs memory
    - Simpler (no old root to clean up)
    - Used by both systemd and dracut
```

## 5. Kernel Early Init Sequence

### From Entry Point to PID 1

```
Kernel boot sequence (after GRUB hands off control):

  1. Decompression
     - vmlinuz contains: decompressor stub + compressed kernel
     - Stub runs, decompresses kernel into memory
     - Compression: gzip, lzma, xz, lzo, lz4, zstd
     - Typical: 12 MB compressed → 60 MB decompressed

  2. Architecture-specific init (arch/x86/kernel/head_64.S)
     - Enable paging (virtual memory)
     - Set up initial page tables (identity mapping)
     - Switch to 64-bit long mode
     - Set up GDT (Global Descriptor Table)
     - Jump to start_kernel()

  3. start_kernel() (init/main.c) — the "real" kernel init
     - setup_arch()         → architecture-specific setup
     - mm_init()            → memory management initialization
     - sched_init()         → scheduler initialization
     - irq_init()           → interrupt handling
     - time_init()          → timekeeping
     - console_init()       → early console (serial, VGA)
     - vfs_caches_init()    → VFS initialization
     - page_cache_init()    → page cache
     - fork_init()          → process creation infrastructure
     - ...dozens more subsystem inits...

  4. rest_init()
     - Creates kernel thread for kernel_init() (becomes PID 1)
     - Creates kernel thread for kthreadd (becomes PID 2)
     - Idle loop (PID 0 becomes the idle task)

  5. kernel_init() (PID 1 in kernel space)
     - wait_for_initramfs() → unpack initramfs
     - do_initcalls()       → run built-in module init functions
     - Run /init from initramfs (or /sbin/init, /etc/init, /bin/init, /bin/sh)
     - kernel_init_freeable() execve's into userspace /init
     - PID 1 is now a userspace process (systemd)
```

### Built-in vs Module Init

```
Device initialization order:

  Built-in drivers (compiled into vmlinuz):
    Initialized during do_initcalls() in kernel_init()
    Order determined by initcall levels:
      0: early_initcall      — earliest possible (architecture)
      1: pure_initcall       — no dependencies
      2: core_initcall       — core subsystems
      3: postcore_initcall   — after core
      4: arch_initcall       — architecture-specific
      5: subsys_initcall     — subsystem initialization
      6: fs_initcall         — filesystem drivers
      7: device_initcall     — most device drivers (DEFAULT)
      8: late_initcall       — everything else

  Loadable modules (.ko files):
    Loaded later by udev (in initramfs or after root mount)
    Order determined by: udev rules, module dependencies, device detection
    modprobe resolves dependencies: modules.dep
```

## 6. systemd PID 1 Boot Sequence

### systemd Startup

```
systemd PID 1 execution (after switch_root from initramfs):

  1. Re-execute (if switching from initramfs systemd)
     - systemd recognizes it's post-switch_root
     - Deserializes state from initramfs systemd
     - Continues with real-root configuration

  2. Load configuration
     - /etc/systemd/system.conf         ← global manager config
     - /etc/systemd/system/             ← admin unit overrides
     - /usr/lib/systemd/system/         ← vendor unit files
     - Determine default target: systemctl get-default
       (symlink: /etc/systemd/system/default.target)

  3. Build transaction
     - Calculate dependency tree for default.target
     - Resolve Wants=, Requires=, Before=, After= relationships
     - Build ordered list of units to start
     - Detect cycles, handle conflicts

  4. Execute transaction (parallel where possible)
     - Start units respecting ordering (After=/Before=)
     - Units with no ordering constraints start in PARALLEL
     - This is the primary source of systemd's boot speed advantage

Boot target dependency chain:

  default.target = graphical.target
    Wants: display-manager.service (GDM, SDDM, etc.)
    Requires: multi-user.target
      Wants: all enabled services (sshd, httpd, crond, etc.)
      Requires: basic.target
        Requires: sockets.target (all .socket units)
        Requires: timers.target (all .timer units)
        Requires: paths.target (all .path units)
        Requires: slices.target (cgroup slices)
        Requires: sysinit.target
          Wants: systemd-tmpfiles-setup.service
          Wants: systemd-sysctl.service
          Wants: systemd-modules-load.service
          Requires: local-fs.target
            After: local-fs-pre.target
            Wants: -.mount (root), /boot.mount, /home.mount, etc.
            Wants: systemd-remount-fs.service (remount root rw)
          Requires: swap.target
            Wants: all swap units
```

### Parallel Startup

```
systemd parallelism model:

  Traditional SysVinit: sequential (S01, S02, ... S99)
    S01network → S02syslog → S03sshd → S04httpd → ...
    Total time: sum of all service startup times

  systemd: dependency-based parallel
    ┌─────────────┐  ┌──────────────┐
    │ sshd.service │  │ httpd.service│  ← independent: start simultaneously
    └──────┬──────┘  └──────┬───────┘
           │                │
    ┌──────▼──────┐  ┌──────▼───────┐
    │ network     │  │ network      │  ← both need network-online.target
    │ -online     │  │ -online      │
    └──────┬──────┘  └──────┬───────┘
           └────────┬───────┘
           ┌────────▼───────┐
           │ network.target  │  ← single dependency, started once
           └────────────────┘

    Total time: max of parallel chains (not sum)

Socket activation enables even more parallelism:
  - sshd.socket starts immediately (just opens port 22)
  - sshd.service starts only when connection arrives
  - Services that depend on sshd can start without waiting
```

## 7. Target Unit Dependencies

### Target Dependency Tree

```
Complete target dependency tree (graphical boot):

  graphical.target
  │
  ├── multi-user.target
  │   │
  │   ├── basic.target
  │   │   │
  │   │   ├── sockets.target
  │   │   │   └── (all .socket units: dbus.socket, sssd.socket, etc.)
  │   │   │
  │   │   ├── timers.target
  │   │   │   └── (all .timer units: logrotate.timer, fstrim.timer, etc.)
  │   │   │
  │   │   ├── paths.target
  │   │   │   └── (all .path units)
  │   │   │
  │   │   ├── slices.target
  │   │   │   └── (cgroup hierarchy: user.slice, system.slice, machine.slice)
  │   │   │
  │   │   └── sysinit.target
  │   │       │
  │   │       ├── local-fs.target
  │   │       │   ├── -.mount (root filesystem)
  │   │       │   ├── boot.mount, home.mount, etc. (from fstab)
  │   │       │   └── systemd-remount-fs.service (remount root rw)
  │   │       │
  │   │       ├── swap.target
  │   │       │   └── swap units (from fstab)
  │   │       │
  │   │       ├── cryptsetup.target
  │   │       │   └── LUKS device units (from crypttab)
  │   │       │
  │   │       ├── systemd-udevd.service
  │   │       ├── systemd-journald.service
  │   │       ├── systemd-tmpfiles-setup.service
  │   │       ├── systemd-sysctl.service
  │   │       └── systemd-modules-load.service
  │   │
  │   ├── network.target
  │   │   └── NetworkManager.service (or systemd-networkd)
  │   │
  │   ├── dbus.service
  │   │
  │   └── (all enabled services: sshd, httpd, postgres, etc.)
  │
  ├── display-manager.service
  │   └── (gdm.service, sddm.service, or lightdm.service)
  │
  └── (graphical-session-related targets)
```

### Special Targets

```
Target                  Purpose                     Equivalent SysVinit
────────────────────────────────────────────────────────────────────────
poweroff.target         Shut down system            runlevel 0
rescue.target           Single-user, minimal        runlevel 1 / single
multi-user.target       Full system, no GUI         runlevel 3
graphical.target        Full system + GUI           runlevel 5
reboot.target           Reboot system               runlevel 6
emergency.target        Emergency shell (root only) —
halt.target             Halt (no power off)         —
hibernate.target        Hibernate to disk           —
suspend.target          Suspend to RAM              —
```

## 8. Boot Time Optimization

### systemd-analyze

```
Analysis tools:

  systemd-analyze                     # total boot time
  # Startup finished in 2.5s (firmware) + 1.2s (loader) + 1.8s (kernel)
  #                     + 3.2s (initrd) + 8.5s (userspace) = 17.2s

  systemd-analyze blame               # per-unit startup time
  # 5.2s NetworkManager-wait-online.service
  # 2.1s firewalld.service
  # 1.8s docker.service
  # ...

  systemd-analyze critical-chain      # longest dependency chain
  # graphical.target @8.5s
  # └── multi-user.target @8.5s
  #     └── docker.service @6.4s +1.8s
  #         └── network-online.target @6.3s
  #             └── NetworkManager-wait-online.service @1.1s +5.2s

  systemd-analyze plot > boot.svg     # graphical boot chart
  systemd-analyze dot | dot -Tsvg > deps.svg  # dependency graph
```

### Common Optimizations

```
Optimization strategies:

  1. Disable unnecessary services
     systemctl disable bluetooth.service   # if not using Bluetooth
     systemctl mask lvm2-monitor.service   # if not using LVM

  2. Reduce NetworkManager-wait-online timeout
     # /etc/systemd/system/NetworkManager-wait-online.service.d/override.conf
     [Service]
     ExecStart=
     ExecStart=/usr/bin/nm-online -s -q --timeout=10

  3. Use socket activation
     # Service starts on demand, not at boot
     systemctl disable cups.service
     systemctl enable cups.socket          # start only when print job submitted

  4. Parallel fsck
     # /etc/fstab: set fs_passno to 2 for parallel fsck on non-root
     /dev/sda2  /home  ext4  defaults  0  2

  5. Reduce initramfs size (dracut)
     # /etc/dracut.conf.d/minimal.conf
     hostonly="yes"                         # only this host's modules
     hostonly_cmdline="yes"

  6. Use zstd compression for initramfs
     # /etc/dracut.conf.d/compress.conf
     compress="zstd"                        # faster decompression than gzip
```

## 9. Boot Comparison: SysVinit vs Upstart vs systemd

```
Feature               SysVinit          Upstart            systemd
─────────────────────────────────────────────────────────────────────────
Init process          /sbin/init        /sbin/init         /usr/lib/systemd/systemd
                      (shell scripts)   (event-based)      (compiled C, units)

Configuration         /etc/inittab      /etc/init/*.conf   /usr/lib/systemd/system/
                      /etc/init.d/      (event/task stanzas)  *.service, *.target

Startup order         Numeric:          Event-based:       Dependency-based:
                      S01, S02, ...     "start on ..."     After=, Requires=
                      Sequential        Some parallelism   Maximum parallelism

Parallelism           None (serial)     Limited (events)   Full (dependency graph)

Service tracking      PID files         PID + ptrace       cgroups (can't escape)
                      (unreliable)      (reliable)         (most reliable)

Socket activation     inetd (separate)  Built-in           Built-in (systemd)

On-demand start       No (all at boot)  Yes (events)       Yes (socket, path, timer)

Logging               syslog (separate) syslog             journald (structured)

Resource control      ulimits only      ulimits            cgroups (CPU, memory,
                                                           IO, device access)

Status                /etc/init.d/X     initctl status X   systemctl status X
                      status (ad-hoc)   (standardized)     (rich output)

Boot speed            Slow (sequential) Moderate           Fast (parallel +
                      (30-60s typical)  (15-30s typical)   socket activation)
                                                           (5-15s typical)

Shutdown              Kill scripts      Event-based stop   Ordered shutdown via
                      (unreliable       (reliable)         dependency graph
                      ordering)

Adopted by            Legacy systems,   Ubuntu 6.10-15.04  RHEL 7+, Fedora 15+,
                      Slackware, some   (obsolete)         Debian 8+, Ubuntu 15.04+,
                      embedded                             Arch, SUSE 12+

Still used?           Slackware, some   No (fully replaced) Virtually all major
                      embedded, legacy  by systemd)         distributions
                      Gentoo (OpenRC)
```

## See Also

- grub
- systemd
- dracut
- kernel
- dmesg
- journalctl

## References

- UEFI Specification: https://uefi.org/specifications
- UEFI Platform Initialization Specification (PI)
- systemd Documentation: https://systemd.io/
- man systemd-analyze(1), man bootup(7), man dracut(8)
- Linux Kernel Documentation: Documentation/admin-guide/kernel-parameters.txt
- Lennart Poettering: "Rethinking PID 1" (systemd design document)
- Michael Kerrisk: "The Linux Programming Interface" (Chapter 37: Daemons)
