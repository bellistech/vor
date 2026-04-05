# Cisco Umbrella (DNS-Layer Security, SIG, and CASB)

Cloud-delivered security platform providing DNS-layer protection, Secure Internet Gateway (SIG) full proxy, CASB for shadow IT discovery, and threat intelligence via Cisco Talos.

## Architecture Overview

### Core Components

```
# Umbrella operates at multiple enforcement points:
#
# 1. DNS-Layer Security (all plans)
#    - Recursive DNS resolvers at anycast IPs: 208.67.222.222, 208.67.220.220
#    - Inspects DNS queries before connection is established
#    - Blocks requests to malicious/unwanted domains at resolution time
#
# 2. Secure Internet Gateway (SIG) — full proxy
#    - Cloud-based proxy for HTTP/HTTPS traffic inspection
#    - SSL/TLS decryption for deep content inspection
#    - DLP, file inspection, AMP (Advanced Malware Protection)
#
# 3. Cloud Access Security Broker (CASB)
#    - Shadow IT discovery via DNS and proxy logs
#    - App risk scoring and granular app controls
#    - Inline and out-of-band modes
#
# 4. Remote Browser Isolation (RBI)
#    - Renders risky sites in cloud containers
#    - Only safe visual stream sent to user browser
#    - Zero-day and drive-by download protection

# Traffic flow (DNS-layer):
#   Client -> Umbrella Anycast DNS -> Policy check -> Allow/Block
#   If allowed: client connects directly to destination
#
# Traffic flow (SIG full proxy):
#   Client -> Umbrella Cloud Proxy -> SSL decrypt -> Policy check
#   -> Content inspection (AMP, DLP) -> Allow/Block -> Destination
```

### Deployment Models

```
# Model 1: DNS-Only (lightest touch)
# - Point DNS resolvers to 208.67.222.222 / 208.67.220.220
# - No agent required for network-wide protection
# - Per-network or per-device policies via registration
# - Protects all ports and protocols at DNS layer

# Model 2: DNS + Intelligent Proxy (selective proxy)
# - DNS-layer blocks known-bad domains
# - "Grey" domains (uncategorized/risky) redirected to intelligent proxy
# - Proxy performs URL filtering and file inspection on grey traffic only
# - Lighter than full proxy; no SSL decrypt on all traffic

# Model 3: SIG Full Proxy (maximum visibility)
# - All web traffic tunneled through Umbrella cloud proxy
# - Full SSL/TLS decryption (requires root CA deployment)
# - DLP content inspection, AMP file analysis
# - PAC file, proxy chaining, or tunnel (IPsec/GRE) based forwarding

# Model 4: Roaming Client (off-network users)
# - Umbrella Roaming Client installed on endpoints
# - Enforces DNS policy regardless of network location
# - AnyConnect Umbrella module for integrated deployment
# - Registers device identity for per-device policy
```

## DNS Policy Configuration

### Creating DNS Policies

```
# Dashboard: Policies > DNS Policies > Add Policy

# Policy components:
# 1. Identities — what the policy applies to
#    - Networks (registered public IPs)
#    - Roaming computers (Umbrella agent)
#    - Network devices (Meraki, ASA, ISR via integration)
#    - AD users and groups (via AD Connector or SAML)
#    - Chromebook users, iOS devices, etc.

# 2. Content categories — what to block
#    - Security categories (always recommended to block):
#      Malware, C2 callbacks, phishing, cryptomining
#    - Content categories:
#      Adult, gambling, drugs, weapons, etc.
#    - Custom allow/block lists (manual domain entries)

# 3. Security settings
#    - Malware protection (block domains hosting malware)
#    - Newly seen domains (block domains < 24h old)
#    - C2 callbacks (block known command and control)
#    - Phishing (block credential harvesting sites)
#    - Cryptomining (block browser-based miners)
#    - DNS tunneling detection (anomalous query patterns)

# 4. Application settings
#    - Block or allow specific SaaS apps (e.g., Dropbox, WeTransfer)
#    - App categories: file sharing, social networking, streaming

# Policy evaluation order:
# - Policies evaluated top-to-bottom; first match wins
# - More specific identities should be higher in the list
# - Default policy at bottom catches everything else
```

