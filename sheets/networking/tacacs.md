# TACACS+

Terminal Access Controller Access-Control System Plus — TCP-based AAA protocol for device administration with full-payload encryption, per-command authorization, and granular accounting (RFC 8907).

## Setup

```bash
# Debian/Ubuntu — original Shrubbery tac_plus
sudo apt-get update
sudo apt-get install tacacs+
# Package name: tacacs+
# Binary: /usr/sbin/tac_plus
# Config: /etc/tacacs+/tac_plus.conf
# Logs:   /var/log/tac_plus.log
sudo systemctl enable --now tacacs_plus
sudo systemctl status tacacs_plus

# RHEL/CentOS/Rocky — install via EPEL
sudo dnf install epel-release
sudo dnf install tac_plus
# Some RH families ship tac_plus.conf at /etc/tac_plus.conf
sudo systemctl enable --now tac_plus

# F5/Pro: Shrubbery tac_plus (the historical reference implementation)
# Source: http://www.shrubbery.net/tac_plus/
wget ftp://ftp.shrubbery.net/pub/tac_plus/tacacs-F4.0.4.28.tar.gz
tar xzf tacacs-F4.0.4.28.tar.gz && cd tacacs-F4.0.4.28
./configure --prefix=/usr/local --with-libwrap
make && sudo make install
# Builds: /usr/local/sbin/tac_plus

# tac_plus-ng — modern fork with IPv6, mTLS, better config parser
# Source: https://github.com/event-driven-servers/tac_plus-ng
git clone https://github.com/event-driven-servers/tac_plus-ng.git
cd tac_plus-ng
./configure --prefix=/usr/local
make && sudo make install
/usr/local/sbin/tac_plus-ng -v

# tac_plus-pam variant — uses PAM as backend for authentication
# Lets you front Linux PAM with TACACS+ protocol
# /etc/tac_plus.conf has `default authentication = pam`

# Cisco's commercial flavor:
# Cisco Secure ACS (end-of-life 2020) — replaced by:
# Cisco Identity Services Engine (ISE) — adds RADIUS, TACACS+, posture, profiling, SXP

# Run in foreground, debug verbosity 8
sudo /usr/sbin/tac_plus -G -d 8 -C /etc/tacacs+/tac_plus.conf
# -G  do not fork into background (for systemd or debugging)
# -d  debug-bitmask (1, 8, 16, 32, 64, 128, 256, 512, 1024 — combine with OR)
# -C  config file
# -P  parse the config and exit (syntax check)
# -v  print version
# -L  syslog logging
# -B  bind address (multi-homed hosts)
# -p  TCP port (default 49)

# Verify config syntax
sudo /usr/sbin/tac_plus -P -C /etc/tacacs+/tac_plus.conf

# Client side: pam_tacplus library on Linux
sudo apt-get install libpam-tacplus libnss-tacplus
# /etc/tacplus.conf — global library config
# /etc/pam.d/sshd  — PAM stack inclusion
```

## Protocol Overview

```
TACACS+ — RFC 8907 (September 2020) — Obsoletes draft RFC 1492 (informational)
History:
  TACACS  — DARPA, RFC 927/1492 — UDP, cleartext (obsolete)
  XTACACS — Cisco extension, also obsolete
  TACACS+ — Cisco proprietary 1993, then RFC 8907 standards-track 2020

Transport: TCP, well-known port 49 (assigned by IANA)
Cipher:    XOR obfuscation of payload using MD5(session_id||key||version||seq)
Header:    12 bytes, never encrypted
Payload:   Authentication / Authorization / Accounting packet types
Sessions:  one TCP per request OR single-connection-mode (long-lived TCP)
```

```
TACACS+ Header (12 bytes, cleartext):

 0                   1                   2                   3
 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7
+---------------+---------------+---------------+---------------+
|major  |minor  |     type      |    seq_no     |     flags     |
|version|version|               |               |               |
+---------------+---------------+---------------+---------------+
|                          session_id                           |
+---------------+---------------+---------------+---------------+
|                            length                             |
+---------------+---------------+---------------+---------------+
|                       payload (encrypted)                     |
+---------------+---------------+---------------+---------------+

major version: 0xC (TACACS+)
minor version: 0x0 default, 0x1 extended (CHAP, MS-CHAP, PAP)
type:    1 = AUTHEN, 2 = AUTHOR, 3 = ACCT
seq_no:  starts at 1, client odd, server even
flags:   bit 0 = TAC_PLUS_UNENCRYPTED_FLAG (DO NOT SET in production)
         bit 2 = TAC_PLUS_SINGLE_CONNECT_FLAG
session_id: random 32-bit, identifies session
length:  payload length in bytes
```

Payload encryption (per RFC 8907 §4.5):

```
pseudo_pad = MD5(session_id || key || version || seq_no) ||
             MD5(session_id || key || version || seq_no || prev_md5) ||
             ...
ciphertext = plaintext XOR pseudo_pad

# Each MD5 produces 16 bytes; concatenate until pad >= length.
# Truncate pad to length and XOR byte-by-byte.
# Header is NEVER encrypted (must be parsed first to know length & flags).
```

Single-connection-mode:

```
If client sets TAC_PLUS_SINGLE_CONNECT_FLAG in first packet AND
   server agrees (echoes flag in reply),
the same TCP socket is reused for many AAA exchanges.
Reduces RTT. Both ends must support; Cisco IOS does, many older
servers don't.
```

## TACACS+ vs RADIUS

| Property | TACACS+ | RADIUS |
|---|---|---|
| Transport | TCP/49 | UDP/1812 (auth), UDP/1813 (acct) |
| Encryption | Full payload (XOR with MD5-keyed pad) | Only the User-Password attribute |
| AAA | Decoupled (3 independent tx) | Combined auth+authz |
| Per-command authz | Yes (cmd= in Authorization) | No (workaround via Filter-Id) |
| Multi-protocol | Yes (PPP, ARA, X.25, NASI, telnet, ssh) | Yes (PPP, EAP, 802.1X) |
| Vendor extensions | AV-pairs (string) | VSA (numeric vendor-id + sub-type) |
| Best fit | Device administration | Network access (802.1X, dial-in) |
| Stateless retry | No (TCP) | Yes (UDP) |
| Detect dead server | TCP RST/timeout | UDP timeout only |
| Standard | RFC 8907 (2020) | RFC 2865/2866 (2000) |
| Vendor support | Cisco, Juniper, Arista, F5, Palo Alto, HP/Aruba | Universal |
| Default port | 49 | 1812, 1813 (legacy 1645, 1646) |
| Accounting granularity | Per-command + session | Per-session only |

```bash
# Rule of thumb:
# - User logs into a switch/router/firewall to type commands → TACACS+
# - User connects a laptop to wifi or wired with 802.1X → RADIUS
# - Some shops run both (ISE/ClearPass do)
```

## AAA Triad

```
Authentication — Who are you?
  Verifies identity. Username/password, certificate, OTP, biometric.
  TACACS+: type=AUTHEN packet, may be START, REPLY, CONTINUE.

Authorization — What can you do?
  After auth succeeds, what privileges? Which commands? Which interfaces?
  TACACS+: type=AUTHOR packet, REQUEST and REPLY.

Accounting — What did you do?
  Audit log of every action. Login time, commands typed, bytes transferred.
  TACACS+: type=ACCT packet, REQUEST (start/watchdog/stop) and REPLY.

# In TACACS+ each leg is independent.
# You can run authentication on one server and authorization on another.
# RADIUS bundles auth+authz; only accounting is separate.
```

```
aaa new-model               # enable AAA on Cisco IOS
aaa authentication ...      # WHO
aaa authorization ...       # WHAT
aaa accounting ...          # WHEN/HOW MUCH
```

## Authentication

Login flow (ASCII / interactive):

