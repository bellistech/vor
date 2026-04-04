# BorgBackup (Deduplicated Encrypted Backup)

Space-efficient backup tool with deduplication, authenticated encryption, compression (lz4, zstd, zlib, lzma), and support for local and remote (SSH) repositories with FUSE mounting for browsing archives.

## Installation

```bash
# Linux
apt install borgbackup                   # Debian/Ubuntu
dnf install borgbackup                   # Fedora/RHEL
pip install borgbackup                   # pip

# macOS
brew install borgbackup

# Verify
borg --version
```

## Repository Initialization

```bash
# Local repository (repokey encryption — key in repo, passphrase-protected)
borg init --encryption=repokey /backup/borg-repo

# Remote repository via SSH
borg init --encryption=repokey ssh://user@host:22/backup/borg-repo

# Keyfile encryption (key stored locally in ~/.config/borg/keys/)
borg init --encryption=keyfile /backup/borg-repo

# No encryption (not recommended)
borg init --encryption=none /backup/borg-repo

# Authenticated but not encrypted
borg init --encryption=authenticated /backup/borg-repo
```

## Creating Archives

```bash
# Basic archive
borg create /backup/repo::archive-name /home/user

# Archive with timestamp
borg create /backup/repo::{hostname}-{now:%Y-%m-%d_%H:%M} /home /etc

# With compression
borg create --compression lz4 /backup/repo::archive /data        # fast
borg create --compression zstd,3 /backup/repo::archive /data     # balanced
borg create --compression zlib,6 /backup/repo::archive /data     # good ratio
borg create --compression lzma,6 /backup/repo::archive /data     # best ratio, slow

# Exclude patterns
borg create --exclude '*.tmp' \
            --exclude 'home/*/.cache' \
            --exclude-caches \
            /backup/repo::archive /home

# Exclude from file
borg create --exclude-from /etc/borg/excludes.txt \
            /backup/repo::archive /home

# With progress and stats
borg create --progress --stats \
            /backup/repo::archive /data

# Dry run
borg create --dry-run --list /backup/repo::archive /data

# Read special files (devices, fifos)
borg create --read-special /backup/repo::archive /dev/sda
```

## Listing & Info

```bash
# List all archives in a repository
borg list /backup/repo

# List contents of an archive
borg list /backup/repo::archive-name

# List with pattern matching
borg list --glob-archives '*daily*' /backup/repo

# Detailed archive info
borg info /backup/repo::archive-name

# Repository info
borg info /backup/repo

# JSON output
borg list --json /backup/repo
borg info --json /backup/repo::archive-name
```

## Extracting (Restore)

```bash
# Extract full archive to current directory
cd /restore
borg extract /backup/repo::archive-name

# Extract specific paths
borg extract /backup/repo::archive-name home/user/documents

# Extract with pattern matching
borg extract /backup/repo::archive-name --pattern '*.conf'

# Dry run (see what would be extracted)
borg extract --dry-run --list /backup/repo::archive-name

# Extract to stdout (single file)
borg extract --stdout /backup/repo::archive-name etc/nginx/nginx.conf > nginx.conf
```

## Mounting Archives (FUSE)

```bash
# Mount entire repository
mkdir /mnt/borg
borg mount /backup/repo /mnt/borg

# Mount single archive
borg mount /backup/repo::archive-name /mnt/borg

# Browse mounted data
ls /mnt/borg/
cp /mnt/borg/home/user/important.txt /restore/

# Unmount
borg umount /mnt/borg
```

## Pruning Archives

```bash
# Prune by retention policy
borg prune --keep-daily 7 \
           --keep-weekly 4 \
           --keep-monthly 6 \
           --keep-yearly 2 \
           /backup/repo

# Prune with prefix filter
borg prune --glob-archives '{hostname}-*' \
           --keep-daily 7 \
           --keep-weekly 4 \
           /backup/repo

# Dry run
borg prune --dry-run --list \
           --keep-daily 7 \
           /backup/repo

# Compact after pruning (borg 1.2+)
borg compact /backup/repo

# Delete specific archive
borg delete /backup/repo::archive-name
```

