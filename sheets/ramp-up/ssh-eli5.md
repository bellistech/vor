# SSH — ELI5

> SSH is the secret handshake plus the sealed tube between your computer and a faraway computer. You prove who you are at the door, the two computers whisper a one-time password to each other, and from then on every keystroke and every byte goes through a tube that nobody outside can read.

## Prerequisites

You should have a vague idea what a "terminal" is (a black window where you type commands and the computer types back). If not, read `ramp-up/bash-eli5` and `ramp-up/linux-kernel-eli5` first. You should also have a vague idea what "the internet" is (a giant pile of computers connected by wires and radio waves). If not, read `ramp-up/tcp-eli5` and `ramp-up/ip-eli5`. None of those are required-required. You can get through this sheet cold. But the words "TCP," "port," and "DNS" come up, and if those don't ring any bells you might want to bookmark this sheet, read those, then come back.

If a word looks weird, scroll down to the **Vocabulary** table near the end. Every weird word in this sheet is in there with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back. We call that "output."

## What Even Is SSH?

### Imagine you are passing notes in class

Picture a classroom. You sit at the front. Your friend sits at the back. You want to send your friend a note. You can't get up and walk it over because the teacher is watching. So you scribble a note on a piece of paper, fold it up, and pass it to the kid next to you. They pass it to the next kid. And the next. Eventually the note gets to your friend at the back.

Here is the problem: every kid the note passes through can read it. Every single one. The kid next to you can unfold it and read it. The kid behind them can read it. Anyone in the chain can read it.

Now imagine you and your friend invented a secret code last week. Before class, you both memorized a tiny rule: "swap every letter for the letter three after it in the alphabet." Now you scribble your note in code. You pass it down the line. Every kid in between sees the note, but they just see "KHOOR" and shrug, because they don't know the code. Your friend at the back unfolds it, swaps every letter back three, and reads "HELLO."

That, in a nutshell, is what SSH does. SSH is the secret code between two computers that keeps every kid in the middle from reading the note.

The wires of the internet are a thousand kids in a row passing notes for you. Every router, every switch, every coffee shop Wi-Fi access point, every internet provider, every backbone link — they all see your note pass through. Without SSH, they can read every keystroke. With SSH, they see gibberish. Total gibberish. Math-grade gibberish that no kid in the chain can untangle, no matter how clever they are or how long they sit and stare at the note.

### Imagine a sealed pneumatic tube

Old buildings used to have these things called pneumatic tubes. You put a paper note inside a little plastic capsule. You stuck the capsule into a pipe in the wall. A burst of air blew the capsule through the pipes, all the way to another floor of the building, and the capsule popped out on the receiver's desk. The pipe was sealed. Nobody could see what was inside the capsule while it traveled. Nobody could pull capsules out of the pipe.

That is the SSH tube. Once SSH is set up between your computer and a faraway server, every byte of your conversation goes through a sealed pipe. You type a key on your keyboard. The key goes into a capsule. The capsule whooshes through the pipe, scrambled, totally invisible. It pops out on the other end. The server sees it. The server scribbles a reply, puts it in a capsule, sends it back. You see the reply on your screen.

To you it looks like you are typing right at the faraway computer. To anyone watching the wires, it looks like a stream of nonsense bytes that they cannot decode and cannot tamper with.

### Imagine a magic mailbox

Here is one more picture, because public-key cryptography is the trickiest idea in this whole sheet, and the magic mailbox is the easiest way to remember it.

Imagine you have a special mailbox. The mailbox has a little slot in the top, like a normal mailbox. Anybody can drop letters in. The mailbox is locked, though. There is a door on the front, and only your special key can open the door.

Now imagine you put copies of the mailbox everywhere. You put one on the corner of your block. You put one downtown. You put one at every coffee shop. You put one at the bus station. You give one to your grandma. You give one to your boss. The mailboxes are all identical and all belong to you. Anyone in the world who wants to send you a secret letter can walk up to any mailbox and drop a letter through the slot. The slot is one-way. Once a letter goes in, it locks inside.

You have one key. Only you. You take the key home. When you want to read your mail, you go to a mailbox, unlock the door with your key, take out the letters, and read them. Nobody else can read them. Even though the mailbox is in public. Even though anyone can drop a letter in. The only person who can take letters out is the person with the matching key.

Hold onto that picture. The mailbox is your **public key.** The matching key is your **private key.** When you do SSH with keys, you are basically posting your mailbox at the server, and the server is using your mailbox to verify that you really are you. We will get to exactly how in a few sections, but the magic mailbox is the picture you should keep in your head.

## Why SSH Replaced Telnet, rsh, and rlogin

Once upon a time, in the 1980s and the early 1990s, people did connect to faraway computers, but they did it with a tool called **telnet.** Telnet was great in one way: it worked. You typed `telnet some-server.com` and you were typing on the faraway server. Magic.

Telnet had one really, really, really bad problem. It sent every keystroke as plain text. No code. No scramble. No secret handshake. If you typed your password, your password went down the wires in plain English. Every kid in the chain saw it. Anybody who tapped any wire between you and the server could see your password and read every command you typed.

In the 1980s, when a few hundred universities were on the internet and everybody knew everybody, this was sort of okay. By the early 1990s, when companies started showing up, this was not okay at all. Eavesdropping on telnet was so easy that it became a whole hobby for hackers. You'd sit at a coffee shop, watch the network for ten minutes, and walk out with a stack of passwords for big company servers.

There were also `rsh` ("remote shell") and `rlogin` ("remote login"). Same family. Same problem. They sent everything in plain text and used a security model called "I trust this machine because it says its name is X," which a child could fake.

In 1995 a Finnish researcher named **Tatu Ylönen** got fed up. His university got attacked. Passwords got stolen off the wire. He sat down and wrote SSH, the **Secure Shell.** The first version came out in July 1995. Within a year, tens of thousands of people switched to it. Within a few years, telnet was dead for any real use.

SSH did three things that telnet didn't:

1. **Encryption.** Every byte goes through a sealed tube. No more eavesdropping.
2. **Authentication you can actually trust.** Not just "I say I'm Steve, please believe me." Real cryptographic proof.
3. **Integrity.** If somebody on the wire tries to flip a bit in your message, SSH notices and drops the connection. Nobody can secretly change what you typed in flight.

That's it. That is why SSH wiped telnet off the planet. Today if a system administrator catches you using telnet for anything but maybe testing a port, they will sit you down for a stern talk.

```
TIMELINE
========
1969 ----- ARPANET begins. Telnet is one of the first protocols.
1983 ----- BSD adds rsh and rlogin. "I trust your hostname" security.
1988 ----- The Morris worm spreads. First taste of how scary unsecured services are.
1995 Jul - Tatu Ylonen releases SSH-1. Free to use.
1995 Dec - SSH Communications Security founded. SSH-1 becomes commercial.
1996 ---- SSH-2 protocol designed. Cleaner crypto.
1999 ---- OpenBSD team forks the last free version. OpenSSH is born.
2006 ---- RFCs 4250-4256 standardize SSH-2.
2010s --- Telnet finally banned by most distros for default install.
2020s --- OpenSSH is on basically every Linux, BSD, macOS, and Windows.
```

## Public-Key Cryptography Quick Tour

Let me try to explain public-key crypto in the simplest way I can.

You have a pair of magic objects. We call them a **key pair.** One is the **public key.** One is the **private key.** They are made together, at the same time, with a math trick. They are forever linked. Anything one of them does, only the other one can undo. Like two halves of a friendship necklace.

You give the public key to anyone. You shout it from rooftops. You print it on T-shirts. You email it. You paste it into chat. You can put it in a billboard if you want. The public key is not a secret.

You keep the private key locked up. You never show it to anyone. You never email it. You never paste it. You don't even let it leave your laptop. If somebody steals your private key, they basically become you.

The math trick is this: if you encrypt a message with the public key, only the private key can decrypt it. And if you encrypt a message with the private key, only the public key can decrypt it.

That's the entire foundation. Two simple rules. Out of those two rules you get every cool thing in modern internet security.

For SSH, the rule we mostly use is: **the server stores the public key. You hold the private key. To prove you are you, the server sends you a challenge, you encrypt it with your private key, and the server decrypts it with the public key. If the answer matches, you're you.**

You don't have to remember the math. You just have to remember the magic mailbox. Public key = mailbox. Private key = the key to the mailbox. Hand out mailboxes everywhere, keep the key in your pocket.