```
Client (NAS)                    Server (tac_plus)
-----------                     -----------------
START   ────────────────────►   (action=login, authen_type=ASCII)
        ◄────────────────────   REPLY status=GETUSER, server_msg="Username:"
CONTINUE (user="alice") ──────► (sends username)
        ◄────────────────────   REPLY status=GETPASS, server_msg="Password:"
CONTINUE (user_msg="s3cr3t") ─► (sends password)
        ◄────────────────────   REPLY status=PASS|FAIL [server_msg="..."]
```

Status codes (REPLY):

```
TAC_PLUS_AUTHEN_STATUS_PASS     = 0x01   user authenticated
TAC_PLUS_AUTHEN_STATUS_FAIL     = 0x02   credentials wrong
TAC_PLUS_AUTHEN_STATUS_GETDATA  = 0x03   server wants more data
TAC_PLUS_AUTHEN_STATUS_GETUSER  = 0x04   server wants username
TAC_PLUS_AUTHEN_STATUS_GETPASS  = 0x05   server wants password
TAC_PLUS_AUTHEN_STATUS_RESTART  = 0x06   restart with different authen_type
TAC_PLUS_AUTHEN_STATUS_ERROR    = 0x07   error (network, db)
TAC_PLUS_AUTHEN_STATUS_FOLLOW   = 0x21   redirect to another server
```

authen_type field selects mechanism:

```
TAC_PLUS_AUTHEN_TYPE_ASCII   = 0x01   plain prompt (most common)
TAC_PLUS_AUTHEN_TYPE_PAP     = 0x02   send password in single CONTINUE
TAC_PLUS_AUTHEN_TYPE_CHAP    = 0x03   challenge-response (PPP)
TAC_PLUS_AUTHEN_TYPE_MSCHAP  = 0x05   Microsoft CHAP
TAC_PLUS_AUTHEN_TYPE_MSCHAPV2= 0x06   Microsoft CHAP v2
```

```bash
# IOS — ASCII login
aaa authentication login default group tacacs+ local
# group tacacs+ first; if all servers down, fall back to local user database

# Send password as PAP (mostly for PPP framing)
aaa authentication ppp default if-needed group tacacs+ local

# Enable mode (the "enable" password)
aaa authentication enable default group tacacs+ enable
# Falls back to the device's enable secret if servers down
```

## Authorization

```bash
# Enable per-command authorization on Cisco
aaa authorization exec default group tacacs+ if-authenticated
aaa authorization commands 1  default group tacacs+ if-authenticated none
aaa authorization commands 7  default group tacacs+ if-authenticated none
aaa authorization commands 15 default group tacacs+ if-authenticated none
# `if-authenticated` — pass-through if user is authenticated and server unreachable
# `none` at the end — last-resort method, allows command if all preceding methods unavailable
```

Per-service authorization — service= attribute selects context:

```
service = shell           interactive CLI (vty/console)
service = ppp             PPP framing
service = exec            Cisco exec mode (shell on most platforms)
service = junos-exec      Juniper Junos shell
service = raccess         Reverse access (asynchronous)
service = nasi            NetWare NASI
service = connection      Outbound connect-style
service = enable          enable-mode authorization
service = system          system-level functions
service = arap            AppleTalk Remote Access
service = x25             X.25 PAD
```

The AV-pair language — used in Authorization REPLY to encode results:

```
key=value   (mandatory) — must be enforced or session denied
key*value   (optional)  — applied if NAS supports/recognizes it

# Example AV-pairs returned by server:
service=shell
priv-lvl=15
idle-timeout=10
autocmd=show version
acl=110
inacl=101
outacl=102
addr=10.0.0.5
```

Each cmd= block in tac_plus.conf evaluates in order:

```
cmd = show {
    permit ".*"                  # allow any "show ..." command
}
cmd = configure {
    permit "terminal"            # allow "configure terminal"
    deny ".*"                    # deny everything else under configure
}
cmd = clear {
    permit "counters .*"
    deny ".*"
}
```

## Accounting

```
TACACS+ Accounting flags (single byte in ACCT REQUEST):
TAC_PLUS_ACCT_FLAG_START    = 0x02   session begin
TAC_PLUS_ACCT_FLAG_STOP     = 0x04   session end
TAC_PLUS_ACCT_FLAG_WATCHDOG = 0x08   interim update
```

```bash
# Cisco — exec sessions (login/logout)
aaa accounting exec default start-stop group tacacs+

# Per-command accounting at privilege level 15
aaa accounting commands 15 default start-stop group tacacs+

# Per-command at all enable levels
aaa accounting commands 1  default start-stop group tacacs+
aaa accounting commands 7  default start-stop group tacacs+
aaa accounting commands 15 default start-stop group tacacs+

# System events (reload, OIR, etc.)
aaa accounting system default start-stop group tacacs+

# Connection — outbound telnet/ssh from device
aaa accounting connection default start-stop group tacacs+

# Network — PPP/SLIP sessions
aaa accounting network default start-stop group tacacs+
```

Watchdog (interim) records:

```bash
# Send periodic interim updates while session is alive
aaa accounting update newinfo periodic 5
# `newinfo` — only when fresh info available
# `periodic 5` — every 5 minutes regardless
```

## tac_plus.conf

Top-level structure:

```
# /etc/tacacs+/tac_plus.conf

# 1) Global settings
key = "CHANGE_ME_secret"            # default shared secret for all clients
accounting file = /var/log/tac_plus.log
authorization log = /var/log/tac_plus_authz.log
default authentication = file /etc/passwd
# Other authentication backends:
#   file /etc/shadow
#   file /etc/tacacs+/passwd
#   pam
#   ldap (with tac_plus-ng)

# 2) Optional logging
logging = /var/log/tac_plus.events
# tac_plus-ng: log = stderr | syslog | file:/path
# Severity:    debug | info | notice | warning | error

# 3) Optional ACL on connecting NAS
acl = mgmt-net {
    permit = "^10\\.0\\.0\\.[0-9]+$"
    permit = "^192\\.168\\.1\\.[0-9]+$"
}

# 4) Per-host overrides
host = 10.0.0.1 {
    key  = "router1_secret"
    type = cisco
    prompt = "Password for router1: "
    enable = file /etc/tacacs+/enable.passwd
    address = 10.0.0.1
    name = "core-router-1"
}

host = 10.0.0.2 {
    key  = "router2_secret"
    type = junos              # adjusts AV-pair semantics for Juniper
}

host = 10.0.0.3 {
    key  = "router3_secret"
    type = alu                # Alcatel-Lucent semantics
}

# 5) Group definitions (must come before users that reference them)
group = read-only {
    default service = deny
    service = exec {
        default attribute = permit
        priv-lvl = 1
        idle-timeout = 15
    }
    cmd = show       { permit ".*" }
    cmd = enable     { permit ".*" }
    cmd = exit       { permit ".*" }
    cmd = ping       { permit ".*" }
    cmd = traceroute { permit ".*" }
    cmd = terminal   { permit ".*" }
}

group = network-admin {
    default service = permit
    service = exec {
        default attribute = permit
        priv-lvl = 15
        idle-timeout = 30
    }
    cmd = show       { permit ".*" }
    cmd = configure  { permit ".*" }
    cmd = interface  { permit ".*" }
    cmd = router     { permit ".*" }
    cmd = ip         { permit ".*" }
    cmd = no         { permit ".*" }
    cmd = clear      { permit ".*" }
    cmd = reload     { deny ".*"   }   # reload reserved for super-admin
}

group = super-admin {
    default service = permit
    service = exec {
        default attribute = permit
        priv-lvl = 15
    }
    # All commands permitted (default service = permit) and accounting captures everything.
}

# 6) Users
user = alice {
    member = network-admin
    login  = des "p3kZ8YxQ.fLdU"          # crypt(3) hash
    chap   = cleartext "alice-chap-pw"     # for PPP/CHAP, separate cred
    pap    = cleartext "alice-pap-pw"
    expires = "Dec 31 2026"
}

user = bob {
    member = read-only
    login = file /etc/tacacs+/passwd       # delegate to a file
}

user = scripted-svc {
    member = network-admin
    login = des "....."
    service = exec {
        autocmd = "show running-config"
        idle-timeout = 1
    }
}

# 7) Default user (catch-all) — usually deny
user = DEFAULT {
    login = nopassword
    service = exec {
        default attribute = deny
    }
}
```

