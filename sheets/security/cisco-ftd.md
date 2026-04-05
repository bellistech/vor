# Cisco Firepower Threat Defense (FTD)

Next-generation firewall platform combining ASA firewall engine with Snort IPS, providing unified threat defense with application visibility, intrusion prevention, file/malware inspection, and SSL decryption.

## Concepts

### FTD Architecture

- **Lina (ASA engine):** Handles connection state, NAT, routing, VPN, failover
- **Snort (IPS engine):** Performs deep packet inspection, application identification, intrusion detection
- **DAQ (Data Acquisition):** Interface between Lina and Snort; passes packets for inspection
- **Management plane:** FMC (Firepower Management Center) or FDM (Firepower Device Manager)
- **Data plane:** Lina + Snort processing pipeline

### FMC vs FDM Management

| Feature | FMC | FDM |
|---------|-----|-----|
| Scale | Manages hundreds of FTDs | Single device only |
| Deployment | Separate appliance or VM | On-box web interface |
| Policy types | Full ACP, IPS, File, Identity, DNS | Simplified ACP, NAT, VPN |
| Reporting | Comprehensive dashboards, correlation | Basic event viewing |
| Multi-domain | Yes (role-based access) | No |
| API | REST API (full) | REST API (limited) |
| HA for management | FMC HA pair | N/A |
| Use case | Enterprise/SP | Small office, branch |

## Packet Flow Through FTD

```
Ingress --> Prefilter --> DAQ --> Snort Engine --> Lina (ASA) --> Egress
              |                    |                |
              |              App ID, IPS,           |
              |              File/Malware,          |
              |              URL filtering          |
              |                    |                |
              |              Verdict:               |
              |              Allow/Block/           |
              |              Drop/Alert             |
              +--- Fast-path                  NAT, Routing,
                   (bypass Snort)             Connection State,
                                              QoS
```

### Detailed Flow

1. **Prefilter policy:** First evaluation; can fast-path (bypass Snort) or tunnel-tag traffic
2. **DAQ:** Copies packet to Snort process via shared memory
3. **Snort - SSL decryption:** Decrypts TLS if SSL policy matched
4. **Snort - App identification:** Identifies application (uses first few packets)
5. **Snort - Access Control Policy:** Matches rules based on zones, networks, apps, URLs, users
6. **Snort - Intrusion policy:** Applies IPS rules (Snort rules) to allowed traffic
7. **Snort - File/Malware policy:** Inspects files (AMP cloud lookup for malware)
8. **Snort verdict:** Returns allow/drop/block to Lina via DAQ
9. **Lina:** Applies NAT, routing, connection tracking, sends packet out

## Access Control Policy (ACP)

### Rule Components

| Component | Options |
|-----------|---------|
| Source/Dest Zones | Security zones (interfaces) |
| Source/Dest Networks | IP addresses, ranges, objects, groups |
| VLAN Tags | 802.1Q VLAN IDs |
| Users | AD users/groups (requires Identity policy) |
| Applications | 4000+ app signatures (e.g., Facebook, YouTube) |
| URLs | Category/reputation or manual URL objects |
| Source/Dest Ports | TCP/UDP ports, port objects |
| Intrusion Policy | Applied to allowed traffic |
| File Policy | Applied to allowed traffic |
| Logging | Connection events (begin/end) |

### Rule Actions

| Action | Behavior |
|--------|----------|
| Allow | Permit traffic, apply IPS and file policy |
| Trust | Permit traffic, bypass Snort (fast-path after initial identification) |
| Monitor | Log and continue evaluation (does not terminate matching) |
| Block | Drop and optionally reset |
| Block with reset | Drop and send TCP RST |
| Interactive Block | Display block page (HTTP only) |

### ACP Rule Evaluation Order

1. Prefilter policy (fast-path, tunnel rules)
2. ACP rules evaluated **top-down** (first match wins)
3. Rules with URL/App conditions may require multiple packets to match
4. Default action (last resort): Block All, Trust All, or Intrusion Prevention

```
! FTD CLI: verify ACP deployment
system support diagnostic-cli
show access-list
show access-control-config
```

## Intrusion Policies (Snort Rules)

### Built-in Policies

| Policy | Description |
|--------|-------------|
| Connectivity over Security | Minimum IPS, best performance |
| Balanced Security and Connectivity | Recommended starting point |
| Security over Connectivity | Aggressive IPS, may impact performance |
| Maximum Detection | All rules enabled, high CPU impact |

