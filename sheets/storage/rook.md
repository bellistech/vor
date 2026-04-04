# Rook (Cloud-Native Storage Orchestrator for Ceph)

> Kubernetes operator that automates Ceph deployment, configuration, scaling, and upgrades — translating CephCluster, CephBlockPool, CephFilesystem, and CephObjectStore custom resources into a fully managed, production-grade distributed storage system running inside your cluster.

## Installation

### Operator Deploy

```bash
# Clone Rook repository
git clone --single-branch --branch v1.15.0 https://github.com/rook/rook.git
cd rook/deploy/examples

# Deploy CRDs and operator
kubectl apply -f crds.yaml -f common.yaml -f operator.yaml

# Wait for operator to be ready
kubectl -n rook-ceph wait --for=condition=ready pod -l app=rook-ceph-operator --timeout=300s

# Verify operator is running
kubectl -n rook-ceph get pods
```

### Helm Install

```bash
# Add Rook Helm repo
helm repo add rook-release https://charts.rook.io/release
helm repo update

# Install operator
helm install --create-namespace --namespace rook-ceph \
  rook-ceph rook-release/rook-ceph \
  --version v1.15.0

# Install cluster (after operator is ready)
helm install --namespace rook-ceph \
  rook-ceph-cluster rook-release/rook-ceph-cluster \
  --version v1.15.0 \
  -f cluster-values.yaml
```

## CephCluster CR

### Basic Cluster

```yaml
apiVersion: ceph.rook.io/v1
kind: CephCluster
metadata:
  name: rook-ceph
  namespace: rook-ceph
spec:
  cephVersion:
    image: quay.io/ceph/ceph:v19.2.0
    allowUnsupported: false
  dataDirHostPath: /var/lib/rook
  mon:
    count: 3
    allowMultiplePerNode: false
  mgr:
    count: 2
    modules:
      - name: pg_autoscaler
        enabled: true
      - name: rook
        enabled: true
  dashboard:
    enabled: true
    ssl: true
  storage:
    useAllNodes: true
    useAllDevices: true
    config:
      osdsPerDevice: "1"
  network:
    provider: host
  resources:
    mon:
      requests:
        cpu: "1"
        memory: "2Gi"
    osd:
      requests:
        cpu: "2"
        memory: "4Gi"
```

## CephBlockPool

### Replicated Pool

```yaml
apiVersion: ceph.rook.io/v1
kind: CephBlockPool
metadata:
  name: replicapool
  namespace: rook-ceph
spec:
  failureDomain: host
  replicated:
    size: 3
    requireSafeReplicaSize: true
  parameters:
    compression_mode: aggressive
```

### StorageClass for Block

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: rook-ceph-block
provisioner: rook-ceph.rbd.csi.ceph.com
parameters:
  clusterID: rook-ceph
  pool: replicapool
  imageFormat: "2"
  imageFeatures: layering,fast-diff,object-map,deep-flatten,exclusive-lock
  csi.storage.k8s.io/provisioner-secret-name: rook-csi-rbd-provisioner
  csi.storage.k8s.io/provisioner-secret-namespace: rook-ceph
  csi.storage.k8s.io/controller-expand-secret-name: rook-csi-rbd-provisioner
  csi.storage.k8s.io/controller-expand-secret-namespace: rook-ceph
  csi.storage.k8s.io/node-stage-secret-name: rook-csi-rbd-node
  csi.storage.k8s.io/node-stage-secret-namespace: rook-ceph
  csi.storage.k8s.io/fstype: ext4
reclaimPolicy: Delete
allowVolumeExpansion: true
```

## CephFilesystem

### Shared Filesystem (CephFS)

```yaml
apiVersion: ceph.rook.io/v1
kind: CephFilesystem
metadata:
  name: ceph-filesystem
  namespace: rook-ceph
spec:
  metadataPool:
    replicated:
      size: 3
  dataPools:
    - name: default
      replicated:
        size: 3
    - name: ec-data
      erasureCoded:
        dataChunks: 2
        codingChunks: 1
  metadataServer:
    activeCount: 1
    activeStandby: true
    resources:
      requests:
        cpu: "1"
        memory: "4Gi"
```

### CephFS StorageClass

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: rook-cephfs
provisioner: rook-ceph.cephfs.csi.ceph.com
parameters:
  clusterID: rook-ceph
  fsName: ceph-filesystem
  pool: ceph-filesystem-default
  csi.storage.k8s.io/provisioner-secret-name: rook-csi-cephfs-provisioner
  csi.storage.k8s.io/provisioner-secret-namespace: rook-ceph
  csi.storage.k8s.io/node-stage-secret-name: rook-csi-cephfs-node
  csi.storage.k8s.io/node-stage-secret-namespace: rook-ceph
reclaimPolicy: Delete
allowVolumeExpansion: true
```

## CephObjectStore

### S3-Compatible Object Store

