# Advanced SELinux — LSM Architecture, Type Enforcement Model, and MLS Implementation

> *SELinux implements mandatory access control through Linux Security Module (LSM) hooks that intercept every security-relevant system call. The security server evaluates access decisions against a compiled binary policy loaded into kernel memory, caching results in the Access Vector Cache (AVC) for performance. Type Enforcement (TE) is the primary enforcement mechanism, modeling the system as a matrix of source domains, target types, and permissions. Multi-Level Security (MLS) layers Bell-LaPadula confidentiality controls on top of TE, while Multi-Category Security (MCS) provides lightweight isolation for containers and virtual machines.*

---

## 1. SELinux Architecture

### LSM (Linux Security Module) Hooks

SELinux is implemented as an LSM — a kernel framework that inserts security check hooks at critical points in kernel code paths:

```
System call flow with SELinux:

  User process: open("/var/www/html/index.html", O_RDONLY)
      │
      ▼
  Kernel VFS layer: vfs_open()
      │
      ├── DAC check (standard UNIX permissions: owner/group/other)
      │   └── Denied? → return -EACCES (SELinux never consulted)
      │
      ├── LSM hook: security_file_open()
      │   └── SELinux: selinux_file_open()
      │       │
      │       ├── Get source context: task's security label (httpd_t)
      │       ├── Get target context: inode's security label (httpd_sys_content_t)
      │       ├── Determine object class: file
      │       ├── Determine permission: read, open
      │       │
      │       ├── AVC lookup(httpd_t, httpd_sys_content_t, file, {read,open})
      │       │   ├── Cache HIT → return cached decision
      │       │   └── Cache MISS → query security server
      │       │       └── Security server evaluates against loaded policy
      │       │           └── Cache result in AVC
      │       │
      │       └── Return: allowed / denied (+ audit if denied)
      │
      └── Proceed with file open (if all checks pass)

LSM hook coverage (~200 hooks):
  File operations:    open, read, write, execute, create, unlink, rename, ...
  Process operations: fork, exec, signal, ptrace, setuid, ...
  Network operations: socket, bind, connect, listen, send, recv, ...
  IPC operations:     semaphore, shared memory, message queue, ...
  Filesystem:         mount, remount, umount, ...
  Capability:         each Linux capability individually checked
```

### Security Server

The security server is the kernel component that evaluates access decisions:

```
Security server components:

  ┌─────────────────────────────────────────────────────┐
  │  Kernel Space                                        │
  │                                                      │
  │  ┌──────────────────┐    ┌────────────────────────┐ │
  │  │  LSM Hooks        │───▶│  AVC (Access Vector   │ │
  │  │  (200+ hooks)     │    │  Cache)                │ │
  │  │                   │    │  Hash table of recent  │ │
  │  │                   │    │  decisions              │ │
  │  └──────────────────┘    └────────────┬───────────┘ │
  │                                       │ miss         │
  │                           ┌───────────▼───────────┐ │
  │                           │  Security Server       │ │
  │                           │  ├── Policy database   │ │
  │                           │  ├── TE rule lookup    │ │
  │                           │  ├── RBAC constraint   │ │
  │                           │  ├── MLS constraint    │ │
  │                           │  └── Conditional eval  │ │
  │                           └───────────────────────┘ │
  │                                       ▲              │
  └───────────────────────────────────────│──────────────┘
                                          │ policy load
  ┌───────────────────────────────────────│──────────────┐
  │  User Space                           │              │
  │  ┌────────────────────────────────────┘            │ │
  │  │  /sys/fs/selinux/load  (selinuxfs)              │ │
  │  │  load_policy tool writes compiled policy here   │ │
  │  │                                                  │ │
  │  │  Policy source → checkmodule → semodule_package │ │
  │  │  → semodule → /sys/fs/selinux/load              │ │
  │  └─────────────────────────────────────────────────┘ │
  └──────────────────────────────────────────────────────┘
```

### Policy Database

The compiled policy is a binary blob loaded into kernel memory:

```
Policy structure in memory:

  Type Enforcement Rules:
    Hash table indexed by (source_type, target_type, class)
    → set of allowed permissions (bitvector)

  Type Transition Rules:
    (source_type, target_type, class) → new_type
    Determines label for newly created objects

  Role Allow Rules:
    (source_role) → set of allowed target roles

  Role Transition Rules:
    (source_role, target_type, class) → new_role

  MLS Constraints:
    Per (class, permission): expression tree over source/target levels

  Conditional Booleans:
    boolean_name → true/false
    Conditional rules activated/deactivated based on boolean state

Policy sizes (typical):
  Targeted policy (RHEL 9):  ~5-8 MB compiled
  MLS policy:                ~10-15 MB compiled
  ~100,000 TE rules, ~400 types, ~80 object classes
```

