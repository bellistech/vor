# SSSD (System Security Services Daemon)

Central daemon for identity and authentication on Linux -- connects to LDAP, Active Directory, IPA, and Kerberos with local caching for offline auth.

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                     Applications                        │
│   login, su, sudo, ssh, httpd, etc.                    │
└──────────┬─────────────────┬───────────────────────────┘
           │ NSS             │ PAM
┌──────────▼─────────┐  ┌───▼───────────────────────────┐
│  sssd_nss          │  │  sssd_pam                      │
│  (NSS Responder)   │  │  (PAM Responder)               │
├────────────────────┤  ├────────────────────────────────┤
│  sssd_sudo         │  │  sssd_ssh     sssd_autofs      │
│  (Sudo Responder)  │  │  (SSH Resp)   (Autofs Resp)    │
└──────────┬─────────┘  └───┬───────────────────────────┘
           │                │
┌──────────▼────────────────▼───────────────────────────┐
│              sssd (Monitor)                            │
│              Manages all child processes               │
└──────────┬────────────────────────────────────────────┘
           │
┌──────────▼────────────────────────────────────────────┐
│  Backend Providers (per domain)                        │
│  sssd_be                                               │
│  ├── id_provider    (identity: users, groups)          │
│  ├── auth_provider  (authentication: password, krb5)   │
│  ├── access_provider (authorization: allow/deny login) │
│  ├── chpass_provider (password changes)                │
│  ├── sudo_provider  (sudo rules)                       │
│  └── autofs_provider (automount maps)                  │
└───────────────────────────────────────────────────────┘
```

## Installation

```bash
# RHEL/CentOS/Fedora
sudo dnf install sssd sssd-tools sssd-ldap sssd-ad sssd-ipa sssd-krb5

# Debian/Ubuntu
sudo apt install sssd sssd-tools sssd-ldap sssd-ad libnss-sss libpam-sss

# Enable and start
sudo systemctl enable --now sssd
```

## Basic Configuration

### Minimal sssd.conf

```bash
# /etc/sssd/sssd.conf (must be mode 0600)
[sssd]
domains = example.com
services = nss, pam, sudo
config_file_version = 2

[domain/example.com]
id_provider = ldap
auth_provider = ldap
ldap_uri = ldaps://ldap.example.com
ldap_search_base = dc=example,dc=com
ldap_default_bind_dn = cn=sssd-bind,ou=Service,dc=example,dc=com
ldap_default_authtok = s3cret
ldap_tls_reqcert = demand
ldap_tls_cacert = /etc/pki/tls/certs/ca-bundle.crt

cache_credentials = true
enumerate = false

[nss]
filter_groups = root
filter_users = root

[pam]
offline_credentials_expiration = 7
```

```bash
sudo chmod 0600 /etc/sssd/sssd.conf
sudo systemctl restart sssd
```

## Identity Providers

### LDAP provider

```ini
[domain/example.com]
id_provider = ldap
ldap_uri = ldaps://ldap1.example.com, ldaps://ldap2.example.com
ldap_search_base = dc=example,dc=com
ldap_user_search_base = ou=People,dc=example,dc=com
ldap_group_search_base = ou=Groups,dc=example,dc=com

# Schema mapping (RFC2307 vs RFC2307bis)
ldap_schema = rfc2307bis                    # AD-style nested groups
# ldap_schema = rfc2307                     # NIS-style (memberUid)

ldap_user_object_class = posixAccount
ldap_user_name = uid
ldap_user_uid_number = uidNumber
ldap_user_gid_number = gidNumber
ldap_user_home_directory = homeDirectory
ldap_user_shell = loginShell

ldap_group_object_class = posixGroup
ldap_group_name = cn
ldap_group_gid_number = gidNumber
ldap_group_member = member                  # rfc2307bis
```

### Active Directory provider

```ini
[domain/ad.example.com]
id_provider = ad
auth_provider = ad
access_provider = ad
chpass_provider = ad

ad_domain = ad.example.com
ad_server = dc1.ad.example.com, dc2.ad.example.com
ad_hostname = linuxhost.ad.example.com

# ID mapping (automatic SID → UID/GID)
ldap_id_mapping = true                       # auto-map SID → UID (no POSIX attrs needed)
# ldap_id_mapping = false                    # use POSIX attrs from AD (uidNumber/gidNumber)

# Home directory and shell
fallback_homedir = /home/%u@%d
default_shell = /bin/bash
override_homedir = /home/%u                  # simpler: strip domain from path

