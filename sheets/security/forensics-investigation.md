# Digital Forensics & Investigation

> Systematic identification, preservation, collection, examination, analysis, and presentation of digital evidence following legally defensible procedures to support investigations and legal proceedings.

## Forensics Process (NIST SP 800-86)

```
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Identification│──>│ Preservation │──>│  Collection  │
│              │   │              │   │              │
│ Detect event │   │ Secure scene │   │ Acquire data │
│ Determine    │   │ Prevent      │   │ Forensic     │
│   scope      │   │   alteration │   │   imaging    │
└──────────────┘   └──────────────┘   └──────────────┘
        │                                     │
        ▼                                     ▼
┌──────────────┐   ┌──────────────┐   ┌──────────────┐
│ Presentation │<──│   Analysis   │<──│ Examination  │
│              │   │              │   │              │
│ Report       │   │ Correlate    │   │ Extract data │
│ Testimony    │   │ Timeline     │   │ Filter       │
│ Document     │   │ Reconstruct  │   │ Parse        │
└──────────────┘   └──────────────┘   └──────────────┘
```

## Evidence Types

```
Type            Definition                       Example
────            ──────────                       ───────
Real            Physical objects                 Hard drive, laptop,
(Physical)      tangible evidence                USB drive, phone

Documentary     Written/recorded info            Logs, emails,
                (must be authenticated)          contracts, screenshots

Demonstrative   Illustrative aids                Timeline diagrams,
                (not direct evidence)            network maps, charts

Testimonial     Witness/expert statements        Expert witness on
                (oral or written)                forensic methodology

Direct          Proves fact without inference    Eyewitness, video of
                                                 attack in progress

Circumstantial  Requires inference               Login times correlating
                                                 with unauthorized access
```

## Evidence Handling

### Order of Volatility (Most to Least)

```
Priority  Source                    Volatility   Method
────────  ──────                    ──────────   ──────
1         CPU registers/cache       Seconds      Live capture tools
2         Routing table, ARP,       Seconds      netstat, arp, ip
          process table
3         Main memory (RAM)         Minutes      Memory dump (LiME,
                                                 WinPMEM, FTK Imager)
4         Temporary filesystems     Minutes      Copy /tmp, swap
5         Disk                      Persistent   Forensic image
6         Remote logging/monitoring Persistent   Collect from SIEM
7         Physical config/topology  Persistent   Document, photograph
8         Archival media            Persistent   Tapes, backups
```

### Write Blockers

```
# Prevent any writes to evidence drive during acquisition

Hardware Write Blockers:
- Tableau T35689iu (USB 3.0, SATA, IDE)
- WiebeTech USB WriteBlocker
- CRU Forensic UltraDock

Software Write Blockers:
- Linux: mount -o ro,noexec,noatime /dev/sdX /mnt/evidence
- Windows: Registry HKLM\SYSTEM\CurrentControlSet\Control\StorageDevicePolicies
           WriteProtect = 1

# IMPORTANT: Hardware write blockers are preferred for legal defensibility
# Software-only methods may be challenged in court
```

### Forensic Imaging

```bash
# Create bit-for-bit forensic image

# dd (basic — no hashing, no logging)
dd if=/dev/sda of=/evidence/case001.dd bs=4M status=progress

# dc3dd (forensic-grade dd with hashing)
dc3dd if=/dev/sda of=/evidence/case001.dd \
  hash=sha256 \
  log=/evidence/case001.log \
  hlog=/evidence/case001.hashlog

# FTK Imager (command line — E01 format)
ftkimager /dev/sda /evidence/case001 \
  --e01 \
  --case-number CASE-2026-001 \
  --evidence-number E001 \
  --examiner "J. Smith" \
  --description "Suspect workstation HDD" \
  --verify

# Guymager (GUI — Linux)
# Supports E01, EWF, AFF, dd formats
# Built-in hash verification

# Image formats
# Raw (dd):  Bit-for-bit copy, largest, simplest
# E01 (EWF): Expert Witness Format — compressed, metadata, checksums
# AFF4:      Advanced Forensics Format — modern, extensible
# VMDK:      Virtual machine disk — useful for VM forensics

# Verify image integrity
sha256sum /evidence/case001.dd
# Compare against hash taken at acquisition time
```

### Chain of Custody

