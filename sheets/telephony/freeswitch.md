# FreeSWITCH

Software-defined telecom platform — SIP/RTP/WebRTC switch, B2BUA, IVR, conference, voicemail. C core + XML config + Lua/Python/JS scripting. Single binary handles thousands of concurrent calls.

## Setup

Install from official repo (Debian/Ubuntu) or build from source for current versions. The `signalwire-freeswitch` package supersedes `freeswitch-meta-all` in modern installs.

```bash
# Debian/Ubuntu (SignalWire repo)
TOKEN=pat_xxxx                                  # SignalWire access token
curl -sSL https://freeswitch.signalwire.com/repo/deb/debian-release/signalwire-freeswitch-repo.gpg | \
  sudo gpg --dearmor -o /usr/share/keyrings/signalwire-freeswitch-repo.gpg
echo "machine freeswitch.signalwire.com login signalwire password $TOKEN" | sudo tee /etc/apt/auth.conf.d/freeswitch.conf
echo "deb [signed-by=/usr/share/keyrings/signalwire-freeswitch-repo.gpg] https://freeswitch.signalwire.com/repo/deb/debian-release/ bookworm main" | sudo tee /etc/apt/sources.list.d/freeswitch.list

sudo apt update
sudo apt install -y freeswitch-meta-all                   # everything
sudo apt install -y freeswitch-meta-vanilla               # just core + sofia + dialplan + voicemail

# RPM (RHEL/Rocky/AlmaLinux 9)
sudo dnf install -y https://files.freeswitch.org/repo/yum/centos-release/freeswitch-release.noarch.rpm
sudo dnf install -y freeswitch-all

# From source (current master)
git clone https://github.com/signalwire/freeswitch.git -b v1.10
cd freeswitch
./bootstrap.sh -j                               # generate configure
./configure --prefix=/usr/local/freeswitch
make -j$(nproc)
sudo make install
sudo make cd-sounds-install cd-moh-install      # download sounds + MOH

# Service (systemd)
sudo systemctl enable --now freeswitch
sudo systemctl status freeswitch
sudo journalctl -u freeswitch -f -n 200

# Layout
/etc/freeswitch/                                # config (vanilla install)
/usr/share/freeswitch/sounds/                   # prompts (en/us/callie/8000|16000|32000|48000)
/var/lib/freeswitch/storage/                    # voicemail, recordings, db
/var/log/freeswitch/                            # freeswitch.log + cdr-csv

# First-run sanity
sudo -u freeswitch fs_cli -x 'version'          # should print build info
sudo -u freeswitch fs_cli -x 'status'           # uptime + sessions

# Linux limits — bump for production
ulimit -n 100000                                # file descriptors
echo 'freeswitch soft nofile 100000' | sudo tee -a /etc/security/limits.d/freeswitch.conf
echo 'freeswitch hard nofile 100000' | sudo tee -a /etc/security/limits.d/freeswitch.conf
```

## fs_cli

The CLI/REPL into a running FreeSWITCH. Talks to mod_event_socket (default port 8021, default password ClueCon — change in production). Supports inline `-x` for one-shot commands and prefix `/` for ESL API calls inside the REPL.

```bash
# Connect to local instance
fs_cli                                          # interactive REPL
fs_cli -p ClueCon                               # password
fs_cli -H 10.0.0.5 -P 8021 -p secret            # remote
fs_cli -x 'version'                             # one-shot, exits

# Logging level inside REPL (1=alert, 7=debug, 0=off)
freeswitch> /log 7
freeswitch> /log info
freeswitch> /nolog

# Core
fs_cli -x 'version'                             # SignalWire FreeSWITCH Version 1.10.x
fs_cli -x 'status'                              # uptime, current sessions, peak, max
fs_cli -x 'uptime'                              # ms since start
fs_cli -x 'shutdown'                            # graceful shutdown
fs_cli -x 'shutdown elegant'                    # wait for calls to drain
fs_cli -x 'shutdown asap'                       # next call ends → quit
fs_cli -x 'fsctl shutdown cancel'               # cancel pending shutdown
fs_cli -x 'fsctl reload mod_sofia'              # reload module
fs_cli -x 'fsctl loglevel debug'                # console loglevel

# Channels
fs_cli -x 'show channels'                       # active call list
fs_cli -x 'show channels count'                 # count only
fs_cli -x 'show calls'                          # bridged pairs only
fs_cli -x 'show calls count'
fs_cli -x 'show registrations'                  # SIP registrations on all profiles
fs_cli -x 'show distinct_channels'              # one row per leg

# Sofia
fs_cli -x 'sofia status'                        # all profiles + gateways
fs_cli -x 'sofia status profile internal'       # one profile detail
fs_cli -x 'sofia status profile internal reg'   # registrations on profile
fs_cli -x 'sofia status gateway gw1'            # gateway state (REGED, FAIL_WAIT…)
fs_cli -x 'sofia profile internal restart reloadxml'
fs_cli -x 'sofia profile internal killgw gw1'
fs_cli -x 'sofia profile internal rescan'
fs_cli -x 'sofia loglevel all 9'                # full SIP trace
fs_cli -x 'sofia loglevel all 0'                # off
fs_cli -x 'sofia tracelevel debug'
fs_cli -x 'sofia global siptrace on'            # log every SIP packet
fs_cli -x 'sofia global siptrace off'

# XML reload
fs_cli -x 'reloadxml'                           # reload dialplan + directory + configs (no profile restart)
fs_cli -x 'reload mod_dialplan_xml'
fs_cli -x 'reload mod_lua'

# Originate (outbound call)
fs_cli -x 'originate user/1001 &echo'           # call extension 1001, bridge into echo app
fs_cli -x 'originate sofia/gateway/gw1/+15551234567 &park'
fs_cli -x 'originate {origination_caller_id_number=18005551212}sofia/gateway/gw1/+15555555555 9999 XML default'
fs_cli -x 'originate sofia/internal/1001@10.0.0.5 9196 XML default'

# Kill
fs_cli -x 'uuid_kill <uuid>'
fs_cli -x 'uuid_kill <uuid> NORMAL_CLEARING'
fs_cli -x 'hupall NORMAL_CLEARING'              # hang up everything (dangerous)
fs_cli -x 'hupall NORMAL_CLEARING dialed_user 1001'   # only matching channels

# Misc
fs_cli -x 'show modules'
fs_cli -x 'show codec'
fs_cli -x 'show endpoint'
fs_cli -x 'show application'
fs_cli -x 'show api'
fs_cli -x 'eval ${hostname}'                    # variable expansion
fs_cli -x 'global_getvar hostname'
fs_cli -x 'global_setvar default_password=verysecret'
```

## XML Configuration

FreeSWITCH config is one big XML tree assembled from many files via `X-PRE-PROCESS`. The root is `freeswitch.xml`; everything else is included by reference. After editing, `reloadxml` rebuilds the in-memory tree (does not restart Sofia profiles or modules).

```xml
<!-- /etc/freeswitch/freeswitch.xml — root document -->
<?xml version="1.0"?>
<document type="freeswitch/xml">
  <X-PRE-PROCESS cmd="include" data="vars.xml"/>          <!-- global vars first -->
  <section name="configuration" description="Various Configuration">
    <X-PRE-PROCESS cmd="include" data="autoload_configs/*.xml"/>
  </section>
  <section name="dialplan" description="Regex/XML Dialplan">
    <X-PRE-PROCESS cmd="include" data="dialplan/*.xml"/>
  </section>
  <section name="directory" description="User Directory">
    <X-PRE-PROCESS cmd="include" data="directory/*.xml"/>
  </section>
</document>

<!-- vars.xml — set $${variable} (compile-time) and ${variable} (runtime) -->
<include>
  <X-PRE-PROCESS cmd="set" data="domain=pbx.example.com"/>
  <X-PRE-PROCESS cmd="set" data="default_password=verysecret_change_me"/>
  <X-PRE-PROCESS cmd="set" data="external_rtp_ip=auto-nat"/>
  <X-PRE-PROCESS cmd="set" data="external_sip_ip=auto-nat"/>
  <X-PRE-PROCESS cmd="set" data="bind_server_ip=auto"/>
  <X-PRE-PROCESS cmd="set" data="internal_sip_port=5060"/>
  <X-PRE-PROCESS cmd="set" data="external_sip_port=5080"/>
  <X-PRE-PROCESS cmd="set" data="global_codec_prefs=OPUS,G722,PCMU,PCMA"/>
  <X-PRE-PROCESS cmd="set" data="outbound_codec_prefs=PCMU,PCMA"/>
</include>

<!-- $${var} = preprocessor (parsed once at load); ${var} = runtime channel/global -->
```

`X-PRE-PROCESS` directives:

- `cmd="include" data="path"` — include a file or glob.
- `cmd="set" data="name=value"` — define a `$${name}` variable.
- `cmd="exec-set" data="name=$(cmd)"` — set from shell exec.
- `cmd="env-set" data="name=ENV_VAR"` — pull from process env.
- `cmd="stun-set" data="name=stun:stun.example.com"` — STUN reflexive lookup.

## Modules

Modules are loaded by `modules.conf.xml`. Add a `<load module="mod_x"/>` line, then `load mod_x` from fs_cli (or restart).

```xml
<!-- /etc/freeswitch/autoload_configs/modules.conf.xml -->
<configuration name="modules.conf" description="Modules">
  <modules>
    <load module="mod_console"/>
    <load module="mod_logfile"/>
    <load module="mod_sofia"/>             <!-- SIP/SDP endpoint -->
    <load module="mod_dialplan_xml"/>      <!-- XML dialplan parser -->
    <load module="mod_commands"/>          <!-- show, status, originate APIs -->
    <load module="mod_dptools"/>           <!-- bridge, answer, playback apps -->
    <load module="mod_event_socket"/>      <!-- ESL TCP server -->
    <load module="mod_lua"/>               <!-- Lua scripting -->
    <load module="mod_conference"/>        <!-- conference rooms -->
    <load module="mod_voicemail"/>
    <load module="mod_callcenter"/>        <!-- ACD queues -->
    <load module="mod_xml_curl"/>          <!-- fetch dialplan/directory over HTTP -->
    <load module="mod_xml_cdr"/>
    <load module="mod_cdr_csv"/>
    <load module="mod_verto"/>             <!-- WebRTC native protocol -->
    <load module="mod_local_stream"/>      <!-- music on hold -->
    <load module="mod_native_file"/>
    <load module="mod_sndfile"/>
    <load module="mod_opus"/>
    <load module="mod_g729"/>
  </modules>
</configuration>
```

