# OSINT (Open-Source Intelligence Gathering)

> For authorized security testing, CTF competitions, and educational purposes only.

Open-source intelligence (OSINT) is the collection and analysis of publicly available
information to support security assessments, threat intelligence, and investigations.
This sheet covers search engine dorking, domain reconnaissance, internet-wide scanners,
social media intelligence, image analysis, breach data, and automation workflows.

---

## Search Engine Dorking

### Google Dorking

```bash
# File type and sensitive data discovery
site:target.com filetype:pdf             # PDF documents
site:target.com filetype:xlsx            # Excel spreadsheets
site:target.com filetype:env             # environment files
site:target.com filetype:sql             # SQL dumps
site:target.com filetype:log             # log files
site:target.com filetype:bak             # backup files
site:target.com filetype:conf            # config files

# Directory and admin panel discovery
site:target.com intitle:"index of"       # open directory listings
site:target.com inurl:admin              # admin panels
site:target.com inurl:login              # login pages
site:target.com inurl:".git"             # exposed git repos
site:target.com inurl:phpmyadmin         # phpMyAdmin instances

# Sensitive information
site:target.com intext:"api_key"         # exposed API keys
site:target.com intext:"password"        # pages containing passwords
site:target.com intext:"sql syntax"      # SQL errors
site:target.com intitle:"phpinfo()"      # PHP info pages
site:target.com ext:xml inurl:sitemap    # sitemaps

# Combined operators
site:target.com (filetype:env | filetype:yml) intext:password
site:target.com -www inurl:dev           # non-www dev subdomains
```

### GitHub and GitLab Dorking

```bash
# Search for leaked secrets on GitHub
"target.com" password
"target.com" api_key
"target.com" secret_key
"target.com" AWS_ACCESS_KEY
org:targetorg filename:.env
org:targetorg filename:credentials
org:targetorg filename:config.yml password

# truffleHog — scan repos for high-entropy strings and secrets
pip install trufflehog
trufflehog git https://github.com/target/repo.git

# GitDorker — GitHub dork automation
git clone https://github.com/obheda12/GitDorker.git
python3 GitDorker.py -t <GITHUB_TOKEN> -d dorks.txt -q target.com

# gitleaks — detect secrets in repos
gitleaks detect --source /path/to/repo --report-path results.json
```

---

## Domain and Infrastructure Intelligence

### DNS and Subdomain Enumeration

```bash
# Certificate Transparency logs (most reliable passive source)
curl -s "https://crt.sh/?q=%25.target.com&output=json" | \
  jq -r '.[].name_value' | sort -u

# subfinder — passive subdomain enumeration
subfinder -d target.com -all -o subdomains.txt

# amass — comprehensive enumeration
amass enum -passive -d target.com -o amass_subs.txt
amass enum -active -d target.com -brute -o amass_active.txt

# DNS record enumeration
dig target.com ANY +noall +answer
dig target.com MX +short
dig target.com TXT +short
dig target.com NS +short
dig -x <IP_ADDRESS> +short              # reverse DNS

# WHOIS and ASN lookup
whois target.com
whois <IP_ADDRESS>
whois -h whois.radb.net -- '-i origin AS12345'
curl -s "https://api.bgpview.io/asn/12345/prefixes" | jq '.data.ipv4_prefixes[].prefix'
```

### theHarvester

```bash
# Aggregate OSINT from multiple sources
pip install theHarvester

# Full harvest with all sources
theHarvester -d target.com -b all -l 500

# Specific sources
theHarvester -d target.com -b google,bing,linkedin,crtsh,dnsdumpster

# Output to XML
theHarvester -d target.com -b all -f results.xml
```

---

## Internet-Wide Scanning Services

### Shodan

