# Binary and Number Systems (From Counting to IEEE 754)

A tiered guide to number systems used in computing.

## ELI5

Computers only have two fingers: 0 and 1. Like a light switch -- on or off.

```
Your Counting (Decimal)     Computer's Counting (Binary)
0                           0         all lights off
1                           1         one light on
2                           10        carried! like going from 9 to 10
3                           11        two lights on
4                           100       carried again
5                           101
6                           110
7                           111       three lights on
8                           1000      big carry! like 999 -> 1000
10                          1010
255                         11111111  eight lights ALL on -- max byte
```

Each on/off is called a **bit**. Eight bits together make a **byte**.

### Hexadecimal -- The Shortcut

Binary gets long. Hex uses 16 digits (0-9, then A-F) as a shortcut:

```
Decimal    Binary     Hex
0          0000       0
9          1001       9
10         1010       A
11         1011       B
15         1111       F
16         10000      10
255        11111111   FF
```

Each hex digit = exactly 4 bits. Two hex digits = one byte. `0xFF = 255`.

When you see `0x` before a number, it means hexadecimal.

## Middle School

### Binary <-> Decimal Conversion

**Decimal to Binary (Division Method):**

```
Convert 200 to binary:
  200 / 2 = 100  remainder 0
  100 / 2 = 50   remainder 0
   50 / 2 = 25   remainder 0
   25 / 2 = 12   remainder 1
   12 / 2 = 6    remainder 0
    6 / 2 = 3    remainder 0
    3 / 2 = 1    remainder 1
    1 / 2 = 0    remainder 1

Read bottom to top: 11001000 = 200
```

**Binary to Decimal (Place Values):**

```
Position:   7    6    5    4    3    2    1    0
Power of 2: 128  64   32   16   8    4    2    1
Bits:       1    1    0    0    1    0    0    0
Value:      128  64   0    0    8    0    0    0

128 + 64 + 8 = 200
```

### Hex Conversion

```
Split binary into groups of 4 bits:
  Binary:  1100 1000
  Hex:      C    8     = 0xC8 = 200

IP address in three forms:
  Decimal:  192.168.1.100
  Binary:   11000000.10101000.00000001.01100100
  Hex:      C0.A8.01.64
```

### Bytes and Data Sizes

```
1 byte   = 8 bits         = values 0-255
1 KB     = 1024 bytes     = 2^10
1 MB     = 1024 KB        = 2^20
1 GB     = 1024 MB        = 2^30
1 TB     = 1024 GB        = 2^40
```

### ASCII Basics

```
Character  Decimal  Hex   Binary
'A'        65       0x41  01000001
'a'        97       0x61  01100001
'0'        48       0x30  00110000
' '        32       0x20  00100000
```

Uppercase to lowercase: add 32 (flip bit 5). `'A' + 32 = 'a'`.

### Simple Binary Addition

```
  0 + 0 = 0
  0 + 1 = 1
  1 + 0 = 1
  1 + 1 = 10  (0, carry 1)

Example:
    0110  (6)
  + 0011  (3)
  ------
    1001  (9)
```

## High School

### Octal (Base 8)

```
Uses digits 0-7. Each octal digit = 3 bits.
Prefix: 0o or leading 0 (C language)

Decimal  Binary     Octal
8        001 000    10
64       001 000 000  100
255      011 111 111  377
511      111 111 111  777

Common use: Unix file permissions
  chmod 755 = rwxr-xr-x
  7 = 111 = rwx    5 = 101 = r-x    5 = 101 = r-x
```

### Two's Complement (Signed Integers)

```
For an n-bit number, the most significant bit is the sign bit.
  0 = positive, 1 = negative

To negate a number: invert all bits, add 1.

8-bit examples:
  Binary      Unsigned  Signed (two's complement)
  00000000    0         0
  00000001    1         1
  01111111    127       127       (max positive)
  10000000    128       -128      (min negative)
  10000001    129       -127
  11111111    255       -1

Range for n bits: -2^(n-1) to 2^(n-1) - 1
  8-bit:   -128 to 127
  16-bit:  -32768 to 32767
  32-bit:  -2147483648 to 2147483647
  64-bit:  -9223372036854775808 to 9223372036854775807
```

