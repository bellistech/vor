# WAF (Web Application Firewall)

Inline security control that inspects, filters, and blocks malicious HTTP/HTTPS traffic targeting web applications based on rule sets and behavioral analysis.

## ModSecurity Installation

```bash
# Nginx + ModSecurity v3 (libmodsecurity)
sudo apt install libmodsecurity3 libmodsecurity-dev
git clone https://github.com/SpiderLabs/ModSecurity-nginx.git
# Rebuild Nginx with --add-dynamic-module=../ModSecurity-nginx

# Apache + ModSecurity v2
sudo apt install libapache2-mod-security2
sudo a2enmod security2

# OWASP Core Rule Set (CRS)
git clone https://github.com/coreruleset/coreruleset.git /etc/modsecurity/crs
cp /etc/modsecurity/crs/crs-setup.conf.example /etc/modsecurity/crs/crs-setup.conf

# Verify installation
nginx -t
apachectl -M | grep security
```

## ModSecurity Configuration

```bash
# /etc/modsecurity/modsecurity.conf

# Enable engine (DetectionOnly for monitoring, On for blocking)
SecRuleEngine DetectionOnly

# Request body handling
SecRequestBodyAccess On
SecRequestBodyLimit 13107200
SecRequestBodyNoFilesLimit 131072

# Response body inspection
SecResponseBodyAccess On
SecResponseBodyMimeType text/plain text/html text/xml application/json

# Audit logging
SecAuditEngine RelevantOnly
SecAuditLogRelevantStatus "^(?:5|4(?!04))"
SecAuditLog /var/log/modsecurity/audit.log
SecAuditLogType Serial
SecAuditLogFormat JSON

# Temp files
SecTmpDir /tmp/modsecurity
SecDataDir /var/log/modsecurity/data

# Unicode mapping
SecUnicodeMapFile /etc/modsecurity/unicode.mapping 20127
```

## CRS Anomaly Scoring

```bash
# crs-setup.conf - anomaly scoring thresholds

# Inbound anomaly score threshold (lower = stricter)
SecAction "id:900110,phase:1,pass,nolog,\
  setvar:tx.inbound_anomaly_score_threshold=5,\
  setvar:tx.outbound_anomaly_score_threshold=4"

# Paranoia level (1=balanced, 2=moderate, 3=strict, 4=extreme)
SecAction "id:900000,phase:1,pass,nolog,\
  setvar:tx.paranoia_level=1"

# Score assignments per severity
# CRITICAL: 5 (SQL injection, RCE)
# ERROR:    4 (XSS, path traversal)
# WARNING:  3 (scanner detection)
# NOTICE:   2 (protocol anomalies)

# Block when cumulative score exceeds threshold
SecRule TX:ANOMALY_SCORE "@ge %{tx.inbound_anomaly_score_threshold}" \
  "id:949110,phase:2,deny,status:403,\
   msg:'Inbound Anomaly Score Exceeded (score %{TX.ANOMALY_SCORE})'"
```

## SecRule Syntax

```bash
# Format: SecRule TARGET OPERATOR [ACTIONS]

# Block SQL injection in parameters
SecRule ARGS "@detectSQLi" \
  "id:100001,phase:2,deny,status:403,\
   msg:'SQL Injection detected',\
   logdata:'Matched Data: %{TX.0}',\
   tag:'OWASP_CRS/WEB_ATTACK/SQL_INJECTION',\
   severity:'CRITICAL'"

# Block XSS in request body
SecRule REQUEST_BODY "@detectXSS" \
  "id:100002,phase:2,deny,status:403,\
   msg:'XSS Attack detected',\
   tag:'OWASP_CRS/WEB_ATTACK/XSS'"

# Path traversal prevention
SecRule REQUEST_URI "\.\./" \
  "id:100003,phase:1,deny,status:403,\
   msg:'Path traversal attempt'"

# Block specific user agents (scanners)
SecRule REQUEST_HEADERS:User-Agent "@pmFromFile scanners-user-agents.data" \
  "id:100004,phase:1,deny,status:403,\
   msg:'Scanner/bot detected'"

# Restrict HTTP methods
SecRule REQUEST_METHOD "!@within GET POST HEAD OPTIONS" \
  "id:100005,phase:1,deny,status:405,\
   msg:'Method not allowed'"

# IP-based rate limiting
SecRule IP:REQUEST_COUNT "@gt 100" \
  "id:100006,phase:1,deny,status:429,\
   msg:'Rate limit exceeded'"
SecAction "id:100007,phase:5,pass,nolog,\
  initcol:ip=%{REMOTE_ADDR},\
  setvar:ip.request_count=+1,\
  expirevar:ip.request_count=60"

# Geo-blocking
SecGeoLookupDB /usr/share/GeoIP/GeoLite2-Country.mmdb
SecRule REMOTE_ADDR "@geoLookup" "chain,id:100008,phase:1,deny"
  SecRule GEO:COUNTRY_CODE "@pmFromFile blocked-countries.data"
```

