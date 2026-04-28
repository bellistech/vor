# Wi-Fi (IEEE 802.11) — Deep Dive

Comprehensive math-heavy reference for Wi-Fi PHY/MAC operation, RF physics, channel
planning, security handshakes, roaming, and capacity engineering. Pair with
`ramp-up/wifi-eli5` for the conceptual primer.

Wi-Fi is a family of layer-1/layer-2 standards under IEEE 802.11 that uses unlicensed
radio spectrum (2.4 GHz, 5 GHz, 6 GHz) to deliver Ethernet-equivalent service over the
air. Every generation since 1997 has rebuilt the PHY (modulation, channel widths,
multi-antenna behavior) while the MAC has accreted features (aggregation, fast
roaming, scheduled access). The math is what separates a deployment that just works
from one that drops packets at 30 clients per AP.

## Standard generations

802.11 amendments alphabetize the order they shipped, not the order on the wire. The
Wi-Fi Alliance retconned marketing names ("Wi-Fi 4" through "Wi-Fi 7") in 2018 to
make consumer messaging tractable.

| Marketing  | IEEE       | Year  | Band(s)              | Max channel | Max spatial streams | Top modulation | Theoretical max PHY rate |
|------------|------------|-------|----------------------|-------------|---------------------|----------------|--------------------------|
| (Wi-Fi 0)  | 802.11     | 1997  | 2.4 GHz              | 22 MHz      | 1                   | DBPSK/DQPSK    | 2 Mbps                   |
| Wi-Fi 1    | 802.11b    | 1999  | 2.4 GHz              | 22 MHz      | 1                   | CCK            | 11 Mbps                  |
| Wi-Fi 2    | 802.11a    | 1999  | 5 GHz                | 20 MHz      | 1                   | 64-QAM         | 54 Mbps                  |
| Wi-Fi 3    | 802.11g    | 2003  | 2.4 GHz              | 20 MHz      | 1                   | 64-QAM         | 54 Mbps                  |
| Wi-Fi 4    | 802.11n    | 2009  | 2.4/5 GHz            | 40 MHz      | 4                   | 64-QAM         | 600 Mbps                 |
| Wi-Fi 5    | 802.11ac   | 2013  | 5 GHz                | 160 MHz     | 8                   | 256-QAM        | 6.93 Gbps                |
| Wi-Fi 6    | 802.11ax   | 2019  | 2.4/5 GHz            | 160 MHz     | 8                   | 1024-QAM       | 9.6 Gbps                 |
| Wi-Fi 6E   | 802.11ax   | 2020  | 2.4/5/6 GHz          | 160 MHz     | 8                   | 1024-QAM       | 9.6 Gbps                 |
| Wi-Fi 7    | 802.11be   | 2024  | 2.4/5/6 GHz          | 320 MHz     | 16                  | 4096-QAM       | 46 Gbps                  |

Notes:

- "Theoretical max" assumes max channel width, max streams, shortest GI, and the
  highest MCS with no PHY overhead. Real-world goodput sits at 50-70% of this even
  in a clean RF environment.
- 802.11ad (60 GHz, "WiGig", 2012) and 802.11ay (60 GHz, 2021) are separate
  short-range high-throughput tracks not in the consumer "Wi-Fi N" naming.
- Wi-Fi 7's 320 MHz channels exist only in the 6 GHz band — the 5 GHz band tops out
  at 160 MHz contiguous.
- Wi-Fi 8 (802.11bn, "UHR" — Ultra-High Reliability) is in draft as of 2025; targets
  reliability rather than headline speed.

### MCS (Modulation and Coding Scheme) tables

MCS index encodes (modulation, coding rate). Multiply by spatial streams (Nss) and
adjust for channel width and guard interval to get the PHY rate. The 802.11n HT
table (1 spatial stream, 20 MHz, 800 ns GI):

| MCS | Modulation | Coding | Data rate (Mbps) |
|-----|------------|--------|------------------|
| 0   | BPSK       | 1/2    | 6.5              |
| 1   | QPSK       | 1/2    | 13.0             |
| 2   | QPSK       | 3/4    | 19.5             |
| 3   | 16-QAM     | 1/2    | 26.0             |
| 4   | 16-QAM     | 3/4    | 39.0             |
| 5   | 64-QAM     | 2/3    | 52.0             |
| 6   | 64-QAM     | 3/4    | 58.5             |
| 7   | 64-QAM     | 5/6    | 65.0             |

802.11ac VHT extends to MCS 8 (256-QAM, 3/4) and MCS 9 (256-QAM, 5/6). 802.11ax HE
adds MCS 10 (1024-QAM, 3/4) and MCS 11 (1024-QAM, 5/6). 802.11be EHT adds MCS 12
(4096-QAM, 3/4) and MCS 13 (4096-QAM, 5/6).

Bits per constellation point:

```
BPSK    = 1 bit
QPSK    = 2 bits
16-QAM  = 4 bits
64-QAM  = 6 bits
256-QAM = 8 bits
1024-QAM = 10 bits
4096-QAM = 12 bits
```

## PHY layer math

### OFDM (802.11a/g/n/ac)

OFDM (Orthogonal Frequency-Division Multiplexing) splits one wide channel into many
narrow subcarriers, each modulated separately. 802.11a/g use 64 subcarriers in a 20
MHz channel — 52 used, 4 pilots, 8 unused (DC + guard).

```
Subcarrier spacing (HT) = 20 MHz / 64 = 312.5 kHz
Symbol duration         = 1 / 312.5 kHz = 3.2 µs
Guard interval (long)   = 0.8 µs
Total symbol time       = 4.0 µs
Useful symbol time      = 3.2 µs (the "FFT period")
Data subcarriers (20 MHz HT) = 52
```

Data rate formula (802.11a/g, 20 MHz, 800 ns GI):

```
Rate = (Ndsc × Nbpsc × R) / Tsym
     = (52 × 6 × 3/4) / 4.0 µs
     = 234 / 4.0 µs
     = 58.5 Mbps  (matches MCS 6)

where:
  Ndsc  = data subcarriers
  Nbpsc = bits per subcarrier (constellation size)
  R     = coding rate (5/6, 3/4, 2/3, 1/2)
  Tsym  = OFDM symbol time including GI
```

802.11n introduces 40 MHz bonded channels, 56 data subcarriers per 20 MHz half (108
total when bonded — slightly better than 2x because the guard between halves is
recovered). 802.11ac extends to 80 MHz (234 data subcarriers) and 160 MHz (468).

### OFDMA (802.11ax+)

OFDMA divides a channel into Resource Units (RUs) — groups of subcarriers — and
assigns RUs to different stations within one transmission. The AP becomes a tiny
LTE-like scheduler.

802.11ax narrows subcarrier spacing 4x to keep RUs precise:

```
Subcarrier spacing (HE) = 78.125 kHz   (was 312.5 kHz)
Symbol duration (FFT)   = 12.8 µs       (was 3.2 µs)
GI options              = 0.8, 1.6, 3.2 µs
Symbol time             = 13.6, 14.4, 16.0 µs (incl. GI)
```

RU sizes in 802.11ax (subcarrier counts):

| RU size       | Subcarriers | Bandwidth (~)  |
|---------------|-------------|----------------|
| RU-26         | 26          | 2 MHz          |
| RU-52         | 52          | 4 MHz          |
| RU-106        | 106         | 8 MHz          |
| RU-242        | 242         | 20 MHz         |
| RU-484        | 484         | 40 MHz         |
| RU-996        | 996         | 80 MHz         |
| RU-2x996      | 1992        | 160 MHz        |

802.11be adds RU-4x996 (320 MHz) and Multi-RU assignment per STA, so a single
station can be allocated, e.g. RU-484 + RU-242.

OFDMA airtime savings example: nine voice clients at 64-byte payloads on 802.11ax
20 MHz can each get an RU-26 in one 20 MHz transmission. On 802.11ac, each client
needed a separate transmission with its own preamble/IFS overhead. OFDMA collapses
preamble overhead 9-to-1 for short frames.

### MIMO and MU-MIMO

Single-User MIMO multiplies the number of independent spatial streams.

