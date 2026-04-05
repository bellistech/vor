# JunOS Screens (Attack Protection)

Screen profiles on SRX detect and drop malformed, suspicious, and attack packets before they reach the flow/session engine. Screens operate at the network and transport layers, providing stateless attack detection that runs before security policy evaluation.

## Screen Profile Basics

### Create and apply a screen profile
```
# Create a screen profile
set security screen ids-option PERIMETER-SCREEN <screen-options>

# Apply screen profile to a zone
set security zones security-zone untrust screen PERIMETER-SCREEN

# Each zone can have ONE screen profile
# Screens are evaluated on ingress to the zone
# Screen processing happens BEFORE session lookup and policy evaluation
```

## ICMP Screens

### ICMP flood protection
```
set security screen ids-option PERIMETER-SCREEN icmp flood
set security screen ids-option PERIMETER-SCREEN icmp flood threshold 1000
# Drops ICMP when rate exceeds threshold (packets per second) per destination
```

### Ping of death
```
set security screen ids-option PERIMETER-SCREEN icmp ping-death
# Detects oversized ICMP packets (> 65535 bytes after reassembly)
# Historically exploited buffer overflows in IP stack implementations
```

### Large ICMP packets
```
set security screen ids-option PERIMETER-SCREEN icmp large
set security screen ids-option PERIMETER-SCREEN icmp large threshold 1024
# Drops ICMP packets larger than threshold (bytes)
# Legitimate ping rarely exceeds 1024 bytes
```

### ICMP fragment
```
set security screen ids-option PERIMETER-SCREEN icmp fragment
# Drops fragmented ICMP packets
# ICMP messages are small — fragmentation indicates evasion or attack
```

## IP Screens

### Bad IP options
```
set security screen ids-option PERIMETER-SCREEN ip bad-option
# Drops packets with malformed or unknown IP options
# IP options are rarely used legitimately — common in reconnaissance
```

### Unknown protocol
```
set security screen ids-option PERIMETER-SCREEN ip unknown-protocol
# Drops packets with undefined IP protocol numbers (> 137)
# Can indicate tunneling or protocol-level evasion
```

### IP fragment screens
```
set security screen ids-option PERIMETER-SCREEN ip block-frag
# Drops ALL IP fragments — aggressive, may break legitimate traffic

set security screen ids-option PERIMETER-SCREEN ip fragment
# Drops specifically malformed fragments (overlapping offsets, tiny fragments)
```

### Source route
```
set security screen ids-option PERIMETER-SCREEN ip source-route-option
# Drops packets with source routing options (loose or strict)
# Source routing allows sender to specify the path — used in MITM attacks
# Should ALWAYS be enabled on Internet-facing zones
```

### Tear drop attack
```
set security screen ids-option PERIMETER-SCREEN ip tear-drop
# Detects overlapping IP fragment offsets
# Overlapping fragments can crash vulnerable IP stacks
# Exploits fragment reassembly bugs
```

### Land attack
```
set security screen ids-option PERIMETER-SCREEN tcp land
# Drops packets where source IP:port == destination IP:port
# Causes infinite loops in vulnerable TCP implementations
```

## TCP Screens

### SYN flood protection
```
set security screen ids-option PERIMETER-SCREEN tcp syn-flood
set security screen ids-option PERIMETER-SCREEN tcp syn-flood alarm-threshold 1000
set security screen ids-option PERIMETER-SCREEN tcp syn-flood attack-threshold 2000
set security screen ids-option PERIMETER-SCREEN tcp syn-flood source-threshold 100
set security screen ids-option PERIMETER-SCREEN tcp syn-flood destination-threshold 4000
set security screen ids-option PERIMETER-SCREEN tcp syn-flood timeout 20

# alarm-threshold: SYN rate (per second) that triggers an alarm
# attack-threshold: SYN rate that activates SYN proxy/cookie protection
# source-threshold: per-source SYN rate limit
# destination-threshold: per-destination SYN rate limit
# timeout: seconds to wait for SYN-ACK before considering SYN abandoned
```

### SYN-FIN attack
```
set security screen ids-option PERIMETER-SCREEN tcp syn-fin
# Drops packets with both SYN and FIN flags set
# Invalid TCP flag combination — used for OS fingerprinting and evasion
```

