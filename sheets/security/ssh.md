# SSH (Secure Shell)

Remote login, file transfer, tunneling, and key-based authentication.

## Key Generation

### Generate Keys

```bash
# Ed25519 (recommended, fast, small keys)
ssh-keygen -t ed25519 -C "alice@acme.com"

# RSA 4096-bit (wider compatibility)
ssh-keygen -t ed25519 -C "alice@acme.com" -f ~/.ssh/id_ed25519_work

# Generate key with no passphrase (CI/deploy keys)
ssh-keygen -t ed25519 -C "deploy@acme.com" -f ~/.ssh/deploy_key -N ""
```

### Change Passphrase on Existing Key

```bash
ssh-keygen -p -f ~/.ssh/id_ed25519
```

### Show Key Fingerprint

```bash
ssh-keygen -lf ~/.ssh/id_ed25519.pub
ssh-keygen -lf ~/.ssh/id_ed25519.pub -E md5   # MD5 format
```

## SSH Agent

### Start and Add Keys

```bash
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
ssh-add ~/.ssh/id_ed25519_work

# List loaded keys
ssh-add -l

# Remove all keys from agent
ssh-add -D
```

### macOS Keychain Integration

```bash
# Add key and store passphrase in Keychain
ssh-add --apple-use-keychain ~/.ssh/id_ed25519
```

## SSH Config (~/.ssh/config)

### Basic Host Configuration

```bash
# ~/.ssh/config
Host prod
    HostName 10.0.1.50
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_ed25519_work

Host dev
    HostName dev.acme.com
    User alice
    ForwardAgent yes

Host bastion
    HostName bastion.acme.com
    User jump

# Jump through bastion to reach internal hosts
Host internal-*
    ProxyJump bastion
    User admin

# Wildcard defaults
Host *
    AddKeysToAgent yes
    IdentitiesOnly yes
    ServerAliveInterval 60
    ServerAliveCountMax 3
```

### Then Connect Simply

```bash
ssh prod                                 # instead of ssh -p 2222 deploy@10.0.1.50
ssh internal-db1                         # auto-jumps through bastion
```

## Authorized Keys

### Add Public Key to Remote Host

```bash
ssh-copy-id -i ~/.ssh/id_ed25519.pub alice@10.0.1.50

# Manual method
cat ~/.ssh/id_ed25519.pub | ssh alice@10.0.1.50 "mkdir -p ~/.ssh && cat >> ~/.ssh/authorized_keys"
```

### Restrict a Key in authorized_keys

```bash
# ~/.ssh/authorized_keys on the remote
command="/usr/local/bin/backup.sh",no-port-forwarding,no-X11-forwarding,no-pty ssh-ed25519 AAAA... deploy@acme.com
```

## Known Hosts

```bash
# Remove stale host entry
ssh-keygen -R 10.0.1.50

# Scan and add host key manually
ssh-keyscan -t ed25519 10.0.1.50 >> ~/.ssh/known_hosts

# Hash known_hosts for privacy
ssh-keygen -H -f ~/.ssh/known_hosts
```

## Port Forwarding and Tunnels

### Local Port Forward (access remote service locally)

```bash
# Forward local:3306 -> remote db on 10.0.1.50:3306
ssh -L 3306:10.0.1.50:3306 bastion

# Background tunnel
ssh -fNL 3306:10.0.1.50:3306 bastion
```

### Remote Port Forward (expose local service to remote)

```bash
# Make local:8080 available as remote:9090
ssh -R 9090:localhost:8080 prod
```

### Dynamic SOCKS Proxy

```bash
ssh -D 1080 bastion
# Then configure browser to use SOCKS5 proxy at localhost:1080
```

## File Transfer

```bash
# Copy file to remote
scp report.pdf alice@10.0.1.50:/tmp/

# Copy directory recursively
scp -r ./dist/ alice@10.0.1.50:/var/www/html/

# rsync over SSH (preferred for large transfers)
rsync -avz -e ssh ./data/ alice@10.0.1.50:/backup/data/
```

## Reverse Tunnels

