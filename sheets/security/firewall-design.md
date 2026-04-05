# Firewall Architecture and Design

Firewall types, placement patterns, zone-based architecture, rule design, high availability, sizing, microsegmentation, cloud firewalls, and policy lifecycle management.

## Firewall Types

### Packet Filter (Stateless)

```
# Operates at Layer 3-4 (IP + TCP/UDP headers only)
# Evaluates each packet independently — no connection tracking
# Fast but limited; no awareness of connection state

# Example: Linux iptables stateless rule
iptables -A FORWARD -s 10.0.0.0/8 -d 192.168.1.0/24 -p tcp --dport 80 -j ACCEPT

# Characteristics:
# - No connection table (every packet evaluated against full ruleset)
# - Cannot enforce "established" connections
# - Vulnerable to ACK scan attacks (attacker sends ACK, packet filter allows)
# - Still used in high-speed environments (router ACLs) where throughput > inspection
# - Cisco IOS extended ACLs are stateless packet filters

# Cisco IOS example:
# ip access-list extended PERIMETER-IN
#   permit tcp any host 203.0.113.10 eq 443
#   permit tcp any host 203.0.113.10 eq 80
#   deny   ip any any log
```

### Stateful Firewall

```
# Operates at Layer 3-4 with connection tracking
# Maintains a state table of active connections
# Return traffic automatically allowed if connection is tracked

# Linux nftables stateful example:
nft add rule inet filter forward ct state established,related accept
nft add rule inet filter forward ip saddr 10.0.0.0/8 tcp dport 443 accept
nft add rule inet filter forward drop

# Cisco ASA stateful inspection:
# access-list OUTSIDE_IN extended permit tcp any host 203.0.113.10 eq 443
# access-group OUTSIDE_IN in interface outside
# (return traffic allowed automatically by ASA stateful engine)

# State table entry (conceptual):
# Proto  Src IP        Src Port  Dst IP        Dst Port  State     Timeout
# TCP    10.0.1.50     49152     93.184.216.34 443       ESTABLISHED 3600
# UDP    10.0.1.50     53421     8.8.8.8       53        ACTIVE      30
# ICMP   10.0.1.50     -         8.8.4.4       -         ACTIVE      10
```

### Next-Generation Firewall (NGFW)

```
# Operates at Layers 3-7 with application awareness
# Combines stateful inspection + application identification + IPS + URL filtering

# Key NGFW capabilities:
# - Application identification (recognizes apps regardless of port)
# - User identity integration (AD, LDAP, SAML — policy per user/group)
# - Intrusion Prevention System (IPS / Snort / Suricata rules)
# - URL filtering and categorization
# - SSL/TLS decryption and inspection
# - File type filtering and malware detection (sandboxing)
# - Threat intelligence feeds

# Palo Alto NGFW rule example:
# Rule: Allow-Web-Browsing
#   Source Zone: Trust
#   Source User: domain\marketing-group
#   Destination Zone: Untrust
#   Application: web-browsing, ssl
#   Service: application-default
#   Action: Allow
#   Profile: AV=strict, IPS=strict, URL=corporate

# Cisco Firepower NGFW (FTD):
# Uses Snort 3 engine for application detection
# Access Control Policy → Rules → Application filters
# Network Analysis Policy → IPS signature sets
```

### Web Application Firewall (WAF)

```
# Operates at Layer 7 (HTTP/HTTPS application layer)
# Protects web applications from application-specific attacks

# Deployment modes:
# - Reverse proxy (inline, terminates client connection)
# - Transparent bridge (inline, no IP change)
# - Out-of-band (monitor only via TAP/SPAN)

# Common WAF rule categories:
# - SQL injection (SQLi)
# - Cross-site scripting (XSS)
# - Cross-site request forgery (CSRF)
# - Directory traversal / path traversal
# - Remote file inclusion (RFI) / Local file inclusion (LFI)
# - Command injection
# - OWASP Top 10 coverage

# ModSecurity (open source WAF) with CRS:
# SecRuleEngine On
# Include /etc/modsecurity/crs/crs-setup.conf
# Include /etc/modsecurity/crs/rules/*.conf

# AWS WAF rule group example:
# aws waf-regional create-rule --name "SQLi-Protection" \
#   --metric-name SQLiProtection
# Uses AWS Managed Rule Groups: AWSManagedRulesSQLiRuleSet
```

