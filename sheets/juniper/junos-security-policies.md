# JunOS Security Policies

Zone-based security policies on SRX control traffic flow between security zones. Policies are evaluated top-down within a from-zone/to-zone context, with first match determining the action. Unified policies extend traditional policies with application identification (AppID).

## Policy Components

### Zones, addresses, applications
```
# Security zones define trust boundaries
set security zones security-zone trust interfaces reth1.0
set security zones security-zone trust interfaces reth2.0
set security zones security-zone untrust interfaces reth0.0
set security zones security-zone dmz interfaces reth3.0

# Address book entries (per-zone or global)
set security zones security-zone trust address-book address WEB-SERVER 10.1.1.100/32
set security zones security-zone trust address-book address DB-SERVER 10.1.2.50/32
set security zones security-zone trust address-book address-set SERVERS address WEB-SERVER
set security zones security-zone trust address-book address-set SERVERS address DB-SERVER

# Global address book (available across all zones)
set security address-book global address RFC1918-10 10.0.0.0/8
set security address-book global address RFC1918-172 172.16.0.0/12
set security address-book global address RFC1918-192 192.168.0.0/16
set security address-book global address-set RFC1918 address RFC1918-10
set security address-book global address-set RFC1918 address RFC1918-172
set security address-book global address-set RFC1918 address RFC1918-192

# Custom applications
set applications application CUSTOM-APP protocol tcp
set applications application CUSTOM-APP destination-port 8443
set applications application CUSTOM-APP-SET application CUSTOM-APP
set applications application CUSTOM-APP-SET application junos-http
```

## Policy Structure

### Basic policy
```
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match source-address any
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match destination-address any
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match application junos-http
set security policies from-zone trust to-zone untrust policy ALLOW-WEB match application junos-https
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then permit

# Deny policy (explicit)
set security policies from-zone untrust to-zone trust policy DENY-ALL match source-address any
set security policies from-zone untrust to-zone trust policy DENY-ALL match destination-address any
set security policies from-zone untrust to-zone trust policy DENY-ALL match application any
set security policies from-zone untrust to-zone trust policy DENY-ALL then deny
```

### Policy actions
```
# Permit: allow traffic, optionally with application services
set security policies from-zone trust to-zone untrust policy WEB then permit

# Permit with logging and application services
set security policies from-zone trust to-zone untrust policy WEB then permit \
    application-services idp-policy IDP-POLICY
set security policies from-zone trust to-zone untrust policy WEB then permit \
    application-services ssl-proxy profile-name SSL-FORWARD

# Deny: drop traffic silently
set security policies from-zone trust to-zone untrust policy BLOCK then deny

# Reject: drop traffic and send ICMP unreachable (TCP RST for TCP)
set security policies from-zone trust to-zone untrust policy REJECT-TELNET then reject
```

## Policy Ordering

### Insert and reorder
```
# Policies evaluated top-to-bottom within a zone pair — first match wins
# New policies are added at the END by default

# Insert before a specific policy
insert security policies from-zone trust to-zone untrust policy NEW-RULE before policy ALLOW-WEB

# Insert after a specific policy
insert security policies from-zone trust to-zone untrust policy NEW-RULE after policy ALLOW-SSH

# Move a policy
insert security policies from-zone trust to-zone untrust policy ALLOW-DNS before policy CATCH-ALL
```

### Shadowing
```
# A policy is "shadowed" when a broader policy above it matches the same traffic

# Example of shadowing:
# Policy 1: permit any → any → any (matches everything)
# Policy 2: deny any → any → junos-telnet (never reached — shadowed by Policy 1)

# Detection:
show security policies hit-count                # shadowed policies have 0 hits over time
show security policies shadow-rules             # explicit shadow detection (if supported)
```

## Global Policies

### Configuration
```
# Global policies apply regardless of zone pair
# Evaluated AFTER zone-pair-specific policies (fallback)

set security policies global policy GLOBAL-DENY-MALWARE match source-address any
set security policies global policy GLOBAL-DENY-MALWARE match destination-address any
set security policies global policy GLOBAL-DENY-MALWARE match application any
set security policies global policy GLOBAL-DENY-MALWARE match dynamic-application junos:MALWARE
set security policies global policy GLOBAL-DENY-MALWARE then deny
set security policies global policy GLOBAL-DENY-MALWARE then log session-close

# Global policy order:
#   1. Zone-pair policies (from-zone X to-zone Y) evaluated first
#   2. If no match → global policies evaluated
#   3. If no match → default policy applied
```

