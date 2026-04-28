# WiFi — ELI5

> WiFi is Ethernet without the wire. Every device shares the same air. They take turns talking so nobody drowns each other out.

## Prerequisites

- `ramp-up/tcp-eli5` — gives you the picture of how data moves end-to-end. WiFi is a way of carrying that data over invisible radio waves instead of a copper cable, so the TCP picture is still useful underneath.
- `ramp-up/ip-eli5` — gives you the picture of IP addresses and packets. WiFi delivers IP packets just like Ethernet does. The two sheets above tell you what is *inside* the packets. This sheet tells you how those packets fly across the room.

If you have not read those, you can still follow this sheet; just know that WiFi is the bottom layer that carries everything else. Whenever you see a phone open a website, the website is built from HTTP, HTTP rides on TCP, TCP rides on IP, and IP rides on WiFi. WiFi is the bottom step of the staircase. This sheet is about that bottom step.

If a word looks weird, jump to the **Vocabulary** table at the bottom. Every weird word is in that table with a one-line plain-English definition.

If you see a `$` at the start of a code block line, that means "type the rest of this line into your terminal." Lines without a `$` are what your computer prints back at you (the output). Many of these commands need `sudo` (the Linux "I'm allowed to do scary things" word). I'll show `sudo` when you need it.

## What Even Is WiFi

### Imagine Ethernet first

Picture your house. In one room you have a desktop computer. In another room you have a router. Between them, in the wall, is a long copper wire. Whenever the computer wants to send data to the internet, it pushes electricity down the wire. The router catches the electricity, decodes it back into data, and sends that data on toward the rest of the internet. That's Ethernet. One device on each end. One wire. The wire is **private**: only the two ends can hear what's on it.

Now imagine you have ten computers in your house, all over the place. Bedroom, kitchen, basement, garage. You don't want to drill ten holes in the wall and run ten wires. You want to be able to walk around with a laptop. You want your phone to work everywhere. Drilling ten wires doesn't help with the laptop and the phone, because they move.

So we got rid of the wire. Or, more precisely, we replaced the wire with the air.

### WiFi is Ethernet over invisible radio waves

That's the entire idea. The data is still the same data. The packets are still the same packets. But instead of being pushed down a copper wire, they are squirted out into the air as radio waves. Other devices in the room hear the radio waves with their own little radios, and they decode the waves back into data.

Picture it like a wire that fills the entire room. Every device in the room is **already plugged in**, just by being in the room. The "wire" is the air. There is no plug. There are no separate wires for separate devices. There is one shared wire, and that wire is everywhere.

This is amazing because now you can move. You can walk to the kitchen. You're still plugged in, because the wire is the air, and the kitchen has air too. You can go on the porch. Still plugged in, as long as the radio waves can reach.

### The catch: everybody hears everybody

If WiFi is "Ethernet over the air," and the air is shared by every device in range, then there's a problem. With a regular Ethernet wire, only the two ends of the wire can hear each other. But with WiFi, the air doesn't have ends. Every device in earshot can hear every other device.

That means if your laptop wants to talk and your phone wants to talk and your TV wants to talk and your roommate's laptop wants to talk, they will all squirt radio waves at the same time. The waves will smash into each other in the air. Nobody's signal will be readable. Everybody's data will be ruined. We call that a **collision**.

It would be like if you and four friends all tried to talk at the same time in a small room. Nobody can hear anything. Everyone's words turn into mush. We call this **the everyone-talks-at-once problem.**

### CSMA/CA: the politeness algorithm

To fix it, WiFi devices follow a rule called **CSMA/CA.** That's a long acronym. It stands for **Carrier Sense Multiple Access with Collision Avoidance.** Don't be scared. We'll break it down.

- **Carrier Sense:** before talking, listen to see if anyone else is talking.
- **Multiple Access:** lots of devices share the same air.
- **Collision Avoidance:** try really hard not to talk over someone else.

Here is how a polite WiFi device behaves before sending a single packet:

1. Listen to the air for a moment. Is anyone else talking? Are there radio waves humming on the channel right now?
2. If yes, don't talk. Wait a tiny random amount of time. Listen again.
3. If no, the air is quiet. Pick a tiny random delay (called **backoff**), wait that long, then squirt the packet.
4. After squirting, wait for the receiver to send back a tiny acknowledgment ("got it!"). That little ack is called an **ACK frame**.
5. If you don't get an ACK back within a millisecond or two, assume your packet got smashed in a collision. Pick a *bigger* random backoff and try again.

That's the whole dance. Listen-before-talk, plus retry with bigger random delays if your message was lost. WiFi devices do this billions of times a day. That's why WiFi looks like it works smoothly even though there's only one shared "wire" (the air) and dozens of devices on it.

The "collision avoidance" part is different from how Ethernet over a wire used to work. Old wired Ethernet did **CSMA/CD** — collision *detection*. The wire could tell when two electrical signals smashed into each other, because the voltage would spike. In the air, you can't tell that easily. So WiFi avoids collisions in the first place by being polite up front, instead of detecting them after they happen. That little C/D vs C/A swap is the whole reason WiFi is its own thing.

### Frequencies are like lanes on a highway

Now imagine you have a freeway with three lanes. Cars in the slow lane go slow, but the lane goes farther into the suburbs. Cars in the fast lane go fast, but it ends sooner. WiFi is like that. The radio waves come in different **bands** (lanes), each with different tradeoffs.

- **2.4 GHz** is the slow neighborhood road. The waves go through walls easily and travel far. But there isn't much room for cars (less bandwidth) and lots of other stuff uses this lane too: microwaves, baby monitors, Bluetooth, old cordless phones. Crowded.
- **5 GHz** is the fast freeway. The waves go faster (more bandwidth) but they don't travel as far and walls block them more. There are way more lanes. Way less crowded.
- **6 GHz** is the brand-new express lane that opened in 2020 (in the US). Even more bandwidth, even more lanes, basically nobody on it yet because it's new. Only WiFi 6E and WiFi 7 devices can use this lane.

A WiFi access point usually offers all three (or two of them, depending on age). Devices pick which lane to take. Newer phones and laptops will jump on the 5 GHz or 6 GHz lane when they can; older devices stay on 2.4 GHz.

### Different ways to think about WiFi

Just like in the kernel sheet, here are a few different mental pictures, because no single picture covers everything.

The **shared wire** picture is best for understanding why everyone has to take turns. The wire goes through everyone's device at the same time, so only one talker can be on the wire at a time, or signals smash.

The **walkie-talkie at a barbeque** picture is best for understanding the politeness. Imagine ten people at a barbeque, each holding a walkie-talkie tuned to channel 6. They all hear each other. If two people press the talk button at the same time, both messages turn into mush. So everyone learns to wait, listen, then talk briefly, then let someone else go.

The **hotel front desk** picture is best for understanding how an access point works. The access point (the AP) is the front desk. Phones, laptops, TVs are guests. Guests can talk to other guests, but in WiFi networks they normally route through the front desk. The front desk hands out room keys (IP addresses), keeps track of who is in the building (the **association table**), and forwards messages out to the rest of the world.

The **lighthouse beam** picture is best for understanding **beacons**. The access point sends out a steady "I am here, my name is HomeNet, here's how to talk to me" message about ten times a second. Like a lighthouse blinking. Devices in range scan all the channels, see the beacons, and pick which one to connect to. We'll see this in detail later.

### Why WiFi is a big deal

Before WiFi, the only way to put a computer on the internet was a wire. Apartments had ethernet jacks. Offices had jacks under every desk. If you wanted to use the internet at the kitchen table, you had to drag a long blue wire across the floor. Laptops were not really mobile, because moving meant unplugging.

The first usable consumer WiFi (802.11b, 1999) made laptops actually portable. Then, in 2007, the iPhone shipped with WiFi, and suddenly **every** device wanted WiFi. Now WiFi runs in your fridge. Your doorbell. Your light bulbs. Your scale. Probably your toothbrush. There are more WiFi devices on Earth than there are people, and the number keeps growing.

The IEEE 802.11 standard (which is the official name for WiFi) is one of the most successful pieces of technology in history. So is its little politeness algorithm.

### The OSI/networking-stack picture

WiFi sits at the bottom of the network stack. Picture the stack like a stack of pancakes, with you at the top and the air at the bottom.

```
+------------------------------------------------------+
|  YOU (a person looking at a webpage)                 |
+------------------------------------------------------+
|  Application: HTTPS (a webpage in your browser)      |
+------------------------------------------------------+
|  Transport: TCP (delivers bytes in order)            |
+------------------------------------------------------+
|  Network: IP (routes packets across the internet)    |
+------------------------------------------------------+
|  Data Link: 802.11 MAC (politeness, framing, crypto) |  <- WiFi
+------------------------------------------------------+
|  Physical: 802.11 PHY (radio waves in the air)       |  <- WiFi
+------------------------------------------------------+
|  AIR (electromagnetic spectrum, 2.4/5/6 GHz)         |
+------------------------------------------------------+
```

Each layer talks to the layer right above and right below. The MAC layer is the politeness brain. The PHY layer is the actual antenna firing radio waves. Above WiFi, the network treats it just like any other layer-2 network — IP doesn't know or care that it's flying through air.

The layer-2 frame in WiFi has the same job as the layer-2 frame in Ethernet: get one packet from this MAC address to that MAC address on the same local network. Above layer 2, everyone speaks IP, and everyone forgets the data ever touched a radio.

### A note on terminology

When you read about WiFi, you'll see terms thrown around: SSID vs ESSID, MAC vs BSSID, channel vs frequency, AP vs router. Here's the cheat sheet:

- **SSID** and **ESSID** mean the same thing in casual usage. The standard distinguishes between BSS (one AP) and ESS (many APs sharing the name), but the name is the same.
- **MAC address** is a generic 6-byte hardware ID. **BSSID** is just the MAC of a specific AP, used as a unique-per-AP identifier.
- **Channel** is the human-friendly index (1, 6, 11, 36...). **Frequency** is the actual MHz value (2412, 2437, 2462, 5180...). They map one-to-one.
- **Access Point** is the WiFi radio. **Router** is a layer-3 device. Most home gear combines them into one box and people call the box a "router," but technically the AP is just one part inside.

## The IEEE 802.11 Family

Every version of WiFi is a flavor of the same standard, 802.11. The letter (or letters) at the end say which version. Here are the ones you will hear about.

### 802.11 (1997, the original)

