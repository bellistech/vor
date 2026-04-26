# OpenSIPS

Open-source, high-performance SIP server, proxy, registrar, presence agent, and B2BUA — the session-aware sibling of Kamailio, both descended from SER. Where Kamailio optimizes for stateless routing throughput, OpenSIPS leans into full session control with first-class B2BUA, dialog tracking, mid-registrar aggregation, and presence/RLS. Used as carrier-grade SBC, multi-tenant proxy, click-to-dial bridge, and WebRTC gateway.

## Setup

OpenSIPS forked from SER (SIP Express Router) in 2008, simultaneously with Kamailio. Before the split, the project was known as OpenSER. The fork was driven by governance disagreements among the core SER developers, not technical hostility — both projects continue to share many modules in spirit, and code occasionally cross-pollinates.

### Lineage

```
        SIP Express Router (SER) — 2002
                |
                | (rename ~2005)
                v
            OpenSER
                |
                +-------- 2008 fork -----+
                |                        |
                v                        v
            OpenSIPS                  Kamailio
        (session-aware)            (stateless speed)
```

OpenSIPS Solutions SRL (Bucharest, Romania) is the commercial entity behind OpenSIPS, founded by Bogdan Andrei Iancu, who remains the project lead and primary maintainer. Bogdan also runs OpenSIPS Bootcamps and the annual OpenSIPS Summit, and authors most of the canonical "Building Block Cookbook" documentation.

### Releases

| Version | Status | Notes |
|---------|--------|-------|
| 1.x | EOL | Legacy; do not deploy |
| 2.4 | EOL | Last 2.x stable |
| 3.0 | EOL | Introduced cluster, MI overhaul |
| 3.1 | EOL | Refined cluster, opensips-cli rewrite |
| 3.2 | EOL | Better TLS, b2b refactor |
| 3.3 | LTS | Long-term stable |
| 3.4 | Stable | Current production track |
| 3.5 | Stable | Latest release; default for new deployments |
| 3.6 | Devel | In progress |

The 3.x line is the only series under active development. All 1.x and 2.x are EOL with no security backports — migrate. The general policy: stick on LTS for production, follow stable for new builds, watch devel for upcoming features.

### Install — package managers

```bash
# Debian / Ubuntu (official OpenSIPS apt repo)
curl https://apt.opensips.org/opensips-org.gpg \
  | sudo apt-key add -
echo "deb https://apt.opensips.org $(lsb_release -sc) 3.5-releases" \
  | sudo tee /etc/apt/sources.list.d/opensips.list
sudo apt update
sudo apt install opensips opensips-cli

# Module packages (install only what you use)
sudo apt install opensips-mysql-module \
                 opensips-postgres-module \
                 opensips-redis-module \
                 opensips-presence-modules \
                 opensips-b2bua-modules \
                 opensips-tls-modules \
                 opensips-tls-wolfssl-module \
                 opensips-rtpengine-module \
                 opensips-mid-registrar-module \
                 opensips-python-module

# RHEL / CentOS / Rocky
sudo yum install -y https://yum.opensips.org/3.5-releases/el9/opensips-yum-release.rpm
sudo yum install opensips opensips-cli

# Source build
git clone https://github.com/OpenSIPS/opensips.git
cd opensips
git checkout 3.5
make all
sudo make install
# Build only specific modules:
make modules=modules/{tm,registrar,usrloc,b2b_logic,rtpengine}
```

### First run

```bash
sudo systemctl enable opensips
sudo systemctl start opensips
sudo systemctl status opensips

# Logs (rsyslog by default)
sudo tail -f /var/log/syslog | grep opensips
sudo journalctl -u opensips -f

# Config and binary
/etc/opensips/opensips.cfg
/usr/sbin/opensips
/usr/bin/opensips-cli
```

### File layout

```
/etc/opensips/
  opensips.cfg               # main config
  opensipsctlrc              # legacy ctl config (DB user, FIFO path)
  cli/                       # opensips-cli config
  tls/                       # tls cert mappings
/usr/lib/x86_64-linux-gnu/opensips/modules/
  *.so                       # loadable modules (one .so per module)
/var/run/opensips/
  opensips.pid
  opensips_fifo              # FIFO MI transport
/var/log/
  opensips.log               # if log_file_name set; default goes to syslog
```

## OpenSIPS vs Kamailio

The most common technical conversation around OpenSIPS is "why not Kamailio?" or vice versa. Both projects share an ancestor and similar config language; the divergence is real but easy to misread.

### Common ancestry

- Both forked from SER (Sip Express Router) in 2008.
- Both still share core idioms: route blocks, modules as `.so` files, `loadmodule` + `modparam`, transaction module (`tm`), registrar, usrloc, dialog, dispatcher, rtpengine integration, dialplan-style number normalization.
- Many modules look textually similar — Kamailio code and OpenSIPS code can sometimes be skim-compatible, but they are not drop-in compatible.

### Divergent design choices

| Axis | Kamailio | OpenSIPS |
|------|----------|----------|
| Primary focus | Stateless routing speed | Session-aware features |
| B2BUA | Optional via `b2b_*` modules; less polished | First-class `b2b_logic` with scenario files, mid-call control |
| Dialog | Available; lighter integration | Deeply integrated with B2BUA, drouting, dispatcher |
| Mid-registrar | Not in core (use `pua`) | First-class `mid_registrar` for SBC consolidation |
| RLS | Available | Available; commonly used with full presence stack |
| Routing language | Same general syntax | Slight differences in module APIs and pseudo-vars |
| Cluster | Built around DMQ + topology hiding + `cluster_*` keepalive | Built around `clusterer` + `bin_listener` + sharded usrloc |
| MI / RPC | RPC: BINRPC, JSON-RPC, HTTP | MI: FIFO, Datagram, HTTP, JSON, opensips-cli |
| CLI | `kamcmd`, `kamctl` | `opensips-cli`, legacy `opensipsctl` |
| Presence stack | `presence`, `presence_xml`, `pua_*` | `presence`, `presence_xml`, `presence_dialoginfo`, `presence_mwi`, `presence_xcapdiff`, `rls` |

### Integration patterns

OpenSIPS and Kamailio are not adversarial in production; they are often deployed together or each is paired with FreeSWITCH/Asterisk for media:

- Kamailio at the edge for stateless DDoS-resistant routing → OpenSIPS for session/B2BUA/recording → FreeSWITCH for media.
- OpenSIPS as multi-tenant SBC with mid_registrar → Asterisk farm for IVR/queues.
- OpenSIPS as WebRTC bridge (proto_wss + rtpengine) → Kamailio core or carrier interconnect.

The shorthand: **Kamailio for stateless routing, OpenSIPS for sessions.** The choice is rarely about raw call rate — both can saturate a 10 GbE NIC — and almost always about which features are in-tree. If you need full B2BUA + mid_registrar + first-class dialog control, OpenSIPS saves you weeks. If you need pure transactional proxy + DMQ-based cluster, Kamailio.

## Architecture

OpenSIPS follows a multi-process model: one main process forks worker children at startup. Each worker handles one SIP message at a time end-to-end, and shared state lives in shared memory (SHM) plus optional external stores (DB, Redis).

```
                     +-------------------------+
                     |  Main process (PID 1)   |
                     +-----------+-------------+
                                 |
                +----------------+----------------+
                |          forks at startup       |
                v                                 v
        +---------------+               +-----------------+
        | UDP children  |  ...  N       |  TCP children   |  ... M
        | (per listener)|               | (one TCP pool)  |
        +---------------+               +-----------------+
                |                                 |
        +---------------+               +-----------------+
        | Timer process |               |  RPC / FIFO MI  |
        +---------------+               +-----------------+
                |                                 |
        +---------------+               +-----------------+
        | Diagnostic    |               | Listener procs  |
        | (sigsegv etc) |               | (proto_ws/wss)  |
        +---------------+               +-----------------+
```

### Process types

- **Attendant (PID 1)** — main process; never handles SIP, only manages children.
- **UDP receivers** — `children=N` per UDP listener, parse-and-route each datagram.
- **TCP workers** — `tcp_children=M` shared across all TCP listeners.
- **Timer** — fires deferred work, dialog timeouts, dispatcher probes.
- **Slow timer** — long-running scheduled work (avoid blocking the fast timer).
- **MI processes** — FIFO/Datagram/HTTP listeners for management commands.
- **HEP receiver** (optional) — `proto_hep` capture endpoints.
- **Module workers** — some modules spawn dedicated processes (e.g., `httpd`, `event_route`).

### Per-listener UDP/TCP sockets

```
listen=udp:eth0:5060           # UDP, public side
listen=udp:eth1:5060           # UDP, private side
listen=tcp:eth0:5060           # TCP, public
listen=tls:eth0:5061           # TLS, public
listen=ws:eth0:80              # WebSocket
listen=wss:eth0:443            # Secure WebSocket
listen=hep_udp:eth0:9060       # HEP capture-receive
```

Each `listen=` line creates its own socket and assigns its own pool of UDP children. Use this to isolate "trusted" traffic from "internet-facing" traffic and to bind specific scripts to specific interfaces via `force_send_socket`.

### Timer / RPC / FIFO interfaces

The Management Interface (MI) is OpenSIPS's name for its control API. Multiple transports expose the same set of commands:

- **FIFO** — `/var/run/opensips/opensips_fifo`; oldest, used by `opensipsctl`.
- **Datagram** — UNIX-domain or UDP datagram socket; lower overhead.
- **HTTP** — JSON over HTTP, used by `opensips-cli` and dashboards.
- **HTTP-JSON-RPC** — equivalent JSON-RPC 2.0 envelope.

Both `opensips-cli` and the legacy `opensipsctl` ultimately funnel commands through MI; modules register handlers like `ul_dump`, `dlg_list`, `ds_list`, `b2b_list`, `t_uac_dlg`, etc.

## opensips.cfg

OpenSIPS configuration is a statement-oriented domain-specific language: each line is a statement, blocks are delimited by `{ }`, and the language is parsed once at startup. The parser is strict: a missing semicolon or unknown identifier aborts startup with `cfg parse error`.

### Top-level structure

```
# 1. Core parameters (no module needed)
log_level=4
log_stderror=no
listen=udp:0.0.0.0:5060
children=8
tcp_children=4
mpath="/usr/lib/x86_64-linux-gnu/opensips/modules/"

# 2. Module loading
loadmodule "tm.so"
loadmodule "rr.so"
...

# 3. Module parameters
modparam("tm", "fr_timeout", 5)
modparam("rr", "enable_full_lr", 1)
...

# 4. Route blocks (the actual logic)
route {
    ...
}

route[from_provider] {
    ...
}

failure_route[main_failure] {
    ...
}

onreply_route[main_reply] {
    ...
}
```

### Route block types

