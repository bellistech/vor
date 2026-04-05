# Web Security Proxy (Cisco WSA / Secure Web Appliance)

Dedicated proxy appliance for web traffic inspection: URL filtering, web reputation, anti-malware (AMP, Sophos, McAfee), HTTPS decryption, application visibility and control (AVC), DLP, authentication (NTLM, Kerberos, SAML), and cognitive threat analytics. Enforces acceptable use and protects against web-based threats.

## Architecture

### Deployment Overview

```
Users / Endpoints           WSA                           Internet
       |                     |                               |
       |--- HTTP/HTTPS ----->|                               |
       |                     | 1. Authenticate user          |
       |                     | 2. Check access policy        |
       |                     |    (URL category, reputation) |
       |                     | 3. Decrypt HTTPS (if enabled) |
       |                     | 4. Scan for malware           |
       |                     | 5. Check DLP policy           |
       |                     | 6. AVC inspection             |
       |                     |                               |
       |                     |--- Forward request ---------->|
       |                     |<-- Response ------------------|
       |                     |                               |
       |                     | 7. Scan response for malware  |
       |                     | 8. Log and report             |
       |                     |                               |
       |<-- Response --------|                               |
```

### Proxy Modes

| Mode | Description | Client Config | Typical Use |
|------|-------------|--------------|-------------|
| Explicit forward proxy | Client configured to use proxy | Browser proxy settings, PAC file | Enterprise desktops |
| Transparent proxy | Traffic intercepted without client config | WCCP, PBR, inline | BYOD, guest, unmanaged devices |
| Cloud proxy (Umbrella SIG) | Cloud-based proxy service | Umbrella roaming client, DNS redirect | Remote workers |

## Explicit Forward Proxy

### Browser Configuration

```
# Direct proxy setting (manual)
# Browser → Settings → Proxy
# HTTP Proxy: wsa.corp.example.com:3128
# HTTPS Proxy: wsa.corp.example.com:3128
# No Proxy: *.corp.example.com, 10.0.0.0/8, 172.16.0.0/12

# Group Policy (Windows)
# Computer Config > Admin Templates > Windows Components >
#   Internet Explorer > Proxy Settings
# Or: HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings
#   ProxyServer = "wsa.corp.example.com:3128"
#   ProxyEnable = 1
```

### PAC File (Proxy Auto-Configuration)

```javascript
// PAC file hosted at http://wpad.corp.example.com/wpad.dat
// Or configured via DHCP option 252 or DNS WPAD record

function FindProxyForURL(url, host) {
    // Direct access for internal hosts
    if (isPlainHostName(host))
        return "DIRECT";

    // Direct access for internal networks
    if (isInNet(host, "10.0.0.0", "255.0.0.0") ||
        isInNet(host, "172.16.0.0", "255.240.0.0") ||
        isInNet(host, "192.168.0.0", "255.255.0.0"))
        return "DIRECT";

    // Direct access for specific domains
    if (dnsDomainIs(host, ".corp.example.com"))
        return "DIRECT";

    // Primary proxy with failover
    return "PROXY wsa1.corp.example.com:3128; PROXY wsa2.corp.example.com:3128; DIRECT";
}
```

### WPAD (Web Proxy Auto-Discovery)

```
# DHCP method: option 252 = "http://wpad.corp.example.com/wpad.dat"

# DNS method: create CNAME or A record
# wpad.corp.example.com → WSA hosting the PAC file
# Browser queries: http://wpad.<domain>/wpad.dat

# WSA hosts the PAC file:
# Network > Proxy Settings > PAC File Hosting
# Upload or edit PAC file on the appliance
```

## Transparent Proxy with WCCP

### WCCP (Web Cache Communication Protocol)

