# SIP Protocol — Deep Dive

Transaction-oriented session signaling theory, state machines, timer math, and threat models.

## Setup

The Session Initiation Protocol (SIP) is a transaction-oriented application-layer signaling protocol. It is not a session protocol — it does not carry media, does not establish bearer channels, and does not maintain ongoing transport state in the way a TCP connection does. SIP exists to negotiate, modify, and tear down sessions whose actual media is conveyed by other protocols, almost always RTP/RTCP for audio and video.

The protocol layering above transport is the central abstraction. SIP messages traverse UDP, TCP, TLS, SCTP, or WebSocket transports, but the SIP layer itself imposes a strict transactional model: every request initiates a transaction, every transaction has a finite-state-machine that governs retransmission, give-up, and termination, and every transaction is independent of every other transaction except where explicitly chained by dialog state. This separation matters because it means SIP can be reasoned about in pieces — one transaction at a time — even when the call as a whole involves dozens of hops, multiple proxies, forking branches, and re-INVITEs.

The naming distinguishes between User Agent Client (UAC) — the entity that originates a request — and User Agent Server (UAS) — the entity that responds. A single physical endpoint flips between UAC and UAS roles routinely; a phone is a UAC when placing a call, a UAS when receiving one. Proxies sit in between but conceptually act as both: they receive a request as a UAS, then forward it as a UAC. This duality is essential for understanding how transactions stack at intermediate hops.

The transactional layer above transport is what gives SIP its reliability story over UDP. Unlike HTTP-over-TCP where the transport guarantees ordered delivery, SIP-over-UDP must implement its own retransmission and acknowledgement logic at the SIP layer. The timers (T1 through K) and the state machines (INVITE/non-INVITE, client/server) are the machinery that does this. Even when SIP runs over TCP — where retransmissions are unnecessary — the state machines still govern give-up timers and final-response handling.

A SIP transaction is bounded: it begins with a request, optionally yields one or more provisional responses (1xx), and concludes with exactly one final response (2xx-6xx). After the final response, the transaction state machine drives toward Terminated through a wait phase whose duration depends on the transport (UDP requires waiting for retransmissions; TCP can terminate immediately).

Sessions, in contrast to transactions, are conceptual. A "call" — what a user perceives as a single phone call — is a SIP dialog whose lifetime spans an INVITE transaction, zero or more re-INVITE transactions for mid-call modifications, possibly a hold/resume, possibly a transfer, and finally a BYE transaction. The dialog persists across multiple transactions; the transactions are short-lived; the media (RTP) flows independently and survives signaling-path changes (e.g., proxy failover). The clean separation between transaction lifetime, dialog lifetime, and media lifetime is one of SIP's most important architectural achievements.

## Why SIP Looks Like HTTP

In 1999, Mark Handley, Henning Schulzrinne, Eve Schooler, and Jonathan Rosenberg authored RFC 2543, the original SIP specification. The decision to model SIP on HTTP was deliberate and somewhat controversial at the time. The competing design space was H.323 — the ITU's binary, ASN.1-encoded, telecom-style signaling stack — which was the dominant VoIP protocol in the late 1990s.

The "HTTP for sessions" pivot rested on three claims. First, that text-based protocols are easier to debug, easier to extend, and easier to inspect than binary ones. Anyone with `tcpdump` and a pair of eyes can read a SIP message; the same is not true for H.225 PDUs. Second, that the request-response paradigm of HTTP — methods, status codes, headers, optional body — was a strong fit for the negotiation pattern of "I want to start a call, here's my media offer, please respond with your answer". Third, that mirroring HTTP's design would let SIP inherit decades of operational tooling: log parsers, load balancers, caches, header-rewriting middleware.

The differences from HTTP, however, are profound and routinely catch newcomers. Caching has no analog in SIP — every request is unique, identified by Call-ID and CSeq, and there is no notion of "fetch the same resource twice and get the same answer". Idempotency is similarly inapplicable: re-sending an INVITE is not idempotent; it creates a new session or, more often, simply triggers a retransmission of the previous response.

Statefulness is inverted. HTTP is stateless at the protocol level; cookies and sessions are application-layer constructions. SIP is stateful at the protocol level: dialogs persist across transactions, transactions persist across messages, and proxies maintain state for routing-loop detection and routing-decision pinning. A "stateless proxy" in SIP terminology is one that does not maintain transaction state, but it still maintains some routing state (Via-header processing, for instance).

Routing is fundamentally different. HTTP traffic is point-to-point, modulo CDN edges; the client knows the server's URL, DNS resolves it, TCP connects, and the request flows. SIP requests are routed through a chain of proxies, each adding a Via header; responses are routed back along the same chain in reverse, by stripping Via headers; and the initial request can be sent to a proxy that does not own the destination, with the proxy resolving and forwarding via its own logic.

Methods diverged. HTTP's GET/PUT/POST/DELETE became SIP's INVITE/REGISTER/BYE/CANCEL/ACK/OPTIONS/INFO/UPDATE/REFER/SUBSCRIBE/NOTIFY/MESSAGE/PUBLISH/PRACK. The semantics overlap superficially — INVITE is "start something", BYE is "end something" — but the parallels break down at the second order. INVITE has a three-way handshake (INVITE/200/ACK); HTTP has no equivalent. CANCEL is meta-protocol — a request to cancel another request — which has no direct HTTP analog (the closest is HTTP/2 RST_STREAM).

The headers superficially mirror HTTP: Via, From, To, Call-ID, CSeq, Contact, Content-Type, Content-Length. But the semantics differ. Via in HTTP is informational and rarely used; Via in SIP is the routing backbone — it is how responses find their way back. From and To in SIP carry the addresses-of-record (AoR), not the actual route; the route is in Via and Record-Route. CSeq is a transaction-numbering field with no HTTP equivalent, though it superficially resembles a request ID.

The deeper takeaway: SIP's HTTP-mimicry is skin-deep. The protocol *looks* familiar, the syntax *parses* like HTTP, but the semantics are designed for telecom-style call control. Engineers who carry HTTP intuitions into SIP debugging routinely misdiagnose problems. The state machines, the timers, the dialog model, the routing — all of these are SIP-native and have no HTTP counterpart.

## Transaction State Machine Theory

A SIP transaction is formally a Mealy machine: a finite-state machine whose outputs depend on both the current state and the current input. The states are discrete, the inputs are messages or timer expirations, and the outputs are the messages emitted, the timers started, and the state transitions performed.

The transaction state machine is the unit of analysis for SIP reliability. Every reliability property — every retransmission, every give-up, every "did this message get through" question — reduces to a question about which state the transaction is in and what timer is running. The state machine is the contract between the SIP layer and the application above it.

Four state machines exist, partitioned by direction (client vs server) and by request type (INVITE vs non-INVITE). The four are:

| Direction | Method                | RFC 3261 Section |
|-----------|-----------------------|------------------|
| Client    | INVITE                | §17.1.1           |
| Client    | non-INVITE            | §17.1.2           |
| Server    | INVITE                | §17.2.1           |
| Server    | non-INVITE            | §17.2.2           |

The split between INVITE and non-INVITE matters because INVITE has a three-way handshake (INVITE → final response → ACK) whereas non-INVITE has a two-way handshake (request → final response). The third leg of the INVITE handshake — the ACK — has its own routing semantics (end-to-end for 2xx, hop-by-hop for non-2xx) and its own state-tracking requirements.

The canonical states across all four machines are some subset of: Calling, Trying, Proceeding, Completed, Confirmed, Terminated. Not every machine has every state — INVITE-server has Confirmed (entered when ACK is received for a non-2xx); non-INVITE machines never have Confirmed.

Transitions are triggered by:
1. Receipt of a message of a specific class (1xx provisional, 2xx success, 3xx-6xx failure).
2. Expiration of a timer (Timer A, B, D, E, F, G, H, I, J, K).
3. Application action (e.g., the application calls "send 200 OK" on the INVITE-server transaction, transitioning it from Proceeding to Completed).
4. Transport error (connection close, ICMP unreachable).

The Mealy formalism makes outputs depend on state-plus-input. Concretely: receiving a 200 OK in state Proceeding emits an ACK and starts Timer D; receiving a 200 OK in state Completed (a retransmitted 200) re-emits the ACK but does not restart Timer D. The state, in other words, modulates the response.

Termination is always reached via a timer or via ACK (for INVITE-server). Timer-driven termination ensures that even if the network drops messages, transactions eventually clean up; the alternative — waiting forever — would leak memory at proxies and endpoints. The timer values are calibrated to give "enough" retransmissions to ride out brief packet loss while not blocking the transaction layer for arbitrarily long.

The full transition table for INVITE Client Transaction is the most studied and the most error-prone to implement; it is presented in the next section.

## INVITE Client Transaction (RFC 3261 §17.1.1)

The INVITE Client Transaction is the state machine that governs the UAC side of a call setup. It begins when the application asks the transaction layer to send an INVITE; it ends when the transaction layer reaches Terminated and discards transaction state.

The states are: Calling, Proceeding, Completed, Accepted (added by RFC 6026), Terminated. The original RFC 3261 had five states; RFC 6026 added Accepted to fix a defect in handling 2xx retransmissions.

State transitions for an unreliable transport (UDP):

| From State | Event                          | Action                              | To State    |
|------------|--------------------------------|-------------------------------------|-------------|
| Init       | App: send INVITE               | Send INVITE; start Timer A, Timer B | Calling     |
| Calling    | Timer A expires                | Retransmit INVITE; Timer A doubles  | Calling     |
| Calling    | Timer B expires                | Inform app: timeout                 | Terminated  |
| Calling    | 1xx received                   | Cancel Timer A, B; pass to app      | Proceeding  |
| Calling    | 2xx received                   | Pass to app; cancel A, B            | Accepted (or Terminated for stateless) |
| Calling    | 3xx-6xx received               | Send ACK; pass to app; start Timer D| Completed   |
| Proceeding | 1xx received                   | Pass to app                         | Proceeding  |
| Proceeding | 2xx received                   | Pass to app                         | Accepted    |
| Proceeding | 3xx-6xx received               | Send ACK; pass to app; start Timer D| Completed   |
| Completed  | 3xx-6xx received (retransmit)  | Re-send ACK                         | Completed   |
| Completed  | Timer D expires                | (cleanup)                           | Terminated  |
| Accepted   | 2xx received (retransmit)      | Pass to app                         | Accepted    |
| Accepted   | Timer M expires                | (cleanup)                           | Terminated  |

Timer values:

- **T1**: round-trip-time estimate, default 500 ms. RFC 3261 mandates T1 = 500 ms unless the implementation has a better RTT estimate.
- **Timer A**: initial value T1, doubles on each retransmission (exponential backoff). After Timer A expires the first time, the next value is 2×T1 = 1000 ms; then 2000, 4000, 8000, 16000 ms.
- **Timer B**: 64×T1 = 32 seconds. This is the give-up timer; if no response arrives in 32 seconds, the transaction is declared timed out.
- **Timer D**: ≥32 seconds for unreliable transport (typically 32s); 0 for reliable transport. This is the "wait for retransmissions of final response" timer; while it runs, retransmitted 3xx-6xx responses are absorbed by re-sending the ACK.
- **Timer M**: 64×T1 = 32 seconds (RFC 6026). The "wait for additional 2xx retransmissions" timer.

The geometric series of Timer A retransmissions is bounded by Timer B. With T1 = 500 ms, retransmissions occur at t = 0.5, 1.0, 2.0, 4.0, 8.0, 16.0 seconds; cumulative retransmissions = 1 + 2 + 4 + 8 + 16 = 31 seconds, so seven INVITE retransmissions before Timer B fires at 32 s.

The Accepted state was added because RFC 3261 had a defect: 2xx responses to INVITE can be retransmitted indefinitely by the UAS (ACK is end-to-end, not hop-by-hop, so the proxy in between never knows the ACK arrived). Without Accepted, the UAC's transaction would terminate on first 2xx, and subsequent 2xx retransmissions would be processed by the dialog layer (or worse, by a fresh transaction matcher with no entry, leading to the "stray response" problem). Accepted holds the transaction open long enough to absorb retransmitted 2xx without spawning new transaction state.

For reliable transports (TCP/TLS/SCTP), Timer A is disabled (no retransmission needed) and Timer D is set to 0 (the transaction can terminate immediately on entering Completed because the transport guarantees no duplicates). The state machine structure is the same; the timer values differ.

## Non-INVITE Client Transaction (RFC 3261 §17.1.2)

The non-INVITE Client Transaction handles every method that is not INVITE: REGISTER, OPTIONS, BYE, CANCEL, INFO, MESSAGE, NOTIFY, REFER, UPDATE, PRACK, PUBLISH, SUBSCRIBE.

The states are: Trying, Proceeding, Completed, Terminated. There is no Calling (that name is reserved for INVITE) and no Accepted (no 2xx retransmission problem because non-INVITE has a two-way handshake; the response itself is the acknowledgement).

State transitions for unreliable transport:

| From State | Event                          | Action                              | To State    |
|------------|--------------------------------|-------------------------------------|-------------|
| Init       | App: send request              | Send request; start Timer E, F      | Trying      |
| Trying     | Timer E expires                | Retransmit; Timer E doubles up to T2| Trying      |
| Trying     | Timer F expires                | Inform app: timeout                 | Terminated  |
| Trying     | 1xx received                   | Pass to app                         | Proceeding  |
| Trying     | 2xx-6xx received               | Pass to app; start Timer K          | Completed   |
| Proceeding | 1xx received                   | Pass to app                         | Proceeding  |
| Proceeding | Timer E expires                | Retransmit; Timer E continues       | Proceeding  |
| Proceeding | Timer F expires                | Inform app: timeout                 | Terminated  |
| Proceeding | 2xx-6xx received               | Pass to app; start Timer K          | Completed   |
| Completed  | response retransmitted         | (absorb)                            | Completed   |
| Completed  | Timer K expires                | (cleanup)                           | Terminated  |

Timer values:

- **T2**: maximum retransmission interval, default 4 seconds. Once Timer E reaches T2, it stops doubling and stays at T2.
- **Timer E**: starts at T1 = 500 ms, doubles each retry until it reaches T2 = 4 s, then stays at T2.
- **Timer F**: 64×T1 = 32 s. Give-up timer.
- **Timer K**: T4 for unreliable, 0 for reliable. T4 = 5 s (the maximum duration a message can remain in the network).

The capping at T2 is the key difference from INVITE. INVITE retransmissions back off exponentially without a cap (until Timer B at 32 s); non-INVITE retransmissions back off until T2 = 4 s and then retry every 4 s until Timer F at 32 s. So non-INVITE produces more retransmissions over the same 32-second window, on the assumption that non-INVITE requests are generally smaller and more frequent.

For non-INVITE, after entering Proceeding, retransmissions of the request continue (Timer E keeps running) — this is unlike INVITE, where Timer A is cancelled on entering Proceeding. The rationale: a 1xx for non-INVITE is rarely sent (only OPTIONS and SUBSCRIBE typically use it); the transaction layer cannot rely on 1xx to indicate "the server got it", so it keeps retransmitting until the final response arrives.

The Trying state is so named because non-INVITE often produces a "100 Trying" provisional response when the server begins processing. INVITE's analogue is the "180 Ringing" or other 18x.

## INVITE Server Transaction (RFC 3261 §17.2.1)

The INVITE Server Transaction governs the UAS side of call setup. It is unique among the four machines in that it terminates not on a timer (after Completed) but on receipt of ACK (transitioning to Confirmed before Terminated). The ACK-tracking complexity is the source of much implementation pain.

The states are: Proceeding, Completed, Confirmed, Accepted (RFC 6026), Terminated. There is no Trying — that's an internal application detail; the transaction enters Proceeding immediately upon receiving the INVITE (which causes a "100 Trying" auto-response in most implementations).

State transitions for unreliable transport:

| From State | Event                          | Action                                           | To State    |
|------------|--------------------------------|--------------------------------------------------|-------------|
| Init       | INVITE received                | Auto-send 100 Trying after 200ms                 | Proceeding  |
| Proceeding | INVITE received (retransmit)   | Re-send most recent provisional response          | Proceeding  |
| Proceeding | App: send 1xx                  | Send response                                     | Proceeding  |
| Proceeding | App: send 2xx                  | Send response; start Timer L                      | Accepted    |
| Proceeding | App: send 3xx-6xx              | Send response; start Timer G, Timer H             | Completed   |
| Completed  | INVITE received (retransmit)   | Re-send 3xx-6xx response                          | Completed   |
| Completed  | Timer G expires                | Re-send 3xx-6xx; Timer G doubles up to T2         | Completed   |
| Completed  | ACK received                   | Stop Timer G, H; start Timer I                    | Confirmed   |
| Completed  | Timer H expires                | Inform app: ACK timeout                           | Terminated  |
| Confirmed  | ACK received (retransmit)      | (absorb)                                          | Confirmed   |
| Confirmed  | Timer I expires                | (cleanup)                                         | Terminated  |
| Accepted   | INVITE received (retransmit)   | Re-send 2xx                                       | Accepted    |
| Accepted   | App: send 2xx (additional)     | Send response                                     | Accepted    |
| Accepted   | Timer L expires                | (cleanup)                                         | Terminated  |

Timer values:

- **Timer G**: starts at T1 = 500 ms, doubles each retransmission up to T2 = 4 s. Retransmits the final 3xx-6xx response.
- **Timer H**: 64×T1 = 32 s. Give-up waiting for ACK.
- **Timer I**: T4 = 5 s for unreliable, 0 for reliable. Wait after ACK to absorb duplicate ACKs.
- **Timer L**: 64×T1 = 32 s (RFC 6026). Wait in Accepted state.

The asymmetry between Completed (for non-2xx) and Accepted (for 2xx) reflects the asymmetric ACK semantics. For non-2xx, ACK is hop-by-hop, matched by branch parameter, generated automatically by the transaction layer on the UAC side, and consumed by the transaction layer on the UAS side. For 2xx, ACK is end-to-end, generated by the UAC application layer (the dialog layer, technically), travels along the dialog's route set, and is delivered to the UAS application layer (not the transaction layer — by the time ACK arrives, the transaction has terminated, and the ACK is matched by Call-ID/From-tag/To-tag/CSeq triple by the dialog matcher).

This is why Confirmed exists for non-2xx: the transaction layer holds state to absorb retransmitted ACKs. For 2xx, there's no Confirmed state; instead, Accepted holds state to absorb retransmitted INVITEs (which trigger 2xx retransmissions).

The auto-generated "100 Trying" is sent only after a 200 ms delay. This is to suppress the 100 Trying when the application is fast — if the app produces a 180 Ringing within 200 ms, no 100 Trying is sent. The 200 ms timer is RFC 3261's compromise between "always send 100 Trying" (wasteful) and "never send 100 Trying" (causes UAC to retransmit unnecessarily).

## Non-INVITE Server Transaction (RFC 3261 §17.2.2)

The simplest of the four. States: Trying, Proceeding, Completed, Terminated.

State transitions for unreliable transport:

| From State | Event                          | Action                              | To State    |
|------------|--------------------------------|-------------------------------------|-------------|
| Init       | non-INVITE received            | (no auto-response)                  | Trying      |
| Trying     | request received (retransmit)  | Drop                                | Trying      |
| Trying     | App: send 1xx                  | Send response                       | Proceeding  |
| Trying     | App: send 2xx-6xx              | Send response; start Timer J        | Completed   |
| Proceeding | request received (retransmit)  | Re-send most recent provisional     | Proceeding  |
| Proceeding | App: send 1xx                  | Send response                       | Proceeding  |
| Proceeding | App: send 2xx-6xx              | Send response; start Timer J        | Completed   |
| Completed  | request received (retransmit)  | Re-send final response              | Completed   |
| Completed  | Timer J expires                | (cleanup)                           | Terminated  |

Timer J: 64×T1 = 32 s for unreliable, 0 for reliable. Wait for retransmissions of the request (which the server must absorb by re-sending the final response).

There is no analogue of Timer G (final-response retransmission) because the non-INVITE final response is acknowledged simply by the request not being retransmitted — i.e., the UAC's Timer K elapsing without a retransmission means the response got through. The UAS does not retransmit final responses on its own initiative; it only re-emits them in response to a re-received request.

The non-INVITE-server has the cleanest state machine because there is no ACK leg, no race condition, no end-to-end vs hop-by-hop distinction. Most non-INVITE traffic in practice is SUBSCRIBE/NOTIFY for presence and BYE for call teardown; both are simple request/response.

## Dialog State

A SIP dialog is established by a 2xx response to INVITE. Dialogs are identified by the triple (Call-ID, local-tag, remote-tag), where the tags are the From and To tag parameters depending on direction (UAC vs UAS).

The early dialog is established when a 1xx response with a To-tag is received. The confirmed dialog is established when a 2xx response is received; this either upgrades the early dialog to confirmed or creates a new confirmed dialog if no 1xx with To-tag had been received.

The reason for early dialogs is that forking proxies can create multiple early dialogs for a single INVITE (each fork branch produces 1xx with its own To-tag). When 2xx finally arrives from one branch, that branch's early dialog becomes the confirmed dialog and other early dialogs are abandoned. The fork-handling logic at the UAC must track multiple early dialogs simultaneously.

