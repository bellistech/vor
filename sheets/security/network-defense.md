# Network Defense (Network security monitoring, anomaly detection, and defensive strategies)

## Traffic Baselining

### Establishing Normal Patterns

```bash
# Capture baseline traffic statistics with vnstat
vnstat --create -i eth0
# Let it collect for 24-72 hours, then review
vnstat -d -i eth0           # daily traffic summary
vnstat -h -i eth0           # hourly traffic summary

# Capture baseline with ntopng for protocol distribution
# Start ntopng on monitoring interface
ntopng -i eth0 -w 3000

# Generate traffic summary with tshark
tshark -i eth0 -q -z io,stat,60 -a duration:3600
# Produces per-minute byte/packet counts over 1 hour

# Protocol distribution baseline
tshark -i eth0 -q -z io,phs -a duration:3600

# Top talkers baseline with iftop
iftop -i eth0 -t -s 300 > /tmp/top_talkers_baseline.txt

# Connection count baseline
ss -s                        # socket summary
ss -tunap | wc -l            # total active connections
```

### Anomaly Detection Patterns

```bash
# Detect unusual DNS query volume (compare to baseline)
tcpdump -i eth0 -nn port 53 -c 10000 -w /tmp/dns_capture.pcap
tshark -r /tmp/dns_capture.pcap -q -z dns,tree
# Look for: query count spikes, unusual record types (TXT, NULL), long domain names

# Detect beaconing behavior (regular interval callbacks)
# Export connection logs, look for periodic patterns
tshark -i eth0 -T fields -e frame.time_epoch -e ip.src -e ip.dst \
  -e tcp.dstport -Y "tcp.flags.syn==1 && tcp.flags.ack==0" \
  -a duration:3600 > /tmp/syn_log.csv
# Analyze intervals between connections to same destination

# Detect port scanning
# Watch for single source hitting many ports
tcpdump -i eth0 -nn 'tcp[tcpflags] & tcp-syn != 0' -c 5000 | \
  awk '{print $3}' | sort | uniq -c | sort -rn | head -20

# Detect unusual outbound connections
ss -tunap | awk '$5 !~ /:(80|443|53)$/ {print}' | sort -t: -k2 -n

# Monitor for large data exfiltration
iftop -i eth0 -f "src net 10.0.0.0/8" -t -s 60 2>/dev/null | \
  grep -E "=>.*[0-9]+[MG]B"
```

## DNS Sinkholing

```bash
# Simple DNS sinkhole with dnsmasq
# Redirect known malicious domains to localhost
cat <<'EOF' >> /etc/dnsmasq.d/sinkhole.conf
# Sinkhole known C2 domains
address=/malicious-domain.com/127.0.0.1
address=/badsite.example.com/127.0.0.1

# Sinkhole entire TLDs if needed
address=/.tk/127.0.0.1

# Log all queries for analysis
log-queries
log-facility=/var/log/dnsmasq-queries.log
EOF

sudo systemctl restart dnsmasq

# Pi-hole as DNS sinkhole (automated blocklist management)
# Install Pi-hole
curl -sSL https://install.pi-hole.net | bash

# Add custom blocklists
pihole -b malicious-domain.com
pihole -g                          # update blocklists

# Monitor sinkhole hits
tail -f /var/log/pihole.log | grep "0.0.0.0"

# Response Policy Zone (RPZ) with BIND
# /etc/bind/named.conf.local
cat <<'EOF'
zone "rpz.local" {
    type master;
    file "/etc/bind/db.rpz.local";
    allow-query { none; };
};
EOF

# /etc/bind/db.rpz.local
cat <<'EOF'
$TTL 60
@ IN SOA localhost. admin.localhost. (1 3600 900 604800 60)
@ IN NS localhost.
; Sinkhole entries
malicious-domain.com CNAME .     ; NXDOMAIN response
*.malicious-domain.com CNAME .   ; Wildcard block
badsite.com A 127.0.0.1          ; Redirect to localhost
EOF
```

## Network Segmentation

### VLAN Configuration