| Block | Trigger |
|-------|---------|
| `route` (no name) | Default entry point for every received request |
| `route[name]` | Called via `route(name)` from elsewhere |
| `branch_route[name]` | Triggered for each branch (per-target leg) created by `tm` |
| `failure_route[name]` | Triggered when a transaction fails (no positive final reply) |
| `onreply_route` | Triggered for every incoming reply (top-level applies to all) |
| `onreply_route[name]` | Triggered only when a transaction was set with `t_on_reply(name)` |
| `local_route` | Triggered for locally-generated requests (e.g., from `t_uac_dlg`) |
| `startup_route` | One-shot, runs once at OpenSIPS startup |
| `timer_route[name, interval]` | Periodic, every `interval` seconds |
| `event_route[name]` | Triggered by an internal event posting (`raise_event`) |
| `error_route` | Triggered when the script raises an unhandled error |

Example using several:

```
startup_route {
    xlog("L_NOTICE", "OpenSIPS started, version $version\n");
    cache_store("local", "active_calls", "0");
}

timer_route[stats_dump, 60] {
    xlog("L_INFO", "[stats] active calls=$stat(dialog_active)\n");
}

route {
    if (!mf_process_maxfwd_header("10")) {
        sl_send_reply("483", "Too Many Hops");
        exit;
    }
    if (is_method("REGISTER")) {
        route(register_handler);
        exit;
    }
    route(relay);
}

route[relay] {
    record_route();
    t_on_failure("main_failure");
    t_on_reply("main_reply");
    if (!t_relay()) {
        sl_reply_error();
    }
    exit;
}

failure_route[main_failure] {
    if (t_check_status("486|408")) {
        xlog("L_INFO", "user busy or timeout, sending VM\n");
        $du = "sip:voicemail@10.0.0.5:5060";
        t_relay();
    }
}

onreply_route[main_reply] {
    if (status =~ "^2[0-9]{2}") {
        xlog("L_INFO", "200 OK back: $rs\n");
    }
}
```

### Pseudo-variables

Pseudo-variables (PVs) are the script-side accessors for SIP message data and runtime state. Common ones:

| PV | Meaning |
|----|---------|
| `$rm` | Request method |
| `$ru` | Request URI |
| `$rU` | Request URI user portion |
| `$rd` | Request URI domain |
| `$fu` / `$fU` / `$fd` | From URI / user / domain |
| `$tu` / `$tU` / `$td` | To URI / user / domain |
| `$ci` | Call-ID |
| `$cs` | CSeq number |
| `$si` / `$sp` | Source IP / source port |
| `$Ri` / `$Rp` | Received IP / port (local socket) |
| `$pr` | Protocol (udp/tcp/tls/ws/wss) |
| `$ua` | User-Agent |
| `$hdr(Name)` | Arbitrary header |
| `$avp(name)` | Attribute-Value Pair (per-message scratch) |
| `$var(name)` | Local variable (script lifetime) |
| `$dlg_val(key)` | Dialog-scoped value (persists across messages of a dialog) |
| `$DLG_status` | Dialog status |
| `$T_branch_idx` | Branch index in `branch_route` |
| `$rs` / `$rr` | Reply status / reason |
| `$socket_in` / `$socket_out` | Listening / sending socket |

### Operators and control flow

```
if ($rm == "INVITE")           # equality
if ($si =~ "^10\.")            # regex match
if ($rU =~ "^\+?44[0-9]{9,11}$")
if (is_method("INVITE|UPDATE"))
if (has_totag())
if ($var(x) > 10)
if (avp_check("auth_check", "eq/i/yes"))

switch ($rU) {
    case /"^800"/:
        route(toll_free);
        break;
    case /"^9[0-9]{10}$"/:
        route(domestic);
        break;
    default:
        route(international);
}

while ($var(i) < 5) {
    $var(i) = $var(i) + 1;
}
```

## Modules

Modules are dynamically loaded `.so` files. The configuration loads them with `loadmodule`, configures them with `modparam`, and calls them with their exported functions inside route blocks.

```
mpath="/usr/lib/x86_64-linux-gnu/opensips/modules/"

loadmodule "sl.so"
loadmodule "tm.so"
loadmodule "rr.so"
loadmodule "maxfwd.so"
loadmodule "sipmsgops.so"
loadmodule "signaling.so"
loadmodule "registrar.so"
loadmodule "usrloc.so"

modparam("usrloc", "db_mode", 0)              # in-memory only
modparam("registrar", "default_expires", 3600)
modparam("tm", "fr_timeout", 5)
modparam("tm", "fr_inv_timeout", 30)
```

### Per-module dependencies

Most modules implicitly require others. The build/runtime will tell you, but the patterns are:

- `tm` is required by anything that does stateful relaying (registrar, b2b_*, dispatcher with stateful mode, drouting).
- `rr` is required by anything that needs Record-Route handling (registrar use, B2BUA mid-call routing).
- `sl` is required for stateless replies.
- `sipmsgops` and `signaling` are general utility modules — load them.
- `auth_db` requires `auth` and a DB driver (`db_mysql`, `db_postgres`, `db_text`).
- `b2b_logic` requires `b2b_entities` and `tm` and (usually) `rr`.
- `dialog` requires `tm` (for dialog tracking via INVITE transactions).

Load order matters less than in some servers because all modules are in-process and resolve symbols on `ready` callbacks, but you must load every required dependency in your config.

## tm Module

`tm` (Transaction Module) implements the SIP transaction state machine: INVITE client transaction, INVITE server transaction, non-INVITE client/server transactions, ACK absorption, retransmission control, branch fan-out, timer A/B/C/D/E/F/G/H/I/J/K, and parallel/serial forking.

Without `tm`, you can only do stateless `forward()` — adequate for some SBC-edge dispatch but unable to track success/failure or fail over.

### Key functions

```
loadmodule "tm.so"
modparam("tm", "fr_timeout", 5)             # Timer F: non-INVITE final response (s)
modparam("tm", "fr_inv_timeout", 30)        # INVITE final response timeout (s)
modparam("tm", "restart_fr_on_each_reply", 0)
modparam("tm", "wt_timer", 5)               # waiting for ACK
modparam("tm", "noisy_ctimer", 1)
modparam("tm", "auto_inv_100", 1)           # auto 100 Trying for INVITE

route[relay] {
    record_route();
    t_on_failure("main_failure");
    t_on_reply("main_reply");
    if (!t_relay()) {
        sl_reply_error();
    }
}

# Branch logic — runs per branch
branch_route[per_branch] {
    xlog("Branch $T_branch_idx going to $du\n");
    if ($du =~ "sip:.*@10\.0\.99\.") {
        # quarantine — drop branch
        drop;
    }
}

# Inspect transaction state
failure_route[main_failure] {
    if (t_check_status("486|408|503")) {
        # try alternate destination
        t_relay("sip:fallback@10.0.0.10");
    }
}
```

### State machine

```
                  INVITE Client Transaction
                  -------------------------
  Calling ---100/1xx---> Proceeding ---2xx---> Terminated
     |                       |
     |                       +---3xx-6xx---> Completed --ACK--> Terminated
     |
     +---timer B---> Terminated (no response)
```

`t_relay()` triggers the appropriate state machine; `t_check_status("486")` matches the final reply class; `t_on_reply` and `t_on_failure` install per-transaction script callbacks.

## registrar Module

Handles the REGISTER request: validates the message, parses Contact headers, persists the binding via `usrloc`, and emits the 200 OK with `Contact` and `Expires`.

```
loadmodule "registrar.so"
modparam("registrar", "default_expires", 3600)
modparam("registrar", "min_expires", 60)
modparam("registrar", "max_expires", 86400)
modparam("registrar", "max_contacts", 10)
modparam("registrar", "received_avp", "$avp(rcvd)")
modparam("registrar", "tcp_persistent_flag", "TCP_PERSIST")
modparam("registrar", "case_sensitive", 0)

route[register_handler] {
    # auth before save
    if (!www_authorize("", "subscriber")) {
        www_challenge("", "auth");
        exit;
    }

    if (!save("location")) {
        sl_reply_error();
    }
    exit;
}

route[lookup_user] {
    if (!lookup("location")) {
        switch ($retcode) {
            case -1:
            case -3:
                t_newtran();
                t_reply("404", "Not Found");
                exit;
            case -2:
                sl_send_reply("405", "Method Not Allowed");
                exit;
        }
    }
}
```

`save("location")` writes to the usrloc table named `location`. `lookup("location")` reads from that table and rewrites the request URI to the registered Contact (or sets multiple branches via `tm` for parallel forking).

## usrloc Module

User Location: the in-memory hash table (and optional DB-backed mirror) of AOR (Address-of-Record) → list of Contacts, with TTL, path, received-from socket, and instance ID.

```
loadmodule "usrloc.so"
modparam("usrloc", "nat_bflag", "NAT")
modparam("usrloc", "db_mode", 2)              # 0=mem, 1=write-through, 2=write-back, 3=DB-only
modparam("usrloc", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("usrloc", "table_name", "location")
modparam("usrloc", "use_domain", 1)
modparam("usrloc", "timer_interval", 60)
modparam("usrloc", "rr_persist", "load-extra")
modparam("usrloc", "matching_mode", 0)
modparam("usrloc", "hash_size", 12)           # 2^12 buckets
modparam("usrloc", "preload", "location")     # warm cache from DB at startup
```

### db_mode semantics

| Mode | Meaning | Use case |
|------|---------|----------|
| 0 | In-memory only | Single-instance, accept loss on restart |
| 1 | Write-through (mem + sync DB) | Highest durability |
| 2 | Write-back (mem + async DB) | Balanced durability + perf |
| 3 | DB-only (no mem cache) | Multi-instance with shared DB |

Pair `db_mode=2` with `preload` to warm in-memory state at startup and avoid lookup misses for already-registered users.

### MI access

```bash
# Dump full usrloc table
opensips-cli -x mi ul_dump

# Show specific AOR
opensips-cli -x mi ul_show_contact location alice

# Add binding manually (testing)
opensips-cli -x mi ul_add location alice sip:alice@1.2.3.4:5060 3600

# Remove
opensips-cli -x mi ul_rm location alice
```

## auth + auth_db Modules

Digest authentication per RFC 2617/7616. `auth` provides the SIP-level WWW-Authenticate / Proxy-Authenticate generation and Digest computation; `auth_db` stores user credentials in a SQL table.

```
loadmodule "auth.so"
loadmodule "auth_db.so"
modparam("auth_db", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("auth_db", "calculate_ha1", 1)
modparam("auth_db", "password_column", "password")
modparam("auth_db", "user_column", "username")
modparam("auth_db", "domain_column", "domain")
modparam("auth_db", "load_credentials", "rpid=rpid")

route[auth_user] {
    if (!proxy_authorize("", "subscriber")) {
        proxy_challenge("", "auth");
        exit;
    }
    if (!db_check_from()) {
        sl_send_reply("403", "Identity check failed");
        exit;
    }
    consume_credentials();
}
```

The `subscriber` table:

```sql
CREATE TABLE subscriber (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    domain VARCHAR(64) NOT NULL,
    password VARCHAR(40) NOT NULL,
    ha1 VARCHAR(64) NOT NULL,
    ha1b VARCHAR(64) NOT NULL,
    rpid VARCHAR(64),
    UNIQUE KEY (username, domain)
);
```

