# LXD (System Container & VM Manager)

Run full Linux system containers and VMs with LXD, managed via the `lxc` client CLI.

## Launching Instances

### Containers

```bash
lxc launch ubuntu:24.04 webserver
lxc launch images:debian/12 dbhost
lxc launch ubuntu:24.04 myvm --vm          # launch as VM instead of container
lxc launch ubuntu:24.04 web -c limits.cpu=2 -c limits.memory=1GB
lxc launch ubuntu:24.04 web -p default -p webserver   # apply profiles
```

### From snapshot or image

```bash
lxc copy webserver/snap0 webserver-clone    # clone from snapshot
lxc init ubuntu:24.04 staging              # create without starting
lxc start staging
```

## Listing and Info

```bash
lxc list                                   # all instances
lxc list --format csv                      # scriptable output
lxc list --columns ns4tS                   # name, state, ipv4, type, snapshots
lxc list running                           # filter by state
lxc info webserver                         # detailed instance info
lxc info webserver --show-log              # show console log
```

## Executing Commands

```bash
lxc exec webserver -- bash
lxc exec webserver -- apt update
lxc exec webserver --env MYVAR=hello -- printenv MYVAR
lxc exec webserver -- su - ubuntu          # switch to non-root user
```

## File Transfer

```bash
lxc file push local.conf webserver/etc/myapp/config.conf
lxc file pull webserver/var/log/syslog ./syslog.txt
lxc file edit webserver/etc/hosts          # edit in place
```

## Configuration

### Instance config

```bash
lxc config set webserver limits.cpu 4
lxc config set webserver limits.memory 2GB
lxc config set webserver security.privileged true
lxc config show webserver
lxc config edit webserver                  # open full config in editor
lxc config set webserver boot.autostart true
```

### Cloud-init

```bash
lxc config set webserver cloud-init.user-data - < cloud-init.yaml
```

### Device management

```bash
lxc config device add webserver myport proxy listen=tcp:0.0.0.0:80 connect=tcp:127.0.0.1:80
lxc config device add webserver shared disk source=/opt/share path=/mnt/share
lxc config device remove webserver myport
lxc config device list webserver
```

## Profiles

```bash
lxc profile list
lxc profile show default
lxc profile create webserver
lxc profile edit webserver                 # opens in editor
lxc profile set webserver limits.cpu 2
lxc profile add webserver security.nesting true
lxc profile assign mycontainer default,webserver
lxc profile copy default custom-default
```

## Storage

### Storage pools

```bash
lxc storage list
lxc storage create fast zfs source=/dev/sdb
lxc storage create mypool dir
lxc storage info default
lxc storage show default
```

### Storage volumes

```bash
lxc storage volume create default data
lxc storage volume attach default data webserver /mnt/data
lxc storage volume detach default data webserver
lxc storage volume list default
lxc storage volume delete default data
```

## Networking

```bash
lxc network list
lxc network create mybridge
lxc network show lxdbr0
lxc network set lxdbr0 ipv4.address 10.10.10.1/24
lxc network attach mybridge webserver eth1
lxc network info lxdbr0                    # show connected instances
```

### Port forwarding (proxy device)

```bash
lxc config device add webserver http proxy listen=tcp:0.0.0.0:80 connect=tcp:127.0.0.1:80
lxc config device add webserver https proxy listen=tcp:0.0.0.0:443 connect=tcp:127.0.0.1:443
```

## Snapshots

```bash
lxc snapshot webserver snap0
lxc snapshot webserver snap-before-upgrade
lxc info webserver                         # lists snapshots
lxc restore webserver snap0
lxc delete webserver/snap0
lxc copy webserver/snap0 new-instance      # create instance from snapshot
lxc snapshot webserver --stateful          # include memory state
```

## Migration

### Between LXD hosts

```bash
lxc remote add prod 10.0.0.5              # add remote LXD server
lxc remote list
lxc copy webserver prod:webserver          # copy to remote
lxc move webserver prod:webserver          # live migrate
lxc copy prod:webserver local:webserver-backup
```

### Export/import

```bash
lxc export webserver webserver-backup.tar.gz
lxc import webserver-backup.tar.gz restored-server
lxc publish webserver --alias my-golden-image   # create image from instance
```

## Instance Lifecycle

```bash
lxc start webserver
lxc stop webserver
lxc stop webserver --force                 # hard stop
lxc restart webserver
lxc pause webserver                        # freeze processes
lxc delete webserver
lxc delete webserver --force               # delete running instance
```

## Tips

- LXD containers are system containers (full init, multiple processes) not app containers like Docker.
- `lxc` is the client, `lxd` is the daemon. Run `lxd init` once on a fresh install to configure storage, networking, and clustering.
- Use profiles to avoid repeating config across instances; stack them with `lxc profile assign inst default,custom`.
- Proxy devices are how you expose container ports to the host -- there is no `-p` flag like Docker.
- `security.nesting=true` is required to run Docker inside an LXD container.
- Snapshots are cheap on ZFS/btrfs (copy-on-write); less so on dir backend.
- For VMs, add `--vm` to launch. VMs support non-Linux guests and hardware passthrough.
- `lxc exec` runs as root by default; use `-- su - username` for non-root shells.

## References

- [LXD Documentation](https://documentation.ubuntu.com/lxd/)
- [LXD Getting Started Guide](https://documentation.ubuntu.com/lxd/en/latest/tutorial/first_steps/)
- [LXD Instance Configuration](https://documentation.ubuntu.com/lxd/en/latest/reference/instance_options/)
- [LXD Networking](https://documentation.ubuntu.com/lxd/en/latest/explanation/networks/)
- [LXD Storage](https://documentation.ubuntu.com/lxd/en/latest/explanation/storage/)
- [LXD Profiles](https://documentation.ubuntu.com/lxd/en/latest/explanation/profiles/)
- [LXD CLI Reference](https://documentation.ubuntu.com/lxd/en/latest/reference/manpages/)
- [LXD Image Server](https://images.linuxcontainers.org/)
- [LXD GitHub Repository](https://github.com/canonical/lxd)
- [LXC/LXD Linux Containers](https://linuxcontainers.org/)
- [Incus — Community Fork of LXD](https://github.com/lxc/incus)
