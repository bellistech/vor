# TUN/TAP Virtual Network Devices

Software-defined network interfaces for userspace packet processing at Layer 3 (TUN) or Layer 2 (TAP).

## TUN vs TAP

```
TUN (tunnel)  — Layer 3 (IP packets)    — used for routed VPNs
TAP (tap)     — Layer 2 (Ethernet frames) — used for bridged VPNs, VMs

TUN: userspace reads/writes raw IP packets (no Ethernet header)
TAP: userspace reads/writes full Ethernet frames (with MAC header)
```

## Creating Devices

```bash
# Create a TUN device
ip tuntap add dev tun0 mode tun user $(whoami)

# Create a TAP device
ip tuntap add dev tap0 mode tap user $(whoami)

# Create with specific group ownership
ip tuntap add dev tun0 mode tun group netdev

# Create persistent device (survives process exit)
ip tuntap add dev tun0 mode tun
ip link set tun0 up

# Assign IP address
ip addr add 10.0.0.1/24 dev tun0
ip link set tun0 up

# Delete device
ip tuntap del dev tun0 mode tun

# List TUN/TAP devices
ip tuntap list
ip link show type tun
```

## Multi-Queue TUN/TAP

```bash
# Create multi-queue device (improves throughput with RSS)
ip tuntap add dev tap0 mode tap multi_queue

# Each queue is opened by a separate file descriptor
# Kernel distributes packets across queues by flow hash
```

## Programming Interface (/dev/net/tun)

```c
#include <linux/if_tun.h>
#include <net/if.h>
#include <fcntl.h>
#include <sys/ioctl.h>

int tun_alloc(char *dev, int flags) {
    struct ifreq ifr;
    int fd, err;

    // Open the clone device
    fd = open("/dev/net/tun", O_RDWR);
    if (fd < 0) return fd;

    memset(&ifr, 0, sizeof(ifr));
    ifr.ifr_flags = flags;  // IFF_TUN or IFF_TAP

    // Optional: IFF_NO_PI disables the 4-byte packet info header
    ifr.ifr_flags |= IFF_NO_PI;

    if (*dev)
        strncpy(ifr.ifr_name, dev, IFNAMSIZ);

    err = ioctl(fd, TUNSETIFF, (void *)&ifr);
    if (err < 0) {
        close(fd);
        return err;
    }

    strcpy(dev, ifr.ifr_name);
    return fd;
}
```

## Read/Write Packets

```c
// Read a packet from TUN device (blocks until packet arrives)
char buf[2048];
int nread = read(tun_fd, buf, sizeof(buf));
// buf now contains an IP packet (TUN) or Ethernet frame (TAP)

// Write a packet back to TUN device (injects into kernel stack)
int nwrite = write(tun_fd, packet, packet_len);

// Non-blocking I/O
int flags = fcntl(tun_fd, F_GETFL, 0);
fcntl(tun_fd, F_SETFL, flags | O_NONBLOCK);

// With epoll for event-driven I/O
struct epoll_event ev;
ev.events = EPOLLIN;
ev.data.fd = tun_fd;
epoll_ctl(epfd, EPOLL_CTL_ADD, tun_fd, &ev);
```

## Persistent Devices

```bash
# Make device persistent (survives owning process exit)
# From C:
ioctl(fd, TUNSETPERSIST, 1);

# From command line (created devices are persistent by default)
ip tuntap add dev tun0 mode tun

# Remove persistence
ioctl(fd, TUNSETPERSIST, 0);
# Or delete the device
ip tuntap del dev tun0 mode tun
```

## Packet Info Header (PI)

```
Without IFF_NO_PI, each packet has a 4-byte header:
  Bytes 0-1: flags (0 = nothing special)
  Bytes 2-3: protocol (ETH_P_IP = 0x0800, ETH_P_IPV6 = 0x86DD)

Recommendation: always use IFF_NO_PI to simplify parsing
```

## VPN Implementation Pattern

```bash
# Simple TUN-based VPN architecture:
#
# Host A (10.0.0.1)              Host B (10.0.0.2)
# +------------------+           +------------------+
# | App traffic      |           | App traffic      |
# |    |             |           |    |             |
# | [tun0 10.8.0.1]  |           | [tun0 10.8.0.2]  |
# |    |             |           |    |             |
# | VPN process      | -------> | VPN process      |
# | (encrypt+send)   |  UDP/TCP | (recv+decrypt)   |
# |    |             |           |    |             |
# | [eth0]           |           | [eth0]           |
# +------------------+           +------------------+

# Enable IP forwarding
sysctl -w net.ipv4.ip_forward=1

# NAT for VPN traffic
iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
```

## QEMU/KVM Usage

