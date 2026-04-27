# gRPC — ELI5

> gRPC is a phone call between two computer programs where both sides agreed in advance on the alphabet.

## Prerequisites

It helps to know what an **API** is. An API ("Application Programming Interface") is just a way for one program to ask another program to do something. If your phone app shows the weather, it didn't make up the weather; it called an API somewhere on the internet that knew the weather, and that API answered.

If "API" is brand-new to you, here is the one-sentence version: an API is a contract that says "if you send me a message that looks like *this*, I will send you a message back that looks like *that*." That's it. The whole rest of this sheet is just one specific way of doing that.

It also helps (a tiny bit) to know what TCP and TLS are. TCP is the part of the internet that makes sure messages arrive in order and don't get lost. TLS is the part that puts the messages in a locked envelope so people in the middle can't read them. If you've never heard of either, do not panic — go run:

```bash
$ cs ramp-up tcp-eli5
$ cs ramp-up tls-eli5
```

…and they will explain those to you the same way this sheet is going to explain gRPC. You can also just keep reading this sheet and look up any weird word in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

## What Even Is gRPC

### A phone call between programs

Imagine two friends, Alice and Bob, who live in different cities. They need to talk to each other every day, but they need to be **really fast** about it because they are running a business together and they don't have time to waste.

If Alice writes Bob a letter, that takes days. If Alice texts Bob a long message in plain English, Bob has to read every word, figure out what Alice meant, decide what to do, and then write a long message back. That is slow.

Instead, Alice and Bob agree to do this:

1. They both buy the same phone.
2. They both agree, ahead of time, on a list of short codes. "When I say `1`, that means 'how are you?'. When I say `2`, that means 'send me the order list.' When I say `3`, that means 'tell me my balance.'"
3. They both write down those codes on the same piece of paper. Alice has a copy. Bob has a copy.
4. Now Alice can call Bob and just say "3" and Bob immediately knows she's asking for her balance and sends a tiny number back. They don't need full sentences. They have a shared shorthand.

**That is gRPC.**