| Module | Role |
|---|---|
| **mod_sofia** | SIP/SDP/RTP endpoint built on Sofia-SIP. Handles UA, registrar, B2BUA. |
| **mod_dialplan_xml** | Parses `dialplan/*.xml` extensions and matches calls. |
| **mod_commands** | Adds `show`, `status`, `originate`, `uuid_*`, `sofia` APIs. |
| **mod_lua** | Embed Lua; runs `scripts/*.lua` from the dialplan or shell. |
| **mod_event_socket** | TCP server (default 8021) speaking ESL — control + event stream. |
| **mod_conference** | Audio bridges, profiles, MOH, energy detection. |
| **mod_voicemail** | Mailboxes + IMAP/SMTP delivery; per-domain config. |
| **mod_callcenter** | ACD queues with strategies (longest-idle-agent, round-robin, ring-all). |
| **mod_xml_curl** | Fetches dialplan/directory/config from HTTP — replaces XML files dynamically. |

```bash
fs_cli -x 'load mod_lua'
fs_cli -x 'unload mod_lua'
fs_cli -x 'reload mod_lua'
fs_cli -x 'show modules' | grep lua
```

## Sofia SIP Profiles

A *profile* is one SIP listener (UDP/TCP/TLS on a host:port pair). Stock config has two — `internal` (port 5060, requires registration, for handsets/softphones) and `external` (port 5080, no auth, for ITSP/gateway traffic). Each profile has its own auth realm, RTP range, codecs, and NAT settings.

```xml
<!-- /etc/freeswitch/sip_profiles/internal.xml -->
<profile name="internal">
  <aliases/>
  <gateways>
    <X-PRE-PROCESS cmd="include" data="internal/*.xml"/>
  </gateways>
  <domains>
    <domain name="all" alias="true" parse="false"/>
  </domains>
  <settings>
    <param name="user-agent-string" value="FreeSWITCH"/>
    <param name="debug" value="0"/>
    <param name="sip-trace" value="no"/>
    <param name="sip-port" value="$${internal_sip_port}"/>           <!-- 5060 -->
    <param name="rtp-ip" value="$${local_ip_v4}"/>                   <!-- bind RTP local -->
    <param name="sip-ip" value="$${local_ip_v4}"/>                   <!-- bind SIP local -->
    <param name="ext-rtp-ip" value="auto-nat"/>                      <!-- public RTP IP -->
    <param name="ext-sip-ip" value="auto-nat"/>                      <!-- public SIP IP -->
    <param name="rtp-timer-name" value="soft"/>
    <param name="rtp-timeout-sec" value="300"/>
    <param name="rtp-hold-timeout-sec" value="1800"/>
    <param name="auth-calls" value="true"/>                          <!-- require auth -->
    <param name="apply-inbound-acl" value="domains"/>
    <param name="local-network-acl" value="localnet.auto"/>
    <param name="context" value="public"/>                           <!-- where unmatched calls go -->
    <param name="dialplan" value="XML"/>
    <param name="dtmf-duration" value="2000"/>
    <param name="codec-prefs" value="$${global_codec_prefs}"/>
    <param name="inbound-codec-negotiation" value="generous"/>
    <param name="hold-music" value="local_stream://moh"/>
    <param name="record-path" value="$${recordings_dir}"/>
    <param name="record-template" value="${caller_id_number}.${strftime(%Y-%m-%d-%H-%M-%S)}.wav"/>
    <param name="manage-presence" value="true"/>
    <param name="presence-hosts" value="$${domain},$${local_ip_v4}"/>
    <param name="force-register-domain" value="$${domain}"/>          <!-- normalize REGISTER To: -->
    <param name="force-subscription-domain" value="$${domain}"/>
    <param name="aggressive-nat-detection" value="true"/>
    <param name="apply-nat-acl" value="nat.auto"/>
    <param name="nonce-ttl" value="60"/>
    <param name="tls" value="false"/>
    <param name="tls-bind-params" value="transport=tls"/>
    <param name="tls-sip-port" value="5061"/>
    <param name="tls-cert-dir" value="$${certs_dir}"/>
    <param name="tls-version" value="tlsv1.2,tlsv1.3"/>
    <param name="ws-binding" value=":5066"/>                          <!-- SIP over WebSocket -->
    <param name="wss-binding" value=":7443"/>                         <!-- SIP over secure WS -->
  </settings>
</profile>
```

`internal` vs `external`:

| Param | internal | external |
|---|---|---|
| `sip-port` | 5060 | 5080 |
| `auth-calls` | true | false |
| `accept-blind-reg` | false | n/a |
| `context` | default (after auth) | public (untrusted) |
| `apply-inbound-acl` | domains | (often none) |
| `force-register-domain` | yes | no |

Restart a profile after settings change:

```bash
fs_cli -x 'sofia profile internal restart reloadxml'
fs_cli -x 'sofia profile external restart reloadxml'
fs_cli -x 'sofia profile internal stop'
fs_cli -x 'sofia profile internal start'
```

## Sofia Gateways

Gateways are *upstream registrations* — FreeSWITCH registering as a UA to a SIP trunk or another PBX. Each gateway lives under a profile (typically `external`).

```xml
<!-- /etc/freeswitch/sip_profiles/external/gw1.xml -->
<include>
  <gateway name="gw1">
    <param name="username" value="ACCT12345"/>
    <param name="realm" value="sip.itsp.example"/>
    <param name="from-domain" value="sip.itsp.example"/>
    <param name="password" value="s3cret"/>
    <param name="register" value="true"/>                  <!-- send REGISTER -->
    <param name="register-transport" value="udp"/>         <!-- udp|tcp|tls -->
    <param name="expire-seconds" value="600"/>
    <param name="retry-seconds" value="30"/>
    <param name="ping" value="25"/>                        <!-- OPTIONS keepalive (sec) -->
    <param name="ping-max" value="3"/>
    <param name="ping-min" value="1"/>
    <param name="caller-id-in-from" value="false"/>
    <param name="contact-params" value="tport=udp"/>
    <param name="extension" value="auto"/>                 <!-- auto = match @gw1 -->
    <param name="proxy" value="sip.itsp.example"/>
    <param name="register-proxy" value="sip.itsp.example"/>
    <param name="codec-prefs" value="PCMU,PCMA"/>
    <param name="dtmf-type" value="rfc2833"/>
  </gateway>
</include>
```

Gateway operations:

```bash
fs_cli -x 'sofia profile external rescan'                  # pick up new gateway file
fs_cli -x 'sofia profile external killgw gw1'              # tear down + reload
fs_cli -x 'sofia profile external startgw gw1'
fs_cli -x 'sofia status gateway gw1'                       # state: REGED | UNREGED | TRYING | FAIL_WAIT
fs_cli -x 'sofia profile external register gw1'            # force re-register

# Originate via gateway
fs_cli -x 'originate sofia/gateway/gw1/+15551234567 &echo'
```

States: `NOREG` (register=false), `REGED` (200 OK), `UNREGED`, `TRYING`, `REGISTER` (sending), `FAILED`, `FAIL_WAIT` (back-off after auth/network failure).

## Directory User config

The directory defines SIP credentials, voicemail boxes, and channel variables per user. One file per user is the convention.

```xml
<!-- /etc/freeswitch/directory/default/1001.xml -->
<include>
  <user id="1001">
    <params>
      <param name="password" value="$${default_password}"/>
      <param name="vm-password" value="1001"/>
      <param name="dial-string" value="{^^:sip_invite_domain=${dialed_domain}:presence_id=${dialed_user}@${dialed_domain}}${sofia_contact(*/${dialed_user}@${dialed_domain})}"/>
    </params>
    <variables>
      <variable name="toll_allow" value="domestic,international,local"/>
      <variable name="accountcode" value="1001"/>
      <variable name="user_context" value="default"/>
      <variable name="effective_caller_id_name" value="Alice Example"/>
      <variable name="effective_caller_id_number" value="1001"/>
      <variable name="outbound_caller_id_name" value="$${outbound_caller_name}"/>
      <variable name="outbound_caller_id_number" value="$${outbound_caller_id}"/>
      <variable name="callgroup" value="techsupport"/>
    </variables>
  </user>
</include>
```

Group membership lives in `default.xml` `<group>` blocks pointing at user IDs:

```xml
<group name="sales">
  <users>
    <user id="1001" type="pointer"/>
    <user id="1002" type="pointer"/>
  </users>
</group>
```

```bash
fs_cli -x 'reloadxml'                                      # pick up new users
fs_cli -x 'sofia status profile internal reg'              # see who is registered
fs_cli -x 'user_exists id 1001 default'                    # → true|false
fs_cli -x 'user_data 1001@$${domain} param password'
```

## Dialplan

Dialplan is XML extensions matched in order against channel variables. First matching `<extension>` wins; `continue="true"` lets execution fall through to the next match.

```xml
<!-- /etc/freeswitch/dialplan/default.xml -->
<include>
  <context name="default">

    <!-- Local extensions -->
    <extension name="Local_Extension">
      <condition field="destination_number" expression="^(10[01][0-9])$">
        <action application="set" data="dialed_extension=$1"/>
        <action application="export" data="dialed_extension=$1"/>
        <action application="set" data="ringback=${us-ring}"/>
        <action application="set" data="transfer_ringback=$${hold_music}"/>
        <action application="set" data="hangup_after_bridge=true"/>
        <action application="set" data="continue_on_fail=NORMAL_TEMPORARY_FAILURE,USER_BUSY,NO_ANSWER,TIMEOUT,NO_ROUTE_DESTINATION"/>
        <action application="bridge" data="user/$1@${domain_name}"/>
        <action application="answer"/>
        <action application="sleep" data="1000"/>
        <action application="voicemail" data="default ${domain_name} $1"/>
      </condition>
    </extension>

    <!-- Outbound PSTN via gateway -->
    <extension name="outbound_pstn">
      <condition field="destination_number" expression="^(\+?1?\d{10})$">
        <action application="set" data="effective_caller_id_number=${outbound_caller_id_number}"/>
        <action application="bridge" data="sofia/gateway/gw1/$1"/>
      </condition>
    </extension>

  </context>
</include>
```

