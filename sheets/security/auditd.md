# Linux Audit Framework (auditd rules, searching, and reporting for system auditing)

## Core Components

### Service Management

```bash
# Check auditd status
sudo systemctl status auditd

# Start/stop/restart auditd
sudo systemctl start auditd
sudo systemctl enable auditd

# Reload rules without restarting
sudo augenrules --load

# Check current audit status
sudo auditctl -s

# List all active rules
sudo auditctl -l

# Delete all rules (runtime only, does not affect persistent rules)
sudo auditctl -D
```

## Audit Rules with auditctl

### File Watch Rules

```bash
# Watch a file for reads, writes, attribute changes, executes
# -w <path> -p <permissions> -k <key>
# Permissions: r=read, w=write, x=execute, a=attribute change

# Monitor /etc/passwd for any changes
sudo auditctl -w /etc/passwd -p wa -k identity_file

# Monitor /etc/shadow for reads and writes
sudo auditctl -w /etc/shadow -p rwa -k shadow_access

# Watch SSH configuration
sudo auditctl -w /etc/ssh/sshd_config -p wa -k sshd_config

# Watch sudoers files
sudo auditctl -w /etc/sudoers -p wa -k sudoers_change
sudo auditctl -w /etc/sudoers.d/ -p wa -k sudoers_change

# Watch cron directories
sudo auditctl -w /etc/cron.d/ -p wa -k cron_change
sudo auditctl -w /etc/crontab -p wa -k cron_change
sudo auditctl -w /var/spool/cron/ -p wa -k cron_change

# Watch kernel modules
sudo auditctl -w /sbin/insmod -p x -k kernel_modules
sudo auditctl -w /sbin/rmmod -p x -k kernel_modules
sudo auditctl -w /sbin/modprobe -p x -k kernel_modules
```

### Syscall Monitoring Rules

```bash
# Format: -a <action>,<filter> -S <syscall> -F <field=value> -k <key>
# Actions: always,exit | never,exit
# Common filters: task, exit, user, exclude

# Monitor file deletions by all users
sudo auditctl -a always,exit -F arch=b64 -S unlink -S unlinkat -S rename -S renameat -k file_deletion

# Monitor privilege escalation (setuid/setgid)
sudo auditctl -a always,exit -F arch=b64 -S execve -F euid=0 -F auid>=1000 -F auid!=4294967295 -k privilege_escalation

# Monitor mount operations
sudo auditctl -a always,exit -F arch=b64 -S mount -S umount2 -k mount_ops

# Monitor time changes
sudo auditctl -a always,exit -F arch=b64 -S adjtimex -S settimeofday -S clock_settime -k time_change

# Monitor network connections (socket, connect, accept)
sudo auditctl -a always,exit -F arch=b64 -S socket -S connect -S accept -k network_connections

# Monitor failed access attempts
sudo auditctl -a always,exit -F arch=b64 -S open -S openat -F exit=-EACCES -k access_denied
sudo auditctl -a always,exit -F arch=b64 -S open -S openat -F exit=-EPERM -k access_denied

# Monitor process execution
sudo auditctl -a always,exit -F arch=b64 -S execve -k process_execution
```

### User and Group Monitoring

```bash
# Monitor user/group management commands
sudo auditctl -w /usr/sbin/useradd -p x -k user_management
sudo auditctl -w /usr/sbin/userdel -p x -k user_management
sudo auditctl -w /usr/sbin/usermod -p x -k user_management
sudo auditctl -w /usr/sbin/groupadd -p x -k group_management
sudo auditctl -w /usr/sbin/groupdel -p x -k group_management
sudo auditctl -w /usr/sbin/groupmod -p x -k group_management

# Monitor login-related files
sudo auditctl -w /var/log/lastlog -p wa -k login_tracking
sudo auditctl -w /var/run/faillock/ -p wa -k login_tracking
```

## Persistent Rules

### Rules Directory

