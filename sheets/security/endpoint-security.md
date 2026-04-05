# Endpoint Security

Protecting endpoints (workstations, servers, mobile devices) from threats using prevention, detection, and response capabilities including Cisco Secure Endpoint (AMP), EDR, and outbreak control.

## Concepts

### EDR Architecture

- **Endpoint Detection and Response (EDR):** Continuous monitoring of endpoint activity, recording events for threat detection, investigation, and response
- **Lightweight connector** installed on each endpoint reports telemetry to cloud or on-prem management console
- **Cloud analytics** correlate events across all endpoints for retrospective detection
- **Cisco Secure Endpoint** (formerly AMP for Endpoints) is Cisco's EDR/EPP platform

### Deployment Modes

- **Cloud-managed:** Connectors report to Cisco cloud; management via `console.amp.cisco.com`
- **Private Cloud:** On-prem virtual appliance for air-gapped or regulated environments; same connector, local disposition database
- **Managed vs Unmanaged:** Managed connectors enforce policy centrally; unmanaged endpoints have no connector and rely on network-level detection only

### Detection Engines

- **TETRA (Cisco Offline Engine):** Full antivirus engine with local signature database; works offline without cloud connectivity
- **ClamAV:** Open-source AV engine used in on-prem and Linux deployments
- **Exploit Prevention (EP):** Memory injection detection, process hollowing, DLL hijacking prevention; protects against fileless attacks
- **Orbital:** Advanced query engine for real-time endpoint investigation; SQL-like queries against endpoint state
- **Machine Learning (Malware Analytics):** Static and dynamic analysis of unknown files via Cisco Threat Grid sandbox
- **Script Protection:** Monitors PowerShell, WScript, CScript, and other scripting engines for malicious behavior

### File Disposition

```
File Submission Flow:
Endpoint --> SHA-256 lookup --> Cloud disposition
                                  |
                          +-------+-------+
                          |       |       |
                        Clean  Malicious  Unknown
                          |       |       |
                        Allow   Block   Sandbox
                                        Analysis
```

- **Clean:** Known good file; allowed to execute
- **Malicious:** Known bad file; blocked, quarantined, or removed
- **Unknown:** Not yet classified; may be sent to sandbox (Threat Grid) for dynamic analysis
- **Custom:** Administrator-defined disposition via Simple Custom Detections (SHA-256 block lists)
- Dispositions can change over time via retrospective security

### Retrospective Security

- Files initially marked clean can be reclassified as malicious when new threat intelligence arrives
- **Retrospective alerts** notify when a previously allowed file is now known malicious
- The platform tracks every file execution across all endpoints, enabling instant identification of all affected devices
- Retrospective detection closes the window between initial compromise and detection

## Trajectory

### File Trajectory

```
# Shows the complete lifecycle of a file across the organization
# Tracks: first seen, last seen, every endpoint that executed it
# Accessible from Dashboard > File Trajectory

Key data points:
- SHA-256 hash of the file
- Every endpoint that downloaded or executed the file
- Parent process that created/launched the file
- Network connections made after execution
- Timeline of disposition changes
```

### Device Trajectory

```
# Shows all activity on a single endpoint over time
# Tracks: file executions, network connections, process trees
# Accessible from Dashboard > Device Trajectory > [hostname]

Key data points:
- Files created, moved, or executed
- Network connections (IP, port, protocol, direction)
- Process ancestry (parent-child relationships)
- Registry modifications (Windows)
- DNS queries
- Events correlated with threat detections
```

## Policy Configuration

### Connector Groups and Policies

```
# Policy hierarchy:
# Organization > Group > Policy > Connector

# Groups organize endpoints by function, location, or OS
# Each group has exactly one policy applied
# Default groups: Audit, Protect, Triage, Server

# Policy components:
# - File scanning (TETRA, ClamAV)
# - Network monitoring (Device Flow Correlation)
# - Exploit Prevention settings
# - Exclusion sets
# - Proxy settings
# - Update schedules
```

### Exclusion Management

