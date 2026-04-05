# Junos OS Software Management --- Deep Dive

> Detailed coverage of the Junos boot process, package validation, upgrade planning,
> downgrade procedures, and storage management. Companion to the cheat sheet at
> `sheets/juniper/junos-software.md`.

## 1. Boot Process Internals

The Junos boot sequence follows a deterministic chain from hardware initialization through daemon startup.

### 1.1 BIOS / Firmware

- Hardware-level power-on self test (POST) runs first
- Checks CPU, memory, storage controllers, and PCI buses
- POST failures generate LED codes or console output before any software loads
- On EX/QFX switches, the BIOS also initializes the switch fabric ASIC

### 1.2 Boot Loader

- Junos uses a FreeBSD-derived boot loader stored on the boot device
- The loader reads `/boot/defaults/loader.conf` and `/boot/loader.conf` for boot parameters
- Presents a brief prompt (default 5-second timeout) where the operator can:
  - Press **Space** to interrupt and enter the loader menu
  - Boot into single-user mode with `boot -s`
  - Select an alternate boot partition
- The loader selects the active root partition (primary or backup) and loads the kernel

### 1.3 Kernel

- FreeBSD kernel initializes the operating environment
- Mounts the root filesystem (UFS on older platforms, or disk-based on newer ones)
- Starts device drivers for NICs, storage, and platform-specific ASICs
- The kernel is a monolithic image specific to the hardware platform

### 1.4 init and Daemon Startup

- `/sbin/init` is the first userland process (PID 1)
- init reads `/etc/rc` scripts to bring up system services in order:
  1. **mgd** (management daemon) — handles CLI, NETCONF, and commit operations
  2. **rpd** (routing protocol daemon) — runs OSPF, BGP, IS-IS, etc.
  3. **chassisd** (chassis daemon) — monitors hardware, fans, PSUs, temperature
  4. **dcd** (device control daemon) — manages interface configuration
  5. **pfed** (packet forwarding engine daemon) — programs the forwarding table into hardware
  6. **snmpd**, **sshd**, **eventd**, and others as configured
- Daemons communicate via the Junos kernel shared memory (krt) and internal IPC sockets
- The system is "ready" once mgd accepts CLI sessions and rpd has converged

### 1.5 Dual RE Boot Behavior

- On dual-RE systems, both REs boot independently
- The primary RE is determined by the `master` configuration or slot priority
- The backup RE synchronizes its configuration from the primary after boot
- With GRES enabled, the backup RE maintains a hot copy of the kernel state

## 2. Package Signing and Validation

### 2.1 Package Structure

- Junos packages are `.tgz` archives containing:
  - A signed manifest listing all files and their SHA-256 checksums
  - The OS image (kernel + base system)
  - Platform-specific ASIC microcode
  - Package metadata (version, compatible platforms, dependencies)
- Packages are signed with Juniper's RSA key; the public key ships with every Junos device

### 2.2 Validation Process

```bash
request system software validate /var/tmp/junos-package.tgz
```

Validation checks, in order:

1. **Signature verification** — confirms the package was signed by Juniper and has not been tampered with
2. **Checksum verification** — every file inside the archive is hashed and compared to the manifest
3. **Compatibility check** — verifies the package matches the hardware platform (RE type, chassis model)
4. **Dependency check** — ensures required base packages or companion packages are present
5. **Space check** — confirms sufficient disk space exists for installation

### 2.3 When to Skip Validation

Using `no-validate` bypasses all the above checks. Appropriate only when:

- The package has already been validated in a prior step
- Time-critical maintenance windows where the package source is fully trusted
- Lab or test environments where speed outweighs risk

In production, always validate. A corrupted package can brick the device.

## 3. Upgrade Planning Methodology

### 3.1 Compatibility Matrix

- Check the Juniper Hardware Compatibility Matrix for your platform and target version
- Key questions:
  - Does the target version support your RE/line card combination?
  - Are there known caveats or required intermediate versions?
  - Is a direct upgrade path supported, or must you step through intermediate releases?
- Junos uses the format `major.minor R release.build` (e.g., 23.4R1.10)
- Generally, you can skip minor releases within the same major train but not across major trains without checking the upgrade path

