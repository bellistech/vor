# PolicyKit (polkit)

Authorization framework for controlling system-wide privileges through D-Bus -- allows unprivileged processes to request privileged operations with fine-grained rules.

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  Unprivileged Application (e.g., gnome-disks, virt-manager)
│  "I want to mount /dev/sdb1"                         │
└──────────┬───────────────────────────────────────────┘
           │ D-Bus method call
           ▼
┌──────────────────────────────────────────────────────┐
│  Privileged Service (e.g., udisksd, libvirtd)        │
│  "Let me check if caller is authorized"              │
│  polkit_authority_check_authorization()               │
└──────────┬───────────────────────────────────────────┘
           │ D-Bus → org.freedesktop.PolicyKit1
           ▼
┌──────────────────────────────────────────────────────┐
│  polkitd (PolicyKit Authority Daemon)                │
│  1. Look up action in .policy XML files              │
│  2. Evaluate .rules JavaScript files                 │
│  3. If needed, ask authentication agent              │
└──────────┬───────────────────────────────────────────┘
           │
           ▼
┌──────────────────────────────────────────────────────┐
│  Authentication Agent (e.g., polkit-gnome-agent,     │
│  polkit-kde-agent, pkttyagent)                       │
│  "Please enter your password"                         │
└──────────────────────────────────────────────────────┘
```

## Actions

### List all registered actions

```bash
pkaction                                     # list all action IDs
pkaction --verbose                           # full details for all actions
pkaction --action-id org.freedesktop.udisks2.filesystem-mount --verbose
```

### Check authorization for an action

```bash
pkcheck --action-id org.freedesktop.udisks2.filesystem-mount \
  --process $$ --allow-user-interaction
# Returns exit code 0 = authorized, 1 = not authorized
```

### Common action IDs

```
Action ID                                              Default
─────────────────────────────────────────────────────────────────
org.freedesktop.udisks2.filesystem-mount               auth_admin
org.freedesktop.udisks2.filesystem-mount-other-seat    auth_admin
org.freedesktop.login1.reboot                          auth_admin_keep
org.freedesktop.login1.power-off                       auth_admin_keep
org.freedesktop.login1.suspend                         yes
org.freedesktop.login1.hibernate                       yes
org.freedesktop.systemd1.manage-units                  auth_admin
org.freedesktop.systemd1.manage-unit-files             auth_admin
org.freedesktop.NetworkManager.settings.modify.system  auth_admin_keep
org.freedesktop.packagekit.system-update               auth_admin
org.libvirt.unix.manage                                auth_admin
org.libvirt.unix.monitor                               yes
org.freedesktop.policykit.exec                         auth_admin
```

### Authorization results

```
Authorization Result     Meaning
──────────────────────────────────────────────────────────
yes                      Always authorized (no prompt)
no                       Always denied
auth_self                User must authenticate as themselves
auth_admin               User must authenticate as admin
auth_self_keep           Like auth_self, credential cached briefly
auth_admin_keep          Like auth_admin, credential cached briefly
```

## pkexec

### Run command as root via polkit

```bash
pkexec /usr/bin/systemctl restart nginx
pkexec --user root /usr/sbin/fdisk -l
pkexec env DISPLAY=$DISPLAY xterm            # GUI app (needs special rule)
```

### pkexec vs sudo

```
pkexec                                 sudo
├── D-Bus authorization framework      ├── setuid binary, /etc/sudoers
├── GUI agent (password dialog)        ├── TTY prompt
├── JavaScript rules (.rules)          ├── sudoers syntax
├── Per-action granularity             ├── Per-command granularity
├── Session-based caching              ├── Timestamp-based caching
└── Designed for desktop integration   └── Designed for CLI/scripts
```

## Policy Files (.policy XML)

### Location

```bash
ls /usr/share/polkit-1/actions/              # system policy files
# Files named: org.freedesktop.*.policy
```

### Policy file structure

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE policyconfig PUBLIC
 "-//freedesktop//DTD PolicyKit Policy Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/PolicyKit/1/policyconfig.dtd">
<policyconfig>
  <vendor>My Application</vendor>
  <vendor_url>https://example.com</vendor_url>

  <action id="com.example.myapp.do-something">
    <description>Perform a privileged operation</description>
    <message>Authentication is required to perform this operation</message>
    <icon_name>dialog-password</icon_name>
    <defaults>
      <allow_any>no</allow_any>
      <allow_inactive>no</allow_inactive>
      <allow_active>auth_admin</allow_active>
    </defaults>
    <annotate key="org.freedesktop.policykit.exec.path">/usr/bin/myapp-helper</annotate>
    <annotate key="org.freedesktop.policykit.exec.allow_gui">true</annotate>
  </action>
</policyconfig>
```

