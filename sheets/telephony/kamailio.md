# Kamailio

High-performance, scriptable open-source SIP server (proxy, registrar, redirect, presence, LB) used as carrier-grade infrastructure handling thousands of CPS.

## Setup

Kamailio descends from the original SIP Express Router (SER), authored at the Fraunhofer Institute FOKUS in 2001 by Andrei Pelinescu-Onciul, Jiri Kuthan, Daniel-Constantin Mierla, Bogdan-Andrei Iancu, and others. The codebase forked in 2005 into OpenSER, which was renamed Kamailio in 2008 after a trademark dispute. A reconciliation in 2008 merged Kamailio and another SER-derived branch back together; for several releases the project shipped jointly as "Kamailio (former OpenSER)" and "SER" with a shared core. Today Kamailio (kamailio.org) and OpenSIPS (opensips.org) are the two surviving active forks; they share heritage and config-language similarity but diverged significantly in module ecosystem and internals.

Install on Debian/Ubuntu via the official repos:

```bash
# add upstream APT repo
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 0x416A0BD7
echo "deb http://deb.kamailio.org/kamailio58 bookworm main" | sudo tee /etc/apt/sources.list.d/kamailio.list
sudo apt update
sudo apt install kamailio kamailio-mysql-modules kamailio-tls-modules kamailio-websocket-modules
sudo systemctl enable --now kamailio
```

Build from source for custom modules:

```bash
git clone --depth=1 https://github.com/kamailio/kamailio.git
cd kamailio
make include_modules="db_mysql tls websocket rtpengine dispatcher presence" cfg
make all
sudo make install
```

Verify version:

```bash
kamailio -V
# version: kamailio 5.8.4 (x86_64/linux)
# flags: STATS: Off, USE_TCP, USE_TLS, USE_SCTP, ...
# Compiled on 14:23:01 Mar 12 2025 with gcc 12.2.0
```

The binary searches `/etc/kamailio/kamailio.cfg` by default; override with `-f`. The init script lives at `/etc/init.d/kamailio` (sysv) or `/lib/systemd/system/kamailio.service` (systemd).

## Architecture

Kamailio is a multi-process, shared-memory SIP server. The main process forks a configurable number of UDP receivers (`children`), TCP receivers (`tcp_children`), TLS workers, timer processes, and a single management process. Each child runs its own instance of the script interpreter; SIP messages dispatch to whichever child reads from the listening socket. Inter-process state (transactions, registrations, dialog hash) lives in System V shared memory or POSIX shm; locks (futexes/sysv sem) protect it.

Routing is expressed in `kamailio.cfg`, a C-like scripting language with route blocks. Top-level `request_route` runs on every incoming SIP request; `reply_route` runs on every reply; `failure_route[n]`, `branch_route[n]`, `onreply_route[n]`, `event_route[]` handle specific phases. Routes invoke loaded module functions (`t_relay()`, `save("location")`, `rtpengine_manage()`) and manipulate pseudo-variables (`$ru`, `$si`, `$rU`, `$tu`, `$ci`).

```
   +-----------------------+
   |  Main / Management    |
   +-----------+-----------+
               |
   +-----------+-----------+----------+----------+
   |           |           |          |          |
 udp[0]     udp[1]      tcp[0]     tls[0]     timer
 udp[2]     udp[3]      tcp[1]                 evapi
                          \\                    rtimer
                          shm  <-- usrloc, tm hash, dialogs, htable
```

Transactions, dialogs, registrations, statistics, and locks are all in shm; restarting Kamailio loses all in-memory state unless backed by `db_mode=2` (write-through) or `db_mode=3` (write-back) to a SQL backend.

## kamailio.cfg

The config file is parsed once at startup. Top section sets globals; `loadmodule`/`modparam` block loads modules; route blocks define logic. Comments are `#` (full line) or `/* … */` (block). Strings are double-quoted. Numbers can be hex (`0x...`), decimal, or octal.

```kamailio
#!KAMAILIO

# global parameters
debug=2
log_stderror=no
log_facility=LOG_LOCAL0
fork=yes
children=8
tcp_children=4
auto_aliases=no
listen=udp:eth0:5060
listen=tcp:eth0:5060
listen=tls:eth0:5061
alias="sip.example.com"
mpath="/usr/lib/x86_64-linux-gnu/kamailio/modules/"

loadmodule "tm.so"
loadmodule "sl.so"
loadmodule "rr.so"
loadmodule "registrar.so"
loadmodule "usrloc.so"

modparam("usrloc", "db_mode", 2)
modparam("usrloc", "db_url", "mysql://kamailio:pwd@localhost/kamailio")

request_route {
    if (!mf_process_maxfwd_header("10")) {
        sl_send_reply("483", "Too Many Hops");
        exit;
    }
    if (is_method("REGISTER")) {
        save("location");
        exit;
    }
    if (loose_route()) {
        t_relay();
        exit;
    }
    lookup("location");
    t_relay();
}
```

Conditional preprocessing uses `#!define`, `#!ifdef`, `#!ifndef`, `#!trydef`, `#!endif`. The `#!substdef` directive performs string substitution. The shebang `#!KAMAILIO` (or `#!OPENSIPS`, `#!MAXCOMPAT`) selects compatibility mode for legacy syntax. Variable references inside strings use `$var(name)` or shorthand `$ru`, `$si`, `$rU`.

## Loadable Modules

`loadmodule "name.so"` loads a shared object from `mpath` (or absolute path). `modparam("module", "param", value)` sets a parameter — value type must match (string in quotes, integer bare). Order matters: a module's deps must load first (e.g. `tm` before `dialog`, `usrloc` before `registrar`).

```kamailio
mpath="/usr/lib/x86_64-linux-gnu/kamailio/modules/"

loadmodule "sl.so"           # stateless replies
loadmodule "tm.so"           # transactions
loadmodule "rr.so"           # record-route
loadmodule "maxfwd.so"
loadmodule "textops.so"
loadmodule "siputils.so"
loadmodule "xlog.so"
loadmodule "sanity.so"
loadmodule "ctl.so"
loadmodule "kex.so"
loadmodule "pv.so"
loadmodule "usrloc.so"
loadmodule "registrar.so"
loadmodule "auth.so"
loadmodule "auth_db.so"
loadmodule "permissions.so"
loadmodule "dispatcher.so"
loadmodule "rtpengine.so"
loadmodule "htable.so"
loadmodule "dialog.so"
loadmodule "nathelper.so"
loadmodule "presence.so"
loadmodule "presence_xml.so"
loadmodule "websocket.so"
loadmodule "tls.so"

modparam("tm", "fr_timer", 30000)
modparam("tm", "fr_inv_timer", 120000)
modparam("registrar", "method_filtering", 1)
modparam("rr", "enable_full_lr", 1)
modparam("rr", "append_fromtag", 0)
```

Some modules accept `modparam` for hash-style key/value strings using `=>` syntax. `loadpath "dir"` (alternative to `mpath`) sets a colon-separated list of search paths. `cfgengine "kamailio"` selects the script engine; alternatives include Lua, Python, JavaScript, Squirrel, Ruby (via `app_lua`, `app_python3`, `app_jsdt`, etc.).

## tm Module

The Transaction Manager handles SIP transactions: matching INVITE/responses, retransmissions, fork/parallel-forking, branch handling, INVITE timer (Timer A/B/C/D from RFC 3261). `t_relay()` is the workhorse — sends the request statefully and creates branches that fire `branch_route`, `onreply_route`, `failure_route`.

```kamailio
loadmodule "tm.so"
modparam("tm", "fr_timer", 30000)         # non-INVITE timer (Timer F)
modparam("tm", "fr_inv_timer", 120000)    # INVITE final response timer
modparam("tm", "max_inv_lifetime", 180000)
modparam("tm", "wt_timer", 5000)          # wait timer (Timer J)

request_route {
    t_on_failure("MANAGE_FAILURE");
    t_on_branch("MANAGE_BRANCH");
    t_on_reply("MANAGE_REPLY");
    if (!t_relay()) {
        sl_reply_error();
    }
    exit;
}

failure_route[MANAGE_FAILURE] {
    if (t_check_status("486|408|480")) {
        # try voicemail
        $ru = "sip:vm@" + $fd;
        append_branch();
        t_relay();
        exit;
    }
}
```

Useful functions: `t_newtran()` (create txn explicitly), `t_check_trans()` (txn matching for late-arrived requests), `t_reply("486","Busy Here")` (stateful reply), `t_lookup_request()`, `t_branch_timeout()`, `t_branch_replied()`, `t_local_replied()`, `t_is_canceled()`, `t_set_fr(inv,non)` (per-txn timer override).

The transaction hash size is set with `modparam("tm","hash_size", 4096)` (must be power of 2). Memory pressure is the leading scaling concern — each txn consumes ~3-5 KB shm.

## Registrar Module

Implements RFC 3261 §10 — REGISTER processing. `save("location")` extracts contact, expiry, q-value, path from a REGISTER, validates, and stores into `usrloc`. `lookup("location")` rewrites Request-URI from registered contact for routing.