Loading order: files in `dialplan/default/*.xml` are concatenated in alphabetical order, so prefix names like `00_local.xml`, `10_outbound.xml`, `99_default.xml` to control match priority.

## Dialplan Conditions

`<condition>` matches a channel variable against a regex. Regex captures `($1, $2, …)` are visible to actions. Multiple conditions on one extension are AND-ed; use nested `<condition break="...">` for OR. Common fields:

| Field | Meaning |
|---|---|
| `destination_number` | Dialed digits |
| `caller_id_number` | Incoming CLI |
| `network_addr` | Source IP (works for ACL) |
| `${user_context}` | Channel variable lookup |
| `context` | Profile-specified context |

```xml
<!-- Single regex -->
<condition field="destination_number" expression="^(\d{4})$">
  <action application="bridge" data="user/$1"/>
</condition>

<!-- Negated condition: only run if no match -->
<extension name="reject_offhours">
  <condition wday="2-6" hour="9-17">
    <anti-action application="hangup" data="NORMAL_TEMPORARY_FAILURE"/>
  </condition>
</extension>

<!-- Time-of-day -->
<condition wday="2-6" hour="9-17">                     <!-- Mon–Fri 9-5 -->
  <action application="bridge" data="group/sales@${domain_name}"/>
</condition>

<!-- Channel variable check -->
<condition field="${vip_caller}" expression="^true$">
  <action application="set" data="ringback=$${us-ring}"/>
</condition>

<!-- IP-based ACL via network_addr -->
<condition field="network_addr" expression="^10\.0\.0\.\d+$">
  <action application="set" data="trusted=true"/>
</condition>

<!-- Multiple extensions, fall-through with continue -->
<extension name="set_ringtone" continue="true">
  <condition field="destination_number" expression="^10[01][0-9]$">
    <action application="set" data="alert_info=&lt;http://ring.example/internal.bell&gt;;info=Alert-Internal"/>
  </condition>
</extension>
```

## Dialplan Actions

Applications run in order inside a matched condition. `application=` runs an app; `data=` is its argument. The most-used apps:

```xml
<!-- Answer/hangup -->
<action application="answer"/>
<action application="pre_answer"/>                              <!-- 183 + early media -->
<action application="hangup"/>
<action application="hangup" data="USER_BUSY"/>

<!-- Bridge a leg -->
<action application="bridge" data="user/1001@${domain_name}"/>
<action application="bridge" data="sofia/gateway/gw1/+15551234567"/>
<action application="bridge" data="loopback/foo/default"/>

<!-- Playback -->
<action application="playback" data="ivr/ivr-welcome.wav"/>
<action application="playback" data="$${sounds_dir}/en/us/callie/ivr/8000/ivr-welcome.wav"/>
<action application="say" data="en number iterated 1234"/>      <!-- mod_say_en -->

<!-- Capture digits while playing -->
<!-- play_and_get_digits min max tries timeout terminator file invalid var regex var-set -->
<action application="play_and_get_digits"
        data="4 4 3 5000 # ivr/ivr-please_enter_extension.wav ivr/ivr-that_was_an_invalid_entry.wav extension \d{4}"/>

<!-- Voicemail -->
<action application="voicemail" data="default ${domain_name} 1001"/>
<action application="voicemail" data="check default ${domain_name}"/>     <!-- VM box check -->

<!-- Conference -->
<action application="conference" data="3000@default"/>
<action application="conference" data="3000@default+flags{moderator}"/>

<!-- Variable manipulation -->
<action application="set" data="hangup_after_bridge=true"/>
<action application="export" data="origination_caller_id_number=18001112222"/>
<action application="unset" data="hangup_after_bridge"/>
<action application="set" data="continue_on_fail=USER_BUSY,NO_ANSWER"/>
<action application="set" data="execute_on_answer=record_session $${recordings_dir}/${uuid}.wav"/>

<!-- Transfer (replace remaining dialplan) -->
<action application="transfer" data="3000 XML default"/>
<action application="transfer" data="-bleg 1001 XML default"/>
<action application="deflect" data="sip:1001@10.0.0.5"/>
<action application="redirect" data="sip:1001@10.0.0.5"/>

<!-- Lua -->
<action application="lua" data="ivr.lua arg1 arg2"/>

<!-- Sleep / DTMF / record -->
<action application="sleep" data="2000"/>
<action application="send_dtmf" data="1234"/>
<action application="record" data="/tmp/${uuid}.wav 30 200"/>
<action application="record_session" data="$${recordings_dir}/${uuid}.wav"/>
<action application="stop_record_session" data="$${recordings_dir}/${uuid}.wav"/>

<!-- Park/Park-and-announce -->
<action application="park"/>
<action application="valet_park" data="valet_lot1 1"/>

<!-- Music on hold -->
<action application="playback" data="local_stream://moh"/>
```

## Bridge Application

`bridge` originates a B-leg and joins it to the A-leg. The destination string supports two separators:

- `,` (comma) — call all destinations in *parallel* (fork-and-fork-cancel).
- `|` (pipe) — call destinations *sequentially* (next on failure).

```xml
<!-- Single endpoint -->
<action application="bridge" data="user/1001"/>
<action application="bridge" data="user/1001@example.com"/>

<!-- Sofia URL -->
<action application="bridge" data="sofia/internal/1001@10.0.0.5:5060"/>

<!-- Gateway -->
<action application="bridge" data="sofia/gateway/gw1/+15551234567"/>

<!-- Parallel ring (fork) — first to answer wins, others canceled -->
<action application="bridge" data="user/1001,user/1002,user/1003"/>

<!-- Hunt: try in order, next on failure -->
<action application="bridge" data="user/1001|user/1002|user/1003"/>

<!-- Inline channel variables: {var=val,var2=val2}target -->
<action application="bridge" data="{leg_timeout=20,absolute_codec_string=PCMU}user/1001"/>

<!-- Mixed: ring 1001 + 1002 in parallel, fall to 1003 if both fail -->
<action application="bridge" data="user/1001,user/1002|user/1003"/>

<!-- Group dial -->
<action application="bridge" data="group/sales@${domain_name}"/>
```

`{...}` syntax sets *channel variables on the originated leg only*. `[...]` syntax sets *cluster variables* applied to every leg in a parallel set.

```xml
<action application="bridge" data="[leg_timeout=20]user/1001,user/1002,user/1003"/>
```

## Origination String

Same syntax used by `originate` API and `bridge`. Common forms:

```
user/<extension>[@domain]                      directory user lookup
user/1001@example.com
sofia/<profile>/<destination>                  raw Sofia URL
sofia/internal/1001@10.0.0.5:5060
sofia/external/+15551234567@sip.itsp.example
sofia/gateway/<gw_name>/<destination>          via configured gateway
sofia/gateway/gw1/+15551234567
loopback/<dest>[/context[/dialplan]]           loopback channel for re-routing
loopback/foo/default
freetdm/<span>/<channel|a|A>/<dest>            T1/E1
group/<name>@<domain>                          parallel ring all group members
```

`originate` examples from fs_cli:

```bash
# call extension, drop into IVR menu
fs_cli -x 'originate user/1001 5000 XML default'

# bridge two legs (no app — use & for app)
fs_cli -x 'originate user/1001 &bridge(user/1002)'
fs_cli -x 'originate user/1001 &echo'
fs_cli -x 'originate user/1001 &park'
fs_cli -x 'originate user/1001 &conference(3000@default)'

# channel vars on the originated leg
fs_cli -x 'originate {origination_caller_id_number=18001112222,origination_caller_id_name=Robocall,leg_timeout=15}user/1001 &echo'

# parallel — first answer wins
fs_cli -x "originate {ignore_early_media=true}user/1001,user/1002 &park"

# bgapi — return UUID immediately, run in background
fs_cli -x 'bgapi originate user/1001 &echo'
```

## ESL (Event Socket)

ESL = TCP control channel for FreeSWITCH (mod_event_socket). Two flavors:

- **Inbound** — your app connects *to* FreeSWITCH (default `127.0.0.1:8021`, password `ClueCon`). Issue API calls, subscribe to events globally.
- **Outbound** — FreeSWITCH connects *to* your app (configured per channel via dialplan `socket` action). Your app drives a single channel.

```xml
<!-- /etc/freeswitch/autoload_configs/event_socket.conf.xml -->
<configuration name="event_socket.conf" description="Socket Client">
  <settings>
    <param name="nat-map" value="false"/>
    <param name="listen-ip" value="127.0.0.1"/>            <!-- 0.0.0.0 only on trusted nets -->
    <param name="listen-port" value="8021"/>
    <param name="password" value="ClueCon"/>               <!-- CHANGE ME -->
    <param name="apply-inbound-acl" value="loopback.auto"/>
  </settings>
</configuration>
```

Wire protocol — newline-delimited headers + blank line, occasionally with a Content-Length body:

```
auth ClueCon\n\n
api status\n\n
bgapi originate user/1001 &echo\n\n
event plain CHANNEL_CREATE CHANNEL_ANSWER CHANNEL_HANGUP DTMF\n\n
event json ALL\n\n
nixevent CHANNEL_HEARTBEAT\n\n
filter Event-Name CHANNEL_HANGUP\n\n
sendmsg <uuid>\ncall-command: execute\nexecute-app-name: playback\nexecute-app-arg: ivr/ivr-welcome.wav\n\n
```

Commands:

| Cmd | Use |
|---|---|
| `api <cmd>` | Synchronous API call; returns body. |
| `bgapi <cmd>` | Async; returns Job-UUID, response arrives as `BACKGROUND_JOB` event. |
| `event plain <NAMES>` | Subscribe — newline-delimited `key: value` body. |
| `event json <NAMES>` | Same but JSON body. |
| `event xml ALL` | XML body. |
| `nixevent <NAMES>` | Unsubscribe specific events. |
| `noevents` | Drop all subscriptions. |
| `filter <K> <V>` | Only deliver events matching header. |
| `sendmsg <uuid>` | Drive a specific channel — execute app, hangup, etc. |
| `divert_events on` | Re-route eventing to outbound socket. |
| `myevents <uuid>` | Outbound only: subscribe to events for this leg. |
| `linger` | Outbound only: keep socket open after channel hangs up. |
| `exit` | Close. |

