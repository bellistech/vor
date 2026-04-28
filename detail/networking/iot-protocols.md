# IoT Protocols — Deep Dive

> *Constrained networks demand constrained protocols. Every byte on the wire, every milliampere drawn from a coin cell, every spreading-factor symbol traded for range — the mathematics of IoT is the mathematics of scarcity. From MQTT's 2-byte fixed header to LoRaWAN's airtime budgets to BLE's advertising intervals, this is the calculus of devices that must whisper for ten years on a battery the size of a coin.*

---

## 1. MQTT (Message Queuing Telemetry Transport)

OASIS standard. Versions in active deployment: **MQTT 3.1.1** (2014, ISO/IEC 20922:2016) and **MQTT 5.0** (2019). Pub/sub message broker over TCP (or TLS, WebSockets). Designed in 1999 at IBM for SCADA over satellite links — the original "constrained network" use case.

### 1.1 Fixed Header Anatomy

Every MQTT control packet starts with a 2-byte minimum fixed header:

```
 7 6 5 4 | 3 2 1 0     <- byte 0: control byte
+---------+--------+
| pkt type | flags |
+---------+--------+
| remaining length  |   <- byte 1..N: varint, 1-4 bytes
+-------------------+
```

Packet types (4 high bits of byte 0):

| Type | Name | Direction | Purpose |
|:---:|:---|:---:|:---|
| 1 | CONNECT | C→S | Client request to connect |
| 2 | CONNACK | S→C | Connect acknowledgment |
| 3 | PUBLISH | both | Publish message |
| 4 | PUBACK | both | QoS 1 acknowledgment |
| 5 | PUBREC | both | QoS 2 publish received |
| 6 | PUBREL | both | QoS 2 publish release |
| 7 | PUBCOMP | both | QoS 2 publish complete |
| 8 | SUBSCRIBE | C→S | Subscribe to topics |
| 9 | SUBACK | S→C | Subscribe acknowledgment |
| 10 | UNSUBSCRIBE | C→S | Unsubscribe |
| 11 | UNSUBACK | S→C | Unsubscribe ack |
| 12 | PINGREQ | C→S | Keep-alive ping |
| 13 | PINGRESP | S→C | Keep-alive pong |
| 14 | DISCONNECT | both | Graceful disconnect |
| 15 | AUTH | both | (MQTT 5.0) extended auth |

The lower 4 bits of byte 0 are flags. For PUBLISH:

```
bit 3   : DUP (duplicate flag, set on retransmission)
bits 2,1: QoS level (00, 01, 10)
bit 0   : RETAIN (broker retains last message on topic)
```

### 1.2 Remaining-Length Varint

The "remaining length" field encodes the number of bytes after the fixed header (variable header + payload). It uses a custom MQTT varint similar to Protobuf varint but big-endian-ish:

```
- 1 byte:  0   - 127             (0x00 - 0x7F)
- 2 bytes: 128 - 16,383          (high bit set on first byte)
- 3 bytes: 16,384 - 2,097,151
- 4 bytes: 2,097,152 - 268,435,455
```

**Maximum payload:** $2^{28} - 1 = 268{,}435{,}455$ bytes ≈ 256 MB. In practice brokers cap at much lower values (Mosquitto default 256 MB, AWS IoT 128 KB, HiveMQ Cloud 256 KB).

Encoding algorithm (from OASIS spec):

```c
do {
    encoded = X % 128;
    X = X / 128;
    if (X > 0) encoded |= 128;   // set continuation bit
    output(encoded);
} while (X > 0);
```

Decoding:

```c
int multiplier = 1, value = 0;
do {
    encoded = input();
    value += (encoded & 127) * multiplier;
    if (multiplier > 128*128*128) error("malformed");
    multiplier *= 128;
} while ((encoded & 128) != 0);
```

### 1.3 QoS Mathematics — Packet Counts

| QoS | Name | Packets per delivery | Broker State | Worst-case Duplicates |
|:---:|:---|:---:|:---:|:---:|
| 0 | At most once | 1 (PUBLISH) | None | 0 (delivery not guaranteed) |
| 1 | At least once | 2 (PUBLISH, PUBACK) | Sender retains until PUBACK | ∞ (until PUBACK arrives) |
| 2 | Exactly once | 4 (PUB, PUBREC, PUBREL, PUBCOMP) | Both sides hold msg-id state | 0 |

Bandwidth overhead per QoS (assuming 20-byte topic, $L$-byte payload):

$$B_0 = 4 + 20 + L$$
$$B_1 = (6 + 20 + L) + 4 = 30 + L$$
$$B_2 = (6 + 20 + L) + 4 + 4 + 4 = 38 + L$$

Overhead ratio at $L=10$: $B_0=34$, $B_1=40$, $B_2=48$. So QoS 2 costs 41% more than QoS 0 for tiny payloads. At $L=1024$: $B_0=1048$, $B_2=1062$ — overhead ~1.3%.

**Latency cost:**

$$\text{latency}_{0} = \frac{1}{2}\text{RTT}$$
$$\text{latency}_{1} = \text{RTT}$$
$$\text{latency}_{2} = 2 \times \text{RTT}$$

QoS 2 is 4× the latency of QoS 0 — devastating on a 600 ms satellite link (2.4 s end-to-end before app sees the message).

### 1.4 Retained Messages

A `PUBLISH` with the RETAIN flag set causes the broker to **store the last message on that topic**. New subscribers get the retained message immediately on subscribe. Critical for "device state" topics:

```
home/livingroom/light/state  →  retained "ON"
```

Newly-online dashboards see the bulb is on without waiting for the next state change.

**Clearing a retained message:** publish a zero-length payload with RETAIN=1. Mosquitto:

```bash
mosquitto_pub -t home/livingroom/light/state -r -n
```

### 1.5 Persistent Sessions and Clean Session

`CONNECT` flag `cleanSession`:
- **1 (clean):** broker discards subscriptions and queued messages on disconnect.
- **0 (persistent):** broker retains subscriptions, QoS 1/2 queues, and unacked PUBRELs across disconnections.

Persistent sessions are how a sleeping IoT device wakes up to find pending messages waiting. The broker keys session state on `clientId`, which **must be unique** (collisions cause the older session to be terminated).

MQTT 5.0 splits this into two fields:
- `Clean Start` (CONNECT flag) — discard prior state on this connect
- `Session Expiry Interval` (CONNECT property) — how long to retain state after this disconnect

### 1.6 Keep-Alive Math

CONNECT contains a 16-bit `Keep Alive` value in seconds (0 = disabled, max 65535 = 18.2 hours). The broker disconnects if no packet (any type, including PINGREQ) is received within $1.5 \times \text{KeepAlive}$:

$$\text{disconnect threshold} = 1.5 \times K$$

So $K=60$ means broker waits up to 90 s before declaring the client dead. Battery-powered devices want $K$ large to avoid PINGREQ traffic; lossy networks want $K$ small for fast failure detection.

**PINGREQ cost:** 2-byte fixed header, no payload. PINGRESP also 2 bytes. Total per keepalive cycle on TCP: 2 (PINGREQ) + 40 (TCP+IP headers) + 2 (PINGRESP) + 40 = ~84 bytes plus TCP ACKs. Over a year at $K=60$: $\frac{86400 \cdot 365}{60} \times 84 \approx 44$ MB just for keepalives.

### 1.7 Topic Tree Wildcards and Matching

Topics use `/`-separated levels. Two wildcards (subscribe-only):

- `+` — single-level wildcard. `home/+/temp` matches `home/kitchen/temp` and `home/bedroom/temp` but not `home/kitchen/sensor/temp`.
- `#` — multi-level wildcard, **must be at the end**. `home/#` matches everything under `home/`.

Topics starting with `$` are reserved (broker stats, e.g. `$SYS/broker/uptime`). The wildcard `#` does **not** match `$SYS/...` topics — by convention.

**Trie-based matching algorithm (broker side):**

```python
class TopicNode:
    def __init__(self):
        self.children = {}   # level_name -> TopicNode
        self.subscribers = []
        self.retained = None

def match(node, levels, depth, subs):
    if depth == len(levels):
        subs.extend(node.subscribers)
        # '#' subscribers under any node also match if at correct depth
        if '#' in node.children:
            subs.extend(node.children['#'].subscribers)
        return
    level = levels[depth]
    if level in node.children:
        match(node.children[level], levels, depth+1, subs)
    if '+' in node.children:
        match(node.children['+'], levels, depth+1, subs)
    if '#' in node.children:
        subs.extend(node.children['#'].subscribers)
```

Time complexity: $O(D \cdot W)$ where $D$ is topic depth and $W$ is the wildcard branching factor at each level — practically constant for sane topic trees.

---

## 2. CoAP (Constrained Application Protocol)

RFC 7252 (2014). HTTP-like semantics over UDP, designed for 8-bit microcontrollers with kilobytes of RAM. Uses datagram transport, optional reliability, and aggressive header compression.

### 2.1 Fixed Header (4 bytes)