```bash
# Create VLAN interfaces on Linux
sudo ip link add link eth0 name eth0.10 type vlan id 10    # Management VLAN
sudo ip link add link eth0 name eth0.20 type vlan id 20    # Production VLAN
sudo ip link add link eth0 name eth0.30 type vlan id 30    # DMZ VLAN
sudo ip link add link eth0 name eth0.99 type vlan id 99    # Guest/Quarantine

# Assign addresses and bring up
sudo ip addr add 10.10.10.1/24 dev eth0.10
sudo ip link set eth0.10 up

# Persistent VLAN config (netplan)
cat <<'EOF' > /etc/netplan/01-vlans.yaml
network:
  version: 2
  ethernets:
    eth0:
      dhcp4: false
  vlans:
    vlan10:
      id: 10
      link: eth0
      addresses: [10.10.10.1/24]
    vlan20:
      id: 20
      link: eth0
      addresses: [10.20.20.1/24]
EOF
sudo netplan apply
```

### Inter-VLAN Firewall Rules

```bash
# Allow management VLAN to reach all others, restrict reverse
sudo iptables -A FORWARD -i eth0.10 -o eth0.20 -j ACCEPT
sudo iptables -A FORWARD -i eth0.20 -o eth0.10 -m state --state ESTABLISHED,RELATED -j ACCEPT
sudo iptables -A FORWARD -i eth0.20 -o eth0.10 -j DROP

# Isolate guest VLAN — internet only, no internal access
sudo iptables -A FORWARD -i eth0.99 -o eth0 -j ACCEPT            # outbound OK
sudo iptables -A FORWARD -i eth0.99 -d 10.0.0.0/8 -j DROP        # block RFC1918
sudo iptables -A FORWARD -i eth0.99 -d 172.16.0.0/12 -j DROP
sudo iptables -A FORWARD -i eth0.99 -d 192.168.0.0/16 -j DROP

# Restrict DMZ to only serve specific ports
sudo iptables -A FORWARD -i eth0 -o eth0.30 -p tcp --dport 80 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o eth0.30 -p tcp --dport 443 -j ACCEPT
sudo iptables -A FORWARD -i eth0 -o eth0.30 -j DROP
```

## Honeypots

### Cowrie (SSH/Telnet Honeypot)

```bash
# Install Cowrie
sudo apt-get install -y git python3-venv
git clone https://github.com/cowrie/cowrie.git /opt/cowrie
cd /opt/cowrie
python3 -m venv cowrie-env
source cowrie-env/bin/activate
pip install -r requirements.txt

# Configure (redirect real SSH to alternate port first)
cp etc/cowrie.cfg.dist etc/cowrie.cfg
# Edit etc/cowrie.cfg:
#   [ssh]
#   listen_endpoints = tcp:2222:interface=0.0.0.0
#   [telnet]
#   listen_endpoints = tcp:2223:interface=0.0.0.0

# Redirect port 22 to Cowrie
sudo iptables -t nat -A PREROUTING -p tcp --dport 22 -j REDIRECT --to-port 2222

# Start Cowrie
bin/cowrie start

# Monitor Cowrie logs (JSON format)
tail -f var/log/cowrie/cowrie.json | python3 -m json.tool
```

### Dionaea (Multi-protocol Honeypot)

```bash
# Install Dionaea
sudo apt-get install -y dionaea

# Configure listening services
# /etc/dionaea/dionaea.cfg — enable SMB, HTTP, FTP, MSSQL, MySQL, SIP

# Start Dionaea
sudo systemctl start dionaea

# Monitor captured samples
ls /var/lib/dionaea/binaries/

# View connection logs
sqlite3 /var/lib/dionaea/logsql.sqlite \
  "SELECT * FROM connections ORDER BY connection_timestamp DESC LIMIT 20;"
```

## Packet Capture Strategies

```bash
# Capture on a specific interface with ring buffer (rotate files)
tcpdump -i eth0 -w /captures/traffic_%Y%m%d_%H%M%S.pcap \
  -G 3600 -W 24 -s 0 -Z root

# Capture specific traffic patterns
tcpdump -i eth0 -w /captures/dns.pcap 'port 53'
tcpdump -i eth0 -w /captures/syn.pcap 'tcp[tcpflags] & tcp-syn != 0'

# Capture with BPF filter for anomaly investigation
tcpdump -i eth0 -w /captures/suspect.pcap \
  'host 10.0.0.50 and not port 80 and not port 443'

# Full packet capture with dumpcap (Wireshark CLI)
dumpcap -i eth0 -b duration:3600 -b files:48 -w /captures/full.pcapng

# Analyze capture file
tshark -r capture.pcap -q -z conv,tcp        # TCP conversations
tshark -r capture.pcap -q -z endpoints,ip    # IP endpoints
tshark -r capture.pcap -q -z http,tree       # HTTP statistics

# Extract files from capture
tshark -r capture.pcap --export-objects http,/tmp/extracted/
```