```
! Router WCCP configuration (redirect web traffic to WSA)

! WCCP for HTTP (port 80)
ip wccp 80 redirect-list WEB-TRAFFIC group-list WSA-FARM
ip wccp 90 redirect-list HTTPS-TRAFFIC group-list WSA-FARM

! Access lists
ip access-list extended WEB-TRAFFIC
 permit tcp any any eq 80
ip access-list extended HTTPS-TRAFFIC
 permit tcp any any eq 443

! WSA farm (standard ACL matching WSA IPs)
ip access-list standard WSA-FARM
 permit 10.1.1.50
 permit 10.1.1.51

! Apply WCCP redirect on the user-facing interface
interface GigabitEthernet0/0
 description User VLAN
 ip wccp 80 redirect in
 ip wccp 90 redirect in

! Exclude WSA traffic from redirection (prevent loops)
ip access-list extended WEB-TRAFFIC
 deny   tcp host 10.1.1.50 any eq 80
 deny   tcp host 10.1.1.51 any eq 80
 permit tcp any any eq 80

! Verify WCCP
show ip wccp
show ip wccp 80 detail
```

### WCCP on WSA

```
! WSA transparent proxy configuration
wsa> networkconfig

! Network > Transparent Redirection > WCCP Router
! Add router IP: 10.1.1.1
! Service ID: 80 (HTTP), 90 (HTTPS)
! Redirect method: GRE (default) or L2 (same VLAN)
! Return method: GRE or L2

! WCCP load balancing:
! - Hash-based (source IP, destination IP, source port, destination port)
! - Mask-based (more flexible, preferred)
```

### Other Transparent Redirection Methods

| Method | How | Pros | Cons |
|--------|-----|------|------|
| WCCP | Router redirects via GRE/L2 | Standard, widely supported | GRE overhead, router config |
| PBR (Policy-Based Routing) | Route-map sends traffic to WSA | Simple, no protocol | No health check, no load balance |
| Layer 4 switch | L4 switch redirects ports 80/443 | High performance | Requires L4 switch |
| Inline / bridge | WSA sits inline on the wire | No infrastructure changes | Single point of failure |

## Access Policies

### Policy Processing Order

```
Request arrives at WSA
        |
        v
1. Identification Profile
   (Who is the user? Which policy group?)
        |
        v
2. Access Policy
   (Is the URL category allowed/blocked?)
        |
        v
3. HTTPS Decryption Policy (if HTTPS)
   (Decrypt, pass through, or drop?)
        |
        v
4. Anti-Malware / AMP Scanning
   (Is the content clean?)
        |
        v
5. AVC Policy
   (Is the application allowed?)
        |
        v
6. DLP Policy
   (Is data leaving appropriately?)
        |
        v
7. Routing Policy
   (How to reach the destination?)
        |
        v
8. Forward to internet / Block
```

### Access Policy Configuration

```
! Web Security Manager > Access Policies

! Policy structure:
! 1. URL Filtering — allow/block by URL category
! 2. Web Reputation — block by reputation score
! 3. Applications — AVC allow/block/throttle
! 4. Objects — block file types, MIME types, size limits
! 5. Anti-Malware — Sophos, McAfee, AMP settings

! URL categories (Cisco Talos):
! - Adult Content, Gambling, Malware, Phishing, Social Media,
!   Streaming, Web-Based Email, File Sharing, etc.
! - Custom URL categories: define your own lists

! URL category actions:
! - Allow — permit access (scan for malware)
! - Block — deny access (show block page)
! - Warn — show warning page, user can proceed
! - Monitor — allow but log (no user notification)
! - Quota — time-based access limits

! Web reputation score (WRS): -10.0 to +10.0
! - -10.0 to -6.0: Block (known malicious)
! - -6.0 to -3.0: Scan aggressively
! - -3.0 to +6.0: Scan normally
! - +6.0 to +10.0: Reduced scanning (trusted)
! - Unlisted (no score): Scan with default policy
```

### Identification Profiles

```
! Identification profiles determine which access policy applies

! Web Security Manager > Identification Profiles

! Profile components:
! 1. Subnet/IP range — match by source IP
! 2. Authentication — require user credentials
! 3. Protocol — HTTP, HTTPS, FTP, SOCKS
! 4. User Agent — match browser/application

! Example profiles:
! - "Corporate_Users" — 10.0.0.0/8, require NTLM auth
! - "Guest_WiFi" — 192.168.100.0/24, no auth
! - "Servers" — 10.1.0.0/24, no auth, minimal filtering
```

