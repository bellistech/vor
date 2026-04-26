# drachtio

Node.js SIP server middleware — the "Express for SIP" — built on a C++ resiprocate-based daemon (drachtio-server) controlled over TCP from JavaScript via the drachtio-srf library, with optional FreeSWITCH media control via drachtio-fsmrf.

## Setup

drachtio is a Node.js SIP application framework. It is not a SIP stack you embed in your Node process — it is a separately running C++ daemon (drachtio-server) that owns the SIP socket, parses messages, and forwards them as JSON-encoded events over a long-lived TCP control channel to your Node.js application. The Node side uses the `drachtio-srf` ("Signaling Resource Framework") npm package, which presents an Express-like API where SIP methods are handled by registered functions and a middleware chain executes per request.

The framing "Express for SIP" is deliberate. If you have written `app.get('/path', (req, res) => res.send(200, body))` you already understand 80% of drachtio. The other 20% is the SIP semantics — Via headers, dialog state, B2BUA mechanics, SDP negotiation — that the framework helps you handle but does not hide.

The project was created and is maintained by Dave Horton (github @davehorton). drachtio is the SIP-signaling foundation under jambonz, an open-source webhook-driven voice platform that competes with Twilio at the application layer. Documentation lives at drachtio.org with API reference at davehorton.github.io/drachtio-srf. The C++ server is at github.com/davehorton/drachtio-server, the Node SDK at github.com/davehorton/drachtio-srf, and the FreeSWITCH integration at github.com/davehorton/drachtio-fsmrf.

drachtio fills a niche between Kamailio/OpenSIPS (very fast, stateless-routing-first, scripted in their own DSL) and FreeSWITCH (media-first, scripted via Lua/JavaScript/dialplan XML). It is signaling-focused: it parses SIP, executes your routing logic in Node, and either proxies the request, B2BUA's it (creating two correlated dialogs), or originates new calls. For media (audio playback, DTMF collection, recording, conferencing) you typically pair it with FreeSWITCH or rtpengine.

```bash
# Install drachtio-server (Ubuntu)
wget https://github.com/davehorton/drachtio-server/releases/download/v0.8.27/drachtio_0.8.27_amd64.deb
sudo dpkg -i drachtio_0.8.27_amd64.deb

# Or build from source
git clone --depth=50 --recursive https://github.com/davehorton/drachtio-server.git
cd drachtio-server && ./autogen.sh && ./configure && make && sudo make install

# Node.js side
mkdir my-sip-app && cd my-sip-app
npm init -y
npm install drachtio-srf

# Optional: FreeSWITCH media bridge
npm install drachtio-fsmrf

# Start daemon (after configuration)
sudo systemctl start drachtio
sudo systemctl enable drachtio

# Run app
node app.js
```

## Architecture

drachtio is a three-tier architecture:

```
┌──────────────────────────────────────┐
│    Your Node.js Application          │  ← business logic
│    (drachtio-srf middleware)         │
└─────────────┬────────────────────────┘
              │  TCP control channel (port 9022)
              │  JSON-encoded SIP events + commands
              │  AUTH on connect with shared secret
              ▼
┌──────────────────────────────────────┐
│    drachtio-server (C++ daemon)      │  ← protocol I/O
│    based on resiprocate stack        │
└─────────────┬────────────────────────┘
              │  SIP messages (UDP/TCP/TLS/WS/WSS)
              │  ports 5060/5061/etc
              ▼
            SIP Network
```

The daemon does the heavy lifting that should not be in JavaScript: socket I/O, SIP parsing, transaction state machines, retransmissions, dialog tracking. Your Node app receives parsed events ("INVITE arrived from 192.0.2.1, here is the JSON of the request") and issues commands ("send 200 OK with this SDP" or "proxy this INVITE to sip:downstream@example.com").

The control channel is bidirectional and persistent. The Node app connects to the daemon (not the other way), authenticates with a shared secret, and then receives events and sends commands until disconnection. This decoupling means:

- you can restart the Node app without losing in-flight SIP transactions if you do it quickly enough (the daemon keeps state)
- you can run the Node app on a different host (TLS the control channel)
- multiple Node apps can connect to the same daemon for HA (with care)
- the daemon can run as `root` to bind low ports, the Node app runs unprivileged

The TCP control channel uses a custom JSON-RPC-ish line-oriented protocol — each message is one JSON object terminated by `\r\n`. drachtio-srf hides this entirely; you only see SIP-level objects.

```
                  drachtio control protocol (simplified)
        Node                                      Daemon
         │                                         │
         │  ──── AUTHENTICATE secret ────────►     │
         │  ◄──── OK ─────────────────────────     │
         │                                         │
         │                       ◄──── SIP INVITE on port 5060
         │  ◄──── SIP/INVITE event (JSON) ────     │
         │                                         │
         │  ──── send 100 Trying ─────────────►    │
         │                                  ────► SIP/2.0 100 Trying
         │  ──── proxy to sip:next@host ──────►    │
         │                                  ────► INVITE sip:next@host
         │  ◄──── 200 OK from upstream ────────    │
         │                                         │
```

## drachtio-server Daemon

The daemon is the C++ side. It is built on resiprocate, a mature SIP stack also used by Cisco and various commercial products. drachtio-server adds the control-channel listener, transaction routing to/from the Node side, and a configurable request-handler dispatch.

Default service:

```bash
# Verify install
which drachtio
drachtio --version

# Default systemd unit (installed by .deb)
systemctl status drachtio

# Logs
journalctl -u drachtio -f

# Or syslog destination if configured in conf
tail -f /var/log/drachtio.log
```

By default the daemon listens on:

- `0.0.0.0:5060/udp` and `0.0.0.0:5060/tcp` for SIP (varies by config)
- `0.0.0.0:9022/tcp` for the control channel (where your Node app connects)
- optionally `0.0.0.0:5061/tls` for SIP over TLS
- optionally `0.0.0.0:8443/wss` for SIP over WebSocket Secure

The daemon is single-process, single-threaded for transaction processing (resiprocate is event-loop based). It is fast enough for tens of thousands of transactions per second on commodity hardware. For higher loads, run multiple daemons behind a UDP/TCP load balancer.

```bash
# Run in foreground for debugging
drachtio --daemon false --loglevel debug

# Specify config file
drachtio -f /etc/drachtio.conf.xml

# Show running configuration
drachtio --print-config

# Generate sample config
drachtio --print-default-config > drachtio.conf.xml.sample
```

## drachtio.conf.xml

The single source of truth for daemon configuration. Lives at `/etc/drachtio.conf.xml` by default. Sample structure:

