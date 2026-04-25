# SSH (Secure Shell)

OpenSSH client/server reference: keys, agents, ssh_config/sshd_config keywords, port forwarding, jump hosts, certificates, error decoding.

## Setup

OpenSSH ships everywhere. The OpenSSH "portable" tree (openssh.com) is what every Linux/macOS distribution packages. libssh and libssh2 are independent libraries used by clients like git/PuTTY/Paramiko — they implement the SSH protocol but are NOT OpenSSH and have their own config quirks.

```bash
# Debian / Ubuntu
sudo apt install openssh-client openssh-server

# Fedora / RHEL / Rocky / Alma
sudo dnf install openssh-clients openssh-server

# Arch
sudo pacman -S openssh

# Alpine
sudo apk add openssh-client openssh-server

# macOS — client preinstalled. For sshd: System Settings > General > Sharing > Remote Login.
# Or via CLI:
sudo systemsetup -setremotelogin on

# Windows 10/11 — Settings > Apps > Optional features > "OpenSSH Client" / "OpenSSH Server"
# PowerShell:
Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0
Start-Service sshd
Set-Service -Name sshd -StartupType Automatic
```

```bash
# Version check (target this sheet at OpenSSH 8.8+)
ssh -V
# OpenSSH_9.6p1, OpenSSL 3.0.13 30 Jan 2024

# sshd version
sshd -V 2>&1 | head -1
# unknown option -- V; sshd: usage:
# (older sshd needs:)
sudo /usr/sbin/sshd -t -d 2>&1 | head -1

# Print compile-time defaults
ssh -G localhost | head -20
sshd -T 2>&1 | head -20

# Service start/stop (systemd)
sudo systemctl enable --now ssh        # Debian/Ubuntu (unit is named "ssh")
sudo systemctl enable --now sshd       # Fedora/RHEL (unit is named "sshd")
sudo systemctl status ssh
```

OpenSSH 8.8 (Sep 2021) disabled `ssh-rsa` (SHA-1 RSA signatures) by default. OpenSSH 9.0 added the `sntrup761x25519-sha512@openssh.com` post-quantum hybrid KEX as default. OpenSSH 9.5+ uses `mlkem768x25519-sha256` (ML-KEM/Kyber) hybrid KEX. These dates matter when an old client/server pair fails to negotiate — see "Common Error Messages" below.

## Key Generation

`ssh-keygen` is client-side. Output is two files: the private key (no extension, mode 0600) and the public key (`.pub`, mode 0644).

```bash
# Modern default — Ed25519 (recommended for ALL new keys)
ssh-keygen -t ed25519 -a 100 -C "alice@laptop-2026" -f ~/.ssh/id_ed25519
#   -t ed25519     algorithm
#   -a 100         KDF rounds for the bcrypt-pbkdf passphrase derivation (100 = ~1s on modern CPU)
#   -C "..."       comment baked into the .pub file (used to identify keys later)
#   -f path        output file path (also writes path.pub)

# RSA — only for compatibility with very old servers
ssh-keygen -t rsa -b 4096 -a 100 -C "alice@legacy" -f ~/.ssh/id_rsa
# 3072 is the minimum acceptable; 4096 is conventional. Anything < 3072 should be rotated.

# ECDSA — discouraged; NIST curves with concerns about parameter trust. Avoid for new keys.
ssh-keygen -t ecdsa -b 521 -f ~/.ssh/id_ecdsa

# DSA — REMOVED in OpenSSH 9.8. Banned. Do not use.

# Passphrase-less key (CI / automation only — protect via filesystem ACL)
ssh-keygen -t ed25519 -N "" -C "ci-deploy" -f ~/.ssh/deploy_key

# Change passphrase on an existing key (re-encrypt private key)
ssh-keygen -p -f ~/.ssh/id_ed25519
# Old passphrase: ...
# New passphrase: ...

# Remove passphrase entirely
ssh-keygen -p -N "" -f ~/.ssh/id_ed25519

# Derive public key from a private key (when .pub is lost)
ssh-keygen -y -f ~/.ssh/id_ed25519 > ~/.ssh/id_ed25519.pub

# Show fingerprint of a key
ssh-keygen -lf ~/.ssh/id_ed25519.pub
# 256 SHA256:abc123... alice@laptop-2026 (ED25519)

ssh-keygen -lf ~/.ssh/id_ed25519.pub -E md5
# 256 MD5:01:23:45:67:... (legacy fingerprint format — github used this until ~2017)

# Find a host's entry inside known_hosts
ssh-keygen -F github.com
ssh-keygen -F github.com -f ~/.ssh/known_hosts

# Remove a host's entry
ssh-keygen -R github.com

# Convert OpenSSH private key to legacy PEM (PKCS#1) — needed by some tools
ssh-keygen -p -m PEM -f ~/.ssh/id_rsa
# -m formats: RFC4716 (default for pub), PKCS8, PEM

# Convert OpenSSH public key to RFC 4716 (SECSH) — needed by Tectia / some commercial servers
ssh-keygen -e -f ~/.ssh/id_ed25519.pub
ssh-keygen -e -m RFC4716 -f ~/.ssh/id_ed25519.pub > id_ed25519.rfc4716

# Convert RFC 4716 back to OpenSSH format
ssh-keygen -i -f remote_key.pub > openssh_key.pub
```

## Key Types and Algorithms

| Type | Status | Use For |
| --- | --- | --- |
| `ed25519` | Modern default | All new keys |
| `ed25519-sk` | Modern default + FIDO2 | Hardware-backed (Yubikey) |
| `rsa` (3072+) | Legacy-OK if `rsa-sha2-256/512` signatures used | Old servers (pre-OpenSSH 7.2) |
| `rsa` with `ssh-rsa` (SHA-1) | Disabled by default since OpenSSH 8.8 | Migration target only |
| `ecdsa` | Discouraged | Avoid; use ed25519 |
| `ecdsa-sk` | OK if you specifically need NIST curve hardware tokens | Niche |
| `dsa` | REMOVED in OpenSSH 9.8 | Never |

The OpenSSH 8.8 `ssh-rsa` story: prior to 8.8, OpenSSH advertised the `ssh-rsa` (SHA-1) signature algorithm as part of its default. Since SHA-1 is broken (SHATTERED, 2017), 8.8 disabled it by default. RSA keys themselves still work — but the *signature algorithm* must be `rsa-sha2-256` or `rsa-sha2-512`. Old servers (OpenSSH < 7.2, pre-2016) only know `ssh-rsa` and fail. Workaround for transition:

```bash
# Per-host opt-in to SHA-1 RSA signatures (transitional!)
Host legacy.example.com
    HostKeyAlgorithms +ssh-rsa
    PubkeyAcceptedAlgorithms +ssh-rsa
```

The `+` prefix appends to the default list. Use sparingly and update the server to ed25519 ASAP.

### FIDO2 / Hardware-Backed Keys

OpenSSH 8.2+ supports `-sk` ("security key") keys that require a physical FIDO2 token (Yubikey, SoloKey, Titan).

```bash
# Discoverable / non-resident — private key blob lives on disk, but
# signing requires the hardware token to be present and touched.
ssh-keygen -t ed25519-sk -O resident -O verify-required -C "alice@yubikey-1"

#   -O resident          store key handle on the token itself (recoverable via ssh-keygen -K)
#   -O verify-required   require PIN entry on every use (not just touch)
#   -O application=ssh:GitHub   namespace the key (multiple ssh-sk keys on one token)
#   -O no-touch-required relax touch requirement (don't!)

# Recover resident keys onto a new machine
ssh-keygen -K
# Writes id_ed25519_sk_rk_<application> and .pub into the cwd.

# ECDSA-sk variant (for tokens without ed25519 support)
ssh-keygen -t ecdsa-sk -C "alice@yubikey-1"
```

### Key File Formats

```bash
# OpenSSH new format (default since OpenSSH 7.8) — encrypted with bcrypt-pbkdf
file ~/.ssh/id_ed25519
# id_ed25519: OpenSSH private key

head -1 ~/.ssh/id_ed25519
# -----BEGIN OPENSSH PRIVATE KEY-----

# Legacy PEM (PKCS#1) format — used by older AWS/GCP/Terraform tooling
head -1 legacy.pem
# -----BEGIN RSA PRIVATE KEY-----

# Force generate as PEM (for compatibility)
ssh-keygen -t rsa -b 4096 -m PEM -f ~/.ssh/id_rsa_pem

# Convert existing OpenSSH key to PEM in place
ssh-keygen -p -m PEM -f ~/.ssh/id_rsa
```

## Public Key Authentication Setup

The server stores acceptable public keys in `~/.ssh/authorized_keys` for the target user. The client sends a signature; the server verifies against the stored public key.

```bash
# Easiest: ssh-copy-id (handles permissions, dedupe, and the file format)
ssh-copy-id -i ~/.ssh/id_ed25519.pub alice@server.example.com

# With a non-default port
ssh-copy-id -p 2222 -i ~/.ssh/id_ed25519.pub alice@server.example.com

# Manual append (when ssh-copy-id is unavailable, e.g. minimal busybox)
cat ~/.ssh/id_ed25519.pub | \
  ssh alice@server.example.com 'mkdir -p -m 700 ~/.ssh && \
                                cat >> ~/.ssh/authorized_keys && \
                                chmod 600 ~/.ssh/authorized_keys'

# Permissions sshd checks (StrictModes yes — default):
#   ~/                       NOT group/world-writable
#   ~/.ssh                   0700 (or stricter)
#   ~/.ssh/authorized_keys   0600 (or stricter)
#   Owner = the target user (root-owned files are accepted only if root is the user)

# Quick fix when sshd silently rejects keys:
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
chmod go-w ~                 # remove group/world write on $HOME
chown -R alice:alice ~/.ssh  # fix accidental root-owned files
```

### authorized_keys Options (Per-Key Restrictions)

Each line of `authorized_keys` is `[options ]keytype keydata [comment]`. Options are comma-separated, no whitespace inside.

```bash
# Restricted command (key only runs the named program — overrides any user shell request)
command="/usr/local/bin/backup.sh",no-port-forwarding,no-X11-forwarding,no-pty ssh-ed25519 AAAA... deploy@ci

# IP allowlist for this key
from="10.0.0.0/24,192.168.1.5",no-agent-forwarding ssh-ed25519 AAAA... alice@office

# Force a TTY allocation off
no-pty ssh-ed25519 AAAA... ci@github

# Restrict port forwards to a single endpoint
permitopen="db.internal:5432" ssh-ed25519 AAAA... db-tunnel@bastion

# Set environment variables on login (server must have PermitUserEnvironment yes)
environment="DEPLOY_ROLE=prod" ssh-ed25519 AAAA... deploy@ci

# Common combo: SCP-only deploy key with rsync allowlist
command="rsync --server -vlogDtprze.iLsfxC . /srv/uploads/",restrict ssh-ed25519 AAAA... ci-rsync
# `restrict` (modern) = no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty,no-user-rc + future restrictions
```

## ssh-agent

The agent caches decrypted private keys in memory and signs auth challenges on demand. Without it, you re-enter your passphrase every connection.

```bash
# Manual start (POSIX shells)
eval "$(ssh-agent -s)"
# Agent pid 12345
# echo $SSH_AUTH_SOCK
# /tmp/ssh-XXXXXX/agent.12344

# Add a key (will prompt for passphrase)
ssh-add ~/.ssh/id_ed25519
ssh-add ~/.ssh/id_ed25519_work
ssh-add ~/.ssh/id_rsa_legacy

# List loaded keys (fingerprints)
ssh-add -l
# 256 SHA256:abc... alice@laptop-2026 (ED25519)

# List loaded public keys (paste into authorized_keys)
ssh-add -L

# Remove a single key
ssh-add -d ~/.ssh/id_ed25519

# Remove ALL keys
ssh-add -D

# Add with timeout (auto-evict after 1 hour)
ssh-add -t 3600 ~/.ssh/id_ed25519
ssh-add -t 1h ~/.ssh/id_ed25519     # OpenSSH 8.8+ also accepts duration suffixes

# Add with confirmation prompt on every use
ssh-add -c ~/.ssh/id_ed25519        # SSH_ASKPASS pops up before each sign

# macOS — store passphrase in Keychain
ssh-add --apple-use-keychain ~/.ssh/id_ed25519
# Older macOS spelled this -K; Apple kept that as an alias.

# In ~/.ssh/config (OpenSSH 7.2+) — auto-add on first use
Host *
    AddKeysToAgent yes
    UseKeychain yes              # macOS only — load passphrase from Keychain
    IdentityFile ~/.ssh/id_ed25519
```

