# Digital Forensics (Tools and Techniques for Evidence Acquisition and Analysis)

Practical reference for disk imaging, memory acquisition, file carving, timeline
analysis, metadata extraction, deleted file recovery, and chain of custody
procedures.

---

## 1. Disk Imaging

### dd (Basic)

```bash
# Standard forensic image (raw format)
dd if=/dev/sda of=/evidence/disk.raw bs=4096 conv=noerror,sync \
  status=progress

# With hash verification
dd if=/dev/sda bs=4096 conv=noerror,sync | tee /evidence/disk.raw | \
  sha256sum > /evidence/disk.raw.sha256

# Image a single partition
dd if=/dev/sda1 of=/evidence/sda1.raw bs=4096 conv=noerror,sync
```

### dc3dd (DoD Computer Forensics Lab)

```bash
# Forensic imaging with built-in hashing and logging
dc3dd if=/dev/sda of=/evidence/disk.raw \
  hash=sha256 \
  log=/evidence/imaging.log \
  hlog=/evidence/hash.log

# Split into multiple segments (2GB each)
dc3dd if=/dev/sda ofs=/evidence/disk.raw.000 \
  ofsz=2G hash=sha256 log=/evidence/imaging.log

# Wipe a drive (for preparation of evidence media)
dc3dd wipe=/dev/sdX pat=00
```

### dcfldd (DCFL dd)

```bash
# Imaging with hash verification and split output
dcfldd if=/dev/sda of=/evidence/disk.raw \
  hash=sha256 hashwindow=1G \
  hashlog=/evidence/hash.log \
  bs=4096 conv=noerror,sync \
  statusinterval=256

# Verify image after creation
dcfldd if=/dev/sda vf=/evidence/disk.raw verifylog=/evidence/verify.log
```

### ewfacquire (E01 Format)

```bash
# Create EnCase E01 format image (compressed, with metadata)
ewfacquire /dev/sda \
  -t /evidence/disk \
  -C "Case IR-2026-001" \
  -D "Suspect workstation HDD" \
  -e "Analyst Name" \
  -E "Case notes" \
  -f encase6 \
  -m fixed \
  -S 2G \
  -c best

# Verify E01 image
ewfverify /evidence/disk.E01

# Mount E01 for examination
ewfmount /evidence/disk.E01 /mnt/ewf
mount -o ro,loop,noexec /mnt/ewf/ewf1 /mnt/evidence
```

### Mounting Images

```bash
# Mount raw image read-only
mount -o ro,loop,noexec,nosuid,nodev /evidence/disk.raw /mnt/evidence

# Mount with offset (for partition within full disk image)
# Find offset with fdisk
fdisk -l /evidence/disk.raw
# Multiply start sector by sector size (usually 512)
# e.g., partition starts at sector 2048: 2048 * 512 = 1048576
mount -o ro,loop,offset=1048576,noexec /evidence/disk.raw /mnt/evidence

# Mount LVM inside image
losetup -r /dev/loop0 /evidence/disk.raw
kpartx -ar /dev/loop0
vgchange -ay
mount -o ro /dev/mapper/vg-root /mnt/evidence
```

---

## 2. Memory Acquisition

### LiME (Linux Memory Extractor)

```bash
# Build LiME module for target kernel
# (build on a system with matching kernel headers)
git clone https://github.com/504ensicsLabs/LiME.git
cd LiME/src
make KVER=$(uname -r)

# Acquire memory
sudo insmod lime-$(uname -r).ko \
  "path=/evidence/memory.lime format=lime"

# Alternative: dump to network
sudo insmod lime-$(uname -r).ko \
  "path=tcp:4444 format=lime"
# On collection host:
nc -l 4444 > /evidence/memory.lime

# Hash the dump
sha256sum /evidence/memory.lime > /evidence/memory.lime.sha256
```

### AVML (Microsoft)

```bash
# Download pre-built binary (no kernel headers needed)
wget https://github.com/microsoft/avml/releases/latest/download/avml

# Acquire memory
sudo ./avml /evidence/memory.lime

# Compressed output
sudo ./avml --compress /evidence/memory.lime.gz
```

### Volatility 3 (Analysis)

