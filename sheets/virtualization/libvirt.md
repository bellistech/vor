# libvirt (virtualization management)

Libvirt provides a unified API and toolset for managing virtual machines across hypervisors including KVM/QEMU, Xen, and LXC, offering domain lifecycle management, storage pool administration, virtual networking, and live migration through virsh CLI and XML-based configuration.

## Domain Lifecycle

### Basic Operations

```bash
# List running domains
virsh list

# List all domains (including stopped)
virsh list --all

# Start a domain
virsh start vm1

# Graceful shutdown (ACPI)
virsh shutdown vm1

# Force stop (like pulling power)
virsh destroy vm1

# Reboot
virsh reboot vm1

# Suspend to memory (pause)
virsh suspend vm1

# Resume from suspend
virsh resume vm1

# Save state to disk (hibernate)
virsh save vm1 /var/lib/libvirt/save/vm1.state

# Restore from saved state
virsh restore /var/lib/libvirt/save/vm1.state

# Autostart on host boot
virsh autostart vm1

# Disable autostart
virsh autostart --disable vm1

# Delete domain (keeps disk)
virsh undefine vm1

# Delete domain and storage
virsh undefine vm1 --remove-all-storage --nvram
```

### Domain Information

```bash
# Detailed domain info
virsh dominfo vm1

# Domain XML dump
virsh dumpxml vm1

# Domain block devices
virsh domblklist vm1

# Domain network interfaces
virsh domiflist vm1

# Domain CPU stats
virsh cpu-stats vm1

# Domain memory stats
virsh dommemstat vm1

# VNC/Spice display port
virsh domdisplay vm1

# Guest agent info (requires qemu-ga)
virsh guestinfo vm1

# Guest filesystem info
virsh domfsinfo vm1
```

## Domain XML Configuration

### Editing and Applying XML

```bash
# Edit domain XML (opens in $EDITOR)
virsh edit vm1

# Define domain from XML file
virsh define vm1.xml

# Create and start from XML (transient)
virsh create vm1.xml

# Dump XML for template
virsh dumpxml vm1 > template.xml

# Apply device changes live
virsh update-device vm1 disk-changes.xml --live

# Attach a disk
virsh attach-disk vm1 /var/lib/libvirt/images/data.qcow2 vdb \
  --driver qemu --subdriver qcow2 --persistent

# Detach a disk
virsh detach-disk vm1 vdb --persistent

# Attach a network interface
virsh attach-interface vm1 bridge br0 --model virtio --persistent

# Detach network interface
virsh detach-interface vm1 bridge --mac 52:54:00:ab:cd:ef --persistent

# Set memory (live and config)
virsh setmem vm1 4G --live --config

# Set vCPUs
virsh setvcpus vm1 4 --live --config
```

### Key XML Sections

```xml
<!-- Minimal domain XML -->
<domain type='kvm'>
  <name>vm1</name>
  <memory unit='GiB'>4</memory>
  <vcpu placement='static'>4</vcpu>
  <cpu mode='host-passthrough'/>
  <os>
    <type arch='x86_64' machine='q35'>hvm</type>
    <boot dev='hd'/>
  </os>
  <devices>
    <disk type='file' device='disk'>
      <driver name='qemu' type='qcow2' cache='none' io='native'/>
      <source file='/var/lib/libvirt/images/vm1.qcow2'/>
      <target dev='vda' bus='virtio'/>
    </disk>
    <interface type='bridge'>
      <source bridge='br0'/>
      <model type='virtio'/>
    </interface>
    <serial type='pty'><target port='0'/></serial>
    <console type='pty'><target type='serial' port='0'/></console>
    <channel type='unix'>
      <target type='virtio' name='org.qemu.guest_agent.0'/>
    </channel>
  </devices>
</domain>
```

## Storage Pools

### Pool Management

```bash
# List storage pools
virsh pool-list --all

# Pool details
virsh pool-info default

# Create directory pool
virsh pool-define-as mypool dir --target /srv/vms
virsh pool-build mypool
virsh pool-start mypool
virsh pool-autostart mypool

# Create LVM pool
virsh pool-define-as lvm-pool logical \
  --source-name vg_vms --target /dev/vg_vms
virsh pool-start lvm-pool

# Create ZFS pool
virsh pool-define-as zfs-pool zfs --source-name zpool/vms

# List volumes in a pool
virsh vol-list default

# Create a volume
virsh vol-create-as default vm2.qcow2 40G --format qcow2

# Clone a volume
virsh vol-clone --pool default vm1.qcow2 vm1-clone.qcow2

# Delete a volume
virsh vol-delete --pool default old-disk.qcow2

# Resize a volume
virsh vol-resize --pool default vm1.qcow2 60G

# Pool refresh (rescan)
virsh pool-refresh default

# Delete pool
virsh pool-destroy mypool
virsh pool-undefine mypool
```

