# useradd (create users)

Create new user accounts on the system.

## Basic User Creation

### Create a User

```bash
# Create user with defaults (home dir creation depends on distro)
useradd deploy

# Create user with home directory
useradd -m deploy

# Create user with specific home directory
useradd -m -d /opt/deploy deploy

# Create user and set password immediately
useradd -m deploy && echo "deploy:secretpass" | chpasswd
```

## Shell

### Set Login Shell

```bash
# Create with bash shell
useradd -m -s /bin/bash deploy

# Create with no login shell (for service accounts)
useradd -s /usr/sbin/nologin serviceuser

# Create with /bin/false (alternative no-login)
useradd -s /bin/false serviceuser
```

## Groups

### Primary and Supplementary Groups

```bash
# Set primary group
useradd -m -g developers deploy

# Add supplementary groups
useradd -m -G sudo,docker,www-data deploy

# Primary group + supplementary groups
useradd -m -g developers -G sudo,docker deploy
```

## System Users

### Service Accounts

```bash
# Create system user (low UID, no home by default, no aging)
useradd -r -s /usr/sbin/nologin myservice

# System user with a specific home for data
useradd -r -m -d /var/lib/myservice -s /usr/sbin/nologin myservice
```

## UID and GID

### Specify IDs

```bash
# Specific UID
useradd -m -u 1500 deploy

# Specific UID and GID
useradd -m -u 1500 -g 1500 deploy

# Create the group first if it doesn't exist
groupadd -g 1500 deploy && useradd -m -u 1500 -g 1500 deploy
```

## Skeleton Directory

### Home Directory Template

```bash
# Use default skeleton (/etc/skel)
useradd -m deploy

# Use custom skeleton directory
useradd -m -k /etc/skel-developers deploy

# /etc/skel typically contains:
#   .bashrc, .profile, .bash_logout
```

## Account Expiry

### Expiration Date

```bash
# Account expires on a specific date
useradd -m -e 2024-12-31 contractor

# No expiry (default)
useradd -m deploy
```

## Comment Field

### Full Name / Description

```bash
useradd -m -c "Deploy User" deploy
useradd -m -c "CI/CD Service Account" cicd
```

## Complete Examples

### Typical Patterns

```bash
# Standard developer account
useradd -m -s /bin/bash -G sudo,docker -c "Jane Developer" jane

# Service account for an application
useradd -r -s /usr/sbin/nologin -d /var/lib/myapp -m myapp

# Temporary contractor with expiry
useradd -m -s /bin/bash -G developers -e 2024-06-30 -c "Contractor Bob" bob

# LDAP/AD user with matching UID
useradd -m -u 10042 -s /bin/bash -G developers deploy
```

## Tips

- `-m` (create home directory) is not the default on all distros. RHEL/CentOS creates it by default; Debian does not. Always use `-m` to be explicit.
- `useradd` is the low-level tool. `adduser` on Debian is a friendlier wrapper that prompts interactively.
- The skeleton directory (`/etc/skel`) is copied to new home directories. Customize it for org-wide defaults.
- `-r` for system users picks a UID below 1000 (or below `SYS_UID_MAX` in `/etc/login.defs`).
- Always set `-s /usr/sbin/nologin` for service accounts to prevent interactive login.
- Check defaults with `useradd -D` and change them with `useradd -D -s /bin/bash`.

## See Also

- usermod, passwd, groups, sudo

## References

- [man useradd(8)](https://man7.org/linux/man-pages/man8/useradd.8.html)
- [man adduser(8)](https://man7.org/linux/man-pages/man8/adduser.8.html)
- [man userdel(8)](https://man7.org/linux/man-pages/man8/userdel.8.html)
- [man passwd(5) — /etc/passwd](https://man7.org/linux/man-pages/man5/passwd.5.html)
- [man login.defs(5)](https://man7.org/linux/man-pages/man5/login.defs.5.html)
- [man useradd defaults — /etc/default/useradd](https://man7.org/linux/man-pages/man8/useradd.8.html)
- [Arch Wiki — Users and Groups](https://wiki.archlinux.org/title/Users_and_groups)
- [Red Hat — Adding Users](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-users-and-groups_configuring-basic-system-settings)
- [Ubuntu — AddUsersHowto](https://help.ubuntu.com/community/AddUsersHowto)
