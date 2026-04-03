# The Mathematics of mount — Filesystem Attachment Internals

> *mount attaches filesystems to the VFS tree. The math covers bind mount overhead, mount namespace isolation, propagation types, and the VFS lookup cost model.*

---

## 1. VFS Layer — The Indirection Model

### The Model

Linux's Virtual Filesystem Switch (VFS) provides a uniform interface to all filesystem types. Each mount point adds a **vfsmount** structure to the mount tree.

### Lookup Cost

Path resolution traverses the dentry cache. For a path `/a/b/c/d/file`:

$$\text{Lookups} = \text{Path Components} = 5$$

Each component is a hash table lookup in the dentry cache:

$$T_{lookup} = O(1) \text{ per component (dentry cache hit)}$$

$$T_{total} = n \times O(1) = O(n)$$

Where $n$ = number of path components.

### Mount Point Crossing

When a path crosses a mount point, VFS redirects to the mounted filesystem's root:

$$T_{mount\_cross} = T_{lookup} + T_{mount\_table\_check}$$

The mount table check is $O(\log m)$ where $m$ = number of mounts (red-black tree since Linux 4.x).

### Worked Example

| Path Depth | Mounts Crossed | Lookups | Mount Checks |
|:---:|:---:|:---:|:---:|
| /file | 0 | 2 | 1 |
| /home/user/data | 1 (/home) | 4 | 2 |
| /var/lib/docker/overlay2/... | 3 | 8 | 4 |

---

## 2. Bind Mounts — Zero-Copy Mount Aliasing

### The Model

Bind mounts create a second view of an existing directory tree without copying data.

### Space Cost

$$\text{Extra Disk Space} = 0 \quad (\text{same inodes, same blocks})$$

$$\text{Extra Kernel Memory} = \text{sizeof(struct mount)} \approx 256 \text{ bytes per bind mount}$$

### Visibility Rules

$$\text{Bind Mount View} = \text{Source Subtree at Mount Time}$$

| Mount Type | Command | Behavior |
|:---|:---|:---|
| Bind | `mount --bind src dst` | Mirror of src at dst |
| Recursive bind | `mount --rbind src dst` | Includes sub-mounts |
| Read-only bind | `mount -o bind,ro src dst` | Read-only view |
| Remount read-only | `mount -o remount,bind,ro dst` | Make existing bind ro |

### Bind Mount Count Impact

Each mount adds entries to `/proc/mounts` and the mount table:

$$\text{Mount Table Memory} = m \times 256 \text{ bytes} + m \times \text{avg path length}$$

| Mounts | Kernel Memory | /proc/mounts Parse Time |
|:---:|:---:|:---:|
| 100 | ~50 KiB | <1 ms |
| 1,000 | ~500 KiB | ~5 ms |
| 10,000 | ~5 MiB | ~50 ms |
| 100,000 | ~50 MiB | ~500 ms |

**Container environments can accumulate thousands of mounts** (each container gets ~15-30 mounts).

---

## 3. Mount Propagation — Namespace Math

### Propagation Types

| Type | Flag | Behavior |
|:---|:---|:---|
| shared | `MS_SHARED` | Events propagate in both directions |
| private | `MS_PRIVATE` | No propagation (default) |
| slave | `MS_SLAVE` | Receive events, don't send |
| unbindable | `MS_UNBINDABLE` | Cannot be bind mounted |

### Propagation Graph

For $n$ mount namespaces with shared propagation:

$$\text{Mount Events} = n - 1 \quad (\text{one event propagates to all peers})$$

$$\text{Total Mounts Created} = \text{Original} + (n - 1) \text{ propagated copies}$$

### Worked Example

*"Docker with 50 containers, each sharing /dev mounts."*

$$\text{With shared propagation: } 50 \text{ mount events per new /dev mount}$$

$$\text{With slave propagation: } 1 \text{ event received, 0 sent per container}$$

$$\text{With private: } 0 \text{ events}$$

---

## 4. Overlay Filesystems — Container Storage Math

### The Model

OverlayFS layers a writable upper dir over a read-only lower dir. Used extensively in Docker.

