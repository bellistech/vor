# The Mathematics of logrotate — Rotation Algebra, Storage Planning & Compression Models

> *logrotate is a storage management algorithm: it maintains a sliding window of log files with bounded total size. The rotation scheme, compression ratios, and retention policies are all computable, and the math prevents disk-full emergencies.*

---

## 1. Rotation as Sliding Window

### The File Chain Model

logrotate maintains a numbered chain of files:

```
app.log → app.log.1 → app.log.2 → ... → app.log.N → (deleted)
```

$$files\_kept = rotate + 1 \text{ (current + N rotated)}$$

### Rotation Arithmetic

At each rotation:

$$\forall i \in [N, 1]: file.i \leftarrow file.(i-1)$$
$$file.1 \leftarrow file$$
$$file \leftarrow \text{new empty file}$$

If $i > rotate$: file is deleted.

### Total Storage

$$storage = size(current) + \sum_{i=1}^{rotate} size(file.i)$$

Without compression:

$$storage_{max} = (rotate + 1) \times max\_file\_size$$

**Example:** `rotate 7`, `size 100M`:

$$storage_{max} = 8 \times 100MB = 800MB$$

---

## 2. Compression — Space Savings

### Compression Ratios for Log Files

Log files compress exceptionally well due to repetitive structure:

| Algorithm | Typical Ratio | Compression Speed | logrotate Option |
|:---|:---:|:---:|:---|
| gzip (default) | 8-15x | 50 MB/s | `compress` |
| bzip2 | 10-20x | 10 MB/s | `compresscmd /usr/bin/bzip2` |
| xz | 12-25x | 5 MB/s | `compresscmd /usr/bin/xz` |
| zstd | 8-18x | 300 MB/s | `compresscmd /usr/bin/zstd` |

### Storage with Compression

$$storage = size(current) + size(file.1) + \sum_{i=2}^{rotate} \frac{size(file.i)}{ratio}$$

With `delaycompress` (compress starting from .2):

$$storage = size_{current} + size_{prev} + (rotate - 1) \times \frac{avg\_size}{ratio}$$

**Example:** 100 MB logs, `rotate 7`, gzip ratio 10x:

$$storage = 100 + 100 + 6 \times \frac{100}{10} = 100 + 100 + 60 = 260 MB$$

vs uncompressed: 800 MB. **Savings: 67.5%**

### delaycompress Rationale

The most recent rotated file (`.1`) is kept uncompressed for quick access by log analyzers. Cost:

$$extra\_storage = size_{file.1} - \frac{size_{file.1}}{ratio} = size \times (1 - \frac{1}{ratio})$$

At 100 MB with 10x ratio: $extra = 100 \times 0.9 = 90 MB$.

---

## 3. Rotation Triggers — Size, Time, and Combined

### Size-Based Rotation

```
size 100M
```

$$rotate\_when: size(current) \geq threshold$$

### Time-Based Rotation

| Directive | Period | Rotations/Year |
|:---|:---:|:---:|
| `hourly` | 1 hour | 8,760 |
| `daily` | 1 day | 365 |
| `weekly` | 1 week | 52 |
| `monthly` | 1 month | 12 |
| `yearly` | 1 year | 1 |

### Combined: maxsize and minsize

| Directive | Meaning | Formula |
|:---|:---|:---|
| `maxsize 100M` | Rotate if size > 100M OR time elapsed | $size > max \lor time > period$ |
| `minsize 10M` | Rotate only if size > 10M AND time elapsed | $size > min \land time > period$ |

### Growth Rate and Rotation Frequency

$$rotation\_freq = \max\left(\frac{1}{period},\ \frac{growth\_rate}{size\_threshold}\right)$$

**Example:** Log grows at 50 MB/day, daily rotation with `size 100M`:

$$time\_trigger: \text{every day}$$
$$size\_trigger: \frac{50}{100} = 0.5/day \text{ (never triggers before daily)}$$

$$rotation\_freq = 1/day$$

But if growth is 200 MB/day with `maxsize 100M`:

$$size\_triggers = \lfloor 200/100 \rfloor = 2 \text{ rotations/day (in addition to daily)}$$

---

## 4. Retention Calculation — How Long Logs Last

### Time-Based Retention

$$retention\_days = rotate \times period\_days$$

| Config | Retention |
|:---|:---:|
| `daily`, `rotate 7` | 7 days |
| `daily`, `rotate 30` | 30 days |
| `weekly`, `rotate 4` | 28 days |
| `monthly`, `rotate 12` | 365 days |
| `weekly`, `rotate 52` | 364 days |

