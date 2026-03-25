# AppArmor (Application Armor)

Path-based mandatory access control for confining programs on Debian/Ubuntu systems.

## Status

```bash
sudo aa-status                           # all profiles and their modes
sudo aa-enabled                          # is AppArmor enabled? prints "Yes"/"No"
sudo systemctl status apparmor
```

## Profile Modes

### Enforce Mode (Block + Log Violations)

```bash
sudo aa-enforce /etc/apparmor.d/usr.sbin.nginx
sudo aa-enforce /usr/sbin/nginx          # can also pass the binary path
```

### Complain Mode (Log Only, No Blocking)

```bash
sudo aa-complain /etc/apparmor.d/usr.sbin.nginx
sudo aa-complain /usr/sbin/nginx
```

### Disable a Profile

```bash
sudo aa-disable /etc/apparmor.d/usr.sbin.nginx
# Creates a symlink in /etc/apparmor.d/disable/
```

### Re-enable a Profile

```bash
# Remove the disable symlink and reload
sudo rm /etc/apparmor.d/disable/usr.sbin.nginx
sudo apparmor_parser -r /etc/apparmor.d/usr.sbin.nginx
```

## Loading and Unloading Profiles

```bash
# Load/reload a profile
sudo apparmor_parser -r /etc/apparmor.d/usr.sbin.nginx

# Load a new profile
sudo apparmor_parser -a /etc/apparmor.d/usr.local.myapp

# Remove a profile from the kernel
sudo apparmor_parser -R /etc/apparmor.d/usr.sbin.nginx

# Reload all profiles
sudo systemctl reload apparmor
```

## Generating Profiles

### Interactive Profile Generator

```bash
# Start the program, then run aa-genprof in another terminal
sudo aa-genprof /usr/local/bin/myapp

# aa-genprof will:
# 1. Set the profile to complain mode
# 2. Ask you to exercise the application
# 3. Scan logs for access patterns
# 4. Prompt you to allow/deny each access
# 5. Save and enforce the profile
```

### Update Profile from Logs

```bash
# After running in complain mode, refine the profile
sudo aa-logprof
# Scans /var/log/syslog for AppArmor events and prompts for decisions
```

## Profile Syntax

### Basic Profile Structure

```bash
# /etc/apparmor.d/usr.local.myapp
#include <tunables/global>

/usr/local/bin/myapp {
  #include <abstractions/base>
  #include <abstractions/nameservice>

  # Read access
  /etc/myapp/** r,
  /etc/ssl/certs/** r,

  # Read-write access
  /var/lib/myapp/** rw,
  /var/log/myapp/** w,
  owner /tmp/myapp-* rw,

  # Execute
  /usr/bin/python3 ix,                   # inherit profile
  /usr/bin/curl Px,                      # transition to curl's profile

  # Network
  network inet stream,                   # TCP
  network inet dgram,                    # UDP

  # Capabilities
  capability net_bind_service,
  capability dac_override,

  # Deny explicitly
  deny /etc/shadow r,
  deny /root/** rwx,
}
```

### Permission Flags

```bash
# r  - read
# w  - write
# a  - append
# x  - execute
# m  - memory map executable
# k  - lock
# l  - link
# ix - inherit (execute under current profile)
# Px - transition to target's profile
# Cx - transition to child profile
# Ux - unconfined (escape AppArmor -- avoid)
```

## Abstractions (Reusable Rule Sets)

```bash
# Common abstractions (in /etc/apparmor.d/abstractions/)
#include <abstractions/base>             # basic system access
#include <abstractions/nameservice>      # DNS, NSS, LDAP
#include <abstractions/openssl>          # SSL/TLS libraries
#include <abstractions/python>           # Python runtime
#include <abstractions/apache2-common>   # Apache shared rules

# List available abstractions
ls /etc/apparmor.d/abstractions/
```

## Debugging

### View AppArmor Log Events

```bash
# Denied access events
sudo journalctl -k | grep apparmor
sudo dmesg | grep apparmor

# Detailed log parsing
sudo grep "apparmor=" /var/log/syslog | tail -20

# Audit messages (DENIED and ALLOWED in complain mode)
sudo grep "apparmor=\"DENIED\"" /var/log/syslog
sudo grep "apparmor=\"ALLOWED\"" /var/log/syslog
```

### Test a Profile Without Enforcing

```bash
# 1. Set to complain mode
sudo aa-complain /etc/apparmor.d/usr.local.myapp

# 2. Run the application and exercise all features

# 3. Review what would have been denied
sudo grep "apparmor=\"ALLOWED\"" /var/log/syslog

# 4. Update the profile with aa-logprof
sudo aa-logprof

# 5. Switch to enforce
sudo aa-enforce /etc/apparmor.d/usr.local.myapp
```

## Utilities Package

```bash
# Install all AppArmor tools
sudo apt install apparmor-utils

# Tools included:
# aa-status      - show loaded profiles
# aa-enforce     - set profile to enforce mode
# aa-complain    - set profile to complain mode
# aa-disable     - disable a profile
# aa-genprof     - generate a new profile interactively
# aa-logprof     - update profiles from logs
# aa-unconfined  - list running processes without profiles
```

### Find Unconfined Processes

```bash
sudo aa-unconfined                       # processes listening on network without profiles
sudo aa-unconfined --paranoid            # all running processes without profiles
```

## Tips

- Start with `aa-complain` and use `aa-logprof` iteratively -- writing profiles from scratch is error-prone
- `aa-genprof` is the fastest way to create a new profile; it watches logs while you exercise the app
- Profile filenames must match the binary path with dots replacing slashes (e.g., `usr.sbin.nginx`)
- AppArmor profiles are cumulative: `#include` stacks, and the most specific path match wins
- Docker and LXD containers can have per-container AppArmor profiles via security options
- Unlike SELinux, AppArmor is path-based; renaming a file can change its access -- be aware of symlinks
- `Ux` (unconfined execute) defeats the purpose of AppArmor; prefer `Px` or `ix` for child processes
- On Ubuntu, AppArmor ships with profiles for common services (snap, cups, tcpdump) -- check `aa-status` before writing your own

## References

- [AppArmor Documentation](https://gitlab.com/apparmor/apparmor/-/wikis/Documentation)
- [AppArmor Core Policy Reference](https://gitlab.com/apparmor/apparmor/-/wikis/Policy_Layout)
- [apparmor(7) Man Page](https://man7.org/linux/man-pages/man7/apparmor.7.html)
- [apparmor_parser(8) Man Page](https://man7.org/linux/man-pages/man8/apparmor_parser.8.html)
- [aa-genprof(8) Man Page](https://man7.org/linux/man-pages/man8/aa-genprof.8.html)
- [Ubuntu — AppArmor](https://ubuntu.com/server/docs/apparmor)
- [SUSE — Confining Privileges with AppArmor](https://documentation.suse.com/sles/15-SP5/html/SLES-all/part-apparmor.html)
- [Arch Wiki — AppArmor](https://wiki.archlinux.org/title/AppArmor)
- [Debian Wiki — AppArmor](https://wiki.debian.org/AppArmor)
