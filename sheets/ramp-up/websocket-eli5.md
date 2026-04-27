# WebSockets — ELI5 (Walkie-Talkies for the Web)

> A WebSocket is a phone line that two computers leave open so they can talk to each other any time, in either direction, without hanging up and dialing again.

## Prerequisites

(none — but a couple of other ELI5 sheets help if you want them)

You do **not** need to know anything about web programming, networking, or computers in general to read this sheet. By the time you reach the bottom, you will know what a WebSocket is, why people invented it, what its handshake looks like, what its frames look like, what masking is, why client frames are masked but server frames are not, what `ws://` and `wss://` mean, why proxies sometimes break WebSockets, and how to poke a real WebSocket server with a real command from a real terminal.

If you want to be cozy first:

- `cs ramp-up tcp-eli5` — WebSocket sits on top of TCP. Knowing TCP makes some of this sheet make a lot more sense, but it is not required.
- `cs ramp-up tls-eli5` — `wss://` (the secure flavour of WebSocket) wraps a WebSocket inside TLS, which is the same thing that wraps an `https://` page. Again, optional.
- `cs ramp-up linux-kernel-eli5` — the Linux kernel actually implements all of this for you under the hood. Optional.

If a word in this sheet feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is a WebSocket?

### The restaurant counter (HTTP)

Picture a regular restaurant. You sit at a table. Every time you want something, you have to walk up to the counter, say what you want, walk back to your table, and wait. If you want to know whether the food is ready, you walk back up to the counter and ask, "Is my food ready?" The cashier says, "No, not yet," and you walk back to your table again. A minute later you walk up again. "Is it ready yet?" "Almost." Walk back. Walk up. "How about now?" "Just a moment."

You spend more time walking back and forth than eating.

That is **HTTP.** Every web page you have ever loaded works this way. Your browser walks up to the server, asks for the page, walks back, the server walks up with the page, hands it over, and the connection ends. Want a new piece of information? Walk up again. Connection ends again. Want another piece? Walk up again. Connection ends again.

For loading a page once, that is fine. For chatting with a friend in real time, it is awful. For watching a stock price tick up and down, it is wasteful. For playing an online game, it is impossible. Walking up to the counter and back takes time. Each "trip" eats CPU on your computer, eats CPU on the server, and burns network on the wire.

### The walkie-talkie (WebSocket)

Now imagine instead the restaurant gives you a walkie-talkie when you sit down. Your walkie-talkie is matched with one in the kitchen. The line is open the whole time you are at the restaurant. Want to ask a question? Push the button and ask. The answer comes back the same way. Better yet, the kitchen can buzz **you**. They can say, "Your food is ready, come pick it up," and you hear it instantly without asking. Either side can talk at any time. The line never closes until somebody hangs up.

That is a **WebSocket.** It is a permanent two-way line between your computer and a server. After a brief polite introduction (called the **handshake**), both sides can send messages whenever they want. No more walking. No more reopening the connection. No more wasted trips.

### Letters vs. phone calls

Another way to think about it:

- **HTTP** is like sending letters in the mail. You write a letter, drop it in a mailbox, wait for a reply letter to come back, write another letter, and so on. Each letter is a complete round trip. Slow. Bursty. Polite, but expensive.
- **WebSocket** is like a phone call that stays connected. You talk. You pause. The other person talks. Either of you can interrupt. When you are done you say goodbye and hang up. While the call is open, sending another message is basically free.

If your application's pattern is "say one thing, get one thing back, done," HTTP is great. If your application's pattern is "talk to each other a lot, in either direction, for a long time," WebSocket is great.

### Where you have already used WebSockets

You probably use WebSockets every day without knowing it.

