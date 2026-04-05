> For authorized security testing, red team exercises, and educational study only.

# Enumeration Techniques (CEH v13 — Module 04)

Systematic extraction of usernames, shares, services, and network details from target systems using protocol-specific queries.

## NetBIOS Enumeration (Ports 137–139)

```bash
# Windows built-in — query remote name table
nbtstat -A <target-ip>

# Scan a subnet for NetBIOS names
nbtscan -r 192.168.1.0/24

# Full NetBIOS/SMB enumeration (Linux)
enum4linux -a <target-ip>

# Query NetBIOS name service (UDP 137)
nmblookup -A <target-ip>
```

| Suffix | Type   | Meaning              |
|--------|--------|----------------------|
| `<00>` | UNIQUE | Workstation Service  |
| `<20>` | UNIQUE | File Server Service  |
| `<1C>` | GROUP  | Domain Controllers   |
| `<1B>` | UNIQUE | Domain Master Browser|

**Countermeasures:** Disable NetBIOS over TCP/IP in adapter settings. Block UDP 137, TCP 138–139 at the perimeter. Remove WINS if unused.

## SMB Enumeration (Port 445)

```bash
# List shares anonymously
smbclient -L //<target-ip> -N

# Connect to a specific share
smbclient //<target-ip>/share_name -U <user>

# enum4linux-ng — modern Python rewrite
enum4linux-ng -A <target-ip>

# CrackMapExec — enumerate users, shares, sessions
crackmapexec smb <target-ip> --shares
crackmapexec smb <target-ip> --users
crackmapexec smb <target-ip> --sessions
crackmapexec smb <target-ip> --rid-brute

# Null session via rpcclient
rpcclient -U "" -N <target-ip>
rpcclient $> enumdomusers
rpcclient $> enumdomgroups
rpcclient $> querydominfo
```

**Countermeasures:** Disable null sessions (`RestrictAnonymous = 1`). Require SMB signing. Use SMBv3 with encryption. Restrict share permissions.

## SNMP Enumeration (UDP 161)

```bash
# Walk the full MIB tree (v1/v2c)
snmpwalk -v2c -c public <target-ip>

# Target specific OIDs
snmpwalk -v2c -c public <target-ip> 1.3.6.1.2.1.25.4.2.1.2   # running processes
snmpwalk -v2c -c public <target-ip> 1.3.6.1.2.1.25.6.3.1.2   # installed software
snmpwalk -v2c -c public <target-ip> 1.3.6.1.4.1.77.1.2.25     # user accounts (Windows)

# Automated check
snmp-check -c public <target-ip>

# SNMPv3 with auth + encryption
snmpwalk -v3 -u <user> -l authPriv -a SHA -A <authpass> -x AES -X <privpass> <target-ip>

# Brute-force community strings
onesixtyone -c /usr/share/seclists/Discovery/SNMP/common-snmp-community-strings.txt <target-ip>
```

| OID Prefix              | Information             |
|-------------------------|-------------------------|
| `1.3.6.1.2.1.1`        | System description      |
| `1.3.6.1.2.1.2`        | Network interfaces      |
| `1.3.6.1.2.1.4.20`     | IP addresses            |
| `1.3.6.1.2.1.25.4.2`   | Running processes       |
| `1.3.6.1.2.1.25.6.3`   | Installed software      |

**Countermeasures:** Change default community strings. Use SNMPv3 with authPriv. Apply ACLs to restrict SNMP access. Disable SNMP if unused.

## LDAP Enumeration (Ports 389, 636)

```bash
# Discover base DN via rootDSE (anonymous)
ldapsearch -x -H ldap://<target-ip> -s base namingContexts

# Enumerate all users
ldapsearch -x -H ldap://<target-ip> -b "DC=corp,DC=local" "(objectClass=user)" cn sAMAccountName

# Enumerate groups and members
ldapsearch -x -H ldap://<target-ip> -b "DC=corp,DC=local" "(objectClass=group)" cn member

# Authenticated query
ldapsearch -x -H ldap://<target-ip> -D "CN=user,DC=corp,DC=local" -W -b "DC=corp,DC=local"

# Find domain controllers
ldapsearch -x -H ldap://<target-ip> -b "DC=corp,DC=local" "(&(objectCategory=computer)(userAccountControl:1.2.840.113556.1.4.803:=8192))"
```

**AD Naming:** `CN=CommonName`, `OU=OrganizationalUnit`, `DC=DomainComponent`. Full DN: `CN=John,OU=Sales,DC=corp,DC=local`

**Countermeasures:** Disable anonymous LDAP binds. Require LDAPS (port 636). Restrict `dsHeuristics` to prevent unauthenticated reads. Audit LDAP queries.

## NFS Enumeration (Port 2049)

```bash
# Show exported shares
showmount -e <target-ip>

# List all mount points
showmount -a <target-ip>

# Enumerate RPC services (NFS relies on RPC)
rpcinfo -p <target-ip>

# Mount an export
mkdir /tmp/nfs_mount
mount -t nfs <target-ip>:/export /tmp/nfs_mount

# NFSv4 — list pseudo-root
mount -t nfs4 <target-ip>:/ /tmp/nfs_mount
```

**Countermeasures:** Restrict exports to specific IPs/subnets in `/etc/exports`. Use `root_squash`. Prefer NFSv4 with Kerberos authentication. Block port 2049 externally.