The phone is the network. The list of short codes is the **schema** (we'll get to that). Alice is the **client**. Bob is the **server**. The act of Alice asking Bob to do something is a **remote procedure call**, or RPC, which is a fancy way of saying "make a function call across a network as if the function were sitting on your own computer."

The "g" at the front officially does not stand for anything (Google's joke is that it stands for a different word every release — "good," "green," "glamorous," "gravity," "gizmo"…). Most people just say "Google" because Google made it. You don't need to remember that. You just need to remember **the phone-call analogy**.

### Why anybody bothered building this

You might be thinking: "We already have the web. Programs already talk to each other. Why do we need a new thing?"

Here's why. The way most websites and apps talk today is called **REST + JSON over HTTP/1**. Let's break that down:

- **REST** is a style. It says "I will model everything as a resource and use the verbs `GET`, `POST`, `PUT`, `DELETE` to act on it." It's a convention, not a strict rule.
- **JSON** is a way of writing data as text. `{"id": 123, "name": "Alice"}` is JSON. Humans can read it. Computers can read it. It's friendly. But it's bulky — every field name (`id`, `name`) is repeated in every single message, even though both sides already know what the fields are.
- **HTTP/1** is the old version of the web's transport protocol. Every time a program wants to ask another program something, it opens a connection, sends a message, gets a reply, and (often) closes the connection. Opening and closing connections is slow.

Now imagine Alice and Bob are using REST + JSON. Every time Alice wants to ask "what's my balance?" she has to:

1. Open a fresh connection (slow).
2. Spell out, in full English text, "GET /accounts/3/balance HTTP/1.1" plus a bunch of headers (verbose).
3. Wait for Bob to send back JSON like `{"account_id": 3, "balance": 1042.50, "currency": "USD"}` (every field name is repeated).
4. Close the connection (slow).

Now do that 10,000 times a second. That's a lot of wasted bytes and a lot of wasted handshakes.

gRPC says: "Stop doing that. Open one connection, keep it open forever, and send tiny binary messages back and forth that both sides already know how to decode."

That is the whole pitch. gRPC is **smaller messages, faster transport, persistent connections, with a schema both sides agreed on in advance.**

### A picture of the whole thing

```
            ┌──────────────────────────────────────────────┐
            │           the .proto file (the schema)        │
            │  service Bank {                               │
            │    rpc GetBalance(BalanceRequest)             │
            │      returns (BalanceResponse);               │
            │  }                                            │
            └──────────────────────────────────────────────┘
                              │
                              │  protoc / buf generate
                              ▼
       ┌──────────────────────────┐    ┌──────────────────────────┐
       │  generated Go client     │    │  generated Python server │
       │  client.GetBalance(...)  │    │  class BankServicer:     │
       └──────────────────────────┘    │    def GetBalance(...)   │
                  │                    └──────────────────────────┘
                  │                                ▲
                  │  binary protobuf bytes         │
                  │  over HTTP/2 stream            │
                  ▼                                │
       ════════════════════════════════════════════
                       network
       ════════════════════════════════════════════
```

The thing in the middle — the `.proto` file — is the **shared shorthand**. The arrows are the phone call. The boxes on each side are the parts that get **generated** for you so you don't have to write them by hand.

### The four magic words

Every gRPC explanation eventually boils down to these four words:

1. **Schema-first.** You write the contract (the `.proto` file) before you write any code. Both sides read the same contract.
2. **Code generation.** A tool reads the contract and generates client code in your language and server code in your language. You don't write the wire format by hand.
3. **Binary.** The actual bytes that travel over the wire are tiny binary numbers, not human-readable text.
4. **Streaming.** Because the connection stays open (HTTP/2), you can send a stream of messages, not just one-shot request/response.

If you remember those four words, you remember gRPC.

## vs REST + JSON

Let's put gRPC and REST+JSON side by side. They both do the same job: let one program ask another program to do something. They are different in *how* they do it.

### Binary vs text

REST+JSON sends text. Look at this:

```
{"user_id": 12345, "name": "Alice", "active": true}
```

That is 51 bytes. The field names `user_id`, `name`, and `active` are written out every single time, even though both sides know the message has a user_id, name, and active field — they could just agree on the order and skip the names.

gRPC sends binary. The same message in protobuf wire format might be:

```
08 b9 60 12 05 41 6c 69 63 65 18 01
```

That's 12 bytes. The names are gone. There are little number tags (`08`, `12`, `18`) that say "this is field 1, this is field 2, this is field 3" and the values come right after. Both sides know from the schema that field 1 is `user_id`, field 2 is `name`, field 3 is `active`. So nobody has to spell it out.

**Result:** gRPC messages are typically 3x to 10x smaller than the equivalent JSON. On a busy server doing millions of requests, that adds up to real money.

You cannot read protobuf bytes with your eyes. That is sometimes annoying when debugging — but tools like `grpcurl` and `evans` will decode them for you using the schema.

### Schema-first vs ad-hoc

In REST+JSON, the schema is usually written in English in some doc somewhere. You read the doc. You hope the doc is up to date. You write code that *should* match the doc. The server writes code that *should* match the doc. If the doc is wrong or stale, you find out when something breaks in production.

There are tools to fix this in the REST world (OpenAPI / Swagger), but they are bolted on after the fact. They are optional. Most REST APIs you'll see in your career are documented in Markdown that disagrees with the actual server.

In gRPC, the schema is the source of truth. It's a `.proto` file. Both the client and the server are *generated from it*. If the proto says a field is an `int32`, both sides see an `int32`. If the proto says a method takes a `BalanceRequest` and returns a `BalanceResponse`, you literally cannot call it with the wrong type — your code won't compile.

This is the single biggest day-to-day win of gRPC. The contract is enforced by the compiler.

### HTTP/2 streaming vs HTTP/1 short-lived

HTTP/1 was designed in 1991. It thinks of the web as: "open a connection, ask for one thing, get one thing back, close the connection." Even with HTTP/1.1's "keep-alive," you still send one request at a time per connection — if you want to send four things in parallel you need four connections.

HTTP/2 (2015) said: "What if one connection could carry many independent message streams at once, each one its own conversation?" That's called **multiplexing.** A single HTTP/2 connection can be carrying 100 simultaneous streams, each of which is a separate gRPC call.

gRPC was built on top of HTTP/2 from day one. It uses HTTP/2 streams as its transport. That means:

- One TCP connection covers many calls.
- Calls don't block each other (no "head-of-line blocking" the way HTTP/1 has).
- Long-running calls can hold a stream open and dribble messages back and forth — that's where the four RPC patterns come from.

### A side-by-side cheat sheet

```
                     REST + JSON           gRPC
─────────────────────────────────────────────────────────
schema             :  optional doc      :  required .proto
wire format        :  human text        :  binary protobuf
transport          :  HTTP/1 (usually)  :  HTTP/2 (always)
streaming          :  no (or hacks)     :  yes (4 patterns)
codegen            :  optional          :  required
debugging          :  curl + eyes       :  grpcurl + schema
size on the wire   :  big               :  3x-10x smaller
versioning         :  free-for-all      :  field numbers
browser support    :  native            :  needs gRPC-Web
```

Neither is "better" in every situation. REST+JSON is friendlier for browsers, public APIs, and anything where humans might poke at the messages by hand. gRPC is faster, smaller, and more strict — perfect for *internal* services talking to each other inside a company.

A common pattern: gRPC inside the data center, REST+JSON at the edge facing the public internet, and a thing called **gRPC-Gateway** that auto-translates between them. We'll get to that.

## Protocol Buffers

Protocol Buffers (protobuf for short) is the schema language. It is its own little programming language whose only job is to describe data. You will spend a real amount of time looking at `.proto` files if you use gRPC.

Crucial point that confuses people: **protobuf and gRPC are not the same thing.** Protobuf is the schema and the binary format. gRPC is the RPC framework that uses protobuf as its schema and binary format. You can use protobuf without gRPC (lots of people do — it's just a fast serialization format). You almost never use gRPC without protobuf.

### A whole `.proto` file from top to bottom

Here is a complete file. We will walk through every line.

```proto
syntax = "proto3";

package bank.v1;

option go_package = "example.com/bank/v1;bankv1";

import "google/protobuf/timestamp.proto";

service BankService {
  rpc GetAccount(GetAccountRequest) returns (Account);
  rpc ListAccounts(ListAccountsRequest) returns (stream Account);
  rpc Deposit(stream DepositChunk) returns (DepositSummary);
  rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

message GetAccountRequest {
  string account_id = 1;
}

message Account {
  string account_id = 1;
  string owner_name = 2;
  int64 balance_cents = 3;
  Currency currency = 4;
  repeated string tags = 5;
  optional string nickname = 6;
  google.protobuf.Timestamp created_at = 7;
  oneof contact {
    string email = 8;
    string phone = 9;
  }
  map<string, string> metadata = 10;
  reserved 11, 12;
  reserved "old_field";
}

enum Currency {
  CURRENCY_UNSPECIFIED = 0;
  CURRENCY_USD = 1;
  CURRENCY_EUR = 2;
  CURRENCY_GBP = 3;
}

message ListAccountsRequest {
  int32 page_size = 1;
  string page_token = 2;
}

message DepositChunk {
  bytes payload = 1;
}

message DepositSummary {
  int64 total_cents = 1;
  int32 chunk_count = 2;
}

message ChatMessage {
  string from = 1;
  string text = 2;
}
```

### `syntax = "proto3";`

The very first line. It tells protoc which version of the protobuf language this file uses. There are two living versions:

- **proto2** — the original. Has `required` and `optional` keywords. Defaults are explicit. Still used in older codebases.
- **proto3** — the modern one (released 2016). Removed `required`. Made everything implicitly "optional." Cleaner syntax. **What you should use unless you have a strong reason not to.**

A historical wrinkle: proto3 originally removed the `optional` keyword too, which turned out to be a mistake (you couldn't tell "field is missing" from "field is set to zero"). It was **re-added in protoc 3.15 (2021)** as `optional` with proper field-presence tracking. So when you see `optional` on a proto3 field today, that's why.

### `package bank.v1;`

A namespace. Two protos can have the same `service` name as long as they're in different packages. By convention, packages include a version (`bank.v1`) so you can later have `bank.v2` without colliding.

### `option go_package = ...`

Per-language options. This one tells the Go code generator where the generated package will live. There are similar options for Java (`java_package`), C# (`csharp_namespace`), and so on. They don't affect the wire format — they only affect generated code paths.

### `import "google/protobuf/timestamp.proto";`

You can pull in other proto files. The `google/protobuf/*.proto` files ship with protoc and are called the **well-known types**: `Timestamp`, `Duration`, `Empty`, `Any`, `Struct`, `Value`, `ListValue`, `NullValue`, `FieldMask`, plus wrapper types (`StringValue`, `Int32Value`, `BoolValue`, etc.) for nullable scalars. Use them whenever they fit — every gRPC ecosystem already understands them.

### `service BankService { ... }`

Defines the actual RPC contract. Each `rpc` line inside is one method that the server implements and the client can call. A service is the *behavior*. A message is the *data*.

### `rpc GetAccount(GetAccountRequest) returns (Account);`

One RPC. Reads as: "There is a method called `GetAccount` that takes a `GetAccountRequest` and returns an `Account`." This is **unary** — one request, one response. We'll see the other three patterns in a second.

The `stream` keyword on either side turns it into a streaming RPC:

- `returns (stream Account)` — server-streaming.
- `(stream DepositChunk)` — client-streaming.
- `(stream …) returns (stream …)` — bidirectional streaming.

### `message Account { ... }`

A struct. A bag of named, numbered fields. The whole point of a message.

### Field rules

Inside a message, every field has:

- **a type** — a scalar like `int32` or `string`, another message, an enum, a `repeated`, an `optional`, a `oneof`, or a `map`.
- **a name** — what your code will call it. Snake_case by convention.
- **a tag number** — the number after `=`. This is what actually goes on the wire. You change the name freely; you must **never** change the tag.

```proto
string account_id = 1;
//                ^---- field tag, NOT a default value
```

Field tags 1–15 use one byte on the wire; 16–2047 use two bytes. Put your hottest fields in the low numbers.

### Scalar types

The list of built-in scalars:

```
double            64-bit float
float             32-bit float
int32 / int64     variable-length signed (cheap for small values, ugly for negatives)
uint32 / uint64   variable-length unsigned
sint32 / sint64   variable-length signed (zig-zag — better for negatives)
fixed32 / fixed64 always 4 / 8 bytes (cheap for big values)
sfixed32/sfixed64 always 4 / 8 bytes, signed
bool              1 byte on the wire
string            UTF-8 text
bytes             raw bytes
```

Don't memorize this. Just remember:

- "I have a number that's usually small and positive": `int32` or `int64`.
- "I have a number that might be very negative": `sint32`.
- "I have a number that's almost always huge": `fixed64`.
- "I have a string of text": `string`.
- "I have raw bytes (image, audio, blob)": `bytes`.

### `repeated`

`repeated string tags = 5;` — this field is a list of strings. There can be 0, 1, 2, or a million tags.

### `optional`

`optional string nickname = 6;` — proto3-modern field with **explicit presence**. The generated code lets you ask "did the sender actually set this field, or is it just defaulting to the empty string?" Without `optional`, an empty string and a not-sent field look identical in proto3.

### `oneof`

```proto
oneof contact {
  string email = 8;
  string phone = 9;
}
```

"Exactly one of these is set, and I want the language to enforce that." Setting `email` automatically clears `phone` and vice versa. Each member of the oneof still gets its own field tag.

### `map<K, V>`

`map<string, string> metadata = 10;` — a key/value dictionary. Keys must be a scalar (no messages as keys). Under the hood it's just sugar for `repeated MetadataEntry`.

### `reserved`

```proto
reserved 11, 12;
reserved "old_field";
```

"Don't reuse these field numbers or names — we used to have something there and we've removed it, and reusing them would corrupt anybody who still has the old schema." Always reserve when you delete a field. Always.

### `enum`

```proto
enum Currency {
  CURRENCY_UNSPECIFIED = 0;
  CURRENCY_USD = 1;
  CURRENCY_EUR = 2;
}
```

Rule: the **first value must be 0**, and convention says it should be a `*_UNSPECIFIED` sentinel. This is because in proto3 the default value of an enum field is whatever has tag 0, and you want that to mean "the sender didn't set it."

You can opt-in to allowing two enum values to share the same tag by adding `option allow_alias = true;` inside the enum.

### `deprecated`

```proto
string old_field = 4 [deprecated = true];
```

A hint to the codegen. Generated code may emit a deprecation warning when you use it. The wire format is unchanged.

### Versioning rule of thumb

Adding fields: free, do whenever you want.
Removing fields: replace with `reserved <tag>; reserved "name";` — never reuse the tag.
Renaming fields: free in proto3, the wire format only cares about tags. (But your generated code will change names, which breaks consumers — so be careful at the *code* level.)
Changing field types: usually breaking. There are a few wire-compatible swaps (e.g., `int32` ↔ `int64` ↔ `uint32` ↔ `uint64` ↔ `bool`), but don't memorize them — just don't change types.

## The Four RPC Patterns

This is the part where streaming earns its keep. There are exactly four shapes a gRPC method can take. Every gRPC method you ever see is one of these four.

### 1. Unary — one request, one response

```
client                                     server
  │                                          │
  │ ──── GetAccount(req) ─────────────────►  │
  │                                          │ (handler runs)
  │  ◄──────────────────── Account(resp) ──  │
  │                                          │
```

Just like a function call. This is what 80% of methods look like. Use it unless you have a real reason to stream.

```proto
rpc GetAccount(GetAccountRequest) returns (Account);
```

### 2. Server-streaming — one request, many responses

```
client                                     server
  │                                          │
  │ ──── ListAccounts(req) ────────────────► │
  │                                          │
  │  ◄────────────────────── Account #1 ───  │
  │  ◄────────────────────── Account #2 ───  │
  │  ◄────────────────────── Account #3 ───  │
  │  ◄────────────────────── Account #4 ───  │
  │  ◄──────────────────────── EOF ────────  │
  │                                          │
```

Client sends one request, server sends a stream of responses, then closes. Good for: large result sets, server-pushed updates, log tails, search results that come in waves.

```proto
rpc ListAccounts(ListAccountsRequest) returns (stream Account);
```

### 3. Client-streaming — many requests, one response

```
client                                     server
  │                                          │
  │ ──── DepositChunk #1 ──────────────────► │
  │ ──── DepositChunk #2 ──────────────────► │
  │ ──── DepositChunk #3 ──────────────────► │
  │ ──── DepositChunk #4 ──────────────────► │
  │ ──── EOF ──────────────────────────────► │
  │                                          │ (handler runs)
  │  ◄──────────────────── DepositSummary ─  │
  │                                          │
```

Client streams chunks, then closes. Server sees them all and sends one final response. Good for: large file uploads, bulk inserts, sensor data dumps.

**Gotcha:** the server **does not see anything** until the client calls `Recv` (or the language's equivalent). If you write a server-side handler that loops on `Recv` to read messages, you're fine. If you forget to loop, you'll never see the messages. People hit this on day one.

```proto
rpc Deposit(stream DepositChunk) returns (DepositSummary);
```

### 4. Bidi-streaming — many requests, many responses, in any order

```
client                                     server
  │                                          │
  │ ──── ChatMessage "hi" ────────────────►  │
  │  ◄──────────────────── ChatMessage "yo"  │
  │ ──── ChatMessage "what's up" ─────────►  │
  │  ◄──────────────────── ChatMessage "nm"  │
  │  ◄──────────────────── ChatMessage "u?"  │
  │ ──── ChatMessage "good" ──────────────►  │
  │ ──── EOF ─────────────────────────────►  │
  │  ◄────────────────────────── EOF ──────  │
  │                                          │
```

Both sides can send and receive at the same time, in any order. Use case: chat, real-time games, collaborative editors, multiplexed RPC over a single call.

```proto
rpc Chat(stream ChatMessage) returns (stream ChatMessage);
```

Bidi is the most flexible and the most error-prone. Get unary working first.

## Code Generation

Schema-first means **we generate code from the schema**. You will not hand-write the network bytes. You will not hand-write the message structs. Tools do that for you. You write the `.proto` and the handler.

### `protoc` — the original

`protoc` is the original protobuf compiler. It reads `.proto` files and emits language-specific code. It's a single binary, written in C++, distributed by the protobuf project at https://protobuf.dev.

`protoc` itself only knows about protobuf messages. To make it also generate gRPC service stubs, you point it at a **plugin** for your language. Plugins are separate executables named `protoc-gen-<thing>`. When you run `protoc --foo_out=...`, protoc looks for an executable on `PATH` called `protoc-gen-foo` and pipes the parsed proto descriptors into it.

```
.proto file ──► protoc parses ──► descriptor ──► plugin emits code
                                                   │
                          protoc-gen-go ───────────┤  // Go message types
                          protoc-gen-go-grpc ──────┤  // Go gRPC stubs
                          protoc-gen-grpc-web ─────┤  // browser stubs
                          protoc-gen-validate ─────┤  // validators
                          protoc-gen-openapiv2 ────┘  // Swagger doc
```

Common plugins per language:

- **Go**: `protoc-gen-go` (messages) + `protoc-gen-go-grpc` (services). Two separate plugins. Both are required.
- **Python**: `grpc_tools.protoc` is a single bundled tool that wraps protoc + the Python plugins; install via `pip install grpcio-tools`.
- **Java**: `protoc-gen-grpc-java` plus protoc's built-in Java emit; usually invoked through Gradle/Maven plugins, not by hand.
- **C++**: protoc has a built-in C++ generator; gRPC stubs come from `grpc_cpp_plugin`.
- **Rust**: most people use `tonic-build` (a build-script crate) rather than calling protoc directly.
- **JS/TS**: `protoc-gen-js` (deprecated) or, much more commonly today, `@bufbuild/protoc-gen-es` and `@bufbuild/protoc-gen-connect-es` from the Connect ecosystem.
- **Ruby**: `grpc_tools_ruby_protoc`.
- **Swift / Kotlin / Dart**: each has its own dedicated plugin (`protoc-gen-swift`, `protoc-gen-kotlin`, `protoc-gen-dart`) plus a gRPC plugin.

### `buf` — the modern way

Driving `protoc` directly is fiddly. You end up with long shell commands, conflicting plugin versions, and reproducibility problems on different machines. **Buf** (https://buf.build) is a modern toolchain that wraps the whole pipeline.

Buf gives you:

- A single `buf.yaml` file to declare your proto module, dependencies, and lint rules.
- A `buf.gen.yaml` file to declare what plugins to run and where to put output.
- Built-in **linting** with curated rule sets (style, naming).
- Built-in **breaking-change detection** — `buf breaking --against` flags removed fields, renamed services, changed types.
- **Buf Schema Registry (BSR)** at `buf.build` — like npm/Docker Hub but for protos. You publish your module, you depend on others, no more vendoring.
- **Remote plugins** — you don't even need protoc-gen-* installed locally; Buf can run them on the BSR and stream the generated code back.

For new projects, start with `buf` and never look back. The two-line summary:

```bash
$ buf generate              # generate code from your .proto files
$ buf lint                  # check style
$ buf breaking --against '.git#branch=main'   # check for breaking changes
```

### Diagram: where the bytes go

```
                .proto file
                    │
                    │ protoc / buf generate
                    │
        ┌───────────┴───────────┐
        │                       │
  client stub               server skeleton
  (your code calls it)      (you implement handlers)
        │                       │
        │  marshal request      │  unmarshal request
        │  (binary protobuf)    │
        ▼                       ▼
  ┌──────────────────────────────────┐
  │    HTTP/2 stream (one of many    │
  │    multiplexed on the same       │
  │    persistent TCP connection)    │
  └──────────────────────────────────┘
        ▲                       ▲
        │  unmarshal response   │  marshal response
        │                       │
   client receives          server returns
   typed response           typed response
```

## Channels and Stubs

A **channel** (sometimes called a "client connection") is the long-lived object that represents the conversation between your client and one or more servers. It owns the TCP connection (or pool of them), does name resolution, picks a load-balancing strategy, and applies retry logic.

A **stub** (sometimes called a "client") is the typed wrapper around the channel that has one method per RPC. The stub is what your code actually calls.

```
Channel ─┬─ TCP connection #1 ──► server A
         ├─ TCP connection #2 ──► server B
         └─ TCP connection #3 ──► server C
                  ▲
                  │
   Stub ──── client.GetAccount(req)  ────► picks a connection, sends an HTTP/2 stream
```

In Go (the canonical example):

```go
// 1. Create the channel
conn, err := grpc.NewClient("dns:///bank.example.com:50051",
    grpc.WithTransportCredentials(insecure.NewCredentials()))
if err != nil {
    log.Fatal(err)
}
defer conn.Close()

// 2. Create the stub
client := bankv1.NewBankServiceClient(conn)

// 3. Call an RPC
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := client.GetAccount(ctx, &bankv1.GetAccountRequest{AccountId: "abc"})
```

A note on `NewClient` vs the older `Dial`: gRPC-Go 1.65 introduced `grpc.NewClient` as the modern replacement. The big behavioral change is that `NewClient` is **lazy** — it does not actually open a connection until the first RPC. Older `Dial` and `DialContext` used to open eagerly, which led to subtle bugs around startup ordering. If you're learning today, use `NewClient`.

### Blocking vs non-blocking

In Java and Python the generated stubs come in flavors:

- **Blocking stub** — call returns when the response is ready (or times out). Easiest to reason about. Best for unary calls in simple programs.
- **Async stub / future stub** — call returns immediately with a future you can poll or attach a callback to. Useful when you want to fire many RPCs in parallel.
- **Streaming stub** — for the streaming patterns. Always non-blocking by nature.

In Go there is only one flavor — every call takes a `context.Context`, and you compose async behavior with goroutines.

In Python, modern code uses `grpc.aio` for `async def` style.

## Interceptors

An interceptor is a function that wraps every RPC. It runs before the handler runs, and again after the response is ready. Same idea as middleware in a web framework.

There are two kinds:

- **Server-side interceptors** — run on the server. Used for: auth, logging, metrics, tracing, rate-limiting, panic recovery.
- **Client-side interceptors** — run on the client. Used for: retry, deadline injection, circuit breaking, request signing, header propagation.

```
[ client ] ──► client interceptor ──► client interceptor ──► wire
                       │                       │
                       └─ retry policy         └─ trace headers

  wire ──► server interceptor ──► server interceptor ──► [ handler ]
                  │                       │
                  └─ auth check           └─ access log
```

Interceptors are stacked. They run in the order you register them on the way in, and the reverse order on the way out (like middleware everywhere else).

For unary RPCs you write a function that gets `(ctx, req, info, handler)` and returns `(resp, err)`. For streams you wrap the stream object so you can intercept each `Send`/`Recv`.

A common production stack on the server might be:

```
Recover-from-panic
  → Request-ID injector
    → Authentication (verify JWT, attach user to context)
      → Authorization (check permissions on the method)
        → Rate limiter
          → OpenTelemetry tracing
            → Prometheus metrics
              → Access log
                → ACTUAL HANDLER
```

And on the client:

```
Retry-with-backoff
  → Circuit breaker
    → Deadline enforcer
      → Add Authorization: Bearer <token>
        → Add X-Request-Id
          → OpenTelemetry tracing
            → ACTUAL CALL
```

Interceptors are how everything operational gets bolted onto gRPC. You almost never put auth or logging directly in a handler.

## Deadlines and Cancellation

This is the section nobody reads carefully, and then everybody gets bitten.

### Deadlines, not timeouts

A **timeout** says "give up if this hasn't finished in 5 seconds." It's local to whatever piece of code set it.

A **deadline** says "everyone working on this request, stop at 12:34:56.789 UTC." It's a wall-clock instant, and it propagates all the way down the call chain.

gRPC uses **deadlines.** When a client makes a call with a 5-second timeout, gRPC sets the deadline on the wire (via the `grpc-timeout` header) so the server knows exactly when to give up. If the server then calls another gRPC service to finish the work, the inherited deadline goes with it. The downstream service knows it has *less than 5 seconds* — whatever's left.

This is why: if Service A calls B (5s deadline), and B calls C, then C should not be allowed to spend 4 seconds when A only has 1 second of budget left. Deadlines fix that.

```
   Client                Server A              Server B              Server C
     │                      │                     │                     │
     │ deadline=12:00:05    │                     │                     │
     │─────────────────────►│                     │                     │
     │  call(ctx)           │ deadline propagates │                     │
     │                      │────────────────────►│ deadline=12:00:05   │
     │                      │  call(ctx)          │                     │
     │                      │                     │────────────────────►│ deadline=12:00:05
     │                      │                     │  call(ctx)          │
     │                      │                     │                     │  ── still has the
     │                      │                     │                     │     same deadline
```

When the deadline expires, every in-flight RPC up and down the chain returns `DeadlineExceeded`. The work stops. Resources don't leak.

### Cancellation

Cancellation is the same idea but voluntary: the client (or a parent) decides to call it off early. In Go you cancel a context. In other languages you call `cancel()` on the call object. The cancellation also propagates over the wire, and downstream servers learn about it via the stream's cancellation signal.

The big practical rules:

1. **Always set a deadline on every client call.** Default deadline is "infinity," which is a terrible default. You will eventually have a server hang and you'll be glad you set 5 seconds instead of waiting forever.
2. **Always check `ctx.Done()` (or your language's equivalent) inside long handlers.** If the deadline passes or the client cancels, you want to bail out, not finish a now-pointless calculation.
3. **Never override a deadline downstream with a longer one.** Doing so breaks the propagation guarantee.

## Metadata

Metadata is the gRPC name for headers. It's a multi-map of string keys to string-or-bytes values that travels alongside the request and response. Metadata is **not** part of your protobuf message — it is on the HTTP/2 frame, like an HTTP header.

Use cases:

- Authentication: `authorization: Bearer eyJhbGc...`
- Tracing: `traceparent: 00-...`
- Tenant or request IDs: `x-request-id: 9f3...`
- Anything else where you want side-channel info that isn't part of the data model.

Two kinds of metadata:

1. **Headers** — sent at the start of the call. The server sees them before the handler runs.
2. **Trailers** — sent at the *end* of a streaming response, after the last data message. They contain the gRPC status code, the human-readable message, and any extra trailing metadata the server wants to send.

Trailers are a big deal: they're the reason gRPC can stream a response and *still* report a final status. A REST+JSON API can't easily do this — once it has started streaming the response body, the status code is already locked in. gRPC reserves the trailers slot precisely so the server can stream forever and *then* say "OK, I finished, status code 0."

```
           HEADERS (start of call)
             │
             │   :path = /bank.v1.BankService/ListAccounts
             │   content-type = application/grpc
             │   authorization = Bearer ...
             ▼
           DATA #1 (Account proto)
           DATA #2 (Account proto)
           DATA #3 (Account proto)
             │
             ▼
           TRAILERS (end of call)
             │
             │   grpc-status = 0
             │   grpc-message = ""
             │   x-server-region = us-east
             ▼
```

Metadata key conventions:

- All keys are lowercase ASCII.
- Keys ending in `-bin` carry binary values base64-encoded. Used for things like `grpc-status-details-bin`.
- Reserved keys: don't use anything starting with `grpc-` for your own metadata; the framework owns those.

## Authentication

gRPC has two layers of credentials:

- **Channel credentials** — apply to the whole connection. TLS server certs, mTLS client certs, ALTS.
- **Call credentials** — apply per-RPC. OAuth tokens, JWTs, API keys typically sent via the `authorization` header.

You combine them.

### TLS — the baseline

Every production gRPC connection should run over TLS. You take the server's PEM certificate, point the client at the right hostname, and you're done. The server's cert chains up to a CA your client trusts.

```bash
$ grpcurl bank.example.com:443 list
```

(No `-plaintext` flag means: use TLS, verify the cert against the system trust store.)

### mTLS — both sides prove who they are

Mutual TLS is when the **client** also presents a cert and the server verifies it. Standard inside a service mesh: both pods have certs issued by an internal CA, and the only people who can talk to your service are people the CA trusts. This is the foundation of zero-trust networking.

### OAuth tokens / JWTs / API keys

These ride in metadata, by convention on the `authorization` header. Same convention as the rest of HTTP.

```
authorization: Bearer eyJhbGciOiJSUzI1NiIs...
```

In a client interceptor, you fetch a fresh token (e.g., from your identity provider) and inject it on every call. In a server interceptor, you parse and verify it, then attach the resulting "user" to the context so handlers can read it.

### ALTS — Google's hop-by-hop scheme

ALTS ("Application Layer Transport Security") is Google's homemade alternative to TLS, designed for service-to-service traffic inside their data centers. Faster handshake (no per-call cert chain to walk), built-in service identity. **You can only use it inside Google infrastructure** (GKE, GCE service identity, etc.). Outside of Google's networks it doesn't really exist. Don't go looking for ALTS to run between two of your laptops; it isn't for that.

## Reflection

Server reflection is an optional gRPC feature where the server publishes its own schema. Once enabled, a client without a `.proto` file can ask the server "what services do you have, what methods, and what are the message shapes?"

That sounds fancy. The practical use is: **debugging tools work**. `grpcurl`, `grpcui`, and `evans` can all talk to a reflection-enabled server with no extra setup. You just point them at a port and start exploring.

To enable it on a Go server in two lines:

```go
import "google.golang.org/grpc/reflection"
reflection.Register(server)
```

There are two protocol versions: **v1** and **v1alpha**. Most servers register both for compatibility with old tools.

You almost always want reflection on in **dev and staging**. In **production**, opinions vary: some teams leave it on (debugging is great), others turn it off (don't reveal the schema to randos). A common compromise is to turn it on but require auth.

```
        ┌─────────────┐
        │ grpcurl     │ ─── ListServices ──►  ┌────────────────┐
        │             │                        │ gRPC server    │
        │             │ ─── DescribeMethod ──► │ (reflection on)│
        │             │ ◄── descriptors ────── │                │
        └─────────────┘                        └────────────────┘
```

## gRPC-Web

Browsers can't speak HTTP/2 the way gRPC needs. Specifically, browsers' fetch API doesn't expose enough of HTTP/2 trailers to make raw gRPC work. So **gRPC-Web** was invented: a slightly modified wire format that uses regular HTTP requests (HTTP/1.1 or HTTP/2) and squashes the trailers into the body.

The architecture:

```
┌──────────────┐   gRPC-Web (HTTP)  ┌─────────────┐  gRPC (HTTP/2)  ┌──────────┐
│  browser JS  │────────────────────►│  proxy      │─────────────────►│  server  │
│              │                     │  (Envoy /   │                  │          │
│              │◄────────────────────│  grpc-web   │◄─────────────────│          │
└──────────────┘                     │  proxy)     │                  └──────────┘
                                     └─────────────┘
```

You run a small proxy (Envoy with the gRPC-Web filter, or a standalone `grpcwebproxy`) in front of your gRPC server. The browser speaks gRPC-Web to the proxy, the proxy speaks regular gRPC to the server. The browser library is generated by `protoc-gen-grpc-web`.

gRPC-Web has a key limitation: it does **not** support client-streaming or bidi-streaming, only unary and server-streaming. The browser stack just can't do it.

If you need full bidi in a browser, look at WebSocket-based protocols, or use **Connect-Web** (next section), which has its own trade-offs.

## gRPC-Gateway

gRPC-Gateway is the inverse: it puts a REST+JSON face on a gRPC server. You annotate your proto methods with HTTP routes, run `protoc-gen-grpc-gateway`, and you get a generated reverse proxy that translates HTTP+JSON requests to gRPC calls.

```proto
import "google/api/annotations.proto";

service BankService {
  rpc GetAccount(GetAccountRequest) returns (Account) {
    option (google.api.http) = {
      get: "/v1/accounts/{account_id}"
    };
  }
}
```

Now the same method is reachable both as gRPC at `/bank.v1.BankService/GetAccount` and as REST at `GET /v1/accounts/abc`.

This is incredibly useful. It lets you build the actual service in gRPC (typed, fast, schema-first) and still expose a REST+JSON facade for browsers, partner integrations, or anybody who isn't ready to adopt gRPC. Many large APIs (including many of Google's public APIs) are built this way.

There is also a sibling tool, `protoc-gen-openapiv2`, which generates an OpenAPI/Swagger doc from the same annotations so your REST surface gets a Swagger UI for free.

## Connect / Connect-Web

Connect (https://connectrpc.com), built by the Buf team and launched in 2022, is a newer protocol that's **wire-compatible with gRPC for unary RPCs** but uses a simpler, browser-friendly framing.

What Connect adds:

- A single Connect server can speak gRPC, gRPC-Web, **and** the Connect protocol, all on the same port. The client picks.
- The Connect protocol is regular HTTP+JSON or HTTP+binary-protobuf — easy to debug with curl.
- No proxy needed for browser support; the Connect-Web client speaks Connect natively.
- Smaller, simpler client libraries than gRPC's official ones.

What Connect doesn't change: the schema is still protobuf (`.proto` files). You still get codegen. You still get a typed client. The wire framing is the only difference.

For new projects, especially ones with a browser frontend, Connect is increasingly the default. For projects already on gRPC, you can stand up a Connect-compatible server and gradually migrate clients, because gRPC clients still work against a Connect server (for unary).

```
                  ┌──────────────────────────┐
                  │  Connect server           │
                  │  (e.g. connect-go)        │
                  │                           │
                  │ ┌─ understands gRPC       │
                  │ ├─ understands gRPC-Web   │
                  │ └─ understands Connect    │
                  └──────────────────────────┘
                          ▲    ▲    ▲
                          │    │    │
                  ┌───────┘    │    └────────┐
                  │            │             │
            gRPC client    gRPC-Web      Connect-Web
            (Go/Java)      browser       browser
                                         (curl-able!)
```

## Common Errors

When something goes wrong, gRPC returns a status code and a message. The error format your client sees is:

```
rpc error: code = <CodeName> desc = <message>
```

Here are the ones you'll see in the wild, with what they mean and how to fix them.

```
rpc error: code = Unavailable desc = connection error: desc = "transport: Error while dialing: dial tcp: lookup bank.example.com on 8.8.8.8:53: no such host"
```
DNS lookup failed. The hostname doesn't resolve. Check spelling, check your DNS server, check that the service is actually deployed.

```
rpc error: code = Unavailable desc = last connection error: connection refused
```
TCP connection refused. Server is not listening on that port, or a firewall blocked it. Try `nc -vz host port` to confirm.

```
rpc error: code = DeadlineExceeded desc = context deadline exceeded
```
You set a deadline, the call took too long. Either the server is slow, or your deadline was too tight. **Always set a deadline.** If you see this, raise the deadline OR fix the slow handler.

```
rpc error: code = Unauthenticated desc = invalid auth token
```
Server rejected your credentials. Token is missing, expired, signed by the wrong issuer, or has the wrong audience.

```
rpc error: code = PermissionDenied desc = caller does not have permission
```
Server knows who you are, but you're not allowed to do this. Different from `Unauthenticated`.

```
rpc error: code = NotFound desc = account "abc" not found
```
The thing you asked for doesn't exist. Application-level — server chose to return this code.

```
rpc error: code = ResourceExhausted desc = quota exceeded
```
You hit a rate limit or a quota. Slow down, retry with backoff, or get a bigger quota.

```
rpc error: code = Internal desc = stream terminated by RST_STREAM with error code: INTERNAL_ERROR
```
The HTTP/2 stream got reset abruptly. Often a bug or a panic on the server. Check server logs.

```
rpc error: code = Canceled desc = context canceled
```
Either you (the client) cancelled, or somewhere up the call chain a parent context was cancelled. Not always an error — sometimes it's the user closing the page.

```
rpc error: code = Unimplemented desc = unknown method GetAccount for service bank.v1.BankService
```
The server doesn't know that method. Schema mismatch — usually because the client and server were built from different `.proto` versions, or you forgot to register the service on the server.

```
rpc error: code = Unimplemented desc = unknown service bank.v1.BankService
```
Whole service is missing. Usually you forgot to call `RegisterBankServiceServer(grpcServer, &myImpl{})` on the server.

```
rpc error: code = Unavailable desc = transport: error while dialing: tls: failed to verify certificate: x509: certificate signed by unknown authority
```
TLS chain validation failed. Either the server's cert is signed by a CA your client doesn't trust (use `--cacert` or install the CA), or the cert is for a different hostname than you connected to.

```
rpc error: code = Internal desc = grpc: received message larger than max (4194305 vs. 4194304)
```
Default max message size is 4 MiB. Either bump it (`grpc.MaxRecvMsgSize` on the server) or chunk your data using a client-streaming RPC.

```
rpc error: code = FailedPrecondition desc = account is frozen
```
The state of the system means this operation can't run right now. Application-level.

```
rpc error: code = Aborted desc = transaction conflict
```
Concurrency conflict (think CAS or optimistic locking). Often retryable.

The full status code map:

```
0   OK                  : success
1   Cancelled           : caller cancelled
2   Unknown             : not categorised
3   InvalidArgument     : caller sent garbage
4   DeadlineExceeded    : ran out of time
5   NotFound            : doesn't exist
6   AlreadyExists       : already exists
7   PermissionDenied    : known caller, not allowed
8   ResourceExhausted   : rate / quota
9   FailedPrecondition  : wrong state
10  Aborted             : concurrency conflict
11  OutOfRange          : iter past end / index too big
12  Unimplemented       : method or service missing
13  Internal            : server bug
14  Unavailable         : transient — retry
15  DataLoss            : irrecoverable loss
16  Unauthenticated     : bad/missing creds
```

`Unavailable` and `ResourceExhausted` and `Aborted` are usually retryable; almost everything else isn't. The standard gRPC retry policy lets you list the codes you'll retry on.

## Hands-On

Every command below is paste-ready. The output blocks show **roughly** what you'll see — version numbers and IPs will differ on your machine.

### Check your tooling

```bash
$ protoc --version
libprotoc 25.1
```

```bash
$ buf --version
1.28.1
```

```bash
$ grpcurl --version
grpcurl v1.8.9
```

```bash
$ ghz --version
ghz: v0.117.0
```

```bash
$ evans --version
evans 0.10.11
```

### Lint and format your protos

```bash
$ buf lint
proto/bank/v1/bank.proto:14:3:RPC request type "GetAccountReq" should be named "GetAccountRequest" or "BankServiceGetAccountRequest".
```

```bash
$ buf format -d
--- proto/bank/v1/bank.proto
+++ proto/bank/v1/bank.proto (formatted)
@@ -10,5 +10,4 @@
 message Account {
-  string  account_id = 1;
-  string owner_name=2;
+  string account_id = 1;
+  string owner_name = 2;
 }
```

### Detect breaking changes

```bash
$ buf breaking --against '.git#branch=main'
proto/bank/v1/bank.proto:24:1:Field "5" with name "tags" on message "Account" changed type from "string" to "int32".
```

That field-type change is a wire-incompatible break — buf catches it before it hits production.

### Push your module to the BSR

```bash
$ buf push buf.build/me/myproto
75c1f7a3a7ff4a52b5a2e24b8e5d7a01
```

### Generate code

```bash
$ buf generate
$ ls gen/
go/  python/  ts/
```

### Drive protoc by hand — Go

```bash
$ protoc \
    --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/bank/v1/bank.proto
$ ls proto/bank/v1/
bank.pb.go  bank.proto  bank_grpc.pb.go
```

### Drive protoc by hand — Python

```bash
$ python -m grpc_tools.protoc \
    -I=proto \
    --python_out=gen/python \
    --grpc_python_out=gen/python \
    proto/bank/v1/bank.proto
$ ls gen/python/bank/v1/
bank_pb2.py  bank_pb2_grpc.py
```

### Drive protoc by hand — Java

```bash
$ protoc \
    --java_out=src/main/java \
    --grpc-java_out=src/main/java \
    --plugin=protoc-gen-grpc-java=$(which protoc-gen-grpc-java) \
    proto/bank/v1/bank.proto
```

### Drive protoc by hand — Node / TypeScript

```bash
$ grpc_tools_node_protoc \
    --js_out=import_style=commonjs:gen/js \
    --grpc_out=grpc_js:gen/js \
    -I proto \
    proto/bank/v1/bank.proto
```

### List services on a running server (reflection enabled)

```bash
$ grpcurl --plaintext localhost:50051 list
bank.v1.BankService
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
```

### Describe a service

```bash
$ grpcurl --plaintext localhost:50051 describe bank.v1.BankService
bank.v1.BankService is a service:
service BankService {
  rpc GetAccount ( .bank.v1.GetAccountRequest ) returns ( .bank.v1.Account );
  rpc ListAccounts ( .bank.v1.ListAccountsRequest ) returns ( stream .bank.v1.Account );
  rpc Deposit ( stream .bank.v1.DepositChunk ) returns ( .bank.v1.DepositSummary );
  rpc Chat ( stream .bank.v1.ChatMessage ) returns ( stream .bank.v1.ChatMessage );
}
```

### Call a unary RPC

```bash
$ grpcurl --plaintext -d '{"account_id":"abc"}' localhost:50051 bank.v1.BankService/GetAccount
{
  "accountId": "abc",
  "ownerName": "Alice",
  "balanceCents": "104250",
  "currency": "CURRENCY_USD"
}
```

### Call without reflection (using the .proto)

```bash
$ grpcurl -import-path=./proto -proto=bank/v1/bank.proto \
    --plaintext -d '{"account_id":"abc"}' \
    localhost:50051 bank.v1.BankService/GetAccount
{
  "accountId": "abc",
  "ownerName": "Alice",
  "balanceCents": "104250"
}
```

### Send custom metadata

```bash
$ grpcurl --plaintext \
    -H 'authorization: Bearer eyJhbGc...' \
    -H 'x-tenant-id: acme' \
    -d '{"account_id":"abc"}' \
    localhost:50051 bank.v1.BankService/GetAccount
{
  "accountId": "abc"
}
```

### Try a server-streaming RPC

```bash
$ grpcurl --plaintext -d '{"page_size":3}' localhost:50051 bank.v1.BankService/ListAccounts
{ "accountId": "abc", "ownerName": "Alice"   }
{ "accountId": "def", "ownerName": "Bob"     }
{ "accountId": "ghi", "ownerName": "Charlie" }
```

### Web UI for gRPC

```bash
$ grpcui --plaintext localhost:50051
gRPC Web UI available at http://127.0.0.1:8080/
```

### Load test with ghz

```bash
$ ghz --insecure \
    --proto ./proto/bank/v1/bank.proto \
    --import-paths=./proto \
    --call bank.v1.BankService/GetAccount \
    -d '{"account_id":"abc"}' \
    -n 100000 -c 50 \
    localhost:50051

Summary:
  Count:        100000
  Total:        4.21 s
  Slowest:      48.17 ms
  Fastest:      0.41 ms
  Average:      1.92 ms
  Requests/sec: 23752.97
```

### Interactive REPL

```bash
$ evans -r repl
bank.v1.BankService@localhost:50051> show service
+--------------+--------------------------------+--------+
|   SERVICE    |              RPC               | STREAM |
+--------------+--------------------------------+--------+
| BankService  | GetAccount                     |        |
| BankService  | ListAccounts                   | server |
+--------------+--------------------------------+--------+
bank.v1.BankService@localhost:50051> call GetAccount
account_id (TYPE_STRING) => abc
{
  "accountId": "abc",
  "ownerName": "Alice"
}
```

### Health check

```bash
$ grpc_health_probe -addr=localhost:50051
status: SERVING
```

### Verify the server is speaking HTTP/2

```bash
$ openssl s_client -connect bank.example.com:443 -alpn h2 </dev/null 2>/dev/null | grep -i 'alpn\|protocol'
ALPN protocol: h2
```

### Hand-craft an HTTP/2 gRPC request with curl (rare, for debugging)

```bash
$ curl --http2-prior-knowledge \
    -X POST \
    -H 'Content-Type: application/grpc' \
    -H 'TE: trailers' \
    --data-binary @req.bin \
    http://localhost:50051/bank.v1.BankService/GetAccount \
    --output resp.bin
```

(The body needs to be a length-prefixed protobuf — most people don't do this directly; they use `grpcurl`.)

### Forward a gRPC port from Kubernetes

```bash
$ kubectl port-forward svc/grpc-server 50051:50051
Forwarding from 127.0.0.1:50051 -> 50051
Forwarding from [::1]:50051 -> 50051
```

```bash
$ kubectl exec -it deploy/grpc-server -- grpc_health_probe -addr=:9000
status: SERVING
```

### Inspect Kubernetes Gateway-API gRPCRoute

```bash
$ kubectl get crd grpcroute.gateway.networking.k8s.io
NAME                                  CREATED AT
grpcroute.gateway.networking.k8s.io   2025-01-12T15:33:21Z
```

### Helm install your gRPC service

```bash
$ helm install grpc-svc ./chart
NAME: grpc-svc
LAST DEPLOYED: Mon Apr 27 09:14:02 2026
STATUS: deployed
```

### Scrape gRPC metrics with Prometheus

If your server exposes `/metrics` (e.g., via `grpc-ecosystem/go-grpc-middleware/v2/metrics/prometheus`):

```bash
$ curl -s localhost:9090/metrics | grep grpc_server_handled_total | head -3
grpc_server_handled_total{grpc_code="OK",grpc_method="GetAccount",grpc_service="bank.v1.BankService"} 1042
grpc_server_handled_total{grpc_code="NotFound",grpc_method="GetAccount",grpc_service="bank.v1.BankService"} 17
grpc_server_handled_total{grpc_code="OK",grpc_method="ListAccounts",grpc_service="bank.v1.BankService"} 233
```

### Add OpenTelemetry tracing

In Go:

```bash
$ go get go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc
```

Then on the server:

```go
grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
```

### TLS posture audit

```bash
$ sslyze --regular bank.example.com:443
 * TLS 1.2 Cipher Suites:
     Attempted to connect using 156 cipher suites; the server accepted 6:
       TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384  ...
 * TLS 1.3 Cipher Suites:
     Attempted to connect using 5 cipher suites; the server accepted 3:
       TLS_AES_256_GCM_SHA384  ...
```

## Common Confusions

### protobuf vs gRPC

These are not the same thing.

- **Protobuf** is a schema language and a binary serialization format.
- **gRPC** is an RPC framework that *uses* protobuf as its schema and serialization.

You can use protobuf without gRPC (as a fast on-disk format, or with a different RPC system, or just to send messages over Kafka). You can technically use gRPC with other formats (gRPC supports JSON encoding via `application/grpc+json`), but in practice nobody does.

### proto2 vs proto3

- proto2 — original, has `required`/`optional` keywords, has explicit field defaults.
- proto3 — newer (2016), simpler, removed `required`. Re-introduced `optional` in 3.15 (2021). Use proto3.

### gRPC requires HTTP/2

This is non-negotiable. The wire protocol is built on HTTP/2 framing (DATA frames carry the message bytes, HEADERS frames carry initial and trailing metadata). Browsers, load balancers, and proxies must all speak HTTP/2 (or you go via gRPC-Web through a proxy).

### The client-side streaming gotcha

When you implement a client-streaming or bidi handler on the server, the server **does not see anything** until you call `Recv` (or your language's equivalent) inside a loop. People assume their handler is broken when really they just forgot to read the stream.

### Deadlines vs timeouts

A timeout is local. A deadline is a global wall-clock instant that propagates through every downstream call. Use deadlines.

### The "Unimplemented" gotcha when proto changes

If a client built from `bank.proto v1` calls a server built from `bank.proto v2` where a method was renamed, the client gets `Unimplemented` (because the server has no method by the old name). Always reserve removed names. Always do a coordinated rollout.

### Reflection vs breaking changes

Reflection lets a client discover the *current* schema of the server, but it doesn't give you the schema the client was *built* with. Reflection is great for live debugging; it does **not** save you from breaking changes deployed across multiple versions of clients.

### gRPC-Web vs Connect-Web

- gRPC-Web — needs a proxy (Envoy or grpc-web-proxy), no client-streaming or bidi.
- Connect-Web — no proxy needed, simpler protocol, good unary and server-streaming, and the same client can talk to a gRPC server too (for unary).

### What is "trailers" and why JSON-over-HTTP can't easily do them

Trailers are HTTP/2 headers sent **after** the response body. gRPC uses them to send the final status code at the end of a streaming response. HTTP/1 supports trailers in theory but most stacks ignore them. JSON-over-HTTP/1 has to encode the final status into the body (e.g., a JSON envelope `{"status": "ok", "data": [...]}`) instead.

### Server interceptor vs handler

The handler is the function that does the actual work for one method. An interceptor is a wrapper that runs around every (or many) handlers, doing cross-cutting work like auth or logging. You write one handler per method. You write one interceptor per concern (auth, log, metric, …) and apply it to many methods.

### Backoff/retry policy

gRPC supports a declarative retry policy as part of the **service config** (a JSON document the channel reads). It says: for these status codes, retry up to N times with exponential backoff. **Default is no retries.** Don't assume gRPC retries for you — it doesn't unless you configure it.

### Load balancing — channel-level vs hostname

`pick_first` (the default before name resolution returned multiple addresses) just connects to one. `round_robin` connects to all and rotates. With **xDS** you get full envoy-style policies. The choice happens **on the client** (channel-level), not the DNS hostname; gRPC clients have first-class load balancing baked in.

### Metadata vs payload

Payload = the protobuf message. Metadata = HTTP/2 headers (and trailers). Don't put auth tokens in the payload; don't put domain data in the metadata.

### How to send credentials per-call

Use **call credentials** (a per-call interceptor that sets the `authorization` header). Don't put the token in the channel — channels are long-lived and tokens rotate.

### ALTS only works on Google infrastructure

ALTS is great if you're inside Google. It is not a general-purpose protocol. If you're on AWS or your own data center, use mTLS.

### Status code "Internal" is not a 500

In HTTP land, a 500 means "I crashed." In gRPC, `Internal` is a specific code distinct from `Unknown`, `DataLoss`, etc. The HTTP/2 *response* status is always 200 OK if the connection worked at all — the gRPC status code lives in trailers, not in the HTTP status line.

## Vocabulary

A long alphabetical list. Look up any word that confused you.

- **ALPN** — "Application Layer Protocol Negotiation." A TLS extension that lets client and server agree on a protocol (e.g., `h2` for HTTP/2) during the TLS handshake.
- **ALPN h2** — the literal string `h2`, meaning HTTP/2. gRPC over TLS requires ALPN to negotiate `h2`.
- **ALTS** — "Application Layer Transport Security." Google's homemade transport security for service-to-service traffic inside Google. Not for use outside Google's infrastructure.
- **API** — "Application Programming Interface." The contract by which one program asks another to do something.
- **api key** — a secret string sent (often as metadata) to identify a caller. Simpler than OAuth, less secure.
- **application/grpc** — the HTTP `Content-Type` for default gRPC (binary protobuf). Variants: `application/grpc+proto` (explicit), `application/grpc+json` (rare).
- **buf** — the modern protobuf build tool from Buf, Inc. Single binary that lints, formats, breaks-checks, and generates code.
- **buf.build** — the company's hosted Schema Registry (BSR).
- **BSR** — "Buf Schema Registry." Hosted package registry for `.proto` modules; also runs remote codegen plugins.
- **BloomRPC** — old GUI client for gRPC. **Deprecated.** Use grpcui or Postman.
- **bidi-streaming** — both sides send streams to each other on the same call, in any order.
- **breaking change** — a schema change that older clients can't safely consume (renamed/removed/typed-differently fields).
- **call credentials** — per-RPC credentials (e.g., an OAuth token in a header).
- **channel** — the long-lived client-side object representing the connection(s) to a server or pool. In Go: `*grpc.ClientConn`.
- **channel credentials** — per-channel credentials (TLS, mTLS, ALTS).
- **channelz** — built-in gRPC introspection service exposing per-channel/per-call metrics. Run `grpcdebug` to query it.
- **circuit breaker** — pattern where a client stops calling a failing server for a while. Configurable in xDS.
- **client** — the side that initiates a call.
- **client-streaming** — many requests, one response.
- **code generation** — turning a `.proto` into a language-specific client/server stub.
- **Connect** — newer RPC protocol from Buf (2022). Wire-compatible with gRPC for unary; simpler framing.
- **Connect-Go / -Web / -Swift / -Kotlin** — official Connect client/server libraries by language.
- **ConnectRPC** — the project name (https://connectrpc.com).
- **content-type** — the HTTP header that names the wire format. For gRPC, `application/grpc` (or `+proto`/`+json`).
- **deadline** — wall-clock instant after which a call should be aborted. Propagates through downstream calls.
- **descriptor** — the parsed, machine-readable form of a `.proto` file. `FileDescriptor`, `MessageDescriptor`, etc.
- **DialContext** — older gRPC-Go function for creating a channel. Replaced by `NewClient` in 1.65.
- **dns:///** — gRPC name-resolver scheme that does DNS lookups.
- **enum** — protobuf enumeration type. First value must be 0 by convention `*_UNSPECIFIED`.
- **Envoy** — popular service proxy that speaks gRPC, gRPC-Web, and many other things.
- **errdetails** — extra structured error info attached to gRPC status. Types include `RetryInfo`, `DebugInfo`, `QuotaFailure`, `ErrorInfo`, `BadRequest`, `RequestInfo`, `ResourceInfo`, `Help`, `LocalizedMessage`.
- **evans** — interactive gRPC REPL.
- **field tag** — the integer after `=` in a proto field. What goes on the wire.
- **flow control** — HTTP/2 mechanism that prevents senders from overwhelming receivers; gRPC inherits it.
- **gogo/protobuf** — older Go protobuf library, **legacy / deprecated**. Use `google.golang.org/protobuf`.
- **google.golang.org/grpc** — the official Go gRPC module path.
- **google.golang.org/protobuf** — the official Go protobuf module path (modern, replaces `github.com/golang/protobuf`).
- **gRPC** — the framework. Sometimes pronounced "GRP-C" (G-R-P-C). Officially the "g" doesn't stand for anything.
- **gRPC-C++** — the C++ implementation of gRPC. Moved to abseil dependency in 2020.
- **gRPC-CSM** — Cloud Service Mesh integration; uses xDS to configure gRPC clients.
- **gRPC-Gateway** — generates a REST+JSON reverse proxy for a gRPC server using `google.api.http` annotations.
- **gRPC-Go** — the Go implementation.
- **gRPC-Java** — the Java implementation.
- **gRPC-LB** — older protocol for load balancing, **deprecated** in favor of xDS.
- **gRPC-Node** — the Node.js implementation. Deprecating the C++ binding in favor of pure JS.
- **gRPC-Python** — the Python implementation. Modern code uses `grpc.aio`.
- **gRPC-Web** — variant wire format that browsers can speak via fetch. Needs a proxy.
- **grpc-go-channelz** — the channelz tooling for Go.
- **grpc-spring-boot** — Spring Boot starter for gRPC (Java).
- **grpc-java-shaded** — older Java distribution that included shaded dependencies; less common today.
- **grpc-status** — trailer header name carrying the integer status code.
- **grpc-message** — trailer header name carrying the human-readable error message.
- **grpc-status-details-bin** — binary trailer with structured `errdetails`.
- **grpcio** — the Python package name for gRPC (`pip install grpcio`).
- **grpc-rs** — older Rust gRPC binding. Most people now use **tonic**.
- **grpcurl** — a curl-like tool for gRPC, supports reflection.
- **grpcui** — a web UI for gRPC, also supports reflection.
- **ghz** — gRPC load testing tool.
- **handler** — the function on the server that implements one RPC method.
- **headers** — metadata sent at the start of a call.
- **hedging** — sending the same request to multiple servers to take whichever responds first. Configurable in service config.
- **HPACK** — HTTP/2 header compression algorithm. Saves bytes on repeated headers like `:path`.
- **HTTP/2** — the underlying transport for gRPC. Multiplexed binary framing.
- **HTTP/2 streams** — independent bidirectional channels within one HTTP/2 connection. Each gRPC call uses one stream.
- **import** — proto keyword to pull in another `.proto` file.
- **Insomnia** — REST/GraphQL client with gRPC support.
- **jwt credentials** — call credentials that attach a JSON Web Token in the `authorization` header.
- **JSON-RPC** — older RPC protocol (text/JSON over HTTP). Less efficient than gRPC; simpler.
- **keepalive_permit_without_calls** — gRPC option allowing keepalive pings even when no RPCs are in flight.
- **keepalive_time_ms** — how often the client sends HTTP/2 PING frames to detect dead connections.
- **keepalive_timeout_ms** — how long to wait for the PING ack before concluding the connection is dead.
- **Linkerd** — service mesh that handles gRPC load balancing transparently.
- **load balancing** — strategy for picking a backend. Built into gRPC clients (channel-level), not just DNS.
- **map<K,V>** — protobuf field type for key/value maps.
- **maxConnectionAge** — server option: disconnect a client after this much wall-clock time. Forces clients to re-resolve DNS.
- **maxConnectionAgeGrace** — additional grace period before forcibly closing a connection at maxConnectionAge.
- **maxConnectionIdle** — server option: disconnect an idle client after this duration.
- **message** — the protobuf word for a struct.
- **metadata** — gRPC's name for headers (and trailers).
- **mTLS** — mutual TLS; both sides present and verify certificates.
- **multiplexing** — sending many independent streams over one HTTP/2 connection.
- **name resolution** — turning a target string like `dns:///bank.example.com:50051` into a list of IPs.
- **NewClient** — the modern (1.65+) gRPC-Go function to create a channel. Lazy by default.
- **NewServer** — the gRPC-Go function to create a server.
- **oauth credentials** — call credentials that attach an OAuth 2.0 access token.
- **oneof** — protobuf field group where exactly one member is set.
- **OpenCensus** — older observability framework, predecessor of OpenTelemetry.
- **OpenTelemetry / OTel** — modern observability framework. `otelgrpc` package instruments gRPC.
- **OpenTracing** — older tracing API, **legacy**.
- **optional** — proto3 keyword for explicit field presence (re-introduced 3.15).
- **package** — proto keyword for namespacing.
- **paths=source_relative** — `protoc-gen-go` option to put generated files next to their `.proto`.
- **PermissionDenied** — status code 7. Caller is known but not allowed.
- **Postman gRPC support** — Postman has had a gRPC tab since 2022.
- **proto2** — older protobuf syntax. Has `required`. Avoid in new code.
- **proto3** — modern protobuf syntax. Use this.
- **protoc** — the protobuf compiler. Calls plugins to generate per-language code.
- **protoc-gen-connect-go** — Connect's Go codegen plugin.
- **protoc-gen-doc** — generates HTML/markdown docs from protos.
- **protoc-gen-go** — Go message codegen plugin.
- **protoc-gen-go-grpc** — Go service codegen plugin.
- **protoc-gen-grpc-gateway** — gRPC-Gateway codegen plugin.
- **protoc-gen-grpc-web** — gRPC-Web JS codegen plugin.
- **protoc-gen-openapiv2** — generate OpenAPI/Swagger doc from protos.
- **protoc-gen-validate** — generate validators from `validate.rules` annotations.
- **protocol buffers** — Google's binary serialization format. The schema language for gRPC.
- **REFLECTION_V1ALPHA** — older reflection service version. Modern is `grpc.reflection.v1`.
- **repeated** — protobuf field type for lists.
- **request** — the input message of an RPC.
- **reserved** — proto keyword to mark deleted field tags / names to prevent reuse.
- **REST/HTTP** — the alternative-to-gRPC paradigm: resources + verbs + JSON.
- **response** — the output message of an RPC.
- **retries** — automatic resending of failed RPCs. Configured in service config; off by default.
- **retry policy** — service config block listing retryable codes, max attempts, backoff.
- **RST_STREAM** — HTTP/2 frame that abruptly terminates a stream. Often appears in `Internal` errors.
- **rpc** — proto keyword for an RPC method definition.
- **scalar** — protobuf primitive type (`int32`, `string`, `bool`, etc.).
- **server** — the side that responds to RPCs.
- **server reflection** — optional gRPC service letting clients discover the schema at runtime.
- **server-streaming** — one request, many responses.
- **service** — proto keyword for a group of RPCs.
- **service config** — a JSON document the gRPC client reads to configure retries, load balancing, etc. Often delivered via DNS TXT records or xDS.
- **stream** — proto keyword turning an RPC parameter into a stream.
- **status code** — the integer code accompanying every gRPC response (0 = OK, 1 = Cancelled, …).
- **stub** — generated typed wrapper around a channel that exposes one method per RPC.
- **syntax** — proto keyword declaring the language version (`proto2` or `proto3`).
- **Timestamp** — well-known type representing a point in time (seconds + nanoseconds since UNIX epoch).
- **tonic** — the dominant Rust gRPC framework, built on Tokio.
- **trailers** — HTTP/2 metadata sent at the *end* of a stream. Carries the gRPC status.
- **Twirp** — alternative RPC framework (HTTP+JSON or HTTP+protobuf, no streaming). Simpler than gRPC, less powerful.
- **Unauthenticated** — status code 16. Caller's credentials are missing/invalid.
- **Unavailable** — status code 14. Transient failure; usually retryable.
- **Unimplemented** — status code 12. Method or service not found on server.
- **unary** — one request, one response.
- **uint32 / uint64** — protobuf unsigned-integer scalars.
- **vocabulary** — this list. Hi.
- **well-known types** — common types shipped with protobuf: `Empty`, `Timestamp`, `Duration`, `Any`, `Struct`, `Value`, `ListValue`, `NullValue`, `FieldMask`, plus wrapper types `BoolValue`, `Int32Value`, `StringValue`, etc.
- **Wireshark gRPC dissector** — Wireshark plugin that decodes gRPC over HTTP/2.
- **xds:///** — gRPC name-resolver scheme that pulls config from an xDS control plane (Envoy-style).
- **xDS** — set of gRPC/Envoy APIs for cluster discovery, load balancing, routing.

## Try This

Hands-on time. Pick one of these tracks based on your background.

### Track A — I have never used gRPC

1. Install `protoc`, `buf`, `grpcurl`, and `ghz`. On macOS: `brew install protobuf bufbuild/buf/buf grpcurl ghz`. On Linux: download from each project's GitHub releases.
2. Run `protoc --version` and confirm it works.
3. Find a public test gRPC server. The grpcb.in project at https://grpcb.in is a good one. Try:

```bash
$ grpcurl grpcb.in:9001 list
addsvc.Add
grpcbin.GRPCBin
```

4. Describe a service and call a method:

```bash
$ grpcurl grpcb.in:9001 describe grpcbin.GRPCBin
$ grpcurl -d '{"check_string": "hello"}' grpcb.in:9001 grpcbin.GRPCBin/Index
```

5. Write your first `.proto`. A minimal one:

```proto
syntax = "proto3";
package hello.v1;
option go_package = "example.com/hello/v1;hellov1";

service Greeter {
  rpc Hello(HelloRequest) returns (HelloResponse);
}
message HelloRequest { string name = 1; }
message HelloResponse { string greeting = 1; }
```

6. Generate Go code with `buf generate` (after a `buf.gen.yaml`).
7. Write a 30-line server that returns "Hello, <name>!" and a 20-line client.
8. Run them locally on `localhost:50051`.
9. Hit the server with `grpcurl --plaintext -d '{"name":"Alice"}' localhost:50051 hello.v1.Greeter/Hello`. Confirm you see `{"greeting":"Hello, Alice!"}`.

### Track B — I have used REST, never gRPC

1. Take an existing REST endpoint you wrote.
2. Define the same operation as a `.proto` `service` with one `rpc`.
3. Pay attention to: how the request body becomes a request `message`; how query params become fields; how status codes map.
4. Generate code, write a thin gRPC server, call it from `grpcurl`.
5. Add `gRPC-Gateway` annotations and put the same operation back on REST automatically.

### Track C — I want to break things

1. Set up a server with reflection on.
2. Run `grpcurl -d '{}' localhost:50051 …/SomeMethod` with an empty body. See what defaults the server sees.
3. Send a deliberately malformed request (e.g., wrong field type) and read the `InvalidArgument` error.
4. Set a tiny deadline (`-d '{}' --max-time 0.001`) and watch `DeadlineExceeded`.
5. Stop the server, hit it again, watch `Unavailable`.
6. Implement a server interceptor that randomly returns `Unavailable` 10% of the time. Configure a client retry policy via service config and watch retries succeed.

## Where to Go Next

Once this sheet feels comfortable:

- `cs api/grpc` — the long-form reference sheet for everyday gRPC use.
- `cs ramp-up/graphql-eli5` — for an alternative API style with similar "schema-first" energy but very different shape.
- `cs ramp-up/http3-quic-eli5` — gRPC's transport story is moving toward HTTP/3 over QUIC; this is the ground truth.
- `cs ramp-up/kubernetes-eli5` — gRPC services in production almost always run on Kubernetes.
- `cs ramp-up/go-eli5` and `cs ramp-up/python-eli5` — the two most-used gRPC languages by far.
- The book *gRPC: Up & Running* by Kasun Indrasiri and Danesh Kuruppu (O'Reilly, 2nd ed.) — the de-facto introduction.

## See Also

- `api/grpc`
- `api/graphql`
- `api/rest`
- `api/openapi`
- `networking/grpc`
- `networking/http2`
- `networking/http3`
- `networking/tcp`
- `security/tls`
- `ramp-up/tcp-eli5`
- `ramp-up/tls-eli5`
- `ramp-up/http3-quic-eli5`
- `ramp-up/graphql-eli5`
- `ramp-up/kubernetes-eli5`
- `ramp-up/go-eli5`
- `ramp-up/python-eli5`
- `ramp-up/linux-kernel-eli5`

## References

- gRPC documentation — https://grpc.io/docs/
- Protocol Buffers documentation — https://protobuf.dev
- *gRPC: Up & Running* — Kasun Indrasiri and Danesh Kuruppu (O'Reilly, 2nd ed., 2020)
- Buf documentation — https://buf.build/docs/
- ConnectRPC — https://connectrpc.com
- Google API Improvement Proposals — https://google.aip.dev
- gRPC GitHub — https://github.com/grpc/grpc
- HTTP/2 RFC 9113 — https://www.rfc-editor.org/rfc/rfc9113
- HTTP/3 RFC 9114 — https://www.rfc-editor.org/rfc/rfc9114
- TLS 1.3 RFC 8446 — https://www.rfc-editor.org/rfc/rfc8446