Common events to subscribe:

```
CHANNEL_CREATE        new leg
CHANNEL_PROGRESS      183
CHANNEL_PROGRESS_MEDIA 183 with SDP
CHANNEL_ANSWER        200 OK
CHANNEL_BRIDGE        legs joined
CHANNEL_UNBRIDGE
CHANNEL_HANGUP        BYE/CANCEL
CHANNEL_HANGUP_COMPLETE
CHANNEL_DESTROY
DTMF                  digit
PRESENCE_IN
CUSTOM <subclass>     e.g. conference::maintenance, sofia::register
BACKGROUND_JOB        bgapi result
HEARTBEAT
```

Inbound example with bash `nc`:

```bash
{ printf 'auth ClueCon\n\n'; sleep 0.1
  printf 'api status\n\n'; sleep 0.1
  printf 'event plain CHANNEL_CREATE CHANNEL_HANGUP\n\n'; sleep 60; } | nc 127.0.0.1 8021
```

Outbound socket (FS dials *out* to your app per channel):

```xml
<extension name="outbound_socket_demo">
  <condition field="destination_number" expression="^9999$">
    <action application="socket" data="127.0.0.1:8084 async full"/>
  </condition>
</extension>
```

Your app, on accepting that TCP connection, runs `connect`, then `myevents`, then drives the leg via `sendmsg`.

## Lua

`mod_lua` embeds Lua 5.x. Scripts live in `/usr/share/freeswitch/scripts/`. Invoke via `lua` action or `luarun` API. The `session` global represents the current channel (only present when called from the dialplan).

```xml
<action application="lua" data="ivr.lua extension=1001"/>
```

```lua
-- /usr/share/freeswitch/scripts/ivr.lua
freeswitch.consoleLog("INFO", "ivr.lua started\n")

if not session:ready() then return end
session:answer()
session:setVariable("hangup_after_bridge", "true")

session:streamFile("ivr/ivr-welcome.wav")

-- Get DTMF
local digits = session:playAndGetDigits(
    1, 4, 3, 5000, "#",
    "ivr/ivr-please_enter_extension.wav",
    "ivr/ivr-that_was_an_invalid_entry.wav",
    "^\\d+$",
    "extension", 5000)
freeswitch.consoleLog("INFO", "user dialed " .. digits .. "\n")

if digits == "" then
    session:streamFile("ivr/ivr-no_response.wav")
    session:hangup("NO_USER_RESPONSE")
    return
end

session:execute("transfer", digits .. " XML default")

-- Originate from API context (no session)
local api = freeswitch.API()
local uuid = api:executeString("create_uuid")
api:executeString("originate {origination_uuid=" .. uuid .. "}user/1001 &park")

-- Database
local db = freeswitch.Dbh("odbc://pbx:user:pass")
assert(db:connected())
db:query("SELECT id, name FROM users WHERE ext = '1001'", function(row)
    freeswitch.consoleLog("INFO", row.name .. "\n")
end)
db:release()
```

Console-log levels: `DEBUG`, `INFO`, `NOTICE`, `WARNING`, `ERR`, `CRIT`, `ALERT`. Run a script outside a call:

```bash
fs_cli -x 'luarun ivr.lua'                 # background
fs_cli -x 'lua ivr.lua'                    # synchronous (blocks until done)
```

## Conference

`mod_conference` provides audio bridges. Rooms are configured by *profile* (mix rate, video, MOH); calls join via the `conference` app or `conference` API.

```xml
<!-- /etc/freeswitch/autoload_configs/conference.conf.xml — excerpt -->
<configuration name="conference.conf" description="Audio Conference">
  <profiles>
    <profile name="default">
      <param name="rate" value="16000"/>                  <!-- mix sample rate -->
      <param name="interval" value="20"/>                 <!-- ptime ms -->
      <param name="energy-level" value="100"/>
      <param name="caller-id-name" value="FreeSWITCH"/>
      <param name="caller-id-number" value="0000000000"/>
      <param name="muted-sound" value="conference/conf-muted.wav"/>
      <param name="unmuted-sound" value="conference/conf-unmuted.wav"/>
      <param name="alone-sound" value="conference/conf-alone.wav"/>
      <param name="moh-sound" value="$${hold_music}"/>
      <param name="enter-sound" value="tone_stream://%(200,0,500,600,700)"/>
      <param name="exit-sound" value="tone_stream://%(500,0,300,200,100,50,25)"/>
      <param name="comfort-noise" value="true"/>
      <param name="auto-record" value="$${recordings_dir}/conf-${conference_name}-${strftime(%Y%m%d-%H%M%S)}.wav"/>
    </profile>
    <profile name="video-mcu-stereo">
      <param name="video-mode" value="mux"/>              <!-- video mixing -->
      <param name="rate" value="48000"/>
      <param name="interval" value="20"/>
      <param name="video-codec-bandwidth" value="2mb"/>
    </profile>
  </profiles>
</configuration>
```

Joining a room from dialplan:

```xml
<extension name="conference">
  <condition field="destination_number" expression="^3000$">
    <action application="answer"/>
    <action application="conference" data="3000@default"/>
  </condition>
</extension>

<!-- with flags -->
<action application="conference" data="3000@default+flags{moderator|nomoh|mute}"/>
```

API control:

```bash
fs_cli -x 'conference 3000 list'
fs_cli -x 'conference 3000 list count'
fs_cli -x 'conference 3000 mute all'
fs_cli -x 'conference 3000 unmute all'
fs_cli -x 'conference 3000 kick <member-id>'
fs_cli -x 'conference 3000 lock'
fs_cli -x 'conference 3000 unlock'
fs_cli -x 'conference 3000 record /tmp/3000.wav'
fs_cli -x 'conference 3000 norecord'
fs_cli -x 'conference 3000 dial user/1001'
fs_cli -x 'conference 3000 bgdial sofia/gateway/gw1/+15551234567'
fs_cli -x 'conference 3000 play file.wav'
fs_cli -x 'conference 3000 say "hello world"'
fs_cli -x 'conference 3000 vid-floor <member-id> force'
fs_cli -x 'conference 3000 hup'                            # hang up everyone
```

## Verto

Verto is FreeSWITCH's WebRTC-native protocol — JSON-RPC over WebSocket plus DTLS-SRTP media. Lower overhead than SIP-over-WS; pairs with the `verto.js` browser client for in-browser softphones.

```xml
<!-- /etc/freeswitch/autoload_configs/verto.conf.xml — minimal -->
<configuration name="verto.conf" description="Verto">
  <settings>
    <param name="debug" value="0"/>
  </settings>
  <profiles>
    <profile name="default-v4">
      <param name="bind-local" value="$${local_ip_v4}:8081"/>     <!-- ws -->
      <param name="bind-local" value="$${local_ip_v4}:8082" secure="true"/>  <!-- wss -->
      <param name="secure-combined" value="$${certs_dir}/wss.pem"/>
      <param name="userauth" value="true"/>
      <param name="context" value="default"/>
      <param name="dialplan" value="XML"/>
      <param name="rtp-ip" value="$${local_ip_v4}"/>
      <param name="ext-rtp-ip" value="$${external_rtp_ip}"/>
      <param name="local-network" value="localnet.auto"/>
      <param name="outbound-codec-string" value="opus,vp8,h264"/>
      <param name="inbound-codec-string" value="opus,vp8,h264"/>
      <param name="apply-candidate-acl" value="localnet.auto"/>
      <param name="apply-candidate-acl" value="wan.auto"/>
    </profile>
  </profiles>
</configuration>
```

Auth uses the directory `password`/`vm-password` of each user against the configured domain; ws/wss URL is `wss://pbx.example.com:8082/`.

```bash
fs_cli -x 'verto status'
fs_cli -x 'verto debug 9'
fs_cli -x 'reload mod_verto'
```

## SIP-over-WebSocket

mod_sofia carries SIP over WS/WSS — the standard transport for SIP.js, JsSIP, sipML5 browser clients. Bind in the profile.

```xml
<!-- inside <profile name="internal"> <settings> -->
<param name="ws-binding"  value=":5066"/>
<param name="wss-binding" value=":7443"/>
<param name="tls-cert-dir" value="$${certs_dir}"/>          <!-- wss.pem -->
<param name="apply-inbound-acl" value="domains"/>
```

ws/wss bindings must coexist with TLS-cert configuration (wss requires tls-cert-dir). Browser clients use SDP with `a=ice-...`, `a=setup:actpass`, `a=fingerprint:...` — FS handles ICE/DTLS-SRTP via the `mod_sofia` ICE engine when `rtcp-mux`, `dtls`, and `apply-candidate-acl` are configured.

```javascript
// SIP.js example
const ua = new SIP.UserAgent({
  uri: SIP.UserAgent.makeURI("sip:1001@pbx.example.com"),
  authorizationUsername: "1001",
  authorizationPassword: "verysecret",
  transportOptions: { server: "wss://pbx.example.com:7443" }
});
ua.start();
```

## Codecs

FreeSWITCH ships with most narrowband and wideband codecs; some need explicit modules (mod_opus, mod_g729). Negotiation is per-profile; `inbound-codec-negotiation` controls greedy vs generous selection.

| Codec | Bitrate | Sample | Module | Notes |
|---|---|---|---|---|
| **PCMU (G.711μ)** | 64 kbps | 8 kHz | core | US PSTN baseline |
| **PCMA (G.711a)** | 64 kbps | 8 kHz | core | EU PSTN baseline |
| **G.722** | 64 kbps | 16 kHz | core | HD voice, IP-only |
| **G.729** | 8 kbps | 8 kHz | mod_g729 | Compressed; license historically; now royalty-free |
| **OPUS** | 6–510 kbps | 8/16/24/48 kHz | mod_opus | Default for WebRTC; VBR |
| **iSAC** | 10–32 kbps | 16/32 kHz | mod_isac | Older WebRTC; deprecated by Chrome 2022+ |
| **iLBC** | 13.3/15.2 kbps | 8 kHz | mod_ilbc | Loss-tolerant |
| **SILK** | 6–40 kbps | 8/12/16/24 kHz | mod_silk | Skype-origin; subset of Opus |

Profile/global settings:

```xml
<!-- vars.xml -->
<X-PRE-PROCESS cmd="set" data="global_codec_prefs=OPUS,G722,PCMU,PCMA,G729"/>
<X-PRE-PROCESS cmd="set" data="outbound_codec_prefs=PCMU,PCMA"/>

<!-- profile -->
<param name="codec-prefs" value="$${global_codec_prefs}"/>
<param name="inbound-codec-negotiation" value="generous"/>
<!-- generous = honor remote pref order; greedy = our pref order; scrooge = first match only -->

<!-- per-call override -->
<action application="set" data="absolute_codec_string=PCMU,PCMA"/>
<action application="set" data="codec_string=PCMU,PCMA"/>
```

Inspect:

```bash
fs_cli -x 'show codec'
fs_cli -x 'show endpoint'
```

## RTP

UDP media, default port range 16384–32768. Configure per-profile via `rtp-port-min`, `rtp-port-max`. `ext-rtp-ip` is the public IP advertised in `c=` and `o=` lines of SDP when behind NAT.

```xml
<!-- /etc/freeswitch/autoload_configs/switch.conf.xml -->
<param name="rtp-start-port" value="16384"/>
<param name="rtp-end-port" value="32768"/>

<!-- profile -->
<param name="rtp-ip" value="$${local_ip_v4}"/>            <!-- bind -->
<param name="ext-rtp-ip" value="auto-nat"/>               <!-- advertise -->
<param name="rtp-timeout-sec" value="300"/>
<param name="rtp-hold-timeout-sec" value="1800"/>
<param name="rtp-timer-name" value="soft"/>
<param name="dtmf-type" value="rfc2833"/>                 <!-- rfc2833 | info | none -->
<param name="suppress-cng" value="true"/>
```

Diagnostics:

```bash
fs_cli -x 'sofia status profile internal'                  # Calls In/Out, RTP stats
fs_cli -x 'show rtp'
ss -ulnp | grep freeswitch                                 # listening UDP ports
fs_cli -x 'uuid_setvar <uuid> rtp_jitter_buffer 60'
fs_cli -x 'uuid_setvar <uuid> jitterbuffer_msec 60:200:5'
```

## NAT

`ext-sip-ip` and `ext-rtp-ip` set what FS advertises in `Contact`, `Via`, and SDP. Use `auto-nat` (NAT-PMP/UPnP/STUN), or `host:stun.example.com`, or a literal IP.

```xml
<!-- vars.xml -->
<X-PRE-PROCESS cmd="set" data="external_rtp_ip=auto-nat"/>
<X-PRE-PROCESS cmd="set" data="external_sip_ip=auto-nat"/>

<!-- alternatives -->
<X-PRE-PROCESS cmd="stun-set" data="external_rtp_ip=stun:stun.freeswitch.org"/>
<X-PRE-PROCESS cmd="set" data="external_sip_ip=203.0.113.10"/>

<!-- profile -->
<param name="ext-rtp-ip" value="$${external_rtp_ip}"/>
<param name="ext-sip-ip" value="$${external_sip_ip}"/>
<param name="aggressive-nat-detection" value="true"/>
<param name="apply-nat-acl" value="nat.auto"/>             <!-- 0.0.0.0/0 minus localnet -->
<param name="local-network-acl" value="localnet.auto"/>
<param name="NDLB-force-rport" value="server-only"/>       <!-- workarounds -->
```

If endpoints are behind symmetric NAT, ICE/STUN/TURN is required (Verto/WebRTC handle this automatically; SIP UAs may need outbound proxy or `nat=force_rport,comedia` style helpers).

## CDR

Two CDR producers — both can run together.

```xml
<!-- /etc/freeswitch/autoload_configs/cdr_csv.conf.xml -->
<configuration name="cdr_csv.conf" description="CDR CSV Format">
  <settings>
    <param name="default-template" value="example"/>
    <param name="rotate-on-hup" value="true"/>
    <param name="legs" value="a"/>                          <!-- a | b | ab -->
    <param name="log-base" value="$${log_dir}/cdr-csv"/>
    <param name="master-file-name" value="Master.csv"/>
  </settings>
  <templates>
    <template name="example">
      "${caller_id_name}","${caller_id_number}","${destination_number}","${context}","${start_stamp}","${answer_stamp}","${end_stamp}","${duration}","${billsec}","${hangup_cause}","${uuid}","${bleg_uuid}","${accountcode}","${read_codec}","${write_codec}"
    </template>
  </templates>
</configuration>
```

```xml
<!-- /etc/freeswitch/autoload_configs/xml_cdr.conf.xml -->
<configuration name="xml_cdr.conf" description="XML CDR HTTP">
  <settings>
    <param name="url" value="http://cdr.example.com/cdr"/>  <!-- POST per call -->
    <param name="auth-scheme" value="basic"/>
    <param name="cred" value="user:secret"/>
    <param name="log-dir" value="$${log_dir}/xml_cdr"/>
    <param name="err-log-dir" value="$${log_dir}/xml_cdr_failed"/>
    <param name="retries" value="3"/>
    <param name="delay" value="5"/>
    <param name="encode" value="true"/>
  </settings>
</configuration>
```

```bash
ls -la /var/log/freeswitch/cdr-csv/
tail -F /var/log/freeswitch/cdr-csv/Master.csv
fs_cli -x 'reload mod_cdr_csv'
fs_cli -x 'reload mod_xml_cdr'
```

## Recording

```xml
<!-- whole-call (B-leg only by default) -->
<action application="set" data="record_session=$${recordings_dir}/${uuid}.wav"/>
<!-- both legs to same file -->
<action application="set" data="record_stereo=true"/>
<action application="record_session" data="$${recordings_dir}/${uuid}.wav"/>

<!-- on-demand from API -->
<!-- uuid_record <uuid> [start|stop|mask|unmask] <path> -->
fs_cli -x 'uuid_record <uuid> start /tmp/foo.wav'
fs_cli -x 'uuid_record <uuid> stop /tmp/foo.wav'
fs_cli -x 'uuid_record <uuid> mask /tmp/foo.wav'           <!-- mute recording during PCI fields -->
fs_cli -x 'uuid_record <uuid> unmask /tmp/foo.wav'

<!-- eavesdrop / monitor / barge -->
fs_cli -x 'originate user/1001 &eavesdrop(<target-uuid>)'
<!-- DTMF in eavesdrop: 0=mute, 1=barge, 2=clear, 3=both legs -->
```

## Voicemail

`mod_voicemail` mailboxes live per-domain; each user has a `vm-password` in the directory. Storage default `/var/lib/freeswitch/storage/voicemail/default/<domain>/<id>/`. Email delivery via `email`, `vm-notify-mailto`, `vm-mailfrom`.

```xml
<!-- /etc/freeswitch/autoload_configs/voicemail.conf.xml — excerpt -->
<configuration name="voicemail.conf" description="Voicemail">
  <profiles>
    <profile name="default">
      <param name="storage-dir" value="$${storage_dir}/voicemail"/>
      <param name="record-greeting-trim-end" value="true"/>
      <param name="email-from" value="voicemail@${domain_name}"/>
      <param name="notify-template-file" value="voicemail/en/notify-voicemail.tpl"/>
      <param name="email-template-file" value="voicemail/en/email-voicemail.tpl"/>
      <param name="attachment-template-file" value="voicemail/en/attachment-voicemail.tpl"/>
      <param name="terminator-key" value="#"/>
    </profile>
  </profiles>
</configuration>
```

Per-user variables (in directory):

```xml
<param name="vm-password" value="1001"/>
<variable name="vm-mailto" value="alice@example.com"/>
<variable name="vm-attach-file" value="true"/>
<variable name="vm-keep-local-after-email" value="true"/>
```

Dialplan:

```xml
<!-- leave a message -->
<action application="voicemail" data="default ${domain_name} 1001"/>

<!-- mailbox check (*98 typical) -->
<extension name="vm_check">
  <condition field="destination_number" expression="^\*98$">
    <action application="answer"/>
    <action application="voicemail" data="check default ${domain_name}"/>
  </condition>
</extension>
```

API:

```bash
fs_cli -x 'vm_list 1001@example.com'
fs_cli -x 'vm_inject 1001@example.com /tmp/hello.wav'
fs_cli -x 'vm_boxcount 1001@example.com'
fs_cli -x 'vm_delete 1001@example.com all'
```

## Call Center

`mod_callcenter` provides ACD queues with agents, tiers, and dispatch strategies. Backed by an internal SQLite DB or external ODBC.

```xml
<!-- /etc/freeswitch/autoload_configs/callcenter.conf.xml -->
<configuration name="callcenter.conf" description="CallCenter">
  <settings>
    <param name="odbc-dsn" value=""/>                       <!-- empty = SQLite -->
  </settings>
  <queues>
    <queue name="support@default">
      <param name="strategy" value="longest-idle-agent"/>
      <param name="moh-sound" value="$${hold_music}"/>
      <param name="time-base-score" value="system"/>
      <param name="max-wait-time" value="0"/>
      <param name="max-wait-time-with-no-agent" value="120"/>
      <param name="tier-rules-apply" value="false"/>
      <param name="tier-rule-wait-second" value="30"/>
      <param name="discard-abandoned-after" value="60"/>
      <param name="abandoned-resume-allowed" value="false"/>
      <param name="record-template" value="$${recordings_dir}/${strftime(%Y-%m-%d)}-${destination_number}-${caller_id_number}-${uuid}.wav"/>
    </queue>
  </queues>
  <agents>
    <agent name="1001@default" type="callback"
           contact="user/1001" status="Logged Out"
           max-no-answer="3" wrap-up-time="10" reject-delay-time="10" busy-delay-time="60"/>
  </agents>
  <tiers>
    <tier agent="1001@default" queue="support@default" level="1" position="1"/>
  </tiers>
</configuration>
```

Strategies: `ring-all`, `longest-idle-agent`, `agent-with-least-talk-time`, `agent-with-fewest-calls`, `sequentially-by-agent-order`, `random`, `top-down`, `round-robin`.

Agent states: `Available`, `Available (On Demand)`, `On Break`, `Logged Out`. Agent statuses: `Idle`, `Waiting`, `Receiving`, `In a queue call`.

```bash
fs_cli -x 'callcenter_config queue list'
fs_cli -x 'callcenter_config agent list'
fs_cli -x 'callcenter_config agent set status 1001@default "Available"'
fs_cli -x 'callcenter_config agent set state 1001@default "Idle"'
fs_cli -x 'callcenter_config tier list'
fs_cli -x 'callcenter_config queue list members support@default'
```

