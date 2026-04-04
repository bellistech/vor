# usermod (modify users)

Modify existing user account properties.

## Change Shell

### Login Shell

```bash
usermod -s /bin/bash deploy
usermod -s /usr/sbin/nologin olduser
```

## Groups

### Modify Group Membership

```bash
# APPEND to supplementary groups (use -a with -G)
usermod -aG docker deploy
usermod -aG sudo,docker,www-data deploy

# SET supplementary groups (replaces all existing groups!)
usermod -G docker deploy

# Change primary group
usermod -g developers deploy
```

## Home Directory

### Change Home

```bash
# Change home directory path
usermod -d /opt/deploy deploy

# Change AND move contents to new location
usermod -d /opt/deploy -m deploy
```

## Lock and Unlock

### Disable/Enable Login

```bash
# Lock account (prepends ! to password hash)
usermod -L deploy

# Unlock account
usermod -U deploy

# Lock and set shell to nologin (belt and suspenders)
usermod -L -s /usr/sbin/nologin deploy
```

## Account Expiry

### Set Expiration

```bash
# Set expiry date
usermod -e 2024-12-31 contractor

# Remove expiry (never expire)
usermod -e "" deploy

# Check current expiry
chage -l deploy
```

## Change Username

### Rename User

```bash
# Rename login name
usermod -l newname oldname

# Rename and move home directory
usermod -l newname -d /home/newname -m oldname
```

## Change UID

### Reassign User ID

```bash
# Change UID
usermod -u 1500 deploy

# Fix file ownership after UID change
find / -user 1000 -exec chown -h 1500 {} \;
```

## Comment

### Update Description

```bash
usermod -c "Deploy Service Account" deploy
```

## Tips

- `usermod -G` without `-a` REPLACES all supplementary groups. This is the single most common mistake. Always use `-aG` to append.
- The user must be logged out (no running processes) to change username or UID. Kill their sessions first.
- `-L` only locks password authentication. The user can still log in via SSH keys. To fully block access, also set the shell to `/usr/sbin/nologin`.
- After changing a UID, files owned by the old UID are orphaned. Use `find / -user OLD_UID` to find and fix them.
- `usermod -e 1` (epoch day 1) effectively disables the account immediately.
- Changes take effect on next login, not for currently active sessions.

## See Also

- useradd, passwd, groups, sudo

## References

- [man usermod(8)](https://man7.org/linux/man-pages/man8/usermod.8.html)
- [man useradd(8)](https://man7.org/linux/man-pages/man8/useradd.8.html)
- [man userdel(8)](https://man7.org/linux/man-pages/man8/userdel.8.html)
- [man passwd(5) — /etc/passwd](https://man7.org/linux/man-pages/man5/passwd.5.html)
- [man shadow(5) — /etc/shadow](https://man7.org/linux/man-pages/man5/shadow.5.html)
- [man login.defs(5)](https://man7.org/linux/man-pages/man5/login.defs.5.html)
- [Arch Wiki — Users and Groups](https://wiki.archlinux.org/title/Users_and_groups)
- [Red Hat — Modifying User Accounts](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-users-and-groups_configuring-basic-system-settings)
- [Ubuntu Manpage — usermod](https://manpages.ubuntu.com/manpages/noble/man8/usermod.8.html)
