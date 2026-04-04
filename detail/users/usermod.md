# The Mathematics of usermod — Account Modification, Group Arithmetic & Migration

> *usermod is atomic account surgery — changing UIDs, groups, shells, and expiration dates on a live system. Every modification has cascading effects: file ownership, group membership sets, and permission recalculations.*

---

## 1. Group Membership — Set Operations

### The Group Model

A user's group membership is a set:

$$G(user) = \{primary\} \cup \{supplementary_1, supplementary_2, ..., supplementary_n\}$$

### -G (Replace Supplementary Groups)

$$G_{new} = \{primary\} \cup specified\_groups$$

**Warning:** This **replaces** all supplementary groups:

$$lost\_groups = G_{old} \setminus (G_{new} \cup \{primary\})$$

**Example:** User in groups {primary, docker, sudo, www-data}. Run `usermod -G docker user`:

$$G_{new} = \{primary, docker\}$$
$$lost = \{sudo, www\text{-}data\}$$

### -aG (Append to Groups)

$$G_{new} = G_{old} \cup specified\_groups$$

No groups are lost. This is almost always what you want:

$$|G_{new}| = |G_{old}| + |specified \setminus G_{old}|$$

### Group Count Limit

$$|G(user)| \leq NGROUPS\_MAX = 65536 \text{ (Linux kernel)}$$

Practical limit: NFS limits to 16 groups (AUTH_SYS). Some LDAP configs: 1024.

$$effective\_limit = \min(NGROUPS\_MAX, protocol\_limits)$$

---

## 2. UID Change — Cascading Ownership

### The -u Flag

$$uid_{old} \to uid_{new}$$

### Automatic Ownership Update (-u with home)

usermod updates ownership of the home directory:

$$\forall f \in home/: \text{if } uid(f) = uid_{old} \to uid(f) = uid_{new}$$

$$T_{chown} = N_{files} \times T_{chown\_per\_file}$$

For 100,000 files: $T_{chown} \approx 100000 \times 10\mu s = 1s$ (SSD).

### Files Outside Home

Files outside home are **NOT automatically updated**:

$$orphaned = \{f : uid(f) = uid_{old} \land f \notin home/\}$$

Find them with: `find / -uid $old_uid`

$$T_{find} = N_{total\_files} \times T_{stat}$$

For 1 million files: $T_{find} \approx 10s$ (SSD).

### GID Change (-g)

Same cascading logic:

$$\forall f \in home/: \text{if } gid(f) = gid_{old} \to gid(f) = gid_{new}$$

---

## 3. Home Directory Move — -m -d

### The Move Operation

`usermod -m -d /new/home user`:

$$operation = rename(old\_home, new\_home) \lor copy + delete$$

### Rename vs Copy

$$T_{rename} = O(1) \text{ (same filesystem — metadata only)}$$

$$T_{copy} = \frac{home\_size}{bandwidth} + N_{files} \times T_{create} \text{ (cross filesystem)}$$

| Home Size | Same FS | Cross FS (SSD) | Cross FS (HDD) |
|:---:|:---:|:---:|:---:|
| 100 MB | < 1 ms | 200 ms | 1 s |
| 1 GB | < 1 ms | 2 s | 10 s |
| 100 GB | < 1 ms | 200 s | 1000 s |

### Symlink Breakage

$$broken\_links = \{l : target(l) \in old\_home/ \text{ and } l \notin old\_home/\}$$

Symlinks from outside pointing into the old home directory will break.

---

## 4. Account Locking and Expiry

### -L (Lock Account)

Prepends `!` to the password hash in `/etc/shadow`:

$$hash_{locked} = \text{!} + hash_{original}$$

$$auth(user) = false \quad \forall \text{ password attempts}$$

But SSH key authentication still works (PAM `pam_unix` only).

### -U (Unlock Account)

$$hash_{unlocked} = hash_{original} = hash_{locked}[1:]$$

### -e (Expiration Date)

