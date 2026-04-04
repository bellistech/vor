# kvm (Kernel-based Virtual Machine)

KVM turns the Linux kernel into a type-1 hypervisor using hardware virtualization extensions (Intel VT-x/AMD-V), providing near-native performance for virtual machines through direct CPU instruction execution with trap-and-emulate for privileged operations.

## Setup and Verification

### Check Hardware Support

```bash
# Check CPU virtualization support
grep -E '(vmx|svm)' /proc/cpuinfo | head -1

# Check if KVM modules are loaded
lsmod | grep kvm

# Load KVM modules manually
sudo modprobe kvm
sudo modprobe kvm_intel   # Intel
sudo modprobe kvm_amd     # AMD

# Verify KVM device exists
ls -la /dev/kvm

# Check KVM capabilities
cat /sys/module/kvm_intel/parameters/nested
cat /sys/module/kvm_intel/parameters/ept
cat /sys/module/kvm_intel/parameters/unrestricted_guest

# Install KVM tools (Debian/Ubuntu)
sudo apt install qemu-kvm libvirt-daemon-system virtinst bridge-utils

# Install KVM tools (RHEL/Fedora)
sudo dnf install @virtualization

# Add user to KVM/libvirt groups
sudo usermod -aG kvm,libvirt $USER
```

## VM Creation with virt-install

### Create VMs from ISO

```bash
# Basic VM creation from ISO
virt-install \
  --name ubuntu-server \
  --ram 4096 \
  --vcpus 4 \
  --cpu host-passthrough \
  --disk path=/var/lib/libvirt/images/ubuntu.qcow2,size=40,format=qcow2 \
  --cdrom /var/lib/libvirt/boot/ubuntu-24.04.iso \
  --os-variant ubuntu24.04 \
  --network bridge=br0 \
  --graphics vnc,listen=0.0.0.0 \
  --console pty,target_type=serial

# Headless server install (text mode)
virt-install \
  --name debian-headless \
  --ram 2048 \
  --vcpus 2 \
  --cpu host-passthrough \
  --disk path=/var/lib/libvirt/images/debian.qcow2,size=20 \
  --location https://deb.debian.org/debian/dists/bookworm/main/installer-amd64/ \
  --os-variant debian12 \
  --network network=default \
  --graphics none \
  --extra-args 'console=ttyS0,115200n8 serial'

# Cloud image with cloud-init
virt-install \
  --name cloud-vm \
  --ram 2048 \
  --vcpus 2 \
  --import \
  --disk path=cloud-vm.qcow2,format=qcow2 \
  --cloud-init user-data=user-data.yaml \
  --os-variant ubuntu24.04 \
  --network network=default \
  --noautoconsole

# List available OS variants
virt-install --osinfo list
```

## CPU Configuration

### CPU Models and Features

```bash
# Pass through host CPU model (best performance)
virsh dumpxml vm1 | grep -A5 '<cpu'

# Set CPU model in domain XML
# <cpu mode='host-passthrough' check='none' migratable='on'/>

# Named CPU model (for migration compatibility)
# <cpu mode='custom' match='exact' check='partial'>
#   <model fallback='allow'>Skylake-Server-v4</model>
# </cpu>

# Pin vCPUs to physical cores
virsh vcpupin vm1 0 2
virsh vcpupin vm1 1 3
virsh vcpupin vm1 2 4
virsh vcpupin vm1 3 5

# Show current pinning
virsh vcpupin vm1

# Set CPU topology
virsh setvcpus vm1 4 --config

# Hot-add vCPUs (if maxvcpus > current)
virsh setvcpus vm1 6 --live
```

## Memory Tuning

### Hugepages Configuration

```bash
# Check current hugepage allocation
cat /proc/meminfo | grep Huge

# Allocate 1024 2MB hugepages (2GB total)
echo 1024 | sudo tee /proc/sys/vm/nr_hugepages

# Persistent hugepage allocation (sysctl)
echo "vm.nr_hugepages = 1024" | sudo tee /etc/sysctl.d/hugepages.conf
sudo sysctl -p /etc/sysctl.d/hugepages.conf

# Mount hugetlbfs
sudo mount -t hugetlbfs hugetlbfs /dev/hugepages

# 1GB hugepages (boot parameter)
# GRUB_CMDLINE_LINUX="hugepagesz=1G hugepages=16 default_hugepagesz=1G"

# Enable hugepages in VM XML
# <memoryBacking>
#   <hugepages>
#     <page size='2048' unit='KiB'/>
#   </hugepages>
# </memoryBacking>
```

### Memory Ballooning