## HTTPS Inspection (SSL Decryption)

### Decryption Policy

```
! HTTPS inspection requires the WSA to act as man-in-the-middle:
! 1. WSA terminates client TLS session
! 2. WSA inspects plaintext content
! 3. WSA creates new TLS session to destination server
! 4. WSA re-signs response with its own CA certificate

! Web Security Manager > Decryption Policies

! Actions per URL category:
! - Decrypt — full inspection (MitM)
! - Pass Through — allow without inspection (privacy-sensitive)
! - Drop — block the connection
! - Monitor — pass through but log SNI/certificate info

! Common pass-through categories:
! - Financial Services (banking sites)
! - Health and Medicine (HIPAA concerns)
! - Government (legal restrictions)

! WSA root CA certificate:
! Network > Certificate Management
! Generate or upload root CA certificate
! Deploy root CA to all endpoints via GPO, MDM, or manual install
! Endpoints must trust WSA CA for decryption to work without errors
```

### Certificate Handling

```
! Certificate validation on upstream connections:
! WSA validates the destination server's certificate:
! - Certificate chain valid
! - Not expired
! - Hostname matches
! - Not revoked (OCSP/CRL)

! Invalid certificate actions:
! - Drop connection (strictest)
! - Decrypt and warn user
! - Pass through with warning

! Certificate pinning handling:
! Some applications use certificate pinning (reject non-original cert)
! WSA must pass through pinned connections:
! - Mobile banking apps
! - OS update services
! - Select SaaS applications
```

## Anti-Malware Scanning

### Engine Configuration

```
! Security Services > Anti-Malware

! Sophos Anti-Malware:
! - Real-time scanning of HTTP responses
! - Signature + heuristic detection
! - Scans: executables, archives, documents, scripts

! McAfee Anti-Malware:
! - Secondary scanning engine
! - Independent signature database
! - Can run alongside Sophos for dual-engine scanning

! AMP (Advanced Malware Protection):
! - File reputation (cloud-based SHA-256 lookup)
! - File analysis (Threat Grid sandbox)
! - Retrospective alerting (verdict changes after delivery)

! AMP file reputation flow:
! 1. User downloads file through WSA
! 2. WSA calculates SHA-256 hash
! 3. WSA queries AMP cloud for file reputation
! 4. Verdict: Clean → deliver | Malicious → block | Unknown → sandbox
! 5. If sandbox: hold or deliver, analyze in background
! 6. If verdict changes later: retrospective alert to admin

! Scanning settings per access policy:
! - Object types to scan (by MIME type)
! - Maximum object size to scan (default: 32MB)
! - Action on malware: block (default), monitor
! - Action on unscannable: block or monitor
```

## AVC (Application Visibility and Control)

```
! AVC identifies and controls applications within HTTP/HTTPS traffic

! Security Services > AVC

! AVC identifies applications by:
! - HTTP headers (User-Agent, Host, URI patterns)
! - SSL/TLS certificate attributes (CN, SAN)
! - Payload signatures (deep packet inspection)
! - Behavioral analysis (connection patterns)

! Application categories:
! - Cloud Storage (Dropbox, Google Drive, OneDrive)
! - Social Media (Facebook, Twitter, LinkedIn)
! - Streaming (YouTube, Netflix, Spotify)
! - Messaging (WhatsApp Web, Slack, Teams)
! - File Sharing (BitTorrent, peer-to-peer)

! AVC actions per access policy:
! - Allow — permit the application
! - Block — deny the application (show block page)
! - Throttle — bandwidth limit (e.g., YouTube to 1 Mbps)
! - Monitor — allow but log usage

! Granular controls:
! - Allow YouTube but block uploads
! - Allow Facebook browsing but block Facebook Chat
! - Allow Dropbox downloads but block uploads (DLP)
```

## Authentication

### Authentication Methods