```
   0                   1                   2                   3
   0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |Ver| T |  TKL  |      Code     |          Message ID           |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |   Token (if any, TKL bytes) ...
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |   Options (if any) ...
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
  |1 1 1 1 1 1 1 1|    Payload (if any) ...                       |
  +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

Fields:
- **Ver (2 bits):** version, currently 1
- **T (2 bits):** message type — 0=CON, 1=NON, 2=ACK, 3=RST
- **TKL (4 bits):** Token Length, 0–8 bytes (values 9–15 reserved)
- **Code (8 bits):** request method or response code (split as 3 bits class + 5 bits detail, e.g. `2.05` = class 2 detail 5 = "Content")
- **Message ID (16 bits):** for duplicate detection and CON/ACK matching
- **Token:** application-level request/response correlation (independent of Message ID)

**Method codes (request):** 0.01 GET, 0.02 POST, 0.03 PUT, 0.04 DELETE, 0.05 FETCH, 0.06 PATCH, 0.07 iPATCH.

**Response code classes:**
- 2.xx — Success (2.01 Created, 2.02 Deleted, 2.03 Valid, 2.04 Changed, 2.05 Content)
- 4.xx — Client error (4.00 Bad Request, 4.04 Not Found, 4.15 Unsupported Content-Format)
- 5.xx — Server error (5.00 Internal Server Error, 5.03 Service Unavailable)

### 2.2 Confirmable vs Non-Confirmable

| Type | T-bits | Reliable? | Use Case |
|:---|:---:|:---:|:---|
| Confirmable (CON) | 00 | Yes (peer ACKs) | Critical actuator commands |
| Non-confirmable (NON) | 01 | No | High-frequency sensor data |
| Acknowledgment (ACK) | 10 | n/a | Response to CON |
| Reset (RST) | 11 | n/a | "I don't understand this msg ID" |

**CON retransmission timer (RFC 7252 §4.8):**

$$ACK\_TIMEOUT \in [2, 3]\ \text{seconds (default 2)}$$
$$ACK\_RANDOM\_FACTOR = 1.5$$
$$\text{Initial timeout} = ACK\_TIMEOUT \times U[1, 1.5]$$
$$\text{Each retry: timeout} \times 2$$
$$MAX\_RETRANSMIT = 4$$

So a CON message can take up to $T_{init} + 2T + 4T + 8T + 16T = 31T$ seconds before giving up — about 45 s default.

Total transmission span:

$$T_{total}(n) = T_0 \cdot (2^{n+1} - 1)$$

For $T_0 = 2$ s, $n=4$ retries: $T_{total} = 2(2^5 - 1) = 62$ s upper bound (with random factor).

### 2.3 Options (TLV Encoding)

CoAP options are TLV-encoded as deltas from the previous option number, sorted ascending:

```
   0   1   2   3   4   5   6   7
 +---------------+---------------+
 |  Option Delta | Option Length |  1 byte
 +---------------+---------------+
 |  Option Delta extended (0-2 bytes)
 +-------------------------------+
 |  Option Length extended (0-2 bytes)
 +-------------------------------+
 |  Option Value
 +-------------------------------+
```

If delta or length is 13, one extra byte holds (value − 13). If 14, two extra bytes hold (value − 269). Value 15 reserved (0xFF marks payload).

Common options (from IANA registry):

| # | Name | Format | Use |
|:---:|:---|:---|:---|
| 1 | If-Match | opaque | Conditional |
| 3 | Uri-Host | string | Host part of URI |
| 4 | ETag | opaque | Cache validator |
| 5 | If-None-Match | empty | Conditional create |
| 7 | Uri-Port | uint | Port part of URI |
| 8 | Location-Path | string | Created resource path |
| 11 | Uri-Path | string | Each path segment is a separate option |
| 12 | Content-Format | uint | Media type (e.g. 50 = application/json) |
| 14 | Max-Age | uint | Freshness lifetime in seconds |
| 15 | Uri-Query | string | Query parameter |
| 17 | Accept | uint | Preferred response media type |
| 23 | Block2 | uint | Block-wise transfer (response) |
| 27 | Block1 | uint | Block-wise transfer (request) |
| 60 | Size1 | uint | Size of request body |

Content-Format values: 0 text/plain, 40 application/link-format, 41 application/xml, 42 application/octet-stream, 50 application/json, 60 application/cbor, 61 application/senml+json, 62 application/senml+cbor.

### 2.4 Block-Wise Transfer (RFC 7959)

UDP datagrams are limited (typical MTU 1280 bytes minus IPv6 + UDP + DTLS = ~1152 payload). Larger resources need fragmentation. Block-wise transfer chunks payloads using the `Block1` (upload) and `Block2` (download) options.

**Block option encoding (4–24 bits):**

```
NUM (4-20 bits) | M (1 bit) | SZX (3 bits)
```

- **NUM:** block number (0-indexed)
- **M:** "more blocks follow" flag
- **SZX:** size exponent. **Block size = $2^{SZX+4}$ bytes**.

| SZX | Block Size |
|:---:|:---:|
| 0 | 16 |
| 1 | 32 |
| 2 | 64 |
| 3 | 128 |
| 4 | 256 |
| 5 | 512 |
| 6 | 1024 |
| 7 | reserved |

So block sizes range from 16 to 1024 bytes, in powers of two.

**Worked example — GET 8 KB resource with SZX=6 (1024-byte blocks):**

Number of blocks: $\lceil 8192 / 1024 \rceil = 8$. Client issues 8 sequential CON GETs:

```
Req 1: GET ?Block2=0/0/6   (NUM=0, M=0, SZX=6)
Resp 1: 2.05 Content Block2=0/1/6 [1024 bytes]   (M=1: more)
Req 2: GET ?Block2=1/0/6
Resp 2: 2.05 Content Block2=1/1/6 [1024 bytes]
...
Req 8: GET ?Block2=7/0/6
Resp 8: 2.05 Content Block2=7/0/6 [1024 bytes]   (M=0: last)
```

Total round trips: 8. With 200 ms RTT: 1.6 s total transfer time.

**SZX trade-off:** larger blocks → fewer round-trips but more wasted bytes if a block is lost (full block retransmit). Optimal SZX maximizes throughput given per-packet loss rate $p$:

$$\text{Goodput}(SZX) = \frac{2^{SZX+4} (1-p)}{2^{SZX+4} + H_{CoAP} + H_{UDP+IP}}$$

Where $H_{CoAP} \approx 8$, $H_{UDP+IP} = 28$ (UDP+IPv4) or 48 (UDP+IPv6).

### 2.5 Observe Extension (RFC 7641)

CoAP extends the request/response model with a publish/subscribe pattern via the `Observe` option. Client sends GET with `Observe=0` (register), server responds with the resource and continues to send notifications when state changes.

```
Client                            Server
  |  GET /temp Observe=0  Token=A  |
  |------------------------------->|
  |  2.05 Content Observe=12 Token=A "23.5C" Max-Age=60
  |<-------------------------------|
  |  ...later, when temp changes...|
  |  2.05 Content Observe=13 Token=A "23.7C" Max-Age=60
  |<-------------------------------|
```

Notifications use the same `Token` so the client correlates them with its registration. The Observe sequence number increments with each notification (modulo $2^{24}$) so the client can detect reordering.

**Freshness via Max-Age:** server sets `Max-Age` (default 60 s). If the client doesn't see a refresh within that window, it should re-register. Mathematics:

$$P(\text{stale at time } t) = \begin{cases} 0 & t < \text{Max-Age} \\ 1 - p_{\text{notify\_arrives}} & t \geq \text{Max-Age} \end{cases}$$

### 2.6 Security: DTLS and OSCORE

**DTLS** (Datagram TLS, RFC 6347 / RFC 9147 for DTLS 1.3) wraps CoAP at the transport layer. Adds ~13-byte record header per datagram + handshake overhead (~6 round trips for full handshake, 1 RTT for resumed sessions).

DTLS handshake byte cost (PSK mode, the typical IoT case):

```
ClientHello  ~50 B
ServerHello + ServerHelloDone  ~80 B
ClientKeyExchange + ChangeCipherSpec + Finished ~60 B
ChangeCipherSpec + Finished  ~50 B
```

Total ~240 bytes plus 4 datagrams in two RTTs. PSK avoids X.509 (saves 1–4 KB of certificate bytes).

**OSCORE** (Object Security for Constrained RESTful Environments, RFC 8613) sits at the application layer. Encrypts CoAP options and payload as a COSE_Encrypt0 object inside an outer CoAP message. Properties:

- End-to-end secure across CoAP-HTTP proxies
- Replay protection via 64-bit Sender Sequence Number
- AEAD with AES-CCM-16-64-128 (default)
- Adds ~13–25 bytes overhead per message (vs DTLS's ~13)
- No handshake required; pre-shared `OSCORE Security Context`

OSCORE wins when the deployment crosses untrusted proxies; DTLS wins when a single end-to-end UDP path exists and you want the standard TLS toolchain.

### 2.7 URI Templating

CoAP URIs follow a similar shape to HTTP:

```
coap://example.com:5683/sensors/temperature?units=C
coaps://[2001:db8::1]:5684/.well-known/core
```

The `.well-known/core` endpoint returns the **CoRE Link Format** (RFC 6690) — a CoAP-native resource directory:

```
</sensors>;ct=40,
</sensors/temp>;rt="temperature";if="sensor",
</sensors/light>;rt="illuminance";if="sensor"
```

Discovery via `coap-client`:

```bash
coap-client -m get coap://[fd00::1]/.well-known/core
```

---

## 3. LwM2M (Lightweight M2M)

Open Mobile Alliance (OMA) standard for device management over CoAP/CoAPs/MQTT. Versions 1.0 (2017), 1.1 (2018), 1.2 (2020), 1.2.1 (2022). Built on top of CoAP.

### 3.1 Object Model

Resources are organized into **Objects** (typed resource collections), each Object can have multiple **Object Instances**, each instance contains **Resources**, and each Resource may have multiple **Resource Instances** (for arrays):

```
/{Object ID}/{Object Instance ID}/{Resource ID}/{Resource Instance ID}
```

Examples:
- `/3/0/0` — Object 3 (Device), instance 0, resource 0 (Manufacturer)
- `/3303/0/5700` — Temperature object, instance 0, sensor value
- `/5/0/3` — Firmware Update object, instance 0, state

### 3.2 Six Standard Management Objects

| ID | Name | Purpose |
|:---:|:---|:---|
| 0 | Security | Bootstrap & registration credentials |
| 1 | Server | Server URI, lifetime, binding |
| 2 | Access Control | Per-object ACL (DTLS PSK aware) |
| 3 | Device | Manufacturer, model, serial, battery, errors |
| 4 | Connectivity Monitoring | Network bearer, IP, signal strength |
| 5 | Firmware Update | OTA update state machine |
| 6 | Location | GPS coordinates, speed, timestamp |
| 7 | Connectivity Statistics | Counters: SMS sent, IP data sent/received |

IPSO-style smart objects (3300–3349 range) cover sensors: 3303 Temperature, 3304 Humidity, 3315 Barometer, 3340 Push Button, etc.

### 3.3 Bootstrap and Registration Flows

Four logical operations:

1. **Bootstrap** — client gets server credentials from a Bootstrap Server (Object 0 instance 0, Bootstrap-Server-URI). Mode: factory bootstrap (preconfigured), Smartcard bootstrap, or Bootstrap-Server-Initiated.
2. **Registration** — client POSTs `</rd>?ep=urn:dev:os:32473-2&lt=86400&lwm2m=1.1&b=U` to register. Server returns Location-Path header (e.g. `/rd/abc123`).
3. **Update** — client POSTs to `/rd/abc123` (refresh registration before lifetime expiry).
4. **De-registration** — client DELETEs `/rd/abc123`.

Lifetime parameter: typical 86400 s (1 day). Client must re-register at $L \times 0.9$ to avoid lapse.

### 3.4 Observation and Notification

Server can observe any resource (CoAP Observe). Pacing controls (Object 1 Server resources):

- `Default Minimum Period` — pmin, smallest interval between notifications
- `Default Maximum Period` — pmax, force a notification at least every pmax even if value unchanged
- `Notification Storing When Disabled` — buffer or drop while offline?
- `Binding` — U (UDP), T (TCP), S (SMS), Q (queue mode), or combinations like "UQ"

**Queue mode (Q):** device is sleepy, server queues commands until device wakes (registration update or "send" operation).

### 3.5 Send vs Read

LwM2M 1.1+ adds **Send** operation: device-initiated push of resources to server (POST `/dp` with composite resource list). Solves the cellular firewall problem (no server-initiated downlink possible from the public internet to a NAT'd device).

```python
# pseudo-code: Send
def send_telemetry():
    payload = senml_encode([
        ("/3303/0/5700", 23.5),
        ("/3304/0/5700", 65.2),
        ("/3/0/9",       87)   # battery percent
    ])
    coap_post("/dp", payload, content_format=110)  # senml+cbor
