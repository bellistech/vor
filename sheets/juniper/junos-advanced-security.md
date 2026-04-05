# JunOS Advanced Security Features

Advanced SRX security features beyond basic zones/policies: SSL proxy for encrypted traffic inspection, ICAP integration, Security Intelligence feeds, encrypted traffic insights, logical systems and tenant systems for multi-tenancy, ATP Cloud for advanced threat detection, and Security Director for centralized management.

## SSL Proxy

### Forward proxy (outbound traffic inspection)
```
# SSL forward proxy: SRX terminates client TLS, inspects cleartext, re-encrypts to server
# Requires CA certificate trusted by clients (or deployed via GPO/MDM)

# 1. Generate or import root CA certificate
request security pki generate-key-pair certificate-id SSL-PROXY-CA size 2048 type rsa
request security pki local-certificate generate-self-signed certificate-id SSL-PROXY-CA \
    subject "CN=SRX-SSL-Proxy,O=Corp,C=US" \
    domain-name proxy.corp.local \
    is-ca

# 2. Configure SSL proxy profile
set services ssl proxy profile FORWARD-PROXY root-ca SSL-PROXY-CA
set services ssl proxy profile FORWARD-PROXY actions log all
set services ssl proxy profile FORWARD-PROXY actions ignore-server-auth-failure

# 3. Whitelist domains that should NOT be intercepted (banking, healthcare)
set services ssl proxy profile FORWARD-PROXY whitelist EXEMPT-LIST
set security policies from-zone trust to-zone untrust policy SSL-INSPECT match source-address any
set security policies from-zone trust to-zone untrust policy SSL-INSPECT match destination-address any
set security policies from-zone trust to-zone untrust policy SSL-INSPECT match application junos-https
set security policies from-zone trust to-zone untrust policy SSL-INSPECT then permit application-services ssl-proxy profile-name FORWARD-PROXY
```

### Reverse proxy (inbound traffic inspection)
```
# SSL reverse proxy: SRX terminates server TLS on behalf of internal servers
# Protects servers from encrypted attacks (SQLi, XSS in HTTPS)

# 1. Import server certificate and key
request security pki local-certificate load certificate-id WEB-SERVER filename /var/tmp/server.pem key /var/tmp/server.key

# 2. Configure reverse proxy profile
set services ssl proxy profile REVERSE-PROXY server-certificate WEB-SERVER
set services ssl proxy profile REVERSE-PROXY protocol-version tls12
set services ssl proxy profile REVERSE-PROXY preferred-ciphers medium

# 3. Apply to inbound policy
set security policies from-zone untrust to-zone dmz policy PROTECT-WEB then permit \
    application-services ssl-proxy profile-name REVERSE-PROXY
```

### SSL proxy exemptions
```
# Exempt specific categories or domains from SSL inspection
set services ssl proxy profile FORWARD-PROXY whitelist FINANCE url-pattern "*.bank.com"
set services ssl proxy profile FORWARD-PROXY whitelist FINANCE url-pattern "*.healthcare.gov"

# Exempt by URL category
set services ssl proxy profile FORWARD-PROXY whitelist CATEGORY url-category Financial-Institutions
set services ssl proxy profile FORWARD-PROXY whitelist CATEGORY url-category Health-and-Medicine

# Verify SSL proxy sessions
show services ssl proxy statistics
show services ssl proxy certificate-cache
show services ssl proxy session
```

## ICAP Redirect

### Configuration
```
# ICAP: Internet Content Adaptation Protocol — offloads content scanning to external server
# SRX forwards HTTP/HTTPS content to ICAP server (DLP, antivirus, content filtering)

set services icap-redirect profile ICAP-SCAN server SCAN-SERVER host 10.5.0.50
set services icap-redirect profile ICAP-SCAN server SCAN-SERVER port 1344
set services icap-redirect profile ICAP-SCAN server SCAN-SERVER reqmod-uri "icap://10.5.0.50:1344/reqmod"
set services icap-redirect profile ICAP-SCAN server SCAN-SERVER respmod-uri "icap://10.5.0.50:1344/respmod"

# Apply ICAP to security policy
set security policies from-zone trust to-zone untrust policy WEB-ACCESS then permit \
    application-services icap-redirect ICAP-SCAN

# Fallback if ICAP server is unreachable
set services icap-redirect profile ICAP-SCAN default-action permit    # or deny
set services icap-redirect profile ICAP-SCAN timeout 30
```

