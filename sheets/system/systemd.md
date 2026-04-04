# systemd (service and system manager)

Manages system services, targets, and the boot process on modern Linux.

## Service Management

### Start, Stop, Restart

```bash
systemctl start nginx
systemctl stop nginx
systemctl restart nginx

# Reload config without full restart (e.g. nginx -s reload equivalent)
systemctl reload nginx

# Restart only if already running
systemctl try-restart nginx
```

### Enable and Disable at Boot

```bash
systemctl enable nginx
systemctl disable nginx

# Enable AND start in one command
systemctl enable --now nginx

# Disable AND stop in one command
systemctl disable --now nginx
```

### Check Status

```bash
systemctl status nginx
systemctl is-active nginx
systemctl is-enabled nginx
systemctl is-failed nginx
```

### Mask and Unmask

```bash
# Prevent a service from being started at all (even manually)
systemctl mask bluetooth
systemctl unmask bluetooth
```

## Unit Files

### List Units

```bash
# All loaded units
systemctl list-units

# Only services
systemctl list-units --type=service

# All units including inactive
systemctl list-units --all

# Failed units only
systemctl list-units --failed

# List unit files and their enable state
systemctl list-unit-files --type=service
```

### View and Edit Unit Files

```bash
# Show full unit file
systemctl cat nginx.service

# Show where a unit file lives
systemctl show -p FragmentPath nginx.service

# Create an override without modifying the vendor file
systemctl edit nginx.service

# Full edit (replace, not override)
systemctl edit --full nginx.service
```

### Reload After Changes

```bash
# After editing any unit file
systemctl daemon-reload
```

### Custom Unit File

```bash
# /etc/systemd/system/myapp.service
cat <<'EOF'
[Unit]
Description=My Application
After=network.target
Wants=postgresql.service

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/myapp
ExecStart=/opt/myapp/bin/server
Restart=on-failure
RestartSec=5
Environment=NODE_ENV=production

[Install]
WantedBy=multi-user.target
EOF
```

## Timers

### List Active Timers

```bash
systemctl list-timers
systemctl list-timers --all
```

## Targets

### Check and Change Targets

```bash
# Current default target (like runlevel)
systemctl get-default

# Set default to multi-user (no GUI)
systemctl set-default multi-user.target

# Set default to graphical
systemctl set-default graphical.target

# Switch immediately (like init 3)
systemctl isolate multi-user.target
```

## System Power

```bash
systemctl reboot
systemctl poweroff
systemctl suspend
systemctl hibernate
```

## Dependencies and Order

```bash
# Show what a unit depends on
systemctl list-dependencies nginx.service

# Reverse — what depends on this unit
systemctl list-dependencies --reverse nginx.service
```

## Tips

- `daemon-reload` is required after any unit file change -- forgetting it is the most common mistake.
- `mask` is stronger than `disable` -- it symlinks the unit to /dev/null so nothing can start it.
- `edit` creates a drop-in override at `/etc/systemd/system/<unit>.d/override.conf` which survives package upgrades.
- Use `Type=notify` for services that signal readiness via sd_notify (e.g. PostgreSQL).
- `Restart=on-failure` only restarts on non-zero exit; use `Restart=always` for services that should never stay down.
- `systemctl --user` manages per-user services (e.g. `systemctl --user start syncthing`).

## See Also

- journalctl, systemd-timers, kernel, ps, cron, dmesg

## References

- [man systemctl(1)](https://man7.org/linux/man-pages/man1/systemctl.1.html)
- [man systemd(1)](https://man7.org/linux/man-pages/man1/systemd.1.html)
- [man systemd.unit(5)](https://man7.org/linux/man-pages/man5/systemd.unit.5.html)
- [man systemd.service(5)](https://man7.org/linux/man-pages/man5/systemd.service.5.html)
- [man systemd.exec(5)](https://man7.org/linux/man-pages/man5/systemd.exec.5.html)
- [systemd Documentation Index](https://www.freedesktop.org/software/systemd/man/latest/)
- [systemd Unit File Reference](https://www.freedesktop.org/software/systemd/man/latest/systemd.unit.html)
- [Arch Wiki — systemd](https://wiki.archlinux.org/title/Systemd)
- [Arch Wiki — systemd FAQ](https://wiki.archlinux.org/title/Systemd/FAQ)
- [Red Hat — Managing System Services](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-system-services-with-systemctl_configuring-basic-system-settings)
- [Ubuntu — systemd](https://manpages.ubuntu.com/manpages/noble/man1/systemd.1.html)
