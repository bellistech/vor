# Physical Security — Theory, Risk Assessment, and Convergence

> *Physical security is the foundation of the security triad. No amount of encryption, firewalls, or access control software matters if an attacker can walk into a server room and pull a hard drive. Physical security theory integrates environmental design, biometric science, fire engineering, and power systems into a unified defense framework.*

---

## 1. Crime Prevention Through Environmental Design (CPTED)

### Core Principles

CPTED uses environmental design to influence human behavior and reduce crime opportunity. Developed by C. Ray Jeffery (1971) and Oscar Newman (1972), it provides a systematic framework for physical security design.

**Four principles:**

| Principle | Definition | Application |
|:---|:---|:---|
| Natural Surveillance | Design that maximizes visibility | Open sight lines, windows facing walkways, adequate lighting, trimmed landscaping |
| Natural Access Control | Design that guides movement | Defined entry points, pathways that channel visitors, hedges as barriers |
| Territorial Reinforcement | Design that defines ownership | Signage, fencing, landscaping, maintenance, distinct zones |
| Maintenance | Upkeep signals active stewardship | Broken windows theory — decay invites crime |

### Second-Generation CPTED

Extends the original four principles with social factors:

- **Social cohesion:** Community engagement in security awareness
- **Connectivity:** Relationship between areas and neighborhoods
- **Community culture:** Shared values about security behavior
- **Threshold capacity:** Maximum stress an environment can handle before disorder

### CPTED Assessment Methodology

1. **Site survey:** Map all physical features, entry points, sight lines
2. **Crime analysis:** Review historical incidents, police reports, threat intelligence
3. **Behavioral observation:** Track how people actually move through the space
4. **Gap analysis:** Compare current design against CPTED principles
5. **Recommendations:** Prioritized improvements with cost-benefit analysis
6. **Post-implementation review:** Measure impact on incident rates

### Lighting Standards

| Area | Minimum Illumination | Purpose |
|:---|:---|:---|
| Parking lots | 10-50 lux | Facial recognition at 15m |
| Building perimeter | 20-50 lux | Intruder detection |
| Entry points | 100-300 lux | Badge/face verification |
| CCTV zones | 50+ lux (or IR) | Camera capture quality |
| Emergency paths | 10 lux minimum | Evacuation visibility |

Lighting should be uniform — high contrast (bright spots with dark shadows) is worse than consistent moderate illumination because shadows provide concealment.

---

## 2. Physical Security Risk Assessment

### Risk Assessment Framework

Physical security risk combines:

$$\text{Risk} = f(\text{Threat}, \text{Vulnerability}, \text{Impact})$$

**Quantitative approach:**

$$\text{ALE} = \text{SLE} \times \text{ARO}$$

Where:
- $\text{SLE}$ (Single Loss Expectancy) = Asset Value $\times$ Exposure Factor
- $\text{ARO}$ (Annualized Rate of Occurrence) = expected frequency per year
- $\text{ALE}$ (Annualized Loss Expectancy) = expected yearly loss

**Example:**

| Parameter | Value |
|:---|:---|
| Asset value (server room equipment) | $2,000,000 |
| Exposure factor (theft scenario) | 0.15 (15% loss) |
| SLE | $300,000 |
| ARO (estimated theft frequency) | 0.05 (once per 20 years) |
| ALE | $15,000/year |

If a mantrap system costs $50,000 to install and $5,000/year to maintain, and reduces ARO to 0.005:
- New ALE = $300,000 $\times$ 0.005 = $1,500/year
- Savings = $15,000 - $1,500 = $13,500/year
- Simple payback = $50,000 / $13,500 = 3.7 years

### Threat Categories

| Category | Threat Actors | Motivation |
|:---|:---|:---|
| Criminal | Thieves, vandals | Financial gain, destruction |
| Activist | Protesters, hacktivists | Ideological, publicity |
| Insider | Disgruntled employees | Revenge, financial, espionage |
| State-sponsored | Intelligence agencies | Espionage, sabotage |
| Natural | Earthquake, flood, fire | Environmental forces |
| Accidental | Careless employees | Human error |

### Vulnerability Assessment Checklist

| Domain | Assessment Items |
|:---|:---|
| Perimeter | Fence condition, gate security, lighting, blind spots, drainage access |
| Building envelope | Door strength, lock quality, window security, roof access, loading docks |
| Interior | Access control zones, badge system audit, tailgating controls, camera coverage |
| Personnel | Background check currency, training records, security awareness scores |
| Procedures | Access review frequency, visitor logs, key control, alarm response time |
| Technology | CCTV health, badge system logs, intrusion detection maintenance |

---

## 3. Biometric Performance Theory

### Error Rates and Operating Points

A biometric system produces a similarity score $s$ between a sample and a template. A threshold $\tau$ determines accept/reject:

$$\text{Decision} = \begin{cases} \text{Accept} & \text{if } s \geq \tau \\ \text{Reject} & \text{if } s < \tau \end{cases}$$

**Error trade-off:**

Increasing $\tau$ (stricter threshold):
- FAR decreases (fewer impostors accepted)
- FRR increases (more legitimate users rejected)

