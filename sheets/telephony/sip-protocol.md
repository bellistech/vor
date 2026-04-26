# SIP Protocol (RFC 3261)

Session Initiation Protocol — text-based application-layer signaling for creating, modifying, terminating sessions (voice, video, IM, presence) — RFC 3261 (June 2002, obsoletes 2543).

## Setup

SIP is the dominant signaling protocol for VoIP, video conferencing, presence, IM, and IMS. It does not transport media — that is RTP. SIP carries SDP in its body to negotiate media. SIP looks like HTTP (request/response, headers + body, methods, status codes) but it is stateful, supports UDP transport, and has dialog/transaction semantics that HTTP lacks.

```text
RFC 3261        Core SIP protocol (June 2002, obsoleted RFC 2543)
RFC 2543        Original SIP (March 1999, obsoleted)
Default port    5060 (sip:), 5061 (sips:/TLS)
Transports      UDP, TCP, TLS, SCTP, WS, WSS
Encoding        UTF-8 text, CRLF line endings, header: value
URI scheme      sip:, sips:, tel:
Carries         Application-layer signaling; media (RTP) negotiated via SDP body
Body            SDP (RFC 4566) for offer/answer (RFC 3264); also MESSAGE bodies, NOTIFY events
```

Quick install / try:

```bash
# Linux
sudo apt install sngrep sip-tester baresip linphone-cli pjsip-tools
# macOS
brew install sngrep pjproject baresip
# Watch SIP on the wire
sudo sngrep -d eth0 -k call-flow
# Capture to PCAP
sudo tshark -i any -w sip.pcap -f "port 5060 or port 5061"
# Decode
tshark -r sip.pcap -V -Y sip
```

Transport choice:

```text
UDP   default; 1472-byte MTU danger; fragmentation breaks SIP — switch to TCP if message > 1300 B (RFC 3261 §18.1.1)
TCP   used when UDP message would exceed 1300 B; long-lived connections preserved with keepalive
TLS   sip: → sips: hop-by-hop encryption; mutual TLS for trunk-to-trunk
SCTP  rare; RFC 4168; multi-homing for HA carrier deployments
WS    RFC 7118; SIP over WebSocket for browser clients (WebRTC)
WSS   WebSocket-Secure variant
```

## SIP URI

Identifies a user, phone, or service. Looks like email + parameters.

```text
sip:alice@atlanta.example.com                     basic AOR
sip:alice@atlanta.example.com:5060                explicit port
sip:alice@atlanta.example.com;transport=tcp       transport hint
sip:alice@10.0.0.5;transport=udp                  IP-only contact (registration)
sips:alice@atlanta.example.com                    TLS-protected (port 5061)
sip:+12025551234@example.com;user=phone           E.164 number embedded
sip:alice@atlanta.example.com?subject=Hi&priority=urgent  headers in URI (used in Refer-To, click-to-call)
sip:%2B12025551234@gw.example.com                 URL-encoded "+"
tel:+12025551234                                  RFC 3966 tel URI; no host part
tel:+12025551234;ext=2345                         tel URI with extension
sip:alice@atlanta.example.com;lr                  loose-router parameter (Record-Route)
sip:alice@atlanta.example.com;maddr=239.255.255.1;ttl=15  multicast address (rare)
sip:alice@atlanta.example.com;gr=urn:uuid:f81d4...        GRUU (RFC 5627)
```

URI parameter precedence: URI parameters override most things; header parameters (after `?`) become headers in the resulting request.

## Architecture

SIP entities, all of which can be combined in one box (e.g., Asterisk = registrar + B2BUA).

```text
UAC     User Agent Client     entity that issues a request (your phone calling)
UAS     User Agent Server     entity that responds (called phone)
UA      User Agent            anything that is both UAC and UAS (every endpoint)
Proxy   intermediate routing element; stateless or stateful
Registrar accepts REGISTER and writes location service binding
Redirect server  responds 3xx to redirect requests instead of forwarding
B2BUA   Back-to-Back UA — terminates one call, originates another (PBX, SBC); no transparent dialog
SBC     Session Border Controller; B2BUA at trust boundary; topology hiding, NAT, transcoding
Location service  DB used by registrar/proxy: AOR → Contact mapping
Forking proxy   sends one INVITE to many endpoints, picks first 200 OK
```

```text
                          +-------------+
                          |  Registrar  |
                          +-------------+
                                |
   +-------+   INVITE   +-----+ |  +-----+   INVITE   +-------+
   |  UAC  |----------->|Proxy|-+->|Proxy|----------->|  UAS  |
   |Alice  |<---------- |  A  |<- |  B  |<-----------| Bob   |
   +-------+   200 OK   +-----+    +-----+   200 OK  +-------+
                            \      /
                             \____/
                              RTP (direct, no proxy)
```

## Methods

Core methods (RFC 3261) plus extensions:

```text
REGISTER    bind Contact to AOR at registrar (RFC 3261)
INVITE      initiate session (RFC 3261)
ACK         confirm final response to INVITE (RFC 3261)
BYE         end session (RFC 3261)
CANCEL      cancel pending INVITE before final response (RFC 3261)
OPTIONS     query capabilities or keepalive (RFC 3261)
INFO        mid-call info (DTMF carrier-style, video FIR) (RFC 6086)
MESSAGE     instant message (RFC 3428)
NOTIFY      send event notification (RFC 6665, was 3265)
SUBSCRIBE   request event notification (RFC 6665)
REFER       ask UAS to issue request (call transfer) (RFC 3515)
UPDATE      modify session before final answer (RFC 3311)
PRACK       provisional ACK for reliable 1xx (RFC 3262)
PUBLISH     publish event state to ESC (RFC 3903)
```

### INVITE

Initiates a session. Carries SDP offer (most common) or empty (then 200 OK has offer). Establishes a dialog upon 2xx. Subject to the INVITE state machine and 3-way handshake (INVITE/200/ACK).

```text
INVITE sip:bob@biloxi.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKnashds8;rport
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 314159 INVITE
Contact: <sip:alice@pc33.atlanta.example.com>
Allow: INVITE, ACK, CANCEL, OPTIONS, BYE, REFER, NOTIFY, MESSAGE, SUBSCRIBE, INFO, PRACK, UPDATE
Supported: replaces, 100rel, timer
Session-Expires: 1800
Min-SE: 90
Content-Type: application/sdp
Content-Length: 142

v=0
o=alice 53655765 2353687637 IN IP4 pc33.atlanta.example.com
s=-
c=IN IP4 pc33.atlanta.example.com
t=0 0
m=audio 49172 RTP/AVP 0 8 101
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-15
a=sendrecv
```

### ACK

Confirms a final response to an INVITE. For 2xx, ACK is end-to-end and starts a new transaction (no response). For non-2xx, ACK is hop-by-hop and part of the same transaction.

```text
ACK sip:bob@192.0.2.4 SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKnashds9
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>;tag=a6c85cf
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 314159 ACK
Content-Length: 0
```

Note: branch differs from INVITE for 2xx ACK (new transaction); same branch for non-2xx ACK.

### BYE

Tears down an established session (after 2xx received and ACKed). Either party can send BYE. Ends the dialog.

```text
BYE sip:alice@pc33.atlanta.example.com SIP/2.0
Via: SIP/2.0/UDP 192.0.2.4:5060;branch=z9hG4bKnashds10
Max-Forwards: 70
From: Bob <sip:bob@biloxi.example.com>;tag=a6c85cf
To: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 231 BYE
Content-Length: 0
```

CSeq must be larger than any prior request from same UA in the dialog. To/From tags are NOT swapped from caller's perspective; they reflect the local/remote of the sender.

### CANCEL

Cancels a pending request (typically INVITE) that has not yet received a final response. Hop-by-hop. Has its own transaction.

```text
CANCEL sip:bob@biloxi.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKnashds8
Max-Forwards: 70
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
To: Bob <sip:bob@biloxi.example.com>
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 314159 CANCEL
Content-Length: 0
```

CANCEL has same Via branch and CSeq number as the INVITE it cancels (only method differs). Race: if 200 arrives, send BYE instead.

### OPTIONS

Queries capabilities. Often used as keepalive heartbeat to test availability of an endpoint or trunk. Can be sent in or out of dialog.

```text
OPTIONS sip:bob@biloxi.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKopt1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>
From: Alice <sip:alice@atlanta.example.com>;tag=1928301775
Call-ID: ka84b4c76e66711@pc33.atlanta.example.com
CSeq: 1 OPTIONS
Contact: <sip:alice@pc33.atlanta.example.com>
Accept: application/sdp
Content-Length: 0
```

200 OK response will list Allow:, Accept:, Supported:.

### REGISTER

Binds a Contact (current location) to an Address-of-Record at a registrar. Sent periodically (Expires).

```text
REGISTER sip:registrar.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKreg1;rport
Max-Forwards: 70
To: Alice <sip:alice@atlanta.example.com>
From: Alice <sip:alice@atlanta.example.com>;tag=456248
Call-ID: 843817637684230@998sdasdh09
CSeq: 1826 REGISTER
Contact: <sip:alice@pc33.atlanta.example.com>;expires=3600
Expires: 3600
User-Agent: Linphone/5.2.0
Content-Length: 0
```

```text
Contact: *                          unregister all bindings (with Expires: 0)
Contact: <sip:...>;expires=0        unregister specific binding
Contact: <sip:...>;q=0.5            preference if multiple bindings
Contact: <sip:...>;+sip.instance="<urn:uuid:...>"  Outbound, distinguishes endpoint
Contact: <sip:...>;reg-id=1         Outbound multiple flow registration
```

### NOTIFY

Sends event notification within a SUBSCRIBE-established subscription. Always inside dialog (created by SUBSCRIBE or REFER).

```text
NOTIFY sip:alice@pc33.atlanta.example.com SIP/2.0
Via: SIP/2.0/UDP es.example.com:5060;branch=z9hG4bKnotify1
Max-Forwards: 70
To: Alice <sip:alice@atlanta.example.com>;tag=1928301774
From: Presence <sip:presence@es.example.com>;tag=8473
Call-ID: presub-12@pc33.atlanta.example.com
CSeq: 1 NOTIFY
Contact: <sip:presence@es.example.com>
Event: presence
Subscription-State: active;expires=3500
Content-Type: application/pidf+xml
Content-Length: ...

<?xml version="1.0" encoding="UTF-8"?>
<presence xmlns="urn:ietf:params:xml:ns:pidf" entity="sip:bob@biloxi.example.com">
  <tuple id="t1"><status><basic>open</basic></status></tuple>
</presence>
```