## Security Intelligence (SecIntel)

### Feed configuration
```
# SecIntel: Juniper's threat intelligence feed service
# Provides C&C server lists, infected host detection, malware domains

# Enable SecIntel
set services security-intelligence url https://services.juniper.net/secintel/
set services security-intelligence authentication auth-token <license-token>

# Configure feed profiles
set services security-intelligence profile CC-FEED category CC
set services security-intelligence profile CC-FEED rule CC-RULE match threat-level [7 8 9 10]
set services security-intelligence profile CC-FEED rule CC-RULE then action block close http redirect-url "http://blocked.corp.local"
set services security-intelligence profile CC-FEED rule CC-RULE then action log

# Infected host feed
set services security-intelligence profile INFECTED category Infected-Hosts
set services security-intelligence profile INFECTED rule INF-RULE match threat-level [5 6 7 8 9 10]
set services security-intelligence profile INFECTED rule INF-RULE then action block drop

# Custom feed (local threat list)
set services security-intelligence profile CUSTOM-FEED category custom-list BLOCKED-IPS
set security dynamic-address feed-server LOCAL-FEED url "https://feeds.corp.local/blocklist"
set security dynamic-address feed-server LOCAL-FEED update-interval 3600
set security dynamic-address address-name BLOCKED-IPS profile feed-server LOCAL-FEED
```

### Apply SecIntel to policy
```
set security policies from-zone trust to-zone untrust policy SEC-INTEL then permit \
    application-services security-intelligence-policy SEC-INTEL-POL

set services security-intelligence policy SEC-INTEL-POL CC CC-FEED
set services security-intelligence policy SEC-INTEL-POL Infected-Hosts INFECTED
```

### Verify SecIntel
```
show services security-intelligence statistics
show services security-intelligence feed-status
show security dynamic-address summary
show security dynamic-address address-name BLOCKED-IPS
```

## Encrypted Traffic Insights

### Without decryption
```
# Encrypted traffic insights (ETI): analyzes TLS metadata without decrypting
# Uses: TLS version, cipher suite, certificate details, JA3/JA3S fingerprints,
#        server name indication (SNI), certificate chain, validity period

# Enable ETI
set services encrypted-traffic-insights enable
set services encrypted-traffic-insights profile ETI-PROFILE action log
set services encrypted-traffic-insights profile ETI-PROFILE action block-on-threat

# ETI can detect:
#   - Known malicious JA3 fingerprints (malware families)
#   - Self-signed certificates used by C&C servers
#   - Expired or mismatched certificates
#   - TLS connections to known-bad SNIs
#   - Anomalous cipher suite negotiation

# Apply to policy
set security policies from-zone trust to-zone untrust policy ETI-CHECK then permit \
    application-services encrypted-traffic-insights ETI-PROFILE
```

## Policy-Based Routing on SRX

### Source-based routing
```
# Route traffic from specific sources via alternate next-hop
set routing-instances ALT-PATH instance-type forwarding
set routing-instances ALT-PATH routing-options static route 0.0.0.0/0 next-hop 10.0.2.1

set firewall family inet filter SRC-ROUTE term GUEST-TRAFFIC from source-address 192.168.100.0/24
set firewall family inet filter SRC-ROUTE term GUEST-TRAFFIC then routing-instance ALT-PATH
set firewall family inet filter SRC-ROUTE term DEFAULT then accept

set interfaces reth1 unit 0 family inet filter input SRC-ROUTE
```

### Application-based routing (with AppID)
```
# Route specific applications via different paths
# Requires unified policy with application identification

set security policies from-zone trust to-zone untrust policy ROUTE-VIDEO match application junos-youtube
set security policies from-zone trust to-zone untrust policy ROUTE-VIDEO then permit
set security policies from-zone trust to-zone untrust policy ROUTE-VIDEO then advanced-policy-based-routing \
    routing-instance VIDEO-PATH
```

## Logical Systems (LSYS)

