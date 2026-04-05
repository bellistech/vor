# Mobile Hacking -- Deep Dive

> For authorized security testing, red team exercises, and educational study only.
>
> This document expands on `sheets/offensive/mobile-hacking.md` with architectural
> detail on mobile security models, Frida internals, certificate pinning, and
> mobile malware analysis methodology. CEH v13 Module 17.

## Prerequisites

- Rooted Android device or emulator (Genymotion, Android Studio AVD with Google APIs)
- Jailbroken iOS device (checkra1n/unc0ver) or Corellium cloud instance
- Frida (`pip install frida-tools`), Objection (`pip install objection`)
- jadx, apktool, MobSF installed
- Burp Suite Professional with mobile proxy configured
- Basic understanding of ARM architecture, Java/Kotlin, Swift/Objective-C
- Familiarity with OWASP Mobile Top 10 (see cheat sheet)

## 1. Android Security Model

### 1.1 Application Sandbox

Every Android application runs in its own Linux process with a unique UID assigned at install
time. This is the foundation of Android's isolation model:

- **Per-app UID**: Each app gets a unique Linux UID (e.g., u0_a123). Files created by the app
  are owned by this UID. Other apps cannot read them unless explicitly shared.
- **Process isolation**: Each app runs in its own Dalvik/ART virtual machine instance. A crash
  in one app does not affect others.
- **File system permissions**: `/data/data/<package>/` is owned by the app's UID with mode 0700.
  External storage (`/sdcard/`) is shared and world-readable -- a common source of data leakage.
- **Shared UID**: Apps signed with the same certificate can request `android:sharedUserId` to
  share a UID and access each other's files. This is used by platform apps but is deprecated
  in newer API levels.

### 1.2 Permissions Model

Android uses a layered permission system:

- **Normal permissions**: Granted automatically at install (e.g., `INTERNET`, `VIBRATE`). No
  user prompt.
- **Dangerous permissions**: Require runtime user consent (Android 6+). Grouped by category
  (e.g., `READ_CONTACTS` and `WRITE_CONTACTS` are in the Contacts group). Once one permission
  in a group is granted, others in the same group may be auto-granted.
- **Signature permissions**: Only granted to apps signed with the same certificate as the app
  that declared the permission. Used for inter-app communication between a vendor's own apps.
- **Special permissions**: Require user to navigate to Settings (e.g., `SYSTEM_ALERT_WINDOW`,
  `WRITE_SETTINGS`).

**Security testing implications**: Check `AndroidManifest.xml` for over-permissioned apps.
Look for dangerous permissions that do not match the app's stated functionality. Examine
whether permission checks are enforced server-side or only client-side.

### 1.3 SELinux (Security-Enhanced Linux)

Since Android 4.3, SELinux runs in enforcing mode:

- **Mandatory Access Control (MAC)**: Even root processes are constrained by SELinux policies.
  A rooted device does not automatically mean all security is bypassed.
- **Domains and types**: Each process runs in a domain (e.g., `untrusted_app`), and each
  file/resource has a type. Policy rules define which domains can access which types.
- **App domain**: Third-party apps run in the `untrusted_app` domain with heavily restricted
  access. They cannot read `/data/data/` of other apps even with a root exploit unless
  SELinux is set to permissive.
- **Neverallow rules**: Certain operations are permanently blocked by policy (e.g., untrusted
  apps cannot load kernel modules).

```bash
# Check SELinux status
adb shell getenforce
# Returns: Enforcing | Permissive | Disabled

# View security context of a process
adb shell ps -Z | grep com.target.app

# View file security labels
adb shell ls -Z /data/data/com.target.app/
```

### 1.4 Verified Boot and dm-verity

Android Verified Boot (AVB) ensures the integrity of the boot chain:

1. **Bootloader**: Verifies the kernel and ramdisk using embedded keys.
2. **dm-verity**: Verifies every block of the system partition at read time using a hash tree.
   Any modification (e.g., by a rootkit) causes a read failure.
3. **Boot states**: GREEN (fully verified), YELLOW (custom key, user warned), ORANGE
   (unlocked bootloader), RED (verification failed).

**Implications for testing**: Unlocking the bootloader (required for rooting) sets the device
to ORANGE state and typically wipes all data. Custom ROMs require flashing a custom AVB key
or disabling verification entirely.