### SUBSCRIBE

Establishes a subscription to an event package (presence, dialog, message-summary, refer, etc.). Refresh by re-SUBSCRIBE; expires=0 to terminate.

```text
SUBSCRIBE sip:bob@biloxi.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKsub1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: presub-12@pc33.atlanta.example.com
CSeq: 1 SUBSCRIBE
Event: presence
Expires: 3600
Accept: application/pidf+xml
Contact: <sip:alice@pc33.atlanta.example.com>
Content-Length: 0
```

Common Event packages: `presence` (3856), `dialog` (4235), `message-summary` (3842 — voicemail), `refer` (3515), `reg` (3680).

### MESSAGE

Out-of-dialog instant message. Carries text/plain or other payload in body.

```text
MESSAGE sip:bob@biloxi.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKmsg1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>
From: Alice <sip:alice@atlanta.example.com>;tag=msgtag-1
Call-ID: imc12345@pc33.atlanta.example.com
CSeq: 1 MESSAGE
Content-Type: text/plain
Content-Length: 19

Hello — meeting in 5
```

### REFER

Asks a remote UA to issue a request to a third party. Used for call transfer (blind, attended). Implicitly subscribes to the `refer` event package; UAS sends NOTIFY to report progress.

```text
REFER sip:bob@biloxi.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKref1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>;tag=a6c85cf
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 230 REFER
Refer-To: <sip:carol@chicago.example.com>
Referred-By: <sip:alice@atlanta.example.com>
Contact: <sip:alice@pc33.atlanta.example.com>
Content-Length: 0
```

Attended transfer adds Replaces header in the Refer-To URI:

```text
Refer-To: <sip:carol@chicago.example.com?Replaces=callid%3Btag1%3Btag2>
```

### INFO

Mid-call signaling for application-specific data. Common uses: out-of-band DTMF (rare; prefer RFC 4733), video Fast-Update Request (FIR/PLI), call-center events.

```text
INFO sip:bob@192.0.2.4 SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKinfo1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>;tag=a6c85cf
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 232 INFO
Content-Type: application/dtmf-relay
Content-Length: 23

Signal=5
Duration=160
```

### PUBLISH

Publishes event state to an Event State Compositor. Dialog-less; uses SIP-If-Match/ETag for atomic update of state.

```text
PUBLISH sip:presence@es.example.com SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKpub1
Max-Forwards: 70
To: <sip:alice@atlanta.example.com>
From: <sip:alice@atlanta.example.com>;tag=pub-1
Call-ID: pub-001@pc33.atlanta.example.com
CSeq: 1 PUBLISH
Event: presence
Expires: 3600
Content-Type: application/pidf+xml
Content-Length: ...

<?xml version="1.0"?>
<presence xmlns="urn:ietf:params:xml:ns:pidf" entity="sip:alice@atlanta.example.com">
  <tuple id="t1"><status><basic>open</basic></status></tuple>
</presence>
```

Refresh: include `SIP-If-Match: <etag>`; remove: Expires: 0 + SIP-If-Match.

### UPDATE

Modifies session parameters (codec, hold, address) before the initial INVITE has been answered. Allowed in early dialog. Useful for early-media re-INVITE-like behavior without sending re-INVITE.

```text
UPDATE sip:bob@192.0.2.4 SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKupd1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>;tag=a6c85cf
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 315 UPDATE
Contact: <sip:alice@pc33.atlanta.example.com>
Content-Type: application/sdp
Content-Length: ...

v=0
o=alice 53655765 2353687638 IN IP4 pc33.atlanta.example.com
s=-
c=IN IP4 pc33.atlanta.example.com
t=0 0
m=audio 49172 RTP/AVP 0 101
a=rtpmap:0 PCMU/8000
a=sendrecv
```

### PRACK

Provisional Acknowledgement (RFC 3262) for reliable 1xx responses. Triggered by 1xx with `Require: 100rel` and an `RSeq:` header.

```text
PRACK sip:bob@192.0.2.4 SIP/2.0
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKprack1
Max-Forwards: 70
To: Bob <sip:bob@biloxi.example.com>;tag=a6c85cf
From: Alice <sip:alice@atlanta.example.com>;tag=1928301774
Call-ID: a84b4c76e66710@pc33.atlanta.example.com
CSeq: 314160 PRACK
RAck: 1 314159 INVITE
Content-Length: 0
```

`RAck:` = RSeq of the 1xx, CSeq num, CSeq method.

## Response Codes

Status codes are 3-digit integers grouped by leading digit. 1xx provisional, 2xx success, 3xx redirect, 4xx request failure, 5xx server failure, 6xx global failure.

### 1xx — Provisional

```text
100 Trying              proxy/UAS received the request; suppress retransmissions
180 Ringing             callee's UA is alerting the user
181 Call Is Being Forwarded   call diverted to another destination
182 Queued              callee unavailable; call queued (with optional reason)
183 Session Progress    UAS providing early media; often carries SDP for ringback
199 Early Dialog Term.  RFC 6228; one early dialog of a forked call has terminated
```

180 vs 183: 180 = local ringback (UAC plays tone); 183 = remote sends actual audio (PSTN announcements, queue music).

### 2xx — Success

```text
200 OK                  request succeeded; for INVITE, must be ACKed
202 Accepted            request accepted for processing (REFER, SUBSCRIBE; obsoleted by RFC 6665 for SUBSCRIBE)
204 No Notification     RFC 5839 (rare)
```

### 3xx — Redirection

```text
300 Multiple Choices    several alternatives in Contact header
301 Moved Permanently   user moved permanently; follow Contact
302 Moved Temporarily   user moved temporarily; follow Contact (call forward)
305 Use Proxy           must route through specified proxy
380 Alternative Service literal alternative (often "voicemail")
```

### 4xx — Request Failure

```text
400 Bad Request                 malformed request (missing header, bad URI)
401 Unauthorized                authentication required (challenge from UAS, esp. registrar)
402 Payment Required            reserved (rarely used)
403 Forbidden                   request understood but refused (no auth retry)
404 Not Found                   user not at this domain
405 Method Not Allowed          server does not support method (Allow: header lists supported)
406 Not Acceptable              cannot meet Accept: requirements
407 Proxy Authentication Required proxy challenge (Proxy-Authenticate)
408 Request Timeout             no answer in reasonable time (timer C/B fired)
410 Gone                        user existed but is permanently gone
412 Conditional Request Failed  precondition failed (PUBLISH SIP-If-Match)
413 Request Entity Too Large    body too large
414 Request-URI Too Long
415 Unsupported Media Type      cannot decode body type (e.g., SDP m= unrecognized)
416 Unsupported URI Scheme      URI scheme not supported (sip vs tel)
417 Unknown Resource-Priority   RFC 4412
420 Bad Extension               option-tag in Require: not supported (Unsupported: header lists)
421 Extension Required          UAS needs an extension that UAC didn't list in Supported:
422 Session Interval Too Small  Session-Expires below Min-SE (RFC 4028)
423 Interval Too Brief          REGISTER expires too short (Min-Expires: header)
424 Bad Location Information    RFC 6442
428 Use Identity Header         RFC 4474; Identity required
429 Provide Referrer Identity   RFC 3892
430 Flow Failed                 RFC 5626; outbound flow failed
433 Anonymity Disallowed        RFC 5079
436 Bad Identity-Info           RFC 4474
437 Unsupported Certificate     RFC 4474
438 Invalid Identity Header
439 First Hop Lacks Outbound    RFC 5626
440 Max-Breadth Exceeded        RFC 5393
469 Bad Info Package            RFC 6086
470 Consent Needed              RFC 5360
480 Temporarily Unavailable     user not online; try later
481 Call/Transaction Does Not Exist   no matching dialog/transaction (sent BYE for closed dialog)
482 Loop Detected               via list shows loop
483 Too Many Hops               Max-Forwards reached 0
484 Address Incomplete          dial more digits (overlap dialing)
485 Ambiguous                   request URI matched multiple users
486 Busy Here                   user busy at this UA (other UAs may answer)
487 Request Terminated          request terminated by CANCEL or BYE before completion
488 Not Acceptable Here         SDP cannot be supported (codec mismatch)
489 Bad Event                   Event: not understood (RFC 6665)
491 Request Pending             concurrent re-INVITE; glare resolution
493 Undecipherable              S/MIME body cannot be decrypted
494 Security Agreement Required RFC 3329
```

### 5xx — Server Failure

```text
500 Server Internal Error       generic server bug
501 Not Implemented             method not implemented
502 Bad Gateway                 downstream gateway returned invalid response
503 Service Unavailable         overload; Retry-After header tells when (often used for graceful shutdown of trunk)
504 Server Time-out             upstream did not respond in time
505 Version Not Supported       SIP version unrecognized
513 Message Too Large           UDP message too big; UAC must switch to TCP
555 Push Notification Service Not Supported  RFC 8599
580 Precondition Failure        RFC 3312 preconditions could not be met
```

### 6xx — Global Failure

```text
600 Busy Everywhere             busy at all UAs; do not retry
603 Decline                     user declined; do not try alternatives (vs 486)
604 Does Not Exist Anywhere     no instance of user anywhere
606 Not Acceptable              global media negotiation failure
607 Unwanted                    RFC 8197; recipient does not want this call (spam-block)
608 Rejected                    RFC 8688; rejected by intermediary on behalf of user
```

## Headers

### General (request and response)

