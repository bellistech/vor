# The Mathematics of Protocol Buffers — Variable-Length Encoding and Information Density

> *Protobuf's wire format achieves compactness through varint encoding, which uses a variable number of bytes proportional to the magnitude of the value being encoded. This approach is grounded in information theory: small values (which dominate most real-world datasets) use fewer bits, while the encoding overhead for large values is bounded. The ZigZag encoding for signed integers is a bijection that maps the signed integer domain onto the unsigned domain, preserving the small-magnitude efficiency.*

---

## 1. Varint Encoding (Variable-Length Integers)

### The Problem

Fixed-width integer encodings waste space: a 64-bit integer uses 8 bytes even for the value 1. How can we encode integers using space proportional to their magnitude while maintaining unambiguous decoding?

### The Formula

A varint uses 7 bits of payload per byte, with the MSB (most significant bit) as a continuation flag:

$$\text{bytes}(n) = \left\lceil \frac{\lfloor \log_2(n) \rfloor + 1}{7} \right\rceil \quad \text{for } n > 0, \quad \text{bytes}(0) = 1$$

The maximum value encodable in $b$ bytes:

$$n_{\max}(b) = 2^{7b} - 1$$

Encoding efficiency compared to fixed-width:

$$\eta(n) = \frac{\lceil \log_2(n+1) \rceil}{8 \cdot \text{bytes}(n)} = \frac{\text{information bits}}{\text{encoded bits}}$$

### Worked Examples

**Encoding the value 300:**

$$300 = 256 + 32 + 8 + 4 = \text{0b}100101100$$

9 bits of information, so $\text{bytes}(300) = \lceil 9/7 \rceil = 2$ bytes.

Byte 1: `10101100` (continuation=1, payload=0101100)
Byte 2: `00000010` (continuation=0, payload=0000010)

Decoded: $0000010 \mathbin\| 0101100 = 100101100_2 = 300$

**Encoding efficiency table:**

| Value | Info bits | Varint bytes | Efficiency $\eta$ |
|-------|-----------|-------------|-------------------|
| 1     | 1         | 1           | $1/8 = 12.5\%$   |
| 127   | 7         | 1           | $7/8 = 87.5\%$   |
| 128   | 8         | 2           | $8/16 = 50.0\%$  |
| 16383 | 14        | 2           | $14/16 = 87.5\%$ |
| $2^{63}$ | 63     | 9           | $63/72 = 87.5\%$ |

The overhead ratio per byte is exactly $1/7 \approx 14.3\%$, asymptotically approaching $87.5\%$ efficiency at each boundary.

---

## 2. ZigZag Encoding (Signed Integer Bijection)

### The Problem

Standard varint encoding of negative 32-bit integers always uses 10 bytes (because protobuf sign-extends to 64 bits). ZigZag encoding maps signed integers to unsigned integers so that small-magnitude values (positive or negative) use few bytes.

### The Formula

ZigZag encoding for a signed $n$-bit integer $x$:

$$\text{ZigZag}(x) = (x \ll 1) \oplus (x \gg (n-1))$$

Where $\ll$ is left shift, $\gg$ is arithmetic right shift, and $\oplus$ is XOR.

The inverse (decoding):

$$\text{ZigZag}^{-1}(z) = (z \ggg 1) \oplus -(z \mathbin{\&} 1)$$

Where $\ggg$ is unsigned (logical) right shift.

The mapping is a bijection $\mathbb{Z} \to \mathbb{N}_0$:

$$0 \mapsto 0, \quad -1 \mapsto 1, \quad 1 \mapsto 2, \quad -2 \mapsto 3, \quad 2 \mapsto 4, \ldots$$

$$\text{ZigZag}(x) = \begin{cases} 2x & \text{if } x \geq 0 \\ -2x - 1 & \text{if } x < 0 \end{cases}$$

### Worked Examples

**Encoding $x = -3$ as sint32:**

$$\text{ZigZag}(-3) = (-3 \ll 1) \oplus (-3 \gg 31)$$
$$= -6 \oplus -1 = \text{0xFFFFFFFA} \oplus \text{0xFFFFFFFF} = 5$$

Varint encoding of 5: single byte `00000101`. Without ZigZag, $-3$ as int32 would use 10 bytes.

**Space savings for small negative values:**

| Value | `int32` bytes | `sint32` bytes | Savings |
|-------|--------------|----------------|---------|
| -1    | 10           | 1              | 90%     |
| -64   | 10           | 1              | 90%     |
| -128  | 10           | 2              | 80%     |
| -1000 | 10           | 2              | 80%     |

---

## 3. Tag-Length-Value Encoding (Field Identification)

### The Problem

Each field in a protobuf message is prefixed with a tag that encodes both the field number and wire type. How much overhead does this tagging add, and how does it enable forward/backward compatibility?

### The Formula

A tag is a varint encoding of:

$$\text{tag} = (\text{field\_number} \ll 3) \mathbin{|} \text{wire\_type}$$

Since wire type uses 3 bits, field numbers 1-15 fit in a single-byte tag ($4 + 3 = 7$ bits, fits in one varint byte):

$$\text{tag\_bytes}(f) = \left\lceil \frac{\lfloor \log_2(f) \rfloor + 4}{7} \right\rceil$$

Total overhead per field:

$$O(f, w) = \text{tag\_bytes}(f) + \begin{cases} 0 & \text{varint (wire type 0)} \\ 0 & \text{fixed (wire type 1, 5)} \\ \text{varint\_bytes}(\text{len}) & \text{length-delimited (wire type 2)} \end{cases}$$

### Worked Examples

**Field 1, string "hello" (5 bytes):**

Tag: $(1 \ll 3) | 2 = 10 = $ varint `0x0A` (1 byte)
Length: 5 = varint `0x05` (1 byte)
Payload: "hello" (5 bytes)
Total: 7 bytes. Overhead: $2/7 = 28.6\%$

**Field 200, int32 value 42:**

Tag: $(200 \ll 3) | 0 = 1600$, varint bytes: $\lceil 11/7 \rceil = 2$ bytes
Value: 42, varint: 1 byte
Total: 3 bytes. Overhead: $2/3 = 66.7\%$

This demonstrates why field numbers 1-15 matter for frequently used fields.

---

## 4. Compression Ratio (Protobuf vs JSON)

### The Problem

How much smaller is protobuf encoding compared to JSON for typical structured data?

### The Formula

JSON encoding of a field with key $k$ and value $v$:

$$B_{\text{JSON}}(k, v) = |k| + |\text{str}(v)| + C_{\text{syntax}}$$

Where $C_{\text{syntax}} \geq 4$ bytes (quotes, colon, comma/bracket).

Protobuf encoding:

$$B_{\text{proto}}(f, v) = \text{tag\_bytes}(f) + \text{value\_bytes}(v)$$

Compression ratio:

$$R = \frac{B_{\text{JSON}}}{B_{\text{proto}}}$$

For a typical message with $n$ fields:

$$R_{\text{avg}} = \frac{\sum_{i=1}^{n} (|k_i| + |\text{str}(v_i)| + 4)}{\sum_{i=1}^{n} (\text{tag}(f_i) + \text{val}(v_i))}$$

### Worked Examples

**User message: `{name: "Alice", age: 30, active: true}`**

JSON: `{"name":"Alice","age":30,"active":true}` = 39 bytes

Protobuf:
- Field 1 (string "Alice"): tag(1) + len(5) + "Alice" = 1 + 1 + 5 = 7
- Field 2 (int32 30): tag(1) + varint(30) = 1 + 1 = 2
- Field 3 (bool true): tag(1) + varint(1) = 1 + 1 = 2

Total: 11 bytes.

$$R = \frac{39}{11} = 3.5\times \text{ smaller}$$

**Large message with 20 fields, 500-byte JSON:**

Typical protobuf: 120-180 bytes. $R \approx 2.8$-$4.2\times$.

---

## 5. Schema Evolution (Compatibility Algebra)

### The Problem

Protobuf supports forward and backward compatibility. What operations are safe, and what invariants must hold for compatibility?

### The Formula

Define compatibility as a partial order on schemas. Schema $S_2$ is backward-compatible with $S_1$ ($S_1 \preceq S_2$) iff:

$$\forall m \in \text{Messages}(S_1): \text{decode}_{S_2}(\text{encode}_{S_1}(m)) \text{ succeeds}$$

Safe evolution operations (preserving backward compatibility):

$$\text{AddField}(f_{\text{new}}) : S \to S' \quad \text{iff } f_{\text{new}} \notin \text{fields}(S) \text{ and } f_{\text{new}} \text{ has default}$$

$$\text{RemoveField}(f) : S \to S' \quad \text{iff } f \text{ was not } \texttt{required} \text{ (proto2)}$$

$$\text{RenameField}(f, f') : S \to S' \quad \text{always safe (field number unchanged)}$$

Unsafe operations (breaking):

$$\text{ChangeType}(f, T \to T') : \text{breaks iff wire type changes}$$

$$\text{ReuseFieldNumber}(f_{\text{deleted}}, f_{\text{new}}) : \text{always breaks}$$

### Worked Examples

**Safe evolution:** Adding field `phone = 8` to User does not break old readers — they skip unknown field 8.

**Breaking change:** Changing field 4 from `int32 age` to `string age` changes wire type 0 (varint) to wire type 2 (length-delimited) — old decoders will corrupt data.

**Safe type change:** `int32` to `int64` — both use wire type 0 (varint), values are upcast safely.

---

## Prerequisites

- information-theory, binary-encoding, bijections, grpc, json, avro
