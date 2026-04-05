# Network Scanning (CEH v13 — Module 03)

> For authorized security testing, red team exercises, and educational study only.

Techniques for discovering hosts, open ports, running services, and operating systems on a target network.

## Port States

| State | Meaning |
|-------|---------|
| **open** | An application is actively accepting connections |
| **closed** | Port is accessible but no application is listening |
| **filtered** | Packet filtering prevents probe from reaching the port |
| **unfiltered** | Port is accessible but nmap cannot determine open/closed (ACK scan) |
| **open\|filtered** | Cannot determine whether port is open or filtered (UDP, FIN, XMAS, NULL) |
| **closed\|filtered** | Cannot determine whether port is closed or filtered (idle scan) |

## TCP Scan Types

### TCP Connect Scan (-sT)

Completes the full three-way handshake. Reliable but noisy — logged by the target.

```bash
nmap -sT -p 1-1000 192.168.1.0/24
```

| Probe | Open Response | Closed Response |
|-------|--------------|-----------------|
| SYN | SYN/ACK → ACK | RST/ACK |

### SYN (Half-Open) Scan (-sS)

Sends SYN, tears down with RST after receiving SYN/ACK. Default scan (requires root). Faster and stealthier than connect scan.

```bash
sudo nmap -sS -p- 10.0.0.1
```

| Probe | Open | Closed | Filtered |
|-------|------|--------|----------|
| SYN | SYN/ACK | RST | No response / ICMP unreachable |

### FIN Scan (-sF)

Sends a bare FIN flag. Exploits RFC 793: open ports should ignore it, closed ports reply RST.

```bash
sudo nmap -sF -p 22,80,443 10.0.0.1
```

| Probe | Open\|Filtered | Closed |
|-------|---------------|--------|
| FIN | No response | RST |

### XMAS Scan (-sX)

Sets FIN, PSH, and URG flags (lit up "like a Christmas tree").

```bash
sudo nmap -sX 10.0.0.1
```

| Probe | Open\|Filtered | Closed |
|-------|---------------|--------|
| FIN+PSH+URG | No response | RST |

### NULL Scan (-sN)

Sends a TCP packet with no flags set.

```bash
sudo nmap -sN 10.0.0.1
```

| Probe | Open\|Filtered | Closed |
|-------|---------------|--------|
| No flags | No response | RST |

### ACK Scan (-sA)

Sends only ACK. Does not determine open/closed — used to map firewall rulesets (stateful vs stateless).

```bash
sudo nmap -sA -p 80,443 10.0.0.1
```

| Probe | Unfiltered | Filtered |
|-------|-----------|----------|
| ACK | RST | No response / ICMP unreachable |

### Window Scan (-sW)

Like ACK scan but examines the TCP window size in RST responses. Some implementations return a positive window for open ports.

```bash
sudo nmap -sW -p 22,80 10.0.0.1
```

### Maimon Scan (-sM)

Sends FIN/ACK. Named after Uriel Maimon. Some BSD-derived stacks drop the packet for open ports instead of sending RST.

```bash
sudo nmap -sM 10.0.0.1
```

## UDP Scanning (-sU)

UDP scanning is slow and unreliable. No handshake means open ports usually give no response.

```bash
# Basic UDP scan (top 1000 ports)
sudo nmap -sU 10.0.0.1

# Combined TCP SYN + UDP scan
sudo nmap -sS -sU -p T:80,443,U:53,161 10.0.0.1

# Speed up with version detection to confirm open
sudo nmap -sU --version-intensity 0 -p 53,161,500 10.0.0.1
```

| Probe | Open | Closed | Filtered |
|-------|------|--------|----------|
| UDP datagram | Response or no response (open\|filtered) | ICMP port unreachable | ICMP unreachable (other) |

**Challenges:** ICMP rate limiting (Linux: 1/sec) makes full scans take 18+ hours. Use `--min-rate` or target specific ports.

## Network Topology Discovery

