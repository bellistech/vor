# LVM (Logical Volume Manager)

Flexible disk management with resizable volumes, snapshots, and multi-disk spanning.

## Physical Volumes (PV)

### Create PV

```bash
sudo pvcreate /dev/sdb
sudo pvcreate /dev/sdc /dev/sdd          # multiple at once
```

### Display PV Info

```bash
sudo pvs                                 # summary
sudo pvdisplay                           # detailed
sudo pvdisplay /dev/sdb                  # specific disk
```

### Remove PV

```bash
sudo pvremove /dev/sdb
```

### Move Data Off a PV (Before Removal)

```bash
sudo pvmove /dev/sdb                     # move extents to other PVs in the VG
sudo pvmove /dev/sdb /dev/sdd            # move extents to a specific PV
```

## Volume Groups (VG)

### Create VG

```bash
sudo vgcreate data_vg /dev/sdb /dev/sdc
```

### Extend VG (Add a Disk)

```bash
sudo pvcreate /dev/sdd
sudo vgextend data_vg /dev/sdd
```

### Reduce VG (Remove a Disk)

```bash
sudo pvmove /dev/sdb                     # evacuate data first
sudo vgreduce data_vg /dev/sdb
```

### Display VG Info

```bash
sudo vgs                                 # summary
sudo vgdisplay                           # detailed
sudo vgdisplay data_vg
```

### Remove VG

```bash
sudo vgremove data_vg
```

## Logical Volumes (LV)

### Create LV

```bash
# Fixed size
sudo lvcreate -L 50G -n app_lv data_vg

# Percentage of free space
sudo lvcreate -l 100%FREE -n app_lv data_vg

# Percentage of VG total
sudo lvcreate -l 80%VG -n app_lv data_vg
```

### Format and Mount

```bash
sudo mkfs.ext4 /dev/data_vg/app_lv
sudo mkdir -p /mnt/app
sudo mount /dev/data_vg/app_lv /mnt/app
```

### Display LV Info

```bash
sudo lvs                                 # summary
sudo lvdisplay                           # detailed
sudo lvdisplay /dev/data_vg/app_lv
```

### Remove LV

```bash
sudo umount /mnt/app
sudo lvremove /dev/data_vg/app_lv
```

## Resizing

### Extend LV + Filesystem (Online)

```bash
# Add 20G
sudo lvextend -L +20G /dev/data_vg/app_lv

# Resize filesystem to fill the LV
sudo resize2fs /dev/data_vg/app_lv       # ext4
sudo xfs_growfs /mnt/app                 # XFS

# One-step: extend LV and resize filesystem together
sudo lvextend -L +20G --resizefs /dev/data_vg/app_lv

# Extend to use all remaining free space
sudo lvextend -l +100%FREE --resizefs /dev/data_vg/app_lv
```

### Reduce LV (ext4 Only, Requires Unmount)

```bash
sudo umount /mnt/app
sudo e2fsck -f /dev/data_vg/app_lv      # must check filesystem first
sudo resize2fs /dev/data_vg/app_lv 30G   # shrink filesystem
sudo lvreduce -L 30G /dev/data_vg/app_lv # shrink LV to match

# One-step shrink (still requires unmount + fsck)
sudo lvreduce -L 30G --resizefs /dev/data_vg/app_lv
```

## Snapshots

### Create Snapshot

```bash
# COW snapshot (needs free space in VG for changes)
sudo lvcreate -s -L 10G -n app_snap /dev/data_vg/app_lv
```

### Mount Snapshot (Read-Only)

```bash
sudo mkdir -p /mnt/snap
sudo mount -o ro /dev/data_vg/app_snap /mnt/snap
```

### Restore from Snapshot

```bash
sudo umount /mnt/app
sudo lvconvert --merge /dev/data_vg/app_snap
# Reactivate the LV
sudo lvchange -an /dev/data_vg/app_lv
sudo lvchange -ay /dev/data_vg/app_lv
sudo mount /dev/data_vg/app_lv /mnt/app
```

### Remove Snapshot

```bash
sudo umount /mnt/snap
sudo lvremove /dev/data_vg/app_snap
```

## Thin Provisioning

### Create Thin Pool

```bash
sudo lvcreate -L 100G --thinpool thin_pool data_vg
```

### Create Thin LV (Over-Provisioned)

```bash
# Virtual size larger than pool (thin provisioned)
sudo lvcreate -V 200G --thin -n thin_app data_vg/thin_pool
```

### Monitor Thin Pool Usage

```bash
sudo lvs -o +data_percent,metadata_percent data_vg/thin_pool
```

## Rename and Move

```bash
# Rename LV
sudo lvrename data_vg app_lv web_lv

# Rename VG
sudo vgrename data_vg storage_vg
```

## Activate and Deactivate

```bash
sudo lvchange -ay /dev/data_vg/app_lv   # activate
sudo lvchange -an /dev/data_vg/app_lv   # deactivate

sudo vgchange -ay data_vg               # activate all LVs in VG
sudo vgchange -an data_vg               # deactivate all LVs in VG
```

## Tips

- Always use `--resizefs` with `lvextend` to grow the filesystem in one step -- avoids forgetting `resize2fs`
- XFS cannot be shrunk; only `ext4` supports `lvreduce --resizefs`
- Snapshot COW space fills up as the origin changes; if it hits 100%, the snapshot becomes invalid
- `pvmove` can be run online and is essential before removing a physical disk from a VG
- LV paths are accessible as `/dev/vg_name/lv_name` or `/dev/mapper/vg_name-lv_name`
- Add LVM entries to `/etc/fstab` using the `/dev/mapper/` path or UUID from `blkid`
- Thin provisioning allows over-commitment but requires monitoring; set up alerts on pool usage
- `lvm.conf` can restrict which devices LVM scans; useful in multipath or VM environments