```xml
<drachtio>
  <admin port="9022" secret="cymru" tls-port="9023">0.0.0.0</admin>

  <sip>
    <contacts>
      <contact>sip:*:5060;transport=udp,tcp</contact>
      <contact>sips:*:5061;transport=tls</contact>
      <contact>sip:*:8443;transport=wss</contact>
    </contacts>

    <spammer-detection>
      <action>silent-discard</action>
      <log-events>false</log-events>
    </spammer-detection>

    <outbound-proxy>sip:upstream-proxy.example.com:5060</outbound-proxy>

    <tls>
      <key-file>/etc/drachtio/tls/server.key</key-file>
      <cert-file>/etc/drachtio/tls/server.crt</cert-file>
      <chain-file>/etc/drachtio/tls/chain.pem</chain-file>
      <dh-param>/etc/drachtio/tls/dhparam.pem</dh-param>
    </tls>
  </sip>

  <request-handlers>
    <invite>http://localhost:3000/invite</invite>
    <register>http://localhost:3000/register</register>
  </request-handlers>

  <cdrs>true</cdrs>

  <monitoring>
    <stats-port>8080</stats-port>
  </monitoring>

  <logging>
    <console/>
    <file>/var/log/drachtio.log</file>
    <syslog>local0</syslog>
    <loglevel>info</loglevel>
    <sofia-loglevel>3</sofia-loglevel>
  </logging>

  <homer>
    <id>22</id>
    <hep-server-address>192.168.1.50</hep-server-address>
    <hep-server-port>9060</hep-server-port>
    <password>myhep</password>
  </homer>
</drachtio>
```

Key sections:

- **admin** — TCP port and shared secret for the Node-control channel. The Node app must use the same secret on `srf.connect()`.
- **sip/contacts** — listening points. Use `sip:*:5060` to bind to all interfaces, or specific IP. Multi-listen by listing multiple `<contact>` entries.
- **spammer-detection** — drop or 403 SIP traffic that matches known scanner patterns (sipvicious, sundayddr, etc.).
- **outbound-proxy** — default route for outbound originations from your app.
- **tls** — certificates for SIP-TLS or WSS.
- **request-handlers** — *optional* HTTP webhook routing (alternative to the persistent TCP control channel for stateless apps). Most apps use the control channel via drachtio-srf and ignore this.

## drachtio-srf

The Node.js SDK. `srf` stands for "Signaling Resource Framework". The npm package is `drachtio-srf`. It exports a `Srf` class.

```js
const Srf = require('drachtio-srf');
const srf = new Srf();
```

The `Srf` instance is your handle to the daemon connection. After connecting, register handlers per SIP method:

```js
srf.invite((req, res) => { /* handle INVITE */ });
srf.register((req, res) => { /* handle REGISTER */ });
srf.options((req, res) => res.send(200));
srf.subscribe((req, res) => { /* handle SUBSCRIBE */ });
srf.notify((req, res) => res.send(200));
srf.message((req, res) => { /* handle MESSAGE (SIP IM) */ });
srf.publish((req, res) => res.send(200));
srf.refer((req, res) => { /* handle REFER (call transfer) */ });
srf.info((req, res) => { /* handle in-dialog INFO */ });
```

The `req` is a `SipRequest` object with the parsed SIP message. `res` is a `SipResponse` you call to send replies. The pattern is intentionally Express-mirrored — if you know Express, you know the surface area.

```js
// req has these (and more):
req.method            // "INVITE"
req.uri               // "sip:bob@example.com"
req.from              // parsed From header object
req.to                // parsed To header object
req.callId            // Call-ID value
req.cseq              // CSeq value
req.headers           // map of header-name to value
req.body              // SDP or message body as string
req.protocol          // "udp", "tcp", "tls", "ws", "wss"
req.source_address    // remote IP
req.source_port       // remote port
req.has(headerName)   // boolean
req.get(headerName)   // header value
req.getParsedHeader(name) // structured (e.g. From, To, Contact)
```

```js
// res methods:
res.send(200);                               // status only
res.send(200, { headers: { 'X-Foo': 'bar' }, body: sdp });
res.send(486, 'Busy Here');                  // status + reason phrase
res.send(401, { headers: { 'WWW-Authenticate': '...' } });
```

## Connecting to Daemon

Two patterns: explicit `connect()` or constructor with options. Always `await` to ensure routes register only after the channel is up.

```js
const Srf = require('drachtio-srf');
const srf = new Srf();

// Pattern 1: explicit
async function main() {
  await srf.connect({
    host: 'localhost',
    port: 9022,
    secret: 'cymru'
  });
  console.log('connected to drachtio');

  srf.invite((req, res) => res.send(486)); // busy everyone
}

main().catch(err => {
  console.error('fatal', err);
  process.exit(1);
});
```

```js
// Pattern 2: event-driven
const srf = new Srf();
srf.on('connect', (err, hostport) => {
  if (err) return console.error(err);
  console.log(`connected to drachtio listening on ${hostport}`);
});
srf.on('error', err => console.error('drachtio error', err));

srf.connect({ host: '127.0.0.1', port: 9022, secret: 'cymru' });
```

The connection is persistent. If the daemon dies or the network drops, drachtio-srf will attempt reconnection automatically (configurable). Mid-call, dialogs survive transient drops if both sides reconnect quickly enough.

```js
// Multi-server failover
srf.connect([
  { host: 'drachtio-1.internal', port: 9022, secret: 'cymru' },
  { host: 'drachtio-2.internal', port: 9022, secret: 'cymru' }
]);
```

## Inbound Routing

The most common pattern is INVITE handling: receive an inbound call and decide what to do.

```js
srf.invite((req, res) => {
  console.log(`inbound INVITE from ${req.from.uri} to ${req.to.uri}`);

  // Stateless reject:
  if (isBlacklisted(req.source_address)) {
    return res.send(403, 'Forbidden');
  }

  // Stateless busy:
  if (allLinesBusy()) {
    return res.send(486, 'Busy Here');
  }

  // Proxy to downstream:
  return req.proxy({ destination: 'sip:downstream.example.com' });
});
```

Three response models:

- **Stateless reply** — `res.send(NNN)` and you are done. Suitable for early rejections (403, 404, 486, etc.).
- **Stateful proxy** — `req.proxy(opts)` forwards the INVITE to one or more destinations and proxies all responses back. The dialog (if accepted) is between caller and downstream; drachtio is in the path for ACK/BYE if `recordRoute: true`.
- **B2BUA** — `srf.createB2BUA(req, res, dest, opts)` terminates the inbound dialog and creates a *new* outbound dialog. Two dialogs, fully decoupled, and your Node app sees every in-dialog message.

