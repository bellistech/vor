# NFS (Network File System)

NFS shares directories over the network, allowing remote clients to mount and access files as if they were local, using a stateless (v3) or stateful (v4+) protocol with configurable caching, authentication, and export permissions.

## Server Setup

### Install NFS Server

```bash
# Debian/Ubuntu
sudo apt install nfs-kernel-server

# RHEL/CentOS/Fedora
sudo dnf install nfs-utils

# Enable and start
sudo systemctl enable --now nfs-server
```

### Export Configuration

```bash
# /etc/exports — define shared directories
# Format: <directory> <client>(options) [client2(options)]

# Share /data to specific subnet (read-write, sync writes)
/data           192.168.1.0/24(rw,sync,no_subtree_check)

# Share /home to single host (read-write, root stays root)
/home           fileserver.local(rw,sync,no_root_squash)

# Share /media read-only to everyone
/media          *(ro,sync,no_subtree_check)

# Share to multiple clients with different permissions
/srv/shared     10.0.0.0/8(rw,sync)  172.16.0.0/12(ro,sync)

# NFSv4 pseudo-root (binds exports under /export)
/export         *(fsid=0,ro,sync,no_subtree_check)
/export/data    192.168.1.0/24(rw,sync,no_subtree_check,nohide)
```

### Export Options Reference

```bash
# Write behavior
rw                  # read-write access
ro                  # read-only access (default)
sync                # reply after changes committed to disk (safe, slower)
async               # reply before commit (fast, risk of corruption on crash)

# User mapping
root_squash         # map root (uid 0) to nobody (default, secure)
no_root_squash      # allow root access (dangerous — use sparingly)
all_squash          # map all users to nobody
anonuid=1000        # set anonymous user UID
anongid=1000        # set anonymous group GID

# Subtree checking
no_subtree_check    # disable subtree check (recommended — faster, more reliable)
subtree_check       # verify file is in exported tree (slow, can cause stale handles)

# Cross-mount
crossmnt            # auto-export filesystems mounted under export
nohide              # show mounted filesystems (NFSv4)
```

### Apply Exports

```bash
sudo exportfs -ra                      # re-export all (apply changes)
sudo exportfs -v                       # show current exports with options
sudo exportfs -u 192.168.1.0/24:/data  # unexport specific share
sudo exportfs -s                       # show exports with security info
```

## Client Setup

### Mount NFS Share

```bash
# Install NFS client
sudo apt install nfs-common             # Debian/Ubuntu
sudo dnf install nfs-utils              # RHEL/CentOS

# Manual mount
sudo mount -t nfs server:/data /mnt/data
sudo mount -t nfs4 server:/data /mnt/data   # force NFSv4

# Mount with options
sudo mount -t nfs -o rw,sync,hard,intr server:/data /mnt/data
sudo mount -t nfs -o vers=4.1,sec=krb5 server:/data /mnt/data

# Unmount
sudo umount /mnt/data
sudo umount -l /mnt/data               # lazy unmount (busy mount)
sudo umount -f /mnt/data               # force unmount
```

### Mount Options (Client)

```bash
# Reliability
hard                # retry indefinitely on failure (default, recommended)
soft                # return error after retrans attempts (risks data corruption)
intr                # allow interrupt of NFS operations (deprecated in newer kernels)
retrans=3           # number of retries before soft failure
timeo=600           # timeout in deciseconds (60s)

# Performance
rsize=1048576       # read buffer size in bytes (1MB, negotiated down if needed)
wsize=1048576       # write buffer size in bytes
async               # buffer writes (faster, less safe)
noatime             # don't update access times (significant perf boost)

# Caching
ac                  # attribute caching on (default)
noac                # disable attribute caching (always check server — slow)
actimeo=30          # cache attributes for 30s (sets acregmin=acregmax=acdirmin=acdirmax)
lookupcache=all     # cache directory lookups (default for v4)

# Security
sec=sys             # AUTH_SYS (UID/GID, default)
sec=krb5            # Kerberos authentication
sec=krb5i           # Kerberos with integrity checking
sec=krb5p           # Kerberos with privacy (encryption)

# Version
vers=3              # force NFSv3
vers=4              # force NFSv4
vers=4.1            # force NFSv4.1 (pNFS support)
vers=4.2            # force NFSv4.2 (server-side copy, sparse files)
```

### Persistent Mounts (fstab)

```bash
# /etc/fstab entries
server:/data  /mnt/data  nfs  defaults,_netdev  0 0
server:/home  /mnt/home  nfs4 rw,hard,noatime,vers=4.2  0 0

# With autofs-like behavior (mount on access)
server:/data  /mnt/data  nfs  noauto,x-systemd.automount,_netdev  0 0

# With timeout (unmount after idle)
server:/data  /mnt/data  nfs  noauto,x-systemd.automount,x-systemd.idle-timeout=300,_netdev  0 0
```

## Autofs (Automounting)

```bash
# Install
sudo apt install autofs                 # Debian/Ubuntu
sudo dnf install autofs                 # RHEL/CentOS

# /etc/auto.master — define mount points
/mnt/nfs   /etc/auto.nfs   --timeout=300

# /etc/auto.nfs — define shares
# key   options   location
data    -rw,sync  server:/data
home    -rw       server:/home
media   -ro       server:/media

# Wildcard (mount any export under /mnt/nfs/<export>)
*       -rw,sync  server:/&

# Restart autofs
sudo systemctl restart autofs

# Access triggers mount automatically
ls /mnt/nfs/data                       # mounts server:/data on first access
```

