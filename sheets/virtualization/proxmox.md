# proxmox (virtualization platform)

Proxmox VE is an open-source server virtualization platform combining KVM hypervisor and LXC containers with a web-based management interface, high availability clustering via Corosync, integrated Ceph storage, and comprehensive backup through Proxmox Backup Server.

## Installation and Access

### Initial Setup

```bash
# Default web UI access after installation
# https://<host-ip>:8006

# Login via CLI
pvesh get /version

# Update package repositories
apt update && apt dist-upgrade

# Add no-subscription repository (non-production)
echo "deb http://download.proxmox.com/debian/pve bookworm pve-no-subscription" \
  > /etc/apt/sources.list.d/pve-no-subscription.list

# Remove enterprise repo nag (if no subscription)
rm /etc/apt/sources.list.d/pve-enterprise.list

# Check cluster status
pvecm status

# System info
pveversion -v
```

## VM Management

### Create and Manage VMs

```bash
# Create VM from ISO
qm create 100 \
  --name ubuntu-server \
  --memory 4096 \
  --cores 4 \
  --cpu host \
  --net0 virtio,bridge=vmbr0 \
  --scsihw virtio-scsi-single \
  --scsi0 local-lvm:40,iothread=1 \
  --ide2 local:iso/ubuntu-24.04.iso,media=cdrom \
  --boot order=ide2 \
  --ostype l26 \
  --agent enabled=1

# Start VM
qm start 100

# Stop VM (graceful)
qm shutdown 100

# Force stop
qm stop 100

# Reboot
qm reboot 100

# Delete VM
qm destroy 100 --purge

# List all VMs
qm list

# Show VM config
qm config 100

# Set VM options
qm set 100 --memory 8192 --cores 8

# Resize disk
qm resize 100 scsi0 +20G

# Clone VM
qm clone 100 101 --name ubuntu-clone --full

# Template from VM
qm template 100

# VM status
qm status 100
```

### Cloud-Init Integration

```bash
# Add cloud-init drive
qm set 100 --ide2 local-lvm:cloudinit

# Configure cloud-init
qm set 100 \
  --ciuser admin \
  --cipassword "$(openssl passwd -6 'secretpass')" \
  --sshkeys ~/.ssh/authorized_keys \
  --ipconfig0 ip=10.0.0.100/24,gw=10.0.0.1 \
  --nameserver 1.1.1.1 \
  --searchdomain lab.local

# Regenerate cloud-init image
qm cloudinit dump 100 user
qm cloudinit update 100
```

## Container Management

### LXC Containers

```bash
# Download container template
pveam update
pveam available --section system
pveam download local debian-12-standard_12.2-1_amd64.tar.zst

# Create container
pct create 200 local:vztmpl/debian-12-standard_12.2-1_amd64.tar.zst \
  --hostname debian-ct \
  --memory 2048 \
  --swap 512 \
  --cores 2 \
  --net0 name=eth0,bridge=vmbr0,ip=dhcp \
  --rootfs local-lvm:8 \
  --unprivileged 1 \
  --features nesting=1

# Start container
pct start 200

# Enter container shell
pct enter 200

# Execute command in container
pct exec 200 -- apt update

# Stop container
pct stop 200

# Container config
pct config 200

# Set container options
pct set 200 --memory 4096

# Mount host directory in container
pct set 200 --mp0 /mnt/data,mp=/data

# List containers
pct list
```

## Clustering

### HA Cluster Setup

```bash
# Create cluster (first node)
pvecm create my-cluster

# Join cluster (additional nodes)
pvecm add <first-node-ip>

# Cluster status
pvecm status

# List nodes
pvecm nodes

# Check quorum
pvecm expected 2

# Remove node from cluster
pvecm delnode node3

# Corosync config
cat /etc/pve/corosync.conf

# Check Corosync ring status
corosync-cfgtool -s

# Enable HA on a VM
ha-manager add vm:100 --state started --group ha-group1

# HA status
ha-manager status

# Create HA group
ha-manager groupadd ha-group1 --nodes node1,node2 --nofailback 0

# Migrate VM to another node
qm migrate 100 node2 --online

# Bulk migrate
pvesh get /cluster/resources --type vm
```

## Storage

### Storage Configuration

