# Protocol Buffers (Protobuf)

> Google's language-neutral, platform-neutral binary serialization format — defines structured data in `.proto` files, generates type-safe code via `protoc` compiler, uses compact varint encoding, and serves as the foundation for gRPC service definitions.

## Proto3 Syntax

### Messages

```protobuf
syntax = "proto3";

package myapp.v1;

option go_package = "github.com/myorg/myapp/proto/v1";

message User {
  string id = 1;
  string name = 2;
  string email = 3;
  int32 age = 4;
  repeated string tags = 5;
  Address address = 6;
  UserStatus status = 7;
}

message Address {
  string street = 1;
  string city = 2;
  string state = 3;
  string zip = 4;
  string country = 5;
}
```

### Enums

```protobuf
enum UserStatus {
  USER_STATUS_UNSPECIFIED = 0;  // must have zero value
  USER_STATUS_ACTIVE = 1;
  USER_STATUS_INACTIVE = 2;
  USER_STATUS_BANNED = 3;
}

// Allow aliases
enum Priority {
  option allow_alias = true;
  PRIORITY_UNSPECIFIED = 0;
  PRIORITY_LOW = 1;
  PRIORITY_NORMAL = 2;
  PRIORITY_DEFAULT = 2;  // alias for NORMAL
  PRIORITY_HIGH = 3;
}
```

### Oneof

```protobuf
message Payment {
  string id = 1;
  double amount = 2;

  oneof method {
    CreditCard credit_card = 3;
    BankTransfer bank_transfer = 4;
    CryptoWallet crypto = 5;
  }
}

message CreditCard {
  string number = 1;
  string expiry = 2;
  string cvv = 3;
}

message BankTransfer {
  string routing = 1;
  string account = 2;
}

message CryptoWallet {
  string address = 1;
  string network = 2;
}
```

### Maps and Repeated

```protobuf
message Config {
  map<string, string> labels = 1;
  map<string, int32> ports = 2;
  repeated string hosts = 3;
  repeated Endpoint endpoints = 4;
}

message Endpoint {
  string url = 1;
  int32 weight = 2;
}
```

## Scalar Types

### Type Reference

```protobuf
// Integers
int32    // varint, inefficient for negatives
int64    // varint, inefficient for negatives
uint32   // varint, unsigned
uint64   // varint, unsigned
sint32   // ZigZag + varint (efficient for negatives)
sint64   // ZigZag + varint (efficient for negatives)
fixed32  // always 4 bytes (efficient for > 2^28)
fixed64  // always 8 bytes (efficient for > 2^56)
sfixed32 // always 4 bytes, signed
sfixed64 // always 8 bytes, signed

// Floating point
float    // 4 bytes
double   // 8 bytes

// Other
bool     // varint (0 or 1)
string   // UTF-8 or 7-bit ASCII
bytes    // arbitrary byte sequence
```

## Field Numbers and Encoding

### Field Number Rules

```protobuf
message Example {
  // Field numbers 1-15 use 1 byte (tag) — use for frequent fields
  string name = 1;
  int32 type = 2;

  // Field numbers 16-2047 use 2 bytes
  string description = 16;

  // Reserved: never reuse deleted field numbers
  reserved 6, 9 to 11;
  reserved "old_field", "deprecated_field";

  // Max field number: 536,870,911 (2^29 - 1)
  // 19000-19999 reserved by protobuf implementation
}
```

### Wire Types

```
Wire Type  Format               Used For
0          Varint               int32, int64, uint32, uint64, sint32,
                                sint64, bool, enum
1          64-bit (fixed)       fixed64, sfixed64, double
2          Length-delimited     string, bytes, embedded messages,
                                repeated fields (packed)
5          32-bit (fixed)       fixed32, sfixed32, float
```

## protoc Compiler

### Code Generation

```bash
# Install protoc
# macOS
brew install protobuf

# Linux
apt-get install -y protobuf-compiler

# Install language plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Python
pip install grpcio-tools

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/v1/*.proto

# Generate Python code
python -m grpc_tools.protoc -I. \
  --python_out=. --grpc_python_out=. \
  proto/v1/*.proto

# Multiple input directories
protoc -I proto/ -I third_party/ \
  --go_out=gen/ --go_opt=paths=source_relative \
  proto/v1/user.proto
```

### buf Tool (Modern Alternative)

