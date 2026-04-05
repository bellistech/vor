# Network Services (Infrastructure Protocols & Management)

Practical reference for the infrastructure services that keep networks running: DHCP relay and snooping, NTP/PTP time synchronization, file transfer for device management, syslog, AAA, SNMP, and discovery protocols.

## DHCP Relay and Server

### ISC DHCP Server — Advanced Configuration

```bash
# /etc/dhcp/dhcpd.conf — Shared networks and failover

# Shared network (multiple subnets on one interface)
shared-network "campus" {
    subnet 10.1.1.0 netmask 255.255.255.0 {
        range 10.1.1.100 10.1.1.200;
        option routers 10.1.1.1;
    }
    subnet 10.1.2.0 netmask 255.255.255.0 {
        range 10.1.2.100 10.1.2.200;
        option routers 10.1.2.1;
    }
}

# DHCP failover (active/standby)
failover peer "dhcp-failover" {
    primary;                            # or "secondary" on standby
    address 10.0.0.10;                  # this server
    peer address 10.0.0.11;             # partner server
    max-response-delay 60;
    max-unacked-updates 10;
    load balance max seconds 3;
    mclt 3600;                          # max client lead time (primary only)
    split 128;                          # 50/50 split (0-256, primary only)
}

subnet 10.1.1.0 netmask 255.255.255.0 {
    pool {
        failover peer "dhcp-failover";
        range 10.1.1.100 10.1.1.200;
    }
}

# Logging
log-facility local7;                    # syslog facility for DHCP events
```

### Cisco IOS DHCP Server

```
! IOS DHCP server
ip dhcp excluded-address 10.1.1.1 10.1.1.10
ip dhcp pool VLAN10
  network 10.1.1.0 255.255.255.0
  default-router 10.1.1.1
  dns-server 8.8.8.8 8.8.4.4
  domain-name example.com
  lease 0 8 0                            ! 0 days, 8 hours, 0 minutes

! Static binding
ip dhcp pool PRINTER
  host 10.1.1.50 255.255.255.0
  hardware-address 0011.2233.4455
  client-name printer01

! Verify
show ip dhcp binding
show ip dhcp pool
show ip dhcp conflict
show ip dhcp server statistics
debug ip dhcp server events
```

### DHCP Relay (ip helper-address)

```
! Cisco IOS relay agent
interface GigabitEthernet0/1
  ip address 10.1.1.1 255.255.255.0
  ip helper-address 10.0.0.50             ! DHCP server
  ip helper-address 10.0.0.51             ! redundant DHCP server

! ip helper-address forwards these UDP ports by default:
!   67  (DHCP/BOOTP server)    69  (TFTP)
!   49  (TACACS)               53  (DNS)
!   37  (Time)                 137 (NetBIOS NS)
!   138 (NetBIOS DS)

! Forward only DHCP (disable other protocols)
no ip forward-protocol udp 69
no ip forward-protocol udp 53

! Linux relay
dhcrelay -i eth0 -i eth1 10.0.0.50 10.0.0.51    # relay between interfaces

# Kea DHCP relay (ISC modern replacement)
# /etc/kea/kea-dhcp4.conf
# "relay": { "ip-addresses": ["10.1.1.1"] }
```

### Option 82 — Relay Agent Information

```
! Enable Option 82 insertion on relay
interface GigabitEthernet0/1
  ip dhcp relay information option

! Circuit-ID sub-option (identifies physical port)
! Format: vlan-module-port (e.g., "Gi0/1:100" for port Gi0/1, VLAN 100)

! Remote-ID sub-option (identifies the relay device)
! Default: relay MAC address; configurable:
ip dhcp relay information option remote-id MySwitch01

! Trust Option 82 on downstream relay
ip dhcp relay information trusted

! Server-side: use Option 82 for pool selection (ISC DHCP)
# /etc/dhcp/dhcpd.conf
class "floor1" {
    match if option agent.circuit-id = "Gi0/1:100";
}
subnet 10.1.1.0 netmask 255.255.255.0 {
    pool {
        allow members of "floor1";
        range 10.1.1.100 10.1.1.150;
    }
}
```

### DHCP Snooping

