# The Mathematics of GRUB — Boot Process, Disk Geometry & Partition Addressing

> *GRUB bridges the gap between firmware and kernel — translating sector addresses, loading compressed images, and managing the boot sequence with precise timing constraints. Every boot begins with disk geometry math.*

---

## 1. Boot Sequence Timing

### Stage Transitions

GRUB uses a multi-stage boot process:

| Stage | Location | Size | Load Time (HDD) |
|:---|:---|:---:|:---:|
| MBR/Stage 1 | Sector 0 | 446 bytes | 0 (firmware loads) |
| Stage 1.5 | Post-MBR gap | 32 KB | ~5 ms |
| Stage 2 (core.img) | Filesystem | 50-500 KB | 10-50 ms |
| Kernel + initrd | Filesystem | 10-50 MB | 500-2000 ms |

### Total GRUB Overhead

$$T_{GRUB} = T_{stage1} + T_{stage1.5} + T_{stage2} + T_{menu} + T_{kernel\_load}$$

$$T_{GRUB} \approx 0 + 5ms + 30ms + T_{menu} + 1000ms$$

With no menu timeout: $T_{GRUB} \approx 1 \text{ second}$. With 5-second timeout: $T_{GRUB} \approx 6 \text{ seconds}$.

---

## 2. Disk Addressing — CHS to LBA

### Legacy CHS (Cylinder-Head-Sector)

$$LBA = (C \times H_{max} + H) \times S_{max} + S - 1$$

Where:
- $C$ = cylinder number (0-indexed)
- $H$ = head number (0-indexed)
- $S$ = sector number (1-indexed, hence the $-1$)
- $H_{max}$ = heads per cylinder
- $S_{max}$ = sectors per track

### CHS Limits

CHS addressing uses 10/8/6 bits:

$$max_{cylinders} = 2^{10} = 1024$$
$$max_{heads} = 2^{8} = 256$$
$$max_{sectors} = 2^{6} = 63$$

$$max_{LBA} = 1024 \times 256 \times 63 = 16,515,072 \text{ sectors}$$

$$max_{bytes} = 16515072 \times 512 = 8,455,716,864 \approx 7.875 \text{ GB}$$

This is the **8 GB CHS barrier**. GRUB's stage 1 must be within this range on legacy BIOS.

### LBA Addressing (Modern)

48-bit LBA:

$$max_{LBA48} = 2^{48} = 281,474,976,710,656 \text{ sectors}$$

$$max_{bytes} = 2^{48} \times 512 = 128 \text{ PB (petabytes)}$$

---

## 3. MBR Partition Table Layout

### MBR Structure (512 bytes)

$$MBR = \underbrace{bootstrap}_{446} + \underbrace{partition\_table}_{64} + \underbrace{signature}_{2}$$

Each partition entry (16 bytes):

| Offset | Size | Field |
|:---|:---:|:---|
| 0 | 1 | Boot indicator (0x80 = active) |
| 1 | 3 | CHS of first sector |
| 4 | 1 | Partition type |
| 5 | 3 | CHS of last sector |
| 8 | 4 | LBA of first sector |
| 12 | 4 | Number of sectors |

### Maximum Partition Size (MBR)

$$max\_sectors = 2^{32} = 4,294,967,296$$

$$max\_size = 2^{32} \times 512 = 2 \text{ TB}$$

This is the **2 TB MBR barrier**.

### GPT vs MBR Capacity

| Scheme | Max Partitions | Max Disk Size | Max Partition Size |
|:---|:---:|:---:|:---:|
| MBR | 4 primary | 2 TB | 2 TB |
| GPT | 128 (default) | 9.4 ZB ($2^{64} \times 512$) | 9.4 ZB |

---

## 4. Kernel Loading — Compression and Memory

### Compressed Kernel Size

$$compression\_ratio = \frac{uncompressed}{compressed}$$

| Algorithm | Ratio | Decompression Speed | Boot Impact |
|:---|:---:|:---:|:---:|
| gzip | 3.5-4x | 250 MB/s | Baseline |
| xz/lzma | 4.5-5.5x | 80 MB/s | Slower decompress |
| lz4 | 2.5-3x | 1.5 GB/s | Fastest |
| zstd | 3.5-4.5x | 500 MB/s | Good balance |

### Kernel Load Time

$$T_{load} = \frac{compressed\_size}{disk\_bandwidth} + \frac{compressed\_size \times ratio}{decompress\_speed}$$

**Example:** 8 MB compressed kernel (gzip, ratio 4x), HDD at 100 MB/s:

$$T_{load} = \frac{8}{100} + \frac{32}{250} = 80ms + 128ms = 208ms$$