| Method | Protocol | User Experience | Best For |
|--------|----------|----------------|----------|
| NTLM | HTTP negotiate | Transparent (SSO) | Windows domain-joined |
| Kerberos | HTTP negotiate | Transparent (SSO) | Modern AD environments |
| LDAP (Basic) | HTTP Basic Auth | Popup prompt | Non-domain, LDAP directories |
| SAML | SAML 2.0 redirect | Browser redirect to IdP | Cloud IdP, SSO federation |
| Certificate | Client TLS certificate | Transparent | High-security environments |
| IP-based | No authentication | Transparent | Servers, non-interactive |

### Authentication Configuration

```
! Network > Authentication

! NTLM/Kerberos (Active Directory):
! 1. Join WSA to AD domain
!    wsa> ntlmconfig
!    Domain: CORP.EXAMPLE.COM
!    Domain Controller: dc1.corp.example.com
!    Machine Account: WSA$
!
! 2. Kerberos keytab (for transparent auth)
!    Upload keytab file to WSA
!    SPN: HTTP/wsa.corp.example.com@CORP.EXAMPLE.COM
!
! 3. Authentication realm configuration
!    Define realm: CORP_AD
!    Scheme: NTLM + Kerberos (negotiate)
!    Fallback: Basic (for non-domain devices)

! SAML Authentication:
! 1. Configure IdP (Okta, Azure AD, Ping)
! 2. Upload IdP metadata to WSA
! 3. WSA generates SP metadata for IdP
! 4. User → WSA → redirect to IdP → authenticate → redirect back

! Authentication surrogates:
! After initial auth, WSA tracks the user session by:
! - IP address (simplest, NAT issues)
! - Cookie (most reliable, HTTP only)
! - Credential caching (re-auth periodically)
```

### Authentication Failure Handling

```
! What happens when authentication fails:

! Guest policy — apply limited access policy (no auth required)
! Block — deny access completely
! Re-prompt — ask for credentials again (max 3 attempts)

! Transparent mode authentication challenges:
! - No browser proxy config → NTLM negotiate more complex
! - Requires redirect to WSA hostname for cookie/NTLM
! - Some applications do not handle 407/302 redirects properly
! - Fall back to IP-based identification for non-interactive traffic
```

## SOCKS Proxy

```
! WSA supports SOCKS v4 and v5 proxy

! Network > Proxy Settings > SOCKS Proxy
! Enable SOCKS proxy: yes
! SOCKS port: 1080

! SOCKS v5 features:
! - TCP and UDP support
! - Username/password authentication
! - IPv4 and IPv6
! - DNS resolution by proxy (client sends hostname)

! SOCKS policy:
! Web Security Manager > SOCKS Policies
! - Match by source IP, authentication, destination
! - Actions: Allow, Block, Monitor
! - Apply URL filtering and reputation checking

! Use case: applications that use SOCKS (SSH tunnels, custom apps)
! SOCKS traffic logged separately from HTTP/HTTPS
```

## Cognitive Threat Analytics (CTA)

```
! CTA uses machine learning to detect compromised endpoints

! Security Services > Cognitive Threat Analytics
! (Cloud-based, WSA sends anonymized traffic metadata to Cisco cloud)

! CTA detects:
! - Command and control (C2) communication patterns
! - Data exfiltration via DNS tunneling, HTTP POST
! - Domain generation algorithms (DGA)
! - Exploit kit traffic patterns
! - Anomalous browsing behavior

! CTA data sources from WSA:
! - Web access logs (URLs, user agents, timing)
! - DNS query logs
! - Connection metadata (not content)

! CTA output:
! - Confirmed threats with confidence score
! - Affected endpoints and users
! - Threat type classification
! - Recommended remediation actions

! Integration: CTA findings feed into Cisco SecureX for investigation
```

## Cisco Umbrella Integration

```
! WSA integrates with Cisco Umbrella for DNS-layer security

! Umbrella provides:
! - DNS-layer blocking (block domains before TCP connection)
! - Cloud proxy (Umbrella SIG — Secure Internet Gateway)
! - Roaming client for off-network protection

! Integration methods:
! 1. WSA forwards unresolved categories to Umbrella for DNS check
! 2. WSA + Umbrella SIG for remote users (cloud-delivered proxy)
! 3. Policy sync between WSA on-prem and Umbrella cloud

! Umbrella SIG (cloud proxy):
! - Full proxy in the cloud (like WSA but cloud-native)
! - HTTPS inspection, AVC, DLP, anti-malware
! - IPsec or PAC-based traffic redirection
! - Protects remote users without VPN
```

