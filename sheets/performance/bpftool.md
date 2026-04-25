# bpftool (eBPF Inspection and Management)

Swiss-army CLI for inspecting, loading, pinning, attaching, and dumping eBPF programs, maps, links, BTF objects, and feature support — the canonical tool that ships in the kernel tree under `tools/bpf/bpftool` for managing the BPF subsystem from userspace.

## Setup

`bpftool` is part of the Linux kernel source tree under `tools/bpf/bpftool`. The version that ships with a distribution is usually pinned to its kernel — running `bpftool` from an old distro against a new kernel often misses subcommands (e.g., `link`, `iter`, `gen min_core_btf`).

### Install via package manager

```bash
# Debian / Ubuntu
sudo apt update
sudo apt install -y linux-tools-common linux-tools-generic linux-tools-$(uname -r)

# Fedora / RHEL / CentOS Stream
sudo dnf install -y bpftool

# Arch Linux
sudo pacman -S bpf

# Alpine
sudo apk add bpftool

# openSUSE
sudo zypper install bpftool
```

### Locate the binary that matches your kernel

On Debian/Ubuntu the kernel-matched binary lives under `/usr/lib/linux-tools/$(uname -r)/bpftool`:

```bash
ls /usr/lib/linux-tools/
ls /usr/lib/linux-tools/$(uname -r)/bpftool
sudo ln -sf /usr/lib/linux-tools/$(uname -r)/bpftool /usr/local/sbin/bpftool
```

### Build from kernel source

If your distro lags, building from the kernel tree is the most reliable path:

```bash
# Clone matching kernel source
git clone --depth=1 --branch v6.10 \
  https://git.kernel.org/pub/scm/linux/kernel/git/torvalds/linux.git
cd linux/tools/bpf/bpftool
make -j"$(nproc)"
sudo make install                # installs to /usr/local/sbin/bpftool
bpftool version
```

### libbpf dependency

`bpftool` links against `libbpf` — usually the in-tree copy under `tools/lib/bpf` when built from source. Distros use a shared `libbpf.so` from `libbpf` or `libbpf1`/`libbpf2`:

```bash
ldd $(which bpftool) | grep libbpf
# libbpf.so.1 => /lib/x86_64-linux-gnu/libbpf.so.1 (0x00007f...)

# Inspect installed libbpf
pkg-config --modversion libbpf
```

### Privilege requirements

Loading BPF programs and reading kernel BTF historically requires `CAP_SYS_ADMIN`. Modern kernels split capabilities:

- `CAP_BPF` — load programs and create maps (Linux 5.8+)
- `CAP_PERF_EVENT` — attach kprobes/uprobes/tracepoints (Linux 5.8+)
- `CAP_NET_ADMIN` — attach XDP, TC, and cgroup networking programs
- `CAP_SYS_ADMIN` — fallback for older kernels and any operation not covered by the split caps

Grant per-binary instead of running everything as root:

```bash
sudo setcap cap_bpf,cap_net_admin,cap_perf_event+ep $(which bpftool)
getcap $(which bpftool)
# /usr/sbin/bpftool cap_bpf,cap_net_admin,cap_perf_event=ep
```

### Mount BPF filesystem

Pinning programs and maps requires the `bpf` virtual filesystem mounted at `/sys/fs/bpf`. Most distros mount it automatically via systemd:

```bash
mount | grep bpf
# bpf on /sys/fs/bpf type bpf (rw,nosuid,nodev,noexec,relatime,mode=700)

# Mount manually if missing
sudo mount -t bpf bpf /sys/fs/bpf
sudo mkdir -p /sys/fs/bpf/programs /sys/fs/bpf/maps
```

### Verify install

```bash
bpftool version
# bpftool v7.4.0
# using libbpf v1.4
# features: libbfd, libbpf_strict, skeletons
```

## Subcommand Catalog

`bpftool` is organized into top-level subcommands. Each maps to a kernel BPF object kind or a userspace helper.

| Subcommand   | What it does                                                  |
|--------------|----------------------------------------------------------------|
| `prog`       | Inspect, load, dump, pin, attach, unload BPF programs         |
| `map`        | Inspect, create, dump, lookup, update, delete BPF maps        |
| `cgroup`     | List/attach/detach cgroup-bound BPF programs                  |
| `perf`       | List perf-event attachments (kprobe/uprobe/tracepoint)        |
| `net`        | List/attach/detach XDP and TC BPF programs per netdev         |
| `feature`    | Probe what BPF features (helpers, prog types, map types) the kernel supports |
| `btf`        | List and dump BTF type information from kernel/programs       |
| `gen`        | Generate libbpf skeleton headers and minimal BTF              |
| `struct_ops` | Manage `struct_ops` programs (custom kernel ops, e.g., TCP CC)|
| `iter`       | Pin and use BPF iterator programs (seq_file-style readers)    |
| `link`       | List/detach BPF link objects (modern attachment)              |
| `version`    | Print bpftool + libbpf version and compiled features          |

Get help on any subcommand:

```bash
bpftool help
bpftool prog help
bpftool map help
bpftool gen help
```

## prog Subcommand

The `prog` subcommand is the entry point for everything related to BPF programs.

### prog show

`bpftool prog show` lists every loaded BPF program in the kernel:

```bash
sudo bpftool prog show
# 12: cgroup_skb  name egress  tag 6deef7357e7b4530  gpl
#         loaded_at 2026-04-22T09:14:11+0000  uid 0
#         xlated 64B  jited 49B  memlock 4096B
#         btf_id 35
# 18: kprobe  name bpf_prog_3a91...  tag 24d471a9b4aabc11
#         loaded_at 2026-04-22T09:18:47+0000  uid 1000
#         xlated 296B  jited 175B  memlock 4096B  map_ids 8,9
#         btf_id 41
```

Output anatomy:

| Field          | Meaning                                                      |
|----------------|--------------------------------------------------------------|
| `id`           | Kernel-assigned numeric program ID (stable while loaded)     |
| `<type>`       | Program type: `kprobe`, `xdp`, `tracepoint`, `cgroup_skb`, `sched_cls`, `lsm`, `tracing`, `cgroup_sock`, `sock_addr`, `flow_dissector`, etc. |
| `name <NAME>`  | Symbol name of the BPF function (truncated to 16 chars by kernel) |
| `tag <TAG>`    | SHA1-derived 8-byte fingerprint of the program bytes         |
| `gpl`          | Program declared `GPL` license — gated helpers available     |
| `loaded_at`    | ISO 8601 timestamp of load                                   |
| `uid`          | UID of the loader                                             |
| `xlated`       | Size in bytes of the verifier-rewritten bytecode             |
| `jited`        | Size in bytes of the JIT machine code (0 if not JITed)       |
| `memlock`      | Bytes accounted against `RLIMIT_MEMLOCK`                     |
| `map_ids`      | Comma list of map IDs the program references                 |
| `btf_id`       | BTF object ID with type info for this program                |
| `pids`         | (with `--show-pids`) PIDs holding fd refs to the program     |

### Filter by id, name, tag, pinned path

```bash
# By ID
sudo bpftool prog show id 12

# By name (matches truncated kernel name)
sudo bpftool prog show name egress

# By tag (full 16-hex SHA1 prefix)
sudo bpftool prog show tag 6deef7357e7b4530

# By pinned path on bpffs
sudo bpftool prog show pinned /sys/fs/bpf/myprog
```

### Show with run-time stats

Run-time stats require `kernel.bpf_stats_enabled=1` (or use `bpftool` itself to enable):

```bash
# Enable run-time accounting (kernel 5.1+)
sudo sysctl -w kernel.bpf_stats_enabled=1

# Or use the dedicated subcommand
sudo bpftool prog profile id 12 duration 5 cycles instructions

sudo bpftool prog show
# 12: ...
#         run_time_ns 184221 run_cnt 9341
```

`run_time_ns / run_cnt` gives mean execution time per invocation — the canonical "is my probe expensive" metric.

