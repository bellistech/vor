# EEM (Embedded Event Manager)

Cisco IOS/IOS-XE subsystem that monitors events and executes automated actions directly on the router.

## EEM Architecture Overview

### Core Components

```
Event Detectors --> Policy Engine --> Actions
   (triggers)      (match+decide)    (execute)

Event Detectors:
  - Syslog, CLI, Interface, Timer, Track, SNMP, OIR, IP SLA
  - Each detector monitors a specific subsystem for state changes

Policy Engine:
  - Matches detected events against registered policies
  - Evaluates environment variables and conditions
  - Invokes action handlers in sequence

Action System:
  - CLI commands, syslog messages, mail, reload, counter, track
  - Actions execute in order within each policy
```

### Event Detector Types

| Detector | Monitors | Trigger Example |
|:---|:---|:---|
| `syslog` | Syslog messages | Pattern match on log output |
| `cli` | CLI command execution | User runs `show` or `clear` |
| `interface` | Interface counters | Input errors exceed threshold |
| `timer` | Time-based events | Cron, countdown, watchdog |
| `track` | Object tracking | Tracked object state changes |
| `snmp` | SNMP OID values | OID crosses threshold |
| `oir` | Online insertion/removal | Module inserted or removed |
| `ipsla` | IP SLA probes | Probe timeout or threshold |
| `resource` | System resources | CPU or memory threshold |
| `rf` | Redundancy framework | Switchover events |
| `routing` | Routing table | Route appears or disappears |
| `counter` | Named counters | Counter reaches threshold |
| `none` | Manual trigger only | `event manager run` command |

## Applet Policies

### Basic Applet Structure

```
event manager applet <NAME> [authorization bypass]
 event <detector> <parameters>
 action <label> <action-type> <arguments>
```

### Minimal Applet (Syslog Trigger)

```
event manager applet LINK_DOWN
 event syslog pattern "LINK-3-UPDOWN.*GigabitEthernet0/1.*down"
 action 1.0 syslog msg "EEM: Gi0/1 went down — auto-remediation starting"
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "interface GigabitEthernet0/1"
 action 5.0 cli command "shutdown"
 action 6.0 cli command "no shutdown"
 action 7.0 cli command "end"
 action 8.0 syslog msg "EEM: Gi0/1 bounce complete"
```

### Action Labels and Ordering

```
! Labels are sorted alphanumerically — they control execution order
! Use decimal notation for clarity:
 action 1.0 ...    ! first
 action 1.5 ...    ! between 1 and 2
 action 2.0 ...    ! second
 action 10.0 ...   ! after 9.x (string sort: "10" > "9" is false!)

! Safe pattern: use zero-padded labels
 action 010 ...
 action 020 ...
 action 100 ...
```

## Event Detectors — Detailed Configuration

### Syslog Event Detector

```
! Trigger on any syslog message matching a regex
event manager applet CATCH_OSPF_ADJ
 event syslog pattern "OSPF-5-ADJCHG.*FULL"
 action 1.0 syslog msg "EEM: OSPF adjacency reached FULL state"

! Trigger on severity level
event manager applet CRITICAL_LOGS
 event syslog pattern ".*" severity 2
 action 1.0 mail server "10.0.0.25" to "noc@example.com" from "router@example.com" subject "CRITICAL SYSLOG" body "Critical log detected on $_event_pub_time"
```

### CLI Event Detector

```
! Trigger when a user executes a specific command
event manager applet AUDIT_CLEAR_COUNTERS
 event cli pattern "clear counters" sync yes skip no
 action 1.0 syslog msg "EEM: User $_cli_user executed: clear counters"

! sync yes  — EEM runs BEFORE the command executes
! sync no   — EEM runs AFTER the command executes
! skip yes  — suppress the original command (intercept)
! skip no   — allow the original command to proceed
```

### Interface Event Detector

