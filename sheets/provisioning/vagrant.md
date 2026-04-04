# Vagrant (Development Environment Manager)

> Define, launch, and manage reproducible virtual machines using a declarative Vagrantfile and provider backends.

## Concepts

### Components

```
# Vagrantfile  — Ruby DSL config declaring VMs, networking, provisioners
# Box          — base image (pre-built VM template from Vagrant Cloud or local)
# Provider     — virtualization backend (VirtualBox, libvirt, Docker, Hyper-V)
# Provisioner  — tool that configures the VM after boot (shell, Ansible, Puppet)
# Synced folder — shared directory between host and guest
```

## CLI Commands

### Lifecycle

```bash
# Create a Vagrantfile from a box template
vagrant init ubuntu/jammy64

# Start and provision the VM
vagrant up
vagrant up --provider=libvirt            # specify provider
vagrant up --no-provision                # skip provisioners

# SSH into the VM
vagrant ssh
vagrant ssh worker-1                     # multi-machine: specify name

# Reload VM (reboot + re-read Vagrantfile)
vagrant reload
vagrant reload --provision               # also re-run provisioners

# Run provisioners on a running VM
vagrant provision

# Stop the VM (graceful shutdown)
vagrant halt

# Destroy the VM (delete disk and metadata)
vagrant destroy -f                       # skip confirmation

# Show VM status
vagrant status                           # current project
vagrant global-status                    # all Vagrant VMs on the system
vagrant global-status --prune            # clean stale entries
```

### Snapshots

```bash
# Save current state
vagrant snapshot save baseline

# List snapshots
vagrant snapshot list

# Restore a snapshot
vagrant snapshot restore baseline

# Delete a snapshot
vagrant snapshot delete baseline

# Quick push/pop (unnamed stack)
vagrant snapshot push
vagrant snapshot pop
```

### Box Management

```bash
# Add a box
vagrant box add ubuntu/jammy64
vagrant box add --name mybox ./custom.box   # local box file

# List installed boxes
vagrant box list

# Update a box
vagrant box update

# Remove old box versions
vagrant box prune
vagrant box remove ubuntu/jammy64 --box-version 20230101.0.0
```

## Vagrantfile

### Basic Configuration

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"
  config.vm.hostname = "devbox"

  # Disable default synced folder
  config.vm.synced_folder ".", "/vagrant", disabled: true

  # Custom synced folder
  config.vm.synced_folder "./src", "/opt/app",
    owner: "vagrant", group: "vagrant",
    type: "rsync",                       # or "nfs", "virtualbox", "smb"
    rsync__exclude: [".git/", "node_modules/"]

  # Provider-specific overrides
  config.vm.provider "virtualbox" do |vb|
    vb.memory = 2048
    vb.cpus   = 2
    vb.name   = "my-devbox"
    vb.gui    = false
  end

  config.vm.provider "libvirt" do |lv|
    lv.memory = 2048
    lv.cpus   = 2
    lv.driver = "kvm"
    lv.disk_bus = "virtio"
  end
end
```

### Networking

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"

  # Port forwarding (host:guest)
  config.vm.network "forwarded_port", guest: 80, host: 8080
  config.vm.network "forwarded_port", guest: 443, host: 8443,
    auto_correct: true                   # pick next available if busy

  # Private network (host-only) with static IP
  config.vm.network "private_network", ip: "192.168.56.10"

  # Private network with DHCP
  config.vm.network "private_network", type: "dhcp"

  # Public network (bridged) — VM gets IP from external DHCP
  config.vm.network "public_network", bridge: "en0: Wi-Fi"
end
```

### Provisioners

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"

  # Inline shell
  config.vm.provision "shell", inline: <<-SHELL
    apt-get update
    apt-get install -y nginx
    systemctl enable --now nginx
  SHELL

  # External shell script
  config.vm.provision "shell", path: "scripts/setup.sh",
    privileged: true,                    # run as root (default)
    env: { "APP_ENV" => "development" }

  # File provisioner
  config.vm.provision "file",
    source: "configs/app.conf",
    destination: "/tmp/app.conf"

  # Ansible provisioner (runs from host)
  config.vm.provision "ansible" do |ansible|
    ansible.playbook       = "ansible/playbook.yml"
    ansible.inventory_path = "ansible/inventory"
    ansible.verbose        = "v"
  end

  # Ansible local (runs inside the guest)
  config.vm.provision "ansible_local" do |ansible|
    ansible.playbook = "playbook.yml"
  end
