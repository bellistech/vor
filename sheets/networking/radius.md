# RADIUS (Remote Authentication Dial-In User Service)

> Centralize authentication, authorization, and accounting (AAA) for network access using FreeRADIUS and 802.1X.

## Concepts

### AAA Model

```
# Authentication — verify user identity (username/password, certificate, token)
# Authorization  — determine access level (VLAN, ACL, session timeout)
# Accounting     — log session data (start, stop, interim updates)

# NAS (Network Access Server) — the device requesting auth (switch, AP, VPN)
# RADIUS server — evaluates auth requests, returns Accept/Reject/Challenge
# Shared secret — symmetric key between NAS and RADIUS server (per-client)
# Ports: Authentication = UDP 1812, Accounting = UDP 1813
```

### 802.1X Components

```
# Supplicant    — client software requesting access (wpa_supplicant, Windows native)
# Authenticator — network device enforcing access (switch port, wireless AP)
# Auth server   — RADIUS server making the decision (FreeRADIUS)

# EAP methods:
#   EAP-TLS   — mutual TLS with client+server certificates (strongest)
#   PEAP      — server certificate + inner MSCHAPv2 (most common, Windows)
#   EAP-TTLS  — server certificate + inner PAP/CHAP/MSCHAPv2 (flexible)
```

## FreeRADIUS Configuration

### Main Config (radiusd.conf)

```conf
# /etc/freeradius/3.0/radiusd.conf (or /etc/raddb/radiusd.conf)
# Key sections:
#   listen { }     — bind addresses and ports
#   modules { }    — loaded modules (eap, ldap, sql, etc.)
#   policy { }     — custom policies
#   instantiate { } — module load order

# Enable sites via symlinks in sites-enabled/
# Main processing: sites-available/default (non-EAP)
# Inner tunnel:    sites-available/inner-tunnel (EAP inner auth)
```

### Client Config (clients.conf)

```conf
# /etc/freeradius/3.0/clients.conf
client switch-floor1 {
    ipaddr    = 10.1.1.10
    secret    = SwitchSecret123
    shortname = floor1-sw
    nastype   = other
}

client wireless-controllers {
    ipaddr    = 10.1.2.0/24          # CIDR range
    secret    = WLCSecret456
    shortname = wlc
}
```

### Local Users File

```conf
# /etc/freeradius/3.0/users (mods-config/files/authorize)

# Simple password auth
bob     Cleartext-Password := "bobpass123"
        Reply-Message = "Welcome, Bob"

# VLAN assignment based on group
alice   Cleartext-Password := "alicepass"
        Tunnel-Type = VLAN,
        Tunnel-Medium-Type = IEEE-802,
        Tunnel-Private-Group-Id = "100"

# Reject a user
mallory Auth-Type := Reject
        Reply-Message = "Account disabled"

# Default (catch-all) — must be last
DEFAULT Auth-Type = Reject
        Reply-Message = "Access denied"
```

### EAP Configuration

```conf
# /etc/freeradius/3.0/mods-available/eap
eap {
    default_eap_type = peap
    timer_expire = 60

    tls-config tls-common {
        private_key_file    = /etc/freeradius/3.0/certs/server.key
        certificate_file    = /etc/freeradius/3.0/certs/server.pem
        ca_file             = /etc/freeradius/3.0/certs/ca.pem
        dh_file             = /etc/freeradius/3.0/certs/dh
        ca_path             = ${cadir}
        cipher_list         = "HIGH"
        tls_min_version     = "1.2"
    }

    peap {
        default_eap_type = mschapv2
        copy_request_to_tunnel = yes
        use_tunneled_reply = yes
    }

    eap-tls {
        tls = tls-common
    }
}
```

### Certificate Setup

```bash
# FreeRADIUS includes a bootstrap CA script
cd /etc/freeradius/3.0/certs/

# Edit ca.cnf, server.cnf, client.cnf with your org details, then:
make ca.pem               # generate CA
make server.pem            # generate server certificate
make client.pem            # generate client certificate (for EAP-TLS)

# Or use your own PKI
openssl req -x509 -newkey rsa:2048 -keyout ca.key -out ca.pem -days 3650
openssl req -newkey rsa:2048 -keyout server.key -out server.csr
openssl x509 -req -in server.csr -CA ca.pem -CAkey ca.key \
    -CAcreateserial -out server.pem -days 730
```

## LDAP / Active Directory Integration

