# EEM -- Embedded Event Manager Architecture and Automation

> *EEM is a policy-driven event subsystem embedded in Cisco IOS that decouples detection from action through a pub-sub event bus, enabling autonomous router-local remediation without external orchestration. Its architecture mirrors classical event-driven systems: detectors publish, the policy engine subscribes and dispatches, and action handlers execute — all within a single control-plane process.*

---

## 1. Event-Driven Architecture (The Core Model)

### The Problem

Network devices generate thousands of state changes per minute — syslog messages, counter increments, protocol transitions, hardware insertions. Without a local event-processing engine, every reaction requires an external system to poll, detect, and push corrective configuration.

### The Architecture

EEM implements a three-stage pipeline:

```
+-----------------+     +----------------+     +----------------+
| Event Detectors |---->| Policy Engine  |---->| Action Handlers|
| (Publishers)    |     | (Dispatcher)   |     | (Executors)    |
+-----------------+     +----------------+     +----------------+
       |                       |                       |
  Syslog, CLI,          Match event to         CLI, syslog,
  Timer, SNMP,          registered policy,     mail, reload,
  Track, OIR,           evaluate guards,       counter, track,
  Interface, SLA        check env vars         info, set, if/else
```

Each event detector operates as an independent monitor. When its condition triggers, it publishes an event record containing:

- **Event type** — which detector fired
- **Timestamp** — `$_event_pub_time` and `$_event_pub_sec`
- **Event ID** — unique identifier for correlation
- **Detector-specific data** — syslog message text, CLI command string, interface name, OID value, track state

The policy engine maintains a registry of applet and Tcl policies. Each policy declares exactly one event subscription (the `event` line in an applet). When a published event matches a subscription, the engine invokes that policy's action sequence.

### Event Flow (Synchronous vs. Asynchronous)

Two execution modes exist:

| Mode | Keyword | Behavior |
|:---|:---|:---|
| **Asynchronous** | `sync no` (default for most detectors) | Event is published, original operation continues, policy runs in parallel |
| **Synchronous** | `sync yes` | Original operation blocks until the policy completes; policy can suppress the operation via `skip yes` |

Synchronous mode is critical for the CLI detector: it allows EEM to intercept and block commands before they execute. This is the mechanism behind command authorization policies implemented purely on-box.

### Execution Isolation

Each applet runs in its own execution context with:

- A dedicated CLI session (virtual TTY)
- Its own variable namespace (no cross-applet variable leakage)
- A `maxrun` timer (default 20 seconds) that kills the applet if exceeded
- Exit status tracking (`$_exit_status`)

---

## 2. Event Detectors (The Sensor Layer)

### The Problem

Different subsystems expose state through different interfaces — syslog is text, SNMP is OID-value pairs, interface counters are integers, tracking objects are boolean. EEM needs a uniform abstraction over these heterogeneous sources.

### Detector Taxonomy

EEM ships approximately 20 event detectors. The most operationally significant:

**Syslog Detector** — The most commonly used detector. It applies regex matching against the syslog buffer in real-time. Key parameters:

- `pattern` — POSIX extended regex matched against each syslog line
- `occurs N period S` — debouncing: require N matches within S seconds before triggering
- `severity` — filter by syslog severity level (0-7)

The detector operates at the syslog subsystem level, not at the console output level. Messages suppressed by `no logging console` still reach the detector if they enter the syslog buffer.

**CLI Detector** — Intercepts IOS CLI commands at the parser level. The command string is available in `$_cli_msg`. Combined with `sync yes` and `skip yes`, this detector can:

1. Audit all privileged commands to syslog or SNMP
2. Block dangerous commands (`debug all`, `write erase`)
3. Modify command behavior (run additional commands before/after)
4. Implement time-of-day command restrictions

**Timer Detector** — Four variants:

| Timer Type | Behavior | Use Case |
|:---|:---|:---|
| `countdown` | Fires once after N seconds | Post-boot initialization |
| `watchdog` | Fires every N seconds, repeating | Periodic health checks |
| `cron` | Fires on cron schedule (5-field) | Scheduled maintenance tasks |
| `absolute` | Fires at a specific date/time | One-time future events |

The cron timer uses standard 5-field cron syntax (`minute hour day month weekday`). The timer subsystem survives process restarts but not device reloads — countdown and absolute timers reset on boot.

**Interface Detector** — Polls interface counters at configurable intervals. Supports threshold-based triggering with entry/exit conditions:

```
Entry condition:  counter >= entry-val    (alarm raised)
Exit condition:   counter <= exit-val     (alarm cleared)
```

