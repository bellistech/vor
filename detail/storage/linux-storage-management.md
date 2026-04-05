# Linux Storage Management — Architecture and Theory

> *Modern Linux storage extends far beyond simple block devices. Device mapper provides a kernel framework for virtual block devices, multipath I/O adds redundancy, iSCSI bridges remote storage over IP, and technologies like Stratis and VDO provide next-generation storage management with deduplication and compression.*

---

## 1. Device Mapper Kernel Framework

### Architecture

Device mapper (dm) is a **kernel framework** that maps virtual block devices to underlying physical devices through a table of target mappings:

$$virtual\_device \xrightarrow{mapping\ table} physical\_device(s)$$

Each mapping entry defines:

| Field | Description |
|:---|:---|
| Start sector | Beginning of mapped region |
| Length | Size in 512-byte sectors |
| Target type | linear, striped, mirror, crypt, etc. |
| Target args | Device paths, offsets, parameters |

### Target Types

The core dm targets form the basis for LVM, LUKS, and multipath:

| Target | Purpose | Used By |
|:---|:---|:---|
| `linear` | Simple remapping of sector ranges | LVM linear LVs |
| `striped` | Distribute I/O across N devices | LVM striped LVs |
| `mirror` | Synchronous mirroring (dm-raid1) | LVM mirror LVs |
| `crypt` | Transparent encryption | LUKS/dm-crypt |
| `multipath` | Multiple paths to same device | DM-multipath |
| `snapshot` | Copy-on-write snapshots | LVM snapshots |
| `thin-pool` | Thin provisioning pool | LVM thin pools |
| `cache` | SSD caching tier | dm-cache, lvmcache |
| `raid` | MD-style RAID via dm | LVM RAID LVs |
| `zero` | Discards writes, reads zeros | Testing |
| `error` | Returns I/O errors | Testing, fencing |

### I/O Path

```
Application I/O
    ↓
VFS / Filesystem
    ↓
Block Layer (submit_bio)
    ↓
Device Mapper (dm_table_map)
    ↓ (table lookup → target handler)
Target-specific map function
    ↓
Physical block device(s)
```

Each bio (block I/O) is remapped by the target's `map()` function before being submitted to the underlying device(s).

### Stacking

Device mapper targets can be stacked arbitrarily:

$$\text{application} \to \text{dm-crypt} \to \text{dm-thin} \to \text{dm-multipath} \to \text{physical}$$

This stacking is how LVM over LUKS over multipath works: each layer is a dm device consuming the one below it.

---

## 2. Multipath I/O Architecture

### Problem Statement

Enterprise storage arrays expose LUNs through multiple physical paths (HBAs, switches, ports). Without multipath, the kernel sees each path as a separate block device:

$$n\_paths = n\_HBAs \times n\_switches \times n\_target\_ports$$

A dual-HBA, dual-switch, dual-port setup yields $2 \times 2 \times 2 = 8$ apparent devices for a single LUN.

### Multipath Components

```
┌──────────────────────────────────┐
│         multipathd (daemon)       │
│  Monitors paths, triggers failover│
└───────────────┬──────────────────┘
                │ netlink / uevent
┌───────────────▼──────────────────┐
│     dm-multipath (kernel target)  │
│  Maps /dev/mapper/mpathX to paths │
└───────────────┬──────────────────┘
        ┌───────┴───────┐
   /dev/sdb         /dev/sdc
   (path 0)         (path 1)
   HBA0→SW0→Port0   HBA1→SW1→Port0
```

### Path Selectors

The path selector determines which path gets the next I/O within an active priority group:

| Selector | Algorithm | Best For |
|:---|:---|:---|
| `round-robin` | Rotate through paths equally | Uniform path latency |
| `queue-length` | Choose path with least queued I/O | Varying path congestion |
| `service-time` | Estimate and choose shortest service time | Mixed path bandwidths |

### Failover Algorithms

**Priority groups** organize paths by preference:

| Policy | Grouping | Use Case |
|:---|:---|:---|
| `multibus` | All paths in one group | Active/active arrays |
| `failover` | One path per group | Active/passive arrays |
| `group_by_prio` | Group by ALUA priority | Arrays with ALUA support |
| `group_by_serial` | Group by controller serial | Dual-controller arrays |

**Failback modes:**

- `immediate` — switch back to higher-priority group as soon as path recovers
- `manual` — stay on current group until admin intervenes
- `followover` — only failback when another node fails over first
- `N` (seconds) — delay N seconds before failback

### ALUA (Asymmetric Logical Unit Access)

ALUA reports per-path access states from the storage array:

| State | Meaning | I/O |
|:---|:---|:---|
| Active/Optimized | Preferred path, lowest latency | Full speed |
| Active/Non-optimized | Functional but not preferred | Higher latency |
| Standby | Available after transition | Activate first |
| Unavailable | Path is down | No I/O |

The `prio alua` setting in multipath.conf enables automatic path prioritization based on these states.

---

## 3. iSCSI Protocol Stack

### Protocol Layers

```
┌──────────────┐
│  SCSI CDB    │  SCSI command descriptor block
├──────────────┤
│  iSCSI PDU   │  Protocol data unit (encapsulation)
├──────────────┤
│  TCP         │  Reliable transport (port 3260)
├──────────────┤
│  IP          │  Network layer
├──────────────┤
│  Ethernet    │  Data link
└──────────────┘
```

### Key Concepts

| Term | Definition |
|:---|:---|
| **Initiator** | Client that sends SCSI commands (the host) |
| **Target** | Server that receives commands (the storage) |
| **Portal** | IP:port combination for connections |
| **TPG** | Target Portal Group — set of portals |
| **LUN** | Logical Unit Number — exported block device |
| **IQN** | iSCSI Qualified Name (globally unique) |
| **ISID** | Initiator Session ID |
| **TSIH** | Target Session Identifying Handle |

### IQN Format

$$\text{iqn.YYYY-MM.reversed\_domain:identifier}$$

Example: `iqn.2024-01.com.example:storage.lun0`

### Session and Connection Model

A single **session** between initiator and target can have multiple **connections** (MC/S — Multiple Connections per Session):

$$\text{Session} = \{Connection_1, Connection_2, ..., Connection_n\}$$

Each connection is a TCP socket. Multiple connections enable load balancing at the iSCSI layer.

### Authentication

iSCSI supports CHAP (Challenge-Handshake Authentication Protocol):

- **One-way CHAP:** Target authenticates initiator
- **Mutual CHAP:** Both sides authenticate each other
- **No authentication:** Open access (not recommended for production)

### Error Recovery Levels

| Level | Recovery Scope |
|:---|:---|
| 0 | Session recovery (drop and reconnect) |
| 1 | Digest recovery (retransmit PDU) |
| 2 | Connection recovery (reconnect within session) |

---

## 4. LIO Target Architecture

### Kernel Target Framework

LIO (Linux-IO) is the in-kernel iSCSI target implementation since Linux 3.1:

```
┌──────────────────────────────────┐
│         targetcli (userspace)     │
│    configfs-based management      │
└───────────────┬──────────────────┘
                │ configfs
┌───────────────▼──────────────────┐
│        target_core_mod            │
│   Fabric-independent SCSI engine  │
├──────────────────────────────────┤
│  Fabric modules:                  │
│  iscsi_target, tcm_fc, srpt,     │
│  tcm_loop, vhost-scsi            │
├──────────────────────────────────┤
│  Backstore handlers:             │
│  iblock, fileio, pscsi, ramdisk  │
└──────────────────────────────────┘
```

### Backstore Types

| Type | Source | Characteristics |
|:---|:---|:---|
| `block` (iblock) | Block device | Direct I/O, best performance |
| `fileio` | File on filesystem | Flexible, file-backed |
| `pscsi` | Pass-through SCSI | Raw SCSI device passthrough |
| `ramdisk` | RAM | Testing, volatile |