```kamailio
loadmodule "registrar.so"
modparam("registrar", "method_filtering", 1)     # only forward methods AOR supports
modparam("registrar", "max_contacts", 5)         # per AOR
modparam("registrar", "default_expires", 3600)
modparam("registrar", "min_expires", 60)
modparam("registrar", "max_expires", 7200)
modparam("registrar", "max_username_size", 64)
modparam("registrar", "received_avp", "$avp(received)")
modparam("registrar", "use_path", 1)
modparam("registrar", "path_mode", 2)            # require Path support

request_route {
    if (is_method("REGISTER")) {
        # auth happens before save (see auth section)
        if (!save("location")) {
            sl_reply_error();
        }
        exit;
    }
    if (!lookup("location")) {
        switch ($rc) {
            case -1:
            case -3: send_reply("404", "Not Found"); exit;
            case -2: send_reply("405", "Method Not Allowed"); exit;
        }
    }
    t_relay();
}
```

The `$rc` (return code) values: `1` = success, `-1` = AOR not found, `-2` = method not allowed for AOR, `-3` = no contacts. `lookup_branches()` after `lookup()` parallel-forks if multiple contacts. `unregister("location","sip:user@d")` removes a contact programmatically.

`reg_fetch_contacts("location","sip:u@d","caller")` populates `$ulc(caller=>...)` to inspect contacts without rewriting Request-URI. `registered("location","sip:u@d")` returns true if AOR has live contacts — useful for routing decisions.

## usrloc Module

Backs the registrar; manages the in-memory contact table optionally synced to SQL. `db_mode` selects persistence: `0` = pure memory, `1` = write-through (every change flushed), `2` = write-back (sync at intervals), `3` = DB-only (no shm cache, slow).

```kamailio
loadmodule "usrloc.so"
modparam("usrloc", "db_mode", 2)
modparam("usrloc", "db_url", "mysql://kamailio:secret@db/kamailio")
modparam("usrloc", "timer_interval", 60)
modparam("usrloc", "timer_procs", 1)
modparam("usrloc", "use_domain", 0)
modparam("usrloc", "matching_mode", 0)         # 0=contact, 1=callid, 2=path
modparam("usrloc", "hash_size", 12)            # 2^12 = 4096 buckets
modparam("usrloc", "preload", "location")
modparam("usrloc", "nat_bflag", 6)
```

The SQL table is `location` by default. `preload "location"` loads all contacts at startup so concurrent requests don't trigger DB lookups during warm-up. Multiple contacts per AOR are kept (forking targets); the table key is `(username, domain, contact, callid)`.

Inspect via kamcmd:

```bash
kamcmd ul.dump
kamcmd ul.lookup location alice
kamcmd ul.rm location alice
kamcmd ul.add location alice sip:alice@10.0.0.5:5060 3600 0.5 ...
```

NAT-detected contacts get the `nat_bflag` set on the contact record so routing can branch-flag them later for `fix_contact()` rewriting.

## Auth + auth_db

Implements HTTP Digest Authentication (RFC 2617/7616) for SIP. `auth` provides primitives; `auth_db` reads credentials from SQL. `pv_auth` allows credentials in pseudo-variables; `auth_radius`, `auth_xkeys`, `auth_diameter` are alternatives.

```kamailio
loadmodule "auth.so"
loadmodule "auth_db.so"
modparam("auth_db", "db_url", "mysql://kamailio:pwd@localhost/kamailio")
modparam("auth_db", "calculate_ha1", 1)     # use plaintext password column
modparam("auth_db", "password_column", "password")
modparam("auth_db", "load_credentials", "$avp(email)=email_address;$avp(rpid)=rpid")
modparam("auth", "nonce_expire", 30)

route[AUTH] {
    if (is_method("REGISTER") || from_uri==myself) {
        if (!auth_check("$fd", "subscriber", "1")) {
            auth_challenge("$fd", "0");
            exit;
        }
        # remove credentials from forwarded request
        consume_credentials();
    }
}
```

`auth_check(realm, table, flag)` does the lookup + nonce verify in one call. `flag=1` verifies that the From username matches the auth username (anti-spoof). `auth_challenge(realm, flags)` sends 401 (REGISTER) or 407 (other) with WWW-Authenticate. `consume_credentials()` strips Proxy-Authorization so downstream peers don't see it.

`subscriber` schema:

```sql
CREATE TABLE subscriber (
  id INT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  username VARCHAR(64) NOT NULL,
  domain VARCHAR(64) NOT NULL,
  password VARCHAR(64) NOT NULL DEFAULT '',
  ha1 VARCHAR(128) NOT NULL DEFAULT '',
  ha1b VARCHAR(128) NOT NULL DEFAULT '',
  email_address VARCHAR(128) NOT NULL DEFAULT '',
  rpid VARCHAR(128) DEFAULT NULL,
  UNIQUE KEY (username, domain)
);
```

For SHA-256 use `algorithm=SHA-256` (RFC 7616) and store HA1 column derived with the new hash. `nonce_count=1` enables NC tracking to defeat replay.

## Permissions Module

IP-based access control list (ACL): trust certain peer IPs to bypass auth (carrier trunks), block others. Stores rules in `address` (with `grp` group) and `trusted` (regex on From URI/header).

```kamailio
loadmodule "permissions.so"
modparam("permissions", "db_url", "mysql://kamailio:pwd@localhost/kamailio")
modparam("permissions", "db_mode", 1)
modparam("permissions", "address_table", "address")
modparam("permissions", "trusted_table", "trusted")

route[CHECK_SOURCE] {
    if (allow_source_address("1")) {
        # peer is in group 1 — trusted carrier
        return 1;
    }
    if (allow_trusted("$si", "$proto")) {
        return 1;
    }
    return -1;
}

request_route {
    if (route(CHECK_SOURCE) == 1) {
        # bypass auth; route directly
        route(RELAY);
        exit;
    }
    route(AUTH);
    route(RELAY);
}
```

`address` schema:

```sql
CREATE TABLE address (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  grp INT UNSIGNED NOT NULL DEFAULT 1,
  ip_addr VARCHAR(50) NOT NULL,
  mask INT NOT NULL DEFAULT 32,
  port INT UNSIGNED NOT NULL DEFAULT 0,
  tag VARCHAR(64),
  UNIQUE KEY (grp, ip_addr, mask, port)
);
```

`allow_address(grp, ip, port)` lets you check arbitrary IP. Refresh with `kamcmd address.reload`. `allow_uri()` checks From/RURI against URI table; `allow_register("register.allow","register.deny")` checks regex files.

## Dispatcher Module

Load-balance SIP traffic across a pool of destinations (PBXes, app servers, SBCs) with health-checking via OPTIONS pings. Configurable algorithms: round-robin, hash on Call-ID, hash on From, weighted, prio, random, sticky.

```kamailio
loadmodule "dispatcher.so"
modparam("dispatcher", "db_url", "mysql://kamailio:pwd@localhost/kamailio")
modparam("dispatcher", "ds_ping_interval", 30)
modparam("dispatcher", "ds_probing_threshold", 3)
modparam("dispatcher", "ds_inactive_threshold", 1)
modparam("dispatcher", "ds_ping_method", "OPTIONS")
modparam("dispatcher", "ds_ping_from", "sip:probe@kamailio.example.com")
modparam("dispatcher", "ds_probing_mode", 1)        # 1=ping all, 2=ping inactive too
modparam("dispatcher", "flags", 2)                  # set BFlag on dispatched

route[DISPATCH] {
    if (!ds_select_dst("1", "4")) {                 # group 1, alg 4 = round-robin
        send_reply("503", "No destination available");
        exit;
    }
    t_on_failure("DISPATCH_FAILURE");
    t_relay();
}

failure_route[DISPATCH_FAILURE] {
    if (t_check_status("5[0-9][0-9]|408")) {
        if (ds_next_dst()) {
            t_on_failure("DISPATCH_FAILURE");
            t_relay();
            exit;
        }
    }
}
```

Algorithms (`alg` argument): `0` hash CallID, `1` hash From, `2` hash To, `3` hash RURI, `4` round-robin, `5` random, `6` hash PV, `7` weight, `8` call-load (CL flag), `9` relative weight, `10` parallel forking, `11` priority list. `ds_next_dst()` retries with the next gateway from the same group.

`dispatcher` schema:

```sql
CREATE TABLE dispatcher (
  id INT AUTO_INCREMENT PRIMARY KEY,
  setid INT NOT NULL DEFAULT 0,
  destination VARCHAR(192) NOT NULL,
  flags INT NOT NULL DEFAULT 0,
  priority INT NOT NULL DEFAULT 0,
  attrs VARCHAR(128),
  description VARCHAR(64) NOT NULL DEFAULT ''
);
```

Reload runtime: `kamcmd dispatcher.reload`. List: `kamcmd dispatcher.list`.

## Dialog Module

Tracks full dialogs (INVITE/200OK/ACK through BYE) — needed for stateful per-call counting, billing CDR triggers, blind transfer mid-call, dialog-specific timers, and CPS limiting.

```kamailio
loadmodule "dialog.so"
modparam("dialog", "dlg_flag", 4)               # bit 4 marks dialog tracking
modparam("dialog", "default_timeout", 21600)    # 6h
modparam("dialog", "db_mode", 1)                # 0=mem, 1=load on start, 2=delayed write, 3=full DB
modparam("dialog", "db_url", "mysql://...")
modparam("dialog", "profiles_with_value", "calls;peer")
modparam("dialog", "profiles_no_value", "inbound;outbound")
modparam("dialog", "rr_param", "did")           # RR parameter for dialog ID

request_route {
    if (is_method("INVITE")) {
        setflag(4);                              # dlg_flag
        dlg_manage();
    }
}

event_route[dialog:start] {
    xlog("L_INFO", "dialog start: $ci\n");
    set_dlg_profile("calls", "$fU");
    if (get_profile_size("calls", "$fU") > 5) {
        sl_send_reply("486", "Too Many Calls");
        dlg_terminate("all", "limit reached");
        exit;
    }
}

event_route[dialog:end] {
    xlog("L_INFO", "dialog end: $ci dur=$DLG_lifetime\n");
}
```