```bash
# systemd user unit — agent that survives logout
mkdir -p ~/.config/systemd/user/

cat > ~/.config/systemd/user/ssh-agent.service <<'EOF'
[Unit]
Description=SSH key agent

[Service]
Type=simple
Environment=SSH_AUTH_SOCK=%t/ssh-agent.socket
ExecStart=/usr/bin/ssh-agent -D -a $SSH_AUTH_SOCK

[Install]
WantedBy=default.target
EOF

# In ~/.bashrc or ~/.zshrc:
export SSH_AUTH_SOCK="$XDG_RUNTIME_DIR/ssh-agent.socket"

systemctl --user enable --now ssh-agent.service
```

```bash
# GNOME — gnome-keyring-daemon already provides an SSH agent compatible socket
echo $SSH_AUTH_SOCK
# /run/user/1000/keyring/ssh

# KDE — ksshaskpass for graphical passphrase prompts
sudo apt install ksshaskpass
export SSH_ASKPASS=/usr/bin/ksshaskpass
```

### 1Password / Bitwarden / op-ssh-sign

Modern password managers act as an SSH agent — keys never leave the vault, and signing requires biometric/PIN unlock.

```bash
# 1Password (8.10+) — turn on Settings > Developer > "Use the SSH agent"
# Then in ~/.ssh/config:
Host *
    IdentityAgent "~/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock"

# Bitwarden CLI plugin (bw-ssh) similar pattern:
Host *
    IdentityAgent ~/.bitwarden-ssh-agent.sock
```

## ssh_config — Client Config

Two locations, first match wins per keyword:

| Path | Scope |
| --- | --- |
| `~/.ssh/config` | Per-user |
| `/etc/ssh/ssh_config` | System-wide |
| `/etc/ssh/ssh_config.d/*.conf` | System drop-ins (modern) |

```bash
# Permissions enforced by ssh client:
chmod 600 ~/.ssh/config

# Generic "for every host" stanza must come LAST — first match wins
# So put specific Host stanzas above the catch-all Host *
```

### Host Pattern Matching

```bash
# Single host
Host prod-db1.example.com

# Multiple aliases for the same stanza
Host prod-db1 prod-db1.internal prod-db1.example.com

# Wildcards
Host *.example.com           # any subdomain
Host prod-*                  # prod-anything
Host !staging-* prod-*       # NOT staging, AND prod (negation must come first)

# Catch-all (always last in file)
Host *
    ServerAliveInterval 60
```

### Match Blocks

```bash
# Match runs after Host blocks; allows conditional logic
Match host bastion.example.com
    User jumpbox-user

# Match by user
Match user alice
    IdentityFile ~/.ssh/id_alice

# Match exec — run a shell command; if exit=0, options apply
Match exec "test -e /tmp/.work-vpn-up"
    ProxyJump work-bastion

# Match against the originally-requested name (before CanonicalizeHostname)
Match originalhost prod
    HostName prod-real.example.com

# Match canonical names (after CanonicalizeHostname expanded short→FQDN)
Match canonical host *.prod.example.com
    User deploy

# All — always matches; useful at the bottom
Match all
    LogLevel QUIET
```

## ssh_config Keywords

Reference of the keywords most useful in `ssh_config` (client-side). Server-side equivalents live in `sshd_config` (next section).

```bash
# AddKeysToAgent <yes|no|ask|confirm|TIME>
#   When prompted for a key passphrase, also load it into the agent.
AddKeysToAgent yes

# AddressFamily <any|inet|inet6>
#   Force IPv4 or IPv6. "inet" fixes the "ssh waits 5s on AAAA lookup" issue.
AddressFamily inet

# BatchMode <yes|no>
#   Disable all interactive prompts (passphrase, password, host-key TOFU). Returns nonzero on prompt.
BatchMode yes

# BindAddress <ip|hostname>
#   Source IP for the outbound connection (multi-homed host).
BindAddress 192.168.1.42

# CanonicalDomains <domain1 domain2 ...>
#   When CanonicalizeHostname is on, append these domains to short names.
CanonicalDomains example.com internal.example.com

# CanonicalizeFallbackLocal <yes|no>
#   If canonicalization fails, fall back to /etc/hosts / system resolver.
CanonicalizeFallbackLocal yes

# CanonicalizeHostname <yes|no|always>
#   Expand short names ("prod") to FQDN ("prod.example.com") via DNS before applying Host stanzas.
CanonicalizeHostname yes

# CanonicalizePermittedCNAMEs <pattern_list>
#   Allow CNAME chains during canonicalization. Default: none (CNAMEs ignored).
CanonicalizePermittedCNAMEs *.example.com:*.aws.example.com

# CertificateFile <path>
#   Use this SSH certificate (signed pubkey) for auth. Pair with IdentityFile.
CertificateFile ~/.ssh/id_ed25519-cert.pub

# ChallengeResponseAuthentication <yes|no>
#   Legacy alias for KbdInteractiveAuthentication. Disabled in OpenSSH 8.7+.
ChallengeResponseAuthentication no

# CheckHostIP <yes|no>
#   Also check the host IP (not just hostname) against known_hosts. Useful for DNS round-robin.
CheckHostIP no

# Ciphers <cipher_list>
#   Order matters: client offers in this order. Use +/- to modify default.
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com

# ClearAllForwardings <yes|no>
#   Cancel all -L/-R/-D forwards from earlier on the command line / config.
ClearAllForwardings yes

# ConnectTimeout <seconds>
#   Give up the TCP connect after N seconds.
ConnectTimeout 10

# ConnectionAttempts <number>
#   Retry TCP connect this many times before failing.
ConnectionAttempts 3

# ControlMaster <yes|no|auto|ask|autoask>
#   See "SSH Multiplexing" section below.
ControlMaster auto

# ControlPath <path>
#   Multiplex socket. %r=user, %h=host, %p=port, %C=hash(...).
ControlPath ~/.ssh/cm-%r@%h:%p

# ControlPersist <yes|no|TIME>
#   Keep multiplex socket alive after the last session for TIME (e.g. 30m, 4h).
ControlPersist 30m

# DynamicForward <[bind:]port>
#   SOCKS proxy on local port (same as -D).
DynamicForward 1080

# EscapeChar <char|none>
#   Escape character for in-session commands (default ~). Set "none" inside scripts.
EscapeChar none

# ExitOnForwardFailure <yes|no>
#   If a forward fails to bind, terminate the connection. Critical for tunnel-only sessions.
ExitOnForwardFailure yes

# ForwardAgent <yes|no>
#   Forward the local agent to the remote. SECURITY: any root on remote can sign with your keys.
ForwardAgent no

# ForwardX11 <yes|no>
#   Forward X11 protocol to the remote.
ForwardX11 no

# ForwardX11Trusted <yes|no>
#   Trust remote X11 clients (no SECURITY extension limits). Equivalent to -Y vs -X.
ForwardX11Trusted no

# GSSAPIAuthentication <yes|no>
#   Kerberos auth. Most non-corporate setups: no (fails fast, saves a roundtrip).
GSSAPIAuthentication no

# GlobalKnownHostsFile <path...>
#   System-wide known_hosts file(s). Default: /etc/ssh/ssh_known_hosts.
GlobalKnownHostsFile /etc/ssh/ssh_known_hosts

# HashKnownHosts <yes|no>
#   Hash hostnames in known_hosts. Privacy if file is leaked. Breaks tab-completion.
HashKnownHosts yes

# Host <pattern_list>
#   Stanza header; everything until next Host/Match applies.
Host prod-*

# HostKeyAlgorithms <list>
#   Acceptable server host-key algorithms. Use +/- to modify default.
HostKeyAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# HostKeyAlias <name>
#   Look up host key under this name in known_hosts (when host has multiple IPs / port-forwarded).
HostKeyAlias prod-real

# HostName <hostname|ip>
#   Real hostname; the Host stanza name becomes an alias.
HostName 10.0.1.5

# HostbasedAuthentication <yes|no>
#   Legacy host-based auth. Almost always: no.
HostbasedAuthentication no

# IdentitiesOnly <yes|no>
#   Use ONLY the IdentityFile(s) listed in this stanza, not every key in the agent.
#   CRITICAL when you have many agent keys — prevents "Too many auth failures".
IdentitiesOnly yes

# IdentityAgent <path|SSH_AUTH_SOCK|none>
#   Use a specific agent socket (e.g. 1Password, Bitwarden).
IdentityAgent "~/Library/Group Containers/.../agent.sock"

# IdentityFile <path>
#   Private key path. Multiple allowed; tried in order.
IdentityFile ~/.ssh/id_ed25519
IdentityFile ~/.ssh/id_work

# Include <path>
#   Recursively include another config file. Glob OK. Modular configs!
Include ~/.ssh/config.d/*.conf

# IPQoS <class [class]>
#   DSCP markings for interactive vs bulk. Default: af21 cs1.
IPQoS af21 cs1

# KbdInteractiveAuthentication <yes|no>
#   Keyboard-interactive (PAM, TOTP, etc.).
KbdInteractiveAuthentication yes

# KexAlgorithms <list>
#   Key exchange algorithms.
KexAlgorithms curve25519-sha256@libssh.org,curve25519-sha256

# KnownHostsCommand <command>
#   Run a command to fetch the host key (e.g. lookup in a CA bundle, vault).
KnownHostsCommand /usr/local/bin/known-hosts-fetcher %H

# LocalCommand <command>
#   Run on the LOCAL host after the connection succeeds. Requires PermitLocalCommand yes.
PermitLocalCommand yes
LocalCommand notify-send "SSH connected: %n"

# LocalForward <[bind:]port host:port>
#   -L equivalent in config.
LocalForward 5432 db.internal:5432

# LogLevel <QUIET|FATAL|ERROR|INFO|VERBOSE|DEBUG|DEBUG1|DEBUG2|DEBUG3>
LogLevel INFO

# MACs <list>
#   Message authentication codes. ETM variants are AEAD-equivalent.
MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com

# NumberOfPasswordPrompts <N>
#   Default 3.
NumberOfPasswordPrompts 1

# PasswordAuthentication <yes|no>
#   Try password auth at all. Set "no" if you want fast-fail when keys don't work.
PasswordAuthentication no

# PermitLocalCommand <yes|no>
#   Allow LocalCommand to run.
PermitLocalCommand no

# PermitRemoteOpen <host:port [host:port ...]>
#   For -R forwards, restrict which (host:port) pairs the remote may open.
PermitRemoteOpen any

# PKCS11Provider <path>
#   Use a PKCS#11 token (smartcard, OpenSC).
PKCS11Provider /usr/lib/opensc-pkcs11.so

# Port <number>
#   Default 22. Per-Host override.
Port 2222

# PreferredAuthentications <list>
#   Auth method order.
PreferredAuthentications publickey,keyboard-interactive,password

# ProxyCommand <command>
#   Pipe ssh's IO through this command. Legacy bastion mechanism.
ProxyCommand ssh bastion -W %h:%p

# ProxyJump <[user@]host[:port][,host2...]>
#   Modern bastion / chained jumps (OpenSSH 7.3+).
ProxyJump bastion.example.com

# ProxyUseFdpass <yes|no>
#   ProxyCommand passes FD instead of streaming. Niche.
ProxyUseFdpass no

# PubkeyAcceptedAlgorithms <list>
#   Acceptable client public-key algorithms (replaces PubkeyAcceptedKeyTypes since 8.5).
PubkeyAcceptedAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# PubkeyAuthentication <yes|no>
PubkeyAuthentication yes

# RekeyLimit <data [time]>
#   Re-negotiate session keys after this much data / time. Default: 1G 1h.
RekeyLimit 1G 1h

# RemoteCommand <command>
#   Run command on remote and exit. Equivalent to: ssh host '<cmd>'
RemoteCommand tmux attach -t main

# RemoteForward <remote_port [bind:]localhost:localport>
#   -R equivalent in config.
RemoteForward 9090 localhost:8080

# RequestTTY <yes|no|force|auto>
#   Whether to allocate a PTY. force = -tt; no = -T.
RequestTTY auto

# RevokedHostKeys <path>
#   Public keys here are NEVER trusted as host keys.
RevokedHostKeys ~/.ssh/revoked_host_keys

# SecurityKeyProvider <path>
#   For non-default FIDO2 middleware (e.g. an alternative libsk-libfido2.so).
SecurityKeyProvider /usr/lib/libsk-libfido2.so

# SendEnv <list>
#   Environment vars to send (LANG, LC_*, GIT_*). Server must allow via AcceptEnv.
SendEnv LANG LC_*

# ServerAliveCountMax <N>
#   How many missed keepalives before declaring the connection dead. Default 3.
ServerAliveCountMax 3

# ServerAliveInterval <seconds>
#   Send SSH-level keepalive every N seconds. 0 disables. Critical behind NAT timeouts.
ServerAliveInterval 60

# SessionType <none|subsystem|default>
#   none = no shell/exec, just forwarding (-N).
SessionType none

# SetEnv <NAME=VALUE ...>
#   Set environment vars on the remote (server must AcceptEnv NAME).
SetEnv MY_VAR=value

# StreamLocalBindMask <octal>
#   umask for unix-domain socket binds. Default 0177.
StreamLocalBindMask 0177

# StreamLocalBindUnlink <yes|no>
#   Remove existing socket file before binding.
StreamLocalBindUnlink yes

# StrictHostKeyChecking <yes|no|accept-new|off|ask>
#   yes = error on unknown / changed; no = auto-add (DANGEROUS); accept-new = add unknown,
#   error on changed (RECOMMENDED for new hosts). Default since 7.6 = ask.
StrictHostKeyChecking accept-new

# SyslogFacility <facility>
#   Default: USER.
SyslogFacility USER

# TCPKeepAlive <yes|no>
#   OS-level TCP keepalive (different from ServerAliveInterval which is SSH-level).
TCPKeepAlive yes

# Tag <string>
#   OpenSSH 9.4+ — tag this stanza for use with `Match tagged <string>`.
Tag work

# Tunnel <yes|no|point-to-point|ethernet>
#   Forward layer-2/3 tunnels via the tun/tap interface. Niche.
Tunnel no

# TunnelDevice <local_tun[:remote_tun]>
#   Tunnel device numbers (any picks first free).
TunnelDevice any:any

# UpdateHostKeys <yes|no|ask>
#   When server presents NEW host keys (after rotation), update known_hosts automatically.
UpdateHostKeys yes

# User <username>
#   Remote login user.
User alice

# UserKnownHostsFile <path...>
UserKnownHostsFile ~/.ssh/known_hosts ~/.ssh/known_hosts.d/work

# VerifyHostKeyDNS <yes|no|ask>
#   Look up SSHFP DNS records (DNSSEC-validated) to verify host keys without TOFU.
VerifyHostKeyDNS yes

# VisualHostKey <yes|no>
#   Print ASCII-art "randomart" of host key on connect (helps spot key changes).
VisualHostKey yes

# XAuthLocation <path>
#   Path to xauth (rarely needed).
XAuthLocation /usr/bin/xauth
```