## DNS Enumeration (Port 53)

```bash
# Zone transfer attempt
dig axfr @<dns-server> <domain>

# Brute-force subdomains
dnsenum <domain>
fierce --domain <domain>
dnsrecon -d <domain> -t brt -D /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt

# Reverse lookup sweep
dnsrecon -d <domain> -r 192.168.1.0/24

# Specific record queries
dig MX <domain>
dig NS <domain>
dig SRV _ldap._tcp.<domain>
dig TXT <domain>
```

**Countermeasures:** Restrict zone transfers to authorized secondaries (`allow-transfer`). Use split-horizon DNS. Enable DNSSEC. Rate-limit DNS queries.

## SMTP Enumeration (Port 25)

```bash
# VRFY — verify a user exists
telnet <target-ip> 25
VRFY admin

# EXPN — expand mailing list
EXPN staff

# RCPT TO — brute-force valid users
smtp-user-enum -M VRFY -U /usr/share/seclists/Usernames/top-usernames-shortlist.txt -t <target-ip>
smtp-user-enum -M RCPT -U users.txt -t <target-ip> -D <domain>

# Nmap scripting
nmap --script smtp-enum-users -p 25 <target-ip>
```

**Countermeasures:** Disable VRFY and EXPN commands. Require authentication for RCPT TO. Use SMTP relay restrictions. Deploy fail2ban for brute-force protection.

## FTP Enumeration (Port 21)

```bash
# Anonymous login attempt
ftp <target-ip>
# user: anonymous, pass: (blank or email)

# Banner grabbing
nc -nv <target-ip> 21
nmap -sV -p 21 <target-ip>

# Nmap FTP scripts
nmap --script ftp-anon,ftp-bounce,ftp-syst -p 21 <target-ip>

# Directory listing after login
ftp> ls -la
ftp> cd ..
ftp> pwd
```

**Countermeasures:** Disable anonymous FTP. Use SFTP or FTPS instead. Restrict FTP directories with chroot. Review banner for version disclosure.

## RPC / NIS Enumeration

```bash
# List registered RPC services
rpcinfo -p <target-ip>

# rpcclient for Windows RPC
rpcclient -U "" -N <target-ip>
rpcclient $> srvinfo
rpcclient $> netshareenum
rpcclient $> lookupnames administrator

# NIS (Network Information Service) — if exposed
ypcat -d <domain> passwd
ypcat -d <domain> hosts
ypwhich -d <domain>
```

**Countermeasures:** Filter RPC ports at the firewall. Migrate from NIS to LDAP/Kerberos. Use `rpcbind` ACLs. Disable unused RPC services.

## Active Directory Enumeration

```bash
# BloodHound — collect AD relationships
bloodhound-python -d <domain> -u <user> -p <pass> -c All -ns <dc-ip>

# PowerView (PowerShell)
Import-Module PowerView.ps1
Get-DomainUser
Get-DomainGroup -MemberIdentity "Domain Admins"
Get-DomainComputer -Properties dnshostname,operatingsystem
Find-DomainShare -CheckShareAccess

# Kerberoasting prep — find SPNs
GetUserSPNs.py <domain>/<user>:<pass> -dc-ip <dc-ip> -request

# AS-REP roastable accounts (no preauth)
GetNPUsers.py <domain>/ -usersfile users.txt -dc-ip <dc-ip> -format hashcat

# ADExplorer — Sysinternals GUI (snapshot AD database)
# Run from domain-joined Windows host
```

**Countermeasures:** Enforce least privilege. Monitor Kerberos ticket requests (Event ID 4769). Require preauth on all accounts. Audit BloodHound-style queries via SIEM. Use tiered admin model.

## BGP Enumeration

```bash
# Lookup ASN for an organization
whois -h whois.radb.net -- '-i origin AS<number>'

# Query BGP route via looking glass
# Use: https://www.bgp4.as/looking-glasses
# Or:  https://route-views.routeviews.org

# BGP path analysis
traceroute -A <target-ip>

# Hurricane Electric BGP toolkit
# https://bgp.he.net/
```

**Countermeasures:** Implement RPKI for route origin validation. Use BGP route filtering. Monitor for BGP hijacking via RIPE RIS or BGPStream.

## Tips

- Start with service discovery (`nmap -sV -sC`) before protocol-specific enumeration
- Default credentials and community strings remain common in real environments
- Null sessions and anonymous binds are low-hanging fruit — always test first
- Combine tools for coverage: automated scanners miss what manual queries catch
- Log every command for your pentest report — timestamps and exact output
- SNMPv1/v2c sends community strings in cleartext; capturing traffic reveals them

## See Also

- `sheets/offensive/scanning-and-reconnaissance.md`
- `sheets/offensive/vulnerability-analysis.md`
- `detail/offensive/enumeration-techniques.md`

## References

- EC-Council CEH v13 — Module 04: Enumeration
- RFC 1001/1002 (NetBIOS), RFC 4511 (LDAP), RFC 3530 (NFSv4), RFC 5321 (SMTP)
- [HackTricks — Enumeration](https://book.hacktricks.xyz/)
- [OWASP Testing Guide — Enumeration](https://owasp.org/www-project-web-security-testing-guide/)
- [MITRE ATT&CK — Discovery](https://attack.mitre.org/tactics/TA0007/)
