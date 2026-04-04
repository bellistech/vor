# Suricata (IDS/IPS/NSM Engine)

High-performance network threat detection engine supporting IDS, IPS, and network security monitoring with multi-threaded packet processing.

## Installation

```bash
# Debian/Ubuntu
sudo apt install suricata suricata-update

# CentOS/RHEL
sudo yum install epel-release
sudo yum install suricata

# From source (latest)
git clone https://github.com/OISF/suricata.git
cd suricata && git clone https://github.com/OISF/libhtp.git -b 0.5.x
./autogen.sh
./configure --prefix=/usr --sysconfdir=/etc --localstatedir=/var \
  --enable-nfqueue --enable-lua --enable-geoip
make && sudo make install-full

```

## Configuration (suricata.yaml)

```bash
# Key sections to configure
vars:
  address-groups:
    HOME_NET: "[192.168.0.0/16,10.0.0.0/8,172.16.0.0/12]"
    EXTERNAL_NET: "!$HOME_NET"
    HTTP_SERVERS: "$HOME_NET"
    DNS_SERVERS: "$HOME_NET"

# Set default log directory
default-log-dir: /var/log/suricata/

# AF_PACKET capture mode (recommended for Linux)
af-packet:
  - interface: eth0
    threads: auto
    cluster-id: 99
    cluster-type: cluster_flow
    defrag: yes
    use-mmap: yes
    ring-size: 200000

# NFQUEUE mode (inline IPS)
nfqueue:
  - mode: accept
    id: 0
    fail-open: yes

```

## Running Suricata

```bash
# IDS mode (AF_PACKET - passive monitoring)
sudo suricata -c /etc/suricata/suricata.yaml --af-packet=eth0

# IPS mode (NFQUEUE - inline blocking)
sudo suricata -c /etc/suricata/suricata.yaml -q 0

# Read from pcap file (offline analysis)
sudo suricata -c /etc/suricata/suricata.yaml -r capture.pcap

# Multi-interface capture
sudo suricata -c /etc/suricata/suricata.yaml --af-packet \
  --set af-packet.0.interface=eth0 \
  --set af-packet.1.interface=eth1

# Run as daemon
sudo suricata -c /etc/suricata/suricata.yaml --af-packet -D

# Reload rules without restart (via unix socket)
sudo suricatasc -c reload-rules

# Check configuration syntax
sudo suricata -T -c /etc/suricata/suricata.yaml

# Dump loaded config
sudo suricata --dump-config
```

## Rule Syntax

```bash
# Format: action protocol src_ip src_port -> dst_ip dst_port (options;)

# Alert on SSH brute force
alert ssh $EXTERNAL_NET any -> $HOME_NET 22 \
  (msg:"Possible SSH brute force"; \
   flow:to_server,established; \
   threshold:type both,track by_src,count 5,seconds 60; \
   sid:1000001; rev:1;)

# Drop SQL injection attempts (IPS mode)
drop http $EXTERNAL_NET any -> $HTTP_SERVERS any \
  (msg:"SQL Injection attempt"; \
   flow:to_server,established; \
   content:"UNION"; nocase; \
   content:"SELECT"; nocase; distance:0; \
   sid:1000002; rev:1;)

# Alert on DNS exfiltration (long queries)
alert dns any any -> any any \
  (msg:"DNS exfiltration - long query"; \
   dns.query; content:"."; offset:50; \
   sid:1000003; rev:1;)

# TLS certificate inspection
alert tls any any -> any any \
  (msg:"Self-signed TLS certificate"; \
   tls.cert_subject; content:"CN=localhost"; \
   sid:1000004; rev:1;)

# File extraction rule
alert http any any -> any any \
  (msg:"EXE download detected"; \
   fileext:"exe"; filestore; \
   sid:1000005; rev:1;)

# Lua scripting in rules
alert http any any -> any any \
  (msg:"Lua detection example"; \
   lua:detect.lua; \
   sid:1000006; rev:1;)
```

## Rule Management (suricata-update)

```bash
# Update rules from default sources (ET Open)
sudo suricata-update

# List available rule sources
sudo suricata-update list-sources

# Enable a source
sudo suricata-update enable-source et/open
sudo suricata-update enable-source oisf/trafficid
sudo suricata-update enable-source ptresearch/attackdetection

# Disable specific rules
echo "1:2027865" | sudo tee -a /etc/suricata/disable.conf
sudo suricata-update

# Modify rule actions (alert -> drop for IPS)
echo 're:ET MALWARE' | sudo tee -a /etc/suricata/modify.conf
# modify.conf format: <match> <from> <to>
# "re:ET MALWARE" alert drop

# Custom rule file
sudo cp local.rules /etc/suricata/rules/
sudo suricata-update --local /etc/suricata/rules/local.rules

# Check rule stats
sudo suricata-update --dump-sample-configs
```

## EVE JSON Logging