## NetFlow and sFlow Analysis

```bash
# Collect NetFlow with nfdump/nfcapd
nfcapd -w -D -l /var/cache/nfdump -p 2055 -T all

# Query flow data
nfdump -R /var/cache/nfdump -o extended -s srcip/bytes    # top sources by bytes
nfdump -R /var/cache/nfdump -o extended -s dstport/flows  # top destination ports
nfdump -R /var/cache/nfdump -o extended -s record/bytes   # top flows

# Filter specific flows
nfdump -R /var/cache/nfdump 'dst port 22 and not src net 10.0.0.0/8'

# Detect large flows (potential exfiltration)
nfdump -R /var/cache/nfdump -o extended -s record/bytes -n 20 'bytes > 100M'

# sFlow collection with sflowtool
sflowtool -p 6343 | head -100

# Flow visualization with ntopng
ntopng -i "tcp://127.0.0.1:5556" -F "nprobe;zmq;tcp://127.0.0.1:5556"

# Bandwidth monitoring with vnstat
vnstat -l -i eth0              # live monitoring
vnstat --json d                # daily stats as JSON
```

## ARP Spoofing Detection

```bash
# Detect ARP spoofing with arpwatch
sudo apt-get install -y arpwatch
sudo arpwatch -i eth0 -f /var/lib/arpwatch/arp.dat
# Arpwatch sends email alerts on MAC/IP changes

# Manual ARP table inspection
arp -a
ip neigh show

# Detect duplicate MAC addresses (ARP spoofing indicator)
ip neigh show | awk '{print $5}' | sort | uniq -d

# Static ARP entries for critical hosts (gateway, DNS)
sudo arp -s 10.0.0.1 aa:bb:cc:dd:ee:ff

# Detect ARP anomalies with arping
arping -D -I eth0 10.0.0.1    # DAD (Duplicate Address Detection)
# If response received, someone else claims that IP

# Monitor gratuitous ARP (common in spoofing)
tcpdump -i eth0 -nn 'arp and arp[6:2] == 2' -c 100
```

## MITM Prevention

```bash
# Enable HSTS (HTTP Strict Transport Security) — web server config
# Nginx
add_header Strict-Transport-Security "max-age=63072000; includeSubDomains; preload" always;

# Enforce certificate pinning (HPKP deprecated, use CT instead)
# Certificate Transparency monitoring
# Subscribe to CT log monitors for your domains (e.g., crt.sh, Facebook CT)

# Detect SSL stripping
# Compare expected HTTPS redirects vs actual traffic
tshark -i eth0 -Y 'http.host contains "yourdomain.com"' -T fields -e http.host -e http.request.uri

# DNSSEC validation
dig +dnssec example.com
dig +cd +dnssec example.com    # check with CD flag

# Enable DNSSEC validation in resolver
# /etc/unbound/unbound.conf
cat <<'EOF'
server:
    auto-trust-anchor-file: "/var/lib/unbound/root.key"
    val-clean-additional: yes
    val-permissive-mode: no
EOF

# 802.1X port-based authentication (wpa_supplicant for wired)
cat <<'EOF' > /etc/wpa_supplicant/wpa_supplicant-eth0.conf
network={
    key_mgmt=IEEE8021X
    eap=TLS
    identity="host/workstation.example.com"
    ca_cert="/etc/pki/tls/certs/ca.pem"
    client_cert="/etc/pki/tls/certs/client.pem"
    private_key="/etc/pki/tls/private/client.key"
}
EOF
```

## Egress Filtering