```
PHY rate (with Nss streams) = Single-stream rate × Nss
```

Theoretical PHY rate at 802.11ax, 160 MHz, MCS 11, 8 spatial streams, 800 ns GI:

```
1 SS rate = (Ndsc × Nbpsc × R) / Tsym
          = (1960 × 10 × 5/6) / 13.6 µs
          ≈ 1201 Mbps
8 SS rate = 1201 × 8
          ≈ 9608 Mbps  ≈ 9.6 Gbps
```

(`1960` is the data subcarrier count in 160 MHz HE.)

MU-MIMO transmits to multiple receivers simultaneously by spatial separation.
802.11ac introduced DL-MU-MIMO (4 users, 5 GHz only). 802.11ax added UL-MU-MIMO and
extended MU groups. 802.11be supports up to 16 spatial streams across multi-user
groups.

The link math: `MIMO_capacity = log2(det(I + (SNR/Nt) · H · H^H))` (Shannon-MIMO).
The matrix `H` is the channel response between transmit and receive antennas; the
information theoretic capacity scales with min(Nt, Nr) when channels are
independent. In practice cross-coupling reduces effective rank.

### Guard interval

GI is the cyclic prefix that protects against multipath inter-symbol interference.

```
802.11a/g/n long GI:   800 ns
802.11n short GI:      400 ns  (only in good RF; ~11% rate boost)
802.11ax 1x GI:        800 ns  (default)
802.11ax 2x GI:       1600 ns
802.11ax 4x GI:       3200 ns  (long-range / outdoor)
802.11be supports the same options
```

Short GI rate increase example, 1 SS / 20 MHz / MCS 7:

```
Long GI : (52 × 6 × 5/6) / 4.0 µs = 65.0 Mbps
Short GI: (52 × 6 × 5/6) / 3.6 µs = 72.2 Mbps
```

Aggressive short GI helps in clean indoor RF; outdoor or heavy-multipath
deployments must stay long.

## Channel & spectrum

### 2.4 GHz

```
2.4 GHz ISM band (US): 2.400-2.4835 GHz
14 channels:  ch1=2.412 GHz, ch6=2.437 GHz, ch11=2.462 GHz (US uses 1-11)
              ch12-13 = EU/JP only,  ch14 = JP DSSS-only
Channel width: 22 MHz (DSSS/CCK), 20 MHz (OFDM)
Center spacing: 5 MHz between adjacent channel numbers
```

Non-overlapping triple in the US: **1 / 6 / 11**. Each pair is 25 MHz apart with a
20 MHz OFDM channel — leaves a 5 MHz guard. Channels 2-5 partially overlap channel
1 *and* channel 6, so they are noise sources, not usable channels.

EU/JP can use 1/5/9/13 in some cases (4 non-overlapping at 20 MHz) because their
band stretches further. In the US the 25 MHz separation is the maximum legal
spacing.

40 MHz operation in 2.4 GHz is essentially never recommended — bonding 1+5 leaves
nothing for a third AP without massive interference. Most enterprise networks
disable 40 MHz at 2.4 GHz outright.

### 5 GHz

The 5 GHz band is fragmented into U-NII sub-bands with different rules.

| Sub-band   | Frequency range (US)  | Channels (20 MHz)   | DFS required | Max EIRP (US) |
|------------|-----------------------|---------------------|--------------|---------------|
| U-NII-1    | 5.150-5.250 GHz       | 36, 40, 44, 48      | No           | 1 W           |
| U-NII-2A   | 5.250-5.350 GHz       | 52, 56, 60, 64      | Yes          | 250 mW        |
| U-NII-2C   | 5.470-5.725 GHz       | 100-144 (12 ch)     | Yes          | 250 mW (1 W*) |
| U-NII-3    | 5.725-5.850 GHz       | 149, 153, 157, 161, 165 | No       | 1 W           |
| U-NII-4    | 5.850-5.925 GHz       | 169, 173, 177       | No (2020+)   | varies        |

(*) U-NII-2C high-power limits depend on whether the AP is registered with the FCC
and is using a certified outdoor antenna profile.

Channel widths in 5 GHz:

```
20 MHz : every numbered channel
40 MHz : pairs (36+40, 44+48, 52+56, 60+64, 100+104, ...)
80 MHz : (36..48), (52..64), (100..112), (116..128), (132..144), (149..161)
160 MHz: (36..64), (100..128) — only two contiguous in US
80+80  : two non-contiguous 80 MHz blocks (rarely deployed)
```

DFS (Dynamic Frequency Selection) is mandatory in U-NII-2A and U-NII-2C. APs must:

1. Perform a 60-second Channel Availability Check (CAC) before transmitting.
2. Continuously monitor for radar pulses while transmitting (in-service monitoring).
3. Vacate the channel within 10 seconds of radar detection (Channel Move Time + Closing Transmission Time = up to 260 ms).
4. Stay off the channel for 30 minutes (Non-Occupancy Period).
5. Some U-NII-2C channels (120, 124, 128) are weather radar — most enterprise APs avoid these because the long detection window and frequent false positives create instability.

### 6 GHz (Wi-Fi 6E / Wi-Fi 7)

The FCC opened 5.925-7.125 GHz for unlicensed use in 2020 — 1200 MHz of new
spectrum.

| Sub-band  | Frequency range       | Channels (20 MHz) | Notes                             |
|-----------|-----------------------|-------------------|-----------------------------------|
| U-NII-5   | 5.925-6.425 GHz       | 1, 5, 9, ... 93   | Indoor LPI + Standard Power       |
| U-NII-6   | 6.425-6.525 GHz       | 97, 101, 105, 109 | Indoor LPI only                   |
| U-NII-7   | 6.525-6.875 GHz       | 113-181           | Indoor LPI + Standard Power       |
| U-NII-8   | 6.875-7.125 GHz       | 185-233           | Indoor LPI only                   |

Channel numbering uses 1, 5, 9, ..., 233 (4 MHz spacing in numbering, 20 MHz width).
Total 20 MHz channels: 59. Width allocation:

```
20 MHz : 59 channels
40 MHz : 29 channels
80 MHz : 14 channels
160 MHz: 7 channels
320 MHz: 3 channels (Wi-Fi 7 only)
```

Three operating power modes:

- **LPI (Low Power Indoor)**: 5 dBm/MHz PSD (~30 dBm in 320 MHz). Indoor only,
  no external antenna.
- **VLP (Very Low Power)**: -8 dBm/MHz, indoor or outdoor. Designed for mobile and
  AR/VR.
- **Standard Power (SP)**: 23 dBm/MHz with AFC (Automated Frequency Coordination)
  database approval. Allows higher-power operation outdoors and indoors away from
  incumbent fixed satellite users.

AFC requirement: APs running Standard Power must contact a cloud AFC service with
GPS coordinates and a list of channels they want to use. The AFC returns the
allowed channel/EIRP set for that location and the AP must re-query at least every
24 hours.

## A-MPDU and A-MSDU aggregation

Aggregation packs multiple frames into one PHY transmission to amortize preamble
and IFS overhead. Two layers, often used together:

- **A-MSDU** (MAC-level): multiple Ethernet payloads share one 802.11 header and
  one FCS.
- **A-MPDU** (PHY-level): multiple A-MSDU-or-MPDUs share one preamble; each
  sub-frame keeps its own header and FCS.

```
+----------+ +----+----+----+----+----+----+----+----+----+----+ +-----+
| Preamble | | M1 | M2 | M3 | M4 | M5 | M6 | M7 | M8 | M9 | M10| | EOF |
+----------+ +----+----+----+----+----+----+----+----+----+----+ +-----+
   ~20 µs           Each MPDU has its own header+CRC; no per-frame ACK
```

Maximum aggregate sizes:

```
802.11n A-MSDU max: 7935 bytes
802.11n A-MPDU max: 65 535 bytes
802.11ac A-MPDU max: 1 048 575 bytes (~1 MB)
802.11ax A-MPDU max: 4 194 303 bytes (~4 MB)
802.11be A-MPDU max: 15 523 200 bytes (theoretical ceiling)
```

### BlockAck

Single ACKs would erase aggregation gains. BlockAck (BA) acks a window of MPDUs at
once.