- **Chat apps.** Discord, Slack, WhatsApp Web, Microsoft Teams. When a friend sends a message, it pops up on your screen instantly. Nobody asked the server "any new messages yet?" The server just buzzed your walkie-talkie.
- **Multiplayer games.** Online games push player movements, shots, scores, and chat through WebSockets. Reaction time matters in games, and WebSockets are fast.
- **Live sports scoreboards.** When a goal is scored, the score on your phone updates in a fraction of a second. WebSocket pushes the new number.
- **Stock and crypto tickers.** Prices update constantly. The server streams the latest number through a WebSocket.
- **Collaborative documents.** Google Docs, Notion, Figma. When someone else moves their cursor or types a letter, you see it appear. WebSockets ferry every keystroke and cursor move.
- **Live notifications.** That little red dot on your inbox tab? A WebSocket pushed it the second the email arrived.
- **Live dashboards.** Grafana, Kibana, Datadog. Numbers and graphs that update every second instead of refreshing the page.
- **Voice and video calls in browsers.** Most signalling for WebRTC (the browser's video-call layer) rides over a WebSocket.

### A simple summary picture

```
HTTP (regular web)              WebSocket
==================              =========
Open. Ask. Answer. Close.       Open. Talk. Talk. Talk. Talk. Close.
Open. Ask. Answer. Close.       (one open the whole time)
Open. Ask. Answer. Close.
(many opens, many closes)
```

That single difference (open once, talk for a long time, in both directions) is the entire point of WebSockets. Everything else in this sheet is about how that simple idea is actually implemented.

## Why HTTP Alone Wasn't Enough

People wanted real-time websites long before WebSockets existed. Chat rooms in the 1990s. Live news tickers. Stock dashboards. Multiplayer browser games. None of these worked well over plain HTTP. So programmers built workarounds, each more clever than the last, to fake real-time over a protocol that did not really support it.

It is worth seeing the workarounds, because each one teaches you something about why WebSockets are a good idea.

### Workaround 1: Polling

Polling means: every few seconds, the browser asks the server, "Anything new?"

```
Browser: "Any new chat messages?" -> Server: "No."
(wait 5 seconds)
Browser: "Any new chat messages?" -> Server: "No."
(wait 5 seconds)
Browser: "Any new chat messages?" -> Server: "Yes! Here you go."
(display message)
(wait 5 seconds)
Browser: "Any new chat messages?" -> Server: "No."
```

This is dumb but it works. The problems are:

- **It wastes everything.** Most of the time the answer is "no," and yet you are paying for the round trip on every poll. Multiply by a million users and you have a million pointless requests per minute.
- **It is laggy.** If you set the poll interval to 5 seconds, your average delay is 2.5 seconds. If you set it to 1 second, you waste five times more bandwidth and CPU. There is no setting that is both fast and cheap.
- **It does not scale.** A single chat server might handle 100 real-time clients on a phone line, but it has to handle a million HTTP requests per minute to do the same job by polling. The server falls over.

But for ages, polling was the only thing that worked in every browser. Many sites used it.

### Workaround 2: Long polling

Long polling is polling that is a tiny bit smarter. Instead of the server immediately answering "no," it just **holds the request open** until something actually happens.

```
Browser: "Any new chat messages?"
Server: (silence... silence... silence...)
Server: (a new message arrives)
Server: "Yes! Here is the message."
Browser: (handles message, immediately asks again)
Browser: "Any new chat messages?"
Server: (silence... silence... silence...)
```

This is better:

- **Lower latency.** As soon as a message arrives on the server, the response is sent immediately. No 5-second wait.
- **Fewer wasted round trips.** The server only responds when there is something to say.

But it still has serious issues:

- **One-way.** Long polling is good for "server tells me when something happens." Going the other way (the client wanting to send a message) still requires a separate HTTP request.
- **Connection limits.** A held-open HTTP request hogs a TCP connection. Browsers limit how many connections per host (usually 6). Servers have to handle a connection per user, all stuck open.
- **Proxies and timeouts.** Many proxies and load balancers kill HTTP requests that have been silent for too long. Long polling has to deal with reconnections constantly.
- **It is still HTTP.** Every reconnection is a fresh HTTP request, with all its headers (cookies, user-agent, accept, etc.) every time. That is hundreds of bytes of overhead per round trip.

Long polling was a clever band-aid, but a band-aid is still a band-aid.

### Workaround 3: Server-Sent Events (SSE)

In the late 2000s the browser people standardized **Server-Sent Events.** This is basically "long polling, but the server can keep sending more data on the same response." The browser opens an HTTP request to a special URL, and the server replies with `Content-Type: text/event-stream` and just keeps writing little chunks of text forever.

```
Browser: GET /events  (with Accept: text/event-stream)
Server:  HTTP/1.1 200 OK
         Content-Type: text/event-stream
         (one long never-ending response)

         event: message
         data: hello

         event: message
         data: world
```

SSE is much nicer than polling:

- **One open connection** for as long as the user is on the page.
- **Server-pushed messages** with low latency.
- **Built into browsers** as the `EventSource` JavaScript class, no library required.
- **Auto-reconnect** with `Last-Event-ID` to resume where you left off.

But SSE has one giant limitation: **it is one-way.** The server can push to the client, but the client cannot send messages back over the same connection. To send something to the server, the client still has to do a regular HTTP POST. This is fine for stock tickers, but not great for chat or games where both sides talk a lot.

SSE is also strictly text. You cannot send binary data over SSE without base64-encoding it (which costs 33% size overhead).

### The real answer: WebSocket

WebSocket showed up in 2011 (RFC 6455) as the proper solution. It says: forget all this polling nonsense. Let us just **upgrade** an HTTP connection into a real two-way persistent message channel. Once upgraded, the connection is no longer HTTP. It is a bidirectional stream of small framed messages, in either direction, until somebody closes it. Text or binary. Low overhead. Low latency. Standardized. Implemented in every browser, every server framework, every mobile platform.

That is what you are reading about. Polling, long polling, SSE were all stops on the road. WebSocket is the destination.

### A quick comparison

```
                    Polling   Long Polling   SSE       WebSocket
Direction           c->s      c->s           s->c      both
Connection per msg  yes       no/yes-ish     no        no
Bytes overhead/msg  high      high           low       very low
Latency             bad       okay           good      best
Binary support      yes       yes            no        yes
Browser support     all       all            all       all (modern)
```

WebSockets won. They are the right tool for almost every "I want real-time bidirectional traffic in a browser" problem.

## The Upgrade Handshake

Here is one of the most clever things about WebSocket. It does not invent a totally new connection. Instead, it **starts** every connection as a perfectly normal HTTP request. Why? Because if it started as something else, it would not work through the world's existing infrastructure. Every firewall, every proxy, every load balancer, every corporate web filter understands HTTP. They would block anything else.

So the trick is: speak HTTP to get past everybody's firewalls, then ask the server, "Hey, can we switch this connection to WebSocket?" If the server says yes, the connection is no longer HTTP. The same TCP connection that was carrying HTTP a moment ago is now carrying WebSocket frames.

This is called the **upgrade handshake.**

### The client request

The client sends an HTTP request that looks slightly weird:

```
GET /chat HTTP/1.1
Host: server.example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Version: 13
Origin: http://example.com
```

Let us walk through every line:

- `GET /chat HTTP/1.1` — a normal HTTP request line. The path `/chat` is whatever endpoint on the server handles WebSockets. The HTTP version is 1.1 (since 2011, this is the only one that the original WebSocket spec works on; HTTP/2 and HTTP/3 use slightly different mechanisms — see later).
- `Host: server.example.com` — standard HTTP, says which website you are talking to.
- `Upgrade: websocket` — the magic word. "I would like to upgrade this connection. The protocol I want to upgrade to is called `websocket`."
- `Connection: Upgrade` — confirms that this connection is requesting an upgrade. (A `Connection` header normally controls per-hop behavior.)
- `Sec-WebSocket-Key: ...` — a randomly generated 16 bytes, base64-encoded. The client makes this up fresh for every connection. It is **not** a secret. It is a way to make sure the response is from a real WebSocket server, not from a confused HTTP server that doesn't know what `Upgrade: websocket` means.
- `Sec-WebSocket-Version: 13` — which version of the WebSocket protocol the client wants. As of 2026, version 13 is the only version that exists. RFC 6455 (2011) defined version 13 and that was that. Earlier drafts existed but are extinct.
- `Origin: http://example.com` — which web origin the WebSocket connection is coming from. The server uses this to decide whether to allow the connection. (More on this in the security section.)

Optional headers you might also see:

- `Sec-WebSocket-Protocol: chat, superchat` — a list of application-level subprotocols the client speaks. The server picks one.
- `Sec-WebSocket-Extensions: permessage-deflate` — extensions the client supports, like compression.
- `Cookie: ...` — the same cookies the server has set on the client. Often used for authentication.

### The server response

If the server understands and agrees, it replies with status code 101:

```
HTTP/1.1 101 Switching Protocols
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

Status code **101 Switching Protocols** is the special HTTP code that says "okay, this connection is now a different protocol from this point on." After this response, no more HTTP travels on this connection. The bytes that follow are WebSocket frames.

`Sec-WebSocket-Accept` is the server's proof that it understood the request. It is not random. It is computed from the client's `Sec-WebSocket-Key` using a fixed recipe:

1. Take the client's `Sec-WebSocket-Key` value (the base64 string).
2. Append the magic GUID: `258EAFA5-E914-47DA-95CA-C5AB0DC85B11`. This is a fixed string defined in RFC 6455. It never changes. Every WebSocket implementation in the world uses this exact GUID.
3. Run SHA-1 on the result.
4. Base64-encode the SHA-1 hash.

That base64 string is what the server sends back as `Sec-WebSocket-Accept`. The client computes the same thing on its end. If they match, the client knows it is talking to a real WebSocket server and not a confused HTTP server that copy-pasted some headers.

You can compute it yourself in the terminal. Try it:

```
$ echo -n "dGhlIHNhbXBsZSBub25jZQ==258EAFA5-E914-47DA-95CA-C5AB0DC85B11" | openssl dgst -sha1 -binary | base64
s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

That matches the value in the example response. The client and server can both verify it.

### A picture of the handshake

```
   CLIENT                                       SERVER
     |                                            |
     |   1. TCP connect to port 80 or 443         |
     |------------------------------------------->|
     |                                            |
     |   2. (optional) TLS handshake (for wss://) |
     |<==========================================>|
     |                                            |
     |   3. HTTP GET with Upgrade: websocket      |
     |   Sec-WebSocket-Key: <random 16 bytes b64> |
     |   Sec-WebSocket-Version: 13                |
     |------------------------------------------->|
     |                                            |
     |   4. HTTP/1.1 101 Switching Protocols      |
     |   Sec-WebSocket-Accept: <SHA1 of key+GUID> |
     |<-------------------------------------------|
     |                                            |
     |   5. Connection is now WEBSOCKET           |
     |   Both sides can send frames any time.     |
     |   <---- text frames ---->                  |
     |   <---- binary frames -->                  |
     |   <---- ping/pong ---->                    |
     |                                            |
     |   6. Either side sends Close frame         |
     |---- close (opcode 0x8, code 1000) -------->|
     |                                            |
     |   7. Other side echoes Close               |
     |<--- close (opcode 0x8, code 1000) ---------|
     |                                            |
     |   8. TCP close                             |
     |<==========================================>|
```

That entire dance, from step 1 to step 5, happens in maybe 100 milliseconds on a normal network. From that point on, both sides just send frames whenever they want.

### Why this design is so clever

Three things are worth appreciating about this handshake:

- **It is HTTP.** Every load balancer, proxy, and firewall in the world that allows port 80/443 will let it through. WebSocket did not have to invent its own port or fight its way past existing infrastructure.
- **It is unambiguous.** The `Sec-WebSocket-Key` / `Sec-WebSocket-Accept` exchange is a simple challenge that proves both sides actually understand WebSocket. A server that just blindly returns 101 without computing the right Accept is rejected by the client, so accidental upgrades cannot happen.
- **After step 5, HTTP is gone.** The handshake is the last HTTP that ever happens on that connection. From then on it is binary-framed WebSocket. There is no more HTTP overhead per message.

### Trying it yourself

You can poke a real WebSocket server with `curl`. Try this. There is a public echo server at `echo.websocket.events`:

```
$ curl -i -N \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 13" \
    -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
    https://echo.websocket.events/ 2>&1 | head -20
HTTP/1.1 101 Switching Protocols
Server: TornadoServer/6.1
Date: Sun, 27 Apr 2026 10:11:12 GMT
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: qGEgH3En71di5rrssAZTmtRTyFk=
```

That `101 Switching Protocols` response, plus the `Sec-WebSocket-Accept` line, tells you the upgrade succeeded. After that point, `curl` cannot do anything useful (it is not a WebSocket client), but the handshake itself is exactly what every WebSocket library does on your behalf.

You computed `Sec-WebSocket-Accept` from `Sec-WebSocket-Key` (which was `SGVsbG8sIHdvcmxkIQ==`) plus the magic GUID. We will verify that for fun:

```
$ echo -n "SGVsbG8sIHdvcmxkIQ==258EAFA5-E914-47DA-95CA-C5AB0DC85B11" | openssl dgst -sha1 -binary | base64
qGEgH3En71di5rrssAZTmtRTyFk=
```

Same value. The server is doing the same computation you just did.

## Frames: How Data Travels

Once the handshake is done, the connection no longer carries HTTP. It carries **frames.**

A frame is a small chunk of bytes with a tiny header at the start and a payload after. Each frame is a complete unit on the wire. Both sides send frames to each other, in either direction, at any time. A "message" in WebSocket terminology can be one frame or many frames glued together.

The frame format is the heart of WebSocket. Once you understand the frame, you understand WebSocket.

### The frame layout

Here is the exact bit layout of a WebSocket frame, from RFC 6455:

```
  0                   1                   2                   3
  0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
 +-+-+-+-+-------+-+-------------+-------------------------------+
 |F|R|R|R| opcode|M| Payload len |    Extended payload length    |
 |I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
 |N|V|V|V|       |S|             |   (if payload len==126/127)   |
 | |1|2|3|       |K|             |                               |
 +-+-+-+-+-------+-+-------------+ - - - - - - - - - - - - - - - +
 |     Extended payload length continued, if payload len == 127  |
 + - - - - - - - - - - - - - - - +-------------------------------+
 |                               |Masking-key, if MASK set to 1  |
 +-------------------------------+-------------------------------+
 | Masking-key (continued)       |          Payload Data         |
 +-------------------------------- - - - - - - - - - - - - - - - +
 :                     Payload Data continued ...                :
 + - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - - +
 |                     Payload Data continued ...                |
 +---------------------------------------------------------------+
```

That ASCII art is straight out of the RFC. It is a bit dense. Let us unpack it piece by piece.

### Byte 1: FIN, RSV1-3, opcode

The very first byte of a frame is split into pieces:

- **FIN (1 bit)** — set to 1 if this is the final frame of a message. Set to 0 if more frames are coming. Most messages fit in one frame so FIN is usually 1.
- **RSV1, RSV2, RSV3 (1 bit each)** — reserved bits for extensions. Normally 0. The compression extension (`permessage-deflate`) uses RSV1.
- **Opcode (4 bits)** — what kind of frame this is. The four bits give 16 possible values; only six are defined.

The opcodes you need to know:

```
0x0  Continuation frame  (continues a multi-frame message)
0x1  Text frame          (UTF-8 text data, like JSON)
0x2  Binary frame        (raw bytes, like images or protobuf)
0x3-0x7  Reserved for future data frames (currently unused)
0x8  Close               (graceful shutdown)
0x9  Ping                (heartbeat: are you alive?)
0xA  Pong                (heartbeat reply: yes I am)
0xB-0xF  Reserved for future control frames (currently unused)
```

Opcodes 0x0-0x7 are **data frames** (carry your message data). Opcodes 0x8-0xF are **control frames** (manage the connection itself). The high bit of the opcode tells you which: 0 means data, 1 means control.

### Byte 2: MASK and payload length

The second byte is also split:

- **MASK (1 bit)** — set to 1 if the payload is masked. Always 1 from client to server. Always 0 from server to client. (More on this in the next section.)
- **Payload length (7 bits)** — how big the payload is, but only kind of.

The 7-bit payload length is too small for big payloads, so the protocol has a clever trick:

- If the value is 0–125, that **is** the payload length. End of story.
- If the value is **126**, the actual length is in the next 2 bytes (a 16-bit unsigned integer, big-endian). Payloads up to 65535 bytes use this form.
- If the value is **127**, the actual length is in the next 8 bytes (a 64-bit unsigned integer). Payloads bigger than 65535 bytes use this form.

So a small frame has just a 2-byte header. A medium frame has 4 bytes. A really big frame has 10 bytes. Compare with HTTP, where the headers alone are usually hundreds of bytes for every request.

### After the length: the masking key

If MASK is 1 (so for every frame from a client to a server), the next 4 bytes are the **masking key.** This is a random 4-byte number generated fresh for every frame. The client uses it to scramble the payload (we will explain how in a moment), and the server uses the same key to unscramble.

Server-to-client frames have MASK = 0 and **no masking key**, so those 4 bytes are not present.

### After everything: the payload

Finally, the actual data. This is your text or binary message. If MASK was 1, the payload bytes are XORed with the masking key. If MASK was 0, the payload is plain.

### A worked example: sending "Hello"

Let us see what an actual client-to-server text frame for the message "Hello" looks like on the wire.

- The message is 5 bytes long (less than 126), so payload length fits in the 7-bit field.
- It is text, so opcode is 0x1.
- It is a single complete message, so FIN is 1.
- It is from client to server, so MASK is 1 and there is a 4-byte masking key.

Header byte 1: `1000 0001` = 0x81 (FIN=1, RSV=000, opcode=0001)
Header byte 2: `1000 0101` = 0x85 (MASK=1, length=0000101 which is 5)
Masking key (random):  `0x37 0xfa 0x21 0x3d`
Payload (XOR-masked):
  - 'H' (0x48) ^ 0x37 = 0x7f
  - 'e' (0x65) ^ 0xfa = 0x9f
  - 'l' (0x6c) ^ 0x21 = 0x4d
  - 'l' (0x6c) ^ 0x3d = 0x51
  - 'o' (0x6f) ^ 0x37 = 0x58

Total bytes on the wire:

```
0x81 0x85 0x37 0xfa 0x21 0x3d 0x7f 0x9f 0x4d 0x51 0x58
```

11 bytes total. The whole message.

For comparison, an equivalent HTTP POST might look like:

```
POST /chat HTTP/1.1
Host: server.example.com
Content-Type: text/plain
Content-Length: 5
Cookie: session=abc123def456...
User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 ...
Accept: */*

Hello
```

That is around 250 bytes for a 5-byte message. WebSocket: 11 bytes. HTTP: 250 bytes. Repeat that for every keystroke in a chat app and you can see the savings add up.

### Server-to-client "Hello" (no masking)

When the server replies "Hello" back, the frame is identical except MASK = 0 and the masking key is not present:

Header byte 1: `1000 0001` = 0x81 (FIN=1, opcode=text)
Header byte 2: `0000 0101` = 0x05 (MASK=0, length=5)
Payload (unmasked): `0x48 0x65 0x6c 0x6c 0x6f` ("Hello")

Total bytes on the wire:

```
0x81 0x05 0x48 0x65 0x6c 0x6c 0x6f
```

7 bytes total. Even shorter. Server-to-client frames have less overhead because they skip the masking key.

### Where to look for frames

You can see frames live with `wireshark` or `tshark`. Try this on a machine with `tshark` installed:

```
$ tshark -i any -Y 'websocket' -V
```

That will print every WebSocket frame on every interface, with all the bits decoded. This is the single best way to learn the frame format: type the command, then open a chat app, then watch frames fly by.

## Why Client Frames Are Masked (And Servers Aren't)

You might be wondering: what is up with the masking? Why does the client XOR every payload with a random key, and why does the server not have to do that?

The answer is one of those weird security stories that sounds made up but is real.

### The cache poisoning attack

Imagine the world before WebSocket was finalized. Many networks had **transparent HTTP proxies** sitting between users and the internet. These proxies cached HTTP responses to save bandwidth. If user A asked for `https://example.com/index.html`, the proxy would cache the response and serve the same bytes to user B when they asked for the same URL.

Now imagine WebSocket without masking. A malicious website running JavaScript in your browser could open a WebSocket and send any bytes it liked to its own attacker server. The bytes would travel through your network's transparent HTTP proxy. The proxy would not understand WebSocket framing — it would see the bytes as some weird-looking HTTP traffic.

But the attacker could **craft the bytes to look like a valid HTTP response.** Something like:

```
HTTP/1.1 200 OK
Content-Length: 1024
Cache-Control: public, max-age=86400

<malicious HTML that runs in the bank's domain>
```

The proxy might think, "Oh, this looks like a cacheable HTTP response for the URL `https://yourbank.com/login`," and store the malicious bytes in its cache. The next time anyone in your network asked for `https://yourbank.com/login`, the proxy would serve the attacker's HTML.

That is **cache poisoning.** The attacker poisoned the proxy's cache by smuggling fake HTTP responses through a WebSocket connection.

### Masking prevents the smuggle

If every byte the client sends is XORed with a random 4-byte key that the proxy does not know, the bytes look like garbage to the proxy. They cannot accidentally form a valid HTTP response. The masking key changes for every frame, so even if an attacker tried to brute-force a specific byte pattern, they would have to guess the key — and a 32-bit key has about 4 billion possible values.

The formal security argument: under the assumption that the masking keys are uniformly random, the probability that a 4-byte window of masked output looks like the start of a valid HTTP response is roughly 2^(-32), which is negligible. In practice, no proxy will ever cache-poison via a properly masked WebSocket payload.

### Why not mask both directions?

Server-to-client frames are not masked because the server is a known endpoint that controls its own bytes. The cache poisoning attack required a malicious **client** to craft fake HTTP responses to confuse intermediary proxies. A real server is not crafting fake HTTP responses — it is sending real WebSocket frames. There is no symmetric attack from the server side.

Also, masking costs CPU. Skipping it on the high-volume direction (server-to-client, often broadcasting to thousands of clients at once) saves a lot of cycles.

### A picture

```
     CLIENT                                SERVER
       |                                    |
       |  every payload XORed with          |
       |  a random 4-byte key per frame     |
       |  (the mask key is sent in the      |
       |   frame header, but it changes)    |
       |  ----------- 0x81 0x85 0x37 0xfa   |
       |              0x21 0x3d ........-->|
       |                                    |
       |                                    |
       |  payload sent in the clear          |
       |  (no masking)                      |
       |<-- 0x81 0x05 0x48 0x65 0x6c -------|
       |                                    |
```

Both sides see actual byte sequences. The masking is a safety belt for the client direction only, against a specific attack that is now mostly historical (most modern proxies understand WebSocket and do not try to cache it). But the mask requirement is still in the protocol because removing it would break compatibility, and because not all middleboxes are modern.

## Fragmentation

A "message" in WebSocket can be longer than one frame. The protocol lets you split a message across multiple frames using fragmentation.

### When you would fragment

- **Streaming.** You want to start sending data before you know the full size. Audio recording, file streaming, generating a long message piece by piece.
- **Backpressure.** You have a 100 MB chunk. You do not want to allocate 100 MB on the receive side at once. Send it in 1 MB pieces, the receiver processes each piece.
- **Avoiding head-of-line blocking on small messages.** If you are sending a 50 MB binary blob and a small text message, you can interleave control frames in between fragments of the big one. (Data frames cannot interleave, but ping/pong control frames can.)

### How fragmentation works

The trick is the FIN bit and opcode 0x0 (continuation).

To send a 3-frame message:

```
Frame 1:  FIN=0, opcode=0x1 (text)        payload="Hello, "
Frame 2:  FIN=0, opcode=0x0 (continuation) payload="middle, "
Frame 3:  FIN=1, opcode=0x0 (continuation) payload="and goodbye!"
```

The first frame has the real opcode (text or binary). All later frames have opcode 0x0 (continuation), which means "more of the previous message." The last frame has FIN=1.

The receiver reassembles the three frames into one message: `"Hello, middle, and goodbye!"`.

### Rules for fragmentation

- The first frame must have a **non-zero opcode** (1 or 2).
- All later frames in the same message must have **opcode 0** (continuation).
- The last frame in the message must have **FIN=1**.
- A message in progress can be **interrupted by control frames** (close, ping, pong) but not by another data message.
- A control frame must always be FIN=1 (it cannot be fragmented) and must be ≤125 bytes.

### A picture

```
TIME -->

Sender:    [text "Hi, "]   [cont " how"]   [cont " are"]   [cont " you?" FIN]
                                                                      ^
                                                                      message ends here
Receiver:                                       reassembles "Hi, how are you?"
```

Most simple WebSocket apps never fragment. They just send each message as one frame. But the framework on the receiving side handles fragmentation if it shows up, so you should know it exists.

## Control Frames: Ping, Pong, Close

Three special opcodes are not for your data — they are for managing the connection itself.

### Ping (opcode 0x9) and Pong (opcode 0xA)

These are heartbeats. Either side can send a Ping at any time. The other side **must** reply with a Pong. The Pong's payload must be the same bytes the Ping carried.

Why ping? Two reasons:

- **Liveness.** TCP is supposed to keep connections alive forever, but in reality firewalls, NAT boxes, and load balancers silently kill idle connections after a few minutes. By sending a ping every 30 seconds or so, you keep the connection looking active to all the middleboxes, so they do not close it.
- **Latency measurement.** If you measure the time between sending a Ping and getting the Pong, you have an estimate of the round-trip time. Useful for game clients, dashboards, and so on.

Pings and pongs are control frames, so they are short (≤125 bytes) and unfragmented. They can be inserted between fragments of a long data message without disturbing it.

### Close (opcode 0x8)

When one side wants to hang up, it sends a **Close frame.** The other side replies with its own Close frame. After both Closes have been exchanged, the TCP connection is closed.

A Close frame's payload starts with a 2-byte status code, followed by an optional UTF-8 reason string.

The standard status codes are:

```
1000  Normal Closure         (everything is fine, peace out)
1001  Going Away             (server is shutting down or browser is navigating away)
1002  Protocol Error         (the other side sent something invalid)
1003  Unsupported Data       (got binary when only text expected, or vice versa)
1005  No Status Received     (close frame had no code; not actually sent on the wire)
1006  Abnormal Closure       (connection died without any close frame; not sent on the wire)
1007  Invalid Data           (got bytes that were not valid UTF-8 in a text frame)
1008  Policy Violation       (server-defined: the message broke a policy)
1009  Message Too Big        (payload exceeded server's max size)
1010  Mandatory Extension    (client wanted an extension server didn't agree to)
1011  Internal Server Error  (something blew up on the server side)
1012  Service Restart        (server is restarting; reconnect later)
1013  Try Again Later        (server is overloaded; reconnect later)
1014  Bad Gateway            (gateway problem)
1015  TLS Handshake Failure  (not actually sent on the wire; reported locally)
```

You will see codes 1000, 1001, 1006, and 1011 most often in real apps. 1006 is special — nobody sends it on the wire. It is what your library reports when the TCP connection died without a clean close handshake (network drop, server crash, etc.).

### The close handshake

The expected sequence is:

```
   Side A                       Side B
     |                             |
     |---- Close, 1000 "bye" ----->|
     |                             |
     |<--- Close, 1000 "bye" ------|
     |                             |
     |---- TCP FIN -------------->|
     |<--- TCP ACK + FIN ----------|
```

Both sides send a Close, then TCP closes. If only one side sends a Close, the other side might still be writing for a while. The close handshake is a polite "we are both done" signal.

In real life, sometimes the close handshake doesn't happen. The browser tab gets killed. The Wi-Fi cuts out. A NAT box drops the connection. In those cases the receiving side eventually realizes the TCP connection is dead and reports close code 1006 (abnormal closure). Your client library should handle this gracefully and reconnect.

## WebSocket Subprotocols and Extensions

WebSocket itself is just a framed message channel. It does not say what your messages mean. Two systems can layer their own conventions on top — these are called **subprotocols** and **extensions.**

### Subprotocols (Sec-WebSocket-Protocol)

A subprotocol is an application-level message format. The client lists which ones it speaks. The server picks one. After that, both sides know how to interpret message bytes.

The header looks like:

```
Client request:   Sec-WebSocket-Protocol: graphql-ws, graphql-transport-ws, mqtt
Server response:  Sec-WebSocket-Protocol: graphql-ws
```

Common subprotocols you will encounter:

- **graphql-ws** / **graphql-transport-ws** — the GraphQL Subscriptions protocol. Messages are JSON objects with `{type, payload, id}` fields. Apollo, Hasura, Postgraphile use this.
- **mqtt** / **mqttv3.1** / **mqtt5** — MQTT (a lightweight pub/sub protocol popular in IoT) running over WebSocket. Messages are MQTT binary packets.
- **stomp** / **v10.stomp** / **v12.stomp** — STOMP (Streaming Text Oriented Messaging Protocol), used by message brokers like RabbitMQ. Messages are text commands like `SUBSCRIBE`, `SEND`, `MESSAGE`.
- **wamp.2.json** / **wamp.2.msgpack** — WAMP (Web Application Messaging Protocol). Combines RPC and pub/sub. JSON or MessagePack arrays.
- **ocpp1.6** / **ocpp2.0** — Open Charge Point Protocol, the standard for talking to EV chargers. Many electric car charging stations are WebSocket clients speaking this.
- **xmpp** — Jabber/XMPP over WebSocket.
- **soap** — SOAP messages over WebSocket. (Old-school but still in use in some enterprise stacks.)
- **wamp.2.cbor** — WAMP with CBOR encoding.

Subprotocols matter because the same WebSocket server can host different application logic on the same URL. The server picks behavior based on the negotiated subprotocol.

### Extensions (Sec-WebSocket-Extensions)

Extensions modify the wire format itself. The most important one is **permessage-deflate.**

#### permessage-deflate (compression)

Defined in RFC 7692. The two sides negotiate compression in the handshake:

```
Client:  Sec-WebSocket-Extensions: permessage-deflate; client_max_window_bits
Server:  Sec-WebSocket-Extensions: permessage-deflate; server_max_window_bits=15
```

If both sides agree, every message payload is compressed using DEFLATE (the same algorithm as gzip and zlib) before framing. The RSV1 bit in the frame header is set to 1 to mark a compressed frame.

Compression typically reduces JSON message sizes by 60–80%. For chat, gaming, and dashboards (which send a lot of JSON), this is a big bandwidth and latency win.

The trade-off: compression uses CPU on both sides, and DEFLATE has memory cost (the sliding window). For tiny messages it might not pay off. For most workloads it is worth it.

#### Other extensions (rare)

- **client_no_context_takeover** / **server_no_context_takeover** — modifies how the DEFLATE state is reset between messages.
- **permessage-bzip2** — a non-standard compression extension; not widely supported.
- A multiplexing extension was drafted in the early days but never standardized. HTTP/2 and WebTransport solve multiplexing differently.

## WebSocket Over TLS

WebSocket has two URL schemes:

- **`ws://`** — plain WebSocket. The TCP connection is unencrypted. Default port 80, same as HTTP.
- **`wss://`** — secure WebSocket. The TCP connection is wrapped in TLS, just like HTTPS. Default port 443, same as HTTPS.

In a real browser, most WebSocket connections are `wss://`. Browsers refuse to load `ws://` from `https://` pages (mixed content), and most operators only run TLS-fronted services anyway.

### How wss:// works

The TLS layer is exactly the same as for HTTPS. The browser:

1. Opens a TCP connection to port 443.
2. Performs a TLS handshake (server cert, ALPN, key exchange).
3. After TLS is established, sends the WebSocket HTTP upgrade request **inside the encrypted tunnel.**
4. Receives the 101 response inside the tunnel.
5. From then on, every WebSocket frame travels inside the TLS record layer.

Picture:

```
   +----------------------------------------+
   |   WebSocket frames                     |   <-- application layer
   +----------------------------------------+
   |   TLS records (encrypted)              |   <-- TLS layer (only with wss://)
   +----------------------------------------+
   |   TCP segments                         |   <-- transport layer
   +----------------------------------------+
   |   IP packets                           |   <-- network layer
   +----------------------------------------+
```

For `ws://`, you remove the TLS layer. The frames go straight over TCP.

### Why wss:// is universal in production

- **Firewalls.** Port 443 is open everywhere. Port 80 is increasingly blocked or downgraded.
- **Privacy.** Anybody on the network between you and the server can read `ws://` frames. That is bad if your messages have anything sensitive (chat content, auth tokens, financial data).
- **Integrity.** Without TLS, an attacker on the network could modify frames in flight. TLS prevents tampering.
- **Authentication.** TLS verifies the server's identity via certificates.
- **Modern browsers.** Browsers will not even let `https://` pages open `ws://` connections.
- **HTTP/2 and HTTP/3 require TLS.** WebSocket-over-HTTP/2 (RFC 8441) and WebSocket-over-HTTP/3 (RFC 9220) only happen over TLS.

In short: if you are deploying a WebSocket service, it should be `wss://`. There is no good reason to use `ws://` in production.

### Trying it yourself

Open a TLS connection to a WebSocket server:

```
$ openssl s_client -connect echo.websocket.events:443 -servername echo.websocket.events
```

You will see the TLS handshake output and certificate details. Once connected, you can type the same HTTP upgrade request you would use for plain WebSocket. The TLS layer is invisible at this point — you just type and read like with `curl -i -N`.

## WebSocket and Proxies / Load Balancers

WebSocket runs on top of TCP and starts with HTTP, so it travels through most existing infrastructure. But there are a few things proxies need to do right.

### nginx WebSocket proxying

The classic nginx config for WebSocket:

```
http {
    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }

    server {
        listen 443 ssl http2;
        server_name example.com;

        location /ws {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection $connection_upgrade;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            proxy_read_timeout  3600s;
            proxy_send_timeout  3600s;
        }
    }
}
```

The two key things:

- **`proxy_http_version 1.1`** — WebSocket needs HTTP/1.1 (or HTTP/2 with RFC 8441). Older versions don't support `Upgrade`.
- **`proxy_set_header Upgrade $http_upgrade`** and **`proxy_set_header Connection $connection_upgrade`** — pass the magic upgrade headers through to the backend. nginx normally strips these (they are hop-by-hop), so you have to explicitly forward them.
- **`proxy_read_timeout 3600s`** — by default nginx kills idle connections after 60 seconds. WebSockets are long-lived, so you have to crank this way up. An hour is a common choice.

### HAProxy WebSocket proxying

HAProxy understands WebSocket out of the box, but you should still tune timeouts:

```
frontend https
    bind *:443 ssl crt /etc/haproxy/cert.pem
    timeout client  3600s
    default_backend ws_backend

backend ws_backend
    timeout server  3600s
    timeout tunnel  3600s
    server ws1 10.0.0.1:8080
    server ws2 10.0.0.2:8080
```

The key option is `timeout tunnel`. This is HAProxy's specific timeout for upgraded connections (WebSocket and CONNECT). If you forget it, HAProxy will use the regular `timeout server`, which is shorter.

### Caddy WebSocket proxying

Caddy is the easiest. WebSocket "just works":

```
example.com {
    reverse_proxy /ws/* localhost:8080
}
```

That is it. Caddy automatically handles the upgrade headers. Timeouts default to a long value.

### Traefik

Traefik also auto-supports WebSocket. The relevant config is the `transport.lifeCycle` and per-router `timeouts`. As of recent versions, you should also confirm sticky sessions if you have multiple backends.

### The sticky session problem

If you have many backend servers behind a load balancer, and a client opens a WebSocket, all the **subsequent frames** on that WebSocket go to the **same backend** because they are on the same TCP connection. Good.

But what if the server has crashed and the client reconnects? The load balancer sends them to a different backend. That new backend does not know about the client's previous session, subscriptions, room memberships, anything. Bad.

The fix is **sticky sessions** (also called **session affinity**): the load balancer should try to send the same client to the same backend across reconnections. Common methods:

- **Cookie-based.** The load balancer sets a cookie on the first connection, then routes future connections with that cookie to the same backend.
- **IP-based (source-hash).** Hash the client's IP and route consistently. Falls apart with NAT and mobile clients moving networks.
- **Connection-ID-based.** Some load balancers support persistent connection IDs.

Sticky sessions help, but they are not a silver bullet. The right fix for many WebSocket apps is to keep session state in a shared store (Redis, NATS, Kafka) so any backend can pick up where another left off. We will see this in the scaling section.

### When proxies break WebSockets

Some old proxies (HTTP/1.0-only, or stripped-down corporate filters) do not understand `Upgrade: websocket` and will either drop the connection or rewrite headers in ways that break it. Symptoms:

- The handshake gets back something other than 101.
- The handshake completes but no frames flow.
- The connection closes immediately with code 1006.

In these cases, the only fix is to control the proxy chain. If you cannot, fall back to long polling (libraries like Socket.IO and SignalR do this automatically).

## WebSocket vs Server-Sent Events vs Long-Polling

A quick comparison table of the real-time options:

```
                    Polling        Long Polling   SSE             WebSocket       WebTransport
==========================================================================================
Direction           c -> s         c -> s         s -> c          both            both
                                                                                  + datagrams
Open connections    1 per req      1 (held)       1 long-lived    1 long-lived    1 (multiplexed)
Bytes overhead/msg  hundreds       hundreds       small (text)    tiny (frames)   tiny
Binary support      yes            yes            no (base64)     yes             yes
Reliability         TCP            TCP            TCP             TCP             choice
Reconnect           by client      by client      built-in        by client       by client
Built into browser  yes (fetch)    yes (fetch)    yes (Eventsr.)  yes (WebSocket) yes (modern)
HTTP/2 friendly     yes            yes            yes             RFC 8441        yes (built on)
HTTP/3 friendly     yes            yes            yes             RFC 9220        yes (built on)
Use when            never          fallback       server pushes   bidirectional   next-gen
                                   only           text only       low-latency     gaming/AR/VR
```

The short version:

- **Use WebSocket** when both sides talk a lot, latency matters, and you want binary support. This covers chat, multiplayer games, collaborative editors, live dashboards.
- **Use SSE** when only the server talks and the data is text. Notifications, news tickers, status updates, log streams. SSE is simpler than WebSocket and reconnects automatically.
- **Use long polling** as a fallback for environments where WebSocket and SSE are blocked. Many libraries (Socket.IO, SignalR) do this automatically — they negotiate the best transport at connect time.
- **Use WebTransport** if you need unreliable delivery (game state where stale packets are useless), multiple parallel streams, or 0-RTT connection setup. It is the next-gen replacement, riding on top of QUIC/HTTP/3. Browser support is still rolling out as of 2026.

## WebSocket Security

WebSocket has a few security gotchas. Most are easy to handle once you know they exist.

### Always use wss://

Plain `ws://` traffic is in the clear. Anybody between client and server can read it. Anybody can modify it. In a typical web app, this means your auth tokens, chat messages, and game state are visible to a coffee shop's free Wi-Fi router. Always TLS.

### Validate the Origin header

The browser sends an `Origin: https://example.com` header in the WebSocket handshake. The server must check this. Without an Origin check:

- Any other website can open a WebSocket to your server while your user is logged in. The browser automatically attaches your auth cookie.
- The malicious site can now send arbitrary messages to your server as your user.

This attack is called **Cross-Site WebSocket Hijacking (CSWSH).** The fix is to validate Origin on the server:

```
allowed_origins = {"https://example.com", "https://www.example.com"}

if request.headers["Origin"] not in allowed_origins:
    reject(403)
```

A subtle point: Origin is set by the browser. A non-browser client (like a CLI tool or a malicious server) can send any Origin it likes. So Origin alone is not enough auth. Combine it with token-based auth on the handshake (Cookie, Authorization header, query param token) for full security.

### Authentication options

WebSocket has no built-in auth. Common patterns:

- **Cookie.** Browsers automatically send cookies with the WebSocket handshake (same-origin). Standard session-cookie auth works. Pair with Origin validation against CSWSH.
- **Authorization header.** Browser JavaScript cannot set custom headers on a WebSocket constructor (a quirk of the browser API), but native and server-side clients can. The server validates the bearer token on handshake.
- **Token in URL.** `wss://example.com/ws?token=eyJhbGciOiJI...`. Easy. Works in browsers. But the token shows up in server access logs, which is unsafe. Use short-lived tokens.
- **First-message auth.** Open the WebSocket without auth, then the first message the client sends is a login message. The server rejects and closes if invalid. Cleaner separation but adds a round trip.
- **Ticket auth.** Get a one-time ticket from a REST endpoint (`POST /ws-ticket`), then connect to `wss://example.com/ws?ticket=<one-time>`. The server validates and consumes the ticket. Most secure, no log leak.

### Validate every incoming message

Anything that comes from a WebSocket client is hostile until proven otherwise. Specifically:

- **Validate JSON shape.** Don't blindly trust fields. Use a schema validator.
- **Validate enum values.** Don't pass message types directly to dispatch tables without checking.
- **Sanitize anything you render.** XSS is the same risk as any HTTP body. Escape in the UI.
- **Limit message size.** Set a `max_payload_size` on the server. If a client sends a 10 GB message, you should reject it with close code 1009 (Message Too Big), not allocate 10 GB.
- **Rate-limit messages.** A misbehaving client sending 1 million messages per second can DoS your server. Limit messages per second per connection.
- **Limit connections.** A misbehaving client opening 1 million connections can DoS your server. Limit connections per IP, per user.

### TLS everything (post-quantum note)

Beyond Origin and auth, TLS gives you confidentiality, integrity, and server identity. With TLS 1.3, the WebSocket handshake itself is encrypted, which means an attacker cannot inject malicious extensions or downgrade the subprotocol.

In 2026 we are starting to see hybrid X25519+ML-KEM key exchange for post-quantum security. WebSocket inherits this for free as TLS upgrades — no application change needed.

## Common WebSocket Errors

Errors come in three flavors: HTTP status codes from the handshake, WebSocket close codes, and library-specific exceptions. Here are the ones you will see.

### HTTP handshake errors (before upgrade)

- **`101 Switching Protocols`** — success! The handshake worked. From here on, frames flow.
- **`400 Bad Request`** — the upgrade request was malformed. Missing `Sec-WebSocket-Key`, wrong `Sec-WebSocket-Version`, or the server doesn't accept upgrades on this URL.
- **`401 Unauthorized`** — auth required. Send a valid token/cookie in the handshake.
- **`403 Forbidden`** — auth was provided but is rejected, or Origin failed validation.
- **`404 Not Found`** — wrong URL.
- **`426 Upgrade Required`** — the server requires WebSocket but the client sent a regular HTTP request.
- **`429 Too Many Requests`** — the client is connecting too often.
- **`500 Internal Server Error`** — the server crashed during the handshake.
- **`502 Bad Gateway`** / **`503 Service Unavailable`** / **`504 Gateway Timeout`** — proxy or backend issues.

### WebSocket close codes (after upgrade)

You saw these earlier; here is the cheat list again:

- **`1000`** — normal closure. Either side wanted to hang up. Most common.
- **`1001`** — going away. Server is restarting, or the browser is navigating away.
- **`1002`** — protocol error. The other side sent something invalid (bad opcode, bad masking, etc.).
- **`1003`** — unsupported data. Got binary when text was expected (or vice versa).
- **`1006`** — abnormal closure. The TCP connection died without any close handshake. This is what your library reports for "the network dropped." Very common in mobile apps.
- **`1008`** — policy violation. Server-defined; the message broke a server policy.
- **`1009`** — message too big. Payload exceeded the receiver's max size.
- **`1011`** — internal server error.
- **`1013`** — try again later. Server is overloaded; please reconnect after a delay.

### JavaScript browser errors

You will see these in the browser console:

- **"WebSocket connection to 'wss://...' failed: Invalid frame header"** — the server sent malformed frames. Almost always a server bug.
- **"WebSocket is already in CLOSING or CLOSED state"** — your code tried to send on a closed socket. Always check `ws.readyState === WebSocket.OPEN` before sending.
- **"WebSocket is closed before the connection is established"** — the handshake hadn't finished and your code tried to send too early. Wait for `onopen`.
- **"Failed to construct 'WebSocket': The URL '...' is invalid"** — typo, missing scheme, or an `http://` instead of `ws://`.
- **"Mixed content: The page at 'https://...' was loaded over HTTPS, but attempted to connect to the insecure WebSocket endpoint 'ws://...'."** — you must use `wss://` from `https://` pages. Never `ws://`.
- **"WebSocket disconnected. Reconnecting..."** — your library is retrying. Common after network blips.

### Server-side errors (Python websockets, Node ws, Go gorilla)

- **`ConnectionClosed`** / **`CloseError`** — the connection has closed. Stop trying to send.
- **`PayloadTooBig`** — the peer sent a frame larger than `max_payload_size`. Server should close with 1009.
- **`InvalidHandshake`** — the upgrade headers were malformed. Server should respond 400.
- **`UnsupportedProtocol`** — no agreed subprotocol. Server should close with 1002 or refuse handshake.

## Hands-On

Time to actually try things. You will need a terminal. On most Linux/Mac computers you can open one with the Terminal app or in your editor.

For each command below, type the part after the `$` and press Enter. The lines without `$` are what the computer prints back. Output may differ in unimportant details (timestamps, server names, exact bytes), but the shape should match.

If a command says "command not found," your computer doesn't have that program. That's okay. Move on.

### Hands-On 1: Hand-roll the handshake with curl

```
$ curl -i -N \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 13" \
    -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
    https://echo.websocket.events/ 2>&1 | head -20
HTTP/1.1 101 Switching Protocols
Server: TornadoServer/6.1
Date: Sun, 27 Apr 2026 10:11:12 GMT
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: qGEgH3En71di5rrssAZTmtRTyFk=
```

The `101 Switching Protocols` is success. The handshake is real. `curl` cannot do anything more (it doesn't speak WebSocket frames), but you have just performed step 3-4 of the handshake by hand. Cool.

### Hands-On 2: Verify Sec-WebSocket-Accept by hand

The server must compute `SHA1(client_key + magic_GUID)` and base64-encode it. Try it:

```
$ echo -n "SGVsbG8sIHdvcmxkIQ==258EAFA5-E914-47DA-95CA-C5AB0DC85B11" | openssl dgst -sha1 -binary | base64
qGEgH3En71di5rrssAZTmtRTyFk=
```

That matches the server's response from Hands-On 1. The math is real. The magic GUID is hard-coded into every WebSocket implementation in the world.

### Hands-On 3: Generate a fresh Sec-WebSocket-Key

The client is supposed to generate a fresh random 16-byte key, base64-encoded:

```
$ openssl rand -base64 16
KuLp2tbU3wEMD0YkPIxK+Q==
```

Run it again and you get a different one. Run it a thousand times and never see a duplicate.

### Hands-On 4: Connect with wscat (interactive WebSocket client)

If you have `wscat` (npm install -g wscat), you can chat with a server:

```
$ wscat -c wss://echo.websocket.events/
Connected (press CTRL+C to quit)
> hello
< Echo from server: hello
> can you hear me
< Echo from server: can you hear me
```

The `>` lines are what you typed; the `<` lines are what the server echoed back. Press Ctrl-C to disconnect. This is the simplest way to play with a real WebSocket.

### Hands-On 5: Connect with websocat (alternative CLI)

`websocat` is similar to `wscat` but written in Rust, single binary, no Node needed:

```
$ websocat wss://echo.websocket.events/
hello
Echo from server: hello
test 123
Echo from server: test 123
```

### Hands-On 6: Send one message from a Python one-liner

If you have Python and the `websockets` library:

```
$ python3 -c "
import asyncio, websockets
async def main():
    async with websockets.connect('wss://echo.websocket.events/') as ws:
        await ws.send('hi from python')
        print(await ws.recv())
asyncio.run(main())
"
Echo from server: hi from python
```

That's the smallest possible WebSocket Python program. Connect, send, receive, exit.

### Hands-On 7: Same thing from Node.js

If you have Node and the `ws` package (`npm i ws`):

```
$ node -e "const W=require('ws');const w=new W('wss://echo.websocket.events/');w.on('open',()=>w.send('hi from node'));w.on('message',m=>{console.log(m.toString());w.close();});"
Echo from server: hi from node
```

### Hands-On 8: Run a tiny Python server

Save this as `server.py`:

```
import asyncio, websockets

async def handler(ws):
    async for msg in ws:
        await ws.send(f"You said: {msg}")

async def main():
    async with websockets.serve(handler, "localhost", 8765):
        await asyncio.Future()

asyncio.run(main())
```

Then in one terminal:

```
$ python3 server.py
```

In another terminal:

```
$ wscat -c ws://localhost:8765
Connected (press CTRL+C to quit)
> hi
< You said: hi
```

You just ran your own WebSocket server. Press Ctrl-C in the first terminal to stop it.

### Hands-On 9: Capture WebSocket traffic with tcpdump

In one terminal, run a capture:

```
$ sudo tcpdump -i any -n -A -s 0 'tcp port 80 or tcp port 8765' | head -50
```

In another, send a `ws://` connection. You should see the HTTP upgrade in the dump (in cleartext for `ws://`).

For `wss://` you cannot read the contents (TLS encrypts), but you can still see the connection sizes and timing.

### Hands-On 10: Inspect WebSocket frames with tshark

Wireshark's CLI sibling decodes WebSocket if it can see plaintext (i.e., `ws://`):

```
$ sudo tshark -i any -Y 'websocket' -V 2>/dev/null
```

Run this, then connect to a `ws://` server with `wscat`. You should see frames decoded with `Fin: True`, `Opcode: 1 (Text)`, `Mask: True`, `Masking-Key: ...`, `Payload`. Genuinely magic to watch.

### Hands-On 11: List established TCP connections

To see WebSocket connections from your machine (or a server), look at long-lived TCP:

```
$ ss -tnpa | head -10
State      Recv-Q  Send-Q  Local Address:Port  Peer Address:Port  Process
ESTAB      0       0       192.168.1.10:54321  140.82.121.3:443   users:(("firefox",pid=3127,fd=72))
ESTAB      0       0       192.168.1.10:54322  104.16.249.89:443  users:(("slack",pid=4112,fd=18))
LISTEN     0       128     0.0.0.0:22          0.0.0.0:*          users:(("sshd",pid=1234,fd=3))
```

Long-running ESTAB connections to port 443 from your browser are very likely WebSocket (the rest are HTTP/2 multiplexed for short bursts).

### Hands-On 12: Filter ss to only port 443 established

```
$ ss -ti dst :443 state established | head -5
ESTAB    0    0    192.168.1.10:54321    140.82.121.3:443     cubic wscale:7,7 rto:204 rtt:1.812/0.654 ato:40 mss:1448 ...
```

The `-i` flag adds congestion-control info per connection. `cubic` is the default Linux algorithm. `rtt:1.812/0.654` is round-trip time and variance in milliseconds.

### Hands-On 13: Verify nginx upgrade map

If you run nginx and have a WebSocket route, the upgrade map should be in the config:

```
$ sudo nginx -T 2>&1 | grep -A 5 -i upgrade
    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_http_version 1.1;
```

If the `map` block is missing, your nginx will not pass `Upgrade` headers correctly.

### Hands-On 14: Check Caddy WebSocket config

```
$ caddy adapt --config /etc/caddy/Caddyfile 2>/dev/null | head -40
{
    "apps": {
        "http": {
            "servers": {
                "srv0": {
                    "listen": [":443"],
                    "routes": [
                        {
                            "match": [{"host": ["example.com"]}],
                            "handle": [
                                {
                                    "handler": "reverse_proxy",
                                    "upstreams": [{"dial": "localhost:8080"}]
                                }
                            ]
                        }
                    ]
                }
            }
        }
    }
}
```

Caddy auto-handles upgrade for any `reverse_proxy`, no special syntax needed.

### Hands-On 15: Run a load test with wrk

```
$ wrk -c 100 -t 4 -d 30s --latency wss://echo.websocket.events/ 2>&1 | head -20
Running 30s test @ wss://echo.websocket.events/
  4 threads and 100 connections
  ... (output depends on wrk's WebSocket support; vanilla wrk only does HTTP)
```

Note: stock `wrk` does not speak WebSocket. For real WebSocket load testing, use `tsung`, `artillery`, `k6`, or `locust`. The example here mostly demonstrates the connection limit.

### Hands-On 16: Check the OpenSSL TLS handshake

```
$ openssl s_client -connect echo.websocket.events:443 -servername echo.websocket.events < /dev/null 2>&1 | head -20
CONNECTED(00000003)
depth=2 C = US, O = Internet Security Research Group, CN = ISRG Root X1
verify return:1
depth=1 C = US, O = Let's Encrypt, CN = R3
verify return:1
depth=0 CN = echo.websocket.events
verify return:1
---
Certificate chain
 0 s:CN = echo.websocket.events
   i:C = US, O = Let's Encrypt, CN = R3
```

This shows the TLS chain. WebSocket-over-TLS uses the same TLS as HTTPS — same certs, same chain validation.

### Hands-On 17: Listen on a port with netcat

You can run a fake server with `nc -lk 8080` and connect a browser DevTools console to it. The first thing the browser sends will be the WebSocket HTTP upgrade. You will see it in `nc`:

```
$ nc -lk 8080
GET /ws HTTP/1.1
Host: localhost:8080
Connection: Upgrade
Pragma: no-cache
Cache-Control: no-cache
Upgrade: websocket
Origin: http://localhost
Sec-WebSocket-Version: 13
User-Agent: Mozilla/5.0 ...
Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
Sec-WebSocket-Extensions: permessage-deflate; client_max_window_bits
```

`nc` doesn't reply with the right `Sec-WebSocket-Accept`, so the handshake fails — but you can see exactly what a real browser sends.

### Hands-On 18: Tail nginx logs for WebSocket upgrades

```
$ sudo journalctl -u nginx -f | grep -i upgrade
```

This stays open and prints any new WebSocket upgrade requests in real time as users connect.

### Hands-On 19: Look at TCP details in dmesg

```
$ dmesg | grep -i tcp | tail -10
[ 1234.567890] TCP: cubic registered
[ 1234.568123] tcp_tw_recycle removed in kernel 4.12
[ 5678.234567] TCP: out of memory -- consider tuning tcp_mem
```

These are kernel-level TCP messages. On a busy WebSocket server, you might see warnings about exhausted buffers or congestion.

### Hands-On 20: Read raw TCP table

```
$ cat /proc/net/tcp | head -3
  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode
   0: 0100007F:1F90 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 12345
   1: 0100007F:13B7 00000000:0000 0A 00000000:00000000 00:00000000 00000000     0        0 23456
```

Ports are in hex. `01BB` = 0x01BB = 443. `0050` = 0x50 = 80. The `st` column is TCP state in hex (`01` = ESTABLISHED, `0A` = LISTEN). To find established connections to port 443:

```
$ cat /proc/net/tcp | grep -E ":01BB.*01" | wc -l
137
```

This server has 137 established TCP connections to port 443. (HTTPS + WebSocket combined; you can't tell them apart at this layer.)

### Hands-On 21: Find Upgrade configs in nginx

```
$ grep -r "Upgrade" /etc/nginx/ 2>/dev/null | head -10
/etc/nginx/conf.d/ws.conf:    proxy_set_header Upgrade $http_upgrade;
/etc/nginx/conf.d/ws.conf:    map $http_upgrade $connection_upgrade {
```

Confirms which files configure the upgrade dance.

### Hands-On 22: Find WebSocket configs in HAProxy

```
$ grep -r "ws_proxy\|tunnel" /etc/haproxy/ 2>/dev/null
/etc/haproxy/haproxy.cfg:    timeout tunnel 3600s
/etc/haproxy/haproxy.cfg:    use_backend ws_backend if path_ws
```

`timeout tunnel` is the WebSocket-specific timeout. If it's missing, your WebSockets get killed at the regular `timeout server` limit.

### Hands-On 23: Check Traefik version and WebSocket help

```
$ traefik version
Version:      3.3.5
Codename:     beaufort
Go version:   go1.22.0
Built:        2026-01-15
OS/Arch:      linux/amd64

$ traefik help 2>&1 | grep -i ws
... (Traefik handles WebSocket automatically; no specific flag)
```

Modern Traefik does not need explicit WebSocket configuration.

### Hands-On 24: Use --include and --no-buffer on curl

An alternative form of the handshake test:

```
$ curl --include --no-buffer \
    --header "Connection: Upgrade" \
    --header "Upgrade: websocket" \
    --header "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
    --header "Sec-WebSocket-Version: 13" \
    "http://echo.websocket.events/" 2>&1 | head -10
HTTP/1.1 101 Switching Protocols
Server: TornadoServer/6.1
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Accept: qGEgH3En71di5rrssAZTmtRTyFk=
```

Same thing, with long-form flag names. Both forms work.

### Hands-On 25: Send and receive in a single shell

You can keep an `nc` open as a fake WebSocket server and watch what a browser does:

```
$ printf 'HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: qGEgH3En71di5rrssAZTmtRTyFk=\r\n\r\n' | nc -l 8080
```

Connect from a browser DevTools console:

```javascript
new WebSocket('ws://localhost:8080/');
```

The browser sends a request with `Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==`, your fake server replies with the right `Sec-WebSocket-Accept`, and the browser thinks it has a WebSocket. Now send a frame from JavaScript:

```javascript
ws.send('hi');
```

You will see the masked bytes in `nc` output. Decoding them by hand is left as an exercise — the masking key is in bytes 3-6 of the frame, and the payload is XORed with that key cycling.

### Hands-On 26: Check the Sec-WebSocket-Version your server speaks

Try a wrong version:

```
$ curl -i -N \
    -H "Connection: Upgrade" \
    -H "Upgrade: websocket" \
    -H "Sec-WebSocket-Version: 8" \
    -H "Sec-WebSocket-Key: SGVsbG8sIHdvcmxkIQ==" \
    https://echo.websocket.events/ 2>&1 | head -10
HTTP/1.1 400 Bad Request
Sec-WebSocket-Version: 13
```

The server tells you what versions it accepts. Almost every server in the world only speaks 13.

### Hands-On 27: Watch Linux network counters

```
$ cat /proc/net/snmp | grep Tcp:
Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens ...
Tcp: 1 200 120000 -1 134567 89234 1023 17 100234 ...
```

`PassiveOpens` is the number of incoming TCP connections accepted. If you are running a busy WebSocket server, this counter ticks up fast.

### Hands-On 28: List network interfaces and rates

```
$ ip -s link show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 ...
    RX: bytes  packets  errors  dropped  overrun  mcast
    1234567    8901     0       0        0        0
    TX: bytes  packets  errors  dropped  carrier  collsns
    1234567    8901     0       0        0        0

2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
```

Watch RX/TX byte counters tick up when WebSocket traffic flows.

## Common Confusions

These come up over and over. If you see a confusing thing in real code, it is probably one of these.

### Confusion 1: "WebSocket runs on port 80/443 the same as HTTP"

**Broken thinking.** People say "WebSocket has its own protocol, it must have its own port." Wrong.

**Fixed thinking.** WebSocket reuses port 80 (for `ws://`) and port 443 (for `wss://`). The reason is that `ws://` starts as plain HTTP and `wss://` starts as HTTPS. After the upgrade handshake, the same TCP connection just keeps going with WebSocket frames instead of HTTP messages. No new port. Same firewall rules. Easy deployment.

### Confusion 2: "Sec-WebSocket-Key is a security secret"

**Broken thinking.** "It's called Sec-WebSocket-Key, so it must be like a password."

**Fixed thinking.** It is **not** a secret. Anybody on the network can read it. It is just a randomly-generated nonce used to prove that the server actually understood the upgrade request (by computing the matching `Sec-WebSocket-Accept`). It does not authenticate anything. For real auth use cookies, bearer tokens, or tickets.

### Confusion 3: "WebSocket is over UDP"

**Broken thinking.** "WebSocket is real-time, so it must be UDP, like for games."

**Fixed thinking.** WebSocket runs on **TCP.** It inherits TCP's reliability, ordering, and congestion control. It also inherits TCP's head-of-line blocking. For UDP-style WebSocket-y traffic, look at **WebTransport**, which runs on QUIC (a UDP-based protocol). WebTransport is newer and not as widely supported.

### Confusion 4: "Once I open a WebSocket I can send anything I want"

**Broken thinking.** "It's a stream, I'll just send raw bytes."

**Fixed thinking.** WebSocket carries **frames**, not raw bytes. Each `ws.send(...)` produces one or more frames. Each frame has a header that specifies whether the payload is text (UTF-8) or binary. Receivers decode the frame and deliver a complete message to the app. You don't see byte-level streaming; you see message-level send/receive.

### Confusion 5: "Server-side frames need to be masked too"

**Broken thinking.** "Symmetry — both sides must do the same thing."

**Fixed thinking.** Only client-to-server frames are masked. Server-to-client frames are not. If a server masks its frames, a strict client should reject them with close code 1002. The reason masking exists at all is the cache-poisoning attack on transparent HTTP proxies, which only applies to the client direction.

### Confusion 6: "I'll just send big messages, the protocol handles fragmentation"

**Broken thinking.** "Fragmentation is automatic; I send a 100 MB message and the protocol splits it."

**Fixed thinking.** The **library** may auto-fragment (or not) — RFC 6455 allows but does not require fragmentation. Some libraries always send one big frame; others fragment at a configurable threshold. If you are sending big messages, set explicit limits and consider streaming through the library's streaming API rather than `send()`-ing 100 MB at once. Also: the receiver has a `max_payload_size` limit that, if exceeded, closes the connection with 1009.

### Confusion 7: "WebSocket is multiplexed like HTTP/2"

**Broken thinking.** "I can have many logical streams over one WebSocket."

**Fixed thinking.** A WebSocket is **one logical channel**. Each connection carries a single bidirectional message stream. If you want multiple streams, either (a) open multiple WebSockets, (b) layer your own multiplexing (most apps add a `streamID` field to each JSON message), or (c) use HTTP/2 (RFC 8441) or HTTP/3 (RFC 9220) to multiplex many WebSockets over one TCP/QUIC connection, or (d) use WebTransport (which has multiplexed streams natively).

### Confusion 8: "Close 1006 means the server crashed"

**Broken thinking.** "1006 is in the spec, so the server must have sent it."

**Fixed thinking.** Code 1006 is a **local-only** code. Nobody actually transmits it on the wire. Your client library reports 1006 when the TCP connection died **without** a close handshake — for example, if Wi-Fi cut out, or a load balancer killed the connection, or a NAT timeout fired. It does not mean "the server is broken." It means "we lost the connection abruptly."

### Confusion 9: "I should disable masking for performance"

**Broken thinking.** "XOR per byte costs CPU; if I'm sending tons of data I should turn it off."

**Fixed thinking.** You **cannot** disable masking on the client side. The protocol requires it. Servers will (and should) reject unmasked client frames with close 1002. If masking CPU is really your bottleneck, you are doing something specialized (like running a high-frequency trading client) and should consider raw TCP, UNIX sockets, or a different protocol entirely.

### Confusion 10: "WebSocket needs cookies"

**Broken thinking.** "All my auth is cookie-based; I'll just send cookies in the handshake."

**Fixed thinking.** Cookies work and are common, but they have a downside: any page in the browser can open a WebSocket with the user's cookies attached automatically. This is the CSWSH attack. To use cookies safely you must validate the `Origin` header on the server. For higher security, prefer ticket-based auth (one-time token from a REST endpoint, sent in the handshake URL) or first-message JWT.

### Confusion 11: "WebSocket replaces REST"

**Broken thinking.** "Why would I use HTTP? I'll just put everything on a WebSocket."

**Fixed thinking.** WebSocket is **bad** for one-shot CRUD. HTTP is great at "ask, get, done" semantics, has caching, has rich tooling (curl, Postman, browser DevTools), maps cleanly to URLs and verbs, and fits load balancers without sticky sessions. WebSocket is great for **streaming, bidirectional, real-time**. Use both. Most modern apps have a REST/GraphQL API and a WebSocket for real-time updates.

### Confusion 12: "Socket.IO is the same as WebSocket"

**Broken thinking.** "I read about Socket.IO; it's just a Node WebSocket library."

**Fixed thinking.** **Socket.IO is not raw WebSocket.** It is a layer on top: it adds rooms, namespaces, automatic reconnection, fallback to long-polling, and a custom message envelope. A vanilla WebSocket client cannot connect to a Socket.IO server, and vice versa. Same for **SignalR** (Microsoft's), **WAMP**, **Centrifugo's protocol**, and many others. They are **WebSocket-using** libraries with their own wire format on top.

### Confusion 13: "Long-lived connections leak file descriptors"

**Broken thinking.** "If I have a million WebSockets I'll run out of file descriptors and crash."

**Fixed thinking.** You will hit the per-process file-descriptor limit if you do not raise it. Linux defaults to 1024 per process. For a WebSocket server, raise it: `ulimit -n 1048576` and configure systemd's `LimitNOFILE`. Modern Linux can handle millions of open sockets per machine if you tune it. You also need to tune TCP buffer sizes and ephemeral port ranges. The operational pattern matters; "leak" implies a bug, but most "leaks" are just hitting limits set for normal HTTP.

### Confusion 14: "I can use WebSocket without TLS internally"

**Broken thinking.** "Inside my private network, `ws://` is fine."

**Fixed thinking.** Internal networks can be hostile (insider threats, lateral movement after a breach, untrusted multi-tenant clusters). Best practice is `wss://` everywhere, terminated at a service mesh proxy if you don't want to manage certs in every app. Cost is small; benefit is significant.

## Vocabulary

Every weird word in this sheet, with a one-line plain-English definition.

- **WebSocket** — a permanent, two-way, framed message channel between a client and a server, opened by upgrading an HTTP connection.
- **ws** — the URL scheme for plain (unencrypted) WebSocket; uses TCP port 80.
- **wss** — the URL scheme for TLS-wrapped WebSocket; uses TCP port 443. Always use this in production.
- **frame** — the unit of data on a WebSocket connection; a small header followed by a payload.
- **opcode** — the 4-bit field in a frame header that says what kind of frame it is (text, binary, close, ping, pong, continuation).
- **FIN bit** — 1 if this is the last frame of a message, 0 if more frames are coming.
- **mask bit** — 1 if the payload is XOR-masked with a key (always for client→server), 0 otherwise.
- **masking key** — a random 4-byte number generated per frame that the client uses to scramble its payload.
- **payload** — the actual message bytes carried inside a frame.
- **control frame** — a frame that manages the connection (close, ping, pong); always ≤125 bytes, never fragmented.
- **data frame** — a frame that carries application data (text or binary).
- **text frame** — a data frame whose payload is UTF-8 text. Opcode 0x1.
- **binary frame** — a data frame whose payload is raw bytes. Opcode 0x2.
- **ping** — a control frame that asks "are you alive?" Opcode 0x9.
- **pong** — a control frame that replies "yes I am." Opcode 0xA.
- **close** — a control frame that starts the shutdown handshake. Opcode 0x8.
- **status code** — the 2-byte number at the start of a Close frame's payload (1000, 1001, etc.).
- **handshake** — the initial HTTP exchange that upgrades a connection from HTTP to WebSocket.
- **Upgrade header** — the HTTP header `Upgrade: websocket` that requests an upgrade.
- **Connection: Upgrade** — the HTTP header that confirms the upgrade request applies to this connection.
- **Sec-WebSocket-Key** — the random base64 nonce sent by the client in the handshake.
- **Sec-WebSocket-Accept** — the server's reply derived from the key plus the magic GUID, base64-encoded SHA-1.
- **magic GUID** — the fixed string `258EAFA5-E914-47DA-95CA-C5AB0DC85B11`, defined by RFC 6455, used in computing Sec-WebSocket-Accept.
- **Sec-WebSocket-Protocol** — the handshake header negotiating an application subprotocol like `graphql-ws`.
- **Sec-WebSocket-Version** — the handshake header negotiating the WebSocket protocol version. Always 13.
- **Sec-WebSocket-Extensions** — the handshake header listing extensions like `permessage-deflate`.
- **permessage-deflate** — the compression extension defined in RFC 7692. Compresses message payloads with DEFLATE.
- **fragmentation** — splitting a single message across multiple frames, used for streaming and backpressure.
- **subprotocol** — an application-level message format negotiated during the handshake (graphql-ws, mqtt, stomp, wamp).
- **RFC 6455** — the original 2011 WebSocket Protocol specification.
- **RFC 7692** — the 2015 spec for compression extensions (permessage-deflate).
- **RFC 8441** — the 2018 spec for bootstrapping WebSockets over HTTP/2.
- **RFC 9220** — the 2022 spec for bootstrapping WebSockets over HTTP/3.
- **STOMP** — Streaming Text Oriented Messaging Protocol; a text-based pub/sub used over WebSocket.
- **WAMP** — Web Application Messaging Protocol; combines RPC and pub/sub.
- **SignalR** — Microsoft's real-time framework that uses WebSocket as a transport with extra features.
- **Socket.IO** — a popular JavaScript library that uses WebSocket as a transport, with rooms, fallback, auto-reconnect. Not raw WebSocket.
- **graphql-ws** — the subprotocol that ferries GraphQL subscriptions over WebSocket.
- **MQTT-over-WebSocket** — running the MQTT pub/sub protocol inside WebSocket frames; common in IoT and web dashboards.
- **AMQP-WebSocket** — running AMQP (the RabbitMQ protocol) inside WebSocket frames.
- **EventSource** — the JavaScript class for Server-Sent Events.
- **SSE** — Server-Sent Events; a one-way server-to-client text streaming protocol over HTTP.
- **long polling** — a workaround where the server holds an HTTP request open until data arrives, then responds.
- **server-sent events** — see SSE.
- **CSWSH** — Cross-Site WebSocket Hijacking; an attack where a malicious site opens a WebSocket using the victim's cookies.
- **Origin header** — the HTTP header that says which origin a request came from. Validated server-side to prevent CSWSH.
- **hijack** — taking over an existing session; in CSWSH, hijacking the user's authenticated WebSocket.
- **heartbeat** — a periodic ping/pong to keep a connection alive and detect stale connections.
- **idle timeout** — the time after which a middle-box (proxy, load balancer, NAT) drops a connection that has no traffic.
- **sticky session** — a load-balancer feature that routes the same client to the same backend across reconnects.
- **session affinity** — synonym for sticky session.
- **proxy buffer** — memory the proxy uses to hold message data; can interfere with WebSocket if too large.
- **upgrade timeout** — the time the proxy waits for the upgrade response; must be long enough for slow servers.
- **NAT keepalive** — periodic packets to prevent NAT translation from expiring on idle connections.
- **TCP keepalive** — kernel-level periodic packets via `SO_KEEPALIVE` to detect dead connections.
- **SO_KEEPALIVE** — the socket option that enables TCP keepalive.
- **close code 1000** — Normal Closure. Standard "we're done" code.
- **close code 1001** — Going Away. Server is shutting down or browser is navigating.
- **close code 1002** — Protocol Error. Peer sent something invalid.
- **close code 1003** — Unsupported Data. Got binary when text expected (or vice versa).
- **close code 1006** — Abnormal Closure. Connection died with no close frame; reported locally.
- **close code 1009** — Message Too Big. Payload exceeded receiver's max size.
- **close code 1011** — Internal Server Error.
- **close code 1013** — Try Again Later. Server is overloaded.
- **abnormal closure** — when the TCP connection dies without a clean WebSocket close handshake; reported as 1006.
- **graceful close** — both sides exchange Close frames before TCP shuts down.
- **half-close** — a TCP state where one direction is closed but the other is still open. WebSocket treats this as full close.
- **binary message** — a message whose payload is raw bytes (not text).
- **text message** — a message whose payload is valid UTF-8 text.
- **message boundary** — the FIN bit; tells the receiver that a complete message has finished arriving.
- **JSON-WS** — informal term for using JSON-encoded text frames as the application format on top of WebSocket.
- **framing layer** — the part of WebSocket that turns a TCP byte stream into discrete framed messages.
- **echo server** — a WebSocket (or other) server that replies with whatever the client sent. `echo.websocket.events` is a public one.
- **WebTransport** — the next-generation web API for bidirectional, multiplexed, possibly-unreliable streams over QUIC/HTTP/3.
- **QUIC** — a UDP-based transport protocol underneath HTTP/3. Solves TCP's head-of-line blocking.
- **HTTP/2** — the second major version of HTTP, supports multiplexing many streams over one TCP connection.
- **HTTP/3** — the third major version of HTTP, runs on QUIC instead of TCP.
- **head-of-line blocking** — when a lost packet stalls every later packet because of in-order delivery; inherent to TCP.
- **bufferbloat** — large network buffers cause sustained latency spikes; bad for real-time apps.
- **TCP_NODELAY** — socket option that disables Nagle's algorithm (small-packet batching), improves latency for tiny messages.
- **TCP_NOTSENT_LOWAT** — Linux 3.12+ socket option that controls write-readiness based on unsent data, reduces buffering latency.
- **CONNECT method** — the HTTP method used to bootstrap WebSocket over HTTP/2 (RFC 8441) and HTTP/3 (RFC 9220).
- **:protocol pseudo-header** — the HTTP/2 pseudo-header set to `websocket` for upgrade-over-HTTP/2.
- **SETTINGS_ENABLE_CONNECT_PROTOCOL** — the HTTP/2 setting that lets a server signal it accepts WebSocket-over-HTTP/2.
- **101 Switching Protocols** — the HTTP status code returned by the server to confirm a successful upgrade.
- **426 Upgrade Required** — the HTTP status code that says "I require a protocol upgrade for this URL."
- **DEFLATE** — the compression algorithm used by gzip, zlib, and `permessage-deflate`. Lossless general-purpose compressor.
- **back-pressure** — slowing down the sender when the receiver cannot keep up.
- **send buffer** — the OS-side memory holding bytes between `send()` and actual network transmission.
- **recv buffer** — the OS-side memory holding incoming bytes until the application reads them.
- **wscat** — a command-line WebSocket client written in Node, useful for testing.
- **websocat** — a command-line WebSocket client written in Rust, single-binary, useful for testing.
- **Tornado** — a Python web framework with built-in WebSocket support.
- **gorilla/websocket** — the most popular Go WebSocket library.
- **ws (npm)** — the most popular Node.js WebSocket library.
- **websockets (PyPI)** — the most popular Python asyncio WebSocket library.
- **autobahn** — a WebSocket protocol conformance test suite, runnable against any implementation.
- **TLS 1.3** — the modern Transport Layer Security version that wraps `wss://` connections.
- **certificate** — the X.509 document a TLS server presents to prove its identity. Same as for HTTPS.
- **ALPN** — Application-Layer Protocol Negotiation; the TLS extension that lets server and client agree on `http/1.1`, `h2`, or `h3` during the handshake.

## Try This

Pick at least three. The point is to get hands on the wire, not just read about it.

### Try This 1: Hand-roll a handshake

Run the curl handshake from Hands-On 1. Then do the SHA-1 + base64 computation by hand from Hands-On 2. Match the server's `Sec-WebSocket-Accept` to your computed value. You will never forget what the magic GUID does after this.

### Try This 2: Echo with wscat

Connect to `wss://echo.websocket.events/` with `wscat`. Type messages. Watch them echo. Press Ctrl-C. Now connect again with `websocat` and do the same. Notice that one is Node-based and the other is Rust-based; both speak the same protocol because the protocol is the standard, not the implementation.

### Try This 3: Inspect frames in Wireshark / tshark

Run `tshark -i any -Y 'websocket' -V` (or open Wireshark, set the filter to `websocket`) and watch real WebSocket frames decode. Connect to a `ws://` server (most easy: `ws://echo.websocket.events/` falls back to `ws://` if you ask it to). Send messages and watch every frame: opcode, FIN, mask, masking key, payload. Try sending a long message that gets fragmented; you will see opcode-0 continuation frames.

### Try This 4: Run your own server

Save the Python `server.py` from Hands-On 8 and run it. Connect with `wscat -c ws://localhost:8765`. You now have a real WebSocket service on your own machine. Modify the handler to add some logic — store messages, broadcast to all connected clients, anything — and reload.

### Try This 5: Force a close code

Modify the server to call `await ws.close(1011, 'simulated crash')` after one second. Watch your client see the 1011 close code. Then change to 1009 ("Message Too Big"), 1013 ("Try Again Later"), and so on. Each time, observe how your client library surfaces the code.

### Try This 6: Watch the OS see TCP

Run `ss -tnpa` while you have a WebSocket open in your browser. Find the long-running ESTAB connection on port 443. Note the local port. Now use `lsof -i :<that port>` to see exactly which process owns it. You will see your browser holding it open.

### Try This 7: Break the handshake on purpose

Send the curl handshake but with `Sec-WebSocket-Version: 8`. The server will respond `400 Bad Request`. Send it without `Sec-WebSocket-Key`. The server will respond `400 Bad Request`. Send it with `Sec-WebSocket-Version: 13` but no `Upgrade` header. The server will treat it as a normal HTTP request. Each broken case is a learning opportunity.

### Try This 8: Inspect a real chat app

Open Discord/Slack in your browser. Open DevTools → Network → WS tab. Find the WebSocket connection. Watch frames arrive in real time as messages flow. Each user typing, each reaction, each heartbeat. The ping/pong frames keep firing every 30 seconds or so to keep the connection alive.

## Where to Go Next

Once this sheet feels easy, the dense engineer-grade material is one command away. Stay in the terminal:

- **`cs networking websocket`** — the dense reference. Real names for every header, every field, every state, every library option.
- **`cs detail networking/websocket`** — the formal underpinning. Frame format math, masking analysis, head-of-line blocking quantitative analysis, eBPF-accelerated WebSocket processing.
- **`cs networking http`** — HTTP/1.1, the protocol that hosts the WebSocket handshake.
- **`cs networking http2`** — HTTP/2, with WebSocket-over-HTTP/2 (RFC 8441).
- **`cs networking http3`** — HTTP/3, with WebSocket-over-HTTP/3 (RFC 9220).
- **`cs networking quic`** — QUIC, the UDP-based transport under HTTP/3 and WebTransport.
- **`cs networking tcp`** — TCP, what WebSocket actually rides on.
- **`cs ramp-up tcp-eli5`** — the friendly version of TCP.
- **`cs ramp-up tls-eli5`** — the friendly version of TLS, what wraps `wss://`.
- **`cs ramp-up http3-quic-eli5`** — the friendly version of HTTP/3 and QUIC.
- **`cs ramp-up linux-kernel-eli5`** — what is actually running this stuff under the hood.
- **`cs web-servers nginx`**, **`cs web-servers caddy`**, **`cs web-servers haproxy`** — proxy WebSocket configuration recipes.
- **`cs security tls`** — TLS internals.

## See Also

- `networking/websocket` — engineer-grade reference for WebSocket.
- `networking/http` — HTTP/1.1.
- `networking/http2` — HTTP/2.
- `networking/http3` — HTTP/3.
- `networking/quic` — QUIC transport.
- `networking/tcp` — TCP.
- `web-servers/nginx` — nginx WebSocket proxy config.
- `web-servers/caddy` — Caddy WebSocket proxy config.
- `web-servers/haproxy` — HAProxy WebSocket proxy config.
- `security/tls` — TLS that wraps `wss://`.
- `ramp-up/tcp-eli5` — friendly TCP intro.
- `ramp-up/tls-eli5` — friendly TLS intro.
- `ramp-up/ip-eli5` — friendly IP intro.
- `ramp-up/linux-kernel-eli5` — friendly Linux kernel intro.

## References

- **RFC 6455** — The WebSocket Protocol (December 2011). The foundational spec. Read this once and you will know more than 95% of working developers.
- **RFC 7692** — Compression Extensions for WebSocket (`permessage-deflate`). The standard message-level compression.
- **RFC 8441** — Bootstrapping WebSockets with HTTP/2 (September 2018). Defines the CONNECT-with-`:protocol`-pseudo-header upgrade for HTTP/2.
- **RFC 9220** — Bootstrapping WebSockets with HTTP/3 (June 2022). Same idea on HTTP/3 / QUIC.
- **`man wscat`** — manual page for the wscat command-line client (after `npm install -g wscat`).
- **`man websocat`** — manual page for the websocat Rust-based command-line client.
- **echo.websocket.events** — public WebSocket echo server for testing. Sends back whatever you send.
- **Autobahn|Testsuite** — protocol conformance test suite. Run it against any WebSocket implementation to verify spec compliance.
- **"High Performance Browser Networking"** by Ilya Grigorik — Chapter 17 covers WebSocket in depth, with diagrams, performance discussion, and comparison to alternatives.
- **The MDN WebSocket reference** — accessible via `cs networking websocket` (mirrored locally).
- **kernel.org Documentation/networking/** — TCP tuning and socket options that matter for WebSocket servers.

— End of ELI5 —

When this sheet feels easy, graduate to **`cs networking websocket`** for the engineer-grade reference and **`cs detail networking/websocket`** for the academic underpinning. By the time you have read both, you will be reading WebSocket library source code without flinching, debugging real production WebSocket issues at the wire level, and arguing about whether RFC 8441 is the right thing for your use case.

### One last thing before you go

Pick one command from the Hands-On section that you have not run yet. Run it right now. Read the output. Try to figure out what each part means using the Vocabulary table as your dictionary. Don't just trust this sheet — see for yourself. WebSockets are real. They are running on every chat app, every live dashboard, every multiplayer game, right now. The commands in this sheet let you peek at them.

Reading is good. Doing is better. Type the commands. Watch the frames flow.

You are now officially started on your WebSocket journey. Welcome.

The whole point of the North Star for the `cs` tool is: never leave the terminal to learn this stuff. Everything you need is here, or one `man` page away, or one `info` page away. There is no Google search you need to do to start understanding WebSockets. You can sit at your terminal, type, watch, read, and learn forever.

Have fun. The protocol is happy to be poked at. Nothing in this sheet will break anything (assuming you're hitting test endpoints like `echo.websocket.events`, not production systems). Try things. Type commands. Read what comes back. The more you do, the more it all clicks into place.