login = backends:

```
login = des "<crypt-hash>"        # crypt(3) DES — legacy
login = des "$1$..."              # MD5-crypt
login = des "$5$..."              # SHA-256
login = des "$6$..."              # SHA-512 (recommended)
login = cleartext "..."           # NEVER use in prod
login = file /etc/tacacs+/passwd  # /etc/passwd-style file
login = nopassword                # auto-pass (testing only)
login = pam                       # delegate to PAM (tac_plus-pam)
login = ldap ...                  # tac_plus-ng with LDAP
```

## tac_plus.conf Reference Examples

Read-only operator group:

```
group = noc-readonly {
    default service = deny
    service = exec {
        default attribute = permit
        priv-lvl = 1
        idle-timeout = 15
    }
    cmd = show {
        deny  "running-config view full"   # hide the secret view
        permit ".*"
    }
    cmd = ping       { permit ".*" }
    cmd = traceroute { permit ".*" }
    cmd = telnet     { permit ".*" }
    cmd = ssh        { permit ".*" }
    cmd = enable     { deny ".*"   }
    cmd = configure  { deny ".*"   }
    cmd = clear      { deny ".*"   }
}
```

Network admin (level 15 with safety rails):

```
group = net-admin {
    default service = permit
    service = exec {
        default attribute = permit
        priv-lvl = 15
        idle-timeout = 30
    }
    cmd = configure { permit ".*" }
    cmd = interface { permit ".*" }
    cmd = no        { permit ".*" }
    cmd = ip        { permit ".*" }
    cmd = router    { permit ".*" }
    cmd = vlan      { permit ".*" }
    cmd = spanning-tree { permit ".*" }
    cmd = ! { permit ".*" }                 # accept comments
    # Lock out destructive commands
    cmd = reload    { deny ".*"   }
    cmd = erase     { deny ".*"   }
    cmd = format    { deny ".*"   }
    cmd = delete    { deny ".*"   }
    cmd = write     { permit "memory"
                       permit "terminal"
                       deny   ".*" }
}
```

Super-admin with full audit:

```
group = super-admin {
    default service = permit
    service = exec {
        default attribute = permit
        priv-lvl = 15
    }
    # Tight identity: this group only permitted from jumphost
    acl = jumphost-only
}

acl = jumphost-only {
    permit = "^10\\.99\\.0\\.10$"
    permit = "^10\\.99\\.0\\.11$"
}
```

Device-specific permissions via host-typed groups:

```
group = dc-only {
    default service = deny
    service = exec {
        default attribute = permit
        priv-lvl = 15
    }
    cmd = show { permit ".*" }
    cmd = vrf  { permit ".*" }
    # restrict to dc-* hosts (regex on the calling client name)
}

# In the host block:
host = 10.10.10.0/24 {        # tac_plus-ng: CIDR
    key  = "dc-secret"
    type = cisco
    name = "dc-fabric"
}
```

Time-of-day restrictions (tac_plus-ng):

```
realm = corp {
    expiration valid {
        daily 09:00 - 18:00 mon-fri
    }
}
```

Or in classic tac_plus via `time = ...` keyword (vendor-dependent).

## AV-Pair Reference (Cisco)

| AV-Pair | Where | Meaning |
|---|---|---|
| service=shell | Authz reply | request shell service |
| service=ppp | Authz reply | PPP service |
| service=exec | Authz reply | Cisco exec mode |
| protocol=ip | Authz reply | for service=ppp, IP framing |
| priv-lvl=N | Authz reply | privilege level (0-15; 15=enable) |
| cmd=command | Authz request | command being authorized |
| cmd-arg=arg | Authz request | argument to the command |
| cmd-arg=<cr> | Authz request | terminating carriage return |
| autocmd=cmd | Authz reply | command auto-executed at login |
| noescape=true | Authz reply | disable escape character |
| nohangup=true | Authz reply | don't hang up after autocmd |
| idle-timeout=N | Authz reply | minutes of idle before disconnect |
| timeout=N | Authz reply | absolute session time limit (min) |
| acl=N | Authz reply | apply numbered ACL on input |
| inacl=N | Authz reply | numbered ACL inbound on this interface |
| outacl=N | Authz reply | numbered ACL outbound |
| addr=ip | Authz reply | override source IP for session |
| addr-pool=name | Authz reply | dynamic IP pool name |
| ip-addresses=list | Authz reply | dynamic IP addresses (RAS) |
| callback-line=N | Authz reply | callback line number |
| callback-rotary=N | Authz reply | callback rotary group |
| callback-dialstring=N | Authz reply | callback dial string |
| nocallback-verify | Authz reply | skip callback verification |
| route=ip mask gw | Authz reply | static route to install |
| routing=true | Authz reply | enable routing on link |
| tunnel-id=name | Authz reply | VPDN tunnel id |
| tunnel-type=L2TP | Authz reply | tunnel protocol |
| nas-password=pw | Authz reply | password for NAS-to-LNS auth |
| gw-password=pw | Authz reply | gateway password |
| cisco-av-pair="ip:..." | Vendor | Cisco-specific extension namespace |
| cisco-av-pair="shell:roles=..." | Vendor | NX-OS RBAC roles |
| cisco-av-pair="shell:priv-lvl=15" | Vendor | priv-lvl alternate form |
| cisco-av-pair="ip:inacl#1=..." | Vendor | dynamic ACL |
| zonename=zone | Authz reply | AppleTalk zone (legacy) |
| max-links=N | Authz reply | multilink PPP max links |
| source-ip=ip | Authz reply | source-IP override (some NAS) |
| ssh-public-key=base64 | Authz reply | inject SSH key (Cumulus, others) |

Juniper-specific (Junos-exec service):

```
service = junos-exec {
    local-user-name = "admin-user"
    allow-commands = "^(show|ping|traceroute|monitor)"
    deny-commands  = "^(start shell|request system reboot)"
    allow-configuration = "interfaces.* unit.* family"
    deny-configuration  = "system root-authentication"
    allow-configuration-regexps = "interfaces.*"
    deny-configuration-regexps  = "system root-authentication"
    user-permissions = "configure view view-configuration"
}
```

```
allow-commands               regex against operational CLI command
deny-commands                regex; deny takes precedence over allow
allow-configuration          regex against config statements
deny-configuration           regex; deny wins
allow-configuration-regexps  list of regex strings (modern syntax)
deny-configuration-regexps   list
user-permissions             space-separated permission keywords
local-user-name              map to a local Junos template user
```

## Service Catalog

```
service = shell { ... }
  Interactive CLI on Cisco IOS, NX-OS (with shell:roles=), Arista EOS.
  AV-pairs: priv-lvl, autocmd, idle-timeout, timeout, noescape, nohangup,
           cmd= sub-blocks for per-command-authz.

service = ppp protocol = ip { ... }
  PPP-over-IP framing.
  AV-pairs: addr, addr-pool, inacl, outacl, route, routing, callback-*,
           idle-timeout, timeout, max-links, mtu.

service = ppp protocol = ipv6 { ... }
  PPP IPv6.

service = ppp protocol = lcp { ... }
  PPP LCP (multilink).

service = exec { ... }
  Cisco exec session — many platforms equate this to `shell`.

service = junos-exec { ... }
  Juniper-only.
  AV-pairs: local-user-name, allow-commands, deny-commands,
           allow-configuration, deny-configuration,
           allow-configuration-regexps, user-permissions.

service = raccess { ... }
  Reverse Access — used for telnet-rotary lines.

service = nasi { ... }
  NetWare Asynchronous Services Interface (legacy).

service = connection { ... }
  Outbound connection authorization (telnet from device).

service = enable { ... }
  Enable-mode authorization on Cisco.

service = system { ... }
  System events.

service = arap { ... }
  AppleTalk Remote Access (legacy).

service = x25 { ... }
  X.25 PAD (legacy).

service = ftp { ... }
  FTP authorization (used by some firewalls).
```