# GPO-based access control
ad_gpo_access_control = enforcing            # enforcing | permissive | disabled
ad_gpo_map_interactive = +allow              # map GPO "Allow log on locally"
```

### IPA (FreeIPA) provider

```ini
[domain/ipa.example.com]
id_provider = ipa
auth_provider = ipa
access_provider = ipa
chpass_provider = ipa
sudo_provider = ipa

ipa_domain = ipa.example.com
ipa_server = ipa1.ipa.example.com
ipa_hostname = client.ipa.example.com

# HBAC (Host-Based Access Control)
# Managed on IPA server, SSSD enforces locally
```

### Kerberos auth provider

```ini
[domain/example.com]
id_provider = ldap
auth_provider = krb5

krb5_server = kdc1.example.com, kdc2.example.com
krb5_realm = EXAMPLE.COM
krb5_kpasswd = kdc1.example.com
krb5_keytab = /etc/krb5.keytab
krb5_renewable_lifetime = 7d
krb5_renew_interval = 3600
```

## AD Integration (realm join)

### Join domain with realmd

```bash
# Install realmd
sudo dnf install realmd oddjob oddjob-mkhomedir adcli samba-common-tools

# Discover domain
realm discover ad.example.com

# Join domain
sudo realm join ad.example.com -U admin@AD.EXAMPLE.COM
# Creates /etc/sssd/sssd.conf, /etc/krb5.keytab automatically

# Verify join
realm list
sudo adcli info ad.example.com

# Permit specific users/groups
sudo realm permit user1@ad.example.com
sudo realm permit -g "linux-admins@ad.example.com"
sudo realm deny --all                        # deny all, then permit specific
```

### Kerberos keytab

```bash
# View keytab entries
sudo klist -ke /etc/krb5.keytab

# Test keytab authentication
sudo kinit -k -t /etc/krb5.keytab host/linuxhost.ad.example.com@AD.EXAMPLE.COM
klist

# Renew keytab (if machine password changes)
sudo adcli update --computer-password-lifetime=30
```

## Access Providers

### Simple access provider

```ini
[domain/example.com]
access_provider = simple

simple_allow_users = admin, user1, user2
simple_allow_groups = linux-users, admins
# simple_deny_users = baduser                # deny takes precedence
```

### LDAP access filter

```ini
[domain/example.com]
access_provider = ldap
ldap_access_filter = (memberOf=cn=linux-users,ou=Groups,dc=example,dc=com)
# Only users in linux-users group can log in
```

### GPO-based access control (AD)

```ini
[domain/ad.example.com]
access_provider = ad
ad_gpo_access_control = enforcing

# Map GPO settings to Linux login types
ad_gpo_map_interactive = +allow              # console login
ad_gpo_map_remote_interactive = +allow       # SSH login
ad_gpo_map_service = +allow                  # service accounts
ad_gpo_map_batch = +allow                    # cron jobs
```

## Sudo Provider

### LDAP-based sudo rules

```ini
[sssd]
services = nss, pam, sudo

[domain/example.com]
sudo_provider = ldap
ldap_sudo_search_base = ou=SUDOers,dc=example,dc=com
```

### IPA-based sudo rules

```ini
[domain/ipa.example.com]
sudo_provider = ipa
# Rules managed on IPA server via:
#   ipa sudorule-add RULE-NAME
#   ipa sudorule-add-user RULE-NAME --users=admin
#   ipa sudorule-add-host RULE-NAME --hosts=client.ipa.example.com
#   ipa sudorule-add-allow-command RULE-NAME --sudocmds=/usr/bin/systemctl
```

## Autofs Provider

```ini
[sssd]
services = nss, pam, autofs

[domain/example.com]
autofs_provider = ldap
ldap_autofs_search_base = ou=automount,dc=example,dc=com
ldap_autofs_map_object_class = automountMap
ldap_autofs_entry_object_class = automount
```

## PAM / NSS Integration

### NSS configuration

```bash
# /etc/nsswitch.conf
passwd:     files sss
shadow:     files sss
group:      files sss
services:   files sss
netgroup:   files sss
automount:  files sss
sudoers:    files sss
```

### PAM configuration

```bash
# /etc/pam.d/system-auth or common-auth (auto-configured by authselect/pam-auth-update)

# RHEL/CentOS — use authselect
sudo authselect select sssd with-mkhomedir
sudo systemctl enable --now oddjobd          # auto-create home dirs

# Debian/Ubuntu — use pam-auth-update
sudo pam-auth-update                         # select "SSS authentication"
```

### Auto-create home directories

```bash
# Via authselect (RHEL)
sudo authselect select sssd with-mkhomedir

