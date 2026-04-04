# gRPC (gRPC Remote Procedure Calls)

High-performance RPC framework built on HTTP/2 and Protocol Buffers, supporting unary, server-streaming, client-streaming, and bidirectional streaming calls with built-in deadlines, cancellation, interceptors, and language-agnostic code generation.

## Core Concepts

```
# gRPC stack:
#   Application Code
#   ├── Generated Stubs (client) / Service Impl (server)
#   ├── gRPC Framework (channels, interceptors, load balancing)
#   ├── HTTP/2 Transport (streams, frames, flow control)
#   └── TLS / TCP

# Key properties:
# - Binary protocol (Protobuf) — smaller + faster than JSON
# - HTTP/2 multiplexing — many RPCs on one connection
# - Bidirectional streaming — client and server send independently
# - Strongly typed — code generated from .proto definitions
# - Deadline propagation — timeouts flow through call chains
# - Language-agnostic — Go, Java, Python, C++, Rust, etc.
```

## Protobuf Definition

```protobuf
// service.proto
syntax = "proto3";

package myservice;

option go_package = "github.com/example/myservice/pb";

// Service definition
service UserService {
    // Unary — one request, one response
    rpc GetUser(GetUserRequest) returns (GetUserResponse);

    // Server streaming — one request, stream of responses
    rpc ListUsers(ListUsersRequest) returns (stream User);

    // Client streaming — stream of requests, one response
    rpc UploadUsers(stream User) returns (UploadUsersResponse);

    // Bidirectional streaming — both sides stream independently
    rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

message GetUserRequest {
    string user_id = 1;
}

message GetUserResponse {
    User user = 1;
}

message User {
    string id = 1;
    string name = 2;
    string email = 3;
    int32 age = 4;
    repeated string roles = 5;
    map<string, string> metadata = 6;
}

message ListUsersRequest {
    int32 page_size = 1;
    string page_token = 2;
}
```

## Code Generation

```bash
# Install protoc compiler
# macOS
brew install protobuf

# Linux
apt install -y protobuf-compiler

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/service.proto

# Generated files:
#   service.pb.go        — message types
#   service_grpc.pb.go   — client stub + server interface

# Python
pip install grpcio grpcio-tools
python -m grpc_tools.protoc -I. --python_out=. --grpc_python_out=. proto/service.proto

# buf (modern protobuf toolchain)
buf generate
buf lint
buf breaking --against '.git#branch=main'
```

## RPC Patterns

### Unary RPC

```go
// Server
func (s *server) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
    user, err := s.db.FindUser(req.UserId)
    if err != nil {
        return nil, status.Errorf(codes.NotFound, "user %s not found", req.UserId)
    }
    return &pb.GetUserResponse{User: user}, nil
}

// Client
resp, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: "123"})
if err != nil {
    st, ok := status.FromError(err)
    if ok && st.Code() == codes.NotFound {
        // handle not found
    }
}
```

### Server Streaming

```go
// Server — sends multiple responses
func (s *server) ListUsers(req *pb.ListUsersRequest, stream pb.UserService_ListUsersServer) error {
    users := s.db.GetAllUsers()
    for _, user := range users {
        if err := stream.Send(user); err != nil {
            return err
        }
    }
    return nil
}

// Client — receives stream
stream, err := client.ListUsers(ctx, &pb.ListUsersRequest{PageSize: 100})
for {
    user, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(user.Name)
}
```

### Client Streaming / Bidirectional Streaming

```go
// Client streaming: server receives stream, sends single response
// Server calls stream.Recv() in loop, returns stream.SendAndClose()

// Bidirectional: both sides call Send() and Recv() independently
// Server loop: stream.Recv() → process → stream.Send()
// Client loop: stream.Send() → stream.Recv() (can be concurrent goroutines)
```

## Deadlines and Timeouts