### Cloud-Native Firewall

```
# Cloud provider built-in firewall constructs
# Operate at the hypervisor/SDN level (no appliance)

# Characteristics:
# - Stateful by default (all major clouds)
# - No throughput limits (scaled by cloud fabric)
# - API-driven (Infrastructure as Code)
# - Instance-level or subnet-level enforcement
# - No single point of failure (distributed enforcement)
```

## Firewall Placement

### Perimeter Firewall

```
# Classic north-south boundary between internal network and internet
#
#   Internet
#      │
#   ┌──┴──┐
#   │ FW  │  ← Perimeter firewall
#   └──┬──┘
#      │
#   Internal Network
#
# Inspects all ingress/egress traffic
# First line of defense; blocks unsolicited inbound
# Handles NAT, VPN termination, basic access control
# Insufficient alone (no internal segmentation, no lateral movement prevention)
```

### DMZ Architecture

```
# Single-firewall DMZ (three-legged):
#
#   Internet
#      │
#   ┌──┴──┐
#   │ FW  │──── DMZ (web servers, mail relay, DNS)
#   └──┬──┘
#      │
#   Internal Network
#
# Firewall has three interfaces: outside, DMZ, inside
# DMZ servers accessible from internet (limited ports)
# DMZ servers can reach internal only on specific ports
# Internal can reach DMZ freely

# Dual-firewall DMZ (recommended):
#
#   Internet
#      │
#   ┌──┴──┐
#   │ FW1 │  ← External firewall (different vendor recommended)
#   └──┬──┘
#      │
#     DMZ
#      │
#   ┌──┴──┐
#   │ FW2 │  ← Internal firewall
#   └──┬──┘
#      │
#   Internal Network
#
# Compromise of one firewall does not expose internal network
# Different vendors = different vulnerability surfaces
# More complex but significantly more secure
```

### Internal Segmentation Firewall (ISFW)

```
# Placed between internal network segments
# Controls east-west (lateral) traffic
# Critical for zero-trust and compliance (PCI DSS segmentation)
#
#   ┌──────────┐     ┌──────┐     ┌──────────┐
#   │ Users    ├─────┤ ISFW ├─────┤ Servers  │
#   │ (VLAN10) │     └──┬───┘     │ (VLAN20) │
#   └──────────┘        │         └──────────┘
#                    ┌──┴───┐
#                    │ ISFW │
#                    └──┬───┘
#                  ┌────┴─────┐
#                  │ Database │
#                  │ (VLAN30) │
#                  └──────────┘
#
# Policy example:
# Users -> Servers: allow HTTPS (443), SSH (22)
# Servers -> Database: allow PostgreSQL (5432)
# Users -> Database: DENY (no direct access)
# Database -> Internet: DENY (exfiltration prevention)
```

## Zone-Based Architecture

### Security Zone Design

```
# Zones group interfaces/segments by trust level
# Policy defined between zone pairs (inter-zone policy)
# Traffic within the same zone typically allowed by default

# Common zone hierarchy (trust levels):
# Zone          Trust Level   Description
# ─────────────────────────────────────────────────────
# Outside       0 (lowest)    Internet / untrusted
# DMZ           25            Public-facing services
# Partners      40            B2B / extranet
# Guest         30            Guest Wi-Fi / BYOD
# Users         60            Corporate endpoints
# Servers       75            Application servers
# Management    90            Network management (OOB)
# Inside        100 (highest) Core infrastructure, DCs

# Inter-zone policy matrix (simplified):
#             Outside  DMZ    Users   Servers  Mgmt
# Outside     -        →web   deny    deny     deny
# DMZ         ←resp    -      deny    →app     deny
# Users       →all     →all   -       →app     deny
# Servers     ←resp    ←resp  ←resp   -        →syslog
# Mgmt        deny     →mgmt  →mgmt  →mgmt    -

# Cisco IOS Zone-Based Policy Firewall (ZBF):
# zone security OUTSIDE
# zone security INSIDE
# zone security DMZ
# zone-pair security IN-TO-OUT source INSIDE destination OUTSIDE
#   service-policy type inspect IN-TO-OUT-POLICY
```

