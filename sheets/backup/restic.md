# Restic (Deduplicated Backup)

Fast, secure, deduplicated backup tool supporting multiple backends including local, SFTP, S3, REST server, Azure, GCS, and B2 with snapshot-based versioning and encryption by default.

## Installation

```bash
# Linux
apt install restic                    # Debian/Ubuntu
dnf install restic                    # Fedora/RHEL
brew install restic                   # macOS

# Self-update
restic self-update
```

## Repository Initialization

```bash
# Local repository
restic init --repo /backup/restic-repo

# SFTP
restic init --repo sftp:user@host:/backup/restic-repo

# S3
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
restic init --repo s3:s3.amazonaws.com/bucket-name

# S3-compatible (MinIO, Wasabi)
restic init --repo s3:https://minio.example.com/backup

# REST server
restic init --repo rest:https://user:pass@backup.example.com:8000/

# Azure Blob Storage
export AZURE_ACCOUNT_NAME=myaccount
export AZURE_ACCOUNT_KEY=mykey
restic init --repo azure:container-name:/

# Google Cloud Storage
export GOOGLE_PROJECT_ID=my-project
restic init --repo gs:bucket-name:/
```

## Creating Backups

```bash
# Basic backup
restic -r /backup/repo backup /home/user

# Multiple paths
restic -r /backup/repo backup /etc /home /var/www

# With tags
restic -r /backup/repo backup --tag daily --tag server1 /data

# Exclude patterns
restic -r /backup/repo backup \
  --exclude="*.tmp" \
  --exclude=".cache" \
  --exclude-caches \
  /home/user

# Exclude file (one pattern per line)
restic -r /backup/repo backup \
  --exclude-file=/etc/restic/excludes.txt \
  /home/user

# Dry run (see what would be backed up)
restic -r /backup/repo backup --dry-run /data

# Read paths from stdin
find /data -name "*.conf" | restic -r /backup/repo backup --stdin
```

## Snapshot Management

```bash
# List snapshots
restic -r /backup/repo snapshots

# List with filters
restic -r /backup/repo snapshots --tag daily
restic -r /backup/repo snapshots --host server1
restic -r /backup/repo snapshots --path /home

# Detailed snapshot info
restic -r /backup/repo cat snapshot <snapshot-id>

# Show files in a snapshot
restic -r /backup/repo ls latest
restic -r /backup/repo ls <snapshot-id> /etc/

# Diff two snapshots
restic -r /backup/repo diff <snap-id-1> <snap-id-2>

# Show stats
restic -r /backup/repo stats
restic -r /backup/repo stats --mode raw-data
```

## Restoring Data

```bash
# Restore full snapshot
restic -r /backup/repo restore latest --target /restore/

# Restore specific paths
restic -r /backup/repo restore latest \
  --target /restore/ \
  --include "/home/user/documents"

# Restore excluding paths
restic -r /backup/repo restore latest \
  --target /restore/ \
  --exclude "*.log"

# Restore specific snapshot
restic -r /backup/repo restore <snapshot-id> --target /restore/

# Dump a single file to stdout
restic -r /backup/repo dump latest /etc/nginx/nginx.conf > nginx.conf
```

## Mount (FUSE)

```bash
# Mount repository as filesystem
mkdir /mnt/restic
restic -r /backup/repo mount /mnt/restic

# Browse snapshots
ls /mnt/restic/snapshots/
ls /mnt/restic/snapshots/latest/

# Unmount
fusermount -u /mnt/restic               # Linux
umount /mnt/restic                       # macOS
```

## Forget & Prune (Retention)

```bash
# Forget by policy (keep snapshots per schedule)
restic -r /backup/repo forget \
  --keep-last 5 \
  --keep-daily 7 \
  --keep-weekly 4 \
  --keep-monthly 12 \
  --keep-yearly 3

# Forget and prune in one step
restic -r /backup/repo forget \
  --keep-daily 7 \
  --keep-weekly 4 \
  --prune

# Forget specific snapshot
restic -r /backup/repo forget <snapshot-id>

# Forget dry run
restic -r /backup/repo forget --keep-last 3 --dry-run

# Prune unreferenced data
restic -r /backup/repo prune

# Prune with max percentage unused
restic -r /backup/repo prune --max-unused 5%
```