## Common ssh_config Recipes

```bash
# ~/.ssh/config — canonical layout

# 1) Specific Host stanzas first
Host bastion
    HostName bastion.example.com
    User jumpbox
    IdentityFile ~/.ssh/id_ed25519_work
    IdentitiesOnly yes
    ControlMaster auto
    ControlPath ~/.ssh/cm-%r@%h:%p
    ControlPersist 4h

Host prod-*
    User deploy
    ProxyJump bastion
    IdentityFile ~/.ssh/id_ed25519_prod
    IdentitiesOnly yes

Host github.com
    User git
    IdentityFile ~/.ssh/id_ed25519_github
    IdentitiesOnly yes
    AddKeysToAgent yes
    UseKeychain yes

Host work-laptop
    HostName 10.0.5.50
    User alice
    Match exec "test -e /tmp/.work-vpn-up"
        ProxyJump work-bastion
    Match exec "! test -e /tmp/.work-vpn-up"
        ProxyJump public-bastion

# 2) Catch-all LAST
Host *
    AddKeysToAgent yes
    ServerAliveInterval 60
    ServerAliveCountMax 3
    HashKnownHosts yes
    StrictHostKeyChecking accept-new
    UpdateHostKeys yes
    VisualHostKey yes
```

```bash
# Modular split via Include
# ~/.ssh/config
Include ~/.ssh/config.d/*

# ~/.ssh/config.d/work
Host work-*
    User alice
    IdentityFile ~/.ssh/id_work
    IdentitiesOnly yes

# ~/.ssh/config.d/personal
Host personal-*
    User stevie
    IdentityFile ~/.ssh/id_personal
    IdentitiesOnly yes
```

```bash
# Anti-pattern: forgetting IdentitiesOnly with many agent keys
# Broken — agent offers EVERY loaded key; server limits to 6 attempts → "Too many auth failures"
Host prod
    HostName prod.example.com
    IdentityFile ~/.ssh/id_prod_specific

# Fixed
Host prod
    HostName prod.example.com
    IdentityFile ~/.ssh/id_prod_specific
    IdentitiesOnly yes
```

## known_hosts

Trust-on-first-use database of server host keys at `~/.ssh/known_hosts`. The first time you connect, you accept the key; from then on, ssh checks every connection against this file.

```bash
~/.ssh/known_hosts            # per-user
/etc/ssh/ssh_known_hosts      # system-wide (managed by config-management)

# Format (each line):
# [host[,host]...] keytype keydata [comment]
# Or with HashKnownHosts:
# |1|<base64-salt>|<base64-hash> keytype keydata [comment]

# Inspect plain-text entries
grep example.com ~/.ssh/known_hosts

# Find a host even when hashed
ssh-keygen -F example.com

# Remove a stale entry (after legitimate server rebuild)
ssh-keygen -R example.com
ssh-keygen -R '[bastion.example.com]:2222'    # non-default port form

# Pre-populate known_hosts (avoid TOFU prompt on first connect)
ssh-keyscan -t ed25519,rsa example.com >> ~/.ssh/known_hosts
ssh-keyscan -p 2222 -t ed25519 bastion.example.com >> ~/.ssh/known_hosts

# Hash an existing known_hosts file (privacy if stolen)
ssh-keygen -H -f ~/.ssh/known_hosts
# Original is saved as known_hosts.old

# Verify a server's fingerprint before TOFU (out-of-band check)
ssh -o StrictHostKeyChecking=ask user@host
# The authenticity of host 'host (1.2.3.4)' can't be established.
# ED25519 key fingerprint is SHA256:abc123...
# Compare against value posted on the company's wiki / printed by sysadmin.
```

### "Remote host identification has changed" Decision Tree

```
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that a host key has just been changed.
The fingerprint for the ED25519 key sent by the remote host is
SHA256:abc123...
Add correct host key in /home/alice/.ssh/known_hosts to get rid of this message.
Offending ED25519 key in /home/alice/.ssh/known_hosts:42
Host key verification failed.
```

Decide: did the server *legitimately* rotate its host key? Verify out-of-band (Slack the sysadmin, check the wiki, ask the cloud console "show host keys"). If yes:

```bash
ssh-keygen -R hostname
ssh hostname        # accept new key
```

If you cannot confirm legitimacy, **stop**. This is what an MITM looks like. Use a different network, then verify.

### SSHFP Records (DNSSEC-Backed Trust)

```bash
# On the server: generate SSHFP records for DNS publication
ssh-keygen -r example.com
# example.com IN SSHFP 1 1 abc123...
# example.com IN SSHFP 1 2 def456...
# example.com IN SSHFP 4 1 ghi789...
# example.com IN SSHFP 4 2 jkl012...
# Type 4 = ed25519, Type 1 = SHA-1, Type 2 = SHA-256

# Publish into DNSSEC-signed zone, then on the client:
# ~/.ssh/config
Host *.example.com
    VerifyHostKeyDNS yes

# Now ssh skips TOFU when DNSSEC-validated SSHFP records match.
```

## SSH Hostname Aliases

The bread-and-butter productivity move. Turn `ssh -i ~/.ssh/id_work -p 2222 deploy@prod-db1.aws.example.com` into `ssh prod`.

```bash
Host prod                               # the alias (you type this)
    HostName prod-db1.aws.example.com   # real DNS name
    User deploy
    Port 2222
    IdentityFile ~/.ssh/id_work
    IdentitiesOnly yes

# Now:
ssh prod
scp file.tgz prod:/tmp/
rsync -avz src/ prod:/srv/dest/

# Aliases also work for git remotes:
# git@prod:org/repo.git → resolves via your ssh_config Host stanza
```

## Port Forwarding

Three flavors, all available on the command line and in config:

| Flag | Direction | Use Case |
| --- | --- | --- |
| `-L localport:remotehost:remoteport` | Local → remote | Tunnel to a service behind a bastion |
| `-R remoteport:localhost:localport` | Remote → local | Expose laptop service to a remote network |
| `-D localport` | Dynamic SOCKS | Browse-via-SSH proxy |

Common helper flags:

```bash
-N           # do not run a remote command (just forward)
-f           # background after auth
-T           # disable PTY (saves resources for tunnel-only)
-g           # gateway: bind to 0.0.0.0 instead of 127.0.0.1 (allow LAN access)
-o ExitOnForwardFailure=yes   # die if a forward can't bind (don't silently miss)
```

### Local Forward (-L)

```bash
# Tunnel local 5432 → remote db.internal:5432 via bastion
ssh -L 5432:db.internal:5432 -N -f bastion.example.com
psql -h localhost -p 5432 -U app appdb

# Bind to LAN interface so other hosts can use the tunnel
ssh -g -L 5432:db.internal:5432 -N bastion.example.com

# In ~/.ssh/config
Host bastion-with-db
    HostName bastion.example.com
    LocalForward 5432 db.internal:5432
    LocalForward 6379 redis.internal:6379
    ExitOnForwardFailure yes
    SessionType none
```

### Remote Forward (-R)

```bash
# Expose laptop's local web service (8080) on a public VPS as 9090
ssh -R 9090:localhost:8080 -N -f vps.example.com

# Bind on all interfaces of the remote (requires GatewayPorts on the remote sshd)
ssh -R 0.0.0.0:9090:localhost:8080 vps.example.com

# Phone home: expose laptop sshd via remote VPS
ssh -R 2222:localhost:22 -N -f vps.example.com
# From elsewhere:
ssh -p 2222 alice@vps.example.com

# In ~/.ssh/config
Host vps-tunnel
    HostName vps.example.com
    RemoteForward 9090 localhost:8080
    ExitOnForwardFailure yes
```

### Dynamic SOCKS (-D)

```bash
# Per-application SOCKS5 proxy on localhost:1080
ssh -D 1080 -N -f bastion.example.com

# Browser: configure SOCKS5 to localhost:1080 (Firefox: about:preferences#general → Network Settings)
# CLI:
curl --socks5-hostname localhost:1080 https://internal.example.com/
git -c http.proxy=socks5h://localhost:1080 clone https://internal.example.com/repo.git
```

## Jump Hosts

Bastion pattern: a hardened gateway is the only host with sshd open to the internet; internal hosts are reachable only from the bastion. Two mechanisms:

```bash
# Modern (OpenSSH 7.3+): ProxyJump
ssh -J bastion.example.com prod-db1
ssh -J bastion1,bastion2 prod-db1            # cascade through two jumps

# In ~/.ssh/config
Host bastion
    HostName bastion.example.com
    User jumpbox
    IdentityFile ~/.ssh/id_work

Host prod-*
    ProxyJump bastion
    User deploy
    IdentityFile ~/.ssh/id_work
    IdentitiesOnly yes

# Now: ssh prod-db1
# Behind the scenes: ssh bastion → tunnel → ssh prod-db1

# ProxyJump %h:%p — useful when target name resolves the same on the bastion
Host *.internal.example.com
    ProxyJump bastion
    HostName %h        # or HostName %h.internal.example.com
```

```bash
# Legacy: ProxyCommand (still needed for non-ssh transports — see "ProxyCommand vs ProxyJump")
Host prod-*
    ProxyCommand ssh bastion -W %h:%p
```

## SCP and rsync over SSH

`scp` is convenient but stuck on a 1990s protocol; OpenSSH 8.8+ uses SFTP under the hood when possible. `rsync -e ssh` is the workhorse for large/incremental transfers.