```bash
# Ping sweep — ICMP echo
nmap -sn 192.168.1.0/24

# TCP SYN ping (bypasses ICMP-blocking firewalls)
nmap -sn -PS80,443 10.0.0.0/24

# TCP ACK ping
nmap -sn -PA80 10.0.0.0/24

# UDP ping
nmap -sn -PU53 10.0.0.0/24

# ARP scan (local subnet only, most reliable)
nmap -sn -PR 192.168.1.0/24
sudo arp-scan -l

# Traceroute
nmap -sn --traceroute 10.0.0.1
traceroute -T -p 80 10.0.0.1    # TCP traceroute
```

## OS Fingerprinting

### Active (-O)

Sends crafted probes and analyzes responses (TTL, window size, DF bit, TCP options order).

```bash
# OS detection
sudo nmap -O 10.0.0.1

# Aggressive OS detection (more guesses)
sudo nmap -O --osscan-guess 10.0.0.1

# Combined scan
sudo nmap -A 10.0.0.1   # -O + -sV + -sC + --traceroute
```

### Passive

Sniff traffic and fingerprint without sending probes.

```bash
# p0f — passive OS fingerprinting
sudo p0f -i eth0

# Analyze a pcap
p0f -r capture.pcap
```

Key TCP/IP stack indicators: initial TTL (Linux 64, Windows 128, Cisco 255), window size, DF bit, TCP options (MSS, SACK, timestamps, window scaling, NOP ordering).

## Service and Version Detection

```bash
# Version detection
nmap -sV 10.0.0.1

# Aggressive version detection
nmap -sV --version-intensity 9 10.0.0.1

# Banner grabbing — manual
echo "" | nc -nv 10.0.0.1 80
curl -sI http://10.0.0.1

# Banner grabbing — nmap
nmap -sV --script=banner -p 21,22,80 10.0.0.1
```

## Nmap Scripting Engine (NSE)

Scripts live in `/usr/share/nmap/scripts/`. Categories: `auth`, `broadcast`, `brute`, `default`, `discovery`, `dos`, `exploit`, `external`, `fuzzer`, `intrusive`, `malware`, `safe`, `version`, `vuln`.

```bash
# Default scripts
nmap -sC 10.0.0.1

# Specific script
nmap --script=http-title 10.0.0.1

# Category
nmap --script=vuln 10.0.0.1

# Multiple categories
nmap --script="safe and discovery" 10.0.0.1

# Script arguments
nmap --script=http-brute --script-args http-brute.path=/admin 10.0.0.1

# Update script database
nmap --script-updatedb

# Useful scripts
nmap --script=smb-os-discovery 10.0.0.1
nmap --script=dns-brute example.com
nmap --script=ssl-enum-ciphers -p 443 10.0.0.1
nmap --script=http-enum 10.0.0.1
nmap --script=vuln -p 445 10.0.0.1
```

Custom NSE scripts are written in Lua with `description`, `categories`, `action`, and optional `portrule`/`hostrule` functions.

## Scan Evasion Techniques

```bash
# Fragmentation (split probes into 8-byte fragments)
sudo nmap -f 10.0.0.1
sudo nmap --mtu 16 10.0.0.1    # custom MTU (must be multiple of 8)

# Decoys (mix your scan with spoofed source IPs)
sudo nmap -D RND:10 10.0.0.1
sudo nmap -D 10.0.0.2,10.0.0.3,ME 10.0.0.1

# Timing (T0=paranoid, T1=sneaky, T2=polite, T3=normal, T4=aggressive, T5=insane)
nmap -T1 10.0.0.1
nmap --max-rate 10 10.0.0.1

# Source port manipulation (use trusted ports)
sudo nmap --source-port 53 10.0.0.1
sudo nmap -g 88 10.0.0.1

# Idle scan (zombie/side-channel — completely blind)
sudo nmap -sI zombie-host:80 10.0.0.1

# MAC address spoofing (must be on same subnet)
sudo nmap --spoof-mac 00:11:22:33:44:55 10.0.0.1
sudo nmap --spoof-mac Dell 10.0.0.1

# Append random data to packets
nmap --data-length 50 10.0.0.1

# Bad checksum (some firewalls don't verify)
nmap --badsum 10.0.0.1
```

## Other Scanning Tools