## Per-Command Authorization Workflow

```
User types:                show running-config

NAS (router) builds:       AUTHOR REQUEST
                           authen_method = TAC_PLUS_AUTHEN_METH_TACACSPLUS
                           priv_lvl      = 15
                           authen_type   = TAC_PLUS_AUTHEN_TYPE_ASCII
                           service       = TAC_PLUS_AUTHEN_SVC_LOGIN
                           user          = "alice"
                           port          = "tty2"
                           rem_addr      = "10.99.0.10"
                           args[0]       = "service=shell"
                           args[1]       = "cmd=show"
                           args[2]       = "cmd-arg=running-config"
                           args[3]       = "cmd-arg=<cr>"

Server checks:             user alice → group net-admin
                           cmd = show { permit ".*" }    ← matches
                           replies AUTHOR REPLY status=PASS_ADD
                                   args[0] = "priv-lvl=15"

NAS:                       executes "show running-config"
                           Then sends ACCT REQUEST flags=START
                           After completion: ACCT REQUEST flags=STOP
```

Status codes (AUTHOR REPLY):

```
TAC_PLUS_AUTHOR_STATUS_PASS_ADD    = 0x01   permit; merge AVs from reply
TAC_PLUS_AUTHOR_STATUS_PASS_REPL   = 0x02   permit; replace AVs with reply
TAC_PLUS_AUTHOR_STATUS_FAIL        = 0x10   deny
TAC_PLUS_AUTHOR_STATUS_ERROR       = 0x11   protocol/server error
TAC_PLUS_AUTHOR_STATUS_FOLLOW      = 0x21   try another server
```

## Encryption Details

```
RFC 8907 §4.5

Pad construction (iterative MD5 chain):

  hash[0] = MD5(session_id || key || version || seq_no)
  hash[i] = MD5(session_id || key || version || seq_no || hash[i-1])

  pad     = hash[0] || hash[1] || ... || hash[n]
            (concatenate until len(pad) >= len(plaintext))
  pad     = pad[0:len(plaintext)]              # truncate

  ciphertext[i] = plaintext[i] XOR pad[i]
```

```python
# Python reference encryption
import hashlib

def tacplus_pad(session_id, key, version, seq_no, length):
    pad = b""
    prev = b""
    while len(pad) < length:
        m = hashlib.md5()
        m.update(session_id.to_bytes(4, "big"))
        m.update(key.encode())
        m.update(bytes([version]))
        m.update(bytes([seq_no]))
        if prev:
            m.update(prev)
        prev = m.digest()
        pad += prev
    return pad[:length]

def encrypt(plaintext, session_id, key, version, seq_no):
    pad = tacplus_pad(session_id, key, version, seq_no, len(plaintext))
    return bytes(p ^ k for p, k in zip(plaintext, pad))
```

Security caveats:

```
- The cipher is XOR with an MD5-derived keystream; if the key leaks
  every captured packet decrypts trivially.
- Header (12 bytes) is plaintext: type, seq, session_id, length all visible.
- An attacker with a capture and a username guess can crack short passwords
  via offline brute-force on the MD5 stream.
- MD5 is broken for collision resistance but the construction here uses it
  as a PRF; the practical issue is the static key, not MD5 per se.
- Modern recommendation: rotate the shared secret quarterly.
- For high-trust deployments tunnel TACACS+ inside IPsec or TLS:
    * RFC 8907 §10.5.2: "TLS or IPsec MUST be used to protect TACACS+"
    * tac_plus-ng supports native TLS (port 4949 commonly used).
- Never set TAC_PLUS_UNENCRYPTED_FLAG (bit 0 of header flags) in production.
```

## Single Connection Mode

```
Bit 2 of the header flags field: TAC_PLUS_SINGLE_CONNECT_FLAG (0x04)

Sequence:
  1. Client opens TCP to server:49.
  2. Client sends first packet with flag set (proposing single-conn).
  3. Server replies; if it also supports it, it echoes the flag set.
  4. Subsequent AUTHEN/AUTHOR/ACCT exchanges reuse the same TCP socket
     with new session_id values.
  5. Either side may close at any time.

Benefits:
  - Eliminates 3-way handshake on every AAA event.
  - Reduces lock contention for high-volume command-authz.

Cisco IOS: enabled by default; toggle with
  tacacs server NAME
   single-connection
   no single-connection           # to disable

tac_plus (Shrubbery): supports it (TF1 — "modern" shorthand).
tac_plus-ng:           supports it natively.
```

## Cisco IOS AAA Integration

Canonical full config:

```bash
! ----- enable AAA -----
aaa new-model

! ----- TACACS+ servers -----
tacacs server TAC1
 address ipv4 10.0.0.10
 key 7 0822455D0A16   ! type-7 (reversible, weak); use service password-encryption
 single-connection
 timeout 5

tacacs server TAC2
 address ipv4 10.0.0.11
 key 7 0822455D0A16
 single-connection
 timeout 5

! ----- group the servers -----
aaa group server tacacs+ TACGRP
 server name TAC1
 server name TAC2
 ! Optional: bind to source interface (matters for VRF / multi-homed)
 ip tacacs source-interface Loopback0
 ! Optional: bind to a VRF
 ip vrf forwarding mgmt

! ----- authentication -----
aaa authentication login    default group TACGRP local
aaa authentication enable   default group TACGRP enable
! console line: separate method-list — never lose console access
aaa authentication login    CONSOLE local enable

! ----- authorization -----
aaa authorization config-commands
aaa authorization exec      default group TACGRP if-authenticated
aaa authorization commands  1  default group TACGRP if-authenticated none
aaa authorization commands  7  default group TACGRP if-authenticated none
aaa authorization commands 15 default group TACGRP if-authenticated none
aaa authorization console

! ----- accounting -----
aaa accounting exec          default start-stop group TACGRP
aaa accounting commands  1   default start-stop group TACGRP
aaa accounting commands  7   default start-stop group TACGRP
aaa accounting commands 15   default start-stop group TACGRP
aaa accounting system        default start-stop group TACGRP
aaa accounting connection    default start-stop group TACGRP
aaa accounting update newinfo periodic 5

! ----- VTY lines (SSH) -----
line vty 0 4
 transport input ssh
 login authentication default
 authorization exec default
 authorization commands 1  default
 authorization commands 7  default
 authorization commands 15 default
 accounting commands 1  default
 accounting commands 7  default
 accounting commands 15 default
 accounting exec default
 exec-timeout 15 0

line vty 5 15
 transport input ssh
 login authentication default
 authorization exec default
 authorization commands 15 default
 accounting commands 15 default
 accounting exec default
 exec-timeout 15 0

! ----- console line: local-only fallback -----
line con 0
 login authentication CONSOLE
 exec-timeout 5 0

! ----- local fallback user -----
username admin-fallback privilege 15 secret 9 $9$randomsalthash
```

```bash
! Verify configuration
show aaa servers                       ! per-server stats and reachability
show tacacs                            ! summary
show aaa method-lists all              ! every method list
show aaa user all                      ! attached AAA state per user
show aaa attributes                    ! list of supported AV-pairs

! Test a user without leaving config
test aaa group TACGRP alice s3cr3t legacy
! "User successfully authenticated" or "User authentication request failed"
```

## Cisco IOS XR