```

---

## 4. MQTT-SN (MQTT for Sensor Networks)

Designed for ZigBee/802.15.4 networks where TCP is impractical (no full IP stack, MTU ~127 bytes). Runs over UDP, IEEE 802.15.4 directly, or Bluetooth.

### 4.1 Differences from MQTT

| Feature | MQTT | MQTT-SN |
|:---|:---:|:---:|
| Transport | TCP | UDP / 802.15.4 / BLE |
| Topic in PUBLISH | string | 16-bit Topic ID |
| Topic registration | implicit | explicit REGISTER round-trip |
| Sleeping clients | persistent session | DISCONNECT with sleep duration + queueing |
| Multicast/Broadcast SEARCHGW | n/a | yes — gateway discovery |

### 4.2 Topic ID Compression

Topics like `home/livingroom/sensor/3/temperature` (40 bytes) are reduced to 2-byte IDs after a one-time REGISTER:

```
Client → Gateway: REGISTER MsgId=1 TopicName="home/livingroom/sensor/3/temperature"
Gateway → Client: REGACK MsgId=1 TopicId=42 Code=Accepted
```

Subsequent PUBLISHes use TopicId=42 instead of the full string. For predefined topics (well-known IDs in the 0x0001–0x00FE range), no REGISTER is needed.

**Short topic names** (2 ASCII chars, e.g. "tH") are encoded directly in the 2-byte field — no registration overhead but only 2 chars of namespace.

### 4.3 Sleeping Clients

DISCONNECT carries an optional `Duration` field. Gateway buffers QoS 1/2 messages destined for the client:

```
Client → GW: DISCONNECT Duration=3600   (sleep 1 hour)
... 1 hour passes ...
Client → GW: PINGREQ ClientId="X"        (wake-up announcement)
GW → Client: queued PUBLISH messages
GW → Client: PINGRESP
Client → GW: DISCONNECT Duration=3600    (back to sleep)
```

Sleep latency: cmd→action delay = up to one sleep cycle (e.g. 3600 s in the example).

---

## 5. LoRaWAN — Physical-Layer Mathematics

LoRa is a **chirp spread spectrum (CSS) PHY** developed by Semtech; LoRaWAN is the LoRa Alliance MAC layer atop it. Sub-GHz ISM bands: EU868 (863–870 MHz), US915 (902–928 MHz), AS923, AU915, others.

### 5.1 Spreading Factor and Data Rate

Each LoRa symbol encodes $SF$ bits, where $SF \in \{7, 8, 9, 10, 11, 12\}$. Symbol duration:

$$T_{sym} = \frac{2^{SF}}{BW}$$

Where $BW$ is the bandwidth (typically 125 kHz, sometimes 250 or 500 kHz).

| SF | $2^{SF}$ chips | $T_{sym}$ at BW=125 kHz | Sensitivity (typical) |
|:---:|:---:|:---:|:---:|
| 7 | 128 | 1.024 ms | -123 dBm |
| 8 | 256 | 2.048 ms | -126 dBm |
| 9 | 512 | 4.096 ms | -129 dBm |
| 10 | 1024 | 8.192 ms | -132 dBm |
| 11 | 2048 | 16.384 ms | -134.5 dBm |
| 12 | 4096 | 32.768 ms | -137 dBm |

Each step in SF doubles airtime and adds ~2.5–3 dB of link budget — roughly doubles the receivable range in free space, but actual gain depends on the propagation environment.

### 5.2 Effective Bit Rate

$$R_b = SF \cdot \frac{BW}{2^{SF}} \cdot CR$$

Where $CR$ is the **coding rate** (forward-error-correction overhead): 4/5, 4/6, 4/7, or 4/8. The expression $CR$ above represents the fraction (e.g. 4/5 = 0.8). Some authors write $CR$ as the denominator-minus-numerator-plus-four shorthand "CR=1" through "CR=4" — be careful which convention the data sheet uses.

Worked values at BW=125 kHz, CR=4/5:

| SF | $R_b$ (bps) |
|:---:|:---:|
| 7 | 5469 |
| 8 | 3125 |
| 9 | 1758 |
| 10 | 977 |
| 11 | 537 |
| 12 | 293 |

So SF12 is **18.7× slower** than SF7. A 50-byte payload at SF7 takes ~73 ms; at SF12 ~1500 ms.

### 5.3 Airtime Calculation (Semtech AN1200.13)

$$T_{packet} = T_{preamble} + T_{payload}$$

**Preamble:**

$$T_{preamble} = (n_{preamble} + 4.25) \cdot T_{sym}$$

Default $n_{preamble} = 8$ → 12.25 symbols.

**Payload symbols:**

$$\text{payloadSymbNb} = 8 + \max\!\left(\left\lceil \frac{8PL - 4SF + 28 + 16CRC - 20H}{4(SF - 2DE)} \right\rceil \cdot (CR + 4),\ 0\right)$$

Where:
- $PL$ = payload length in bytes
- $SF$ = spreading factor
- $H$ = 0 if header enabled, 1 if implicit
- $DE$ = 1 if low-data-rate optimization on (SF11/12 below 125 kHz), else 0
- $CRC$ = 1 if CRC enabled, else 0
- $CR$ = coding rate index (1 = 4/5, 4 = 4/8)

**Payload time:**

$$T_{payload} = \text{payloadSymbNb} \cdot T_{sym}$$

**Worked example — 50-byte payload at SF7, BW=125, CR=4/5, CRC on, header on, DE off:**

$$T_{sym} = \frac{128}{125000} = 1.024\ \text{ms}$$

Numerator: $8 \cdot 50 - 4 \cdot 7 + 28 + 16 \cdot 1 - 20 \cdot 0 = 400 - 28 + 28 + 16 = 416$

Denominator: $4(7 - 0) = 28$

Inner ceil: $\lceil 416/28 \rceil = \lceil 14.857 \rceil = 15$

PayloadSymbNb: $8 + 15 \cdot (1 + 4) = 8 + 75 = 83$

$T_{payload} = 83 \cdot 1.024 = 84.992$ ms

$T_{preamble} = 12.25 \cdot 1.024 = 12.544$ ms

$T_{packet} = 97.5$ ms.

**Same payload at SF12, BW=125, CR=4/5, DE on:**

$T_{sym} = 4096 / 125000 = 32.768$ ms

Numerator: $400 - 48 + 28 + 16 = 396$

Denominator: $4(12 - 2) = 40$

Inner ceil: $\lceil 396/40 \rceil = 10$

PayloadSymbNb: $8 + 10 \cdot 5 = 58$

$T_{payload} = 58 \cdot 32.768 = 1900.5$ ms

$T_{preamble} = 12.25 \cdot 32.768 = 401.4$ ms

$T_{packet} = 2301.9$ ms ≈ **2.3 seconds**.

So SF12 is ~24× longer airtime than SF7 for the same payload. This is **why LoRaWAN duty-cycle math matters**.

### 5.4 Duty-Cycle Limits (EU868)

The EU ETSI regulation caps transmissions per hour per channel:

| Sub-band | Duty cycle | Max airtime/hour |
|:---:|:---:|:---:|
| g (863.0–865.0) | 0.1% | 3.6 s |
| g1 (868.0–868.6) | 1% | 36 s |
| g2 (868.7–869.2) | 0.1% | 3.6 s |
| g3 (869.4–869.65) | 10% | 360 s |
| g4 (869.7–870.0) | 1% | 36 s |

At SF12 with 2.3 s airtime per packet on a 1% sub-band: max **15 packets per hour per channel**. At SF7 with 97 ms: **371 packets per hour**.

$$\text{max}\_packets/hr = \left\lfloor \frac{3600 \cdot d}{T_{packet}} \right\rfloor$$

Where $d$ is the duty cycle as a fraction.

### 5.5 Device Classes

| Class | Battery | Downlink | Use Case |
|:---:|:---:|:---|:---|
| A | days–years | Only after uplink (RX1, RX2 windows ~1, 2 s after TX) | Most sensors |
| B | months | Scheduled beacons (every 128 s default) for ping slots | Tracked assets needing predictable downlink |
| C | wall-powered | Always listening except during TX | Actuators, mains-powered |

Class A energy advantage: receiver only powered ~3 s per uplink ($T_{TX} + T_{RX1} + T_{RX2}$) vs Class C continuous listening (~30 mA × 24 h = 720 mAh/day, eats most batteries in days).

Class B beacon timing:

$$T_{beacon\_period} = 128\ \text{s}$$
$$T_{ping\_slot} = 30\ \text{ms}$$
$$\text{ping\_offset} = \text{HMAC}(devAddr, beaconTime) \mod (T_{beacon\_period}/T_{ping\_slot})$$

### 5.6 ADR — Adaptive Data Rate

Network server measures recent SNR margin from the device's last 20 uplinks; if margin > threshold, requests the device step down SF (faster, less airtime). LinkADRReq MAC command:

```
LinkADRReq:
  DataRate (4 bits)  | TXPower (4 bits) | ChMask (16 bits) | Redundancy (8 bits)