```bash
# Install Volatility 3
pip install volatility3

# Basic analysis commands
# Process listing
vol3 -f /evidence/memory.lime linux.pslist.PsList
vol3 -f /evidence/memory.lime linux.pstree.PsTree

# Network connections
vol3 -f /evidence/memory.lime linux.netstat.Netstat

# Open files
vol3 -f /evidence/memory.lime linux.lsof.Lsof

# Bash history from memory
vol3 -f /evidence/memory.lime linux.bash.Bash

# Loaded kernel modules
vol3 -f /evidence/memory.lime linux.lsmod.Lsmod

# Check for process injection
vol3 -f /evidence/memory.lime linux.malfind.Malfind

# Enumerate environment variables
vol3 -f /evidence/memory.lime linux.envars.Envars

# Extract files from memory
vol3 -f /evidence/memory.lime linux.pagecache.Files
vol3 -f /evidence/memory.lime -o /evidence/extracted/ \
  linux.pagecache.Files --dump
```

---

## 3. File Carving

### foremost

```bash
# Carve files from raw image
foremost -i /evidence/disk.raw -o /evidence/carved/ -v

# Carve specific file types only
foremost -t jpg,png,pdf,doc -i /evidence/disk.raw -o /evidence/carved/

# Carve from unallocated space
# First extract unallocated with blkls:
blkls /evidence/disk.raw > /evidence/unallocated.raw
foremost -i /evidence/unallocated.raw -o /evidence/carved_unalloc/

# Output: /evidence/carved/audit.txt lists all recovered files
```

### scalpel

```bash
# Configure /etc/scalpel/scalpel.conf
# Uncomment file types you want to carve

# Carve files
scalpel /evidence/disk.raw -o /evidence/scalpel_output/

# Carve with custom config
scalpel -c /path/to/custom_scalpel.conf \
  /evidence/disk.raw -o /evidence/carved/
```

### photorec

```bash
# Interactive file recovery (ncurses UI)
photorec /evidence/disk.raw

# Recover from partition
photorec /dev/sda1

# Output goes to recup_dir.1/, recup_dir.2/, etc.
```

### bulk_extractor

```bash
# Extract structured data (emails, URLs, credit cards, etc.)
bulk_extractor -o /evidence/bulk_output/ /evidence/disk.raw

# Key output files:
# email.txt         — email addresses found
# url.txt           — URLs found
# ccn.txt           — credit card numbers
# domain.txt        — domain names
# ip.txt            — IP addresses
# telephone.txt     — phone numbers
# find.txt          — search terms
```

---

## 4. Timeline Analysis

### plaso / log2timeline

```bash
# Generate super timeline from disk image
log2timeline.py /evidence/timeline.plaso /evidence/disk.raw

# Generate from mounted evidence
log2timeline.py /evidence/timeline.plaso /mnt/evidence/

# Filter and convert to CSV
psort.py -o l2tcsv /evidence/timeline.plaso \
  -w /evidence/timeline.csv

# Filter by date range
psort.py -o l2tcsv /evidence/timeline.plaso \
  "date > '2026-01-01 00:00:00' AND date < '2026-01-31 23:59:59'" \
  -w /evidence/timeline_january.csv

# Filter by source type
psort.py -o l2tcsv /evidence/timeline.plaso \
  "source_short == 'FILE'" \
  -w /evidence/timeline_files.csv

# Output to Elasticsearch for analysis
psort.py -o elastic /evidence/timeline.plaso \
  --server 127.0.0.1 --port 9200 --index_name case_timeline
```

### Manual Timeline with find/stat

```bash
# Filesystem MAC timeline (quick method)
find /mnt/evidence -printf '%T+ %m %u %g %s %p\n' 2>/dev/null | \
  sort > /evidence/timeline_mtime.txt

# Access times
find /mnt/evidence -printf '%A+ %m %u %g %s %p\n' 2>/dev/null | \
  sort > /evidence/timeline_atime.txt

# Change times (metadata change)
find /mnt/evidence -printf '%C+ %m %u %g %s %p\n' 2>/dev/null | \
  sort > /evidence/timeline_ctime.txt

# Combined timeline (requires The Sleuth Kit)
fls -r -m "/" /evidence/disk.raw > /evidence/bodyfile.txt
mactime -b /evidence/bodyfile.txt -d > /evidence/mac_timeline.csv
```