```bash
! XR uses a slightly different syntax tree
tacacs-server host 10.0.0.10 port 49
 key 7 0822455D0A16
 single-connection
!
tacacs-server host 10.0.0.11 port 49
 key 7 0822455D0A16
 single-connection
!
tacacs source-interface Loopback0 vrf default

aaa group server tacacs+ TACGRP
 server 10.0.0.10
 server 10.0.0.11
 vrf default

aaa authentication login default group TACGRP local
aaa authorization exec default group TACGRP local
aaa authorization commands default group TACGRP none
aaa accounting exec default start-stop group TACGRP
aaa accounting commands default start-stop group TACGRP
```

## Cisco NX-OS

```bash
! NX-OS requires the feature first
feature tacacs+

tacacs-server host 10.0.0.10 key 0 "secret-here"
tacacs-server host 10.0.0.11 key 0 "secret-here"
tacacs-server timeout 5
tacacs-server deadtime 10
ip tacacs source-interface mgmt0

aaa group server tacacs+ TACGRP
  server 10.0.0.10
  server 10.0.0.11
  use-vrf management
  source-interface mgmt0

aaa authentication login default group TACGRP local
aaa authentication login console local
aaa authorization commands default group TACGRP local
aaa authorization config-commands default group TACGRP local
aaa accounting default group TACGRP

! NX-OS uses cisco-av-pair="shell:roles=..." for RBAC
! Server returns:  cisco-av-pair="shell:roles=\"network-admin vdc-admin\""
```

## Juniper Junos

```bash
set system tacplus-server 10.0.0.10 secret "shared-secret"
set system tacplus-server 10.0.0.10 single-connection
set system tacplus-server 10.0.0.10 timeout 5
set system tacplus-server 10.0.0.11 secret "shared-secret"

# Auth order: try TACACS+ first, then local password file
set system authentication-order [ tacplus password ]

# Accounting
set system accounting destination tacplus
set system accounting events [ login change-log interactive-commands ]
set system accounting destination tacplus server 10.0.0.10 secret "shared-secret"

# Map remote user to local template (so Junos has a UID/GID)
set system login user remote-admin uid 9001
set system login user remote-admin class super-user
set system login user remote-admin authentication no-public-keys

# Default class for unmatched users
set system login user remote uid 9000
set system login user remote class read-only
```

Show / verify:

```
show system tacplus-server statistics
show log messages | match tacplus
```

## Arista EOS

```bash
tacacs-server host 10.0.0.10 vrf MGMT key 7 0822455D0A16
tacacs-server host 10.0.0.11 vrf MGMT key 7 0822455D0A16
tacacs-server timeout 5
ip tacacs vrf MGMT source-interface Management1

aaa group server tacacs+ TACGRP
   server 10.0.0.10 vrf MGMT
   server 10.0.0.11 vrf MGMT

aaa authentication login default group tacacs+ local
aaa authentication enable default group tacacs+ enable
aaa authorization exec default group tacacs+ local
aaa authorization commands all default group tacacs+ none
aaa accounting exec default start-stop group tacacs+
aaa accounting commands all default start-stop group tacacs+
```

## F5 BIG-IP

```bash
# tmsh — TMOS shell
tmsh create auth tacacs system-auth \
  servers add { 10.0.0.10 10.0.0.11 } \
  secret "shared-secret" \
  service ppp \
  protocol ip \
  authentication use-first-server \
  encryption enabled \
  accounting send-to-all-servers \
  debug enabled

tmsh modify auth source type tacacs
tmsh modify auth remote-user default-role admin
tmsh modify auth remote-role role-info add { \
  netadmin { attribute "F5-LTM-User-Info-1=netadmin" line-order 1 role admin user-partition All } \
}

tmsh save sys config
```

## Palo Alto

```
GUI path: Device → Server Profiles → TACACS+
  Name:           ts-prof
  Server:         10.0.0.10  Port: 49  Secret: shared-secret  Timeout: 3
  Server:         10.0.0.11  Port: 49  Secret: shared-secret  Timeout: 3
  Use single-connection: yes

Device → Authentication Profile
  Name:           ts-auth
  Type:           TACACS+
  Server Profile: ts-prof
  Domain:         (blank)

Device → Authentication Sequence
  Name:           ts-seq
  Order:          ts-auth, then local-db

Device → Admin Roles
  Map TACACS+ AV-pair  pa-vsys-role=superuser   →  Superuser

Device → Setup → Management → Authentication Settings
  Authentication Profile: ts-seq
  Idle Timeout: 60
```

CLI equivalent:

```
configure
set shared server-profile tacplus ts-prof server "ts1" address 10.0.0.10
set shared server-profile tacplus ts-prof server "ts1" secret "shared-secret"
set shared server-profile tacplus ts-prof use-single-connection yes
set shared authentication-profile ts-auth method tacplus server-profile ts-prof
commit
```

## Linux Login via TACACS+

`/etc/tacplus.conf` — pam_tacplus library config:

```
# global library config — read by libpam-tacplus, libnss-tacplus
secret=shared-secret-here
server=10.0.0.10
server=10.0.0.11
timeout=5
debug=true
login=login
service=ppp
protocol=ip
# optional: bind source IP
source_ip=10.0.0.50
```

`/etc/pam.d/sshd` (Debian-style):

```
# Authenticate via TACACS+ first, fall back to local
auth        sufficient   pam_tacplus.so server=10.0.0.10 secret=shared-secret service=ppp protocol=ip login=login try_first_pass
auth        sufficient   pam_tacplus.so server=10.0.0.11 secret=shared-secret service=ppp protocol=ip login=login try_first_pass
auth        required     pam_unix.so   nullok try_first_pass

# Authorization phase — also via TACACS+
account     sufficient   pam_tacplus.so server=10.0.0.10 secret=shared-secret service=ppp protocol=ip
account     sufficient   pam_tacplus.so server=10.0.0.11 secret=shared-secret service=ppp protocol=ip
account     required     pam_unix.so

# Accounting via TACACS+
session     optional     pam_tacplus.so server=10.0.0.10 secret=shared-secret service=ppp protocol=ip
session     required     pam_unix.so
```

`/etc/pam.d/login` — same structure. `/etc/pam.d/sudo` for sudo invocations.

`/etc/nsswitch.conf` — for tacplus-mapped users:

```
passwd:  files tacplus
group:   files
shadow:  files tacplus
```

Verify:

```bash
# Watch live
sudo journalctl -u ssh -f
# pam_tacplus log lines
sudo journalctl -t pam_tacplus -f
# Or grep auth.log
sudo tail -F /var/log/auth.log | grep -i tacplus

# Test connectivity to server
nc -vz 10.0.0.10 49
sudo tcpdump -i any -nn tcp port 49
```

Sample log:

```
sshd[12345]: pam_tacplus(sshd:auth): user [alice] logged in via TACACS+
sshd[12345]: pam_tacplus(sshd:account): tac_authorize for [alice] OK
sshd[12345]: pam_tacplus(sshd:session): START accounting sent
sshd[12345]: Accepted password for alice from 10.99.0.10 port 51234 ssh2
sshd[12345]: pam_unix(sshd:session): session opened for user alice(uid=9001) ...
```

## Common Deployment Topology

```
             +--------------------+
             |  Identity source   |
             |  (AD / LDAP / DB)  |
             +---------+----------+
                       |
        +--------------+--------------+
        |              |              |
   +----v-----+   +----v-----+   +----v-----+
   | TAC1     |   | TAC2     |   | TAC3     |
   | primary  |   | secondary|   | tertiary |
   | 10.0.0.10|   | 10.0.0.11|   | 10.0.0.12|
   +----+-----+   +----+-----+   +----+-----+
        |              |              |
        +--------------+--------------+
                       |
              +--------v--------+
              | NAS clients     |
              | (routers, fws,  |
              |  switches)      |
              +-----------------+

Typical: 2-3 servers in a cluster
- Active-Active: clients pick primary; failover after timeout
- Active-Standby: secondary only used after dead-time on primary
- Per-region pairs to localize latency
- All synchronize from a central identity store (AD/LDAP)
```

