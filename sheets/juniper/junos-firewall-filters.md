# JunOS Firewall Filters (Stateless Packet Filtering)

Stateless packet filters applied at interface level in the forwarding plane — NOT routing policy. Terms evaluated sequentially, first match wins, implicit deny at end.

## Filter vs Routing Policy

```
Firewall Filters                     Routing Policy
├── Operate on PACKETS               ├── Operate on ROUTES
├── Forwarding plane (PFE)           ├── Control plane (RE)
├── Match IP headers, ports, flags   ├── Match prefixes, AS-paths, communities
├── Actions: accept/discard/reject   ├── Actions: accept/reject/modify attributes
└── Applied on interfaces            └── Applied on protocols (import/export)
```

## Filter Structure

### Basic hierarchy
```
firewall {
    family inet {
        filter FILTER-NAME {
            term TERM-NAME {
                from {
                    <match-conditions>;
                }
                then {
                    <actions>;
                }
            }
        }
    }
}
```

### Configure a filter
```
set firewall family inet filter PROTECT-RE term ALLOW-SSH from source-address 10.0.0.0/8
set firewall family inet filter PROTECT-RE term ALLOW-SSH from protocol tcp
set firewall family inet filter PROTECT-RE term ALLOW-SSH from destination-port ssh
set firewall family inet filter PROTECT-RE term ALLOW-SSH then accept

set firewall family inet filter PROTECT-RE term ALLOW-SNMP from source-address 10.0.0.0/8
set firewall family inet filter PROTECT-RE term ALLOW-SNMP from protocol udp
set firewall family inet filter PROTECT-RE term ALLOW-SNMP from destination-port snmp
set firewall family inet filter PROTECT-RE term ALLOW-SNMP then accept

set firewall family inet filter PROTECT-RE term DENY-ALL then discard
```

### View filter
```
show firewall filter PROTECT-RE            # operational mode
show configuration firewall                # configuration hierarchy
show firewall                              # filter statistics (counters)
```

## Terms

### Evaluation rules
```
# Terms are evaluated TOP to BOTTOM — first match wins
# If a term has no "from" clause, it matches ALL packets
# If a term has no "then" clause, the default action is accept
# If no term matches, the IMPLICIT DENY (discard) drops the packet
# "next term" action continues evaluation to the next term
```

### Multiple match conditions
```
# Within a single "from" clause:
#   Same field  = OR  (source-address 10.0.0.0/8 OR 172.16.0.0/12)
#   Diff fields = AND (protocol tcp AND destination-port 22)

set firewall family inet filter EXAMPLE term MULTI from source-address 10.0.0.0/8
set firewall family inet filter EXAMPLE term MULTI from source-address 172.16.0.0/12
set firewall family inet filter EXAMPLE term MULTI from protocol tcp
set firewall family inet filter EXAMPLE term MULTI from destination-port 22
# Matches: (src 10/8 OR src 172.16/12) AND tcp AND dport 22
```

## Match Conditions

### Address and protocol
```
from {
    source-address 10.0.0.0/8;          # source IP/prefix
    destination-address 192.168.1.0/24;  # destination IP/prefix
    protocol tcp;                         # ip protocol (tcp, udp, icmp, ospf, gre...)
    source-port 1024-65535;              # source port or range
    destination-port [22 80 443];        # destination port list
}
```

### TCP and ICMP
```
from {
    tcp-flags syn;                       # match SYN flag
    tcp-flags "syn & !ack";              # SYN without ACK (initial)
    tcp-established;                     # shorthand: ACK or RST set
    tcp-initial;                         # shorthand: SYN set, ACK not set
    icmp-type echo-request;              # ping
    icmp-type echo-reply;                # ping reply
    icmp-type unreachable;               # destination unreachable
}
```

### Advanced conditions
```
from {
    dscp ef;                             # DSCP value (ef, af11, be, cs1...)
    forwarding-class expedited-forwarding;  # CoS forwarding class
    interface ge-0/0/0;                  # ingress interface
    packet-size 1500;                    # packet size (bytes)
    fragment-offset 0;                   # fragment offset
    ip-options any;                      # packets with IP options
}
```

## Actions — Terminating

```
then {
    accept;                              # permit the packet
    discard;                             # silently drop (no notification)
    reject;                              # drop + send ICMP unreachable
    reject message-type;                 # reject with specific ICMP message
}
# Terminating actions STOP term evaluation — no further terms checked
```

## Actions — Non-Terminating

```
then {
    count COUNTER-NAME;                  # increment named counter
    log;                                 # log to forwarding-table log buffer
    syslog;                              # send to system syslog
    policer POLICER-NAME;               # apply rate limiter
    forwarding-class expedited-forwarding;  # set CoS class
    loss-priority high;                  # set loss priority (high/medium-high/medium-low/low)
    next term;                           # continue to next term (explicit)
}
# Non-terminating actions execute, then CONTINUE to next term
# Multiple non-terminating + one terminating action allowed per term
```

## Applying Filters to Interfaces