```bash
# Check current memory
virsh dominfo vm1 | grep -i mem

# Set memory dynamically (balloon down)
virsh setmem vm1 2G --live

# Set maximum memory
virsh setmaxmem vm1 8G --config

# Disable ballooning for latency-sensitive VMs
# <memballoon model='none'/>

# Monitor balloon stats
virsh dommemstat vm1
```

## Nested Virtualization

### Enable Nested Virt

```bash
# Check if nested virt is enabled
cat /sys/module/kvm_intel/parameters/nested

# Enable nested virt (runtime)
sudo modprobe -r kvm_intel
sudo modprobe kvm_intel nested=1

# Enable nested virt (persistent)
echo "options kvm_intel nested=1" | sudo tee /etc/modprobe.d/kvm-nested.conf

# AMD equivalent
echo "options kvm_amd nested=1" | sudo tee /etc/modprobe.d/kvm-nested.conf

# Verify L2 guest sees vmx flag
grep vmx /proc/cpuinfo  # inside L1 guest
```

## IOMMU and Device Passthrough

### IOMMU Setup

```bash
# Enable IOMMU (Intel) - add to GRUB_CMDLINE_LINUX
# intel_iommu=on iommu=pt

# Enable IOMMU (AMD)
# amd_iommu=on iommu=pt

# Update GRUB
sudo update-grub
# or
sudo grub2-mkconfig -o /boot/grub2/grub.cfg

# Verify IOMMU groups
find /sys/kernel/iommu_groups/ -type l | sort -V

# List devices in an IOMMU group
ls -la /sys/kernel/iommu_groups/1/devices/

# Check IOMMU group for a specific device
readlink /sys/bus/pci/devices/0000:01:00.0/iommu_group

# Bind device to vfio-pci
echo "vfio-pci" | sudo tee /sys/bus/pci/devices/0000:01:00.0/driver_override
echo "0000:01:00.0" | sudo tee /sys/bus/pci/drivers_probe
```

## Live Migration

### Migrate VMs Between Hosts

```bash
# Prerequisites: shared storage, same CPU family, libvirtd on both hosts

# Live migration (default)
virsh migrate --live vm1 qemu+ssh://target-host/system

# Live migration with bandwidth limit (MiB/s)
virsh migrate --live --bandwidth 100 vm1 qemu+ssh://target-host/system

# Offline migration (VM must be shut off)
virsh migrate --offline --persistent vm1 qemu+ssh://target-host/system

# Migration with storage (no shared storage needed)
virsh migrate --live --copy-storage-all vm1 qemu+ssh://target-host/system

# Monitor migration progress
virsh domjobinfo vm1

# Abort migration
virsh domjobabort vm1

# Set migration speed
virsh migrate-setspeed vm1 500
```

## Monitoring and Diagnostics

### Performance Metrics

```bash
# VM CPU usage
virsh cpu-stats vm1

# VM block device stats
virsh domblkstat vm1 vda

# VM network stats
virsh domifstat vm1 vnet0

# Top-like view for VMs
virt-top

# KVM event tracing
sudo perf kvm stat live

# Record KVM events
sudo perf kvm stat record -a sleep 10
sudo perf kvm stat report

# Check for KVM-related kernel messages
dmesg | grep -i kvm
```

## Tips

- Always enable IOMMU even without passthrough for better security isolation via VT-d/AMD-Vi
- Use `host-passthrough` CPU mode for best performance but `host-model` for migration compatibility
- Hugepages eliminate TLB misses for VM memory, critical for latency-sensitive workloads
- Pin vCPUs to physical cores on the same NUMA node as the VM's memory for optimal locality
- Enable nested virtualization only when needed as it adds overhead to all VM exits
- Set `iommu=pt` (passthrough) in kernel params to avoid DMA translation overhead for host devices
- Use `virt-top` instead of `top` for per-VM resource monitoring across the hypervisor
- KVM halt-polling (`halt_poll_ns`) trades CPU for latency: increase for latency-sensitive, decrease for density
- Memory ballooning is incompatible with hugepages; choose based on workload needs
- Check `/sys/kernel/iommu_groups/` to verify all devices in a passthrough group can be isolated

## See Also

- qemu, libvirt, proxmox, hugepages, numa, iommu

## References

- [KVM Documentation](https://www.linux-kvm.org/page/Main_Page)
- [KVM API Documentation](https://www.kernel.org/doc/html/latest/virt/kvm/api.html)
- [Red Hat Virtualization Tuning Guide](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/9/html/configuring_and_managing_virtualization/)
- [Arch Wiki: KVM](https://wiki.archlinux.org/title/KVM)
- [IOMMU Groups Explained](https://vfio.blogspot.com/2014/08/iommu-groups-inside-and-out.html)