```go
// Client — set timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
resp, err := client.GetUser(ctx, req)

// Deadline propagation — automatically forwarded through call chain
// ServiceA (5s deadline) → ServiceB (remaining ~4.5s) → ServiceC (remaining ~4s)
```

## Status Codes

```
Code                  Number  HTTP Equiv  Description
──────────────────────────────────────────────────────────────────────
OK                    0       200         Success
CANCELLED             1       499         Client cancelled
UNKNOWN               2       500         Unknown error
INVALID_ARGUMENT      3       400         Bad request parameter
DEADLINE_EXCEEDED     4       504         Timeout
NOT_FOUND             5       404         Resource not found
ALREADY_EXISTS        6       409         Resource already exists
PERMISSION_DENIED     7       403         Not authorized
RESOURCE_EXHAUSTED    8       429         Rate limited / quota
FAILED_PRECONDITION   9       400         System not in required state
ABORTED               10      409         Concurrency conflict
OUT_OF_RANGE          11      400         Value out of valid range
UNIMPLEMENTED         12      501         Method not implemented
INTERNAL              13      500         Internal server error
UNAVAILABLE           14      503         Service temporarily unavailable
DATA_LOSS             15      500         Unrecoverable data loss
UNAUTHENTICATED       16      401         Missing/invalid credentials
```

## Interceptors (Middleware)

```go
// Unary server interceptor
func loggingInterceptor(
    ctx context.Context,
    req interface{},
    info *grpc.UnaryServerInfo,
    handler grpc.UnaryHandler,
) (interface{}, error) {
    start := time.Now()
    resp, err := handler(ctx, req)
    log.Printf("method=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
    return resp, err
}

// Stream server interceptor
func streamLoggingInterceptor(
    srv interface{},
    ss grpc.ServerStream,
    info *grpc.StreamServerInfo,
    handler grpc.StreamHandler,
) error {
    start := time.Now()
    err := handler(srv, ss)
    log.Printf("stream=%s duration=%s err=%v", info.FullMethod, time.Since(start), err)
    return err
}

// Register interceptors
server := grpc.NewServer(
    grpc.UnaryInterceptor(loggingInterceptor),
    grpc.StreamInterceptor(streamLoggingInterceptor),
    // Chain multiple interceptors
    grpc.ChainUnaryInterceptor(authInterceptor, loggingInterceptor, metricsInterceptor),
)
```

## Reflection

```go
// Enable server reflection (for grpcurl, grpc-client-cli, etc.)
import "google.golang.org/grpc/reflection"

server := grpc.NewServer()
pb.RegisterUserServiceServer(server, &myServer{})
reflection.Register(server)  // enables reflection
```

## grpcurl

```bash
# List services
grpcurl -plaintext localhost:50051 list

# Describe a service
grpcurl -plaintext localhost:50051 describe myservice.UserService

# Describe a message type
grpcurl -plaintext localhost:50051 describe myservice.User

# Unary call
grpcurl -plaintext -d '{"user_id": "123"}' \
    localhost:50051 myservice.UserService/GetUser

# Server streaming
grpcurl -plaintext -d '{"page_size": 10}' \
    localhost:50051 myservice.UserService/ListUsers

# With TLS
grpcurl -cacert ca.pem -cert client.pem -key client-key.pem \
    myhost:50051 myservice.UserService/GetUser

# Headers (metadata)
grpcurl -plaintext -H 'authorization: Bearer TOKEN' \
    -d '{"user_id": "123"}' localhost:50051 myservice.UserService/GetUser

# Using proto file instead of reflection
grpcurl -plaintext -import-path ./proto -proto service.proto \
    -d '{"user_id": "123"}' localhost:50051 myservice.UserService/GetUser
```

## Load Balancing