Dialplan entry:

```xml
<extension name="cc_support">
  <condition field="destination_number" expression="^7000$">
    <action application="answer"/>
    <action application="callcenter" data="support@default"/>
  </condition>
</extension>
```

## Presence/BLF

`mod_sofia` publishes SUBSCRIBE/NOTIFY for SIP presence (BLF — busy-lamp-field). `manage-presence=true` on the profile is required.

```xml
<param name="manage-presence" value="true"/>
<param name="presence-hosts" value="$${domain},$${local_ip_v4}"/>
<param name="force-publish-expires" value="3600"/>
```

In the directory user, `presence_id` correlates BLF subscribes to channel events:

```xml
<param name="dial-string" value="{...,presence_id=${dialed_user}@${dialed_domain}}..."/>
```

Provision the phone with a programmable line-key subscribing to `<sip:1001@pbx.example.com>` — lights up when 1001 is busy. Trigger inbound from API:

```bash
fs_cli -x 'sofia profile internal flush_inbound_reg 1001@example.com'
fs_cli -x 'sofia status profile internal pres'
fs_cli -x 'PRESENCE_IN' | head                              # see events flowing
```

## Music on Hold

`mod_local_stream` reads a directory of audio files and serves them as a continuously-mixed stream — joined late, every listener hears the same content.

```xml
<!-- /etc/freeswitch/autoload_configs/local_stream.conf.xml -->
<configuration name="local_stream.conf" description="Local Stream">
  <directory name="default/8000" path="$${sounds_dir}/music/8000">
    <param name="rate" value="8000"/>
    <param name="shuffle" value="true"/>
    <param name="channels" value="1"/>
    <param name="interval" value="20"/>
    <param name="timer-name" value="soft"/>
  </directory>
  <directory name="default/16000" path="$${sounds_dir}/music/16000">
    <param name="rate" value="16000"/>
  </directory>
</configuration>
```

Use it:

```xml
<param name="hold-music" value="local_stream://moh"/>             <!-- profile -->
<action application="set" data="hold_music=local_stream://default"/>
<action application="playback" data="local_stream://default"/>     <!-- play directly -->
```

```bash
fs_cli -x 'show files'                                       <!-- active streams -->
fs_cli -x 'reload mod_local_stream'
```

## Speech (mod_unimrcp)

`mod_unimrcp` bridges FreeSWITCH to MRCPv1/v2 servers (for example UniMRCP server, AWS Polly, Microsoft Speech). Required for `play_and_get_speech`, `say` with TTS engines.

```xml
<!-- /etc/freeswitch/autoload_configs/unimrcp.conf.xml — excerpt -->
<configuration name="unimrcp.conf" description="UniMRCP">
  <settings>
    <param name="default-tts-profile" value="unimrcpserver"/>
    <param name="default-asr-profile" value="unimrcpserver"/>
  </settings>
  <profiles>
    <X-PRE-PROCESS cmd="include" data="mrcp_profiles/*.xml"/>
  </profiles>
</configuration>
```

Dialplan use:

```xml
<action application="speak" data="unimrcp:tts-profile|voice|Hello, world."/>
<action application="play_and_detect_speech"
        data="say:'Please say a number':detect:asr-profile {start-input-timers=true,no-input-timeout=5000}grammar.xml"/>
```

## ACL

`acl.conf.xml` defines named CIDR allow/deny lists. Profiles reference these via `apply-inbound-acl`, `apply-register-acl`, `apply-nat-acl`. Lists also rebuilt automatically (`localnet.auto`, `nat.auto`, `wan.auto`, `rfc1918.auto`).

```xml
<!-- /etc/freeswitch/autoload_configs/acl.conf.xml -->
<configuration name="acl.conf" description="Network Lists">
  <network-lists>
    <list name="trusted" default="deny">
      <node type="allow" cidr="10.0.0.0/8"/>
      <node type="allow" cidr="192.168.0.0/16"/>
      <node type="allow" cidr="203.0.113.5/32"/>             <!-- ITSP signaling IP -->
    </list>
    <list name="rfc1918" default="deny">
      <node type="allow" cidr="10.0.0.0/8"/>
      <node type="allow" cidr="172.16.0.0/12"/>
      <node type="allow" cidr="192.168.0.0/16"/>
    </list>
    <list name="domains" default="deny">
      <node type="allow" domain="$${domain}"/>
    </list>
  </network-lists>
</configuration>
```

Apply:

```xml
<param name="apply-inbound-acl" value="trusted"/>
<param name="apply-register-acl" value="trusted"/>
<param name="apply-nat-acl" value="nat.auto"/>
<param name="local-network-acl" value="localnet.auto"/>
```

```bash
fs_cli -x 'show acl'
fs_cli -x 'reloadacl'                                        # rebuild
fs_cli -x 'reloadxml; reloadacl'
```

## Variables

