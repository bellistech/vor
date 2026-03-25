# fdisk (Partition Table Manipulator)

Interactive tool for creating, deleting, and managing MBR and GPT disk partitions.

## List Partitions

```bash
sudo fdisk -l                            # all disks
sudo fdisk -l /dev/sda                   # specific disk
sudo fdisk -l /dev/nvme0n1
```

## Interactive Mode

```bash
sudo fdisk /dev/sdb
```

### Key Commands Inside fdisk

```bash
# Navigation
m    # help menu
p    # print partition table
F    # list free unpartitioned space

# Partition operations
n    # new partition
d    # delete partition
t    # change partition type
l    # list known partition types

# Write and quit
w    # write changes to disk (destructive)
q    # quit without saving
```

## Create MBR Partition Table

```bash
sudo fdisk /dev/sdb
# o    -> create new MBR (DOS) partition table
# n    -> new partition
# p    -> primary (or e for extended)
# 1    -> partition number
# Enter -> default first sector
# +50G -> size (or Enter for remaining space)
# w    -> write and exit
```

## Create GPT Partition Table

```bash
sudo fdisk /dev/sdb
# g    -> create new GPT partition table
# n    -> new partition
# 1    -> partition number
# Enter -> default first sector
# +100G -> size
# w    -> write and exit
```

## Delete a Partition

```bash
sudo fdisk /dev/sdb
# d    -> delete
# 2    -> partition number to delete
# w    -> write
```

## Change Partition Type

```bash
sudo fdisk /dev/sdb
# t    -> change type
# 2    -> partition number
# l    -> list types (common: 83=Linux, 82=swap, 8e=LVM, fd=RAID)
# 8e   -> set to Linux LVM
# w    -> write
```

### Common GPT Partition Type GUIDs

```bash
# In GPT mode, use aliases:
# 1   = EFI System
# 20  = Linux filesystem
# 22  = Linux swap
# 30  = Linux LVM
# 29  = Linux RAID
```

## Non-Interactive (Scripted)

```bash
# Create a single partition filling the entire disk
echo -e "g\nn\n1\n\n\nw" | sudo fdisk /dev/sdb

# Pipe multiple commands
sudo fdisk /dev/sdb <<EOF
g
n
1

+512M
t
1
n
2


w
EOF
```

## Verify Partition Table

```bash
sudo fdisk -l /dev/sdb
sudo partprobe /dev/sdb                  # inform kernel of changes
lsblk /dev/sdb                           # tree view
```

## Backup and Restore Partition Table

```bash
# MBR backup (first 512 bytes)
sudo dd if=/dev/sdb of=sdb-mbr.bak bs=512 count=1

# Restore MBR
sudo dd if=sdb-mbr.bak of=/dev/sdb bs=512 count=1

# For GPT, use sfdisk instead
sudo sfdisk --dump /dev/sdb > sdb-gpt.bak
sudo sfdisk /dev/sdb < sdb-gpt.bak
```

## Tips

- fdisk supports both MBR and GPT (since util-linux 2.23); for older systems, use `gdisk` for GPT
- MBR is limited to 4 primary partitions and 2 TB disk size; use GPT for modern systems
- Changes are only written when you press `w`; press `q` to abandon changes safely
- Always run `partprobe` or reboot after partitioning so the kernel re-reads the table
- For UEFI boot, you need a GPT table with an EFI System Partition (ESP) of at least 512 MB
- `fdisk` cannot resize partitions; use `parted` or delete and recreate with the same start sector
- Scripting fdisk is fragile; prefer `sfdisk` or `parted --script` for automation
- On NVMe drives, partitions are named `nvme0n1p1` not `nvme0n11`; the `p` separates device from partition number

## References

- [fdisk(8) Man Page](https://man7.org/linux/man-pages/man8/fdisk.8.html)
- [sfdisk(8) Man Page](https://man7.org/linux/man-pages/man8/sfdisk.8.html)
- [cfdisk(8) Man Page](https://man7.org/linux/man-pages/man8/cfdisk.8.html)
- [Arch Wiki — Partitioning](https://wiki.archlinux.org/title/Partitioning)
- [Arch Wiki — fdisk](https://wiki.archlinux.org/title/Fdisk)
- [Red Hat RHEL 9 — Getting Started with Partitions](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/managing_storage_devices/getting-started-with-partitions_managing-storage-devices)
- [Kernel Block Device Documentation](https://www.kernel.org/doc/html/latest/block/)
