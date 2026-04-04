# SIP (Session Initiation Protocol)

Application-layer signaling protocol (RFC 3261) for creating, modifying, and terminating multimedia sessions over IP networks. SIP handles call setup, registration, and presence — the actual media flows over RTP/SRTP negotiated via SDP.

---

## Architecture

### Core Components

- **UAC (User Agent Client):** Initiates SIP requests (the caller)
- **UAS (User Agent Server):** Responds to SIP requests (the callee)
- **Proxy Server:** Routes requests between UAs; stateful or stateless
- **Registrar:** Accepts REGISTER requests and updates the location service
- **Redirect Server:** Returns 3xx responses directing UAC to alternate URIs
- **B2BUA:** Acts as UAS to one side, UAC to the other; full signaling control

### SIP URIs and Transport

```bash
# SIP URI format: sip:user@domain[:port][;parameters]
sip:alice@atlanta.example.com
sip:alice@atlanta.example.com:5060;transport=tcp
sips:bob@biloxi.example.com          # SIP over TLS
tel:+14155551234                      # Tel URI for phone numbers

# Transports: UDP (5060 default), TCP (5060), TLS (5061), WebSocket (RFC 7118)
ss -tlnp | grep -E '506[01]'
```

## Request Methods

### Core and Extension Methods

```bash
# Core (RFC 3261)
# INVITE  — initiate session    ACK     — confirm INVITE response
# BYE     — terminate session   CANCEL  — cancel pending INVITE
# REGISTER — bind contact URI   OPTIONS — query capabilities

# Extensions
# SUBSCRIBE/NOTIFY (RFC 6665) — event notifications
# REFER (RFC 3515)            — call transfer
# MESSAGE (RFC 3428)          — instant messaging
# INFO (RFC 6086)             — mid-dialog application info
# UPDATE (RFC 3311)           — modify session pre-answer
# PRACK (RFC 3262)            — reliable provisional responses
# PUBLISH (RFC 3903)          — publish event state
```

## Response Codes

### Status Code Classes

```bash
# 1xx Provisional
100 Trying          180 Ringing         183 Session Progress

# 2xx Success
200 OK              202 Accepted

# 3xx Redirection
301 Moved Permanently   302 Moved Temporarily

# 4xx Client Error
400 Bad Request     401 Unauthorized    403 Forbidden
404 Not Found       407 Proxy Auth Req  408 Request Timeout
480 Temp Unavail    486 Busy Here       487 Request Terminated
488 Not Acceptable

# 5xx Server Error
500 Internal Error  503 Service Unavail

# 6xx Global Failure
600 Busy Everywhere 603 Decline
```

## SDP Integration

### Offer/Answer Model (RFC 3264)

```bash
# SDP (RFC 4566) carried in INVITE body / 200 OK body
# Content-Type: application/sdp
#
# v=0
# o=alice 2890844526 2890844526 IN IP4 192.168.1.100
# s=SIP Call
# c=IN IP4 192.168.1.100
# t=0 0
# m=audio 49170 RTP/AVP 0 8 101
# a=rtpmap:0 PCMU/8000           # G.711 u-law 64kbps
# a=rtpmap:8 PCMA/8000           # G.711 A-law 64kbps
# a=rtpmap:101 telephone-event/8000  # DTMF (RFC 4733)
# a=fmtp:101 0-16
# a=sendrecv
```

## Call Setup and Teardown

### Basic INVITE Flow

```bash
# Alice (UAC) -> Proxy -> Bob (UAS)
# Alice           Proxy           Bob
#   |--- INVITE ---->|               |
#   |<-- 100 Trying -|               |
#   |                 |--- INVITE --->|
#   |                 |<-- 180 Ring --|
#   |<-- 180 Ringing -|               |
#   |                 |<-- 200 OK ----|
#   |<-- 200 OK ------|               |
#   |--- ACK -------->|--- ACK ------>|
#   |<============ RTP Media ========>|
#   |--- BYE -------->|--- BYE ------>|
#   |<-- 200 OK ------|<-- 200 OK ----|
```

### Registration and Authentication

```bash
# REGISTER binds AOR to Contact address
# First attempt: 401 Unauthorized (with WWW-Authenticate challenge)
# Second attempt: Authorization header with digest credentials
# Response: 200 OK (with Contact/Expires headers)
# Re-register before expiry (default 3600s), de-register with Expires: 0

# Digest auth (RFC 2617):
# HA1 = MD5(username:realm:password)
# HA2 = MD5(method:uri)
# response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
```

## NAT Traversal

### STUN/TURN/ICE and Media Proxy

```bash
# NAT is the #1 cause of one-way audio in SIP
# Problem: SDP contains private IPs unreachable from outside

# Kamailio NAT detection and RTPEngine integration
loadmodule "nathelper.so"
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:2223")

route {
    if (nat_uac_test("19")) {
        fix_nated_contact();
        force_rport();
        rtpengine_manage("replace-origin replace-session-connection ICE=remove");
    }
}

# RTPEngine — high-performance kernel-mode media proxy
sudo apt install ngcp-rtpengine
sudo systemctl start ngcp-rtpengine
```

## Server Configuration

