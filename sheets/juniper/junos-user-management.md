# Junos User Management (JNCIA-Junos)

Junos user account management, authentication, login classes, system services configuration (NTP, SNMP, syslog), rescue configuration, configuration archival, and configuration groups -- everything needed to secure and manage a Junos device from initial setup through production operations.

## Factory-Default State

```
# Out-of-box Junos device state:
# - No root password set (must set one before commit)
# - Only console access available (no SSH, no Telnet)
# - Management interface (fxp0/em0) unconfigured
# - All interfaces administratively down
# - No routing protocols configured
# - Factory-default configuration loaded

# View factory-default configuration
show configuration | display set

# Return device to factory defaults (DESTRUCTIVE)
request system zeroize
# - Erases all configuration and logs
# - Resets root password
# - Reboots to factory-default state
# - Use with extreme caution -- no confirmation prompt on some platforms

# Load factory-default config without zeroize
load factory-default
set system root-authentication plain-text-password
commit
```

## Initial Configuration

```
# Set hostname
set system host-name ROUTER01

# Set root password (required before first commit)
set system root-authentication plain-text-password
New password: ********
Retype new password: ********

# Or set root password with encrypted string
set system root-authentication encrypted-password "$6$abc..."

# Set root SSH key
set system root-authentication ssh-rsa "ssh-rsa AAAA..."

# Configure management interface
set interfaces fxp0 unit 0 family inet address 10.0.0.1/24
# or em0 on some platforms
set interfaces em0 unit 0 family inet address 10.0.0.1/24

# Set DNS
set system name-server 8.8.8.8
set system name-server 8.8.4.4
set system domain-name example.com
set system domain-search example.com

# Set default route for management traffic
set routing-options static route 0.0.0.0/0 next-hop 10.0.0.254

# Enable SSH access
set system services ssh root-login deny
set system services ssh protocol-version v2
set system services ssh max-sessions-per-connection 5

# Set system time zone
set system time-zone America/New_York

# Set login message
set system login message "Authorized access only. All activity is monitored."
```

## User Accounts

```
# Create a user account
set system login user admin class super-user
set system login user admin authentication plain-text-password
New password: ********
Retype new password: ********

# Set user with encrypted password
set system login user admin authentication encrypted-password "$6$abc..."

# Set user with SSH public key
set system login user admin authentication ssh-rsa "ssh-rsa AAAA..." key-comment

# User with multiple SSH keys
set system login user admin authentication ssh-rsa "ssh-rsa AAAA...key1" admin-laptop
set system login user admin authentication ssh-rsa "ssh-rsa AAAA...key2" admin-desktop

# Set full name for user
set system login user admin full-name "Network Administrator"

# Set user idle timeout (minutes)
set system login user admin idle-timeout 15

# Delete a user
delete system login user admin

# View configured users
show system login user

# View who is logged in
show system users
```

## Login Classes

### Built-In Classes

```
# Junos built-in login classes:

# super-user   -- full access to all commands and configuration
# operator     -- can view config but limited changes (clear, network, trace, view)
# read-only    -- can only view operational output (view)
# unauthorized -- no permissions at all (deny everything)

# Assign built-in class
set system login user viewer class read-only
set system login user ops class operator
set system login user admin class super-user
```

### Permission Bits

```
# Permission bits control what a login class can do:
#
# all                -- all permissions (equivalent to super-user)
# clear              -- clear (reset) learned info (clear commands)
# configure          -- enter configuration mode
# control            -- restart processes and modify runtime params
# field              -- field-debug commands (TAC-level)
# firewall           -- view/modify firewall filter config
# flow-tap           -- view/modify flow-tap config
# interface          -- view/modify interface config
# maintenance        -- system maintenance (reboot, upgrade)
# network            -- access network via ping, traceroute, ssh, telnet
# reset              -- restart software processes
# rollback           -- rollback to previous configurations
# routing            -- view/modify routing protocol config
# secret             -- view secret/password fields in config
# security           -- view/modify security config
# shell              -- access FreeBSD shell (start shell)
# snmp               -- view/modify SNMP config
# system             -- view/modify system-level config
# trace              -- view/modify traceoptions
# view               -- view operational commands and config
# view-configuration -- view configuration (no operational commands)
```