CPS / concurrent-call limiting via `dlg_profile` is the canonical use. `dlg_bye("all"|"caller"|"callee", "reason")` ends a call. `dlg_get(callid, fromtag, totag)` looks up an existing dialog by triplet. `kamcmd dlg.list` enumerates active dialogs.

Dialog state machine: `Init → Early (1xx) → Confirmed (200 OK + ACK) → Terminated (BYE)`. The module emits `dialog:start`, `dialog:end`, `dialog:failed`, `dialog:expired` events. Memory cost is ~1-2 KB per dialog; 10k concurrent calls ≈ 20 MB shm.

## NAT Helper

The `nathelper` module provides primitives to detect and rewrite NATted contacts. SIP signaling embeds private RFC 1918 IPs in Via, Contact, and SDP (`c=`, `m=`); `fix_nated_contact()`, `fix_nated_register()`, `fix_nated_sdp()` rewrite them with the source IP and port observed by the proxy.

```kamailio
loadmodule "nathelper.so"
modparam("nathelper", "natping_interval", 30)
modparam("nathelper", "natping_processes", 1)
modparam("nathelper", "ping_nated_only", 1)
modparam("nathelper", "received_avp", "$avp(received)")
modparam("nathelper", "sipping_from", "sip:pinger@kamailio.example.com")
modparam("nathelper", "sipping_method", "OPTIONS")
modparam("registrar", "received_avp", "$avp(received)")

route[NATDETECT] {
    force_rport();
    if (nat_uac_test("19")) {                   # bitmask: 1=Via, 2=Contact, 16=Source-Contact mismatch
        if (is_method("REGISTER")) {
            fix_nated_register();
        } else {
            fix_nated_contact();
        }
        setbflag(6);                            # tag branch as NATted
    }
}
```

`nat_uac_test()` test bits: `1` = Contact/RFC1918, `2` = Via/RFC1918, `4` = source-Contact host mismatch, `8` = source-Contact port mismatch, `16` = source-Via host:port mismatch, `32` = Contact rport. Bitmask `19` (1+2+16) is the common combo. NAT pings (`OPTIONS` or empty UDP keepalives) keep firewall pinholes open at typical 30s interval.

`add_contact_alias()` / `handle_ruri_alias()` is the modern preferred mechanism: appends `;alias=ip~port~proto` to Contact so subsequent in-dialog requests can `handle_ruri_alias()` to restore the routed destination — survives storage-and-replay better than direct fix_nated_contact().

## RTPengine Module

External media relay/transcoder controlled via NG protocol over UDP (port 22222). Solves SIP-trapezoid-doesn't-include-media problem: SIP signaling routes through Kamailio, but RTP/SRTP must traverse a media-aware proxy because of NAT, codec mismatch, transcoding, recording, ICE, DTLS-SRTP for WebRTC.

```kamailio
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")
modparam("rtpengine", "extra_id_pv", "$avp(extra_id)")

request_route {
    if (has_body("application/sdp")) {
        rtpengine_manage("replace-origin replace-session-connection ICE=force");
    }
    t_on_reply("MANAGE_MEDIA");
    t_relay();
}

onreply_route[MANAGE_MEDIA] {
    if (has_body("application/sdp")) {
        rtpengine_manage();
    }
}

route[BYE] {
    if (is_method("BYE") || is_method("CANCEL")) {
        rtpengine_delete();
    }
}
```

`rtpengine_manage()` auto-detects offer vs answer based on method + direction. Flags: `replace-origin` (rewrite SDP `o=`), `replace-session-connection` (rewrite `c=`), `ICE=force` (force ICE for WebRTC), `RTP/SAVPF` (force SRTP+feedback), `DTLS=passive`, `record-call=yes`, `transcode-PCMA`, `transcode-OPUS`, `media-address=1.2.3.4`. `rtpengine_offer()` and `rtpengine_answer()` give explicit control vs auto-`manage`.

Tear down with `rtpengine_delete()` on `BYE`/`CANCEL`. Not calling delete leaks the media port until idle timeout (~5min default). The kamcmd RPC `kamcmd rtpengine.show all` lists active calls; `kamcmd rtpengine.reload` re-reads sock list; `rtpengine.ping` health-checks.

## SQLops

Module `sqlops` (or `sql`) provides ad-hoc SQL queries from script — bypassing the per-module `db_url`. Useful for routing tables, custom rate plans, on-the-fly enumerations. Does not pool connections per request: rely on a connection pool defined in `db_*` config.

```kamailio
loadmodule "sqlops.so"
modparam("sqlops", "sqlcon", "ca=>mysql://kamailio:pwd@localhost/kamailio")

route[CHECK_PLAN] {
    sql_query("ca", "SELECT plan_id, max_cps FROM customer WHERE id='$var(cid)'", "ra");
    if ($dbr(ra=>rows) == 0) {
        sl_send_reply("403", "Unknown customer");
        exit;
    }
    $var(plan)  = $dbr(ra=>[0,0]);
    $var(maxcps) = $dbr(ra=>[0,1]);
    sql_result_free("ra");
}
```

`$dbr(name=>rows)` row count, `$dbr(name=>cols)` column count, `$dbr(name=>[i,j])` cell. `sql_query_async()` is non-blocking. `sql_xquery()` accepts xavp values and returns into xavp containers, which is more idiomatic for multi-row processing. Always `sql_result_free()` to release shm.

Multiple connections: `modparam("sqlops","sqlcon","ca=>mysql://..."); modparam("sqlops","sqlcon","cb=>postgres://...")`.

## HTable

In-memory hash tables — the workhorse for rate-limiting counters, blocklists, dispatcher caches, and short-lived state. Optionally backed by SQL for persistence across restart.

```kamailio
loadmodule "htable.so"
modparam("htable", "htable", "ipban=>size=8;autoexpire=300;dbtable=ipban;")
modparam("htable", "htable", "cps=>size=10;autoexpire=1;")
modparam("htable", "htable", "auth_attempts=>size=8;autoexpire=900;dbtable=auth_attempts;dmqreplicate=1;")

route[CHECK_RATE] {
    $var(key) = $si + ":" + $rm;
    $sht(cps=>$var(key)) = $sht(cps=>$var(key)) + 1;
    if ($sht(cps=>$var(key)) > 50) {
        xlog("L_WARN", "rate limit hit: $var(key)\n");
        sl_send_reply("503", "Rate Limited");
        exit;
    }
}
```

Operations: `$sht(name=>key)` read/write, `$shtex(name=>key)` returns expire time, `sht_lock("name=>key")` advisory lock, `sht_inc("name=>key", 1)` atomic increment, `sht_match_name("name", "value=", "regex")` regex iter.

`autoexpire=N` per-key expiry seconds; `size=N` is power-of-two-bucket count `2^N`. `dmqreplicate=1` synchronizes via DMQ across cluster nodes. `kamcmd htable.dump ipban`, `kamcmd htable.reload ipban`, `kamcmd htable.delete ipban 1.2.3.4`.

## Pike

Anti-flood / DoS protection: detects request floods per source IP and triggers a configurable response (drop, blocklist, alarm). Sliding-window leaky bucket.

```kamailio
loadmodule "pike.so"
modparam("pike", "sampling_time_unit", 2)       # window = 2 seconds
modparam("pike", "reqs_density_per_unit", 16)   # threshold 16 reqs/sec/ip
modparam("pike", "remove_latency", 4)           # red-zone delay before reset

request_route {
    if (!pike_check_req()) {
        xlog("L_ALERT", "PIKE: source $si flooding\n");
        $sht(ipban=>$si) = 1;
        exit;
    }
    if ($sht(ipban=>$si) != $null) {
        sl_send_reply("403", "Banned");
        exit;
    }
}
```

Pike uses an internal trie keyed by IP fragments (`/24` subnet smoothing) so distributed floods from one /24 still trigger. Combine with htable to persist bans beyond pike's internal window. The `sanity` module rejects malformed SIP early to avoid expensive parsing on bad input.

For more sophisticated rate limiting use `ratelimit` module (token bucket per pipe) or `pipelimit` (queue-based shaping).

## Path

RFC 5626 / RFC 3327 — Path header for SIP outbound. When a UA is behind NAT and registers via an edge proxy, the registrar must not return contacts pointing directly to the NATted peer; instead it stores the edge proxy in the Path header and routes subsequent in-dialog requests through it.

```kamailio
loadmodule "path.so"
modparam("registrar", "use_path", 1)
modparam("registrar", "path_mode", 2)           # 2 = require Path-supported

route[REGISTER] {
    if (is_method("REGISTER")) {
        if (!add_path_received()) {
            sl_send_reply("503", "Internal");
            exit;
        }
        save("location");
        exit;
    }
}

# inbound to registered user
route[INBOUND] {
    if (lookup("location")) {
        # path is already restored, t_relay routes via edge
        t_relay();
    }
}
```

