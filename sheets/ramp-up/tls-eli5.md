# HTTPS / TLS — ELI5 (Secret Codes and the Internet's Sealed Envelopes)

> HTTPS is just regular HTTP wrapped in a magic invisible envelope called TLS — and TLS is the lock-and-key system that makes the envelope work.

## Prerequisites

(none — but `cs ramp-up linux-kernel-eli5` helps if you want to know what's
running below)

This sheet is the very first stop for anyone who has ever wondered why some websites have a little padlock and others do not, why "HTTPS" has an "S" at the end and "HTTP" does not, why your browser sometimes throws up a giant red scary page that yells about "certificate errors," and what on earth a "TLS handshake" actually is. You do not need to know any math. You do not need to know any programming. You do not need to know any networking. By the end of this sheet you will have run real commands against real websites and you will have watched the secret handshake happen with your own eyes.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition, and there are over eighty of them.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is HTTPS?

### Imagine you are passing notes in class

Imagine you are eight years old and you are sitting in the back of a classroom. Your best friend is sitting four rows away on the other side of the room. You want to tell your best friend something important. Maybe you want to tell them about the funny thing the lunch lady did. Maybe you want to ask them what they got on the math quiz. Maybe you want to plan what to do at recess.

You can't just shout it across the room. The teacher will hear. So you write it down on a little folded piece of paper. Now you have a problem: how does the note get from you to your friend?

You can't get up and walk over there. So you have to hand the note to the kid sitting next to you. That kid passes it to the next kid. The next kid passes it to the next kid. And so on, all the way across the classroom, until it reaches your friend.

Here is the problem. **Every kid in the chain can read the note.** They can unfold it. They can read it. They can fold it back up and pretend they didn't read it. They can even change what's written on it — they could cross out "let's play tag at recess" and write "let's eat worms at recess" and your friend would read it and think it came from you. They could throw the note away and tell you it never arrived. They could write a note pretending to be from your friend and pass it back to you.

That is the regular internet. That is HTTP. **HTTP is passing notes folded once with no envelope through a chain of strangers, every single one of whom can read or change what you wrote.**

### The sealed envelope

Now imagine instead you have a magic envelope. You put your note inside the envelope. You seal the envelope with a special wax seal that only you and your best friend know how to make. You hand the envelope to the kid next to you. They pass it down the chain. None of them can open it. None of them can read it. None of them can change it without breaking the seal — and if the seal gets broken, your friend will see immediately and know not to trust the note.

That is HTTPS. **HTTPS is HTTP wrapped in a magic envelope that only the sender and the receiver can open.**

The "S" in HTTPS stands for **Secure**. The magic that makes the envelope work is called **TLS** — Transport Layer Security. TLS is the lock-and-key system that creates the envelope and seals it.

So if anyone ever asks you, "what is the difference between HTTP and HTTPS," the answer is:

> HTTP is a postcard. HTTPS is a sealed envelope. TLS is the wax seal.

That's it. That's the whole core idea. Everything else in this sheet is just *how the wax seal works* and *why nobody can fake it*.

### Postcards versus envelopes

Let's make a little table that compares the two:

| Thing                  | HTTP (postcard)                                  | HTTPS (envelope)                                 |
| ---------------------- | ------------------------------------------------ | ------------------------------------------------ |
| What it's like         | A postcard everyone in the mail can read         | A locked envelope only the recipient can open    |
| Your password          | Visible to anyone watching                       | Looks like complete gibberish to snoopers        |
| Your credit card       | Visible — basically printed on the side          | Totally scrambled, completely safe               |
| Your bank balance      | Visible — your bank statement on a postcard      | Hidden — only you and the bank can read it       |
| The URL bar            | Says `http://`                                   | Says `https://` with a little padlock icon       |
| ISP can read it        | Yes                                              | No                                               |
| Coffee shop wifi snoop | Yes                                              | No                                               |
| Government can read    | Yes (with a tap on the wire)                     | No (without breaking the seal — see Forward Secrecy) |
| Hacker on your network | Yes                                              | No                                               |
| Can it be changed      | Yes — anyone can edit                            | No — any change breaks the seal and you'd know   |
| Default in 2026        | No — every browser warns                         | Yes — every modern site                          |

If a website you log in to does not show that little padlock, do not log in. Your password is being sent across the internet in plain text where anyone in the chain can read it. Close the tab and walk away.

### The "S" is the seal

If you remember nothing else from this entire sheet, remember this:

> The S in HTTPS is the **seal** on the envelope.

The seal does three things at once:
1. **Privacy.** The note inside is encrypted — gibberish to anyone who isn't you or the website.
2. **Authentication.** You know the envelope really came from the website it claims to be from. Not a fake. Not an impersonator.
3. **Integrity.** Nobody messed with the note in transit. If the seal is broken, you know.

These three things are the whole job of TLS. Privacy + authentication + integrity. That's it. Every single feature of TLS — every cipher, every certificate, every handshake step — is in service of one of those three goals.

### What does the lock icon actually mean?

When you go to a website and you see the little padlock icon in the address bar, what your browser is telling you is:

1. The site is using HTTPS. (The envelope is being used.)
2. The certificate the site presented is signed by a Certificate Authority your browser trusts. (The wax seal looks legit.)
3. The certificate has not expired and has not been revoked. (The seal is fresh.)
4. The certificate's name matches the domain you're visiting. (The seal says the right name on it.)

If any of those four things is wrong, the lock turns into a scary red warning page. Most modern browsers won't even let you click past it without typing in a magic phrase or hitting some hidden button. That is on purpose. It is meant to be hard. Because if the seal is wrong, the most likely explanation is that someone is trying to eavesdrop on you or impersonate the site.

### When did the internet get HTTPS?

The internet did not start out with HTTPS. It started out with HTTP — postcards. For about the first ten years (1989 through the late 1990s) almost everything was on postcards. Banks. Email. Logins. Passwords. All postcards. People did not realize the danger because the internet was small and felt private. It was not.

Then in 1994 a company called Netscape (the people who made the very first popular web browser) invented something called **SSL** — Secure Sockets Layer. SSL was the first version of the envelope. It was used mostly for online shopping at first. By the late 1990s SSL had been renamed to **TLS** (because the standards committee that took it over decided "Secure Sockets Layer" was a Netscape brand name and they wanted a neutral name). TLS version 1.0 came out in 1999. Version 1.1 in 2006. Version 1.2 in 2008. Version 1.3 in 2018.

For a long time HTTPS was used only on important things — banking, email, shopping. Most regular websites used HTTP because TLS was slow and certificates cost money. Then between 2014 and 2018 a few things happened:

- Computers got fast enough that TLS basically had no speed cost.
- A new free certificate authority called **Let's Encrypt** started giving out certificates for free.
- Browsers started shaming any site that used HTTP by showing a "Not Secure" warning.
- Google started using HTTPS as a search ranking signal.

By 2026 essentially every real website on the internet uses HTTPS. HTTP is dead in the water. If you see a site on `http://` today, something is either very wrong or very old.

## Why You Can't Just Use One Big Lock

### The locked-box problem

Now we get to the very first puzzle. Sealing an envelope is easy if you and your friend are sitting in the same room. You just write down a secret code on a piece of paper, hand it to your friend, and then later use that secret code to scramble your messages.

But the internet is not the same room. The internet is the whole entire world.

When you visit a website for the first time, you have never met that website before. You have no shared secret. You have nothing in common. You are a complete stranger to the website, and the website is a complete stranger to you. You can't even hand the website a piece of paper through the screen. Anything you send to the website goes through hundreds of routers and switches owned by hundreds of different companies in dozens of countries, and any one of those routers could be reading along.

So how do you create a shared secret with a website you have never met before, when the only way to talk to the website is through a chain of strangers who can read everything you say?

This is called the **key exchange problem** and it is one of the deepest puzzles in all of computer science. For decades nobody knew how to solve it. The standard answer was "you can't — you have to meet in person first." Spies during the Cold War would have to physically meet to exchange one-time-pad codebooks. Embassies would send couriers with diplomatic pouches. There was no way for two strangers to create a shared secret without meeting first.

Then in 1976 two researchers named Whitfield Diffie and Martin Hellman figured out a trick.

### Two kinds of locks

Before we get to the trick, we need to understand that there are two completely different families of secret codes:

**Family one: symmetric codes** — these are codes where the same key locks and unlocks. You have a decoder ring. Your friend has a copy of the same decoder ring. You both use the same ring to scramble and unscramble. These codes are super fast. They can scramble billions of bits per second. But they have one big problem: how did you and your friend both get the ring in the first place?

**Family two: asymmetric codes** — these are codes where there are *two different keys*: a key that locks (the public key) and a key that unlocks (the private key). The lock and unlock keys are mathematically related but it is essentially impossible to figure out the unlock key from the lock key. So you can hand out the lock key to absolutely everybody — even your enemies — and only you, with the unlock key, can read what they sent. These codes solve the key-exchange problem. But they are very slow. They can scramble maybe a few thousand bits per second.

So neither one alone is good enough for the internet. Symmetric codes are fast but you can't get the key to your friend. Asymmetric codes solve the key-getting problem but they are too slow to actually use for everyday data.

The trick TLS uses is to combine them: **use the slow asymmetric code to set up a shared secret, then use the fast symmetric code to actually send all your data.** That is the entire architecture of TLS in one sentence. It is brilliant. It is simple. It works.

### The magic mailbox

Let's make the asymmetric idea concrete with a picture.

Imagine you have a special mailbox. The mailbox has two parts: a slot on the top that anybody can drop letters into, and a door on the side that has a key. Anyone walking down the street can drop a letter through the slot. Once a letter is in the box, nobody can fish it back out. Only you, the owner of the mailbox, have the key to the side door, and only you can open the door and take letters out.

You can put copies of the slot anywhere. You can hand out copies. You can paint the slot on a billboard. You can publish a picture of the slot in the newspaper. The slot is **public**. Anyone who wants to send you a letter can drop one in. But only you, with your private key, can ever take letters out.

That is asymmetric encryption. The slot is the **public key**. The side-door key is the **private key**. Anyone can lock something with the public key. Only the owner can unlock it.

Now imagine the website has one of these mailboxes. The website has published the public slot. The website is keeping the private side-door key locked in a safe in their server room. You, the visitor, want to talk to the website. You take a piece of paper, write "let's use this number as our shared secret: 73928473," fold it up, and drop it through the public slot. Anybody walking by can see you drop a letter in the slot — that's fine. The slot only goes one direction. Once the letter is in the box, only the website can open the box and read it.

The website opens the box. Now both of you know the secret shared number. You agreed on a secret number without ever meeting and without anybody who was watching being able to learn the number.