### Default authorization elements

```xml
<defaults>
  <allow_any>no</allow_any>                  <!-- remote / any session -->
  <allow_inactive>no</allow_inactive>        <!-- local inactive session -->
  <allow_active>auth_admin</allow_active>    <!-- local active session -->
</defaults>

<!-- Values: yes | no | auth_self | auth_admin | auth_self_keep | auth_admin_keep -->
```

### Create a custom action

```bash
# /usr/share/polkit-1/actions/com.example.restart-nginx.policy
```

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE policyconfig PUBLIC
 "-//freedesktop//DTD PolicyKit Policy Configuration 1.0//EN"
 "http://www.freedesktop.org/standards/PolicyKit/1/policyconfig.dtd">
<policyconfig>
  <action id="com.example.restart-nginx">
    <description>Restart Nginx web server</description>
    <message>Authentication is required to restart Nginx</message>
    <defaults>
      <allow_any>no</allow_any>
      <allow_inactive>no</allow_inactive>
      <allow_active>auth_admin_keep</allow_active>
    </defaults>
  </action>
</policyconfig>
```

## Rules Files (.rules JavaScript)

### Location and evaluation order

```bash
/etc/polkit-1/rules.d/                       # local/admin rules (higher priority)
/usr/share/polkit-1/rules.d/                 # vendor/package rules

# Files evaluated in lexicographic order:
#   /etc/polkit-1/rules.d/00-early.rules     ← first
#   /etc/polkit-1/rules.d/49-custom.rules
#   /usr/share/polkit-1/rules.d/50-default.rules
#   /etc/polkit-1/rules.d/99-late.rules      ← last
# First rule that returns a result wins (no further evaluation)
```

### Allow group to perform action without password

```javascript
// /etc/polkit-1/rules.d/10-allow-wheel-mount.rules
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.udisks2.filesystem-mount" &&
        subject.isInGroup("wheel")) {
        return polkit.Result.YES;
    }
});
```

### Allow specific user to manage systemd units

```javascript
// /etc/polkit-1/rules.d/20-systemd-webadmin.rules
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.systemd1.manage-units" &&
        subject.user == "webadmin") {
        return polkit.Result.YES;
    }
});
```

### Allow libvirt management for virt-admins group

```javascript
// /etc/polkit-1/rules.d/30-libvirt.rules
polkit.addRule(function(action, subject) {
    if (action.id.indexOf("org.libvirt.unix.manage") == 0 &&
        subject.isInGroup("virt-admins")) {
        return polkit.Result.YES;
    }
});
```

### Allow NetworkManager changes for netadmin group

```javascript
// /etc/polkit-1/rules.d/40-network.rules
polkit.addRule(function(action, subject) {
    if (action.id.indexOf("org.freedesktop.NetworkManager") == 0 &&
        subject.isInGroup("netadmin")) {
        return polkit.Result.YES;
    }
});
```

### Deny action for specific users

```javascript
// /etc/polkit-1/rules.d/05-deny-reboot.rules
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.login1.reboot" &&
        subject.user == "kiosk") {
        return polkit.Result.NO;
    }
});
```

### Require auth_self instead of auth_admin

```javascript
// /etc/polkit-1/rules.d/50-self-auth-updates.rules
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.packagekit.system-update" &&
        subject.local &&
        subject.active) {
        return polkit.Result.AUTH_SELF;
    }
});
```

### Log all authorization checks (debugging)

```javascript
// /etc/polkit-1/rules.d/00-log-all.rules
polkit.addRule(function(action, subject) {
    polkit.log("action=" + action.id +
               " user=" + subject.user +
               " local=" + subject.local +
               " active=" + subject.active);
    return null;  // don't decide, let other rules evaluate
});
```

### Subject properties available in rules

```javascript
subject.user          // username (string)
subject.groups        // array of group names (NOT available — use isInGroup)
subject.isInGroup("wheel")  // group membership check (boolean)
subject.local         // is session local? (boolean)
subject.active        // is session active? (boolean)
subject.session       // session ID (string)
subject.pid           // process ID (number)
```

## Polkit with systemd

### Authorize systemd operations

```javascript
// Allow devops group to restart/stop/start specific units
// /etc/polkit-1/rules.d/25-systemd-devops.rules
polkit.addRule(function(action, subject) {
    if (action.id == "org.freedesktop.systemd1.manage-units" &&
        subject.isInGroup("devops") &&
        (action.lookup("unit") == "nginx.service" ||
         action.lookup("unit") == "postgresql.service" ||
         action.lookup("unit") == "redis.service")) {
        return polkit.Result.YES;
    }
});
```

### Machine-wide power management

```javascript
// /etc/polkit-1/rules.d/35-power.rules
polkit.addRule(function(action, subject) {
    // Allow all local active users to suspend/hibernate
    if ((action.id == "org.freedesktop.login1.suspend" ||
         action.id == "org.freedesktop.login1.hibernate") &&
        subject.local && subject.active) {
        return polkit.Result.YES;
    }

    // Require admin auth for reboot/poweroff
    if ((action.id == "org.freedesktop.login1.reboot" ||
         action.id == "org.freedesktop.login1.power-off") &&
        subject.local && subject.active) {
        return polkit.Result.AUTH_ADMIN_KEEP;
    }
});
```

## Authentication Agents

### Available agents

```bash
# Desktop agents (GUI password dialog)
/usr/lib/polkit-gnome/polkit-gnome-authentication-agent-1   # GNOME
/usr/lib/polkit-kde-authentication-agent-1                  # KDE