`add_path()`, `add_path_received()` (adds `;received=ip:port` for symmetric NAT), `add_path_user()` add the path. The registrar saves Path; `lookup()` restores it onto the route set so the inbound INVITE traverses the same edge. `path_mode=0` allows clients without Path support; `mode=2` rejects them with 421/420 — best practice for SIP outbound clusters.

## UAC + UAC_redirect

`uac` module turns Kamailio into a User-Agent-Client: send arbitrary requests, replace From/To headers, do auth as a UAC against an upstream. Useful for SIP trunk auth-relay (you authenticate to the carrier instead of clients), notify generation, OPTIONS keepalives.

```kamailio
loadmodule "uac.so"
modparam("uac", "auth_username_avp", "$avp(auser)")
modparam("uac", "auth_password_avp", "$avp(apass)")
modparam("uac", "auth_realm_avp",    "$avp(arealm)")
modparam("uac", "restore_mode", "auto")
modparam("uac", "restore_passwd", "secretkey")

route[TRUNK_AUTH] {
    $avp(auser) = "trunk_user";
    $avp(apass) = "trunk_secret";
    $avp(arealm) = "carrier.example.com";
    uac_auth();
}

failure_route[CARRIER] {
    if (t_check_status("401|407")) {
        $avp(auser) = "trunk_user";
        $avp(apass) = "trunk_secret";
        if (uac_auth()) {
            t_relay();
            exit;
        }
    }
}

route[REPLACE_FROM] {
    uac_replace_from("Carrier", "sip:dnis@trunk.example.com");
    uac_replace_to("", "sip:" + $rU + "@trunk.example.com");
}
```

`uac_redirect` handles 3xx responses: when an upstream returns 302 Moved Temporarily, the proxy can transparently follow the redirect with new contacts rather than passing it back to the originator.

```kamailio
loadmodule "uac_redirect.so"
failure_route[REDIR] {
    if (t_check_status("3[0-9][0-9]")) {
        if (get_redirects("3:5", "1")) {
            t_relay();
            exit;
        }
    }
}
```

## TLS Module

SIPS (RFC 3261 §26) — TLS-encrypted SIP. Required for WebRTC over secure WebSocket (WSS), and for inter-carrier SIP trunks. Configuration in a separate `tls.cfg`.

```kamailio
# kamailio.cfg
listen=tls:eth0:5061
loadmodule "tls.so"
modparam("tls", "config", "/etc/kamailio/tls.cfg")
modparam("tls", "tls_log", 3)
modparam("tls", "init_mode", 1)
```

```ini
# /etc/kamailio/tls.cfg
[server:default]
method = TLSv1.2+
verify_certificate = yes
require_certificate = no
private_key = /etc/letsencrypt/live/sip.example.com/privkey.pem
certificate = /etc/letsencrypt/live/sip.example.com/fullchain.pem
ca_list = /etc/ssl/certs/ca-certificates.crt
cipher_list = ECDHE+AESGCM:ECDHE+CHACHA20:DHE+AESGCM
server_name = sip.example.com
server_id = sip.example.com

[client:default]
method = TLSv1.2+
verify_certificate = yes
require_certificate = yes
ca_list = /etc/ssl/certs/ca-certificates.crt
```

Force TLS on certain branches: `force_tls(); rewritehostport("sip.peer.com:5061;transport=tls");`. Pseudo-vars `$tls_peer_subject`, `$tls_peer_issuer`, `$tls_session_cipher`. Reload runtime: `kamcmd tls.reload`. Renegotiate certs after Let's Encrypt renewal: `systemctl reload kamailio` (HUP triggers TLS reinit but keeps existing connections).

## WebSocket Module

SIP-over-WebSocket per RFC 7118. Required for WebRTC: browser SIP libraries (JsSIP, SIP.js) can only open WebSocket transports. Kamailio listens on TCP 80 (WS) or TCP 443 (WSS) with the `Sec-WebSocket-Protocol: sip` header.

```kamailio
listen=tcp:eth0:8080
listen=tls:eth0:8443

loadmodule "xhttp.so"
loadmodule "websocket.so"
loadmodule "nathelper.so"

modparam("websocket", "keepalive_mechanism", 1)
modparam("websocket", "keepalive_interval", 30)
modparam("websocket", "keepalive_timeout", 5)
modparam("websocket", "ping_application_data", "kamailio-ping")

event_route[xhttp:request] {
    if ($hu =~ "^/ws") {
        ws_handle_handshake();
        exit;
    }
    xhttp_reply("404", "Not Found", "text/plain", "");
}

request_route {
    if (proto == WS || proto == WSS) {
        # WebRTC client
        force_rport();
        if (!isflagset(5)) {
            setflag(5);
            add_contact_alias();
        }
    }
}
```

`ws_handle_handshake()` upgrades the HTTP request to WebSocket. The handshake includes `Sec-WebSocket-Protocol: sip` (or `sip,sip-msrp,...`). After upgrade the connection appears as `proto=WS` (clear) or `proto=WSS` (TLS) to script. Symmetric routing works: `add_contact_alias()` on REGISTER and `handle_ruri_alias()` on inbound to keep traffic on the same TCP socket.

`kamcmd ws.dump` lists active connections; `kamcmd ws.close <id>` force-closes. `xhttp` provides a built-in HTTP/JSON-RPC layer if you want a REST-ish API in addition to WS.

## Presence Module

RFC 3856 SIP presence (PUBLISH/SUBSCRIBE/NOTIFY) — buddy lists, BLF, dialog-state events, MWI. Supports presence types via sub-modules: `presence_xml` (RFC 4480 PIDF), `presence_dialoginfo` (RFC 4235), `presence_mwi` (RFC 3842), `presence_xcap_caps`.

```kamailio
loadmodule "presence.so"
loadmodule "presence_xml.so"
loadmodule "presence_dialoginfo.so"
loadmodule "presence_mwi.so"
modparam("presence", "db_url", "mysql://kamailio:pwd@localhost/kamailio")
modparam("presence", "server_address", "sip:presence@kamailio.example.com")
modparam("presence", "clean_period", 60)
modparam("presence", "expires_offset", 30)
modparam("presence", "fallback2db", 1)
modparam("presence_xml", "force_active", 1)

request_route {
    if (is_method("PUBLISH|SUBSCRIBE")) {
        route(AUTH);
        handle_publish();
        if ($rc == 1) exit;
        handle_subscribe();
        exit;
    }
}
```

`handle_publish()` stores PIDF/dialog-info; `handle_subscribe()` creates a watcher. Notifications fire automatically. Watchers count: `kamcmd presence.cleanup`, `kamcmd presence.refreshWatchers user@d presence`.

For BLF specifically: phones SUBSCRIBE to `dialog;sla` event package; Kamailio publishes dialog-info from `dialog` module via `pua_dialoginfo` aggregation. RLS (Resource List Server, RFC 4662) for buddy-list aggregation lives in `rls` module.

## Sample Routes

Production-style request_route covering sanity, NAT detection, registration, auth, and proxy relay. Modular via named routes.

```kamailio
request_route {
    route(REQINIT);
    route(NATDETECT);
    route(WITHINDLG);
    if (!is_method("REGISTER|MESSAGE")) record_route();
    if (is_method("REGISTER")) {
        route(AUTH);
        route(REGISTRAR);
        exit;
    }
    if ($rU==$null) {
        sl_send_reply("484", "Address Incomplete");
        exit;
    }
    route(AUTH);
    route(LOCATION);
    route(RELAY);
}

route[REQINIT] {
    if (!mf_process_maxfwd_header("10")) {
        sl_send_reply("483", "Too Many Hops");
        exit;
    }
    if (!sanity_check("17895", "7")) {
        xlog("Malformed SIP from $si\n");
        exit;
    }
    if (!pike_check_req()) {
        xlog("L_ALERT", "PIKE blocking $si\n");
        exit;
    }
}

route[WITHINDLG] {
    if (!has_totag()) return;
    if (loose_route()) {
        if (is_method("BYE")) {
            setflag(1);                 # ACC
            rtpengine_delete();
        } else if (is_method("INVITE")) {
            record_route();
        }
        route(RELAY);
    } else {
        if (is_method("ACK")) {
            if (t_check_trans()) {
                t_relay();
                exit;
            }
            exit;
        }
        sl_send_reply("404","Not here");
    }
    exit;
}

route[REGISTRAR] {
    if (!save("location")) sl_reply_error();
    exit;
}

route[LOCATION] {
    if (!lookup("location")) {
        $var(rc) = $rc;
        switch ($var(rc)) {
            case -1: case -3:
                t_newtran();
                t_reply("404", "Not Found");
                exit;
            case -2:
                sl_send_reply("405", "Method Not Allowed");
                exit;
        }
    }
}

route[RELAY] {
    if (has_body("application/sdp")) {
        rtpengine_manage("replace-origin replace-session-connection");
    }
    if (!t_relay()) sl_reply_error();
    exit;
}
```

This is the skeleton of `kamailio.cfg.sample` shipped in `etc/`. Tune for your topology.

## kamcmd

Out-of-band RPC interface over a Unix socket (`/var/run/kamailio/kamailio_ctl`) and/or FIFO. Provided by the `ctl` module. Used for inspection, runtime reload, and operational control without restart.

