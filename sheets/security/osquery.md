# osquery (SQL-Powered Endpoint Visibility)

SQL-based framework for querying operating system state as relational tables, enabling real-time endpoint visibility for security monitoring, compliance, and incident response.

## Installation

```bash
# Debian/Ubuntu
export OSQUERY_KEY=1484120AC4E9F8A1A577AEEE97A80C63C9D8B80B
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys $OSQUERY_KEY
sudo add-apt-repository 'deb [arch=amd64] https://pkg.osquery.io/deb deb main'
sudo apt update && sudo apt install osquery

# CentOS/RHEL
curl -L https://pkg.osquery.io/rpm/GPG | sudo tee /etc/pki/rpm-gpg/RPM-GPG-KEY-osquery
sudo yum-config-manager --add-repo https://pkg.osquery.io/rpm/osquery-s3-rpm.repo
sudo yum install osquery

# macOS
brew install osquery

# Windows (PowerShell)
choco install osquery
# Or download MSI from https://osquery.io/downloads
```

## Interactive Shell (osqueryi)

```bash
# Launch interactive shell
osqueryi

# Basic system info
SELECT * FROM system_info;
SELECT hostname, cpu_brand, physical_memory FROM system_info;

# List all tables
.tables

# Describe table schema
.schema processes
.schema file

# Show query plan (performance check)
.explain SELECT * FROM processes WHERE name = 'sshd';

# Output modes
.mode csv
.mode json
.mode pretty
.mode line

# Run query from command line
osqueryi --json "SELECT pid, name, path FROM processes WHERE on_disk = 0"
```

## Process Monitoring

```bash
# All running processes with resource usage
SELECT pid, name, path, cmdline, uid,
       resident_size/1048576 AS mem_mb,
       total_time
FROM processes
ORDER BY resident_size DESC LIMIT 20;

# Processes without backing binary (in-memory only)
SELECT pid, name, path, cmdline
FROM processes
WHERE on_disk = 0;

# Processes listening on network ports
SELECT p.pid, p.name, p.path, l.port, l.protocol, l.address
FROM listening_ports l
JOIN processes p ON l.pid = p.pid
WHERE l.port != 0
ORDER BY l.port;

# Process tree (parent-child)
SELECT p.pid, p.name, p.path, p.parent,
       pp.name AS parent_name, pp.path AS parent_path
FROM processes p
LEFT JOIN processes pp ON p.parent = pp.pid
WHERE p.name = 'bash';

# Unsigned or suspicious processes (macOS)
SELECT p.pid, p.name, p.path, s.authority, s.identifier
FROM processes p
JOIN signature s ON p.path = s.path
WHERE s.signed = 0;

# Processes with open file handles
SELECT p.pid, p.name, pof.path AS open_file
FROM processes p
JOIN process_open_files pof ON p.pid = pof.pid
WHERE pof.path LIKE '/etc/%';
```

## Network Monitoring

```bash
# Active network connections
SELECT pid, local_address, local_port,
       remote_address, remote_port, state
FROM process_open_sockets
WHERE state = 'ESTABLISHED'
  AND family = 2;  -- IPv4

# Connections to external IPs (non-RFC1918)
SELECT p.name, p.pid, s.remote_address, s.remote_port
FROM process_open_sockets s
JOIN processes p ON s.pid = p.pid
WHERE s.remote_address NOT LIKE '10.%'
  AND s.remote_address NOT LIKE '192.168.%'
  AND s.remote_address NOT LIKE '172.1%'
  AND s.remote_address != '127.0.0.1'
  AND s.remote_address != ''
  AND s.state = 'ESTABLISHED';

# DNS resolvers and ARP table
SELECT * FROM dns_resolvers;
SELECT address, mac, interface FROM arp_cache;
```

## File Integrity Monitoring (FIM)

```bash
# osquery.conf FIM configuration
{
  "file_paths": {
    "system_binaries": [
      "/usr/bin/%%",
      "/usr/sbin/%%",
      "/bin/%%",
      "/sbin/%%"
    ],
    "configuration": [
      "/etc/%%"
    ],
    "ssh_keys": [
      "/home/%/.ssh/%%",
      "/root/.ssh/%%"
    ]
  },
  "file_accesses": ["system_binaries"]
}

# Query file changes
SELECT target_path, category, action, time, md5, sha256
FROM file_events
WHERE action IN ('CREATED', 'MODIFIED', 'DELETED')
ORDER BY time DESC LIMIT 50;

# Check specific file hashes
SELECT path, filename, md5, sha256, size, mtime
FROM hash
WHERE path = '/usr/bin/sudo';

# Find recently modified files in sensitive directories
SELECT path, filename, mtime, size, uid, gid, mode
FROM file
WHERE directory = '/etc/'
  AND mtime > (strftime('%s','now') - 3600);

# SUID/SGID binaries
SELECT path, filename, uid, gid, mode
FROM file
WHERE directory IN ('/usr/bin/', '/usr/sbin/', '/usr/local/bin/')
  AND (mode LIKE '%4%' OR mode LIKE '%2%')
  AND substring(mode, 1, 1) >= '4';
```

## Scheduled Queries and Packs