## HA Patterns

```
1. Round-robin via DNS — DON'T DO IT.
   Reason: TACACS+ sessions are stateful and clients expect deterministic
   server selection during a multi-packet exchange.

2. Explicit primary/secondary in client config.
   - Cisco: order in `aaa group server tacacs+ TACGRP` matters.
   - Junos: order of `set system tacplus-server` lines.

3. Failover timing.
   - timeout per request:        3-5 seconds
   - dead-time after N timeouts: 10-15 minutes (skip dead server entirely)
   - keep enough servers so that if primary dies you reach secondary
     before user gives up

4. Always-have-local-fallback rule.
   - aaa authentication login default group TACGRP local
                                                      ^^^^^
                                                      critical
   - One local username with a randomly-generated password stored in vault
   - Console line uses a SEPARATE method list that is local-only

5. Dead detection on Cisco
   tacacs-server dead-criteria time 5 tries 3
   tacacs-server deadtime 10
   ! mark a server dead after 3 timeouts within 5s
   ! skip it for 10 minutes

6. VRF-aware reachability.
   - TACACS+ traffic on management VRF
   - aaa group server with `ip vrf forwarding` matching
   - Source-interface bound to a stable loopback inside that VRF
```

## Shared Secret Rotation

```
Procedure (zero-downtime):

1. Generate new secret. Use 32+ random chars; avoid shell metachars
   if possible.
       openssl rand -base64 32

2. Pre-load on TACACS+ servers FIRST.
   tac_plus.conf supports per-host overrides; you can have a transition
   period where each host has both old and new secret? — no, tac_plus has
   one key per host. So:

   a. Add new host stanza with new key while still permitting old.
      Some tac_plus-ng builds allow `key = "new" "old"` (key list);
      classic tac_plus does NOT.
   b. If your tac_plus version doesn't support key lists:
      — schedule maintenance window
      — change clients to new key in batches matching server change
   c. Some shops run a parallel TACACS+ instance on a second port
      with new key during transition.

3. Update clients.
       Cisco:   tacacs server TAC1 / key 7 <new-encrypted>
       Junos:   set system tacplus-server 10.0.0.10 secret "<new>"
       Arista:  tacacs-server host 10.0.0.10 key 7 <new>

4. Verify with `test aaa group ...` and tcpdump (look for non-rejected sessions).

5. Remove old key from servers.

Cadence: rotate quarterly at minimum; immediately on suspected leak,
employee departure, or any audit finding.

Storage: never commit secrets to git. Use HashiCorp Vault, AWS Secrets
Manager, Ansible Vault, or sops. Inject at config-generation time.

Cisco type-7 password-encryption is REVERSIBLE (Vigenère cipher):
       service password-encryption           ! type-7 — weak
       password encryption aes               ! type-6 — uses primary key
       key config-key password-encrypt       ! sets primary key for type-6
On modern IOS use type-6 for the TACACS+ key when possible.
Type-9 (scrypt) is for local user passwords — not applicable to keys.
```

## The Lockout Disaster Pattern

```
Failure mode:

  aaa authentication login default group TACGRP
                                                  ^^^ no local fallback
  line con 0
   login authentication default
  line vty 0 4
   login authentication default

  -> All TACACS+ servers go unreachable
  -> Console asks for credentials -> tries TACACS+ -> fails
  -> No local user -> no way in
  -> Truck-roll to ROMMON / password recovery -> service outage

Fix:

  aaa authentication login default group TACGRP local
                                                ^^^^^
  username breakglass privilege 15 secret 9 $9$randomsalthash

  ! console line gets its OWN method list, local-only
  aaa authentication login CONSOLE local
  line con 0
   login authentication CONSOLE
   exec-timeout 5 0
  line vty 0 15
   login authentication default

  ! Bonus: management VRF for TACACS+ traffic so a data-plane outage
  !        doesn't brick AAA
  ip tacacs source-interface Loopback99
  vrf definition mgmt
  aaa group server tacacs+ TACGRP
   ip vrf forwarding mgmt
   server name TAC1

Quarterly drills:
  - Disable TACACS+ in a maintenance window and verify breakglass works.
  - Rotate breakglass password. Store in vault, audit access.
  - Verify console hardware works (serial cable in correct port, baud).
```

## Logging

```
TACACS+ accounting log fields (typical tac_plus.log line):

Apr 25 14:23:15  10.99.0.10  alice  tty2  10.99.0.10  start    task_id=12  service=shell
Apr 25 14:23:18  10.99.0.10  alice  tty2  10.99.0.10  update   task_id=12  cmd=show running-config <cr>
Apr 25 14:23:55  10.99.0.10  alice  tty2  10.99.0.10  stop     task_id=12  service=shell  elapsed_time=40

Fields (space-separated by default; configurable):
  timestamp
  NAS-IP
  username
  port (tty/vty/console)
  remote-addr (where the user came from)
  record-type (start | update | stop)
  task_id (correlator across start/update/stop)
  service / cmd / cmd-arg AV-pairs
  elapsed_time (in stop record)
  status (in command-authz logs: succeed/fail)
```

Custom log format (tac_plus-ng):

```
log accounting {
    destination = file:/var/log/tac_plus/acct.log
    format = "{date} {nas} {user} {service} {cmd}"
}
```

Forward to syslog:

```bash
# /etc/rsyslog.d/tac_plus.conf
$ModLoad imfile
$InputFileName /var/log/tac_plus.log
$InputFileTag tac_plus:
$InputFileStateFile stat-tac_plus
$InputFileSeverity info
$InputFileFacility local6
$InputRunFileMonitor

local6.*  @@siem.example.com:6514
```

SIEM correlation patterns:

```
- Same user logged in from N different NAS IPs in M minutes (lateral movement)
- enable-mode usage outside business hours
- "configure terminal" followed by route changes by non-network-admin
- failed authentications spike (brute-force)
- breakglass user logged in (page on-call immediately)
- accounting STOP without START (NAS reboot or session lost)
```

## Common Errors

```
%TAC+: TACACS+ server not responding
    Server unreachable, port 49 blocked, or wrong IP.
    Check:  show tacacs ; show aaa servers
            ping <server>
            telnet <server> 49
            ACL between NAS and server
            VRF (source-interface)

%TAC+: shared key with X bad
    Wrong shared secret. The packet decrypts to garbage so the parser fails.
    Check:  the `key` line on the device matches `key` line in tac_plus.conf
            beware of escaped quotes, trailing whitespace, type-7 vs type-6

%AAA-3-SERVERREJECTED: Authentication failed for user X
    Server returned FAIL. Bad password, expired user, account locked,
    user not in any group, group has `default service = deny` and no
    matching service block.
    Check:  tac_plus.conf user/group definitions
            tac_plus -d 8 trace
            password backend (file/PAM/LDAP)

% Authorization failed
    User authenticated but no service block matched.
    Check:  user has `member = G` and group G has `service = exec`
            default service = permit  or specific service blocks present
            on Cisco the keyword `if-authenticated` may be missing causing
            secondary failure when servers unavailable

% Authorization for cmd 'X' is denied
    Per-command authorization rejected the command.
    Check:  cmd = X { ... } block in tac_plus.conf
            regex matches what NAS sent (note trailing space and <cr>)
            permit/deny order — first match wins

% Accounting failed for user X
    Accounting record could not be sent or server returned ERROR.
    Check:  tac_plus.conf accounting file is writable
            disk full
            single-connect TCP got reset mid-session

ERROR: tac_plus.conf line N: syntax error
    Config parse failure on startup.
    Check:  tac_plus -P -C /etc/tacacs+/tac_plus.conf  (parse-and-exit)
            unmatched braces { }
            missing quotes around strings with spaces
            user/group ordering — group must be defined before user references it

no host found
    `host = X { ... }` block uses a name not resolvable.
    Use IP literal, or ensure DNS/hosts has the name.

tac_plus: connection refused
    Firewall (iptables, NAS ACL, security group) blocks 49/tcp.
    Or tac_plus daemon is not running.
    Check:  ss -tlnp | grep :49
            systemctl status tacacs_plus
            iptables -L -n -v | grep 49

TACACS+ Authentication Failed
    Generic — can come from any device. Check device-specific debug:
            debug aaa authentication
            debug tacacs
            debug tacacs packet
    Look for sequence: START -> REPLY (GETPASS) -> CONTINUE -> REPLY (FAIL).

%TAC-3-SOCKETCLOSE: Socket close on connection to server
    Server closed TCP. Possibly:
      - server crashed
      - shared secret wrong (server gives up)
      - single-connect mismatch (server doesn't speak it)

% Bad secrets
    Some platforms emit this instead of "shared key bad."

ERROR: PAP login attempt with no password
    PAP authentication arrived without a password field. Often means client
    is misconfigured (sending CHAP frames to PAP-only server).

malformed packet
    Header length mismatch with payload, or post-decrypt parse error.
    Almost always the shared secret or VRF source-IP mismatch.
```

