# Inotify (Filesystem Event Monitoring)

Inotify provides a Linux kernel mechanism for monitoring filesystem events such as file creation, modification, deletion, and moves, enabling applications to react to changes in real time without polling.

## inotifywait (Command-Line Monitoring)

```bash
# Watch a directory for any changes
inotifywait -m /var/log/

# Watch recursively
inotifywait -mr /project/src/

# Watch for specific events
inotifywait -m -e modify,create,delete /var/log/

# Watch and output in CSV format
inotifywait -mr --format '%T %w %f %e' --timefmt '%Y-%m-%d %H:%M:%S' /project/

# Watch a single file
inotifywait -m -e modify /etc/nginx/nginx.conf

# Wait for a single event (non-monitor mode)
inotifywait -e create /tmp/
# Blocks until a file is created in /tmp, then exits

# Watch and trigger rebuild
inotifywait -mr -e modify,create,delete --include '\.go$' /project/src/ |
  while read path action file; do
    echo "Change detected: $action $path$file"
    make build
  done

# Exclude patterns
inotifywait -mr --exclude '(\.git|node_modules|__pycache__)' /project/

# Multiple directories
inotifywait -m /var/log/ /etc/nginx/ /home/app/
```

## inotifywatch (Event Statistics)

```bash
# Collect event statistics for 60 seconds
inotifywatch -r -t 60 /project/src/

# Output:
# total  modify  create  delete  filename
# 150    120     20      10      /project/src/
# 80     75      3       2       /project/src/pkg/
# 45     40      3       2       /project/src/cmd/

# Watch specific events
inotifywatch -r -t 120 -e modify,access /var/log/

# Useful for profiling which directories are most active
```

## Event Types

| Event | Constant | Description |
|-------|----------|-------------|
| `access` | IN_ACCESS | File was read |
| `modify` | IN_MODIFY | File was written |
| `attrib` | IN_ATTRIB | Metadata changed (perms, timestamps, xattrs) |
| `close_write` | IN_CLOSE_WRITE | File opened for writing was closed |
| `close_nowrite` | IN_CLOSE_NOWRITE | File opened read-only was closed |
| `open` | IN_OPEN | File was opened |
| `moved_from` | IN_MOVED_FROM | File moved out of watched directory |
| `moved_to` | IN_MOVED_TO | File moved into watched directory |
| `create` | IN_CREATE | File/directory created in watched directory |
| `delete` | IN_DELETE | File/directory deleted from watched directory |
| `delete_self` | IN_DELETE_SELF | Watched file/directory itself was deleted |
| `move_self` | IN_MOVE_SELF | Watched file/directory itself was moved |
| `unmount` | IN_UNMOUNT | Filesystem containing watched object was unmounted |

### Composite Events

```bash
# Watch all close events
inotifywait -m -e close /path/
# Combines: close_write + close_nowrite

# Watch all move events
inotifywait -m -e move /path/
# Combines: moved_from + moved_to

# Most common: watch for "file saved" events
inotifywait -m -e close_write /path/
# close_write fires AFTER the write is complete (safer than modify)
```

## System Limits and Configuration

```bash
# Maximum watches per user
cat /proc/sys/fs/inotify/max_user_watches
# 65536 (default on many distros, 8192 on older)

# Maximum inotify instances per user
cat /proc/sys/fs/inotify/max_user_instances
# 128

# Maximum queued events
cat /proc/sys/fs/inotify/max_queued_events
# 16384

# Increase watches (needed for large codebases)
echo 524288 > /proc/sys/fs/inotify/max_user_watches

# Persistent via sysctl
echo "fs.inotify.max_user_watches=524288" >> /etc/sysctl.d/99-inotify.conf
sysctl -p /etc/sysctl.d/99-inotify.conf

# Increase instances
echo "fs.inotify.max_user_instances=512" >> /etc/sysctl.d/99-inotify.conf

# Check current watch usage per process
for pid in /proc/[0-9]*/; do
  count=$(find "${pid}fdinfo/" -maxdepth 1 -exec grep -c inotify {} + 2>/dev/null | awk -F: '{sum+=$2} END{print sum}')
  [ "${count:-0}" -gt 0 ] && echo "$count $(cat ${pid}comm 2>/dev/null) $(basename $pid)"
done | sort -rn | head -10

# Alternative: count watches via fd
find /proc/*/fd -lname anon_inode:inotify 2>/dev/null |
  cut -d/ -f3 | sort -u |
  while read pid; do
    count=$(grep -c inotify /proc/$pid/fdinfo/* 2>/dev/null || echo 0)
    echo "$count $(cat /proc/$pid/comm 2>/dev/null) $pid"
  done | sort -rn | head -10
```