## 2. iOS Security Model

### 2.1 Code Signing

Every executable on iOS must be signed by Apple or by a developer with a valid Apple-issued
certificate:

- **App Store apps**: Signed by the developer, then encrypted with FairPlay DRM and
  counter-signed by Apple. The kernel verifies the signature before allowing execution.
- **Enterprise apps**: Signed with an enterprise distribution certificate. Can be installed
  outside the App Store but require the device to trust the enterprise profile.
- **Mandatory code signing**: The kernel enforces `CS_ENFORCEMENT` -- unsigned or tampered
  code pages are killed immediately. This is why jailbreaks must patch the kernel or use
  return-oriented programming (ROP) to bypass this check.
- **Entitlements**: Embedded in the code signature, entitlements grant specific capabilities
  (e.g., keychain access groups, push notifications, app groups). Over-entitled apps are
  a review target.

### 2.2 Application Sandbox

iOS uses a stricter sandbox than Android:

- **Container directories**: Each app gets its own container at
  `/var/mobile/Containers/Data/Application/<UUID>/`. The app cannot access other containers.
- **Seatbelt (sandbox profiles)**: The kernel enforces sandbox profiles that restrict system
  calls, file access, and IPC. Third-party apps use the `container` profile which is highly
  restrictive.
- **No shared filesystem**: Unlike Android's SD card, iOS apps cannot share files except
  through explicit APIs (AirDrop, share extensions, app groups with shared containers).
- **IPC restrictions**: Apps communicate through XPC services, URL schemes, and app extensions
  -- all mediated by the system. Direct process-to-process communication is blocked.

### 2.3 Secure Enclave Processor (SEP)

The Secure Enclave is a hardware coprocessor with its own boot ROM and OS:

- **Key storage**: Cryptographic keys (Touch ID/Face ID templates, device passcode derivation
  keys) are stored in the SEP and never leave it. The main processor sends data to the SEP
  for encryption/decryption and receives results -- it never sees the raw keys.
- **Hardware UID key**: A unique 256-bit key fused into the SoC during manufacturing. Not
  readable by any software, including Apple. Used to derive the file encryption keys.
- **Anti-replay**: The SEP maintains a monotonic counter to prevent replay attacks against
  the passcode retry mechanism.
- **Biometric data**: Touch ID fingerprint templates and Face ID depth maps are stored as
  encrypted mathematical representations inside the SEP. They are never sent to Apple or
  backed up to iCloud.

### 2.4 Data Protection Classes

iOS encrypts all files with per-file keys, and those keys are protected by class keys tied
to the device passcode:

| Class | Constant | Availability |
|-------|----------|-------------|
| Complete Protection | `NSFileProtectionComplete` | Only when device is unlocked |
| Protected Unless Open | `NSFileProtectionCompleteUnlessOpen` | Can finish writing after lock |
| Protected Until First Auth | `NSFileProtectionCompleteUntilFirstUserAuthentication` | After first unlock until reboot (default) |
| No Protection | `NSFileProtectionNone` | Always accessible |

**Security testing**: Check which protection class files and keychain items use. Sensitive data
should use `Complete` protection. The default class (`Until First Auth`) means data is accessible
after the first unlock -- a problem on devices that are powered on but locked (e.g., seized
devices).

```bash
# On a jailbroken device, check file protection class
fileDP /var/mobile/Containers/Data/Application/<UUID>/Documents/secret.db
# Or use objection:
# ios plist cat <path>
# ios keychain dump --json  (shows access control flags)
```

## 3. Frida Instrumentation Internals

### 3.1 Architecture Overview

Frida is a dynamic instrumentation toolkit consisting of:

- **frida-server**: A daemon running on the target device (requires root/jailbreak for full
  access). Listens on a TCP port (default 27042) or USB.
- **frida-gadget**: A shared library that can be injected into an app without root by
  embedding it in the APK/IPA. Used for non-rooted testing.
- **frida-core**: The host-side library that communicates with frida-server or frida-gadget
  over the Frida protocol.
- **GumJS**: The JavaScript runtime inside the target process. Built on V8 (or Duktape on
  resource-constrained targets).

### 3.2 Injection Mechanism

**On Android (rooted)**:

1. frida-server runs as root and uses `ptrace()` to attach to the target process.
2. It injects a shared library (`frida-agent.so`) into the process's memory space via
   `dlopen()` or manual mapping.
3. The agent starts a GumJS runtime, loads the user's JavaScript, and establishes a
   bidirectional channel back to frida-server.
4. `ptrace()` is detached after injection -- the agent runs inside the process with the
   process's own permissions.

**On iOS (jailbroken)**:

1. frida-server uses `task_for_pid()` (a Mach kernel call) to get a task port for the
   target process.
2. It allocates memory in the target process and writes the agent dylib.
3. A remote thread is created in the target process to call `dlopen()` on the injected dylib.
4. The agent initializes and connects back to frida-server.

**Gadget mode (non-rooted)**:

1. The APK is unpacked, and `libfrida-gadget.so` is added to `lib/<abi>/`.
2. A `System.loadLibrary("frida-gadget")` call is inserted into the app's main activity
   (via smali patching) or the gadget config file triggers auto-load.
3. On launch, the gadget starts listening for connections before the app's `onCreate()` runs.
4. The tester connects from the host: `frida -U -H localhost Gadget`.

### 3.3 JavaScript Bridge (GumJS API)

Key Frida APIs and their internal mechanisms:

```javascript
// Interceptor: inline hooking via code patching
// Replaces the first instructions of the target function with a trampoline
// that jumps to Frida's handler. Original instructions are relocated.
Interceptor.attach(targetAddr, {
  onEnter(args) {
    // 'args' is a NativePointer array (this.context has registers)
    console.log("arg0 =", args[0].readUtf8String());
  },
  onLeave(retval) {
    retval.replace(ptr(0x1));  // modify return value
  }
});

// Interceptor.replace: completely replace function implementation
Interceptor.replace(targetAddr, new NativeCallback(function(arg0) {
  return 1;  // always return 1
}, 'int', ['pointer']));

// Java.perform: attach to ART VM and enumerate classes
Java.perform(function() {
  var Activity = Java.use("android.app.Activity");
  Activity.onCreate.implementation = function(bundle) {
    console.log("[*] onCreate called");
    this.onCreate(bundle);  // call original
  };
});

// ObjC.classes: enumerate Objective-C runtime
var NSString = ObjC.classes.NSString;
Interceptor.attach(NSString["- isEqualToString:"].implementation, {
  onEnter(args) {
    console.log("Comparing: " + ObjC.Object(args[2]).toString());
  }
});

// Memory scanning
Memory.scan(moduleBase, moduleSize, "48 89 5C 24 ?? 48 89 6C", {
  onMatch(address, size) {
    console.log("Found pattern at: " + address);
  },
  onComplete() {}
});
```

### 3.4 Detection and Evasion

Apps may detect Frida through:

- **Port scanning**: Check if tcp/27042 (frida-server default) is open.
- **Library scanning**: Enumerate loaded libraries looking for `frida-agent` strings.
- **Memory scanning**: Search the process memory for Frida-specific strings
  ("LIBFRIDA", "frida-gadget", "gum-js-loop").
- **Inline hook detection**: Verify function prologues have not been patched (compare
  against known-good bytes).
- **Named pipes**: Check `/proc/self/fd/` for links to `linjector`.

**Evasion**: Rename frida-server binary, use Frida with `--runtime=v8` to change signatures,
patch detection checks with Frida itself before they execute, or use tools like
`frida-antidetect` that strip identifiable strings from the agent.

## 4. Certificate Pinning Implementation and Bypass

### 4.1 What Certificate Pinning Solves

Standard TLS validates that the server certificate chains to a trusted root CA. This is
vulnerable to:

- **Compromised CA**: If any CA in the device's trust store is compromised, an attacker
  can issue valid certificates for any domain.
- **User-installed CAs**: Corporate proxies, testing tools (Burp), and malware can install
  root CAs that the device trusts.
- **Government CAs**: Some device manufacturers include government-controlled CAs.

Certificate pinning restricts which certificates the app accepts, regardless of the
system trust store.

### 4.2 Pinning Strategies

**Certificate pinning**: Pin the exact leaf or intermediate certificate (DER-encoded).
Requires app update when the certificate rotates (typically annually).

**Public key pinning (SPKI)**: Pin the SHA-256 hash of the Subject Public Key Info. More
resilient because the key can remain the same across certificate renewals.

