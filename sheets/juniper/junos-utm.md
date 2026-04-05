# JunOS UTM (Unified Threat Management)

SRX UTM provides anti-virus, web filtering, anti-spam, and content filtering integrated into the security policy framework. UTM profiles are attached to security policies — traffic matching the policy is inspected by the configured UTM engines.

## UTM Architecture

```
Security Policy                  UTM Processing
┌────────────────┐              ┌─────────────────────────────────┐
│ from trust     │   permit +   │ 1. Anti-virus scan              │
│ to untrust     │   utm-policy │ 2. Web filtering check          │
│ match: web     │ ──────────→  │ 3. Anti-spam check              │
│ then: permit   │              │ 4. Content filtering            │
│   utm: UTM-POL │              │ 5. Pass / Block                 │
└────────────────┘              └─────────────────────────────────┘

# UTM is applied AFTER security policy permits the traffic
# Only inspects permitted sessions — denied traffic is never UTM-scanned
```

## Anti-Virus

### Sophos engine (cloud-assisted)
```
# Sophos is the default AV engine — uses cloud lookup for file hashes
set security utm feature-profile anti-virus type sophos-engine

set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS fallback-options default log-and-permit
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS fallback-options content-size log-and-permit
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS fallback-options engine-not-ready log-and-permit
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS fallback-options too-many-requests log-and-permit
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS fallback-options timeout log-and-permit
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS notification-options virus-detection type protocol-only
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS notification-options virus-detection notify-mail-sender
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS scan-options content-size-limit 10000
set security utm feature-profile anti-virus sophos-engine profile AV-SOPHOS scan-options timeout 120
```

### Avira engine (local scanning)
```
set security utm feature-profile anti-virus type avira-engine

set security utm feature-profile anti-virus avira-engine profile AV-AVIRA
set security utm feature-profile anti-virus avira-engine profile AV-AVIRA fallback-options default block
set security utm feature-profile anti-virus avira-engine profile AV-AVIRA fallback-options content-size log-and-permit
set security utm feature-profile anti-virus avira-engine profile AV-AVIRA scan-options content-size-limit 20000
set security utm feature-profile anti-virus avira-engine profile AV-AVIRA scan-options timeout 180
set security utm feature-profile anti-virus avira-engine profile AV-AVIRA scan-options decompress-layer-limit 4
```

### AV pattern updates
```
# Manual pattern update
request security utm anti-virus sophos-engine pattern-update
request security utm anti-virus avira-engine pattern-update

# Automatic updates
set security utm feature-profile anti-virus sophos-engine pattern-update url https://update.juniper.net/SAV/
set security utm feature-profile anti-virus sophos-engine pattern-update interval 60
```

## Web Filtering

### Enhanced web filtering (cloud-based — Websense/Forcepoint)
```
set security utm feature-profile web-filtering type juniper-enhanced

# Create a web filtering profile
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED default block
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED custom-block-message "Access denied by corporate policy"
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED fallback-settings default log-and-permit
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED timeout 10

# Category actions
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED category Enhanced_Adult_Content action block
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED category Enhanced_Gambling action block
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED category Enhanced_Malware action block
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED category Enhanced_Social_Networking action permit
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED category Enhanced_News action log-and-permit

# Site reputation
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED site-reputation-action very-safe permit
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED site-reputation-action moderately-safe permit
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED site-reputation-action fairly-safe log-and-permit
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED site-reputation-action suspicious block
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED site-reputation-action harmful block
```

### Local web filtering (URL lists)
```
set security utm feature-profile web-filtering type juniper-local

# Custom URL pattern lists
set security utm custom-objects url-pattern BLOCKED-SITES value http://malware.example.com
set security utm custom-objects url-pattern BLOCKED-SITES value http://*.gambling.com
set security utm custom-objects url-pattern ALLOWED-SITES value http://internal.company.com

# Custom URL category using patterns
set security utm custom-objects custom-url-category BLOCK-LIST value BLOCKED-SITES
set security utm custom-objects custom-url-category ALLOW-LIST value ALLOWED-SITES

# Local web filtering profile
set security utm feature-profile web-filtering juniper-local profile WF-LOCAL
set security utm feature-profile web-filtering juniper-local profile WF-LOCAL default permit
set security utm feature-profile web-filtering juniper-local profile WF-LOCAL custom-block-message "Site blocked"
set security utm feature-profile web-filtering juniper-local profile WF-LOCAL block-list BLOCK-LIST
set security utm feature-profile web-filtering juniper-local profile WF-LOCAL allow-list ALLOW-LIST
```