```
# Exclusion types:
# 1. Cisco-maintained exclusions (auto-updated, OS-specific)
# 2. Custom exclusions (admin-defined)

# Exclusion scopes:
# - Path exclusions: C:\Program Files\AppName\*
# - Process exclusions: exclude scanning when specific process accesses files
# - Wildcard exclusions: *.log, *.tmp
# - Threat exclusion: allow specific detection names

# Best practice: apply minimal exclusions
# Over-excluding creates blind spots attackers can exploit
# Always test exclusions in Audit mode before Protect mode
```

### Connector Deployment

```bash
# Windows — MSI installer (silent)
msiexec /i amp_installer.msi /quiet /norestart /log install.log

# macOS — PKG installer
sudo installer -pkg amp_connector.pkg -target /

# Linux — RPM-based
sudo rpm -ivh amp_connector.rpm
# Linux — DEB-based
sudo dpkg -i amp_connector.deb

# Verify connector status
# Windows
"C:\Program Files\Cisco\AMP\ampupdater.exe" /status

# Linux
/opt/cisco/amp/bin/ampcli status

# macOS
/Library/Application\ Support/Cisco/AMP\ for\ Endpoints\ Connector/ampcli status
```

## Outbreak Control

### Simple Custom Detections (SCD)

```
# Block files by SHA-256 hash
# Dashboard > Outbreak Control > Simple Custom Detections

# Upload a text file containing one SHA-256 per line:
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
a1b2c3d4e5f6... (additional hashes)

# Use cases:
# - Immediately block a known-bad file across all endpoints
# - Emergency response to new malware not yet in signature databases
# - Block files identified during incident investigation
```

### Advanced Custom Detections (ACD)

```
# ClamAV-based custom signatures for on-prem or advanced matching
# Upload custom ClamAV signature sets (.ndb, .ldb, .hdb format)

# Example ClamAV hash signature (HDB format):
# md5:filesize:signaturename
44d88612fea8a8f36de82e1278abb02f:68:EICAR-Test-File

# Example logical signature (LDB format):
# SignatureName;TargetType;LogicalExpression;Subsig0;Subsig1
Custom.Malware;Target:0;(0&1);content:"malicious_string";content:"secondary_indicator"
```

### Application Blocking

```
# Block applications by SHA-256 regardless of disposition
# Prevents execution even if the file is classified as clean
# Dashboard > Outbreak Control > Application Blocking

# Use cases:
# - Block unauthorized software (crypto miners, remote access tools)
# - Enforce software compliance policy
# - Block specific versions of legitimate tools used maliciously
#   (e.g., older PsExec, vulnerable Java versions)
```

### IP Block and Allow Lists

```
# Block or allow network connections to specific IPs
# Dashboard > Outbreak Control > IP Block & Allow Lists

# Block list: connections to listed IPs generate alerts and can be blocked
# Allow list: override cloud-based IP reputation for known-good addresses

# Use cases:
# - Block known C2 (command and control) server IPs
# - Allow internal IPs flagged as false positives
# - Emergency blocking during active incident
```

## Indicators of Compromise (IoC)

```
# IoC types tracked by Secure Endpoint:
# - File hashes (SHA-256, SHA-1, MD5)
# - IP addresses (C2 servers, exfil destinations)
# - Domains (malicious, DGA-generated)
# - URLs (phishing, exploit kit landing pages)
# - File paths (known malware drop locations)
# - Registry keys (persistence mechanisms)
# - Mutex names (malware coordination)

# Importing IoCs:
# Dashboard > Accounts > Threat Intelligence
# Supports STIX/TAXII feeds for automated IoC ingestion
# Cisco Talos threat intelligence integrated by default
```

## Threat Response and SecureX Integration

```
# SecureX orchestration ties Secure Endpoint into broader security stack
# Pivot from an alert in Secure Endpoint to:
# - Umbrella DNS logs (was the domain queried elsewhere?)
# - Secure Email (did the file arrive via email?)
# - Secure Firewall (was the C2 IP seen in network logs?)
# - Meraki dashboard (which network segment?)

# Threat Response investigation:
# 1. Enter observable (hash, IP, domain) in SecureX ribbon
# 2. Automatic lookup across all integrated products
# 3. Relation graph shows connections between observables
# 4. Take action: block hash, isolate host, quarantine email

# Endpoint isolation:
# Remotely isolate a compromised endpoint from the network
# Maintains management connectivity to Secure Endpoint cloud
# Dashboard > Computers > [host] > Start Isolation
```