```text
Via                Records each hop, used for response routing back. Contains branch.
From               Originator AOR + tag (dialog ID component). Stays same in dialog.
To                 Target AOR + tag (added by UAS in dialog-establishing response).
Call-ID            Globally unique dialog identifier; same across requests in dialog.
CSeq               Number + method; orders requests within dialog.
Contact            Direct URI for follow-up requests; bypasses original routing.
Max-Forwards       Hop-count, default 70; decremented per proxy; 0 → 483.
Allow              List of methods supported by UA (in INVITE, 200 OK, OPTIONS).
Supported          Option-tags UA supports (e.g., "100rel,replaces,timer").
Require            Option-tags peer MUST support; if not, 420 Bad Extension.
Unsupported        Option-tags peer doesn't support (in 420 response).
Accept             Body MIME types accepted in response.
Accept-Encoding    Encodings accepted (rare).
Accept-Language    Languages accepted (Reason text).
User-Agent         UA software identifier (parallels HTTP).
Server             UAS software identifier in response.
Date               Wall-clock from sender (used by NOTIFY).
Organization       Free-text org name.
Subject            Free-text subject (rendered by UA).
Priority           normal / urgent / non-urgent / emergency.
Timestamp          Round-trip-time measurement.
```

### Request-only

```text
Authorization       Credentials for UAS challenge response (401).
Proxy-Authorization Credentials for proxy challenge response (407).
Route               Pre-loaded routing list; built from Record-Route in reverse.
Refer-To            Target URI for REFER.
Referred-By         Identity of referrer (RFC 3892).
Replaces            Dialog to replace (attended transfer; RFC 3891).
Reply-To            Suggested reply address.
In-Reply-To         Call-IDs being replied to.
Hide                (deprecated, RFC 2543) topology hiding.
RAck                PRACK acknowledgement (RFC 3262).
Session-Expires     RFC 4028 timer.
Min-SE              Minimum acceptable session-expires.
Min-Expires         (response only) shortest acceptable Expires from registrar.
SIP-If-Match        ETag for PUBLISH (RFC 3903).
Subscription-State  active/pending/terminated (NOTIFY only).
Event               Event package (SUBSCRIBE/NOTIFY/PUBLISH/REFER).
```

### Response-only

```text
WWW-Authenticate    UAS challenge (401).
Proxy-Authenticate  Proxy challenge (407).
Authentication-Info Server's auth response after success (RFC 7616).
Record-Route        Forces in-dialog requests to traverse this proxy.
Reason              RFC 3326; cause for response (CANCEL/BYE).
Retry-After         Seconds before retry (503, 480).
Warning             Free-text warning (e.g., "399 host:port \"text\"").
Error-Info          URI to additional error info / pre-recorded message.
SIP-ETag            Returned by ESC after PUBLISH (RFC 3903).
Min-Expires         Sent in 423 to indicate registrar minimum.
RSeq                Provisional response sequence (RFC 3262).
```

### Body

```text
Content-Type        application/sdp / message/sipfrag / application/dtmf-relay / text/plain / application/pidf+xml / multipart/mixed
Content-Length      Octet length of body (including final CRLF only if present); MUST for TCP.
Content-Encoding    gzip etc. (rare; Linphone does not use).
Content-Language    en, es, etc.
Content-Disposition session (default for SDP) / render / icon / alert / signal / by-reference / info-package
MIME-Version        1.0 (rarely sent).
```

### Privacy / Identity

```text
P-Asserted-Identity   RFC 3325; trusted-domain asserted identity (real caller behind anonymous From).
P-Preferred-Identity  RFC 3325; UA hints which identity to assert.
Privacy               RFC 3323; "id"/"header"/"session"/"user"/"none"/"critical".
Remote-Party-ID       (deprecated) pre-3325 vendor header (still common in SBCs).
P-Charging-Vector     IMS charging info (RFC 3455).
P-Asserted-Service    IMS (RFC 6050).
P-Early-Media         IMS (RFC 5009): inactive/sendonly/recvonly/sendrecv/gated.
History-Info          RFC 7044; tracks redirect history.
Diversion             vendor / RFC 5806 (informational); call-forward provenance.
```

## Deep dive: Via

`Via:` is added by every hop that processes a request. The response retraces the Via list in reverse — each hop strips its own Via and forwards using the next.

```text
Via: SIP/2.0/UDP pc33.atlanta.example.com:5060;branch=z9hG4bKnashds8;rport=37621;received=198.51.100.1;ttl=16;maddr=...
```

Parameters:

```text
branch     Transaction identifier. MUST start with z9hG4bK (magic cookie indicating RFC 3261-style branch).
rport      RFC 3581. UAC adds bare "rport" to ask UAS to record source port; UAS fills "rport=NNN".
received   Source IP (per UAS). Set by next hop if differs from sent-by.
ttl        Multicast TTL.
maddr      Multicast address.
sent-by    host:port the sender used (after the "/UDP ").
```

The branch on retransmissions of the same request MUST be identical (so the receiver matches it to the existing transaction). For ACK to a non-2xx, branch matches the original INVITE. For ACK to a 2xx, branch is new.

## Deep dive: From / To / tags

The 3-tuple `Call-ID + From-tag + To-tag` is the dialog identifier. In the request, From-tag is set by UAC; To has no tag until the UAS adds one in a non-100 response.

```text
INVITE  →  From: ...;tag=alice123    To: ...                          (no to-tag)
180     ←  From: ...;tag=alice123    To: ...;tag=bob456              (UAS added tag)
200     ←  From: ...;tag=alice123    To: ...;tag=bob456              (same tag)
ACK     →  From: ...;tag=alice123    To: ...;tag=bob456              (echoed)
```

Forking creates multiple early dialogs sharing Call-ID and From-tag but differing in To-tag. Whichever finally answers wins.

## Deep dive: Call-ID

Globally unique string per dialog. UA's responsibility to ensure uniqueness; typical format `random@host` or `random@ip`. Same Call-ID for all requests in the dialog plus same Call-ID for transactions like INVITE/CANCEL/ACK plus future BYE.

```text
Call-ID: f81d4fae-7dec-11d0-a765-00a0c91e6bf6@198.51.100.5
```

Re-REGISTER from same UA usually keeps Call-ID stable (so registrar treats it as refresh, not new flow). Some UAs use one Call-ID for all REGISTERs and another set for calls.

## Deep dive: CSeq

Monotonically increasing 32-bit unsigned integer + method name. Per-direction within a dialog. CANCEL uses the same number as the request it cancels; ACK to non-2xx uses INVITE number; ACK to 2xx is new transaction (still same number per RFC 3261 §17.1.1.3).

```text
CSeq: 314159 INVITE     → CSeq: 314159 CANCEL      → CSeq: 314159 ACK    (non-2xx)
CSeq: 314159 INVITE     → CSeq: 314160 BYE         → CSeq: 314161 INFO
```

A re-INVITE in dialog uses next CSeq number. Each direction tracks its own CSeq counter.

## Deep dive: Contact

`Contact:` carries the direct URI to reach the UA — bypasses the registrar/proxy used for the original request. Used by the peer for in-dialog requests (BYE, re-INVITE) and by registrar for AOR binding.

```text
Contact: "Alice" <sip:alice@10.0.0.5:5060;ob>;expires=600;+sip.instance="<urn:uuid:f81d4fae...>"
```

`ob` = Outbound (RFC 5626). `+sip.instance` = unique endpoint UUID. Without Outbound, UA behind NAT fills Contact with private IP and breaks return path — proxies must rewrite or use Record-Route + rport + received.

## Deep dive: Route / Record-Route

A proxy that wants to remain in the path of all in-dialog requests adds `Record-Route:` to the initial INVITE and its 200 OK. UAs save the list. Subsequent requests in the dialog include `Route:` — popped one entry per hop.

```text
Proxy A and B both Record-Route. UAC sees:
  Record-Route: <sip:proxy-b.example.com;lr>, <sip:proxy-a.example.com;lr>
UAC sends BYE with:
  Route: <sip:proxy-a.example.com;lr>, <sip:proxy-b.example.com;lr>
  BYE sip:bob@10.0.0.5 SIP/2.0
```

`;lr` = loose-router. Strict routing (pre-RFC 3261) put intermediate URI in Request-URI; do not use except for legacy interop.

## Deep dive: Authentication (Digest, RFC 7616)

SIP uses HTTP Digest Authentication. Hop-by-hop. Server returns 401/407 with WWW-Authenticate/Proxy-Authenticate, UAC retries with Authorization/Proxy-Authorization.

Challenge:

```text
SIP/2.0 401 Unauthorized
WWW-Authenticate: Digest realm="atlanta.example.com",
                         qop="auth,auth-int",
                         nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093",
                         opaque="5ccc069c403ebaf9f0171e9517f40e41",
                         algorithm=SHA-256,
                         stale=false
```

Response:

```text
Authorization: Digest username="alice",
                      realm="atlanta.example.com",
                      nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093",
                      uri="sip:bob@biloxi.example.com",
                      qop=auth,
                      nc=00000001,
                      cnonce="0a4f113b",
                      algorithm=SHA-256,
                      response="6629fae49393a05397450978507c4ef1",
                      opaque="5ccc069c403ebaf9f0171e9517f40e41"
```

Computation (qop=auth):

```text
HA1 = H( username ":" realm ":" password )
HA2 = H( method ":" uri )
response = H( HA1 ":" nonce ":" nc ":" cnonce ":" qop ":" HA2 )
H = MD5 (legacy) or SHA-256 / SHA-512-256 (RFC 7616 modern)
```

`qop=auth-int` extends HA2 to include H(body): `HA2 = H(method ":" uri ":" H(body))`.

`nc` is incremented on each request reusing same nonce. Server may rotate nonces (`stale=true` triggers reauth without password prompt).

## Transactions vs Dialogs

```text
Transaction   single request + all responses to it (one ladder rung)
              identified by:
                client transaction = branch + sent-by
                server transaction = branch + sent-by + method
              types: INVITE, non-INVITE
              short-lived (seconds to a minute via timers)

Dialog        peer-to-peer SIP relationship (the call as a whole)
              established by 2xx (or early dialog by 1xx with To-tag)
              identifier = Call-ID + local-tag + remote-tag
              long-lived (entire call duration)
              re-INVITE / UPDATE / BYE happen within
```

Dialog state lasts from INVITE 2xx through BYE 200. Multiple transactions per dialog. CANCEL has its own transaction that targets the INVITE transaction.

## INVITE Transaction State Machine

UAC side:

```text
            INVITE sent
                |
                v
+-----------+   1xx     +-------------+    2xx   +------------+
|  Calling  |---------> | Proceeding  |--------> | Terminated |
+-----------+           +-------------+          +------------+
     |  300-699                 |  300-699            ^
     |                          v                     |
     |                     +-----------+   ACK sent   |
     +-------------------> | Completed |--------------+
              ACK sent     +-----------+
                                Timer D (32s UDP / 0s TCP)
```

UAS side:

```text
+------------+   1xx    +-------------+
| Proceeding |--------> | Proceeding  |
+------------+          +-------------+
     |                       |
     | 300-699                | 2xx
     v                       v
+------------+         +-------------+
| Completed  |  ACK    | Accepted    |
| (300-699)  |-------> | Confirmed   |
+------------+         +-------------+
     | Timer H/I              | Timer L
     v                        v
+------------+         +-------------+
| Terminated |         | Terminated  |
+------------+         +-------------+
```

Timers:

```text
T1     500 ms        Round-trip-time estimate (default).
T2     4 s            Maximum non-INVITE retransmission interval.
T4     5 s            Maximum response retain time.
A      T1, doubles   INVITE request retransmit interval.
B      64*T1 = 32 s  INVITE transaction timeout (Calling → Terminated if no response).
C      > 3 min       Proxy INVITE timeout.
D      ≥ 32 s (UDP), 0 (TCP)   Wait time for response retransmissions in Completed state.
E      T1, doubles   non-INVITE request retransmit interval.
F      64*T1 = 32 s  non-INVITE transaction timeout.
G      T1, doubles   INVITE response retransmit (UAS).
H      64*T1 = 32 s  Wait for ACK after final response.
I      T4 (UDP) or 0 (TCP)   Wait in Confirmed state.
J      64*T1 (UDP) or 0 (TCP)   non-INVITE server transaction state.
K      T4 (UDP) or 0 (TCP)   non-INVITE client wait for response retransmissions.
L      64*T1               Wait for ACK in Accepted state (RFC 6026).
M      64*T1               Wait for retransmitted 2xx (RFC 6026).
```

## SDP (RFC 4566)

Session Description Protocol. Carried in SIP body to negotiate media.

Format:

```text
v=  protocol version (must be 0)
o=  origin: <user> <sess-id> <sess-version> <nettype> <addrtype> <unicast-address>
s=  session name (- for none)
i=  session info (optional)
u=  URI (optional)
e=  email (optional)
p=  phone (optional)
c=  connection info: <nettype> <addrtype> <connection-address>
b=  bandwidth: <bwtype>:<bandwidth>   e.g., AS:128, CT:512, TIAS:128000
t=  time: <start> <stop>             0 0 = unbounded
r=  repeat times
z=  time zone adjustments
k=  encryption key (deprecated; use crypto in m=)
a=  attributes (session-level or media-level)
m=  media: <media> <port> <proto> <fmt list>
```

Order: v, o, s, i, u, e, p, c, b, t, r, z, k, a, m, i, c, b, k, a (m+ block repeats).

Example (audio + video, BUNDLE, DTLS-SRTP, SDES):

```text
v=0
o=alice 53655765 2353687637 IN IP4 198.51.100.5
s=-
c=IN IP4 198.51.100.5
t=0 0
a=group:BUNDLE audio video
a=msid-semantic: WMS *
m=audio 49170 UDP/TLS/RTP/SAVPF 111 0 8 101
c=IN IP4 198.51.100.5
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10;useinbandfec=1
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-15
a=ptime:20
a=maxptime:120
a=sendrecv
a=mid:audio
a=ice-ufrag:F7gI
a=ice-pwd:x9cml/YzichV2+XlhiMu8g
a=ice-options:trickle
a=fingerprint:sha-256 12:34:56:78:9A:...
a=setup:actpass
a=rtcp:49171 IN IP4 198.51.100.5
a=rtcp-mux
a=rtcp-fb:111 transport-cc
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=ssrc:1110000 cname:slfYZ
a=msid:streamid trackid
m=video 49180 UDP/TLS/RTP/SAVPF 96 97
c=IN IP4 198.51.100.5
a=rtpmap:96 VP8/90000
a=rtpmap:97 H264/90000
a=fmtp:97 profile-level-id=42e01f;packetization-mode=1
a=rtcp-fb:96 nack
a=rtcp-fb:96 nack pli
a=rtcp-fb:96 ccm fir
a=rtcp-fb:96 transport-cc
a=rtcp-fb:96 goog-remb
a=mid:video
a=sendrecv
```

### m-line

```text
m=<media> <port> <proto> <fmt list>
   media:   audio | video | text | application | message
   port:    UDP port (or 0 to disable this media)
   proto:   RTP/AVP            (RTP, no security)
            RTP/SAVP           (SRTP, SDES)
            UDP/TLS/RTP/SAVP   (DTLS-SRTP)
            UDP/TLS/RTP/SAVPF  (DTLS-SRTP + feedback profile, WebRTC)
            TCP/MSRP           (instant message)
            DTLS/SCTP          (WebRTC data channels)
   fmt:     RTP payload type numbers (mapped via a=rtpmap)
```

### a-line attributes

Common attribute lines:

```text
a=rtpmap:<pt> <encoding>/<clock>[/<channels>]   maps payload type to codec
a=fmtp:<pt> key=value;key=value                 codec-specific params
a=ptime:<ms>                                    target packetization time (default 20 for audio)
a=maxptime:<ms>                                 max packet time
a=sendrecv | sendonly | recvonly | inactive     direction
a=rtcp:<port> [IN IP4 <addr>]                   override RTCP port
a=rtcp-mux                                      RTP and RTCP share port
a=rtcp-fb:<pt> nack | nack pli | ccm fir | goog-remb | transport-cc
a=ice-ufrag:<frag>                              ICE username fragment (RFC 5245/8445)
a=ice-pwd:<pwd>                                 ICE password
a=ice-options:trickle                           trickle-ICE supported
a=candidate:1 1 UDP 2130706431 198.51.100.5 49170 typ host       ICE candidate
a=fingerprint:sha-256 12:34:...                 DTLS cert fingerprint
a=setup:actpass | active | passive | holdconn   DTLS role (RFC 5763)
a=mid:<id>                                      media identifier (BUNDLE)
a=msid:<stream> <track>                         WebRTC stream/track ID
a=ssrc:<ssrc> cname:<id>                        RTP source synchronization
a=ssrc-group:FID <ssrc1> <ssrc2>                FEC/RTX grouping
a=group:BUNDLE audio video data                 RFC 8843 BUNDLE
a=extmap:<id> <uri>                             RFC 5285 RTP header extension
a=crypto:<tag> AES_CM_128_HMAC_SHA1_80 inline:<key>   SDES SRTP key (RFC 4568)
a=tls-id:<id>                                   TLS connection identifier
a=hold | unhold                                 (legacy hold; use sendonly/inactive)
a=quality:<0-10>                                (rare)
a=charset:UTF-8                                 (text/MSRP)
```

### SDP offer/answer (RFC 3264)

```text
Offer    Sender lists media streams + codecs in order of preference
Answer   Receiver MUST keep same number of m-lines, in same order
         Reject by setting port to 0 (a=inactive optional)
         For each remaining stream: pick subset of codecs, possibly reverse direction
Direction transformations:
   sendrecv → sendrecv | sendonly | recvonly | inactive
   sendonly → recvonly | inactive
   recvonly → sendonly | inactive
   inactive → inactive
```

Offer in INVITE → Answer in 200 OK. Empty INVITE → Offer in 200 OK, Answer in ACK. Re-INVITE for hold: send same stream with `sendonly`. Peer answers `recvonly`. Resume: re-INVITE with `sendrecv`.

### Early media (183 Session Progress with SDP)

UAS may send 183 with SDP before 200 OK; UAC opens RTP path for in-band ringback / IVR / "press 1". UAS direction often `sendonly` or `sendrecv`. Final 200 OK echoes the SDP (often unchanged).

```text
SIP/2.0 183 Session Progress
Via: ...
To: ...;tag=earlytag
From: ...;tag=alice123
CSeq: 1 INVITE
Contact: <sip:bob@biloxi>
Content-Type: application/sdp
Content-Length: ...

v=0
o=bob 53655766 2353687638 IN IP4 192.0.2.4
s=-
c=IN IP4 192.0.2.4
t=0 0
m=audio 49180 RTP/AVP 0
a=rtpmap:0 PCMU/8000
a=sendonly
```

P-Early-Media (RFC 5009) coordinates which streams are gated open.

## NAT Traversal

UAs behind NAT have private IPs in Via, Contact, SDP c=, and m= port. Without help, response goes to the wrong place and RTP never reaches the endpoint.

### rport (RFC 3581)

UAC adds bare `rport` to Via. Receiving proxy/UAS replaces with the source port and adds `received` IP. Response is sent to that port/IP, traversing the NAT pinhole. MUST use symmetric signaling (response from same port as request landed).

```text
Outgoing:  Via: SIP/2.0/UDP 10.0.0.5:5060;branch=...;rport
Recorded:  Via: SIP/2.0/UDP 10.0.0.5:5060;branch=...;rport=37621;received=198.51.100.1
Reply to:  198.51.100.1:37621 (NAT public side)
```

Asterisk: `nat=force_rport,comedia` (chan_sip) or `rtp_symmetric=yes, force_rport=yes, rewrite_contact=yes` (chan_pjsip).

### STUN / TURN / ICE

```text
STUN  RFC 8489. UA queries STUN server for its public reflexive address; uses it in SDP.
TURN  RFC 8656. Allocates a relay address on a TURN server; media flows through relay.
ICE   RFC 8445. UA gathers candidates (host, srflx via STUN, relay via TURN), sends list in SDP, performs connectivity checks, picks best pair.
```

ICE attributes in SDP: `a=ice-ufrag`, `a=ice-pwd`, `a=candidate`, `a=ice-options:trickle`, `a=end-of-candidates`. WebRTC requires ICE; classic SIP rarely uses it (relies on SBC + rport).

### SIP-ALG gotchas

Many SOHO routers run SIP application-layer gateway code that rewrites SDP and Via on the fly. Often broken (truncates messages, mishandles TLS, rewrites only IPv4, misses ports above range). Disable when possible.