```
# Required documentation for every piece of evidence

Evidence Tag:
┌─────────────────────────────────────────────────┐
│ Case Number: CASE-2026-001                      │
│ Evidence Number: E001                           │
│ Description: Dell Latitude 5540, S/N: XXXXX     │
│ Date/Time Collected: 2026-04-05 14:30 UTC       │
│ Collected By: Investigator J. Smith, Badge #123 │
│ Location Found: Office 302, Desk 4B             │
│ Condition: Powered on, screen locked            │
│ Hash (at acquisition): SHA-256: a1b2c3d4...     │
└─────────────────────────────────────────────────┘

# Chain of Custody Log
Date/Time    From          To            Purpose       Signature
─────────    ────          ──            ───────       ─────────
2026-04-05   Scene         Evidence      Collection    J. Smith
  14:30      (Office 302)  Locker #7
2026-04-06   Evidence      Forensic      Imaging       J. Smith →
  09:00      Locker #7     Lab                         K. Jones
2026-04-06   Forensic      Evidence      Return after  K. Jones
  17:00      Lab           Locker #7     imaging

# Every transfer recorded — unbroken chain required for court
```

## Disk Forensics

### Sleuth Kit + Autopsy

```bash
# Autopsy = GUI frontend for The Sleuth Kit (TSK)

# TSK command-line tools
# List partitions in image
mmls case001.dd

# List files in partition
fls -r -o 2048 case001.dd          # -o = partition offset

# Display file metadata
istat case001.dd -o 2048 12345     # inode 12345

# Extract file by inode
icat case001.dd -o 2048 12345 > extracted_file.doc

# Search for deleted files
fls -d -r -o 2048 case001.dd      # -d = deleted only

# Keyword search across image
srch_strings -a case001.dd | grep -i "password"

# Timeline generation
fls -r -m "/" -o 2048 case001.dd > bodyfile.txt
mactime -b bodyfile.txt -d > timeline.csv

# File carving (recover files by header/footer signatures)
foremost -i case001.dd -o /evidence/carved/
scalpel -c /etc/scalpel/scalpel.conf -o /evidence/carved/ case001.dd
photorec case001.dd    # interactive file recovery
```

### Key Forensic Artifacts by OS

```
Windows:
  Registry hives     %SYSTEMROOT%\System32\config\
                     NTUSER.DAT (per user)
  Event logs         %SYSTEMROOT%\System32\winevt\Logs\
  Prefetch           %SYSTEMROOT%\Prefetch\*.pf
  Amcache            %SYSTEMROOT%\AppCompat\Programs\Amcache.hve
  ShimCache          SYSTEM\CurrentControlSet\Control\Session
                     Manager\AppCompatCache
  MFT                $MFT (NTFS Master File Table)
  USN Journal        $UsnJrnl (file change journal)
  $I30 index         Directory listing (including deleted entries)
  Recycle Bin        $Recycle.Bin\<SID>\$I* and $R*
  Browser history    AppData\Local\<Browser>\
  Jump Lists         AppData\Roaming\Microsoft\Windows\Recent\

Linux:
  Auth logs          /var/log/auth.log, /var/log/secure
  Syslog             /var/log/syslog, /var/log/messages
  Bash history       ~/.bash_history
  Cron logs          /var/log/cron
  Package logs       /var/log/apt/, /var/log/yum.log
  Login records      /var/log/wtmp, /var/log/btmp, /var/run/utmp
  Systemd journal    /var/log/journal/
  /tmp contents      Temporary files (may contain attacker tools)
  /etc/passwd,shadow User accounts
  SSH keys           ~/.ssh/authorized_keys

macOS:
  Unified log        /var/db/diagnostics/
  FSEvents           /.fseventsd/ (file system changes)
  Spotlight          /.Spotlight-V100/ (indexed metadata)
  Quarantine DB      ~/Library/Preferences/com.apple.LaunchServices
  KnowledgeC.db      User activity database
  TCC.db             Privacy permission grants
```

## Memory Forensics

### Memory Acquisition

```bash
# Linux — LiME (Linux Memory Extractor)
# Load kernel module
insmod lime.ko "path=/evidence/mem.lime format=lime"

# Or via /dev/mem (older systems)
dd if=/dev/mem of=/evidence/mem.dd bs=1M

# Windows — WinPMEM
winpmem_mini_x64.exe /evidence/mem.raw

# Windows — FTK Imager
# File → Capture Memory → select output path

# macOS — osxpmem
osxpmem -o /evidence/mem.raw
```

### Volatility Analysis

