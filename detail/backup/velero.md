# The Mathematics of Velero — Kubernetes State Serialization and Recovery

> *Velero's backup and restore operations involve graph theory over Kubernetes resource dependency DAGs, consistency models for distributed state capture, and storage cost optimization through retention policy scheduling.*

---

## 1. Kubernetes Resource Graph (Dependency DAG)

### Resource Dependency Model

A Kubernetes cluster's state forms a directed acyclic graph $G = (V, E)$ where vertices are resources and edges are ownership/reference relationships:

$$V = \{r_1, r_2, \ldots, r_n\} \quad E = \{(r_i, r_j) : r_i \text{ depends on } r_j\}$$

### Resource Categories

| Category | Examples | Count (typical cluster) |
|:---|:---|:---:|
| Cluster-scoped | Namespaces, CRDs, ClusterRoles | 50-200 |
| Namespace-scoped | Deployments, Services, ConfigMaps | 500-10,000 |
| Volume resources | PVCs, PVs, VolumeSnapshots | 50-1,000 |
| Generated | Pods, ReplicaSets, Endpoints | 1,000-50,000 |

### Backup Serialization Order

Velero must serialize resources respecting dependency order. For a DAG with $n$ nodes and $e$ edges:

$$T_{serialize} = O(n + e) \quad (\text{topological sort})$$

The restore order is the reverse topological sort to ensure dependencies exist before dependents:

$$\text{Restore Order} = \text{reverse}(\text{TopSort}(G))$$

### Restore Conflict Resolution

When restoring to an existing cluster, for each resource $r$:

$$\text{Action}(r) = \begin{cases} \text{create} & \text{if } r \notin \text{cluster} \\ \text{patch/merge} & \text{if policy = merge} \\ \text{skip} & \text{if policy = skip (default)} \end{cases}$$

---

## 2. Backup Size Model

### Resource Serialization Size

Each Kubernetes resource serialized to JSON:

$$S_{resource} = S_{metadata} + S_{spec} + S_{status}$$

Typical sizes:

| Resource Type | Avg JSON Size | Per 100 Resources |
|:---|:---:|:---:|
| ConfigMap | 2-50 KiB | 0.2-5 MiB |
| Secret | 1-10 KiB | 0.1-1 MiB |
| Deployment | 3-8 KiB | 0.3-0.8 MiB |
| Service | 1-3 KiB | 0.1-0.3 MiB |
| CRD instance | 5-100 KiB | 0.5-10 MiB |
| Full namespace (100 resources) | - | 5-50 MiB |

### Total Backup Size

$$S_{backup} = S_{metadata\_tarball} + S_{resource\_json} + S_{volume\_data}$$

$$S_{metadata\_tarball} = \sum_{i=1}^{n} S_{resource_i} + 512n \quad (\text{tar headers})$$

For volume snapshots, the snapshot itself is stored by the cloud provider, so:

$$S_{stored} = S_{metadata\_tarball} \quad (\text{volume snapshots are references only})$$

For file-level backup (Kopia/Restic):

$$S_{stored} = S_{metadata\_tarball} + \sum_{j=1}^{m} S_{volume_j} \times (1 - \text{dedup ratio}_j)$$

---

## 3. Retention Policy and Storage Cost

### TTL-Based Retention

With schedule interval $I$ and TTL $T$:

$$\text{Max Concurrent Backups} = \left\lfloor \frac{T}{I} \right\rfloor + 1$$

| Schedule | Interval | TTL | Max Backups | Storage (50 MiB each) |
|:---|:---:|:---:|:---:|:---:|
| Hourly | 1h | 48h | 49 | 2.4 GiB |
| Every 6h | 6h | 168h (7d) | 29 | 1.4 GiB |
| Daily | 24h | 720h (30d) | 31 | 1.5 GiB |
| Weekly | 168h | 2160h (90d) | 14 | 0.7 GiB |

### Storage Cost Model

$$C_{monthly} = \text{Max Backups} \times S_{backup} \times P_{storage} + N_{snapshots} \times S_{snap} \times P_{snapshot}$$

Where $P_{storage}$ is per-GiB object storage cost and $P_{snapshot}$ is per-GiB snapshot cost.

| Provider | Object Storage ($/GiB/mo) | Snapshot ($/GiB/mo) |
|:---|:---:|:---:|
| AWS S3 Standard | $0.023 | $0.05 (EBS) |
| AWS S3-IA | $0.0125 | $0.05 (EBS) |
| GCS Standard | $0.020 | $0.04 (PD) |
| Azure Blob Hot | $0.018 | $0.05 (Managed Disk) |

