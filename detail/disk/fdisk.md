# The Mathematics of fdisk — Partition Table Geometry

> *fdisk manipulates partition tables (MBR and GPT). The math covers CHS-to-LBA translation, partition alignment, MBR address limits, and GPT capacity calculations.*

---

## 1. CHS to LBA Translation — Legacy Geometry

### The Model

Legacy MBR uses Cylinder-Head-Sector (CHS) addressing. Modern disks use Logical Block Addressing (LBA).

### CHS to LBA Formula

$$\text{LBA} = (C \times H_{max} + H) \times S_{max} + (S - 1)$$

Where:
- $C$ = cylinder number (0-based)
- $H$ = head number (0-based)
- $S$ = sector number (1-based)
- $H_{max}$ = heads per cylinder (typically 255)
- $S_{max}$ = sectors per track (typically 63)

### Worked Example

*"CHS = (100, 50, 30), with 255 heads and 63 sectors per track."*

$$\text{LBA} = (100 \times 255 + 50) \times 63 + (30 - 1)$$

$$= (25,500 + 50) \times 63 + 29$$

$$= 25,550 \times 63 + 29$$

$$= 1,609,650 + 29 = 1,609,679$$

### CHS Address Limit

MBR CHS fields: 10 bits cylinder, 8 bits head, 6 bits sector:

$$\text{Max CHS} = 1024 \times 256 \times 63 = 16,515,072 \text{ sectors}$$

$$\text{Max CHS Size} = 16,515,072 \times 512 = 8,455,716,864 \text{ bytes} \approx 7.875 \text{ GiB}$$

**This is the CHS barrier — why MBR disks historically had issues above ~8 GiB.**

---

## 2. MBR Partition Limits

### LBA Addressing in MBR

MBR uses 32-bit LBA fields:

$$\text{Max LBA Sectors} = 2^{32} = 4,294,967,296$$

$$\text{Max Disk Size} = 2^{32} \times 512 = 2,199,023,255,552 \text{ bytes} = 2 \text{ TiB}$$

### MBR Structure

| Field | Offset | Size | Purpose |
|:---|:---:|:---:|:---|
| Boot code | 0 | 446 bytes | Bootloader |
| Partition 1 | 446 | 16 bytes | First partition entry |
| Partition 2 | 462 | 16 bytes | Second partition entry |
| Partition 3 | 478 | 16 bytes | Third partition entry |
| Partition 4 | 494 | 16 bytes | Fourth partition entry |
| Signature | 510 | 2 bytes | 0x55AA |

Each partition entry:

| Field | Size | Meaning |
|:---|:---:|:---|
| Status | 1 byte | Active/bootable flag |
| CHS Start | 3 bytes | Legacy start address |
| Type | 1 byte | Partition type (0x83=Linux, 0x82=swap) |
| CHS End | 3 bytes | Legacy end address |
| LBA Start | 4 bytes | Start sector (32-bit) |
| LBA Size | 4 bytes | Sector count (32-bit) |

$$\text{Max Partition Size} = 2^{32} \times 512 = 2 \text{ TiB}$$

---

## 3. GPT — GUID Partition Table

### GPT Capacity

GPT uses 64-bit LBA addressing:

$$\text{Max Disk Size} = 2^{64} \times 512 = 9.4 \text{ ZiB (zettabytes)}$$

With 4 KiB sectors:

$$\text{Max Disk Size} = 2^{64} \times 4096 = 75.6 \text{ ZiB}$$

### GPT Structure

| Area | Location | Size |
|:---|:---|:---|
| Protective MBR | LBA 0 | 1 sector (512 bytes) |
| GPT Header | LBA 1 | 1 sector |
| Partition Entries | LBA 2-33 | 32 sectors (128 entries × 128 bytes) |
| Data Area | LBA 34 to N-34 | Disk capacity |
| Backup Entries | LBA N-33 to N-2 | 32 sectors |
| Backup Header | LBA N-1 | 1 sector |

$$\text{GPT Overhead} = 34 + 33 = 67 \text{ sectors} = 34,304 \text{ bytes} \approx 33.5 \text{ KiB}$$

$$\text{Max Partitions (default)} = 128$$

$$\text{Partition Entry Size} = 128 \text{ bytes each}$$

---

## 4. Partition Alignment — Performance Math

### The Model

Modern drives have physical sectors of 4 KiB (4Kn or 512e). Misaligned partitions cause read-modify-write penalties.

