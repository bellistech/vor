# Juniper ATP Cloud / Sky ATP

Advanced threat prevention cloud service that integrates with SRX firewalls for malware analysis, C&C detection, and threat intelligence feeds.

## ATP Cloud Architecture

```
                        Juniper ATP Cloud
                    ┌─────────────────────────┐
                    │  Threat Intelligence     │
                    │  ├── C&C feeds           │
                    │  ├── GeoIP feeds         │
                    │  ├── Infected host feeds │
                    │  └── Custom feeds        │
                    │                          │
                    │  Cloud Sandbox           │
                    │  ├── Static analysis     │
                    │  ├── Dynamic analysis    │
                    │  └── ML classification   │
                    │                          │
                    │  Encrypted Traffic       │
                    │  Analysis (ETA)          │
                    └──────────┬───────────────┘
                               │ SecIntel
                    ┌──────────▼───────────────┐
                    │  SRX Series Firewall      │
                    │  ├── SecIntel feed ingest │
                    │  ├── File inspection      │
                    │  └── Policy enforcement   │
                    └───────────────────────────┘
```

## ATP Cloud Enrollment

### Enroll SRX to ATP Cloud

```
# Step 1 — Generate enrollment token on ATP Cloud web portal
#   Enroll → Enrollment → Generate Token
#   Token format: XXXXXXXXXXXXXXXX (16-char)

# Step 2 — Configure ATP Cloud connection on SRX
set services advanced-anti-malware connection url https://amer.sky.junipersecurity.net
set services advanced-anti-malware connection authentication tls-profile ATP-TLS-PROFILE

# Step 3 — Enroll SRX
request services advanced-anti-malware enroll token XXXXXXXXXXXXXXXX

# Step 4 — Verify enrollment
show services advanced-anti-malware status
```

### ATP Cloud realm URLs

```
Americas:    https://amer.sky.junipersecurity.net
EMEA:        https://emea.sky.junipersecurity.net
APAC:        https://apac.sky.junipersecurity.net
Canada:      https://canada.sky.junipersecurity.net
```

### TLS profile for ATP Cloud

```
set security pki ca-profile AMER-SKY ca-identity "GeoTrust RSA CA 2018"
set security pki ca-profile AMER-SKY enrollment url https://amer.sky.junipersecurity.net

set services ssl initiation profile ATP-TLS-PROFILE
set services ssl initiation profile ATP-TLS-PROFILE trusted-ca AMER-SKY
set services ssl initiation profile ATP-TLS-PROFILE actions crl disable
```

## Threat Intelligence Feeds (SecIntel)

### Feed types

```
Feed Type            Description                          Default Action
─────────────────────────────────────────────────────────────────────────
C&C                  Known command-and-control servers     Block
Malware              Known malware distribution sites      Block
GeoIP                Country-based IP classification       Log / Block
Infected Hosts       Internal hosts with suspicious traffic  Quarantine
Custom Feeds         User-defined threat indicators        Configurable
DNS Feeds            Malicious domain names                Sinkhole
```

### Configure SecIntel feed policy

```
# Enable SecIntel on SRX
set services security-intelligence url https://amer.sky.junipersecurity.net/api/v1/manifest.xml
set services security-intelligence authentication tls-profile ATP-TLS-PROFILE

# C&C feed profile
set services security-intelligence profile CC-PROFILE category CC
set services security-intelligence profile CC-PROFILE rule CC-RULE match threat-level [7 8 9 10]
set services security-intelligence profile CC-PROFILE rule CC-RULE then action block drop
set services security-intelligence profile CC-PROFILE rule CC-RULE then log

# GeoIP feed profile
set services security-intelligence profile GEOIP-BLOCK category GeoIP
set services security-intelligence profile GEOIP-BLOCK rule BLOCK-COUNTRIES match threat-level [1-10]
set services security-intelligence profile GEOIP-BLOCK rule BLOCK-COUNTRIES then action block drop
```

### Apply SecIntel policy to security policy

```
set security policies from-zone TRUST to-zone UNTRUST policy INTERNET-ACCESS match source-address any
set security policies from-zone TRUST to-zone UNTRUST policy INTERNET-ACCESS match destination-address any
set security policies from-zone TRUST to-zone UNTRUST policy INTERNET-ACCESS match application any
set security policies from-zone TRUST to-zone UNTRUST policy INTERNET-ACCESS then permit
set security policies from-zone TRUST to-zone UNTRUST policy INTERNET-ACCESS then permit application-services security-intelligence-policy SEC-INTEL-POLICY

set services security-intelligence policy SEC-INTEL-POLICY CC CC-PROFILE
set services security-intelligence policy SEC-INTEL-POLICY GeoIP GEOIP-BLOCK
```

### Infected host feed

```
# Infected host detection uses lateral movement + C&C callbacks
set services security-intelligence profile INFECTED-HOST-PROFILE category Infected-Hosts
set services security-intelligence profile INFECTED-HOST-PROFILE rule IH-RULE match threat-level [7 8 9 10]
set services security-intelligence profile INFECTED-HOST-PROFILE rule IH-RULE then action block drop
set services security-intelligence profile INFECTED-HOST-PROFILE rule IH-RULE then log

# Add to SecIntel policy
set services security-intelligence policy SEC-INTEL-POLICY Infected-Hosts INFECTED-HOST-PROFILE
```

### Custom threat feeds

