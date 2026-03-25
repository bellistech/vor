# at (one-time scheduled commands)

Schedule a command to run once at a specific time.

## Schedule a Job

### Interactive Input

```bash
# Schedule for a specific time
at 14:30
# Type commands, then Ctrl+D to submit
# at> /usr/local/bin/deploy.sh
# at>

# Schedule for a specific date and time
at 14:30 2024-03-15

# Schedule relative to now
at now + 5 minutes
at now + 2 hours
at now + 1 day
at now + 3 weeks
```

### From a File or Pipe

```bash
# Read commands from a file
at 02:00 -f /usr/local/bin/backup.sh

# Pipe commands
echo "/usr/local/bin/cleanup.sh" | at midnight

# Multiple commands via heredoc
at now + 30 minutes <<'EOF'
/usr/local/bin/stop_service.sh
/usr/local/bin/rotate_logs.sh
/usr/local/bin/start_service.sh
EOF
```

## Time Formats

### Supported Expressions

```bash
at midnight         # 00:00
at noon             # 12:00
at teatime          # 16:00
at tomorrow         # same time tomorrow
at 9am tomorrow
at 2:30pm Friday
at now + 1 hour
at now + 30 minutes
at 10:00 Jul 31
at 10:00 2024-12-25
```

## List Queued Jobs

### atq

```bash
# List pending jobs
atq

# Same as
at -l
```

## Remove a Job

### atrm

```bash
# Remove job number 5
atrm 5

# Same as
at -r 5

# Remove multiple
atrm 3 5 7
```

## View a Queued Job

### Show Job Contents

```bash
at -c 5
```

## Batch Jobs

### Run When Load is Low

```bash
# Run when system load drops below 1.5 (default threshold)
batch <<'EOF'
/usr/local/bin/heavy_report.sh
EOF

# batch is equivalent to at with load-based scheduling
```

## Access Control

### Allow and Deny

```bash
# /etc/at.allow — only listed users can use at
# /etc/at.deny  — listed users cannot use at

# Same precedence rules as cron:
# If at.allow exists, only those users are allowed
# If only at.deny exists, everyone except listed is allowed
```

## Tips

- `at` jobs inherit the current environment (unlike cron), including `$PATH`, `$HOME`, and current directory.
- Jobs run with `/bin/sh` by default. Start the script with `#!/bin/bash` or set `SHELL` if you need bash features.
- `at` requires the `atd` daemon to be running: `systemctl start atd`.
- `batch` is useful for deferring heavy tasks until the machine is idle -- it checks load average before running.
- The output of at jobs is mailed to the user. Redirect to a file or `/dev/null` to avoid mail.
- `at -c <job>` shows the full environment and commands that will run -- useful for debugging.
