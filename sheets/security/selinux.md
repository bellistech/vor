# SELinux (Security-Enhanced Linux)

Mandatory access control system that confines processes to the minimum privileges they need.

## Status and Modes

### Check Current Mode

```bash
getenforce                               # Enforcing, Permissive, or Disabled
sestatus                                 # full status with policy details
```

### Change Mode (Runtime)

```bash
sudo setenforce 1                        # Enforcing (blocks + logs)
sudo setenforce 0                        # Permissive (logs only, no blocking)
```

### Change Mode (Permanent)

```bash
# /etc/selinux/config
SELINUX=enforcing                        # enforcing | permissive | disabled
SELINUXTYPE=targeted                     # targeted | minimum | mls
```

```bash
# Reboot required after changing from disabled to enforcing
sudo reboot
```

## Security Contexts

### View Contexts

```bash
ls -Z /var/www/html/                     # file contexts
ps -eZ | grep httpd                      # process contexts
id -Z                                    # current user context
```

### Context Format

```bash
# user:role:type:level
# system_u:object_r:httpd_sys_content_t:s0
# The "type" field is what matters most in targeted policy
```

### Change File Context (Temporary)

```bash
sudo chcon -t httpd_sys_content_t /var/www/html/index.html
sudo chcon -R -t httpd_sys_content_t /var/www/html/
```

### Set Default Context (Permanent)

```bash
# Add a file context rule
sudo semanage fcontext -a -t httpd_sys_content_t "/srv/www(/.*)?"

# Apply the rule to disk
sudo restorecon -Rv /srv/www/
```

### List Custom File Contexts

```bash
sudo semanage fcontext -l -C             # only local customizations
sudo semanage fcontext -l | grep /srv    # search for path
```

## Restoring Contexts

```bash
# Restore default contexts recursively
sudo restorecon -Rv /var/www/html/

# Preview what would change (dry run)
sudo restorecon -Rvn /var/www/html/

# Relabel entire filesystem on next boot
sudo touch /.autorelabel && sudo reboot
```

## Booleans

### List and Query Booleans

```bash
getsebool -a                             # all booleans
getsebool httpd_can_network_connect      # specific boolean
sudo semanage boolean -l | grep httpd    # with descriptions
```

### Set Booleans

```bash
# Runtime only
sudo setsebool httpd_can_network_connect on

# Persistent across reboots
sudo setsebool -P httpd_can_network_connect on
sudo setsebool -P httpd_can_sendmail on
```

### Common Useful Booleans

```bash
sudo setsebool -P httpd_can_network_connect on       # HTTPD to backend services
sudo setsebool -P httpd_can_network_connect_db on    # HTTPD to databases
sudo setsebool -P httpd_enable_homedirs on           # HTTPD serve ~/public_html
sudo setsebool -P ftpd_full_access on                # FTP write access
sudo setsebool -P samba_export_all_rw on             # Samba read-write
```

## Port Labeling

```bash
# List port labels
sudo semanage port -l | grep http

# Allow httpd to bind to port 8080
sudo semanage port -a -t http_port_t -p tcp 8080

# Modify existing port label
sudo semanage port -m -t http_port_t -p tcp 8443

# Delete custom port label
sudo semanage port -d -t http_port_t -p tcp 8080
```

## Troubleshooting

### Check for AVC Denials

```bash
# Recent denials from audit log
sudo ausearch -m avc -ts recent

# Denials for a specific process
sudo ausearch -m avc -c httpd

# Today's denials
sudo ausearch -m avc -ts today

# Human-readable with sealert (setroubleshoot package)
sudo sealert -a /var/log/audit/audit.log
```

### Generate Policy from Denials

```bash
# Create a policy module from recent denials
sudo ausearch -m avc -ts recent | audit2allow -M mypolicy

# Review the module before installing
cat mypolicy.te

# Install the module
sudo semodule -i mypolicy.pp
```

### Audit2Why (Explain Denials)

```bash
sudo ausearch -m avc -ts recent | audit2why
# Shows which boolean to enable or what context to fix
```

## Policy Modules

### List Loaded Modules

```bash
sudo semodule -l
```

### Install / Remove Module

```bash
sudo semodule -i mypolicy.pp
sudo semodule -r mypolicy                # remove by name
```

### Disable a Module

```bash
sudo semodule -d mypolicy               # disable without removing
sudo semodule -e mypolicy               # re-enable
```

## User Mapping

```bash
# List SELinux user mappings
sudo semanage login -l

# Map Linux user to SELinux user
sudo semanage login -a -s staff_u alice

# Confined administrator
sudo semanage login -a -s sysadm_u admin_user
```

## Tips

- Start in Permissive mode when deploying new services; switch to Enforcing after resolving all denials
- `restorecon` is safe and idempotent -- run it whenever file contexts look wrong
- `chcon` changes are lost on relabel; always use `semanage fcontext` + `restorecon` for permanent changes
- Install `setroubleshoot-server` and `policycoreutils-python-utils` for `sealert` and `semanage`
- `audit2allow` is a quick fix but can be overly permissive; review generated `.te` files before installing
- Docker and Podman work with SELinux via the `:z` (shared) and `:Z` (private) volume mount suffixes
- The `targeted` policy only confines specific daemons; unconfined processes run as `unconfined_t`
- After changing SELINUX from `disabled` to `enforcing`, a full filesystem relabel is required (can take 10+ minutes)