```

Margin calculation:

$$margin\_db = SNR_{measured} - SNR_{required}(SF) - install\_margin$$

Where required SNR for demodulation:

| SF | Required SNR (dB) |
|:---:|:---:|
| 7 | -7.5 |
| 8 | -10 |
| 9 | -12.5 |
| 10 | -15 |
| 11 | -17.5 |
| 12 | -20 |

Each ADR step-down reduces SF by 1 (or increments DR by 1) — until margin ≤ 0, at which point ADR holds.

---

## 6. Zigbee — IEEE 802.15.4 + Zigbee Cluster Library

Zigbee 3.0 unifies Home Automation, Light Link, etc. on a single profile. Built on **IEEE 802.15.4** (2.4 GHz, 250 kbps, 127-byte MAC frames) plus Zigbee NWK (mesh routing) and APS (application support).

### 6.1 Frame Sizing

802.15.4 PHY/MAC budget:

```
PHY preamble + SFD + PHR    : 6 bytes
MAC header (FCF, seq, addr) : 9-21 bytes
MAC payload                 : up to ~104 bytes
MAC FCS                     : 2 bytes
TOTAL                       : 127 bytes max
```

Subtract NWK header (8 bytes typical, more with security), APS header (8+ bytes), Zigbee Cluster Library header (3-5 bytes) — leaves **~70 bytes** of application payload.

### 6.2 Mesh Routing — AODV

Zigbee uses an AODV-derived routing protocol. Routers maintain route tables; on a cache miss they broadcast a Route Request (RREQ); destination or an intermediate router with a fresh route replies with Route Reply (RREP).

**Path cost metric:** sum of link cost over hops, where:

$$C_{link} = \min(7, \lceil 1 / p_{success}^4 \rceil)$$

Lower path cost wins. The exponent 4 weights link quality heavily — a 90% link costs $1/0.6561 \approx 1.5$, a 60% link costs $1/0.1296 \approx 7.7$ → clamped to 7.

### 6.3 Device Types

| Type | Function | Sleep? |
|:---|:---|:---:|
| Coordinator (ZC) | Forms network, holds trust center | No (typically) |
| Router (ZR) | Mesh forwarding, parent for end devices | No |
| End Device (ZED) | Leaf node, single parent | Yes — sleepy ZED is the battery case |

A sleepy end device (SED) periodically polls its parent for buffered data. Poll interval determines latency vs battery life:

$$E_{daily} = \frac{86400}{T_{poll}} \cdot E_{poll} + 86400 \cdot I_{sleep} \cdot V_{batt}$$

For $T_{poll} = 7.5$ s, $E_{poll} = 30\ \mu C$ per poll, $I_{sleep} = 2\ \mu A$:

Polls/day: $86400/7.5 = 11520$
Polling charge: $11520 \times 30\mu C = 345\ mC = 0.0959\ mAh$
Sleep current: $2\mu A \times 24h = 0.048\ mAh$
Total: ~0.144 mAh/day. CR2032 (220 mAh) → **4.2 years** ignoring radio TX.

### 6.4 Zigbee Cluster Library (ZCL)

Application-layer "cluster" model. A cluster is a collection of attributes and commands, identified by a 16-bit cluster ID:

| Cluster ID | Name |
|:---:|:---|
| 0x0000 | Basic |
| 0x0006 | On/Off |
| 0x0008 | Level Control |
| 0x0300 | Color Control |
| 0x0402 | Temperature Measurement |
| 0x0500 | IAS Zone (security) |

A bulb implements clusters 0x0000, 0x0006, 0x0008, 0x0300 on its Endpoint 1. A controller binds to those clusters via APS Bind requests.

### 6.5 Zigbee Green Power

Subset for **batteryless** energy-harvesting devices (e.g. piezo light switches that produce a brief power burst when pressed). Constraints:

- Single-frame transmission ("send and forget"); no MAC-level ACK
- Transmits on Channel 11 by default (sometimes channel-rotating)
- Frame counter for replay protection (random 32-bit per device, monotonically increasing)
- Frequency-agile mode: scan 11/15/20/25 in sequence

GPD frame is ~40 bytes. Energy budget per press: ~50 µJ enough to power the radio for 1 ms transmission.

---

## 7. BLE — Bluetooth Low Energy

Bluetooth 4.0+ "LE" subset. Designed for sub-mW devices.

### 7.1 GATT Hierarchy

```
Profile
  └── Service (e.g. Heart Rate Service, UUID 0x180D)
       ├── Characteristic (Heart Rate Measurement, 0x2A37)
       │    ├── Value
       │    └── Descriptor (Client Characteristic Configuration, 0x2902)
       └── Characteristic (Body Sensor Location, 0x2A38)
```

Each characteristic has:
- 16-bit or 128-bit UUID
- Properties (Read, Write, WriteWithoutResponse, Notify, Indicate, etc.)
- Value (up to 512 bytes per attribute, ATT MTU caps actual transfer)
- Descriptors (CCCD enables/disables notifications)

### 7.2 PHY Variants and Throughput

BLE PHY layers (since BT 5.0):

| PHY | Symbol Rate | Coding | Net Throughput | Range Multiplier |
|:---:|:---:|:---:|:---:|:---:|
| LE 1M | 1 Msym/s | 1:1 | ~800 kbps | 1× |
| LE 2M | 2 Msym/s | 1:1 | ~1400 kbps | ~0.8× |
| LE Coded S=2 | 1 Msym/s | 2:1 (FEC) | ~500 kbps | ~2× |
| LE Coded S=8 | 1 Msym/s | 8:1 (FEC) | ~125 kbps | ~4× |

LE 2M doubles raw bandwidth at the cost of ~1 dB SNR (range slightly reduced). LE Coded trades throughput for range — S=8 gives ~4× the link distance of LE 1M for the same TX power.

Connection event throughput (theoretical):

$$T_{conn\_event}\_max = \frac{T_{event\_window} - T_{IFS}}{T_{packet}}$$

With LE 2M, max 251-byte payload, ~1.5 ms event windows: ~10–12 packets per event → ~3 Mbps theoretical, ~1.4 Mbps practical.

### 7.3 Advertising

Advertiser broadcasts on three channels (37, 38, 39) at the **advertising interval** $T_{adv}$. Default 1.28 s, range 20 ms – 10.24 s. Each advertisement is up to 31 bytes (legacy adv) or 254 bytes (extended adv, BT 5.0).

Energy per advertisement event:

$$E_{adv\_event} = 3 \times (E_{tx\_pkt} + E_{rx\_window})$$

Roughly 100–300 µJ per event for typical SoCs. Daily:

$$E_{adv\_daily} = \frac{86400}{T_{adv}} \cdot E_{adv\_event}$$

At $T_{adv}=1.28$ s, $E_{adv\_event}=200\ \mu J$:

$E_{adv\_daily} = 67500 \times 200\mu J = 13.5\ J/day \approx 1.04\ mAh/day$ at 3.6 V.

CR2032 (220 mAh) → ~7 months of advertising-only life.

Increasing $T_{adv}$ to 5 s extends life to ~2.7 years.

### 7.4 Connection Mode

Once a Central connects, the Peripheral responds during **connection events** at the **connection interval** $T_{conn}$ (7.5 ms – 4 s). Slave Latency parameter lets the Peripheral skip up to N consecutive events to save power, with the cost of higher Central-to-Peripheral latency.

Effective average duty cycle:

$$D = \frac{T_{event}}{(1 + L_{slave}) \cdot T_{conn}}$$

For $T_{event} = 1$ ms, $T_{conn} = 100$ ms, $L_{slave} = 4$: $D = 1/500 = 0.2\%$ — very low power.

---

## 8. Thread / Matter

Thread (Thread Group, 2014) is a low-power, IPv6-based mesh protocol over IEEE 802.15.4. Matter (CSA, 2022) is the application-layer standard layered atop Thread or Wi-Fi.

### 8.1 Thread Stack

```
+---------------------------------------+
|  Application (Matter, custom UDP/COAP) |
+---------------------------------------+
|  UDP                                   |
+---------------------------------------+
|  IPv6 + 6LoWPAN (RFC 6282 compression)|
+---------------------------------------+
|  IEEE 802.15.4 MAC + PHY              |
+---------------------------------------+
```

**6LoWPAN compression:** the full 40-byte IPv6 header compresses to ~2–10 bytes when source/dest are link-local or context-known. Stateless compression uses elision flags (ECN, traffic class, flow label, hop limit) to drop redundant bytes.

### 8.2 Mesh Roles

| Role | Function |
|:---|:---|
| Leader | Single node that distributes network configuration (Mesh Local Prefix, mesh ID) |
| Router | Forwards mesh traffic; up to 32 routers per partition |
| REED (Router-Eligible End Device) | Acts as End Device but can promote to Router |
| Sleepy End Device (SED) | Polls parent every $T_{poll}$ seconds |
| Minimal End Device (MED) | RX-on, single parent |

Leader election via the **Mesh Routing Cost Algorithm** uses RIP-style distance vectors with link quality factored in.

### 8.3 Matter Operational Dataset

Matter exposes the same data model regardless of underlying transport (Thread or Wi-Fi). Operational dataset includes:

- Active Operational Dataset (Thread credentials, channel, network key, PAN ID, extended PAN ID, Mesh Local prefix)
- Pending Operational Dataset (for atomic credentials swap)
- Commissioning credentials (PASE — Password-Authenticated Session Establishment using SPAKE2+)

Commissioning flow:

```
1. Commissionee enters BLE advertising mode showing 11-digit setup code
2. Commissioner connects over BLE
3. PASE handshake using setup code (SPAKE2+) → secure channel
4. Commissioner provisions Thread/Wi-Fi credentials over secure channel
5. Commissionee joins operational network
6. CASE (Certificate Authenticated Session Establishment) for ongoing comms
```

PASE replaces brittle WPS-style approaches; SPAKE2+ is a PAKE — even if attacker observes the handshake, they cannot brute-force the setup code offline.

### 8.4 Matter Cluster Model

Borrows ZCL's cluster idea but with strict typed data model. Cluster IDs:

| ID | Name |
|:---:|:---|
| 0x0006 | OnOff (same as Zigbee) |
| 0x0008 | Level Control |
| 0x0028 | Basic Information |
| 0x0035 | Network Commissioning |
| 0x0040 | Fixed Label |
| 0x0050 | OTA Software Update Provider |
| 0x0301 | Color Control |

---

## 9. OPC UA (IEC 62541)

OPC Foundation industrial automation protocol. Replaces the COM/DCOM-based "Classic OPC". Used on factory floors, SCADA, MES.

### 9.1 Architecture

Two transport bindings:

- **OPC UA Binary (TCP):** opc.tcp://, port 4840 default. UA-TCP framing with chunking.
- **OPC UA HTTPS (SOAP):** opc.https://, mostly legacy.

PubSub model (OPC UA 1.04+):

- **DataSetWriter** publishes a typed message stream to a topic (UDP multicast, MQTT, AMQP)
- **DataSetReader** subscribes and decodes
- Wire formats: UADP (binary, multicast-friendly) or JSON (broker-friendly via MQTT)

### 9.2 Information Model

Nodes form a typed object graph:

```
Object  (FolderType, BaseObjectType, FolderType)
  Variable (DataType, AccessLevel, MinimumSamplingInterval)
  Method  (InputArguments, OutputArguments)
  ReferenceType (HasComponent, HasProperty, Organizes)
  ObjectType, VariableType (templates)
  DataType (built-in or structured)
