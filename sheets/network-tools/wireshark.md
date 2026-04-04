# Wireshark (Network Protocol Analyzer)

Capture and interactively analyze network traffic at the packet level.

## Capture Filters (BPF Syntax)

```bash
# Capture filters use Berkeley Packet Filter syntax
# Applied BEFORE packets are captured (reduces file size)

# By host
host 192.168.1.100
src host 10.0.0.1
dst host 10.0.0.1

# By network
net 192.168.1.0/24
src net 10.0.0.0/8

# By port
port 80
dst port 443
portrange 8000-9000

# By protocol
tcp
udp
icmp
arp

# Compound filters
tcp port 80 and host 192.168.1.100
not arp and not broadcast
tcp port 25 and (host 10.0.0.1 or host 10.0.0.2)
(tcp[tcpflags] & tcp-syn) != 0        # SYN packets only
tcp[tcpflags] & (tcp-syn|tcp-fin) != 0 # SYN or FIN

# VLAN tagged traffic
vlan and host 192.168.1.100

# Payload size filter
greater 1000          # packets > 1000 bytes
less 64               # packets < 64 bytes
```

## Display Filters (Wireshark Syntax)

```bash
# Display filters use Wireshark's own syntax
# Applied AFTER capture (filter what you see)

# IP filters
ip.addr == 192.168.1.100
ip.src == 10.0.0.1
ip.dst == 10.0.0.1
ip.addr == 192.168.1.0/24
!(ip.addr == 10.0.0.1)

# TCP filters
tcp.port == 80
tcp.dstport == 443
tcp.srcport == 1024-65535
tcp.flags.syn == 1
tcp.flags.reset == 1
tcp.flags.fin == 1
tcp.analysis.retransmission
tcp.analysis.duplicate_ack
tcp.analysis.zero_window
tcp.stream eq 5
tcp.len > 0

# HTTP filters
http.request
http.response
http.request.method == "POST"
http.request.uri contains "/api"
http.response.code == 200
http.response.code >= 400
http.host == "example.com"
http.content_type contains "json"

# DNS filters
dns
dns.qr == 0               # queries only
dns.qr == 1               # responses only
dns.qry.name == "example.com"
dns.qry.type == 1          # A record
dns.qry.type == 28         # AAAA record
dns.flags.rcode != 0       # DNS errors

# TLS filters
tls.handshake
tls.handshake.type == 1    # Client Hello
tls.handshake.type == 2    # Server Hello
tls.record.version == 0x0303  # TLS 1.2
tls.handshake.extensions.server_name == "example.com"  # SNI

# SMTP filters
smtp
smtp.req.command == "MAIL"
smtp.response.code == 250

# Combining filters
http.request and ip.dst == 10.0.0.1
dns.qr == 0 and !(dns.qry.name contains "local")
tcp.flags.syn == 1 and tcp.flags.ack == 0   # SYN only (new connections)
```

## tshark CLI

```bash
# Capture to file
tshark -i eth0 -w capture.pcapng
tshark -i eth0 -f "tcp port 80" -w http_traffic.pcapng

# Capture with ring buffer (5 files of 100MB each)
tshark -i eth0 -b filesize:102400 -b files:5 -w ring.pcapng

# Read and filter a capture file
tshark -r capture.pcapng -Y "http.request"
tshark -r capture.pcapng -Y "dns" -T fields -e dns.qry.name -e dns.a

# Output specific fields
tshark -r capture.pcapng -Y "http.request" \
  -T fields -e ip.src -e http.host -e http.request.uri

# Statistics: protocol hierarchy
tshark -r capture.pcapng -qz io,phs

# Statistics: conversations
tshark -r capture.pcapng -qz conv,tcp
tshark -r capture.pcapng -qz conv,ip

# Statistics: endpoints
tshark -r capture.pcapng -qz endpoints,ip

# Export HTTP objects
tshark -r capture.pcapng --export-objects http,/tmp/http_objects/

# JSON output
tshark -r capture.pcapng -Y "http.request" -T json

# Continuous live capture with display filter
tshark -i eth0 -Y "tcp.analysis.retransmission" -T fields \
  -e frame.time -e ip.src -e ip.dst -e tcp.stream

# Capture on multiple interfaces
tshark -i eth0 -i eth1 -w multi.pcapng

# Decrypt TLS with key log file
tshark -r capture.pcapng -o tls.keylog_file:/path/to/sslkeylog.txt \
  -Y "http" -T fields -e http.request.uri
```