### 3.2 Pre-Upgrade Backup Checklist

1. **Save rescue configuration:** `request system configuration rescue save`
2. **Export current config:** `show configuration | save /var/tmp/pre-upgrade-config.txt`
3. **Snapshot the root partition:** `request system snapshot`
4. **Record current version:** `show version | save /var/tmp/pre-upgrade-version.txt`
5. **Verify hardware health:** `show chassis alarms`, `show chassis environment`, `show system alarms`
6. **Check available disk space:** `show system storage`
7. **Transfer the package to the device:** `file copy` or SCP to `/var/tmp/`
8. **Validate the package:** `request system software validate /var/tmp/junos-package.tgz`

### 3.3 Upgrade Execution

- Schedule during a maintenance window with rollback plan documented
- On single-RE systems, expect a full outage during reboot (typically 5-15 minutes depending on platform)
- On dual-RE systems with ISSU, traffic disruption should be sub-second
- After upgrade, verify:
  - `show version` — correct version running
  - `show system alarms` — no unexpected alarms
  - `show bgp summary` / `show ospf neighbor` — routing adjacencies re-established
  - `show chassis fpc` — all line cards online

## 4. Downgrade Procedures and Considerations

### 4.1 When Downgrade Is Needed

- New version introduces bugs affecting your environment
- Feature regression or incompatibility with third-party equipment
- Performance degradation after upgrade

### 4.2 Downgrade Process

```bash
# Install the older version
request system software add /var/tmp/junos-older-version.tgz

# Reboot to activate
request system reboot
```

### 4.3 Downgrade Risks

- **Configuration incompatibility** — features configured in the newer version may not exist in the older one; commit will fail with "unknown command" errors
- **Schema changes** — the configuration database format may have changed between versions; rollback configurations saved on the newer version may not parse correctly
- **Forwarding table changes** — ASIC microcode differences can cause forwarding inconsistencies during the transition
- **License issues** — some features tied to version-specific licenses may deactivate

### 4.4 Safe Downgrade Strategy

1. Before upgrading, save a complete configuration compatible with the current (soon-to-be-old) version
2. After downgrading, load this saved configuration rather than relying on the running config
3. Test in a lab first if possible
4. On dual-RE systems, downgrade the backup RE first, verify, then switchover and downgrade the former primary

## 5. Storage Management

### 5.1 Checking Disk Usage

```bash
# Show filesystem usage
show system storage

# Detailed storage breakdown
show system storage detail
```

### 5.2 Cleaning Up Old Packages

```bash
# Remove old software packages and temp files
request system storage cleanup

# Dry run — see what would be deleted
request system storage cleanup dry-run

# Force cleanup without confirmation
request system storage cleanup no-confirm
```

### 5.3 What Gets Cleaned

- Previous Junos installation packages in `/var/sw/pkg/`
- Old core dump files in `/var/crash/`
- Temporary files in `/var/tmp/`
- Old log files that have been rotated out
- Cached package files from prior installs (especially when `no-copy` was not used)

### 5.4 Preventing Space Issues

- Use `no-copy` during installs to avoid retaining the `.tgz` in `/var/sw/pkg/`
- Schedule periodic `request system storage cleanup` via event policies or cron
- Monitor `/var/tmp/` and `/var/log/` usage — these grow over time
- On platforms with small flash storage (older EX or SRX), remove unused language packs and optional packages
- After a successful upgrade and validation, remove the install package from `/var/tmp/`

### 5.5 Dual Root Partition Management

- Each root partition holds a full copy of Junos
- After upgrade, the old version remains on the backup partition as a safety net
- `request system snapshot` copies the active partition to backup, overwriting the fallback
- Only snapshot after confirming the new version is stable — snapshotting too early removes your rollback path

## Prerequisites

- Console access (serial or virtual) for recovery procedures
- Sufficient free space on `/var/tmp/` for the install package (check with `show system storage`)
- For ISSU: dual REs with NSR and GRES already enabled and committed
- Transfer method available: SCP, FTP, USB, or TFTP for getting packages onto the device
- Current root password (or physical console access for password recovery)
- Maintenance window scheduled for upgrades on production equipment