```bash
# scp
scp file.tgz user@host:/tmp/                       # local → remote
scp user@host:/tmp/file.tgz .                      # remote → local
scp -r ./dist/ user@host:/var/www/html/            # recursive
scp -P 2222 file.tgz user@host:/tmp/               # non-default port (capital P!)
scp -i ~/.ssh/id_work file.tgz user@host:/tmp/     # specific key
scp -3 hostA:/srv/file.tgz hostB:/tmp/             # copy via local ("3-corner")
scp -O file.tgz user@host:/tmp/                    # OpenSSH 9.0+: force legacy scp protocol
                                                    # (default is now SFTP-based; -O for old servers)

# Common scp pitfall: forgetting that ":" denotes remote
scp ./file user@host                               # broken — interpreted as local copy "user@host"
scp ./file user@host:                              # fixed — copies to remote $HOME
scp ./file user@host:.                             # also fine
```

```bash
# rsync — preferred for >100MB or repeated syncs
rsync -avzP --partial-dir=.rsync-partial \
      -e 'ssh -i ~/.ssh/id_work -p 2222' \
      ./src/ user@host:/srv/dest/
#   -a archive (-rlptgoD): recursive, preserve perms, times, symlinks
#   -v verbose
#   -z compress (skip on fast LAN — CPU bottleneck)
#   -P show progress + --partial (resume on disconnect)
#   --partial-dir=.rsync-partial — store partials in a hidden subdir for safe resume
#   -e specifies the remote shell command

# DRY RUN FIRST — rsync deletes are irreversible
rsync -avzn --delete ./src/ user@host:/srv/dest/   # -n / --dry-run

# Excludes / includes
rsync -avz \
      --exclude='*.tmp' --exclude='node_modules/' --exclude='.git/' \
      --include='dist/' \
      ./ user@host:/srv/app/

# Mirror — make remote identical to local (deletes orphans on remote)
rsync -avzP --delete ./src/ user@host:/srv/dest/

# Via ProxyJump
rsync -avz -e 'ssh -J bastion' ./src/ user@host:/srv/dest/

# Resume aborted scp-style transfer
rsync -avzP --append-verify ./bigfile.iso user@host:/tmp/
```

| Tool | Resume | Compress | Incremental | Encrypted? |
| --- | --- | --- | --- | --- |
| `scp` | No | No | No | Yes (over SSH) |
| `sftp -r` | Partial | No | No | Yes (over SSH) |
| `rsync -e ssh` | Yes (`--partial`) | Yes (`-z`) | Yes (deltas) | Yes (over SSH) |

## SFTP

Interactive file transfer subsystem. Server-side: `Subsystem sftp internal-sftp` (or `/usr/lib/openssh/sftp-server`).

```bash
# Interactive
sftp user@host
sftp -P 2222 user@host
sftp -i ~/.ssh/id_work user@host
sftp -J bastion user@host

# Common interactive commands:
#   ls, cd, pwd        remote
#   lls, lcd, lpwd     local
#   get <remote>        download
#   put <local>         upload
#   get -r <dir>        recursive
#   mput a/* b/*        upload multiple
#   mkdir, rmdir, rm    remote operations
#   chmod, chown, chgrp on remote
#   bye / quit / exit
#   !cmd                run cmd on local shell

# Non-interactive batchfile
cat > /tmp/sftp.batch <<'EOF'
cd /uploads
put ./report.pdf
put -r ./logs
ls -l
bye
EOF
sftp -b /tmp/sftp.batch user@host

# One-liner (single command)
echo "put report.pdf" | sftp -b - user@host
```

### SFTP-Only Accounts (Chroot Jail)

Lock a user to SFTP, no shell, jailed to a directory.

```bash
# /etc/ssh/sshd_config (or drop-in)
Match Group sftp-only
    ChrootDirectory /srv/sftp/%u
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
    PermitTunnel no

# Server-side prep (chroot requirements):
sudo groupadd sftp-only
sudo useradd -g sftp-only -s /sbin/nologin uploader
sudo mkdir -p /srv/sftp/uploader/incoming
sudo chown root:root /srv/sftp /srv/sftp/uploader     # CHROOT root MUST be root-owned + not writable
sudo chmod 755 /srv/sftp /srv/sftp/uploader
sudo chown uploader:sftp-only /srv/sftp/uploader/incoming
sudo systemctl reload sshd
```

## SSHFS

Mount a remote directory over SSH as a local filesystem (FUSE).

```bash
# Linux
sudo apt install sshfs
sshfs user@host:/srv/data /mnt/data \
      -o reconnect,IdentityFile=~/.ssh/id_ed25519,allow_other,default_permissions,idmap=user

# macOS — uses macFUSE + sshfs
brew install --cask macfuse
brew install gromgit/fuse/sshfs-mac
sshfs user@host:/srv/data ~/sshfs/data -o reconnect,IdentityFile=~/.ssh/id_ed25519,defer_permissions

# Unmount
fusermount -u /mnt/data        # Linux
umount ~/sshfs/data            # macOS

# Useful options
#   reconnect           re-establish on disconnect
#   allow_other         other local users can access (requires user_allow_other in /etc/fuse.conf)
#   idmap=user          map remote user → local user
#   ServerAliveInterval=15  pass-through to ssh
#   cache_timeout=300   cache dir entries (perf)
#   compression=no      skip on fast networks

# Common gotcha: stale mount after laptop sleep
# Symptom: ls /mnt/data hangs; fusermount -u says "device or resource busy"
# Fix: lazy unmount, then remount
fusermount -uz /mnt/data
sshfs user@host:/srv/data /mnt/data -o reconnect
```

## SOCKS Proxy Patterns

```bash
# Tunnel browsing through a corporate VPN/bastion
ssh -D 1080 -N -f bastion.example.com

# Browser config: SOCKS5 host=127.0.0.1 port=1080  (Firefox: also tick "Proxy DNS when using SOCKS v5")

# Per-command proxying
curl --socks5-hostname localhost:1080 https://internal.example.com/
# --socks5-hostname (DNS done remotely) vs --socks5 (DNS done locally — leaks names!)

git config --global http.proxy socks5h://localhost:1080      # all https
git config --global https.example.com.proxy socks5h://localhost:1080  # only one host

# Apt, npm, pip, docker pull — all support HTTP_PROXY=socks5h://localhost:1080 / ALL_PROXY
ALL_PROXY=socks5h://localhost:1080 npm install
ALL_PROXY=socks5h://localhost:1080 docker pull internal.registry/image:tag

# Toggle via systemd-resolved? Not necessary — SOCKS5h does its own DNS over the tunnel.

# In ~/.ssh/config
Host bastion-socks
    HostName bastion.example.com
    DynamicForward 1080
    ExitOnForwardFailure yes
    SessionType none
    ServerAliveInterval 30
```

## SSH Multiplexing

A single TCP connection carries many SSH sessions. Second `ssh prod` after the first connects in <100ms (no TCP+TLS+auth round trips).

```bash
# ~/.ssh/config
Host *
    ControlMaster auto
    ControlPath ~/.ssh/cm-%C       # %C = hash(user, host, port) — short, safe path length
    ControlPersist 30m

# Older form (more readable but longer paths can hit UNIX_PATH_MAX = 108)
Host *
    ControlPath ~/.ssh/cm-%r@%h:%p

# Manage the master
ssh -O check prod          # is it alive?
# Master running (pid=12345)

ssh -O exit prod           # kill master gracefully (sessions warned)
ssh -O stop prod           # stop accepting new connections; existing remain

# Forward sharing — add a forward to a running master
ssh -O forward -L 5432:db.internal:5432 prod
ssh -O cancel  -L 5432:db.internal:5432 prod
```

```bash
# Stale-socket pitfall — first session crashes, socket file remains, new ssh hangs
# Symptom: ssh prod hangs at "mux_client_request_session: read from master failed"
# Fix:
ssh -O exit prod 2>/dev/null
rm -f ~/.ssh/cm-* ~/.ssh/cm-*@*

# Or set a short fallback timeout
Host *
    ControlMaster auto
    ControlPath ~/.ssh/cm-%C
    ControlPersist 30m
    ConnectTimeout 5            # don't hang forever waiting for a dead master
```

## Server Setup (sshd)

Server-side configuration lives in `/etc/ssh/sshd_config` (and `/etc/ssh/sshd_config.d/*.conf` if `Include` is in the main file, default since Debian 12 / Ubuntu 22.04).

```bash
# Validate config (catches typos; safer than restarting blind)
sudo sshd -t

# Show effective config (resolves Match blocks, includes, defaults)
sudo sshd -T

# Show effective config for a specific match context
sudo sshd -T -C user=alice,host=10.0.5.5,addr=10.0.5.5

# Reload (most settings) vs restart (port, listen address)
sudo systemctl reload sshd
sudo systemctl restart sshd

# ALWAYS keep an existing session open while changing sshd; if you lock yourself out
# you can fix it from the still-connected session.
```

### sshd_config Keywords