### Asterisk PBX (PJSIP)

```bash
# /etc/asterisk/pjsip.conf
[transport-udp]
type=transport
protocol=udp
bind=0.0.0.0:5060

[alice]
type=endpoint
context=internal
disallow=all
allow=ulaw,alaw,g722
auth=alice-auth
aors=alice

[alice-auth]
type=auth
auth_type=userpass
username=alice
password=secretpass123

[alice]
type=aor
max_contacts=3
```

### Kamailio Proxy with Load Balancing

```bash
# kamailio.cfg — registrar + dispatcher
loadmodule "registrar.so"
loadmodule "usrloc.so"
loadmodule "auth_db.so"
loadmodule "dispatcher.so"

modparam("registrar", "default_expires", 3600)
modparam("dispatcher", "ds_ping_interval", 10)
modparam("dispatcher", "ds_probing_mode", 1)

route {
    if (is_method("REGISTER")) {
        if (!auth_check("example.com", "subscriber", "1")) {
            auth_challenge("example.com", "1");
            exit;
        }
        save("location");
        exit;
    }
    # Route to media server farm
    if (!ds_select_dst("1", "4")) {  # round-robin
        sl_send_reply("502", "No Media Server");
        exit;
    }
    t_relay();
}
```

## Debugging

### sngrep and tshark

```bash
# sngrep — interactive SIP flow viewer
sngrep -d eth0
sngrep -d any port 5060
sngrep -O /tmp/sip.pcap          # save capture
sngrep -I /tmp/capture.pcap      # read pcap

# tshark — SIP field extraction
tshark -i eth0 -f "port 5060" -T fields \
    -e sip.Method -e sip.Status-Code -e sip.From -e sip.To

# SIPp — load testing
sipp -sn uac 192.168.1.100:5060 -s 1001 -r 10 -l 100  # 10 cps, 100 max
sipp -sn uas -p 5060                                      # answer calls

# sipsak — quick SIP OPTIONS probe
sipsak -s sip:alice@example.com -v
```

## Security

### TLS and Fail2ban

```bash
# SIP TLS certificate
openssl req -new -x509 -days 365 -nodes \
    -out /etc/ssl/sip-server.crt \
    -keyout /etc/ssl/sip-server.key \
    -subj "/CN=sip.example.com"

# Kamailio TLS
loadmodule "tls.so"
modparam("tls", "certificate", "/etc/ssl/sip-server.crt")
modparam("tls", "private_key", "/etc/ssl/sip-server.key")
modparam("tls", "tls_method", "TLSv1.2+")
listen=tls:0.0.0.0:5061

# Fail2ban for SIP brute force
# /etc/fail2ban/jail.d/kamailio.conf
# [kamailio]
# enabled=true  filter=kamailio  logpath=/var/log/kamailio/kamailio.log
# maxretry=5  findtime=300  bantime=3600
```

---

## Tips

- Always use SIP over TLS and SRTP in production; unencrypted SIP leaks credentials and call metadata on the wire.
- Use a media proxy (RTPEngine) for any deployment with NAT; STUN alone fails behind symmetric NATs.
- Set `max_contacts` to 2-3 on registrar to prevent registration flooding attacks.
- Use dispatcher or load balancer in front of media servers; never expose Asterisk/FreeSWITCH directly to the internet.
- Keep SDP codecs ordered by preference; the answerer picks the first matching codec from the offer.
- Enable PRACK for reliable provisional responses when early media or ringback tones must be guaranteed.
- Watch for SIP ALG on consumer routers — it rewrites headers incorrectly causing one-way audio; disable it everywhere.
- Test with SIPp before production to know registrar and proxy limits under concurrent call volume.
- After changing route-maps or ACLs, monitor with `kamcmd ul.dump` to catch silent registration failures.
- Set Timer B appropriately for failover; the 32-second default is often too long.

---

## See Also

- webrtc, tls, radius, dns

## References

- [RFC 3261 — SIP: Session Initiation Protocol](https://www.rfc-editor.org/rfc/rfc3261)
- [RFC 4566 — SDP: Session Description Protocol](https://www.rfc-editor.org/rfc/rfc4566)
- [RFC 3264 — An Offer/Answer Model with SDP](https://www.rfc-editor.org/rfc/rfc3264)
- [RFC 3262 — Reliability of Provisional Responses (PRACK)](https://www.rfc-editor.org/rfc/rfc3262)
- [RFC 3515 — The SIP REFER Method](https://www.rfc-editor.org/rfc/rfc3515)
- [RFC 7118 — SIP over WebSocket](https://www.rfc-editor.org/rfc/rfc7118)
- [RFC 8224 — Authenticated Identity Management in SIP](https://www.rfc-editor.org/rfc/rfc8224)
- [Kamailio SIP Server Documentation](https://www.kamailio.org/wikidocs/)
- [Asterisk PJSIP Configuration](https://docs.asterisk.org/Configuration/Channel-Drivers/SIP/Configuring-res_pjsip/)
- [FreeSWITCH Documentation](https://developer.signalwire.com/freeswitch/)
- [sngrep — SIP Messages Flow Viewer](https://github.com/irontec/sngrep)