---

## 5. Hash Verification

```bash
# Generate hashes for evidence integrity
sha256sum /evidence/disk.raw > /evidence/disk.raw.sha256
md5sum /evidence/disk.raw > /evidence/disk.raw.md5

# Verify at any point during analysis
sha256sum -c /evidence/disk.raw.sha256
# Expected: /evidence/disk.raw: OK

# Hash individual files for IOC comparison
sha256sum /mnt/evidence/tmp/suspicious_binary
md5sum /mnt/evidence/tmp/suspicious_binary

# Recursive hash of directory
find /mnt/evidence/home/user/ -type f -exec sha256sum {} \; \
  > /evidence/user_files_hashes.txt

# Compare hashes against known-malware databases
# VirusTotal API
curl -s "https://www.virustotal.com/api/v3/files/<sha256_hash>" \
  -H "x-apikey: <YOUR_API_KEY>" | python3 -m json.tool

# NSRL (National Software Reference Library) hash lookup
# Download NSRL RDS from https://www.nist.gov/itl/ssd/software-quality-group/national-software-reference-library-nsrl
# Use hfind (The Sleuth Kit)
hfind -i nsrl-sha1 /path/to/NSRLFile.txt <sha1_hash>

# Hash sets for known-good comparison
# Create baseline hash set
find /usr/bin /usr/sbin /bin /sbin -type f -exec sha256sum {} \; \
  > /baseline/system_binaries.sha256

# Compare against evidence
diff <(sort /baseline/system_binaries.sha256) \
     <(find /mnt/evidence/usr/bin /mnt/evidence/usr/sbin \
       /mnt/evidence/bin /mnt/evidence/sbin -type f \
       -exec sha256sum {} \; | sed 's|/mnt/evidence||' | sort)
```

---

## 6. Deleted File Recovery

### extundelete (ext3/ext4)

```bash
# Recover all deleted files from ext4 partition
extundelete /evidence/sda1.raw --restore-all

# Recover specific file by inode
extundelete /evidence/sda1.raw --restore-inode 12345

# Recover files deleted after specific date
extundelete /evidence/sda1.raw --after $(date -d "2026-01-01" +%s) \
  --restore-all

# Output goes to RECOVERED_FILES/ directory
```

### photorec

```bash
# Recover deleted files (works across many filesystem types)
photorec /evidence/disk.raw

# Supports: ext2/3/4, NTFS, FAT, HFS+, and more
# Recovers: documents, images, videos, archives, databases
```

### The Sleuth Kit (TSK)

```bash
# List files including deleted (prefixed with *)
fls -r -d /evidence/disk.raw

# Recover specific file by inode
icat /evidence/disk.raw <inode> > /evidence/recovered_file

# File system stats
fsstat /evidence/disk.raw

# Search for file content in unallocated space
blkls /evidence/disk.raw | strings | grep -i "password\|secret\|key"

# Find deleted files by name pattern
fls -r -d /evidence/disk.raw | grep -i "\.pdf$\|\.docx$\|\.xlsx$"
# Then recover with icat using the inode number
```

---

## 7. Metadata Extraction

### exiftool

```bash
# Extract all metadata from a file
exiftool /evidence/document.pdf

# Extract metadata from all files in directory
exiftool -r /evidence/carved/

# Extract GPS coordinates from images
exiftool -gps* /evidence/photo.jpg

# Extract specific fields
exiftool -Author -CreateDate -ModifyDate -Producer /evidence/document.pdf

# Extract metadata in JSON format
exiftool -json /evidence/document.pdf > /evidence/metadata.json

# Remove metadata (for sanitization, not during investigation)
# exiftool -all= document.pdf

# Bulk metadata extraction to CSV
exiftool -csv -r /evidence/carved/ > /evidence/all_metadata.csv
```

### Additional Metadata Tools

```bash
# PDF metadata
pdfinfo /evidence/document.pdf
pdftotext /evidence/document.pdf /evidence/document.txt

# Office document metadata (python-oletools)
pip install oletools
oleid /evidence/document.docx
olevba /evidence/document.docx   # Extract VBA macros
olemeta /evidence/document.doc

# Image metadata
identify -verbose /evidence/image.jpg   # ImageMagick

# ELF binary metadata
readelf -a /evidence/suspicious_binary
file /evidence/suspicious_binary
strings /evidence/suspicious_binary | head -50

# Filesystem metadata
stat /mnt/evidence/path/to/file
# Shows: atime, mtime, ctime, inode, permissions, owner
```