```
! Enable DHCP snooping globally
ip dhcp snooping
ip dhcp snooping vlan 10,20,30

! Trust uplinks to legitimate DHCP servers
interface GigabitEthernet0/24
  ip dhcp snooping trust

! Untrusted access ports (default) — drops server messages (OFFER, ACK)
interface range GigabitEthernet0/1-23
  ip dhcp snooping limit rate 15          ! max 15 DHCP packets/sec per port

! Verify
show ip dhcp snooping
show ip dhcp snooping binding             ! the snooping database
show ip dhcp snooping statistics

! DHCP snooping database persistence
ip dhcp snooping database flash:/dhcp-snoop.db
ip dhcp snooping database write-delay 300

! Option 82 with snooping (inserted automatically)
ip dhcp snooping information option

! DAI and IP Source Guard depend on the snooping binding table
ip arp inspection vlan 10
interface GigabitEthernet0/1
  ip verify source                        ! IP Source Guard
```

## NTP (Network Time Protocol)

### Stratum Hierarchy

```
Stratum 0 — Reference clocks (GPS, cesium, rubidium)
     |       Hardware devices, not network-accessible
     v
Stratum 1 — Servers directly connected to stratum 0
     |       time.nist.gov, time.google.com
     v
Stratum 2 — Synced to stratum 1 (pool.ntp.org)
     |
     v
Stratum 3..15 — Each level synced to one above
Stratum 16 — Unsynchronized (invalid)

Rules:
  - A server's stratum = source stratum + 1
  - Client selects lowest stratum with best quality metrics
  - Maximum stratum is 15; stratum 16 = "unsynchronized"
```

### NTP Authentication

```bash
# chrony with NTS (Network Time Security, RFC 8915)
# /etc/chrony/chrony.conf
server time.cloudflare.com iburst nts
server nts.netnod.se iburst nts
ntsdumpdir /var/lib/chrony

# chrony with symmetric key authentication
# /etc/chrony/chrony.conf
server 10.0.0.1 iburst key 1
keyfile /etc/chrony/chrony.keys

# /etc/chrony/chrony.keys
1 SHA256 HEX:A3B2C1D4E5F6...

# ntpd with symmetric key authentication
# /etc/ntp.conf
server 10.0.0.1 key 1
keys /etc/ntp/keys
trustedkey 1

# /etc/ntp/keys
1 SHA1 MySecretKey123
```

### Cisco IOS NTP

```
! NTP client
ntp server 10.0.0.1 prefer
ntp server 10.0.0.2
ntp source Loopback0

! NTP authentication
ntp authenticate
ntp authentication-key 1 md5 NtpSecret!
ntp trusted-key 1
ntp server 10.0.0.1 key 1

! NTP access control
ntp access-group peer 10               ! ACL 10 can peer
ntp access-group serve-only 20         ! ACL 20 can only query

! NTP master (make this device a stratum source)
ntp master 3                           ! act as stratum 3

! Verify
show ntp status
show ntp associations detail
show ntp packets
show clock detail
```

### NTPv4 Improvements

```
NTPv3 vs NTPv4:
  - IPv6 native support
  - Autokey (public key authentication, deprecated by NTS)
  - Modified clock selection algorithm
  - Dynamic server discovery via manycast
  - Extension field support (future-proofing)
  - Improved clock discipline algorithm
  - Kiss-of-death (KoD) packets for rate limiting
```

## PTP / IEEE 1588

```
Precision Time Protocol — sub-microsecond accuracy (nanosecond with hardware timestamping)

Hierarchy:
  Grandmaster Clock (GMC) — best clock, elected by BMCA
     |
  Boundary Clock (BC) — PTP-aware switch, re-syncs each port
     |
  Ordinary Clock (OC) — end device (slave)

Transparent Clock (TC) — switch that adjusts correction field
                         without being a full PTP participant

# Linux PTP (linuxptp)
sudo apt install linuxptp

# Check hardware timestamping support
ethtool -T eth0 | grep -i ptp

# Run PTP slave
sudo ptp4l -i eth0 -s -m
#   -i eth0     interface with HW timestamping
#   -s          slave-only mode
#   -m          print to stdout

# Run PHC-to-system clock sync
sudo phc2sys -s eth0 -c CLOCK_REALTIME -w -m

# PTP master
sudo ptp4l -i eth0 -m              # default: can become master

# Configuration file
# /etc/linuxptp/ptp4l.conf
[global]
twoStepFlag     1
clientOnly      0
priority1       128
priority2       128
domainNumber    0
clockClass      248
transportSpecific 0x0

# PTP profiles
Profile              Use Case                 Accuracy
Default (IEEE 1588)  General purpose           ~1 us
Telecom (ITU-T G.8275.1)  Carrier networks   ~10-100 ns
AES67                Audio/video broadcast     ~1 us
SMPTE ST 2059        Professional media        ~1 us
gPTP (802.1AS)       Automotive/TSN            <1 us
```

