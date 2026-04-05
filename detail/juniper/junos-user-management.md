# Junos User Management -- AAA Architecture, Permission Design, and Authentication Best Practices

> *Deep dive into Junos authentication, authorization, and accounting: RADIUS/TACACS+ protocol flows, permission bit mechanics, custom login class design patterns, AAA failover strategies, password complexity enforcement, and production hardening for Juniper environments.*

---

## 1. RADIUS Authentication Flow

### Protocol Mechanics

RADIUS (Remote Authentication Dial-In User Service) uses UDP for transport and
operates on a simple request-response model. When a user attempts to log in to
a Junos device configured with RADIUS authentication:

```
+----------+         +-------------+         +--------------+
|  User    |         | Junos Device|         | RADIUS Server|
| (SSH/    |         | (NAS /      |         | (FreeRADIUS, |
|  Console)|         |  AAA Client)|         |  Cisco ISE)  |
+----+-----+         +------+------+         +------+-------+
     |                       |                        |
     | 1. SSH login          |                        |
     | username + password   |                        |
     |---------------------->|                        |
     |                       |                        |
     |                       | 2. Access-Request      |
     |                       | (UDP 1812)             |
     |                       | - Username             |
     |                       | - Password (encrypted) |
     |                       | - NAS-IP-Address       |
     |                       | - NAS-Port-Type        |
     |                       |----------------------->|
     |                       |                        |
     |                       |                        | 3. Validate credentials
     |                       |                        |    against user database
     |                       |                        |
     |                       | 4. Access-Accept       |
     |                       |    OR Access-Reject    |
     |                       |    + VSA attributes    |
     |                       |<-----------------------|
     |                       |                        |
     | 5. Login granted      |                        |
     |    or denied           |                        |
     |<----------------------|                        |
     |                       |                        |
     |                       | 6. Accounting-Request  |
     |                       | (UDP 1813, Start)      |
     |                       |----------------------->|
     |                       |                        |
```

### RADIUS Packet Structure

RADIUS encapsulates authentication data in attribute-value pairs (AVPs).
Juniper uses Vendor-Specific Attributes (VSAs) under vendor ID 2636 to
communicate Junos-specific authorization data:

| Attribute | Type | Purpose |
|:---|:---|:---|
| User-Name | Standard (1) | Login username |
| User-Password | Standard (2) | Encrypted password (shared secret + MD5) |
| NAS-IP-Address | Standard (4) | IP address of the Junos device |
| NAS-Port-Type | Standard (61) | Connection type (Virtual = SSH) |
| Juniper-Local-User-Name | VSA (2636, 1) | Maps to local Junos user template |
| Juniper-Allow-Commands | VSA (2636, 2) | Regex of permitted CLI commands |
| Juniper-Deny-Commands | VSA (2636, 3) | Regex of denied CLI commands |
| Juniper-Allow-Configuration | VSA (2636, 4) | Permitted config hierarchies |
| Juniper-Deny-Configuration | VSA (2636, 5) | Denied config hierarchies |
| Juniper-User-Permissions | VSA (2636, 6) | Permission bit string |

### RADIUS Shared Secret and Security

The RADIUS shared secret is used to encrypt the User-Password attribute and
to authenticate the integrity of RADIUS packets via the Response Authenticator.
The encryption is MD5-based:

```
Encrypted-Password = User-Password XOR MD5(Shared-Secret + Request-Authenticator)
```

This is a known weakness of RADIUS over UDP. Mitigations:

- Use long, random shared secrets (32+ characters)
- Restrict RADIUS traffic to a dedicated management VLAN
- Consider RADSEC (RADIUS over TLS, RFC 6614) where supported
- Use IPsec tunnels between NAS and RADIUS server
- TACACS+ encrypts the entire packet body, not just the password field

### Junos RADIUS Configuration Best Practices

```
# Primary and secondary RADIUS servers with source address pinning
set system radius-server 10.1.1.100 secret "$9$encrypted-secret"
set system radius-server 10.1.1.100 timeout 3
set system radius-server 10.1.1.100 retry 2
set system radius-server 10.1.1.100 source-address 10.0.0.1
set system radius-server 10.1.1.100 accounting-port 1813

set system radius-server 10.1.1.101 secret "$9$encrypted-secret"
set system radius-server 10.1.1.101 timeout 3
set system radius-server 10.1.1.101 retry 2
set system radius-server 10.1.1.101 source-address 10.0.0.1

# Authentication order with local fallback
set system authentication-order [radius password]

# RADIUS user template (remote users inherit this class)
set system login user REMOTE full-name "RADIUS Authenticated User"
set system login user REMOTE class operator
```