**Pin set with backup**: Include a backup pin for a different key to avoid bricking the
app if the primary key is compromised or lost.

### 4.3 Implementation Examples

**Android Network Security Config** (recommended for Android 7+):

```xml
<!-- res/xml/network_security_config.xml -->
<network-security-config>
  <domain-config>
    <domain includeSubdomains="true">api.example.com</domain>
    <pin-set expiration="2025-06-01">
      <pin digest="SHA-256">AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=</pin>
      <!-- Backup pin -->
      <pin digest="SHA-256">BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=</pin>
    </pin-set>
  </domain-config>
</network-security-config>
```

**iOS with TrustKit**:

```swift
let config: [String: Any] = [
    kTSKSwizzleNetworkDelegates: true,
    kTSKPinnedDomains: [
        "api.example.com": [
            kTSKEnforcePinning: true,
            kTSKPublicKeyHashes: [
                "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
                "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
            ]
        ]
    ]
]
TrustKit.initSharedInstance(withConfiguration: config)
```

**OkHttp (Android)**:

```kotlin
val pinner = CertificatePinner.Builder()
    .add("api.example.com", "sha256/AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
    .add("api.example.com", "sha256/BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB=")
    .build()

val client = OkHttpClient.Builder()
    .certificatePinner(pinner)
    .build()
```

### 4.4 Bypass Techniques

**Frida universal SSL pinning bypass** (works against most implementations):

```javascript
// Android: Hook TrustManagerFactory and X509TrustManager
Java.perform(function() {
  var TrustManagerFactory = Java.use("javax.net.ssl.TrustManagerFactory");
  var X509TrustManager = Java.use("javax.net.ssl.X509TrustManager");

  // Replace the TrustManager with one that accepts all certificates
  var TrustManager = Java.registerClass({
    name: "com.custom.TrustManager",
    implements: [X509TrustManager],
    methods: {
      checkClientTrusted: function(chain, authType) {},
      checkServerTrusted: function(chain, authType) {},
      getAcceptedIssuers: function() { return []; }
    }
  });

  // Hook SSLContext.init to inject our TrustManager
  var SSLContext = Java.use("javax.net.ssl.SSLContext");
  SSLContext.init.overload(
    "[Ljavax.net.ssl.KeyManager;",
    "[Ljavax.net.ssl.TrustManager;",
    "java.security.SecureRandom"
  ).implementation = function(km, tm, sr) {
    this.init(km, [TrustManager.$new()], sr);
  };
});
```

**Objection** (simplest approach):

```bash
# One command, handles most pinning libraries
objection -g com.target.app explore -s "android sslpinning disable"
objection -g com.target.app explore -s "ios sslpinning disable"
```

**Android 7+ system CA injection** (for apps that only trust system CAs):

```bash
# Convert Burp cert to Android system format
openssl x509 -inform DER -in burp.der -out burp.pem
HASH=$(openssl x509 -inform PEM -subject_hash_old -in burp.pem | head -1)
cp burp.pem /system/etc/security/cacerts/${HASH}.0
chmod 644 /system/etc/security/cacerts/${HASH}.0
# Reboot or remount /system
```

### 4.5 Pinning Libraries by Platform

| Platform | Library | Bypass Difficulty |
|----------|---------|-------------------|
| Android  | Network Security Config | Low (Frida/Objection) |
| Android  | OkHttp CertificatePinner | Low |
| Android  | TrustKit-Android | Low |
| Android  | Custom X509TrustManager | Medium (must identify class) |
| iOS      | TrustKit | Low (Objection) |
| iOS      | AFNetworking/Alamofire | Low |
| iOS      | Custom NSURLSession delegate | Medium |
| Both     | Certificate Transparency + pinning | High (multiple checks) |

## 5. Mobile Malware Analysis Methodology

### 5.1 Triage Phase

1. **Obtain the sample**: Download from app store, pull from device, or receive from incident
   response. Calculate hashes (MD5, SHA-256) for tracking.
2. **VirusTotal / malware bazaar check**: Submit hash (never submit confidential apps).
   Check for known family attribution.