This implements hysteresis — the event fires when entering the alarm state and optionally when exiting. Without exit conditions, the detector fires on every poll where the entry condition is true.

**Track Detector** — Subscribes to IOS track objects. Track objects abstract reachability probes (IP SLA), route existence, interface state, and boolean combinations. The track detector fires on state transitions (up-to-down or down-to-up), providing a clean integration point between IP SLA monitoring and EEM automation.

**SNMP Detector** — Polls any SNMP OID at configurable intervals and applies threshold logic. This enables monitoring of MIB values that have no native syslog or track integration — for example, custom MIBs from third-party modules.

**OIR Detector** — Fires on hardware insertion/removal events. On modular platforms (Catalyst 6500, ASR 1000), this detects line card insertions, power supply changes, and fan tray events.

---

## 3. Policy Types (Applets vs. Tcl)

### The Problem

Simple event-action mappings (syslog pattern triggers CLI commands) are the most common EEM use case, but some scenarios require full programming logic — loops, data structures, socket I/O, file manipulation. EEM must support both without forcing complexity on simple cases.

### Applet Policies

Applets are configured inline in the running configuration. Their structure is:

```
event manager applet <NAME>
 event <detector> <parameters>
 action <label> <action-type> <arguments>
 action <label> <action-type> <arguments>
 ...
```

**Strengths:**

- No external files needed — lives in running-config
- Survives `copy run start` — backed up with configuration
- Simple to audit — `show running | section event manager`
- Supports basic conditionals (`if/else/end`), regex, string operations, math

**Limitations:**

- No loops (no `for`, `while`, `foreach`)
- No data structures beyond scalar variables
- No external I/O beyond CLI, syslog, mail, SNMP
- No subroutines or function calls
- Limited error handling

### Label Sorting (A Critical Detail)

Action labels are sorted **lexicographically** (string sort), not numerically. This means:

```
Intended order:  1, 2, 3, ..., 9, 10, 11
Actual order:    1, 10, 11, 2, 3, ..., 9
```

Safe labeling strategies:

| Strategy | Example | Notes |
|:---|:---|:---|
| Zero-padded integers | `001`, `002`, ... `010` | Works up to 999 actions |
| Decimal notation | `1.0`, `1.5`, `2.0` | Allows insertion between existing labels |
| Hierarchical | `1.0` parent, `1.1` child in if-block | Visually groups logical blocks |

### Tcl Policies

Tcl policies are external scripts stored on the device filesystem (flash:, disk0:, bootflash:). They offer:

- Full Tcl 8.x language features (loops, lists, arrays, procedures)
- Socket I/O for HTTP, SMTP, custom protocols
- File I/O for reading/writing flash files
- Complex string processing and parsing
- Multi-step decision trees

The Tcl EEM API is exposed through two namespaces:

| Namespace | Purpose |
|:---|:---|
| `::cisco::eem::*` | Event registration, action execution, event info retrieval |
| `::cisco::lib::*` | CLI session management, output parsing |

Key Tcl API functions:

```
event_reqinfo          — returns event details as array
cli_open               — opens a CLI session, returns session descriptor
cli_exec $fd "cmd"     — executes a CLI command, returns output
cli_close $fd          — closes the CLI session
action_syslog          — generates a syslog message
action_mail            — sends an email
action_track           — manipulates track objects
```

### Registration and Lifecycle

Tcl policies must be explicitly registered:

1. Place the `.tcl` file in a configured policy directory
2. Register: `event manager policy <name>.tcl type user`
3. The policy engine compiles and validates the event registration line
4. On event match, the Tcl interpreter executes the script

Tcl policies are **not** part of the running configuration — they must be managed as separate files. This creates a backup/restore consideration: `copy running-config` does not capture Tcl policy content, only the registration reference.

---

## 4. The Action System (Execution Engine)

### The Problem

Once an event triggers a policy, the system must execute a sequence of actions atomically, handling failures, capturing output, and managing side effects. The action system must be both powerful enough for complex remediation and safe enough to run unsupervised on production routers.

### CLI Action Internals

The `cli` action type creates a virtual TTY session on the router. This session behaves identically to a console or VTY session:

1. Starts in user EXEC mode
2. Requires `enable` to reach privileged EXEC
3. Requires `configure terminal` to reach global config
4. Supports all IOS commands available to the privilege level
5. Output of each command is captured in `$_cli_result`

The virtual TTY session persists across all CLI actions within a single applet execution. This means:

- Mode transitions persist: `enable` in action 1 means action 2 is already in privileged mode
- Interface context persists: `interface Gi0/1` in action 3 means action 4 is in interface config mode
- The session is destroyed when the applet completes

### Authorization Bypass

By default, EEM CLI actions are subject to the same AAA authorization as interactive users. The `authorization bypass` keyword on the applet declaration exempts the applet from AAA:

```
event manager applet CRITICAL_FIX authorization bypass
```

This is essential for remediation applets that must function even when the AAA server is unreachable — a common failure scenario when the network itself is degraded.

### Output Capture and Parsing

The `$_cli_result` variable contains the complete output of the most recent CLI action. Combined with `regexp` actions, this enables:

```
Pattern:  action N regexp "<regex>" "$_cli_result" match group1 group2
Result:   $_regexp_result = 1 (match) or 0 (no match)
          $match = full match
          $group1, $group2 = capture groups
```

This is the mechanism for conditional logic in applets — execute a show command, regex the output, branch on the result.

### Mail Action (SMTP Integration)

The mail action implements a minimal SMTP client directly in the IOS process:

- Connects to the specified SMTP server on port 25
- Sends a single message with configurable to, from, subject, body
- No authentication support (relies on SMTP relay configuration)
- No TLS/SSL support
- Body is limited to a single string (no multi-line without Tcl)

For production use, the SMTP server must be configured to accept unauthenticated relay from the router's IP address.

---

## 5. Environment Variables (Configuration Abstraction)

### The Problem

Hardcoding IP addresses, email addresses, thresholds, and paths in applets creates maintenance burden. Changing the TFTP server address would require editing every applet that references it.

### The Solution

EEM environment variables provide a global key-value store accessible to all policies:

```
event manager environment _var_name value
```

Convention: user-defined variables use underscore prefix (`_email_to`, `_tftp_server`) to distinguish from built-in variables (`$_cli_result`, `$_event_pub_time`).

### Variable Scope Hierarchy

| Scope | Lifetime | Access |
|:---|:---|:---|
| **Environment variables** | Persistent (saved in config) | All policies, read-only within policy |
| **Built-in variables** | Per-event | Current policy only |
| **Local variables** | Per-execution | Current applet run only (via `set`) |

### Built-in Variable Categories

**Event metadata:**
- `$_event_pub_time` — human-readable timestamp
- `$_event_pub_sec` — epoch seconds (useful for filenames)
- `$_event_id` — unique event identifier
- `$_event_type` — numeric detector type
- `$_event_type_string` — human-readable detector name

**CLI detector specific:**
- `$_cli_msg` — the CLI command that triggered the event
- `$_cli_user` — username who entered the command

**Syslog detector specific:**
- `$_syslog_msg` — the full syslog message text

**Action results:**
- `$_cli_result` — output of last `cli command` action
- `$_regexp_result` — 1 if last `regexp` matched, 0 otherwise
- `$_exit_status` — exit status of last action
- `$_info_routername` — device hostname (from `info type routername`)

---

## 6. EEM Scheduling and Timer Architecture

### The Problem

Network operations require both reactive automation (respond to events) and proactive automation (scheduled tasks). The timer subsystem must provide cron-like scheduling, one-shot delays, and periodic polling — all without an external scheduler.

### Timer Implementation

EEM timers are managed by a dedicated timer process within the IOS scheduler. Each registered timer creates a timer entry with:

- **Name** — unique identifier for the timer
- **Type** — countdown, watchdog, cron, absolute
- **Interval/schedule** — seconds (countdown/watchdog) or cron expression
- **Associated policy** — the applet or Tcl policy to invoke

**Watchdog Timer Drift:**

The watchdog timer is not a precise interval timer. The actual period is:

$$T_{actual} = T_{configured} + T_{execution} + T_{scheduling\_jitter}$$

Where $T_{execution}$ is the applet runtime and $T_{scheduling\_jitter}$ is the IOS process scheduler's granularity (typically 4-16ms on modern platforms). For a 60-second watchdog, expect actual intervals of 60-62 seconds depending on applet complexity.

**Cron Timer Precision:**

Cron timers fire at minute boundaries. The cron expression `0 2 * * *` fires at 02:00:00 plus scheduling jitter — typically within the first second of the minute. The 5-field format follows standard Unix conventions:

```
+---------- minute (0-59)
| +-------- hour (0-23)
| | +------ day of month (1-31)
| | | +---- month (1-12)
| | | | +-- day of week (0-7, 0 and 7 = Sunday)
| | | | |
* * * * *
```

