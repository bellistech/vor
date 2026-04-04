# The Mathematics of parted — Advanced Partition Management

> *parted handles partition creation, resizing, and alignment beyond fdisk's capabilities. The math covers optimal alignment calculations, resize operations, filesystem geometry, and partition table conversions.*

---

## 1. Optimal Alignment — The Performance Foundation

### The Model

parted's `--align optimal` ensures partitions align to physical sector boundaries, SSD erase blocks, and RAID stripe widths.

### Alignment Calculation

$$\text{Aligned Start} = \lceil \frac{\text{Desired Start}}{\text{Optimal I/O Size}} \rceil \times \text{Optimal I/O Size}$$

parted reads alignment info from:

$$\text{Optimal I/O Size} = \max(\text{physical\_sector\_size}, \text{optimal\_io\_size}, 1 \text{ MiB})$$

These values come from `/sys/block/sdX/queue/`:

| Parameter | Typical Values | Source |
|:---|:---:|:---|
| `logical_block_size` | 512, 4096 | Smallest addressable unit |
| `physical_block_size` | 512, 4096 | Physical sector size |
| `optimal_io_size` | 0, 65536, 1048576 | Optimal I/O alignment |
| `alignment_offset` | 0 | Offset for alignment |

### Alignment Checking

$$\text{Is Aligned} = (\text{Start Sector} \times \text{Sector Size}) \mod \text{Optimal I/O Size} = 0$$

### Worked Examples

| Disk Type | Physical Sector | Optimal I/O | First Aligned Sector (512b logical) |
|:---|:---:|:---:|:---:|
| 512b native | 512 | 1 MiB | 2,048 |
| 4Kn (4K native) | 4,096 | 1 MiB | 256 (at 4K sectors) |
| 512e (4K emulated) | 4,096 | 1 MiB | 2,048 (at 512b sectors) |
| RAID (256K stripe) | 512 | 256 KiB | 2,048 |
| SSD (4MB erase) | 4,096 | 4 MiB | 8,192 |

---

## 2. Partition Resizing — Safe Boundaries

### The Model

Resizing a partition requires three separate operations: resize filesystem, resize partition, then (optionally) resize filesystem again to fill.

### Shrink Calculation

$$\text{New Partition End} = \text{Start} + \left\lceil \frac{\text{New Size}}{\text{Sector Size}} \right\rceil - 1$$

$$\text{Free Space Created} = \text{Old Size} - \text{New Size}$$

### Grow Calculation

$$\text{Max Growth} = \text{Next Partition Start} - \text{Current End} - 1 \text{ sector}$$

$$\text{New Size} = \text{Old Size} + \text{Max Growth}$$

### Worked Example

*"Partition at sector 2048, size 200 GiB, next partition at sector 419,432,448."*

$$\text{Current End} = 2048 + \frac{200 \times 2^{30}}{512} - 1 = 2048 + 419,430,400 - 1 = 419,432,447$$

$$\text{Gap to Next} = 419,432,448 - 419,432,447 - 1 = 0 \text{ sectors (no room)}$$

| Operation | Safety Check | Risk |
|:---|:---|:---|
| Shrink filesystem, then partition | FS must fit in new size | Data loss if FS > new size |
| Grow partition, then filesystem | Free space must exist after | None if space available |
| Move partition | Copy all data | Slow, power loss = data loss |

---

## 3. Partition Table Conversion — MBR to GPT

### Space Requirements

| Table | Max Partitions | Max Disk Size | Overhead |
|:---|:---:|:---:|:---:|
| MBR | 4 primary (or 3+extended) | 2 TiB | 512 bytes |
| GPT | 128 (default) | 9.4 ZiB | ~33.5 KiB |

### Conversion Feasibility Check

$$\text{Can Convert MBR→GPT if:}$$

$$\text{1. Sectors 1-33 are free (GPT header + entries)}$$

$$\text{2. Last 33 sectors are free (backup GPT)}$$

$$\text{3. No partitions overlap these regions}$$

### Logical Partition to Primary Conversion

MBR extended partitions use a linked list of EBR (Extended Boot Record) headers:

$$\text{EBR Overhead per Logical} = 1 \text{ sector (512 bytes)}$$

$$\text{Usable in Logical} = \text{Partition Size} - 512 \text{ bytes}$$

---

## 4. Free Space Analysis

### The Model

parted can identify unallocated gaps between partitions.

### Gap Calculation

$$\text{Gap}_i = \text{Start}_{i+1} - \text{End}_i - 1 \text{ sectors}$$