## 2. Type Enforcement Model

### Domain Transitions

When a process executes a binary, SELinux may transition the process to a new domain (type). This is how the principle of least privilege is enforced -- each daemon runs in its own confined domain:

```
Domain transition requirements:
  Three rules must ALL be present:

  1. allow source_domain target_exec_type : file { execute };
     → source domain can execute the binary

  2. allow source_domain target_domain : process { transition };
     → source domain can transition to target domain

  3. type_transition source_domain target_exec_type : process target_domain;
     → automatic transition (without explicit runcon)

Example: init (init_t) starting httpd:

  # init_t can execute httpd_exec_t
  allow init_t httpd_exec_t : file { read getattr execute open };

  # init_t can transition to httpd_t
  allow init_t httpd_t : process { transition };

  # When init_t executes httpd_exec_t, transition to httpd_t
  type_transition init_t httpd_exec_t : process httpd_t;

  # httpd_t is allowed to be entered (entrypoint)
  allow httpd_t httpd_exec_t : file { entrypoint read execute };

Transition flow:
  init_t (PID 1)
    └── execve("/usr/sbin/httpd")  [labeled httpd_exec_t]
        └── Kernel checks:
            1. Can init_t execute httpd_exec_t? ✓
            2. Can init_t transition to httpd_t? ✓
            3. type_transition rule exists? ✓ → automatic
            4. Is httpd_exec_t an entrypoint for httpd_t? ✓
            └── New process runs as httpd_t
```

### File Type Transitions

When a process creates a file, SELinux determines the new file's label:

```
File type transition:
  type_transition source_domain target_dir_type : file new_file_type;

Example: httpd creating log files
  type_transition httpd_t var_log_t : file httpd_log_t;
  → When httpd_t creates a file in a dir labeled var_log_t,
    the new file gets type httpd_log_t

Without a transition rule:
  New file inherits the type of the parent directory
  (This is why restorecon exists — to fix labels that inherited wrong type)

Named file transitions (more specific):
  type_transition httpd_t etc_t : file httpd_config_t "httpd.conf";
  → Only files named exactly "httpd.conf" created in etc_t get httpd_config_t
```

### Access Vector Computation

The security server computes the access vector (set of allowed permissions) for a (source, target, class) triple:

```
Access decision algorithm:

  Input: (source_type, target_type, object_class, requested_permissions)

  Step 1: Check type enforcement rules
          Search policy for: allow source_type target_type : class { perms }
          Result: set of allowed permissions (bitvector)

  Step 2: Check conditional rules (boolean-gated)
          For each conditional rule matching (source, target, class):
            If boolean is TRUE → include permissions
            If boolean is FALSE → exclude permissions
          Example: httpd_can_network_connect boolean gates:
            allow httpd_t port_type : tcp_socket { name_connect }

  Step 3: Check constraints (MLS, RBAC)
          Even if TE allows the access, constraints can deny it:
            MLS: source level must dominate target level (for read)
            RBAC: source role must be allowed to access target type

  Step 4: Compute final decision
          allowed = (TE_allowed ∩ conditional_allowed) - constrained_denied
          If requested_permissions ⊆ allowed → PERMIT
          Else → DENY (audit log generated)
```

## 3. RBAC in SELinux

### Role-Based Access Control

RBAC in SELinux constrains which types (domains) a user's role can access:

```
Role hierarchy:

  Role           Allowed Types (domains)
  ─────────────────────────────────────────────────
  object_r       All file/object types (not a user role)
  user_r         user_t, user_tmp_t, user_home_t, ...
  staff_r        staff_t, user_t, ...
  sysadm_r       sysadm_t, all admin domains
  system_r       All system daemon domains
  unconfined_r   unconfined_t (no type restrictions)

Role allow rules:
  allow staff_r sysadm_r;          # staff_r can transition to sysadm_r
  # This is what enables "sudo -r sysadm_r" for staff_u users

Role transition rules:
  role_transition staff_r sudo_exec_t sysadm_r;
  # When staff_r executes sudo (sudo_exec_t), transition to sysadm_r

Constraint (enforced in addition to TE):
  constrain process { transition }
    (u1 == u2  or  t1 == privrangetrans);
  # User must be the same before/after transition,
  # unless the source type has privrangetrans attribute
```

## 4. MLS/MCS Bell-LaPadula Implementation

