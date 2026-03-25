# SNMP (Simple Network Management Protocol)

> Query and manage network devices using OIDs, MIB trees, and community strings (v2c) or USM authentication (v3).

## Concepts

### Architecture

```
# Manager (NMS) — polls agents, receives traps
# Agent — runs on managed device, exposes MIB data
# MIB (Management Information Base) — tree of OIDs describing available data
# OID (Object Identifier) — dotted numeric path (e.g., 1.3.6.1.2.1.1.1.0)
# Community string (v2c) — plaintext password for read/write access
```

### SNMP Versions

```
# v1    — community-based, no encryption, limited error handling
# v2c   — community-based, bulk operations, improved error codes
# v3    — USM authentication + encryption, access control (VACM)
#         Security levels:
#           noAuthNoPriv — username only (no auth, no encryption)
#           authNoPriv   — username + auth (MD5/SHA), no encryption
#           authPriv     — username + auth + encryption (DES/AES)
```

## Commands

### snmpget — Fetch a Single OID

```bash
# SNMPv2c
snmpget -v2c -c public 192.168.1.1 sysDescr.0
snmpget -v2c -c public 192.168.1.1 1.3.6.1.2.1.1.1.0    # numeric OID

# SNMPv3 with authPriv
snmpget -v3 -l authPriv \
    -u monitorUser \
    -a SHA -A "AuthPass123" \
    -x AES -X "PrivPass456" \
    192.168.1.1 sysUpTime.0
```

### snmpwalk — Traverse a Subtree

```bash
# Walk the entire system subtree
snmpwalk -v2c -c public 192.168.1.1 system

# Walk interface table
snmpwalk -v2c -c public 192.168.1.1 ifTable

# Use -Oq for quiet output (OID = value), -Ov for values only
snmpwalk -v2c -c public -Oq 192.168.1.1 ifDescr

# SNMPv3 walk
snmpwalk -v3 -l authPriv -u monitorUser \
    -a SHA -A "AuthPass123" -x AES -X "PrivPass456" \
    192.168.1.1 hrStorageTable
```

### snmpbulkwalk — Efficient Bulk Retrieval (v2c/v3)

```bash
# Faster than snmpwalk; uses GETBULK requests
snmpbulkwalk -v2c -c public -Cr25 192.168.1.1 ifTable   # 25 rows per request
```

### snmptable — Display Tabular Data

```bash
# Formatted table output
snmptable -v2c -c public -Cb 192.168.1.1 ifTable
snmptable -v2c -c public -Cb 192.168.1.1 hrStorageTable
```

### snmpset — Write a Value

```bash
# Set sysContact (requires RW community)
snmpset -v2c -c private 192.168.1.1 sysContact.0 s "admin@example.com"

# Types: i=INTEGER, s=STRING, a=IPADDRESS, o=OID, u=UNSIGNED, x=HEX
snmpset -v2c -c private 192.168.1.1 ifAdminStatus.2 i 2   # disable interface 2
```

### snmptrap — Send Traps and Informs

```bash
# SNMPv2c trap
snmptrap -v2c -c public 10.0.0.5 "" \
    1.3.6.1.4.1.99999 \
    1.3.6.1.4.1.99999.1 s "Link down on port 3"

# SNMPv3 inform (acknowledged trap)
snmpinform -v3 -l authPriv -u trapUser \
    -a SHA -A "AuthPass123" -x AES -X "PrivPass456" \
    10.0.0.5 "" 1.3.6.1.4.1.99999
```

## Common OIDs

### System and Interface OIDs

```
# System group (1.3.6.1.2.1.1)
sysDescr.0         .1.3.6.1.2.1.1.1.0      # device description
sysObjectID.0      .1.3.6.1.2.1.1.2.0      # vendor OID
sysUpTime.0        .1.3.6.1.2.1.1.3.0      # uptime in timeticks
sysContact.0       .1.3.6.1.2.1.1.4.0      # admin contact
sysName.0          .1.3.6.1.2.1.1.5.0      # hostname
sysLocation.0      .1.3.6.1.2.1.1.6.0      # physical location

# Interface table (1.3.6.1.2.1.2.2)
ifDescr             .1.3.6.1.2.1.2.2.1.2    # interface name
ifOperStatus        .1.3.6.1.2.1.2.2.1.8    # 1=up, 2=down
ifInOctets          .1.3.6.1.2.1.2.2.1.10   # bytes received
ifOutOctets         .1.3.6.1.2.1.2.2.1.16   # bytes transmitted

# Host resources (1.3.6.1.2.1.25)
hrStorageDescr      .1.3.6.1.2.1.25.2.3.1.3   # storage description
hrStorageUsed       .1.3.6.1.2.1.25.2.3.1.6   # blocks used
hrProcessorLoad     .1.3.6.1.2.1.25.3.3.1.2   # CPU load per core
```

