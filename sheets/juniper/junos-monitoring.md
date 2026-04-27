# Junos Monitoring & Troubleshooting (JNCIA-Junos)

Essential show, monitor, and diagnostic commands for day-to-day Junos operations and JNCIA exam prep.

## Show Commands — System

```bash
# System overview
show system uptime                     # uptime, load averages, user count
show system processes extensive        # per-process CPU/memory (like top)
show system storage                    # filesystem usage (/ /config /var)
show system alarms                     # active system alarms
show system boot-messages              # kernel boot log
show system connections                # active network sockets (like netstat)
show system statistics                 # protocol-level packet counters
show system users                      # currently logged-in users
show system license                    # installed feature licenses
show system core-dumps                 # list any crash core files
show system commit                     # commit history with timestamps/users
show system rollback compare N         # diff current config vs rollback N
show system snapshot media internal    # dual-root partition status
```

## Show Commands — Chassis

```bash
show chassis hardware                  # FPC, PIC, serial numbers, DRAM
show chassis environment               # temp sensors, fan status, power supplies
show chassis alarms                    # hardware alarms (major/minor)
show chassis routing-engine            # RE CPU/memory, uptime, mastership
show chassis fpc                       # line card status (Online/Offline)
show chassis fpc pic-status            # PIC state per FPC slot
show chassis firmware                  # firmware versions per component
```

## Show Commands — Interfaces

```bash
show interfaces terse                  # one-line-per-interface: admin/link/proto
show interfaces brief                  # interface list with description
show interfaces detail                 # full counters, MTU, speed, MAC
show interfaces extensive              # everything including error counters
show interfaces ge-0/0/0              # single interface detail
show interfaces ge-0/0/0.0            # logical unit (subinterface)
show interfaces descriptions           # admin descriptions only
show interfaces filters                # applied firewall filters
show interfaces statistics             # aggregate I/O statistics
show interfaces diagnostics optics     # optical power levels (SFP/XFP)
```

## Show Commands — Routing

```bash
show route                             # full RIB (all tables)
show route summary                     # route count per protocol/table
show route table inet.0                # IPv4 unicast table
show route table inet6.0               # IPv6 unicast table
show route 10.0.0.0/24                 # specific prefix lookup
show route 10.0.0.0/24 exact           # exact match only
show route 10.0.0.0/24 longer         # all more-specifics
show route protocol ospf               # OSPF-learned routes only
show route protocol bgp                # BGP-learned routes only
show route protocol static             # static routes only
show route protocol direct             # connected routes only
show route forwarding-table            # FIB (forwarding plane)
show route resolution unresolved       # next-hops that can't resolve
```

## Show Commands — OSPF

```bash
show ospf overview                     # router-id, areas, SPF count
show ospf neighbor                     # adjacency state (Full/2Way/Init)
show ospf neighbor detail              # DR/BDR, dead timer, options
show ospf interface                    # OSPF-enabled interfaces, area, cost
show ospf interface detail             # hello/dead timers, network type
show ospf database                     # full LSDB summary
show ospf database detail              # LSA details (LS age, seq, checksum)
show ospf database router              # router LSAs (type 1)
show ospf database network             # network LSAs (type 2)
show ospf database netsummary          # summary LSAs (type 3)
show ospf database external            # external LSAs (type 5)
show ospf route                        # OSPF SPF results
show ospf statistics                   # SPF run count, LSA counts
show ospf log                          # OSPF event log
```

## Show Commands — BGP

```bash
show bgp summary                       # peer state, prefixes, AS, uptime
show bgp neighbor                      # full peer detail
show bgp neighbor 10.0.0.1             # specific peer
show bgp neighbor 10.0.0.1 received-routes   # pre-policy received
show bgp neighbor 10.0.0.1 advertised-routes # what we advertise
show bgp group                         # peer group config/status
show route receive-protocol bgp 10.0.0.1    # accepted routes from peer
show route advertising-protocol bgp 10.0.0.1 # routes sent to peer
show policy                            # routing policy summary
```

## Show Commands — Configuration & Logs

```bash
show configuration                     # full active config
show configuration interfaces          # interfaces stanza
show configuration protocols           # protocols stanza
show configuration | display set       # set-command format
show configuration | compare rollback 1 # diff against rollback 1

show log messages                      # main syslog file
show log messages | last 50            # last 50 lines
show log messages | match error        # grep for "error"
show log messages | match "UI_COMMIT"  # commit events
show log chassisd                      # chassis daemon log
show log interactive-commands          # CLI command audit trail
```