```js
srf.invite(async (req, res) => {
  // proxy mode
  try {
    const result = await req.proxy({
      destination: 'sip:upstream.example.com',
      recordRoute: true,
      provisionalTimeout: '2s',
      finalTimeout: '60s'
    });
    console.log('proxy completed', result.connected);
  } catch (err) {
    console.error('proxy failed', err);
  }
});
```

## Express-Style Middleware

`srf.use(fn)` registers middleware that runs before the method handler.

```js
function logger(req, res, next) {
  console.log(`${req.protocol} ${req.method} ${req.uri} from ${req.source_address}`);
  next();
}

function auth(req, res, next) {
  if (!req.has('Authorization')) {
    return res.send(401, {
      headers: {
        'WWW-Authenticate': `Digest realm="example.com", nonce="${randomNonce()}", algorithm=MD5, qop="auth"`
      }
    });
  }
  // validate digest...
  next();
}

srf.use(logger);
srf.use('register', auth); // method-specific
srf.use('invite', auth);

srf.register((req, res) => res.send(200));
```

Method-specific middleware: `srf.use(methodOrArray, fn)`. The chain runs in registration order. Calling `next(err)` short-circuits to the error handler.

```js
function errorHandler(err, req, res, next) {
  console.error('middleware error', err);
  res.send(500, 'Server Internal Error');
}
srf.use(errorHandler);
```

## Stateful Proxy Mode

Proxying preserves the call's signaling end-to-end while letting you make routing decisions. Drachtio implements RFC 3261 stateful proxy semantics — Via, Record-Route, transaction matching, retransmissions all handled.

```js
srf.invite((req, res) => {
  return req.proxy({
    destination: ['sip:primary@upstream:5060', 'sip:backup@upstream:5060'],
    recordRoute: true,
    forking: 'sequential',          // or 'parallel'
    provisionalTimeout: '2s',
    finalTimeout: '30s',
    followRedirects: true,
    rejectUnauthorized: false,
    headers: {
      'X-Tenant-Id': req.get('X-Tenant-Id') || 'default',
      'P-Asserted-Identity': `<sip:${req.from.uri.user}@trusted.example.com>`
    },
    remainInDialog: true            // get ACK/BYE for billing
  }).then(result => {
    console.log(`proxy outcome: ${result.connected ? 'connected' : 'failed'}, status=${result.finalStatus}`);
  });
});
```

Options summary:

- `destination` — single URI or array
- `forking` — sequential (try in order) or parallel (race)
- `recordRoute` — add Record-Route so we stay in the path
- `provisionalTimeout` / `finalTimeout` — RFC 3261 Timer C/D analogs
- `followRedirects` — chase 3xx
- `headers` — append/override request headers on the outbound leg
- `remainInDialog` — keep dialog state so ACK/BYE come back to us
- `preserveRouting` — preserve original Route headers (advanced)

## B2BUA Mode

Back-to-Back User Agent: terminate the inbound INVITE, then originate a new INVITE to the destination, gluing the two dialogs together. You see (and can rewrite) every in-dialog message.

```js
srf.invite(async (req, res) => {
  try {
    const { uas, uac } = await srf.createB2BUA(req, res, 'sip:bob@upstream.example.com', {
      proxyRequestHeaders: ['from', 'subject'],
      proxyResponseHeaders: ['contact'],
      headers: {
        'X-Originator': 'b2bua-bridge'
      },
      auth: { username: 'me', password: 'secret' }
    });

    console.log('B2BUA bridged');

    // hangup propagation
    uas.on('destroy', () => uac.destroy());
    uac.on('destroy', () => uas.destroy());

    // mid-call info
    uas.on('info', (req, res) => {
      console.log('uas got INFO', req.body);
      uac.request({ method: 'INFO', body: req.body });
      res.send(200);
    });

  } catch (err) {
    console.error('B2BUA failed', err);
    if (!res.finalResponseSent) res.send(500);
  }
});
```

Returns `{uas, uac}` — two `Dialog` handles. Both have `.destroy()`, `.request()`, `.modify()`, and event emitters for `bye`, `info`, `refer`, `notify`, `update`.

```js
// Rewriting SDP between legs (e.g. transcoding hint):
const { uas, uac } = await srf.createB2BUA(req, res, dest, {
  localSdpA: customSdpForCallerSide,    // SDP we offer caller
  localSdpB: customSdpForCalleeSide,    // SDP we offer callee
  proxyRequestHeaders: ['*']
});
```

## UAC Mode

Originate calls (you are the User Agent Client).

```js
async function makeCall(toUri) {
  const uac = await srf.createUAC(toUri, {
    headers: {
      'From': '<sip:robocaller@example.com>',
      'Contact': '<sip:app@app.example.com:5060>'
    },
    localSdp: ourOfferSdp,
    auth: { username: 'me', password: 'secret' }
  }, {
    cbProvisional: response => console.log('provisional', response.status),
    cbRequest: req => console.log('request sent', req.uri)
  });

  console.log('call established', uac.sip.callId);

  uac.on('destroy', () => console.log('remote hung up'));

  setTimeout(() => uac.destroy(), 30000); // hangup after 30s
}
```

## UAS Mode

Accept calls explicitly when you do not want to delegate to B2BUA or proxy. Useful for media-server applications where Node *is* the endpoint.

```js
srf.invite(async (req, res) => {
  try {
    const uas = await srf.createUAS(req, res, {
      localSdp: generateOurSdp(req.body)
    });
    console.log('call answered', uas.sip.callId);

    uas.on('destroy', () => console.log('caller hung up'));
    uas.on('info', (req, res) => res.send(200));

  } catch (err) {
    console.error('UAS failed', err);
  }
});
```

## Registration Handling

REGISTER requests authenticate the device and update its current contact (where to send INVITEs). drachtio gives you the request; you store the binding in your DB.

```js
const Redis = require('ioredis');
const redis = new Redis();

srf.register(async (req, res) => {
  // 1. Auth (digest middleware ran before us, set req.authorization)
  const aor = req.to.uri;
  const contact = req.getParsedHeader('Contact')[0].uri;
  const expires = parseInt(req.get('Expires') || '3600');

  if (expires === 0) {
    // de-registration
    await redis.del(`reg:${aor}`);
    return res.send(200);
  }

  await redis.setex(`reg:${aor}`, expires, JSON.stringify({
    contact,
    source: `${req.source_address}:${req.source_port}`,
    protocol: req.protocol,
    registeredAt: Date.now()
  }));

  res.send(200, {
    headers: {
      'Contact': `${contact};expires=${expires}`,
      'Date': new Date().toUTCString()
    }
  });
});

// Later — INVITE routing uses the registration:
srf.invite(async (req, res) => {
  const targetAor = req.uri;
  const reg = await redis.get(`reg:${targetAor}`);
  if (!reg) return res.send(404, 'Not Found');
  const { contact } = JSON.parse(reg);
  return req.proxy({ destination: contact });
});
```