```bash
# Install buf
brew install bufbuild/buf/buf

# Initialize project
buf config init

# Lint proto files
buf lint

# Detect breaking changes
buf breaking --against '.git#branch=main'

# Generate code (buf.gen.yaml)
buf generate
```

```yaml
# buf.gen.yaml
version: v2
plugins:
  - remote: buf.build/protocolbuffers/go
    out: gen/go
    opt: paths=source_relative
  - remote: buf.build/grpc/go
    out: gen/go
    opt: paths=source_relative
```

## Proto3 vs Proto2

### Key Differences

```protobuf
// Proto2: explicit required/optional
// Proto3: all fields optional by default (no "required")

// Proto2
message UserV2 {
  required string name = 1;
  optional int32 age = 2;
  optional string email = 3 [default = "none"];
}

// Proto3 — no required, no defaults, optional keyword for presence tracking
message UserV3 {
  string name = 1;         // always has zero value if unset
  int32 age = 2;
  optional string email = 3;  // has_email() available
}
```

## Well-Known Types

### Common Types

```protobuf
import "google/protobuf/timestamp.proto";
import "google/protobuf/duration.proto";
import "google/protobuf/any.proto";
import "google/protobuf/struct.proto";
import "google/protobuf/wrappers.proto";
import "google/protobuf/field_mask.proto";
import "google/protobuf/empty.proto";

message Event {
  string id = 1;
  google.protobuf.Timestamp created_at = 2;
  google.protobuf.Duration ttl = 3;
  google.protobuf.Any payload = 4;
  google.protobuf.Struct metadata = 5;
  google.protobuf.StringValue nullable_name = 6;
  google.protobuf.FieldMask update_mask = 7;
}
```

```go
// Go usage of Timestamp
import timestamppb "google.golang.org/protobuf/types/known/timestamppb"

event := &pb.Event{
    CreatedAt: timestamppb.Now(),
    Ttl:       durationpb.New(5 * time.Minute),
}

t := event.GetCreatedAt().AsTime() // -> time.Time
```

## gRPC Service Definitions

### Service and RPC

```protobuf
service UserService {
  // Unary
  rpc GetUser(GetUserRequest) returns (GetUserResponse);

  // Server streaming
  rpc ListUsers(ListUsersRequest) returns (stream User);

  // Client streaming
  rpc UploadUsers(stream User) returns (UploadUsersResponse);

  // Bidirectional streaming
  rpc Chat(stream ChatMessage) returns (stream ChatMessage);
}

message GetUserRequest {
  string id = 1;
}

message GetUserResponse {
  User user = 1;
}

message ListUsersRequest {
  int32 page_size = 1;
  string page_token = 2;
  string filter = 3;
}
```

## Tips

- Use field numbers 1-15 for your most frequently populated fields — they encode in a single byte
- Never reuse or reassign field numbers — use `reserved` to prevent accidental reuse of removed fields
- Always use `sint32`/`sint64` for fields that will hold negative numbers — standard `int32` uses 10 bytes for negatives
- Name enum zero values as `_UNSPECIFIED` to represent the default/unknown state explicitly
- Use `optional` keyword in proto3 when you need to distinguish "field not set" from "field is zero value"
- Use `google.protobuf.FieldMask` for partial update APIs to specify exactly which fields to modify
- Prefer `buf` over raw `protoc` — it provides linting, breaking change detection, and dependency management
- Use `oneof` instead of multiple optional fields when exactly one variant should be set
- Keep proto files in a versioned directory structure (`proto/v1/`, `proto/v2/`) for API evolution
- Wrap primitives with `google.protobuf.StringValue` etc. when null semantics are needed (e.g., PATCH APIs)
- Set `option go_package` in every proto file to control Go import paths explicitly

## See Also

- avro, json, yaml, toml, xml

## References

- [Protocol Buffers Language Guide (proto3)](https://protobuf.dev/programming-guides/proto3/)
- [Protocol Buffers Encoding](https://protobuf.dev/programming-guides/encoding/)
- [gRPC Documentation](https://grpc.io/docs/)
- [Buf CLI Documentation](https://buf.build/docs/)
- [Protocol Buffers Well-Known Types](https://protobuf.dev/reference/protobuf/google.protobuf/)
- [Google API Design Guide](https://cloud.google.com/apis/design)