```text
DD-WRT     uci set firewall.@helper[0].name=sip; ...
Cisco IOS  no ip nat service sip
Asus       LAN → NAT Passthrough → SIP Passthrough: Disable
```

### force_rport / comedia / direct-media

```text
force_rport       Always send response to source IP/port (ignore Via sent-by).
comedia           Symmetric RTP — observe RTP source IP/port and send back there.
direct-media=no   Force B2BUA/proxy to relay RTP; required for many NAT scenarios.
```

## SIPS / TLS

```text
sips: URI       requires TLS hop-by-hop end-to-end (per RFC 3261; relaxed in practice).
Default port    5061
DNS discovery   _sips._tcp.<domain> SRV records.
Mutual TLS      both UAS and UAC present certificates (SBC trunk).
SNI             Client signals target hostname so server picks cert (RFC 6066).
ALPN            "sip" registered (RFC 7301).
```

OpenSSL test:

```bash
openssl s_client -connect sip.example.com:5061 -starttls sip   # rare; SIP usually not STARTTLS
openssl s_client -connect sip.example.com:5061 -servername sip.example.com  # plain TLS
```

Cipher policy: TLS 1.2+, prefer ECDHE-RSA-AES128-GCM-SHA256, ECDHE-ECDSA-AES256-GCM-SHA384.

## SRV / NAPTR Discovery (RFC 3263)

UA resolving `sip:alice@example.com` performs:

```text
1. NAPTR query for example.com → lists supported transports (SIP+D2T = TCP, SIPS+D2T = TLS, SIP+D2U = UDP)
2. For chosen transport, SRV query e.g. _sip._tcp.example.com → host:port + priority/weight
3. A/AAAA on resolved host
```

Example records:

```text
example.com.  IN NAPTR 50 50 "s" "SIPS+D2T" "" _sips._tcp.example.com.
example.com.  IN NAPTR 90 50 "s" "SIP+D2T"  "" _sip._tcp.example.com.
example.com.  IN NAPTR 100 50 "s" "SIP+D2U" "" _sip._udp.example.com.
_sips._tcp.example.com. IN SRV 10 100 5061 sip1.example.com.
_sip._tcp.example.com.  IN SRV 10 100 5060 sip1.example.com.
_sip._udp.example.com.  IN SRV 10 100 5060 sip1.example.com.
sip1.example.com.       IN A   198.51.100.10
```

If no NAPTR, UA tries SRV directly per scheme; if no SRV, A record + default port.

## WebRTC + SIP

```text
RFC 7118       SIP-over-WebSocket (WS, WSS). URI hint: ;transport=ws.
JsSIP, SIP.js, sipML5    JavaScript SIP stacks for browser.
Asterisk       res_websocket / chan_pjsip type=transport bind=ws://0.0.0.0:8088
FreeSWITCH     mod_sofia, profile binds wss-binding=":7443"
```

Browser flow:

```text
1. wss://sip.example.com:7443 (TLS WebSocket).
2. Browser sends SIP REGISTER over WS frames.
3. SIP signaling carries SDP for WebRTC media (UDP/TLS/RTP/SAVPF, BUNDLE, ICE, DTLS-SRTP).
4. Server (Asterisk/FreeSWITCH) terminates WebRTC media, transcodes to G.711 for PSTN trunk if needed.
```

Outbound (RFC 5626) is mandatory for WS to map registration to flow.

## SIP Outbound (RFC 5626)

Solves the "two NATs" problem: UA registers from a flow (TCP/TLS/WS connection), and proxy must reuse the same flow for inbound calls — even when multiple UAs share the same AOR.

Key elements:

```text
+sip.instance="<urn:uuid:f81d4fae-7dec-11d0-a765-00a0c91e6bf6>"   in Contact
;ob                                                                in Contact URI
;reg-id=1                                                          in Contact (parallel registrations)
Path: <sip:edge-proxy.example.com;lr;ob>                            recorded by proxy
Supported: outbound, path                                           UA + proxy
```

UA opens persistent connection (TLS keepalive via CRLF-CRLF heartbeat per RFC 5626 §3.5.1). Inbound call is delivered over that flow rather than opening a new one.

## RFC Catalog

```text
3261   SIP core (June 2002, obsoletes 2543)
3262   PRACK / 100rel reliable provisional responses
3263   Locating SIP servers (NAPTR/SRV)
3264   Offer/Answer model with SDP
3265   SUBSCRIBE/NOTIFY (obsoleted by 6665)
3311   UPDATE method
3325   P-Asserted-Identity / P-Preferred-Identity / Privacy
3326   Reason header
3428   MESSAGE method
3515   REFER method
3581   Via rport (NAT)
3856   Presence event package
3892   Referred-By header
3903   PUBLISH method
3966   tel: URI
4566   SDP
4733   RTP DTMF events
5359   Service examples
5626   SIP Outbound
6086   INFO method (obsoletes 2976)
6228   199 Early Dialog Terminated
6665   SUBSCRIBE/NOTIFY (obsoletes 3265)
7118   SIP-over-WebSocket
7616   HTTP Digest Authentication (used by SIP)
7621   Trusted Identity Domain
8445   ICE
8489   STUN
8656   TURN
8843   BUNDLE for RTP/RTCP multiplexing
8898   STIR/SHAKEN authenticated identity (RFC 8224 PASSporT)
```

## Common Errors (verbatim text and cause)

```text
401 Unauthorized
WWW-Authenticate: Digest realm="example.com",nonce="...",algorithm=MD5,qop="auth"
   → UAS demands credentials. UAC re-sends with Authorization: Digest ... response="...".

403 Forbidden
   → UAS understands but refuses. Usually wrong username, deactivated trunk, ACL deny.
   → Do NOT retry with creds; provider will rate-limit.

404 Not Found
   → Registrar has no binding for the Request-URI's user@domain.
   → Caller's destination doesn't exist (typo, deregistered).

407 Proxy Authentication Required
Proxy-Authenticate: Digest realm="proxy.example.com",nonce="..."
   → Proxy demands creds. UAC adds Proxy-Authorization.

408 Request Timeout
   → Timer B fired (no provisional/final response in 32s).
   → Or proxy could not reach upstream within Timer C.

415 Unsupported Media Type
Accept: application/sdp
   → UAS does not accept the body MIME (rare for SDP; common for video proprietary formats).

486 Busy Here
   → User is busy (other call in progress on this UA). Try forking; do not assume global busy.

488 Not Acceptable Here
Warning: 304 sip.example.com "Incompatible media format"
   → SDP negotiation failed at this UA. Common with codec mismatch (G.722 vs G.711).
   → Or DTLS fingerprint missing/mismatch in WebRTC.

503 Service Unavailable
Retry-After: 30
   → Server overload / graceful drain. UAC should mark target failed and try alternates.
```

## Common Gotchas

1. Missing To-tag in 200 OK to in-dialog request

```text
Broken:
   BYE sip:bob@... SIP/2.0
   ...
   To: Bob <sip:bob@...>            ← no tag
   ...
   Result: 481 Call/Transaction Does Not Exist
Fixed:
   To: Bob <sip:bob@...>;tag=a6c85cf
```

2. Wrong Digest realm

```text
Broken:
   WWW-Authenticate: Digest realm="ProxyA",...
   Authorization: Digest realm="ProxyB",...   ← UAC used cached realm
   Result: 401 retry, eventually fail.
Fixed:
   Match the realm verbatim from the latest WWW-Authenticate.
```

3. rport not set, response lost behind NAT

```text
Broken:
   Via: SIP/2.0/UDP 10.0.0.5:5060;branch=z9hG4bK1
   Response sent to 10.0.0.5:5060 → black-holed.
Fixed:
   Via: SIP/2.0/UDP 10.0.0.5:5060;branch=z9hG4bK1;rport
   UAS replies to 198.51.100.1:37621 (received/rport recorded).
```

4. Glare on simultaneous re-INVITE (RFC 3261 §14.1)

```text
Broken:
   Both UAs send re-INVITE at same time → both get 491 Request Pending.
Fixed:
   On 491, wait random delay:
     if owns higher Call-ID + From-tag: 2.1 to 4.0 s
     else:                              0 to 2.0 s
   Then re-send re-INVITE with new CSeq.
```

5. Max-Forwards = 0 yields 483 Too Many Hops

```text
Broken:
   INVITE sip:bob@... SIP/2.0
   Max-Forwards: 0
   Result: 483 Too Many Hops (proxy returns; will not decrement past 0).
Fixed:
   Max-Forwards: 70 (default).
```

6. Missing PRACK with Require: 100rel

```text
Broken:
   180 Ringing arrives with Require: 100rel and RSeq: 1 — UAC ignores PRACK.
   UAS retransmits 180 every T1, 2T1, 4T1, ... until timer; eventually 504 or call drops.
Fixed:
   UAC sends PRACK with RAck: 1 314159 INVITE; UAS stops retransmits.
```

7. SDP setup mismatch (DTLS)

```text
Broken:
   Offer  a=setup:actpass
   Answer a=setup:actpass        ← both, neither client nor server
   Result: DTLS handshake never completes; ICE OK but media silent.
Fixed:
   Answer must be a=setup:active (UAC offers actpass, UAS picks active or passive but not actpass).
```

8. Codec ordering ambiguity

```text
Broken:
   m=audio 49170 RTP/AVP 0 8 9
   a=rtpmap:0 PCMU/8000
   a=rtpmap:8 PCMA/8000
   a=rtpmap:9 G722/8000
   UAS picks PCMA (in middle) — UAC wanted G.722 first.
Fixed:
   Use offerer order in fmt list; place preferred codec first:
   m=audio 49170 RTP/AVP 9 0 8
```

9. REFER without Replaces for attended transfer

```text
Broken:
   REFER with Refer-To: <sip:carol@...>  (blind transfer)
   But user wanted attended (introduced + warm).
Fixed:
   Refer-To: <sip:carol@...?Replaces=callid%3Bto-tag%3Dxyz%3Bfrom-tag%3Dabc>
   (Replaces value URL-encoded with %3B for ;)
```

10. Contact with private IP

