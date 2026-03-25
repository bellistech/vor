# passwd (manage passwords)

Change user passwords and manage password aging policies.

## Change Password

### Interactive

```bash
# Change your own password
passwd

# Change another user's password (root)
passwd deploy

# Force password change on next login
passwd -e deploy
```

### Non-Interactive

```bash
# Set password from stdin (for scripting)
echo "deploy:newpassword" | chpasswd

# Using passwd with stdin (some systems)
echo "newpassword" | passwd --stdin deploy

# Generate and set a random password
openssl rand -base64 16 | passwd --stdin deploy
```

## Lock and Unlock

### Account Locking

```bash
# Lock (disable password authentication)
passwd -l deploy

# Unlock
passwd -u deploy

# Check lock status
passwd -S deploy
# Output: deploy P 2024-01-15 0 99999 7 -1
#         P = active password
#         L = locked
#         NP = no password
```

## Password Status

### View Account Info

```bash
# Show password status
passwd -S deploy

# Show all accounts status (root)
passwd -Sa

# Detailed aging info
chage -l deploy
```

## Password Aging

### Set Password Policies

```bash
# Minimum days between password changes
passwd -n 1 deploy

# Maximum days before password must change
passwd -x 90 deploy

# Warning days before expiry
passwd -w 7 deploy

# Days after expiry before account is disabled
passwd -i 30 deploy

# Set all aging with chage (more flexible)
chage -m 1 -M 90 -W 7 -I 30 deploy

# Force password change on next login
chage -d 0 deploy

# Set specific expiration date
chage -E 2024-12-31 deploy

# Remove expiration
chage -E -1 deploy
```

## Delete Password

### Remove Password

```bash
# Remove password (allow passwordless login — dangerous)
passwd -d deploy
```

## Tips

- `passwd -l` only disables password-based login. SSH key authentication still works. Use `usermod -L -s /usr/sbin/nologin` for a full lockout.
- `chpasswd` is the standard tool for bulk password setting in scripts; `passwd --stdin` is not portable (RHEL-specific).
- Password hashes are stored in `/etc/shadow`, not `/etc/passwd`. Only root can read `/etc/shadow`.
- `passwd -S` output format: `username status last_change min max warn inactive`.
- PAM modules (`/etc/pam.d/`) control password complexity requirements -- not `passwd` itself.
- `passwd -e` (expire) forces a password change at next login and is the standard way to provision temporary passwords.

## References

- [man passwd(1)](https://man7.org/linux/man-pages/man1/passwd.1.html)
- [man passwd(5) — /etc/passwd](https://man7.org/linux/man-pages/man5/passwd.5.html)
- [man shadow(5) — /etc/shadow](https://man7.org/linux/man-pages/man5/shadow.5.html)
- [man chage(1) — Password Aging](https://man7.org/linux/man-pages/man1/chage.1.html)
- [man login.defs(5)](https://man7.org/linux/man-pages/man5/login.defs.5.html)
- [man crypt(5) — Password Hashing](https://man7.org/linux/man-pages/man5/crypt.5.html)
- [Arch Wiki — Users and Groups](https://wiki.archlinux.org/title/Users_and_groups)
- [Red Hat — Managing User Passwords](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_basic_system_settings/managing-users-and-groups_configuring-basic-system-settings)
- [Ubuntu — User Management](https://help.ubuntu.com/community/AddUsersHowto)