### Bell-LaPadula Model

SELinux MLS implements the Bell-LaPadula (BLP) confidentiality model:

```
Bell-LaPadula properties:
  1. Simple Security (ss): No Read Up
     A subject at level L cannot READ objects at level > L
     Process at s1 (Confidential) cannot read file at s2 (Secret)

  2. Star Property (*): No Write Down
     A subject at level L cannot WRITE objects at level < L
     Process at s2 (Secret) cannot write file at s1 (Confidential)

  3. Strong Star Property: Read/Write only at own level
     Most restrictive interpretation

SELinux MLS levels:
  s0  = Unclassified
  s1  = Confidential
  s2  = Secret
  s3  = Top Secret
  (up to s15, configurable)

MLS range for subjects:
  s1-s3 means: user cleared for Confidential through Top Secret
  Current level can be any value within range
  Login default: low end of range

Dominance relation:
  s2 dominates s1 (s2 ≥ s1)
  s2:c1,c2 dominates s2:c1 (same level, superset of categories)
  s3:c1 dominates s2:c1,c2,c3 (higher level, regardless of categories)
```

### MLS Constraint Implementation

```
MLS constraints in SELinux policy:

  # File read: subject level must dominate object level
  mlsconstrain file { read getattr execute }
    (l1 dom l2 or t1 == mlsfileread);
  # l1 = subject's current level
  # l2 = object's level
  # dom = dominates (greater or equal)
  # mlsfileread = attribute for types exempt from MLS read checks

  # File write: object level must dominate subject level
  mlsconstrain file { write create setattr append unlink }
    (l1 domby l2 or t1 == mlsfilewrite);
  # domby = dominated by (less or equal) — enforces no write down

  # File read-write (same level): exact match
  mlsconstrain file { read write }
    (l1 eq l2 or t1 == mlsfilereadwrite);

Category constraints:
  Categories provide compartmentalization within a sensitivity level
  s2:c1 and s2:c2 are at the same sensitivity but different compartments
  A subject needs the category to access categorized objects
  s2:c1,c2 can access s2:c1 objects (superset of categories)
  s2:c1 cannot access s2:c1,c2 objects (not all categories present)
```

### MCS for Container Isolation

MCS is a simplified MLS that uses only categories (all at s0):

```
MCS in targeted policy:
  All subjects and objects at s0
  Categories c0-c1023 provide 1024 compartments
  Used primarily for container/VM isolation (sVirt)

Container isolation mechanism:
  Container A: s0:c100,c200
  Container B: s0:c300,c400

  Container A process (container_t:s0:c100,c200):
    Can access files labeled s0:c100,c200       ✓ (matching categories)
    Cannot access files labeled s0:c300,c400    ✗ (different categories)
    Cannot access files labeled s0              ✗ (no categories in subject)

  Category pair assignment:
    Random selection from c0-c1023 (2 categories = ~500K unique pairs)
    Assigned by container runtime (Docker/Podman/libvirt)
    Stored in container metadata

  Unconfined processes (s0:c0.c1023) can access ALL categories
    → Host admin can inspect any container's files
```

## 5. Policy Compilation

### Build Pipeline

```
Policy source compilation pipeline:

  Source files:
  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
  │  .te files   │  │  .if files   │  │  .fc files   │
  │  (type       │  │  (interface  │  │  (file       │
  │  enforcement)│  │  macros)     │  │  contexts)   │
  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘
         │                 │                 │
         ▼                 ▼                 ▼
  ┌──────────────────────────────────────────────────┐
  │  checkmodule -M -m -o module.mod module.te       │
  │  (compiles TE source to intermediate module)     │
  └──────────────────────┬───────────────────────────┘
                         │
                         ▼
  ┌──────────────────────────────────────────────────┐
  │  semodule_package -o module.pp -m module.mod     │
  │  [-f module.fc]                                  │
  │  (packages module + file contexts into .pp)      │
  └──────────────────────┬───────────────────────────┘
                         │
                         ▼
  ┌──────────────────────────────────────────────────┐
  │  semodule -i module.pp                           │
  │  (installs module into module store, rebuilds    │
  │   and reloads complete policy)                   │
  └──────────────────────┬───────────────────────────┘
                         │
                         ▼
  ┌──────────────────────────────────────────────────┐
  │  /sys/fs/selinux/load                            │
  │  (kernel loads new binary policy)                │
  └──────────────────────────────────────────────────┘

Module store location:
  /var/lib/selinux/targeted/active/modules/  (RHEL 8+)
  Each module: priority/name/ directory with .cil or .pp

Full policy rebuild on semodule -i:
  All active modules combined → single binary policy
  This is why semodule can be slow (seconds to minutes)
```