That is, roughly, how the original RSA-based TLS handshake worked. (We will see in a moment that TLS 1.3 uses a slightly different and even cooler trick called Diffie-Hellman, but the idea is the same.)

### The shared decoder ring

Once you have a shared secret number, both sides can use it as the key for a fast symmetric code. Symmetric codes use the same key on both ends. So now you and the website both have the same secret number. You both feed the secret number into a symmetric encryption algorithm — say AES — and now every message you send is scrambled with that algorithm and unscrambled at the other end.

The symmetric code is fast. Modern CPUs have special hardware instructions for AES so they can encrypt and decrypt at literally the speed of memory bandwidth. Several gigabits per second. You don't notice it. The website doesn't notice it. The internet doesn't notice it. But if anyone is watching the wire, all they see is random-looking bytes.

| Lock type           | How many keys      | Speed                                      | What it's good for                                   | Real-world example                                |
| ------------------- | ------------------ | ------------------------------------------ | ---------------------------------------------------- | ------------------------------------------------- |
| Asymmetric (mailbox)| Two: public+private| Slow (kilobytes per second)                | Setting up a shared secret with a stranger           | RSA, ECDSA, Ed25519, X25519                       |
| Symmetric (decoder ring) | One: shared    | Super fast (gigabits per second)           | Encrypting all your actual data after setup          | AES-128-GCM, AES-256-GCM, ChaCha20-Poly1305       |

You will see this two-step pattern over and over in cryptography. Use the slow expensive thing to set up a session. Use the fast cheap thing to do the work. It is almost always the right answer.

### The Diffie-Hellman paint trick

The Diffie-Hellman key exchange is even cooler than the magic mailbox. It uses a trick that lets two people who have never met agree on a shared secret in public — *without anyone needing a private mailbox at all*. It works like this. (Pay attention; this is one of the most beautiful ideas in all of computer science.)

Imagine you and your friend want to agree on a secret color. But you have to do it in public. Anyone listening must not be able to figure out the color.

1. You and your friend publicly agree on a starting color. Say, **yellow**. Everyone hears you. That's fine. The starting color is not secret.
2. Privately, in your head, you pick a secret color. Say, **red**. You don't tell anybody. Privately, your friend picks a secret color too. Say, **blue**.
3. You take your secret red and mix it with the public yellow. You get **orange**. You send the orange to your friend. Anyone watching sees you send orange.
4. Your friend takes their secret blue and mixes it with the public yellow. They get **green**. They send the green to you. Anyone watching sees them send green.
5. Now you take the green you got from your friend and mix it with your secret red. You get **brown**.
6. Your friend takes the orange they got from you and mixes it with their secret blue. They get **brown** too.

You and your friend both have brown. The eavesdropper saw yellow, orange, and green — but they cannot figure out brown. Because the only way to make brown is to combine your secret red with their secret blue, or to start from green and add red, or start from orange and add blue. The eavesdropper has none of those secrets. They have only the public mixtures.

That is Diffie-Hellman. The colors are actually huge numbers (hundreds of digits long). The "mixing" is actually a math operation called modular exponentiation, or in modern TLS, point multiplication on an elliptic curve. The "secret colors" are random numbers each side picks. The "public mixtures" are points on the curve. The "brown at the end" is the shared secret that both sides derive without ever transmitting it directly.

In TLS 1.3 the version of this used is called **ECDHE** — Elliptic Curve Diffie-Hellman Ephemeral. The "ephemeral" part means each session uses fresh random colors. We will come back to that in the section on Forward Secrecy because it is the single most important property of modern TLS.

### Why both halves matter

So in TLS, you actually get *both* kinds of crypto in one connection:

- **Asymmetric crypto** is used for two things:
  1. The key exchange — agreeing on a shared secret with a stranger.
  2. Authentication — proving the server is who it says it is, by signing things with its private key.
- **Symmetric crypto** is used for one thing:
  1. Encrypting all the actual data after the handshake is done.

You need both. Asymmetric without symmetric would be too slow. Symmetric without asymmetric would have no way to bootstrap. Neither one alone is enough. Together they are perfect.

## The Handshake (Establishing the Secret)

### What is a handshake, anyway?

When two people meet, they shake hands. The handshake is a quick ritual. It says "I am here, you are there, we are now in contact, let us begin." It is short. It is conventional. Both parties know exactly what to do. They grasp hands, pump twice, let go.

A TLS handshake is the same idea except instead of two people, it is two computers, and instead of grasping hands, they exchange a short series of messages. The handshake takes a few milliseconds. By the end of it, both computers have agreed on:

- Which version of TLS to use (1.2 or 1.3 — almost always 1.3 in 2026).
- Which cipher suite to use (which symmetric algorithm and hash).
- Who the server is (proven by certificate).
- What the shared secret is (derived via ECDHE).

After the handshake is done, both sides flip a switch and start encrypting everything.

### The TLS 1.3 handshake at ELI5 level

Let's walk through the handshake step by step at ELI5 level. We are using TLS 1.3 — the modern version everyone should be using. The TLS 1.3 handshake takes **one round trip**, meaning one message from client to server and one from server to client and then we can start sending data. (TLS 1.2 took two round trips. TLS 1.3 cut it in half.)

Imagine you are walking up to the front desk of a fancy hotel.

```
You (client)                                        Hotel (server)
    |                                                    |
    |  "Hi! I'm here. Here's a list of secret           |
    |  handshakes I know how to do, and here's          |
    |  half of a secret-color exchange."                |
    |  ----- ClientHello + key_share ------------------>|
    |                                                    |
    |                                                    |  *picks a handshake*
    |                                                    |  *finishes the color*
    |                                                    |  *grabs ID badge*
    |                                                    |  *signs everything*
    |                                                    |
    |  "Hi back! Let's use this handshake. Here's       |
    |  the other half of the color. Here's my ID        |
    |  badge. Here's a signature proving I really       |
    |  am the hotel. And here's my Finished signal."    |
    |  <---- ServerHello + key_share + Cert + Sig + Fin |
    |                                                    |
    |  *checks ID badge*                                 |
    |  *finishes the color*                              |
    |  *verifies signature*                              |
    |  "Great, your ID checks out. Here's my            |
    |  Finished signal."                                |
    |  ----- Finished -------------------------------->|
    |                                                    |
    |  ============= Encrypted application data ========|
    |  ====================== flows freely =============|
```

That's it. One trip from you to the hotel. One trip back. Then encrypted talk. Total time: usually a few tens of milliseconds.

### Step 1: ClientHello — "Hi, I'd like to be secret friends!"

The very first message your computer sends is called **ClientHello**. It says, in computer language:

- "Hi! I am a TLS 1.3 client."
- "Here is a random number I just generated. Please use it to make my keys unique."
- "Here is a list of cipher suites I know how to do. Pick one."
- "Here is the hostname I am trying to reach. (This is the Server Name Indication or SNI.)"
- "Here is a list of application protocols I'm willing to use, in order of preference. (This is ALPN: h2, http/1.1, etc.)"
- "Here is half of a Diffie-Hellman key exchange. (This is `key_share`.)"
- "Here are the elliptic curve groups I support."
- "Here are the signature algorithms I accept."

The ClientHello is **not encrypted**. Anybody on the wire can see it. This is fine. Nothing in it is a secret. The only things in it that *will become* a secret are derived from the random number and the key share, and those need both halves of the key exchange to mean anything.

Notice that the client puts its half of the Diffie-Hellman key share *in the very first message*. This is a TLS 1.3 trick. In TLS 1.2 the key exchange happened later, which is why TLS 1.2 took two round trips. By being optimistic — by guessing what curve the server will pick and including the share up front — TLS 1.3 saves a whole round trip.

### Step 2: ServerHello — "Pleased to meet you. Here's my ID."

The server replies with **ServerHello**. The server says:

- "Hi! I am also a TLS 1.3 server."
- "Here is a random number I just generated."
- "I picked **this cipher suite** from your list. We will use AES-256-GCM with SHA-384." (or one of the other choices.)
- "Here is the other half of the Diffie-Hellman key exchange."

At this point both the client and the server have everything they need to derive the shared secret. The client has its own private color and the server's public mixture. The server has its own private color and the client's public mixture. Both can derive the same shared secret. Both do this calculation. Now they have a shared secret called the **handshake secret**, and from this they derive the **handshake traffic keys**.

From this moment on, everything else in the handshake is encrypted.

### Step 3: EncryptedExtensions, Certificate, CertificateVerify

These all happen in the same round trip as ServerHello but are *encrypted* with the handshake key:

- **EncryptedExtensions**: extra negotiated parameters that don't need to be public. (Things like ALPN choice.)
- **Certificate**: the server's ID badge. This is the certificate chain — leaf certificate plus any intermediates. We'll talk about certificates in the next section.
- **CertificateVerify**: a signature, made with the server's private key, over a hash of every handshake message so far. This proves two things at once: that the server really has the private key matching the public key in the certificate, and that nobody tampered with any of the previous messages.
- **Finished**: an HMAC over the handshake transcript, proving the handshake came through unmodified end to end.