Dialog state includes:
- Local URI (From or To, depending on direction)
- Remote URI
- Local sequence number (CSeq for outgoing requests within the dialog)
- Remote sequence number
- Call-ID
- Local tag, remote tag
- Route set (from Record-Route headers)
- Remote target (from Contact header in initial response)
- Secure flag (whether SIPS-URI was used)

The route set is the ordered list of proxies that the dialog must traverse. It is derived from Record-Route headers in the response. Subsequent in-dialog requests (BYE, re-INVITE, UPDATE) MUST follow this route. The remote target is the URI to which the request is addressed; it is the Contact header from the dialog-creating response.

Dialog termination: a confirmed dialog is terminated by a BYE request, by a 481 Call/Transaction Does Not Exist response, by a network error, or by application timeout. An early dialog is terminated by a non-2xx final response, by 408/481/487, or by superseding 2xx from another fork.

The dialog identifier triple is asymmetric in a counter-intuitive way: from the UAC's perspective, local-tag = From-tag and remote-tag = To-tag; from the UAS's perspective, local-tag = To-tag and remote-tag = From-tag. So the same dialog has two different "local" identifiers depending on which endpoint is doing the bookkeeping.

## Branch Identifier

Every SIP request must include a Via header with a `branch` parameter. Per RFC 3261, the branch parameter MUST start with the magic cookie `z9hG4bK`, which serves two purposes: it identifies the request as conformant to RFC 3261 (older RFC 2543 implementations did not use this prefix), and it makes the branch parameter globally unique with high probability.

The branch parameter, after the magic cookie, is a unique string. It identifies the transaction. Specifically, the transaction is matched by the triple (branch, sent-by, method) where sent-by is the value of the Via's sent-by parameter (host:port of the originator at this hop) and method is the request method (with ACK matching INVITE for non-2xx, ACK being its own transaction for 2xx).

Per-hop uniqueness is the key property. Each proxy that forwards a request adds its own Via header with its own unique branch. The branch is unique per transaction, per hop. This means a single end-to-end request has multiple branch parameters in its Via stack — one per proxy hop — and each transaction at each proxy is tracked independently.

Branch generation algorithms typically combine: a random 64-bit value, a hash of the Call-ID/From-tag/To-tag/CSeq, and a sequence number. The objective is to make replay attacks computationally hard while keeping the branch short enough to fit in a Via header.

Loop detection uses branches: a proxy receiving a request inspects the Via headers; if any Via has the proxy's own sent-by address with a branch the proxy generated, the request has looped. This is the proxy-loop-detection RFC 3261 §16.3 algorithm. Note that loop detection works on branch parameters, not on Max-Forwards (Max-Forwards is a backstop).

## SIP Routing Algorithm (RFC 3263)

When a UAC has a SIP URI and needs to send a request, RFC 3263 specifies how to resolve the URI to (transport, host, port). The algorithm is:

1. **Extract URI components**: parse the SIP URI to get the host part (FQDN or IP), port (if present), transport parameter (if present), and scheme (sip vs sips).

2. **If the host is an IP literal**: use it directly. Default transport is UDP (or TLS for sips). Default port is 5060 (or 5061 for sips). Skip to step 5.

3. **If port is explicit**: skip NAPTR and SRV; do an A/AAAA lookup on the host; use the configured port and transport.

4. **NAPTR lookup**: query the DNS for a NAPTR record on the host. NAPTR records map a service to an SRV record name. Service field will be `SIP+D2U` (UDP), `SIP+D2T` (TCP), `SIPS+D2T` (TLS), or `SIP+D2W`/`SIPS+D2W` (WebSocket). Sort by NAPTR order and preference; iterate.

5. **SRV lookup**: for each NAPTR's Replacement field (or, if no NAPTR found, for `_sip._udp.host`, `_sip._tcp.host`, `_sips._tcp.host`), query the SRV record. SRV gives priority, weight, port, and target (which is the hostname to A/AAAA-resolve).

6. **A/AAAA lookup**: resolve the SRV target to IPv4 (A) and IPv6 (AAAA) addresses.

7. **Fallback**: if no NAPTR and no SRV, A/AAAA-resolve the original host directly; use default port (5060/5061) and default transport (UDP for sip, TLS for sips).

The ladder is NAPTR → SRV → A/AAAA. Each step refines: NAPTR picks the transport, SRV picks the host:port, A/AAAA picks the IP.

NAPTR's purpose is to express provider preferences. A SIP provider with both UDP and TCP servers can use NAPTR to express "prefer TCP, but UDP is available" by setting the preference fields appropriately. SRV's purpose is to express load-balancing and failover among multiple servers for the same transport.

In practice, many small deployments skip NAPTR and rely solely on SRV; some skip SRV and rely on A/AAAA. The algorithm tolerates this — each step has a fallback to the next.

The "default port" rule for sips is 5061; for sip it's 5060. These are the IANA-assigned ports. If a sips: URI has an explicit non-5061 port, it still uses TLS but on that port.

## The Trapezoid

The classic SIP architecture is the trapezoid: caller → caller's outbound proxy → callee's inbound proxy → callee. The four corners are the two endpoints and the two proxies. The path from caller to callee traverses the proxies; the path from callee back uses the same proxy chain in reverse (Via-header reversal).

The trapezoid model presumes mutual trust between the two providers' proxies. In practice, this trust is established via:
- TLS with mutual cert auth between proxies (rare in PSTN-grade SIP).
- Static-route configuration: provider A's outbound proxy is configured to send all calls to provider B's inbound proxy at a specific IP:port.
- IP-allowlist authentication: provider B's inbound proxy accepts SIP from provider A's IP only.
- TGREP (Telephony Gateway REGistration Protocol, RFC 3219) for inter-domain peering, used in larger inter-provider VoIP networks.

The endpoints typically use REGISTER to bind their AoR (Address-of-Record, e.g., `sip:alice@example.com`) to a Contact URI (e.g., `sip:alice@192.0.2.5:5060`). The proxy at example.com maintains the binding and forwards calls to the Contact when example.com's proxy receives a request for `sip:alice@example.com`.

The trapezoid is named for its shape: caller and callee at the bottom corners, the two proxies at the top corners, with the horizontal bar across the top representing the inter-provider link.

In modern deployments, the trapezoid model is often degenerate. With cloud-based PBXes (e.g., 3CX, FreePBX, Asterisk), the "outbound proxy" and "inbound proxy" are often the same server, and the trapezoid collapses to a triangle (caller → PBX → callee). For inter-domain SIP federation, true trapezoid is rare; most calls go through PSTN gateways, which break the SIP-end-to-end model entirely.

## Forking

When a proxy receives a request and has multiple Contact bindings for the AoR, it can fork: send the request to multiple destinations in parallel. RFC 3261 §16.7 specifies the fork-and-collect logic.

Parallel forking sends INVITEs to all branches simultaneously. Sequential forking tries one, waits for failure, then tries the next. Most proxies do parallel forking by default.

The "200 OK race" is the central problem. If a proxy forks an INVITE to three destinations and all three answer with 200 OK, the proxy must:
1. Select one as the winner (typically the first 2xx received).
2. Cancel the losing branches (send CANCEL to them; receive 487 Request Terminated).
3. Forward the winning 2xx to the UAC.
4. Suppress (or reject) the losing 2xx.

The UAS that produced a losing 2xx typically expects an ACK and will retransmit the 2xx until ACK arrives or 64×T1 elapses. The proxy must absorb these retransmissions or send a BYE on the dialog to tear it down. The "stray 2xx" handling at the proxy is a common source of bugs.

CANCEL of losing branches must occur in the early-dialog window (before 2xx), not after. Once the 2xx has been processed by the proxy, it's too late to CANCEL — the only recourse is BYE.

Forking interacts with PRACK (reliable provisional responses) in subtle ways: if multiple branches send 18x, the UAC must PRACK each independently, and each branch's transaction state machine tracks PRACKs separately.

Forking is the reason early-dialog state matters. The UAC may have multiple early dialogs concurrently (one per fork branch, each with its own To-tag) and must track them all until 2xx arrives from one (which becomes the confirmed dialog) or all branches fail.

## SIP Outbound (RFC 5626)

Traditional SIP REGISTER assumes one Contact URI per AoR per device. SIP Outbound generalizes this to multiple "flows" — persistent connections — between the device and the registrar. Each flow is identified by a flow-token, which the registrar issues and the device echoes back on subsequent registrations.

The motivation: NAT-bound devices cannot accept inbound SIP because their NAT mapping is ephemeral. SIP Outbound says: keep the connection open; route inbound calls back along the open connection.

Concretely, the device opens a persistent TCP/TLS/WebSocket connection to its proxy and registers via that connection. The registrar binds the AoR to (Contact, flow-token, instance-id). When a call arrives for the AoR, the proxy looks up the flow-token, determines the open connection, and forwards the INVITE down that connection. The proxy must not try to open a new connection to the device — that would fail behind NAT.

The Outbound dual-flow model provides redundancy: the device opens two flows to two distinct edge proxies. Both register the same AoR. The originating proxy can forward the call to either flow; if one is down (e.g., one edge proxy crashed), the other is used. The device knows which flows are alive via SIP keepalive (CRLF for TCP, ping for WebSocket).

Flow-token format: opaque string, but typically encodes (registrar-id, connection-id, expiry). The device echoes it in the Path header on re-registration. The registrar validates the flow-token before refreshing the binding.

Outbound is now the default for any modern SIP-over-TCP/TLS deployment with NAT-bound endpoints. Pure UDP deployments often skip Outbound and rely on Symmetric NAT keepalives via short-Expires REGISTER refreshes.

## ACK Transport Quirk

ACK has different routing semantics depending on whether it acknowledges a 2xx or a 3xx-6xx response.

For non-2xx (3xx-6xx) responses, ACK is a hop-by-hop transaction: it is generated by the UAC's INVITE Client Transaction state machine, sent to the next hop (the proxy in the Via stack), and consumed by the UAS's INVITE Server Transaction state machine. The branch parameter on the ACK matches the original INVITE's branch (because RFC 3261 §17.1.1.3 says ACK for non-2xx uses the same branch as the INVITE for transaction matching purposes). Each proxy in the chain sees ACK and matches it to its INVITE Server Transaction; no application-layer processing.

For 2xx responses, ACK is end-to-end: it is generated by the UAC's dialog layer (NOT the transaction layer), sent along the dialog's route set, and consumed by the UAS's dialog layer. Each proxy in between sees ACK as an in-dialog request and forwards it according to the route set. The branch parameter on the ACK is a NEW unique branch (because ACK for 2xx is its own transaction, not a leg of the INVITE transaction). The CSeq number is the same as the INVITE's CSeq, but with the method ACK rather than INVITE.