```

NodeIDs are 4-tuples: `(NamespaceIndex, IdentifierType, Identifier, ServerIndex)`. Common namespace 0 holds the OPC UA standard types (e.g. NodeId `i=85` is "Objects" folder).

### 9.3 Secure Channel

```
OpenSecureChannel:
  - Asymmetric handshake (RSA-2048+ or ECC P-256+)
  - Negotiates symmetric keys (AES-256 or AES-128)
  - Channel renewed every TokenLifetime (default 1 hour)
  - X.509 certificate authentication with optional GDS (Global Discovery Server)

CreateSession + ActivateSession:
  - User authentication: anonymous, username/password, X.509, JWT
  - Session token tied to the secure channel
```

### 9.4 Subscriptions and MonitoredItems

Client creates a **Subscription** with a publishing interval (e.g. 1000 ms). Within that subscription, **MonitoredItems** track variables with sampling intervals (e.g. 250 ms — the server samples 4× per publish cycle).

```
PublishingInterval = 1000 ms
  MonitoredItem A: SamplingInterval=250ms, QueueSize=4, Filter=DeadbandPercent(2.0)
  MonitoredItem B: SamplingInterval=1000ms, QueueSize=1
```

PublishResponse carries up to QueueSize accumulated changes per item per cycle.

---

## 10. Modbus

Modicon (now Schneider Electric) 1979. Master/slave (now "client/server" since v1.1b3) over RS-485 (RTU), RS-232 (ASCII), or TCP.

### 10.1 Modbus RTU vs TCP Framing

**RTU (serial):**

```
+-----+----+-----+-----+
| Adr |Func| Data|CRC16|
+-----+----+-----+-----+
  1B   1B  0-252B  2B
```

3.5 character times of silence frames each transaction. CRC-16-IBM polynomial 0xA001.

**TCP:**

```
+----------------+----+-----+----+
| MBAP Header(7) | Adr|Func |Data|
+----------------+----+-----+----+
```

MBAP = Modbus Application Protocol header: TransactionID (2B), ProtocolID=0 (2B), Length (2B), UnitID (1B). No CRC (TCP handles integrity). Default port 502.

### 10.2 Address Spaces (Reference Model)

Four legacy data tables, each with 16-bit addresses (0–65535):

| Table | Type | Size | Function Codes |
|:---|:---|:---:|:---:|
| Coils | 1-bit, R/W | 65536 | 1 (read), 5 (write single), 15 (write multi) |
| Discrete Inputs | 1-bit, R | 65536 | 2 (read) |
| Input Registers | 16-bit, R | 65536 | 4 (read) |
| Holding Registers | 16-bit, R/W | 65536 | 3 (read), 6 (write single), 16 (write multi) |

PLC convention prefixes: 0xxxx coils, 1xxxx discrete inputs, 3xxxx input registers, 4xxxx holding registers (1-based, NOT identical to Modbus wire 0-based addresses — common gotcha).

### 10.3 Function Codes

| FC | Name | Direction |
|:---:|:---|:---|
| 1 | Read Coils | Master→Slave |
| 2 | Read Discrete Inputs | M→S |
| 3 | Read Holding Registers | M→S |
| 4 | Read Input Registers | M→S |
| 5 | Write Single Coil | M→S |
| 6 | Write Single Register | M→S |
| 7 | Read Exception Status | M→S |
| 8 | Diagnostics | M→S |
| 11 | Get Comm Event Counter | M→S |
| 15 | Write Multiple Coils | M→S |
| 16 | Write Multiple Registers | M→S |
| 17 | Report Slave ID | M→S |
| 22 | Mask Write Register | M→S |
| 23 | Read/Write Multiple Registers | M→S |
| 43 | Encapsulated Interface (Device ID) | M→S |
| 0x80+FC | Exception response | S→M |

Exception codes: 1 Illegal Function, 2 Illegal Data Address, 3 Illegal Data Value, 4 Slave Device Failure, 5 Acknowledge, 6 Slave Device Busy, 8 Memory Parity Error.

### 10.4 Read Holding Registers (FC 3) Walk-through

Request:
```
[UnitID][03][StartHi][StartLo][CountHi][CountLo]
   01    03   00 6B    00 03      => read 3 registers starting at 0x006B
```

Response:
```
[UnitID][03][ByteCount][Data...]
   01    03      06     02 2B 00 00 00 64
