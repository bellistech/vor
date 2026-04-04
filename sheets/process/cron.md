# cron (scheduled tasks)

Schedule recurring commands using crontab.

## Crontab Management

### Edit and List

```bash
# Edit current user's crontab
crontab -e

# List current user's crontab
crontab -l

# Edit another user's crontab (root)
crontab -e -u deploy

# List another user's crontab
crontab -l -u deploy

# Remove entire crontab (dangerous)
crontab -r

# Remove with confirmation prompt
crontab -ri
```

## Crontab Syntax

### Field Format

```bash
# .---------------- minute (0-59)
# |  .------------- hour (0-23)
# |  |  .---------- day of month (1-31)
# |  |  |  .------- month (1-12 or jan-dec)
# |  |  |  |  .---- day of week (0-7, 0 and 7 are Sunday, or mon-sun)
# |  |  |  |  |
# *  *  *  *  *  command
```

### Common Patterns

```bash
# Every minute
* * * * * /usr/local/bin/check_health.sh

# Every 5 minutes
*/5 * * * * /usr/local/bin/metrics.sh

# Every hour at minute 0
0 * * * * /usr/local/bin/hourly.sh

# Daily at 2:30am
30 2 * * * /usr/local/bin/backup.sh

# Weekdays at 9am
0 9 * * 1-5 /usr/local/bin/report.sh

# Every Monday at 6am
0 6 * * 1 /usr/local/bin/weekly.sh

# First of every month at midnight
0 0 1 * * /usr/local/bin/monthly.sh

# Every 15 minutes during business hours
*/15 9-17 * * 1-5 /usr/local/bin/poll.sh

# Twice a day at 8am and 8pm
0 8,20 * * * /usr/local/bin/sync.sh

# Every quarter (Jan, Apr, Jul, Oct)
0 0 1 1,4,7,10 * /usr/local/bin/quarterly.sh
```

### Shorthand Schedules

```bash
@reboot    /usr/local/bin/startup.sh
@hourly    /usr/local/bin/hourly.sh      # 0 * * * *
@daily     /usr/local/bin/daily.sh       # 0 0 * * *
@weekly    /usr/local/bin/weekly.sh      # 0 0 * * 0
@monthly   /usr/local/bin/monthly.sh     # 0 0 1 * *
@yearly    /usr/local/bin/yearly.sh      # 0 0 1 1 *
```

## Environment

### Setting Variables

```bash
# Set environment variables at the top of crontab
SHELL=/bin/bash
PATH=/usr/local/bin:/usr/bin:/bin
MAILTO=admin@example.com
HOME=/home/deploy

# Disable email output
MAILTO=""

0 2 * * * /usr/local/bin/backup.sh
```

## System Cron

### /etc/cron.d and Directories

```bash
# System cron files (include username field)
# /etc/cron.d/myapp
*/5 * * * * deploy /opt/myapp/bin/healthcheck.sh

# Drop-in directories (scripts, no crontab syntax)
/etc/cron.hourly/
/etc/cron.daily/
/etc/cron.weekly/
/etc/cron.monthly/

# Scripts in these dirs must be executable and have no extension
chmod +x /etc/cron.daily/backup
```

## Output Handling

### Logging and Mail

```bash
# Redirect output to a log file
0 2 * * * /usr/local/bin/backup.sh >> /var/log/backup.log 2>&1

# Discard all output
0 * * * * /usr/local/bin/quiet.sh > /dev/null 2>&1

# Send only errors via mail
0 2 * * * /usr/local/bin/backup.sh > /dev/null
```

## Access Control

### Allow and Deny

```bash
# /etc/cron.allow — only listed users can use cron
# /etc/cron.deny  — listed users are denied

# If cron.allow exists, only users in it can use cron
# If only cron.deny exists, everyone except listed users can use cron
# If neither exists, behavior depends on the distro (often only root)
```

## Tips

- Cron jobs run with a minimal environment -- always use full paths for commands and set `PATH` explicitly.
- `MAILTO=""` prevents surprise email buildup in `/var/spool/mail/` from noisy jobs.
- `crontab -r` deletes your entire crontab with no confirmation -- always use `crontab -ri` or keep a backup.
- Cron has no concept of "missed" runs -- if the machine is off at the scheduled time, the job does not run. Use anacron or systemd timers for catch-up.
- `/etc/cron.d/` files require a username field between the schedule and the command (unlike user crontabs).
- Scripts in `/etc/cron.daily/` etc. must not have a `.sh` extension on some systems (run-parts ignores files with dots).
- Debug with `grep CRON /var/log/syslog` or `journalctl -u cron`.

## See Also

- at, systemd-timers, bash, nice, kill

## References

- [man crontab(5) — File Format](https://man7.org/linux/man-pages/man5/crontab.5.html)
- [man crontab(1) — User Command](https://man7.org/linux/man-pages/man1/crontab.1.html)
- [man cron(8)](https://man7.org/linux/man-pages/man8/cron.8.html)
- [man anacrontab(5)](https://man7.org/linux/man-pages/man5/anacrontab.5.html)
- [man run-parts(8)](https://man7.org/linux/man-pages/man8/run-parts.8.html)
- [Arch Wiki — Cron](https://wiki.archlinux.org/title/Cron)
- [Ubuntu — CronHowto](https://help.ubuntu.com/community/CronHowto)
- [Red Hat — Automating System Tasks with cron](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-system-services-with-systemctl_configuring-basic-system-settings)
- [crontab.guru — Cron Expression Editor](https://crontab.guru/)