---

## 8. Chain of Custody Documentation

### Evidence Collection Form

```
DIGITAL EVIDENCE COLLECTION FORM
=================================
Case Number:        IR-2026-_____
Evidence Number:    E-_____
Date/Time (UTC):    ____-__-__ __:__

ITEM DESCRIPTION
Device Type:        [ ] HDD  [ ] SSD  [ ] USB  [ ] Phone  [ ] Other: ____
Make/Model:         ________________________________
Serial Number:      ________________________________
Capacity:           ________________________________
Condition:          [ ] Powered on  [ ] Powered off  [ ] Damaged

ACQUISITION
Method:             [ ] dd  [ ] dc3dd  [ ] dcfldd  [ ] ewfacquire  [ ] LiME
Image Format:       [ ] Raw (.dd)  [ ] E01  [ ] AFF  [ ] LiME
Image File:         ________________________________
MD5 Hash:           ________________________________
SHA-256 Hash:       ________________________________
Write Blocker Used: [ ] Yes (Model: _____)  [ ] No (Justification: _____)

COLLECTED BY
Name:               ________________________________
Title:              ________________________________
Signature:          ________________________________
Date:               ________________________________
```

### Transfer Log

```
EVIDENCE TRANSFER LOG — Case IR-2026-_____
============================================
Evidence #   Date/Time     Released By      Received By      Purpose
-----------  -----------   ---------------  ---------------  ----------
E-001        ____-__-__    _______________  _______________  ___________
E-001        ____-__-__    _______________  _______________  ___________
```

### Best Practices

```bash
# Always verify image integrity at each stage
sha256sum /evidence/disk.raw
# Compare against original hash

# Use write blockers — always
# Hardware: Tableau/WiebeTech forensic bridges
# Software: blockdev --setro /dev/sdX (less reliable)

# Photograph evidence before handling
# Document: serial numbers, labels, damage, connections

# Store evidence in anti-static bags, in locked evidence locker
# Maintain temperature and humidity controls

# Create working copies — never analyze originals
cp /evidence/disk.raw /working/disk_copy.raw
sha256sum /working/disk_copy.raw  # verify copy matches
```

---

## Tips

- Never work directly on original evidence; always use verified copies.
- Document every action with timestamps, commands used, and results.
- Use write blockers for every physical acquisition without exception.
- Hash evidence at every transfer point and verify before/after analysis.
- Keep a detailed forensic notebook (physical or digital) for each case.
- Validate your tools regularly on test data with known outcomes.
- Be mindful of time zones; normalize all timestamps to UTC in reports.
- Maintain tool versions in your reports; forensic results must be reproducible.
- If evidence might be used in legal proceedings, consult with legal counsel
  before beginning analysis.
- Keep forensic workstations isolated from the network during analysis.

---

## See Also

- incident-response, log-analysis, auditd, threat-hunting, cryptography

## References

- [NIST SP 800-86 — Guide to Integrating Forensic Techniques into Incident Response](https://csrc.nist.gov/publications/detail/sp/800-86/final)
- [NIST SP 800-72 — Guidelines on PDA Forensics](https://csrc.nist.gov/publications/detail/sp/800-72/final)
- [SANS Digital Forensics and Incident Response](https://www.sans.org/digital-forensics-incident-response/)
- [The Sleuth Kit / Autopsy](https://www.sleuthkit.org/)
- [Volatility 3 Documentation](https://volatility3.readthedocs.io/)
- [LiME — Linux Memory Extractor](https://github.com/504ensicsLabs/LiME)
- [AVML — Acquire Volatile Memory for Linux](https://github.com/microsoft/avml)
- [plaso / log2timeline](https://plaso.readthedocs.io/)
- [ExifTool Documentation](https://exiftool.org/)
- [NSRL — National Software Reference Library](https://www.nist.gov/itl/ssd/software-quality-group/national-software-reference-library-nsrl)
- [SWGDE Best Practices for Digital Evidence](https://www.swgde.org/)