## Default Policy

### Configuration
```
# Default policy: action when no explicit policy matches (including global)
# Default is DENY (implicit deny all)

# Change default policy (not recommended for production)
set security policies default-policy permit-all       # permit all unmatched traffic
set security policies default-policy deny-all         # deny all unmatched (default behavior)

# In practice: always keep default-policy deny-all and use explicit policies
```

## Unified Policies (with AppID)

### Dynamic application matching
```
# Unified policies match on Layer 7 application identity (not just port)
# Requires application identification (AppID) engine

set security policies from-zone trust to-zone untrust policy ALLOW-YOUTUBE match source-address any
set security policies from-zone trust to-zone untrust policy ALLOW-YOUTUBE match destination-address any
set security policies from-zone trust to-zone untrust policy ALLOW-YOUTUBE match dynamic-application junos:YOUTUBE
set security policies from-zone trust to-zone untrust policy ALLOW-YOUTUBE then permit

set security policies from-zone trust to-zone untrust policy BLOCK-FACEBOOK match source-address any
set security policies from-zone trust to-zone untrust policy BLOCK-FACEBOOK match destination-address any
set security policies from-zone trust to-zone untrust policy BLOCK-FACEBOOK match dynamic-application junos:FACEBOOK-ACCESS
set security policies from-zone trust to-zone untrust policy BLOCK-FACEBOOK then deny
```

### Application firewall
```
# Application firewall rules use AppID for granular control
set security application-firewall rule-sets SOCIAL-MEDIA rule ALLOW-LINKEDIN match \
    dynamic-application-group junos:social-networking:linkedin
set security application-firewall rule-sets SOCIAL-MEDIA rule ALLOW-LINKEDIN then permit

set security application-firewall rule-sets SOCIAL-MEDIA rule BLOCK-REST match \
    dynamic-application-group junos:social-networking
set security application-firewall rule-sets SOCIAL-MEDIA rule BLOCK-REST then deny

# Apply to security policy
set security policies from-zone trust to-zone untrust policy SOCIAL then permit \
    application-services application-firewall rule-set SOCIAL-MEDIA
```

### URL categories in policy
```
# Match traffic based on URL category (requires enhanced web filtering license)
set security policies from-zone trust to-zone untrust policy BLOCK-GAMBLING match source-address any
set security policies from-zone trust to-zone untrust policy BLOCK-GAMBLING match destination-address any
set security policies from-zone trust to-zone untrust policy BLOCK-GAMBLING match application any
set security policies from-zone trust to-zone untrust policy BLOCK-GAMBLING match url-category Gambling
set security policies from-zone trust to-zone untrust policy BLOCK-GAMBLING then deny
set security policies from-zone trust to-zone untrust policy BLOCK-GAMBLING then log session-init

# URL categories: Gambling, Adult, Malware, Phishing, Proxy-Avoidance, etc.
```

## Policy Logging

### Session logging
```
# Log at session initialization (when policy is matched)
set security policies from-zone trust to-zone untrust policy WEB then log session-init

# Log at session close (captures duration, bytes, packets)
set security policies from-zone trust to-zone untrust policy WEB then log session-close

# Log both
set security policies from-zone trust to-zone untrust policy WEB then log session-init
set security policies from-zone trust to-zone untrust policy WEB then log session-close

# Configure log destination
set security log mode stream
set security log source-address 10.1.1.1
set security log stream SIEM-LOG host 10.5.0.10
set security log stream SIEM-LOG host port 514
set security log stream SIEM-LOG format sd-syslog     # structured data syslog (RFC 5424)
set security log stream SIEM-LOG category all
```

### Event mode logging
```
# Event mode: logs written to local file (for low-volume environments)
set security log mode event
set security log event-rate 1000                       # max events per second

# Stream mode: logs sent to external server in real-time (recommended for production)
set security log mode stream
```

## Policy Counting

### Enable counters
```
# Count packets and bytes per policy
set security policies from-zone trust to-zone untrust policy WEB then count

# View counters
show security policies hit-count
show security policies hit-count from-zone trust to-zone untrust
show security policies detail

# Reset counters
clear security policies statistics
```

## Policy Scheduling

