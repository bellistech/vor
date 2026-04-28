# IoT Protocols — ELI5

> IoT protocols are walkie-talkies for tiny battery-powered things — they whisper instead of shout so a coin cell can last five years.

## Prerequisites

- `ramp-up/ip-eli5` — what an IP address is, what a packet is, why packets get lost
- `ramp-up/wifi-eli5` — what radios are, what channels are, how a thing joins a network

You do not need to be an electrical engineer. You do not need to know what a "transistor" is. You do not need to have ever soldered anything. You do not need to own any "smart" anything. By the end of this sheet you will know why your friend's smart doorbell needs Wi-Fi but their door sensor doesn't, why a Zigbee bulb is more reliable than a Wi-Fi bulb, and why the cattle tracker on a farm in Wyoming uses a radio that can talk for ten miles but only sends one number per minute.

If a word feels weird, look it up in the **Vocabulary** table near the bottom (chunk 4). Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is IoT

### The toaster joke

Some people make fun of "IoT," which stands for **Internet of Things**. The joke goes: "the S in IoT stands for Security." (There is no S. That's the joke. IoT devices are famously badly secured.)

Other people roll their eyes and say, "Why does my fridge need internet?" Fair question. Most of the time it doesn't.

But forget the toaster jokes for a minute. IoT is not really about smart fridges. IoT is about **tiny computers that don't have a person sitting in front of them.** That's it. That's the whole idea.

### What "thing" means

A "thing" in IoT is anything that:

1. Has a tiny computer inside it (a **microcontroller**, much smaller and weaker than the laptop or phone you're reading this on).
2. Has at least one **sensor** (something that measures a thing — temperature, motion, light, vibration, water level, gas concentration, position, weight, magnetic field, sound, anything) or one **actuator** (something that does a thing — turn a valve, flip a switch, push a piston, ring a bell, blink a light).
3. Talks over a network to either another "thing" or to a **gateway** (a translator) or to a **cloud server** (someone else's computer somewhere on the internet).

That is the whole list. A door sensor is a thing. A soil moisture probe in a vineyard is a thing. A vibration sensor on a factory machine is a thing. A heart-rate strap is a thing. A smart light bulb is a thing. A car's tire-pressure monitor is a thing. A bus that reports its GPS location is a thing. A pacemaker is a thing.

### Why "things" are different from laptops

Your laptop has:

- A wall socket, or at least a big rechargeable battery you charge every day.
- Loads of memory (gigabytes).
- A fast CPU.
- A big screen, a keyboard, a person sitting in front of it.
- A Wi-Fi card that can scream at the access point in 80 MHz of spectrum and download a movie in three minutes.

A door sensor has:

- One coin cell battery (CR2032 — like the watch battery in a car key fob). The user wants it to last **five years** before they have to change it.
- Eight kilobytes of RAM. Not gigabytes. Not megabytes. Kilobytes.
- A CPU running at 32 MHz. That's about a thousand times slower than your laptop.
- No screen. No keyboard. No person.
- A radio that can only whisper, because radios eat batteries.

Those two devices need radically different networking. If you put your laptop's Wi-Fi card in the door sensor, the battery would last about **four hours**. That's not a typo. Wi-Fi is just that hungry. You'd need to climb a ladder and change the battery every four hours forever.

So we don't put Wi-Fi in the door sensor. We put a different radio in. A radio designed to sip, not gulp. And once we use a different radio, we have to use different protocols on top of it, because the old protocols (TCP, HTTP, the things web browsers use) assume you have plenty of battery and bandwidth and memory, which we don't.

That is what this sheet is about. **The radios that sip and the protocols that whisper.**

## Why IoT Needs Different Networking

There are four constraints that drive every single IoT design decision. Memorise these. If you know these, you can predict what any IoT engineer will say next.

### 1. Battery life

The thing has to run for years on a tiny battery. That means **the radio has to be off almost all the time.** A door sensor's radio is on for maybe one millisecond out of every five seconds — and that's only when the door is moving. When the door is closed and nothing is happening, the radio might be on for one millisecond every minute, just to say "still alive."

Compare that to your phone. Your phone's Wi-Fi radio is on **continuously** while the screen is on. Even when the screen is off, it wakes up every 100 milliseconds to check for incoming traffic. That's why your phone needs charging every day.

Battery life means: short messages, infrequent transmissions, simple protocols. The whole network has to be designed around a thing being **asleep** by default and only waking up to say something brief.

### 2. Latency (or lack thereof — IoT often doesn't care)

Your web browser cares a lot about latency. If the page takes more than a couple seconds to load you get angry.

A soil moisture sensor does not care about latency. If the moisture reading takes ten minutes to reach the cloud, fine. The soil isn't going anywhere. The decision the cloud makes ("turn on the sprinklers") can happen on a five-minute schedule. Nobody dies.

Some IoT does care about latency — a smoke detector, a panic button, a car's brake-by-wire system. But most IoT is fine with seconds-to-minutes of latency. That gives us a lot of design freedom: we can let the device sleep for ages, wake up briefly, send the message, and go back to sleep.

### 3. Range

Wi-Fi works for about 30 metres indoors. Sometimes less. If your access point is upstairs and the sensor is in the basement, Wi-Fi probably won't reach.

Some IoT use cases need much longer range:

- A cattle tracker on a 10,000-acre ranch — the cow might be 5 km from the gateway.
- A water meter buried in the ground at the edge of a property.
- A parking sensor on the street, half a kilometre from the city's nearest receiver.
- A shipping-container tracker in the middle of an ocean (well, that needs satellite).

For these use cases we have **LPWAN** — Low Power Wide Area Networks. LoRaWAN, Sigfox, NB-IoT, LTE-M. These radios use much lower frequencies (sub-GHz) and much narrower bandwidths than Wi-Fi, which lets them shout across kilometres while still sipping power. The trade-off is they can only send a tiny amount of data — sometimes 12 bytes per message, with maybe one message per hour allowed.

### 4. Payload size

A picture from your phone is a couple of megabytes. A web page is hundreds of kilobytes. An IoT message is often **fewer than 50 bytes**. That's not a typo. Fifty bytes total. A LoRaWAN message at the longest range is twelve bytes. A Sigfox message is twelve bytes. A Zigbee message is around a hundred bytes maximum.

This means you cannot use HTTP. HTTP headers alone are a hundred bytes before you even put a body in. You cannot use JSON the way you'd use it on the web — `{"temperature":21.5}` is twenty bytes just for one number, and that's after you compress the whitespace. You need binary protocols, with single-byte field IDs and tightly packed values.

### Putting it together

Battery, latency, range, payload. These four constraints push IoT toward:

- **Radios that sip:** BLE, Zigbee, Z-Wave, Thread, LoRaWAN, NB-IoT, LTE-M.
- **Protocols that whisper:** CoAP instead of HTTP, MQTT (and especially MQTT-SN) instead of REST, CBOR instead of JSON.
- **Stateless connections:** the device sends one message and forgets the connection ever existed, because remembering would require staying awake.
- **Edge gateways:** a battery-free, plugged-in box that does the heavy lifting (TCP, TLS, JSON parsing) on behalf of the sleepy device.

Now we'll walk through each radio family and each protocol, in order from "shortest range, most popular" to "longest range, most niche."

## Bluetooth Low Energy (BLE)

### The radio in your pocket already knows how to whisper

You've heard of Bluetooth. It's the thing that connects your wireless headphones to your phone. What you may not know is that there are **two different Bluetooths**, and they are mostly unrelated:

1. **Bluetooth Classic** — the older one, designed for streaming audio. High data rate, high power, designed to be on for hours at a time.
2. **Bluetooth Low Energy (BLE)** — the newer one (introduced in Bluetooth 4.0 in 2010), designed for IoT. Low data rate, very low power, designed to be off for almost all of the time.

Same name. Same chip (modern Bluetooth chips do both). Almost completely different protocol stacks. When people talk about "Bluetooth" for headphones, they mean Classic. When they talk about "Bluetooth" for fitness trackers, smartwatches, beacons, door sensors, and the like — they mean BLE.

### What BLE looks like in real life

- Your fitness tracker pairs with your phone over BLE.
- Your smartwatch shows phone notifications over BLE.
- A grocery store's "near you" notifications are BLE beacons.
- A glucose monitor on a diabetic patient's arm sends readings over BLE.
- A wireless keyboard or mouse uses BLE.
- A Tile or AirTag finder uses BLE.
- A car's keyless entry uses BLE.

The pattern: **a sleepy thing with a battery talks to a not-sleepy thing with mains power or a much bigger battery (your phone) over short range, briefly, every few seconds or minutes.**

BLE range is similar to Wi-Fi — about 10 metres typical, 30 metres if you're lucky, more outdoors with line of sight. BLE 5 (2016) introduced a long-range mode (called Coded PHY) that can reach 100 metres or more by sending each bit multiple times to make it more robust to noise.

### How BLE saves power

The trick is that BLE devices spend almost all of their time **asleep with the radio off.** They wake up briefly to either:

- **Advertise** (broadcast a short message saying "hi, I exist, here are my IDs and capabilities"), or
- **Connect** (briefly exchange a few packets with a peer, then disconnect or go back to sleep).

A typical BLE peripheral wakes up for **10 milliseconds** every **second**. That's a 1% duty cycle. The radio is off 99% of the time. That's how you get five-year battery life from a coin cell.

The advertising packet itself is tiny — 31 bytes of payload. Enough for a device name, a service ID, and maybe one sensor reading. Enough for a doorbell to say "ding," for a beacon to say "I'm beacon #42 at aisle 7," or for a temperature sensor to say "21.5 degrees."

## BLE Stack (PHY, Link Layer, L2CAP, ATT, GATT, GAP)

### A tower of layers

Like every networking protocol, BLE is a tower of layers. Each layer talks to the layer above and below it. Each layer adds a little bit to the message before sending it down, and strips a little bit off as the message comes up.

Here is the BLE protocol stack tower, from the bottom (the radio waves themselves) to the top (your app):

```
+-----------------------------------------+
|        Your app: "blink the LED"        |  <- application code
+-----------------------------------------+
|                  GATT                   |  <- "characteristics" (named values)
+-----------------------------------------+
|                  ATT                    |  <- read/write to attribute handles
+-----------------------------------------+
|                  L2CAP                  |  <- channels (ATT, signalling, custom)
+-----------------------------------------+
|              Link Layer                 |  <- packets, retransmits, encryption
+-----------------------------------------+
|                  PHY                    |  <- 2.4 GHz radio waves
+-----------------------------------------+
|              ((( radio )))              |
+-----------------------------------------+
```

(GAP — the **Generic Access Profile** — is not in this tower because it's not a layer. It's a sideways thing, a set of rules that says how devices discover each other and start connections. It governs the lower layers from the side, not from above. We'll cover it in a moment.)

Let's walk up the tower from the bottom.

### PHY — the radio itself

PHY stands for "physical layer." This is the actual radio at 2.4 GHz, the same band Wi-Fi uses. BLE divides this band into 40 channels of 2 MHz each. Three of those channels (37, 38, and 39) are **advertising channels** — channels devices use to broadcast "I exist." The other 37 channels are **data channels** — channels used for actual two-way conversations.

The advertising channels are deliberately spread across the band so that even if Wi-Fi is hammering one part of the spectrum, at least one of the three advertising channels will probably get through. Devices broadcast on all three advertising channels in turn — first 37, then 38, then 39, then back to 37.

BLE 5 added two new PHYs:

- **2M PHY** — twice the data rate (2 megabits per second instead of 1), for shorter air time and slightly less power per byte. Useful for firmware updates over BLE.
- **Coded PHY** — slower (125 kbps or 500 kbps) but each bit is repeated, so noise can corrupt some bits and the receiver still gets the message. This is how you get long-range BLE.

### Link Layer — packets, retransmits, encryption

Above the PHY sits the Link Layer. This is where bits get grouped into packets, where packets get checksums, where retransmissions happen if a packet was lost, and where encryption keys live.

The Link Layer also runs the **state machine** — every BLE device is in one of a few states at any moment:

- **Standby** — radio off. No traffic. The default state.
- **Advertising** — broadcasting "I exist" packets on the advertising channels. Done by **peripherals** (the sleepy things).
- **Scanning** — listening for advertising packets. Done by **centrals** (the phones).
- **Initiating** — heard an advertisement, now sending a CONNECT_REQ to start a connection.
- **Connection** — a connection is open. Both sides exchange packets at agreed intervals.

A peripheral that just wants to announce its existence can stay in advertising forever. A peripheral that wants two-way conversation will advertise, accept a connection, exchange data, then either drop the connection or stay connected.

Connection intervals are negotiable from 7.5 milliseconds to 4 seconds. Long intervals save power but mean a slow conversation. Short intervals are responsive but eat battery.

### L2CAP — channels

L2CAP stands for "Logical Link Control and Adaptation Protocol." Everyone calls it L2CAP. It does two jobs:

1. **Multiplexing** — letting multiple "channels" (logical streams of data) share the single Link Layer connection. There's a channel for ATT, a channel for the Security Manager, a channel for the Signalling protocol, and there can be custom channels for your own app.
2. **Fragmentation and reassembly** — if a message is bigger than what the Link Layer can carry in one packet, L2CAP chops it up and the receiver glues it back together.

Most of the time you don't think about L2CAP. It just works. But if you ever want to do high-throughput streaming over BLE — say, to send audio or a firmware blob — you'll create a custom L2CAP channel (called a CoC, Connection-oriented Channel) and pump bytes through it directly.

### ATT — the Attribute Protocol

ATT stands for "Attribute Protocol." This is where BLE starts to look really different from Wi-Fi.

In ATT, every BLE device exposes a list of **attributes**. Each attribute has:

- A **handle** — a 16-bit number, like a row number in a table.
- A **type** — a UUID that says what kind of thing this attribute is.
- A **value** — the actual bytes.
- **Permissions** — read, write, both, or read with notify.

A central can:

- **Read** an attribute by handle.
- **Write** an attribute by handle.
- **Subscribe** to notifications, so the peripheral can push the value when it changes.

That's basically it. ATT is just a key-value store living inside the peripheral, where the keys are 16-bit handles and the values are byte arrays. Everything in BLE is built on top of this idea.

This is wildly different from HTTP, where you have URLs and methods and bodies. In BLE you have handles and reads and writes. The "URL" is a number from 0 to 65535. The "body" is up to a few hundred bytes.

### GATT — the Generic Attribute Profile

GATT stands for "Generic Attribute Profile." If ATT is the raw key-value store, GATT is the convention for organising it.

GATT says: group attributes into **services**, and inside each service, define **characteristics**, and inside each characteristic, define **descriptors**.

Here is the GATT tree for a heart-rate monitor:

```
Heart Rate Monitor (peripheral)
|
+-- Service: Heart Rate (UUID 0x180D)
|   |
|   +-- Characteristic: Heart Rate Measurement (UUID 0x2A37)
|   |   |
|   |   +-- Descriptor: Client Characteristic Configuration (notify on/off)
|   |   +-- Value: <heart rate bytes, updated every second>
|   |
|   +-- Characteristic: Body Sensor Location (UUID 0x2A38)
|   |   |
|   |   +-- Value: 0x01  (chest)
|   |
|   +-- Characteristic: Heart Rate Control Point (UUID 0x2A39)
|       |
|       +-- Value: <write 0x01 here to reset energy expended>
|
+-- Service: Battery Service (UUID 0x180F)
|   |
|   +-- Characteristic: Battery Level (UUID 0x2A19)
|       |
|       +-- Descriptor: Client Characteristic Configuration
|       +-- Value: 0x5A  (90 % full)
|
+-- Service: Device Information (UUID 0x180A)
    |
    +-- Characteristic: Manufacturer Name String  ("Acme Wearables")
    +-- Characteristic: Model Number String       ("HR-100")
    +-- Characteristic: Firmware Revision String  ("1.2.3")
```

Read this tree top to bottom:

1. The peripheral hosts several **services**. Each service is a group of related features.
2. Each service contains one or more **characteristics**. A characteristic is one named, typed value (or a small set of values), with permissions (readable, writable, notifiable).
3. Each characteristic can have **descriptors** — small bits of metadata. The most common one is the Client Characteristic Configuration Descriptor (CCCD), which is a one-byte switch that says "yes, please notify me when this value changes" or "no, don't bother."

A central app, like your phone's heart-rate app, **discovers** this tree by walking it. It asks "what services do you have?" gets a list, then for each service "what characteristics?" gets a list, then for each characteristic "what descriptors?" After that, it knows the layout. Now it can read characteristics, write to them, or subscribe to notifications.

The Bluetooth SIG (Special Interest Group) maintains a list of standard service and characteristic UUIDs. Heart Rate is `0x180D`. Battery is `0x180F`. Device Information is `0x180A`. There are dozens. If your device exposes one of these standard services with the standard characteristics, then *any* phone app that understands "Heart Rate Service" can talk to *any* manufacturer's heart-rate monitor. That's the power of GATT — a common vocabulary for sleepy things.

### GAP — the Generic Access Profile

GAP defines **roles** and **modes**. A device can be:

- **Peripheral** — the sleepy thing that advertises. Battery-powered.
- **Central** — the not-sleepy thing that scans and connects. Phone, hub, gateway.
- **Broadcaster** — only advertises, never accepts connections (a beacon).
- **Observer** — only listens to advertisements, never connects (a presence detector).

A device can be in one or more **GAP modes**:

- **Connectable** — willing to accept connection requests.
- **Discoverable** — including its name in advertising packets so a user can pick it from a list.
- **Bondable** — willing to remember pairing keys for next time.

GAP also handles **pairing** and **bonding** — the song and dance where two devices exchange keys so they can encrypt their connection in the future. There are several pairing methods (Just Works, Numeric Comparison, Passkey Entry, Out of Band) with different levels of security.

When you tap "pair this device" on your phone and type in `0000` or `1234` or watch six digits flash on a smartwatch screen, you are using GAP's pairing layer.

## BLE Profiles (HID, Heart Rate, Health Thermometer)

A **profile** in BLE is a recipe. It says "if you want to be a Heart Rate Monitor, here's the GATT tree you must expose, here are the GAP roles you must support, here are the security requirements."

A few common profiles:

- **HID over GATT (HOGP)** — for keyboards, mice, game controllers. Maps the existing USB HID descriptors onto BLE characteristics. This is why your wireless keyboard "just works" with your laptop.
- **Heart Rate Profile (HRP)** — heart-rate monitors. Defines the Heart Rate service and required characteristics.
- **Health Thermometer Profile (HTP)** — medical thermometers. Defines a temperature characteristic with a timestamp and a measurement-type field.
- **Cycling Speed and Cadence (CSC)** — bike speed and pedalling cadence sensors.
- **Glucose Profile (GLP)** — continuous glucose monitors.
- **Proximity Profile (PXP)** — alerts when your phone goes too far from a tagged item.
- **Find Me Profile (FMP)** — make a paired device beep so you can find it.

There are around 50 standard profiles, all defined by the Bluetooth SIG. If your device follows a standard profile, every off-the-shelf app for that profile will work with your device. If your device makes up its own custom service UUIDs (a "vendor-specific" profile), then only your app will know how to talk to it.

The advantage of standard profiles is interoperability. The advantage of custom profiles is freedom — you can do whatever you want, even if no standard exists. Most consumer IoT devices use a mix: a few standard services (Battery, Device Information) plus a custom service for the device's main feature.

## BLE Mesh (vs central/peripheral)

### One-to-one is sometimes not enough

Standard BLE is a **star** — one central in the middle, several peripherals around it. The phone is the central. The smartwatch, the heart-rate strap, the wireless mouse — those are all peripherals connected to the same phone.

That works fine for personal devices. But what if you want to control a hundred light bulbs in an office building? You don't want to walk a phone within 10 metres of each bulb. You want any single switch to control any single bulb, regardless of where they are in the building. You need the bulbs to **forward messages for each other.**

That's a **mesh network**.

### How BLE Mesh works

BLE Mesh (introduced in 2017) sits on top of the BLE PHY but uses a completely different higher-level model. Instead of central/peripheral and connections, BLE Mesh uses **flooding**.

When a switch wants to turn on a bulb, it broadcasts a message saying "bulb #42, turn on." Every nearby BLE Mesh node hears this message. Each node decides:

- "Is this for me? If yes, act on it."
- "Have I seen this exact message before? If yes, throw it away."
- "If I haven't seen it before, **rebroadcast** it after a tiny random delay."

Within a few hops, the message reaches every node in the building. Even if the switch is in the lobby and the bulb is on the third floor, the message hops from node to node until it arrives.

Each message has a **time-to-live (TTL)** counter that starts at, say, 7. Every rebroadcast decrements the TTL by 1. When the TTL hits 0, the message stops being forwarded. This prevents the network from echoing forever.

### Friend nodes and low-power nodes

Flooding sounds like it would burn battery. And it would, if every node had to listen all the time. So BLE Mesh defines a special pattern called the **Friendship**:

- A **Low-Power Node (LPN)** is a battery-powered node that mostly sleeps. It can't listen to the flood.
- A **Friend Node** is a mains-powered (or big-battery) node that's always awake. It listens to all the flood traffic on behalf of the LPN.
- The LPN periodically wakes up, asks its Friend "any messages for me?", drains the queue, and goes back to sleep.

This means a BLE Mesh light switch can run on a coin cell for years even though the rest of the network is gossiping like crazy. The switch only wakes up when somebody presses it, sends one message, and sleeps again. The bulbs (which are mains-powered) carry the network traffic.

### Models and elements

BLE Mesh has its own concepts that are loosely analogous to GATT services, but they're called something different:

- An **element** is one addressable unit on a node. A multi-channel light might have one element per channel.
- A **model** is a behaviour an element supports — the Generic OnOff model for switching, the Light HSL model for colour, the Sensor model for measurements.
- A **state** is a value inside a model — Generic OnOff has a state called "OnOff," which is exactly what you'd think.

If you've used Zigbee, this might sound familiar. BLE Mesh and Zigbee solved very similar problems and ended up with very similar abstractions, even though the radios and security models are quite different.

## Bluetooth Classic vs BLE

Quick reference, because people get these confused:

| Aspect | Bluetooth Classic | BLE |
|---|---|---|
| Year introduced | 1999 (1.0) | 2010 (4.0) |
| Use cases | Audio streaming, file transfer | Sensors, beacons, peripherals |
| Power | Watts | Milliwatts |
| Battery life | Hours to a day | Months to years |
| Data rate | 1-3 Mbps (sustained) | 1 Mbps PHY, but most devices use a tiny fraction |
| Latency | Low (audio) | Configurable (7.5 ms to 4 s) |
| Topology | Piconet (1 master + up to 7 slaves) | Star, plus Mesh extension |
| Service model | SDP + RFCOMM (serial-like) | GATT (attribute database) |
| Discovery | Inquiry, then page | Advertising on 3 channels |
| Pairing | Various, often PIN | LE Secure Connections (ECDH) |
| Same chip? | Yes — modern chips do both | Yes |

If you see a wireless headset, a phone-to-laptop file transfer, or a hands-free car kit — that's Bluetooth Classic. If you see a smartwatch, a fitness tracker, a beacon, a wireless mouse, a smart lightbulb, a wireless keyboard, a heart-rate strap, a glucose monitor, a tile finder, a smart doorbell sensor — that's BLE.

The Bluetooth SIG has been gently sunsetting Classic. New profiles are mostly BLE. Some old profiles (A2DP for audio) still require Classic. There's a new BLE-based audio standard (LE Audio, with the LC3 codec) that aims to replace A2DP in the next few years.

## Zigbee (802.15.4 PHY + Zigbee stack)

### The thing in your smart bulb

Zigbee is another short-range, low-power radio standard. It's the radio inside most smart bulbs (Philips Hue, IKEA Tradfri, Sengled), most door and motion sensors (Aqara, SmartThings, Hue Motion), and many smart locks and thermostats.

Like BLE, Zigbee uses the 2.4 GHz band. Unlike BLE, Zigbee was **mesh from day one** — every Zigbee device that's mains-powered acts as a router, and battery-powered end devices (like a door sensor) talk to the nearest router. This is why Zigbee lights are so good as a backbone: every plugged-in bulb is a relay.

Zigbee's PHY is the IEEE 802.15.4 standard, the same PHY used by Thread, by Matter (over Thread), and by some industrial protocols. 802.15.4 is the equivalent of Wi-Fi's 802.11 — a low-level radio standard that everyone shares, with different higher-level protocols (Zigbee, Thread, others) layered on top.

### Zigbee layers

```
+-----------------------------------------+
|       Application: "turn on bulb"       |
+-----------------------------------------+
|       Zigbee Cluster Library (ZCL)      |  <- clusters (OnOff, Level, Color)
+-----------------------------------------+
|       Application Support Sublayer      |  <- profiles, endpoints, bindings
+-----------------------------------------+
|        Zigbee Network Layer (NWK)       |  <- mesh routing, addresses
+-----------------------------------------+
|         IEEE 802.15.4 MAC               |  <- frames, ACKs, retries
+-----------------------------------------+
|         IEEE 802.15.4 PHY               |  <- 2.4 GHz radio, 250 kbps
+-----------------------------------------+
```

The big picture:

- **PHY** sends bits over the air at 250 kbps on one of 16 channels.
- **MAC** does packet framing, error detection, acknowledgement, and CSMA/CA (listen-before-talk).
- **NWK** is mesh routing — knows how to forward a packet from one Zigbee device to another, possibly via several hops.
- **APS** (Application Support Sublayer) handles addressing of "endpoints" within a device (a multi-bulb fixture might have one endpoint per bulb), groups (broadcasting to many bulbs at once), and **bindings** (the equivalent of "this switch controls those bulbs").
- **ZCL** (Zigbee Cluster Library) is a library of standard "clusters" — OnOff, Level Control, Color Control, Door Lock, Thermostat, Power Configuration. Each cluster defines a set of attributes (with IDs) and commands.

If GATT in BLE is "a key-value store on the peripheral with a tree of services and characteristics," ZCL in Zigbee is "a key-value store on each endpoint with a tree of clusters and attributes." Different names, very similar idea.

### Device types

A Zigbee network has three types of nodes:

- **Coordinator** — exactly one per network. Forms the network at startup, hands out addresses, holds the master security key. Always mains-powered.
- **Router** — forwards packets, can have children. Always mains-powered. Bulbs and plugs are routers.
- **End Device** — sleeps most of the time, talks to one parent router. Battery-powered. Door sensors, motion sensors, remotes are end devices.

Routers form a mesh among themselves; end devices hang off the mesh as leaves.

### Zigbee 3.0, profiles, and the dialect problem

For a long time Zigbee had a fragmentation problem: every vendor implemented their own slightly different profile (Zigbee Light Link, Zigbee Home Automation, Zigbee Pro, etc.) and devices from different vendors didn't always work together. Zigbee 3.0 (2017) merged all the profiles into one and dramatically improved interoperability.

There's still a residual issue: many vendors implement vendor-specific clusters on top of the standard ones, so a Hue bulb's full feature set isn't exposed to a non-Hue hub. Zigbee2MQTT is a popular open-source project that translates between every vendor's quirks and a clean MQTT API for home automation.

## Z-Wave (proprietary, controlled by Silicon Labs)

### What Z-Wave is

Z-Wave is another mesh-networking standard for home automation, very similar in scope to Zigbee. Where Zigbee uses 2.4 GHz, Z-Wave uses **sub-GHz** (around 868 MHz in Europe, 908 MHz in North America, varies by country). Sub-GHz penetrates walls better than 2.4 GHz and has less interference from Wi-Fi, microwaves, and Bluetooth.

The trade-off is that sub-GHz is heavily regulated by country, so Z-Wave devices are not portable — a Z-Wave lock you bought in the US won't work in Europe. The radios run on different frequencies in different regions and you have to buy the right region's chip.

Z-Wave was originally proprietary. Silicon Labs owned (and still effectively owns) the silicon — almost every Z-Wave device used a chip from Sigma Designs, then Silicon Labs after they acquired Sigma's Z-Wave business in 2018. The protocol specification is now public (since 2020) but the chips are still mostly single-source.

### How it works in one paragraph

Z-Wave uses a simple mesh: a hub (the controller) and lots of nodes that route for each other. There are no "end devices vs routers" distinctions in the same way Zigbee has them — battery-powered Z-Wave devices use a "FLiRS" (Frequently Listening Receiver Slave) mode where they wake up briefly to listen, similar to BLE Mesh's Low Power Nodes. Most door locks and battery sensors use FLiRS so they can be woken by the hub on demand.

Z-Wave's data rate is much lower than Zigbee's (40 or 100 kbps depending on the version), but the longer-range sub-GHz radio compensates. A typical Z-Wave network covers a whole house easily, often better than Zigbee in the same house.

### Why it still exists

When you can use Zigbee or Thread, why pick Z-Wave? A few reasons:

- **Better range and wall penetration** in sub-GHz.
- **Less interference** with Wi-Fi (different band).
- **Strict certification** — every Z-Wave device must pass Silicon Labs' certification, which means cross-vendor compatibility is much better than Zigbee historically was. A Z-Wave lock from one vendor really does work with a hub from another.
- **Long-running ecosystem** — many smart-home installers in North America standardised on Z-Wave 10+ years ago and have huge inventories of devices.

Z-Wave is slowly losing ground to Thread/Matter and Zigbee, especially as Matter starts to fill the cross-vendor compatibility role Z-Wave used to own. But it's still a real and shipping protocol you'll find in lots of installed smart homes.

## Thread (IPv6 + 802.15.4)

### What if Zigbee, but with IP addresses

Thread is a relatively new (2014) protocol that uses the same 802.15.4 PHY as Zigbee, but with a completely different network stack. Thread runs **IPv6** on top of 802.15.4, using a compression scheme called **6LoWPAN** (we'll get to this in chunk 2) to squeeze IPv6 into the tiny 802.15.4 frame.

Why does this matter? Because if your sleepy thing has a real IPv6 address, it can be addressed and reached just like any other internet device. You don't need a Zigbee-to-IP gateway that translates between two completely different worlds. You just need a **border router** that bridges the Thread mesh to your home's IPv6 network. Each Thread device gets a real address and can theoretically be reached by any other IPv6 device in your house (subject to firewalling and Thread's own access controls, which are strict by default).

### Thread architecture

A Thread network has several roles:

- **Border Router** — a mains-powered device with both Thread and Wi-Fi or Ethernet. It bridges the Thread mesh to the IP network. Apple HomePods, Google Nest hubs, and Amazon Eero routers are border routers.
- **Router** — mains-powered Thread node that forwards packets. Up to 32 routers per Thread network.
- **REED** (Router-Eligible End Device) — a node that's currently a leaf but can be promoted to a router if needed.
- **End Device** — a leaf node. Either an FED (Full End Device, always listening, mains-powered) or an SED (Sleepy End Device, battery-powered, mostly off).
- **Leader** — exactly one router per network is the "leader" and manages router IDs and network parameters. If the leader goes offline, another router takes over automatically.

Thread is **self-healing** — if a router goes offline, the mesh re-routes around it. There's no central coordinator the way Zigbee has; Thread's leader is a soft role that can fail over.

### Thread + Matter

Thread by itself is just transport — it gets IPv6 packets from one node to another. To do anything useful you need an application protocol on top. The application protocol of choice in 2024-2026 is **Matter** (formerly known as "Project CHIP" — Connected Home over IP). Matter is the successor to Zigbee's ZCL and Z-Wave's command classes — a standardised way for smart-home devices to expose features (lights, switches, locks, thermostats, sensors) that any Matter-compatible hub can understand.

Matter runs over Thread for low-power devices and over Wi-Fi for higher-power devices (TVs, big appliances). Both transports use the same Matter protocol on top, so the user-facing experience is identical regardless of which radio is underneath.

This is a big deal. For the first time, a single device can be paired into Apple Home, Google Home, Amazon Alexa, and Samsung SmartThings simultaneously, because all four hubs speak Matter. The bad old days of "this bulb works with Hue but not with HomeKit" are ending. Mostly.

### Why Thread instead of BLE Mesh or Zigbee

- **Real IPv6 addresses** — easier to bridge to the wider internet, easier to debug with normal tools (you can ping a Thread node).
- **No central coordinator** — the network heals itself; there's no single point of failure.
- **Lower latency than BLE Mesh** — Thread is a routed mesh (each packet takes the shortest path), not a flooded mesh (where every packet is rebroadcast). Less air time, less collision, lower latency at scale.
- **Designed for Matter** — every major smart-home platform is committing to Thread + Matter as the future. Even Samsung SmartThings is shifting away from Zigbee toward Thread.

The catch is that Thread requires more memory and CPU than Zigbee — IPv6, even in compressed form, is more complex than Zigbee's NWK layer. And Thread border routers are more expensive than Zigbee coordinators because they need both radios. But chip costs have come down to the point where this isn't really a barrier any more, and the long-term direction of the industry is clearly Thread.

## Matter (the unification: app layer over Thread + WiFi + Ethernet — one ecosystem)

Picture a neighborhood block party where every house brought a different kind of cookie. House A brought chocolate chip. House B brought oatmeal raisin. House C brought peanut butter. House D brought sugar cookies. They all taste fine on their own, but the kids running between yards have to keep switching plates, switching napkins, and switching cups depending on whose yard they are in. Every yard has its own rules. Every yard has its own way of doing things. By the end of the party, the kids are exhausted from switching, the parents are arguing about who has to clean up which plate, and nobody can find their own cup anymore.

Now imagine the parents got together one weekend before the party and said, "You know what? Let's all use the same plates, the same napkins, the same cups, and the same kind of table. We can each still bring our own cookies — chocolate chip is still chocolate chip, peanut butter is still peanut butter — but the *plates* are all the same. The *napkins* are all the same. The *cups* are all the same. So a kid running from House A's yard to House D's yard does not have to switch anything. They just keep their plate."

That is what Matter does for smart-home devices. The "yards" are different radio networks — Thread, Wi-Fi, Ethernet. The "cookies" are different kinds of devices — light bulbs, door locks, thermostats, plugs, sensors. Before Matter, each ecosystem (Apple HomeKit, Amazon Alexa, Google Home, Samsung SmartThings) had its own plates and napkins and cups. A bulb that worked with Apple did not work with Amazon. A lock that worked with Google did not work with Samsung. You bought a thing and prayed it spoke the language your phone spoke. If you had a mixed household, your fridge magnet held a stack of business cards from four different apps and you opened a different one for each room.

Matter says: every plate is the same plate now. Every napkin is the same napkin. The cookies (the devices) can still be made by anyone, but they all sit on the same plate (the Matter data model) and you can pick them up with the same napkin (the Matter commissioning flow) regardless of whose yard you are in.

### The actual technical bit, still in plain English

Matter is **only the application layer**. It is not a radio. It does not invent a new way to send bits through the air. Instead, Matter sits on top of three things that already exist:

```
+----------------------------------------------------+
|                Matter (application)                |  <- the same plates and napkins
+----------------------------------------------------+
|        IPv6 (everybody speaks Internet)            |  <- the same table
+----------------------------------------------------+
|  Thread        |   Wi-Fi          |   Ethernet     |  <- different yards
| (low-power     |   (high-bandwidth|   (wired,      |
|  mesh, 802.15.4)|   2.4/5 GHz)    |    rare)       |
+----------------+------------------+----------------+
```

Notice the key thing: **everything above the radios is the same**. A Thread bulb and a Wi-Fi bulb both speak Matter, both speak IPv6. To the app on your phone, they look identical. The phone does not care that one bulb whispers to its neighbors at 250 kbps over 802.15.4 and the other bulb shouts directly at the router over Wi-Fi. Both bulbs answer to the same commands, expose the same attributes, and live in the same address book.

### Commissioning (the "I just bought this thing, now what?" dance)

When you take a Matter device out of the box, there is a little QR code on the side. You point your phone at it. That QR code carries a code (a Setup Passcode) and a discriminator (a 12-bit ID so your phone can tell *this* bulb apart from any other Matter bulb that is also unconfigured nearby). Your phone uses Bluetooth Low Energy to walk up to the device, prove it knows the passcode, and hand it the keys to your network — your Wi-Fi password, or your Thread network's credentials. That is it. The device joins your house. From then on, BLE is no longer used; the device talks over Thread or Wi-Fi like a normal IPv6 device.

The big deal is **multi-admin**. The same device, after one commissioning, can be invited into Apple Home and Google Home and Amazon Alexa and Samsung SmartThings *at the same time*. You do not have to choose. The device holds multiple fabric credentials, one per ecosystem, and answers to all of them.

### What it does NOT do

Matter does not do video. Matter does not do audio. Matter does not do firmware updates over the air for everything (it has a spec for OTA, but it is optional and not all devices implement it). Matter does not do wide-area — it is local-only. Your devices talk to each other and to the Matter controllers in your house. There is no cloud requirement (though vendors can still add their own clouds on the side).

### Why you should care

If you are about to buy a smart bulb, **buy a Matter one**. You will not be locked in. If you switch ecosystems in two years (you sell your iPhone, you go all-in on Google), the bulbs come with you. That has never been true before in the smart-home world.

### A concrete commissioning walkthrough

Let us walk through exactly what happens, second by second, when you commission a Matter bulb. You bring the bulb home. You plug it in. The bulb's onboard LED starts pulsing — that means "I am unconfigured, please commission me." On the side of the box there is a small QR code. You open the Apple Home app on your phone, tap the plus sign, tap Add Accessory, and point your phone's camera at the QR code.

What just happened: the QR code encoded a few key things — a 27-bit Setup Passcode (the secret), a 12-bit Discriminator (so your phone can tell *this* unconfigured bulb apart from any *other* unconfigured bulb that happens to be nearby), a Vendor ID, and a Product ID. Your phone now knows the secret. It does not yet know the bulb.

Your phone turns on its Bluetooth Low Energy radio and starts scanning for advertisements that match the discriminator. The bulb has been advertising on BLE the whole time it has been unconfigured. The phone sees the bulb's advertisement, recognizes the discriminator, and connects over BLE.

Now they do a cryptographic handshake using the Setup Passcode. This is called PASE (Passcode-Authenticated Session Establishment). The phone proves it knows the passcode, the bulb proves it knows the passcode, and they end up sharing a temporary session key. Note: they did not just send the passcode in the clear. They proved knowledge of it without exposing it.

With the session key in hand, the phone now uses BLE to deliver the actual provisioning data: your Wi-Fi SSID and password (if the bulb is Wi-Fi), or the Thread network credentials (if the bulb is Thread). The bulb stores those, switches off BLE, and joins the production network. From this moment on, the bulb is just another IPv6 host on your home network. The whole process took maybe thirty seconds.

The phone then exchanges Operational Certificates with the bulb. These are X.509 certificates issued by the Apple Home fabric's root CA. The bulb stores the certificates and from now on knows it belongs to the Apple Home fabric in your house. If you also want this bulb in Google Home, you tap "Share with Google Home" and the same dance happens with a different fabric's certificates — the bulb now holds two operational certificates and answers to both ecosystems independently.

## LoRa / LoRaWAN (long-range, low-power, license-free ISM bands 868/915 MHz)

Imagine you live way out in the country. Your nearest neighbor is a mile away. You want to leave a message for them. You could:

- Walk over (slow, expensive in shoe leather, you get tired).
- Yell really loud (does not reach a mile).
- Send a text (works, but uses a cell phone bill, which costs money every month).
- Tie a note to a balloon and let it drift over (cute but unreliable).
- Use a really long string and two cans (works for short distances, breaks down).

LoRa is the "really loud whistle" option. It is a way to send a tiny bit of information — like a single sentence, "the cow tank is full" — across miles of countryside, using almost no power, on radio frequencies that do not require a license. You buy a tiny device, you stick a battery in it, and that device can send a short message to a base station ten kilometers away. The battery can last **ten years** because the device sleeps almost all the time and only wakes up for a few seconds to send a tiny burp of data.

The trick is something called **chirp spread spectrum**. Instead of sending a clean tone like "beep," LoRa sends a tone that slides up or down across a band of frequencies, like the noise a science-fiction laser makes — "pew-eeeeooooo" or "wooooooop." This sliding tone is incredibly easy for a receiver to dig out of noise, even when the signal is *below* the noise floor. That is why LoRa can hear a signal that is so faint a regular radio would think nothing was there at all.

```
A boring radio tone (FSK):
   *********    *********    *********
   |       |    |       |    |       |
   |       |    |       |    |       |
freq -------------------- time -->

A LoRa chirp:
       /        /        /
      /        /        /
     /        /        /         <- frequency rising over time
    /        /        /
   /        /        /
freq -------------------- time -->
```

Because the chirp spans a wide frequency range, even if part of the band is full of noise, the receiver still catches the rest of the chirp and can reconstruct the message.

### LoRa vs LoRaWAN — the difference matters

**LoRa** is the radio modulation. It is the physical layer. It is the actual chirping noise. LoRa, by itself, just gets bits from one antenna to another. It does not say *who* the bits are for, *what they mean*, or *what to do with them*.

**LoRaWAN** is the network on top of LoRa. It is the rules: how a device joins the network, how messages are addressed, how encryption works, how acknowledgments work, how downlinks (messages from network to device) are scheduled. LoRaWAN is what you build a real product on. Plain LoRa by itself is just two radios pinging each other.

Think of LoRa as the engine and LoRaWAN as the car. The engine alone is not transportation. The car (with engine, wheels, brakes, steering, seats) is transportation.

### Frequency bands (where in the radio dial)

LoRa lives in the ISM (industrial, scientific, medical) bands, which are unlicensed. You do not need permission from the government. The catch is that ISM bands have **duty cycle limits** in some countries — you can only transmit for a tiny fraction of each hour, to avoid hogging the airwaves.

- **EU 868 MHz** — Europe. 1% duty cycle in most sub-bands. So in any one hour you can transmit for 36 seconds, then must shut up for 24 minutes.
- **US 915 MHz** — Americas. No duty cycle limit, but a frequency-hopping requirement instead.
- **AS 433 / 470 / 915 MHz** — varies by country in Asia.
- **AU 915 MHz** — Australia.

If you are designing a global product, you have to build for multiple regions. The radio chip is the same. Just the firmware tunes to a different band.

### A LoRa range story to give you a feel

To give you a sense of how absurd the range can be: in 2019, a high-altitude balloon experiment in Australia pinged a LoRa packet **832 kilometers** from the balloon down to a ground station. That is not a typo. Eight hundred and thirty-two kilometers, with a tiny coin-cell-powered transmitter and a single chip. Of course, that requires line of sight and an unobstructed path through the atmosphere; in a dense city with concrete buildings, you might only get one to three kilometers. The point is the *fundamental capability* is huge.

For real deployments: budget two to five kilometers in a dense city, five to fifteen kilometers in a town with mixed buildings, and twenty to fifty kilometers in flat rural terrain with elevated antennas. If you are doing something like cattle tracking on a ranch, one gateway on a windmill can cover a whole farm. If you are doing parking-spot sensors in a downtown core, you may need a gateway every few blocks.

### Spreading factor — the speed/distance dial

Inside LoRa there is a parameter called the **spreading factor (SF)**, ranging from SF7 to SF12. A high SF means the chirp is stretched out over a longer time, which makes it easier to hear at long range, but each packet takes longer on the air. A low SF means short, fast packets that do not carry as far.

- **SF7** — fastest, shortest range, ~0.4 seconds per small packet.
- **SF12** — slowest, longest range, ~3 seconds per small packet, ~10x more battery drain per packet.

ADR (Adaptive Data Rate) is the network server feature that monitors how cleanly your gateway is hearing each device and tells weak ones to bump up SF and strong ones to drop down. You enable ADR and forget about it; the network tunes each device for you.

## LoRaWAN Classes A/B/C

Three flavors of LoRaWAN device exist, called Class A, Class B, and Class C. They differ in how *often* the device listens for messages from the network. Listening uses power. More listening = more power. So you pick the class that fits how chatty your application is.

### Class A — the sleepy one (default, lowest power)

The device wakes up only when it has something to say. It transmits its message ("temperature is 18 °C"), then opens **two short receive windows** — one short window 1 second after it transmitted, then a second slightly longer window 2 seconds after. If the network has nothing to say back, both windows close empty. The device goes back to sleep until next time. That is it. The network *cannot* talk to a Class A device unless the device just talked first.

```
Class A timeline (one cycle):

device:   [TX]------>[RX1][RX2]----------(sleep for an hour)----------[TX]----...
                      ^    ^
                      |    |
                  short receive windows after every uplink

network: ---only chance to send a downlink is in those two short slots---
```

This is why Class A devices last a decade on a battery. They sleep 99.99% of the time.

### Class B — the scheduled one (medium power)

The device opens extra receive windows on a fixed schedule, synchronized to a beacon transmitted by the gateway every 128 seconds. So instead of only getting downlinks after an uplink, the device can get a downlink at any of its scheduled "ping slots." This is useful if the network needs to push a command at a roughly predictable time without waiting for the device to send something first. Battery life takes a hit but is still measured in years.

### Class C — the always-listening one (highest power)

The device listens **all the time** except when it is transmitting. The network can send a downlink whenever it wants and the device will hear it almost instantly. Battery life is now measured in days or weeks, so Class C is for mains-powered devices — actuators, valves, things plugged into the wall. You would never run a leaf-soil-moisture sensor as Class C.

### Picking a class — a simple rule

- **Sensor that reports occasionally and never receives commands?** Class A.
- **Sensor that needs commands within a couple of minutes?** Class B.
- **Actuator plugged into the wall that must respond instantly?** Class C.

### A battery-life calculation walkthrough, in plain English

Imagine a Class A LoRa soil moisture sensor with a 2400 mAh AA-sized lithium battery. It transmits one short packet every 30 minutes. Each transmission, including the two short receive windows after, drains roughly 0.1 mAh. So per day: 48 transmissions × 0.1 mAh = 4.8 mAh per day. Plus a tiny sleep current of about 0.005 mA × 24 hours = 0.12 mAh per day. Total: roughly 5 mAh per day. With 2400 mAh available, that is 2400 / 5 = **480 days, or 1.3 years**.

Now switch that to Class C — always listening. The receive current is ~10 mA. Even with no transmissions, just listening, that burns 240 mAh per day. The same battery dies in **10 days**. That is the difference Class A makes. You buy a decade of life by sleeping.

If you want even longer battery life on Class A: report less often. Report once an hour instead of twice — battery doubles. Report once every six hours — battery sextuples. Many real-world LoRa sensors report once or twice a day and last *over a decade* on a primary lithium D-cell.

## LoRaWAN Network Architecture (gateway, network server, app server, join server)

A LoRaWAN system has more moving parts than a Zigbee or Z-Wave system. It is not a mesh — it is a **star of stars**. Many devices talk to many gateways, all of which forward to a single central brain.

```
+---------+     +---------+     +---------+     +---------+
| Sensor  |     | Sensor  |     | Sensor  |     | Sensor  |   <- end devices
+---------+     +---------+     +---------+     +---------+   (Class A/B/C)
     \              |               |              /
      \             |               |             /
       \            |               |            /
        \           |               |           /
         v          v               v          v
       +---------------+        +---------------+
       |   Gateway A   |        |   Gateway B   |   <- receive ALL packets
       +---------------+        +---------------+      they hear, dumb forwarders
              |                         |
              +-------------------------+
                          |
                    (Internet, IP backhaul)
                          |
                          v
                +---------------------+
                |   Network Server    |   <- deduplicates, decrypts MAC layer,
                |        (NS)         |      manages ADR, routes uplinks
                +---------------------+
                          |
                          +-----------------------+
                          |                       |
                          v                       v
                +-----------------+      +-----------------+
                |  Join Server    |      |  Application    |
                |     (JS)        |      |  Server (AS)    |
                +-----------------+      +-----------------+
                  handles OTAA              actual app logic,
                  device join / keys        decrypts payload,
                                            speaks to your DB / dashboard
```

### What each piece does

- **End device** — the sensor or actuator. Battery, radio, microcontroller. Cheap. Dumb. Knows only its own keys and how to chirp.

- **Gateway** — a box on a roof, on a tower, on a windowsill. It hears every LoRa packet within range (could be many kilometers) and forwards it over the Internet (Wi-Fi, Ethernet, cellular) to the network server. It is dumb on purpose: it does *not* decrypt, it does *not* address, it just forwards everything it hears with a timestamp and signal-strength reading. Multiple gateways will hear the same packet from a device — that is fine, they all forward, and the network server deduplicates.

- **Network Server (NS)** — the brain. It receives packet copies from many gateways, picks the best copy (highest signal), checks the MAC-layer message integrity code (MIC), decrypts the MAC header, runs ADR (Adaptive Data Rate, which tells weak devices to spread harder and strong devices to spread less so they save battery), and forwards the still-encrypted application payload to the application server.

- **Join Server (JS)** — when a new device boots up and wants to join the network using OTAA (Over-the-Air Activation), the join server does the cryptographic handshake. It holds the device's root keys (AppKey) and derives session keys (NwkSKey for the network server, AppSKey for the application server) on the fly. Splitting the join server out from the network server is a security feature — your network operator never sees your application keys.

- **Application Server (AS)** — your code. It decrypts the payload with AppSKey, parses it ("ah, this is bytes 0–1 = temperature in tenths of a degree Celsius, byte 2 = battery percentage"), and shoves the data into your database, dashboard, alerting system, whatever.

The key insight: a LoRaWAN device does not pair with a gateway. It just chirps. Whichever gateways are within earshot all hear it, all forward it, and the server figures out who said what. Add more gateways, you get more coverage, with no configuration of devices needed.

### Public networks vs private networks

There are two ways to deploy LoRaWAN. **Public** networks are run by an operator like The Things Network (a community-run, free, best-effort network with thousands of volunteer gateways), Helium (a token-incentivized network where individuals run gateways and earn tokens), or commercial operators like Senet or Actility. You buy or build a device, you register it on the operator's portal, and you pay (or do not, in TTN's case) per device per month. You do not own the gateways — they are someone else's.

**Private** networks are ones you build yourself. You buy a few gateways, you run a network server (open-source ChirpStack is the most popular, or Things Stack Open Source), and the data never leaves your premises. This is the choice for industrial sites, agriculture operations, large warehouses, mines, and anywhere data privacy matters or coverage is too sparse to rely on a public operator.

You can mix. Many real deployments are private gateways feeding a public network server, or public gateways feeding a private network server, or a hybrid where some devices roam between networks.

### OTAA vs ABP — two ways a device joins

**OTAA (Over-the-Air Activation)** is the modern, secure way. The device boots up with a root key (AppKey) burned into it at the factory. On first power-up, it sends a Join Request, the join server cryptographically negotiates a session, and fresh session keys are minted. Every time the device reboots or rejoins, fresh session keys are minted again. If a device is stolen and someone reads its flash, they get only the root key for *that one device*, not the network's master key.

**ABP (Activation By Personalization)** is the older, simpler way. The session keys are burned into the device at the factory, and the device just *starts using them* with no handshake. Faster, simpler, but less secure — if a device is compromised, the same session keys live forever. Use OTAA unless you have a very specific reason not to.

## Sigfox (alternative LPWAN, France-based)

Sigfox is the French cousin of LoRaWAN. It does roughly the same thing — long-range, low-power, license-free, tiny messages, decade-long batteries — but with **even tinier messages and a closed network**.

A Sigfox device sends **12-byte uplinks**. Twelve bytes. That is it. You get 140 of them per day, max. Downlinks are 8 bytes and you get **4 per day**. That is barely enough to say "I'm fine" once every ten minutes.

The catch: there is **only one Sigfox network operator in each country**. You cannot stand up your own Sigfox network the way you can with LoRaWAN. You buy a Sigfox-certified device, you buy a subscription from your country's Sigfox operator, and the data shows up in your back-end via a Sigfox cloud callback. Your device is not joining your gateway; your device is joining *Sigfox the company*.

This makes Sigfox simpler to deploy (no gateways to install) but riskier as a long-term bet (if the operator goes under or changes pricing, you have a paperweight). The original Sigfox SAS company in France went into receivership in 2022 and was bought by UnaBiz, so the network continues, but it was a wake-up call. LoRaWAN, where you can run your own network, has been winning the larger ecosystem battle.

### Why a 12-byte limit?

Sigfox uses **ultra-narrowband** modulation — about 100 Hz wide signals. That is incredibly skinny. A signal that skinny goes very, very far on very, very little power, but it cannot carry much data per second. So you trade throughput for range. Twelve bytes is what you get out of one 100 Hz signal in a fraction of a second, repeated three times for redundancy on different frequencies.

The Sigfox philosophy: most things in the field do not need to say much. A water meter only needs to send "I have used N liters since yesterday." A pet collar only needs to send "I am at GPS coordinates X,Y." A pallet sensor only needs to send "I am still here, I am still cold enough." Twelve bytes is plenty for any of those.

### Why three transmissions on three frequencies?

Sigfox devices send each message **three times** on **three different sub-frequencies** within the band. They are tiny chunks of bandwidth and there is no acknowledgment for uplinks (in Class 0U devices), so redundancy is the safety net. If one of the three transmissions collides with another device or hits noise, two more chances exist for the message to land at any of the gateways within earshot. Multiple gateways forward to the back-end and dedup happens server-side, just like LoRaWAN.

The practical consequence: a Sigfox uplink takes roughly 6 seconds of total airtime (three transmissions across three frequencies, each lasting about 2 seconds at 100 bps). This is why the daily limit of 140 uplinks is what it is — it is the duty-cycle limit math working backwards.

## NB-IoT (3GPP cellular IoT, low-power LTE)

NB-IoT means **Narrowband IoT**. It is the cellular industry's answer to LoRa and Sigfox. Instead of using unlicensed ISM bands, NB-IoT lives **inside the licensed cellular spectrum your phone uses** — slid into either a tiny 200 kHz slice next to LTE, inside an LTE channel's guard band, or even reusing one resource block of an LTE carrier.

The pitch: you already have a cellular operator. Cellular coverage is everywhere humans live. So instead of putting up your own LoRaWAN gateways or trusting a single Sigfox operator, you let the existing cell towers carry your tiny IoT messages. Same towers. Same antennas. Just a different, very-low-bandwidth radio mode that the tower runs on the side.

NB-IoT throughput is around **20–60 kbps** (compared to LTE's many megabits per second). Latency is several seconds, sometimes ten or more — totally fine for a parking sensor saying "spot 47 is occupied" but useless for a video call. Devices use **Power Saving Mode (PSM)** and **Extended Discontinuous Reception (eDRX)** to sleep for hours between check-ins, just like LoRaWAN Class A.

### The penetration superpower

The big trick of NB-IoT is **deep coverage**. The signal is narrow and slow, which makes it incredibly easy to decode at very low power. NB-IoT can reach **20 dB more link budget** than regular LTE. That means a device buried in a basement, behind concrete walls, in a metal cabinet, at the bottom of a manhole — places where your phone has zero bars — can still send NB-IoT packets. Water meters, gas meters, parking spot sensors, manhole-cover sensors are the canonical examples.

### The tradeoff

You are now paying a cell carrier. Every device has a SIM (or eSIM, or iSIM). Every device has a monthly fee. The fees are tiny — pennies per device per month — but it is not zero, and it is not in your control the way a private LoRaWAN network is. If your carrier raises prices or sunsets the band, you are stuck.

### PSM and eDRX — the two sleep modes

**PSM (Power Saving Mode)** is the deep-sleep mode. The device tells the network "I am going to sleep for the next four hours, do not bother trying to reach me until then." The network agrees, marks the device unreachable, and queues any downlinks. The device shuts down its radio almost completely — pulling only microamps from the battery — and wakes up on its own schedule. When it wakes, it checks for pending downlinks, sends its uplink, and goes back to sleep. Battery life: a decade is realistic.

**eDRX (Extended Discontinuous Reception)** is the lighter-sleep mode. Instead of going completely dark, the device opens listening windows on a schedule that can be tens of seconds, minutes, or up to ~44 minutes apart. The network can reach the device with bounded latency. Battery life is shorter than PSM but far longer than always-on LTE.

Most NB-IoT meters use PSM and check in once or twice a day. Most asset trackers use eDRX with a window every few minutes so they can be commanded to "ping now" when needed.

## LTE-M / Cat-M1 (cellular IoT with more bandwidth)

LTE-M (also called Cat-M1) is NB-IoT's chubbier sibling. Same idea — cellular IoT, licensed spectrum, low power — but with **more bandwidth and lower latency** at the cost of a bit more power use.

Numbers (rough):
- **Throughput:** up to ~1 Mbps (vs NB-IoT's ~60 kbps).
- **Latency:** tens of milliseconds (vs NB-IoT's seconds).
- **Voice over LTE (VoLTE):** supported. Yes, you can do voice calls over LTE-M, which is why it is used in things like elderly-fall-detection pendants and connected fire alarms.
- **Mobility:** supports handover between cells, so the device can move (in a car, on a person). NB-IoT often does *not* — it assumes you stuck the device in one place forever.

Pick LTE-M when you need a phone-call-able device, or one that moves, or one that needs a firmware update over the air without making the user wait three days. Pick NB-IoT when the device is buried, stationary, and only needs to whisper a number once an hour.

### A real-world example: the connected pet door

A connected pet door (yes, this exists, and it is great) needs to: detect a microchip in the cat collar, decide if this is "your" cat or the neighbor's stray, unlock the door, log the entry, send a notification to the owner, and accept commands like "lock for the night, do not let anyone in until 7 AM." It is mains-powered (or has a big internal battery). It is mounted on an exterior door. It needs near-instant downlinks ("lock now, that raccoon is coming back"). It needs to support firmware updates so new ML models can be downloaded.

LTE-M is the right radio. Wi-Fi would also work *if* the door is close to a router, but the latency requirements and update size make LTE-M's combination of low-latency, decent throughput, and deep coverage attractive. Plus the device needs to keep working if the homeowner's Wi-Fi is down.

A connected gas meter, by contrast, is the wrong fit for LTE-M. It sits in a basement, behind concrete walls, transmits one number per day, and never moves. NB-IoT's deep penetration and pennies-per-month cost are a perfect match.

## 5G IoT (massive Machine Type Communication, mMTC)

5G is not just one thing. It is **three usage modes** that share the same radio:

- **eMBB** — enhanced Mobile Broadband. Your phone. Fast video. The thing the marketing posters were selling.
- **URLLC** — Ultra-Reliable Low-Latency Communications. For factory robots and self-driving cars. A few milliseconds of latency, a 1-in-a-million packet loss.
- **mMTC** — massive Machine Type Communication. The IoT mode. **Up to 1 million devices per square kilometer**, low data rate, low power, similar in spirit to NB-IoT but baked in from the start of the 5G design.

Not all 5G networks have all three modes turned on yet. As of 2026, eMBB is widely deployed, URLLC is rolling out in industrial and stadium scenarios, and mMTC is being layered in. For most IoT teams shipping today, NB-IoT and LTE-M (which are technically 4G but supported on 5G networks) are still the defaults — they ride on 5G base stations seamlessly. 5G-native mMTC features like RedCap (Reduced Capability) are starting to appear in chipsets but are not yet ubiquitous.

The big future picture: as 5G mMTC matures, you will get richer power-saving features, better positioning (the cell network can locate your device to within meters, no GPS needed), and direct device-to-satellite fallback through Non-Terrestrial Networks (NTN). Your sensor in the middle of an ocean, a desert, or a mountain range can talk to a low-Earth-orbit satellite using the same radio protocol as a sensor in downtown Tokyo.

### Network slicing — a tiny mention

A 5G concept worth knowing, even at this level: **network slicing**. The same physical 5G network can be divided into virtual "slices" with different guarantees. One slice is mMTC for a million parking sensors. Another slice is URLLC for a port crane that must respond in 5 ms. Another slice is eMBB for stadium spectators streaming highlights. Each slice gets a chunk of the radio resources reserved with its own performance promises. From the device's point of view it just connects, but the network knows which slice the SIM belongs to and routes accordingly. This is how 5G plans to keep your IoT fleet from being elbowed off the air by Friday-night Netflix users.

### RedCap — the bridge

NB-IoT and LTE-M are technically 4G technologies that ride on 5G base stations. Pure 5G IoT chipsets are starting to appear under the name **RedCap (Reduced Capability)**. Think of it as "LTE-M reborn natively in 5G." Lower complexity than full 5G, higher throughput than NB-IoT, designed for wearables and industrial sensors. Adoption is just starting in 2026 — most teams should still pick NB-IoT or LTE-M today, but watch RedCap if you are designing a product that ships in 2027 or later.

## 6LoWPAN (IPv6 over Low-Power WPAN, RFC 6282 compression)

6LoWPAN is a long, ugly name that means "IPv6 over Low-power Wireless Personal Area Networks." The big idea: **let tiny battery-powered radios speak the same Internet language as your laptop**. Specifically, IPv6.

Why is this hard? An IPv6 packet has a **40-byte header**. IEEE 802.15.4 (the radio that Zigbee and Thread sit on) has a maximum frame size of **127 bytes**. So if you sent a raw IPv6 packet, the header alone would eat one third of every frame, leaving almost no room for actual data. You'd burn battery sending headers all day.

6LoWPAN's solution: an **adaptation layer** that compresses the IPv6 header into something tiny when possible.

### Header compression visualization

A normal IPv6 packet header on the wire:

```
+--------+--------+--------+---------+--------+--------+--------+--------+
| Version (4 bits) | Traffic Class (8 bits) | Flow Label (20 bits)        |
+--------+--------+--------+--------+--------+--------+--------+--------+
| Payload Length (16 bits)          | Next Header (8) | Hop Limit (8)    |
+--------+--------+--------+--------+--------+--------+--------+--------+
|                                                                       |
|                  Source Address (128 bits = 16 bytes)                 |
|                                                                       |
+-----------------------------------------------------------------------+
|                                                                       |
|              Destination Address (128 bits = 16 bytes)                |
|                                                                       |
+-----------------------------------------------------------------------+
                                                  TOTAL: 40 bytes
```

After 6LoWPAN compression (best case):

```
+----------+--------+
| LOWPAN_  | Hops   |   <- LOWPAN_IPHC dispatch + 1-byte hop limit
| IPHC (1) | (1)    |
+----------+--------+
                       TOTAL: 2 bytes (when source/dest are link-local
                       and derivable from the 802.15.4 MAC addresses)
```

Wait, what? **Forty bytes squeezed into two?** Yes — it works because most fields in a normal IPv6 header are predictable in a 6LoWPAN context:

- **Version** is always 6, no need to send it.
- **Traffic class and flow label** are usually zero, no need to send them.
- **Payload length** is already in the 802.15.4 header, no need to repeat.
- **Next header** is often UDP and follows a known pattern, can be elided.
- **Hop limit** is 1 byte if needed.
- **Source address** can often be derived from the 802.15.4 source MAC address — if so, do not send it.
- **Destination address** can often be derived too.

The **LOWPAN_IPHC** dispatch byte tells the receiver which fields were elided and how to reconstruct them. If you cannot elide everything, the dispatch byte says "I elided this and that, here is the rest inline." Worst case is still smaller than uncompressed.

There is also **fragmentation**: if a packet really must be longer than 127 bytes (because it has actual data and even after compression overflows), 6LoWPAN can split it into multiple 802.15.4 frames and reassemble at the other end.

### Why this matters

Because of 6LoWPAN, a Thread bulb and a laptop and a server in a data center are all just IPv6 endpoints. The bulb has a real IPv6 address. You can ping it (well, the firewall will probably block you, but you *could*). The bulb does not need a translation gateway turning Zigbee-flavored data into IP-flavored data. Thread, Matter-over-Thread, and any sensor network built on RFC 6282 are all just IP devices. This is why Matter could exist — Thread already did the hard work of giving every bulb a real address.

### A worked compression example

Suppose a Thread bulb wants to send a UDP packet to its border router asking "what time is it?" over NTP. Without 6LoWPAN, the headers look like this:

```
IPv6 header:    40 bytes (most of it predictable)
UDP header:      8 bytes (most of it predictable)
NTP request:    48 bytes (the actual content)
                --
Total:          96 bytes
```

In an 802.15.4 frame with 127 bytes payload, that leaves 31 bytes of slack — plenty. But if the request had been a slightly larger DNS query or a CoAP message with options, we would have run out of room.

With 6LoWPAN compression:

```
LOWPAN_IPHC:    2 bytes (most IPv6 fields elided, derived from MAC)
LOWPAN_NHC for UDP: 1 byte + 2-byte port pair (compressed)
NTP request:    48 bytes
                --
Total:          53 bytes
```

We saved 43 bytes. That is a 45% reduction. We now have plenty of headroom for a CoAP wrapper around the NTP request, or for the response, or for routing headers if we need them. Headroom is everything in a 127-byte world.

### Mesh-Under vs Route-Over

Two ways a 6LoWPAN mesh can route packets between hops. **Mesh-Under** does the routing at layer 2 (below IP), so all the hops within the mesh share one IPv6 hop. **Route-Over** does it at layer 3 (IP), with each hop being a separate IP hop. Thread uses Route-Over, which is cleaner for IP folks to reason about — every device is its own IP node.

This matters because Route-Over networks can use standard IP tools to debug. You can traceroute through a Thread mesh. You can ping the third-hop bulb. Mesh-Under hides the mesh from IP, which can be simpler for some applications but harder to debug.

## Comparison Table (BLE / Zigbee / Z-Wave / Thread / WiFi / LoRaWAN / NB-IoT / LTE-M on range/throughput/power/ecosystem/license axes)

This table is the cheat-sheet of cheat-sheets. Print it, tape it next to your monitor.

| Tech         | Typical Range          | Throughput          | Power               | Topology      | Spectrum            | License                | Best For                                |
|--------------|------------------------|---------------------|---------------------|---------------|---------------------|------------------------|-----------------------------------------|
| **BLE**      | 10–100 m               | 1–2 Mbps (LE 2M)    | very low            | star, mesh    | 2.4 GHz ISM         | unlicensed             | wearables, beacons, phone-to-thing      |
| **Zigbee**   | 10–100 m / mesh wider  | 250 kbps            | very low            | mesh          | 2.4 GHz ISM         | unlicensed             | smart home (legacy), Philips Hue        |
| **Z-Wave**   | 30–100 m / mesh wider  | 9.6/40/100 kbps     | very low            | mesh          | sub-GHz (908/868)   | unlicensed             | smart home (legacy), interop ecosystem  |
| **Thread**   | 10–100 m / mesh wider  | 250 kbps            | very low            | mesh, IPv6    | 2.4 GHz ISM         | unlicensed             | Matter devices, smart home (modern)     |
| **Wi-Fi**    | 30–100 m               | up to multi-Gbps    | high (mains usual)  | star (AP)     | 2.4/5/6 GHz ISM     | unlicensed             | cameras, streaming, anything mains-powered |
| **LoRaWAN**  | 2–15 km urban / 50 km rural | 0.3–50 kbps    | extremely low       | star of stars | sub-GHz ISM (868/915)| unlicensed (duty cycle)| asset tracking, agriculture, smart city |
| **Sigfox**   | 10 km urban / 40 km rural | 100 bps        | extremely low       | star          | sub-GHz ISM         | unlicensed (operator)  | simple sensors, asset tracking          |
| **NB-IoT**   | several km / deep indoor | 20–60 kbps        | extremely low       | star (cellular)| licensed LTE bands | licensed (carrier fee) | meters, manhole sensors, parking         |
| **LTE-M**    | several km             | up to ~1 Mbps       | low                 | star (cellular)| licensed LTE bands | licensed (carrier fee) | mobile assets, alarms, voice IoT        |
| **5G mMTC**  | several km             | scales w/ mode      | extremely low       | star (cellular)| licensed 5G       | licensed (carrier fee) | future massive deployments              |

### How to read this table

If your device is **mains-powered, indoors, and needs lots of bandwidth** — Wi-Fi.

If your device is **battery-powered, indoors, and part of a smart-home ecosystem** — Thread (preferred for new builds), Zigbee, or Z-Wave (legacy choices).

If your device is **a wearable or paired with a phone** — BLE.

If your device is **outdoors, miles from anything, and just needs to whisper a number** — LoRaWAN (private), Sigfox (operator, simpler), or NB-IoT (carrier, deepest indoor coverage).

If your device is **mobile, voice-capable, or needs occasional firmware updates** — LTE-M.

If your device is **a million-strong fleet you want to deploy in a stadium or factory** — 5G mMTC, eventually.

### The license question, in plain English

"License-free" or "unlicensed" means you can transmit on those frequencies without asking the government, **as long as you obey the rules** for that band — typically a power limit and a duty-cycle limit. You do not pay anyone. You do not get a guarantee that the band will not be crowded.

"Licensed" means a carrier paid the government a lot of money for exclusive use of the band. They run their network on it. You pay them to use it. You get a guarantee of clean spectrum and continent-scale coverage. You also get a monthly bill per device.

Both models work. Which one fits depends on whether you want capex (build your own) or opex (rent someone else's). LoRaWAN is the canonical capex play; NB-IoT and LTE-M are the canonical opex play.

### A topology cheat sheet

It is worth holding all five topologies in your head at once because the same words mean slightly different things in different protocols.

```
Star:        every device talks to one central hub
              D       D       D
               \      |      /
                \     |     /
                 +----H----+
                 (Wi-Fi AP, Sigfox tower, NB-IoT cell)

Star of stars: many devices to many gateways, all to one server
              D    D    D    D    D
               \   |    |    |   /
                G----G------G----G   <- gateways are peers
                 \   |     |    /
                  \  |     |   /
                   +-Server-+
                   (LoRaWAN)

Mesh:        every device can relay for every other device
              D----D----D
              |    |    |
              D----D----D
              |    |    |
              D----D----D
              (Zigbee, Thread, Z-Wave, BLE Mesh)

Tree:        hierarchical, root with branches
                     R
                    / \
                   N   N
                  /|   |\
                 D D   D D
                 (older Zigbee profiles, some industrial WSNs)

Point-to-point: one to one
              D --------- D
              (BLE pairing, classic Bluetooth)
```

Most modern IoT settles on either **mesh** (smart home, indoor sensor networks) or **star of stars** (long-range outdoor sensor networks). Pure star is reserved for things where the device must be directly attached to infrastructure: cellular, Wi-Fi.

### Memory cheat — when to pick what, in one paragraph

In your house: **Matter over Thread for new devices, Zigbee or Z-Wave for legacy ecosystems, Wi-Fi for cameras and streaming, BLE for wearables**. On a farm or city: **LoRaWAN if you want to own the network, NB-IoT or LTE-M if you want a carrier to handle it**. In a factory: **Wi-Fi 6E or 5G URLLC for real-time control, LoRaWAN or LTE-M for asset tracking**. On the move (vehicles, pets, people): **LTE-M with NB-IoT fallback, GPS for location, BLE for short-range pairing**. That covers 90% of decisions you will ever face.

### One last warning about marketing brochures

Vendor brochures will tell you their radio gets "10 km range" or "10-year battery." Both numbers are true *under specific conditions*. The 10 km is line-of-sight on a flat plain with no buildings; in your real city, divide by five. The 10-year battery is at one transmission per day with the smallest possible payload; if you transmit hourly with larger payloads, divide by twelve. Always read the spec sheet's footnote that defines the test conditions, then map those conditions to your actual deployment. Pilot a handful of devices on the real site for a month before committing to ten thousand units. Reality is always lossier than the brochure.

## CoAP (Constrained Application Protocol — UDP-based REST for tiny devices, RFC 7252)

CoAP is HTTP for things that are too small to do HTTP.

If you have ever used a website, you have used HTTP without knowing it. When you click a link, your browser sends an HTTP message to a server. The server sends an HTTP message back. The message is full of headers and text and curly braces and JSON and all kinds of stuff. HTTP is friendly to humans because humans can read it. You can open a tool called `curl` and type an HTTP request by hand and read the answer.

But HTTP is fat. A single HTTP request can easily be 500 bytes just for the headers. Add some JSON and you are at 2 kilobytes. For a laptop with gigabytes of memory and a fast Wi-Fi connection, 2 kilobytes is nothing. For a tiny battery-powered sensor that has 32 kilobytes of RAM total and is sending data over a slow Zigbee mesh, 2 kilobytes is a disaster. The sensor cannot fit a full HTTP message in memory all at once. The Zigbee radio cannot send 2 kilobytes without splitting it into many tiny packets, and every packet costs battery.

So smart people sat down and asked: what if we made HTTP smaller? Like, *much* smaller? Like, what if instead of headers being words like `Content-Type: application/json`, headers were just numbers? And what if we ran on UDP instead of TCP, so we did not have to do a three-way handshake every time? And what if the whole thing fit in one packet for most messages?

That is CoAP. CoAP stands for Constrained Application Protocol. "Constrained" means "for things with very little of everything." Little memory, little CPU, little battery, little bandwidth. Constrained.

### What CoAP looks like

A CoAP request is shaped like an HTTP request. There is a method (GET, POST, PUT, DELETE — same words as HTTP). There is a URI (`coap://sensor.local/temperature`). There is a payload. There is a response code (2.05 means "OK, here is the data"; 4.04 means "Not Found", same as HTTP 404 just written differently).

But the bytes on the wire are tiny. A typical CoAP GET request is around 10 to 20 bytes. A typical response is around 30 bytes (including the temperature reading). The whole conversation fits in two UDP packets and finishes in a few milliseconds.

```
CoAP request flow (single GET, no security)
==========================================

Client                                         Server
  |                                              |
  |  GET coap://server/temp  (Token: 0xA1)      |
  |--------------------------------------------->|
  |  [4 bytes header + 4 bytes URI + token]     |
  |                                              |
  |                                              | look up temp
  |                                              | temp = 21
  |                                              |
  |  2.05 Content  (Token: 0xA1)  payload="21"  |
  |<---------------------------------------------|
  |                                              |
```

The Token is how the client matches a response to a request. UDP does not have ordering, so two requests sent at the same time might come back in either order. The token is a label. Whichever response comes back with token `0xA1` belongs to whichever request used token `0xA1`.

### Confirmable vs non-confirmable

UDP packets can get lost. If a client sends a CoAP request and the packet vanishes into the ether, the client will wait forever for a response that never comes. CoAP solves this with two message types:

**Confirmable (CON)** — the client says "I want an acknowledgment." If the server gets the request, it sends back an ACK. If the client does not see an ACK within a timeout (a few seconds), the client retransmits. This is the equivalent of TCP reliability, bolted on top of UDP, but only when you ask for it.

**Non-confirmable (NON)** — fire and forget. The client sends the request and does not care if it arrives. Used for sensor data where the next reading is coming in 10 seconds anyway, so a missed reading is not a big deal.

You pick which mode based on whether you can tolerate loss. A "turn off the lights" command should be CON. A "current temperature is 21 degrees" notification can be NON.

### Observe — push without polling

A regular HTTP-style protocol is pull-based. The client says "give me the temperature." The server answers. If the client wants new readings every minute, the client has to ask every minute. That is a lot of asking.

CoAP has an extension called **Observe** (RFC 7641). The client says "give me the temperature, *and tell me whenever it changes*." The server remembers the client. Whenever the temperature changes, the server sends a new response on its own — without being asked. The client gets a stream of updates as long as it stays interested.

This is huge for IoT because the device that has the data (the sensor) is usually the one running on battery. If the sensor has to wake up every time a client polls, that is a lot of wake-ups. With Observe, the sensor only wakes when its data changes. The client gets a notification, the sensor goes back to sleep. Way less battery.

### CoAP over DTLS

CoAP runs on UDP, so it cannot use TLS (TLS needs TCP). Instead it uses **DTLS** — Datagram TLS — which is TLS adapted for UDP. DTLS does the same job: encrypts the packets, authenticates the server, sometimes authenticates the client. It just works on top of UDP instead of TCP. We will talk more about DTLS in the security section below.

### Where CoAP shows up

- **Smart home sensors over Thread** — Thread devices love CoAP because both are designed for tiny constrained nodes. Matter actually uses CoAP under the hood for some operations (commissioning).
- **Industrial sensors** — temperature, pressure, vibration sensors that report up a mesh and into a gateway.
- **LoRaWAN application servers** — sometimes the application protocol on top of LoRaWAN is CoAP.

If you ever see a `coap://` or `coaps://` URL in a device's docs, that is CoAP. The `coaps://` version is the encrypted one (DTLS).

## MQTT (Message Queuing Telemetry Transport — pub/sub broker)

MQTT is the most common IoT protocol on planet Earth. If a "smart" device talks to a cloud, there is a very good chance it talks MQTT.

MQTT was invented in 1999 by IBM for oil pipelines. Yes, oil pipelines. The problem was: you have sensors all over a pipeline that runs through the desert. The sensors talk over satellite links that are slow, expensive, and unreliable. You need to send tiny status messages back to a control center. HTTP is way too heavy. So IBM built MQTT.

The name "Message Queuing Telemetry Transport" is a mouthful and most people don't even know what the letters stand for. Just say "MQTT" — it is pronounced like the letters: em-cue-tee-tee.

### Pub/sub — the big idea

MQTT does not work like HTTP. There is no "client asks server, server answers." Instead, there is a thing in the middle called a **broker**. Every device connects to the broker. Every device either **publishes** messages, or **subscribes** to messages, or both.

A publish looks like: "hey broker, I have a message about the topic `home/livingroom/temperature` and the value is `21`."

A subscribe looks like: "hey broker, whenever anyone publishes anything to `home/livingroom/temperature`, please send it to me."

The broker keeps a list of who is subscribed to what. When a message comes in for a topic, the broker forwards it to every subscriber for that topic. The publisher does not know or care who the subscribers are. The subscribers do not know or care who the publisher is. They both just talk to the broker.

This is called **pub/sub** (publish/subscribe). It is a fundamentally different shape from HTTP's request/response.

```
MQTT broker pub/sub topology
=============================

  +-------------+
  | Sensor A    |   publish "home/temp" = 21
  | (publisher) |--------------------+
  +-------------+                    |
                                     v
  +-------------+              +----------+              +-------------+
  | Sensor B    |  publish     |  BROKER  |   forward    | Phone App   |
  | (publisher) |------------->|          |------------->| (subscriber)|
  +-------------+              |  (Mosq.  |              +-------------+
                               |   EMQ X, |
                               |   AWS    |              +-------------+
                               |   etc.)  |--forward---->| Cloud Logger|
                               +----------+              | (subscriber)|
                                     ^                   +-------------+
                                     |
  +-------------+                    |
  | Light Bulb  |  subscribe "home/light/+"
  | (subscriber)|--------------------+
  +-------------+
```

The publisher sends one message. The broker fans it out to all matching subscribers. If there are zero subscribers, the message just gets dropped (with one exception, see "retained messages" below).

### Topics

A topic is a slash-separated string. There is no global registry — you make up your own topic names. Conventions:

- `home/livingroom/temperature`
- `factory/line3/machine7/vibration`
- `device/abc123/status`

Topics support **wildcards** when subscribing:

- `+` matches one level. `home/+/temperature` matches `home/livingroom/temperature` and `home/kitchen/temperature` but not `home/upstairs/bedroom/temperature`.
- `#` matches everything below. `home/#` matches every topic that starts with `home/`. Must be at the end.

Wildcards are how a logger app says "send me everything" without having to enumerate every topic.

### QoS — Quality of Service

MQTT has three delivery guarantees:

**QoS 0 — at most once.** Fire and forget. Publisher sends, broker forwards, that's it. If a packet is lost, oh well.

**QoS 1 — at least once.** Publisher sends, waits for an ACK from the broker. If no ACK within a timeout, retransmit. The downside is that retransmits can cause duplicates — the subscriber might see the same message twice.

**QoS 2 — exactly once.** A four-step handshake guarantees the message is delivered, and only once. Slowest. Most expensive. Used for stuff like billing events where neither loss nor duplication is acceptable.

Most IoT traffic is QoS 0 or 1. QoS 2 is rare.

### Retained messages

By default, if you publish a message and there are no subscribers, the message is gone. The broker does not store it.

But you can publish with the **retain** flag. The broker stores the *last* retained message for each topic. When a new subscriber connects and subscribes to that topic, the broker immediately sends them the retained message — so they get the current value without waiting for the next publish.

This is how you make "what is the current temperature?" work over a system that is fundamentally event-based. The publisher publishes every reading with retain=true. New subscribers get the latest reading the moment they connect.

### Last Will and Testament

When a client connects to the broker, it can register a "last will" message. Topic, payload, QoS. The broker remembers it.

If the client disconnects gracefully (sends a `DISCONNECT` packet), the will is forgotten.

If the client disconnects ungracefully (TCP timeout, power loss, network drop), the broker publishes the will message on behalf of the dead client.

This is how you do "presence" detection. Every device's will is `device/<id>/status` = `offline`. Every device's connect message publishes `device/<id>/status` = `online` with retain=true. If the device drops, the broker publishes `offline` automatically. Subscribers immediately know.

It is a beautiful design. Three lines of config and you have automatic dead-device detection.

### MQTT 5

MQTT 5 (released 2019) is the modern version. It adds:

- **Reason codes** — the broker tells you exactly why your connection was rejected, not just "denied."
- **User properties** — custom key/value headers on messages.
- **Topic aliases** — replace a long topic with a short integer to save bytes.
- **Shared subscriptions** — load-balance a topic across multiple subscribers (`$share/group/topic`).
- **Session expiry** — clients can specify how long the broker should keep their session if they disconnect.

Most new deployments use MQTT 5. Old deployments still use 3.1.1, which is fine.

## MQTT-SN (Sensor Network, even more constrained)

MQTT runs on TCP. TCP is a fairly heavy protocol — three-way handshake, sequence numbers, retransmits, congestion control. For most IoT this is fine. For *really* constrained networks (Zigbee, sub-GHz mesh, 802.15.4), TCP is too much.

MQTT-SN is MQTT redesigned for these networks. It runs on UDP (or even directly over the radio, no IP at all). Topics become two-byte integers (`temp` becomes `0x0001`) negotiated in advance. There is a gateway that translates between MQTT-SN on the constrained side and regular MQTT on the IP side, so cloud subscribers do not even know they are talking to MQTT-SN devices.

### Why integer topic IDs

In regular MQTT, every PUBLISH carries the full topic string. If your topic is `factory/line3/sensor7/vibration/x-axis`, that is 41 bytes of overhead on every single message. For a sensor sending one reading per second, that is 41 bytes a second of pure topic-name traffic, forever. Across thousands of devices, this becomes the dominant cost.

MQTT-SN says: register the topic name once, get back a 2-byte ID, and use the ID for the rest of the session. Now every PUBLISH is 2 bytes of topic plus the actual data. On a slow Zigbee link sending 50-byte messages, this is the difference between fitting the message in one frame and having to fragment.

There is also a "predefined" topic ID space — the topics are agreed in advance (in firmware, in config) so you do not even need a registration round trip. You just start publishing to topic ID `0x0001` and both sides know what it means.

### The MQTT-SN gateway

MQTT-SN devices do not talk to a regular MQTT broker directly. They talk to a **gateway**. The gateway sits at the edge of the constrained network (often on the same box as the Zigbee/Thread border router) and does protocol translation:

```
Constrained side                  IP side
================                  =======

  +--------+    MQTT-SN     +---------+    MQTT     +--------+
  |Sensor  | -- UDP/802.15.4 -> | Gateway | --- TCP/TLS --> |Broker  |
  |(MQTT-SN)|                |         |              |(MQTT)  |
  +--------+                  +---------+              +--------+
```

The broker thinks it is talking to one client (the gateway). The gateway multiplexes hundreds of constrained devices through that one connection. Cloud subscribers do not know or care that the data originally came from a tiny Zigbee node.

You see MQTT-SN in:
- Zigbee/Thread devices that need pub/sub semantics on top of the mesh.
- Sub-GHz proprietary radios that don't have IP at all.
- Battery sensors that wake up, publish one number, and sleep for an hour.

If you are doing IP-based IoT, you will use regular MQTT. MQTT-SN is for the special low-power cases.

## AMQP for IoT (rare; AWS/Azure use HTTP+MQTT)

AMQP — Advanced Message Queuing Protocol — is another pub/sub protocol. It came out of the financial services world (1990s, J.P. Morgan). It is heavier than MQTT, more enterprise-y, has more features (transactions, exchanges, multiple message types).

AMQP shows up in IoT mostly as a *backend* protocol — between cloud services, not between the device and the cloud. Azure IoT Hub speaks AMQP toward downstream services. Some industrial gateways use AMQP. But the device side is almost always MQTT or HTTP.

### Why MQTT won for devices

AMQP has a concept of "exchanges" with several routing modes (direct, topic, fanout, headers). It has channels, frames, transactions. The header machinery alone is bigger than the entire MQTT protocol. For a server-to-server message bus inside a data center, this is great. For a battery sensor, it is way too much.

MQTT's pitch is the opposite: do one thing (pub/sub on a tree of topic strings) and do it in 50 lines of code on the device. The CONNECT packet is 12 bytes plus the client ID. The PUBLISH packet is 2 bytes plus the topic plus the payload. A whole MQTT client implementation can fit in 8KB of flash.

So for the device-to-cloud link, MQTT won by being smaller. For service-to-service inside the cloud, AMQP and Kafka are better choices because they have better backpressure, better routing, better at-least-once semantics for batch processing.

If you are building a new IoT product and you are choosing between MQTT and AMQP for the device-to-cloud link, choose MQTT. AMQP is overkill for a sensor.

### HTTP as a fallback

The other "non-MQTT" option you see is plain HTTPS. AWS IoT lets you POST messages over HTTPS as well as MQTT. The device just makes a regular HTTPS request, signed with the device's certificate. It is heavier per message (TLS handshake, HTTP headers) but easier to debug (you can use `curl`) and works through corporate firewalls that block port 8883 (MQTT-over-TLS).

For very-low-rate devices (one message per hour), HTTPS is fine. For anything that publishes constantly, MQTT wins on bandwidth and battery.

## The IoT Cloud (AWS IoT Core, Azure IoT Hub, Google Cloud IoT (deprecated 2023), Mosquitto, EMQ X, HiveMQ)

The cloud side is where your devices talk to. It is a big MQTT broker (usually) plus a bunch of services that do things with the messages — store them in a database, run rules, send alerts, push commands back to devices.

### AWS IoT Core

Amazon's offering. The MQTT broker is multi-tenant and fully managed — you do not run any servers. Devices authenticate with X.509 certificates (each device gets its own cert). Topics are organized under your AWS account. Rules can route messages to other AWS services: Lambda, DynamoDB, S3, Kinesis. Device Shadow gives every device a JSON document representing its desired and reported state, so cloud apps can read the latest state without having to wait for the device to publish.

Sizing: AWS IoT Core handles billions of devices. It is the default choice if you are already on AWS.

### Azure IoT Hub

Microsoft's offering. Similar in shape to AWS IoT Core. Supports MQTT, AMQP, and HTTPS. Has Device Twins (analogous to AWS Device Shadows). Has Direct Methods for command-style invocations (not just pub/sub). Integrates with Azure Functions, Stream Analytics, Service Bus.

If you are an enterprise on Azure already, this is the path of least resistance.

### Google Cloud IoT Core (deprecated 2023)

Google had an IoT service. Then they shut it down in August 2023. If you see references to "Google Cloud IoT" in a tutorial, the tutorial is out of date. Google's pitch now is: use a third-party broker (HiveMQ, EMQ X) and pipe the messages into Google's data services (Pub/Sub, BigQuery). It works, but you run more infrastructure yourself.

### Mosquitto

Open-source MQTT broker. Lightweight, single-binary, written in C. Runs on a Raspberry Pi. Default broker for hobbyists and home-assistant deployments. Not great at huge scale (millions of devices) but fine for hundreds or thousands.

### EMQ X

Open-source MQTT broker designed for scale. Written in Erlang. Clusters across many machines. Handles millions of concurrent connections. Common choice for self-hosted production deployments.

### HiveMQ

Commercial MQTT broker. Java. Highly scalable. Lots of enterprise features (clustering, monitoring, integrations with Kafka, etc.). Common in industrial IoT.

### What to pick

- **Hobbyist / home lab** — Mosquitto on a Pi.
- **Production, AWS shop** — AWS IoT Core.
- **Production, Azure shop** — Azure IoT Hub.
- **Production, self-hosted, big scale** — EMQ X or HiveMQ.
- **Production, self-hosted, small scale** — Mosquitto behind a load balancer.

### What "the cloud" actually does for you

The broker is just the front door. The interesting work happens behind it:

- **Ingestion** — receive messages, write them to durable storage (Kinesis, Event Hubs, Kafka).
- **Routing** — based on topic patterns, fan messages out to different downstream systems (database, alerting, analytics).
- **Device shadow / twin** — cache the latest reported state of each device in a JSON document so apps can read "current temperature of device 12345" without waiting for the device to publish again.
- **Command pipeline** — push commands from cloud apps down to devices via the same MQTT connection (subscribe to `cmd/<deviceId>` on the device side).
- **Rules engine** — "if temperature > 80, publish alert to Slack." All declarative, runs in the cloud, never touches the device.
- **Authorization** — per-device certificates with per-device topic policies, so a compromised sensor can only read/write its own topics.
- **Fleet provisioning** — bootstrap mechanism to give a brand-new device its real credentials. The device ships with a generic provisioning cert, makes one call, gets back a per-device cert, throws away the bootstrap cert.
- **Jobs / OTA** — orchestrate firmware updates across the fleet: "roll out firmware v2.3 to 1% of devices, watch for crashes, then ramp up."

You can build all of this yourself. You probably should not, unless you have a really good reason. The hyperscaler IoT services exist precisely so you do not have to.

## IoT Security (TLS-PSK, DTLS, TLS 1.3, hardware security modules, secure boot, OTA firmware updates)

Security in IoT is a horror movie. Devices ship with default passwords. Devices never get firmware updates. Devices have hard-coded private keys baked into the firmware. The Mirai botnet that took down half the internet in 2016 was made of webcams whose default password was `admin`.

Doing security properly requires getting several layers right. We will go through them.

### Layer 1 — Encryption on the wire (TLS / DTLS)

Every byte that leaves the device should be encrypted. Otherwise anyone on the same network (or sniffing the wireless) can read your sensor data and, worse, send fake commands.

**TLS** (Transport Layer Security) is the standard. It is the same TLS your browser uses for HTTPS. TLS 1.2 is everywhere. TLS 1.3 (RFC 8446, 2018) is faster, simpler, and more secure — fewer round trips, removed broken ciphers, only the modern key exchanges allowed.

**DTLS** is TLS for UDP. Same security properties, just adapted for an unreliable, datagram transport. CoAP over DTLS is `coaps://`. DTLS 1.2 is the deployed version; DTLS 1.3 is finalized (RFC 9147) and rolling out.

For tiny devices that cannot do full RSA or ECDSA, there is **TLS-PSK** (Pre-Shared Key) — instead of a certificate, the device and server share a symmetric secret key configured at the factory. PSK is cheap to compute but has the obvious problem that if the key leaks (firmware extraction, memory readout), every device using that key is compromised. Modern best practice: use per-device PSKs, not a single PSK shared across all devices.

### Layer 2 — Device identity

The device needs to prove who it is. The cloud needs to prove who it is. There are three common approaches:

**Username/password.** The device has a username and password baked in. It sends them at MQTT CONNECT. Easy to implement. Easy to leak — anyone with firmware access can read them. Not recommended for production.

**Per-device X.509 certificates.** The device has its own private key and a certificate signed by a CA. The TLS handshake proves the device knows the private key. The cloud knows it is talking to device-with-cert-id-12345 because that cert is in the cloud's database. Strong. Standard. Best practice.

**Pre-shared key (PSK).** Symmetric secret. Cheaper than certs. Used in DTLS-PSK and some TLS-PSK profiles. Per-device PSKs only.

**TPM / Secure Element.** A small chip on the device that stores the private key in tamper-resistant hardware. The key never leaves the chip. Even if you decap the main MCU, you cannot read the key. The chip signs things on demand. Microchip ATECC608A, NXP A71CH, Infineon OPTIGA Trust are common. This is the gold standard.

### Layer 3 — Secure boot

The MCU should refuse to run firmware that is not signed by you.

```
Secure boot chain
=================

  +-----------+
  | ROM boot  |   immutable, factory-burned
  | (stage 0) |   contains public key fingerprint
  +-----------+
        |
        |   verify signature of stage 1
        |   using burned-in public key
        v
  +-----------+
  | bootloader|   signed by your private key
  | (stage 1) |   embedded in flash
  +-----------+
        |
        |   verify signature of application
        |   using bootloader's trusted key
        v
  +-----------+
  | application|  signed by your private key
  | (stage 2) |  the actual firmware
  +-----------+
        |
        |   running, can request firmware update
        v
       ...
```

Each stage verifies the next. The chain root is the ROM bootloader, which is burned into the chip at manufacture and cannot be changed. If an attacker tries to flash unsigned firmware, the bootloader sees the bad signature and refuses to boot.

You see this on every modern smartphone. You also see it on serious IoT chips: STM32 with TrustZone, ESP32 secure boot, NXP HABv4, Nordic nRF53/nRF91.

Without secure boot, a five-minute physical attack with a JTAG probe can replace your firmware with anything. With secure boot, that attack does not work — the chip will not run the bad firmware.

### Layer 4 — OTA firmware updates

Devices ship with bugs. Devices need to get patched. Without an OTA (Over-The-Air) update mechanism, you have a million devices in the field with a known vulnerability and no way to fix them.

A safe OTA system has:

- **Signed firmware images.** The device verifies the signature before installing. Same key as secure boot.
- **Atomic updates.** Either the new firmware is fully installed or the device rolls back to the old one. Never half-installed.
- **A/B partitioning.** Two firmware slots. Boot from slot A. Install new firmware to slot B. Try to boot B. If B works for a while, mark B as the new default. If B crashes, fall back to A.
- **Anti-rollback.** Don't let an attacker push old (vulnerable) signed firmware. The device tracks the minimum allowed version.

OTA is hard. People get it wrong. Devices get bricked. Plan for it from day one.

### Layer 5 — Authentication on top of MQTT/CoAP

Even with TLS, you still need application-level authentication. MQTT has username/password and certificate-based auth. CoAP often relies on DTLS-PSK identity. AWS IoT and Azure IoT both use the device certificate as the identity for authorization.

You also need **authorization** — what is this device allowed to publish? What is it allowed to subscribe to? AWS IoT uses IAM-style policies. Azure uses RBAC. Mosquitto uses an ACL file. Make sure each device can only access *its own* topics — otherwise a compromised device can read every other device's data.

## Common Errors

Verbatim error strings from real device logs, real broker logs, real network sniffer captures.

### `BLE: connection failed (no advertiser)`

**Where you see it:** trying to connect to a BLE peripheral that is not advertising. Your phone scans, the peripheral does not show up, you get this.

**Why:** the peripheral is asleep, out of range, or its advertising interval is way too long (you stopped scanning before it transmitted). Could also be an iOS/Android caching the device under a randomized address that has rotated.

**Fix:** wake the peripheral (press the button), check it is in range, increase your scan duration, or clear the Bluetooth cache on your phone.

### `ATT_ERROR_INVALID_HANDLE (0x01)`

**Where you see it:** trying to read or write a GATT characteristic on a BLE device.

**Why:** the handle (a 16-bit identifier for the characteristic) you used does not exist on this device. Usually because you cached handles from a previous connection, the device's GATT table changed (firmware update), and now your stale handles are wrong.

**Fix:** rediscover services after every connection, or implement service-changed notifications (the BLE spec has a way for the device to tell the client "my GATT layout changed, please rescan").

### `GATT_INSUFFICIENT_AUTHENTICATION (0x05)`

**Where you see it:** trying to read or write a characteristic that requires pairing.

**Why:** the characteristic's permissions say "must be authenticated" but you are not paired with the device.

**Fix:** initiate pairing first, then retry the read/write. On iOS this happens automatically in most cases; on Android and embedded clients, you usually have to call the pair API explicitly.

### `Bluetooth pairing failed: PIN code mismatch`

**Where you see it:** Bluetooth Classic or BLE legacy pairing.

**Why:** the user typed the wrong PIN, or the two devices are using different PIN codes (one says `0000`, the other expects `1234`), or the pairing method negotiation failed.

**Fix:** check the device manual for the correct PIN. Many devices use `0000` or `1234` by default. If the device has a display, it should show the PIN to type; if it has a keyboard, you type the PIN it shows.

### `Zigbee: no parent — orphan condition`

**Where you see it:** a Zigbee end device that lost touch with its router/coordinator parent.

**Why:** parent power-cycled, parent moved out of range, parent's network changed, or the end device's poll interval was too long and it got expired from the parent's child table.

**Fix:** the end device performs a rejoin scan — looks for a parent that knows about it (using its long-form 64-bit IEEE address), or starts a fresh association. Make sure your parent keeps child entries alive long enough; tune `nwkEndDeviceTimeout`.

### `LoRaWAN: join request rejected (DevEUI not in NS)`

**Where you see it:** the device tries to OTAA-join the LoRaWAN network and the network server (NS) refuses.

**Why:** the DevEUI (the device's unique 64-bit ID) is not provisioned in the network server's database. Either you forgot to add it, or you typed it wrong, or the AppKey is wrong (which makes the join MIC fail and the NS records it as "unknown device").

**Fix:** verify DevEUI, JoinEUI, and AppKey in both the device and the network server. They have to match exactly. Check the NS logs — most LoRaWAN servers will tell you whether the DevEUI was unknown or the MIC failed.

### `LoRaWAN: ADR not converging`

**Where you see it:** Adaptive Data Rate is supposed to push the device to a higher SF/lower SF based on RSSI/SNR, but the device's data rate just bounces around or never improves.

**Why:** the link is genuinely marginal (the SNR really is on the edge), or the device is moving (so the link quality changes faster than ADR can adapt), or the network server's ADR algorithm is overly conservative.

**Fix:** for stationary devices, leave it alone — ADR can take hours to converge. For mobile devices, *disable* ADR (`ADR=off`) and pick a fixed SF that works in worst-case conditions. ADR only makes sense for devices that don't move.

### `MQTT: connack 0x05 (not authorized)`

**Where you see it:** MQTT 3.1.1 CONNACK return code 5, or MQTT 5 reason code 0x87 / 0x97.

**Why:** the broker accepted the TCP connection and the CONNECT packet but refused authentication. Wrong username/password, wrong certificate, certificate not in the broker's trust store, or device is allowed to connect but not allowed to publish to the topic it tried.

**Fix:** check broker logs (they usually tell you exactly which auth check failed). Verify the cert is correctly signed by a CA the broker trusts. Verify the username/password (case-sensitive). Check the policy/ACL allows the device to do what it is trying to do.

### `MQTT: connection lost (KeepAlive timeout)`

**Where you see it:** the broker disconnects the client because no PINGREQ arrived within the KeepAlive window.

**Why:** the device went to sleep and forgot to PING. Or the network dropped and TCP did not detect it (silent NAT timeouts are common). Or the device crashed.

**Fix:** make sure the device sends a PINGREQ at half the KeepAlive interval. If KeepAlive is 60s, send PING every 30s. If the device is going to sleep for longer than KeepAlive, gracefully DISCONNECT first and reconnect on wake.

### `CoAP: 4.04 Not Found`

**Where you see it:** CoAP response code from a server.

**Why:** the URI you requested does not exist on the server. Either you typed it wrong, or the resource was deleted, or the server's endpoint structure does not match what you assumed.

**Fix:** GET `/.well-known/core` on the server to discover what resources actually exist (this is the CoAP equivalent of asking the server "what URLs do you have?"). Then use the correct path.

### `CoAP: 4.13 Request Entity Too Large`

**Where you see it:** you tried to POST or PUT a payload bigger than the server can accept in one packet.

**Why:** CoAP runs on UDP, so the whole message has to fit in one IP datagram (typically under ~1280 bytes for IPv6 to avoid fragmentation). If you exceed that, the server says no.

**Fix:** use the **Block-Wise Transfer** extension (RFC 7959). Split the payload into blocks of, say, 64 bytes, send them with `Block1` options, and the server reassembles. Most CoAP libraries handle this transparently; you just have to enable it.

### `Matter: PASE/CASE handshake failed`

**Where you see it:** trying to commission a Matter device or trying to talk to one that is already commissioned.

**Why (PASE):** PASE (Password Authenticated Session Establishment) is the initial commissioning handshake. It uses the PIN/QR code as the shared secret. If the PIN is wrong, the handshake fails. Also fails if the device is already commissioned to another fabric and not in commissioning mode.

**Why (CASE):** CASE (Certificate Authenticated Session Establishment) is what commissioned devices use after onboarding. Uses node operational certificates. Fails if the device's NOC is missing, expired, or revoked, or if the controller does not have a matching trust root.

**Fix (PASE):** double-check the PIN. Put the device into commissioning mode (factory reset usually works). Make sure your commissioner has the correct vendor ID and product ID.

**Fix (CASE):** re-commission the device into your fabric. If certificates expired (yes, this happens — Matter NOCs can have expiry), provision new ones via the fabric's CA. Check that your controller's root CA matches the device's installed root.

### `MQTT: connection refused (broker not running)` / `ECONNREFUSED`

**Where you see it:** TCP refuses on port 1883 or 8883.

**Why:** the broker isn't running, or it is running on a different port, or a firewall is dropping the SYN.

**Fix:** verify the broker is up (`systemctl status mosquitto`, or check the cloud broker's status page). Verify the port (1883 plain, 8883 TLS, 443 with WebSockets fallback). Check firewalls on both ends.

### `MQTT: SSL_ERROR_BAD_CERT` / TLS handshake failure

**Where you see it:** the device can reach the broker on TCP but the TLS handshake fails.

**Why:** wrong CA bundle on the device (it does not trust the broker's cert), wrong hostname (the cert is for `iot.example.com` but you connected to the IP), expired cert on the broker, or the device's clock is so wrong that the cert appears to be in the future.

**Fix:** install the correct root CA on the device. Connect by hostname, not IP. Sync the device's clock (NTP, or take time from the cellular network, or take a hint from the TLS server's clock during handshake). Renew the broker cert if expired.

### `LoRaWAN: MIC mismatch on uplink`

**Where you see it:** the network server sees an uplink, but the Message Integrity Code does not validate.

**Why:** the device's session keys (NwkSKey, AppSKey) and the network server's stored keys do not match. Usually because the device rejoined and got new keys but the server still has old ones, or the frame counter rolled and the server's counter cache is stale.

**Fix:** force a fresh OTAA join. Or, if using ABP, double-check the keys typed on both sides. Frame counter desync is also a thing — make sure relaxed-frame-counter mode is on if your device might reset.

### Closing thought on this chunk

Application-layer protocols (MQTT, CoAP) and the cloud are where the IoT story stops being about radios and starts being about software. The radio chapters are full of physics and regulatory rules. This chapter is full of certificates, ACLs, and JSON. Both halves matter. A device with a perfect radio link and a misconfigured TLS cert is just as broken as a device with great firmware and a dead antenna.

The next chunk pulls all of this together — gateways, edge computing, time series, dashboards, real-world deployments, and the gotchas you only learn by shipping a product.

## Hands-On

You will not have every IoT radio sitting on your desk. That is fine. Most of these commands run on a plain Linux laptop with a Bluetooth chip. Some require a USB BLE dongle. Some require an MQTT broker. Some require an LPWAN gateway. We will mark which ones need extra hardware. Everything else you can type today.

Whenever you see `$` at the start of a line, type the rest. The lines after are what your computer prints back. We will keep the output literal so you know what "right" looks like.

### Bluetooth — finding the chip

Plug nothing in. Just type:

```bash
$ hciconfig
hci0:   Type: Primary  Bus: USB
        BD Address: AC:DE:48:00:11:22  ACL MTU: 1021:8  SCO MTU: 64:1
        UP RUNNING
        RX bytes:312 acl:0 sco:0 events:21 errors:0
        TX bytes:1020 acl:0 sco:0 commands:21 errors:0
```

If `UP RUNNING` is not there, the chip is asleep. Wake it up:

```bash
$ sudo hciconfig hci0 up
$ hciconfig hci0
hci0:   Type: Primary  Bus: USB
        BD Address: AC:DE:48:00:11:22  ACL MTU: 1021:8  SCO MTU: 64:1
        UP RUNNING
```

`hciconfig` is the old, deprecated tool, but it is everywhere and it still works. The new tool is `bluetoothctl`. Open it interactively:

```bash
$ bluetoothctl
Agent registered
[bluetooth]# show
Controller AC:DE:48:00:11:22 (public)
        Name: my-laptop
        Alias: my-laptop
        Class: 0x006c010c
        Powered: yes
        Discoverable: no
        Pairable: yes
        UUID: Generic Attribute Profile (00001801-0000-1000-8000-00805f9b34fb)
        UUID: A/V Remote Control        (0000110e-0000-1000-8000-00805f9b34fb)
        UUID: Generic Access Profile    (00001800-0000-1000-8000-00805f9b34fb)
        Modalias: usb:v1D6Bp0246d0540
[bluetooth]# 
```

### Bluetooth — scanning for nearby devices

Inside `bluetoothctl`:

```text
[bluetooth]# scan on
Discovery started
[CHG] Controller AC:DE:48:00:11:22 Discovering: yes
[NEW] Device 7C:64:56:A1:B2:C3 BLE-Lightbulb
[NEW] Device 12:34:56:78:9A:BC Heart Rate Monitor
[NEW] Device 00:11:22:33:44:55 [unknown]
[CHG] Device 7C:64:56:A1:B2:C3 RSSI: -54
[CHG] Device 12:34:56:78:9A:BC RSSI: -71
```

`RSSI` is the signal strength in dBm. `-54` is closer than `-71`. The closer to zero, the stronger.

To stop:

```text
[bluetooth]# scan off
Discovery stopped
```

The classic command-line way (not interactive) is:

```bash
$ sudo hcitool lescan
LE Scan ...
7C:64:56:A1:B2:C3 BLE-Lightbulb
12:34:56:78:9A:BC Heart Rate Monitor
00:11:22:33:44:55 (unknown)
^C
```

Press `Ctrl-C` to stop. `lescan` only finds Low-Energy advertisers, not Classic Bluetooth devices.

### Bluetooth — pairing and connecting

Inside `bluetoothctl`:

```text
[bluetooth]# pair 7C:64:56:A1:B2:C3
Attempting to pair with 7C:64:56:A1:B2:C3
[CHG] Device 7C:64:56:A1:B2:C3 Connected: yes
Request confirmation
[agent] Confirm passkey 123456 (yes/no): yes
[CHG] Device 7C:64:56:A1:B2:C3 Bonded: yes
[CHG] Device 7C:64:56:A1:B2:C3 Paired: yes
Pairing successful
```

After pairing, connect:

```text
[bluetooth]# connect 7C:64:56:A1:B2:C3
Attempting to connect to 7C:64:56:A1:B2:C3
[CHG] Device 7C:64:56:A1:B2:C3 Connected: yes
Connection successful
```

To see what services the device exposes:

```text
[bluetooth]# info 7C:64:56:A1:B2:C3
Device 7C:64:56:A1:B2:C3 (public)
        Name: BLE-Lightbulb
        Alias: BLE-Lightbulb
        Paired: yes
        Bonded: yes
        Trusted: yes
        Blocked: no
        Connected: yes
        LegacyPairing: no
        UUID: Generic Access Profile    (00001800-0000-1000-8000-00805f9b34fb)
        UUID: Generic Attribute Profile (00001801-0000-1000-8000-00805f9b34fb)
        UUID: Device Information        (0000180a-0000-1000-8000-00805f9b34fb)
        UUID: Battery Service           (0000180f-0000-1000-8000-00805f9b34fb)
        UUID: Light Control             (0000ffe0-0000-1000-8000-00805f9b34fb)
        RSSI: -54
        TxPower: 4
```

The UUIDs are GATT services. `0x180F` is the standard battery service. `0xFFE0` is a vendor-defined light control.

### Bluetooth — reading and writing GATT characteristics

```bash
$ gatttool -I -b 7C:64:56:A1:B2:C3
[7C:64:56:A1:B2:C3][LE]> connect
Attempting to connect to 7C:64:56:A1:B2:C3
Connection successful
[7C:64:56:A1:B2:C3][LE]> primary
attr handle: 0x0001, end grp handle: 0x0007 uuid: 00001800-0000-1000-8000-00805f9b34fb
attr handle: 0x0008, end grp handle: 0x000b uuid: 00001801-0000-1000-8000-00805f9b34fb
attr handle: 0x000c, end grp handle: 0x000f uuid: 0000180a-0000-1000-8000-00805f9b34fb
attr handle: 0x0010, end grp handle: 0x0014 uuid: 0000180f-0000-1000-8000-00805f9b34fb
attr handle: 0x0020, end grp handle: 0xffff uuid: 0000ffe0-0000-1000-8000-00805f9b34fb
[7C:64:56:A1:B2:C3][LE]> char-read-hnd 0x0012
Characteristic value/descriptor: 5d 
[7C:64:56:A1:B2:C3][LE]> char-write-req 0x0024 01
Characteristic value was written successfully
[7C:64:56:A1:B2:C3][LE]> disconnect
[7C:64:56:A1:B2:C3][LE]> exit
```

Handle `0x0012` is the battery level (`0x5d` = 93%). Handle `0x0024` is the lightbulb power; we wrote `0x01` to turn it on.

`gatttool` is deprecated upstream but still ships on most distros. The modern way is `bluetoothctl`'s `gatt.select-attribute` and `gatt.read`/`gatt.write`, or a Python library:

```python
# bleak.py
import asyncio
from bleak import BleakClient

ADDRESS = "7C:64:56:A1:B2:C3"
LIGHT = "0000ffe1-0000-1000-8000-00805f9b34fb"

async def main():
    async with BleakClient(ADDRESS) as c:
        await c.write_gatt_char(LIGHT, b"\x01")
        print("light on")

asyncio.run(main())
```

```bash
$ python3 bleak.py
light on
```

### Bluetooth — sniffing the radio

`btmon` shows raw HCI packets — every frame the chip sends or receives.

```bash
$ sudo btmon
Bluetooth monitor ver 5.66
= Note: Linux version 6.5.0 (x86_64)
= Note: Bluetooth subsystem version 2.22
= New Index: AC:DE:48:00:11:22 (Primary,USB,hci0)
> HCI Event: LE Meta Event (0x3e) plen 43
      LE Advertising Report (0x02)
        Num reports: 1
        Event type: Connectable undirected - ADV_IND (0x00)
        Address type: Public (0x00)
        Address: 7C:64:56:A1:B2:C3
        Data length: 31
        Flags: 0x06
          LE General Discoverable Mode
          BR/EDR Not Supported
        Complete local name: 'BLE-Lightbulb'
        RSSI: -54 dBm
```

`hcidump` is the older, equivalent tool. `btmgmt info` and `btmgmt --help` give a more API-style view of what BlueZ knows.

```bash
$ btmgmt info
Index list with 1 item
hci0:   Primary controller
        addr AC:DE:48:00:11:22 version 12 manufacturer 2 class 0x6c010c
        supported settings: powered connectable fast-connectable discoverable bondable
                            link-security ssp br/edr hs le advertising secure-conn
                            debug-keys privacy configuration static-addr
        current settings: powered bondable ssp br/edr le secure-conn
        name my-laptop
        short name my-laptop
```

### Bluetooth — what's plugged in

```bash
$ lsusb | grep -i bluetooth
Bus 001 Device 005: ID 0a12:0001 Cambridge Silicon Radio, Ltd Bluetooth Dongle (HCI mode)
```

```bash
$ journalctl -u bluetooth -f
Apr 27 12:44:01 my-laptop bluetoothd[1234]: Bluetooth daemon 5.66
Apr 27 12:44:01 my-laptop bluetoothd[1234]: Starting SDP server
Apr 27 12:44:01 my-laptop bluetoothd[1234]: Bluetooth management interface 1.22 initialized
Apr 27 12:44:02 my-laptop bluetoothd[1234]: Endpoint registered: sender=:1.42 path=/MediaEndpoint/A2DPSink/sbc
```

`-f` follows the log live. Useful when you are debugging pairing failures.

### MQTT — running a broker on your laptop

```bash
$ sudo apt install mosquitto mosquitto-clients
...
$ mosquitto -v
1701234567: mosquitto version 2.0.18 starting
1701234567: Using default config.
1701234567: Starting in local only mode.
1701234567: Opening ipv4 listen socket on port 1883.
1701234567: Opening ipv6 listen socket on port 1883.
1701234567: mosquitto version 2.0.18 running
```

Open a second terminal and subscribe:

```bash
$ mosquitto_sub -h localhost -t '#' -v
```

Open a third terminal and publish:

```bash
$ mosquitto_pub -h localhost -t sensors/livingroom/temperature -m '21.4'
```

Back in the second terminal:

```text
sensors/livingroom/temperature 21.4
```

The `#` is the multi-level wildcard; `+` is single-level. Try:

```bash
$ mosquitto_sub -h localhost -t 'sensors/+/temperature' -v &
$ mosquitto_pub -h localhost -t sensors/kitchen/temperature -m '19.1'
sensors/kitchen/temperature 19.1
$ mosquitto_pub -h localhost -t sensors/livingroom/humidity -m '47'
$ # nothing — humidity does not match the filter
```

### MQTT — passwords

```bash
$ sudo mosquitto_passwd -c /etc/mosquitto/passwd alice
Password: ********
Reenter password: ********
$ sudo cat /etc/mosquitto/passwd
alice:$7$101$...truncated...
```

Edit `/etc/mosquitto/conf.d/auth.conf`:

```text
allow_anonymous false
password_file /etc/mosquitto/passwd
```

Restart:

```bash
$ sudo systemctl restart mosquitto
$ mosquitto_pub -h localhost -t test -m hi
Connection error: Connection Refused: not authorised.
$ mosquitto_pub -h localhost -u alice -P 'mypass' -t test -m hi
$ # success — no output is good
```

### MQTT — QoS, retain, last will

```bash
$ mosquitto_pub -h localhost -t announce/online -m '{"node":"pi1"}' -r
$ mosquitto_sub -h localhost -t announce/online -v
announce/online {"node":"pi1"}
^C
```

The `-r` made the message *retained*. Any new subscriber gets it instantly even if the publisher is long gone.

QoS 1 (at-least-once):

```bash
$ mosquitto_pub -h localhost -t orders/new -m '{"id":42}' -q 1
```

The broker will resend until a `PUBACK` is received. QoS 2 (exactly-once):

```bash
$ mosquitto_pub -h localhost -t orders/new -m '{"id":42}' -q 2
```

Adds a four-step handshake, used rarely outside finance and billing.

`mqtt-cli` (a JVM-based client from HiveMQ) is fancier:

```bash
$ mqtt sub -h localhost -t '#' -v
2026-04-27 12:30:01 sensors/livingroom/temperature 21.4
```

### CoAP — the HTTP-for-tiny-things

```bash
$ sudo apt install libcoap3-bin
$ coap-client -m get coap://[::1]:5683/.well-known/core
v:1 t:CON c:GET i:7c19 {} [ ]
</.well-known/core>;ct=40,</hello>;rt="HelloWorld";if="basic",</time>;rt="time";obs
```

`/.well-known/core` is a discovery endpoint — every CoAP server must expose it. The output is CoRE Link Format (RFC 6690). Each item is a resource with metadata.

Run a server in another terminal:

```bash
$ libcoap-server -v 7
coap-server starting on port 5683
```

Get a real resource:

```bash
$ coap-client -m get coap://[::1]:5683/hello
v:1 t:CON c:GET i:1a2b {} [ ]
Hello World!
```

OBSERVE (subscribe to changes, RFC 7641):

```bash
$ coap-client -s 60 -m get coap://[::1]:5683/time
v:1 t:CON c:GET i:c0de {Observe:0} [ ]
2026-04-27T12:30:00Z
2026-04-27T12:30:05Z
2026-04-27T12:30:10Z
^C
```

Python with `aiocoap`:

```python
import asyncio
from aiocoap import Context, Message, GET

async def main():
    proto = await Context.create_client_context()
    req = Message(code=GET, uri="coap://localhost/hello")
    resp = await proto.request(req).response
    print(resp.payload.decode())

asyncio.run(main())
```

### Zigbee — bridging to MQTT

`zigbee2mqtt` is the popular bridge. With a CC2531 USB dongle plugged in:

```bash
$ docker run -d --name z2m \
    --device=/dev/ttyACM0 \
    -v /var/lib/z2m:/app/data \
    koenkk/zigbee2mqtt
$ docker logs z2m
[2026-04-27 12:32:01] info: Starting Zigbee2MQTT version 1.36.0
[2026-04-27 12:32:01] info: Coordinator firmware version: '20211226'
[2026-04-27 12:32:01] info: Currently 0 devices are joined
[2026-04-27 12:32:02] info: Zigbee2MQTT started!
```

Pair a device by putting it in pairing mode (usually triple-press a reset button), then in MQTT you will see:

```bash
$ mosquitto_sub -h localhost -t 'zigbee2mqtt/#' -v
zigbee2mqtt/bridge/event {"type":"device_joined","data":{"friendly_name":"0x00158d000123","ieee_address":"0x00158d0001234567"}}
zigbee2mqtt/bridge/event {"type":"device_interview","data":{"friendly_name":"0x00158d000123","status":"started"}}
```

`zigbee-herdsman` is the underlying Node.js library. `deconz` is an alternative bridge from Dresden Elektronik, often used with the Conbee II stick.

### LoRaWAN — talking to a network server

ChirpStack is open-source. If you run it locally and have a gateway pointed at your laptop:

```bash
$ chirpstack-cli devices list --application-id 1
+----------------------------------+-----------------------+--------+
| dev_eui                          | name                  | enabled|
+----------------------------------+-----------------------+--------+
| 0102030405060708                 | field-soil-1          | true   |
+----------------------------------+-----------------------+--------+
$ docker logs chirpstack-gateway-bridge
INFO  starting Gateway Bridge version=4.0.7
INFO  backend/concentratord: connecting concentratord_event=ipc:///tmp/concentratord_event
INFO  uplink received freq=868100000 dr=5 sf=7 rssi=-87 snr=8.5
```

A real LoRa packet on the air looks like:

```text
PHYPayload: 4002030405000100020A6F4E1B
MIC: 0x1B4E6F0A
DevAddr: 0x05040302
FCnt: 0x0001
```

### NB-IoT and LTE-M — talking to a cellular modem

These connect over a serial port (`/dev/ttyUSB0` is typical):

```bash
$ sudo screen /dev/ttyUSB0 115200
AT
OK
AT+CGMI
Quectel
OK
AT+CGATT?
+CGATT: 1
OK
AT+CSQ
+CSQ: 22,99
OK
AT+QENG="servingcell"
+QENG: "servingcell","NOCONN","LTE","FDD",262,02,1234567,...,B20,6300,...,-92,-12,-65,15,-,-,-,-
OK
```

`+CSQ: 22,99` means RSSI ≈ −69 dBm, BER unknown. `B20` is band 20 (800 MHz, common for IoT in Europe). To attach with a specific APN:

```text
AT+CGDCONT=1,"IP","iot.example.net"
OK
AT+CGACT=1,1
OK
```

### MQTT-SN — the tiny twin

```bash
$ sudo apt install mqtt-sn-tools
$ mqtt-sn-pub -h localhost -p 1885 -t sensors/temp -m '21.5'
$ mqtt-sn-sub -h localhost -p 1885 -t 'sensors/+'
sensors/temp 21.5
```

You also need an MQTT-SN gateway (e.g. `mqtt-sn-gateway` from RSMB or Eclipse Paho). The gateway translates short topic IDs to full MQTT topic strings.

### ESP-IDF — flashing an ESP32

```bash
$ git clone --recursive https://github.com/espressif/esp-idf
$ . esp-idf/export.sh
$ cd esp-idf/examples/get-started/blink
$ idf.py menuconfig
$ idf.py build
...
$ idf.py -p /dev/ttyUSB0 flash monitor
Connecting...
Chip is ESP32-D0WD-V3 (revision v3.0)
...
Hash of data verified.
Leaving...
Hard resetting via RTS pin...
=== Booting
I (29) boot: ESP-IDF v5.2 2nd stage bootloader
I (199) cpu_start: App version:      1
I (200) blink_example: Example configured to blink GPIO LED!
```

`idf.py monitor` opens a serial console; `Ctrl-]` exits.

### Zephyr — `west` for everything else

```bash
$ pip install west
$ west init zephyrproject
$ cd zephyrproject
$ west update
$ west build -b nrf52840dk_nrf52840 zephyr/samples/bluetooth/peripheral_hr
$ west flash
-- west build: running target flash
[1/1] Flashing nrf52840dk_nrf52840
Flashing file: build/zephyr/zephyr.hex
Verified OK.
Done.
```

### Cloud IoT — what is provisioned

```bash
$ aws iot describe-thing --thing-name field-sensor-001
{
    "defaultClientId": "field-sensor-001",
    "thingName": "field-sensor-001",
    "thingId": "abcdef12-3456-7890-abcd-ef1234567890",
    "thingArn": "arn:aws:iot:us-east-1:111122223333:thing/field-sensor-001",
    "attributes": {
        "deployment": "field",
        "model": "rev3"
    },
    "version": 4
}
```

```bash
$ az iot hub device-identity show --hub-name myhub --device-id field-sensor-001
{
  "authentication": {
    "symmetricKey": null,
    "type": "selfSigned",
    "x509Thumbprint": {
      "primaryThumbprint": "0123...EF",
      "secondaryThumbprint": null
    }
  },
  "deviceId": "field-sensor-001",
  "status": "enabled"
}
```

### Paho MQTT in C — the smallest meaningful client

```c
// pub.c
#include <MQTTClient.h>
#include <stdio.h>

int main(void) {
    MQTTClient c;
    MQTTClient_create(&c, "tcp://localhost:1883", "demo",
                      MQTTCLIENT_PERSISTENCE_NONE, NULL);
    MQTTClient_connectOptions o = MQTTClient_connectOptions_initializer;
    MQTTClient_connect(c, &o);
    MQTTClient_message m = MQTTClient_message_initializer;
    m.payload = "21.4"; m.payloadlen = 4; m.qos = 1;
    MQTTClient_deliveryToken t;
    MQTTClient_publishMessage(c, "sensors/t", &m, &t);
    MQTTClient_waitForCompletion(c, t, 1000);
    MQTTClient_disconnect(c, 1000);
    MQTTClient_destroy(&c);
    return 0;
}
```

```bash
$ cc pub.c -lpaho-mqtt3c -o pub
$ ./pub
$ # exit code 0 = success
```

That is the entire IoT stack in forty commands. You will not memorise them. You will copy them into your shell and watch what happens, and the next time you read about MQTT or BLE you will picture these traces in your head.

## Common Confusions

These are the mistakes everyone makes the first time. Read them now and you will save yourself a week of head-scratching later.

### "Bluetooth 5 made BLE four times faster"

**Wrong:** Bluetooth 5.0 added an optional 2 Mbps PHY, but most BLE devices still run at 1 Mbps. Range was the bigger leap, via the new long-range Coded PHY (S=8) at 125 kbps.

**Right:** BLE 5 *can* be faster (2 Mbps) or longer-range (Coded PHY, slower) but the headline numbers are *options*, not defaults. Most BLE peripherals you buy default to 1 Mbps for compatibility.

### "Central means master, peripheral means slave"

**Wrong:** Treating central/peripheral and master/slave as identical. Many BLE devices act as peripheral but become *master* of the connection in some sub-protocols.

**Right:** GAP roles (central/peripheral) describe who initiates. Link-layer roles (master/slave, now called central/peripheral too) describe who controls timing. They usually align, but in mesh and LE Audio they can differ.

### "A GATT service and a characteristic are the same thing"

**Wrong:** Reading a service and expecting bytes.

**Right:** A *service* is a folder that groups related *characteristics*. You read/write characteristics. Reading a service handle returns the service definition, not user data.

### "Pairing and bonding are interchangeable"

**Wrong:** Saying "they paired" and meaning the keys persist forever.

**Right:** *Pairing* is the act of exchanging keys for one session. *Bonding* is storing those keys to disk so the next reconnect is instant. You can pair without bonding.

### "Zigbee and Z-Wave are basically the same"

**Wrong:** They share a niche (low-power home automation mesh), so they must use the same radio.

**Right:** Zigbee uses 2.4 GHz IEEE 802.15.4. Z-Wave uses sub-GHz proprietary radio (908 MHz US, 868 MHz EU). Z-Wave goes through walls better; Zigbee has more bandwidth. They do not interoperate.

### "Thread is just Zigbee with IP"

**Wrong:** Treating Thread as a re-skin of Zigbee.

**Right:** Both ride the same 802.15.4 radio at 2.4 GHz, but Thread uses 6LoWPAN + IPv6 + UDP. Each Thread node has a real IPv6 address. Zigbee invented its own application stack on top of 802.15.4.

### "Matter is a radio"

**Wrong:** "We added Matter support to our chip."

**Right:** Matter is an *application layer* that runs over Wi-Fi, Thread, or Ethernet. The chip has Wi-Fi or Thread. Matter is software on top.

### "LoRa and LoRaWAN are the same thing"

**Wrong:** Calling a LoRa modem a "LoRaWAN modem."

**Right:** LoRa is the chirp-spread-spectrum *physical layer* (Semtech IP). LoRaWAN is the *MAC layer + network architecture* on top of it. You can do peer-to-peer LoRa without LoRaWAN — it just won't talk to a public network server.

### "LoRaWAN devices can receive whenever the server wants"

**Wrong:** Class A devices listening on demand.

**Right:** Class A nodes only open two short receive windows after each uplink. The server *must* queue downlinks until the next uplink. Latency is unbounded. Class B (scheduled beacons) and Class C (always-on) trade battery for responsiveness.

### "6LoWPAN is a different protocol from IPv6"

**Wrong:** Treating 6LoWPAN as a competitor to IPv6.

**Right:** 6LoWPAN is *header compression* for IPv6 over 802.15.4. The IPv6 packet is real and end-to-end; 6LoWPAN squashes the header so it fits inside an 81-byte 802.15.4 frame.

### "MQTT and CoAP do the same thing"

**Wrong:** "Pick whichever, they're equivalent."

**Right:** MQTT is broker-based, TCP-only, pub/sub. CoAP is peer-to-peer, UDP-only, request/response (HTTP-shaped). CoAP fits constrained networks; MQTT fits stable backhaul.

### "MQTT-SN is just MQTT over UDP"

**Wrong:** Replacing TCP with UDP and calling it done.

**Right:** MQTT-SN re-encodes the packets with short topic IDs, fixed-size headers, and optional sleep/wake support. A *gateway* translates between MQTT-SN and full MQTT. The two protocols share a name and a model, not a wire format.

### "QoS 2 is always safer than QoS 0"

**Wrong:** Cranking everything to QoS 2.

**Right:** QoS 2 doubles the round-trips and broker state. For a temperature reading you publish every 10 s, QoS 0 is fine — the next reading replaces stale data anyway. Use QoS 2 only when each message is uniquely critical (orders, billing, irreversible commands).

### "BLE has plenty of MTU; just send what you want"

**Wrong:** Sending 250-byte JSON in one BLE write.

**Right:** Default ATT MTU is **23 bytes**, so payload is **20 bytes** unless both sides negotiate higher. Even after MTU exchange, on-air fragmentation kicks in. Always design protocols around 16-byte chunks.

### "iBeacon and Eddystone are both Apple"

**Wrong:** Lumping the formats together.

**Right:** iBeacon is Apple's manufacturer-data format (UUID + major + minor). Eddystone is Google's (UID/URL/TLM/EID). AltBeacon is open-source. They all ride BLE non-connectable advertising — just with different payloads.

### "Cert pinning works fine on tiny devices"

**Wrong:** Storing one CA root and hoping for the best.

**Right:** When the cert rotates and the device cannot reach the new CA, you brick the fleet. Plan for OTA-updatable trust stores from day one, or use multiple pins with overlap windows.

### "Secure boot is for expensive chips"

**Wrong:** "We use a $1 MCU; we cannot do secure boot."

**Right:** STM32 has RDP. nRF52 has APPROTECT. ESP32 has Flash Encryption + Secure Boot v2. RP2040 is the awkward one (no built-in). Even at \$1 you can verify a signed bootloader if you choose the part carefully.

### "Firmware OTA is just downloading and rebooting"

**Wrong:** Writing the new image straight over the running one.

**Right:** Power can fail mid-write. Use A/B partitions (MCUboot, Mender) so the bootloader can fall back. Verify signatures *before* swap. Mark the new image as confirmed only after a successful boot.

### "Google Cloud IoT Core is the standard cloud option"

**Wrong:** Designing 2026 hardware around Google Cloud IoT Core.

**Right:** Google retired Cloud IoT Core in **August 2023**. Live options are AWS IoT Core, Azure IoT Hub, and self-hosted MQTT (Mosquitto, EMQ X, HiveMQ). Migrate any old design before deployment.

## Vocabulary

| Term | Meaning in plain English |
|---|---|
| **IoT** | Internet of Things — small computers in everyday objects, networked. |
| **M2M** | Machine-to-machine — older name for IoT, usually cellular. |
| **embedded** | A computer hidden inside something that isn't "a computer." |
| **MCU** | Microcontroller unit — CPU + flash + RAM + I/O on one chip. |
| **SoC** | System on chip — bigger MCU, often with radio and memory controller. |
| **MPU** | Microprocessor unit — CPU only; RAM is external. |
| **Cortex-M0** | Smallest ARM core, 32-bit, ~10 MHz, used in $0.20 chips. |
| **Cortex-M3** | Mid-range ARM, 32-bit, used in STM32F1 and nRF51. |
| **Cortex-M4** | M3 plus DSP and (sometimes) FPU; nRF52, STM32F4. |
| **Cortex-M7** | High-performance Cortex-M; STM32H7, ATSAM E70. |
| **Cortex-M33** | M-profile with TrustZone-M for secure boot; nRF53/91. |
| **RISC-V for IoT** | Open ISA used by ESP32-C3/C6, BL602, and CH32V. |
| **ESP32** | Espressif Wi-Fi + BLE SoC, very common in hobby IoT. |
| **ESP8266** | Older Wi-Fi-only Espressif chip; legacy. |
| **STM32** | STMicroelectronics ARM Cortex-M family. |
| **nRF52** | Nordic BLE SoC; nRF52832, nRF52840. |
| **nRF53** | Dual-core BLE + Thread + Matter SoC. |
| **nRF91** | Nordic LTE-M / NB-IoT cellular SoC. |
| **RP2040** | Raspberry Pi's dual-core Cortex-M0+ MCU. |
| **BLE** | Bluetooth Low Energy. |
| **Bluetooth Classic** | Older, higher-bandwidth Bluetooth (audio, file transfer). |
| **BT 4.0** | First BLE; introduced 2010. |
| **BT 5.0** | 2 Mbps PHY, Coded PHY long-range, more advertising channels. |
| **BT 5.1** | Direction Finding (AoA, AoD). |
| **BT 5.2** | LE Audio, Isochronous Channels. |
| **BT 5.3** | Periodic advertising enhancements, channel classification. |
| **BT 5.4** | Encrypted advertising data, Periodic Advertising with Responses. |
| **BT 6.0** | Channel sounding (cm-accurate ranging), 2024 spec. |
| **peripheral** | BLE role: advertises, accepts connections (e.g., heart-rate strap). |
| **central** | BLE role: scans, initiates connections (e.g., phone). |
| **advertisement** | Broadcast packet a peripheral sends so centrals can find it. |
| **scan response** | Extra data packet a peripheral sends when scanned. |
| **GATT** | Generic Attribute Profile — how BLE structures data. |
| **service** | GATT folder grouping related characteristics. |
| **characteristic** | GATT data item — readable, writable, or notifiable. |
| **descriptor** | Metadata attached to a characteristic (units, CCCD). |
| **attribute handle** | 16-bit ID for any GATT item. |
| **ATT** | Attribute Protocol — the wire format below GATT. |
| **L2CAP** | Logical Link Control and Adaptation — Bluetooth multiplexer. |
| **GAP** | Generic Access Profile — discovery, advertising, connection. |
| **beacon** | BLE device that only advertises, never connects. |
| **iBeacon** | Apple's beacon advertising format. |
| **Eddystone** | Google's beacon advertising format (deprecated 2018 but still seen). |
| **AltBeacon** | Radius Networks' open beacon format. |
| **BLE Mesh** | Many-to-many BLE topology for lighting and sensors. |
| **Zigbee 3.0** | Unified Zigbee profile (replaces HA, ZLL, etc.). |
| **Zigbee Pro** | Mesh stack from the Zigbee Alliance. |
| **IEEE 802.15.4** | Low-rate PHY/MAC for short-range wireless. |
| **FFD** | Full-function device — can route in 802.15.4. |
| **RFD** | Reduced-function device — leaf only. |
| **coordinator** | One-per-network 802.15.4 node that runs the PAN. |
| **router** | 802.15.4 node that forwards for others. |
| **end device** | 802.15.4 leaf that sleeps most of the time. |
| **beacon mode** | Synchronous 802.15.4 mode with periodic beacons. |
| **Z-Wave** | Sub-GHz proprietary mesh protocol from Silicon Labs. |
| **Z-Wave 700** | 700-series chip generation, S2 security required. |
| **Z-Wave 800** | 800-series, longer range and lower power. |
| **Thread 1.1** | First public Thread spec, 2017. |
| **Thread 1.4** | 2024 spec adding TREL and credential sharing. |
| **Border Router** | Gateway between Thread and your home IP network. |
| **Leader** | Single Thread router elected as network coordinator. |
| **REED** | Router-Eligible End Device — can become a router. |
| **FED** | Full End Device — does not route. |
| **MED** | Minimal End Device — basic leaf. |
| **SED** | Sleepy End Device — leaf with low duty cycle. |
| **MLE** | Mesh Link Establishment — Thread's neighbour discovery. |
| **Matter** | Application-layer IoT standard from CSA (formerly Project CHIP). |
| **CSA** | Connectivity Standards Alliance, owner of Matter and Zigbee. |
| **OTBR** | OpenThread Border Router. |
| **HomeKit** | Apple's smart-home framework, now Matter-aware. |
| **Google Home** | Google's smart-home platform, Matter Controller. |
| **Alexa** | Amazon smart-home, Matter Controller. |
| **SmartThings** | Samsung's smart-home, Matter Controller, runs Thread. |
| **LoRa** | Chirp-spread-spectrum sub-GHz PHY from Semtech. |
| **LoRaWAN** | LoRa-based MAC and network architecture. |
| **Class A** | LoRaWAN end device with two RX windows after TX. |
| **Class B** | LoRaWAN device that schedules RX via beacons. |
| **Class C** | LoRaWAN device with continuous RX (mains-powered). |
| **gateway** | LoRaWAN packet forwarder, no MAC logic. |
| **network server** | LoRaWAN brain that decodes, decrypts, and routes. |
| **application server** | Receives application payloads from network server. |
| **join server** | Holds AppKey and runs OTAA join. |
| **OTAA** | Over-the-Air Activation — modern LoRaWAN join. |
| **ABP** | Activation By Personalisation — pre-baked keys, legacy. |
| **DevEUI** | 64-bit device identifier (LoRaWAN, Zigbee, Thread). |
| **JoinEUI** | 64-bit join server identifier. |
| **AppKey** | LoRaWAN root key for OTAA. |
| **NwkSKey** | LoRaWAN network session key. |
| **AppSKey** | LoRaWAN application session key. |
| **FCntUp** | Uplink frame counter. |
| **FCntDown** | Downlink frame counter. |
| **FPort** | Application port number in LoRaWAN frame. |
| **MIC** | Message Integrity Code — 4-byte AES-CMAC. |
| **ADR** | Adaptive Data Rate — server picks SF for each device. |
| **DR** | Data Rate — combination of SF and BW. |
| **SF** | Spreading Factor (7–12 in LoRa). |
| **CR** | Coding Rate (4/5–4/8 in LoRa). |
| **BW** | Bandwidth (125, 250, 500 kHz). |
| **Sigfox** | Ultra-narrowband LPWAN, 100 bps, 12-byte messages. |
| **NB-IoT** | 3GPP narrowband IoT (Cat-NB1, NB2). |
| **LTE-M** | 3GPP Cat-M1, broader bandwidth than NB-IoT. |
| **5G mMTC** | 5G Massive Machine-Type Communications. |
| **URLLC** | Ultra-Reliable Low-Latency Communications. |
| **eMBB** | Enhanced Mobile Broadband. |
| **6LoWPAN** | IPv6 over Low-power Wireless Personal Area Networks. |
| **RFC 6282** | 6LoWPAN header compression. |
| **RFC 6775** | Neighbor Discovery for 6LoWPAN. |
| **RFC 6550** | RPL — IPv6 Routing for Low-power Networks. |
| **RPL** | Routing Protocol for Low-power and Lossy Networks. |
| **MPL** | Multicast Protocol for Low-power and Lossy Networks. |
| **CoAP** | Constrained Application Protocol, RFC 7252. |
| **OBSERVE** | CoAP option (RFC 7641) for subscriptions. |
| **DTLS** | TLS over UDP. |
| **TLS-PSK** | TLS using a pre-shared key, no certificates. |
| **EDHOC** | Ephemeral Diffie-Hellman over COSE, RFC 9528. |
| **OSCORE** | Object Security for Constrained RESTful Environments, RFC 8613. |
| **MQTT v3.1.1** | Most-deployed MQTT version. |
| **MQTT v5.0** | Adds reason codes, properties, shared subscriptions. |
| **MQTT-SN** | MQTT for Sensor Networks — UDP-friendly variant. |
| **QoS 0** | At-most-once — fire and forget. |
| **QoS 1** | At-least-once — ack required, may duplicate. |
| **QoS 2** | Exactly-once — four-way handshake. |
| **retained message** | MQTT message stored by broker, sent to new subscribers. |
| **last will** | MQTT message broker publishes when client drops. |
| **broker** | MQTT server. |
| **client** | MQTT publisher or subscriber. |
| **topic** | Slash-delimited string for routing MQTT messages. |
| **wildcard #** | MQTT multi-level wildcard. |
| **wildcard +** | MQTT single-level wildcard. |
| **AWS IoT Core** | Amazon's managed MQTT + device shadow service. |
| **Azure IoT Hub** | Microsoft's managed device-to-cloud bridge. |
| **Mosquitto** | Open-source MQTT broker from Eclipse. |
| **EMQ X** | High-throughput open-source MQTT broker. |
| **HiveMQ** | Commercial MQTT broker, popular in automotive. |
| **VerneMQ** | Erlang-based open-source MQTT broker. |
| **AWS Greengrass** | AWS edge runtime for IoT. |
| **EdgeX Foundry** | Linux Foundation open-source IoT edge framework. |
| **Zephyr** | Linux Foundation RTOS for MCUs. |
| **FreeRTOS** | Most popular small RTOS, now under Amazon. |
| **mbed OS** | Arm's RTOS, deprecated but still in use. |
| **NuttX** | POSIX-like RTOS, used in PX4 drones. |
| **RIOT** | Friendly RTOS aimed at IoT. |
| **ESP-IDF** | Espressif's SDK for ESP32. |
| **MCUboot** | Open-source secure bootloader for MCUs. |
| **secure boot** | Bootloader that verifies firmware signature. |
| **hardware root of trust** | Immutable key burnt into silicon. |
| **SE** | Secure Element — separate chip with private keys. |
| **TPM** | Trusted Platform Module — bigger SE for PCs. |
| **TrustZone** | ARM CPU mode separating secure / non-secure worlds. |
| **OTA firmware update** | Over-the-air image swap. |
| **A/B partition** | Two firmware slots; bootloader picks the working one. |
| **Mender** | Open-source OTA platform. |
| **hawkBit** | Eclipse OTA platform. |
| **balena.io** | Container-based device fleet manager. |
| **particle.io** | Cellular IoT platform with hardware. |
| **Home Assistant** | Open-source home automation hub. |
| **Node-RED** | Visual flow-based IoT programming. |
| **Tasmota** | Open firmware for Sonoff/ESP devices. |
| **ESPHome** | YAML-driven ESP firmware generator. |
| **ESP-NOW** | Espressif's connectionless 2.4 GHz protocol. |
| **Ubertooth One** | Open-source Bluetooth sniffer hardware. |
| **BlueZ** | Linux Bluetooth stack. |
| **hcitool** | Old Linux BlueZ command-line tool. |
| **btmon** | Modern BlueZ HCI tracer. |
| **bluepy** | Python wrapper for BlueZ. |
| **bleak** | Cross-platform Python BLE library. |
| **paho-mqtt** | Eclipse MQTT client libraries. |
| **libcoap** | C CoAP library and CLI. |
| **aiocoap** | Async Python CoAP. |
| **scapy-dot15d4** | Scapy plug-in for 802.15.4 packet crafting. |
| **dBm** | Decibel-milliwatts; -100 dBm is weaker than -50 dBm. |
| **RSSI** | Received Signal Strength Indicator. |
| **PHY** | Physical layer. |
| **MAC** | Medium Access Control layer. |
| **PAN** | Personal Area Network (802.15.4 segment). |
| **MIC vs MAC** | MIC = message integrity code; MAC = address or layer. |

## Try This

You don't need a lab. Most of these are doable with a phone, a laptop, and free software.

1. **Install nRF Connect on your phone.** Walk around your home and look at every BLE device that advertises. Note the names, the manufacturer codes, and the RSSI as you move around. You will quickly find sensors, headphones, and TVs you didn't know were broadcasting.
2. **Set up Mosquitto on your laptop.** Subscribe with `mosquitto_sub` and publish with `mosquitto_pub`. Now publish a *retained* message and disconnect both clients. Reconnect the subscriber and watch the retained message appear — that is exactly how the cloud delivers "last known state" to a slow-waking device.
3. **Install Wireshark with the BLE-baseband dissector** (`btatt`, `btsmp`, `btl2cap`). If you have a Nordic dongle in sniffer mode, capture a pairing flow and step through the SMP messages — random, confirm, distribute keys.
4. **Buy one Zigbee bulb and one CC2531 USB stick.** Run `zigbee2mqtt` and watch the JSON appear when you press the bulb's reset button. Try sending `{"state":"ON"}` from `mosquitto_pub` to `zigbee2mqtt/<friendly_name>/set`.
5. **Spin up a TTN (The Things Network) device.** Free, public LoRaWAN. Set up a Heltec WiFi LoRa 32 to do an OTAA join and send a "hello" every 5 minutes. Watch the gateway list to see which receivers caught your packet.
6. **Run a CoAP server with `aiocoap-server`.** Add a resource that returns the current temperature. Use `coap-client` to GET it. Now subscribe with OBSERVE and update the resource every second.
7. **Capture an MQTT session with Wireshark** on `tcp.port == 1883`. Watch CONNECT → CONNACK → SUBSCRIBE → SUBACK → PUBLISH → DISCONNECT. Note how each packet is just a one-byte type, a length, and a payload.
8. **Stand up an EMQ X cluster** with two nodes in Docker. Subscribe on one, publish on the other. Watch how the broker bridges the cluster automatically.
9. **Try MQTT v5 features.** `mosquitto_pub -V mqttv5 -D PUBLISH user-property "trace=42"` adds a custom property. Subscribe with `-V mqttv5 -F '%X'` to see it.
10. **Read the Bluetooth Core Spec.** Just twenty pages. Pick the SMP chapter (volume 3, part H) and follow how the long-term key is derived. It is shorter than people fear.
11. **Install OpenThread CLI on a single nRF52840 dongle.** `wpantund` brings up the Thread interface. `ipconfig -A` shows you a real IPv6 address you can ping from your laptop.
12. **Decompile an OTA firmware image.** Grab a public ESP32 binary, run `esptool.py image_info`, and look at the segment table. Then run Ghidra on the `app` segment and see how much of it is FreeRTOS.

## Where to Go Next

You now understand the whole IoT stack: the radios, the gateways, the protocols, the cloud, the security model, and the surprises. There is one big question left: which slice do you want to specialise in?

- **Radio person.** Read IEEE 802.15.4-2020 cover to cover, then the Bluetooth Core Spec. Get a HackRF or a software-defined radio. Learn to read modulation diagrams. Move toward 5G NR PHY if you want a job.
- **Protocol person.** Read every RFC linked in the references below. Build CoAP, MQTT-SN, and OSCORE clients from scratch in C. Submit a draft to the IETF LWIG or T2TRG working group.
- **Embedded person.** Pick one MCU family (nRF, STM32, ESP32) and live there. Read the chip's reference manual. Learn DMA, interrupts, and the bootloader. Build a product that shipts.
- **Cloud person.** Run AWS IoT Core, Azure IoT Hub, and a Mosquitto cluster in parallel. Compare device shadow models. Learn fleet provisioning, just-in-time registration, and OTA campaign tooling.
- **Security person.** Move on to `offensive/iot-ot-hacking` and `offensive/wireless-hacking`. Pick up an Ubertooth One. Read the Hackers Choice Bluetooth papers. Reverse a smart-bulb's firmware. Find a CVE.
- **Mesh person.** Run Thread, Zigbee, BLE Mesh, and Matter side by side. Read the routing protocols (RPL, AODV, MMRP). Find out why mesh networks "self-heal" and what happens when they don't.

You don't have to pick one forever. The IoT stack rewards generalists who can talk to all of these specialists.

## See Also

- [networking/quic](../networking/quic.md) — modern transport that some IoT cloud platforms now offer in preview.
- [networking/mqtt](../networking/mqtt.md) — production-grade MQTT reference: brokers, clusters, MQTT v5, bridging.
- [networking/dpdk](../networking/dpdk.md) — when you outgrow the kernel network stack at the edge gateway.
- [networking/cisco-wireless](../networking/cisco-wireless.md) — enterprise Wi-Fi controllers, often the back-haul for IoT-on-Wi-Fi.
- [networking/coredns](../networking/coredns.md) — DNS for IoT discovery, mDNS bridging, Matter commissioning.
- [security/tls](../security/tls.md) — the cipher-suite reality you must accept on tiny devices.
- [offensive/iot-ot-hacking](../offensive/iot-ot-hacking.md) — Modbus, BACnet, S7Comm, smart-bulb firmware, Zigbee key extraction.
- [offensive/wireless-hacking](../offensive/wireless-hacking.md) — BLE sniffing, jamming, replay attacks, KRACK.
- [ramp-up/wifi-eli5](wifi-eli5.md) — Wi-Fi explained the same way as this sheet.
- [ramp-up/tls-eli5](tls-eli5.md) — TLS for non-cryptographers, with the same gentle voice.
- [ramp-up/ip-eli5](ip-eli5.md) — IPv4 and IPv6 from zero.
- [ramp-up/tcp-eli5](tcp-eli5.md) — handshakes, retransmits, congestion control.
- [ramp-up/udp-eli5](udp-eli5.md) — the protocol underneath CoAP and DTLS.
- [ramp-up/linux-kernel-eli5](linux-kernel-eli5.md) — the kernel that runs your gateway and your cloud.

## References

- **Bluetooth Core Specification 6.0** (Bluetooth SIG, 2024) — the complete authoritative spec. Volume 1: Architecture. Volume 3 Part F: Attribute Protocol. Volume 3 Part G: Generic Attribute Profile. Volume 3 Part H: Security Manager. Volume 6: Low Energy Controller. <https://www.bluetooth.com/specifications/specs/core-specification-6-0/>
- **IEEE 802.15.4-2020** — Standard for Low-Rate Wireless Networks. The PHY/MAC underneath Zigbee, Thread, and 6LoWPAN. <https://standards.ieee.org/standard/802_15_4-2020.html>
- **RFC 6282** — Compression Format for IPv6 Datagrams over IEEE 802.15.4-Based Networks. The 6LoWPAN compression rules. <https://www.rfc-editor.org/rfc/rfc6282>
- **RFC 6550** — RPL: IPv6 Routing Protocol for Low-Power and Lossy Networks. The DODAG construction algorithm. <https://www.rfc-editor.org/rfc/rfc6550>
- **RFC 6775** — Neighbor Discovery Optimization for IPv6 over LoWPANs. <https://www.rfc-editor.org/rfc/rfc6775>
- **RFC 7252** — The Constrained Application Protocol (CoAP). The whole protocol in 112 pages. <https://www.rfc-editor.org/rfc/rfc7252>
- **RFC 7641** — Observing Resources in the CoAP. The OBSERVE option that turns CoAP into a subscribe-friendly protocol. <https://www.rfc-editor.org/rfc/rfc7641>
- **RFC 8613** — Object Security for Constrained RESTful Environments (OSCORE). Application-layer security for CoAP. <https://www.rfc-editor.org/rfc/rfc8613>
- **RFC 9528** — Ephemeral Diffie-Hellman Over COSE (EDHOC). Lightweight key exchange for constrained devices. <https://www.rfc-editor.org/rfc/rfc9528>
- **MQTT Version 5.0** — OASIS Standard, 7 March 2019. The current MQTT spec. <https://docs.oasis-open.org/mqtt/mqtt/v5.0/mqtt-v5.0.html>
- **MQTT-SN Protocol Specification v1.2** — Eurotech / IBM, 2013. Still the canonical MQTT-SN reference. <https://www.oasis-open.org/committees/download.php/66091/MQTT-SN_spec_v1.2.pdf>
- **LoRaWAN 1.0.4 Specification** — LoRa Alliance, 2020. Class A/B/C and the join procedure. <https://lora-alliance.org/resource_hub/lorawan-104-specification-package/>
- **LoRaWAN 1.1 Specification** — LoRa Alliance, 2017. Adds dual-key OTAA. <https://lora-alliance.org/resource_hub/lorawan-specification-v1-1/>
- **Thread Specification 1.4** — Thread Group, 2024. Mesh, MLE, TREL, credential sharing. <https://www.threadgroup.org/support#specifications>
- **Matter 1.4 Specification** — Connectivity Standards Alliance, 2024. The cross-vendor smart-home application layer. <https://csa-iot.org/all-solutions/matter/>
- **3GPP TS 36.300** — LTE / NB-IoT / LTE-M overall description.
- **3GPP TS 38.300** — 5G NR overall description, including mMTC.
- **Sigfox Connected Objects Radio Specifications** — Sigfox, 2020.
- **Russ White, *Computer Networks: A Systems Approach*** — Chapter 29: IoT. The systems view of how all of this fits together. <https://book.systemsapproach.org/>
- **NIST SP 800-213** — IoT Device Cybersecurity Guidance for the Federal Government. <https://csrc.nist.gov/publications/detail/sp/800-213/final>
- **ETSI EN 303 645** — Cyber Security for Consumer IoT. The European baseline.
- **OWASP IoT Top 10 (2018)** — still useful, despite age.
- **Espressif ESP-IDF Programming Guide** — <https://docs.espressif.com/projects/esp-idf/>
- **Nordic nRF Connect SDK Documentation** — <https://docs.nordicsemi.com/>
- **Zephyr Project Documentation** — <https://docs.zephyrproject.org/>
- **OpenThread Documentation** — <https://openthread.io/>
- **AWS IoT Core Developer Guide** — <https://docs.aws.amazon.com/iot/>
- **Azure IoT Hub Documentation** — <https://learn.microsoft.com/azure/iot-hub/>

That is the full IoT stack. Radios, gateways, protocols, cloud, security, and the long list of names. You will not remember every term. You don't have to. You now have a picture in your head of how a sensor gets a reading from a beet field to your phone, what every layer is doing, and where to look when it breaks. The rest is practice.