### JSON output

Every `prog show` invocation accepts `-j` (compact) or `-p` (pretty):

```bash
sudo bpftool -p prog show
sudo bpftool prog show --json | jq '.[] | {id, name, type, run_time_ns, run_cnt}'
```

JSON shape per program:

```json
{
  "id": 12,
  "type": "cgroup_skb",
  "name": "egress",
  "tag": "6deef7357e7b4530",
  "gpl_compatible": true,
  "loaded_at": 1745312051,
  "uid": 0,
  "bytes_xlated": 64,
  "jited": true,
  "bytes_jited": 49,
  "bytes_memlock": 4096,
  "btf_id": 35
}
```

## prog dump xlated

`prog dump xlated` prints the verifier-rewritten BPF bytecode — what actually runs after the kernel patches helpers, inline assembly, and CO-RE relocations:

```bash
sudo bpftool prog dump xlated id 12
# 0: (b7) r0 = 0
# 1: (95) exit
```

### Annotate with line info

If the program was compiled with BTF and line info (`-g`):

```bash
sudo bpftool prog dump xlated id 12 linum
#    0: (b7) r0 = 0      ; return XDP_PASS;  // file.c:42
#    1: (95) exit
```

### Render as control-flow graph

`xlated visual` emits a Graphviz `dot` description of basic blocks and edges:

```bash
sudo bpftool prog dump xlated id 12 visual > prog.dot
dot -Tpng prog.dot -o prog.png
# Or render inline
sudo bpftool prog dump xlated id 12 visual | dot -Tsvg > prog.svg
```

### Read raw opcodes

```bash
sudo bpftool prog dump xlated id 12 opcodes
#    0: (b7) r0 = 0
#       b7 00 00 00 00 00 00 00
#    1: (95) exit
#       95 00 00 00 00 00 00 00
```

Each line shows mnemonic + the 8-byte `bpf_insn` struct.

### Output to file

```bash
sudo bpftool prog dump xlated id 12 file /tmp/prog.bin
```

Useful for offline disassembly with `llvm-objdump` or hand inspection.

## prog dump jited

`prog dump jited` prints the architecture-specific machine code emitted by the JIT:

```bash
sudo bpftool prog dump jited id 12
#    0:   nopl   0x0(%rax,%rax,1)
#    5:   xchg   %ax,%ax
#    7:   push   %rbp
#    8:   mov    %rsp,%rbp
#   ...
#   28:   xor    %eax,%eax
#   2a:   leaveq
#   2b:   retq
```

Jited dump requires `CONFIG_BPF_JIT=y` and `net.core.bpf_jit_enable=1`:

```bash
sudo sysctl net.core.bpf_jit_enable
# net.core.bpf_jit_enable = 1

sudo sysctl -w net.core.bpf_jit_enable=1
```

### Annotate with source lines

```bash
sudo bpftool prog dump jited id 12 linum
```

### Why jited dumps matter

- Spot-check that the JIT inlined helper calls
- Verify alignment/padding for hot paths
- Compare instruction count between kernels to spot regressions
- Reverse engineer programs loaded by closed-source agents

```bash
# Save jited code for offline analysis
sudo bpftool prog dump jited id 12 file /tmp/prog.jit
hexdump -C /tmp/prog.jit | head
```

## prog load

`prog load` loads an ELF object compiled by clang into the kernel and pins it under `/sys/fs/bpf`.

```bash
# Compile a BPF C source with clang
clang -O2 -g -target bpf -c probe.c -o probe.bpf.o

# Load and pin
sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe
sudo bpftool prog show pinned /sys/fs/bpf/probe
```

### Specify program type explicitly

If section name conventions don't disambiguate (older clang), pass the type:

```bash
sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe \
  type kprobe \
  pinmaps /sys/fs/bpf/maps
```

Recognized type strings: `socket`, `kprobe`, `kretprobe`, `uprobe`, `uretprobe`, `tracepoint`, `raw_tracepoint`, `xdp`, `perf_event`, `cgroup/skb`, `cgroup/sock`, `cgroup/dev`, `lwt_in`, `lwt_out`, `lwt_xmit`, `sock_ops`, `sk_skb`, `sk_msg`, `lirc_mode2`, `flow_dissector`, `cgroup/sysctl`, `cgroup/sock_release`, `cgroup/sock_create`, `cgroup/sock_post_bind`, `lsm`, `iter`, `tracing`, `struct_ops`, `ext`, `sk_lookup`, `cgroup/getsockopt`, `cgroup/setsockopt`.

### ELF section convention

When using libbpf-style ELF sections, type discovery is automatic:

```c
SEC("kprobe/sys_clone")  /* type=kprobe */
SEC("xdp")               /* type=xdp */
SEC("tracepoint/syscalls/sys_enter_open")
SEC("cgroup/skb")
SEC("tp_btf/sched_switch")
SEC("fentry/inet_listen")
SEC("lsm/file_open")
```

### Pin maps with the program

`pinmaps DIR` pins every map the program references under `DIR/<map_name>`:

```bash
sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe \
  pinmaps /sys/fs/bpf/probe_maps
ls /sys/fs/bpf/probe_maps/
# events  stats  ringbuf
```

### Reuse an existing pinned map

```bash
sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe2 \
  map name shared_map pinned /sys/fs/bpf/maps/shared_map
```

This is the canonical "two programs share state" pattern.

### Load with attached object name override

```bash
sudo bpftool prog loadall probe.bpf.o /sys/fs/bpf/probe_dir
# Pins every program inside the ELF under /sys/fs/bpf/probe_dir/<sec_name>
```

## prog detach / unload

There is no direct `prog unload` — eBPF programs are reference-counted. They are unloaded when:

1. All file descriptors holding them are closed
2. All pinned paths are removed
3. All attachments (cgroup, XDP, TC, link) are detached

### The canonical "remove via unpin" workflow

```bash
# Detach attachments first
sudo bpftool cgroup detach /sys/fs/cgroup/myapp egress \
  pinned /sys/fs/bpf/probe

# Or for XDP
sudo bpftool net detach xdp dev eth0

# Then unpin
sudo rm /sys/fs/bpf/probe
sudo rm -rf /sys/fs/bpf/probe_maps

# Verify gone
sudo bpftool prog show
```

### Modern: detach via link

If the program was attached using the `link` API (Linux 5.7+), detaching is a single op:

```bash
sudo bpftool link show
# 5: cgroup  prog 12
#         cgroup_id 1234  attach_type egress
sudo bpftool link detach id 5
```

### Force-removing stuck programs

If references leak (rare — usually a buggy loader process), find them:

```bash
sudo bpftool prog show id 12 --show-pids
# 12: ... pids my_loader(1234)
sudo kill 1234
```

## map Subcommand

Maps are typed key/value containers shared between BPF programs and userspace.

### map show

```bash
sudo bpftool map show
# 8: hash  name flow_table  flags 0x0
#         key 16B  value 32B  max_entries 4096  memlock 327680B
#         btf_id 41
# 9: ringbuf  name events  flags 0x0
#         key 0B  value 0B  max_entries 16777216  memlock 16781312B
# 10: percpu_array  name stats  flags 0x0
#         key 4B  value 8B  max_entries 256  memlock 12288B
```

Output anatomy:

| Field          | Meaning                                                      |
|----------------|--------------------------------------------------------------|
| `id`           | Kernel-assigned numeric map ID                               |
| `<type>`       | Map type (see catalog below)                                 |
| `name`         | Map name from C source (`SEC(".maps")` or libbpf-skel)       |
| `flags`        | Bitmask: `BPF_F_NO_PREALLOC`, `BPF_F_RDONLY`, `BPF_F_WRONLY`, `BPF_F_NUMA_NODE`, `BPF_F_MMAPABLE` |
| `key`          | Key size in bytes                                            |
| `value`        | Value size in bytes                                          |
| `max_entries`  | Capacity                                                     |
| `memlock`      | Bytes accounted against `RLIMIT_MEMLOCK`                     |
| `btf_id`       | BTF object describing key/value types                        |