## Common Gotchas

Broken / Fixed pairs:

```
[broken]
  aaa authentication login default group tacacs+
  ! No local fallback. Servers down -> console rejects everyone -> on-site truck roll.
[fixed]
  aaa authentication login default group tacacs+ local
  username breakglass privilege 15 secret 9 $9$random
  aaa authentication login CONSOLE local
  line con 0
   login authentication CONSOLE

[broken]
  tacacs server TAC1
   address ipv4 10.0.0.10
   key cleartext "p@ssw0rd!$"
  ! Shell-evaluated $ in some config-deploy pipelines becomes empty.
[fixed]
  tacacs server TAC1
   address ipv4 10.0.0.10
   key cleartext "p@ssw0rd\\\!\\\$"          ! escape, OR
   key 7 0822455D0A165843...                  ! pre-encrypted form
  ! Verify on both sides byte-for-byte; tac_plus.conf needs the same string.

[broken]
  ip tacacs source-interface Loopback0       ! global VRF
  aaa accounting commands 15 default start-stop group TACGRP
  ! TACACS+ auth uses mgmt VRF, accounting uses default VRF — half traffic missing.
[fixed]
  vrf definition mgmt
  ip tacacs source-interface Loopback0 vrf mgmt
  aaa group server tacacs+ TACGRP
   ip vrf forwarding mgmt
   server name TAC1

[broken]
  user = alice {
      member = net-admin
      ...
  }
  group = net-admin { ... }
  ! Group defined AFTER user — many tac_plus parsers reject this.
[fixed]
  group = net-admin { ... }                   ! group first
  user = alice { member = net-admin }

[broken]
  cmd = show {
      permit "running-config"
  }
  ! User types "show running" and gets rejected — regex too strict.
[fixed]
  cmd = show {
      permit "running-config.*"
      permit "ip route.*"
      permit "interfaces.*"
      ! Or be permissive with default permit and deny only sensitive:
      permit ".*"
      deny "running-config view full"
  }

[broken]
  aaa authorization commands 15 default group TACGRP
  ! No `if-authenticated` and no `none`. If TACGRP is unreachable:
  ! the user has no way to issue any command — even after authentication.
[fixed]
  aaa authorization commands 15 default group TACGRP if-authenticated none

[broken]
  tacacs-server timeout 1
  ! 1-second timeout flips servers dead during normal congestion.
[fixed]
  tacacs-server timeout 5
  tacacs-server dead-criteria time 5 tries 3
  tacacs-server deadtime 10

[broken]
  line vty 0 4
   ! no `login authentication default` — falls back to `login local`
   ! which checks the local password (often unset) -> permit-any.
[fixed]
  line vty 0 4
   login authentication default
   authorization exec default
   authorization commands 15 default

[broken]
  ! NAS in DC, TACACS+ in mgmt subnet, but no ACL permit for 49/tcp
  ! Symptom: %TAC+: TACACS+ server not responding intermittently.
[fixed]
  ip access-list extended ACL-TO-TAC
   permit tcp any host 10.0.0.10 eq tacacs
   permit tcp any host 10.0.0.11 eq tacacs

[broken]
  ! Using tac_plus-ng config syntax with classic tac_plus binary
  realm = corp { ... }
  ! Errors: "syntax error: realm" — classic tac_plus has no realm keyword
[fixed]
  ! Either:
  !   - use tac_plus-ng binary (event-driven-servers/tac_plus-ng)
  !   - rewrite using classic syntax (host blocks, no realms)

[broken]
  aaa authorization commands 1 default group TACGRP if-authenticated none
  aaa authorization commands 15 default group TACGRP if-authenticated none
  ! Missing level 7 — junior admins escalate to 7 with "enable 7", then
  ! all their commands skip authorization entirely.
[fixed]
  aaa authorization commands 1  default group TACGRP if-authenticated none
  aaa authorization commands 7  default group TACGRP if-authenticated none
  aaa authorization commands 15 default group TACGRP if-authenticated none

[broken]
  service password-encryption                 ! type-7
  tacacs server TAC1
   key 7 060506324F41
  ! `show running-config` reveals decryptable secret to anyone with
  ! show-cmd authority.
[fixed]
  password encryption aes                     ! type-6
  key config-key password-encrypt 9w8e7r6t5y4u3i2o1pAesDeVKey
  tacacs server TAC1
   key 6 ABCDEF...                            ! type-6 (AES-128)
  ! Plus restrict show running-config to super-admin via cmd= block.
```

## Diagnostic Tools

```bash
# Cisco: real-time AAA tracing
debug tacacs                       # high-level
debug tacacs packet                # packet-level (verbose)
debug aaa authentication
debug aaa authorization
debug aaa accounting
terminal monitor                   ! send debug to current vty
no debug all                       ! ALWAYS turn off when done

show aaa servers                   # per-server stats, dead/alive
show aaa method-lists all
show tacacs                        # summary
show tacacs counters
show users                         # active sessions
show ip tacacs                     # source-interface, VRF

# tac_plus daemon
sudo tac_plus -G -d 8  -C /etc/tacacs+/tac_plus.conf
# Debug bitmask:
#    1   = parser
#    8   = AUTHEN
#   16   = AUTHOR
#   32   = ACCT
#   64   = config
#  128   = packets (binary dump)
#  256   = encryption / pad chain
#  512   = lock
# 1024   = regex
# OR them: -d 248 = AUTHEN+AUTHOR+ACCT+packets

# Network capture
sudo tcpdump -i any -nn -X tcp port 49
# -X dumps hex+ASCII; payload is encrypted but header is readable:
#   first byte 0xC0 = TACACS+ v12.0
#   second byte 0x01 (AUTHEN), 0x02 (AUTHOR), 0x03 (ACCT)
sudo tshark -i any -f "tcp port 49" -V
# Wireshark has a TACACS+ dissector that decodes the header and (with
# the shared secret entered in Preferences->Protocols->TACACS+) the payload.

# pam_tacplus debug
# /etc/tacplus.conf:  debug=true
sudo journalctl -t pam_tacplus -f

# Quick connectivity test
nc -vz 10.0.0.10 49

# Test a credential without leaving Cisco
test aaa group TACGRP alice <password> legacy
test aaa group TACGRP alice <password> new-code
```

## ISE / Cisco Secure ACS