## Key Management

```bash
# Export key (CRITICAL — back this up)
borg key export /backup/repo /safe/location/borg-key.txt

# Import key
borg key import /backup/repo /safe/location/borg-key.txt

# Change passphrase
borg key change-passphrase /backup/repo
```

## Integrity Checking

```bash
# Verify repository integrity
borg check /backup/repo

# Check and verify data (reads all chunks)
borg check --verify-data /backup/repo

# Check specific archive
borg check --last 5 /backup/repo

# Repair (use with caution)
borg check --repair /backup/repo
```

## Remote Repositories

```bash
# SSH connection
borg create ssh://user@backup-server:22/backup/repo::archive /data

# With custom SSH command
export BORG_RSH="ssh -i /home/user/.ssh/borg_key -p 2222"
borg create user@host:/backup/repo::archive /data

# Rate limiting
borg create --remote-ratelimit 5000 \     # 5000 KiB/s upload limit
    ssh://user@host/backup/repo::archive /data
```

## Automation Script

```bash
#!/bin/bash
# /etc/borg/backup.sh
export BORG_REPO="ssh://borg@backup-server/backup/repo"
export BORG_PASSPHRASE="$(cat /etc/borg/passphrase)"

# Create archive
borg create --compression zstd,3 \
    --exclude-caches \
    --exclude '/home/*/.cache' \
    --exclude '/var/tmp/*' \
    ::{hostname}-{now:%Y-%m-%d_%H:%M} \
    /home /etc /var/www /var/lib/postgresql

# Prune old archives
borg prune --keep-daily 7 --keep-weekly 4 --keep-monthly 6

# Compact repository
borg compact

# Verify integrity
borg check --last 3
```

## Environment Variables

```bash
export BORG_REPO="/backup/repo"              # default repository
export BORG_PASSPHRASE="secret"              # passphrase (use BORG_PASSCOMMAND instead)
export BORG_PASSCOMMAND="gpg --decrypt /etc/borg/pass.gpg"
export BORG_RSH="ssh -i ~/.ssh/borg_key"     # custom SSH command
export BORG_CACHE_DIR="/var/cache/borg"      # cache directory
export BORG_FILES_CACHE_SUFFIX=".cache"      # files cache per repo
```

## Tips

- Always export your encryption key with `borg key export` and store it separately. Without the key the repository is permanently unrecoverable.
- Use `--compression zstd,3` for a good balance of speed and ratio. Use `lz4` when CPU is the bottleneck and `lzma,6` for cold archival.
- Run `borg compact` after `borg prune` in borg 1.2+ because pruning only marks data as deleted without reclaiming disk space.
- Use `BORG_PASSCOMMAND` with a secrets manager or GPG instead of `BORG_PASSPHRASE` to keep passphrases out of environment dumps.
- Append-only mode (`--append-only` in `borg serve`) prevents a compromised client from deleting old archives.
- Test restores regularly with `borg extract --dry-run --list` to verify archives contain the expected files.
- Use `--exclude-caches` and `--exclude-if-present .nobackup` to let directories self-declare exclusion.
- Pin borg version on both client and server. Major version mismatches can cause protocol incompatibilities.
- Use `--checkpoint-interval 300` for long-running backups to allow resumption if interrupted.
- The `--one-file-system` flag prevents accidentally backing up mounted network shares or pseudo-filesystems.
- Monitor `borg info` output for repository size growth to detect dedup ratio degradation early.
- Use `borg diff archive1 archive2` to inspect exactly what changed between two backups.

## See Also

restic, rsync, rclone, tar, velero

## References

- [BorgBackup Documentation](https://borgbackup.readthedocs.io/)
- [BorgBackup GitHub Repository](https://github.com/borgbackup/borg)
- [BorgBackup Installation](https://borgbackup.readthedocs.io/en/stable/installation.html)
- [BorgBackup FAQ](https://borgbackup.readthedocs.io/en/stable/faq.html)
- [BorgBase — Hosted Borg Repos](https://www.borgbase.com/)
- [Borgmatic — Borg Wrapper](https://torsion.org/borgmatic/)