### Destination Lists (Custom Allow/Block)

```
# Dashboard: Policies > Destination Lists

# Global block list — applies to all policies
# Global allow list — bypasses all policy checks
# Per-policy lists — scoped to specific policies

# Adding entries:
# - Individual domains: malicious-site.example.com
# - URLs (for proxy policies): https://malicious-site.example.com/path
# - IPs (for IP-layer enforcement): 198.51.100.0/24

# Best practices:
# - Use allow lists sparingly (bypasses security inspection)
# - Block newly registered domains (< 30 days) for high-security orgs
# - Import threat intel feeds into block lists via API
# - Review block page bypass requests regularly
```

## Content Categories and Filtering

### Security Categories (Block Recommended)

```
# Category                     Description
# ─────────────────────────────────────────────────────────────────
# Malware                      Domains distributing malware
# Command and Control          Known C2 infrastructure
# Phishing                     Credential harvesting sites
# Cryptomining                 In-browser cryptocurrency mining
# Newly Seen Domains           Domains observed for < 24 hours
# Dynamic DNS                  Free DDNS services (often abused)
# Potentially Harmful          Domains with suspicious indicators

# These categories use Cisco Talos threat intelligence
# Updated continuously; no manual signature updates needed
```

### Content Categories

```
# High Risk (commonly blocked in enterprise):
# - Adult Content, Pornography
# - Gambling
# - Illegal Activities, Drugs
# - Proxy/Anonymizer (blocks VPN/proxy bypass tools)
# - P2P/File Sharing (unauthorized data transfer)

# Moderate Risk (policy-dependent):
# - Social Networking (Facebook, Twitter, etc.)
# - Streaming Media (Netflix, YouTube)
# - Gaming
# - Web-based Email (personal email services)
# - Chat/Messaging (WhatsApp Web, Telegram)

# Custom categories:
# - Create custom categories by domain patterns
# - Assign custom categories to policies
# - Override default categorization for specific domains
```

## App Discovery and Control (CASB)

### Shadow IT Discovery

```
# Dashboard: Reporting > App Discovery

# Discovery sources:
# - DNS logs (all plans) — identifies apps by domain patterns
# - Proxy logs (SIG plans) — deeper URL-level app identification
# - API connectors — out-of-band cloud app monitoring

# App risk scoring (1-10, higher = riskier):
# Factors considered:
# - Data breach history
# - Encryption standards (TLS version, cert quality)
# - Compliance certifications (SOC2, ISO 27001, GDPR)
# - Data retention and privacy policies
# - MFA support, SSO support
# - Admin audit logging capability

# Common discoveries:
# - Unauthorized file sharing (personal Dropbox, WeTransfer)
# - Unapproved SaaS apps (Trello, Notion, Airtable)
# - Shadow cloud infrastructure (personal AWS/GCP accounts)
# - Unsanctioned communication tools (Discord, Telegram)
```

### App Blocking and Control

```
# Block by application:
# Dashboard: Policies > DNS or Web Policy > Application Settings
# - Search for specific app (e.g., "Dropbox Personal")
# - Set action: Block, Allow, or Warn
# - Granular controls (SIG only): block uploads but allow downloads

# Block by category:
# - File Sharing & Storage
# - Social Networking
# - Cloud Infrastructure
# - Developer Tools
# - Set policy per identity group (e.g., block social for finance dept)

# Tenant restrictions (SIG full proxy):
# - Allow corporate Office 365 tenant, block personal
# - Allow corporate Google Workspace, block consumer Gmail
# - Inject tenant restriction headers in proxied traffic
# - Requires SSL decryption enabled
```

## Intelligent Proxy

### Selective Inspection