## TFTP / FTP / SCP for IOS Image Management

### TFTP

```bash
# Start TFTP server (Linux)
sudo apt install tftpd-hpa
# Config: /etc/default/tftpd-hpa
TFTP_DIRECTORY="/srv/tftp"
TFTP_ADDRESS=":69"
TFTP_OPTIONS="--secure --create"

sudo systemctl restart tftpd-hpa

# Test TFTP client
tftp 10.0.0.100
> get ios-image.bin
> put running-config.txt
> quit
```

```
! IOS — Copy running image to TFTP server (backup)
copy flash:c2960-lanbasek9-mz.150-2.SE11.bin tftp:
! Address: 10.0.0.100
! Filename: c2960-lanbasek9-mz.150-2.SE11.bin

! IOS — Download new image from TFTP
copy tftp: flash:
! Address: 10.0.0.100
! Filename: c2960-lanbasek9-mz.152-7.E7.bin

! Verify image integrity
verify /md5 flash:c2960-lanbasek9-mz.152-7.E7.bin

! Set boot variable
boot system flash:c2960-lanbasek9-mz.152-7.E7.bin
write memory
reload
```

### SCP (Preferred — Encrypted)

```
! Enable SCP server on IOS
ip scp server enable
aaa authorization exec default local
username admin privilege 15 secret StrongPass!

! Copy from IOS to remote SCP server
copy flash:running-config scp://admin@10.0.0.100/backups/switch01.cfg

! Copy from remote SCP to IOS
copy scp://admin@10.0.0.100/images/new-ios.bin flash:
```

```bash
# SCP from workstation to device
scp new-ios.bin admin@10.1.1.1:flash:

# SCP from device to workstation
scp admin@10.1.1.1:flash:running-config ./backup/
```

### FTP

```
! IOS FTP client
ip ftp username admin
ip ftp password FtpPass!
copy ftp://10.0.0.100/images/new-ios.bin flash:

! Archive config to FTP
archive
  path ftp://10.0.0.100/configs/$h-
  write-memory
  time-period 1440                      ! auto-archive every 24h
```

### IOS Image Verification

```
! Verify MD5 hash
verify /md5 flash:ios-image.bin
! Compare against vendor-published hash

! Show flash contents
show flash:
dir flash:

! Show current boot image
show version
show boot

! IOS resilience — protect boot image
secure boot-image
secure boot-config
```

## Syslog

### Facility and Severity Matrix

```
Severity Levels (0 = most critical):
Level  Keyword       Description                  Cisco Keyword
──────────────────────────────────────────────────────────────────
0      emerg         System unusable               emergencies
1      alert         Immediate action needed        alerts
2      crit          Critical conditions            critical
3      err           Error conditions               errors
4      warning       Warning conditions             warnings
5      notice        Normal but significant         notifications
6      info          Informational                  informational
7      debug         Debug messages                 debugging

Facility Codes:
Code  Keyword         Description
───────────────────────────────────────────────
0     kern            Kernel messages
1     user            User-level messages
3     daemon          System daemons
4     auth            Security/authorization
5     syslog          Syslog daemon itself
6     lpr             Printer subsystem
9     cron            Cron subsystem
10    authpriv        Private auth messages
16    local0          Local use 0 (network devices)
17    local1          Local use 1
...
23    local7          Local use 7

Priority = Facility * 8 + Severity
Example:  local7 (23) + warning (4) = 23*8+4 = 188  → <188>
```

### Remote Syslog Configuration

