# WebSocket

Full-duplex, bidirectional communication protocol over a single TCP connection, initiated via an HTTP/1.1 upgrade handshake, enabling real-time data exchange between clients and servers on ws:// (port 80) or wss:// (port 443).

## WebSocket Upgrade Handshake

```
Client Request:
  GET /chat HTTP/1.1
  Host: server.example.com
  Upgrade: websocket
  Connection: Upgrade
  Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==
  Sec-WebSocket-Version: 13
  Sec-WebSocket-Protocol: chat, superchat
  Sec-WebSocket-Extensions: permessage-deflate

Server Response:
  HTTP/1.1 101 Switching Protocols
  Upgrade: websocket
  Connection: Upgrade
  Sec-WebSocket-Accept: s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
  Sec-WebSocket-Protocol: chat
  Sec-WebSocket-Extensions: permessage-deflate

Accept key derivation:
  SHA1(Key + "258EAFA5-E914-47DA-95CA-5AB5DC11505A") → base64
  SHA1("dGhlIHNhbXBsZSBub25jZQ==" + GUID) → base64 → s3pPLMBiTxaQ9kYGzzhZRbK+xOo=
```

## WebSocket Frame Format

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-------+-+-------------+-------------------------------+
|F|R|R|R| opcode|M| Payload len |    Extended payload length    |
|I|S|S|S|  (4)  |A|     (7)     |             (16/64)           |
|N|V|V|V|       |S|             |   (if payload len==126/127)   |
| |1|2|3|       |K|             |                               |
+-+-+-+-+-------+-+-------------+-------------------------------+
|     Extended payload length continued, if payload len == 127  |
+-------------------------------+-------------------------------+
|                               | Masking-key, if MASK set to 1 |
+-------------------------------+-------------------------------+
| Masking-key (continued)       |          Payload Data         |
+-------------------------------+-------------------------------+
|                     Payload Data continued ...                |
+---------------------------------------------------------------+

FIN=1: final fragment
Opcodes: 0x0=continuation, 0x1=text, 0x2=binary,
         0x8=close, 0x9=ping, 0xA=pong
MASK=1: client-to-server frames MUST be masked
Payload lengths: 0-125 (7-bit), 126 (16-bit extended), 127 (64-bit extended)
```

## Frame Types and Control Messages

```
Text Frame (opcode 0x1):
  - UTF-8 encoded text data
  - FIN=1 for single frame, FIN=0 for fragmented

Binary Frame (opcode 0x2):
  - Raw binary data (images, protobuf, msgpack)

Ping (opcode 0x9):
  - Keepalive/heartbeat, server or client can send
  - Must respond with Pong containing same payload

Pong (opcode 0xA):
  - Response to Ping, copies the payload
  - Unsolicited Pong is allowed (unidirectional heartbeat)

Close (opcode 0x8):
  - Initiates closing handshake
  - Payload: 2-byte status code + optional UTF-8 reason
  - Both sides must send Close frame

Close Status Codes:
  1000 — Normal closure
  1001 — Going away (server shutdown, page navigation)
  1002 — Protocol error
  1003 — Unsupported data type
  1006 — Abnormal closure (no close frame, connection lost)
  1007 — Invalid payload (bad UTF-8)
  1008 — Policy violation
  1009 — Message too big
  1010 — Missing expected extension
  1011 — Unexpected server error
  1015 — TLS handshake failure
```

## Client-Side JavaScript

```javascript
// Basic WebSocket connection
const ws = new WebSocket('wss://server.example.com/chat');

// Connection opened
ws.addEventListener('open', (event) => {
  console.log('Connected');
  ws.send('Hello Server!');
  ws.send(new Blob([binaryData]));       // Binary via Blob
  ws.send(new ArrayBuffer(16));          // Binary via ArrayBuffer
});

// Listen for messages
ws.addEventListener('message', (event) => {
  if (typeof event.data === 'string') {
    console.log('Text:', event.data);
  } else {
    // Binary data (Blob or ArrayBuffer depending on binaryType)
    console.log('Binary:', event.data);
  }
});

// Error handling
ws.addEventListener('error', (event) => {
  console.error('WebSocket error:', event);
});

// Connection closed
ws.addEventListener('close', (event) => {
  console.log(`Closed: code=${event.code} reason=${event.reason}`);
  if (!event.wasClean) {
    // Reconnect logic
  }
});

// Control
ws.binaryType = 'arraybuffer';  // or 'blob' (default)
ws.close(1000, 'Done');         // Clean close
console.log(ws.readyState);     // 0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED
console.log(ws.bufferedAmount); // Bytes queued but not yet sent
```

## Server-Side: Node.js (ws library)

```javascript
const { WebSocketServer } = require('ws');

const wss = new WebSocketServer({ port: 8080 });