## Following Streams

```bash
# In Wireshark GUI:
# Right-click packet -> Follow -> TCP Stream / HTTP Stream / TLS Stream

# tshark: extract a TCP stream
tshark -r capture.pcapng -qz follow,tcp,ascii,5
# Stream index 5, ASCII output

# Follow HTTP stream
tshark -r capture.pcapng -qz follow,http,ascii,0

# Follow TLS stream (requires key log)
tshark -r capture.pcapng \
  -o tls.keylog_file:sslkeylog.txt \
  -qz follow,tls,ascii,0
```

## Statistics and Analysis

```bash
# I/O graph data (packets per interval)
tshark -r capture.pcapng -qz io,stat,1
# 1-second intervals

# I/O with filter
tshark -r capture.pcapng -qz io,stat,1,"COUNT(frame)frame","COUNT(frame)tcp.analysis.retransmission"

# Expert info (warnings, errors)
tshark -r capture.pcapng -qz expert

# HTTP request/response stats
tshark -r capture.pcapng -qz http,tree
tshark -r capture.pcapng -qz http_req,tree
tshark -r capture.pcapng -qz http_srv,tree

# DNS response time stats
tshark -r capture.pcapng -qz dns,tree

# TCP stream timing
tshark -r capture.pcapng -qz rtp,streams   # for VoIP
```

## TLS Decryption

```bash
# Method 1: SSLKEYLOGFILE (works with all ciphers)
# Set environment variable BEFORE starting the application
export SSLKEYLOGFILE=/tmp/sslkeys.log

# Start browser or curl
curl -v https://example.com

# In Wireshark: Edit -> Preferences -> Protocols -> TLS
# (Pre)-Master-Secret log filename: /tmp/sslkeys.log

# Method 2: RSA private key (only for RSA key exchange, not ECDHE)
# Edit -> Preferences -> Protocols -> TLS -> RSA keys list
# IP: any, Port: 443, Protocol: http, Key file: server.key

# tshark with key log
tshark -r encrypted.pcapng \
  -o tls.keylog_file:/tmp/sslkeys.log \
  -Y http -T fields -e http.host -e http.request.uri
```

## Remote Capture

```bash
# sshdump (capture on remote host via SSH)
# In Wireshark: Capture -> Options -> Manage Interfaces -> Remote
# Or use sshdump extcap:
wireshark -k -i sshdump -o "extcap.sshdump.remotehost:192.168.1.1" \
  -o "extcap.sshdump.remoteusername:root" \
  -o "extcap.sshdump.remoteinterface:eth0" \
  -o "extcap.sshdump.remotecapturecommand:tcpdump"

# Manual remote capture via SSH pipe
ssh root@remote "tcpdump -U -i eth0 -w - 'not port 22'" | wireshark -k -i -

# Or save remotely, copy later
ssh root@remote "tcpdump -i eth0 -w /tmp/capture.pcapng -c 10000 'port 80'"
scp root@remote:/tmp/capture.pcapng .
```

## Profiles and Coloring Rules

```bash
# Profiles: separate configurations for different tasks
# Edit -> Configuration Profiles -> New
# Stored in: ~/.config/wireshark/profiles/<name>/

# Coloring rules (Edit -> Coloring Rules)
# Priority order (first match wins):
# Name          | Filter                          | FG      | BG
# Bad TCP       | tcp.analysis.flags              | black   | red
# HTTP Errors   | http.response.code >= 400       | black   | orange
# DNS           | dns                             | black   | lightblue
# SYN/FIN       | tcp.flags.syn==1||tcp.flags.fin==1 | black | yellow
# Retransmit    | tcp.analysis.retransmission     | white   | darkred

# Export/import coloring rules
# ~/.config/wireshark/colorfilters

# Column customization (Edit -> Preferences -> Columns)
# Add: Delta Time (tcp.time_delta), Stream Index (tcp.stream)
```

