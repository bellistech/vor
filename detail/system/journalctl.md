# The Mathematics of journalctl — Structured Logging, Storage Models & Query Performance

> *journalctl is a database query engine for system logs. The journal uses a structured binary format with hash-table indexing, giving it O(1) field lookups and O(log N) time-based seeks — but at a storage cost that demands careful capacity planning.*

---

## 1. Journal Storage Architecture

### Binary Format vs Text

The journal stores entries in a binary format with these components:

| Component | Size per Entry | Purpose |
|:---|:---:|:---|
| Entry header | 64 bytes | Timestamp, boot ID, sequence |
| Field headers | 16 bytes each | Key-value metadata |
| Field data | Variable | MESSAGE, UNIT, PID, etc. |
| Hash table entry | 8 bytes | Lookup index |
| **Typical total** | **200-500 bytes** | Per log line |

### Comparison to Text Logs

$$compression\_ratio = \frac{text\_log\_size}{journal\_size}$$

Journal is typically **1.5-3x smaller** than equivalent text (binary encoding + implicit compression).

Text syslog line: `~150 bytes` average. Journal entry: `~300 bytes` but includes ~10 structured fields that text discards.

### Storage Sizing Formula

$$storage = entries\_per\_day \times avg\_entry\_size \times retention\_days$$

**Example:** Web server producing 50,000 log lines/day:

$$storage = 50000 \times 400B \times 30 = 600 \text{ MB for 30-day retention}$$

---

## 2. Storage Limits — The Budget Model

### Configuration Parameters

| Parameter | Default | Formula |
|:---|:---:|:---|
| SystemMaxUse | min(10% of fs, 4 GB) | $\min(0.10 \times fs\_size, 4G)$ |
| SystemKeepFree | min(15% of fs, 4 GB) | $\min(0.15 \times fs\_size, 4G)$ |
| SystemMaxFileSize | SystemMaxUse / 8 | $max\_use / 8$ |
| SystemMaxFiles | 100 | Hard limit on journal files |

### Effective Limit

$$effective\_max = \min(SystemMaxUse,\ fs\_size - SystemKeepFree)$$

**Example:** 100 GB filesystem:

$$SystemMaxUse = \min(10G, 4G) = 4G$$
$$SystemKeepFree = \min(15G, 4G) = 4G$$
$$effective = \min(4G, 100G - 4G) = 4G$$

### Retention Calculation

$$retention\_days = \frac{effective\_max}{daily\_log\_volume}$$

| Daily Volume | 4 GB Limit | 500 MB Limit |
|:---:|:---:|:---:|
| 10 MB | 400 days | 50 days |
| 100 MB | 40 days | 5 days |
| 500 MB | 8 days | 1 day |
| 1 GB | 4 days | 12 hours |

---

## 3. Query Performance — Index Structures

### Hash Table Lookups

Journal fields are indexed by a **hash table** with chaining:

$$T_{lookup}(field = value) = O(1 + \alpha)$$

Where $\alpha = n/m$ is the load factor (entries/buckets).

### Time-Based Seek

Journal files store entries in chronological order with a **monotonic timestamp index**:

$$T_{seek}(timestamp) = O(\log N) \text{ via binary search on file headers}$$

### Query Type Performance

| Query | Complexity | Example |
|:---|:---:|:---|
| `--since "1 hour ago"` | $O(\log N)$ | Binary search on timestamp |
| `-u nginx.service` | $O(k)$ | Hash lookup, $k$ = matching entries |
| `-p err` | $O(N)$ | Scan all entries, check priority |
| `--grep "pattern"` | $O(N \times L)$ | Full scan with regex |
| `-b -1` | $O(\log N)$ | Boot ID index lookup |

### Combined Query Cost

$$T_{combined} = T_{index\_seek} + T_{scan\_candidates} \times P(match)$$

**Example:** `journalctl -u nginx --since "1h" --grep "502"`:

1. Seek to 1 hour ago: $O(\log N)$
2. Filter by unit (hash index): reduces to $k$ entries
3. Grep through $k$ entries: $O(k \times L)$

---

## 4. Priority Levels — Severity Filtering

### Syslog Priority Mapping

$$priority \in \{0, 1, 2, 3, 4, 5, 6, 7\}$$