**Why two's complement works:** addition needs no special sign handling.

```
  5 + (-3) using two's complement (8-bit):
    00000101    (5)
  + 11111101    (-3: invert 00000011, add 1)
  ----------
  1 00000010    (2, ignore carry-out)
```

### Binary Arithmetic

```
Subtraction: a - b = a + (~b + 1)   (add two's complement of b)

Multiplication (shift and add):
    1011  (11)
  x 0110  (6)
  ------
    0000  (1011 * 0)
   1011   (1011 * 1, shifted left 1)
  1011    (1011 * 1, shifted left 2)
 0000     (1011 * 0, shifted left 3)
  --------
  1000010  (66)
```

### Bitwise Operations

```
Operation  Symbol  Rule                     Example
AND        &       Both 1 -> 1              1010 & 1100 = 1000
OR         |       Either 1 -> 1            1010 | 1100 = 1110
XOR        ^       Exactly one 1 -> 1       1010 ^ 1100 = 0110
NOT        ~       Flip all bits            ~1010 = 0101
Left Shift <<      Shift left, fill 0       0001 << 2 = 0100
Right Shift>>      Shift right              1000 >> 2 = 0010
```

**Practical uses:**

```
Masking:       x & 0xFF          extract lowest byte
Setting bits:  x | (1 << n)      set bit n
Clearing bits: x & ~(1 << n)     clear bit n
Toggling bits: x ^ (1 << n)      toggle bit n
Testing bits:  x & (1 << n)      nonzero if bit n is set
Multiply by 2: x << 1
Divide by 2:   x >> 1
Check odd/even: x & 1            1 = odd, 0 = even
Swap (no temp): a ^= b; b ^= a; a ^= b
```

### Hex in Memory

```
Memory dump (typical debugger output):
  0x7fff5c00:  48 65 6c 6c 6f 20 57 6f  72 6c 64 00
               H  e  l  l  o     W  o   r  l  d  \0

Pointer sizes:
  32-bit: 0x00000000 - 0xFFFFFFFF (4 GB address space)
  64-bit: 0x0000000000000000 - 0xFFFFFFFFFFFFFFFF (16 EB)
```

### Connection to Subnetting

```
Subnet mask = AND operation:
  IP:       192.168.1.100  = 11000000.10101000.00000001.01100100
  Mask:     255.255.255.0  = 11111111.11111111.11111111.00000000
  AND:      192.168.1.0    = 11000000.10101000.00000001.00000000
                             (network ID extracted)
```

## College

### IEEE 754 Floating Point

```
Format:  (-1)^sign * 1.mantissa * 2^(exponent - bias)

Single precision (32-bit / float):
  [1 sign][8 exponent][23 mantissa]
  Bias: 127
  Range: ~1.18e-38 to 3.40e+38
  Precision: ~7 decimal digits

Double precision (64-bit / double):
  [1 sign][11 exponent][52 mantissa]
  Bias: 1023
  Range: ~2.23e-308 to 1.80e+308
  Precision: ~15-16 decimal digits

Half precision (16-bit):
  [1 sign][5 exponent][10 mantissa]
  Bias: 15
```

**Decoding example:**

```
Binary: 0 10000010 10110000000000000000000

Sign:     0 -> positive
Exponent: 10000010 = 130, 130 - 127 = 3
Mantissa: 1.10110 (implicit leading 1)

Value: 1.10110 * 2^3 = 1101.10 = 13.5
```

### Special Values

```
Category        Exponent     Mantissa     Value
Zero            all 0s       all 0s       +0 or -0 (sign bit)
Denormalized    all 0s       nonzero      (-1)^s * 0.m * 2^(-126)
Normalized      1 to 254     any          (-1)^s * 1.m * 2^(e-127)
Infinity        all 1s       all 0s       +inf or -inf
NaN             all 1s       nonzero      Not a Number

NaN types:
  Quiet NaN (qNaN):      mantissa bit 22 = 1, propagates silently
  Signaling NaN (sNaN):  mantissa bit 22 = 0, raises exception
```

### Floating-Point Errors