### Basic Reverse Tunnel (Expose Local Service to Remote)

```bash
# From behind-NAT machine: make local port 8080 available on remote as port 9090
ssh -R 9090:localhost:8080 vps.example.com

# Background reverse tunnel
ssh -fNR 9090:localhost:8080 vps.example.com

# Anyone on the remote network can reach your local service at vps:9090
# (requires GatewayPorts yes in sshd_config on the remote)
```

### Persistent Reverse Tunnel with autossh

```bash
# Install autossh
sudo apt install autossh        # Debian/Ubuntu
sudo dnf install autossh        # Fedora

# autossh reconnects automatically on disconnect
# -M 0 disables autossh's monitoring port (uses SSH's own keepalive instead)
autossh -M 0 -fN \
  -o "ServerAliveInterval 30" \
  -o "ServerAliveCountMax 3" \
  -R 9090:localhost:8080 vps.example.com
```

### Persistent Reverse Tunnel as systemd Service

```bash
# /etc/systemd/system/ssh-reverse-tunnel.service
[Unit]
Description=Persistent SSH Reverse Tunnel
After=network-online.target
Wants=network-online.target

[Service]
User=tunnel
ExecStart=/usr/bin/autossh -M 0 -N \
  -o "ServerAliveInterval 30" \
  -o "ServerAliveCountMax 3" \
  -o "ExitOnForwardFailure yes" \
  -i /home/tunnel/.ssh/id_ed25519 \
  -R 9090:localhost:8080 tunnel@vps.example.com
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ssh-reverse-tunnel
```

### Reverse Tunnel for SSH Access (Phone Home)

```bash
# From behind-NAT machine: make its own SSH accessible via VPS
ssh -fNR 2222:localhost:22 vps.example.com

# From anywhere, SSH to the NAT'd machine via the VPS
ssh -p 2222 user@vps.example.com

# Or chain it: jump through VPS to reach the NAT'd box
ssh -J vps.example.com -p 2222 user@localhost
```

### Remote sshd_config for Reverse Tunnels

```bash
# On the VPS/remote server — /etc/ssh/sshd_config or drop-in
# Allow reverse tunnel ports to bind on all interfaces (not just loopback)
GatewayPorts yes

# Or let the client decide which interface to bind
GatewayPorts clientspecified
# Then client uses: ssh -R 0.0.0.0:9090:localhost:8080 vps
```

## SSHD Server Configuration (/etc/ssh/sshd_config)

### Config File Locations

```bash
# Main config
/etc/ssh/sshd_config

# Drop-in directory (overrides main config — use this for custom settings)
/etc/ssh/sshd_config.d/*.conf
# Files loaded in lexical order; later entries override earlier

# Example drop-in
cat <<'EOF' | sudo tee /etc/ssh/sshd_config.d/50-hardening.conf
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
MaxAuthTries 3
X11Forwarding no
EOF
```

### Hardened Production Config

```bash
# /etc/ssh/sshd_config.d/50-hardening.conf
# --- Authentication ---
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile .ssh/authorized_keys
KbdInteractiveAuthentication no
ChallengeResponseAuthentication no
MaxAuthTries 3
MaxSessions 5
LoginGraceTime 30

# --- Access Control ---
AllowUsers deploy alice                  # whitelist specific users
# AllowGroups ssh-users                  # or whitelist by group
DenyUsers root                           # explicit deny

# --- Forwarding ---
AllowTcpForwarding yes                   # set 'no' unless tunnels needed
AllowStreamLocalForwarding no
X11Forwarding no
PermitTunnel no

# --- Keepalive ---
ClientAliveInterval 300
ClientAliveCountMax 2

# --- Crypto (restrict to strong algorithms) ---
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com
MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com
HostKeyAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# --- Misc ---
Port 2222                               # non-default port
AddressFamily inet                       # ipv4 only (or inet6, or any)
UseDNS no                               # skip reverse DNS (faster login)
PrintMotd no
Banner /etc/ssh/banner.txt              # optional login banner
LogLevel VERBOSE                         # more auth detail in logs
```