```bash
# Default deny outbound, allow specific services
sudo iptables -P OUTPUT DROP

# Allow DNS (to specific resolvers only)
sudo iptables -A OUTPUT -p udp -d 10.0.0.53 --dport 53 -j ACCEPT
sudo iptables -A OUTPUT -p tcp -d 10.0.0.53 --dport 53 -j ACCEPT

# Allow HTTP/HTTPS through proxy
sudo iptables -A OUTPUT -p tcp -d 10.0.0.10 --dport 3128 -j ACCEPT

# Allow NTP
sudo iptables -A OUTPUT -p udp --dport 123 -j ACCEPT

# Allow established/related return traffic
sudo iptables -A OUTPUT -m state --state ESTABLISHED,RELATED -j ACCEPT

# Block direct outbound to known bad IP ranges
sudo iptables -A OUTPUT -d 198.51.100.0/24 -j DROP -m comment --comment "Known C2 range"

# Log dropped outbound traffic for analysis
sudo iptables -A OUTPUT -j LOG --log-prefix "EGRESS_DROP: " --log-level 4
sudo iptables -A OUTPUT -j DROP

# Monitor egress violations
tail -f /var/log/kern.log | grep "EGRESS_DROP"
```

## DDoS Mitigation Basics

```bash
# SYN flood protection (kernel tuning)
sudo sysctl -w net.ipv4.tcp_syncookies=1
sudo sysctl -w net.ipv4.tcp_max_syn_backlog=4096
sudo sysctl -w net.ipv4.tcp_synack_retries=2
sudo sysctl -w net.core.somaxconn=4096

# Rate-limit incoming connections with iptables
sudo iptables -A INPUT -p tcp --dport 80 -m connlimit --connlimit-above 50 -j DROP
sudo iptables -A INPUT -p tcp --dport 80 -m limit --limit 25/minute --limit-burst 100 -j ACCEPT

# Rate-limit ICMP (ping flood)
sudo iptables -A INPUT -p icmp --icmp-type echo-request -m limit --limit 1/s --limit-burst 4 -j ACCEPT
sudo iptables -A INPUT -p icmp --icmp-type echo-request -j DROP

# Block invalid packets
sudo iptables -A INPUT -m state --state INVALID -j DROP

# Drop fragmented packets
sudo iptables -A INPUT -f -j DROP

# Nftables equivalent (modern replacement for iptables)
cat <<'EOF'
table inet filter {
    chain input {
        type filter hook input priority 0; policy drop;
        ct state invalid drop
        ct state established,related accept
        tcp dport 80 ct count over 50 drop
        tcp dport 80 limit rate 25/minute burst 100 packets accept
        icmp type echo-request limit rate 1/second burst 4 packets accept
    }
}
EOF

# Monitor connection states during attack
ss -s
conntrack -C                   # connection tracking count
conntrack -S                   # connection tracking stats
```

## Tips

- Establish traffic baselines before you need them; you cannot detect anomalies without knowing what normal looks like.
- Segment networks with VLANs and enforce inter-VLAN filtering at the firewall; flat networks let attackers move laterally.
- Deploy honeypots on unused IP addresses within production VLANs to detect internal lateral movement.
- Use ring-buffer packet capture for continuous monitoring; keep at least 24 hours of full packet capture on critical segments.
- Implement egress filtering as strictly as the environment allows; many attacks depend on unrestricted outbound access.
- Monitor DNS traffic closely; DNS tunneling, DGA domains, and excessive TXT queries are common exfiltration channels.
- Use NetFlow/sFlow for long-term traffic analysis; full packet capture is for forensics, flow data is for trending.
- Enable SYN cookies and connection limits on internet-facing services as baseline DDoS protection.
- Static ARP entries for critical infrastructure (gateways, DNS servers) prevent ARP spoofing attacks.

## References

- [NIST SP 800-94 - Intrusion Detection and Prevention Systems](https://csrc.nist.gov/publications/detail/sp/800-94/final)
- [NIST SP 800-41 - Firewall and Firewall Policy Guidelines](https://csrc.nist.gov/publications/detail/sp/800-41/rev-1/final)
- [Cowrie SSH Honeypot](https://github.com/cowrie/cowrie)
- [Dionaea Honeypot](https://github.com/DinoTools/dionaea)
- [Wireshark / tshark Documentation](https://www.wireshark.org/docs/)
- [nfdump - NetFlow Tools](https://github.com/phaag/nfdump)
- [ntopng - Network Traffic Monitoring](https://www.ntop.org/products/traffic-analysis/ntop/)
- [arpwatch](https://ee.lbl.gov/)
- [Pi-hole - Network-wide DNS Sinkhole](https://pi-hole.net/)
- [CIS Benchmarks - Network Devices](https://www.cisecurity.org/cis-benchmarks)
