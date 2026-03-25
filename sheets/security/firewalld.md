# firewalld (Dynamic Firewall Manager)

Zone-based firewall for RHEL, CentOS, Fedora, and derivatives using nftables/iptables backends.

## Zones

### Managing Zones

```bash
# List all available zones
firewall-cmd --get-zones

# Get the default zone
firewall-cmd --get-default-zone

# Set the default zone
firewall-cmd --set-default-zone=internal

# List active zones and their bound interfaces
firewall-cmd --get-active-zones

# List everything configured in a zone
firewall-cmd --zone=public --list-all

# List all zones with their full configuration
firewall-cmd --list-all-zones

# Create a custom zone (permanent only)
firewall-cmd --permanent --new-zone=webapp
firewall-cmd --reload
```

### Interfaces and Sources

```bash
# Assign an interface to a zone
firewall-cmd --zone=trusted --change-interface=eth1

# Remove an interface from a zone
firewall-cmd --zone=trusted --remove-interface=eth1

# Bind a source IP range to a zone
firewall-cmd --zone=internal --add-source=192.168.1.0/24

# Remove a source from a zone
firewall-cmd --zone=internal --remove-source=192.168.1.0/24

# Query which zone handles an interface
firewall-cmd --get-zone-of-interface=eth0
```

## Services

```bash
# List services enabled in the default zone
firewall-cmd --list-services

# List all predefined service names
firewall-cmd --get-services

# Add a service to the default zone
firewall-cmd --add-service=http

# Add a service to a specific zone
firewall-cmd --zone=public --add-service=https

# Remove a service
firewall-cmd --remove-service=ftp

# Query whether a service is enabled
firewall-cmd --query-service=ssh
```

## Ports

```bash
# Open a TCP port
firewall-cmd --add-port=8080/tcp

# Open a UDP port
firewall-cmd --add-port=5353/udp

# Open a range of ports
firewall-cmd --add-port=3000-3100/tcp

# Remove a port
firewall-cmd --remove-port=8080/tcp

# List open ports
firewall-cmd --list-ports
```

## Runtime vs Permanent

```bash
# Runtime change (lost on reload/reboot)
firewall-cmd --add-service=http

# Permanent change (survives reload/reboot, not active until reload)
firewall-cmd --permanent --add-service=http

# Reload to apply permanent changes
firewall-cmd --reload

# Make current runtime configuration permanent
firewall-cmd --runtime-to-permanent

# Common pattern: apply both at once
firewall-cmd --permanent --add-service=http && firewall-cmd --reload
```

## Rich Rules

```bash
# Allow a specific IP to access SSH
firewall-cmd --add-rich-rule='rule family="ipv4" source address="10.0.0.5" service name="ssh" accept'

# Deny traffic from a subnet
firewall-cmd --add-rich-rule='rule family="ipv4" source address="192.168.50.0/24" reject'

# Rate-limit connections (accept with limit)
firewall-cmd --add-rich-rule='rule service name="http" accept limit value="10/m"'

# Log and drop traffic from a source
firewall-cmd --add-rich-rule='rule family="ipv4" source address="10.99.0.0/16" log prefix="BLOCKED:" level="warning" drop'

# Remove a rich rule (must match exactly)
firewall-cmd --remove-rich-rule='rule family="ipv4" source address="10.0.0.5" service name="ssh" accept'

# List rich rules
firewall-cmd --list-rich-rules
```

## Masquerading and Port Forwarding

```bash
# Enable masquerading (NAT) on a zone
firewall-cmd --zone=external --add-masquerade

# Check masquerade status
firewall-cmd --zone=external --query-masquerade

# Forward port 80 to a different internal host
firewall-cmd --add-forward-port=port=80:proto=tcp:toport=8080:toaddr=192.168.1.10

# Forward port locally (same host, different port)
firewall-cmd --add-forward-port=port=443:proto=tcp:toport=8443

# Remove a port forward
firewall-cmd --remove-forward-port=port=80:proto=tcp:toport=8080:toaddr=192.168.1.10
```