```bash
kamcmd core.uptime
# now: Thu Apr 25 12:34:56 2026
# up_since: Thu Apr 25 09:00:00 2026
# uptime: 12356 [sec]

kamcmd core.psx                          # process list
kamcmd core.shmmem                       # shm usage
kamcmd core.modules                      # loaded modules
kamcmd core.tcp_info                     # tcp conn state
kamcmd stats.fetch all                   # all counters
kamcmd stats.fetch tm                    # tm-specific

kamcmd ul.dump                           # registrations
kamcmd dispatcher.list                   # dispatcher state
kamcmd dispatcher.reload
kamcmd dlg.list                          # active dialogs
kamcmd htable.dump ipban
kamcmd tls.reload
kamcmd permissions.addressReload
kamcmd debug.reset_msgid                 # reset internal counters
kamcmd cfg.list_groups                   # tunable param groups
kamcmd cfg.set_now_int tm fr_timer 60000 # change timer at runtime
```

`kamctl` is the higher-level wrapper for user/subscriber/db management. `sercmd` is an older alias. JSONRPC over HTTP via `xhttp_rpc` module gives the same RPC over `http://host:port/RPC`.

## SQL Backend Schema

Default Kamailio SQL schema (kamailio-mysql package or `make install-utils`) creates ~30 tables. Key ones:

```sql
-- subscribers (auth)
CREATE TABLE subscriber (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(64), domain VARCHAR(64),
  password VARCHAR(64), ha1 CHAR(64), ha1b CHAR(64),
  email_address VARCHAR(128), rpid VARCHAR(128),
  UNIQUE KEY (username, domain)
);

-- registered contacts (usrloc)
CREATE TABLE location (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  ruid VARCHAR(64), username VARCHAR(64), domain VARCHAR(64),
  contact VARCHAR(255), received VARCHAR(128), path VARCHAR(255),
  expires DATETIME, q FLOAT, callid VARCHAR(255), cseq INT,
  user_agent VARCHAR(255), socket VARCHAR(64), methods INT,
  flags INT, cflags INT, last_modified DATETIME,
  UNIQUE KEY (ruid)
);

-- ACL trusted peers
CREATE TABLE address (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  grp INT UNSIGNED, ip_addr VARCHAR(50), mask INT,
  port INT UNSIGNED, tag VARCHAR(64)
);

-- LB destinations
CREATE TABLE dispatcher (
  id INT AUTO_INCREMENT PRIMARY KEY,
  setid INT, destination VARCHAR(192),
  flags INT, priority INT, attrs VARCHAR(128),
  description VARCHAR(64)
);

-- accounting (acc)
CREATE TABLE acc (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  method VARCHAR(16), from_tag VARCHAR(64), to_tag VARCHAR(64),
  callid VARCHAR(255), sip_code CHAR(3), sip_reason VARCHAR(32),
  time DATETIME
);

-- presence
CREATE TABLE presentity (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(64), domain VARCHAR(64),
  event VARCHAR(64), etag VARCHAR(64),
  expires INT, received_time INT,
  body LONGBLOB, sender VARCHAR(255),
  UNIQUE KEY (username, domain, event, etag)
);

-- speed dial / aliases
CREATE TABLE dbaliases (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  alias_username VARCHAR(64), alias_domain VARCHAR(64),
  username VARCHAR(64), domain VARCHAR(64),
  UNIQUE KEY (alias_username, alias_domain)
);

-- htable persistence
CREATE TABLE htable (
  id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  key_name VARCHAR(64), key_type INT, value_type INT,
  key_value VARCHAR(128), expires INT
);
```

`kamdbctl create kamailio` provisions all tables. Migrate between releases with `kamdbctl migrate`. Tables are MyISAM by default; switch to InnoDB for ACID + replication: `ALTER TABLE location ENGINE=InnoDB;`.

## Routing Cookbook

Anti-flood blocklist with htable + pike:

```kamailio
modparam("htable", "htable", "ipban=>size=8;autoexpire=600;")
modparam("pike", "reqs_density_per_unit", 16)

request_route {
    if ($sht(ipban=>$si) != $null) {
        sl_send_reply("403", "Banned");
        exit;
    }
    if (!pike_check_req()) {
        $sht(ipban=>$si) = 1;
        exit;
    }
}
```

NAT detection + rewrite (compact):

```kamailio
force_rport();
if (nat_uac_test("19")) {
    if (is_method("REGISTER")) fix_nated_register();
    else add_contact_alias();
    setbflag(6);
}
```

Transport translation (UDP-to-TCP for large messages):

```kamailio
if ($proto == "udp" && msg:len >= 1300) {
    $du = "sip:" + $rd + ":" + $rp + ";transport=tcp";
}
```

Mid-Registrar (B2B-style registration relay) — clients register here, this Kamailio forwards filtered REGISTER upstream:

```kamailio
loadmodule "registrar.so"
modparam("registrar", "outbound_mode", 0)
modparam("uac", "auth_username_avp", "$avp(auser)")

route[REGISTER] {
    save("location");
    if ($rc != 1) exit;
    # propagate to upstream
    $du = "sip:upstream-reg.example.com";
    uac_replace_from("", "sip:" + $au + "@upstream.example.com");
    t_relay();
}
```

Number portability (LNP) lookup via SQL:

```kamailio
sql_query("ca", "SELECT carrier FROM lnp WHERE number='$rU'", "ra");
if ($dbr(ra=>rows) > 0) {
    $du = "sip:" + $rU + "@" + $dbr(ra=>[0,0]) + ".carrier.com";
}
sql_result_free("ra");
```

Registrar-only (no media path):

```kamailio
if (is_method("REGISTER")) {
    save("location");
    exit;
}
if (is_method("INVITE|MESSAGE|...")) {
    lookup("location");
    t_relay();
}
```

## siptrace

Captures SIP messages and ships them to a remote collector — typically Homer/HEPv3 (sipcapture.org) for offline replay and call-flow visualization.

```kamailio
loadmodule "siptrace.so"
modparam("siptrace", "duplicate_uri", "sip:homer.example.com:9060")
modparam("siptrace", "hep_mode_on", 1)
modparam("siptrace", "hep_version", 3)
modparam("siptrace", "hep_capture_id", 2001)
modparam("siptrace", "trace_to_database", 0)
modparam("siptrace", "trace_on", 1)

request_route {
    sip_trace();
}

onreply_route {
    sip_trace();
}
```

HEPv3 binary protocol over UDP encapsulates the SIP message + metadata (timestamp, src/dst, transport, capture-id) and is collected by Homer's `heplify-server`, stored in PostgreSQL, indexed by Call-ID, and rendered as ladder diagrams in the Homer UI. Capture-ID is your node identifier so you can filter "show all calls through edge-2".

To trace selectively, set a flag on dialogs of interest:

```kamailio
if ($fU == "trace-me") {
    setflag(22);
    sip_trace_mode("d");        # trace this dialog
}
```

## WebRTC Bridge Setup

Browser ↔ Kamailio (WSS, SIP-over-WebSocket, DTLS-SRTP via rtpengine) ↔ FreeSWITCH or carrier (UDP, RTP).

```kamailio
listen=tls:eth0:443
listen=udp:eth0:5060

loadmodule "tls.so"
loadmodule "websocket.so"
loadmodule "xhttp.so"
loadmodule "rtpengine.so"
loadmodule "nathelper.so"

modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")
modparam("websocket", "keepalive_mechanism", 1)

event_route[xhttp:request] {
    if ($hu =~ "^/ws") {
        ws_handle_handshake();
        exit;
    }
}

request_route {
    if (proto == WS || proto == WSS) {
        force_rport();
        add_contact_alias();
        setflag(WEBRTC);
    }
    if (has_body("application/sdp")) {
        if (isflagset(WEBRTC)) {
            rtpengine_manage("trust-address replace-origin replace-session-connection ICE=force RTP/SAVPF DTLS=passive");
        } else {
            rtpengine_manage("trust-address replace-origin replace-session-connection RTP/AVP ICE=remove");
        }
    }
    if (is_method("INVITE") && !proxy_authorize(...)) ...
    handle_ruri_alias();
    t_relay();
}
```

Bridging direction is automatic — browser-side SDP arrives `RTP/SAVPF` and rtpengine offers `RTP/AVP` to FreeSWITCH (or vice versa), demuxing SRTP↔RTP and ICE↔non-ICE. Kamailio never sees media. Configure FreeSWITCH `sofia.conf.xml` with `ws-binding="80"` or `wss-binding="443"` if doing browser → FS direct (no Kamailio); but the Kamailio-fronted topology scales much better.

The TURN server (coturn) runs alongside for clients behind symmetric NAT — provided in `iceServers` to the JS WebRTC client, not configured in Kamailio.

## Performance Tuning

```kamailio
children=16            # UDP receivers per listen interface
tcp_children=8         # TCP/WS workers
tls_max_connections=4096
tcp_max_connections=4096
tcp_connection_lifetime=3604
shm_mem_size=2048      # MB
pkg_mem_size=64        # MB per process
async_workers=8
auto_aliases=no

modparam("tm", "hash_size", 16384)
modparam("dialog", "hash_size", 16384)
modparam("usrloc", "hash_size", 14)        # 2^14 = 16384
modparam("tm", "fr_timer", 30000)
modparam("registrar", "max_contacts", 5)
```