```bash
# EVE log output in suricata.yaml
outputs:
  - eve-log:
      enabled: yes
      filetype: regular
      filename: eve.json
      types:
        - alert
        - http:
            extended: yes
        - dns
        - tls:
            extended: yes
        - files:
            force-magic: yes
            force-hash: [md5, sha256]
        - flow
        - netflow
        - stats:
            totals: yes
            threads: yes

# Parse EVE JSON with jq
cat /var/log/suricata/eve.json | jq 'select(.event_type=="alert")'

# Get top triggered signatures
cat /var/log/suricata/eve.json | \
  jq -r 'select(.event_type=="alert") | .alert.signature' | \
  sort | uniq -c | sort -rn | head -20

# Extract DNS queries
cat /var/log/suricata/eve.json | \
  jq 'select(.event_type=="dns" and .dns.type=="query") | \
  {timestamp, src_ip: .src_ip, query: .dns.rrname}'

# TLS fingerprinting (JA3)
cat /var/log/suricata/eve.json | \
  jq 'select(.event_type=="tls") | {src: .src_ip, ja3: .tls.ja3.hash, sni: .tls.sni}'

# Send EVE to syslog
outputs:
  - eve-log:
      enabled: yes
      filetype: syslog
      identity: suricata
      facility: local5
      level: Info
```

## Multi-Threading and Performance

```bash
# Check CPU count for thread allocation
nproc

# suricata.yaml threading config
threading:
  set-cpu-affinity: yes
  cpu-affinity:
    - management-cpu-set:
        cpu: [0]
    - receive-cpu-set:
        cpu: [1]
    - worker-cpu-set:
        cpu: [2-7]
        mode: exclusive

# Stream engine tuning
stream:
  memcap: 256mb
  reassembly:
    memcap: 512mb
    depth: 1mb
    toserver-chunk-size: 2560
    toclient-chunk-size: 2560

# Detection engine tuning
detect:
  profile: high
  sgh-mpm-context: auto
  inspection-recursion-limit: 3000

# Check runtime performance
sudo suricatasc -c dump-counters | jq '.message.detect'
```

## File Extraction

```bash
# Enable file extraction in suricata.yaml
file-store:
  version: 2
  enabled: yes
  dir: /var/log/suricata/filestore
  write-fileinfo: yes
  stream-depth: 0
  force-hash: [sha256, md5]

# Extract all PE files
alert http any any -> any any \
  (msg:"PE file download"; \
   filemagic:"PE32"; filestore; \
   sid:1000010; rev:1;)

# List extracted files
ls /var/log/suricata/filestore/
find /var/log/suricata/filestore -name "*.meta" -exec cat {} \;
```

## Threshold Configuration

```bash
# /etc/suricata/threshold.config

# Suppress alerts from scanner host
suppress gen_id 1, sig_id 2027865, track by_src, ip 10.0.0.100

# Rate limit alerts (max 1 per 60s per source)
rate_filter gen_id 1, sig_id 1000001, track by_src, count 1, seconds 60, new_action alert, timeout 120

# Threshold (only alert after N hits)
threshold gen_id 1, sig_id 1000002, type threshold, track by_src, count 10, seconds 60

# Event filter (limit alert frequency)
event_filter gen_id 1, sig_id 1000003, type limit, track by_src, count 1, seconds 300
```

## IP Reputation

```bash
# Enable IP reputation in suricata.yaml
reputation-categories-file: /etc/suricata/iprep/categories.txt
default-reputation-path: /etc/suricata/iprep/

# categories.txt format: id,shortname,description
echo "1,BadHosts,Known malicious hosts" > /etc/suricata/iprep/categories.txt
echo "2,CnC,Command and Control servers" >> /etc/suricata/iprep/categories.txt

# reputation.list format: ip,category,score (0-127)
echo "203.0.113.50,1,120" > /etc/suricata/iprep/reputation.list
echo "198.51.100.23,2,127" >> /etc/suricata/iprep/reputation.list

# Rule using IP reputation
alert ip any any -> any any \
  (msg:"Traffic to known C2"; \
   iprep:dst,CnC,>,100; \
   sid:1000020; rev:1;)
```

## Tips

- Always run `suricata -T` to validate config before restarting the service
- Use `cluster_flow` AF_PACKET mode for load balancing across worker threads
- Set `stream.reassembly.depth` to limit memory usage on high-traffic sensors
- Enable `fail-open: yes` in NFQUEUE mode to prevent network outage if Suricata crashes
- Use JA3/JA3S hashes in TLS rules for encrypted traffic fingerprinting
- Pipe EVE JSON to Elasticsearch via Filebeat for searchable alert history
- Separate management, receive, and worker threads on dedicated CPU cores
- Use `suricata-update` cron jobs to keep rulesets current (daily recommended)
- Monitor `capture.kernel_drops` in stats to detect packet loss
- Enable `file-store` selectively; storing all files will fill disk rapidly
- Test rules against pcap files before deploying to live sensors

## See Also

- Zeek (Bro) for complementary network analysis
- Elasticsearch/Kibana for EVE JSON visualization
- Filebeat for log shipping
- Wireshark/tcpdump for packet capture
- Snort for alternative IDS engine comparison
- MITRE ATT&CK for mapping alerts to adversary techniques

## References

- [Suricata Documentation](https://docs.suricata.io/en/latest/)
- [Suricata Rule Format](https://docs.suricata.io/en/latest/rules/intro.html)
- [ET Open Ruleset](https://rules.emergingthreats.net/open/)
- [suricata-update Guide](https://suricata-update.readthedocs.io/en/latest/)
- [Suricata GitHub Repository](https://github.com/OISF/suricata)
- [OISF Training Materials](https://www.openinfosecfoundation.org/training/)
- [Stamus Networks Suricata Guides](https://www.stamus-networks.com/suricata)