### Zone Design Principles

```
# 1. Least privilege between zones
#    - Default deny between all zone pairs
#    - Explicitly permit only required traffic
#    - Document every inter-zone rule with business justification

# 2. Data flow drives zone boundaries
#    - Map application data flows before designing zones
#    - Group systems with similar trust and communication patterns
#    - Avoid over-segmentation (creates management overhead)

# 3. Compliance alignment
#    - PCI DSS: cardholder data environment (CDE) in its own zone
#    - HIPAA: ePHI systems isolated from general network
#    - Separate regulated data from general-purpose zones

# 4. Management plane isolation
#    - Out-of-band management network (dedicated VLAN, dedicated firewall interface)
#    - SSH/HTTPS management traffic never traverses production zones
#    - Console servers for emergency access

# 5. Micro-zones for high-value assets
#    - Database tier in separate zone from application tier
#    - Individual zones for PCI vs non-PCI databases
#    - Per-application zones in multi-tenant environments
```

## Firewall Rule Design

### Rule Ordering and Evaluation

```
# Firewall rules evaluated top-to-bottom, first match wins
# (exception: some platforms use best-match or weighted evaluation)

# Rule ordering best practices:
# 1. Explicit deny rules for known threats (blacklist)     ← TOP
# 2. Explicit allow rules for critical services
# 3. Specific allow rules (narrow source/dest/service)
# 4. Broader allow rules (wider scope)
# 5. Default deny (implicit or explicit)                   ← BOTTOM

# Example ordered ruleset:
# Seq  Action  Src          Dst          Service     Log   Comment
# ───────────────────────────────────────────────────────────────────
# 10   DENY    Threat-Feed  any          any         yes   Block known IOCs
# 20   ALLOW   Mgmt-Net     FW-Self      SSH,HTTPS   yes   Firewall management
# 30   ALLOW   Users        DNS-Servers  DNS         no    Internal DNS
# 40   ALLOW   Users        Proxy        HTTP,HTTPS  no    Web via proxy
# 50   ALLOW   App-Servers  DB-Servers   PG(5432)    yes   App to database
# 60   ALLOW   DMZ-Web      App-Servers  HTTPS       yes   DMZ to app tier
# 90   DENY    any          any          any         yes   Default deny (cleanup)
```

### Least Privilege Rule Design

```
# Every rule should specify:
# 1. Source (as narrow as possible — single host > subnet > zone)
# 2. Destination (specific server or service group)
# 3. Service/port (exact port > port range > any)
# 4. Direction (unidirectional; return handled by state table)
# 5. Logging (enable for security-relevant rules; disable for high-volume)
# 6. Comment (business justification, ticket number, expiration date)

# BAD rule (overly permissive):
# ALLOW  10.0.0.0/8  any  any  "Allow internal to everything"

# GOOD rule (specific):
# ALLOW  10.1.5.0/24  10.2.10.50/32  tcp/443  "Marketing to CRM — JIRA-1234"

# Rule hygiene:
# - Every rule has an owner and a ticket reference
# - Temporary rules have expiration dates
# - "Any" in source, destination, or service requires VP approval
# - Review unused rules quarterly (hit-count analysis)
```

### Rule Shadowing and Redundancy