```

ByteCount = 6 (3 registers × 2 bytes). Values: 0x022B, 0x0000, 0x0064 → 555, 0, 100.

### 10.5 The 32-Bit Float Problem

Modbus registers are 16-bit. Floating-point values (32-bit IEEE 754) span two registers. Word order is **not** standardized:

| Variant | Order |
|:---|:---|
| ABCD | Big-endian, MSW first (most common) |
| CDAB | Little-endian word-swap (Schneider variants) |
| BADC | Byte-swapped within each word |
| DCBA | Full reverse |

This is the classic interop pain. Always check vendor docs and test with a known value (e.g. write 1.0 = 0x3F800000 and inspect).

### 10.6 Modbus TCP Quirks

- No transaction security at all. **Treat as plaintext on a trusted segregated network.** Modbus/TCP Security (Modbus Organization spec, 2018) wraps in TLS but adoption is sparse.
- TCP handles framing, but slaves still echo the request format — no streaming.
- A single TCP connection serializes requests; deep pipelines benefit from multiple connections to the same slave (if it supports them).

---

## 11. Power Consumption Mathematics

The fundamental IoT trade-off: **range vs. throughput vs. battery life**. The energy budget of a 3 V coin cell (CR2032, ~220 mAh = 792 C of charge, ~2.4 kJ at 3 V) caps everything.

### 11.1 LoRa SF7 vs SF12 Energy

For the same 50-byte payload (Section 5.3 worked examples):

- SF7: 97.5 ms airtime
- SF12: 2301.9 ms airtime → **23.6× more time on air**

Energy per transmit (assuming TX current $I_{TX} = 30$ mA at 3.3 V):

$$E_{TX} = V \cdot I_{TX} \cdot T_{packet}$$

| SF | $T_{packet}$ | $E_{TX}$ |
|:---:|:---:|:---:|
| 7 | 97.5 ms | 9.65 mJ |
| 12 | 2301.9 ms | 227.9 mJ |

So SF12 costs **~24× more energy per packet**. Combined with the fact that SF12's lower data rate forces longer airtime, the duty-cycle limits also shrink the maximum reporting frequency.

The headline number is "5× more energy at SF12 for the same payload" if you compare typical ADR-managed deployments where most devices land at SF7-SF9; the worst-case 24× shows up only when a device permanently sits at SF12 due to range.

Battery life for a "send 50 B every 15 minutes" use case:

$$N_{packets/year} = \frac{31536000}{900} = 35040$$

At SF7: $E_{year} = 35040 \times 9.65\ \text{mJ} = 338\ \text{J}$
At SF12: $E_{year} = 35040 \times 227.9\ \text{mJ} = 7986\ \text{J}$

A 2400 J coin cell lasts ~7 years at SF7, **~3.6 months at SF12** (ignoring sleep/RX overhead).

### 11.2 BLE Advertising Energy

Default $T_{adv}=1.28$ s, advertising payload 31 bytes legacy. Each adv event transmits on 3 channels with brief RX windows (for scan requests) between.

Approximate energy per event for nRF52-class SoC:

$$E_{adv\_event} \approx 3 \cdot (E_{TX,pkt} + E_{RX,window}) \approx 3 \cdot (60\ \mu J + 30\ \mu J) = 270\ \mu J$$

Daily total at $T_{adv}=1.28$ s:

$$N = 86400 / 1.28 = 67500$$
$$E_{day} = 67500 \times 270\ \mu J = 18.2\ J$$

CR2032 (792 C × 3 V = 2376 J usable): 2376 / 18.2 = **131 days** of pure advertising.

Stretching $T_{adv}$ to 5 s extends to ~1.4 years. iBeacon-style deployments often use $T_{adv}=100$ ms (much higher discoverability) — battery life ~10 days from a CR2032 (hence iBeacons usually use AA batteries).

### 11.3 Zigbee Sleepy End Device (SED)

Polling parent every $T_{poll}$ for buffered downlink. Latency is bounded by $T_{poll}$ (worst case wait), and energy:

$$E_{daily,SED} = N_{poll} \cdot E_{poll} + I_{sleep} V T_{day}$$

With $E_{poll} \approx 100\ \mu J$ and $I_{sleep} = 1\ \mu A$ at 3 V:

| $T_{poll}$ | Polls/day | Polling E | Sleep E | Total |
|:---:|:---:|:---:|:---:|:---:|
| 1 s | 86400 | 8.64 J | 0.26 J | 8.9 J |
| 7.5 s | 11520 | 1.15 J | 0.26 J | 1.41 J |
| 30 s | 2880 | 0.29 J | 0.26 J | 0.55 J |
| 300 s | 288 | 0.029 J | 0.26 J | 0.29 J |

CR2032 life:
- 1 s polling: 268 days
- 7.5 s polling: 4.6 years
- 30 s polling: 12 years (limited by self-discharge first)

Trade-off: 7.5 s poll means worst-case 7.5 s downlink command latency (e.g. "turn off light").

### 11.4 General Energy-per-Bit

Across protocols, energy per useful bit ($E_b$):

$$E_b = \frac{V \cdot I_{TX} \cdot T_{packet}}{8 \cdot L_{payload}}$$

Order of magnitude (typical, for small payloads):

| Tech | $E_b$ |
|:---:|:---:|
| BLE LE 2M | ~0.1 µJ/bit |
| Zigbee | ~0.5 µJ/bit |
| LoRa SF7 | ~25 µJ/bit |
| LoRa SF12 | ~600 µJ/bit |
| NB-IoT | ~5 µJ/bit |
| LTE-M | ~1 µJ/bit |
| Wi-Fi (HaLow 802.11ah) | ~0.5 µJ/bit |

LoRa is **5000× more expensive per bit than Bluetooth**. The justification: LoRa reaches kilometers, BLE reaches meters.

---

## 12. Protocol Comparison Matrix

| Protocol | Throughput | Range | Latency | Topology | Power | Security | Best For |
|:---|:---:|:---:|:---:|:---:|:---:|:---:|:---|
| MQTT (TCP) | Link-bound (kbps–Mbps) | LAN/WAN | 10 ms–1 s | Pub/Sub broker | Medium | TLS | Cloud telemetry |
| MQTT-SN | 250 kbps | 100 m mesh | 100 ms–10 s | Pub/Sub gateway | Low | DTLS | Sub-MQTT bridges |
| CoAP | UDP-bound | LAN/WAN | 50–500 ms | REST | Medium-low | DTLS / OSCORE | Resource-style M2M |
| LwM2M | UDP-bound | LAN/WAN | 100 ms–1 s | Device mgmt | Low | DTLS | Cellular IoT mgmt |
| LoRaWAN | 0.3–50 kbps | 2–15 km | seconds | Star-of-stars | Very low | AES-128 | Long-range telemetry |
| Zigbee | 250 kbps shared | 30 m hop, mesh | 50–500 ms | Mesh | Low | AES-128-CCM* | Home automation |
| BLE | 0.1–2 Mbps | 10–100 m | 7.5 ms–4 s | PAN | Very low | AES-CCM | Wearables, beacons |
| Thread | 250 kbps | 30 m hop, mesh | 50–500 ms | Mesh IPv6 | Low | DTLS, J-PAKE | Matter ecosystem |
| Matter | inherits Thread/Wi-Fi | inherits | 100 ms–1 s | App on Thread/IP | inherits | DTLS+CASE | Cross-vendor smart home |
| OPC UA | TCP/UDP-bound | LAN | ms | Client/Server, PubSub | n/a | TLS+X.509 | Industrial |
| Modbus | 19.2 kbps RTU / TCP | RS-485 1.2 km | ms–100 ms | Master/Slave | n/a | none / TLS | Legacy industrial |

### 12.1 Selection Decision Tree

Start from the constraint that hurts most:

```
"I need X kB/s at Y range with Z battery life — which protocol?"

If battery life > 1 year on coin cell:
  If range > 1 km → LoRaWAN (kB/day budget)
  If range < 100 m → BLE advertising or Zigbee SED
  If always-on power available → reconsider; use Wi-Fi/Thread
If throughput > 100 kbps:
  If LAN reach OK → BLE LE 2M, Zigbee/Thread, Wi-Fi
  If WAN reach needed → cellular (LTE-M, NB-IoT) + MQTT/CoAP
If multi-vendor smart home → Matter (over Thread or Wi-Fi)
If industrial floor:
  If brownfield/PLC → Modbus
  If new install → OPC UA
If asset tracking 1+ km outdoor → LoRaWAN GPS payload
```

**Cost dimensions to also weigh:**

- Spectrum licensing (cellular: pay per MB; LoRa, BLE, Zigbee: free ISM)
- Gateway cost (cellular: built-in; LoRa: $500–$5000 gateway every 2–10 km; Zigbee: hub per home)
- Provisioning friction (Matter: QR code; LoRa: device EUIs + AppKey; Modbus: manual config)
- Regulatory: duty cycle in EU (LoRa 1%), FCC dwell time in US, channel planning indoor vs outdoor

---

## 13. Worked Examples

### 13.1 MQTT QoS 2 Four-Way Handshake — Byte-Level

Topic `home/livingroom/temp` (21 chars), payload `23.5` (4 bytes), packet ID 0x0042.

**PUBLISH (sent):**

```
30 1F                  fixed header: type 3 (PUBLISH), flags QoS=2 (0x04 → 30+04=34 actually for QoS+RETAIN bits, but...)
                       NOTE: byte 0 = (3 << 4) | (DUP<<3) | (QoS<<1) | RETAIN
                       For QoS 2 first attempt: 0x34
34 1F                  0x34 = type 3 (PUBLISH), QoS 2, no RETAIN, no DUP
                       0x1F = remaining length 31 bytes
00 15                  topic length 21
68 6F 6D 65 2F 6C 69 76 69 6E 67 72 6F 6F 6D 2F 74 65 6D 70   "home/livingroom/temp"
                       Wait — that's only 20 chars; let me recount: h-o-m-e-/-l-i-v-i-n-g-r-o-o-m-/-t-e-m-p = 20.
                       Use 0x14 = 20 then.
00 14                  topic length 20
[20 bytes ASCII]       topic
00 42                  packet id 0x0042
32 33 2E 35            payload "23.5"
```

**PUBREC (received from broker):**

```
50 02      fixed header: type 5 (PUBREC), flags=0, remaining length 2
00 42      packet id 0x0042
```

**PUBREL (sent to broker):**

```
62 02      fixed header: type 6 (PUBREL), flags=0x2 (mandatory per spec), len 2
00 42      packet id 0x0042
```

**PUBCOMP (received from broker):**

```
70 02      type 7 (PUBCOMP), len 2
00 42      packet id 0x0042
```

Total bytes on wire (one direction each):
- PUBLISH: 33 bytes
- PUBREC: 4 bytes
- PUBREL: 4 bytes
- PUBCOMP: 4 bytes

**Sum: 45 bytes** for delivering a 4-byte payload exactly once. Compare QoS 0: `30 18 00 14 [20 byte topic] 32 33 2E 35` = 26 bytes. **QoS 2 is 73% more bytes** for this small payload.

### 13.2 CoAP Block-Wise GET of 8 KB Resource (SZX=6, 1024-byte blocks)

Device fetches `coap://server/firmware.bin` (8192 bytes) in 1024-byte chunks.

```
Token=0xAB12, Message IDs incrementing.

C → S: CON GET /firmware.bin Block2=(0/0/6) MID=1
S → C: ACK 2.05 Content Block2=(0/1/6) [1024 bytes] MID=1   (M=1: more)

C → S: CON GET /firmware.bin Block2=(1/0/6) MID=2
S → C: ACK 2.05 Content Block2=(1/1/6) [1024 bytes] MID=2

C → S: CON GET /firmware.bin Block2=(2/0/6) MID=3
S → C: ACK 2.05 Content Block2=(2/1/6) [1024 bytes] MID=3

... blocks 3..6 ...

C → S: CON GET /firmware.bin Block2=(7/0/6) MID=8
S → C: ACK 2.05 Content Block2=(7/0/6) [1024 bytes] MID=8   (M=0: last)
```