```
# Define custom feed source (CSV, STIX/TAXII)
set services security-intelligence custom-feed CUSTOM-BLOCKLIST url https://feeds.example.com/blocklist.csv
set services security-intelligence custom-feed CUSTOM-BLOCKLIST feed-interval 3600

# Profile for custom feed
set services security-intelligence profile CUSTOM-PROFILE category custom-feed
set services security-intelligence profile CUSTOM-PROFILE feed CUSTOM-BLOCKLIST
set services security-intelligence profile CUSTOM-PROFILE rule CUSTOM-RULE then action block drop
```

## Malware Analysis (Cloud Sandbox)

### Threat prevention profile

```
# Create threat prevention policy
set services advanced-anti-malware policy MALWARE-POLICY
set services advanced-anti-malware policy MALWARE-POLICY inspection-profile default
set services advanced-anti-malware policy MALWARE-POLICY verdict-threshold 7
set services advanced-anti-malware policy MALWARE-POLICY action block
set services advanced-anti-malware policy MALWARE-POLICY notification log
set services advanced-anti-malware policy MALWARE-POLICY fallback-options action permit

# File types for inspection
set services advanced-anti-malware policy MALWARE-POLICY file-types exe
set services advanced-anti-malware policy MALWARE-POLICY file-types dll
set services advanced-anti-malware policy MALWARE-POLICY file-types pdf
set services advanced-anti-malware policy MALWARE-POLICY file-types doc
set services advanced-anti-malware policy MALWARE-POLICY file-types jar
```

### Attach to security policy

```
set security policies from-zone TRUST to-zone UNTRUST policy WEB-ACCESS then permit application-services advanced-anti-malware-policy MALWARE-POLICY
```

### Verdict levels

```
Verdict Score   Threat Level    Recommended Action
─────────────────────────────────────────────────
1               Clean           Permit
2-3             Informational   Log
4-6             Suspicious      Log + Alert
7-8             Malicious       Block
9-10            Critical        Block + Quarantine
```

## Encrypted Traffic Analysis (ETA)

### Enable ETA

```
# ETA analyzes metadata of encrypted sessions without decryption
# Uses TLS handshake fingerprinting, certificate analysis, traffic patterns

set services advanced-anti-malware policy ETA-POLICY encrypted-traffic-insights enable
set services advanced-anti-malware policy ETA-POLICY encrypted-traffic-insights action block
set services advanced-anti-malware policy ETA-POLICY encrypted-traffic-insights verdict-threshold 7
```

### ETA with DNS-based detection

```
# DNS sinkhole for known malicious domains
set services dns-filtering profile DNS-FILTER default-action permit
set services dns-filtering profile DNS-FILTER category command-and-control action sinkhole
set services dns-filtering profile DNS-FILTER category malware action sinkhole
set services dns-filtering profile DNS-FILTER category phishing action sinkhole

set security policies from-zone TRUST to-zone UNTRUST policy DNS-POLICY then permit application-services dns-filtering-profile DNS-FILTER
```

## Allowlists and Blocklists

### Global allowlist

```
set services security-intelligence profile ALLOWLIST category AllowList
set services advanced-anti-malware policy MALWARE-POLICY whitelist-notification log
```

### Configure allowlist entries (ATP Cloud portal)

```
# In ATP Cloud web portal:
#   Configure → Allowlist → Add
#   - IP addresses
#   - URLs / Domains
#   - File hashes (SHA-256)
```

### Configure blocklist entries

```
# ATP Cloud portal: Configure → Blocklist → Add
# Or use custom feed:
set services security-intelligence custom-feed LOCAL-BLOCKLIST filename /var/db/scripts/blocklist.txt
set services security-intelligence custom-feed LOCAL-BLOCKLIST feed-interval 300
```

## Verification and Monitoring

### ATP Cloud status

```
show services advanced-anti-malware status           # enrollment status + connection
show services advanced-anti-malware statistics        # file submission counts + verdicts
show services advanced-anti-malware counters          # per-protocol counters
```

### SecIntel feed status

```
show services security-intelligence statistics
show services security-intelligence feed-status       # last update time per feed
show services security-intelligence category summary  # feed category stats
```

### Threat log

```
show log messages | match "RT_UTM\|IDP\|AAMW"
show security log                                     # security event log
```

### Real-time monitoring

```
monitor traffic interface ge-0/0/0                    # packet captures
show services advanced-anti-malware web-management status  # web portal connectivity
show security flow session                            # active session table
```

### Feed download verification

```
show services security-intelligence feed-status
# Look for:
#   Feed name: cc_ip, cc_url, cc_domain
#   Last update: <recent timestamp>
#   Status: OK
#   Feed count: <non-zero>
```

### Troubleshooting

```
# ATP Cloud enrollment fails
show services advanced-anti-malware status
request services advanced-anti-malware enroll token XXXXXXXXXXXXXXXX

# Check DNS resolution for ATP Cloud
run ping amer.sky.junipersecurity.net

# Check TLS connectivity
request security pki ca-certificate verify ca-profile AMER-SKY

# Debug SecIntel feed download
set system syslog file secintel any any
set system syslog file secintel match "SECINTEL"

# Traceoptions
set services security-intelligence traceoptions file secintel-trace
set services security-intelligence traceoptions flag all
```

## See Also

- junos-firewall-filters
- junos-routing-policy
- cisco-umbrella
- ids-ips
- threat-hunting
- threat-modeling

## References

- Juniper TechLibrary: ATP Cloud Administration Guide
- Juniper TechLibrary: SecIntel Configuration Guide
- Juniper TechLibrary: SRX Series Advanced Threat Prevention
- Juniper Day One: Deploying ATP Cloud with SRX
- MITRE ATT&CK: Command and Control (TA0011)