### Redirect web filtering (ICAP/redirect to external server)
```
set security utm feature-profile web-filtering type websense-redirect

set security utm feature-profile web-filtering websense-redirect profile WF-REDIRECT
set security utm feature-profile web-filtering websense-redirect profile WF-REDIRECT server host 10.1.1.50
set security utm feature-profile web-filtering websense-redirect profile WF-REDIRECT server port 15868
set security utm feature-profile web-filtering websense-redirect profile WF-REDIRECT sockets 8
set security utm feature-profile web-filtering websense-redirect profile WF-REDIRECT timeout 15
set security utm feature-profile web-filtering websense-redirect profile WF-REDIRECT fallback-settings default log-and-permit
```

### Custom URL categories
```
# Custom categories override built-in category classifications
set security utm custom-objects url-pattern INTERNAL-APPS value http://app1.corp.local
set security utm custom-objects url-pattern INTERNAL-APPS value http://app2.corp.local

set security utm custom-objects custom-url-category CORPORATE-APPS value INTERNAL-APPS

# Use in enhanced web filtering profile
set security utm feature-profile web-filtering juniper-enhanced profile WF-ENHANCED category CORPORATE-APPS action permit
```

## Anti-Spam

### Server-based anti-spam (SBL — Sophos Blocklist)
```
set security utm feature-profile anti-spam sbl profile AS-SBL
set security utm feature-profile anti-spam sbl profile AS-SBL sbl-default-server
set security utm feature-profile anti-spam sbl profile AS-SBL spam-action block
set security utm feature-profile anti-spam sbl profile AS-SBL no-sbl-default-server custom-tag-string "[SPAM]"

# Custom whitelist/blacklist
set security utm feature-profile anti-spam address-whitelist WL1 10.1.1.0/24
set security utm feature-profile anti-spam address-blacklist BL1 192.168.99.0/24
```

### Local anti-spam lists
```
set security utm custom-objects url-pattern SPAM-SENDERS value mail.spam-domain.com
set security utm custom-objects url-pattern LEGIT-SENDERS value mail.partner.com
```

## Content Filtering

### MIME type filtering
```
set security utm feature-profile content-filtering profile CF-PROFILE
set security utm feature-profile content-filtering profile CF-PROFILE block-mime mime-pattern application/x-javascript
set security utm feature-profile content-filtering profile CF-PROFILE block-mime mime-pattern application/x-shockwave-flash

# Custom MIME list
set security utm custom-objects mime-pattern BLOCKED-MIME value application/x-msdownload
set security utm custom-objects mime-pattern BLOCKED-MIME value application/x-executable
set security utm feature-profile content-filtering profile CF-PROFILE block-mime mime-list BLOCKED-MIME
```

### File extension filtering
```
set security utm custom-objects filename-extension EXE-FILES value exe
set security utm custom-objects filename-extension EXE-FILES value bat
set security utm custom-objects filename-extension EXE-FILES value cmd
set security utm custom-objects filename-extension EXE-FILES value scr
set security utm custom-objects filename-extension EXE-FILES value pif

set security utm feature-profile content-filtering profile CF-PROFILE block-extension-list EXE-FILES
```

### Protocol command filtering
```
# Block specific FTP commands
set security utm feature-profile content-filtering profile CF-PROFILE permit-command ftp-command "STOR"
set security utm feature-profile content-filtering profile CF-PROFILE permit-command ftp-command "RETR"
# Only listed commands are permitted; all others blocked

# Block specific HTTP methods
set security utm feature-profile content-filtering profile CF-PROFILE block-content-type activex
set security utm feature-profile content-filtering profile CF-PROFILE block-content-type java-applet
set security utm feature-profile content-filtering profile CF-PROFILE block-content-type http-cookie

# Content size limit
set security utm feature-profile content-filtering profile CF-PROFILE notification-options type protocol-only
```

## UTM Policies

### Create UTM policy combining all profiles
```
set security utm utm-policy UTM-FULL anti-virus http-profile AV-SOPHOS
set security utm utm-policy UTM-FULL anti-virus ftp upload-profile AV-SOPHOS
set security utm utm-policy UTM-FULL anti-virus ftp download-profile AV-SOPHOS
set security utm utm-policy UTM-FULL anti-virus smtp-profile AV-SOPHOS
set security utm utm-policy UTM-FULL anti-virus pop3-profile AV-SOPHOS
set security utm utm-policy UTM-FULL anti-virus imap-profile AV-SOPHOS

set security utm utm-policy UTM-FULL web-filtering http-profile WF-ENHANCED
set security utm utm-policy UTM-FULL anti-spam smtp-profile AS-SBL
set security utm utm-policy UTM-FULL content-filtering http-profile CF-PROFILE
set security utm utm-policy UTM-FULL content-filtering ftp upload-profile CF-PROFILE
set security utm utm-policy UTM-FULL content-filtering ftp download-profile CF-PROFILE

# Traffic options
set security utm utm-policy UTM-FULL traffic-options sessions-per-client-limit 50
set security utm utm-policy UTM-FULL traffic-options sessions-per-client-over-limit log-and-permit
```

