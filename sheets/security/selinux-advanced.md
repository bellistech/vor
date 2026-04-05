# Advanced SELinux

Advanced SELinux topics: custom policy modules, MLS/MCS, confined users, type enforcement rules, and policy development workflow.

## SELinux Modes and Policy Types

### Modes

```bash
getenforce                                   # Enforcing | Permissive | Disabled
sestatus                                     # full status

# Runtime change
sudo setenforce 1                            # Enforcing
sudo setenforce 0                            # Permissive (log only)

# Permanent change (/etc/selinux/config)
SELINUX=enforcing                            # enforcing | permissive | disabled
SELINUXTYPE=targeted                         # policy type (see below)
# Reboot required when changing from disabled
```

### Policy types

```
Policy Type   Description                         Use Case
──────────────────────────────────────────────────────────────────
targeted      Only targeted daemons confined       Default for RHEL/Fedora
              (httpd, named, sshd, etc.)           Most processes run unconfined
              ~400 policy modules

minimum       Subset of targeted — only            Minimal installs,
              selected modules loaded              appliances

mls           Multi-Level Security with full       Government / military
              Bell-LaPadula enforcement            (classified data handling)
              All processes confined                Requires MLS-aware userspace
```

## Security Contexts

### Context format

```bash
# user:role:type:level
# system_u:system_r:httpd_t:s0
#
# user  — SELinux user (not Linux user). Maps Linux users to SELinux policy
# role  — RBAC role. Controls which types a user can transition to
# type  — Type Enforcement label. THE key enforcement mechanism in targeted policy
# level — MLS/MCS sensitivity + categories. s0 = default in targeted policy

# View contexts
ls -Z /var/www/html/                         # file contexts
ps -eZ | grep httpd                          # process (domain) contexts
id -Z                                        # current user context
```

### SELinux users

```bash
# List SELinux user mappings
semanage login -l
# Linux User     SELinux User      MLS/MCS Range
# __default__    unconfined_u      s0-s0:c0.c1023
# root           unconfined_u      s0-s0:c0.c1023

# Map Linux user to confined SELinux user
sudo semanage login -a -s staff_u username
sudo semanage login -a -s user_u restricteduser

# SELinux users and their allowed roles
semanage user -l
# SELinux User    Roles
# guest_u         guest_r                    ← no sudo, no GUI, no networking, no su
# xguest_u        xguest_r                   ← GUI + web browser only
# user_u          user_r                     ← no sudo, no su, no networking daemons
# staff_u         staff_r sysadm_r           ← can sudo to sysadm_r
# sysadm_u        sysadm_r                   ← full admin
# unconfined_u    unconfined_r system_r       ← no restrictions (default)
```

## Type Enforcement (TE) Rules

### View type enforcement rules

```bash
# Search for allow rules
sesearch --allow -s httpd_t                  # rules where httpd_t is source
sesearch --allow -t httpd_sys_content_t      # rules where type is target
sesearch --allow -s httpd_t -t httpd_log_t   # specific source → target

# Search for type transitions
sesearch --type_trans -s httpd_t             # domain transitions from httpd_t

# Search for deny (neverallow) rules
sesearch --neverallow -s httpd_t
```

### TE rule syntax

```
# allow source_type target_type : object_class { permissions };

allow httpd_t httpd_sys_content_t : file { read getattr open };
allow httpd_t httpd_log_t : file { create write append open };
allow httpd_t httpd_port_t : tcp_socket { name_bind };

# type_transition: automatic type assignment
type_transition httpd_t httpd_log_t : file httpd_log_t;
# When httpd_t creates a file in a dir labeled httpd_log_t,
# the new file gets type httpd_log_t automatically
```

## File Contexts

### View and manage file contexts

```bash
# View current file context
ls -Z /var/www/html/index.html
# system_u:object_r:httpd_sys_content_t:s0

# View default context rules
semanage fcontext -l | grep /var/www
# /var/www(/.*)?    all files    system_u:object_r:httpd_sys_content_t:s0
```

