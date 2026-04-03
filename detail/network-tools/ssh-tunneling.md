# The Mathematics of SSH Tunneling — Port Forwarding and Proxy Internals

> *SSH tunneling creates encrypted channels for port forwarding, SOCKS proxying, and VPN-like connectivity. The math covers tunnel overhead, multiplexing, bandwidth-delay product, and dynamic forwarding performance.*

---

## 1. Local Port Forwarding — The Basic Tunnel

### The Model

Local forwarding: `ssh -L local_port:target_host:target_port user@ssh_server`

$$\text{Client} \xrightarrow{\text{localhost:local\_port}} \text{SSH Client} \xrightarrow[\text{encrypted}]{\text{SSH Channel}} \text{SSH Server} \xrightarrow{\text{plaintext}} \text{Target}$$

### Latency Addition

$$T_{tunneled} = T_{direct} + 2 \times T_{ssh\_hop}$$

$$T_{ssh\_hop} = T_{encryption} + T_{channel\_overhead}$$

| Path | Direct RTT | Tunneled RTT | Overhead |
|:---|:---:|:---:|:---:|
| LAN (0.5ms) | 0.5 ms | 0.7 ms | +0.2 ms |
| WAN (50ms) | 50 ms | 50.5 ms | +0.5 ms |
| Multi-hop (100ms) | 100 ms | 101 ms | +1 ms |

### Throughput Through Tunnel

$$\text{Tunnel BW} = \min(\text{SSH Link BW}, \text{Cipher Throughput}, \text{SSH Window})$$

$$\text{SSH Window Limit} = \frac{\text{Channel Window Size}}{\text{RTT}}$$

Default SSH channel window = 2 MiB:

| RTT | Window (2 MiB) | Max Throughput |
|:---:|:---:|:---:|
| 1 ms | 2 MiB | 2 GiB/s |
| 10 ms | 2 MiB | 200 MiB/s |
| 50 ms | 2 MiB | 40 MiB/s |
| 200 ms | 2 MiB | 10 MiB/s |

---

## 2. Remote Port Forwarding — Reverse Tunnel

### The Model

Remote forwarding: `ssh -R remote_port:target_host:target_port user@ssh_server`

$$\text{External Client} \xrightarrow{\text{ssh\_server:remote\_port}} \text{SSH Server} \xrightarrow[\text{encrypted}]{\text{SSH Channel}} \text{SSH Client} \xrightarrow{\text{plaintext}} \text{Target}$$

### Use Case Calculations

**Exposing a local service through NAT:**

$$\text{Total RTT} = \text{Client→SSH Server RTT} + \text{SSH Server→Local RTT}$$

### Security Boundary

$$\text{Attack Surface} = \text{Remote port on SSH server} \quad (\text{default: localhost only})$$

With `GatewayPorts yes`:

$$\text{Attack Surface} = \text{Remote port on all interfaces}$$

---

## 3. Dynamic Forwarding — SOCKS Proxy

### The Model

Dynamic forwarding: `ssh -D local_port user@ssh_server`

Creates a SOCKS5 proxy. Each connection through the proxy creates an SSH channel.

### SOCKS5 Overhead

$$\text{Handshake} = 3 \text{ messages (greeting + auth + connect request)} = 3 \times \text{RTT}$$

$$\text{Per-Connection Overhead} = \text{SOCKS Handshake} + \text{SSH Channel Open}$$

### Worked Example

*"Browsing through SOCKS proxy with 50ms RTT to SSH server."*

$$T_{page\_load} = T_{socks\_setup} + T_{tls\_setup} + T_{http\_request}$$

$$T_{socks\_setup} = 3 \times 50\text{ms} = 150\text{ms}$$

$$T_{tls\_setup} = 2 \times 50\text{ms} = 100\text{ms (TLS over tunnel)}$$

$$\text{Added latency per connection} = 250\text{ms}$$

For a page with 30 resources (HTTP/2 multiplexed = 1 connection):

$$T_{overhead} = 250\text{ms (one-time per connection)}$$

For HTTP/1.1 (6 connections):

$$T_{overhead} = 6 \times 250\text{ms} = 1.5\text{s (parallel, so } \approx 250\text{ms)}$$

---

