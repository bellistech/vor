# Ceph (Distributed Storage Cluster)

Scalable distributed storage providing block (RBD), filesystem (CephFS), and object (RADOS/RGW) storage.

## Cluster Health and Status

```bash
sudo ceph status                         # overall cluster health
sudo ceph health detail                  # detailed health warnings
sudo ceph -s                             # short alias for status
sudo ceph df                             # cluster-wide storage usage
sudo ceph osd df                         # per-OSD usage
sudo ceph osd tree                       # CRUSH topology (host/rack/OSD tree)
```

## OSD Management

### List OSDs

```bash
sudo ceph osd ls
sudo ceph osd stat
sudo ceph osd tree
```

### Add OSD

```bash
# Using ceph-volume (modern method)
sudo ceph-volume lvm create --data /dev/sdb

# With a dedicated WAL/DB device
sudo ceph-volume lvm create --data /dev/sdb --block.db /dev/nvme0n1p1
```

### Remove OSD

```bash
# Mark OSD out (starts data migration)
sudo ceph osd out osd.5

# Wait for rebalance to complete
sudo ceph -w                             # watch cluster events

# Stop the OSD daemon
sudo systemctl stop ceph-osd@5

# Remove from CRUSH map and auth
sudo ceph osd crush remove osd.5
sudo ceph auth del osd.5
sudo ceph osd rm osd.5
```

### Mark OSD Down/Up

```bash
sudo ceph osd down osd.5
sudo ceph osd up osd.5
```

### Reweight OSD

```bash
sudo ceph osd reweight osd.5 0.8         # reduce data on this OSD
sudo ceph osd crush reweight osd.5 3.5   # set CRUSH weight (by TB)
```

## Pools

### Create Pool

```bash
# Replicated pool (3x replication, 128 PGs)
sudo ceph osd pool create mypool 128

# Erasure-coded pool
sudo ceph osd pool create ec-pool 128 erasure
```

### List and Get Pool Info

```bash
sudo ceph osd pool ls
sudo ceph osd pool ls detail
sudo ceph osd pool get mypool size       # replication factor
sudo ceph osd pool get mypool pg_num
```

### Set Pool Properties

```bash
sudo ceph osd pool set mypool size 3     # replication count
sudo ceph osd pool set mypool min_size 2 # minimum copies for I/O
sudo ceph osd pool set mypool pg_num 256 # increase PGs (auto-scales by default)
```

### Enable Pool Features

```bash
sudo ceph osd pool application enable mypool rbd
sudo ceph osd pool application enable mypool cephfs
sudo ceph osd pool application enable mypool rgw
```

### Delete Pool

```bash
# Requires mon_allow_pool_delete = true in ceph.conf
sudo ceph osd pool delete mypool mypool --yes-i-really-really-mean-it
```

## RBD (RADOS Block Device)

### Create Block Image

```bash
sudo rbd create mypool/myimage --size 100G
```

### List Images

```bash
sudo rbd ls mypool
sudo rbd info mypool/myimage
```

### Map and Mount

```bash
sudo rbd map mypool/myimage
# Output: /dev/rbd0

sudo mkfs.ext4 /dev/rbd0
sudo mount /dev/rbd0 /mnt/rbd
```

### Unmap

```bash
sudo umount /mnt/rbd
sudo rbd unmap /dev/rbd0
```

### Resize Image

```bash
sudo rbd resize mypool/myimage --size 200G
```

### Snapshots

```bash
sudo rbd snap create mypool/myimage@snap1
sudo rbd snap ls mypool/myimage
sudo rbd snap rollback mypool/myimage@snap1
sudo rbd snap rm mypool/myimage@snap1
```

## CephFS

### Create Filesystem

```bash
sudo ceph fs volume create myfs
# Or manually:
sudo ceph osd pool create cephfs_data 128
sudo ceph osd pool create cephfs_metadata 64
sudo ceph fs new myfs cephfs_metadata cephfs_data
```