`children` should match expected concurrent UDP RPS / 200 (a single UDP child can handle ~200 simple-routed CPS). `tcp_children` similarly for TCP/WS/TLS. `shm_mem_size` must hold all transactions, dialogs, registrations, htables — for 100k registered users budget 200 MB usrloc + dialog. `pkg_mem_size` is per-process; 64 MB suffices unless heavy script work (Lua/Python) inflates per-message allocation.

System tuning:

```bash
sysctl -w net.core.rmem_max=33554432 net.core.wmem_max=33554432
sysctl -w net.ipv4.udp_mem="262144 524288 786432"
sysctl -w net.netfilter.nf_conntrack_max=1048576
sysctl -w fs.file-max=2097152
ulimit -n 65535        # in /etc/security/limits.conf for kamailio user
```

Watch `kamcmd core.shmmem` (`fragmentation` rising means shm leak), `kamcmd stats.fetch tm:UAS_transactions`, `kamcmd core.tcp_info`, OS metrics (load, RSS per child, UDP recv-q drops via `netstat -anu`).

## Sample Cookbook

Registrar-proxy (most common minimum-viable Kamailio):

```kamailio
loadmodule "tm.so"; loadmodule "sl.so"; loadmodule "rr.so";
loadmodule "registrar.so"; loadmodule "usrloc.so";
loadmodule "auth.so"; loadmodule "auth_db.so";
modparam("usrloc", "db_url", "mysql://...");
modparam("auth_db", "db_url", "mysql://...");

request_route {
    if (is_method("REGISTER")) {
        if (!auth_check("$fd", "subscriber", "1")) {
            auth_challenge("$fd", "0"); exit;
        }
        consume_credentials();
        save("location"); exit;
    }
    if (loose_route()) { t_relay(); exit; }
    if (!lookup("location")) {
        sl_send_reply("404", "Not Found"); exit;
    }
    t_relay();
}
```

LB across N FreeSWITCH backends:

```kamailio
loadmodule "dispatcher.so";
modparam("dispatcher", "ds_ping_interval", 30);
# dispatcher.list rows for setid=1: sip:fs1:5060, sip:fs2:5060, sip:fs3:5060

request_route {
    if (is_method("INVITE")) {
        ds_select_dst("1", "4");        # round-robin
        t_on_failure("FS_FAIL");
        rtpengine_manage();
        t_relay(); exit;
    }
}
failure_route[FS_FAIL] {
    if (t_check_status("5[0-9][0-9]|408")) {
        if (ds_next_dst()) {
            t_on_failure("FS_FAIL");
            t_relay(); exit;
        }
    }
}
```

WebRTC bridge to PSTN trunk: combine WebSocket listen, rtpengine ICE/SRTP↔RTP transcoding, and dispatcher to the carrier.

ITSP trunk with UAC auth:

```kamailio
modparam("uac", "auth_username_avp", "$avp(auser)");
modparam("uac", "auth_password_avp", "$avp(apass)");
$avp(auser) = "trunkuser";
$avp(apass) = "trunkpass";

request_route {
    if (is_method("INVITE") && uri == myself && !is_uri_host_local()) {
        uac_replace_from("", "sip:trunkuser@itsp.example.com");
        $du = "sip:itsp.example.com:5060";
        t_on_failure("ITSP_AUTH");
        t_relay(); exit;
    }
}
failure_route[ITSP_AUTH] {
    if (t_check_status("401|407")) {
        if (uac_auth()) { t_relay(); exit; }
    }
}
```

Multi-tenant SBC: tenant detection by inbound IP (permissions/address grp = tenant_id), realm-based auth, per-tenant CPS htable, per-tenant dispatcher set.

```kamailio
if (allow_address_group(0, "$si", "$sp")) {
    $var(tid) = $allow_address_grp;
    $avp(tenant) = $var(tid);
    ds_select_dst("$avp(tenant)", "4");
} else {
    sl_send_reply("403", "Unknown peer"); exit;
}
```

## Common Errors verbatim

```
ERROR: <core> [core/cfg.y:3568]: yyerror_at(): parse error in cfg or any
```
Syntax error in kamailio.cfg — the line number that follows is your culprit. Run `kamailio -c -f /etc/kamailio/kamailio.cfg` to validate without starting.

```
ERROR: load_module(): could not open module <...>: cannot open shared object file: No such file or directory
```
`mpath` wrong, or module package not installed (e.g. `kamailio-tls-modules`). Check `ls /usr/lib/x86_64-linux-gnu/kamailio/modules/`.

```
ERROR: <core> [core/tcp_main.c:5057]: tcpconn_send_put(): tcp connection refused — Connection refused
```
Upstream TCP/TLS peer rejected the connection. Verify `dispatcher.list` health, firewall, peer's `listen` address.

```
ERROR: tm [t_lookup.c:614]: t_check_msg(): transaction does not exist
```
ACK or response arrived after txn timed out and was deleted, or t_lookup_request() ran in a non-stateful path. Use `t_check_trans()` first.

```
ERROR: permissions [allow.c:153]: allow_routing(): address not allowed
```
Source IP not in `address` table for the specified group. `kamcmd permissions.addressReload` after editing the table.

```
ERROR: auth [auth_mod.c:289]: pre_auth(): nonce check failed (auth_check)
```
Nonce expired (`nonce_expire` too short for clock skew) or nonce-count reuse mismatch. Bump `nonce_expire`, inspect client clock.

```
WARNING: registrar [lookup.c:289]: lookup_helper(): contact for [user@dom] not found
```
AOR has no live registrations. `kamcmd ul.dump` to confirm.

```
ERROR: tls [tls_init.c:691]: ssl3_get_server_certificate: certificate verify failed
```
Peer cert chain not trusted: missing `ca_list`, hostname mismatch (`server_name` vs. SNI), expired cert, intermediate not bundled in `fullchain.pem`. `openssl s_client -connect peer:5061 -showcerts` to debug.

```
ERROR: <core> [core/mem/q_malloc.c:289]: qm_malloc(): no more shared memory
```
shm exhausted. Bump `shm_mem_size`, find leak via `kamcmd core.shmmem` (`max_used_size`/`fragmentation`).

```
ERROR: dialog [dlg_handlers.c:854]: dlg_onreply(): no dialog for callid
```
Late 200 OK arrived after dialog state expired — either bumped `default_timeout` is needed, or upstream is sending duplicates after BYE.

```
ERROR: rtpengine [rtpengine.c:1432]: send_rtpp_command(): no response from rtpengine
```
rtpengine daemon down or `rtpengine_sock` wrong. `kamcmd rtpengine.show all` and `systemctl status rtpengine`.

## Common Gotchas

```kamailio
# BROKEN: registrar before usrloc — module loaded before dependency
loadmodule "registrar.so"
loadmodule "usrloc.so"
# error at startup: "registrar: module dependency 'usrloc' not satisfied"
```
```kamailio
# FIXED: load deps first
loadmodule "usrloc.so"
loadmodule "registrar.so"
```

```kamailio
# BROKEN: htable contents lost across restart
modparam("htable", "htable", "ipban=>size=8;autoexpire=300;")
# blocklist resets after each restart
```
```kamailio
# FIXED: persist via dbtable
modparam("htable", "htable", "ipban=>size=8;autoexpire=300;dbtable=ipban;")
modparam("htable", "db_url", "mysql://...")
```

```kamailio
# BROKEN: missing check_self / myself test
request_route {
    t_relay();    # forwards to whatever Request-URI says — open relay
}
```
```kamailio
# FIXED: only relay our own AOR or auth-required
if (uri==myself) {
    lookup("location");
} else if (!allow_address_group(...)) {
    sl_send_reply("403", "Forbidden"); exit;
}
t_relay();
```

```kamailio
# BROKEN: fix_nated_contact called twice — Contact header gets two ;received
route[NAT] {
    fix_nated_contact();
    fix_nated_contact();    # second call appends again
}
```
```kamailio
# FIXED: use add_contact_alias which is idempotent and SIP-compliant
if (!isflagset(5)) {
    setflag(5);
    add_contact_alias();
}
```

```kamailio
# BROKEN: dispatcher empty list at startup
modparam("dispatcher", "list_file", "/etc/kamailio/dispatcher.list")
# but dispatcher.list is empty / db row count = 0
# all calls get 503 No destination available
```
```kamailio
# FIXED: bootstrap rows + reload after edits
# INSERT INTO dispatcher (setid, destination) VALUES (1, 'sip:fs1:5060');
# kamcmd dispatcher.reload
```

```kamailio
# BROKEN: SQL conn pool exhaustion under load
modparam("db_mysql", "ping_interval", 5)
# many children x many sql ops = thousands of connections, MySQL caps out
```
```kamailio
# FIXED: pool + connection limit
modparam("db_mysql", "ping_interval", 60)
modparam("usrloc", "connection_pooling", 1)
# and tune MySQL max_connections to children * sql_modules + headroom
```

```kamailio
# BROKEN: TLS module loaded but no TLS listener
loadmodule "tls.so"
# but listen=udp:eth0:5060 only — TLS reload errors at startup
```
```kamailio
# FIXED: declare TLS listen
listen=tls:eth0:5061
loadmodule "tls.so"
modparam("tls", "config", "/etc/kamailio/tls.cfg")
```

