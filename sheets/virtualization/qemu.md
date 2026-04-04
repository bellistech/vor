# qemu (machine emulator)

QEMU is an open-source machine emulator and virtualizer that can run operating systems for any machine on any supported architecture, supporting full system emulation with hardware acceleration via KVM and device passthrough for near-native performance.

## Installation

### Package Managers

```bash
# Debian/Ubuntu
sudo apt install qemu-system-x86 qemu-utils qemu-system-arm

# RHEL/Fedora
sudo dnf install qemu-kvm qemu-img

# Arch
sudo pacman -S qemu-full

# macOS (x86_64 emulation on ARM)
brew install qemu

# Check installed version
qemu-system-x86_64 --version

# List supported machines
qemu-system-x86_64 -machine help

# List supported CPUs
qemu-system-x86_64 -cpu help
```

## Disk Image Management

### Create and Convert Images

```bash
# Create qcow2 image (thin provisioned)
qemu-img create -f qcow2 disk.qcow2 40G

# Create raw image (preallocated)
qemu-img create -f raw disk.raw 40G

# Create qcow2 with backing file (copy-on-write)
qemu-img create -f qcow2 -b base.qcow2 -F qcow2 overlay.qcow2

# Convert raw to qcow2
qemu-img convert -f raw -O qcow2 disk.raw disk.qcow2

# Convert qcow2 to raw (for dd to physical disk)
qemu-img convert -f qcow2 -O raw disk.qcow2 disk.raw

# Convert with compression
qemu-img convert -f qcow2 -O qcow2 -c input.qcow2 compressed.qcow2

# Inspect image info
qemu-img info disk.qcow2

# Check image for errors
qemu-img check disk.qcow2

# Resize image (grow only for qcow2)
qemu-img resize disk.qcow2 +20G

# Shrink image (must specify --shrink flag)
qemu-img resize --shrink disk.qcow2 30G
```

### Snapshots

```bash
# Create internal snapshot
qemu-img snapshot -c snap1 disk.qcow2

# List snapshots
qemu-img snapshot -l disk.qcow2

# Restore snapshot
qemu-img snapshot -a snap1 disk.qcow2

# Delete snapshot
qemu-img snapshot -d snap1 disk.qcow2

# Create external snapshot (live, via QEMU monitor)
# (ctrl+alt+2 to access monitor in graphical mode)
# savevm snap_name
# loadvm snap_name
# delvm snap_name
# info snapshots
```

## Basic VM Launch

### Common Startup Commands

```bash
# Minimal boot from ISO with KVM acceleration
qemu-system-x86_64 \
  -enable-kvm \
  -m 4G \
  -smp 4 \
  -cpu host \
  -drive file=disk.qcow2,format=qcow2 \
  -cdrom ubuntu-24.04.iso \
  -boot d

# Headless VM with VNC
qemu-system-x86_64 \
  -enable-kvm \
  -m 2G \
  -smp 2 \
  -cpu host \
  -drive file=disk.qcow2,format=qcow2 \
  -vnc :1 \
  -daemonize

# Boot with EFI firmware (OVMF)
qemu-system-x86_64 \
  -enable-kvm \
  -m 4G \
  -cpu host \
  -drive if=pflash,format=raw,readonly=on,file=/usr/share/OVMF/OVMF_CODE.fd \
  -drive if=pflash,format=raw,file=OVMF_VARS.fd \
  -drive file=disk.qcow2,format=qcow2

# ARM64 emulation (no KVM on x86 host)
qemu-system-aarch64 \
  -M virt \
  -cpu cortex-a72 \
  -m 2G \
  -bios /usr/share/qemu-efi-aarch64/QEMU_EFI.fd \
  -drive file=arm-disk.qcow2,format=qcow2 \
  -nographic
```

## VirtIO Drivers

### High-Performance Virtual Devices

```bash
# VirtIO disk (fastest virtual storage)
-drive file=disk.qcow2,format=qcow2,if=virtio

# VirtIO block with IO thread
-object iothread,id=io1 \
-drive file=disk.qcow2,format=qcow2,if=none,id=drive0,aio=native,cache=none \
-device virtio-blk-pci,drive=drive0,iothread=io1

# VirtIO network
-device virtio-net-pci,netdev=net0 \
-netdev user,id=net0

# VirtIO RNG (random number generator)
-device virtio-rng-pci

# VirtIO balloon (dynamic memory)
-device virtio-balloon-pci

# VirtIO serial console
-device virtio-serial-pci \
-chardev socket,id=channel0,path=/tmp/qga.sock,server=on,wait=off \
-device virtserialport,chardev=channel0,name=org.qemu.guest_agent.0
```

## Networking

### Network Modes

