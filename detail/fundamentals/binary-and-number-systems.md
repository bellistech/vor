# Binary and Number Systems -- From Positional Notation to IEEE 754

> *Number systems are the atomic layer of computation. Every integer, address, pixel, and instruction reduces to positional encoding over finite alphabets. Understanding their algebraic structure -- from grade-school binary to IEEE 754 floating point -- is prerequisite to reasoning about overflow, precision, representation error, and the fundamental limits of digital arithmetic.*

---

## 1. Positional Number Systems: Formal Definition

### The Problem

Define the representation of numbers in arbitrary radix (base) and establish conversion algorithms between bases.

### The Formula

A number $N$ in base $b$ with digits $d_{n-1} d_{n-2} \ldots d_1 d_0 . d_{-1} d_{-2} \ldots d_{-m}$ has value:

$$N = \sum_{i=-m}^{n-1} d_i \cdot b^i$$

where each $d_i \in \{0, 1, \ldots, b-1\}$.

### Key Bases in Computing

```
Base  Name           Digits            Prefix     Bits/Digit
2     Binary         0-1               0b         1
8     Octal          0-7               0o         3
10    Decimal        0-9               (none)     ~3.322
16    Hexadecimal    0-9, A-F          0x         4
```

The relationship between bases that are powers of 2 is direct: each octal digit maps to exactly 3 binary digits, each hex digit to exactly 4. This is not coincidence -- it follows from $\log_2 b = k$ when $b = 2^k$, meaning $k$ binary digits encode one base-$b$ digit with no information loss.

### Conversion Algorithms