## False Positive Tuning

```bash
# Rule exclusion (disable specific rule for specific URI)
SecRule REQUEST_URI "@beginsWith /api/upload" \
  "id:100100,phase:1,pass,nolog,\
   ctl:ruleRemoveById=942100"

# Exclude parameter from inspection
SecRule REQUEST_URI "@beginsWith /admin/editor" \
  "id:100101,phase:1,pass,nolog,\
   ctl:ruleRemoveTargetById=941100;ARGS:content"

# Whitelist IP addresses
SecRule REMOTE_ADDR "@ipMatch 10.0.0.0/8,172.16.0.0/12" \
  "id:100102,phase:1,pass,nolog,\
   ctl:ruleEngine=Off"

# Disable CRS rules by tag
SecRuleRemoveByTag "attack-sqli"

# Tuning workflow: review audit log for false positives
cat /var/log/modsecurity/audit.log | \
  jq 'select(.transaction.messages[].details.ruleId == "942100")'

# Common false positive patterns
# - Rich text editors triggering XSS rules
# - JSON APIs triggering SQL injection rules
# - File upload endpoints triggering body inspection
# - REST APIs with unusual HTTP methods
```

## Custom Rules for OWASP Top 10

```bash
# A01:2021 - Broken Access Control
SecRule REQUEST_URI "@rx /admin" \
  "id:100200,phase:1,chain,deny,status:403,msg:'Admin access from non-admin IP'"
  SecRule REMOTE_ADDR "!@ipMatch 10.0.1.0/24"

# A02:2021 - Cryptographic Failures (block sensitive data in responses)
SecRule RESPONSE_BODY "@rx (?:\d{4}[-\s]?){4}" \
  "id:100201,phase:4,deny,status:500,\
   msg:'Credit card number in response'"

# A03:2021 - Injection (command injection)
SecRule ARGS "@rx (?:;|\||`|\$\()" \
  "id:100202,phase:2,deny,status:403,\
   msg:'OS command injection attempt'"

# A07:2021 - Authentication bypass attempt
SecRule REQUEST_URI "@rx /api/" "chain,id:100203,phase:1,deny,status:401"
  SecRule &REQUEST_HEADERS:Authorization "@eq 0"

# A08:2021 - SSRF prevention
SecRule ARGS "@rx https?://(?:127\.|10\.|192\.168\.|172\.(?:1[6-9]|2\d|3[01])\.)" \
  "id:100204,phase:2,deny,status:403,\
   msg:'SSRF attempt to internal address'"

# A09:2021 - Security logging
SecRule RESPONSE_STATUS "@rx ^(?:4\d{2}|5\d{2})" \
  "id:100205,phase:5,pass,\
   msg:'Error response logged',\
   logdata:'%{RESPONSE_STATUS} %{REQUEST_URI}'"
```

## Cloud WAF Configuration

```bash
# AWS WAF (via AWS CLI)
# Create IP set for blocking
aws wafv2 create-ip-set --name "BlockedIPs" \
  --scope REGIONAL --ip-address-version IPV4 \
  --addresses "203.0.113.0/24" "198.51.100.0/24"

