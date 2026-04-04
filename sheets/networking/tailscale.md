# Tailscale (WireGuard Mesh VPN)

Tailscale builds a peer-to-peer mesh VPN on top of WireGuard, providing zero-configuration encrypted networking with NAT traversal, identity-based access controls, MagicDNS, and automatic key rotation across all major platforms.

## Installation

```bash
# Linux (official script)
curl -fsSL https://tailscale.com/install.sh | sh

# Debian/Ubuntu
curl -fsSL https://pkgs.tailscale.com/stable/ubuntu/jammy.noarmor.gpg | sudo tee /usr/share/keyrings/tailscale-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/tailscale-archive-keyring.gpg] https://pkgs.tailscale.com/stable/ubuntu jammy main" | sudo tee /etc/apt/sources.list.d/tailscale.list
sudo apt update && sudo apt install tailscale

# macOS
brew install tailscale

# Docker
docker run -d --name=tailscale \
  --cap-add=NET_ADMIN --cap-add=SYS_MODULE \
  -v /dev/net/tun:/dev/net/tun \
  -v tailscale-state:/var/lib/tailscale \
  tailscale/tailscale
```

## Core Commands

```bash
# Connect to tailnet
sudo tailscale up

# Disconnect
sudo tailscale down

# Check status
tailscale status

# Show current node IP addresses
tailscale ip
tailscale ip -4                        # IPv4 only
tailscale ip -6                        # IPv6 only

# Show detailed connection info
tailscale status --json | jq .

# Ping a node (checks direct vs DERP relay)
tailscale ping <hostname-or-ip>

# Network diagnostic
tailscale netcheck

# Show current Tailscale version
tailscale version

# Re-authenticate
sudo tailscale up --force-reauth

# Logout
sudo tailscale logout
```

## Subnet Routing

```bash
# Advertise local subnets to the tailnet
sudo tailscale up --advertise-routes=192.168.1.0/24,10.0.0.0/8

# Accept routes from subnet routers (on other nodes)
sudo tailscale up --accept-routes

# Enable IP forwarding (required on subnet router)
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf
echo 'net.ipv6.conf.all.forwarding = 1' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Approve advertised routes in admin console or via CLI
# Routes must be approved in the admin panel at https://login.tailscale.com/admin/machines
```

## Exit Nodes

```bash
# Advertise this machine as an exit node
sudo tailscale up --advertise-exit-node

# Use a specific exit node (route all traffic through it)
sudo tailscale up --exit-node=<exit-node-ip-or-hostname>

# Use exit node and allow LAN access
sudo tailscale up --exit-node=<exit-node> --exit-node-allow-lan-access

# Stop using exit node
sudo tailscale up --exit-node=
```

## MagicDNS

```bash
# MagicDNS is enabled by default in the admin console
# Access machines by name
ping myserver                          # short name
ping myserver.tail12345.ts.net         # FQDN

# Check DNS configuration
tailscale dns status

# Set custom DNS nameservers (admin console or API)
# Split DNS: route specific domains to internal resolvers

# Show the tailnet domain
tailscale status --json | jq -r '.MagicDNSSuffix'
```

## ACL Policies (HuJSON)

```jsonc
// Example ACL policy (admin console > Access Controls)
{
  "groups": {
    "group:engineering": ["user1@example.com", "user2@example.com"],
    "group:devops": ["admin@example.com"]
  },
  "tagOwners": {
    "tag:server": ["group:devops"],
    "tag:monitoring": ["group:devops"]
  },
  "acls": [
    // Engineering can access servers on specific ports
    {
      "action": "accept",
      "src": ["group:engineering"],
      "dst": ["tag:server:22,443,8080"]
    },
    // DevOps has full access
    {
      "action": "accept",
      "src": ["group:devops"],
      "dst": ["*:*"]
    },
    // All users can reach monitoring dashboards
    {
      "action": "accept",
      "src": ["*"],
      "dst": ["tag:monitoring:3000,9090"]
    }
  ],
  "ssh": [
    {
      "action": "accept",
      "src": ["group:devops"],
      "dst": ["tag:server"],
      "users": ["root", "ubuntu"]
    }
  ]
}
```

## Tailscale SSH