### Mount CephFS

```bash
# Kernel client
sudo mount -t ceph mon1:6789:/ /mnt/cephfs -o name=admin,secret=<key>

# FUSE client
sudo ceph-fuse /mnt/cephfs

# With keyring file
sudo mount -t ceph mon1:/ /mnt/cephfs -o name=admin,secretfile=/etc/ceph/admin.secret
```

### List Filesystems

```bash
sudo ceph fs ls
sudo ceph fs status myfs
```

## RADOS (Low-Level Object Store)

```bash
# List objects in a pool
sudo rados -p mypool ls

# Put/get objects
sudo rados -p mypool put myobject /path/to/file
sudo rados -p mypool get myobject /path/to/output

# Delete object
sudo rados -p mypool rm myobject

# Benchmark
sudo rados bench -p mypool 30 write --no-cleanup
sudo rados bench -p mypool 30 seq
```

## CRUSH Map

### View CRUSH Map

```bash
sudo ceph osd crush dump
sudo ceph osd crush tree
sudo ceph osd crush rule ls
```

### Add Bucket (Host/Rack)

```bash
sudo ceph osd crush add-bucket rack1 rack
sudo ceph osd crush move rack1 root=default
sudo ceph osd crush move host1 rack=rack1
```

## PG (Placement Group) States

```bash
sudo ceph pg stat
sudo ceph pg dump
sudo ceph pg dump_stuck unclean           # stuck PGs

# Common PG states:
# active+clean     - healthy
# active+degraded  - missing replicas, I/O continues
# peering          - OSDs agreeing on PG state
# recovering       - restoring replicas
# backfilling      - moving data for rebalance
# stale            - PG not updated (OSD down?)
# inconsistent     - checksum mismatch (run scrub)
```

### Repair Inconsistent PG

```bash
sudo ceph pg repair 1.2a                 # PG ID from status output
```

## Authentication

```bash
# List auth keys
sudo ceph auth ls

# Create client key
sudo ceph auth get-or-create client.myapp \
  mon 'allow r' \
  osd 'allow rw pool=mypool'

# Get key for a client
sudo ceph auth get client.myapp
```

## Tips

- A healthy cluster shows `HEALTH_OK`; never ignore `HEALTH_WARN` -- it often precedes data issues
- PG count should be roughly 100 per OSD per pool; since Nautilus, PG autoscaler handles this
- Never remove multiple OSDs simultaneously; wait for rebalance between each removal
- Erasure-coded pools save space but cannot be used for RBD snapshots or CephFS metadata
- Monitor memory: each OSD uses 3-5 GB RAM; monitors need 2-4 GB each
- `ceph -w` (watch mode) is invaluable during maintenance; it streams cluster events in real time
- `min_size` determines the minimum number of replicas needed for I/O; setting it to 1 risks data loss
- BlueStore is the only production-ready backend since Nautilus; FileStore is deprecated
- CRUSH rules control data placement across failure domains; always verify rules match your physical topology

## References

- [Ceph Documentation](https://docs.ceph.com/)
- [Ceph Architecture Overview](https://docs.ceph.com/en/latest/architecture/)
- [Ceph RADOS — Cluster Operations](https://docs.ceph.com/en/latest/rados/)
- [Ceph RBD — Block Device Guide](https://docs.ceph.com/en/latest/rbd/)
- [CephFS — File System Guide](https://docs.ceph.com/en/latest/cephfs/)
- [Ceph Object Gateway (RGW)](https://docs.ceph.com/en/latest/radosgw/)
- [Ceph Placement Groups](https://docs.ceph.com/en/latest/rados/operations/placement-groups/)
- [Ceph Hardware Recommendations](https://docs.ceph.com/en/latest/start/hardware-recommendations/)
- [Red Hat Ceph Storage — Administration Guide](https://access.redhat.com/documentation/en-us/red_hat_ceph_storage/)
- [Arch Wiki — Ceph](https://wiki.archlinux.org/title/Ceph)