```
# Shadowed rule: a rule that can never match because a broader rule
# above it matches all the same traffic first

# Example of shadowing:
# 10   ALLOW   any          any          HTTPS       # This shadows rule 20
# 20   DENY    Attackers    Web-Server   HTTPS       # NEVER MATCHES

# Redundant rule: a rule that matches traffic already handled by
# another rule (produces same result, wastes evaluation time)

# Detection methods:
# - Firewall management tools: Tufin, AlgoSec, FireMon
# - Manual analysis: compare source/dest/service overlap between rules
# - Hit-count analysis: rules with zero hits over 90 days may be redundant
#   show access-list (Cisco ASA)
#   show rule-hit-count (Palo Alto)

# Rule optimization:
# - Merge rules with identical action and overlapping scope
# - Move high-hit-count rules higher in the list (performance)
# - Remove shadowed and zero-hit rules after verification
```

## High Availability

### Active/Standby

```
# One firewall processes traffic; the other is hot standby
# State table synchronized continuously
# Failover on health check failure (link, process, heartbeat)

# Cisco ASA Active/Standby:
# failover lan unit primary
# failover lan interface FAILOVER GigabitEthernet0/3
# failover key <shared-secret>
# failover link STATE GigabitEthernet0/4
# failover interface ip FAILOVER 10.0.99.1 255.255.255.0 standby 10.0.99.2

# show failover           — verify failover status
# show failover state     — active/standby role
# no failover active      — force failover to standby

# Failover triggers:
# - Interface link failure (monitored interfaces)
# - Hardware failure (power, fan, memory)
# - Process crash (inspection engine, routing)
# - Manual failover (maintenance window)

# Failover time: typically 1-5 seconds (depends on state sync)
```

### Active/Active

```
# Both firewalls process traffic simultaneously
# Traffic distributed across both units (load sharing)
# Each unit is primary for different contexts/VLANs

# Advantages over active/standby:
# - Utilizes both units (no idle standby hardware)
# - Higher aggregate throughput
# - Per-context or per-VLAN failover granularity

# Challenges:
# - More complex configuration
# - Asymmetric routing must be handled
# - State synchronization overhead higher

# Cisco ASA Active/Active (multi-context):
# Context A: Unit 1 = primary, Unit 2 = secondary
# Context B: Unit 1 = secondary, Unit 2 = primary
# If Unit 1 fails: Unit 2 becomes primary for both contexts

# Palo Alto Active/Active:
# Requires session setup synchronization
# Floating IPs for ARP-based failover
# Session owner vs session setup concepts
```

### Clustering

```
# Multiple firewalls (3+) operating as a single logical unit
# Traffic distributed across all cluster members
# Horizontal scaling of throughput and connection capacity

# Cisco Firepower Clustering (up to 16 units):
# cluster group FTD-CLUSTER
#   local-unit unit-1
#   cluster-interface port-channel1 ip 10.0.99.1 255.255.255.0
#   priority 1
#   enable

# Palo Alto HA Clustering:
# - Up to 16 firewalls in a cluster
# - Cluster uses hash-based load distribution
# - Session state replicated to at least one other member

# Cluster considerations:
# - Spanned EtherChannel or vPC across cluster members
# - Control plane traffic between members (cluster-link bandwidth)
# - Individual member failure: remaining members absorb traffic
# - Cluster-wide throughput = N x single-unit throughput (near-linear)
# - Firmware upgrades: rolling upgrade (one member at a time, ISSU)
```

## Firewall Sizing

### Key Metrics