```
# The intelligent proxy sits between DNS-layer and full SIG proxy
# It inspects traffic only for domains that are "grey" — not clearly
# safe or malicious

# How it works:
# 1. DNS query arrives at Umbrella resolver
# 2. Domain reputation checked:
#    - Known safe (e.g., microsoft.com) -> Allow, no proxy
#    - Known bad (e.g., malware C2) -> Block at DNS
#    - Uncategorized / risky -> Redirect to intelligent proxy
# 3. Intelligent proxy fetches content and inspects:
#    - URL reputation check
#    - AMP file scanning (downloads)
#    - Cisco Threat Grid sandboxing (suspicious files)
#    - Retrospective alerts if file later found malicious

# Enable intelligent proxy:
# Dashboard: Policies > DNS Policy > edit policy
# -> Enable Intelligent Proxy
# -> Enable File Inspection (AMP)
# -> File types to inspect: executables, archives, documents

# SSL decryption for intelligent proxy:
# - Deploy Umbrella root CA certificate to endpoints
# - Required for HTTPS content inspection
# - Bypass list for pinned-certificate sites (banking, healthcare)
```

## Data Loss Prevention (DLP)

### DLP Policies (SIG Required)

```
# Dashboard: Policies > Web Policy > DLP

# Built-in data identifiers:
# - Credit card numbers (PCI DSS)
# - Social Security numbers (PII)
# - Health records (HIPAA)
# - Financial data patterns
# - Source code patterns
# - Custom regex patterns

# DLP policy configuration:
# 1. Select data identifiers to detect
# 2. Set threshold (e.g., block if > 5 SSNs in a single upload)
# 3. Define action: Block, Warn, Monitor (log only)
# 4. Apply to identities (user groups, networks)
# 5. Set direction: uploads only, downloads only, or both

# DLP inspection requires:
# - SIG full proxy or intelligent proxy
# - SSL decryption enabled for HTTPS
# - File type inspection for document scanning

# Supported protocols:
# - HTTP/HTTPS uploads and downloads
# - FTP uploads (when proxied)
# - Cloud app uploads (via CASB inline controls)
```

## Remote Browser Isolation (RBI)

```
# RBI renders web content in cloud containers, sending only a
# visual stream (pixels) to the user's browser

# Use cases:
# - Uncategorized/risky domains (redirect grey traffic to RBI)
# - Phishing protection (users can view but not enter credentials)
# - Zero-day protection (malicious scripts execute in disposable container)
# - Controlled file downloads (scan before allowing download)

# Configuration:
# Dashboard: Policies > DNS or Web Policy
# Action for risky/uncategorized domains: Isolate (RBI)

# RBI modes:
# - Full isolation: all rendering in cloud, pixel-push to client
# - Read-only isolation: user can view but not interact (no clicks, no typing)
# - File download control: block, scan-then-allow, or block-all

# Session behavior:
# - Each RBI session is a disposable container
# - Container destroyed after session timeout (default: 30 min)
# - No persistent state between RBI sessions
# - Clipboard disabled by default (configurable)
```

## Umbrella Roaming Client

### Deployment

```bash
# The roaming client enforces Umbrella policy off-network
# Available as standalone agent or AnyConnect module

# Standalone roaming client:
# - Download from Dashboard: Deployments > Roaming Computers
# - Each org has unique OrgID baked into installer
# - MSI package for Windows: UmbrellaRoamingClient_x64.msi
# - PKG for macOS: UmbrellaRoamingClient.pkg

# Silent install (Windows):
msiexec /i UmbrellaRoamingClient_x64.msi /qn
# Verify installation:
# Check service: Umbrella_RC (UmbrellaRoamingClient)
# Registry: HKLM\SOFTWARE\OpenDNS

# AnyConnect Umbrella module:
# - Deploy via AnyConnect profile with Umbrella module enabled
# - OrgInfo.json contains registration data:
# {
#   "organizationId": "1234567",
#   "fingerprint": "abcdef123456",
#   "userId": "7654321"
# }

# macOS deployment (MDM):
# Deploy PKG via Jamf, Mosyle, Kandji, etc.
# Post-install verification:
# /opt/cisco/secureclient/bin/umbrella_diagnostic
```

