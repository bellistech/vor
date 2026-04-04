# The Mathematics of D-Bus — Message Routing and Access Control Lattices

> *D-Bus is a message-oriented middleware where routing follows a publish-subscribe model for signals and a point-to-point model for method calls. The access control system forms a security lattice, and message serialization follows a type algebra with alignment constraints.*

---

## 1. Message Routing (Graph Theory)

### The Problem

The D-Bus daemon routes messages between connected clients. Method calls are point-to-point (unicast), while signals are multicast to all subscribers matching a set of rules.

### The Formula

For unicast method calls, routing is a function:

$$\text{route}: (sender, destination, path, interface, method) \to \text{connection}$$

For signals with $n$ connected clients and $k$ match rules per client, the signal fan-out for signal $s$:

$$F(s) = |\{c \in C \mid \exists r \in R_c : \text{match}(r, s) = \text{true}\}|$$

Total message throughput:

$$\Theta = \lambda_{\text{call}} + \lambda_{\text{signal}} \cdot \bar{F}$$

where $\bar{F}$ is the average fan-out factor.

### Worked Examples

50 connected clients, signal rate $\lambda_{\text{signal}} = 100$/s, average fan-out $\bar{F} = 5$, method call rate $\lambda_{\text{call}} = 200$/s:

$$\Theta = 200 + 100 \times 5 = 700 \text{ messages/s delivered}$$

With match rule filtering reducing fan-out to $\bar{F} = 2$:

$$\Theta = 200 + 100 \times 2 = 400 \text{ messages/s}$$

---

## 2. Type System Algebra (Type Theory)

### The Problem

D-Bus messages carry typed data using a compact type signature system. The type system is an algebraic data type with fixed-size primitives, arrays, structs, and dictionaries.

### The Formula

The D-Bus type universe $\mathcal{T}$:

$$\mathcal{T} = \mathcal{B} \cup \text{Array}(\mathcal{T}) \cup \text{Struct}(\mathcal{T}^*) \cup \text{Dict}(\mathcal{B}_s, \mathcal{T}) \cup \text{Variant}$$

where $\mathcal{B} = \{y, b, n, q, i, u, x, t, d, s, o, g, h\}$ are basic types.

The size of a serialized value $v$ of type $\tau$:

$$\text{size}(\tau, v) = \text{align}(\tau) + \begin{cases}
|\tau|_{\text{fixed}} & \text{if } \tau \in \mathcal{B}_{\text{fixed}} \\
4 + |v| & \text{if } \tau = s \text{ (string: length prefix + data + NUL)} \\
4 + \sum_{e \in v} \text{size}(\tau_e, e) & \text{if } \tau = \text{Array}(\tau_e)
\end{cases}$$

Alignment rules:

$$\text{align}(\tau) = \begin{cases}
1 & \tau \in \{y, g, v\} \\
2 & \tau \in \{n, q\} \\
4 & \tau \in \{b, i, u, s, o, a\} \\
8 & \tau \in \{x, t, d, \text{struct}, \text{dict}\}
\end{cases}$$

### Worked Examples

Signature `(siu)`: struct of (string, int32, uint32).

For value `("hello", 42, 7)`:

| Field | Align | Size | Running Total |
|:---|:---:|:---:|:---:|
| struct start | 8-byte | 0 | 0 (padding to 8) |
| string "hello" | 4-byte | 4 + 5 + 1 = 10 | 10 |
| padding | - | 2 | 12 |
| int32 42 | 4-byte | 4 | 16 |
| uint32 7 | 4-byte | 4 | 20 |

Total: 20 bytes on the wire.

---

## 3. Access Control Lattice (Lattice Theory)

### The Problem

D-Bus policy rules form a layered access control system. Rules are evaluated in priority order: default context, group policies, user policies, and mandatory policies.

### The Formula

Policy evaluation for action $a$ by user $u$:

$$\text{decision}(a, u) = \text{last\_match}(P_{\text{mandatory}} \cup P_u \cup P_{G(u)} \cup P_{\text{default}})$$

The security lattice ordering:

$$P_{\text{default}} \sqsubseteq P_{G(u)} \sqsubseteq P_u \sqsubseteq P_{\text{mandatory}}$$

For $n$ rules, the effective permission:

$$\text{perm}(a, u) = \bigsqcup_{i=1}^{n} \begin{cases}
\text{allow} & \text{if rule}_i \text{ matches and allows} \\
\text{deny} & \text{if rule}_i \text{ matches and denies} \\
\bot & \text{if rule}_i \text{ does not match}
\end{cases}$$