```yaml
apiVersion: ceph.rook.io/v1
kind: CephObjectStore
metadata:
  name: ceph-objectstore
  namespace: rook-ceph
spec:
  metadataPool:
    replicated:
      size: 3
  dataPool:
    erasureCoded:
      dataChunks: 2
      codingChunks: 1
  gateway:
    type: s3
    port: 80
    securePort: 443
    instances: 2
    resources:
      requests:
        cpu: "1"
        memory: "2Gi"
```

```bash
# Get S3 credentials after creating CephObjectStoreUser
kubectl -n rook-ceph get secret rook-ceph-object-user-ceph-objectstore-my-user \
  -o jsonpath='{.data.AccessKey}' | base64 -d
kubectl -n rook-ceph get secret rook-ceph-object-user-ceph-objectstore-my-user \
  -o jsonpath='{.data.SecretKey}' | base64 -d
```

## OSD Management

### OSD Operations

```bash
# Check OSD status
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd tree
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd status

# Mark OSD out (before removing)
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd out osd.3

# Remove OSD (purge)
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd purge osd.3 --yes-i-really-mean-it

# Check rebalancing progress
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph -w
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph status
```

### PG Autoscaling

```bash
# Check PG autoscaler status
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd pool autoscale-status

# Enable autoscaler on a pool
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd pool set replicapool pg_autoscale_mode on

# Set target ratio
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd pool set replicapool target_size_ratio 0.3
```

## Dashboard

### Accessing the Dashboard

```bash
# Get dashboard password
kubectl -n rook-ceph get secret rook-ceph-dashboard-password \
  -o jsonpath="{['data']['password']}" | base64 -d

# Port-forward dashboard
kubectl -n rook-ceph port-forward svc/rook-ceph-mgr-dashboard 8443:8443

# Access at https://localhost:8443
# Username: admin
```

## Monitoring

### Prometheus and Health Checks

```bash
# Enable Prometheus module
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph mgr module enable prometheus
# Metrics at rook-ceph-mgr:9283/metrics

# Health checks via toolbox
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph health detail
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph df
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd df
```

## Upgrades

### Ceph and Rook Upgrades

```bash
# Upgrade Rook operator (Helm)
helm upgrade --namespace rook-ceph rook-ceph rook-release/rook-ceph --version v1.15.1

# Upgrade Ceph version (patch CephCluster)
kubectl -n rook-ceph patch cephcluster rook-ceph --type=merge \
  -p '{"spec":{"cephVersion":{"image":"quay.io/ceph/ceph:v19.2.1"}}}'

# Monitor upgrade progress
kubectl -n rook-ceph get pods -w
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph versions
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph status
```

## Disaster Recovery

### Backup and Recovery

```bash
# Export cluster config
kubectl -n rook-ceph get cephcluster rook-ceph -o yaml > cluster-backup.yaml

# Mon recovery (if quorum lost): scale operator to 0, patch mon count to 1, scale back
kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas=0
kubectl -n rook-ceph patch cephcluster rook-ceph --type=merge \
  -p '{"spec":{"mon":{"count":1}}}'
kubectl -n rook-ceph scale deploy rook-ceph-operator --replicas=1

# Set noout before maintenance to prevent rebalancing
kubectl -n rook-ceph exec deploy/rook-ceph-tools -- ceph osd set noout
```

## Tips

- Always deploy the toolbox pod (`toolbox.yaml`) — it is essential for running `ceph` CLI commands for debugging
- Set `failureDomain: host` in pools to survive entire node failures, not just disk failures
- Enable `pg_autoscaler` module to avoid manual PG tuning — it handles most workloads correctly
- Use erasure coding (2+1 or 4+2) for cold/archive data to save 33-50% space versus 3x replication
- Set resource requests and limits on all Ceph daemons — OSD memory bloat can cause OOM kills
- Monitor `ceph health detail` regularly and fix HEALTH_WARN issues before they escalate to HEALTH_ERR
- Use device classes (hdd/ssd/nvme) with CRUSH rules to pin pools to specific storage tiers
- Back up the CRUSH map before making topology changes: `ceph osd getcrushmap -o crush.bin`
- During upgrades, set `noout` flag to prevent unnecessary rebalancing: `ceph osd set noout`
- Use `host` network mode for production clusters to avoid container network overhead on OSD traffic
- Test CephFS and RGW independently before combining them — each has distinct failure modes

## See Also

- ceph, longhorn, zfs, lvm, btrfs

## References

- [Rook Official Documentation](https://rook.io/docs/rook/latest/)
- [Rook GitHub Repository](https://github.com/rook/rook)
- [Rook CephCluster CRD Reference](https://rook.io/docs/rook/latest/CRDs/Cluster/ceph-cluster-crd/)
- [Ceph Official Documentation](https://docs.ceph.com/en/latest/)
- [Rook Troubleshooting Guide](https://rook.io/docs/rook/latest/Troubleshooting/ceph-common-issues/)