```bash
# Create TAP device for VM networking
ip tuntap add dev tap0 mode tap user $(whoami)
ip link set tap0 up
ip link set tap0 master br0  # Bridge to host network

# Launch QEMU with TAP backend
qemu-system-x86_64 \
    -netdev tap,id=net0,ifname=tap0,script=no,downscript=no \
    -device virtio-net-pci,netdev=net0 \
    disk.img

# With vhost-net for kernel-level packet forwarding
qemu-system-x86_64 \
    -netdev tap,id=net0,ifname=tap0,script=no,vhost=on \
    -device virtio-net-pci,netdev=net0 \
    disk.img

# macvtap alternative (no bridge needed)
ip link add link eth0 name macvtap0 type macvtap mode bridge
ip link set macvtap0 up
```

## MTU Considerations

```bash
# Default MTU is 1500 for TAP, 1500 for TUN
ip link set dev tun0 mtu 1400

# VPN tunneling reduces effective MTU:
# Ethernet MTU:          1500
# - IP header:            -20
# - UDP header:            -8
# - VPN overhead:         -32 (varies by protocol)
# = Tunnel MTU:          1440
#
# WireGuard overhead:     60 bytes (IPv4) / 80 bytes (IPv6)
# OpenVPN overhead:       ~50-70 bytes (depends on cipher)

# Avoid fragmentation — set tunnel MTU correctly
ip link set dev tun0 mtu 1420  # WireGuard default

# Enable path MTU discovery
sysctl -w net.ipv4.ip_no_pmtu_disc=0

# Clamp MSS to prevent TCP fragmentation
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
    -j TCPMSS --clamp-mss-to-pmtu
```

## Python Example (Simple TUN Reader)

```python
import os, struct, fcntl

TUNSETIFF = 0x400454ca
IFF_TUN   = 0x0001
IFF_NO_PI = 0x1000

fd = os.open("/dev/net/tun", os.O_RDWR)
ifr = struct.pack('16sH', b'tun0', IFF_TUN | IFF_NO_PI)
fcntl.ioctl(fd, TUNSETIFF, ifr)

os.system("ip addr add 10.0.0.1/24 dev tun0")
os.system("ip link set tun0 up")

while True:
    packet = os.read(fd, 2048)
    print(f"Got {len(packet)} byte packet")
    # Parse IP header, process, re-inject...
```

## Go Example

```go
package main

import (
    "fmt"
    "os"
    "syscall"
    "unsafe"
)

func main() {
    fd, _ := syscall.Open("/dev/net/tun", syscall.O_RDWR, 0)
    var ifr [40]byte
    copy(ifr[:], "tun0")
    *(*uint16)(unsafe.Pointer(&ifr[16])) = 0x0001 | 0x1000 // IFF_TUN | IFF_NO_PI
    syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd),
        0x400454ca, uintptr(unsafe.Pointer(&ifr[0])))

    buf := make([]byte, 2048)
    for {
        n, _ := syscall.Read(fd, buf)
        fmt.Printf("Read %d bytes\n", n)
    }
}
```

## Tips

- Always use `IFF_NO_PI` unless you specifically need the packet info header
- Set the TUN/TAP device MTU to account for encapsulation overhead to avoid fragmentation
- Use `multi_queue` for high-throughput applications; it scales with CPU cores
- TAP devices can join bridges (`ip link set tap0 master br0`) for L2 connectivity
- TUN devices need routing rules (`ip route add ... dev tun0`) to direct traffic
- Use `vhost-net` with QEMU TAP devices for near-native network performance
- File descriptor leaks will keep persistent devices alive; always clean up on exit
- The `user` parameter lets unprivileged processes attach to the device
- Use `SO_MARK` on the outer socket to prevent routing loops in VPN implementations
- For production VPNs, prefer WireGuard or OpenVPN over hand-rolled TUN implementations
- Use `epoll` rather than `select`/`poll` for handling multiple TUN/TAP fds efficiently
- Check `/dev/net/tun` exists; load the `tun` module with `modprobe tun` if missing

## See Also

- WireGuard (modern VPN built on TUN)
- veth (virtual Ethernet pairs for container networking)
- macvtap (MAC-based TAP for direct VM attachment)
- bridge (Linux software bridge for L2 forwarding)
- XDP (high-performance packet processing)

## References

- [TUN/TAP Kernel Documentation](https://www.kernel.org/doc/html/latest/networking/tuntap.html)
- [Universal TUN/TAP Device Driver (kernel.org)](https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git/tree/Documentation/networking/tuntap.rst)
- [QEMU Networking Documentation](https://wiki.qemu.org/Documentation/Networking)
- [WireGuard Protocol Specification](https://www.wireguard.com/protocol/)
- [vhost-net and virtio-net Architecture](https://www.redhat.com/en/blog/introduction-virtio-networking-and-vhost-net)