```
Representation error:
  0.1 in binary = 0.0001100110011... (repeating)
  float(0.1) = 0.100000001490116...
  double(0.1) = 0.1000000000000000055511151231257827021181583404541015625

Catastrophic cancellation:
  x = 1.0000001, y = 1.0000000
  x - y = 0.0000001   (only 1 significant digit from 7)

Absorption:
  1.0e20 + 1.0 = 1.0e20   (1.0 is below the precision threshold)

Machine epsilon:
  float:   2^-23 ~ 1.19e-7
  double:  2^-52 ~ 2.22e-16
```

### Comparing Floats

```
WRONG:  if (a == b)
RIGHT:  if (fabs(a - b) < epsilon)
BETTER: if (fabs(a - b) <= max(1.0, fabs(a), fabs(b)) * epsilon)

Relative epsilon comparison (ULP-aware):
  |a - b| <= epsilon * max(|a|, |b|)
```

### Fixed-Point Arithmetic

```
Q notation: Qm.n = m integer bits, n fractional bits
  Q8.8:  8 integer, 8 fractional, 16 bits total
  Q1.15: 1 sign, 15 fractional (audio DSP)

Value = raw_integer / 2^n

Example Q8.8:
  0x0180 = 384 / 256 = 1.5
  0xFF80 = -128 / 256 = -0.5 (signed)

Advantages: deterministic, no rounding surprises, fast on integer-only hardware
Disadvantages: fixed range and precision, manual scaling
```

### BCD (Binary-Coded Decimal)

```
Each decimal digit stored as 4 bits:

  Decimal 92 = 1001 0010 (BCD)
             vs 01011100 (pure binary)

Packed BCD:  two digits per byte   (92 = 0x92)
Unpacked BCD: one digit per byte   (9 = 0x09, 2 = 0x02)

Use cases: financial calculations (exact decimal), clock chips, COBOL
```

### Arbitrary Precision

```
Libraries: GMP (C), BigInteger (Java), big.Int (Go), int (Python 3)

Representation: array of machine words (limbs)
  123456789012345678901234567890 stored as:
  [0x4B3B4CA85A86C47A, 0x098A224000000000, ...] (base 2^64 limbs)

Karatsuba multiplication: O(n^1.585) vs naive O(n^2)
Schoolbook addition: O(n) -- linear in limb count
```

### Endianness

```
Value: 0x01020304

Big-endian (network byte order):
  Address:  0x00  0x01  0x02  0x03
  Byte:     01    02    03    04
  (most significant byte at lowest address)

Little-endian (x86, ARM default):
  Address:  0x00  0x01  0x02  0x03
  Byte:     04    03    02    01
  (least significant byte at lowest address)

Conversion functions:
  C:      htons(), htonl(), ntohs(), ntohl()
  Go:     binary.BigEndian, binary.LittleEndian
  Python: struct.pack('>I', val), int.from_bytes(b, 'big')
```

## Tips

- `0b` prefix for binary literals (most languages): `0b1010 == 10`
- `0x` prefix for hex: `0xFF == 255`
- `0o` prefix for octal: `0o777 == 511`
- Powers of 2 to memorize: 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768, 65536
- `2^10 = 1024 ~ 10^3` -- useful for quick estimates (2^32 ~ 4.3 billion)
- Never compare floats with `==`. Always use epsilon-based comparison.
- When doing financial math, use integers (cents) or decimal types, never float/double.
- XOR swap trick is a curiosity -- modern compilers optimize regular swaps better.
- Two's complement overflow is undefined behavior in C (signed); wraps in unsigned.

## See Also

- boolean-algebra
- subnetting
- ascii
- unicode
- algorithm-analysis

## References

- IEEE 754-2019, "IEEE Standard for Floating-Point Arithmetic"
- Goldberg, D. "What Every Computer Scientist Should Know About Floating-Point Arithmetic" (1991), ACM Computing Surveys
- Knuth, D. E. "The Art of Computer Programming, Vol. 2: Seminumerical Algorithms" (3rd ed., Addison-Wesley, 1997)
- Patterson, D. & Hennessy, J. "Computer Organization and Design" (6th ed., Morgan Kaufmann, 2020)
- Parhami, B. "Computer Arithmetic: Algorithms and Hardware Designs" (2nd ed., Oxford, 2010)
- RFC 791 -- Internet Protocol (IPv4 address structure)
- RFC 4291 -- IP Version 6 Addressing Architecture