This asymmetry is a notorious source of bugs:

- Implementations that assume ACK always uses the same branch as the INVITE break for 2xx ACK.
- Implementations that assume ACK is always processed by the transaction layer fail to forward 2xx ACK at proxies.
- Implementations that don't track route sets correctly can route the 2xx ACK to the wrong destination.
- Forking proxies: if 2xx arrives and the proxy hasn't terminated its INVITE Server Transaction (because Timer L is running), the ACK arrives as an in-dialog request and must be forwarded to the winning branch's UAS, not absorbed by the proxy.

The rationale for the asymmetry: 2xx must be retransmitted by the UAS (because the UAC's ACK-generation timing is application-controlled, not transaction-controlled), and the only way to stop the 2xx retransmission is for the ACK to reach the UAS. Therefore ACK must be end-to-end. For non-2xx, the UAS does not retransmit (the proxy can absorb retransmissions via the Server Transaction's Completed state), so ACK can safely be hop-by-hop.

## Re-INVITE

A re-INVITE is an INVITE sent within an established dialog (i.e., with the same Call-ID, From-tag, To-tag as a confirmed dialog). It is used to re-negotiate the SDP — e.g., to add video, change codec, change media address (after IP change), or place a call on hold.

Re-INVITE follows the same INVITE Client/Server Transaction state machines as the initial INVITE, but with these differences:
- The CSeq number is the next number in the dialog's local CSeq sequence (incremented).
- The To-tag and From-tag match the dialog.
- The Contact header may change (if the endpoint moved).
- The Route header carries the dialog's route set.
- A failure (3xx-6xx) does NOT terminate the dialog — the dialog stays in its previous SDP state.

The "glare" scenario occurs when both endpoints send re-INVITE simultaneously. Both INVITE Client Transactions are running, both UASs receive the re-INVITE, and both must respond. The protocol resolves glare via 491 Request Pending: the UAS that received the re-INVITE while having an outstanding re-INVITE of its own responds 491. The UAC then waits a randomized interval and retries.

The randomized retry interval is specified in RFC 3261 §14.1: "The UAS Core MUST generate a 491 (Request Pending) response to receipt of a re-INVITE if it has a pending re-INVITE in the dialog." And: "the UAC SHOULD wait 2.1 seconds + uniform random delay if it is the owner of the Call-ID with the lower lexical value, or 0 to 2 seconds + uniform random delay if it is the owner of the higher Call-ID value." The asymmetric wait times prevent both sides from retrying simultaneously and re-creating the glare.

In practice, glare is rare because most re-INVITEs are user-driven (hold/resume) and human reaction times are slow enough that glare rarely occurs. But automated mid-call modifications (e.g., a media-server-driven codec switch) can trigger glare reproducibly.

## UPDATE (RFC 3311)

The UPDATE method allows mid-dialog parameter renegotiation (typically SDP) without the overhead of re-INVITE. UPDATE is a non-INVITE method, so it uses the simpler non-INVITE state machine and does not have an ACK leg.

UPDATE is most useful for:
- Codec switches mid-call (e.g., from G.722 to G.711 due to bandwidth).
- Hold/resume signaled via SDP changes without restarting media negotiation.
- Early-media SDP updates before 200 OK is received.

The advantage over re-INVITE: UPDATE completes in one round-trip (request + response); re-INVITE requires three legs (INVITE + response + ACK). Faster, simpler, but less flexible.

The disadvantage: UPDATE cannot be used to renegotiate the dialog itself (it's not an INVITE), and many implementations don't support UPDATE for mid-call modifications. Re-INVITE is more universally supported.

UPDATE is required for some scenarios — early-media SDP updates (before 200 OK) cannot be done with re-INVITE because the dialog isn't yet confirmed. UPDATE is the only option.

## PRACK (RFC 3262)

Provisional responses (1xx) in vanilla SIP are not reliable — they are sent once, and if they get lost, the UAS does not retransmit. For most 1xx (e.g., 100 Trying) this is fine; for some (e.g., 183 Session Progress with early-media SDP), losing the response is a problem.

PRACK (Provisional Response ACKnowledgement) provides reliability for 1xx. The protocol:

1. UAS sends 1xx with `Require: 100rel` and an `RSeq` header (a per-response sequence number within the transaction).
2. UAC, upon receiving the 1xx, sends a PRACK request with `RAck: <RSeq> <CSeq>` echoing the 1xx's RSeq and CSeq.
3. UAS, upon receiving PRACK, sends 200 OK to the PRACK.
4. UAS retransmits the 1xx (with the same RSeq) until it receives the PRACK or 64×T1 elapses.

The handshake adds a round-trip but ensures the 1xx was received. PRACK is required (capability negotiated via Supported: 100rel and Require: 100rel headers) when the UAS wants reliable 1xx.

PRACK transactions are non-INVITE transactions (separate state machine from the INVITE that triggered the 1xx). They are within the early dialog if the 1xx had a To-tag.

The most common use case for PRACK: early-media SDP. The 183 Session Progress carries SDP with the early-media media stream description. If the 183 is lost, the UAC doesn't know about the early-media stream and cannot play it. PRACK ensures the 183 (and its SDP) is reliably delivered.

PRACK is mandatory for 3GPP IMS networks; it's optional for general SIP. If an endpoint sends Require: 100rel to a peer that doesn't support it, the call fails with 420 Bad Extension.

## Digest Authentication (RFC 7616)

SIP authentication uses HTTP Digest, modernized by RFC 7616 (which obsoletes RFC 2617).

The flow:
1. UAC sends request without credentials.
2. UAS responds 401 Unauthorized (or 407 Proxy Authentication Required for proxy auth) with a `WWW-Authenticate` (or `Proxy-Authenticate`) header containing realm, nonce, qop, algorithm.
3. UAC computes Digest-Response and re-sends the request with `Authorization` header.

The computation:
- HA1 = H(username:realm:password) where H is the algorithm hash (MD5, SHA-256, SHA-512-256).
- HA2 = H(method:uri) for `qop=auth`, or H(method:uri:H(body)) for `qop=auth-int`.
- response = H(HA1:nonce:nc:cnonce:qop:HA2)

The qop (quality of protection) parameter selects the integrity model:
- `auth`: response includes nonce, nc (nonce count), cnonce (client nonce), but no body integrity.
- `auth-int`: response includes a hash of the message body, so tampering with the body invalidates the response.

`qop=auth-int` is rarely used in practice because:
1. It requires the body to be hashed before the request is sent, which complicates streaming.
2. It interacts poorly with proxies that modify the body (e.g., for anonymization).
3. Most SIP messages have small bodies (SDP) and the integrity benefit is marginal.

Algorithm-confusion attacks: if the server advertises multiple algorithms (e.g., MD5 and SHA-256), an active attacker can downgrade to the weakest. RFC 7616 recommends server pick the strongest the client supports and not advertise weak algorithms. In practice, many SIP deployments still use MD5.

Nonce expiry: the server should expire nonces after a short interval (typically 5 minutes) to prevent replay. The `nc` (nonce count) parameter prevents replay within the nonce's validity window.

Digest authentication does not encrypt the request — it only authenticates. For confidentiality, TLS (sips:) is required.

## SIP Identity (RFC 8224)

SIP Identity provides cryptographic assertion of the calling party. The originating provider signs the request and inserts an `Identity` header containing the signature; downstream providers verify the signature using a public key fetched via the `Identity-Info` header.

The signed canonical form covers: From, To, Call-ID, CSeq, Date, the SDP body. A valid Identity signature attests that the signer (the originating provider, identified by the cert) authorized the use of the From URI.

Identity is the foundation of STIR/SHAKEN. Without Identity, caller-ID spoofing is trivial (just put any value in From and the call goes through). With Identity, downstream providers can verify the From assertion.

The cert chain: Identity-Info points to a URL where the cert is fetched. The cert is signed by a CA in the cert chain. For STIR/SHAKEN, the CA chain terminates at the ATIS-managed Policy Administrator (PA), which authorizes ServiceProviderCode (SPC) tokens.

## STIR/SHAKEN

STIR/SHAKEN is the North American (and increasingly global) framework for caller-ID authentication. STIR (Secure Telephone Identity Revisited) is the IETF protocol family (RFCs 8224-8226); SHAKEN (Signature-based Handling of Asserted Information using toKENs) is the ATIS profile.

The Identity header conveys a JSON Web Token (JWT) called a PASSporT (RFC 8225). The PASSporT contains the calling number, called number, timestamp, and an attestation level:

- **A (Full Attestation)**: the originating provider knows the customer and verified the customer is authorized to use the calling number. Highest confidence.
- **B (Partial Attestation)**: the originating provider knows the customer but cannot verify the customer's authority over the calling number. Medium confidence.
- **C (Gateway Attestation)**: the originating provider received the call from another provider (e.g., a PSTN gateway) and cannot verify the caller. Lowest confidence.

Terminating providers display the attestation to the called party (e.g., "Verified Caller" badge for level A) or use it as input to a call-blocking algorithm.

The cert chain: PASSporT is signed with a private key; the cert is issued by a STIR Certification Authority (STI-CA), which is authorized by the STIR Policy Administrator (STI-PA), which is the ATIS Telephone Numbering Council in the US.

STIR/SHAKEN does not prevent spoofing — it just attests to who's claiming what. A provider that wants to spoof caller-IDs can sign with attestation C, which is technically valid but provides no trust. The market response: terminating providers downweight or block C-attested calls.

## SIP-over-WebSocket (RFC 7118)

WebSocket (RFC 6455) provides a bidirectional, framed, upgraded-HTTP transport that runs in browsers. RFC 7118 defines a SIP-over-WebSocket binding.

The WebSocket connection is initiated by an HTTP Upgrade request:
```
GET /sip HTTP/1.1
Host: sip.example.com
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: ...
Sec-WebSocket-Protocol: sip
```

The `Sec-WebSocket-Protocol: sip` subprotocol advertises that the connection will carry SIP messages. The server responds 101 Switching Protocols, and from then on the connection carries WebSocket frames.

Each WebSocket frame contains exactly one SIP message. The Content-Length is implied by the frame length. There is no message-boundary parsing within a frame.

The Via header for SIP-over-WebSocket uses the `transport=ws` (or `transport=wss` for TLS-secured WebSocket) parameter:
```
Via: SIP/2.0/WS df7jal23ls0d.invalid;branch=z9hG4bK...
```

The `df7jal23ls0d.invalid` is a placeholder hostname — the WebSocket client doesn't have a routable hostname (it's behind NAT in a browser), so RFC 7118 specifies a literal `.invalid` host.