```bash
# Volatility 3 (Python 3)

# Identify OS profile
vol -f mem.raw banners.Banners

# Process listing
vol -f mem.raw windows.pslist.PsList
vol -f mem.raw windows.pstree.PsTree
vol -f mem.raw windows.psscan.PsScan    # find hidden processes

# Network connections
vol -f mem.raw windows.netscan.NetScan
vol -f mem.raw windows.netstat.NetStat

# DLL listing
vol -f mem.raw windows.dlllist.DllList --pid 1234

# Command history
vol -f mem.raw windows.cmdline.CmdLine
vol -f mem.raw windows.consoles.Consoles

# Registry analysis
vol -f mem.raw windows.registry.hivelist.HiveList
vol -f mem.raw windows.registry.printkey.PrintKey \
  --key "Software\Microsoft\Windows\CurrentVersion\Run"

# File extraction from memory
vol -f mem.raw windows.filescan.FileScan
vol -f mem.raw windows.dumpfiles.DumpFiles --pid 1234

# Malware detection
vol -f mem.raw windows.malfind.Malfind
vol -f mem.raw windows.vadinfo.VadInfo --pid 1234

# Linux memory analysis
vol -f mem.lime linux.pslist.PsList
vol -f mem.lime linux.bash.Bash
vol -f mem.lime linux.check_syscall.Check_syscall
vol -f mem.lime linux.proc.Maps --pid 1234
```

## Network Forensics

### Packet Capture

```bash
# Full packet capture
tcpdump -i eth0 -w /evidence/capture.pcap -C 100 -W 50
# -C 100 = rotate at 100 MB, -W 50 = max 50 files

# Capture with ring buffer (overwrite oldest)
tcpdump -i eth0 -w /evidence/capture.pcap -C 100 -W 10 -Z root

# Targeted capture
tcpdump -i eth0 -w /evidence/exfil.pcap \
  'host 10.0.0.50 and (port 443 or port 8443)'

# Analysis with tshark
tshark -r capture.pcap -Y "http.request" -T fields \
  -e ip.src -e http.host -e http.request.uri

# Extract files from capture
tshark -r capture.pcap --export-objects "http,/evidence/http_objects/"

# DNS query analysis
tshark -r capture.pcap -Y "dns.qr == 0" -T fields \
  -e ip.src -e dns.qry.name | sort | uniq -c | sort -rn
```

### NetFlow Analysis

```bash
# NetFlow records: src/dst IP, ports, bytes, packets, timestamps
# Lower storage than full pcap; good for connection analysis

# nfdump analysis
nfdump -r /var/flow/2026/04/05/nfcapd.202604051400 \
  -s srcip/bytes    # top source IPs by bytes

# Find large data transfers (potential exfiltration)
nfdump -r flows.nfcapd \
  'bytes > 100000000' \
  -o extended

# Connection analysis to suspicious IP
nfdump -r flows.nfcapd \
  'dst ip 203.0.113.100' \
  -o long
```

## Log Forensics

```bash
# Centralized log analysis for investigation

# Search for failed logins
grep "Failed password" /var/log/auth.log | \
  awk '{print $11}' | sort | uniq -c | sort -rn | head -20

# SSH login timeline
grep "Accepted" /var/log/auth.log | \
  awk '{print $1,$2,$3,$9,$11}'

# Windows Event Log analysis (on Linux)
python3 -m evtx_dump Security.evtx | \
  grep -A 5 "EventID.*4625"    # Failed logon

# Key Windows Event IDs for investigation
# 4624  Successful logon
# 4625  Failed logon
# 4648  Logon with explicit credentials (runas)
# 4672  Special privileges assigned (admin logon)
# 4688  New process created
# 4697  Service installed
# 4698  Scheduled task created
# 4720  User account created
# 4732  User added to group
# 7045  New service installed (System log)
# 1102  Audit log cleared (Security log)

# Syslog parsing
journalctl --since "2026-04-05 00:00" --until "2026-04-05 23:59" \
  -u sshd --no-pager | grep -i "fail\|error\|denied"

# Web server log analysis
awk '{print $1}' /var/log/nginx/access.log | \
  sort | uniq -c | sort -rn | head -20    # top IPs

# Find SQL injection attempts
grep -iE "(union.*select|or.*1=1|drop.*table|';)" \
  /var/log/nginx/access.log
```

## Mobile Forensics