$$\text{Total Free} = \sum_{i=0}^{n} \text{Gap}_i$$

### Worked Example

*"1 TiB disk with 3 partitions."*

| Region | Start Sector | End Sector | Size |
|:---|:---:|:---:|:---:|
| GPT header | 0 | 2,047 | 1 MiB |
| Partition 1 | 2,048 | 1,050,623 | 512 MiB |
| **Gap 1** | **1,050,624** | **2,099,199** | **512 MiB** |
| Partition 2 | 2,099,200 | 1,050,673,151 | 500 GiB |
| Partition 3 | 1,050,673,152 | 2,097,151,999 | 499 GiB |
| **Gap 2** | **2,097,152,000** | **2,147,483,614** | **24 GiB** |
| Backup GPT | 2,147,483,615 | 2,147,483,647 | 16.5 KiB |

$$\text{Total Free} = 512 \text{ MiB} + 24 \text{ GiB} = 24.5 \text{ GiB}$$

---

## 5. Filesystem-Aware Operations

### Resize Limits by Filesystem

| Filesystem | Online Grow | Online Shrink | Offline Shrink | Min Size |
|:---|:---:|:---:|:---:|:---|
| ext4 | Yes | No | Yes | Used space + overhead |
| XFS | Yes | No | No | Cannot shrink |
| Btrfs | Yes | Yes | Yes | Used space |
| NTFS | No | No | Yes | Used space + MFT |
| FAT32 | No | No | Yes | Used space |

### Minimum Partition Size Formula

$$\text{Min Size} = \text{Used Data} + \text{FS Metadata} + \text{Reserved Blocks}$$

For ext4:

$$\text{Min Size} = \text{Used} + \text{Inode Tables} + \text{Journal} + \text{Superblocks} + \text{Reserved}$$

### Worked Example

*"ext4 partition: 500 GiB, 200 GiB used, 5% reserved."*

$$\text{Metadata} \approx 8 \text{ GiB (inode tables + journal + super)}$$

$$\text{Reserved} = 500 \times 0.05 = 25 \text{ GiB}$$

$$\text{Min Size} \approx 200 + 8 + 25 = 233 \text{ GiB}$$

After removing reserved blocks (`tune2fs -m 0`):

$$\text{Min Size} \approx 200 + 8 = 208 \text{ GiB}$$

---

## 6. Partition Flags and Boot Requirements

### UEFI Boot Partition Sizing

$$\text{ESP Size} \geq \text{Kernels} \times \text{Kernel Size} + \text{Bootloader} + \text{Safety Margin}$$

| Components | Size | Total |
|:---|:---:|:---:|
| 2 kernels (initramfs) | 2 × 50 MiB | 100 MiB |
| GRUB/systemd-boot | 10 MiB | 10 MiB |
| Safety margin | 100 MiB | 100 MiB |
| **Minimum ESP** | | **210 MiB** |
| **Recommended ESP** | | **512 MiB** |

### Partition Flags

| Flag | Purpose | Required For |
|:---|:---|:---|
| boot | Legacy BIOS bootable | MBR boot partition |
| esp | EFI System Partition | UEFI boot |
| lvm | LVM Physical Volume | LVM setup |
| raid | Software RAID | mdadm |
| swap | Swap partition | Swap identification |
| bios_grub | BIOS boot partition | GPT + legacy BIOS |

---

## 7. Summary of Functions by Type

| Formula | Math Type | Application |
|:---|:---|:---|
| $\lceil \frac{x}{\text{align}} \rceil \times \text{align}$ | Ceiling alignment | Optimal alignment |
| $\text{Start} + \frac{\text{Size}}{\text{Sector}} - 1$ | Linear arithmetic | Partition end sector |
| $\text{Start}_{i+1} - \text{End}_i - 1$ | Subtraction | Gap calculation |
| $\text{Used} + \text{Meta} + \text{Reserved}$ | Addition | Minimum partition size |
| $\text{Sector} \times \text{Size} \mod \text{Align}$ | Modular arithmetic | Alignment check |

---

*Every `parted mkpart`, `resizepart`, and `align-check` operates on sector-level arithmetic — integer math that maps physical geometry to logical partitions with precision that matters for performance.*

## Prerequisites

- Partition table formats (MBR, GPT)
- Sector and block size concepts
- Alignment requirements for SSDs and 4K-sector drives
- Root/sudo access for partition operations

## Complexity

- **Beginner:** Creating and listing partitions
- **Intermediate:** Optimal alignment calculations, partition resizing
- **Advanced:** Sector-level arithmetic for precise placement, GPT header/entry structure analysis
