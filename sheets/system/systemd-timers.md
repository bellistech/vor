# systemd-timers (scheduled tasks with systemd)

Systemd timers replace cron with better logging, dependency management, and resource control.

## Creating a Timer

### Service Unit (the task)

```bash
# /etc/systemd/system/backup.service
cat <<'EOF'
[Unit]
Description=Daily backup job

[Service]
Type=oneshot
ExecStart=/usr/local/bin/backup.sh
User=deploy
StandardOutput=journal
StandardError=journal
EOF
```

### Timer Unit (the schedule)

```bash
# /etc/systemd/system/backup.timer
cat <<'EOF'
[Unit]
Description=Run backup daily at 2am

[Timer]
OnCalendar=*-*-* 02:00:00
Persistent=true

[Install]
WantedBy=timers.target
EOF
```

### Enable the Timer

```bash
systemctl daemon-reload
systemctl enable --now backup.timer
```

## OnCalendar Syntax

### Realtime (Calendar) Schedules

```bash
# Daily at midnight
OnCalendar=daily

# Daily at 3:30am
OnCalendar=*-*-* 03:30:00

# Every Monday at 9am
OnCalendar=Mon *-*-* 09:00:00

# Weekdays at 6pm
OnCalendar=Mon..Fri *-*-* 18:00:00

# First of every month at noon
OnCalendar=*-*-01 12:00:00

# Every 15 minutes
OnCalendar=*:0/15

# Every 4 hours
OnCalendar=*-*-* 0/4:00:00

# Twice a day at 8am and 8pm
OnCalendar=*-*-* 08,20:00:00

# Quarterly (Jan, Apr, Jul, Oct 1st)
OnCalendar=*-01,04,07,10-01 00:00:00

# Test calendar expressions
systemd-analyze calendar "Mon..Fri *-*-* 09:00:00"
systemd-analyze calendar --iterations=5 "daily"
```

## Monotonic Timers

### Relative Schedules

```bash
# 15 minutes after boot
OnBootSec=15min

# 1 hour after the unit was last activated
OnUnitActiveSec=1h

# 5 minutes after the timer was started
OnActiveSec=5min

# Combine: first run 1min after boot, then every 30min
OnBootSec=1min
OnUnitActiveSec=30min
```

## Persistent Timers

### Catch Up After Downtime

```bash
# If the machine was off when the timer should have fired,
# run the job immediately on next boot
[Timer]
OnCalendar=daily
Persistent=true
```

## Randomized Delay

### Spread Load

```bash
# Add up to 1 hour random delay
[Timer]
OnCalendar=daily
RandomizedDelaySec=1h
```

## Managing Timers

### List, Start, Stop

```bash
# List all timers with next/last fire times
systemctl list-timers

# All timers including inactive
systemctl list-timers --all

# Start timer manually (fires the associated service)
systemctl start backup.timer

# Stop timer
systemctl stop backup.timer

# Disable timer
systemctl disable backup.timer

# Run the service immediately (without waiting for timer)
systemctl start backup.service

# Check timer status
systemctl status backup.timer

# Check service logs
journalctl -u backup.service
```

## Tips

- The timer unit name must match the service unit name (e.g., `backup.timer` triggers `backup.service`) unless you specify `Unit=` in the `[Timer]` section.
- `Persistent=true` is critical for daily/weekly tasks -- without it, if the machine is off at the scheduled time, the job is skipped entirely.
- Use `systemd-analyze calendar` to validate OnCalendar expressions before deploying.
- `Type=oneshot` in the service unit is required for tasks that run and exit (scripts, backups, etc.).
- Timer accuracy is 1 minute by default; set `AccuracySec=1s` if you need second-level precision.
- Advantages over cron: output goes to the journal automatically, resource limits via `MemoryMax`/`CPUQuota`, dependency ordering, and no mail surprises.

## See Also

- systemd, cron, at, journalctl

## References

- [man systemd.timer(5)](https://man7.org/linux/man-pages/man5/systemd.timer.5.html)
- [man systemd.time(7) — Calendar Events](https://man7.org/linux/man-pages/man7/systemd.time.7.html)
- [man systemd.service(5)](https://man7.org/linux/man-pages/man5/systemd.service.5.html)
- [man systemctl(1) — list-timers](https://man7.org/linux/man-pages/man1/systemctl.1.html)
- [systemd.timer Documentation](https://www.freedesktop.org/software/systemd/man/latest/systemd.timer.html)
- [systemd Calendar Events Syntax](https://www.freedesktop.org/software/systemd/man/latest/systemd.time.html#Calendar%20Events)
- [Arch Wiki — Systemd/Timers](https://wiki.archlinux.org/title/Systemd/Timers)
- [Red Hat — Using systemd Timer Units](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-system-services-with-systemctl_configuring-basic-system-settings)
- [Ubuntu — systemd Timers](https://manpages.ubuntu.com/manpages/noble/man5/systemd.timer.5.html)