```bash
# rsyslog — forward to remote server
# /etc/rsyslog.conf or /etc/rsyslog.d/remote.conf

# UDP forwarding (traditional, unreliable)
*.* @syslog.example.com:514

# TCP forwarding (reliable)
*.* @@syslog.example.com:514

# TLS forwarding (encrypted, rsyslog + GnuTLS)
$DefaultNetstreamDriverCAFile /etc/rsyslog.d/ca.pem
$DefaultNetstreamDriver gtls
$ActionSendStreamDriverMode 1
$ActionSendStreamDriverAuthMode x509/name
*.* @@(o)syslog.example.com:6514

# Filter by facility and severity
local0.warning    /var/log/network-warnings.log
local7.*          /var/log/network-all.log
*.err             /var/log/errors.log

# Template for structured output
template(name="json-syslog" type="list") {
    constant(value="{")
    constant(value="\"timestamp\":\"") property(name="timereported" dateFormat="rfc3339") constant(value="\",")
    constant(value="\"host\":\"") property(name="hostname") constant(value="\",")
    constant(value="\"severity\":\"") property(name="syslogseverity-text") constant(value="\",")
    constant(value="\"facility\":\"") property(name="syslogfacility-text") constant(value="\",")
    constant(value="\"message\":\"") property(name="msg" format="json") constant(value="\"")
    constant(value="}\n")
}
local7.* action(type="omfile" file="/var/log/network.json" template="json-syslog")
```

### Syslog-ng

```bash
# /etc/syslog-ng/syslog-ng.conf

source s_network {
    tcp(ip("0.0.0.0") port(514));
    udp(ip("0.0.0.0") port(514));
};

filter f_network { facility(local0, local7); };

destination d_network {
    file("/var/log/network/${HOST}/${YEAR}-${MONTH}-${DAY}.log"
         create-dirs(yes));
};

log { source(s_network); filter(f_network); destination(d_network); };
```

### Structured Syslog (RFC 5424)

```
RFC 5424 message format:
<PRI>VERSION TIMESTAMP HOSTNAME APP-NAME PROCID MSGID [SD-ID SD-PARAMS] MSG

Example:
<165>1 2026-04-05T14:30:00.000Z switch01 ospfd 12345 ADJCHANGE
  [origin ip="10.1.1.1" software="Quagga"][meta sequenceId="42"]
  OSPF neighbor 10.1.1.2 state changed to Full

Fields:
  PRI       — facility*8+severity, e.g., <165> = local4(20)*8 + notice(5)
  VERSION   — always "1" for RFC 5424
  TIMESTAMP — ISO 8601 with microsecond precision
  SD-ID     — structured data block identifier
  SD-PARAMS — key="value" pairs inside structured data
```

### Cisco IOS Syslog

```
! Send logs to remote syslog server
logging host 10.0.0.200
logging host 10.0.0.201 transport tcp port 1514

! Set facility and severity
logging facility local7
logging trap informational              ! send severity 0-6 to remote
logging console warnings                ! severity 0-4 to console
logging buffered 65536 debugging        ! severity 0-7 to local buffer

! Timestamps with milliseconds
service timestamps log datetime msec localtime show-timezone
service timestamps debug datetime msec localtime show-timezone

! Source interface
logging source-interface Loopback0

! Rate limiting
logging rate-limit 100                  ! max 100 messages/sec

! Verify
show logging
show logging history
```

## AAA (Authentication, Authorization, Accounting)

### TACACS+ vs RADIUS

```
Feature              TACACS+                     RADIUS
───────────────────────────────────────────────────────────────
Protocol             TCP/49                      UDP/1812,1813
Encryption           Full packet encrypted       Password only encrypted
AAA separation       Separate A, A, A            Combined auth+authz
Multiprotocol        Cisco proprietary            Standard (RFC 2865)
Command authz        Yes (per-command control)    No
Accounting           Separate TCP session         UDP, less reliable
Primary use          Device admin (CLI access)    Network access (802.1X, VPN)
Attribute-Value      AV pairs (flexible)          Standard + VSAs
```

### Cisco AAA Configuration

