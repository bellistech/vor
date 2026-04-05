# JunOS IDS/IPS (Intrusion Detection and Prevention — IDP)

SRX IDP inspects traffic for attacks using signatures, protocol anomaly detection, and application identification. IDP policies define rule bases with attack objects and actions. Integrated into the security policy framework — attached per-policy like UTM.

## IDP Architecture on SRX

```
Security Policy                     IDP Processing
┌────────────────────┐              ┌────────────────────────────────┐
│ from trust         │   permit +   │ 1. Protocol decoding           │
│ to untrust         │   idp-policy │ 2. Application identification  │
│ match: any         │ ──────────→  │ 3. Signature matching          │
│ then: permit       │              │ 4. Protocol anomaly detection  │
│   idp: IDP-POLICY  │              │ 5. Action (drop/close/ignore)  │
└────────────────────┘              └────────────────────────────────┘

# IDP runs in the data plane after security policy permits the session
# Uses both RE (policy compilation, sig updates) and PFE (inspection)
```

## IDP Policies

### Basic IDP policy structure
```
# IDP policy contains one or more rule bases
# Each rule base contains ordered rules
# Rules match traffic → attack objects → actions

set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 match from-zone trust
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 match to-zone untrust
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 match source-address any
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 match destination-address any
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 match application default
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 match attacks predefined-attack-groups "Recommended - All attacks"
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 then action recommended
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 then notification log-attacks
set security idp idp-policy IDP-POLICY rulebase-ips rule RULE1 then severity major
```

### Rule base types
```
# rulebase-ips — main IPS rule base (signature + anomaly detection)
# rulebase-exempt — exemption rules (exclude false positives)

# IPS rules are evaluated in order within rulebase-ips
# Exempt rules are checked BEFORE IPS rules
```

## Rule Bases

### IPS rule base
```
# Critical servers — strict protection
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS match from-zone untrust
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS match to-zone dmz
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS match destination-address WEB-SERVERS
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS match application junos:HTTP
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS match attacks predefined-attack-groups "Web - All"
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS then action drop-connection
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS then notification log-attacks
set security idp idp-policy IDP-POLICY rulebase-ips rule PROTECT-SERVERS then notification alert

# General traffic — standard protection
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL match from-zone any
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL match to-zone any
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL match source-address any
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL match destination-address any
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL match application default
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL match attacks predefined-attack-groups "Recommended - All attacks"
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL then action recommended
set security idp idp-policy IDP-POLICY rulebase-ips rule GENERAL then notification log-attacks
```

### Exempt rule base
```
# Exclude known false positives from triggering
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-SCANNER match from-zone trust
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-SCANNER match to-zone dmz
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-SCANNER match source-address VULN-SCANNER
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-SCANNER match destination-address any
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-SCANNER match attacks predefined-attacks "HTTP:AUDIT:URL-SCAN"

# Exempt a specific attack from a specific source
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-FP match source-address MONITORING-SERVER
set security idp idp-policy IDP-POLICY rulebase-exempt rule EXEMPT-FP match attacks predefined-attacks "DNS:QUERY:LONG-NAME"
```

## Attack Objects

### Predefined attack objects
```
# Single attacks
show security idp attack table | match HTTP
show security idp attack table | match SQL

# Predefined attack groups
show security idp attack group
# Common groups:
#   "Recommended - All attacks"     — Juniper-curated high-confidence
#   "Web - All"                      — all web/HTTP attack signatures
#   "DNS - All"                      — all DNS attack signatures
#   "OS - All"                       — all OS-specific attacks
#   "Server - All"                   — all server-side attacks
#   "Client - All"                   — all client-side attacks
```

### Attack object types
```
# Signature-based attacks
# - Pattern matching on packet payload
# - Exact byte sequences, regex patterns
# - Stateful: can match across multiple packets in a session

# Protocol anomaly attacks
# - Detect deviations from RFC-defined protocol behavior
# - Examples: malformed HTTP headers, invalid DNS flags, oversized fields

# Compound attacks (chain)
# - Multiple conditions that must occur in sequence
# - Example: specific HTTP request followed by specific response
```