## Inotify in C

```c
#include <stdio.h>
#include <stdlib.h>
#include <sys/inotify.h>
#include <unistd.h>
#include <string.h>

#define EVENT_SIZE  (sizeof(struct inotify_event))
#define BUF_LEN    (1024 * (EVENT_SIZE + 16))

int main() {
    int fd = inotify_init1(IN_NONBLOCK);
    if (fd < 0) { perror("inotify_init1"); exit(1); }

    int wd = inotify_add_watch(fd, "/tmp",
        IN_CREATE | IN_DELETE | IN_MODIFY | IN_MOVED_FROM | IN_MOVED_TO);
    if (wd < 0) { perror("inotify_add_watch"); exit(1); }

    char buf[BUF_LEN];
    while (1) {
        int len = read(fd, buf, BUF_LEN);
        if (len < 0) { usleep(100000); continue; }

        int i = 0;
        while (i < len) {
            struct inotify_event *event = (struct inotify_event *)&buf[i];
            if (event->len) {
                if (event->mask & IN_CREATE)
                    printf("Created: %s\n", event->name);
                if (event->mask & IN_DELETE)
                    printf("Deleted: %s\n", event->name);
                if (event->mask & IN_MODIFY)
                    printf("Modified: %s\n", event->name);
            }
            i += EVENT_SIZE + event->len;
        }
    }

    inotify_rm_watch(fd, wd);
    close(fd);
    return 0;
}
```

## fsnotify in Go

```go
package main

import (
    "log"
    "github.com/fsnotify/fsnotify"
)

func main() {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }
    defer watcher.Close()

    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                if event.Has(fsnotify.Write) {
                    log.Println("Modified:", event.Name)
                }
                if event.Has(fsnotify.Create) {
                    log.Println("Created:", event.Name)
                }
                if event.Has(fsnotify.Remove) {
                    log.Println("Removed:", event.Name)
                }
                if event.Has(fsnotify.Rename) {
                    log.Println("Renamed:", event.Name)
                }
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                log.Println("Error:", err)
            }
        }
    }()

    // Watch a directory (non-recursive)
    err = watcher.Add("/tmp")
    if err != nil {
        log.Fatal(err)
    }

    // Watch another path
    err = watcher.Add("/var/log")
    if err != nil {
        log.Fatal(err)
    }

    // Block forever
    select {}
}
```

## Inotify in Python

```python
import inotify.adapters
import inotify.constants

# Watch a directory
notifier = inotify.adapters.Inotify()
notifier.add_watch('/tmp',
    mask=inotify.constants.IN_CREATE |
         inotify.constants.IN_DELETE |
         inotify.constants.IN_MODIFY)

for event in notifier.event_gen(yield_nones=False):
    (_, type_names, path, filename) = event
    print(f"Event: {type_names} Path: {path}/{filename}")

# Recursive watch
notifier = inotify.adapters.InotifyTree('/project/src')
for event in notifier.event_gen(yield_nones=False):
    (_, type_names, path, filename) = event
    print(f"{type_names}: {path}/{filename}")
```

## Common Patterns

### Auto-Reload Configuration

```bash
# Reload nginx on config change
inotifywait -m -e close_write /etc/nginx/nginx.conf |
  while read path action file; do
    nginx -t && systemctl reload nginx
    echo "$(date): Reloaded nginx"
  done
```