The original WiFi. 2 megabits per second. That's about how fast a 1990s dial-up modem could push if you were lucky. Almost nobody actually used it. It was basically a proof of concept.

### 802.11a (1999)

5 GHz only. Up to 54 Mbps. Was actually faster than the b version that came out the same year, but more expensive, so it didn't win the market.

### 802.11b (1999)

2.4 GHz only. Up to 11 Mbps. Cheap. Compatible. This is the version that actually made WiFi popular.

### 802.11g (2003)

2.4 GHz. Up to 54 Mbps. Compatible with b. Most home routers from the mid-2000s ran this.

### 802.11n (2009) — also called WiFi 4

Both 2.4 and 5 GHz. Introduced **MIMO** (multiple antennas) and channel bonding (gluing two 20 MHz channels together for 40 MHz). Up to about 600 Mbps. Almost every laptop from the early 2010s has this.

### 802.11ac (2013) — also called WiFi 5

5 GHz only. Wider channels (80 and 160 MHz). More antennas. Started doing **MU-MIMO** (talking to several devices at once on different antennas). Up to several gigabits per second on paper. This is the most common WiFi flavor in homes today.

### 802.11ax (2019) — also called WiFi 6

Both bands. Adds **OFDMA** (more on this later) so the access point can talk to many devices in the same airtime, just like a courier delivering a bunch of small packages on one truck instead of one truck per package. Adds **target wake time** so phones can sleep more. Up to 9.6 Gbps theoretical.

### 802.11ax-6E (2021)

Same as WiFi 6 above, but unlocks the brand-new 6 GHz band. The "E" stands for "Extended." Same protocol; just gets to drive on a much emptier highway.

### 802.11be (2024) — also called WiFi 7

Up to 320 MHz channels (twice as wide as WiFi 6). **4096-QAM** encoding (more bits per wave). **Multi-Link Operation** (MLO), so a device can use 2.4, 5, and 6 GHz at the same time on the same connection. Theoretical max around 46 Gbps. As of 2026 the high-end phones, laptops, and access points have it.

### A quick timeline

```
1997  802.11      2 Mbps      original, basically unused
1999  802.11a    54 Mbps      5 GHz, expensive
1999  802.11b    11 Mbps      2.4 GHz, cheap, won the market
2003  802.11g    54 Mbps      2.4 GHz, the dominant home WiFi for years
2009  802.11n   600 Mbps      WiFi 4, both bands, MIMO
2013  802.11ac    ~3 Gbps     WiFi 5, 5 GHz, MU-MIMO
2019  802.11ax   ~9 Gbps      WiFi 6, OFDMA, TWT
2021  802.11ax-6E             WiFi 6E, opens 6 GHz band
2024  802.11be  ~46 Gbps      WiFi 7, 320 MHz, MLO, 4096-QAM
```

Don't memorize the numbers. Memorize the tradeoffs. Newer = wider channels, more antennas, smarter sharing, fancier encoding, and more bands.

## The Frequency Bands

A WiFi radio doesn't pour all its waves out at one note. It uses a specific narrow slice of the radio spectrum, called a **channel.** Each band has its own channels.

### Why are there bands at all?

Radio waves travel at the speed of light, but they need a frequency. Imagine the entire radio spectrum as a giant piano keyboard. Each key is a frequency. Some keys are reserved for radio stations. Others for TV. Others for cell phones. Others for satellites. Others for radar. Others for amateur radio.

Governments (the FCC in the US, Ofcom in the UK, ETSI in Europe) decide who gets which keys. Some keys are reserved for very specific licensed uses. Others, called **ISM bands** (Industrial, Scientific, Medical), are unlicensed: anyone can transmit there as long as they keep power below a limit and accept any interference from other unlicensed users.

WiFi happens entirely on ISM bands. That's why your microwave oven (also on 2.4 GHz, because that frequency happens to wiggle water molecules) can interfere with your WiFi. They're sharing the same swath of unlicensed spectrum.

### A note on power and range

The reason WiFi can't shout across town is regulatory: the FCC limits how much **EIRP** (effective radiated power) you can blast on these bands, typically 1 watt or less. Compare that with a cell tower, which can use tens or hundreds of watts on its licensed spectrum. WiFi is whisper-quiet by law.

Range also depends on **path loss**: signal weakens as it travels, very roughly with the square of distance. Walls, especially metal, absorb a lot. Water absorbs a lot (which is why a fish tank or a packed crowd ruins WiFi). 5 GHz attenuates faster than 2.4 GHz, which is why 2.4 reaches farther even at the same power.

### 2.4 GHz band

Eleven channels in the US (1 through 11). Fourteen in some countries. Each channel is officially 22 MHz wide (in the original spec) or 20 MHz wide (in newer modes), but they're spaced only 5 MHz apart. So adjacent channels overlap heavily.

```
2.4 GHz channels (US, 20 MHz each):

ch1   ch6   ch11
[==]  [==]  [==]
   [..ch2..ch3..ch4..ch5..]   <- these overlap with 1 AND 6
                  [..ch7..ch8..ch9..ch10..]   <- overlap 6 AND 11
```

Notice how only **1, 6, and 11** don't overlap each other. That's why everyone in the world picks 1, 6, or 11 for their 2.4 GHz network. If your neighbor is on channel 6 and you set up on channel 4, you are stomping all over each other. Pick 1, 6, or 11.

### 5 GHz band

Many more channels. Roughly 25 of them, in four sub-bands called **UNII-1**, **UNII-2**, **UNII-2-Extended**, and **UNII-3**. Each channel is 20 MHz, but you can glue them together (channel bonding) into 40 MHz, 80 MHz, or 160 MHz wide channels for more speed.

Some 5 GHz channels are shared with weather and military radar. If your AP uses one of those, the AP has to listen for a radar signal, and if it hears one, jump to a different channel within 10 seconds. That's called **DFS — Dynamic Frequency Selection.** It's why some routers seem to "blink" off the network for a few seconds occasionally; they may be doing a DFS check.

### 6 GHz band (WiFi 6E and 7 only)

Even more channels. About 59 channels of 20 MHz width, which can also be bonded into 40, 80, 160, or in WiFi 7, **320 MHz** wide channels. Almost nothing else uses this band yet, so it's clean and fast.

```
Quick comparison (rough numbers):

Band     Range     Throughput    Wall Penetration   Crowdedness
2.4 GHz  far       low           great              very high
5 GHz    medium    high          okay               medium
6 GHz    short     huge          poor               very low (for now)
```

### Channel widths

When we say "20 MHz channel" we mean the channel is 20 MHz wide on the spectrum. Wider channel = more data per second, but easier to bump into someone else. So you want wide channels in clean bands and narrow channels in crowded bands.

```
20 MHz   |==|                  small but safe (used in 2.4 GHz)
40 MHz   |====|                doubles the bandwidth
80 MHz   |========|            common in 5 GHz at home
160 MHz  |================|    big, often hits DFS in 5 GHz
320 MHz  |================================|  WiFi 7 only, 6 GHz only
```

## Encoding

Sending bits over radio waves means turning a stream of 1s and 0s into actual wiggles in the air. There are several ways to do that.

### OFDM (Orthogonal Frequency-Division Multiplexing)

OFDM is the trick used since 802.11a/g. Instead of squirting one big signal on one frequency, you split your channel into a bunch of tiny **subcarriers** (little narrow slices) and squirt different bits on each one in parallel. Picture a long pipe organ: instead of playing one big note, OFDM plays many small notes at once, each carrying a few bits.

Why? Because radio waves bounce off walls and arrive at the antenna multiple times, slightly delayed. That bouncing causes some frequencies to fade. With OFDM, if one subcarrier is faded, the others are fine. You only lose a little bit. Without OFDM, one fade ruins everything.

### OFDMA (in WiFi 6 and beyond)

OFDM gives one device the whole channel for one chunk of time. **OFDMA** (the A is for "Access") lets the access point split a single airtime slot among **multiple devices.** The AP says, "in this microsecond, phone A gets these subcarriers, laptop B gets those subcarriers, smart bulb C gets these." Suddenly the AP isn't sending one big package per second per device; it's sending dozens of little packages all at once.

Picture a delivery truck. With OFDM, the truck can only deliver to one address per trip. With OFDMA, the truck has shelves inside, and each shelf goes to a different address. One trip, many deliveries.

### QAM (Quadrature Amplitude Modulation)

QAM is how each subcarrier actually carries bits. The wave's amplitude (loudness) and phase (timing) are tweaked into one of N possible "symbols," and each symbol carries a fixed number of bits.

- **BPSK** = 1 bit per symbol
- **QPSK** = 2 bits per symbol
- **16-QAM** = 4 bits per symbol
- **64-QAM** = 6 bits per symbol
- **256-QAM** = 8 bits per symbol (used in 11ac)
- **1024-QAM** = 10 bits per symbol (used in 11ax / WiFi 6)
- **4096-QAM** = 12 bits per symbol (used in 11be / WiFi 7)

Higher QAM = more bits per wave = faster. But also more sensitive to noise: you need a really clean signal, because the gaps between the symbol points get smaller and smaller. That's why your phone uses 4096-QAM only when it's right next to the AP, and falls back to 256 or 64-QAM when you walk away.

## The MAC Layer

WiFi has a **PHY** (physical) layer that actually sends the radio waves, and a **MAC** (Media Access Control) layer that decides who gets to talk and when. The MAC layer is the politeness brain.

### CSMA/CA in detail

Here's the more careful version of the listen-before-talk dance.

1. **Channel sensing.** Before transmitting, the device listens. Specifically: is the energy on the channel above a threshold? If yes, the channel is **busy**. If no, **idle.**
2. **DIFS wait.** When idle, wait a fixed quiet time called **DIFS** (DCF Interframe Space, about 34 microseconds for 802.11g, shorter on newer specs). DIFS makes sure the channel really is quiet, not just briefly silent between packets.
3. **Backoff.** Pick a random number from 0 to (CW - 1), where CW is the contention window. Multiply by **slot time** (about 9 microseconds). Wait that long, listening the whole time. If anyone else starts talking, freeze your countdown until they're done, then keep counting down.
4. **Transmit.** When your countdown reaches 0 and the channel is still idle, send your frame.
5. **Wait for ACK.** The receiver, if all went well, sends an ACK frame back almost immediately (after a tiny gap called **SIFS**, about 16 microseconds).
6. **No ACK?** Assume collision. **Double the contention window.** Try again with a bigger random delay. Keep doubling until you give up (typically after 7 retries).

