# mdadm (Linux Software RAID)

Create and manage Linux software RAID arrays (md devices).

## Create Arrays

### RAID 1 (Mirror)

```bash
sudo mdadm --create /dev/md0 --level=1 --raid-devices=2 /dev/sdb /dev/sdc
```

### RAID 5 (Striped with Single Parity)

```bash
sudo mdadm --create /dev/md0 --level=5 --raid-devices=3 /dev/sdb /dev/sdc /dev/sdd
```

### RAID 6 (Striped with Double Parity)

```bash
sudo mdadm --create /dev/md0 --level=6 --raid-devices=4 /dev/sdb /dev/sdc /dev/sdd /dev/sde
```

### RAID 10 (Mirrored Stripes)

```bash
sudo mdadm --create /dev/md0 --level=10 --raid-devices=4 /dev/sdb /dev/sdc /dev/sdd /dev/sde
```

### With Spare Disk

```bash
sudo mdadm --create /dev/md0 --level=5 --raid-devices=3 --spare-devices=1 \
  /dev/sdb /dev/sdc /dev/sdd /dev/sde
```

### Format and Mount

```bash
sudo mkfs.ext4 /dev/md0
sudo mkdir -p /mnt/raid
sudo mount /dev/md0 /mnt/raid
```

## Status and Monitoring

### Array Detail

```bash
sudo mdadm --detail /dev/md0
```

### Brief Status (All Arrays)

```bash
cat /proc/mdstat
```

### Examine a Component Disk

```bash
sudo mdadm --examine /dev/sdb
```

### Scan for Arrays

```bash
sudo mdadm --assemble --scan
```

## Save Configuration

```bash
# Generate config and save (critical for boot-time assembly)
sudo mdadm --detail --scan >> /etc/mdadm/mdadm.conf

# On RHEL/CentOS
sudo mdadm --detail --scan >> /etc/mdadm.conf

# Update initramfs so array assembles at boot
sudo update-initramfs -u                 # Debian/Ubuntu
sudo dracut --force                      # RHEL/CentOS
```

## Managing Disks

### Fail a Disk (Simulate or Mark Failure)

```bash
sudo mdadm /dev/md0 --fail /dev/sdc
```

### Remove a Failed Disk

```bash
sudo mdadm /dev/md0 --remove /dev/sdc
```

### Add Replacement Disk

```bash
sudo mdadm /dev/md0 --add /dev/sde
# Rebuild starts automatically
```

### Add Hot Spare

```bash
sudo mdadm /dev/md0 --add /dev/sdf
# If array is healthy, disk becomes a spare
```

### Fail + Remove + Replace (Full Workflow)

```bash
sudo mdadm /dev/md0 --fail /dev/sdc
sudo mdadm /dev/md0 --remove /dev/sdc
# Physically swap the disk
sudo mdadm /dev/md0 --add /dev/sdc      # new disk, same slot
```

## Rebuild and Grow

### Monitor Rebuild Progress

```bash
cat /proc/mdstat                         # shows rebuild percentage
watch cat /proc/mdstat                   # live monitoring
```

### Grow Array (Add More Disks)

```bash
# Add disk first
sudo mdadm /dev/md0 --add /dev/sdf

# Grow RAID5 from 3 to 4 disks
sudo mdadm --grow /dev/md0 --raid-devices=4

# After grow completes, resize filesystem
sudo resize2fs /dev/md0                  # ext4
sudo xfs_growfs /mnt/raid               # XFS
```

### Set Rebuild Speed

```bash
# Increase rebuild speed (default is often throttled)
echo 200000 | sudo tee /proc/sys/dev/raid/speed_limit_min
echo 500000 | sudo tee /proc/sys/dev/raid/speed_limit_max
```

## Assemble and Stop

### Assemble Array Manually

```bash
sudo mdadm --assemble /dev/md0 /dev/sdb /dev/sdc /dev/sdd
```

### Stop Array

```bash
sudo umount /dev/md0
sudo mdadm --stop /dev/md0
```

### Assemble All Known Arrays

```bash
sudo mdadm --assemble --scan
```

## Destroy Array

```bash
sudo umount /dev/md0
sudo mdadm --stop /dev/md0

# Clear superblock from each disk
sudo mdadm --zero-superblock /dev/sdb
sudo mdadm --zero-superblock /dev/sdc
sudo mdadm --zero-superblock /dev/sdd

# Remove from config
sudo vi /etc/mdadm/mdadm.conf           # remove the ARRAY line
sudo update-initramfs -u
```

## Monitoring

### Email Alerts

```bash
# /etc/mdadm/mdadm.conf
MAILADDR admin@acme.com

# Start monitoring daemon
sudo mdadm --monitor --scan --daemonise
```

### Systemd Monitor

```bash
sudo systemctl enable mdmonitor
sudo systemctl start mdmonitor
```

### Manual Check

```bash
# Trigger data scrub (online integrity check)
echo check | sudo tee /sys/block/md0/md/sync_action
cat /sys/block/md0/md/mismatch_cnt      # should be 0
```

## Tips

- Always save config to `mdadm.conf` and update initramfs after creating or modifying arrays -- otherwise the array may not assemble at boot
- RAID is not a backup; it protects against disk failure, not accidental deletion or corruption
- `cat /proc/mdstat` is the quickest way to check rebuild progress and array health
- Rebuild speed limits are conservative by default; increase them during maintenance windows for faster recovery
- Never force-assemble a degraded array with `--force` unless you understand the data loss implications
- RAID5 with large disks (4TB+) has a nontrivial probability of a second disk failing during rebuild; prefer RAID6 or RAID10
- Use `--assume-clean` only for new arrays with no data; skipping initial sync on existing data risks inconsistency
- Hot spares automatically replace failed disks, reducing exposure time significantly
- The `mdmonitor` service sends email alerts on failure; always configure `MAILADDR` in production
