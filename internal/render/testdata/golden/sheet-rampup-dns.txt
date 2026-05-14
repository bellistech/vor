# DNS — ELI5 (The Internet's Address Book)

> DNS is the internet's address book: you say a name like `google.com`, and DNS hands back the actual address of the building so your packets can go there.

## Prerequisites

(none — but `cs ramp-up ip-eli5` and `cs ramp-up udp-eli5` help)

You do not need to know what DNS stands for. You do not need to know how the internet works. You do not need to be able to read a packet capture, or know what an IP address looks like. By the end of this sheet you will know all of those things in plain English, and you will have typed real DNS commands and watched real names turn into real numbers.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

If you have not yet read the IP and UDP ELI5 sheets, that is okay. They will help your understanding be deeper. But this sheet is written so you can read it cold, with no prior knowledge.

## What Even Is DNS?

### The plain English version

The internet is a giant city. Every computer on it has a numeric street address called an **IP address.** It looks like `142.251.40.78` or `2607:f8b0:4005:80c::200e`. Computers love these numbers. Humans hate these numbers. Nobody wants to type `142.251.40.78` into their browser. Nobody wants to remember `74.125.224.72`. Nobody wants to send their grandma a text saying "click on `2606:4700:20::ac43:4747`, it's a really cute video of a puppy."

So we use names instead. `google.com`. `youtube.com`. `wikipedia.org`. `your-bank-please-don-t-leak-money.com`. Names are easy. Names are friendly. Names you can write on a billboard.

But the internet itself does not use names. The internet uses numbers. So somewhere, somehow, those nice friendly names have to be turned into the actual numeric addresses. **That conversion machine is DNS.**

DNS stands for **Domain Name System.** A domain name is the friendly name (`google.com`). The DNS is the system that turns it into an IP address. Boring acronym. Powerful idea.

Every single time your computer talks to anything on the internet by name, DNS happened. You opened your browser. DNS happened. You opened a chat app. DNS happened. You looked at the weather widget. DNS happened. You loaded this very page if it came from somewhere online. DNS happened. DNS is the silent, constant machinery in the background that keeps the internet feeling like it has names instead of numbers.

### The phone book picture

Imagine an enormous phone book — bigger than any phone book that has ever existed in physical paper. There are billions of entries. Each entry has a name on the left and a number on the right.

```
ENTRY                                      NUMBER
google.com                                 142.251.40.78
wikipedia.org                              198.35.26.96
news.bbc.co.uk                             212.58.244.10
your-bank.example                          203.0.113.42
my-favorite-cat-blog.example.org           198.51.100.7
...billions more...
```

When you say "I want to go to `google.com`," some helper opens the phone book, finds the line with `google.com` on it, reads the number on the right side, and hands it back to you.

That helper is the DNS resolver. The phone book is the DNS system. The line you read is called a **DNS record.**

Of course, no real phone book has billions of entries. So the phone book is split up. Different sections of the phone book are stored on different computers all around the world. The helper might have to walk to several different storage rooms to find the right section. That walk is what we call a **DNS lookup.**

### The postal-service-plus-GPS picture

Here is another way to think about it. Pretend the internet is a postal system. You want to send a letter to your friend. You write their name on the envelope: "John Smith." But the postman cannot deliver the letter just from a name. The postman needs an address: "123 Main Street, Springfield."

Most countries have a thing called a **directory service.** You look up the name "John Smith" and you get back an address "123 Main Street." Then you write the address on the envelope, and the postman can deliver it.

DNS is the directory service for the internet. Browsers, apps, and your phone all need real numeric addresses (IP addresses) to send packets. They start with names. They look up the names in DNS. DNS hands them addresses. They send packets.

### The ask-the-postman picture

Or imagine this. You walk up to a postman in a really big city. You say, "Hi, can you take me to John Smith's house?" The postman doesn't know everyone in the city. So the postman walks you over to an office that handles every house in the city. That office says, "John Smith? Hmm, that depends on which neighborhood. Let me check." The neighborhood office says, "John Smith on Main Street? Sure, his house is at 123 Main." Great, now you know.

That is roughly how a DNS lookup actually walks the tree. We will draw the actual walk in the next section.

### Why we needed DNS in the first place

Way back in the very early days of the internet, before there were billions of computers, there was a much simpler approach. Every single computer had a file called `hosts.txt`. It was just a list of every name and every number for the entire internet. Like, all of it. In one file. About once a day, a single computer at Stanford would update the master list, and every other computer in the world would download the latest version.

That worked when the entire internet was a few hundred computers. It stopped working when the internet got bigger. Imagine if every phone in the world had to download a complete copy of the entire global phone book once a day. That is not going to scale.

So in 1983, two people named **Paul Mockapetris** and **Jon Postel** designed DNS. The original spec is **RFC 882 and RFC 883.** Two years later it was updated to **RFC 1034 and RFC 1035** (in 1987), and those are still the foundational documents almost forty years later. Yes, the whole internet is still running on a paper from 1987. (We have added newer pieces since, but the core has not changed much. We will see those newer pieces.)

The genius idea: don't keep the whole phone book in one file. Split it up. Each section of the phone book is owned by whoever owns that part of the internet. Google owns `google.com`. Wikipedia owns `wikipedia.org`. The British government owns `gov.uk`. Each owner publishes their own section. When you need to look up a name, you ask the owner of that part of the phone book directly. The phone book is decentralized. Nobody has to download the whole thing.

That is the breakthrough that lets DNS scale to billions of names today.

### Why so many pictures?

You might wonder why we have a phone book picture, a postal-service picture, and a directory-service picture. Same reason as in the kernel sheet: nobody can see DNS. DNS is invisible. So we have to use pictures. Different pictures help with different ideas.

The **phone book** picture is best for understanding that DNS is a name → number lookup table.

The **postal service** picture is best for understanding that the internet uses numeric addresses, and DNS is the human-friendly name layer on top.

The **postman walking through neighborhoods** picture is best for understanding that the lookup actually traverses a tree.

If one picture is not clicking, switch to another. Whichever one feels right is the one you should keep in your head.

### What DNS is NOT

A common mistake is to think DNS is the thing that delivers your data. It is not. DNS only translates names into addresses. The actual delivery of bytes is done by a totally different layer (TCP, UDP, QUIC — all topics for other sheets).

DNS is just the address book. It does not carry the letters. It does not deliver the pizza. It just tells you where the pizza shop is. The pizza itself comes by a separate truck.

Another mistake: DNS is not the same as the internet. You can have an internet without DNS. It would be terrible (everyone would have to type IP addresses), but it would work. And there have been brief outages when DNS broke and the internet looked broken even though the network itself was fine. The internet works without DNS, but humans don't.

Another mistake: DNS is not just for web pages. Email uses DNS. Chat apps use DNS. Video calls use DNS. Game servers use DNS. Any time a computer talks to another computer by name, DNS is in the middle. The web is just one user of DNS. There are dozens.

## How a DNS Lookup Actually Works

### Two kinds of DNS server

Before we walk through the lookup, we have to learn two ideas. There are two flavors of DNS server in the world, and they do completely different jobs.

**A recursive resolver** is a DNS server that does the legwork for you. You ask it a question, it goes and finds the answer, and it hands you back the answer. It might have to talk to several other DNS servers along the way. You don't see that. You just see the question and the answer. The recursive resolver is like a personal assistant who runs all your errands.

**An authoritative server** is a DNS server that knows the answer for one specific section of the phone book. Google's authoritative servers know the answers for `google.com`. Wikipedia's authoritative servers know the answers for `wikipedia.org`. The root servers know the answers for "where to find the authoritative servers for `.com` and `.org` and so on." An authoritative server only answers about names it knows. If you ask it about a name it doesn't own, it says "not me, ask someone else."

Recursive resolver = personal assistant who fetches anything.
Authoritative server = the source of truth for one specific section.

A recursive resolver gets answers by asking authoritative servers. The authoritative servers never ask each other. They only answer for their own area.

### The walk

Here is what actually happens when you type `google.com` into your browser. We will use a fresh resolver with no cache, so we see the whole walk. (In real life, most lookups skip most of this because of caching. We will get to caching soon.)

```
        +---------+
        |   YOU   |  "where does google.com live?"
        +----+----+
             |
             v
        +---------+    "let me ask the resolver"
        |  STUB   |    (tiny piece of DNS in your operating system)
        | RESOLVER|
        +----+----+
             |
             | UDP port 53, "A google.com?"
             v
        +-----------+    "I'll find out for you"
        | RECURSIVE |
        | RESOLVER  |    (e.g., your ISP's DNS, or 1.1.1.1)
        +----+------+
             |
             | "where are the .com servers?"
             v
        +-----------+
        |  ROOT     |    a, b, c, d, ... 13 named servers
        |  SERVER   |    (.) the very top of the tree
        +----+------+
             |
             | "ask the .com TLD servers at these addresses..."
             v
        +-----------+
        |  TLD      |    .com, .org, .net, .io, .uk, ...
        |  SERVER   |    (.com)
        +----+------+
             |
             | "ask google's authoritative servers at these addresses..."
             v
        +-----------+
        | AUTHOR.   |
        |  SERVER   |    ns1.google.com, ns2.google.com, ...
        +----+------+
             |
             | "google.com lives at 142.251.40.78"
             v
        +-----------+
        | RECURSIVE |    "I have the answer. Here, take it."
        | RESOLVER  |    (caches it for next time)
        +----+------+
             |
             | "google.com is at 142.251.40.78"
             v
        +---------+
        |   YOU   |   browser opens TCP to 142.251.40.78
        +---------+
```

That is the full walk for `google.com` from cold. Let's go through each step in plain English.

**Step 1.** You type `google.com` into the address bar. The browser hands the name to your operating system: "I need the IP for `google.com`."

**Step 2.** Your operating system has a tiny DNS client built into it called a **stub resolver.** The stub resolver does almost nothing on its own. It just packages up the question into a DNS query and sends it to whatever DNS server is configured on your machine. (You can see what server that is by looking at `/etc/resolv.conf` or running `resolvectl status` — we will do that later.)

**Step 3.** The DNS server you send the query to is called a **recursive resolver.** This might be a server run by your ISP, or by your home router, or by a public service like 1.1.1.1 (Cloudflare) or 8.8.8.8 (Google). The recursive resolver does all the heavy lifting from here.

**Step 4.** The recursive resolver does not know the answer. It does know who to ask first: the **root servers.** There are 13 logical root servers named `a.root-servers.net` through `m.root-servers.net`. They are the very top of the DNS tree. Their addresses are baked in to every DNS server in the world (they almost never change). The resolver picks one and asks: "Who handles `.com`?"

**Step 5.** The root server doesn't know `google.com` either. It only knows about the top-level domains. It says: "I don't know about `google.com`, but the `.com` servers are at these addresses, ask them." This is called a **referral.**

**Step 6.** The resolver then asks the `.com` server: "Who handles `google.com`?" The `.com` server doesn't know the IP of `google.com`. It only knows which authoritative servers Google operates. It says: "Ask `ns1.google.com` or `ns2.google.com`. Here are their IP addresses." (The IPs are included so the resolver doesn't have to do another lookup just to find them. Those are called **glue records.**)