### Filter selectors

```bash
sudo bpftool map show id 8
sudo bpftool map show name flow_table
sudo bpftool map show pinned /sys/fs/bpf/maps/flow_table
```

### Map type catalog

| Type                | Use case                                                |
|---------------------|---------------------------------------------------------|
| `hash`              | General-purpose key/value table                         |
| `array`             | Indexed by `u32`, fixed size, dense                     |
| `prog_array`        | Tail-call jump table — values are program fds          |
| `perf_event_array`  | Output channel into perf ring buffer                   |
| `percpu_hash`       | One hash table per CPU, no locking needed              |
| `percpu_array`      | One array per CPU                                       |
| `stack_trace`       | Stack ID → resolved frames (for stack sampling)        |
| `cgroup_array`      | Index → cgroup fd                                       |
| `lru_hash`          | Hash with LRU eviction                                  |
| `lru_percpu_hash`   | Per-CPU LRU                                             |
| `lpm_trie`          | Longest-prefix-match trie (CIDR routing)                |
| `array_of_maps`     | Outer array of inner-map fds                            |
| `hash_of_maps`      | Outer hash of inner-map fds                             |
| `devmap` / `devmap_hash` | Index/key → netdev (XDP redirect)                  |
| `sockmap`           | Index → socket fd (sk_skb / sk_msg dispatch)            |
| `sockhash`          | Hash variant of sockmap                                  |
| `cpumap`            | Index → CPU (XDP redirect to CPU)                       |
| `xskmap`            | Index → AF_XDP socket                                    |
| `reuseport_sockarray` | Index → reuseport socket                               |
| `queue` / `stack`   | FIFO / LIFO without keys                                |
| `sk_storage`        | Socket-local storage                                     |
| `task_storage`      | Task-local storage                                       |
| `inode_storage`     | Inode-local storage                                      |
| `cgrp_storage`      | Cgroup-local storage                                     |
| `ringbuf`           | MPSC ring buffer (BPF→userspace, kernel 5.8+)           |
| `bloom_filter`      | Probabilistic membership test (kernel 5.16+)            |
| `user_ringbuf`      | User→BPF ring buffer (kernel 6.1+)                      |
| `struct_ops`        | Pluggable kernel struct (TCP CC etc.)                   |

### JSON output for scripts

```bash
sudo bpftool map show -j | jq '.[] | select(.type == "ringbuf")'
```

## map Operations

### map dump

Walks every key/value pair and prints them. Format depends on whether BTF is present.

```bash
sudo bpftool map dump id 10
# [{
#         "key": 0,
#         "values": [{
#                 "cpu": 0,
#                 "value": 184
#             },{
#                 "cpu": 1,
#                 "value": 247
#             }]
#     }]

sudo bpftool map dump pinned /sys/fs/bpf/maps/flow_table
sudo bpftool map dump name flow_table
```

For maps without BTF, output is hex bytes:

```bash
sudo bpftool map dump id 8
# key:
# 0a 00 00 01 ...
# value:
# 00 00 00 00 ...
```

### map lookup

```bash
sudo bpftool map lookup id 8 key 0x0a 0x00 0x00 0x01
# key:
# 0a 00 00 01
# value:
# 7d 02 00 00 00 00 00 00

# Hex string form
sudo bpftool map lookup id 8 key hex 0a 00 00 01

# Per-cpu lookup returns one value per CPU
sudo bpftool map lookup id 10 key 0
```

### map update

```bash
sudo bpftool map update id 8 \
  key 0x0a 0x00 0x00 0x01 \
  value 0x7d 0x02 0x00 0x00 0x00 0x00 0x00 0x00

# With "any/exist/noexist" flags
sudo bpftool map update id 8 key hex 0a 00 00 01 \
  value hex 7d 02 00 00 00 00 00 00 any
sudo bpftool map update id 8 key hex 0a 00 00 01 \
  value hex 7d 02 00 00 00 00 00 00 exist
sudo bpftool map update id 8 key hex 0a 00 00 01 \
  value hex 7d 02 00 00 00 00 00 00 noexist
```

### map delete

```bash
sudo bpftool map delete id 8 key 0x0a 0x00 0x00 0x01
sudo bpftool map delete pinned /sys/fs/bpf/maps/flow_table key hex 0a 00 00 01
```

### map create

Creates a freshly pinned map without loading any program:

```bash
sudo bpftool map create /sys/fs/bpf/mymap \
  type hash \
  key 4 \
  value 8 \
  entries 1024 \
  name mymap
sudo bpftool map show pinned /sys/fs/bpf/mymap
```

Useful for staging shared state before programs that consume it are loaded.

### map enqueue / dequeue (queue, stack)

```bash
sudo bpftool map enqueue id 22 value hex 01 02 03 04
sudo bpftool map dequeue id 22
```

### map peek (queue, stack)

```bash
sudo bpftool map peek id 22
```

### map pin / unpin

```bash
sudo bpftool map pin id 8 /sys/fs/bpf/maps/flow_table
sudo rm /sys/fs/bpf/maps/flow_table
```

### map freeze (read-only after this point)

```bash
sudo bpftool map freeze id 8
# Kernel rejects further userspace writes to this map.
```

## cgroup Subcommand

Manages cgroup-attached BPF programs — the canonical mechanism for per-container packet filtering, bind-port filtering, sock_addr rewriting, and cgroup-scoped LSM hooks.

### cgroup tree

Walk every cgroup under a path and report attached BPF programs:

```bash
sudo bpftool cgroup tree /sys/fs/cgroup
# CgroupPath                                 ID       AttachType      AttachFlags     Name
# /sys/fs/cgroup
# /sys/fs/cgroup/system.slice
#     12       egress
#     12       ingress
# /sys/fs/cgroup/user.slice
```

### cgroup show

Show attached BPF programs for one cgroup:

```bash
sudo bpftool cgroup show /sys/fs/cgroup/system.slice/myapp.service
# ID       AttachType      AttachFlags     Name
# 12       egress                          egress
# 14       ingress         multi           ingress
```

### cgroup attach

```bash
sudo bpftool cgroup attach /sys/fs/cgroup/system.slice/myapp.service \
  egress \
  pinned /sys/fs/bpf/myprog
```

Attach types:

| Type                | Hook                                                      |
|---------------------|-----------------------------------------------------------|
| `ingress`           | `cgroup_skb` ingress packet filter                        |
| `egress`            | `cgroup_skb` egress packet filter                         |
| `sock_create`       | `cgroup_sock` socket create                               |
| `post_bind4`        | `cgroup_sock` after IPv4 bind                             |
| `post_bind6`        | `cgroup_sock` after IPv6 bind                             |
| `connect4`/`connect6`| `cgroup_sock_addr` connect rewrite                       |
| `bind4`/`bind6`     | `cgroup_sock_addr` bind                                   |
| `sendmsg4`/`sendmsg6`| `cgroup_sock_addr` UDP sendmsg                          |
| `recvmsg4`/`recvmsg6`| `cgroup_sock_addr` UDP recvmsg                          |
| `getpeername4`/`getpeername6`| `cgroup_sock_addr` getpeername                  |
| `getsockname4`/`getsockname6`| `cgroup_sock_addr` getsockname                  |
| `sock_ops`          | TCP state machine callbacks                               |
| `device`            | `cgroup_device` access control                            |
| `sysctl`            | `cgroup_sysctl` read/write                                |
| `getsockopt`/`setsockopt`| `cgroup_sockopt`                                     |
| `sock_release`      | Socket release callback                                   |

### Attach flags

```bash
# Override existing single attachment
sudo bpftool cgroup attach /sys/fs/cgroup/myapp egress pinned /sys/fs/bpf/myprog override

# Multi-prog attach (requires multi flag on first attach)
sudo bpftool cgroup attach /sys/fs/cgroup/myapp egress pinned /sys/fs/bpf/myprog multi
```