```
Acquisition Methods:
  Manual:     Screen interaction, screenshots, photos
  Logical:    File system copy via backup tools (iTunes, ADB)
  File System: Root/jailbreak access, full file system copy
  Physical:   Chip-off, JTAG/ISP — bit-for-bit image
  Cloud:      iCloud, Google account data extraction

# Android
adb backup -all -f /evidence/android_backup.ab
adb pull /data/data/ /evidence/app_data/
# Root required for full file system access

# iOS
# Logical acquisition via iTunes/Finder backup
# Backup location: ~/Library/Application Support/MobileSync/Backup/
# Encrypted backups contain more data (keychain, health, etc.)

# Tools
# Cellebrite UFED      (commercial, comprehensive)
# MSAB XRY             (commercial)
# Magnet AXIOM         (commercial)
# Andriller            (open source, Android)
# iLEAPP / ALEAPP      (open source, iOS/Android log parsers)
```

## Cloud Forensics

```
# Challenges
# - No physical access to hardware
# - Shared infrastructure (multi-tenant)
# - Data jurisdiction issues
# - Volatile instances (auto-scaling, serverless)
# - Provider cooperation required for some data
# - Encryption key management

# AWS forensics
# Snapshot EC2 volume
aws ec2 create-snapshot --volume-id vol-xxxxx \
  --description "Forensic snapshot CASE-001"

# Copy snapshot to forensics account
aws ec2 copy-snapshot --source-region us-east-1 \
  --source-snapshot-id snap-xxxxx \
  --destination-region us-east-1 \
  --description "Forensic copy"

# Attach to forensic workstation (read-only)
aws ec2 attach-volume --volume-id vol-forensic \
  --instance-id i-forensic-ws --device /dev/xvdf

# CloudTrail log analysis (API call history)
aws cloudtrail lookup-events \
  --lookup-attributes AttributeKey=Username,AttributeValue=suspect_user \
  --start-time 2026-04-01 --end-time 2026-04-05

# Azure forensics — disk snapshot
az snapshot create --resource-group forensics-rg \
  --name forensic-snap-001 \
  --source /subscriptions/.../disks/suspect-disk

# GCP forensics — disk snapshot
gcloud compute disks snapshot suspect-disk \
  --snapshot-names forensic-snap-001 \
  --zone us-central1-a
```

## Anti-Forensics Techniques

```
Technique          Method                    Detection
─────────          ──────                    ─────────
Data wiping        Secure erase, shred,      Compare allocated vs
                   overwrite tools           unallocated space ratios
Timestomping       Modify file timestamps    Compare $MFT vs $STDINFO,
                   ($STDINFO only)           $FILENAME timestamps differ
Log clearing       Delete/truncate logs      Gaps in log sequence,
                                             Event ID 1102 (audit cleared)
Steganography      Hide data in images,      Statistical analysis,
                   audio, video              steganalysis tools
Encryption         Full disk, file-level     Cannot defeat without key;
                   encryption                legal compulsion varies
Process hiding     Rootkits, DKOM            Memory forensics, cross-
                                             reference /proc vs kernel
Trail obfuscation  VPN, Tor, proxy chains    Timing analysis, endpoint
                                             forensics
Fileless malware   Memory-only, PowerShell,  Memory forensics, ETW,
                   living-off-the-land       process monitoring
Artifact removal   Clear browser history,    Recovery from slack space,
                   shred configs             volume shadow copies
```

## Expert Witness Testimony

```
# Daubert Standard (US Federal Courts)
# Judge determines if expert testimony is admissible based on:
# 1. Testing: Has the theory/technique been tested?
# 2. Peer review: Has it been subjected to peer review?
# 3. Error rate: What is the known or potential error rate?
# 4. Standards: Are there standards controlling its operation?
# 5. General acceptance: Is it generally accepted in the field?

# Expert witness responsibilities
- Qualified by education, training, experience
- Methodology must be scientifically sound
- Opinions based on sufficient facts/data
- Maintain objectivity (not an advocate)
- Explain technical concepts to non-technical audience
- Withstand cross-examination
- Document methodology reproducibly

# Report requirements
- Statement of qualifications
- Materials reviewed
- Methodology used
- Findings of fact
- Opinions and conclusions
- Basis for each opinion
```

## See Also

- forensics
- incident-response
- log-analysis
- security-operations
- threat-hunting
- siem

## References

- NIST SP 800-86: Guide to Integrating Forensic Techniques into Incident Response
- NIST SP 800-101r1: Guidelines on Mobile Device Forensics
- RFC 3227: Guidelines for Evidence Collection and Archiving
- SWGDE: Scientific Working Group on Digital Evidence
- ISO/IEC 27037: Guidelines for Digital Evidence Identification, Collection, Acquisition
- ACPO Good Practice Guide for Digital Evidence
- Sleuth Kit: https://www.sleuthkit.org/
- Volatility Foundation: https://volatilityfoundation.org/
