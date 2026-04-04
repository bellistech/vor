# FUSE (Filesystem in Userspace)

Interface for implementing custom filesystems in userspace programs, without kernel module development.

## Architecture

```
+------------------+
| Application      |  open("/mnt/fuse/file")
+------------------+
        |  VFS
+------------------+
| FUSE kernel      |  /dev/fuse
| module           |
+------------------+
        |  fd read/write
+------------------+
| FUSE daemon      |  userspace filesystem process
| (your code)      |
+------------------+
        |
| Backend storage  |  local disk, network, cloud, etc.
+------------------+
```

## Install libfuse

```bash
# Debian/Ubuntu
apt install libfuse3-dev fuse3

# Fedora/RHEL
dnf install fuse3-devel fuse3

# macOS (macFUSE)
brew install macfuse

# Verify
fusermount3 -V
ls /dev/fuse
```

## High-Level vs Low-Level API

```
High-Level API (fuse_main / fuse_operations):
  - Path-based: operations receive file paths
  - Simpler to implement
  - Handles threading automatically
  - Best for: prototypes, simple filesystems

Low-Level API (fuse_session / fuse_lowlevel_ops):
  - Inode-based: operations receive inode numbers
  - Full control over replies and buffering
  - Better performance (avoids path lookups)
  - Best for: production filesystems, network backends
```

## Minimal FUSE Filesystem (High-Level, C)

```c
#define FUSE_USE_VERSION 31
#include <fuse3/fuse.h>
#include <string.h>
#include <errno.h>

static const char *hello_str = "Hello, FUSE!\n";

static int hello_getattr(const char *path, struct stat *stbuf,
                         struct fuse_file_info *fi) {
    memset(stbuf, 0, sizeof(struct stat));
    if (strcmp(path, "/") == 0) {
        stbuf->st_mode = S_IFDIR | 0755;
        stbuf->st_nlink = 2;
    } else if (strcmp(path, "/hello") == 0) {
        stbuf->st_mode = S_IFREG | 0444;
        stbuf->st_nlink = 1;
        stbuf->st_size = strlen(hello_str);
    } else {
        return -ENOENT;
    }
    return 0;
}

static int hello_readdir(const char *path, void *buf,
                         fuse_fill_dir_t filler,
                         off_t offset, struct fuse_file_info *fi,
                         enum fuse_readdir_flags flags) {
    filler(buf, ".", NULL, 0, 0);
    filler(buf, "..", NULL, 0, 0);
    filler(buf, "hello", NULL, 0, 0);
    return 0;
}

static int hello_read(const char *path, char *buf, size_t size,
                      off_t offset, struct fuse_file_info *fi) {
    size_t len = strlen(hello_str);
    if (offset >= len) return 0;
    if (offset + size > len) size = len - offset;
    memcpy(buf, hello_str + offset, size);
    return size;
}

static const struct fuse_operations hello_ops = {
    .getattr  = hello_getattr,
    .readdir  = hello_readdir,
    .read     = hello_read,
};

int main(int argc, char *argv[]) {
    return fuse_main(argc, argv, &hello_ops, NULL);
}
```

## Compile and Mount

```bash
# Compile
gcc -Wall hello.c $(pkg-config fuse3 --cflags --libs) -o hellofs

# Mount
mkdir -p /tmp/fuse
./hellofs /tmp/fuse

# Mount with debug output
./hellofs -d /tmp/fuse

# Mount in foreground
./hellofs -f /tmp/fuse

# Unmount
fusermount3 -u /tmp/fuse
# Or
umount /tmp/fuse
```

## Key FUSE Operations

```c
struct fuse_operations {
    // Metadata
    int (*getattr)(const char *, struct stat *, struct fuse_file_info *);
    int (*chmod)(const char *, mode_t, struct fuse_file_info *);
    int (*chown)(const char *, uid_t, gid_t, struct fuse_file_info *);
    int (*utimens)(const char *, const struct timespec[2], struct fuse_file_info *);

    // Directory
    int (*mkdir)(const char *, mode_t);
    int (*rmdir)(const char *);
    int (*readdir)(const char *, void *, fuse_fill_dir_t, off_t,
                   struct fuse_file_info *, enum fuse_readdir_flags);

    // File
    int (*create)(const char *, mode_t, struct fuse_file_info *);
    int (*open)(const char *, struct fuse_file_info *);
    int (*read)(const char *, char *, size_t, off_t, struct fuse_file_info *);
    int (*write)(const char *, const char *, size_t, off_t, struct fuse_file_info *);
    int (*truncate)(const char *, off_t, struct fuse_file_info *);
    int (*unlink)(const char *);
    int (*rename)(const char *, const char *, unsigned int);
    int (*release)(const char *, struct fuse_file_info *);

    // Symlinks / Links
    int (*symlink)(const char *, const char *);
    int (*readlink)(const char *, char *, size_t);
    int (*link)(const char *, const char *);

    // Extended attributes
    int (*setxattr)(const char *, const char *, const char *, size_t, int);
    int (*getxattr)(const char *, const char *, char *, size_t);
    int (*listxattr)(const char *, char *, size_t);

    // Filesystem
    int (*statfs)(const char *, struct statvfs *);
    int (*fsync)(const char *, int, struct fuse_file_info *);
};
```