### Match Blocks (Per-User/Group/Address Overrides)

```bash
# Allow SFTP-only access for a group
Match Group sftp-only
    ForceCommand internal-sftp
    ChrootDirectory /data/sftp/%u
    AllowTcpForwarding no
    X11Forwarding no
    PermitTunnel no

# Allow password auth from internal network only
Match Address 10.0.0.0/8,192.168.0.0/16
    PasswordAuthentication yes

# Restrict a specific user to tunnel-only (no shell)
Match User tunnel
    AllowTcpForwarding yes
    PermitOpen localhost:8080
    ForceCommand /bin/false
    X11Forwarding no
```

### Apply and Validate

```bash
# Test config syntax before restarting (catches errors)
sudo sshd -t

# Extended test — shows effective config
sudo sshd -T

# Show effective config for a specific user/host match
sudo sshd -T -C user=deploy,host=10.0.1.50

# Restart (or reload for non-port changes)
sudo systemctl restart sshd
sudo systemctl reload sshd

# ALWAYS keep an existing session open when changing sshd config
# If the new config locks you out, the existing session saves you
```

### Two-Factor Authentication (TOTP)

```bash
# Install Google Authenticator PAM module
sudo apt install libpam-google-authenticator    # Debian/Ubuntu

# Configure for a user
google-authenticator    # generates QR code, scratch codes

# /etc/pam.d/sshd — add at top
auth required pam_google_authenticator.so

# /etc/ssh/sshd_config.d/60-2fa.conf
KbdInteractiveAuthentication yes
AuthenticationMethods publickey,keyboard-interactive
# Requires both key AND TOTP code
```

## Debugging SSH Connections

### Client-Side Debugging

```bash
# Verbose output — shows auth negotiation step by step
ssh -v user@host
ssh -vv user@host            # more detail
ssh -vvv user@host           # maximum detail

# Key things to look for in debug output:
#   "Authentications that can continue:" — what the SERVER accepts
#   "Offering public key:" — what keys the CLIENT is trying
#   "Server accepts key:" — key was accepted
#   "No more authentication methods to try" — nothing worked

# Test with specific auth method
ssh -o PreferredAuthentications=publickey user@host
ssh -o PreferredAuthentications=keyboard-interactive -o PubkeyAuthentication=no user@host
ssh -o PreferredAuthentications=password -o PubkeyAuthentication=no user@host

# Test with specific key
ssh -i ~/.ssh/id_ed25519_work -o IdentitiesOnly=yes user@host

# Test connectivity without full login
ssh -o BatchMode=yes -o ConnectTimeout=5 user@host echo ok
```

### Server-Side Debugging

```bash
# Check auth log (real-time)
sudo tail -f /var/log/auth.log              # Debian/Ubuntu
sudo journalctl -fu sshd                    # systemd

# Run sshd in debug mode on alternate port (doesn't affect running sshd)
sudo /usr/sbin/sshd -d -p 2233
# Then connect: ssh -p 2233 user@host
# Shows exactly why auth fails on the server side

# Check which keys sshd would accept for a user
sudo sshd -T -C user=alice | grep authorizedkeysfile

# Verify authorized_keys is readable by sshd
sudo -u alice cat ~alice/.ssh/authorized_keys
namei -l ~alice/.ssh/authorized_keys    # check every directory in the path
```

### Common Auth Failures and Fixes

