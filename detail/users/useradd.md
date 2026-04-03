# The Mathematics of useradd — UID Allocation, Home Directory Provisioning & Security Defaults

> *useradd is a transaction: allocate a UID, create groups, provision a home directory, and copy skeleton files — all atomically. The mathematics involve range allocation, permission bits, and storage provisioning.*

---

## 1. UID Allocation — Range Partitioning

### UID Ranges (Defined in /etc/login.defs)

$$UID\_space = [0, 2^{32} - 2] = [0, 4294967294]$$

| Range | Purpose | Default (RHEL/Debian) |
|:---|:---|:---|
| 0 | root | Fixed |
| 1-999 | System accounts | `SYS_UID_MIN` - `SYS_UID_MAX` |
| 1000-60000 | Regular users | `UID_MIN` - `UID_MAX` |
| 60001-65533 | Reserved/dynamic | System services |
| 65534 | nobody | Fixed by convention |
| 65535+ | Available (extended) | Rarely used |

### Next UID Algorithm

useradd selects the next available UID:

$$uid = \min\{n : n \geq UID\_MIN \land n \notin used\_uids\}$$

Where $used\_uids$ is read from `/etc/passwd`.

### UID Exhaustion

$$available\_uids = |[UID\_MIN, UID\_MAX] \setminus used\_uids|$$

$$remaining = UID\_MAX - UID\_MIN + 1 - |used\_uids|$$

Default range: $60000 - 1000 + 1 = 59001$ possible UIDs.

### UID Uniqueness Guarantee

$$\forall u_1, u_2 \in users : u_1 \neq u_2 \implies uid(u_1) \neq uid(u_2)$$

This invariant can be violated with `-o` (non-unique) flag — generally a security risk.

---

## 2. Group Allocation — UPG Model

### User Private Group (UPG)

By default, useradd creates a group with the same name as the user:

$$gid_{new} = \min\{g : g \geq GID\_MIN \land g \notin used\_gids\}$$

$$group_{name} = user_{name}$$

### Group Membership Model

$$groups(user) = \{primary\_group\} \cup supplementary\_groups$$

$$|groups(user)| \leq NGROUPS\_MAX = 65536 \text{ (Linux)}$$

### Permission Implications of UPG

With UPG, default umask 002 is safe:

$$permissions_{new\_file} = 0666 \& \sim 0002 = 0664 \text{ (rw-rw-r--)}$$

Since each user has a private group, group-writable files are only writable by the user.

Without UPG (shared group), umask 022 is needed:

$$permissions_{new\_file} = 0666 \& \sim 0022 = 0644 \text{ (rw-r--r--)}$$

---

## 3. Home Directory Provisioning

### Skeleton Copy

$$home = base\_dir / username$$

Default: $base\_dir = /home$.

Files copied from `/etc/skel/`:

$$\forall f \in skel/: copy(f, home/f)$$

### Skeleton Size

$$T_{create} = T_{mkdir} + |skel\_files| \times T_{copy} + T_{chown\_recursive}$$

| Component | Typical Cost |
|:---|:---:|
| mkdir | 0.1 ms |
| Copy skel (~10 files) | 1-5 ms |
| chown -R | 0.5-2 ms |
| **Total** | **2-8 ms** |

### Storage Per User

$$storage_{home} = storage_{skel} + storage_{user\_data\_over\_time}$$

Typical skel: 5-20 KB. User data grows over time.

### Quota (Optional)

$$quota(user) = \begin{cases} blocks_{soft}, blocks_{hard} & \text{disk space limits} \\ inodes_{soft}, inodes_{hard} & \text{file count limits} \end{cases}$$

---

## 4. Password Handling — Shadow Entry

### /etc/shadow Fields

useradd creates a shadow entry:

$$shadow = (username, hash, lastchange, min, max, warn, inactive, expire)$$

### Password Aging Model

$$T_{valid} = [lastchange + min\_days,\ lastchange + max\_days]$$

$$T_{warn} = lastchange + max\_days - warn\_days$$

$$T_{lock} = lastchange + max\_days + inactive\_days$$

### Worked Example

`useradd -e 2027-01-01 -f 30` with `PASS_MAX_DAYS=90`, `PASS_MIN_DAYS=7`, `PASS_WARN_AGE=14`:

$$password\_must\_change\_by = creation + 90 \text{ days}$$
$$warning\_starts = creation + 76 \text{ days}$$
$$account\_locked\_at = creation + 120 \text{ days}$$
$$account\_expires = 2027\text{-}01\text{-}01 \text{ (regardless of password)}$$