# Via PAM manually
# /etc/pam.d/common-session (Debian/Ubuntu)
session required pam_mkhomedir.so skel=/etc/skel/ umask=0077
```

## Smart Card Authentication

```ini
[pam]
pam_cert_auth = true
p11_child_timeout = 60

[domain/example.com]
certificate_verification = ocsp_dgst=sha256
```

```bash
# Install smart card support
sudo dnf install sssd-tools opensc pcsc-lite
sudo systemctl enable --now pcscd
```

## Cache Management

### Cache operations

```bash
# Invalidate all cached data
sudo sss_cache -E

# Invalidate specific user
sudo sss_cache -u username

# Invalidate specific group
sudo sss_cache -g groupname

# Invalidate sudo rules
sudo sss_cache -s                            # all sudo rules
sudo sss_cache -S sudorulename               # specific rule

# View cache contents
sudo sssctl cache-list                       # list cached entries
sudo sssctl user-show username               # show cached user
sudo sssctl group-show groupname             # show cached group
```

### Cache tuning

```ini
[domain/example.com]
# How long cached entries are valid before refresh
entry_cache_timeout = 5400                   # 90 min (default)
entry_cache_user_timeout = 5400
entry_cache_group_timeout = 5400
entry_cache_sudo_timeout = 5400

# How long to use cache when offline
cache_credentials = true
offline_credentials_expiration = 7           # days (0 = forever)

# Enumerate users/groups (AVOID for large directories)
enumerate = false
```

## Failover and SRV Records

### Manual failover

```ini
[domain/example.com]
ldap_uri = ldaps://ldap1.example.com, ldaps://ldap2.example.com, ldaps://ldap3.example.com
# Tried in order; fails over to next on connection failure
```

### DNS SRV record discovery

```ini
[domain/example.com]
id_provider = ad
# With AD provider, SRV discovery is automatic:
#   _ldap._tcp.ad.example.com   → LDAP servers
#   _kerberos._tcp.ad.example.com → KDC servers
#   _gc._tcp.ad.example.com     → Global Catalog servers

dns_discovery_domain = ad.example.com
# Or override:
# ad_server = _srv_                          # explicit SRV lookup
# ad_backup_server = dc3.ad.example.com      # backup if SRV fails
```

## Troubleshooting

### Debug logging

```ini
[domain/example.com]
debug_level = 6                              # 0-9 (6+ for detailed debug)

[nss]
debug_level = 6

[pam]
debug_level = 6
```

```bash
sudo systemctl restart sssd
# Logs in /var/log/sssd/
#   sssd.log          — monitor process
#   sssd_example.com.log — backend provider
#   sssd_nss.log      — NSS responder
#   sssd_pam.log      — PAM responder
```

### sssctl diagnostic tool

```bash
sssctl domain-list                           # list configured domains
sssctl domain-status example.com             # domain connectivity
sssctl config-check                          # validate sssd.conf syntax
sssctl user-checks username                  # test user lookup + auth
sssctl logs-fetch /tmp/sssd-debug.tar.gz     # collect all logs
sssctl logs-remove                           # clear log files
```

### Common troubleshooting

```bash
# User not found
getent passwd username                       # test NSS lookup
id username                                  # test identity resolution
sudo sss_cache -u username                   # clear user cache
sudo sssctl user-checks username             # full diagnostic

# Authentication fails
sudo journalctl -u sssd -n 50               # recent SSSD logs
sudo cat /var/log/sssd/sssd_pam.log          # PAM-specific issues

# Connection issues
sudo sssctl domain-status example.com
sudo ldapsearch -x -H ldaps://ldap.example.com -b "dc=example,dc=com" "(uid=testuser)"
sudo kinit admin@EXAMPLE.COM                 # test Kerberos directly

# Permission denied on sssd.conf
ls -la /etc/sssd/sssd.conf                   # must be 0600, owned by root
sudo chmod 0600 /etc/sssd/sssd.conf

# SSSD won't start after config change
sudo sssctl config-check                     # validate syntax
sudo rm -f /var/lib/sss/db/*                 # clear all caches (nuclear option)
sudo systemctl restart sssd
```

## See Also

- ldap
- kerberos
- pam
- saml
- oidc

## References

- SSSD Documentation: https://sssd.io/docs/
- Red Hat: Configuring Authentication and Authorization (RHEL 9)
- FreeIPA Guide: https://www.freeipa.org/page/Documentation
- man sssd.conf(5), man sssd-ldap(5), man sssd-ad(5), man sssd-krb5(5)
- man sss_cache(8), man sssctl(8)