Decreasing $\tau$ (looser threshold):
- FAR increases
- FRR decreases

### Crossover Error Rate (CER / EER)

The **Equal Error Rate** is the operating point where $\text{FAR} = \text{FRR}$:

$$\text{CER} = \tau^* \text{ such that } \text{FAR}(\tau^*) = \text{FRR}(\tau^*)$$

Lower CER indicates better overall biometric system accuracy. However, operational deployment rarely uses the CER point:

- **High security (vault):** Set $\tau$ above CER point → very low FAR, higher FRR
- **High throughput (building entrance):** Set $\tau$ below CER point → lower FRR, higher FAR

### DET Curve (Detection Error Tradeoff)

The DET curve plots FRR vs FAR on a log scale, providing a comprehensive view of system performance across all thresholds. Better systems have curves closer to the origin.

### Multi-Modal Biometrics

Combining multiple biometric modalities improves accuracy:

**Score-level fusion:** Combine similarity scores from multiple systems:

$$s_{combined} = w_1 \cdot s_{fingerprint} + w_2 \cdot s_{iris} + w_3 \cdot s_{face}$$

**Decision-level fusion:** Combine individual accept/reject decisions:
- AND rule: all must accept (very low FAR, high FRR)
- OR rule: any can accept (low FRR, higher FAR)
- Majority vote: quorum must accept (balanced)

### Anti-Spoofing (Presentation Attack Detection)

| Biometric | Spoofing Attack | Countermeasure |
|:---|:---|:---|
| Fingerprint | Silicone mold, printed image | Liveness detection (pulse, sweat, capacitance) |
| Face | Photo, video, 3D mask | 3D depth sensing, blink detection, IR imaging |
| Iris | Printed iris image, contact lens | Pupil dilation response, specular reflection |
| Voice | Recording, synthesis | Challenge-response, liveness phrases |

---

## 4. Fire Classes and Suppression Selection

### Selection Matrix

| Facility Type | Primary Risk | Recommended System | Rationale |
|:---|:---|:---|:---|
| Data center | Electrical (Class C) | FM-200 or Novec 1230 | Safe for electronics, safe for people, fast discharge |
| Archive/museum | Paper/artifacts (Class A) | Pre-action (double interlock) | Prevents accidental water discharge |
| Office space | General (Class A) | Wet pipe sprinkler | Simplest, most reliable, lowest cost |
| Cold storage | Frozen pipes risk | Dry pipe | No standing water in pipes |
| Chemical storage | Flammable liquid (Class B) | Foam or deluge | Covers large area rapidly |
| Unmanned facility | Any | CO2 | Highly effective but lethal to humans |
| Telecom room | Electrical (Class C) | Inergen or FM-200 | Safe for equipment, habitable concentration |

### Clean Agent Design Considerations

**FM-200 (HFC-227ea):**
- Design concentration: 7-9% by volume
- Safe for occupied spaces up to 9% (NOAEL)
- Discharge time: 10 seconds
- Hold time: 10 minutes minimum
- Room integrity test required (door fan test)
- Ozone Depletion Potential (ODP): 0
- Global Warming Potential (GWP): 3220 (high — regulatory pressure)

**Novec 1230 (FK-5-1-12):**
- Design concentration: 4.2-5.9% by volume
- Safe for occupied spaces (NOAEL = 10%)
- Discharge time: 10 seconds
- ODP: 0, GWP: 1 (excellent environmental profile)
- Atmospheric lifetime: 3-5 days (vs 33 years for FM-200)
- More expensive than FM-200 but environmentally preferred

### Room Integrity

Clean agent systems require sealed rooms to maintain design concentration:

$$\text{Hold time} = f(\text{agent volume}, \text{leakage rate}, \text{enclosure volume})$$

**Door fan test (NFPA 2001):**
- Pressurize and depressurize the room
- Measure equivalent leakage area
- Calculate agent retention time
- Minimum hold time: 10 minutes at design concentration

---

## 5. UPS Sizing Calculations

### Basic Sizing

$$\text{UPS Capacity (kVA)} = \frac{\text{Total Load (kW)}}{\text{Power Factor}}$$

$$\text{Runtime (hours)} = \frac{\text{Battery Capacity (Ah)} \times \text{Battery Voltage (V)} \times \text{Efficiency}}{\text{Total Load (W)}}$$

### Practical Example

| Component | Qty | Watts Each | Total Watts |
|:---|:---:|:---:|:---:|
| Servers | 20 | 750 W | 15,000 W |
| Network switches | 4 | 200 W | 800 W |
| Storage arrays | 2 | 1,200 W | 2,400 W |
| Cooling (in-row) | 2 | 3,000 W | 6,000 W |
| **Total** | | | **24,200 W** |

$$\text{UPS Capacity} = \frac{24,200}{0.9} = 26,889 \text{ VA} \approx 30 \text{ kVA (with 10% headroom)}$$

**Redundancy levels:**

| Config | UPS Units | Capacity Each | Notes |
|:---|:---:|:---:|:---|
| N | 1 | 30 kVA | No redundancy |
| N+1 | 2 | 30 kVA | One spare module |
| 2N | 2 | 30 kVA | Fully duplicated path |
| 2(N+1) | 4 | 15 kVA | Two paths, each with redundancy |

