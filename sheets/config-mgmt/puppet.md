# Puppet (Configuration Management)

Model-driven configuration management tool using a declarative DSL and agent/server architecture.

## Agent & Apply

### Run puppet agent

```bash
puppet agent -t                    # test run (verbose, one-time)
puppet agent -t --noop             # dry run, no changes
puppet agent -t --environment staging
puppet agent --enable              # re-enable after disable
puppet agent --disable "maintenance window"
```

### Apply a manifest directly

```bash
puppet apply site.pp
puppet apply -e 'package { "nginx": ensure => present }'
puppet apply site.pp --noop        # dry run
```

## Resource Types

### Package

```bash
# package { 'nginx':
#   ensure => installed,
# }
# package { 'htop':
#   ensure => '3.2.1',   # specific version
# }
# package { 'telnet':
#   ensure => absent,
# }
```

### Service

```bash
# service { 'nginx':
#   ensure => running,
#   enable => true,
#   require => Package['nginx'],
# }
```

### File

```bash
# file { '/etc/nginx/nginx.conf':
#   ensure  => file,
#   owner   => 'root',
#   group   => 'root',
#   mode    => '0644',
#   source  => 'puppet:///modules/nginx/nginx.conf',
#   notify  => Service['nginx'],
# }
# file { '/opt/app':
#   ensure => directory,
#   owner  => 'deploy',
#   mode   => '0755',
# }
```

### User and Group

```bash
# user { 'deploy':
#   ensure     => present,
#   shell      => '/bin/bash',
#   home       => '/home/deploy',
#   managehome => true,
#   groups     => ['sudo'],
# }
```

### Exec

```bash
# exec { 'apt-update':
#   command     => '/usr/bin/apt-get update',
#   refreshonly => true,   # only runs when notified
# }
```

### Cron

```bash
# cron { 'db-backup':
#   command => '/usr/local/bin/backup.sh',
#   user    => 'root',
#   hour    => 2,
#   minute  => 0,
# }
```

## Manifests & Classes

### Define a class

```bash
# class nginx (
#   Integer $worker_processes = 4,
#   String  $log_dir = '/var/log/nginx',
# ) {
#   package { 'nginx': ensure => installed }
#   file { '/etc/nginx/nginx.conf':
#     content => template('nginx/nginx.conf.erb'),
#     notify  => Service['nginx'],
#   }
#   service { 'nginx':
#     ensure => running,
#     enable => true,
#   }
# }
```

### Include a class

```bash
# include nginx
# class { 'nginx':
#   worker_processes => 8,
# }
```

## Modules

### Module directory structure

```bash
# modules/nginx/
#   manifests/
#     init.pp          # class nginx
#     config.pp        # class nginx::config
#   templates/
#     nginx.conf.erb
#   files/
#     default.conf
#   lib/
#     facter/          # custom facts
```

### Install a module from Forge

```bash
puppet module install puppetlabs-apache
puppet module list
puppet module uninstall puppetlabs-apache
```

## Node Definitions

### site.pp

```bash
# node 'web1.example.com' {
#   include nginx
#   include app
# }
# node /^db\d+\.example\.com$/ {
#   include postgresql
# }
# node default {
#   include base
# }
```

## Hiera

### hiera.yaml

```bash
# ---
# version: 5
# hierarchy:
#   - name: "Per-node"
#     path: "nodes/%{trusted.certname}.yaml"
#   - name: "Per-OS"
#     path: "os/%{facts.os.family}.yaml"
#   - name: "Common"
#     path: "common.yaml"
```

### Hiera data file (common.yaml)

```bash
# nginx::worker_processes: 4
# nginx::log_dir: /var/log/nginx
```

### Lookup in manifests

```bash
# $port = lookup('app::port', Integer, 'first', 8080)
```

## Facter

### List all facts

```bash
facter
facter os.family
facter networking.ip
```

### Use facts in manifests

```bash
# if $facts['os']['family'] == 'Debian' {
#   package { 'apt-transport-https': ensure => installed }
# }
```

## Resource Inspection

### List resource types

```bash
puppet describe --list
puppet describe file          # show all attributes
```

### Query current state

```bash
puppet resource user deploy
puppet resource package nginx
puppet resource service nginx
```

## Tips

- `puppet agent -t --noop` is your best friend before any production change.
- `puppet resource <type> <name>` inspects the live state of any resource, even without manifests.
- Use `notify =>` for forward references and `subscribe =>` for reverse references. Both trigger refresh events.
- Hiera automatic parameter lookup: a class parameter `nginx::port` is automatically looked up as `nginx::port` in Hiera.
- Ordering: Puppet does not apply resources in file order by default. Use `require`, `before`, `notify`, and `subscribe` to set dependencies.
- `--environment` lets you test changes in a separate code branch before promoting to production.
- `r10k` or Code Manager automates deploying Puppet code from Git branches to environments.
- `puppet parser validate manifest.pp` checks syntax without applying.

## References

- [Puppet Documentation](https://www.puppet.com/docs/puppet/)
- [Puppet Language Reference](https://www.puppet.com/docs/puppet/latest/lang_summary.html)
- [Puppet Resource Type Reference](https://www.puppet.com/docs/puppet/latest/type.html)
- [Puppet Built-in Functions](https://www.puppet.com/docs/puppet/latest/function.html)
- [Puppet Module Fundamentals](https://www.puppet.com/docs/puppet/latest/modules_fundamentals.html)
- [Hiera — Data Lookup](https://www.puppet.com/docs/puppet/latest/hiera_intro.html)
- [Puppet Forge — Module Registry](https://forge.puppet.com/)
- [Facter — System Inventory](https://www.puppet.com/docs/puppet/latest/facter.html)
- [r10k — Code Management](https://github.com/puppetlabs/r10k)
- [Puppet GitHub Repository](https://github.com/puppetlabs/puppet)
- [puppet-agent(8) man page](https://www.puppet.com/docs/puppet/latest/man/agent.html)
