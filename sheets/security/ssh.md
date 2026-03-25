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

## SSHD Server Hardening (/etc/ssh/sshd_config)

### Recommended Settings

```bash
# /etc/ssh/sshd_config
Port 2222                                # non-default port
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile .ssh/authorized_keys
MaxAuthTries 3
ClientAliveInterval 300
ClientAliveCountMax 2
X11Forwarding no
AllowTcpForwarding no
AllowUsers deploy alice                  # whitelist users

# Restrict to specific group
AllowGroups ssh-users
```

### Apply Changes

```bash
sudo sshd -t                            # test config syntax
sudo systemctl restart sshd
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