# TTY agent (for headless/SSH sessions)
pkttyagent --process $$                      # register TTY agent for current session
pkttyagent --process $$ --notify-fd 3        # with notification FD
```

### Using pkttyagent for SSH sessions

```bash
# In SSH session, polkit has no GUI agent
# Register TTY agent first:
pkttyagent --process $$ &
pkexec /usr/bin/systemctl restart nginx
# Agent prompts on TTY for password
```

## D-Bus Authorization

### polkit protects D-Bus system services

```bash
# Check D-Bus bus policy
cat /usr/share/dbus-1/system.d/org.freedesktop.UDisks2.conf
# D-Bus allows the method call, but the service checks polkit authorization

# Monitor polkit D-Bus interface
dbus-monitor --system "interface='org.freedesktop.PolicyKit1.Authority'"

# Introspect polkit authority
busctl introspect org.freedesktop.PolicyKit1 /org/freedesktop/PolicyKit1/Authority
```

## Verification and Debugging

### Check polkit daemon status

```bash
systemctl status polkit
journalctl -u polkit -f                      # follow polkit logs
```

### Test authorization

```bash
# Check if current user can perform action
pkcheck --action-id org.freedesktop.systemd1.manage-units \
  --process $$ --allow-user-interaction && echo "Authorized" || echo "Denied"

# Check for specific user (as root)
pkcheck --action-id org.freedesktop.udisks2.filesystem-mount \
  --process $$ --user webadmin
```

### Debug rules evaluation

```bash
# 1. Add logging rule (see 00-log-all.rules above)
# 2. Watch journal
sudo journalctl -u polkit -f

# 3. Trigger an action in another terminal
pkexec /bin/true

# 4. Check log for evaluation trace
```

### List installed policy files

```bash
ls /usr/share/polkit-1/actions/              # .policy XML files
ls /etc/polkit-1/rules.d/                    # local .rules files
ls /usr/share/polkit-1/rules.d/              # vendor .rules files
```

### Common issues

```bash
# "Not authorized" / no password prompt in SSH
# → No authentication agent running in SSH session
pkttyagent --process $$ &                    # register TTY agent

# Rules file syntax error (polkit ignores the file silently)
sudo journalctl -u polkit | grep -i error    # check for JS parse errors
# Validate JS syntax manually:
node -c /etc/polkit-1/rules.d/10-custom.rules  # if node.js installed

# polkitd not running
sudo systemctl start polkit
sudo systemctl enable polkit

# Rule not taking effect
# Check filename ordering — earlier files win
ls -la /etc/polkit-1/rules.d/ /usr/share/polkit-1/rules.d/
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
- man polkit(8), man polkitd(8), man pkexec(1), man pkaction(1), man pkcheck(1)
- man pkttyagent(1)
- freedesktop.org: PolicyKit Specification
- Red Hat: Using PolicyKit for Access Control (RHEL 9)
