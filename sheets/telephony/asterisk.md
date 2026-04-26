# Asterisk PBX

Open-source telephony platform: SIP/PJSIP, IAX2, DAHDI, dialplan, AGI/AMI/ARI. Single-process, channel-based, drives carrier-grade PBX, IVR, conferencing, queues, voicemail.

## Setup

Install on Debian/Ubuntu/RHEL/source. Versions: 16 LTS, 18 LTS, 20 LTS, 21, 22 LTS. Use LTS for production. PJSIP is the default SIP stack since 12 — chan_sip removed in 21.

```bash
# Debian/Ubuntu — distro package (often older)
sudo apt-get install asterisk asterisk-modules

# RHEL/Rocky/Alma
sudo dnf install asterisk

# Source build (preferred for current LTS)
wget https://downloads.asterisk.org/pub/telephony/asterisk/asterisk-20-current.tar.gz
tar xzf asterisk-20-current.tar.gz
cd asterisk-20.*
sudo contrib/scripts/install_prereq install
./configure --with-jansson-bundled --with-pjproject-bundled
make menuselect                # tick codecs, app_meetme, etc
make -j$(nproc)
sudo make install
sudo make samples              # default configs in /etc/asterisk/
sudo make config               # systemd unit
sudo make install-logrotate

# Service
sudo systemctl enable asterisk
sudo systemctl start asterisk
sudo systemctl status asterisk

# User/group (do NOT run as root)
sudo groupadd asterisk
sudo useradd -r -d /var/lib/asterisk -g asterisk asterisk
sudo chown -R asterisk:asterisk /etc/asterisk /var/{lib,log,spool}/asterisk /usr/lib/asterisk

# /etc/default/asterisk (Debian)
AST_USER="asterisk"
AST_GROUP="asterisk"
```

Key directories:

```
/etc/asterisk/        # config files (~/.conf, modules.conf, pjsip.conf, extensions.conf)
/var/lib/asterisk/    # sounds, AGI scripts, keys, moh
/var/spool/asterisk/  # voicemail, monitor recordings, outgoing call files
/var/log/asterisk/    # full log, messages, queue_log, cdr-csv/
/usr/lib/asterisk/modules/  # *.so modules
```

## Console (`asterisk -rvvvv`)

The Asterisk CLI is what `fs_cli` is to FreeSWITCH — interactive console attached to the running process via UNIX socket.

```bash
# Attach to running Asterisk (4 v's = verbose level 4)
sudo asterisk -rvvvv

# Even more verbose
sudo asterisk -rvvvvvvvvv     # nine v's
sudo asterisk -r -v 9         # explicit level

# One-shot command (no interactive shell)
sudo asterisk -rx "core show channels"
sudo asterisk -rx "pjsip show endpoints"
sudo asterisk -rx "module reload"

# Start in foreground (debug)
sudo asterisk -cvvvvg
#  -c console foreground
#  -v verbose
#  -g dump core on crash

# Start daemonized
sudo asterisk -F
```

In-console commands:

```
core show version
core show uptime
core show channels [verbose|concise|count]
core show channel SIP/foo-00000001
core show applications
core show application Dial
core show functions
core show function CALLERID
core set verbose 5
core set debug 5
core restart now           # full restart
core restart gracefully    # wait for calls to drop
core restart when convenient
core stop now
core stop gracefully
core reload                # reload everything
module reload pjsip.so
module reload chan_pjsip.so
module reload res_pjsip.so
module load <name>
module unload <name>
module show like pjsip
dialplan reload
dialplan show
dialplan show internal     # show context "internal"
dialplan show 100@internal # specific extension
pjsip reload
pjsip show endpoints
pjsip show endpoint 1001
pjsip show registrations
pjsip show transports
pjsip show contacts
pjsip show aors
pjsip show auths
pjsip set logger on        # SIP packet trace
sip set debug on           # chan_sip equivalent (legacy)
rtp set debug on
rtcp set debug on
queue show
queue show sales
voicemail show users
agi set debug on
manager show users
manager show connected
ari show apps
logger reload
logger rotate
logger show channels
core show locks
core show taskprocessors
file convert in.wav out.gsm
```

Shortcut: `!shell-cmd` runs a shell command from inside the CLI:

```
*CLI> !ls /var/spool/asterisk/voicemail/default/
*CLI> !tail -f /var/log/asterisk/full
```

## Configuration File Format (INI)

All `.conf` files are INI-style with sections (`[name]`), key/value lines (`key = value` or `key => value`), and `;` comments. Most files support templates and inheritance.

```ini
; Comment
[section-name]
key = value
key => value          ; equivalent (legacy "object" syntax)

; Template — define once, inherit everywhere
[base-endpoint](!)            ; the (!) makes it a template
context = internal
disallow = all
allow = ulaw,alaw,g722

[1001](base-endpoint)         ; inherits from base-endpoint
auth = 1001
aors = 1001

; Multiple inheritance
[1002](base-endpoint,extra-template)
auth = 1002

; #include another file (relative to /etc/asterisk/)
#include "users.conf"
#include "/etc/asterisk/extensions_extra.conf"

; #tryinclude — no error if file missing
#tryinclude "local.conf"

; #exec runs a command, includes stdout (must be enabled in asterisk.conf)
#exec "/usr/local/bin/gen-extensions.sh"
```

`asterisk.conf` controls runtime options:

```ini
[directories](!)
astetcdir => /etc/asterisk
astmoddir => /usr/lib/asterisk/modules
astvarlibdir => /var/lib/asterisk
astdbdir => /var/lib/asterisk
astkeydir => /var/lib/asterisk
astdatadir => /var/lib/asterisk
astagidir => /var/lib/asterisk/agi-bin
astspooldir => /var/spool/asterisk
astrundir => /var/run/asterisk
astlogdir => /var/log/asterisk

[options]
verbose = 3
debug = 0
runuser = asterisk
rungroup = asterisk
languageprefix = yes
execincludes = yes      ; allow #exec
live_dangerously = no   ; block dangerous funcs (System(), SHELL())
```

## modules.conf

Controls which `.so` modules load at startup. Default is `autoload=yes` — explicitly disable what you don't need.

```ini
[modules]
autoload=yes

; Don't load these (chan_sip removed in 21 anyway)
noload => chan_sip.so
noload => chan_mgcp.so
noload => chan_skinny.so
noload => chan_unistim.so
noload => res_hep.so
noload => res_hep_pjsip.so
noload => res_hep_rtcp.so
noload => app_meetme.so          ; deprecated, use ConfBridge
noload => cdr_mysql.so
noload => cdr_pgsql.so
noload => app_voicemail_imap.so
noload => app_voicemail_odbc.so

; Force-load before autoload pass
preload => res_odbc.so
preload => res_config_odbc.so

; Explicit load (when autoload=no)
load => res_pjsip.so
load => chan_pjsip.so
load => app_dial.so
```

After editing: `module reload` from CLI, or `core restart now` if you toggled `noload`.

## logger.conf

Controls log files, levels, and per-channel routing.

```ini
[general]
dateformat = %F %T.%3q       ; ISO8601 + ms
exec_after_rotate = gzip -9 ${filename}.2
queue_log = yes
queue_log_to_file = yes
queue_log_name = queue_log
event_log = yes
rotatestrategy = timestamp   ; or 'sequential' or 'rotate'
use_callids = yes

[logfiles]
; filename => level1,level2,...
console => notice,warning,error
messages => notice,warning,error
full => notice,warning,error,debug,verbose(5),dtmf,fax
security => security
queue_log => queue_log
debug => debug

; Log levels: debug, verbose, notice, warning, error, dtmf, fax, security
; verbose can be qualified: verbose(5) means levels 1-5
```

CLI:

```
logger reload
logger rotate
logger show channels
logger set level DEBUG on
logger set level VERBOSE on
logger add channel /tmp/mylog notice,warning,error
logger remove channel /tmp/mylog
```

## chan_pjsip vs chan_sip

`chan_sip` was Asterisk's original SIP channel driver. `chan_pjsip` (since 12, default since 13) is built on PJProject and is the only supported SIP stack going forward.

| Aspect | chan_sip (legacy) | chan_pjsip (modern) |
|---|---|---|
| Status | Removed in Asterisk 21 | Default and only supported |
| Config file | `sip.conf` | `pjsip.conf` |
| Channel prefix | `SIP/peer-xxxxxxxx` | `PJSIP/endpoint-xxxxxxxx` |
| Multiple transports | One per protocol | Multiple per protocol |
| Multiple identities | No | Yes (multiple endpoints same auth) |
| TLS | Limited | Full |
| WebRTC | Limited | First-class |
| ICE | No | Yes |
| Outbound registration | One per peer | Many per endpoint |
| CLI command | `sip show peers` | `pjsip show endpoints` |
| Realtime | sippeers | ps_endpoints, ps_aors, ps_auths, ps_contacts |

Migrate: `contrib/scripts/sip_to_pjsip/sip_to_pjsip.py /etc/asterisk/sip.conf > /etc/asterisk/pjsip.conf`

## pjsip.conf — transport + endpoint + auth + aor

Four object types form the core of any PJSIP setup:

- **transport** — listening socket (UDP/TCP/TLS/WS/WSS, IP, port)
- **endpoint** — call-routing identity (codecs, context, NAT, DTMF)
- **auth** — username/password (separable so one auth can serve many endpoints)
- **aor** — Address of Record, where to send INVITEs (registered contact or static URI)

Minimum viable PJSIP config for one extension:

```ini
;========== transport ==========
[transport-udp]
type = transport
protocol = udp
bind = 0.0.0.0:5060
external_media_address = 203.0.113.10
external_signaling_address = 203.0.113.10
local_net = 192.168.0.0/16
local_net = 10.0.0.0/8

[transport-tcp]
type = transport
protocol = tcp
bind = 0.0.0.0:5060

[transport-tls]
type = transport
protocol = tls
bind = 0.0.0.0:5061
cert_file = /etc/asterisk/keys/asterisk.crt
priv_key_file = /etc/asterisk/keys/asterisk.key
ca_list_file = /etc/asterisk/keys/ca.crt
method = tlsv1_2
verify_client = no
verify_server = no
require_client_cert = no

[transport-wss]
type = transport
protocol = wss
bind = 0.0.0.0:8089
cert_file = /etc/asterisk/keys/asterisk.crt
priv_key_file = /etc/asterisk/keys/asterisk.key

;========== template ==========
[endpoint-internal](!)
type = endpoint
context = internal
disallow = all
allow = ulaw
allow = alaw
allow = g722
direct_media = no
force_rport = yes
rewrite_contact = yes
rtp_symmetric = yes
trust_id_inbound = yes
device_state_busy_at = 1
language = en
dtmf_mode = rfc4733

[auth-userpass](!)
type = auth
auth_type = userpass

[aor-single](!)
type = aor
max_contacts = 1
remove_existing = yes
qualify_frequency = 60

;========== extension 1001 ==========
[1001](endpoint-internal)
auth = 1001
aors = 1001
callerid = "Alice" <1001>
mailboxes = 1001@default

[1001](auth-userpass)
username = 1001
password = SuperSecret_change_me

[1001](aor-single)
```

CLI verification:

```
pjsip show transports
pjsip show endpoints
pjsip show endpoint 1001
pjsip show contacts
pjsip show aors
pjsip show auths
pjsip set logger on
pjsip reload
```

## PJSIP Endpoints

The endpoint object holds everything about how Asterisk treats calls to/from this identity. Most-used options:

```ini
[1001]
type = endpoint
context = internal              ; dialplan context for inbound calls
disallow = all
allow = ulaw,alaw,g722
allow = opus                    ; WebRTC

callerid = "Alice" <1001>       ; caller ID for outbound calls
trust_id_inbound = yes          ; trust P-Asserted-Identity / Remote-Party-ID
trust_id_outbound = yes
send_pai = yes                  ; send P-Asserted-Identity
send_rpid = no
identify_by = username,auth_username,ip

dtmf_mode = rfc4733             ; rfc4733 | inband | info | auto
                                ; rfc4733 = RFC 2833/4733 telephone-event
                                ; auto only switches if rfc4733 not negotiated

direct_media = no               ; let RTP go peer-to-peer? (no = anchor at Asterisk)
direct_media_method = invite    ; invite | reinvite | update
direct_media_glare_mitigation = none
disable_direct_media_on_nat = yes

force_rport = yes               ; reply to source port even if Via says otherwise
rewrite_contact = yes           ; replace contact with source IP:port (NAT)
rtp_symmetric = yes             ; send RTP back to where we got it from
ice_support = yes               ; WebRTC
use_avpf = yes                  ; WebRTC SAVPF/AVPF profile
media_encryption = no           ; no | sdes | dtls
media_encryption_optimistic = no
media_use_received_transport = no

transport = transport-udp       ; pin to specific transport (rare)
outbound_proxy = sip:proxy.example.com:5060\;lr

mailboxes = 1001@default        ; voicemail MWI subscription
moh_suggest = default           ; music-on-hold class
language = en
tone_zone = us

call_group = 1
pickup_group = 1
named_call_group = office
named_pickup_group = office

allow_subscribe = yes           ; SIP SUBSCRIBE for BLF/MWI
sub_min_expiry = 60
device_state_busy_at = 1        ; appear busy after this many calls

record_on_feature = automon
record_off_feature = automon

t38_udptl = no                  ; faxing
t38_udptl_ec = redundancy
t38_udptl_maxdatagram = 0
fax_detect = no

100rel = yes                    ; reliable provisional responses (PRACK)
timers = yes                    ; SIP session timers (RFC 4028)
timers_min_se = 90
timers_sess_expires = 1800

allow_transfer = yes
inband_progress = no

webrtc = yes                    ; shorthand: enables ice_support, use_avpf, media_encryption=dtls,
                                ; media_use_received_transport=yes, rtcp_mux=yes, dtls settings
```

## PJSIP Registrations

Outbound registrations (Asterisk registering itself to an upstream provider):

```ini
;========== register to ITSP ==========
[itsp-reg]
type = registration
transport = transport-udp
outbound_auth = itsp-auth
server_uri = sip:sip.itsp.example.com
client_uri = sip:asterisk@sip.itsp.example.com
contact_user = asterisk
retry_interval = 60
forbidden_retry_interval = 600
expiration = 3600
max_retries = 10000
auth_rejection_permanent = no
line = yes                      ; identify inbound on this registration
endpoint = itsp                 ; route inbound to this endpoint

[itsp-auth]
type = auth
auth_type = userpass
username = ACCOUNT_SID
password = AUTH_TOKEN
```

CLI:

```
pjsip show registrations
pjsip send register itsp-reg
pjsip send unregister itsp-reg
```

Inbound registrations (devices registering to Asterisk) just need `aors` + `auth` on the endpoint — see the 1001 example above.

## PJSIP Trunks

A "trunk" is just an endpoint plus optionally an outbound registration plus an aor with a static URI.

```ini
;========== outbound trunk to ITSP (registration-based) ==========
[itsp]
type = endpoint
transport = transport-udp
context = from-itsp
disallow = all
allow = ulaw,alaw
outbound_auth = itsp-auth
aors = itsp
from_user = ACCOUNT_SID
from_domain = sip.itsp.example.com
direct_media = no
rtp_symmetric = yes
force_rport = yes
rewrite_contact = yes
identify_by = username,auth_username

[itsp]
type = aor
contact = sip:sip.itsp.example.com:5060
qualify_frequency = 60

[itsp]
type = identify              ; match inbound by source IP (no auth)
endpoint = itsp
match = 54.172.60.0/23       ; CIDR ranges of provider

;========== outbound trunk to ITSP (IP-auth, no registration) ==========
[carrier-iptrunk]
type = endpoint
transport = transport-udp
context = from-carrier
disallow = all
allow = ulaw,g729
aors = carrier-iptrunk
direct_media = no
rtp_symmetric = yes

[carrier-iptrunk]
type = aor
contact = sip:carrier.example.com:5060
qualify_frequency = 60

[carrier-iptrunk]
type = identify
endpoint = carrier-iptrunk
match = 198.51.100.0/24
```

In dialplan, dial through a trunk:

```ini
exten => _1NXXNXXXXXX,1,Dial(PJSIP/${EXTEN}@itsp,30,t)
```

## extensions.conf — Dialplan

Asterisk's dialplan is procedural: contexts contain extensions, extensions contain priorities, each priority runs an application.

```ini
[general]
static = yes
writeprotect = no
clearglobalvars = no

[globals]
TRUNK = PJSIP/itsp
RECORD_DIR = /var/spool/asterisk/monitor

[default]
exten => s,1,NoOp(Default catch-all)
 same => n,Hangup()

[from-itsp]
; DID 5551234 lands on extension 1001
exten => 5551234,1,NoOp(Inbound from ITSP for ${EXTEN})
 same => n,Set(CALLERID(name)=External)
 same => n,Dial(PJSIP/1001,30,tT)
 same => n,Voicemail(1001@default,u)
 same => n,Hangup()

[internal]
include => parkedcalls
include => features

; Local extensions 1000-1099
exten => _10XX,1,NoOp(Internal call to ${EXTEN})
 same => n,Set(CALLERID(num)=${CALLERID(num)})
 same => n,Dial(PJSIP/${EXTEN},20,tT)
 same => n,GotoIf($["${DIALSTATUS}" = "BUSY"]?busy:unavail)
 same => n(busy),Voicemail(${EXTEN}@default,b)
 same => n,Hangup()
 same => n(unavail),Voicemail(${EXTEN}@default,u)
 same => n,Hangup()

; Outbound — strip 9 prefix, dial through trunk
exten => _9NXXNXXXXXX,1,NoOp(Outbound 10-digit)
 same => n,Set(CALLERID(num)=5550100)
 same => n,Dial(${TRUNK}/${EXTEN:1},60,tT)
 same => n,Hangup()

exten => _91NXXNXXXXXX,1,NoOp(Outbound 11-digit)
 same => n,Dial(${TRUNK}/${EXTEN:1},60,tT)
 same => n,Hangup()

; Voicemail access
exten => *97,1,VoiceMailMain(${CALLERID(num)}@default)
 same => n,Hangup()

; Echo test
exten => *43,1,Answer()
 same => n,Echo()
 same => n,Hangup()
```

Syntax:

```
exten => name,priority,Application(args)
 same => n,Application(args)        ; n = "next" priority

; Labels for Goto
exten => 100,1,Answer()
 same => n(start),Playback(welcome)
 same => n,Goto(start)

; Hints (for BLF presence)
exten => 1001,hint,PJSIP/1001
exten => 1001,hint,PJSIP/1001&PJSIP/1001-mobile

; Special extensions
; s     — start (when no extension is provided, e.g. analog FXS)
; i     — invalid extension entered
; t     — timeout waiting for digits
; h     — hangup handler (run on call termination)
; T     — absolute timeout reached
; o     — operator (zero out from voicemail)
; a     — voicemail "*" key
```

## Pattern Matching (X / Z / N / . / !)

