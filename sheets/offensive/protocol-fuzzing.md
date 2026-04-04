# Protocol Fuzzing (Protocol-Specific Fuzzing Methodology)

> For authorized security testing, CTF competitions, and educational purposes only.

Protocol fuzzing systematically mutates network protocol messages to discover parsing
vulnerabilities, state machine errors, and memory corruption in protocol implementations.
This covers wire format mutation strategies, stateful fuzzing with session tracking,
grammar-based generation, custom harness design, corpus construction, and crash triage
using tools like Boofuzz, Peach Fuzzer, and Scapy.

---

## Wire Format Mutation Strategies

### Basic Mutation Types

```bash
# Field overflow — exceed expected sizes
# Integer fields: set to 0, 1, MAX-1, MAX, MAX+1
python3 << 'EOF'
import struct
# 2-byte length field mutations
mutations = [
    struct.pack('>H', 0),        # zero length
    struct.pack('>H', 1),        # minimum
    struct.pack('>H', 0xFFFE),   # near max
    struct.pack('>H', 0xFFFF),   # max value
    b'\x00\x00\x01',             # 3-byte overflow of 2-byte field
]
# String field mutations
string_mutations = [
    b'',                          # empty
    b'A' * 256,                   # overflow
    b'A' * 65536,                 # large overflow
    b'\x00',                      # null byte
    b'%s%s%s%s%s',                # format string
    b'\xff' * 100,                # non-ASCII
]
EOF

# Version confusion — send unexpected protocol versions
python3 << 'EOF'
# HTTP version mutations
version_mutations = [
    b'HTTP/0.9',       # ancient version
    b'HTTP/1.0',       # old version
    b'HTTP/1.1',       # standard
    b'HTTP/2.0',       # newer (wrong format — should be HTTP/2)
    b'HTTP/3.0',       # future version
    b'HTTP/99.99',     # invalid version
    b'HTTP/',          # incomplete
    b'JUNK/1.1',       # wrong protocol
]
EOF

# Truncation — send partial messages
python3 << 'EOF'
# Send progressively truncated packets
full_packet = b'\x01\x02\x03\x04\x05\x06\x07\x08'
for i in range(len(full_packet)):
    truncated = full_packet[:i]
    # send(truncated)  — each truncation tests bounds checking
EOF
```

### CRC and Checksum Manipulation

```bash
# CRC collision — valid checksum with mutated payload
python3 << 'EOF'
import binascii

def crc32_forge(data, target_crc, offset):
    """Modify 4 bytes at offset to achieve target CRC32"""
    # Calculate CRC up to modification point
    pre = binascii.crc32(data[:offset]) & 0xffffffff
    # Calculate CRC from modification point to end
    post_data = data[offset+4:]
    # Solve for the 4 bytes that produce target CRC
    # (simplified — real implementation uses CRC algebra)
    return data[:offset] + b'\x00\x00\x00\x00' + post_data

# Checksum mutations
def fuzz_checksum(packet, checksum_offset, checksum_size):
    mutations = []
    # Valid checksum with mutated data
    mutations.append(('valid_crc_bad_data', mutate_data_fix_crc(packet)))
    # Invalid checksum with valid data
    mutations.append(('bad_crc_valid_data', flip_checksum(packet)))
    # Zero checksum
    mutations.append(('zero_crc', zero_checksum(packet)))
    return mutations
EOF
```

### Bit-Level Mutations

```bash
# Bit flipping across protocol headers
python3 << 'EOF'
def bitflip_mutator(data, header_len):
    """Flip each bit in protocol header one at a time"""
    mutations = []
    for byte_idx in range(min(header_len, len(data))):
        for bit_idx in range(8):
            mutated = bytearray(data)
            mutated[byte_idx] ^= (1 << bit_idx)
            mutations.append(bytes(mutated))
    return mutations

# Byte-level mutations
def byte_mutator(data, header_len):
    """Replace each header byte with interesting values"""
    interesting = [0x00, 0x01, 0x7f, 0x80, 0xfe, 0xff]
    mutations = []
    for idx in range(min(header_len, len(data))):
        for val in interesting:
            mutated = bytearray(data)
            mutated[idx] = val
            mutations.append(bytes(mutated))
    return mutations
EOF
```

