# RPKI (Resource Public Key Infrastructure)

Resource Public Key Infrastructure cryptographically verifies that BGP route announcements are authorized by the legitimate holder of the IP address prefix. Defined across RFC 6480-6811 and RFC 8205-8210, RPKI creates a certificate hierarchy rooted at Regional Internet Registries (RIRs), issues Route Origin Authorizations (ROAs), and enables BGP routers to validate the origin AS of every prefix. RPKI is the primary defense against BGP hijacking, route leaks, and accidental misorigination.

---

## Core Concepts

### Certificate Hierarchy

```bash
# RPKI uses a strict hierarchy anchored at the five RIRs:
#
# IANA (root, Trust Anchor)
#  ├── ARIN       (North America)
#  ├── RIPE NCC   (Europe, Middle East, Central Asia)
#  ├── APNIC      (Asia-Pacific)
#  ├── LACNIC     (Latin America, Caribbean)
#  └── AFRINIC    (Africa)
#       ├── LIR/ISP (e.g., your upstream provider)
#       │    └── End Entity (your organization)
#       └── LIR/ISP
#            └── End Entity

# Each entity holds an X.509 resource certificate listing:
# - IP address prefixes they are authorized to hold
# - AS numbers they are authorized to hold
# - Public key for signing child certificates and ROAs

# Trust Anchor Locator (TAL) — starting point for validation
# Download TALs from each RIR (required for validators)
# ARIN TAL requires acceptance of agreement at:
# https://www.arin.net/resources/manage/rpki/tal/
```

### Route Origin Authorization (ROA)

```bash
# A ROA is a signed object that says:
# "AS 64500 is authorized to originate 203.0.113.0/24 (max length /24)"
#
# ROA fields:
# - ASN: the authorized origin AS
# - Prefix: the IP prefix
# - Max Length: maximum prefix length that can be announced
#   (controls whether more-specifics are valid)

# Example ROAs:
# ASN 64500 | 203.0.113.0/24 | maxLength 24
#   → only /24 is valid from AS 64500
#
# ASN 64500 | 203.0.113.0/24 | maxLength 28
#   → /24, /25, /26, /27, /28 all valid from AS 64500
#
# ASN 0 | 203.0.113.0/24 | maxLength 24
#   → "this prefix should never appear in BGP" (blackhole ROA)
```

## ROA Creation and Management

### Creating ROAs via RIR Portals

```bash
# Each RIR provides a web portal for ROA management:
#
# ARIN:    https://account.arin.net → RPKI → ROAs
# RIPE:    https://my.ripe.net → Resources → RPKI
# APNIC:   https://myapnic.net → Resources → RPKI
# LACNIC:  https://lacnic.net/rpki
# AFRINIC: https://my.afrinic.net → RPKI

# When creating a ROA, specify:
# 1. Origin AS number
# 2. Prefix and prefix length
# 3. Maximum prefix length
#
# Best practices:
# - Set maxLength = exact prefix length (most restrictive)
# - Create ROAs for ALL your announced prefixes
# - Create ROAs BEFORE announcing new prefixes
# - Create "AS 0" ROAs for prefixes you hold but do not announce
```

### ROA via RIPE API

```bash
# RIPE provides a REST API for ROA management
# List existing ROAs
curl -s https://my.ripe.net/api/rpki/roas \
  -H "Authorization: Bearer $TOKEN" | jq .

# Create a ROA
curl -X POST https://my.ripe.net/api/rpki/roas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "asn": "AS64500",
    "prefix": "203.0.113.0/24",
    "maxLength": 24
  }'

# Check ROA status in the RIPE RPKI dashboard
# https://rpki-dashboard.ripe.net/
```

## BGP Origin Validation

### Validation States

```bash
# When a BGP route is received, the router checks it against RPKI data:
#
# VALID    — A ROA exists, and the origin AS + prefix match
#            → route is authorized, prefer it
#
# INVALID  — A ROA exists for this prefix, but:
#            - Origin AS does not match, OR
#            - Prefix length exceeds maxLength
#            → route may be a hijack, drop or deprioritize
#
# NOT FOUND (Unknown) — No ROA exists for this prefix
#            → no RPKI data available, treat normally
#            → ~60% of routes are still in this state (2024)

# Validation decision matrix:
# ROA exists? | Origin AS matches? | Length <= maxLength? | Result
# -----------+--------------------+---------------------+--------
# No         | N/A                | N/A                 | NOT FOUND
# Yes        | Yes                | Yes                 | VALID
# Yes        | Yes                | No                  | INVALID
# Yes        | No                 | N/A                 | INVALID
```

