# SSH Tunneling (Port Forwarding & Proxying)

Use SSH to create encrypted tunnels — local forwarding, remote forwarding, dynamic SOCKS proxy, and jump hosts.

## Local Port Forwarding (-L)

Forward a local port through SSH to a remote destination.

### Access a remote service locally
```bash
# Forward local:8080 -> remote-db:5432 through ssh-host
ssh -L 8080:remote-db:5432 user@ssh-host
# Now connect to localhost:8080 to reach remote-db:5432

# Access a web UI behind a bastion
ssh -L 9090:internal-grafana:3000 user@bastion

# MySQL through a tunnel
ssh -L 3307:127.0.0.1:3306 user@db-server
# Connect with: mysql -h 127.0.0.1 -P 3307

# Bind to all interfaces (not just localhost)
ssh -L 0.0.0.0:8080:target:80 user@ssh-host
```

### Background tunnel (no shell)
```bash
ssh -L 8080:target:80 -N -f user@ssh-host
# -N = no remote command
# -f = go to background after auth
```

## Remote Port Forwarding (-R)

Expose a local service through the remote SSH server.

### Make a local service reachable from remote
```bash
# Expose local:3000 as remote:8080
ssh -R 8080:localhost:3000 user@remote-server
# Anyone on remote-server can now reach localhost:8080

# Expose to all interfaces on remote (requires GatewayPorts yes in sshd_config)
ssh -R 0.0.0.0:8080:localhost:3000 user@remote-server
```

### Reverse tunnel for NAT traversal
```bash
# From behind NAT, make your SSH reachable on the remote server
ssh -R 2222:localhost:22 user@public-server
# Then from public-server: ssh -p 2222 localhost
```

## Dynamic SOCKS Proxy (-D)

Create a SOCKS5 proxy through SSH — route arbitrary traffic.

### Start SOCKS proxy
```bash
ssh -D 1080 user@ssh-host
# Configure browser/app to use SOCKS5 proxy at 127.0.0.1:1080

# Background SOCKS proxy
ssh -D 1080 -N -f user@ssh-host

# Use with curl
curl --proxy socks5h://127.0.0.1:1080 https://example.com
# socks5h = DNS resolution on the remote side
```

## Jump Host / ProxyJump (-J)

Connect through intermediate hosts.

### Single jump
```bash
ssh -J bastion user@internal-host
```

### Multiple jumps
```bash
ssh -J bastion1,bastion2 user@internal-host
```

### Equivalent ProxyCommand
```bash
ssh -o ProxyCommand="ssh -W %h:%p bastion" user@internal-host
```

## Multiplexing (ControlMaster)

Reuse a single SSH connection for multiple sessions.

### Enable multiplexing
```bash
ssh -M -S /tmp/ssh-mux-%r@%h:%p -N -f user@host   # start master
ssh -S /tmp/ssh-mux-%r@%h:%p user@host              # reuse connection
ssh -S /tmp/ssh-mux-%r@%h:%p -O check user@host     # check status
ssh -S /tmp/ssh-mux-%r@%h:%p -O exit user@host      # close master
```

## SSH Config Patterns

### ~/.ssh/config for tunnels
```bash
# Local forward to database
Host db-tunnel
    HostName bastion.example.com
    User ops
    LocalForward 5432 db.internal:5432
    LocalForward 6379 redis.internal:6379

# Jump host config
Host internal-*
    ProxyJump bastion.example.com
    User deploy

# SOCKS proxy
Host socks-proxy
    HostName remote.example.com
    User ops
    DynamicForward 1080
    RequestTTY no

# Multiplexing for all hosts
Host *
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
    ControlPersist 600
```

### Create socket directory
```bash
mkdir -p ~/.ssh/sockets
chmod 700 ~/.ssh/sockets
```

## Multiple Forwards

### Forward several ports at once
```bash
ssh -L 5432:db:5432 -L 6379:redis:6379 -L 9090:grafana:3000 user@bastion
```

## X11 Forwarding

### Forward graphical applications
```bash
ssh -X user@host            # X11 forwarding (with security restrictions)
ssh -Y user@host            # trusted X11 forwarding (no restrictions)
```

## Tunnel with Specific Key

### Use identity file
```bash
ssh -i ~/.ssh/tunnel_key -L 8080:target:80 -N -f user@bastion
```

## Tips

- `-N` (no command) + `-f` (background) is the standard combo for persistent tunnels
- Use `autossh` for tunnels that must survive network interruptions: `autossh -M 0 -L 8080:target:80 user@host`
- `socks5h://` (with the `h`) resolves DNS on the remote side — essential for accessing internal hostnames
- ControlMaster multiplexing dramatically speeds up repeated SSH/SCP/Git operations to the same host
- `ControlPersist 600` keeps the master alive for 10 minutes after the last session closes
- `-L 0.0.0.0:` binds to all interfaces; the default binds only to localhost
- Remote forwarding to `0.0.0.0` requires `GatewayPorts yes` in the remote `sshd_config`
- ProxyJump (`-J`) is simpler than ProxyCommand and supports chaining
- SSH tunnels are encrypted end-to-end between your machine and the SSH server — traffic beyond the server is not encrypted by SSH
- Kill a backgrounded tunnel: `ssh -S /path/to/socket -O exit user@host` or find the PID with `ps aux | grep ssh`

## See Also

- rsync
- scp
- sftp
- socat
- lateral-movement

## References

- [OpenSSH Official Manual Pages](https://www.openssh.com/manual.html)
- [man ssh — SSH Client](https://man7.org/linux/man-pages/man1/ssh.1.html)
- [man ssh_config — SSH Client Configuration](https://man7.org/linux/man-pages/man5/ssh_config.5.html)
- [man sshd_config — SSH Server Configuration](https://man7.org/linux/man-pages/man5/sshd_config.5.html)
- [RFC 4253 — The Secure Shell (SSH) Transport Layer Protocol](https://www.rfc-editor.org/rfc/rfc4253)
- [RFC 4254 — The Secure Shell (SSH) Connection Protocol (Port Forwarding)](https://www.rfc-editor.org/rfc/rfc4254)
- [OpenSSH — SSH Tunneling/Port Forwarding](https://www.openssh.com/features.html)
- [Arch Wiki — OpenSSH Tunneling](https://wiki.archlinux.org/title/OpenSSH#Tunneling)