## drachtio-fsmrf

The FreeSWITCH Media Resource Framework. Bridges drachtio (signaling) with FreeSWITCH (media: codec transcoding, IVR prompts, recording, conferencing). The npm package is `drachtio-fsmrf`.

```js
const Srf = require('drachtio-srf');
const Mrf = require('drachtio-fsmrf');

const srf = new Srf();
const mrf = new Mrf(srf);

await srf.connect({ host: 'localhost', port: 9022, secret: 'cymru' });
const ms = await mrf.connect({
  address: '127.0.0.1',
  port: 8021,
  secret: 'ClueCon'
});
```

`ms` is a `MediaServer` representing your FreeSWITCH instance. From it you create `Endpoints` (one per call leg you want to media-handle).

```js
const endpoint = await ms.createEndpoint();
// endpoint.local.sdp is the SDP FreeSWITCH offers
```

drachtio-fsmrf speaks ESL (Event Socket Layer) to FreeSWITCH internally — you do not write ESL commands, you call high-level methods like `endpoint.play()`.

## drachtio-fsmrf Workflow

Standard pattern for media-handled inbound call:

```js
srf.invite(async (req, res) => {
  // 1. Allocate FS endpoint
  const endpoint = await ms.createEndpoint();

  // 2. Send 200 OK with FS's SDP back to caller (UAS)
  const uas = await srf.createUAS(req, res, { localSdp: endpoint.local.sdp });

  // Now caller's RTP flows to FreeSWITCH

  // 3. Tell FS to receive caller's SDP
  await endpoint.modify(uas.remote.sdp);

  // 4. Run media app
  try {
    await endpoint.play('ivr/8000/welcome.wav');
    const dtmf = await endpoint.play_collect({
      file: 'ivr/8000/please_press_1.wav',
      max: 1, timeout: 5
    });
    console.log('caller pressed', dtmf.digits);

    if (dtmf.digits === '1') {
      // bridge to agent
      await endpoint.bridge('sofia/external/sip:agent@upstream.example.com');
    } else {
      await endpoint.play('ivr/8000/goodbye.wav');
    }
  } finally {
    // 5. Cleanup
    await endpoint.destroy();
    uas.destroy();
  }
});
```

Key endpoint methods:

```js
endpoint.play(audioFile)                          // play audio
endpoint.play_collect({file, min, max, timeout})  // play + collect DTMF
endpoint.say({text, voice, language})             // TTS via mod_say
endpoint.speak({text, voice})                     // TTS via mod_tts
endpoint.record({path, maxDuration})              // record audio
endpoint.bridge(uri)                              // bridge to another leg
endpoint.transfer(extension)                      // FS transfer
endpoint.modify(sdp)                              // re-INVITE / SDP change
endpoint.set('variable', 'value')                 // FS channel variable
endpoint.api('uuid_kill', endpoint.uuid)          // raw FS API command
endpoint.destroy()                                // hangup + free resources
```

## IVR Pattern

drachtio handles SIP, fsmrf controls FS for media. The IVR menu lives in JavaScript:

```js
async function mainMenu(endpoint) {
  while (true) {
    const result = await endpoint.play_collect({
      file: 'ivr/main_menu.wav',
      max: 1,
      timeout: 5,
      invalidFile: 'ivr/invalid.wav'
    });

    switch (result.digits) {
      case '1':
        await salesMenu(endpoint);
        return;
      case '2':
        await supportMenu(endpoint);
        return;
      case '3':
        await endpoint.bridge('sofia/external/sip:operator@upstream');
        return;
      default:
        await endpoint.play('ivr/invalid.wav');
        // loop
    }
  }
}

srf.invite(async (req, res) => {
  const endpoint = await ms.createEndpoint();
  const uas = await srf.createUAS(req, res, { localSdp: endpoint.local.sdp });
  await endpoint.modify(uas.remote.sdp);
  try {
    await mainMenu(endpoint);
  } catch (err) {
    console.error('ivr error', err);
  } finally {
    await endpoint.destroy();
    uas.destroy();
  }
});
```

## WebRTC + drachtio

drachtio-server can listen on `wss://` for SIP-over-WebSocket, the standard browser-to-PBX transport. Pair with rtpengine for media (ICE/DTLS-SRTP termination) since browsers cannot speak plain RTP.

```xml
<!-- drachtio.conf.xml -->
<sip>
  <contacts>
    <contact>sips:*:8443;transport=wss</contact>
    <contact>sip:*:5060;transport=udp,tcp</contact>
  </contacts>
  <tls>
    <key-file>/etc/drachtio/tls/cert.key</key-file>
    <cert-file>/etc/drachtio/tls/cert.crt</cert-file>
  </tls>
</sip>
```

```js
// Node app: detect WebRTC INVITEs and offload to rtpengine
const Rtpengine = require('rtpengine-client');
const rtpengine = new Rtpengine.Client();

srf.invite(async (req, res) => {
  const isWebRTC = req.body.includes('UDP/TLS/RTP/SAVPF');
  if (isWebRTC) {
    const offer = await rtpengine.offer({
      'call-id': req.callId,
      'from-tag': req.from.params.tag,
      sdp: req.body,
      flags: ['trust address', 'replace origin', 'replace session connection']
    });
    // offer.sdp is the rewritten SDP for the SIP-side leg
    return req.proxy({ destination: dest, headers: { 'Content-Type': 'application/sdp' }, body: offer.sdp });
  }
  // non-WebRTC path
  return req.proxy({ destination: dest });
});
```

## Authentication

Digest authentication per RFC 3261 §22 / RFC 2617. drachtio-srf provides `digestChallenge` and `parseAuthHeader` helpers; community packages like `drachtio-mw-digest-auth` package the full flow.

```js
const digestAuth = require('drachtio-mw-digest-auth');

srf.use('register', digestAuth({
  realm: 'example.com',
  passwordLookup: async (username, realm) => {
    const user = await db.users.findOne({ username });
    return user ? user.sipPassword : null;
  }
}));

srf.register((req, res) => {
  // by here req.authorization is populated and verified
  res.send(200);
});
```

Manual digest:

```js
const crypto = require('crypto');
function md5(s) { return crypto.createHash('md5').update(s).digest('hex'); }

function calculateDigest({username, realm, password, method, uri, nonce, nc, cnonce, qop}) {
  const ha1 = md5(`${username}:${realm}:${password}`);
  const ha2 = md5(`${method}:${uri}`);
  return md5(`${ha1}:${nonce}:${nc}:${cnonce}:${qop}:${ha2}`);
}
```

## Database Integration

Node.js's full DB ecosystem is yours. Common choices for telephony apps:

```js
// PostgreSQL (pg)
const { Pool } = require('pg');
const pgpool = new Pool({ connectionString: process.env.DATABASE_URL });
const result = await pgpool.query('SELECT * FROM subscribers WHERE aor=$1', [aor]);

// MongoDB (mongoose)
const mongoose = require('mongoose');
await mongoose.connect(process.env.MONGO_URL);
const Reg = mongoose.model('Reg', new mongoose.Schema({ aor: String, contact: String, expires: Date }));
await Reg.findOne({ aor });

// Redis (ioredis) — fast for registrar binding cache
const Redis = require('ioredis');
const redis = new Redis();
await redis.setex(`reg:${aor}`, 3600, JSON.stringify(binding));
```

Pattern: hot lookups (registrations, rate-limit counters) in Redis; persistent records (subscribers, CDRs, route plans) in Postgres or Mongo.

## Logging

drachtio-srf uses `pino` internally. You can pass your own logger:

```js
const pino = require('pino');
const logger = pino({ level: 'debug' });

const srf = new Srf();
srf.locals.logger = logger;

srf.invite((req, res) => {
  const log = logger.child({ callId: req.callId, from: req.from.uri });
  log.info('inbound INVITE');
  // ...
  log.info({ duration: 30 }, 'call ended');
});
```

`pino-pretty` for dev:

```bash
node app.js | npx pino-pretty
```

Per-request trace ID:

```js
const { v4: uuid } = require('uuid');
srf.use((req, res, next) => {
  req.traceId = uuid();
  req.log = logger.child({ traceId: req.traceId, callId: req.callId });
  next();
});
```

## Error Handling

Async middleware needs try/catch:

```js
srf.invite(async (req, res) => {
  try {
    const dest = await db.lookupDestination(req.uri);
    if (!dest) return res.send(404);
    await req.proxy({ destination: dest });
  } catch (err) {
    req.log?.error({ err }, 'invite handler failed');
    if (!res.finalResponseSent) res.send(500);
  }
});
```

Process-level safety net:

```js
process.on('unhandledRejection', err => {
  console.error('unhandled rejection', err);
  // do NOT exit on every rejection — log and continue
});
process.on('uncaughtException', err => {
  console.error('uncaught exception', err);
  // exit and let supervisor restart, after letting in-flight requests drain
  process.exitCode = 1;
});
```

Custom error response middleware:

```js
srf.use((err, req, res, next) => {
  if (res.finalResponseSent) return;
  if (err.code === 'ENOTFOUND') return res.send(404);
  if (err.code === 'ETIMEDOUT') return res.send(408);
  res.send(500);
});
```

## drachtio-cli

Administrative CLI (separate npm: `drachtio-cli`):

```bash
npm install -g drachtio-cli

# Connect and show server status
drachtio-cli --host localhost --port 9022 --secret cymru status

# List active dialogs / sessions
drachtio-cli sessions

# Show configured SIP contacts (listening points)
drachtio-cli contacts

# Show in-flight transactions
drachtio-cli transactions

# Reload configuration without restart
drachtio-cli reload

# Set log level on the fly
drachtio-cli loglevel debug
```

## drachtio-cluster

For higher load: run multiple drachtio-server instances behind a SIP-aware load balancer (Kamailio dispatcher, OpenSIPS dispatcher, F5 BIG-IP with SIP profile, etc.). Each daemon has its own Node app or a shared app (one Node connection per daemon). Use Redis or a database for shared state (registrations, rate-limit counters).

```
                      ┌───────────────┐
              SIP ──► │ Kamailio LB   │ ──► drachtio-1 ─► node-app-1 ─► Redis
                      │ (dispatcher)  │ ──► drachtio-2 ─► node-app-2 ─►   ▲
                      └───────────────┘ ──► drachtio-3 ─► node-app-3 ─► (shared state)
```

Caveat: in-dialog requests (BYE, re-INVITE) must come back to the same drachtio that handled the initial INVITE. Use Record-Route with a unique URI per drachtio, or sticky-session in the LB on Call-ID hash.

## Sample Apps

Existing public examples:

- **drachtio-registrar** — minimal SIP registrar with Redis backend
- **drachtio-bridge** — minimal B2BUA call bridge
- **drachtio-ivr** — IVR with fsmrf
- **drachtio-rtpengine-webrtc** — WebRTC-to-SIP gateway
- **drachtio-simple-sbc** — basic SBC pattern
- **jambonz** — full webhook-driven voice platform

```bash
git clone https://github.com/davehorton/drachtio-srf.git
ls drachtio-srf/examples/
```

## drachtio + jambonz

jambonz (jambonz.org) is a higher-level voice platform built on drachtio. Where drachtio gives you SIP middleware in Node, jambonz lets you write call flows as webhooks (HTTP POSTs) returning JSON verbs (very similar to Twilio TwiML/JSON):

```json
[
  { "verb": "say", "text": "Hello, please choose 1 for sales or 2 for support." },
  { "verb": "gather", "input": ["digits"], "numDigits": 1, "actionHook": "/menu-pick" }
]
```

If you want the productivity of webhooks-not-code, jambonz; if you want fine-grained control or unusual flows, raw drachtio.

## Comparison vs Kamailio/OpenSIPS

|                  | drachtio                        | Kamailio / OpenSIPS              |
|------------------|---------------------------------|----------------------------------|
| Language         | JavaScript (Node.js)            | Custom config DSL (kamailio.cfg) |
| Performance      | High (10k+ tps)                 | Very high (100k+ tps stateless)  |
| Paradigm         | B2BUA-first, also stateful proxy| Stateless proxy first, also stateful |
| Dev experience   | Express-like, npm ecosystem     | Steeper learning curve, modules  |
| Built-in features| Less (lean core, npm modules)   | More (everything in modules)     |
| Memory model     | V8 GC (watch for heap)          | Native C (lighter)               |
| Best for         | App-layer logic, B2BUA, WebRTC GW| High-volume routing, registrar farms |

Many production setups use both: Kamailio at the edge for high-volume routing/load-balancing, drachtio behind for application logic.

## Comparison vs FreeSWITCH ESL

ESL (Event Socket Layer) is FreeSWITCH's scripting interface — you control FreeSWITCH from Node/Python/etc. drachtio does not control FreeSWITCH; it is a separate SIP signaling layer. They are complementary:

- ESL: media (transcoding, conferencing, IVR, recording) and FS-side logic
- drachtio: SIP signaling (proxy, B2BUA, registrar, presence)
- drachtio-fsmrf: glues drachtio to FreeSWITCH so one Node app does both

A typical architecture:

```
Customer SIP ──► drachtio (signaling) ──► FreeSWITCH (media) ──► PSTN gateway
                       │                        │
                       └────── drachtio-fsmrf bridge ─────┘
```

## Sample Code

### Hello World

```js
// hello.js — answer every INVITE with 200 OK and dummy SDP
const Srf = require('drachtio-srf');
const srf = new Srf();

const dummySdp = [
  'v=0',
  'o=- 0 0 IN IP4 127.0.0.1',
  's=-',
  'c=IN IP4 127.0.0.1',
  't=0 0',
  'm=audio 16000 RTP/AVP 0',
  'a=rtpmap:0 PCMU/8000'
].join('\r\n');

(async () => {
  await srf.connect({ host: 'localhost', port: 9022, secret: 'cymru' });
  console.log('connected');

  srf.invite(async (req, res) => {
    console.log('INVITE from', req.from.uri);
    try {
      const uas = await srf.createUAS(req, res, { localSdp: dummySdp });
      uas.on('destroy', () => console.log('caller hung up'));
    } catch (err) {
      console.error('UAS failed', err);
    }
  });

  srf.options((req, res) => res.send(200));
})();
```

### Minimal Registrar

```js
// registrar.js
const Srf = require('drachtio-srf');
const Redis = require('ioredis');
const digestAuth = require('drachtio-mw-digest-auth');

const srf = new Srf();
const redis = new Redis();

srf.use('register', digestAuth({
  realm: 'example.com',
  passwordLookup: async (user, realm) => {
    const r = await redis.hget('users', user);
    return r ? JSON.parse(r).password : null;
  }
}));

srf.register(async (req, res) => {
  const aor = req.to.uri;
  const expires = parseInt(req.get('Expires') || '3600', 10);
  const contactHdr = req.getParsedHeader('Contact')[0];

  if (expires === 0) {
    await redis.del(`reg:${aor}`);
    return res.send(200);
  }

  await redis.setex(`reg:${aor}`, expires, JSON.stringify({
    contact: contactHdr.uri,
    source: `${req.source_address}:${req.source_port}`,
    proto: req.protocol
  }));
  res.send(200, { headers: { Contact: `${contactHdr.uri};expires=${expires}` } });
});

(async () => {
  await srf.connect({ host: 'localhost', port: 9022, secret: 'cymru' });
  console.log('registrar listening');
})();
```

### Minimal B2BUA Bridge

```js
// bridge.js — bridge inbound INVITE to a fixed downstream
const Srf = require('drachtio-srf');
const srf = new Srf();

const DOWNSTREAM = 'sip:gateway.example.com';

srf.invite(async (req, res) => {
  try {
    const { uas, uac } = await srf.createB2BUA(req, res, DOWNSTREAM, {
      proxyRequestHeaders: ['from', 'to', 'subject'],
      proxyResponseHeaders: ['contact'],
      headers: { 'X-B2BUA': 'drachtio-bridge' }
    });
    uas.on('destroy', () => uac.destroy());
    uac.on('destroy', () => uas.destroy());
  } catch (err) {
    console.error('bridge failed', err);
    if (!res.finalResponseSent) res.send(500);
  }
});

(async () => {
  await srf.connect({ host: 'localhost', port: 9022, secret: 'cymru' });
  console.log('B2BUA bridge ready');
})();
```

## Configuration Files

drachtio.conf.xml is the daemon config; the Node app config is whatever you want, typically `.env` plus a JSON/YAML file.

```bash
# .env
DRACHTIO_HOST=localhost
DRACHTIO_PORT=9022
DRACHTIO_SECRET=cymru
FREESWITCH_HOST=localhost
FREESWITCH_PORT=8021
FREESWITCH_SECRET=ClueCon
REDIS_URL=redis://localhost:6379
DATABASE_URL=postgres://app@localhost/voicedb
LOG_LEVEL=info
```

```js
// config.js
require('dotenv').config();
module.exports = {
  drachtio: {
    host: process.env.DRACHTIO_HOST,
    port: parseInt(process.env.DRACHTIO_PORT, 10),
    secret: process.env.DRACHTIO_SECRET
  },
  freeswitch: {
    address: process.env.FREESWITCH_HOST,
    port: parseInt(process.env.FREESWITCH_PORT, 10),
    secret: process.env.FREESWITCH_SECRET
  },
  redis: { url: process.env.REDIS_URL },
  pg: { url: process.env.DATABASE_URL },
  logging: { level: process.env.LOG_LEVEL || 'info' }
};
```

## Deployment

Three components to supervise:

1. **drachtio-server** — systemd unit (shipped in .deb)
2. **Node app** — PM2 or systemd (supervisor that respawns on crash)
3. **FreeSWITCH** (optional) — its own systemd unit

```bash
# PM2
npm install -g pm2
pm2 start app.js --name sip-app --max-memory-restart 1G
pm2 save
pm2 startup       # generates systemd unit for PM2 itself

# Or systemd directly for the Node app
sudo tee /etc/systemd/system/sip-app.service <<'EOF'
[Unit]
Description=drachtio Node.js SIP app
After=network.target drachtio.service
Requires=drachtio.service

[Service]
Type=simple
User=sip-app
WorkingDirectory=/srv/sip-app
ExecStart=/usr/bin/node app.js
Restart=on-failure
RestartSec=2
Environment=NODE_ENV=production
EnvironmentFile=/srv/sip-app/.env
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now sip-app
```

Docker:

```dockerfile
# Dockerfile.app
FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm ci --omit=dev
COPY . .
USER node
CMD ["node", "app.js"]
```

```yaml
# docker-compose.yml
services:
  drachtio:
    image: drachtio/drachtio-server:latest
    network_mode: host
    volumes:
      - ./drachtio.conf.xml:/etc/drachtio.conf.xml:ro
    restart: unless-stopped

  app:
    build: .
    network_mode: host
    environment:
      - DRACHTIO_HOST=127.0.0.1
      - DRACHTIO_PORT=9022
      - DRACHTIO_SECRET=cymru
    depends_on: [drachtio]
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    restart: unless-stopped
```

A reverse proxy like nginx is typically *not* needed in front — the drachtio daemon binds directly to public SIP ports. Use a SIP-aware LB (Kamailio, OpenSIPS, or cloud SIP LB) if you scale out.

## TLS

drachtio-server terminates TLS for SIP-TLS and WSS. The Node app talks plaintext TCP to the local daemon — that's fine because they share a host.

```xml
<sip>
  <contacts>
    <contact>sips:*:5061;transport=tls</contact>
    <contact>sip:*:8443;transport=wss</contact>
    <contact>sip:*:5060;transport=udp,tcp</contact>
  </contacts>
  <tls>
    <key-file>/etc/drachtio/tls/example.com.key</key-file>
    <cert-file>/etc/drachtio/tls/example.com.crt</cert-file>
    <chain-file>/etc/drachtio/tls/fullchain.pem</chain-file>
    <dh-param>/etc/drachtio/tls/dhparam.pem</dh-param>
  </tls>
</sip>
```