---

## Stateful Fuzzing

### Session Tracking

```bash
# Stateful protocol fuzzer skeleton
python3 << 'EOF'
import socket
import time

class StatefulFuzzer:
    def __init__(self, host, port):
        self.host = host
        self.port = port
        self.state = 'INIT'
        self.session_data = {}

    def connect(self):
        self.sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        self.sock.settimeout(5)
        self.sock.connect((self.host, self.port))
        self.state = 'CONNECTED'

    def handshake(self):
        """Complete protocol handshake before fuzzing"""
        # Send legitimate handshake
        self.sock.send(self.build_handshake())
        resp = self.sock.recv(4096)
        self.session_data['session_id'] = self.parse_session_id(resp)
        self.state = 'AUTHENTICATED'

    def fuzz_authenticated_endpoint(self, mutations):
        """Fuzz messages that require prior authentication"""
        for mutation in mutations:
            try:
                self.connect()
                self.handshake()  # establish valid session first
                self.sock.send(mutation)  # then send mutated message
                resp = self.sock.recv(4096)
                self.log_result(mutation, resp)
            except Exception as e:
                self.log_crash(mutation, e)
            finally:
                self.sock.close()
EOF
```

### Protocol State Machine

```bash
# Define protocol state machine for fuzzing
python3 << 'EOF'
from enum import Enum, auto

class ProtocolState(Enum):
    INIT = auto()
    HANDSHAKE = auto()
    AUTHENTICATED = auto()
    DATA_TRANSFER = auto()
    CLOSING = auto()
    ERROR = auto()

# State transition table
transitions = {
    ProtocolState.INIT: {
        'connect': ProtocolState.HANDSHAKE,
    },
    ProtocolState.HANDSHAKE: {
        'hello_ok': ProtocolState.AUTHENTICATED,
        'hello_fail': ProtocolState.ERROR,
    },
    ProtocolState.AUTHENTICATED: {
        'request': ProtocolState.DATA_TRANSFER,
        'logout': ProtocolState.CLOSING,
    },
    ProtocolState.DATA_TRANSFER: {
        'response': ProtocolState.AUTHENTICATED,
        'error': ProtocolState.ERROR,
    },
}

# Fuzz transitions: send messages valid for OTHER states
# (e.g., send DATA_TRANSFER message during HANDSHAKE)
def generate_state_confusion_tests(current_state, all_messages):
    """Generate messages that are invalid for the current state"""
    valid_messages = transitions.get(current_state, {}).keys()
    return [msg for msg in all_messages if msg not in valid_messages]
EOF
```

---

## Boofuzz Framework

### Basic Boofuzz Setup

```bash
# Install boofuzz
pip install boofuzz

# Basic TCP protocol fuzzer
python3 << 'EOF'
from boofuzz import *

def main():
    session = Session(
        target=Target(
            connection=SocketConnection("target.local", 9999, proto='tcp')
        ),
        fuzz_loggers=[FuzzLoggerText()],
        crash_threshold_request=10,      # stop after 10 crashes per request
        crash_threshold_element=3,       # stop after 3 crashes per element
    )

    # Define protocol message structure
    s_initialize("login")
    s_string("USER", fuzzable=False)
    s_delim(" ", fuzzable=False)
    s_string("admin", name="username")    # fuzz this field
    s_static("\r\n")
    s_string("PASS", fuzzable=False)
    s_delim(" ", fuzzable=False)
    s_string("password", name="password")  # fuzz this field
    s_static("\r\n")

    # Connect request blocks
    session.connect(s_get("login"))

    # Start fuzzing
    session.fuzz()

if __name__ == "__main__":
    main()
EOF
```

### Multi-Stage Boofuzz Session