```bash
# Search engine for internet-connected devices
pip install shodan
shodan init <API_KEY>

# Basic host and organization searches
shodan search "hostname:target.com"
shodan search "org:Target Inc"
shodan search "net:192.168.0.0/16"
shodan search "ssl.cert.subject.cn:target.com"
shodan host <IP_ADDRESS>

# Find specific services and vulnerabilities
shodan search "hostname:target.com port:22"
shodan search "hostname:target.com vuln:CVE-2021-44228"

# Web interface queries (shodan.io):
# port:9200 "elastic" org:"Target Inc"     # Elasticsearch
# port:27017 "MongoDB" org:"Target Inc"    # MongoDB
# "X-Jenkins" "200 OK" org:"Target Inc"    # Jenkins
```

### Censys and ZoomEye

```bash
# Censys — internet-wide scan data
pip install censys && censys config
censys search "services.tls.certificates.leaf.subject.common_name: target.com"
censys view <IP_ADDRESS>

# ZoomEye (zoomeye.org) — queries:
# hostname:target.com
# app:"Apache" hostname:target.com
# cidr:192.168.0.0/24
```

---

## Recon Frameworks

### Recon-ng

```bash
# Modular OSINT framework
pip install recon-ng && recon-ng

# Inside recon-ng:
# workspaces create target_recon
# marketplace install all

# modules load recon/domains-hosts/certificate_transparency
# options set SOURCE target.com
# run

# modules load recon/domains-contacts/whois_pocs
# options set SOURCE target.com
# run

# Export: modules load reporting/html
# show hosts
# show contacts
```

### SpiderFoot

```bash
# Automated OSINT collection with web UI
pip install spiderfoot
spiderfoot -l 127.0.0.1:5001             # start web interface

# CLI scan
spiderfoot -s target.com -t INTERNET_NAME,IP_ADDRESS,EMAILADDR -q
spiderfoot -s target.com -o csv -q > results.csv
```

---

## Social Media OSINT

### Username and Profile Enumeration

```bash
# Sherlock — find usernames across social networks
pip install sherlock-project
sherlock targetusername --print-found
sherlock targetusername --output results.txt --csv
sherlock targetusername --site twitter --site github --site linkedin

# Maigret — advanced username search (Sherlock fork)
pip install maigret
maigret targetusername --all-sites --reports-dir ./reports

# social-analyzer — API-based analysis
pip install social-analyzer
social-analyzer --username "targetusername" --metadata --filter "good"
```

### Social Media Search Techniques

```bash
# Twitter/X advanced search
# from:targetuser since:2024-01-01 until:2024-12-31
# "@targetuser" -from:targetuser (replies to user)
# from:targetuser url:target.com (tweets with specific URLs)

# LinkedIn OSINT via Google
# site:linkedin.com/in/ "Target Company" "engineer"
# site:linkedin.com/pub/dir/ "firstname" "lastname"
```

---

## Image and Geolocation OSINT

### EXIF and Metadata Extraction

```bash
# exiftool — comprehensive metadata extraction
exiftool photo.jpg                       # all metadata
exiftool -GPSLatitude -GPSLongitude photo.jpg   # GPS coordinates
exiftool -Make -Model -Software photo.jpg       # camera/device info
exiftool -AllDates photo.jpg             # timestamps
exiftool -r -gps* /path/to/images/       # recursive GPS extraction
exiftool -all= photo.jpg                 # strip metadata (sanitize)

# Reverse image search engines:
# Google Images — general reverse search
# TinEye — tracks image origin and modifications
# Yandex Images — best for facial matching
# Bing Visual Search — alternative engine
```

### Document Metadata

```bash
# metagoofil — extract metadata from public documents
metagoofil -d target.com -t pdf,doc,xls,ppt -l 100 -o /tmp/meta -f results.html

# Bulk document metadata extraction
wget -r -l 1 -A "*.pdf,*.docx,*.xlsx" https://target.com/documents/
find . -name "*.pdf" -exec exiftool -Author -Creator {} \; | sort -u
# Reveals internal usernames, software versions, directory paths
```

---

## Breach Data and Credential Intelligence