### Change file context (temporary)

```bash
sudo chcon -t httpd_sys_content_t /var/www/html/index.html
sudo chcon -R -t httpd_sys_content_t /var/www/html/
sudo chcon -u system_u -r object_r -t httpd_sys_content_t /var/www/html/file.html
```

### Set default file context (permanent)

```bash
# Add file context rule
sudo semanage fcontext -a -t httpd_sys_content_t "/srv/www(/.*)?"

# Apply the context (relabel)
sudo restorecon -Rv /srv/www/

# Modify existing rule
sudo semanage fcontext -m -t httpd_sys_rw_content_t "/srv/www/uploads(/.*)?"
sudo restorecon -Rv /srv/www/uploads/

# Delete custom rule
sudo semanage fcontext -d "/srv/www(/.*)?"

# List custom (local) fcontext rules only
sudo semanage fcontext -l -C
```

### Restore default contexts

```bash
restorecon -v /var/www/html/index.html       # single file
restorecon -Rv /var/www/html/                # recursive
restorecon -Rv /                             # full system (slow)

# Check what restorecon would do without changing
restorecon -Rvn /var/www/html/               # dry-run
```

## Booleans

### Manage booleans

```bash
# List all booleans
getsebool -a
getsebool -a | grep httpd

# Check specific boolean
getsebool httpd_can_network_connect
# httpd_can_network_connect --> off

# Set boolean (runtime only)
sudo setsebool httpd_can_network_connect on

# Set boolean (persistent across reboots)
sudo setsebool -P httpd_can_network_connect on

# List all booleans with description
semanage boolean -l | grep httpd
```

### Common booleans

```bash
# Web server
httpd_can_network_connect          # allow httpd outbound connections
httpd_can_network_connect_db       # allow httpd to connect to databases
httpd_enable_homedirs              # allow httpd to read user home dirs
httpd_can_sendmail                 # allow httpd to send mail
httpd_use_nfs                      # allow httpd to access NFS mounts

# Samba
samba_enable_home_dirs             # allow Samba to share home dirs
samba_export_all_ro                # allow Samba read-only access everywhere
samba_export_all_rw                # allow Samba read-write access everywhere

# NFS
nfs_export_all_ro                  # allow NFS read-only exports
nfs_export_all_rw                  # allow NFS read-write exports
use_nfs_home_dirs                  # allow NFS home directories

# General
allow_execmem                      # allow processes to execute memory (JIT)
allow_user_exec_content            # allow users to execute in home/tmp
ftp_home_dir                       # allow FTP access to home dirs
```

## Port Contexts

### Manage port labels

```bash
# List port contexts
semanage port -l | grep http
# http_port_t       tcp      80, 81, 443, 488, 8008, 8009, 8443, 9000

# Add custom port
sudo semanage port -a -t http_port_t -p tcp 8080
sudo semanage port -a -t http_port_t -p tcp 3000

# Modify port type
sudo semanage port -m -t http_port_t -p tcp 8888

# Delete custom port
sudo semanage port -d -t http_port_t -p tcp 8080

# List local (custom) port modifications
sudo semanage port -l -C
```

## Custom Policy Modules

### Workflow: audit2allow

```bash
# Step 1: Trigger the denial (run the application in permissive or check logs)
sudo ausearch -m avc -ts recent              # find recent denials
sudo ausearch -m avc -c httpd                # denials for httpd

# Step 2: Generate human-readable explanation
sudo ausearch -m avc -ts recent | audit2why

# Step 3: Generate policy module
sudo ausearch -m avc -ts recent | audit2allow -M mymodule
# Creates: mymodule.te (type enforcement) and mymodule.pp (compiled policy)

# Step 4: Review the .te file BEFORE installing
cat mymodule.te

# Step 5: Install the module
sudo semodule -i mymodule.pp

# Step 6: Verify
sudo semodule -l | grep mymodule
```

