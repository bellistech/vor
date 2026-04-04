# TACACS+ (Terminal Access Controller Access-Control System Plus)

TCP-based AAA protocol (RFC 8907) providing separate authentication, authorization, and accounting services for network device management. TACACS+ encrypts the entire packet body (unlike RADIUS which only encrypts the password), making it the standard choice for controlling administrative access to routers, switches, and firewalls.

---

## Protocol Fundamentals

### Architecture and Packet Structure

```bash
# Client-server model: network device (client) -> TACACS+ daemon (server)
# TCP port 49 (reliable, unlike RADIUS UDP)
# Full packet body encryption (shared secret + MD5 pad)
# Separates authentication, authorization, and accounting
# Session-oriented with interactive prompt support

# Packet header (12 bytes, ALWAYS unencrypted):
# Major=0xC | Minor | Type(1=authen,2=author,3=acct) | Seq_no
# Flags(0x01=unencrypted, 0x04=single-connect) | Session_ID(32-bit random)
# Length (of encrypted body)

# Encryption: body XOR MD5-based pseudo-pad
# pad_1 = MD5(session_id || key || version || seq_no)
# pad_n = MD5(session_id || key || version || seq_no || pad_{n-1})
# Each packet has unique pad (different seq_no)
```

## Authentication

### Authentication Flow Types

```bash
# ASCII — Interactive login (multi-step prompts)
#   START -> GETUSER -> CONTINUE(user) -> GETPASS -> CONTINUE(pass) -> PASS/FAIL

# PAP — Single exchange
#   START(username+password) -> PASS/FAIL

# CHAP — Challenge-response
#   START(username+CHAP_response) -> PASS/FAIL

# MS-CHAPv2 — Microsoft CHAP v2
#   START(username+MSCHAP_response) -> PASS/FAIL

# Status codes:
# PASS=0x01  FAIL=0x02  GETDATA=0x03  GETUSER=0x04
# GETPASS=0x05  RESTART=0x06  ERROR=0x07  FOLLOW=0x21
```

## Authorization

### Per-Command Authorization

```bash
# Authorization is SEPARATE from authentication
# Device sends authorization request for each user action

# Shell command authorization attributes:
#   service=shell  cmd=show  cmd-arg=running-config  cmd-arg=<cr>
#   priv-lvl=15

# Network access authorization:
#   service=ppp  protocol=ip  addr=10.0.0.0/24
#   inacl=ACL_NAME  outacl=ACL_NAME  timeout=3600

# Response status:
# PASS_ADD=0x01 (pass, add attributes)
# PASS_REPL=0x02 (pass, replace attributes)
# FAIL=0x10  ERROR=0x11  FOLLOW=0x21
```

## Accounting

### Start/Stop/Watchdog Records

```bash
# Accounting tracks user actions after authentication/authorization
# Record flags: START=0x02, STOP=0x04, WATCHDOG=0x08

# Common attributes:
#   task_id=42  start_time=1680000000  stop_time=1680003600
#   elapsed_time=3600  service=shell  priv-lvl=15
#   cmd=configure terminal  bytes_in=1024  bytes_out=2048

# Command accounting — each CLI command logged individually:
# ACCT_REQUEST(START|STOP, cmd=show running-config) -> ACCT_REPLY(SUCCESS)
```

## Cisco IOS Configuration

### Full AAA Setup

```bash
# Define TACACS+ servers
tacacs server TAC-PRIMARY
 address ipv4 10.0.1.100
 key 0 MySharedSecret123!
 timeout 5
 single-connection

tacacs server TAC-SECONDARY
 address ipv4 10.0.1.101
 key 0 MySharedSecret123!
 timeout 5

aaa group server tacacs+ TAC-SERVERS
 server name TAC-PRIMARY
 server name TAC-SECONDARY

# Enable AAA
aaa new-model
aaa authentication login default group TAC-SERVERS local
aaa authentication login CONSOLE local          # local-only console fallback
aaa authentication enable default group TAC-SERVERS enable
aaa authorization exec default group TAC-SERVERS local
aaa authorization commands 15 default group TAC-SERVERS local
aaa accounting exec default start-stop group TAC-SERVERS
aaa accounting commands 15 default start-stop group TAC-SERVERS

# Apply to lines
line vty 0 15
 login authentication default
line con 0
 login authentication CONSOLE              # emergency local access
```

### Cisco NX-OS

```bash
feature tacacs+
tacacs-server host 10.0.1.100 key MySharedSecret123!
tacacs-server host 10.0.1.101 key MySharedSecret123!
tacacs-server timeout 5

aaa group server tacacs+ TAC-SERVERS
 server 10.0.1.100
 server 10.0.1.101
 use-vrf management
 source-interface mgmt0

aaa authentication login default group TAC-SERVERS local
aaa authorization commands default group TAC-SERVERS local
aaa accounting default group TAC-SERVERS
```

## Other Vendor Configuration

### Arista EOS and Juniper JunOS

```bash
# Arista EOS
tacacs-server host 10.0.1.100 key MySharedSecret123!
tacacs-server host 10.0.1.101 key MySharedSecret123!
aaa group server tacacs+ TAC-SERVERS
 server 10.0.1.100
 server 10.0.1.101
aaa authentication login default group TAC-SERVERS local
aaa authorization commands all default group TAC-SERVERS local
aaa accounting commands all default start-stop group TAC-SERVERS

# Juniper JunOS
set system tacplus-server 10.0.1.100 secret MySharedSecret123!
set system tacplus-server 10.0.1.100 timeout 5
set system tacplus-server 10.0.1.100 single-connection
set system tacplus-server 10.0.1.101 secret MySharedSecret123!
set system authentication-order [tacplus password]
set system tacplus-options service-name junos-exec
set system tacplus-options strict-authorization
set system accounting events interactive-commands
set system accounting destination tacplus server 10.0.1.100
```