Generate dhparam once:

```bash
openssl dhparam -out /etc/drachtio/tls/dhparam.pem 2048
```

For Let's Encrypt:

```bash
certbot certonly --standalone -d sip.example.com
# Copy or symlink the result into /etc/drachtio/tls/
# Reload drachtio after renewal:
echo 'systemctl reload drachtio' >> /etc/letsencrypt/renewal-hooks/post/drachtio.sh
chmod +x /etc/letsencrypt/renewal-hooks/post/drachtio.sh
```

The "TLS at the edge, plain inside" model: customer-facing connections are TLS/WSS; the control channel from Node to local drachtio is plain TCP on loopback. If you must run them on different hosts, use the `tls-port` admin attribute and a TLS-enabled `srf.connect()`.

## Performance

drachtio-server's resiprocate-based core is fast. A single drachtio handles tens of thousands of transactions per second on modern hardware (depending on CPS, dialog count, transport mix). It is single-threaded for transaction state — vertical scaling is bounded by single-core throughput.

For higher load:

- Run multiple drachtio-server instances behind a SIP LB (Kamailio dispatcher is a good choice).
- Multiple Node apps per drachtio (each connects on its own TCP control connection; daemon round-robins request handler invocations across them).
- Profile your Node app — drachtio is rarely the bottleneck; your DB queries and JSON serialization usually are.

```bash
# Daemon stats
drachtio-cli stats

# Or poll the monitoring port if enabled in conf
curl http://localhost:8080/stats
```

Node-side: use `pm2 monit`, `clinic doctor`, or `--prof` for V8 profiling under load.

```bash
# Stress-test signaling with sipp (no media)
sipp -sn uac -i 192.0.2.1 -d 5000 -s test target.example.com -r 100 -m 10000
```

## Common Errors

### "Unable to connect to drachtio-server"

```
Error: connect ECONNREFUSED 127.0.0.1:9022
```

Daemon not running, wrong port, or firewall blocking. Check `systemctl status drachtio` and `ss -tlnp | grep 9022`.

### "401 Unauthorized" coming from your app

```
SIP/2.0 401 Unauthorized
WWW-Authenticate: Digest realm="example.com", nonce="..."
```

Auth middleware ran but the Authorization header was missing or wrong. Confirm the device sent valid credentials; if expected, this is normal for the first leg of a digest exchange.

### "B2BUA failed: invalid SDP"

```
Error: B2BUA failed: invalid SDP from upstream
```

The downstream sent SDP that drachtio could not relay (rare — usually means it sent a 200 OK without a body). Inspect with `sngrep`. Check `req.body` and `uac.remote.sdp`.

### "fsmrf: ESL connection failed"

```
Error: ESL connection failed: connect ECONNREFUSED 127.0.0.1:8021
```

FreeSWITCH not running, or `event_socket.conf.xml` not configured to allow connections from your app's IP. Verify `fs_cli` works locally first.

### "tls: cert not found"

```
Failed to read certificate file: /etc/drachtio/tls/example.com.crt
```

Daemon cannot read the cert. Check path in drachtio.conf.xml, file existence, and that the drachtio user can read it (`-rw-r--r--`, owned by root or `drachtio`).

### "Out of memory" / V8 heap exhaustion

```
FATAL ERROR: Reached heap limit Allocation failed - JavaScript heap out of memory
```

Default Node heap is ~1.7 GB. Raise it for high-load apps and find the leak (probably uncleaned dialog references):

```bash
node --max-old-space-size=4096 app.js
```

Use `--inspect` and Chrome DevTools heap snapshot to find retainers.

## Common Gotchas

### 1. `srf.connect()` not awaited before handlers register

**broken:**
```js
const srf = new Srf();
srf.connect({ host: 'localhost', port: 9022, secret: 'cymru' });
srf.invite(handler);  // race: may register before connection is up
```

**fixed:**
```js
const srf = new Srf();
await srf.connect({ host: 'localhost', port: 9022, secret: 'cymru' });
srf.invite(handler);  // safely after connection
```

### 2. Stateless proxy when stateful was needed

**broken** — `res.send(302, ...)` redirect that loses Record-Route, ACK never reaches you for billing:

```js
srf.invite((req, res) => res.send(302, { headers: { Contact: '<sip:next@upstream>' } }));
```

**fixed:**
```js
srf.invite((req, res) => req.proxy({
  destination: 'sip:next@upstream',
  recordRoute: true,
  remainInDialog: true
}));
```

### 3. B2BUA without hangup propagation

**broken** — caller hangs up but the callee leg leaks:

```js
const { uas, uac } = await srf.createB2BUA(req, res, dest);
// missing destroy plumbing
```

**fixed:**
```js
const { uas, uac } = await srf.createB2BUA(req, res, dest);
uas.on('destroy', () => uac.destroy());
uac.on('destroy', () => uas.destroy());
```

### 4. Auth middleware after route middleware

**broken:**
```js
srf.register((req, res) => res.send(200));   // route registered first
srf.use('register', digestAuth(opts));        // auth never runs!
```

**fixed:**
```js
srf.use('register', digestAuth(opts));
srf.register((req, res) => res.send(200));
```

### 5. fsmrf endpoint not torn down on call end → leak

**broken:**
```js
srf.invite(async (req, res) => {
  const ep = await ms.createEndpoint();
  const uas = await srf.createUAS(req, res, { localSdp: ep.local.sdp });
  await ep.modify(uas.remote.sdp);
  await ep.play('hello.wav');
  // forgot ep.destroy()
});
```

**fixed:**
```js
srf.invite(async (req, res) => {
  const ep = await ms.createEndpoint();
  let uas;
  try {
    uas = await srf.createUAS(req, res, { localSdp: ep.local.sdp });
    await ep.modify(uas.remote.sdp);
    uas.on('destroy', () => ep.destroy());
    await ep.play('hello.wav');
  } catch (err) {
    if (uas) uas.destroy();
    await ep.destroy();
    throw err;
  }
});
```

### 6. drachtio.conf.xml secret mismatch

**broken** — daemon has `secret="cymru"`, app uses `secret: 'CYMRU'`. Connection fails with auth error.

**fixed:** match exactly. Treat the secret as a literal string, case-sensitive, no whitespace.

### 7. Multi-instance Node without sticky sessions

**broken** — load balancer round-robins SIP requests across drachtios; mid-call BYE goes to a different drachtio than the original INVITE.