```go
// Client-side load balancing (built into gRPC)
// Uses name resolver + load balancing policy

// Round-robin
conn, err := grpc.Dial(
    "dns:///myservice.example.com:50051",
    grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
    grpc.WithTransportCredentials(insecure.NewCredentials()),
)

// Pick-first (default) — uses first resolved address
// Round-robin — distributes across all resolved addresses
// Custom — implement grpc.Balancer interface

// gRPC health checking (for LB integration)
import "google.golang.org/grpc/health"
import healthpb "google.golang.org/grpc/health/grpc_health_v1"

healthServer := health.NewServer()
healthpb.RegisterHealthServer(server, healthServer)
healthServer.SetServingStatus("myservice.UserService", healthpb.HealthCheckResponse_SERVING)
```

```bash
# Proxy-based LB: Envoy, nginx (1.13.10+), HAProxy support gRPC over HTTP/2
# nginx: grpc_pass grpc://backend:50051;
# Envoy: route by /package.Service/ prefix
```

## Server Setup (Go)

```go
server := grpc.NewServer(
    grpc.MaxRecvMsgSize(10*1024*1024),         // 10 MB max message
    grpc.MaxSendMsgSize(10*1024*1024),
    grpc.MaxConcurrentStreams(1000),             // per connection
    grpc.KeepaliveParams(keepalive.ServerParameters{
        MaxConnectionIdle: 15*time.Minute,
        MaxConnectionAge:  30*time.Minute,
        Time:              5*time.Minute,       // ping interval
        Timeout:           1*time.Second,       // ping timeout
    }),
)
pb.RegisterUserServiceServer(server, &myServer{})
lis, _ := net.Listen("tcp", ":50051")
server.Serve(lis)
```

## Tips

- Always set deadlines on client calls. Without a deadline, a hung server will block the client forever. Even generous deadlines (30-60 seconds) are better than none. Use `context.WithTimeout` consistently.
- gRPC status codes are not HTTP status codes. Return `codes.NotFound` (not 404), `codes.InvalidArgument` (not 400). Use `status.Errorf()` to attach codes to errors. Never return raw Go errors from gRPC handlers.
- Protobuf field numbers are permanent. Once a message is in production, never reuse or change field numbers. Delete fields by reserving the number: `reserved 5, 6;`. This prevents future collisions.
- gRPC uses HTTP/2, which requires TLS in most deployments. For local development, use `grpc.WithTransportCredentials(insecure.NewCredentials())` on the client and no TLS on the server. Never ship this to production.
- Server reflection should be enabled in development and staging but may be disabled in production for security. It exposes your entire API schema to anyone who can connect.
- grpcurl is the curl of gRPC. Install it (`go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest`) and use it for ad-hoc testing instead of writing throwaway client code.
- Streaming RPCs hold HTTP/2 streams open. Each open stream consumes server resources (goroutine, memory). Set `MaxConcurrentStreams` to prevent a single client from exhausting the server.
- Client-side load balancing with `round_robin` requires a name resolver that returns multiple addresses (DNS with multiple A records, or a custom resolver). `pick_first` (default) only uses one address.
- Large messages (>4 MB default limit) require increasing `MaxRecvMsgSize` and `MaxSendMsgSize`. But if you are sending >10 MB, consider chunked streaming instead of giant unary messages.
- Use `buf` instead of raw `protoc` for protobuf management. It handles linting, breaking change detection, and code generation with a much better developer experience than hand-crafted protoc commands.

## See Also

- http, tcp, quic, curl

## References

- [gRPC Official Documentation](https://grpc.io/docs/)
- [Protocol Buffers Language Guide (proto3)](https://protobuf.dev/programming-guides/proto3/)
- [gRPC Go Package](https://pkg.go.dev/google.golang.org/grpc)
- [gRPC Status Codes](https://grpc.github.io/grpc/core/md_doc_statuscodes.html)
- [grpcurl](https://github.com/fullstorydev/grpcurl)
- [buf — Modern Protobuf Tooling](https://buf.build/docs/)
- [RFC 9113 — HTTP/2 (gRPC transport)](https://www.rfc-editor.org/rfc/rfc9113)
- [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
