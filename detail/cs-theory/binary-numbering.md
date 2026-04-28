# Binary Numbering — Deep Dive

The mathematical companion to the ramp-up sheet — bit-level definitions, derivations, and worked encodings for positional notation, two's complement, IEEE 754 floating point, endianness, character encodings, base64, CRC, information theory, and bit-manipulation idioms. Where the cheat sheet says "use this trick," this deep dive proves why it works and walks through the bit pattern.

## Setup

A computer is, at the end of all the abstractions, a finite-state machine that operates on a fixed-width vector of bits. Every number, character, image, sound, and program is encoded as such a vector. The choice of encoding determines how arithmetic works, how comparisons behave, what range is representable, and where rounding error appears.

This document treats the encodings rigorously. Each section gives the mathematical definition, the bit-by-bit layout, and a worked example. The goal is that after reading you can decode any 32-bit word by hand: identify whether it is an unsigned integer, a two's complement integer, an IEEE 754 float, a UTF-8 fragment, or a fixed-point fraction; extract its semantic value; and reason about precision, overflow, and representational pitfalls.

The mathematics of bits is older than the digital computer. Leibniz published a binary numeration system in 1703. George Boole gave us Boolean algebra in 1854. Claude Shannon married the two in his 1937 master's thesis, showing that switching circuits realized Boolean expressions and that a bit is the unit of information. The hardware came later. The math has not changed.

## Number systems

A positional number system represents an integer n in base b using digits d_i ∈ {0, 1, ..., b-1}:

```
n = Σ d_i × b^i,  i = 0, 1, ..., k-1
```

The digit d_0 is the least significant; d_{k-1} is the most significant. The number has k digits in base b. The maximum representable value with k digits is b^k − 1; the minimum is 0.

Common bases in computing:

| Base | Name     | Digits         | Use                                     |
|------|----------|----------------|-----------------------------------------|
| 2    | binary   | 0, 1           | machine native; logic                   |
| 8    | octal    | 0..7           | Unix permissions; legacy mainframes     |
| 10   | decimal  | 0..9           | humans; financial calculations          |
| 16   | hex      | 0..9, A..F     | memory addresses; bytes                 |
| 64   | base64   | A..Z, a..z, 0..9, +, / | binary-to-text encoding         |

The base does not change the value; only the surface notation. The decimal 255, the hex 0xFF, the octal 0o377, and the binary 0b11111111 are the same integer.

### Why powers of two?

Because each bit independently can be 0 or 1, k bits can encode 2^k distinct values. This is the source of the powers-of-two rhythm that pervades computing:

```
1 bit   = 2 values
4 bits  = 16 values  (one hex digit)
8 bits  = 256 values (one byte)
10 bits = 1024 values (KiB boundary)
16 bits = 65536 values (one short)
20 bits = 1,048,576 values (1 MiB)
30 bits = 1,073,741,824 values (≈1 GiB)
32 bits = 4,294,967,296 values
40 bits = ≈1 TiB
64 bits = 18,446,744,073,709,551,616 values
```

### Converting bases

**Decimal → base b** by repeated division. Divide n by b; the remainder is d_0. Divide the quotient by b; the remainder is d_1. Repeat until the quotient is 0. Read the remainders in reverse order.

Example: 156 → binary

```
156 ÷ 2 = 78 r 0   ← d_0
 78 ÷ 2 = 39 r 0   ← d_1
 39 ÷ 2 = 19 r 1   ← d_2
 19 ÷ 2 =  9 r 1   ← d_3
  9 ÷ 2 =  4 r 1   ← d_4
  4 ÷ 2 =  2 r 0   ← d_5
  2 ÷ 2 =  1 r 0   ← d_6
  1 ÷ 2 =  0 r 1   ← d_7

Read up: 10011100  →  156 = 0b10011100 = 0x9C
```

Verify: 128 + 16 + 8 + 4 = 156. ✓

**Base b → decimal** by Horner's method. Walk the digits from most to least significant, accumulator a starts at 0:

```
a ← 0
for each digit d (msb first):
    a ← a × b + d
```

Example: 0x9C → decimal

```
a ← 0
a ← 0 × 16 + 9 = 9
a ← 9 × 16 + 12 = 144 + 12 = 156
```

**Binary ↔ hex** by grouping. Each hex digit corresponds to exactly four binary bits. Group from the right:

```
0b1001 1100
   9    C
   = 0x9C
```

This is why hex is the natural shorthand for byte-oriented data: a byte is exactly two hex digits.

**Binary ↔ octal** by grouping three. Less common in modern systems but appears in Unix permissions:

```
0b 110 100 100
    6   4   4
    = 0o644  →  rw-r--r--
```

### Fractional conversion

For fractions, multiply by b instead of dividing. Each multiplication produces an integer part that is the next digit:

Example: 0.625 → binary

```
0.625 × 2 = 1.25   → integer part 1 → d_{-1} = 1
0.25  × 2 = 0.5    → 0              → d_{-2} = 0
0.5   × 2 = 1.0    → 1              → d_{-3} = 1
0     × 2 = 0      → halt (zero)

0.625 = 0.101_2
```

Verify: 1/2 + 1/8 = 5/8 = 0.625. ✓

Some terminating decimals are non-terminating in binary:

```
0.1 × 2 = 0.2  → 0
0.2 × 2 = 0.4  → 0
0.4 × 2 = 0.8  → 0
0.8 × 2 = 1.6  → 1
0.6 × 2 = 1.2  → 1
0.2 × 2 = 0.4  → 0  (cycle!)

0.1 = 0.0001100110011..._2 (repeating)
```

This is the root cause of `0.1 + 0.2 != 0.3` in IEEE 754.

## Bit operations

Bitwise operations apply Boolean logic to corresponding bit positions of two operands. They are the primitive operations of digital logic: combinational circuits realize them in propagation-time gates.

### Truth tables

```
AND (∧, &)       OR (∨, |)        XOR (⊕, ^)        NOT (¬, ~)
A B | A∧B        A B | A∨B        A B | A⊕B         A | ¬A
----+----        ----+----        ----+----         ---+----
0 0 |  0         0 0 |  0         0 0 |  0          0 |  1
0 1 |  0         0 1 |  1         0 1 |  1          1 |  0
1 0 |  0         1 0 |  1         1 0 |  1
1 1 |  1         1 1 |  1         1 1 |  0
```

Identities (where 0 is the all-zeros word and 1 is the all-ones word):

```
x & 0 = 0           x | 0 = x          x ^ 0 = x          ~~x = x
x & 1 = x           x | 1 = 1          x ^ 1 = ~x         x & ~x = 0
x & x = x           x | x = x          x ^ x = 0          x | ~x = 1
```

XOR is its own inverse: `(a ^ b) ^ b = a`. This underlies the swap-without-temp idiom and one-time pad cryptography.

De Morgan's laws move negation across operations:

```
~(a & b) = ~a | ~b
~(a | b) = ~a & ~b
```

### Shifts

A logical left shift by k positions multiplies by 2^k (modulo word width):

```
x << k = x · 2^k  (mod 2^w)
```

Bits shifted off the left are discarded; zeros fill in from the right:

```
   0b 0001 0011  (19)
<< 2
   0b 0100 1100  (76 = 19 × 4)
```

A logical right shift by k positions divides an unsigned value by 2^k, discarding remainder:

```
x >> k = ⌊x / 2^k⌋   (unsigned)
```

Bits shifted off the right are discarded; zeros fill in from the left.

An arithmetic right shift fills the vacated MSBs with copies of the original sign bit. For two's complement signed integers, this gives floor-division by 2^k:

```
x >> k = ⌊x / 2^k⌋   (signed; floor toward −∞)
```

But beware: for negative x with non-zero remainder, this is not the same as integer division (which rounds toward zero in C and most languages). `(-3) >> 1 = -2` (floor) but `(-3) / 2 = -1` (trunc).

In C, the result of right-shifting a negative signed integer is implementation-defined, though virtually all compilers produce arithmetic shifts. C++20 made this defined.

### Rotates

A rotate left by k is a left shift that wraps the displaced MSBs back into the LSBs:

```
rol(x, k) = (x << k) | (x >> (w - k))   (no carry, w = word width)
```

A rotate right is the dual:

```
ror(x, k) = (x >> k) | (x << (w - k))
```

Rotates preserve all bits — no information is lost. They are common in cryptography (SHA, AES) and CRC computations.

x86 has dedicated `ROL`/`ROR` instructions and `RCL`/`RCR` (rotate through carry, which involves the CPU carry flag as an extra bit, making it a w+1 bit rotate).

### Common patterns

**Clear lowest set bit:** `x & (x - 1)`

If x = `...10110000`, then `x - 1 = ...10101111`. AND yields `...10100000`. The lowest 1 became 0; bits below stayed 0.

```
x       = 0b 0010 1100
x - 1   = 0b 0010 1011
x &(x-1)= 0b 0010 1000  ← lowest set bit (bit 2) cleared
```

This is the kernel of Brian Kernighan's popcount: each iteration clears one bit, so the loop runs popcount(x) times instead of w times.

```c
int popcount(uint64_t x) {
    int count = 0;
    while (x) {
        x &= x - 1;
        count++;
    }
    return count;
}
```

**Isolate lowest set bit:** `x & -x`

In two's complement, `-x = ~x + 1`. The bits below the lowest set bit of x are 0; flipping them gives 1; adding 1 propagates a carry that flips them back. The lowest set bit of x flips to 0 but the carry sets it back to 1. All bits above the lowest set bit are flipped. The result is that ANDing x with -x leaves only the lowest set bit:

```
x       = 0b 0010 1100
-x      = 0b 1101 0100
x & -x  = 0b 0000 0100  ← only bit 2
```

**Test if power of 2:** `x && !(x & (x - 1))`

A non-zero power of 2 has exactly one bit set; clearing the lowest set bit yields 0. Zero is excluded by the `x &&` guard.

**Round up to next power of 2** (for 32-bit x):

```c
uint32_t round_up_pow2(uint32_t x) {
    x--;
    x |= x >> 1;
    x |= x >> 2;
    x |= x >> 4;
    x |= x >> 8;
    x |= x >> 16;
    return x + 1;
}
```

After `x--`, the highest set bit propagates downward via the shifts, filling all bits below. Adding 1 produces the next power of 2.

**Trailing zeros (ctz):** count zeros below the lowest set bit. With dedicated instructions (`TZCNT` on x86, `RBIT`+`CLZ` on ARM) this is one cycle. Software fallback uses De Bruijn sequences:

```c
static const int debruijn32[32] = {
    0,  1, 28,  2, 29, 14, 24,  3,
   30, 22, 20, 15, 25, 17,  4,  8,
   31, 27, 13, 23, 21, 19, 16,  7,
   26, 12, 18,  6, 11,  5, 10,  9
};

int ctz(uint32_t x) {
    return debruijn32[((x & -x) * 0x077CB531U) >> 27];
}
```

**Leading zeros (clz):** count zeros above the highest set bit. `__builtin_clz` in GCC, `_lzcnt_u32` intrinsic. `clz(x)` is undefined for x = 0.

**Population count (popcount):** count of set bits. Hardware: `POPCNT` (x86, since SSE4.2), `CNT` (ARMv8). Software:

```c
int popcount(uint64_t x) {
    x = x - ((x >> 1) & 0x5555555555555555ULL);
    x = (x & 0x3333333333333333ULL) + ((x >> 2) & 0x3333333333333333ULL);
    x = (x + (x >> 4)) & 0x0F0F0F0F0F0F0F0FULL;
    return (x * 0x0101010101010101ULL) >> 56;
}
```

This is the SWAR (SIMD-within-a-register) algorithm: parallel reductions that compute pairwise sums, then 4-way sums, then 8-way (byte) sums, then a multiply that broadcasts and sums the bytes.

## Two's complement

Two's complement is the universal encoding for signed integers in modern hardware. It encodes a w-bit signed integer in the same w bits as the unsigned encoding but reinterprets the meaning so that:

```
n_signed = -d_{w-1} · 2^(w-1) + Σ d_i · 2^i,  i = 0..w-2
```

The MSB has a negative weight equal to the absolute value of the largest power of two; all other bits have positive weights. The range is:

```
[-2^(w-1), 2^(w-1) - 1]
```

For w = 8: range is [−128, 127].
For w = 16: range is [−32768, 32767].
For w = 32: range is [−2,147,483,648, 2,147,483,647].
For w = 64: range is [−9.22 × 10¹⁸, 9.22 × 10¹⁸].

Note the asymmetry: there is one more negative number than positive. This is because zero takes one of the 2^w bit patterns, and there is no "negative zero" in two's complement (unlike sign-magnitude).

### Why two's complement?

Because addition, subtraction, and equality work without sign-aware logic. The same hardware adder produces correct results for both signed and unsigned operands, with the only difference being which condition flags are tested for overflow.

```
   8-bit example:
   3 - 5 = ?

   3      = 0b 0000 0011
  -5      = 0b 1111 1011  (two's complement of 5)

   3 + (-5) = 0b 1111 1110  = -2  ✓  (using signed interpretation)
```

The same bit pattern `0b 1111 1110` represents 254 if interpreted as unsigned and −2 if interpreted as signed. Hardware does the addition; software (or compiler-emitted condition codes) interprets the result.

### Negation

To compute `−x` in two's complement: invert all bits and add 1.

```
x  =  5 = 0b 0000 0101
~x =     0b 1111 1010
~x + 1 = 0b 1111 1011 = -5
```