```bash
# /etc/osquery/osquery.conf
{
  "options": {
    "config_plugin": "filesystem",
    "logger_plugin": "filesystem",
    "logger_path": "/var/log/osquery",
    "disable_logging": "false",
    "schedule_splay_percent": "10",
    "events_expiry": "3600",
    "database_path": "/var/osquery/osquery.db",
    "verbose": "false",
    "worker_threads": "2",
    "enable_monitor": "true",
    "host_identifier": "hostname"
  },
  "schedule": {
    "process_snapshot": {
      "query": "SELECT pid, name, path, cmdline FROM processes;",
      "interval": 300,
      "snapshot": true
    },
    "listening_ports": {
      "query": "SELECT pid, port, protocol FROM listening_ports WHERE port != 0;",
      "interval": 60
    },
    "ssh_authorized_keys": {
      "query": "SELECT * FROM authorized_keys;",
      "interval": 3600
    },
    "crontab_changes": {
      "query": "SELECT * FROM crontab;",
      "interval": 300
    }
  },
  "packs": {
    "incident-response": "/usr/share/osquery/packs/incident-response.conf",
    "vuln-management": "/usr/share/osquery/packs/vuln-management.conf",
    "hardware-monitoring": "/usr/share/osquery/packs/hardware-monitoring.conf"
  }
}

# Run osqueryd (daemon mode)
sudo osqueryd --config_path=/etc/osquery/osquery.conf \
  --flagfile=/etc/osquery/osquery.flags

# Check daemon status
sudo osqueryctl status
sudo osqueryctl config-check
```

## Security Investigation Queries

```bash
# Persistence mechanisms - crontabs
SELECT command, path, minute, hour, day_of_month
FROM crontab;

# Startup items (Linux)
SELECT name, path, source, type, status
FROM startup_items;

# Kernel modules loaded
SELECT name, size, status, address
FROM kernel_modules
WHERE status = 'Live';

# Users and login history
SELECT uid, username, shell, directory
FROM users;

SELECT username, type, time, host, pid
FROM last
ORDER BY time DESC LIMIT 20;

# SSH authorized keys audit
SELECT ak.uid, u.username, ak.key_file, ak.key, ak.algorithm
FROM authorized_keys ak
JOIN users u ON ak.uid = u.uid;

# Installed packages (vulnerability surface)
SELECT name, version, source
FROM deb_packages
ORDER BY name;  -- or rpm_packages

# Docker containers
SELECT id, name, image, status, started_at
FROM docker_containers;

# Browser extensions (Chrome)
SELECT u.username, ce.name, ce.identifier, ce.version, ce.path
FROM chrome_extensions ce
JOIN users u ON ce.uid = u.uid;
```

## Fleet Management

```bash
# FleetDM enrollment
# Generate enrollment secret
fleetctl get enroll-secret

# Deploy osquery with Fleet config
osqueryd --tls_hostname=fleet.internal:8080 \
  --tls_server_certs=/etc/osquery/fleet-ca.pem \
  --enroll_secret_path=/etc/osquery/enroll.secret \
  --enroll_tls_endpoint=/api/osquery/enroll \
  --config_plugin=tls \
  --config_tls_endpoint=/api/osquery/config \
  --logger_plugin=tls \
  --logger_tls_endpoint=/api/osquery/log \
  --distributed_plugin=tls \
  --distributed_tls_read_endpoint=/api/osquery/distributed/read \
  --distributed_tls_write_endpoint=/api/osquery/distributed/write

# FleetDM live query via API
curl -X POST https://fleet.internal:8080/api/v1/fleet/queries/run \
  -H "Authorization: Bearer $FLEET_TOKEN" \
  -d '{"query":"SELECT * FROM processes WHERE name = \"nc\"",
       "selected":{"hosts":[1,2,3]}}'

```

## Auto Table Construction (ATC)

```bash
# ATC config - create custom tables from structured data
{
  "auto_table_construction": {
    "custom_app_inventory": {
      "query": "SELECT name, version, install_path FROM apps",
      "path": "/opt/app-inventory/apps.sqlite",
      "columns": ["name", "version", "install_path"],
      "platform": "linux"
    }
  }
}

# Query ATC tables
SELECT * FROM custom_app_inventory WHERE version LIKE '1.%';
```

## Tips

- Use `WHERE` clauses aggressively; unbounded queries on large tables (file, hash) will consume significant resources
- Set `schedule_splay_percent` to 10-20% to prevent all hosts from running scheduled queries simultaneously
- Use `snapshot` mode for queries that return full state; use differential (default) for change detection
- Combine osquery with a fleet manager (FleetDM) for live queries across thousands of endpoints
- Monitor `osquery_info` and `osquery_schedule` tables to detect query performance issues
- Use `JOIN` across tables to correlate processes with their network connections, files, and users
- Ship results to a SIEM via the TLS logger plugin for centralized analysis and alerting
- Test queries in `osqueryi` with `.explain` before adding them to production schedules
- Use packs to organize related queries (incident response, compliance, vulnerability management)
- Keep scheduled query intervals proportional to data volatility: processes every 60s, packages every hour
- Enable FIM events only on directories that matter; monitoring all of `/` will overwhelm the system

## See Also

- SIEM for centralizing osquery results
- CIS Benchmarks for compliance queries via osquery
- MITRE ATT&CK for mapping osquery detections to techniques
- Suricata for network-level visibility (complement to endpoint)
- FleetDM for osquery fleet management

## References

- [osquery Documentation](https://osquery.readthedocs.io/en/stable/)
- [osquery Schema Reference](https://osquery.io/schema/)
- [osquery GitHub Repository](https://github.com/osquery/osquery)
- [FleetDM Documentation](https://fleetdm.com/docs)
- [osquery Packs Collection](https://github.com/osquery/osquery/tree/master/packs)
- [Palantir osquery Configuration](https://github.com/palantir/osquery-configuration)
- [osquery Performance Guide](https://osquery.readthedocs.io/en/stable/deployment/performance-safety/)