### Optimal Schedule

Minimize cost while meeting RPO (Recovery Point Objective):

$$\min_{I, T} \quad C(I, T) = \frac{T}{I} \times S_{backup} \times P$$

$$\text{subject to} \quad I \leq \text{RPO}$$

---

## 4. Recovery Time and RPO/RTO Analysis

### Recovery Point Objective (RPO)

$$\text{RPO} = I + T_{backup}$$

Maximum data loss equals the backup interval plus the time to complete a backup:

$$\text{Worst-case data loss} = I + T_{backup} \approx I \quad (\text{when } T_{backup} \ll I)$$

### Recovery Time Objective (RTO)

$$\text{RTO} = T_{detect} + T_{decide} + T_{restore}$$

$$T_{restore} = T_{metadata} + T_{volumes} + T_{reconcile}$$

| Component | Typical Duration | Depends On |
|:---|:---:|:---|
| $T_{detect}$ | 1-15 min | Monitoring/alerting |
| $T_{decide}$ | 5-60 min | Runbook, approval |
| $T_{metadata}$ | 1-5 min | Resource count |
| $T_{volumes}$ (snapshot) | 5-30 min | Volume size, IOPS |
| $T_{volumes}$ (file-level) | 30-180 min | Data size, bandwidth |
| $T_{reconcile}$ | 5-15 min | Controller reconciliation |

### Restore Parallelism

Velero restores resources in parallel within each priority group:

$$T_{restore\_resources} = \sum_{g=1}^{G} \max_{r \in g} T_{create}(r)$$

where $G$ is the number of priority groups (default: 4 groups for cluster-scoped, namespaces, high-priority, normal).

---

## 5. Volume Snapshot Consistency

### Crash Consistency vs Application Consistency

Without hooks, snapshots are crash-consistent:

$$\text{Crash-consistent}: \quad \text{State} = S(t_{snap}) \quad (\text{point-in-time disk state})$$

With pre-backup hooks (fsync, database flush):

$$\text{App-consistent}: \quad \text{State} = S(t_{flush}) \quad \text{where } t_{flush} \leq t_{snap}$$

### Consistency Window

The inconsistency window for a multi-volume backup:

$$\Delta t_{inconsistency} = \max(t_{snap_i}) - \min(t_{snap_i}) \quad \text{for volumes } i = 1, \ldots, m$$

For CSI group snapshots (VolumeGroupSnapshot):

$$\Delta t_{inconsistency} \approx 0 \quad (\text{atomic multi-volume snapshot})$$

---

## 6. Migration Transfer Model

### Cross-Cluster Migration

Total migration time using file-level backup through object storage:

$$T_{migration} = T_{backup} + T_{upload} + T_{download} + T_{restore}$$

$$T_{upload} = \frac{S_{total}}{BW_{source \to storage}}$$

$$T_{download} = \frac{S_{total}}{BW_{storage \to target}}$$

| Data Volume | Upload (100 Mbps) | Download (100 Mbps) | Total Transfer |
|:---:|:---:|:---:|:---:|
| 10 GiB | 14 min | 14 min | 28 min |
| 100 GiB | 2.3 hr | 2.3 hr | 4.6 hr |
| 1 TiB | 23 hr | 23 hr | 46 hr |
| 10 TiB | 9.6 days | 9.6 days | 19.2 days |

### Resource Transformation

During migration, namespace mappings are an $O(n)$ string replacement across all $n$ resources:

$$T_{transform} = O(n) \quad \text{where } n = \text{total resources}$$

---

## Prerequisites

- Kubernetes resource model (API objects, namespaces, ownership)
- Directed acyclic graphs and topological sorting
- Cloud storage pricing models
- Consistency models (crash, application, transaction)
- Backup theory (RPO, RTO, retention)

## Complexity

| Operation | Time | Space |
|:---|:---|:---|
| Backup (metadata) | $O(n + e)$ resource graph | $O(n)$ serialized resources |
| Backup (volumes, snapshot) | $O(m)$ API calls | Provider-managed |
| Backup (volumes, file-level) | $O(d)$ total data bytes | $O(d)$ in object storage |
| Restore (metadata) | $O(n + e)$ topological order | $O(n)$ in API server |
| Restore (volumes) | $O(d)$ data transfer | $O(d)$ on target PVs |
| Schedule evaluation | $O(1)$ cron check | $O(b)$ backup count |