```
! Enable AAA
aaa new-model

! Authentication method lists
aaa authentication login default group tacacs+ local
aaa authentication login CONSOLE local
aaa authentication login VTY group tacacs+ local enable

! Apply to lines
line console 0
  login authentication CONSOLE
line vty 0 15
  login authentication VTY
  transport input ssh

! Authorization method lists
aaa authorization exec default group tacacs+ local
aaa authorization commands 15 default group tacacs+ local

! Accounting
aaa accounting exec default start-stop group tacacs+
aaa accounting commands 15 default start-stop group tacacs+
aaa accounting network default start-stop group radius

! TACACS+ server
tacacs server PRIMARY
  address ipv4 10.0.0.100
  key 0 TacacsSecret!
  timeout 5

tacacs server BACKUP
  address ipv4 10.0.0.101
  key 0 TacacsSecret!
  timeout 5

aaa group server tacacs+ TACACS_GROUP
  server name PRIMARY
  server name BACKUP
  ip tacacs source-interface Loopback0

! RADIUS server
radius server RADIUS01
  address ipv4 10.0.0.110 auth-port 1812 acct-port 1813
  key 0 RadiusSecret!
  timeout 3
  retransmit 2

aaa group server radius RADIUS_GROUP
  server name RADIUS01
  ip radius source-interface Loopback0
```

### Authorization Privilege Levels

```
! Cisco IOS privilege levels (0-15)
Level   Default Access
0       disable, enable, exit, help, logout
1       User EXEC (show commands, ping, traceroute)
2-14    Custom (assigned by admin)
15      Privileged EXEC (full access, configure terminal)

! Custom privilege level
privilege exec level 7 show running-config
privilege exec level 7 show interfaces
privilege exec level 7 ping

! Assign level to user
username netops privilege 7 secret NetOps!

! TACACS+ server returns privilege level via AV pair:
!   priv-lvl=7
```

### RADIUS for 802.1X

```
! 802.1X with RADIUS
aaa authentication dot1x default group radius
aaa authorization network default group radius

dot1x system-auth-control

interface GigabitEthernet0/1
  switchport mode access
  switchport access vlan 10
  authentication port-control auto
  dot1x pae authenticator
  authentication order dot1x mab        ! try 802.1X, then MAB fallback
  authentication fallback GUEST_ACL

! RADIUS server must return VLAN via attributes:
!   Tunnel-Type = VLAN (13)
!   Tunnel-Medium-Type = IEEE-802 (6)
!   Tunnel-Private-Group-ID = 10
```

## SNMP (Simple Network Management Protocol)

### v2c vs v3 Quick Reference

```
Feature              SNMPv2c                  SNMPv3
──────────────────────────────────────────────────────────────
Authentication       Community string         USM (MD5/SHA/SHA-256)
Encryption           None                     DES/3DES/AES-128/256
Access control       Community-based          VACM (views, groups)
Security level       N/A                      noAuthNoPriv, authNoPriv, authPriv
Best for             Lab/legacy               Production environments
```

### MIB Walks and Queries

```bash
# Walk entire MIB tree
snmpwalk -v2c -c public 10.1.1.1 .1

# Walk specific subtrees
snmpwalk -v2c -c public 10.1.1.1 ifTable            # interface table
snmpwalk -v2c -c public 10.1.1.1 ifXTable           # extended interface stats
snmpwalk -v2c -c public 10.1.1.1 ipRouteTable       # routing table

# Bulk walk (faster, fewer packets)
snmpbulkwalk -v2c -c public -Cr25 10.1.1.1 ifTable  # 25 OIDs per PDU

# Get specific OIDs
snmpget -v2c -c public 10.1.1.1 sysUpTime.0
snmpget -v2c -c public 10.1.1.1 ifOperStatus.1      # interface 1 status

# SNMPv3 walk
snmpwalk -v3 -l authPriv \
    -u monitor -a SHA -A "AuthPass!" \
    -x AES -X "PrivPass!" 10.1.1.1 system

# Table query with formatted output
snmptable -v2c -c public -Ci 10.1.1.1 ifTable

# Translate OID names
snmptranslate -On IF-MIB::ifOperStatus          # → .1.3.6.1.2.1.2.2.1.8
snmptranslate -Of .1.3.6.1.2.1.2.2.1.8          # → iso.org.dod...ifOperStatus
```

### Traps and Informs

