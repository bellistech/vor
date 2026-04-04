# Longhorn (Cloud-Native Distributed Storage)

> Lightweight, reliable distributed block storage for Kubernetes — built by Rancher/SUSE, using microservices architecture with per-volume engines, automated replica management, incremental snapshots, and backup to S3/NFS for disaster recovery.

## Installation

### Helm Install

```bash
# Add Longhorn Helm repo
helm repo add longhorn https://charts.longhorn.io
helm repo update

# Install Longhorn
helm install longhorn longhorn/longhorn \
  --namespace longhorn-system \
  --create-namespace \
  --version 1.7.0

# Verify pods are running
kubectl -n longhorn-system get pods

# Access the UI (port-forward)
kubectl -n longhorn-system port-forward svc/longhorn-frontend 8080:80
```

### Prerequisites Check

```bash
# Run Longhorn environment check
curl -sSfL https://raw.githubusercontent.com/longhorn/longhorn/v1.7.0/scripts/environment_check.sh | bash

# Required kernel modules
modprobe iscsi_tcp
modprobe dm_crypt

# Required packages (per node)
# Ubuntu/Debian: open-iscsi, nfs-common, cryptsetup
# RHEL/CentOS: iscsi-initiator-utils, nfs-utils, cryptsetup
apt-get install -y open-iscsi nfs-common
systemctl enable --now iscsid
```

## StorageClass

### Default StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: longhorn
  annotations:
    storageclass.kubernetes.io/is-default-class: "true"
provisioner: driver.longhorn.io
allowVolumeExpansion: true
reclaimPolicy: Delete
volumeBindingMode: Immediate
parameters:
  numberOfReplicas: "3"
  staleReplicaTimeout: "2880"
  fromBackup: ""
  fsType: "ext4"
```

### High-Performance StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: longhorn-fast
provisioner: driver.longhorn.io
allowVolumeExpansion: true
parameters:
  numberOfReplicas: "2"
  dataLocality: "best-effort"
  diskSelector: "ssd"
  nodeSelector: "storage"
```

## Volume Management

### Creating Volumes

```yaml
# PVC using Longhorn StorageClass
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-data
spec:
  accessModes:
    - ReadWriteOnce
  storageClassName: longhorn
  resources:
    requests:
      storage: 10Gi
```

```bash
# List volumes via kubectl
kubectl -n longhorn-system get volumes.longhorn.io

# Describe a volume
kubectl -n longhorn-system describe volume pvc-abc123

# Expand volume (edit PVC)
kubectl patch pvc my-data -p '{"spec":{"resources":{"requests":{"storage":"20Gi"}}}}'
```

## Replica Management

### Configuring Replicas

```bash
# Set default replica count (Longhorn Settings)
kubectl -n longhorn-system edit settings.longhorn.io default-replica-count

# Per-volume replica count via StorageClass parameters
# numberOfReplicas: "3"

# Check replica status
kubectl -n longhorn-system get replicas.longhorn.io | grep pvc-abc123

# Rebuild replica (automatic on node recovery, manual via UI)
# UI: Volume → Replica tab → Rebuild
```

## Snapshots

### Creating and Managing Snapshots

```bash
# Create snapshot via kubectl
kubectl -n longhorn-system apply -f - <<EOF
apiVersion: longhorn.io/v1beta2
kind: Snapshot
metadata:
  name: my-snap-$(date +%Y%m%d)
  namespace: longhorn-system
spec:
  volume: pvc-abc123
EOF

# List snapshots for a volume
kubectl -n longhorn-system get snapshots.longhorn.io -l longhornvolume=pvc-abc123

# Delete old snapshots
kubectl -n longhorn-system delete snapshot my-snap-20260301

# Recurring snapshots via RecurringJob
kubectl -n longhorn-system apply -f - <<EOF
apiVersion: longhorn.io/v1beta2
kind: RecurringJob
metadata:
  name: daily-snapshot
  namespace: longhorn-system
spec:
  cron: "0 2 * * *"
  task: snapshot
  retain: 7
  concurrency: 2
  groups:
    - default
EOF
```

## Backups

### Backup Target Configuration

```bash
# Set S3 backup target
kubectl -n longhorn-system edit settings.longhorn.io backup-target
# Value: s3://my-bucket@us-east-1/longhorn-backups

# Create S3 credentials secret
kubectl -n longhorn-system create secret generic s3-secret \
  --from-literal=AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE \
  --from-literal=AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY \
  --from-literal=AWS_ENDPOINTS=https://s3.us-east-1.amazonaws.com

# Set backup credential secret
kubectl -n longhorn-system edit settings.longhorn.io backup-target-credential-secret
# Value: s3-secret

# NFS backup target
# Value: nfs://nfs-server.example.com:/longhorn-backups
```

### Backup Operations