```
# Throughput (Mbps/Gbps):
# - Firewall throughput: raw packet forwarding (large packets, no inspection)
# - IPS throughput: with intrusion prevention enabled
# - NGFW throughput: with app-id + IPS + URL filtering
# - Threat prevention throughput: with AV + sandboxing + all features
# - SSL decryption throughput: with TLS inspection (most CPU-intensive)
#
# Typical throughput degradation:
# Feature Stack                   % of Raw Throughput
# ──────────────────────────────────────────────────────
# Stateful firewall only          100%
# + App identification            60-80%
# + IPS                           40-60%
# + SSL decryption                20-40%
# + AV/sandboxing                 15-30%
# + All features enabled          10-25%

# Connections per second (CPS):
# - New TCP connections established per second
# - Critical for environments with many short-lived connections
# - Web servers, API gateways, load balancers stress this metric

# Concurrent sessions:
# - Maximum simultaneous connections tracked in state table
# - Each entry consumes memory (typically 1-4 KB per session)
# - Size for peak concurrent connections + 30% headroom

# Sizing example:
# 1 Gbps internet link, 80% utilization, NGFW features
# Required: ~3 Gbps rated NGFW throughput (accounting for 30% degradation)
# Expected CPS: 10,000-50,000 (depends on traffic profile)
# Concurrent sessions: 500,000-1,000,000 (typical enterprise)
```

## Microsegmentation

### Concept and Implementation

```
# Traditional segmentation: VLANs + firewall between VLANs
# Microsegmentation: per-workload firewall policy (VM or container level)

# Implementation approaches:
# 1. Host-based firewall (iptables/nftables on each server)
#    - Policy managed centrally, pushed to each host
#    - Tools: Ansible + iptables, Chef + nftables
#    - Pro: no network infrastructure changes
#    - Con: OS-dependent, requires agent on every host

# 2. Hypervisor-distributed firewall (VMware NSX, Cisco ACI)
#    - Policy enforced at the virtual switch (vNIC level)
#    - Zero-trust between VMs on the same host
#    - Pro: works regardless of guest OS
#    - Con: vendor lock-in, license cost

# 3. Container network policies (Kubernetes NetworkPolicy)
#    - Policy per pod, namespace, or label selector
#    - Enforced by CNI plugin (Calico, Cilium, Weave)
#    - Example:
#      kind: NetworkPolicy
#      spec:
#        podSelector:
#          matchLabels:
#            app: database
#        ingress:
#          - from:
#              - podSelector:
#                  matchLabels:
#                    app: backend
#            ports:
#              - port: 5432

# 4. Identity-based microsegmentation (Cisco Tetration / Illumio)
#    - Policy based on workload identity, not IP address
#    - Automatically adapts as workloads move or scale
#    - Uses process, user, and app context for decisions
```

### East-West Firewalling

```
# East-west traffic = lateral movement between servers/workloads
# In modern data centers, east-west traffic is 70-80% of total volume
# Traditional perimeter firewalls see none of this traffic

# East-west firewall placement:
#
# ┌─────────────────────────────────────────┐
# │              Data Center                 │
# │  ┌────┐    ┌────┐    ┌────┐    ┌────┐  │
# │  │Web1├──┐ │Web2├──┐ │App1├──┐ │DB1 │  │
# │  └────┘  │ └────┘  │ └────┘  │ └────┘  │
# │          │         │         │    ▲     │
# │       ┌──┴─────────┴─────────┴────┤     │
# │       │    Distributed Firewall    │     │
# │       │    (per-workload policy)   │     │
# │       └────────────────────────────┘     │
# └─────────────────────────────────────────┘

# Key policies for east-west:
# - Web tier can reach App tier on HTTPS only
# - App tier can reach DB tier on database port only
# - DB tier cannot initiate connections to any tier
# - No direct Web-to-DB communication
# - Management access only from jump box / bastion
```

## Cloud Firewalls

### AWS Security Groups and NACLs