### How the Roaming Client Works

```
# On-network detection:
# 1. Client checks if internal DNS resolvers are reachable
# 2. If on corporate network (internal resolvers answer): client goes passive
#    -> Corporate DNS policies apply (via network registration)
# 3. If off-network: client activates
#    -> Intercepts DNS queries at the OS level
#    -> Forwards to Umbrella resolvers (208.67.222.222/220)
#    -> Encrypts DNS queries (DNSCrypt)
#    -> Device identity sent with each query for per-device policy

# Internal domain handling:
# - Configure internal domains in Dashboard
# - Client bypasses Umbrella for internal domains
# - Routes internal DNS to local resolver instead
# - Prevents split-horizon DNS breakage

# VPN compatibility:
# - Compatible with most VPN clients (AnyConnect, GlobalProtect, etc.)
# - VPN split-tunnel: roaming client handles non-VPN DNS
# - VPN full-tunnel: roaming client defers to VPN DNS resolver
```

## Network Device Integration

### Meraki MX Integration

```
# Meraki Dashboard: Security & SD-WAN > Threat Protection
# Enable Umbrella integration:
# 1. Link Meraki org to Umbrella org via API key
# 2. Dashboard: Organization > Configure > Umbrella
# 3. Enter Umbrella API credentials (Network Devices key)
# 4. Meraki MX automatically registers as network identity
# 5. DNS traffic forwarded to Umbrella with device identity tags

# Meraki MR (wireless) integration:
# - Wireless > Firewall & traffic shaping > Layer 7 rules
# - DNS traffic from SSID forwarded to Umbrella
# - Per-SSID Umbrella policy assignment
```

### Cisco ASA / Firepower Integration

```
# ASA Umbrella Connector (9.10+):
# Configure DNS inspection to redirect to Umbrella

# ASA configuration:
# umbrella-global
#   token <REGISTRATION_TOKEN>
#   local-domain-bypass "internal.example.com"
#   dnscrypt
# !
# interface GigabitEthernet0/0
#   umbrella in

# Firepower Threat Defense (FTD):
# Objects > Security Intelligence > DNS Policy
# Add Umbrella connector via API token
# Apply DNS policy to access control policy

# Registration token:
# Dashboard: Deployments > Network Devices
# Add > Cisco ASA/Firepower
# Copy token for device configuration
```

### IOS/IOS-XE Router Integration

```
# ISR/CSR routers (IOS-XE 16.10+):
# Uses DNS-layer integration via Umbrella connector

# Configuration:
parameter-map type umbrella global
  token <REGISTRATION_TOKEN>
  local-domain-bypass "corp.example.com"
  dnscrypt
!
interface GigabitEthernet0/0
  umbrella out

# Verify:
show umbrella config
show umbrella deviceid
show umbrella dnscrypt
# Output shows registration status, device-id, and resolver connectivity
```

## API Integration

### Umbrella Management API

```bash
# Base URL: https://api.umbrella.com/v2

# Authentication: OAuth2 with API key and secret
# Dashboard: Admin > API Keys > Create API Key
# Scopes: policies, reports, deployments, admin

# Get OAuth2 token:
curl -s -X POST "https://api.umbrella.com/auth/v2/token" \
  -u "API_KEY:API_SECRET" \
  -d "grant_type=client_credentials" | jq .access_token

# List destination lists:
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://api.umbrella.com/policies/v2/destinationlists"

# Add domain to block list:
curl -s -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  "https://api.umbrella.com/policies/v2/destinationlists/$LIST_ID/destinations" \
  -d '[{"destination": "malicious.example.com", "comment": "threat intel"}]'

# Get activity report (last 24h):
curl -s -H "Authorization: Bearer $TOKEN" \
  "https://api.umbrella.com/reports/v2/activity?from=-1days&to=now&limit=100"
```