### Configuration
```
# LSYS: virtual routers within a single SRX — each with independent zones, policies, NAT
# Used for multi-tenancy on a single physical device

# Create logical system
set logical-systems TENANT-A security zones security-zone trust
set logical-systems TENANT-A security zones security-zone trust interfaces reth1.100
set logical-systems TENANT-A security zones security-zone untrust
set logical-systems TENANT-A security zones security-zone untrust interfaces reth0.100

# Assign interfaces
set interfaces reth1 unit 100 vlan-id 100
set interfaces reth1 unit 100 family inet address 10.100.1.1/24
set logical-systems TENANT-A interfaces reth1 unit 100

# LSYS security policies
set logical-systems TENANT-A security policies from-zone trust to-zone untrust policy PERMIT-ALL \
    match source-address any destination-address any application any
set logical-systems TENANT-A security policies from-zone trust to-zone untrust policy PERMIT-ALL \
    then permit

# LSYS resource limits
set logical-systems TENANT-A security-profile TENANT-A-LIMITS
set security profile TENANT-A-LIMITS policy maximum 100
set security profile TENANT-A-LIMITS zone maximum 4
set security profile TENANT-A-LIMITS flow-session maximum 50000
set security profile TENANT-A-LIMITS nat-source-pool maximum 5
```

### Inter-LSYS communication
```
# Logical tunnel (lt) interfaces connect LSYS to each other or to root
set interfaces lt-0/0/0 unit 0 encapsulation ethernet
set interfaces lt-0/0/0 unit 0 peer-unit 1
set interfaces lt-0/0/0 unit 1 encapsulation ethernet
set interfaces lt-0/0/0 unit 1 peer-unit 0

# Assign lt units to LSYS
set logical-systems TENANT-A interfaces lt-0/0/0 unit 0
set logical-systems TENANT-B interfaces lt-0/0/0 unit 1
```

### Enter LSYS context
```
set cli logical-system TENANT-A        # enter LSYS config context
show security policies                  # shows TENANT-A policies only
clear cli logical-system               # return to root
```

## Tenant Systems

### Configuration
```
# Tenant systems: lightweight multi-tenancy (simpler than LSYS)
# Share routing instance, separate security domains

set tenants CUSTOMER-1 security zones security-zone trust interfaces reth2.200
set tenants CUSTOMER-1 security zones security-zone untrust interfaces reth0.200
set tenants CUSTOMER-1 security policies from-zone trust to-zone untrust policy WEB \
    match source-address any destination-address any application junos-http
set tenants CUSTOMER-1 security policies from-zone trust to-zone untrust policy WEB then permit

# Resource profile
set system security-profile CUST1-PROFILE tenant CUSTOMER-1
set system security-profile CUST1-PROFILE policy maximum 50
set system security-profile CUST1-PROFILE zone maximum 4
```

## Juniper ATP Cloud

### Configuration
```
# ATP Cloud: cloud-based advanced threat prevention
# File inspection (malware sandboxing), C&C detection, threat intelligence

# Enroll SRX with ATP Cloud
request services advanced-anti-malware enroll

# Configure threat prevention policy
set services advanced-anti-malware policy ATP-POLICY http inspection-profile ATP-HTTP
set services advanced-anti-malware policy ATP-POLICY http action block
set services advanced-anti-malware policy ATP-POLICY smtp inspection-profile ATP-SMTP
set services advanced-anti-malware policy ATP-POLICY smtp action permit notification log

# Configure verdicts
set services advanced-anti-malware policy ATP-POLICY verdict-threshold 7    # block score >= 7

# Apply to security policy
set security policies from-zone trust to-zone untrust policy INSPECT then permit \
    application-services advanced-anti-malware-policy ATP-POLICY
```

### Sky ATP (legacy name)
```
# Sky ATP is the previous branding for ATP Cloud
# Same functionality: cloud-based sandboxing and threat prevention
# Configuration is identical to ATP Cloud
# Check enrollment status:
show services advanced-anti-malware status
show services advanced-anti-malware statistics
```

### Threat prevention
```
# Threat prevention integrates SecIntel + ATP + custom feeds
set services security-intelligence profile THREAT-PREV category all
set services security-intelligence profile THREAT-PREV rule ALL-THREATS match threat-level [1 2 3 4 5 6 7 8 9 10]
set services security-intelligence profile THREAT-PREV rule BLOCK-HIGH match threat-level [8 9 10]
set services security-intelligence profile THREAT-PREV rule BLOCK-HIGH then action block drop
set services security-intelligence profile THREAT-PREV rule LOG-MED match threat-level [4 5 6 7]
set services security-intelligence profile THREAT-PREV rule LOG-MED then action permit
set services security-intelligence profile THREAT-PREV rule LOG-MED then action log
```