end
```

### Multi-Machine

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"

  config.vm.define "controller" do |c|
    c.vm.hostname = "controller"
    c.vm.network "private_network", ip: "192.168.56.10"
    c.vm.provider "virtualbox" do |vb|
      vb.memory = 2048
    end
    c.vm.provision "shell", inline: "echo 'I am the controller'"
  end

  config.vm.define "worker-1" do |w|
    w.vm.hostname = "worker-1"
    w.vm.network "private_network", ip: "192.168.56.11"
    w.vm.provision "shell", inline: "echo 'I am worker-1'"
  end

  config.vm.define "worker-2" do |w|
    w.vm.hostname = "worker-2"
    w.vm.network "private_network", ip: "192.168.56.12"
    w.vm.provision "shell", inline: "echo 'I am worker-2'"
  end
end
```

### Loop-Based Multi-Machine

```ruby
Vagrant.configure("2") do |config|
  config.vm.box = "ubuntu/jammy64"

  (1..3).each do |i|
    config.vm.define "node-#{i}" do |node|
      node.vm.hostname = "node-#{i}"
      node.vm.network "private_network", ip: "192.168.56.#{10 + i}"
      node.vm.provider "virtualbox" do |vb|
        vb.memory = 1024
        vb.cpus   = 1
      end
    end
  end
end
```

## Plugins

### Common Plugins

```bash
# Install a plugin
vagrant plugin install vagrant-vbguest       # auto-install VBox guest additions
vagrant plugin install vagrant-libvirt       # KVM/libvirt provider
vagrant plugin install vagrant-disksize      # resize VM disks
vagrant plugin install vagrant-hostmanager   # manage /etc/hosts

# List installed plugins
vagrant plugin list

# Update plugins
vagrant plugin update

# Uninstall
vagrant plugin uninstall vagrant-vbguest
```

### Plugin Usage in Vagrantfile

```ruby
# vagrant-hostmanager
if Vagrant.has_plugin?("vagrant-hostmanager")
  config.hostmanager.enabled     = true
  config.hostmanager.manage_host = true    # update host machine's /etc/hosts
end

# vagrant-disksize
config.disksize.size = "50GB"
```

## Tips

- Use `vagrant ssh-config` to get SSH connection details for use with `ssh` or `scp` directly.
- Set `VAGRANT_DEFAULT_PROVIDER=libvirt` to avoid typing `--provider` every time.
- Use `config.vm.provision "shell", run: "always"` for provisioners that must run on every `vagrant up`.
- Provisioners run in the order they appear in the Vagrantfile; use `before` and `after` for ordering.
- Use `vagrant package --output mybox.box` to export a running VM as a reusable box.
- NFS synced folders are significantly faster than VirtualBox shared folders for large codebases.

## See Also

- packer
- ansible
- docker
- lxd
- terraform
- puppet

## References

- [Vagrant Documentation](https://developer.hashicorp.com/vagrant/docs)
- [Vagrantfile Reference](https://developer.hashicorp.com/vagrant/docs/vagrantfile)
- [Vagrant Cloud — Box Catalog](https://app.vagrantup.com/boxes/search)
- [Vagrant Provider Documentation](https://developer.hashicorp.com/vagrant/docs/providers)
- [Vagrant CLI Reference](https://developer.hashicorp.com/vagrant/docs/cli)
- [Vagrant Provisioners](https://developer.hashicorp.com/vagrant/docs/provisioning)
- [Vagrant Networking](https://developer.hashicorp.com/vagrant/docs/networking)
- [Vagrant Synced Folders](https://developer.hashicorp.com/vagrant/docs/synced-folders)
- [Vagrant Multi-Machine](https://developer.hashicorp.com/vagrant/docs/multi-machine)
- [Vagrant GitHub Repository](https://github.com/hashicorp/vagrant)
- [Vagrant Plugin Development](https://developer.hashicorp.com/vagrant/docs/plugins)