WebSocket SIP is the foundation for browser-based SIP clients. JsSIP, sip.js, and similar libraries implement SIP UAs in JavaScript and connect to the SIP server via WebSocket.

## WebRTC SIP

WebRTC (Web Real-Time Communications) is the browser-native real-time-media stack: RTP/SRTP, ICE, DTLS, codec negotiation. WebRTC does not itself include signaling; it requires an external signaling protocol, of which SIP (over WebSocket) is the most common.

The integration:
1. Browser opens WebSocket to SIP server.
2. Browser (UA) sends INVITE with SDP describing the WebRTC media. The SDP includes:
   - `a=fingerprint`: the SHA-256 of the local DTLS cert.
   - `a=setup`: actpass, active, or passive (DTLS role).
   - `a=ice-ufrag`, `a=ice-pwd`: ICE credentials.
   - `a=candidate`: ICE candidates (host, srflx, relay).
3. SIP server forwards INVITE to remote UA. Response carries remote SDP.
4. Browsers extract DTLS fingerprint, ICE credentials, and candidates from SDP and pass to WebRTC stack.
5. WebRTC stack performs ICE connectivity checks, negotiates DTLS, and exchanges SRTP.

The security properties: DTLS-SRTP is mandatory for WebRTC. SDP's `a=fingerprint` binds the DTLS cert to the SDP (which is signed by the SIP Identity if STIR/SHAKEN is used), preventing MITM on the media path.

The "SDP munging" problem: SIP servers that pass SDP unchanged work; SIP servers that try to rewrite SDP (e.g., to insert their own media-relay address) break WebRTC because the fingerprint binding is invalidated. WebRTC SDP must flow end-to-end intact.

## Reliability of Provisional Responses

Provisional responses (1xx) in vanilla SIP are unreliable. The UAS sends each 1xx once; if it's lost, the UAC will never know.

The most common 1xx — 100 Trying — is fine to lose. The UAC will retransmit the INVITE (Timer A), the UAS will re-send 100 Trying. The transaction recovers naturally.

The problematic 1xx is 180 Ringing, especially in fork scenarios. If a fork has multiple branches and only one rings, losing the 180 from that branch means the UAC never knows the branch is alive. Worse, 183 Session Progress with early-media SDP, if lost, leaves the UAC without the early-media descriptor and the early-media stream cannot be played.

PRACK (RFC 3262) is the cure. PRACK reliably delivers 1xx via a request/response handshake. The UAS retransmits the 1xx until PRACK arrives.

The "180 Ringing retransmission gotcha": some UAS implementations retransmit 180 Ringing periodically as a keepalive while the phone rings, even without PRACK. This is non-standard but common. UAC implementations must tolerate duplicate 180s.

## NAT Traversal Theory

SIP signaling is mostly above-NAT-friendly: the UAC initiates the connection, the proxy responds along the same connection. Inbound SIP (where the proxy initiates) is harder, addressed by SIP Outbound and persistent connections.

Media (RTP) is harder. RTP is UDP, peer-to-peer, and the destination IP:port is signaled in SDP. If the SDP carries the UAC's private IP, the remote UAS cannot reach it through NAT.

STUN (Session Traversal Utilities for NAT, RFC 5389) is the foundational tool. The UAC sends a STUN Binding Request to a public STUN server; the STUN server reflects the UAC's public IP:port in the response. The UAC inserts the public IP:port into the SDP.

NAT type matters:
- **Endpoint-Independent Mapping (EIM, formerly "full cone")**: the NAT maps the internal IP:port to the same external IP:port for all destinations. STUN reflexive address works.
- **Address-Dependent Mapping (formerly "restricted cone")**: the NAT maps internal:port to external:port, but the mapping is destination-IP-dependent. STUN works for the STUN server's address, but not for arbitrary peers.
- **Address-and-Port-Dependent Mapping (formerly "port-restricted cone")**: mapping is destination IP:port dependent. STUN works only for the STUN server's IP:port.
- **Endpoint-Dependent Mapping (formerly "symmetric")**: every (internal, destination) pair gets a fresh external port. STUN fails for any peer other than the STUN server.

For symmetric NATs, TURN (Traversal Using Relays around NAT, RFC 5766) is required. The UAC sends media to a TURN relay; the relay forwards to the peer; the peer sends to the relay; the relay forwards back. This is a fallback because it consumes server bandwidth.

ICE (Interactive Connectivity Establishment, RFC 5245) is the algorithm that combines STUN and TURN. ICE candidate gathering produces:
- **Host candidates**: local IPs (often private).
- **Server-reflexive (srflx) candidates**: the STUN-reflected public IP:port.
- **Relay candidates**: the TURN-allocated relay IP:port.

ICE connectivity checks: each side tries each pair of candidates (local × remote), sending STUN binding requests over the data channel. Successful checks are paired up; the highest-priority successful pair is selected.

Trickle ICE (RFC 8838) allows candidates to be sent incrementally as they're gathered, rather than all at once. This reduces call-setup latency because the call can begin on the first successful pair while more pairs are still being checked.

## SDP Offer/Answer Model (RFC 3264)

SDP (Session Description Protocol, RFC 8866) is a text format for describing media sessions: codecs, IPs, ports, attributes. SDP is carried in SIP message bodies.

The offer/answer model (RFC 3264) governs how SDP is negotiated:
1. **Offerer** (typically the UAC) sends SDP describing what it's willing to send and receive.
2. **Answerer** (typically the UAS) responds with SDP that constrains the offer to what it agrees to.

The answerer is constrained by the offerer's capabilities — it cannot add media that the offerer didn't offer. It can disable media (port 0) but cannot enable media that wasn't in the offer.

Each m-line (media) has a direction:
- `a=sendrecv` (default if absent): both directions.
- `a=sendonly`: send media but don't expect to receive.
- `a=recvonly`: receive media but don't send.
- `a=inactive`: neither send nor receive.

The interaction matters: if the offerer says sendrecv, the answerer can say sendrecv (full duplex), sendonly (offerer becomes recvonly), recvonly (offerer becomes sendonly), or inactive (no media).

The answerer's "inverse" rule: the answerer's sendonly corresponds to offerer's recvonly, and vice versa. This is intuitive — "I send" maps to "you receive" — but easy to get wrong in code.

Codec negotiation: each m-line has an `m=audio <port> RTP/AVP <pt1> <pt2> ...` listing payload types. `a=rtpmap:<pt> <name>/<clock>` defines the codec for each payload type. The answer must include a subset of the offered payload types (in any order, but the first in the answer is the preferred codec).

## Hold Pattern

Putting a call on hold is signaled via SDP modification, sent in re-INVITE or UPDATE.

Historical (RFC 2543, deprecated): change the connection address to `c=IN IP4 0.0.0.0`. The 0.0.0.0 is sentinel for "no media address". Receivers stop sending; senders may stop too.

