# The Mathematics of PTP -- Clock Offset, Delay Measurement, and BMCA

> *PTP achieves sub-microsecond synchronization by combining hardware-assisted timestamping with a four-timestamp delay model, a deterministic best master clock election, and a servo loop that continuously disciplines the local oscillator. Understanding the math behind offset calculation, delay asymmetry compensation, and transparent clock correction fields is essential for diagnosing synchronization issues in production networks.*

---

## 1. Clock Offset and Delay Calculation

### The Four-Timestamp Model

PTP uses four timestamps from a Sync/Delay_Req exchange to compute the clock offset and one-way delay. Let master clock M and slave clock S have a true offset $\theta$ (slave ahead of master by $\theta$), and let the one-way delays be $d_{ms}$ (master to slave) and $d_{sm}$ (slave to master).

The four timestamps:

- $t_1$: Sync departure time at master (master's clock)
- $t_2$: Sync arrival time at slave (slave's clock)
- $t_3$: Delay_Req departure time at slave (slave's clock)
- $t_4$: Delay_Req arrival time at master (master's clock)

The relationships:

$$t_2 = t_1 + d_{ms} + \theta$$

$$t_4 = t_3 - \theta + d_{sm}$$

### Deriving Offset and Delay

From the two equations above:

$$t_2 - t_1 = d_{ms} + \theta \quad \text{...(1)}$$

$$t_4 - t_3 = d_{sm} - \theta \quad \text{...(2)}$$

Adding (1) and (2):

$$(t_2 - t_1) + (t_4 - t_3) = d_{ms} + d_{sm} = d_{\text{round-trip}}$$

$$d_{\text{mean}} = \frac{(t_2 - t_1) + (t_4 - t_3)}{2}$$

Subtracting (2) from (1):

$$(t_2 - t_1) - (t_4 - t_3) = d_{ms} - d_{sm} + 2\theta$$

Assuming symmetric delay ($d_{ms} = d_{sm}$):

$$\theta = \frac{(t_2 - t_1) - (t_4 - t_3)}{2}$$

### Worked Example

**Example:** $t_1 = 0$, $t_2 = 0.000\,000\,350$, $t_3 = 0.001\,000\,000$, $t_4 = 0.001\,000\,280$

Mean path delay:

$$d_{\text{mean}} = \frac{(0.000\,000\,350 - 0) + (0.001\,000\,280 - 0.001\,000\,000)}{2}$$

$$= \frac{0.000\,000\,350 + 0.000\,000\,280}{2} = \frac{0.000\,000\,630}{2} = 0.000\,000\,315 \text{ sec} = 315 \text{ ns}$$

Clock offset:

$$\theta = \frac{(0.000\,000\,350 - 0) - (0.001\,000\,280 - 0.001\,000\,000)}{2}$$

$$= \frac{0.000\,000\,350 - 0.000\,000\,280}{2} = \frac{0.000\,000\,070}{2} = 0.000\,000\,035 \text{ sec} = 35 \text{ ns}$$

The slave clock is 35 ns ahead of the master, with a mean path delay of 315 ns.

### Delay Asymmetry Error

If the true one-way delays are asymmetric ($d_{ms} \neq d_{sm}$), the symmetric assumption introduces an error:

$$\theta_{\text{error}} = \frac{d_{ms} - d_{sm}}{2}$$

**Example:** If $d_{ms} = 400$ ns and $d_{sm} = 230$ ns (asymmetric due to different fiber lengths or switch queuing):

$$\theta_{\text{error}} = \frac{400 - 230}{2} = 85 \text{ ns}$$

The computed offset will be wrong by 85 ns. This is the fundamental limitation of PTP -- delay asymmetry cannot be measured, only compensated if known a priori via the `delayAsymmetry` configuration parameter:

```
# ptp4l.conf — compensate known asymmetry
delayAsymmetry    85000    # in scaled nanoseconds (ns * 2^16)
```

---

## 2. Peer-to-Peer Delay Measurement

### The Pdelay Exchange

P2P delay measurement uses a separate three-message exchange between adjacent nodes to measure per-link delay, independent of the master-slave hierarchy.

Timestamps for a Pdelay exchange between requester A and responder B:

- $t_1$: Pdelay_Req departure at A (A's clock)
- $t_2$: Pdelay_Req arrival at B (B's clock)
- $t_3$: Pdelay_Resp departure at B (B's clock)
- $t_4$: Pdelay_Resp arrival at A (A's clock)

The peer link delay (assuming symmetry on the single link):

$$d_{\text{peer}} = \frac{(t_4 - t_1) - (t_3 - t_2)}{2}$$

This measures only the link delay between A and B, not the full path. Each link in the network measures its own delay independently.

### Cumulative Path Delay with P2P

For a slave N hops from the master, the total path delay is the sum of all per-link delays:

$$d_{\text{total}} = \sum_{i=1}^{N} d_{\text{peer},i}$$

Each transparent clock adds its measured peer delay to the Sync message's correctionField, so the slave receives the cumulative correction and computes offset using only the residual.

---

## 3. Transparent Clock Correction Field

### Residence Time Correction

A transparent clock (TC) measures how long a PTP event message resides inside the switch (ingress to egress) and adds this residence time to the message's correctionField.

For a two-step TC processing a Sync message:

$$\text{correctionField}_{\text{out}} = \text{correctionField}_{\text{in}} + (t_{\text{egress}} - t_{\text{ingress}})$$

For a chain of $N$ transparent clocks:

$$\text{correctionField}_{\text{final}} = \sum_{i=1}^{N} (t_{\text{egress},i} - t_{\text{ingress},i})$$

### Adjusted Offset Calculation

The slave uses the correctionField to remove switch residence times from its offset calculation:

$$\theta = \frac{(t_2 - t_1 - \text{CF}_{\text{sync}}) - (t_4 - t_3 - \text{CF}_{\text{delay}})}{2}$$

Where $\text{CF}_{\text{sync}}$ is the correctionField from the Sync/Follow_Up and $\text{CF}_{\text{delay}}$ is from the Delay_Resp.

### Worked Example

**Example:** Sync traverses two transparent clocks with residence times of 450 ns and 320 ns.

$$\text{CF}_{\text{sync}} = 450 + 320 = 770 \text{ ns}$$

If $t_1 = 0$, $t_2 = 0.000\,001\,200$ (raw arrival including switch delays):

$$t_2 - t_1 - \text{CF}_{\text{sync}} = 1200 - 770 = 430 \text{ ns}$$

This 430 ns represents only the wire propagation delay, with switch residence times removed. Without the TC correction, the slave would compute a path delay inflated by 770 ns.

### E2E vs P2P Transparent Clocks

```
E2E Transparent Clock:
  Corrects Sync AND Delay_Req/Delay_Resp messages
  Adds residence time to correctionField
  Does NOT measure link delays
  Simpler but requires consistent path for Sync and Delay_Req

P2P Transparent Clock:
  Corrects Sync messages only
  Adds residence time AND measured peer link delay to correctionField
  correctionField += residence_time + peer_delay
  Better accuracy: slave sees total correction for entire path
  Requires every device in the path to support P2P
```

---

## 4. Best Master Clock Algorithm (BMCA) -- Detailed

### Dataset Comparison

The BMCA uses a strict ordering of clock properties to determine the best clock. The comparison between two clocks A and B follows this decision tree:

```
1. Compare GM priority1
   A.priority1 < B.priority1 -> A wins
   A.priority1 > B.priority1 -> B wins
   Equal -> continue

2. Compare GM clockClass
   Lower class wins (6 = primary reference, 7 = holdover PRC,
   13 = ARB locked, 52 = degradation master, 187 = alt PRC,
   248 = default, 255 = slave-only)

3. Compare GM clockAccuracy
   Lower value wins:
   0x20 = 25 ns       0x24 = 2.5 us
   0x21 = 100 ns      0x25 = 10 us
   0x22 = 250 ns      0x31 = 10 ms
   0x23 = 1 us        0xFE = unknown

4. Compare GM offsetScaledLogVariance
   Lower value wins (Allan variance of the clock oscillator)

5. Compare GM priority2
   Lower value wins (administrator-configured tiebreaker)

6. Compare GM clockIdentity
   Lower value wins (EUI-64, derived from MAC address)
   Deterministic final tiebreaker -- no ties possible
```

### State Machine

Each PTP port runs a state machine influenced by BMCA results:

```
States:
  INITIALIZING  ->  LISTENING   (port enabled)
  LISTENING     ->  MASTER      (BMCA: this clock is best)
  LISTENING     ->  SLAVE       (BMCA: another clock is best)
  LISTENING     ->  PASSIVE     (BMCA: better path exists to same GM)
  MASTER        ->  SLAVE       (BMCA: new, better GM appeared)
  SLAVE         ->  MASTER      (BMCA: GM lost, this clock elected)
  SLAVE         ->  LISTENING   (announce receipt timeout, re-evaluate)

Announce Receipt Timeout:
  timeout = announceReceiptTimeout * 2^(logAnnounceInterval)
  Default: 3 * 2^1 = 6 seconds
  After timeout with no Announce from current GM, port returns
  to LISTENING and BMCA re-evaluates
```

### Alternate BMCA (IEEE 1588-2019)

PTPv2.1 introduced an alternate BMCA that supports external inputs, allowing network management systems to influence GM selection beyond the standard dataset comparison. This is used in telecom profiles where operators need deterministic GM placement.

---

## 5. Servo Loop Tuning

### PI Controller

The ptp4l servo uses a proportional-integral (PI) controller to discipline the local oscillator. The frequency adjustment applied at each Sync interval:

$$f_n = K_p \cdot \theta_n + K_i \cdot \sum_{k=0}^{n} \theta_k$$

Where:
- $f_n$ is the frequency correction in ppb at step $n$
- $\theta_n$ is the measured clock offset at step $n$
- $K_p$ is the proportional gain
- $K_i$ is the integral gain

Default ptp4l servo constants:

```
# ptp4l.conf
pi_proportional_const    0.0       # auto-calculated if 0.0
pi_integral_const        0.0       # auto-calculated if 0.0
pi_proportional_scale    0.0       # alternative: scale factor
pi_integral_scale        0.0
pi_proportional_exponent -0.3
pi_integral_exponent     0.4
pi_proportional_norm_max 0.7
pi_integral_norm_max     0.3
step_threshold           0.0       # step if offset > threshold (0 = never step after initial)
first_step_threshold     0.00002   # 20 us — step on first correction if offset > this
servo_offset_threshold   0         # ignore offsets below this (ns)
servo_num_offset_values  10        # samples before starting servo
```

### Tuning Guidelines

```
Symptom                         Adjustment
Offset oscillates               Reduce Kp (proportional gain)
Slow convergence                Increase Kp
Steady-state offset drift       Increase Ki (integral gain)
Offset never settles            Reduce Ki (over-integration)
Large initial step              Adjust first_step_threshold
Jitter from noisy timestamps    Increase servo_num_offset_values
```

### Servo States

```
s0  UNLOCKED     Not synchronized; servo accumulating samples
s1  STEP         Clock stepped to match master (one-time correction)
s2  LOCKED       Servo running; frequency adjustment only
```

A well-tuned system transitions from s0 -> s1 -> s2 within a few seconds and remains in s2 with sub-100 ns offset.

---

## 6. Hardware vs Software Timestamping

### Accuracy Comparison

```
Timestamping Point          Typical Accuracy     Source of Error
PHC hardware (NIC silicon)  <10 ns               Oscillator granularity
Kernel socket (SO_TIMESTAMPING) 1-10 us          Interrupt latency, scheduling
User-space gettimeofday     10-100 us            System call overhead, context switch
```

### Where Timestamps Are Captured

```
Software timestamping:
  Application
       |
       v
  Socket layer  <--- timestamp captured here (kernel)
       |                 Includes: scheduler jitter,
       v                 interrupt coalescing,
  Network driver         driver processing time
       |
       v
  NIC hardware   <--- actual packet TX/RX time

Hardware timestamping:
  Application
       |
       v
  Socket layer
       |
       v
  Network driver
       |
       v
  NIC hardware   <--- timestamp captured here (PHC)
                      No software jitter above this point
```

### Impact on PTP Accuracy

Software timestamping adds jitter from interrupt latency, kernel scheduling, and driver processing. Under load, these contribute 1-100 us of non-deterministic error that the servo cannot remove because it appears as path delay variation (PDV).

Hardware timestamping eliminates this entire class of error. The PHC captures the timestamp at the precise moment the packet's timestamp point (start of frame for Ethernet) crosses the MII/GMII boundary.

### PHC (PTP Hardware Clock)

The PHC is a free-running counter in the NIC, driven by the NIC's local oscillator (typically 25 MHz crystal). ptp4l disciplines the PHC to track the grandmaster. phc2sys then disciplines CLOCK_REALTIME to track the PHC.

```
Grandmaster (GPS) -> ptp4l -> PHC (/dev/ptp0) -> phc2sys -> CLOCK_REALTIME
                     (PTP)   (NIC clock)         (SHM)      (system clock)
```

---

## 7. PTPv1 vs PTPv2 (IEEE 1588-2008/2019)

```
Feature                PTPv1 (2002)              PTPv2 (2008 / 2019)
Standard               IEEE 1588-2002            IEEE 1588-2008 / 2019
Accuracy target        Microsecond               Sub-nanosecond
Transport              UDP only                  UDP, L2 Ethernet, DeviceNet
Transparent clocks     No                        Yes (E2E and P2P)
P2P delay mechanism    No                        Yes
One-step operation     No                        Yes
Unicast negotiation    No                        Yes (signaling messages)
TLV extensions         No                        Yes
Domain support         Subdomain (string)        domainNumber (0-255)
BMCA                   Basic                     Enhanced (priority1/2, class)
Management messages    Limited                   Full management TLV set
Profiles               None                      Default, telecom, power, gPTP

PTPv1 and PTPv2 are NOT interoperable on the same domain.
Migrating requires upgrading all PTP devices simultaneously
or running PTPv1 and PTPv2 in separate domains with a translator.
```

---

## 8. Telecom Profile Details

### ITU-T G.8275.1 (Full Timing Support)

G.8275.1 requires every node in the timing path to be PTP-aware (boundary clock or transparent clock). This is the "gold standard" for telecom timing.

```
Key Parameters:
  Transport              Ethernet (L2, EtherType 0x88F7)
  Delay mechanism        E2E
  domainNumber           24 (default for telecom)
  logSyncInterval        -4 (16 Sync messages per second)
  logAnnounceInterval    -3 (8 Announce messages per second)
  logMinDelayReqInterval -4 (16 Delay_Req per second)
  announceReceiptTimeout 3

Clock Classes (telecom-specific):
  6    Primary reference (e.g., GPS-locked)
  7    Primary reference in holdover
  135  ARB PRC traceable
  140  Holdover process completed — degraded
  150  Degraded
  160  T-BC/T-TSC in freerun
  165  Local clock
  248  Default
  255  Slave-only
```

### ITU-T G.8275.2 (Partial Timing Support)

G.8275.2 allows non-PTP-aware nodes (routers, L3 switches) in the timing path. It uses UDP/IP (L4) transport and unicast negotiation to traverse IP networks.

```
Key Parameters:
  Transport              UDPv4 or UDPv6
  Delay mechanism        E2E
  domainNumber           44-63
  Unicast negotiation    Required
  logSyncInterval        -4 to 0 (negotiated)
  logMinDelayReqInterval -4 to 0 (negotiated)

# ptp4l.conf for G.8275.2 slave
[global]
network_transport       UDPv4
delay_mechanism         E2E
domainNumber            44
hybrid_e2e              1
inhibit_multicast_service 1
unicast_master_table    /etc/ptp4l_unicast.conf
unicast_req_duration    300

# /etc/ptp4l_unicast.conf
# GM IP addresses for unicast negotiation
10.0.0.1
10.0.0.2
```

### Accuracy Budget

Telecom networks define an end-to-end time error budget from the primary reference to the cell site:

```
Node                          Max Time Error (ns)
PRTC (GPS receiver)           +/- 100
T-BC per hop (G.8275.1)       +/- 50 per hop
Cumulative (10 hops)          +/- 600 (100 + 10*50)
Cell site (3GPP requirement)  +/- 1500 (1.5 us for TDD)

Budget for G.8275.2 is larger due to packet delay variation
in non-PTP-aware nodes:
  PDV per non-PTP hop          100-1000 ns (load-dependent)
  Maximum non-PTP hops         Vendor-specific (typically 5-10)
```

---

## 9. Frequency Transfer and Holdover

### Syntonization vs Synchronization

```
Syntonization:  Matching frequency (rate) only. Two clocks tick at the
                same rate but may have a constant offset.
                Measured in: ppb (parts per billion)

Synchronization: Matching both phase (offset) and frequency.
                 Clocks agree on what time it is.
                 Measured in: ns (nanoseconds of offset)

PTP provides both. SyncE (Synchronous Ethernet) provides frequency only.
Some deployments combine SyncE (frequency) + PTP (phase) for best results.
```

### Holdover Performance

When the grandmaster becomes unreachable, the slave enters holdover -- maintaining time using only its local oscillator.

```
Oscillator Type            Drift Rate          Time to 1 us Error
Standard crystal (XO)      +/- 20 ppm          50 ms
TCXO (temp-compensated)    +/- 1 ppm           1 sec
OCXO (oven-controlled)     +/- 10 ppb          100 sec
Rubidium                   +/- 0.1 ppb         10,000 sec (~2.8 hr)

Holdover error accumulation:
  error(t) = drift_rate * t + aging_rate * t^2 / 2

For OCXO with 10 ppb drift:
  After 1 sec:  10 ns error
  After 10 sec: 100 ns error
  After 100 sec: 1 us error (exceeds typical PTP budget)
```

---

## References

- [IEEE 1588-2019 -- Precision Time Protocol](https://standards.ieee.org/standard/1588-2019.html)
- [ITU-T G.8275.1 -- Full Timing Support](https://www.itu.int/rec/T-REC-G.8275.1)
- [ITU-T G.8275.2 -- Partial Timing Support](https://www.itu.int/rec/T-REC-G.8275.2)
- [ITU-T G.8271 -- Time and Phase Synchronization](https://www.itu.int/rec/T-REC-G.8271)
- [linuxptp Source and Documentation](https://linuxptp.sourceforge.net/)
- [IEEE 802.1AS-2020 -- Timing and Synchronization for TSN](https://standards.ieee.org/standard/802_1AS-2020.html)
- [NIST -- Precision Time Protocol](https://www.nist.gov/el/intelligent-systems-division-73500/ieee-1588)