### Battery Considerations

| Battery Type | Life | Cost | Weight | Maintenance |
|:---|:---|:---|:---|:---|
| VRLA (Valve-Regulated Lead-Acid) | 3-5 years | Low | Heavy | Quarterly testing |
| Lithium-Ion | 8-10 years | High | Light | Minimal |
| Nickel-Cadmium | 15-20 years | Very High | Medium | Annual |

Battery testing: annual capacity test (load bank), quarterly float voltage check, continuous monitoring (internal resistance trending).

---

## 6. Physical Security in Defense-in-Depth

### Integration with Logical Security

Physical and logical security must be coordinated:

| Physical Event | Logical Response |
|:---|:---|
| Badge swipe at entry | Enable network port for user's assigned port |
| After-hours building access | Trigger enhanced logging, MFA requirement |
| Visitor badge issued | Enable guest VLAN, restrict to internet only |
| Emergency evacuation | Lock workstations, disable remote access |
| Forced door alarm | Isolate network segment, alert SOC |

### Convergence Model

Traditional organizations have separate physical and logical security teams. Convergence combines them:

**Converged SOC:**
- Single monitoring console for physical and cyber events
- Correlated alerts: "Badge swipe in Building A but VPN login from Building B"
- Unified incident response: physical access revocation alongside account lockout
- Shared threat intelligence: physical surveillance tied to cyber threat indicators

**Benefits:**
- Insider threat detection: correlate badge access patterns with data access patterns
- Faster response: physical isolation of compromised assets
- Reduced overhead: single team, single platform
- Better situational awareness: holistic view of security posture

---

## 7. Insider Threat Physical Controls

### Detection Indicators

| Category | Physical Indicators |
|:---|:---|
| Behavioral | After-hours access, accessing unauthorized areas, photographing screens |
| Material | Removing equipment/media, unusual bags/containers, copying documents |
| Access pattern | Accessing areas outside job function, excessive badge failures, tailgating |
| Social | Disgruntlement, financial stress, resignation announcement |

### Technical Controls

| Control | Purpose | Implementation |
|:---|:---|:---|
| Badge analytics | Detect anomalous access patterns | AI/ML on badge log data |
| USB port locks | Prevent data exfiltration via removable media | Physical port blockers + DLP |
| Camera analytics | Detect suspicious behavior | Video AI: tailgating, loitering, object removal |
| Clean desk policy | Prevent information exposure | Daily audits, locked cabinets |
| Screen privacy filters | Prevent shoulder surfing | 3M privacy screens on monitors |
| Two-person integrity | Prevent solo access to critical assets | Dual-badge requirement for sensitive rooms |

### Exit Procedures

```
1. Immediate badge deactivation upon termination
2. Escort from building by security (not manager)
3. Return of all equipment: laptop, phone, badge, keys
4. Verify no removable media carried out
5. Change shared credentials (admin, safe combos)
6. Update access lists within 24 hours
7. Retain access logs for 90+ days post-departure
8. Monitor for post-employment physical access attempts
```

---

## 8. Environmental Threats and Mitigation

### Natural Disaster Resilience

| Threat | Mitigation | Standard |
|:---|:---|:---|
| Earthquake | Seismic bracing for racks, raised floor anchoring, flexible pipe joints | IBC Seismic Zone requirements |
| Flood | Elevated site, no basement equipment, water detection sensors, sump pumps | FEMA flood zone maps |
| Hurricane/tornado | Reinforced structure, impact-resistant glass, wind-rated doors | ASCE 7 wind load requirements |
| Lightning | Lightning rods, surge protection, bonded grounding grid | NFPA 780 |
| Wildfire | Defensible space, fire-resistant materials, ember-resistant vents | NFPA 1144 |

### Water Damage Prevention

Data center water damage sources:
- HVAC condensation (most common)
- Roof leaks
- Plumbing failures
- Fire suppression (accidental or post-fire)
- Groundwater intrusion

**Detection:** Cable-based leak detection under raised floors, point sensors at CRAC units, rope sensors along pipe runs.

**Prevention:** No water pipes above IT equipment, drip pans under CRAC units, sloped floors toward drains, pre-action fire suppression (not wet pipe).

---

## References

- Jeffery, C.R. "Crime Prevention Through Environmental Design" (1971)
- Newman, Oscar. "Defensible Space" (1972)
- ASIS International. "Physical Security Professional (PSP)" Body of Knowledge
- NFPA 2001: Standard on Clean Agent Fire Extinguishing Systems
- NFPA 75: Standard for Protection of IT Equipment
- TIA-942: Telecommunications Infrastructure Standard for Data Centers
- ASHRAE TC 9.9: Thermal Guidelines for Data Processing Environments
- ISO 27001 Annex A.11: Physical and Environmental Security
- NIST SP 800-116: Guidelines for PIV Card Authentication
- Uptime Institute: Data Center Tier Standards
- IEEE 3001.2: Recommended Practice for Evaluating UPS Systems