The RADIUS server returns the `Juniper-Local-User-Name` VSA to map the
authenticated user to a local Junos user template. If this VSA is not
returned, the user inherits the permissions of the template user configured
as the default RADIUS user.

---

## 2. TACACS+ Authentication Flow

### Protocol Mechanics

TACACS+ (Terminal Access Controller Access-Control System Plus) uses TCP port 49
and encrypts the entire packet body (not just the password). It separates
authentication, authorization, and accounting into distinct protocol exchanges:

```
+----------+         +-------------+         +--------------+
|  User    |         | Junos Device|         | TACACS+ Srvr |
| (SSH/    |         | (NAS /      |         | (Cisco ISE,  |
|  Console)|         |  AAA Client)|         |  tac_plus)   |
+----+-----+         +------+------+         +------+-------+
     |                       |                        |
     | 1. SSH login          |                        |
     |---------------------->|                        |
     |                       |                        |
     |                       | 2. AUTHEN START        |
     |                       | (TCP 49, encrypted)    |
     |                       | - Username             |
     |                       |----------------------->|
     |                       |                        |
     |                       | 3. AUTHEN REPLY        |
     |                       |    GETPASS (send pwd)  |
     |                       |<-----------------------|
     |                       |                        |
     |                       | 4. AUTHEN CONTINUE     |
     |                       | - Password (encrypted) |
     |                       |----------------------->|
     |                       |                        |
     |                       | 5. AUTHEN REPLY        |
     |                       |    PASS or FAIL        |
     |                       |<-----------------------|
     |                       |                        |
     |                       | --- AUTHORIZATION ---  |
     |                       |                        |
     |                       | 6. AUTHOR REQUEST      |
     |                       | - service=junos-exec   |
     |                       | - cmd=*                |
     |                       |----------------------->|
     |                       |                        |
     |                       | 7. AUTHOR REPLY        |
     |                       |    PASS_ADD attrs:     |
     |                       |    local-user-name=X   |
     |                       |    allow-commands=...  |
     |                       |<-----------------------|
     |                       |                        |
     |                       | --- ACCOUNTING ---     |
     |                       |                        |
     |                       | 8. ACCT REQUEST        |
     |                       | - start/stop/cmd       |
     |                       |----------------------->|
     |                       |                        |
```

### TACACS+ vs RADIUS Comparison

| Feature | RADIUS | TACACS+ |
|:---|:---|:---|
| Transport | UDP (1812/1813) | TCP (49) |
| Encryption | Password field only | Entire packet body |
| AAA separation | Combined authn + authz | Separate authn, authz, acct |
| Per-command authz | Limited (VSA-based) | Native per-command support |
| Accounting detail | Basic start/stop | Per-command logging |
| Multivendor | Excellent (RFC standard) | Good (de facto standard) |
| Failover detection | Slow (UDP timeout) | Fast (TCP connection state) |
| Bandwidth | Lower (UDP, smaller packets) | Higher (TCP overhead) |

### Per-Command Authorization with TACACS+

TACACS+ can authorize (or deny) individual CLI commands in real time. When
configured on Junos, every command the user types is sent to the TACACS+
server for approval before execution:

```
# Enable per-command authorization
set system tacplus-options authorization-time-interval 0
set system tacplus-options service-name junos-exec

# On the TACACS+ server (tac_plus.conf example):
# user = noc-engineer {
#     service = junos-exec {
#         local-user-name = NOC-TEMPLATE
#         allow-commands = "show.*|ping.*|traceroute.*"
#         deny-commands = "request system.*|file delete.*"
#     }
# }
```

This provides granular access control that is centrally managed on the TACACS+
server rather than distributed across every Junos device.

---

## 3. Permission Bits Mapping to CLI Commands

### Complete Permission Bit Reference

Each permission bit grants access to specific CLI command hierarchies and
configuration sections. Understanding the mapping is essential for designing
custom login classes:

| Permission Bit | Operational Commands Granted | Configuration Access |
|:---|:---|:---|
| **all** | Everything below | Full configuration |
| **clear** | `clear` (all clear commands) | None |
| **configure** | `configure` (enter config mode) | Depends on other bits |
| **control** | `restart`, `start`, `stop` (daemon control) | None |
| **field** | `request ... field-diagnostics` | None (TAC/JTAC use) |
| **firewall** | `show firewall` | `firewall`, `forwarding-options` |
| **flow-tap** | `show flow-tap` | `services flow-tap` |
| **interface** | `show interfaces` | `interfaces`, `class-of-service` |
| **maintenance** | `request system reboot/halt/snapshot`, `file` | None |
| **network** | `ping`, `traceroute`, `ssh`, `telnet` | None |
| **reset** | `restart` (software process restart) | None |
| **rollback** | `rollback N` (in config mode) | Previous configurations |
| **routing** | `show route`, `show bgp`, `show ospf` | `protocols`, `routing-options`, `routing-instances` |
| **secret** | View passwords/keys in config | `authentication` fields |
| **security** | `show security` | `security` |
| **shell** | `start shell` (FreeBSD shell access) | None |
| **snmp** | `show snmp` | `snmp` |
| **system** | `show system`, `show chassis` | `system` (except `login`) |
| **trace** | `show log`, `monitor` | `traceoptions` (all levels) |
| **view** | All `show` commands | View-only config access |
| **view-configuration** | None (no operational) | View-only config access |

### Permission Bit Interactions

Several permission bits have dependencies and interactions:

- **configure** alone grants the ability to enter configuration mode but not
  to view or change any config. You need additional bits like `interface`,
  `routing`, `firewall`, etc. to actually modify anything.

- **view** includes read-only access to operational commands AND configuration.
  It is the most commonly assigned base permission.

- **view-configuration** grants config viewing only, without any operational
  `show` commands. Rarely used alone.

- **secret** is dangerous: it reveals encrypted passwords, shared secrets,
  and authentication keys in the configuration. Never assign it to classes
  that do not need it.

- **all** is equivalent to super-user and grants every permission bit. It
  should only be assigned to the most trusted administrators.

### Building Permission Sets by Role

Common role-based permission sets:

```
# Help Desk / Level 1 (view and basic network tools)
permissions [view network clear]

# NOC Engineer / Level 2 (view, network tools, interface changes)
permissions [view network clear configure interface trace]

# Network Engineer / Level 3 (full config except system/security)
permissions [view network clear configure interface routing firewall
             rollback trace snmp]

# Senior Engineer (nearly full access, no shell or secret)
permissions [view network clear configure control interface routing
             firewall rollback trace snmp system security maintenance reset]

# Security Admin (security-focused access)
permissions [view configure security firewall]

# Auditor (read-only with trace access for log review)
permissions [view trace]
```

---

## 4. Custom Login Class Design Patterns

### Pattern 1: Layered Deny with Broad Permissions

Start with a generous permission set, then use `deny-commands` and
`deny-configuration` to remove specific dangerous capabilities:

```
set system login class SENIOR-NOC permissions [view configure interface routing
                                               firewall rollback trace network clear]

# Deny dangerous operational commands
set system login class SENIOR-NOC deny-commands "(request system zeroize|request system halt|request system reboot)"

# Deny access to system-level and security configuration
set system login class SENIOR-NOC deny-configuration "(system|security|snmp)"
```

This pattern is intuitive: "you can do everything except X." However, it is
fragile because new Junos features may introduce commands that should be denied
but are not yet in the deny list.

### Pattern 2: Minimal Permissions with Allow Overrides

Start with minimal permissions and use `allow-commands` and
`allow-configuration` to grant specific capabilities:

```
set system login class MONITORING permissions [view]

# Allow specific operational commands beyond basic view
set system login class MONITORING allow-commands "(show interfaces.*|show route.*|show bgp.*|show ospf.*|show chassis.*|ping.*|traceroute.*)"

# No allow-configuration needed -- view permission provides read-only config access
```

This pattern follows the principle of least privilege. It is more secure
because unknown/new commands are denied by default, but it requires ongoing
maintenance as new commands are needed.

### Pattern 3: TACACS+ Delegated Authorization

Offload all authorization decisions to the TACACS+ server, using a minimal
local class as a template:

```
# Local template class -- minimal permissions
set system login class TACACS-TEMPLATE permissions [view]

# TACACS+ user template
set system login user TACACS-USER full-name "TACACS+ Authenticated User"
set system login user TACACS-USER class TACACS-TEMPLATE

# TACACS+ server handles per-user, per-command authorization
set system tacplus-options authorization-time-interval 0
set system tacplus-options service-name junos-exec
```

Authorization logic lives centrally on the TACACS+ server, where it can be
updated for all devices simultaneously. The Junos device only needs the
template class.

### Pattern 4: Configuration Groups for Login Class Standardization

Use configuration groups to define login classes once and apply them
consistently across all devices:

```
set groups STANDARD-CLASSES system login class NOC permissions [view network clear configure interface trace]
set groups STANDARD-CLASSES system login class NOC idle-timeout 30
set groups STANDARD-CLASSES system login class NOC login-alarms

set groups STANDARD-CLASSES system login class ENGINEER permissions [view network clear configure interface routing firewall rollback trace snmp]
set groups STANDARD-CLASSES system login class ENGINEER idle-timeout 60

set groups STANDARD-CLASSES system login class AUDITOR permissions [view trace]
set groups STANDARD-CLASSES system login class AUDITOR idle-timeout 15

set apply-groups STANDARD-CLASSES
```

This pattern ensures consistency across a fleet of devices. When a login class
needs updating, change the group definition and commit on each device (or push
via automation).

---

## 5. AAA Best Practices for Juniper Environments

### Defense in Depth Authentication Strategy

A production Junos AAA deployment should implement multiple layers of
authentication and authorization:

```
Authentication Layers:
+----------------------------------------------------+
| Layer 1: Network Access Control                    |
|   - Management VLAN isolation                      |
|   - ACLs on management interfaces                  |
|   - lo0 firewall filter (restrict SSH source IPs)  |
+----------------------------------------------------+
| Layer 2: Primary Authentication (TACACS+)          |
|   - Centralized user database (AD/LDAP backend)   |
|   - Per-command authorization                      |
|   - Detailed command accounting                    |
+----------------------------------------------------+
| Layer 3: Secondary Authentication (RADIUS)         |
|   - Backup AAA server from different vendor        |
|   - Separate infrastructure for resilience         |
+----------------------------------------------------+
| Layer 4: Local Fallback                            |
|   - Emergency local accounts (super-user)          |
|   - Break-glass credentials in password vault      |
|   - Console-only access as last resort             |
+----------------------------------------------------+
```

### Authentication Order Recommendations

```
# Production recommendation: TACACS+ primary, local fallback
set system authentication-order [tacplus password]

# Alternative: RADIUS primary, TACACS+ secondary, local fallback
set system authentication-order [radius tacplus password]
```

Key considerations:

- **Always include `password` as the last entry.** If all remote AAA servers
  are unreachable (network outage, server failure), local accounts are the
  only way to access the device. Without `password` in the order, you are
  locked out if AAA servers are down.

- **TACACS+ over RADIUS when per-command authorization is needed.** TACACS+
  natively supports authorizing individual commands, while RADIUS can only
  provide authorization at login time via VSAs.

- **Set aggressive timeouts.** A 30-second RADIUS timeout with 3 retries means
  90 seconds of waiting before fallback. Use timeout 3, retry 2 for a maximum
  9-second delay per server.

- **Configure source-address.** Always pin RADIUS/TACACS+ traffic to a specific
  source address (usually the loopback or management interface). This ensures
  the AAA server's client definition matches regardless of which interface the
  traffic egresses.

### Accounting Configuration

```
# TACACS+ command accounting (logs every command to TACACS+ server)
set system accounting events login
set system accounting events change-log
set system accounting events interactive-commands
set system accounting destination tacplus server 10.1.1.200 secret "$9$..."

# Syslog backup for accounting (local record)
set system syslog file auth-log authorization info
set system syslog file auth-log interactive-commands info
set system syslog file auth-log match "UI_AUTH|UI_LOGIN|UI_CMDLINE"
```

### Emergency Access ("Break-Glass") Procedure

Every network should have a documented break-glass procedure for when all
AAA servers are unavailable:

1. **Maintain at least one local super-user account** with a strong, unique
   password stored in a password vault (not in anyone's memory).

2. **Rotate the break-glass password regularly** (quarterly minimum) and
   after every use.

3. **Alert on local account usage.** Configure syslog and/or SNMP traps to
   fire when a local account is used, since this indicates either an AAA
   outage or an unauthorized access attempt.

4. **Document the procedure.** The break-glass process should be in the
   runbook and tested during disaster recovery exercises.

```
# Break-glass account
set system login user emergency class super-user
set system login user emergency full-name "Emergency Break-Glass Account"
set system login user emergency authentication encrypted-password "$6$..."

# Alert on local login
set system syslog host 10.1.1.60 authorization warning
```

### SSH Hardening

```
# Disable root login over SSH (console only)
set system services ssh root-login deny

# Restrict SSH to v2 only
set system services ssh protocol-version v2

# Set connection limits
set system services ssh max-sessions-per-connection 5
set system services ssh connection-limit 10
set system services ssh rate-limit 5

# Disable password authentication (SSH keys only)
set system services ssh no-password-authentication

# Restrict SSH to management interface only
set system services ssh listen-address 10.0.0.1
```

---

## 6. Password Complexity Requirements

### Junos Built-In Password Controls

Junos enforces password complexity through configuration under
`system login password`:

```
# Minimum password length (default is 6, recommended 12+)
set system login password minimum-length 12

# Require mixed character types
set system login password minimum-changes 4
# minimum-changes = minimum number of character set changes
# (transitions between uppercase, lowercase, digits, specials)

# Maximum password length
set system login password maximum-length 128

# Password change restrictions
set system login password minimum-reuse 5
# Prevents reusing the last N passwords

# Format of stored passwords
# Junos stores passwords as SHA-512 hashes ($6$ prefix)
# Older devices may use MD5 ($1$) or DES -- avoid these
```

### Password Hash Formats

Junos uses standard Unix crypt formats:

| Prefix | Algorithm | Security |
|:---|:---|:---|
| `$1$` | MD5 | Weak -- do not use |
| `$5$` | SHA-256 | Acceptable |
| `$6$` | SHA-512 | Recommended |
| `$9$` | Junos-specific | Reversible encoding (not a hash -- used for shared secrets) |

The `$9$` format is notable: it is not a cryptographic hash but a reversible
encoding used for RADIUS/TACACS+ shared secrets and similar values. It provides
obfuscation (prevents casual shoulder-surfing) but not cryptographic security.
Anyone with access to the Junos device can decode `$9$` values.

### Operational Password Management

```
# View password settings
show system login password

# Force a user to change password at next login
# (Not natively supported -- use RADIUS/TACACS+ server-side password policies)

# Verify password hash format in use
show configuration system login user admin authentication | display set
# Look for $6$ prefix (SHA-512)
```

### Recommendations for Production

- Set `minimum-length 12` or higher
- Set `minimum-changes 4` to enforce complexity
- Set `minimum-reuse 5` to prevent password cycling
- Use SSH keys as the primary authentication method; passwords as backup only
- Store root and break-glass passwords in a vault (HashiCorp Vault, CyberArk)
- Use `$6$` (SHA-512) hashes exclusively; reject `$1$` (MD5) configurations
- Audit password hashes during compliance checks: any `$1$` hash is a finding

---

## Prerequisites

- junos-user-management (cheat sheet)
- junos-architecture (control plane / management plane concepts)
- networking fundamentals (IP addressing, TCP/UDP, encryption basics)

## References

- Juniper Networks JNCIA-Junos Study Guide (Official Certification Guide)
- Junos OS User Access and Authentication -- Juniper TechLibrary
- Junos OS RADIUS Authentication Configuration Guide -- Juniper TechLibrary
- Junos OS TACACS+ Configuration Guide -- Juniper TechLibrary
- RFC 2865 -- Remote Authentication Dial In User Service (RADIUS)
- RFC 8907 -- The TACACS+ Protocol
- RFC 6614 -- Transport Layer Security (TLS) Encryption for RADIUS
- "JUNOS Enterprise Routing" by Doug Marschke & Harry Reynolds (O'Reilly)
- "Day One: Junos for IOS Engineers" (Juniper Books)
- "Network Security with OpenSSL" by Viega, Messier, Chandra (O'Reilly)
- CIS Juniper Junos OS Benchmark (Center for Internet Security)