```bash
# Security Groups (SG) — stateful, instance-level
# - Applied to ENI (network interface)
# - Allow rules only (no explicit deny)
# - Return traffic automatically allowed
# - Evaluated as a whole (not ordered)
# - Can reference other SGs as source/destination

# Create security group:
aws ec2 create-security-group \
  --group-name web-server-sg \
  --description "Web server security group" \
  --vpc-id vpc-12345678

# Add inbound rules:
aws ec2 authorize-security-group-ingress \
  --group-id sg-12345678 \
  --protocol tcp --port 443 --cidr 0.0.0.0/0

# Allow traffic from another security group:
aws ec2 authorize-security-group-ingress \
  --group-id sg-database \
  --protocol tcp --port 5432 \
  --source-group sg-appserver

# Network ACLs (NACL) — stateless, subnet-level
# - Applied to subnet
# - Allow and deny rules
# - Return traffic must be explicitly allowed
# - Rules evaluated in order (lowest number first)
# - Applies to all instances in the subnet

# NACL rules (stateless — must allow return traffic):
# Inbound:
#   100  ALLOW  TCP  443  0.0.0.0/0       (HTTPS in)
#   200  ALLOW  TCP  1024-65535  10.0.0.0/8  (return traffic from internal)
#   *    DENY   ALL  ALL  0.0.0.0/0       (implicit deny)
# Outbound:
#   100  ALLOW  TCP  1024-65535  0.0.0.0/0  (return traffic for HTTPS)
#   200  ALLOW  TCP  5432  10.0.2.0/24     (to database subnet)
#   *    DENY   ALL  ALL  0.0.0.0/0        (implicit deny)
```

### Azure NSG

```bash
# Network Security Groups — stateful, subnet or NIC level
# - Priority-based rules (100-4096, lower = higher priority)
# - Allow and deny rules
# - Default rules cannot be deleted but can be overridden
# - Application Security Groups (ASG) for grouping by role

# Create NSG:
az network nsg create --name web-nsg --resource-group myRG

# Add rule:
az network nsg rule create \
  --nsg-name web-nsg --resource-group myRG \
  --name Allow-HTTPS --priority 100 \
  --direction Inbound --access Allow \
  --protocol Tcp --destination-port-ranges 443 \
  --source-address-prefixes '*' \
  --destination-address-prefixes '*'

# Application Security Groups (logical grouping):
az network asg create --name web-servers --resource-group myRG
# Then reference ASG in NSG rules instead of IP ranges
```

### GCP Firewall Rules

```bash
# VPC Firewall Rules — stateful, applied by network tags or service accounts
# - Priority-based (0-65535, lower = higher priority)
# - Allow and deny rules
# - Applied to instances via network tags or service accounts
# - Ingress and egress rules separately

# Create firewall rule:
gcloud compute firewall-rules create allow-https \
  --network=my-vpc \
  --allow=tcp:443 \
  --source-ranges=0.0.0.0/0 \
  --target-tags=web-server \
  --priority=1000

# Allow internal communication by service account:
gcloud compute firewall-rules create allow-internal-db \
  --network=my-vpc \
  --allow=tcp:5432 \
  --source-service-accounts=app-server@project.iam.gserviceaccount.com \
  --target-service-accounts=db-server@project.iam.gserviceaccount.com \
  --priority=1000
```

## Firewall Policy Lifecycle

### Rule Review and Optimization

```
# Policy lifecycle phases:
# 1. Request   — business submits change request with justification
# 2. Review    — security team reviews for risk, compliance, necessity
# 3. Implement — network team implements in change window
# 4. Verify    — confirm rule works as intended (test connectivity)
# 5. Monitor   — track hit counts, review logs for anomalies
# 6. Audit     — quarterly review of all rules
# 7. Cleanup   — remove unused, expired, or redundant rules

# Quarterly audit checklist:
# [ ] Rules with zero hits in 90+ days — candidate for removal
# [ ] Rules with "any" in source/dest/service — justify or narrow
# [ ] Rules without comments or ticket references — add documentation
# [ ] Rules with expired dates — remove or renew
# [ ] Rules allowing direct internet access — verify necessity
# [ ] Rules created for temporary needs — verify still needed
# [ ] Shadowed rules — consolidate or remove
# [ ] Overly permissive rules — restrict to actual usage patterns

# Tools for policy analysis:
# - Tufin SecureTrack — rule usage, risk analysis, compliance
# - AlgoSec Firewall Analyzer — topology-aware policy analysis
# - FireMon — real-time rule monitoring and cleanup recommendations
# - Palo Alto Expedition — rule optimization and migration tool
# - Native hit-count analysis on all major firewall platforms
```