Two flavors: `$${var}` is preprocessor (set at XML parse time, can't change without `reloadxml`); `${var}` is runtime — channel variable lookup or global var.

```xml
<!-- in vars.xml: $${var} -->
<X-PRE-PROCESS cmd="set" data="domain=pbx.example.com"/>
<X-PRE-PROCESS cmd="exec-set" data="local_ip_v4=$(hostname -I | awk '{print $1}')"/>
<X-PRE-PROCESS cmd="env-set" data="public_key=PUB_KEY"/>

<!-- channel var via set -->
<action application="set" data="my_var=hello"/>
<action application="export" data="leg_timeout=30"/>           <!-- propagates to B leg -->
<action application="export" data="nolocal:something=val"/>     <!-- B leg only, not A -->

<!-- expansion -->
<action application="log" data="INFO ${caller_id_number} dialed ${destination_number}"/>

<!-- common channel variables -->
${uuid}                      <!-- channel UUID -->
${caller_id_name}            <!-- inbound CID name -->
${caller_id_number}          <!-- inbound CID number -->
${effective_caller_id_name}  <!-- presented to B leg -->
${effective_caller_id_number}
${origination_caller_id_name}
${destination_number}
${dialed_user}
${dialed_domain}
${domain_name}
${context}
${network_addr}              <!-- src IP -->
${sip_from_uri}
${sip_to_uri}
${sip_call_id}
${hangup_cause}
${start_stamp}, ${answer_stamp}, ${end_stamp}
${duration}, ${billsec}      <!-- seconds -->
${read_codec}, ${write_codec}
${rtp_audio_in_quality_percentage}
```

Helper functions inside `${...}`:

```
${cond(${duration} > 60 ? long : short)}
${strftime(%Y-%m-%d-%H-%M-%S)}
${expr(${count} + 1)}
${db(select/voicemail/box-1001)}
${user_data(1001@${domain} param password)}
${sofia_contact(*/1001@${domain})}
```

API:

```bash
fs_cli -x 'global_getvar domain'
fs_cli -x 'global_setvar default_password=newsecret'
fs_cli -x 'eval ${domain}'
fs_cli -x 'uuid_getvar <uuid> caller_id_number'
fs_cli -x 'uuid_setvar <uuid> hangup_after_bridge true'
```

## Common Errors verbatim

Verbatim console messages, what they mean, and the canonical fix.

```text
ERR  [sofia.c] Sofia profile 'internal' failed to start
```
Port already bound or `sip-port` clash. Check `ss -ulnp | grep 5060`. Stop conflicting service, or change `sip-port`. Then `sofia profile internal start`.

```text
NOTICE [sofia_reg.c] Failed Registration from sip:1001@10.0.0.5, ip 10.0.0.50
```
Wrong password, missing user in directory, or `apply-register-acl` blocking source. Check `fs_cli -x 'sofia loglevel all 9'` for full SIP. Confirm `user_data 1001@$${domain} param password`.

```text
WARNING [switch_core_state_machine.c] User_busy ... cause: NO_ROUTE_DESTINATION
```
Dialplan didn't match. Inspect with `sofia loglevel all 9` then check the *exact* `destination_number` and `context`. Often the call hit `public` but the dialplan match is in `default`.

```text
ERR  [sofia.c] USER_NOT_REGISTERED
```
Bridging `user/1001` but 1001 has no current registration. `sofia status profile internal reg` to confirm. If `register=false` for an upstream peer, dial via `sofia/gateway/<gw>/...`, not `user/...`.

```text
ERR  ... cause: DESTINATION_OUT_OF_ORDER
```
Endpoint reachable but rejecting (often the gateway's far-side carrier is failing). Compare with `sofia status gateway gw1`. May be DID not provisioned, or 503 from upstream.

```text
ERR  [mod_sofia.c] Gateway 'gw1' Down
```
Gateway state ≠ REGED. `sofia status gateway gw1`. Failure causes: `OPTIONS` ping timeout (network), 401 with no/wrong credentials, 403 (IP not whitelisted at carrier), 408 (carrier slow), `ping-max` exceeded.

```text
ERR  Gateway has no proxy or registrar
```
Missing both `register=true` *and* `proxy` in `<gateway>`. Add at least `proxy` for static routes; or `register=true` + credentials.

```text
WARNING [switch_rtp.c] Timeout waiting for RTP from 1.2.3.4:30000
```
NAT/firewall — RTP not flowing. Check `ext-rtp-ip`, RTP port range open in firewall, and STUN/TURN.

```text
ERR  [switch_core_codec.c] Cannot select a real codec for /tmp/foo.wav
```
Codec module not loaded. `show codec`; `load mod_opus`.

```text
ERR  XML PRE-PROCESSING ... unable to open file
```
`X-PRE-PROCESS include` glob is wrong or perm-denied. Re-check `data=` path and ownership of `/etc/freeswitch`.

## Hangup Cause Catalog

Names follow Q.850 ISDN cause codes. Used in CDRs, dialplan `continue_on_fail`, and `hangup` action data.

| Cause | Meaning |
|---|---|
| **NORMAL_CLEARING** (16) | Caller/callee hung up cleanly |
| **USER_BUSY** (17) | 486 Busy Here / busy tone |
| **NO_USER_RESPONSE** (18) | Endpoint reached, didn't reply (ringback fail) |
| **NO_ANSWER** (19) | Phone rang, no pickup within `leg_timeout` |
| **CALL_REJECTED** (21) | 603 Decline / explicit reject |
| **EXCHANGE_ROUTING_ERROR** (25) | Misrouted at switch |
| **INVALID_NUMBER_FORMAT** (28) | 484 Address Incomplete |
| **DESTINATION_OUT_OF_ORDER** (27) | Far end reachable but service down (508/503) |
| **NO_ROUTE_DESTINATION** (3) | Dialplan no match |
| **NO_ROUTE_TRANSIT_NET** (2) | Upstream gateway not configured |
| **RECOVERY_ON_TIMER_EXPIRE** (102) | Internal timer expired |
| **GATEWAY_DOWN** | Sofia gateway not registered/reachable |
| **SWITCH_CONGESTION** (42) | Resource exhaustion (max sessions hit) |
| **MANDATORY_IE_MISSING** (96) | Malformed SIP — missing required header/SDP |
| **NORMAL_TEMPORARY_FAILURE** (41) | Transient — retry candidate |
| **ATTENDED_TRANSFER** | Special — transferred away, A leg consumed |
| **REQUESTED_CHAN_UNAVAIL** (44) | Specific channel/trunk unavailable |
| **MEDIA_TIMEOUT** | RTP stopped flowing for `rtp-timeout-sec` |
| **PROGRESS_TIMEOUT** | No 18x within timer |
| **NORMAL_UNSPECIFIED** (31) | Catch-all clean hangup |
| **PROTOCOL_ERROR** (111) | SIP/SDP parse failure |
| **INTERWORKING** (127) | Cause not mappable to SIP |

```xml
<!-- continue dialplan after specific causes -->
<action application="set" data="continue_on_fail=USER_BUSY,NO_ANSWER,NORMAL_TEMPORARY_FAILURE"/>
<action application="set" data="hangup_after_bridge=true"/>
<action application="bridge" data="user/1001"/>
<action application="answer"/>                              <!-- runs only if bridge failed -->
<action application="voicemail" data="default ${domain_name} 1001"/>

<!-- explicit hangup with cause -->
<action application="hangup" data="CALL_REJECTED"/>
```

## Common Gotchas

Twelve broken-then-fixed traps that bite every FreeSWITCH operator.

### 1. `reloadxml` doesn't restart Sofia profiles

```bash
# BROKEN — edited internal.xml, rang reload, profile still has old settings
fs_cli -x 'reloadxml'
# old sip-port, old codecs still in effect

# FIXED — restart the profile (or rescan for gateway-only changes)
fs_cli -x 'sofia profile internal restart reloadxml'
fs_cli -x 'sofia profile external rescan'                  # gateway add only
```

`reloadxml` rebuilds the in-memory tree but does *not* re-bind sockets, re-apply codec lists, or change Sofia internals. Always `restart reloadxml` for profile-level changes.

### 2. Default password `1234` / `ClueCon` left in production

```xml
<!-- BROKEN — vanilla install ships these -->
<X-PRE-PROCESS cmd="set" data="default_password=1234"/>
<param name="password" value="ClueCon"/>                   <!-- event_socket -->

<!-- FIXED — change both, listen on loopback -->
<X-PRE-PROCESS cmd="set" data="default_password=$(openssl rand -base64 24)"/>
<param name="password" value="$(openssl rand -base64 24)"/>
<param name="listen-ip" value="127.0.0.1"/>
```

Public IPs with default creds get hijacked within hours by SIP scanners.

### 3. Port 5080 vs 5060 — wrong profile

```bash
# BROKEN — ITSP sends INVITE to 5060 (internal), gets challenged for auth
2024-... NOTICE Failed Registration from sip:itsp@1.2.3.4

# FIXED — ITSP traffic must hit external (5080), set proxy/register-proxy at trunk
fs_cli -x 'sofia status'                                   # confirm both listening
# carrier-side: configure their SBC to send to pbx.example.com:5080
```

`internal` profile *requires* registration and challenges every INVITE. Carrier traffic belongs on `external` (auth-calls=false, ACL-restricted instead).

### 4. `rtp-ip auto` picks the wrong NIC

```xml
<!-- BROKEN — dual-homed host, 'auto' chose the management NIC -->
<param name="rtp-ip" value="auto"/>
<param name="ext-rtp-ip" value="auto-nat"/>
<!-- One-way audio: SDP advertises mgmt IP, RTP arrives on data IP -->

<!-- FIXED — pin to the right interface -->
<param name="rtp-ip" value="10.0.1.5"/>                    <!-- data NIC -->
<param name="ext-rtp-ip" value="203.0.113.10"/>            <!-- public IP -->
<param name="sip-ip" value="10.0.1.5"/>
```

### 5. Codec mismatch — call connects, no audio

```text
WARNING [switch_core_codec.c] No codec match.
ERR  Cannot select a real codec
```

```xml
<!-- BROKEN — gateway pref PCMA only, profile pref OPUS only -->
<param name="codec-prefs" value="OPUS"/>

<!-- FIXED — overlap with PSTN baseline -->
<param name="codec-prefs" value="OPUS,G722,PCMU,PCMA"/>
<param name="inbound-codec-negotiation" value="generous"/>

<!-- Or per-call force -->
<action application="set" data="absolute_codec_string=PCMU,PCMA"/>
```

### 6. Context confusion — call routes to `public`, not `default`

```xml
<!-- BROKEN — extension defined in default context, but inbound from external lands in public -->
<context name="default">
  <extension name="ring_1001">...</extension>
</context>

<!-- FIXED — bridge from public into default for known DIDs, OR define in public -->
<context name="public">
  <extension name="public_did">
    <condition field="destination_number" expression="^\+?1?5551112222$">
      <action application="set" data="domain_name=${domain}"/>
      <action application="transfer" data="1001 XML default"/>
    </condition>
  </extension>
</context>
```

The `external` profile sets `context=public` for incoming calls. `default` is for authenticated, post-registration users.

### 7. One-way audio (RTP firewall)

```bash
# BROKEN — UFW open for SIP only, RTP blocked
sudo ufw status
# 5060/udp ALLOW

# FIXED — open the RTP range
sudo ufw allow 16384:32768/udp
```

```xml
<!-- and the right ext-rtp-ip -->
<param name="ext-rtp-ip" value="$${external_rtp_ip}"/>     <!-- not 'auto' on misdetected NAT -->
```

### 8. Registration drops every minute

```xml
<!-- BROKEN — phone behind NAT, expires too long, NAT pinhole expires -->
<param name="expire-seconds" value="3600"/>

<!-- FIXED — short rebind, force-rport on profile -->
<param name="expire-seconds" value="120"/>
<param name="NDLB-force-rport" value="server-only"/>
<param name="aggressive-nat-detection" value="true"/>
```

Most consumer routers expire UDP NAT bindings after 30–120s. Phone-side `register-expires` should be ≤ 60.

### 9. Verto 401/403 — credentials right but realm wrong

```javascript
// BROKEN — passing only a user, no domain
new Verto({ login: "1001", passwd: "secret" });

// FIXED — Verto auth always needs user@domain
new Verto({ login: "1001@pbx.example.com", passwd: "secret" });
```

Check `fs_cli -x 'show registrations'` — if it shows nothing for that user, the realm/domain wasn't matched.

### 10. SIP-over-WS: "TLS required" but binding is `ws`

```text
sip:status: 488 (Not Acceptable Here)
```

```xml
<!-- BROKEN — only ws (cleartext) -->
<param name="ws-binding" value=":5066"/>

<!-- FIXED — wss for browser-secure-context calls -->
<param name="wss-binding" value=":7443"/>
<param name="tls-cert-dir" value="$${certs_dir}"/>           <!-- needs wss.pem -->
```

Browsers (Chrome/Firefox) refuse WS to non-localhost from HTTPS pages. Use `wss://` with a real cert.

### 11. `originate` returns instantly with `-ERR NO_ANSWER`

```bash
# BROKEN — bgapi but reading wrong return
fs_cli -x 'originate user/1001 &echo'                       # blocks until call ends!

# FIXED — bgapi returns a Job-UUID immediately
fs_cli -x 'bgapi originate user/1001 &echo'
# Job-UUID: ...; result delivered as BACKGROUND_JOB event
```

`originate` blocks until the call hangs up (and reports its hangup_cause). Use `bgapi` for fire-and-forget.

### 12. `record_session` records only B leg

```xml
<!-- BROKEN — only B-leg audio captured -->
<action application="record_session" data="$${recordings_dir}/${uuid}.wav"/>

<!-- FIXED — set record_stereo before bridge, OR use mux mode -->
<action application="set" data="record_stereo=true"/>
<action application="set" data="RECORD_STEREO=true"/>
<action application="record_session" data="$${recordings_dir}/${uuid}.wav"/>

<!-- alt: use uuid_record from external trigger after CHANNEL_BRIDGE -->
fs_cli -x 'uuid_record <uuid> start /tmp/${uuid}.wav'
```

## Diagnostic Recipes

The exact sequence for the most common debugging questions.

```bash
# "Is FreeSWITCH up at all?"
fs_cli -x 'status'
fs_cli -x 'version'

# "What channels are active right now?"
fs_cli -x 'show channels'
fs_cli -x 'show calls'
fs_cli -x 'show registrations'
fs_cli -x 'show calls count'

# "Why won't user 1001 register?" — full SIP wire trace
fs_cli
freeswitch> /log 7
freeswitch> sofia loglevel all 9
freeswitch> sofia global siptrace on
# now ask 1001 to register; watch INVITE/REGISTER/401/200 OK
freeswitch> sofia loglevel all 0
freeswitch> sofia global siptrace off
freeswitch> /nolog

# "Why did call X drop?" — find UUID then dump
fs_cli -x 'show channels' | grep '5551234'
fs_cli -x 'uuid_dump <uuid>'                              # all channel variables
fs_cli -x 'uuid_getvar <uuid> hangup_cause'

# "Is the gateway up?"
fs_cli -x 'sofia status'
fs_cli -x 'sofia status gateway gw1'
# Expected: State: REGED  Ping-Status: REACH  Pings: ...

# "Console too noisy / not noisy enough"
fs_cli
freeswitch> /log 6                                         # info
freeswitch> /log 7                                         # debug — verbose
freeswitch> /log 0                                         # off
freeswitch> fsctl loglevel debug                           # core
freeswitch> console loglevel debug                         # console (mod_console)

# "Why did dialplan not match?"
fs_cli -x 'fsctl debug_level 9'                            # see all match decisions
# call again; watch: "Dialplan: ... matching at line X"
fs_cli -x 'fsctl debug_level 0'

# "DB-backed dialplan slow?"
fs_cli -x 'fsctl debug_sql'                                # log every SQL stmt
fs_cli -x 'fsctl debug_sql off'

# "Profile failed to start"
sudo journalctl -u freeswitch -n 200 | grep -i sofia
fs_cli -x 'sofia profile internal start'
ss -ulnp | grep 5060                                       # who else is listening
sudo ss -tulnp | grep -E ':5060|:5080'

# "Recording empty / silent"
ls -la /var/lib/freeswitch/recordings/
fs_cli -x 'uuid_dump <uuid>' | grep -i record
fs_cli -x 'uuid_setvar <uuid> RECORD_STEREO true'

# "What's the actual current value of $${domain}?"
fs_cli -x 'eval ${domain}'
fs_cli -x 'global_getvar domain'

# "Replay a stuck channel state"
fs_cli -x 'show channels'                                  # look for state STUCK or PARK
fs_cli -x 'uuid_kill <uuid>'

# "Hammer-test originate"
for i in {1..50}; do
  fs_cli -x "bgapi originate {origination_caller_id_number=test$i}user/1001 &echo"
done
fs_cli -x 'show calls count'
```

## Performance

- `max-sessions` (`switch.conf.xml`) — hard cap on concurrent calls. Default 1000.
- `sessions-per-second` — admission rate; bursts above this get 503.
- `rtp-port-min` / `-max` — must be wide enough for `2 × max-sessions` (in + out).
- File descriptor ulimit — every channel uses several fds; bump to 100k+.
- `min-idle-cpu` — refuse new calls when CPU idle drops below this percent.

```xml
<!-- /etc/freeswitch/autoload_configs/switch.conf.xml -->
<param name="max-sessions" value="5000"/>
<param name="sessions-per-second" value="100"/>
<param name="min-idle-cpu" value="20"/>
<param name="rtp-start-port" value="16384"/>
<param name="rtp-end-port" value="32768"/>
<param name="rtp-enable-zrtp" value="false"/>
<param name="loglevel" value="warning"/>                   <!-- console -->
```

```bash
fs_cli -x 'fsctl max_sessions 5000'                       # runtime change
fs_cli -x 'fsctl sps 100'
fs_cli -x 'fsctl min_idle_cpu 20'
fs_cli -x 'show status'
fs_cli -x 'fsctl pause'                                   # quiesce — refuse new calls
fs_cli -x 'fsctl resume'

# OS tuning
echo 'net.core.rmem_max=33554432' | sudo tee -a /etc/sysctl.d/freeswitch.conf
echo 'net.core.wmem_max=33554432' | sudo tee -a /etc/sysctl.d/freeswitch.conf
echo 'net.ipv4.udp_rmem_min=8192' | sudo tee -a /etc/sysctl.d/freeswitch.conf
echo 'net.ipv4.udp_wmem_min=8192' | sudo tee -a /etc/sysctl.d/freeswitch.conf
sudo sysctl --system
```

Per-call cost (rule of thumb): ~150 KB RAM + a few % of one core for transcoding (G.711 passthrough is nearly free; OPUS↔G.729 transcoding is expensive — pin enough cores).

## Security

- **Passwords** — strong unique per-user; rotate `default_password` away from `1234`. `vm-password` and SIP password should differ.
- **ESL** — bind `127.0.0.1` only; if remote ESL is needed, tunnel via SSH or stunnel.
- **ACL** — restrict `apply-register-acl`/`apply-inbound-acl` to known networks/CIDRs; never expose `internal` profile to public IP without ACL.
- **fail2ban** — pattern-match `Failed Registration from` and `Auth failure` in `/var/log/freeswitch/freeswitch.log`.

```ini
# /etc/fail2ban/filter.d/freeswitch.conf
[Definition]
failregex = ^.+Failed Registration from .* sip:.+ ip <HOST>.*$
            ^.+SIP auth failure .* from <HOST>.*$
ignoreregex =
```

```ini
# /etc/fail2ban/jail.d/freeswitch.local
[freeswitch]
enabled = true
filter = freeswitch
action = iptables-allports[name=freeswitch]
logpath = /var/log/freeswitch/freeswitch.log
maxretry = 5
findtime = 600
bantime = 3600
```

- **TLS / SRTP** — encrypt SIP and media end-to-end:

```xml
<param name="tls" value="true"/>
<param name="tls-bind-params" value="transport=tls"/>
<param name="tls-sip-port" value="5061"/>
<param name="tls-cert-dir" value="$${certs_dir}"/>
<param name="tls-version" value="tlsv1.2,tlsv1.3"/>
<param name="tls-ciphers" value="ALL:!ADH:!LOW:!EXP:!MD5:@STRENGTH"/>
<param name="tls-only" value="false"/>

<param name="rtp-secure-media" value="mandatory"/>
<param name="rtp-secure-media-inbound" value="mandatory"/>
<param name="rtp-secure-media-outbound" value="mandatory"/>
<!-- mandatory | optional | forbidden — pairs SAVP/SAVPF with SDP crypto -->
```

```xml
<!-- per-call force encryption -->
<action application="set" data="sip_secure_media=mandatory"/>
<action application="export" data="sip_secure_media=mandatory"/>
<action application="set" data="rtp_secure_media=mandatory"/>
```

- **Cert layout** — `$${certs_dir}/agent.pem` (combined cert+key), `wss.pem` for Verto. Let's Encrypt:

```bash
sudo certbot certonly --standalone -d pbx.example.com
sudo cat /etc/letsencrypt/live/pbx.example.com/{privkey,fullchain}.pem | \
  sudo tee /etc/freeswitch/tls/agent.pem
sudo chown freeswitch:freeswitch /etc/freeswitch/tls/agent.pem
sudo chmod 600 /etc/freeswitch/tls/agent.pem
fs_cli -x 'sofia profile internal restart reloadxml'
```

- **Toll-fraud** — restrict outbound dial: per-user `toll_allow` variable + dialplan check `<condition field="${toll_allow}" expression="international">`. Reject `^011\d{6,}$` or `^00\d{6,}$` patterns by default.
- **Rate-limit** — `mod_limit` or `limit_hash` to cap per-user/per-account concurrent or per-second:

```xml
<action application="limit" data="hash realm $caller_id_number 3 !USER_BUSY"/>
<action application="limit_hash" data="account ${accountcode} 5 !SWITCH_CONGESTION"/>
```

- **Disable open proxying** — `auth-calls=true` and `accept-blind-auth=false` on `internal`; `auth-all-packets=false` only for trusted ITSPs.
- **Log + monitor** — ship `freeswitch.log` to a central system; alert on `Failed Registration` rate, `GATEWAY_DOWN`, and unusual `originate` volumes.

## Idioms

```xml
<!-- Hangup B leg if A leg disappears (the default but be explicit) -->
<action application="set" data="hangup_after_bridge=true"/>

<!-- Try voicemail only if the user is offline/busy/no-answer -->
<action application="set" data="continue_on_fail=USER_BUSY,NO_ANSWER,USER_NOT_REGISTERED,NORMAL_TEMPORARY_FAILURE"/>
<action application="bridge" data="user/$1@${domain_name}"/>
<action application="answer"/>
<action application="sleep" data="500"/>
<action application="voicemail" data="default ${domain_name} $1"/>

<!-- Ringback while bridging across trunk -->
<action application="set" data="ringback=$${us-ring}"/>
<action application="set" data="transfer_ringback=$${hold_music}"/>

<!-- Force codec on this call only (transcoding off) -->
<action application="set" data="absolute_codec_string=PCMU,PCMA"/>

<!-- Whisper / barge a target call -->
fs_cli -x 'originate user/9999 &eavesdrop(<target-uuid>)'
<!-- DTMF inside: 0 mute / 1 barge / 2 clear / 3 both -->

<!-- Schedule a hangup -->
<action application="sched_hangup" data="+3600 ALLOTTED_TIMEOUT"/>

<!-- Loopback for "redial through dialplan" -->
<action application="bridge" data="loopback/+15551234567/default"/>

<!-- Capture entire dialplan as Lua call -->
<extension name="lua_inline">
  <condition field="destination_number" expression="^.+$">
    <action application="lua" data="route.lua"/>
  </condition>
</extension>

<!-- Inbound/outbound CDR with duration -->
${strftime(${start_stamp}|%F %T)} ${caller_id_number} → ${destination_number} ${billsec}s ${hangup_cause}

<!-- Bind dialplan match to an account/tenant -->
<condition field="${sip_to_user}" expression="^${effective_caller_id_number}$">
  <action application="set" data="accountcode=${user_data(${dialed_user}@${domain_name} param accountcode)}"/>
</condition>
```

## See Also

- asterisk
- sip-protocol
- rtp-sdp
- ip-phone-provisioning
- sip-trunking

## References

- FreeSWITCH project: https://freeswitch.org/
- Source/issues: https://github.com/signalwire/freeswitch
- Confluence (legacy wiki, still authoritative for many APIs): https://developer.signalwire.com/freeswitch/
- Sofia-SIP: http://sofia-sip.sourceforge.net/
- mod_sofia parameter reference: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_sofia/
- mod_dptools application reference: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Dialplan/
- ESL (Event Socket Library) docs: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_event_socket_1048924/
- mod_lua: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_lua/
- mod_conference: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_conference_1048864/
- mod_callcenter: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Modules/mod_callcenter_1048948/
- mod_verto: https://developer.signalwire.com/freeswitch/FreeSWITCH-Explained/Client-and-Developer-Interfaces/Verto/
- ITU-T Q.850 cause codes (mapped to SIP cause): https://www.itu.int/rec/T-REC-Q.850
- RFC 3261 — SIP: https://www.rfc-editor.org/rfc/rfc3261
- RFC 3550 — RTP: https://www.rfc-editor.org/rfc/rfc3550
- RFC 4568 — SDP Crypto / SRTP: https://www.rfc-editor.org/rfc/rfc4568
- RFC 8866 — SDP: https://www.rfc-editor.org/rfc/rfc8866
- RFC 7118 — SIP over WebSocket: https://www.rfc-editor.org/rfc/rfc7118