### Alignment Formula

$$\text{Aligned Start} = \lceil \frac{\text{Desired Start}}{\text{Alignment}} \rceil \times \text{Alignment}$$

Standard alignment: 1 MiB (2048 sectors at 512 bytes/sector).

### Why 1 MiB Alignment

$$1 \text{ MiB} = 2048 \times 512 = 1,048,576 \text{ bytes}$$

This aligns to:
- 4 KiB physical sectors ($1 \text{ MiB} / 4 \text{ KiB} = 256$ — integer)
- 8 KiB physical sectors ($1 \text{ MiB} / 8 \text{ KiB} = 128$ — integer)
- SSD erase block sizes (128 KiB - 4 MiB — all divide evenly)
- RAID stripe widths (64 KiB - 1 MiB — all divide evenly)

### Misalignment Penalty

$$\text{Misaligned Write} = 2 \times \text{Aligned Write} \quad (\text{read-modify-write on both physical sectors})$$

$$\text{Performance Loss} \approx 50\% \text{ for random writes}$$

| Scenario | Aligned IOPS | Misaligned IOPS | Penalty |
|:---|:---:|:---:|:---:|
| HDD random write | 150 | 75 | 50% |
| SSD random write | 50,000 | 25,000 | 50% |
| Sequential write | ~0% penalty | ~0% penalty | Minimal |

---

## 5. Partition Size Calculations

### Usable Space

$$\text{Usable} = \text{Disk Size} - \text{Partition Table} - \text{Alignment Padding}$$

For GPT with 1 MiB alignment:

$$\text{Usable} = \text{Disk Size} - 1 \text{ MiB (start align)} - 33.5 \text{ KiB (backup GPT)}$$

### Sector Arithmetic

$$\text{Partition Size (bytes)} = \text{Sector Count} \times \text{Sector Size}$$

$$\text{Sectors Needed} = \lceil \frac{\text{Desired Size}}{\text{Sector Size}} \rceil$$

### Worked Example

*"2 TiB disk, GPT, three partitions: 512 MiB boot, 32 GiB swap, rest for data."*

$$\text{Total Sectors} = \frac{2 \times 2^{40}}{512} = 4,294,967,296$$

| Partition | Start Sector | Size (sectors) | Size |
|:---|:---:|:---:|:---:|
| GPT header | 0 | 2,048 | 1 MiB |
| /boot (EFI) | 2,048 | 1,048,576 | 512 MiB |
| swap | 1,050,624 | 67,108,864 | 32 GiB |
| /data | 68,159,488 | 4,226,740,574 | ~1.965 TiB |
| Backup GPT | 4,294,900,062 | 67,234 | 33.5 KiB |

---

## 6. Partition Type Codes

### Common MBR Types

| Hex Code | Type | Notes |
|:---:|:---|:---|
| 0x00 | Empty | Unused entry |
| 0x82 | Linux swap | Swap partition |
| 0x83 | Linux | Standard Linux filesystem |
| 0x8e | Linux LVM | LVM physical volume |
| 0xfd | Linux RAID | mdadm auto-detect |
| 0xef | EFI System | UEFI boot partition |
| 0x07 | NTFS/exFAT | Windows |

### GPT Type GUIDs

| GUID | Type |
|:---|:---|
| C12A7328-F81F-11D2-BA4B-00A0C93EC93B | EFI System |
| 0FC63DAF-8483-4772-8E79-3D69D8477DE4 | Linux filesystem |
| E6D6D379-F507-44C2-A23C-238F2A3DF928 | Linux LVM |
| A19D880F-05FC-4D3B-A006-743F0F84911E | Linux RAID |
| 0657FD6D-A4AB-43C4-84E5-0933C84B4F4F | Linux swap |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $(C \times H + h) \times S + s - 1$ | Linear combination | CHS to LBA |
| $2^{32} \times 512$ | Exponential / constant | MBR size limit |
| $2^{64} \times \text{sector}$ | Exponential | GPT capacity |
| $\lceil \frac{x}{\text{align}} \rceil \times \text{align}$ | Ceiling alignment | Partition alignment |
| $\text{Sectors} \times \text{Size}$ | Linear arithmetic | Partition sizing |

---

*Every `fdisk -l`, `gdisk`, and `parted print` reads the first and last sectors of the disk — a 67-sector data structure that maps the entire drive's layout.*