Patterns start with `_`. Without `_` it's a literal.

| Token | Matches |
|---|---|
| `X` | digit 0–9 |
| `Z` | digit 1–9 |
| `N` | digit 2–9 |
| `[abc]` | one of a, b, c |
| `[1-5]` | range |
| `.` | one or more of any character |
| `!` | zero or more (early-match for Goto/Macro) |

```
_NXXXXXX        7-digit, area-code-less
_NXXNXXXXXX     10-digit NANP
_1NXXNXXXXXX    11-digit NANP with 1+
_011.           international (eats anything after 011)
_X.             at least one digit
_*XX            star code
_+1NXXNXXXXXX   E.164 NANP
```

Variable substring/manipulation works on `${EXTEN}`:

```
${EXTEN}           full extension
${EXTEN:1}         strip first character (e.g. drop "9" prefix)
${EXTEN:-4}        last 4 chars
${EXTEN:1:3}       chars 1-3
${EXTEN:0:-2}      drop last 2 chars
```

## Variables (${EXTEN}, ${CALLERID}, ${CHANNEL})

Variable scopes: channel (`${VAR}`), global (`${GLOBAL(VAR)}`), shared (`${SHARED(VAR,CHAN)}`), and AstDB (`${DB(family/key)}`).

```ini
exten => 100,1,Set(MYVAR=hello)            ; channel scope
 same => n,Set(GLOBAL(GVAR)=globalvalue)   ; global
 same => n,NoOp(${MYVAR})
 same => n,NoOp(${GLOBAL(GVAR)})
 same => n,Set(__INHERIT=yes)              ; double underscore = inherited to child channels
 same => n,Set(_TEMPHERIT=yes)             ; single underscore = inherited 1 level
```

Common built-ins:

```
${EXTEN}                    extension being dialed
${EXTEN:1}                  strip first digit
${CONTEXT}                  current context
${PRIORITY}                 current priority
${CHANNEL}                  channel name (PJSIP/1001-00000005)
${CHANNEL(name)}            base name
${UNIQUEID}                 channel unique ID
${LINKEDID}                 originating channel ID
${CALLERID(all)}            "Alice" <1001>
${CALLERID(name)}           Alice
${CALLERID(num)}            1001
${CALLERID(rdnis)}          redirecting DNIS
${CALLERID(dnid)}           dialed number
${CDR(billsec)}             billable seconds
${CDR(duration)}            duration
${CDR(disposition)}         ANSWERED|NO ANSWER|BUSY|FAILED
${DIALSTATUS}               result of last Dial()
${HANGUPCAUSE}              Q.850 cause code
${ANSWEREDTIME}
${EPOCH}                    unix time
${DATETIME}                 YYYY-MM-DD HH:MM:SS
${RAND(min,max)}            random int
${LEN(string)}              string length
${SHELL(command)}           run shell, return stdout (live_dangerously must be yes)
${DB(family/key)}           AstDB read
${IF($[1=1]?yes:no)}        ternary
${TIMEOUT(absolute)}        seconds
${PJSIP_HEADER(read,From)}  read inbound SIP header
${PJSIP_HEADER(add,X-Tag,foo)}  add header for outbound
${SIP_HEADER(From)}         legacy chan_sip
${HASH(myhash,key)}         associative array
${ARG1} ${ARG2}             Gosub/Macro arguments
${VOICEMAIL_INFO(fullname,1001)}
```

## Common Applications

```
Answer([delay])
Hangup([cause])
Dial(channel,timeout,options)
Playback(filename[,options])      ; play file, ignore DTMF
Background(filename)              ; play file, listen for DTMF
WaitExten([timeout[,options]])    ; wait for DTMF, jump to matched extension
Read(var,filename,maxdigits,options,attempts,timeout)
ReadExten(var,filename,context,options,timeout)
Goto([context,][exten,]priority)
GotoIf(condition?true-target:false-target)
GotoIfTime(times,weekdays,mdays,months[,timezone]?label)
ExecIf(condition?app(args):app(args))
Macro(name,args)                  ; deprecated, use Gosub
Gosub([context,][exten,]priority(args))
Return([retval])
Set(var=value)
Verbose(level,message)
NoOp(message)                     ; for log breadcrumbs
SayDigits(digits)
SayNumber(number)
SayAlpha(string)
Voicemail(box[,options])          ; leave message; options: u=unavail, b=busy, s=skip greeting
VoicemailMain([box[,options]])    ; check messages
Queue(queue[,options[,url[,announce[,timeout[,agi[,macro[,gosub[,rule[,position]]]]]]]]])
MeetMe(conference[,options[,pin]])    ; deprecated
ConfBridge(conf[,bridge_profile[,user_profile[,menu]]])
Record(filename,silence,maxduration,options)
Page(devices,options,timeout)
Originate(channel,exten|app,context|appdata,priority|appdata,timeout,options)
MixMonitor(filename[,options[,command]])  ; modern recording
Monitor([format[,fname-base[,options]]])  ; legacy recording
Wait(seconds)
WaitForSilence(silence,iterations,timeout)
Bridge(channel,options)
Park([parkinglot])
ParkedCall(parkinglot,space)
Pickup(extension@context)
ChanSpy([channel-prefix],options)
Authenticate(password,options)
Directory(context[,dialcontext[,options]])
Transfer(target)
SendDTMF(digits[,timeout])
DBput(family/key=value)           ; AstDB write
DBget(var=family/key)
DBdel(family/key)
System(command)                   ; run shell — needs live_dangerously
TrySystem(command)
AGI(script[,args])
EAGI(script[,args])
StasisApp(name,args)
```

## Dial() Application

Workhorse application. Channel syntax: `TECH/peer[/extension]`. Multiple destinations comma-separated within ampersands inside the channel arg.

```
Dial(channel[&channel2&...],timeout,options[,URL])

Dial(PJSIP/1001,30,tT)
Dial(PJSIP/1001&PJSIP/1001-mobile,30,tT)        ; ring both
Dial(PJSIP/${EXTEN}@itsp,60,tT)                 ; through trunk
Dial(Local/1001@internal,30)                    ; local channel re-enters dialplan
Dial(DAHDI/g0/${EXTEN},,T)                      ; PRI group
```

Most-used options:

| Opt | Effect |
|---|---|
| `t` | callee can transfer (#) |
| `T` | caller can transfer (#) |
| `h` | callee can hangup (*) |
| `H` | caller can hangup (*) |
| `m` | play MOH instead of ringback |
| `r` | generate ringback locally |
| `R` | indicate ringing while remote rings |
| `A(file)` | play `file` to callee on answer |
| `b(ctx^ext^pri)` | run pre-bridge subroutine on callee channel |
| `B(ctx^ext^pri)` | run pre-bridge subroutine on caller channel |
| `U(sub)` | run Gosub subroutine on connect |
| `M(macro)` | run macro on connect (deprecated) |
| `g` | go to next priority on hangup (don't end call) |
| `G(ctx^ext^pri)` | go to context/exten/priority for both legs on answer |
| `L(x[:y[:z]])` | hangup after x ms, warn at y ms, then every z ms |
| `D(dtmf)` | send DTMF on answer |
| `F(ctx^ext^pri)` | continue at this priority on caller hangup |
| `i` | ignore forwarded calls |
| `j` | jump to n+101 priority on busy/cong (legacy) |
| `o` | preserve original CallerID |
| `S(x)` | hangup x seconds after answer |
| `x` | allow callee to start MixMonitor recording |
| `X` | allow caller to start MixMonitor recording |
| `c` | reset CDR after each call attempt |
| `C` | reset CDR after this call |
| `e` | execute h-extension on hangup |
| `n` | don't sound a ringing tone |
| `p` | privacy mode |
| `Q(cause)` | hang up with specific cause if no one answers |
| `z` | use the secondary calling presentation |

`${DIALSTATUS}` after Dial() returns:

```
ANSWER       ; remote answered
BUSY         ; busy signal
NOANSWER     ; rang but timed out
CANCEL       ; caller hung up
CONGESTION   ; circuits busy / network failure
CHANUNAVAIL  ; channel unavailable (peer offline)
DONTCALL     ; call screening rejected
TORTURE      ; call screening rejected (torture)
INVALIDARGS  ; bad Dial() args
```

```ini
exten => _10XX,1,Dial(PJSIP/${EXTEN},20,tT)
 same => n,GotoIf($["${DIALSTATUS}" = "BUSY"]?busy)
 same => n,GotoIf($["${DIALSTATUS}" = "NOANSWER"]?noans)
 same => n,Hangup()
 same => n(busy),Voicemail(${EXTEN}@default,b)
 same => n,Hangup()
 same => n(noans),Voicemail(${EXTEN}@default,u)
 same => n,Hangup()
```

## Voicemail

`voicemail.conf` defines mailboxes per context. Storage on disk under `/var/spool/asterisk/voicemail/<context>/<box>/`.

```ini
[general]
format = wav49|gsm|wav
serveremail = asterisk@example.com
attach = yes
maxmsg = 100
maxsecs = 180
minsecs = 3
maxgreet = 60
skipms = 3000
maxsilence = 10
silencethreshold = 128
maxlogins = 3
moveheard = yes
forward_urgent_auto = no
emaildateformat = %A, %B %d, %Y at %r
mailcmd = /usr/sbin/sendmail -t
emailbody = Hello ${VM_NAME},\n\nNew voicemail (${VM_DUR}s) from ${VM_CALLERID}.
sendvoicemail = yes
operator = yes
review = yes

[default]
1001 => 1234,Alice Smith,alice@example.com,,attach=yes|tz=eastern
1002 => 4321,Bob Jones,bob@example.com

[zonemessages]
eastern = America/New_York|'vm-received' Q 'digits/at' IMp
```

Dialplan:

```ini
exten => _10XX,1,Dial(PJSIP/${EXTEN},20,tT)
 same => n,Voicemail(${EXTEN}@default,u)        ; u = unavailable greeting
 same => n,Hangup()

exten => *97,1,VoiceMailMain(${CALLERID(num)}@default)  ; my mailbox
 same => n,Hangup()

exten => *98,1,VoiceMailMain(@default)          ; prompt for box
 same => n,Hangup()
```

Voicemail() options:

```
u  unavailable greeting
b  busy greeting
s  skip greeting (jump straight to "leave a message" beep)
g(file)  prepend file before greeting
U  urgent only
P  priority only
d(c)  custom DTMF for ops menu
```

CLI:

```
voicemail show users
voicemail show users for default
voicemail reload
```

## Queues

`queues.conf` — agent queues, ring strategies, call distribution.

```ini
[general]
persistentmembers = yes
autofill = yes
monitor-type = MixMonitor
shared_lastcall = yes
log_membername_as_agent = no

[sales]
strategy = ringall
; rrmemory | leastrecent | fewestcalls | random | roundrobin | linear | wrandom
musicclass = default
announce = queue-thereare
context = qoutcontext
timeout = 15
retry = 5
weight = 0
wrapuptime = 10
maxlen = 0
servicelevel = 60
joinempty = yes
leavewhenempty = no
ringinuse = no
reportholdtime = no
memberdelay = 0
timeoutpriority = app
timeoutrestart = no
periodic-announce = queue-periodic-announce
periodic-announce-frequency = 60
announce-frequency = 90
announce-holdtime = once
announce-position = yes
announce-round-seconds = 10
random-periodic-announce = no

; Static members (penalty 0 = highest priority)
member => PJSIP/1001,0,Alice,hint:1001@internal
member => PJSIP/1002,0,Bob,hint:1002@internal
member => PJSIP/1003,1,Carol,hint:1003@internal
```

Dialplan:

```ini
exten => 5000,1,Answer()
 same => n,Queue(sales,tT,,,180)
 same => n,Voicemail(5000@default,s)
 same => n,Hangup()
```

CLI:

```
queue show
queue show sales
queue add member PJSIP/1004 to sales penalty 0
queue remove member PJSIP/1004 from sales
queue pause member PJSIP/1004 queue sales reason "lunch"
queue unpause member PJSIP/1004 queue sales
queue reload all
queue reset stats sales
queue log dump
```

## ConfBridge

Modern conference bridge (replaces deprecated MeetMe). Configured in `confbridge.conf`.

```ini
;========== confbridge.conf ==========
[general]

[default_bridge](!)
type = bridge
max_members = 50
record_conference = no
mixing_interval = 20
internal_sample_rate = 16000
language = en
video_mode = none

[default_user](!)
type = user
quiet = no
announce_user_count = no
announce_user_count_all = no
announce_only_user = yes
wait_marked = no
end_marked = no
talk_detection_events = no
dtmf_passthrough = no
denoise = yes
jitterbuffer = yes
music_on_hold_when_empty = yes
music_on_hold_class = default

[admin_user](default_user)
admin = yes
end_marked = yes
marked = yes

[main_menu]
type = menu
1 = toggle_mute
4 = decrease_listening_volume
6 = increase_listening_volume
7 = decrease_talking_volume
9 = increase_talking_volume
* = playback_and_continue(conf-usermenu)
0 = admin_toggle_conference_locked
```

Dialplan:

```ini
exten => 8888,1,Answer()
 same => n,ConfBridge(${EXTEN},default_bridge,default_user,main_menu)
 same => n,Hangup()

; Admin entry with PIN
exten => 8889,1,Answer()
 same => n,Authenticate(1234)
 same => n,ConfBridge(8888,default_bridge,admin_user,main_menu)
 same => n,Hangup()
```

CLI:

```
confbridge list
confbridge list 8888
confbridge kick 8888 PJSIP/1001-00000007
confbridge mute 8888 PJSIP/1001-00000007
confbridge unmute 8888 PJSIP/1001-00000007
confbridge lock 8888
confbridge unlock 8888
confbridge record start 8888
confbridge record stop 8888
```

## AGI (sync + FastAGI)

AGI = Asterisk Gateway Interface. External script speaks line-protocol over stdin/stdout (sync) or TCP (FastAGI).

Dialplan:

```ini
exten => 100,1,Answer()
 same => n,AGI(myscript.py,arg1,arg2)
 same => n,AGI(agi://192.0.2.10:4573/myscript)   ; FastAGI
 same => n,EAGI(myscript.py)                     ; enhanced AGI (stream audio over fd 3)
 same => n,Hangup()
```

Sync AGI script (`/var/lib/asterisk/agi-bin/myscript.py`):

```python
#!/usr/bin/env python3
import sys

# Read environment
env = {}
while True:
    line = sys.stdin.readline().strip()
    if not line:
        break
    k, _, v = line.partition(":")
    env[k.strip()] = v.strip()

def cmd(c):
    sys.stdout.write(c + "\n")
    sys.stdout.flush()
    return sys.stdin.readline().strip()

cmd("ANSWER")
cmd('STREAM FILE "hello-world" ""')
cmd('SAY DIGITS 12345 ""')
cmd("SET VARIABLE FOOBAR baz")
cmd('VERBOSE "Got callerid: %s" 1' % env.get("agi_callerid"))
cmd("HANGUP")
```

Mark executable, owned by `asterisk:asterisk`, 0750.

FastAGI server skeleton (Python):

```python
import socketserver

class Handler(socketserver.StreamRequestHandler):
    def handle(self):
        env = {}
        for raw in self.rfile:
            line = raw.decode().strip()
            if not line: break
            k, _, v = line.partition(":")
            env[k.strip()] = v.strip()
        self._cmd("ANSWER")
        self._cmd('STREAM FILE "hello-world" ""')
        self._cmd("HANGUP")

    def _cmd(self, c):
        self.wfile.write((c + "\n").encode())
        self.wfile.flush()
        return self.rfile.readline().decode().strip()

with socketserver.ThreadingTCPServer(("0.0.0.0", 4573), Handler) as srv:
    srv.serve_forever()
```

Common AGI commands:

```
ANSWER
HANGUP [channel]
STREAM FILE filename "escape-digits"
EXEC application "args"
SET VARIABLE name value
GET VARIABLE name
GET FULL VARIABLE ${expr}
SAY DIGITS digits "escape-digits"
SAY NUMBER number "escape-digits"
WAIT FOR DIGIT timeout
GET DATA filename timeout maxdigits
RECORD FILE filename format escape-digits timeout offset BEEP s=silence
NOOP
VERBOSE "msg" level
SET CONTEXT context
SET EXTENSION exten
SET PRIORITY priority
DATABASE GET family key
DATABASE PUT family key value
DATABASE DEL family key
```

## AMI (manager.conf)

AMI = Asterisk Manager Interface — TCP control plane (port 5038), text protocol over single connection.

```ini
;========== manager.conf ==========
[general]
enabled = yes
port = 5038
bindaddr = 127.0.0.1
displayconnects = yes
allowmultiplelogin = yes
webenabled = no
httptimeout = 60

[admin]
secret = ChangeMeNow
deny = 0.0.0.0/0.0.0.0
permit = 127.0.0.1/255.255.255.255
permit = 10.0.0.0/255.0.0.0
read = system,call,log,verbose,command,agent,user,config,dtmf,reporting,cdr,dialplan,originate,security
write = system,call,agent,user,config,command,reporting,originate
writetimeout = 5000
```

Wire format (TCP, port 5038, CRLF-terminated lines, blank line ends an action):

```
Action: Login
Username: admin
Secret: ChangeMeNow

Action: Originate
Channel: PJSIP/1001
Context: internal
Exten: 100
Priority: 1
CallerID: "Auto" <0000>
Timeout: 30000
ActionID: 12345

Action: Hangup
Channel: PJSIP/1001-00000007
ActionID: 67890

Action: Status
ActionID: 11111

Action: Logoff
```

Quick test:

```bash
{ echo "Action: Login"; echo "Username: admin"; echo "Secret: ChangeMeNow"; echo; sleep 1; \
  echo "Action: Ping"; echo; sleep 1; \
  echo "Action: Logoff"; echo; sleep 1; } | nc 127.0.0.1 5038
```

CLI:

```
manager show users
manager show user admin
manager show connected
manager show eventq
manager set debug on
manager reload
```

## ARI (the modern REST replacement)

ARI = Asterisk REST Interface. JSON-over-HTTP for control + WebSocket for events. Replaces AMI for new development. The dialplan hands a channel to a "Stasis app" using `Stasis(myapp,arg1)` and the external app drives it via REST.

```ini
;========== http.conf ==========
[general]
enabled = yes
bindaddr = 0.0.0.0
bindport = 8088
prefix =
sessionlimit = 100
session_inactivity = 30000
tlsenable = yes
tlsbindaddr = 0.0.0.0:8089
tlscertfile = /etc/asterisk/keys/asterisk.crt
tlsprivatekey = /etc/asterisk/keys/asterisk.key

;========== ari.conf ==========
[general]
enabled = yes
pretty = yes
allowed_origins = *
auth_realm = Asterisk REST Interface

[ariuser]
type = user
read_only = no
password = ChangeMeNow
password_format = plain
```

REST surface:

```
GET    /ari/api-docs/resources.json
GET    /ari/asterisk/info
GET    /ari/channels
POST   /ari/channels                        ; originate
GET    /ari/channels/{id}
DELETE /ari/channels/{id}                   ; hangup
POST   /ari/channels/{id}/answer
POST   /ari/channels/{id}/play
POST   /ari/channels/{id}/record
POST   /ari/channels/{id}/dial
POST   /ari/channels/{id}/continue          ; back to dialplan
GET    /ari/bridges
POST   /ari/bridges
POST   /ari/bridges/{id}/addChannel
POST   /ari/bridges/{id}/removeChannel
GET    /ari/applications
GET    /ari/sounds
GET    /ari/endpoints
POST   /ari/events/user/{eventName}
```

WebSocket:

```
wss://host:8089/ari/events?api_key=ariuser:ChangeMeNow&app=myapp&subscribeAll=true
```

Dialplan handoff:

```ini
[from-itsp]
exten => _X.,1,NoOp(Sending to Stasis)
 same => n,Stasis(myapp,${EXTEN},${CALLERID(num)})
 same => n,Hangup()
```

curl example:

```bash
# List channels
curl -u ariuser:ChangeMeNow http://localhost:8088/ari/channels

# Originate to extension 1001 then hand to Stasis app
curl -u ariuser:ChangeMeNow -X POST \
  "http://localhost:8088/ari/channels?endpoint=PJSIP/1001&app=myapp&callerId=Auto"

# Hangup
curl -u ariuser:ChangeMeNow -X DELETE \
  "http://localhost:8088/ari/channels/1685647832.5"
```

## Stasis Application

Stasis is the dialplan hook into ARI. When `Stasis(myapp,...)` runs, the channel is routed to whichever process has subscribed to `app=myapp` over the ARI WebSocket.

Node.js client (ari-client):

```javascript
const ari = require('ari-client');
ari.connect('http://localhost:8088', 'ariuser', 'ChangeMeNow', (err, client) => {
  if (err) throw err;
  client.on('StasisStart', async (event, channel) => {
    console.log('Channel entered Stasis:', channel.id);
    await channel.answer();
    await channel.play({media: 'sound:hello-world'});
    setTimeout(() => channel.hangup(), 5000);
  });
  client.on('StasisEnd', (event, channel) => {
    console.log('Channel left Stasis:', channel.id);
  });
  client.start('myapp');
});
```

## Codecs

`disallow = all` then `allow = X` in order of preference. Codec selection happens in SDP negotiation.

| Codec | Bandwidth | Notes |
|---|---|---|
| `ulaw` (G.711 µ-law) | 64 kbps | NA standard, no license |
| `alaw` (G.711 A-law) | 64 kbps | EU standard |
| `gsm` | 13 kbps | low-fi |
| `g722` | 64 kbps | wideband 16kHz |
| `g729` | 8 kbps | needs license/passthrough |
| `g726` | 16/24/32/40 kbps | toll-quality |
| `opus` | 6–510 kbps | requires `codec_opus.so` |
| `silk` | variable | requires `codec_silk.so` |
| `slin` / `slin16` / `slin48` | n/a | internal signed-linear, transcoding hub |

```ini
[endpoint-internal](!)
type = endpoint
disallow = all
allow = ulaw
allow = alaw
allow = g722
; opus only for WebRTC peers
```

CLI:

```
core show codecs
core show codec ulaw
core show translation
```

Tip: never run `disallow = all` then nothing — endpoint won't negotiate any codec and calls fail with `Cannot create translator path`.

## RTP

`rtp.conf` controls the RTP port range and quality knobs. Default range 10000–20000.

```ini
[general]
rtpstart = 10000
rtpend = 20000
rtpchecksums = no
dtmftimeout = 3000
rtcpinterval = 5000
strictrtp = yes              ; drop unexpected RTP source IPs (security)
probation = 4
icesupport = yes
stunaddr = stun:stun.l.google.com:19302
turnaddr = turn:turn.example.com:3478
turnusername = user
turnpassword = pass
```

Open ports `10000-20000/udp` on firewall + UPnP if NAT.

CLI:

```
rtp set debug on
rtp set debug ip 192.0.2.50
rtcp set debug on
rtcp set stats on
core show channelvars
```

## NAT Traversal

The single most common source of one-way audio. Everything below is mandatory if Asterisk is behind NAT.

```ini
[transport-udp]
type = transport
protocol = udp
bind = 0.0.0.0:5060
external_media_address = 203.0.113.10        ; public IP for SDP rewrite
external_signaling_address = 203.0.113.10    ; public IP for Via/Contact rewrite
local_net = 192.168.0.0/16
local_net = 10.0.0.0/8
local_net = 172.16.0.0/12

[endpoint-internal](!)
force_rport = yes              ; respond to source port not Via
rewrite_contact = yes          ; replace Contact: with source IP:port
rtp_symmetric = yes            ; send RTP back where we got it
direct_media = no              ; anchor RTP at Asterisk
disable_direct_media_on_nat = yes
ice_support = yes              ; STUN/TURN candidates (WebRTC + smart UAs)
```

DNAT/SNAT on the firewall must NOT alter SIP payload; if it does, disable any "SIP ALG" feature on the router — these break Asterisk in 99% of cases.

Diagnostic:

```bash
# Inbound INVITE — does Contact: have private IP?
sudo asterisk -rx "pjsip set logger on"
# Watch for "Contact: <sip:1001@10.0.0.50:5060>" — that's the device's local addr

# RTP one-way? Run tcpdump on Asterisk
sudo tcpdump -i any -n udp portrange 10000-20000
# If you see RTP outbound but nothing inbound -> NAT/firewall blocking
```

## TLS / SIP-over-TLS

Generate certs (Asterisk ships `ast_tls_cert` script):

```bash
cd /etc/asterisk/keys
sudo /var/lib/asterisk/scripts/ast_tls_cert -C asterisk.example.com -O "ExampleOrg" -d .
# produces ca.crt, asterisk.crt, asterisk.key, asterisk.pem

# Or use Let's Encrypt
sudo certbot certonly --standalone -d asterisk.example.com
sudo cat /etc/letsencrypt/live/asterisk.example.com/fullchain.pem \
        /etc/letsencrypt/live/asterisk.example.com/privkey.pem \
     | sudo tee /etc/asterisk/keys/asterisk.pem
sudo chown asterisk:asterisk /etc/asterisk/keys/asterisk.pem
sudo chmod 600 /etc/asterisk/keys/asterisk.pem
```

Transport:

```ini
[transport-tls]
type = transport
protocol = tls
bind = 0.0.0.0:5061
cert_file = /etc/asterisk/keys/asterisk.crt
priv_key_file = /etc/asterisk/keys/asterisk.key
ca_list_file = /etc/asterisk/keys/ca.crt
method = tlsv1_2
verify_client = no
verify_server = no
require_client_cert = no
allow_reload = no
cipher = ECDHE-RSA-AES256-GCM-SHA384:ECDHE-RSA-AES128-GCM-SHA256
```

Endpoint pinned to TLS:

```ini
[1001]
type = endpoint
transport = transport-tls
media_encryption = sdes        ; or dtls for WebRTC
```

## SRTP

Media encryption — `media_encryption = sdes` for normal phones, `dtls` for WebRTC.

```ini
[1001]
type = endpoint
transport = transport-tls
media_encryption = sdes
media_encryption_optimistic = no   ; require SRTP
```

For DTLS-SRTP (WebRTC):

```ini
[webrtc-endpoint](!)
type = endpoint
transport = transport-wss
context = internal
disallow = all
allow = opus,ulaw
webrtc = yes                       ; sets the rest below
; force_rport = yes
; rewrite_contact = yes
; rtp_symmetric = yes
; ice_support = yes
; use_avpf = yes
; media_encryption = dtls
; dtls_verify = fingerprint
; dtls_setup = actpass
; dtls_cert_file = /etc/asterisk/keys/asterisk.crt
; dtls_private_key = /etc/asterisk/keys/asterisk.key
; rtcp_mux = yes
```

## WebRTC

Browser-based SIP. Needs WSS transport, DTLS-SRTP media, opus codec, and ICE.

```ini
;========== pjsip.conf ==========
[transport-wss]
type = transport
protocol = wss
bind = 0.0.0.0:8089

[webrtc-endpoint](!)
type = endpoint
transport = transport-wss
context = internal
disallow = all
allow = opus
allow = ulaw
webrtc = yes
dtls_cert_file = /etc/asterisk/keys/asterisk.crt
dtls_private_key = /etc/asterisk/keys/asterisk.key
dtls_setup = actpass
dtls_verify = fingerprint

[6001](webrtc-endpoint)
auth = 6001
aors = 6001

[6001]
type = auth
auth_type = userpass
username = 6001
password = SuperSecret

[6001]
type = aor
max_contacts = 5
remove_existing = yes
```

Browser side (sip.js / JsSIP):

```javascript
const ua = new SIP.UA({
  uri: 'sip:6001@asterisk.example.com',
  authorizationUser: '6001',
  password: 'SuperSecret',
  transportOptions: {
    wsServers: ['wss://asterisk.example.com:8089/ws'],
    traceSip: true,
  },
});
ua.start();
```

## CDR

Call Detail Records — one row per call leg.

```ini
;========== cdr.conf ==========
[general]
enable = yes
unanswered = no
congestion = no
endbeforehexten = no
initiatedseconds = no
batch = no
size = 100
time = 300
scheduleronly = no
safeshutdown = yes
```

Default backend `cdr_csv` writes `/var/log/asterisk/cdr-csv/Master.csv`:

```
"accountcode","src","dst","dcontext","clid","channel","dstchannel","lastapp","lastdata","start","answer","end","duration","billsec","disposition","amaflags","uniqueid","userfield"
```

Other backends: `cdr_adaptive_odbc.conf`, `cdr_mysql.conf`, `cdr_pgsql.conf`, `cdr_radius.conf`, `cdr_manager.conf` (publish CDR over AMI).

CLI:

```
cdr show status
```

`${CDR(field)}` accessors in dialplan: `accountcode`, `src`, `dst`, `dcontext`, `clid`, `channel`, `dstchannel`, `lastapp`, `lastdata`, `start`, `answer`, `end`, `duration`, `billsec`, `disposition`, `amaflags`, `uniqueid`, `userfield`.

## CEL

Channel Event Logging — finer-grained than CDR, every channel event (CHAN_START, ANSWER, BRIDGE_ENTER, HANGUP, etc).

```ini
;========== cel.conf ==========
[general]
enable = yes
apps = dial,park,queue,confbridge
events = ALL
;events = CHAN_START,CHAN_END,ANSWER,HANGUP,BRIDGE_ENTER,BRIDGE_EXIT,USER_DEFINED

;========== cel_custom.conf ==========
[mappings]
Master.csv => ${CSV_QUOTE(${eventtype})},${CSV_QUOTE(${eventtime})},${CSV_QUOTE(${CALLERID(name)})},${CSV_QUOTE(${CALLERID(num)})},${CSV_QUOTE(${CALLERID(ANI)})},${CSV_QUOTE(${CALLERID(RDNIS)})},${CSV_QUOTE(${CALLERID(DNID)})},${CSV_QUOTE(${CHANNEL(exten)})},${CSV_QUOTE(${CHANNEL(context)})},${CSV_QUOTE(${CHANNEL(channame)})},${CSV_QUOTE(${CHANNEL(appname)})},${CSV_QUOTE(${CHANNEL(appdata)})},${CSV_QUOTE(${eventextra})},${CSV_QUOTE(${CHANNEL(uniqueid)})},${CSV_QUOTE(${CHANNEL(linkedid)})},${CSV_QUOTE(${BRIDGEPEER})},${CSV_QUOTE(${CHANNEL(accountcode)})}
```

## IAX2

Inter-Asterisk eXchange v2 — Asterisk-to-Asterisk single-port (4569/UDP) protocol with built-in trunking and authentication. Use for two Asterisks behind firewalls; for everything else, use SIP.

```ini
;========== iax.conf ==========
[general]
bindport = 4569
bindaddr = 0.0.0.0
delayreject = yes
disallow = all
allow = ulaw,alaw,g722
trunkfreq = 20
trunktimestamps = yes
language = en
encryption = aes128
forceencryption = no

[user-template](!)
type = friend
context = internal
disallow = all
allow = ulaw,alaw

[siteB](user-template)
auth = md5
secret = trunkSecret
host = dynamic
trunk = yes
qualify = yes
```

Dialplan dial: `Dial(IAX2/siteB/${EXTEN})`.

CLI:

```
iax2 show peers
iax2 show users
iax2 show registry
iax2 show channels
iax2 set debug on
```

## DAHDI / Analog / PRI

DAHDI = Digium/Sangoma Asterisk Hardware Device Interface — kernel drivers for analog/T1/E1/BRI cards.

```bash
# Detect/configure hardware
sudo dahdi_genconf
sudo dahdi_cfg -vvv
sudo dahdi_hardware
sudo dahdi_scan
sudo lsdahdi
sudo dahdi_test
sudo dahdi_monitor 1
```

`/etc/dahdi/system.conf`:

```
loadzone = us
defaultzone = us

# 1 FXS port (analog phone)
fxoks = 1
echocanceller = mg2,1

# 1 FXO port (PSTN line)
fxsks = 2
echocanceller = mg2,2

# T1 PRI (channels 1-23, channel 24 = D)
span = 1,1,0,esf,b8zs
bchan = 1-23
dchan = 24
echocanceller = mg2,1-23
```

`/etc/asterisk/chan_dahdi.conf`:

```ini
[trunkgroups]

[channels]
context = from-pstn
language = en
usecallerid = yes
hidecallerid = no
callwaiting = yes
threewaycalling = yes
echocancel = yes
echocancelwhenbridged = no
echotraining = 800
faxdetect = both
busydetect = yes
busycount = 4

;----- analog FXS -----
signalling = fxo_ks
group = 1
channel => 1

;----- analog FXO -----
signalling = fxs_ks
context = from-pstn
group = 2
channel => 2

;----- T1 PRI -----
signalling = pri_cpe
switchtype = national
pridialplan = unknown
prilocaldialplan = unknown
overlapdial = no
group = 0
context = from-pri
channel => 1-23
```

Dial through PRI:

```ini
exten => _9NXXNXXXXXX,1,Dial(DAHDI/g0/${EXTEN:1},60,T)
```

CLI:

```
dahdi show status
dahdi show channels
dahdi show channel 1
pri show spans
pri show span 1
pri set debug on span 1
```

## Common Errors (verbatim)

```
WARNING[12345]: chan_sip.c:23456 handle_request_register: Failed to authenticate device sip:1001@192.0.2.1
  -> bad password, bad username, or bad realm. Check 'sip set debug on' / 'pjsip set logger on'.

WARNING[12345]: res_pjsip/pjsip_distributor.c:712 log_failed_request: Request 'REGISTER' from '<sip:1001@x>' failed for '192.0.2.1:5060' (callid: 8e8...) - Failed to authenticate
  -> Same as above for PJSIP. password mismatch.

NOTICE[12345]: res_pjsip_registrar.c:891 registrar_on_rx_request: Endpoint '1001' has no configured AORs
  -> add `aors = 1001` to endpoint, and create `[1001] type=aor` section.

NOTICE[12345]: res_pjsip_endpoint_identifier_user.c:101 username_identify: Endpoint '1001' not found
  -> endpoint not defined, or `pjsip reload` not run, or username mismatch with `identify_by`.

NOTICE[12345]: res_pjsip_session.c: 4567 ast_sip_session_alloc: Failed to create session, dropping incoming call from <sip:x@y> to <sip:5551234@asterisk>
  -> no matching endpoint identification; check `identify_by` and `[type=identify]` blocks.

WARNING[12345]: res_pjsip_outbound_registration.c:601 sip_outbound_registration_response_cb: 401 Unauthorized in 'REGISTER' to 'sip:sip.itsp.example.com'. Got SIP response 401: Unauthorized
  -> ITSP rejecting your registration. wrong creds, wrong From URI, or wrong realm.

WARNING[12345]: chan_sip.c: Got SIP response 408 "Request Timeout" back from x.x.x.x
  -> SIP signalling reachable one-way. firewall, NAT ALG, or wrong port.

WARNING[12345]: res_pjsip_session.c: Got SIP response 503 "Service Unavailable" back from sip.itsp.example.com
  -> upstream provider unavailable, or your account suspended/over-quota.

WARNING[12345]: chan_pjsip.c: Could not create dialog to endpoint 'foo'. Trying again
  -> AOR has no contact, or all contacts qualify-fail.

WARNING[12345]: pbx.c: No application 'Diall' for extension (internal, _10XX, 1)
  -> typo in dialplan. Use `dialplan show internal` to verify.

WARNING[12345]: file.c:1234 ast_openstream_full: File hello-world does not exist in any format
  -> sound file missing. Check /var/lib/asterisk/sounds/ for the format actually installed.

ERROR[12345]: chan_pjsip.c: Endpoint: '1001' Contact: 'sip:1001@10.0.0.50:50123'  Could not create dialog to invalid contact.
  -> contact has private IP; need `rewrite_contact = yes` so Asterisk uses the source IP.

WARNING[12345]: res_pjsip.c: Outbound registration to 'sip:sip.itsp.example.com' has been rejected with response code 403
  -> 403 Forbidden — usually IP not whitelisted, account disabled, or invalid From-domain.

NOTICE[12345]: chan_dahdi.c: Couldn't open /dev/dahdi/channel: No such file or directory
  -> DAHDI module not loaded; run `sudo dahdi_cfg -vvv`.

WARNING[12345]: rtp_engine.c: No translator path exists for channel type alaw to slin
  -> needed translation module not loaded. Check `module show like codec`.

WARNING[12345]: pbx.c:6789 pbx_extension_helper: No application 'Stasis' for extension (default, 100, 1)
  -> res_stasis.so not loaded. `module load res_stasis.so`.

WARNING[12345]: app_dial.c: Unable to create channel of type 'PJSIP' (cause 3 - No route to destination)
  -> endpoint has no reachable contact. `pjsip show contacts <endpoint>`.

NOTICE[12345]: res_pjsip_session.c: Call from '1001' (192.0.2.1:5060) to extension '5551234' rejected because extension not found in context 'internal'.
  -> dialplan missing pattern. Add `_NXXNXXXXXX` etc to that context.

WARNING[12345]: chan_pjsip.c: Couldn't allocate codecs for channel
  -> endpoint disallow=all but no allow=. Add at least one allow= line.

WARNING[12345]: res_pjsip_pubsub.c: No registered handler for event 'message-summary'
  -> SUBSCRIBE for MWI without `mailboxes=` on endpoint. Set `mailboxes = 1001@default`.

NOTICE[12345]: chan_sip.c: Registration from '<sip:1001@asterisk>' failed for '192.0.2.1:5060' - No matching peer found
  -> chan_sip can't find peer. Either define [1001] in sip.conf, or migrate to PJSIP.

WARNING[12345]: cdr.c: CDR already posted
  -> dialplan calls Hangup() after CDR submission. Usually benign.

WARNING[12345]: ast_coredumper: Asterisk has been killed by signal 11 (SIGSEGV)
  -> crash. Get core file (`live_dangerously` is unrelated). gdb for backtrace, file bug.

WARNING[12345]: app_voicemail.c: No more messages -- maximum messages reached
  -> mailbox at `maxmsg`. Increase in voicemail.conf or have user delete messages.

NOTICE[12345]: res_pjsip_session.c: Could not create dialog to invalid contact 'sip:1001@10.0.0.50:5060;rinstance=abc'
  -> contact contains rinstance with private IP — rewrite_contact didn't fire. Verify NAT block.
```

## Common Gotchas

Twelve broken-then-fixed pairs. Each is real and bites every Asterisk admin eventually.

**1. SIP ALG on the router rewriting headers**

```ini
; Symptom: registers OK then 401s, or one-way audio
; Broken:  router has "SIP ALG" / "SIP fixup" enabled
; Fixed:   disable SIP ALG on router AND set:
[transport-udp]
external_media_address = 203.0.113.10
external_signaling_address = 203.0.113.10
local_net = 192.168.0.0/16
```

**2. NAT settings missing on endpoint**

```ini
; Broken:
[1001]
type = endpoint
context = internal

; Fixed:
[1001]
type = endpoint
context = internal
force_rport = yes
rewrite_contact = yes
rtp_symmetric = yes
direct_media = no
```

**3. `disallow = all` without `allow =`**

```ini
; Broken (no codecs negotiated, call fails):
[1001](endpoint-base)
disallow = all
; ... no allow ...

; Fixed:
[1001](endpoint-base)
disallow = all
allow = ulaw
allow = alaw
```

**4. Outbound registration rejected with 401 forever**

```ini
; Broken: outbound_auth on endpoint instead of registration
[itsp]
type = endpoint
outbound_auth = itsp-auth      ; correct here
; but ALSO need:
[itsp-reg]
type = registration
outbound_auth = itsp-auth      ; <-- forgot this
```

**5. Dialplan reload not enough after pjsip.conf edit**

```bash
# Broken:
asterisk -rx "dialplan reload"        # only reloads extensions.conf

# Fixed:
asterisk -rx "pjsip reload"
asterisk -rx "module reload res_pjsip.so"
```

**6. `same =>` after `exten =>` requires same extension**

```ini
; Broken:
exten => 100,1,Answer()
exten => 101,2,Playback(hello)        ; runs as priority 2 of 101, not 100

; Fixed:
exten => 100,1,Answer()
 same => n,Playback(hello)
```

**7. Pattern match without leading underscore**

```ini
; Broken (literal _NXXNXXXXXX never matches):
exten => NXXNXXXXXX,1,Dial(...)

; Fixed:
exten => _NXXNXXXXXX,1,Dial(...)
```

**8. Voicemail not connecting because of MWI mismatch**

```ini
; Broken: phone subscribes, no MWI events sent
[1001]
type = endpoint
; mailboxes missing

; Fixed:
[1001]
type = endpoint
mailboxes = 1001@default      ; matches voicemail.conf [default] 1001 => ...
```

**9. Manager (AMI) login from remote IP fails despite correct password**

```ini
; Broken:
[admin]
secret = ChangeMeNow
; permit defaults to nothing

; Fixed:
[admin]
secret = ChangeMeNow
deny = 0.0.0.0/0.0.0.0
permit = 10.0.0.0/255.0.0.0
```

**10. ARI returns "Authentication required" on /events WebSocket**

```bash
# Broken:
wscat -c "ws://localhost:8088/ari/events?app=myapp"

# Fixed (api_key=user:pass):
wscat -c "ws://localhost:8088/ari/events?api_key=ariuser:ChangeMeNow&app=myapp"
```

**11. WebRTC fails — "DTLS handshake failed"**

```ini
; Broken: no DTLS material
[6001]
type = endpoint
webrtc = yes

; Fixed: webrtc=yes implies dtls_* but only if cert paths exist
[6001]
type = endpoint
webrtc = yes
dtls_cert_file = /etc/asterisk/keys/asterisk.crt
dtls_private_key = /etc/asterisk/keys/asterisk.key
dtls_setup = actpass
dtls_verify = fingerprint
```

**12. `Dial(PJSIP/1001)` when 1001 isn't registered**

```ini
; Broken: ${DIALSTATUS}=CHANUNAVAIL with no fallback
exten => 1001,1,Dial(PJSIP/1001,30)
 same => n,Hangup()

; Fixed: handle CHANUNAVAIL via voicemail
exten => _10XX,1,Dial(PJSIP/${EXTEN},30,tT)
 same => n,GotoIf($["${DIALSTATUS}" = "CHANUNAVAIL"]?vm)
 same => n,GotoIf($["${DIALSTATUS}" = "BUSY"]?vm)
 same => n,GotoIf($["${DIALSTATUS}" = "NOANSWER"]?vm)
 same => n,Hangup()
 same => n(vm),Voicemail(${EXTEN}@default,u)
 same => n,Hangup()
```

**13. extensions.conf comments using `#`**

```ini
; Broken (# is not a comment in INI):
# this is a comment

; Fixed:
; this is a comment
```

**14. Sounds in the wrong format**

```ini
; Broken:
exten => 100,1,Playback(welcome.gsm)         ; .gsm in arg confuses lookup

; Fixed:
exten => 100,1,Playback(welcome)             ; Asterisk picks best format
```

## Diagnostic Recipes

### One-way audio

```bash
# 1. Check NAT settings
sudo asterisk -rx "pjsip show endpoint 1001" | grep -iE 'force_rport|rewrite|rtp_sym|direct_media|external'

# 2. SIP trace
sudo asterisk -rx "pjsip set logger on"
# Look at the SDP `c=IN IP4 ...` and `m=audio PORT` lines:
#   - is the IP private (10.x, 192.168.x, 172.16-31.x)?
#   - is `external_media_address` set on the transport?

# 3. RTP
sudo asterisk -rvvv
*CLI> rtp set debug on
# Watch RTP frames in/out. If only outbound, NAT/firewall.

# 4. tcpdump
sudo tcpdump -nn -i any udp portrange 10000-20000

# 5. Common fix
echo '[transport-udp]
external_media_address = <PUBLIC_IP>
external_signaling_address = <PUBLIC_IP>
local_net = 10.0.0.0/8
local_net = 192.168.0.0/16' | sudo tee -a /etc/asterisk/pjsip.conf
sudo asterisk -rx "pjsip reload"
```

### Calls dropping mid-call

```bash
# 1. Session timers misconfig
sudo asterisk -rx "pjsip show endpoint 1001" | grep -i timer
# Set timers = yes; timers_min_se = 90; timers_sess_expires = 1800

# 2. Qualify failure (NAT keepalive lost)
sudo asterisk -rx "pjsip show contacts"
# State = Unreachable → contact disappeared. Increase qualify_frequency in AOR.

# 3. RTP timeout
grep -iE 'rtptimeout|rtpkeepalive' /etc/asterisk/pjsip.conf
# Default 0 = disabled. If set, RTP gap drops the call.

# 4. ITSP-side B2BUA disconnecting
sudo tail -f /var/log/asterisk/full | grep -i 'BYE\|hangup'
```

### Constant 401s on inbound registration

```bash
# Watch the SIP exchange
sudo asterisk -rx "pjsip set logger on"

# Look at REGISTER -> 401 -> REGISTER (with auth) -> 401 again
#   - First 401 is normal challenge
#   - Second 401 means the response is wrong

# Check:
asterisk -rx "pjsip show auth 1001"
# Verify username, auth_type=userpass, password set

# Common: username on the auth doesn't match the SIP From username
[1001]
type = auth
username = 1001          ; <- this MUST match REGISTER's From: <sip:1001@...>

# Realm mismatch (multi-domain ITSP)
realm = asterisk         ; default; ITSP may require its own
```

## Performance Tuning

```ini
;========== asterisk.conf ==========
[options]
maxcalls = 0                 ; 0 = unlimited
maxload = 0.0                ; reject if loadavg above this
hideconnect = yes
lockmode = lockfile          ; or 'flock'

;========== rtp.conf ==========
rtpstart = 10000
rtpend = 20000

;========== pjsip.conf — global ==========
[system]
type = system
threadpool_initial_size = 0
threadpool_auto_increment = 5
threadpool_idle_timeout = 60
threadpool_max_size = 0       ; 0 = unlimited
disable_tcp_switch = no
follow_early_media_fork = yes
accept_multiple_sdp_answers = no

[global]
type = global
max_forwards = 70
keep_alive_interval = 30
contact_expiration_check_interval = 30
disable_multi_domain = no
endpoint_identifier_order = ip,username,anonymous
default_realm = asterisk
mwi_tps_queue_high = 500
mwi_tps_queue_low = -1
unidentified_request_count = 5
unidentified_request_period = 5
unidentified_request_prune_interval = 30
default_outbound_endpoint = default_outbound_endpoint
```

OS tuning:

```bash
# Increase file descriptors
echo "asterisk soft nofile 65536" | sudo tee -a /etc/security/limits.conf
echo "asterisk hard nofile 65536" | sudo tee -a /etc/security/limits.conf

# UDP buffer
sudo sysctl -w net.core.rmem_max=16777216
sudo sysctl -w net.core.wmem_max=16777216

# Conntrack table (firewall)
sudo sysctl -w net.netfilter.nf_conntrack_max=524288
```

CLI:

```
core show taskprocessors      ; high "max depth" = backed up queue
core show taskprocessor pjsip/distributor
core show locks
core show channels count
core show sysinfo
```

## Security

```bash
# fail2ban — block brute-force REGISTERs
sudo apt-get install fail2ban
```

`/etc/fail2ban/jail.local`:

```ini
[asterisk]
enabled = true
filter = asterisk
action = iptables-allports[name=asterisk, protocol=all]
logpath = /var/log/asterisk/messages
maxretry = 5
findtime = 600
bantime = 86400
```

`/etc/fail2ban/filter.d/asterisk.conf`:

```ini
[Definition]
failregex = NOTICE.*<HOST>.*Registration from .* failed for
            NOTICE.*<HOST>.*failed to authenticate
            NOTICE.*<HOST>.*No matching endpoint found
            NOTICE.*<HOST>.*Request '\w+' from .* failed for
ignoreregex =
```

PJSIP security checklist:

```ini
; 1. allowguest = no  (block anonymous calls in dialplan via context)
; 2. Use [system] threadpool_max_size to cap exhaustion
; 3. Deny known-bad sources
[acl-default]
type = acl
deny = 0.0.0.0/0
permit = 10.0.0.0/8
permit = 192.168.0.0/16

[transport-udp]
type = transport
protocol = udp
bind = 0.0.0.0:5060

; 4. Require auth (no [type=identify] for unknown peers)
[global]
unidentified_request_count = 5
unidentified_request_period = 5
unidentified_request_prune_interval = 30

; 5. Disable Allowed Methods leakage
allow_subscribe = no   ; on endpoints that don't need MWI/BLF
```

AMI ACL (manager.conf):

```ini
[admin]
secret = LongRandomString
deny = 0.0.0.0/0.0.0.0
permit = 127.0.0.1/255.255.255.255
read = system,call,reporting,cdr,security
write = system,call,originate
```

TLS + SRTP for SIP everywhere external; never expose 5060/UDP to the internet without a SBC.

```bash
# iptables — only allow SIP from trusted sources
iptables -A INPUT -p udp --dport 5060 -s <SBC_IP>/32 -j ACCEPT
iptables -A INPUT -p udp --dport 5060 -j DROP
iptables -A INPUT -p tcp --dport 5061 -j ACCEPT
iptables -A INPUT -p udp --dport 10000:20000 -j ACCEPT
```

## AsteriskNOW / FreePBX

FreePBX = web GUI on top of Asterisk; AsteriskNOW = Linux distro with FreePBX preinstalled. Don't hand-edit `/etc/asterisk/*.conf` on a FreePBX system — files are regenerated from the GUI's MariaDB on every "Apply Config".

Custom dialplan goes in `extensions_custom.conf`, custom PJSIP in `pjsip.endpoint_custom_post.conf` and friends. Modules live under `/var/www/html/admin/modules/`.

Common FreePBX commands:

```bash
sudo fwconsole reload                # apply config
sudo fwconsole restart               # restart asterisk
sudo fwconsole stop / start
sudo fwconsole ma list               # modules
sudo fwconsole ma upgradeall
sudo fwconsole ma installall
sudo fwconsole chown                 # fix permissions
sudo asterisk -rvvvv
```

`/etc/asterisk/extensions_custom.conf`:

```ini
[from-internal-custom]
exten => *777,1,Answer()
 same => n,Playback(beep)
 same => n,Hangup()
```

## Idioms

```ini
; --- Time-of-day routing ---
exten => 5551234,1,GotoIfTime(09:00-17:00,mon-fri,*,*?open,1:closed,1)

[open]
exten => 1,1,Dial(PJSIP/1001&PJSIP/1002,20,tT)
 same => n,Voicemail(1001@default,u)
 same => n,Hangup()

[closed]
exten => 1,1,Playback(after-hours)
 same => n,Voicemail(1001@default,u)
 same => n,Hangup()

; --- IVR with tries-counter ---
[ivr-main]
exten => s,1,Set(TRIES=0)
 same => n(top),Background(welcome)
 same => n,WaitExten(5)
 same => n,Set(TRIES=$[${TRIES}+1])
 same => n,GotoIf($[${TRIES} >= 3]?goodbye)
 same => n,Goto(top)
 same => n(goodbye),Playback(goodbye)
 same => n,Hangup()

exten => 1,1,Goto(internal,1001,1)
exten => 2,1,Goto(internal,1002,1)
exten => 0,1,Goto(internal,operator,1)
exten => i,1,Playback(invalid)        ; invalid digit
 same => n,Goto(s,top)
exten => t,1,Playback(timeout)        ; timeout
 same => n,Goto(s,top)

; --- Hangup handler (cleanup, CDR) ---
exten => _X.,1,Set(__CHANHANGUPHANDLER=hangup-handler,s,1)
 same => n,Dial(PJSIP/1001,30,tT)
 same => n,Hangup()

[hangup-handler]
exten => s,1,NoOp(Channel hung up — disposition=${CDR(disposition)})
 same => n,Set(SHARED(LASTCALL,${CHANNEL(linkedid)})=ended)
 same => n,Return()

; --- Call recording ---
exten => _X.,1,MixMonitor(${UNIQUEID}.wav,b)
 same => n,Dial(PJSIP/${EXTEN},30,tT)
 same => n,Hangup()

; --- Dynamic queue agent login/logout ---
exten => *5,1,AddQueueMember(sales,PJSIP/${CALLERID(num)})
 same => n,Playback(agent-loginok)
 same => n,Hangup()

exten => *6,1,RemoveQueueMember(sales,PJSIP/${CALLERID(num)})
 same => n,Playback(agent-loggedoff)
 same => n,Hangup()

; --- Click-to-call originate via AMI ---
; Action: Originate
; Channel: PJSIP/1001
; Context: internal
; Exten: 5551234
; Priority: 1
; Async: yes

; --- Originate via dialplan (call file) ---
; /var/spool/asterisk/outgoing/calltest.call
;   Channel: PJSIP/1001
;   Context: internal
;   Extension: 5551234
;   Priority: 1
;   MaxRetries: 0
;   RetryTime: 60
;   WaitTime: 30
;   CallerID: "Auto" <0000>
; chmod 0640, chown asterisk:asterisk, mv (don't cp) into /var/spool/asterisk/outgoing/

; --- Local channel re-entering dialplan (useful for callbacks/queues) ---
exten => 100,1,Dial(Local/1001@internal/n)         ; /n = no optimization

; --- Detect fax and divert ---
exten => 5551234,1,Set(FAXOPT(faxdetect)=yes)
 same => n,Dial(PJSIP/1001,20)
 same => n,Hangup()

exten => fax,1,ReceiveFax(/var/spool/fax/${UNIQUEID}.tif)
 same => n,Hangup()

; --- BLF hint for desk phone ---
exten => 1001,hint,PJSIP/1001
exten => *81,hint,Custom:doorlock          ; custom device state

; --- Set custom device state ---
exten => *81,1,Set(DEVICE_STATE(Custom:doorlock)=BUSY)
 same => n,Wait(2)
 same => n,Set(DEVICE_STATE(Custom:doorlock)=NOT_INUSE)
 same => n,Hangup()

; --- Subroutine call (modern Macro replacement) ---
exten => _X.,1,Gosub(check-blacklist,s,1(${CALLERID(num)}))
 same => n,Dial(PJSIP/1001,30,tT)
 same => n,Hangup()

[check-blacklist]
exten => s,1,GotoIf($["${DB_EXISTS(blacklist/${ARG1})}" = "1"]?reject)
 same => n,Return()
 same => n(reject),Playback(blacklist-message)
 same => n,Hangup()
```

## See Also

- [sip-protocol](../protocols/sip-protocol.md) — SIP message structure, methods, response codes
- [rtp-sdp](../protocols/rtp-sdp.md) — RTP packets, SDP offer/answer
- [freeswitch](freeswitch.md) — alternative open-source telephony engine
- [ip-phone-provisioning](ip-phone-provisioning.md) — Yealink/Polycom/Cisco autoprovisioning
- [sip-trunking](sip-trunking.md) — ITSP integration patterns
- [tls](../security/tls.md) — TLS basics for SIP-over-TLS

## References

- Official wiki: https://wiki.asterisk.org/
- PJSIP Configuration: https://wiki.asterisk.org/wiki/display/AST/Asterisk+18+Configuration_res_pjsip
- Asterisk REST Interface: https://wiki.asterisk.org/wiki/display/AST/Asterisk+REST+Interface
- AMI Action reference: https://wiki.asterisk.org/wiki/display/AST/Asterisk+18+ManagerAction
- Dialplan Applications: https://wiki.asterisk.org/wiki/display/AST/Asterisk+18+Application_Dial
- Source: https://github.com/asterisk/asterisk
- Mailing lists: https://lists.digium.com/mailman/listinfo/asterisk-users
- RFC 3261 — SIP: https://www.rfc-editor.org/rfc/rfc3261
- RFC 4566 — SDP: https://www.rfc-editor.org/rfc/rfc4566
- RFC 4733 — RTP DTMF: https://www.rfc-editor.org/rfc/rfc4733
- RFC 3711 — SRTP: https://www.rfc-editor.org/rfc/rfc3711
- RFC 5763 — DTLS-SRTP: https://www.rfc-editor.org/rfc/rfc5763
- RFC 8825 — WebRTC overview: https://www.rfc-editor.org/rfc/rfc8825
- man pages: `man asterisk`, `man dahdi_cfg`, `man dahdi_genconf`