This is **DCF** — the **Distributed Coordination Function.** "Distributed" means there is no central traffic cop; every device runs the same algorithm and they all play nice together because they all follow the same rules.

### NAV — the "I'm reserving the air" stamp

Sometimes a device knows it's about to send a big chunk of data and wants the air to itself for a while. It sets a **NAV** (Network Allocation Vector) field in its frame headers. The NAV is basically "I'm going to be talking for the next X microseconds; everybody, please stay quiet." Other devices see that, set their own internal "channel busy until X microseconds from now" timer, and don't even try to listen for actual energy. They just wait.

NAV is a virtual channel sensing mechanism, while regular CSMA is a physical one. Modern WiFi uses both.

### A picture of the hidden node

Before we get to the fix, here's the picture again with the airwaves drawn in.

```
   Wall (signal-blocking floor or thick wall)
   ============================================

   [LAPTOP] ))))                       ((((  [PHONE]
        \   energy from laptop  X      energy from phone   /
         \  doesn't reach phone        doesn't reach laptop /
          \                                                /
           \                                              /
            \-----------)) [   AP   ] ((-----------------/
                  AP can hear both, but laptop and phone
                  cannot hear each other.
```

Both devices listen for energy on the air. Both hear silence (because their counterpart is on the other side of a wall). Both transmit at the same time. At the AP, the two signals arrive simultaneously and corrupt each other. The AP gets garbage. Both transmitters never receive an ACK, both retry, both back off, both try again — and the same thing happens. WiFi looks "broken" even though every device followed the rules perfectly.

### RTS/CTS for hidden node

Sometimes two devices can hear the access point but cannot hear each other. Imagine your laptop in the basement and your phone in the attic; both are connected to the AP in the hallway, but a thick floor blocks them from hearing each other directly. From the basement laptop's view, the channel sounds quiet, even when the attic phone is mid-transmission to the AP. Boom: collision at the AP, even though both followed CSMA/CA perfectly.

This is called the **hidden node problem**.

```
   [LAPTOP] -----)))))---X---((((((--- [PHONE]
       \                             /
        \                           /
         \                         /
          \--->  [   AP   ]  <----/
                (everyone reaches AP,
                 nobody reaches each other)
```

The fix is **RTS/CTS** — Request To Send / Clear To Send.

1. Laptop sends a tiny **RTS** frame to the AP saying "I want to send a big frame."
2. AP responds with a **CTS** frame, broadcast for everyone to hear. The CTS says "Laptop has the air for the next X microseconds."
3. Phone, even though it can't hear the laptop, *can* hear the AP, so it hears the CTS and sets its NAV.
4. Laptop sends its big frame. No collision.

RTS/CTS adds overhead (two extra little frames), so it's only used for big frames, controlled by the **RTS threshold** setting on the AP.

### Aggregation: A-MPDU and A-MSDU

Sending one packet at a time is wasteful, because every packet has fixed overhead (preamble, headers, ACK gap). Modern WiFi groups packets together.

- **A-MSDU** (Aggregated MAC Service Data Unit): glues several IP packets together into one big WiFi frame. One header, many packets. If a single bit is corrupted, the whole big frame is lost.
- **A-MPDU** (Aggregated MAC Protocol Data Unit): bundles many separate WiFi frames into one big radio transmission, but each sub-frame still has its own header and its own CRC. If one is corrupted, the others are fine.

Modern WiFi uses both, often together. This is one of the big reasons WiFi 6 and 7 feel snappy.

## Frame Types

Every blip of data in WiFi is a **frame**. There are three big buckets.

### Management frames

These set up and tear down connections. They don't carry user data. Examples:

- **Beacon** — the AP's "I am here" broadcast, sent about 10 times per second on every channel the AP is using.
- **Probe Request** — a device shouting "any APs out there?" Often sent when scanning.
- **Probe Response** — APs replying to a probe request.
- **Authentication frame** — the first step in joining a network. (In WPA-protected networks this part is mostly a formality; the real auth happens later in the 4-way handshake.)
- **Association Request** — "Hello AP, may I join?"
- **Association Response** — "Yes, you may. Here's your slot."
- **Disassociation** — "I'm leaving, bye."
- **Deauthentication** — "We're done. You're cut off." Often used during an attack to kick devices off.

### Control frames

These keep the politeness algorithm running. Tiny frames, very fast.

- **ACK** — "I got your data."
- **RTS / CTS** — described above.
- **PS-Poll** — "Hey AP, I just woke up from power save, do you have anything for me?"
- **Block Ack / Block Ack Request** — used with A-MPDU to ack many frames at once.

### Data frames

Carry actual user data: your IP packets, your TCP segments, your HTTP responses, your video frames, etc. The data frame has a header (with addresses and sequence numbers) and a payload (the IP packet inside).

### What's in the header

The 802.11 frame header is bigger than Ethernet's because it has up to **four MAC addresses**.

```
+------+------+--------+--------+--------+--------+--------+
| Frm  | Dur  | Addr1  | Addr2  | Addr3  | SeqCtl | Payload |
| Ctl  | / ID |        |        |        |        |   ...   |
+------+------+--------+--------+--------+--------+--------+
2 B    2 B    6 B      6 B      6 B      2 B     0..2304 B
```

The four-address case is for special cases like wireless distribution systems (mesh, repeaters), where the frame might be from device A to device D but pass through APs B and C. In a normal client-to-AP frame, you see three addresses: source, destination, and BSSID.

## The Connection Dance

When your phone joins a WiFi network, here is exactly what happens, in order. Watching this once makes WiFi suddenly make sense.

### Step 1: Scan

Your phone sends **probe requests** on every channel it can use ("any AP named HomeNet?") or it just listens passively for **beacons** ("HomeNet, channel 6, BSSID aa:bb:cc:dd:ee:ff, supports WPA3"). Most phones do both.

The result is a list of nearby APs, each with a name (SSID), a MAC address (BSSID), a channel, and a signal strength.

### Step 2: Authenticate (the easy part)

