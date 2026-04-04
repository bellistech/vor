# tcpdump (Packet Capture)

Command-line packet analyzer — captures and displays network traffic using libpcap.

## Basic Capture

### Capture on an interface
```bash
tcpdump -i eth0                    # specific interface
tcpdump -i any                     # all interfaces
tcpdump -i eth0 -c 100             # stop after 100 packets
tcpdump -i eth0 -n                 # don't resolve hostnames
tcpdump -i eth0 -nn                # don't resolve hostnames or ports
tcpdump -D                         # list available interfaces
```

### Verbosity
```bash
tcpdump -i eth0 -v                 # verbose (TTL, ID, total length)
tcpdump -i eth0 -vv                # more verbose (full protocol decode)
tcpdump -i eth0 -vvv               # maximum verbosity
tcpdump -i eth0 -q                 # quiet — minimal output
```

### Show packet contents
```bash
tcpdump -i eth0 -X                 # hex + ASCII
tcpdump -i eth0 -XX                # hex + ASCII including link header
tcpdump -i eth0 -A                 # ASCII only (good for HTTP)
tcpdump -i eth0 -s 0               # capture full packets (no truncation)
```

## Writing and Reading PCAP

### Save to file
```bash
tcpdump -i eth0 -w capture.pcap
tcpdump -i eth0 -w capture.pcap -c 10000       # limit packet count
tcpdump -i eth0 -w capture.pcap -G 3600 -W 24  # rotate hourly, keep 24 files
tcpdump -i eth0 -w capture.pcap -C 100         # rotate at 100MB
```

### Read from file
```bash
tcpdump -r capture.pcap
tcpdump -r capture.pcap -nn                     # no resolution
tcpdump -r capture.pcap 'tcp port 80'           # apply filter to saved capture
tcpdump -r capture.pcap -c 50                   # first 50 packets
```

## Filters (BPF)

### By host
```bash
tcpdump -i eth0 host 10.0.0.5
tcpdump -i eth0 src host 10.0.0.5
tcpdump -i eth0 dst host 10.0.0.5
tcpdump -i eth0 net 192.168.1.0/24              # subnet
```

### By port
```bash
tcpdump -i eth0 port 80
tcpdump -i eth0 src port 443
tcpdump -i eth0 dst port 53
tcpdump -i eth0 portrange 8000-8100
```

### By protocol
```bash
tcpdump -i eth0 tcp
tcpdump -i eth0 udp
tcpdump -i eth0 icmp
tcpdump -i eth0 arp
tcpdump -i eth0 ip6
```

### Combining filters
```bash
tcpdump -i eth0 'tcp and port 80'
tcpdump -i eth0 'host 10.0.0.5 and tcp port 443'
tcpdump -i eth0 'src 10.0.0.5 or src 10.0.0.6'
tcpdump -i eth0 'not port 22'                         # exclude SSH
tcpdump -i eth0 'not (port 22 or port 53)'            # exclude SSH and DNS
```

### TCP flags
```bash
tcpdump -i eth0 'tcp[tcpflags] & tcp-syn != 0'        # SYN packets
tcpdump -i eth0 'tcp[tcpflags] & tcp-rst != 0'        # RST packets
tcpdump -i eth0 'tcp[tcpflags] & (tcp-syn|tcp-fin) != 0'  # SYN or FIN
tcpdump -i eth0 'tcp[tcpflags] == tcp-syn'             # SYN only (no ACK)
```

### Packet size
```bash
tcpdump -i eth0 'greater 1000'                 # packets > 1000 bytes
tcpdump -i eth0 'less 100'                     # packets < 100 bytes
```

## Common Expressions

### HTTP traffic
```bash
tcpdump -i eth0 -A -s 0 'tcp port 80 and (((ip[2:2] - ((ip[0]&0xf)<<2)) - ((tcp[12]&0xf0)>>2)) != 0)'
```

### DNS queries
```bash
tcpdump -i eth0 -nn 'udp port 53'
```

### ICMP (ping) traffic
```bash
tcpdump -i eth0 'icmp[icmptype] == icmp-echo or icmp[icmptype] == icmp-echoreply'
```

### Capture only SYN (new connections)
```bash
tcpdump -i eth0 -nn 'tcp[tcpflags] == tcp-syn'
```

### Traffic between two hosts
```bash
tcpdump -i eth0 'host 10.0.0.1 and host 10.0.0.2'
```

### VLAN-tagged traffic
```bash
tcpdump -i eth0 'vlan and tcp port 80'
```

## Timestamps

### Timestamp formats
```bash
tcpdump -i eth0 -t                 # no timestamp
tcpdump -i eth0 -tt                # Unix epoch seconds
tcpdump -i eth0 -ttt               # delta from previous packet
tcpdump -i eth0 -tttt              # date + time
tcpdump -i eth0 -ttttt             # delta from first packet
```

## Tips

- Always use `-n` (or `-nn`) in production to avoid DNS lookups that slow output and leak queries
- `-s 0` captures full packets; the default snaplen (262144 on modern systems) is usually fine
- Quote BPF filters to prevent shell interpretation: `'tcp port 80 and not host 10.0.0.1'`
- Writing to file (`-w`) is much faster than printing to terminal — use for high-rate captures
- `-G` (time rotation) + `-W` (file count) = bounded disk usage for long captures
- Use `-Z <user>` to drop privileges after opening the capture device
- `tcpdump -r file.pcap | wc -l` quickly counts packets in a capture
- PCAP files from tcpdump open in Wireshark/tshark for detailed analysis
- On busy interfaces, capture to file and analyze later to avoid dropping packets

## See Also

- tshark, nmap, ss, tcp, udp

## References

- [tcpdump Official Man Page](https://www.tcpdump.org/manpages/tcpdump.1.html)
- [tcpdump/libpcap Project](https://www.tcpdump.org/)
- [pcap-filter — BPF Filter Syntax](https://www.tcpdump.org/manpages/pcap-filter.7.html)
- [libpcap Documentation](https://www.tcpdump.org/manpages/pcap.3pcap.html)
- [man tcpdump](https://man7.org/linux/man-pages/man1/tcpdump.1.html)
- [Wireshark — Capture Filters (BPF Syntax)](https://wiki.wireshark.org/CaptureFilters)
- [Cloudflare Blog — BPF and tcpdump](https://blog.cloudflare.com/bpf-the-forgotten-bytecode/)
- [Daniel Miessler — tcpdump Tutorial and Primer](https://danielmiessler.com/p/tcpdump/)