### Umbrella Enforcement API

```bash
# Enforcement API — push block events from external systems
# Use case: SIEM/SOAR integration to auto-block IOCs

# Endpoint: https://s-platform.api.opendns.com/1.0/events
# Authentication: customer-specific API key (in URL parameter)

# Push a block event:
curl -s -X POST \
  "https://s-platform.api.opendns.com/1.0/events?customerKey=$ENFORCEMENT_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "alertTime": "2026-04-05T12:00:00.000Z",
    "deviceId": "firewall-01",
    "deviceVersion": "1.0",
    "eventTime": "2026-04-05T11:59:00.000Z",
    "protocolVersion": "1.0a",
    "providerName": "SIEM Integration",
    "dstDomain": "evil.example.com",
    "dstUrl": "https://evil.example.com/payload",
    "eventType": "malware",
    "eventSeverity": "high"
  }'

# Delete a domain from enforcement list:
curl -s -X DELETE \
  "https://s-platform.api.opendns.com/1.0/domains/evil.example.com?customerKey=$ENFORCEMENT_KEY"
```

## Umbrella Investigate

### Threat Intelligence Lookups

```bash
# Investigate API — enrichment and threat intel
# Base URL: https://investigate.api.umbrella.com
# Auth: Bearer token (Investigate API key)

# Domain risk score:
curl -s -H "Authorization: Bearer $INVESTIGATE_TOKEN" \
  "https://investigate.api.umbrella.com/domains/score/suspicious.example.com"
# Returns: risk score -100 (safe) to +100 (malicious)

# Domain categorization:
curl -s -H "Authorization: Bearer $INVESTIGATE_TOKEN" \
  "https://investigate.api.umbrella.com/domains/categorization/suspicious.example.com"
# Returns: category labels + security status (blocked/allowed)

# WHOIS history:
curl -s -H "Authorization: Bearer $INVESTIGATE_TOKEN" \
  "https://investigate.api.umbrella.com/whois/suspicious.example.com"

# DNS query volume (passive DNS):
curl -s -H "Authorization: Bearer $INVESTIGATE_TOKEN" \
  "https://investigate.api.umbrella.com/dnsdb/name/a/suspicious.example.com"

# Co-occurrences (domains queried together):
curl -s -H "Authorization: Bearer $INVESTIGATE_TOKEN" \
  "https://investigate.api.umbrella.com/recommendations/name/suspicious.example.com"

# Security information (ASN, geolocation, threat indicators):
curl -s -H "Authorization: Bearer $INVESTIGATE_TOKEN" \
  "https://investigate.api.umbrella.com/security/name/suspicious.example.com"
```

### Investigate Console (Dashboard)

```
# Dashboard: Investigate > Domain Search
#
# Key investigation views:
# - Timeline: DNS query volume over time (spikes = campaign activity)
# - Co-occurrences: domains frequently queried alongside target
# - Related domains: shared infrastructure, registrant, IP space
# - WHOIS: registrant info, creation date, registrar
# - IP geolocation: hosting location, ASN ownership
# - Passive DNS: historical A/AAAA/NS/MX records
# - Sample artifacts: malware samples associated with domain
#
# Verdict indicators:
# - Green shield: known safe / high confidence benign
# - Grey shield: uncategorized / insufficient data
# - Red shield: known malicious / high confidence threat
# - Risk score: -100 to +100 (negative = safe, positive = risky)
```

## Reporting and Logging

### Built-in Reports