### Attach UTM policy to security policy
```
set security policies from-zone trust to-zone untrust policy ALLOW-WEB then permit application-services utm-policy UTM-FULL

# UTM can also be applied to global policies
set security policies global policy GLOBAL-INSPECT then permit application-services utm-policy UTM-FULL
```

### Minimal UTM policy (web filtering only)
```
set security utm utm-policy UTM-WF-ONLY web-filtering http-profile WF-ENHANCED
set security policies from-zone trust to-zone untrust policy WEB then permit application-services utm-policy UTM-WF-ONLY
```

## SSL Proxy for UTM

```
# SSL proxy decrypts HTTPS so UTM can inspect encrypted traffic
set services ssl proxy profile SSL-INSPECT root-ca SSL-PROXY-CA
set services ssl proxy profile SSL-INSPECT trusted-ca all

# Whitelist domains to exclude from decryption
set services ssl proxy profile SSL-INSPECT whitelist banking.example.com
set services ssl proxy profile SSL-INSPECT whitelist healthcare.example.com

# Attach SSL proxy to security policy
set security policies from-zone trust to-zone untrust policy INSPECT-HTTPS then permit application-services ssl-proxy profile-name SSL-INSPECT
set security policies from-zone trust to-zone untrust policy INSPECT-HTTPS then permit application-services utm-policy UTM-FULL

# Generate root CA for SSL proxy
request security pki generate-key-pair certificate-id SSL-PROXY-CA size 2048 type rsa
request security pki local-certificate generate-self-signed certificate-id SSL-PROXY-CA domain-name proxy.example.com subject "CN=SRX SSL Proxy CA,O=Example" add-ca-constraint
# Deploy the CA certificate to all client trust stores
```

## Verification Commands

```
# UTM status
show security utm status
show security utm session

# Anti-virus
show security utm anti-virus status
show security utm anti-virus statistics
show security utm anti-virus sophos-engine pattern-update-status

# Web filtering
show security utm web-filtering statistics
show security utm web-filtering status
show security utm web-filtering category

# Anti-spam
show security utm anti-spam status
show security utm anti-spam statistics

# Content filtering
show security utm content-filtering statistics

# UTM policy hits
show security utm utm-policy UTM-FULL

# SSL proxy
show services ssl proxy statistics
show services ssl proxy certificate-cache

# General session with UTM
show security flow session extensive
show security flow session application-firewall

# Clear counters
clear security utm anti-virus statistics
clear security utm web-filtering statistics
clear security utm anti-spam statistics
clear security utm content-filtering statistics
```

## Tips

- UTM is applied only to permitted traffic — if the security policy denies it, UTM never sees it
- SSL proxy is mandatory for inspecting HTTPS — without it, web filtering can only see the SNI/hostname, not the URL path
- Always whitelist sensitive domains (banking, healthcare) from SSL decryption — compliance and certificate pinning
- Sophos AV (cloud) is faster but requires internet connectivity; Avira (local) works offline but needs pattern updates
- Enhanced web filtering (cloud) provides the most categories but adds latency per URL lookup — set reasonable timeouts
- Content size limits prevent DoS from huge files — set appropriate limits for your user base
- Anti-spam only works on SMTP (not web-based email) — combine with web filtering for comprehensive coverage
- Per-protocol AV profiles allow different scanning behavior for HTTP vs FTP vs SMTP
- Session-per-client limits in UTM policy prevent a single host from consuming all UTM resources
- Fallback actions determine what happens when UTM engine is overloaded — "block" is safer, "log-and-permit" avoids user complaints

## See Also

- junos-srx, junos-ids-ips, junos-nat, junos-ipsec-vpn, junos-firewall-filters

## References

- [Juniper TechLibrary — UTM Overview](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/utm-overview.html)
- [Juniper TechLibrary — Anti-Virus](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/utm-antivirus-overview.html)
- [Juniper TechLibrary — Web Filtering](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/utm-web-filtering-overview.html)
- [Juniper TechLibrary — Anti-Spam](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/utm-antispam-overview.html)
- [Juniper TechLibrary — Content Filtering](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/utm-content-filtering-overview.html)
- [Juniper TechLibrary — SSL Proxy](https://www.juniper.net/documentation/us/en/software/junos/utm/topics/concept/ssl-proxy-overview.html)