In TLS 1.2 these last three steps were all unencrypted. In TLS 1.3 they are encrypted with the handshake key. This means the server's certificate — which contains the server's name — is encrypted. An eavesdropper watching a TLS 1.3 handshake cannot see *which website* you are visiting based on the certificate. (They can still see the SNI in the unencrypted ClientHello — but ECH, which we'll see in vocab, is fixing that too.)

### Step 4: Client Finished — "I'm satisfied. Let's go."

The client now has the certificate. It does these checks:

- Is the certificate signed (eventually) by a CA in my trust store?
- Has it not yet expired?
- Has it not yet been revoked?
- Does the name on it match the hostname I asked for?
- Does the CertificateVerify signature actually verify with the public key in the cert?
- Does the Finished MAC match what I expect?

If all of those are yes, the client is satisfied. It sends its own Finished message, also encrypted with the handshake key. Both sides now derive the **application traffic keys** from the handshake transcript and the master secret. The handshake is done.

### Step 5: Application data starts flowing

Both sides switch to the application traffic keys. Every byte from now on — every HTTP request, every HTTP response, every HTML page, every cookie, every download, every WebSocket message, every gRPC stream — is encrypted with AES-GCM (or ChaCha20-Poly1305) using the application traffic key.

The eavesdropper sees only random-looking bytes.

### TLS 1.2 vs TLS 1.3 — why we love 1.3

Let's compare the two:

| Feature                  | TLS 1.2                                   | TLS 1.3                                  |
| ------------------------ | ----------------------------------------- | ---------------------------------------- |
| Round trips before data  | 2 (2-RTT)                                 | 1 (1-RTT) — half the latency             |
| 0-RTT resumption         | Hacky non-standard                        | Built in (with replay caveat)            |
| Cipher suites            | Many — some weak (RC4, 3DES, CBC modes)   | 5 — all strong AEAD                      |
| Key exchange             | RSA or ECDHE — RSA had no forward secrecy | ECDHE only — forward secrecy required    |
| Static RSA               | Allowed                                   | Removed                                  |
| Compression              | Allowed (CRIME-vulnerable)                | Removed                                  |
| Renegotiation            | Allowed (attack surface)                  | Removed                                  |
| MAC algorithms           | MAC-then-encrypt (lucky 13)               | AEAD only                                |
| Certificate encrypted    | No                                        | Yes                                      |
| Most of handshake encrypted | No                                     | Yes (after ServerHello)                  |
| Hash for HKDF            | Variable                                  | SHA-256 or SHA-384                       |
| Year ratified            | 2008                                      | 2018 (RFC 8446)                          |

TLS 1.3 is faster, simpler, more secure, and removes basically every algorithm that has had a serious vulnerability over the last two decades. Use it. Disable 1.2 if you can. Anything older than 1.2 has been deprecated since 2020 and is disabled in all major browsers.

### 0-RTT (early data) — the dangerous shortcut

TLS 1.3 has an option called **0-RTT** or **early data**. It works like this:

The first time you connect to a server, you do a normal 1-RTT handshake. At the end, the server sends you a **session ticket** — an encrypted blob that the server can later decrypt to remember who you are. You save the ticket.

The next time you connect to the same server, you can include the session ticket in your ClientHello, AND you can include encrypted application data in the very first message — before the handshake is finished. Latency: zero round trips before data. This is faster than even a fresh TLS 1.3 handshake.

The catch: **0-RTT data has no replay protection.** An attacker who recorded your 0-RTT data can replay it later — possibly multiple times — and the server may execute it multiple times. If your 0-RTT request was "transfer $100 from my account," replaying that is bad.

The rule of thumb: only use 0-RTT for **idempotent** operations. GETs are usually safe. POSTs are usually not. RFC 8470 defines an HTTP status code 425 ("Too Early") for servers that want to say "this needs the full handshake, please retry."

In practice almost no web servers enable 0-RTT by default. CDNs sometimes use it for cached GETs. If you don't know what you're doing, leave it off.

## Certificates: How Do You Trust the Lock?

### The ID badge problem

Recall that during the handshake, the server sends you its certificate. The certificate contains the server's public key and the server's name. The client uses the public key as the foundation for the key exchange and for verifying signatures.

But here's the thing: how do you know the public key actually belongs to the server you're trying to talk to? An attacker in the middle could substitute their own public key, do the key exchange with you using their key, do another handshake with the real server, and decrypt and re-encrypt everything in between, reading and modifying as they go. This is called a **man-in-the-middle attack** or MITM.

The defense against MITM is the **certificate**. A certificate is the server saying "this is my public key" plus a signature from a trusted third party saying "yes, this is really their public key." The trusted third party is called a **Certificate Authority** or **CA**.

### CAs as the DMV

A Certificate Authority is like the Department of Motor Vehicles. The DMV's job is to verify that you are who you say you are, look at your birth certificate, look at your other ID, and then issue you a driver's license that has your name on it and the DMV's hologram seal. Once you have the license, you can show it anywhere as proof of who you are. People trust the license because they trust the DMV did its job.

A CA does the same thing for websites. The CA verifies that you (the website operator) really own the domain `bellis.tech` (usually by making you put a special file at `bellis.tech/.well-known/acme-challenge/...` or by making you add a special DNS record). Once verified, the CA signs a certificate that says "the public key in this certificate belongs to whoever controls bellis.tech." Anyone can then look at the certificate, verify the CA's signature, and trust that the public key is the real one.

Your computer, your phone, and your browser all come pre-loaded with a list of about 100 to 150 trusted root CAs. When you visit a website, your browser walks the certificate chain back from the leaf to a root in the trust store. If the chain is good and the root is trusted, the cert is valid.

### Root, intermediate, leaf — the chain of trust

```
                    Your computer's trust store
                           (Mozilla / Apple / Microsoft / Google)
                                       |
                                       | <-- contains 100-150 root CA certs
                                       v
                    +--------------------------------------+
                    |          Root CA Certificate          |
                    |       (self-signed, in trust store)   |
                    |     "I am ISRG Root X1; trust me"     |
                    +--------------------------------------+
                                       |
                                       | <-- root signs intermediate
                                       v
                    +--------------------------------------+
                    |     Intermediate CA Certificate      |
                    |     (signed by Root CA)              |
                    |   "I am Let's Encrypt R3, signed     |
                    |    by ISRG Root X1; trust me too"    |
                    +--------------------------------------+
                                       |
                                       | <-- intermediate signs leaf
                                       v
                    +--------------------------------------+
                    |        Leaf / Server Certificate     |
                    |     (signed by Intermediate)         |
                    |  "I am bellis.tech, signed by R3"    |
                    +--------------------------------------+
```

Three layers:

- **Root CA**: at the top. Self-signed (it signs its own cert because there is nothing above it). The private key is kept offline in a literal physical safe in a literal physical bunker. It only comes out for ceremonies that are video recorded. The root CA only signs intermediates, never end-entity certificates. There are about 100 to 150 of these in your trust store.
- **Intermediate CA**: in the middle. Signed by a root. Used for day-to-day signing. If an intermediate gets compromised it can be revoked without revoking the root. Most CAs have a few intermediates each.
- **Leaf certificate**: the actual cert your website uses. Signed by an intermediate. Has your domain name on it. Lasts 90 days (Let's Encrypt) to about 13 months (paid CAs in 2026 — used to be longer but the limit keeps going down).

When the server sends you its certificate, it should send the **whole chain** (leaf + all intermediates). The root is already in your trust store, so the server doesn't need to send it. Your browser walks: leaf signed by intermediate (good), intermediate signed by root (good), root in trust store (good). Chain validates. Cert is trusted.

### Subject Alternative Name (SAN) — all the nicknames

A certificate covers a list of domain names. The list is in a field called **Subject Alternative Name** or SAN. SAN looks like:

```
DNS: bellis.tech, DNS: www.bellis.tech, DNS: api.bellis.tech, DNS: *.bellis.tech
```

That last one is a **wildcard**. `*.bellis.tech` matches `foo.bellis.tech`, `bar.bellis.tech`, anything-bellis.tech. Wildcards can only be one level deep — `*.bellis.tech` does NOT match `foo.bar.bellis.tech`.

There used to also be a field called **Common Name (CN)**. CN was the original way to put the domain name in the cert. CN has been deprecated in favor of SAN since 2017. Modern browsers ignore CN and only look at SAN. If a cert has only CN and no SAN, it will fail validation.

If you visit `https://bellis.tech` and the cert's SAN does not include `bellis.tech`, you get the error `x509: certificate is valid for X, not Y`. We'll see that error later.

### Certificate Transparency (CT)

There's one more layer of trust on top of the CA system: **Certificate Transparency**, defined in RFC 6962. CT requires every CA to log every certificate it issues to public, append-only logs. Before a browser will trust a certificate, the cert must include 2 or 3 **Signed Certificate Timestamps (SCTs)** proving it was logged.

Why does this matter? Because if a CA is compromised, or if a CA misbehaves, or if a CA gets tricked into issuing a certificate for a domain to someone who doesn't own that domain, the misissuance shows up in the public log. Domain owners can monitor the logs (using tools like `crt.sh`) and see if anyone has issued a certificate for their domain that they didn't request. Misbehaving CAs get distrusted and removed from browser trust stores.

CT is the reason Symantec, the world's largest CA at the time, was distrusted in 2017. They had been issuing certificates badly and CT made the badness public. Browsers stopped trusting them. Symantec sold the CA business and exited.

### ACME and Let's Encrypt

Until 2015 you had to pay money for a certificate. CAs charged anywhere from $10 to several hundred dollars per cert per year. The CA business was very profitable. This is one reason much of the web stayed on HTTP for so long — small operators didn't want to pay.

In 2015 the Internet Security Research Group (ISRG) launched **Let's Encrypt**, a free, automated, nonprofit CA. Along with it they defined the **ACME** protocol (Automated Certificate Management Environment, RFC 8555). ACME lets a server prove it controls a domain (via HTTP-01 challenge or DNS-01 challenge) and get a certificate, all in seconds, with no humans in the loop. Tools like `certbot` automated the whole thing.

By 2026 Let's Encrypt has issued over 4 billion certificates and most of the public web is on free, auto-renewed Let's Encrypt certs that rotate every 90 days.

## Cipher Suites

### The menu of locks

Recall that during the handshake the client sends a list of cipher suites it supports, and the server picks one. A cipher suite is a bundle of algorithms — a "menu of locks" they pick from. In TLS 1.3, a cipher suite has two parts:

1. The symmetric AEAD algorithm (used for encrypting application data).
2. The hash function (used for the HKDF key derivation).

The five cipher suites in TLS 1.3 are:

| Cipher suite                   | Symmetric algorithm    | Hash       | Key size  | Where used                              |
| ------------------------------ | ---------------------- | ---------- | --------- | --------------------------------------- |
| TLS_AES_128_GCM_SHA256         | AES-128-GCM            | SHA-256    | 128 bits  | Default for most servers; fast on x86   |
| TLS_AES_256_GCM_SHA384         | AES-256-GCM            | SHA-384    | 256 bits  | Higher security; slightly slower        |
| TLS_CHACHA20_POLY1305_SHA256   | ChaCha20-Poly1305      | SHA-256    | 256 bits  | Fast on phones/ARM without AES hardware |
| TLS_AES_128_CCM_SHA256         | AES-128-CCM            | SHA-256    | 128 bits  | Embedded systems                        |
| TLS_AES_128_CCM_8_SHA256       | AES-128-CCM (8-byte tag)| SHA-256   | 128 bits  | Embedded with very tight bandwidth      |

In practice you will see the first three. The CCM ones are for tiny devices that can't do GCM efficiently.

### What is AEAD?

**AEAD** stands for **Authenticated Encryption with Associated Data**. It is a kind of symmetric encryption that does two things at once:

1. **Encrypts** the plaintext (privacy).
2. **Authenticates** the ciphertext (integrity — detects tampering).

Older modes (CBC) did encryption and integrity in two separate steps and turned out to have subtle bugs (Lucky 13, BEAST). AEAD does both in a single combined operation that has been formally analyzed and proven secure.

The two AEAD families used in TLS 1.3:

- **AES-GCM**: AES in Galois/Counter Mode. Very fast on x86 CPUs that have AES-NI hardware acceleration. Slightly slower on phones.
- **ChaCha20-Poly1305**: a stream cipher (ChaCha20) plus a polynomial MAC (Poly1305). Designed by Dan Bernstein. Constant-time. Fast in software. Used by phones and any platform without AES hardware.

When the AEAD encrypts, it produces a ciphertext plus a 16-byte authentication tag. When the AEAD decrypts, it checks the tag. If the tag doesn't match — even by one bit — the decryption fails and the connection is torn down. This is how TLS detects tampering.

### TLS 1.2 cipher suite naming

TLS 1.2 cipher suite names were verbose because they encoded *everything* in the name: key exchange, signature algorithm, symmetric cipher, and hash. Like:

```
TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
 |    |     |        |       |     |
 |    |     |        |       |     +-- PRF/MAC hash function
 |    |     |        |       +-------- AEAD mode (GCM)
 |    |     |        +---------------- Symmetric algorithm + key size
 |    |     +------------------------- Authentication algorithm (RSA cert)
 |    +------------------------------- Key exchange (Elliptic Curve Diffie-Hellman Ephemeral)
 +------------------------------------ Protocol prefix
```

TLS 1.3 cleaned this up. In TLS 1.3, the key exchange and signature algorithm are negotiated separately from the cipher suite, so the suite name only encodes the symmetric algorithm and hash:

```
TLS_AES_256_GCM_SHA384
TLS_CHACHA20_POLY1305_SHA256
```

### ECDHE — fresh keys every conversation

The "E" at the end of ECDHE stands for **ephemeral**. It means each session uses a *fresh, random* Diffie-Hellman key. The server doesn't reuse the same DH private key for every session — it generates a new one for every handshake.

This is the single most important property of modern TLS. It is called **forward secrecy**. We discuss it in the next section because it deserves its own.

## Forward Secrecy

### The "record now, decrypt later" problem

Here is the deepest "why TLS is amazing" idea. Imagine you are an intelligence agency. You don't know how to break TLS today, but you have lots of disk storage. So you record every encrypted packet that goes across a particular wire. Terabytes of encrypted data per day. You stash it in a giant data center and wait.

Years later, you somehow get hold of the server's private key — maybe by hacking the server, maybe by court order, maybe by buying it on the black market, maybe by waiting for a future cryptanalysis breakthrough. With the private key, can you go back and decrypt all that recorded data?

In old TLS (1.2 with RSA key exchange) the answer was YES. Because the server's private key was used to encrypt the shared secret during the handshake. If you record the handshake and later get the private key, you can decrypt the shared secret and then decrypt the entire session.

In TLS 1.3 (and TLS 1.2 with ECDHE) the answer is NO. The server's long-term private key is used only for *signing*, not for the key exchange. The actual shared secret is derived via ephemeral Diffie-Hellman — using random keys that exist only in RAM during the handshake and are destroyed immediately afterward. There is no recoverable copy of the shared secret anywhere except in the brief minutes the session was active. If you steal the server's long-term private key tomorrow, you still cannot decrypt any past session. The shared secrets are gone forever.

That is **forward secrecy** (or "perfect forward secrecy" if you're feeling fancy).

The eavesdropper can record all the encrypted data they want. Even if they get the private key. Even if they get a court order. Even if they wait fifty years and the math gets faster. The session is gone. The keys are gone. The data is forever opaque.

Forward secrecy is required by TLS 1.3. There is no way to do TLS 1.3 without forward secrecy. (Static RSA key exchange was removed entirely.) For TLS 1.2, you get forward secrecy if and only if the cipher suite starts with `ECDHE_` or `DHE_`.

### Why this matters in real life

Forward secrecy means:

- A server compromise tomorrow does not retroactively expose your past sessions.
- A government compelling the disclosure of the server's key cannot decrypt past traffic.
- A future cryptanalysis breakthrough that can read the server's key cannot read past sessions.
- A lawful intercept of the encrypted wire is useless without active interception.

This is the deepest property of modern TLS. It is the reason post-Snowden everyone moved to ECDHE. It is the reason TLS 1.3 made it mandatory.

### The dark side: you can't decrypt your own logs

The flip side of forward secrecy is: if you wanted to decrypt your own past traffic — say for debugging — you can't. The shared secret was destroyed. There is no copy.

There is a workaround for development: most TLS libraries can be configured to dump session keys to a file (the `SSLKEYLOGFILE` environment variable). You can then load those keys into Wireshark to decrypt captures. But this only works if you set up the dump *before* the session. You can't go back and decrypt a session that has already ended unless you saved the keys at the time.

This is by design. The whole point is that the keys are gone.

## Common TLS Errors and What They Mean

You will run into TLS errors. Eventually. Everyone does. Here is the canonical list, in roughly the order of how often you hit them. Each one shows the verbatim error text, what causes it, and how to fix it.

### `x509: certificate signed by unknown authority`

The literal text the Go runtime emits when a certificate's chain doesn't end at a CA in your trust store.

Why: the server's certificate was signed by a CA your computer doesn't trust. Most common causes:

1. The cert is self-signed (signed by itself, not by any real CA). Common for development.
2. The cert is signed by an internal corporate CA you haven't installed on this machine.
3. The cert is signed by a CA your OS doesn't ship with (rare in 2026).
4. The server is sending an incomplete chain (missing intermediate). Less common but happens.

Fix:

- For development: trust the cert manually. On macOS: `security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain cert.pem`. On Linux: copy the CA cert to `/usr/local/share/ca-certificates/` and run `update-ca-certificates`.
- For Go programs: in code, load the corporate CA into a `tls.Config{RootCAs: pool}`. Don't ever set `InsecureSkipVerify: true` in production.
- For curl: `--cacert path/to/ca.pem` (proper) or `-k` (skip verification, insecure, never in production).
- For incomplete chain: have the server send the full chain. Tools like `openssl s_client -showcerts` show you what the server is actually sending.

### `x509: certificate has expired or is not yet valid`

Why: the current time is outside the certificate's `notBefore`/`notAfter` window.

Most often this means the cert expired and someone forgot to renew it. Less often it means the system clock on the client (or server) is wrong.

Fix:

- Renew the cert (`certbot renew`).
- Check system time: `date`. Fix with `sudo ntpdate pool.ntp.org` (Linux) or by enabling automatic time on macOS/Windows.
- Set up automatic renewal with cron: `0 3 * * * certbot renew --quiet`.

You can check a cert's dates with:

```
$ openssl x509 -in cert.pem -dates -noout
notBefore=Feb 14 00:00:00 2026 GMT
notAfter=May 14 23:59:59 2026 GMT
```

### `x509: certificate is valid for X, not Y`

Why: the SAN list in the certificate does not include the hostname you connected to.

Example: cert is valid for `bellis.tech` and `www.bellis.tech` but you tried to visit `api.bellis.tech`.

Fix:

- Reissue the cert with the right SAN list.
- For wildcards: `*.bellis.tech` covers `api.bellis.tech` and `www.bellis.tech` but does NOT cover `bellis.tech` itself or `foo.api.bellis.tech`. Include both `bellis.tech` and `*.bellis.tech` if you want the apex covered.
- Check the SAN list: `openssl x509 -in cert.pem -ext subjectAltName -noout`.

### `tls: handshake failure`

Why: a generic handshake error. The two sides could not agree on parameters. Most common causes:

1. No cipher suites in common (one side requires TLS 1.2+ and the other only offers TLS 1.0).
2. No supported groups in common (server only supports P-256 and client only supports X25519).
3. No signature algorithms in common.
4. Client cert was required (mTLS) and the client did not provide one.
5. Connection broken mid-handshake (firewall dropping packets).

Fix:

- Run `openssl s_client -connect host:443 -tls1_3 -msg` and look at the actual messages. The server's Alert message will often tell you exactly what went wrong.
- Check `nmap --script ssl-enum-ciphers -p 443 host` to see what the server actually supports.
- For mTLS: provide a client cert with `--cert client.pem --key client.key`.

### `tls: no application protocol`

Why: the client offered ALPN values like `h2,http/1.1` and the server didn't accept any of them. Common when an HTTP/2-only server gets a client that only knows http/1.1, or vice versa.

Fix:

- Make sure both sides agree on at least one ALPN value.
- For Go's `net/http`: HTTP/2 is enabled by default for HTTPS. If you're hitting an http/1.1-only server, something is off.

### `ssl: certificate verify failed`

Why: Python (or another OpenSSL-based tool) couldn't verify the cert chain. Same root cause as `x509: certificate signed by unknown authority` — just different wording.

Fix:

- For Python `requests`: `requests.get(url, verify='/path/to/ca-bundle.pem')`. Or `pip install certifi` and use `certifi.where()`.
- Don't ever use `verify=False` in production.

### `SEC_ERROR_OCSP_*`

Why: Firefox couldn't get a fresh OCSP response and the cert is configured to require it (`OCSP-must-staple`).

Fix:

- Server should enable OCSP stapling (`ssl_stapling on` in nginx).
- If you're seeing this and you're the developer of a client app, your OCSP fetcher might be misconfigured.

### `ERR_CERT_DATE_INVALID` (browser)

Why: Chrome saying the cert is expired or not yet valid. Same as the `x509: ... expired` error above.

Fix: renew the cert. Check the clock.

### `ERR_CERT_AUTHORITY_INVALID` (browser)

Why: Chrome saying the cert chain ends at a CA the browser doesn't trust.

Fix: same as `x509: certificate signed by unknown authority` above.

### `certificate has been revoked`

Why: the cert is in a CRL (Certificate Revocation List) or returns "revoked" from OCSP. The CA has explicitly invalidated this cert — usually because the private key was compromised, or because the cert was issued in error.

Fix: get a new cert. Investigate why the old one was revoked.

### `no cipher suites in common`

Why: a more specific version of `tls: handshake failure`. The client and server cipher suite lists do not intersect.

Fix:

- Check what the server supports: `nmap --script ssl-enum-ciphers -p 443 host`.
- Update one side to support modern ciphers.
- Most common cause: client is too old (e.g., Java 7) and the server has disabled all the old/weak ciphers.

### `wrong version number`

Why: usually means you're talking to a server that isn't actually doing TLS — like a plain HTTP server or a SSH server — on the port you thought was TLS.

Fix:

- Double-check the port and the protocol.
- `openssl s_client -connect host:port` will fail. Try `curl http://host:port/` (no TLS) and see what comes back.

## TLS Attacks: A Brief Tour

A quick tour of historic TLS attacks. The point of this section is not to scare you. The point is to show you that TLS has been thoroughly attacked over the years and the response has been to fix the protocol. TLS 1.3 (the version you should be using) is immune to almost all of these.

### BEAST (2011)

**Browser Exploit Against SSL/TLS.** Affected TLS 1.0 with CBC cipher modes. The IV for each record was the previous record's last block, which was predictable. An attacker could exploit this to recover plaintext one byte at a time.

Fix: use TLS 1.1+ (which uses random IVs) and prefer AEAD modes (GCM, ChaCha20-Poly1305) over CBC. **Use TLS 1.3.**

### CRIME (2012)

**Compression Ratio Info-leak Made Easy.** TLS supported compression. If an attacker could make you send a request that included a guessed secret next to a known prefix, the size of the compressed result would leak whether the guess was correct.

Fix: turn off TLS-level compression. **Use TLS 1.3** (which removed compression entirely).

### BREACH (2013)

Same idea as CRIME but exploiting HTTP-level compression instead of TLS-level. Affects responses that reflect URL parameters into compressed HTTP bodies.

Fix: at the application layer — randomize lengths, mask CSRF tokens, separate sensitive responses from user input. (Not a TLS-level fix.) **Use TLS 1.3** doesn't fix this directly because it's an HTTP-layer issue, but TLS 1.3 + careful application code fixes it.

### Heartbleed (2014)

Not a protocol attack — a bug in OpenSSL's implementation of the heartbeat extension. A bad heartbeat request could read up to 64KB of arbitrary memory from the server, including private keys.

Fix: patch OpenSSL. The bug was fixed within hours of disclosure. Tens of thousands of certificates were rotated worldwide.

### POODLE (2014)

**Padding Oracle On Downgraded Legacy Encryption.** Affected SSLv3 with CBC. An attacker who could force a downgrade to SSLv3 could decrypt cookies one byte at a time using a padding oracle.

Fix: disable SSLv3. **Use TLS 1.3** (no SSLv3 fallback).

### Logjam (2015)

Affected TLS 1.0/1.1/1.2 with DHE_EXPORT cipher suites that used 512-bit DH groups (a 1990s export-grade limitation). 512-bit DH is breakable in hours.

Fix: use 2048-bit+ DH groups, prefer ECDHE. **Use TLS 1.3** (no export-grade ciphers).

### FREAK (2015)

**Factoring RSA Export Keys.** Similar to Logjam but for RSA — servers were tricked into using 512-bit RSA export keys.

Fix: disable export-grade ciphers. **Use TLS 1.3** (no RSA key exchange).

### Lucky 13 (2013)

Padding-oracle timing attack against TLS 1.0/1.1/1.2 CBC modes. The MAC verification took slightly different amounts of time depending on padding correctness.

Fix: constant-time MAC verification. **Use TLS 1.3** (no CBC, AEAD only).

### ROBOT (2017)

**Return Of Bleichenbacher's Oracle Threat.** A re-discovery of Bleichenbacher's 1998 padding-oracle attack on RSA PKCS#1 v1.5 — many TLS implementations were still vulnerable nineteen years later.

Fix: constant-time RSA PKCS#1 handling, but really, **use TLS 1.3** (no RSA key exchange at all).

### Bleichenbacher (1998)

The original. Padding-oracle attack on RSA PKCS#1 v1.5 encryption. Took about a million queries to recover a session key.

Fix: same as ROBOT. **Use TLS 1.3.**

### Downgrade attacks (various years)

A class of attacks where an active attacker forces the negotiation to use a weaker protocol or cipher than both endpoints actually support. E.g., strip the ClientHello's TLS 1.3 hint so the server falls back to TLS 1.2.

Fix: TLS 1.3 includes a *downgrade canary* in `ServerHello.random` (the bytes "DOWNGRD" + version sentinel). A TLS 1.3 client that sees this canary aborts. **Use TLS 1.3.**

### SSL stripping (2009)

Not a TLS attack — an active attacker on the network rewrites HTTPS links to HTTP and sits in the middle as a plain-HTTP proxy. The client never tries TLS, so TLS can't help.

Fix: **HSTS** (HTTP Strict Transport Security). The first time you connect over HTTPS, the server sends a header saying "always use HTTPS for the next 2 years." The browser remembers this and refuses to downgrade. Even better: get on the **HSTS preload list** so the browser knows to use HTTPS even on the very first visit.

### The pattern

Notice how almost every fix is "use TLS 1.3." TLS 1.3 was designed in the wake of all these attacks. Static RSA gone. CBC gone. Compression gone. Renegotiation gone. Weak hashes gone. Export ciphers gone. Downgrade canary added. AEAD-only.

Use TLS 1.3.

## Hands-On

Time to actually run things. These commands all work on a normal Mac or Linux box with `openssl` and `curl` installed (which is to say, every machine made in the last decade).

### 1. The full handshake trace

```
$ openssl s_client -connect example.com:443 -servername example.com
CONNECTED(00000003)
depth=2 C = US, O = DigiCert Inc, OU = www.digicert.com, CN = DigiCert Global Root CA
verify return:1
depth=1 C = US, O = DigiCert Inc, CN = DigiCert Global G2 TLS RSA SHA256 2020 CA1
verify return:1
depth=0 C = US, ST = California, L = Los Angeles, O = "Internet Corporation for Assigned Names and Numbers", CN = www.example.org
verify return:1
---
Certificate chain
 0 s:C = US, ST = California, L = Los Angeles, O = "Internet Corporation for Assigned Names and Numbers", CN = www.example.org
   i:C = US, O = DigiCert Inc, CN = DigiCert Global G2 TLS RSA SHA256 2020 CA1
 1 s:C = US, O = DigiCert Inc, CN = DigiCert Global G2 TLS RSA SHA256 2020 CA1
   i:C = US, O = DigiCert Inc, OU = www.digicert.com, CN = DigiCert Global Root CA
---
SSL handshake has read 5854 bytes and written 384 bytes
Verification: OK
---
New, TLSv1.3, Cipher is TLS_AES_256_GCM_SHA384
Server public key is 2048 bit
Secure Renegotiation IS NOT supported
Compression: NONE
Expansion: NONE
No ALPN negotiated
Early data was not sent
Verify return code: 0 (ok)
---
```

That's a complete TLS 1.3 handshake. Note `Cipher is TLS_AES_256_GCM_SHA384` and `Verification: OK`. After that line, you can type HTTP requests and the connection will carry them encrypted.

Press `Ctrl-D` to close.

### 2. Show the full cert chain

```
$ openssl s_client -connect example.com:443 -servername example.com -showcerts
... (same as above plus)
Certificate chain
 0 s:CN = www.example.org
   -----BEGIN CERTIFICATE-----
   MIIH...
   -----END CERTIFICATE-----
 1 s:CN = DigiCert Global G2 TLS RSA SHA256 2020 CA1
   -----BEGIN CERTIFICATE-----
   MIIE...
   -----END CERTIFICATE-----
```

Now you have the actual cert PEMs. You could pipe them to `openssl x509 -text -noout` to decode each one.

### 3. STARTTLS (SMTP)

Many protocols don't start with TLS but upgrade to TLS via a STARTTLS command. Email is the most common.

```
$ openssl s_client -connect smtp.gmail.com:587 -starttls smtp
CONNECTED(00000003)
depth=2 OU = GlobalSign Root CA - R2, O = GlobalSign, CN = GlobalSign
verify return:1
...
Verify return code: 0 (ok)
---
220 smtp.gmail.com ESMTP
```

Other supported `-starttls` values: `imap`, `pop3`, `ftp`, `xmpp`, `lmtp`, `nntp`, `irc`, `postgres`, `mysql`.

### 4. Decode a certificate (full text)

```
$ openssl x509 -in cert.pem -text -noout
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number:
            03:f4:a2:b8:9e:5b:7d:6e:1f:84:5c:8a:9d:24
        Signature Algorithm: ecdsa-with-SHA384
        Issuer: C = US, O = Let's Encrypt, CN = E5
        Validity
            Not Before: Feb 14 00:00:00 2026 GMT
            Not After : May 14 23:59:59 2026 GMT
        Subject: CN = bellis.tech
        Subject Public Key Info:
            Public Key Algorithm: id-ecPublicKey
                Public-Key: (256 bit)
                pub:
                    04:1b:6e:7d:...
                ASN1 OID: prime256v1
                NIST CURVE: P-256
        X509v3 extensions:
            X509v3 Subject Alternative Name:
                DNS:bellis.tech, DNS:www.bellis.tech
            X509v3 Key Usage: critical
                Digital Signature
            X509v3 Extended Key Usage:
                TLS Web Server Authentication, TLS Web Client Authentication
            ...
```

Everything inside a cert: serial, issuer, validity, subject, public key, SAN, key usage, extensions.

### 5. Just the dates

```
$ openssl x509 -in cert.pem -dates -noout
notBefore=Feb 14 00:00:00 2026 GMT
notAfter=May 14 23:59:59 2026 GMT
```

Useful for monitoring scripts. Pipe to `date -d` to compute days remaining.

### 6. Fingerprint

```
$ openssl x509 -in cert.pem -fingerprint -sha256 -noout
sha256 Fingerprint=A1:B2:C3:D4:E5:F6:78:90:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88
```

Used for cert pinning, comparison, audit.

### 7. Verify a cert against a chain

```
$ openssl verify -CAfile chain.pem cert.pem
cert.pem: OK
```

If anything is wrong:

```
$ openssl verify -CAfile chain.pem cert.pem
cert.pem: O = Let's Encrypt, CN = E5
error 20 at 0 depth lookup: unable to get local issuer certificate
```

### 8. Handshake speed

```
$ openssl s_time -connect example.com:443 -new
Collecting connection statistics for 30 seconds
**********
**********
85 connections in 0.32s; 265.62 connections/user sec, bytes read 0
85 connections in 30 real seconds, 0 bytes read per connection
```

Useful for capacity planning. Each connection here did a fresh full handshake.

### 9. curl with TLS info

```
$ curl -v https://example.com 2>&1 | grep -E "subject|issuer|TLS|cipher"
* SSL connection using TLSv1.3 / TLS_AES_256_GCM_SHA384 / x25519 / RSASSA-PSS
* Server certificate:
*  subject: C=US; ST=California; L=Los Angeles; O=...; CN=www.example.org
*  issuer: C=US; O=DigiCert Inc; CN=DigiCert Global G2 TLS RSA SHA256 2020 CA1
```

`curl -v` prints the negotiated protocol, cipher, key exchange group, and signature algorithm. `grep` plus a pattern shows you just the bits you want.

### 10. OCSP stapling check

```
$ curl --cert-status https://example.com -o /dev/null -s -w "%{http_code}\n"
200
```

`--cert-status` requires the server to have stapled an OCSP response. If it didn't, curl fails.

### 11. Force TLS version

```
$ curl --tls-max 1.2 https://example.com -o /dev/null -s -w "%{http_version} %{ssl_verify_result}\n"
2 0
```

`--tls-max 1.2` forces TLS 1.2 max. Useful for testing what versions a server supports. Use `--tlsv1.3` to require TLS 1.3 minimum.

### 12. nmap cipher enumeration

```
$ nmap --script ssl-enum-ciphers -p 443 example.com
PORT    STATE SERVICE
443/tcp open  https
| ssl-enum-ciphers:
|   TLSv1.2:
|     ciphers:
|       TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 (rsa 2048) - A
|       TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256 (rsa 2048) - A
|     compressors:
|       NULL
|     cipher preference: server
|   TLSv1.3:
|     ciphers:
|       TLS_AES_256_GCM_SHA384 (ecdh_x25519) - A
|       TLS_CHACHA20_POLY1305_SHA256 (ecdh_x25519) - A
|       TLS_AES_128_GCM_SHA256 (ecdh_x25519) - A
|     cipher preference: server
|_  least strength: A
```

Shows you exactly what protocols and ciphers a server supports, with strength grades.

### 13. testssl.sh

`testssl.sh` is a comprehensive third-party test tool. It checks everything: protocols, ciphers, certificate, HSTS, OCSP, vulnerability checks for every major attack.

```
$ testssl.sh https://example.com
Start 2026-04-27 ...
Testing protocols via sockets except NPN+ALPN
 SSLv2     not offered (OK)
 SSLv3     not offered (OK)
 TLS 1     not offered
 TLS 1.1   not offered
 TLS 1.2   offered (OK)
 TLS 1.3   offered (OK): final
... (hundreds of lines)
```

Install with `brew install testssl` or `apt install testssl.sh`.

### 14. gnutls-cli

GnuTLS's equivalent of `openssl s_client`. Slightly different output, sometimes useful for diagnosing OpenSSL bugs.

```
$ gnutls-cli --strict-tofu example.com:443
Processed 132 CA certificate(s).
Resolving 'example.com:443'...
Connecting to '93.184.216.34:443'...
- Certificate type: X.509
- Got a certificate list of 2 certificates.
- Certificate[0] info:
 - subject `CN=www.example.org,...
- Status: The certificate is trusted.
- Description: (TLS1.3-X.25519)-(ECDHE-SECP256R1)-(ECDSA-SECP256R1-SHA256)-(AES-256-GCM)
- Session ID: ...
- Handshake was completed
- Simple Client Mode:
```

### 15. List ciphers TLS 1.3 supports

```
$ openssl ciphers -v 'TLSv1.3'
TLS_AES_256_GCM_SHA384         TLSv1.3 Kx=any      Au=any  Enc=AESGCM(256)            Mac=AEAD
TLS_CHACHA20_POLY1305_SHA256   TLSv1.3 Kx=any      Au=any  Enc=CHACHA20/POLY1305(256) Mac=AEAD
TLS_AES_128_GCM_SHA256         TLSv1.3 Kx=any      Au=any  Enc=AESGCM(128)            Mac=AEAD
```

### 16. List ECDHE+AESGCM TLS 1.2 ciphers

```
$ openssl ciphers -v 'ECDHE+AESGCM'
ECDHE-ECDSA-AES256-GCM-SHA384  TLSv1.2 Kx=ECDH     Au=ECDSA Enc=AESGCM(256)            Mac=AEAD
ECDHE-RSA-AES256-GCM-SHA384    TLSv1.2 Kx=ECDH     Au=RSA   Enc=AESGCM(256)            Mac=AEAD
ECDHE-ECDSA-AES128-GCM-SHA256  TLSv1.2 Kx=ECDH     Au=ECDSA Enc=AESGCM(128)            Mac=AEAD
ECDHE-RSA-AES128-GCM-SHA256    TLSv1.2 Kx=ECDH     Au=RSA   Enc=AESGCM(128)            Mac=AEAD
```

### 17. Force a specific group

```
$ openssl s_client -connect example.com:443 -tls1_3 -groups X25519
```

This restricts the key_share groups offered to just X25519. Useful for testing group support.

### 18. Generate an RSA private key + CSR

```
$ openssl req -newkey rsa:4096 -keyout key.pem -out csr.pem
Generating a RSA private key
....+++++
.+++++
writing new private key to 'key.pem'
Enter PEM pass phrase:
Verifying - Enter PEM pass phrase:
-----
You are about to be asked to enter information that will be incorporated
into your certificate request.
...
Country Name (2 letter code) [AU]:US
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:Bellis Tech
Common Name (e.g. server FQDN or YOUR name) []:bellis.tech
```

Produces a 4096-bit RSA key and a CSR (Certificate Signing Request). Send the CSR to a CA.

### 19. Decode a CSR

```
$ openssl req -in csr.pem -text -noout
Certificate Request:
    Data:
        Version: 1 (0x0)
        Subject: C = US, O = Bellis Tech, CN = bellis.tech
        Subject Public Key Info:
            Public Key Algorithm: rsaEncryption
                RSA Public-Key: (4096 bit)
                Modulus:
                    00:c4:...
                Exponent: 65537 (0x10001)
        Attributes:
            (none)
    Signature Algorithm: sha256WithRSAEncryption
         8a:1b:...
```

### 20. Generate an Ed25519 key

```
$ openssl genpkey -algorithm ED25519 -out ed25519.key
$ openssl pkey -in ed25519.key -pubout -text
Public-Key: (256 bit)
pub:
    8b:b3:...
```

Ed25519 keys are tiny (32 bytes for the public key) and very fast. Used for SSH and increasingly for code signing, but not yet widely supported by web CAs.

### 21. Show OCSP URI

```
$ openssl x509 -in cert.pem -ocsp_uri -noout
http://r3.o.lencr.org
```

This is the URL the cert says to query for OCSP status.

### 22. Show SAN list

```
$ openssl x509 -in cert.pem -ext subjectAltName -noout
X509v3 Subject Alternative Name:
    DNS:bellis.tech, DNS:www.bellis.tech, DNS:api.bellis.tech
```

The list of names this cert is valid for.

### 23. Generate a CRL

```
$ openssl ca -gencrl -keyfile ca.key -cert ca.crt -out crl.pem
Using configuration from /etc/ssl/openssl.cnf
$ openssl crl -in crl.pem -text -noout
Certificate Revocation List (CRL):
        Version 2 (0x1)
        Signature Algorithm: sha256WithRSAEncryption
        Issuer: CN = My CA
        Last Update: Apr 27 14:00:00 2026 GMT
        Next Update: May 27 14:00:00 2026 GMT
...
```

Useful for running your own internal CA.

### 24. Just the dates from a remote cert (one-liner)

```
$ openssl s_client -connect example.com:443 -servername example.com < /dev/null 2>/dev/null | openssl x509 -noout -dates
notBefore=Jan 30 00:00:00 2026 GMT
notAfter=Mar  1 23:59:59 2027 GMT
```

Drop this into a monitoring script. If `notAfter` is less than 14 days away, alert.

### 25. step certificate inspect

`step` from Smallstep is a modern alternative to openssl, with much friendlier output.

```
$ step certificate inspect https://example.com
Certificate:
    Data:
        Version: 3 (0x2)
        Serial Number: ...
        Signature Algorithm: ECDSA-SHA384
        Issuer: C=US,O=DigiCert Inc,CN=DigiCert Global G2 TLS RSA SHA256 2020 CA1
        Validity:
            Not Before: 2026-01-30T00:00:00Z
            Not After : 2027-03-01T23:59:59Z
        Subject: C=US,ST=California,L=Los Angeles,O=Internet Corporation for Assigned Names and Numbers,CN=www.example.org
...
```

Install: `brew install step` or download from smallstep.com.

### 26. Generate an EC private key (P-256)

```
$ openssl ecparam -name prime256v1 -genkey -noout -out ec.key
$ openssl ec -in ec.key -text -noout
read EC key
Private-Key: (256 bit)
priv:
    8a:b2:...
pub:
    04:1b:6e:7d:...
ASN1 OID: prime256v1
NIST CURVE: P-256
```

### 27. Make a self-signed cert in one step

```
$ openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
    -keyout self.key -out self.crt -days 365 -nodes \
    -subj '/CN=localhost'
```

Now you have `self.key` and `self.crt`. Browsers will not trust them but you can use them for local development.

### 28. Inspect a cert from stdin

```
$ openssl s_client -connect example.com:443 -servername example.com < /dev/null 2>/dev/null \
    | openssl x509 -text -noout | head -30
```

Pipe the handshake output into `x509` to decode whatever cert came back.

## Common Confusions

These are the questions everyone asks. Each one is "broken understanding" then "actual answer."

### "Is HTTPS the same as TLS?"

**Broken:** "HTTPS = TLS, right?"

**Fixed:** No. HTTPS is HTTP-over-TLS. TLS is the lock-and-envelope system that wraps things. HTTP is the language inside the envelope. You can use TLS to wrap *other* protocols too — IMAP-over-TLS, SMTP-over-TLS, SIP-over-TLS, your own custom binary protocol-over-TLS. HTTPS is just the most famous one.

### "Is SSL still a thing?"

**Broken:** "I want to enable SSL on my site."

**Fixed:** SSL is the old name. The protocol that today is called TLS used to be called SSL. SSLv2 (1995), SSLv3 (1996), then it got renamed to TLS in 1999. So TLS 1.0 was effectively SSLv3.1. Today people often say "SSL" when they mean "TLS" — old habit. The actual SSL versions (1, 2, 3) have been deprecated and disabled in all browsers for over a decade. If a vendor's docs say "SSL" in 2026 they almost certainly mean TLS.

### "Why does my self-signed cert fail?"

**Broken:** "I made a cert with openssl. Why does the browser show a warning?"

**Fixed:** A self-signed cert has no chain back to a CA in the trust store. The browser has no way to know whether the cert is genuine. The defense is correct — refusing to trust a stranger's cert is the right behavior. For dev work either install your local CA into the trust store or use `mkcert`, which creates a local CA, installs it, and signs dev certs with it.

### "Why doesn't TLS 1.0 work anymore?"

**Broken:** "My old client uses TLS 1.0. Can the server enable that?"

**Fixed:** TLS 1.0 (1999) and TLS 1.1 (2006) were both deprecated by RFC 8996 in 2021. All major browsers, OSes, and tools have removed them. The reason: they're vulnerable to BEAST, POODLE, Lucky 13, and other attacks. The fix is to update the old client to support TLS 1.2 or 1.3. There is no good reason to keep 1.0 alive in 2026.

### "Why is curl -k bad?"

**Broken:** "I'm getting cert errors. I'll just curl with -k and move on."

**Fixed:** `curl -k` (or `--insecure`) tells curl to skip *all* certificate validation — chain check, expiry, hostname match, everything. It is functionally equivalent to using HTTP with extra steps. Anyone in the middle can intercept. Use it for one-off testing if you must, but never in scripts, never in production, never in CI. If a script needs `-k` to work, the script is wrong; fix the cert configuration instead.

### "Forward secrecy means I can't decrypt logs anymore?"

**Broken:** "We used to be able to decrypt traffic captures with the server's private key."

**Fixed:** Yes. That's the whole point. With ECDHE, the long-term private key is not used for the key exchange — it's only used for signing. The actual session keys are derived from ephemeral DH and exist only in the running process's memory. Even with the server's private key in hand, you cannot decrypt past sessions. If you need to decrypt for debugging, set up `SSLKEYLOGFILE` *before* the session and Wireshark can use it.

### "Why do I need OCSP stapling?"

**Broken:** "OCSP just works, why bother stapling?"

**Fixed:** Without stapling, when a client visits your site, the *client* has to query the CA's OCSP responder to check if your cert is revoked. This (a) leaks to the CA which sites everyone is visiting (privacy), (b) introduces latency on the first connection, (c) creates a single point of failure (if the CA's OCSP responder is down, certs can't be checked). With stapling, *the server* fetches the OCSP response periodically and includes it in the handshake. No client query, no privacy leak, no latency. Always enable stapling.

### "Why do certs only last 90 days now?"

**Broken:** "It's such a hassle to renew certs every 90 days."

**Fixed:** Short-lived certs are a security improvement. If a cert (or its private key) is compromised, the damage window is limited. CRL/OCSP revocation is unreliable in practice (clients often don't check). A 90-day cert that gets compromised expires in <90 days regardless. Combined with auto-renewal (certbot, ACME), renewal is no work at all. The CA/Browser Forum has been gradually shortening the maximum cert lifetime: 5 years, 3 years, 2 years, 13 months, and the proposed 90 days for all certs by 2027.

### "Can I just use one wildcard cert for everything?"

**Broken:** "I'll use *.bellis.tech for all my services."

**Fixed:** You can, and it's convenient, but be aware:
1. Wildcards only cover one level. `*.bellis.tech` covers `api.bellis.tech` but NOT `v1.api.bellis.tech`.
2. Wildcards do NOT cover the apex (`bellis.tech`) — you need a separate entry for that.
3. If the wildcard private key is compromised, ALL services are affected. Per-service certs limit blast radius.
4. CAs have started requiring DNS-01 validation for wildcards (you can't use HTTP-01).

### "What's HSTS?"

**Broken:** "HSTS sounds like more cert stuff."

**Fixed:** HSTS (HTTP Strict Transport Security) is a header the server sends saying "for the next N seconds, always use HTTPS for this domain — never let the user click through a cert warning, never accept a downgrade to HTTP." It defends against SSL stripping. Once a browser has seen the header, it remembers it and refuses HTTP. Get on the **HSTS preload list** to make this work even on the very first visit.

### "What's the difference between a cert and a key?"

**Broken:** "Aren't they the same thing?"

**Fixed:** No.
- The **private key** is a secret number. Only the server has it. Never send it. Never put it in a git repo. Never email it.
- The **certificate** is a file containing the *public* key plus signatures from CAs. It is meant to be sent to clients in every handshake. It is public; it is not a secret.
The cert is the public part. The key is the private part. They come as a pair: a cert is "useful" only with the corresponding key, and vice versa.

### "What's mTLS and when do I need it?"

**Broken:** "Standard TLS is enough for everything, right?"

**Fixed:** Standard TLS authenticates the server to the client. Mutual TLS (mTLS) also authenticates the *client* to the server, using a client certificate. It's the right answer for service-to-service auth in microservices, API-to-API auth in zero-trust networks, and any case where you want strong identity for both sides. In a public-internet web app, mTLS is usually overkill (most sites use passwords/OAuth instead). In a private internal network, mTLS is the gold standard.

### "Why does my cert work in my browser but not in curl?"

**Broken:** "Browser is happy. curl says cert error."

**Fixed:** Different trust stores. Browsers use the OS trust store (macOS Keychain, Windows Cert Store, Mozilla NSS). curl on Linux uses `/etc/ssl/certs/ca-certificates.crt` (or `/etc/ssl/cert.pem` on Mac). curl on macOS sometimes uses Keychain, sometimes not, depending on how it was built. Either install your CA to the right place, or use `--cacert /path/to/your-ca.pem` to point curl at a specific bundle.

### "Why does my cert work over IPv4 but not IPv6?"

**Broken:** "It works on `curl 1.2.3.4` but fails on `curl [::1]`!"

**Fixed:** The cert's SAN list might not include the IPv6 address. Cert SANs include DNS names and IP addresses. If your cert is for `bellis.tech` only, then it works whenever DNS resolves the hostname. If you're connecting by raw IP (which you usually shouldn't), the SAN must include that IP. For IP-based access, include both v4 and v6 in the SAN.

## Vocabulary

You will see these terms over and over. Each one is one line of plain English.

| Term                          | Plain-English meaning                                                                                  |
| ----------------------------- | ------------------------------------------------------------------------------------------------------ |
| HTTP                          | The plain-text web protocol. Postcards.                                                                |
| HTTPS                         | HTTP wrapped in TLS. Sealed envelopes.                                                                 |
| TLS                           | Transport Layer Security. The lock-and-envelope system itself.                                         |
| SSL                           | The old name for TLS. SSLv2 and v3 are dead. "SSL" today usually means TLS.                            |
| Symmetric                     | Same key locks and unlocks. Decoder-ring style. Fast.                                                  |
| Asymmetric                    | Two keys: one locks (public), one unlocks (private). Magic-mailbox style. Slow.                        |
| Public key                    | The half of a key pair you share with everyone.                                                        |
| Private key                   | The half of a key pair you keep secret. Loss = total compromise.                                       |
| Key exchange                  | How two strangers agree on a shared secret over a public channel.                                      |
| ECDHE                         | Elliptic Curve Diffie-Hellman Ephemeral. Modern key exchange. Fresh keys each session.                 |
| RSA                           | Old asymmetric algorithm based on factoring large numbers. Slow. Used for signatures.                  |
| X25519                        | Modern elliptic curve for key exchange. Designed by Bernstein. Fast. Constant-time. TLS 1.3 default.   |
| P-256                         | NIST elliptic curve. ~128-bit security. Wide hardware support.                                          |
| P-384                         | NIST elliptic curve. ~192-bit security. Required by US gov CNSA 1.0.                                   |
| AES                           | Advanced Encryption Standard. Symmetric block cipher. The workhorse.                                   |
| AES-GCM                       | AES in Galois/Counter Mode. AEAD: encrypts + authenticates in one shot.                                |
| ChaCha20-Poly1305             | Stream cipher + MAC. Designed by Bernstein. Fast in software.                                          |
| AEAD                          | Authenticated Encryption with Associated Data. Encrypts AND detects tampering in one operation.        |
| MAC                           | Message Authentication Code. Tag that proves a message wasn't changed.                                 |
| HMAC                          | MAC built from a hash function (e.g., HMAC-SHA-256).                                                   |
| Hash                          | A one-way function. SHA-256, SHA-384. Used everywhere in TLS for integrity.                            |
| SHA-256                       | A 256-bit hash function. Universal default.                                                            |
| SHA-384                       | A 384-bit hash function. Used with stronger ciphers.                                                   |
| SHA-3                         | Newer hash, sponge construction. Not yet widely used in TLS.                                           |
| BLAKE2                        | Modern hash function. Faster than SHA-2. Not in TLS yet.                                               |
| Certificate                   | The server's ID badge. Public key + identity + CA signature.                                           |
| CA                            | Certificate Authority. The DMV that issues cert badges.                                                |
| Root CA                       | A self-signed CA at the top of the chain. Lives in your trust store.                                   |
| Intermediate                  | A CA in the middle. Signed by root, used for everyday signing.                                         |
| Leaf cert                     | The end-entity cert. The one your website actually presents.                                           |
| Chain                         | The list of certs from leaf to (but not including) root.                                               |
| SAN                           | Subject Alternative Name. The list of domains a cert is valid for.                                     |
| CN                            | Common Name. Old way to put the domain in a cert. Deprecated. Use SAN.                                 |
| CSR                           | Certificate Signing Request. The thing you send the CA to ask for a cert.                              |
| ACME                          | Automated Certificate Management Environment. RFC 8555. The Let's Encrypt protocol.                    |
| Let's Encrypt                 | Free automated CA, run by ISRG. 4+ billion certs issued.                                               |
| OCSP                          | Online Certificate Status Protocol. Real-time revocation check.                                        |
| CRL                           | Certificate Revocation List. CA's list of revoked cert serials.                                        |
| OCSP stapling                 | Server fetches OCSP response and attaches to handshake. Fast and private.                              |
| Must-staple                   | Cert flag saying "this cert MUST be served with stapled OCSP."                                         |
| Certificate transparency (CT) | Public append-only logs of all issued certs. RFC 6962.                                                 |
| SCT                           | Signed Certificate Timestamp. Proof a cert was logged to a CT log.                                     |
| HSTS                          | HTTP Strict Transport Security. Header that says "always use HTTPS."                                   |
| HPKP                          | HTTP Public Key Pinning. Deprecated. Was too easy to brick a site.                                     |
| CAA                           | Certification Authority Authorization. DNS record saying which CAs can issue for your domain.          |
| DANE                          | DNS-based Authentication of Named Entities. Cert pinning via DNSSEC.                                   |
| DNSSEC                        | DNS Security Extensions. Cryptographically signed DNS.                                                 |
| ESNI                          | Encrypted SNI. Old draft, replaced by ECH.                                                             |
| ECH                           | Encrypted Client Hello. Hides SNI from eavesdroppers.                                                  |
| SNI                           | Server Name Indication. Hostname the client wants to reach. In ClientHello.                            |
| ALPN                          | Application-Layer Protocol Negotiation. Picks h2/h3/http/1.1 in the handshake.                         |
| h2                            | HTTP/2. ALPN value.                                                                                     |
| h3                            | HTTP/3 (over QUIC). ALPN value.                                                                         |
| NPN                           | Next Protocol Negotiation. Predecessor of ALPN. Deprecated.                                            |
| Session ticket                | Encrypted blob the server gives the client to enable resumption.                                       |
| Session ID                    | TLS 1.2 way to enable resumption. Replaced by tickets.                                                 |
| Resumption                    | Skipping most of the handshake by reusing prior session state.                                         |
| 0-RTT                         | Zero round-trip-time. Sending data with the very first packet. Fast but replay-vulnerable.             |
| Early data                    | Same as 0-RTT.                                                                                          |
| Replay protection             | Detecting and rejecting duplicated traffic.                                                            |
| Downgrade protection          | Defending against an attacker forcing weaker parameters.                                               |
| ChangeCipherSpec              | TLS 1.2 message that flipped the cipher switch. Vestigial in TLS 1.3.                                  |
| Encrypted Extensions          | TLS 1.3 message: server's encrypted negotiated parameters.                                             |
| Finished                      | Handshake message that proves transcript integrity. Both sides send it.                                |
| ClientHello                   | First handshake message: client's offer.                                                               |
| ServerHello                   | Second handshake message: server's choice.                                                             |
| HelloRetryRequest             | Server's "I want a different group" reply to ClientHello. Adds a round trip.                           |
| Key schedule                  | The HKDF cascade that derives all the keys in TLS 1.3.                                                 |
| HKDF                          | HMAC-based Key Derivation Function. RFC 5869.                                                          |
| PRF                           | Pseudo-random function. The thing key schedules are built from.                                        |
| Master secret                 | TLS 1.2 root key. In TLS 1.3 there's a similar Master Secret in the schedule.                          |
| Traffic secret                | TLS 1.3 secret used to derive symmetric keys for application data.                                     |
| Exporter                      | Mechanism to derive an external secret from a TLS session, for protocols that bind to TLS.             |
| Application data              | The actual HTTP/whatever bytes after the handshake. Encrypted with traffic keys.                       |
| Record layer                  | The byte-level framing inside TLS. Each record has a 5-byte header.                                    |
| Fragmentation                 | TLS records max out at 16 KB; longer plaintexts are split.                                             |
| Padding                       | Extra bytes added to hide plaintext length. Optional in TLS 1.3.                                       |
| MAC-then-encrypt              | Old order: compute MAC, encrypt MAC + plaintext. Lucky-13 vulnerable. Don't use.                       |
| Encrypt-then-MAC              | Better order: encrypt, then compute MAC over ciphertext. AEAD does this implicitly.                    |
| Post-quantum                  | Crypto designed to resist future quantum-computer attacks.                                             |
| Kyber / ML-KEM                | Lattice-based KEM. NIST FIPS 203. The leading post-quantum key exchange.                               |
| X25519+Kyber hybrid           | Combine classical X25519 with PQ Kyber for both classical and post-quantum security.                   |
| Cipher suite                  | A bundle of (symmetric algorithm, hash) (TLS 1.3) or (KX, auth, sym, hash) (TLS 1.2).                  |
| Forward secrecy               | Property that compromise of long-term key does not expose past session keys.                           |
| Perfect forward secrecy       | Same as forward secrecy. The "perfect" is mostly marketing.                                            |
| mTLS                          | Mutual TLS. Both sides present certs.                                                                  |
| MITM                          | Man-in-the-middle. Active attacker on the wire.                                                        |
| Padding oracle                | Bug class where an attacker learns plaintext from how decryption fails.                                |
| Constant-time                 | Code that runs in the same time regardless of secret inputs. Defends against timing side channels.     |
| KEM                           | Key Encapsulation Mechanism. The general form of "encrypt to a public key."                            |
| Trust store                   | Your computer's list of trusted root CAs.                                                              |
| TOFU                          | Trust On First Use. SSH-style: trust the key the first time you see it.                                |
| Pinning                       | Hard-coding which key/cert is acceptable. Defends against rogue CAs but brittle.                       |
| BPF                           | Berkeley Packet Filter, used by Wireshark to capture TLS traffic.                                      |
| nonce                         | Number used once. Each AEAD encryption needs a fresh one.                                              |

That's well over eighty terms. If you don't see one you need, it's probably in `cs security tls` (the dense reference).

## Try This

Pick any of these and actually do them. Don't just read.

### Experiment 1: Watch a real handshake with tcpdump and Wireshark

```
$ sudo tcpdump -i any -nn 'port 443' -w /tmp/tls.pcap
```

In another terminal:

```
$ curl https://example.com -o /dev/null
```

Stop the tcpdump (Ctrl-C). Open the pcap in Wireshark:

```
$ wireshark /tmp/tls.pcap
```

Filter for `tls`. You will see ClientHello, ServerHello, Certificate, Finished. Click each one and walk through the bytes.

### Experiment 2: Negotiated cipher and version

```
$ echo Q | openssl s_client -tls1_3 -connect cloudflare.com:443 2>&1 \
    | grep -E "Cipher|Server Temp Key|Verification|TLSv"
New, TLSv1.3, Cipher is TLS_AES_256_GCM_SHA384
Server Temp Key: X25519, 253 bits
Verification: OK
```

You just confirmed: TLS 1.3, AES-256-GCM, X25519 key exchange, valid cert chain.

### Experiment 3: Decode every cert your browser trusts

```
$ awk -v cmd='openssl x509 -noout -subject' '
    /BEGIN CERTIFICATE/ { close(cmd) }
    { print | cmd }' < /etc/ssl/cert.pem | head -20
```

Lists every root CA your system trusts.

### Experiment 4: Watch certs renew

If you run Let's Encrypt:

```
$ sudo journalctl -u certbot.timer --since "30 days ago" | tail -50
$ sudo certbot renew --dry-run
```

You can watch the renewal cycle in the journal. Expect attempts every ~12 hours, actual renewals when expiry < 30 days.

### Experiment 5: Introduce a deliberate cert error

Generate a self-signed cert and serve a tiny HTTPS server with it. Then connect with curl normally and watch it fail:

```
$ openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:prime256v1 \
    -keyout self.key -out self.crt -days 1 -nodes -subj '/CN=localhost'
$ python3 -c "
import http.server, ssl
ctx = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
ctx.load_cert_chain('self.crt', 'self.key')
srv = http.server.HTTPServer(('127.0.0.1', 8443), http.server.SimpleHTTPRequestHandler)
srv.socket = ctx.wrap_socket(srv.socket, server_side=True)
srv.serve_forever()
" &
$ curl https://localhost:8443/
curl: (60) SSL certificate problem: self-signed certificate
$ curl --cacert self.crt https://localhost:8443/
<!DOCTYPE html ...
```

You just lived through the "untrusted CA" error and the fix.

### Experiment 6: Find a host with weak crypto

Scan a server you control with testssl.sh and read the report:

```
$ testssl.sh https://your-test-server.example > report.txt
$ grep -E 'WARN|VULN|NOT ok' report.txt
```

Anything red is a finding. Most public hosts are clean. Old hosts have surprises.

### Experiment 7: Capture and dump session keys

Set the env var, run curl, open the pcap with the keylog file:

```
$ export SSLKEYLOGFILE=/tmp/keys.log
$ sudo tcpdump -i any -nn 'port 443' -w /tmp/tls.pcap &
$ curl https://example.com -o /dev/null
$ kill %1
$ wireshark -o "tls.keylog_file:$SSLKEYLOGFILE" /tmp/tls.pcap
```

Wireshark now shows the *decrypted* application data.

### Experiment 8: Enumerate all SANs of a real cert

```
$ openssl s_client -connect www.cloudflare.com:443 -servername www.cloudflare.com < /dev/null 2>/dev/null \
    | openssl x509 -ext subjectAltName -noout
```

Cloudflare's certs have hundreds of SANs because one cert covers many customers.

## Where to Go Next

- `cs security tls` — the dense reference: every flag, every RFC, every cipher, every gotcha
- `cs detail security/tls` — the math: ECDHE, AES-GCM internals, key schedule
- `cs troubleshooting tls-errors` — error → fix lookup table
- `cs networking http` — what HTTPS is wrapping
- `cs networking http2` — h2 over TLS
- `cs networking http3` — h3 over QUIC (with TLS 1.3 baked in)
- `cs ramp-up http3-quic-eli5` — the modern transport that bakes TLS in
- `cs security pki` — the CA / cert authority side
- `cs web-servers nginx` — server-side TLS configuration
- `cs web-servers caddy` — automatic HTTPS

## See Also

- `security/tls`
- `troubleshooting/tls-errors`
- `security/pki`
- `security/cryptography`
- `networking/http`
- `networking/http2`
- `networking/http3`
- `networking/quic`
- `ramp-up/linux-kernel-eli5`
- `ramp-up/http3-quic-eli5`
- `web-servers/nginx`
- `web-servers/caddy`

## References

- RFC 8446 — TLS 1.3 (the spec; required reading if you ever debug TLS)
- RFC 5246 — TLS 1.2 (legacy; needed when working with older clients)
- RFC 6066 — TLS Extensions (SNI, OCSP stapling, max fragment length)
- RFC 7301 — ALPN
- RFC 8879 — TLS Certificate Compression
- RFC 6962 — Certificate Transparency
- RFC 6960 — OCSP
- RFC 8954 — OCSP Nonce
- RFC 5869 — HKDF
- RFC 7748 — X25519 / X448
- RFC 8032 — Ed25519 / Ed448
- RFC 5116 — AEAD interface
- RFC 5280 — X.509 PKI profile
- RFC 8555 — ACME (Let's Encrypt's protocol)
- RFC 6797 — HSTS
- RFC 8470 — HTTP 425 Too Early (for 0-RTT)
- RFC 8996 — Deprecating TLS 1.0 and 1.1
- RFC 9001 — Using TLS to Secure QUIC
- man openssl(1)
- man openssl-s_client(1)
- man openssl-x509(1)
- man openssl-verify(1)
- man openssl-ciphers(1)
- man openssl-req(1)
- man openssl-genpkey(1)
- "Bulletproof TLS and PKI" by Ivan Ristić — definitive operator reference
- "Cryptography Engineering" by Ferguson, Schneier, Kohno — the why behind the algorithms
- "Serious Cryptography" by Jean-Philippe Aumasson — modern crypto from first principles
- testssl.sh — comprehensive third-party TLS scanner
- sslyze — fast Python-based TLS scanner
- sslscan — quick CLI scanner
- nmap ssl-enum-ciphers — built into nmap
- Cloudflare's TLS 1.3 explainer — `lynx https://blog.cloudflare.com/why-tls-1-3-is-so-amazing/`
- The IETF TLS WG mailing list archives — definitive history of every design decision

> The S in HTTPS is the seal on the envelope. TLS is how the seal is made. Everything else is detail. — and the detail is where the magic actually lives.
