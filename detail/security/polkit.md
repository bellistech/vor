# PolicyKit — D-Bus Authorization Framework, Rule Engine, and Decision Architecture

> *PolicyKit (polkit) is a D-Bus-centric authorization framework that mediates between unprivileged applications and privileged system services. Unlike sudo's command-centric model, polkit authorizes discrete actions identified by reverse-DNS strings, evaluated through a JavaScript rule engine with access to subject identity, session state, and action metadata. The architecture separates policy definition (XML .policy files), policy evaluation (JavaScript .rules files), and user interaction (authentication agents) into distinct components.*

---

## 1. PolicyKit as D-Bus Authorization Framework

### The Authorization Problem

Traditional UNIX privilege escalation is binary: you are root or you are not. setuid binaries (like `sudo`) grant full root to run a command. D-Bus system services need finer granularity -- a user should be able to mount a USB drive without being able to format the system disk.

```
Problem with setuid/sudo model:
  Application → setuid binary → FULL ROOT PRIVILEGES → perform operation
  No way to say "this user can mount removable drives but not fixed drives"

polkit model:
  Application → D-Bus request → Service checks polkit → specific action authorized?
  Action: org.freedesktop.udisks2.filesystem-mount (removable=true)  → YES
  Action: org.freedesktop.udisks2.filesystem-mount (removable=false) → auth_admin
```

### D-Bus Integration Points

polkit authorization happens at the D-Bus system service level, not at the D-Bus daemon level:

```
Authorization flow in detail:

  1. Client calls D-Bus method:
     busctl call org.freedesktop.UDisks2 \
       /org/freedesktop/UDisks2/block_devices/sdb1 \
       org.freedesktop.UDisks2.Filesystem Mount "a{sv}" 0

  2. D-Bus daemon routes message to udisksd (no authorization at this point)
     D-Bus bus policy (/usr/share/dbus-1/system.d/*.conf) only controls
     which users can SEND messages — not whether the operation is authorized

  3. udisksd receives Mount request, extracts caller info:
     - Caller's D-Bus unique name (:1.42)
     - Caller's PID, UID (via D-Bus credentials)
     - Action parameters (which device, mount options)

  4. udisksd calls polkit:
     polkit_authority_check_authorization_sync(
       authority,
       subject,              // PolkitUnixProcess(pid, uid, start_time)
       "org.freedesktop.udisks2.filesystem-mount",
       details,              // key-value pairs (device, mount-point, etc.)
       POLKIT_CHECK_AUTHORIZATION_FLAGS_ALLOW_USER_INTERACTION,
       cancellable,
       &error
     )

  5. polkitd evaluates:
     a. Load .policy file for action → get defaults
     b. Evaluate .rules files in order → first result wins
     c. If result requires authentication → signal agent
     d. Return: AUTHORIZED / NOT_AUTHORIZED / CHALLENGE

  6. udisksd acts on result:
     AUTHORIZED → perform mount
     NOT_AUTHORIZED → return D-Bus error to client
```

### Subject Identity Resolution

polkit identifies authorization subjects by process, not by user alone:

```
Subject identification:
  PolkitUnixProcess {
    pid:        12345
    uid:        1000
    start_time: 1712345678.123456  // prevents PID reuse attacks
  }

From the process, polkit derives:
  user:    getpwuid(uid) → "jdoe"
  groups:  getgrouplist("jdoe") → ["jdoe", "wheel", "virt-admins"]
  session: logind session for PID → session properties:
    local:  true/false  (local console vs remote SSH)
    active: true/false  (foreground session vs background)
    type:   "tty" | "x11" | "wayland" | "unspecified"
    seat:   "seat0" (physical seat assignment)

Security: start_time prevents TOCTOU attacks where:
  1. Authorized process exits
  2. Malicious process takes same PID
  3. polkit checks stale PID → wrong process!
  start_time ensures PID + start_time uniquely identify a process
```

## 2. Agent Architecture

### Agent Types and Registration

Authentication agents are user-facing programs that collect credentials when polkit requires interactive authentication:

```
Agent registration flow:
  1. Agent starts (typically at desktop session login)
  2. Agent registers with polkitd over D-Bus:
     RegisterAuthenticationAgent(subject, locale, object_path)
     - subject: the session to handle (current logind session)
     - locale: for localized messages
     - object_path: D-Bus path where agent listens

  3. When polkitd needs authentication:
     polkitd → D-Bus → Agent.BeginAuthentication(
       action_id,           // "org.freedesktop.udisks2.filesystem-mount"
       message,            // "Authentication is required to mount /dev/sdb1"
       icon_name,          // "dialog-password"
       cookie,             // unique session cookie
       identities          // which users can authenticate (admin users)
     )

  4. Agent displays UI (GTK dialog, KDE dialog, or TTY prompt)
  5. User enters password
  6. Agent calls polkitd:
     AuthenticationAgentResponse(cookie, identity)
  7. polkitd verifies password via PAM
  8. Returns result to original authorization check

Agent types:
  polkit-gnome-agent     — GTK3 dialog (GNOME, Xfce, etc.)
  polkit-kde-agent       — Qt dialog (KDE Plasma)
  polkit-lxqt-agent      — LXQt dialog
  pkttyagent             — terminal/TTY prompt (SSH, console)
  lxpolkit               — LXDE agent
```

### Agent Selection

```
Agent priority:
  1. Only ONE agent per session (last registered wins)
  2. Desktop environments auto-start their agent via XDG autostart
  3. For sessions without a registered agent:
     - polkitd returns NOT_AUTHORIZED (no way to prompt)
     - This is why SSH sessions need manual pkttyagent registration

Session → Agent mapping:
  logind session "c1" (local, GNOME) → polkit-gnome-agent (auto-registered)
  logind session "2"  (SSH, no DE)   → no agent (until pkttyagent started)
  logind session "3"  (SSH + tmux)   → pkttyagent (if manually started)
```

### Credential Caching

When authorization result is `auth_admin_keep` or `auth_self_keep`, the credential is cached temporarily:

```
Cache behavior:
  Scope:   per-subject (PID + start_time)
  Duration: implementation-defined, typically ~5 minutes
  Storage:  in-memory only (lost on polkitd restart)

  auth_admin_keep → "remember that user X authenticated as admin for 5 min"
  auth_self_keep  → "remember that user X authenticated as self for 5 min"

  Unlike sudo's timestamp:
    - polkit cache is per-process, not per-terminal
    - No configurable timeout in standard polkit (implementation varies)
    - Some implementations (polkit-1 0.120+) use a 2-minute default
```

## 3. Authorization Decision Flow

### Complete Decision Algorithm

```
polkitd receives CheckAuthorization(subject, action_id, details, flags):

  Step 1: Validate action_id exists in .policy files
          Not found → return NOT_AUTHORIZED

  Step 2: Resolve subject → user, groups, session properties

  Step 3: Evaluate JavaScript rules (in file-sort order):
          /etc/polkit-1/rules.d/*.rules (sorted lexicographically)
          /usr/share/polkit-1/rules.d/*.rules (sorted lexicographically)

          For each addRule callback:
            result = callback(action, subject)
            if result != null → use this result, STOP evaluation
            if result == null → continue to next rule

  Step 4: If no rule returned a result, use .policy defaults:
          Determine session type:
            local + active  → use <allow_active>
            local + inactive → use <allow_inactive>
            remote / other  → use <allow_any>

  Step 5: Map result to response:
          YES        → AUTHORIZED
          NO         → NOT_AUTHORIZED
          AUTH_SELF  → check credential cache
                       cached? → AUTHORIZED
                       not cached + ALLOW_USER_INTERACTION flag? → trigger agent
                       not cached + no flag? → CHALLENGE
          AUTH_ADMIN → same as AUTH_SELF but identities = admin users

  Step 6: If agent triggered:
          Agent collects password → PAM verification
          PAM success → AUTHORIZED (cache credential if _keep variant)
          PAM failure → NOT_AUTHORIZED
```

### Admin Identity Resolution

When `auth_admin` is required, polkit needs to know who counts as an "admin":

```
Default admin identification (can be overridden in rules):

  polkit.addAdminRule(function(action, subject) {
      // Default: members of "wheel" group are admins
      return ["unix-group:wheel"];
  });

  // Or specify individual users:
  polkit.addAdminRule(function(action, subject) {
      return ["unix-user:admin1", "unix-user:admin2", "unix-group:sudo"];
  });

  // Per-action admin rules:
  polkit.addAdminRule(function(action, subject) {
      if (action.id.indexOf("org.libvirt") == 0) {
          return ["unix-group:virt-admins"];
      }
      return ["unix-group:wheel"];
  });
```

## 4. Rules Evaluation Order

### File Ordering