```kamailio
# BROKEN: allow_trusted with empty trusted table
if (!allow_trusted("$si", "$proto")) {
    # always returns false — calls all rejected
}
```
```kamailio
# FIXED: populate the trusted table or use allow_address
# INSERT INTO trusted (src_ip, proto, from_pattern) VALUES ('1.2.3.4','any','.*');
# or use IP-only ACL: allow_address_group(...)
```

```kamailio
# BROKEN: forgetting rtpengine_delete on dialog teardown
if (loose_route()) {
    if (is_method("BYE")) {
        # nothing — leaks media port
    }
    t_relay();
}
```
```kamailio
# FIXED: tear media down on BYE
if (loose_route()) {
    if (is_method("BYE") || is_method("CANCEL")) {
        rtpengine_delete();
    }
    t_relay();
}
```

```kamailio
# BROKEN: SIP outbound clients break without Path
# REGISTER comes via edge proxy, but registrar stores only Contact
# inbound INVITE goes direct to NATted client → fails
modparam("registrar", "use_path", 0)
```
```kamailio
# FIXED: enable Path support on registrar and edge
modparam("registrar", "use_path", 1)
modparam("registrar", "path_mode", 2)
add_path_received();
```

```kamailio
# BROKEN: missing semicolon
modparam("tm", "fr_timer", 30000)
modparam("tm", "fr_inv_timer" 120000)    # syntax error: missing comma
```
```kamailio
# FIXED:
modparam("tm", "fr_inv_timer", 120000)
```

```kamailio
# BROKEN: exec module disabled but trying to use exec_msg/exec_dset
loadmodule "exec.so"
# but kamailio binary built without --with-exec → load_module fails
exec_msg("/usr/local/bin/lookup.sh");
```
```kamailio
# FIXED: rebuild kamailio with exec module, or replace exec calls with sqlops/http_async
sql_query("ca", "SELECT route FROM lookup WHERE num='$rU'", "ra");
```

```kamailio
# BROKEN: forgetting record_route() on initial INVITE
# in-dialog ACKs/BYEs bypass kamailio → media tear-down fails
request_route {
    if (is_method("INVITE")) t_relay();   # no record_route
}
```
```kamailio
# FIXED: always record_route on initial INVITE
if (is_method("INVITE") && !has_totag()) record_route();
```

## Diagnostic Tools

`kamcmd` — primary RPC for live state.

```bash
kamcmd core.uptime
kamcmd stats.fetch all | grep -E "tm|usrloc|dialog"
kamcmd ul.dump
kamcmd dlg.list_ctx
```

`ngrep` — packet-level, plain SIP only:

```bash
ngrep -W byline -d eth0 -p 'sip' port 5060
```

`sngrep` — interactive curses SIP call-flow:

```bash
sngrep -d eth0 -r           # rotate buffer
# UI keys: F2 save, Enter expand, F8 SDP, F4 raw
sngrep -I capture.pcap     # offline pcap
```

`SIPp` — load test / functional test, scenario XML:

```bash
sipp -sn uac -d 1000 -s 1000 sip-server:5060 -r 10 -m 1000
sipp -sf custom-uas.xml -p 5070 -trace_msg
```

`Wireshark` — full decode incl. TLS with key log:

```bash
SSLKEYLOGFILE=/tmp/sslkeys.log openssl s_client -connect sip:5061
# Wireshark: Edit → Preferences → Protocols → TLS → (Pre)-Master-Secret log filename
```

`Homer/HEPv3` — long-term call capture, ladder diagrams, search by Call-ID/From/To, indexed in PostgreSQL. Source: github.com/sipcapture/homer.

## CLI Cheatsheet

```bash
# kamcmd RPC
kamcmd core.uptime
kamcmd core.psx
kamcmd core.shmmem
kamcmd core.modules
kamcmd core.tcp_info
kamcmd core.version

# stats
kamcmd stats.fetch all
kamcmd stats.fetch tm
kamcmd stats.reset_all
kamcmd stats.reset_one tm

# usrloc
kamcmd ul.dump
kamcmd ul.lookup location alice
kamcmd ul.add location alice sip:alice@1.2.3.4:5060 3600 0.5 ...
kamcmd ul.rm location alice
kamcmd ul.flush

# dispatcher
kamcmd dispatcher.list
kamcmd dispatcher.reload
kamcmd dispatcher.set_state ip 1 sip:fs1:5060

# dialog
kamcmd dlg.list
kamcmd dlg.list_ctx
kamcmd dlg.terminate_dlg <h_entry> <h_id>

# htable
kamcmd htable.dump ipban
kamcmd htable.delete ipban 1.2.3.4
kamcmd htable.reload ipban

# permissions
kamcmd permissions.addressReload
kamcmd permissions.trustedReload
kamcmd permissions.allowUri uri proto

# tls
kamcmd tls.reload
kamcmd tls.list

# debug
kamcmd debug 4              # set debug level
kamcmd cfg.list_groups
kamcmd cfg.set_now_int tm fr_timer 60000

# kamctl (DB management wrapper)
kamctl add alice secretpass
kamctl rm alice
kamctl passwd alice newpw
kamctl ul show
kamctl trusted add 1.2.3.4 udp ".*" 1
kamctl address add 1 1.2.3.4 32 5060 "carrier"
kamctl acl grant alice local
kamctl dispatcher show
kamctl monitor
```

## Deployment Topologies

Single instance: one Kamailio on a Linux box, MySQL local, rtpengine local. Easy, but no HA — outage on host failure.

Active/Standby HA: two Kamailio instances behind a virtual IP (keepalived/VRRP). Shared SQL backend (MySQL replication or single Postgres). Active handles all traffic; standby kept warm via DMQ replication of dialog/usrloc state. Failover is sub-second on VRRP advert miss.

```
        VIP 1.2.3.4
       /            \
   [kam-A] <-- DMQ --> [kam-B]
       \            /
       [shared MySQL/MariaDB]
```

Active/Active dispatcher-distributed: N Kamailios in front, all listening on different IPs (or anycast), DNS SRV or upstream LB hashing on Call-ID. State synchronized via DMQ for usrloc/htable/dialog. Linear scaling for stateless work; stateful work (dialog tracking) limited by DMQ replication bandwidth.

Edge/Core split: edge proxies handle TLS/WSS termination, NAT detection, registration only; core proxies handle routing/dispatcher/dialog/billing. Edge → Core via internal trust (permissions/address). Each layer scales independently.

```
   [WebRTC clients]               [SIP UA / desk phones]
         |                                |
   [edge-1]  [edge-2]   ...      [edge-N]
        \      |      ___________/
         \     |     /
        [core-1]   [core-2]
              \     /
            [FreeSWITCH cluster]
              \     /
            [PSTN/ITSP trunks]
```

DMQ (`dmq` + `dmq_usrloc`) keeps usrloc/htable/dialog in sync between nodes via a peer-to-peer protocol over SIP. Configure once: `loadmodule "dmq.so"`; `modparam("dmq","server_address","sip:1.2.3.4:5070")`; `modparam("dmq","notification_address","sip:dmq.example.com:5070")`.

## Idioms

`$var(name)` — script-local variable, lifetime = single SIP message:

```kamailio
$var(rt) = 1 + 2;       # local int
$var(s) = "hello";
```

`$avp(name)` — Attribute-Value Pair, persists across script invocations within the transaction. Use to pass data between request_route and failure_route or carry credentials for `uac_auth`.

```kamailio
$avp(auser) = "alice";
$avp(apass) = "secret";
# survives across t_relay → failure_route
```

`$xavp(root=>name)` — eXtended AVP, structured (think nested JSON):

```kamailio
$xavp(call=>peer)   = "fs1";
$xavp(call=>cost)   = "0.05";
xlog("peer=$xavp(call=>peer) cost=$xavp(call=>cost)\n");
```

PV — pseudo-variable, read-only request fields:

| PV | Meaning |
|----|---------|
| `$ru` | Request-URI |
| `$rU` | Request-URI user |
| `$rd` | Request-URI domain |
| `$rp` | Request-URI port |
| `$si` | source IP |
| `$sp` | source port |
| `$proto` | transport (UDP/TCP/TLS/WS/WSS) |
| `$fu` | From URI |
| `$fU` | From user |
| `$fd` | From domain |
| `$tu` | To URI |
| `$ci` | Call-ID |
| `$cs` | CSeq |
| `$rm` | request method |
| `$rs` | reply status |
| `$au` | auth username |
| `$ar` | auth realm |
| `$DLG_lifetime` | dialog duration |

`$ru` rewrite — change Request-URI before relay:

```kamailio
$ru = "sip:" + $rU + "@upstream.example.com:5060;transport=udp";
$du = "sip:upstream.example.com:5060";   # destination override (proxy hop)
t_relay();
```

`drop` early — terminate processing without sending a SIP reply (for blackholed traffic):

```kamailio
if ($si == "1.1.1.1") drop;     # silent drop, no 403
```

vs. `exit` (stops route, allows pending replies) and `return` (returns from the named route to caller).

## Pseudo-Variable (PV) Reference

Kamailio's PVs are runtime accessors for SIP message fields and registry state. The `$ru` form is the most-common.