## Troubleshooting

### CLI Commands

```
! System status
wsa> status
wsa> status detail

! View access logs in real time
wsa> tail accesslogs

! Access log format (Squid-compatible):
! timestamp elapsed_ms client_ip result_code/status bytes method url
! content_type auth_user policy decision_tag

! Example access log entry:
! 1712300000.000 234 10.1.1.100 TCP_MISS/200 15234 GET
!   https://www.example.com/page.html application/html
!   "CORP\jsmith" ALLOW_DEFAULT_ACTION_Scan

! Test URL categorization
wsa> advancedproxyconfig > miscellaneous > URL category lookup
! Enter URL → shows category and reputation score

! DNS diagnostics
wsa> dnsflush       ! flush DNS cache
wsa> diagnostic > network > nslookup

! Authentication test
wsa> authcache > clear       ! clear authentication cache
wsa> ntlmconfig > test       ! test AD connectivity

! Proxy diagnostics
wsa> advancedproxyconfig
! Various advanced proxy tuning options

! Packet capture
wsa> packetcapture start
wsa> packetcapture stop
```

### Common Issues

| Issue | Symptom | Resolution |
|-------|---------|------------|
| Certificate error | Browser shows untrusted certificate | Deploy WSA root CA to endpoint trust store |
| Auth loop | Repeated 407 prompts | Check Kerberos SPN, time sync, realm config |
| Slow browsing | High latency through proxy | Check scanning queue, increase proxy threads |
| WCCP flap | Traffic bypasses proxy intermittently | Verify WCCP router config, GRE tunnel health |
| Pinned cert app failure | App refuses WSA-signed certificate | Add app domain to decryption pass-through |

## Tips

- Deploy WSA root CA certificate to all endpoints before enabling HTTPS decryption; untrusted certificate warnings will flood your helpdesk otherwise.
- Start HTTPS decryption with a small number of URL categories and expand gradually; some applications break with MitM inspection.
- Always pass through financial, healthcare, and certificate-pinned application categories to avoid breaking trust and compliance.
- Use PAC files with WPAD for flexible, centralized proxy configuration; avoid hardcoding proxy addresses in browser settings.
- Enable AMP file reputation on the WSA for an additional malware detection layer beyond Sophos/McAfee signature scanning.
- WCCP with GRE requires IP protocol 47 allowed through all firewalls between the router and WSA.
- Authentication surrogates by IP address break in NAT environments; use cookie-based surrogates instead.
- AVC bandwidth throttling is more user-friendly than blocking; users can still access streaming at reduced quality.
- Monitor the WSA access logs for "BLOCK" and "MONITOR" actions to tune policies and reduce false positives.
- Integrate WSA with Cisco Umbrella for consistent policy enforcement for both on-network and off-network users.

## See Also

- tls, pki, cisco-ise, dns, waf

## References

- [Cisco Secure Web Appliance User Guide](https://www.cisco.com/c/en/us/td/docs/security/wsa/wsa-15-0/user-guide/b_WSA_UserGuide_15_0.html)
- [Cisco Secure Web Appliance CLI Reference](https://www.cisco.com/c/en/us/td/docs/security/wsa/wsa-15-0/cli-reference/b_WSA_CLI_Reference_15_0.html)
- [Cisco WCCP Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/ipapp/configuration/xe-16/iap-xe-16-book/iap-wccp.html)
- [Cisco Umbrella Documentation](https://docs.umbrella.com/)
- [RFC 1928 — SOCKS Protocol Version 5](https://www.rfc-editor.org/rfc/rfc1928)
- [RFC 7235 — HTTP/1.1 Authentication](https://www.rfc-editor.org/rfc/rfc7235)
- [RFC 4559 — HTTP Negotiate (SPNEGO/Kerberos)](https://www.rfc-editor.org/rfc/rfc4559)