8 round trips. With 200 ms RTT: 1.6 s minimum transfer time. Total bytes (response side, ignoring CoAP option overhead):

$$8 \times (1024 + 12) = 8288\ \text{bytes}$$

Versus a single HTTP GET over TCP: 8192 + ~150 (HTTP headers) = 8342 bytes. CoAP wins on overhead per byte but costs more in round-trip count.

If lossy network (5% per-block loss), each lost block triggers CoAP CON retransmission. Expected blocks transferred:

$$E[\text{transmissions}] = 8 / (1 - 0.05) = 8.42$$

Time penalty: ~5% retransmits × 2 s timeout = 800 ms extra worst case.

### 13.3 LoRaWAN SF7 vs SF12 Airtime — 50-Byte Payload

From Section 5.3:

| SF | $T_{sym}$ | $T_{packet}$ | Packets/hr at 1% duty |
|:---:|:---:|:---:|:---:|
| 7 | 1.024 ms | 97.5 ms | $\lfloor 36/0.0975 \rfloor = 369$ |
| 12 | 32.768 ms | 2301.9 ms | $\lfloor 36/2.302 \rfloor = 15$ |

So at SF12 a device can send at most 15 messages per hour per channel due to ETSI duty cycle. With 3 default EU868 sub-band-g1 channels: 45 messages/hour. At SF7: 1100 messages/hour.

### 13.4 BLE GATT Read Latency Budget

Central reads a 50-byte characteristic via ATT Read on connection interval $T_{conn}=100$ ms.

```
T+0 ms     : Central queues Read at app
T+0–100 ms : Wait for next connection event (uniform, avg 50 ms)
T+102 ms   : Connection event: Central transmits ATT Read Request (~12 B)
T+102.3 ms : T_IFS gap (150 µs)
T+102.4 ms : Peripheral transmits ATT Read Response (~57 B, Read Long if >MTU)
T+102.7 ms : Done
```

End-to-end average: **~52 ms**. Worst case: 102.7 ms. Drop $T_{conn}$ to 7.5 ms → average ~5.7 ms but battery cost rises ~13×.

If the payload exceeds ATT MTU (default 23 bytes; 244 with Data Length Extension), GATT uses Read Long with multiple round trips, multiplying latency by $\lceil L / (MTU - 1) \rceil$.

### 13.5 LwM2M Registration → Observe → Notify Timing

Cellular IoT device with NB-IoT Cat NB1 (typical 50 kbps uplink, 200 kbps downlink, ~1 s RTT due to PSM and eDRX).

```
T+0    : Device exits PSM (Power Saving Mode), attaches to cellular: ~1–5 s
T+5 s  : Bootstrap flow (CoAP CON to bootstrap server): 2 round trips ≈ 2 s
T+7 s  : Register POST /rd → 201 Created Location: /rd/abc123 ≈ 1 s
T+8 s  : Server sends Observe on /3303/0/5700 (temp). Server-initiated, 1 round trip ≈ 1 s
T+9 s  : Initial notification (current temp) ≈ 200 ms
... time passes; Send mode triggers on threshold ...
T+3600 : Device wakes, posts Send /dp [SenML payload] ≈ 800 ms
T+3601 : ACK from server, device returns to PSM
```

Total active radio time per hour: ~5 seconds. NB-IoT device current: 200 mA TX, 50 mA RX, 5 µA PSM. Daily energy:

$$E_{day} = 24 \times (5 \times 3.7 \times 0.2) + 86395 \times 3.7 \times 5\mu = 88.8 + 1.6 = 90.4\ \text{J}$$

A 2 Ah AA-class battery (26 kJ) lasts ~290 days.

---

## 14. Security Threat Models

### 14.1 Replay Attacks

**Threat:** attacker captures a legitimate "open garage door" command and replays it later.

**Mitigations:**

| Protocol | Replay defense |
|:---|:---|
| MQTT 5.0 + TLS | TLS sequence number prevents replay within session; full reconnect requires new ClientId or session resumption |
| CoAP DTLS | DTLS sequence number window (default 64-packet bitmap) |
| CoAP OSCORE | 64-bit Sender Sequence Number + 32-bit nonce; receiver tracks replay window |
| LoRaWAN | 16-bit FCntUp (uplink frame counter) and FCntDown; counters monotonically increase, replays detected |
| Zigbee | 32-bit frame counter per device; rolling key rotation |
| Matter | CASE session secrets bound to TI/RI counters; attestation prevents impersonation |

LoRaWAN's 16-bit FCntUp is a known weakness for chatty devices. After 65,536 uplinks the counter wraps; without FCnt32 (LoRaWAN 1.1) the device must rejoin (security context refresh) before the wrap. LoRaWAN 1.1 expanded FCnt to 32 bits and made the counter handling stricter.

### 14.2 DoS via Duty-Cycle Exhaustion (LoRaWAN)

**Threat:** attacker forces a device to transmit until its 1% duty-cycle budget is consumed, jamming legitimate traffic for the rest of the hour.

**Vectors:**
- Confirmed downlink bombs — every confirmed downlink requires an uplink ACK
- Bogus MAC commands triggering re-bootstrap
- Forced ADR adjustments that push to higher SF (more airtime per packet)

**Mitigations:**
- Network server rate-limits per device
- Cap confirmed downlinks per hour (LoRa Alliance guidance: ≤10/day per device)
- Refuse MAC commands from unauthenticated sources
- Monitor airtime budgets and alert on outliers

### 14.3 MITM on PSK-Only Deployments

**Threat:** identical pre-shared key (PSK) baked into firmware across thousands of devices; one device extraction compromises all.

**Mitigations:**
- Per-device unique keys (device-unique AppKey in LoRaWAN, per-device PSK in DTLS)
- Use certificate-based DTLS where compute allows (Cortex-M0+ struggles; M3+ is fine)
- Key derivation from a hardware secure element (ATECC608, NXP A71CH) so the key never leaves the chip
- OTP fuses for boot keys; secure boot to prevent firmware extraction

LoRaWAN 1.1 introduces split keys: AppSKey (app-layer encryption) is held by the application server, NwkSKey (MAC-layer) by the network server. Compromising one doesn't compromise the other.

### 14.4 Rogue-Device Attacks via Misconfigured Pairing

**Threat:** open Zigbee Touchlink commissioning lets any attacker inject themselves into a Zigbee network during the 5-second commissioning window. Reported CVE-2020-6007 (Hue bulb hijack).

