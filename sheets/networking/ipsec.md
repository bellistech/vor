# IPsec (Internet Protocol Security)

> Encrypt and authenticate IP traffic using Security Associations, IKE negotiation, and ESP/AH encapsulation.

## Concepts

### IKE Phases

```
# IKE Phase 1 (IKE_SA_INIT + IKE_AUTH in IKEv2)
# - Establishes a secure channel between peers
# - Negotiates crypto parameters (encryption, hash, DH group)
# - Authenticates peers (PSK or certificates)
# - Result: IKE SA (Security Association)

# IKE Phase 2 (CREATE_CHILD_SA in IKEv2)
# - Negotiates IPsec SAs for actual data traffic
# - Establishes ESP/AH parameters
# - Optionally enables PFS (Perfect Forward Secrecy)
# - Result: Child SA (pair of unidirectional SAs)
```

### Security Protocols

```
# ESP (Encapsulating Security Payload) — protocol 50
# - Provides confidentiality (encryption) + integrity + authentication
# - Most commonly used; supports NAT traversal (UDP 4500)

# AH (Authentication Header) — protocol 51
# - Provides integrity + authentication only (no encryption)
# - Authenticates entire packet including outer IP header
# - Incompatible with NAT (modifies IP header fields AH protects)
```

### Tunnel vs Transport Mode

```
# Tunnel mode (default for site-to-site VPNs)
# - Encapsulates entire original IP packet
# - New outer IP header added
# [New IP Header][ESP Header][Original IP Header][Payload][ESP Trailer]

# Transport mode (host-to-host)
# - Only encrypts/authenticates the payload
# - Original IP header preserved
# [Original IP Header][ESP Header][Payload][ESP Trailer]
```

### Perfect Forward Secrecy (PFS)

```
# PFS performs a new Diffie-Hellman exchange for each Child SA
# Compromise of one session key does not compromise others
# Enabled by specifying a DH group in the child/phase2 proposal
```

## strongSwan Configuration

### ipsec.conf (Legacy Format)

```conf
# /etc/ipsec.conf
config setup
    charondebug="ike 2, knl 2, cfg 2"

conn site-to-site
    type=tunnel
    keyexchange=ikev2
    left=203.0.113.1
    leftsubnet=10.1.0.0/16
    leftcert=server.pem
    leftid=@vpn.example.com
    right=198.51.100.1
    rightsubnet=10.2.0.0/16
    rightid=@remote.example.com
    ike=aes256-sha256-modp2048!
    esp=aes256-sha256-modp2048!
    auto=start
```

### swanctl.conf (Modern vici Format)

```conf
# /etc/swanctl/swanctl.conf
connections {
    site-to-site {
        version = 2
        local_addrs = 203.0.113.1
        remote_addrs = 198.51.100.1

        local {
            auth = pubkey
            certs = server.pem
            id = vpn.example.com
        }
        remote {
            auth = pubkey
            id = remote.example.com
        }

        children {
            net-net {
                local_ts = 10.1.0.0/16
                remote_ts = 10.2.0.0/16
                esp_proposals = aes256-sha256-modp2048
                start_action = start
                dpd_action = restart
            }
        }

        proposals = aes256-sha256-modp2048
    }
}
```

### PSK Authentication

```conf
# /etc/ipsec.secrets (legacy)
203.0.113.1 198.51.100.1 : PSK "MySharedSecret123"

# swanctl.conf PSK
secrets {
    ike-psk {
        secret = "MySharedSecret123"
        id-1 = 203.0.113.1
        id-2 = 198.51.100.1
    }
}
```

### Certificate-Based Auth

```bash
# Generate CA key and certificate
ipsec pki --gen --type rsa --size 4096 --outform pem > ca-key.pem
ipsec pki --self --ca --lifetime 3650 \
    --in ca-key.pem --dn "CN=VPN CA" --outform pem > ca-cert.pem

# Generate server key and certificate
ipsec pki --gen --type rsa --size 2048 --outform pem > server-key.pem
ipsec pki --pub --in server-key.pem | \
    ipsec pki --issue --lifetime 730 \
    --cacert ca-cert.pem --cakey ca-key.pem \
    --dn "CN=vpn.example.com" \
    --san vpn.example.com \
    --flag serverAuth --outform pem > server-cert.pem

# Install certificates (strongSwan)
cp ca-cert.pem /etc/swanctl/x509ca/
cp server-cert.pem /etc/swanctl/x509/
cp server-key.pem /etc/swanctl/private/
```

## libreswan Equivalents

### Configuration Differences