### Custom Tuning

- Enable/disable individual Snort rules (SIDs)
- Create custom Snort rules
- Set rule actions: Generate Events, Drop and Generate Events, Disable
- Variable sets: define HOME_NET, EXTERNAL_NET, HTTP_PORTS, etc.

## File and Malware Policy (AMP)

### File Policy Actions

| Action | Description |
|--------|-------------|
| Detect Files | Log file events, no blocking |
| Block Files | Block specific file types |
| Malware Cloud Lookup | SHA256 lookup against AMP cloud |
| Block Malware | Block files with malware disposition |
| Dynamic Analysis | Submit unknown files to Threat Grid sandbox |

### AMP Dispositions

| Disposition | Meaning |
|-------------|---------|
| Clean | Known good file |
| Malware | Known malicious |
| Unknown | Not seen before (can submit for analysis) |
| Custom Detection | Matches custom file list |
| Unavailable | Cloud lookup failed |

## SSL/TLS Decryption Policy

### Decryption Modes

| Mode | Description |
|------|-------------|
| Decrypt - Resign | FTD acts as man-in-the-middle, re-signs with internal CA |
| Decrypt - Known Key | FTD has the server's private key (for inbound traffic) |
| Do Not Decrypt | Pass encrypted traffic without inspection |
| Block | Block encrypted traffic matching the rule |

### Configuration Steps

1. Generate or import an internal CA certificate on FMC
2. Deploy the CA to client trust stores (AD GPO, MDM, etc.)
3. Create SSL policy with decrypt rules
4. Associate SSL policy with ACP
5. Define undecryptable actions (certificate pinning, etc.)

```
! Verify SSL decryption
system support diagnostic-cli
show ssl-policy-config
debug snort tls
```

## NAT Configuration

### NAT Rule Types

| Type | Priority | Description |
|------|----------|-------------|
| Twice NAT (Manual NAT) | Section 1 (top) | Full control: source AND destination NAT in one rule |
| Auto NAT | Section 2 | Object-based: one NAT rule per network object |
| Twice NAT (after-auto) | Section 3 (bottom) | Manual NAT evaluated after auto NAT |

### NAT Processing Order

1. **Section 1:** Twice NAT rules (manual, top priority)
2. **Section 2:** Auto NAT rules (sorted by prefix length, longest match first)
3. **Section 3:** Twice NAT rules (after-auto)

### Common NAT Patterns

```
! FTD CLI (diagnostic mode)
system support diagnostic-cli

! Show NAT configuration
show nat
show nat detail
show xlate

! PAT (Port Address Translation) — most common for outbound
! Configured via FMC: Objects > NAT > Add Auto NAT Rule
! Interface PAT: translate source to egress interface IP

! Static NAT — inbound server access
! Map public IP to internal server IP 1:1

! Twice NAT — source and destination translation
! Used for overlapping address spaces, DNS doctoring
```

## Identity Policy

### Purpose

- Maps IP addresses to Active Directory users and groups
- Enables user-based ACP rules
- Sources: AD agent (passive), captive portal (active), ISE/pxGrid

### Identity Sources

| Source | Method | Inline? |
|--------|--------|---------|
| AD Agent (TS Agent) | Monitors AD login events | No (passive) |
| Captive Portal | HTTP redirect for authentication | Yes (active) |
| ISE/pxGrid | Receives user-IP mapping from ISE | No (passive) |

## Prefilter Policy

### Purpose

- First-touch policy before ACP
- Handles traffic that cannot or should not go through Snort
- Tunnel rules: tag encapsulated traffic (GRE, IP-in-IP) for ACP matching
- Prefilter rules: fast-path or analyze specific traffic early

### Use Cases

- Fast-path backup traffic (high volume, trusted)
- Fast-path encrypted traffic you cannot decrypt
- Tag GRE tunnel traffic to apply different ACP rules per tunnel
- Block known-bad traffic before it reaches Snort (save CPU)

## FlexConfig

- CLI-based configuration for features not exposed in FMC GUI
- Uses ASA-style CLI commands with Jinja-like templating
- Deployed as part of the policy push from FMC
- Common uses: advanced routing, WCCP, policy-based routing, EIGRP tweaks

```
! Example FlexConfig for policy-based routing
route-map PBR permit 10
 match ip address ACL_PBR
 set ip next-hop 10.1.1.1
!
interface GigabitEthernet0/0
 policy-route route-map PBR
```