### Time-based policies
```
# Create a schedule
set schedulers scheduler BUSINESS-HOURS start-date 2024-01-01.00:00 stop-date 2030-12-31.23:59
set schedulers scheduler BUSINESS-HOURS monday start-time 08:00 stop-time 18:00
set schedulers scheduler BUSINESS-HOURS tuesday start-time 08:00 stop-time 18:00
set schedulers scheduler BUSINESS-HOURS wednesday start-time 08:00 stop-time 18:00
set schedulers scheduler BUSINESS-HOURS thursday start-time 08:00 stop-time 18:00
set schedulers scheduler BUSINESS-HOURS friday start-time 08:00 stop-time 18:00

# Apply schedule to policy
set security policies from-zone trust to-zone untrust policy SOCIAL-MEDIA match source-address any
set security policies from-zone trust to-zone untrust policy SOCIAL-MEDIA match destination-address any
set security policies from-zone trust to-zone untrust policy SOCIAL-MEDIA match dynamic-application junos:FACEBOOK-ACCESS
set security policies from-zone trust to-zone untrust policy SOCIAL-MEDIA then deny
set security policies from-zone trust to-zone untrust policy SOCIAL-MEDIA scheduler-name BUSINESS-HOURS

# Policy is active ONLY during scheduled times — outside schedule, policy is skipped
```

## Policy-Based Forwarding

### Steer traffic to routing instance
```
# Route specific traffic via alternate path based on policy match
set security policies from-zone trust to-zone untrust policy ROUTE-VIDEO then permit
set security policies from-zone trust to-zone untrust policy ROUTE-VIDEO then permit \
    advanced-policy-based-routing routing-instance VIDEO-WAN

set routing-instances VIDEO-WAN instance-type forwarding
set routing-instances VIDEO-WAN routing-options static route 0.0.0.0/0 next-hop 10.0.2.1
```

## Verification Commands

### Policy inspection
```
show security policies                                   # summary of all policies
show security policies from-zone trust to-zone untrust   # specific zone pair
show security policies detail                            # detailed view with counters
show security policies hit-count                         # hit counts per policy
show security policies hit-count from-zone trust to-zone untrust  # specific zone pair hits
show security policies global                            # global policies
```

### Policy test
```
# Test which policy matches a specific flow (without sending traffic)
show security match-policies from-zone trust to-zone untrust source-ip 10.1.1.50 \
    destination-ip 203.0.113.10 source-port 50000 destination-port 443 protocol tcp
```

### Application identification
```
show security application-tracking counters              # AppID statistics
show security application-tracking session               # sessions with AppID
show services application-identification application summary  # known applications
show services application-identification application detail <app-name>
```

### Active sessions
```
show security flow session                               # all active sessions
show security flow session summary                       # session count summary
show security flow session source-prefix 10.1.1.0/24     # filter by source
show security flow session application junos-http        # filter by application
show security flow session policy-name WEB               # sessions matching policy
```

## Tips

- Always place more specific policies above broader ones — first match wins, and a broad permit-all shadows everything below it
- Use `show security policies hit-count` regularly to identify shadowed rules (perpetual zero-hit policies)
- Global policies are evaluated after zone-pair policies — use them for organization-wide rules (malware block, compliance)
- Unified policies with dynamic-application require multiple packets for AppID — the first few packets are handled by a preliminary permit
- Enable session-close logging on all policies — session-init alone misses duration and byte counts needed for forensics
- Policy scheduling skips the policy outside the schedule — ensure a fallback policy handles traffic during off-hours
- The `match-policies` command is invaluable for troubleshooting — test before committing new policies
- Address-sets and application-sets reduce policy count and improve readability
- Keep the default policy as deny-all — never change to permit-all in production
- URL category matching requires DNS resolution and HTTP inspection — HTTPS requires SSL proxy for full URL visibility

## See Also

- junos-firewall-filters, junos-nat-security, junos-screens, junos-advanced-security

## References

- [Juniper TechLibrary — Security Policies Overview](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/security-policy-overview.html)
- [Juniper TechLibrary — Unified Policies](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/unified-policies-overview.html)
- [Juniper TechLibrary — Application Identification](https://www.juniper.net/documentation/us/en/software/junos/application-identification/topics/concept/application-identification-overview.html)
- [Juniper TechLibrary — Global Policies](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/global-policy-overview.html)
- [Juniper TechLibrary — Policy Logging](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/security-policy-logging.html)
- [Juniper TechLibrary — URL Filtering](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/utm-url-filtering-overview.html)
- [Juniper TechLibrary — Policy Scheduling](https://www.juniper.net/documentation/us/en/software/junos/security-policies/topics/concept/security-policy-scheduling.html)