### Space-Based Retention

With `maxsize` and finite disk:

$$retention = \frac{available\_space}{growth\_rate}$$

**Example:** 10 GB allocated, 500 MB/day growth:

$$retention = \frac{10000}{500} = 20 \text{ days}$$

This should align with `rotate` count:

$$rotate \geq \frac{retention}{period} = \frac{20}{1} = 20$$

### Compliance Requirements

| Standard | Min Retention | Recommended Config |
|:---|:---:|:---|
| PCI-DSS | 1 year | `monthly rotate 12` or `daily rotate 365` |
| HIPAA | 6 years | `monthly rotate 72` |
| SOX | 7 years | `monthly rotate 84` |
| GDPR | As short as possible | `daily rotate 30` (or less) |

---

## 5. Disk Planning — Total Log Storage Budget

### Budget Formula

$$total\_log\_storage = \sum_{logfile} (rotate_i + 1) \times \frac{avg\_size_i}{ratio_i}$$

### Worked Example

System with three services:

| Service | Growth/Day | Rotation | Rotate | Ratio | Storage |
|:---|:---:|:---|:---:|:---:|:---:|
| nginx | 500 MB | daily | 14 | 10x | $500 + 14 \times 50 = 1200$ MB |
| app | 200 MB | daily | 30 | 10x | $200 + 30 \times 20 = 800$ MB |
| syslog | 50 MB | weekly | 4 | 10x | $350 + 4 \times 35 = 490$ MB |
| **Total** | | | | | **2490 MB** |

**Budget:** Allocate 5 GB for logs (2x safety margin).

### Monitoring Formula

$$days\_until\_full = \frac{free\_space - total\_log\_storage}{daily\_growth\_rate}$$

If current disk usage is growing beyond projections, increase rotation frequency or reduce retention.

---

## 6. copytruncate vs create — Data Loss Risk

### create (Default)

```
rename current → .1
create new empty file
signal application to reopen
```

$$data\_loss\_window = T_{rename} + T_{create} + T_{signal} \approx 0$$

No data loss if application handles SIGHUP correctly.

### copytruncate

```
copy current → .1
truncate current to 0
```

$$data\_loss\_window = T_{copy}$$

$$data\_lost = growth\_rate \times T_{copy}$$

**Example:** Log growing at 10 MB/s, file size 100 MB, copy at 500 MB/s:

$$T_{copy} = \frac{100MB}{500MB/s} = 0.2s$$

$$data\_lost = 10 \times 0.2 = 2 MB$$

### When to Use copytruncate

$$use\_copytruncate \iff \neg \text{app\_supports\_SIGHUP} \land acceptable\_loss > growth \times T_{copy}$$

Applications that can't reopen log files (legacy apps, some Java frameworks) require copytruncate.

---

## 7. postrotate/prerotate — Hook Timing

### Script Execution Model

$$T_{rotation} = T_{prerotate} + T_{rotate} + T_{compress} + T_{postrotate}$$

### Common postrotate Costs

| Action | Command | Cost |
|:---|:---|:---:|
| Reload nginx | `nginx -s reload` | 50-200 ms |
| HUP syslog | `kill -HUP $(cat pid)` | 1-5 ms |
| Restart service | `systemctl restart svc` | 500-5000 ms |
| Nothing | (none) | 0 ms |

### sharedscripts Optimization

Without `sharedscripts`: postrotate runs once per file.

$$T_{total} = N_{files} \times T_{postrotate}$$

With `sharedscripts`: postrotate runs once for all files:

$$T_{total} = 1 \times T_{postrotate}$$

For 10 nginx log files with 100 ms reload: $1000ms$ vs $100ms$.

---

## 8. Summary of logrotate Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Files kept | $rotate + 1$ | Count |
| Max storage (plain) | $(rotate + 1) \times max\_size$ | Capacity |
| Storage (compressed) | $size + size + (rotate-1) \times size/ratio$ | With delaycompress |
| Retention | $rotate \times period$ | Duration |
| Days until full | $(free - log\_budget) / growth$ | Projection |
| copytruncate loss | $growth\_rate \times T_{copy}$ | Data risk |
| Rotation frequency | $\max(1/period, growth/threshold)$ | Trigger rate |
| Compression savings | $1 - 1/ratio$ | Percentage |

## Prerequisites

- capacity planning, compression ratios, filesystem management, cron scheduling, log management

---

*logrotate is capacity planning in a config file. Every directive — rotate count, size threshold, compression — is a parameter in the storage equation, and getting it wrong means either running out of disk at 3 AM or losing critical audit logs.*