### Configuration Hierarchy

```
/
├── backstores/
│   ├── block/
│   ├── fileio/
│   ├── pscsi/
│   └── ramdisk/
└── iscsi/
    └── iqn.2024-01.com.example:storage/
        └── tpg1/
            ├── portals/    ← IP:port listeners
            ├── luns/       ← exported backstores
            └── acls/       ← initiator access control
                └── iqn.2024-01.com.example:client01/
                    └── mapped_lun0 → lun0
```

---

## 5. NFS Protocol Versions Comparison

### Version Evolution

| Feature | NFSv3 | NFSv4.0 | NFSv4.1 | NFSv4.2 |
|:---|:---|:---|:---|:---|
| Transport | UDP/TCP | TCP only | TCP only | TCP only |
| Port | 2049 + portmap | 2049 only | 2049 only | 2049 only |
| State | Stateless | Stateful | Stateful | Stateful |
| Locking | NLM (separate) | Built-in | Built-in | Built-in |
| Security | AUTH_SYS, Kerberos | Kerberos mandatory | Kerberos mandatory | Kerberos mandatory |
| Delegation | No | Yes | Yes (enhanced) | Yes (enhanced) |
| pNFS | No | No | Yes | Yes |
| Server-side copy | No | No | No | Yes |
| Sparse files | No | No | No | Yes |

### NFSv4 Stateful Model

NFSv4 maintains **state** on the server through:

- **Client ID** — uniquely identifies the client
- **State ID** — tracks open files and locks
- **Lease** — time-limited grant that must be renewed

$$\text{if } t_{current} - t_{last\_renewal} > lease\_period \implies \text{state expired}$$

Default lease period: 90 seconds. Clients renew via any operation or explicit RENEW.

### pNFS (Parallel NFS)

NFSv4.1 introduced pNFS for parallel data access:

```
Client → Metadata Server (MDS): LAYOUTGET
         MDS returns layout map
Client → Data Server 1: READ/WRITE (direct)
Client → Data Server 2: READ/WRITE (direct)
Client → Data Server N: READ/WRITE (direct)
```

Layout types: files, blocks, objects, flexfiles.

---

## 6. Stratis Architecture

### Design Philosophy

Stratis provides a **volume-managing filesystem** — combining the roles of LVM + XFS into a unified management layer:

```
┌─────────────────────────────┐
│     stratis-cli (D-Bus)      │
├─────────────────────────────┤
│     stratisd (daemon)        │
│  Pool management, snapshots  │
├─────────────────────────────┤
│     Device Mapper layers     │
│  dm-thin + dm-cache + dm-*   │
├─────────────────────────────┤
│     XFS (auto-formatted)     │
├─────────────────────────────┤
│     Block devices            │
└─────────────────────────────┘
```

### Key Properties

| Property | Value |
|:---|:---|
| Filesystem type | Always XFS |
| Provisioning | Thin (overprovisioned) |
| Snapshots | Copy-on-write via dm-thin |
| Cache | dm-cache for SSD tiering |
| Encryption | LUKS2 per-pool (optional) |
| Redundancy | Not yet (planned dm-integrity + RAID) |

### Thin Provisioning Model

Stratis uses dm-thin-pool for thin provisioning:

$$virtual\_size \gg physical\_size$$

A pool with 100 GB physical storage can host multiple filesystems claiming 1 TB each. Actual allocation happens on write.

**Monitoring is critical** — if the thin pool fills, I/O errors occur:

$$\text{if } used\_data \geq pool\_capacity \implies \text{ENOSPC}$$

---

## 7. VDO Deduplication and Compression Theory

### Deduplication

VDO uses a **Universal Deduplication Service (UDS)** index to identify duplicate blocks:

$$hash(block) \to UDS\_index \xrightarrow{?} existing\_block\_reference$$

Process:
1. Compute MurmurHash3 of incoming 4 KB block
2. Query UDS index for hash match
3. If match: increment reference count, skip write
4. If no match: write block, add to index