```bash
# Problem: "Permission denied (publickey)"
# Server only allows pubkey auth but client has no matching key

# Check what auth methods server offers
ssh -v user@host 2>&1 | grep "Authentications that can continue"

# Fix 1: Temporarily enable password auth to copy key
echo "PasswordAuthentication yes" | sudo tee /etc/ssh/sshd_config.d/temp-password.conf
sudo systemctl reload sshd
ssh-copy-id -i ~/.ssh/id_ed25519.pub user@host
sudo rm /etc/ssh/sshd_config.d/temp-password.conf
sudo systemctl reload sshd

# Fix 2: Manually add key if you have console access
cat id_ed25519.pub >> ~user/.ssh/authorized_keys
chmod 600 ~user/.ssh/authorized_keys
chown user:user ~user/.ssh/authorized_keys

# Problem: Key works interactively but not in scripts/cron
# Usually: agent not available or wrong key
ssh -o BatchMode=yes -i /path/to/key -o IdentitiesOnly=yes user@host

# Problem: "Too many authentication failures"
# Client is trying too many keys before the right one
# Fix: use IdentitiesOnly and IdentityFile in ~/.ssh/config
# Or: ssh -o IdentitiesOnly=yes -i ~/.ssh/correct_key user@host

# Problem: Permissions too open
# sshd refuses keys if permissions are wrong
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys ~/.ssh/id_ed25519 ~/.ssh/config
chmod 644 ~/.ssh/id_ed25519.pub
# Home directory must NOT be group/world-writable
chmod go-w ~
# On SELinux: restorecon -Rv ~/.ssh
```

### Connection Multiplexing

```bash
# ~/.ssh/config — reuse connections (faster subsequent connects)
Host *
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
    ControlPersist 600               # keep socket open 10 minutes after last session

# Create the socket directory
mkdir -p ~/.ssh/sockets

# Check active multiplexed connections
ssh -O check prod

# Terminate a multiplexed connection
ssh -O exit prod
```

### SSH Escape Sequences (In-Session)

```bash
# Press Enter, then ~ followed by:
~.      # disconnect (kill hung session)
~^Z     # suspend ssh (bg/fg to return)
~#      # list forwarded connections
~&      # background ssh (waiting for connections to close)
~?      # show all escape sequences
~C      # open command line (add forwards on the fly)

# Add a forward to an existing session (press Enter, then ~C)
~C
ssh> -L 3306:localhost:3306    # add local forward
ssh> -R 9090:localhost:8080    # add reverse forward
ssh> -KL 3306                  # cancel local forward
```

## File Permissions

```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519             # private key
chmod 644 ~/.ssh/id_ed25519.pub         # public key
chmod 600 ~/.ssh/authorized_keys
chmod 600 ~/.ssh/config
```

## Tips

- Ed25519 is preferred over RSA: faster, smaller keys, and better security properties
- Use `IdentitiesOnly yes` in config to prevent the agent from trying every loaded key
- `ServerAliveInterval 60` prevents idle disconnects without relying on client-side keepalive hacks
- `ProxyJump` (or `-J`) replaced `ProxyCommand` for bastion hosts in OpenSSH 7.3+
- Never disable `StrictHostKeyChecking` in production; it protects against MITM attacks
- Use `ssh -v` (or `-vvv`) to debug connection issues; output shows auth method negotiation
- `ssh-copy-id` fails if password auth is already disabled; use the manual method instead
- On SELinux systems, run `restorecon -Rv ~/.ssh` after modifying authorized_keys

## References

- [OpenSSH Manual Pages](https://www.openssh.com/manual.html)
- [ssh(1) Man Page](https://man7.org/linux/man-pages/man1/ssh.1.html)
- [sshd(8) Man Page](https://man7.org/linux/man-pages/man8/sshd.8.html)
- [ssh_config(5) Man Page](https://man7.org/linux/man-pages/man5/ssh_config.5.html)
- [sshd_config(5) Man Page](https://man7.org/linux/man-pages/man5/sshd_config.5.html)
- [ssh-keygen(1) Man Page](https://man7.org/linux/man-pages/man1/ssh-keygen.1.html)
- [ssh-agent(1) Man Page](https://man7.org/linux/man-pages/man1/ssh-agent.1.html)
- [RFC 4251 — SSH Protocol Architecture](https://www.rfc-editor.org/rfc/rfc4251)
- [RFC 4253 — SSH Transport Layer Protocol](https://www.rfc-editor.org/rfc/rfc4253)
- [Arch Wiki — OpenSSH](https://wiki.archlinux.org/title/OpenSSH)
- [Red Hat RHEL 9 — Configuring Secure Communication with SSH](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/securing_networks/assembly_using-secure-communications-between-two-systems-with-openssh_securing-networks)