### Route Origin Validation Policy

```bash
# Three common ROV policies:

# 1. Permissive (monitoring only)
# Accept all routes, tag RPKI state in communities
# Log INVALID routes but do not drop
# Good for initial deployment

# 2. Moderate (recommended starting point)
# VALID: prefer (boost local-pref)
# INVALID: drop
# NOT FOUND: accept normally

# 3. Strict
# VALID: accept
# INVALID: drop
# NOT FOUND: lower preference (or drop)
# Risk: drops legitimate routes without ROAs
```

## RPKI Validators

### Routinator (NLnet Labs)

```bash
# Routinator is a widely deployed RPKI relying party software
# Written in Rust, production-grade, used by major ISPs

# Install Routinator
# Debian/Ubuntu
sudo apt install routinator

# Or via cargo
cargo install routinator

# Initialize (downloads TALs, creates config)
routinator init
# Accept ARIN TAL when prompted

# TAL directory
ls ~/.rpki-cache/tals/
# afrinic.tal  apnic.tal  arin.tal  lacnic.tal  ripe.tal

# Start in server mode (RTR + HTTP)
routinator server \
  --rtr 127.0.0.1:3323 \
  --http 127.0.0.1:8323

# Start as systemd service
sudo systemctl start routinator
sudo systemctl enable routinator

# Configuration: /etc/routinator/routinator.conf
# [server]
# rtr-listen = ["127.0.0.1:3323"]
# http-listen = ["127.0.0.1:8323"]
# refresh = 3600
# retry = 600
# expire = 7200
```

### Routinator Queries

```bash
# Query Routinator HTTP API for VRPs (Validated ROA Payloads)
# All VRPs in JSON
curl -s http://127.0.0.1:8323/api/v1/validity | jq .

# Check validity of a specific route
curl -s "http://127.0.0.1:8323/api/v1/validity/AS64500/203.0.113.0/24" | jq .

# Export all VRPs in various formats
curl -s http://127.0.0.1:8323/json          # JSON
curl -s http://127.0.0.1:8323/csv           # CSV
curl -s http://127.0.0.1:8323/rpsl          # RPSL
curl -s http://127.0.0.1:8323/bird          # BIRD filter format
curl -s http://127.0.0.1:8323/bird2         # BIRD 2.x format
curl -s http://127.0.0.1:8323/openbgpd      # OpenBGPd format

# Check Routinator health
curl -s http://127.0.0.1:8323/api/v1/status | jq .

# View metrics (Prometheus format)
curl -s http://127.0.0.1:8323/metrics
```

### Fort Validator

```bash
# Fort is an alternative RPKI validator (NIC Mexico)
# Install
sudo apt install fort-validator

# Configuration: /etc/fort/fort.conf
# {
#   "tal": "/etc/fort/tals",
#   "local-repository": "/var/lib/fort/repository",
#   "server": {
#     "address": "127.0.0.1",
#     "port": 8324
#   }
# }

# Start Fort
sudo systemctl start fort-validator

# Fort serves RTR protocol directly to routers
```

### OctoRPKI (Cloudflare)

```bash
# OctoRPKI — Cloudflare's RPKI validator (Go-based)
# Produces JSON output consumed by GoRTR

# Install
go install github.com/cloudflare/cfrpki/cmd/octorpki@latest

# Run OctoRPKI to fetch and validate RPKI data
octorpki -tal.root /etc/rpki/tals/ -output.sign.key /etc/rpki/key.pem

# Output: a signed JSON file of VRPs

# GoRTR serves the VRP file via RTR protocol to routers
go install github.com/cloudflare/gortr@latest

gortr \
  -cache "https://rpki.cloudflare.com/rpki.json" \
  -verify \
  -bind ":3323"
```

### rpki-client (OpenBSD)

```bash
# rpki-client — OpenBSD's RPKI validator (also available on Linux)
# Minimal, audited codebase, used by major route servers

# Install on Debian/Ubuntu
sudo apt install rpki-client

# Run validation (cron-friendly, exits after one cycle)
rpki-client -v

# Output files in /var/db/rpki-client/
ls /var/db/rpki-client/
# bird    — BIRD filter rules
# bird2   — BIRD 2 filter rules
# csv     — CSV format
# json    — JSON format
# openbgpd — OpenBGPd filter rules

# Cron: run every hour
# 0 * * * * rpki-client -v 2>&1 | logger -t rpki-client

# Check specific prefix
jq '.roas[] | select(.prefix == "203.0.113.0/24")' /var/db/rpki-client/json
```

## RPKI-to-Router Protocol (RTR)