**Countdown Timer and Boot Sequences:**

Countdown timers start their countdown from the moment of registration. On device boot, this means the timer starts when the configuration is parsed — typically 30-120 seconds after power-on, depending on POST and IOS load time. This makes countdown timers useful for post-boot initialization but imprecise for exact timing.

---

## 7. Bidirectional CLI Interaction

### The Problem

Some CLI commands prompt for user input (`Are you sure? [yes/no]`, `Destination filename?`). EEM applets must handle these prompts without human intervention.

### The Mechanism

EEM CLI actions can send responses to interactive prompts by issuing the response as a subsequent CLI command:

```
action 1.0 cli command "enable"
action 2.0 cli command "copy running-config tftp://10.0.0.5/backup.cfg"
action 3.0 cli command ""
```

Action 3.0 sends an empty string (Enter key) to accept the default filename prompt. For commands with explicit prompts:

```
action 1.0 cli command "enable"
action 2.0 cli command "write erase"
action 3.0 cli command ""
action 4.0 cli command "reload"
action 5.0 cli command "yes"
```

### Pattern Matching for Prompts

In Tcl policies, prompt handling is more sophisticated:

```tcl
set output [cli_exec $fd "copy running-config tftp://10.0.0.5/backup.cfg"]
if {[regexp {Destination filename} $output]} {
    cli_exec $fd ""
}
if {[regexp {confirm} $output]} {
    cli_exec $fd "yes"
}
```

### Timeout Handling

If an EEM CLI action encounters a prompt that is not answered, the action will block until the `maxrun` timer expires. The applet then terminates with an error status. To prevent this:

1. Always account for prompts in the action sequence
2. Set appropriate `maxrun` values (default 20 seconds is often too short for file operations)
3. Use `action N cli command "cmd" pattern "prompt_regex"` (available on some platforms) to wait for specific prompts

---

## 8. Auto-Remediation Patterns

### The Problem

The most valuable EEM use case is automated remediation — detecting a fault condition and applying a corrective action without human intervention. This requires careful design to avoid:

1. **Remediation loops** — the fix triggers the same event, causing infinite recursion
2. **Partial remediation** — the fix is applied but fails, leaving the system in a worse state
3. **False positives** — the event fires but the condition does not actually require remediation

### Pattern 1: Interface Bounce with Debounce

```
Event:   Interface flap (link-3-updown)
Guard:   occurs 1 period 30 (at most once per 30 seconds)
Action:  Wait, shutdown, wait, no shutdown
Safety:  maxrun prevents hang; debounce prevents loop
```

The `occurs 1 period 30` guard is critical — without it, the `shutdown`/`no shutdown` sequence generates its own syslog messages, which could re-trigger the applet.

### Pattern 2: Failover with State Tracking

```
Event:   Track object down (IP SLA probe failure)
Action:  Remove primary route, install backup route
Reverse: Track object up -> remove backup, restore primary
Safety:  Track object provides hysteresis; two applets ensure bidirectional handling
```

This pattern requires two applets — one for the down transition and one for the up transition. A single applet cannot handle both because each applet registers for exactly one event.

### Pattern 3: Resource Exhaustion Response

```
Event:   SNMP OID (CPU > 80%) polled every 60 seconds
Guard:   Entry/exit thresholds with hysteresis
Action:  Log top processes, generate alert, optionally clear specific caches
Safety:  Exit threshold prevents repeated alerts; action is diagnostic, not destructive
```

### Pattern 4: Security Response

```
Event:   Syslog pattern for repeated auth failures (occurs 5 period 300)
Action:  Log event, send alert, optionally apply ACL to block source
Safety:  occurs/period debounce; ACL application is additive (does not remove existing rules)
```

### Anti-Patterns to Avoid

| Anti-Pattern | Risk | Mitigation |
|:---|:---|:---|
| Reload action without guard | Boot loop if the condition persists | Add occurs/period; use track objects |
| Write erase in applet | Data loss on false positive | Never automate destructive operations |
| No maxrun | Applet hangs on unexpected prompt | Always set maxrun explicitly |
| Syslog trigger + syslog action on same facility | Infinite loop | Use different facility/severity or add occurs guard |
| CLI action without `enable` | Commands fail silently | Always start with `enable` |

---

## 9. EEM in Multi-Platform Context

### The Problem

EEM exists on IOS, IOS-XE, IOS-XR, and NX-OS, but with different capabilities and syntax on each platform.

### Platform Comparison