### Custom Login Classes

```
# Create a custom login class
set system login class NOC-ENGINEER permissions [view configure interface routing]
set system login class NOC-ENGINEER idle-timeout 30
set system login class NOC-ENGINEER login-alarms
set system login class NOC-ENGINEER login-tip

# Assign custom class to a user
set system login user noc1 class NOC-ENGINEER
set system login user noc1 authentication plain-text-password

# Custom class with deny-commands (regex)
set system login class JUNIOR-ADMIN permissions [view configure interface]
set system login class JUNIOR-ADMIN deny-commands "(request system zeroize|request system halt)"

# Custom class with allow-commands (regex override)
set system login class MONITORING permissions [view]
set system login class MONITORING allow-commands "show interfaces.*|show route.*|show bgp.*"

# Custom class with deny-configuration (restrict config access)
set system login class INTERFACE-ONLY permissions [view configure interface]
set system login class INTERFACE-ONLY deny-configuration "protocols|routing-options|system"

# Custom class with allow-configuration
set system login class FIREWALL-ADMIN permissions [view configure firewall]
set system login class FIREWALL-ADMIN allow-configuration "firewall"

# View login class permissions
show system login class
```

## User Authentication Methods

```
# LOCAL PASSWORD AUTHENTICATION
set system login user admin authentication plain-text-password
set system login user admin authentication encrypted-password "$6$..."

# SSH KEY AUTHENTICATION
set system login user admin authentication ssh-rsa "ssh-rsa AAAA..."
set system login user admin authentication ssh-ecdsa "ecdsa-sha2-nistp256 AAAA..."
set system login user admin authentication ssh-ed25519 "ssh-ed25519 AAAA..."

# RADIUS AUTHENTICATION
set system radius-server 10.1.1.100 secret "radiusSecret123"
set system radius-server 10.1.1.100 port 1812
set system radius-server 10.1.1.100 accounting-port 1813
set system radius-server 10.1.1.100 retry 3
set system radius-server 10.1.1.100 timeout 5
set system radius-server 10.1.1.100 source-address 10.0.0.1
set system radius-server 10.1.1.101 secret "radiusSecret123"   # backup server

# TACACS+ AUTHENTICATION
set system tacplus-server 10.1.1.200 secret "tacacsSecret123"
set system tacplus-server 10.1.1.200 port 49
set system tacplus-server 10.1.1.200 timeout 5
set system tacplus-server 10.1.1.200 source-address 10.0.0.1
set system tacplus-server 10.1.1.201 secret "tacacsSecret123"  # backup server

# AUTHENTICATION ORDER (tried in sequence)
set system authentication-order [radius tacplus password]
# radius   -- try RADIUS server first
# tacplus  -- try TACACS+ if RADIUS unavailable
# password -- fall back to local password if both fail

# TACACS+ with per-command authorization
set system tacplus-options authorization-time-interval 0
set system tacplus-options service-name junos-exec

# Verify authentication configuration
show system authentication-order
show system radius-server
show system tacplus-server
```

## Root Password Recovery

```
# Boot to single-user mode for password recovery:
#
# 1. Connect console cable to device
# 2. Reboot/power-cycle the device
# 3. When "Hit [Enter] to boot immediately" appears, press SPACE
# 4. At loader> prompt, type: boot -s
# 5. System boots to single-user mode
# 6. At # prompt, enter recovery mode:

recovery                     # enter recovery mode
set system root-authentication plain-text-password
New password: ********
Retype new password: ********
commit
exit                         # exit configuration mode
reboot                       # reboot normally

# NOTE: Physical console access required
# NOTE: boot -s bypasses normal authentication entirely
# NOTE: Secure the console port to prevent unauthorized recovery
```

## Rescue Configuration