`calculate_ha1=1` means OpenSIPS computes HA1 = MD5(username:domain:password) on the fly from the cleartext `password`. For higher security set `calculate_ha1=0` and pre-compute the `ha1` column, then drop or wipe `password`.

### Common gotcha: nonce expired

Every challenge issues a one-time `nonce` valid for `nonce_expire` seconds (default 30). Slow user agents — particularly mobile clients with bad clocks or proxies that delay re-challenges — return after expiry and you log `auth: nonce expired`. Increase `nonce_expire` to 90+ for mobile-heavy traffic, or enable `nc_enabled` (nonce count) so the same nonce can be re-used.

## permissions Module

IP-based ACL: gates whether a SIP message is allowed in based on source IP, port, and (optionally) URI pattern. Backed by either flat files or DB.

```
loadmodule "permissions.so"
modparam("permissions", "default_allow_file", "/etc/opensips/permissions.allow")
modparam("permissions", "default_deny_file",  "/etc/opensips/permissions.deny")
modparam("permissions", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("permissions", "trusted_table", "address")
modparam("permissions", "address_table", "address")

route[trust_check] {
    if (allow_trusted("$si", "$pr")) {
        setflag(TRUSTED);
        return;
    }
    sl_send_reply("403", "Forbidden");
    exit;
}
```

The `address` SQL table:

```sql
CREATE TABLE address (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    grp INT UNSIGNED DEFAULT 1,
    ip VARCHAR(50) NOT NULL,
    mask INT DEFAULT 32,
    port INT DEFAULT 0,
    proto VARCHAR(4) DEFAULT 'any',
    pattern VARCHAR(64),
    context_info VARCHAR(32)
);
```

Use `grp` to bucket peers (e.g., 1 = upstream carriers, 2 = customer trunks, 3 = internal media), then check group membership in script:

```
if (check_address("1", "$si", "$sp", "$pr")) {
    xlog("trusted carrier source\n");
    setflag(FROM_CARRIER);
}
```

The "allow_trusted" model is the canonical SIP-trunk gate: every trunk peer's IP is in the table, every untrusted IP gets 403 before further script runs.

## dispatcher Module

Distribution to a pool of downstream destinations using a configurable algorithm. Tracks each gateway's health via OPTIONS pings and removes dead targets from rotation.

```
loadmodule "dispatcher.so"
modparam("dispatcher", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("dispatcher", "table_name", "dispatcher")
modparam("dispatcher", "ds_ping_method", "OPTIONS")
modparam("dispatcher", "ds_ping_interval", 30)
modparam("dispatcher", "ds_probing_threshold", 3)
modparam("dispatcher", "ds_probing_mode", 1)        # all targets
modparam("dispatcher", "options_reply_codes", "200,404,405")

route[dispatch] {
    if (!ds_select_dst("1", "4")) {
        sl_send_reply("503", "No destination available");
        exit;
    }
    t_on_failure("dispatch_failure");
    t_relay();
}

failure_route[dispatch_failure] {
    if (t_check_status("5[0-9][0-9]") || t_was_cancelled()) {
        if (ds_next_dst()) {
            t_relay();
        }
    }
}
```

### Algorithms (second arg to ds_select_dst)

| Algo | Behaviour |
|------|-----------|
| 0 | hash on Call-ID |
| 1 | hash on From URI |
| 2 | hash on To URI |
| 3 | hash on Request-URI |
| 4 | round-robin |
| 5 | hash on authorization username |
| 6 | random |
| 7 | hash on PV (third arg) |
| 8 | priority order |
| 9 | weighted (uses dispatcher.weight column) |
| 10 | dynamic-weight |

Round-robin (4) and weighted (9) are the production-typical choices; hash-based modes are essential when you need session affinity (e.g., upstream is stateful and stickying calls to one host).

### Schema

```sql
CREATE TABLE dispatcher (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    setid INT NOT NULL DEFAULT 0,
    destination VARCHAR(192) NOT NULL,
    socket VARCHAR(128),
    state INT DEFAULT 0,
    weight INT DEFAULT 1,
    priority INT DEFAULT 0,
    attrs VARCHAR(128),
    description VARCHAR(64) DEFAULT '',
    KEY (setid)
);
```

### MI commands

```bash
opensips-cli -x mi ds_list
opensips-cli -x mi ds_set_state ai 1 sip:gw1.example.com:5060
opensips-cli -x mi ds_reload
```

## dialog Module

Dialog tracking: from INVITE → 200 → ACK → BYE, OpenSIPS holds an in-memory record of every active dialog, with timeouts, hop info, dialog-scoped variables, and replication slots.

```
loadmodule "dialog.so"
modparam("dialog", "dlg_match_mode", 0)
modparam("dialog", "default_timeout", 14400)        # 4h max call
modparam("dialog", "db_mode", 2)
modparam("dialog", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("dialog", "default_flags", "P")            # always create dialog
modparam("dialog", "profiles_no_value", "in_call;recording")
modparam("dialog", "profiles_with_value", "tenant;trunk")

route {
    if (is_method("INVITE")) {
        create_dialog();
        set_dlg_profile("in_call");
        $dlg_val(tenant) = $avp(tenant_id);
    }

    if (is_method("BYE") || is_method("CANCEL")) {
        $var(t) = $dlg_val(tenant);
        xlog("call ending for tenant $var(t)\n");
    }
}

# Local route runs for OpenSIPS-originated requests
local_route {
    if (is_method("BYE")) {
        xlog("Sending BYE to tear down dialog\n");
    }
}
```

### Mid-call control

```bash
# List active dialogs
opensips-cli -x mi dlg_list

# Forcibly tear down
opensips-cli -x mi dlg_end_dlg <dlg_id>

# Profile counts
opensips-cli -x mi profile_get_size in_call
opensips-cli -x mi profile_get_size tenant TENANT_42
```

The dialog table is the foundation for B2BUA, `mid_registrar` keep-alive, billing, and any feature that requires "where is this call right now?" knowledge.

## b2b_logic Module

The headline OpenSIPS feature: full-featured Back-to-Back User Agent.

A B2BUA splits a call into two coupled UAS+UAC pairs — call leg A talks to UAS-A inside OpenSIPS, leg B is initiated by UAC-B from OpenSIPS — letting you inject mid-call control, record-route every message, manipulate SDP independently per leg, and impose flow rules a stateless proxy cannot.

### XML scenarios

`b2b_logic` reads scenarios from XML files. Each scenario describes the dialog state machine for a particular feature.

```xml
<!-- /etc/opensips/b2b_scenarios/topology_hiding.xml -->
<scenario id="top_hide">
  <init>
    <state>start</state>
  </init>
  <rules>
    <rule>
      <state>start</state>
      <action>
        <client_create destination="$ru" from="$fu" to="$tu"/>
      </action>
    </rule>
  </rules>
</scenario>
```

```
loadmodule "b2b_entities.so"
loadmodule "b2b_logic.so"

modparam("b2b_logic", "script_scenario", "topology_hiding")
modparam("b2b_logic", "script_scenario", "click_to_dial")
modparam("b2b_logic", "script_scenario", "call_recording")
modparam("b2b_logic", "server_address", "sip:opensips@$Ri:$Rp")

route[b2bua] {
    b2b_init_request("topology_hiding");
    exit;
}
```

### Use cases

- **Topology hiding** — opaque the internal SIP topology to upstream/downstream.
- **Transfers** — handle REFER as a full B2BUA-mediated re-INVITE pair, masking the transferer.
- **Hold/resume** — accept `a=sendonly` from leg A, propagate the right SDP to leg B without exposing.
- **Click-to-dial** — generate two outbound INVITEs, glue their media when both legs answer.
- **Call recording bridge** — fork audio to a recorder via the rtpengine `start-recording` flag, while the B2BUA mediates each leg's signaling.
- **Pre-call IVR** — inject an IVR leg before bridging A↔B.

### MI

```bash
opensips-cli -x mi b2b_list
opensips-cli -x mi b2b_trigger_scenario click_to_dial '{"caller":"alice@example.com","callee":"bob@example.com"}'
```

## b2b_entities Module

Lower-level B2BUA primitives: the dialog tables and operations on raw entities (server_dlg, client_dlg). Most users do not call `b2b_entities` directly; they use `b2b_logic` scenarios. But for custom scenarios — e.g., implementing your own conferencing logic in script — `b2b_entities` exposes:

- `b2b_server_new()` — create a server-side leg from an incoming request.
- `b2b_client_new()` — initiate an outbound leg.
- `b2b_pass_request()` — replay a request between paired legs.

```
loadmodule "b2b_entities.so"
modparam("b2b_entities", "server_hsize", 9)
modparam("b2b_entities", "client_hsize", 9)
modparam("b2b_entities", "db_mode", 2)
modparam("b2b_entities", "db_url", "mysql://opensips:pass@localhost/opensips")
```

`db_mode=2` (write-back) is the production setting: dialogs persist across restarts so an in-flight call survives an OpenSIPS reload.

## nathelper Module

NAT traversal: detect that a UAC is behind NAT, fix Contact headers and Via rport, generate keep-alive OPTIONS pings.

```
loadmodule "nathelper.so"
modparam("nathelper", "natping_interval", 30)
modparam("nathelper", "ping_nated_only", 1)
modparam("nathelper", "sipping_method", "OPTIONS")
modparam("nathelper", "sipping_from", "sip:keepalive@example.com")
modparam("nathelper", "received_avp", "$avp(rcvd)")

route[handle_nat] {
    force_rport();

    if (nat_uac_test("23")) {
        if (is_method("REGISTER")) {
            fix_nated_register();
        } else {
            fix_nated_contact();
        }
        setflag(NAT);
    }
}
```

### nat_uac_test bits

| Bit | Test |
|-----|------|
| 1 | Contact host != source IP |
| 2 | Via host != source IP |
| 4 | source IP is private (RFC 1918) |
| 8 | Via rport set |
| 16 | Contact host is private |
| 32 | source port != Via port |

`nat_uac_test("23")` = bits 1+2+4+16 = "any of the typical NAT signs."

`force_rport()` adds `;rport` to Via so replies return to the actual source port; `fix_nated_contact()` rewrites the Contact URI to match the observed source; `fix_nated_register()` does the same for REGISTER and stamps the binding into usrloc.

`natping_interval` ensures NAT bindings on routers don't expire mid-call: OpenSIPS pings every NAT'd registered UA every N seconds.

## rtpproxy / rtpengine Modules

External media proxies handle the RTP plane: OpenSIPS rewrites SDP `c=`/`m=` lines so both legs speak to the proxy, and the proxy relays media. This is required for NAT traversal of RTP, ICE rewriting, transcoding, recording, and DTLS-SRTP↔SRTP for WebRTC.

`rtpproxy` is the legacy Sippy Software project; `rtpengine` is the Sipwise Project's modern rewrite, with kernel-mode forwarding via `xt_RTPENGINE` for line-rate performance.

**Always use rtpengine in new deployments.**

