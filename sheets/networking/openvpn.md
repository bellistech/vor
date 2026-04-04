# OpenVPN (SSL/TLS VPN)

OpenVPN is an open-source VPN solution using SSL/TLS for key exchange, supporting both routed (tun) and bridged (tap) modes with flexible authentication via certificates, pre-shared keys, or username/password, running over UDP or TCP on any port.

## Installation

```bash
# Debian/Ubuntu
sudo apt install openvpn easy-rsa

# RHEL/CentOS
sudo dnf install epel-release
sudo dnf install openvpn easy-rsa

# macOS
brew install openvpn

# Check version
openvpn --version
```

## PKI Setup (easy-rsa)

### Initialize PKI

```bash
# Set up easy-rsa directory
make-cadir ~/openvpn-ca
cd ~/openvpn-ca

# Initialize PKI
./easyrsa init-pki

# Build CA (will prompt for passphrase)
./easyrsa build-ca

# Generate server certificate and key
./easyrsa gen-req server nopass
./easyrsa sign-req server server

# Generate Diffie-Hellman parameters
./easyrsa gen-dh

# Generate TLS auth key (HMAC firewall)
openvpn --genkey secret ta.key

# Generate client certificate
./easyrsa gen-req client1 nopass
./easyrsa sign-req client client1

# Generate CRL (certificate revocation list)
./easyrsa gen-crl
```

### Revoke a Client Certificate

```bash
./easyrsa revoke client1
./easyrsa gen-crl
# Copy updated CRL to server
sudo cp pki/crl.pem /etc/openvpn/server/
```

## Server Configuration

### Basic server.conf (Routed / tun)

```ini
# /etc/openvpn/server/server.conf
port 1194
proto udp                              # UDP is faster, TCP for restrictive firewalls
dev tun                                # tun = routed (Layer 3), tap = bridged (Layer 2)

ca ca.crt
cert server.crt
key server.key
dh dh.pem
tls-auth ta.key 0                      # HMAC firewall (direction 0 = server)
# tls-crypt ta.key                     # alternative: encrypts control channel too

server 10.8.0.0 255.255.255.0         # VPN subnet
ifconfig-pool-persist /var/log/openvpn/ipp.txt

# Push routes to clients
push "route 192.168.1.0 255.255.255.0"
push "dhcp-option DNS 10.8.0.1"

# All traffic through VPN (full tunnel)
# push "redirect-gateway def1 bypass-dhcp"

keepalive 10 120
cipher AES-256-GCM
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305
auth SHA256
persist-key
persist-tun
status /var/log/openvpn/openvpn-status.log
log-append /var/log/openvpn/openvpn.log
verb 3
max-clients 100
user nobody
group nogroup

# Enable client-to-client communication
# client-to-client
```

### Bridged Mode (tap)

```ini
# /etc/openvpn/server/bridge.conf
dev tap0
server-bridge 192.168.1.1 255.255.255.0 192.168.1.200 192.168.1.250
push "dhcp-option DNS 192.168.1.1"
```

## Client Configuration

### Basic client.conf

```ini
# /etc/openvpn/client/client.conf
client
dev tun
proto udp
remote vpn.example.com 1194
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server

ca ca.crt
cert client1.crt
key client1.key
tls-auth ta.key 1                      # direction 1 = client

cipher AES-256-GCM
data-ciphers AES-256-GCM:AES-128-GCM:CHACHA20-POLY1305
auth SHA256
verb 3
```

### Inline Certificate Config (.ovpn)

```ini
client
dev tun
proto udp
remote vpn.example.com 1194
resolv-retry infinite
nobind
persist-key
persist-tun
remote-cert-tls server
cipher AES-256-GCM
auth SHA256
key-direction 1

<ca>
-----BEGIN CERTIFICATE-----
# CA certificate here
-----END CERTIFICATE-----
</ca>

<cert>
-----BEGIN CERTIFICATE-----
# Client certificate here
-----END CERTIFICATE-----
</cert>

<key>
-----BEGIN PRIVATE KEY-----
# Client private key here
-----END PRIVATE KEY-----
</key>

<tls-auth>
-----BEGIN OpenVPN Static key V1-----
# TLS auth key here
-----END OpenVPN Static key V1-----
</tls-auth>
```