```
# Save current committed config as rescue configuration
request system configuration rescue save

# Rescue config is stored separately from normal config
# Used as a known-good fallback configuration

# Delete rescue configuration
request system configuration rescue delete

# Rollback to rescue configuration (in configuration mode)
rollback rescue
commit

# View rescue configuration
show system configuration rescue

# When to use rescue config:
# - Before making risky configuration changes
# - As a baseline "known working" config
# - Recovery when current config is broken
# - Accessible from boot menu if device becomes unreachable
```

## Configuration Archival

```
# Automatically archive config to remote server on every commit
set system archival configuration transfer-on-commit

# Define archive sites (tried in order)
set system archival configuration archive-sites "ftp://10.1.1.50/configs/"
set system archival configuration archive-sites "scp://user@10.1.1.51/configs/" password "pass"

# Archive filename format includes hostname and timestamp
# Example: ROUTER01_juniper.conf.gz_20260405_120000

# Transfer interval (alternative to transfer-on-commit)
set system archival configuration transfer-interval 1440   # minutes (1440 = daily)

# Verify archival status
show system archival
```

## NTP Configuration

```
# Configure NTP server
set system ntp server 10.1.1.10
set system ntp server 10.1.1.11

# Set preferred NTP server
set system ntp server 10.1.1.10 prefer

# NTP boot server (used during system boot only)
set system ntp boot-server 10.1.1.10

# NTP authentication
set system ntp authentication-key 1 type md5 value "ntpSecret"
set system ntp trusted-key 1
set system ntp server 10.1.1.10 key 1

# Set NTP source address
set system ntp source-address 10.0.0.1

# Verify NTP
show ntp associations
show ntp status
show system uptime
```

## SNMP Configuration

```
# SNMPv2c community (read-only)
set snmp community public authorization read-only

# SNMPv2c community (read-write)
set snmp community private authorization read-write

# Restrict SNMP access by client address
set snmp community public clients 10.0.0.0/8
set snmp community public clients 0.0.0.0/0 restrict

# SNMP system info
set snmp name "ROUTER01"
set snmp location "Data Center Rack A1"
set snmp contact "noc@example.com"
set snmp description "Core Router 01"

# SNMPv3 (preferred -- encrypted and authenticated)
set snmp v3 usm local-engine user snmpV3user authentication-md5 authentication-password "authPass"
set snmp v3 usm local-engine user snmpV3user privacy-aes128 privacy-password "privPass"
set snmp v3 vacm security-to-group security-model usm security-name snmpV3user group SNMPV3-GRP
set snmp v3 vacm access group SNMPV3-GRP default-context-prefix security-model any security-level authentication read-view ALL
set snmp view ALL oid .1 include

# SNMP trap groups
set snmp trap-group TRAPS-TO-NMS targets 10.1.1.50
set snmp trap-group TRAPS-TO-NMS categories chassis
set snmp trap-group TRAPS-TO-NMS categories link
set snmp trap-group TRAPS-TO-NMS categories configuration
set snmp trap-group TRAPS-TO-NMS categories routing

# Verify SNMP
show snmp statistics
show snmp mib walk system
```

## Syslog Configuration

```
# Send logs to remote syslog server
set system syslog host 10.1.1.60 any notice
set system syslog host 10.1.1.60 authorization info
set system syslog host 10.1.1.60 interactive-commands info

# Log to local file
set system syslog file messages any notice
set system syslog file messages authorization info
set system syslog file interactive-commands interactive-commands any

# Syslog facilities:
# any, authorization, daemon, ftp, kernel, user, local0-local7,
# change-log, conflict-log, dfc, external, firewall, interactive-commands, pfe

# Severity levels (least to most severe):
# none, debug, info, notice, warning, error, critical, alert, emergency

# Log to console
set system syslog console "*.err"

# Syslog source address
set system syslog source-address 10.0.0.1

# Archive log files
set system syslog archive size 5m files 10

# Structured syslog
set system syslog host 10.1.1.60 structured-data

# Verify syslog
show log messages
show log messages | match "LOGIN"
show system syslog
```

## Configuration Groups