## Ring Buffer Capture

```bash
# Capture with size-limited ring buffer
# 10 files of 50MB each = 500MB max disk usage
tshark -i eth0 -b filesize:51200 -b files:10 -w /var/captures/ring.pcapng

# Duration-based ring buffer (new file every hour, keep 24)
tshark -i eth0 -b duration:3600 -b files:24 -w /var/captures/hourly.pcapng

# Combined: new file at 100MB or 1 hour, keep 48 files
tshark -i eth0 -b filesize:102400 -b duration:3600 -b files:48 \
  -w /var/captures/combined.pcapng

# dumpcap (lower overhead than tshark for long captures)
dumpcap -i eth0 -b filesize:102400 -b files:10 -w /var/captures/ring.pcapng
```

## Useful One-Liners

```bash
# Top 10 talkers by packet count
tshark -r capture.pcapng -qz endpoints,ip | sort -t'|' -k3 -rn | head -10

# Extract all DNS queries
tshark -r capture.pcapng -Y "dns.qr==0" -T fields -e dns.qry.name | sort -u

# Find all unique TLS SNI values (Server Name Indication)
tshark -r capture.pcapng -Y "tls.handshake.type==1" \
  -T fields -e tls.handshake.extensions.server_name | sort -u

# Count HTTP response codes
tshark -r capture.pcapng -Y "http.response" \
  -T fields -e http.response.code | sort | uniq -c | sort -rn

# Find TCP retransmissions by stream
tshark -r capture.pcapng -Y "tcp.analysis.retransmission" \
  -T fields -e tcp.stream -e ip.dst | sort | uniq -c | sort -rn

# Measure connection setup time (SYN to SYN-ACK)
tshark -r capture.pcapng -Y "tcp.flags.syn==1 && tcp.flags.ack==1" \
  -T fields -e tcp.stream -e frame.time_delta_displayed

# Extract all URLs from HTTP traffic
tshark -r capture.pcapng -Y "http.request" \
  -T fields -e http.host -e http.request.uri | awk '{print "http://"$1$2}'

# Merge multiple capture files
mergecap -w merged.pcapng file1.pcapng file2.pcapng file3.pcapng

# Split capture by TCP stream
editcap -c 1000 large.pcapng split.pcapng   # 1000 packets per file
```

## Tips

- Use capture filters (BPF) to limit what is saved to disk; use display filters to explore after capture.
- Start with broad captures and narrow with display filters -- you cannot analyze what you did not capture.
- Use `dumpcap` instead of `tshark` for long-running captures -- it uses less CPU and memory.
- Set SSLKEYLOGFILE before launching the application to decrypt TLS traffic without private keys.
- Ring buffer captures (`-b filesize -b files`) prevent disk exhaustion on production servers.
- Use `tcp.stream eq N` to isolate a single connection, then Follow TCP Stream for readable output.
- The Expert Info panel (Analyze -> Expert Information) quickly highlights retransmissions, resets, and anomalies.
- Create separate profiles for different tasks (web debugging, VoIP, mail analysis) with custom columns and colors.
- Use `tshark -T fields -e ...` for scriptable output -- pipe to `sort | uniq -c` for quick statistics.
- Remote capture via SSH pipe (`ssh host tcpdump | wireshark -k -i -`) avoids copying large files.
- Save display filters as buttons for frequently used queries.
- Use `frame.time_delta_displayed` as a column to spot delays between filtered packets.

## See Also

- iperf (generate controlled traffic for Wireshark analysis)
- postfix (SMTP traffic capture and analysis)
- email-security (TLS and authentication protocol inspection)

## References

- [Wireshark User's Guide](https://www.wireshark.org/docs/wsug_html_chunked/)
- [Wireshark Display Filter Reference](https://www.wireshark.org/docs/dfref/)
- [Wireshark Wiki](https://wiki.wireshark.org/)
- [tshark Manual Page](https://www.wireshark.org/docs/man-pages/tshark.html)
- [BPF Filter Syntax (pcap-filter)](https://www.tcpdump.org/manpages/pcap-filter.7.html)
- [Wireshark TLS Decryption](https://wiki.wireshark.org/TLS)