## Split Tunnel vs Full Tunnel

```ini
# Full tunnel — ALL traffic through VPN
push "redirect-gateway def1 bypass-dhcp"
push "dhcp-option DNS 8.8.8.8"

# Split tunnel — only specific networks through VPN
push "route 10.0.0.0 255.0.0.0"
push "route 172.16.0.0 255.240.0.0"
push "route 192.168.0.0 255.255.0.0"

# Client-side route exclusion (in client config)
route 10.10.0.0 255.255.0.0 vpn_gateway
route 0.0.0.0 0.0.0.0 net_gateway        # default via local gateway
```

## Systemd Integration

```bash
# Enable and start server
sudo systemctl enable openvpn-server@server
sudo systemctl start openvpn-server@server
sudo systemctl status openvpn-server@server

# Enable and start client
sudo systemctl enable openvpn-client@client
sudo systemctl start openvpn-client@client

# View logs
journalctl -u openvpn-server@server -f
```

## IP Forwarding and NAT

```bash
# Enable IP forwarding
echo 'net.ipv4.ip_forward = 1' | sudo tee /etc/sysctl.d/99-openvpn.conf
sudo sysctl -p /etc/sysctl.d/99-openvpn.conf

# NAT (masquerade) for VPN clients
sudo iptables -t nat -A POSTROUTING -s 10.8.0.0/24 -o eth0 -j MASQUERADE
sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o tun0 -m state --state RELATED,ESTABLISHED -j ACCEPT

# Persist iptables
sudo apt install iptables-persistent
sudo netfilter-persistent save
```

## Performance Tuning

```ini
# Server-side performance options
sndbuf 393216
rcvbuf 393216
push "sndbuf 393216"
push "rcvbuf 393216"

# Use UDP for best performance
proto udp

# Fast I/O (Linux only)
fast-io

# Compression (optional, can be a security risk — VORACLE attack)
# compress lz4-v2
# push "compress lz4-v2"

# Multi-process (one per CPU core)
# Run multiple instances on different ports instead

# MTU tuning
tun-mtu 1500
fragment 1300
mssfix 1200
```

## Management Interface

```bash
# Enable management interface in server.conf
# management 127.0.0.1 7505

# Connect via telnet
telnet 127.0.0.1 7505

# Management commands
status                                 # show connected clients
kill client-name                       # disconnect a client
signal SIGUSR1                         # restart without disconnecting
signal SIGHUP                          # reload config
log on all                             # enable real-time log
```

## Troubleshooting

```bash
# Test connection with verbose logging
openvpn --config client.conf --verb 6

# Verify certificate
openssl x509 -in server.crt -text -noout

# Check TLS handshake
openvpn --config client.conf --verb 9 2>&1 | grep -i tls

# Test UDP port reachability
nc -uzv vpn.example.com 1194

# Check routing table
ip route show
netstat -rn
```

## Tips

- Use `tls-crypt` instead of `tls-auth` for newer deployments; it encrypts the entire control channel, not just HMAC
- Always use `remote-cert-tls server` on clients to prevent MITM attacks from rogue client certificates
- UDP is almost always faster than TCP; only use TCP when UDP is blocked by firewalls
- Set `verb 3` for production, `verb 6` for debugging, `verb 9` for TLS-level tracing
- Use inline `.ovpn` files for single-file client distribution instead of separate cert/key files
- Avoid compression (`compress`) in production to mitigate VORACLE-style attacks
- Generate unique certificates per client rather than sharing; makes revocation practical
- Use `persist-tun` and `persist-key` to survive temporary privilege drops after initialization
- Set `max-clients` to prevent resource exhaustion on the server
- Monitor with the management interface on localhost for real-time client status
- Use `crl-verify crl.pem` to enforce certificate revocation checking on the server

## See Also

- tailscale, zerotier, ipsec, wireguard

## References

- [OpenVPN Community Documentation](https://openvpn.net/community-resources/)
- [OpenVPN Manual (2.6)](https://openvpn.net/community-resources/reference-manual-for-openvpn-2-6/)
- [easy-rsa Documentation](https://easy-rsa.readthedocs.io/en/latest/)
- [NIST SP 800-77 Rev 1: Guide to IPsec VPNs](https://csrc.nist.gov/publications/detail/sp/800-77/rev-1/final)