**Step 7.** The resolver finally asks `ns1.google.com`: "What is the address of `google.com`?" `ns1.google.com` is the authoritative server for the `google.com` zone. It knows. It looks up the answer in its zone file and replies: "142.251.40.78."

**Step 8.** The resolver gets the answer. It does two things at once: (1) It saves the answer in its **cache** so it doesn't have to do this whole walk next time. (2) It forwards the answer back to the stub resolver in your computer.

**Step 9.** Your stub resolver hands the answer to the browser. The browser now opens a TCP connection to `142.251.40.78`. The DNS part is over. Everything that happens next is a different protocol entirely.

The whole walk above might take 50 to 200 milliseconds the first time. After that, the recursive resolver has the answer in its cache, so the next person who asks gets the answer back in maybe 1 to 10 milliseconds. Caching is a huge deal in DNS. We will spend a whole section on it.

### What does a query actually look like?

A DNS query is a small UDP packet. The whole packet is usually under 100 bytes. The reply is usually under 200 bytes (more if it includes lots of records). Here is the rough shape:

```
+--------------------------+
| HEADER (12 bytes)        |  ID, flags, counts of each section
+--------------------------+
| QUESTION                 |  "A google.com?"
+--------------------------+
| ANSWER (only in reply)   |  "google.com IN A 142.251.40.78"
+--------------------------+
| AUTHORITY (sometimes)    |  who is authoritative for this name
+--------------------------+
| ADDITIONAL (sometimes)   |  glue records, EDNS0 stuff
+--------------------------+
```

The header is 12 bytes. Inside it are flags that tell us things like:

- **QR** — query (0) or response (1)?
- **AA** — authoritative answer? Set by the server when it owns the zone.
- **TC** — truncated? Set when the answer didn't fit in the packet.
- **RD** — recursion desired? You ask the resolver to recurse for you.
- **RA** — recursion available? The server tells you yes, it does recurse.
- **AD** — authenticated data? DNSSEC validation passed.
- **CD** — checking disabled? You're saying "don't bother validating DNSSEC for me."

The question section is just the name, the type (A, AAAA, MX, etc.), and the class (almost always IN, for "Internet").

The answer section, in a reply, contains zero or more **resource records (RRs)**. Each RR has the same five fields: name, type, class, TTL (time-to-live), and the actual data. We will learn the types in the next big section.

### Recursion vs. iteration

This is a confusing pair of words. Let me make it simple.

**Recursive query**: "Hey, please go figure out the answer for me, walk all the trees you need to walk, and only come back when you have the actual answer." Your computer talks to a recursive resolver. The resolver does all the work.

**Iterative query**: "Hey, just tell me what you know. If you don't know the answer, tell me the next server I should ask, and I'll go ask them myself." A recursive resolver talks to authoritative servers iteratively. Each authoritative server says "I know this much, ask the next person for the rest."

Picture:

```
RECURSIVE (you to your resolver):

   YOU ---ask--->  RESOLVER
                   "(goes off, does many iterative queries)"
                   "(eventually returns)"
   YOU <--answer-- RESOLVER  "google.com is 142.251.40.78"


ITERATIVE (resolver to authoritative servers):

   RESOLVER --ask--> ROOT
   RESOLVER <--here are .com servers-- ROOT
   RESOLVER --ask--> .COM
   RESOLVER <--here are google.com servers-- .COM
   RESOLVER --ask--> NS1.GOOGLE.COM
   RESOLVER <--google.com is 142.251.40.78-- NS1.GOOGLE.COM
```

So your laptop talks to a resolver in **recursive mode** (one question, one answer, the resolver does the work). The resolver talks to authoritative servers in **iterative mode** (many separate questions, each authoritative server hands off to the next).

If you want to see this in action, the `dig +trace google.com` command shows the iterative walk. We'll run that later.

## Record Types

A DNS record (an "RR" — resource record) is one line in the giant phone book. Every record has a type. The type tells you what kind of information the record holds. There are dozens of types, but most domains only use a handful. Here are the ones you will see all the time, plus a worked example for each.

### A — IPv4 address

The most basic record. It maps a name to an IPv4 address (a four-number address like `142.251.40.78`).

```
google.com.   300   IN   A   142.251.40.78
```

That line says: "The name `google.com` (in the IN class, with a TTL of 300 seconds) has an A record pointing to `142.251.40.78`." The browser uses this to know which IP to connect to.

A name can have multiple A records. That is how big sites give different users different servers (load balancing).

```
google.com.   300   IN   A   142.251.40.78
google.com.   300   IN   A   172.217.16.142
google.com.   300   IN   A   142.250.69.206
```

Three A records all under `google.com`. Resolvers usually return them in random order, so different users connect to different IPs. This is called **DNS round-robin.**

### AAAA — IPv6 address

Same idea as A, but for IPv6 addresses (the long colon-separated kind). Pronounced "quad-A."

```
google.com.   300   IN   AAAA   2607:f8b0:4005:80c::200e
```

Modern systems try AAAA first. If they get an IPv6 address, they use IPv6. Otherwise they fall back to A and IPv4. This is called **happy eyeballs** (RFC 8305): try both, use whichever responds first.

### CNAME — alias

A CNAME record says "this name is just an alias for some other name." The DNS resolver follows the chain.

```
www.google.com.   300   IN   CNAME   google.com.
```

If you ask for `www.google.com`, the resolver sees the CNAME, follows it, and now starts looking up `google.com` instead. Eventually it returns the A record for `google.com`. Your browser doesn't know any of this happened.

CNAMEs are very common. They let you point lots of names at one main service.

You cannot put a CNAME at the **apex** of a zone (the bare domain like `example.com` itself, with no subdomain). That is why some DNS providers offer fake "ALIAS" or "ANAME" records that work like CNAMEs at the apex. Those are vendor-specific tricks, not real DNS records.

### MX — mail exchanger

Tells you where to deliver email for a domain.

```
gmail.com.   3600   IN   MX   5 gmail-smtp-in.l.google.com.
gmail.com.   3600   IN   MX   10 alt1.gmail-smtp-in.l.google.com.
gmail.com.   3600   IN   MX   20 alt2.gmail-smtp-in.l.google.com.
```

The number before the name is the **priority** (lower = preferred). Mail servers try the lowest first, then fall back to higher numbers if the first one is unreachable. This is how email finds where to deliver.

### TXT — text

A TXT record is a freeform piece of text attached to a name. People use it for verification (proving they own a domain), for SPF (saying who is allowed to send email from this domain), for DKIM (signing keys for email), and DMARC (email security policy).

```
google.com.   300   IN   TXT   "v=spf1 include:_spf.google.com ~all"
```

That is an SPF record. It says "for email claiming to be from google.com, only servers listed in `_spf.google.com` are legitimate; otherwise mark as suspicious."

### PTR — reverse lookup

A PTR record maps an IP address back to a name. The opposite of an A record.

```
4.4.8.8.in-addr.arpa.   86400   IN   PTR   dns.google.
```

This says "the IP address `8.8.8.4` (note the address is reversed and put under `in-addr.arpa`) is named `dns.google`." Mail servers, log analyzers, and security tools use this to print human names instead of raw IPs.

### NS — name server

NS records say which servers are authoritative for a zone.

```
google.com.   86400   IN   NS   ns1.google.com.
google.com.   86400   IN   NS   ns2.google.com.
google.com.   86400   IN   NS   ns3.google.com.
google.com.   86400   IN   NS   ns4.google.com.
```

When the `.com` TLD server tells the resolver "ask Google's servers," these are the names it returns. Then the resolver has to look up the IPs of those NS names — except those IPs are already provided in the same response as **glue records**, so the resolver doesn't have to do another full lookup. (Without glue, resolving `google.com` would require resolving `ns1.google.com`, but resolving `ns1.google.com` requires asking `ns1.google.com`. Chicken and egg.)

### SOA — start of authority

Every zone has exactly one SOA record. It is the "title page" of the zone. It contains:

```
google.com.   3600   IN   SOA   ns1.google.com. dns-admin.google.com. (
                                  2024010101  ; serial
                                  900         ; refresh
                                  900         ; retry
                                  1800        ; expire
                                  60          ; negative TTL
                              )
```

- **Primary NS**: the main authoritative server for this zone.
- **Email**: the contact email (with `.` replacing `@`). Here `dns-admin@google.com`.
- **Serial**: a number that increments every time the zone changes. Secondary servers compare their serial to the primary's to know when to refresh.
- **Refresh**: how often secondaries should check for changes (seconds).
- **Retry**: if a refresh fails, how long to wait before trying again.
- **Expire**: how long secondaries should keep serving stale data if they cannot reach the primary.
- **Negative TTL**: how long resolvers should remember "this name does not exist" (NXDOMAIN).

You will not need to write SOA records by hand often. Just know what they are.

### SRV — service

SRV records map a service-name to a host. Used heavily by SIP, XMPP, Microsoft Active Directory, and Minecraft. The name is structured as `_service._proto.domain`.

```
_sip._tcp.example.com.   3600   IN   SRV   10 60 5060 sipserver.example.com.
```

That says: for the SIP service over TCP at example.com, use `sipserver.example.com` on port `5060`, with priority 10 and weight 60.

The four fields are: priority, weight, port, target.

### CAA — certificate authority authorization

CAA records tell certificate authorities which CAs are allowed to issue TLS certificates for your domain.

```
example.com.   3600   IN   CAA   0 issue "letsencrypt.org"
example.com.   3600   IN   CAA   0 issuewild "letsencrypt.org"
```

This says: only Let's Encrypt is allowed to issue certificates (including wildcards) for this domain. If a CA receives a request for a certificate but its name is not listed in CAA, it must refuse. This is a defense against rogue or compromised CAs issuing certificates for your domain.

### HTTPS / SVCB — service binding

A newer record type (RFC 9460, 2023). It tells clients about HTTPS service parameters: the protocols supported (HTTP/1.1, HTTP/2, HTTP/3), the IP hints, ALPN tags, ECH (encrypted client hello) keys, and so on. This lets a browser do "everything for connecting to this site" in one DNS lookup instead of several.

```
example.com.   3600   IN   HTTPS   1 . alpn="h3,h2" ipv4hint=192.0.2.1
```

You will start seeing these more as HTTP/3 and ECH roll out.

### DS, DNSKEY, RRSIG, NSEC, NSEC3 — DNSSEC records

These are the records that make DNSSEC work (signed DNS, see DNSSEC section below).

- **DNSKEY**: the public key for a zone. Lives in the zone itself.
- **DS**: a hash of the child zone's DNSKEY. Lives in the parent zone. This is the thing that links the parent's signature chain to the child.
- **RRSIG**: a signature over an RRset. There is one RRSIG for every signed RRset in the zone.
- **NSEC**: "the next name in this zone is X." Used for proving that something does NOT exist. Without NSEC, it is hard to prove a name doesn't exist in a signed way.
- **NSEC3**: same as NSEC but with hashed names, so an attacker cannot enumerate the entire zone by walking the NSEC chain.
- **NSEC3PARAM**: parameters for NSEC3 (hash algorithm, salt, iterations).
- **DLV**: an old, deprecated way to do DNSSEC validation. Don't use it.