```bash
# Enable Tailscale SSH on a node
sudo tailscale up --ssh

# Connect via Tailscale SSH (no key management needed)
ssh user@hostname

# SSH ACLs are defined in the policy file (see above)
# Tailscale handles key distribution and certificate rotation
```

## Taildrop (File Sharing)

```bash
# Send a file to another node
tailscale file cp myfile.txt hostname:

# Receive files (default: ~/Tailscale/)
tailscale file get .

# Send multiple files
tailscale file cp *.log hostname:
```

## DERP Relay Servers

```bash
# Check which DERP relay is being used
tailscale netcheck

# Show DERP map
tailscale debug derp-map

# Custom DERP server (in ACL policy)
# {
#   "derpMap": {
#     "Regions": {
#       "900": {
#         "RegionID": 900,
#         "RegionCode": "myderp",
#         "Nodes": [{
#           "Name": "myderp1",
#           "HostName": "derp.example.com",
#           "DERPPort": 443
#         }]
#       }
#     }
#   }
# }
```

## Headscale (Self-Hosted Control Plane)

```bash
# Install headscale
wget https://github.com/juanfont/headscale/releases/latest/download/headscale_linux_amd64
chmod +x headscale_linux_amd64
sudo mv headscale_linux_amd64 /usr/local/bin/headscale

# Create config
sudo mkdir -p /etc/headscale
sudo headscale generate config > /etc/headscale/config.yaml

# Start headscale
sudo headscale serve

# Create a user (namespace)
headscale users create myuser

# Generate a pre-auth key
headscale preauthkeys create --user myuser --reusable --expiration 24h

# Register a node
headscale nodes register --user myuser --key nodekey:abc123

# Connect client to headscale
sudo tailscale up --login-server https://headscale.example.com
```

## Tailscale API

```bash
# List devices
curl -s -H "Authorization: Bearer tskey-api-xxxx" \
  https://api.tailscale.com/api/v2/tailnet/-/devices | jq .

# Get device details
curl -s -H "Authorization: Bearer tskey-api-xxxx" \
  https://api.tailscale.com/api/v2/device/<deviceID> | jq .

# Authorize a device
curl -X POST -H "Authorization: Bearer tskey-api-xxxx" \
  -H "Content-Type: application/json" \
  -d '{"authorized": true}' \
  https://api.tailscale.com/api/v2/device/<deviceID>/authorized

# Delete a device
curl -X DELETE -H "Authorization: Bearer tskey-api-xxxx" \
  https://api.tailscale.com/api/v2/device/<deviceID>

# Get ACL policy
curl -s -H "Authorization: Bearer tskey-api-xxxx" \
  https://api.tailscale.com/api/v2/tailnet/-/acl | jq .
```

## Systemd Management

```bash
sudo systemctl enable tailscaled
sudo systemctl start tailscaled
sudo systemctl status tailscaled
journalctl -u tailscaled -f
```

## Tips

- Use `tailscale ping` to verify whether connections are direct (peer-to-peer) or relayed through DERP
- Tag machines with `--advertise-tags=tag:server` for ACL-based access without tying to user identity
- Enable Tailscale SSH to eliminate SSH key management entirely; Tailscale handles certs
- Use pre-auth keys (`tailscale up --authkey=tskey-auth-xxx`) for automated/headless provisioning
- Split DNS lets you resolve internal domains via corporate DNS while using Tailscale for everything else
- Headscale is a solid self-hosted alternative if you need full control over the coordination server
- Set `--accept-dns=false` if Tailscale DNS conflicts with local resolver configuration
- Use `tailscale netcheck` to diagnose connectivity issues and verify UDP hole-punching works
- ACL policies use HuJSON (JSON with comments and trailing commas) for readability
- Funnel exposes a local service to the public internet through Tailscale without port forwarding

## See Also

- openvpn, zerotier, wireguard, coredns

## References

- [Tailscale Documentation](https://tailscale.com/kb/)
- [Tailscale ACL Policy Reference](https://tailscale.com/kb/1018/acls/)
- [Headscale GitHub](https://github.com/juanfont/headscale)
- [Tailscale API Reference](https://tailscale.com/api)
- [How Tailscale Works](https://tailscale.com/blog/how-tailscale-works/)