```bash
# AcceptEnv <list>
#   Which env vars the client may set via SendEnv. Default: none.
AcceptEnv LANG LC_*

# AddressFamily <any|inet|inet6>
AddressFamily inet

# AllowAgentForwarding <yes|no>
#   Permit -A agent forwarding from this server.
AllowAgentForwarding no

# AllowGroups <group ...>
#   Whitelist by primary or supplementary group. Pattern lists allowed.
AllowGroups ssh-users admins

# AllowStreamLocalForwarding <yes|no|local|remote>
#   UNIX-domain forwarding. Default yes.
AllowStreamLocalForwarding no

# AllowTcpForwarding <yes|no|local|remote>
#   -L (local), -R (remote), or both/none.
AllowTcpForwarding yes

# AllowUsers <user ...>
#   Whitelist users. Patterns allowed (deploy@10.0.0.0/8).
AllowUsers deploy alice bob@10.0.0.0/8

# AuthenticationMethods <list[,list...]>
#   Each comma-separated set is an "OR"; within a set "+"-joined methods are required AND.
AuthenticationMethods publickey,keyboard-interactive   # require BOTH (key + 2FA)

# AuthorizedKeysCommand <path>
#   Run this command to fetch authorized keys (instead of file). Stdout is treated as the file.
AuthorizedKeysCommand /usr/sbin/auth-keys-from-ldap %u

# AuthorizedKeysCommandUser <user>
#   The user the AuthorizedKeysCommand runs as. REQUIRED if Command set.
AuthorizedKeysCommandUser nobody

# AuthorizedKeysFile <path...>
#   Default: .ssh/authorized_keys .ssh/authorized_keys2  (relative to user's home)
AuthorizedKeysFile .ssh/authorized_keys /etc/ssh/authorized_keys.d/%u

# AuthorizedPrincipalsCommand / AuthorizedPrincipalsCommandUser
#   Like AuthorizedKeysCommand but yields PRINCIPAL names (used with SSH certificates).

# AuthorizedPrincipalsFile <path>
AuthorizedPrincipalsFile /etc/ssh/auth_principals/%u

# Banner <path>
#   Pre-auth banner shown to client.
Banner /etc/ssh/banner.txt

# CASignatureAlgorithms <list>
#   CA cert signature algorithms accepted. Use +/- vs default.
CASignatureAlgorithms ssh-ed25519,rsa-sha2-512

# ChallengeResponseAuthentication <yes|no>
#   Legacy alias. Use KbdInteractiveAuthentication.

# ChrootDirectory <path>
#   Chroot the user after auth. Path components must be root-owned + not group/world-writable.
ChrootDirectory /srv/sftp/%u

# Ciphers <list>
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com

# ClientAliveCountMax <N>
#   How many missed ClientAlive probes before disconnect. Default 3.
ClientAliveCountMax 3

# ClientAliveInterval <seconds>
#   Server sends a probe every N seconds when channel is idle. Default 0 (disabled).
ClientAliveInterval 60

# Compression <yes|no|delayed>
Compression delayed

# DenyGroups <group ...>
DenyGroups deny-ssh

# DenyUsers <user ...>
DenyUsers eve mallory

# DisableForwarding <yes|no>
#   Master switch — disables ALL forwarding (TCP, X11, agent, local). Default no.
DisableForwarding no

# ExposeAuthInfo <yes|no>
#   Write auth method info into a file readable via $SSH_USER_AUTH inside the session.
ExposeAuthInfo no

# ForceCommand <command>
#   Run only this command on login (overrides any user-supplied command). Use for SFTP-only.
ForceCommand internal-sftp

# GSSAPIAuthentication <yes|no>
GSSAPIAuthentication no

# GatewayPorts <yes|no|clientspecified>
#   no = -R binds 127.0.0.1 only (default). yes = always 0.0.0.0. clientspecified = -R bind decides.
GatewayPorts no

# HostCertificate <path>
#   Server's signed certificate (replacing host key TOFU with a CA).
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub

# HostKey <path>
#   Server private host key. Multiple HostKey lines = multiple algorithms supported.
HostKey /etc/ssh/ssh_host_ed25519_key
HostKey /etc/ssh/ssh_host_rsa_key

# HostKeyAlgorithms <list>
#   Algorithms the server offers (and what it advertises in keyx).
HostKeyAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# HostbasedAcceptedAlgorithms / HostbasedAuthentication / HostbasedUsesNameFromPacketOnly
#   Legacy host-based auth. Almost always: no.
HostbasedAuthentication no

# IPQoS <interactive bulk>
IPQoS af21 cs1

# IgnoreRhosts <yes|no>
#   Don't read user .rhosts/.shosts. Default yes (good).
IgnoreRhosts yes

# IgnoreUserKnownHosts <yes|no>
#   For host-based auth. Default no.
IgnoreUserKnownHosts no

# KbdInteractiveAuthentication <yes|no>
KbdInteractiveAuthentication no

# KerberosAuthentication <yes|no>
KerberosAuthentication no

# KexAlgorithms <list>
KexAlgorithms sntrup761x25519-sha512@openssh.com,curve25519-sha256,curve25519-sha256@libssh.org

# ListenAddress <ip[:port]>
#   Bind only to specific IPs/ports. Default: all interfaces.
ListenAddress 10.0.5.5
ListenAddress [::1]:22

# LogLevel <QUIET|FATAL|ERROR|INFO|VERBOSE|DEBUG|DEBUG1|DEBUG2|DEBUG3>
LogLevel VERBOSE   # logs key fingerprint of accepted key — handy for forensics

# LoginGraceTime <seconds>
#   Time to complete authentication. 0 = unlimited (don't!).
LoginGraceTime 30

# MACs <list>
MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com

# Match <criteria>
#   Conditional block. See "Match Blocks" below.

# MaxAuthTries <N>
#   Max failed auth attempts per connection. Default 6 — drop to 3.
MaxAuthTries 3

# MaxSessions <N>
#   Max sessions/channels per connection. Default 10.
MaxSessions 5

# MaxStartups <start[:rate:full]>
#   Concurrent unauthenticated connections. Default 10:30:100 (drop 30% at 10, all at 100).
MaxStartups 10:30:60

# ModuliFile <path>
#   DH moduli file. Default /etc/ssh/moduli. Regenerate periodically (hours of compute).

# PasswordAuthentication <yes|no>
PasswordAuthentication no

# PermitEmptyPasswords <yes|no>
#   NEVER yes. Default no.
PermitEmptyPasswords no

# PermitListen <[bind:]port [bind:]port ...>
#   For -R, restrict which (bind, port) pairs the client may listen on.
PermitListen 9000 9001 localhost:9090

# PermitOpen <host:port [host:port ...]>
#   For -L, restrict which (host, port) pairs the client may open.
PermitOpen db.internal:5432 cache.internal:6379

# PermitRootLogin <yes|no|prohibit-password|forced-commands-only>
#   Default since 7.0 = prohibit-password (key only, no password). prefer "no".
PermitRootLogin no

# PermitTTY <yes|no>
#   Whether to allocate a PTY. no = tunnel-only / scripted accounts.
PermitTTY no

# PermitTunnel <yes|no|point-to-point|ethernet|all>
PermitTunnel no

# PermitUserEnvironment <yes|no|pattern_list>
#   Allow user's ~/.ssh/environment and authorized_keys environment="...".
PermitUserEnvironment no

# PermitUserRC <yes|no>
#   Run ~/.ssh/rc on session start.
PermitUserRC yes

# PerSourceMaxStartups <N|none>
#   Limit unauthenticated connections per source CIDR. Default: none.
PerSourceMaxStartups 10

# PerSourceNetBlockSize <ipv4_prefix[:ipv6_prefix]>
#   Bucket size for PerSourceMaxStartups. Default 32:128 (one bucket per IP).
PerSourceNetBlockSize 24:48

# PidFile <path>
PidFile /var/run/sshd.pid

# Port <number>
Port 22

# PrintLastLog / PrintMotd
PrintLastLog yes
PrintMotd no

# PubkeyAcceptedAlgorithms <list>
PubkeyAcceptedAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# PubkeyAuthOptions <option_list>
#   no-touch-required, verify-required (for FIDO2 keys).
PubkeyAuthOptions touch-required

# PubkeyAuthentication <yes|no>
PubkeyAuthentication yes

# RDomain <name>
#   Routing domain for OpenBSD. Niche.

# RekeyLimit <data [time]>
RekeyLimit 1G 1h

# RequiredRSASize <bits>
#   Minimum RSA key size accepted. OpenSSH 9.1+. Default 1024 (raise!).
RequiredRSASize 3072

# RevokedKeys <path>
#   Public keys (or KRL) listed here are NEVER trusted, even if in authorized_keys / signed by CA.
RevokedKeys /etc/ssh/revoked_keys

# SecurityKeyProvider <path>
#   FIDO2 middleware override.

# SetEnv <NAME=VALUE>
SetEnv SSHD_LANG=en_US.UTF-8

# StreamLocalBindMask <octal>
StreamLocalBindMask 0177

# StreamLocalBindUnlink <yes|no>
StreamLocalBindUnlink yes

# StrictModes <yes|no>
#   Refuse keys if file/dir perms are wrong. KEEP yes.
StrictModes yes

# Subsystem <name> <command>
#   Default sftp subsystem.
Subsystem sftp internal-sftp
# Subsystem sftp /usr/lib/openssh/sftp-server -f AUTHPRIV -l INFO

# SyslogFacility <facility>
SyslogFacility AUTH

# TCPKeepAlive <yes|no>
TCPKeepAlive yes

# TrustedUserCAKeys <path>
#   Public keys of CAs that may sign user certificates trusted by this host.
TrustedUserCAKeys /etc/ssh/user_ca.pub

# UseDNS <yes|no>
#   Reverse DNS lookups on client IPs. Slows logins; default no since 6.8.
UseDNS no

# UsePAM <yes|no>
UsePAM yes

# UserKnownHostsFile (server-side host-based auth — niche)

# VersionAddendum <string>
#   Append to version banner. Empty = ssh-2.0-OpenSSH_9.6 only.
VersionAddendum none

# X11DisplayOffset <N>
X11DisplayOffset 10

# X11Forwarding <yes|no>
X11Forwarding no

# X11UseLocalhost <yes|no>
X11UseLocalhost yes
```

## sshd Hardening

Modern Mozilla "Modern" profile + sensible production defaults.

```bash
# /etc/ssh/sshd_config.d/50-hardening.conf

# --- Authentication ---
PermitRootLogin no
PasswordAuthentication no
PubkeyAuthentication yes
KbdInteractiveAuthentication no
AuthorizedKeysFile .ssh/authorized_keys
PermitEmptyPasswords no

# --- Access control ---
AllowGroups ssh-users
MaxAuthTries 3
MaxSessions 5
LoginGraceTime 30
PerSourceMaxStartups 10:30:60

# --- Forwarding ---
AllowAgentForwarding no
AllowTcpForwarding no
AllowStreamLocalForwarding no
GatewayPorts no
X11Forwarding no
PermitTunnel no
PermitUserEnvironment no

# --- Crypto (Mozilla Modern, 2024+) ---
KexAlgorithms sntrup761x25519-sha512@openssh.com,curve25519-sha256,curve25519-sha256@libssh.org
HostKeyAlgorithms ssh-ed25519,ssh-ed25519-cert-v01@openssh.com,rsa-sha2-512,rsa-sha2-256
PubkeyAcceptedAlgorithms ssh-ed25519,ssh-ed25519-cert-v01@openssh.com,rsa-sha2-512,rsa-sha2-256
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com,aes256-ctr,aes192-ctr,aes128-ctr
MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com,umac-128-etm@openssh.com
RequiredRSASize 3072

# --- Misc ---
ClientAliveInterval 60
ClientAliveCountMax 3
LogLevel VERBOSE
UseDNS no
PrintMotd no
StrictModes yes
```

```bash
# Apply
sudo sshd -t
sudo systemctl reload sshd
```

```bash
# Optional: replace MaxAuthTries with fail2ban for actual rate limiting
sudo apt install fail2ban
# /etc/fail2ban/jail.d/sshd.local
[sshd]
enabled = true
maxretry = 3
findtime = 10m
bantime = 24h
```

## SSH Certificates / CA-Signed Keys

Replace per-host TOFU and per-user authorized_keys management with a single CA. Each user/host gets a short-lived certificate signed by the CA; servers trust the CA, not individual keys.

```bash
# 1. Generate a USER CA (the key that signs user identities)
ssh-keygen -t ed25519 -f ~/.ssh/user_ca -C "user-ca-2026"

# 2. Generate a HOST CA (the key that signs host keys)
ssh-keygen -t ed25519 -f ~/.ssh/host_ca -C "host-ca-2026"

# 3. Sign a user public key
ssh-keygen -s ~/.ssh/user_ca \
           -I "alice@laptop-2026"           \  # certificate ID (logs/audit trail)
           -n alice,deploy,readonly         \  # principals (which "users" this cert is for)
           -V +52w                          \  # validity 52 weeks
           -t rsa-sha2-512                  \  # signature alg (irrelevant for ed25519 CA)
           ~/.ssh/id_ed25519.pub
# Output: id_ed25519-cert.pub

# 4. Sign a host public key
ssh-keygen -s ~/.ssh/host_ca \
           -I "prod-db1.example.com"        \
           -h                               \  # this is a HOST cert, not user
           -n prod-db1,prod-db1.example.com,10.0.5.5 \  # acceptable hostnames/IPs
           -V +52w                          \
           /etc/ssh/ssh_host_ed25519_key.pub

# 5. View certificate contents
ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub
# Type: ssh-ed25519-cert-v01@openssh.com user certificate
# Public key: ED25519-CERT SHA256:...
# Signing CA: ED25519 SHA256:... (using ssh-ed25519)
# Key ID: "alice@laptop-2026"
# Serial: 0
# Valid: from 2026-04-25T12:00:00 to 2027-04-24T12:00:00
# Principals:
#         alice
#         deploy
# Critical Options: (none)
# Extensions:
#         permit-X11-forwarding
#         permit-agent-forwarding
#         ...
```

```bash
# CLIENT side (ssh_config) — present the cert
Host prod-*
    IdentityFile ~/.ssh/id_ed25519
    CertificateFile ~/.ssh/id_ed25519-cert.pub

# SERVER side (sshd_config)
TrustedUserCAKeys /etc/ssh/user_ca.pub      # trust this CA for user auth
HostCertificate /etc/ssh/ssh_host_ed25519_key-cert.pub   # serve our host cert

# Optional: per-user principal mapping
# /etc/ssh/auth_principals/alice contains the principals that user "alice" may use
AuthorizedPrincipalsFile /etc/ssh/auth_principals/%u
```

```bash
# CLIENT side — trust the host CA (no more TOFU per host)
# ~/.ssh/known_hosts
@cert-authority *.example.com ssh-ed25519 AAAAC3Nz... host-ca-2026
```

```bash
# Revocation — Key Revocation List
ssh-keygen -k -f /etc/ssh/krl -s ~/.ssh/user_ca \
           ~/.ssh/revoked_alice.pub             # revoke specific pubkey
# Or specify by serial / key-id:
echo "id: alice@old-laptop" >> krl_spec
echo "serial: 42" >> krl_spec
ssh-keygen -k -f /etc/ssh/krl -s ~/.ssh/user_ca krl_spec

# sshd_config
RevokedKeys /etc/ssh/krl
```

## Match Blocks