This works because `x + ~x = -1` (all ones in two's complement is −1), so `~x = -1 − x`, and `~x + 1 = −x`.

The smallest negative number INT_MIN has no two's-complement positive counterpart: negating it overflows back to itself. `INT_MIN = 0x80000000` for int32; `−INT_MIN` would need to be `0x80000000` again. Compilers may treat `-INT_MIN` as undefined behavior or wrap to INT_MIN.

### Sign extension

To widen a signed value from k bits to k+m bits, replicate the sign bit into the new MSBs:

```
8-bit:  -5 = 0b 1111 1011
16-bit: -5 = 0b 1111 1111 1111 1011  (sign-extend 1 into upper byte)

8-bit:   5 = 0b 0000 0101
16-bit:  5 = 0b 0000 0000 0000 0101  (sign-extend 0)
```

C's `(int)((char)x)` does sign extension if the source is signed; `(int)((unsigned char)x)` does zero extension. The x86 `MOVSX` instruction sign-extends; `MOVZX` zero-extends.

### Overflow detection for addition

Add a + b. Overflow occurred iff a and b have the same sign but the result has the opposite sign:

```
OF = (sign_a == sign_b) && (sign_result != sign_a)
```

Mathematically: signed-overflow happened iff the unsigned sum modulo 2^w differs from the true mathematical sum being expressible in w-bit signed form.

For unsigned addition, overflow is detected by carry-out (the carry from the MSB position):

```
CF = carry_out_of_msb_position
```

x86 sets `OF` and `CF` separately; signed code branches on `OF`, unsigned code branches on `CF`.

### Subtraction overflow

Subtraction is just `a + (~b) + 1`. Overflow detection: if a and b have opposite signs but the result has the sign of b, overflow occurred:

```
OF = (sign_a != sign_b) && (sign_result == sign_b)
```

### Multiplication overflow

For w-bit signed multiplication, the full product is up to 2w bits. To detect overflow of the truncated w-bit result, x86 provides `IMUL` which sets `OF` and `CF` if the high w bits are not the sign extension of the low w bits.

Software check before multiplication:

```c
bool would_overflow_smul(int32_t a, int32_t b) {
    int64_t prod = (int64_t)a * b;
    return prod < INT32_MIN || prod > INT32_MAX;
}
```

## IEEE 754 floating point

Floating point represents real numbers in scientific notation with a fixed-width sign bit, exponent, and mantissa (also called significand or fraction). IEEE 754-1985 standardized the formats and rounding rules; IEEE 754-2008 added decimal floats and fused multiply-add; IEEE 754-2019 added augmented operations.

### Bit layout

**float32 (binary32):** 1 + 8 + 23 = 32 bits

```
[s][  exp_8  ][        mantissa_23        ]
 31  30..23   22                          0
```

**float64 (binary64):** 1 + 11 + 52 = 64 bits

```
[s][  exp_11   ][              mantissa_52              ]
 63  62..52     51                                     0
```

**float16 (binary16, IEEE half):** 1 + 5 + 10 = 16 bits — used in ML and graphics.

**bfloat16 (Google brain float):** 1 + 8 + 7 = 16 bits — same exponent range as float32, less mantissa precision.

**float128 (binary128, "quad"):** 1 + 15 + 112 = 128 bits — rare in hardware; implemented in software.

### Encoded value

For normalized numbers (0 < exp < max):

```
value = (-1)^s · 1.mantissa · 2^(exp - bias)
```

The leading 1 is implicit ("hidden bit"). The bias is 2^(e−1) − 1, where e is the exponent width:

```
float16:  bias = 15
float32:  bias = 127
float64:  bias = 1023
float128: bias = 16383
```

The bias allows the exponent field to represent both positive and negative exponents using only unsigned values, simplifying comparisons (signed and unsigned compare yield the same result on biased exponents — almost; the sign bit of the float still matters).

For subnormals (exp = 0, mantissa ≠ 0):

```
value = (-1)^s · 0.mantissa · 2^(1 - bias)
```

The implicit leading bit is 0 instead of 1; the exponent is fixed at 1 − bias. Subnormals fill the gap between zero and the smallest normalized number, providing gradual underflow.

### Special values

The maximum exponent value (all ones) is reserved:

```
exp = max, mantissa = 0:        ±∞
exp = max, mantissa ≠ 0:        NaN

  - if mantissa MSB = 1:        quiet NaN (qNaN)   - propagates through ops
  - if mantissa MSB = 0:        signaling NaN (sNaN) - traps if enabled
```

The minimum exponent value (all zeros) is reserved:

```
exp = 0, mantissa = 0:          ±0
exp = 0, mantissa ≠ 0:          subnormal (denormal)
```

### Range and precision

float32:

```
smallest normal:   2^-126        ≈ 1.18 × 10^-38
largest normal:    (2 - 2^-23) · 2^127 ≈ 3.40 × 10^38
smallest subnormal: 2^-149       ≈ 1.40 × 10^-45
machine epsilon:   2^-23         ≈ 1.19 × 10^-7
decimal precision: ~7.2 digits   (since log10(2^24) ≈ 7.22)
```

float64:

```
smallest normal:   2^-1022       ≈ 2.23 × 10^-308
largest normal:    (2 - 2^-52) · 2^1023 ≈ 1.80 × 10^308
smallest subnormal: 2^-1074      ≈ 4.94 × 10^-324
machine epsilon:   2^-52         ≈ 2.22 × 10^-16
decimal precision: ~15.95 digits (since log10(2^53) ≈ 15.95)
```

Machine epsilon ε is the smallest number such that `1 + ε > 1` in float arithmetic; equivalently, the gap between 1.0 and the next representable value above it.

### Rounding modes

IEEE 754 specifies four rounding modes; the default is round-to-nearest-even (also called banker's rounding):

```
0. round-to-nearest, ties-to-even   (default)
1. round-toward-zero (truncation)
2. round-toward-+∞   (ceiling)
3. round-toward-−∞   (floor)
```

C99 added `<fenv.h>` with `fesetround()` to change the mode at runtime.

Round-to-nearest-even avoids the statistical bias of always rounding 0.5 up: half the time we round up, half down, depending on the parity of the LSB.

```
1.5 → 2  (round to even)
2.5 → 2  (round to even, NOT 3!)
3.5 → 4
4.5 → 4  (round to even)
```

## Float32/64 worked examples

### Encode 1.0 as float32

1.0 = 1.0 × 2^0. So mantissa = 0 (since the leading 1 is implicit), exp = 0 + 127 = 127:

```
sign  = 0
exp   = 127 = 0b 0111 1111
mant  = 0   = 0b 000 0000 0000 0000 0000 0000

bits  = 0 01111111 00000000000000000000000
        = 0x3F800000
```

### Encode -1.0 as float32

Same as 1.0 but sign = 1:

```
bits  = 1 01111111 00000000000000000000000
        = 0xBF800000
```

### Encode 2.0 as float32

2.0 = 1.0 × 2^1. exp = 128:

```
bits  = 0 10000000 00000000000000000000000
        = 0x40000000
```

### Encode 0.5 as float32

0.5 = 1.0 × 2^-1. exp = 126:

```
bits  = 0 01111110 00000000000000000000000
        = 0x3F000000
```

### Encode 0.1 as float32 (which cannot be represented exactly)

In binary: 0.1 = 0.0001100110011001100... (repeating with period 4)

Normalize: 0.1 = 1.10011001100110011001100... × 2^-4

Mantissa (the bits AFTER the implicit leading 1, rounded to 23 bits, ties-to-even):

```
1001 1001 1001 1001 1001 101  (rounded — the next bit was 1, and round-to-even decided up)
```

Wait — let's redo this carefully. The fraction starts after the leading 1:

```
1.10011001100110011001100|11001...  ← 23 bits + guard
                          ^^^ overflow region
```

The 23-bit mantissa keeps `10011001100110011001100`. The next bit (24th) is `1`, so we round up (ties-to-even on `...100|110...` rounds the LSB up because round-up is unambiguous when the discarded bits are not exactly 0.5):

```
10011001100110011001100 + 1 = 10011001100110011001101
```

Final encoding:

```
sign = 0
exp  = -4 + 127 = 123 = 0b 0111 1011
mant = 0b 100 1100 1100 1100 1100 1101

bits = 0 01111011 10011001100110011001101
     = 0x3DCCCCCD
```

Decoding 0x3DCCCCCD back gives `1.10011001100110011001101_2 × 2^-4`. The exact value is:

```
(1 + 0.6000000238...) × 0.0625 ≈ 0.10000000149011612
```

So the float32 representation of `0.1` is actually `0.10000000149011612`, not exactly `0.1`. This is the source of "0.1 + 0.2 ≠ 0.3":

```
0.1 (f32) ≈ 0.10000000149011612
0.2 (f32) ≈ 0.20000000298023224
sum       ≈ 0.30000000447034836
0.3 (f32) ≈ 0.30000001192092896

sum != 0.3 (different roundings)
```

### Encode π as float64

π = 3.14159265358979323846...

In binary: 11.0010010000111111011010101000100010000101101000110000100011010011...

Normalize: 1.10010010000111111011010101000100010000101101000110000100011010011 × 2^1

Take 52 bits of mantissa, round-to-nearest-even:

```
1001001000011111101101010100010001000010110100011000  + round bit decision
```

Standard double for π is `0x400921FB54442D18`. Decoding:

```
sign = 0
exp  = 0x400 = 1024 → unbiased = 1024 - 1023 = 1
mant = 0x921FB54442D18

  bits = 1001 0010 0001 1111 1011 0101 0100 0100 0100 0010 1101 0001 1000

value = 1 + (mant / 2^52)
      = 1 + 2570638124657944 / 4503599627370496
      = 1 + 0.5707963267948966
      = 1.5707963267948966

× 2^1 = 3.141592653589793
```

That matches π to all the digits float64 can hold.

### Encode +∞ as float32

```
sign = 0
exp  = 255 (all ones)
mant = 0

bits = 0 11111111 00000000000000000000000
     = 0x7F800000
```

`-∞` is `0xFF800000`.

### Encode NaN as float32

Any pattern with `exp = 255` and `mant ≠ 0` is a NaN. The C macro `NAN` typically produces `0x7FC00000` (qNaN with the canonical "indefinite" payload):

```
bits = 0 11111111 10000000000000000000000
     = 0x7FC00000  ← qNaN (mantissa MSB = 1)

bits = 0 11111111 00000000000000000000001
     = 0x7F800001  ← sNaN (mantissa MSB = 0)
```

NaN propagates through arithmetic: any operation with a NaN operand produces NaN. NaN is not equal to anything, including itself:

```c
float n = NAN;
assert(n != n);  // true!
assert(isnan(n)); // proper check
```

### Why 0.1 + 0.2 ≠ 0.3

Verification in Python:

```python
>>> import struct
>>> def hexbits(x): return hex(struct.unpack('<Q', struct.pack('<d', x))[0])
>>> hexbits(0.1)
'0x3fb999999999999a'
>>> hexbits(0.2)
'0x3fc999999999999a'
>>> hexbits(0.3)
'0x3fd3333333333333'
>>> hexbits(0.1 + 0.2)
'0x3fd3333333333334'  ← differs in last bit
>>> 0.1 + 0.2 == 0.3
False
>>> 0.1 + 0.2 - 0.3
5.551115123125783e-17
```

The error is one ULP (unit in the last place), the smallest discriminable unit at this magnitude.

## Endianness

When a multi-byte value is stored in memory, byte order matters. Two conventions dominate:

**Little-endian:** the least-significant byte at the lowest address.

```
uint32_t x = 0x12345678;
        addr 0    1    2    3
byte    0x78 0x56 0x34 0x12
```

**Big-endian:** the most-significant byte at the lowest address.

```
uint32_t x = 0x12345678;
        addr 0    1    2    3
byte    0x12 0x34 0x56 0x78
```

### Architecture conventions

```
Little-endian:  x86, x86_64, ARM (default; configurable), RISC-V (default), ARM64
Big-endian:     PowerPC (default), SPARC, m68k, IBM mainframes (z/Arch)
Network order:  big-endian (RFC 1700, "network byte order")
```

ARM and PowerPC are bi-endian: the architecture supports either; the kernel/firmware chooses at boot. Linux on ARM has historically been little-endian.

### Byte-swapping

Convert between endiannesses with a swap:

```c
uint32_t bswap32(uint32_t x) {
    return ((x & 0x000000FFu) << 24)
         | ((x & 0x0000FF00u) <<  8)
         | ((x & 0x00FF0000u) >>  8)
         | ((x & 0xFF000000u) >> 24);
}

uint16_t bswap16(uint16_t x) {
    return ((x & 0x00FFu) << 8) | ((x & 0xFF00u) >> 8);
}

uint64_t bswap64(uint64_t x) {
    return  ((x & 0x00000000000000FFULL) << 56)
          | ((x & 0x000000000000FF00ULL) << 40)
          | ((x & 0x0000000000FF0000ULL) << 24)
          | ((x & 0x00000000FF000000ULL) <<  8)
          | ((x & 0x000000FF00000000ULL) >>  8)
          | ((x & 0x0000FF0000000000ULL) >> 24)
          | ((x & 0x00FF000000000000ULL) >> 40)
          | ((x & 0xFF00000000000000ULL) >> 56);
}
```

x86 has dedicated `BSWAP` (32-bit and 64-bit), `XCHG` for 16-bit byte-swap. ARM has `REV`/`REV16`/`REV32`.

GCC/Clang builtins:

```c
__builtin_bswap16(x)
__builtin_bswap32(x)
__builtin_bswap64(x)
```

### Network byte order conversion

POSIX provides:

```c
#include <arpa/inet.h>
uint32_t htonl(uint32_t hostlong);   // host to network long (32-bit)
uint16_t htons(uint16_t hostshort);  // host to network short (16-bit)
uint32_t ntohl(uint32_t netlong);    // network to host
uint16_t ntohs(uint16_t netshort);   // network to host
```

On big-endian machines these are no-ops (return the argument unchanged). On little-endian machines they perform a byte swap.

For 64-bit values use `htobe64`/`be64toh` (Linux), or write your own.

### Detecting endianness

```c
#include <stdint.h>

bool is_little_endian(void) {
    uint32_t x = 1;
    return *(uint8_t *)&x == 1;
}
```

C++20 has `std::endian::native`. C23 provides `__STDC_ENDIAN_NATIVE__`.

### Pitfall: type punning

Aliasing a `uint32_t*` to a `uint8_t*` for endianness inspection is well-defined (char-pointer aliasing is allowed). Aliasing through a different non-char type violates strict aliasing in C and yields undefined behavior. Use `memcpy` for safe type punning.

### Bit endianness vs byte endianness

Within a byte, bits are not stored in addressable order; the LSB-first vs MSB-first ordering is a hardware concern, exposed only in serial protocols (RS-232, SPI mode bit ordering) and some bitstream formats (e.g. JPEG, Deflate). The byte's value is the same regardless.

## Encoding schemes

A character set is a mapping between glyphs (visible characters, including invisible whitespace and control codes) and integers (codepoints). An encoding is a way to serialize codepoints into a sequence of bytes.

### ASCII

The American Standard Code for Information Interchange, ratified 1963. 7 bits per character; 128 codepoints (0x00..0x7F).

```
0x00..0x1F : control codes (CR, LF, TAB, BEL, ...)
0x20       : space
0x21..0x2F : punctuation ! " # $ % & ' ( ) * + , - . /
0x30..0x39 : digits 0..9
0x3A..0x40 : punctuation : ; < = > ? @
0x41..0x5A : uppercase A..Z
0x5B..0x60 : punctuation [ \ ] ^ _ `
0x61..0x7A : lowercase a..z
0x7B..0x7E : punctuation { | } ~
0x7F       : DEL
```

Properties:
- Uppercase ↔ lowercase differ by one bit (0x20). `'A' | 0x20 = 'a'`. `'a' & ~0x20 = 'A'`.
- Digit value: `c − '0'`. `'7' − '0' = 7`.
- Hex digit value: `c <= '9' ? c − '0' : (c | 0x20) − 'a' + 10`.

ASCII is fully a subset of all the encodings discussed below.

### Extended ASCII / Latin-1

8-bit encodings using the 0x80..0xFF range for accented characters and other glyphs. The most common is Latin-1 (ISO 8859-1), covering Western European languages. Many regional variants existed (8859-2 Eastern European, 8859-5 Cyrillic, 8859-7 Greek, etc.). All have been largely superseded by UTF-8.

In Latin-1: 0xE9 = é, 0xF1 = ñ, 0xDF = ß. The 256-codepoint range is too small for global text.

### Unicode

Unicode (1991) defines a global codepoint space from U+0000 to U+10FFFF (1,114,112 possible codepoints, of which ~150k are assigned in current revisions). Codepoints group into 17 "planes" of 65536 each:

```
Plane 0 (Basic Multilingual Plane, BMP):   U+0000..U+FFFF
  - Latin, Cyrillic, Greek, Arabic, Hebrew, Devanagari, CJK Unified Ideographs
Plane 1 (Supplementary Multilingual Plane, SMP): U+10000..U+1FFFF
  - emoji, ancient scripts, mathematical alphanumerics
Plane 2 (Supplementary Ideographic Plane): U+20000..U+2FFFF
  - rare CJK ideographs
...
Plane 14 (Supplementary Special-purpose):  U+E0000..U+EFFFF
Planes 15-16 (Private Use Areas):          U+F0000..U+10FFFF
```

The first 128 Unicode codepoints are identical to ASCII.

### UTF-8

A variable-width encoding of Unicode codepoints using 1 to 4 bytes per codepoint. Designed by Ken Thompson and Rob Pike in 1992. RFC 3629 (2003) formalizes the modern restricted form (4 bytes max, no overlong encodings).

### UTF-16

A variable-width encoding using 2 or 4 bytes per codepoint. Codepoints in the BMP (U+0000..U+FFFF) take 2 bytes; codepoints outside the BMP (U+10000..U+10FFFF) take 4 bytes via "surrogate pairs":

```
codepoint ≥ 0x10000:
  cp' = cp − 0x10000           (yields 20-bit value)
  high = 0xD800 | (cp' >> 10)  (high surrogate, 0xD800..0xDBFF)
  low  = 0xDC00 | (cp' & 0x3FF) (low surrogate, 0xDC00..0xDFFF)
```

Codepoints U+D800..U+DFFF are reserved as surrogates and are invalid as standalone codepoints in any well-formed Unicode text.

UTF-16 is endianness-sensitive: byte order is signaled by a leading BOM (Byte Order Mark, U+FEFF). UTF-16BE/UTF-16LE are explicit. Used internally by Java strings, Windows APIs (UTF-16LE), and the JavaScript string type.

### UTF-32

A fixed-width encoding using 4 bytes per codepoint. Each codepoint is stored as a 32-bit unsigned integer (with at most 21 bits used). Endianness-sensitive. Rare in storage but used internally in some text-processing libraries (ICU's `UChar32`).

## UTF-8 byte structure

UTF-8 prefixes the first byte of each multi-byte sequence with a count of how many bytes follow:

```
bytes  | 1st byte    | continuation bytes      | codepoint range       | bits
-------+-------------+--------------------------+-----------------------+-----
1      | 0xxxxxxx    | (none)                   | U+0000..U+007F        |  7
2      | 110xxxxx    | 10xxxxxx                 | U+0080..U+07FF        | 11
3      | 1110xxxx    | 10xxxxxx 10xxxxxx        | U+0800..U+FFFF        | 16
4      | 11110xxx    | 10xxxxxx 10xxxxxx 10xxxxxx | U+10000..U+10FFFF   | 21
```

Properties:
- ASCII (U+0000..U+007F) is encoded as a single byte identical to ASCII. UTF-8 is fully ASCII-compatible.
- The high bit of any non-leading byte is 1; the high bit of any byte that starts a sequence is determined by the byte count.
- A continuation byte begins with `10`; a leading byte begins with `0`, `110`, `1110`, or `11110`.
- No byte can be `0xC0`, `0xC1`, `0xF5`..`0xFF` in well-formed UTF-8 (these would start "overlong" or out-of-range sequences).
- The encoding is self-synchronizing: from any byte you can find the start of the current codepoint by walking backward to the first non-continuation byte (at most 3 steps).
- Lexicographic byte order matches lexicographic codepoint order (a property called "binary-collation safe").

### Encode worked examples

**'A' = U+0041:** 1 byte (since 0x41 < 0x80)

```
0x41 = 0b 0100 0001  (no transformation; ASCII)

UTF-8 bytes: 0x41
```

**'ñ' = U+00F1:** 2 bytes (since 0x80 ≤ 0xF1 < 0x800)

```
codepoint:    0b 0000 0000 1111 0001 = 0xF1

split into 5+6 bits:
   high 5 = 0b 00011  ← top 5 bits
   low  6 = 0b 110001 ← bottom 6 bits

byte 1: 110 00011 = 0b 11000011 = 0xC3
byte 2: 10 110001 = 0b 10110001 = 0xB1

UTF-8 bytes: 0xC3 0xB1
```

**'€' = U+20AC:** 3 bytes (since 0x800 ≤ 0x20AC < 0x10000)

```
codepoint:    0b 0010 0000 1010 1100 = 0x20AC

split into 4+6+6 bits:
   high 4 = 0b 0010
   mid  6 = 0b 000010
   low  6 = 0b 101100

byte 1: 1110 0010 = 0xE2
byte 2: 10 000010 = 0x82
byte 3: 10 101100 = 0xAC

UTF-8 bytes: 0xE2 0x82 0xAC
```

**'𝒜' (script capital A) = U+1D49C:** 4 bytes

```
codepoint:    0b 0 0001 1101 0100 1001 1100 = 0x1D49C  (21 bits)

split into 3+6+6+6 bits:
   high 3 = 0b 000
   mid1 6 = 0b 011101
   mid2 6 = 0b 010010
   low  6 = 0b 011100

byte 1: 11110 000 = 0xF0
byte 2: 10 011101 = 0x9D
byte 3: 10 010010 = 0x92
byte 4: 10 011100 = 0x9C

UTF-8 bytes: 0xF0 0x9D 0x92 0x9C
```

**'😀' (grinning face) = U+1F600:** 4 bytes

```
codepoint:    0b 0 0001 1111 0110 0000 0000 = 0x1F600

high 3 = 0b 000
mid1 6 = 0b 011111
mid2 6 = 0b 011000
low  6 = 0b 000000

byte 1: 11110 000 = 0xF0
byte 2: 10 011111 = 0x9F
byte 3: 10 011000 = 0x98
byte 4: 10 000000 = 0x80

UTF-8 bytes: 0xF0 0x9F 0x98 0x80
```

### Decode worked example

Given bytes `0xE2 0x9C 0x93`, decode:

```
byte 1: 0xE2 = 0b 1110 0010 → 3-byte sequence; payload = 0b 0010
byte 2: 0x9C = 0b 1001 1100 → continuation; payload = 0b 011100
byte 3: 0x93 = 0b 1001 0011 → continuation; payload = 0b 010011

codepoint: 0b 0010 011100 010011
         = 0b 0010 0111 0001 0011
         = 0x2713  →  U+2713 (✓ HEAVY CHECK MARK)
```

### Validation rules

A well-formed UTF-8 byte sequence:
1. Each byte starts with one of: `0`, `110`, `1110`, `11110`, or `10`.
2. After a leading byte with k continuation bytes expected, exactly k continuation bytes follow.
3. The encoding is the shortest possible (no overlong encodings: e.g., U+0041 must not be encoded as two bytes).
4. The decoded codepoint is in the range U+0000..U+10FFFF, excluding the surrogate range U+D800..U+DFFF.

A naive decoder that does not enforce (3) is vulnerable to security exploits — early IIS allowed `..` directory traversal via overlong-encoded `/`.

## Base64 encoding

RFC 4648 specifies base64. The encoding takes 3 input bytes (24 bits) and produces 4 output characters (each representing 6 bits):

```
input:   AAAAAAAA BBBBBBBB CCCCCCCC
         |------ 24 bits -----|

split:   aaaaaa aabbbb bbbbcc cccccc
         |- 6 -||- 6 -||- 6 -||- 6 -|
         char 1 char 2 char 3 char 4
```

The 64-character alphabet:

```
index | char     index | char     index | char     index | char
------+-----     ------+-----     ------+-----     ------+-----
  0   |  A         16  |  Q         32  |  g         48  |  w
  1   |  B         17  |  R         33  |  h         49  |  x
  2   |  C         18  |  S         34  |  i         50  |  y
  3   |  D         19  |  T         35  |  j         51  |  z
  4   |  E         20  |  U         36  |  k         52  |  0
  5   |  F         21  |  V         37  |  l         53  |  1
  6   |  G         22  |  W         38  |  m         54  |  2
  7   |  H         23  |  X         39  |  n         55  |  3
  8   |  I         24  |  Y         40  |  o         56  |  4
  9   |  J         25  |  Z         41  |  p         57  |  5
 10   |  K         26  |  a         42  |  q         58  |  6
 11   |  L         27  |  b         43  |  r         59  |  7
 12   |  M         28  |  c         44  |  s         60  |  8
 13   |  N         29  |  d         45  |  t         61  |  9
 14   |  O         30  |  e         46  |  u         62  |  +
 15   |  P         31  |  f         47  |  v         63  |  /
```

If the input length is not a multiple of 3, the last group is padded:

```
input length mod 3 == 1  →  encode 1 byte as 2 chars + "==" padding
input length mod 3 == 2  →  encode 2 bytes as 3 chars + "=" padding
```

### URL-safe variant

RFC 4648 §5: replace `+` with `-` and `/` with `_`. Padding `=` may be omitted (length implies padding).

### Example

Encode "Man" (3 ASCII bytes):

```
input bytes:
  M = 0x4D = 0b 01001101
  a = 0x61 = 0b 01100001
  n = 0x6E = 0b 01101110

24-bit concatenation:
  0b 010011010110000101101110

split into 6-bit chunks:
  0b 010011 = 19 → 'T'
  0b 010110 = 22 → 'W'
  0b 000101 =  5 → 'F'
  0b 101110 = 46 → 'u'

output: "TWFu"
```

Encode "Ma" (2 bytes, requires "=" padding):

```
M = 0x4D = 0b 01001101
a = 0x61 = 0b 01100001

24-bit (zero-padded):
  0b 010011010110000100000000

split:
  0b 010011 = 19 → 'T'
  0b 010110 = 22 → 'W'
  0b 0001|00 =  4 → 'E'   (last 2 bits are zero-pad, marked with =)
  (no 4th group)

output: "TWE="
```

Encode "M" (1 byte, requires "==" padding):

```
M = 0x4D = 0b 01001101

24-bit (zero-padded):
  0b 010011010000000000000000

split:
  0b 010011 = 19 → 'T'
  0b 010000 = 16 → 'Q'
  (last two groups are pure padding)

output: "TQ=="
```

### Size overhead

Base64 expands data by a factor of 4/3 ≈ 1.33×, plus padding. A 1 MiB binary becomes ~1.34 MiB encoded. This is unavoidable: 8 bits encoded in 6 bits requires 4/3 expansion.

### Other encodings

```
base16 (hex)    : 2 chars per byte; 2× expansion; trivial decode
base32 (RFC 4648): 8 chars per 5 bytes; 1.6× expansion; case-insensitive (uppercase)
base58           : Bitcoin addresses; avoids 0/O, I/l ambiguity; ~1.37× expansion
base85 / Ascii85 : 5 chars per 4 bytes; 1.25× expansion; PDF/PostScript
base91 / basE91  : 1.23× expansion; less standard
```

## CRC and checksum math

A checksum is a small fixed-size value computed from a message that detects accidental corruption. Different checksums target different threat models.

### CRC-32

CRC stands for Cyclic Redundancy Check. CRC-32 (used in Ethernet, ZIP, gzip, PNG, SATA) treats the message as a polynomial over GF(2) and computes the remainder when divided by a fixed generator polynomial.

The standard CRC-32 generator (IEEE 802.3, used by Ethernet/zlib/zip):

```
G(x) = x^32 + x^26 + x^23 + x^22 + x^16 + x^12 + x^11 + x^10 + x^8 + x^7 + x^5 + x^4 + x^2 + x + 1
```

In polynomial-coefficient form (msb to lsb): `0x104C11DB7`. The high bit (x^32) is implicit; the stored constant is `0x04C11DB7`.

**Bitwise CRC-32:**

```c
uint32_t crc32_bitwise(const uint8_t *data, size_t len) {
    uint32_t crc = 0xFFFFFFFFu;
    for (size_t i = 0; i < len; i++) {
        crc ^= data[i];
        for (int j = 0; j < 8; j++) {
            if (crc & 1)
                crc = (crc >> 1) ^ 0xEDB88320u;  // reflected polynomial
            else
                crc >>= 1;
        }
    }
    return crc ^ 0xFFFFFFFFu;
}
```

The constant `0xEDB88320` is `0x04C11DB7` bit-reversed; the reflected form lets us shift right (which is faster on most CPUs with the LSB conventionally on the right).

**Table-driven CRC-32:**

Precompute a 256-entry table of the CRC of each possible byte. Each iteration becomes one lookup and one shift:

```c
static uint32_t crc_table[256];

void crc32_init(void) {
    for (uint32_t i = 0; i < 256; i++) {
        uint32_t c = i;
        for (int j = 0; j < 8; j++)
            c = (c >> 1) ^ ((c & 1) ? 0xEDB88320u : 0);
        crc_table[i] = c;
    }
}

uint32_t crc32(const uint8_t *data, size_t len) {
    uint32_t crc = 0xFFFFFFFFu;
    for (size_t i = 0; i < len; i++)
        crc = crc_table[(crc ^ data[i]) & 0xFFu] ^ (crc >> 8);
    return crc ^ 0xFFFFFFFFu;
}
```

The 1 KiB table makes this ~8× faster than the bitwise version. SSE 4.2 introduced `CRC32` instructions for hardware acceleration, though they use a different polynomial (`0x1EDC6F41`, "Castagnoli", aka CRC-32C, used by SCTP and BTRFS).

### CRC error-detection guarantees

A CRC-n with a "good" polynomial detects:
- All single-bit errors
- All double-bit errors (within ~2^n−1 bit distance, the polynomial period)
- All burst errors of length ≤ n
- Any odd number of bit errors (if the polynomial includes the (x+1) factor)

CRC-32 detects all error patterns of weight 1, 2, 3 in messages up to 2^31 − 1 bits, plus all bursts up to 32 bits. It does NOT detect adversarial modifications: an attacker can choose corrupt data with the correct CRC. Use HMAC or a true cryptographic MAC for authenticity.

### Adler-32 (zlib)

Adler-32 is faster than CRC-32 but weaker. It splits the running sum into two 16-bit values:

```
A = 1 + sum of bytes      (mod 65521)
B = sum over all i of (1 + sum of bytes[0..i])  (mod 65521)
checksum = (B << 16) | A
```

The modulus 65521 is the largest prime less than 2^16. Adler-32 is used in zlib's compressed-stream wrapper.

### Internet checksum (RFC 1071)

Used by IPv4, ICMP, UDP, TCP. A 16-bit one's complement of the one's complement sum:

```c
uint16_t internet_checksum(const uint16_t *data, size_t count) {
    uint32_t sum = 0;
    for (size_t i = 0; i < count; i++)
        sum += data[i];
    while (sum >> 16)
        sum = (sum & 0xFFFF) + (sum >> 16);  // fold carries
    return (uint16_t)~sum;
}
```

The receiver computes the same sum including the checksum field; the result is 0xFFFF (in one's complement, an "all-ones zero") if uncorrupted.

The Internet checksum detects all single-bit errors and most double-bit errors but fails on byte-swaps and value-preserving rearrangements. It is much weaker than CRC-32 but cheap to compute incrementally (TCP checksum offload requires this property: routers can update the checksum after rewriting a TTL or NAT'd port without recomputing from scratch).

### SHA-256

A cryptographic hash. 256 bits of output. Detects any modification with probability ≈ 1 − 2^−128 (preimage resistance) and 1 − 2^−256 (collision resistance, but birthday bound is 2^128 work).

CRC-32 detects accidental errors in transit. SHA-256 detects deliberate tampering. They serve different purposes:

```
purpose                           use
--------------------------------  -----------
detect bit flips on a wire        CRC
detect bit flips in storage       CRC + replication
detect accidental file corruption MD5/SHA-1 (legacy) or SHA-256 (modern)
detect tampering by adversary     HMAC-SHA-256, Ed25519 sig
authenticate a sender             public-key signature
deduplicate identical files       SHA-256 (fingerprint)
```

## Information theory

Claude Shannon's 1948 *A Mathematical Theory of Communication* founded the field. The fundamental quantity is entropy.

### Entropy

For a discrete random variable X with probabilities p(x):

```
H(X) = -Σ p(x) · log_2 p(x)   bits
```

Entropy measures the expected information content (in bits) per symbol. A fair coin (p=½, p=½) has H = 1 bit. A biased coin with p(heads) = 0.99 has H ≈ 0.081 bits — almost nothing to learn from each flip.

For uniformly distributed X over n outcomes: H(X) = log_2 n. For an English text approximated as a 27-symbol alphabet (26 letters + space): H ≈ 4.7 bits per character if uniform; actual entropy of English (with statistical structure) is ≈ 1.0–1.5 bits per character (Shannon's experiments).

### Source coding theorem

For an iid source X with entropy H(X), the average length of any uniquely-decodable code is at least H(X). And there exist codes with average length < H(X) + 1.

This bounds the compression ratio: a perfectly random source cannot be compressed at all; structured sources can be compressed down to their entropy.

### Huffman coding

Optimal among prefix codes: builds a binary tree by repeatedly merging the two least-probable symbols. The codeword for each symbol is the path from root to leaf.

For symbols with probabilities (0.4, 0.3, 0.2, 0.1):

```
  Step: merge 0.1 + 0.2 → 0.3
    {0.4, 0.3, 0.3}
  Step: merge 0.3 + 0.3 → 0.6
    {0.6, 0.4}
  Step: merge 0.6 + 0.4 → 1.0  (root)

  Tree:                Codes:
       1.0              A (0.4) → 0
       / \              B (0.3) → 11
     0.4 0.6            C (0.2) → 100
      A  / \            D (0.1) → 101
       0.3 0.3
        B  / \
          C  D

  Average length: 0.4(1) + 0.3(2) + 0.2(3) + 0.1(3) = 1.9 bits per symbol.
  Entropy: H = -(0.4 log 0.4 + 0.3 log 0.3 + 0.2 log 0.2 + 0.1 log 0.1) ≈ 1.85 bits.
  Within 0.05 bits of optimal.
```

Huffman is optimal but not perfect: it codes one symbol at a time. Arithmetic coding can achieve closer-to-entropy by jointly coding sequences. ANS (Asymmetric Numeral Systems) achieves both speed and entropy-bound compression.

### LZ77

Lempel-Ziv 1977 — the basis for DEFLATE (gzip, zlib, PNG), LZ4, Snappy, Zstandard. The compressor outputs:

```
- a literal byte: emit it directly
- a back-reference (distance, length): "copy <length> bytes from <distance> bytes ago"
```

```
input:    "abcabcabc"

LZ77 output:
  literal 'a'
  literal 'b'
  literal 'c'
  copy(distance=3, length=6)   ← copies "abcabc" sliding-window style

compressed: 4 ops vs 9 input bytes
```

The sliding window (typically 32 KiB) limits how far back a match can reach. DEFLATE further encodes the LZ77 output with a Huffman code on the literals and back-references.

### LZW

Lempel-Ziv-Welch 1984 — used by GIF, the original Unix `compress`, TIFF. Builds a dictionary of seen substrings; outputs codes that reference dictionary entries. Patented for many years (Unisys), now expired.

## Bit manipulation tricks

A collection of idioms used in low-level code, kernel hot paths, and competitive programming. Many are explained in Hacker's Delight (Henry Warren).

### Swap without temporary

```c
a ^= b;   // a' = a ^ b
b ^= a;   // b' = b ^ (a ^ b) = a
a ^= b;   // a'' = (a ^ b) ^ a = b
```

After the three XORs, a and b have swapped values. Note: this fails if a and b are aliases of the same variable (the first XOR zeros it). Modern compilers do not benefit — they emit `MOV` pairs or use `XCHG`.

### Parity

The parity of x (1 if odd number of 1 bits, 0 if even):

```c
int parity(uint32_t x) {
    x ^= x >> 16;
    x ^= x >> 8;
    x ^= x >> 4;
    x ^= x >> 2;
    x ^= x >> 1;
    return x & 1;
}
```

Or: `popcount(x) & 1`.

### Power-of-two test

```c
bool is_power_of_two(uint32_t x) {
    return x && !(x & (x - 1));
}
```

`is_power_of_two(0) = false` (zero is excluded by convention; zero is not 2^k for any k).

### Round up to next power of two

```c
uint32_t round_up_pow2(uint32_t x) {
    if (x == 0) return 1;
    x--;
    x |= x >> 1;
    x |= x >> 2;
    x |= x >> 4;
    x |= x >> 8;
    x |= x >> 16;
    return x + 1;
}
```

### Integer log2 via leading zeros

```c
int ilog2(uint32_t x) {
    return 31 - __builtin_clz(x);  // undefined if x == 0
}
```

Hardware: x86's `LZCNT` (BMI1, 2013), or `BSR` (Bit Scan Reverse, since 386); ARM's `CLZ`.

### Reverse bits in a 32-bit word

```c
uint32_t reverse(uint32_t x) {
    x = ((x & 0xAAAAAAAAu) >> 1) | ((x & 0x55555555u) << 1);  // swap pairs
    x = ((x & 0xCCCCCCCCu) >> 2) | ((x & 0x33333333u) << 2);  // swap nibbles
    x = ((x & 0xF0F0F0F0u) >> 4) | ((x & 0x0F0F0F0Fu) << 4);
    x = ((x & 0xFF00FF00u) >> 8) | ((x & 0x00FF00FFu) << 8);
    x = (x >> 16) | (x << 16);
    return x;
}
```

ARM has `RBIT` (single-instruction). x86 lacks a direct instruction.

### Sign function (no branches)

```c
int sign(int32_t x) {
    return (x > 0) - (x < 0);  // -1, 0, or +1
}
```

### Absolute value (no branches)

```c
int32_t abs(int32_t x) {
    int32_t mask = x >> 31;          // -1 if negative, 0 if non-negative
    return (x ^ mask) - mask;        // negate iff mask is -1
}
```

### Min/max (no branches)

```c
int32_t min(int32_t a, int32_t b) {
    return b ^ ((a ^ b) & -(a < b));
}

int32_t max(int32_t a, int32_t b) {
    return a ^ ((a ^ b) & -(a < b));
}
```

Modern compilers emit `CMOV` for the branchy version, often beating these tricks.

### Saturating add (clamp to UINT32_MAX)

```c
uint32_t add_sat(uint32_t a, uint32_t b) {
    uint32_t s = a + b;
    return (s < a) ? UINT32_MAX : s;  // overflow detected by wrap
}
```

ARM has `UQADD` (unsigned saturating add); some x86 SIMD instructions provide saturated arithmetic.

### Detect zero byte in a 32-bit word

Used in C's `strlen` and `strchr`:

```c
bool has_zero_byte(uint32_t x) {
    return ((x - 0x01010101u) & ~x & 0x80808080u) != 0;
}
```

When a byte of x is 0, subtracting 1 borrows, setting the byte to 0xFF; ANDing with `~x` (which has 0xFF in that byte) preserves it; ANDing with 0x80808080 isolates the high bit per byte.

### Branchless conditional (predication)

```c
int select(int cond, int a, int b) {
    int mask = -!!cond;            // all-ones if cond, all-zeros if not
    return (a & mask) | (b & ~mask);
}
```

Useful for constant-time cryptographic code that must avoid timing side channels.

### XOR linked list

Memory-saving doubly-linked list using one pointer field per node:

```
node->link = prev XOR next
```

Walk forward by XORing the previous node's address; walk backward symmetrically. Has fallen out of favor (cache-unfriendly, debugger-hostile, breaks memory-safety analysis).

## Worked examples

### Encode -42 as int32 (two's complement)

```
42  = 0x0000002A = 0b 00000000 00000000 00000000 00101010
~42 =             0b 11111111 11111111 11111111 11010101
-42 = ~42 + 1 =   0b 11111111 11111111 11111111 11010110
    = 0xFFFFFFD6
```

Verify: -42 unsigned = 4,294,967,254 = 2^32 - 42. ✓

In memory (little-endian):

```
addr 0    1    2    3
byte 0xD6 0xFF 0xFF 0xFF
```

### Encode 3.14159 as float64

3.14159 in binary: 11.001001000011111101101010100010001000010110100011...

Normalize: 1.10010010000111111011010101000100010000101101000110000100011010011 × 2^1

Exponent: 1 + 1023 = 1024 = 0x400 = 0b 100 0000 0000

Mantissa (52 bits, the part after the implicit leading 1, rounded):

```
1001 0010 0001 1111 1011 0101 0100 0010 0010 0001 0110 0100 0000   (rounded from 0011...)
```

Hmm — let me redo this with more care. The exact π is `0x400921FB54442D18`. The exact 3.14159 is slightly different; let's encode 3.14159 exactly:

```python
>>> import struct
>>> hex(struct.unpack('<Q', struct.pack('<d', 3.14159))[0])
'0x400921f9f01b866e'
```

Decoding `0x400921F9F01B866E`:

```
sign:     0
exp:      0x400 = 1024 → unbiased = 1
mantissa: 0x921F9F01B866E

   bits: 1001 0010 0001 1111 1001 1111 0000 0001 1011 1000 0110 0110 1110

value = 1 + (mantissa / 2^52)
      = 1 + 2570631753828974 / 4503599627370496
      ≈ 1.5707950000000003

× 2^1 ≈ 3.141590000000001  (matches 3.14159 to all stored bits)
```

In memory (little-endian) the byte order is:

```
addr 0    1    2    3    4    5    6    7
byte 0x6E 0x86 0x1B 0xF0 0xF9 0x21 0x09 0x40
```

### UTF-8 encode "A€𝒜"

Three codepoints: U+0041, U+20AC, U+1D49C. Per the rules above:

```
U+0041:    1 byte    →  0x41
U+20AC:    3 bytes   →  0xE2 0x82 0xAC
U+1D49C:   4 bytes   →  0xF0 0x9D 0x92 0x9C

Total: 8 bytes:  41 E2 82 AC F0 9D 92 9C
```

The same string as a Python `str` is 3 codepoints; the same string as `bytes` (UTF-8) is 8 bytes. UTF-16: 4 code units (8 bytes) since 𝒜 needs a surrogate pair.

### CRC-32 of "abc"

Using the IEEE 802.3 polynomial (reflected `0xEDB88320`), the CRC-32 of `"abc"` (bytes 0x61 0x62 0x63) is:

```
0x352441C2
```

Verification:

```python
>>> import zlib
>>> hex(zlib.crc32(b'abc'))
'0x352441c2'
```

The bitwise computation (init = 0xFFFFFFFF, final XOR with 0xFFFFFFFF):

```
input bytes:  0x61, 0x62, 0x63

For 0x61:
  crc ^= 0x61
  shift+conditional XOR 8 times against 0xEDB88320
  ...
  (full computation produces an intermediate)

After all three bytes:
  crc = 0xCADBBE3D
  final ^= 0xFFFFFFFF → 0x352441C2
```

### Parity bit

The 7-bit ASCII for 'A' is 0x41 = 0b 1000001 — three set bits, odd.

Even-parity adds a parity bit such that total set bits is even:

```
'A' with even parity: 0b 11000001 = 0xC1   (added a 1 to make 4 set bits)
```

Odd-parity adds a parity bit such that total set bits is odd:

```
'A' with odd parity:  0b 01000001 = 0x41   (added a 0; was already 3, odd)
```

A single bit error flips parity; the receiver detects mismatch but cannot correct (insufficient redundancy). Hamming codes, BCH codes, Reed-Solomon, and LDPC codes provide error correction (single-bit or burst).

### Integer overflow demonstration

```c
int32_t a = INT32_MAX;     // 0x7FFFFFFF =  2147483647
int32_t b = 1;
int32_t c = a + b;          // 0x80000000 = -2147483648 (signed overflow → UB)

uint32_t u = UINT32_MAX;    // 0xFFFFFFFF = 4294967295
uint32_t v = 1;
uint32_t w = u + v;          // 0x00000000 = 0  (unsigned wraps; well-defined)
```

In C, signed overflow is undefined behavior (UB) — compilers can assume it never happens and optimize accordingly. This is why `for (int i = 0; i < N; i++)` is safer than relying on overflow loops. Use unsigned types when wraparound is desired (counters, hashes), `__builtin_add_overflow` when you need an explicit overflow check, or saturating arithmetic when clamping is desired.

```c
int32_t safe_add(int32_t a, int32_t b, int32_t *out) {
    int32_t r;
    if (__builtin_add_overflow(a, b, &r)) return -1;  // overflow
    *out = r;
    return 0;
}
```

GCC and Clang provide `__builtin_{add,sub,mul}_overflow` for s32, s64, u32, u64.

### Hex to int parsing

```c
int hex_digit(char c) {
    if (c >= '0' && c <= '9') return c - '0';
    if (c >= 'a' && c <= 'f') return c - 'a' + 10;
    if (c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

uint32_t parse_hex32(const char *s) {
    uint32_t r = 0;
    for (; *s; s++) {
        int d = hex_digit(*s);
        if (d < 0) return UINT32_MAX;  // error
        r = (r << 4) | (uint32_t)d;
    }
    return r;
}
```

Each hex digit contributes 4 bits, shifted into the accumulator from the right. 8 hex digits fully fill a 32-bit value.

### Float bit-pattern manipulation: `nextafter` via integer increment

A subtle but useful property: float bits, interpreted as integers (with sign-magnitude convention), are monotone with the float value (for non-negative floats; for negative floats, the order is reversed).

```c
float nextafter_pos(float x) {
    // For x > 0, the next representable float is one bit pattern higher.
    uint32_t bits;
    memcpy(&bits, &x, 4);
    bits++;  // increment as integer
    float r;
    memcpy(&r, &bits, 4);
    return r;
}
```

This works because the IEEE 754 layout was designed for it: incrementing the bit pattern of a positive normal float gives the next representable value. Subnormals and zero connect smoothly (subnormal max + 1 bit = smallest normal). Use `nextafterf` from `<math.h>` for the standardized version; the trick above is for understanding.

## Summary

A bit is the unit; everything compounds from there. Two's complement makes signed arithmetic free; IEEE 754 makes real-number arithmetic finite-precision but well-defined; UTF-8 makes text universal at the cost of variable width. CRC catches accidents; cryptographic hashes catch adversaries. Bit tricks compress branches into arithmetic. Endianness is a portability tax. Information theory bounds compression and channel capacity.

The shape of every program is determined by these encodings. The bugs you will spend the most time chasing — silent overflow, NaN propagation, lost precision, `0.1 + 0.2`, mojibake, surrogate pair mishandling, off-by-one in mask construction, subtle CRC misuse, byte-order assumptions — all live here. Mastering this layer pays compounding returns: a compiler engineer, a kernel hacker, a cryptographer, a games programmer, a database internals engineer, an embedded developer, and a systems verifier all owe their daily bread to the math in this document.

The data is bits. The interpretation is yours.

## See Also

- [cs-theory/distributed-systems](../cs-theory/distributed-systems.md) — protocol-level checksums, replication, byte-order
- [cs-theory/information-theory](../cs-theory/information-theory.md) — entropy, source coding, channel capacity
- [cs-theory/big-o-complexity](../cs-theory/big-o-complexity.md) — bit-cost analysis, complexity of arithmetic
- [data-formats/regex](../../sheets/data-formats/regex.md) — UTF-8 byte-level pattern matching
- [languages/c](../../sheets/languages/c.md) — fixed-width integer types, bitwise operators, overflow semantics
- [ramp-up/binary-numbering-eli5](../../sheets/ramp-up/binary-numbering-eli5.md) — narrative companion
- [ramp-up/assembly-eli5](../../sheets/ramp-up/assembly-eli5.md) — bit operations as machine instructions

## References

- IEEE 754-2019, *IEEE Standard for Floating-Point Arithmetic*
- RFC 3629, *UTF-8, a transformation format of ISO 10646* — Yergeau, 2003
- RFC 4648, *The Base16, Base32, and Base64 Data Encodings* — Josefsson, 2006
- RFC 1071, *Computing the Internet Checksum* — Braden, Borman, Partridge, 1988
- ITU-T V.42, ISO 3309, IEEE 802.3 — CRC-32 polynomial specifications
- Knuth, *The Art of Computer Programming, Vol 4A: Combinatorial Algorithms* — bit-level enumeration, Gray codes
- Henry S. Warren Jr., *Hacker's Delight* (2nd ed., 2012) — the canonical reference for bit manipulation
- Charles Petzold, *Code: The Hidden Language of Computer Hardware and Software* (2000) — accessible foundations
- Donald Goldberg, *What Every Computer Scientist Should Know About Floating-Point Arithmetic* — ACM Computing Surveys, 1991
- William Kahan, *Lecture Notes on the Status of IEEE Standard 754 for Binary Floating-Point Arithmetic* (1997)
- Claude Shannon, *A Mathematical Theory of Communication* — Bell System Technical Journal, 1948
- Lempel & Ziv, *A Universal Algorithm for Sequential Data Compression* — IEEE Transactions on Information Theory, 1977
- David Huffman, *A Method for the Construction of Minimum-Redundancy Codes* — Proceedings of the IRE, 1952
- Unicode Consortium, *The Unicode Standard, Version 15.1* (2023)
- Intel 64 and IA-32 Architectures Software Developer's Manual, Volume 2 — instruction reference
- ARM Architecture Reference Manual (ARMv8-A) — bitfield instructions, `RBIT`, `CLZ`