```bash
# masscan — fastest port scanner (async SYN, own TCP stack)
sudo masscan -p1-65535 10.0.0.0/24 --rate=10000
sudo masscan -p80,443 0.0.0.0/0 --rate=100000 --excludefile exclude.txt

# rustscan — fast port discovery, pipes to nmap
rustscan -a 10.0.0.1 -- -sV -sC

# zmap — single-port internet-wide scanning
sudo zmap -p 80 10.0.0.0/24 -o results.csv

# unicornscan — async stateless scanning
sudo unicornscan -mT -p1-65535 10.0.0.1

# hping3 — craft custom packets
sudo hping3 -S -p 80 10.0.0.1            # SYN scan
sudo hping3 -F -p 80 10.0.0.1            # FIN scan
sudo hping3 --scan 1-1000 -S 10.0.0.1    # port range
sudo hping3 -1 10.0.0.1                  # ICMP ping
sudo hping3 -2 -p 53 10.0.0.1            # UDP
```

## Countermeasures

| Technique | Defense |
|-----------|---------|
| Port scanning | IDS/IPS rules (Snort: `threshold type both, track by_src, count 25, seconds 5`) |
| SYN scan | Stateful firewall, SYN cookies |
| Stealth scans (FIN/XMAS/NULL) | Stateful packet inspection |
| OS fingerprinting | Modify default TTL, window size; use OS fingerprint scrubbers |
| Banner grabbing | Remove/modify service banners |
| General | Rate limiting, port knocking, honeypots, network segmentation |
| Decoy detection | Reverse path filtering (uRPF), TTL analysis |

```bash
# Snort rule — detect port scan
alert tcp any any -> $HOME_NET any (msg:"Port scan detected"; \
  flags:S; threshold:type both, track by_src, count 25, seconds 5; sid:1000001;)

# iptables — rate limit SYN
iptables -A INPUT -p tcp --syn -m limit --limit 10/s --limit-burst 20 -j ACCEPT
iptables -A INPUT -p tcp --syn -j DROP

# Port knocking example (knockd)
# Sequence: 7000, 8000, 9000 → opens port 22
```

## Practical Examples

```bash
# Full recon scan
sudo nmap -sS -sV -O -sC -p- -T4 -oA full_scan 10.0.0.1

# Quick top-ports scan
nmap --top-ports 100 -T4 10.0.0.1

# Scan through a proxy
nmap --proxy socks4://127.0.0.1:9050 10.0.0.1

# Output formats
nmap -oN normal.txt -oX output.xml -oG grepable.txt 10.0.0.1
nmap -oA all_formats 10.0.0.1    # all three at once

# Scan a range
nmap 10.0.0.1-254
nmap -iL targets.txt

# Exclude hosts
nmap 10.0.0.0/24 --exclude 10.0.0.1
nmap 10.0.0.0/24 --excludefile skip.txt

# Resume interrupted scan
nmap --resume scan.xml
```

## Tips

- Always start with a ping sweep (`-sn`) to identify live hosts before port scanning.
- Use `-Pn` to skip host discovery when targets are known to block pings.
- SYN scan (`-sS`) is the default when running as root; connect scan (`-sT`) is the fallback for unprivileged users.
- Combine `-sS` and `-sU` to scan both TCP and UDP in one pass.
- FIN, XMAS, and NULL scans do not work against Windows (Windows sends RST for all, violating RFC 793).
- The idle scan (`-sI`) is the only truly blind scan — your IP never appears in the target's logs.
- Use `-oA` to save output in all formats; the XML output feeds well into tools like searchsploit and Metasploit.
- For large networks, use masscan for port discovery, then nmap for version/script scanning on discovered ports.
- Timing template `-T4` is good for fast scans on reliable networks; use `-T1` or `-T2` for stealth.

## See Also

- `sheets/offensive/enumeration.md` — post-scan enumeration techniques
- `sheets/offensive/vulnerability-analysis.md` — using scan results for vuln assessment
- `sheets/defensive/ids-ips.md` — intrusion detection and scan prevention

## References

- CEH v13 Official Study Guide — Module 03: Scanning Networks
- RFC 793 — Transmission Control Protocol
- nmap.org — Nmap Reference Guide (https://nmap.org/book/man.html)
- Gordon "Fyodor" Lyon — *Nmap Network Scanning* (nmap.org/book/)
- p0f v3 documentation (https://lcamtuf.coredump.cx/p0f3/)
- SANS — Port Scanning Techniques (https://www.sans.org/white-papers/)