```bash
# /etc/ssh/sshd_config

# Default: aggressive
PermitRootLogin no
PasswordAuthentication no
AllowTcpForwarding no

# Override for a specific user
Match User alice
    AuthorizedKeysFile /etc/ssh/keys/alice/authorized_keys

# Override for a group
Match Group admins
    AllowAgentForwarding yes
    AllowTcpForwarding yes
    PermitOpen any

# Override by source address
Match Address 10.0.0.0/24
    PasswordAuthentication yes      # internal LAN only

# Combine criteria
Match User backups Address 10.0.5.0/24
    ForceCommand /usr/local/bin/backup-handler
    AllowTcpForwarding no
    X11Forwarding no
    PermitTTY no

# LocalPort matching (if listening on multiple ports)
Match LocalPort 2222
    AuthenticationMethods publickey,publickey

# Match all (catch-all reset; often used at end)
Match all
```

Match keywords supported: `User`, `Group`, `Host`, `LocalAddress`, `LocalPort`, `Address`, `RDomain`. Combine with whitespace = AND. Negate with `!` (e.g., `Match User !root`). The "Match-overrides-defaults" pattern: put strict defaults at the top of the file, then `Match` blocks below for specific exceptions. `Match` does NOT inherit the previous block; it starts fresh from the defaults.

## SSH Agent Forwarding

`ssh -A` (or `ForwardAgent yes`) makes the local agent socket available on the remote, so a second hop from the remote can authenticate using your local keys.

```bash
ssh -A bastion
ssh prod-db1                # uses YOUR keys via the forwarded agent

# Config form
Host bastion
    ForwardAgent yes
```

**SECURITY**: anyone with root on the bastion can read the forwarded socket and use your keys to sign auth challenges to anywhere they like — including hosts you've never visited. Worse, this includes any process running as your user (compromised shell rc, etc.).

Mitigations:

```bash
# 1. Use ProxyJump instead — only the bastion forwards stdio, your agent stays local
ssh -J bastion prod-db1

# 2. If you must forward, require touch on every signature (FIDO2 keys do this naturally)
ssh-add -c ~/.ssh/id_ed25519       # confirm-on-use

# 3. Use an isolated agent for forwarding (separate keys)
SSH_AUTH_SOCK=/tmp/forward-agent.sock ssh-agent -a /tmp/forward-agent.sock
SSH_AUTH_SOCK=/tmp/forward-agent.sock ssh-add ~/.ssh/id_jumpkey
SSH_AUTH_SOCK=/tmp/forward-agent.sock ssh -A bastion

# 4. Restrict agent forwarding on bastion sshd
AllowAgentForwarding no
```

## X11 Forwarding

```bash
ssh -X user@host xeyes        # untrusted X11 (security extension limits)
ssh -Y user@host xeyes        # trusted X11 (no limits — can keylog your local)

# Config form
Host gui-host
    ForwardX11 yes
    ForwardX11Trusted no       # prefer untrusted

# On the server
X11Forwarding yes
X11UseLocalhost yes            # bind X server display to localhost only
X11DisplayOffset 10            # start at :10 to avoid clashing with local X
```

```bash
# Common error
# X11 forwarding request failed on channel 0
# Causes:
#   - sshd has X11Forwarding no (server admin disabled it)
#   - xauth missing on server (apt install xauth)
#   - $DISPLAY ends up unset (sshd didn't allocate one — check sshd logs)
```

X11 forwarding is rarely needed in 2026 — VNC/RDP/Wayland-noVNC/Mosh+TUI cover most "remote GUI" cases.

## SSH Configuration Conditionals

```bash
# Conditional based on VPN state
Match exec "test -e /tmp/.vpn-up"
    ProxyJump corp-bastion
    User corp-alice

Match exec "! test -e /tmp/.vpn-up"
    ProxyJump public-bastion
    User pub-alice

# Conditional based on network (e.g. on-LAN vs off-LAN)
Match host laptop exec "iwgetid -r | grep -q work-wifi"
    HostName 10.0.5.5

Match host laptop exec "! iwgetid -r | grep -q work-wifi"
    HostName laptop.dyn.example.com

# Conditional based on cloud metadata
Match exec "curl -s -m 1 http://169.254.169.254/latest/ > /dev/null 2>&1"
    User ec2-user
    IdentityFile ~/.ssh/aws-keypair
```

```bash
# Modular config via Include + per-environment files
# ~/.ssh/config
Include ~/.ssh/config.d/work
Include ~/.ssh/config.d/personal
Include ~/.ssh/config.d/clients/*

Host *                              # always last
    ServerAliveInterval 60
```

## ProxyCommand vs ProxyJump

```bash
# Modern preferred form (since OpenSSH 7.3, 2016)
ProxyJump bastion              # equivalent CLI: ssh -J bastion target

# Legacy form (still required for non-ssh transports)
ProxyCommand ssh bastion -W %h:%p

# When you STILL need ProxyCommand:
# 1. SSH over an HTTP CONNECT proxy (corkscrew)
ProxyCommand corkscrew proxy.example.com 8080 %h %p

# 2. SSH over Cloudflare Tunnel
ProxyCommand cloudflared access ssh --hostname %h

# 3. SSH via Google Cloud IAP
ProxyCommand gcloud compute start-iap-tunnel %h 22 --listen-on-stdin --project=myproject

# 4. SSH via AWS SSM Session Manager
ProxyCommand sh -c "aws ssm start-session --target %h --document-name AWS-StartSSHSession --parameters portNumber=%p"

# 5. SSH over a TLS tunnel (sslh, stunnel)
ProxyCommand openssl s_client -connect proxy.example.com:443 -servername %h -quiet
```

## SSH over HTTPS

When your firewall only allows :80/:443, SSH can ride on top.

```bash
# GitHub publishes an SSH-over-:443 endpoint:
ssh -p 443 git@ssh.github.com
# Use it via ssh_config:
Host github.com
    HostName ssh.github.com
    Port 443
    User git

# corkscrew through a corporate HTTP CONNECT proxy
sudo apt install corkscrew
# ~/.ssh/config
Host github.com
    HostName ssh.github.com
    Port 443
    User git
    ProxyCommand corkscrew corporate-proxy.example.com 8080 %h %p

# sslh on the server — multiplex 22/SSH and 443/HTTPS on a single port
sudo apt install sslh
# /etc/default/sslh
DAEMON_OPTS="--user sslh --listen 0.0.0.0:443 --ssh 127.0.0.1:22 --tls 127.0.0.1:8443 --pidfile /var/run/sslh.pid"

# Cloudflare Tunnel — internet-routable SSH without exposing port 22
cloudflared tunnel create my-tunnel
cloudflared tunnel route dns my-tunnel ssh.example.com
# config.yml — route ssh through the tunnel
ingress:
  - hostname: ssh.example.com
    service: ssh://localhost:22
  - service: http_status:404
# Client side:
ssh -o ProxyCommand="cloudflared access ssh --hostname %h" ssh.example.com
```

## SSH on Non-Standard Ports

Moving sshd off :22 doesn't add cryptographic security but cuts log noise (drive-by scans hammer :22 constantly).

```bash
# Server side
# /etc/ssh/sshd_config (or drop-in)
Port 2222
# Or multiple
Port 22
Port 2222

# Open the firewall
sudo ufw allow 2222/tcp
sudo firewall-cmd --permanent --add-port=2222/tcp && sudo firewall-cmd --reload

# SELinux — allow sshd to bind on non-default port
sudo semanage port -a -t ssh_port_t -p tcp 2222

# Restart sshd
sudo systemctl restart sshd

# Client side
ssh -p 2222 user@host
scp -P 2222 file user@host:/tmp/        # capital P
sftp -P 2222 user@host
rsync -e 'ssh -p 2222' ./src/ user@host:/dest/

# Persist via ssh_config
Host myserver
    HostName host.example.com
    Port 2222
```

```bash
# fail2ban needs to know the new port
# /etc/fail2ban/jail.d/sshd.local
[sshd]
port = 2222
enabled = true
```

## Verbose Diagnostics

```bash
ssh -v user@host       # debug1
ssh -vv user@host      # debug2 (also packet types)
ssh -vvv user@host     # debug3 (every packet, KEX inputs, etc.)

# What each level reveals:
# -v   reading config, identity files tried, auth methods offered/accepted, channel open
# -vv  + KEX algorithm negotiation, host key check, agent socket details
# -vvv + raw protocol packets, key derivation, every byte of negotiation

# Key lines to grep for:
ssh -v user@host 2>&1 | grep -E '(debug1|Authenticated|Permission denied|Offering)'
```

```text
debug1: Reading configuration data /home/alice/.ssh/config
debug1: /home/alice/.ssh/config line 5: Applying options for prod
debug1: Connecting to prod.example.com [10.0.5.5] port 22.
debug1: Connection established.
debug1: identity file /home/alice/.ssh/id_ed25519 type 3
debug1: kex_input_ext_info: server-sig-algs=<rsa-sha2-256,rsa-sha2-512>
debug1: SSH2_MSG_NEWKEYS sent
debug1: Host 'prod.example.com' is known and matches the ED25519 host key.
debug1: Will attempt key: /home/alice/.ssh/id_ed25519 ED25519 SHA256:abc... explicit
debug1: Authentications that can continue: publickey,password
debug1: Next authentication method: publickey
debug1: Offering public key: /home/alice/.ssh/id_ed25519 ED25519 SHA256:abc... explicit
debug1: Server accepts key: /home/alice/.ssh/id_ed25519 ED25519 SHA256:abc... explicit
debug1: Authentication succeeded (publickey).
```

The two diagnostic gold-standard lines:

```text
debug1: Authentications that can continue: publickey,password
   ^^^ what the SERVER will accept
debug1: Offering public key: <path> <type> SHA256:<fp> <source>
   ^^^ what the CLIENT is trying (and in what order — agent keys offered before file keys unless IdentitiesOnly)
```

```bash
# Server-side debug
sudo journalctl -u sshd -f                # systemd
sudo tail -f /var/log/auth.log            # Debian/Ubuntu
sudo tail -f /var/log/secure              # RHEL/Fedora

# Run sshd in foreground on alternate port (non-disruptive)
sudo /usr/sbin/sshd -d -p 2233            # one connection then exits
sudo /usr/sbin/sshd -ddd -p 2233          # very verbose, multiple connections

# Show effective config after Match resolution
sudo sshd -T -C user=alice,host=10.0.5.5,addr=10.0.5.5
```

## SSH Connection Lifecycle

```
1. TCP open                       client → server :22
2. Banner exchange                "SSH-2.0-OpenSSH_9.6p1"
3. Algorithm negotiation (KEX)    KEX, host key alg, ciphers, MACs, compression
4. Key exchange                   ECDH (curve25519) or hybrid (sntrup761x25519)
5. Server host key                presented + signed; client checks against known_hosts
6. NEWKEYS                        switch to encrypted channel
7. Service request                "ssh-userauth"
8. User auth                      none → server lists allowed → publickey → success
9. Service request                "ssh-connection"
10. Channel open                  session / exec / subsystem (sftp) / direct-tcpip (-L) / forwarded-tcpip (-R)
11. Data flow                     stdin/stdout/stderr multiplexed in channel windows
12. Channel/connection close      EOF on each direction → close → TCP FIN
```

Every step has an error mode. `ssh -vvv` lets you pinpoint which one fails.

## Common Error Messages and Fixes

### `Permission denied (publickey,password)`

The server allowed the connection but rejected every auth method.

```bash
# Check that the server even sees your offered key
ssh -vvv user@host 2>&1 | grep -E '(Authentications that can continue|Offering|Server accepts)'

# Common causes:
# 1. Wrong key tried first; agent has many keys; server's MaxAuthTries hit
#    Fix:
ssh -o IdentitiesOnly=yes -i ~/.ssh/correct_key user@host

# 2. Key exists but server's authorized_keys doesn't have the matching pubkey
#    Verify on server:
sudo -u alice cat /home/alice/.ssh/authorized_keys
ssh-keygen -lf /home/alice/.ssh/authorized_keys     # show fingerprint
# Compare to fingerprint of the key being offered (-vvv shows fp)

# 3. Bad permissions trip StrictModes
chmod 700 ~/.ssh
chmod 600 ~/.ssh/authorized_keys
chown alice:alice ~/.ssh ~/.ssh/authorized_keys
chmod go-w ~

# 4. SELinux: file context wrong after manual edits
restorecon -Rv ~/.ssh

# 5. PasswordAuthentication no on server (password just won't be tried)
#    Server side:
sudo sshd -T | grep -i passwordauth
```

### `Permission denied (publickey)` even though "Server accepts key"

Race / config issue. Usually the wrong AuthorizedKeysFile path on a chroot/multi-user host.