## Popular FUSE Filesystems

```bash
# sshfs — mount remote directories over SSH
sshfs user@host:/remote/path /mnt/ssh
fusermount3 -u /mnt/ssh

# s3fs — mount S3 bucket as local filesystem
s3fs mybucket /mnt/s3 -o passwd_file=~/.passwd-s3fs
fusermount3 -u /mnt/s3

# rclone mount — mount any cloud storage (S3, GCS, Drive, etc.)
rclone mount remote:bucket /mnt/cloud --vfs-cache-mode full
fusermount3 -u /mnt/cloud

# GlusterFS — distributed filesystem (FUSE client)
mount -t glusterfs server:/volume /mnt/gluster

# NTFS-3G — read/write NTFS via FUSE
mount -t ntfs-3g /dev/sda1 /mnt/ntfs
```

## Performance Tuning

```bash
# Writeback cache (kernel 3.15+, FUSE 3.x)
# Reduces write syscalls by buffering in page cache
./myfs -o writeback_cache /mnt/fuse

# Splice (zero-copy between kernel and userspace)
# Enabled by default in libfuse3 for read operations

# Large read/write sizes
./myfs -o max_read=131072 /mnt/fuse

# Parallel directory operations
./myfs -o max_background=32 /mnt/fuse

# Kernel-level caching
./myfs -o kernel_cache /mnt/fuse       # Cache file data aggressively
./myfs -o auto_cache /mnt/fuse         # Invalidate cache on mtime change
./myfs -o entry_timeout=60 /mnt/fuse   # Cache dir entries for 60s
./myfs -o attr_timeout=60 /mnt/fuse    # Cache attributes for 60s

# Multi-threading (default in high-level API)
# For low-level: use fuse_session_loop_mt()
```

## Mount Options

```bash
# Allow other users to access the mount
./myfs -o allow_other /mnt/fuse

# Allow root to access the mount
./myfs -o allow_root /mnt/fuse

# Set default permissions checking
./myfs -o default_permissions /mnt/fuse

# Set UID/GID for all files
./myfs -o uid=1000,gid=1000 /mnt/fuse

# Read-only mount
./myfs -o ro /mnt/fuse

# Must edit /etc/fuse.conf to enable allow_other globally
echo "user_allow_other" >> /etc/fuse.conf
```

## Python (pyfuse3)

```python
import pyfuse3
import errno, stat, os

class HelloFS(pyfuse3.Operations):
    async def getattr(self, inode, ctx=None):
        entry = pyfuse3.EntryAttributes()
        if inode == pyfuse3.ROOT_INODE:
            entry.st_mode = stat.S_IFDIR | 0o755
            entry.st_size = 0
        return entry

    async def lookup(self, parent_inode, name, ctx=None):
        if parent_inode != pyfuse3.ROOT_INODE or name != b'hello':
            raise pyfuse3.FUSEError(errno.ENOENT)
        return self.getattr(2)

    async def readdir(self, fh, start_id, token):
        pyfuse3.readdir_reply(token, b'hello', self.getattr(2), 1)

# Run: python3 hellofs.py /mnt/fuse
```

## Debugging

```bash
# Debug mode (foreground + verbose logging)
./myfs -d /tmp/fuse

# Trace FUSE operations
strace -f -e trace=read,write,ioctl ./myfs -f /tmp/fuse

# Check /dev/fuse activity
cat /sys/fs/fuse/connections/*/waiting
```

## Tips

- Always implement `getattr` first; everything depends on it
- Return negative errno values from operations (e.g., `-ENOENT`, `-EACCES`)
- Use `direct_io` to bypass page cache for backends with their own caching (databases, cloud)
- `writeback_cache` dramatically improves small-write performance but may cause data loss on crash
- Set reasonable `entry_timeout` and `attr_timeout` to reduce kernel-to-userspace round trips
- Use the low-level API for network-backed filesystems where inode-based lookup avoids repeated path parsing
- `allow_other` requires `user_allow_other` in `/etc/fuse.conf` to work
- Multi-threaded mode is default; use `-s` for single-threaded debugging
- FUSE adds ~10-20us latency per operation vs in-kernel filesystems; batch when possible
- Use `splice` (enabled by default in libfuse3) for zero-copy reads of large files
- Test with `fio` and `bonnie++` to benchmark your FUSE filesystem against expectations
- Handle `SIGTERM` gracefully by calling `fuse_session_exit()` for clean unmount

## See Also

- OverlayFS (union filesystem built into the kernel)
- NFS (Network File System for remote mounts)
- 9P/virtio-fs (guest-host filesystem sharing in VMs)
- io_uring (async I/O interface that can improve FUSE performance)
- CUSE (Character Device in Userspace, FUSE for char devices)

## References

- [libfuse GitHub Repository](https://github.com/libfuse/libfuse)
- [FUSE Kernel Documentation](https://www.kernel.org/doc/html/latest/filesystems/fuse.html)
- [sshfs Repository](https://github.com/libfuse/sshfs)
- [FUSE Protocol Description](https://john-googler.github.io/fuse-protocol/)
- [Writing a FUSE Filesystem (Julia Evans)](https://jvns.ca/blog/2023/10/09/fuse-filesystem/)
- [pyfuse3 Documentation](https://pyfuse3.readthedocs.io/)
