# Velero (Kubernetes Backup & Restore)

Kubernetes-native backup and restore tool for cluster resources and persistent volumes, supporting scheduled backups, disaster recovery, cluster migration, and integration with cloud-native storage snapshots.

## Installation

```bash
# Download CLI
curl -fsSL https://github.com/vmware-tanzu/velero/releases/download/v1.14.0/velero-v1.14.0-linux-amd64.tar.gz | \
  tar xz && mv velero-v1.14.0-linux-amd64/velero /usr/local/bin/

# macOS
brew install velero

# Verify
velero version
```

## Server Install (AWS Example)

```bash
# Install Velero server with AWS plugin
velero install \
  --provider aws \
  --bucket my-velero-backups \
  --secret-file ./credentials-velero \
  --backup-location-config region=us-east-1 \
  --snapshot-location-config region=us-east-1 \
  --plugins velero/velero-plugin-for-aws:v1.10.0

# Verify installation
kubectl get pods -n velero
velero version
```

### Credential File Format (AWS)

```ini
# credentials-velero
[default]
aws_access_key_id=AKIAIOSFODNN7EXAMPLE
aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
```

## Creating Backups

```bash
# Full cluster backup
velero backup create full-backup

# Backup specific namespace
velero backup create ns-backup --include-namespaces production

# Multiple namespaces
velero backup create multi-ns --include-namespaces production,staging

# Exclude namespaces
velero backup create cluster-backup --exclude-namespaces kube-system,velero

# Backup specific resources
velero backup create deploy-backup \
  --include-resources deployments,services,configmaps

# Label selector
velero backup create app-backup \
  --selector app=my-app

# Exclude resources
velero backup create no-secrets \
  --exclude-resources secrets

# With TTL (auto-delete after duration)
velero backup create daily-backup \
  --include-namespaces production \
  --ttl 720h                                # 30 days

# Include cluster-scoped resources
velero backup create full-backup \
  --include-cluster-resources=true

# Wait for completion
velero backup create my-backup --wait
```

## Scheduled Backups

```bash
# Daily backup at midnight
velero schedule create daily-prod \
  --schedule="0 0 * * *" \
  --include-namespaces production \
  --ttl 720h

# Every 6 hours
velero schedule create frequent \
  --schedule="0 */6 * * *" \
  --include-namespaces production,staging

# Weekly backup (Sunday 2am)
velero schedule create weekly-full \
  --schedule="0 2 * * 0" \
  --ttl 2160h                               # 90 days

# List schedules
velero schedule get

# Describe schedule
velero schedule describe daily-prod

# Trigger schedule manually
velero backup create --from-schedule daily-prod

# Pause schedule
velero schedule pause daily-prod

# Unpause schedule
velero schedule unpause daily-prod

# Delete schedule
velero schedule delete daily-prod
```

## Listing & Inspecting Backups

```bash
# List all backups
velero backup get

# Describe backup (detailed)
velero backup describe full-backup

# Describe with volume details
velero backup describe full-backup --details

# View backup logs
velero backup logs full-backup

# Download backup
velero backup download full-backup
```

## Restoring

```bash
# Restore from backup (full)
velero restore create --from-backup full-backup

# Named restore
velero restore create my-restore --from-backup full-backup

# Restore specific namespaces
velero restore create --from-backup full-backup \
  --include-namespaces production

# Restore specific resources
velero restore create --from-backup full-backup \
  --include-resources deployments,services

# Restore with namespace mapping (migration)
velero restore create --from-backup full-backup \
  --namespace-mappings old-ns:new-ns

# Restore excluding resources
velero restore create --from-backup full-backup \
  --exclude-resources storageclasses,nodes

# Restore with label selector
velero restore create --from-backup full-backup \
  --selector app=critical

# Restore status
velero restore get
velero restore describe my-restore
velero restore logs my-restore
```

## Volume Snapshots

### Using CSI Snapshots

```bash
# Install CSI plugin
velero install --features=EnableCSI \
  --plugins velero/velero-plugin-for-csi:v0.7.0,...

# Verify VolumeSnapshotClass
kubectl get volumesnapshotclass

# Backup with CSI snapshots (automatic for CSI-backed PVCs)
velero backup create pvc-backup --include-namespaces production
```

### Using Restic/Kopia (File-Level)