```bash
# Multi-stage protocol with dependencies
python3 << 'EOF'
from boofuzz import *

session = Session(
    target=Target(connection=SocketConnection("target.local", 8080, proto='tcp')),
    fuzz_loggers=[FuzzLoggerText(), FuzzLoggerCsv(file_handle=open("results.csv","w"))],
)

# Stage 1: Handshake
s_initialize("handshake")
s_bytes(b"\x01", name="msg_type", fuzzable=False)
s_size("payload", length=2, endian=BIG_ENDIAN, fuzzable=True)
s_block_start("payload")
s_bytes(b"\x00\x01", name="version")    # version field — fuzz
s_string("ClientHello", name="client_id")
s_block_end()

# Stage 2: Authentication (depends on handshake)
s_initialize("auth")
s_bytes(b"\x02", fuzzable=False)
s_size("auth_data", length=2, endian=BIG_ENDIAN, fuzzable=True)
s_block_start("auth_data")
s_string("admin", name="username", max_len=256)
s_string("secret", name="token", max_len=1024)
s_block_end()

# Stage 3: Command (depends on auth)
s_initialize("command")
s_bytes(b"\x03", fuzzable=False)
s_size("cmd_data", length=4, endian=BIG_ENDIAN, fuzzable=True)
s_block_start("cmd_data")
s_byte(0x01, name="cmd_id")
s_dword(0, name="param1", endian=BIG_ENDIAN)
s_string("", name="param2", max_len=4096)
s_block_end()

# Define state transitions
session.connect(s_get("handshake"))
session.connect(s_get("handshake"), s_get("auth"))
session.connect(s_get("auth"), s_get("command"))

session.fuzz()
EOF
```

### Boofuzz Process Monitoring

```bash
# Monitor target process for crashes
python3 << 'EOF'
from boofuzz import *

# Process monitor (runs on target machine)
# Start: process_monitor_unix.py -c target_config.py

session = Session(
    target=Target(
        connection=SocketConnection("target.local", 9999),
        monitors=[
            ProcessMonitor("target.local", 26002),  # monitor RPC port
        ],
    ),
)

# Network monitor — capture traffic for replay
session = Session(
    target=Target(
        connection=SocketConnection("target.local", 9999),
        monitors=[
            NetworkMonitor("/tmp/fuzzing_pcaps/", "eth0"),
        ],
    ),
)
EOF
```

---

## Scapy Protocol Mutation

### Custom Protocol Fuzzing with Scapy

```bash
# Define custom protocol layer and fuzz it
python3 << 'EOF'
from scapy.all import *
import random

# Define custom binary protocol
class CustomProto(Packet):
    name = "CustomProtocol"
    fields_desc = [
        ByteField("version", 1),
        ByteEnumField("msg_type", 0, {0: "hello", 1: "data", 2: "bye"}),
        ShortField("length", None),
        IntField("session_id", 0),
        StrLenField("payload", b"", length_from=lambda pkt: pkt.length),
    ]
    def post_build(self, pkt, pay):
        if self.length is None:
            pkt = pkt[:2] + struct.pack(">H", len(self.payload)) + pkt[4:]
        return pkt + pay

# Mutation-based fuzzer
def mutate_packet(base_pkt):
    """Apply random mutations to a base packet"""
    raw = bytearray(bytes(base_pkt))
    mutation_type = random.choice(['bitflip', 'byteflip', 'insert', 'delete', 'havoc'])

    if mutation_type == 'bitflip':
        pos = random.randint(0, len(raw)-1)
        raw[pos] ^= (1 << random.randint(0, 7))
    elif mutation_type == 'byteflip':
        pos = random.randint(0, len(raw)-1)
        raw[pos] = random.randint(0, 255)
    elif mutation_type == 'insert':
        pos = random.randint(0, len(raw))
        raw.insert(pos, random.randint(0, 255))
    elif mutation_type == 'delete' and len(raw) > 1:
        pos = random.randint(0, len(raw)-1)
        del raw[pos]
    elif mutation_type == 'havoc':
        for _ in range(random.randint(1, 10)):
            pos = random.randint(0, len(raw)-1)
            raw[pos] = random.randint(0, 255)

    return bytes(raw)

# Fuzz loop
base = CustomProto(version=1, msg_type=1, session_id=0x41414141, payload=b"test_data")
for i in range(10000):
    mutated = mutate_packet(base)
    send(IP(dst="target.local")/TCP(dport=9999)/Raw(mutated), verbose=0)
EOF
```

### Protocol-Aware Scapy Fuzzing