```bash
# User-mode networking (NAT, no root needed)
-netdev user,id=net0 \
-device virtio-net-pci,netdev=net0

# User-mode with port forwarding (host 2222 -> guest 22)
-netdev user,id=net0,hostfwd=tcp::2222-:22,hostfwd=tcp::8080-:80 \
-device virtio-net-pci,netdev=net0

# TAP networking (bridged, requires root)
sudo ip tuntap add dev tap0 mode tap user $USER
sudo ip link set tap0 up
sudo brctl addif br0 tap0
qemu-system-x86_64 \
  -netdev tap,id=net0,ifname=tap0,script=no,downscript=no \
  -device virtio-net-pci,netdev=net0,mac=52:54:00:12:34:56

# Bridge helper (no root after setup)
-netdev bridge,id=net0,br=br0 \
-device virtio-net-pci,netdev=net0

# Multiple NICs
-device virtio-net-pci,netdev=net0 -netdev user,id=net0 \
-device virtio-net-pci,netdev=net1 -netdev tap,id=net1,ifname=tap0,script=no
```

## Device Passthrough

### GPU and USB Passthrough

```bash
# USB device passthrough (by vendor:product)
-device usb-host,vendorid=0x1234,productid=0x5678

# USB passthrough (by bus.addr)
-device usb-host,hostbus=1,hostaddr=3

# PCI/GPU passthrough (VFIO)
# 1. Unbind from host driver
echo "0000:01:00.0" | sudo tee /sys/bus/pci/devices/0000:01:00.0/driver/unbind
# 2. Bind to vfio-pci
echo "vfio-pci" | sudo tee /sys/bus/pci/devices/0000:01:00.0/driver_override
echo "0000:01:00.0" | sudo tee /sys/bus/pci/drivers/vfio-pci/bind
# 3. Launch VM with passthrough
qemu-system-x86_64 \
  -enable-kvm \
  -m 8G \
  -cpu host \
  -device vfio-pci,host=01:00.0,multifunction=on \
  -device vfio-pci,host=01:00.1

# 9p filesystem sharing (host directory in guest)
-virtfs local,path=/shared,mount_tag=hostshare,security_model=mapped-xattr,id=fs0
```

## QEMU Monitor

### Runtime Control

```bash
# Access monitor (Ctrl+Alt+2 in graphical mode, or via socket)
-monitor unix:/tmp/qemu-monitor.sock,server,nowait

# Connect to monitor socket
socat - UNIX-CONNECT:/tmp/qemu-monitor.sock

# Common monitor commands
# info status          — VM running state
# info block           — block device info
# info network         — network info
# info snapshots       — list snapshots
# stop                 — pause VM
# cont                 — resume VM
# system_reset         — hard reset
# system_powerdown     — ACPI shutdown
# quit                 — kill QEMU process
# screendump file.ppm  — screenshot
# sendkey ctrl-alt-del — send key combo
# migrate "exec:gzip -c > vm.gz"  — save VM state
# change ide1-cd0 new.iso         — swap CD
```

## Performance Tuning

### CPU and Memory

```bash
# Pin vCPUs to physical cores
taskset -c 0-3 qemu-system-x86_64 -smp 4 ...

# Hugepages memory backend
-m 4G \
-mem-path /dev/hugepages \
-mem-prealloc

# NUMA topology (2 nodes, 4 cores each)
-smp 8,sockets=2,cores=4,threads=1 \
-numa node,nodeid=0,cpus=0-3,mem=4G \
-numa node,nodeid=1,cpus=4-7,mem=4G

# Disable unnecessary devices for speed
-nodefaults \
-no-user-config \
-nographic \
-serial none \
-parallel none
```

## Tips

- Always use `-enable-kvm` on Linux for 10-50x speedup over pure emulation
- Use `-cpu host` to expose all host CPU features to the guest for maximum compatibility
- Prefer qcow2 over raw for snapshots, compression, and thin provisioning
- VirtIO drivers give near-native I/O performance but require guest driver support
- Use `-nographic` with `-serial mon:stdio` for headless server VMs to mux serial and monitor
- Set `cache=none,aio=native` on VirtIO block devices for best disk performance with O_DIRECT
- Port forwarding with user-mode networking is simplest for single-VM SSH access
- Backing files create instant clones ideal for test environments and ephemeral VMs
- Use QEMU Guest Agent (qemu-ga) for clean shutdown and filesystem freeze from host
- Always specify `format=qcow2` explicitly to prevent image format probing vulnerabilities

## See Also

- kvm, libvirt, proxmox, virt-install, virsh, cloud-init

## References

- [QEMU Documentation](https://www.qemu.org/docs/master/)
- [QEMU Wiki](https://wiki.qemu.org/Main_Page)
- [QEMU Disk Images](https://www.qemu.org/docs/master/system/images.html)
- [VirtIO Specification](https://docs.oasis-open.org/virtio/virtio/v1.2/virtio-v1.2.html)
- [VFIO GPU Passthrough Guide](https://wiki.archlinux.org/title/PCI_passthrough_via_OVMF)