### Input and output
```
set interfaces ge-0/0/0 unit 0 family inet filter input FILTER-IN
set interfaces ge-0/0/0 unit 0 family inet filter output FILTER-OUT
```

### Input list (multiple filters)
```
set interfaces ge-0/0/0 unit 0 family inet filter input-list FILTER-A
set interfaces ge-0/0/0 unit 0 family inet filter input-list FILTER-B
# Filters evaluated in order listed
```

### Verify applied filters
```
show interfaces ge-0/0/0 detail | match filter
show configuration interfaces ge-0/0/0 unit 0 family inet filter
```

## Loopback Filter — Protecting the RE

### Why lo0 matters
```
# lo0.0 = the Routing Engine's interface to the forwarding plane
# ALL traffic destined to the device itself hits lo0 input filter
# This includes: SSH, SNMP, NTP, BGP, OSPF, ICMP, DNS, RADIUS
# Without a lo0 filter, the RE is exposed to all traffic — CRITICAL RISK
```

### Protect-RE filter
```
set firewall family inet filter PROTECT-RE term ALLOW-BGP from source-address 10.0.0.0/24
set firewall family inet filter PROTECT-RE term ALLOW-BGP from protocol tcp
set firewall family inet filter PROTECT-RE term ALLOW-BGP from destination-port bgp
set firewall family inet filter PROTECT-RE term ALLOW-BGP then accept

set firewall family inet filter PROTECT-RE term ALLOW-OSPF from protocol ospf
set firewall family inet filter PROTECT-RE term ALLOW-OSPF then accept

set firewall family inet filter PROTECT-RE term ALLOW-SSH from source-address 10.1.0.0/24
set firewall family inet filter PROTECT-RE term ALLOW-SSH from protocol tcp
set firewall family inet filter PROTECT-RE term ALLOW-SSH from destination-port ssh
set firewall family inet filter PROTECT-RE term ALLOW-SSH then count SSH-COUNT
set firewall family inet filter PROTECT-RE term ALLOW-SSH then accept

set firewall family inet filter PROTECT-RE term ALLOW-SNMP from source-address 10.1.0.0/24
set firewall family inet filter PROTECT-RE term ALLOW-SNMP from protocol udp
set firewall family inet filter PROTECT-RE term ALLOW-SNMP from destination-port snmp
set firewall family inet filter PROTECT-RE term ALLOW-SNMP then accept

set firewall family inet filter PROTECT-RE term ALLOW-NTP from protocol udp
set firewall family inet filter PROTECT-RE term ALLOW-NTP from destination-port ntp
set firewall family inet filter PROTECT-RE term ALLOW-NTP then accept

set firewall family inet filter PROTECT-RE term ALLOW-ICMP from protocol icmp
set firewall family inet filter PROTECT-RE term ALLOW-ICMP from icmp-type echo-request
set firewall family inet filter PROTECT-RE term ALLOW-ICMP then policer ICMP-LIMIT
set firewall family inet filter PROTECT-RE term ALLOW-ICMP then accept

set firewall family inet filter PROTECT-RE term DENY-ALL then count DENIED
set firewall family inet filter PROTECT-RE term DENY-ALL then log
set firewall family inet filter PROTECT-RE term DENY-ALL then syslog
set firewall family inet filter PROTECT-RE term DENY-ALL then discard

# Apply to loopback
set interfaces lo0 unit 0 family inet filter input PROTECT-RE
```

## Policers

### Configure a policer
```
set firewall policer RATE-LIMIT-1M if-exceeding bandwidth-limit 1m
set firewall policer RATE-LIMIT-1M if-exceeding burst-size-limit 625k
set firewall policer RATE-LIMIT-1M then discard

set firewall policer ICMP-LIMIT if-exceeding bandwidth-limit 512k
set firewall policer ICMP-LIMIT if-exceeding burst-size-limit 15k
set firewall policer ICMP-LIMIT then discard
```

### Apply policer in filter
```
set firewall family inet filter EDGE term POLICE-ICMP from protocol icmp
set firewall family inet filter EDGE term POLICE-ICMP then policer ICMP-LIMIT
set firewall family inet filter EDGE term POLICE-ICMP then accept
```

### View policer statistics
```
show policer
show firewall filter EDGE
```

## Unicast RPF (uRPF)

### Strict mode
```
# Packet source address MUST be reachable via the SAME interface it arrived on
set interfaces ge-0/0/0 unit 0 family inet rpf-check

# Strict mode drops if:
#   - No route to source address exists, OR
#   - Route exists but points to a DIFFERENT interface
```

### Loose mode
```
# Packet source address must have ANY route in the table (any interface)
set interfaces ge-0/0/0 unit 0 family inet rpf-check mode loose

# Loose mode drops only if:
#   - No route to source address exists at all
#   - Default route DOES satisfy loose check
```

### Feasible paths
```
# Accept if source is reachable via ANY equal-cost or feasible path
set interfaces ge-0/0/0 unit 0 family inet rpf-check feasible-paths
```