The phone picks an AP and sends an **Authentication frame**. In open and WPA2-PSK networks, this is just a formality — the AP replies with "okay, you're authenticated" without checking anything secret. In WPA3 SAE, the real cryptographic dance happens here (we'll get to that).

### Step 3: Associate

The phone sends an **Association Request** with its capabilities (which encryption modes, which speeds, MIMO support, etc.). The AP replies with an **Association Response** saying "yes you're in, here's your **AID** (Association ID)." Now the phone is **associated**, but if it's a protected network, it can't send actual data yet.

### Step 4: 4-Way Handshake (WPA/WPA2/WPA3)

Now the phone and AP do a four-message exchange that proves they both know the WiFi password without ever sending the password over the air, AND derives a fresh per-session encryption key.

Picture this part as a careful spy meeting:

1. **AP → Phone:** "Here's a random number, the **ANonce**."
2. **Phone → AP:** "Here's my random number, the **SNonce**, plus a hash of [ANonce + SNonce + your MAC + my MAC + the password]. The hash is called a **MIC** (Message Integrity Code)."
3. **AP → Phone:** "I computed the same hash. They match. You know the password. Here's the **GTK** (Group Temporal Key) so you can read broadcast traffic."
4. **Phone → AP:** "Got it."

After step 4, both sides have the same fresh **PTK** (Pairwise Transient Key). All data frames from now on are encrypted with that PTK.

The crucial trick: the password itself never crosses the air. Both sides do the same math from the password and the random numbers, and check that the hashes match. If the password is wrong, the hashes don't match, and you see the famous "WPA: 4-Way Handshake failed" error.

### Step 5: DHCP

Now the phone is on the network at the WiFi layer, but it doesn't have an IP address yet. It sends a **DHCP DISCOVER** broadcast: "any DHCP server out there?" The router (or some other DHCP server) replies with **DHCP OFFER** ("here's an IP address you can have, like 192.168.1.42"), and the phone says **DHCP REQUEST** ("yes please") and the server confirms with **DHCP ACK**. This is the 4-step DHCP dance, completely separate from the WiFi 4-way handshake.

### Step 6: Routable

Now the phone has an IP address and can talk to the internet. From here on it's TCP/IP world. Every packet still flies over WiFi underneath, but at the IP layer the phone is just another device on the LAN.

### A timeline picture

```
PHONE                              AP                  ROUTER
  |                                |                     |
  |---- Probe Request ------------>|                     |
  |<--- Probe Response ------------|                     |
  |                                |                     |
  |---- Authentication ----------->|                     |
  |<--- Authentication OK ---------|                     |
  |                                |                     |
  |---- Assoc Request ------------>|                     |
  |<--- Assoc Response (AID) ------|                     |
  |                                |                     |
  |<--- 4WH Msg 1 (ANonce) --------|                     |
  |---- 4WH Msg 2 (SNonce + MIC) ->|                     |
  |<--- 4WH Msg 3 (GTK + MIC) -----|                     |
  |---- 4WH Msg 4 (Ack) ---------->|                     |
  |                                |                     |
  |---- DHCP DISCOVER ------------>|------- (broadcast)->|
  |<--- DHCP OFFER ----------------|<---- 192.168.1.42---|
  |---- DHCP REQUEST ------------->|------- (request) -->|
  |<--- DHCP ACK ------------------|<---- confirmed -----|
  |                                |                     |
  +---- now routable ---------------------------------+
```

That's it. Every time your phone wakes up and joins WiFi, it does this whole dance in well under a second. If something fails at any step, you get an error that we'll see later.

### The WPA3 SAE handshake (zoomed in)

WPA3 replaces step 4 above with a more careful dance. Both sides commit to a guess at the password without revealing it, then prove they got the same answer.

```
PHONE                                              AP

|---- SAE Commit (a scalar, an element) --------->|
|                                                 |
|<--- SAE Commit (b scalar, b element) -----------|
|                                                 |
|     (each side computes a shared secret K       |
|      from its own guess + the other side's)     |
|                                                 |
|---- SAE Confirm (hash of K + nonces) ---------->|
|                                                 |
|<--- SAE Confirm (hash of K + nonces) -----------|
|                                                 |
|     (if confirms match: same password used,     |
|      shared PMK is derived, no leaks)           |
|                                                 |
|---- 4-way handshake (same as WPA2) ------------>|
```

The clever part is the math. The "commit" message contains values from elliptic-curve operations. An eavesdropper sees the commits and confirms but cannot test password guesses against them, because each guess would require the elliptic-curve operations to be redone *interactively* — which the AP can rate-limit. Brute forcing is back to "online only," which is a huge improvement over WPA2.

This is the **Dragonfly** name: the math has a "dragonfly" curve shape under the hood (well, sort of — the name comes from an earlier draft).

## Security Generations

WiFi security has been through five generations. Each one fixed problems in the one before.

### Open

No encryption at all. Anyone listening can read every packet. Used in coffee shops with captive portals. Modern devices warn you when you connect to one because it's basically a postcard: anyone in range can read it.

### WEP (1997, broken 2001)

Wired Equivalent Privacy. Used a 40-bit (later 104-bit) shared key and the RC4 cipher with a 24-bit IV. Sounded fine in 1997. By 2001, researchers showed the IV scheme leaked enough information that you could recover the key in minutes by capturing enough packets. **Tools like aircrack-ng will crack any WEP network in well under five minutes.** WEP is not a security mechanism. It is a Do Not Disturb sign that any thief can ignore. Don't use it.

### WPA (2003)

WiFi Protected Access. A stopgap to make existing WEP-only hardware safer until WPA2 hardware was ready. Used **TKIP** (a wrapper around RC4 that rotated keys). Better than WEP but still considered weak by modern standards. If your AP only supports WPA, replace it.

### WPA2 (2004)

The big leap. Used **AES-CCMP** (Advanced Encryption Standard in Counter Mode with CBC-MAC) — real strong crypto. Came with mandatory 4-way handshake. Was the dominant home WiFi security for nearly fifteen years.

In 2017, researchers found the **KRACK** attack (Key Reinstallation Attacks) — a flaw in how the 4-way handshake worked when retransmitted, letting an attacker force a key to be reinstalled and possibly decrypt traffic. Patched in OS updates within months. WPA2 is still considered safe if your devices are patched, but it's been superseded.

### WPA3 (2018, mandatory in WiFi 6 certification)

Replaces the 4-way handshake with **SAE (Simultaneous Authentication of Equals)** — also called the **Dragonfly handshake**. SAE uses a different mathematical structure (a "password authenticated key exchange") that doesn't leak any information about the password even if an attacker captures the entire handshake. Brute forcing requires actually trying each guess against the AP, which the AP can rate-limit.

WPA3 also makes **Protected Management Frames (PMF, 802.11w)** mandatory, which prevents the deauthentication attack that's been used to kick devices off WiFi for over a decade.

### OWE (Opportunistic Wireless Encryption / Enhanced Open)

An extension that adds encryption to "open" networks (no password) by doing a Diffie-Hellman key exchange during association. Each client gets its own session key. Doesn't authenticate the AP (so still vulnerable to evil twins) but means a passive eavesdropper can no longer just read everyone's traffic. If your AP supports it, turn it on for guest networks.

### Quick comparison

```
Generation     Cipher         Auth method         Status
Open           none           none                avoid
WEP            RC4 (broken)   shared key          NEVER use
WPA-PSK        TKIP           4-way handshake     deprecated
WPA2-PSK       AES-CCMP       4-way handshake     OK if patched
WPA3-SAE       AES-CCMP/GCMP  SAE/Dragonfly       use this
WPA3-Enterpr.  AES-GCMP-256   802.1X+EAP+SAE      enterprise gold
```

## Enterprise vs Personal

There are two big modes for WPA2 and WPA3.

### Personal (PSK or SAE)

You type a password into your phone. The phone uses that password in the 4-way handshake (WPA2-PSK) or in the Dragonfly SAE handshake (WPA3). Everyone on the network shares the same password.

This is what every home WiFi is. It's also called **WPA2-Personal** or **WPA3-Personal**.

### Enterprise (802.1X + RADIUS + EAP)

No shared password. Each user has their own credentials (a username and password, or a client certificate, or a smart card). Authentication is delegated to a back-end **RADIUS server**.

Picture a hotel with a front desk. In personal mode, every guest knows the same hotel passcode. In enterprise mode, each guest checks in with the front desk separately, and the front desk gives them their own room key.

The pieces:

- **Supplicant** — your laptop (the thing trying to log in).
- **Authenticator** — the access point or switch (the bouncer at the door).
- **Authentication Server** — the RADIUS server (the manager in the back office who actually checks the credentials).

**802.1X** is the standard that defines this three-party dance. **EAP** (Extensible Authentication Protocol) is the language they speak inside it. **RADIUS** is the protocol the authenticator uses to talk to the auth server.

The flow:

```
[Supplicant]         [Authenticator]       [RADIUS Server]
   laptop              access point         back-office auth
      |                     |                       |
      |--- EAPOL Start ---->|                       |
      |<-- EAP Identity ----|                       |
      |--- "alice" -------->|--- RADIUS Access -->  |
      |                     |     -Request          |
      |                     |<-- RADIUS Access ---  |
      |                     |    -Challenge         |
      |<-- EAP Challenge ---|                       |
      | (some EAP-method-specific exchange happens) |
      |--- EAP credentials >|--- RADIUS Access -->  |
      |                     |     -Request          |
      |                     |<-- RADIUS Access ---  |
      |                     |    -Accept            |
      |<-- EAP Success -----|                       |
      | (now do 4-way handshake using PMK from EAP) |
```

After EAP succeeds, the RADIUS server hands the AP a **PMK (Pairwise Master Key)**. The AP and laptop then run the same 4-way handshake we saw earlier, using the EAP-derived PMK instead of one derived from a shared password. From here it looks the same as personal mode.

## Common EAP Methods

EAP is a framework, not a single protocol. There are dozens of EAP methods. Here are the ones you will actually run into.

### EAP-TLS

Both the client and the server present a **certificate**. The server certificate proves the network is real ("yes, this is corporate.example.com's RADIUS, signed by our internal CA"). The client certificate proves the user is real. Mutual authentication. No passwords.

Considered the gold standard. Hard to phish. Requires distributing a per-user client certificate, which is the operational headache.

### EAP-PEAP (Protected EAP)

The server has a certificate; the client doesn't. They first set up a TLS tunnel (like an HTTPS connection), and then the client sends its username and password through the tunnel. Inside the tunnel, the inner EAP method is usually **MS-CHAPv2** (a Microsoft challenge/response).

PEAP-MSCHAPv2 is by far the most common enterprise WiFi auth in business networks because it works with existing Active Directory passwords.

### EAP-TTLS (Tunneled TLS)

Like PEAP, but more flexible about what runs inside the tunnel. Can do plain PAP (username and password), CHAP, MS-CHAP, MS-CHAPv2, etc. Common in non-Windows shops.

### EAP-FAST (Flexible Authentication via Secure Tunneling)

Cisco's homegrown alternative to PEAP. Uses a **PAC** (Protected Access Credential) instead of a server certificate. Faster setup but less standard. Mostly seen in Cisco shops.

### EAP-PWD (Password)

Direct password-based EAP method that doesn't need a TLS tunnel. Uses a Dragonfly-style key exchange. Not widely deployed but it's clean and modern.

### EAP-SIM and EAP-AKA

For phones authenticating to carrier WiFi using their SIM card credentials. You don't usually configure these by hand; the carrier provisions them.

## Roaming

Big networks have many APs covering one big area. Picture a hotel: one AP per floor or one per hallway. The phone walks down the hall, gets out of range of the lobby AP, gets into range of the hallway AP, and somehow has to switch without dropping the call/video/whatever.

That switch is **roaming.**

### BSSID vs ESSID

- **BSSID** (Basic Service Set ID) = the MAC address of one specific AP. Unique per AP.
- **ESSID** (or just SSID, Extended Service Set ID) = the human-readable network name, shared across all APs in one logical network.

So your phone connects to "HomeOffice" (the ESSID) and is currently associated to AP `aa:bb:cc:dd:ee:01` (the BSSID). If you walk to another floor, you might still be on "HomeOffice" (same ESSID) but now associated to `aa:bb:cc:dd:ee:02` (different BSSID). The roaming was: ESSID stayed the same, BSSID changed.

### Plain old roaming

By default, the phone decides when to roam. It watches the signal strength of its current AP. When that gets weak and it sees a stronger AP on the same SSID, it tears down the current association and brings up a new one with the new AP. That includes a fresh 4-way handshake. The whole gap is somewhere between 100ms and 2 seconds, depending on the device. Long enough to drop a VoIP call.

### 802.11r — Fast BSS Transition (FT)

This standard makes roaming much faster. The first time the phone connects to any AP in the **mobility domain**, the AP and the phone exchange enough cryptographic material that the phone can pre-compute future PMKs for any other AP in that domain. When it roams, it just sends a single FT-Action frame with the new key, skipping the full 4-way handshake. Roaming time drops to about 50ms — fast enough that your call stays up.

### OKC — Opportunistic Key Caching

A simpler, vendor-style alternative to 802.11r used in many enterprise networks. The first AP shares the PMK with other APs in the controller's domain. When the phone roams, the new AP already has the PMK, so the 4-way handshake is faster (still happens, just no full re-auth needed).

### 802.11k — Radio Resource Measurement

Lets the AP send the phone a "neighbor report": "here's a list of other APs nearby, on these channels, with these BSSIDs." Saves the phone from scanning all channels itself. Faster decisions.

### 802.11v — BSS Transition Management

Lets the AP say to the phone, "you should probably move to AP X." The phone can take the hint or ignore it. Useful when the AP can see that one BSSID is overloaded.

### 802.11r/k/v together

Modern enterprise WiFi uses all three. The k report tells the phone where to look. The v hint tells the phone when to move. The r/FT mechanism makes the actual move fast.

## Mesh

A mesh is a network made of multiple APs talking to each other wirelessly. No ethernet backhaul between them. They form their own little WiFi network among themselves, and clients see one big SSID.

### 802.11s (the IEEE standard)

Standard mesh networking. APs run a routing protocol called **HWMP** (Hybrid Wireless Mesh Protocol). They figure out which AP can reach which other APs, and which paths are best. New APs can join automatically.

### Vendor proprietary mesh

Most consumer "mesh systems" — eero, Plume, Asus AiMesh, Google Nest WiFi, Netgear Orbi — use their own proprietary protocols. They are not 802.11s. They are 802.11 + some extra logic on top, often using a dedicated 5 GHz radio just for AP-to-AP "backhaul" traffic so the user-facing radio doesn't have to share airtime.

The benefit of vendor systems is they're easier to set up (one app, "add a node, push the button"). The cost is you can't mix brands. An eero satellite won't talk to a Plume base station.

### How a client moves through a mesh

The client sees one SSID. The mesh decides internally which node to attach the client to, based on signal strength and load. When the client walks across the house, the mesh can either let the client roam naturally (using 802.11r/k/v) or actively push the client to a different node ("zero handoff" — the entire mesh pretends to be one giant AP, and frames are forwarded between nodes invisibly).

## WiFi 6/6E/7 Improvements

This section is the headline reel of what's new and why it matters.

### OFDMA — many small deliveries per truck

Already explained above. The big takeaway: WiFi 6 is way more efficient when there are many small devices (smart bulbs, doorbells, watches) sending little packets, because the AP can serve many of them in one airtime slot. Older WiFi would serialize them: one device per slot, big or small.

```
   OFDM (WiFi 5 and earlier):
   +-----------------------------------------------+
   | Slot 1: ALL subcarriers go to phone A         |
   | Slot 2: ALL subcarriers go to phone B         |
   | Slot 3: ALL subcarriers go to phone C         |
   +-----------------------------------------------+
        Phone A waits 2 slots before its turn even
        if it only has 50 bytes to send.

   OFDMA (WiFi 6 and beyond):
   +-----------------------------------------------+
   | Slot 1: subs 1-30 -> A, 31-60 -> B, 61-90 -> C|
   | Slot 2: subs 1-30 -> A, 31-60 -> B, 61-90 -> C|
   +-----------------------------------------------+
        All three phones get airtime in one slot.
```

### MU-MIMO — multiple devices, same time, different antennas

WiFi 5 introduced **downlink** MU-MIMO: the AP could send to multiple clients at the same time using its multiple antennas, each antenna pointing a beam at a different client. WiFi 6 adds **uplink** MU-MIMO: multiple clients can transmit at the same time, with the AP sorting their signals.

For MU-MIMO to work, the client must support it, and the AP needs more antennas than clients in the group. So in practice you get good MU-MIMO when you have a 4x4 or 8x8 AP and a few modern clients in the room.

```
   MU-MIMO simultaneous transmission (downlink):

                       +--------+
                       |   AP   |
                       | 4 ant  |
                       +--/-\---+
                        / | \ \
                  beam1/  |  \ \beam3
                      /   |   \ \
              beam2->/  beam4   \
                    /     |      \
                   /      |       \
              [phoneA] [phoneB] [phoneC]
              gets data gets data gets data
              at same time, different beams
```

Without MU-MIMO, all three phones would have to take turns. With MU-MIMO and good geometry, they share the slot.

### BSS Coloring (spatial reuse)

In dense apartment buildings, your AP and your neighbor's AP share the same channel. Old WiFi would treat any signal on the channel as "channel busy, wait." WiFi 6 adds a per-AP "BSS Color" tag in the preamble. If you receive a frame from a different color AP at low signal strength, you can ignore it and transmit anyway. This is called **OBSS-PD** (OBSS Packet Detection threshold) and it can dramatically improve throughput in dense deployments.

### Target Wake Time (TWT)

The AP and client schedule exact wake times. The client says "I'll be listening at 12:34:56.000 for 2 ms, every 1000 ms." The rest of the time it can deep-sleep its radio. Battery savings on phones, watches, and IoT devices.

### 320 MHz channels (WiFi 7)

Twice as wide as WiFi 6. Only available in the 6 GHz band. In a clean environment, doubles the data rate over WiFi 6E.

### Multi-Link Operation (MLO)

WiFi 7 lets a single client device's connection use multiple bands at once. Your laptop can be on 5 GHz AND 6 GHz at the same time, with packets striped across both. If one band gets noisy, the other carries the load. Or if one band is faster, packets go that way. This is the most important new WiFi 7 feature for real-world performance.

### Preamble Puncturing

WiFi 7 can use a wide channel (say 160 MHz) even when one 20 MHz subchannel inside is busy or has interference. It just "punctures" that subchannel and uses the rest. Older WiFi had to fall back to a narrower channel completely.

## Hidden Networks

A WiFi network whose AP has been told not to broadcast its SSID in beacons. The beacon still goes out (it has to — it's how clients see the AP at all), but with the SSID field blanked.

People often turn this on thinking it's a security feature. **It is not.** Once any client connects, the SSID is in every probe request the client ever sends. Any passive sniffer in range will see the SSID within seconds. Tools like `airodump-ng` will show hidden SSIDs as soon as a client joins.

What hidden networks actually do:

- Annoy users (you have to type the SSID manually).
- Make clients waste battery sending probe requests (because they can't passively wait for a beacon with the right SSID).
- Leak the SSID anyway through probe requests.

If you want security, use WPA3 with a strong passphrase. Hiding the SSID is theatre.

## Common Errors

Verbatim error messages you will see in the real world. If you're staring at one of these in your logs, this is your starting point.

### `Failed to associate`

The AP rejected your association request. Could be: wrong band capability, AP at max client count, MAC filter denying you, or the AP just being overloaded.

### `deauthenticated`

The AP sent a deauth frame. You're off the network. This can mean: signal got too weak, key didn't match, idle timeout, AP rebooted, or someone is running a deauth attack against you.

### `reason=15 (4-way handshake timeout)`

The 4-way handshake didn't complete in time. Almost always means the AP and your device disagree on something cryptographic, often the password.

### `reason=2 (previous authentication no longer valid)`

The AP forgot you. Happens after AP reboot, key rotation issues, or after a deauth. Reconnect.

### `reason=7 (class 3 frame from unassociated)`

You sent a data frame to an AP you weren't associated with. Usually a bug or a roam gone wrong. Common after the AP forgot you but you didn't notice.

### `reason=23 (IEEE 802.1X authentication failed)`

EAP/802.1X failed. Wrong username, wrong password, expired certificate, RADIUS server unreachable.

### `WPA: 4-Way Handshake failed - pre-shared key may be incorrect`

This is the "wrong password" message. Verbatim from `wpa_supplicant`. Fix: re-check the passphrase you typed.

### `CTRL-EVENT-CONNECTED bssid=aa:bb:cc:dd:ee:ff`

You're in. This is the "all is well" message from `wpa_supplicant`. Show your IP after this with `ip addr show`.

### `CTRL-EVENT-DISCONNECTED bssid=aa:bb:cc:dd:ee:ff reason=3`

You got disconnected. The reason code tells you what happened (3 = deauth because leaving). This message keeps repeating if you can't keep a connection up.

### `failed to connect: SSID not found in scan results`

The SSID you asked for isn't visible. Maybe you spelled it wrong, maybe it's hidden, maybe you're out of range, maybe the AP is on a band your radio doesn't support.

### `nl80211: Could not configure driver mode`

`wpa_supplicant` couldn't put your card into the right mode (e.g., couldn't switch from monitor mode back to managed mode). Often fixed by restarting NetworkManager or unloading and reloading the driver.

### `rfkill: WiFi disabled by hardware switch`

There's a physical switch (or function key) on your laptop disabling the radio. Or `rfkill` was called from software. Run `rfkill list` and `rfkill unblock wifi`.

### `wlan0: AP-STA-DISCONNECTED <mac>`

(From hostapd's logs.) An AP-side message that a client just left. Useful for debugging "why does my phone keep falling off the AP?"

### `wlan0: WPA: invalid MIC in msg 2/4 of 4-Way Handshake`

Cryptographic mismatch in the second message of the 4-way handshake. Almost always: wrong password or cipher mismatch.

## Hands-On

This is the section where you actually do things. Open a terminal on a Linux box with a WiFi card and follow along. Each command is paste-ready. The expected output is approximate; yours will look similar.

### Look at your wireless devices

```
$ iw dev
phy#0
        Interface wlan0
                ifindex 3
                wdev 0x1
                addr aa:bb:cc:11:22:33
                ssid HomeNet
                type managed
                channel 36 (5180 MHz), width: 80 MHz, center1: 5210 MHz
                txpower 22.00 dBm
```

`iw dev` lists every wireless interface and what it's doing. "type managed" means normal client mode. The channel and width tell you which slice of which band you're on right now.

### See your current link

```
$ iw dev wlan0 link
Connected to aa:bb:cc:dd:ee:ff (on wlan0)
        SSID: HomeNet
        freq: 5180
        RX: 1842291 bytes (10238 packets)
        TX: 192371 bytes (1234 packets)
        signal: -52 dBm
        rx bitrate: 866.7 MBit/s
        tx bitrate: 866.7 MBit/s
        bss flags:      short-slot-time
        dtim period:    1
        beacon int:     100
```

`signal: -52 dBm` is your signal strength. -30 is "right next to AP, perfect." -67 is "fine for everything." -75 is "video might glitch." -85 is "you've fallen off." Higher numbers (closer to zero) are stronger.

### Scan for nearby APs

```
$ sudo iw dev wlan0 scan | grep -E 'SSID|signal'
        signal: -42.00 dBm
        SSID: HomeNet
        signal: -67.00 dBm
        SSID: Neighbor5G
        signal: -71.00 dBm
        SSID: CoffeeShop
        signal: -83.00 dBm
        SSID: xfinitywifi
```

`iw dev wlan0 scan` walks every channel and shows every AP it can hear. Sometimes you need `sudo`. The output is long; the grep above keeps the useful bits.

### See connected stations (when you're an AP)

```
$ sudo iw dev wlan0 station dump
Station 11:22:33:44:55:66 (on wlan0)
        inactive time:  1240 ms
        rx bytes:       139281
        rx packets:     1023
        tx bytes:       728100
        tx packets:     510
        tx retries:     34
        tx failed:      2
        signal:         -45 dBm
        tx bitrate:     866.7 MBit/s
        rx bitrate:     866.7 MBit/s
        connected time: 8523 seconds
```

Useful when you're running an AP via `hostapd` and want to know who's connected and how strong their signal is.

### Set the channel manually (only works in some modes)

```
$ sudo iw dev wlan0 set channel 36
```

In managed mode (normal client mode), the channel is set by the AP. You can only set it manually in monitor or AP mode.

### Legacy command: iwconfig

```
$ iwconfig
wlan0     IEEE 802.11  ESSID:"HomeNet"
          Mode:Managed  Frequency:5.18 GHz  Access Point: aa:bb:cc:dd:ee:ff
          Bit Rate=866.7 Mb/s   Tx-Power=22 dBm
          Retry short limit:7   RTS thr:off   Fragment thr:off
          Power Management:on
          Link Quality=70/70  Signal level=-40 dBm
          Rx invalid nwid:0  Rx invalid crypt:0  Rx invalid frag:0
          Tx excessive retries:0  Invalid misc:0   Missed beacon:0
```

`iwconfig` is older but still around on most distros. Same info as `iw dev wlan0 link`, different format.

### Legacy scan

```
$ sudo iwlist wlan0 scanning | head -50
wlan0     Scan completed :
          Cell 01 - Address: aa:bb:cc:dd:ee:ff
                    Channel:36
                    Frequency:5.18 GHz (Channel 36)
                    Quality=70/70  Signal level=-40 dBm
                    ESSID:"HomeNet"
                    ...
```

### Get just the current SSID

```
$ iwgetid
wlan0     ESSID:"HomeNet"
```

### NetworkManager command line

```
$ nmcli dev wifi list
IN-USE  BSSID              SSID         MODE   CHAN  RATE        SIGNAL
*       AA:BB:CC:DD:EE:FF  HomeNet      Infra  36    540 Mbit/s  90
        AA:BB:CC:DD:EE:01  Neighbor5G   Infra  149   270 Mbit/s  60
        AA:BB:CC:DD:EE:02  CoffeeShop   Infra  6     65 Mbit/s   45
```

```
$ nmcli dev wifi connect HomeNet password 'mySecretPass'
Device 'wlan0' successfully activated with 'abc12345-...'.
```

```
$ nmcli con show
NAME       UUID                                  TYPE      DEVICE
HomeNet    abc12345-1111-2222-3333-444444444444  wifi      wlan0
Wired      def67890-...                          ethernet  --
```

### Save the password un-encrypted in the connection profile

Useful for headless boxes that can't unlock a keyring at boot.

```
$ nmcli con modify HomeNet wifi-sec.psk-flags 0
```

`psk-flags 0` = "store the PSK in this file in plain text, no agent prompt."

### Interactive TUI

```
$ nmtui
```

Opens an arrow-key interface for picking and editing connections. Useful when you're SSH'd in and don't have a desktop.

### wpa_supplicant directly

If you don't use NetworkManager, you talk to `wpa_supplicant` directly. Many embedded systems and minimalist Linux setups do this.

```
$ sudo wpa_cli status
bssid=aa:bb:cc:dd:ee:ff
freq=5180
ssid=HomeNet
id=0
mode=station
pairwise_cipher=CCMP
group_cipher=CCMP
key_mgmt=WPA2-PSK
wpa_state=COMPLETED
ip_address=192.168.1.42
```

```
$ sudo wpa_cli scan
OK
$ sudo wpa_cli scan_results
bssid / frequency / signal level / flags / ssid
aa:bb:cc:dd:ee:ff   5180  -42  [WPA2-PSK-CCMP][WPS][ESS]  HomeNet
aa:bb:cc:dd:ee:01   5745  -67  [WPA2-PSK-CCMP][ESS]       Neighbor5G
```

```
$ sudo wpa_cli list_networks
network id / ssid / bssid / flags
0       HomeNet         any     [CURRENT]
```

```
$ sudo wpa_cli reconnect
OK
$ sudo wpa_cli disconnect
OK
```

### Start wpa_supplicant manually

```
$ sudo wpa_supplicant -B -i wlan0 -c /etc/wpa_supplicant/wpa_supplicant.conf
```

`-B` = run in background. `-i wlan0` = the interface. `-c /etc/wpa_supplicant/wpa_supplicant.conf` = the config file with your network blocks.

### Run an AP with hostapd

```
$ sudo hostapd -B /etc/hostapd/hostapd.conf
$ sudo hostapd_cli all_sta
11:22:33:44:55:66
11:22:33:44:55:67
```

`hostapd` is the program that turns a Linux box into an access point.

### Monitor mode (for capturing raw 802.11)

```
$ sudo airmon-ng start wlan0
PHY     Interface       Driver          Chipset
phy0    wlan0           iwlwifi         Intel ...
                (mac80211 monitor mode vif enabled for [phy0]wlan0 on [phy0]wlan0mon)
$ sudo airodump-ng wlan0mon
 BSSID              PWR Beacons   #Data, #/s  CH  MB   ENC  CIPHER AUTH ESSID
 AA:BB:CC:DD:EE:FF  -42      45      120  3   36  866  WPA2 CCMP   PSK  HomeNet
 AA:BB:CC:DD:EE:01  -67      33       12  0   149 1300 WPA2 CCMP   PSK  Neighbor5G
```

Monitor mode lets you see all 802.11 traffic in range, not just frames addressed to you.

**Ethics note:** `aireplay-ng --deauth` exists. It sends deauth frames to kick clients off APs you don't control. **Don't run that against networks you don't own.** This sheet mentions it because you'll see it referenced and you should know what it does (and doesn't) do.

### See your WiFi card's capabilities

```
$ iw phy phy0 info | head -50
Wiphy phy0
        max # scan SSIDs: 20
        max scan IEs length: 357 bytes
        Retry short limit: 7
        Retry long limit: 4
        Coverage class: 0
        Available Antennas: TX 0x3 RX 0x3
        Configured Antennas: TX 0x3 RX 0x3
        Supported interface modes:
                 * IBSS
                 * managed
                 * AP
                 * monitor
                 * mesh point
        Frequencies:
                * 2412 MHz [1] (20.0 dBm)
                * 2417 MHz [2] (20.0 dBm)
                ...
        Supported HT capabilities:
                ...
```

```
$ iw phy phy0 info | grep -A 10 'Supported channels'
```

### Check if WiFi is hardware-killed

```
$ rfkill list
0: phy0: Wireless LAN
        Soft blocked: no
        Hard blocked: no
1: hci0: Bluetooth
        Soft blocked: no
        Hard blocked: no
```

If "Hard blocked: yes" — there's a physical switch or a function key disabling the radio. Software cannot un-hardblock; you have to flip the physical switch.

```
$ sudo rfkill unblock wifi
```

This handles soft blocks (those caused by software).

### Bring the interface up or down

```
$ sudo ip link set wlan0 down
$ sudo ip link set wlan0 up
```

### Look at kernel WiFi messages

```
$ dmesg | grep -i wifi | tail -10
[   12.345678] iwlwifi 0000:00:14.0: loaded firmware version 50.7.7
[   12.456789] wlan0: associate with aa:bb:cc:dd:ee:ff (try 1/3)
[   12.567890] wlan0: RX AssocResp from aa:bb:cc:dd:ee:ff
[   12.678901] wlan0: associated
```

### Watch NetworkManager logs in real time

```
$ sudo journalctl -u NetworkManager -f
```

You'll see association attempts, deauths, EAP exchanges, DHCP. Press Ctrl-C to stop.

### Quick stats from sysfs

```
$ cat /sys/class/net/wlan0/wireless/link
70
```

A score from 0 (no link) to about 70 (perfect). Older format than `iw`.

```
$ cat /proc/net/wireless
Inter-| sta-|   Quality        |   Discarded packets               | Missed | WE
 face | tus | link level noise |  nwid  crypt   frag  retry   misc | beacon | 22
 wlan0: 0000   70.  -40.  -256        0      0      0      4      0        0
```

Same idea, different format.

## Common Confusions

Pairs and ideas that trip everyone up.

### 2.4 GHz vs 5 GHz tradeoff

2.4 = farther but slower and crowded. 5 = faster but shorter range. Pick 5 when you can; fall back to 2.4 only when range matters more than speed. There is no "better" — there is "better for what."

### How does WPA2 4-way handshake authenticate without sending the password?

Both sides know the password. They each derive the **PMK** by hashing `password + SSID` 4096 times (PBKDF2). They exchange random nonces. They each compute the same hash of `PMK + nonces + MACs`. They send the hash. If the hashes match, both sides knew the same password. The password itself never goes over the air. The hash does, but you can't reverse a hash to recover the password — *unless* you guess the password and recompute the hash. That's what offline brute-force attacks do.

### SAE / Dragonfly handshake in WPA3

SAE goes one step further. Even if you capture the entire handshake, you cannot test password guesses offline. To test a guess, you have to actually attempt the handshake against the AP, which the AP can rate-limit. This protects weak passwords that would have been crackable under WPA2.

### What does PMF (Protected Management Frames) do?

It cryptographically signs important management frames like deauth and disassoc, so an attacker can't forge a "you're kicked off" frame to your phone. Without PMF, anyone in range can deauth anyone with one command. With PMF (mandatory in WPA3), they can't.

### Why hidden SSID isn't security

The SSID leaks via probe requests from any client that has connected. Tools see it instantly. It's not security; it's annoyance.

### What is "Wireless Isolation" / AP isolation

A switch that prevents clients on the AP from talking to each other. They can talk to the gateway and out to the internet, but not to each other. Useful for guest networks. Also called "client isolation."

### WiFi 6 vs WiFi 6E

**WiFi 6 = 802.11ax on 2.4 and 5 GHz**. **WiFi 6E = 802.11ax with the 6 GHz band added.** Same protocol; 6E just unlocks the new band. WiFi 7 is the next step up (802.11be).

### What does MU-MIMO actually require?

The AP needs more antennas than clients in the group, the client must support MU-MIMO (most modern phones do), they must be in different physical locations relative to the AP (so beams can point apart), and there must be enough simultaneous demand to make grouping worthwhile. In a typical home with one AP and one device active at a time, MU-MIMO does basically nothing.

### BSSID vs ESSID

BSSID = MAC of one specific AP. ESSID/SSID = the name of the whole network, spanning many APs.

### How roaming actually works

The phone's radio decides. It scans periodically, sees a stronger AP on the same SSID, and decides to switch. The switch involves a fresh association on the new AP. With 802.11r/k/v, the switch is faster and smarter.

### What causes "fast roaming"

Specifically: 802.11r FT means the new AP and the phone don't have to redo the full 4-way handshake; they have pre-derived keys. Result: roaming time drops from a couple of seconds to ~50 ms.

### The WPS PIN attack

WiFi Protected Setup with the 8-digit PIN had a flaw: it tested the first 4 digits and the second 4 digits separately, so brute force was 2 * 10^4 = 20,000 attempts instead of 10^8. Tools like Reaver can crack vulnerable APs in a few hours. **Disable WPS on your router.**

### The KRACK vulnerability

A 2017 finding that the WPA2 4-way handshake could be tricked into reinstalling an already-used key when message 3 was retransmitted, which reset the nonce counter and allowed traffic decryption. Patched in OS updates. Still safe if you're up-to-date.

### Why captive portals work via DNS hijack

Coffee shop WiFi uses a captive portal: you connect, but you can't get to the internet until you accept the terms on a web page. The way it works: the AP intercepts your DNS lookups and returns its own IP for any hostname, plus blocks all your other traffic. Your browser tries to load some HTTP page, hits the AP's portal page instead, and shows you the terms. RFC 8910 (capport) defines a cleaner API for this so phones can detect a portal without doing weird HTTP probes.

### The hidden node problem and RTS/CTS solution

Two clients can hear the AP but not each other; they collide at the AP. Solution: before a big transmission, send a tiny RTS to reserve the air, AP replies with CTS that everyone hears, including the hidden node, who then stays quiet.

### Why does my WiFi say "9.6 Gbps" but I get 400 Mbps?

The number on the box is the **PHY rate** with maxed-out antennas, narrowest guard interval, no overhead, and a single client right next to the AP in a noise-free room. Real **throughput** after MAC overhead, ACK gaps, retries, and contention is typically 50–60% of the PHY rate, sometimes much less. This is normal. Always test with `iperf3`, never trust the box number.

### Why does my phone show full bars but pages don't load?

Bars = signal strength (you can hear the AP fine). Pages not loading = something else: AP-to-internet link is down, DNS broken, captive portal not yet accepted, NAT table full, or your IP lease expired. Run `ping 8.8.8.8` to test the network past the AP, and `ping google.com` to test DNS.

### Why does my old laptop fall off WiFi 6 networks?

WiFi 6 APs sometimes drop weak, badly-behaved old clients to keep the network healthy. Check if your AP has a "minimum data rate" or "client steering" setting; that may be what's kicking the laptop off.

### Why does my microwave kill 2.4 GHz when it runs?

Microwave ovens use 2.45 GHz to heat water in food (it's the resonant frequency that makes water molecules wiggle). They leak a tiny bit of that energy out, right in the middle of the 2.4 GHz WiFi band. Channels 6-11 take the brunt. The fix is to use 5 GHz, which is far away from microwave frequencies.

### Why does the AP have multiple BSSIDs in scans?

Modern APs broadcast a separate beacon (and BSSID) for each SSID they offer. So one physical AP with "HomeNet", "HomeNet-Guest", and "HomeNet-IoT" will show three BSSIDs in a scan. Each is computed from the base MAC plus a small offset.

### Why is my speed half what it should be?

WiFi is **half-duplex**: at any moment, the air is either being used for an upload or a download, not both. So a 1 Gbps "rate" really means 1 Gbps shared between the two directions. Plus, MAC overhead, ACKs, and contention eat 30-50% of the airtime. Hitting 60% of advertised PHY rate as actual TCP throughput is a great real-world result.

## Vocabulary

| Term | Plain English |
|---|---|
| WiFi | Ethernet over invisible radio waves; the popular name for IEEE 802.11. |
| WLAN | Wireless Local Area Network. The technical name for a WiFi network. |
| IEEE 802.11 | The standard that defines WiFi. Many sub-standards (a, b, g, n, ac, ax, be...). |
| 802.11a | 1999, 5 GHz, up to 54 Mbps. |
| 802.11b | 1999, 2.4 GHz, up to 11 Mbps. |
| 802.11g | 2003, 2.4 GHz, up to 54 Mbps. |
| 802.11n | 2009, both bands, MIMO, up to 600 Mbps. Also called WiFi 4. |
| 802.11ac | 2013, 5 GHz, MU-MIMO, up to several Gbps. WiFi 5. |
| 802.11ax | 2019, both bands, OFDMA. WiFi 6. |
| 802.11ax-6E | 2021, ax with 6 GHz band added. WiFi 6E. |
| 802.11be | 2024, MLO, 320 MHz, 4096-QAM. WiFi 7. |
| WiFi 4 | Marketing name for 802.11n. |
| WiFi 5 | Marketing name for 802.11ac. |
| WiFi 6 | Marketing name for 802.11ax. |
| WiFi 6E | 802.11ax with 6 GHz band. |
| WiFi 7 | Marketing name for 802.11be. |
| AP | Access Point. The radio device clients connect to. |
| access point | Same as AP. |
| station | A WiFi client device. Often abbreviated STA. |
| STA | Station (a client). |
| BSSID | The MAC address of one specific AP. |
| ESSID | The human-readable name of a WiFi network, possibly spanning many APs. |
| SSID | Same as ESSID in normal use. |
| BSS | Basic Service Set: one AP and its associated clients. |
| ESS | Extended Service Set: multiple APs sharing one SSID, allowing roaming. |
| IBSS | Independent BSS: ad-hoc, no AP, devices talk peer-to-peer. |
| DS | Distribution System: the wired backbone connecting APs in an ESS. |
| frame | A single unit of WiFi data. Like a packet, but at the WiFi layer. |
| beacon | The "I am here" frame an AP broadcasts ~10 times per second. |
| probe request | A frame a client sends asking "any APs out there?" |
| probe response | The AP's reply with details. |
| association request | "May I join?" |
| association response | "Yes, here's your slot." |
| authentication frame | The first formal step in joining; mostly trivial for WPA2-PSK. |
| deauth frame | A "you're cut off" notice. |
| disassoc frame | A "I/we are leaving" notice. |
| ACK | Acknowledgment that a frame was received. |
| RTS | Request To Send: "I want to transmit, please clear the air." |
| CTS | Clear To Send: "Air is yours for X microseconds." |
| PS-Poll | "I just woke up; is anything queued for me?" |
| A-MPDU | Aggregated MPDU: many WiFi frames bundled into one transmission. |
| A-MSDU | Aggregated MSDU: many IP packets in one big WiFi frame. |
| DCF | Distributed Coordination Function — the standard CSMA/CA mode. |
| CSMA/CA | Listen-before-talk politeness algorithm with collision avoidance. |
| NAV | Network Allocation Vector: virtual channel-busy timer set by other devices' frames. |
| CW | Contention Window: random backoff range. |
| backoff | Random delay before transmitting to avoid collisions. |
| hidden node | Two clients can hear AP but not each other; cause of collisions. |
| exposed node | A device hears another nearby talker and stays quiet, even though their transmissions wouldn't actually collide. Mirror of hidden node. |
| OFDM | Orthogonal Frequency-Division Multiplexing: split a channel into many subcarriers, send bits in parallel. |
| OFDMA | OFDM + multiple users per slot; introduced in WiFi 6. |
| BPSK | Binary phase-shift keying: 1 bit per symbol. |
| QPSK | Quadrature PSK: 2 bits per symbol. |
| 16-QAM | 4 bits per symbol. |
| 64-QAM | 6 bits per symbol. |
| 256-QAM | 8 bits per symbol; used in WiFi 5. |
| 1024-QAM | 10 bits per symbol; used in WiFi 6. |
| 4096-QAM | 12 bits per symbol; used in WiFi 7. |
| MIMO | Multiple In Multiple Out: multiple antennas on each side. |
| MU-MIMO | Multi-User MIMO: AP can talk to several clients simultaneously. |
| SU-MIMO | Single-User MIMO: AP-to-one-client multi-antenna mode. |
| beamforming | Aiming the radio energy in the direction of a particular client. |
| spatial streams | Independent data streams transmitted in parallel via separate antennas. |
| channel bonding | Gluing adjacent channels together to make a wider channel. |
| 20 MHz | Default channel width. |
| 40 MHz | Two bonded 20 MHz channels. |
| 80 MHz | Four bonded; common in 5 GHz home use. |
| 160 MHz | Eight bonded; rare due to DFS in 5 GHz. |
| 320 MHz | Sixteen bonded; WiFi 7 only, 6 GHz only. |
| channel allocation 2.4 GHz | Use 1, 6, or 11; the only non-overlapping options. |
| UNII-1 | 5 GHz sub-band: channels 36, 40, 44, 48 (low end). |
| UNII-2 | 5 GHz sub-band with DFS rules. |
| UNII-2-Extended | More 5 GHz channels with DFS rules. |
| UNII-3 | 5 GHz sub-band: channels 149-165 (high end), no DFS in US. |
| UNII-4 | The 5.9 GHz sliver added in some regulatory domains. |
| DFS | Dynamic Frequency Selection: AP must vacate channel if radar detected. |
| TPC | Transmit Power Control: can/must lower power based on conditions. |
| regulatory domain | The set of allowed frequencies and powers in a given country. |
| country code | The two-letter setting that picks the regulatory domain (e.g., US, GB, DE). |
| EIRP | Effective Isotropic Radiated Power: total power including antenna gain. |
| antenna gain | How much an antenna concentrates a signal in one direction (dBi). |
| receive sensitivity | The lowest signal a radio can decode. |
| RSSI | Received Signal Strength Indicator. |
| SNR | Signal-to-Noise Ratio: how loud the signal is vs the background. |
| noise floor | Background radio noise level when no one is talking. |
| MCS index | Modulation and Coding Scheme index: which QAM + coding rate is in use. |
| link rate | The current PHY rate negotiated with the AP. |
| throughput | The actual user-visible data rate after overhead. |
| retry rate | Fraction of frames that needed a retransmission. |
| A-MPDU subframe | A single packet inside an aggregated MPDU. |
| retransmission | Re-sending a frame because no ACK was received. |
| fragmentation | Splitting a big frame into smaller pieces for transmission. |
| RTS threshold | Frame size above which RTS/CTS is used. |
| fragmentation threshold | Frame size above which fragmentation kicks in. |
| beacon interval | Time between beacons; default 100 TU = ~102.4 ms. |
| DTIM interval | How often beacons say "buffered multicast traffic awaits sleeping clients." |
| listen interval | How often a sleeping client wakes to check for beacons. |
| power save mode | Client tells AP it's sleeping; AP buffers frames until it wakes. |
| target wake time | TWT: scheduled wake times for fine-grained battery saving. |
| BSS coloring | Per-AP tag in preamble enabling spatial reuse of overlapping channels. |
| spatial reuse | Letting nearby APs transmit simultaneously when signals are weak enough. |
| OBSS-PD | Overlapping BSS Packet Detection threshold (the spatial reuse mechanism). |
| MLO | Multi-Link Operation: WiFi 7 client uses multiple bands on one connection. |
| EHT-MAC | Extremely High Throughput MAC layer in WiFi 7. |
| EHT-PHY | WiFi 7 PHY layer. |
| OFDMA RU | Resource Unit: the block of subcarriers + time the AP allocates to a client in OFDMA. |
| preamble puncturing | Skipping over a busy/interfered subchannel within a wider channel. |
| TID-to-link mapping | WiFi 7 mechanism mapping traffic categories to specific MLO links. |
| WPA | First-generation WiFi Protected Access; deprecated. |
| WPA2 | Second-generation; AES-CCMP; standard for ~15 years. |
| WPA3 | Third-generation; SAE handshake; PMF mandatory. |
| PMF | Protected Management Frames; 802.11w; signs deauth/disassoc to prevent spoofing. |
| 802.11w | The PMF amendment. |
| SAE | Simultaneous Authentication of Equals; the WPA3 handshake. |
| Dragonfly handshake | Another name for SAE. |
| OWE | Opportunistic Wireless Encryption / Enhanced Open: adds encryption to "open" networks. |
| 4-way handshake | The four-message dance that derives the per-session key in WPA/WPA2. |
| GTK | Group Temporal Key: encrypts broadcast/multicast traffic. |
| PTK | Pairwise Transient Key: per-client per-session encryption key. |
| MIC | Message Integrity Code: authentication tag in handshake messages. |
| EAPOL | EAP Over LAN: the protocol that carries 4-way handshake messages. |
| AES-CCMP | The cipher used in WPA2 and most WPA3 modes. |
| GCMP-256 | A faster modern cipher used in WPA3-Enterprise 192-bit mode. |
| TKIP | Temporal Key Integrity Protocol; weak; deprecated. |
| WEP | Wired Equivalent Privacy; broken since 2001; never use. |
| 802.1X | Port-based access control standard; the framework for enterprise WiFi auth. |
| RADIUS | Remote Authentication Dial-In User Service: the back-end auth protocol. |
| EAP | Extensible Authentication Protocol: the language inside 802.1X. |
| EAP-TLS | Mutual certificate-based EAP. |
| EAP-PEAP | Server cert + tunneled inner method (usually MS-CHAPv2). |
| EAP-TTLS | Server cert + flexible tunneled inner method. |
| EAP-FAST | Cisco's PAC-based EAP. |
| EAP-PWD | Password-based EAP without TLS. |
| EAP-SIM | EAP using GSM SIM credentials. |
| EAP-AKA | EAP using 3G/4G SIM credentials. |
| supplicant | The client side in 802.1X (your laptop). |
| authenticator | The middle (the AP or switch). |
| authentication server | The back-end (RADIUS). |
| captive portal | The "accept terms" page on hotel/coffee-shop WiFi. |
| walled garden | The set of sites you can reach before accepting the captive portal. |
| RFC 8910 | The standard for clean captive portal detection. |
| 802.11r | Fast BSS Transition: pre-derived keys for fast roaming. |
| FT | Fast Transition (the mechanism in 802.11r). |
| 802.11k | Radio Resource Measurement: neighbor reports. |
| 802.11v | BSS Transition Management: AP suggests clients move. |
| 802.11i | The amendment that introduced WPA2. |
| 802.11s | Standard mesh networking for WiFi. |
| mesh network | Multiple APs talking to each other wirelessly to cover an area. |
| mesh portal | A mesh node that also has a wired uplink. |
| mesh point | A mesh node without uplink (relays only). |
| ZeroHandoff | A vendor mesh feature where APs forward client frames so the client never roams. |
| eero | Amazon-owned consumer mesh brand. |
| Plume | Consumer mesh / cloud-managed brand. |
| AiMesh | Asus's mesh feature in their routers. |
| Velop | Linksys's consumer mesh brand. |
| hostapd | Linux user-space program that turns a card into an AP. |
| wpa_supplicant | Linux user-space program for joining WPA-protected networks. |
| NetworkManager | Higher-level Linux network configuration daemon; uses wpa_supplicant under the hood. |
| iwd | Intel's modern alternative to wpa_supplicant. |
| nl80211 | The kernel API used to configure wireless devices on modern Linux. |
| cfg80211 | The kernel framework for wireless config that nl80211 sits on. |
| mac80211 | The kernel framework drivers use to implement most of 802.11 in software. |
| CRDA | Central Regulatory Domain Agent: looks up the regulatory database. |
| regulatory db | The compiled database of allowed frequencies/powers per country. |
| Wireshark | Packet analyzer; can decode 802.11 frames if you have a capture. |
| monitor mode | Card mode where it captures all 802.11 frames in the air, not just yours. |
| RFMON | Same as monitor mode. |
| packet injection | Sending crafted raw 802.11 frames; needed for many security tools. |
| frame injection | Same as packet injection. |
| pcap | Packet capture file format. |
| radiotap | Header format that prepends per-packet radio metadata (signal, freq, etc.) on captures. |
| MAC randomization | Picking a different MAC per network for privacy. |
| DHCP | Dynamic Host Configuration Protocol: how you get an IP after joining. |
| ANonce | Authenticator nonce: AP's random number in the 4-way handshake. |
| SNonce | Supplicant nonce: client's random number in the 4-way handshake. |
| PMK | Pairwise Master Key: shared secret derived from password (or EAP). |
| AID | Association ID: the slot number the AP assigns you when you associate. |
| KRACK | The 2017 attack on the WPA2 4-way handshake. |
| WPS | WiFi Protected Setup: the 8-digit PIN or push-button feature; vulnerable. |
| Evil Twin | A malicious AP impersonating a legitimate SSID. |
| Captive Portal | (also above) the "accept terms" page on a guest network. |
| iperf3 | Tool for measuring real WiFi throughput between two endpoints. |
| airmon-ng | Aircrack-ng tool to put cards into monitor mode. |
| airodump-ng | Aircrack-ng tool to capture and display 802.11 traffic. |
| aircrack-ng | Aircrack-ng's WEP/WPA password cracker. |
| reaver | Tool that exploits the WPS PIN flaw. |
| dragonfly | (also above) the SAE handshake's other name. |

## Try This

Five experiments to deepen your understanding. Pick whichever is interesting.

### 1) See your own beacon

If you have a Linux box with a WiFi card that supports monitor mode, fire up `airmon-ng` and `airodump-ng` and find your own AP. Watch the beacon counter tick up about 10 times per second. Move closer; watch the signal level rise. Move farther; watch it drop. You're literally seeing the lighthouse blink in your terminal.

### 2) Force a roam

If you have two APs on the same SSID and a way to walk between them, watch `iw dev wlan0 link` while you walk. The BSSID line will change when you cross the threshold and the radio decides to switch. Note how long the gap is — try pinging 8.8.8.8 in another terminal and look for the lost ping packets during the switch.

### 3) Watch the 4-way handshake fail

Set up an AP with a known passphrase. Configure your client with the wrong passphrase. Watch `journalctl -u NetworkManager -f`. You'll see the literal `WPA: 4-Way Handshake failed` message we listed in errors above. Now fix the passphrase and watch it succeed.

### 4) Compare 2.4 GHz vs 5 GHz throughput

Run `iperf3 -s` on a wired machine and `iperf3 -c <ip>` on your laptop while connected to the 2.4 GHz SSID. Then connect to the 5 GHz SSID and rerun. The numbers will probably be very different. Move farther from the AP; rerun. 5 GHz drops off much faster.

### 5) Look at all your neighbors

Run `sudo iw dev wlan0 scan` (or `nmcli dev wifi list`). Count how many networks are visible from your home. Now think about how the air is shared. Every one of those networks is using slices of the same 2.4 GHz or 5 GHz band as you. CSMA/CA is getting a workout.

### 6) Watch your MAC randomize

Modern phones and laptops use a different MAC address per WiFi network for privacy. On Linux, set `nmcli con modify <name> 802-11-wireless.cloned-mac-address random` and reconnect. Then run `ip link show wlan0` and compare with the address printed by your card's hardware (`cat /sys/class/net/wlan0/address`). They should differ. Reconnect to a different SSID and watch the address change again. This is what makes it harder for stores and ad networks to track you across networks just by sniffing probe requests.

### 7) Capture a beacon in Wireshark

If you have monitor mode working, run `sudo tcpdump -i wlan0mon -w beacon.pcap "type mgt subtype beacon"`. Let it run for a few seconds. Open the file in Wireshark. You'll see all the AP capability bits laid out: supported rates, channel widths, security suites, vendor-specific tags. This is what your phone reads to decide if it can connect, and what mode to use.

## Where to Go Next

- `ramp-up/tcp-eli5` and `ramp-up/ip-eli5` if you haven't read them yet — they sit on top of WiFi.
- `ramp-up/tls-eli5` for the encryption used above WiFi (your HTTPS lives there).
- `networking/ethernet` for the wired cousin of WiFi.
- `networking/cisco-wireless` for the enterprise gear side of WiFi.
- `security/dot1x` for a deeper look at 802.1X enterprise auth.
- `security/macsec` for layer-2 encryption on Ethernet (a different problem space).
- `offensive/wireless-hacking` if you want the attacker's perspective (purely educational; only use against networks you own).

## See Also

- networking/ethernet
- networking/cisco-wireless
- networking/lacp
- networking/lldp
- security/dot1x
- security/macsec
- offensive/wireless-hacking
- ramp-up/tcp-eli5
- ramp-up/ip-eli5
- ramp-up/tls-eli5
- ramp-up/linux-kernel-eli5

## References

- IEEE 802.11-2020 (and the 2024 maintenance revision) — the canonical standard. Free to read at IEEE GET.
- Matthew Gast, *802.11 Wireless Networks: The Definitive Guide*, O'Reilly. The classic textbook.
- Renderlab and Dragorn at the Renderlab.net writeups on WiFi attacks; classic introductions to WEP/WPA cracking.
- kernel.org wireless wiki — `https://wireless.wiki.kernel.org/` — definitive Linux wireless stack docs.
- hostap.epitest.fi — the hostap project (hostapd and wpa_supplicant), with detailed documentation.
- Megumi Takeshita, *Wireshark for WiFi Analysis*, for hands-on capture interpretation.
- RFC 8910 — Captive-Portal Identification in DHCP and Router Advertisements.
- RFC 5247 — Extensible Authentication Protocol Key Management Framework.
- RFC 7170 — TEAP (Tunneled EAP) for the enterprise-savvy.
- The Wi-Fi Alliance certification pages at wi-fi.org for marketing-name to standard-name mappings.
- Mathy Vanhoef's papers on KRACK, FragAttacks, and Dragonblood for a security researcher's view of the WiFi protocols.
- The Aruba ClearPass docs and Cisco ISE guides for vendor-specific 802.1X deployment patterns.
- "WiFi 6 / WiFi 7" Wi-Fi Alliance certification pages for the latest feature support matrix.
- The freeradius.org documentation for the canonical open-source RADIUS server you'll use for enterprise auth labs.