### Layer Composition

$$\text{Visible File} = \begin{cases} \text{Upper layer version} & \text{if exists in upper} \\ \text{Lower layer version} & \text{if only in lower} \\ \text{whiteout} & \text{if deleted (opaque marker)} \end{cases}$$

### Storage Efficiency

$$\text{Total Storage} = \text{Base Image (shared)} + \sum_{i=1}^{n} \text{Container Layer}_i$$

$$\text{Savings vs Full Copy} = 1 - \frac{\text{Base} + \sum \text{Layers}}{n \times (\text{Base} + \text{Avg Layer})}$$

### Worked Example

*"100 containers from a 500 MiB base image, each with 50 MiB writable layer."*

$$\text{Overlay total} = 500 + 100 \times 50 = 5,500 \text{ MiB} = 5.4 \text{ GiB}$$

$$\text{Full copy total} = 100 \times (500 + 50) = 55,000 \text{ MiB} = 53.7 \text{ GiB}$$

$$\text{Savings} = 1 - \frac{5,500}{55,000} = 90\%$$

| Containers | Base Image | Layer Size | Overlay Total | Full Copy | Savings |
|:---:|:---:|:---:|:---:|:---:|:---:|
| 10 | 500 MiB | 50 MiB | 1,000 MiB | 5,500 MiB | 82% |
| 50 | 500 MiB | 50 MiB | 3,000 MiB | 27,500 MiB | 89% |
| 100 | 500 MiB | 50 MiB | 5,500 MiB | 55,000 MiB | 90% |
| 500 | 1 GiB | 100 MiB | 49 GiB | 537 GiB | 91% |

---

## 5. Loop Devices — File-Backed Block Devices

### The Model

Loop mounts back a block device with a regular file. Each loop device has an I/O translation layer.

### Performance Overhead

$$\text{Loop I/O Path} = \text{VFS} \rightarrow \text{Loop Driver} \rightarrow \text{VFS} \rightarrow \text{Backing FS} \rightarrow \text{Block Layer}$$

$$\text{Extra Latency} \approx 5-15\% \text{ (one additional VFS traversal)}$$

### Loop Device Limits

$$\text{Default max\_loop} = 256 \text{ (kernel parameter)}$$

$$\text{Configurable up to} = 2^{20} = 1,048,576$$

---

## 6. Mount Flags — Performance Impact Matrix

### Read-Write Flags

| Flag | Effect | Performance Impact |
|:---|:---|:---|
| `ro` | Read-only | Eliminates all write overhead |
| `rw` | Read-write (default) | Normal |
| `sync` | Synchronous I/O | 10-100x slower writes |
| `async` | Asynchronous (default) | Normal |
| `noatime` | No access time updates | Eliminates read-caused writes |
| `nodiratime` | No dir access time | Reduces metadata writes |
| `nosuid` | Ignore setuid bits | Security, no perf impact |
| `noexec` | Prevent execution | Security, no perf impact |
| `nodev` | Ignore device files | Security, no perf impact |

### Security vs Performance Matrix

| Mount Point | Recommended Flags | Rationale |
|:---|:---|:---|
| / | `defaults` | Need full access for system |
| /home | `nosuid,nodev` | Users shouldn't have suid/devices |
| /tmp | `nosuid,nodev,noexec` | Ephemeral, no execution needed |
| /var/log | `nosuid,nodev,noexec,noatime` | Append-only workload |
| /data | `noatime,nosuid,nodev` | Performance + security |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $O(n)$ path components | Linear | VFS lookup |
| $O(\log m)$ mount check | Logarithmic | Mount table lookup |
| $m \times 256$ bytes | Linear scaling | Mount table memory |
| $\text{Base} + n \times \text{Layer}$ | Linear | Overlay storage |
| $1 - \frac{\text{Overlay}}{\text{Full Copy}}$ | Ratio | Storage savings |

---

*Every `mount`, `umount`, and `/proc/mounts` read interacts with the VFS mount tree — a kernel data structure that defines how your entire storage hierarchy is assembled into a single directory tree.*