where later matching rules override earlier ones within the same policy level.

### Worked Examples

User `alice` in group `users`, action: send to `com.example.Admin`:

1. Default policy: `<deny send_destination="com.example.Admin"/>` -> deny
2. Group `users` policy: (no matching rule) -> $\bot$
3. User `alice` policy: `<allow send_destination="com.example.Admin"/>` -> allow

Result: allow (user policy overrides default).

---

## 4. Name Ownership (Queueing Theory)

### The Problem

Well-known bus names are owned by connections. When the owner disconnects, ownership passes to the next in a FIFO queue of waiters.

### The Formula

For well-known name $N$ with owner queue $Q_N = [c_1, c_2, \ldots, c_m]$:

$$\text{owner}(N) = Q_N[0]$$

On disconnect of $c_i$:

$$Q_N' = Q_N \setminus \{c_i\}$$

$$\text{owner}'(N) = \begin{cases}
Q_N'[0] & \text{if } |Q_N'| > 0 \\
\emptyset & \text{otherwise}
\end{cases}$$

The NameOwnerChanged signal fires when:

$$\text{owner}(N) \neq \text{owner}'(N)$$

### Worked Examples

Queue for `org.freedesktop.NetworkManager`: $[\text{:1.45}, \text{:1.102}]$.

Connection `:1.45` disconnects:

$$Q' = [\text{:1.102}], \quad \text{owner}' = \text{:1.102}$$

Signal emitted: `NameOwnerChanged("org.freedesktop.NetworkManager", ":1.45", ":1.102")`

---

## 5. Match Rule Filtering (Predicate Logic)

### The Problem

Clients register match rules to receive signals. The daemon evaluates each signal against all registered match rules to determine delivery targets.

### The Formula

A match rule $r$ is a conjunction of predicates:

$$\text{match}(r, s) = \bigwedge_{f \in \text{fields}(r)} (s.f = r.f)$$

Fields: $\{\text{type}, \text{sender}, \text{interface}, \text{member}, \text{path}, \text{path\_namespace}, \text{destination}, \text{arg}N\}$.

Selectivity of a match rule:

$$\sigma(r) = \prod_{f \in \text{fields}(r)} P(s.f = r.f)$$

Expected messages matched per second:

$$\lambda_{\text{matched}} = \lambda_{\text{total}} \cdot \sigma(r)$$

### Worked Examples

Total signal rate: 1000/s. Match rule: `type=signal, interface=org.freedesktop.login1.Manager`.

If 5% of signals are from that interface:

$$\sigma = 1.0 \times 0.05 = 0.05$$

$$\lambda_{\text{matched}} = 1000 \times 0.05 = 50 \text{ signals/s}$$

Adding `member=SessionNew` (20% of interface signals):

$$\sigma = 1.0 \times 0.05 \times 0.20 = 0.01$$

$$\lambda_{\text{matched}} = 1000 \times 0.01 = 10 \text{ signals/s}$$

---

## 6. Message Serialization Efficiency (Information Theory)

### The Problem

D-Bus wire protocol pads data for alignment, creating overhead. The serialization efficiency depends on the type signature and data distribution.

### The Formula

Serialization efficiency:

$$\eta = \frac{\text{payload\_bytes}}{\text{wire\_bytes}} = \frac{\sum_{i} |v_i|}{\sum_{i} (|v_i| + \text{pad}_i) + H}$$

where $H$ is the fixed header size (16 bytes minimum) and $\text{pad}_i$ is alignment padding.

Worst-case padding for type $\tau$ with alignment $a$:

$$\text{pad}_{\max}(\tau) = a(\tau) - 1$$

### Worked Examples

Message with signature `yxyx` (byte, int64, byte, int64):

| Field | Data | Padding | Wire Bytes |
|:---|:---:|:---:|:---:|
| byte | 1 | 7 (align to 8) | 8 |
| int64 | 8 | 0 | 8 |
| byte | 1 | 7 (align to 8) | 8 |
| int64 | 8 | 0 | 8 |

Payload: 18 bytes. Wire: 32 bytes + 16 header = 48 bytes.

$$\eta = \frac{18}{48} = 0.375 = 37.5\%$$

Reordering to `yyxx` (pack bytes together):

Wire: 2 + 6 padding + 8 + 8 = 24 + 16 header = 40 bytes.

$$\eta = \frac{18}{40} = 0.45 = 45\%$$

---

## Prerequisites

- graph-theory, type-theory, lattice-theory, queueing-theory, predicate-logic, information-theory