## Show Commands — ARP, LLDP, NDP

```bash
show arp                               # ARP table (IPv4 → MAC)
show arp no-resolve                    # skip DNS reverse lookup
show ipv6 neighbors                    # IPv6 NDP table
show lldp                              # LLDP global status
show lldp neighbors                    # discovered LLDP peers
show lldp neighbors detail             # full TLV detail (system name, caps)
show lldp local-information            # what this device advertises
```

## Monitor Commands

```bash
# Real-time interface counters (refreshes every second)
monitor interface ge-0/0/0             # live I/O bytes/pps/errors
monitor interface traffic               # all interfaces, summary view

# Packet capture (tcpdump wrapper)
monitor traffic interface ge-0/0/0     # capture on interface
monitor traffic interface ge-0/0/0 no-resolve  # skip DNS
monitor traffic interface ge-0/0/0 size 1500   # full packet
monitor traffic interface ge-0/0/0 matching "tcp port 80"  # BPF filter
monitor traffic interface ge-0/0/0 write-file /var/tmp/cap.pcap  # save pcap

# Log monitoring (tail -f equivalent)
monitor start messages                 # stream syslog to terminal
monitor start <filename>               # stream any log file
monitor stop                           # stop all log streaming

# Combine filters
monitor start messages | match "OSPF|BGP"  # filter log stream
```

## Interface Statistics & Error Counters

```bash
show interfaces ge-0/0/0 extensive     # full counter dump
show interfaces ge-0/0/0 extensive | find "error"  # jump to errors
clear interfaces statistics ge-0/0/0   # zero counters (for delta testing)
```

| Counter              | Meaning                                    | Troubleshoot                          |
|----------------------|--------------------------------------------|---------------------------------------|
| Input bytes/packets  | Total received                             | Baseline for utilization              |
| Output bytes/packets | Total transmitted                          | Baseline for utilization              |
| Input errors         | Aggregate of all receive errors            | Drill into specific error type below  |
| Output errors        | Aggregate of all transmit errors           | Check duplex, cable, remote end       |
| CRC errors           | Frame checksum failed                      | Bad cable, SFP, EMI, duplex mismatch  |
| Framing errors       | Invalid frame delimiter                    | Layer 1 issue: cable, clocking        |
| Runts                | Frame smaller than 64 bytes                | Collision (half-duplex), bad NIC      |
| Giants               | Frame larger than max MTU                  | MTU mismatch, jumbo frame config      |
| Input drops          | Frames dropped (queue full, policer, etc.) | Check CoS queuing, interface speed    |
| Output drops         | TX ring full, congestion                   | Oversubscription, CoS shaping         |
| Carrier transitions  | Link flap count                            | Unstable cable/SFP, remote reboot     |
| Collisions           | Ethernet collision (half-duplex only)      | Set full-duplex, check auto-neg       |

## Network Diagnostic Tools

```bash
# Ping (ICMP echo)
ping 10.0.0.1                          # basic connectivity test
ping 10.0.0.1 count 5                  # send exactly 5 probes
ping 10.0.0.1 rapid                    # flood ping (100 pps)
ping 10.0.0.1 rapid count 1000        # 1000 rapid probes (loss test)
ping 10.0.0.1 size 1472               # test MTU (1472 + 28 = 1500)
ping 10.0.0.1 size 1472 do-not-fragment # MTU path discovery
ping 10.0.0.1 source 192.168.1.1      # set source address
ping 10.0.0.1 ttl 10                   # set TTL (loop detection)
ping 10.0.0.1 routing-instance VRF1   # ping within VRF

# Traceroute
traceroute 10.0.0.1                    # hop-by-hop path
traceroute 10.0.0.1 source 192.168.1.1 # set source
traceroute 10.0.0.1 gateway 10.0.0.254 # force first hop
traceroute 10.0.0.1 as-number-lookup   # show AS number per hop
traceroute 10.0.0.1 no-resolve         # skip DNS (faster)

# Remote access
telnet 10.0.0.1                        # telnet to remote host
telnet 10.0.0.1 port 8080             # test arbitrary TCP port
ssh 10.0.0.1                           # SSH to remote host
ssh 10.0.0.1 -l admin                 # SSH with username
```

## Logging & Syslog Configuration

