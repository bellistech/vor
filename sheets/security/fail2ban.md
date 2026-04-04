# Fail2ban (Intrusion Prevention)

Monitor log files and automatically ban IPs that show malicious behavior.

## Service Management

```bash
sudo systemctl start fail2ban
sudo systemctl enable fail2ban
sudo systemctl status fail2ban

# Check fail2ban version and backend
sudo fail2ban-client version
sudo fail2ban-client ping                # should respond "pong"
```

## Status and Monitoring

### Overall Status

```bash
sudo fail2ban-client status
```

### Jail-Specific Status

```bash
sudo fail2ban-client status sshd
# Shows: currently banned IPs, total banned, total failed
```

### Check Banned IPs Across All Jails

```bash
sudo fail2ban-client banned
```

## Ban and Unban

### Manually Ban an IP

```bash
sudo fail2ban-client set sshd banip 203.0.113.50
```

### Unban an IP

```bash
sudo fail2ban-client set sshd unbanip 203.0.113.50
```

### Unban From All Jails

```bash
sudo fail2ban-client unban 203.0.113.50
```

### Unban Everything

```bash
sudo fail2ban-client unban --all
```

## Configuration

### Main Config (/etc/fail2ban/jail.local)

```bash
# /etc/fail2ban/jail.local (overrides jail.conf -- never edit jail.conf directly)
[DEFAULT]
bantime  = 1h
findtime = 10m
maxretry = 5
banaction = iptables-multiport
ignoreip = 127.0.0.1/8 10.0.0.0/8       # never ban these

# Email notifications
destemail = admin@acme.com
sender = fail2ban@acme.com
action = %(action_mwl)s                  # ban + mail with whois + logs

[sshd]
enabled = true
port    = 2222                           # if using non-default SSH port
logpath = /var/log/auth.log
maxretry = 3
bantime = 24h
```

### Reload After Config Changes

```bash
sudo fail2ban-client reload
sudo fail2ban-client reload sshd         # reload single jail
```

## Custom Jails

### Nginx Auth Jail

```bash
# /etc/fail2ban/jail.local
[nginx-auth]
enabled  = true
port     = http,https
filter   = nginx-http-auth
logpath  = /var/log/nginx/error.log
maxretry = 3
bantime  = 1h
```

### WordPress Login Jail

```bash
# /etc/fail2ban/filter.d/wordpress-auth.conf
[Definition]
failregex = ^<HOST> -.* "POST /wp-login.php
ignoreregex =

# /etc/fail2ban/jail.local
[wordpress-auth]
enabled  = true
port     = http,https
filter   = wordpress-auth
logpath  = /var/log/nginx/access.log
maxretry = 5
bantime  = 30m
```

### Rate Limit Any HTTP Endpoint

```bash
# /etc/fail2ban/filter.d/nginx-ratelimit.conf
[Definition]
failregex = ^<HOST> -.* "POST /api/login
ignoreregex =

# /etc/fail2ban/jail.local
[nginx-ratelimit]
enabled  = true
port     = http,https
filter   = nginx-ratelimit
logpath  = /var/log/nginx/access.log
maxretry = 10
findtime = 1m
bantime  = 15m
```

## Filters

### Test a Filter Against a Log

```bash
# Dry run: see what a filter matches
sudo fail2ban-regex /var/log/auth.log /etc/fail2ban/filter.d/sshd.conf

# Test with a custom regex
sudo fail2ban-regex /var/log/nginx/access.log "^<HOST> -.* \"POST /wp-login.php"
```

### Filter Regex Syntax

```bash
# /etc/fail2ban/filter.d/myfilter.conf
[Definition]
# <HOST> is replaced with the IP-matching regex
failregex = ^<HOST> -.* 401
            ^Authentication failure from <HOST>
ignoreregex = ^<HOST> -.* admin_healthcheck
```

## Actions

### List Available Actions

```bash
ls /etc/fail2ban/action.d/
```

### Common Actions

```bash
# iptables (default)
banaction = iptables-multiport

# nftables
banaction = nftables-multiport

# UFW integration
banaction = ufw

# Cloudflare API ban
banaction = cloudflare
actionban = curl -s -X POST "https://api.cloudflare.com/client/v4/..." \
  -d '{"mode":"block","configuration":{"target":"ip","value":"<ip>"}}'
```

## Incremental Banning

```bash
# /etc/fail2ban/jail.local
[DEFAULT]
bantime.increment = true
bantime.factor = 2                       # double ban time each offense
bantime.maxtime = 4w                     # cap at 4 weeks
bantime.overalljails = true              # count across all jails
```

## Logging and Debugging

```bash
# Fail2ban's own log
sudo tail -f /var/log/fail2ban.log

# Increase log verbosity
sudo fail2ban-client set loglevel DEBUG
sudo fail2ban-client get loglevel

# Check database of bans
sudo fail2ban-client get dbpurgeage
```

## Tips

- Always use `jail.local` for overrides; `jail.conf` gets overwritten on package upgrades
- `ignoreip` should include your own IP and management subnets to avoid locking yourself out
- Use `fail2ban-regex` to test filters before deploying -- saves debugging time
- `bantime.increment` is essential for persistent attackers; exponential backoff is very effective
- The SQLite database at `/var/lib/fail2ban/fail2ban.sqlite3` persists bans across restarts
- On systems with both iptables and nftables, make sure `banaction` matches your active firewall
- `findtime` and `bantime` accept suffixes: `s` (seconds), `m` (minutes), `h` (hours), `d` (days), `w` (weeks)
- Aggressive `maxretry=1` on SSH is fine if you only use key auth -- password typos won't be an issue

## See Also

- ufw, firewalld, iptables, nftables, ssh

## References

- [Fail2ban Documentation](https://www.fail2ban.org/wiki/index.php/MANUAL_0_8)
- [Fail2ban Official Wiki](https://www.fail2ban.org/wiki/index.php/Main_Page)
- [Fail2ban GitHub Repository](https://github.com/fail2ban/fail2ban)
- [fail2ban-client(1) Man Page](https://man7.org/linux/man-pages/man1/fail2ban-client.1.html)
- [fail2ban-jail.conf(5) Man Page](https://man7.org/linux/man-pages/man5/jail.conf.5.html)
- [Arch Wiki — Fail2ban](https://wiki.archlinux.org/title/Fail2ban)
- [Ubuntu — Fail2ban Setup](https://help.ubuntu.com/community/Fail2ban)
- [Red Hat — Using Fail2ban to Secure the Server](https://access.redhat.com/solutions/2115553)
