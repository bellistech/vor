# Chef (Configuration Management)

Ruby-based configuration management tool using cookbooks, recipes, and a client/server model.

## Knife (CLI)

### Bootstrap a node

```bash
knife bootstrap 192.168.1.10 -U deploy -N web1 --sudo --run-list 'role[webserver]'
```

### Node management

```bash
knife node list
knife node show web1
knife node edit web1
knife node run_list add web1 'recipe[nginx]'
knife node run_list remove web1 'recipe[nginx]'
knife node delete web1 -y
```

### Search

```bash
knife search node 'role:webserver'
knife search node 'platform:ubuntu AND role:webserver'
knife search node '*:*' -a ipaddress     # show just IP
```

### Upload cookbook

```bash
knife cookbook upload nginx
knife cookbook upload nginx --freeze       # prevent overwrites
knife cookbook list
knife cookbook show nginx
knife cookbook delete nginx 1.0.0 -y
```

### SSH via knife

```bash
knife ssh 'role:webserver' 'sudo systemctl restart nginx' -x deploy
```

## Cookbooks

### Generate a cookbook

```bash
chef generate cookbook cookbooks/nginx
```

### Cookbook directory structure

```bash
# cookbooks/nginx/
#   metadata.rb          # name, version, depends
#   recipes/
#     default.rb         # main recipe
#     config.rb
#   templates/
#     nginx.conf.erb
#   files/
#     default.conf
#   attributes/
#     default.rb
#   libraries/
#   resources/
#   spec/
#   test/
```

### metadata.rb

```bash
# name 'nginx'
# version '1.2.0'
# depends 'apt'
# supports 'ubuntu'
```

## Recipes

### Install and configure nginx

```bash
# package 'nginx' do
#   action :install
# end
#
# template '/etc/nginx/nginx.conf' do
#   source 'nginx.conf.erb'
#   owner 'root'
#   group 'root'
#   mode '0644'
#   variables(worker_processes: node['nginx']['workers'])
#   notifies :restart, 'service[nginx]'
# end
#
# service 'nginx' do
#   action [:enable, :start]
# end
```

## Resources

### Common resources

```bash
# package 'htop'                          # install package
# package 'telnet' do action :remove end  # remove package

# file '/tmp/hello.txt' do
#   content 'Hello, world!'
#   mode '0644'
# end

# directory '/opt/app' do
#   owner 'deploy'
#   mode '0755'
#   recursive true
# end

# user 'deploy' do
#   shell '/bin/bash'
#   home '/home/deploy'
#   manage_home true
# end

# execute 'apt-get update' do
#   action :nothing                       # only runs when notified
# end

# cron 'backup' do
#   minute '0'
#   hour '2'
#   command '/usr/local/bin/backup.sh'
#   user 'root'
# end

# cookbook_file '/etc/app/config.yml' do
#   source 'config.yml'
#   owner 'deploy'
#   mode '0644'
# end
```

## Attributes

### Default attributes (attributes/default.rb)

```bash
# default['nginx']['workers'] = 4
# default['nginx']['port'] = 80
# default['app']['version'] = '2.1.0'
```

### Attribute precedence (lowest to highest)

```bash
# default -> normal -> override -> automatic (ohai)
```

### Use in recipes

```bash
# node['nginx']['port']
# node['platform']        # from ohai
```

## Roles

### Create a role (JSON)

```bash
# {
#   "name": "webserver",
#   "run_list": [
#     "recipe[base]",
#     "recipe[nginx]",
#     "recipe[app]"
#   ],
#   "default_attributes": {
#     "nginx": { "workers": 8 }
#   }
# }
```

### Upload a role

```bash
knife role from file roles/webserver.json
knife role list
knife role show webserver
```

## Data Bags

### Create a data bag

```bash
knife data bag create users
```

### Create an item

```bash
knife data bag from file users users/deploy.json
```

### Encrypted data bags

```bash
knife data bag create secrets db_creds --secret-file /etc/chef/encrypted_data_bag_secret
knife data bag show secrets db_creds --secret-file /etc/chef/encrypted_data_bag_secret
```

### Use in recipes

```bash
# deploy_user = data_bag_item('users', 'deploy')
# deploy_user['password']
```

## Environments

### Create an environment

```bash
# {
#   "name": "production",
#   "cookbook_versions": {
#     "nginx": "= 1.2.0"
#   },
#   "default_attributes": {
#     "app": { "env": "production" }
#   }
# }
```

```bash
knife environment from file environments/production.json
knife environment list
knife node environment_set web1 production
```

## Test Kitchen

### Initialize

```bash
kitchen init
```

### .kitchen.yml

```bash
# driver:
#   name: vagrant
# provisioner:
#   name: chef_zero
# platforms:
#   - name: ubuntu-22.04
# suites:
#   - name: default
#     run_list:
#       - recipe[nginx::default]
```

### Workflow

```bash
kitchen create        # create VM
kitchen converge      # apply recipes
kitchen verify        # run tests
kitchen test          # full cycle: create -> converge -> verify -> destroy
kitchen destroy       # tear down
kitchen login         # SSH into the instance
kitchen list          # show status
```

## Tips

- `chef-client -W` runs in why-run mode (dry run).
- Use `knife cookbook upload --freeze` in production to prevent accidental overwrites.
- `berks install && berks upload` manages cookbook dependencies via Berkshelf.
- `notifies :restart` is lazy (end of run) by default. Use `notifies :restart, 'service[x]', :immediately` for instant restart.
- Ohai auto-detects system attributes (platform, IP, memory). Access them via `node['platform']`.
- Test Kitchen with Docker (`kitchen-docker`) is faster than Vagrant for testing.
- `chef-client -o 'recipe[nginx]'` overrides the run list for a single run.
- Data bag secrets should never be committed to version control.