```
[A-MPDU with 64 MPDUs] -- SIFS -- [BAR] -- SIFS -- [BlockAck bitmap]
```

The bitmap (256 bits in HT, 1024 bits in VHT/HE/EHT) marks which MPDUs were
received. Senders only retransmit the missing ones.

Per-frame vs aggregated airtime example (1500-byte payload at 65 Mbps PHY):

```
Per-frame:
  Preamble + frame:  ~205 µs
  SIFS:              16 µs
  ACK:               ~30 µs
  DIFS + backoff:   ~70 µs (avg)
  Total:           ~321 µs/frame  → 4.7 Mbps goodput

A-MPDU of 32:
  Preamble + 32 frames: ~6 ms
  SIFS:                 16 µs
  BlockAck:             ~30 µs
  DIFS + backoff:      ~70 µs (avg)
  Total:               ~6.1 ms / 32 frames = 191 µs/frame → 7.85 Mbps goodput
```

The headline gain grows with PHY rate: at 1 Gbps PHY the per-frame transmit time
shrinks but DIFS+SIFS+ACK overhead does not, so single frames sit on 30-40%
efficiency. Aggregation pushes this to 80%+.

## CSMA/CA

CSMA/CA = Carrier Sense Multiple Access with Collision Avoidance. Radios cannot
detect collisions while transmitting (their own TX swamps the receiver), so 802.11
uses pre-transmission sensing plus randomized backoff.

### IFS timing

| IFS          | 802.11g (2.4 GHz) | 802.11a/n/ac (5 GHz) | Purpose                          |
|--------------|-------------------|----------------------|----------------------------------|
| SIFS         | 10 µs             | 16 µs                | Pre-ACK gap; highest priority    |
| Slot time    | 9 µs (short) / 20 µs (long) | 9 µs       | Backoff slot length              |
| PIFS         | SIFS + 1 slot     | SIFS + 1 slot        | Point coordination (HCCA)        |
| DIFS         | SIFS + 2 slots    | SIFS + 2 slots       | Distributed access (BE)          |
| AIFS[AC]     | SIFS + N slots    | SIFS + N slots       | Per-AC priority (EDCA, see below)|
| EIFS         | SIFS + ACK + DIFS | SIFS + ACK + DIFS    | After errored frame              |

EDCA (Enhanced Distributed Channel Access) gives four access categories different
AIFS and contention window parameters.

| AC          | AIFSN | CWmin | CWmax | TXOP limit (5 GHz) |
|-------------|-------|-------|-------|---------------------|
| AC_VO (voice)| 2    | 3     | 7     | 1.504 ms            |
| AC_VI (video)| 2    | 7     | 15    | 3.008 ms            |
| AC_BE (best-effort)| 3 | 15  | 1023  | 0 (one MPDU at a time)|
| AC_BK (background)| 7 | 15  | 1023  | 0                   |

### Contention window math

Backoff slot count is uniformly random in `[0, CW]`. The window doubles after each
collision (binary exponential backoff):

```
CW(retry n) = min(2^n × (CWmin + 1) - 1, CWmax)

Retry 0: CW = CWmin
Retry 1: CW = 2·CWmin + 1
Retry 2: CW = 4·CWmin + 3
...
Retry 5: CW capped at CWmax
```

For AC_BE on 5 GHz:

```
Retry 0: CW = 15        → backoff slots in [0, 15]   → 0-135 µs   (avg 67.5 µs)
Retry 1: CW = 31        → backoff slots in [0, 31]   → 0-279 µs   (avg 139.5 µs)
Retry 2: CW = 63        → backoff slots in [0, 63]   → 0-567 µs   (avg 283.5 µs)
Retry 3: CW = 127       →                            → 0-1143 µs
Retry 4: CW = 255       →                            → 0-2295 µs
Retry 5: CW = 511       →                            → 0-4599 µs
Retry 6: CW = 1023      → CWmax                      → 0-9207 µs  (avg 4.6 ms)
Retry 7+: CW = 1023     → stays at CWmax             → 0-9207 µs
```

Default 802.11 retry limit is 7 short retries (no RTS) or 4 long retries
(RTS/CTS). After the limit the frame is dropped and a TX failure is reported up.

### Channel access state machine

```
┌─[ Idle ]
│   │
│   │ Frame queued
│   ▼
│ ┌─[ Sensing: medium busy? ]──busy──>[ Wait for medium ]
│ │           │                              │
│ │           idle for DIFS+AIFS             ▼
│ │           ▼                       [ Medium becomes idle ]
│ │  [ Pick random backoff slots ]            │
│ │           │                               │
│ │           ▼                               │
│ │  [ Decrement on each idle slot ]<────────┘
│ │   - freeze decrement if medium goes busy
│ │   - resume after medium goes idle for AIFS
│ │           │
│ │     reaches 0
│ │           ▼
│ │   [ Transmit ]
│ │           │
│ │     ┌─────┴────┐
│ │   ACK         no ACK
│ │     │           │
│ │     ▼           ▼
│ │  [ Reset      [ Double CW;
│ │    CW; idle ]   retransmit ]
└─┴─...
```

## RTS/CTS

Hidden-node mitigation. STA A and STA C may both hear the AP but not each other.
Their transmissions collide at the AP. RTS/CTS reserves the channel by exchange:

```
STA → AP : RTS  (Duration field = remaining airtime)
AP  → STA: CTS  (Duration field updated)
[ all STAs that heard the CTS set their NAV (Network Allocation Vector) ]
STA → AP : DATA (transmitted under NAV protection)
AP  → STA: ACK / BlockAck
```

The CTS is the key — every STA that hears the AP's CTS knows the medium is
reserved, even if it did not hear the original RTS.

### RTS threshold

Typical default: 2347 bytes (effectively off). Lowering it engages RTS/CTS for
smaller frames at the cost of overhead:

```
RTS frame   = 20 bytes  → ~36 µs at 6 Mbps base rate
CTS frame   = 14 bytes  → ~28 µs at 6 Mbps
SIFS gaps   = 2 × 16 µs = 32 µs

RTS/CTS overhead: ~96 µs per data frame
```

For a 1500-byte data frame at 65 Mbps, that's ~50% airtime tax. Use RTS/CTS only
when hidden-node losses cost more than the overhead — typically in mesh, dense PtMP
backhaul, or when neighbor APs share a channel.

```bash
# Linux: per-interface RTS threshold
iw dev wlan0 set rts 1024
# Disable
iw dev wlan0 set rts 2347   # or 'off' on some drivers

# hostapd.conf
rts_threshold=1024
fragm_threshold=2346
```

## Beacon timing

Beacons advertise the BSS, carry capability info, and synchronize TSF (Timing
Synchronization Function). The AP picks a Beacon Period (typically 102 400 µs ≈
102.4 ms — the canonical "100 ms" beacon).

### TBTT (Target Beacon Transmission Time)

```
TBTT_n = Beacon_Period × n  (anchored to TSF)
```

Beacons are queued at TBTT but contend for the medium like any other frame. They
back off if the channel is busy. This is why TBTT and actual beacon transmit time
diverge under load — the TSF marks the *intended* moment.

### DTIM and TIM

The TIM (Traffic Indication Map) field in beacons tells associated STAs whether
the AP is buffering downlink frames for them. The DTIM (Delivery TIM) is a
periodic "wake everyone for multicast/broadcast" beacon.

```
DTIM Period: integer count of beacons between DTIMs (typical: 1-3)
DTIM Period × Beacon Period = DTIM interval

  DTIM=1, BP=102.4ms → DTIM interval = 102.4 ms
  DTIM=3, BP=102.4ms → DTIM interval = 307.2 ms
```

A power-save STA can sleep through non-DTIM beacons (just enough to refresh TSF
and check its own bit in the TIM bitmap). It must wake for every DTIM beacon
because that is when buffered multicast/broadcast frames go out.

Engineering guidance:

- Voice handsets: DTIM=1 keeps multicast latency at ~100 ms (matters for SIP
  signaling, push-to-talk).
