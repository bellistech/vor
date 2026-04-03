# The Mathematics of groups — Permission Sets, Access Control & Membership Graphs

> *Unix groups are sets, and file permissions are functions over those sets. Every access decision is a set membership test: does the requesting user's group set intersect with the file's group? Understanding this algebra clarifies permission models.*

---

## 1. Group Membership as Sets

### The Membership Model

$$G(user) = \{g_1, g_2, ..., g_n\} \text{ where } g_1 = primary\_group$$

### Set Operations in Practice

| Operation | Command | Set Meaning |
|:---|:---|:---|
| List groups | `groups user` | $G(user)$ |
| Add to group | `usermod -aG g user` | $G' = G \cup \{g\}$ |
| Replace groups | `usermod -G g1,g2 user` | $G' = \{primary\} \cup \{g_1, g_2\}$ |
| Remove from group | `gpasswd -d user group` | $G' = G \setminus \{group\}$ |
| Common groups | — | $G(u_1) \cap G(u_2)$ |

### Group Membership Matrix

For $U$ users and $G$ groups, membership is a binary matrix:

$$M[u][g] = \begin{cases} 1 & \text{if } g \in G(u) \\ 0 & \text{otherwise} \end{cases}$$

| | docker | sudo | www-data | dev |
|:---|:---:|:---:|:---:|:---:|
| alice | 1 | 1 | 0 | 1 |
| bob | 1 | 0 | 1 | 1 |
| charlie | 0 | 0 | 1 | 0 |

### Cardinality Limits

$$|G(user)| \leq NGROUPS\_MAX = 65536 \text{ (Linux)}$$

$$|users(group)| \leq \infty \text{ (no kernel limit, limited by /etc/group line length)}$$

Practical limit for `/etc/group`: line length ~1024-4096 bytes. With 8-char usernames:

$$max\_members \approx \frac{4096}{9} \approx 455 \text{ per group line}$$

---

## 2. Permission Evaluation — The Access Decision

### DAC (Discretionary Access Control) Algorithm

For a process with $uid$, $gid$, and supplementary groups $G$, accessing a file:

$$access(file) = \begin{cases}
owner\_perms & \text{if } uid = file.uid \\
group\_perms & \text{if } gid = file.gid \lor file.gid \in G \\
other\_perms & \text{otherwise}
\end{cases}$$

### Permission Bits as Bitmask

$$permission = rwx = 4 \times r + 2 \times w + 1 \times x$$

$$mode = owner(3\ bits) + group(3\ bits) + other(3\ bits) = 9\ bits$$

### Effective Permission Calculation

$$effective = requested\_perms \& allowed\_perms$$

$$granted = (effective == requested\_perms)$$

**Example:** File mode 0750, user is in file's group, requests read:

$$group\_perms = 5 = r\_x \text{ (read + execute)}$$

$$read\_request = 4 = r\_\_$$

$$4\ \&\ 5 = 4 = r\_\_ \text{ → granted}$$

---

## 3. Group Inheritance — setgid and Default Groups

### setgid on Directories

When a directory has setgid bit:

$$gid(new\_file) = gid(parent\_directory) \text{ (not creator's primary group)}$$

### Collaboration Model

Without setgid: files get creator's primary group.

$$gid(file) = primary\_group(creator)$$

With setgid on project directory:

$$gid(file) = gid(project\_dir) \quad \forall \text{ files created in } project\_dir/$$

### umask Interaction

$$permissions(new\_file) = default\_mode \& \sim umask$$

| umask | File (from 0666) | Dir (from 0777) | Group Writable? |
|:---:|:---:|:---:|:---:|
| 022 | 0644 (rw-r--r--) | 0755 (rwxr-xr-x) | No |
| 002 | 0664 (rw-rw-r--) | 0775 (rwxrwxr-x) | Yes |
| 077 | 0600 (rw-------) | 0700 (rwx------) | No |

### UPG + umask 002 Model

$$safe\_with\_UPG \iff umask = 002 \land |users(primary\_group)| = 1$$

Group-writable is safe because the primary group has only one member.

---

## 4. ACL Extensions — Fine-Grained Group Access

### POSIX ACL Model

ACLs extend the 9-bit permission model:

$$ACL = \{(type, qualifier, perms)\}$$

Types: user, group, mask, other.

### Effective Permissions with ACL

$$effective(entry) = entry.perms \& mask$$

The mask limits all named user and group entries:

$$\forall named\_entry: effective = perms \& mask$$

### ACL Example

```
user::rwx       (owner)
user:bob:rw-     (named user)
group::r-x       (owning group)
group:dev:rwx    (named group)
mask::rwx        (maximum for named entries)
other::---       (everyone else)
```

Bob's effective: $rw\_ \& rwx = rw\_$. Dev group effective: $rwx \& rwx = rwx$.

If mask changed to `r-x`:

Bob's effective: $rw\_ \& r\_x = r\_\_$. Dev group: $rwx \& r\_x = r\_x$.

---

## 5. Group as a Graph — Membership Relationships

### Bipartite Graph Model

Users and groups form a **bipartite graph**:

$$B = (U, G, E) \text{ where } E = \{(u, g) : u \in members(g)\}$$

### Shared Access

Two users can access the same group-owned resources if:

$$shared\_access(u_1, u_2) = G(u_1) \cap G(u_2) \neq \emptyset$$

### Reachability

Users $u_1$ and $u_2$ are "connected" through group membership:

$$connected(u_1, u_2) = \exists g : u_1 \in members(g) \land u_2 \in members(g)$$

### Graph Metrics

$$degree(user) = |G(user)| \text{ (number of groups)}$$

$$degree(group) = |members(group)| \text{ (number of members)}$$

$$density = \frac{|E|}{|U| \times |G|}$$

For 100 users, 20 groups, 300 memberships: $density = 300 / 2000 = 15\%$.

---

## 6. Primary Group — Performance and Security

### Primary Group Selection

$$primary = gid \text{ in } /etc/passwd$$

$$supplementary = \{g : user \in members(g) \text{ in } /etc/group\} \setminus \{primary\}$$

### Performance Impact

Group membership is checked at process creation and cached:

$$T_{group\_lookup} = O(|supplementary|) \text{ per getgroups() call}$$

Kernel caches groups in `struct cred`:

$$memory_{per\_process} = |G(user)| \times sizeof(gid\_t) = |G| \times 4 \text{ bytes}$$

For 100 supplementary groups: 400 bytes per process credential structure.

### Primary Group for New Files

$$gid(new\_file) = \begin{cases} primary\_group(creator) & \text{if parent dir no setgid} \\ gid(parent\_dir) & \text{if parent dir has setgid} \end{cases}$$

---

## 7. /etc/group File Format and Scaling

### Entry Format

```
group_name:password:GID:member_list
```

### Lookup Performance

| Backend | Lookup | Scale |
|:---|:---:|:---|
| /etc/group (flat file) | $O(N_{groups})$ | < 500 groups |
| nscd cache | $O(1)$ amortized | < 10,000 groups |
| SSSD + LDAP | $O(\log N)$ + RTT | Unlimited |

### File Size

$$size_{group} = N_{groups} \times (avg\_name + 5 + avg\_members\_per\_group \times avg\_username\_len)$$

For 100 groups, average 10 members, 8-char names:

$$size = 100 \times (10 + 5 + 10 \times 9) = 100 \times 105 = 10.5 KB$$

### nsswitch Resolution Order

```
group: files sss
```

$$T_{lookup} = T_{files} + P(miss_{files}) \times T_{sss}$$

If 90% of lookups hit files: $T_{avg} = T_{files} + 0.1 \times T_{sss}$.

---

## 8. Summary of groups Mathematics

| Concept | Formula | Type |
|:---|:---|:---|
| Membership | $G(user) = \{primary\} \cup supplementary$ | Set |
| Access check | $gid \in G(user)$ | Set membership |
| Permission bits | $4r + 2w + x$ | Bitmask |
| setgid inheritance | $gid(file) = gid(dir)$ | Propagation |
| ACL effective | $perms \& mask$ | Bitwise AND |
| umask | $default \& \sim umask$ | Bitwise complement |
| Group limit | $NGROUPS\_MAX = 65536$ | Kernel constant |
| Graph density | $|E| / (|U| \times |G|)$ | Bipartite metric |

---

*Groups are the foundation of Unix collaborative access — a set-theoretic model where membership determines capability. Every file access is a set intersection test, every permission a bitmask operation, and every ACL an extension of the same algebra.*
