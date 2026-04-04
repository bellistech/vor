# etcd (Distributed Key-Value Store)

Strongly consistent distributed key-value store using Raft consensus, providing reliable configuration storage, service discovery, and leader election for distributed systems like Kubernetes.

## etcdctl Basics

```bash
# Set API version (v3 is current)
export ETCDCTL_API=3

# Put and get
etcdctl put /config/db/host "postgres.local"
etcdctl get /config/db/host

# Get with metadata
etcdctl get /config/db/host -w json

# Get value only (no key)
etcdctl get /config/db/host --print-value-only

# Put with previous value returned
etcdctl put /config/db/host "postgres2.local" --prev-kv
```

## Key Operations

```bash
# Get all keys with prefix
etcdctl get /config/ --prefix

# Get all keys
etcdctl get "" --prefix

# Get keys only (no values)
etcdctl get "" --prefix --keys-only

# Get range [start, end)
etcdctl get /config/a /config/z

# Count keys
etcdctl get "" --prefix --count-only

# Delete key
etcdctl del /config/db/host

# Delete with prefix
etcdctl del /config/db/ --prefix

# Delete range
etcdctl del /config/a /config/z

# Delete and return previous value
etcdctl del /config/db/host --prev-kv
```

## Watch

```bash
# Watch a key for changes
etcdctl watch /config/db/host

# Watch prefix
etcdctl watch /config/ --prefix

# Watch from specific revision
etcdctl watch /config/ --prefix --rev=42

# Watch with progress notifications
etcdctl watch /config/ --prefix --progress-notify

# Watch multiple keys
etcdctl watch -i
# watch /config/db/host
# watch /config/db/port

# Watch and execute command on change
etcdctl watch /config/ --prefix -- sh -c 'echo "changed: $ETCD_WATCH_KEY=$ETCD_WATCH_VALUE"'
```

## Leases (TTL)

```bash
# Grant a lease (TTL in seconds)
etcdctl lease grant 60
# Returns: lease 694d81898c1c6513 granted with TTL(60s)

# Put with lease (key expires when lease expires)
etcdctl put /service/web/node1 "alive" --lease=694d81898c1c6513

# Keep lease alive (renew indefinitely)
etcdctl lease keep-alive 694d81898c1c6513

# Get lease info
etcdctl lease timetolive 694d81898c1c6513
etcdctl lease timetolive 694d81898c1c6513 --keys   # show attached keys

# List leases
etcdctl lease list

# Revoke lease (immediately deletes attached keys)
etcdctl lease revoke 694d81898c1c6513
```

## Transactions (Compare-and-Swap)

```bash
# Atomic transaction: if key == value then put else get
etcdctl txn -i
# compares:
# value("/config/lock") = "free"
# success requests:
# put /config/lock "taken"
# failure requests:
# get /config/lock

# One-liner transaction
etcdctl txn -i <<EOF
value("/config/version") = "1"

put /config/version "2"

get /config/version
EOF

# Compare operators:
# value("key") = "val"     — value equals
# version("key") = 0       — key does not exist
# create("key") > 0        — key exists
# mod("key") > 100         — modified after revision 100
```

## Cluster Management

```bash
# Check cluster health
etcdctl endpoint health
etcdctl endpoint health --endpoints=http://etcd1:2379,http://etcd2:2379,http://etcd3:2379

# Cluster status
etcdctl endpoint status -w table
etcdctl endpoint hashkv -w table

# List members
etcdctl member list -w table

# Add member
etcdctl member add etcd4 --peer-urls=http://etcd4:2380

# Remove member
etcdctl member remove <member-id>

# Update member peer URLs
etcdctl member update <member-id> --peer-urls=http://etcd4:2380

# Move leader (force leader election)
etcdctl move-leader <target-member-id>
```

## Compaction & Defrag