```bash
# Send SNMPv2c trap
snmptrap -v2c -c public 10.0.0.200 '' \
    IF-MIB::linkDown ifIndex i 2 ifOperStatus i 2

# Send SNMPv3 inform (acknowledged, reliable)
snmpinform -v3 -l authPriv \
    -u trapuser -a SHA -A "Auth!" -x AES -X "Priv!" \
    10.0.0.200 '' IF-MIB::linkDown ifIndex i 2

# Trap vs Inform:
#   Trap   — fire-and-forget UDP, no confirmation
#   Inform — acknowledged, retransmitted if no response (more reliable)
```

### SNMP Views (v3 Access Control)

```
! Cisco IOS SNMPv3 configuration
snmp-server view READONLY iso included
snmp-server view READONLY internet included
snmp-server view LIMITED system included
snmp-server view LIMITED interfaces included

snmp-server group MONITORS v3 priv read READONLY
snmp-server group ADMINS v3 priv read READONLY write READONLY

snmp-server user monitor MONITORS v3 auth sha AuthPass123 priv aes 128 PrivPass456
snmp-server user admin ADMINS v3 auth sha AdminAuth! priv aes 128 AdminPriv!

! Trap receiver
snmp-server host 10.0.0.200 version 3 priv monitor
snmp-server enable traps

! ACL to restrict SNMP access
snmp-server community public RO 99
access-list 99 permit 10.0.0.0 0.0.0.255
```

## CDP and LLDP

### CDP (Cisco Discovery Protocol)

```
! CDP — Cisco proprietary, Layer 2, multicast to 01:00:0C:CC:CC:CC

! Global commands
cdp run                                 ! enable globally (default on)
no cdp run                              ! disable globally

! Per-interface
interface GigabitEthernet0/1
  cdp enable                            ! enable on this port
  no cdp enable                         ! disable on this port (security)

! Timers
cdp timer 60                            ! advertisement interval (default 60s)
cdp holdtime 180                        ! hold time (default 180s)

! Verification
show cdp neighbors                      ! summary of neighbors
show cdp neighbors detail               ! full details (IP, platform, version)
show cdp entry *                        ! all entries, all detail
show cdp interface                      ! CDP-enabled interfaces
show cdp traffic                        ! packet statistics
```

### LLDP (IEEE 802.1AB — Vendor-Neutral)

```
! LLDP — standard, Layer 2, multicast to 01:80:C2:00:00:0E

! Global commands
lldp run                                ! enable globally

! Per-interface
interface GigabitEthernet0/1
  lldp transmit                         ! send LLDP frames
  lldp receive                          ! process received LLDP frames
  no lldp transmit                      ! disable transmit (stealth)

! Timers
lldp timer 30                           ! transmit interval (default 30s)
lldp holdtime 120                       ! hold time (default 120s)
lldp reinit 2                           ! reinit delay (default 2s)

! TLV selection
lldp tlv-select management-address      ! advertise management address
lldp tlv-select port-description        ! advertise port description
lldp tlv-select system-name             ! advertise system name
lldp tlv-select system-capabilities     ! advertise system capabilities

! Verification
show lldp neighbors
show lldp neighbors detail
show lldp entry *
show lldp interface
show lldp traffic
```

```bash
# Linux LLDP (lldpd)
sudo apt install lldpd
sudo systemctl enable lldpd

# Show neighbors
lldpcli show neighbors
lldpcli show neighbors detail

# Configure
lldpcli configure system hostname "server01"
lldpcli configure system description "Web Server"

# Show local chassis info
lldpcli show chassis
lldpcli show statistics
```

### CDP vs LLDP Comparison

```
Feature              CDP                        LLDP
────────────────────────────────────────────────────────────
Standard             Cisco proprietary           IEEE 802.1AB
Multicast address    01:00:0C:CC:CC:CC          01:80:C2:00:00:0E
EtherType            —                          0x88CC
Default timer        60s                        30s
Default holdtime     180s                       120s
VLAN info            Native VLAN                802.1Q via org-specific TLV
Power (PoE)          CDP power negotiation       LLDP-MED (802.1AB + TIA-1057)
VoIP support         Cisco IP Phone             LLDP-MED (network policy TLV)
```

## UDLD (UniDirectional Link Detection)