```
Evaluation sequence (critical for understanding precedence):

  /etc/polkit-1/rules.d/00-deny-kiosk.rules       ← evaluated FIRST
  /etc/polkit-1/rules.d/10-allow-wheel-mount.rules
  /etc/polkit-1/rules.d/49-custom.rules
  /usr/share/polkit-1/rules.d/50-default.rules     ← vendor defaults
  /etc/polkit-1/rules.d/80-late-overrides.rules
  /usr/share/polkit-1/rules.d/99-fallback.rules    ← evaluated LAST

  Within a single file: addRule callbacks evaluated in definition order
  First non-null return from ANY callback → final answer

Strategy:
  00-09: Deny rules (always deny specific users/actions)
  10-49: Allow rules (grant access to specific groups/users)
  50:    Vendor defaults (shipped with packages)
  51-89: Late overrides (override vendor defaults)
  90-99: Catch-all / logging rules
```

### Rule Interaction Patterns

```
Pattern 1: Deny overrides allow
  // 00-deny.rules
  polkit.addRule(function(action, subject) {
      if (subject.user == "contractor") return polkit.Result.NO;
  });
  // 10-allow.rules (NEVER reached for "contractor")
  polkit.addRule(function(action, subject) {
      if (subject.isInGroup("developers")) return polkit.Result.YES;
  });

Pattern 2: Specific overrides general
  // 10-specific.rules
  polkit.addRule(function(action, subject) {
      if (action.id == "org.freedesktop.login1.reboot" &&
          subject.user == "operator") return polkit.Result.YES;
  });
  // 50-general.rules (reached for other users/actions)
  polkit.addRule(function(action, subject) {
      if (action.id.indexOf("org.freedesktop.login1") == 0)
          return polkit.Result.AUTH_ADMIN;
  });

Pattern 3: Logging without deciding
  // 00-log.rules
  polkit.addRule(function(action, subject) {
      polkit.log("CHECK: " + action.id + " by " + subject.user);
      return null;  // pass-through — does NOT consume the check
  });
```

## 5. JavaScript Rule Engine

### Engine Details

polkit uses the Duktape JavaScript engine (embedded, lightweight):

```
Engine characteristics:
  - ECMAScript E5/E5.1 compliant (NOT ES6+)
  - No modules, no require(), no import
  - No network access, no filesystem access
  - No setTimeout/setInterval
  - Sandboxed: only polkit.* API available
  - Each rules file evaluated in shared global scope
  - Rules persist in memory until polkitd restart

Available API:
  polkit.addRule(function(action, subject) { ... })
  polkit.addAdminRule(function(action, subject) { ... })
  polkit.log(string)       // output to polkitd journal
  polkit.spawn(argv)       // DANGEROUS: run external command (disabled in some distros)

  polkit.Result.YES
  polkit.Result.NO
  polkit.Result.AUTH_SELF
  polkit.Result.AUTH_ADMIN
  polkit.Result.AUTH_SELF_KEEP
  polkit.Result.AUTH_ADMIN_KEEP

Action object:
  action.id              // "org.freedesktop.udisks2.filesystem-mount"
  action.lookup("key")   // action-specific detail (set by requesting service)

Subject object:
  subject.user           // "jdoe"
  subject.isInGroup("g") // true/false
  subject.local          // true/false
  subject.active         // true/false
  subject.session        // logind session ID
  subject.pid            // process ID
```

### polkit.spawn() Security

```
polkit.spawn() executes an external command and returns its stdout.
This is powerful but DANGEROUS — it runs as the polkitd user (usually root).

Example (check LDAP group — NOT recommended for production):
  polkit.addRule(function(action, subject) {
      var output = polkit.spawn(["/usr/bin/ldapsearch", "-x",
          "-b", "cn=virt-admins,ou=groups,dc=example,dc=com",
          "(memberUid=" + subject.user + ")"]);
      if (output.indexOf("memberUid: " + subject.user) >= 0) {
          return polkit.Result.YES;
      }
  });

Risks:
  - Command injection if subject.user contains shell metacharacters
  - Performance: blocks polkitd while command runs
  - Availability: if LDAP is down, polkitd hangs
  - Some distributions disable polkit.spawn() entirely

Better alternative:
  - Use local group membership (subject.isInGroup) — groups synced by SSSD
  - Pre-compute authorization in local files
```

## 6. polkit vs sudo Comparison