## ISE Integration

```
# Cisco ISE (Identity Services Engine) integration enables:
# - Posture assessment: is the connector installed and healthy?
# - Adaptive Network Control (ANC): quarantine endpoint via ISE
#   when Secure Endpoint detects a compromise
# - Authorization policy based on endpoint compliance status

# Flow:
# 1. Secure Endpoint detects malware on endpoint
# 2. pxGrid sends event to ISE
# 3. ISE applies ANC policy (quarantine VLAN, ACL, or SGT)
# 4. Endpoint is isolated at the network level
# 5. Remediation workflow begins

# ISE authorization policy example:
# IF AMP_Threat_Detected = TRUE
# THEN assign Quarantine_ACL + Quarantine_VLAN
```

## Troubleshooting

### Connector Issues

```bash
# Check connector health
/opt/cisco/amp/bin/ampcli status
# Look for: Connected, Scan engine up-to-date, Policy applied

# Connector not connecting to cloud
# Verify network connectivity to cloud management
curl -v https://intake.amp.cisco.com
curl -v https://cloud-ec.amp.cisco.com

# Check connector logs
# Windows: C:\Program Files\Cisco\AMP\sfc.exe.log
# Linux: /var/log/cisco/amp.log
# macOS: /Library/Logs/Cisco/amp.log

# High CPU from scanning — check exclusions
# Enable debug logging temporarily
/opt/cisco/amp/bin/ampcli debuglevel 1
# Review which files are being scanned excessively
# Add appropriate exclusions for high-churn directories
```

### False Positive Handling

```
# 1. Identify the detection name and SHA-256
# 2. Submit to Cisco Talos for reclassification if legitimate file
# 3. Add SHA-256 to allow list (Outbreak Control > Application Allow)
# 4. Add path or process exclusion if recurring
# 5. Avoid over-broad exclusions (don't exclude entire C:\ drive)
```

## Tips

- Deploy in Audit mode first to identify false positives before switching to Protect mode; this prevents business disruption.
- Use Orbital queries for proactive threat hunting; query for IOCs across all endpoints without waiting for detection.
- Keep TETRA definitions current for offline protection; schedule updates during low-usage windows.
- Integrate with ISE via pxGrid for automated network quarantine of compromised endpoints.
- Use file trajectory to understand the full blast radius of a compromise before beginning remediation.
- Set up automated actions (quarantine, isolation) for high-confidence detections to reduce response time.
- Create separate policies for servers vs workstations; servers need different exclusions and scan schedules.
- Review the Compromised Hosts dashboard daily; it correlates multiple low-severity events into high-confidence detections.
- Use IP block lists for emergency C2 blocking during active incidents while longer-term firewall rules are prepared.
- Enable exploit prevention on all endpoints; it stops fileless attacks that traditional AV misses entirely.

## See Also

- cloud-security, security-operations, ipsec, radius

## References

- [Cisco Secure Endpoint User Guide](https://docs.amp.cisco.com/en/SecureEndpoint/Secure%20Endpoint%20User%20Guide.pdf)
- [Cisco Secure Endpoint Deployment Strategy Guide](https://www.cisco.com/c/en/us/products/collateral/security/amp-for-endpoints/deployment-strategy-guide.html)
- [Cisco SecureX Documentation](https://docs.securex.cisco.com/)
- [Cisco ISE and AMP Integration Guide](https://www.cisco.com/c/en/us/support/docs/security/identity-services-engine/215236-ise-and-amp-integration-guide.html)
- [ClamAV Documentation](https://docs.clamav.net/)
- [MITRE ATT&CK Framework](https://attack.mitre.org/)
- [Cisco Talos Intelligence](https://talosintelligence.com/)