### RPF check with fail filter
```
# Apply a filter to packets that FAIL the RPF check (instead of just dropping)
set interfaces ge-0/0/0 unit 0 family inet rpf-check fail-filter RPF-FAIL-LOG

set firewall family inet filter RPF-FAIL-LOG term LOG-AND-DROP from source-address 0.0.0.0/0
set firewall family inet filter RPF-FAIL-LOG term LOG-AND-DROP then count RPF-FAILURES
set firewall family inet filter RPF-FAIL-LOG term LOG-AND-DROP then log
set firewall family inet filter RPF-FAIL-LOG term LOG-AND-DROP then discard
```

## Filter-Based Forwarding

### Steering traffic to a specific routing instance
```
set firewall family inet filter FBF term ROUTE-WEB from destination-port 80
set firewall family inet filter FBF term ROUTE-WEB then routing-instance WEB-VR

set firewall family inet filter FBF term DEFAULT then accept

# Apply as input filter
set interfaces ge-0/0/0 unit 0 family inet filter input FBF

# Requires RIB group or routing-instance configuration
set routing-instances WEB-VR instance-type forwarding
set routing-instances WEB-VR routing-options static route 0.0.0.0/0 next-hop 10.0.0.1
```

## Transit vs Exception Traffic

```
# Transit traffic: passes THROUGH the device (forwarded)
#   - Hits input filter on ingress interface
#   - Hits output filter on egress interface
#   - Does NOT hit lo0 filter

# Exception traffic: destined TO the device itself
#   - Hits input filter on ingress interface
#   - Hits lo0 input filter (if configured)
#   - Examples: SSH to device, BGP sessions, SNMP polls, pings to device IP
```

## Practical Examples

### Rate-limit ICMP on all interfaces
```
set firewall policer ICMP-POLICE if-exceeding bandwidth-limit 1m
set firewall policer ICMP-POLICE if-exceeding burst-size-limit 15k
set firewall policer ICMP-POLICE then discard

set firewall family inet filter LIMIT-ICMP term ICMP from protocol icmp
set firewall family inet filter LIMIT-ICMP term ICMP then policer ICMP-POLICE
set firewall family inet filter LIMIT-ICMP term ICMP then accept
set firewall family inet filter LIMIT-ICMP term ALL-ELSE then accept

set interfaces ge-0/0/0 unit 0 family inet filter input LIMIT-ICMP
```

### Allow only SSH and SNMP to device
```
set firewall family inet filter MGMT-ONLY term SSH from protocol tcp
set firewall family inet filter MGMT-ONLY term SSH from destination-port 22
set firewall family inet filter MGMT-ONLY term SSH from source-address 10.1.0.0/24
set firewall family inet filter MGMT-ONLY term SSH then accept

set firewall family inet filter MGMT-ONLY term SNMP from protocol udp
set firewall family inet filter MGMT-ONLY term SNMP from destination-port 161
set firewall family inet filter MGMT-ONLY term SNMP from source-address 10.1.0.0/24
set firewall family inet filter MGMT-ONLY term SNMP then accept

set firewall family inet filter MGMT-ONLY term DENY then discard

set interfaces lo0 unit 0 family inet filter input MGMT-ONLY
```

## Tips

- Always end filters with an explicit deny term that counts and logs — easier to troubleshoot than the implicit deny
- Put the most-matched terms first (e.g., established traffic) for performance
- Use `count` on every term during testing — verify which terms are actually matching
- Test filters in lab before applying to production lo0 — a misconfigured RE filter locks you out
- `reject` sends ICMP unreachable (useful for debugging); `discard` is silent (better for security)
- `tcp-established` is NOT stateful — it just checks for ACK or RST flags
- `next term` is the only way to continue evaluation after a non-terminating action set
- Use `input-list` to apply multiple modular filters instead of one monolithic filter
- Policer burst-size should be at least 10x the MTU to avoid dropping legitimate traffic
- uRPF strict mode can break asymmetric routing — use loose mode or feasible-paths in those cases

## See Also

- iptables, nftables, bgp, ospf

## References

- [Juniper TechLibrary — Firewall Filters Overview](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/firewall-filter-overview.html)
- [Juniper TechLibrary — Firewall Filter Match Conditions](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/ref/statement/firewall-filter-match-conditions.html)
- [Juniper TechLibrary — Policer Configuration](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/policer-overview.html)
- [Juniper TechLibrary — Unicast RPF](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/concept/unicast-rpf-overview.html)
- [Juniper TechLibrary — Protecting the Routing Engine](https://www.juniper.net/documentation/us/en/software/junos/routing-policy/topics/example/firewall-filter-protect-re.html)
- [Juniper JNCIA-Junos Study Guide](https://www.juniper.net/us/en/training/certification/tracks/junos/jncia-junos.html)
- [RFC 2827 — BCP38 Network Ingress Filtering (uRPF)](https://www.rfc-editor.org/rfc/rfc2827)
- [RFC 3704 — Ingress Filtering for Multihomed Networks](https://www.rfc-editor.org/rfc/rfc3704)
