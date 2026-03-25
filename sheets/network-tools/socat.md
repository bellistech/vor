# socat (SOcket CAT)

Multipurpose relay tool — bidirectional data transfer between two endpoints (sockets, files, pipes, devices, and more).

## Basic Syntax

```bash
# socat [options] ADDRESS1 ADDRESS2
# Data flows bidirectionally between ADDRESS1 and ADDRESS2
```

## TCP Connections

### TCP client
```bash
socat - TCP:example.com:80             # connect and interact via stdin/stdout
socat - TCP:10.0.0.5:8080             # connect to IP:port
socat - TCP:example.com:443,verify=0  # TCP (not TLS) to port 443
```

### TCP server (listener)
```bash
socat TCP-LISTEN:8080,reuseaddr,fork -        # listen, fork per client, echo to stdout
socat TCP-LISTEN:8080,reuseaddr,fork EXEC:/bin/cat   # echo server
socat TCP-LISTEN:8080,bind=127.0.0.1,reuseaddr,fork -  # bind to localhost only
```

### TCP relay (port forwarder)
```bash
socat TCP-LISTEN:8080,reuseaddr,fork TCP:target-host:80
# Accepts connections on 8080, forwards to target-host:80

socat TCP-LISTEN:3307,reuseaddr,fork TCP:db-server:3306
# MySQL proxy on local port 3307
```

## SSL/TLS

### TLS client
```bash
socat - OPENSSL:example.com:443                # TLS connection
socat - OPENSSL:example.com:443,verify=0       # skip cert verification
socat - OPENSSL:example.com:443,cafile=ca.pem  # custom CA
```

### TLS server
```bash
socat OPENSSL-LISTEN:443,cert=server.pem,key=server-key.pem,reuseaddr,fork -
```

### TLS relay (terminate TLS, forward plaintext)
```bash
socat OPENSSL-LISTEN:443,cert=server.pem,key=server-key.pem,reuseaddr,fork TCP:backend:8080
```

### Add TLS to a plaintext service
```bash
socat TCP-LISTEN:8080,reuseaddr,fork OPENSSL:remote:443,verify=0
# Accepts plaintext on 8080, connects to remote via TLS
```

## Unix Domain Sockets

### Connect to a Unix socket
```bash
socat - UNIX-CONNECT:/var/run/docker.sock       # connect to Docker socket
socat - UNIX-CONNECT:/tmp/app.sock              # connect to app socket
```

### Create a Unix socket listener
```bash
socat UNIX-LISTEN:/tmp/app.sock,fork EXEC:/usr/local/bin/handler
```

### Unix socket to TCP (expose locally)
```bash
socat TCP-LISTEN:2375,reuseaddr,fork UNIX-CONNECT:/var/run/docker.sock
# Access Docker API via TCP on port 2375
```

### TCP to Unix socket
```bash
socat UNIX-LISTEN:/tmp/proxy.sock,fork TCP:remote:8080
```

## UDP

### UDP client
```bash
echo "hello" | socat - UDP:10.0.0.5:5000
```

### UDP listener
```bash
socat UDP-LISTEN:5000,fork -                    # listen for UDP
```

### UDP relay
```bash
socat UDP-LISTEN:5353,reuseaddr,fork UDP:dns-server:53
```

## Serial Ports

### Connect to a serial device
```bash
socat - /dev/ttyUSB0,b9600,cs8,parenb=0,cstopb=0,raw  # 9600 8N1
socat - /dev/ttyUSB0,b115200,raw,echo=0                 # 115200 baud
```

### Serial to TCP (serial port server)
```bash
socat TCP-LISTEN:5000,reuseaddr,fork /dev/ttyUSB0,b9600,raw
```

### TCP to serial
```bash
socat TCP:serial-server:5000 /dev/ttyUSB1,b9600,raw
```

## File Descriptors and Pipes

### Named pipe relay
```bash
socat PIPE:/tmp/mypipe TCP:remote:8080
```

### Stdin/stdout relay
```bash
socat STDIN TCP:host:80                          # same as socat - TCP:host:80
socat TCP-LISTEN:8080,fork STDOUT               # accept and print to stdout
```

### Process execution
```bash
socat TCP-LISTEN:9000,reuseaddr,fork EXEC:/bin/bash        # remote shell (dangerous!)
socat TCP-LISTEN:9000,reuseaddr,fork EXEC:"cat /etc/hostname"  # run command per connection
socat TCP-LISTEN:9000,reuseaddr,fork SYSTEM:"date; uptime"     # shell command
```

## Advanced Patterns

### Bidirectional proxy with logging
```bash
socat -v TCP-LISTEN:8080,reuseaddr,fork TCP:target:80
# -v prints data to stderr in readable format

socat -x TCP-LISTEN:8080,reuseaddr,fork TCP:target:80
# -x prints hex dump to stderr
```

### Timeout on connections
```bash
socat -T 30 TCP-LISTEN:8080,reuseaddr TCP:target:80
# -T 30 = timeout inactive connections after 30 seconds
```

### Bind to specific source address
```bash
socat TCP-LISTEN:8080,reuseaddr,fork TCP:target:80,bind=10.0.0.1
```

### Rate limiting / throttling
```bash
socat TCP-LISTEN:8080,reuseaddr,fork TCP:target:80,sndbuf=4096,rcvbuf=4096
```

### SOCKS proxy client
```bash
socat - SOCKS4A:proxy-host:target.com:80,socksport=1080
```

### Create a virtual TUN/TAP device
```bash
socat TUN:10.0.0.1/24,tun-type=tun TCP:remote:7000
```

## HTTP Tricks

### One-shot HTTP response
```bash
echo -e "HTTP/1.1 200 OK\r\n\r\nHello World" | socat - TCP-LISTEN:8080
```

### Test HTTP request
```bash
echo -e "GET / HTTP/1.1\r\nHost: example.com\r\nConnection: close\r\n\r\n" | socat - TCP:example.com:80
```

## IPv6

### IPv6 connections
```bash
socat - TCP6:example.com:80                     # IPv6 TCP
socat TCP6-LISTEN:8080,reuseaddr,fork TCP6:target:80
```

## Tips

- `fork` is essential for servers — without it, socat exits after the first client disconnects
- `reuseaddr` lets you restart socat immediately without waiting for TIME_WAIT to expire
- `-v` (readable) and `-x` (hex) are invaluable for debugging protocol issues
- `socat` is more powerful than `nc` but with a steeper learning curve; use `nc` for simple tasks
- NEVER expose `EXEC:/bin/bash` on untrusted networks — it creates an unauthenticated shell
- Address types are case-sensitive: `TCP-LISTEN`, `UNIX-CONNECT`, `OPENSSL`
- Use `verify=0` only for testing; in production, set `cafile=` for proper TLS verification
- `socat` can bridge any two address types — TCP to Unix socket, serial to TCP, file to UDP, etc.
- The `SYSTEM:` address runs commands through `/bin/sh`; `EXEC:` runs them directly (no shell)
- For persistent services, run socat under systemd or supervisord rather than in a shell
