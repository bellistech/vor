# rsyslog (System Logging)

High-performance syslog daemon with filtering, templates, and log forwarding.

## Configuration

### Main config (/etc/rsyslog.conf)

```bash
# Global settings
$MaxMessageSize 64k
$WorkDirectory /var/spool/rsyslog

# Load modules
module(load="imuxsock")     # local system logging
module(load="imklog")       # kernel logging
module(load="imudp")        # UDP syslog reception
input(type="imudp" port="514")
module(load="imtcp")        # TCP syslog reception
input(type="imtcp" port="514")

# Include drop-in configs
$IncludeConfig /etc/rsyslog.d/*.conf
```

## Facilities

### Standard syslog facilities

```bash
# kern       kernel messages
# user       user-level messages
# mail       mail system
# daemon     system daemons
# auth       security/authorization
# syslog     syslog internal
# lpr        printing
# news       network news
# cron       clock/cron daemon
# local0-7   locally defined (custom apps)
```

## Priorities (Severity)

### From highest to lowest

```bash
# emerg      system is unusable
# alert      action must be taken immediately
# crit       critical conditions
# err        error conditions
# warning    warning conditions
# notice     normal but significant
# info       informational
# debug      debug-level messages
```

## Filtering Rules

### Facility.priority based

```bash
# Log all kernel messages to /var/log/kern.log
kern.*                          /var/log/kern.log

# Log auth messages at info or higher
auth,authpriv.*                 /var/log/auth.log

# Log mail messages
mail.*                          /var/log/mail.log

# Log everything except mail and auth
*.*;auth,authpriv.none;mail.none   /var/log/syslog

# Log emergencies to all logged-in users
*.emerg                         :omusrmsg:*

# Log cron
cron.*                          /var/log/cron.log

# Log local apps
local0.*                        /var/log/myapp.log
local1.err                      /var/log/myapp-errors.log
```

### Property-based filters

```bash
# Match by program name
:programname, isequal, "nginx"     /var/log/nginx/syslog.log
:programname, startswith, "docker" /var/log/docker.log

# Match by message content
:msg, contains, "error"            /var/log/errors.log
:msg, regex, "failed.*authentication" /var/log/auth-failures.log

# Discard messages (stop processing)
:programname, isequal, "chatty-app"  stop
```

### Expression-based filters (RainerScript)

```bash
# if $programname == 'myapp' and $syslogseverity <= 4 then {
#     action(type="omfile" file="/var/log/myapp-important.log")
# }
#
# if $msg contains 'SQL' then {
#     action(type="omfile" file="/var/log/sql.log")
#     stop
# }
```

## Templates

### Custom log format

```bash
# template(name="CustomFormat" type="string"
#     string="%TIMESTAMP% %HOSTNAME% %syslogtag%%msg:::sp-if-no-1st-sp%%msg:::drop-last-lf%\n"
# )
# local0.*  /var/log/myapp.log;CustomFormat
```

### JSON template

```bash
# template(name="JsonFormat" type="list") {
#     constant(value="{")
#     constant(value="\"timestamp\":\"")  property(name="timereported" dateFormat="rfc3339")
#     constant(value="\",\"host\":\"")    property(name="hostname")
#     constant(value="\",\"program\":\"") property(name="programname")
#     constant(value="\",\"severity\":\"") property(name="syslogseverity-text")
#     constant(value="\",\"message\":\"") property(name="msg" format="jsonf")
#     constant(value="\"}\n")
# }
```

### Dynamic file naming

```bash
# template(name="PerHostLog" type="string"
#     string="/var/log/remote/%HOSTNAME%/%PROGRAMNAME%.log"
# )
# *.* ?PerHostLog
```

## Forwarding

### Forward to remote syslog server

```bash
# UDP forwarding (single @)
*.* @logserver.example.com:514

# TCP forwarding (double @@)
*.* @@logserver.example.com:514

# Forward only errors
*.err @@logserver.example.com:514
```

### Forward with queue (reliable)

```bash
# action(type="omfwd"
#     target="logserver.example.com"
#     port="514"
#     protocol="tcp"
#     queue.type="LinkedList"
#     queue.size="10000"
#     queue.filename="fwd_queue"
#     queue.saveonshutdown="on"
#     action.resumeRetryCount="-1"
# )
```

### Forward specific app

```bash
# if $programname == 'myapp' then {
#     action(type="omfwd" target="logserver.example.com" port="514" protocol="tcp")
# }
```

## Log Rotation Integration

### Write to file with rsyslog

```bash
# local0.*  /var/log/myapp.log
# # Then configure logrotate for /var/log/myapp.log
```

### Reopen log files after rotation

```bash
# In logrotate config:
# postrotate
#     /usr/lib/rsyslog/rsyslog-rotate
# endscript
# Or:
# postrotate
#     systemctl kill -s HUP rsyslog
# endscript
```

## Operations

### Test configuration

```bash
rsyslogd -N1                              # check config syntax
rsyslogd -N1 -f /etc/rsyslog.conf
```

### Restart

```bash
systemctl restart rsyslog
systemctl status rsyslog
```

### Send test message

```bash
logger "Test message from command line"
logger -p local0.info "App log message"
logger -t myapp "Tagged message"
logger -p auth.warning "Auth test"
```

### Check rsyslog internal stats

```bash
# module(load="impstats" interval="60" format="json" log.file="/var/log/rsyslog-stats.log")
```

## Tips

- `@@` is TCP, `@` is UDP. Use TCP for reliable delivery, especially over unreliable networks.
- Disk-assisted queues (`queue.type="LinkedList"` with `queue.filename`) prevent log loss during network outages.
- `stop` (or the legacy `~`) discards messages. Place discard rules before catch-all rules.
- Use `local0` through `local7` for custom applications to avoid conflicting with system facilities.
- `logger -p facility.priority` sends test messages. Essential for verifying routing rules.
- Templates with `%HOSTNAME%` in the filename create per-host log directories for central log servers.
- `rsyslogd -N1` validates config but does not catch runtime issues. Watch `/var/log/syslog` after restart.
- Rate limiting is on by default (`$SystemLogRateLimitInterval`). Disable for high-volume apps with `$SystemLogRateLimitInterval 0`.

## References

- [man rsyslogd(8)](https://man7.org/linux/man-pages/man8/rsyslogd.8.html)
- [man rsyslog.conf(5)](https://man7.org/linux/man-pages/man5/rsyslog.conf.5.html)
- [man syslog(3)](https://man7.org/linux/man-pages/man3/syslog.3.html)
- [rsyslog Official Documentation](https://www.rsyslog.com/doc/master/)
- [rsyslog Configuration — RainerScript](https://www.rsyslog.com/doc/master/rainerscript/index.html)
- [rsyslog Modules Reference](https://www.rsyslog.com/doc/master/configuration/modules/index.html)
- [rsyslog GitHub Repository](https://github.com/rsyslog/rsyslog)
- [Arch Wiki — rsyslog](https://wiki.archlinux.org/title/Rsyslog)
- [Red Hat — Configuring rsyslog](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/assembly_troubleshooting-problems-using-log-files_configuring-basic-system-settings)
- [Ubuntu — rsyslog](https://help.ubuntu.com/community/Rsyslog)
