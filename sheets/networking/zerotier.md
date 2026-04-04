# ZeroTier (Software-Defined Networking)

ZeroTier is a peer-to-peer software-defined virtual network layer that creates flat Ethernet networks spanning any infrastructure, using planet/moon topology for root server hierarchy, cryptographic addressing, and flow rules for microsegmentation.

## Installation

```bash
# Linux (official installer)
curl -s https://install.zerotier.com | sudo bash

# Debian/Ubuntu
sudo apt install gpg
curl -s https://raw.githubusercontent.com/zerotier/ZeroTierOne/main/doc/contact%40zerotier.com.gpg | gpg --dearmor | sudo tee /usr/share/keyrings/zerotier-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/zerotier-archive-keyring.gpg] https://download.zerotier.com/debian/bookworm bookworm main" | sudo tee /etc/apt/sources.list.d/zerotier.list
sudo apt update && sudo apt install zerotier-one

# macOS
brew install zerotier-one

# Docker
docker run -d --name zerotier \
  --device /dev/net/tun \
  --cap-add NET_ADMIN \
  -v zerotier-one:/var/lib/zerotier-one \
  zerotier/zerotier:latest

# Check version
zerotier-cli info
```

## Core CLI Commands

```bash
# Show node status (address, version, online status)
sudo zerotier-cli info

# Join a network
sudo zerotier-cli join <network-id>

# Leave a network
sudo zerotier-cli leave <network-id>

# List joined networks
sudo zerotier-cli listnetworks

# List peers (direct and relay connections)
sudo zerotier-cli listpeers

# Show peer detail
sudo zerotier-cli peers

# Get local node address (10-digit hex)
sudo zerotier-cli info | awk '{print $3}'

# Dump local node identity
sudo zerotier-cli dump
```

## Network Management (Central API)

```bash
# Create a network
curl -s -X POST -H "Authorization: token $ZT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config":{"name":"my-network","private":true}}' \
  https://api.zerotier.com/api/v1/network | jq .

# List networks
curl -s -H "Authorization: token $ZT_TOKEN" \
  https://api.zerotier.com/api/v1/network | jq '.[] | {id, config: {name: .config.name}}'

# Get network details
curl -s -H "Authorization: token $ZT_TOKEN" \
  https://api.zerotier.com/api/v1/network/<network-id> | jq .

# Authorize a member
curl -s -X POST -H "Authorization: token $ZT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config":{"authorized":true}}' \
  "https://api.zerotier.com/api/v1/network/<network-id>/member/<member-id>"

# List members
curl -s -H "Authorization: token $ZT_TOKEN" \
  "https://api.zerotier.com/api/v1/network/<network-id>/member" | jq '.[] | {nodeId, config: {authorized: .config.authorized, ipAssignments: .config.ipAssignments}}'

# Assign IP to a member
curl -s -X POST -H "Authorization: token $ZT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config":{"ipAssignments":["10.147.20.50"]}}' \
  "https://api.zerotier.com/api/v1/network/<network-id>/member/<member-id>"

# Delete a member
curl -s -X DELETE -H "Authorization: token $ZT_TOKEN" \
  "https://api.zerotier.com/api/v1/network/<network-id>/member/<member-id>"
```

## IP Assignment and Managed Routes

```bash
# Auto-assign IP pools (via API or Central UI)
# Network config: ipAssignmentPools
curl -s -X POST -H "Authorization: token $ZT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "config": {
      "ipAssignmentPools": [
        {"ipRangeStart": "10.147.20.1", "ipRangeEnd": "10.147.20.254"}
      ],
      "routes": [
        {"target": "10.147.20.0/24"},
        {"target": "192.168.1.0/24", "via": "10.147.20.1"}
      ]
    }
  }' \
  "https://api.zerotier.com/api/v1/network/<network-id>"

# Enable IP forwarding on bridge/router node
echo 'net.ipv4.ip_forward = 1' | sudo tee -a /etc/sysctl.conf
sudo sysctl -p

# Allow bridge mode on a member
curl -s -X POST -H "Authorization: token $ZT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config":{"activeBridge":true}}' \
  "https://api.zerotier.com/api/v1/network/<network-id>/member/<member-id>"
```

## Planet and Moon Topology