## TACACS+ Server Setup

### tac_plus Configuration

```bash
# Install: sudo apt install tacacs+
# /etc/tacacs+/tac_plus.conf
key = "MySharedSecret123!"
accounting file = /var/log/tacacs/accounting.log

host = 10.0.0.0/24 {
    key = "MySharedSecret123!"
}

group = network-admin {
    default service = permit
    service = exec { priv-lvl = 15 }
}

group = network-operator {
    default service = deny
    service = exec { priv-lvl = 1 }
    cmd = show { permit .* }
    cmd = ping { permit .* }
    cmd = traceroute { permit .* }
}

user = admin {
    member = network-admin
    login = des $1$abc$hashedpasswordhere
}

user = operator {
    member = network-operator
    login = des $1$def$hashedpasswordhere
}

user = DEFAULT {
    service = deny
}

# Start daemon
sudo systemctl enable tacacs_plus
sudo systemctl start tacacs_plus
```

## TACACS+ vs RADIUS

### Comparison

```bash
# Feature              | TACACS+           | RADIUS
# ---------------------|-------------------|------------------
# Transport            | TCP (port 49)     | UDP (1812/1813)
# Encryption           | Full body         | Password only
# AAA separation       | Yes               | Combined
# Cmd authorization    | Yes (per-command)  | Limited
# Interactive auth     | Yes (multi-step)   | No
# EAP support          | No                | Yes
# CoA support          | No                | Yes (RFC 5176)
# Typical use          | Device admin      | Network access

# Rule of thumb:
# SSH/console to routers/switches -> TACACS+
# 802.1X / VPN / wireless access  -> RADIUS
# Both needed? Use both.
```

## Cisco ISE Integration

### Device Administration Workflow

```bash
# Cisco ISE provides GUI-based TACACS+ with policy engine:
# 1. Enable Device Administration (Administration > Deployment)
# 2. Add Network Devices with TACACS+ shared secrets
# 3. Create Shell Profiles (privilege levels, auto-commands)
# 4. Create Command Sets (permit/deny specific commands)
# 5. Define Policy Sets matching user/group to profiles
# 6. Authentication: internal users, Active Directory, LDAP
# 7. Authorization: conditional rules per device type/location
```

## Troubleshooting

### Debug and Packet Capture

```bash
# Cisco IOS debug
debug tacacs authentication
debug tacacs authorization
debug tacacs accounting
show tacacs
show aaa servers
show aaa sessions
test aaa group TAC-SERVERS admin password123 new-code

# Common failures:
# 1. Key mismatch — timeout, no response; verify key on both sides
# 2. TCP 49 blocked — test: telnet 10.0.1.100 49
# 3. Source interface — set ip tacacs source-interface Loopback0
# 4. No local fallback — locked out when server down; add "local" to method list
# 5. Single-connection issues — disable or increase server threads

# Packet capture
tcpdump -i eth0 -nn tcp port 49 -w /tmp/tacacs.pcap
tshark -r /tmp/tacacs.pcap -Y "tcp.flags.reset==1"  # check for resets
```

---

## Tips

- Always include `local` as fallback in AAA method lists; without it, a server outage locks you out of every device.
- Use different shared secrets per device or device group; a single compromised key should not expose the entire network.
- Enable command accounting on all devices; the audit trail of who-ran-what-when is invaluable for incident response.
- Configure `single-connection` mode to reuse one TCP session for multiple AAA exchanges, reducing overhead.
- Test failover regularly by shutting down the primary server; verify devices fall to secondary within timeout.
- Set 5-second timeout per server; the default 10-second timeout means 20+ seconds of delay during failover.
- Keep a local emergency account with privilege 15 on every device, authenticated locally.
- Encrypt shared secrets in config with `service password-encryption` or type-8 passwords.
- Log accounting to both local syslog and centralized SIEM for compliance and change detection.
- Use group-based authorization with role separation: admin (full), operator (show only), security (audit only).

---

## See Also

- radius, ldap, kerberos, ssh

## References

- [RFC 8907 — The TACACS+ Protocol](https://www.rfc-editor.org/rfc/rfc8907)
- [RFC 2865 — Remote Authentication Dial In User Service (RADIUS)](https://www.rfc-editor.org/rfc/rfc2865)
- [Cisco IOS TACACS+ Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_usr_tacacs/configuration/xe-16/sec-usr-tacacs-xe-16-book.html)
- [Cisco ISE Device Administration Guide](https://www.cisco.com/c/en/us/td/docs/security/ise/3-1/admin_guide/b_ise_admin_3_1/b_ISE_admin_31_device_admin.html)
- [Arista EOS TACACS+ Configuration](https://www.arista.com/en/um-eos/eos-tacacs)
- [Juniper JunOS TACACS+ Authentication](https://www.juniper.net/documentation/us/en/software/junos/user-access/topics/topic-map/user-access-tacacs-authentication.html)
- [tac_plus — Open Source TACACS+ Daemon](https://github.com/facebook/tac_plus)
