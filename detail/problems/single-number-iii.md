# The Mathematics of Single Number III -- Bit Manipulation and Group Partitioning

> *XOR is the perfect detector of uniqueness: it accumulates differences without remembering values, and a single distinguishing bit is enough to split two unknowns into separate solvable universes.*

---

## 1. XOR as a Self-Inverse Group Operation (Abstract Algebra)

### The Problem

Why does XOR cancel paired elements? Formalize the algebraic structure.

### The Formula

The integers under XOR form an abelian group $(\mathbb{Z}, \oplus)$ where:

- **Identity:** $a \oplus 0 = a$
- **Self-inverse:** $a \oplus a = 0$
- **Associativity:** $(a \oplus b) \oplus c = a \oplus (b \oplus c)$
- **Commutativity:** $a \oplus b = b \oplus a$

For an array with elements $\{p_1, p_1, p_2, p_2, \ldots, a, b\}$:

$$\bigoplus_i \text{nums}[i] = (p_1 \oplus p_1) \oplus (p_2 \oplus p_2) \oplus \cdots \oplus a \oplus b = 0 \oplus 0 \oplus \cdots \oplus a \oplus b = a \oplus b$$

### Worked Examples

Array $[1, 2, 1, 3, 2, 5]$:

$$1 \oplus 2 \oplus 1 \oplus 3 \oplus 2 \oplus 5 = (1 \oplus 1) \oplus (2 \oplus 2) \oplus 3 \oplus 5 = 0 \oplus 0 \oplus 3 \oplus 5 = 6$$

Binary: $3 = 011$, $5 = 101$, so $3 \oplus 5 = 110 = 6$. Correct.

---

## 2. Isolating the Lowest Set Bit (Number Theory / Bit Arithmetic)

### The Problem

Given $x = a \oplus b$, extract a single bit position where $a$ and $b$ differ.

### The Formula

In two's complement representation, the lowest set bit of $x$ is:

$$\text{lsb}(x) = x \mathbin{\&} (-x)$$

This works because $-x = \sim x + 1$. Flipping all bits and adding 1 propagates a carry
through the trailing zeros, setting the lowest set bit position and clearing all others.

For any $x \ne 0$, $\text{lsb}(x)$ is a power of 2: $\text{lsb}(x) = 2^k$ where $k$ is the position of the lowest set bit.

### Worked Examples

$x = 6 = 110_2$:

$$-6 = \sim(110) + 1 = 001 + 1 = 010$$
$$6 \mathbin{\&} (-6) = 110 \mathbin{\&} 010 = 010 = 2$$

So bit position 1 is where $a = 3$ and $b = 5$ differ. Indeed: $3 = 0\mathbf{1}1$, $5 = 1\mathbf{0}1$ -- bit 1 differs.

---

## 3. Partition by a Distinguishing Bit (Set Theory)

### The Problem

Prove that partitioning by the distinguishing bit separates $a$ and $b$ while keeping
each pair together.

### The Formula

Let $d = \text{lsb}(a \oplus b)$. Define two groups:

$$G_1 = \{n \in \text{nums} \mid n \mathbin{\&} d \ne 0\}$$
$$G_0 = \{n \in \text{nums} \mid n \mathbin{\&} d = 0\}$$

**Claim 1:** $a \in G_1$ and $b \in G_0$ (or vice versa).

*Proof:* Since $d$ is a bit where $a$ and $b$ differ, exactly one of $a \mathbin{\&} d$ and $b \mathbin{\&} d$ is nonzero.

**Claim 2:** For every paired element $p$, both copies are in the same group.

*Proof:* Both copies have the same value, so $p \mathbin{\&} d$ is the same for both. They land in the same group.

**Result:** $\bigoplus_{n \in G_1} n = a$ (or $b$), and $\bigoplus_{n \in G_0} n = b$ (or $a$), since paired elements cancel within each group.

### Worked Examples

Array $[1, 2, 1, 3, 2, 5]$, $d = 2 = 010_2$:

| Element | Binary | Bit 1 set? | Group |
|:---:|:---:|:---:|:---:|
| 1 | 001 | No | $G_0$ |
| 2 | 010 | Yes | $G_1$ |
| 1 | 001 | No | $G_0$ |
| 3 | 011 | Yes | $G_1$ |
| 2 | 010 | Yes | $G_1$ |
| 5 | 101 | No | $G_0$ |

$G_0 = \{1, 1, 5\}$: $1 \oplus 1 \oplus 5 = 5$

$G_1 = \{2, 3, 2\}$: $2 \oplus 3 \oplus 2 = 3$

Result: $[3, 5]$.

---

## 4. Two's Complement and Edge Cases (Computer Architecture)

### The Problem

What happens when the XOR result is the minimum representable integer (e.g., `i32::MIN`)?

### The Formula

For a $w$-bit two's complement integer, the range is $[-2^{w-1}, 2^{w-1}-1]$.

The value $-2^{w-1}$ (e.g., $-2^{31}$ for `i32`) has a special property:

$$-(-2^{w-1}) = 2^{w-1}$$

This overflows -- $2^{w-1}$ is not representable. In wrapping arithmetic:

$$\text{wrapping\_neg}(-2^{w-1}) = -2^{w-1}$$

So $x \mathbin{\&} \text{wrapping\_neg}(x) = x \mathbin{\&} x = x = -2^{w-1} = 2^{w-1}$.

This is still a valid power-of-two mask (the sign bit), so the algorithm works correctly.

### Worked Examples

Rust: `i32::MIN = -2147483648 = 0x80000000`

```
wrapping_neg(0x80000000) = 0x80000000
0x80000000 & 0x80000000 = 0x80000000  (bit 31)
```

Partition by bit 31 = partition by sign. Negative numbers go to $G_1$, non-negative to $G_0$.

---

## 5. Generalization to k Unique Elements (Information Theory)

### The Problem

Can this approach extend to finding $k > 2$ unique elements?

### The Formula

The XOR approach gives us $a \oplus b$ (1 equation, 2 unknowns). With 1 distinguishing
bit, we split into 2 groups of 1 unknown each -- solvable.

For $k = 3$ unique elements, XOR gives $a \oplus b \oplus c$. We cannot split 3 unknowns
with a single bit partition (one group gets 2 unknowns). Additional algebraic structure
is needed.

**General approaches for $k$ unique elements:**

- Counting sort in $O(n + R)$ time, $O(R)$ space (R = value range)
- Hash map in $O(n)$ time, $O(n)$ space
- For $k = 3$: use XOR and sum equations together (2 equations, 3 unknowns -- still hard)

The bit manipulation approach is uniquely elegant for $k \in \{1, 2\}$.

### Worked Examples

$k = 1$ (Single Number I): XOR all $\Rightarrow$ answer directly.

$k = 2$ (Single Number III): XOR all $\Rightarrow$ $a \oplus b$ $\Rightarrow$ partition $\Rightarrow$ two subproblems of $k=1$.

$k = 3$: No known $O(1)$ space, $O(n)$ time bit manipulation solution exists in the general case.

---

## Prerequisites

- XOR properties (commutativity, associativity, self-inverse)
- Two's complement integer representation
- Bitwise AND, OR, NOT operations
- Basic set theory (partitioning)

## Complexity

| Level | Description |
|-------|-------------|
| **Beginner** | Solve Single Number I (k=1) using XOR. Understand why pairs cancel. Verify with small examples by hand. |
| **Intermediate** | Implement the full 3-step algorithm. Understand why `x & -x` isolates the lowest set bit. Handle negative numbers and the `wrapping_neg` edge case in Rust. |
| **Advanced** | Prove correctness of the partition step formally. Analyze why the approach fails for k=3. Study the relationship between XOR and GF(2) linear algebra. Implement constant-space solutions for Single Number II (k=1, others appear 3 times). |