```bash
# Create backup of a volume
kubectl -n longhorn-system apply -f - <<EOF
apiVersion: longhorn.io/v1beta2
kind: Backup
metadata:
  name: backup-$(date +%Y%m%d)
  namespace: longhorn-system
spec:
  snapshotName: my-snap-20260403
  labels:
    environment: production
EOF

# List backups
kubectl -n longhorn-system get backups.longhorn.io

# Recurring backup job
kubectl -n longhorn-system apply -f - <<EOF
apiVersion: longhorn.io/v1beta2
kind: RecurringJob
metadata:
  name: weekly-backup
  namespace: longhorn-system
spec:
  cron: "0 3 * * 0"
  task: backup
  retain: 4
  concurrency: 1
  groups:
    - default
EOF
```

## Disaster Recovery

### DR Volumes

```bash
# Create DR volume from backup (standby volume)
# In the DR cluster, set the same backup target, then:
kubectl -n longhorn-system apply -f - <<EOF
apiVersion: longhorn.io/v1beta2
kind: Volume
metadata:
  name: dr-volume
  namespace: longhorn-system
spec:
  fromBackup: "s3://my-bucket@us-east-1/longhorn-backups?volume=pvc-abc123&backup=backup-20260403"
  numberOfReplicas: 3
  standby: true
EOF

# Activate DR volume (converts standby → active)
# UI: Volume → Activate DR Volume
# Or patch the volume:
kubectl -n longhorn-system patch volume dr-volume --type=merge -p '{"spec":{"standby":false}}'
```

## Node and Disk Management

### Node Scheduling

```bash
# Label nodes for storage
kubectl label node worker-01 node.longhorn.io/create-default-disk=true
kubectl label node worker-01 longhorn.io/node=storage

# Disable scheduling on a node
kubectl -n longhorn-system patch nodes.longhorn.io worker-03 \
  --type=merge -p '{"spec":{"allowScheduling":false}}'

# Evict replicas from a node (before maintenance)
kubectl -n longhorn-system patch nodes.longhorn.io worker-03 \
  --type=merge -p '{"spec":{"evictionRequested":true}}'
```

### Disk Management

```bash
# Add additional disk to a node
kubectl -n longhorn-system patch nodes.longhorn.io worker-01 --type=merge -p '
{
  "spec": {
    "disks": {
      "ssd-data": {
        "path": "/mnt/ssd",
        "allowScheduling": true,
        "storageReserved": 10737418240,
        "tags": ["ssd"]
      }
    }
  }
}'

# Check disk status
kubectl -n longhorn-system get nodes.longhorn.io worker-01 -o yaml | grep -A 20 diskStatus
```

## Monitoring

### Prometheus Metrics

```bash
# Longhorn exposes metrics at longhorn-manager:9500/metrics
# Key metrics:
# longhorn_volume_actual_size_bytes
# longhorn_volume_capacity_bytes
# longhorn_volume_state (1=attached, 0=detached)
# longhorn_volume_robustness (1=healthy, 0=degraded, -1=faulted)
# longhorn_node_storage_capacity_bytes
# longhorn_node_storage_usage_bytes
```

## Tips

- Always run the environment check script before installing — missing `iscsid` is the number one cause of failed installs
- Set `staleReplicaTimeout` to at least 2880 minutes (48h) to avoid premature replica eviction during node maintenance
- Use `dataLocality: best-effort` for workloads that benefit from local reads — reduces network I/O significantly
- Enable recurring snapshot and backup jobs via RecurringJob CRs — do not rely on manual snapshots for production
- For RWX volumes, Longhorn spins up an NFS share-manager pod — monitor its resource usage separately
- Use disk tags (`ssd`, `hdd`) with `diskSelector` in StorageClass to pin performance-sensitive volumes to fast storage
- Test DR volume activation regularly — a backup you have never restored is a backup you do not have
- Monitor `longhorn_volume_robustness` metric and alert on any value below 1 (degraded or faulted)
- Set `storageReserved` on each disk to prevent Longhorn from consuming all available space
- Keep Longhorn and Kubernetes versions in sync with the support matrix — version skew causes subtle issues
- Use volume encryption (LUKS) for sensitive data — enable via `encrypted: "true"` in StorageClass parameters

## See Also

- ceph, rook, zfs, lvm, btrfs

## References

- [Longhorn Official Documentation](https://longhorn.io/docs/)
- [Longhorn GitHub Repository](https://github.com/longhorn/longhorn)
- [Longhorn Architecture](https://longhorn.io/docs/latest/concepts/)
- [Longhorn Backup and Restore Guide](https://longhorn.io/docs/latest/snapshots-and-backups/)
- [Longhorn Helm Chart Values](https://github.com/longhorn/charts/blob/master/charts/longhorn/values.yaml)