### Days Since Epoch

Shadow dates are stored as days since 1970-01-01:

$$shadow\_date = \lfloor \frac{unix\_timestamp}{86400} \rfloor$$

2026-04-03: $shadow\_date = \lfloor 1775020800 / 86400 \rfloor = 20544$

---

## 5. Batch User Creation — Scaling

### Performance Model

$$T_{batch} = N \times (T_{passwd\_update} + T_{shadow\_update} + T_{group\_update} + T_{home\_create})$$

| N Users | Time (HDD) | Time (SSD) |
|:---:|:---:|:---:|
| 1 | 10 ms | 5 ms |
| 100 | 1 s | 0.5 s |
| 1000 | 10 s | 5 s |
| 10000 | 100 s | 50 s |

### Lock Contention

useradd locks `/etc/passwd` and `/etc/shadow` during modification:

$$T_{lock} = T_{acquire} + T_{modify} + T_{release}$$

Parallel useradd is serialized by file locks — no speedup from parallelism.

### File Growth

Each passwd entry: ~60 bytes. Shadow entry: ~80 bytes. Group entry: ~30 bytes.

$$file\_growth = N \times (60 + 80 + 30) = N \times 170 \text{ bytes}$$

10,000 users: $\approx 1.7 \text{ MB}$ across all files.

### nsswitch Lookup Impact

With 10,000 entries in `/etc/passwd`:

$$T_{lookup} = O(N) \text{ for flat file}$$

$$T_{lookup\_ldap} = O(\log N) + RTT \text{ for directory service}$$

At 10,000+ users, consider switching to LDAP/SSSD for $O(\log N)$ lookups.

---

## 6. System Account Creation (-r flag)

### Differences from Regular Users

| Feature | Regular (-m) | System (-r) |
|:---|:---:|:---:|
| UID range | 1000-60000 | 1-999 |
| Home directory | Created | Not created |
| Login shell | /bin/bash | /sbin/nologin |
| Password aging | Enabled | Disabled |
| Group | UPG created | System group |

### Why System UIDs < 1000?

Convention ensures:
1. System accounts are visually identifiable
2. `UID_MIN` filters in `passwd` enumerate only human users
3. Display managers show only UIDs $\geq 1000$

$$visible\_users = \{u : uid(u) \geq UID\_MIN\}$$

---

## 7. Security Defaults — login.defs

### Key Parameters

| Parameter | Default | Formula/Meaning |
|:---|:---:|:---|
| UMASK | 022 | $0777 \& \sim UMASK$ for dirs |
| PASS_MAX_DAYS | 99999 | $\approx 274$ years (effectively none) |
| PASS_MIN_DAYS | 0 | Can change immediately |
| PASS_MIN_LEN | 5 | Minimum password length |
| ENCRYPT_METHOD | SHA512 | Hash algorithm |
| SHA_CRYPT_ROUNDS | 5000 | $T_{hash} \approx rounds \times T_{SHA512}$ |

### Hash Cost Calculation

$$T_{hash} = rounds \times T_{SHA512\_block} \approx 5000 \times 0.5\mu s = 2.5ms$$

This makes brute force expensive:

$$T_{brute\_force} = |password\_space| \times T_{hash}$$

For 8-char alphanumeric: $|space| = 62^8 = 2.18 \times 10^{14}$

$$T_{crack} = 2.18 \times 10^{14} \times 2.5ms = 1.73 \times 10^{4} \text{ years (single core)}$$

---

## 8. Summary of useradd Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| UID allocation | $\min\{n \geq UID\_MIN : n \notin used\}$ | Sequential search |
| UID capacity | $UID\_MAX - UID\_MIN + 1$ | Range size |
| Home creation | $T_{mkdir} + \|skel\| \times T_{copy}$ | Provisioning cost |
| Shadow dates | $\lfloor epoch / 86400 \rfloor$ | Day conversion |
| Password aging | $lastchange + max\_days$ | Date arithmetic |
| File growth | $N \times 170$ bytes | Linear |
| Hash cost | $rounds \times T_{SHA512}$ | Computational hardening |

---

*useradd is a controlled allocation: take the next UID from a range, create a group, copy a skeleton, and set security defaults. Every field in passwd and shadow is a parameter in the security equation, and the defaults are the sysadmin's first line of defense.*