```bash
# Find which authorized_keys sshd actually reads for this user
sudo sshd -T -C user=alice | grep authorizedkeysfile
# authorizedkeysfile .ssh/authorized_keys /etc/ssh/authorized_keys.d/%u

# Check both
ls -la ~alice/.ssh/authorized_keys /etc/ssh/authorized_keys.d/alice 2>&1
```

### `Host key verification failed`

```text
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
```

```bash
# 1. Verify out-of-band that the host key change is legitimate (server rebuild?)
#    Ask sysadmin / check cloud console / compare published fingerprint
ssh-keyscan -t ed25519 host.example.com | ssh-keygen -lf -

# 2. Once verified, remove the stale entry and reconnect
ssh-keygen -R host.example.com
ssh-keygen -R '[host.example.com]:2222'    # if non-default port
ssh host.example.com
```

### `Too many authentication failures`

The agent is offering every key it knows; server hits MaxAuthTries (default 6).

```bash
# Broken — agent has 12 keys; server hits 6 and disconnects
ssh prod-db1
# Received disconnect from 10.0.5.5: Too many authentication failures

# Fixed — IdentitiesOnly + specific key
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_prod prod-db1

# Or in ssh_config:
Host prod-*
    IdentityFile ~/.ssh/id_prod
    IdentitiesOnly yes
```

### `kex_exchange_identification: Connection closed by remote host`

Server-side rate-limit, `tcpwrappers` deny, sshd died, network drop.

```bash
# Check from the client
ssh -vvv user@host 2>&1 | head -30

# Check from the server
sudo journalctl -u sshd | tail -50
sudo tail -50 /var/log/auth.log

# Common cause: PerSourceMaxStartups / fail2ban banned your IP
sudo fail2ban-client status sshd
sudo fail2ban-client set sshd unbanip 1.2.3.4
```

### `no matching host key type found. Their offer: ssh-rsa`

OpenSSH 8.8+ disabled `ssh-rsa` (SHA-1) by default.

```bash
# Broken — modern client, ancient server
ssh user@oldbox
# Unable to negotiate with 10.0.5.5 port 22: no matching host key type found.
# Their offer: ssh-rsa

# Transitional fix (per host)
ssh -o HostKeyAlgorithms=+ssh-rsa -o PubkeyAcceptedAlgorithms=+ssh-rsa user@oldbox

# Or in ssh_config:
Host oldbox
    HostKeyAlgorithms +ssh-rsa
    PubkeyAcceptedAlgorithms +ssh-rsa

# Real fix — regenerate the SERVER's host keys to ed25519:
sudo ssh-keygen -A                              # generates all default host keys
sudo systemctl restart sshd
ssh-keygen -R oldbox                             # remove stale known_hosts entry
ssh oldbox                                       # accept new ed25519 key
```

### `no matching MAC found` / `no matching key exchange method`

```bash
# Algorithm mismatch with very old server (e.g. corp legacy appliance)
# Client supports only modern algs by default; server only knows old ones.

# Broken
ssh appliance
# Unable to negotiate with 10.0.0.10 port 22: no matching key exchange method found.
# Their offer: diffie-hellman-group14-sha1,diffie-hellman-group1-sha1

# Transitional fix (additive, just for this host)
ssh -o KexAlgorithms=+diffie-hellman-group14-sha1 appliance

# In ssh_config — list what the appliance offers as a + override
Host appliance
    KexAlgorithms +diffie-hellman-group14-sha1
    HostKeyAlgorithms +ssh-rsa
    Ciphers +aes128-cbc
    MACs +hmac-sha1
```

### `channel 0: open failed: administratively prohibited`

The server's sshd refuses your forward (PermitOpen / AllowTcpForwarding).

```bash
# Broken — tunnel attempt blocked
ssh -L 5432:db:5432 bastion
# channel 0: open failed: administratively prohibited: open failed

# Fixed — server config (sshd_config)
AllowTcpForwarding yes
PermitOpen db.internal:5432

sudo systemctl reload sshd
```

### `Connection to X closed by remote host` (after idle)

```bash
# Broken — corporate NAT/firewall drops idle TCP connections after ~5 min
# Fix client side
Host *
    ServerAliveInterval 60
    ServerAliveCountMax 3
    TCPKeepAlive yes

# Or fix server side
ClientAliveInterval 60
ClientAliveCountMax 3
```

### `Bad owner or permissions on /home/user/.ssh/config`

```bash
# Broken — group-writable config
chmod 600 ~/.ssh/config
chown $(id -u):$(id -g) ~/.ssh/config
```

### `WARNING: UNPROTECTED PRIVATE KEY FILE!`

```text
Permissions 0644 for '/home/alice/.ssh/id_ed25519' are too open.
It is required that your private key files are NOT accessible by others.
This private key will be ignored.
```

```bash
# Fixed
chmod 600 ~/.ssh/id_ed25519
```

### `Agent admitted failure to sign using the key`

Agent cached an old/bad key. Or the key being offered to the agent isn't actually the one matching the cert.

```bash
ssh-add -d ~/.ssh/id_ed25519     # remove
ssh-add ~/.ssh/id_ed25519        # re-add
# Combined with IdentitiesOnly + IdentityFile:
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 user@host
```

### `ssh: connect to host X port 22: Connection refused`

sshd not running, wrong port, or firewall actively rejecting.

```bash
# Check if sshd is up
sudo systemctl status sshd
sudo ss -tlnp | grep -E ':22\b|:2222\b'

# Test from client side
nc -vz host 22

# Check firewall
sudo iptables -L -n -v | grep 22
sudo ufw status
sudo firewall-cmd --list-all
```

### `ssh: connect to host X port 22: No route to host`

Routing or firewall ICMP-block. Different from "Connection refused".

```bash
ip route get 10.0.5.5         # check the route
ping -c 1 10.0.5.5            # test reachability
traceroute 10.0.5.5
```

### `ssh: Could not resolve hostname X: Name or service not known`

DNS issue.

```bash
getent hosts host.example.com
dig host.example.com
ssh user@10.0.5.5             # bypass DNS with IP
# Quick fix: add to /etc/hosts
echo "10.0.5.5 host.example.com" | sudo tee -a /etc/hosts
```

## Common Gotchas

```bash
# GOTCHA: Forgot ssh-agent + multiple keys → "Too many authentication failures"
# Broken
ssh prod
# Received disconnect: Too many authentication failures
# Fixed
Host prod
    IdentityFile ~/.ssh/id_prod
    IdentitiesOnly yes
```

```bash
# GOTCHA: Agent forwarded across boundaries leaks identity
# Broken — root on bastion can sign arbitrary auth challenges with your laptop's key
ssh -A bastion
# (compromised bastion → attacker uses your forwarded agent socket → ssh anywhere as you)
# Fixed
ssh -J bastion target          # ProxyJump keeps the agent local
```

```bash
# GOTCHA: Committing private keys to git
# Broken
git add ~/.ssh/id_ed25519
# Fixed: .gitignore the entire dir, audit history with truffleHog/gitleaks if leaked
echo '.ssh/' >> .gitignore
gitleaks detect --source . --no-banner
```

```bash
# GOTCHA: ssh-rsa with modern server
# Broken
ssh -o HostKeyAlgorithms=ssh-rsa modernhost     # forces deprecated alg
# Fixed — let modern defaults apply
ssh modernhost
```

```bash
# GOTCHA: Copy-paste from Windows mangles authorized_keys (CRLF + line breaks)
# Broken
notepad authorized_keys → server, key has CRLF + smart quotes
# Fixed
dos2unix ~/.ssh/authorized_keys
sed -i 's/\r$//' ~/.ssh/authorized_keys
# Or just use ssh-copy-id
```

```bash
# GOTCHA: Passphrase-locked dev key with no agent → typed every connection
# Broken
ssh prod   # prompts for passphrase
ssh prod   # prompts again
# Fixed — agent + AddKeysToAgent
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
# Or in ssh_config
Host *
    AddKeysToAgent yes
    UseKeychain yes        # macOS
```

```bash
# GOTCHA: X11 forwarding when you didn't need it
# Broken — slow connect, useless DISPLAY var, security risk
ssh -X user@host        # always-on X forwarding in your shell rc
# Fixed
unalias ssh             # remove the alias
ssh user@host           # no -X
```

```bash
# GOTCHA: ControlPath path too long (UNIX_PATH_MAX = 108 chars)
# Broken
ControlPath ~/Library/Application Support/MyApp/.ssh/cm-%r@%h:%p
# unix_listener: path "/Users/alice/Library/Application Support/MyApp/.ssh/cm-..." too long for Unix domain socket
# Fixed
ControlPath ~/.ssh/cm-%C       # %C = 16-char hash; safe everywhere
```

```bash
# GOTCHA: known_hosts entry gone after server rebuild → "Host key verification failed"
# Broken — assume MITM, panic
# Fixed — verify out-of-band, then ssh-keygen -R + reconnect (see "known_hosts" section)
```

```bash
# GOTCHA: scp -p (lowercase) is preserve-perms, scp -P (uppercase) is port
# Broken
scp -p 2222 file user@host:    # lowercase p = preserve, "2222" treated as a file!
# Fixed
scp -P 2222 file user@host:    # uppercase P = port
```

```bash
# GOTCHA: Forgetting ExitOnForwardFailure → silent broken tunnel
# Broken — port already in use, ssh stays connected, tunnel doesn't work, no error
ssh -L 5432:db:5432 bastion
# bind: Address already in use
# (but ssh keeps running, app fails confusingly)
# Fixed
ssh -o ExitOnForwardFailure=yes -L 5432:db:5432 bastion
```

```bash
# GOTCHA: rsync trailing slash semantics
# Broken — copies "src" directory under /dest/
rsync -av src/   user@host:/dest      # "/dest/src/" results
rsync -av src    user@host:/dest      # ALSO "/dest/src/"
# Fixed — depends on intent
rsync -av src/   user@host:/dest/     # contents of src → contents of /dest/
rsync -av src    user@host:/dest/     # src → /dest/src/
```

## Performance Tips

```bash
# Multiplexing: 100x faster on repeated connects (no new TCP+TLS+auth)
Host *
    ControlMaster auto
    ControlPath ~/.ssh/cm-%C
    ControlPersist 30m

# Skip IPv6 lookup delay if your network is IPv4-only
Host *
    AddressFamily inet

# Compression — DISABLE on fast networks (>= 1 Gbps); CPU becomes the bottleneck
Host *
    Compression no

# Skip reverse DNS on the server side (sshd UseDNS no)
# Slow login on a misconfigured network often = sshd waiting for reverse PTR

# ServerAliveInterval prefers SSH-level keepalive over TCP keepalive
# (TCP keepalive default ≈ 2 hours; SSH keepalive is per-second under your control)
ServerAliveInterval 30
ServerAliveCountMax 3

# Bigger ciphers aren't always slower — chacha20-poly1305 outpaces aes-cbc on CPUs without AES-NI
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com

# Larger TCP windows for transcontinental rsync
rsync -avzP -e 'ssh -o ServerAliveInterval=30' \
      --bwlimit=0 \
      ./src/ user@host:/dest/

# Disable PTY for tunnel-only sessions (saves a context switch per byte)
ssh -T -N -L 5432:db:5432 bastion
```

## Idioms

