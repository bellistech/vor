# parted (GNU Partition Editor)

Advanced partition manager with GPT support, resize, and scriptable interface.

## Interactive Mode

```bash
sudo parted /dev/sdb
```

### Common Commands Inside parted

```bash
print                                    # show partition table
print free                               # show free space
mklabel gpt                             # create GPT table (erases all data)
mkpart primary ext4 1MiB 100GiB         # create partition
rm 2                                     # remove partition 2
resizepart 1 200GiB                      # resize partition 1
align-check optimal 1                    # verify alignment
quit
```

## Create Partition Table

```bash
# GPT (modern, supports >2TB, unlimited partitions)
sudo parted /dev/sdb mklabel gpt

# MBR (legacy BIOS)
sudo parted /dev/sdb mklabel msdos
```

## Create Partitions

### Script Mode (Non-Interactive)

```bash
# EFI System Partition
sudo parted -s /dev/sdb mkpart ESP fat32 1MiB 512MiB
sudo parted -s /dev/sdb set 1 esp on

# Root partition (remaining space)
sudo parted -s /dev/sdb mkpart root ext4 512MiB 100%

# Swap partition
sudo parted -s /dev/sdb mkpart swap linux-swap 100GiB 108GiB
```

### Full Disk Setup Example

```bash
sudo parted -s /dev/sdb \
  mklabel gpt \
  mkpart ESP fat32 1MiB 512MiB \
  set 1 esp on \
  mkpart root ext4 512MiB 50GiB \
  mkpart home ext4 50GiB 100%
```

## Print Partition Table

```bash
sudo parted /dev/sdb print
sudo parted /dev/sdb print free          # include free space
sudo parted -l                           # all disks

# Machine-readable output (for scripts)
sudo parted -m /dev/sdb print
```

## Resize Partition

```bash
# Grow partition 2 to 200GiB
sudo parted /dev/sdb resizepart 2 200GiB

# Grow partition 2 to fill remaining disk
sudo parted /dev/sdb resizepart 2 100%

# NOTE: parted only resizes the partition, not the filesystem.
# After resizing:
sudo resize2fs /dev/sdb2                 # ext4
sudo xfs_growfs /mountpoint              # XFS
```

## Remove Partition

```bash
sudo parted /dev/sdb rm 3               # remove partition 3
```

## Set Flags

```bash
sudo parted /dev/sdb set 1 esp on       # EFI System Partition
sudo parted /dev/sdb set 1 boot on      # boot flag (MBR)
sudo parted /dev/sdb set 2 lvm on       # LVM physical volume
sudo parted /dev/sdb set 3 raid on      # software RAID
sudo parted /dev/sdb set 1 swap on      # swap partition

# Toggle flag off
sudo parted /dev/sdb set 1 boot off
```

## Alignment

```bash
# Check alignment (should say "aligned")
sudo parted /dev/sdb align-check optimal 1

# Optimal alignment starts at 1MiB (2048 sectors for 512-byte sectors)
# Always use MiB/GiB units, not MB/GB, to ensure alignment
```

## Unit Selection

```bash
# In interactive mode
(parted) unit GiB
(parted) unit MiB
(parted) unit s                          # sectors
(parted) unit compact                    # auto-select readable unit

# In script mode
sudo parted -s /dev/sdb unit GiB print
```

## Rescue Lost Partition

```bash
sudo parted /dev/sdb rescue 1MiB 100GiB
# Searches for filesystem signatures in the range and offers to recreate
```

## Tips

- Use `MiB` and `GiB` (binary units), not `MB` and `GB` -- this ensures proper alignment on 4K-sector disks
- `parted` writes changes immediately (unlike fdisk); there is no "write and quit" safety net
- `-s` (script mode) suppresses prompts and is essential for automation
- `resizepart` can only grow partitions rightward; to shrink, the filesystem must be resized first
- XFS cannot be shrunk; only ext4 supports both grow and shrink
- After any partition change, run `partprobe /dev/sdb` to update the kernel's partition table
- For UEFI systems, the ESP must be FAT32 and at least 100 MiB (512 MiB recommended)
- `parted` does not format partitions; after creating them, use `mkfs.ext4`, `mkfs.xfs`, etc.
- GPT reserves the first and last 34 sectors (17 KiB) for the partition table and backup; start at 1MiB

## References

- [GNU Parted Manual](https://www.gnu.org/software/parted/manual/)
- [parted(8) Man Page](https://man7.org/linux/man-pages/man8/parted.8.html)
- [partprobe(8) Man Page](https://man7.org/linux/man-pages/man8/partprobe.8.html)
- [Arch Wiki — GNU Parted](https://wiki.archlinux.org/title/GNU_Parted)
- [Arch Wiki — Partitioning](https://wiki.archlinux.org/title/Partitioning)
- [Red Hat RHEL 9 — Getting Started with Partitions](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_storage_devices/getting-started-with-partitions_managing-storage-devices)
- [Ubuntu — Partitioning with parted](https://help.ubuntu.com/community/HowtoPartition/PartitioningwithGParted)