wss.on('connection', (ws, req) => {
  const ip = req.socket.remoteAddress;
  console.log(`Client connected: ${ip}`);

  ws.on('message', (data, isBinary) => {
    // Broadcast to all connected clients
    wss.clients.forEach((client) => {
      if (client.readyState === 1) {
        client.send(data, { binary: isBinary });
      }
    });
  });

  ws.on('close', (code, reason) => {
    console.log(`Client disconnected: ${code}`);
  });

  // Heartbeat
  ws.isAlive = true;
  ws.on('pong', () => { ws.isAlive = true; });
});

// Ping all clients every 30 seconds
setInterval(() => {
  wss.clients.forEach((ws) => {
    if (!ws.isAlive) return ws.terminate();
    ws.isAlive = false;
    ws.ping();
  });
}, 30000);
```

## Server-Side: Go (gorilla/websocket)

```go
import (
    "net/http"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        return true // Validate origin in production!
    },
}

func handler(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil { return }
    defer conn.Close()

    for {
        msgType, msg, err := conn.ReadMessage()
        if err != nil { break }
        err = conn.WriteMessage(msgType, msg) // Echo
        if err != nil { break }
    }
}
```

## WebSocket with nginx Reverse Proxy

```nginx
# /etc/nginx/sites-available/websocket
upstream ws_backend {
    server 127.0.0.1:8080;
}

server {
    listen 443 ssl;
    server_name ws.example.com;

    ssl_certificate     /etc/letsencrypt/live/ws.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/ws.example.com/privkey.pem;

    location /ws {
        proxy_pass http://ws_backend;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # Timeouts for WebSocket (longer than HTTP)
        proxy_read_timeout 3600s;
        proxy_send_timeout 3600s;
    }
}
```

## Testing with CLI Tools

```bash
# websocat — versatile WebSocket CLI
websocat ws://localhost:8080/ws
websocat wss://echo.websocket.org

# wscat (Node.js)
npx wscat -c ws://localhost:8080/ws

# curl (experimental WebSocket support, curl >= 7.86)
curl --include \
  --no-buffer \
  --header "Connection: Upgrade" \
  --header "Upgrade: websocket" \
  --header "Sec-WebSocket-Version: 13" \
  --header "Sec-WebSocket-Key: $(openssl rand -base64 16)" \
  https://server.example.com/ws

# Python one-liner
python3 -c "
import asyncio, websockets
async def main():
    async with websockets.connect('ws://localhost:8080') as ws:
        await ws.send('hello')
        print(await ws.recv())
asyncio.run(main())
"
```

## permessage-deflate Compression

```
# Negotiated via extensions header
Sec-WebSocket-Extensions: permessage-deflate;
  client_max_window_bits=15;
  server_max_window_bits=15;
  client_no_context_takeover;
  server_no_context_takeover

# Parameters:
#   server_max_window_bits (8-15): LZ77 window size server uses
#   client_max_window_bits (8-15): LZ77 window size client uses
#   server_no_context_takeover: reset compression context per message
#   client_no_context_takeover: reset compression context per message

# Trade-offs:
#   context_takeover=yes: better compression, more memory per connection
#   context_takeover=no:  worse compression, less memory per connection
#   For 10K+ connections, use no_context_takeover to save memory
```

## Tips

- Always use `wss://` (WebSocket over TLS) in production; many proxies and firewalls block plain `ws://`
- Implement application-level heartbeats (ping/pong) every 30 seconds; do not rely on TCP keepalive alone
- Set `proxy_read_timeout` to a large value in nginx (3600s); the default 60s will kill idle WebSocket connections
- Use exponential backoff with jitter for reconnection; thundering herd after a server restart can DDoS yourself
- Check `bufferedAmount` before sending; if it grows, the client is sending faster than the network can handle
- Always validate the `Origin` header server-side to prevent cross-site WebSocket hijacking (CSWSH)
- Use subprotocols (`Sec-WebSocket-Protocol`) to version your application protocol; reject unknown subprotocols
- Enable `permessage-deflate` for text-heavy payloads but disable it for already-compressed binary data
- Fragment large messages; a single 100MB frame blocks the connection for all control frames including pong
- WebSocket connections count against browser per-domain limits (typically 6); use a single multiplexed connection
- Handle close code 1006 (abnormal closure) as a network error requiring reconnection, not a clean shutdown
- For HTTP/2 environments, consider WebSocket over HTTP/2 (RFC 8441) to share the same TLS connection

## See Also

- http, tls, nginx, haproxy, sse, grpc, socket-io

## References

- [RFC 6455 — The WebSocket Protocol](https://datatracker.ietf.org/doc/html/rfc6455)
- [RFC 7692 — Compression Extensions (permessage-deflate)](https://datatracker.ietf.org/doc/html/rfc7692)
- [RFC 8441 — Bootstrapping WebSockets with HTTP/2](https://datatracker.ietf.org/doc/html/rfc8441)
- [MDN WebSocket API](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)
- [ws — Node.js WebSocket Library](https://github.com/websockets/ws)
- [gorilla/websocket — Go WebSocket Library](https://github.com/gorilla/websocket)