```
+----------------------+         +----------------------+
|      YOU             |         |      SERVER          |
|                      |         |                      |
|   private key        |         |   public key         |
|   (in your pocket)   |         |   (on the server)    |
|                      |         |                      |
|   "Sign this nonce"  |<------- |   sends random nonce |
|                      |         |                      |
|   signs nonce        | ------->|   verifies signature |
|   with private key   |         |   with public key    |
|                      |         |                      |
|                      |  PASS!  |                      |
+----------------------+         +----------------------+
```

The arrow going right is your reply. The arrow going left is the server's challenge. The "nonce" is a random number the server invents fresh every time so attackers can't replay an old answer.

## Key Types: Which One Should I Use?

When you make an SSH key pair, you have to pick a flavor. There are several flavors. They all do the same job. Some are old. Some are new. Some are big. Some are small. Some are fast. Some are slow. Here is the menu, in plain English.

### RSA

The grandparent. RSA is the oldest public-key algorithm still in everyday use. It's named after three guys: **R**ivest, **S**hamir, **A**dleman. They wrote the paper in 1977. The math is based on multiplying two huge prime numbers together and the fact that nobody knows a fast way to un-multiply them.

RSA keys come in sizes. The size is in bits. More bits means bigger keys, slower math, but harder to crack.

- **RSA-1024.** Grandma key. Considered weak since around 2012. Some old systems still use it. Don't make new ones.
- **RSA-2048.** Standard for most of the 2010s. Still considered fine for general use today, but the smart kids have moved on.
- **RSA-3072.** Equivalent to a 128-bit symmetric key. Used in some compliance-driven environments.
- **RSA-4096.** Belt and suspenders. Big files, slow math, but very tough.

RSA works everywhere. Every SSH client, every SSH server, every router, every embedded gadget. If you absolutely don't know what to use and you have to talk to weird old hardware, RSA is the universal language.

### DSA (deprecated, do not use)

DSA was an old key type from the 1990s. It had a nasty bug: if your random-number generator was even a tiny bit broken, an attacker could recover your private key from a few signatures. OpenSSH 7.0 (2015) disabled DSA by default. Don't use DSA. If you see DSA, treat it like a banana from 1995.

### ECDSA

**E**lliptic **C**urve **D**igital **S**ignature **A**lgorithm. Same idea as DSA but on a different kind of math. Smaller keys for the same strength. ECDSA comes in three sizes named after the curves they use: **P-256**, **P-384**, and **P-521**.

ECDSA works fine. The only complaint is that the curves were standardized by NIST, and some people are paranoid that NIST cooked the curves to make them secretly breakable. There is no proof of this. But the Ed25519 designers made a different curve specifically to dodge that worry, and most people now prefer Ed25519 just to skip the argument.

### Ed25519 (the recommended one)

The cool kid. Ed25519 was designed by **Daniel J. Bernstein** and published in 2011. It uses elliptic curves but on a curve called **Curve25519** that was chosen specifically so nobody can accuse anyone of cooking the curve.

Ed25519 keys are tiny. About 256 bits. The public key fits on one line. The signatures are fast. The math is fast. The keys are easy to copy and paste. They are also extremely secure, equivalent to about RSA-3000 in strength.

OpenSSH added Ed25519 in version 6.5 (January 2014). Today it is the default in OpenSSH 8.0 and later. If you are starting fresh, **use Ed25519.**

### Ed448

Big sibling of Ed25519. About 448 bits. Slower than Ed25519, also extremely secure. Used in some very-high-security environments. You probably don't need it. Ed25519 is plenty.

### Hardware-backed: sk-ed25519 and sk-ecdsa

Since OpenSSH 8.2 (February 2020) you can make a key that lives on a hardware token like a YubiKey. The "sk" stands for "security key." When you generate the key, the actual private key never leaves the hardware. To use the key, you have to physically tap the YubiKey. This makes stealing the key very, very hard, because the thief would have to also steal your physical YubiKey.

If you can use a hardware key, you should. They are worth the money for any sensitive account.

### So which one?

Default answer for **new keys**: Ed25519.

```
$ ssh-keygen -t ed25519 -C "you@yourbox"
```

If you have to talk to ancient gear that doesn't speak Ed25519, use RSA-4096.

Avoid DSA, RSA-1024, and ECDSA-256 unless you know what you are doing.

## Generating Keys

Time to make a real key. Open a terminal and type this:

```
$ ssh-keygen -t ed25519 -C "you@yourbox"
```

The command says: make me a new key, type "ed25519," and put a comment "you@yourbox" at the end so I can remember which key this is.

The command will ask you a few things:

```
Generating public/private ed25519 key pair.
Enter file in which to save the key (/home/you/.ssh/id_ed25519):
```

Just press Enter. The default place is fine.

```
Enter passphrase (empty for no passphrase):
```

This is a question worth thinking about. A **passphrase** is a password on top of your key. If somebody steals your private key file, they still can't use it without the passphrase. If you forget the passphrase, you cannot recover it. There is no "forgot password" link. The key is gone forever.

For a long time, every safety guide said "always use a passphrase." Today, with hardware keys and SSH agents, the picture is more complicated. For now, type any short passphrase you can remember. We will talk about how to make passphrases painless later with the SSH agent.

```
Enter same passphrase again:
```

Type it again.

```
Your identification has been saved in /home/you/.ssh/id_ed25519
Your public key has been saved in /home/you/.ssh/id_ed25519.pub
The key fingerprint is:
SHA256:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx you@yourbox
The key's randomart image is:
+--[ED25519 256]--+
|        ..oo.    |
|       . oo.+    |
|        oo.* +   |
|       . *o.+ .  |
|        S+oo +   |
|        .+. =    |
|        .o + .   |
|         o.o     |
|         .=o     |
+----[SHA256]-----+
```

Done. You now have a key pair.

The two files are:

- `~/.ssh/id_ed25519` — the **private key.** Never share. Never email. Never copy to another machine unless you absolutely have to.
- `~/.ssh/id_ed25519.pub` — the **public key.** Safe to share. Paste it into GitHub, paste it into your work directory of trust, paste it into any server's `authorized_keys` file.

If you want a different filename (because you keep multiple keys for different jobs), you can pass `-f`:

```
$ ssh-keygen -t ed25519 -f ~/.ssh/work_ed25519 -C "you@work"
```

Now the files are `~/.ssh/work_ed25519` and `~/.ssh/work_ed25519.pub`. Same idea.

## The .ssh Directory and File Permissions

This is one of the most common places people stumble. SSH cares **a lot** about permissions on its files. If your `.ssh` directory or your private key is readable by other users on the system, SSH will refuse to work. It does this on purpose to protect you.

Here are the magic numbers:

```
~/.ssh/                    drwx------    700    you only
~/.ssh/id_ed25519          -rw-------    600    you only, read+write
~/.ssh/id_ed25519.pub      -rw-r--r--    644    everyone can read
~/.ssh/authorized_keys     -rw-------    600    you only
~/.ssh/known_hosts         -rw-r--r--    644    everyone can read
~/.ssh/config              -rw-------    600    you only
```

If you mess up, fix it like this:

```
$ chmod 700 ~/.ssh
$ chmod 600 ~/.ssh/id_ed25519 ~/.ssh/authorized_keys ~/.ssh/config
$ chmod 644 ~/.ssh/id_ed25519.pub ~/.ssh/known_hosts
```

If you skip this and try to log in, SSH will say things like:

```
Permissions 0644 for '/home/you/.ssh/id_ed25519' are too open.
It is required that your private key files are NOT accessible by others.
This private key will be ignored.
```

That's SSH protecting you. Tighten the permissions and try again.

## authorized_keys vs. known_hosts

Two files in `~/.ssh/` look kind of similar but do completely opposite jobs. People mix them up all the time. Let me draw it out.

```
+------------------------------------------------+
|                YOUR LAPTOP                     |
|                                                |
|   ~/.ssh/id_ed25519           (your priv key)  |
|   ~/.ssh/id_ed25519.pub       (your pub key)   |
|   ~/.ssh/known_hosts          <-- "I trust     |
|                                    these       |
|                                    SERVERS"    |
+------------------------------------------------+
                       |
                       | ssh user@server.com
                       v
+------------------------------------------------+
|                THE SERVER                      |
|                                                |
|   ~user/.ssh/authorized_keys  <-- "I trust     |
|                                    these       |
|                                    USERS"      |
+------------------------------------------------+
```

`authorized_keys` lives on the server, in the user's home directory you log in as. Each line is a public key. If your public key is in there, the server lets you in.