```
Cisco Secure ACS — End of Sale 30 Aug 2017, End of Support 31 Aug 2020.
  Replaced by Cisco Identity Services Engine (ISE).
  License:  base, plus, apex, device-admin (TACACS+ requires Device Admin).

ISE nodes:
  PAN  — Primary Admin
  SAN  — Secondary Admin
  MnT  — Monitoring & Troubleshooting
  PSN  — Policy Service Node (does the actual TACACS+ work)

Workflow (TACACS+ Device Admin in ISE):
  1. Enable Device Administration license on PSN.
  2. Add Network Device — IP, name, shared secret.
  3. Define TACACS+ Profile — AV-pairs to return.
  4. Define TACACS+ Command Set — list of permit/deny commands.
  5. Build Policy Set — Authentication, Authorization, Accounting.
       Authentication: AD/LDAP/internal-users
       Authorization:  match conditions -> assign Profile + Command Set
       Accounting:     enabled by default in TACACS+ Device Admin

GUI-driven configuration; full audit trail in MnT node.

Competitors:
  Aruba ClearPass         — strong wireless + RADIUS, supports TACACS+
  Microsoft NPS           — RADIUS only, no native TACACS+
  Pulse Secure            — broader access mgmt
  Open-source: tac_plus + tac_plus-ng + Auth0/LDAP frontend
```

## tac_plus-ng

Modern fork (event-driven-servers org):

```
Improvements over Shrubbery tac_plus:
  - Event-driven (libev), single-threaded, scales to many connections
  - IPv6 native
  - mTLS option (RFC 8907 §10.5.2 compliance)
  - LDAP, RADIUS, PAM auth backends pluggable
  - Real-time config reload without restart (HUP signal handling)
  - Better config grammar (supports realm, group inheritance, includes)
  - Per-realm key, ACL, and policy
  - JSON logging output option
  - IPv6 ACLs
  - More AV-pair types validated against schema
  - Active maintenance (Shrubbery tac_plus last release F4.0.4.28 in 2017)

Sample tac_plus-ng config:

  id = tac_plus-ng

  log = stderr {
      destination = file:/var/log/tac_plus-ng.log
      format = "{date} {tag} {msg}"
  }

  realm = corp {
      key = "shared-secret"
      address = 0.0.0.0/0
      authentication { ... }
      authorization  { ... }
  }

  user backend = ldap {
      uri = ldaps://ldap.corp.example.com
      base = "ou=People,dc=example,dc=com"
      bind-dn = "cn=tacacs,ou=Service,dc=example,dc=com"
      bind-pass = file:/etc/tac_plus-ng/ldap.pass
  }

  group net-admin {
      members = "alice", "bob"
      cmd { permit ".*" }
      service = shell { priv-lvl = 15 }
  }
```

## RFC 8907 Modern Compliance

```
RFC 8907 — "The Terminal Access Controller Access-Control System Plus
            (TACACS+) Protocol" (September 2020) — Standards Track

Key normative changes over the historical draft (RFC 1492 + Cisco draft):

  §4.5  Obfuscation MUST be applied (the XOR-MD5 cipher) — encryption
        flag MUST NOT be set in production.
  §10.5 Security considerations:
        "The obfuscation provided by TACACS+ is insufficient for the
         security of devices that hold high-value secrets."
  §10.5.2 RECOMMENDS deployment behind a transport with confidentiality
          and integrity:
            - IPsec tunnel (IKEv2)
            - TLS — implementations MAY listen on a dedicated port for TLS
                    (commonly 4949)
  §10.5.6 Shared key rotation MUST be possible without downtime.

Backward compatibility:
  - On-the-wire format unchanged.
  - Existing clients/servers interoperate.
  - mTLS-aware deployments live alongside legacy XOR-only deployments.

Deprecated:
  - TAC_PLUS_UNENCRYPTED_FLAG MUST NOT be set; servers MUST drop such packets.
  - MD4-based custom hashes (some early implementations) — not in RFC 8907.

Compliance checklist:
  [ ] All shared secrets >= 16 random bytes
  [ ] No plaintext-flag packets observed
  [ ] Quarterly key rotation
  [ ] TLS or IPsec wrapper in non-trusted networks
  [ ] Single-connection-mode supported
  [ ] Audit trail for every priv-lvl 15 command (accounting commands 15)
  [ ] Local breakglass user defined and rotated quarterly
  [ ] Console line uses local-only method list
```

## Idioms

```
"Always have a local fallback."
    aaa authentication login default group TACGRP local

"Console + VTY use separate AAA method lists."
    line con 0 / login authentication CONSOLE   ! local-only
    line vty 0 15 / login authentication default ! TACACS+ first

"Rotate keys quarterly. Always."
    Calendar reminder. Document procedure. Drill it.

"Audit every privileged command."
    aaa accounting commands 15 default start-stop group TACGRP
    Forward to SIEM. Retain >= 1 year.

"Use group inheritance for role-based perms."
    group = net-admin { member = base }
    group = base      { cmd = show { ... } cmd = ping { ... } }

"VPN/IPsec the TACACS+ link."
    Especially across WAN, internet, untrusted DMZ.
    XOR cipher is obfuscation, not encryption — assume nation-state can break it.

"Management-VRF the TACACS+ traffic."
    Independent of data-plane outages.
    Separate routing table, independent default gateway, isolated ACLs.

"Test with `test aaa group ...` after every change."
    Validates auth path without leaving config mode.

"Quarterly DR drill: kill TACACS+, verify breakglass works."

"Shared secret entropy >= 128 bits."
    openssl rand -base64 32

"One method list per role."
    Don't reuse `default` across console, vty, http-mgmt, http-admin.

"Don't trust type-7 — it is reversible."
    service password-encryption is type-7. Use type-6 for keys.

"Never `aaa authorization commands` without `if-authenticated none`."
    Otherwise unreachable server = no commands at all.

"Single-connection-mode unless your server doesn't support it."
    Cuts AAA latency dramatically under load.

"Never expose port 49 to the internet."
    Even with strong key, the protocol's confidentiality is fragile.
```

## See Also

- kerberos
- radius
- ssh
- ssh-trusted-platforms
- openssl
- vault

## References

- RFC 8907 — The Terminal Access Controller Access-Control System Plus (TACACS+) Protocol — https://www.rfc-editor.org/rfc/rfc8907
- RFC 1492 — An Access Control Protocol, Sometimes Called TACACS (informational, historical) — https://www.rfc-editor.org/rfc/rfc1492
- Shrubbery Networks tac_plus — http://www.shrubbery.net/tac_plus/
- tac_plus-ng (event-driven-servers) — https://github.com/event-driven-servers/tac_plus-ng
- pam_tacplus — https://github.com/kravietz/pam_tacplus
- libnss-tacplus — https://github.com/daveolson53/libnss-tacplus
- Cisco IOS AAA Configuration Guide — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_usr_aaa/configuration/15-mt/sec-usr-aaa-15-mt-book.html
- Cisco IOS XR AAA Commands — https://www.cisco.com/c/en/us/support/ios-nx-os-software/ios-xr-software/products-command-reference-list.html
- Cisco NX-OS AAA Configuration — https://www.cisco.com/c/en/us/td/docs/switches/datacenter/sw/security/configuration_guide/b_Cisco_Nexus_Security_Configuration_Guide.html
- Cisco Identity Services Engine (ISE) — https://www.cisco.com/c/en/us/products/security/identity-services-engine/index.html
- Juniper Junos OS Authentication and Authorization — https://www.juniper.net/documentation/us/en/software/junos/user-access/index.html
- Arista EOS User Manual — TACACS+ — https://www.arista.com/en/um-eos/eos-aaa
- F5 BIG-IP TMOS: Implementations — TACACS+ — https://techdocs.f5.com/
- Palo Alto Networks PAN-OS Administrator's Guide — Authentication — https://docs.paloaltonetworks.com/
- Wireshark TACACS+ dissector — https://wiki.wireshark.org/TACACS
- IANA Service Name and Transport Protocol Port Number Registry (port 49) — https://www.iana.org/assignments/service-names-port-numbers/
