# MQTT (Message Queuing Telemetry Transport)

Lightweight publish-subscribe messaging protocol designed for constrained devices and low-bandwidth networks, using a broker architecture over TCP (port 1883) or TLS (port 8883) with three QoS levels and retained messages.

## MQTT Architecture

```
                        +-----------+
Publisher A ----pub---->|           |----sub----> Subscriber X
  (sensor)    topic:    |   MQTT    |   topic:     (dashboard)
              temp/room1|  Broker   |   temp/#
                        |           |
Publisher B ----pub---->| (mosquitto|----sub----> Subscriber Y
  (camera)    topic:    |  or EMQX) |   topic:     (alerting)
              cam/front |           |   cam/+
                        +-----------+
                             |
                        Retained msgs
                        Will messages
                        Session state
```

## Topic Structure and Wildcards

```
# Topic hierarchy uses / as separator
sensors/building1/floor2/temp
home/livingroom/light/status
devices/device-abc123/telemetry

# Single-level wildcard: +
sensors/+/floor2/temp
  matches: sensors/building1/floor2/temp
           sensors/building2/floor2/temp
  no match: sensors/building1/floor3/temp

# Multi-level wildcard: #
sensors/#
  matches: sensors/building1/floor2/temp
           sensors/building1/floor2/humidity
           sensors/anything/at/any/depth

# System topics (broker info)
$SYS/broker/clients/connected
$SYS/broker/messages/received
$SYS/broker/uptime

# Topic design best practices
# Good: {category}/{location}/{device}/{measurement}
# Bad:  flat names, spaces, special characters
```

## QoS Levels

```
QoS 0 — At most once ("fire and forget")
  Publisher ---PUBLISH---> Broker
  No acknowledgment, no retry
  Use for: telemetry where occasional loss is acceptable

QoS 1 — At least once (may duplicate)
  Publisher ---PUBLISH---> Broker
  Publisher <---PUBACK---- Broker
  Retries until PUBACK received; subscriber may get duplicates
  Use for: logging, alerts where duplicates are tolerable

QoS 2 — Exactly once (four-step handshake)
  Publisher ---PUBLISH---> Broker
  Publisher <---PUBREC---- Broker    (received)
  Publisher ---PUBREL---> Broker     (release)
  Publisher <---PUBCOMP--- Broker    (complete)
  Use for: billing, critical commands where duplicates are harmful

# QoS is set independently for publish and subscribe
# Effective QoS = min(publish QoS, subscribe QoS)
```

## mosquitto Broker Configuration

```bash
# Install
sudo apt install mosquitto mosquitto-clients   # Debian/Ubuntu
brew install mosquitto                         # macOS

# Main config: /etc/mosquitto/mosquitto.conf

# Listener
listener 1883
protocol mqtt

# TLS listener
listener 8883
certfile /etc/mosquitto/certs/server.crt
keyfile /etc/mosquitto/certs/server.key
cafile /etc/mosquitto/certs/ca.crt

# WebSocket listener (for browser clients)
listener 9001
protocol websockets

# Authentication
password_file /etc/mosquitto/passwd
allow_anonymous false

# ACL (access control)
acl_file /etc/mosquitto/acl

# Persistence
persistence true
persistence_location /var/lib/mosquitto/

# Logging
log_dest file /var/log/mosquitto/mosquitto.log
log_type all

# Limits
max_connections 1024
max_inflight_messages 20
max_queued_messages 1000
message_size_limit 268435456

# Bridge to another broker
connection bridge-remote
address remote-broker.example.com:8883
topic sensors/# out 1
topic commands/# in 1
bridge_cafile /etc/mosquitto/certs/ca.crt

# Restart
sudo systemctl restart mosquitto
```

## mosquitto User Management

```bash
# Create password file
sudo mosquitto_passwd -c /etc/mosquitto/passwd alice

# Add user
sudo mosquitto_passwd /etc/mosquitto/passwd bob

# Delete user
sudo mosquitto_passwd -D /etc/mosquitto/passwd bob

# ACL file format (/etc/mosquitto/acl)
# Per-user rules
user alice
topic readwrite sensors/#
topic read $SYS/#

user bob
topic read sensors/+/temp
topic write commands/+

# Pattern-based (uses %u for username, %c for client id)
pattern readwrite devices/%u/#
```

## mosquitto_pub / mosquitto_sub