- Battery-sensitive IoT: DTIM=3-10 trades multicast latency for sleep duration.
- Beacon period 100 ms is the de facto standard. Lowering to 50 ms doubles airtime
  beacon overhead (already 1-2% per AP); raising to 200 ms slows scanning and
  delays roaming.

```bash
# hostapd
beacon_int=100
dtim_period=1

# iw to inspect a STA's view
iw dev wlan0 link
iw dev wlan0 scan
```

## Authentication

802.11 authentication is plumbed through the **4-way handshake**, but the
upstream credential model varies.

### Open

No authentication. Anyone can associate. Modern variants:

- **OWE** (Opportunistic Wireless Encryption, RFC 8110): unauthenticated but
  encrypted — Diffie-Hellman key agreement protects the air. Used in "Enhanced
  Open" by Wi-Fi Alliance.

### WEP (broken — listed for context only)

```
Cipher: RC4
Keys  : 40 bits ("64-bit WEP") or 104 bits ("128-bit WEP")
IV    : 24 bits (per-frame, in the clear)
ICV   : CRC-32
```

Fatal flaws:

- **IV reuse**: 24-bit IV exhausts in 16 M frames (~5 hours on a busy AP). Two
  frames with the same IV and key produce a key-stream collision and reveal both
  plaintexts to XOR.
- **Weak-IV attack** (FMS, 2001): certain IVs leak key bytes. Aircrack-ng can
  recover a 104-bit WEP key with ~50 000 captured frames.
- **CRC-32 is linear**: an attacker can flip plaintext bits and patch the CRC
  without knowing the key.

WEP has been deprecated since 2004. WPA was the interim replacement; WPA2 the
permanent one.

### WPA-PSK / WPA2-PSK

Pre-shared key + per-frame keying.

```
Passphrase (8-63 ASCII chars)
        │
        ▼ PBKDF2-HMAC-SHA1 (4096 iterations, salt = SSID)
        │
        ▼
    PMK (256-bit Pairwise Master Key)
        │
        ▼ 4-way handshake (PRF expand)
        │
        ▼
    PTK (Pairwise Transient Key)
        │
        ├── KCK (Key Confirmation Key, 128 bits) — protects EAPOL-Key MIC
        ├── KEK (Key Encryption Key, 128 bits) — wraps GTK
        └── TK  (Temporal Key) — encrypts data frames
                CCMP-128: 128-bit AES-CCM
                GCMP-256: 256-bit AES-GCM (WPA3 + 802.11ax)
```

PBKDF2 details:

```
PMK = PBKDF2-HMAC-SHA1(passphrase, SSID, 4096, 256)
```

The 4096 iteration count was sized for 2003-era hardware. On modern GPUs an
attacker grinds ~2 million PMK candidates per second per GPU. A 10-character
random passphrase is brute-forceable in years; an 8-character common-word
passphrase falls in hours.

### KRACK (CVE-2017-13077..-13088)

Key Reinstallation Attack. The 4-way handshake message 3 retransmission allows the
attacker to force the supplicant to reinstall the same TK with a reset packet
counter (PN). Reused PNs against AES-CCM yield key-stream reuse and (against
TKIP/WPA-only) full key recovery.

Mitigations:
- Patch supplicants to reject TK reinstallation (Linux/Android: wpa_supplicant
  fixed in 2.6).
- Disable TKIP entirely (use AES-CCMP only).
- Migrate to WPA3 where SAE replaces PSK.

### WPA3-SAE (Dragonfly / Simultaneous Authentication of Equals)

PSK with mutual authentication via password-authenticated key exchange.

```
Both sides:
  1. Convert password → group element via Hunting and Pecking (or hash-to-curve
     in SAE-H2E from WPA3 R3).
  2. Pick random scalar + element, compute commit = scalar·G + element·PE.
  3. Exchange commits.
  4. Compute shared secret z = scalar_local · commit_remote (after subtracting
     the password element correctly).
  5. Derive PMK from z.
  6. Confirm with HMAC.
```

Properties:

- **Forward secrecy**: each session derives a fresh PMK; capturing past traffic
  and learning the password later does not decrypt them.
- **Offline-dictionary-attack resistant**: an attacker who passively records the
  exchange cannot run a dictionary attack against the captured commits.
- **Clogging defense**: SAE includes an anti-clogging cookie to make brute force
  online resource-expensive.

Hash-to-curve (H2E) shipped in WPA3 R3 (2021) to fix timing side channels in the
original Hunting-and-Pecking variant (Dragonblood, CVE-2019-9494, CVE-2019-9495).

### 802.1X / EAP

Enterprise Wi-Fi binds authentication to a RADIUS server (or AAA cluster).
Architecture:

```
[ STA / Supplicant ] <── 802.11 EAPOL ──> [ AP / Authenticator ] <── RADIUS ──> [ EAP server ]
```

Common EAP methods:

| Method     | Inner credentials       | Server cert? | Client cert? | Notes                              |
|------------|-------------------------|--------------|--------------|------------------------------------|
| EAP-TLS    | mutual X.509            | Yes          | Yes          | Strongest; PKI required           |
| EAP-PEAP   | inner PAP/MSCHAPv2      | Yes          | No           | Most common in MSFT shops         |
| EAP-TTLS   | inner PAP/CHAP/MSCHAPv2 | Yes          | No           | More flexible inner methods        |
| EAP-FAST   | PAC                     | optional     | No           | Cisco; PAC provisioning headache   |
| EAP-SIM/AKA| (U)SIM credentials      | No           | No           | Cellular hand-off                  |

Always validate server certificates on the supplicant. An EAP-PEAP/MSCHAPv2 client
that does not check the server cert is trivially MITM-able with `eaphammer` or
`hostapd-mana`.

```bash
# wpa_supplicant.conf for EAP-PEAP
network={
    ssid="CorpWiFi"
    key_mgmt=WPA-EAP
    eap=PEAP
    identity="alice"
    password="hunter2"
    phase2="auth=MSCHAPV2"
    ca_cert="/etc/ssl/corp-root.pem"   # NEVER omit
    altsubject_match="DNS:radius.corp.example"
}
```

## 4-way handshake

After authentication agrees on a PMK, the 4-way handshake proves both sides have
it and derives the per-session PTK.

```
Nonces
  ANonce: AP-chosen random
  SNonce: STA-chosen random
  AA    : Authenticator (AP) MAC
  SPA   : Supplicant (STA) MAC

PTK = PRF-X(PMK, "Pairwise key expansion", min(AA, SPA) || max(AA, SPA)
                 || min(ANonce, SNonce) || max(ANonce, SNonce))

X = 384 bits (CCMP) or 512 bits (TKIP) or 384/512 (GCMP-128/256)
```

Frame walkthrough:

```
M1: AP → STA
    EAPOL-Key
    Key Info: Pairwise=1, Install=0, Ack=1, MIC=0, Secure=0
    Key Replay Counter: r
    ANonce
    Key MIC: 0 (cleared)
    -- STA derives PTK using its own SNonce + received ANonce

M2: STA → AP
    EAPOL-Key
    Key Info: Pairwise=1, Install=0, Ack=0, MIC=1, Secure=0
    Key Replay Counter: r
    SNonce
    RSN IE (proves STA knows the AP's beacon RSN IE — anti-downgrade)
    Key MIC: HMAC over EAPOL frame with KCK
    -- AP now derives PTK using its ANonce + received SNonce
    -- AP verifies MIC with KCK

M3: AP → STA
    EAPOL-Key
    Key Info: Pairwise=1, Install=1, Ack=1, MIC=1, Secure=1
    Key Replay Counter: r+1
    ANonce (echo)
    GTK (encrypted with KEK)
    Key MIC: HMAC with KCK

M4: STA → AP
    EAPOL-Key
    Key Info: Pairwise=1, Install=1, Ack=0, MIC=1, Secure=1
    Key Replay Counter: r+1
    Key MIC: HMAC with KCK
    -- both sides install the PTK and the GTK (group key)
    -- 802.1X port opens: data traffic permitted
```

KCK / KEK / TK split:

```
PTK[0..127]   = KCK   (Key Confirmation Key — MICs the EAPOL frames)
PTK[128..255] = KEK   (Key Encryption Key   — wraps GTK and group keys)
PTK[256..383] = TK    (Temporal Key         — encrypts the data frames)
```