```bash
# IDIOM 1: Canonical ~/.ssh/config
Host *
    AddKeysToAgent yes
    HashKnownHosts yes
    StrictHostKeyChecking accept-new
    UpdateHostKeys yes
    ServerAliveInterval 60
    ControlMaster auto
    ControlPath ~/.ssh/cm-%C
    ControlPersist 30m

# IDIOM 2: Bastion + ProxyJump for everything inside a private VPC
Host bastion
    HostName bastion.example.com
    User jumpbox
    IdentityFile ~/.ssh/id_work
    IdentitiesOnly yes

Host *.internal.example.com
    ProxyJump bastion
    User deploy
    IdentityFile ~/.ssh/id_work
    IdentitiesOnly yes

# IDIOM 3: GitHub deploy keys (per-repo SSH key)
Host github-myrepo
    HostName github.com
    User git
    IdentityFile ~/.ssh/myrepo_deploy_key
    IdentitiesOnly yes

# Then in the local clone:
git clone git@github-myrepo:org/myrepo.git
git remote set-url origin git@github-myrepo:org/myrepo.git

# IDIOM 4: Password manager as agent (1Password)
Host *
    IdentityAgent "~/Library/Group Containers/2BUA8C4S2C.com.1password/t/agent.sock"
    AddKeysToAgent yes

# IDIOM 5: FIDO2 hardware key (touch-required everywhere)
ssh-keygen -t ed25519-sk -O resident -O verify-required -C "alice@yubikey-1"
# Public key goes in authorized_keys; the magic happens in the token.

# IDIOM 6: Sign git commits with SSH keys (since git 2.34)
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true

# Trust the signer locally
echo "alice@example.com namespaces=\"git\" $(cat ~/.ssh/id_ed25519.pub)" \
    > ~/.config/git/allowed_signers
git config --global gpg.ssh.allowedSignersFile ~/.config/git/allowed_signers

# Verify
git log --show-signature

# IDIOM 7: SSH-based file backup (rsync over ssh + cron)
0 3 * * * rsync -aqz --delete \
    -e 'ssh -i /home/backup/.ssh/id_ed25519 -o BatchMode=yes' \
    /srv/data/ backup@vps.example.com:/backup/$(hostname)/

# IDIOM 8: tmux + ssh — survive disconnects
ssh prod
tmux new -s work
# work happens here
# C-b d   detach
# disconnect, reconnect, tmux attach -t work

# IDIOM 9: ssh+autossh+systemd phone-home tunnel (see "Reverse Tunnels" up-top)

# IDIOM 10: Per-org config split via Include
# ~/.ssh/config
Include ~/.ssh/config.d/*
# ~/.ssh/config.d/work
# ~/.ssh/config.d/personal
# ~/.ssh/config.d/clients
```

## In-Session Escape Sequences

```text
[Press Enter, then ~]
~.        Disconnect (kill hung session even when terminal is dead)
~^Z       Suspend ssh (return to local shell; fg to resume)
~#        List forwarded connections
~&        Background ssh (waits for forwards to close)
~?        List all escape sequences
~~        Send a literal ~
~B        Send a BREAK
~C        Open command line — add/remove forwards on the fly
~R        Request rekey (OpenSSH 5.6+)
~V        Decrease verbosity
~v        Increase verbosity

# Add a forward in mid-session
[Enter] ~ C
ssh> -L 8080:localhost:8080
Forwarding port.

ssh> -KL 8080
Cancelled forwarding.

ssh> -R 9090:localhost:9090
Forwarding port.

ssh> ?
help message
```

The escape character defaults to `~`. Inside scripts (`ssh host < script`), set `EscapeChar none` or use `-e none` to avoid accidentally triggering on a `~` at the start of a line.

## File Permissions Cheat Sheet

```bash
# Client side — tight permissions or sshd refuses to use the keys
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519              # private key
chmod 600 ~/.ssh/id_ed25519_work
chmod 644 ~/.ssh/id_ed25519.pub          # public keys can be world-readable
chmod 600 ~/.ssh/authorized_keys
chmod 644 ~/.ssh/known_hosts
chmod 600 ~/.ssh/config
chmod go-w ~                             # $HOME must NOT be group/world-writable

# Quick all-in-one fix
find ~/.ssh -type d -exec chmod 700 {} \;
find ~/.ssh -type f -exec chmod 600 {} \;
chmod 644 ~/.ssh/*.pub ~/.ssh/known_hosts ~/.ssh/known_hosts.old 2>/dev/null

# SELinux file contexts (RHEL/Fedora)
restorecon -Rv ~/.ssh

# Server side — system files
sudo chmod 755 /etc/ssh
sudo chmod 600 /etc/ssh/ssh_host_*_key
sudo chmod 644 /etc/ssh/ssh_host_*_key.pub
sudo chmod 644 /etc/ssh/sshd_config
sudo chmod -R go-w /etc/ssh
```

## Two-Factor / TOTP

Pair public-key with a one-time code (Google Authenticator, Authy, hardware OTP).

```bash
# Server side
sudo apt install libpam-google-authenticator   # Debian/Ubuntu
sudo dnf install google-authenticator          # Fedora/RHEL EPEL

# As the user, generate the secret:
google-authenticator        # answers: time-based yes; update file yes; rate-limit yes
                            # writes ~/.google_authenticator (mode 0400)
                            # shows QR code → scan with Authy/Authenticator app
                            # records 5 emergency scratch codes — STORE THEM

# /etc/pam.d/sshd — at top
auth required pam_google_authenticator.so

# /etc/ssh/sshd_config (or drop-in)
KbdInteractiveAuthentication yes
AuthenticationMethods publickey,keyboard-interactive
UsePAM yes

sudo sshd -t && sudo systemctl reload sshd

# Client side — nothing changes; ssh prompts for the TOTP code after key auth
ssh user@host
# Authenticated using "publickey".
# Verification code:
```

## Recovery / Lockout

```bash
# Locked out — must keep one root/sudo session open while changing sshd
# If new config breaks login:
sudo sshd -t                                  # validate
sudo systemctl reload sshd                    # apply
# (broken!) — from the still-open session:
sudo cp /etc/ssh/sshd_config.bak /etc/ssh/sshd_config
sudo systemctl reload sshd

# Cloud / IaaS rescue (no console)
# AWS:    EC2 Instance Connect, AWS Systems Manager Session Manager
# GCP:    gcloud compute ssh --tunnel-through-iap, OS Login
# Azure:  az vm run-command, Bastion service
# DO:     Recovery console + reset root password via web

# Boot to single-user / rescue mode for last-resort fix
# Edit /etc/ssh/sshd_config, restart sshd from rescue init.

# Reset root via cloud-init
echo 'ssh_pwauth: True' | sudo tee /etc/cloud/cloud.cfg.d/99_pwauth.cfg
sudo cloud-init clean && sudo reboot
```

## Audit and Monitoring

```bash
# Show recent SSH auth events
sudo last | head -20                          # successful logins
sudo lastb | head -20                         # failed logins (Debian: lastb; needs btmp)

# journalctl
sudo journalctl -u sshd --since "1 hour ago"
sudo journalctl -u sshd | grep -E '(Accepted|Failed|Invalid|preauth)'

# Audit which keys logged in (LogLevel VERBOSE writes fingerprint)
sudo grep "Accepted publickey" /var/log/auth.log
# Apr 25 12:34:56 host sshd[12345]: Accepted publickey for alice from 10.0.5.5 port 49102 ssh2: ED25519 SHA256:abc...

# List currently active SSH sessions
who -u
w
ss -tnp 'sport = :22'

# Audit authorized_keys for stale entries
ssh-keygen -lf ~/.ssh/authorized_keys
# 256 SHA256:abc... alice@laptop-2026 (ED25519)
# 4096 SHA256:def... alice@old-mac-2018 (RSA)   # stale!

# Audit logged-in users' ~/.ssh/authorized_keys org-wide (Ansible)
ansible all -i hosts -m shell -a 'getent passwd | awk -F: "\$3>=1000 {print \$6}" | xargs -I{} cat {}/.ssh/authorized_keys 2>/dev/null'

# fail2ban — current bans
sudo fail2ban-client status sshd
```

## Tips

- **Ed25519 everywhere** — small keys, fast, mathematically clean. Use RSA only when forced.
- **`IdentitiesOnly yes`** plus a specific `IdentityFile` per host eliminates the "Too many authentication failures" trap once you have more than 5 keys in your agent.
- **`ServerAliveInterval 60`** is the most reliable defence against NAT/idle disconnects (Wi-Fi roaming, corporate VPN drops).
- **`ProxyJump`** has replaced `ProxyCommand` for ssh-over-ssh bastions. Use ProxyCommand only for non-ssh transports (HTTPS-CONNECT, IAP, AWS SSM).
- **Never disable `StrictHostKeyChecking`** in production. Use `accept-new` for first connections; the WARNING on key change should always pause you.
- **`ssh -vvv`** + `grep "Authentications that can continue"` + `grep "Offering public key"` solves 80% of auth failures in 30 seconds.
- **`ssh-copy-id`** fails silently if the server has `PasswordAuthentication no` — fall back to manual `cat | ssh ... cat >>`.
- **SELinux**: after editing `~/.ssh/authorized_keys`, run `restorecon -Rv ~/.ssh` if logins suddenly fail.
- **`ControlPath ~/.ssh/cm-%C`** — `%C` = 16-char hash; safer than `%r@%h:%p` against UNIX_PATH_MAX (108).
- **macOS**: `ssh-add --apple-use-keychain` (formerly `-K`) stores the passphrase in Keychain so you only enter it once.
- **OpenSSH 9.0+ scp uses SFTP under the hood**. Add `-O` to force the legacy protocol when talking to ancient servers.
- **GitHub on `:443`**: `ssh -p 443 git@ssh.github.com` for restrictive networks.
- **Sign git commits with SSH keys** (git 2.34+, `gpg.format=ssh`) — no GPG infrastructure required.
- **FIDO2 keys with `verify-required`** force a touch + PIN for every signing — the safest dev-laptop posture available in 2026.
- **CA-signed user keys + `TrustedUserCAKeys`** scale better than authorized_keys for fleets > 50 hosts.
- **Always test sshd config with `sudo sshd -t`** before reload, and KEEP an existing session open while changing sshd config.

## See Also

- bash, zsh, openssl, gpg, vault, polyglot

## References

- [openssh.com — Manual Pages](https://www.openssh.com/manual.html)
- [openssh.com — Release Notes](https://www.openssh.com/releasenotes.html)
- [ssh(1) — client](https://man.openbsd.org/ssh)
- [sshd(8) — server daemon](https://man.openbsd.org/sshd)
- [ssh_config(5) — client config](https://man.openbsd.org/ssh_config)
- [sshd_config(5) — server config](https://man.openbsd.org/sshd_config)
- [ssh-keygen(1) — key utility](https://man.openbsd.org/ssh-keygen)
- [ssh-agent(1) — auth agent](https://man.openbsd.org/ssh-agent)
- [ssh-add(1) — agent key tool](https://man.openbsd.org/ssh-add)
- [scp(1)](https://man.openbsd.org/scp) — `-O` flag, SFTP-backed implementation
- [sftp(1)](https://man.openbsd.org/sftp) and [sftp-server(8)](https://man.openbsd.org/sftp-server)
- [ssh-keyscan(1)](https://man.openbsd.org/ssh-keyscan)
- [Mozilla — OpenSSH Security Guidelines](https://infosec.mozilla.org/guidelines/openssh)
- [RFC 4251 — SSH Protocol Architecture](https://www.rfc-editor.org/rfc/rfc4251)
- [RFC 4252 — SSH Authentication Protocol](https://www.rfc-editor.org/rfc/rfc4252)
- [RFC 4253 — SSH Transport Layer Protocol](https://www.rfc-editor.org/rfc/rfc4253)
- [RFC 4254 — SSH Connection Protocol](https://www.rfc-editor.org/rfc/rfc4254)
- [RFC 4255 — SSHFP DNS Resource Record](https://www.rfc-editor.org/rfc/rfc4255)
- [RFC 4256 — Generic Message Exchange (KbdInteractive)](https://www.rfc-editor.org/rfc/rfc4256)
- [RFC 4716 — SSH Public Key File Format](https://www.rfc-editor.org/rfc/rfc4716)
- [RFC 8332 — RSA SHA-256/SHA-512 in SSH](https://www.rfc-editor.org/rfc/rfc8332)
- [RFC 8709 — Ed25519 / Ed448 in SSH](https://www.rfc-editor.org/rfc/rfc8709)
- [PROTOCOL.certkeys — OpenSSH SSH certificate format](https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.certkeys)
- [PROTOCOL.krl — OpenSSH Key Revocation Lists](https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.krl)
- [Arch Wiki — OpenSSH](https://wiki.archlinux.org/title/OpenSSH)
- [Red Hat — Configuring Secure Communication with SSH](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/securing_networks/assembly_using-secure-communications-between-two-systems-with-openssh_securing-networks)
- [Debian Wiki — SSH](https://wiki.debian.org/SSH)
- [ssh.com — Secure Shell Documentation](https://www.ssh.com/academy/ssh)
- [smallstep — SSH Best Practices](https://smallstep.com/blog/ssh-best-practices/)
- [fail2ban — sshd jail](https://github.com/fail2ban/fail2ban/blob/master/config/jail.conf)
- [Yubico — SSH FIDO2 Setup](https://developers.yubico.com/SSH/Securing_SSH_with_FIDO2.html)