## Keys & Passwords

```bash
# List repository keys
restic -r /backup/repo key list

# Add new key
restic -r /backup/repo key add

# Remove a key
restic -r /backup/repo key remove <key-id>

# Change password
restic -r /backup/repo key passwd

# Password from file
restic -r /backup/repo --password-file /etc/restic/password backup /data

# Password from command
restic -r /backup/repo --password-command "gpg --decrypt /etc/restic/pw.gpg" backup /data
```

## Checking Repository Integrity

```bash
# Quick check (metadata only)
restic -r /backup/repo check

# Full check (read all data)
restic -r /backup/repo check --read-data

# Check subset of packs
restic -r /backup/repo check --read-data-subset=5%
```

## Automation with Cron

```bash
# /etc/restic/backup.sh
#!/bin/bash
export RESTIC_REPOSITORY="s3:s3.amazonaws.com/my-backup"
export RESTIC_PASSWORD_FILE="/etc/restic/password"
export AWS_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
export AWS_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/example"

restic backup /home /etc /var/www --tag automated
restic forget --keep-daily 7 --keep-weekly 4 --keep-monthly 6 --prune
restic check
```

```bash
# Crontab entry
0 2 * * * /etc/restic/backup.sh >> /var/log/restic.log 2>&1
```

## Environment Variables

```bash
export RESTIC_REPOSITORY="/backup/repo"   # default repository
export RESTIC_PASSWORD="mysecretpassword" # password (not recommended)
export RESTIC_PASSWORD_FILE="/etc/restic/password"
export RESTIC_CACHE_DIR="/var/cache/restic"
export RESTIC_COMPRESSION="auto"          # auto, off, max
export RESTIC_PROGRESS_FPS="1"            # progress update rate
```

## REST Server

```bash
# Install rest-server
go install github.com/restic/rest-server/cmd/rest-server@latest

# Run rest-server
rest-server --path /backup/data --listen :8000

# With authentication
rest-server --path /backup/data --listen :8000 --htpasswd-file /etc/restic/.htpasswd

# With TLS
rest-server --path /backup/data --tls --tls-cert cert.pem --tls-key key.pem
```

## Tips

- Always test restores regularly. A backup you cannot restore is worthless.
- Use `--password-file` instead of environment variables for passwords in scripts to avoid leaking secrets in process lists.
- Run `restic check --read-data-subset=5%` weekly and full `--read-data` monthly to verify repository integrity without excessive I/O.
- Set `RESTIC_COMPRESSION=max` for S3/remote backends to reduce transfer costs (restic 0.14+).
- Use `--exclude-caches` to skip directories containing a `CACHEDIR.TAG` file.
- Combine `forget --prune` in a single command to avoid orphaned data packs accumulating between runs.
- Tag backups with host and purpose (`--tag daily --tag db`) to make filtering snapshots easier.
- Use rest-server with `--append-only` mode for ransomware protection so compromised hosts cannot delete old snapshots.
- Lock the repository during long operations with `restic unlock` if a stale lock blocks subsequent runs.
- Bandwidth limit with `--limit-upload` and `--limit-download` (KiB/s) for shared or metered connections.
- Use `restic copy` to replicate snapshots between repositories for off-site redundancy.
- Keep the cache directory (`~/.cache/restic`) persistent to speed up subsequent operations significantly.

## See Also

borgbackup, rsync, rclone, velero, tar

## References

- [Restic Documentation](https://restic.readthedocs.io/)
- [Restic GitHub Repository](https://github.com/restic/restic)
- [REST Server Documentation](https://github.com/restic/rest-server)
- [Restic Design Document](https://restic.readthedocs.io/en/latest/100_references.html)
- [Restic Man Pages](https://restic.readthedocs.io/en/latest/manual_rest.html)