```bash
# Fuzz specific protocol fields with Scapy's fuzz()
python3 << 'EOF'
from scapy.all import *

# DNS fuzzing
for i in range(1000):
    pkt = IP(dst="target.local")/UDP(dport=53)/fuzz(DNS())
    send(pkt, verbose=0)

# TLS ClientHello fuzzing
from scapy.layers.tls.handshake import TLSClientHello
from scapy.layers.tls.record import TLS

for i in range(1000):
    pkt = IP(dst="target.local")/TCP(dport=443)/fuzz(TLS()/TLSClientHello())
    send(pkt, verbose=0)

# Custom field-targeted fuzzing
class SmartFuzzer:
    def __init__(self, target_ip, target_port):
        self.target = target_ip
        self.port = target_port

    def fuzz_length_fields(self, base_pkt, length_offsets):
        """Specifically target length fields with boundary values"""
        interesting_lengths = [0, 1, 0x7f, 0x80, 0xff, 0x100, 0x7fff, 0x8000, 0xffff]
        raw = bytearray(bytes(base_pkt))
        for offset in length_offsets:
            for length in interesting_lengths:
                mutated = bytearray(raw)
                struct.pack_into('>H', mutated, offset, length & 0xffff)
                send(IP(dst=self.target)/TCP(dport=self.port)/Raw(bytes(mutated)), verbose=0)
EOF
```

---

## Grammar-Based Generation

### Protocol Grammar Definition

```bash
# Grammar-based protocol message generation
python3 << 'EOF'
import random
import struct

class ProtocolGrammar:
    """Generate valid-structure messages with fuzzed values"""

    def __init__(self):
        self.rules = {
            'message': ['header', 'body'],
            'header': ['magic', 'version', 'type', 'length'],
            'body': ['field*'],  # zero or more fields
            'field': ['field_type', 'field_length', 'field_data'],
        }

        self.generators = {
            'magic': lambda: struct.pack('>I', 0xDEADBEEF),
            'version': lambda: struct.pack('>H', random.choice([1, 2, 0, 0xFFFF, 3])),
            'type': lambda: struct.pack('B', random.randint(0, 255)),
            'length': None,  # calculated after body generation
            'field_type': lambda: struct.pack('B', random.randint(0, 10)),
            'field_length': None,  # calculated per field
            'field_data': lambda: self._fuzz_data(),
        }

    def _fuzz_data(self):
        strategies = [
            lambda: b'A' * random.randint(0, 1024),
            lambda: bytes(random.randint(0, 255) for _ in range(random.randint(1, 256))),
            lambda: b'\x00' * random.randint(1, 100),
            lambda: b'%s' * random.randint(1, 20),
            lambda: struct.pack('>Q', random.choice([0, 1, 0x7FFFFFFFFFFFFFFF, 0xFFFFFFFFFFFFFFFF])),
        ]
        return random.choice(strategies)()

    def generate(self):
        """Generate a complete fuzzed message"""
        magic = self.generators['magic']()
        version = self.generators['version']()
        msg_type = self.generators['type']()

        # Generate 0-5 fields
        fields = b''
        for _ in range(random.randint(0, 5)):
            ftype = self.generators['field_type']()
            fdata = self._fuzz_data()
            flength = struct.pack('>H', len(fdata))
            fields += ftype + flength + fdata

        length = struct.pack('>I', len(fields))
        return magic + version + msg_type + length + fields

grammar = ProtocolGrammar()
for i in range(10000):
    msg = grammar.generate()
    # send to target
EOF
```

---

## Corpus Construction

### Building Initial Corpus

```bash
# Capture legitimate traffic for corpus
tcpdump -i eth0 -w corpus_capture.pcap host target.local

# Extract protocol payloads from PCAP
python3 << 'EOF'
from scapy.all import rdpcap
import os

os.makedirs('corpus', exist_ok=True)
packets = rdpcap('corpus_capture.pcap')
for i, pkt in enumerate(packets):
    if pkt.haslayer('TCP') and pkt.haslayer('Raw'):
        payload = bytes(pkt['Raw'])
        if len(payload) > 0:
            with open(f'corpus/pkt_{i:06d}.bin', 'wb') as f:
                f.write(payload)
print(f"Extracted {i+1} payloads")
EOF

# Minimize corpus (remove redundant inputs)
# AFL-style: keep only inputs that trigger new coverage
afl-cmin -i corpus/ -o corpus_min/ -- ./target_parser @@

# Create seed files from protocol specification
python3 << 'EOF'
import struct
# Generate edge-case seed files from spec
seeds = {
    'minimal_valid': struct.pack('>IHB', 0xDEADBEEF, 1, 0),
    'max_version': struct.pack('>IHB', 0xDEADBEEF, 0xFFFF, 0),
    'all_msg_types': b''.join(struct.pack('>IHB', 0xDEADBEEF, 1, t) for t in range(256)),
    'empty_payload': struct.pack('>IHBI', 0xDEADBEEF, 1, 1, 0),
    'max_payload': struct.pack('>IHBI', 0xDEADBEEF, 1, 1, 0xFFFFFFFF) + b'A' * 1000,
}
for name, data in seeds.items():
    with open(f'corpus/{name}.bin', 'wb') as f:
        f.write(data)
EOF
```