**Mitigations:**
- Disable Touchlink permanently (Zigbee 3.0 lets you turn it off)
- Use install codes (16-byte device-unique secret printed on the bulb's QR code) for commissioning
- Network-wide "permit join" set to false outside short admin windows
- For Matter: setup PIN + DAC (Device Attestation Certificate) chain rooted at a CSA-trusted CA

### 14.5 Physical-Layer Jamming

**Threat:** attacker transmits on the same channel to drown legitimate signals.

**Mitigations:**
- LoRa CSS coding gives ~10–15 dB processing gain — harder to jam than narrowband
- Frequency hopping (LoRaWAN allocates 8 default channels in EU868; devices rotate)
- BLE adaptive frequency hopping (1600 hops/sec across 37 data channels, dynamically blacklists noisy channels)
- Zigbee channel scan + auto-rejoin on different channel
- 802.15.4 CCA (clear channel assessment) avoids in-band collisions but doesn't help against malicious jammer

### 14.6 Firmware Update Attack Surface

**Threat:** attacker pushes hostile firmware via OTA update channel.

**Mitigations:**
- Image signed with manufacturer key; bootloader verifies signature before reflashing
- Anti-rollback counters (eFUSE) prevent downgrade to known-vulnerable firmware
- Encrypted image (AES-CTR) so an attacker who captures the OTA can't reverse the update
- Authenticated download endpoint (LwM2M Object 5 supports HTTPS/CoAPS firmware sources)
- Atomic A/B partitioning so a corrupt update doesn't brick the device

### 14.7 Side-Channel and Physical Attacks

**Threat:** attacker physically accesses device (returned, stolen, parked car), reads keys via JTAG/SWD, side-channel timing, or fault injection.

**Mitigations:**
- Disable debug interfaces in production (lock SWD via fuses)
- Secure element (ATECC608A, Microchip CryptoAuthLib, NXP SE050) holds keys in tamper-resistant silicon
- Constant-time crypto implementations (mbedTLS, tinycrypt with CT primitives)
- Anti-tamper case + accelerometer triggers key wipe on movement (high-security only)
- Glitch-resistant boot sequences (double-check, redundant comparisons)

---

## 15. Operational Snippets

### 15.1 Mosquitto Quick Reference

```bash
# subscribe with QoS 1 to a wildcard topic
mosquitto_sub -h broker.example.com -p 8883 \
  --cafile ca.crt --cert client.crt --key client.key \
  -t 'sensors/+/temp' -q 1 -v

# publish a retained state message
mosquitto_pub -h broker -t home/light/livingroom/state -m ON -r -q 1

# clear a retained message
mosquitto_pub -h broker -t home/light/livingroom/state -n -r

# set keep-alive 30 s and clean session false
mosquitto_sub -h broker -i myclient -k 30 -c -t test/#
```

### 15.2 coap-client (libcoap)

```bash
# basic GET
coap-client -m get coap://[fd00::1]/sensors/temp

# discovery
coap-client -m get coap://[fd00::1]/.well-known/core

# observe with 60 s timeout
coap-client -m get -s 60 coap://[fd00::1]/sensors/temp

# block-wise GET, force SZX=6 (1024 byte)
coap-client -m get -b 1024 coap://[fd00::1]/firmware.bin -o out.bin

# DTLS PSK
coap-client -m get -k secret -u client01 coaps://[fd00::1]/sensors/temp
```

### 15.3 LoRaWAN Airtime Calculator (Python)

```python
import math

def lora_airtime(payload_bytes, sf, bw=125000, cr=1, header=True, crc=True, low_dr_opt=None):
    """
    cr: 1 = 4/5, 2 = 4/6, 3 = 4/7, 4 = 4/8
    Returns airtime in seconds.
    """
    if low_dr_opt is None:
        low_dr_opt = (sf >= 11 and bw == 125000)
    de = 1 if low_dr_opt else 0
    h = 0 if header else 1
    crc_bit = 1 if crc else 0

    t_sym = (2 ** sf) / bw
    n_preamble = 8
    t_preamble = (n_preamble + 4.25) * t_sym

    numerator = 8 * payload_bytes - 4 * sf + 28 + 16 * crc_bit - 20 * h
    denom = 4 * (sf - 2 * de)
    payload_sym_nb = 8 + max(math.ceil(numerator / denom) * (cr + 4), 0)

    t_payload = payload_sym_nb * t_sym
    return t_preamble + t_payload

# Sanity check
print(lora_airtime(50, 7))   # ~0.0975 s
print(lora_airtime(50, 12))  # ~2.302 s
```

### 15.4 Modbus TCP Read (pymodbus)

```python
from pymodbus.client import ModbusTcpClient

client = ModbusTcpClient('192.168.1.50', port=502)
client.connect()

# Read holding registers 0..9 from unit 1
rr = client.read_holding_registers(address=0, count=10, slave=1)
if rr.isError():
    print("error:", rr)
else:
    print(rr.registers)  # [0x0123, 0x0456, ...]

# Write single register 100 = 42
client.write_register(address=100, value=42, slave=1)

client.close()
```

### 15.5 Zigbee2MQTT Topic Layout

```
zigbee2mqtt/                       # bridge state
zigbee2mqtt/bridge/state           # online|offline
zigbee2mqtt/bridge/devices         # full inventory JSON
zigbee2mqtt/bridge/request/...     # control commands
zigbee2mqtt/<friendly_name>        # device state (JSON)
zigbee2mqtt/<friendly_name>/set    # device commands
```

Bridge "request" topics let dashboards manage bindings, OTA updates, and pairing without raw Zigbee tooling.

---

## 16. Pitfalls and Operational Wisdom

### 16.1 Topic Sprawl in MQTT

A common anti-pattern: deeply nested topics like `factory/site/zone/cell/machine/sensor/temperature`. Every level adds bytes to **every PUBLISH** (full topic string transmitted, not just for first publish like AMQP). At 100 publishes/sec from 10,000 devices, a 50-byte topic is **50 MB/s of pure topic strings**.

**Mitigations:**
- Keep topics ≤ 32 chars when possible
- Use MQTT 5.0 Topic Aliases (numeric ID instead of full topic after first publish — saves bytes)
- Use MQTT-SN if running on constrained networks where bytes hurt

### 16.2 CoAP UDP Fragmentation

CoAP over IPv6 over 6LoWPAN has an effective MTU of ~80 bytes per fragment. A 200-byte CoAP message over Thread fragments at 6LoWPAN, requiring multiple radio frames and ACKs. Block-wise transfer + DTLS adds yet more overhead. Targets: keep CoAP messages under 80 bytes payload where possible, use OSCORE rather than DTLS to avoid 60-byte handshake bloat.

### 16.3 LoRaWAN Joining Storm

A power outage knocks 10,000 devices offline. When power returns they all rejoin simultaneously. The network server gets overwhelmed; gateways saturate; devices time out and rejoin again, exacerbating the storm.

**Mitigations:**
- Random join backoff (devices delay rejoin by `rand(0, 60s)` after power-on)
- Class A staggering — devices remember their FCnt across power cycles via NVM
- Scale gateway capacity for peak rejoin load (10x steady state)

### 16.4 Modbus Word-Order Mismatch

A vendor labels their float as "Big-endian" but means register-order big-endian, while the master assumes byte-order big-endian. Result: 1.0 reads as 0x803F0000 instead of 0x3F800000 → garbled value.

**Diagnostic:** write a known float (1.0 = 0x3F800000) from a master with verified byte order, read raw registers, compare against the four orderings (ABCD, CDAB, BADC, DCBA).

### 16.5 BLE Connection Parameter Negotiation

A peripheral suggests a 100 ms connection interval; the central (e.g. iOS) silently bumps it to 1500 ms because Apple's Connection Parameter Update Procedure constraints. Application sees mysterious latency. **Always check connection parameters after connect on iOS** (CBPeripheral's `maximumWriteValueLength`, `connectionInterval` via the CB API or BlueZ's `mgmt-api`).

### 16.6 OPC UA Certificate Trust Stores

OPC UA uses two trust lists per endpoint: trusted certs (whitelist) and rejected certs (auto-quarantine). When a new client first connects, its cert lands in `rejected/`; an operator must manually move it to `trusted/`. **First-connect failures are not bugs** — they're security policy.

### 16.7 Matter Commissioning QR Code Format

Matter setup codes encode (in a 90-bit big-endian integer):
```
Version (3) | VendorID (16) | ProductID (16) | CommissioningFlow (2) | DiscoveryCapMask (8) | Discriminator (12) | Passcode (27) | Padding (4) | TLV (variable)
```

Encoded as base-38 in `MT:` prefix. Don't try to encode by hand — use chip-tool or matter-shell.

---

## 17. Vocabulary Reference

| Term | Meaning |
|:---|:---|
| AEAD | Authenticated Encryption with Associated Data (e.g. AES-CCM, AES-GCM) |
| ADR | Adaptive Data Rate (LoRaWAN) — server commands SF/power changes |
| BDP | Bandwidth-Delay Product |
| CCA | Clear Channel Assessment (802.15.4 listen-before-talk) |
| CON | Confirmable (CoAP message type, requires ACK) |
| CSS | Chirp Spread Spectrum (LoRa modulation) |
| CSL | Coordinated Sampled Listening (802.15.4 low-power MAC) |
| DTLS | Datagram TLS, RFC 6347 / 9147 |
| GATT | Generic Attribute Profile (BLE) |
| Goodput | Useful payload throughput (excludes headers, retransmits) |
| IPSO | IP Smart Objects (LwM2M Smart Object Registry) |
| MIC | Message Integrity Code (LoRaWAN, 32-bit AES-CMAC) |
| NON | Non-confirmable (CoAP message type, fire-and-forget) |
| OSCORE | Object Security for Constrained RESTful Environments, RFC 8613 |
| PASE | Password Authenticated Session Establishment (Matter) |
| PSK | Pre-Shared Key |
| QoS | Quality of Service (MQTT delivery levels) |
| RSSI | Received Signal Strength Indicator |
| SED | Sleepy End Device (Zigbee/Thread) |
| SF | Spreading Factor (LoRa, 7–12) |
| SPAKE2+ | PAKE used by Matter PASE |
| SZX | Block Size eXponent (CoAP Block-wise) |
| TLV | Type-Length-Value encoding |
| TKL | Token Length (CoAP) |
| ZCL | Zigbee Cluster Library |
| ZED | Zigbee End Device |

---

## 18. See Also

- `networking/mqtt` — the MQTT cheat sheet (broker, topics, QoS, mosquitto)
- `security/tls` — TLS handshake mathematics, also background for DTLS
- `ramp-up/iot-protocols-eli5` — narrative ramp-up companion to this deep-dive

---

## 19. References

- **OASIS MQTT 5.0** — MQTT Version 5.0 OASIS Standard, 2019. https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html
- **OASIS MQTT 3.1.1** — MQTT Version 3.1.1 OASIS Standard, 2014. (ISO/IEC 20922:2016)
- **RFC 7252** — The Constrained Application Protocol (CoAP), Shelby et al, 2014
- **RFC 7641** — Observing Resources in the Constrained Application Protocol, Hartke, 2015
- **RFC 7959** — Block-Wise Transfers in the Constrained Application Protocol (CoAP), Bormann & Shelby, 2016
- **RFC 8613** — Object Security for Constrained RESTful Environments (OSCORE), Selander et al, 2019
- **RFC 6347** — Datagram Transport Layer Security Version 1.2, Rescorla & Modadugu, 2012
- **RFC 9147** — The Datagram Transport Layer Security (DTLS) Protocol Version 1.3, Rescorla et al, 2022
- **RFC 6282** — Compression Format for IPv6 Datagrams over IEEE 802.15.4-Based Networks, Hui & Thubert, 2011
- **RFC 6690** — Constrained RESTful Environments (CoRE) Link Format, Shelby, 2012
- **OMA LwM2M** — Lightweight Machine to Machine Technical Specification, Open Mobile Alliance, 1.2.1 (2022)
- **LoRa Alliance LoRaWAN L2 1.0.4** — LoRaWAN Link Layer Specification, October 2020
- **LoRa Alliance LoRaWAN 1.1** — LoRaWAN Specification v1.1, October 2017
- **Semtech AN1200.13** — LoRa Modem Designer's Guide (airtime formulas)
- **IEEE 802.15.4-2020** — IEEE Standard for Low-Rate Wireless Networks
- **Zigbee R23** — Zigbee Specification Revision 23, Connectivity Standards Alliance, 2023
- **Zigbee Cluster Library Specification R8** — CSA, 2021
- **Bluetooth Core Specification 5.4** — Bluetooth SIG, 2023
- **Thread 1.3 Specification** — Thread Group, 2022
- **Matter 1.3 Core Specification** — Connectivity Standards Alliance, 2024
- **OPC UA — IEC 62541** — OPC Foundation; UA Part 1 (Overview), Part 4 (Services), Part 6 (Mappings), Part 14 (PubSub)
- **Modbus Application Protocol Specification V1.1b3** — Modbus Organization, 2012
- **Modbus over TCP/IP Implementation Guide V1.0b** — Modbus Organization, 2006
- **Modbus/TCP Security** — Modbus Organization, 2018
- **OWASP IoT Top Ten** — OWASP, current revision

---