```bash
# Persistent rules location
# /etc/audit/rules.d/*.rules
# Files are processed in alphabetical order

# Recommended naming convention:
# 10-base-config.rules      — buffer size, failure mode
# 30-cis-compliance.rules   — CIS benchmark rules
# 50-custom.rules           — site-specific rules
# 99-finalize.rules         — lock configuration

# Example: /etc/audit/rules.d/10-base-config.rules
cat <<'EOF' | sudo tee /etc/audit/rules.d/10-base-config.rules
# Increase buffer size for busy systems
-b 8192

# Set failure mode (0=silent, 1=printk, 2=panic)
-f 1

# Rate limit messages per second (0=no limit)
-r 100
EOF

# Example: /etc/audit/rules.d/99-finalize.rules
cat <<'EOF' | sudo tee /etc/audit/rules.d/99-finalize.rules
# Make rules immutable (requires reboot to change)
-e 2
EOF

# Merge and load all rules
sudo augenrules --load

# Check for rule syntax errors
sudo augenrules --check
```

### CIS Compliance Rules

```bash
# /etc/audit/rules.d/30-cis-compliance.rules
cat <<'EOF' | sudo tee /etc/audit/rules.d/30-cis-compliance.rules
# CIS 4.1.4 - Events that modify date and time
-a always,exit -F arch=b64 -S adjtimex -S settimeofday -k time-change
-a always,exit -F arch=b64 -S clock_settime -k time-change
-w /etc/localtime -p wa -k time-change

# CIS 4.1.5 - Events that modify user/group information
-w /etc/group -p wa -k identity
-w /etc/passwd -p wa -k identity
-w /etc/gshadow -p wa -k identity
-w /etc/shadow -p wa -k identity
-w /etc/security/opasswd -p wa -k identity

# CIS 4.1.6 - Events that modify network environment
-a always,exit -F arch=b64 -S sethostname -S setdomainname -k system-locale
-w /etc/issue -p wa -k system-locale
-w /etc/issue.net -p wa -k system-locale
-w /etc/hosts -p wa -k system-locale
-w /etc/hostname -p wa -k system-locale
-w /etc/sysconfig/network -p wa -k system-locale

# CIS 4.1.7 - Events that modify MAC policy
-w /etc/selinux/ -p wa -k MAC-policy
-w /etc/apparmor/ -p wa -k MAC-policy
-w /etc/apparmor.d/ -p wa -k MAC-policy

# CIS 4.1.8 - Login and logout events
-w /var/log/faillog -p wa -k logins
-w /var/log/lastlog -p wa -k logins
-w /var/log/tallylog -p wa -k logins

# CIS 4.1.9 - Session initiation
-w /var/run/utmp -p wa -k session
-w /var/log/wtmp -p wa -k session
-w /var/log/btmp -p wa -k session

# CIS 4.1.11 - Privileged commands (generate with find)
# find / -xdev -type f \( -perm -4000 -o -perm -2000 \) -print | while read f; do
#   echo "-a always,exit -F path=$f -F perm=x -F auid>=1000 -F auid!=4294967295 -k privileged"
# done

# CIS 4.1.14 - File access attempts
-a always,exit -F arch=b64 -S open -S truncate -S ftruncate -S creat -S openat -F exit=-EACCES -F auid>=1000 -F auid!=4294967295 -k access
-a always,exit -F arch=b64 -S open -S truncate -S ftruncate -S creat -S openat -F exit=-EPERM -F auid>=1000 -F auid!=4294967295 -k access

# CIS 4.1.15 - Successful file system mounts
-a always,exit -F arch=b64 -S mount -F auid>=1000 -F auid!=4294967295 -k mounts

# CIS 4.1.16 - File deletion events
-a always,exit -F arch=b64 -S unlink -S unlinkat -S rename -S renameat -F auid>=1000 -F auid!=4294967295 -k delete

# CIS 4.1.17 - Changes to sysadmin scope (sudoers)
-w /etc/sudoers -p wa -k scope
-w /etc/sudoers.d/ -p wa -k scope
EOF
```

## Searching Audit Logs with ausearch

```bash
# Search by key
sudo ausearch -k identity_file

# Search by time range
sudo ausearch -ts today
sudo ausearch -ts recent                  # last 10 minutes
sudo ausearch -ts 03/25/2026 09:00:00 -te 03/25/2026 17:00:00

# Search by user (audit UID)
sudo ausearch -ua 1000

# Search by event type
sudo ausearch -m USER_LOGIN
sudo ausearch -m EXECVE
sudo ausearch -m AVC                      # SELinux denials

# Search by process
sudo ausearch -p 12345                    # by PID
sudo ausearch -c sshd                     # by command name

# Search for failed events
sudo ausearch --success no

# Search by file
sudo ausearch -f /etc/passwd

# Combine filters (AND logic)
sudo ausearch -k identity_file -ua 1000 -ts today

# Output in interpretable format
sudo ausearch -k identity_file -i

# Output raw for piping
sudo ausearch -k identity_file --raw | aureport --file
```