Modern (RFC 3264): change media direction to `a=sendonly`. The held side becomes sendonly (it sends music-on-hold but doesn't receive); the holder becomes implicitly recvonly. To resume, re-INVITE with `a=sendrecv`.

For double-hold (both parties hold), use `a=inactive`. Both sides stop sending and receiving.

The 0.0.0.0 method has issues with NAT and ICE — the address is invalid on the wire and breaks connectivity checks. Modern implementations use sendonly/inactive exclusively.

The call flow:
1. Alice presses Hold.
2. Alice sends re-INVITE with `a=sendonly` in SDP.
3. Bob's UA answers 200 OK with `a=recvonly` (the inverse).
4. Alice ACKs.
5. Alice plays music-on-hold to Bob (sendonly direction).
6. Bob hears music-on-hold; Bob's UA does not send media.

To resume:
1. Alice presses Resume.
2. Alice sends re-INVITE with `a=sendrecv`.
3. Bob answers `a=sendrecv`.
4. Alice ACKs.
5. Both directions of media resume.

## Transfer (REFER)

REFER (RFC 3515) is the mechanism for transferring a call. The transferor sends REFER to the transferee; the transferee then initiates a new INVITE to the transfer target.

Blind transfer: Alice (the transferor) sends REFER to Bob (the transferee) with `Refer-To: sip:carol@example.com`. Bob receives 202 Accepted. Bob's UA sends INVITE to Carol. Bob hangs up the call with Alice (BYE) once Carol answers (or fails).

Attended transfer: Alice has two calls — one with Bob and one with Carol. Alice sends REFER to Bob with `Refer-To: <sip:carol@example.com?Replaces=<call-id>;to-tag=<tag>;from-tag=<tag>>`. The Replaces header tells Bob to replace his existing call with Carol with a new call to Carol. Bob's UA sends INVITE to Carol with `Replaces: <dialog-id>`. Carol's UA sees the Replaces, replaces the dialog with Alice, and the old call is implicitly terminated.

NOTIFY for transfer status: REFER establishes an implicit subscription. The transferee sends NOTIFY messages to the transferor reporting on the transfer's progress (`SIP/2.0 200 OK` for success, `SIP/2.0 4xx` for failure). The Subscription-State header indicates active or terminated.

The Replaces header (RFC 3891) is critical for attended transfer because it lets the transferred-to party (Carol) reuse her existing dialog with Bob, rather than creating a new dialog. This avoids the "call drop" appearance from Carol's side.

## SIP-INFO vs INFO Package

SIP INFO (RFC 6086) is a method for sending mid-dialog information without changing the dialog state. The original INFO method was generic — the body could contain anything, with no schema.

The "INFO Package" framework (RFC 6086) introduces structured info: an `Info-Package` header names the package, and the body's content-type matches the package's schema. Examples: `application/dtmf-relay+xml` for DTMF tones (RFC 7038), `application/conference-info+xml` for conference state.

The historical use case: in-band signaling that doesn't fit INVITE/UPDATE/PRACK/REFER. Examples include:
- DTMF digits during a call (now RFC 4733 RTP-based DTMF is preferred).
- Vendor-specific call features (e.g., Cisco's call-park notification).
- Conference event package (announcing participants).

INFO is generally discouraged for new signaling; SUBSCRIBE/NOTIFY or specific RFC-defined methods are preferred. The reason: INFO bypasses the SIP state machine — it doesn't fit the dialog-modification pattern, doesn't have well-defined error semantics, and doesn't compose well with existing event packages.

## Common Pitfalls

**Via header order**: Via headers stack on the way out (UAC at bottom, last proxy at top); responses unwind from top to bottom. Reordering or dropping Via headers breaks the response routing. Most proxies are forbidden from reordering.

**rport vs received parameter**: When a proxy sees a request from a NATed peer, the source IP doesn't match the Via's sent-by host. RFC 3581 introduces `rport` — the UAC adds `;rport` to its Via, the proxy responds with `;received=<source-IP>;rport=<source-port>`, and the response is sent to the received/rport tuple. Implementations that don't support rport can't traverse symmetric NAT.

**CSeq mismatch**: each method has its own CSeq sequence within a dialog. Re-INVITE increments CSeq from the previous re-INVITE; BYE has its own CSeq; OPTIONS has another. Implementations that don't track per-method CSeq correctly can produce out-of-order CSeq, which causes 481 or silent drops.

**Max-Forwards 0 looping**: Max-Forwards (default 70) is a backstop for routing loops. Each proxy decrements; at 0, the proxy responds 483 Too Many Hops. If a routing loop exists and Max-Forwards is high enough to not catch it, the request circulates until Max-Forwards hits 0. Loop detection via Via headers is the primary defense; Max-Forwards is the secondary.

**Contact missing in REGISTER**: a REGISTER with no Contact is a query for current bindings, not a registration. Some implementations treat empty Contact as "register at the source IP", which is non-standard.

**Tags on To header in REGISTER**: REGISTER MUST NOT have a To-tag. Adding one creates ambiguity with dialog matching.

**ACK to 2xx using transaction layer**: as discussed, ACK to 2xx is end-to-end and uses the dialog layer, not the transaction layer. Sending ACK with the original INVITE's branch is a common bug.

## SIPS vs SIP

The `sips:` URI scheme requires TLS on every hop from caller to callee. Specifically: for every Route header and the Request-URI, SIPS implies that the next-hop transport must be TLS. This is "hop-by-hop SIPS".

In practice, true hop-by-hop SIPS is rare. Most providers use TLS only on the access leg (UAC ↔ outbound proxy) and switch to UDP or TCP for inter-provider transit. This violates SIPS semantics but is operationally pragmatic.

The pragmatic replacement is "best-effort SIP-over-TLS": the UAC uses sip: URIs but specifies `transport=tls` in the URI's transport parameter. The proxy chain attempts TLS on each hop where the upstream supports it. There are no semantic guarantees, but the result is "TLS where possible".

Some implementations extend SIPS to be downgrade-tolerant: SIPS in the URI but accept TCP on the next hop if TLS isn't available. This is non-conformant with RFC 3261's strict SIPS semantics, but operationally common.

The takeaway: SIPS is a strong-but-rare guarantee; SIP-with-TLS is a weak-but-common one. Production SIP deployments usually run SIP-over-TLS without claiming SIPS conformance.

## Timer Math: Why 32 Seconds

The choice of 64×T1 = 32 seconds for give-up timers (Timer B, Timer F, Timer H, Timer J) is not arbitrary. It is derived from the geometric series of retransmissions plus a buffer for in-flight messages.

The retransmission schedule for INVITE Client Transaction:
- t=0: send INVITE.
- t=0.5: retransmit (Timer A=T1=500 ms doubled, but first fire is at T1).
- Actually corrected: Timer A starts at T1=500ms; first retransmission at t=0.5s; second at t=1.5s (Timer A doubled to 1s, fires 1s after restart); third at t=3.5s (Timer A=2s); fourth at t=7.5s (Timer A=4s); fifth at t=15.5s (Timer A=8s); sixth at t=31.5s (Timer A=16s).
- t=32: Timer B fires; transaction terminates.

The cumulative retransmission count is 6, and the elapsed time is 32 seconds. With T1=500ms, this is 64×T1=32s. The factor of 64 comes from the binary exponentiation: 1+2+4+8+16+32 = 63, so 6 retransmissions cover 63 T1 intervals; the 64th is the give-up.

If T1 is increased (e.g., on high-latency links), all timers scale proportionally. T1=1500ms (1.5s) means Timer B = 96s. This is rare but supported.

The 32-second value matches typical user expectations: a phone that "rings" for 30+ seconds without an answer is socially understood to be ignored; longer waits are unusual. The transaction layer's 32s give-up aligns with this convention.

For non-INVITE (Timer F=64×T1=32s), the same arithmetic. But the retransmission schedule differs: Timer E doubles up to T2=4s, then stays at 4s. So retransmissions occur at t=0.5, 1.5, 3.5, 7.5, 11.5, 15.5, 19.5, 23.5, 27.5, 31.5 — ten retransmissions in 32 seconds, vs six for INVITE. The non-INVITE produces denser retransmissions because non-INVITE requests are smaller and the bandwidth cost is lower.

## Mealy vs Moore: Why SIP Chose Mealy

A Mealy machine outputs depend on (state, input). A Moore machine outputs depend on (state) only. SIP's transaction state machines are Mealy.

The choice matters because SIP's behavior is input-sensitive. Receiving 200 OK in Proceeding produces output A (forward 2xx, send ACK); receiving 200 OK in Completed produces output B (re-send ACK). The output is not a function of state alone; it depends on which message arrives.

A Moore machine equivalent would require more states. To represent "Proceeding-having-just-received-200" vs "Completed-having-just-received-200" as separate states, the state space doubles. Mealy compresses by encoding the input in the transition.

The downside of Mealy: outputs can change asynchronously with respect to state. If the state-change is delayed (e.g., due to a slow processing thread), the output might be emitted in the "wrong" state. SIP avoids this by making state transitions atomic with the corresponding outputs.

Implementations typically code SIP transaction state machines as table-driven dispatch:
```c
typedef enum { CALLING, PROCEEDING, COMPLETED, ACCEPTED, TERMINATED } state_t;
typedef enum { EV_1XX, EV_2XX, EV_3XX_6XX, EV_TIMER_A, EV_TIMER_B, EV_TIMER_D, EV_TIMER_M } event_t;

typedef struct {
    state_t next_state;
    void (*action)(transaction_t *);
} transition_t;

static transition_t transitions[5][7] = {
    /* CALLING */    { {PROCEEDING, on_1xx}, {ACCEPTED, on_2xx}, {COMPLETED, on_3xx_6xx}, {CALLING, retransmit_invite}, {TERMINATED, timeout}, {0, NULL}, {0, NULL} },
    /* PROCEEDING */ { {PROCEEDING, on_1xx}, {ACCEPTED, on_2xx}, {COMPLETED, on_3xx_6xx}, {0, NULL}, {0, NULL}, {0, NULL}, {0, NULL} },
    /* COMPLETED */  { {0, NULL}, {0, NULL}, {COMPLETED, resend_ack}, {0, NULL}, {0, NULL}, {TERMINATED, cleanup}, {0, NULL} },
    /* ACCEPTED */   { {0, NULL}, {ACCEPTED, on_2xx_retransmit}, {0, NULL}, {0, NULL}, {0, NULL}, {0, NULL}, {TERMINATED, cleanup} },
    /* TERMINATED */ { {0, NULL}, {0, NULL}, {0, NULL}, {0, NULL}, {0, NULL}, {0, NULL}, {0, NULL} }
};
```

This kind of table makes the Mealy-machine structure explicit and is the typical implementation pattern.

## Loop Detection Algorithm (RFC 3261 §16.3)

Routing loops in SIP can occur when proxies have inconsistent routing data. Without loop detection, a request can circulate indefinitely (until Max-Forwards hits zero, which is the backstop).

The primary loop detection uses the Via header. When a proxy P forwards a request, it adds a Via header with a unique branch derived from the input request's Via stack:

```
new_branch = z9hG4bK + hash(input_top_via_branch || From-tag || To-tag || Call-ID || CSeq || Request-URI)
```

The hash function is implementation-defined (typically MD5 or SHA-1, taking the first 8 hex digits). The key property: if the proxy receives a request whose Via stack includes a Via with sent-by=P and branch=new_branch (for any value of new_branch the proxy ever produced for this transaction), the request has looped.

Loop detection at proxy P:
1. Extract input Via top-most branch.
2. Iterate through Via headers in the request; for each Via with sent-by=P:
   a. Compute the branch P would have produced for this request (using the saved branch and current From/To/Call-ID/CSeq).
   b. If the saved branch matches a Via in the stack, the request has looped.
3. If loop detected, respond 482 Loop Detected.

The matching is "would-have-produced": the proxy regenerates the branch it would assign to this request (for outbound) and checks if that branch is already in the Via stack. If yes, the request has visited the proxy before with the same routing context.

Spiral detection: if a request visits the same proxy with different routing context (e.g., different Request-URI), this is a spiral, not a loop. Spirals are valid (e.g., a redirect causes the proxy to forward to a new URI). Loops are invalid.

The distinction between loop and spiral is encoded in the branch hash inputs. Branch is a function of (Via, From, To, Call-ID, CSeq, Request-URI). If Request-URI changes, the branch changes, and the same Via is not a loop. If Request-URI is the same, the same Via is a loop.

## OPTIONS Probing (Keepalive)

OPTIONS is a non-INVITE method that queries a UA's capabilities. In practice, OPTIONS is also used as a keepalive: a periodic OPTIONS to a peer verifies that the peer is reachable and responsive.

Common OPTIONS keepalive intervals:
- Endpoint-to-proxy: every 30-60 seconds (to maintain NAT bindings, which typically time out at 30-180s).
- Proxy-to-proxy: every 5-10 minutes (lower frequency since both ends are stable).
- Trunk-to-trunk: every 30 seconds (for SIP trunks where uptime is critical).

The OPTIONS keepalive serves several purposes:
1. **NAT traversal**: keeps the NAT mapping alive.
2. **Reachability check**: verifies the peer is up.
3. **Capability re-discovery**: the peer's response includes Allow, Supported, Accept headers; the requestor can detect capability changes.

The cost: each OPTIONS adds traffic. For 1000 endpoints with 30-second OPTIONS, that's 33 OPTIONS/second per direction = 67 OPTIONS/second total to a centralized SIP proxy. At ~500 bytes per OPTIONS, that's ~33 KB/s of overhead — significant for low-bandwidth links.

Alternatives to OPTIONS keepalive:
- **CRLF keepalive (RFC 5626)**: SIP-Outbound clients send a bare CRLF over TCP/TLS as a keepalive. Lighter than OPTIONS.
- **STUN keepalive**: the media-path STUN keepalive doubles as a NAT-binding refresh for the signaling path (when both flow through the same NAT).
- **No keepalive**: rely on registration refresh (REGISTER) for the keepalive function.

## Record-Route Analysis

Record-Route is the proxy's mechanism for inserting itself into the dialog's route. When a proxy adds Record-Route to a request, the dialog's route set will include this proxy, ensuring all in-dialog requests (BYE, re-INVITE, ACK for 2xx) traverse this proxy.

The use cases:
1. **Stateful media: the proxy needs to track call state for billing, lawful intercept, or feature insertion (e.g., music-on-hold, conference-mixing).
2. **Topology hiding**: the proxy presents itself as the "endpoint" to the other side, hiding the actual UA behind it.
3. **NAT traversal**: the proxy provides a stable, public-IP signaling endpoint for NAT-bound UAs.

Record-Route is two-way: each Record-Route header in the response is added to the dialog's route set. The order matters: routes are stacked in the order proxies process the request, then reversed for the response.

The "lr" (loose routing, RFC 3261) parameter on Record-Route URIs is essential. Without lr, the proxy expects strict routing (where the Request-URI is rewritten at each hop); with lr, the proxy uses Route headers and keeps the Request-URI intact. Modern SIP uses loose routing exclusively.

Implications for proxies that don't Record-Route:
- The dialog's route set is empty (or contains only proxies that did Record-Route).
- Subsequent in-dialog requests bypass the proxy.
- The proxy loses visibility into the call after INVITE.

For features that require in-call visibility (e.g., per-call billing, mid-call feature insertion), Record-Route is mandatory.

## SIP-Frag and SIP-Specific Event Notification (RFC 3265, RFC 6665)

SUBSCRIBE/NOTIFY is the SIP event-notification framework. The SUBSCRIBE method establishes a subscription; the NOTIFY method delivers events.

The SUBSCRIBE flow:
1. Subscriber sends SUBSCRIBE with `Event: <package-name>` (e.g., `presence`, `dialog`, `message-summary`).
2. Notifier responds 200 OK (subscription established).
3. Notifier immediately sends NOTIFY with the current state.
4. Notifier sends additional NOTIFYs as state changes.
5. Subscription expires per `Expires` header (typically 3600s); subscriber sends fresh SUBSCRIBE to refresh.

Event packages are vendor-specific or RFC-defined:
- **presence (RFC 3856)**: presence info.
- **dialog (RFC 4235)**: call state (idle, ringing, in-call, hold).
- **message-summary (RFC 3842)**: voicemail count.
- **conference (RFC 4575)**: conference participant list.
- **kpml (RFC 4730)**: keypad markup language for DTMF.

NOTIFY message bodies are package-specific. For dialog: an XML document describing all dialogs the subscribed-to entity is part of.

SIP-Frag (RFC 3420) is a fragment format for SIP messages, used in NOTIFY bodies to convey partial SIP messages (e.g., the status of a transferred call, conveyed as a fragment of the NOTIFY).

## Forking Variants

The basic fork-and-collect model has variants:

**Parallel forking with race**: send INVITE to all branches simultaneously; first 2xx wins. Latency = min(branch latencies).

**Sequential forking with timeout**: try one branch; if 4xx-6xx (or no response in T seconds), try next. Latency = sum of failed-branch latencies + winning-branch latency.

**Hybrid forking**: parallel within priority groups; sequential between groups. Priority assigned per Contact (q-value, RFC 3261). Highest q-value group tried first; if all fail, next-priority group.

**Static forking**: the fork list is determined at the proxy by its configuration (not by Contact bindings). Useful for hunt groups (a set of phones that should all ring on a call to a published number).

**Dynamic forking**: the fork list is determined per-call by application logic (e.g., based on time-of-day, caller-ID, customer-status).

The CANCEL handling: when one branch wins, the proxy CANCELs the others. The timing matters: if a CANCEL arrives at a UAS that has just sent 2xx, the UAS responds 487 (Request Terminated for the CANCELed branch's INVITE) or — if 2xx already left — the UAS will retransmit 2xx, and the proxy must absorb (since the proxy already forwarded the winning 2xx). The "stray 2xx" handling absorbs these by sending BYE.

## SDP Negotiation Pathologies

**Codec mismatch**: offerer offers G.722; answerer's policy disallows G.722. Answerer responds 488 Not Acceptable Here. Caller can retry with a different codec list, or call fails.

**Asymmetric codec lists**: offerer lists G.722, G.711; answerer responds with G.711 only. Both sides use G.711 (the answer is the contract).

**Late offer**: the INVITE has no SDP body. The 2xx has the offer; the ACK has the answer. This is "late SDP" (RFC 3261 §13.2.1 forbids it for 2xx, but it's sometimes used). Most modern endpoints reject late offer.

**Re-offer in re-INVITE**: re-INVITE may carry a fresh offer. The answer is in the 2xx response. The previous SDP is discarded.

**SDP-less re-INVITE**: re-INVITE with no SDP is interpreted as "no media change". The 2xx is also SDP-less.

**Glare on offer**: both sides simultaneously send re-INVITE with offer. RFC 3261 §14.1 specifies the 491 Request Pending mechanism (described in the Re-INVITE section).

## ENUM (RFC 6116)

ENUM (E.164 NUmber Mapping) is a DNS-based mechanism for mapping E.164 phone numbers to URIs. The format: a phone number is represented as a DNS name in the e164.arpa domain by reversing the digits and adding "e164.arpa".

Example: phone number +1-202-555-0100 becomes 0.0.1.0.5.5.5.2.0.2.1.e164.arpa.

The DNS NAPTR record at this name maps the phone number to a URI:
```
0.0.1.0.5.5.5.2.0.2.1.e164.arpa. NAPTR 100 10 "u" "E2U+sip" "!^.*$!sip:alice@example.com!" .
```

The "!^.*$!sip:...!" is a sed-like substitution that transforms the input phone number into the SIP URI. In simple cases, the substitution is a constant (the URI doesn't depend on the phone number).

ENUM is rarely deployed publicly (the e164.arpa root is not authoritatively populated for most numbers). Private ENUM is more common: a service provider maintains its own ENUM root for its customers' numbers.

The use case: a SIP user agent dialing a phone number can ENUM-resolve the number to a SIP URI and bypass the PSTN entirely. If both endpoints are ENUM-enabled, calls are SIP-end-to-end with no PSTN interconnect costs.

## TLS Cipher Suites for SIPS

SIPS requires TLS. The cipher suite negotiation must balance security and interoperability:

Modern recommendations (TLS 1.3, RFC 8446):
- TLS_AES_128_GCM_SHA256
- TLS_AES_256_GCM_SHA384
- TLS_CHACHA20_POLY1305_SHA256

Legacy (TLS 1.2, RFC 5246):
- TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
- TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
- TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
- TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384

Avoid:
- Any RC4 cipher (broken).
- Any cipher with CBC mode and HMAC-SHA1 (vulnerable to BEAST, Lucky13).
- Any cipher with RSA key exchange (no forward secrecy).
- TLS 1.0 / 1.1 (deprecated; vulnerable to BEAST and POODLE).

For SIPS, mTLS (mutual cert authentication) is recommended for inter-proxy trust. The cert chains are typically rooted in private CAs operated by the provider.

## SRTP Key Negotiation

Media (RTP) is encrypted via SRTP. The encryption key must be negotiated; the negotiation methods:

**SDES (RFC 4568)**: Session Description Protocol Security Descriptions. The key is in the SDP as `a=crypto`. The SDP is sent over TLS-protected SIP (sips: or SIP-over-TLS), so the key is confidential as long as TLS is intact.

**ZRTP (RFC 6189)**: peer-to-peer key agreement performed in the media path (RTP). Independent of SIP; SDP is not involved. Provides forward secrecy without requiring TLS on SIP.

**DTLS-SRTP (RFC 5763)**: TLS handshake performed in the media path; the SRTP keys are derived from the DTLS master secret. SDP carries `a=fingerprint` (the DTLS cert hash) and `a=setup` (DTLS role). This is the mandatory mode for WebRTC.

**MIKEY (RFC 3830)**: Multimedia Internet KEYing. A key-exchange protocol; can be used with SIP via `a=key-mgmt:mikey` in SDP. Less common in practice.

Each method has tradeoffs:
- SDES: simple but depends on SIP-path TLS; if TLS fails, SRTP keys are leaked.
- ZRTP: independent of SIP TLS; uses Diffie-Hellman in the media path; may be blocked by media-path firewalls.
- DTLS-SRTP: most secure (forward secrecy, MITM detection via fingerprint binding); the WebRTC standard.
- MIKEY: flexible but complex; rare in production.

## SIP-T and SIP-I (PSTN Interworking)

When SIP interconnects with PSTN, ISUP (ISDN User Part) signaling must be converted to/from SIP. SIP-T (RFC 3372) and SIP-I (ITU-T Q.1912.5) are encapsulation profiles.

SIP-T encapsulates ISUP messages in SIP body parts (multipart/mixed; one part is SDP, the other is the ISUP message). The SIP message acts as a transport for the ISUP message.

SIP-I is similar but specifies more detail on ISUP-to-SIP mapping (which SIP method corresponds to which ISUP message; how Cause codes map to SIP status codes).

Use case: a tandem switch that handles both SIP and PSTN traffic. Calls traversing the switch from SIP to PSTN are converted; calls from PSTN to SIP are also converted. The encapsulation preserves the original ISUP information so it can be reconstructed on the other side.

In practice, SIP-T and SIP-I are used in carrier-grade VoIP networks (interconnecting Tier-1 carriers) and rare in enterprise SIP.

## Header Folding and Compact Forms

SIP message parsing must handle folded headers and compact header forms.

**Header folding**: long headers can be split across multiple lines by inserting CRLF + whitespace. Example:
```
Via: SIP/2.0/UDP 192.0.2.5:5060;branch=z9hG4bK-abc,
     SIP/2.0/UDP 192.0.2.10:5060;branch=z9hG4bK-def
```
The parser must un-fold (replace CRLF+WS with a single space) before processing.

**Compact forms**: certain headers have one-letter aliases for bandwidth efficiency:
- `i` = Call-ID
- `m` = Contact
- `e` = Content-Encoding
- `l` = Content-Length
- `c` = Content-Type
- `f` = From
- `s` = Subject
- `k` = Supported
- `t` = To
- `v` = Via

Compact forms are still common in WebRTC and IMS deployments where bandwidth matters. A robust parser must accept both forms interchangeably.

## SIP Message Length Considerations

Maximum SIP message size is implementation-defined but RFC 3261 specifies a minimum of 1300 bytes for UDP (to fit in an Ethernet MTU).

For larger messages (e.g., INVITEs with full SDP including video, ICE candidates, multiple audio codecs), UDP fragmentation becomes a concern. RFC 3261 §18.1.1 specifies that if a UDP message exceeds path MTU, the sender SHOULD use TCP instead.

The cutover heuristic: compute the message size; if it exceeds 1300 bytes, switch to TCP. The 1300 figure is conservative (Ethernet MTU 1500 minus IP/UDP headers minus a safety margin for tunnels).

In practice, modern endpoints (especially WebRTC) routinely produce >1300 byte INVITEs and use TCP/TLS or WebSocket exclusively. Pure-UDP SIP is rare in modern deployments.

## SIPFRAG and Multipart Bodies

SIP message bodies can be multipart/mixed, multipart/alternative, or multipart/related (MIME conventions).

Common uses:
- **multipart/mixed**: SDP + ISUP (for SIP-T/SIP-I PSTN interworking).
- **multipart/related**: SDP + JPEG (for photo-of-caller in some legacy systems).
- **message/sipfrag**: a SIP message fragment, used in NOTIFY for transfer status.

Example multipart body:
```
Content-Type: multipart/mixed; boundary=boundary42

--boundary42
Content-Type: application/sdp

v=0
o=alice 53655765 2353687637 IN IP4 192.0.2.5
s=-
c=IN IP4 192.0.2.5
t=0 0
m=audio 49170 RTP/AVP 0
a=rtpmap:0 PCMU/8000

--boundary42
Content-Type: application/isup

(binary ISUP message)

--boundary42--
```

The parser must split on the boundary string and decode each part separately.

## SIP Privacy (RFC 3323, RFC 3325)

RFC 3323 introduces the Privacy header for caller-anonymization. Values:
- `none`: no privacy (default).
- `header`: hide From and other identifying headers.
- `session`: hide media-session details (don't relay caller's IP in SDP).
- `user`: anonymize the user-portion of From.
- `id`: hide the P-Asserted-Identity header.

The Privacy header is an instruction to the proxy; the proxy is responsible for anonymizing on behalf of the user.

RFC 3325 introduces P-Asserted-Identity (PAI), a header inserted by the originating proxy carrying the caller's "real" identity (verified by the proxy). When Privacy: id is set, the PAI is stripped before the request leaves the trust domain.

The trust domain: a set of proxies that mutually trust each other to populate PAI correctly. Typically a single provider or a federation of providers.

Use case: legal interception or billing requires the real caller-ID; the called party should see a privacy-asserted name (e.g., "Anonymous"). PAI lets the provider track the real caller while presenting an anonymized one to the callee.

## SIP Service Examples (Asterisk-style Dialplan)

A SIP-based PBX (e.g., Asterisk, FreeSWITCH) implements SIP services via a dialplan. Common patterns:

**Inbound DID routing**:
```
exten => +12025550100,1,Dial(SIP/alice)
exten => +12025550101,1,Dial(SIP/bob)
exten => +12025550102,1,Voicemail(carol@default)
```

**Hunt group**:
```
exten => 100,1,Dial(SIP/alice&SIP/bob&SIP/carol,30)
exten => 100,n,Voicemail(group100@default)
```

**Time-of-day routing**:
```
exten => 100,1,GotoIfTime(09:00-17:00|mon-fri|*|*?business)
exten => 100,n,Goto(after-hours)
exten => 100,n(business),Dial(SIP/alice)
exten => 100,n(after-hours),Voicemail(alice@default)
```

**Conference**:
```
exten => 600,1,Answer()
exten => 600,n,ConfBridge(roomA)
```

These map to underlying SIP transactions: Dial creates an outbound INVITE; ConfBridge anchors media at the PBX; Voicemail records to a file. The dialplan is the application logic; SIP is the protocol.

## SIP Forking and Distributed Ringing

When a single user has multiple phones (desk phone, mobile softclient, tablet), forking allows all phones to ring simultaneously.

The configuration:
- User has multiple Contact bindings (one per phone, registered separately).
- Proxy forks INVITE to all Contacts.
- All phones ring; first to answer wins.
- Other phones receive CANCEL.

The race condition: if two phones answer simultaneously (e.g., user picks up desk phone and mobile rings just as user reaches over), both phones produce 2xx. The proxy selects one as winner; the loser is sent CANCEL (or BYE if 2xx already left).

Some systems implement "pickup notifications": when one phone answers, the others display a "call answered elsewhere" message instead of just stopping the ring. This is implemented via SUBSCRIBE/NOTIFY on the dialog package.

## Call Transfer Variants

**Blind transfer (unattended)**: A presses transfer, dials C's number, hangs up. A's phone sends REFER to B with Refer-To: <C>. B's phone calls C. A is no longer involved.

**Attended transfer (consultative)**: A presses transfer, calls C, waits for C to answer, presses transfer again. A's phone sends REFER to B with Refer-To: <C?Replaces=<A-C-dialog>>. B's phone calls C, includes Replaces header. C's phone sees Replaces, replaces the A-C dialog with the B-C dialog. A and the original A-C call are dropped.

**Semi-attended transfer**: A presses transfer, calls C, presses transfer before C answers. Variant of blind transfer with the call to C already in progress.

**Three-way conferencing**: A, B, C are in a single conference. Implemented either by the PBX (which mixes audio) or by a media server. SIP signaling: A is in two dialogs (A-B and A-C); A's phone (or a conference-bridge endpoint) mixes the audio.

## SIP Timing Diagrams

A canonical INVITE with 200 OK and BYE:
```
UAC                  Proxy                  UAS
 |                    |                      |
 |--- INVITE -------->|                      |
 |                    |--- INVITE ---------->|
 |                    |<--- 100 Trying ------|
 |<-- 100 Trying -----|                      |
 |                    |<--- 180 Ringing -----|
 |<-- 180 Ringing ----|                      |
 |                    |<--- 200 OK ----------|
 |<-- 200 OK ---------|                      |
 |--- ACK ---------------------------------->|  (end-to-end for 2xx)
 |                                            |
 |==== RTP media (bidirectional) ============ |
 |                                            |
 |--- BYE ------------>                       |
 |                    |--- BYE ------------->|
 |                    |<--- 200 OK ----------|
 |<-- 200 OK ---------|                      |
```

A canonical INVITE with 486 Busy Here:
```
UAC                  Proxy                  UAS
 |                    |                      |
 |--- INVITE -------->|                      |
 |                    |--- INVITE ---------->|
 |                    |<--- 100 Trying ------|
 |<-- 100 Trying -----|                      |
 |                    |<--- 486 Busy --------|
 |                    |--- ACK ------------->|  (hop-by-hop for non-2xx)
 |<-- 486 Busy -------|                      |
 |--- ACK ----------->|                      |
```

The ACK paths differ — for 2xx, end-to-end through proxy; for non-2xx, hop-by-hop generated by proxy. This is a frequent source of confusion.

## Common SIP Status Codes Reference

The full status-code taxonomy is RFC 3261 §21. Commonly seen:

**1xx Provisional**:
- 100 Trying — UAS received and is processing.
- 180 Ringing — phone is ringing.
- 181 Call Is Being Forwarded.
- 182 Queued — call is queued.
- 183 Session Progress — provisional with media (early-media SDP).

**2xx Success**:
- 200 OK — request succeeded.
- 202 Accepted — request accepted, processing async (used for REFER, SUBSCRIBE).

**3xx Redirection**:
- 301 Moved Permanently — new URI is permanent.
- 302 Moved Temporarily — new URI is temporary; subsequent requests use original URI.
- 305 Use Proxy.

**4xx Client Error**:
- 400 Bad Request — malformed.
- 401 Unauthorized — auth required (UAS-level).
- 403 Forbidden — auth doesn't help.
- 404 Not Found — user not found at this domain.
- 405 Method Not Allowed.
- 407 Proxy Authentication Required (proxy-level auth).
- 408 Request Timeout.
- 410 Gone.
- 415 Unsupported Media Type.
- 480 Temporarily Unavailable.
- 481 Call/Transaction Does Not Exist.
- 482 Loop Detected.
- 483 Too Many Hops (Max-Forwards expired).
- 484 Address Incomplete.
- 486 Busy Here.
- 487 Request Terminated (typically after CANCEL).
- 488 Not Acceptable Here (codec mismatch).
- 491 Request Pending (glare).

**5xx Server Error**:
- 500 Server Internal Error.
- 503 Service Unavailable (overload).

**6xx Global Failure**:
- 600 Busy Everywhere.
- 603 Decline.
- 604 Does Not Exist Anywhere.
- 606 Not Acceptable.

The 6xx is "globally true" — no other branch should be tried (in fork scenarios). 4xx-5xx are "branch-specific" — fork can try other branches.

## References

- **RFC 3261** — SIP: Session Initiation Protocol (https://www.rfc-editor.org/rfc/rfc3261.html)
- **RFC 3262** — Reliability of Provisional Responses in SIP (https://www.rfc-editor.org/rfc/rfc3262.html)
- **RFC 3263** — Locating SIP Servers (https://www.rfc-editor.org/rfc/rfc3263.html)
- **RFC 3264** — An Offer/Answer Model with SDP (https://www.rfc-editor.org/rfc/rfc3264.html)
- **RFC 3311** — The SIP UPDATE Method (https://www.rfc-editor.org/rfc/rfc3311.html)
- **RFC 3515** — The SIP Refer Method (https://www.rfc-editor.org/rfc/rfc3515.html)
- **RFC 3581** — An Extension to SIP for Symmetric Response Routing (rport)
- **RFC 3891** — The SIP Replaces Header
- **RFC 5245** — Interactive Connectivity Establishment (ICE) (obsoleted by RFC 8445)
- **RFC 5389** — Session Traversal Utilities for NAT (STUN) (obsoleted by RFC 8489)
- **RFC 5626** — Managing Client-Initiated Connections in SIP (Outbound)
- **RFC 5766** — Traversal Using Relays around NAT (TURN) (obsoleted by RFC 8656)
- **RFC 6026** — Correct Transaction Handling for 2xx Responses to INVITE
- **RFC 6086** — SIP INFO Method and Package Framework
- **RFC 6455** — The WebSocket Protocol
- **RFC 7038** — INFO Package for DTMF
- **RFC 7118** — The WebSocket Protocol as a Transport for SIP
- **RFC 7616** — HTTP Digest Access Authentication
- **RFC 8224** — Authenticated Identity Management in SIP (STIR)
- **RFC 8225** — PASSporT: Personal Assertion Token
- **RFC 8226** — Secure Telephone Identity Credentials
- **RFC 8445** — Interactive Connectivity Establishment (ICE), revision
- **RFC 8489** — STUN, revision
- **RFC 8656** — TURN, revision
- **RFC 8838** — Trickle ICE
- **RFC 8866** — SDP: Session Description Protocol, revision
- **ATIS-1000074** — SHAKEN: Signature-based Handling of Asserted information using toKENs
- Henning Schulzrinne, "The Session Initiation Protocol (SIP)", IEEE Communications Magazine, 2003
- Mark Handley et al., "SIP: Session Initiation Protocol", original 1999 IETF design notes
- Alan B. Johnston, "SIP: Understanding the Session Initiation Protocol", 4th ed., Artech House, 2015