### Manual policy module creation

```bash
# Step 1: Write type enforcement file
cat > myapp.te << 'EOF'
policy_module(myapp, 1.0)

require {
    type httpd_t;
    type myapp_data_t;
    class file { read getattr open };
    class dir { search getattr };
}

# Allow httpd to read myapp data
allow httpd_t myapp_data_t:file { read getattr open };
allow httpd_t myapp_data_t:dir { search getattr };
EOF

# Step 2: Compile
checkmodule -M -m -o myapp.mod myapp.te

# Step 3: Package
semodule_package -o myapp.pp -m myapp.mod

# Step 4: Install
sudo semodule -i myapp.pp
```

### Define new types and transitions

```bash
cat > myapp-types.te << 'EOF'
policy_module(myapp_types, 1.0)

require {
    type httpd_t;
    type var_t;
}

# Define new file type
type myapp_data_t;
files_type(myapp_data_t)

# Define new domain
type myapp_t;
type myapp_exec_t;
init_daemon_domain(myapp_t, myapp_exec_t)

# File context for data directory
# (also add via semanage fcontext)

# Allow rules
allow myapp_t myapp_data_t:file { read write create getattr open };
allow myapp_t myapp_data_t:dir { read write add_name remove_name search };

# Allow network
corenet_tcp_bind_generic_port(myapp_t)
EOF
```

### Manage modules

```bash
sudo semodule -l                             # list all modules
sudo semodule -l | grep myapp                # find specific module
sudo semodule -d myapp                       # disable module
sudo semodule -e myapp                       # enable module
sudo semodule -r myapp                       # remove module
sudo semodule -i myapp.pp                    # install/update module
```

## MCS (Multi-Category Security)

### Categories in targeted policy

```bash
# MCS uses categories (c0-c1023) for container/VM isolation
# Format: s0:c1,c2 (sensitivity s0, categories c1 and c2)

# View categories
seinfo -c                                    # list defined categories

# svirt (container/VM isolation)
# Each container/VM gets unique category pair:
#   container 1: s0:c100,c200
#   container 2: s0:c300,c400
# Processes in container 1 cannot access files of container 2
```

### MCS with containers (sVirt)

```bash
# Docker/Podman automatically assign MCS labels
# View container process label
ps -eZ | grep container
# system_u:system_r:container_t:s0:c123,c456

# View container file labels
ls -Z /var/lib/containers/storage/
# Each container's files labeled with matching categories

# Run container with specific MCS label
podman run --security-opt label=level:s0:c100,c200 nginx

# Disable SELinux for a container
podman run --security-opt label=disable nginx
```

## MLS (Multi-Level Security)

### Sensitivity levels

```bash
# MLS format: s0-s15 (16 sensitivity levels)
# s0 = unclassified, s1 = confidential, s2 = secret, s3 = top secret

# MLS range: sLow-sHigh (user can access from sLow to sHigh)
# Example: s0-s2 means user can access unclassified through secret

# Change process level
runcon -l s1 /usr/bin/myapp                  # run at sensitivity s1

# Change file level
chcon -l s1 /path/to/file                    # set file sensitivity

# MLS policy enforces Bell-LaPadula:
#   No read up:    process at s1 cannot read file at s2
#   No write down: process at s2 cannot write file at s1
```

## Confined vs Unconfined Users

### Check confinement

```bash
# Check if current user is confined
id -Z
# unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023  ← UNCONFINED

# Check a specific user's mapping
semanage login -l
```

### Confine a user

```bash
# Map user to confined SELinux user
sudo semanage login -a -s user_u restricteduser
# user_u cannot:
#   - Run sudo / su
#   - Execute in /tmp or ~/
#   - Start network services

# Verify confinement
sudo -u restricteduser id -Z
# user_u:user_r:user_t:s0

# More restrictive: guest_u
sudo semanage login -a -s guest_u guestuser
# guest_u cannot:
#   - Use network at all
#   - Run su/sudo
#   - Execute in home/tmp

# Remove mapping (revert to default)
sudo semanage login -d restricteduser
```