$$account\_active = (now \leq expire\_date) \lor (expire\_date = -1)$$

$$expire\_epoch = \lfloor \frac{date}{86400} \rfloor$$

### -f (Inactive Days)

$$account\_locked = now > (password\_expire + inactive\_days)$$

$$T_{lock} = T_{password\_set} + PASS\_MAX\_DAYS + inactive\_days$$

**Example:** Password set 2026-01-01, PASS_MAX_DAYS=90, inactive=30:

$$password\_expires = 2026\text{-}04\text{-}01$$
$$account\_locks = 2026\text{-}05\text{-}01$$

---

## 5. Shell Change — -s

### Valid Shells

$$valid\_shells = \{s : s \in /etc/shells\}$$

### Security Implications

| Shell | Purpose | Login? |
|:---|:---|:---:|
| /bin/bash | Full interactive | Yes |
| /bin/sh | POSIX compatible | Yes |
| /usr/sbin/nologin | Block login | No |
| /bin/false | Block login | No |
| /usr/bin/git-shell | Git-only access | Limited |

### Login Denial

$$login\_allowed = shell(user) \notin \{/usr/sbin/nologin, /bin/false\}$$

Setting nologin shell:

$$authentication\_works = true \text{ (password verified)}$$
$$session\_created = false \text{ (shell exits immediately)}$$

This blocks SSH, console, su, but NOT sudo (which doesn't invoke login shell).

---

## 6. Rename Operations — -l (Login Name)

### Impact Analysis

`usermod -l newname oldname` changes:

| Updated | Not Updated |
|:---|:---|
| /etc/passwd entry | Home directory name |
| /etc/shadow entry | File ownership |
| /etc/group entries | crontabs |
| Primary group (if matching) | Application configs |
| | sudoers rules |

### Cascading Updates Needed

$$manual\_updates = crontab + sudoers + app\_configs + \text{any hardcoded references}$$

$$T_{total\_rename} = T_{usermod} + T_{home\_rename} + T_{manual\_updates}$$

### Complete User Rename Procedure

1. `usermod -l newname oldname` — rename account
2. `usermod -d /home/newname -m newname` — rename and move home
3. `groupmod -n newname oldname` — rename primary group
4. Update sudoers, crontabs, application configs manually

---

## 7. Concurrent Modification Safety

### File Locking

usermod acquires locks on:

$$locks = \{/etc/passwd, /etc/shadow, /etc/group, /etc/gshadow\}$$

$$T_{locked} = T_{read} + T_{modify} + T_{write}$$

### Concurrent Access Risk

$$risk = P(read\_during\_modify) \times P(inconsistent\_data)$$

With file locking: $risk \approx 0$ for usermod vs usermod.

Without locking (manual edits): $risk > 0$.

### NSS Cache Invalidation

After usermod, the Name Service Cache Daemon (nscd/sssd) may serve stale data:

$$T_{stale} = cache\_TTL \text{ (typically 10-600 seconds)}$$

Force refresh: `nscd -i passwd` or `sss_cache -U`.

---

## 8. Summary of usermod Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Group append | $G_{new} = G_{old} \cup specified$ | Set union |
| Group replace | $G_{new} = \{primary\} \cup specified$ | Set replacement |
| Ownership cascade | $N_{files} \times T_{chown}$ | Linear scan |
| Home move (same FS) | $O(1)$ — rename | Metadata only |
| Home move (cross FS) | $size / bandwidth$ | I/O bound |
| Account expiry | $now \leq expire\_date$ | Date comparison |
| Lock mechanism | `!` prefix on hash | String operation |
| Password lock date | $set + max\_days + inactive$ | Date arithmetic |

## Prerequisites

- set theory (group membership), file ownership, UID/GID mapping, /etc/passwd and /etc/shadow formats, filesystem operations

---

*usermod is the scalpel of user management — precise modifications to a running account. But every cut has consequences: changing a UID orphans files, changing groups revokes access, and moving a home breaks symlinks. Measure twice, usermod once.*