### FIN without ACK
```
set security screen ids-option PERIMETER-SCREEN tcp fin-no-ack
# Drops FIN packets that do not have ACK set
# Normal TCP FIN always accompanies ACK — absence indicates scan/evasion
```

### TCP no flag
```
set security screen ids-option PERIMETER-SCREEN tcp tcp-no-flag
# Drops TCP packets with no flags set (NULL scan)
# NULL packets are never valid in TCP — used for stealth port scanning
```

### SYN fragment
```
set security screen ids-option PERIMETER-SCREEN tcp syn-frag
# Drops SYN packets that are fragmented
# TCP SYN is small (40-60 bytes) — fragmentation indicates evasion attempt
```

### TCP sweep and port scan
```
# Port scan detection (single source, multiple ports on single destination)
set security screen ids-option PERIMETER-SCREEN tcp port-scan
set security screen ids-option PERIMETER-SCREEN tcp port-scan threshold 5000
# threshold: time interval (microseconds) between ports scanned
# Faster scanning = smaller interval = more aggressive

# IP sweep detection (single source, same port on multiple destinations)
set security screen ids-option PERIMETER-SCREEN icmp ip-sweep
set security screen ids-option PERIMETER-SCREEN icmp ip-sweep threshold 5000
# threshold: time interval (microseconds) between hosts swept
```

## UDP Screens

### UDP flood protection
```
set security screen ids-option PERIMETER-SCREEN udp flood
set security screen ids-option PERIMETER-SCREEN udp flood threshold 1000
# Drops UDP when rate exceeds threshold (packets per second) per destination
# UDP flood is common in DDoS (NTP amplification, DNS amplification, memcached)
```

## Session Limit Screens

### Connection limits
```
# Limit sessions from a single source IP
set security screen ids-option PERIMETER-SCREEN limit-session source-ip-based 100
# Max 100 concurrent sessions from any single source IP

# Limit sessions to a single destination IP
set security screen ids-option PERIMETER-SCREEN limit-session destination-ip-based 5000
# Max 5000 concurrent sessions to any single destination IP
```

## Applying Screens to Zones

### Zone binding
```
# Apply screen profile to the untrust zone (Internet-facing)
set security zones security-zone untrust screen PERIMETER-SCREEN

# Create different profiles for different zones
set security screen ids-option DMZ-SCREEN tcp syn-flood
set security screen ids-option DMZ-SCREEN tcp syn-flood attack-threshold 5000
set security screen ids-option DMZ-SCREEN tcp syn-flood destination-threshold 10000
set security screen ids-option DMZ-SCREEN icmp flood
set security screen ids-option DMZ-SCREEN icmp flood threshold 500

set security zones security-zone dmz screen DMZ-SCREEN

# Internal zones typically need lighter screening
set security screen ids-option INTERNAL-SCREEN tcp syn-fin
set security screen ids-option INTERNAL-SCREEN tcp land
set security screen ids-option INTERNAL-SCREEN tcp tcp-no-flag
set security screen ids-option INTERNAL-SCREEN ip source-route-option

set security zones security-zone trust screen INTERNAL-SCREEN
```

## Complete Perimeter Screen Profile

### Production-ready screen configuration
```
# Comprehensive screen for Internet-facing zone
set security screen ids-option PERIMETER-SCREEN icmp flood threshold 1000
set security screen ids-option PERIMETER-SCREEN icmp ping-death
set security screen ids-option PERIMETER-SCREEN icmp large threshold 1024
set security screen ids-option PERIMETER-SCREEN icmp fragment
set security screen ids-option PERIMETER-SCREEN icmp ip-sweep threshold 5000

set security screen ids-option PERIMETER-SCREEN ip bad-option
set security screen ids-option PERIMETER-SCREEN ip unknown-protocol
set security screen ids-option PERIMETER-SCREEN ip source-route-option
set security screen ids-option PERIMETER-SCREEN ip tear-drop
set security screen ids-option PERIMETER-SCREEN ip record-route-option
set security screen ids-option PERIMETER-SCREEN ip timestamp-option
set security screen ids-option PERIMETER-SCREEN ip security-option
set security screen ids-option PERIMETER-SCREEN ip stream-option
set security screen ids-option PERIMETER-SCREEN ip spoofing

set security screen ids-option PERIMETER-SCREEN tcp syn-flood alarm-threshold 512
set security screen ids-option PERIMETER-SCREEN tcp syn-flood attack-threshold 1024
set security screen ids-option PERIMETER-SCREEN tcp syn-flood source-threshold 50
set security screen ids-option PERIMETER-SCREEN tcp syn-flood destination-threshold 4000
set security screen ids-option PERIMETER-SCREEN tcp syn-flood timeout 20
set security screen ids-option PERIMETER-SCREEN tcp syn-fin
set security screen ids-option PERIMETER-SCREEN tcp fin-no-ack
set security screen ids-option PERIMETER-SCREEN tcp tcp-no-flag
set security screen ids-option PERIMETER-SCREEN tcp syn-frag
set security screen ids-option PERIMETER-SCREEN tcp land
set security screen ids-option PERIMETER-SCREEN tcp port-scan threshold 5000

set security screen ids-option PERIMETER-SCREEN udp flood threshold 1000

set security screen ids-option PERIMETER-SCREEN limit-session source-ip-based 200
set security screen ids-option PERIMETER-SCREEN limit-session destination-ip-based 8000

set security zones security-zone untrust screen PERIMETER-SCREEN
```