## Security Director

### Overview
```
# Security Director: centralized security management platform (Junos Space application)
# Manages: security policies, NAT, IPS/IDP, VPN, UTM, SSL proxy across multiple SRX
# Provides: policy visualization, change management, compliance reporting

# Connect SRX to Security Director
set system services netconf ssh
set system services netconf rfc-compliant
set system services rest http port 8080
set system services rest https port 8443

# Security Director uses NETCONF/REST to push configuration
# SRX acts as a managed device — no agent required
```

## Verification Commands

### SSL proxy
```
show services ssl proxy statistics                      # session counts, cache hits
show services ssl proxy certificate-cache               # cached server certificates
show services ssl proxy session                         # active SSL proxy sessions
show services ssl proxy errors                          # error counts (handshake failures)
show services ssl proxy whitelist                       # exempted domains
```

### SecIntel and threat prevention
```
show services security-intelligence statistics          # feed hit counts
show services security-intelligence feed-status         # feed update status
show services advanced-anti-malware status              # ATP Cloud enrollment
show services advanced-anti-malware statistics          # file inspection stats
show security dynamic-address summary                   # dynamic address entries
show security dynamic-address address-name <name>       # specific feed entries
```

### LSYS and tenants
```
show logical-systems                                    # list all LSYS
show logical-systems TENANT-A security policies         # LSYS policy summary
show logical-systems TENANT-A security zones            # LSYS zone config
show system security-profile all                        # resource profiles and usage
show tenants                                            # list all tenants
show tenants CUSTOMER-1 security policies               # tenant policy summary
```

### Encrypted traffic insights
```
show services encrypted-traffic-insights statistics     # ETI match counts
show services encrypted-traffic-insights threat-info    # detected threats
show security flow session extensive | match ja3        # JA3 fingerprints in sessions
```

## Tips

- SSL forward proxy requires deploying the CA cert to all clients — without it, every HTTPS site shows a certificate warning
- Always whitelist banking, healthcare, and government sites from SSL inspection — legal and privacy requirements
- ICAP default-action should be `permit` in most cases — blocking all traffic when the ICAP server is down is usually worse than missing a scan
- SecIntel feeds require a valid license and internet connectivity — verify feed-status regularly
- Encrypted traffic insights is useful when full SSL decryption is not feasible (privacy constraints, performance limits)
- LSYS provides stronger isolation than tenant systems — use LSYS when tenants need independent routing tables
- Tenant systems are lighter weight and easier to manage — use when tenants share routing but need separate security domains
- ATP Cloud file inspection adds latency — sandbox analysis can take 30+ seconds for unknown files
- JA3 fingerprints can identify specific malware families without decrypting any payload
- Security Director is the recommended approach for managing more than 3-5 SRX devices

## See Also

- junos-security-policies, junos-nat-security, junos-screens, junos-ha-security

## References

- [Juniper TechLibrary — SSL Proxy](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/ssl-proxy-overview.html)
- [Juniper TechLibrary — Security Intelligence](https://www.juniper.net/documentation/us/en/software/junos/security-intelligence/topics/concept/security-intelligence-overview.html)
- [Juniper TechLibrary — ICAP Redirect](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/icap-redirect-overview.html)
- [Juniper TechLibrary — Logical Systems](https://www.juniper.net/documentation/us/en/software/junos/logical-systems-security/topics/concept/logical-systems-security-overview.html)
- [Juniper TechLibrary — Tenant Systems](https://www.juniper.net/documentation/us/en/software/junos/tenant-systems/topics/concept/tenant-systems-overview.html)
- [Juniper TechLibrary — ATP Cloud](https://www.juniper.net/documentation/us/en/software/junos/advanced-threat-prevention/topics/concept/atp-cloud-overview.html)
- [Juniper TechLibrary — Encrypted Traffic Insights](https://www.juniper.net/documentation/us/en/software/junos/encrypted-traffic-insights/topics/concept/encrypted-traffic-insights-overview.html)
- [Juniper Security Director Documentation](https://www.juniper.net/documentation/product/us/en/security-director/)
