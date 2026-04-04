# tshark (Terminal Shark)

Command-line version of Wireshark — capture and analyze network traffic with full protocol dissection.

## Capture

### Basic capture
```bash
tshark -i eth0                         # capture on interface
tshark -i any                          # capture on all interfaces
tshark -i eth0 -c 100                  # stop after 100 packets
tshark -D                              # list available interfaces
tshark -i eth0 -a duration:60          # capture for 60 seconds
tshark -i eth0 -a filesize:10000       # stop at 10MB
```

### Write to file
```bash
tshark -i eth0 -w capture.pcap
tshark -i eth0 -w capture.pcap -c 5000
tshark -i eth0 -b filesize:100000 -w capture.pcap  # ring buffer, 100MB per file
tshark -i eth0 -b files:10 -b filesize:100000 -w capture.pcap  # keep 10 files max
```

### Capture filters (BPF — applied during capture)
```bash
tshark -i eth0 -f 'tcp port 80'
tshark -i eth0 -f 'host 10.0.0.5'
tshark -i eth0 -f 'not port 22'
tshark -i eth0 -f 'udp port 53'
tshark -i eth0 -f 'tcp port 443 and host 10.0.0.5'
```

## Display Filters (applied during output)

### Filter captured/read packets
```bash
tshark -i eth0 -Y 'http'                          # HTTP traffic
tshark -i eth0 -Y 'dns'                           # DNS traffic
tshark -i eth0 -Y 'tcp.port == 443'               # traffic on port 443
tshark -i eth0 -Y 'ip.addr == 10.0.0.5'           # specific IP
tshark -i eth0 -Y 'ip.src == 10.0.0.5'            # source IP
tshark -i eth0 -Y 'http.request.method == GET'    # HTTP GET requests
tshark -i eth0 -Y 'tcp.flags.syn == 1 && tcp.flags.ack == 0'  # SYN only
tshark -i eth0 -Y 'dns.qry.name contains example' # DNS queries for example
tshark -i eth0 -Y 'tcp.analysis.retransmission'   # TCP retransmissions
tshark -i eth0 -Y 'tls.handshake.type == 1'       # TLS Client Hello
```

### Read from file with filter
```bash
tshark -r capture.pcap -Y 'http.response.code == 500'
tshark -r capture.pcap -Y 'frame.time >= "2024-01-15 10:00:00"'
```

## Output Control

### Select fields
```bash
tshark -i eth0 -T fields -e ip.src -e ip.dst -e tcp.port
tshark -i eth0 -T fields -e frame.time -e ip.src -e dns.qry.name -Y 'dns'
tshark -r capture.pcap -T fields -e http.host -e http.request.uri -Y 'http.request'
tshark -i eth0 -T fields -E header=y -E separator=, -e ip.src -e ip.dst  # CSV with header
```

### Output formats
```bash
tshark -r capture.pcap -T json                     # JSON output
tshark -r capture.pcap -T jsonraw                  # raw JSON
tshark -r capture.pcap -T ek                       # Elasticsearch bulk format
tshark -r capture.pcap -T pdml                     # XML (protocol dissection)
tshark -r capture.pcap -T psml                     # XML (packet summary)
tshark -r capture.pcap -T tabs                     # tab-separated
```

### Verbose decode
```bash
tshark -r capture.pcap -V                          # full protocol tree
tshark -r capture.pcap -V -O http                  # verbose only for HTTP layer
tshark -r capture.pcap -x                          # hex dump
```

## Statistics

### Conversation statistics
```bash
tshark -r capture.pcap -z conv,tcp                 # TCP conversations
tshark -r capture.pcap -z conv,ip                  # IP conversations
tshark -r capture.pcap -z endpoints,ip             # IP endpoints
```

### Protocol hierarchy
```bash
tshark -r capture.pcap -z io,phs                   # protocol hierarchy stats
```

### HTTP statistics
```bash
tshark -r capture.pcap -z http,tree                # HTTP request/response tree
tshark -r capture.pcap -z http_req,tree            # HTTP request stats
```

### I/O statistics
```bash
tshark -r capture.pcap -z io,stat,1                # packets per second
tshark -r capture.pcap -z io,stat,10,'tcp','udp'   # 10s intervals, TCP vs UDP
```

### DNS statistics
```bash
tshark -r capture.pcap -z dns,tree                 # DNS query/response stats
```

### Expert info
```bash
tshark -r capture.pcap -z expert                   # warnings, errors, notes
```

## Protocol Decode

### Force protocol decode
```bash
tshark -r capture.pcap -d tcp.port==8080,http      # decode port 8080 as HTTP
tshark -r capture.pcap -d udp.port==5353,dns       # decode port 5353 as DNS
```

### TLS decryption (with key log)
```bash
tshark -r capture.pcap -o tls.keylog_file:sslkeys.log -Y http
# Set SSLKEYLOGFILE env var in browser/app to generate the key log
```

## Common Tasks

### Extract HTTP URLs
```bash
tshark -r capture.pcap -T fields -e http.host -e http.request.uri -Y 'http.request' | sort -u
```

### List DNS queries
```bash
tshark -r capture.pcap -T fields -e dns.qry.name -Y 'dns.flags.response == 0' | sort | uniq -c | sort -rn
```

### Find slow TCP connections
```bash
tshark -r capture.pcap -z conv,tcp | sort -t'<' -k5 -rn | head -20
```

### Extract files from HTTP
```bash
tshark -r capture.pcap --export-objects http,/tmp/extracted/
```

### Count packets by protocol
```bash
tshark -r capture.pcap -z io,phs -q
```

## Tips

- Capture filters (`-f`) use BPF syntax and are applied in the kernel — more efficient for high-rate traffic
- Display filters (`-Y`) use Wireshark syntax and are applied after capture — more expressive but slower
- `-T fields -e` is the best way to extract specific data for scripting
- `-z` statistics run in a single pass and are very efficient for large captures
- Use `-q` (quiet) with `-z` to suppress packet output and show only statistics
- `tshark` can read any pcap/pcapng file — captures from tcpdump work fine
- `-d` decode-as is essential for non-standard ports (apps on high ports)
- For TLS decryption, set `SSLKEYLOGFILE=/path/to/keys.log` before starting the client application
- `-b files:N -b filesize:M` creates a ring buffer — essential for continuous monitoring without filling disk
- `tshark` requires the same permissions as tcpdump (root or cap_net_raw)

## See Also

- tcpdump, nmap, ss, curl, nc

## References

- [Wireshark/tshark Official Documentation](https://www.wireshark.org/docs/)
- [tshark Man Page](https://www.wireshark.org/docs/man-pages/tshark.html)
- [Wireshark Display Filter Reference](https://www.wireshark.org/docs/dfref/)
- [Wireshark User's Guide](https://www.wireshark.org/docs/wsug_html_chunked/)
- [Wireshark Wiki — CaptureFilters](https://wiki.wireshark.org/CaptureFilters)
- [Wireshark Wiki — DisplayFilters](https://wiki.wireshark.org/DisplayFilters)
- [editcap Man Page](https://www.wireshark.org/docs/man-pages/editcap.html)
- [mergecap Man Page](https://www.wireshark.org/docs/man-pages/mergecap.html)
- [pcap-filter — BPF Filter Syntax](https://www.tcpdump.org/manpages/pcap-filter.7.html)