## 4. SSH Multiplexing — ControlMaster

### The Model

SSH multiplexing reuses a single TCP connection for multiple SSH sessions.

### Connection Savings

$$T_{new\_ssh} = T_{TCP} + T_{key\_exchange} + T_{auth} \approx 200-500\text{ms}$$

$$T_{multiplex} = T_{channel\_open} \approx 1 \times \text{RTT} \approx 1-50\text{ms}$$

$$\text{Speedup} = \frac{T_{new\_ssh}}{T_{multiplex}} = 4-500\times$$

### Resource Sharing

$$\text{Single multiplexed connection:}$$

$$\text{TCP connections} = 1 \quad (\text{vs } n \text{ without multiplexing})$$

$$\text{Key exchanges} = 1 \quad (\text{vs } n)$$

$$\text{Auth attempts} = 1 \quad (\text{vs } n)$$

### Channel Limits

$$\text{Max channels per connection} = 2^{31} \quad (\text{theoretical, practical limit ~100-1000})$$

---

## 5. Tunnel-in-Tunnel — ProxyJump

### The Model

`ssh -J jump_host target_host` creates a tunnel through an intermediate host.

### Multi-Hop Latency

$$T_{total} = \sum_{i=1}^{n} T_{hop_i}$$

$$\text{RTT}_{e2e} = \sum_{i=1}^{n} \text{RTT}_{hop_i}$$

### Encryption Layers

$$\text{Data passes through } n \text{ encryption layers}$$

$$\text{CPU overhead} = n \times T_{cipher}$$

### Worked Example

*"SSH through 3 hops: client → bastion → internal jump → target."*

| Hop | RTT | Cipher |
|:---|:---:|:---|
| Client → Bastion | 20 ms | AES-256-GCM |
| Bastion → Jump | 5 ms | AES-256-GCM |
| Jump → Target | 1 ms | AES-256-GCM |

$$\text{Total RTT} = 20 + 5 + 1 = 26\text{ms}$$

$$\text{Encryption layers} = 3 \quad (\text{3x cipher CPU cost})$$

$$\text{Bandwidth} = \min(\text{BW per hop})$$

---

## 6. VPN-Mode — SSH TUN/TAP

### The Model

SSH can create Layer 2 (TAP) or Layer 3 (TUN) tunnels, acting as a VPN.

### TUN (Layer 3) Overhead

$$\text{Overhead per Packet} = \text{SSH Header (25-50 bytes)} + \text{Padding}$$

$$\text{Effective MTU} = \text{Link MTU} - \text{SSH Overhead} - \text{Tunnel Header}$$

$$\text{Typical: } 1500 - 50 - 20 = 1430 \text{ bytes}$$

### Throughput Comparison

| Method | MTU | Overhead | Typical Throughput |
|:---|:---:|:---:|:---:|
| Native | 1500 | 0% | 100% |
| SSH TUN | 1430 | 4.7% | ~90% |
| SSH SOCKS | N/A | Per-connection | ~85% |
| WireGuard | 1420 | 5.3% | ~95% (kernel) |
| OpenVPN (UDP) | 1400 | 6.7% | ~80% |

### TAP (Layer 2) Additional Overhead

$$\text{TAP Header} = 14 \text{ bytes (Ethernet)} + \text{optional VLAN (4 bytes)}$$

$$\text{Effective MTU}_{TAP} = 1500 - 50 - 20 - 14 = 1416 \text{ bytes}$$

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\frac{\text{Window}}{\text{RTT}}$ | Division | Tunnel max throughput |
| $T_{direct} + 2 \times T_{hop}$ | Addition | Tunnel latency |
| $3 \times \text{RTT}$ | Multiplication | SOCKS setup overhead |
| $\frac{T_{new}}{T_{mux}}$ | Ratio | Multiplexing speedup |
| $\sum T_{hop_i}$ | Summation | Multi-hop RTT |
| $\text{MTU} - \text{Headers}$ | Subtraction | Effective tunnel MTU |

---

*Every `ssh -L`, `ssh -R`, `ssh -D`, and `ssh -J` creates encrypted data channels — a tunneling system so versatile it can replace VPNs, port forwarders, SOCKS proxies, and jump hosts, all through a single protocol that was designed for terminal access.*