# Create rate-based rule
aws wafv2 create-web-acl --name "AppWAF" \
  --scope REGIONAL --default-action Allow={} \
  --rules '[{
    "Name": "RateLimit",
    "Priority": 1,
    "Statement": {
      "RateBasedStatement": {
        "Limit": 2000,
        "AggregateKeyType": "IP"
      }
    },
    "Action": {"Block": {}},
    "VisibilityConfig": {
      "SampledRequestsEnabled": true,
      "CloudWatchMetricsEnabled": true,
      "MetricName": "RateLimit"
    }
  }]'

# AWS Managed Rules (OWASP CRS equivalent)
# AWSManagedRulesCommonRuleSet
# AWSManagedRulesSQLiRuleSet
# AWSManagedRulesKnownBadInputsRuleSet

# Cloudflare WAF (via API)
curl -X POST "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/firewall/rules" \
  -H "Authorization: Bearer $CF_TOKEN" \
  -H "Content-Type: application/json" \
  -d '[{
    "filter": {"expression": "http.request.uri.path contains \"/wp-admin\" and ip.src ne 10.0.0.0/8"},
    "action": "block",
    "description": "Block external wp-admin access"
  }]'
```

## Bypass Testing (Security Assessment)

```bash
# Case manipulation
curl "http://target/page?q=UnIoN+SeLeCt+1,2,3"

# Double encoding
curl "http://target/page?q=%2527%2520OR%25201%253D1"

# Comment insertion (SQL)
curl "http://target/page?q=UN/**/ION+SEL/**/ECT"

# Null byte injection
curl "http://target/page?file=../../../etc/passwd%00.jpg"

# HTTP parameter pollution
curl "http://target/page?q=safe&q=UNION+SELECT"

# Chunked transfer encoding
printf 'POST /page HTTP/1.1\r\nTransfer-Encoding: chunked\r\n\r\n3\r\nUNI\r\n9\r\nON SELECT\r\n0\r\n\r\n' | nc target 80

# JSON content type bypass
curl -X POST "http://target/api" \
  -H "Content-Type: application/json" \
  -d '{"query":"UNION SELECT 1,2,3"}'

# Unicode normalization bypass
curl "http://target/page?q=%EF%BC%B5NION+%EF%BC%B3ELECT"
```

## Tips

- Start in DetectionOnly mode and analyze logs for 2-4 weeks before switching to blocking
- Use anomaly scoring mode over traditional deny mode for fewer false positives
- Set paranoia level 1 initially; increase only after thorough tuning at each level
- Exclude known-good parameters from inspection rather than disabling entire rules
- Monitor the anomaly score distribution to set thresholds; most legitimate traffic scores 0
- Log all blocked requests with full request details for incident investigation
- Test WAF bypass techniques against your own deployment before attackers do
- Implement virtual patching via WAF rules for newly disclosed CVEs before application patches
- Use different rule sets per URI path: stricter for login pages, relaxed for file upload endpoints
- Review CRS release notes before upgrading; new rules at higher paranoia levels may cause breakage
- Pair WAF with rate limiting and bot detection for defense in depth

## See Also

- Suricata for network-level IDS/IPS
- SIEM for WAF log correlation and alerting
- MITRE ATT&CK for mapping WAF detections to techniques
- CIS Benchmarks for web server hardening
- OWASP Top 10 for understanding protected vulnerabilities

## References

- [ModSecurity Reference Manual](https://github.com/owasp-modsecurity/ModSecurity/wiki/Reference-Manual-(v3.x))
- [OWASP Core Rule Set](https://coreruleset.org/docs/)
- [OWASP ModSecurity CRS Documentation](https://coreruleset.org/)
- [AWS WAF Developer Guide](https://docs.aws.amazon.com/waf/latest/developerguide/)
- [Cloudflare WAF Documentation](https://developers.cloudflare.com/waf/)
- [OWASP Top 10 (2021)](https://owasp.org/Top10/)
- [ModSecurity Handbook](https://www.feistyduck.com/books/modsecurity-handbook/)