### Custom attack objects (signatures)
```
# Custom signature — match specific payload pattern
set security idp custom-attack MY-CUSTOM-SIG severity major
set security idp custom-attack MY-CUSTOM-SIG attack-type signature
set security idp custom-attack MY-CUSTOM-SIG attack-type signature context http-url-parsed
set security idp custom-attack MY-CUSTOM-SIG attack-type signature pattern ".*cmd\.exe.*"
set security idp custom-attack MY-CUSTOM-SIG attack-type signature direction client-to-server
set security idp custom-attack MY-CUSTOM-SIG attack-type signature protocol-binding application HTTP

# Custom protocol anomaly
set security idp custom-attack MY-ANOMALY severity critical
set security idp custom-attack MY-ANOMALY attack-type anomaly
set security idp custom-attack MY-ANOMALY attack-type anomaly service HTTP
set security idp custom-attack MY-ANOMALY attack-type anomaly test MALFORMED-HEADER
set security idp custom-attack MY-ANOMALY attack-type anomaly direction any

# Custom attack group
set security idp custom-attack-group MY-ATTACKS add MY-CUSTOM-SIG
set security idp custom-attack-group MY-ATTACKS add MY-ANOMALY

# Use custom attacks in rules
set security idp idp-policy IDP-POLICY rulebase-ips rule CUSTOM match attacks custom-attacks MY-CUSTOM-SIG
set security idp idp-policy IDP-POLICY rulebase-ips rule CUSTOM match attacks custom-attack-groups MY-ATTACKS
```

## IDP Actions

```
# no-action          — detect only, no prevention (IDS mode)
set security idp idp-policy P rulebase-ips rule R then action no-action

# ignore-connection  — stop inspecting this session (whitelist after match)
set security idp idp-policy P rulebase-ips rule R then action ignore-connection

# mark-diffserv      — mark packet with DSCP value (for QoS/rate-limit)
set security idp idp-policy P rulebase-ips rule R then action mark-diffserv 46

# drop-packet        — drop the offending packet, session continues
set security idp idp-policy P rulebase-ips rule R then action drop-packet

# drop-connection    — drop packet + all subsequent packets in session
set security idp idp-policy P rulebase-ips rule R then action drop-connection

# close-client       — send TCP RST to client, drop session
set security idp idp-policy P rulebase-ips rule R then action close-client

# close-server       — send TCP RST to server, drop session
set security idp idp-policy P rulebase-ips rule R then action close-server

# close-client-and-server — send TCP RST to both, drop session
set security idp idp-policy P rulebase-ips rule R then action close-client-and-server

# recommended        — use the action recommended by Juniper for each attack
set security idp idp-policy P rulebase-ips rule R then action recommended
# This is the most common setting — Juniper assigns appropriate actions per signature
```

### Action severity and notification
```
# IP action — block source or destination IP for a duration
set security idp idp-policy P rulebase-ips rule R then ip-action ip-block
set security idp idp-policy P rulebase-ips rule R then ip-action target source-address
set security idp idp-policy P rulebase-ips rule R then ip-action timeout 600
# Blocks all traffic from the source IP for 600 seconds after attack detected

# Logging
set security idp idp-policy P rulebase-ips rule R then notification log-attacks
set security idp idp-policy P rulebase-ips rule R then notification alert
set security idp idp-policy P rulebase-ips rule R then notification log-attacks alert

# Packet capture on match
set security idp idp-policy P rulebase-ips rule R then notification packet-log
set security idp idp-policy P rulebase-ips rule R then notification packet-log pre-attack 5
set security idp idp-policy P rulebase-ips rule R then notification packet-log post-attack 10
set security idp idp-policy P rulebase-ips rule R then notification packet-log post-attack-timeout 30
```

## Sensor Configuration

### IDP sensor settings
```
# Configure IDP detector engine
set security idp sensor-configuration security-configuration flow-tracking
set security idp sensor-configuration security-configuration log suppression enable
set security idp sensor-configuration security-configuration log suppression max-logs-operate 500
set security idp sensor-configuration security-configuration log suppression start-log 10
set security idp sensor-configuration security-configuration log suppression include-destination-address

# Performance tuning
set security idp sensor-configuration security-configuration detection-mode detect
set security idp sensor-configuration security-configuration ips-process-port 0-65535
```

### Apply IDP policy globally
```
# Activate the IDP policy
set security idp active-policy IDP-POLICY

# Attach to security policy
set security policies from-zone trust to-zone untrust policy INSPECT then permit application-services idp-policy IDP-POLICY
set security policies from-zone untrust to-zone dmz policy TO-DMZ then permit application-services idp-policy IDP-POLICY
```

## IDP Signature Updates

### Manual update
```
# Download and install signature database
request security idp security-package download
request security idp security-package download status
request security idp security-package install
request security idp security-package install status

# Download full update (if incremental fails)
request security idp security-package download full-update
```

### Automatic updates
```
# Schedule automatic updates
set security idp security-package automatic enable
set security idp security-package automatic interval 24
set security idp security-package automatic start-time 2024-01-01.03:00:00
set security idp security-package automatic download-timeout 60

# URL for signature updates
set security idp security-package url https://signatures.juniper.net/cgi-bin/index.cgi
```

### Signature database info
```
show security idp security-package-version
show security idp attack table
show security idp attack table | count
show security idp attack table | match "HTTP"
```