`known_hosts` lives on your laptop. Each line is a server's public key. When you connect somewhere, SSH looks up the server in `known_hosts`. If the server's key matches, great. If it doesn't match, SSH shouts at you.

A good way to remember: `authorized_keys` = "users I let in." `known_hosts` = "servers I've met before."

## The First Connection Dance

The very first time you connect to a server, something a little strange happens. Watch this:

```
$ ssh you@new-server.example.com
The authenticity of host 'new-server.example.com (203.0.113.7)' can't be established.
ED25519 key fingerprint is SHA256:abc123...long-thing...
This key is not known by any other names.
Are you sure you want to continue connecting (yes/no/[fingerprint])?
```

What happened? SSH asked the server for its host key. The server gave it. SSH looked in `known_hosts` for that server. It wasn't there. So SSH stopped and asked you, the human, "Hey, this server claims this is its public key. Do you believe it?"

This is called **TOFU**, which stands for **Trust On First Use.** You trust the server the first time. After that, SSH remembers, and if the server ever shows a different key later, SSH freaks out.

Ideally, before you say yes, you should verify the fingerprint by some other channel. Did your sysadmin email it to you? Is it printed in your company's wiki? Did the server's hosting provider publish it on their website? Compare the fingerprint they gave you with the one SSH is showing. If they match, type `yes` and continue.

In real life, most people just type `yes` and hope. This is fine for low-stakes stuff. For production servers and bank stuff, please verify the fingerprint.

After you say yes:

```
Warning: Permanently added 'new-server.example.com' (ED25519) to the list of known hosts.
you@new-server.example.com's password:
```

Now `new-server.example.com` is in your `known_hosts`. From now on, SSH will silently check the server's key against the saved one every single time. If they ever disagree:

```
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that a host key has just been changed.
The fingerprint for the ED25519 key sent by the remote host is
SHA256:differentkey...
Please contact your system administrator.
Add correct host key in /home/you/.ssh/known_hosts to get rid of this message.
Offending ED25519 key in /home/you/.ssh/known_hosts:42
Host key verification failed.
```

Big scary box. Read it slowly. SSH is telling you that the server's key changed. There are two reasons this happens:

1. **The server got rebuilt.** Sysadmin reinstalled the OS. The host key got regenerated. This is benign. Update your `known_hosts`.
2. **Somebody is in the middle.** An attacker is intercepting your connection and pretending to be the server. This is bad.

Always assume case #2 until you can confirm case #1 with a human. Don't just type yes blindly.

To clean up after a benign rebuild:

```
$ ssh-keygen -R new-server.example.com
```

That removes the old line from `known_hosts`. Then you can connect again, do the TOFU dance, and accept the new key.

## The Authentication Methods

When you knock on an SSH server's door, the server can accept several different ways for you to prove who you are. Here is the menu.

### password

You type a password. The server checks it against a saved hash. If it matches, you're in. Easy. Lots of people still use this. It has the same problem as any password: it can be guessed, leaked, phished, reused on other sites.

### publickey

The good way. You have a key pair. Your public key is in the server's `authorized_keys`. The server sends you a random challenge. You sign the challenge with your private key. The server verifies the signature with your public key. If it checks out, you're in. No password ever crosses the wire. Even if somebody is watching the wire forever, they can't fake it.

This is the method we have been talking about. It is the recommended way. It is what every "configure SSH for production" guide tells you to use.

### keyboard-interactive

The server can send you a question. You type an answer. The server can send another question. You type another answer. This is how SSH supports things like one-time codes from an authenticator app. The server says "give me your code from the app," you type it, the server checks.

Often used together with password or publickey for **two-factor authentication.**

### hostbased

A really old method where the SSH server trusts your **whole machine** instead of you specifically. If the request comes from `trusted-host.example.com` and it claims you are user `you`, the server just believes it. This is rarely used today because it depends on machines trusting each other in a way that doesn't scale on the modern internet.

### gssapi-with-mic

The Kerberos method. If your company has a Kerberos system (common in big enterprises and universities), SSH can use a Kerberos ticket as proof of who you are. You log into the company's system once at the start of the day, get a ticket, and SSH uses the ticket to authenticate you to every server until the ticket expires.

### publickey + cert from a CA

A fancy version of publickey. Instead of putting every user's public key on every server, the server trusts a single **Certificate Authority (CA)**. The CA signs each user's public key with its own key. The server lets in any user whose key is signed by the CA. We have a whole section on this later.

## SSH Agent and ssh-add

If you used a passphrase on your key, every time you SSH somewhere you have to type the passphrase. That gets old fast. There is a fix. It's called the **SSH agent.**

The SSH agent is a little background program that holds your keys for you. You unlock your key once, hand it to the agent, and from then on the agent answers every SSH server's challenge for you. You stop typing your passphrase. The agent dies when you log out (or you can shut it down manually), and your key goes with it.

```
+----------+        +-------------+
|  YOU     |        |  SSH AGENT  |
| (typing) |        | (in memory) |
|          | ssh    |             |
|          | adds   |             |
|          +------->| holds key   |
|          |        |             |
|          |        |             |
|          | ssh me@server        |
|          +--------------+       |
|          |              |       |
|          |        +-----v-------+
|          |        | SSH CLIENT  |
|          |        |             |
|          |        | "agent, can |
|          |        |  you sign?" |
|          |        |             |
|          |        +-----+-------+
|          |              |
+----------+              v server
```

To start using the agent on most modern systems:

```
$ eval "$(ssh-agent -s)"
Agent pid 12345
$ ssh-add ~/.ssh/id_ed25519
Enter passphrase for /home/you/.ssh/id_ed25519:
Identity added: /home/you/.ssh/id_ed25519
```

Now SSH won't ask for the passphrase anymore as long as the agent is running.

To list the keys the agent currently has:

```
$ ssh-add -L
ssh-ed25519 AAAAC3...long-stuff... you@yourbox
```

To remove one key:

```
$ ssh-add -d ~/.ssh/id_ed25519
```

To remove all keys:

```
$ ssh-add -D
```

To add a key with a timeout (the agent forgets it after, say, an hour):

```
$ ssh-add -t 3600 ~/.ssh/id_ed25519
```

This is good practice on shared machines. Keys auto-expire so they're not sitting in memory for days.

On macOS, the system has an SSH agent built in that hooks into Keychain. On most Linux desktops, the agent is started automatically when you log in. Windows 10+ and Windows 11 include OpenSSH and a built-in `ssh-agent` Windows service.

## SSH Config: ~/.ssh/config

Typing `ssh -i ~/.ssh/work_ed25519 -p 2222 myuser@server.example.com` ten times a day gets old. SSH has a config file that lets you give shortcuts to all that.

The file is `~/.ssh/config`. Make it if it doesn't exist. Set permissions to 600.

A simple config block looks like:

```
Host work
    HostName server.example.com
    User myuser
    Port 2222
    IdentityFile ~/.ssh/work_ed25519
```

Now instead of typing the long form, you type:

```
$ ssh work
```

That's it. Same connection. Way less typing.

You can have many blocks. You can use wildcards. You can use globs. Here is a more useful example:

```
# All work servers
Host work-*
    User myuser
    IdentityFile ~/.ssh/work_ed25519
    IdentitiesOnly yes

# Specific work server
Host work-db
    HostName db.work.example.com
    Port 2222

Host work-api
    HostName api.work.example.com
    Port 22

# Personal
Host home
    HostName home.mydomain.com
    User stevie
    IdentityFile ~/.ssh/id_ed25519
    Port 22

# Defaults for everything else
Host *
    ServerAliveInterval 60
    ServerAliveCountMax 3
    AddKeysToAgent yes
    IdentitiesOnly yes
```

The patterns work top-down. The first matching block wins, and earlier blocks override later ones. The `Host *` block at the bottom catches everything that didn't match earlier.

### Common config options

- `HostName` — the actual address to connect to.
- `User` — the username on the remote.
- `Port` — TCP port. Default 22.
- `IdentityFile` — which private key to use.
- `IdentitiesOnly yes` — only try the listed key. Without this, SSH tries every key in your agent, which can hit `Too many authentication failures`.
- `ForwardAgent yes` — forward your local agent to the remote (use with care, see below).
- `ServerAliveInterval N` — every N seconds, send a keep-alive ping so the connection doesn't die behind a NAT.
- `ServerAliveCountMax N` — give up after N missed pings.
- `AddKeysToAgent yes` — auto-add a key to the agent the first time you use it.
- `ControlMaster auto`, `ControlPath ...`, `ControlPersist N` — connection multiplexing (see below).
- `ProxyJump bastion` — route through a jump host.
- `ProxyCommand ...` — even fancier routing through arbitrary commands.

