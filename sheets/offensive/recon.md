# Reconnaissance (Passive & Active Recon for Penetration Testing)

> For authorized security testing, CTF competitions, and educational purposes only.

Systematic information gathering before engaging a target. Passive recon leaves no
footprint on the target; active recon touches target systems directly.

---

## Passive Reconnaissance

### WHOIS Lookups

```bash
# Domain registration details
whois example.com

# IP WHOIS (find ASN, netblock owner)
whois 93.184.216.34

# Reverse WHOIS — find domains registered by same org
# (use tools like ViewDNS.info, DomainTools, or whoxy.com)

# Parse registrant info
whois example.com | grep -iE 'registrant|admin|tech|name server'
```

### DNS Enumeration

```bash
# Basic DNS queries with dig
dig example.com A              # IPv4 address
dig example.com AAAA           # IPv6 address
dig example.com MX             # Mail servers
dig example.com NS             # Name servers
dig example.com TXT            # TXT records (SPF, DKIM, verification tokens)
dig example.com SOA            # Start of authority
dig example.com ANY            # All records (often blocked)

# Short output
dig +short example.com A

# Trace delegation chain
dig +trace example.com

# Reverse DNS
dig -x 93.184.216.34

# Zone transfer attempt (AXFR)
dig axfr example.com @ns1.example.com

# DNS over specific server
dig @8.8.8.8 example.com A
```

### Subdomain Enumeration

```bash
# Subfinder — passive subdomain discovery
subfinder -d example.com -o subs.txt
subfinder -d example.com -silent | httpx -silent  # pipe to httpx for live check

# Amass — comprehensive enumeration (passive mode)
amass enum -passive -d example.com -o amass_passive.txt

# Amass — active mode (touches target DNS)
amass enum -active -d example.com -o amass_active.txt

# Amass with config for API keys (Shodan, Censys, VirusTotal, etc.)
amass enum -d example.com -config ~/.config/amass/config.ini

# Fierce — DNS brute force and zone transfer
fierce --domain example.com
fierce --domain example.com --wordlist /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt

# DNSRecon
dnsrecon -d example.com -t std    # standard enumeration
dnsrecon -d example.com -t brt    # brute force subdomains
dnsrecon -d example.com -t axfr   # zone transfer

# Assetfinder
assetfinder --subs-only example.com
```

### Certificate Transparency

```bash
# Query crt.sh for subdomains via SSL certificates
curl -s "https://crt.sh/?q=%25.example.com&output=json" | jq -r '.[].name_value' | sort -u

# Filter for unique subdomains
curl -s "https://crt.sh/?q=%25.example.com&output=json" \
  | jq -r '.[].name_value' \
  | sed 's/\*\.//g' \
  | sort -u > ct_subs.txt

# Certspotter
curl -s "https://api.certspotter.com/v1/issuances?domain=example.com&include_subdomains=true" \
  | jq -r '.[].dns_names[]' | sort -u
```

### Google Dorking

```bash
# Find login pages
site:example.com inurl:login OR inurl:admin OR inurl:portal

# Exposed files
site:example.com filetype:pdf OR filetype:doc OR filetype:xlsx
site:example.com filetype:sql OR filetype:bak OR filetype:log

# Directory listings
site:example.com intitle:"index of"

# Configuration files
site:example.com filetype:env OR filetype:cfg OR filetype:conf
site:example.com filetype:xml inurl:web.config

# Error messages revealing stack info
site:example.com intext:"sql syntax" OR intext:"mysql_fetch" OR intext:"pg_query"

# Sensitive directories
site:example.com inurl:"/wp-admin" OR inurl:"/phpmyadmin" OR inurl:"/.git"

# Exposed API keys/tokens
site:example.com intext:"api_key" OR intext:"apikey" OR intext:"access_token"

# Cached/old versions
cache:example.com
```

### Shodan & Censys

```bash
# Shodan CLI
shodan init YOUR_API_KEY
shodan host 93.184.216.34                  # info about a specific IP
shodan search "hostname:example.com"       # search by hostname
shodan search "org:\"Example Inc\""        # search by organization
shodan search "ssl.cert.subject.cn:example.com"  # by SSL cert CN
shodan count "apache port:443 country:US"  # count matching hosts
shodan download results.json.gz "hostname:example.com"  # bulk download

# Censys CLI
censys search "services.tls.certificates.leaf.names: example.com"
censys view 93.184.216.34
```

### OSINT Techniques

```bash
# theHarvester — emails, subdomains, IPs from public sources
theHarvester -d example.com -b google,bing,linkedin,dnsdumpster

# Email format discovery
# Check hunter.io, phonebook.cz, or email-format.com

# GitHub dorking for secrets
# Search GitHub: "example.com" password OR secret OR api_key OR token
# Use trufflehog or gitleaks on found repos
trufflehog github --org=examplecorp
gitleaks detect --source /path/to/repo

# Wayback Machine — historical snapshots
curl -s "http://web.archive.org/cdx/search/cdx?url=*.example.com/*&output=json&fl=original&collapse=urlkey" \
  | jq -r '.[][]' | sort -u

# waybackurls tool
echo "example.com" | waybackurls | sort -u > wayback_urls.txt

# Pastebin/paste site search (use IntelligenceX, Dehashed, or psbdmp.ws)

# Social media OSINT
# Tools: Sherlock (username search), Maltego, SpiderFoot
sherlock targetusername
```

---

## Active Reconnaissance