```
# Define a configuration group (template)
set groups DNS-SERVERS system name-server 8.8.8.8
set groups DNS-SERVERS system name-server 8.8.4.4
set groups DNS-SERVERS system domain-name example.com

set groups NTP-SERVERS system ntp server 10.1.1.10
set groups NTP-SERVERS system ntp server 10.1.1.11

set groups SYSLOG-STANDARD system syslog host 10.1.1.60 any notice
set groups SYSLOG-STANDARD system syslog host 10.1.1.60 authorization info
set groups SYSLOG-STANDARD system syslog file messages any notice

set groups INTERFACE-DEFAULTS interfaces <*> mtu 9192
set groups INTERFACE-DEFAULTS interfaces <*> unit <*> family inet mtu 9000

# Apply groups to the configuration
set apply-groups [DNS-SERVERS NTP-SERVERS SYSLOG-STANDARD INTERFACE-DEFAULTS]

# Apply groups except (exclude specific groups)
set apply-groups-except INTERFACE-DEFAULTS

# View group definitions
show groups

# View inherited configuration (shows where values come from)
show interfaces | display inheritance

# Groups use wildcard matching with angle brackets
# <*> matches any name at that hierarchy level
# Groups are applied top-down; first match wins
# apply-groups at top level applies everywhere
# apply-groups under a hierarchy applies only to that branch

# Delete a group
delete groups DNS-SERVERS

# Verify group application
show configuration | display inheritance
show configuration | display inheritance defaults
```

## Tips

- A Junos device out of box requires a root password before the first commit -- you cannot commit factory-default config without setting one.
- `request system zeroize` wipes everything including logs; use it only when decommissioning or returning a device.
- Authentication order matters: if RADIUS is listed first and the server is unreachable, there will be a timeout delay before falling back to the next method.
- Always configure at least one local super-user account as a fallback when using RADIUS/TACACS+ -- if both AAA servers are down, local accounts are your only way in.
- The `secret` permission bit allows viewing encrypted passwords in the configuration; omit it from custom classes unless absolutely necessary.
- The `shell` permission grants FreeBSD shell access (`start shell`) -- restrict this to super-users only in production.
- Rescue configuration is your safety net: save one before making risky changes, and you can rollback to it even from the boot menu.
- Configuration groups with `<*>` wildcards apply to all matching hierarchy nodes -- this is powerful but can cause unexpected inheritance if not carefully scoped.
- SNMPv3 is strongly preferred over v2c in production; v2c community strings are sent in cleartext.
- NTP `boot-server` is only used during initial boot; `server` is used for ongoing synchronization -- configure both for reliable time.

## See Also

- junos-architecture, junos-interfaces, junos-firewall-filters

## References

- [Juniper JNCIA-Junos Study Guide](https://www.juniper.net/us/en/training/certification/tracks/junos/jncia-junos.html)
- [Junos OS User Access and Authentication](https://www.juniper.net/documentation/us/en/software/junos/user-access/topics/concept/user-access-overview.html)
- [Junos OS Login Classes](https://www.juniper.net/documentation/us/en/software/junos/user-access/topics/concept/login-classes-overview.html)
- [Junos OS RADIUS Authentication](https://www.juniper.net/documentation/us/en/software/junos/user-access/topics/topic-map/radius-authentication.html)
- [Junos OS TACACS+ Authentication](https://www.juniper.net/documentation/us/en/software/junos/user-access/topics/topic-map/tacplus-authentication.html)
- [Junos OS NTP Configuration](https://www.juniper.net/documentation/us/en/software/junos/network-mgmt/topics/topic-map/ntp.html)
- [Junos OS SNMP Configuration](https://www.juniper.net/documentation/us/en/software/junos/network-mgmt/topics/topic-map/snmp-configuring.html)
- [Junos OS System Logging](https://www.juniper.net/documentation/us/en/software/junos/network-mgmt/topics/topic-map/syslog.html)
- [Junos OS Configuration Groups](https://www.juniper.net/documentation/us/en/software/junos/cli/topics/concept/configuration-groups-overview.html)
- [Junos OS Rescue Configuration](https://www.juniper.net/documentation/us/en/software/junos/cli/topics/topic-map/junos-software-rescue-configuration.html)
- Day One: Junos for IOS Engineers (Juniper Books)