With NVMe (3 GB/s):

$$T_{load} = \frac{8}{3000} + \frac{32}{250} = 2.7ms + 128ms = 131ms$$

Decompression dominates on fast storage — disk speed barely matters.

### initrd Loading

$$T_{initrd} = \frac{initrd\_size}{disk\_bandwidth}$$

Typical initrd: 20-80 MB. On HDD: 200-800 ms. On NVMe: 7-27 ms.

---

## 5. GRUB Menu and Timeout

### Timeout Model

$$T_{wait} = \min(timeout, T_{user\_input})$$

If `GRUB_TIMEOUT=5`:

$$T_{wait} = \begin{cases} T_{keypress} & \text{if key pressed before 5s} \\ 5s & \text{otherwise (auto-boot default)} \end{cases}$$

### Hidden Timeout

`GRUB_TIMEOUT_STYLE=hidden`: Menu hidden, timeout still active. Pressing Shift reveals menu:

$$T_{hidden} = \begin{cases} T_{keypress} + T_{menu\_interaction} & \text{if Shift detected} \\ 0s & \text{otherwise (no menu at all)} \end{cases}$$

### Countdown Precision

GRUB uses firmware timer (typically 18.2 Hz on BIOS, or UEFI timer services):

$$resolution = \frac{1}{timer\_frequency}$$

BIOS: $1/18.2 \approx 55ms$. UEFI: typically $100\mu s$.

---

## 6. Filesystem Reading — GRUB's Mini-Drivers

### Filesystem Support Cost

GRUB includes mini filesystem drivers. Read performance:

$$T_{file\_read} = T_{inode\_lookup} + \lceil \frac{file\_size}{block\_size} \rceil \times T_{block\_read}$$

### Block Mapping (ext4)

For ext4 with extents:

$$blocks = \lceil \frac{file\_size}{block\_size} \rceil$$

$$extent\_lookups = O(\log_b(blocks)) \text{ where } b \approx 340 \text{ (B-tree branching)}$$

A 50 MB kernel file with 4 KB blocks:

$$blocks = \frac{50 \times 10^6}{4096} = 12,207$$

$$extent\_lookups = \log_{340}(12207) \approx 1.6 \text{ (2 tree levels)}$$

### GRUB Read Limitations

GRUB has no write capability — it's a **read-only** filesystem client. This simplifies the driver but means:

- No journal replay (can't recover from unclean shutdown in GRUB)
- No quota/ACL checks
- No encryption (unless specific module loaded)

---

## 7. UEFI Boot — EFI System Partition

### ESP Layout

$$ESP\_size \geq \sum bootloader\_images + kernel\_images + buffer$$

| Component | Typical Size |
|:---|:---:|
| grubx64.efi | 1-3 MB |
| Kernel (each) | 8-12 MB |
| initrd (each) | 20-80 MB |
| Firmware updates | 10-50 MB |

### Recommended ESP Sizing

$$ESP = n_{kernels} \times (size_{kernel} + size_{initrd}) + size_{grub} + margin$$

For 3 kernel versions with 10 MB kernels and 50 MB initrds:

$$ESP = 3 \times (10 + 50) + 3 + 100 = 283 \text{ MB}$$

Common recommendation: 512 MB ESP (allows growth).

### Secure Boot Chain

$$trust\_chain: firmware \to shim.efi \to grubx64.efi \to vmlinuz$$

Each stage verifies the signature of the next:

$$verify(image) = signature\_check(image, trusted\_key)$$

If any verification fails: boot halts.

---

## 8. Summary of GRUB Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| CHS to LBA | $(C \times H_{max} + H) \times S_{max} + S - 1$ | Address translation |
| CHS limit | $1024 \times 256 \times 63 \times 512 = 7.875$ GB | Capacity barrier |
| MBR partition limit | $2^{32} \times 512 = 2$ TB | Capacity barrier |
| Kernel load time | $size/bandwidth + uncompressed/decompress\_speed$ | I/O + CPU |
| Compression ratio | $uncompressed / compressed$ | Storage |
| ESP sizing | $n \times (kernel + initrd) + margin$ | Capacity planning |
| Boot timing | $\sum stage\_times + timeout$ | Sequential phases |

## Prerequisites

- disk geometry (CHS/LBA), partition tables (MBR/GPT), UEFI/BIOS firmware, filesystem basics, kernel boot process

---

*GRUB is the bridge between firmware and operating system — a miniature OS that reads filesystems, decompresses kernels, and manages a boot menu, all from a 446-byte bootstrap in sector zero. Every byte of that bootstrap is precious real estate.*