### Network Scanning with Nmap

```bash
# Host discovery (ping sweep)
nmap -sn 192.168.1.0/24
nmap -sn -PE -PP -PM 10.0.0.0/24   # ICMP echo, timestamp, netmask

# TCP SYN scan (stealth, default with root)
nmap -sS -p- 10.0.0.1              # all 65535 ports
nmap -sS -p 1-1024 10.0.0.1        # first 1024 ports
nmap -sS --top-ports 1000 10.0.0.1 # top 1000 common ports

# TCP connect scan (no root required)
nmap -sT -p 80,443,8080 10.0.0.1

# UDP scan
nmap -sU --top-ports 100 10.0.0.1

# Version detection
nmap -sV -p 22,80,443 10.0.0.1
nmap -sV --version-intensity 5 10.0.0.1

# OS detection
nmap -O 10.0.0.1
nmap -A 10.0.0.1   # OS + version + scripts + traceroute

# NSE scripts
nmap --script=default 10.0.0.1
nmap --script=vuln 10.0.0.1
nmap --script=http-enum,http-headers 10.0.0.1 -p 80
nmap --script=smb-vuln* 10.0.0.1
nmap --script=ssl-heartbleed 10.0.0.1 -p 443

# Output formats
nmap -oN scan.txt 10.0.0.1         # normal
nmap -oX scan.xml 10.0.0.1         # XML
nmap -oG scan.gnmap 10.0.0.1       # grepable
nmap -oA scan_all 10.0.0.1         # all formats

# Evasion techniques
nmap -sS -T2 -f --data-length 24 10.0.0.1  # slow, fragment, pad
nmap -D RND:5 10.0.0.1                      # decoy scan
nmap -S 10.0.0.99 -e eth0 10.0.0.1          # spoof source
```

### Masscan (Fast Port Scanning)

```bash
# Scan full port range at high speed
masscan -p1-65535 10.0.0.0/24 --rate=10000 -oL masscan_out.txt

# Specific ports
masscan -p80,443,8080 10.0.0.0/16 --rate=50000

# Banner grabbing
masscan -p80 10.0.0.0/24 --banners --rate=1000
```

### Service Fingerprinting

```bash
# Grab banners manually
nc -nv 10.0.0.1 22          # SSH banner
nc -nv 10.0.0.1 80          # HTTP banner
echo "HEAD / HTTP/1.0\r\n\r\n" | nc 10.0.0.1 80

# curl for HTTP headers
curl -I https://example.com
curl -sI https://example.com | grep -i "server\|x-powered-by\|x-aspnet"

# whatweb — web technology fingerprinting
whatweb example.com
whatweb -a 3 example.com   # aggressive mode

# Wappalyzer CLI (if installed)
wappalyzer https://example.com

# sslscan — TLS/SSL analysis
sslscan example.com
sslscan --no-failed example.com

# testssl.sh
./testssl.sh example.com
./testssl.sh --vulnerable example.com
```

### Web Content Discovery

```bash
# Gobuster — directory/file brute force
gobuster dir -u https://example.com -w /usr/share/seclists/Discovery/Web-Content/raft-medium-directories.txt
gobuster dir -u https://example.com -w /usr/share/wordlists/dirbuster/directory-list-2.3-medium.txt -x php,html,txt
gobuster vhost -u https://example.com -w /usr/share/seclists/Discovery/DNS/subdomains-top1million-5000.txt

# ffuf — fast web fuzzer
ffuf -u https://example.com/FUZZ -w /usr/share/seclists/Discovery/Web-Content/common.txt
ffuf -u https://example.com/FUZZ -w wordlist.txt -mc 200,301,302  # match codes
ffuf -u https://example.com/FUZZ -w wordlist.txt -fc 404          # filter codes
ffuf -u https://example.com/FUZZ -w wordlist.txt -fs 4242         # filter by size

# feroxbuster — recursive content discovery
feroxbuster -u https://example.com -w /usr/share/seclists/Discovery/Web-Content/raft-medium-directories.txt

# Nikto — web vulnerability scanner
nikto -h https://example.com
nikto -h https://example.com -Tuning x 6   # test for specific categories
```

---

## Tips

- Always start with passive recon to minimize detection risk before going active
- Combine multiple subdomain tools — no single tool finds everything
- Use `-oA` with nmap to save all output formats for later analysis
- Rate-limit active scans on production targets to avoid causing outages
- Keep detailed notes of every finding — recon feeds into all later phases
- Check robots.txt and sitemap.xml early for hidden paths
- Validate subdomain lists with `httpx` or `httprobe` to find live hosts
- API keys for Shodan, Censys, VirusTotal, SecurityTrails dramatically improve passive recon
- Use `scope` files to avoid accidentally scanning out-of-scope targets

---

## References

- [Nmap Reference Guide](https://nmap.org/book/man.html)
- [Amass User Guide](https://github.com/owasp-amass/amass)
- [Subfinder](https://github.com/projectdiscovery/subfinder)
- [Shodan CLI Docs](https://cli.shodan.io/)
- [Google Hacking Database (GHDB)](https://www.exploit-db.com/google-hacking-database)
- [SecLists Wordlists](https://github.com/danielmiessler/SecLists)
- [theHarvester](https://github.com/laramies/theHarvester)
- [ffuf](https://github.com/ffuf/ffuf)
- [Gobuster](https://github.com/OJ/gobuster)
- [crt.sh Certificate Search](https://crt.sh/)