### CIL (Common Intermediate Language)

Modern SELinux uses CIL as an intermediate representation:

```
CIL replaces the older binary .pp format:
  - Human-readable S-expression syntax
  - Richer semantics (namespaces, macros, inheritance)
  - Better for programmatic policy generation

CIL example:
  (type httpd_t)
  (role system_r)
  (roletype system_r httpd_t)
  (allow httpd_t httpd_sys_content_t (file (read getattr open)))

Compilation with CIL:
  secilc policy.cil -o policy.bin    # direct CIL to binary compilation
  semodule -i module.cil             # install CIL module directly
```

## 6. AVC Cache and Performance

### AVC Design

The Access Vector Cache is critical for SELinux performance. Without it, every system call would require a full policy lookup:

```
AVC structure:
  Hash table in kernel memory
  Key: (source_type, target_type, object_class)
  Value: allowed permission bitvector + audit flags

  Default size: 512 entries (configurable via avc_cache_threshold)
  Hash function: Jenkins hash of (src, tgt, class)
  Collision resolution: chaining

Performance characteristics:
  AVC hit:  ~100 nanoseconds (hash lookup + bitvector check)
  AVC miss: ~10-100 microseconds (security server policy evaluation)
  Hit rate:  typically 99%+ in steady-state workloads

  Impact on system calls:
    Without SELinux: syscall overhead = DAC check only
    With SELinux:    syscall overhead = DAC check + AVC lookup
    Typical overhead: 1-7% on macro benchmarks (kernel compilation, etc.)
    Worst case: 10-15% on syscall-intensive microbenchmarks
```

### AVC Statistics

```bash
# View AVC statistics
cat /sys/fs/selinux/avc/cache_stats
# lookups  hits  misses  allocations  reclaims  frees
# 1234567  1234000  567  567          0         0

# AVC cache size
cat /sys/fs/selinux/avc/cache_threshold    # max entries (default 512)
echo 1024 > /sys/fs/selinux/avc/cache_threshold  # increase for busy systems

# Flush AVC cache (forces re-evaluation of all decisions)
echo 1 > /sys/fs/selinux/avc/cache_stats   # or
avcstat                                     # human-readable stats
```

### AVC Invalidation

```
AVC entries are invalidated when:
  1. Policy reload (semodule -i, setsebool)
     → Full AVC flush (all entries invalidated)
     → Temporary performance dip as cache repopulates

  2. Boolean change (setsebool)
     → Only entries affected by the boolean are invalidated
     → More efficient than full flush

  3. Cache full (LRU eviction)
     → Oldest/least-used entries evicted
     → Normal operation, not a problem unless cache is too small

  4. Security context change (chcon, restorecon)
     → Entries for changed context invalidated
```

## 7. SELinux vs AppArmor Comparison

```
Dimension              SELinux                      AppArmor
─────────────────────────────────────────────────────────────────────────
Label model            Labels on ALL objects         Path-based rules
                       (type enforcement)            (no object labels)

Labeling               Every file, process, port,   Rules reference file
                       socket has a context          PATHS — no labeling needed
                       (label stored in xattr)

Policy model           Allow-list (default deny,    Allow-list (default deny
                       explicit allow rules)        per profile, explicit allow)

Granularity            Very fine (per-type, per-    Moderate (per-path, per-
                       class, per-permission)       capability)

MLS/MCS                Full BLP implementation      No equivalent
                       + category-based isolation

Hard links             Secure (labels follow        Insecure (hard link to
                       inode, not path)             different path bypasses rule)

Rename/move            Labels persist (inode-       Rules may not apply (path
                       based)                       changed, profile may not
                                                    match new path)

Complexity             High (steep learning curve,  Moderate (easier to write
                       complex policy language)     profiles, familiar path-based)

Filesystem support     Requires xattr support       No filesystem requirements
                       (ext4, xfs, btrfs — NOT     (works on any filesystem)
                       NFS without special config)

Default on             RHEL, Fedora, CentOS,       Ubuntu, SUSE, Debian
                       Rocky, Alma                  (optional)

Container isolation    sVirt with MCS categories    Profile stacking (since 2.13)

Performance            ~1-7% overhead              ~1-5% overhead
                       (AVC cache is very efficient) (path lookup + profile match)

Tooling                semanage, audit2allow,       aa-genprof, aa-logprof,
                       sealert, sesearch            aa-complain, aa-enforce

Use case fit           High-security, compliance    General-purpose server
                       (government, military,       hardening, desktop
                       financial), container        confinement, simpler
                       isolation at scale           environments
```