```bash
# Have I Been Pwned — check email in known breaches
curl -s -H "hibp-api-key: $HIBP_KEY" \
  "https://haveibeenpwned.com/api/v3/breachedaccount/user@target.com" | jq .

# Password check via k-anonymity (no API key needed, no full hash sent)
echo -n "password" | sha1sum | cut -c1-5
curl -s "https://api.pwnedpasswords.com/range/5BAA6"
# Check if your hash suffix appears in results

# Dehashed — breach data search (requires subscription)
curl -s "https://api.dehashed.com/search?query=domain:target.com" \
  -u "$DEHASHED_EMAIL:$DEHASHED_KEY" | jq .
```

---

## Certificate Transparency

```bash
# crt.sh — Certificate Transparency log search
curl -s "https://crt.sh/?q=%25.target.com&output=json" | \
  jq -r '.[].name_value' | sort -u

# Recently issued certificates only
curl -s "https://crt.sh/?q=target.com&output=json" | \
  jq -r '.[] | select(.not_before > "2024-01-01") | .name_value' | sort -u

# certspotter API
curl -s "https://api.certspotter.com/v1/issuances?domain=target.com&include_subdomains=true&expand=dns_names" | \
  jq -r '.[].dns_names[]' | sort -u
```

---

## Automation Pipeline

```bash
# Complete passive reconnaissance pipeline

# Step 1: Subdomain enumeration from multiple sources
subfinder -d target.com -silent -o subs_subfinder.txt
amass enum -passive -d target.com -o subs_amass.txt 2>/dev/null
curl -s "https://crt.sh/?q=%25.target.com&output=json" | \
  jq -r '.[].name_value' | sort -u > subs_crt.txt
cat subs_*.txt | sort -u > all_subdomains.txt
echo "Found $(wc -l < all_subdomains.txt) unique subdomains"

# Step 2: Resolve and identify live hosts
cat all_subdomains.txt | httpx -silent -status-code -title -o live_hosts.txt

# Step 3: Screenshot live hosts for visual review
cat all_subdomains.txt | httpx -silent | gowitness file -f - -P screenshots/

# Step 4: Port scan live IPs
cat all_subdomains.txt | dnsx -silent -a -resp-only | sort -u > ips.txt
nmap -iL ips.txt -Pn -sV --top-ports 1000 -oA nmap_results

# Step 5: Generate summary report
echo "# OSINT Report for target.com" > report.md
echo "## Subdomains: $(wc -l < all_subdomains.txt)" >> report.md
echo "## Live Hosts: $(wc -l < live_hosts.txt)" >> report.md
echo "## Unique IPs: $(wc -l < ips.txt)" >> report.md
```

---

## Tips

- Always start with passive techniques — they leave no trace on the target
- Certificate Transparency logs are one of the most reliable subdomain sources
- Combine multiple tools for coverage — no single tool finds everything
- Google dorking with `site:` and `filetype:` often uncovers exposed credentials
- Shodan and Censys cache historical data — check for services now firewalled
- EXIF data in images frequently contains GPS coordinates and device info
- Use Yandex for reverse image search when Google and TinEye fail
- HIBP uses k-anonymity for password checks — your full hash is never sent
- Document metadata reveals internal usernames, software versions, and paths
- GitHub search is case-insensitive for code but case-sensitive for filenames

---

## See Also

- recon
- social-engineering
- nmap

## References

- [Shodan Search Engine](https://www.shodan.io/)
- [Censys Search](https://search.censys.io/)
- [crt.sh Certificate Transparency](https://crt.sh/)
- [theHarvester](https://github.com/laramies/theHarvester)
- [Sherlock](https://github.com/sherlock-project/sherlock)
- [SpiderFoot](https://github.com/smicallef/spiderfoot)
- [Recon-ng](https://github.com/lanmaster53/recon-ng)
- [Have I Been Pwned](https://haveibeenpwned.com/)
- [ExifTool](https://exiftool.org/)
- [OSINT Framework](https://osintframework.com/)