```text
Broken:
   Contact: <sip:alice@10.0.0.5:5060>
   BYE goes to 10.0.0.5 — drops.
Fixed:
   Proxy uses Record-Route + rport + received.
   Or UA discovers public IP (STUN) and uses it.
   Or use SIP Outbound (;ob, +sip.instance).
```

11. Forgetting Record-Route ;lr triggers strict-routing

```text
Broken:
   Record-Route: <sip:proxy.example.com>     ← no ;lr
   Subsequent BYE has Request-URI = sip:proxy.example.com and Route includes original target.
   Most UAs cannot handle (strict routing in RFC 2543).
Fixed:
   Record-Route: <sip:proxy.example.com;lr>
```

12. 200 OK retransmission not handled

```text
Broken:
   UAS sent 200 OK, UAC ACKed but NAT lost it.
   UAS keeps retransmitting per Timer G — UAC sees duplicate 200s and gets confused.
Fixed:
   UAC must absorb duplicate 200 (per dialog) and re-ACK.
   ACK is end-to-end; sent every time a duplicate 200 arrives.
```

13. Wrong Content-Length over TCP

```text
Broken:
   TCP framed messages depend on Content-Length; off-by-one truncates body.
Fixed:
   Count octets of body (including leading CRLF only if header-body separator is doubled).
   Always set Content-Length: 0 for empty bodies.
```

14. CSeq decreasing in dialog

```text
Broken:
   CSeq: 5 INVITE  → CSeq: 3 BYE
   UAS rejects 500 / ignores.
Fixed:
   Always increment.
```

15. CANCEL race with 200 OK

```text
Broken:
   UAC sends CANCEL.
   UAS already sent 200 OK simultaneously.
   UAC sees 200 and the CANCEL transaction times out.
Fixed:
   On receiving 200 OK after sending CANCEL: ACK the 200, then send BYE.
   200 OK to CANCEL is unrelated to call termination.
```

16. Missing Allow header in 405 response

```text
Broken:
   405 Method Not Allowed                    ← no Allow header
Fixed:
   405 Method Not Allowed
   Allow: INVITE, ACK, CANCEL, BYE, OPTIONS, MESSAGE
```

17. Ignoring Min-SE causes 422

```text
Broken:
   INVITE
   Session-Expires: 60
   Min-SE: 60
   Result: 422 Session Interval Too Small
   Min-SE: 90
Fixed:
   Re-INVITE with Session-Expires: 90 and Min-SE: 90.
```

18. SUBSCRIBE without Event header

```text
Broken:
   SUBSCRIBE sip:bob@... SIP/2.0
   ...
   (no Event header)
   Result: 489 Bad Event
Fixed:
   Event: presence
   Accept: application/pidf+xml
```

## Example Call Flow (annotated ladder)

```text
Alice@atlanta            Proxy A           Proxy B           Bob@biloxi

INVITE 1 ─────────────►
                  100 Trying ◄────
                          INVITE 1 ────────►
                                    100 Trying ◄──
                                            INVITE 1 ─────►
                                                       180 Ringing ◄────
                                            180 Ringing ◄──
                          180 Ringing ◄────
180 Ringing ◄────
                                                       200 OK ◄────
                                            200 OK ◄──
                          200 OK ◄────
200 OK ◄────
ACK ─────────────────────────────────────────────────────►   (in-dialog, follows Route)

   ====== RTP flows directly between Alice and Bob ======

BYE ─────────────────────────────────────────────────────►
                                                       200 OK ◄────
200 OK ◄────────────────────────────────────────────────
```

If proxies inserted Record-Route, BYE traverses them; otherwise BYE goes directly UA-to-UA via Contact.

## Tools

### sngrep — interactive SIP capture and ladder display

```bash
# Live capture with TLS decryption (server private key required)
sudo sngrep -d eth0 -k tls.key

# Read PCAP, filter
sngrep -I capture.pcap -f "host 198.51.100.1"

# Filter by Call-ID once inside
F → enter Call-ID

# Save selected dialog
sngrep ... → Save → PCAP / TXT
```

Keys: F=filter, S=save, C=settings, X=close call, ENTER=open ladder, F2=save raw.

### SIPp — load testing

```bash
# UAC scenario, default UAC.xml
sipp -sn uac sip-server.example.com

# Run a custom scenario, 50 cps, 1000 calls total
sipp -sf my_scenario.xml -r 50 -m 1000 -s alice 192.0.2.10:5060

# UAS waiting for INVITE
sipp -sn uas

# CSV-driven calls
sipp -sf my_scenario.xml -inf calls.csv -m 1000 192.0.2.10
```

Scenario XML core:

```text
<scenario>
  <send><![CDATA[ INVITE ... ]]></send>
  <recv response="100" optional="true"/>
  <recv response="180" optional="true"/>
  <recv response="200" rtd="true"/>
  <send><![CDATA[ ACK ... ]]></send>
  <pause milliseconds="3000"/>
  <send><![CDATA[ BYE ... ]]></send>
  <recv response="200"/>
</scenario>
```

### Wireshark + TLS keylog

```bash
# Capture
sudo tshark -i any -w sip.pcap -f "port 5060 or port 5061 or portrange 10000-20000"

# Decode SIP (UDP 5060 already auto-detected)
wireshark sip.pcap

# Force TCP non-default port to be SIP
Edit → Preferences → Protocols → SIP → "TCP ports" = 5060

# TLS decryption
TLS keylog file path:  /tmp/sslkeys.log   (set NSS export SSLKEYLOGFILE=/tmp/sslkeys.log on the SIP UA)
Edit → Preferences → Protocols → TLS → Pre-Master-Secret log = /tmp/sslkeys.log

# RTP analysis
Telephony → RTP → Stream Analysis → forward + reverse → save .au audio
Telephony → VoIP Calls → Flow → ladder-style flow chart
```

Display filters:

```text
sip                              all SIP traffic
sip.Method == "INVITE"           only INVITEs
sip.Status-Code == 200           200 responses
sip.Call-ID == "abc@host"        specific dialog
sdp                              SDP only
rtp.ssrc == 0x12345678           specific RTP source
```

### sipsak — Swiss-Army knife for SIP

```bash
sipsak -s sip:alice@example.com               OPTIONS to AOR
sipsak -T -s sip:alice@example.com            full INVITE-trace
sipsak -F -s sip:alice@example.com -e 12345   send "12345" via flood-mode (load test)
sipsak -M -s sip:alice@example.com -B "hi"    MESSAGE with body
sipsak -U -C sip:alice@registrar -a secret    REGISTER
```

### pjsua / baresip / linphone-cli — soft phones for scripting

```bash
# pjsua interactive
pjsua --id sip:alice@example.com --registrar sip:example.com \
      --realm '*' --username alice --password secret

# baresip CLI
baresip -e "/dial sip:bob@example.com"
baresip -e "/auplay sox"

# linphone-cli
linphonecsh init
linphonecsh dial sip:bob@example.com
linphonecsh hangup
linphonecsh exit
```

## Idioms

```text
Probe trunk health:
   while true; do sipsak -s sip:trunk.example.com; sleep 30; done

Capture only new calls (sngrep autoscroll):
   sudo sngrep -O auto -L info

Extract Call-ID from PCAP:
   tshark -r sip.pcap -Y sip -T fields -e sip.Call-ID -e ip.src -e ip.dst | sort -u

Decode digest auth:
   echo -n "alice:atlanta:secret" | md5sum   # → HA1

Test TLS handshake:
   openssl s_client -connect sip.example.com:5061 -servername sip.example.com -tlsextdebug

Force IPv4 on Linphone:
   linphonecsh proxy address sip:example.com\;transport=tcp

Asterisk SIP debug (chan_pjsip):
   asterisk -rx "pjsip set logger on"
   asterisk -rx "pjsip show endpoints"

FreeSWITCH SIP debug:
   fs_cli -x "sofia loglevel all 9"
   fs_cli -x "sofia status profile internal reg"

Find calls hung in early state:
   asterisk -rx "core show channels concise" | awk -F! '$5=="Ring" && $9>30'

Force RTP through SBC (Asterisk PJSIP):
   set "direct_media=no" + "rtp_symmetric=yes" on endpoint.

Generate UUID for sip.instance:
   uuidgen   # → urn:uuid:<value>

Inspect Via stack live:
   tshark -r sip.pcap -Y sip -T fields -e sip.Via | head
```

## Topology hiding (SBC behaviour)

Border SBC strips internal Via lines and Record-Route entries before forwarding to outside, replaces Contact with its own URI, removes proprietary headers (X-, P-Charging-Vector inside trust boundary). On response, it reverses. Used to prevent enumerating internal proxy chains.

## STIR/SHAKEN (RFCs 8224, 8225, 8226)

For US/Canada anti-spoof: each call carries an `Identity:` header with a JWT (PASSporT) signed by originating carrier proving caller-ID legitimacy.

```text
Identity: eyJhbGciOiJFUzI1NiIsInBwdCI6InNoYWtlbiIsInR5cCI6InBhc3Nwb3J0IiwieDV1IjoiaHR0cHM6Ly9jZXJ0LmV4YW1wbGUuY29tL2NlcnQucGVtIn0.eyJhdHRlc3QiOiJBIiwiZGVzdCI6eyJ0biI6WyIxMjAyNTU1MTIzNCJdfSwiaWF0IjoxNDQzMjA4MzQ1LCJvcmlnIjp7InRuIjoiMTIwMjU1NTAwMDAifSwib3JpZ2lkIjoiMTIzNDU2In0.signature;info=<https://cert.example.com/cert.pem>;alg=ES256;ppt=shaken
```

Attestation:

```text
A    Full attestation     carrier authenticated subscriber + verified caller-ID
B    Partial attestation  carrier authenticated subscriber but not caller-ID
C    Gateway attestation  carrier did not authenticate origin (e.g., international gateway)
```

## Asterisk + SIP examples

`pjsip.conf`:

```text
[transport-tls]
type=transport
protocol=tls
bind=0.0.0.0:5061
cert_file=/etc/asterisk/cert.pem
priv_key_file=/etc/asterisk/key.pem
allow_reload=yes

[alice]
type=endpoint
context=internal
disallow=all
allow=opus,ulaw,alaw,telephone-event
auth=alice-auth
aors=alice
direct_media=no
rtp_symmetric=yes
force_rport=yes
rewrite_contact=yes
ice_support=yes
dtls_auto_generate_cert=yes
media_encryption=dtls
dtls_setup=actpass
dtls_verify=fingerprint

[alice-auth]
type=auth
auth_type=userpass
username=alice
password=secret

[alice]
type=aor
max_contacts=5
remove_existing=yes
qualify_frequency=60
```

## FreeSWITCH SIP examples

`sip_profiles/internal.xml` essentials:

```text
<param name="sip-port"   value="5060"/>
<param name="rtp-ip"     value="$${local_ip_v4}"/>
<param name="ext-rtp-ip" value="auto-nat"/>
<param name="ext-sip-ip" value="auto-nat"/>
<param name="apply-nat-acl"   value="nat.auto"/>
<param name="rtp-rewrite-timestamps" value="true"/>
<param name="dtmf-duration"   value="2000"/>
<param name="rfc2833-pt"      value="101"/>
<param name="dtmf-type"       value="rfc2833"/>
<param name="auth-calls"      value="true"/>
<param name="accept-blind-reg" value="false"/>
<param name="user-agent-string" value="FreeSWITCH"/>
<param name="enable-3pcc"     value="true"/>
```

## Worked Digest Authentication Example

The full HMAC-MD5 computation showing exactly what each side computes. This is the most common SIP auth flow and the one most often misconfigured.

Server challenge:

```text
SIP/2.0 401 Unauthorized
Via: SIP/2.0/UDP 192.0.2.1:5060;branch=z9hG4bK-abc123
From: Alice <sip:alice@example.com>;tag=1928301774
To: Bob <sip:bob@example.com>;tag=as73c2f4a8
Call-ID: a84b4c76e66710@pc33.example.com
CSeq: 314159 INVITE
WWW-Authenticate: Digest realm="example.com",
   qop="auth",
   nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093",
   opaque="5ccc069c403ebaf9f0171e9517f40e41",
   algorithm=MD5,
   stale=FALSE
Content-Length: 0
```

Client computes (assume password = "secret"):

```text
HA1 = MD5("alice:example.com:secret")
    = MD5("alice:example.com:secret")
    = 939e7578ed9e3c518a452acee763bce9

HA2 = MD5("INVITE:sip:bob@example.com")
    = 39aff3a2bab6126f332b942af96d3366

# qop=auth requires nc + cnonce
nc      = 00000001  (nonce-count, hex 8-digit)
cnonce  = 0a4f113b  (client nonce, random)

response = MD5(HA1:nonce:nc:cnonce:qop:HA2)
         = MD5("939e7578ed9e3c518a452acee763bce9:dcd98b7102dd2f0e8b11d0f600bfb0c093:00000001:0a4f113b:auth:39aff3a2bab6126f332b942af96d3366")
         = 6629fae49393a05397450978507c4ef1
```

Client retry:

```text
INVITE sip:bob@example.com SIP/2.0
Via: SIP/2.0/UDP 192.0.2.1:5060;branch=z9hG4bK-abc124
From: Alice <sip:alice@example.com>;tag=1928301774
To: Bob <sip:bob@example.com>
Call-ID: a84b4c76e66710@pc33.example.com
CSeq: 314160 INVITE
Authorization: Digest username="alice",
   realm="example.com",
   nonce="dcd98b7102dd2f0e8b11d0f600bfb0c093",
   uri="sip:bob@example.com",
   qop=auth,
   nc=00000001,
   cnonce="0a4f113b",
   response="6629fae49393a05397450978507c4ef1",
   opaque="5ccc069c403ebaf9f0171e9517f40e41",
   algorithm=MD5
Contact: <sip:alice@192.0.2.1:5060>
Max-Forwards: 70
Content-Length: 0
```

For algorithm=MD5-sess: HA1 is recomputed as `MD5(MD5(user:realm:pass):nonce:cnonce)`. For qop=auth-int: HA2 includes message-body digest as `MD5(method:uri:MD5(entity-body))`. The SHA-256/SHA-512-256 RFC 7616 variants substitute the hash function but keep the same overall structure.

## INVITE Client Transaction Timer Progression

Concrete timeline showing what happens when a UDP INVITE encounters packet loss. Numbers come from RFC 3261 §17.1.1.2 with default timer values.

```text
T = 0.0s   →  send INVITE         (Calling state, Timer A starts at 500ms)
T = 0.5s   →  Timer A fires       resend INVITE; Timer A doubles to 1.0s
T = 1.5s   →  Timer A fires       resend INVITE; Timer A doubles to 2.0s
T = 3.5s   →  Timer A fires       resend INVITE; Timer A doubles to 4.0s
T = 7.5s   →  Timer A fires       resend INVITE; Timer A doubles to 8.0s
T = 15.5s  →  Timer A fires       resend INVITE; Timer A doubles to 16.0s
T = 31.5s  →  Timer A fires       resend INVITE; Timer A doubles to 32.0s
T = 32.0s  →  Timer B fires       transaction terminated (no response received)
              UAC reports 408 Request Timeout to TU
```

If a 1xx provisional response arrives during Calling state:

```text
T = 0.0s   →  send INVITE
T = 0.4s   →  receive 100 Trying  → state moves Calling → Proceeding
              Timer A cancels; no more retransmissions until Proceeding ends
              Timer B continues; remains armed at original 32.0s deadline
T = 0.6s   →  receive 180 Ringing
T = 30.0s  →  receive 200 OK      → state moves Proceeding → Terminated
              UAC sends ACK; transaction completes successfully
```

If a 4xx-6xx final arrives:

```text
T = 0.0s   →  send INVITE
T = 0.5s   →  receive 486 Busy    → state moves Calling → Completed
              UAC sends ACK; Timer D starts (32s for UDP)
T = 32.5s  →  Timer D fires       state moves Completed → Terminated
```

Timer D's purpose: absorb retransmitted final responses from a slow server. Without Timer D, a delayed retransmission of "486 Busy Here" arriving after the transaction ended would be treated as a new transaction.

For TCP (reliable transport), Timer A is unused (no retransmissions); Timer B fires immediately when the connection closes; Timer D = 0s (no need to absorb retransmissions).

## More Verbatim 4xx-6xx Examples

```text
SIP/2.0 480 Temporarily Unavailable
Via: SIP/2.0/UDP 192.0.2.10:5060;branch=z9hG4bK-...;received=192.0.2.10
From: Alice <sip:alice@example.com>;tag=...
To: Bob <sip:bob@example.com>;tag=server-tag
Call-ID: ...
CSeq: 314 INVITE
Retry-After: 30
Content-Length: 0
```

```text
SIP/2.0 482 Loop Detected
Via: SIP/2.0/UDP proxy3.example.com:5060;branch=...
From: ...
To: ...
Reason: SIP;cause=482;text="Routing loop detected via Via headers"
Content-Length: 0
```

```text
SIP/2.0 483 Too Many Hops
Via: SIP/2.0/UDP proxy7.example.com:5060;branch=...
From: ...
To: ...
Max-Forwards: 0
Content-Length: 0
```

```text
SIP/2.0 485 Ambiguous
Contact: <sip:bob@10.1.0.5>
Contact: <sip:bob@10.1.0.6>
Content-Length: 0
```

```text
SIP/2.0 488 Not Acceptable Here
Warning: 304 example.com "Incompatible media format - codec mismatch"
Reason: Q.850;cause=88;text="Incompatible destination"
Content-Length: 0
```

```text
SIP/2.0 491 Request Pending
# Sent in response to a re-INVITE while another re-INVITE is in flight
# (the "glare" condition); UAC waits and retries with random backoff
```

```text
SIP/2.0 503 Service Unavailable
Retry-After: 30
Reason: SIP;cause=503;text="Backend overloaded"
Content-Length: 0
```

## Sample SDP Variants

### Basic G.711 audio offer

```text
v=0
o=alice 2890844526 2890844526 IN IP4 192.0.2.1
s=-
c=IN IP4 192.0.2.1
t=0 0
m=audio 49170 RTP/AVP 0 8 101
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtpmap:101 telephone-event/8000
a=fmtp:101 0-16
a=ptime:20
a=sendrecv
```

### WebRTC audio + video with BUNDLE + DTLS-SRTP

```text
v=0
o=- 1591574623 2 IN IP4 127.0.0.1
s=-
t=0 0
a=group:BUNDLE 0 1
a=msid-semantic:WMS *

m=audio 9 UDP/TLS/RTP/SAVPF 111 103 9 0 8 105 13 110 113 126
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:F7gG
a=ice-pwd:x9cml/YzichV2+XlhiMu8g
a=ice-options:trickle
a=fingerprint:sha-256 75:74:5A:A6:A4:E5:52:F4:A7:67:4C:01:C7:E9:31:F1:84:24:3C:CC:77:34:A1:74:1A:DA:C8:14:6F:DC:F0:43
a=setup:actpass
a=mid:0
a=extmap:1 urn:ietf:params:rtp-hdrext:ssrc-audio-level
a=sendrecv
a=msid:- a-stream-id
a=rtcp-mux
a=rtpmap:111 opus/48000/2
a=fmtp:111 minptime=10;useinbandfec=1
a=rtpmap:103 ISAC/16000
a=rtpmap:9 G722/8000
a=rtpmap:0 PCMU/8000
a=rtpmap:8 PCMA/8000
a=rtcp-fb:111 transport-cc
a=ssrc:1234567890 cname:abc123

m=video 9 UDP/TLS/RTP/SAVPF 96 97 98 99 100 101 102 121 127 120 125 107 108 109 124 119 123
c=IN IP4 0.0.0.0
a=rtcp:9 IN IP4 0.0.0.0
a=ice-ufrag:F7gG
a=ice-pwd:x9cml/YzichV2+XlhiMu8g
a=fingerprint:sha-256 75:74:5A:A6:A4:E5:52:F4:A7:67:4C:01:C7:E9:31:F1:84:24:3C:CC:77:34:A1:74:1A:DA:C8:14:6F:DC:F0:43
a=setup:actpass
a=mid:1
a=sendrecv
a=msid:- v-stream-id
a=rtcp-mux
a=rtcp-rsize
a=rtpmap:96 VP8/90000
a=rtcp-fb:96 goog-remb
a=rtcp-fb:96 transport-cc
a=rtcp-fb:96 ccm fir
a=rtcp-fb:96 nack
a=rtcp-fb:96 nack pli
a=rtpmap:97 rtx/90000
a=fmtp:97 apt=96
a=rtpmap:98 H264/90000
a=fmtp:98 level-asymmetry-allowed=1;packetization-mode=1;profile-level-id=42e01f
```

