# logrotate (Log Rotation)

Manages log file rotation, compression, and removal to prevent unbounded disk usage.

## Configuration

### Global config (/etc/logrotate.conf)

```bash
# Default settings
weekly
rotate 4
create
dateext
compress
include /etc/logrotate.d
```

### Per-application config (/etc/logrotate.d/myapp)

```bash
/var/log/myapp/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    create 0640 myapp myapp
    sharedscripts
    postrotate
        systemctl reload myapp 2>/dev/null || true
    endscript
}
```

## Frequency

```bash
# daily           rotate every day
# weekly          rotate every week
# monthly         rotate every month
# yearly          rotate every year
```

## Size-Based Rotation

```bash
/var/log/myapp.log {
    size 100M               # rotate when file exceeds 100MB
    rotate 5
    compress
    missingok
    notifempty
}
```

### Combined size and frequency

```bash
/var/log/myapp.log {
    daily
    maxsize 50M              # rotate if >50M even if not daily yet
    minsize 1M               # skip rotation if <1M even on schedule
    rotate 7
    compress
}
```

## Compress

```bash
/var/log/myapp.log {
    daily
    rotate 30
    compress                 # gzip old logs
    delaycompress            # keep most recent rotated file uncompressed
    compresscmd /usr/bin/xz  # use xz instead of gzip
    compressext .xz
    compressoptions "-9"     # max compression
}
```

## Postrotate & Prerotate

### Postrotate (run after rotation)

```bash
/var/log/nginx/*.log {
    daily
    rotate 14
    compress
    delaycompress
    sharedscripts
    postrotate
        if [ -f /var/run/nginx.pid ]; then
            kill -USR1 $(cat /var/run/nginx.pid)
        fi
    endscript
}
```

### Prerotate (run before rotation)

```bash
/var/log/myapp.log {
    daily
    rotate 7
    prerotate
        echo "Starting rotation at $(date)" >> /var/log/rotation.log
    endscript
    postrotate
        systemctl reload myapp
    endscript
}
```

### Firstaction / Lastaction

```bash
/var/log/myapp/*.log {
    daily
    rotate 7
    sharedscripts
    firstaction
        # runs once before any log is rotated
        echo "Rotation batch starting"
    endscript
    lastaction
        # runs once after all logs are rotated
        echo "Rotation batch complete"
    endscript
}
```

## Copytruncate

### For apps that hold the file open

```bash
/var/log/myapp.log {
    daily
    rotate 7
    compress
    copytruncate            # copy file, then truncate original
    # No postrotate needed — app keeps writing to same file
}
```

Use `copytruncate` when the application cannot be signaled to reopen its log file. There is a small window where log lines can be lost.

## Create

### Set permissions on new log file

```bash
/var/log/myapp.log {
    daily
    rotate 7
    create 0640 myapp adm   # mode owner group
}
```

### Do not create new file

```bash
/var/log/myapp.log {
    daily
    rotate 7
    nocreate                 # app creates its own log file
}
```

## Sharedscripts

### Run scripts once for all matched files

```bash
/var/log/nginx/*.log {
    daily
    rotate 14
    sharedscripts            # postrotate runs once, not per file
    postrotate
        kill -USR1 $(cat /var/run/nginx.pid 2>/dev/null) 2>/dev/null || true
    endscript
}
```

Without `sharedscripts`, `postrotate` runs once per matched file, which can be wasteful.

## Common Directives

```bash
# missingok        no error if log file is missing
# notifempty       do not rotate empty files
# ifempty          rotate even if empty (default)
# dateext          use date instead of number for rotated files
# dateformat -%Y%m%d  customize date format (with dateext)
# extension .log   keep extension on rotated files
# olddir /var/log/archive  move rotated files to another directory
# maxage 90        remove rotated files older than 90 days
# mail admin@example.com  mail deleted log to address
# mailfirst        mail the just-rotated file (not the one about to be removed)
# su myapp myapp   run rotation as specific user/group
# tabooext + .dpkg-old .dpkg-new  skip files with these extensions
```

## Testing & Debugging

### Dry run

```bash
logrotate -d /etc/logrotate.d/myapp        # debug mode, no changes
```

### Force rotation

```bash
logrotate -f /etc/logrotate.d/myapp        # force rotate now
logrotate -f /etc/logrotate.conf           # force all
```

### Verbose run

```bash
logrotate -v /etc/logrotate.conf           # show what happens
```

### Manual execution (same as cron runs)

```bash
logrotate /etc/logrotate.conf
```

### Check state file

```bash
cat /var/lib/logrotate/status               # last rotation times
```

## Common Configurations

### Nginx

```bash
/var/log/nginx/*.log {
    daily
    rotate 14
    compress
    delaycompress
    missingok
    notifempty
    sharedscripts
    postrotate
        [ -f /var/run/nginx.pid ] && kill -USR1 $(cat /var/run/nginx.pid)
    endscript
}
```

### Syslog

```bash
/var/log/syslog /var/log/messages {
    weekly
    rotate 4
    compress
    delaycompress
    missingok
    notifempty
    postrotate
        /usr/lib/rsyslog/rsyslog-rotate
    endscript
}
```

### Docker container logs

```bash
/var/lib/docker/containers/*/*.log {
    daily
    rotate 7
    compress
    size 50M
    missingok
    copytruncate
}
```

## Tips

- `logrotate -d` (debug/dry-run) is essential before deploying new configs. It shows exactly what would happen.
- `copytruncate` avoids the need for a postrotate signal but can lose a few log lines during the copy-truncate gap.
- `delaycompress` keeps the most recent rotated file uncompressed, which is useful when postrotate scripts or monitoring tools need to read it.
- `sharedscripts` ensures postrotate runs only once when the glob matches multiple files. Without it, nginx would be signaled once per log file.
- The state file (`/var/lib/logrotate/status`) tracks when each file was last rotated. Delete an entry to force re-rotation.
- `su myapp myapp` is required when logrotate runs as root but the log directory is owned by a non-root user.
- `maxage 90` automatically removes rotated files older than 90 days, acting as a cleanup mechanism.
- logrotate runs via cron (usually `/etc/cron.daily/logrotate`). Check that cron is running if rotation stops happening.

## See Also

- rsyslog, journalctl, cron, systemd-timers

## References

- [man logrotate(8)](https://man7.org/linux/man-pages/man8/logrotate.8.html)
- [man logrotate.conf(5)](https://man7.org/linux/man-pages/man5/logrotate.conf.5.html)
- [logrotate GitHub Repository](https://github.com/logrotate/logrotate)
- [logrotate README](https://github.com/logrotate/logrotate/blob/main/README.md)
- [Arch Wiki — Logrotate](https://wiki.archlinux.org/title/Logrotate)
- [Red Hat — Managing Log Files with logrotate](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/assembly_troubleshooting-problems-using-log-files_configuring-basic-system-settings)
- [Ubuntu — Logrotate](https://manpages.ubuntu.com/manpages/noble/man8/logrotate.8.html)
- [Ubuntu Server Guide — Log Rotation](https://help.ubuntu.com/community/LinuxLogFiles)
