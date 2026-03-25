# WireGuard (Modern VPN)

Fast, simple, kernel-level VPN using modern cryptography (Curve25519, ChaCha20, Poly1305).

## Key Generation

```bash
# Generate private key
wg genkey > private.key

# Derive public key
wg pubkey < private.key > public.key

# Generate preshared key (optional, adds post-quantum resistance)
wg genpsk > preshared.key

# One-liner: generate both
wg genkey | tee private.key | wg pubkey > public.key
```

## Interface Setup (Manual)

### Create Interface

```bash
sudo ip link add wg0 type wireguard
sudo ip addr add 10.100.0.1/24 dev wg0
sudo wg setconf wg0 /etc/wireguard/wg0.conf
sudo ip link set wg0 up
```

### Remove Interface

```bash
sudo ip link del wg0
```

## Configuration Files

### Server Config (/etc/wireguard/wg0.conf)

```bash
[Interface]
Address = 10.100.0.1/24
ListenPort = 51820
PrivateKey = <server-private-key>
# Optional: run commands on up/down
PostUp = iptables -A FORWARD -i wg0 -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i wg0 -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

[Peer]
# Client 1
PublicKey = <client1-public-key>
PresharedKey = <preshared-key>
AllowedIPs = 10.100.0.2/32

[Peer]
# Client 2
PublicKey = <client2-public-key>
AllowedIPs = 10.100.0.3/32
```

### Client Config (/etc/wireguard/wg0.conf)

```bash
[Interface]
Address = 10.100.0.2/24
PrivateKey = <client-private-key>
DNS = 1.1.1.1, 9.9.9.9

[Peer]
PublicKey = <server-public-key>
PresharedKey = <preshared-key>
Endpoint = vpn.acme.com:51820
AllowedIPs = 0.0.0.0/0, ::/0            # route all traffic through VPN
PersistentKeepalive = 25                 # keep NAT mappings alive
```

### Split Tunnel (Only Route Specific Subnets)

```bash
[Peer]
PublicKey = <server-public-key>
Endpoint = vpn.acme.com:51820
AllowedIPs = 10.0.0.0/8, 192.168.1.0/24 # only internal traffic through VPN
```

## wg-quick (Recommended)

### Start and Stop

```bash
sudo wg-quick up wg0                    # reads /etc/wireguard/wg0.conf
sudo wg-quick down wg0
```

### Enable at Boot

```bash
sudo systemctl enable wg-quick@wg0
sudo systemctl start wg-quick@wg0
sudo systemctl status wg-quick@wg0
```

## Runtime Management

### Show Interface Status

```bash
sudo wg show                             # all interfaces
sudo wg show wg0                         # specific interface
sudo wg show wg0 dump                    # machine-readable
```

### Add Peer at Runtime

```bash
sudo wg set wg0 peer <public-key> \
  allowed-ips 10.100.0.4/32 \
  endpoint 203.0.113.50:51820
```

### Remove Peer at Runtime

```bash
sudo wg set wg0 peer <public-key> remove
```

### Sync Config (Apply Changes Without Restart)

```bash
sudo wg syncconf wg0 <(wg-quick strip wg0)
```

## NAT and Routing (Server Side)

### Enable IP Forwarding

```bash
# Runtime
sudo sysctl -w net.ipv4.ip_forward=1

# Permanent
echo "net.ipv4.ip_forward = 1" | sudo tee /etc/sysctl.d/99-wireguard.conf
sudo sysctl -p /etc/sysctl.d/99-wireguard.conf
```

### Firewall Rules

```bash
# Allow WireGuard port
sudo ufw allow 51820/udp

# Allow forwarding on the WireGuard interface
sudo ufw route allow in on wg0 out on eth0
```

### iptables NAT (if not using PostUp in config)

```bash
sudo iptables -A FORWARD -i wg0 -j ACCEPT
sudo iptables -A FORWARD -o wg0 -j ACCEPT
sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
```

## Site-to-Site VPN

### Site A Config

```bash
[Interface]
Address = 10.100.0.1/24
ListenPort = 51820
PrivateKey = <site-a-private-key>

[Peer]
PublicKey = <site-b-public-key>
Endpoint = site-b.acme.com:51820
AllowedIPs = 10.100.0.2/32, 192.168.2.0/24  # Site B's VPN IP + LAN
PersistentKeepalive = 25
```

### Site B Config

```bash
[Interface]
Address = 10.100.0.2/24
ListenPort = 51820
PrivateKey = <site-b-private-key>

[Peer]
PublicKey = <site-a-public-key>
Endpoint = site-a.acme.com:51820
AllowedIPs = 10.100.0.1/32, 192.168.1.0/24  # Site A's VPN IP + LAN
PersistentKeepalive = 25
```

## Troubleshooting

```bash
# Check interface is up
ip a show wg0

# Check handshake happened (should show "latest handshake" timestamp)
sudo wg show wg0

# Test connectivity
ping 10.100.0.1

# Check routing table
ip route | grep wg0

# Trace packet flow
sudo tcpdump -i wg0 -n
sudo tcpdump -i eth0 udp port 51820

# Check for kernel module
lsmod | grep wireguard
```

## Tips

- WireGuard is stateless -- peers that haven't completed a handshake in 5 minutes are considered inactive
- `AllowedIPs` acts as both an ACL and a routing table; it controls which IPs are accepted AND routed through a peer
- `PersistentKeepalive = 25` is essential for clients behind NAT; without it, the NAT mapping expires and incoming packets are dropped
- Config file permissions must be `600` or `wg-quick` will warn: `chmod 600 /etc/wireguard/wg0.conf`
- There is no "connection state" -- if `wg show` has no "latest handshake", traffic has not flowed yet (try pinging)
- DNS in the `[Interface]` section is only handled by `wg-quick`; manual `wg` setup ignores it
- WireGuard uses UDP only (default port 51820); if blocked, consider running on port 443/udp
- `wg syncconf` applies peer changes without dropping existing connections, unlike `wg-quick down && up`

## References

- [WireGuard Official Documentation](https://www.wireguard.com/)
- [WireGuard Whitepaper](https://www.wireguard.com/papers/wireguard.pdf)
- [WireGuard Quick Start](https://www.wireguard.com/quickstart/)
- [wg(8) Man Page](https://man7.org/linux/man-pages/man8/wg.8.html)
- [wg-quick(8) Man Page](https://man7.org/linux/man-pages/man8/wg-quick.8.html)
- [Arch Wiki — WireGuard](https://wiki.archlinux.org/title/WireGuard)
- [Ubuntu — WireGuard VPN](https://ubuntu.com/server/docs/wireguard-vpn)
- [Red Hat RHEL 9 — Configuring a WireGuard VPN](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_and_managing_networking/assembly_setting-up-a-wireguard-vpn_configuring-and-managing-networking)
- [Kernel WireGuard Module Documentation](https://www.kernel.org/doc/html/latest/networking/wireguard.html)
