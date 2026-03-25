# PAM (Pluggable Authentication Modules)

> Modular framework for authenticating users on Linux, separating auth logic from applications.

## PAM Stack

### Module Types

```
auth        # Verify identity (password, token, biometric)
account     # Check account validity (expiration, access restrictions)
password    # Update authentication tokens (password changes)
session     # Setup/teardown session (mount homedir, set ulimits, logging)
```

### Control Flags

```
required    # Must pass; continue checking other modules even on failure
requisite   # Must pass; stop immediately on failure
sufficient  # If passes (and no prior required failed), accept immediately
optional    # Result only matters if it is the only module for this type
include     # Include rules from another PAM config file
substack    # Like include, but failure only affects the substack
```

## Configuration

### /etc/pam.d/ Structure

```bash
ls /etc/pam.d/
# common-auth        # Shared auth rules (Debian/Ubuntu)
# common-account     # Shared account rules
# common-password    # Shared password rules
# common-session     # Shared session rules
# sshd              # SSH-specific config
# login             # Console login
# sudo              # sudo-specific config
# su                # su-specific config
```

### Config File Format

```
# type    control    module                 [arguments]
auth      required   pam_env.so
auth      required   pam_unix.so            nullok try_first_pass
auth      sufficient pam_google_authenticator.so
account   required   pam_unix.so
password  required   pam_unix.so            sha512 shadow remember=5
session   required   pam_limits.so
session   required   pam_unix.so
```

## Common Modules

### pam_unix — Standard Password Auth

```
auth      required   pam_unix.so nullok try_first_pass
account   required   pam_unix.so
password  required   pam_unix.so sha512 shadow remember=5 minlen=8
session   required   pam_unix.so

# Options:
#   nullok          — allow empty passwords
#   try_first_pass  — use password from prior module before prompting
#   sha512          — hash algorithm for passwords
#   shadow          — use /etc/shadow
#   remember=N      — reject last N passwords
```

### pam_ldap — LDAP Authentication

```
auth      sufficient pam_ldap.so use_first_pass
account   sufficient pam_ldap.so
password  sufficient pam_ldap.so use_authtok
session   optional   pam_ldap.so

# Requires /etc/ldap.conf or /etc/pam_ldap.conf
#   base   dc=example,dc=com
#   uri    ldap://ldap.example.com
#   binddn cn=proxyuser,dc=example,dc=com
```

### pam_google_authenticator — TOTP 2FA

```bash
# Install
apt install libpam-google-authenticator    # Debian/Ubuntu
yum install google-authenticator           # RHEL/CentOS

# Per-user setup
google-authenticator   # Generates QR code and scratch codes
```

```
# /etc/pam.d/sshd — add before or after pam_unix
auth      required   pam_google_authenticator.so nullok

# Options:
#   nullok           — allow users without 2FA configured
#   no_increment     — don't increment counter on failed attempts
#   forward_pass     — pass OTP as password to next module
#   noskewadj        — disable time skew compensation
```

### pam_faillock — Account Lockout

```
# /etc/pam.d/common-auth (or system-auth)
auth      required   pam_faillock.so preauth silent deny=5 unlock_time=900
auth      required   pam_unix.so try_first_pass
auth      [default=die] pam_faillock.so authfail deny=5 unlock_time=900

# Options:
#   deny=N           — lock after N failed attempts
#   unlock_time=N    — unlock after N seconds (0 = manual unlock)
#   fail_interval=N  — count failures within N seconds
#   even_deny_root   — apply lockout to root as well
```

```bash
# View failed attempts
faillock --user username

# Reset lockout
faillock --user username --reset
```

### Other Useful Modules

```
pam_limits.so     # Enforce /etc/security/limits.conf (ulimits)
pam_access.so     # Host/network based access control (/etc/security/access.conf)
pam_time.so       # Time-based access control (/etc/security/time.conf)
pam_mkhomedir.so  # Auto-create home directory on first login
pam_motd.so       # Display message of the day
pam_env.so        # Set environment variables from /etc/security/pam_env.conf
pam_wheel.so      # Restrict su to wheel group members
pam_tally2.so     # Legacy login counter (deprecated, use pam_faillock)
```

## Example Configurations

### SSH with 2FA

```
# /etc/pam.d/sshd
auth       required     pam_env.so
auth       required     pam_unix.so try_first_pass
auth       required     pam_google_authenticator.so
account    required     pam_unix.so
session    required     pam_limits.so
session    required     pam_unix.so
```

```
# /etc/ssh/sshd_config
ChallengeResponseAuthentication yes
AuthenticationMethods publickey,keyboard-interactive
```

## Tips

- Test PAM changes with a second open root session; a misconfiguration can lock you out entirely.
- Use `pam_warn.so` in a test stack to log module execution without affecting auth.
- The order of modules matters: `sufficient` short-circuits, so place it after `required` modules.
- Debian uses `common-*` files; RHEL uses `system-auth` and `password-auth`.
- Always keep a rescue/root console available when editing PAM configs.

## References

- [Linux-PAM Documentation](http://www.linux-pam.org/Linux-PAM-html/)
- [pam(8) Man Page](https://man7.org/linux/man-pages/man8/pam.8.html)
- [pam.conf(5) / pam.d(5) Man Page](https://man7.org/linux/man-pages/man5/pam.d.5.html)
- [pam_unix(8) Man Page](https://man7.org/linux/man-pages/man8/pam_unix.8.html)
- [pam_faillock(8) Man Page](https://man7.org/linux/man-pages/man8/pam_faillock.8.html)
- [pam_tally2(8) Man Page](https://man7.org/linux/man-pages/man8/pam_tally2.8.html)
- [pam_limits(8) Man Page](https://man7.org/linux/man-pages/man8/pam_limits.8.html)
- [Red Hat RHEL 9 — Configuring PAM](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/security_hardening/configuring-pam-for-hardening-authentication_security-hardening)
- [Arch Wiki — PAM](https://wiki.archlinux.org/title/PAM)
- [Ubuntu — PAM Configuration Guide](https://ubuntu.com/server/docs/pam-configuration)