```
! Trigger on interface counter thresholds
event manager applet HIGH_INPUT_ERRORS
 event interface name GigabitEthernet0/0 parameter input_errors entry-op ge entry-val 1000 poll-interval 60
 action 1.0 syslog msg "EEM: Gi0/0 input errors exceeded 1000"
 action 2.0 cli command "enable"
 action 3.0 cli command "show interface GigabitEthernet0/0 | include errors"

! Parameters: input_errors, output_errors, input_drops, output_drops,
!             rxload, txload, receive_rate, transmit_rate
! entry-op: ge, gt, le, lt, eq, ne
! exit-op/exit-val: optional — defines when the event clears
```

### Timer Event Detectors

```
! Countdown timer — fires once after delay
event manager applet DELAYED_START
 event timer countdown time 300 name BOOT_DELAY
 action 1.0 syslog msg "EEM: 5-minute boot delay expired"

! Watchdog timer — fires repeatedly at interval
event manager applet PERIODIC_CHECK
 event timer watchdog time 3600 name HOURLY
 action 1.0 cli command "enable"
 action 2.0 cli command "show ip route summary"
 action 3.0 syslog msg "EEM: Hourly routing check completed"

! Cron timer — fires on cron schedule
event manager applet NIGHTLY_BACKUP
 event timer cron cron-entry "0 2 * * *" name BACKUP_2AM
 action 1.0 cli command "enable"
 action 2.0 cli command "copy running-config tftp://10.0.0.5/backups/$_event_pub_time-config"
 action 3.0 syslog msg "EEM: Nightly config backup completed"

! Absolute timer — fires at a specific date/time
event manager applet MAINTENANCE_WINDOW
 event timer absolute time "22:00:00 Jan 15 2026" name MAINT
 action 1.0 syslog msg "EEM: Maintenance window starting"
```

### Track Event Detector

```
! Trigger when a tracked object changes state
event manager applet TRACK_DOWN
 event track 10 state down
 action 1.0 syslog msg "EEM: Tracked object 10 went DOWN"
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "ip route 0.0.0.0 0.0.0.0 10.0.0.2"
 action 5.0 cli command "end"

! Combined with IP SLA tracking
track 10 ip sla 1 reachability
ip sla 1
 icmp-echo 8.8.8.8 source-interface GigabitEthernet0/0
 frequency 30
ip sla schedule 1 start-time now life forever
```

### SNMP Event Detector

```
! Trigger on SNMP OID threshold
event manager applet CPU_HIGH
 event snmp oid 1.3.6.1.4.1.9.2.1.58.0 get-type exact entry-op ge entry-val "75" poll-interval 60
 action 1.0 syslog msg "EEM: CPU exceeds 75%"
 action 2.0 cli command "enable"
 action 3.0 cli command "show processes cpu sorted | head 10"

! Common OIDs:
! 1.3.6.1.4.1.9.2.1.58.0  — avgBusy5 (5-min CPU)
! 1.3.6.1.4.1.9.9.48.1.1.1.6.1 — ciscoMemoryPoolFree (processor)
! 1.3.6.1.4.1.9.9.48.1.1.1.6.2 — ciscoMemoryPoolFree (I/O)
```

### OIR (Online Insertion and Removal) Event Detector

```
event manager applet MODULE_INSERT
 event oir
 action 1.0 syslog msg "EEM: Hardware OIR event detected"
 action 2.0 cli command "enable"
 action 3.0 cli command "show module"
```

### None (Manual) Event Detector

```
event manager applet MANUAL_TASK
 event none
 action 1.0 syslog msg "EEM: Manual task started"
 action 2.0 cli command "enable"
 action 3.0 cli command "show running-config"

! Execute manually:
event manager run MANUAL_TASK
```

### Resource Event Detector

```
event manager applet LOW_MEMORY
 event resource policy mem-limit direction decrease level critical
 action 1.0 syslog msg "EEM: Memory critically low"
 action 2.0 cli command "enable"
 action 3.0 cli command "show memory statistics"
```

## Action Types

### CLI Actions (Most Common)

```
! Full CLI interaction pattern
 action 1.0 cli command "enable"
 action 2.0 cli command "configure terminal"
 action 3.0 cli command "interface loopback99"
 action 4.0 cli command "ip address 99.99.99.99 255.255.255.255"
 action 5.0 cli command "end"
 action 6.0 cli command "write memory"
```

### Syslog Actions