### cgroup detach

```bash
sudo bpftool cgroup detach /sys/fs/cgroup/system.slice/myapp.service \
  egress \
  pinned /sys/fs/bpf/myprog
```

## perf Subcommand

`perf show` lists every perf-event-style attachment (kprobe, uprobe, tracepoint, raw_tracepoint) and the BPF program attached to it — the canonical "what is hooked into kprobe X" tool:

```bash
sudo bpftool perf show
# pid 1234  fd 7: prog_id 12  kprobe  func sys_clone  offset 0
# pid 1234  fd 8: prog_id 12  kretprobe  func sys_clone  offset 0
# pid 5678  fd 12: prog_id 18  tracepoint  syscalls/sys_enter_open
# pid 9012  fd 22: prog_id 24  uprobe  filename /usr/lib/x86_64-linux-gnu/libssl.so.3  offset 0x1d2c0
```

JSON for scripting:

```bash
sudo bpftool perf show -j | jq '.[] | select(.type == "kprobe") | {pid, prog_id, func}'
```

### Use case: discover all probes touching one symbol

```bash
sudo bpftool perf show -j \
  | jq '.[] | select(.func == "tcp_v4_connect")'
```

## net Subcommand

`net show` lists XDP and TC BPF attachments for every netdev:

```bash
sudo bpftool net show
# xdp:
# eth0(2) generic id 12 tag 6deef7357e7b4530
# eth0(2) driver  id 14 tag 1bb45a32...
#
# tc:
# eth0(2) clsact/ingress  bpf_prefetch_egress_55c id 18 tag 67c2dc...
# eth0(2) clsact/egress   bpf_prefetch_ingress_55c id 19 tag bbf213...
#
# flow_dissector:
```

### XDP modes

| Mode      | Where                                            |
|-----------|--------------------------------------------------|
| `xdpdrv`  | Native driver (best perf, requires driver support)|
| `xdpoffload` | Offloaded to NIC (Netronome smart NICs)       |
| `xdpgeneric` | Generic skb-based fallback (works everywhere) |

### Attach XDP

```bash
sudo bpftool net attach xdp \
  pinned /sys/fs/bpf/xdp_drop \
  dev eth0 \
  overwrite
```

### Replace and detach XDP

```bash
# Replace running XDP program
sudo bpftool net attach xdpdrv pinned /sys/fs/bpf/v2 dev eth0 overwrite

# Detach all XDP from a device
sudo bpftool net detach xdp dev eth0
sudo bpftool net detach xdpgeneric dev eth0
```

### Attach TC

`bpftool net attach` does not handle TC clsact directly on every kernel — `tc filter add` is still the canonical TC path:

```bash
sudo tc qdisc add dev eth0 clsact
sudo tc filter add dev eth0 ingress bpf da object-pinned /sys/fs/bpf/tc_prog
sudo bpftool net show dev eth0
```

## feature Subcommand

`feature probe` reports which BPF capabilities the running kernel supports — the indispensable portability check before deploying.

### Quick probe

```bash
sudo bpftool feature probe
# Scanning system configuration...
# /proc/sys/net/core/bpf_jit_enable is set to 1.
# bpf() syscall restricted to privileged users.
# CONFIG_BPF is set to y.
# CONFIG_BPF_SYSCALL is set to y.
# ...
# Scanning eBPF program types...
# eBPF program_type socket_filter is available
# eBPF program_type kprobe is available
# eBPF program_type sched_cls is available
# ...
# Scanning eBPF map types...
# eBPF map_type hash is available
# eBPF map_type array is available
# ...
# Scanning eBPF helper functions...
# eBPF helpers supported for program type kprobe:
#         - bpf_map_lookup_elem
#         - bpf_map_update_elem
#         - bpf_probe_read_kernel
#         ...
```

### Verbose / unprivileged probe

```bash
sudo bpftool feature probe full        # include all helpers per prog type
sudo bpftool feature probe unprivileged
sudo bpftool feature probe dev eth0    # device-level (XDP offload) probe
```

### Generate C macros

`feature probe macros` emits `#define HAVE_*` lines suitable for inclusion in compile-time conditionals:

```bash
sudo bpftool feature probe macros > bpf_features.h
head bpf_features.h
# /*
#  * Generated by bpftool feature probe macros.
#  */
# #ifndef BPFTOOL_FEATURES_H
# #define BPFTOOL_FEATURES_H
# #define HAVE_V1_PROG_TYPE
# #define HAVE_KPROBE_PROG_TYPE
# #define HAVE_RINGBUF_MAP_TYPE
# #define HAVE_BPF_MAP_LOOKUP_ELEM_HELPER
# ...
```

### Probe one feature programmatically

```bash
sudo bpftool feature list_builtins prog_types
sudo bpftool feature list_builtins map_types
sudo bpftool feature list_builtins attach_types
sudo bpftool feature list_builtins helpers
```

## btf Subcommand

BTF (BPF Type Format) carries type information that powers CO-RE, ringbuf type-aware dumps, and pretty-printed map output.

### btf list

```bash
sudo bpftool btf show
# 1: name [vmlinux]  size 5234123B
# 35: name <anon>  size 1024B  prog_ids 12  map_ids 8,9
# 41: name <anon>  size 2048B  prog_ids 18  map_ids 10
```

### btf dump

Render BTF as C-style declarations:

```bash
sudo bpftool btf dump id 35 format c
# struct flow_key {
#     __u32 src_ip;
#     __u32 dst_ip;
#     __u16 src_port;
#     __u16 dst_port;
# };

# Read the kernel BTF directly
sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c | less

# Dump as raw form (verbose, opcodes)
sudo bpftool btf dump id 35 format raw
```

### Inspect kernel struct layout for CO-RE

```bash
sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c \
  | grep -A 20 'struct task_struct {'
```

This is the canonical way to verify a kernel struct field exists at the offset your CO-RE relocation expects.

### Per-module BTF (kernel 5.11+)

```bash
ls /sys/kernel/btf/
# vmlinux  drm  i2c_core  nvme_core  ...
sudo bpftool btf dump file /sys/kernel/btf/nvme_core format c
```

### btf dump on programs/maps

```bash
sudo bpftool btf dump prog id 12 format c
sudo bpftool btf dump map id 8 format c
```

## iter Subcommand

BPF iterators (Linux 5.4+) are programs that walk kernel data structures and produce `seq_file`-style output readable from userspace.

### Pin an iter

Compile a `SEC("iter/tcp")` or `SEC("iter/task")` program, then:

```bash
sudo bpftool prog load tcp_iter.bpf.o /sys/fs/bpf/tcp_iter_prog
sudo bpftool iter pin /sys/fs/bpf/tcp_iter_prog /sys/fs/bpf/tcp_iter
```

### Read the iter

```bash
sudo cat /sys/fs/bpf/tcp_iter
# 0a000001:0050  0a000002:8b6c  ESTAB
# 0a000001:01bb  0a000003:c2ac  TIME-WAIT
```

The iter program runs every time the file is read — no caching.

### Available iter targets

| Target              | Walks                                                |
|---------------------|------------------------------------------------------|
| `iter/task`         | Every `struct task_struct` in the system             |
| `iter/task_file`    | Every open fd in every task                          |
| `iter/task_vma`     | Every VMA of every task                              |
| `iter/tcp` / `iter/tcp4` / `iter/tcp6` | TCP socket table                  |
| `iter/udp` / `iter/udp4` / `iter/udp6` | UDP socket table                  |
| `iter/unix`         | UNIX domain sockets                                  |
| `iter/bpf_map`      | Every loaded BPF map                                 |
| `iter/bpf_map_elem` | Every entry of one map                               |
| `iter/bpf_prog`     | Every loaded BPF program                             |
| `iter/bpf_link`     | Every BPF link object                                |
| `iter/bpf_sk_storage_map` | sk_storage entries                              |
| `iter/cgroup`       | Every cgroup                                         |
| `iter/sockmap`      | Sockets in a sockmap                                 |
| `iter/ksym`         | Kernel symbol table                                  |

