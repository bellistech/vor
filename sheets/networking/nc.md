# nc (Netcat)

The "Swiss army knife" of networking — read and write data across TCP/UDP connections.

## Connecting

### TCP client
```bash
nc example.com 80                     # connect to port 80
nc -v example.com 443                 # verbose (shows connection status)
nc -w 5 example.com 80               # 5 second timeout
nc -z example.com 80                  # zero I/O mode (test if port is open)
```

### UDP client
```bash
nc -u example.com 53                  # UDP connection
nc -u -z example.com 123             # test UDP port (unreliable)
```

## Listening

### TCP server
```bash
nc -l 8080                            # listen on port 8080
nc -l -p 8080                         # some versions need -p
nc -l -k 8080                         # keep listening after client disconnects
nc -l 8080 -v                         # verbose — show connections
```

### UDP listener
```bash
nc -l -u 5000                         # listen for UDP on port 5000
```

## Port Scanning

### Scan ports
```bash
nc -z -v example.com 20-25           # scan ports 20-25
nc -z -v example.com 80 443 8080     # scan specific ports
nc -z -w 1 example.com 1-1024        # scan with 1s timeout per port
nc -z -v -n 10.0.0.1 1-65535 2>&1 | grep succeeded  # full scan, show open only
```

## File Transfer

### Send file (receiver listens)
```bash
# On receiver:
nc -l 9000 > received-file.tar.gz

# On sender:
nc receiver-host 9000 < file.tar.gz
```

### Send file (sender listens)
```bash
# On sender:
nc -l 9000 < file.tar.gz

# On receiver:
nc sender-host 9000 > received-file.tar.gz
```

### Transfer directory
```bash
# On receiver:
nc -l 9000 | tar xzf -

# On sender:
tar czf - /path/to/dir | nc receiver-host 9000
```

## Chat / Simple Communication

### Two-way chat
```bash
# Host A:
nc -l 9000

# Host B:
nc host-a 9000
# Type messages on either side — they appear on the other
```

## HTTP Testing

### Manual HTTP request
```bash
echo -e "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n" | nc example.com 80
```

### Simple HTTP server (one response)
```bash
echo -e "HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\n\r\nHello" | nc -l 8080
```

## Proxy and Relay

### Simple TCP relay (with named pipes)
```bash
mkfifo /tmp/ncpipe
nc -l 8080 < /tmp/ncpipe | nc target-host 80 > /tmp/ncpipe
rm /tmp/ncpipe
```

### Port forwarding with two netcats
```bash
mkfifo /tmp/relay
nc -l 3000 < /tmp/relay | nc internal-host 3306 > /tmp/relay &
```

## Banners and Service Detection

### Grab service banners
```bash
nc -v -w 3 example.com 22             # SSH banner
nc -v -w 3 example.com 25             # SMTP banner
nc -v -w 3 example.com 21             # FTP banner
echo "QUIT" | nc -w 3 example.com 25  # SMTP banner then quit
```

## Variants

### ncat (Nmap version — more features)
```bash
ncat --ssl example.com 443            # SSL/TLS connection
ncat -l 8080 --ssl                    # SSL listener
ncat -l 8080 --allow 10.0.0.0/24     # IP allowlist
ncat -l 8080 -e /bin/bash             # bind shell (dangerous!)
ncat --sh-exec "echo hello" -l 8080   # execute command per connection
```

### socat (more powerful alternative)
```bash
# See the socat cheatsheet for advanced relays and protocol handling
```

## Tips

- There are multiple netcat implementations (`nc`, `ncat`, `netcat`, OpenBSD vs GNU) with different flags
- OpenBSD `nc` uses `-l` without `-p`; traditional `nc` needs `-l -p <port>`
- `-z` (zero I/O) for port scanning is more portable and simpler than `nmap` for quick checks
- `-w <seconds>` sets both connect and idle timeout — always use for scripting
- File transfers via `nc` are unencrypted and unauthenticated — use on trusted networks only
- `ncat` (from nmap) supports SSL, access control, and command execution — prefer it when available
- UDP port scanning with `nc -zu` is unreliable — an "open" result just means no ICMP unreachable was received
- Named pipe relays are fragile; use `socat` for production relays
- Never expose `-e /bin/bash` on an untrusted network — it creates a backdoor

## See Also

- nmap, curl, tcpdump, ss, tcp

## References

- [man ncat (Nmap version)](https://man7.org/linux/man-pages/man1/ncat.1.html)
- [Ncat Users' Guide](https://nmap.org/ncat/guide/)
- [Ncat Reference Guide](https://nmap.org/ncat/)
- [OpenBSD netcat Man Page](https://man.openbsd.org/nc.1)
- [GNU Netcat Project](https://netcat.sourceforge.net/)
- [socat — Multipurpose Relay (nc alternative)](http://www.dest-unreach.org/socat/doc/socat.html)