---

## Coverage Tracking

### Coverage-Guided Protocol Fuzzing

```bash
# Compile target with coverage instrumentation
# For C/C++ protocol parsers
clang -fsanitize=fuzzer,address -fprofile-instr-generate -fcoverage-mapping \
  -o fuzz_target protocol_parser.c

# AFL++ for network protocol fuzzing via desock
# Use preeny or AFL's network mode
AFL_PRELOAD=libdesock.so afl-fuzz -i corpus/ -o findings/ -- ./target_server

# LibFuzzer harness for protocol parser
cat << 'HARNESS' > fuzz_harness.c
#include <stdint.h>
#include <stddef.h>

extern int parse_protocol_message(const uint8_t *data, size_t size);

int LLVMFuzzerTestOneInput(const uint8_t *data, size_t size) {
    parse_protocol_message(data, size);
    return 0;
}
HARNESS

clang -fsanitize=fuzzer,address fuzz_harness.c protocol_parser.c -o fuzzer
./fuzzer corpus/ -max_len=65536 -timeout=5

# honggfuzz for protocol fuzzing
honggfuzz -i corpus/ -o crashes/ --threads 4 -- ./target_parser ___FILE___

# Coverage report generation
llvm-profdata merge -sparse *.profraw -o merged.profdata
llvm-cov show ./fuzz_target -instr-profile=merged.profdata \
  --format=html > coverage_report.html
```

---

## Crash Deduplication and Triage

### Crash Analysis Pipeline

```bash
# Deduplicate crashes by stack trace
python3 << 'EOF'
import os
import subprocess
import hashlib
from collections import defaultdict

crash_dir = 'findings/crashes'
unique_crashes = defaultdict(list)

for crash_file in os.listdir(crash_dir):
    path = os.path.join(crash_dir, crash_file)
    # Run crash under debugger to get stack trace
    result = subprocess.run(
        ['gdb', '-batch', '-ex', 'run', '-ex', 'bt', '--args', './target_parser', path],
        capture_output=True, text=True, timeout=10
    )
    # Extract crash location (top 3 frames)
    frames = [l.strip() for l in result.stdout.split('\n') if l.strip().startswith('#')][:3]
    trace_hash = hashlib.md5('\n'.join(frames).encode()).hexdigest()
    unique_crashes[trace_hash].append(crash_file)

print(f"Total crashes: {sum(len(v) for v in unique_crashes.values())}")
print(f"Unique crashes: {len(unique_crashes)}")
for h, files in unique_crashes.items():
    print(f"  {h}: {len(files)} instances - {files[0]}")
EOF

# ASan crash analysis
ASAN_OPTIONS=detect_leaks=0:print_legend=0 ./target_parser crash_input
# Crash types:
# - heap-buffer-overflow: read/write past allocation
# - stack-buffer-overflow: stack smashing
# - use-after-free: dangling pointer
# - null-deref: null pointer dereference (usually DoS only)
# - integer-overflow: arithmetic overflow

# Severity classification
# Critical: heap-buffer-overflow (write), use-after-free (write) -> RCE potential
# High: stack-buffer-overflow -> possible RCE
# Medium: heap-buffer-overflow (read) -> info leak
# Low: null-deref, assertion failure -> DoS only
```

### Minimizing Crash Inputs

