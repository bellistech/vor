# sudo (superuser do)

Execute commands as another user, typically root.

## Basic Usage

### Run as Root

```bash
# Run a single command as root
sudo systemctl restart nginx

# Open a root shell
sudo -i

# Root shell preserving current environment
sudo -s

# Run as a different user
sudo -u deploy /opt/deploy/run.sh

# Run as a group
sudo -g docker docker ps

# Edit a file as root (safe editor)
sudo -e /etc/nginx/nginx.conf
# Same as
sudoedit /etc/nginx/nginx.conf
```

### Check Permissions

```bash
# List what the current user can run
sudo -l

# List for another user (requires root)
sudo -l -U deploy

# Validate/refresh sudo timestamp without running a command
sudo -v

# Invalidate cached credentials (force password next time)
sudo -k
```

## Editing sudoers

### visudo

```bash
# Always use visudo to edit sudoers (validates syntax)
visudo

# Edit a specific file
visudo -f /etc/sudoers.d/deploy

# Check syntax without editing
visudo -c
visudo -c -f /etc/sudoers.d/deploy
```

## sudoers Syntax

### User Specifications

```bash
# Format: WHO WHERE=(AS_WHOM) WHAT
# user  host=(runas) command

# Full root access
deploy ALL=(ALL:ALL) ALL

# Specific commands only
deploy ALL=(ALL) /usr/bin/systemctl restart nginx, /usr/bin/systemctl reload nginx

# No password required
deploy ALL=(ALL) NOPASSWD: ALL

# No password for specific commands
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx

# Run as specific user only
deploy ALL=(postgres) /usr/bin/pg_dump
```

### Group Rules

```bash
# Allow all members of a group (prefix with %)
%sudo ALL=(ALL:ALL) ALL
%docker ALL=(ALL) NOPASSWD: /usr/bin/docker
%developers ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart myapp
```

### Aliases

```bash
# User aliases
User_Alias ADMINS = alice, bob, carol

# Command aliases
Cmnd_Alias SERVICES = /usr/bin/systemctl start *, /usr/bin/systemctl stop *, /usr/bin/systemctl restart *
Cmnd_Alias LOGS = /usr/bin/journalctl, /usr/bin/tail /var/log/*

# Host aliases
Host_Alias WEBSERVERS = web1, web2, web3

# Combine
ADMINS WEBSERVERS=(ALL) SERVICES, LOGS
```

## Drop-in Files

### /etc/sudoers.d/

```bash
# Create modular sudoers rules
echo 'deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart nginx' > /etc/sudoers.d/deploy

# Set correct permissions (required)
chmod 0440 /etc/sudoers.d/deploy

# Validate
visudo -c -f /etc/sudoers.d/deploy
```

## Environment

### secure_path and env_keep

```bash
# Set PATH for sudo commands (in sudoers)
Defaults    secure_path="/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

# Preserve specific environment variables
Defaults    env_keep += "SSH_AUTH_SOCK"
Defaults    env_keep += "http_proxy https_proxy"

# Reset environment (default, more secure)
Defaults    env_reset

# Disable lecture message
Defaults    !lecture
```

## Logging

### Audit sudo Usage

```bash
# sudo logs to /var/log/auth.log (Debian) or /var/log/secure (RHEL)
grep sudo /var/log/auth.log

# Or via journal
journalctl -u sudo
journalctl _COMM=sudo
```

## Tips

- Always use `visudo` to edit sudoers files. A syntax error in sudoers can lock you out of sudo entirely.
- Files in `/etc/sudoers.d/` must have mode `0440` and must not contain `.` or `~` in the filename (they are silently ignored).
- `NOPASSWD` should be limited to specific commands, not blanket `ALL`, in production.
- `sudo -e` / `sudoedit` is safer than `sudo vim` because it copies the file, lets you edit as your user, then copies it back -- preventing editor shell escapes.
- `sudo -i` starts a login shell as root (reads root's `.profile`); `sudo -s` starts a non-login shell (keeps your environment).
- The `Defaults timestamp_timeout=N` directive sets how many minutes sudo caches credentials (default is usually 5-15).

## References

- [man sudo(8)](https://man7.org/linux/man-pages/man8/sudo.8.html)
- [man sudoers(5)](https://man7.org/linux/man-pages/man5/sudoers.5.html)
- [man visudo(8)](https://man7.org/linux/man-pages/man8/visudo.8.html)
- [man sudo.conf(5)](https://man7.org/linux/man-pages/man5/sudo.conf.5.html)
- [Sudo Project — Manual Pages](https://www.sudo.ws/docs/man/)
- [Sudo Project — sudoers Manual](https://www.sudo.ws/docs/man/sudoers.man/)
- [Arch Wiki — sudo](https://wiki.archlinux.org/title/Sudo)
- [Red Hat — Configuring sudo Access](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-sudo-access_configuring-basic-system-settings)
- [Ubuntu — Sudoers](https://help.ubuntu.com/community/Sudoers)