## Diagnostics and Troubleshooting

### Show Available Exports

```bash
# From client — query server for exports (NFSv3 only)
showmount -e server                    # show exports
showmount -a server                    # show mounted clients
showmount -d server                    # show directories being mounted

# NFSv4 — showmount may not work, use mount directly
sudo mount -t nfs4 server:/ /mnt/test && ls /mnt/test
```

### NFS Status and Statistics

```bash
nfsstat                                # NFS statistics (client + server)
nfsstat -s                             # server stats only
nfsstat -c                             # client stats only
nfsstat -m                             # show mount options for mounted shares

mountstats                             # detailed per-mount statistics
cat /proc/mounts | grep nfs            # current NFS mounts
cat /proc/net/rpc/nfs                  # raw RPC stats

# Check NFS daemon status
rpcinfo -p server                      # list RPC services on server
rpcinfo -t server nfs                  # test NFS service
ss -tlnp | grep -E '(2049|111)'       # NFS port (2049), rpcbind (111)
```

### Firewall Rules

```bash
# NFS uses port 2049 (nfsd) and 111 (rpcbind)
sudo firewall-cmd --permanent --add-service=nfs
sudo firewall-cmd --permanent --add-service=mountd
sudo firewall-cmd --permanent --add-service=rpc-bind
sudo firewall-cmd --reload

# Or with iptables
sudo iptables -A INPUT -p tcp --dport 2049 -j ACCEPT
sudo iptables -A INPUT -p tcp --dport 111 -j ACCEPT
```

## NFS Version Comparison

```bash
# NFSv3 — stateless, rpcbind required, portmapper for auxiliary services
#   - UDP or TCP
#   - Separate mount protocol
#   - File locking via NLM (separate protocol)

# NFSv4 — stateful, single port (2049), no rpcbind needed
#   - TCP only
#   - Integrated mount, locking, ACLs
#   - Kerberos authentication built-in
#   - Delegations for client-side caching

# NFSv4.1 — sessions, pNFS (parallel NFS for clustered storage)
#   - Trunking (multiple network paths)
#   - Directory delegations

# NFSv4.2 — server-side copy, sparse files, labeled NFS (SELinux)
#   - clone/copy_file_range (server-side copy between files)
#   - SEEK (sparse file holes)
```

## Tips

- Always use `sync` on exports for data safety -- `async` is faster but risks corruption if the server crashes.
- `no_subtree_check` is recommended for most exports -- subtree checking causes stale file handle errors and hurts performance.
- `no_root_squash` is a security risk -- only use it when clients genuinely need root access (e.g., diskless boot).
- NFSv4 uses only port 2049 -- much simpler for firewalls than NFSv3 which needs rpcbind (111) plus dynamically assigned ports.
- Use `hard` mounts in production -- `soft` mounts can cause silent data corruption when the server is temporarily unreachable.
- `noatime` on NFS mounts eliminates a write RPC for every read, significantly improving read-heavy workload performance.
- Kerberos (`sec=krb5p`) adds encryption at the NFS level -- use it instead of relying on network-level trust.
- `showmount -e` only works with NFSv3 -- for NFSv4, mount the root and list directories instead.
- Use systemd automount (`x-systemd.automount`) instead of autofs for simpler configuration on modern systems.
- Match `rsize`/`wsize` to your network MTU -- jumbo frames (9000 MTU) can improve throughput by 20-30%.
- Test NFS performance with `dd if=/dev/zero of=/mnt/nfs/test bs=1M count=1024` to measure write throughput.
- Always verify UID/GID mapping between client and server -- NFS trusts UIDs, so mismatched IDs cause permission chaos.

## See Also

ext4, xfs, tmpfs, fstab, iptables, ssh, kerberos

## References

- [man nfs(5)](https://man7.org/linux/man-pages/man5/nfs.5.html) -- NFS mount options and behavior
- [man exports(5)](https://man7.org/linux/man-pages/man5/exports.5.html) -- NFS export configuration
- [man exportfs(8)](https://man7.org/linux/man-pages/man8/exportfs.8.html) -- maintain NFS export table
- [man showmount(8)](https://man7.org/linux/man-pages/man8/showmount.8.html) -- show NFS server exports
- [man nfsstat(8)](https://man7.org/linux/man-pages/man8/nfsstat.8.html) -- NFS statistics
- [man auto.master(5)](https://man7.org/linux/man-pages/man5/auto.master.5.html) -- autofs master map
- [RFC 7530 - NFSv4](https://datatracker.ietf.org/doc/html/rfc7530) -- NFSv4.0 protocol specification
- [RFC 8881 - NFSv4.1](https://datatracker.ietf.org/doc/html/rfc8881) -- NFSv4.1 with sessions and pNFS
- [RFC 7862 - NFSv4.2](https://datatracker.ietf.org/doc/html/rfc7862) -- NFSv4.2 extensions
- [Linux NFS Wiki](http://wiki.linux-nfs.org/) -- kernel NFS implementation details
