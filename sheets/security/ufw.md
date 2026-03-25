# UFW (Uncomplicated Firewall)

Simplified frontend for iptables/nftables on Debian/Ubuntu systems.

## Enable and Disable

```bash
sudo ufw enable                          # activate firewall (persists across reboots)
sudo ufw disable                         # deactivate firewall
sudo ufw reset                           # reset to factory defaults
```

## Status

```bash
sudo ufw status                          # simple view
sudo ufw status verbose                  # show default policies + rules
sudo ufw status numbered                 # rules with numbers (for deletion)
```

## Default Policies

```bash
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw default deny routed             # for forwarded traffic
```

## Allow Rules

### By Port

```bash
sudo ufw allow 22                        # TCP and UDP
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 60000:61000/udp           # port range (mosh)
```

### By Service Name

```bash
sudo ufw allow ssh
sudo ufw allow http
sudo ufw allow https
```

### From Specific Source

```bash
sudo ufw allow from 10.0.0.0/8
sudo ufw allow from 10.0.1.50 to any port 22
sudo ufw allow from 192.168.1.0/24 to any port 5432 proto tcp
```

### To Specific Interface

```bash
sudo ufw allow in on eth0 to any port 80
sudo ufw allow in on wg0                # allow all WireGuard traffic
```

## Deny and Reject Rules

```bash
# Deny (silently drop)
sudo ufw deny 23/tcp
sudo ufw deny from 203.0.113.50

# Reject (send ICMP unreachable)
sudo ufw reject 23/tcp

# Deny outgoing
sudo ufw deny out 25/tcp                # block outbound SMTP
```

## Delete Rules

```bash
# By rule number
sudo ufw status numbered
sudo ufw delete 3

# By rule specification
sudo ufw delete allow 80/tcp
sudo ufw delete allow from 10.0.1.50

# Delete and confirm non-interactively
sudo ufw --force delete 3
```

## Insert Rules (Order Matters)

```bash
# Insert at position 1 (top priority)
sudo ufw insert 1 deny from 203.0.113.0/24
sudo ufw insert 1 allow from 10.0.1.50 to any port 22
```

## Application Profiles

### List Available Profiles

```bash
sudo ufw app list
```

### Show Profile Details

```bash
sudo ufw app info "Nginx Full"
sudo ufw app info OpenSSH
```

### Allow by Profile

```bash
sudo ufw allow "Nginx Full"             # opens 80 and 443
sudo ufw allow "Nginx HTTP"             # opens 80 only
sudo ufw allow OpenSSH
```

### Create Custom Profile

```bash
# /etc/ufw/applications.d/myapp
[MyApp]
title=My Application
description=Custom web service
ports=8080,8443/tcp
```

```bash
sudo ufw app update MyApp
sudo ufw allow MyApp
```

## Logging

```bash
sudo ufw logging on
sudo ufw logging medium                 # off | low | medium | high | full

# View logs
sudo tail -f /var/log/ufw.log
sudo journalctl -u ufw
```

## Rate Limiting

```bash
# Limit connection attempts (6 connections per 30 seconds per IP)
sudo ufw limit 22/tcp
sudo ufw limit ssh
```

## IPv6

```bash
# Ensure IPv6 is enabled in /etc/default/ufw
# IPV6=yes

# Rules work identically
sudo ufw allow from 2001:db8::/32 to any port 443
```

## Routing and Forwarding

```bash
# /etc/default/ufw
DEFAULT_FORWARD_POLICY="ACCEPT"          # for VPN/NAT setups

# /etc/ufw/sysctl.conf
net/ipv4/ip_forward=1

# Allow forwarded traffic between interfaces
sudo ufw route allow in on wg0 out on eth0
sudo ufw route allow in on eth0 out on wg0
```

## Tips

- Always `allow ssh` (or your SSH port) BEFORE running `ufw enable` to avoid locking yourself out
- UFW rules are evaluated top-down; first match wins -- use `insert` for priority overrides
- `ufw limit` is a simple rate limiter; for advanced rate limiting, use fail2ban instead
- UFW wraps iptables by default; on newer Ubuntu (22.04+), it uses nftables as the backend
- Application profiles live in `/etc/ufw/applications.d/` and are a clean way to manage multi-port services
- Docker bypasses UFW by default (writes its own iptables rules); use `ufw-docker` or manage Docker's iptables separately
- `ufw reset` deletes all rules and disables the firewall -- useful for starting fresh
- Rules persist across reboots automatically when UFW is enabled

## References

- [Ubuntu UFW Documentation](https://help.ubuntu.com/community/UFW)
- [ufw(8) Man Page](https://man7.org/linux/man-pages/man8/ufw.8.html)
- [ufw-framework(8) Man Page](https://man7.org/linux/man-pages/man8/ufw-framework.8.html)
- [Ubuntu Server — Firewall (UFW)](https://ubuntu.com/server/docs/firewalls)
- [Arch Wiki — Uncomplicated Firewall](https://wiki.archlinux.org/title/Uncomplicated_Firewall)
- [Debian Wiki — Uncomplicated Firewall](https://wiki.debian.org/Uncomplicated%20Firewall%20%28ufw%29)
- [UFW GitHub Repository](https://launchpad.net/ufw)
