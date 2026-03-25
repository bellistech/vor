# journalctl (systemd journal viewer)

Query and display logs from the systemd journal.

## Following Logs

### Live Tail

```bash
# Follow all logs (like tail -f)
journalctl -f

# Follow a specific unit
journalctl -f -u nginx

# Follow multiple units
journalctl -f -u nginx -u php-fpm
```

## Filtering by Unit

### Service Logs

```bash
journalctl -u nginx
journalctl -u ssh.service

# Last 100 lines of a unit
journalctl -u nginx -n 100

# No pager — dump straight to stdout
journalctl -u nginx --no-pager
```

## Time Filters

### Since and Until

```bash
journalctl --since "2024-01-15 09:00:00"
journalctl --since "1 hour ago"
journalctl --since yesterday
journalctl --since today

journalctl --since "2024-01-15" --until "2024-01-16"
journalctl --since "09:00" --until "10:00"
```

## Priority Levels

### Filter by Severity

```bash
# 0=emerg, 1=alert, 2=crit, 3=err, 4=warning, 5=notice, 6=info, 7=debug
journalctl -p err
journalctl -p warning

# Range: warning and above
journalctl -p 0..4

# Errors from a specific service
journalctl -p err -u nginx
```

## Boot Logs

### Current and Previous Boots

```bash
# Current boot
journalctl -b

# Previous boot
journalctl -b -1

# Two boots ago
journalctl -b -2

# List all recorded boots
journalctl --list-boots
```

## Kernel Messages

### Kernel Ring Buffer

```bash
# Kernel messages only (like dmesg)
journalctl -k

# Kernel messages from current boot
journalctl -k -b
```

## Output Formats

### JSON and Others

```bash
journalctl -u nginx -o json
journalctl -u nginx -o json-pretty

# Short with full timestamps
journalctl -o short-precise

# Verbose — all fields
journalctl -o verbose

# Export format for piping to another journal
journalctl -o export
```

## Disk Usage

### Check and Control Journal Size

```bash
# How much space the journal uses
journalctl --disk-usage

# Shrink journal to 500M
journalctl --vacuum-size=500M

# Remove entries older than 2 weeks
journalctl --vacuum-time=2weeks

# Remove old files until only 5 remain
journalctl --vacuum-files=5
```

## Persistent Storage

### Enable Persistent Journals

```bash
# Create the directory (journal becomes persistent automatically)
mkdir -p /var/log/journal
systemctl restart systemd-journald

# Or set in /etc/systemd/journald.conf:
# Storage=persistent
```

## Advanced Filters

### By PID, UID, Executable

```bash
journalctl _PID=1234
journalctl _UID=1000
journalctl _COMM=sshd

# By syslog identifier
journalctl -t myapp

# Grep-like pattern matching
journalctl -u nginx -g "error|timeout"
```

## Tips

- Journals are not persistent by default on many distros -- `/var/log/journal/` must exist or set `Storage=persistent` in `journald.conf`.
- `--no-pager` is essential for scripting; otherwise journalctl pipes through `less`.
- `-x` adds explanatory text from the message catalog (useful for systemd's own messages).
- `-r` reverses output (newest first) which pairs well with `-n`.
- The `-g` grep flag requires systemd 246+; on older systems pipe through `grep`.
- Journal vacuum only removes archived files -- it will not shrink the active journal file.