```bash
# Install with file-system backup enabled
velero install --use-node-agent \
  --uploader-type=kopia ...

# Annotate pods to opt-in for FS backup
kubectl annotate pod my-pod \
  backup.velero.io/backup-volumes=data-volume

# Annotate to opt-out
kubectl annotate pod my-pod \
  backup.velero.io/backup-volumes-excludes=cache-volume
```

## Backup Locations

```bash
# List backup storage locations
velero backup-location get

# Create additional location
velero backup-location create secondary \
  --provider aws \
  --bucket secondary-bucket \
  --config region=eu-west-1

# Set default backup location
velero backup-location set primary --default

# Snapshot locations
velero snapshot-location get
```

## Disaster Recovery

```bash
# Scenario: Restore entire cluster from scratch

# 1. Install Velero on new cluster pointing to same bucket
velero install \
  --provider aws \
  --bucket my-velero-backups \
  --secret-file ./credentials-velero \
  --backup-location-config region=us-east-1

# 2. Verify backups are visible
velero backup get

# 3. Restore everything
velero restore create full-dr \
  --from-backup latest-full-backup

# 4. Monitor progress
velero restore describe full-dr
velero restore logs full-dr
```

## Cluster Migration

```bash
# Source cluster: create backup
velero backup create migration-backup \
  --include-namespaces app-ns \
  --include-cluster-resources=true \
  --snapshot-volumes=false \
  --default-volumes-to-fs-backup

# Target cluster: install Velero with same bucket
velero install --provider aws --bucket my-velero-backups ...

# Target cluster: restore
velero restore create migration-restore \
  --from-backup migration-backup \
  --namespace-mappings old-ns:new-ns
```

## Hooks (Pre/Post Backup)

```yaml
# Pod annotation for backup hooks
apiVersion: v1
kind: Pod
metadata:
  annotations:
    # Pre-backup hook (e.g., flush database)
    pre.hook.backup.velero.io/command: '["/bin/sh", "-c", "pg_dump > /backup/dump.sql"]'
    pre.hook.backup.velero.io/container: postgres
    pre.hook.backup.velero.io/timeout: 120s
    # Post-backup hook
    post.hook.backup.velero.io/command: '["/bin/sh", "-c", "rm /backup/dump.sql"]'
```

## Troubleshooting

```bash
# Check Velero pod logs
kubectl logs -n velero deployment/velero

# Check node-agent logs (for FS backups)
kubectl logs -n velero daemonset/node-agent

# Describe failed backup
velero backup describe failed-backup --details

# Check BSL validity
velero backup-location get

# Debug plugin issues
velero backup create test --log-level debug
kubectl logs -n velero deployment/velero -c velero
```

## Tips

- Set appropriate TTL on backups to prevent unbounded storage growth. Use shorter TTL for frequent schedules and longer for weekly/monthly.
- Always test restores in a staging cluster before relying on backups for disaster recovery.
- Use `--include-cluster-resources=true` when backing up for migration to capture ClusterRoles, CRDs, and other cluster-scoped objects.
- Annotate pods with `backup.velero.io/backup-volumes` to explicitly control which volumes get file-level backup.
- Use namespace mappings during restore (`--namespace-mappings`) to avoid conflicts when restoring to the same cluster.
- Schedule a weekly `velero backup describe` review to catch silently failing backups before you need them.
- For databases, use pre-backup hooks to create consistent dumps before the volume snapshot is taken.
- Prefer CSI volume snapshots over file-level (Kopia/Restic) backups for large volumes as they are significantly faster.
- Store backup credentials in a Kubernetes Secret and reference it rather than passing credential files directly.
- Use `--snapshot-volumes=false --default-volumes-to-fs-backup` for cross-provider migration since volume snapshots are provider-specific.
- Run `velero backup-location get` regularly to verify the storage location remains accessible and valid.
- Keep Velero and its plugins at matching versions to avoid compatibility issues during backup and restore.

## See Also

restic, borgbackup, kubectl, etcd, kustomize

## References

- [Velero Documentation](https://velero.io/docs/)
- [Velero GitHub Repository](https://github.com/vmware-tanzu/velero)
- [Velero Plugin for AWS](https://github.com/vmware-tanzu/velero-plugin-for-aws)
- [Velero Plugin for GCP](https://github.com/vmware-tanzu/velero-plugin-for-gcp)
- [Velero Plugin for Azure](https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure)
- [Velero CSI Plugin](https://github.com/vmware-tanzu/velero-plugin-for-csi)
- [Velero Troubleshooting](https://velero.io/docs/main/troubleshooting/)