## Reporting with aureport

```bash
# Summary report of all events
sudo aureport --summary

# Authentication report
sudo aureport --auth

# Failed authentication attempts
sudo aureport --auth --failed

# Anomaly report
sudo aureport --anomaly

# File access report
sudo aureport --file

# Executable report
sudo aureport --executable

# User report
sudo aureport --user

# System event report
sudo aureport --event

# Login report
sudo aureport --login --failed

# Reports by time range
sudo aureport --auth -ts today
sudo aureport --login -ts this-week

# Key-based report (shows events by audit key)
sudo aureport --key

# Terminal-based report
sudo aureport --tty
```

## PAM and Sudo Auditing

```bash
# Audit PAM configuration changes
sudo auditctl -w /etc/pam.d/ -p wa -k pam_config

# Audit sudo usage
sudo auditctl -w /var/log/sudo.log -p wa -k sudo_log
sudo auditctl -w /usr/bin/sudo -p x -k sudo_usage
sudo auditctl -w /usr/bin/su -p x -k su_usage

# Search for sudo events
sudo ausearch -k sudo_usage -i
```

## Log Management

### Log Rotation

```bash
# Audit log configuration in /etc/audit/auditd.conf
# Key settings in /etc/audit/auditd.conf
log_file = /var/log/audit/audit.log
log_format = ENRICHED
max_log_file = 50           # Max log file size in MB
num_logs = 10               # Number of rotated logs to keep
max_log_file_action = ROTATE
space_left = 75             # MB remaining triggers space_left_action
space_left_action = SYSLOG
admin_space_left = 50
admin_space_left_action = SUSPEND
disk_full_action = SUSPEND
disk_error_action = SUSPEND
```

### Remote Logging

```bash
# Configure audispd to send logs to remote syslog
# /etc/audit/plugins.d/syslog.conf (or /etc/audisp/plugins.d/syslog.conf)
active = yes
direction = out
path = builtin_syslog
type = builtin
args = LOG_INFO
format = string

# For dedicated audit log forwarding with audisp-remote
# /etc/audit/plugins.d/au-remote.conf
active = yes
direction = out
path = /sbin/audisp-remote
type = always

# /etc/audit/audisp-remote.conf
remote_server = audit-collector.example.com
port = 60
transport = tcp
```

## Tips

- Use meaningful key names (`-k`) for every rule; they make `ausearch` and `aureport` filtering practical.
- Start with CIS benchmark rules as a baseline and add site-specific rules on top.
- Use `-e 2` in your finalize rules to make the audit configuration immutable until reboot.
- Set buffer size (`-b`) high enough for your workload; dropped events defeat the purpose of auditing.
- Monitor `/var/log/audit/` disk usage; audit logs grow fast on busy systems.
- Use `aureport --summary` regularly to spot trends and anomalies.
- Combine `ausearch` with `aureport` by piping raw output: `ausearch --raw -k mykey | aureport --file`.
- Forward audit logs to a central SIEM for correlation and long-term retention.
- Test new rules in permissive/non-immutable mode before locking them down with `-e 2`.

## See Also

- selinux, log-analysis, forensics, hardening-linux, acl

## References

- [Linux Audit Documentation](https://github.com/linux-audit/audit-documentation/wiki)
- [auditd(8) man page](https://man7.org/linux/man-pages/man8/auditd.8.html)
- [auditctl(8) man page](https://man7.org/linux/man-pages/man8/auditctl.8.html)
- [CIS Benchmarks](https://www.cisecurity.org/cis-benchmarks)
- [NIST SP 800-92 - Guide to Computer Security Log Management](https://csrc.nist.gov/publications/detail/sp/800-92/final)
- [Red Hat - System Auditing](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/security_hardening/auditing-the-system_security-hardening)
- [aureport(8) man page](https://man7.org/linux/man-pages/man8/aureport.8.html)
- [ausearch(8) man page](https://man7.org/linux/man-pages/man8/ausearch.8.html)