| Value | Name | Meaning | Typical Volume |
|:---:|:---|:---|:---|
| 0 | emerg | System unusable | < 0.001% |
| 1 | alert | Immediate action | < 0.01% |
| 2 | crit | Critical | < 0.1% |
| 3 | err | Error | 1-5% |
| 4 | warning | Warning | 5-10% |
| 5 | notice | Significant | 10-20% |
| 6 | info | Informational | 50-70% |
| 7 | debug | Debug | 10-30% |

### Priority-Based Filtering Efficiency

`journalctl -p err` shows priorities 0-3:

$$\%scanned = \frac{N_{0} + N_{1} + N_{2} + N_{3}}{N_{total}} \approx 1-5\%$$

This dramatically reduces output but still requires scanning (priority is checked per-entry unless indexed).

### Volume Reduction by Priority

$$volume(p) = N_{total} \times \sum_{i=0}^{p} fraction(i)$$

$$reduction = 1 - \frac{volume(p)}{volume(7)}$$

Setting `MaxLevelStore=warning` (4) in `journald.conf`:

$$storage\_saved = fraction(notice) + fraction(info) + fraction(debug) \approx 70-90\%$$

---

## 5. Rate Limiting

### journald Rate Limiting

Default: `RateLimitIntervalSec=30s`, `RateLimitBurst=10000`

$$max\_rate = \frac{RateLimitBurst}{RateLimitIntervalSec} = \frac{10000}{30} = 333 \text{ messages/s per service}$$

When exceeded, journald suppresses messages and logs:

```
Suppressed N messages from unit.service
```

### Message Loss Calculation

$$suppressed = max(0, actual\_rate - max\_rate) \times interval$$

**Example:** Service logging at 1000 msg/s, limit 333 msg/s:

$$suppressed = (1000 - 333) \times 30 = 20,010 \text{ messages per 30s interval}$$

$$\%lost = \frac{667}{1000} = 66.7\%$$

---

## 6. Boot Log Segmentation

### Boot ID Index

Journal entries are tagged with `_BOOT_ID`. The boot index enables:

$$T_{boot\_select} = O(\log B) \text{ where } B = \text{number of boots}$$

### Boot History Analysis

$$uptime_i = boot\_time_{i+1} - boot\_time_i$$

$$\overline{uptime} = \frac{\sum uptime_i}{n_{boots} - 1}$$

$$availability = \frac{\sum uptime_i}{T_{total}} \times 100\%$$

**Example:** 5 boots in 30 days, total downtime 2 hours:

$$availability = \frac{30 \times 24 - 2}{30 \times 24} = \frac{718}{720} = 99.72\%$$

### Mean Time Between Failures

$$MTBF = \frac{total\_uptime}{n_{failures}}$$

$$MTTR = \frac{total\_downtime}{n_{failures}}$$

$$availability = \frac{MTBF}{MTBF + MTTR}$$

---

## 7. Disk I/O Impact

### Write Amplification

Journal writes are not simple appends. Each entry requires:

1. Write entry data: $size_{entry}$
2. Update hash index: $\approx 64$ bytes
3. Sync to disk (if `Storage=persistent`): `fdatasync()` cost

$$write\_amplification = \frac{size_{entry} + size_{index\_update}}{size_{entry}} \approx 1.1-1.3\times$$

### Sync Cost

With `SyncIntervalSec=5m` (default):

$$syncs\_per\_hour = \frac{3600}{300} = 12$$

$$sync\_cost = 12 \times T_{fdatasync} \approx 12 \times 5ms = 60ms/hour$$

Negligible on SSD. On HDD with 10ms sync: $120ms/hour$.

### Journal Vacuuming

`journalctl --vacuum-size=500M` deletes oldest files:

$$files\_deleted = N_{files} - \lceil \frac{target\_size}{avg\_file\_size} \rceil$$

$$space\_freed = current\_size - target\_size$$

---

## 8. Summary of journalctl Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Storage budget | $\min(0.10 \times fs, 4G)$ | Capacity |
| Retention | $max\_storage / daily\_volume$ | Duration |
| Hash lookup | $O(1 + \alpha)$ | Index performance |
| Time seek | $O(\log N)$ | Binary search |
| Rate limiting | $burst / interval$ | Throttling |
| Suppression | $(actual - max) \times interval$ | Message loss |
| Availability | $MTBF / (MTBF + MTTR)$ | Reliability |
| Write amplification | $(entry + index) / entry$ | I/O overhead |

---

*journalctl is a time-series database disguised as a log viewer. Its binary format trades human readability for queryability — and that tradeoff, properly understood, makes it the most powerful log analysis tool on a systemd system.*