```
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:localhost:22222")
modparam("rtpengine", "rtpengine_sock", "udp:rtp1.example.com:22222=2")
modparam("rtpengine", "extra_id_pv", "$var(extra_id)")

route[handle_media] {
    if (has_body("application/sdp")) {
        if (is_method("INVITE") && !has_totag()) {
            # Initial offer
            rtpengine_offer();
        } else if (is_method("INVITE") && has_totag()) {
            # Re-INVITE
            rtpengine_offer();
        } else if (is_method("ACK")) {
            # ACK with answer — rare, but legal
            rtpengine_answer();
        }
    }
}

onreply_route {
    if (has_body("application/sdp")) {
        rtpengine_answer();
    }
}

route[end_media] {
    if (is_method("BYE") || is_method("CANCEL")) {
        rtpengine_delete();
    }
}
```

### Offer/Answer/Delete API

The rtpengine NG protocol has three primary verbs:

- **offer** — call coming in, here's the offer SDP, give me a rewritten version.
- **answer** — answer SDP coming back, give me the rewritten answer.
- **delete** — call ending, free the session.

Optional flags (string concatenated into the second `rtpengine_offer` arg):

```
rtpengine_offer("trust-address replace-origin replace-session-connection ICE=remove RTP/SAVPF SDES-off");

rtpengine_offer("ICE=force RTP/SAVPF DTLS=passive");          # WebRTC offer side
rtpengine_offer("RTP/AVP ICE=remove DTLS=off");               # bridge to legacy SIP
```

For recording:

```
rtpengine_start_recording();
...
rtpengine_stop_recording();
```

## presence Module

Full RFC 3856 (Presence) + RFC 3265 (SUBSCRIBE/NOTIFY) implementation. Handles SUBSCRIBE state machine, NOTIFY generation, PUBLISH ingestion, watcher subscription tracking, presentity state storage.

```
loadmodule "presence.so"
modparam("presence", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("presence", "fallback2db", 1)
modparam("presence", "expires_offset", 10)
modparam("presence", "max_expires", 3600)
modparam("presence", "server_address", "sip:opensips@example.com")
modparam("presence", "clean_period", 100)
modparam("presence", "db_update_period", 100)

route[presence_handler] {
    if (is_method("SUBSCRIBE")) {
        handle_subscribe();
        exit;
    }
    if (is_method("PUBLISH")) {
        handle_publish();
        exit;
    }
    if (is_method("NOTIFY")) {
        # rare, but support it
        t_relay();
        exit;
    }
}
```

OpenSIPS handles the back-end — you only invoke `handle_subscribe()` / `handle_publish()` and the module:

1. Validates Event header against registered packages.
2. Stores PUBLISH content in `presentity` table.
3. Fans out NOTIFY to every active watcher.
4. Manages SUBSCRIBE refresh / dialog state.
5. Authorizes via XCAP (if `presence_xcapdiff` loaded).

## presence_xml + presence_dialoginfo + presence_mwi + presence_xcapdiff

Specific event packages, each loaded as a separate module on top of `presence`.

### presence_xml — Event: presence

The "user is online/offline" presence per RFC 3863 (PIDF XML body).

```
loadmodule "presence_xml.so"
modparam("presence_xml", "force_active", 1)
modparam("presence_xml", "pidf_manipulation", 1)
modparam("presence_xml", "force_dummy_presence", 1)
```

Sample PUBLISH body:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<presence xmlns="urn:ietf:params:xml:ns:pidf"
          entity="sip:alice@example.com">
  <tuple id="phone">
    <status><basic>open</basic></status>
    <contact>sip:alice@1.2.3.4:5060</contact>
  </tuple>
</presence>
```

### presence_dialoginfo — Event: dialog (BLF)

Busy Lamp Field per RFC 4235. PBX → BLF receiver shows whether each monitored extension is idle / ringing / busy. The bread-and-butter of PBX desk-phones with side-cars.

```
loadmodule "presence_dialoginfo.so"
modparam("presence_dialoginfo", "force_single_dialog", 1)
```

### presence_mwi — Event: message-summary

Voicemail Message-Waiting Indicator per RFC 3842. Voicemail server PUBLISHes count of new/old messages, subscribers see the lamp.

```
loadmodule "presence_mwi.so"

