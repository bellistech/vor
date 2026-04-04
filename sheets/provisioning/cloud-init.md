# cloud-init (Cloud Instance Initialization)

> Bootstrap cloud instances with users, packages, files, and network config via declarative YAML user-data.

## Concepts

### Data Sources

```
# cloud-init reads configuration from a datasource at boot:
#   EC2        — instance metadata at http://169.254.169.254/
#   GCE        — Google Compute Engine metadata server
#   Azure      — Azure IMDS + OVF
#   OpenStack  — config drive or metadata service
#   NoCloud    — local seed directory or ISO (for libvirt, LXD, manual VMs)
#   None       — fallback when no datasource found
```

### Boot Stages

```
# 1. Generator  — detect datasource, decide if cloud-init should run
# 2. Local      — apply network config (before network is up)
# 3. Network    — fetch remote user-data, run early modules
# 4. Config     — run config modules (packages, users, files)
# 5. Final      — run final modules (runcmd, scripts, phone_home)
```

### User-Data Formats

```
# cloud-config YAML — starts with #cloud-config
# Shell script      — starts with #!/bin/bash (or other shebang)
# MIME multipart    — combine multiple formats in one payload
# Gzip              — compressed user-data (auto-detected)
# Include file      — #include <url> to fetch remote config
```

## Cloud-Config Modules

### Users and Groups

```yaml
#cloud-config
users:
  - name: deploy
    groups: [sudo, docker]
    shell: /bin/bash
    sudo: ['ALL=(ALL) NOPASSWD:ALL']
    ssh_authorized_keys:
      - ssh-ed25519 AAAA... deploy@example.com
    lock_passwd: true                    # disable password login

  - name: appuser
    system: true                         # system account (no home by default)
    shell: /usr/sbin/nologin

groups:
  - monitoring
  - developers
```

### Package Installation

```yaml
#cloud-config
package_update: true                     # apt update / yum check-update
package_upgrade: true                    # upgrade all packages at boot
package_reboot_if_required: true         # reboot if kernel updated

packages:
  - nginx
  - postgresql-client
  - python3-pip
  - ['git', '1:2.39*']                  # version pinning
```

### Write Files

```yaml
#cloud-config
write_files:
  - path: /etc/nginx/conf.d/app.conf
    content: |
      server {
          listen 80;
          server_name app.example.com;
          location / {
              proxy_pass http://127.0.0.1:8080;
          }
      }
    owner: root:root
    permissions: '0644'

  - path: /opt/app/config.env
    encoding: b64
    content: QVBJX0tFWT1zZWNyZXQ=       # base64-encoded content
    permissions: '0600'
```

### Run Commands

```yaml
#cloud-config
# bootcmd runs early (every boot), before networking modules
bootcmd:
  - echo "net.core.somaxconn=1024" >> /etc/sysctl.conf
  - sysctl -p

# runcmd runs once on first boot (final stage)
runcmd:
  - systemctl enable --now nginx
  - curl -fsSL https://get.docker.com | sh
  - usermod -aG docker deploy
  - [sh, -c, 'echo "Setup complete at $(date)" > /var/log/cloud-init-done']
```

### SSH Keys and Config

```yaml
#cloud-config
ssh_deletekeys: true                     # regenerate host keys
ssh_genkeytypes: [ed25519, rsa]

# Disable password SSH globally
ssh_pwauth: false

# Import SSH keys from GitHub/Launchpad
ssh_import_id:
  - gh:username
```

### Timezone and Locale

```yaml
#cloud-config
timezone: America/New_York
locale: en_US.UTF-8

ntp:
  enabled: true
  servers:
    - 0.pool.ntp.org
    - 1.pool.ntp.org
```

### Disk Setup and Mounts

```yaml
#cloud-config
disk_setup:
  /dev/sdb:
    table_type: gpt
    layout: true

fs_setup:
  - device: /dev/sdb1
    filesystem: ext4
    label: data

mounts:
  - [/dev/sdb1, /mnt/data, ext4, "defaults,noatime", "0", "2"]

# Grow root partition to fill available space
growpart:
  mode: auto
  devices: ['/']
resize_rootfs: true
```

## Network Configuration

### Version 2 (Netplan-style)

```yaml
# Provided via network-config (separate from user-data)
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
      addresses:
        - 10.0.1.50/24
      routes:
        - to: 0.0.0.0/0
          via: 10.0.1.1
      nameservers:
        addresses: [8.8.8.8, 8.8.4.4]
        search: [example.com]
```

### NoCloud Datasource (Local VMs)

```bash
# Create a seed ISO for libvirt/QEMU VMs
mkdir -p /tmp/nocloud
cat > /tmp/nocloud/meta-data <<EOF
instance-id: vm-001
local-hostname: testvm
EOF

cat > /tmp/nocloud/user-data <<EOF
#cloud-config
users:
  - name: admin
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - ssh-ed25519 AAAA... admin@host
EOF

# Generate ISO
genisoimage -output seed.iso -volid cidata -joliet -rock \
    /tmp/nocloud/user-data /tmp/nocloud/meta-data
```

## Debugging

### Status and Logs

```bash
# Check cloud-init status
cloud-init status                        # done, running, error, disabled
cloud-init status --long                 # show stage details and errors

# Validate user-data syntax
cloud-init schema --config-file user-data.yaml

# Re-run cloud-init (for testing — forces fresh execution)
cloud-init clean --logs                  # remove state and logs
cloud-init init                          # re-run init stages
cloud-init modules --mode=final          # re-run final stage only

# Key log files
cat /var/log/cloud-init.log              # detailed module output
cat /var/log/cloud-init-output.log       # stdout/stderr from runcmd/scripts
cat /run/cloud-init/result.json          # final status summary

# Instance metadata (what cloud-init consumed)
cat /run/cloud-init/instance-data.json
cloud-init query userdata                # show consumed user-data
cloud-init query ds.meta_data            # show datasource metadata
```

## Tips

- User-data must start with exactly `#cloud-config` (no leading spaces) or a shebang.
- Use `cloud-init schema --config-file` to catch YAML errors before deploying.
- `runcmd` runs as root during final stage; use `su -c` or `sudo -u` for other users.
- `write_files` runs before `runcmd`, so files are available when commands execute.
- For multi-part configs, use MIME multipart to combine cloud-config + shell scripts.
- Set `final_message` to log a custom completion message with timestamps.

## See Also

- packer
- terraform
- ansible
- vagrant
- lxd

## References

- [cloud-init Official Documentation](https://cloudinit.readthedocs.io/)
- [cloud-init Module Reference](https://cloudinit.readthedocs.io/en/latest/reference/modules.html)
- [cloud-init Datasource Reference](https://cloudinit.readthedocs.io/en/latest/reference/datasources.html)
- [cloud-config Examples](https://cloudinit.readthedocs.io/en/latest/reference/examples.html)
- [cloud-init CLI Reference](https://cloudinit.readthedocs.io/en/latest/reference/cli.html)
- [cloud-init Network Configuration](https://cloudinit.readthedocs.io/en/latest/reference/network-config.html)
- [cloud-init GitHub Repository](https://github.com/canonical/cloud-init)
- [cloud-init on Ubuntu](https://documentation.ubuntu.com/cloud-init/)
- [cloud-init Instance Metadata](https://cloudinit.readthedocs.io/en/latest/explanation/instancedata.html)
- [cloud-init Debugging and Troubleshooting](https://cloudinit.readthedocs.io/en/latest/howto/debugging.html)