```
! Generate syslog messages with priority
 action 1.0 syslog msg "EEM: Custom message here" priority informational
 action 2.0 syslog msg "EEM ALERT: Critical event" priority critical

! Facility options: informational, warnings, errors, critical, alerts, emergencies
```

### Mail Actions

```
 action 1.0 mail server "10.0.0.25" to "noc@example.com" from "router@example.com" subject "Alert from Router" body "Interface went down at $_event_pub_time"
```

### Reload Actions

```
 action 1.0 reload
```

### Counter Actions

```
 action 1.0 counter name "link_flaps" value 1 op inc
 action 2.0 counter name "link_flaps" value 0 op nop

! Use counter value in conditions
```

### Track Actions

```
! Set or clear a track object
 action 1.0 track set 20
 action 2.0 track read 20
 action 3.0 track clear 20
```

### Info Actions (Capture Output)

```
 action 1.0 info type snmp oid 1.3.6.1.4.1.9.2.1.58.0 get-type exact
 action 2.0 info type routername
 action 3.0 syslog msg "Router: $_info_routername"
```

### If/Else Logic in Applets

```
event manager applet CONDITIONAL_CHECK
 event timer watchdog time 300
 action 1.0 cli command "enable"
 action 2.0 cli command "show ip sla statistics 1 | include Return code"
 action 3.0 regexp "Return code: OK" "$_cli_result"
 action 4.0 if $_regexp_result eq 1
 action 4.1  syslog msg "EEM: SLA probe OK"
 action 5.0 else
 action 5.1  syslog msg "EEM: SLA probe FAILED — activating backup route"
 action 5.2  cli command "configure terminal"
 action 5.3  cli command "ip route 0.0.0.0 0.0.0.0 10.0.0.2"
 action 5.4  cli command "end"
 action 6.0 end
```

### String and Regex Operations

```
 action 1.0 cli command "enable"
 action 2.0 cli command "show version | include uptime"
 action 3.0 regexp "uptime is (.+)" "$_cli_result" match uptime_str
 action 4.0 syslog msg "EEM: Device uptime: $uptime_str"

! String manipulation
 action 5.0 string trim "$_cli_result"
 action 6.0 string length "$_cli_result"
```

### Increment and Math Operations

```
 action 1.0 set count "0"
 action 2.0 increment count 1
 action 3.0 syslog msg "Count is now: $count"
```

## Environment Variables

### Built-in Variables

```
$_event_pub_time       — timestamp of event
$_event_pub_sec        — seconds since epoch
$_event_id             — unique event ID
$_event_type           — event detector type
$_event_type_string    — human-readable event type
$_cli_result           — output of last CLI action
$_cli_user             — user who triggered CLI event
$_info_routername      — router hostname
$_regexp_result        — 1 if last regexp matched, 0 otherwise
$_exit_status          — exit status of last action
$_syslog_msg           — full syslog message that triggered event
$_counter_value_remain — remaining counter value
$_track_state          — current track object state
```

### User-Defined Environment Variables

```
! Set global EEM variables
event manager environment _email_to noc@example.com
event manager environment _email_from router@example.com
event manager environment _tftp_server 10.0.0.5
event manager environment _snmp_community public123
event manager environment _threshold_cpu 80
event manager environment _backup_path backups/

! Use in applets
event manager applet USE_ENV_VARS
 event none
 action 1.0 mail server "10.0.0.25" to "$_email_to" from "$_email_from" subject "Alert" body "Test"
 action 2.0 cli command "enable"
 action 3.0 cli command "copy running-config tftp://$_tftp_server/$_backup_path$_info_routername.cfg"

! View configured variables
show event manager environment
```

## EEM with IP SLA

### Failover on SLA Failure

```
! Define the SLA probe
ip sla 1
 icmp-echo 8.8.8.8 source-interface GigabitEthernet0/0
 threshold 500
 timeout 1000
 frequency 10
ip sla schedule 1 start-time now life forever

! Track SLA reachability
track 10 ip sla 1 reachability

! EEM reacts to track state change
event manager applet FAILOVER_PRIMARY
 event track 10 state down
 action 1.0 syslog msg "EEM: Primary path down — switching to backup"
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "no ip route 0.0.0.0 0.0.0.0 10.1.1.1"
 action 5.0 cli command "ip route 0.0.0.0 0.0.0.0 10.2.2.1"
 action 6.0 cli command "end"

event manager applet RESTORE_PRIMARY
 event track 10 state up
 action 1.0 syslog msg "EEM: Primary path restored — reverting"
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "no ip route 0.0.0.0 0.0.0.0 10.2.2.1"
 action 5.0 cli command "ip route 0.0.0.0 0.0.0.0 10.1.1.1"
 action 6.0 cli command "end"
```