### Confined user capabilities

```
SELinux User    su/sudo    Network    Execute in ~    GUI    X11
────────────────────────────────────────────────────────────────
unconfined_u    yes        yes        yes            yes    yes
sysadm_u        yes        yes        yes            yes    yes
staff_u         sudo only  yes        yes            yes    yes
user_u          no         yes        no             yes    yes
xguest_u        no         limited    no             yes    browser only
guest_u         no         no         no             no     no
```

## SELinux Troubleshooting

### Find denials

```bash
# Recent AVC denials
sudo ausearch -m avc -ts recent
sudo ausearch -m avc -ts today

# Denials for specific command
sudo ausearch -m avc -c httpd
sudo ausearch -m avc -c nginx

# Denials for specific type
sudo ausearch -m avc | grep httpd_t
```

### Analyze denials

```bash
# Human-readable analysis (requires setroubleshoot)
sudo sealert -a /var/log/audit/audit.log     # analyze full audit log
sudo sealert -l <alert-id>                   # specific alert details

# Why was access denied?
sudo ausearch -m avc -ts recent | audit2why

# Possible outputs:
#   Boolean: suggest setsebool
#   File context: suggest restorecon
#   Policy: suggest custom policy module
```

### Troubleshooting workflow

```bash
# 1. Check if SELinux is the issue
sudo setenforce 0                            # switch to permissive
# Test the application
# If it works → SELinux was blocking

# 2. Find the denial
sudo ausearch -m avc -ts recent

# 3. Analyze
sudo ausearch -m avc -ts recent | audit2why

# 4. Fix (in order of preference)
#    a. Fix file context
sudo restorecon -Rv /path/to/files

#    b. Enable appropriate boolean
sudo setsebool -P httpd_can_network_connect on

#    c. Add port context
sudo semanage port -a -t http_port_t -p tcp 8080

#    d. Create custom policy module (last resort)
sudo ausearch -m avc -ts recent | audit2allow -M fix
sudo semodule -i fix.pp

# 5. Re-enable enforcing
sudo setenforce 1

# 6. Test again
```

### Audit log format

```bash
# AVC denial message anatomy:
# type=AVC msg=audit(1712345678.123:456): avc:  denied  { read }
#   for  pid=1234 comm="httpd" name="index.html"
#   dev="sda1" ino=789
#   scontext=system_u:system_r:httpd_t:s0
#   tcontext=system_u:object_r:user_home_t:s0
#   tclass=file permissive=0
#
# Key fields:
#   denied { read }     — what permission was denied
#   comm="httpd"        — the process name
#   scontext=...httpd_t — source (process) context
#   tcontext=...user_home_t — target (file) context
#   tclass=file         — object class
#   permissive=0        — enforcing mode (1 = would have allowed)
```

### Relabeling

```bash
# Full system relabel (after policy change or mode change)
sudo fixfiles -F onboot                      # schedule relabel on next boot
sudo touch /.autorelabel && sudo reboot      # alternative method

# Relabel specific directory
sudo restorecon -Rv /var/www/

# Check for mislabeled files
sudo fixfiles check                          # show what needs relabeling
```

## See Also

- selinux
- apparmor
- capabilities
- auditd
- hardening-linux
- polkit

## References

- Red Hat SELinux User's and Administrator's Guide (RHEL 9)
- SELinux Project Wiki: https://selinuxproject.org/
- man semanage(8), man restorecon(8), man audit2allow(1)
- man seinfo(1), man sesearch(1), man sealert(8)
- Dan Walsh Blog: https://danwalsh.livejournal.com/
- SELinux Notebook: https://github.com/SELinuxProject/selinux-notebook