```bash
# List storage
pvesm status

# Add NFS storage
pvesm add nfs nfs-share \
  --server 10.0.0.50 \
  --export /mnt/vms \
  --content images,iso,vztmpl,backup

# Add local LVM-thin
pvesm add lvmthin local-lvm \
  --vgname pve \
  --thinpool data \
  --content images,rootdir

# Add ZFS pool
pvesm add zfspool zfs-store \
  --pool rpool/data \
  --content images,rootdir

# Add Ceph (RBD) storage
pvesm add rbd ceph-pool \
  --monhost 10.0.0.1,10.0.0.2,10.0.0.3 \
  --pool vm-pool \
  --content images,rootdir \
  --username admin

# Add SMB/CIFS share
pvesm add cifs smb-share \
  --server 10.0.0.60 \
  --share backups \
  --username admin \
  --password secret \
  --content backup

# Storage info
pvesm list local-lvm
pvesm path local-lvm:vm-100-disk-0
```

### Ceph Integration

```bash
# Install Ceph packages
pveceph install

# Initialize Ceph
pveceph init --network 10.0.0.0/24

# Create monitors (on 3+ nodes)
pveceph mon create

# Create managers
pveceph mgr create

# Create OSDs (one per disk)
pveceph osd create /dev/sdb
pveceph osd create /dev/sdc

# Create storage pool
pveceph pool create vm-pool --pg_num 128

# Ceph status
ceph status
ceph osd tree
ceph df

# Add CephFS (for shared filesystem)
pveceph mds create
pveceph fs create cephfs --pg_num 64 --add-storage
```

## Backup and Restore

### vzdump and PBS

```bash
# Backup a VM
vzdump 100 --storage local --mode snapshot --compress zstd

# Backup all VMs on node
vzdump --all --storage local --mode snapshot

# Restore VM from backup
qmrestore /var/lib/vz/dump/vzdump-qemu-100-2024_01_01-00_00_00.vma.zst 100

# Restore to different ID
qmrestore /var/lib/vz/dump/vzdump-qemu-100-*.vma.zst 105 --storage local-lvm

# Restore container
pct restore 200 /var/lib/vz/dump/vzdump-lxc-200-*.tar.zst

# List backups
vzdump --list

# Proxmox Backup Server integration
pvesm add pbs pbs-store \
  --server 10.0.0.70 \
  --datastore main \
  --username backup@pbs \
  --password secret \
  --fingerprint <sha256>
```

## API Access

### REST API

```bash
# Create API token (persistent)
pveum user token add root@pam automation --privsep 0

# List VMs via API
curl -k -H "Authorization: PVEAPIToken=root@pam!automation=<token>" \
  https://proxmox:8006/api2/json/cluster/resources?type=vm

# Start VM via API
curl -k -X POST \
  -H "Authorization: PVEAPIToken=root@pam!automation=<token>" \
  https://proxmox:8006/api2/json/nodes/pve1/qemu/100/status/start
```

## Tips

- Use `virtio-scsi-single` with `iothread=1` for best disk performance in VMs
- Enable the QEMU guest agent (`agent enabled=1`) for clean shutdown and filesystem freeze during backup
- Use snapshot backup mode instead of stop mode to avoid VM downtime during backup
- ZFS provides transparent compression and snapshots but avoid mixing ZFS with Ceph on the same node
- VLAN-aware bridges simplify network management by handling VLAN tagging at the bridge level
- Create templates from cloud images with cloud-init for rapid provisioning of identical VMs
- Always use odd numbers of nodes (3, 5, 7) for Ceph and HA clusters to maintain quorum
- API tokens with `privsep=0` inherit user privileges and are simpler for automation
- Proxmox Backup Server supports deduplication and incremental backups, saving significant storage
- Use unprivileged containers with `nesting=1` for Docker-in-LXC workloads

## See Also

- qemu, kvm, libvirt, ceph, zfs, corosync, cloud-init

## References

- [Proxmox VE Documentation](https://pve.proxmox.com/pve-docs/)
- [Proxmox VE API Reference](https://pve.proxmox.com/pve-docs/api-viewer/)
- [Proxmox Wiki](https://pve.proxmox.com/wiki/Main_Page)
- [Proxmox Backup Server Docs](https://pbs.proxmox.com/docs/)
- [Proxmox Ceph Integration](https://pve.proxmox.com/pve-docs/chapter-pveceph.html)