**Base-b to decimal:** Evaluate the polynomial (Horner's method for efficiency).

```
Horner's method for binary 11001000:
  ((((((1*2 + 1)*2 + 0)*2 + 0)*2 + 1)*2 + 0)*2 + 0)*2 + 0
  = 200

Complexity: O(n) multiplications and additions for n digits.
vs naive polynomial: O(n) multiplications but requires computing b^i.
```

**Decimal to base-b:** Repeated division (integer part) and repeated multiplication (fractional part).

```
Integer part (200 to binary):
  200 = 100*2 + 0   -> d0 = 0
  100 = 50*2  + 0   -> d1 = 0
  50  = 25*2  + 0   -> d2 = 0
  25  = 12*2  + 1   -> d3 = 1
  12  = 6*2   + 0   -> d4 = 0
  6   = 3*2   + 0   -> d5 = 0
  3   = 1*2   + 1   -> d6 = 1
  1   = 0*2   + 1   -> d7 = 1
  Result: 11001000

Fractional part (0.1 to binary):
  0.1 * 2 = 0.2  -> d-1 = 0
  0.2 * 2 = 0.4  -> d-2 = 0
  0.4 * 2 = 0.8  -> d-3 = 0
  0.8 * 2 = 1.6  -> d-4 = 1
  0.6 * 2 = 1.2  -> d-5 = 1
  0.2 * 2 = 0.4  -> d-6 = 0   (cycle repeats)
  Result: 0.000110011001100... (non-terminating)
```

### Termination Theorem

A fraction $p/q$ in lowest terms has a terminating representation in base $b$ if and only if every prime factor of $q$ is also a prime factor of $b$.

- Base 10: terminates iff $q = 2^a \cdot 5^b$ (hence 1/3 = 0.333... is non-terminating)
- Base 2: terminates iff $q = 2^a$ (hence 1/10 is non-terminating in binary, since 10 = 2 * 5)

This theorem is the root cause of floating-point representation error for decimal fractions.

---

## 2. Integer Representations

### Unsigned Integers

An $n$-bit unsigned integer represents values in $[0, 2^n - 1]$.

```
n bits -> 2^n distinct values

 8-bit:  0 to 255
16-bit:  0 to 65535
32-bit:  0 to 4294967295
64-bit:  0 to 18446744073709551615
```

### Signed Integer Representations

Four historical methods, with two's complement dominant:

**Sign-magnitude:** MSB is sign, remaining bits are magnitude.

```
+5 = 0 0000101
-5 = 1 0000101

Problems: two zeros (+0, -0), addition requires sign comparison.
Range: -(2^(n-1) - 1) to +(2^(n-1) - 1)
```

**One's complement:** Negate by flipping all bits.

```
+5 = 00000101
-5 = 11111010

Problems: two zeros (00000000, 11111111), end-around carry needed.
Range: -(2^(n-1) - 1) to +(2^(n-1) - 1)
Used in: Internet checksum (RFC 1071), historical machines (CDC 6600).
```

**Two's complement:** Negate by flipping all bits and adding 1.

```
+5  = 00000101
-5  = 11111011   (invert: 11111010, add 1: 11111011)

Algebraic definition: -x = 2^n - x

Properties:
  - Single zero representation
  - Addition works identically to unsigned (hardware reuse)
  - One extra negative value: -2^(n-1) has no positive counterpart
  - Range: -2^(n-1) to 2^(n-1) - 1

Why it works:
  x + (-x) = x + (2^n - x) = 2^n
  In n bits, 2^n overflows to 0. Therefore x + (-x) = 0.
```

**Excess-K (biased):** Store value + bias. Used in IEEE 754 exponents.

```
Excess-127 (8-bit):
  Stored value 0   -> actual = 0 - 127 = -127
  Stored value 127 -> actual = 127 - 127 = 0
  Stored value 254 -> actual = 254 - 127 = +127

Advantage: unsigned comparison of biased values gives correct
ordering of actual values (important for hardware float comparison).
```

### Overflow Detection

```
Unsigned overflow: carry out of MSB
  255 + 1 = 0 (with carry)
  Test: result < operand

Signed (two's complement) overflow: when sign of result is wrong
  127 + 1 = -128 (overflow!)
  Test: both operands same sign, result different sign
  Hardware: XOR of carry into MSB with carry out of MSB
```

---

## 3. Bitwise Operations as Algebraic Structure

### Boolean Algebra over Bit Vectors

The set $\{0, 1\}^n$ with AND, OR, NOT forms a Boolean algebra. XOR with AND forms a Boolean ring (field $GF(2^n)$).

```
AND (conjunction, intersection):
  x & y: bit is 1 iff both bits are 1
  Identity: x & 0xFFFF...F = x
  Annihilator: x & 0 = 0
  Idempotent: x & x = x

OR (disjunction, union):
  x | y: bit is 1 iff at least one bit is 1
  Identity: x | 0 = x
  Annihilator: x | 0xFFFF...F = 0xFFFF...F
  Idempotent: x | x = x

XOR (symmetric difference, addition mod 2):
  x ^ y: bit is 1 iff bits differ
  Identity: x ^ 0 = x
  Self-inverse: x ^ x = 0
  Associative and commutative (abelian group)
  x ^ y ^ y = x  (basis of XOR swap, simple ciphers, RAID parity)

NOT (complement):
  ~x: flip every bit
  Involution: ~~x = x
  De Morgan: ~(x & y) = ~x | ~y
             ~(x | y) = ~x & ~y

Shifts:
  x << k: multiply by 2^k (unsigned), fill low bits with 0
  x >> k: divide by 2^k
    Logical shift: fill high bits with 0 (unsigned)
    Arithmetic shift: fill high bits with sign bit (signed)
```

### Bit Manipulation Identities

```
Isolate lowest set bit:       x & (-x)
Clear lowest set bit:         x & (x - 1)
Set all bits below lowest:    x | (x - 1)
Population count (popcount):  number of 1-bits (hardware: POPCNT)
Leading zeros (clz):          position of highest set bit
Trailing zeros (ctz):         position of lowest set bit

Power-of-2 test: x > 0 && (x & (x-1)) == 0
Next power of 2:  round up via bit manipulation:
  v--; v|=v>>1; v|=v>>2; v|=v>>4; v|=v>>8; v|=v>>16; v++

Kernighan's popcount:
  count = 0
  while x != 0:
    x &= x - 1    // clear lowest set bit
    count++
  // Runs in O(popcount) iterations, not O(n)
```

---

## 4. IEEE 754 Floating-Point Arithmetic

### The Problem

Represent real numbers in a fixed number of bits with controlled precision and well-defined behavior for exceptional cases (overflow, underflow, division by zero, invalid operations).

### The Standard

IEEE 754 (first published 1985, revised 2008 and 2019) defines:

```
Format          Bits  Sign  Exponent  Mantissa  Bias   Decimal Digits
binary16        16    1     5         10        15     ~3.3
binary32        32    1     8         23        127    ~7.2
binary64        64    1     11        52        1023   ~15.9
binary128       128   1     15        112       16383  ~34.0
binary256       256   1     19        236       262143 ~71.3
```

### Encoding

A floating-point number is encoded as three fields:

```
(-1)^s * m * 2^e

where:
  s = sign bit (0 = positive, 1 = negative)
  e = exponent (biased: stored_exponent - bias)
  m = significand (mantissa) with implicit leading 1 for normal numbers
```

**Normal numbers:** Exponent field is neither all-zeros nor all-ones.

```
Value = (-1)^s * 1.fraction * 2^(exponent - bias)

The leading "1." is implicit (hidden bit), giving one extra bit of precision.

Example: binary32 encoding of -13.5
  13.5 = 1101.1 binary = 1.1011 * 2^3
  Sign: 1 (negative)
  Exponent: 3 + 127 = 130 = 10000010
  Fraction: 10110000000000000000000 (23 bits)

  Encoding: 1 10000010 10110000000000000000000
  Hex: 0xC1580000
```

### Denormalized (Subnormal) Numbers

When the exponent field is all zeros, the implicit leading bit is 0 instead of 1, and the exponent is fixed at $1 - \text{bias}$:

```
Value = (-1)^s * 0.fraction * 2^(1 - bias)

Purpose: gradual underflow -- fills the gap between 0 and the
smallest normal number with evenly spaced values.

Without denormals:
  smallest positive normal (binary32) = 1.0 * 2^(-126) ~ 1.18e-38
  next smaller value: 0 (abrupt gap!)

With denormals:
  smallest denormal = 0.00000000000000000000001 * 2^(-126)
                    = 2^(-23) * 2^(-126) = 2^(-149) ~ 1.4e-45
  spacing between denormals = 2^(-149)

Property: |a - b| < smallest_denormal implies a == b
(no two distinct floats are closer than the smallest denormal)
```

### Special Values

```
+0:    s=0, exponent=0, fraction=0
-0:    s=1, exponent=0, fraction=0
  +0 == -0 is true (IEEE comparison)
  But: 1/+0 = +inf, 1/-0 = -inf (distinguishable)

+inf:  s=0, exponent=all 1s, fraction=0
-inf:  s=1, exponent=all 1s, fraction=0
  inf + inf = inf, inf - inf = NaN
  inf * 0 = NaN, inf / inf = NaN
  x / inf = 0 (for finite x)

NaN:   exponent=all 1s, fraction != 0
  Quiet NaN (qNaN):    fraction MSB = 1, propagates through operations
  Signaling NaN (sNaN): fraction MSB = 0, raises exception on use

  NaN != NaN is TRUE (the only value not equal to itself)
  This is the standard NaN check: x != x implies x is NaN
```

### Rounding Modes

IEEE 754 mandates five rounding modes:

```
Mode                   Alias              Behavior
roundTiesToEven       "banker's rounding"  Round to nearest; ties to even LSB
roundTiesToAway       (rare)               Round to nearest; ties away from 0
roundTowardPositive   ceil                 Round toward +infinity
roundTowardNegative   floor                Round toward -infinity
roundTowardZero       truncation           Round toward zero

Default: roundTiesToEven (statistically unbiased for sums)

Example (rounding to integer):
  0.5 -> 0 (tie, even)    1.5 -> 2 (tie, even)
  2.5 -> 2 (tie, even)    3.5 -> 4 (tie, even)
```

---

## 5. Floating-Point Error Analysis

### The Problem

Quantify and bound the error introduced by floating-point representation and arithmetic.

### Machine Epsilon

Machine epsilon $\epsilon$ is the smallest value such that $fl(1 + \epsilon) \neq 1$ in floating-point arithmetic.

```
binary32:  eps = 2^(-23) ~ 1.19e-7   (24-bit significand)
binary64:  eps = 2^(-52) ~ 2.22e-16  (53-bit significand)
binary128: eps = 2^(-112) ~ 1.93e-34 (113-bit significand)
```

### Unit in the Last Place (ULP)

$\text{ulp}(x)$ is the weight of the least significant bit of $x$'s significand:

```
For a normal number x with exponent e:
  ulp(x) = 2^(e - p + 1)    where p is significand precision

Relative error of a correctly rounded result:
  |fl(x) - x| / |x| <= 0.5 * eps    (roundTiesToEven)

In terms of ULP:
  |fl(x) - x| <= 0.5 * ulp(x)
```

### Error Propagation

```
Basic operation errors (assuming exact inputs):
  fl(x op y) = (x op y)(1 + delta)  where |delta| <= eps/2

Addition/subtraction:
  Absolute error is bounded, relative error can be catastrophic.

  Catastrophic cancellation:
    x = 1.00000000 (8 digits)
    y = 0.99999999
    x - y = 0.00000001 (1 significant digit from 8!)

    Relative error amplification: ~10^7 for this example

Multiplication/division:
  Relative error is well-behaved.
  fl(x * y) = x * y * (1 + delta),  |delta| <= eps/2
  Relative errors add: rel_err(x*y) <= rel_err(x) + rel_err(y) + eps/2

Summation (Kahan compensated sum):
  Naive sum of n values: error bound = (n-1) * eps * sum(|x_i|)
  Kahan sum: error bound = 2 * eps * sum(|x_i|) + O(n * eps^2)

  // Kahan summation algorithm:
  sum = 0.0
  c = 0.0              // compensation for lost low-order bits
  for each x_i:
    y = x_i - c         // compensated input
    t = sum + y          // may lose low bits of y
    c = (t - sum) - y    // recover what was lost
    sum = t
```

### Goldberg's Key Results

David Goldberg's "What Every Computer Scientist Should Know About Floating-Point Arithmetic" (1991) is the canonical reference. Key results:

**Theorem 1 (Rounding Error):** If floating-point addition, subtraction, multiplication, or division is performed with $p$ digits of precision and rounding to nearest, the relative error is at most $\frac{1}{2} \beta^{1-p}$ where $\beta$ is the base.

**Theorem 2 (Guard Digits):** Using a single guard digit during subtraction reduces the relative error bound from $\beta - 1$ to $2\epsilon$.

```
Without guard digit (beta=10, p=3):
  1.00 * 10^0 - 9.99 * 10^(-1)
  = 1.00 * 10^0 - 0.99 * 10^0    (shift, truncate to p digits)
  = 0.01 * 10^0
  Correct: 0.001, error = 0.009, relative error = 9.0 (900%!)

With guard digit:
  = 1.00 * 10^0 - 0.999 * 10^0   (one extra digit kept)
  = 0.001 * 10^0
  Correct answer, no error.
```

**Theorem 3 (Sterbenz):** If $x/2 \leq y \leq 2x$, then $fl(x - y) = x - y$ exactly (no rounding error).

**Theorem 4 (Exact Rounding):** The IEEE 754 requirement that basic operations produce the correctly rounded result of the exact operation ensures reproducibility: the same computation on any conforming implementation produces the same result (modulo expression evaluation order).

### Numerical Stability Patterns

```
UNSTABLE: Computing variance in one pass
  var = (sum_x2 - (sum_x)^2/n) / (n-1)
  Catastrophic cancellation when values are large with small variance.

STABLE: Welford's online algorithm
  M = 0, S = 0
  for k = 1 to n:
    old_M = M
    M = M + (x_k - M) / k
    S = S + (x_k - M) * (x_k - old_M)
  variance = S / (n - 1)

UNSTABLE: Quadratic formula
  x = (-b + sqrt(b^2 - 4ac)) / (2a)
  When b^2 >> 4ac, cancellation in -b + sqrt(b^2 - ...).

STABLE: Use the alternative form
  x1 = (-b - sign(b)*sqrt(b^2 - 4ac)) / (2a)
  x2 = c / (a * x1)                    // Vieta's formula

UNSTABLE: Computing e^x - 1 for small x
  exp(0.0000001) = 1.0000001, then subtract 1 -> only 1 digit

STABLE: Use expm1() library function (Taylor series: x + x^2/2 + ...)
Similarly: log(1+x) -> log1p(x) for small x
```

---

## 6. The Real Number Line and Floating-Point Density

### Distribution of Floating-Point Numbers

Floating-point numbers are NOT uniformly distributed. They are logarithmically spaced:

```
Between consecutive powers of 2, there are exactly 2^(p-1) values
(where p is the significand precision).

binary32: 2^23 = 8388608 values between each [2^e, 2^(e+1))

Spacing between consecutive floats at magnitude x:
  gap(x) = ulp(x) = 2^(floor(log2(|x|)) - p + 1)

At x = 1.0:    gap = 2^(-23)  ~ 1.19e-7
At x = 1000.0: gap = 2^(-13)  ~ 1.22e-4
At x = 1e10:   gap = 2^10     = 1024

Half of all representable floats are in [-1, 1].
Half of the remaining are in [-2, -1] union [1, 2].
And so on -- density halves with each doubling of magnitude.
```

### Wobble Factor

The relative spacing between consecutive floats varies by a factor of $\beta$ (the base):

```
Just above 1.0:   relative gap = eps          (tightest)
Just below 2.0:   relative gap = eps          (same exponent)
Just above 2.0:   relative gap = 2*eps        (new exponent, wider)

This "wobble" means the relative precision oscillates between
eps and beta*eps across the number line.

For beta=2 (IEEE 754): wobble factor = 2 (worst case 2x wider spacing)
For beta=16 (IBM hex float): wobble factor = 16 (much worse)
This is why binary floats are preferred over hex floats.
```

---

## 7. Fixed-Point, BCD, and Alternative Representations

### Fixed-Point Arithmetic

```
Representation: Qm.n format
  m = integer bits (including sign for signed)
  n = fractional bits
  Total bits = m + n (+ sign if separate)

Value = raw_bits / 2^n    (for signed: interpret raw_bits as two's complement)

Operations:
  Addition:        (a + b) with same Q format, result same format
  Multiplication:  Qm1.n1 * Qm2.n2 = Q(m1+m2).(n1+n2)
                   Must shift right by n to get back to Qm.n
  Division:        Pre-shift dividend left by n bits, then integer divide

Applications:
  DSP (audio, radio): Q1.15, Q1.31
  Game engines: fixed-point was standard before FPU ubiquity
  Financial: exact decimal fixed-point (2 fractional digits for cents)
  Embedded: microcontrollers without FPU
```

### BCD Arithmetic

```
BCD addition with carry correction:
  Add as if hex, then adjust digits > 9 by adding 6.

  27 + 35:
    0010 0111   (BCD 27)
  + 0011 0101   (BCD 35)
  -----------
    0101 1100   (hex result: 5C)

  Low nibble C > 9, add 6:  1100 + 0110 = 0010, carry 1
    0110 0010   (BCD 62)   correct!

x86 instructions: DAA (decimal adjust after addition),
                  DAS (decimal adjust after subtraction)
These were removed in x86-64 long mode.
```

### Posit Numbers (Gustafson, 2017)

```
An alternative to IEEE 754 with:
  - No NaN, no -0, one infinity (projective closure)
  - Tapered precision: more bits near 1.0, fewer at extremes
  - Format: [sign][regime][exponent][fraction]
    Regime is a variable-length field (run of same bits)

Claims: better accuracy-per-bit than IEEE 754 for many workloads.
Status: Research. Some FPGA implementations. Not yet standardized.
```

---

## 8. Number-Theoretic Foundations

### Modular Arithmetic in Hardware

All unsigned integer arithmetic is performed modulo $2^n$:

```
8-bit unsigned: arithmetic mod 256
  200 + 100 = 300 mod 256 = 44  (overflow/wrap)
  0 - 1 = -1 mod 256 = 255     (underflow/wrap)

Two's complement signed: also mod 2^n, but interpreted differently
  Signed overflow is UNDEFINED BEHAVIOR in C/C++
  (compilers may optimize assuming it never happens)
  Unsigned overflow is well-defined wrapping in C.
```

### Fermat and Mersenne Numbers

```
Mersenne numbers: M_n = 2^n - 1
  Mersenne primes: n = 2, 3, 5, 7, 13, 17, 19, 31, ...
  Used in: hash functions, PRNGs (Mersenne Twister uses M_19937)

  Connection to computing: 2^n - 1 = all n bits set = max unsigned value
  Checking if x = 2^n - 1:  (x & (x+1)) == 0

Fermat numbers: F_n = 2^(2^n) + 1
  F_0=3, F_1=5, F_2=17, F_3=257, F_4=65537
  F_4 = 65537 is used as the RSA public exponent (0x10001)
  Only 5 Fermat primes are known.
```

### Galois Fields GF(2^n)

```
GF(2) = {0, 1} with XOR as addition, AND as multiplication.

GF(2^n) = polynomials over GF(2) modulo an irreducible polynomial
  of degree n.

Applications:
  CRC checksums: polynomial division over GF(2)
  AES encryption: arithmetic in GF(2^8) mod x^8+x^4+x^3+x+1
  Reed-Solomon codes: arithmetic in GF(2^8) for error correction
  Carry-less multiplication: PCLMULQDQ instruction (x86)

Example GF(2^8) for AES:
  Elements: bytes 0x00 through 0xFF
  Addition: XOR
  Multiplication: polynomial multiply mod P(x) = x^8+x^4+x^3+x+1
    0x53 * 0xCA = 0x01 (multiplicative inverse exists for all nonzero)
```

### p-adic Numbers (Alternative Completion)

```
Real numbers = completion of rationals by Archimedean absolute value.
p-adic numbers = completion by p-adic absolute value.

The p-adic absolute value: |x|_p = p^(-v_p(x))
  where v_p(x) is the largest power of p dividing x.

In the 2-adic integers:
  ...11111111 = -1  (infinite string of 1s to the left!)
  This is the 2-adic representation of -1, analogous to
  two's complement with infinitely many bits.

Two's complement is literally truncated 2-adic arithmetic.
The "magic" of two's complement is explained by the fact that
Z_2 (2-adic integers) naturally represent negative numbers
with leading 1s extending to infinity.
```

---

## 9. Endianness and Byte Order

### Theory

```
A multi-byte value V occupying addresses a, a+1, ..., a+n-1:

Big-endian: byte at address a is the MOST significant byte
  V = sum(byte[a+i] * 256^(n-1-i)) for i = 0..n-1
  "Big end first" -- like English left-to-right reading

Little-endian: byte at address a is the LEAST significant byte
  V = sum(byte[a+i] * 256^i) for i = 0..n-1
  "Little end first"

0x01020304 at address 0x1000:
  Big-endian:    1000: 01  1001: 02  1002: 03  1003: 04
  Little-endian: 1000: 04  1001: 03  1002: 02  1003: 01
```

### Mixed Endianness

```
Middle-endian (PDP-11): 0x01020304 stored as 02 01 04 03
  (16-bit words in big-endian, but words stored little-endian)
  Mostly historical curiosity.

Bi-endian architectures: ARM, MIPS, PowerPC, RISC-V
  Can switch endianness via control register.
  ARM default: little-endian (since ARMv7).

Network byte order: big-endian (RFC 1700)
  All multi-byte protocol fields use big-endian.
  htons/htonl/ntohs/ntohl convert between host and network order.
```

### Endianness in IEEE 754

```
IEEE 754 does NOT specify byte order of floating-point values.
In practice:
  x86/x64: little-endian (both integer and float)
  ARM: configurable, default little-endian
  SPARC/MIPS: historically big-endian

  3.14 as binary64 (double):
  Hex: 0x40091EB851EB851F
  Big-endian:    40 09 1E B8 51 EB 85 1F
  Little-endian: 1F 85 EB 51 B8 1E 09 40
```

---

## 10. Connections to Information Theory

### Entropy and Optimal Encoding

```
A set of 2^n distinct symbols requires at least n bits to encode
(pigeonhole principle). This is the information-theoretic minimum.

8 bits -> 256 symbols (one byte encodes one ASCII-range character)
32 bits -> ~4.3 billion values (one IPv4 address)
128 bits -> ~3.4e38 values (one IPv6 address, one UUID)

Shannon entropy: H = -sum(p_i * log2(p_i))
  Gives the minimum average bits per symbol for a given distribution.
  Uniform distribution over 2^n symbols: H = n (maximum entropy).
```

### Radix Economy

```
The "cost" of representing a number N in base b is proportional to
b * ceil(log_b(N)) (digits needed times symbol count).

Minimized at b = e ~ 2.718...
Integer nearest: b = 3 (ternary is theoretically most efficient)
Binary (b = 2): radix economy = 2 * log2(N) / log3(N) ~ 1.26x worse
But: binary has enormous engineering advantages
  (simple gates, noise immunity, reliable storage).

Practical footnote: ternary computing was built (Setun, 1958, Moscow)
but binary won decisively due to engineering simplicity.
```

---

## See Also

- boolean-algebra
- subnetting
- ascii
- unicode
- algorithm-analysis

## References

- IEEE 754-2019, "IEEE Standard for Floating-Point Arithmetic"
- Goldberg, D. "What Every Computer Scientist Should Know About Floating-Point Arithmetic" (1991), ACM Computing Surveys 23(1), pp. 5-48
- Knuth, D. E. "The Art of Computer Programming, Vol. 2: Seminumerical Algorithms" (3rd ed., Addison-Wesley, 1997)
- Higham, N. J. "Accuracy and Stability of Numerical Algorithms" (2nd ed., SIAM, 2002)
- Muller, J.-M. et al. "Handbook of Floating-Point Arithmetic" (2nd ed., Birkhauser, 2018)
- Overton, M. L. "Numerical Computing with IEEE Floating Point Arithmetic" (SIAM, 2001)
- Patterson, D. & Hennessy, J. "Computer Organization and Design" (6th ed., Morgan Kaufmann, 2020)
- Parhami, B. "Computer Arithmetic: Algorithms and Hardware Designs" (2nd ed., Oxford, 2010)
- Gustafson, J. "Posit Arithmetic" (2017), technical report
- Kahan, W. "A Logarithm Too Clever by Half" (2004), UC Berkeley lecture notes
- Sterbenz, P. H. "Floating-Point Computation" (Prentice-Hall, 1974)
- Koren, I. "Computer Arithmetic Algorithms" (2nd ed., A K Peters, 2002)