### RTR Protocol (RFC 8210)

```bash
# RTR delivers VRP data from validator to router
# Runs over TCP (port 323 or custom) or TLS
# Supports incremental updates (Serial Notify → Serial Query → Cache Response)

# RTR message flow:
# Router → Validator: Reset Query (on startup)
# Validator → Router: Cache Response (all VRPs)
# Validator → Router: End of Data (with session ID + serial)
# ... time passes ...
# Validator → Router: Serial Notify (new data available)
# Router → Validator: Serial Query (send me updates since serial N)
# Validator → Router: Cache Response (only changes)

# RTR timers:
# Refresh interval: how often router polls (default: 3600s)
# Retry interval: retry after failed query (default: 600s)
# Expire interval: drop VRP data if no successful query (default: 7200s)
```

### Router Configuration (BIRD 2)

```bash
# BIRD 2 — open-source BGP daemon with RPKI support

# /etc/bird/bird.conf — RPKI configuration
# rpki1 table;
#
# protocol rpki rpki1 {
#     roa4 { table roa_v4; };
#     roa6 { table roa_v6; };
#
#     remote 127.0.0.1 port 3323;
#
#     retry keep 90;
#     refresh keep 3600;
#     expire keep 7200;
# }
#
# # Apply ROV in BGP filter
# filter bgp_in {
#     if (roa_check(roa_v4, net, bgp_path.last) = ROA_INVALID) then {
#         reject;
#     }
#     if (roa_check(roa_v4, net, bgp_path.last) = ROA_VALID) then {
#         bgp_local_pref = bgp_local_pref + 20;
#     }
#     accept;
# }

# Check RPKI table in BIRD
birdc show route table roa_v4
birdc show route table roa_v6
birdc show protocols all rpki1
```

### Router Configuration (FRRouting)

```bash
# FRRouting — open-source routing suite with RPKI support

# vtysh configuration
# router bgp 64500
#   rpki
#     rpki cache 127.0.0.1 3323 preference 1
#     rpki polling_period 300
#   exit
#
#   address-family ipv4 unicast
#     neighbor 198.51.100.1 route-map rpki-validation in
#   exit
# exit
#
# route-map rpki-validation permit 10
#   match rpki valid
#   set local-preference 200
# route-map rpki-validation permit 20
#   match rpki notfound
#   set local-preference 100
# route-map rpki-validation deny 30
#   match rpki invalid

# Check RPKI status in FRR
vtysh -c "show rpki prefix-table"
vtysh -c "show rpki cache-server"
vtysh -c "show rpki cache-connection"
```

### Router Configuration (Cisco IOS-XR)

```bash
# Cisco IOS-XR RPKI configuration
# router bgp 64500
#   rpki server 127.0.0.1
#     transport tcp port 3323
#     refresh-time 300
#   !
#   address-family ipv4 unicast
#     neighbor 198.51.100.1
#       route-policy rpki-validate in
#   !
# !
# route-policy rpki-validate
#   if validation-state is valid then
#     set local-preference 200
#     done
#   endif
#   if validation-state is invalid then
#     drop
#     done
#   endif
#   set local-preference 100
#   done
# end-policy

# Check RPKI status
# show bgp rpki table
# show bgp rpki server summary
```

## Monitoring and Verification

### Checking Your Own Prefixes

```bash
# Online tools to verify RPKI status of any prefix

# RIPE RPKI Validator (web)
# https://rpki-validator.ripe.net/

# Cloudflare RPKI Portal
# https://rpki.cloudflare.com/

# NIST RPKI Monitor
# https://rpki-monitor.antd.nist.gov/

# CLI: query Routinator for your prefix
curl -s "http://127.0.0.1:8323/api/v1/validity/AS64500/203.0.113.0/24" | jq .

# CLI: search rpki-client output
jq '.roas[] | select(.prefix | startswith("203.0.113"))' /var/db/rpki-client/json

# BGP looking glass with RPKI status
# https://lg.ring.nlnog.net/
# https://bgp.he.net/
```

### Monitoring RPKI Health

```bash
# Check validator synchronization status
curl -s http://127.0.0.1:8323/api/v1/status | jq '.vrps'

# Monitor VRP count over time (should be ~400,000+ as of 2024)
curl -s http://127.0.0.1:8323/api/v1/status | jq '.vrps_count'

# Alert on validator staleness (Prometheus + Alertmanager)
# routinator_last_update_duration_seconds
# routinator_vrps_count (should not drop suddenly)

# Monitor RTR session to router
curl -s http://127.0.0.1:8323/metrics | grep rtr_client

# Check for ROA expiry (ROAs have validity periods)
# rpki-client logs warnings for expiring ROAs
rpki-client -v 2>&1 | grep -i "expire\|stale"
```

