# iptables (Packet Filtering & NAT)

Linux kernel packet filter — rules organized in tables (filter, nat, mangle, raw) and chains (INPUT, OUTPUT, FORWARD, PREROUTING, POSTROUTING).

## Viewing Rules

### List rules
```bash
iptables -L                          # filter table, all chains
iptables -L -n -v                    # numeric, verbose (packet counts)
iptables -L -n -v --line-numbers     # with rule numbers (for delete/insert)
iptables -t nat -L -n -v             # NAT table
iptables -t mangle -L -n -v          # mangle table
iptables -S                          # dump rules as commands (restorable)
iptables -t nat -S                   # NAT rules as commands
```

## Chain Management

### Set default policy
```bash
iptables -P INPUT DROP               # drop all incoming by default
iptables -P FORWARD DROP             # drop all forwarded
iptables -P OUTPUT ACCEPT            # allow all outgoing
```

### Custom chains
```bash
iptables -N LOGGING                  # create chain
iptables -X LOGGING                  # delete empty chain
iptables -F INPUT                    # flush all rules in INPUT
iptables -F                          # flush all chains in filter table
iptables -t nat -F                   # flush NAT rules
```

## Adding Rules

### Basic allow/deny
```bash
iptables -A INPUT -p tcp --dport 22 -j ACCEPT          # allow SSH
iptables -A INPUT -p tcp --dport 80 -j ACCEPT           # allow HTTP
iptables -A INPUT -p tcp --dport 443 -j ACCEPT          # allow HTTPS
iptables -A INPUT -p icmp -j ACCEPT                     # allow ping
iptables -A INPUT -s 10.0.0.0/8 -j DROP                 # block RFC1918
iptables -A INPUT -i lo -j ACCEPT                       # allow loopback
```

### Stateful rules (conntrack)
```bash
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -m conntrack --ctstate INVALID -j DROP
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW -j ACCEPT
```

### Rate limiting
```bash
# Limit SSH to 3 new connections per minute per source
iptables -A INPUT -p tcp --dport 22 -m conntrack --ctstate NEW \
  -m limit --limit 3/min --limit-burst 3 -j ACCEPT

# Limit ICMP
iptables -A INPUT -p icmp --icmp-type echo-request \
  -m limit --limit 1/s --limit-burst 4 -j ACCEPT
```

### Insert and delete
```bash
iptables -I INPUT 1 -p tcp --dport 443 -j ACCEPT    # insert at position 1
iptables -D INPUT -p tcp --dport 80 -j ACCEPT        # delete by spec
iptables -D INPUT 3                                   # delete rule #3
```

### Source/dest matching
```bash
iptables -A INPUT -s 192.168.1.100 -j ACCEPT
iptables -A INPUT -s 192.168.1.0/24 -p tcp --dport 3306 -j ACCEPT
iptables -A OUTPUT -d 10.0.0.0/8 -j DROP
```

### Multiport
```bash
iptables -A INPUT -p tcp -m multiport --dports 80,443,8080 -j ACCEPT
```

## NAT

### SNAT (source NAT — outbound)
```bash
iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE                  # dynamic IP
iptables -t nat -A POSTROUTING -o eth0 -j SNAT --to-source 203.0.113.5  # static IP
```

### DNAT (destination NAT — port forwarding)
```bash
# Forward port 8080 on host to 10.0.0.2:80
iptables -t nat -A PREROUTING -p tcp --dport 8080 -j DNAT --to-destination 10.0.0.2:80
# Must also allow the forwarded traffic
iptables -A FORWARD -p tcp -d 10.0.0.2 --dport 80 -j ACCEPT
```

### Redirect (local port redirect)
```bash
iptables -t nat -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
```

### Enable IP forwarding (required for NAT)
```bash
sysctl -w net.ipv4.ip_forward=1
echo 'net.ipv4.ip_forward = 1' >> /etc/sysctl.conf
```

## Logging

### Log before dropping
```bash
iptables -N LOGGING
iptables -A INPUT -j LOGGING
iptables -A LOGGING -m limit --limit 5/min -j LOG --log-prefix "iptables-drop: " --log-level 4
iptables -A LOGGING -j DROP
```

## Save and Restore

### Persist rules
```bash
iptables-save > /etc/iptables/rules.v4          # save current rules
iptables-restore < /etc/iptables/rules.v4       # restore rules
ip6tables-save > /etc/iptables/rules.v6         # IPv6 rules
```

### On Debian/Ubuntu
```bash
apt install iptables-persistent
netfilter-persistent save
netfilter-persistent reload
```

## Common Patterns

### Minimal server firewall
```bash
iptables -F
iptables -P INPUT DROP
iptables -P FORWARD DROP
iptables -P OUTPUT ACCEPT
iptables -A INPUT -i lo -j ACCEPT
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -p tcp -m multiport --dports 22,80,443 -j ACCEPT
iptables -A INPUT -p icmp --icmp-type echo-request -j ACCEPT
```

### Block a specific IP
```bash
iptables -I INPUT 1 -s 203.0.113.99 -j DROP
```

### Allow established outbound, block inbound
```bash
iptables -A INPUT -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -j DROP
```

## Tips

- Always allow loopback (`-i lo`) and ESTABLISHED,RELATED before restrictive rules
- Rule order matters — first match wins; put specific rules before general ones
- `iptables -S` output can be piped back into `iptables-restore` (with header)
- Use `-I` (insert) to add urgent rules at the top; `-A` (append) for normal additions
- MASQUERADE is slower than SNAT but handles dynamic IPs (DHCP, PPPoE)
- `iptables` is being replaced by `nftables` — new deployments should prefer `nft`
- Don't lock yourself out: test with `at now + 5 minutes <<< 'iptables -F'` before applying strict rules over SSH
- IPv6 uses `ip6tables` — a separate ruleset that must be configured independently

## See Also

- nftables, ip, tcpdump, ufw, firewalld

## References

- [man iptables](https://man7.org/linux/man-pages/man8/iptables.8.html)
- [man iptables-extensions](https://man7.org/linux/man-pages/man8/iptables-extensions.8.html)
- [Netfilter/iptables Project Documentation](https://www.netfilter.org/documentation/)
- [Netfilter — Packet Flow Diagram](https://www.netfilter.org/documentation/HOWTO/packet-filtering-HOWTO.html)
- [Linux Kernel — Netfilter Documentation](https://www.kernel.org/doc/html/latest/networking/netfilter.html)
- [iptables Tutorial by Oskar Andreasson](https://www.frozentux.net/iptables-tutorial/iptables-tutorial.html)
- [Red Hat — iptables and ip6tables](https://docs.redhat.com/en/documentation/red_hat_enterprise_linux/7/html/security_guide/sec-using_firewalls)
- [Debian Wiki — iptables](https://wiki.debian.org/iptables)