3. **Manifest review**: Examine permissions, components, intent filters. Red flags:
   - `RECEIVE_SMS`, `SEND_SMS` (banking trojans)
   - `BIND_ACCESSIBILITY_SERVICE` (overlay attacks)
   - `BIND_DEVICE_ADMIN` (ransomware, device lock)
   - `SYSTEM_ALERT_WINDOW` (clickjacking)
   - Receivers for `BOOT_COMPLETED` (persistence)

### 5.2 Static Analysis Phase

```bash
# Automated scan with MobSF
# Upload to MobSF web interface -- produces comprehensive report

# Manual decompilation
jadx -d decompiled target.apk
# or for iOS:
class-dump -H App.app/App -o headers/

# Identify suspicious patterns
grep -rn "DexClassLoader\|PathClassLoader\|loadDex" decompiled/  # dynamic code loading
grep -rn "getRuntime().exec\|ProcessBuilder" decompiled/          # command execution
grep -rn "TelephonyManager\|getDeviceId\|getSubscriberId" decompiled/  # device fingerprinting
grep -rn "SmsManager\|sendTextMessage" decompiled/                # SMS abuse
grep -rn "AccessibilityService\|onAccessibilityEvent" decompiled/ # overlay/keylogging
grep -rn "Cipher\|SecretKeySpec\|AES\|DES" decompiled/           # crypto (C2 comms, payload decryption)

# String analysis
strings libnative.so | grep -i "http\|socket\|cmd\|shell\|su"

# Certificate analysis
keytool -printcert -jarfile target.apk
# Self-signed or debug certs in production are red flags
```

### 5.3 Dynamic Analysis Phase

Set up an isolated analysis environment:

1. **Network isolation**: Use an emulator or device on an isolated network segment.
   Route all traffic through a transparent proxy.
2. **DNS monitoring**: Run a local DNS server to log all resolution requests.
   Reveals C2 domains without executing the full payload.
3. **Behavioral monitoring**:

```bash
# Monitor file system changes
inotifywait -mr /data/data/com.suspicious.app/ -e create,modify,delete

# Monitor network connections
adb shell cat /proc/net/tcp    # active TCP connections
strace -f -e network -p <pid>  # system calls

# Monitor API calls with Frida
frida-trace -U -j "*!*http*" -j "*!*connect*" com.suspicious.app

# Hook crypto operations to extract keys
Java.perform(function() {
  var Cipher = Java.use("javax.crypto.Cipher");
  Cipher.doFinal.overload("[B").implementation = function(input) {
    console.log("[Cipher.doFinal] input: " + bytesToHex(input));
    var result = this.doFinal(input);
    console.log("[Cipher.doFinal] output: " + bytesToHex(result));
    return result;
  };
});
```

### 5.4 C2 Communication Analysis

- **Protocol identification**: Most mobile malware uses HTTPS, but some use custom protocols
  over raw sockets, or abuse legitimate services (Telegram bots, Firebase, Pastebin).
- **Domain generation algorithms (DGA)**: Some malware generates C2 domains algorithmically.
  Hook `InetAddress.getByName()` or `nslookup` to capture generated domains.
- **Payload decryption**: Many trojans download encrypted second-stage payloads. Hook the
  decryption function to capture the plaintext payload.
- **Traffic replay**: Capture C2 traffic with mitmproxy, then replay modified requests to
  understand the protocol.

### 5.5 Common Mobile Malware Families (CEH Reference)

| Family | Type | Platform | Technique |
|--------|------|----------|-----------|
| Joker/Bread | Billing fraud | Android | SMS interception, WAP billing |
| FluBot | Banking trojan | Android | Overlay attacks, SMS worm |
| Pegasus | Spyware | iOS/Android | Zero-click exploits, full device compromise |
| Cerberus | Banking trojan | Android | Accessibility abuse, screen recording |
| XcodeGhost | Supply chain | iOS | Infected Xcode, data exfiltration |

### 5.6 Indicators of Compromise (IOCs)

When documenting mobile malware, capture:

- **File hashes**: SHA-256 of APK/IPA and individual DEX/SO files
- **Package names and signing certificates**: Often reused across campaigns
- **C2 infrastructure**: Domains, IPs, URL paths, TLS certificate fingerprints
- **Mutex/lock files**: Files created to prevent duplicate execution
- **Permissions profile**: The specific combination of permissions requested
- **Behavioral signatures**: Specific API call sequences (e.g., register receiver for
  SMS -> read SMS -> send HTTP POST)