### IPv6 SDP

```text
v=0
o=- 1591574999 1 IN IP6 2001:db8::1
s=-
c=IN IP6 2001:db8::1
t=0 0
m=audio 50000 RTP/AVP 0 101
a=rtpmap:0 PCMU/8000
a=rtpmap:101 telephone-event/8000
a=sendrecv
```

## ICE Candidate-Pair Scoring Walk-Through

Two endpoints behind separate NATs. Each gathered candidates:

```text
Alice (controlling):
  host        192.168.1.10:50000  prio=2113929471
  srflx       203.0.113.5:50000   prio=1677729535  (via STUN at 203.0.113.5)
  relay       198.51.100.99:55000 prio=8388607     (TURN)

Bob (controlled):
  host        10.0.0.42:60000     prio=2113929471
  srflx       198.51.100.42:60000 prio=1677729535
  relay       198.51.100.99:65000 prio=8388607
```

Pair priority (RFC 8445 §6.1.2.3): pair_priority = 2^32 * MIN(G, D) + 2*MAX(G, D) + (G > D ? 1 : 0), where G is the controlling agent's component priority and D is the controlled agent's. The candidate-pair list is then sorted descending and STUN connectivity checks are sent in priority order.

```text
Pair 1: host-host        Alice 192.168.1.10:50000  ↔ Bob 10.0.0.42:60000
        # Different LANs — packets dropped, fails
Pair 2: srflx-srflx      Alice 203.0.113.5:50000   ↔ Bob 198.51.100.42:60000
        # Public IPs but different NAT types — may succeed depending on NAT behavior
Pair 3: host-srflx       Alice 192.168.1.10:50000  ↔ Bob 198.51.100.42:60000
        # Alice's private IP not reachable; fails
Pair 4: relay-host       Alice 198.51.100.99:55000 ↔ Bob 10.0.0.42:60000
        # Alice has public TURN, but Bob's host is private; fails
Pair 5: relay-relay      Alice 198.51.100.99:55000 ↔ Bob 198.51.100.99:65000
        # Both relay — guaranteed to work via TURN; nominate as last resort
```

In practice ICE finds the best working pair within ~200ms over WebSocket-signaled candidates; the 3-RTT-worst-case for STUN connectivity check is amortized by parallelism.

## More Common Errors With Cause + Fix

```text
SIP/2.0 408 Request Timeout
# Cause: Server side of UDP transaction never replied; client retransmissions exhausted Timer B (32s).
# Fix: 1) confirm UAS is reachable (ping, dig, traceroute);
#      2) confirm rport is set so server can reply through symmetric NAT;
#      3) increase tcp_connection_timeout if using TCP.
```

```text
SIP/2.0 482 Loop Detected
# Cause: Outgoing INVITE has matched a Via header value already in the request.
# Fix: Inspect the proxy chain: an upstream proxy is forwarding back to a node
#      that already touched the request. Common with misconfigured ENUM/DNS SRV.
#      Use sngrep to trace the Via stack.
```

```text
SIP/2.0 488 Not Acceptable Here
# Warning: 304 example.com "Incompatible media format"
# Cause: SDP offer specified codecs not supported by callee.
# Fix: Inspect a=rtpmap lines on both sides; ensure overlap.
#      For G.722 vs G.722.1 (different specs!) check the clock rate.
```

```text
SIP/2.0 491 Request Pending
# Cause: re-INVITE arrived while another mid-dialog transaction is in flight.
# Fix: UAC waits T1 + T2 random backoff and retries.
#      In practice this means the application must serialize re-INVITEs.
```

```text
ERR pjsip Endpoint X has no AORs configured
# Cause: PJSIP endpoint has no aors= line referencing a defined [aor X] block.
# Fix: Add aors=X to the [endpoint X] block, and define a [aor X] section
#      with at least a contact= line (or set max_contacts=N for registrar mode).
```

```text
ERR rtp_io: ssl_handshake: tls error: certificate verify failed
# Cause: TLS peer certificate not trusted; usually missing CA in trust store.
# Fix: Add the issuing CA to /etc/asterisk/keys/ or /etc/freeswitch/tls/ca-bundle.crt
#      and reference it in cert_file/ca_file directives. Verify with:
#        openssl s_client -connect host:5061 -showcerts -CAfile bundle.crt
```

## See Also

- [asterisk](../telephony/asterisk.md) — open-source PBX, chan_pjsip / chan_sip implementations
- [freeswitch](../telephony/freeswitch.md) — modular soft-switch with mod_sofia
- [rtp-sdp](../telephony/rtp-sdp.md) — Real-time Transport Protocol carrying media
- [ip-phone-provisioning](../telephony/ip-phone-provisioning.md) — config templates, TFTP/HTTP boot
- [sip-trunking](../telephony/sip-trunking.md) — carrier interconnect, SBC, STIR/SHAKEN
- [tls](../security/tls.md) — Transport Layer Security used for sips: and WSS
- [dns](../networking/dns.md) — NAPTR/SRV records for SIP discovery

## References

- RFC 3261 — SIP: Session Initiation Protocol — https://tools.ietf.org/html/rfc3261
- RFC 3262 — Reliability of Provisional Responses — https://tools.ietf.org/html/rfc3262
- RFC 3263 — Locating SIP Servers — https://tools.ietf.org/html/rfc3263
- RFC 3264 — Offer/Answer Model with SDP — https://tools.ietf.org/html/rfc3264
- RFC 3265 — SUBSCRIBE/NOTIFY (obsoleted) — https://tools.ietf.org/html/rfc3265
- RFC 3311 — UPDATE method — https://tools.ietf.org/html/rfc3311
- RFC 3325 — P-Asserted-Identity — https://tools.ietf.org/html/rfc3325
- RFC 3326 — Reason header — https://tools.ietf.org/html/rfc3326
- RFC 3428 — MESSAGE method — https://tools.ietf.org/html/rfc3428
- RFC 3515 — REFER method — https://tools.ietf.org/html/rfc3515
- RFC 3581 — Symmetric Response (rport) — https://tools.ietf.org/html/rfc3581
- RFC 3680 — reg event package — https://tools.ietf.org/html/rfc3680
- RFC 3856 — Presence event package — https://tools.ietf.org/html/rfc3856
- RFC 3892 — Referred-By header — https://tools.ietf.org/html/rfc3892
- RFC 3903 — PUBLISH method — https://tools.ietf.org/html/rfc3903
- RFC 3966 — tel: URI — https://tools.ietf.org/html/rfc3966
- RFC 4028 — Session Timers — https://tools.ietf.org/html/rfc4028
- RFC 4168 — SCTP transport for SIP — https://tools.ietf.org/html/rfc4168
- RFC 4235 — dialog event package — https://tools.ietf.org/html/rfc4235
- RFC 4474 — Identity (obsoleted by 8224) — https://tools.ietf.org/html/rfc4474
- RFC 4566 — SDP — https://tools.ietf.org/html/rfc4566
- RFC 4733 — RTP Telephone-Events — https://tools.ietf.org/html/rfc4733
- RFC 5359 — Service examples — https://tools.ietf.org/html/rfc5359
- RFC 5626 — SIP Outbound — https://tools.ietf.org/html/rfc5626
- RFC 5763 — DTLS for SDP setup — https://tools.ietf.org/html/rfc5763
- RFC 6086 — INFO method — https://tools.ietf.org/html/rfc6086
- RFC 6228 — 199 Early Dialog Terminated — https://tools.ietf.org/html/rfc6228
- RFC 6665 — SUBSCRIBE/NOTIFY — https://tools.ietf.org/html/rfc6665
- RFC 7044 — History-Info header — https://tools.ietf.org/html/rfc7044
- RFC 7118 — SIP-over-WebSocket — https://tools.ietf.org/html/rfc7118
- RFC 7616 — HTTP Digest Authentication — https://tools.ietf.org/html/rfc7616
- RFC 7621 — Trusted Identity Domain — https://tools.ietf.org/html/rfc7621
- RFC 8197 — 607 Unwanted — https://tools.ietf.org/html/rfc8197
- RFC 8224 — STIR Authenticated Identity — https://tools.ietf.org/html/rfc8224
- RFC 8225 — PASSporT — https://tools.ietf.org/html/rfc8225
- RFC 8226 — STIR Certificates — https://tools.ietf.org/html/rfc8226
- RFC 8445 — ICE — https://tools.ietf.org/html/rfc8445
- RFC 8489 — STUN — https://tools.ietf.org/html/rfc8489
- RFC 8599 — Push Notification (555) — https://tools.ietf.org/html/rfc8599
- RFC 8656 — TURN — https://tools.ietf.org/html/rfc8656
- RFC 8688 — 608 Rejected — https://tools.ietf.org/html/rfc8688
- RFC 8843 — BUNDLE — https://tools.ietf.org/html/rfc8843
- IANA SIP Parameters — https://www.iana.org/assignments/sip-parameters/sip-parameters.xhtml
- IANA SDP Parameters — https://www.iana.org/assignments/sdp-parameters/sdp-parameters.xhtml
- IETF SIPCORE WG — https://datatracker.ietf.org/wg/sipcore/about/
- IETF SIP-over-WebSocket — https://datatracker.ietf.org/wg/sipcore/documents/
- sngrep — https://github.com/irontec/sngrep
- SIPp — https://sipp.sourceforge.net/
- pjsip — https://www.pjsip.org/
- Wireshark SIP — https://wiki.wireshark.org/SIP
- Asterisk PJSIP — https://docs.asterisk.org/Configuration/Channel-Drivers/SIP/
- FreeSWITCH mod_sofia — https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod-sofia_3965776/
- JsSIP — https://jssip.net/
- SIP.js — https://sipjs.com/