### SLA Jitter Monitoring

```
ip sla 2
 udp-jitter 10.0.0.50 16384 codec g729a
 frequency 30
ip sla schedule 2 start-time now life forever

track 20 ip sla 2 state

event manager applet JITTER_ALERT
 event track 20 state down
 action 1.0 syslog msg "EEM: Voice SLA jitter probe failing"
 action 2.0 mail server "10.0.0.25" to "$_email_to" from "$_email_from" subject "Jitter Alert" body "Voice path degraded at $_event_pub_time"
```

## Tcl Policies

### Basic Tcl Policy

```tcl
# File: flash:/eem/check_bgp.tcl
::cisco::eem::event_register_timer watchdog time 600

namespace import ::cisco::eem::*
namespace import ::cisco::lib::*

proc check_bgp {} {
    set result [cli_open]
    set fd [lindex $result 1]

    cli_exec $fd "enable"
    set output [cli_exec $fd "show ip bgp summary | include Active"]

    if {[string length $output] > 0} {
        action_syslog msg "EEM-TCL: BGP neighbors in Active state detected"
        action_syslog msg "EEM-TCL: $output"
    }

    cli_close $fd $result
}

check_bgp
```

### Register Tcl Policy

```
! Copy script to flash
copy tftp://10.0.0.5/eem/check_bgp.tcl flash:/eem/

! Configure policy directory and register
event manager directory user policy "flash:/eem/"
event manager policy check_bgp.tcl type user

! Verify
show event manager policy registered
```

### Tcl Policy with SMTP

```tcl
::cisco::eem::event_register_syslog pattern "LINK-3-UPDOWN" maxrun 120

namespace import ::cisco::eem::*
namespace import ::cisco::lib::*

array set event_info [event_reqinfo]
set syslog_msg $event_info(msg)

set result [cli_open]
set fd [lindex $result 1]

cli_exec $fd "enable"
set hostname [cli_exec $fd "show run | include hostname"]
set iface_status [cli_exec $fd "show ip interface brief"]

cli_close $fd $result

set body "Host: $hostname\n\nEvent: $syslog_msg\n\nInterface Status:\n$iface_status"

action_mail -to "noc@example.com" -from "eem@example.com" \
    -server "10.0.0.25" -subject "Link Down Alert" -body $body
```

## Common Use Cases

### Auto-Recover Downed Interface

```
event manager applet AUTO_RECOVER_INTF
 event syslog pattern "LINK-3-UPDOWN.*GigabitEthernet0/1.*down"
 action 1.0 wait 10
 action 2.0 cli command "enable"
 action 3.0 cli command "configure terminal"
 action 4.0 cli command "interface GigabitEthernet0/1"
 action 5.0 cli command "shutdown"
 action 6.0 wait 5
 action 7.0 cli command "no shutdown"
 action 8.0 cli command "end"
 action 9.0 syslog msg "EEM: Gi0/1 auto-recovery bounce completed"
```

### Scheduled Config Backup

```
event manager applet CONFIG_BACKUP_DAILY
 event timer cron cron-entry "0 3 * * *" name DAILY_BACKUP
 action 1.0 cli command "enable"
 action 1.5 info type routername
 action 2.0 cli command "copy running-config tftp://10.0.0.5/backups/$_info_routername-$_event_pub_sec.cfg"
 action 3.0 syslog msg "EEM: Daily backup saved for $_info_routername"
```

### Alert on Repeated Authentication Failures