## High Availability

### Active/Standby Failover

```
! FTD HA is configured via FMC
! Requirements:
! - Same model and version
! - Dedicated failover link (and optional state link)
! - Same interface configuration
! - Same licenses

! FTD CLI: check failover status
show failover
show failover state
show failover history

! Force failover
failover active
no failover active
```

### Multi-Instance

- FTD on Firepower 4100/9300 supports multiple container instances
- Each instance is an independent FTD with its own policies
- Resource profiles define CPU, memory allocation per instance
- Managed independently in FMC

## FTD CLI Troubleshooting

### Accessing CLI Modes

```
! FTD has two CLI modes:
! 1. Clish (default) — FTD-specific commands
! 2. Diagnostic CLI — ASA-style commands

! Enter diagnostic CLI from Clish
system support diagnostic-cli

! Return to Clish
exit

! Expert mode (Linux shell, use with caution)
expert
```

### Packet Capture

```
! Capture on an interface
system support diagnostic-cli
capture CAP1 interface INSIDE match ip host 10.1.1.100 host 8.8.8.8
capture CAP1 interface INSIDE match tcp host 10.1.1.100 host 93.184.216.34 eq 443

! Show captures
show capture CAP1
show capture CAP1 detail
show capture CAP1 dump

! Export capture (copy to FMC or TFTP)
copy /pcap capture:CAP1 tftp://10.1.1.200/capture.pcap

! Remove capture
no capture CAP1
```

### Packet Tracer

```
! Simulate packet flow through FTD
system support diagnostic-cli
packet-tracer input INSIDE tcp 10.1.1.100 12345 8.8.8.8 443

! Packet tracer shows:
! - Phase 1: Route lookup
! - Phase 2: Access list (prefilter)
! - Phase 3: NAT (un-nat)
! - Phase 4: Access list (ACP, mapped to Lina ACL)
! - Phase 5: NAT (nat)
! - Phase 6: IP-options
! - Phase 7: Snort verdict
! - Phase 8: Route lookup (egress)
! - Result: ALLOW or DROP (with reason)
```

### Connection Events

```
! Show active connections
show conn
show conn detail
show conn address 10.1.1.100

! Show connection count
show conn count

! Show Snort statistics
show snort statistics

! Show ASP drops (accelerated security path)
show asp drop

! Show interface stats
show interface
show interface ip brief
```

### Snort Instance Monitoring

```
! From Clish
show snort instances
show snort counters

! Snort process restart (disruptive)
restart snort

! Snort statistics
system support diagnostic-cli
show snort statistics
show snort tls-offload
```

## Tips

- Always use prefilter to fast-path high-volume trusted traffic (backups, replication) to reduce Snort load.
- ACP rules are evaluated top-down; place the most specific and most-hit rules at the top.
- Trust action bypasses Snort after initial app identification; use it for known-good traffic you do not need to inspect.
- SSL decryption is CPU intensive; decrypt only what you need to inspect (not all traffic).
- packet-tracer is the single most useful troubleshooting tool on FTD; always start there.
- FTD connection events in FMC are essential for troubleshooting; enable logging on relevant ACP rules.
- NAT rules process Section 1 (manual) before Section 2 (auto); when in doubt, check with "show nat detail."
- FlexConfig is powerful but fragile; test in lab before deploying to production.
- Multi-instance mode requires Firepower 4100/9300 chassis; 2100 series does not support it.
- Always verify Snort process health after policy deployment; a crashed Snort instance causes traffic drops.

## See Also

- iptables, nftables, ipsec, radius, cisco-ise

## References

- [Cisco FTD Configuration Guide](https://www.cisco.com/c/en/us/td/docs/security/firepower/70/configuration/guide/fpmc-config-guide-v70.html)
- [Cisco FMC REST API Guide](https://www.cisco.com/c/en/us/td/docs/security/firepower/70/api/rest/firepower-management-center-rest-api-quick-start-guide-70.html)
- [Cisco FTD CLI Reference](https://www.cisco.com/c/en/us/td/docs/security/firepower/command_ref/b_Command_Reference_for_Firepower_Threat_Defense.html)
- [Snort 3 Documentation](https://www.snort.org/documents)
- [Cisco TALOS Intelligence](https://talosintelligence.com/)
- [Cisco Firepower Migration Tool](https://www.cisco.com/c/en/us/td/docs/security/firepower/migration-tool/migration-guide.html)