### Detach iter

```bash
sudo rm /sys/fs/bpf/tcp_iter
sudo rm /sys/fs/bpf/tcp_iter_prog
```

## link Subcommand

The `link` API (Linux 5.7+) is the modern way to attach BPF programs. A `link` is a kernel-managed attachment object referenced by file descriptor — pinning the link means the attachment outlives the process that created it.

### link show

```bash
sudo bpftool link show
# 5: cgroup  prog 12
#         cgroup_id 1234  attach_type egress
# 8: xdp  prog 14
#         ifindex 2  iface eth0
# 12: kprobe_multi  prog 18
#         func_cnt 1234  link_type single  flags 0x0
# 22: tracing  prog 24  prog_type tracing  attach_type trace_fentry  attach_func tcp_v4_connect
```

Filter selectors:

```bash
sudo bpftool link show id 5
sudo bpftool link show pinned /sys/fs/bpf/links/mylink
```

### link pin / detach

```bash
sudo bpftool link pin id 5 /sys/fs/bpf/links/cgroup_egress
sudo bpftool link detach id 5
sudo bpftool link detach pinned /sys/fs/bpf/links/cgroup_egress
```

### Why link is better than legacy attach

- One detach call removes the attachment, no walking the cgroup hierarchy
- Survives loader process exit when pinned
- Works for `kprobe_multi`, `uprobe_multi`, `tcx`, `netkit`, `tracing` (fentry/fexit), `lsm` — none of which use legacy attach paths

## struct_ops Subcommand

`struct_ops` programs implement pluggable kernel structs from BPF — most notably TCP congestion control algorithms, struct sched_ext, and HID-BPF.

### struct_ops show

```bash
sudo bpftool struct_ops show
# 1: name bpf_cubic  type tcp_congestion_ops
# 2: name dctcp_bpf  type tcp_congestion_ops
```

### struct_ops register

```bash
sudo bpftool struct_ops register cubic_bpf.bpf.o /sys/fs/bpf/cubic_bpf
```

After registration, the algorithm is selectable via the existing kernel APIs:

```bash
sudo sysctl net.ipv4.tcp_available_congestion_control
# net.ipv4.tcp_available_congestion_control = reno cubic bbr cubic_bpf
sudo sysctl -w net.ipv4.tcp_congestion_control=cubic_bpf
```

### struct_ops unregister

```bash
sudo bpftool struct_ops unregister id 2
sudo bpftool struct_ops unregister pinned /sys/fs/bpf/cubic_bpf
```

### struct_ops dump

```bash
sudo bpftool struct_ops dump id 2
sudo bpftool struct_ops dump name bpf_cubic
```

## gen Subcommand — Skeleton Generation

`bpftool gen skeleton` produces a libbpf "skeleton" header — a strongly-typed C struct that encapsulates an entire BPF object file. This is the modern alternative to manually calling `bpf_object__find_program_by_name`, `bpf_object__find_map_by_name`, and friends.

### Generate skeleton

```bash
clang -O2 -g -target bpf -c probe.bpf.c -o probe.bpf.o
bpftool gen skeleton probe.bpf.o > probe.skel.h
```

`probe.skel.h` exposes:

- `struct probe_bpf` — the loaded object
- `probe_bpf__open()` / `probe_bpf__open_opts()` — parse ELF, don't load
- `probe_bpf__load()` — verify and load into kernel
- `probe_bpf__attach()` — attach every program by SEC name
- `probe_bpf__detach()` — detach all
- `probe_bpf__destroy()` — unload and free
- `obj->progs.<name>` — typed program handle
- `obj->maps.<name>` — typed map handle
- `obj->bss`, `obj->data`, `obj->rodata` — typed read/write access to global vars

### Minimal userspace using skeleton

```c
#include "probe.skel.h"

int main(void) {
    struct probe_bpf *obj = probe_bpf__open_and_load();
    if (!obj) return 1;
    if (probe_bpf__attach(obj)) {
        probe_bpf__destroy(obj);
        return 1;
    }
    while (running) {
        int v = obj->bss->counter;
        printf("counter=%d\n", v);
        sleep(1);
    }
    probe_bpf__destroy(obj);
    return 0;
}
```

Build:

```bash
clang -O2 -lbpf -lelf -lz probe.c -o probe
```

## gen Subcommand — Subskeleton

`gen subskeleton` produces a "partial" skeleton suitable for a process that does not load the program but only reads/writes a subset of its maps:

```bash
bpftool gen subskeleton probe.bpf.o > probe.subskel.h
```

The subskeleton has no `__open` / `__load` / `__attach` — instead it accepts an already-loaded `bpf_object` and exposes the maps and global variables for reading. Use cases:

- Cilium-style "agent loads programs, sidecar reads metrics"
- Long-lived loader plus short-lived dump utility
- Multi-process visibility into a single BPF program's state

### Use it

```c
#include "probe.subskel.h"

int main(void) {
    struct bpf_object *obj = bpf_object__open_file("/sys/fs/bpf/probe", NULL);
    if (libbpf_get_error(obj)) return 1;
    struct probe_subbpf *sub = probe_subbpf__open(obj);
    int v = sub->bss->counter;
    probe_subbpf__destroy(sub);
    bpf_object__close(obj);
}
```

## gen Subcommand — Minimal Vmlinux

`gen min_core_btf` extracts only the kernel types referenced by one or more BPF objects and writes them to a fresh BTF file — the canonical "CO-RE everywhere" distribution pattern, used to ship BPF programs that only need a sliver of `vmlinux.btf`.

```bash
sudo cp /sys/kernel/btf/vmlinux /tmp/vmlinux.btf
bpftool gen min_core_btf /tmp/vmlinux.btf min.btf prog1.bpf.o prog2.bpf.o
ls -lh min.btf
# -rw-r--r-- 1 user user 18K min.btf

# Sizes:
#   /sys/kernel/btf/vmlinux: 5.0 MiB
#   min.btf: 18 KiB (only the types our programs reference)
```

### Use min.btf at runtime on systems lacking BTF

```bash
sudo bpftool gen min_core_btf /tmp/vmlinux.btf min.btf myprog.bpf.o
# Ship min.btf with your binary; libbpf can load it via
# struct bpf_object_open_opts opts = { .btf_custom_path = "min.btf" };
```

This is the `BTFGen` workflow used by Tracee, Pixie, and Inspektor Gadget.

### gen object

```bash
bpftool gen object combined.bpf.o a.bpf.o b.bpf.o
```

Links multiple BPF object files into one — equivalent to `bpf_linker`.

## version

`bpftool version` reports the tool version, the libbpf version it was built against, and the optional features compiled in:

```bash
bpftool version
# bpftool v7.4.0
# using libbpf v1.4
# features: libbfd, libbpf_strict, skeletons
```

| Feature        | What it enables                                            |
|----------------|------------------------------------------------------------|
| `libbfd`       | Disassembling jited output to host architecture mnemonics  |
| `libbpf_strict`| Strict libbpf 1.0 API mode                                 |
| `skeletons`    | `gen skeleton` and `gen subskeleton` subcommands           |

`--version` is an alias on most builds:

```bash
bpftool --version
```

## JSON Output

`-j` for compact, `-p` for pretty. JSON shape is API-stable across bpftool versions; the human-readable text format is not.

```bash
sudo bpftool -j prog show
sudo bpftool -p map show

# Filter: every kprobe-type program with run_cnt > 0
sudo bpftool prog show -j \
  | jq '.[] | select(.type=="kprobe") | select(.run_cnt > 0) | {id, name, run_time_ns, run_cnt, mean_ns: (.run_time_ns / .run_cnt)}'

# Filter: every map larger than 1 MiB memlock
sudo bpftool map show -j \
  | jq '.[] | select(.bytes_memlock > 1048576) | {id, name, type, bytes_memlock}'
```

### Common scripted-pipeline patterns

