# SSH Errors

Verbatim OpenSSH client and server error messages with cause, diagnostic ladder, and fix — never leave the terminal to debug `Permission denied (publickey)`, host-key changes, algorithm mismatches, agent failures, or port-forwarding refusals again.

## Setup

OpenSSH ships as both client (`ssh`, `scp`, `sftp`, `ssh-keygen`, `ssh-add`, `ssh-agent`, `ssh-keyscan`) and server (`sshd`). Default versions per platform:

```text
Ubuntu 20.04 LTS    OpenSSH 8.2p1
Ubuntu 22.04 LTS    OpenSSH 8.9p1
Ubuntu 24.04 LTS    OpenSSH 9.6p1
Debian 11           OpenSSH 8.4p1
Debian 12           OpenSSH 9.2p1
RHEL 8 / Rocky 8    OpenSSH 8.0p1
RHEL 9 / Rocky 9    OpenSSH 8.7p1
Alpine 3.19         OpenSSH 9.6p1
Amazon Linux 2      OpenSSH 7.4p1
Amazon Linux 2023   OpenSSH 8.7p1
macOS 13 (Ventura)  OpenSSH 9.0p1
macOS 14 (Sonoma)   OpenSSH 9.4p1
macOS 15 (Sequoia)  OpenSSH 9.7p1
FreeBSD 14          OpenSSH 9.5p1
Windows 10/11       OpenSSH 8.6p1 (built-in) / 9.5p1 (winget)
```

Watershed releases:

```text
OpenSSH 7.0  (2015)  — DSA disabled by default
OpenSSH 8.2  (2020)  — FIDO/U2F support (sk- key types)
OpenSSH 8.5  (2021)  — PerSourcePenalties, %k token in known_hosts
OpenSSH 8.7  (2021)  — Default scp protocol still SCP, sftp recommended
OpenSSH 8.8  (2021)  — RSA-SHA1 (ssh-rsa) DISABLED by default — biggest breaker
OpenSSH 9.0  (2022)  — scp(1) defaults to SFTP protocol
OpenSSH 9.5  (2023)  — ML-KEM hybrid key exchange (post-quantum)
OpenSSH 9.6  (2023)  — Terrapin attack mitigation (CVE-2023-48795)
OpenSSH 9.8  (2024)  — regreSSHion fix (CVE-2024-6387)
```

Client commands:

```bash
ssh user@host                 # default port 22
ssh -p 2222 user@host
ssh -i ~/.ssh/id_ed25519 user@host
ssh -v user@host              # verbose; shows config and auth attempts
ssh -vv user@host             # more verbose
ssh -vvv user@host            # most verbose; protocol-level detail
ssh -G user@host              # print effective config; never connects
ssh -F /path/to/config user@host  # use a different config file
ssh -F /dev/null user@host    # ignore all config files (clean test)
ssh -o Option=Value user@host # one-shot config override
ssh -N user@host              # don't run a remote command (port-forward only)
ssh -f user@host              # background after auth
ssh -A user@host              # forward agent (use sparingly)
ssh -L 8080:remote:80 user@host  # local forward
ssh -R 9090:localhost:80 user@host  # remote forward
ssh -D 1080 user@host         # SOCKS dynamic forward
ssh -J jump@host target@host  # ProxyJump
ssh -W host:port jump@host    # forward stdio over connection
```

Key generation:

```bash
ssh-keygen -t ed25519 -C "stevie@bellis.tech"   # modern, recommended
ssh-keygen -t ed25519-sk -C "stevie@bellis.tech"  # FIDO2/Yubikey
ssh-keygen -t rsa -b 4096 -C "stevie@bellis.tech"  # RSA 4096; legacy
ssh-keygen -t ecdsa -b 521 -C "stevie@bellis.tech"  # ECDSA NIST P-521

ssh-keygen -p -f ~/.ssh/id_ed25519              # change passphrase
ssh-keygen -p -m PEM -f keyfile                 # convert to old PEM (RSA)
ssh-keygen -p -m PKCS8 -f keyfile               # convert to PKCS#8
ssh-keygen -p -m RFC4716 -f keyfile             # convert to RFC4716

ssh-keygen -y -f ~/.ssh/id_ed25519              # derive .pub from private
ssh-keygen -lf ~/.ssh/id_ed25519.pub            # print fingerprint
ssh-keygen -lvf ~/.ssh/id_ed25519.pub           # ASCII-art fingerprint
ssh-keygen -F host                              # search known_hosts
ssh-keygen -R host                              # remove host from known_hosts
ssh-keygen -H -f ~/.ssh/known_hosts             # hash known_hosts entries

ssh-keyscan -t ed25519,ecdsa,rsa host > known_hosts.new
ssh-keyscan -p 2222 host >> ~/.ssh/known_hosts
```

Agent:

```bash
eval "$(ssh-agent -s)"          # start agent in shell
ssh-add ~/.ssh/id_ed25519       # add key
ssh-add -l                      # list fingerprints in agent
ssh-add -L                      # list public keys in agent
ssh-add -D                      # delete all identities
ssh-add -d ~/.ssh/id_ed25519    # delete one key
ssh-add -t 3600                 # add with 1-hour lifetime
ssh-add --apple-use-keychain ~/.ssh/id_ed25519  # macOS Keychain
ssh-add --apple-load-keychain   # macOS, restore from Keychain
```

Server commands:

```bash
sudo systemctl status sshd
sudo systemctl restart sshd
sudo sshd -T                    # print effective sshd_config
sudo sshd -T -C user=alice,host=app01,addr=10.0.0.5
sudo sshd -t                    # test config syntax
sudo sshd -ddd -p 2222          # foreground debug on alt port
journalctl -u sshd -f           # systemd logs
sudo tail -f /var/log/auth.log  # Debian/Ubuntu
sudo tail -f /var/log/secure    # RHEL/CentOS
```

Reading `ssh -vvv user@host`:

```text
debug1: Reading configuration data /home/$USER/.ssh/config
debug1: Reading configuration data /etc/ssh/ssh_config
debug2: resolving "host.example.com" port 22
debug3: resolve_host: lookup host.example.com:22
debug1: Connecting to host.example.com [203.0.113.5] port 22.
debug1: Connection established.
debug1: identity file /home/$USER/.ssh/id_rsa type 0
debug1: identity file /home/$USER/.ssh/id_ed25519 type 3
debug1: Local version string SSH-2.0-OpenSSH_9.6
debug1: Remote protocol version 2.0, remote software version OpenSSH_8.9p1
debug1: kex: algorithm: curve25519-sha256
debug1: kex: host key algorithm: ssh-ed25519
debug1: Server host key: ssh-ed25519 SHA256:abc...
debug1: Host 'host.example.com' is known and matches the ED25519 host key.
debug1: rekey out after 134217728 blocks
debug1: SSH2_MSG_NEWKEYS sent / received
debug1: Authentications that can continue: publickey,password
debug1: Next authentication method: publickey
debug1: Offering public key: ED25519 SHA256:xyz... id_ed25519
debug1: Server accepts key: ED25519 SHA256:xyz...
debug1: Authentication succeeded (publickey).
debug1: channel 0: new [client-session]
```

Each `debug1:` line corresponds to a state the protocol just entered. When debugging, scan for: which keys offered, which accepted, which auth method server requested, which user@host the client resolved.

## Permission denied (publickey,...)

Verbatim messages:

```text
Permission denied (publickey).
Permission denied (publickey,password).
Permission denied (publickey,gssapi-keyex,gssapi-with-mic,password).
Permission denied (publickey,keyboard-interactive).
Permission denied, please try again.
```

The string in parentheses is the **server's offered authentication methods** at the time the client gave up — read it carefully. `publickey` only means the server doesn't accept passwords; `publickey,password` means both are offered and both failed.

### Diagnostic Ladder

#### Step 1 — Run `ssh -v` and read which keys are offered

```bash
ssh -v user@host 2>&1 | grep -E 'Offering|Server accepts|identity file'
```

Look for these lines:

```text
debug1: identity file /home/me/.ssh/id_rsa type 0
debug1: identity file /home/me/.ssh/id_ed25519 type -1   <-- type -1 means file missing
debug1: Offering public key: ED25519 SHA256:... /home/me/.ssh/id_ed25519
debug1: Authentications that can continue: publickey
debug1: No more authentication methods to try.
```

If you see `Offering public key:` but never `Server accepts key:`, the key is being rejected by the server — move to step 5.

If you see no `Offering public key:` at all, the client isn't even trying — move to step 2.

#### Step 2 — Verify client-side key permissions

```bash
ls -la ~/.ssh/
# expected:
# drwx------   me me  .            (0700)
# -rw-------   me me  id_ed25519   (0600)
# -rw-r--r--   me me  id_ed25519.pub (0644)
# -rw-------   me me  config       (0600)
# -rw-r--r--   me me  known_hosts  (0644)

stat -c '%a %U %n' ~/.ssh ~/.ssh/*
```

The notorious "permissions too open" client-side error:

```text
Permissions 0644 for '/home/me/.ssh/id_ed25519' are too open.
It is required that your private key files are NOT accessible by others.
This private key will be ignored.
Load key "/home/me/.ssh/id_ed25519": bad permissions
```

Fix:

```bash
chmod 700 ~/.ssh
chmod 600 ~/.ssh/id_ed25519
chmod 644 ~/.ssh/id_ed25519.pub
chmod 600 ~/.ssh/config
chmod 644 ~/.ssh/known_hosts
chmod 600 ~/.ssh/authorized_keys   # if relevant on this host
```

#### Step 3 — Verify `~/.ssh` directory permissions

```bash
stat -c '%a %n' ~/.ssh
# 700 /home/me/.ssh  -- correct
chmod 700 ~/.ssh
```

If `~/.ssh` is `0755` ssh will still read it client-side, but server-side sshd will reject it (see step 5).

#### Step 4 — Verify `~/.ssh/config` permissions

```bash
stat -c '%a %n' ~/.ssh/config
chmod 600 ~/.ssh/config
```

A world-readable config triggers:

```text
Bad owner or permissions on /home/me/.ssh/config
```

#### Step 5 — Verify server-side `~/.ssh/authorized_keys`

This is the single most common cause. SSH on the server **silently refuses** to read `authorized_keys` if any of the following are true:

- `$HOME` is group- or world-writable
- `~/.ssh` is anything except `0700` (or `0750` if owned by user)
- `authorized_keys` is anything except `0600` (or `0644` with strict modes off — don't rely on it)
- Any of those files are not owned by the connecting user

On the server:

```bash
sudo -u $TARGET_USER stat -c '%a %U %n' \
  /home/$TARGET_USER \
  /home/$TARGET_USER/.ssh \
  /home/$TARGET_USER/.ssh/authorized_keys

# expected:
# 750 alice /home/alice            (or 755; not 770/775/777)
# 700 alice /home/alice/.ssh
# 600 alice /home/alice/.ssh/authorized_keys
```

Fix:

```bash
chmod go-w /home/alice
chmod 700 /home/alice/.ssh
chmod 600 /home/alice/.ssh/authorized_keys
chown -R alice:alice /home/alice/.ssh
```

#### Step 6 — Check `sshd_config` directives

```bash
sudo grep -E '^(PubkeyAuthentication|AuthorizedKeysFile|StrictModes)' /etc/ssh/sshd_config

# desired:
# PubkeyAuthentication yes
# AuthorizedKeysFile   .ssh/authorized_keys .ssh/authorized_keys2
# StrictModes          yes

sudo sshd -T | grep -iE 'pubkey|authorizedkeys|strictmodes'
```

`AuthorizedKeysFile` may be relative to `$HOME` or absolute. Tokens: `%h` (home), `%u` (username), `%U` (uid).

A common centralized override:

```text
AuthorizedKeysFile /etc/ssh/authorized_keys/%u
```

If the file lives there, the per-user `~/.ssh/authorized_keys` is ignored.

#### Step 7 — Server logs

```bash
sudo journalctl -u ssh -n 50
sudo journalctl -u sshd -n 50
sudo tail -50 /var/log/auth.log     # Debian/Ubuntu
sudo tail -50 /var/log/secure       # RHEL/CentOS
```

Decode messages — see "Server-Side sshd Errors" section.

For more detail set `LogLevel VERBOSE` (or `DEBUG`/`DEBUG2`/`DEBUG3`) in `sshd_config`, restart, retry, then revert.

#### Step 8 — Check SELinux on the server

```bash
ls -Z ~/.ssh/authorized_keys
# expected:
# unconfined_u:object_r:ssh_home_t:s0  authorized_keys
```

If you see anything other than `ssh_home_t`:

```bash
restorecon -Rv ~/.ssh/
# or
chcon -t ssh_home_t ~/.ssh/authorized_keys
```

A relocated home directory (`/data/home/$USER`) needs:

```bash
sudo semanage fcontext -a -t ssh_home_t '/data/home/[^/]+/\.ssh(/.*)?'
sudo restorecon -Rv /data/home/
```

Audit logs reveal SELinux denials:

```bash
sudo ausearch -m AVC -ts recent | grep ssh
sudo grep AVC /var/log/audit/audit.log | tail -20
```

#### Step 9 — `AllowUsers` / `DenyUsers` / `AllowGroups` / `DenyGroups`

```bash
sudo sshd -T | grep -iE '^(allow|deny)(users|groups)'
```

If `AllowUsers` is set and your user isn't listed, no key works:

```text
User alice from 203.0.113.5 not allowed because not listed in AllowUsers
```

Add the user:

```text
AllowUsers alice bob ops
```

Or use a group:

```text
AllowGroups ssh-users
```

```bash
sudo gpasswd -a alice ssh-users
```

#### Step 10 — `Match` blocks

```text
Match User ops
    AuthenticationMethods publickey,password
    PasswordAuthentication yes
Match Address 10.0.0.0/8
    PermitRootLogin yes
Match all
```

`sshd_config` is read top-down; `Match` blocks **override** earlier directives for matching connections. Use `sshd -T -C` to simulate:

```bash
sudo sshd -T -C user=alice,host=app01,addr=10.0.0.5
```

If `AuthenticationMethods publickey,password` is in effect for your user, password failure after pubkey success still yields `Permission denied`.

## Permission denied (password)

Verbatim:

```text
Permission denied, please try again.
user@host: Permission denied (password).
Authentication failed.
```

Causes and fixes:

```bash
# 1. Wrong password — confirm caps lock, layout, special chars
# 2. PAM denied
sudo grep auth /var/log/auth.log | tail
# look for: pam_unix(sshd:auth): authentication failure

# 3. Server has PasswordAuthentication off
sudo sshd -T | grep -iE 'passwordauthentication|kbdinteractive|challenge'
# expected (server allowing passwords):
# passwordauthentication yes
# kbdinteractiveauthentication yes

# 4. ChallengeResponseAuthentication mode (older sshd)
echo 'ChallengeResponseAuthentication yes' | sudo tee -a /etc/ssh/sshd_config
sudo systemctl restart sshd

# 5. Account locked
sudo passwd -S alice              # alice L  ...  L = locked
sudo passwd -u alice              # unlock
sudo usermod -e '' alice          # remove expiry

# 6. /etc/security/access.conf or pam_access denying
sudo grep -E 'sshd|pam_access' /etc/pam.d/sshd

# 7. nologin restriction
ls -l /etc/nologin                # remove if present and intended
```

`KbdInteractiveAuthentication` (renamed from `ChallengeResponseAuthentication` in 8.7) controls keyboard-interactive PAM-driven password prompts. If `PasswordAuthentication no` but `KbdInteractiveAuthentication yes`, you'll still get a password prompt via PAM.

## Host Key Verification Failed

Verbatim, the most alarming message in SSH:

```text
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that a host key has just been changed.
The fingerprint for the ED25519 key sent by the remote host is
SHA256:abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQR.
Please contact your system administrator.
Add correct host key in /home/me/.ssh/known_hosts to get rid of this message.
Offending ED25519 key in /home/me/.ssh/known_hosts:42
Host key for 203.0.113.5 has changed and you have requested strict checking.
Host key verification failed.
```

### Legitimate causes

- Server reinstalled — host keys regenerated
- IP address reused for a different VM
- DNS now resolves the same hostname to a different machine
- OpenSSH upgrade migrated key formats (rare)
- Cloud snapshot rolled back

### Bad cause

- Active man-in-the-middle attack — your traffic is going through an attacker who terminated TLS/SSH and re-presented its own host key

### Verifying authenticity before trusting

Out-of-band, on the server, fetch the host key fingerprint:

```bash
# On the server
sudo ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub
sudo ssh-keygen -lf /etc/ssh/ssh_host_ecdsa_key.pub
sudo ssh-keygen -lf /etc/ssh/ssh_host_rsa_key.pub

# Compare to the SHA256 fingerprint in the warning message
# Equal? safe to update known_hosts
# Different? STOP — possible MITM
```

Cloud providers expose host keys on first boot:

```bash
# AWS EC2
aws ec2 get-console-output --instance-id i-xxx --output text \
  | sed -n '/BEGIN SSH HOST KEY FINGERPRINTS/,/END SSH HOST KEY FINGERPRINTS/p'

# GCE
gcloud compute instances get-serial-port-output INSTANCE | grep -A 5 'ssh-rsa\|ssh-ed25519'

# Azure
az vm boot-diagnostics get-boot-log --name VM --resource-group RG | grep -A 5 'fingerprint'
```

### Fix (after verification)

```bash
# Remove old entry by hostname (works even with HashKnownHosts yes)
ssh-keygen -R 203.0.113.5
ssh-keygen -R host.example.com
ssh-keygen -R '[host.example.com]:2222'    # nondefault port

# Re-add by connecting (TOFU prompt)
ssh user@host

# Or pre-populate from ssh-keyscan (after verifying!)
ssh-keyscan -t ed25519 host.example.com >> ~/.ssh/known_hosts

# Specifically remove a numbered line
sed -i '42d' ~/.ssh/known_hosts            # be sure!
```

### HashKnownHosts gotcha

Default on most distros:

```text
HashKnownHosts yes
```

This stores hostnames as hashed values in `known_hosts` (so a stolen file doesn't leak hostnames). Side effect: you can't `grep` for them.

```bash
# Don't grep — use ssh-keygen
ssh-keygen -F host.example.com           # find
ssh-keygen -R host.example.com           # remove
ssh-keygen -F '[host.example.com]:2222'  # nondefault port

# Disable hashing for new entries
echo 'HashKnownHosts no' >> ~/.ssh/config
```

To re-hash an existing plaintext file:

```bash
ssh-keygen -H -f ~/.ssh/known_hosts      # creates known_hosts.old as backup
```

### `StrictHostKeyChecking` modes

```text
StrictHostKeyChecking yes         # refuse if host not in known_hosts (no TOFU)
StrictHostKeyChecking accept-new  # auto-add new hosts but reject changed (DEFAULT since 7.6)
StrictHostKeyChecking ask         # legacy default — prompt for new
StrictHostKeyChecking no          # accept everything; disable verification — DANGEROUS
StrictHostKeyChecking off         # alias for no
```

CLI:

```bash
ssh -o StrictHostKeyChecking=accept-new user@host
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null user@host  # ephemeral testing only
```

`UserKnownHostsFile=/dev/null` is the canonical "I don't care about identity, this is throwaway" mode — never use against a host you'd send credentials or data to.

## No Matching Host Key Type

Verbatim:

```text
Unable to negotiate with 203.0.113.5 port 22: no matching host key type found. Their offer: ssh-rsa
Unable to negotiate with 203.0.113.5 port 22: no matching host key type found. Their offer: ssh-rsa,ssh-dss
no hostkey alg
```

Cause: OpenSSH 8.8 (Sep 2021) **disabled `ssh-rsa` (RSA with SHA-1) by default**. The remote server still offers only the legacy SHA-1 RSA host key. The client refuses.

This is the most-encountered breaking change in OpenSSH history. It usually surfaces against:

- Old IOS/IOS-XE Cisco devices
- Old NetScreen/SRX/Junos
- Old SonicWall, F5, Cisco ASA
- Embedded routers, IoT devices
- Paramiko-based servers pinned to old algorithms
- GitHub Enterprise pre-2.22 (pre-2020)

### Quick fix (one connection)

```bash
ssh -oHostKeyAlgorithms=+ssh-rsa -oPubkeyAcceptedAlgorithms=+ssh-rsa user@host
```

### Per-host fix in `~/.ssh/config`

```text
Host legacy-router
    HostName 203.0.113.5
    HostKeyAlgorithms +ssh-rsa
    PubkeyAcceptedAlgorithms +ssh-rsa
    User admin
```

Note: starting in OpenSSH 8.5, `PubkeyAcceptedKeyTypes` was renamed to `PubkeyAcceptedAlgorithms`. Both names work for now; new configs should use the latter.

### Better fix — modernize the server

If you can touch the server:

```bash
# Generate ed25519 host key (instant, modern)
sudo ssh-keygen -t ed25519 -f /etc/ssh/ssh_host_ed25519_key -N ''
sudo systemctl restart sshd

# In sshd_config:
HostKey /etc/ssh/ssh_host_ed25519_key
HostKey /etc/ssh/ssh_host_rsa_key       # keep RSA-SHA2 (256/512) supported alongside
```

Modern RSA still works — only **RSA with SHA-1 signatures** was disabled. RSA keys with `rsa-sha2-256` and `rsa-sha2-512` signatures continue to be accepted.

### Diagnose what's offered

```bash
ssh -vvv user@host 2>&1 | grep -iE 'host key|hostkey|kex'

ssh -Q HostKeyAlgorithms              # algorithms supported by your client
ssh -Q PubkeyAcceptedAlgorithms

ssh-keyscan -t rsa,ecdsa,ed25519 host  # what server offers
```

## Too Many Authentication Failures

Verbatim:

```text
Received disconnect from 203.0.113.5 port 22:2: Too many authentication failures
Disconnected from 203.0.113.5 port 22
```

Cause: SSH offers each key in your agent / config one by one. Server enforces `MaxAuthTries` (default `6`). If you have 7+ keys loaded, server disconnects before reaching the right one.

```bash
ssh-add -l | wc -l        # how many keys in agent right now
```

### Fix — `IdentitiesOnly`

```bash
ssh -o IdentitiesOnly=yes -i ~/.ssh/id_ed25519 user@host
```

In `~/.ssh/config`:

```text
Host work
    HostName work.example.com
    User alice
    IdentityFile ~/.ssh/work_ed25519
    IdentitiesOnly yes
```

`IdentitiesOnly yes` means: **only** offer the keys explicitly listed via `IdentityFile`, ignoring anything in the agent.

### Fix — clear and re-add

```bash
ssh-add -D                              # remove all
ssh-add ~/.ssh/work_ed25519             # add only what you need
```

### Fix — server-side

```text
# /etc/ssh/sshd_config — only do this if you understand the brute-force tradeoff
MaxAuthTries 10
```

Default `6` is conservative; CIS benchmarks recommend `4`. Don't raise blindly.

## No Such File or Directory (Identity File)

Verbatim:

```text
Warning: Identity file /home/me/.ssh/missing not accessible: No such file or directory.
Could not open identity 'X': No such file or directory
```

Cause: `IdentityFile` path is wrong, file deleted, or `~` not expanded.

```bash
ls -la ~/.ssh/                                  # confirm what exists
ssh -G user@host | grep -i identityfile         # see effective path
```

### Tilde expansion gotcha

In `~/.ssh/config`, both work:

```text
IdentityFile ~/.ssh/id_ed25519
IdentityFile %d/.ssh/id_ed25519        # %d = home dir
```

But in `/etc/ssh/ssh_config` (system-wide), `~` may not expand for users running ssh from a context with no `$HOME`. Always prefer absolute paths or `%d`:

```text
# /etc/ssh/ssh_config
Match User *
    IdentityFile %d/.ssh/id_ed25519
```

Tokens (see `man ssh_config` → `TOKENS`):

```text
%%   literal %
%C   hash of %l%h%p%r
%d   user's home dir
%h   target host
%i   local user ID
%j   ProxyJump host
%k   home of user (for sshd)
%l   local host
%n   original target host (pre-CanonicalizeHostname)
%p   port
%r   remote user
%T   tunnel device (sshd)
%u   local user
```

## Connection Refused / Timeout / No Route / DNS

Verbatim:

```text
ssh: connect to host host.example.com port 22: Connection refused
ssh: connect to host host.example.com port 22: Connection timed out
ssh: connect to host host.example.com port 22: No route to host
ssh: connect to host host.example.com port 22: Network is unreachable
ssh: Could not resolve hostname host.example.com: Name or service not known
ssh: Could not resolve hostname host.example.com: Temporary failure in name resolution
ssh: Could not resolve hostname host.example.com: nodename nor servname provided, or not known
kex_exchange_identification: Connection closed by remote host
kex_exchange_identification: read: Connection reset by peer
kex_exchange_identification: client sent invalid protocol identifier "..."
Connection closed by 203.0.113.5 port 22
banner exchange: Connection to 203.0.113.5 port 22: invalid format
```

### Connection refused

Port 22 reachable but **nothing is listening**. Either sshd isn't running, or it's bound to a different port.

```bash
# Local check (on server)
sudo systemctl status sshd
sudo ss -tlnp | grep ssh
# tcp LISTEN 0 128 0.0.0.0:22 0.0.0.0:* users:(("sshd",pid=1234,fd=3))

# From client
nc -zv host.example.com 22
nmap -p 22 host.example.com

# Different port?
ssh -p 2222 user@host
```

Fix:

```bash
sudo systemctl start sshd
sudo systemctl enable sshd
```

If sshd silently refuses to start, run with debug:

```bash
sudo /usr/sbin/sshd -ddd -p 2222    # foreground
```

### Connection timed out

Network/firewall problem — packet never arrives or response is dropped.

```bash
ping -c 3 host.example.com           # ICMP reachability
mtr -n host.example.com              # path
traceroute host.example.com
nmap -Pn -p 22 host.example.com      # scan even if ICMP blocked
nc -zv -w 5 host.example.com 22

# On server, check firewall
sudo iptables -L INPUT -n -v --line-numbers
sudo nft list ruleset
sudo firewall-cmd --list-all
sudo ufw status verbose

# Cloud security groups
aws ec2 describe-security-groups --group-ids sg-xxx
gcloud compute firewall-rules list
az network nsg rule list -g RG --nsg-name NSG
```

### No route to host

Routing issue — kernel has no route to that destination.

```bash
ip route get 203.0.113.5
ip route show
ip neigh show                        # ARP for L2
```

Possible: VPN dropped, default gateway changed, network unplugged.

### Name or service not known (DNS)

```bash
host host.example.com
dig host.example.com
nslookup host.example.com
getent hosts host.example.com        # respects /etc/nsswitch.conf
cat /etc/resolv.conf
ping 8.8.8.8                         # confirm internet at all
```

Fix temporarily:

```bash
# /etc/hosts
203.0.113.5  host.example.com
```

### `kex_exchange_identification: Connection closed by remote host`

The TCP connection completed but the server hung up before exchanging identification banners. Common causes:

- **fail2ban / sshguard** — your IP got rate-limited
- **TCP wrappers** — `/etc/hosts.deny` denying you
- **PerSourcePenalties** (OpenSSH 9.8+) — temporary IP penalty after repeated bad attempts
- **sshd crashed during startup** — check `journalctl`
- **sshd Match block dropping connection** before banner

```bash
# On server, check fail2ban
sudo fail2ban-client status sshd
sudo fail2ban-client unban 203.0.113.5

# Check TCP wrappers
cat /etc/hosts.deny
cat /etc/hosts.allow

# Check OpenSSH 9.8+ penalties
sudo journalctl -u ssh | grep -i penalty
```

### `kex_exchange_identification: client sent invalid protocol identifier`

Something other than an SSH client connected — TLS scanner, HTTP probe, port forwarder, malformed banner:

```bash
sudo journalctl -u sshd | grep -i invalid
```

Often harmless attack noise; log and ignore.

## Too Many Open Connections

Verbatim:

```text
channel 1: open failed: administratively prohibited: open failed
channel 2: open failed: connect failed: open failed
ssh_exchange_identification: read: Connection reset by peer
ssh_exchange_identification: Connection closed by remote host
```

### `MaxSessions`

Per established SSH connection, the server limits multiplexed sessions (default `10`):

```text
# /etc/ssh/sshd_config
MaxSessions 20
```

Restart sshd. Affects:

- Multiplexed `ControlMaster` connections sharing a single TCP socket
- `ssh -t -t` opening many sub-shells
- Multiple `scp` calls reusing a control socket

### `MaxStartups`

Default `10:30:100`:

```text
MaxStartups start:rate:full
```

Meaning: start dropping incoming-but-unauthenticated connections at 10 concurrent; randomly drop at `rate%` linear from `start` to `full`; 100% drop above 100 concurrent.

```text
# Loosen for high-traffic bastion
MaxStartups 100:30:200
```

### `LoginGraceTime`

Default `120` seconds. Time to authenticate before sshd disconnects. If you're slow at typing 2FA codes:

```text
LoginGraceTime 5m
```

## Cipher / Algorithm Errors

Verbatim:

```text
Unable to negotiate with 203.0.113.5 port 22: no matching cipher found.
  Their offer: 3des-cbc,aes128-cbc,blowfish-cbc,cast128-cbc,arcfour,arcfour128,arcfour256,aes192-cbc,aes256-cbc,rijndael-cbc@lysator.liu.se
Unable to negotiate with 203.0.113.5 port 22: no matching MAC found.
  Their offer: hmac-sha1,hmac-sha1-96,hmac-md5,hmac-md5-96
Unable to negotiate with 203.0.113.5 port 22: no matching key exchange method found.
  Their offer: diffie-hellman-group1-sha1
Unable to negotiate with 203.0.113.5 port 22: no matching host key type found.
  Their offer: ssh-rsa,ssh-dss
```

OpenSSH 7.0+ removed many weak algorithms. OpenSSH 8.8+ disabled `ssh-rsa` (SHA-1). OpenSSH 9.0+ removed `chacha20-poly1305@openssh.com` from defaults briefly (then restored).

### Quick re-enable per connection

```bash
ssh -oKexAlgorithms=+diffie-hellman-group1-sha1 \
    -oCiphers=+aes256-cbc \
    -oMACs=+hmac-sha1 \
    -oHostKeyAlgorithms=+ssh-rsa \
    -oPubkeyAcceptedAlgorithms=+ssh-rsa \
    user@legacy-host
```

### Per-host config

```text
Host legacy-*
    KexAlgorithms +diffie-hellman-group1-sha1,diffie-hellman-group14-sha1
    Ciphers +aes256-cbc,aes128-cbc,3des-cbc
    MACs +hmac-sha1
    HostKeyAlgorithms +ssh-rsa,ssh-dss
    PubkeyAcceptedAlgorithms +ssh-rsa
```

`+algo` = append to defaults. `algo` (no prefix) = replace. `-algo` = remove. `^algo` = prepend.

### Inspect supported algorithms

```bash
ssh -Q kex                # supported KEX
ssh -Q cipher
ssh -Q mac
ssh -Q HostKeyAlgorithms
ssh -Q PubkeyAcceptedAlgorithms
ssh -Q sig                # signature algorithms
ssh -Q help               # all queryable types

# What server offers
ssh -vvv host 2>&1 | grep -iE 'kex|mac|cipher|hostkey'
nmap --script ssh2-enum-algos -p 22 host.example.com
```

### Better fix — modernize

If the legacy side is a Cisco/Juniper:

```text
# Cisco IOS-XE 16.x+
ip ssh version 2
ip ssh server algorithm encryption aes128-ctr aes192-ctr aes256-ctr
ip ssh server algorithm mac hmac-sha2-256 hmac-sha2-512
ip ssh server algorithm kex diffie-hellman-group14-sha256 ecdh-sha2-nistp256
ip ssh server algorithm hostkey ssh-rsa rsa-sha2-256 rsa-sha2-512

# Juniper Junos 19.4+
set system services ssh ciphers aes256-ctr
set system services ssh macs hmac-sha2-512
set system services ssh key-exchange ecdh-sha2-nistp521
set system services ssh hostkey-algorithm ssh-rsa
```

## Agent-Related Errors

Verbatim:

```text
Could not open a connection to your authentication agent.
Error connecting to agent: No such file or directory
SSH_AUTH_SOCK: command not found
agent admitted failure to sign using the key
sign_and_send_pubkey: signing failed: agent refused operation
The agent has no identities.
Identity added: /home/me/.ssh/id_ed25519 (stevie@bellis.tech)
ssh-add: communication with agent failed
ssh_askpass: exec(/usr/lib/ssh/ssh-askpass): No such file or directory
```

### Could not open a connection to your authentication agent

`ssh-agent` not running, or `SSH_AUTH_SOCK` not set in this shell.

```bash
# Start agent, set env vars
eval "$(ssh-agent -s)"
ssh-add -l                   # confirm
ssh-add ~/.ssh/id_ed25519
```

Persistent across shells (Linux):

```bash
# ~/.bashrc or ~/.zshrc
if [ -z "$SSH_AUTH_SOCK" ]; then
    eval "$(ssh-agent -s)" >/dev/null
    trap 'ssh-agent -k >/dev/null 2>&1' EXIT
fi
```

Better: use systemd user unit:

```bash
systemctl --user enable --now ssh-agent
# /etc/environment or ~/.config/environment.d/ssh-agent.conf:
# SSH_AUTH_SOCK=$XDG_RUNTIME_DIR/ssh-agent.socket
```

Or KDE/GNOME keyring (already running on most desktops):

```bash
echo $SSH_AUTH_SOCK     # /run/user/1000/keyring/ssh
```

### `agent admitted failure to sign using the key`

Three common causes:

```text
1. Yubikey detached or USB issue (sk-ed25519 keys)
2. gpg-agent in SSH-emulation mode lost the card
3. ed25519-sk key without --apple-use-keychain on macOS, agent forgot
```

```bash
# Confirm what the agent thinks it has
ssh-add -L

# For gpg-agent
gpg --card-status
gpg-connect-agent updatestartuptty /bye
ssh-add -l

# For Yubikey
lsusb | grep -i yubico
ykman list
gpg --card-status
```

Fix:

```bash
ssh-add -D                          # forget all
ssh-add ~/.ssh/id_ed25519           # re-add
```

### `Error connecting to agent: No such file or directory`

`SSH_AUTH_SOCK` points to a path that doesn't exist (stale tmux session, sudo'd shell):

```bash
echo $SSH_AUTH_SOCK
ls -la $SSH_AUTH_SOCK
unset SSH_AUTH_SOCK
eval "$(ssh-agent -s)"
ssh-add ~/.ssh/id_ed25519
```

Inside `tmux`/`screen`, refresh the env when reattaching:

```bash
# in ~/.tmux.conf
set -g update-environment 'DISPLAY SSH_ASKPASS SSH_AUTH_SOCK SSH_CONNECTION WINDOWID XAUTHORITY'

# or wrapper for ssh
ssh () {
    if [ -S "$HOME/.ssh/auth-sock" ]; then
        export SSH_AUTH_SOCK="$HOME/.ssh/auth-sock"
    fi
    /usr/bin/ssh "$@"
}
```

### macOS Keychain integration

```bash
ssh-add --apple-use-keychain ~/.ssh/id_ed25519        # store passphrase in Keychain
ssh-add --apple-load-keychain                          # restore on login
```

`~/.ssh/config`:

```text
Host *
    UseKeychain yes
    AddKeysToAgent yes
    IgnoreUnknown UseKeychain
    IdentityFile ~/.ssh/id_ed25519
```

`IgnoreUnknown` is essential because non-macOS OpenSSH errors out on the unknown directive otherwise. `AddKeysToAgent yes` causes ssh to add the key to the running agent on first use.

### Agent forwarding (`-A` / `ForwardAgent`)

```bash
ssh -A user@bastion         # one-shot forward
```

```text
# ~/.ssh/config
Host bastion
    ForwardAgent yes
```

Security note: anyone with root on the bastion can use your agent socket to authenticate as you to other hosts. Prefer `ProxyJump` or short-lived signed certs.

## ProxyCommand / ProxyJump Errors

Verbatim:

```text
ssh: Could not resolve hostname target.example.com: Temporary failure in name resolution
channel 0: open failed: connect failed: Connection refused
channel 0: open failed: administratively prohibited: open failed
kex_exchange_identification: read: Connection reset by peer
no matching key exchange method found.
ssh_exchange_identification: Connection closed by remote host
ssh: connect to host target.example.com port 22: Connection refused (via jump)
```

### `ProxyJump` (modern, preferred)

```bash
# Single jump
ssh -J jump.example.com target.internal
ssh -J alice@jump.example.com:2222 ops@target.internal

# Multiple jumps (chain)
ssh -J jump1.example.com,jump2.internal target.deeper

# In config
```

```text
Host jump
    HostName jump.example.com
    User alice
    IdentityFile ~/.ssh/jump_ed25519

Host target
    HostName 10.0.5.42
    User ops
    ProxyJump jump
    IdentityFile ~/.ssh/target_ed25519
```

### `ProxyCommand` (legacy)

```text
Host target
    HostName 10.0.5.42
    User ops
    ProxyCommand ssh -W %h:%p jump
    # Older style:
    # ProxyCommand ssh -q jump nc %h %p
```

`ssh -W host:port` uses the modern protocol channel; `nc` (netcat) is the pre-7.3 way. `-W` is preferred — no need for `nc` on the jump.

### Diagnosing ProxyJump failures

```bash
ssh -vvv -J jump target 2>&1 | grep -iE 'proxy|connecting|kex|host key'
```

Common patterns:

```text
debug1: Setting up ProxyJump
debug1: Executing proxy command: exec ssh -W target:22 -l alice jump
```

If the proxy command fails, the inner ssh exits and target sees `Connection closed`. Run the proxy command standalone:

```bash
ssh -W target:22 jump
# (will hang if successful — Ctrl-C; error message means actual problem)
```

### Algorithm mismatch in chain

Each hop negotiates separately. A modern client → modern jump → ancient target needs the **target-only** options for the inner connection:

```text
Host target
    HostName 10.0.5.42
    ProxyJump jump
    KexAlgorithms +diffie-hellman-group1-sha1
    HostKeyAlgorithms +ssh-rsa
    PubkeyAcceptedAlgorithms +ssh-rsa
```

These apply to the inner (jump→target) leg, not the outer (you→jump) leg.

## SSH Config Issues

Verbatim:

```text
ssh: Bad configuration option: PassWordAuth
ssh: /home/me/.ssh/config line 5: Bad configuration option: ...
ssh: Could not resolve hostname mywork: nodename nor servname provided, or not known
Bad owner or permissions on /home/me/.ssh/config
/home/me/.ssh/config line 12: Match supports the following criteria...
```

### Verify config syntax

```bash
ssh -G user@host             # print effective config; never connects
ssh -G work                  # using a Host alias
ssh -F ~/.ssh/config -G work # explicit config file
```

### `Host` vs `HostName`

```text
Host work-prod              <-- alias to type as `ssh work-prod`
    HostName 10.0.0.5       <-- actual address
    User alice
    Port 2222
```

Common mistake — using `Host` for the address:

```text
# BROKEN
Host 10.0.0.5
    User alice

# Result: ssh 10.0.0.5 works, but ssh work doesn't
```

### Wildcards and `Match`

```text
# Match all
Host *
    ServerAliveInterval 60
    ServerAliveCountMax 3
    Compression yes

# Pattern
Host *.example.com
    User alice

# Negation
Host !legacy *.internal
    HostKeyAlgorithms ssh-ed25519,rsa-sha2-512

# Match block (8.8+ supports more keywords)
Match user alice host *.example.com
    IdentityFile ~/.ssh/work_ed25519

Match originalhost prod-*
    LogLevel DEBUG2

Match exec "test -f /tmp/use-vpn-cert"
    CertificateFile ~/.ssh/cert-via-vpn.pub
    IdentityFile ~/.ssh/cert-key
```

`Match` keywords: `host`, `originalhost`, `user`, `localuser`, `address`, `localnetwork`, `tagged`, `exec`, `final`, `all`.

### First-match-wins

ssh applies the **first** matching value for each directive. So order specific blocks **before** wildcard blocks:

```text
Host work-bastion
    HostName 10.0.0.5
    Port 2222

Host *
    Port 22       <-- this would win if order was reversed
```

### `Include` directive

```text
# ~/.ssh/config
Include ~/.ssh/config.d/*.conf
Include /etc/ssh/ssh_config.d/*.conf
```

Useful for separating work from personal keys.

## Public Key Format Issues

Verbatim:

```text
Load key "/home/me/.ssh/oldkey": invalid format
Load key "/home/me/.ssh/oldkey": error in libcrypto
/home/me/.ssh/oldkey is not a public key file.
Could not load host key: /etc/ssh/ssh_host_key
Could not load my certificate "/home/me/.ssh/id_ed25519-cert.pub": No such file
puttykeyfile.ppk is a PuTTY SSH-2 private key.
```

### Formats — what you might encounter

```text
PEM (RSA, traditional)         -----BEGIN RSA PRIVATE KEY-----
PKCS#8 (RSA / EC)              -----BEGIN PRIVATE KEY-----
                                -----BEGIN ENCRYPTED PRIVATE KEY-----
OpenSSH (modern, all types)    -----BEGIN OPENSSH PRIVATE KEY-----
PuTTY PPK                       PuTTY-User-Key-File-2: ssh-rsa
RFC4716 public                  ---- BEGIN SSH2 PUBLIC KEY ----
OpenSSH public                  ssh-ed25519 AAAA... user@host
```

### Convert formats with `ssh-keygen`

```bash
# Inspect format
head -1 ~/.ssh/id_rsa

# Re-encrypt / change passphrase / change format
ssh-keygen -p -f ~/.ssh/id_rsa                  # change passphrase, keep format
ssh-keygen -p -m PEM -f ~/.ssh/id_rsa           # OpenSSH → PEM
ssh-keygen -p -m PKCS8 -f ~/.ssh/id_rsa         # OpenSSH → PKCS8
ssh-keygen -p -m RFC4716 -f ~/.ssh/id_rsa       # OpenSSH → RFC4716

# Convert public key to RFC4716
ssh-keygen -e -f ~/.ssh/id_rsa.pub > id_rsa.rfc4716.pub

# Import RFC4716 public to OpenSSH
ssh-keygen -i -f id_rsa.rfc4716.pub > id_rsa.openssh.pub

# Derive .pub from private (lost the .pub file)
ssh-keygen -y -f ~/.ssh/id_rsa > ~/.ssh/id_rsa.pub
```

### PuTTY `.ppk` → OpenSSH

```bash
# Linux/macOS
sudo apt install putty-tools         # Debian/Ubuntu
brew install putty                   # macOS

puttygen key.ppk -O private-openssh -o ~/.ssh/id_rsa
puttygen key.ppk -O public-openssh -o ~/.ssh/id_rsa.pub

# Windows: open PuTTYgen, Conversions → Export OpenSSH key
```

### "no newline at end of authorized_keys"

A pasted key without a trailing newline can prevent the next key from being parsed. Always:

```bash
# Append safely
cat new_key.pub >> ~/.ssh/authorized_keys
echo '' >> ~/.ssh/authorized_keys     # ensure trailing newline

# Or use ssh-copy-id which handles newlines correctly
ssh-copy-id -i ~/.ssh/id_ed25519.pub user@host
ssh-copy-id -i ~/.ssh/id_ed25519.pub -p 2222 user@host
```

### Fingerprint and SHA256 vs MD5

```bash
ssh-keygen -lf ~/.ssh/id_ed25519.pub
# 256 SHA256:abc123... user@host (ED25519)

ssh-keygen -l -E md5 -f ~/.ssh/id_ed25519.pub
# 256 MD5:11:22:33... user@host (ED25519)
```

OpenSSH 6.8+ uses SHA256 by default; older versions used MD5. Match what your remote system shows.

## Server-Side sshd Errors

These appear in `/var/log/auth.log` (Debian/Ubuntu) or `/var/log/secure` (RHEL/CentOS) or `journalctl -u sshd`:

```text
Authentication refused: bad ownership or modes for directory /home/alice
Authentication refused: bad ownership or modes for file /home/alice/.ssh/authorized_keys
User alice from 203.0.113.5 not allowed because not listed in AllowUsers
User alice from 203.0.113.5 not allowed because account is locked
User alice from 203.0.113.5 not allowed because shell /usr/sbin/nologin does not exist
input_userauth_request: invalid user nonexistent [preauth]
Failed publickey for alice from 203.0.113.5 port 53412 ssh2: ED25519 SHA256:abc...
Failed password for alice from 203.0.113.5 port 53412 ssh2
Accepted publickey for alice from 203.0.113.5 port 53412 ssh2: ED25519 SHA256:abc...
Accepted password for alice from 203.0.113.5 port 53412 ssh2
Disconnected from authenticating user alice 203.0.113.5 port 53412 [preauth]
Disconnected from invalid user nonexistent 203.0.113.5 port 53412 [preauth]
Connection closed by 203.0.113.5 port 53412 [preauth]
Connection closed by authenticating user alice 203.0.113.5 port 53412 [preauth]
ssh_dispatch_run_fatal: Connection from 203.0.113.5 port 53412: Software caused connection abort [preauth]
fatal: ssh_packet_read_poll2: Connection closed by 203.0.113.5 port 53412
error: kex_exchange_identification: client sent invalid protocol identifier "GET / HTTP/1.1"
error: maximum authentication attempts exceeded for alice from 203.0.113.5 port 53412 ssh2 [preauth]
error: Could not load host key: /etc/ssh/ssh_host_ed25519_key
error: Bind to port 22 on 0.0.0.0 failed: Address already in use.
fatal: Cannot bind any address.
fatal: Read from socket failed: Connection reset by peer [preauth]
PAM service(sshd) ignoring max retries; 6 > 3
pam_unix(sshd:auth): authentication failure; logname= uid=0 euid=0 tty=ssh ruser= rhost=203.0.113.5 user=alice
pam_unix(sshd:session): session opened for user alice by (uid=0)
pam_systemd(sshd:session): Failed to create session: Maximum number of sessions reached
```

Each line maps to a distinct cause:

| Log message | Cause | Fix |
|---|---|---|
| bad ownership or modes for directory | `$HOME` group/world-writable | `chmod g-w,o-w $HOME` |
| bad ownership or modes for file | `authorized_keys` not 0600 | `chmod 600 ~/.ssh/authorized_keys` |
| not listed in AllowUsers | sshd_config restriction | add user or remove restriction |
| account is locked | `passwd -l` ran | `passwd -u $USER` |
| shell ... does not exist | `/etc/passwd` shell removed | `usermod -s /bin/bash $USER` |
| invalid user X | username doesn't exist | typo or attack |
| Failed publickey | offered key not in authorized_keys | step 5 above |
| Failed password | wrong password / PAM denied | check pam_unix line below |
| Disconnected ... [preauth] | client gave up | look earlier for cause |
| Connection closed ... [preauth] | client/network closed | check fail2ban |
| ssh_dispatch_run_fatal | client crashed mid-handshake | network issue or bug |
| invalid protocol identifier | non-SSH client connected | scanner / probe |
| maximum authentication attempts | tried too many keys | use IdentitiesOnly |
| Could not load host key | host key file missing | regenerate via ssh-keygen |
| Bind to port 22 ... Address already in use | another sshd / xinetd holding 22 | `sudo lsof -i:22` |
| Cannot bind any address | all listen addresses fail | check ListenAddress directive |
| Maximum number of sessions reached | logind UserTasksMax / SystemMaxSessions | `loginctl --no-pager` to inspect |

## Common SELinux / AppArmor Blocks

### SELinux contexts

```bash
# Confirm enforcing
sudo getenforce
# Enforcing  /  Permissive  /  Disabled

# Inspect labels
ls -Z /home/alice/.ssh/
# unconfined_u:object_r:ssh_home_t:s0   authorized_keys

# Restore default contexts
sudo restorecon -Rv /home/alice/.ssh/

# View denials in audit log
sudo ausearch -m AVC -ts recent | grep ssh
sudo grep AVC /var/log/audit/audit.log | tail
sealert -a /var/log/audit/audit.log | head -200

# Add a new home location
sudo semanage fcontext -a -t ssh_home_t '/data/home/[^/]+/\.ssh(/.*)?'
sudo semanage fcontext -a -t ssh_home_t '/data/home/[^/]+/\.ssh/authorized_keys'
sudo restorecon -Rv /data/home/

# Allow sshd to read non-standard locations
sudo setsebool -P ssh_chroot_rw_homedirs on
sudo setsebool -P ssh_sysadm_login on
```

Common SELinux booleans:

```bash
getsebool -a | grep ssh
# ssh_chroot_rw_homedirs --> off
# ssh_keysign --> off
# ssh_sysadm_login --> off
# ssh_use_tcpd --> off
```

### AppArmor

```bash
sudo aa-status
sudo dmesg | grep -i apparmor

# Disable a profile temporarily
sudo aa-disable /etc/apparmor.d/usr.sbin.sshd
# Or set complain mode
sudo aa-complain /etc/apparmor.d/usr.sbin.sshd

# Reload after edits
sudo apparmor_parser -r /etc/apparmor.d/usr.sbin.sshd
```

A non-standard `AuthorizedKeysFile` location may be rejected by an AppArmor profile that whitelists only standard paths.

## Port-Forwarding Errors

### Local forward (`-L`)

Bring a remote port to your local box:

```bash
ssh -L 8080:internal.example.com:80 user@bastion
ssh -L 0.0.0.0:8080:internal:80 user@bastion       # listen on all interfaces (needs GatewayPorts)
ssh -fNL 5432:db.internal:5432 user@bastion        # background, no command
```

Errors:

```text
bind [::]:8080: Address already in use
channel_setup_fwd_listener_tcpip: cannot listen to port: 8080
Could not request local forwarding.
channel 2: open failed: connect failed: Connection refused
channel 2: open failed: connect failed: Connection timed out
channel 2: open failed: administratively prohibited: open failed
```

| Error | Cause | Fix |
|---|---|---|
| `Address already in use` | local 8080 already bound | `ss -tlnp \| grep 8080`; use different port |
| `cannot listen to port` | privileged port (<1024) without root | use ≥1024 or `sudo ssh` |
| `Could not request local forwarding` | server has `AllowTcpForwarding no` | enable on server |
| `connect failed: Connection refused` | destination not listening | verify on bastion: `nc -zv internal 80` |
| `administratively prohibited` | server `PermitOpen` doesn't allow target | adjust `PermitOpen` |

### Remote forward (`-R`)

Expose your local port on the remote box:

```bash
ssh -R 9090:localhost:80 user@bastion
ssh -R 0.0.0.0:9090:localhost:80 user@bastion      # bind to all on bastion (needs GatewayPorts)
```

Errors:

```text
remote port forwarding failed for listen port 9090
Warning: remote port forwarding failed for listen port 9090
Server has disabled GatewayPorts. Forwarded ports will only be available locally.
```

Server fixes:

```text
# /etc/ssh/sshd_config
AllowTcpForwarding yes
GatewayPorts clientspecified       # or yes for global; no = localhost-only (default)
```

`GatewayPorts clientspecified` means: if client says `0.0.0.0:9090:...`, bind to `0.0.0.0`; else bind to localhost. Safer than `yes`.

### Dynamic forward / SOCKS (`-D`)

```bash
ssh -D 1080 user@bastion
# Use 127.0.0.1:1080 as SOCKS5 proxy in browser, curl --socks5-hostname, etc.

curl --socks5-hostname localhost:1080 https://internal.example.com/
```

Errors:

```text
bind [::]:1080: Address already in use
Could not request dynamic forwarding.
```

### Server-side controls

```bash
sudo sshd -T | grep -iE 'forward|gatewayports|permit'
# allowtcpforwarding yes
# allowstreamlocalforwarding yes
# disableforwarding no
# gatewayports no
# permitopen any
# permittunnel no
```

Restrict per-user:

```text
Match User confined
    AllowTcpForwarding no
    PermitOpen 127.0.0.1:5432
    X11Forwarding no
```

## SCP / SFTP Errors

Verbatim:

```text
scp: Connection closed
scp: not a regular file
scp: /path/to/dir: not a regular file
scp: error: unexpected filename: .
scp: error: unexpected filename: ..
subsystem request failed on channel 0
Received message too long 1212501832
sftp: server received message too long ...
sftp> Couldn't read packet: Connection reset by peer
ash: scp: command not found
This is the recommended path for the new SCP protocol [...]
```

### Connection closed (banner contamination)

If your `.bashrc` / `.profile` writes to stdout (banners, color codes, `motd`-like noise), it may contaminate the SCP/SFTP byte stream:

```bash
# BROKEN — produces "received message too long" on scp
echo "Welcome alice $(date)"      # in ~/.bashrc

# Fix: emit only when interactive
if [ -t 0 ] && [ -t 1 ]; then
    echo "Welcome alice"
fi

# Or guard against non-interactive
[[ $- != *i* ]] && return
```

### `subsystem request failed on channel 0`

`sftp-server` not configured on server.

```bash
# /etc/ssh/sshd_config
Subsystem sftp /usr/lib/openssh/sftp-server      # Debian/Ubuntu
Subsystem sftp /usr/libexec/openssh/sftp-server  # RHEL/CentOS
Subsystem sftp internal-sftp                     # built-in
```

`internal-sftp` is faster (no fork) and is required for chroot:

```text
Match Group sftp-only
    ChrootDirectory /srv/sftp/%u
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
```

### scp protocol vs sftp protocol

Since OpenSSH 9.0, `scp` defaults to using the **SFTP protocol** (not the legacy SCP protocol). Behavior differences:

```text
scp: warning: SCP protocol over SSH is deprecated. Set SCP_USE_LEGACY_PROTOCOL=1 to revert.

# Legacy if you need it:
scp -O src dst                          # force old SCP protocol
SCP_USE_LEGACY_PROTOCOL=1 scp src dst   # env var

# Force new (SFTP) explicitly:
scp -s src dst                          # force SFTP transport
```

Differences that bite:

- New protocol expands globs **on client**, not server (so `~/*.txt` may resolve differently)
- New protocol stricter about path traversal and special filenames
- Transfer of millions of small files is faster on SFTP transport
- `~user/file` syntax may not work the same

### scp `not a regular file`

```bash
# WRONG — copying a directory without -r
scp -P 22 user@host:/etc/nginx /tmp/
# scp: /etc/nginx: not a regular file

# Fix
scp -rP 22 user@host:/etc/nginx /tmp/
```

### rsync over SSH (alternative to scp)

```bash
rsync -avz --progress -e 'ssh -p 2222 -i ~/.ssh/id_ed25519' \
    src/ user@host:/dest/

rsync -avzP --delete src/ user@host:/dest/   # mirror

rsync -avz --rsync-path='sudo rsync' src/ user@host:/etc/destination/   # sudo on remote
```

`rsync` over SSH retries, resumes, computes deltas — preferred for anything other than single small files.

## Verbose Mode Reading Guide

Levels:

```bash
ssh -v user@host       # debug1
ssh -vv user@host      # debug1 + debug2
ssh -vvv user@host     # debug1 + debug2 + debug3 (protocol-level)
```

Annotated successful connection:

```text
debug1: Reading configuration data /home/me/.ssh/config
                                                    -- which user config used
debug1: /home/me/.ssh/config line 5: Applying options for *
debug1: /home/me/.ssh/config line 12: Applying options for work
                                                    -- which Host blocks matched
debug1: Reading configuration data /etc/ssh/ssh_config
                                                    -- which system config used
debug1: Connecting to work.example.com [203.0.113.5] port 22.
                                                    -- DNS resolved
debug1: Connection established.
                                                    -- TCP handshake complete
debug1: identity file /home/me/.ssh/id_rsa type 0
                                                    -- type 0 = file present and parsed
debug1: identity file /home/me/.ssh/id_rsa-cert type -1
                                                    -- type -1 = file not found
debug1: identity file /home/me/.ssh/id_ed25519 type 3
                                                    -- type 3 = ed25519
debug1: Local version string SSH-2.0-OpenSSH_9.6
                                                    -- client version sent
debug1: Remote protocol version 2.0, remote software version OpenSSH_8.9p1
                                                    -- server version received
debug1: compat_banner: match: OpenSSH_8.9p1
debug1: Authenticating to work.example.com:22 as 'alice'
debug1: load_hostkeys: fopen /home/me/.ssh/known_hosts2: No such file or directory
                                                    -- harmless; legacy file
debug1: SSH2_MSG_KEXINIT sent
debug1: SSH2_MSG_KEXINIT received
debug1: kex: algorithm: curve25519-sha256
debug1: kex: host key algorithm: ssh-ed25519
debug1: kex: server->client cipher: chacha20-poly1305@openssh.com MAC: <implicit> compression: none
debug1: kex: client->server cipher: chacha20-poly1305@openssh.com MAC: <implicit> compression: none
                                                    -- algorithms agreed
debug1: expecting SSH2_MSG_KEX_ECDH_REPLY
debug1: SSH2_MSG_KEX_ECDH_REPLY received
debug1: Server host key: ssh-ed25519 SHA256:abcd...
                                                    -- host key offered
debug1: Host 'work.example.com' is known and matches the ED25519 host key.
debug1: Found key in /home/me/.ssh/known_hosts:42
                                                    -- which line of known_hosts matched
debug1: rekey out after 134217728 blocks
debug1: SSH2_MSG_NEWKEYS sent / received
debug1: rekey in after 134217728 blocks
debug1: Will attempt key: /home/me/.ssh/id_rsa RSA SHA256:efgh...
debug1: Will attempt key: /home/me/.ssh/id_ed25519 ED25519 SHA256:ijkl... explicit
debug1: SSH2_MSG_EXT_INFO received
debug1: kex_input_ext_info: server-sig-algs=<rsa-sha2-256,rsa-sha2-512,ssh-ed25519,...>
debug1: SSH2_MSG_SERVICE_ACCEPT received
debug1: Authentications that can continue: publickey,password
                                                    -- server's offered methods
debug1: Next authentication method: publickey
debug1: Offering public key: /home/me/.ssh/id_rsa RSA SHA256:efgh...
debug1: Authentications that can continue: publickey,password
                                                    -- key rejected (silently)
debug1: Offering public key: /home/me/.ssh/id_ed25519 ED25519 SHA256:ijkl... explicit
debug1: Server accepts key: /home/me/.ssh/id_ed25519 ED25519 SHA256:ijkl... explicit
                                                    -- key accepted
debug1: Authentication succeeded (publickey).
debug1: Authenticated to work.example.com ([203.0.113.5]:22) using "publickey".
debug1: channel 0: new [client-session]
debug1: Sending environment.
debug1: Sending env LANG = en_US.UTF-8
debug1: Sending command: <interactive>
                                                    -- shell time
```

### Diagnostic patterns

| Pattern in `-vvv` | Meaning |
|---|---|
| `identity file ... type -1` | file missing |
| `Offering public key:` then `Authentications that can continue:` | server rejected silently — check authorized_keys |
| `Server accepts key:` | success |
| `no mutual signature algorithm` | algorithm mismatch (pubkey type) — RSA-SHA1 disabled? |
| `Connection reset by peer` early | fail2ban / firewall |
| `Connection closed by remote host` after KEX | server-side `Match` denial |
| `Permission denied (publickey)` after offering | no offered key matched authorized_keys |
| `kex_exchange_identification: ...` | both sides connected but server hung up before banner |
| `no matching X found` | algorithm mismatch — see Cipher section |

## Multiplexing (ControlMaster) Issues

### Setup

```text
# ~/.ssh/config
Host *
    ControlMaster auto
    ControlPath ~/.ssh/cm-%r@%h:%p
    ControlPersist 10m

# Better, with shorter path
Host *
    ControlMaster auto
    ControlPath ~/.ssh/cm/%C
    ControlPersist 600
```

`%C` (OpenSSH 6.7+) is a hash — keeps paths short. Always pre-create the directory:

```bash
mkdir -m 700 -p ~/.ssh/cm
```

### Errors

```text
Control socket connect(/home/me/.ssh/cm-alice@work.example.com:22): No such file or directory
ControlPath "/very/long/path/that/exceeds/the/socket/path/limit/cm-...": path too long
control socket: /home/me/.ssh/cm-...: Connection refused
debug1: ControlPath in use, but path is invalid
mux_client_request_session: read from master failed: Broken pipe
```

| Error | Cause | Fix |
|---|---|---|
| `path too long` | Unix socket path > 104 bytes (Linux) | use `%C` hash form, or shorter dir |
| `Connection refused` | master process died | re-establish with first `ssh` |
| `read from master failed` | master killed | `ssh -O exit` and reconnect |

### Multiplex commands

```bash
ssh -O check user@host         # is master alive?
ssh -O exit user@host          # cleanly close master
ssh -O stop user@host          # stop accepting new sessions, keep existing
ssh -O forward -L8080:host:80 user@host    # add forward to existing master
ssh -O cancel -L8080:host:80 user@host     # remove forward
```

Benefit: subsequent `ssh user@host` reuses the existing TCP+TLS+auth, connecting in <100ms.

### Disabling per-host

```text
Host nomultiplex
    ControlMaster no
    ControlPath none
```

### Master died — orphan socket

```bash
ls -la ~/.ssh/cm/
# remove stale socket
rm ~/.ssh/cm/*
ssh user@host         # creates new master
```

## Windows / Git for Windows / WSL

### OpenSSH for Windows

Built-in since Windows 10 1809:

```powershell
# Install client (usually pre-installed)
Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0

# Install server
Add-WindowsCapability -Online -Name OpenSSH.Server~~~~0.0.1.0

# Start service
Start-Service sshd
Set-Service -Name sshd -StartupType Automatic

# Verify
Get-Service sshd
ssh -V
```

Config file path:

```text
%PROGRAMDATA%\ssh\sshd_config
%USERPROFILE%\.ssh\config
%USERPROFILE%\.ssh\authorized_keys
```

For administrators, OpenSSH for Windows uses a special file:

```text
%PROGRAMDATA%\ssh\administrators_authorized_keys
```

Permissions:

```powershell
$path = "C:\ProgramData\ssh\administrators_authorized_keys"
icacls $path /inheritance:r
icacls $path /grant "Administrators:F"
icacls $path /grant "SYSTEM:F"
```

### Permissions on Windows

Windows doesn't honor POSIX `chmod` directly. OpenSSH for Windows uses ACLs:

```text
Permissions for 'C:\Users\me\.ssh\id_ed25519' are too open.
It is required that your private key files are NOT accessible by others.
```

Fix via `icacls`:

```powershell
icacls C:\Users\me\.ssh\id_ed25519 /inheritance:r
icacls C:\Users\me\.ssh\id_ed25519 /grant:r "$($env:USERNAME):(R)"
icacls C:\Users\me\.ssh\id_ed25519 /remove "BUILTIN\Users"
icacls C:\Users\me\.ssh\id_ed25519 /remove "NT AUTHORITY\Authenticated Users"
```

### Git Bash / MSYS2

`chmod 600` from Git Bash usually works because Git Bash translates to NTFS ACLs:

```bash
# Git Bash
chmod 600 ~/.ssh/id_ed25519
chmod 700 ~/.ssh
```

If it doesn't take, fall back to `icacls` or use Windows OpenSSH directly.

### WSL

Inside WSL, normal Linux SSH applies — `~/.ssh/` lives in the WSL filesystem (`/home/$USER/.ssh`).

Cross-WSL:

```bash
# WSL → host Windows OpenSSH server
ssh username@$(cat /etc/resolv.conf | grep nameserver | awk '{print $2}')

# Host Windows → WSL OpenSSH server
# 1. Install openssh-server in WSL
sudo apt install openssh-server
sudo service ssh start
# 2. WSL has dynamic IP; configure port-proxy
netsh interface portproxy add v4tov4 listenport=2222 listenaddress=0.0.0.0 connectport=22 connectaddress=$(wsl hostname -I)
# 3. Connect
ssh -p 2222 wsluser@127.0.0.1
```

Better: use Windows OpenSSH for inbound, WSL for development.

## macOS-Specific

### Touch ID / Apple Watch / Keychain

```bash
ssh-add --apple-use-keychain ~/.ssh/id_ed25519
ssh-add --apple-load-keychain
```

`~/.ssh/config`:

```text
Host *
    AddKeysToAgent yes
    UseKeychain yes
    IgnoreUnknown UseKeychain
    IdentityFile ~/.ssh/id_ed25519
```

### `ssh-add: agent refused operation`

```bash
ssh-add -l
# The agent has no identities.

ssh-add ~/.ssh/id_ed25519
# Asks for passphrase

# If error: agent refused
# Fix: kill stale agent
launchctl list | grep ssh
sudo killall ssh-agent
ssh-add ~/.ssh/id_ed25519
```

### macOS Sequoia 15 — keychain prompt

15.0+ may prompt for keychain access on each ssh, even with `UseKeychain yes`. Fix:

```bash
# Re-add and check 'Always Allow' in the keychain dialog
security find-generic-password -a $USER -s 'SSH:'  # see entries
ssh-add --apple-use-keychain ~/.ssh/id_ed25519
```

### Apple Silicon vs Intel

Old `ssh` builds compiled for Intel run via Rosetta 2. The system `ssh` (`/usr/bin/ssh`) is universal. Issues arise with:

- Homebrew SSH (`/opt/homebrew/bin/ssh` Apple Silicon vs `/usr/local/bin/ssh` Intel)
- VPN clients that hook DNS — confirm with `which ssh`

```bash
which ssh
ssh -V
file $(which ssh)
```

## Common Gotchas — Broken vs Fixed

### 1. `$HOME` group-writable

```text
# auth.log
Authentication refused: bad ownership or modes for directory /home/alice
```

```bash
# BROKEN
ls -ld /home/alice
# drwxrwsr-x  alice alice  /home/alice

# FIX
chmod g-w /home/alice
chmod o-w /home/alice
ls -ld /home/alice
# drwxr-xr-x  alice alice  /home/alice
```

### 2. `authorized_keys` with mangled newlines

```bash
# BROKEN — copy-pasted from notepad with \r\n
hexdump -C ~/.ssh/authorized_keys | head -2
# 0000  73 73 68 2d 65 64 32 35 35 31 39 20 41 41 41 41  ssh-ed25519 AAAA
# ...
# 0050  3d 3d 20 75 73 65 72 0d 0a                       == user..

# FIX
sed -i 's/\r$//' ~/.ssh/authorized_keys
chmod 600 ~/.ssh/authorized_keys

# Or use ssh-copy-id which won't introduce CRs
ssh-copy-id -i ~/.ssh/id_ed25519.pub user@host
```

### 3. Smart quotes / unicode artifacts

```bash
# BROKEN — pasted via certain webmail clients
hexdump -C ~/.ssh/authorized_keys | grep -E '\xe2|\xc2'
# Found: e2 80 9c (LEFT DOUBLE QUOTATION MARK), e2 80 9d (RIGHT)

# FIX
# Re-paste as plain text via terminal echo or ssh-copy-id
```

### 4. User mismatch

```bash
# BROKEN
ssh root@host       # but only alice's key is in /home/alice/.ssh/authorized_keys
# Permission denied (publickey).

# FIX
ssh alice@host
```

### 5. Agent has key but ssh isn't using it

```bash
# BROKEN
ssh-add -l         # shows key
ssh -v user@host   # never offers it
# Possibly because ssh-agent isn't on this shell's SSH_AUTH_SOCK

# FIX
echo $SSH_AUTH_SOCK
ssh-add -l         # confirm in the right agent
```

If `~/.ssh/config` has `IdentitiesOnly yes` and no `IdentityFile` line, ssh won't use the agent. Add an explicit:

```text
Host work
    IdentityFile ~/.ssh/work_ed25519
    IdentitiesOnly yes
    AddKeysToAgent yes
```

### 6. `AllowTcpForwarding no` on hardened server

```bash
# BROKEN
ssh -L 5432:db:5432 user@bastion
# debug1: Local connections to LOCALHOST:5432 forwarded to remote address db:5432
# bind succeeded, but...
# channel 2: open failed: administratively prohibited: open failed

# FIX (server-side)
sudo grep -i forwarding /etc/ssh/sshd_config
echo 'AllowTcpForwarding yes' | sudo tee -a /etc/ssh/sshd_config
sudo systemctl restart sshd
```

### 7. Port 22 reachable from your network but firewall drops you

```bash
# BROKEN
ssh user@public-host
# (long timeout)
# ssh: connect to host public-host port 22: Connection timed out

# Diagnose
nc -zv -w 5 public-host 22
mtr -n public-host
sudo tcpdump -nni any host public-host

# FIX (case A — server firewall)
sudo ufw allow from 203.0.113.0/24 to any port 22 proto tcp

# FIX (case B — cloud security group)
aws ec2 authorize-security-group-ingress --group-id sg-xxx --protocol tcp --port 22 --cidr 203.0.113.0/24
```

### 8. SFTP-only chroot user with bind-mount

```text
# /etc/ssh/sshd_config
Match Group sftponly
    ChrootDirectory /srv/chroot/%u
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
```

Common breakage: `ChrootDirectory` must be **owned by root** with no group/world write:

```bash
# BROKEN
ls -ld /srv/chroot/alice
# drwxr-xr-x alice alice ...
# sshd refuses → Connection closed

# FIX
chown root:root /srv/chroot/alice
chmod 755 /srv/chroot/alice
mkdir -p /srv/chroot/alice/upload
chown alice:sftponly /srv/chroot/alice/upload
```

### 9. Old key cached in agent ignoring `IdentityFile`

```bash
# BROKEN
ssh-add ~/.ssh/old_id_rsa
ssh -i ~/.ssh/new_ed25519 user@host
# Server tries the agent's old key first, hits MaxAuthTries

# FIX (option 1)
ssh -i ~/.ssh/new_ed25519 -o IdentitiesOnly=yes user@host

# FIX (option 2)
ssh-add -d ~/.ssh/old_id_rsa
ssh-add ~/.ssh/new_ed25519

# FIX (option 3, permanent)
# In ~/.ssh/config:
Host work
    IdentityFile ~/.ssh/new_ed25519
    IdentitiesOnly yes
```

### 10. Wrong known_hosts entry due to floating-IP DNS

```bash
# BROKEN
# host.example.com behind LB; one backend returns ed25519 SHA256:A
# another returns ed25519 SHA256:B
# Random failures with "Host key verification failed"

# FIX (option 1) — add multiple entries
ssh-keyscan -t ed25519 host.example.com.backend1 >> ~/.ssh/known_hosts
ssh-keyscan -t ed25519 host.example.com.backend2 >> ~/.ssh/known_hosts

# FIX (option 2) — sync host keys across backends
# Pick one machine's keys, distribute to all backends:
sudo scp /etc/ssh/ssh_host_* root@backend2:/etc/ssh/
sudo systemctl restart sshd
```

### 11. Disabled root login

```bash
# BROKEN
ssh root@host
# Permission denied (publickey).

# Confirm
sudo grep -i permitrootlogin /etc/ssh/sshd_config
# PermitRootLogin no

# FIX (correct way) — use a sudo-able user
ssh alice@host
sudo -i

# FIX (only if appropriate) — allow root with key only
sudo sed -i 's/^PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config
sudo systemctl restart sshd
```

### 12. PAM `motd` error noise

```text
# auth.log
sshd[1234]: pam_motd(sshd:session): pam_motd: error executing /etc/update-motd.d/50-landscape-sysinfo
```

Cause: a script in `/etc/update-motd.d/` failed. Doesn't block login but pollutes logs.

```bash
sudo bash -c 'for f in /etc/update-motd.d/*; do echo "--- $f ---"; bash "$f"; done'
sudo chmod -x /etc/update-motd.d/50-landscape-sysinfo  # disable broken one
```

### 13. SSH_AUTH_SOCK lost after `sudo`

```bash
# BROKEN
sudo ssh user@host
# Could not open a connection to your authentication agent

# FIX — preserve
sudo --preserve-env=SSH_AUTH_SOCK ssh user@host

# Or in /etc/sudoers (visudo)
Defaults env_keep+="SSH_AUTH_SOCK"
```

### 14. Host with multiple IPs, one stale

```bash
# BROKEN — host has dual-stack and IPv6 path is broken
ssh host
# (long timeout, eventually IPv4 works)

# FIX — force IPv4
ssh -4 host
# or in config:
# AddressFamily inet
```

### 15. `LocaleVar` mismatch

```bash
# BROKEN — server complains about locale
ssh user@host
perl: warning: Setting locale failed.
perl: warning: Please check that your locale settings:
        LANGUAGE = (unset),
        LC_ALL = (unset),
        LANG = "en_US.UTF-8"
    are supported and installed on your system.

# FIX (option 1) — server side
sudo locale-gen en_US.UTF-8

# FIX (option 2) — stop sending locale
# In ~/.ssh/config:
SendEnv -LANG -LC_*
# Or remove from /etc/ssh/ssh_config: AcceptEnv LANG LC_*
```

## Hardening Quick Reference

Recommended `sshd_config`:

```text
# /etc/ssh/sshd_config

# Protocol & port
Port 22
AddressFamily any
ListenAddress 0.0.0.0
ListenAddress ::

# Host keys (modern only)
HostKey /etc/ssh/ssh_host_ed25519_key
HostKey /etc/ssh/ssh_host_rsa_key

# Cipher hardening (OpenSSH 9.x)
KexAlgorithms sntrup761x25519-sha512@openssh.com,curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com,aes256-ctr,aes192-ctr,aes128-ctr
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com,umac-128-etm@openssh.com
HostKeyAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256
PubkeyAcceptedAlgorithms ssh-ed25519,rsa-sha2-512,rsa-sha2-256

# Authentication
PermitRootLogin prohibit-password
PubkeyAuthentication yes
PasswordAuthentication no
PermitEmptyPasswords no
ChallengeResponseAuthentication no
KbdInteractiveAuthentication no
UsePAM yes
AuthenticationMethods publickey

# Limits
MaxAuthTries 3
MaxSessions 10
MaxStartups 10:30:60
LoginGraceTime 60

# User restrictions
AllowGroups ssh-users wheel
DenyUsers nobody root

# Forwarding
AllowTcpForwarding yes
AllowAgentForwarding no
X11Forwarding no
GatewayPorts no
PermitTunnel no

# Idle behavior
ClientAliveInterval 300
ClientAliveCountMax 2
TCPKeepAlive yes

# Logging
LogLevel VERBOSE
SyslogFacility AUTH

# Banner & motd
Banner /etc/issue.net
PrintMotd no
PrintLastLog yes

# Subsystem
Subsystem sftp internal-sftp -f AUTHPRIV -l INFO

# Per-group rules
Match Group sftp-only
    ChrootDirectory /srv/sftp/%u
    ForceCommand internal-sftp
    AllowTcpForwarding no
    X11Forwarding no
```

Validate before restart:

```bash
sudo sshd -t                                    # syntax check
sudo sshd -T | grep -iE 'permitroot|password|pubkey|maxauth'  # effective values

# Test config from another shell BEFORE killing your session
sudo /usr/sbin/sshd -p 2222 -d
# In another terminal:
ssh -p 2222 user@host
# If works, restart real sshd:
sudo systemctl restart sshd

# Always have a backup login channel (console, IPMI, second ssh) when changing sshd_config
```

CIS / STIG quick wins:

```text
PermitRootLogin no
PasswordAuthentication no
MaxAuthTries 4
ClientAliveInterval 300
ClientAliveCountMax 0
LoginGraceTime 60
Banner /etc/issue.net
Ciphers aes256-gcm@openssh.com,chacha20-poly1305@openssh.com,aes256-ctr,aes192-ctr,aes128-gcm@openssh.com,aes128-ctr
MACs hmac-sha2-512-etm@openssh.com,hmac-sha2-256-etm@openssh.com,umac-128-etm@openssh.com
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org,diffie-hellman-group16-sha512,diffie-hellman-group18-sha512
```

`fail2ban` for brute-force mitigation:

```text
# /etc/fail2ban/jail.local
[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
findtime = 10m
bantime = 1h
```

```bash
sudo systemctl restart fail2ban
sudo fail2ban-client status sshd
sudo fail2ban-client unban 203.0.113.5
```

`PerSourcePenalties` (OpenSSH 9.8+) — built-in:

```text
PerSourcePenalties yes
PerSourceMaxStartups 3
PerSourceNetBlockSize 32:128
```

## Diagnostic Tools

```bash
# Client side
ssh -v user@host                  # verbose level 1
ssh -vv user@host                 # level 2
ssh -vvv user@host                # level 3 (protocol)
ssh -G user@host                  # print effective config; never connects
ssh -F /dev/null user@host        # connect with no config (bare defaults)
ssh -o LogLevel=DEBUG3 user@host  # one-shot debug
ssh -Q kex                        # supported KEX algorithms
ssh -Q cipher                     # supported ciphers
ssh -Q mac                        # supported MACs
ssh -Q HostKeyAlgorithms          # supported host-key algorithms
ssh -Q sig                        # supported signature algorithms
ssh -Q help                       # all queryable types

ssh-keygen -y -f keyfile          # derive .pub from private
ssh-keygen -lf keyfile            # print fingerprint
ssh-keygen -lvf keyfile           # ASCII-art bubble
ssh-keygen -F hostname            # search known_hosts (works with hashing)
ssh-keygen -R hostname            # remove from known_hosts
ssh-keygen -H -f known_hosts      # rehash known_hosts

ssh-keyscan -t rsa,ecdsa,ed25519 host    # fetch host keys
ssh-keyscan -p 2222 host                  # nondefault port
ssh-keyscan -T 5 host                     # 5-second timeout

ssh-add -l                        # list fingerprints
ssh-add -L                        # list public keys
ssh-add -D                        # remove all
ssh-add -d keyfile                # remove one
ssh-add -t 3600 keyfile           # add with TTL
ssh-add --apple-use-keychain key  # macOS Keychain

# Server side
sudo sshd -T                      # print effective sshd_config
sudo sshd -T -C user=alice,host=app01,addr=10.0.0.5  # simulate Match
sudo sshd -t                      # config syntax check
sudo sshd -ddd -p 2222            # foreground debug (alt port)

journalctl -u sshd -f             # live log
journalctl -u sshd -n 200         # last 200 lines
journalctl -u sshd --since "10 min ago"
journalctl _COMM=sshd -p err

tail -F /var/log/auth.log         # Debian/Ubuntu
tail -F /var/log/secure           # RHEL/CentOS

# Network
nc -zv host 22
nmap -p 22 host
nmap --script ssh2-enum-algos -p 22 host
nmap --script ssh-hostkey,ssh-auth-methods -p 22 host
ss -tlnp | grep ssh
sudo lsof -i:22

# Auth audit
sudo lastb -a 50                  # failed logins
sudo last -a 50                   # successful logins
sudo aulastlog                    # per-user last login (audit)
sudo journalctl -u sshd | grep -i "Accepted\|Failed" | tail
```

## Idioms

- Always start debugging with `ssh -v` (or `-vv` / `-vvv`)
- Match permissions exactly: `0700` on `~/.ssh`, `0600` on private keys and config and authorized_keys
- Use `ssh-agent` plus `ssh-add` (with `--apple-use-keychain` on macOS) — type passphrase once per session
- Use `IdentitiesOnly yes` with `-i` (or `IdentityFile`) when you have many keys to avoid `Too many authentication failures`
- Use `ProxyJump` (`-J`) over `ProxyCommand` for multi-hop — simpler and uses the SSH protocol channel
- Prefer `ed25519` keys; if you must use RSA, generate at least 4096-bit and use SHA-2 signatures
- Verify host-key fingerprints out-of-band on first connection — TOFU is convenience, not security
- `sshd -T` to view effective server config; `ssh -G` for client config
- Test `sshd_config` with `sshd -t` before `systemctl restart sshd`; keep a second login channel open
- For large deployments, use SSH certificates (`ssh-keygen -s ca_key -I id user@host`) instead of distributing keys
- `rsync -e ssh` over `scp` for any non-trivial file transfer
- Use `ControlMaster auto` + `ControlPersist 10m` to make repeated connections instant
- When a key fails silently, look at server `auth.log` first — the cause is almost always there

## See Also

- [ssh](../verify/ssh.md)
- [sshd](../verify/sshd.md)
- [openssl](../verify/openssl.md)
- [troubleshooting/dns-errors](../troubleshooting/dns-errors.md)
- [troubleshooting/linux-errors](../troubleshooting/linux-errors.md)
- [troubleshooting/git-errors](../troubleshooting/git-errors.md)

## References

- `man ssh(1)` — client manual
- `man ssh_config(5)` — client config
- `man sshd(8)` — server manual
- `man sshd_config(5)` — server config
- `man ssh-keygen(1)` — key management
- `man ssh-agent(1)` — agent
- `man ssh-add(1)` — add keys to agent
- `man ssh-keyscan(1)` — fetch host keys
- `man scp(1)` — secure copy
- `man sftp(1)` — secure FTP-like client
- `man sftp-server(8)` — SFTP subsystem server
- OpenSSH home page — https://www.openssh.com/
- OpenSSH manual page — https://www.openssh.com/manual.html
- OpenSSH release notes — https://www.openssh.com/releasenotes.html (especially 8.8 ssh-rsa deprecation, 9.0 scp protocol switch, 9.6 Terrapin, 9.8 regreSSHion)
- RFC 4250 — SSH Protocol Assigned Numbers
- RFC 4251 — SSH Protocol Architecture
- RFC 4252 — SSH Authentication Protocol
- RFC 4253 — SSH Transport Layer
- RFC 4254 — SSH Connection Protocol
- RFC 4256 — Generic Message Exchange Authentication (keyboard-interactive)
- RFC 4716 — SSH Public Key File Format
- RFC 5656 — Elliptic Curve Algorithm Integration in the SSH Transport Layer
- RFC 6668 — SHA-2 Data Integrity Verification for SSH
- RFC 8332 — Use of RSA Keys with SHA-256 and SHA-512
- RFC 8709 — Ed25519 and Ed448 Public Key Algorithms for SSH
- RFC 8731 — SSH Key Exchange Method Using Curve25519 and Curve448
- CIS Benchmarks — Linux SSH Server hardening
- DISA STIG — Red Hat / Ubuntu OpenSSH STIGs
- CVE-2023-48795 — Terrapin attack
- CVE-2024-6387 — regreSSHion (OpenSSH server signal handler race)