```bash
# Configure syslog (edit system syslog)
set system syslog host 10.0.0.100 any warning        # remote syslog, warning+
set system syslog host 10.0.0.100 authorization info  # auth events to remote
set system syslog file messages any notice             # local file, notice+
set system syslog file messages authorization info     # auth to messages
set system syslog file interactive-commands interactive-commands any  # CLI audit
set system syslog file security-log authorization info # dedicated auth log
set system syslog time-format year millisecond         # add year + ms to stamps
set system syslog source-address 192.168.1.1           # syslog source IP

# Syslog severity levels (low → high)
# debug → info → notice → warning → error → critical → alert → emergency

# Log filtering in CLI
show log messages | match "error|warning"
show log messages | match "Mar  5"                     # date filter
show log messages | except "SNMP"                      # exclude pattern
show log messages | count                              # line count
show log messages | last 100 | match "kernel"          # combine filters
```

## Traceoptions (Protocol Debugging)

```bash
# OSPF tracing
set protocols ospf traceoptions file ospf-trace size 10m files 5
set protocols ospf traceoptions flag hello detail
set protocols ospf traceoptions flag spf
set protocols ospf traceoptions flag lsa-update

# BGP tracing
set protocols bgp traceoptions file bgp-trace size 10m files 5
set protocols bgp traceoptions flag open detail
set protocols bgp traceoptions flag update detail
set protocols bgp traceoptions flag keepalive

# View trace output
show log ospf-trace
show log ospf-trace | last 50
monitor start ospf-trace               # real-time tail

# IMPORTANT: disable tracing when done (CPU + disk impact)
deactivate protocols ospf traceoptions
```

## Request Commands

```bash
request system reboot                  # graceful reboot (prompts confirm)
request system reboot at 22:00         # scheduled reboot
request system reboot in 5             # reboot in 5 minutes
request system reboot message "maintenance window"  # notify users

request system halt                    # graceful shutdown (power off)

request system snapshot slice alternate # snapshot to backup partition
request system recover                 # recover from alternate slice

request system software add /var/tmp/junos-package.tgz  # install JunOS
request system software rollback       # rollback to previous JunOS version

request system configuration rescue save   # save rescue config
request system zeroize                 # factory reset (DESTRUCTIVE)
```

## Real-Time System Monitoring

```bash
# CPU and processes
show system processes extensive        # per-process CPU/mem (like top)
show system processes extensive | match "rpd|chassisd|mgd"  # key daemons

# Storage
show system storage                    # df -h equivalent
show system storage partitions         # detailed partition table
request system storage cleanup         # purge old logs, cores, tmps

# Uptime and load
show system uptime                     # uptime, users, load avg
show system statistics                 # global protocol counters

# Chassis health
show chassis environment               # temps, fans, PSUs, all sensors
show chassis alarms                    # active alarms (red/yellow)
show chassis routing-engine            # RE CPU, memory, mastership
show chassis temperature-thresholds    # alarm trigger temps

# Combine for quick health check
show system alarms
show chassis alarms
show chassis environment | match "status|Temp"
show system storage | match "Mounted|avail"
```

## Tips

- Use `| display xml` after any show command to get structured XML output for scripting.
- `show interfaces extensive` is the go-to for troubleshooting: it includes every counter.
- Carrier transitions > 0 on a "stable" link means it has flapped — investigate physical layer.
- CRC errors that keep incrementing point to bad cable/SFP, not software.
- `monitor traffic` is a live tcpdump — use `no-resolve` to avoid DNS delays.
- `monitor start/stop` is for streaming log files to your terminal, not packet capture.
- Always `deactivate` traceoptions after debugging; they consume CPU and fill disk.
- `request system snapshot` before any upgrade — your rollback safety net.
- `show route forwarding-table` shows what the PFE (forwarding plane) actually uses, not just the RE routing table.
- Use `clear interfaces statistics` before testing to get clean delta counters.

## See Also

- `sheets/juniper/junos-routing-fundamentals.md` — CLI navigation, modes, pipe commands
- `sheets/juniper/junos-routing-fundamentals.md` — OSPF, BGP, static route configuration
- `sheets/juniper/junos-firewall-filters.md` — Firewall filters and policers

## References

- Juniper JNCIA-Junos Study Guide (JN0-105) — Official Exam Objectives
- Juniper TechLibrary: CLI Reference — https://www.juniper.net/documentation/
- Juniper Day One: Monitoring and Troubleshooting — https://www.juniper.net/dayone
- RFC 3164 — BSD Syslog Protocol
- RFC 5424 — The Syslog Protocol