| PV | Meaning | Read | Write | Example |
|---|---|---|---|---|
| `$ru` | Request URI (full) | yes | yes | `$ru = "sip:bob@example.com"` |
| `$rU` | Request URI username | yes | yes | `$rU = "bob"` |
| `$rd` | Request URI domain | yes | yes | `$rd = "example.com"` |
| `$rp` | Request URI port | yes | yes | `$rp = 5060` |
| `$rP` | Request URI transport | yes | no | UDP/TCP/TLS |
| `$fu` | From URI | yes | yes | `$fu == "sip:alice@example.com"` |
| `$fU` | From username | yes | yes | |
| `$fd` | From domain | yes | yes | |
| `$fn` | From display name | yes | yes | |
| `$ft` | From tag | yes | no | per-dialog identifier |
| `$tu` | To URI | yes | yes | |
| `$tU` | To username | yes | yes | |
| `$td` | To domain | yes | yes | |
| `$tt` | To tag | yes | no | |
| `$ci` | Call-ID | yes | no | unique per call |
| `$cs` | CSeq number | yes | no | per-transaction sequence |
| `$cT` | Content-Type | yes | yes | |
| `$cL` | Content-Length | yes | no | |
| `$rs` | Reply status code | yes | yes | only in onreply_route / failure_route |
| `$rr` | Reply reason | yes | yes | |
| `$rm` | SIP method | yes | no | INVITE/REGISTER/etc. |
| `$si` | Source IP | yes | no | actual transport-source IP |
| `$sp` | Source port | yes | no | |
| `$Ri` | Received IP (this proxy's interface) | yes | no | |
| `$Rp` | Received port | yes | no | |
| `$proto` | Transport protocol | yes | no | "udp"/"tcp"/"tls"/"sctp"/"ws"/"wss" |
| `$au` | Authenticated user | yes | no | populated by www_authenticate / auth_db |
| `$ar` | Authentication realm | yes | no | |
| `$ai` | Authenticated identity | yes | no | |
| `$br` | Branch | yes | no | top-most branch |
| `$ct` | Contact header | yes | yes | |
| `$cl` | Content-Length value | yes | no | |
| `$hdr(Name)` | Specific header value | yes | no | `$hdr(User-Agent)` |
| `$ua` | User-Agent | yes | no | |
| `$T_branch_idx` | Current branch index | yes | no | for parallel forking |
| `$T_reply_code` | Final reply code (in failure_route) | yes | no | |
| `$mb` | SIP message body | yes | yes | rewrite SDP, body, etc. |
| `$ml` | SIP message length | yes | no | |
| `$src_ip` | Source IP (alias for $si) | yes | no | |
| `$Tb` | Branch start timestamp (sec.usec) | yes | no | |
| `$Ts` | Current Unix timestamp | yes | no | |
| `$pp` | Process PID | yes | no | |
| `$pr` | Protocol (1=UDP, 2=TCP, 3=TLS) | yes | no | numeric |
| `$rb` | Request body | yes | yes | |
| `$dd` | Destination domain | yes | no | next-hop domain |
| `$di` | Destination IP | yes | no | next-hop IP |
| `$dp` | Destination port | yes | no | |

### AVP (Attribute-Value Pair) usage

```text
$avp(name)         # generic AVP, scoped to current message
$avp(s:my_var)     # string-typed AVP
$avp(i:user_id)    # integer-typed AVP

# Set
$avp(my_avp) = "stored value";
$avp(user_count) = $avp(user_count) + 1;

# Read
xlog("L_INFO", "AVP value is: $avp(my_avp)\n");

# AVPs persist across the duration of the message processing — across
# multiple route blocks, route() invocations, etc. — but not across
# transactions or dialogs (use htable for that).

# Multi-value AVPs (lists)
$(avp(my_list)[*]) = "value1";
$(avp(my_list)[*]) = "value2";
$(avp(my_list)[*]) = "value3";
# Iterate via $(avp(my_list)[0]), [1], [2], ...
```

### Common $ru rewrite patterns

```text
# Rewrite Request-URI to upstream proxy
$ru = "sip:" + $rU + "@upstream.example.com";

# Add prefix for international dialing
$rU = "+1" + $rU;

# Strip prefix
if ($rU =~ "^9") {
    $rU = $(rU{s.substr,1,0});
}

# Set port
$rp = 5060;

# Force transport
$rP = "TCP";

# Lowercase domain
$rd = $(rd{s.tolower});
```

## More Sample kamailio.cfg Block Patterns

### Standard incoming-INVITE routing flow

```text
route {
    # 1. Sanity check — drop malformed
    if (!sanity_check()) {
        sl_send_reply("400", "Bad Request");
        exit;
    }

    # 2. Source IP filter (anti-spam)
    if (src_ip != myself && !allow_trusted()) {
        if (!auth_check("$fd", "subscriber", "1")) {
            sl_send_reply("403", "Forbidden");
            exit;
        }
    }

    # 3. NAT detection / rewrite
    if (nat_uac_test("19")) {
        force_rport();
        if (is_method("REGISTER")) {
            fix_nated_register();
        } else {
            fix_nated_contact();
        }
        setflag(FLAG_NATED);
    }

    # 4. Method-specific routing
    if (is_method("REGISTER")) {
        route(REGISTRAR);
    } else if (is_method("INVITE")) {
        route(INVITE);
    } else if (is_method("BYE|CANCEL|ACK")) {
        route(WITHIN_DLG);
    } else if (is_method("OPTIONS")) {
        sl_send_reply("200", "OK");
        exit;
    } else {
        sl_send_reply("405", "Method Not Allowed");
        exit;
    }
}

route[REGISTRAR] {
    if (!save("location")) {
        sl_reply_error();
    }
    exit;
}

route[INVITE] {
    # Lookup destination from usrloc
    if (!lookup("location")) {
        sl_send_reply("404", "Not Found");
        exit;
    }

    # NAT-aware media handling
    if (isflagset(FLAG_NATED)) {
        rtpengine_offer("trust-address replace-origin replace-session-connection RTP/AVP");
    }

    # Forward via stateful proxy
    t_on_reply("REPLY_HANDLER");
    t_on_failure("FAILURE_HANDLER");
    if (!t_relay()) {
        sl_reply_error();
    }
}
```

### dispatcher-based load balancer

```text
loadmodule "dispatcher.so"
modparam("dispatcher", "ds_ping_interval", 30)
modparam("dispatcher", "ds_ping_method", "OPTIONS")
modparam("dispatcher", "list_file", "/etc/kamailio/dispatcher.list")
modparam("dispatcher", "flags", 2)  # tracking + failover

# dispatcher.list:
# 1 sip:fs1.example.com:5060
# 1 sip:fs2.example.com:5060
# 1 sip:fs3.example.com:5060

route[INVITE] {
    if (!ds_select_dst("1", "4")) {  # set 1, alg 4 = round-robin
        sl_send_reply("503", "All FreeSWITCH instances down");
        exit;
    }

    t_on_failure("DISPATCHER_FAILURE");
    t_relay();
}

failure_route[DISPATCHER_FAILURE] {
    if (t_check_status("503|408")) {
        if (ds_next_dst()) {
            t_on_failure("DISPATCHER_FAILURE");
            t_relay();
            exit;
        }
    }
}
```

### mid-registrar pattern

```text
loadmodule "mid_registrar.so"
modparam("mid_registrar", "default_expires", 3600)
modparam("mid_registrar", "outgoing_expires", 60)  # upstream sees fewer REGISTERs
modparam("mid_registrar", "mode", 1)  # mid-registrar mode (consume + aggregate)

route[REGISTRAR] {
    if (is_method("REGISTER")) {
        # Mid-registrar consumes the REGISTER, validates auth, stores in
        # local usrloc, and forwards a periodic REGISTER upstream
        if (!mid_registrar_save("location_table", "outgoing_user", "outgoing_proxy")) {
            sl_reply_error();
        }
        exit;
    }
}
```

## See Also

- opensips — OpenSIPS, sister fork of SER, similar capabilities, different module ecosystem
- drachtio — Node.js-based SIP server framework, modern alternative for JS shops
- rtpengine — companion media proxy, almost always paired with Kamailio for NAT/SRTP/recording
- sip-protocol — RFC 3261, the underlying SIP wire protocol
- asterisk — full-PBX, can sit behind Kamailio as a B2BUA
- freeswitch — full-featured B2BUA media server, common Kamailio backend

## References

- kamailio.org — official site, downloads, news
- kamailio.org/docs/modules/stable/ — per-module documentation, every parameter and function
- "Kamailio SIP Server" book — Daniel-Constantin Mierla, asipto.com, the authoritative deep-dive
- github.com/kamailio/kamailio — source, issues, PRs, GitHub Discussions
- lists.kamailio.org/mailman/listinfo/sr-users — user mailing list
- lists.kamailio.org/mailman/listinfo/sr-dev — developer mailing list
- kamailio.org/wiki — community wiki, tutorials, recipes
- kamailio.org/docs/cookbooks/stable/core/ — core cookbook (config syntax, globals)
- kamailio.org/docs/cookbooks/stable/pseudovariables/ — exhaustive PV reference
- kamailio.org/docs/cookbooks/stable/transformations/ — string/number transformations on PVs
- RFC 3261 — Session Initiation Protocol
- RFC 3263 — Locating SIP Servers (DNS NAPTR/SRV)
- RFC 3327 — Path header
- RFC 3856 — SIP for Presence
- RFC 4480 — RPID for PIDF
- RFC 5626 — SIP Outbound
- RFC 7118 — SIP over WebSocket
- RFC 8866 — SDP
- sipcapture.org — Homer/HEPv3 collector for siptrace
- asipto.com — training, consulting, paid support, courses