### Build-on-Save

```bash
# Auto-build Go project on file change (with debounce)
inotifywait -mr -e close_write --include '\.go$' /project/ |
  while read path action file; do
    # Simple debounce: sleep and drain pending events
    sleep 0.5
    while read -t 0.1 _ _ _ 2>/dev/null; do :; done
    echo "Building..."
    cd /project && go build ./...
  done
```

### Log Rotation Detection

```bash
# Detect log rotation and reopen
inotifywait -m -e create,moved_to /var/log/ --include 'app\.log$' |
  while read path action file; do
    echo "Log rotated, notifying app"
    kill -USR1 $(cat /var/run/app.pid)
  done
```

## Inotify vs Fanotify

```bash
# fanotify (Linux 2.6.37+): filesystem-wide monitoring
# Advantages over inotify:
# - Can monitor entire mount points (no per-directory watches)
# - Supports permission events (allow/deny access)
# - More efficient for large directory trees
# - Used by antivirus and audit systems

# Key differences:
# inotify:  per-directory watches, user-space, no permission events
# fanotify: per-mount watches, requires CAP_SYS_ADMIN, permission events

# fanotify is NOT available from command line
# Must use C API: fanotify_init() + fanotify_mark()
```

## Troubleshooting

```bash
# "No space left on device" when adding watches
# Not a disk space issue -- inotify watch limit reached
cat /proc/sys/fs/inotify/max_user_watches
echo 524288 > /proc/sys/fs/inotify/max_user_watches

# Events not firing for network filesystems
# inotify does NOT work on: NFS, CIFS/SMB, FUSE, sshfs
# Use polling-based solutions instead (entr, watchman with polling)

# Missing events (queue overflow)
# IN_Q_OVERFLOW event indicates the queue is full
# Increase: echo 65536 > /proc/sys/fs/inotify/max_queued_events
# Or consume events faster

# Events on symlinks
# inotify watches the REAL file, not the symlink
# Changes via symlink will report the real path

# Recursive watching is NOT native
# inotify_add_watch works per-directory only
# Tools like inotifywait -r add watches on all subdirectories
# New subdirectories need new watches (handle IN_CREATE + IN_ISDIR)
```

## Tips

- Use `close_write` instead of `modify` for triggering builds; `modify` fires for every write() call, while `close_write` fires once when the file is closed
- Always use `--exclude` patterns for `.git`, `node_modules`, and build output directories to avoid watch exhaustion
- Set `max_user_watches=524288` on development machines; IDEs like VS Code and IntelliJ consume thousands of watches
- Inotify does not work on NFS, CIFS, or FUSE filesystems; use polling-based alternatives for network mounts
- Implement debouncing in watch scripts to avoid triggering builds multiple times for a single save operation
- Recursive watching requires adding new watches for newly created subdirectories manually in the C API
- The `IN_Q_OVERFLOW` event means you are losing events; increase `max_queued_events` or process faster
- Each inotify watch consumes approximately 1 KB of kernel memory (non-swappable)
- Use `fsnotify` in Go for cross-platform filesystem watching; it uses inotify on Linux and kqueue on macOS
- Fanotify is more efficient than inotify for monitoring entire mount points but requires root privileges
- In containers, inotify watches count against the host's limits, not the container's
- Combine inotify with SIGUSR1 to implement live-reload patterns in daemon processes

## See Also

proc-sys, signals, xargs

## References

- [inotify(7) Man Page](https://man7.org/linux/man-pages/man7/inotify.7.html)
- [inotify_add_watch(2)](https://man7.org/linux/man-pages/man2/inotify_add_watch.2.html)
- [inotify-tools GitHub](https://github.com/inotify-tools/inotify-tools)
- [fsnotify Go Package](https://pkg.go.dev/github.com/fsnotify/fsnotify)
- [fanotify(7) Man Page](https://man7.org/linux/man-pages/man7/fanotify.7.html)
- [Linux Kernel Inotify Documentation](https://docs.kernel.org/filesystems/inotify.html)