```bash
# Minimize crash input to smallest reproducer
afl-tmin -i crash_input -o crash_minimized -- ./target_parser @@

# LibFuzzer minimize
./fuzzer -minimize_crash=1 -exact_artifact_path=minimized crash_input

# Manual binary search minimization
python3 << 'EOF'
import subprocess

def crashes(data):
    """Returns True if input causes a crash"""
    result = subprocess.run(['./target_parser'], input=data, capture_output=True, timeout=5)
    return result.returncode != 0

with open('crash_input', 'rb') as f:
    data = bytearray(f.read())

# Remove bytes from the end
while len(data) > 1:
    half = data[:len(data)//2]
    if crashes(bytes(half)):
        data = half
    else:
        break

with open('crash_minimized.bin', 'wb') as f:
    f.write(bytes(data))
print(f"Minimized: {len(data)} bytes")
EOF
```

---

## Peach Fuzzer Configuration

### Peach Pit File

```bash
# Peach Fuzzer protocol definition (XML pit file)
cat << 'PEACHPIT' > protocol.xml
<?xml version="1.0" encoding="utf-8"?>
<Peach xmlns="http://peachfuzzer.com/2012/Peach"
       author="Tester" description="Custom Protocol Fuzzer">

  <DataModel name="ProtocolMessage">
    <Number name="Magic" size="32" value="0xDEADBEEF" signed="false" mutable="false"/>
    <Number name="Version" size="16" value="1" signed="false"/>
    <Number name="Type" size="8" value="0" signed="false"/>
    <Number name="Length" size="32" signed="false">
      <Relation type="size" of="Payload"/>
    </Number>
    <Blob name="Payload" value="" minOccurs="0"/>
  </DataModel>

  <StateModel name="ProtocolSession" initialState="SendMessage">
    <State name="SendMessage">
      <Action type="output">
        <DataModel ref="ProtocolMessage"/>
      </Action>
      <Action type="input">
        <DataModel name="Response">
          <Blob name="data"/>
        </DataModel>
      </Action>
    </State>
  </StateModel>

  <Agent name="LocalAgent">
    <Monitor class="Process">
      <Param name="Executable" value="./target_server"/>
      <Param name="Arguments" value="-p 9999"/>
      <Param name="RestartOnEachTest" value="true"/>
    </Monitor>
  </Agent>

  <Test name="Default">
    <Agent ref="LocalAgent"/>
    <StateModel ref="ProtocolSession"/>
    <Publisher class="TcpClient">
      <Param name="Host" value="127.0.0.1"/>
      <Param name="Port" value="9999"/>
    </Publisher>
  </Test>
</Peach>
PEACHPIT

# Run Peach Fuzzer
peach protocol.xml
```

---

## Tips

- Start with a valid protocol exchange capture before mutating — knowing the baseline prevents false negatives from connection-level failures
- Fuzz length fields first — length/size mismatches are the most common source of memory corruption in protocol parsers
- Use coverage tracking to measure fuzzing effectiveness; blind fuzzing plateaus quickly on complex protocol state machines
- Build a comprehensive seed corpus from real traffic, RFCs/specs, and manually crafted edge cases before starting mutation
- Monitor the target with AddressSanitizer during fuzzing — many bugs produce silent memory corruption without immediate crashes
- Separate crashes by unique stack trace hash to avoid wasting time analyzing the same bug triggered by different inputs
- Fuzz both client and server implementations — client-side parsing bugs are often overlooked but equally exploitable
- Test state machine transitions by sending valid messages in invalid sequences (e.g., data before authentication)
- Use network-level replay tools to verify crashes are reproducible before reporting findings
- Implement adaptive mutation strategies that focus on fields that have previously triggered new code coverage

---

## See Also

- fuzzing
- scapy
- wireshark
- sanitizers

## References

- [Boofuzz Documentation](https://boofuzz.readthedocs.io/)
- [Peach Fuzzer Community Edition](https://github.com/MozillaSecurity/peach)
- [Scapy Documentation](https://scapy.readthedocs.io/)
- [AFL++ Protocol Fuzzing](https://github.com/AFLplusplus/AFLplusplus)
- [Google OSS-Fuzz](https://github.com/google/oss-fuzz)
- [The Fuzzing Book](https://www.fuzzingbook.org/)
- [LibFuzzer Documentation](https://llvm.org/docs/LibFuzzer.html)