## Security Intelligence (SecIntel)

```
# SecIntel provides threat intelligence feeds from Juniper ATP Cloud
# Integrates with IDP to block known C2 servers, malware domains, etc.

# Enable SecIntel
set services security-intelligence url https://aticloud.juniper.net/v2
set services security-intelligence authentication auth-token TOKEN-STRING

# SecIntel profiles
set services security-intelligence profile SECINTEL-CC category CC
set services security-intelligence profile SECINTEL-CC category CC feed-name cc-feed
set services security-intelligence profile SECINTEL-CC default-rule then action block-drop
set services security-intelligence profile SECINTEL-CC default-rule then log

set services security-intelligence profile SECINTEL-MALWARE category Malware
set services security-intelligence profile SECINTEL-MALWARE default-rule then action block-drop

# Apply SecIntel to security policy
set services security-intelligence policy SECINTEL-POLICY CC SECINTEL-CC
set services security-intelligence policy SECINTEL-POLICY Malware SECINTEL-MALWARE

set security policies from-zone trust to-zone untrust policy WEB then permit application-services security-intelligence-policy SECINTEL-POLICY

# Custom threat feeds
set services security-intelligence profile CUSTOM-FEED category custom
set services security-intelligence profile CUSTOM-FEED category custom feed-name my-blocklist
```

### SecIntel feed types
```
# Command & Control (CC)     — known C2 server IPs and domains
# Malware                     — known malware distribution sites
# GeoIP                       — block by geographic region
# Infected Host               — internal hosts showing infection indicators
# DNS Threat                  — malicious DNS queries
# Custom                      — user-defined threat feeds
```

## Verification Commands

```
# IDP status
show security idp status
show security idp memory

# IDP policy
show security idp policies
show security idp active-policy

# Attack table
show security idp attack table
show security idp attack table | count
show security idp attack table | match "severity: critical"
show security idp attack detail "HTTP:SQL:INJ:UNION-SELECT"

# IDP counters and statistics
show security idp counters
show security idp counters packet
show security idp counters flow
show security idp counters ips
show security idp counters log

# Active attack matches
show security idp attack table running
show security idp security-package-version

# IDP sessions
show security flow session idp

# SecIntel
show services security-intelligence category summary
show services security-intelligence feed summary

# Log review
show log idpd
show security log

# Clear counters
clear security idp counters
clear security idp attack table running

# Troubleshooting
show security idp sensor-configuration
request security idp security-package download status
request security idp security-package install status
```

## Tips

- Start with "recommended" action and "Recommended - All attacks" group — Juniper's curated list minimizes false positives
- Use exempt rules to suppress known false positives instead of disabling signatures globally
- IDP inspection only works on permitted sessions — it cannot inspect traffic denied by security policy
- The "recommended" action uses Juniper's per-signature recommendation: critical attacks get drop-connection, low-severity get no-action
- IP-action with source blocking is powerful but dangerous — a spoofed source can block legitimate IPs
- Packet logging on critical rules provides forensic evidence but consumes disk space rapidly
- Signature updates are critical — run automatic updates at least daily for new vulnerability coverage
- IDP significantly impacts throughput — test in lab before enabling on production traffic paths
- SecIntel complements IDP: IDP detects attacks in-flight, SecIntel blocks known-bad destinations before the attack begins
- Custom signatures use regex — test thoroughly to avoid performance-killing patterns (catastrophic backtracking)
- Log suppression prevents alert fatigue — group repeated attacks from the same source into a single log entry

## See Also

- junos-srx, junos-utm, junos-nat, junos-firewall-filters, junos-ipsec-vpn

## References

- [Juniper TechLibrary — IDP Overview](https://www.juniper.net/documentation/us/en/software/junos/idp-policy/topics/concept/idp-overview.html)
- [Juniper TechLibrary — IDP Policies](https://www.juniper.net/documentation/us/en/software/junos/idp-policy/topics/concept/idp-policy-overview.html)
- [Juniper TechLibrary — IDP Attack Objects](https://www.juniper.net/documentation/us/en/software/junos/idp-policy/topics/concept/idp-attack-object-overview.html)
- [Juniper TechLibrary — Custom Attack Objects](https://www.juniper.net/documentation/us/en/software/junos/idp-policy/topics/concept/idp-custom-attack-object-overview.html)
- [Juniper TechLibrary — Security Intelligence](https://www.juniper.net/documentation/us/en/software/junos/security-intelligence/topics/concept/security-intelligence-overview.html)
- [Juniper TechLibrary — IDP Signature Updates](https://www.juniper.net/documentation/us/en/software/junos/idp-policy/topics/task/idp-signature-database-update.html)