## Verification Commands

### Screen status
```
show security screen ids-option PERIMETER-SCREEN          # screen profile configuration
show security screen statistics zone untrust              # screen hit counters per zone
show security screen statistics interface reth0           # per-interface screen stats
show security screen statistics zone untrust detail       # detailed counters per screen type
```

### Individual screen counters
```
show security screen statistics zone untrust | match "syn flood"
show security screen statistics zone untrust | match "port scan"
show security screen statistics zone untrust | match "icmp flood"
show security screen statistics zone untrust | match "source route"
```

### Clear counters
```
clear security screen statistics zone untrust             # reset counters for zone
clear security screen statistics interface reth0          # reset counters for interface
```

### Alarms
```
show security alarms                                      # active security alarms
show security alarms detail                               # alarm details with timestamp
clear security alarms                                     # acknowledge and clear alarms
```

## Tips

- Always apply screens to the untrust zone at minimum — this is the primary attack surface
- SYN flood thresholds must be tuned to your environment — too low causes false positives, too high provides no protection
- Source-route-option should be enabled on every Internet-facing zone — there is no legitimate use for source routing from the Internet
- Port scan and IP sweep thresholds are in microseconds — lower values mean more aggressive detection (more false positives)
- Session limits protect against resource exhaustion — set source-ip-based limits to prevent a single host from consuming all sessions
- Screens run before the session/flow engine — they add minimal latency and protect the session table from being exhausted by attacks
- Monitor screen statistics regularly — high drop counts for specific screens indicate active attacks or misconfigured thresholds
- ICMP fragment screen should almost always be enabled — legitimate ICMP is never fragmented
- land attack and tear-drop screens have near-zero false positive rates — always enable them
- Use different screen profiles for different zones — untrust needs aggressive screening, trust needs lighter protection

## See Also

- junos-security-policies, junos-firewall-filters, junos-nat-security, junos-ha-security

## References

- [Juniper TechLibrary — Screen Overview](https://www.juniper.net/documentation/us/en/software/junos/security-services/topics/concept/security-screen-overview.html)
- [Juniper TechLibrary — SYN Flood Protection](https://www.juniper.net/documentation/us/en/software/junos/security-services/topics/concept/security-screen-syn-flood.html)
- [Juniper TechLibrary — ICMP Screens](https://www.juniper.net/documentation/us/en/software/junos/security-services/topics/concept/security-screen-icmp.html)
- [Juniper TechLibrary — IP Screens](https://www.juniper.net/documentation/us/en/software/junos/security-services/topics/concept/security-screen-ip.html)
- [Juniper TechLibrary — TCP Screens](https://www.juniper.net/documentation/us/en/software/junos/security-services/topics/concept/security-screen-tcp.html)
- [Juniper TechLibrary — Session Limit Screens](https://www.juniper.net/documentation/us/en/software/junos/security-services/topics/concept/security-screen-session-limit.html)
- [RFC 4987 — TCP SYN Flooding Attacks and Common Mitigations](https://www.rfc-editor.org/rfc/rfc4987)
- [RFC 6918 — Deprecating ICMP Source Quench](https://www.rfc-editor.org/rfc/rfc6918)