```
! UDLD detects unidirectional links (one fiber strand down, bad SFP, etc.)

! Enable globally (normal mode — alerts only)
udld enable

! Aggressive mode (err-disables the port on detection)
udld aggressive

! Per-interface
interface GigabitEthernet0/1
  udld port aggressive

! Verify
show udld neighbors
show udld GigabitEthernet0/1

! Recovery from err-disabled
errdisable recovery cause udld
errdisable recovery interval 300        ! retry every 300 seconds

! Manual recovery
shutdown
no shutdown

! UDLD modes:
!   Normal     — detects, logs, but does not shut port
!   Aggressive — sends 8 probes; if no response, err-disables port
!
! UDLD message interval default: 15 seconds
udld message time 7                     ! faster detection (7 seconds)
```

## Tips

- Use DHCP snooping on all access switches. It builds the binding table that Dynamic ARP Inspection (DAI) and IP Source Guard depend on. Without snooping, those features have no data to work with.
- Option 82 is automatically inserted when DHCP snooping is enabled. If a downstream switch also inserts Option 82, the upstream device may drop the packet unless you configure `ip dhcp relay information trusted`.
- Always configure NTP authentication in production. An attacker who can shift your clocks by minutes can break Kerberos, invalidate TLS certificates, and corrupt log correlation.
- For sub-microsecond accuracy, use PTP (IEEE 1588) with hardware timestamping. Software timestamping adds kernel scheduling jitter that limits accuracy to tens of microseconds at best.
- TACACS+ encrypts the entire packet; RADIUS only encrypts the password field. Use TACACS+ for device administration where command authorization matters, and RADIUS for network access (802.1X, VPN).
- SNMPv2c community strings are sent in cleartext. If you must use v2c, restrict it with ACLs. For production monitoring, always use SNMPv3 with authPriv.
- SNMP informs are more reliable than traps because they require acknowledgment. Use informs for critical alerts (link down, CPU threshold) and traps for high-volume events where occasional loss is acceptable.
- Disable CDP on external-facing interfaces. CDP reveals device model, IOS version, IP addresses, and VLAN information that an attacker can use for targeted exploits.
- Use LLDP instead of CDP in multi-vendor environments. LLDP-MED extends LLDP for VoIP phone provisioning (VLAN, QoS, PoE negotiation) without requiring Cisco-only infrastructure.
- Enable UDLD aggressive on all fiber links. A single unidirectional fiber failure can cause STP loops, asymmetric routing, and silent black-holing that is extremely difficult to diagnose.
- Set syslog to TCP or TLS for reliable delivery. UDP syslog silently drops messages under congestion, which is exactly when you need your logs most.
- Always configure a local AAA fallback method (`local` or `enable`) after your TACACS+/RADIUS group. If the AAA server is unreachable and there is no fallback, you are locked out of your own devices.

## See Also

- dhcp, dhcpv6, ntp, ptp, snmp, radius, tacacs, lldp, syslog, ftp

## References

- [RFC 2131 — DHCP](https://www.rfc-editor.org/rfc/rfc2131)
- [RFC 3046 — DHCP Relay Agent Information Option](https://www.rfc-editor.org/rfc/rfc3046)
- [RFC 5905 — NTPv4](https://www.rfc-editor.org/rfc/rfc5905)
- [RFC 8915 — NTS (Network Time Security)](https://www.rfc-editor.org/rfc/rfc8915)
- [IEEE 1588-2019 — PTP v2.1](https://standards.ieee.org/standard/1588-2019.html)
- [RFC 5424 — Syslog Protocol](https://www.rfc-editor.org/rfc/rfc5424)
- [RFC 5425 — TLS Transport for Syslog](https://www.rfc-editor.org/rfc/rfc5425)
- [RFC 2865 — RADIUS](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 8907 — TACACS+ (standardized)](https://www.rfc-editor.org/rfc/rfc8907)
- [RFC 3411-3418 — SNMPv3 Framework](https://www.rfc-editor.org/rfc/rfc3411)
- [IEEE 802.1AB — LLDP](https://standards.ieee.org/standard/802_1AB-2016.html)
- [Cisco DHCP Snooping Configuration Guide](https://www.cisco.com/c/en/us/td/docs/switches/lan/catalyst2960x/software/15-2_7_e/configuration_guide/b_1527e_consolidated_2960x_cg/b_1527e_consolidated_2960x_cg_chapter_011010.html)
- [ISC Kea DHCP Documentation](https://kea.readthedocs.io/)
- [linuxptp Project](https://linuxptp.sourceforge.net/)