```bash
# Find the program ID currently attached to XDP on eth0
sudo bpftool net show -j \
  | jq -r '.xdp[] | select(.devname=="eth0") | .id'

# List every cgroup with a BPF egress program attached
sudo bpftool cgroup tree -j \
  | jq -r '.[] | select(.programs[]?.attach_type=="egress") | .cgroup'

# Dump every map name/type pair as TSV
sudo bpftool map show -j \
  | jq -r '.[] | "\(.id)\t\(.type)\t\(.name)"'
```

### --debug output

```bash
sudo bpftool --debug prog load probe.bpf.o /sys/fs/bpf/probe
# libbpf: loading object 'probe.bpf.o' from buffer
# libbpf: elf: section(3) tracepoint/syscalls/sys_enter_open
# ...
```

`--debug` enables libbpf debug logs — invaluable when load fails with a generic `Operation not permitted`.

### --legacy

Forces use of the legacy `bpf()` attach paths instead of `BPF_LINK_CREATE`. Useful when running modern bpftool against an old kernel that lacks the link API.

```bash
sudo bpftool --legacy prog load probe.bpf.o /sys/fs/bpf/probe
```

## Common Workflow Recipes

### Recipe: load → pin → attach → inspect → modify → detach

```bash
# 1. Compile
clang -O2 -g -target bpf -c probe.bpf.c -o probe.bpf.o

# 2. Load and pin program + maps
sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe \
  pinmaps /sys/fs/bpf/probe_maps

# 3. Attach to a cgroup
sudo bpftool cgroup attach /sys/fs/cgroup/system.slice/myapp.service \
  egress pinned /sys/fs/bpf/probe

# 4. Inspect
sudo bpftool prog show pinned /sys/fs/bpf/probe
sudo bpftool prog dump xlated pinned /sys/fs/bpf/probe linum
sudo bpftool map dump pinned /sys/fs/bpf/probe_maps/stats

# 5. Modify map state from userspace
sudo bpftool map update pinned /sys/fs/bpf/probe_maps/config \
  key 0 0 0 0 \
  value 1 0 0 0

# 6. Detach + unpin
sudo bpftool cgroup detach /sys/fs/cgroup/system.slice/myapp.service \
  egress pinned /sys/fs/bpf/probe
sudo rm /sys/fs/bpf/probe
sudo rm -rf /sys/fs/bpf/probe_maps
```

### Recipe: discover what eBPF is hooked into kprobe X

```bash
sudo bpftool perf show -j \
  | jq -r '.[] | select(.func == "tcp_v4_connect") | "pid=\(.pid) prog=\(.prog_id)"'

# Then dump the program
sudo bpftool prog dump xlated id 18
```

### Recipe: what XDP / TC is on every interface

```bash
sudo bpftool net show -j \
  | jq -r '.xdp[]? | "xdp \(.devname) prog=\(.id)"; .tc[]? | "tc \(.devname) \(.kind) prog=\(.id)"'
```

### Recipe: replace a running XDP program with no drop

```bash
clang -O2 -g -target bpf -c new.bpf.c -o new.bpf.o
sudo bpftool prog load new.bpf.o /sys/fs/bpf/xdp_new
sudo bpftool net attach xdpdrv pinned /sys/fs/bpf/xdp_new dev eth0 overwrite
# Old program auto-unloaded when no fds remain.
```

### Recipe: portable feature gate before deploying

```bash
sudo bpftool feature probe -j > kernel-features.json
jq '.helpers.kprobe_available_helpers | index("bpf_get_current_pid_tgid")' \
  kernel-features.json
# 6
# (non-null = available)

# Generate macros for use in BPF C source
sudo bpftool feature probe macros > include/kernel_features.h
```

### Recipe: extract minimal BTF for portable distribution

```bash
sudo cp /sys/kernel/btf/vmlinux ./vmlinux.btf
bpftool gen min_core_btf vmlinux.btf min.btf $(ls *.bpf.o)
tar czf bundle.tgz programs/ min.btf
```

### Recipe: live-debug a program with prog dump

```bash
sudo bpftool prog show -j | jq '.[] | select(.name=="my_prog")'
PID=$(sudo bpftool prog show -j | jq '.[] | select(.name=="my_prog") | .id')
sudo bpftool prog dump xlated id $PID linum
sudo bpftool prog dump jited id $PID linum
sudo bpftool prog profile id $PID duration 5 \
  cycles instructions cache_misses
```

### Recipe: enable run-time stats globally

```bash
sudo sysctl -w kernel.bpf_stats_enabled=1
# Watch for hot programs
watch -n1 "sudo bpftool prog show -j | jq -r '.[] | select(.run_cnt>0) | \"\\(.id) \\(.name) mean=\\(.run_time_ns/.run_cnt)ns\"' | sort -k4 -n -r | head"
```

## Common Errors and Fixes

| Error                                                | Cause / fix                                                                 |
|------------------------------------------------------|-----------------------------------------------------------------------------|
| `Error: bpftool: Permission denied`                  | Run with `sudo`, or grant `cap_bpf,cap_net_admin,cap_perf_event+ep` via `setcap` |
| `libbpf: failed to find BTF for ...`                 | Kernel BTF missing or wrong version; check `/sys/kernel/btf/vmlinux` exists; `CONFIG_DEBUG_INFO_BTF=y` required |
| `Error: failed to open object file: No such file or directory` | Wrong path to `.bpf.o` — use absolute path                       |
| `Error: type not found in vmlinux BTF`               | A CO-RE relocation references a kernel type that doesn't exist on this kernel; use `gen min_core_btf` per-target or `__builtin_preserve_field_info` guards |
| `Error: program already loaded`                      | Unload before reload (`rm` the pinned path) or use a different pin name     |
| `libbpf: prog 'foo': BPF program load failed: Argument list too long` | Verifier rejected — usually too many instructions or stack > 512 bytes; split program or reduce stack |
| `libbpf: prog 'foo': BPF program load failed: Invalid argument` | Verifier rejected — read full log with `--debug` to see line + reason |
| `Error: argument cannot be a number for `pin`...`    | Pin path on `bpffs` must be a filesystem path under `/sys/fs/bpf/...`        |
| `cannot find map 'xyz' in object`                    | Map name typo or section attribute mismatch in BPF C                        |
| `Error: Failed to load program: Operation not permitted` | Missing capabilities OR `kernel.unprivileged_bpf_disabled=2`             |
| `error while loading shared libraries: libbpf.so.1`  | Missing `libbpf` package; `sudo apt install libbpf1` or rebuild bpftool statically |
| `BPF program is too large. Processed 1000001 insn`   | Hit verifier instruction limit (1M for privileged); split into tail-calls   |
| `Cannot allocate memory`                             | Hit `RLIMIT_MEMLOCK`; raise with `ulimit -l unlimited` or use `BPF_F_NUMA_NODE` accounting (kernel 5.11+ uses `memcg`) |
| `bpf_prog_attach: Invalid argument`                  | Attach type mismatch with program type — check `SEC()` name                 |
| `bpf_obj_pin: File exists`                           | Pin path already in use; `rm` it or pick a new name                          |
| `dump xlated: Operation not supported`               | Kernel built without `CONFIG_BPF_JIT` or `bpftool` lacks `libbfd`           |
| `Error: requested key length (X) is different from map's expected length (Y)` | Wrong key/value byte count in `lookup`/`update`           |
| `Error: link not found`                              | The link ID went away — likely auto-detached when its loader exited         |

### Read the full verifier log

```bash
sudo bpftool --debug prog load probe.bpf.o /sys/fs/bpf/probe 2>&1 | less
```

The verifier log is long but precisely identifies the failing instruction.

## Common Gotchas

### Gotcha 1: bpftool version mismatch with kernel