### LDAP Module Config

```conf
# /etc/freeradius/3.0/mods-available/ldap
ldap {
    server   = "ldap://dc.example.com"
    port     = 389
    identity = "CN=radius-bind,OU=Service,DC=example,DC=com"
    password = "LDAPBindPass"
    base_dn  = "DC=example,DC=com"

    user {
        base_dn   = "${..base_dn}"
        filter    = "(sAMAccountName=%{%{Stripped-User-Name}:-%{User-Name}})"
    }

    group {
        base_dn   = "${..base_dn}"
        filter    = "(objectClass=group)"
        membership_attribute = "memberOf"
    }
}

# Enable in sites-available/default:
#   authorize { ldap }
#   authenticate { Auth-Type LDAP { ldap } }
```

## Testing

### radtest

```bash
# Basic auth test
radtest bob bobpass123 localhost 0 testing123

# Test against specific NAS
radtest alice alicepass 10.1.1.1:1812 0 SwitchSecret123

# Output: Access-Accept (success) or Access-Reject (failure)
```

### eapol_test (802.1X Testing)

```bash
# Install: part of wpa_supplicant source (hostap)
# Test PEAP-MSCHAPv2
cat > eapol_test.conf <<EOF
network={
    ssid="test"
    key_mgmt=WPA-EAP
    eap=PEAP
    identity="bob"
    password="bobpass123"
    phase2="autheap=MSCHAPV2"
    ca_cert="/etc/freeradius/3.0/certs/ca.pem"
}
EOF

eapol_test -c eapol_test.conf -a 127.0.0.1 -s testing123
# Look for: SUCCESS in output
```

### Accounting Verification

```bash
# Check accounting log
cat /var/log/freeradius/radacct/*/detail-*

# Test accounting packet
radclient -x 127.0.0.1:1813 acct testing123 <<EOF
User-Name = "bob"
Acct-Status-Type = Start
Acct-Session-Id = "test-session-001"
NAS-IP-Address = 10.1.1.10
EOF
```

## Troubleshooting

### Debug Mode

```bash
# Stop the service and run in foreground with full debug
systemctl stop freeradius
freeradius -X                     # full debug output to stdout

# Key things to look for in -X output:
# - "Login OK" / "Login incorrect"
# - "Found Auth-Type = ..."
# - EAP state machine transitions
# - "rlm_ldap" or "rlm_sql" module errors
# - Certificate verification failures

# Test config syntax without starting
freeradius -C                     # check config and exit
```

## Tips

- Always run `freeradius -X` first when debugging; it shows the full request processing pipeline.
- Use `unlang` in virtual server sections for conditional logic (if/else, switch, update).
- Keep shared secrets long (16+ characters) and unique per NAS device.
- Enable `status_server` in the listen section for monitoring (`radclient` Status-Server).
- For high availability, run two RADIUS servers; most NAS devices support primary/secondary.
- Log accounting to SQL (`rlm_sql`) for reporting and session tracking at scale.

## References

- [RFC 2865 — Remote Authentication Dial In User Service (RADIUS)](https://www.rfc-editor.org/rfc/rfc2865)
- [RFC 2866 — RADIUS Accounting](https://www.rfc-editor.org/rfc/rfc2866)
- [RFC 3579 — RADIUS Support For Extensible Authentication Protocol (EAP)](https://www.rfc-editor.org/rfc/rfc3579)
- [RFC 5216 — The EAP-TLS Authentication Protocol](https://www.rfc-editor.org/rfc/rfc5216)
- [RFC 6614 — Transport Layer Security (TLS) Encryption for RADIUS](https://www.rfc-editor.org/rfc/rfc6614)
- [RFC 6613 — RADIUS over TCP](https://www.rfc-editor.org/rfc/rfc6613)
- [FreeRADIUS Official Documentation](https://freeradius.org/documentation/)
- [FreeRADIUS Community Wiki](https://wiki.freeradius.org/)
- [FreeRADIUS GitHub Repository](https://github.com/FreeRADIUS/freeradius-server)
- [IANA RADIUS Types Registry](https://www.iana.org/assignments/radius-types/radius-types.xhtml)
- [Cisco RADIUS Configuration Guide](https://www.cisco.com/c/en/us/td/docs/ios-xml/ios/sec_usr_aaa/configuration/xe-16/sec-usr-aaa-xe-16-book/sec-cfg-radius.html)
