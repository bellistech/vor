# nmap (Network Mapper)

Network exploration and security auditing tool — host discovery, port scanning, service/OS detection, and scripting.

## Host Discovery

### Find live hosts
```bash
nmap -sn 192.168.1.0/24               # ping scan (no port scan)
nmap -sn 10.0.0.1-50                   # IP range
nmap -sn -PS22,80,443 192.168.1.0/24  # TCP SYN discovery on specific ports
nmap -sn -PA80 192.168.1.0/24         # TCP ACK discovery
nmap -sn -PU53 192.168.1.0/24         # UDP discovery
nmap -sn -PE 192.168.1.0/24           # ICMP echo only
nmap -Pn example.com                   # skip discovery (assume host is up)
```

### From a list
```bash
nmap -sn -iL hosts.txt                 # read targets from file
nmap -sn 192.168.1.0/24 --exclude 192.168.1.1  # exclude gateway
nmap -sn 192.168.1.0/24 --excludefile skip.txt
```

## Port Scanning

### Scan types
```bash
nmap example.com                       # default: top 1000 TCP ports
nmap -p 22,80,443 example.com         # specific ports
nmap -p 1-1024 example.com            # port range
nmap -p- example.com                  # all 65535 ports
nmap -p U:53,T:25,80 example.com     # mix TCP and UDP
nmap --top-ports 100 example.com      # top 100 most common
```

### Scan techniques
```bash
nmap -sS example.com                   # SYN scan (default, stealthy, needs root)
nmap -sT example.com                   # full TCP connect scan (no root needed)
nmap -sU example.com                   # UDP scan (slow)
nmap -sA example.com                   # ACK scan (detect firewall rules)
nmap -sN example.com                   # NULL scan (no flags)
nmap -sF example.com                   # FIN scan
nmap -sX example.com                   # Xmas scan (FIN+PSH+URG)
```

### Speed tuning
```bash
nmap -T0 example.com                   # paranoid (IDS evasion)
nmap -T1 example.com                   # sneaky
nmap -T2 example.com                   # polite
nmap -T3 example.com                   # normal (default)
nmap -T4 example.com                   # aggressive (recommended for LANs)
nmap -T5 example.com                   # insane (may miss ports)
nmap --min-rate 1000 example.com       # at least 1000 packets/sec
```

## Service and Version Detection

### Detect services
```bash
nmap -sV example.com                   # probe open ports for service info
nmap -sV --version-intensity 5 example.com  # default intensity
nmap -sV --version-all example.com     # try every probe (slow but thorough)
nmap -sV -p 22,80,443 example.com     # version scan on specific ports
```

## OS Detection

### Identify operating system
```bash
nmap -O example.com                    # OS detection (needs root)
nmap -O --osscan-guess example.com    # aggressive OS guessing
nmap -A example.com                   # OS + version + scripts + traceroute
```

## NSE Scripts

### Nmap Scripting Engine
```bash
nmap --script=default example.com              # default scripts (same as -sC)
nmap -sC example.com                           # shorthand for default scripts
nmap --script=vuln example.com                 # vulnerability scripts
nmap --script=http-title example.com           # specific script
nmap --script=http-enum example.com            # enumerate web paths
nmap --script=ssl-enum-ciphers -p 443 example.com  # TLS cipher audit
nmap --script=smb-enum-shares example.com      # SMB shares
nmap --script=dns-brute example.com            # DNS subdomain brute force
```

### Script arguments
```bash
nmap --script=http-brute --script-args 'userdb=users.txt,passdb=pass.txt' example.com
nmap --script=http-put --script-args 'http-put.url=/upload/,http-put.file=shell.php' example.com
```

### List and search scripts
```bash
ls /usr/share/nmap/scripts/
nmap --script-help=http-title
```

## Output Formats

### Save results
```bash
nmap -oN scan.txt example.com          # normal text output
nmap -oX scan.xml example.com          # XML output
nmap -oG scan.gnmap example.com        # greppable output
nmap -oA scan-results example.com      # all formats (creates .nmap, .xml, .gnmap)
```

### Append and verbose
```bash
nmap -v example.com                    # verbose
nmap -vv example.com                   # very verbose
nmap -d example.com                    # debug
nmap --reason example.com             # show why port is in each state
nmap --open example.com               # show only open ports
```

## Common Scan Profiles

### Quick network survey
```bash
nmap -sn 192.168.1.0/24
```

### Fast scan of common ports
```bash
nmap -F -T4 192.168.1.0/24            # fast mode (top 100 ports)
```

### Full audit of a single host
```bash
nmap -A -T4 -p- example.com           # all ports, OS, version, scripts, traceroute
```

### Stealth scan
```bash
nmap -sS -T2 -f --data-length 24 example.com  # fragmented, slow, padded
```

### Web server scan
```bash
nmap -sV -p 80,443,8080,8443 --script=http-title,http-headers example.com
```

### Firewall detection
```bash
nmap -sA -p 80 example.com            # ACK scan reveals filtered vs unfiltered
```

## Tips

- SYN scan (`-sS`) is the default and requires root; use `-sT` (connect scan) without root
- `-T4` is safe for most networks and much faster than default; use `-T5` only on localhost/LANs
- UDP scans (`-sU`) are very slow; combine with `--top-ports 20` to keep it manageable
- `--open` filters output to only open ports — much cleaner for large scans
- `-Pn` skips host discovery — necessary when ICMP is blocked but you know the host is up
- `-oA` saves all output formats at once — always use for scans you might want to review later
- NSE scripts can be noisy and intrusive; only run `--script=vuln` against hosts you own
- Scan only networks you have authorization to test; unauthorized scanning may be illegal
- `nmap -sV` can trigger IDS alerts — the probes are distinctive
- Greppable output (`-oG`) is ideal for piping into `awk`/`grep` for bulk analysis

## See Also

- nc, ss, tcpdump, tshark, vulnerability-scanning

## References

- [Nmap Official Documentation](https://nmap.org/docs.html)
- [Nmap Reference Guide](https://nmap.org/book/man.html)
- [man nmap](https://man7.org/linux/man-pages/man1/nmap.1.html)
- [Nmap Scripting Engine (NSE) Documentation](https://nmap.org/nsedoc/)
- [Nmap Network Scanning — Online Book](https://nmap.org/book/)
- [Nmap — Port Scanning Techniques](https://nmap.org/book/man-port-scanning-techniques.html)
- [Nmap — Service and Version Detection](https://nmap.org/book/man-version-detection.html)
- [Nmap — OS Detection](https://nmap.org/book/man-os-detection.html)
- [Ncat Users' Guide](https://nmap.org/ncat/guide/)