## Virtual Networking

### Network Management

```bash
# List networks
virsh net-list --all

# Network details
virsh net-info default
virsh net-dumpxml default

# Create NAT network
cat > nat-network.xml << 'EOF'
<network>
  <name>nat-net</name>
  <forward mode='nat'>
    <nat>
      <port start='1024' end='65535'/>
    </nat>
  </forward>
  <bridge name='virbr1' stp='on' delay='0'/>
  <ip address='192.168.100.1' netmask='255.255.255.0'>
    <dhcp>
      <range start='192.168.100.100' end='192.168.100.254'/>
      <host mac='52:54:00:aa:bb:01' ip='192.168.100.10'/>
    </dhcp>
  </ip>
</network>
EOF
virsh net-define nat-network.xml
virsh net-start nat-net
virsh net-autostart nat-net

# Create bridge network (host bridge must exist)
cat > bridge-network.xml << 'EOF'
<network>
  <name>host-bridge</name>
  <forward mode='bridge'/>
  <bridge name='br0'/>
</network>
EOF
virsh net-define bridge-network.xml
virsh net-start host-bridge

# Create macvtap network (direct host NIC)
cat > macvtap-network.xml << 'EOF'
<network>
  <name>direct-net</name>
  <forward mode='bridge'>
    <interface dev='eth0'/>
  </forward>
</network>
EOF
virsh net-define macvtap-network.xml

# Delete network
virsh net-destroy nat-net
virsh net-undefine nat-net

# DHCP leases
virsh net-dhcp-leases default
```

## Snapshots

### Snapshot Management

```bash
# Create snapshot (with memory state)
virsh snapshot-create-as vm1 snap1 "Clean install snapshot"

# Create disk-only snapshot (no memory)
virsh snapshot-create-as vm1 snap2 "Disk only" --disk-only

# List snapshots
virsh snapshot-list vm1

# Snapshot info
virsh snapshot-info vm1 snap1

# Revert to snapshot
virsh snapshot-revert vm1 snap1

# Delete snapshot
virsh snapshot-delete vm1 snap1

# Delete snapshot but keep data
virsh snapshot-delete vm1 snap1 --metadata
```

## Migration

### Live and Offline Migration

```bash
# Live migration (shared storage)
virsh migrate --live --persistent --verbose vm1 \
  qemu+ssh://target-host/system

# Live migration with bandwidth limit (MiB/s)
virsh migrate --live --bandwidth 500 vm1 \
  qemu+ssh://target-host/system

# Copy storage during migration (no shared storage)
virsh migrate --live --copy-storage-all --persistent vm1 \
  qemu+ssh://target-host/system

# Offline migration (copy config only)
virsh migrate --offline --persistent vm1 \
  qemu+ssh://target-host/system

# Monitor migration
virsh domjobinfo vm1

# Cancel migration
virsh domjobabort vm1

# Set migration speed
virsh migrate-setspeed vm1 1000
```

## Tips

- Use `virsh edit` instead of manually modifying XML files to get validation and proper daemon notification
- Always pass `--persistent` with `virsh migrate` to ensure the domain is defined on the target host
- Use `host-passthrough` CPU mode for performance and `host-model` for migration between different CPU generations
- Set `cache='none'` and `io='native'` on VirtIO disks for best I/O performance with O_DIRECT
- Use macvtap (`mode='bridge'`) for simplest bridged networking without creating a Linux bridge
- Storage pools with autostart ensure volumes are available after host reboot
- The QEMU guest agent (`qemu-ga`) enables filesystem freeze, guest exec, and clean shutdown from host
- Static DHCP reservations in network XML give VMs predictable IPs without configuring inside the guest
- Run `virsh pool-refresh` after externally modifying pool contents to update libvirt's volume cache

## See Also

- qemu, kvm, proxmox, virt-install, cloud-init, bridge-utils

## References

- [Libvirt Documentation](https://libvirt.org/docs.html)
- [Libvirt Domain XML Format](https://libvirt.org/formatdomain.html)
- [Libvirt Network XML Format](https://libvirt.org/formatnetwork.html)
- [Libvirt Storage XML Format](https://libvirt.org/formatstorage.html)
- [Virsh Command Reference](https://libvirt.org/manpages/virsh.html)