| Feature | IOS/IOS-XE | IOS-XR | NX-OS |
|:---|:---|:---|:---|
| Applet policies | Yes | Yes (different syntax) | Yes |
| Tcl policies | Yes | Yes | Yes |
| Python policies | No | No | Yes (NX-SDK) |
| CLI detector | Yes | Yes | Yes |
| Syslog detector | Yes | Yes | Yes |
| Interface detector | Yes | Limited | Yes |
| Timer detectors | All 4 types | All 4 types | All 4 types |
| SNMP detector | Yes | Yes | Yes |
| Track detector | Yes | Yes (different) | Yes |
| Config modes | Running config | Commit model | Running config |

### IOS-XR EEM Differences

On IOS-XR, EEM operates within the commit model:

```
! IOS-XR applet syntax
event manager applet EXAMPLE
 event syslog pattern "OSPF-5-ADJCHG"
 action 1 syslog msg "EEM: OSPF adjacency change"
 action 2 cli command "show ospf neighbor"
```

Key differences:

1. CLI actions execute in XR CLI context (not IOS CLI)
2. Configuration changes require `commit` after `configure`
3. The `admin` mode requires separate event registration
4. Process restart is available as an action (not available on IOS)

### NX-OS EEM Extensions

NX-OS adds:

- Python script policies (in addition to Tcl)
- `event sysmgr` detector for process lifecycle events
- `event module` for module online/offline
- `event fex` for Fabric Extender events
- Enhanced `action` types including `policy-default` and `event-default`

---

## 10. Operational Best Practices

### The Problem

EEM runs unsupervised on production infrastructure. Poorly designed policies can cause outages. A disciplined operational framework is essential.

### Development Lifecycle

1. **Design** — Define event, guard conditions, actions, and failure modes on paper
2. **Lab test** — Deploy with `event none` (manual trigger) in a lab environment
3. **Dry run** — Change to real event detector but make actions diagnostic only (syslog, no config changes)
4. **Staged deployment** — Deploy to a single production device with aggressive maxrun
5. **Fleet rollout** — Push to all devices via configuration management (Ansible/NAPALM)

### Monitoring EEM Health

```
show event manager policy registered     — verify policies are loaded
show event manager history               — check recent executions
show event manager statistics            — detector hit counts
show event manager session cli           — active CLI sessions (detect hangs)
```

### Common Failure Modes

| Symptom | Cause | Fix |
|:---|:---|:---|
| Applet never fires | Event pattern does not match | Test pattern with `show logging` output |
| Applet fires but CLI fails | Missing `enable` action | Add `action X cli command "enable"` |
| Applet hangs | Unexpected CLI prompt | Add prompt responses; increase maxrun |
| Applet fires repeatedly | No debounce on event | Add `occurs N period S` |
| Mail action fails | SMTP server unreachable or relay denied | Verify connectivity and relay config |
| Config changes lost | Missing `write memory` action | Add `copy run start` at end |
| Tcl policy not found | Policy directory not configured | Set `event manager directory user policy` |

### Security Considerations

EEM applets run with the privilege level of the event manager process (typically privilege 15). This means:

- Any applet can execute any CLI command
- The `authorization bypass` keyword exempts applets from AAA entirely
- Tcl policies can open network sockets and access flash filesystem
- A compromised applet has full control of the device

Mitigation:

- Restrict `event manager applet` configuration to specific users via privilege levels
- Audit all registered policies regularly
- Use AAA command accounting to log EEM CLI actions
- Store Tcl policies on read-only media where possible

---

## Prerequisites

- Familiarity with Cisco IOS CLI modes (user EXEC, privileged EXEC, global config, interface config)
- Understanding of syslog message format and severity levels
- Basic regex syntax (POSIX extended regular expressions)
- For Tcl policies: basic Tcl programming (variables, procedures, string operations)
- For IP SLA integration: understanding of IP SLA probe types and track objects

---

## References

- Cisco EEM Configuration Guide, IOS 15.x — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/configuration/guide.html
- Cisco EEM Tcl Command Extension Reference — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/configuration/guide/eem-tcl-extensions.html
- Cisco IOS-XR EEM Configuration Guide — https://www.cisco.com/c/en/us/td/docs/routers/asr9000/software/eem/configuration/guide.html
- Cisco NX-OS EEM Configuration Guide — https://www.cisco.com/c/en/us/td/docs/switches/datacenter/nexus9000/sw/eem/configuration/guide.html
- RFC 3877 — Alarm Management Information Base (MIB)
- Cisco IOS IP SLA Configuration Guide — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipsla/configuration/guide.html