```bash
# Bad — Ubuntu 20.04 bpftool against a 6.x kernel:
$ bpftool version
bpftool v5.4.0           # missing `link`, `iter`, `gen min_core_btf`
$ uname -r
6.10.0-generic

# Fixed — use the kernel-matched binary:
$ sudo ln -sf /usr/lib/linux-tools/$(uname -r)/bpftool /usr/local/sbin/bpftool
$ bpftool version
bpftool v7.4.0
```

### Gotcha 2: relying on text output in scripts

```bash
# Bad — text format is unstable across versions:
$ bpftool prog show | awk '{print $2}' | grep kprobe
# Breaks when output adds/removes columns.

# Fixed — always use JSON for scripts:
$ bpftool prog show -j | jq '.[] | select(.type=="kprobe") | .id'
```

### Gotcha 3: pinning to `/tmp` instead of `/sys/fs/bpf`

```bash
# Bad — /tmp is tmpfs, not bpffs:
$ sudo bpftool prog load probe.bpf.o /tmp/probe
Error: pin 'path' must be on a bpffs

# Fixed — pin under /sys/fs/bpf:
$ sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe
```

### Gotcha 4: programs not being unloaded after rm

```bash
# Bad — process still holds an fd:
$ sudo rm /sys/fs/bpf/probe
$ sudo bpftool prog show id 12       # still there!
12: kprobe ...

# Fixed — find and stop the holder, then re-check:
$ sudo bpftool prog show id 12 --show-pids
12: ... pids my_loader(1234)
$ sudo kill 1234
$ sudo bpftool prog show id 12
Error: get by id (12): No such file or directory
```

### Gotcha 5: forgetting CO-RE relocations require BTF

```bash
# Bad — kernel built without BTF:
$ ls /sys/kernel/btf/vmlinux
ls: cannot access '/sys/kernel/btf/vmlinux': No such file or directory
$ sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe
libbpf: failed to find BTF for vmlinux

# Fixed — kernel must have CONFIG_DEBUG_INFO_BTF=y, or ship min.btf:
$ grep -E 'CONFIG_DEBUG_INFO_BTF|CONFIG_DEBUG_INFO_BTF_MODULES' /boot/config-$(uname -r)
CONFIG_DEBUG_INFO_BTF=y
CONFIG_DEBUG_INFO_BTF_MODULES=y
```

### Gotcha 6: missing capabilities on a setcap'd binary

```bash
# Bad — bpftool gained CAP_BPF but kernel needs CAP_PERF_EVENT for kprobe attach:
$ getcap $(which bpftool)
/usr/sbin/bpftool cap_bpf=ep
$ bpftool prog load kprobe.bpf.o /sys/fs/bpf/k
Error: failed to attach kprobe: Operation not permitted

# Fixed — grant the full set:
$ sudo setcap cap_bpf,cap_net_admin,cap_perf_event+ep $(which bpftool)
```

### Gotcha 7: forgetting to enable bpf_stats

```bash
# Bad — run_time_ns always 0:
$ sudo bpftool prog show id 12 -j | jq .run_time_ns
0

# Fixed — enable globally or per-program:
$ sudo sysctl -w kernel.bpf_stats_enabled=1
$ sudo bpftool prog profile id 12 duration 5 cycles instructions
```

### Gotcha 8: non-libbpf section names not recognized

```bash
# Bad — old "kprobe/sys_open" without trailing function name:
SEC("kprobe")          // bpftool can't infer attach point
$ sudo bpftool prog load probe.bpf.o /sys/fs/bpf/p
Error: bpftool: cannot infer attach type

# Fixed — use libbpf canonical sections:
SEC("kprobe/sys_open")
SEC("fentry/inet_listen")
SEC("xdp")
SEC("tracepoint/syscalls/sys_enter_open")
```

### Gotcha 9: legacy attach when link API is available

```bash
# Bad — uses legacy attach, loader process death detaches the program:
$ sudo bpftool cgroup attach /sys/fs/cgroup/myapp egress pinned /sys/fs/bpf/p

# Fixed — use bpftool link with pinning:
$ sudo bpftool prog attach pinned /sys/fs/bpf/p flow_dissector
$ sudo bpftool link pin id 5 /sys/fs/bpf/links/myapp_egress
# Now survives reboots / loader exits.
```

### Gotcha 10: `unprivileged_bpf_disabled` blocks userspace

```bash
# Bad — non-root user gets generic EPERM:
$ bpftool prog show
Error: can't get next program: Operation not permitted

# Investigate:
$ sysctl kernel.unprivileged_bpf_disabled
kernel.unprivileged_bpf_disabled = 2

# Fixed — either run as root, grant CAP_BPF, or relax sysctl (security tradeoff):
$ sudo sysctl -w kernel.unprivileged_bpf_disabled=0
```

### Gotcha 11: forgetting `pinmaps` separates maps from program pin

```bash
# Bad — pinmaps placed inside the same path as prog:
$ sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe \
    pinmaps /sys/fs/bpf/probe
Error: pin 'maps' is the same path as program pin

# Fixed — separate dir:
$ sudo mkdir -p /sys/fs/bpf/probe_maps
$ sudo bpftool prog load probe.bpf.o /sys/fs/bpf/probe \
    pinmaps /sys/fs/bpf/probe_maps
```

### Gotcha 12: trying to dump xlated without CONFIG_BPF_JIT_ALWAYS_ON

```bash
# Bad — JIT disabled so jited dump empty:
$ sudo sysctl net.core.bpf_jit_enable
net.core.bpf_jit_enable = 0
$ sudo bpftool prog dump jited id 12
no instructions returned

# Fixed:
$ sudo sysctl -w net.core.bpf_jit_enable=1
```

## Performance Considerations

### run_time_ns and run_cnt

Each program has two kernel-tracked counters when `kernel.bpf_stats_enabled=1`:

- `run_cnt` — total invocations
- `run_time_ns` — cumulative wall-clock nanoseconds inside the program

Mean cost per invocation = `run_time_ns / run_cnt`.

```bash
sudo sysctl -w kernel.bpf_stats_enabled=1
sudo bpftool prog show -j \
  | jq -r '.[] | select(.run_cnt>0) | "\(.id) \(.name) \(.run_time_ns/.run_cnt|floor)ns/call \(.run_cnt) calls"' \
  | sort -k3 -n -r | head
```

### prog profile

`bpftool prog profile` reads perf-event counters scoped to one program (kernel 5.7+):

```bash
sudo bpftool prog profile id 12 duration 5 cycles instructions

#  56301 run_cnt
#  4128411 cycles
#  6182733 instructions
#  73.1 ns mean run_time
```

Available counters: `cycles`, `instructions`, `l1d_loads`, `llc_misses`, `itlb_misses`, `dtlb_misses`.

### CONFIG_BPF_JIT impact

JIT-compiled programs run native machine code; interpreter mode is ~2-3x slower. Verify:

```bash
grep CONFIG_BPF_JIT /boot/config-$(uname -r)
# CONFIG_BPF_JIT=y
# CONFIG_BPF_JIT_ALWAYS_ON=y     # forces JIT, removes interpreter

sysctl net.core.bpf_jit_enable
# 0 = disabled (interpreter)
# 1 = enabled
# 2 = enabled + log JIT to dmesg (debug only)

sysctl net.core.bpf_jit_harden
# 0 = no hardening
# 1 = harden unprivileged
# 2 = harden everywhere (some perf cost)
```

### Memlock and memcg accounting

Pre-5.11 kernels charge map memory against `RLIMIT_MEMLOCK` (default 64 KiB on most distros). Modern kernels charge against the cgroup memory accounting (`memcg_account=1`).

```bash
ulimit -l                     # bytes
ulimit -l unlimited
```

Or for systemd services:

```ini
[Service]
LimitMEMLOCK=infinity
```

### Stack and instruction limits

| Limit                       | Default                                          |
|-----------------------------|--------------------------------------------------|
| Max instructions (verified) | 1,000,000 (CAP_BPF), 4,096 (unprivileged)        |
| Stack frame size            | 512 bytes per call                               |
| Max function call depth     | 8                                                |
| Max BPF map entries         | type-dependent (e.g., hash: `INT_MAX`)           |