### ProxyJump (jumping through a bastion)

Many companies put their servers behind a "bastion" or "jump host." You can't reach the inside servers directly. You have to SSH to the bastion first, then SSH from the bastion to the inside server. SSH has a built-in shortcut for this.

```
Host bastion
    HostName bastion.example.com
    User myuser
    IdentityFile ~/.ssh/id_ed25519

Host inside-*
    User myuser
    ProxyJump bastion
    IdentityFile ~/.ssh/id_ed25519
```

Now `ssh inside-db` connects to `bastion`, then through `bastion` to `inside-db`, all in one command.

ASCII picture:

```
+--------+   +---------+   +---------+
| YOU    |-->| BASTION |-->| inside  |
|        |   |         |   | server  |
+--------+   +---------+   +---------+
```

You can also do it on the command line without a config:

```
$ ssh -J myuser@bastion.example.com myuser@inside-db
```

You can chain multiple jumps with commas:

```
$ ssh -J jump1,jump2,jump3 finaltarget
```

### ControlMaster (connection multiplexing)

Setting up an SSH connection takes a moment. The handshake, the key exchange, all that math. If you SSH to the same host ten times a minute, that adds up.

ControlMaster fixes this. The first connection you open becomes a "master." Subsequent connections to the same host hop into a tunnel that is already open. Way faster.

```
Host *
    ControlMaster auto
    ControlPath ~/.ssh/cm-%r@%h:%p
    ControlPersist 600
```

`%r` is the remote user, `%h` is the host, `%p` is the port. `ControlPersist 600` means "keep the master connection alive for 600 seconds (10 minutes) after the last child closes." This way, if you reconnect within 10 minutes, you skip the whole handshake.

ASCII picture:

```
First ssh:

+-------+    handshake    +-------+
|  YOU  |---------------->|SERVER |
|       |<================|       |   <-- master tunnel established
+-------+                 +-------+

Second ssh (within ControlPersist):

+-------+      hop in!    +-------+
|  YOU  |---------------->|SERVER |
|       |<----------------|       |   <-- reuses master, no handshake
+-------+                 +-------+
```

If a `ControlMaster` socket gets confused, you can stop it with:

```
$ ssh -O exit work
```

Or you can delete the socket file directly: `rm ~/.ssh/cm-*`.

## Port Forwarding: -L, -R, -D

This is where SSH stops being just a remote shell and starts being a Swiss army knife for networks. SSH can carry arbitrary TCP traffic through its tunnel. There are three flavors of forwarding.

### Local forwarding (-L)

You make a port on your local machine that, when anything connects to it, the connection is tunneled through SSH to a destination on the remote side.

Use case: there is a web server on the remote machine that listens only on `localhost` (so only the remote machine itself can see it). You want to view that web page from your laptop.

```
$ ssh -L 8080:localhost:80 you@remote
```

Read this as: "make a port 8080 on my laptop. When something connects to it, tunnel through to the remote, then connect to `localhost:80` from the remote's point of view."

Now in your laptop's browser, go to `http://localhost:8080/`. You see the remote's web page, even though that page was never directly exposed to the internet.

```
+--------+        SSH tunnel        +--------+
| LAPTOP |==========================|REMOTE  |
|  :8080 |                          | :80    |
|   ^    |                          |   ^    |
|   |    |                          |   |    |
| browser|                          | nginx  |
+--------+                          +--------+
```

### Remote forwarding (-R)

The opposite direction. You make a port on the **remote** machine that, when anything connects to it, the connection is tunneled back through SSH to a destination on **your** side.

Use case: you are running a web server on your laptop on port 9000, and you want to expose it through a public server.

```
$ ssh -R 9000:localhost:9000 you@remote
```

Read this as: "make a port 9000 on the remote. When something connects to it, tunnel through to me and connect to my `localhost:9000`."

Now `http://remote.example.com:9000/` shows your laptop's web server.

```
+--------+        SSH tunnel        +--------+
| LAPTOP |==========================|REMOTE  |
|  :9000 |                          | :9000  |
|   ^    |                          |   ^    |
|   |    |                          |   |    |
| your   |                          |  the   |
| server |                          | world  |
+--------+                          +--------+
```

By default the remote's port 9000 only accepts connections from the remote's localhost. If you want the world to reach it, you need `GatewayPorts yes` in the remote's `sshd_config`.

### Dynamic forwarding (-D, SOCKS proxy)

The really fancy one. You make a SOCKS proxy on your local machine. Any program you point at the SOCKS proxy will tunnel through SSH and come out on the remote, with the remote making the actual outbound connection.

```
$ ssh -D 1080 you@remote
```

Now configure your browser to use SOCKS5 proxy at `localhost:1080`. Every web page you load is fetched by `remote`, not your laptop. Useful for getting around weird local network restrictions, or for safe browsing on a hotel Wi-Fi, or for accessing internal company sites that only let you in if you're connecting from inside the office.

```
+--------+    SSH SOCKS proxy    +--------+      +--------+
| LAPTOP |======================>|REMOTE  |----->|TARGET  |
|  :1080 |                       |        |      |        |
|   ^    |                       |        |      |        |
|   |    |                       |        |      |        |
| browser|                       |        |      |        |
+--------+                       +--------+      +--------+
```

## SSH Tunneling Use Cases

A bunch of patterns that come up over and over:

- **Reach a database that only listens on localhost.** `ssh -L 5432:localhost:5432 user@db-server`. Now `psql -h localhost` from your laptop talks to the remote's Postgres.
- **Browse internal-only websites.** `ssh -D 1080 user@office`. Set browser SOCKS proxy. Done.
- **Expose a local dev server through a public box.** `ssh -R 80:localhost:3000 user@public-server`. Be sure `GatewayPorts yes` is on, and be careful about who can reach it.
- **Get a stable IP for a server behind NAT.** `ssh -R 2222:localhost:22 user@public-server`. From the public server, `ssh -p 2222 localhost` reaches the NAT'd box.
- **Make X11 work over a long link.** `ssh -X user@server` then run `xeyes` or whatever. The window appears on your laptop.

## SCP and SFTP

SSH is mostly for shells, but the same protocol is used to move files.

### scp

The simple way. `scp` works just like `cp` but with `host:` prefixes for remote paths.

Local to remote:

```
$ scp myfile.txt user@server:/path/to/dest/
```

Remote to local:

```
$ scp user@server:/path/to/file.txt .
```

Whole directory:

```
$ scp -r mydir/ user@server:/path/to/
```

Recursive flag is `-r`. Note that as of OpenSSH 9.0 (April 2022), `scp` is being phased out as the default protocol for the `scp` command. The same `scp` command still exists, but under the hood it now speaks SFTP. Old servers that only speak the legacy SCP protocol can be reached with `scp -O ...`.

### sftp

The interactive version. `sftp` gives you a shell-like prompt where you can `ls`, `cd`, `get`, `put`, `mkdir`, etc.

```
$ sftp user@server
Connected to server.
sftp> ls
sftp> cd /tmp
sftp> put localfile
sftp> get remotefile
sftp> bye
```

You can also run sftp non-interactively with a script of commands. Useful for automation.

## SSHFS: Mount a Remote Directory

SSHFS lets you mount a remote directory as if it were a local folder. Works on Linux and macOS (with extra software).

```
$ sshfs user@server:/path/to/dir /local/mountpoint
```

Now `/local/mountpoint` shows the contents of `/path/to/dir` on the server. You can read, write, edit, delete. All of it tunnels through SSH.

To unmount on Linux:

```
$ fusermount -u /local/mountpoint
```

On macOS:

```
$ umount /local/mountpoint
```

SSHFS is great for editing remote config files in your local editor. Just remember it's slower than a local filesystem because every operation is a round trip over SSH.

## rsync over SSH

`rsync` is a tool for syncing files between two places. It is famously efficient because it only sends the differences between files, not the whole files.

Over SSH:

```
$ rsync -avz -e ssh src/ user@server:dst/
```

Flags:

- `-a` — archive mode (recursive, preserve perms, times, etc.).
- `-v` — verbose.
- `-z` — compress in transit.
- `-e ssh` — use SSH as the transport (this is the default in modern rsync, but spelling it out doesn't hurt).

The trailing slash on `src/` matters: `src/` means "the contents of src," `src` (no slash) means "the directory src itself."

For a one-time backup that resumes if interrupted:

```
$ rsync -avz --partial --progress src/ user@server:/backup/dst/
```

For mirroring (delete files on the destination that aren't on the source):

```
$ rsync -avz --delete src/ user@server:/backup/dst/
```

`--delete` is dangerous. Read it five times before pressing Enter.

## SSH Certificates: For Fleets

Imagine you have a hundred servers. Imagine you have ten engineers. If you do user-keys-in-authorized_keys, every server has to know every engineer's public key. Adding a new engineer means updating a hundred servers. Removing an engineer who quit means the same. People forget. People leave keys lying around. It's a mess.

Certificates fix this. Instead of putting every user's key in every `authorized_keys`, you put **one** thing in every server: the CA's public key. The CA is a special key pair that you create for your fleet. From now on, when someone joins, you sign their public key with the CA. The signed thing is a **certificate.** When they SSH, they show their cert. The server checks the signature against the CA. If it's valid, they're in.

Make the CA:

```
$ ssh-keygen -f ~/.ssh/my_ca -t ed25519 -C "fleet CA"
```

Sign a user's public key:

```
$ ssh-keygen -s ~/.ssh/my_ca -I "stevie@bellis.tech" -n stevie -V +52w stevie_id_ed25519.pub
```

What that says: sign with the CA private key (`-s`), put the identity "stevie@bellis.tech" in the cert (`-I`), allow login as user `stevie` (`-n`), make it valid for 52 weeks (`-V`).

Output: `stevie_id_ed25519-cert.pub`. Stevie now has both their normal `stevie_id_ed25519` (private) and their cert. SSH automatically presents the cert when it sees the matching private key.

On every server, in `/etc/ssh/sshd_config`:

```
TrustedUserCAKeys /etc/ssh/my_ca.pub
```

Where `/etc/ssh/my_ca.pub` is the **public** half of your CA. (Never put the CA private key on a server.)

Now any user with a valid cert from your CA can log in. Add an engineer? Sign their key. Remove one? Their cert expires next week, or you publish a **KRL** (Key Revocation List) listing the cert serial numbers to deny.

ASCII picture:

```
+----------+
|  ADMIN   |
|  CA priv | <-- locked away in a safe place
+----+-----+
     | sign user keys
     v
+----------+      logs in       +-----------+
| USER     | -----------------> | SERVER    |
| user key |                    | trusts CA |
| + cert   |                    | pub only  |
+----------+                    +-----------+
```

You can also issue **host certificates**, where the CA signs each server's host key. Then clients trust the CA instead of TOFU-ing every server. No more `ssh-keygen -R` after a rebuild. The new host key is signed by the CA, the client recognizes the CA, all is well.

## SSH Hardening

Out of the box, OpenSSH on most distros is pretty safe. But there are still knobs to tighten. Look at `/etc/ssh/sshd_config`. Here are the headlines:

```
# Don't let root log in directly.
PermitRootLogin no

# Don't let people log in with passwords.
PasswordAuthentication no

# Block empty passwords too.
PermitEmptyPasswords no

# Only allow specific users.
AllowUsers stevie alice bob

# Or specific groups.
AllowGroups sshusers

# Limit how many bad password tries per connection.
MaxAuthTries 3

# Limit how many half-open connections.
MaxStartups 10:30:60

# Disable old, weak protocols.
Protocol 2
```

`Protocol 2` is a no-op on modern OpenSSH because SSH-1 was removed years ago, but writing it makes audit reports happy.

Disable agent forwarding by default:

```
AllowAgentForwarding no
```

Restrict X11 forwarding:

```
X11Forwarding no
```

Set strong key exchange and ciphers:

```
KexAlgorithms curve25519-sha256,curve25519-sha256@libssh.org
HostKeyAlgorithms ssh-ed25519,ssh-ed25519-cert-v01@openssh.com
Ciphers chacha20-poly1305@openssh.com,aes256-gcm@openssh.com,aes128-gcm@openssh.com
MACs hmac-sha2-256-etm@openssh.com,hmac-sha2-512-etm@openssh.com
```

After editing, always check the file is valid before reloading:

```
$ sudo sshd -t
$ sudo systemctl reload sshd
```

Run a tool like **fail2ban** or **denyhosts** to block IPs that hammer your SSH port with bad logins.

```
$ sudo apt install fail2ban   # Debian/Ubuntu
```

There is a long-running debate about **changing port 22 to a high port like 2222 or 22022.** Pros: way fewer drive-by automated bots. Cons: it's "security by obscurity"; doesn't actually make a real attacker work harder. Most people now do both: change the port to cut log noise, *and* harden everything else.

## Common Errors (Verbatim)

Here are the exact error strings you will see, what they mean, and how to fix.

### Permission denied (publickey)

```
$ ssh user@host
user@host: Permission denied (publickey).
```

Your key is not being accepted. Things to check:

- Is your public key actually in the server's `~/.ssh/authorized_keys`?
- Are the permissions right on `~/.ssh` and `authorized_keys`? (700 and 600)
- Are you offering the right key? (`ssh -i ~/.ssh/right_key user@host`)
- Is `IdentitiesOnly yes` set? Otherwise SSH may try other keys first.
- Run `ssh -v` to see what's happening. The verbose output tells you which keys are being offered.
- On the server, look at `/var/log/auth.log` (Debian/Ubuntu) or `/var/log/secure` (RHEL/CentOS). It often says exactly why the key was refused (perms, missing, wrong user, etc.).

### Host key verification failed

```
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
...
Host key verification failed.
```

The server's key changed. Either it was rebuilt (benign) or somebody is in the middle (bad). Verify out-of-band, then:

```
$ ssh-keygen -R hostname
```

To clear the bad entry. Then reconnect and TOFU the new key (after verifying it).

### Connection refused

```
$ ssh user@host
ssh: connect to host host port 22: Connection refused
```

The host is up and reachable but nothing is listening on port 22. Did sshd crash? Did somebody change the port? Try `ssh -p 22022 user@host` if you suspect a non-default port. Try `nc -vz host 22` to test the port directly.

### Connection timed out

```
$ ssh user@host
ssh: connect to host host port 22: Connection timed out
```

Packets aren't getting through at all. The host could be down. There could be a firewall blocking your IP. The DNS could be wrong. Run `ping host`. Run `traceroute host`. Run `nc -vz host 22`.

### kex_exchange_identification: read: Connection reset by peer

```
$ ssh user@host
kex_exchange_identification: read: Connection reset by peer
Connection reset by peer
```

The server actively dropped your connection during the very first key exchange. Common causes: fail2ban already banned your IP, an intrusion-detection system reset the connection, the server has `MaxStartups` exceeded, a load balancer is misbehaving. Wait a few minutes and retry. Check the server's `/var/log/auth.log` for clues.

### Too many authentication failures

```
$ ssh user@host
Received disconnect from host port 22:2: Too many authentication failures
```

Your SSH client is offering more keys than the server allows. Often happens when your agent has many keys and `IdentitiesOnly` isn't set. Fix:

```
$ ssh -o IdentitiesOnly=yes -i ~/.ssh/specific_key user@host
```

Or in `~/.ssh/config`, add `IdentitiesOnly yes` to your Host blocks.

### sign_and_send_pubkey: signing failed for ECDSA

```
sign_and_send_pubkey: signing failed for ECDSA
```

Often means the agent has the public key but lost the private key, or the FIDO2 device wasn't tapped. Re-add the key with `ssh-add`. If using a hardware key, tap it.

### subsystem request failed on channel 0

```
subsystem request failed on channel 0
```

Common when `sftp` cannot start because the server doesn't have an SFTP subsystem configured. Look in `sshd_config` for a `Subsystem sftp ...` line.

### channel 0: open failed: administratively prohibited

```
channel 0: open failed: administratively prohibited: open failed
```

You tried to use port forwarding and the server has it disabled. Look in `sshd_config` for `AllowTcpForwarding`, `PermitOpen`, or `GatewayPorts`. Talk to the admin.

### ControlMaster connection has not been activated

```
ssh: connect to host work port 22: Connection refused
ControlSocket /home/you/.ssh/cm-...:.lock already exists, disabling multiplexing
```

A stale ControlMaster socket. Clean it up:

```
$ ssh -O exit work
$ rm ~/.ssh/cm-*
```

### Could not chdir to home directory

```
Could not chdir to home directory /home/user: No such file or directory
```

The login worked but your home directory doesn't exist (or has wrong permissions). Common with new accounts that didn't get a home dir created. Create it:

```
$ sudo mkhomedir_helper user
```

Or:

```
$ sudo mkdir -p /home/user
$ sudo chown user:user /home/user
$ sudo chmod 700 /home/user
```

## Hands-On

A pile of commands to try. Many will require you to have a server you can SSH to. If you don't have one, install OpenSSH server on your local machine and SSH to `localhost`.

```
# Generate a fresh ed25519 key
$ ssh-keygen -t ed25519 -C "you@yourbox"

# Generate an RSA-4096 key (for old gear)
$ ssh-keygen -t rsa -b 4096 -C "you@yourbox"

# Generate a key with a custom filename
$ ssh-keygen -t ed25519 -f ~/.ssh/work_ed25519 -C "you@work"

# Copy your public key to a server's authorized_keys
$ ssh-copy-id user@host

# Show your private key's fingerprint
$ ssh-keygen -lf ~/.ssh/id_ed25519

# Show your public key (re-derive from private)
$ ssh-keygen -y -f ~/.ssh/id_ed25519

# Connect with verbose output (for debugging)
$ ssh -v user@host

# Connect with EXTRA verbose output (for serious debugging)
$ ssh -vvv user@host

# Connect on a non-default port
$ ssh -p 2222 user@host

# Connect with a specific key
$ ssh -i ~/.ssh/work_ed25519 user@host

# Connect through a jump host
$ ssh -J jump@bastion target

# Local port forwarding
$ ssh -L 8080:localhost:80 user@host

# Remote port forwarding
$ ssh -R 9000:localhost:9000 user@host

# Dynamic SOCKS proxy
$ ssh -D 1080 user@host

# Background tunnel (no shell, run in background)
$ ssh -N -f -L 8080:localhost:80 user@host

# Force a TTY (for sudo over SSH)
$ ssh -t user@host 'sudo something'

# Quietly run one command
$ ssh user@host 'uptime'

# Run a script local-side on the remote
$ ssh user@host 'bash -s' < local-script.sh

# Start an SSH agent in a fresh shell
$ ssh-agent bash

# Start an agent and add a key
$ eval "$(ssh-agent -s)"
$ ssh-add ~/.ssh/id_ed25519

# List keys held by the agent
$ ssh-add -L

# Remove one key from the agent
$ ssh-add -d ~/.ssh/id_ed25519

# Remove all keys from the agent
$ ssh-add -D

# Add a key with a 1-hour timeout
$ ssh-add -t 3600 ~/.ssh/id_ed25519

# Look up a host's public key without connecting
$ ssh-keyscan host >> ~/.ssh/known_hosts

# Look up a host's ed25519 key only
$ ssh-keyscan -t ed25519 host

# Find a host in your known_hosts
$ ssh-keygen -F host

# Remove a host from your known_hosts
$ ssh-keygen -R host

# Sign a user public key with your CA
$ ssh-keygen -s ca_key -I user_id -n user1,user2 user_key.pub

# Sign a host key with your CA
$ ssh-keygen -s ca_key -I host_id -n host.example.com -h host_key.pub

# Look at a certificate
$ ssh-keygen -L -f user_key-cert.pub

# Test an SSH config without connecting
$ ssh -G hostname

# SCP a file up
$ scp file.txt user@host:/path/

# SCP a directory recursively
$ scp -r mydir/ user@host:/path/

# SCP a file down
$ scp user@host:/path/file.txt .

# SFTP into a host
$ sftp user@host

# rsync over SSH
$ rsync -avz src/ user@host:dst/

# rsync with progress and ability to resume
$ rsync -avz --partial --progress src/ user@host:dst/

# Mount a remote directory locally with sshfs
$ sshfs user@host:/path /mnt

# Unmount sshfs (Linux)
$ fusermount -u /mnt

# Test a server's SSH config syntax
$ sudo sshd -t

# See active SSH sessions
$ who
$ w

# See login history
$ last

# Generate moduli for diffie-hellman-group-exchange
$ ssh-keygen -M generate -O bits=2048 moduli-2048.candidates

# Use mosh as an alternative for flaky networks
$ mosh user@host

# Use autossh for connections that auto-reconnect
$ autossh -M 0 -o "ServerAliveInterval 30" user@host

# Disable host key checking for one connection (risky, only for testing)
$ ssh -o StrictHostKeyChecking=no user@host

# Use a bastion as a one-off jump
$ ssh -o ProxyJump=bastion target

# Open a tunnel for a database
$ ssh -L 5432:localhost:5432 user@db-server

# Force IPv4
$ ssh -4 user@host

# Force IPv6
$ ssh -6 user@host
```

## Common Confusions

### RSA vs Ed25519

People ask "should I use RSA-4096 or Ed25519?" Answer: Ed25519 unless you absolutely have to use RSA. Ed25519 is smaller, faster, and at least as secure. RSA-4096 is fine for compatibility with old gear. Ed25519 is what you want for everything new.

### Passphrase vs passwordless key

A passphrase encrypts your private key file. Without the passphrase, the file is useless. The whole point is "if my laptop is stolen, the thief can't use my key." A passwordless key is just a file. Anybody who copies the file can pretend to be you forever.

In practice, with `ssh-agent` (which holds the unlocked key in memory until you log out), the inconvenience of a passphrase is tiny: you type it once when adding the key. So: passphrase. Always passphrase. Unless it's a key for unattended automation, in which case use a hardware-backed key or a tightly restricted key with `command="..."` in `authorized_keys`.

### What is ssh-agent doing?

`ssh-agent` holds your unlocked private keys in memory. When SSH needs to sign a challenge, it asks the agent. The agent does the signing without ever giving the key back to SSH. The benefit: you type the passphrase once. The agent dies with your session. The keys go with it.

### ProxyCommand vs ProxyJump

`ProxyJump` is the modern, simple way: "use this host as a hop." `ProxyCommand` is the old way, which lets you specify any arbitrary command that connects you to the target (so you could use `nc`, or a SOCKS client, or whatever). Use `ProxyJump` unless you have a weird requirement.

### What does -t do?

`-t` forces a pseudo-terminal allocation. Necessary if you're running an interactive command on the remote (like `sudo`, which wants to read a password). Without `-t`, programs that expect a terminal misbehave.

### Agent forwarding security risks

`ForwardAgent yes` lets the remote machine talk to your local agent. This is convenient (you can SSH from the remote to a third host without copying keys around), but it means anyone with root on the remote can hijack your agent and SSH as you everywhere. **Only use agent forwarding to hosts you completely trust.** Better: use `ProxyJump`, which doesn't require forwarding the agent.

### Why is my key not being offered?

A few reasons:

1. Your `IdentityFile` doesn't match. Check `ssh -G hostname`.
2. `IdentitiesOnly yes` isn't set, and the server hit `MaxAuthTries` before reaching your key.
3. Your private key has wrong permissions and SSH is silently ignoring it.
4. You named your key something non-default and there's no `IdentityFile` line for it.
5. The agent has the key but `IdentitiesOnly yes` excludes everything not in `IdentityFile`.

Run `ssh -v` and read the output. SSH lists every key it tries.

### How does StrictHostKeyChecking work?

`StrictHostKeyChecking yes` means "if the host isn't in known_hosts, refuse to connect." `no` means "accept any new host without asking." `accept-new` (default since OpenSSH 7.6) means "accept new hosts silently, but yell if a known host's key changes."

Don't set `no` in production. It throws away the protection that catches man-in-the-middle attacks.

### .ssh/config vs system /etc/ssh/ssh_config

`~/.ssh/config` is your personal config. `/etc/ssh/ssh_config` is the system-wide config that applies to every user on the machine. Personal config wins on options that apply at the same scope. Both are merged, with personal config first.

### UseDNS yes vs no

In `sshd_config`. If `yes`, the server does a reverse DNS lookup on the connecting IP and uses that name in `from="..."` matches. Slow and breaks under flaky DNS. Most people set `UseDNS no` to save startup time.

### What is the control master?

The first SSH connection to a host. Subsequent connections to the same host hop into its tunnel instead of starting a new handshake. Saves time. Lives only as long as `ControlPersist` says.

### Why does SSH ask "yes/no/[fingerprint]" the first time?

That's the TOFU prompt. The server's host key isn't in your known_hosts yet. SSH wants you to confirm you trust the key before it remembers it.

### Why won't ssh-copy-id work?

Common reasons: password auth is disabled on the server, the user account is locked, `~/.ssh` perms are wrong, or the public key file doesn't exist. Check `ssh-copy-id -i ~/.ssh/id_ed25519.pub user@host` is pointing at the right key.

### What's the difference between known_hosts and authorized_keys?

`known_hosts` is on YOUR machine. It lists servers you've met. `authorized_keys` is on the SERVER. It lists users who can log in. Same idea (a list of public keys), opposite direction.

### Why does my connection drop after some idle time?

NAT routers age out idle TCP connections. Set `ServerAliveInterval 60` and `ServerAliveCountMax 3` in your `~/.ssh/config` to keep things alive.

### Why does sshd ask for the password twice?

If you have keys but they failed for some reason, sshd may fall back to password auth. The first prompt was probably the key trying and failing silently, the second is the password. Run `ssh -v` to see exactly what happened.

### What is "kex" in error messages?

KEX = "key exchange." The first thing SSH does is agree on a session key with the server. Errors at the KEX stage usually mean the client and server can't agree on a cipher or algorithm. Old servers with old configs sometimes can't talk to new clients.

## Vocabulary

| Word | Plain English |
| --- | --- |
| SSH | Secure Shell. The protocol and the program. |
| SSH-2 | The current version. Everything since 2006 uses it. |
| SSH-1 | The original. Deprecated. Removed from OpenSSH years ago. Don't use. |
| OpenSSH | The most common SSH implementation. Open source, on basically every Unix. |
| libssh | A C library for embedding SSH client/server in other programs. |
| libssh2 | A different C library for SSH client functionality. |
| dropbear | A small SSH implementation for embedded devices. |
| paramiko | An SSH library written in Python. |
| jsch | An SSH library written in Java. |
| sshd | The SSH server daemon. Listens for connections. |
| ssh-agent | A program that holds unlocked private keys in memory. |
| ssh-add | The command-line tool that adds keys to the agent. |
| ssh-copy-id | A helper that copies your public key into a server's authorized_keys. |
| ssh-keyscan | A tool to grab a server's public host key over the network. |
| ssh-keygen | The tool that generates, signs, and inspects keys. |
| Public key | The half of a key pair you share. |
| Private key | The half of a key pair you keep secret. |
| Key pair | The two halves together. They are mathematically linked. |
| Passphrase | A password used to encrypt your private key file. |
| RSA | An old public-key algorithm based on prime number multiplication. Still works. |
| DSA | An old public-key algorithm. Deprecated. Don't use. |
| ECDSA | Elliptic-curve DSA. Comes in P-256, P-384, P-521. |
| Ed25519 | A modern elliptic-curve algorithm. Recommended default. |
| Ed448 | Bigger sibling of Ed25519. |
| X25519 | The Diffie-Hellman flavor of Curve25519. Used during key exchange. |
| Curve25519 | The elliptic curve Ed25519 and X25519 use. |
| NIST P-256 | An elliptic curve standardized by NIST. Used by ECDSA-256. |
| NIST P-384 | Bigger NIST curve. ECDSA-384. |
| NIST P-521 | Biggest NIST curve. ECDSA-521. |
| SHA-256 fingerprint | A short hash of a key. Used to identify it visually. |
| MD5 fingerprint | An old short hash. Legacy display, used to be default. |
| SHA-1 | A hash function deprecated for SSH host keys in OpenSSH 8.2+. |
| known_hosts | The file on your machine that lists servers you've met. |
| authorized_keys | The file on a server that lists users who can log in. |
| authorized_keys2 | An old alternate name. OpenSSH no longer reads it by default. |
| `command="..."` restriction | A line in authorized_keys that forces a specific command on login. |
| no-port-forwarding | Restriction on a public key that disables port forwarding. |
| no-agent-forwarding | Disables agent forwarding for a key. |
| no-X11-forwarding | Disables X11 forwarding for a key. |
| no-pty | Disables terminal allocation for a key. |
| Principals | Names a certificate is valid for (users or hosts). |
| OpenSSH certificate | A signed key. Lets a CA delegate trust to many users. |
| CA | Certificate Authority. The signing authority. |
| Signing | Using a private key to make a verifiable proof. |
| ssh-cert | An OpenSSH-format certificate file. |
| valid_principals | The list of users or hosts a cert is good for. |
| valid_after | When a cert becomes valid. |
| valid_before | When a cert expires. |
| force_command | A cert option that locks the user to one command. |
| source_address | A cert option that locks the user to specific IPs. |
| ssh_config | The client config file. |
| sshd_config | The server config file. |
| Host | A pattern in ssh_config that matches names. |
| HostName | The actual address to connect to. |
| User | The username on the remote. |
| Port | TCP port. SSH default is 22. |
| IdentityFile | Which private key to use. |
| IdentitiesOnly | Only try the listed key, not every key in the agent. |
| ForwardAgent | Forward your local agent to the remote. |
| ForwardX11 | Forward X11 GUI to your local display. |
| ForwardX11Trusted | Forward X11 with full access to your local display. |
| ProxyCommand | Run an arbitrary command to make the connection. |
| ProxyJump | Hop through a bastion host. |
| ControlMaster | Multiplex many SSH sessions over one connection. |
| ControlPath | Where to put the multiplex socket. |
| ControlPersist | How long to keep the master alive after children close. |
| ServerAliveInterval | How often the client pings the server. |
| ServerAliveCountMax | How many missed pings before giving up. |
| TCPKeepAlive | Whether to send TCP keepalives at the socket level. |
| ClientAliveInterval | sshd version of ServerAliveInterval. |
| MaxSessions | How many session channels per connection. |
| MaxStartups | How many half-open connections sshd accepts. |
| MaxAuthTries | How many auth attempts before sshd disconnects. |
| AuthorizedKeysFile | Path to authorized_keys. Default `~/.ssh/authorized_keys`. |
| AuthorizedKeysCommand | A program sshd runs to fetch authorized keys (e.g. from LDAP). |
| ChallengeResponseAuthentication | An older name for keyboard-interactive. |
| KbdInteractiveAuthentication | The newer name. Used for 2FA prompts. |
| PubkeyAuthentication | Whether public-key auth is allowed. |
| PasswordAuthentication | Whether password auth is allowed. |
| PermitRootLogin | Whether root can SSH in directly. |
| PermitEmptyPasswords | Whether empty passwords are accepted. |
| AllowUsers | Whitelist of usernames that may log in. |
| AllowGroups | Whitelist of groups that may log in. |
| DenyUsers | Blacklist of usernames. |
| DenyGroups | Blacklist of groups. |
| Match block | Apply different sshd_config options to specific users/hosts. |
| Banner | Text shown to clients before login. |
| X11Forwarding | Whether the server allows X11 forwarding. |
| X11UseLocalhost | Whether the X11 forwarder binds to localhost only. |
| AcceptEnv | Which environment variables sshd accepts from clients. |
| SendEnv | Which environment variables ssh sends to the server. |
| Agent forwarding | Letting the remote use your local agent. |
| X forwarding | Tunneling X11 GUIs over SSH. |
| Local port forwarding | -L. Tunnel a remote port to a local port. |
| Remote port forwarding | -R. Tunnel a local port to a remote port. |
| Dynamic forwarding | -D. SOCKS5 proxy through SSH. |
| Tunneling | Carrying arbitrary protocols inside SSH. |
| SOCKS5 proxy | A simple proxy protocol that any modern app can use. |
| SCP | Secure copy. Old protocol, still widely used. |
| SFTP | SSH File Transfer Protocol. Modern, structured. |
| SFTP server | The server-side process that handles SFTP. |
| sftp-server | The OpenSSH SFTP subsystem binary. |
| internal-sftp | A built-in SFTP subsystem. Used with chroot. |
| ChrootDirectory | Lock a user into a directory tree. Often used with internal-sftp. |
| Jumphost | A server you SSH to first to reach inside servers. |
| Bastion | Same as jumphost. |
| Multiplexing | Sharing one SSH connection for many sessions. |
| ControlMaster auto | Tell SSH to multiplex when possible. |
| ssh-add -t | Add a key with a timeout. |
| ssh-add -d | Delete one key from the agent. |
| ssh-add -D | Delete all keys from the agent. |
| gpg-agent SSH support | Using GnuPG to also serve SSH keys. |
| YubiKey SSH | Using a YubiKey hardware token for SSH. |
| FIDO2 SSH key | A key whose private half lives on a hardware token. |
| sk-ed25519 | A FIDO2-backed Ed25519 SSH key. |
| sk-ecdsa | A FIDO2-backed ECDSA SSH key. |
| Hardware-backed key | A key where the private half never leaves a chip. |
| Certificate-based auth | Login by presenting a CA-signed key. |
| KRL | Key Revocation List. Lets a server reject specific certs. |
| HMAC-SHA2-256 | A modern message authentication code. |
| HMAC-SHA2-512 | The bigger version. |
| chacha20-poly1305 | A modern AEAD cipher. Default in many OpenSSH versions. |
| AES-128-GCM | An authenticated AES variant. |
| AES-256-GCM | A bigger authenticated AES variant. |
| KexAlgorithms | List of allowed key-exchange algorithms. |
| HostKeyAlgorithms | List of allowed host-key algorithm names. |
| MACs | List of allowed message-authentication-code algorithms. |
| Ciphers | List of allowed bulk ciphers. |
| RekeyLimit | When to negotiate a fresh session key. |
| IPQoS | The TOS/DSCP bits SSH puts on its packets. |
| Server cert | A certificate signed for use as a host key. |
| Host cert | Same as server cert. |
| Principal | A name (user or host) inside a certificate. |
| Validity period | How long a cert is valid. |
| Nonce | A random number used once to prevent replay. |
| TOFU | Trust On First Use. Trust the host's key the first time. |
| Man-in-the-middle | An attacker between you and the server who can read or change traffic. |
| KEX | Key exchange. The first phase of an SSH connection. |
| Diffie-Hellman | A way for two parties to agree on a secret over a public channel. |
| ECDH | Elliptic-curve Diffie-Hellman. |
| Ephemeral key | A key created for one session and thrown away. |
| Forward secrecy | Property that breaking today's session key doesn't reveal past sessions. |
| TCP | Transmission Control Protocol. SSH runs on top of it. |
| Port 22 | The default TCP port for SSH. |
| Banner grab | Reading the SSH version string the server sends first. |
| Login shell | The shell you get when you SSH in interactively. |
| TTY | A terminal device. Needed for interactive programs. |
| pty | Pseudo-TTY. A software TTY for remote sessions. |
| Verbose mode | -v, -vv, -vvv. More logs about what SSH is doing. |
| StrictHostKeyChecking | Whether to refuse new or changed host keys. |
| accept-new | StrictHostKeyChecking mode that accepts new hosts but warns on changes. |
| GatewayPorts | Whether sshd binds forwarded ports to all interfaces. |
| AllowTcpForwarding | Whether sshd allows -L and -R. |
| PermitOpen | Restrict which destinations -L can reach through this server. |
| AllowStreamLocalForwarding | Whether sshd allows Unix-socket forwarding. |
| TrustedUserCAKeys | sshd config: which CA public keys to trust for user certs. |
| HostCertificate | sshd config: a CA-signed cert to present as the host key. |
| RevokedKeys | sshd config: a list of keys (or a KRL) to refuse. |
| Visual host key | The little ASCII-art picture printed when you generate a key. |
| RandomArt | The same little ASCII-art picture. |
| moduli | The DH prime numbers used during diffie-hellman-group-exchange. |
| ssh_known_hosts | The system-wide version of known_hosts in /etc/ssh. |

## Try This

If you have access to a Linux or macOS machine, try this whole sequence end to end. It is the "first day" rite of passage for SSH.

1. Generate a fresh key.

   ```
   $ ssh-keygen -t ed25519 -C "you@yourbox"
   ```

2. Look at the two files it made.

   ```
   $ ls -la ~/.ssh/id_ed25519*
   ```

3. Copy your public key to a server you have access to (substitute your real server).

   ```
   $ ssh-copy-id user@server.example.com
   ```

4. Connect.

   ```
   $ ssh user@server.example.com
   ```

5. Run a command on the remote.

   ```
   $ ssh user@server.example.com 'uptime'
   ```

6. Try a port forward. From your laptop:

   ```
   $ ssh -L 8080:localhost:80 user@server.example.com
   ```

   In another terminal on your laptop, while that SSH is open:

   ```
   $ curl http://localhost:8080/
   ```

7. Make a config block.

   ```
   $ cat >> ~/.ssh/config <<EOF
   Host myserver
       HostName server.example.com
       User user
       IdentityFile ~/.ssh/id_ed25519
   EOF
   $ chmod 600 ~/.ssh/config
   $ ssh myserver
   ```

8. Set up a SOCKS proxy for fun.

   ```
   $ ssh -D 1080 -N -f myserver
   $ curl --socks5 localhost:1080 https://api.ipify.org
   ```

9. Tear down the background SSH.

   ```
   $ pkill -f "ssh -D 1080"
   ```

10. Generate a CA, sign your own user key, log in via cert.

    ```
    $ ssh-keygen -f ~/my_ca -t ed25519 -C "my CA"
    $ ssh-keygen -s ~/my_ca -I "test-id" -n user -V +1d ~/.ssh/id_ed25519.pub
    $ ls ~/.ssh/id_ed25519-cert.pub
    $ ssh-keygen -L -f ~/.ssh/id_ed25519-cert.pub
    ```

If everything worked, congratulations. You are now further along with SSH than 80% of the engineers on the planet.

## Where to Go Next

- Read `security/ssh` for the operations-level cheat sheet.
- Read `network-tools/ssh-tunneling` for advanced tunneling patterns.
- Read `troubleshooting/ssh-errors` for a deeper run-through of error states.
- Read `security/pki` for the world of certificates outside of SSH.
- Read `security/cryptography` for the algorithms underneath all of this.
- Read `security/tls` and `ramp-up/tls-eli5` to compare with SSH's cousin.
- Read Michael W. Lucas's book *SSH Mastery* for the canonical deep dive.
- Read `man ssh`, `man ssh_config`, `man sshd_config`, `man ssh-keygen`. The manuals are dense but honest.

## See Also

- security/ssh
- network-tools/ssh-tunneling
- troubleshooting/ssh-errors
- security/pki
- security/cryptography
- security/tls
- networking/dns
- ramp-up/tls-eli5
- ramp-up/tcp-eli5
- ramp-up/linux-kernel-eli5
- ramp-up/bash-eli5

## References

- RFC 4250 — The Secure Shell (SSH) Protocol Assigned Numbers
- RFC 4251 — The Secure Shell (SSH) Protocol Architecture
- RFC 4252 — The Secure Shell (SSH) Authentication Protocol
- RFC 4253 — The Secure Shell (SSH) Transport Layer Protocol
- RFC 4254 — The Secure Shell (SSH) Connection Protocol
- RFC 4255 — Using DNS to Securely Publish SSH Key Fingerprints (SSHFP)
- RFC 4256 — Generic Message Exchange Authentication for SSH
- RFC 8709 — Ed25519 and Ed448 Public Key Algorithms for SSH
- RFC 8731 — SSH Key Exchange Method using Curve25519 and Curve448
- OpenSSH release notes — https://www.openssh.com/releasenotes.html
- "SSH Mastery" by Michael W. Lucas — the standard book
- man ssh
- man ssh_config
- man sshd_config
- man ssh-keygen
- man ssh-agent
- man ssh-add
- man scp
- man sftp
- man ssh-keyscan
- OpenSSH 9.x deprecation list (legacy SCP protocol, ssh-rsa for HostKeyAlgorithms by default in 8.2+, removal of ssh-dss everywhere it lingered)

## Version Notes

- **OpenSSH 6.5 (2014):** Ed25519 added.
- **OpenSSH 7.0 (2015):** SSH protocol 1 disabled by default. DSA disabled by default. Old SHA-1 KEX removed.
- **OpenSSH 7.6 (2017):** `StrictHostKeyChecking accept-new` introduced.
- **OpenSSH 8.0 (2019):** Default RSA size pumped to 3072 for new keys. Ed25519 the default if `-t` is given without size.
- **OpenSSH 8.2 (2020):** FIDO2/WebAuthn keys (sk-ed25519, sk-ecdsa) added. ssh-rsa (SHA-1) deprecation announcement for HostKeyAlgorithms.
- **OpenSSH 8.5 (2021):** PerSourceMaxStartups for finer rate limiting.
- **OpenSSH 8.8 (2021):** ssh-rsa (SHA-1) disabled by default in client. Use `rsa-sha2-256` and `rsa-sha2-512` instead.
- **OpenSSH 9.0 (2022):** scp(1) uses SFTP protocol by default. Legacy SCP available with `-O`.
- **OpenSSH 9.5 (2023):** Default key types again pruned; legacy DSA removed entirely from compile-time options on most distros.

If you remember nothing else: **use Ed25519, use a passphrase, use the agent, use `~/.ssh/config`, never share your private key, and verify host fingerprints out-of-band the first time.** Those six habits cover 95% of staying out of trouble.
