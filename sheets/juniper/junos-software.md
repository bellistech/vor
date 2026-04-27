# Junos OS Software Management (JNCIA-Junos)

Quick reference for Junos software installation, upgrades, boot process, recovery, and system maintenance.

## Software Installation and Upgrades

```bash
# Standard software upgrade
request system software add /var/tmp/junos-package.tgz

# Skip validation (faster, less safe)
request system software add /var/tmp/junos-package.tgz no-validate

# Reboot automatically after install
request system software add /var/tmp/junos-package.tgz reboot

# Don't copy package to /var/sw/pkg (saves space)
request system software add /var/tmp/junos-package.tgz no-copy

# Combine flags
request system software add /var/tmp/junos-package.tgz no-validate reboot

# Install from USB
request system software add /dev/da1s1:/junos-package.tgz

# Validate package without installing
request system software validate /var/tmp/junos-package.tgz
```

## Viewing System Software

```bash
# Show current Junos version
show version

# Show installed software packages
show system software

# Show detailed version info (model, serial, etc.)
show version detail

# Show dual root partition status
show system snapshot media internal
```

## ISSU (In-Service Software Upgrade)

**Requirements:**
- Dual Routing Engines (RE)
- Nonstop Active Routing (NSR) enabled
- Graceful Routing Engine Switchover (GRES) enabled

```bash
# Enable NSR
set routing-options nonstop-routing

# Enable GRES
set chassis redundancy graceful-switchover

# Perform ISSU
request system software in-service-upgrade /var/tmp/junos-package.tgz
```

**Unified ISSU vs Standard Upgrade:**
- **Unified ISSU** — upgrades both REs with minimal traffic disruption; control plane switches over gracefully
- **Standard upgrade** — requires full reboot; causes traffic outage during restart

## Boot Sequence

```
Power On → POST (Power-On Self Test)
  → Boot Loader (reads boot device)
    → FreeBSD Kernel loads
      → init process starts
        → Junos daemons launch (mgd, rpd, chassisd, etc.)
```

## Shutting Down

```bash
# Graceful halt (stops OS, safe to power off)
request system halt

# Graceful power off (halts and powers down)
request system power-off

# Halt a specific RE on dual-RE system
request system halt member 1
```

> Always use graceful shutdown. Pulling power risks filesystem corruption
> and configuration loss.

## Root Password Recovery

```
1. Reboot the device
2. Press SPACE at boot loader prompt to interrupt boot
3. Boot into single-user mode:
     boot -s
4. Enter recovery mode at prompt:
     recovery
5. Enter configuration mode:
     configure
6. Set the root password:
     set system root-authentication plain-text-password
     (enter new password twice)
7. Commit and exit:
     commit
     exit
8. Reboot:
     request system reboot
```

## Rescue Configuration

```bash
# Save current config as rescue
request system configuration rescue save

# Delete rescue configuration
request system configuration rescue delete

# Rollback to rescue configuration
rollback rescue
commit

# View rescue config
show system configuration rescue
```

> Use rescue config as a known-good fallback before making risky changes.

## Configuration Archival

```
[edit system archival]
set configuration transfer-on-commit
set configuration archive-sites "scp://user@host:/path/"
set configuration archive-sites "ftp://user@host:/path/"
```

```bash
# Verify archival settings
show system archival
```

## Dual Root Partitions

```bash
# Snapshot active partition to backup
request system snapshot

# Snapshot to specific media
request system snapshot media internal slice alternate

# Show partition status
show system snapshot media internal
```

> Junos maintains active and backup root partitions. If the active
> partition fails, the device boots from backup automatically.

## Autorecovery

```bash
# Check autorecovery state
request system autorecovery state check

# Enable autorecovery
request system autorecovery state save
```

## Factory Reset

```bash
# Zeroize — wipe all config and logs, reboot to factory state
request system zeroize

# Load factory defaults (in config mode)
load factory-default
set system root-authentication plain-text-password
commit
```

> `zeroize` is destructive and irreversible. `load factory-default` keeps
> you in config mode so you can set root password before commit.

## Tips

- Always back up the current config before any upgrade: `request system configuration rescue save`
- Use `no-validate` only when you trust the package and need speed — validation catches corruption
- ISSU requires both NSR and GRES enabled *before* the upgrade; enable and commit first
- `no-copy` is useful when `/var/tmp` space is limited but means you cannot reinstall without re-transferring
- After upgrade, verify with `show version` and `show system alarms`
- On dual-RE systems, upgrade both REs to the same version to avoid mismatches
- Graceful shutdown prevents filesystem corruption — never just pull power

## See Also

- `sheets/juniper/junos-routing-fundamentals.md` — CLI navigation and configuration basics
- `sheets/juniper/junos-routing-fundamentals.md` — Routing fundamentals
- `sheets/juniper/junos-interfaces.md` — Interface configuration

## References

- Juniper JNCIA-Junos Study Guide — Software Installation and Maintenance
- Juniper TechLibrary: Installing Software on Junos Devices
- Juniper TechLibrary: ISSU Overview
- Juniper TechLibrary: Root Password Recovery
- Juniper Day One: Junos for IOS Engineers — Chapter on Software Management