## Direct Rules

```bash
# Add a direct iptables rule (bypass zone logic)
firewall-cmd --direct --add-rule ipv4 filter INPUT 0 -s 10.0.0.0/8 -j ACCEPT

# List direct rules
firewall-cmd --direct --get-all-rules

# Remove a direct rule
firewall-cmd --direct --remove-rule ipv4 filter INPUT 0 -s 10.0.0.0/8 -j ACCEPT
```

## ICMP Blocking

```bash
# List ICMP type names
firewall-cmd --get-icmptypes

# Block ping (echo-request)
firewall-cmd --add-icmp-block=echo-request

# Remove ICMP block
firewall-cmd --remove-icmp-block=echo-request

# Enable ICMP block inversion (block everything except listed types)
firewall-cmd --add-icmp-block-inversion
```

## Panic Mode

```bash
# Enable panic mode (drops ALL traffic immediately)
firewall-cmd --panic-on

# Disable panic mode
firewall-cmd --panic-off

# Check panic mode status
firewall-cmd --query-panic
```

## IP Sets

```bash
# Create an ipset (permanent only)
firewall-cmd --permanent --new-ipset=blocklist --type=hash:ip

# Add entries to an ipset
firewall-cmd --permanent --ipset=blocklist --add-entry=10.0.0.5
firewall-cmd --permanent --ipset=blocklist --add-entry=10.0.0.6

# Use an ipset as a source in a zone
firewall-cmd --permanent --zone=drop --add-source=ipset:blocklist

# List ipset entries
firewall-cmd --permanent --ipset=blocklist --get-entries

# Remove an ipset
firewall-cmd --permanent --delete-ipset=blocklist
firewall-cmd --reload
```

## Lockdown

```bash
# Enable lockdown (restrict firewall changes to allowed apps only)
firewall-cmd --lockdown-on

# Disable lockdown
firewall-cmd --lockdown-off

# Query lockdown status
firewall-cmd --query-lockdown
```

## Tips

- Always test rules at runtime first, then make them permanent once confirmed working.
- Use `--runtime-to-permanent` to save a known-good runtime config instead of re-entering each rule with `--permanent`.
- The default zone applies to any interface not explicitly assigned to another zone.
- Rich rules are evaluated in order; place more specific rules before broader ones.
- Direct rules bypass zone processing and should be used sparingly; prefer rich rules.
- Use `firewall-cmd --reload` after permanent changes; `--complete-reload` restarts the entire firewall and drops active connections.
- Panic mode is for emergencies only; it drops all inbound and outbound traffic including established connections.
- IP sets are more efficient than multiple rich rules when blocking or allowing many individual addresses.
- Check the backend (`nftables` vs `iptables`) with `firewall-cmd --version` and `/etc/firewalld/firewalld.conf`.
- Use `journalctl -u firewalld` to troubleshoot firewalld issues.

## References

- [firewalld Documentation](https://firewalld.org/documentation/)
- [firewalld Zone Configuration](https://firewalld.org/documentation/zone/)
- [firewalld Rich Language](https://firewalld.org/documentation/man-pages/firewalld.richlanguage.html)
- [firewall-cmd(1) Man Page](https://man7.org/linux/man-pages/man1/firewall-cmd.1.html)
- [firewalld(1) Man Page](https://man7.org/linux/man-pages/man1/firewalld.1.html)
- [Red Hat RHEL 9 — Using firewalld](https://access.redhat.com/documentation/en-us/red_hat_enterprise_linux/9/html/configuring_firewalls_and_packet_filters/using-and-configuring-firewalld_firewall-packet-filters)
- [Arch Wiki — firewalld](https://wiki.archlinux.org/title/Firewalld)
- [Fedora Quick Docs — firewalld](https://docs.fedoraproject.org/en-US/quick-docs/firewalld/)