## RPKI Impact and Deployment

### What RPKI Prevents

```bash
# 1. BGP Hijacking (prefix theft)
#    Attacker announces your prefix from their AS
#    RPKI: INVALID (origin AS mismatch) → route dropped by ROV-enabled peers

# 2. Sub-prefix Hijacking
#    Attacker announces more-specific of your prefix (e.g., /25 of your /24)
#    RPKI with maxLength=24: INVALID (prefix length > maxLength) → dropped

# 3. Accidental Misorigination
#    Fat-finger route leak or misconfigured BGP
#    RPKI: INVALID → caught automatically before propagating

# What RPKI does NOT prevent:
# - Path hijacking (attacker in the AS path but not the origin)
# - Legitimate origin with unauthorized path (BGPsec addresses this, RFC 8205)
# - Routes for prefixes with no ROA (NOT FOUND = accepted)
```

### Deployment Statistics

```bash
# Check global RPKI deployment status
# https://stats.labs.apnic.net/rpki
# https://www.manrs.org/netops/participants/

# As of 2024:
# ~52% of global routes have RPKI ROAs
# ~38% of routes are RPKI VALID
# ~40% of networks perform ROV (Route Origin Validation)
# All Tier 1 transit providers now perform ROV

# Key RPKI milestones:
# 2012: First ROAs created
# 2019: NIST/DHS mandate for US government networks
# 2022: Cloudflare, Google, Amazon drop INVALID routes
# 2023: AT&T, NTT, Lumen drop INVALID routes
# 2024: >50% route coverage, approaching critical mass
```

---

## Tips

- Create ROAs for all your announced prefixes before deploying ROV. If you filter INVALID routes but your own prefixes lack ROAs, you are safe (NOT FOUND is accepted), but you miss protection against hijacks.
- Set maxLength equal to your exact announced prefix length. A /24 with maxLength /28 allows an attacker to create a "valid" /25 from a different AS that also has a ROA. Keep maxLength tight.
- Create AS 0 ROAs for address space you hold but do not announce. This makes any unauthorized announcement of those prefixes INVALID instead of NOT FOUND.
- Run your RPKI validator locally (Routinator, rpki-client, Fort). Do not depend on a remote validator for production RTR feeds. Validator failure should not affect routing.
- Monitor VRP count from your validator. A sudden drop (e.g., from 400K to 0) indicates a synchronization failure and should trigger an alert before the expire timer removes all VRPs from the router.
- RPKI does not protect the AS path, only the origin. BGPsec (RFC 8205) addresses path security but has essentially zero deployment due to performance overhead.
- Start ROV deployment in monitoring mode (log INVALID, do not drop). Graduate to dropping INVALID after verifying no legitimate traffic is affected.
- The RPKI expire timer (default 7200s) determines how long a router trusts VRP data after losing contact with the validator. Set this high enough to survive validator restarts.
- rpki-client produces static output files (JSON, BIRD, OpenBGPd) refreshed by cron, making it easy to integrate with any routing daemon without RTR protocol support.
- Route leaks (AS prepends the full path but should not transit) are partially mitigated by RPKI (leak may have wrong origin) but fully addressed by ASPA (RFC 9473, Autonomous System Provider Authorization).

---

## See Also

- bgp, tls, pki, certificate

## References

- [RFC 6480 — An Infrastructure to Support Secure Internet Routing (RPKI)](https://www.rfc-editor.org/rfc/rfc6480)
- [RFC 6811 — BGP Prefix Origin Validation](https://www.rfc-editor.org/rfc/rfc6811)
- [RFC 8210 — The Resource Public Key Infrastructure (RPKI) to Router Protocol, Version 1](https://www.rfc-editor.org/rfc/rfc8210)
- [RFC 8205 — BGPsec Protocol Specification](https://www.rfc-editor.org/rfc/rfc8205)
- [RFC 9319 — The Use of maxLength in RPKI](https://www.rfc-editor.org/rfc/rfc9319)
- [RFC 9473 — Autonomous System Provider Authorization (ASPA)](https://www.rfc-editor.org/rfc/rfc9473)
- [Routinator — NLnet Labs RPKI Validator](https://routinator.docs.nlnetlabs.nl/)
- [rpki-client — OpenBSD RPKI Validator](https://www.rpki-client.org/)
- [Cloudflare RPKI Portal](https://rpki.cloudflare.com/)
- [NIST RPKI Monitor](https://rpki-monitor.antd.nist.gov/)
