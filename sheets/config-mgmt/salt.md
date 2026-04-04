# Salt (Configuration Management & Remote Execution)

Event-driven automation platform using a master/minion architecture with YAML states.

## Master & Minion

### Start master and minion

```bash
systemctl start salt-master
systemctl start salt-minion
```

### Accept minion keys

```bash
salt-key -L               # list all keys
salt-key -A               # accept all pending
salt-key -a web1          # accept one minion
salt-key -d web1          # delete a key
salt-key -r web1          # reject a key
```

### Test connectivity

```bash
salt '*' test.ping
salt 'web1' test.version
```

## Targeting

### By minion ID

```bash
salt 'web1' cmd.run 'uptime'
salt 'web[1-3]' cmd.run 'uptime'    # glob
```

### By grain

```bash
salt -G 'os:Ubuntu' cmd.run 'lsb_release -a'
salt -G 'roles:webserver' state.apply
```

### By pillar

```bash
salt -I 'env:production' cmd.run 'hostname'
```

### Compound targeting

```bash
salt -C 'G@os:Ubuntu and web*' test.ping
```

### By list

```bash
salt -L 'web1,web2,db1' test.ping
```

## Remote Execution

### Run a shell command

```bash
salt '*' cmd.run 'df -h'
```

### Install a package

```bash
salt '*' pkg.install nginx
salt '*' pkg.version nginx
```

### Manage services

```bash
salt '*' service.start nginx
salt '*' service.restart nginx
salt '*' service.status nginx
```

### File operations

```bash
salt '*' file.read /etc/hostname
salt '*' file.write /tmp/test "hello world"
salt '*' file.file_exists /etc/nginx/nginx.conf
```

### User management

```bash
salt '*' user.add deploy shell=/bin/bash
salt '*' user.info deploy
```

### Check disk and memory

```bash
salt '*' disk.usage
salt '*' status.meminfo
```

## States

### Apply all states (highstate)

```bash
salt '*' state.apply              # same as state.highstate
salt 'web1' state.apply           # single minion
```

### Apply a specific state

```bash
salt '*' state.apply nginx
salt '*' state.apply nginx test=True   # dry run
```

### State file example (nginx.sls)

```bash
# nginx:
#   pkg.installed: []
#   service.running:
#     - enable: True
#     - require:
#       - pkg: nginx
#
# /etc/nginx/nginx.conf:
#   file.managed:
#     - source: salt://nginx/files/nginx.conf
#     - user: root
#     - group: root
#     - mode: 644
#     - watch_in:
#       - service: nginx
```

## Top File

### top.sls (state assignment)

```bash
# base:
#   '*':
#     - common
#   'web*':
#     - nginx
#     - app
#   'db*':
#     - postgresql
```

## Pillar

### Define pillar data

```bash
# /srv/pillar/top.sls
# base:
#   '*':
#     - common
#   'web*':
#     - nginx

# /srv/pillar/nginx.sls
# nginx:
#   worker_processes: 4
#   port: 80
```

### Access pillar in states

```bash
# {{ pillar['nginx']['port'] }}
# {{ pillar.get('nginx:port', 80) }}
```

### Refresh pillar data

```bash
salt '*' saltutil.refresh_pillar
salt '*' pillar.items
salt '*' pillar.get nginx:port
```

## Grains

### List all grains

```bash
salt '*' grains.items
salt '*' grains.get os
salt '*' grains.get ip_interfaces
```

### Set a custom grain

```bash
salt 'web1' grains.setval role webserver
```

### Use grains in states

```bash
# {% if grains['os'] == 'Ubuntu' %}
# ...
# {% endif %}
```

## Common State Modules

### pkg / service / file / user / cron

```bash
# pkg.installed, pkg.removed, pkg.latest
# service.running, service.dead, service.enabled
# file.managed, file.directory, file.absent, file.symlink
# user.present, user.absent
# cron.present, cron.absent
```

## Tips

- Use `state.apply test=True` to dry-run before applying changes.
- `salt-run manage.status` shows which minions are up or down.
- Pillar data is per-minion and encrypted in transit. Use it for secrets.
- Grains are facts about the minion (OS, IPs, CPU). Custom grains survive reboots.
- `salt '*' sys.doc pkg.install` shows documentation for any module inline.
- `salt-call --local state.apply nginx` runs states without a master (masterless mode).
- Requisites matter: `require` ensures ordering, `watch` triggers restarts on change, `onchanges` runs only when something changed.
- Salt SSH (`salt-ssh`) works without a minion agent, similar to Ansible.

## See Also

- ansible
- puppet
- chef
- terraform
- ssh

## References

- [Salt Documentation](https://docs.saltproject.io/)
- [Salt Module Reference](https://docs.saltproject.io/en/latest/ref/modules/all/index.html)
- [Salt State Reference](https://docs.saltproject.io/en/latest/ref/states/all/index.html)
- [Salt Pillar Documentation](https://docs.saltproject.io/en/latest/topics/pillar/index.html)
- [Salt Grains Reference](https://docs.saltproject.io/en/latest/topics/grains/index.html)
- [Salt Targeting (Glob, Regex, Compound)](https://docs.saltproject.io/en/latest/topics/targeting/index.html)
- [Salt Formulas](https://docs.saltproject.io/en/latest/topics/development/conventions/formulas.html)
- [Salt Orchestration Runner](https://docs.saltproject.io/en/latest/topics/orchestrate/orchestrate_runner.html)
- [Salt GitHub Repository](https://github.com/saltstack/salt)
- [Salt Best Practices](https://docs.saltproject.io/en/latest/topics/best_practices.html)