### Tail calls and program arrays

Tail calls (`bpf_tail_call`) jump to another program in a `prog_array` map without growing the stack — bypass the 1M instruction limit by chaining up to 33 levels deep.

```bash
sudo bpftool map create /sys/fs/bpf/jumptable \
  type prog_array key 4 value 4 entries 16 name jumptable
sudo bpftool map update pinned /sys/fs/bpf/jumptable \
  key 0 0 0 0 \
  value pinned /sys/fs/bpf/handler_0
```

## Idioms

### Idiom: load → pin → inspect → modify → detach

The canonical lifecycle for any production BPF program — see the recipe above.

### Idiom: feature probe before deploying

Before shipping a BPF binary, check the target kernel:

```bash
sudo bpftool feature probe -j > target-kernel.json
diff <(jq -S . current-kernel.json) <(jq -S . target-kernel.json)
```

Fail-closed if a required helper or program type is missing.

### Idiom: libbpf + skeleton for development

The modern BPF userspace pattern:

```bash
clang -O2 -g -target bpf -c probe.bpf.c -o probe.bpf.o
bpftool gen skeleton probe.bpf.o > probe.skel.h
clang probe.c -lbpf -lelf -lz -o probe
```

Skip the legacy `bpf_object__find_program_by_name` dance; use `obj->progs.<name>` and `obj->maps.<name>`.

### Idiom: use bpf link, never legacy attach

For any new code:

```bash
# Yes:
sudo bpftool link pin id 5 /sys/fs/bpf/links/mylink

# Avoid (legacy, no link object):
sudo bpftool prog attach ...
```

The link API is supported by `cgroup`, `xdp`, `tcx`, `netkit`, `kprobe_multi`, `uprobe_multi`, `tracing`, `lsm`, `iter`, `flow_dissector`, `perf_event`.

### Idiom: pin maps in a known schema

Establish a repository convention:

```text
/sys/fs/bpf/<app>/programs/<name>
/sys/fs/bpf/<app>/maps/<name>
/sys/fs/bpf/<app>/links/<name>
```

This lets multiple processes find each other's state predictably.

### Idiom: ship min.btf for portability

For tooling distributed to systems that may lack BTF or run different kernel versions:

```bash
bpftool gen min_core_btf vmlinux.btf min.btf myprog.bpf.o
# Bundle min.btf with the binary; libbpf auto-discovers via $BTF or btf_custom_path.
```

### Idiom: prog dump xlated for verifier debugging

When the verifier rejects, the rewritten bytecode tells you what it inferred:

```bash
sudo bpftool --debug prog load failing.bpf.o /sys/fs/bpf/p 2>&1 | less
# Read the full verifier log.

# After load succeeds, compare actual rewrite to source intent:
sudo bpftool prog dump xlated id 12 linum
```

### Idiom: prog profile during load testing

Hot programs change cost under load; benchmark them:

```bash
sudo bpftool prog profile id 12 duration 30 \
  cycles instructions cache_misses
```

Compare under workload vs idle.

### Idiom: cgroup tree for observability

For a "what BPF is in the system" overview:

```bash
sudo bpftool cgroup tree
sudo bpftool net show
sudo bpftool perf show
sudo bpftool link show
```

Bookmark these four — they describe nearly every BPF attachment.

## Tips

- Always pass `-j` or `-p` and pipe to `jq` for any scripting; the human format is not a contract.
- `bpftool prog tracelog` tails `/sys/kernel/debug/tracing/trace_pipe` — the canonical place `bpf_printk()` output lands. Run as root.
- `bpftool map dump` prints raw bytes when the map lacks BTF; recompile your BPF C with `-g` to keep BTF in the ELF.
- `bpftool prog tracelog` is a thin wrapper — equivalent to `sudo cat /sys/kernel/debug/tracing/trace_pipe`. If `tracefs` isn't mounted: `sudo mount -t tracefs nodev /sys/kernel/debug/tracing`.
- `bpftool` itself is a BPF program loader and uses the standard libbpf code path — anything `bpftool` can do, your custom userspace can do.
- For interactive debugging of a misbehaving program, attach with `bpftool prog dump xlated id ... linum` first, then `dump jited` if the issue is performance, not logic.
- `bpftool` is reasonably stable under SIGINT — pinned objects survive Ctrl-C cleanly because pinning is durable on `bpffs`.
- `bpftool` supports tab completion via the `bash-completion` package; install it on your dev box:

```bash
sudo apt install -y bash-completion
sudo cp /usr/share/bash-completion/completions/bpftool ~/  # if you want to vendor it
```

- `bpftool batch file FILE` reads commands one per line — useful for atomic provisioning scripts:

```bash
cat > batch.txt <<'EOF'
prog load /tmp/p.bpf.o /sys/fs/bpf/p
map update pinned /sys/fs/bpf/p_maps/cfg key 0 0 0 0 value 1 0 0 0
cgroup attach /sys/fs/cgroup/myapp egress pinned /sys/fs/bpf/p
EOF
sudo bpftool batch file batch.txt
```

- The `--mapcompat` and `--legacy` flags exist for cross-version compatibility; usually you want neither, but they're a recovery path when the modern API misbehaves.
- When debugging a slow program, look at `run_cnt` deltas across two `prog show` calls 1s apart — if it isn't running, your hook is wrong.
- Use `bpftool prog tracelog` together with `bpftool map dump` to correlate userspace state with the kernel's view.
- Symbol names in `prog show` are truncated to 16 characters by the kernel; the `tag` field disambiguates programs that collide on name.
- Pinned paths are namespaces: programs pinned in one `bpffs` mount are not visible from another mount unless re-pinned.
- `bpftool` can output dot graphs for any program; combine with `xdot` for interactive viewing: `bpftool prog dump xlated id 12 visual | xdot -`.
- For continuous monitoring, run inside a `watch` or pipe to `jq -c` and append to a JSONL log.

## See Also

- [ebpf](./ebpf.md) — The kernel BPF subsystem; program types, maps, verifier, and the broader ecosystem
- [bpftrace](./bpftrace.md) — High-level tracing language built on top of BPF, complementing low-level bpftool inspection
- [perf](./perf.md) — Linux performance counters; integrates with BPF via `prog profile` and `perf show`
- [polyglot](../languages/polyglot.md) — Cross-language reference; libbpf in C, Cilium ebpf in Go, libbpf-rs in Rust
- [bash](../languages/bash.md) — Shell scripting for the bpftool/jq pipelines documented above

## References

- `man 8 bpftool` — top-level manual page
- `man 8 bpftool-prog`, `bpftool-map`, `bpftool-cgroup`, `bpftool-perf`, `bpftool-net`, `bpftool-feature`, `bpftool-btf`, `bpftool-gen`, `bpftool-iter`, `bpftool-link`, `bpftool-struct_ops` — per-subcommand pages
- Kernel source: `tools/bpf/bpftool/` — authoritative implementation
- `Documentation/bpf/` in the kernel tree — the BPF reference
- `https://ebpf.io/` — community portal, tutorials, project index
- `https://docs.cilium.io/en/stable/bpf/` — extensive BPF reference and architecture notes
- `https://www.brendangregg.com/ebpf.html` — Brendan Gregg's collected BPF posts
- `https://github.com/iovisor/bcc/blob/master/docs/reference_guide.md` — BCC Reference Guide (helpers, prog types, map types)
- `https://github.com/libbpf/libbpf` — libbpf source, the userspace API bpftool is built on
- `https://github.com/libbpf/bpftool` — standalone bpftool repository (kept in sync with kernel tree)
- `https://nakryiko.com/posts/libbpf-bootstrap/` — Andrii Nakryiko's libbpf-bootstrap walkthrough
- `https://github.com/cilium/ebpf` — Go BPF library, complementary to bpftool inspection
- `https://lwn.net/Kernel/Index/#Berkeley_Packet_Filter` — LWN's curated BPF article index