## 8. Custom Policy Development Methodology

### Recommended Development Workflow

```
Step 1: Identify the application
  - What binary? What user runs it?
  - What files does it read/write/execute?
  - What network ports does it use?
  - What IPC mechanisms (pipes, sockets, shared memory)?
  - What other processes does it interact with?

Step 2: Run in permissive mode and generate denials
  # Create a permissive domain for just this app
  semanage permissive -a myapp_t
  # Run the application through ALL its functionality
  # Collect ALL AVC denials

Step 3: Generate initial policy
  sudo ausearch -m avc -c myapp | audit2allow -R -M myapp
  # -R flag generates reference policy-style module (uses macros)
  # Review the .te file — audit2allow is often too permissive

Step 4: Refine the policy
  # Remove overly broad rules
  # Replace type access with appropriate macros:
  #   files_read_etc_files(myapp_t)     instead of    allow myapp_t etc_t:file read
  #   corenet_tcp_bind_http_port(myapp_t) instead of  allow myapp_t http_port_t:tcp_socket name_bind
  # Add file context definitions (.fc file)
  # Add interface definitions (.if file) if other modules need to interact

Step 5: Test
  # Install module
  sudo semodule -i myapp.pp
  # Switch domain to enforcing
  sudo semanage permissive -d myapp_t
  # Test ALL application functionality
  # Check for new denials: ausearch -m avc -c myapp

Step 6: Iterate
  # New denials → analyze → add specific rules → recompile → test
  # Repeat until clean operation in enforcing mode

Step 7: Package and distribute
  # Include .te, .if, .fc, and Makefile in RPM/DEB package
  # Run semodule -i in %post scriptlet
  # Run semodule -r in %preun scriptlet
```

### Policy Module Template

```
# myapp.te — Type Enforcement
policy_module(myapp, 1.0.0)

########################################
# Type declarations
########################################
type myapp_t;                        # process domain
type myapp_exec_t;                   # executable type
type myapp_conf_t;                   # configuration files
type myapp_data_t;                   # data files
type myapp_log_t;                    # log files
type myapp_var_run_t;                # PID/socket files

# Mark as a daemon domain
init_daemon_domain(myapp_t, myapp_exec_t)

# Mark file types
files_type(myapp_conf_t)
files_type(myapp_data_t)
logging_log_file(myapp_log_t)
files_pid_file(myapp_var_run_t)

########################################
# Policy rules
########################################

# Read configuration
allow myapp_t myapp_conf_t:file { read getattr open };
allow myapp_t myapp_conf_t:dir { search getattr };

# Read/write data
allow myapp_t myapp_data_t:file { read write create getattr setattr open append unlink };
allow myapp_t myapp_data_t:dir { read write add_name remove_name search getattr };

# Write logs
allow myapp_t myapp_log_t:file { create write append getattr setattr open };
logging_log_filetrans(myapp_t, myapp_log_t, file)

# PID file
allow myapp_t myapp_var_run_t:file { create write read getattr setattr unlink open };
files_pid_filetrans(myapp_t, myapp_var_run_t, file)

# Network
corenet_tcp_bind_generic_port(myapp_t)
sysnet_dns_name_resolve(myapp_t)

# System access
files_read_etc_files(myapp_t)
miscfiles_read_localization(myapp_t)
kernel_read_system_state(myapp_t)
```

```
# myapp.fc — File Contexts
/usr/bin/myapp                     -- gen_context(system_u:object_r:myapp_exec_t,s0)
/etc/myapp(/.*)?                      gen_context(system_u:object_r:myapp_conf_t,s0)
/var/lib/myapp(/.*)?                  gen_context(system_u:object_r:myapp_data_t,s0)
/var/log/myapp(/.*)?                  gen_context(system_u:object_r:myapp_log_t,s0)
/run/myapp\.pid                    -- gen_context(system_u:object_r:myapp_var_run_t,s0)
```

## See Also

- selinux
- apparmor
- capabilities
- auditd
- hardening-linux
- polkit

## References

- SELinux Notebook: https://github.com/SELinuxProject/selinux-notebook
- Smalley, Vance, Salamon: "Implementing SELinux as a Linux Security Module" (NSA Technical Report)
- Red Hat SELinux Coloring Book (visual introduction)
- SELinux Project: https://selinuxproject.org/
- Fedora SELinux Policy Source: https://github.com/fedora-selinux/selinux-policy
- CIL Reference Guide: https://github.com/SELinuxProject/cil