The handshake is also when **GTK rotation** delivers fresh group keys, used to
encrypt multicast and broadcast traffic.

## Roaming

A STA is "associated" to one AP at a time. Roaming is the process of moving
association to a new AP without dropping the connection. Three IEEE amendments
matter:

### 802.11k — Radio Resource Measurement

The AP tells the STA "here are your neighbor APs and their measured load" so the
STA does not have to scan blindly. Useful frames:

- **Neighbor Report Request/Response**: sent by STA, answered by AP with a list
  of nearby APs (BSSID, channel, capability, optional priority).
- **Beacon Report**: STA reports back what it heard.

This shrinks scanning time from ~150 ms (passive scan all channels) to ~30 ms
(probe only the listed channels).

### 802.11v — BSS Transition Management

The AP can *suggest* a STA roam ("you should associate with BSSID X — this
candidate is going to give you better service"). The STA may accept or refuse.

```
BSS Transition Management Request:
  Disassociation Imminent: 1 (or 0)
  Disassociation Timer: ~5 s
  BSS Termination Included: 0
  Candidate List:
    BSSID 1, Score, Phy, Channel
    BSSID 2, Score, Phy, Channel
    ...
```

"Sticky" clients ignore these. Aggressive deployments combine 11v hints with
forced disassociation (the AP eventually kicks the client) — be wary, it breaks
poorly written supplicants.

### 802.11r — Fast BSS Transition (FT)

Pre-authentication and key caching. Without 11r, every roam re-runs the 4-way
handshake (typically ~120 ms). With 11r, the keys are mobile within the mobility
domain.

```
Key hierarchy:
  MSK (from EAP)  →  PMK-R0 (one per Mobility Domain)
                    │
                    ├── PMK-R1[AP1] = KDF(PMK-R0, AP1 BSSID)
                    ├── PMK-R1[AP2] = KDF(PMK-R0, AP2 BSSID)
                    ├── PMK-R1[AP3] = KDF(PMK-R0, AP3 BSSID)
                    └── ...

When roaming AP1 → AP2:
  STA sends Auth Req with FT IE  →  contains PMK-R1[AP2] derivation seed
  AP2 already has PMK-R1[AP2] from the R0 holder
  4-way handshake folded into reassociation
  Roam latency: ~30 ms (vs ~120 ms baseline)
```

11r requires careful planning: STAs and APs must agree on FT method (over-the-air
vs over-the-DS), and the mobility domain ID must match. Some legacy clients (and
many printers) refuse to associate with an FT-only SSID — the typical workaround
is dual-SSID (one with FT, one without) or 11r mixed mode (both FT and non-FT
allowed).

### Roaming triggers

A STA roams when the current link quality crosses internally-configured
thresholds. Common heuristic:

```
Roam if:
  RSSI < -75 dBm (poor signal)
  AND a candidate AP exists with RSSI > current_RSSI + 10 dB

Or:
  RSSI < -85 dBm (severely poor — roam to anything in range)

Or:
  Retry rate > 30% over 1 second (link quality degrading even with strong RSSI)
```

The 10 dB sticky threshold prevents flapping. Some vendors expose this as a
"roaming aggressiveness" slider; on Intel Wi-Fi cards it is `RoamingAggressiveness`
1-5 (Lowest to Highest).

## Mesh — 802.11s

802.11s defines a self-forming, self-healing mesh of MPs (Mesh Points), MAPs
(Mesh APs that also serve clients), and MPPs (Mesh Portal Points that bridge to
wired networks).

Path selection: HWMP (Hybrid Wireless Mesh Protocol).

- **Reactive** (on-demand): PREQ (Path Request) flooded; matching MP responds with
  PREP (Path Reply). Routes installed as needed. Similar to AODV.
- **Proactive**: root MP sends periodic PREQs; reverse paths to root pre-installed.
  Useful when most traffic flows toward an MPP.

Path metric: airtime link metric.

```
ca = O + (Bt / r) · (1 / (1 - ef))

where:
  O   = channel access overhead per frame (IFS + backoff + ACK)
  Bt  = test frame size
  r   = link rate in Mbps
  ef  = frame error rate
```

The lower `ca`, the better the link. Path cost is the sum of `ca` along the route;
HWMP picks the minimum-cost path.

Mesh frames have an extra Mesh Control field that adds time-to-live, sequence
number, and address extension (4-address mode is mandatory for mesh).

## RF math

### Free-space path loss (FSPL)

```
FSPL(dB) = 20·log10(d) + 20·log10(f) + 32.44
   d in km, f in MHz

Equivalent for d in m, f in GHz:
FSPL(dB) = 20·log10(d) + 20·log10(f) + 32.45 + 60   (offset because of unit shift)
```

Per-doubling rule: every doubling of distance adds 6 dB of path loss.
Per-band rule: every doubling of frequency adds 6 dB.

Example, 5 GHz at 100 m:

```
FSPL = 20·log10(0.1) + 20·log10(5000) + 32.44
     = 20·(-1) + 20·(3.699) + 32.44
     = -20 + 73.98 + 32.44
     = 86.42 dB
```

Same distance at 2.4 GHz:

```
FSPL = 20·log10(0.1) + 20·log10(2400) + 32.44
     = -20 + 67.60 + 32.44
     = 80.04 dB
```

The 6 dB difference between bands is why 2.4 GHz "reaches further" — for the same
TX power and antenna pattern, 2.4 GHz arrives 6 dB stronger than 5 GHz at 100 m.
The actual range advantage is offset by the smaller channel widths and worse SNR
math at 2.4 GHz, so coverage equivalence is design-dependent.

### Link budget

```
RxPower (dBm) = TxPower (dBm)
              + Gtx     (transmit antenna gain, dBi)
              + Grx     (receive antenna gain, dBi)
              − PathLoss (dB)
              − Cable/connector losses (dB)
              − Fade margin (dB, typical 10-15 dB)
```

EIRP (Effective Isotropic Radiated Power):

```
EIRP (dBm) = TxPower + Gtx − cable losses
```

Regulatory caps EIRP, not raw TX power. US 5 GHz U-NII-3: 1 W = 30 dBm EIRP for
omnidirectional, or up to 53 dBm EIRP for fixed point-to-point (with antenna
gain trade rules — PtP gets 1 dB EIRP added per 3 dB of antenna gain above 6 dBi).

### Sensitivity vs MCS

Higher modulations need higher SNR. 802.11ax indoor sensitivity targets (per
spec, dBm referred to the receive antenna):

| MCS | Modulation | Min SNR | Sensitivity (20 MHz) |
|-----|------------|---------|----------------------|
| 0   | BPSK 1/2   | 2 dB    | -82 dBm              |
| 1   | QPSK 1/2   | 5 dB    | -79 dBm              |
| 2   | QPSK 3/4   | 9 dB    | -77 dBm              |
| 3   | 16-QAM 1/2 | 11 dB   | -74 dBm              |
| 4   | 16-QAM 3/4 | 15 dB   | -70 dBm              |
| 5   | 64-QAM 2/3 | 18 dB   | -66 dBm              |
| 6   | 64-QAM 3/4 | 20 dB   | -65 dBm              |
| 7   | 64-QAM 5/6 | 25 dB   | -64 dBm              |
| 8   | 256-QAM 3/4| 29 dB   | -59 dBm              |
| 9   | 256-QAM 5/6| 31 dB   | -57 dBm              |
| 10  | 1024-QAM 3/4| 34 dB  | -54 dBm              |
| 11  | 1024-QAM 5/6| 36 dB  | -52 dBm              |

Wider channels reduce sensitivity by 3 dB per doubling (more thermal noise
captured by the wider receiver). 160 MHz sensitivity at MCS 0 is roughly -73 dBm
(vs -82 dBm at 20 MHz). This is why "wider isn't always better" — 160 MHz only
helps if your RSSI is strong enough to clear the higher noise floor.

### Receiver noise floor

```
N (dBm) = -174 + 10·log10(BW Hz) + NF
        = -174 + 10·log10(20 000 000) + 7
        ≈ -174 + 73.0 + 7
        = -94 dBm     (20 MHz, 7 dB noise figure)
```

160 MHz: `N ≈ -85 dBm`. 320 MHz: `N ≈ -82 dBm`. The Wi-Fi 7 SNR challenge for
4096-QAM is real — you need 36+ dB of SNR over a 320 MHz noise floor at -82 dBm,
i.e. RxPower ≥ -46 dBm, which is roughly 5 m line-of-sight from a ceiling AP.

## Scanning

### Passive scan

The STA dwells on each channel and listens for beacons.

```
Dwell time: typically 100-150 ms per channel (≥1 beacon period)
Channels (5 GHz, US):  ~25 channels   →  ~3.5 s total scan
Channels (2.4 + 5 GHz):                →  ~5 s total scan
```

Passive scan is mandatory on DFS channels (regulatory: a STA may not actively
probe before knowing the channel is available).

### Active scan

The STA sends a Probe Request and listens briefly for Probe Responses.

```
Dwell time: ~30 ms per channel
Probe Request: broadcast or directed to specific SSID
Probe Response: per-AP unicast
```

Active scan is faster but illegal on DFS channels until the STA has heard a
beacon there.

### Fast scan strategies

802.11k Neighbor Reports tell the STA which channels to probe; cuts scan time
to ~30 ms. Some vendors (Apple, Intel) implement background scans that interleave
~10 ms scans with normal traffic to keep neighbor knowledge fresh.

```bash
# Linux: trigger a scan and dump results
iw dev wlan0 scan
# Show last RSSI per neighbor
iw dev wlan0 scan dump | awk '/BSS|signal|SSID/'
# Watch link metrics
watch -n1 'iw dev wlan0 link'
```

## Common operational issues

### Sticky clients

A STA stays associated with a distant AP even though a closer one would serve
better. Causes:

- Hysteresis threshold too aggressive (radio designed for laptops in a coffee
  shop, not enterprise mobility).
- 802.11k/v not enabled.
- Driver bug.

Mitigation:

- Enable 11k/11v hints.
- Lower minimum-RSSI on the AP (force the AP to deauth STAs below, e.g., -75 dBm
  so they roam).
- Reduce AP TX power so cell sizes shrink and STAs face cleaner roam decisions.

### Rogue APs

An unauthorized AP advertises the corporate SSID to capture credentials. WIPS
(Wireless Intrusion Prevention) detects:

- BSSID not in the trusted list.
- Beacons from authorized SSIDs not from authorized APs.
- Channel mismatches.
- STAs probing for known SSIDs (KARMA-style attacks).

Counter-measures:
- Containment via crafted deauth/disassoc frames (legally fraught — the FCC has
  fined enterprises for "containing" guest hotspot APs in conference centers).
- 802.1X EAP-TLS with strict server-cert validation.
- Switch port lockdown to prevent rogue uplinks.

### Co-channel interference (CCI)

Two APs on the same channel hearing each other share the medium via CSMA/CA.
The penalty is roughly:

```
Effective throughput per AP ≈ Single AP capacity / N

where N = number of APs on the same channel within hearing range (~ -85 dBm)
```

So six APs on channel 1 in 2.4 GHz get one-sixth the throughput each.

Adjacent-channel interference (ACI) is worse: APs on overlapping but different
channels do not back off, so transmissions collide. ACI is why "channel 3" in 2.4
GHz never works.

### DFS radar events

A radar pulse on a U-NII-2A or U-NII-2C channel forces the AP to move within
seconds. STAs lose association for the move (no smooth migration unless the AP
implements "Channel Switch Announcement" beacons in advance — and only some STAs
honor them).

Mitigation if DFS instability is operational:

- Disable U-NII-2C (channels 100-144) on installations near airports, weather
  radar, or shipping ports.
- Use long-tenure 5 GHz channels (36-48 in U-NII-1, 149-165 in U-NII-3) for
  voice and other latency-sensitive workloads.
- Move VoIP to 6 GHz where DFS does not apply (yet).

### Hidden node problem

```
   STA-A ──────→ AP ←────── STA-C
        (heard but not by each other)
```

Both STAs sense the medium independently; CSMA/CA does not save them. RTS/CTS
mitigates by reserving via the AP. Other tactics:

- Reduce cell size so all STAs are close enough to the AP to hear each other.
- Use higher TX power on STAs (rare — usually capped by hardware/regulatory).
- Place a wired AP in the dead zone.

### Exposed node problem (the dual)

A STA suppresses transmission because it hears another transmitter, even though
its own destination is in a different direction. CSMA/CA over-protects and wastes
airtime. Cannot be fixed without 802.11ax spatial reuse / OBSS-PD (next).

### OBSS-PD (802.11ax spatial reuse)

The receiver can decode a frame even with an "interfering" overlapping-BSS frame
if the interfering signal is below a programmable threshold. Spatial reuse uses
this to allow simultaneous transmissions in dense deployments.

```
OBSS-PD threshold: -82 to -62 dBm (configurable)

If the AP detects an OBSS preamble below the threshold, it ignores the
"medium busy" indication and transmits anyway. Boosts dense-AP capacity 30-100%
but requires careful tuning — too aggressive causes mutual interference.
```

## Worked examples

### Example 1: Wi-Fi 6, 4×4 MU-MIMO, 160 MHz, MCS 11

```
Spec values:
  Ndsc (160 MHz HE)  = 1960 data subcarriers
  Nbpsc (1024-QAM)   = 10 bits/subcarrier
  Coding rate (MCS 11) = 5/6
  Tsym (HE 1x GI)    = 13.6 µs
  Spatial streams (Nss) = 4

Per-stream PHY rate = (Ndsc × Nbpsc × R) / Tsym
                    = (1960 × 10 × 5/6) / 13.6 µs
                    = 16 333.3 / 13.6
                    ≈ 1201.0 Mbps

PHY rate (4 × MIMO) = 4 × 1201.0 = 4804 Mbps ≈ 4.8 Gbps

Sustained TCP goodput estimate: 0.65 × 4800 ≈ 3.1 Gbps
   (35% PHY/MAC overhead: A-MPDU efficiency, IFS, BlockAck, retries)
```

### Example 2: Outdoor PtP at 5 km, 5 GHz

```
Givens:
  TxPower      = 24 dBm
  Tx antenna   = 24 dBi parabolic dish
  Rx antenna   = 24 dBi parabolic dish
  Cable losses = 1 dB each end
  Frequency    = 5500 MHz
  Distance     = 5 km
  Fade margin  = 12 dB

FSPL = 20·log10(5) + 20·log10(5500) + 32.44
     = 20·(0.699) + 20·(3.740) + 32.44
     = 13.98 + 74.81 + 32.44
     = 121.23 dB

EIRP = 24 + 24 − 1 = 47 dBm   (well below 53 dBm cap for PtP at 5 GHz)

Rx Signal = TxPower + Gtx + Grx − PathLoss − Cable losses
          = 24 + 24 + 24 − 121.23 − 2
          = -51.23 dBm

Rx noise floor (20 MHz)  = -94 dBm
SNR = -51.23 − (-94) = 42.77 dB
   → comfortably enough for MCS 9 (256-QAM 5/6 needs 31 dB)
   → with 12 dB fade margin: usable SNR floor still 30 dB → MCS 9 in clear weather

Predicted PHY rate (1 SS, 20 MHz, MCS 9, 800 ns GI):
  (52 × 8 × 5/6) / 4.0 = 86.7 Mbps
With 2x2 MIMO: ≈ 173 Mbps PHY → ≈ 110 Mbps TCP goodput.
```

### Example 3: 100 clients on 2.4 GHz vs 5 GHz

Assume average client demand = 1 Mbps sustained, 5 Mbps peak. Single AP per band,
typical office RF.

```
2.4 GHz:
  Usable channels: 1, 6, 11 (only 1 reusable per AP)
  Best case PHY rate: 802.11n 40 MHz disabled → MCS 7, 1 SS = 65 Mbps
  Aggregate airtime budget per AP: ~50% (CSMA, retries, beacons, 1 Mbps base
    rate management overhead)
  Effective per-AP capacity: ~30 Mbps
  100 clients × 1 Mbps sustained = 100 Mbps demand
  → CANNOT SUPPORT  (need 4 APs minimum, but only 3 channels exist; CCI eats
    50%+ of any expansion)

5 GHz (Wi-Fi 6, 80 MHz):
  Channels available: 25 (US, all 80 MHz blocks)
  PHY rate: MCS 9, 2 SS, 80 MHz, 800 ns GI ≈ 1200 Mbps
  Effective per-AP capacity (MU-MIMO + OFDMA): ~600 Mbps
  100 clients × 1 Mbps = 100 Mbps → 2-3 APs at 80 MHz
  Channel reuse via cell planning: comfortable
  → SUPPORTABLE
```

The conclusion isn't unique to this example: 2.4 GHz is for IoT and legacy
clients only; capacity planning lives in 5 GHz and 6 GHz.

## Operational commands

### Linux iw / iwconfig / hostapd

```bash
# Show interface info
iw dev wlan0 info
iw dev wlan0 link

# Scan
iw dev wlan0 scan
iw dev wlan0 scan freq 2412   # only channel 1

# Set channel manually (interface must be down or in monitor mode)
iw dev wlan0 set channel 36 HT40+
iw dev wlan0 set channel 100 80MHz

# Power save
iw dev wlan0 set power_save off

# Monitor mode capture
iw phy phy0 interface add mon0 type monitor
ip link set mon0 up
tcpdump -i mon0 -y IEEE802_11_RADIO -w air.pcap

# hostapd minimal AP (5 GHz, WPA3)
cat <<'EOF' >/etc/hostapd/hostapd.conf
interface=wlan0
driver=nl80211
ssid=corp-wifi
hw_mode=a
channel=36
ieee80211n=1
ieee80211ac=1
ieee80211ax=1
country_code=US
ht_capab=[HT40+][SHORT-GI-40][TX-STBC][RX-STBC1]
vht_capab=[VHT80][SHORT-GI-80]
he_oper_chwidth=1
wpa=2
wpa_key_mgmt=SAE
sae_password=correct horse battery staple
rsn_pairwise=CCMP
ieee80211w=2  # mandatory MFP for SAE
EOF
hostapd -dd /etc/hostapd/hostapd.conf

# wpa_cli inspection
wpa_cli -i wlan0 status
wpa_cli -i wlan0 signal_poll
wpa_cli -i wlan0 list_networks
wpa_cli -i wlan0 reassociate
```

### Spectrum analysis

```bash
# AP-side: vendor tools (Cisco "show spectrum", Aruba "show ap monitor spectrum")
# Linux: airodump-ng for survey
airmon-ng start wlan0
airodump-ng wlan0mon --band abg --write survey
# Off-the-shelf hardware: Ubiquiti AirView, MetaGeek WiSpy, Ekahau Sidekick

# Channel utilization (Linux):
iw dev wlan0 survey dump
# Shows: channel time, channel time busy, channel time receiving, channel time
# transmitting, noise — per channel.
```

### Capture analysis with Wireshark

Filter snippets:

```
# Beacons from a specific BSS
wlan.fc.type_subtype == 0x08 && wlan.bssid == aa:bb:cc:dd:ee:ff

# 4-way handshake
eapol

# Probe responses for a SSID
wlan.fc.type_subtype == 0x05 && wlan.ssid == "corp-wifi"

# Failed deauths
wlan.fc.type_subtype == 0x0c

# Just RSN-protected traffic (data frames after handshake)
wlan.fc.protected == 1
```

## Capacity engineering checklist

1. **Radio plan** — spectrum analyzer survey before anything else. RSSI > -65
   dBm in all client zones, SNR > 25 dB.
2. **Channel widths** — 20 or 40 MHz at 2.4 GHz, 40 or 80 MHz at 5 GHz, 80-160
   MHz at 6 GHz. Avoid 160 MHz unless AFC or LPI gives you headroom and clients
   support it.
3. **AP spacing** — typical office: 1 AP per 1500-2500 ft² (140-230 m²); high
   density (auditoriums): 1 AP per 250-500 ft² (23-46 m²).
4. **TX power** — match AP and STA power. STAs typically cap at 10-15 dBm; APs
   at 20-30 dBm. An AP at 30 dBm with STAs at 10 dBm creates an asymmetric link
   where STAs hear the AP but the AP barely hears them (low RSSI uplink → low
   MCS uplink → wasted airtime).
5. **Min basic rate** — set to 12 Mbps or higher to drop legacy slow clients
   (saves airtime for everyone else; old IoT may break).
6. **DTIM / beacon interval** — 1-3 / 100 ms unless dense IoT changes the
   trade-off.
7. **Roaming features** — enable 11k, 11v, 11r where the supplicant supports it.
   Test with realistic clients before rolling out.
8. **Security** — WPA3-SAE Personal or WPA3-Enterprise EAP-TLS. PMF mandatory.
   Disable TKIP and WEP entirely.
9. **Monitoring** — track per-AP channel utilization, RX retries, MCS
   distribution, MFP rejections, DFS events. Baseline weekly; alert on shifts.

## Vendor configuration snippets

### Cisco IOS-XE WLC (9800)

```
wireless profile policy POL-corp
  no shutdown
  vlan 100
  radius-profiling
  fast-roam
  dhcp-tlv-caching
  http-tlv-caching
  ipv4 dhcp required

wireless tag policy TAG-corp-pol
  description "Corp default"
  wlan corp-wifi policy POL-corp

wlan corp-wifi 1 corp-wifi
  no shutdown
  client vlan 100
  no security wpa wpa2
  security wpa wpa3
  security wpa psk set-key ascii 0 "correct horse battery staple"
  security pmf mandatory

ap profile DEFAULT-AP
  ssh
  radio-policy 5ghz
  radio-policy 6ghz
  radio-policy 2dot4ghz
  capwap retransmit timers 5 3
```

### Aruba ArubaOS-CX

```
wlan ssid-profile corp-wifi
   essid corp-wifi
   wpa-passphrase "correct horse battery staple"
   opmode wpa3-aes-ccm-128
   mfp-required

wlan virtual-ap VAP-corp
   ssid-profile corp-wifi
   aaa-profile aaa-corp
   forward-mode tunnel
   vlan 100

ap-group corp-aps
   virtual-ap VAP-corp
   regulatory-domain-profile US
```

### MikroTik RouterOS

```
/interface wireless security-profiles
add name=wpa3 mode=dynamic-keys authentication-types=wpa3-psk \
    wpa3-pre-shared-key="correct horse battery staple" \
    management-protection=required

/interface wireless
set wlan1 ssid=corp-wifi country=united-states3 frequency=5180 \
    band=5ghz-ax wireless-protocol=802.11 \
    channel-width=20/40/80mhz-XXXX security-profile=wpa3 \
    disabled=no
```

## Counter-intuitive tunings

- **Lower TX power often improves coverage.** A loud AP creates loud STAs
  whose retransmits drown each other. Reducing AP TX from 23 dBm to 17 dBm
  often raises throughput by tightening cell size and improving SINR everywhere.
- **Disable lower data rates.** 1, 2, 5.5, 11 Mbps (the "DSSS rates") use 22 MHz
  of spectrum to deliver 11 Mbps and force long airtime. Disabling them on
  modern deployments breaks 802.11b clients (basically nothing today) and saves
  20-30% airtime.
- **Beacon transmission rate.** Beacons by default are sent at the lowest
  basic rate (1 Mbps in 2.4 GHz, 6 Mbps in 5 GHz). Raising the beacon rate
  saves 1-2% airtime per AP and shrinks the cell — useful for high density.
- **Wider isn't always better.** 160 MHz needs 9 dB more SNR than 20 MHz to
  hit the same MCS. If RSSI is borderline, 80 MHz at MCS 9 outperforms 160 MHz
  at MCS 5.
- **Disable mesh from clients.** Some Wi-Fi cards (Apple) speak 802.11s in a
  way that creates ghost mesh paths. Set mesh interfaces to AP-only for stability.

## Wi-Fi 7 highlights (802.11be EHT)

- **320 MHz channels** in 6 GHz (3 in U-NII-5..U-NII-8 combined).
- **4096-QAM** (12 bits/subcarrier; needs ~36 dB SNR at the receiver).
- **Multi-Link Operation (MLO)**: a STA associates with one MLD (Multi-Link
  Device) but uses 2-3 radios simultaneously across bands.
  - **STR-MLO** (Simultaneous TX/RX): full-duplex equivalent across links.
  - **NSTR-MLO** (Non-STR): half-duplex per link, but distinct links can be
    in different states.
  - **eMLSR** (enhanced Multi-Link Single Radio): listen on multiple links,
    transmit on the best one.
- **Multi-RU per STA**: a STA can be assigned non-contiguous RUs.
- **Preamble Puncturing**: tolerate a 20 MHz block of interference inside a
  wider channel by skipping that subcarrier set rather than abandoning the
  whole channel.
- **Restricted TWT**: deterministic latency for AR/VR (carved-out scheduled
  airtime — competing access classes are prohibited from transmitting).

PHY math at the limit (Wi-Fi 7, 320 MHz, 4096-QAM, 16 SS):

```
Ndsc (320 MHz EHT) = 3920
Nbpsc (4096-QAM)   = 12
R (MCS 13)         = 5/6
Tsym (1x GI)       = 13.6 µs
Nss                = 16

Per-stream rate = (3920 × 12 × 5/6) / 13.6 = 39 200 / 13.6 = 2882 Mbps
16 streams       = 46 116 Mbps ≈ 46.1 Gbps
```

This is theoretical — no shipping endpoints support 16 SS as of 2025. Realistic
high-end Wi-Fi 7 hardware is 4 SS, 320 MHz, 4096-QAM — about 11.5 Gbps PHY,
~7 Gbps TCP goodput on a clean link.

## Power-save extensions

802.11 power management has evolved across generations:

- **PS-Poll** (legacy): STA sleeps, wakes for beacons, polls AP for buffered
  frames one at a time. Slow but universal.
- **WMM-PS / U-APSD**: STAs trigger AP to deliver buffered frames in bursts.
  Voice/video friendly.
- **TWT (Target Wake Time, 802.11ax)**: STA and AP negotiate a schedule.
  Battery-sensitive STAs sleep deterministically.
- **Restricted TWT (Wi-Fi 7)**: AP holds airtime open exclusively for a
  scheduled STA — no other STA can transmit during that window.

TWT example math:

```
Sleep window: 100 ms
Wake duration: 5 ms
Duty cycle: 5/100 = 5%
STA radio power: 1.5 W when on, 0.5 mW when sleeping
Average draw (TWT 5/100): 0.05 × 1.5 + 0.95 × 0.0005 = 0.075 W

Vs always-on listen: 1.5 W
TWT saves: 95% of receive-mode current → 20× battery life on idle workloads
```

## Common diagnostic recipes

### Bad RSSI with "good" link

```
$ iw dev wlan0 link
Connected to 12:34:56:78:9a:bc (on wlan0)
        SSID: corp-wifi
        freq: 5180
        RX: 14523 bytes (78 packets)
        TX: 9281 bytes (62 packets)
        signal: -78 dBm
        rx bitrate: 6.5 MBit/s
        tx bitrate: 6.0 MBit/s
```

`signal -78 dBm` is borderline; `tx bitrate 6 Mbps` is the basic rate floor —
the radio is dropping to MCS 0 to keep the link alive. Either move closer or
move the AP.

### Channel utilization above 70%

```
$ iw dev wlan0 survey dump | head -20
Survey data from wlan0
        frequency:                      5180 MHz [in use]
        noise:                          -94 dBm
        channel active time:            120000 ms
        channel busy time:              98000 ms     ← 81% busy
        channel receive time:            70000 ms
        channel transmit time:           20000 ms
```

81% busy means co-channel APs or constant scan. Shrink AP cell, change channel,
disable scans during peak hours.

### Roaming refusing to happen

```
$ wpa_cli -i wlan0 signal_poll
RSSI=-72
LINKSPEED=12
NOISE=-90
FREQUENCY=5180

$ iw dev wlan0 scan dump | grep -E "BSS|signal|SSID"
BSS aa:bb:cc:dd:ee:01 ...
   SSID: corp-wifi
   signal: -67 dBm     ← stronger candidate
BSS aa:bb:cc:dd:ee:02 ...
   SSID: corp-wifi
   signal: -72 dBm     ← currently associated
```

A 5 dB delta is not enough to trigger roaming on most STAs (the 10 dB hysteresis
prevents flapping). Either reduce TX power on the current AP to break the
association, or push 11v hint from the WLC.

## Glossary (selected)

- **A-MPDU / A-MSDU** — aggregation forms.
- **AID** (Association ID) — short identifier the AP assigns to each STA.
- **AKM** (Authentication and Key Management suite) — what the RSN IE advertises
  (WPA, WPA2, WPA3, FT-WPA2, etc).
- **BSSID** — MAC address of the AP's radio (one per BSS).
- **BSS** — Basic Service Set: one AP and its associated STAs.
- **DS** — Distribution System: the wired side of the AP.
- **EIRP** — Effective Isotropic Radiated Power; what regulators cap.
- **ESS** — Extended Service Set: multiple BSSes sharing an SSID.
- **GTK** — Group Temporal Key (multicast/broadcast encryption key).
- **IBSS** — Independent BSS (ad-hoc).
- **MFP** — Management Frame Protection (802.11w); cryptographically protects
  deauth/disassoc frames.
- **MLD** — Multi-Link Device (Wi-Fi 7).
- **NAV** — Network Allocation Vector; virtual carrier sense.
- **PMF** — same as MFP, the post-2014 marketing name.
- **PMK / PTK / GTK** — Pairwise Master / Pairwise Transient / Group Temporal
  keys.
- **PSD** — Power Spectral Density.
- **RSN IE** — Robust Security Network Information Element; the beacon's
  security capabilities advertisement.
- **STA** — Station; any 802.11-capable client.
- **TIM / DTIM** — Traffic Indication Map / Delivery TIM.
- **TWT** — Target Wake Time.

## See Also

- `networking/cisco-wireless` — Cisco WLC + AP architecture, IOS-XE 9800 config
- `networking/bridge` — Linux bridge (where AP traffic lands on the wired side)
- `networking/dhcp` — STA address assignment after association
- `ramp-up/wifi-eli5` — narrative ELI5 introduction to Wi-Fi
- `ramp-up/osi-model-eli5` — where 802.11 sits in the OSI/TCP-IP layering

## References

- IEEE Std 802.11-2020 — base standard (~4400 pages)
- IEEE Std 802.11ax-2021 — High Efficiency (Wi-Fi 6 / 6E)
- IEEE Std 802.11be — Extremely High Throughput (Wi-Fi 7), in publication 2024
- Wi-Fi Alliance: WPA3 Specification (current revision: WPA3 R3, 2022)
- Wi-Fi Alliance: 6 GHz AFC System Specification
- RFC 7833 — A PEAP-EAP-TLS Authentication Method
- RFC 5216 — The EAP-TLS Authentication Protocol
- RFC 8110 — Opportunistic Wireless Encryption (OWE)
- RFC 7170 — TEAP Version 1 (Tunnel-based EAP for enterprises)
- FCC 47 CFR Part 15 — Unlicensed radio devices, Subpart E (U-NII)
- ETSI EN 301 893 / EN 300 328 — European 5/2.4 GHz regulatory standards
- Aruba Wireless 802.11ax Reference Design Guide
- Cisco Wi-Fi 6 / 6E Design and Deployment Guide
- Cisco Catalyst 9800 Series Wireless Controller Configuration Guide
- Vivek Ramachandran, "Backtrack 5 Wireless Penetration Testing" — KRACK / WEP demos
- Mathy Vanhoef, KRACK paper (2017): "Key Reinstallation Attacks: Forcing Nonce Reuse in WPA2"
- Mathy Vanhoef, Dragonblood paper (2019): "Dragonblood: A Security Analysis of WPA3's SAE Handshake"
- Ekahau, Hamina, iBwave — RF planning tools
- Atheros / Broadcom / Qualcomm chipset datasheets for sensitivity tables