```conf
# /etc/ipsec.conf (libreswan)
conn site-to-site
    ikev2=insist                          # strongSwan: keyexchange=ikev2
    left=203.0.113.1
    leftsubnet=10.1.0.0/16
    leftcert=server.pem
    leftid=@vpn.example.com
    right=198.51.100.1
    rightsubnet=10.2.0.0/16
    rightid=@remote.example.com
    phase2alg=aes256-sha256;modp2048      # strongSwan: esp=
    ike=aes256-sha256;modp2048            # same keyword
    auto=start
```

## Show Commands

### Status and Diagnostics

```bash
# strongSwan — legacy ipsec tool
ipsec status                              # brief SA overview
ipsec statusall                           # full detail
ipsec up site-to-site                     # initiate connection
ipsec down site-to-site                   # tear down connection
ipsec restart                             # restart daemon

# strongSwan — swanctl (modern)
swanctl --list-sas                        # list active SAs
swanctl --list-conns                      # list configured connections
swanctl --initiate --child net-net        # bring up child SA
swanctl --terminate --ike site-to-site    # tear down IKE SA
swanctl --load-all                        # reload all config and creds
swanctl --log                             # live charon log stream

# Kernel SA and policy inspection
ip xfrm state                             # show kernel-level SAs
ip xfrm policy                            # show SPD (Security Policy Database)

# libreswan equivalents
ipsec whack --trafficstatus               # traffic counters
ipsec whack --status                      # full status
```

## Troubleshooting

### IKE Negotiation Failures

```bash
# Enable verbose logging (strongSwan)
# In /etc/strongswan.conf or charon section:
#   charondebug = "ike 4, net 4, cfg 4, knl 4"

# Common failures:
# - NO_PROPOSAL_CHOSEN: mismatched encryption/hash/DH proposals
# - AUTHENTICATION_FAILED: wrong PSK, expired/untrusted cert, wrong ID
# - TS_UNACCEPTABLE: traffic selector (subnet) mismatch
# - INVALID_KE_PAYLOAD: DH group mismatch between peers

# Check negotiation with tcpdump
tcpdump -n -i eth0 port 500 or port 4500  # IKE traffic
```

### MTU and PMTUD Issues

```bash
# IPsec overhead reduces effective MTU
# ESP tunnel mode: ~57-73 bytes overhead (depending on cipher + IV)

# Set lower MTU on tunnel interface
ip link set mtu 1400 dev ipsec0

# Clamp MSS for TCP traffic passing through
iptables -t mangle -A FORWARD -p tcp --tcp-flags SYN,RST SYN \
    -o eth0 -j TCPMSS --set-mss 1360

# Test PMTUD
ping -M do -s 1400 -c 3 10.2.0.1         # send with DF bit set
```

## Tips

- Always prefer IKEv2 over IKEv1 for simplicity, security, and MOBIKE support.
- Use `!` after proposal strings in ipsec.conf to enforce strict matching (no fallback).
- NAT traversal encapsulates ESP in UDP 4500; ensure firewalls allow it.
- Enable DPD (Dead Peer Detection) to recover from stale tunnels automatically.
- For road warriors, use virtual IP assignment (`rightsourceip` pool) with EAP auth.
- Keep SA lifetimes aligned between peers to avoid rekeying race conditions.

## References

- [RFC 4301 — Security Architecture for the Internet Protocol](https://www.rfc-editor.org/rfc/rfc4301)
- [RFC 7296 — Internet Key Exchange Protocol Version 2 (IKEv2)](https://www.rfc-editor.org/rfc/rfc7296)
- [RFC 4303 — IP Encapsulating Security Payload (ESP)](https://www.rfc-editor.org/rfc/rfc4303)
- [RFC 4302 — IP Authentication Header (AH)](https://www.rfc-editor.org/rfc/rfc4302)
- [RFC 6071 — IPsec and IKE Document Roadmap](https://www.rfc-editor.org/rfc/rfc6071)
- [strongSwan Documentation](https://docs.strongswan.org/)
- [strongSwan — IKEv2 Configuration Examples](https://docs.strongswan.org/docs/5.9/config/IKEv2.html)
- [Libreswan Documentation and Man Pages](https://libreswan.org/man/)
- [Linux Kernel — XFRM (IPsec) Documentation](https://www.kernel.org/doc/html/latest/networking/xfrm_device.html)
- [man ip-xfrm](https://man7.org/linux/man-pages/man8/ip-xfrm.8.html)
- [Cisco IOS IPsec Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_conn_ike2vpn/configuration/xe-16/sec-flex-vpn-xe-16-book.html)
- [Juniper IPsec VPN Configuration Guide](https://www.juniper.net/documentation/us/en/software/junos/vpn-ipsec/topics/topic-map/security-ipsec-vpn-overview.html)