```bash
# Planet = global root servers (managed by ZeroTier Inc.)
# Moon = custom root servers (self-hosted, for private infrastructure)

# Generate moon config from existing identity
sudo zerotier-idtool initmoon /var/lib/zerotier-one/identity.public > moon.json

# Edit moon.json — add stable endpoints
# "stableEndpoints": ["203.0.113.10/9993"]

# Generate moon file
sudo zerotier-idtool genmoon moon.json

# Install moon on the root server
sudo mkdir -p /var/lib/zerotier-one/moons.d
sudo mv 000000*.moon /var/lib/zerotier-one/moons.d/
sudo systemctl restart zerotier-one

# Orbit a moon from other nodes
sudo zerotier-cli orbit <moon-id> <moon-id>

# Deorbit a moon
sudo zerotier-cli deorbit <moon-id>
```

## Flow Rules

```text
# Network flow rules (set via Central UI or API)
# Default: allow all
drop
  not ethertype ipv4
  and not ethertype arp
  and not ethertype ipv6
;

# Allow only TCP/443 and ICMP
accept
  ipprotocol tcp
  and dport 443
;

accept
  ipprotocol icmp
;

# Tag-based rules
tag department
  id 1
  enum 100 engineering
  enum 200 marketing
  enum 300 finance
;

# Only engineering can reach servers
accept
  tdiff department 0
;

# Drop everything else
drop;
```

## Self-Hosted Controller (ztncui)

```bash
# Install ztncui (web UI for self-hosted controller)
git clone https://github.com/key-networks/ztncui.git
cd ztncui/src
npm install

# Configure
cp .env.example .env
# Edit .env: set ZT_TOKEN, HTTPS_PORT, etc.

# Generate ZT API token
sudo cat /var/lib/zerotier-one/authtoken.secret

# Start the controller UI
npm start
# Access at https://localhost:3443
```

## Systemd Management

```bash
sudo systemctl enable zerotier-one
sudo systemctl start zerotier-one
sudo systemctl status zerotier-one
sudo systemctl restart zerotier-one
journalctl -u zerotier-one -f
```

## Multicast and Broadcast

```bash
# ZeroTier supports multicast and broadcast (unlike many VPNs)
# Multicast is enabled by default on private networks
# Multicast limit can be set per network:

curl -s -X POST -H "Authorization: token $ZT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"config":{"multicastLimit":64}}' \
  "https://api.zerotier.com/api/v1/network/<network-id>"

# Useful for: mDNS/Bonjour, DHCP, ARP, service discovery
# Warning: high multicast rates can increase bandwidth usage
```

## Troubleshooting

```bash
# Check connectivity
sudo zerotier-cli info                 # should show "ONLINE"
sudo zerotier-cli listnetworks         # check network status (OK/ACCESS_DENIED)

# Check peer connections (DIRECT vs RELAY)
sudo zerotier-cli peers

# Verify port 9993/UDP is open
sudo ss -ulnp | grep 9993

# Check local service
sudo systemctl status zerotier-one
sudo journalctl -u zerotier-one --since "10 min ago"

# Reset node identity (last resort)
sudo systemctl stop zerotier-one
sudo rm /var/lib/zerotier-one/identity.*
sudo systemctl start zerotier-one
```

## Tips

- ZeroTier networks are identified by 16-character hex IDs; always keep these handy
- Use private networks (default) and explicitly authorize members to prevent unauthorized joins
- Moons reduce latency for geographically clustered nodes by acting as local root servers
- Flow rules provide microsegmentation at the network level; use tags for role-based access
- Enable `activeBridge` on gateway nodes to bridge ZeroTier and physical LAN segments
- The default MTU is 2800 for the virtual interface; ZeroTier handles fragmentation transparently
- ZeroTier supports multicast and broadcast, making it suitable for protocols like mDNS and DHCP
- Use managed routes with `via` to route traffic to physical subnets through a ZeroTier bridge node
- Port 9993/UDP must be reachable for direct peer connections; blocked ports force relay through roots
- Self-hosted controllers (ztncui or ZeroTier's own controller) give full sovereignty over the network

## See Also

- tailscale, openvpn, wireguard, coredns

## References

- [ZeroTier Documentation](https://docs.zerotier.com/)
- [ZeroTier Knowledge Base](https://zerotier.atlassian.net/wiki/spaces/SD/overview)
- [ZeroTier Flow Rules](https://docs.zerotier.com/zerotier/rules/)
- [ZeroTier Central API](https://docs.zerotier.com/central/v1/)
- [ztncui GitHub](https://github.com/key-networks/ztncui)