```
Dimension            polkit                        sudo
─────────────────────────────────────────────────────────────────────────
Model                Action-based                  Command-based
                     "can user do X?"              "can user run Y as root?"

Granularity          Per-action with metadata      Per-command with arguments
                     (action.lookup("unit"))       (/usr/bin/systemctl restart *)

Configuration        .policy XML + .rules JS       /etc/sudoers
                     Separate definition/eval      Single file

Authentication       Pluggable agents (GUI/TTY)    TTY-only (tty_tickets)
                     Per-session caching            Timestamp file caching

Desktop integration  Native (designed for it)      Bolted on (pkexec wraps sudo)

Remote (SSH)         Requires agent registration   Works out of the box

Audit                Via polkitd journal            Via sudo log / sudoers log_output

Complexity           Higher (D-Bus, JS engine,     Lower (single binary, single
                     multiple components)          config file)

Use case             Desktop privilege prompts,    CLI administration,
                     service authorization         automated scripts,
                                                   fine-grained command control

Can run arbitrary    Only via pkexec               Yes (primary use case)
commands?            (limited, clunky)

Environment          Sanitized (minimal env)       Configurable (env_keep,
                                                   env_reset)

Typical deployment   Desktop Linux, libvirt,       Everywhere (servers, desktops,
                     NetworkManager, systemd       containers, CI/CD)
```

## 7. polkit in Container/VM Management

### libvirt Integration

libvirt is one of the most common polkit consumers:

```
libvirt polkit actions:
  org.libvirt.unix.monitor    → connect read-only (default: yes)
  org.libvirt.unix.manage     → connect read-write (default: auth_admin)

When virt-manager connects to libvirtd:
  1. virt-manager → D-Bus → libvirtd
  2. libvirtd checks polkit for org.libvirt.unix.manage
  3. polkit evaluates rules → triggers agent if needed
  4. On success: full VM management access

Common rule for developer workstations:
  // Allow libvirt group to manage VMs without password
  polkit.addRule(function(action, subject) {
      if (action.id.indexOf("org.libvirt") == 0 &&
          subject.isInGroup("libvirt") &&
          subject.local && subject.active) {
          return polkit.Result.YES;
      }
  });
```

### Container runtime authorization

```
Podman (rootless containers) uses polkit for certain privileged operations:
  org.freedesktop.machine1.manage-machines    → systemd-machined
  org.freedesktop.systemd1.manage-units       → starting container units

Flatpak uses polkit for:
  org.freedesktop.Flatpak.app-install         → installing applications
  org.freedesktop.Flatpak.runtime-install     → installing runtimes
  org.freedesktop.Flatpak.modify-repo         → adding/removing remotes
```

## 8. Security Implications

### Attack Surface

```
polkitd attack surface:
  1. D-Bus interface (org.freedesktop.PolicyKit1)
     - Any local user can call CheckAuthorization
     - Race conditions in subject validation (CVE-2021-4034: pkexec)

  2. JavaScript rule engine
     - Sandboxed but bugs in Duktape could escape
     - polkit.spawn() if enabled is a command injection risk

  3. Authentication agent
     - Agent runs in user session (not as root)
     - Fake agent could trick user into authenticating for wrong action
     - Agent ↔ polkitd communication authenticated by D-Bus

  4. PID-based subject identification
     - Mitigated by start_time check
     - /proc race conditions historically exploitable (CVE-2018-19788)

Notable CVEs:
  CVE-2021-4034 (PwnKit): pkexec local privilege escalation
    - Null argv[0] caused out-of-bounds write
    - Trivial root on any system with pkexec installed
    - Fixed in polkit 0.120+

  CVE-2021-3560: D-Bus message timing attack
    - Kill D-Bus message mid-flight → polkitd processes partial request
    - Could bypass authentication requirement
    - Fixed in polkit 0.119+
```

### Hardening Recommendations

```
1. Minimize installed .policy files
   - Remove packages you don't need (each brings polkit actions)
   - Audit: pkaction | wc -l (how many actions registered?)

2. Deny-first rule strategy
   - /etc/polkit-1/rules.d/00-deny-all.rules as baseline
   - Explicit allow rules for required operations only

3. Disable polkit.spawn() if not needed
   - Compile polkit without spawn support, or
   - Audit all rules for spawn usage

4. Monitor polkit decisions
   - journalctl -u polkit (all authorization checks logged)
   - Forward to SIEM for anomaly detection

5. Keep polkit updated
   - Critical privilege escalation CVEs are common
   - Subscribe to distribution security advisories
```

## See Also

- pam
- capabilities
- selinux
- apparmor
- dbus
- systemd

## References

- polkit Reference Manual: https://www.freedesktop.org/software/polkit/docs/latest/
- polkit Source Code: https://gitlab.freedesktop.org/polkit/polkit
- David Zeuthen: "Making PolicyKit Just Work" (GNOME design document)
- CVE-2021-4034 (PwnKit): https://blog.qualys.com/vulnerabilities-threat-research/2022/01/25/pwnkit
- CVE-2021-3560: https://github.blog/2021-06-10-privilege-escalation-polkit-root-on-linux-with-bug/