# In voicemail server, generate PUBLISH like:
# Event: message-summary
# Content-Type: application/simple-message-summary
# Messages-Waiting: yes
# Voice-Message: 3/5 (1/2)
```

### presence_xcapdiff — Event: xcap-diff

XCAP server change notifications per RFC 5875. Useful when your buddy lists or call-routing rules live in an XCAP server and clients want to be told when their copy is stale.

```
loadmodule "presence_xcapdiff.so"
modparam("presence_xcapdiff", "force_active", 1)
```

## rls Module

Resource List Server per RFC 4662: a single SUBSCRIBE → buddy list → fan-out to N back-end SUBSCRIBEs → composite NOTIFY back to the subscriber.

```
loadmodule "rls.so"
modparam("rls", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("rls", "xcap_root", "http://xcap.example.com/xcap-root")
modparam("rls", "rls_table_name", "rls_presentity")
modparam("rls", "rlsubs_table_name", "rls_watchers")
modparam("rls", "to_presence_code", 1)
modparam("rls", "max_expires", 7200)
modparam("rls", "server_address", "sip:rls@example.com")

route[rls_handler] {
    if (is_method("SUBSCRIBE") && $hdr(Event) == "presence") {
        if (rls_handle_subscribe()) {
            exit;
        }
    }
    if (is_method("NOTIFY")) {
        rls_handle_notify();
    }
}
```

The classic BLF-side-car case: a desk phone subscribes to `sip:floor3@example.com`, RLS fetches that resource list from XCAP (which contains 30 extensions), issues 30 internal SUBSCRIBEs, aggregates the NOTIFY responses into one composite multipart NOTIFY back to the desk phone.

## dialplan Module

Number-translation tables: regex → replacement, applied to a PV.

```
loadmodule "dialplan.so"
modparam("dialplan", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("dialplan", "table_name", "dialplan")

route[normalize] {
    # dpid 1 = E.164 normalization
    if (dp_translate("1", "$rU/$var(normalized)")) {
        $rU = $var(normalized);
    }
}
```

### Schema

```sql
CREATE TABLE dialplan (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    dpid INT UNSIGNED NOT NULL,
    pr INT DEFAULT 0,                -- priority
    match_op INT NOT NULL,           -- 0=string, 1=regex
    match_exp VARCHAR(255) NOT NULL,
    match_flags INT DEFAULT 0,
    subst_exp VARCHAR(255),
    repl_exp VARCHAR(255),
    timerec VARCHAR(255),
    disabled INT DEFAULT 0,
    attrs VARCHAR(255)
);

INSERT INTO dialplan (dpid,pr,match_op,match_exp,subst_exp,repl_exp)
VALUES
  (1, 1, 1, '^00(.+)$',     '^00(.+)$',     '+\1'),
  (1, 2, 1, '^0([1-9].+)$', '^0([1-9].+)$', '+44\1'),
  (1, 3, 1, '^011(.+)$',    '^011(.+)$',    '+\1');
```

`opensips-cli -x mi dp_reload` reloads the table after a change.

The "dial-string normalization" use case: take any ITU-T / E.164 / national / LRN format and produce a canonical `+44...` URI before routing.

## drouting Module

Dynamic Routing: database-backed prefix-based routing across multiple gateways grouped into carriers, with priority, weight, time-of-day rules, and CLI-based filtering.

```
loadmodule "drouting.so"
modparam("drouting", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("drouting", "use_partitions", 0)
modparam("drouting", "drr_table", "dr_rules")
modparam("drouting", "drg_table", "dr_groups")
modparam("drouting", "drc_table", "dr_carriers")
modparam("drouting", "drd_table", "dr_gateways")

route[outbound] {
    if (!do_routing("0", "FW")) {
        sl_send_reply("503", "No route");
        exit;
    }
    t_relay();
}

failure_route[fw_fail] {
    if (use_next_gw()) {
        t_relay();
    } else {
        t_reply("503", "All gateways failed");
    }
}
```

### Schema overview

```sql
-- gateways
CREATE TABLE dr_gateways (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    gwid VARCHAR(64) NOT NULL UNIQUE,
    type INT NOT NULL,
    address VARCHAR(255) NOT NULL,
    strip INT DEFAULT 0,
    pri_prefix VARCHAR(16),
    attrs VARCHAR(255),
    probe_mode INT DEFAULT 0,
    state INT DEFAULT 0
);

-- carriers (groups of gateways)
CREATE TABLE dr_carriers (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    carrierid VARCHAR(64) NOT NULL UNIQUE,
    gwlist VARCHAR(255) NOT NULL,    -- e.g. "gw1=10,gw2=20" weighted
    flags INT DEFAULT 0,
    sort_alg VARCHAR(16),
    attrs VARCHAR(255)
);

-- routing groups (e.g., per-tenant or per-trunk)
CREATE TABLE dr_groups (
    id INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(64) NOT NULL,
    domain VARCHAR(64),
    groupid INT NOT NULL
);

-- routing rules (the actual dial plan)
CREATE TABLE dr_rules (
    ruleid INT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    groupid VARCHAR(255) NOT NULL,
    prefix VARCHAR(64) NOT NULL,
    timerec VARCHAR(255),
    priority INT DEFAULT 0,
    routeid VARCHAR(255),
    gwlist VARCHAR(255) NOT NULL,
    attrs VARCHAR(255),
    description VARCHAR(255)
);
```

### Reload after schema change

```bash
opensips-cli -x mi dr_reload
```

Failing to call `dr_reload` after editing rows is the single most common drouting bug: the in-memory tree still has the old rules, and your "new" routing isn't applied.

## lcr Module

Least-Cost Routing: like `drouting` but with explicit per-route cost columns and runtime cost-sorted selection.

```
loadmodule "lcr.so"
modparam("lcr", "db_url", "mysql://opensips:pass@localhost/opensips")

route[lcr_route] {
    if (load_gws("1", "$rU", "$fd")) {
        if (next_gw()) {
            t_on_failure("lcr_fail");
            t_relay();
        } else {
            sl_send_reply("503", "No gateway");
        }
    }
}

failure_route[lcr_fail] {
    if (next_gw()) {
        t_relay();
    }
}
```

`drouting` is more flexible (carriers, groups, time-of-day rules) and is more commonly used; `lcr` is the legacy module — pick one or the other, not both.

## siptrace Module

Capture every SIP message in/out of OpenSIPS and ship to a SIP capture server (Homer/HEP, MySQL, syslog).

```
loadmodule "siptrace.so"
modparam("siptrace", "trace_id", "[homer]uri=hep:hep1.example.com:9060")
modparam("siptrace", "trace_id", "[mysql]uri=mysql://opensips:pass@localhost/opensips")
modparam("siptrace", "trace_id", "[syslog]uri=syslog:")

route {
    sip_trace("homer", "d");          # direction: in/out
    sip_trace("homer", "m");          # message
    ...
}
```

For Homer integration, SIPTrace ships HEPv3 packets that Homer indexes for full per-call PCAP-equivalent inspection in the Homer web UI.

## event_route Module

The `event_route[name]` block is the script-side hook for events raised by other modules or by `raise_event` from the script. Modules emit events at well-known points: dialog state changes, registration changes, dispatcher gateway state changes, b2b dialog events, etc.

```
loadmodule "event_route.so"

event_route[E_DLG_STATE_CHANGED] {
    xlog("Dialog $param(dlg_id) state -> $param(state)\n");
}

event_route[E_DISPATCHER_STATUS] {
    xlog("Dispatcher gw $param(uri) state $param(state)\n");
    if ($param(state) == "Inactive") {
        # raise pager
        rest_post("https://alerts.example.com/oncall",
                  "gw=$param(uri)", "$var(resp)");
    }
}

event_route[E_UL_AOR_INSERT] {
    xlog("New registration: $param(aor)\n");
}
```

Useful for hooking your business logic (alerting, billing, cache invalidation) into module events without polling MI.

## http Module

The `httpd` module exposes an in-process HTTP server (used by MI HTTP transport, status pages, custom JSON endpoints), and `rest_client` makes outbound HTTP calls from script.

```
loadmodule "httpd.so"
modparam("httpd", "ip", "127.0.0.1")
modparam("httpd", "port", 8888)

loadmodule "mi_http.so"
modparam("mi_http", "mi_http_root", "mi")        # http://host:8888/mi

loadmodule "rest_client.so"
modparam("rest_client", "connection_timeout", 3000)
modparam("rest_client", "curl_timeout", 5000)

route[lookup_via_rest] {
    rest_get("https://crm.example.com/lookup?phone=$rU",
             "$var(body)", "$var(ct)", "$var(rcode)");
    if ($var(rcode) == 200) {
        $var(target) = $(var(body){json,target});
        $du = "sip:$var(target)@upstream.example.com";
        t_relay();
    }
}
```

`rest_post`, `rest_put`, `rest_delete`, `rest_append_hf` round out the API. Useful for REST callbacks during call setup — query a CRM, validate a JWT against an auth service, post a CDR to a billing webhook.

## python / perl / ruby Modules

Embed scripting languages inside the OpenSIPS configuration: write a Python (or Perl, or Ruby) function and call it like any other route.

```
loadmodule "python.so"
modparam("python", "script_name", "/etc/opensips/script.py")

route[py_route] {
    python_exec("handle_invite", "$rU", "$fd");
}
```

```python
# /etc/opensips/script.py
import opensips

def handle_invite(msg, rU, fd):
    msg.LM_INFO("Python sees INVITE rU=%s fd=%s\n" % (rU, fd))
    if rU.startswith("999"):
        msg.set_var("$avp(blocked)", "yes")
    return 1
```

`perl.so` and `ruby.so` are equivalent for those languages. Useful for prototyping logic, integrating with Python AI/ML services, or reusing existing Ruby/Perl libraries.

## mid_registrar Module

The OpenSIPS-only feature that gives carrier SBC operators a 10× win on REGISTER scaling: aggregate many endpoint REGISTERs into a few upstream REGISTERs.

The architecture: endpoints REGISTER to OpenSIPS as if it were the SIP server. OpenSIPS, instead of forwarding every REGISTER upstream, keeps the binding in its own usrloc and *aggregates* — sending one shared REGISTER per AOR upstream, refreshing on a dampened cadence. When endpoints disappear, OpenSIPS sends a final unregister upstream.

```
loadmodule "mid_registrar.so"
modparam("mid_registrar", "mode", 1)         # mid-mode (vs. throttle-mode)
modparam("mid_registrar", "default_expires", 3600)
modparam("mid_registrar", "outgoing_expires", 86400)
modparam("mid_registrar", "max_contacts", 0)
modparam("mid_registrar", "tcp_persistent_flag", "TCP_PERSIST")

route[mid_register] {
    # First, authenticate the REGISTER on this side
    if (!www_authorize("", "subscriber")) {
        www_challenge("", "auth");
        exit;
    }

    if (mid_registrar_save("location")) {
        # OpenSIPS now owns the binding; upstream gets aggregated REGISTER
        exit;
    }
    sl_reply_error();
}

route[mid_lookup] {
    if (!mid_registrar_lookup("location")) {
        sl_send_reply("404", "Not Found");
        exit;
    }
}
```

### Why it matters

- Multi-tenant SBC with 50,000 endpoints, each registering every 60s = 833 REGISTER/s upstream. With mid_registrar, that becomes maybe 50 REGISTER/s (one per tenant per refresh).
- Upstream PBX/server has fewer connections, much lower CPU.
- Failover and re-registration after a network blip happen at the SBC, not propagated upstream.

Pair `mid_registrar` with NAT keep-alives to the endpoints (via `nathelper`) so endpoints don't lose their NAT bindings while the upstream sees a "stable" registration.

## b2b_sca Module

Shared Call Appearance via B2BUA. Multiple devices share the same line — pick up on phone A, see the call on phone B's line indicator. Implemented as a B2BUA scenario keyed on the SCA group.

```
loadmodule "b2b_sca.so"
modparam("b2b_sca", "db_url", "mysql://opensips:pass@localhost/opensips")
modparam("b2b_sca", "watchers_avp", "$avp(sca_watchers)")
modparam("b2b_sca", "presentity_table", "b2b_sca")

route[sca] {
    b2b_sca_init_request();
}
```

The classic 4-line desk phone with shared appearance for the receptionist's bonded numbers — every line button reflects state across the whole pool, including bridged-pickup and barge-in.

## media_exchange Module

REFER processing for blind and attended transfers: handle the REFER request, validate the new target, and originate the necessary INVITE chain.

```
loadmodule "media_exchange.so"
modparam("media_exchange", "default_callee_address", "sip:transfer@example.com")

route[handle_refer] {
    if (is_method("REFER")) {
        if (media_exchange_from_call("$hdr(Refer-To)", "1")) {
            exit;
        }
    }
}
```

For attended transfer (REFER with `Replaces`), `media_exchange` parses `Replaces`, locates the target dialog, and orchestrates the swap. For blind transfer, it generates a fresh INVITE to the new target while ending the existing leg.

## tls / wolfssl_tls Modules

TLS support over TCP — provides `sip:`-over-TLS at port 5061 (the default) and is the foundation for TLS-secured SIP trunks.

```
loadmodule "tls_mgm.so"
modparam("tls_mgm", "tls_db_enabled", 0)
modparam("tls_mgm", "certificate", "/etc/opensips/tls/cert.pem")
modparam("tls_mgm", "private_key", "/etc/opensips/tls/key.pem")
modparam("tls_mgm", "ca_list", "/etc/opensips/tls/ca-bundle.pem")
modparam("tls_mgm", "verify_cert", 1)
modparam("tls_mgm", "require_cert", 0)
modparam("tls_mgm", "tls_method", "TLSv1_2+")
modparam("tls_mgm", "cipher_list", "HIGH:!aNULL:!MD5")

loadmodule "proto_tls.so"
modparam("proto_tls", "tls_port", 5061)

listen=tls:0.0.0.0:5061
```

Per-domain certificates:

```
modparam("tls_mgm", "domain", "[client_default]")
modparam("tls_mgm", "domain", "[server_default]")
modparam("tls_mgm", "domain", "[client_carrier]:1.2.3.4:5061")
modparam("tls_mgm", "certificate", "[client_carrier]/etc/opensips/tls/carrier.crt")
modparam("tls_mgm", "private_key",  "[client_carrier]/etc/opensips/tls/carrier.key")
```

`wolfssl_tls.so` is the alternative TLS backend using wolfSSL instead of OpenSSL — same parameters, different binary, FIPS-friendly.

## proto_ws + proto_wss Modules

SIP-over-WebSocket transports per RFC 7118. Required for WebRTC clients (browser-based softphones, JsSIP, sipML5).

```
loadmodule "proto_ws.so"
modparam("proto_ws", "ws_port", 80)
modparam("proto_ws", "ws_max_msg_chunks", 4)

loadmodule "proto_wss.so"
modparam("proto_wss", "wss_port", 443)
modparam("proto_wss", "wss_max_msg_chunks", 4)

listen=ws:0.0.0.0:8080
listen=wss:0.0.0.0:443
```

WebRTC clients open `wss://opensips.example.com:443` from JavaScript, exchange SIP messages over the WSS frame layer, and OpenSIPS bridges the SIP into the rest of your network. Pair with `rtpengine` for the media side (DTLS-SRTP ↔ SRTP/RTP).

## Sample opensips.cfg — minimal proxy + registrar

```
####### Global parameters #######
log_level=4
log_stderror=no
log_facility=LOG_LOCAL0
udp_workers=8
tcp_workers=4

socket=udp:0.0.0.0:5060
socket=tcp:0.0.0.0:5060

mpath="/usr/lib/x86_64-linux-gnu/opensips/modules/"

####### Modules #######
loadmodule "signaling.so"
loadmodule "sl.so"
loadmodule "tm.so"
loadmodule "rr.so"
loadmodule "maxfwd.so"
loadmodule "sipmsgops.so"
loadmodule "uri.so"
loadmodule "registrar.so"
loadmodule "usrloc.so"
loadmodule "auth.so"
loadmodule "auth_db.so"
loadmodule "db_mysql.so"
loadmodule "nathelper.so"
loadmodule "rtpengine.so"
loadmodule "mi_fifo.so"
loadmodule "httpd.so"
loadmodule "mi_http.so"

####### Module params #######
modparam("mi_fifo", "fifo_name", "/var/run/opensips/opensips_fifo")

modparam("usrloc", "db_mode", 2)
modparam("usrloc|auth_db", "db_url",
         "mysql://opensips:opensipsrw@localhost/opensips")

modparam("registrar", "default_expires", 3600)
modparam("registrar", "max_contacts", 5)

modparam("auth_db", "calculate_ha1", 1)
modparam("auth_db", "password_column", "password")

modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")

modparam("nathelper", "natping_interval", 30)

####### Routing logic #######
route {
    if (!mf_process_maxfwd_header("10")) {
        sl_send_reply("483", "Too Many Hops");
        exit;
    }

    if ($rm == "OPTIONS" && $ru =~ "ping") {
        sl_send_reply("200", "OK");
        exit;
    }

    force_rport();
    if (nat_uac_test("23")) {
        if ($rm == "REGISTER") {
            fix_nated_register();
        } else {
            fix_nated_contact();
        }
        setflag(NAT);
    }

    if (loose_route()) {
        if ($rm == "BYE" || $rm == "CANCEL") {
            route(media_relay);
        }
        route(relay);
        exit;
    }

    if ($rm == "REGISTER") {
        route(register);
        exit;
    }

    if (!is_uri_host_local()) {
        sl_send_reply("403", "Relaying not allowed");
        exit;
    }

    if (!lookup("location")) {
        switch ($retcode) {
            case -1:
            case -3:
                sl_send_reply("404", "Not Found");
                exit;
            case -2:
                sl_send_reply("405", "Method Not Allowed");
                exit;
        }
    }

    record_route();
    if ($rm == "INVITE") {
        route(media_relay);
    }
    route(relay);
}

route[register] {
    if (!www_authorize("", "subscriber")) {
        www_challenge("", "auth");
        exit;
    }
    if (!save("location")) {
        sl_reply_error();
    }
}

route[media_relay] {
    if (has_body("application/sdp")) {
        rtpengine_offer();
    }
}

route[relay] {
    t_on_failure("main_failure");
    t_on_reply("main_reply");
    if (!t_relay()) {
        sl_reply_error();
    }
}

onreply_route[main_reply] {
    if (has_body("application/sdp")) {
        rtpengine_answer();
    }
}

failure_route[main_failure] {
    if (t_check_status("486|408") && isflagset(NAT) == 0) {
        # could send to voicemail here
    }
    if (is_method("INVITE")) {
        rtpengine_delete();
    }
}
```

## opensips-cli

Modern Python-based CLI introduced in 3.0 to replace the bash-based `opensipsctl`. Talks to OpenSIPS via the MI HTTP / Datagram / FIFO transport.

```bash
# Status
opensips-cli -x mi version
opensips-cli -x mi uptime
opensips-cli -x mi ps                    # process list
opensips-cli -x mi get_statistics all

# Database admin
opensips-cli -x database create
opensips-cli -x database add-tables presence b2b drouting

# usrloc
opensips-cli -x mi ul_dump
opensips-cli -x user add alice@example.com mypassword
opensips-cli -x user remove alice@example.com

# dispatcher
opensips-cli -x mi ds_list
opensips-cli -x mi ds_reload
opensips-cli -x mi ds_set_state ip 1 sip:gw1.example.com:5060

# dialog
opensips-cli -x mi dlg_list
opensips-cli -x mi dlg_end_dlg <dlg_id>

# drouting
opensips-cli -x mi dr_reload
opensips-cli -x mi dr_gw_status

# presence
opensips-cli -x mi presentity_list

# trace control
opensips-cli -x mi sip_trace on
opensips-cli -x mi sip_trace off

# Interactive shell
opensips-cli
> mi ul_dump
> mi dlg_list
> exit
```

The `~/.opensips-cli.cfg` file configures the default MI transport, password, and pretty-print. Use `output_type = pretty-print` to get colorized JSON.

## opensipsctl

Legacy bash CLI from the OpenSER days. Still bundled, still usable, but `opensips-cli` is the canonical interface in modern deployments.

```bash
# Original commands, FIFO-driven
opensipsctl start
opensipsctl stop
opensipsctl restart
opensipsctl monitor

opensipsctl add alice mypassword
opensipsctl rm alice

opensipsctl ul show
opensipsctl ul show alice

opensipsctl fifo dlg_list
opensipsctl fifo ul_dump
```

The control file `/etc/opensips/opensipsctlrc` defines the FIFO path, DB credentials, and module-specific paths.

## mi (Management Interface)

The MI is OpenSIPS's RPC abstraction. Every module that wants to expose a runtime command registers it; modules transports (`mi_fifo`, `mi_datagram`, `mi_http`, `mi_json_rpc`) all expose the same registry.

### Common commands

| Command | Module | What |
|---------|--------|------|
| `version` | core | OpenSIPS version |
| `uptime` | core | Uptime |
| `ps` | core | Process list |
| `get_statistics` | core | Statistics counters |
| `reload_routes` | core | Reload only the routes |
| `pwd` | core | Working directory |
| `kill` | core | Trigger shutdown |
| `ul_dump` | usrloc | Dump usrloc table |
| `ul_show_contact` | usrloc | One AOR's contacts |
| `ul_add` / `ul_rm` | usrloc | Manual binding mgmt |
| `dlg_list` | dialog | List dialogs |
| `dlg_end_dlg` | dialog | Force-end a dialog |
| `profile_get_size` | dialog | Profile counters |
| `ds_list` | dispatcher | Dispatcher state |
| `ds_set_state` | dispatcher | Force GW state |
| `ds_reload` | dispatcher | Reload from DB |
| `dr_reload` | drouting | Reload routing tree |
| `dr_gw_status` | drouting | Gateway state |
| `b2b_list` | b2b_logic | Active B2BUA dialogs |
| `presentity_list` | presence | Presence subscribers |
| `t_uac_dlg` | tm | Inject a UAC request |

### Transports

```bash
# FIFO
echo ":ul_dump:\n" > /var/run/opensips/opensips_fifo

# Datagram (UDP)
echo '{"jsonrpc":"2.0","method":"ul_dump","id":1}' \
  | nc -u 127.0.0.1 8080

# HTTP (with mi_http)
curl -X POST http://127.0.0.1:8888/mi \
  -d '{"jsonrpc":"2.0","method":"ul_dump","id":1}' \
  -H 'Content-Type: application/json'

# Via opensips-cli
opensips-cli -x mi ul_dump
```

## SQL Backend

OpenSIPS uses standard SQL backing tables — many shared with Kamailio (location, subscriber, address, dialog, dispatcher, dr_*, presentity, watchers) — plus OpenSIPS-specific:

- `b2b_entities` / `b2b_logic` / `b2b_sca` — B2BUA persistence
- `mid_registrar_contacts` — mid-registrar bindings
- `cachedb` — cache backend mapping
- `clusterer` — cluster node state
- `userblacklist` — blacklist per user

### Drivers

- `db_mysql` — MySQL/MariaDB, mature, default for most deployments
- `db_postgres` — PostgreSQL
- `db_oracle` — legacy
- `db_text` — flat-file CSV (testing only)
- `db_unixodbc` — generic ODBC
- `db_perlvdb` / `db_http` — esoteric

### Schema management

```bash
# Create database + base schema
opensips-cli -x database create

# Add module-specific schemas
opensips-cli -x database add-tables presence
opensips-cli -x database add-tables b2b
opensips-cli -x database add-tables drouting
opensips-cli -x database add-tables dispatcher

# Migration after upgrade
opensips-cli -x database migrate <from-version> <to-version>
```

The shipped `*.sql` files under `/usr/share/opensips/<driver>/` define the schemas and are kept in sync with the binary.

## SQL-via-cachedb Pattern

`usrloc`, `dialog`, `permissions`, `dispatcher`, and `drouting` all support a `cachedb` front for hot data, with SQL as the durable store.

```
loadmodule "cachedb_redis.so"
modparam("cachedb_redis", "cachedb_url", "redis://127.0.0.1:6379/0")

loadmodule "usrloc.so"
modparam("usrloc", "db_mode", 3)            # DB-only mode
modparam("usrloc", "cachedb_url", "redis://127.0.0.1:6379/0/usrloc/")
```

The pattern: lookups hit Redis first; on miss, fall back to MySQL; writes go to both. For multi-instance OpenSIPS clusters this gives you a shared registration cache without each node hitting MySQL on every lookup.

`cachedb_mongodb`, `cachedb_cassandra`, `cachedb_memcached`, and `cachedb_local` (in-process) round out the backends. Redis is by far the most common.

## Routing Cookbook

Same building blocks as Kamailio (anti-flood, NAT-handling, transport-translation, registrar) plus OpenSIPS-specific aggregation, B2BUA orchestration, and recording.

### Anti-flood

```
loadmodule "pike.so"
modparam("pike", "sampling_time_unit", 2)
modparam("pike", "reqs_density_per_unit", 30)
modparam("pike", "remove_latency", 4)

route {
    if (pike_check_req()) {
        # source over threshold
        sl_send_reply("503", "Throttled");
        exit;
    }
    ...
}
```

### NAT-handling

(Already covered above with `nathelper`.)

### Transport translation

```
route {
    if ($pr == "tls" && $rd == "internal.example.com") {
        # TLS in, UDP to internal
        $du = "sip:internal.example.com:5060;transport=udp";
        force_send_socket(udp:eth1:5060);
    }
    ...
}
```

### Registrar (basic)

(Covered in Sample opensips.cfg above.)

### Mid-registrar (consume + aggregate)

```
modparam("mid_registrar", "mode", 1)
modparam("mid_registrar", "outgoing_expires", 86400)

route[handle_register] {
    if (!www_authorize("", "subscriber")) {
        www_challenge("", "auth");
        exit;
    }
    mid_registrar_save("location");
    exit;
}
```

The endpoints see a normal SIP registrar at OpenSIPS; the upstream sees a few aggregated REGISTERs.

### B2BUA-based hold/transfer

```
modparam("b2b_logic", "script_scenario", "hold_resume")

route[invite] {
    if (is_method("INVITE")) {
        b2b_init_request("hold_resume");
        exit;
    }
}
```

The `hold_resume.xml` scenario intercepts re-INVITEs with `a=sendonly` SDP and holds leg B without leaking the hold semantics across the call.

### Click-to-dial via b2b_logic

```
event_route[E_CLICK_TO_DIAL] {
    $var(caller) = $param(caller);
    $var(callee) = $param(callee);

    b2b_trigger_scenario("click_to_dial",
                         '{"caller":"$var(caller)","callee":"$var(callee)"}');
}
```

Triggered by an HTTP webhook (via `httpd` + `mi_http`):

```bash
curl -X POST 'http://opensips.example.com:8888/mi' \
  -H 'Content-Type: application/json' \
  -d '{
        "jsonrpc":"2.0",
        "method":"b2b_trigger_scenario",
        "params":["click_to_dial",
                  {"caller":"alice","callee":"+441234567"}],
        "id":1
      }'
```

### Call recording bridge via b2b_logic

```
modparam("b2b_logic", "script_scenario", "recording")

route[invite] {
    if (is_method("INVITE") && avp_check("record","eq/i/yes")) {
        b2b_init_request("recording");
        exit;
    }
}
```

The `recording.xml` scenario forks the audio to the rtpengine recorder via `start-recording` flag, while the B2BUA mediates each leg of the call.

## WebRTC + OpenSIPS

The pipeline:

```
Browser (DTLS-SRTP, ICE, WSS)
   |
   | WSS
   v
OpenSIPS (proto_wss.so + tls_mgm.so)
   |
   | rtpengine offer/answer with DTLS=passive RTP/AVP ICE=remove
   v
rtpengine (DTLS termination)
   |
   | RTP/SRTP
   v
FreeSWITCH or Asterisk
```

Configuration:

```
listen=wss:0.0.0.0:443

loadmodule "tls_mgm.so"
loadmodule "proto_wss.so"

# rtpengine handles WebRTC ⇄ legacy SIP media translation
loadmodule "rtpengine.so"
modparam("rtpengine", "rtpengine_sock", "udp:rtp.example.com:22222")

route {
    if ($pr == "ws" || $pr == "wss") {
        setflag(WEBRTC);
    }

    if (has_body("application/sdp")) {
        if (isflagset(WEBRTC)) {
            rtpengine_offer("trust-address replace-origin "
                            "ICE=force RTP/AVP DTLS=passive");
        } else {
            rtpengine_offer("trust-address replace-origin "
                            "ICE=remove RTP/AVP DTLS=off");
        }
    }
    ...
}
```

The flags are doing real work: when bridging WebRTC to legacy SIP, you need DTLS termination on the WebRTC side and plain RTP on the SIP side, with ICE candidates rewritten so each side believes it's talking to a peer it can reach.

## OpenSIPS-2-Asterisk Integration

The most common architectural pairing for SMB and mid-market PBX deployments:

```
+----------------+
|   Endpoints    |     SIP REGISTER, INVITE, etc
+--------+-------+
         |
         v
+----------------+
| OpenSIPS       |     edge SBC, mid_registrar, NAT, anti-flood
| (front-door)   |     B2BUA for transfer/hold/recording
+--------+-------+
         |
         v
+----------------+
| Asterisk       |     IVR, ACD, queues, voicemail, conferencing
| (media app)    |     FreePBX/PBXact for admin UI
+----------------+
```

Why split? Asterisk's media path is excellent (codec menu, recording, IVR), but its SIP stack and registrar do not scale well to thousands of registrations behind NAT. Putting OpenSIPS in front offloads the SIP-side scaling and security challenges, while Asterisk does what it's best at.

```
# OpenSIPS sample relay to Asterisk farm
modparam("dispatcher", "ds_ping_method", "OPTIONS")

route[to_asterisk] {
    ds_select_dst("1", "4");                # round-robin Asterisk pool
    t_on_failure("ast_fail");
    t_relay();
}

failure_route[ast_fail] {
    if (ds_next_dst()) {
        t_relay();
    }
}
```

## Performance Tuning

The OpenSIPS performance triangle is processes × DB × media. Tune all three.

### Process pool

```
udp_workers=16          # or `children=16` (alias)
tcp_workers=8
```

A rule of thumb: `udp_workers` ≈ CPU count × 2, never less than 8 for production. Each UDP worker handles one message at a time end-to-end, so blocking I/O (DB, REST callbacks) inside a route blocks one worker until it returns.

### Per-listener tuning

```
socket=udp:eth0:5060 use_workers 16     # dedicated pool for this socket
socket=udp:eth1:5060 use_workers 4
```

Useful when one interface (e.g., the carrier-facing one) has 20× the traffic of another.

### Database

- Use `db_mode=2` for usrloc (write-back) with `preload`.
- Keep MySQL/Postgres on the same VLAN, pool connections with `db_max_async_connections`.
- Skip `subscriber` queries by setting `force_active=1` for presence and pre-loading via cron.
- For >1000 reg/s, move usrloc to Redis via cachedb.

### rtimer

```
loadmodule "rtimer.so"
modparam("rtimer", "timer", "name=hourly_cleanup;interval=3600;type=2")
modparam("rtimer", "timer", "name=stats_dump;interval=60;type=2")
modparam("rtimer", "exec", "timer=hourly_cleanup;route=do_cleanup")
modparam("rtimer", "exec", "timer=stats_dump;route=dump_stats")

route[do_cleanup] {
    # avp / dlg cleanup once an hour
}

route[dump_stats] {
    xlog("active=$stat(dialog_active) registered=$stat(location_users)\n");
}
```

`rtimer` is the canonical way to do scheduled work in OpenSIPS — better than `timer_route` for anything > 1Hz because it runs on a dedicated process and won't starve the fast timer.

### Stats

```bash
opensips-cli -x mi get_statistics all
opensips-cli -x mi get_statistics dialog:
opensips-cli -x mi get_statistics tm:
opensips-cli -x mi get_statistics shmem:

# Plug into Prometheus via exporter
# https://github.com/voxility/opensips_exporter
```

## Common Errors verbatim

```
ERROR: <core>: cfg parse error
```
Syntax error in opensips.cfg. The previous lines of the log will give the exact line number. Most often a missing `;`, an unclosed `}`, or an unknown identifier (typo of a function name, or a function from a module you forgot to `loadmodule`).

```
ERROR: <db_mysql>: connection refused
```
MySQL is not running, the credentials in `db_url` are wrong, or `bind-address` in MySQL is `127.0.0.1` and OpenSIPS is using a different IP. Check `mysql -u opensips -p opensips` from the same host.

```
ERROR: <auth>: nonce expired
```
A REGISTER (or INVITE) returned with an expired Digest nonce. Increase `modparam("auth","nonce_expire", 90)` and consider enabling nonce-count `modparam("auth","nc_enabled", 1)`.

```
ERROR: <b2b_logic>: scenario not found
```
The scenario referenced by `b2b_init_request("name")` is not loaded — either the XML file path was wrong in `script_scenario`, or the file's `<scenario id="...">` doesn't match the name passed.

```
WARNING: <usrloc>: contact expired
```
Just a notice, but if it floods, your endpoints aren't refreshing — check `default_expires` and `min_expires` and any keep-alive on the endpoint side.

```
ERROR: <permissions>: source IP not allowed
```
The source IP is not in the `address` table or in `permissions.allow`. Either insert it (and `opensips-cli -x mi address_reload`) or fix the routing/NAT to use a known peer IP.

```
ERROR: <rtpengine>: ng failed
```
The rtpengine NG control message timed out. Either rtpengine is down (`systemctl status rtpengine`), the listening socket in `rtpengine_sock` doesn't match rtpengine's `--listen-ng`, or a firewall is blocking UDP/22222.

```
ERROR: <tls>: handshake failed
```
TLS handshake didn't complete: peer cert untrusted (check `ca_list`), wrong cipher suite, or peer presented a cert OpenSIPS doesn't accept. `tls_method=TLSv1_2+` and `cipher_list="HIGH:!aNULL:!MD5"` cover the modern peer side.

## Common Gotchas

### 1. Module load order

```
# BROKEN — registrar.so before usrloc.so it depends on
loadmodule "registrar.so"
loadmodule "usrloc.so"
```

```
# FIXED
loadmodule "usrloc.so"
loadmodule "registrar.so"
```

OpenSIPS will sometimes still resolve symbols at startup, but `modparam`s and module init order can produce confusing diagnostics. Load dependencies first.

### 2. mid_registrar without keep-alive → upstream registration expires

```
# BROKEN — mid_registrar but no NAT pings to keep upstream alive
modparam("mid_registrar", "mode", 1)
modparam("mid_registrar", "outgoing_expires", 86400)
# (no nathelper natping_interval, upstream sees nothing for 1 day)
```

```
# FIXED
modparam("nathelper", "natping_interval", 30)
modparam("nathelper", "ping_nated_only", 0)        # ping all reg'd
modparam("mid_registrar", "outgoing_expires", 3600)
```

`outgoing_expires=86400` does not absolve OpenSIPS from refresh — the upstream PBX expects keep-alive even within the negotiated TTL. Pair with nathelper natping.

### 3. B2BUA scenario file syntax wrong → b2b_init returns -1

```xml
<!-- BROKEN — typo in <state> tag -->
<scenario id="topology">
  <init><stat>start</stat></init>
  ...
</scenario>
```

```xml
<!-- FIXED -->
<scenario id="topology">
  <init><state>start</state></init>
  ...
</scenario>
```

`b2b_init_request` returns -1 with a warning. Validate scenarios with an XML schema check before deploying.

### 4. drouting reload not triggered after schema update

```sql
-- updated rules, no reload
INSERT INTO dr_rules ... ;
```

```bash
# FIXED
opensips-cli -x mi dr_reload
```

The drouting tree is read once at startup and held in memory. Without `dr_reload`, your new rule is sitting in the DB but never applied.

### 5. rtpengine module compiled against wrong version of rtpengine

```
ERROR: <rtpengine>: NG protocol version mismatch
```
You upgraded rtpengine to a new release but `rtpengine.so` was built against an older NG protocol. Reinstall `opensips-rtpengine-module` from the same release channel as your OpenSIPS, or rebuild the module from source against the running rtpengine version.

### 6. dispatcher ds_set_id mismatched

```
# BROKEN — script uses set 1, DB has set 2
ds_select_dst("1", "4");
```

```sql
SELECT setid FROM dispatcher;        -- shows setid=2
```

```
# FIXED — match it
ds_select_dst("2", "4");
```

Or update the DB, or use a `setid` MI to figure it out at runtime. Consistent integer IDs between SQL and script is mandatory.

### 7. presence module without watcher table populated

```
# BROKEN — presence loaded, but watchers table empty, no NOTIFY going out
modparam("presence", "fallback2db", 1)
```

```
# FIXED — add startup_route to verify schema
opensips-cli -x database add-tables presence
opensips-cli -x mi presentity_list   # confirm
```

The `watchers` table is auto-populated as SUBSCRIBE arrives, but `presentity` requires PUBLISH or `force_dummy_presence`. If both are empty, no notify ever fires.

### 8. cluster_relay() in OpenSIPS distributed deployment without bin_listener

```
# BROKEN — clusterer module loaded but no bin_listener configured
loadmodule "clusterer.so"
modparam("clusterer", "current_id", 1)
modparam("clusterer", "db_url", "mysql://...")
# missing: bin_listener
```

```
# FIXED
listen=bin:eth1:5555
modparam("clusterer", "current_id", 1)
```

Cluster nodes communicate over a binary protocol on its own listener. Without `listen=bin:...`, replication silently doesn't happen.

### 9. cachedb_redis without redis-server up

```
ERROR: <cachedb_redis>: failed to connect to redis
```
Redis is not running or not on the address in `cachedb_url`. Start it (`systemctl start redis`) or fix the URL. Most deployments use a localhost Redis on the same host as OpenSIPS to keep latency low.

### 10. CFG syntax differences from Kamailio

```
# Kamailio uses
sl_send_reply("486", "Busy Here");
```
```
# OpenSIPS uses (different module)
sl_send_reply(486, "Busy Here");      # in some versions
```

Many functions accept a status string in OpenSIPS 3.x and an integer in some module variants — check the `opensipsdoc` man page for your exact version. Always validate after a paste from Kamailio cookbook material.

### 11. exec module disabled by default in security-hardened

```
# BROKEN — old habit
exec_msg("/usr/local/bin/lookup.sh $rU");
```
```
# FIXED — the modern way
rest_get("https://lookup.example.com/api?n=$rU", "$var(b)", "", "$var(rc)");
```

Forking a shell out of OpenSIPS for each call is a security and performance disaster. Use `rest_client` or one of the embedded scripting modules.

### 12. Dialog cleanup failure → dialog table grows unbounded

```
# BROKEN — dialog db_mode=1 (write-through) but never cleans up failed dialogs
modparam("dialog", "db_mode", 1)
# (no default_timeout set, dialogs never expire)
```

```
# FIXED
modparam("dialog", "default_timeout", 14400)         # 4 hours
modparam("dialog", "db_update_period", 60)
modparam("dialog", "db_cleanup_pace", 100)
```

Without `default_timeout`, half-open dialogs (no BYE) stay forever and the DB grows without bound. Set a sane wall-clock cap on every dialog.

## Diagnostic Tools

### opensips-cli

The first stop. Already covered.

### ngrep

```bash
sudo ngrep -d any -W byline port 5060
sudo ngrep -d eth0 -W byline -l '^(INVITE|REGISTER)' port 5060
```

Old-school but unbeatable for ad-hoc terminal SIP debugging.

### sngrep

```bash
sudo sngrep -d any -i 192.0.2.1 port 5060
sudo sngrep -r capture.pcap            # offline read
```

Curses UI showing SIP call flows in ladder diagrams. The fastest way to "see" what's happening across multiple parallel calls.

### Homer / HEP integration

OpenSIPS ships first-class HEP (Homer Encapsulation Protocol) capture via `siptrace` + `proto_hep`. Every SIP message can be exfiltrated to a Homer cluster for searchable, indexed PCAP-equivalent inspection in a web UI:

```
loadmodule "proto_hep.so"
modparam("proto_hep", "hep_id", "[hep1]capture.example.com:9060;version=3")
```

Then any production grade troubleshooting is "open Homer, search by Call-ID, see ladder + RTCP + SDP everything."

### opensips-mi via FIFO/Datagram/HTTP

When the daemon is in a weird state but still alive, ask it directly:

```bash
opensips-cli -x mi ps
opensips-cli -x mi shm_info
opensips-cli -x mi list_blacklists
opensips-cli -x mi profile_list_dlgs in_call
opensips-cli -x mi cache_fetch local active_calls
```

## Sample Cookbook

### Registrar-proxy

(See "Sample opensips.cfg" above.)

### Mid-registrar SBC

```
####### Modules #######
loadmodule "tm.so"
loadmodule "rr.so"
loadmodule "registrar.so"
loadmodule "usrloc.so"
loadmodule "auth.so"
loadmodule "auth_db.so"
loadmodule "db_mysql.so"
loadmodule "mid_registrar.so"
loadmodule "nathelper.so"
loadmodule "rtpengine.so"

####### Module params #######
modparam("usrloc|auth_db", "db_url",
         "mysql://opensips:rw@localhost/opensips")
modparam("usrloc", "db_mode", 2)
modparam("registrar", "default_expires", 60)         # endpoints fast-refresh
modparam("mid_registrar", "mode", 1)
modparam("mid_registrar", "outgoing_expires", 3600)  # upstream lazy-refresh
modparam("nathelper", "natping_interval", 30)
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")

####### Routing #######
route {
    force_rport();
    if (nat_uac_test("23")) setflag(NAT);

    if (is_method("REGISTER")) {
        if (!www_authorize("", "subscriber")) {
            www_challenge("", "auth");
            exit;
        }
        if (isflagset(NAT)) fix_nated_register();
        mid_registrar_save("location");
        exit;
    }

    if (is_method("INVITE")) {
        if (!mid_registrar_lookup("location")) {
            sl_send_reply("404", "Not Found");
            exit;
        }
    }

    if (has_body("application/sdp")) rtpengine_offer();
    record_route();
    t_relay();
}
```

### B2BUA call recording

```
loadmodule "b2b_entities.so"
loadmodule "b2b_logic.so"
loadmodule "rtpengine.so"

modparam("b2b_logic", "script_scenario", "recording")
modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")

route {
    if (is_method("INVITE") && $avp(record) == "yes") {
        b2b_init_request("recording");
        exit;
    }
    ...
}
```

`recording.xml` orchestrates: leg A INVITE → answer with SDP, mediates to leg B, then issues `rtpengine_start_recording()` so audio is forked to the recorder while the call continues normally.

### WebRTC bridge to FreeSWITCH

```
listen=wss:0.0.0.0:443

loadmodule "tls_mgm.so"
loadmodule "proto_wss.so"
loadmodule "rtpengine.so"

modparam("rtpengine", "rtpengine_sock", "udp:127.0.0.1:22222")

route {
    if ($pr == "wss") setflag(WEBRTC);

    if (has_body("application/sdp") && is_method("INVITE")) {
        if (isflagset(WEBRTC)) {
            rtpengine_offer("trust-address replace-origin "
                            "ICE=force RTP/AVP DTLS=passive");
        } else {
            rtpengine_offer("trust-address replace-origin "
                            "ICE=remove RTP/AVP DTLS=off");
        }
    }

    $du = "sip:freeswitch.example.com:5060;transport=udp";
    record_route();
    t_relay();
}

onreply_route {
    if (has_body("application/sdp")) rtpengine_answer();
}
```

### Multi-tenant SBC

```
loadmodule "permissions.so"
loadmodule "dialplan.so"
loadmodule "drouting.so"
loadmodule "mid_registrar.so"

modparam("permissions", "address_table", "address")

route {
    # Tenant identification by source IP
    if (allow_trusted("$si", "$pr")) {
        $avp(tenant_id) = $tA;        # trusted-address attrs hold tenant
    } else {
        sl_send_reply("403", "Forbidden");
        exit;
    }

    # Per-tenant dial-string normalization
    dp_translate("$avp(tenant_id)", "$rU/$var(rU2)");
    $rU = $var(rU2);

    # Per-tenant routing group
    if (is_method("INVITE")) {
        do_routing("$avp(tenant_id)", "FW");
        record_route();
        t_relay();
    } else if (is_method("REGISTER")) {
        mid_registrar_save("location");
    }
}
```

## Deployment Topologies

### Single instance

The simplest: one OpenSIPS process, one SQL DB, optional rtpengine on the same host. Adequate up to a few thousand concurrent calls and tens of thousands of registrations.

```
+---------+
| OpenSIPS|--+--+ MySQL
|         |  |  |
|         |--+--+ rtpengine (localhost)
+---------+
```

### Dual-AS HA with VIP

Two identical OpenSIPS nodes, one active and one standby, sharing a Virtual IP via keepalived/VRRP. The DB is shared (replicated MySQL or Galera cluster), rtpengine usually has its own pair behind their own VIP.

```
   +------+    +------+
   | OS-1 |    | OS-2 |     <-- keepalived VIP floats
   +--+---+    +--+---+
      |           |
      +-----+-----+
            v
        +-------+
        | MySQL |  (replicated/galera)
        +-------+
```

State:
- `usrloc` `db_mode=2` so usrloc state survives a failover (preload from DB).
- `dialog` `db_mode=2` for dialog continuity.
- Registrations refresh quickly enough that brief gaps are invisible.

### Cluster mode (3.0+) for distributed deployment

Multiple active nodes share state via the `clusterer` module over `bin_listener`. Each node owns a shard of the work and replicates to peers; on failure, peers take over.

```
listen=bin:eth1:5555

loadmodule "clusterer.so"
modparam("clusterer", "current_id", 1)
modparam("clusterer", "db_url", "mysql://...")

# Replicated usrloc
modparam("usrloc", "cluster_mode", "full-sharing")
modparam("usrloc", "cluster_id", 1)
modparam("usrloc", "active_partitioning", 1)

# Replicated dialog
modparam("dialog", "cluster_id", 1)
modparam("dialog", "cluster_mode", "full-sharing")
```

```
+-------+   +-------+   +-------+
| OS-1  |---| OS-2  |---| OS-3  |    <-- bin replication mesh
+-------+   +-------+   +-------+
    |          |           |
    +----+-----+-----+-----+
              v
          MySQL/Galera
```

The cluster topology supports `full-sharing` (every node knows about every dialog/registration) and `sharded` (each dialog/registration owned by one node, with a small replica set for failover) modes.

## Idioms

> **B2BUA when you need mid-call control.**
> If you need to inject a re-INVITE, hold one leg, transfer with REFER+Replaces, fork audio to a recorder, or hide topology — that's a B2BUA. Routing-only proxy (Record-Route + t_relay) is far cheaper but has no mid-call agency.

> **Kamailio for stateless routing, OpenSIPS for sessions.**
> The cliché stands. OpenSIPS scales fine for stateless workloads; Kamailio can do session-aware work; but the path of least resistance for mid_registrar / B2BUA / first-class dialog is OpenSIPS, and the path of least resistance for raw multi-million-CPS routing is Kamailio.

> **Always use rtpengine over legacy rtpproxy.**
> rtpengine has kernel-mode forwarding, native SRTP, ICE, DTLS, recording, transcoding, and active development. Old `rtpproxy` is an unmaintained codepath; do not start a new project on it.

> **Mid-registrar for multi-tenant scale.**
> If you have >10k endpoints behind a single SBC, mid_registrar collapses upstream REGISTER traffic by 10–100×. The downside is one more failure domain (the SBC must be HA).

> **Reload, don't restart.**
> `opensips-cli -x mi dr_reload`, `ds_reload`, `address_reload`, `dp_reload` apply DB changes live. Reserve `systemctl restart opensips` for binary upgrades or config changes.

> **Trace everything with HEP/Homer.**
> Plain text logs and ngrep get you 80% of the way; for production-grade post-mortem and intermittent issues, a Homer cluster with HEP receive on every box pays for itself the first weird customer ticket.

> **Treat opensips.cfg like code.**
> Versioned in git, reviewed in PR, deployed via Ansible/Salt/Puppet with `opensips -C` syntax-check pre-deploy and `opensips -E` runtime test. Rollback is a redeploy of the prior config + reload.

> **Write benchmarks before tuning.**
> Don't bump `udp_workers` or change DB drivers without `sipp` numbers proving it. The default config is tuned for "good enough"; deviate based on data, not folklore.

> **Module choice is a commitment.**
> Mixing `auth_db` + `mid_registrar` + `b2b_logic` + `presence` + `rls` is fine, but each module is a piece of the runtime surface that adds attack surface, memory footprint, and update obligations. Load only what you call.

> **OpenSIPS is not a media server.**
> It does signaling and limited media-control via rtpengine. For IVR, recording playback, conferencing — pair it with FreeSWITCH or Asterisk. Do not try to bolt media into OpenSIPS itself.

## See Also

- `kamailio` — sister project; same SER ancestor, divergent design (stateless speed vs. session richness)
- `drachtio` — Node.js-friendly SIP server alternative for application-layer routing
- `rtpengine` — the kernel-accelerated media proxy used with OpenSIPS for NAT/WebRTC/SRTP
- `sip-protocol` — RFC 3261 protocol primer underlying everything in this sheet
- `asterisk` — common media-side pairing for IVR/queues/voicemail
- `freeswitch` — alternative media-side pairing, especially for WebRTC bridging

## References

- OpenSIPS official documentation — https://opensips.org/Documentation/Manuals
- OpenSIPS module documentation per release — https://opensips.org/Documentation/Modules
- The OpenSIPS Building Block Cookbook — https://opensips.org/Documentation/Tutorials
- OpenSIPS source on GitHub — https://github.com/OpenSIPS/opensips
- OpenSIPS-CLI source — https://github.com/OpenSIPS/opensips-cli
- OpenSIPS Solutions (commercial) — https://www.opensips-solutions.com
- OpenSIPS Summit (annual conference, recordings online) — https://opensips.org/events
- RFC 3261 — SIP: Session Initiation Protocol
- RFC 3265 — SUBSCRIBE/NOTIFY framework
- RFC 3856 — Presence event package
- RFC 3863 — PIDF presence document format
- RFC 3842 — Message Waiting Indication event package
- RFC 4235 — Dialog event package (BLF)
- RFC 4662 — Resource List Server (RLS)
- RFC 5875 — XCAP Diff event package
- RFC 7118 — SIP-over-WebSocket transport
- RFC 8656 — TURN (used by WebRTC + rtpengine)
- rtpengine project — https://github.com/sipwise/rtpengine
- Homer SIP capture project — https://sipcapture.org