```bash
# Subscribe to all sensor topics
mosquitto_sub -h broker.example.com -t "sensors/#" -v

# Subscribe with QoS 1
mosquitto_sub -h broker.example.com -t "alerts/+" -q 1

# Subscribe with TLS
mosquitto_sub -h broker.example.com -p 8883 \
  --cafile ca.crt --cert client.crt --key client.key \
  -t "sensors/#"

# Subscribe with authentication
mosquitto_sub -h broker.example.com -u alice -P secret -t "data/#"

# Publish a message
mosquitto_pub -h broker.example.com -t "sensors/temp" -m "22.5"

# Publish retained message
mosquitto_pub -h broker.example.com -t "config/device1" -m '{"interval":60}' -r

# Publish with QoS 2
mosquitto_pub -h broker.example.com -t "billing/charge" -m '{"amount":9.99}' -q 2

# Publish from stdin
echo "hello" | mosquitto_pub -h broker.example.com -t "test" -l

# Publish binary data from file
mosquitto_pub -h broker.example.com -t "firmware/update" -f /path/to/firmware.bin

# Last Will and Testament
mosquitto_sub -h broker.example.com -t "devices/#" \
  --will-topic "devices/sensor1/status" \
  --will-payload "offline" \
  --will-qos 1 \
  --will-retain
```

## MQTT 5.0 Features

```
Feature                    Description
─────────────────────────  ─────────────────────────────────────
Reason Codes               Detailed error codes for all ACKs
User Properties            Key-value metadata on any packet
Shared Subscriptions       Load-balance messages across subscribers
                           $share/group/topic
Topic Aliases              Integer aliases to reduce bandwidth
Message Expiry Interval    TTL on published messages (seconds)
Session Expiry Interval    How long broker keeps session after disconnect
Request/Response           Response Topic + Correlation Data headers
Flow Control               Receive Maximum limits in-flight messages
Subscription Options       No Local, Retain As Published, Retain Handling
Auth Enhancement           Extended AUTH packet for SCRAM, Kerberos
Will Delay Interval        Delay before sending Last Will message
```

## Python paho-mqtt Client

```python
import paho.mqtt.client as mqtt

def on_connect(client, userdata, flags, rc, properties=None):
    print(f"Connected: {rc}")
    client.subscribe("sensors/#", qos=1)

def on_message(client, userdata, msg):
    print(f"{msg.topic}: {msg.payload.decode()}")

client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)
client.username_pw_set("alice", "secret")
client.tls_set(ca_certs="ca.crt")

client.on_connect = on_connect
client.on_message = on_message

# Last Will
client.will_set("status/client1", "offline", qos=1, retain=True)

client.connect("broker.example.com", 8883)
client.loop_forever()
```

## MQTT over WebSocket

```bash
# mosquitto WebSocket listener
listener 9001
protocol websockets

# nginx reverse proxy for MQTT-WS
location /mqtt {
    proxy_pass http://127.0.0.1:9001;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400s;
}
```

## Tips

- Use QoS 0 for high-frequency telemetry (temperature, GPS); QoS 1 for alerts; QoS 2 only for critical commands
- Always set a Last Will message so subscribers know when a device disconnects unexpectedly
- Use retained messages for device status topics so new subscribers immediately get the current state
- Design topic hierarchies with wildcards in mind; avoid putting variable data (timestamps) in topic names
- Set `clean_session=false` (MQTT 3.1.1) or `session_expiry_interval > 0` (MQTT 5.0) for durable subscriptions
- Enable TLS (port 8883) and require authentication; MQTT credentials are sent in plaintext without TLS
- Use shared subscriptions (`$share/group/topic`) in MQTT 5.0 to load-balance across multiple consumers
- Set `message_size_limit` in mosquitto to prevent memory exhaustion from oversized payloads
- Monitor `$SYS/#` topics for broker health: connected clients, message rates, retained message count
- Use bridge connections between brokers for geographic distribution rather than clustering
- Keep client IDs stable and unique; changing client IDs breaks session persistence
- Set reasonable `max_inflight_messages` (default 20) to prevent overwhelming slow subscribers

## See Also

- mosquitto, websocket, tls, rabbitmq, kafka, coap

## References

- [MQTT 5.0 Specification (OASIS)](https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html)
- [MQTT 3.1.1 Specification (OASIS)](https://docs.oasis-open.org/mqtt/mqtt/v3.1.1/mqtt-v3.1.1.html)
- [Eclipse Mosquitto](https://mosquitto.org/)
- [EMQX — Scalable MQTT Broker](https://www.emqx.io/)
- [Eclipse Paho MQTT Clients](https://eclipse.dev/paho/)
- [MQTT.org](https://mqtt.org/)
