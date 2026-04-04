# The Mathematics of Avro — Schema Resolution and Encoding Efficiency

> *Avro's schema resolution algorithm performs a deterministic mapping between writer and reader schemas, enabling backward and forward compatibility. The resolution rules form a lattice of type promotions and field matching that can be analyzed as a partial order. Avro's binary encoding achieves near-optimal density by omitting field tags entirely — the schema itself provides the decoding template.*

---

## 1. Binary Encoding Efficiency (Tag-Free Serialization)

### The Problem

Unlike Protobuf (which tags every field) or JSON (which embeds field names), Avro encodes values in schema-defined order with no field markers. How much space does this save, and what trade-offs does it introduce?

### The Formula

Avro encoding size for a record with $n$ fields:

$$B_{\text{Avro}} = \sum_{i=1}^{n} \text{enc}(v_i)$$

No tags, no field names, no delimiters. Compare with Protobuf:

$$B_{\text{Proto}} = \sum_{i=1}^{n} (\text{tag}(f_i) + \text{enc}(v_i))$$

And JSON:

$$B_{\text{JSON}} = \sum_{i=1}^{n} (|k_i| + |\text{str}(v_i)| + C_i) + C_{\text{delim}}$$

Avro's overhead relative to raw data:

$$O_{\text{Avro}} = \frac{B_{\text{Avro}} - \sum |v_i|_{\text{raw}}}{\sum |v_i|_{\text{raw}}}$$

For varints, the overhead is the continuation bits ($1/7$ per byte). For strings: length prefix only.

### Worked Examples

**Record: {id: 42, name: "Alice", active: true}**

Avro encoding:
- `id` (long): varint(42) = 1 byte (ZigZag: $42 \to 84$, varint: 1 byte)
- `name` (string): varint(5) + "Alice" = 1 + 5 = 6 bytes
- `active` (boolean): 1 byte

$$B_{\text{Avro}} = 1 + 6 + 1 = 8 \text{ bytes}$$

Protobuf equivalent:
- Field 1 tag + varint(42) = 1 + 1 = 2 bytes
- Field 2 tag + len + "Alice" = 1 + 1 + 5 = 7 bytes
- Field 3 tag + varint(1) = 1 + 1 = 2 bytes

$$B_{\text{Proto}} = 2 + 7 + 2 = 11 \text{ bytes}$$

JSON: `{"id":42,"name":"Alice","active":true}` = 38 bytes

$$\text{Avro:Proto:JSON} = 8 : 11 : 38 = 1.0 : 1.375 : 4.75$$

Avro is 27% smaller than Protobuf and 79% smaller than JSON for this record.

---

## 2. Schema Resolution (Lattice Theory)

### The Problem

When a reader has a different schema version than the writer, Avro must resolve fields between the two schemas. What mathematical structure governs which resolutions are valid?

### The Formula

Schema resolution defines a partial order $\preceq$ on the set of schemas $\mathcal{S}$. Schema $S_w$ (writer) is resolvable to $S_r$ (reader) iff a resolution function $\rho$ exists:

$$\rho : S_w \times S_r \to \text{Decoder}$$

Type promotion rules form a directed acyclic graph:

$$\text{int} \preceq \text{long} \preceq \text{float} \preceq \text{double}$$
$$\text{string} \preceq \text{bytes}$$
$$\text{bytes} \preceq \text{string}$$

For records, field resolution follows:

$$\rho_{\text{field}}(f_w, f_r) = \begin{cases}
\text{match} & \text{if } f_w.\text{name} = f_r.\text{name} \land f_w.\text{type} \preceq f_r.\text{type} \\
\text{default} & \text{if } f_w \notin S_r \land f_r.\text{default} \neq \bot \\
\text{skip} & \text{if } f_w \notin S_r \land f_r.\text{default} = \bot \\
\text{fail} & \text{if } f_w.\text{type} \not\preceq f_r.\text{type}
\end{cases}$$

The number of valid resolution paths between two schemas with $n_w$ writer fields and $n_r$ reader fields:

$$|\text{resolutions}| = \sum_{k=0}^{\min(n_w, n_r)} \binom{n_w}{k} \binom{n_r}{k} k! \cdot P(k)$$

Where $P(k)$ is the probability that $k$ matched fields have promotable types.

### Worked Examples

**Writer schema V1: {id: long, name: string}**
**Reader schema V2: {id: long, name: string, email: string, default: ""}**

Resolution:
1. `id`: matched, `long = long` (exact match)
2. `name`: matched, `string = string` (exact match)
3. `email`: writer field missing, reader has default `""` -- use default

Result: Valid resolution. Reader fills `email` with `""`.

**Type promotion: Writer has `int`, reader has `long`**

Since $\text{int} \preceq \text{long}$, the 32-bit value is widened to 64-bit: $v_r = \text{sign\_extend}_{64}(v_w)$.

---

## 3. Container File Structure (Block Compression Model)

### The Problem

Avro container files (.avro) group records into blocks, each independently compressed. How does block size affect compression ratio and random access granularity?

### The Formula

An Avro file consists of a header (magic + schema + sync marker) followed by $B$ data blocks:

$$\text{File} = \text{Header} + \sum_{b=1}^{B} \text{Block}_b$$

Each block stores $n_b$ records with total uncompressed size $U_b$ and compressed size $C_b$:

$$\text{Block}_b = \text{count}(n_b) + \text{size}(C_b) + \text{compressed\_data} + \text{sync\_marker}$$

Compression ratio of block $b$:

$$r_b = \frac{C_b}{U_b}$$

Overall file compression ratio:

$$r = \frac{\sum C_b + H}{\sum U_b + H} \approx \frac{\sum C_b}{\sum U_b} \quad \text{(for large files)}$$

Random access granularity is one block. To read record $i$, we must decompress the entire block containing it. Expected decompression waste:

$$W_{\text{avg}} = \frac{n_b - 1}{2} \cdot \bar{s}$$

Where $\bar{s}$ is the average record size.

### Worked Examples

**1M records, average 200 bytes each, Snappy compression (ratio 0.4), 64KB block target:**

Records per block: $n_b = \lfloor 65{,}536 / 200 \rfloor = 327$

Number of blocks: $B = \lceil 1{,}000{,}000 / 327 \rceil = 3{,}059$

Uncompressed: $200 \times 10^6 = 200$ MB
Compressed: $200 \times 0.4 = 80$ MB

Random access waste per read: $\frac{326}{2} \times 200 = 32{,}600$ bytes $\approx 32$ KB (half a block on average).

---

## 4. Schema Compatibility Algebra (Formal Verification)

### The Problem

Given compatibility level $\ell$ (BACKWARD, FORWARD, FULL) and a sequence of schema versions $S_1, S_2, \ldots, S_n$, what invariants must hold for the sequence to be valid?

### The Formula

Define the compatibility relation $C_\ell$ between schemas:

$$C_{\text{BACKWARD}}(S_{\text{new}}, S_{\text{old}}) \iff \rho(S_{\text{old}}, S_{\text{new}}) \text{ succeeds}$$
$$C_{\text{FORWARD}}(S_{\text{new}}, S_{\text{old}}) \iff \rho(S_{\text{new}}, S_{\text{old}}) \text{ succeeds}$$
$$C_{\text{FULL}}(S_{\text{new}}, S_{\text{old}}) \iff C_{\text{BACKWARD}} \land C_{\text{FORWARD}}$$

Transitive variants require compatibility with all prior versions:

$$C_{\text{FULL\_TRANS}}(S_n) \iff \forall i < n: C_{\text{FULL}}(S_n, S_i)$$

The set of allowed operations under each level:

$$\text{BACKWARD}: \{\text{add\_field\_with\_default}, \text{remove\_field}, \text{promote\_type}\}$$
$$\text{FORWARD}: \{\text{add\_field}, \text{remove\_field\_with\_default}, \text{promote\_type}\}$$
$$\text{FULL}: \{\text{add\_field\_with\_default}, \text{remove\_field\_with\_default}, \text{promote\_type}\}$$

### Worked Examples

**Three-version evolution under FULL_TRANSITIVE:**

$S_1$: `{a: int, b: string}`
$S_2$: `{a: int, b: string, c: long, default: 0}` -- add with default, FULL compatible with $S_1$
$S_3$: `{a: long, b: string, c: long, default: 0}` -- promote `a: int -> long`, FULL compatible with $S_1$ and $S_2$

All pairwise checks pass: $C_{\text{FULL}}(S_3, S_1)$, $C_{\text{FULL}}(S_3, S_2)$, $C_{\text{FULL}}(S_2, S_1)$.

---

## 5. Kafka Wire Format (Schema ID Overhead)

### The Problem

Confluent's Kafka serializer prepends a 5-byte header (magic byte + 4-byte schema ID) to every message. What is the overhead as a function of message size?

### The Formula

Wire format: $[0x00][\text{schema\_id (4 bytes)}][\text{Avro payload}]$

$$B_{\text{wire}} = 5 + B_{\text{Avro}}$$

Overhead fraction:

$$O = \frac{5}{5 + B_{\text{Avro}}}$$

For a Kafka topic with $N$ messages per second and average payload $\bar{B}$:

$$\text{Bandwidth}_{\text{overhead}} = 5N \text{ bytes/s}$$

### Worked Examples

**Small messages (IoT sensors), 20 bytes payload:**

$$O = \frac{5}{25} = 20\%$$

At 100K messages/s: $5 \times 100{,}000 = 500$ KB/s overhead.

**Large messages (events), 500 bytes payload:**

$$O = \frac{5}{505} = 0.99\%$$

Negligible. This shows why Avro's tag-free encoding matters most for small, high-volume messages where the 5-byte fixed overhead dominates.

**Break-even with JSON field names:** If JSON overhead averages 60% ($B_{\text{JSON}} = 1.6 \times B_{\text{raw}}$):

$$5 + B_{\text{Avro}} < B_{\text{JSON}} \implies B_{\text{raw}} > \frac{5}{0.6} = 8.3 \text{ bytes}$$

Avro wins for any payload larger than about 8 bytes.

---

## Prerequisites

- information-theory, binary-encoding, lattice-theory, compression, protobuf, json, parquet