### Change Management

```
# Firewall changes should follow ITIL change management:
# 1. RFC (Request for Change) with business justification
# 2. Impact assessment (which traffic flows affected)
# 3. Rollback plan (save running config before change)
# 4. Change window (scheduled maintenance or emergency)
# 5. Implementation (apply changes with verification)
# 6. Post-implementation review (confirm no outages)

# Pre-change verification:
# - Save running configuration:
#   copy running-config startup-config (Cisco)
#   request system configuration save (Palo Alto)
#   export configuration (Fortinet)
# - Document current hit counts for modified rules
# - Notify affected teams and monitoring

# Post-change verification:
# - Verify intended traffic flows work
# - Verify unintended access is still blocked
# - Monitor firewall logs for unexpected denies
# - Check for session drops or connection resets
# - Confirm HA state synchronization
```

## Tips

- Default deny is non-negotiable. Every firewall should have an explicit deny-all rule at the bottom of the ruleset, with logging enabled.
- Never use "any" for source, destination, and service in the same rule. At minimum, one of these must be specific.
- Rule comments are not optional. Every rule should reference a ticket number, owner, creation date, and business justification.
- Place high-hit-count rules near the top of the ruleset for performance. Use hit-count analysis to identify and reorder.
- For dual-firewall DMZ designs, use different firewall vendors for the external and internal firewalls. A vulnerability in one vendor does not compromise both layers.
- Size firewalls based on NGFW throughput with all intended features enabled, not raw firewall throughput. SSL decryption can reduce throughput by 60-80%.
- In cloud environments, use security groups (stateful, instance-level) as the primary control and NACLs (stateless, subnet-level) as a backstop. Do not rely on NACLs alone.
- Microsegmentation does not replace perimeter firewalls. Use both: perimeter for north-south, microsegmentation for east-west.
- Schedule quarterly rule audits. An unreviewed firewall ruleset grows permissive over time as rules accumulate and nobody removes the old ones.
- Test firewall failover regularly (monthly or quarterly). An untested HA pair is a false sense of security.
- Log all deny actions and a representative sample of allow actions. Firewall logs are critical for incident investigation and compliance evidence.

## See Also

- iptables, nftables, firewalld, ufw, cisco-ftd, zero-trust, network-security-infra, pki, tls

## References

- [NIST SP 800-41 Rev 1 — Guidelines on Firewalls and Firewall Policy](https://csrc.nist.gov/publications/detail/sp/800-41/rev-1/final)
- [CIS Benchmarks — Firewall Configuration](https://www.cisecurity.org/cis-benchmarks)
- [PCI DSS v4.0 — Network Segmentation Requirements](https://www.pcisecuritystandards.org/)
- [Cisco ASA Configuration Guide](https://www.cisco.com/c/en/us/td/docs/security/asa/asa920/configuration/firewall/asa-920-firewall-config.html)
- [Palo Alto Networks — Best Practice Assessment](https://docs.paloaltonetworks.com/best-practices)
- [Fortinet FortiGate Administration Guide](https://docs.fortinet.com/product/fortigate/)
- [AWS Security Groups Documentation](https://docs.aws.amazon.com/vpc/latest/userguide/vpc-security-groups.html)
- [Azure Network Security Groups](https://learn.microsoft.com/en-us/azure/virtual-network/network-security-groups-overview)
- [GCP VPC Firewall Rules](https://cloud.google.com/vpc/docs/firewalls)
- [Kubernetes Network Policies](https://kubernetes.io/docs/concepts/services-networking/network-policies/)
- [VMware NSX Distributed Firewall](https://docs.vmware.com/en/VMware-NSX/)