## net-snmp Configuration

### Agent Config (snmpd.conf)

```conf
# /etc/snmp/snmpd.conf

# v2c community access
rocommunity public  10.0.0.0/8          # read-only from 10.x
rwcommunity private 10.0.0.5/32         # read-write from NMS only

# v3 user setup (run net-snmp-create-v3-user or add manually)
# createUser monitorUser SHA "AuthPass123" AES "PrivPass456"
# rouser monitorUser priv

# System info
syslocation "Rack 4, DC-East"
syscontact  "admin@example.com"

# Listening address
agentaddress udp:161,udp6:161

# Limit exposed subtree
view systemonly included .1.3.6.1.2.1.1
view systemonly included .1.3.6.1.2.1.25
access notConfigGroup "" any noauth exact systemonly none none
```

### Creating v3 Users

```bash
# Stop snmpd first
systemctl stop snmpd

# Create user with auth+priv
net-snmp-create-v3-user -ro -a SHA -A "AuthPass123" -x AES -X "PrivPass456" monitorUser

systemctl start snmpd
```

## Monitoring Integration

### Polling with Scripts

```bash
# Simple bandwidth monitor (in/out octets delta)
PREV_IN=$(snmpget -v2c -c public -Ov -Oq 192.168.1.1 ifInOctets.1)
sleep 60
CURR_IN=$(snmpget -v2c -c public -Ov -Oq 192.168.1.1 ifInOctets.1)
echo "Bytes/sec: $(( (CURR_IN - PREV_IN) / 60 ))"

# Enumerate all interfaces and status
snmpwalk -v2c -c public -Oq 192.168.1.1 ifOperStatus | \
    while read oid val; do echo "$oid -> $val"; done
```

## Tips

- Always use v3 with authPriv in production; v2c communities are plaintext on the wire.
- MIB files let you use names instead of numeric OIDs; install with `apt install snmp-mibs-downloader`.
- Comment out `mibs :` in `/etc/snmp/snmp.conf` to enable MIB name resolution by default.
- Use `-On` to force numeric OID output for scripting consistency.
- Bulk operations (`snmpbulkwalk`) are significantly faster than `snmpwalk` for large tables.
- Trap receivers: `snmptrapd` listens on UDP 162; configure in `/etc/snmp/snmptrapd.conf`.

## References

- [RFC 3411 — An Architecture for Describing SNMP Management Frameworks](https://www.rfc-editor.org/rfc/rfc3411)
- [RFC 3412 — Message Processing and Dispatching for SNMP](https://www.rfc-editor.org/rfc/rfc3412)
- [RFC 3414 — User-based Security Model (USM) for SNMPv3](https://www.rfc-editor.org/rfc/rfc3414)
- [RFC 3416 — Version 2 of the Protocol Operations for SNMP](https://www.rfc-editor.org/rfc/rfc3416)
- [RFC 3418 — Management Information Base (MIB) for SNMPv2](https://www.rfc-editor.org/rfc/rfc3418)
- [RFC 3826 — The Advanced Encryption Standard (AES) Cipher Algorithm in the SNMP USM](https://www.rfc-editor.org/rfc/rfc3826)
- [Net-SNMP Official Documentation](http://www.net-snmp.org/docs/)
- [Net-SNMP Community Wiki](http://www.net-snmp.org/wiki/)
- [Net-SNMP — snmpwalk Man Page](http://www.net-snmp.org/docs/man/snmpwalk.html)
- [IANA — SNMP Number Spaces](https://www.iana.org/assignments/smi-numbers/smi-numbers.xhtml)
- [Cisco SNMP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/snmp/configuration/xe-16/snmp-xe-16-book.html)