```
event manager applet AUTH_FAIL_ALERT
 event syslog pattern "SEC_LOGIN-5-LOGIN_FAILED" occurs 5 period 300
 action 1.0 syslog msg "EEM: 5 login failures in 5 minutes — possible brute force"
 action 2.0 mail server "10.0.0.25" to "security@example.com" from "router@example.com" subject "Auth Alert" body "Multiple login failures detected at $_event_pub_time"
```

### Config Change Notification

```
event manager applet CONFIG_CHANGE_ALERT
 event syslog pattern "SYS-5-CONFIG_I"
 action 1.0 cli command "enable"
 action 2.0 cli command "show archive config differences"
 action 3.0 mail server "10.0.0.25" to "noc@example.com" from "router@example.com" subject "Config Changed" body "$_cli_result"
```

### Prevent Specific Commands

```
event manager applet BLOCK_DEBUG_ALL
 event cli pattern "debug all" sync yes skip yes
 action 1.0 syslog msg "EEM: Blocked 'debug all' by user $_cli_user"
 action 2.0 puts "*** Command blocked by policy: debug all is not permitted ***"
```

### BGP Neighbor Down Response

```
event manager applet BGP_NEIGHBOR_DOWN
 event syslog pattern "BGP-5-ADJCHANGE.*Down"
 action 1.0 syslog msg "EEM: BGP adjacency lost"
 action 2.0 cli command "enable"
 action 3.0 cli command "show ip bgp summary"
 action 4.0 cli command "show logging last 20"
 action 5.0 mail server "10.0.0.25" to "noc@example.com" from "router@example.com" subject "BGP Neighbor Down" body "$_cli_result"
```

### HSRP State Change Notification

```
event manager applet HSRP_STATE_CHANGE
 event syslog pattern "STANDBY-6-STATECHANGE"
 action 1.0 syslog msg "EEM: HSRP state change detected"
 action 2.0 cli command "enable"
 action 3.0 cli command "show standby brief"
 action 4.0 syslog msg "EEM: HSRP status: $_cli_result"
```

### Memory Leak Detection

```
event manager applet MEMORY_MONITOR
 event timer watchdog time 1800 name MEM_CHECK
 action 1.0 cli command "enable"
 action 2.0 cli command "show memory statistics | include Processor"
 action 3.0 regexp "Processor +([0-9]+) +([0-9]+) +([0-9]+)" "$_cli_result" match total used free
 action 4.0 set pct_used "0"
 action 5.0 multiply $used 100
 action 5.5 divide $_ $total
 action 6.0 set pct_used "$_"
 action 7.0 if $pct_used ge 85
 action 7.1  syslog msg "EEM: Memory usage at $pct_used% — investigate leak"
 action 7.2  cli command "show processes memory sorted | head 15"
 action 8.0 end
```

## EEM Verification Commands

```
! Show all registered applets
show event manager policy registered

! Show EEM event history
show event manager history

! Show environment variables
show event manager environment

! Show detector statistics
show event manager statistics

! Show available event detectors
show event manager detector all

! Show EEM session information
show event manager session all

! Debug EEM
debug event manager action cli
debug event manager detector all
```

## Tips

- Always start CLI actions with `action X.X cli command "enable"` to enter privileged exec
- Use `maxrun` parameter on event lines to limit execution time: `event syslog pattern "..." maxrun 120`
- Use `occurs N period S` on syslog events to debounce (N occurrences within S seconds)
- Set `authorization bypass` on the applet line to skip AAA authorization for EEM CLI actions
- Action labels are sorted as strings: `2` comes after `19` (use `02` and `19` or decimals)
- Use `$_cli_result` to capture the output of the previous CLI action
- Use `wait <seconds>` action to introduce delays between actions
- Tcl policies offer full programming logic; applets are limited to linear execution with basic conditionals
- Test applets with `event none` first, then switch to the real event detector
- Limit mail actions to critical events to avoid flooding the mailbox

## See Also

- IP SLA
- Track Objects
- Cisco IOS XR
- SNMP

## References

- Cisco EEM Configuration Guide — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/configuration/guide.html
- Cisco EEM Command Reference — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/command/eem-cr-book.html
- Cisco EEM Tcl Command Extension Reference — https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/eem/configuration/guide/eem-tcl-extensions.html
- RFC 3877 — Alarm Management MIB (relevant to SNMP event detectors)