We will visit these again in the DNSSEC section.

### Other record types you may see

- **TLSA** — TLS certificate associations (used by DANE, an alternative to CAs).
- **OPENPGPKEY** — PGP key for an email address.
- **SMIMEA** — S/MIME certificate for an email address.
- **SSHFP** — SSH host key fingerprints.
- **NAPTR** — used by SIP and ENUM for telephony.
- **LOC** — geographic location of a host (rarely used).
- **HINFO** — hardware/OS info (rarely used now, often returned by servers refusing ANY queries).
- **ANY** — a query type, not a record type. Asks for "all types." Most resolvers refuse.

## Caching and TTL

### Why caching exists

If every DNS query had to walk all the way from your computer to a root server to a TLD server to an authoritative server every time, the internet would crawl. A cold DNS lookup takes 50 to 200 milliseconds. A cached one takes 1 to 10 milliseconds.

So every DNS resolver caches answers. Every recursive resolver. Every stub resolver in every operating system. Every browser. Sometimes even the application itself caches. There are layers of cache between you and the authoritative server, and each layer holds onto answers as long as it is allowed to.

How long is it allowed to? That is what TTL is for.

### TTL — time to live

Every DNS record has a **TTL** (time to live) field. It is a number of seconds. It says: "you can hold onto this answer for this many seconds before you have to throw it away and ask again."

A small TTL (60 seconds) means resolvers re-ask often. Good for things that change frequently. Bad for traffic — every minute, somebody has to do a full walk.

A large TTL (86400 seconds = 1 day) means resolvers hold onto the answer for a long time. Less traffic. But if you change the answer, it takes a day for everyone to start using the new answer.

Picture the cache:

```
   YOUR LAPTOP              RESOLVER                    AUTH SERVER
   +-----------+        +----------------+         +---------------+
   |stub cache |        |bigger cache    |         |source of truth|
   |(small)    |   <--> |(huge)          |  <-->   |(zone files)   |
   +-----------+        +----------------+         +---------------+
   30 seconds           300 seconds                forever (or till you change it)
   maybe                some answers
                        even longer
```

Each layer caches based on TTL. The closer to you, the smaller the cache (and shorter TTL effective lifetime). The closer to the source, the bigger.

### Negative caching

What happens if you ask for a name that doesn't exist? Like `this-domain-does-not-exist-12345.example`? The authoritative server responds with **NXDOMAIN** (non-existent domain). The resolver caches that answer too — for the **negative TTL** specified in the SOA record.

This is good. Otherwise, every typo would cause a full walk. Negative caching is usually 5 minutes to an hour. Configurable via the SOA's last field.

### The danger of long TTL during emergencies

Imagine your website is hosted at IP `203.0.113.10`. You set the TTL on the A record to 86400 seconds (1 day) because the IP almost never changes. Suddenly, your hosting provider has an outage and you need to move the IP to a backup server at `203.0.113.99`.

You change the DNS record. But every resolver in the world that already has the old answer cached will keep returning `203.0.113.10` for up to 24 hours. Your users will see broken pages for a whole day even after you fixed it.

The fix: lower the TTL **before** you make changes. A common pattern is "lower TTL to 60 seconds 2 days before a planned change, do the change, raise it back after a week." This way, when you cut over, every cache flushes within a minute.

### The cost of short TTL

The opposite problem: very low TTL means more queries. If your TTL is 5 seconds, every resolver has to re-ask every 5 seconds. That is more traffic, more load on your authoritative servers, and more latency for users who have to wait through more cold lookups.

Big sites with crazy traffic (Google, Facebook, Cloudflare) can afford very short TTLs because their authoritative servers are massively scaled. Small sites usually use TTLs in the 5-60 minute range as a compromise.

### Caching that goes wrong

Caching is great until it isn't. Some classic disasters:

- **Stale cache after migration**: forgot to lower TTL before moving servers. Users see broken pages for a day.
- **Browser caches your IP forever**: some browsers (especially older Internet Explorer) ignore TTL and cache for hours regardless. Test in incognito mode.
- **Negative cache locks you out**: a typo'd a name during testing, got NXDOMAIN cached for an hour, even after the typo is fixed users are still getting NXDOMAIN.
- **DNS cache poisoning** (rare now): an attacker injected a fake answer into a resolver's cache. We mostly fixed this with random source ports and DNSSEC.

## Recursive vs Iterative Queries

### Why this distinction matters

We touched on this earlier, but let's spend a minute on why it matters.

When you ask your resolver for an answer, you want one round-trip in your code. You don't want to write logic to handle "well, the resolver didn't know but it told me to ask `.com`, so let me parse that response and send another query, and then another." That would be terrible. Your code would be enormous.

So resolvers offer **recursion as a service.** You send one query with the **RD** (recursion desired) flag set. The resolver does all the iterative work behind the scenes. It sends back one final answer. Easy.

But authoritative servers do not recurse. They cannot. They only know about their own zones. If a resolver asked an authoritative server for a name it doesn't own, the authoritative server can only say "not me, here's a referral." So the resolver has to walk the tree itself, iteratively.

### Walk-through of `dig +trace`

The `dig +trace` command shows you the iterative walk. You can run it yourself. We'll do that in Hands-On. Here is a sample of what it prints:

```
$ dig +trace google.com

; <<>> DiG 9.18.18 <<>> +trace google.com
;; global options: +cmd
.                       86400   IN      NS      a.root-servers.net.
.                       86400   IN      NS      b.root-servers.net.
... (13 root server lines) ...
;; Received 525 bytes from 192.168.1.1#53(192.168.1.1) in 8 ms

com.                    172800  IN      NS      a.gtld-servers.net.
com.                    172800  IN      NS      b.gtld-servers.net.
... (13 .com server lines) ...
;; Received 1180 bytes from 198.41.0.4#53(a.root-servers.net) in 16 ms

google.com.             172800  IN      NS      ns1.google.com.
google.com.             172800  IN      NS      ns2.google.com.
google.com.             172800  IN      NS      ns3.google.com.
google.com.             172800  IN      NS      ns4.google.com.
;; Received 644 bytes from 192.5.6.30#53(a.gtld-servers.net) in 24 ms

google.com.             300     IN      A       142.251.40.78
;; Received 56 bytes from 216.239.32.10#53(ns1.google.com) in 12 ms
```

Read it from the bottom up:
- The final answer (A record) came from `ns1.google.com` in 12 ms.
- That server was found by asking `a.gtld-servers.net` (a `.com` TLD server) which took 24 ms.
- That server was found by asking `a.root-servers.net` (a root server) which took 16 ms.
- The root server's address came from your local resolver's hints (8 ms).

Each step is one iterative query. The whole thing took about 60 ms total. Real-world resolvers parallelize and cache aggressively, so most lookups are way faster than this trace suggests.

## DNS over UDP vs TCP

### UDP first

DNS was designed in 1987, when networks were slow and TCP setup was expensive. So DNS uses **UDP** by default. UDP is "send a packet, get a packet, no setup, no teardown." A DNS query is one tiny packet. A DNS reply is one tiny packet. Done. Very fast.

Both queries and replies go on **port 53.**

### TCP fallback

Originally, the DNS spec said "if your reply is bigger than 512 bytes, you have to truncate it and tell the client to retry over TCP." That is the **TC** (truncated) flag in the header. The client sees TC=1, opens a TCP connection to the same server on port 53, and retries the query.

This is rare in practice now thanks to EDNS0 (next section). But it still matters for:

- **AXFR** (zone transfer): always uses TCP because zones can be huge.
- **IXFR** (incremental zone transfer): also TCP usually.
- **DNSSEC**: signed responses are bigger, often need TCP without EDNS0.
- **Anti-spoofing**: some servers prefer TCP for high-value queries because TCP is harder to spoof than UDP.

### Why UDP is still the default

UDP is faster. No three-way handshake. One round trip total. For a query that is going to be cached anyway, the extra TCP setup cost is wasted. Most DNS queries today still use UDP, often with EDNS0 to allow bigger replies.

### Picture

```
UDP DNS QUERY:
   client --[query packet]--> server
   client <--[answer packet]-- server
   total: 1 round-trip, no setup

TCP DNS QUERY (when reply >512B without EDNS0):
   client -[SYN]-> server
   client <-[SYN-ACK]- server
   client -[ACK]-> server
   client -[query]-> server
   client <-[answer]- server
   client -[FIN]-> server
   client <-[FIN-ACK]- server
   total: many round-trips, more setup
```

That is why we want to keep DNS on UDP whenever possible.

## EDNS0

### What it is

**EDNS0** (Extension Mechanisms for DNS, version 0) is a way to add new features to DNS without breaking old servers. Defined in **RFC 6891** (originally **RFC 2671** in 1999). It is a special pseudo-record called **OPT** that goes in the additional section of a DNS message.

The OPT record carries:
- **A bigger UDP buffer size**: the client says "I can accept replies up to 4096 bytes over UDP." This way, big replies don't have to truncate and fall back to TCP.
- **The DO bit**: "DNSSEC OK." Tells the server "please send me DNSSEC records too."
- **Other extensions**: ECS (EDNS Client Subnet), DNS Cookies, padding, and more.

### EDNS Client Subnet (ECS)

Some big resolvers (Google, Cloudflare) use **ECS** (EDNS Client Subnet) to tell the authoritative server which subnet the original client lives in. This lets a CDN return a different IP for users in different parts of the world. Google.com served from a server in Tokyo for users in Japan, and from a server in London for users in the UK.

ECS is a privacy concern (it leaks rough location info to authoritative servers). Cloudflare's `1.1.1.1` does NOT send ECS, on purpose, for privacy.

### Why EDNS0 mattered

Before EDNS0, replies bigger than 512 bytes forced TCP fallback. With EDNS0 declaring 4096-byte UDP buffers, almost no normal answer triggers TCP anymore. This is a huge speedup for DNSSEC and large records.

### Picture of an EDNS0-enabled query

```
+------------------+
| HEADER           |  ARCOUNT=1 (one OPT record)
+------------------+
| QUESTION         |  google.com A?
+------------------+
| ANSWER (empty)   |
+------------------+
| AUTHORITY (empty)|
+------------------+
| ADDITIONAL       |
|   OPT record     |  type=41, max UDP=4096, DO bit=1
+------------------+
```

The OPT record looks like a record but isn't a real record — it just carries flags and options.

## DoH and DoT (Encrypted DNS)

### Why encrypt?

Plain old DNS is unencrypted. Anybody on the path between you and your resolver can:
- See every name you look up (your ISP, the coffee shop Wi-Fi, anyone listening on the wire).
- Modify replies (a malicious network can redirect you to a phishing site by changing the answer).
- Block answers (censorship at the DNS layer is the cheapest way to censor the internet).

To protect against this, two new protocols put DNS inside a TLS tunnel.

### DoT — DNS over TLS

**DoT** (DNS-over-TLS, **RFC 7858**, 2016) wraps DNS in a regular TLS connection on **port 853**. From the wire, it looks like an unidentifiable encrypted stream. From the application, it looks like normal DNS — your DNS query goes in, your DNS answer comes out, just encrypted.