### Dedup Index Sizing

The UDS index is stored in the first portion of the VDO device:

$$index\_size = \frac{deduplicated\_data}{average\_block\_per\_entry} \times entry\_size$$

Each index entry: ~16 bytes. Index types:

| Type | Memory | Blocks Indexed | Use Case |
|:---|:---|:---|:---|
| Dense | 1 GB RAM per 1 TB data | All blocks | Maximum dedup |
| Sparse | 256 MB RAM per 1 TB data | Recent blocks | Lower memory footprint |

### Compression

After deduplication, VDO applies **LZ4 compression** to unique blocks:

$$compressed\_size = original\_size \times (1 - compression\_ratio)$$

VDO packs multiple compressed blocks into a single 4 KB physical block:

$$packing\_ratio = \frac{4096}{\sum compressed\_block\_sizes}$$

### Space Savings Calculation

$$savings = 1 - \frac{physical\_used}{logical\_used}$$

$$effective\_capacity = \frac{physical\_capacity}{1 - savings}$$

Example: 50 GB physical, 65% savings:

$$effective = \frac{50}{1 - 0.65} = \frac{50}{0.35} \approx 142.9 \text{ GB}$$

### Write Path

```
Write request (4 KB block)
    ↓
Compute hash → query UDS index
    ↓
┌── Duplicate found ──→ Reference existing block (skip write)
└── Unique block
        ↓
    LZ4 compress
        ↓
    Pack compressed fragments into physical blocks
        ↓
    Write to physical media + update index
```

---

## 8. LUKS Key Management

### LUKS2 Header Structure

```
┌──────────────────────────┐ Sector 0
│  LUKS2 Primary Header     │
│  (JSON metadata area)     │
├──────────────────────────┤
│  Keyslot Area             │
│  (up to 32 keyslots)     │
│  Each: AF-split key +     │
│        PBKDF2/Argon2id    │
├──────────────────────────┤
│  LUKS2 Secondary Header   │
│  (redundant copy)         │
├──────────────────────────┤
│  Encrypted Data           │
│  (AES-XTS-plain64)       │
└──────────────────────────┘
```

### Key Derivation

LUKS derives the master key from the passphrase using a key derivation function:

$$master\_key = \text{KDF}(passphrase, salt, iterations)$$

| LUKS Version | Default KDF | Parameters |
|:---|:---|:---|
| LUKS1 | PBKDF2-SHA256 | Iterations (time-based) |
| LUKS2 | Argon2id | Memory, iterations, parallelism |

Argon2id is **memory-hard**, resisting GPU/ASIC attacks:

$$cost_{attack} \propto memory \times time \times parallelism$$

### Anti-Forensic Splitter

Each keyslot stores the master key through an **anti-forensic splitter** (AF-split):

$$split\_key = AF\text{-}split(master\_key, stripes)$$

Default: 4000 stripes. This ensures that even partial recovery of a keyslot cannot reconstruct the master key.

### Encryption Modes

| Cipher | Mode | Key Size | Notes |
|:---|:---|:---|:---|
| AES | XTS | 256/512 bit | Default, hardware-accelerated (AES-NI) |
| AES | CBC-ESSIV | 256 bit | Legacy, vulnerable to watermarking |
| Serpent | XTS | 256/512 bit | Conservative alternative |
| Twofish | XTS | 256/512 bit | Alternative |

XTS mode provides **tweakable encryption** where each sector gets a unique tweak, preventing identical plaintext blocks from producing identical ciphertext.

---

## References

- kernel.org: Documentation/admin-guide/device-mapper/
- kernel.org: Documentation/device-mapper/dm-raid.txt
- RFC 7143 — Internet Small Computer System Interface (iSCSI) Protocol
- RFC 7530 — NFS Version 4 Protocol
- RFC 8881 — NFS Version 4.1 Protocol
- LUKS2 On-Disk Format Specification
- Red Hat Storage Administration Guide
- Stratis Project Documentation (stratis-storage.github.io)