```bash
# Get current revision
etcdctl endpoint status -w json | jq '.[0].Status.header.revision'

# Compact history (free up space, remove old revisions)
etcdctl compact 1000                       # compact to revision 1000
etcdctl compact $(etcdctl endpoint status -w json | jq '.[0].Status.header.revision')

# Enable auto-compaction (in etcd config)
# --auto-compaction-retention=1h           # keep 1 hour of history
# --auto-compaction-mode=periodic          # periodic | revision

# Defragment (reclaim disk space after compaction)
etcdctl defrag
etcdctl defrag --endpoints=http://etcd1:2379,http://etcd2:2379

# Check disk usage
etcdctl endpoint status -w table
# DB SIZE column shows current disk usage
```

## Backup & Restore

```bash
# Snapshot (backup)
etcdctl snapshot save /backup/etcd-$(date +%Y%m%d).db

# Verify snapshot
etcdctl snapshot status /backup/etcd-20240101.db -w table

# Restore from snapshot (creates new data directory)
etcdctl snapshot restore /backup/etcd-20240101.db \
  --name=etcd1 \
  --initial-cluster=etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380 \
  --initial-cluster-token=etcd-cluster-1 \
  --initial-advertise-peer-urls=http://etcd1:2380 \
  --data-dir=/var/lib/etcd-restored
```

## Authentication & TLS

```bash
# Enable auth
etcdctl user add root --new-user-password="rootpass"
etcdctl role add root
etcdctl user grant-role root root
etcdctl auth enable

# Create user with role
etcdctl user add appuser --new-user-password="apppass"
etcdctl role add app-readwrite
etcdctl role grant-permission app-readwrite readwrite /app/ --prefix
etcdctl user grant-role appuser app-readwrite

# TLS connection
etcdctl --endpoints=https://etcd1:2379 \
  --cacert=/etc/etcd/ca.pem \
  --cert=/etc/etcd/client.pem \
  --key=/etc/etcd/client-key.pem \
  endpoint health
```

## Configuration Flags

```bash
# Common etcd startup flags
etcd \
  --name=etcd1 \
  --data-dir=/var/lib/etcd \
  --listen-client-urls=http://0.0.0.0:2379 \
  --advertise-client-urls=http://etcd1:2379 \
  --listen-peer-urls=http://0.0.0.0:2380 \
  --initial-advertise-peer-urls=http://etcd1:2380 \
  --initial-cluster=etcd1=http://etcd1:2380,etcd2=http://etcd2:2380,etcd3=http://etcd3:2380 \
  --initial-cluster-token=my-cluster \
  --initial-cluster-state=new \
  --quota-backend-bytes=8589934592 \
  --auto-compaction-retention=1h \
  --max-request-bytes=1572864 \
  --snapshot-count=10000
```

## Tips

- Always run etcd in clusters of 3 or 5 nodes; even numbers provide no advantage and risk split-brain
- Set `--auto-compaction-retention` to prevent unbounded disk growth from revision history
- Run `defrag` after compaction to actually reclaim disk space (compaction only marks revisions as deleted)
- Use leases for service discovery and ephemeral keys; the keep-alive mechanism handles heartbeats
- Set `--quota-backend-bytes` to limit database size (default 2 GB; raise for large Kubernetes clusters)
- Use prefix watches for service discovery patterns where services register under a common prefix
- Take regular snapshots for disaster recovery; snapshots are consistent point-in-time copies
- Use transactions for compare-and-swap operations like distributed locks and leader election
- Monitor the `etcd_disk_wal_fsync_duration_seconds` metric; slow disk kills etcd performance
- Keep key and value sizes small (< 1 MB each); etcd is designed for metadata, not bulk data
- Use `--max-request-bytes` to prevent oversized writes from clients

## See Also

redis, consul, zookeeper, kubernetes, raft

## References

- [etcd Documentation](https://etcd.io/docs/)
- [etcdctl Reference](https://etcd.io/docs/latest/dev-guide/interacting_v3/)
- [Raft Consensus Algorithm](https://raft.github.io/)
- [etcd Operations Guide](https://etcd.io/docs/latest/op-guide/)
- [etcd Performance](https://etcd.io/docs/latest/op-guide/performance/)