DoT is favored by network admins because port 853 is dedicated, so they can see "hey, this is DNS-over-TLS happening" and decide whether to allow or block it.

### DoH — DNS over HTTPS

**DoH** (DNS-over-HTTPS, **RFC 8484**, 2018) wraps DNS in a regular HTTPS connection on **port 443**. From the wire, it looks like any other HTTPS traffic. Network admins can't easily distinguish DoH from regular web browsing.

DoH is favored by browsers (Firefox, Chrome) and privacy-focused users because it is harder to block. It is disfavored by network admins for the same reason.

### DoQ — DNS over QUIC

**DoQ** (DNS-over-QUIC, **RFC 9250**, 2022) puts DNS inside QUIC instead of TCP+TLS. Faster connection setup. Same privacy benefits. Used by 1.1.1.1 and other modern resolvers.

### Picture

```
PLAIN DNS:
   client --[unencrypted DNS query, port 53]--> resolver
   anyone on the path can read or modify

DoT:
   client --[TLS handshake on port 853]--> resolver
   client --[encrypted DNS query]--> resolver
   client <--[encrypted DNS answer]-- resolver

DoH:
   client --[HTTPS handshake on port 443]--> resolver
   client --[POST /dns-query, body=query]--> resolver
   client <--[200 OK, body=answer]-- resolver
```

### Why this matters

Encrypted DNS protects:
- **Privacy**: your ISP can't see what sites you look up just from DNS.
- **Integrity**: your network can't fake answers.
- **Censorship resistance**: harder to block at the network level.