**fixed** — sticky session by Call-ID hash on the LB, or use `recordRoute` with a unique Record-Route URI per drachtio so subsequent in-dialog requests are addressed to the right one.

### 8. Custom headers not properly serialized

**broken:**
```js
res.send(200, { headers: { 'X-Custom': someObject } });  // [object Object]
```

**fixed:**
```js
res.send(200, { headers: { 'X-Custom': JSON.stringify(someObject) } });
```

### 9. Memory leak via uncleaned dialog references

**broken:**
```js
const dialogs = [];
srf.invite(async (req, res) => {
  const uas = await srf.createUAS(req, res, opts);
  dialogs.push(uas);   // never removed → grows unbounded
});
```

**fixed:**
```js
const dialogs = new Map();
srf.invite(async (req, res) => {
  const uas = await srf.createUAS(req, res, opts);
  dialogs.set(uas.sip.callId, uas);
  uas.on('destroy', () => dialogs.delete(uas.sip.callId));
});
```

### 10. Forgot to handle uncaught promise rejection

**broken** — async handler that throws crashes the Node process or silently fails.

**fixed:**
```js
srf.invite(async (req, res) => {
  try {
    await doStuff();
  } catch (err) {
    req.log?.error({ err }, 'invite failed');
    if (!res.finalResponseSent) res.send(500);
  }
});

process.on('unhandledRejection', err => logger.error({ err }, 'unhandled rejection'));
```

### 11. Wrong port in drachtio.conf.xml vs srf.connect()

**broken** — daemon `<admin port="9023">`, app `srf.connect({ port: 9022 })`. ECONNREFUSED.

**fixed:** make them match, and consider configuring both from the same `.env`.

### 12. SIP-over-WS without proto-ws-listener

**broken** — your `<contact>` only declares UDP/TCP; browser WSS clients fail to connect.

**fixed:**
```xml
<contact>sips:*:8443;transport=wss</contact>
<contact>sip:*:8080;transport=ws</contact>
```

### 13. Re-using one `req.proxy` promise without await

**broken:**
```js
srf.invite((req, res) => {
  req.proxy({ destination });   // no return, no await
  console.log('done');           // misleading, proxy not actually finished
});
```

**fixed:**
```js
srf.invite(async (req, res) => {
  await req.proxy({ destination });
  console.log('proxy completed');
});
```

### 14. Treating `res.send` as final when it isn't

**broken** — sending 100 Trying then trying to send 200 OK assumes Trying was final:

```js
res.send(100);
// later:
res.send(200);  // works for INVITE but be aware Trying is not "final"
```

**fixed** — let drachtio handle 100 Trying automatically, and treat any 2xx/4xx/5xx/6xx as final. Use `res.finalResponseSent` to test.

## Diagnostic Tools

- **drachtio-cli** — daemon admin (status, sessions, log level)
- **sngrep** — terminal SIP packet capture and ladder diagrams; first-line tool for "is the SIP right?"
- **Wireshark / tshark** — full packet capture, RTP analysis, decode-as for SIP-WS
- **Node `--inspect`** — Chrome DevTools attached to your Node process for breakpoints, heap snapshots, profiles
- **pino-pretty** — readable JSON log streaming
- **clinic.js** — Node.js performance profiler (`clinic doctor`, `clinic flame`)
- **sipp** — SIP load generator and protocol-correctness tester
- **homer / sipcapture.org** — long-term SIP capture and analytics; drachtio has built-in HEP support
- **rtpengine-recording-daemon** — RTP capture for media issue diagnosis

```bash
# Quick SIP capture on the wire
sudo sngrep -d eth0 port 5060

# Decode SIP-over-WS in Wireshark
# In Edit → Preferences → Protocols → HTTP → Reassemble HTTP headers
# Then right-click any WS frame → Decode As → SIP

# Node inspector
node --inspect=0.0.0.0:9229 app.js
# Open chrome://inspect

# sipp UAC test
sipp -sn uac -d 30000 -s 1000 192.0.2.5
```

## Idioms

- **"Express middleware pattern for SIP."** If you can write Express, you can write drachtio. Don't fight the abstraction; write small middleware functions and chain them.
- **"drachtio for app, FreeSWITCH for media."** Keep signaling logic in drachtio (Node), keep media (codecs, prompts, conferences) in FreeSWITCH. Glue them with drachtio-fsmrf.
- **"Always use stateful proxy for routing"** unless you have a specific reason for stateless. You almost always want Record-Route, retransmission handling, and ACK/BYE visibility.
- **"B2BUA for mid-call control."** If you need to insert/extract media, change codecs, do hold-music, transfer logic, or per-leg billing — B2BUA, not proxy.
- **"jambonz when you want webhooks instead of code."** If your team would rather POST JSON than write JavaScript, use jambonz. If you need flexibility jambonz cannot give you, drop down to drachtio.
- **"Loopback the control channel."** Run drachtio-server and the Node app on the same host; control channel on 127.0.0.1; only the SIP ports face the network.
- **"One Srf per process."** Don't try to use multiple `Srf` instances per Node process — it works but is rarely useful.
- **"Test signaling with sipp before scaling out."** Per-instance throughput numbers from sipp guide your horizontal-scale decisions.
- **"Log Call-ID on every event."** Threading logs by Call-ID is the only way to debug call flows in production.
- **"Hangup is your friend."** The most common bug class is "leg leaked because hangup propagation was wrong." Wire `destroy` events from the start.

## See Also

- kamailio — alternative SIP proxy/registrar, faster, lower-level, scripted in own DSL
- opensips — Kamailio fork with similar capabilities and module ecosystem
- rtpengine — RTP/SRTP relay, often paired with drachtio for WebRTC and NAT
- asterisk — full PBX, alternative architecture (dialplan + AMI/ARI)
- freeswitch — media-server companion to drachtio (paired via fsmrf)
- sip-protocol — RFC 3261 SIP fundamentals (URI, transactions, dialogs, methods)

## References

- drachtio.org — project home, blog, getting-started guides
- davehorton.github.io/drachtio-srf — drachtio-srf API reference
- github.com/davehorton/drachtio-server — C++ daemon source
- github.com/davehorton/drachtio-srf — Node.js library source
- github.com/davehorton/drachtio-fsmrf — FreeSWITCH MRF source
- github.com/davehorton/drachtio-mw-digest-auth — digest auth middleware
- github.com/jambonz/jambonz-api-server — jambonz platform built on drachtio
- jambonz.org — webhook-driven voice platform documentation
- RFC 3261 — SIP base specification
- RFC 3263 — Locating SIP Servers
- RFC 7118 — WebSocket transport for SIP
- RFC 8866 — SDP (current revision)
- resiprocate.org — underlying C++ SIP stack used by drachtio-server