```
# Dashboard: Reporting

# Activity Search:
# - Search DNS and proxy logs by domain, identity, category
# - Filter by action: allowed, blocked, proxied
# - Time range: last hour to last 30 days
# - Export to CSV

# Security Overview:
# - Top threats blocked (by category)
# - Top blocked domains
# - Top identities triggering blocks
# - Threat trend over time

# App Discovery Report:
# - All SaaS apps detected in DNS/proxy traffic
# - Risk score per app
# - Number of users/requests per app
# - Block/allow recommendation

# Destination Lists Report:
# - Domains in custom block/allow lists
# - Hit counts per domain
# - Last query time

# Total Requests:
# - Overall DNS query volume over time
# - Breakdown by allowed/blocked/proxied
```

### Log Export and SIEM Integration

```bash
# Amazon S3 log export:
# Dashboard: Admin > Log Management > Enable S3 Export
# - Logs written to your S3 bucket every 10 minutes
# - Format: CSV or JSON (configurable)
# - Fields: timestamp, identity, source IP, domain, categories,
#           action (allowed/blocked), query type, response code

# Syslog integration (SIG plans):
# - Configure syslog destination in Dashboard
# - CEF or LEEF format
# - Real-time streaming to SIEM (Splunk, QRadar, ArcSight)

# Splunk integration:
# - Install "Cisco Umbrella Add-on for Splunk" from Splunkbase
# - Configure S3 input to ingest Umbrella logs from S3 bucket
# - Alternatively use Cisco Umbrella Reporting API for pull-based ingestion

# Log fields (DNS log):
# Timestamp, PolicyIdentity, Identities, InternalIP, ExternalIP,
# Action, QueryType, ResponseCode, Domain, Categories,
# PolicyIdentityType, IdentityTypes, BlockedCategories
```

## Tips

- Start with DNS-only deployment to gain immediate visibility with zero endpoint changes. Simply point network DNS to 208.67.222.222 and 208.67.220.220.
- Register your public egress IPs as network identities so Umbrella can apply per-network policies without any agent deployment.
- Always block the security categories (malware, C2, phishing, cryptomining, newly seen domains) in every policy. There is no legitimate reason to allow these.
- Use the intelligent proxy for uncategorized domains rather than blocking them outright. This provides inspection without over-blocking.
- Deploy the Umbrella root CA to managed endpoints before enabling SSL decryption. Maintain a bypass list for certificate-pinned applications (banking, healthcare portals).
- Internal domains must be configured in the roaming client settings to avoid breaking split-horizon DNS and internal application access.
- Integrate Umbrella with your SIEM via S3 log export or the Reporting API. DNS logs are a goldmine for threat hunting (DGA detection, beaconing, data exfiltration patterns).
- Use the Enforcement API to create automated block workflows from your SOAR platform, feeding IOCs directly into Umbrella policy.
- Review the App Discovery report monthly to identify shadow IT and enforce corporate app standards.
- For compliance (PCI, HIPAA), enable DLP policies on the SIG proxy and retain logs for the required period via S3 export.

## See Also

- dns, tls, zero-trust, cisco-ise, cisco-ftd, waf, network-security-infra

## References

- [Cisco Umbrella Documentation](https://docs.umbrella.com/)
- [Cisco Umbrella Deployment Guide](https://docs.umbrella.com/deployment-umbrella/docs)
- [Umbrella API Documentation](https://developer.cisco.com/docs/cloud-security/)
- [Umbrella Investigate API](https://docs.umbrella.com/investigate-api/docs)
- [Umbrella Roaming Client Admin Guide](https://docs.umbrella.com/deployment-umbrella/docs/appx-a-roaming-client-deployment)
- [Cisco Talos Intelligence](https://talosintelligence.com/)
- [Cisco Umbrella SIG User Guide](https://docs.umbrella.com/deployment-umbrella/docs/welcome-to-cisco-umbrella)
- [Umbrella + Meraki Integration Guide](https://documentation.meraki.com/MR/Other_Topics/Cisco_Umbrella_Integration_Guide)
- [Umbrella + ASA Configuration Guide](https://www.cisco.com/c/en/us/td/docs/security/asa/asa910/configuration/firewall/asa-910-firewall-config/inspect-dns.html)