It does not protect against:
- The resolver itself logging your queries (you have to trust whoever runs the resolver).
- The IP addresses you connect to afterwards (anyone watching the network sees you talk to `142.251.40.78`, even if they don't see the name `google.com`).
- The TLS SNI in your browser request (can leak the name of the site even when DNS is encrypted; Encrypted Client Hello / ECH is fixing this).

So DoH and DoT are part of a bigger picture. They are necessary but not sufficient for full privacy.

## DNSSEC

### The problem DNSSEC solves

Plain DNS has no signatures. When a recursive resolver gets back an answer "google.com is at 142.251.40.78," it has no way to verify that the answer is real. Maybe somebody intercepted the query and forged a reply. Maybe a poisoned cache returned a stale or malicious answer. Plain DNS cannot tell.

**DNSSEC** (DNS Security Extensions, **RFC 4033/4034/4035**, 2005) adds digital signatures to DNS records. Every record in a signed zone has a signature (RRSIG). The recursive resolver can verify the signature against the zone's public key (DNSKEY). If the signature doesn't match, the answer is rejected.

### The trust chain

How does the resolver know the public key (DNSKEY) is real? The parent zone signs a hash of the child's DNSKEY. This is a **DS** (Delegation Signer) record. So:

```
Root zone (.)
   |
   | signs DS for .com  (root signs a hash of .com's DNSKEY)
   v
.com zone
   |
   | signs DS for google.com (.com signs a hash of google.com's DNSKEY)
   v
google.com zone
   |
   | signs every record in the zone
   v
google.com A 142.251.40.78  (RRSIG attached)
```

The resolver follows the chain backwards. It validates the A record against google.com's DNSKEY. It validates google.com's DNSKEY against the DS in .com. It validates the DS against .com's DNSKEY. It validates .com's DNSKEY against the DS in root. It validates the root's DNSKEY against a trust anchor that is hard-coded into the resolver software (the **root trust anchor**, also called the **KSK**).

If every link checks, the answer is genuine. If any link fails, the resolver returns SERVFAIL.

### KSK vs ZSK

Each signed zone has two keys:
- **KSK** (Key-Signing Key): used to sign the DNSKEY records. Long-lived (years). Hash is published in the parent zone as a DS record.
- **ZSK** (Zone-Signing Key): used to sign all the other records. Short-lived (months). Rolled over more often.

This separation lets you change the ZSK without re-signing everything in the parent zone. Only the KSK has to be communicated to the parent.

### NSEC and NSEC3

Signed DNS still needs to prove that something does NOT exist (NXDOMAIN). It does this with **NSEC** records. NSEC says "in this zone, the next name after `apple.example.com` is `cherry.example.com`." If you ask for `banana.example.com`, the resolver gets back the NSEC for `apple.example.com` and confirms that yes, `banana` falls between `apple` and `cherry`, and there is no record for it.

But NSEC has a problem: by walking the NSEC chain, an attacker can enumerate every name in the zone. So **NSEC3** was invented (RFC 5155, 2008). NSEC3 hashes the names before signing, so an attacker cannot easily walk the chain.

### Why most domains still don't use DNSSEC

DNSSEC is operationally complex. Key rollover is tricky. A bad rollover takes your domain offline for hours. Many DNS providers don't support it well. Many TLDs require manual coordination to update DS records. So adoption is slow.

As of 2024, less than 5% of domains in `.com` are signed with DNSSEC. The root zone, most TLDs, and most ccTLDs are signed, but most leaf zones are not.

For high-value zones (banks, governments), DNSSEC is essential. For your blog, it might not be worth the operational pain.

### Picture of DNSSEC trust chain

```
         +-----------+
         |  ROOT (.) |
         |  KSK + ZSK|
         +-----+-----+
               |
               | signs DS for .com
               v
         +-----+-----+
         |   .COM    |
         |  KSK + ZSK|
         +-----+-----+
               |
               | signs DS for google.com
               v
         +------------+
         |GOOGLE.COM  |
         |  KSK + ZSK |
         +-----+------+
               |
               | signs every record
               v
         google.com IN A 142.251.40.78  RRSIG attached
```

Validation walks the same chain in reverse: A → ZSK → DS → parent ZSK → parent DS → root.

## Anycast

### The trick

Anycast is a routing trick. Multiple servers in different parts of the world all advertise the same IP address using BGP (Border Gateway Protocol). When a client sends a packet to that IP, the network routes it to the closest server. Different clients reach different physical servers, all using the same IP.

This is how `8.8.8.8`, `1.1.1.1`, `9.9.9.9`, and the root servers all work. There aren't really 13 root servers — there are hundreds of physical machines all over the world, all pretending to be `a.root-servers.net` or `b.root-servers.net` etc., each advertising the same IPs from different locations.

### Why this is huge

DNS needs to be fast and reliable. Anycast does both:
- **Fast**: the closest server answers, so latency is low everywhere.
- **Reliable**: if one location goes down, BGP routes to the next-closest. No client has to change anything.
- **DDoS-resistant**: an attacker has to attack every site at once to take the service down.

### Picture

```
            +------------------+
            |  CLIENT IN UK    |
            +-------+----------+
                    |
                    | ping 1.1.1.1
                    v
            BGP routes to LONDON
                    |
                    v
              +-----+-----+
              | 1.1.1.1   |   (London server)
              +-----------+

            +------------------+
            |  CLIENT IN JAPAN |
            +-------+----------+
                    |
                    | ping 1.1.1.1
                    v
            BGP routes to TOKYO
                    |
                    v
              +-----+-----+
              | 1.1.1.1   |   (Tokyo server)
              +-----------+

   Same IP. Different physical machines.
   Same answer (because both servers serve the same data).
   Closer = lower latency.
```

This is why public resolvers can handle billions of queries per day with low latency for everyone.

## Public Resolvers

You can use any public resolver you trust. Here are the big ones.

### 8.8.8.8 / 8.8.4.4 — Google Public DNS

Run by Google. Fast. Available since 2009. Sends EDNS Client Subnet (some privacy concern). Logs queries (Google says they delete after 24-48h). Default DNSSEC validation. Supports DoT and DoH.

### 1.1.1.1 / 1.0.0.1 — Cloudflare

Run by Cloudflare. Very fast. Available since 2018. Does NOT send ECS (privacy preserving). Privacy-focused (KPMG audited). Logs are anonymized and deleted within 24h. Default DNSSEC validation. Supports DoT, DoH, and DoQ. There is a `1.1.1.2` variant that adds malware blocking, and a `1.1.1.3` variant that adds adult-content blocking.

### 9.9.9.9 — Quad9

Run by a Swiss nonprofit. Adds threat blocking by default — known malware domains return NXDOMAIN. Privacy-focused. No ECS. Supports DoT and DoH.

### 4.2.2.4 — Level 3 (now CenturyLink)

An old-school resolver from when ISPs ran their own. Still works. Less privacy-focused. Often used as a fallback when other resolvers are unreachable.

### 208.67.222.222 — OpenDNS

Run by Cisco. Has filtering categories. Some logging. Less privacy-focused than 1.1.1.1.

### Choosing one

- **Speed + privacy**: 1.1.1.1
- **Speed + threat blocking**: 9.9.9.9
- **Just works, well-known**: 8.8.8.8
- **Specifically for kids / family**: 1.1.1.3 or OpenDNS Family Shield
- **Privacy-focused, blocks ads**: NextDNS, AdGuard DNS, or your own Pi-hole

If you can run your own resolver (Unbound, Knot Resolver, BIND), that gives you the most privacy: nobody else logs your queries.

## Common DNS Errors

DNS error codes are called **RCODEs.** They appear in the header of every DNS response. Here are the ones you will see, what they mean, and how to fix them.

### NXDOMAIN — non-existent domain

**Verbatim message** (from `dig`):
```
;; ->>HEADER<<- opcode: QUERY, status: NXDOMAIN, id: 12345
```

**What it means**: the name does not exist. The authoritative server says "I'm authoritative for this zone, and there is no record with this name."

**Canonical fix**: check the spelling. Make sure the domain is registered. Make sure you queried the right server. Make sure the record actually exists (use the registrar's web UI or the authoritative server's zone file). Negative caching can cause NXDOMAIN to stick for a while even after you fix the record — flush the cache or wait for the negative TTL.

### NODATA — name exists, but not this type

**Not technically an RCODE** — it's NOERROR with zero answers. But people call it NODATA in conversation.

**What it means**: the name exists, but there is no record of the type you asked for. For example, if `example.com` has an A record but no AAAA record, asking for AAAA returns NODATA.

**Canonical fix**: query the right type, or add the missing record.

### SERVFAIL — server failure

**Verbatim message**:
```
;; ->>HEADER<<- opcode: QUERY, status: SERVFAIL, id: 12345
```

**What it means**: something broke. The resolver couldn't get an answer. Maybe the authoritative server is down, maybe DNSSEC validation failed, maybe the resolver hit a timeout, maybe there's a misconfiguration.

**Canonical fix**: check authoritative server is reachable (ping, traceroute). Check DNSSEC chain (`delv +rtrace`). Check if the resolver's upstream is reachable. Try a different resolver (`dig @8.8.8.8` vs `dig @1.1.1.1`). Look at the resolver's logs.

### REFUSED — server refused to answer

**Verbatim message**:
```
;; ->>HEADER<<- opcode: QUERY, status: REFUSED, id: 12345
```

**What it means**: the server is configured not to answer this query. Often because the server doesn't recurse for non-customers (open recursion is a security risk), or because the server is authoritative only and you asked for something it doesn't own.

**Canonical fix**: use a different resolver, or configure your authoritative server to allow your client.

### NOTIMP — not implemented

**What it means**: the server doesn't support this opcode or query. Rare in practice.

**Canonical fix**: change the query type or use a different server.

### FORMERR — format error

**What it means**: the server couldn't parse the query. Usually means a malformed packet (truncation, bad encoding).

**Canonical fix**: this is almost always a bug somewhere. Capture the packet, look at it in Wireshark, and figure out what's malformed. Update your DNS client library.

### YXDOMAIN, YXRRSET, NXRRSET, NOTAUTH

These appear during dynamic DNS updates (RFC 2136). Rare in normal use.

### BADKEY (DNSSEC), BADTIME (DNSSEC), BADSIG, etc.

DNSSEC-specific RCODEs. They appear in the EDNS extended RCODE field.

- **BADKEY**: the key used to sign isn't trusted.
- **BADTIME**: the signature is outside its valid time window. Often means the system clock is wrong.
- **BADSIG**: the signature doesn't match.

**Canonical fix for BADTIME**: check the system clock. Run `timedatectl` or `chronyc tracking`. DNSSEC requires accurate time. If your clock is off by more than a few minutes, DNSSEC fails everywhere.

## Hands-On

Time to actually try things. You will need a terminal. On most Linux systems, open one with **Ctrl-Alt-T** or look for an app called "Terminal." On macOS, open the "Terminal" app. On Windows, use WSL or PowerShell with the modern DNS commands.

For each command below, type the part after the `$` and press Enter. The lines without `$` are what the computer prints back. Your output might differ (different IPs, different version numbers) but the shape will be the same.

If a command does not work and you get **`command not found`**, you can install it. On Ubuntu/Debian: `sudo apt install dnsutils bind9-host knot-dnsutils ldnsutils`. On Fedora: `sudo dnf install bind-utils knot-utils ldns-utils`. On macOS: `brew install bind ldns knot`.

### 1. Look up a name with dig

`dig` is the standard DNS lookup tool. It prints the answer and a lot of context. Stands for "Domain Information Groper."

```
$ dig google.com

; <<>> DiG 9.18.18 <<>> google.com
;; global options: +cmd
;; Got answer:
;; ->>HEADER<<- opcode: QUERY, status: NOERROR, id: 38520
;; flags: qr rd ra; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 1

;; OPT PSEUDOSECTION:
; EDNS: version: 0, flags:; udp: 1232
;; QUESTION SECTION:
;google.com.                    IN      A

;; ANSWER SECTION:
google.com.             224     IN      A       142.251.40.78

;; Query time: 16 msec
;; SERVER: 192.168.1.1#53(192.168.1.1) (UDP)
;; WHEN: Sat Apr 27 14:32:11 PDT 2026
;; MSG SIZE  rcvd: 55
```

Lots of output. Look for the **ANSWER SECTION**. That's the actual answer. `google.com` has an A record pointing to `142.251.40.78` with a TTL of 224 seconds. The query took 16 ms via your local resolver at 192.168.1.1 over UDP.

### 2. Just the answer, please

Often you just want the IP, not all the context. Use `+short`.

```
$ dig +short google.com
142.251.40.78
```

One line. The IP. Done.

### 3. See the iterative walk

Ask `dig` to walk from the root and show every step.

```
$ dig +trace google.com

; <<>> DiG 9.18.18 <<>> +trace google.com
;; global options: +cmd
.                       86400   IN      NS      a.root-servers.net.
.                       86400   IN      NS      b.root-servers.net.
.                       86400   IN      NS      c.root-servers.net.
.                       86400   IN      NS      d.root-servers.net.
.                       86400   IN      NS      e.root-servers.net.
.                       86400   IN      NS      f.root-servers.net.
.                       86400   IN      NS      g.root-servers.net.
.                       86400   IN      NS      h.root-servers.net.
.                       86400   IN      NS      i.root-servers.net.
.                       86400   IN      NS      j.root-servers.net.
.                       86400   IN      NS      k.root-servers.net.
.                       86400   IN      NS      l.root-servers.net.
.                       86400   IN      NS      m.root-servers.net.
;; Received 525 bytes from 192.168.1.1#53(192.168.1.1) in 8 ms

com.                    172800  IN      NS      a.gtld-servers.net.
com.                    172800  IN      NS      b.gtld-servers.net.
... (more .com servers) ...
;; Received 1180 bytes from 198.41.0.4#53(a.root-servers.net) in 16 ms

google.com.             172800  IN      NS      ns1.google.com.
google.com.             172800  IN      NS      ns2.google.com.
google.com.             172800  IN      NS      ns3.google.com.
google.com.             172800  IN      NS      ns4.google.com.
;; Received 644 bytes from 192.5.6.30#53(a.gtld-servers.net) in 24 ms

google.com.             300     IN      A       142.251.40.78
;; Received 56 bytes from 216.239.32.10#53(ns1.google.com) in 12 ms
```

This is the live iterative walk. Read it bottom up. Each step is real. You can repeat any step manually with `dig @<server> <name>`.

### 4. Use a specific resolver

Use `@` to send the query to a specific server, bypassing your default.

```
$ dig @8.8.8.8 google.com

; <<>> DiG 9.18.18 <<>> @8.8.8.8 google.com
;; ANSWER SECTION:
google.com.             46      IN      A       142.251.40.78

;; SERVER: 8.8.8.8#53(8.8.8.8) (UDP)
;; Query time: 28 msec
```

That sent the query directly to Google Public DNS instead of your local resolver. Useful for testing, debugging, or comparing answers from different resolvers.

### 5. Look up a different record type

Use `-t` (or just put the type after the name) to ask for something other than A.

```
$ dig -t MX google.com

;; ANSWER SECTION:
google.com.             300     IN      MX      10 smtp.google.com.
```

That's the mail exchanger for `google.com`. If you sent email to `someone@google.com`, this is where the mail would be delivered.

### 6. IPv6 lookup

```
$ dig -t AAAA google.com

;; ANSWER SECTION:
google.com.             300     IN      AAAA    2607:f8b0:4005:80c::200e
```

`google.com` over IPv6.

### 7. Look up a TXT record

DMARC records are TXT records under `_dmarc.<domain>`.

```
$ dig -t TXT _dmarc.google.com

;; ANSWER SECTION:
_dmarc.google.com.      300     IN      TXT     "v=DMARC1; p=reject; rua=mailto:mailauth-reports@google.com"
```

That tells the world: "If you receive email claiming to be from `google.com` and it fails authentication, REJECT it (don't bounce, don't quarantine, just refuse). Send aggregate reports here."

### 8. Look up DNSSEC records

```
$ dig +dnssec example.com

;; ANSWER SECTION:
example.com.            71849   IN      A       93.184.216.34
example.com.            71849   IN      RRSIG   A 13 2 86400 20260506...

;; AUTHORITY SECTION:
example.com.            71849   IN      NS      a.iana-servers.net.
...

;; ADDITIONAL SECTION:
... (more RRSIG and DNSKEY records) ...
```

The `+dnssec` flag asks the resolver to include DNSSEC records (RRSIG, DNSKEY, NSEC). You can see the signature attached to each RRset.

### 9. Reverse lookup (PTR)

Convert an IP back to a name with `-x`.

```
$ dig -x 8.8.8.8

;; ANSWER SECTION:
8.8.8.8.in-addr.arpa.   75976   IN      PTR     dns.google.
```

So `8.8.8.8` reverses to `dns.google`. Handy when you see a strange IP in your logs and want to know who owns it.

### 10. Force TCP

Use `+tcp` to force TCP instead of UDP.

```
$ dig +tcp google.com

;; ANSWER SECTION:
google.com.             300     IN      A       142.251.40.78
;; SERVER: 192.168.1.1#53(192.168.1.1) (TCP)
```

Notice the `(TCP)` at the end — confirms the query went over TCP.

### 11. The friendlier `host` command

`host` is a simpler tool than `dig`.

```
$ host google.com
google.com has address 142.251.40.78
google.com has IPv6 address 2607:f8b0:4005:80c::200e
google.com mail is handled by 10 smtp.google.com.
```

Three lines, three record types, all at once.

### 12. The classic `nslookup`

`nslookup` is the oldest tool. Still installed everywhere.

```
$ nslookup google.com
Server:         192.168.1.1
Address:        192.168.1.1#53

Non-authoritative answer:
Name:   google.com
Address: 142.251.40.78
Name:   google.com
Address: 2607:f8b0:4005:80c::200e
```

Old syntax, still works. "Non-authoritative answer" means the answer came from a cache, not from the authoritative server directly.

### 13. Use the system's name resolution stack

`getent hosts` runs the same resolution path that any application uses. It honors `/etc/nsswitch.conf` (which says "first try /etc/hosts, then DNS").

```
$ getent hosts google.com
142.251.40.78   google.com
```

If `/etc/hosts` has an entry for `google.com`, that overrides DNS — getent will return whatever's in `/etc/hosts`.

### 14. All addresses for a name

`getent ahosts` returns both IPv4 and IPv6.

```
$ getent ahosts google.com
142.251.40.78   STREAM google.com
142.251.40.78   DGRAM
142.251.40.78   RAW
2607:f8b0:4005:80c::200e STREAM
2607:f8b0:4005:80c::200e DGRAM
2607:f8b0:4005:80c::200e RAW
```

Three lines per address (one for each socket type — STREAM=TCP, DGRAM=UDP, RAW=ICMP).

### 15. See your DNS configuration

```
$ cat /etc/resolv.conf
# Generated by NetworkManager
nameserver 192.168.1.1
nameserver 8.8.8.8
search lan
```

This file tells the stub resolver which DNS servers to use. `nameserver` lines are tried in order. The `search` line is appended to short names.

On systemd systems, `/etc/resolv.conf` may be a symlink to a file managed by `systemd-resolved`. Don't edit it by hand on those systems.

### 16. Check name resolution policy

```
$ cat /etc/nsswitch.conf | grep hosts
hosts:          files dns
```

This says "for hostname lookups, first check `/etc/hosts` (`files`), then ask DNS." Some systems also have `mdns` (multicast DNS), `nis`, or `myhostname`.

### 17. Check systemd-resolved status

If your system uses systemd's built-in resolver, this shows you the current state.

```
$ resolvectl status
Global
       Protocols: -LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
resolv.conf mode: stub

Link 2 (wlp3s0)
    Current Scopes: DNS
         Protocols: +DefaultRoute +LLMNR -mDNS -DNSOverTLS DNSSEC=no/unsupported
Current DNS Server: 192.168.1.1
       DNS Servers: 192.168.1.1
        DNS Domain: lan
```

You can see which resolver each network interface is using.

### 18. Look up a name via systemd-resolved

```
$ resolvectl query google.com
google.com: 142.251.40.78
            2607:f8b0:4005:80c::200e
-- Information acquired via protocol DNS in 14.7ms.
-- Data is authenticated: no; Data was acquired via local or encrypted transport: no
-- Data from: cache network
```

This goes through the systemd-resolved cache, so it can be much faster than `dig` on the second run.

### 19. The Knot CLI: `kdig`

`kdig` is the Knot project's equivalent of `dig`. Same syntax, slightly cleaner output.

```
$ kdig google.com
;; ->>HEADER<<- opcode: QUERY; status: NOERROR; id: 42031
;; Flags: qr rd ra; QUERY: 1; ANSWER: 1; AUTHORITY: 0; ADDITIONAL: 1

;; QUESTION SECTION:
;; google.com.                  IN      A

;; ANSWER SECTION:
google.com.             223     IN      A       142.251.40.78

;; Received 55 B
;; From 192.168.1.1@53(UDP) in 9.2 ms
```

### 20. The ldns CLI: `drill`

`drill` is from the ldns library. Another `dig`-style tool.

```
$ drill google.com
;; ->>HEADER<<- opcode: QUERY, rcode: NOERROR, id: 19478
;; flags: qr rd ra ; QUERY: 1, ANSWER: 1, AUTHORITY: 0, ADDITIONAL: 0
;; QUESTION SECTION:
;; google.com.  IN      A

;; ANSWER SECTION:
google.com.     228     IN      A       142.251.40.78

;; Query time: 12 msec
;; SERVER: 192.168.1.1
;; WHEN: Sat Apr 27 14:35:01 2026
;; MSG SIZE  rcvd: 44
```

Same kind of output. Different tool. Pick whichever you like.

### 21. Validate DNSSEC

`delv` (BIND's DNSSEC lookup tool) prints validation status.

```
$ delv +rtrace example.com

;; fetch: example.com/A
;; fetch: example.com/DNSKEY
;; fetch: example.com/DS
;; fetch: com/DNSKEY
;; fetch: com/DS
;; fetch: ./DNSKEY
; fully validated
example.com.            86400   IN      A       93.184.216.34
example.com.            86400   IN      RRSIG   A 13 2 86400 20260506...
```

The `; fully validated` line means DNSSEC succeeded. If validation fails, you'll see why (BADKEY, BADTIME, missing chain, etc.).

### 22. Look up domain ownership

`whois` is not DNS, but it's the companion tool. It queries the registry for ownership info.

```
$ whois google.com | head -20
   Domain Name: GOOGLE.COM
   Registry Domain ID: 2138514_DOMAIN_COM-VRSN
   Registrar WHOIS Server: whois.markmonitor.com
   Registrar URL: http://www.markmonitor.com
   Updated Date: 2019-09-09T15:39:04Z
   Creation Date: 1997-09-15T04:00:00Z
   Registry Expiry Date: 2028-09-14T04:00:00Z
   Registrar: MarkMonitor Inc.
   Registrar IANA ID: 292
   Registrar Abuse Contact Email: abuse@markmonitor.com
   Registrar Abuse Contact Phone: +1.2086851750
   Domain Status: clientDeleteProhibited
   ...
```

`google.com` was registered in 1997. Domain expires in 2028. Registrar is MarkMonitor (a corporate registrar specifically for big brands).

### 23. DoH-style query (mention)

`dig` doesn't do DoH directly, but `kdig` does (with `+https`):

```
$ kdig @1.1.1.1 +https google.com
;; HTTPS session (HTTP/2-POST)-(cloudflare-dns.com/dns-query)-(status: 200)
;; ANSWER SECTION:
google.com.             300     IN      A       142.251.40.78
```

For real DoH testing, browsers and tools like `cloudflared` are more typical.

### 24. Watch DNS traffic on the wire

`tcpdump` filters network traffic. UDP port 53 is DNS.

```
$ sudo tcpdump -i any -n udp port 53
listening on any, link-type LINUX_SLL (Linux cooked v1), capture size 262144 bytes
14:38:01.123456 IP 192.168.1.42.43210 > 192.168.1.1.53: 12345+ A? google.com. (28)
14:38:01.137819 IP 192.168.1.1.53 > 192.168.1.42.43210: 12345 1/0/1 A 142.251.40.78 (55)
```

Two lines: query and response. The `12345+` is the transaction ID with the recursion-desired flag. `A?` means "asking for an A record." The response `12345 1/0/1` means "answer ID 12345, 1 answer, 0 authority, 1 additional record."

### 25. Pretty-print DNS in tshark

Wireshark's command-line cousin.

```
$ sudo tshark -i any -Y 'dns' -O dns
Capturing on 'any'
Frame 1: 70 bytes on wire, 70 captured
Domain Name System (query)
    Transaction ID: 0x3039
    Flags: 0x0100 Standard query
    Questions: 1
    Queries
        google.com: type A, class IN
            Name: google.com
            Type: A (Host Address) (1)
            Class: IN (0x0001)
```

Tshark gives you the full decoded DNS message, super readable. Good for learning or debugging weird DNS issues.

### 26. Probe for open recursion

`nmap` has DNS scripts. This one checks if a server allows recursion for non-customers (which is how it should NOT be configured for an authoritative server).

```
$ nmap --script dns-recursion 8.8.8.8
Starting Nmap 7.94 ...
Nmap scan report for dns.google (8.8.8.8)
Host is up (0.0084s latency).

PORT   STATE SERVICE
53/tcp open  domain
| dns-recursion: Recursion appears to be enabled
```

8.8.8.8 is supposed to be a public recursor, so recursion is enabled by design. If you ran this against an authoritative-only server and it said recursion was enabled, that would be a misconfiguration.

### 27. Dump the BIND cache

If you run BIND yourself, you can ask it to dump its cache. (This requires a running BIND with `rndc` enabled.)

```
$ sudo rndc dumpdb -cache
$ cat /var/cache/bind/named_dump.db | head -20
;
; Start view _default
;
; Cache dump of view '_default' (cache _default)
;
$DATE 20260427143511
; authanswer
google.com.             223     A       142.251.40.78
                        223     RRSIG   A 13 2 300 ...
;
; Address database dump
;
ns1.google.com.         86400   A       216.239.32.10
ns2.google.com.         86400   A       216.239.34.10
```

You can see exactly what BIND has cached, with TTLs counting down.

### 28. Dump the Unbound cache

If you run Unbound (the other big open-source recursive resolver):

```
$ sudo unbound-control dump_cache | head -10
START_RRSET_CACHE
;rrset 144 1 0 7 3
google.com.     224     IN      A       142.251.40.78
;rrset 86392 1 0 11 3
google.com.     86392   IN      NS      ns1.google.com.
google.com.     86392   IN      NS      ns2.google.com.
```

Same idea. See exactly what Unbound has cached.

### 29. Try an ANY query (most resolvers refuse)

ANY queries used to ask for "all record types." They're now widely refused or return only a tiny subset because they're often used in DDoS amplification attacks (small query, big answer).

```
$ dig +short ANY google.com
"v=spf1 include:_spf.google.com ~all"
```

Just one line. Most resolvers used to return many records here. Modern best practice (RFC 8482) is to refuse ANY or return minimal data.

### 30. Multicast DNS (mDNS)

Local-network name resolution without a DNS server. Apple calls it Bonjour. Linux calls it Avahi. Uses port 5353 and the multicast group 224.0.0.251.

```
$ dig -p 5353 -t PTR _services._dns-sd._udp.local @224.0.0.251

;; QUESTION SECTION:
;_services._dns-sd._udp.local.  IN      PTR

;; ANSWER SECTION:
_services._dns-sd._udp.local. 4500 IN   PTR     _printer._tcp.local.
_services._dns-sd._udp.local. 4500 IN   PTR     _ipp._tcp.local.
_services._dns-sd._udp.local. 4500 IN   PTR     _airplay._tcp.local.
```

That asks the local network "what services advertise themselves over mDNS?" You'll see printers, AirPlay receivers, and other devices.

### 31. Compare resolvers

Quick way to compare answers from multiple resolvers:

```
$ for r in 8.8.8.8 1.1.1.1 9.9.9.9; do echo "==$r=="; dig +short @$r google.com; done
==8.8.8.8==
142.251.40.78
==1.1.1.1==
142.251.40.78
==9.9.9.9==
142.251.40.78
```

If they all agree, fine. If they disagree, that's interesting — could be DNS-based load balancing, geo-DNS, or something weirder.

### 32. Time the lookup

```
$ time dig +short google.com
142.251.40.78

real    0m0.025s
user    0m0.014s
sys     0m0.005s
```

25 milliseconds total. Most of that was probably the DNS query itself.

### 33. Force IPv6 transport

```
$ dig -6 google.com
```

Forces the query to be sent over IPv6 instead of IPv4. Useful for testing.

### 34. Use an alternate port

```
$ dig -p 5353 google.com @127.0.0.1
```

Useful if you have a local DNS server (like Pi-hole, dnsmasq, or your own resolver) on a non-default port.

## Common Confusions

### "Isn't DNS just one big server somewhere?"

**The confusion:** people think DNS is a single computer that has every name in the world.

**The fix:** DNS is a tree of millions of authoritative servers, each owning a tiny piece. There is no single server with everything. The root servers are the very top, and they only know about the top-level domains. Every level below is owned by somebody else. The whole point of DNS is that nobody has the whole phone book.

### "If I change my IP address, everyone sees the new one immediately, right?"

**The confusion:** I update the DNS record, the world updates with me.

**The fix:** No. Caches keep the old answer for as long as the TTL allows. If your TTL was 86400 seconds, it can take a full day for everyone's cache to expire. Plan ahead: lower TTL before you change, raise it back after the dust settles.

### "DoH and a VPN are the same thing, right?"

**The confusion:** both encrypt my traffic.

**The fix:** DoH only encrypts DNS queries. The actual website you connect to is still visible to anyone watching the network (because they can see your TCP packets to that IP, and the TLS SNI in your handshake). A VPN encrypts all traffic, not just DNS. They solve different (overlapping) problems.

### "If DNS is broken, the whole internet is broken, right?"

**The confusion:** DNS = internet.

**The fix:** The internet keeps working without DNS — packets still flow if you know the IP address. But almost everything humans do uses DNS, so when DNS breaks, it feels like the internet is broken. You can sometimes still reach a site by typing its IP directly. Try it: `dig +short example.com`, then put the IP in your browser and see what happens (probably an error from the web server expecting a hostname, but the network connection works).

### "A CNAME and an A record do the same thing, right?"

**The confusion:** both end up at an IP.

**The fix:** A is direct: this name → this IP. CNAME is indirect: this name → another name (which then gets resolved to an IP). CNAMEs add a step. They also cannot be at the apex of a zone (you can't have `example.com IN CNAME something.else.com`). And they cannot coexist with other records at the same name (so an MX and a CNAME on the same name is illegal).

### "An IP address and a domain name are the same thing, just different names?"

**The confusion:** I see `google.com` and `142.251.40.78` mean the same thing.

**The fix:** They are different layers. The IP is the actual address of a physical (or virtual) machine. The domain name is just a label that points at one or more IPs. Many domains can point at one IP (shared hosting). One domain can point at many IPs (round-robin, geo-DNS, anycast). The name and the address are not interchangeable identities.

### "Why do I need both A and AAAA records?"

**The confusion:** I have an A, why also AAAA?

**The fix:** Different clients use different protocols. IPv4-only clients use A. IPv6-capable clients prefer AAAA. To reach the maximum audience, publish both. If you only publish A, you cut off IPv6-only networks (which are growing). If you only publish AAAA, you cut off old IPv4-only networks (still common).

### "MX records contain the IP of the mail server, right?"

**The confusion:** MX = mail IP.

**The fix:** MX records contain the **name** of the mail server, not the IP. The receiver looks up the MX, gets a name, then does another lookup to get the A or AAAA of that name. This is on purpose — it lets you change the mail server's IP without updating the MX record.

### "Why is my system using a different DNS server than I configured?"

**The confusion:** I set 1.1.1.1 in the network settings but `dig` uses something else.

**The fix:** Modern systems have layers. NetworkManager talks to systemd-resolved, which has its own configuration. `/etc/resolv.conf` might be a symlink to a stub file at `127.0.0.53` that systemd-resolved is listening on. Run `resolvectl status` to see what each interface is actually using. If you bypassed systemd-resolved, you might be hitting a totally different server than you expected.

### "DNSSEC means encrypted DNS, right?"

**The confusion:** they both have "Sec" or sound secure.

**The fix:** DNSSEC = Authenticity, not encryption. DNSSEC signs records so you can verify they are real. Anyone watching the wire can still see your queries and answers in plain text — DNSSEC doesn't hide them. To hide DNS traffic, use DoT or DoH (DNS over TLS / DNS over HTTPS). DNSSEC and DoT/DoH are complementary, not the same.

### "If a CNAME points at a name, can I have circular CNAMEs?"

**The confusion:** can I make name A point to name B and name B point to name A?

**The fix:** Technically you can configure that, but resolvers will refuse to follow it after a few hops to prevent infinite loops. They will return SERVFAIL or just give up. Don't make CNAME chains longer than 2-3 hops.

### "I changed my DNS, why does my browser still show the old site?"

**The confusion:** I updated DNS, refreshed the page, and still see the old version.

**The fix:** Many layers of cache. (1) Your browser has a DNS cache. (2) Your operating system has a DNS cache. (3) Your router has a DNS cache. (4) Your ISP's resolver has a DNS cache. Each holds the answer until its TTL expires. To flush: restart the browser, run `sudo systemd-resolve --flush-caches` (or `sudo killall -HUP mDNSResponder` on macOS), or just wait for the TTL.

### "8.8.8.8 is the same server as 8.8.4.4, right?"

**The confusion:** they're both Google.

**The fix:** Both are Google Public DNS, but they are separate IPs hosted in (often) different physical machines using anycast. Listing both gives you redundancy: if one is unreachable, your stub resolver tries the other.

### "Why does my friend get a different IP than me for the same site?"

**The confusion:** geographic load balancing seems wrong.

**The fix:** Big sites use geo-DNS or anycast to direct different users to different servers based on location. So you might get a server in Frankfurt and your friend in Tokyo gets one in Tokyo, even though you both typed the same name. This is on purpose — lower latency for everyone.

### "Why is my DNS lookup so slow on the first request?"

**The confusion:** the first time I visit a site, DNS is slow. Why?

**The fix:** Cold cache. Your resolver has to walk all the way to the authoritative server. Subsequent requests use the cache and are way faster. This is also why "first packet latency" matters — most sites optimize for this with low TTLs near edge or aggressive prefetch.

### "Why does this site sometimes work and sometimes not?"

**The confusion:** intermittent DNS failures.

**The fix:** Usually one of: a particular authoritative server is flaky and your resolver hits it sometimes; DNSSEC validation is failing intermittently; one of multiple A records points at a dead server; or your local resolver is handing back stale records sometimes. Try `dig` against a different resolver and compare. Try `dig +trace` to see if all the authoritative servers are healthy.

## Vocabulary

| Term | Plain English |
|------|---------------|
| **DNS** | Domain Name System. The internet's address book. Turns names into IPs. |
| **Query** | A DNS request. "What is the IP of `google.com`?" |
| **Response** | A DNS reply. The answer. |
| **Recursive** | A query mode where the resolver does all the work and returns the final answer. |
| **Iterative** | A query mode where the server says "I don't know, ask this other server." |
| **Authoritative** | A server that is the source of truth for a zone. |
| **Recursive resolver** | A DNS server that walks the tree on your behalf. (e.g., 8.8.8.8) |
| **Stub resolver** | The tiny DNS client built into your OS. Sends queries to a recursive resolver. |
| **Root server** | The top of the DNS tree. Knows about TLDs. 13 named servers (anycast to hundreds of physical machines). |
| **TLD** | Top-Level Domain. `.com`, `.org`, `.uk`, `.io`, etc. |
| **gTLD** | Generic TLD. `.com`, `.org`, `.net`, `.app`, etc. |
| **ccTLD** | Country-code TLD. `.uk`, `.de`, `.jp`, etc. |
| **Zone** | A piece of the DNS tree owned by one entity. `google.com` is a zone. |
| **Glue record** | An IP record included with a referral so the resolver can contact the next NS without another lookup. |
| **NS** | Name Server record. Tells you which servers are authoritative for a zone. |
| **SOA** | Start Of Authority record. The "title page" of a zone. |
| **A** | An IPv4 address record. |
| **AAAA** | An IPv6 address record (pronounced "quad-A"). |
| **CNAME** | Canonical Name. An alias from one name to another. |
| **ANAME** | A vendor-specific fake CNAME-at-apex. Not a real DNS record. |
| **MX** | Mail Exchanger. Tells you where to deliver email for a domain. |
| **TXT** | Free-form text record. Used for SPF, DKIM, DMARC, domain verification. |
| **SPF** | Sender Policy Framework. A TXT record listing legit email senders for a domain. |
| **DKIM** | DomainKeys Identified Mail. Email signing using a TXT record holding a public key. |
| **DMARC** | Domain-based Message Authentication. Email policy in `_dmarc.<domain>` TXT record. |
| **PTR** | Pointer record. Reverse lookup, IP → name. |
| **SRV** | Service record. `_service._proto.domain` → `priority weight port target`. |
| **CAA** | Certificate Authority Authorization. Limits which CAs can issue certificates for your domain. |
| **HTTPS RR** | Newer record (RFC 9460) carrying HTTPS service parameters in DNS. |
| **SVCB** | Service Binding record. The general form of HTTPS RR. |
| **DS** | Delegation Signer. A hash of a child zone's DNSKEY, lives in the parent. |
| **DNSKEY** | A zone's public key. Used to verify RRSIGs. |
| **RRSIG** | A signature over an RRset. The core of DNSSEC. |
| **NSEC** | Next Secure record. Proves a name doesn't exist. |
| **NSEC3** | NSEC with hashed names, prevents zone walking. |
| **NSEC3PARAM** | Parameters for NSEC3 (algorithm, salt, iterations). |
| **DLV** | DNSSEC Lookaside Validation. Old, deprecated way to do DNSSEC. |
| **TLSA** | TLS Association record. Used by DANE for TLS cert binding via DNSSEC. |
| **OPENPGPKEY** | PGP key for an email address, in DNS. |
| **SMIMEA** | S/MIME certificate for an email address, in DNS. |
| **NXDOMAIN** | Non-eXistent DOMAIN. The name does not exist. |
| **NODATA** | The name exists, but not for this record type. (Technically NOERROR with empty answer.) |
| **SERVFAIL** | Server failure. Something broke. Often DNSSEC validation, server down, or timeout. |
| **REFUSED** | The server refused to answer. Usually a permissions/policy issue. |
| **NOTIMP** | Not implemented. The server doesn't support this kind of query. |
| **FORMERR** | Format error. The query was malformed. |
| **BADKEY** | DNSSEC: untrusted signing key. |
| **BADTIME** | DNSSEC: signature outside its valid time window. Often a clock skew issue. |
| **TTL** | Time To Live. Seconds a record can be cached. |
| **Negative cache** | Caching of NXDOMAIN/NODATA. TTL set by the SOA's last field. |
| **EDNS0** | Extension Mechanisms for DNS. Adds bigger UDP buffers, DNSSEC support, etc. |
| **DO bit** | DNSSEC OK. Set in EDNS0 to ask for DNSSEC records. |
| **AD bit** | Authenticated Data. Set in response when DNSSEC validation passed. |
| **RD bit** | Recursion Desired. Set in query when you want the resolver to recurse. |
| **RA bit** | Recursion Available. Set in response when the server offers recursion. |
| **AA bit** | Authoritative Answer. Set when the answer comes from an authoritative server. |
| **TC bit** | Truncated. Set when the answer didn't fit in the packet. |
| **Anycast** | One IP advertised from many locations via BGP. Same IP, different physical servers. |
| **BGP** | Border Gateway Protocol. The internet's routing protocol. |
| **ECS** | EDNS Client Subnet. Tells authoritative servers which subnet the client is in (for geo-DNS). |
| **DoH** | DNS over HTTPS (RFC 8484, 2018). Port 443. Encrypted. |
| **DoT** | DNS over TLS (RFC 7858, 2016). Port 853. Encrypted. |
| **DoQ** | DNS over QUIC (RFC 9250, 2022). Encrypted. |
| **DNSCrypt** | An older encrypted DNS protocol from before DoT/DoH. |
| **dnsmasq** | A lightweight DNS+DHCP server. Used in home routers and Raspberry Pis. |
| **BIND** | The classic open-source DNS server. From ISC. |
| **PowerDNS** | A modern open-source DNS server, often paired with a SQL backend. |
| **Knot** | An open-source DNS server (authoritative + Knot Resolver for recursive). |
| **Unbound** | A popular open-source recursive resolver. |
| **NSD** | Name Server Daemon. Open-source authoritative-only DNS server. |
| **CoreDNS** | A pluggable DNS server, often used in Kubernetes. |
| **Cloudflare** | Run 1.1.1.1, a public resolver. Privacy-focused. |
| **Google Public DNS** | 8.8.8.8 / 8.8.4.4. Run by Google. Fast, widely used. |
| **Quad9** | 9.9.9.9. Privacy-focused, threat-blocking. Swiss nonprofit. |
| **OpenDNS** | 208.67.222.222. Run by Cisco. Has filtering categories. |
| **Root hints** | A list of root server IPs baked into resolver software. Used to bootstrap recursion. |
| **Root zone** | The top of the DNS tree. Contains DS records for every TLD. |
| **AXFR** | Full zone transfer. Used to copy a whole zone from primary to secondary. |
| **IXFR** | Incremental zone transfer. Only the changes. |
| **NOTIFY** | A primary server tells secondaries "I have updates, come fetch them." |
| **Validating resolver** | A recursive resolver that checks DNSSEC signatures. |
| **Lame delegation** | When a parent zone says "ask this NS" but that NS doesn't actually answer for the zone. |
| **In-bailiwick** | A name server that lives within the zone it serves. (e.g., `ns1.google.com` for `google.com`.) |
| **Out-of-bailiwick** | A name server outside its zone. (e.g., `ns1.example.net` for `google.com`.) |
| **FQDN** | Fully Qualified Domain Name. Ends with a dot, like `google.com.`. |
| **Label** | One piece of a domain name between dots. `google.com` has labels `google` and `com`. |
| **Dotted notation** | The way DNS names are written: labels separated by dots. |
| **IDN** | Internationalized Domain Name. Names with non-ASCII characters. |
| **Punycode** | Encoding for IDNs into ASCII (e.g., `xn--n3h` for the snowman emoji). |
| **DNSSEC chain of trust** | Root → TLD → zone → record. Each level signs a hash of the next. |
| **KSK** | Key Signing Key. Long-lived. Signs the DNSKEY records. Hash published as DS in parent. |
| **ZSK** | Zone Signing Key. Short-lived. Signs the actual records in the zone. |
| **Key rollover** | Replacing a DNSSEC key with a new one without breaking validation. |
| **RRset** | A group of records sharing the same name, type, and class. (e.g., all the A records for `google.com`.) |
| **Owner name** | The name a record applies to. (Left side of a record.) |
| **Class** | Almost always IN (Internet). Other classes (CH, HS) are historical. |
| **DDoS amplification** | Using DNS responses (small query, big reply) to amplify an attack. |
| **Response Rate Limiting (RRL)** | Authoritative server feature that throttles responses to prevent amplification. |
| **Spoofing** | Sending a fake DNS response from an attacker. |
| **Cache poisoning** | Injecting a fake answer into a resolver's cache. |
| **Kaminsky attack** | A 2008 cache poisoning attack. Forced the move to randomized source ports and (later) DNSSEC. |
| **0x20 case randomization** | A trick where queries randomize letter case (`gOoGlE.cOm`) to make spoofing harder. |
| **mDNS** | Multicast DNS. Local-network name resolution. Used by Apple Bonjour, Avahi. Port 5353. |
| **LLMNR** | Link-Local Multicast Name Resolution. Microsoft's mDNS-like thing. Port 5355. |
| **dig** | The standard DNS lookup tool. From BIND. |
| **host** | A simpler DNS lookup tool. |
| **nslookup** | The oldest DNS tool. Still around. |
| **kdig** | Knot's version of dig. |
| **drill** | ldns's version of dig. |
| **delv** | BIND's DNSSEC-validating lookup tool. |
| **whois** | Tool to query domain registration info. Not DNS itself, but the companion tool. |
| **resolvectl** | systemd-resolved's CLI for managing DNS. |
| **resolv.conf** | `/etc/resolv.conf`. Stub resolver config. |
| **nsswitch.conf** | `/etc/nsswitch.conf`. Tells the system in what order to consult name sources. |
| **/etc/hosts** | A static name → IP mapping file. Overrides DNS for listed names. |
| **DNS cookies** | RFC 7873. A small token in queries to make spoofing harder. |
| **Trust anchor** | A public key hardcoded in resolver software (the root KSK) used to bootstrap DNSSEC validation. |
| **Negative trust anchor** | A way to disable DNSSEC validation for a specific zone (for testing or broken zones). |
| **Zone walking** | Using NSEC chains to enumerate every name in a zone. NSEC3 prevents this (mostly). |

## Try This

Here are some safe experiments. None of these will break anything. They will teach you to feel comfortable poking at DNS.

### Experiment 1: See your default resolver

```
$ resolvectl status | grep "DNS Server"
```

Or look at the file:

```
$ cat /etc/resolv.conf
```

That tells you which DNS server your computer talks to by default.

### Experiment 2: Compare cold and warm lookups

Run a lookup, then immediately run it again. The second one will be much faster because it's cached.

```
$ dig +short example-not-cached-yet.com
$ dig +short example-not-cached-yet.com
```

The first might take 100ms, the second 1ms.

### Experiment 3: Walk the tree manually

Pick a TLD and a domain. Walk it yourself.

```
$ dig @a.root-servers.net . NS              # ask root for itself
$ dig @a.root-servers.net com NS            # ask root about .com
$ dig @a.gtld-servers.net google.com NS     # ask .com about google.com
$ dig @ns1.google.com google.com A          # ask google about google.com
```

You just did what `dig +trace` does, by hand.

### Experiment 4: Find your reverse DNS

What does the world see for your IP?

```
$ dig -x $(curl -s ifconfig.me)
```

Often nothing — most home users don't have reverse DNS. ISPs sometimes have generic names like `pool-12-34-56-78.dyn.example.net`.

### Experiment 5: Watch your DNS traffic

Run `tcpdump` and then make a few web requests in another window.

```
$ sudo tcpdump -i any -n udp port 53
```

You'll see your queries fly by. Press Ctrl-C to stop.

### Experiment 6: Test DNSSEC with a known-bad zone

`dnssec-failed.org` is a deliberately broken DNSSEC zone for testing.

```
$ dig dnssec-failed.org
;; status: SERVFAIL
```

If your resolver validates DNSSEC, this returns SERVFAIL. If it doesn't validate, this returns NOERROR with the answer. Try `dig @8.8.8.8 dnssec-failed.org` (Google validates) vs `dig @4.2.2.4 dnssec-failed.org` (Level3 historically did not).

### Experiment 7: Make your computer use a different DNS server

Edit `/etc/resolv.conf` (or `resolvectl dns` on systemd-resolved):

```
$ sudo resolvectl dns wlp3s0 1.1.1.1
$ resolvectl status
```

Try lookups, see the latency change, change it back.

### Experiment 8: Test all major resolvers

```
$ for r in 8.8.8.8 1.1.1.1 9.9.9.9 4.2.2.4 208.67.222.222; do echo "==$r=="; dig @$r +short example.com; done
```

All should return `93.184.216.34` (or whatever the current IP is). If one disagrees, that's interesting.

### Experiment 9: Try mDNS

If you have a printer or AirPlay device on your network:

```
$ dig -p 5353 -t PTR _airplay._tcp.local @224.0.0.251
```

You should see your AirPlay devices.

### Experiment 10: Watch the DNS cache fill

If you run a local resolver like Unbound:

```
$ sudo unbound-control flush_zone .
$ dig google.com
$ sudo unbound-control dump_cache | grep google
```

You'll see the records appear in the cache.

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs networking dns`** — the dense reference. Real syntax for every record type, every flag, every config file.
- **`cs detail networking/dns`** — algorithmic underpinnings. Tree walks, signing math, anycast routing.
- **`cs networking dig`** — every flag of dig. Reading dig output like a pro.
- **`cs networking coredns`** — the Kubernetes DNS server. Plugins, config, ops.
- **`cs networking doh-dot`** — encrypted DNS deep dive. DoH, DoT, DoQ, ECH.
- **`cs ramp-up ip-eli5`**, **`cs ramp-up udp-eli5`**, **`cs ramp-up tcp-eli5`**, **`cs ramp-up tls-eli5`** — the layers above and below DNS.

## See Also

- `networking/dns` — engineer-grade reference.
- `networking/dig` — full dig manual with every flag.
- `networking/coredns` — CoreDNS server config and plugins.
- `networking/doh-dot` — encrypted DNS protocols.
- `networking/dhcp` — how your machine learns its DNS server in the first place.
- `networking/dhcpv6` — IPv6 DHCP, including DNS option.
- `networking/ipv4` — what an IPv4 address actually is.
- `networking/ipv6` — what an IPv6 address actually is.
- `networking/tcp` — the protocol DNS sometimes falls back to.
- `networking/udp` — the default protocol for DNS queries.
- `networking/quic` — the modern transport that DoQ uses.
- `ramp-up/ip-eli5` — IP addresses in plain English.
- `ramp-up/udp-eli5` — UDP in plain English.
- `ramp-up/tcp-eli5` — TCP in plain English.
- `ramp-up/tls-eli5` — TLS in plain English.
- `ramp-up/icmp-eli5` — ICMP in plain English.
- `ramp-up/linux-kernel-eli5` — the kernel that runs all of this.

## References

- **RFC 1034** — Domain Names — Concepts and Facilities (1987). The conceptual foundation.
- **RFC 1035** — Domain Names — Implementation and Specification (1987). The wire format.
- **RFC 2782** — A DNS RR for specifying the location of services (DNS SRV) (2000).
- **RFC 6891** — Extension Mechanisms for DNS (EDNS(0)) (2013). Updated from RFC 2671 (1999).
- **RFC 7858** — Specification for DNS over Transport Layer Security (DoT) (2016).
- **RFC 8484** — DNS Queries over HTTPS (DoH) (2018).
- **RFC 9250** — DNS over Dedicated QUIC Connections (DoQ) (2022).
- **RFC 4033, 4034, 4035** — DNS Security Introduction and Requirements / Resource Records / Protocol Modifications (2005). The DNSSEC trio.
- **RFC 5155** — DNS Security (DNSSEC) Hashed Authenticated Denial of Existence (NSEC3) (2008).
- **RFC 7873** — Domain Name System (DNS) Cookies (2016).
- **RFC 8806** — Running a Root Server Local to a Resolver (2020). For privacy.
- **RFC 9460** — Service Binding and Parameter Specification via the DNS (SVCB and HTTPS Resource Records) (2023).
- **RFC 8482** — Providing Minimal-Sized Responses to DNS Queries That Have QTYPE=ANY (2019).
- **RFC 8305** — Happy Eyeballs Version 2: Better Connectivity Using Concurrency (2017). How clients pick between A and AAAA.
- **RFC 2136** — Dynamic Updates in the Domain Name System (DNS UPDATE) (1997).
- **`man dig`** — full dig manual.
- **`man named`** — BIND server manual.
- **`man unbound`** — Unbound resolver manual.
- **`man named.conf`** — BIND config syntax.
- **`man resolv.conf`** — stub resolver config syntax.
- **`man nsswitch.conf`** — name service switch config.
- **"DNS and BIND"** by Cricket Liu and Paul Albitz — the classic DNS book. Sixth edition is from 2024. Read this once and you'll never have to google DNS again.
- **bind9-utils package** — `dig`, `host`, `nsupdate`, `delv`. Install via your package manager.
- **knot-dnsutils** — `kdig`, `khost`. Modern alternatives.
- **ldnsutils / ldns-utils** — `drill`. Another modern toolkit.
- **isc.org** — Internet Systems Consortium. Maintainers of BIND. Lots of docs.
- **Knot Resolver docs** — knot-resolver.readthedocs.io. Excellent practical guide.
- **Unbound docs** — unbound.docs.nlnetlabs.nl. Detailed config reference.

Tip: every reference above can be read inside your terminal. Most are accessible via `man` or by `dig`-ing the local copies. The book references can be downloaded as PDFs and read in `zathura` or `less`. You really do not need to leave the terminal.

— End of ELI5 —

When this sheet feels boring (and it will, faster than you think), graduate to `cs networking dns` — the engineer-grade reference. It uses real names for everything: zones, records, flags, RFCs, configuration files. After that, `cs detail networking/dns` gives you the academic underpinning. By the time you have read both, you will be reading DNS responses without a flinch.

### One last thing before you go

Pick one command from the Hands-On section that you have not run yet. Run it right now. Read the output. Try to figure out what each part means, using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. DNS is real. It is happening on your computer, right now, every time anything connects to anything. The commands in this sheet let you peek at it.

Reading is good. Doing is better. Type the commands. Watch DNS respond.

You are now officially started on your DNS journey. Welcome.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one `dig` away. There is no Google search you need to do to start understanding DNS. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. DNS is happy to be poked at. Nothing on this sheet will break anything. Try things. Type commands. Read what comes back. The more you do, the more it all clicks into place.

— End of ELI5 — (really this time!)
