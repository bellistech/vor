# Binary, Hex, IP Numbering — ELI5 (How Computers Count and Address Each Other)

> Computers only have two fingers, so they count in light switches; we then borrow that idea to give every computer in the world a house number.

## Prerequisites

(none — counting on fingers is enough)

This sheet is the second stop after the kernel ELI5. You do not need to know anything about computers to read it. You do not need to know what binary is. You do not need to know what an IP address is. You do not need to know what a subnet is. You do not need to remember any of the math you might have done in school. By the end of this sheet you will be able to:

- count in binary on your fingers without making a face;
- read a number written in hex (the weird thing with letters in it) and turn it into a normal number;
- look at an IP address like `192.168.1.100` and understand what each piece is doing;
- look at a "slash 24" written next to an address and know what that means;
- split a network into pieces with a pencil and paper, and check your work in the terminal;
- do all of the above for IPv6, the new long version of an IP address.

If a word feels weird, look it up in the **Vocabulary** table near the bottom. Every weird word in this sheet is in that table with a one-line plain-English definition. If a section is not clicking, jump to **Hands-On** and run a few commands first; sometimes seeing the answer come out of your terminal makes the explanation make sense.

If you see a `$` at the start of a line in a code block, that means "type the rest of this line into your terminal." You do not type the `$`. The lines underneath that don't have a `$` are what your computer prints back at you. We call that "output."

This sheet covers the **counting and address-math side** of networking. The companion sheet `cs ramp-up ip-eli5` covers the protocol side — how packets actually get pushed across a wire, who answers when. This sheet is the math. That sheet is the road system. Read this one first.

## What Even Are Number Systems?

### You have ten fingers

Hold up your hands. Count: 0, 1, 2, 3, 4, 5, 6, 7, 8, 9. You ran out of fingers. Now what? You write a "1" in the next column over and start again with the fingers: 10, 11, 12, ... 19. Out of fingers again. Carry: 20. And so on.

That is decimal. **Base 10.** It is called base 10 because there are ten different one-digit numbers (0 through 9), and every time you carry over, you multiply by ten. The "1" in "10" is really "one ten." The "1" in "100" is "one hundred," which is "ten tens." Each new column to the left is worth ten times the column to its right.

Why ten? Because you have ten fingers. That's it. There is no deeper reason. Some old cultures used base 12 or base 60 (you can still see the leftovers in clocks and angles — 12 hours, 60 minutes, 360 degrees). But almost everybody in the world today writes numbers in base 10 because that is how many fingers we have.

### A computer has two fingers

A computer is made of switches. A switch is either ON or OFF. There is no in-between. Inside a chip there are billions of tiny switches called **transistors**, and every single one of them is either ON or OFF. There is no half-on, no maybe-on, no "kind of on if you squint." Just ON or OFF.

So a computer has two fingers. That is all it can count with: ON and OFF. We call them 1 and 0. That is **binary.** **Base 2.** It is called base 2 because there are only two one-digit numbers (0 and 1), and every time you carry over you multiply by two.

A binary "1" is one. A binary "10" is two. A binary "100" is four. A binary "1000" is eight. Each new column to the left is worth twice the column to its right.

### Why do we even have other bases?

Decimal is great for humans. Binary is great for computers. But binary numbers get long fast. The number 200, which is just three digits in decimal, is **eight digits** in binary: 11001000. Imagine reading a phone number written in binary. Imagine reading a credit card number written in binary. Your eyes would melt out of your skull.

So computer people invented two more bases that are easier on human eyes:

- **Octal (base 8)** — eight digits (0–7). Each octal digit is exactly three binary bits. Used to be popular before hex took over. Still pops up in Linux file permissions like `chmod 755`.
- **Hexadecimal (base 16)** — sixteen digits (0–9 plus A–F). Each hex digit is exactly four binary bits. This is the one you'll see all the time. We call it **hex** for short.

We will spend the rest of this sheet using binary and hex. Octal is a side character, mentioned in passing.

### Why each base is useful

Decimal is what you already know. You count in it. You shop in it. You read prices in it.

Binary is what every computer actually uses inside itself. Every single thing a computer ever does — every picture, every sound, every number, every letter — is stored as a long string of binary digits. The whole machine is built out of switches.

Hex is the comfortable middle. Hex is what programmers use to write down binary numbers without going blind. Two hex digits is exactly one byte (eight bits). Eight hex digits is exactly thirty-two bits — exactly the size of an IPv4 address. Thirty-two hex digits is exactly one hundred twenty-eight bits — exactly the size of an IPv6 address. So hex shows up everywhere in networking.

Octal is what you'll see for Unix file permissions and almost nowhere else.

## Binary (Base 2)

### Counting in binary

Let's just count in binary, slowly, side by side with decimal:

```
DECIMAL   BINARY      WHAT IT MEANS
0         0           all switches off
1         1           one switch on
2         10          carried — first switch off, second switch on
3         11          two switches on
4         100         carried again — third switch on
5         101
6         110
7         111         three switches on
8         1000        big carry — fourth switch on
9         1001
10        1010
11        1011
12        1100
13        1101
14        1110
15        1111        four switches on
16        10000       big carry — fifth switch on
...
255       11111111    eight switches on (we'll see why this matters)
256       100000000   carried — ninth switch on
```

Notice how every time you reach a power of two (1, 2, 4, 8, 16, 32, 64, 128, 256, ...), a new column lights up.

### Place values

In decimal, each column is worth ten times more than the one to its right. The columns are 1, 10, 100, 1000, 10000.

In binary, each column is worth two times more than the one to its right. The columns are 1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024.

Side by side:

```
DECIMAL place values:  ... 100000   10000   1000    100     10      1
BINARY  place values:  ... 32       16      8       4       2       1
```

To read a binary number, you just look at which columns have a 1, then add up those columns.

```
Binary:    1   1   0   0   1   0   0   0
Column:    128 64  32  16  8   4   2   1
Lit:       Y   Y   .   .   Y   .   .   .
                                          add: 128 + 64 + 8 = 200
```

So `11001000` in binary is `200` in decimal.

### Decimal to binary, the easy way

To turn a decimal number into binary, find the biggest power of 2 that fits, mark a 1, subtract, and keep going. Example with 200:

```
200 - 128 = 72       (128 fits — write 1)
 72 -  64 =  8       (64 fits — write 1)
                     (32 doesn't fit — write 0)
                     (16 doesn't fit — write 0)
  8 -   8 =  0       (8 fits — write 1)
                     (4 doesn't fit — write 0)
                     (2 doesn't fit — write 0)
                     (1 doesn't fit — write 0)
```

Read top to bottom: `1 1 0 0 1 0 0 0`. That's 200 in binary.

### Decimal to binary, the divide-by-2 way

If powers of 2 in your head feels hard, here is the back-up trick. Divide by 2 over and over. Write down the remainders. Read them backwards. Same example:

```
200 / 2 = 100  remainder 0
100 / 2 =  50  remainder 0
 50 / 2 =  25  remainder 0
 25 / 2 =  12  remainder 1
 12 / 2 =   6  remainder 0
  6 / 2 =   3  remainder 0
  3 / 2 =   1  remainder 1
  1 / 2 =   0  remainder 1
```

Read the remainder column from bottom to top: `1 1 0 0 1 0 0 0`. Same answer. Both ways work.

### Powers of 2 to memorize

Memorize these. You will use them every single day for the rest of your computer life.

```
2^0  =  1
2^1  =  2
2^2  =  4
2^3  =  8
2^4  =  16
2^5  =  32
2^6  =  64
2^7  =  128
2^8  =  256
2^9  =  512
2^10 =  1024     ("about a thousand," called a "kilobyte")
2^16 =  65536    (size of one big chunk of an IPv6 address)
2^24 =  16777216 (about 16 million)
2^32 =  4294967296   (about 4.3 billion — total IPv4 addresses)
2^64 =  18446744073709551616  (about 18 quintillion — IPv6 subnet size)
2^128 = 340282366920938463463374607431768211456  (about 340 undecillion — total IPv6)
```

The really useful ones to have on the tip of your tongue are 2, 4, 8, 16, 32, 64, 128, 256. Those eight numbers cover almost every IPv4 subnet you'll ever look at. The big ones (2^16, 2^32, 2^64, 2^128) come up when you're sizing networks or counting addresses.

### Why "10" in binary is just 2 in decimal

Every base writes "10" the moment it runs out of single digits. In base 10, you run out of digits after 9, so 10 is your first carry. In base 2, you run out after 1, so 10 is your first carry — and that's the number two. In base 16, you run out after F (which is 15), so 10 is your first carry — and that's the number sixteen.

The number "10" is just "I just ran out and carried." It only equals ten in base 10. In other bases it equals something else.

### Bits, bytes, nibbles, and friends

A single binary digit (a 0 or a 1) is called a **bit.** "Bit" is short for "binary digit." A bit is the smallest unit of information in a computer.

Eight bits in a row is called a **byte.** A byte is the smallest unit that most computers actually work with. Memory is measured in bytes. Files are measured in bytes. Almost every CPU instruction reads or writes some number of bytes at a time.

Four bits — half a byte — is called a **nibble.** Yes, really. Because a nibble is half a bite. Computer people had a sense of humor in the 70s. A nibble can hold one hex digit (since one hex digit needs four bits to write).

Two bytes (16 bits) is called a **word** on most older systems. The word "word" is sloppy because different CPUs use different sized words; on a 32-bit CPU "word" might mean 32 bits, and on a 64-bit CPU it might mean 64 bits. To avoid confusion, people often say:

- **byte** — 8 bits
- **word** — 16 bits (or "whatever the CPU thinks a word is" if context is unclear)
- **dword** ("double word") — 32 bits
- **qword** ("quad word") — 64 bits

In networking we usually use specific terms:

- An IPv4 address is **four bytes** = **32 bits**.
- Each part of `192.168.1.100` is **one byte** = **8 bits**, called an **octet** in networking. ("Octet" just means "eight things." Networking people prefer it because "byte" can be ambiguous.)
- An IPv6 address is **sixteen bytes** = **128 bits**, made of eight **hextets** (16-bit pieces, written in hex).

If somebody says "octet" they mean "8 bits" and they are talking networking. If somebody says "byte" they probably mean the same thing but they might be talking about something else (a file, memory). Same idea, different word in a different room.

### A whole byte, drawn

Here is one byte, with its place values written underneath each bit:

```
   bit position:  7   6   5   4   3   2   1   0
   place value:  128  64  32  16   8   4   2   1
   example:       1   1   0   0   1   0   0   0   = 200

   what's lit:   yes yes  no  no yes  no  no  no
   add the lit:  128+64+0+0+8+0+0+0                = 200
```

The leftmost bit, with place value 128, is called the **most significant bit** (MSB) — it's the most "important" because flipping it changes the number the most. The rightmost bit, with place value 1, is the **least significant bit** (LSB).

When you see a byte written down, by convention the MSB is on the left and the LSB is on the right. Just like decimal: in the number 200, the "2" on the left is more significant (it's the hundreds place) than the "0" on the right (the ones place).

### The maximum value of a byte

A byte has 8 bits. If every single bit is 1, the value is:

```
128 + 64 + 32 + 16 + 8 + 4 + 2 + 1 = 255
```

So a byte goes from `0` (all bits off) to `255` (all bits on). That is exactly **256 different values** (zero counts as one of them).

The number 255 is going to show up a million times in this sheet. It is the maximum value of one byte. It is the highest number you can write in 8 bits. Every time you see `255` in networking, your brain should auto-complete: "ah, all eight bits are on."

### Half a byte: 0 to 15

A nibble has 4 bits. If all 4 bits are on:

```
8 + 4 + 2 + 1 = 15
```

So a nibble goes from `0` to `15` (16 values). This is exactly the range of one hex digit. That's not a coincidence. We'll see why in a second.

## Hex (Base 16)

### Why hex even exists

Reading binary is exhausting. Look:

```
01001000 01100101 01101100 01101100 01101111
```

That's the word "Hello" in ASCII. Forty bits. Your eyeballs already gave up. Hex was invented so we could write binary numbers shorter. Same data in hex:

```
48 65 6C 6C 6F
```

Ten hex digits instead of forty binary digits. Much friendlier.

The whole trick is: **one hex digit = exactly four binary bits.** That's it. Hex is just binary with the bits grouped in fours, and each group given a single shorter symbol.

### Counting in hex

Hex has 16 digits. The first ten are the same as decimal: 0, 1, 2, 3, 4, 5, 6, 7, 8, 9. Then it runs out. The next six digits are letters: A, B, C, D, E, F.

```
DECIMAL    BINARY     HEX
0          0000       0
1          0001       1
2          0010       2
3          0011       3
4          0100       4
5          0101       5
6          0110       6
7          0111       7
8          1000       8
9          1001       9
10         1010       A    <-- letters start here
11         1011       B
12         1100       C
13         1101       D
14         1110       E
15         1111       F
16         10000      10   <-- hex carries
17         10001      11
...
255        11111111   FF
256        100000000  100
65535      1111111111111111  FFFF
```

In hex, A means 10. B means 11. C means 12. D means 13. E means 14. F means 15. Then it carries: F + 1 = 10 (in hex), which is 16 in decimal.

Hex letters can be written upper case (`FF`) or lower case (`ff`). Both mean the same thing. Most tools accept either. Most printed output uses lower case because it's easier on the eyes; many books use upper case because it stands out from variable names. You'll see both. They are the same thing.

### The "0x" prefix

When you see `0xFF` written in source code or a tool's output, the `0x` part is just a marker that says "this is hex." It's not part of the number; it's a label. So `0xFF` means "the number written `FF` in base 16," which is 255.

Some tools use other markers:

- `0xFF` — the C/Python/Go/Rust style. Most common.
- `FFh` — old assembly style (suffix `h`).
- `\xFF` — string-escape style (used inside text strings in many languages).
- `#FF0000` — web color style (used in HTML/CSS for colors). The `#` plays the role of `0x`.

They are all hex. Different makeup, same number underneath.

### Decimal to hex

Same trick as decimal-to-binary, just dividing by 16 instead of 2:

```
255 / 16 = 15 remainder 15      -> remainder F
 15 / 16 =  0 remainder 15      -> remainder F
                                   read bottom-to-top: FF
```

Or, since you already know binary: write the number in binary, group into nibbles, convert each nibble to a hex digit.

```
255 in binary: 11111111
group in fours: 1111 1111
each group:       F    F
result: FF
```

### Hex to decimal

Each digit is multiplied by a power of 16. Same formula as decimal-to-decimal but with 16 as the base.

```
0xFF
 = F * 16^1 + F * 16^0
 = 15 * 16  + 15 * 1
 = 240 + 15
 = 255
```

```
0x1A
 = 1 * 16^1 + 10 * 16^0
 = 16 + 10
 = 26
```

```
0x100
 = 1 * 16^2 + 0 * 16^1 + 0 * 16^0
 = 256
```

### Mental shortcuts to memorize

A few hex values you should burn into your brain:

```
0x0   = 0    = 0000
0x1   = 1    = 0001
0x8   = 8    = 1000
0xF   = 15   = 1111
0x10  = 16   = 0001 0000      (one hex carry)
0xFF  = 255  = 1111 1111      (one byte all on)
0x100 = 256  = 0001 0000 0000 (one byte plus one bit)
0xFFFF = 65535 = 1111 1111 1111 1111 (two bytes all on)
0xFFFFFFFF = 4294967295 = thirty-two bits all on (max IPv4)
```

If you remember `0xFF = 255 = one byte on`, you can derive almost everything else from there.

### Binary-to-hex grouping (the trick)

This is the single most useful conversion shortcut in this entire sheet. Memorize it. To turn a binary number into hex:

```
1. Pad the left side with zeros so the total length is a multiple of 4.
2. Cut into groups of 4 from the right.
3. Look up each group of 4 in your mental table.
4. Stick the hex digits together.
```

Example: convert `11001000` to hex.

```
binary:       11001000
length:       8        (already a multiple of 4, no padding needed)
groups of 4:  1100 1000
look up:      1100 = C
              1000 = 8
result:       C8
in hex:       0xC8
in decimal:   12*16 + 8 = 200    ✓ (matches our earlier 200)
```

Another example: `10110011 11110000`.

```
groups of 4:  1011 0011 1111 0000
look up:      1011 = B
              0011 = 3
              1111 = F
              0000 = 0
result:       B3F0
in hex:       0xB3F0
```

A picture of the grouping operation:

```
binary:   1  0  1  1   0  0  1  1   1  1  1  1   0  0  0  0
group:   |________ |  |________|  |________|  |________|
nibble:    11 (0xB)   3 (0x3)      15 (0xF)    0 (0x0)
hex:        B            3            F             0
final:    0xB3F0
```

To go the other way (hex to binary): expand each hex digit into its 4-bit binary form and string them together. `0xC8` → `1100` + `1000` → `11001000`. Done.

### Why hex shows up everywhere in networking

- An **IPv4 address** is 32 bits, which is exactly 8 hex digits. So `0xC0A80101` is `192.168.1.1` written as hex. (The address `192` = `0xC0`, `168` = `0xA8`, `1` = `0x01`, `1` = `0x01`.)
- An **IPv6 address** is 128 bits, which is exactly 32 hex digits. IPv6 just writes the hex out directly with colons sprinkled in.
- A **MAC address** (the hardware ID of a network card) is 48 bits, written as 12 hex digits, like `00:1A:2B:3C:4D:5E`.
- **Memory addresses** when programs crash are written in hex.
- **Colors** on the web are written in hex (`#FF0000` is bright red).
- Most things that involve "exact bit patterns" are written in hex because hex maps cleanly onto bits.

If you can read hex, you can read most low-level computer stuff. If you can read hex *and* picture the bits underneath, you can read almost anything.

## Bitwise Operations

Now we get to the operations you can do on individual bits. These are the building blocks of how a computer manipulates raw data.

### AND (& symbol)

AND takes two bits and returns 1 only if **both** are 1. Otherwise it returns 0. The truth table:

```
A   B   A AND B
0   0   0
0   1   0
1   0   0
1   1   1
```

You can do AND across whole numbers, bit by bit:

```
  1 0 1 0   (decimal 10)
& 1 1 0 0   (decimal 12)
---------
  1 0 0 0   (decimal 8)
```

Position by position: 1&1=1, 0&1=0, 1&0=0, 0&0=0.

**Why AND matters:** AND is how you **mask** bits. If you want to keep some bits and zero out the rest, AND with a number that has 1s where you want to keep and 0s where you want to zero. This is the operation behind subnet masks (more on that later — a lot more).

```
keep the upper nibble of 0xAB:   0xAB & 0xF0 = 0xA0
keep the lower nibble of 0xAB:   0xAB & 0x0F = 0x0B
```

### OR (| symbol)

OR takes two bits and returns 1 if **either** (or both) is 1. Returns 0 only if both are 0. Truth table:

```
A   B   A OR B
0   0   0
0   1   1
1   0   1
1   1   1
```

Across whole numbers:

```
  1 0 1 0   (decimal 10)
| 1 1 0 0   (decimal 12)
---------
  1 1 1 0   (decimal 14)
```

**Why OR matters:** OR is how you **set** a bit. If you want to turn certain bits ON without disturbing the others, OR with a number that has 1s where you want to set.

```
turn on the high bit of 0x0F:   0x0F | 0x80 = 0x8F
combine two flag sets:          0x01 | 0x04 | 0x10 = 0x15
```

### XOR (^ symbol)

XOR (exclusive OR) returns 1 if **exactly one** of the two bits is 1. Returns 0 if both are 0 or both are 1. Truth table:

```
A   B   A XOR B
0   0   0
0   1   1
1   0   1
1   1   0
```

Across whole numbers:

```
  1 0 1 0   (decimal 10)
^ 1 1 0 0   (decimal 12)
---------
  0 1 1 0   (decimal 6)
```

**Why XOR matters:** XOR is how you **toggle** a bit (flip it). XORing with a 1 flips the bit; XORing with a 0 leaves it alone.

```
toggle the low bit of 0x0E:    0x0E ^ 0x01 = 0x0F
```

XOR has a magical property: `A XOR A = 0`, and `A XOR 0 = A`. So if you XOR something with itself, you get zero. And if you XOR something with a "key" twice, you get the original back. This is how some really simple encryption works (more sophisticated encryption uses much more, but the spirit is XOR).

XOR is also how computers compute **parity** — a quick "is the number of 1 bits even or odd" check used in some error detection schemes. You XOR all the bits together; the result is 1 if there were an odd number of 1s, 0 if even.

### NOT (~ symbol)

NOT takes one bit and flips it. 0 becomes 1; 1 becomes 0. Truth table:

```
A   NOT A
0   1
1   0
```

Across a whole number, every bit flips:

```
~ 1 0 1 0   = 0 1 0 1   (in 4 bits: ~10 = 5)

In 8 bits:
~ 0 0 0 0 1 0 1 0  = 1 1 1 1 0 1 0 1   (~10 = 245)
```

That's a key gotcha: NOT depends on how many bits you say the number is. `~0xFF` in 8 bits is `0x00`. `~0xFF` in 32 bits is `0xFFFFFF00` (every bit not in the original FF is flipped on, leaving only the lowest 8 bits as 0).

**Why NOT matters:** NOT inverts. If you have a subnet mask `255.255.255.0` (binary `11111111.11111111.11111111.00000000`), then NOT of that is `0.0.0.255` (binary `00000000.00000000.00000000.11111111`). That's called the **wildcard mask**, and it's used by some routers (particularly Cisco) to describe subnets in access-control lists.

### Shift left (<< symbol)

Shift left moves all the bits some number of positions to the left, filling in zeros on the right.

```
0001 << 1  = 0010
0001 << 2  = 0100
0001 << 3  = 1000
```

Each shift left by 1 multiplies the value by 2. So `1 << 8` = 256. `1 << 16` = 65536. `1 << 32` = 4294967296.

This is how you build up powers of 2 fast in code.

### Shift right (>> symbol)

Shift right moves the bits to the right, filling in zeros on the left (for unsigned numbers).

```
1000 >> 1  = 0100
1000 >> 2  = 0010
1000 >> 3  = 0001
```

Each shift right by 1 divides by 2 (truncating any remainder). So `255 >> 1` = 127 (the lowest bit is lost). `0x100 >> 4` = `0x10`.

For **signed** numbers (numbers that can be negative), there are actually two kinds of right shift:

- **Logical shift right** — fills with 0 on the left.
- **Arithmetic shift right** — copies the sign bit into the new positions on the left, so the number stays the same sign.

In most languages, the operator `>>` does the logical version on unsigned types and the arithmetic version on signed types. You usually do not have to think about this for networking; we work almost exclusively with unsigned values.

### Combining the operations: set, clear, toggle, test

Putting them all together, here are the four "what you do to one bit" patterns:

```
SET   bit n of x:   x = x | (1 << n)
CLEAR bit n of x:   x = x & ~(1 << n)
TOGGLE bit n of x:  x = x ^ (1 << n)
TEST  bit n of x:   if (x & (1 << n)) { ... it was set ... }
```

Read those four lines a few times. Every low-level networking program has these patterns sprinkled all over it. They are how a router decides which interface a packet came in on. They are how a kernel decides whether a TCP flag is set. They are how an eBPF program reads a header byte at line rate.

### Truth tables side-by-side

Here are AND, OR, XOR, and NOT in one place for quick reference:

```
A   B   AND  OR  XOR        A   NOT
0   0   0    0   0          0   1
0   1   0    1   1          1   0
1   0   0    1   1
1   1   1    1   0
```

If you can fill in those eight rows on a napkin, you can do all the bitwise math in this sheet.

## IPv4 Addresses Are Just 32-Bit Numbers

### The dotted-quad notation

You have probably seen IP addresses like `192.168.1.100` your whole life. That funny dotted notation is just a way to write a 32-bit number that's easy on human eyes.

A 32-bit number in pure decimal is awful. `3232235876` — what even is that? You can't tell anything from it. So networking people invented the **dotted-quad** style: take the 32 bits, chop them into four groups of 8 bits each, write each 8-bit group in decimal (0–255), and put dots between them.

```
192.168.1.100
 |   |   | |
 |   |   | +-- last octet  = 100
 |   |   +---- third octet = 1
 |   +-------- second octet= 168
 +------------ first octet = 192
```

Each "octet" is one byte. Four bytes total. Thirty-two bits total.

### From dotted-quad to binary

Here is `192.168.1.100` written all the way out in binary:

```
192      .   168      .   1        .   100
11000000 .   10101000 .   00000001 .   01100100
```

Each block of 8 binary digits is one byte (one octet). Concatenated together (no dots) you get the 32-bit number `11000000101010000000000101100100`.

Diagram:

```
DOTTED-QUAD:        192     .      168     .       1     .     100
BINARY  (per octet): 1100 0000     1010 1000     0000 0001     0110 0100
HEX     (per octet):    C 0           A 8           0 1           6 4

THE 32-BIT NUMBER:
 1100 0000 1010 1000 0000 0001 0110 0100
   C    0    A    8    0    1    6    4

WRITTEN AS HEX:    0xC0A80101
WRITTEN AS DECIMAL: 3232235876
```

`192.168.1.100` and `0xC0A80101` and `3232235876` are all the same 32-bit number. Three different ways to write it down. The dotted-quad version is the only one humans can read at a glance.

### Total IPv4 address count

There are 32 bits. Each bit can be 0 or 1. So the total number of different addresses is `2^32` = `4,294,967,296`. About 4.3 billion. At first glance that sounds like a lot. There are about 8 billion humans, so it's not even one address per person, and that's before you count phones, laptops, IoT toasters, factory equipment, and printers.

That's why IPv4 ran out. It actually ran out for new allocations from the central pool around 2011-2015. The world has been getting by since then with NAT (which lets many devices share one public IP) and with people slowly moving to IPv6 (which has so many addresses that "running out" stops being a real concern; we'll see why later).

### From any octet to binary, fast

Each octet is just a byte. So pretend you forgot what the rest of the address looks like and just convert that one number from decimal to binary. We did this for 200 already; here are a few common ones:

```
0      = 00000000
1      = 00000001
2      = 00000010
4      = 00000100
8      = 00001000
16     = 00010000
32     = 00100000
64     = 01000000
128    = 10000000
192    = 11000000
224    = 11100000
240    = 11110000
248    = 11111000
252    = 11111100
254    = 11111110
255    = 11111111
```

Notice the pattern in the bottom half. `128`, `192`, `224`, `240`, ... — those are the values you get when you light up the high bits one at a time. Each one is the previous plus the next power of 2 below it. They are exactly the values you'll see in **subnet masks**, because subnet masks are always a run of 1s on the left followed by a run of 0s on the right, and these are the byte-sized "I lit up some 1s on the left" values.

We will see them again in literally one section.

## CIDR (Classless Inter-Domain Routing)

### The slash notation

Networking people write a network address followed by a slash and a number, like:

```
192.168.1.0/24
10.0.0.0/8
2001:db8::/32
```

The number after the slash is called the **prefix length** or **CIDR prefix.** It is the number of bits at the front of the address that are the **network part.** Whatever bits are left are the **host part.**

```
192.168.1.0/24
              ^^
              this is the slash: 24 means "the first 24 bits are network"

binary form (32 bits total):
11000000.10101000.00000001.00000000
\___________________________/\______/
  first 24 bits = NETWORK     last 8 bits = HOSTS
```

The first 24 bits identify which network this is. The last 8 bits (32-24=8) identify which specific host on that network. 8 host bits = 2^8 = 256 possible host slots in this subnet (more on the "two are reserved" detail in a moment).

### The whole CIDR table for IPv4

Memorize the friends. The middle column is the dotted-decimal subnet mask. The right column is "how many addresses are in one subnet of this size."

```
PREFIX  MASK              ADDRESSES IN ONE SUBNET
/0      0.0.0.0           4,294,967,296 (the whole IPv4 internet)
/1      128.0.0.0         2,147,483,648
/2      192.0.0.0         1,073,741,824
/3      224.0.0.0           536,870,912
/4      240.0.0.0           268,435,456
/5      248.0.0.0           134,217,728
/6      252.0.0.0            67,108,864
/7      254.0.0.0            33,554,432
/8      255.0.0.0            16,777,216  ("a slash 8" — really big)
/9      255.128.0.0           8,388,608
/10     255.192.0.0           4,194,304
/11     255.224.0.0           2,097,152
/12     255.240.0.0           1,048,576
/13     255.248.0.0             524,288
/14     255.252.0.0             262,144
/15     255.254.0.0             131,072
/16     255.255.0.0              65,536  ("a slash 16" — also big)
/17     255.255.128.0            32,768
/18     255.255.192.0            16,384
/19     255.255.224.0             8,192
/20     255.255.240.0             4,096
/21     255.255.248.0             2,048
/22     255.255.252.0             1,024
/23     255.255.254.0               512
/24     255.255.255.0               256  ("a slash 24" — typical home)
/25     255.255.255.128             128
/26     255.255.255.192              64
/27     255.255.255.224              32
/28     255.255.255.240              16
/29     255.255.255.248               8
/30     255.255.255.252               4  (point-to-point classic)
/31     255.255.255.254               2  (point-to-point modern, RFC 3021)
/32     255.255.255.255               1  (single host)
```

### Reading the table

Each step from `/24` to `/25` to `/26` cuts the size in half. /24 has 256, /25 has 128, /26 has 64, /27 has 32, /28 has 16, /29 has 8, /30 has 4, /31 has 2, /32 has 1. Same in the bigger sizes: each bigger prefix length doubles the size.

The mask is just "1s for network, 0s for host." A `/24` means 24 ones, then 8 zeros: `11111111.11111111.11111111.00000000` = `255.255.255.0`.

The math is easy once you see it:

```
/n means n ones at the front, (32-n) zeros at the back.
Number of addresses in a /n: 2^(32-n).
Number of usable hosts: 2^(32-n) - 2 (subtract the network address and the broadcast address — see below).
Exceptions:
  /31 is treated as 2 usable (point-to-point links, RFC 3021).
  /32 is treated as 1 usable (single host route).
```

### Some sample masks drawn in binary

Here are a few common prefixes drawn out in 32 bits, with the boundary marked:

```
/8:   11111111  00000000  00000000  00000000      = 255.0.0.0
        net    | host   |  host   |  host

/16:  11111111  11111111  00000000  00000000      = 255.255.0.0
        net    |  net    | host   |  host

/24:  11111111  11111111  11111111  00000000      = 255.255.255.0
        net    |  net    |  net    | host

/26:  11111111  11111111  11111111  11000000      = 255.255.255.192
        net    |  net    |  net    |nn|host

/28:  11111111  11111111  11111111  11110000      = 255.255.255.240
        net    |  net    |  net    |nnnn|host

/30:  11111111  11111111  11111111  11111100      = 255.255.255.252
        net    |  net    |  net    |nnnnnn|h  (only 2 host bits)

/31:  11111111  11111111  11111111  11111110      = 255.255.255.254
        net    |  net    |  net    |nnnnnnn|h (only 1 host bit; RFC 3021)

/32:  11111111  11111111  11111111  11111111      = 255.255.255.255
        all bits are network — this IS one specific address, no hosts.
```

Read those lines until they feel obvious. The mask is always 1s on the left and 0s on the right. The prefix length is just "how many 1s are there." That's it. That's the whole concept.

## Subnet Masks

### What the mask does (the AND operation)

A subnet mask is a 32-bit number you AND with an IP address to extract the network portion. The 1s in the mask "keep" their corresponding bits in the IP. The 0s "zero out" their corresponding bits.

The classic example, in binary:

```
IP Address:    192.168.1.100   = 11000000 . 10101000 . 00000001 . 01100100
Subnet Mask:   255.255.255.0   = 11111111 . 11111111 . 11111111 . 00000000
                                 -------- AND --------
Network ID:    192.168.1.0     = 11000000 . 10101000 . 00000001 . 00000000
```

That's the whole job of a subnet mask. AND it with the address and you get the network number. Easy.

In decimal, for `255.255.255.0` style masks, the AND has a really nice shortcut: any byte where the mask is 255 is unchanged, and any byte where the mask is 0 becomes 0. So `192.168.1.100 & 255.255.255.0 = 192.168.1.0`. Just zero out the bytes where the mask is 0.

For non-byte-aligned masks (like `255.255.255.192`, which is `/26`), you can't take that shortcut for the partial byte; you have to actually do the AND on that one byte.

### A drawing of the mask-AND-address operation

Here is a /26 example, where the mask cuts through the middle of the last byte:

```
ADDRESS:         192      .   168      .   1        .   100
                 11000000 .   10101000 .   00000001 .   01100100

MASK (/26):      255      .   255      .   255      .   192
                 11111111 .   11111111 .   11111111 .   11000000
                 ------------- bit-by-bit AND -------------------
NETWORK:         192      .   168      .   1        .   64
                 11000000 .   10101000 .   00000001 .   01000000

(In the last byte, only the top two bits passed through the mask.
 The bottom six bits of 100 were zeroed out, leaving 64.)
```

That's why `192.168.1.100/26` is part of the `192.168.1.64/26` subnet. The mask zeroes out the lower bits and reveals the network.

### The broadcast address

Every IPv4 subnet has a special address called the **broadcast address.** It's the address with the host bits **all set to 1**. To get it:

1. Take the network address (host bits all 0).
2. Replace the host bits with all 1s.

Or equivalently: take the address, OR it with the **wildcard mask** (NOT of the subnet mask).

```
NETWORK:         192      .   168      .   1        .   64
                 11000000 .   10101000 .   00000001 .   01000000

WILDCARD:        0        .   0        .   0        .   63
                 00000000 .   00000000 .   00000000 .   00111111
                 ------------- bit-by-bit OR --------------------
BROADCAST:       192      .   168      .   1        .   127
                 11000000 .   10101000 .   00000001 .   01111111
```

The broadcast address is the address that "everybody on the subnet" listens to. If you send a packet to the broadcast address, every host on that subnet sees it. We do not use this very often anymore (most modern protocols prefer multicast or unicast), but it is reserved and not assignable to a host.

### Reserved addresses in a subnet

Every IPv4 subnet of size `/30` or bigger reserves two addresses that you can't give to a real machine:

- **Network address** — the lowest address in the range, host bits all 0. Example: `192.168.1.0` in `192.168.1.0/24`.
- **Broadcast address** — the highest address, host bits all 1. Example: `192.168.1.255` in `192.168.1.0/24`.

So in a `/24` you have 256 total addresses but only 254 **usable host addresses.** In a `/26` you have 64 total but only 62 usable. In a `/30` you have 4 total but only 2 usable (network address `.0`, broadcast `.3`, hosts `.1` and `.2`).

`/31` (RFC 3021) is special: it skips the network/broadcast convention and lets both addresses be hosts. Used for point-to-point links between routers, where you only need two endpoints and you don't want to waste two addresses on broadcasts.

`/32` is for a single host. There are no host bits at all. The "subnet" is exactly one address. You will see this used for loopback addresses on routers, and for advertising single-host routes in BGP.

## Subnetting (the Math)

### Splitting a /24 into /26 subnets

Now we put it together. Take `192.168.1.0/24` (256 addresses, mask `255.255.255.0`). Split it into 4 equal-sized subnets.

We need 4 subnets. 4 = 2^2, so we need to "borrow 2 bits" from the host part to make a subnet number. New prefix: `/24 + 2` = `/26`.

```
ORIGINAL: 192.168.1.0/24
   binary mask: 11111111.11111111.11111111.00000000

NEW:      192.168.1.0/26 (and 3 siblings)
   binary mask: 11111111.11111111.11111111.11000000
                                            ^^
                                            two bits stolen from host

The two stolen bits give us 4 possible values: 00, 01, 10, 11.
That's our 4 subnets.
```

The 4 subnets are exactly:

```
SUBNET 1: 192.168.1.0/26
   stolen bits: 00
   binary range: 11000000.10101000.00000001.00 000000
                                            \________ 6 host bits, 2^6 = 64 addresses
   range: 192.168.1.0   – 192.168.1.63
   network:   192.168.1.0
   first host:192.168.1.1
   last host: 192.168.1.62
   broadcast: 192.168.1.63
   usable: 62

SUBNET 2: 192.168.1.64/26
   stolen bits: 01
   range: 192.168.1.64  – 192.168.1.127
   network:   192.168.1.64
   first host:192.168.1.65
   last host: 192.168.1.126
   broadcast: 192.168.1.127
   usable: 62

SUBNET 3: 192.168.1.128/26
   stolen bits: 10
   range: 192.168.1.128 – 192.168.1.191
   network:   192.168.1.128
   first host:192.168.1.129
   last host: 192.168.1.190
   broadcast: 192.168.1.191
   usable: 62

SUBNET 4: 192.168.1.192/26
   stolen bits: 11
   range: 192.168.1.192 – 192.168.1.255
   network:   192.168.1.192
   first host:192.168.1.193
   last host: 192.168.1.254
   broadcast: 192.168.1.255
   usable: 62
```

### Diagram of the /24-to-four-/26 split

```
                  192.168.1.0/24    (256 addresses)
                 /          |          |          \
        /26 #1     /26 #2       /26 #3       /26 #4
   .0 - .63      .64 - .127    .128 - .191   .192 - .255
   64 addrs      64 addrs      64 addrs      64 addrs
   62 usable     62 usable     62 usable     62 usable

   stolen bits   stolen bits   stolen bits   stolen bits
       00            01            10            11
```

### The "magic number" shortcut

There's a fast way to figure out the subnet boundary in your head. Take the value of the interesting (last non-255) octet of the mask, subtract from 256.

```
Mask /26 = 255.255.255.192
Magic: 256 - 192 = 64

Subnet boundaries are multiples of 64:
  0, 64, 128, 192, 256(=next /24)

Which subnet does 192.168.1.100 fall in?
  100 / 64 = 1 remainder 36  →  starts at 1*64 = 64
  → 192.168.1.64/26
```

Same with /28 (mask `.240`): magic number 256 - 240 = 16. Subnets start at 0, 16, 32, 48, 64, 80, 96, 112, 128, 144, 160, 176, 192, 208, 224, 240. Sixteen subnets of sixteen addresses each.

Same with /27 (mask `.224`): magic number 256 - 224 = 32. Subnets start at 0, 32, 64, 96, 128, 160, 192, 224. Eight subnets of thirty-two addresses each.

### Splitting smaller: /27, /28, /30

Same trick, smaller pieces. Let's split `192.168.1.0/24` into eight `/27` subnets.

```
8 subnets means borrowing 3 bits (2^3 = 8). /24 + 3 = /27.
Each /27: 2^(32-27) = 2^5 = 32 addresses.
Magic number: 256 - 224 = 32.

  192.168.1.0/27    (.0   - .31)
  192.168.1.32/27   (.32  - .63)
  192.168.1.64/27   (.64  - .95)
  192.168.1.96/27   (.96  - .127)
  192.168.1.128/27  (.128 - .159)
  192.168.1.160/27  (.160 - .191)
  192.168.1.192/27  (.192 - .223)
  192.168.1.224/27  (.224 - .255)
```

Or sixteen `/28` subnets of 16 addresses each. Or sixty-four `/30` subnets of 4 addresses each (handy for point-to-point links).

### A really small one: /30 for a router-to-router link

Two routers have a link. Each end needs an IP. So two host addresses. The smallest classic subnet that has two usable addresses is `/30`.

```
192.168.1.0/30:
   network:   192.168.1.0
   router A:  192.168.1.1
   router B:  192.168.1.2
   broadcast: 192.168.1.3

   total addresses: 4
   usable hosts: 2
   waste: 2 (network + broadcast)
```

Half the addresses are "wasted" on network and broadcast for a /30. This was acceptable when address space was infinite. Now that IPv4 is precious, RFC 3021 says: use `/31` instead.

```
192.168.1.0/31:
   no separate network or broadcast — both addresses are usable.
   router A:  192.168.1.0
   router B:  192.168.1.1

   total addresses: 2
   usable hosts: 2
   waste: 0
```

Most modern equipment supports `/31`. Some old or weird equipment doesn't, so `/30` is still common in the wild. Understand both, use `/31` when you can.

### A really big one: /16

`/16` gives you `2^16 = 65,536` addresses. Mask `255.255.0.0`. Network bits are the first two octets. Host bits are the last two octets. Example: `10.5.0.0/16` covers `10.5.0.0` through `10.5.255.255`. 65,534 usable hosts (subtracting network and broadcast).

People often subnet a /16 internally into many /24s. Each /24 is 256 addresses (254 usable). 256 /24s fit in one /16. Naming: `10.5.0.0/24`, `10.5.1.0/24`, `10.5.2.0/24`, ..., `10.5.255.0/24`.

## Variable-Length Subnet Masking (VLSM)

### Why one size doesn't fit all

If you have a `/24` and you split it strictly into four `/26`s, every subnet has 62 usable hosts whether you need them or not. Real networks aren't shaped like that. You might have:

- An engineering subnet with 100 hosts.
- A sales subnet with 50 hosts.
- A management subnet with 20 hosts.
- A server VLAN with 10 hosts.
- Two point-to-point router links with 2 hosts each.

If you used /26 subnets for all of them, the engineering subnet wouldn't even fit (only 62 usable in a /26), and the point-to-point links would waste 60 addresses each. VLSM (Variable-Length Subnet Masking) lets you slice the address space at different sizes for different needs.

### The procedure

1. Sort your requirements largest to smallest by host count.
2. For each requirement, pick the smallest subnet size that fits.
3. Assign blocks in order, starting from the bottom of your address pool.
4. Make sure each block is **power-of-2 aligned** (the network number must be a multiple of the block size — we'll see why in a moment).

### Example: design for a small office

Given: `10.1.0.0/24` (256 addresses).

```
Need:
  Engineering   100 hosts -> need /25 (126 usable, fits 100)
  Sales          50 hosts -> need /26 (62 usable, fits 50)
  Management     20 hosts -> need /27 (30 usable, fits 20)
  Servers        10 hosts -> need /28 (14 usable, fits 10)
  P2P link 1      2 hosts -> need /30 (2 usable, fits 2; or /31 if equipment supports)
  P2P link 2      2 hosts -> need /30
```

Allocate largest first:

```
10.1.0.0/25     Engineering   (.0 network, .1 - .126 hosts, .127 broadcast)
10.1.0.128/26   Sales         (.128 network, .129 - .190 hosts, .191 broadcast)
10.1.0.192/27   Management    (.192 network, .193 - .222 hosts, .223 broadcast)
10.1.0.224/28   Servers       (.224 network, .225 - .238 hosts, .239 broadcast)
10.1.0.240/30   P2P link 1    (.240 network, .241 - .242 hosts, .243 broadcast)
10.1.0.244/30   P2P link 2    (.244 network, .245 - .246 hosts, .247 broadcast)
10.1.0.248/29   Spare         (8 addresses left over for future use)
```

Total used: 100 + 50 + 20 + 10 + 2 + 2 = 184 hosts. Total available in `/24`: 254. We have 70 spare for growth, neatly packed in one /29 and a few smaller leftovers.

### Why "largest first"

If you allocate the smallest first, you can leave the address space fragmented in a way that big chunks no longer fit. Imagine you had only 256 addresses and started by handing out a /30 at `10.1.0.0/30`, then a /28 at `10.1.0.16/28`, then a /27 at `10.1.0.32/27`, then... by the time you need the /25 for engineering, the only big enough chunk would be `10.1.0.128/25`, which is fine in this small example, but in tighter packings the small allocations can corner you.

Allocating largest to smallest is **provably optimal** for power-of-2-sized blocks with power-of-2 alignment. It's a special case of bin packing where the greedy strategy happens to be the right answer. Just always go big to small.

### Power-of-2 alignment

A `/n` subnet must start at an address that is a multiple of `2^(32-n)`. So:

- `/24` blocks (256 addresses each) start at multiples of 256: `.0.0`, `.1.0`, `.2.0`, ...
- `/26` blocks (64 addresses each) start at multiples of 64: `.0`, `.64`, `.128`, `.192`.
- `/27` blocks (32 addresses each) start at multiples of 32: `.0`, `.32`, `.64`, `.96`, `.128`, `.160`, `.192`, `.224`.
- `/30` blocks (4 addresses each) start at multiples of 4: `.0`, `.4`, `.8`, ..., `.252`.

You **cannot** put a `/26` at `192.168.1.10` for example. It must start at `.0`, `.64`, `.128`, or `.192`. Hardware lookup engines depend on this alignment to do fast prefix matching, and CIDR's whole arithmetic falls apart if you violate it. The rule isn't bureaucratic; it's mathematical.

### Route summarization (the reverse of subnetting)

Sometimes you have a bunch of contiguous, aligned subnets and you want to advertise them as one bigger thing. That's called **summarization** or **supernetting**. Suppose you have:

```
10.1.0.0/24
10.1.1.0/24
10.1.2.0/24
10.1.3.0/24
```

Look at them in binary on the third octet:

```
10.1.00000000.0/24
10.1.00000001.0/24
10.1.00000010.0/24
10.1.00000011.0/24
       ^^^^^^
       common prefix is the first 6 bits of the third octet: 000000
       total network bits in common: 16 + 6 = 22

Summary: 10.1.0.0/22  (covers 10.1.0.0 through 10.1.3.255)
```

For summarization to be valid:

- All subnets must be **contiguous** (no gaps).
- The summary must be **aligned** at a block boundary (here, `10.1.0.0/22` starts at 0 mod 4 in the third octet — yes, that's aligned).
- The summary must not accidentally include networks that don't belong to you.

`10.1.4.0/24` and friends could be summarized as `10.1.4.0/22` (next aligned /22). But `10.1.1.0/22` would be invalid because `1 mod 4 ≠ 0` — that subnet doesn't start on a /22 boundary.

## Reserved Address Blocks

Not every IPv4 address is fair game. Various blocks are set aside for specific uses by RFC, by tradition, or by IANA decree. Knowing the reserved blocks saves you debugging time, because if you see a packet from `169.254.x.x`, you immediately know "DHCP probably failed."

### The big private blocks (RFC 1918)

These are reserved for use inside private networks. They are **not routable on the public internet.** Anybody can use them, anywhere, without asking. Your home router uses them. Your office uses them. Cloud providers use them inside VPCs. They get reused everywhere.

```
10.0.0.0/8        10.0.0.0   - 10.255.255.255   16,777,216 addresses
172.16.0.0/12     172.16.0.0 - 172.31.255.255    1,048,576 addresses
192.168.0.0/16    192.168.0.0- 192.168.255.255      65,536 addresses
```

If you are designing a private network, pick from one of these. Your laptop's `192.168.1.x` address came from one of these. So did your office VPN range. Almost everybody starts here.

### Loopback (RFC 6890)

```
127.0.0.0/8       127.0.0.0  - 127.255.255.255
```

The whole `127.x.x.x` range is reserved for "this very machine." Most often you see `127.0.0.1` (called `localhost`) used as the standard loopback. Traffic to `127.0.0.1` never leaves your computer. It's how a program on your laptop can talk to another program on your laptop using normal network sockets without involving any actual network gear.

There's a whole `/8` (16 million addresses) reserved for this, even though 99% of the time only `127.0.0.1` is used. The rest is just precaution.

### Link-local / APIPA (RFC 3927)

```
169.254.0.0/16    169.254.0.0 - 169.254.255.255
```

When a machine plugs into a network and asks for an IP via DHCP and gets no answer, many operating systems pick a random address in this range and use that. It's called "link-local" because it only works for talking to other machines on the same physical link (it's not routable). Microsoft calls this "APIPA" (Automatic Private IP Addressing).

If you ever see a `169.254.x.x` address on a machine, that's a giant red flag: the DHCP server didn't answer. Either there's no DHCP server on that network, or the server is down, or your machine's network cable is in the wrong port.

### Multicast and reserved

```
224.0.0.0/4       224.0.0.0  - 239.255.255.255   multicast
240.0.0.0/4       240.0.0.0  - 255.255.255.254   reserved (former Class E)
255.255.255.255   limited broadcast
```

The whole `224.0.0.0/4` range is reserved for multicast (one-sender, many-receivers) traffic. Routing protocols like OSPF use addresses in this range to talk between routers (`224.0.0.5` is "all OSPF routers"; `224.0.0.6` is "all OSPF designated routers"). Streaming video used to use multicast more; today most internet streaming is unicast over CDNs.

The `240.0.0.0/4` range was historically called **Class E** and reserved "for future use." It has never been allocated and probably never will be. Most operating systems even refuse to use addresses in this range.

`255.255.255.255` is the **limited broadcast** address. A packet sent to this address is meant to reach every host on the local segment but to never be forwarded by routers. This is what DHCP discovery messages use before a host has any IP.

### Carrier-grade NAT shared range (RFC 6598)

```
100.64.0.0/10     100.64.0.0 - 100.127.255.255
```

When ISPs ran out of public IPv4 addresses to give to customers, they started doing **carrier-grade NAT (CGN or CGNAT)**: they put many customers behind one public IP, and gave each customer a private-looking address from inside the ISP. The catch: ISPs didn't want to use `10.x.x.x` because customers might already be using that for their home networks (causing collisions). RFC 6598 reserved `100.64.0.0/10` specifically for this.

If you ever see a `100.64.x.x` or `100.127.x.x` address on your home gateway's WAN side, your ISP is doing CGNAT and you don't have a real public IP. This affects things like running a server at home (port forwarding might not work).

### Documentation ranges (RFC 5737)

```
192.0.2.0/24       TEST-NET-1
198.51.100.0/24    TEST-NET-2
203.0.113.0/24     TEST-NET-3
```

These are reserved for use in **documentation and example output.** When you see a tutorial that says "imagine you have IP address X," it should ideally use one of these so nobody gets confused with a real address. Some books still use real-looking IPs, which can be misleading.

### Quick lookup table

```
RANGE                  CIDR                PURPOSE
0.0.0.0                0.0.0.0/32          unspecified / default route
0.0.0.0/8              0.0.0.0/8           "this network" (RFC 1122)
10.0.0.0/8             10.0.0.0/8          private (RFC 1918)
100.64.0.0/10          100.64.0.0/10       CGNAT shared (RFC 6598)
127.0.0.0/8            127.0.0.0/8         loopback
169.254.0.0/16         169.254.0.0/16      link-local / APIPA
172.16.0.0/12          172.16.0.0/12       private (RFC 1918)
192.0.0.0/24           192.0.0.0/24        IETF protocol assignments
192.0.2.0/24           192.0.2.0/24        TEST-NET-1 (docs)
192.88.99.0/24         192.88.99.0/24      6to4 anycast (deprecated)
192.168.0.0/16         192.168.0.0/16      private (RFC 1918)
198.18.0.0/15          198.18.0.0/15       benchmarking
198.51.100.0/24        198.51.100.0/24     TEST-NET-2 (docs)
203.0.113.0/24         203.0.113.0/24      TEST-NET-3 (docs)
224.0.0.0/4            224.0.0.0/4         multicast (Class D)
240.0.0.0/4            240.0.0.0/4         reserved (Class E)
255.255.255.255        255.255.255.255/32  limited broadcast
```

## IPv6 Addresses Are 128-Bit Numbers

### The shift: 32 bits → 128 bits

IPv4 has 4.3 billion addresses. Sounds big until you count phones. So in the late 90s, the IETF designed IPv6 to replace it. IPv6 has **128 bits** instead of 32. That is **`2^128 = 340,282,366,920,938,463,463,374,607,431,768,211,456`** addresses. Three hundred forty undecillion. Three hundred forty followed by 36 zeros.

To put it in perspective: there are about `7 × 10^27` atoms in a human body. IPv6 has roughly **fifty trillion addresses for every atom in your body.** "Running out" stops being a real concern. We will not run out of IPv6 in the lifetime of the universe at any plausible allocation rate.

### The hex-with-colons notation

Writing 128 bits in dotted-quad would be hideous (sixteen octets, fifteen dots). Writing it in pure decimal would be a 39-digit number. So IPv6 uses a different style: write the 128 bits as **eight groups of four hex digits separated by colons.** Each group is 16 bits, called a **hextet** (because "octet" is 8 bits, "hextet" is 16).

A full IPv6 address looks like this:

```
2001:0db8:85a3:0000:0000:8a2e:0370:7334
^^^^ ^^^^ ^^^^ ^^^^ ^^^^ ^^^^ ^^^^ ^^^^
 1    2    3    4    5    6    7    8     -- 8 hextets
16   16   16   16   16   16   16   16     -- bits per hextet
                                              total: 128 bits
```

Each hextet is 4 hex digits = 16 bits = 2 bytes.

### Compression rule 1: drop leading zeros in each hextet

You can drop leading zeros within each hextet:

```
Original: 2001:0db8:85a3:0000:0000:8a2e:0370:7334
Trimmed:  2001:db8:85a3:0:0:8a2e:370:7334
```

Note: only **leading** zeros within a hextet. You cannot drop trailing zeros. `0db8` becomes `db8`, but `7330` stays `7330`. And `0000` becomes `0` — at least one digit must remain.

### Compression rule 2: replace ONE run of zero hextets with `::`

If you have two or more zero hextets in a row, you can replace them with `::` (two colons):

```
After rule 1: 2001:db8:85a3:0:0:8a2e:370:7334
After rule 2: 2001:db8:85a3::8a2e:370:7334
```

The `::` means "fill in as many zero hextets as needed to make the total length 8 hextets." So `2001:db8:85a3::8a2e:370:7334` has 6 written hextets, meaning the `::` represents `8 - 6 = 2` zero hextets.

**Important:** you can only use `::` **once** per address. Otherwise it would be ambiguous (which run of zeros did the `::` refer to?). If there are multiple runs of zeros, by RFC 5952 you compress the longest run; if multiple runs are tied, you compress the leftmost one.

### Compression diagram

```
ORIGINAL:    2001:0db8:0000:0000:0000:0000:0000:0001
                    \_____ run of 5 zero hextets ____/

after rule 1: 2001:db8:0:0:0:0:0:1
after rule 2: 2001:db8::1

reading 2001:db8::1 right back:
   "2001 db8 [fill with zeros to total 8 hextets] 1"
   = 2001:0db8:0000:0000:0000:0000:0000:0001  ✓
```

Other examples:

```
fe80:0000:0000:0000:1234:5678:9abc:def0
   → fe80::1234:5678:9abc:def0

ff02:0000:0000:0000:0000:0000:0000:0001
   → ff02::1

0000:0000:0000:0000:0000:0000:0000:0001
   → ::1                  (the IPv6 loopback)

0000:0000:0000:0000:0000:0000:0000:0000
   → ::                   (the IPv6 unspecified address)
```

### IPv6 doesn't use a subnet mask the same way

IPv6 only uses CIDR-style prefix length notation. There's no dotted-quad mask. You write `2001:db8::/64` and that means "the first 64 bits are network, the last 64 are interface ID." There are no `255.255.255.255` style masks for IPv6. Just the prefix.

### IPv6 prefixes and "interface ID"

A typical global IPv6 unicast address is split as:

```
+-----------------------------+--------+--------------------------+
|   Global Routing Prefix     | Subnet |     Interface ID         |
|       (48 bits typical)     |   ID   |       (64 bits)          |
|                             |16 bits |                          |
+-----------------------------+--------+--------------------------+
   given to you by the ISP       you      identifies the host on
   or RIR                       choose    the subnet
```

- **Global Routing Prefix** (typically `/48`): assigned to your organization by the ISP or by an RIR. The first 48 bits.
- **Subnet ID** (typically 16 bits): your internal subnetting field. Lets you have 65,536 subnets at a site.
- **Interface ID** (typically 64 bits): identifies the host on the subnet.

So the standard subnet size in IPv6 is `/64`. Yes, every standard IPv6 subnet has `2^64 = 18,446,744,073,709,551,616` addresses. Yes, that's roughly two hundred and fifty thousand times more addresses per subnet than the entire IPv4 internet has total. We are not going to run out of subnet space.

## IPv6 Subnet Sizes

### Why /64 is the standard subnet

The 64-bit "interface ID" wasn't an arbitrary choice. It was sized to support **SLAAC** (StateLess Address AutoConfiguration), where a host can derive its own interface ID without DHCP. Originally hosts derived the ID from their MAC address (48 bits) plus 16 padding bits called modified EUI-64. Modern hosts use random values for privacy (RFC 4941) but still need the full 64-bit interface ID space.

If you make your subnet smaller than `/64` (say, `/96`), SLAAC breaks. Don't do it. The few exceptions are:

- `/127` for point-to-point links between routers (RFC 6164).
- `/128` for single-host routes (loopbacks, anycast service addresses).

For everything else, `/64` is the smallest subnet you should design.

### The hierarchy of common IPv6 prefix sizes

```
/12       Regional Internet Registry (RIR) allocation     2^116 hosts
/32       ISP allocation                                   2^96  hosts
/48       Site / enterprise allocation (typical)           2^80  hosts
/56       Residential ISP (typical)                        2^72  hosts
/60       Smaller residential allocation                   2^68  hosts
/64       One subnet (the standard)                        2^64  hosts
/96       NAT64 prefix                                     2^32  hosts
/127      Router-to-router link (RFC 6164)                       2 hosts
/128      Single host                                            1 host
```

A `/48` site has `2^16 = 65,536` `/64` subnets to play with. That's enough subnets for a really big enterprise. A `/56` residential allocation has `2^8 = 256` /64 subnets, which is enough for a very large home network and then some.

### Subnetting an IPv6 /48

Given: `2001:db8:abcd::/48` (a typical site allocation).

The "subnet ID" is the 16 bits between bit 48 and bit 64. That's hextet 4 of the address. You have `2^16 = 65,536` possible subnet IDs.

```
2001:db8:abcd:0000::/64    Management
2001:db8:abcd:0001::/64    Engineering
2001:db8:abcd:0002::/64    Sales
2001:db8:abcd:0010::/64    Server VLAN 1
2001:db8:abcd:0011::/64    Server VLAN 2
2001:db8:abcd:0100::/64    Guest WiFi
2001:db8:abcd:0200::/64    IoT devices
2001:db8:abcd:ffff::/64    Out-of-band management
```

You don't have to use the subnet ID space densely. People often pick numbers that mean something (e.g., `0010` for a server VLAN starts with "internal infrastructure" hex). You will literally never exhaust the `/48` you've been given. There's room for hundreds of thousands of subnets, and you'll use a couple hundred at most.

### Why we don't worry about IPv6 address scarcity

Even if you did burn through your `/48` (you won't), you can ask for a bigger prefix. Even if every site in the world got a `/48`, the whole `2000::/3` space (the global unicast block) holds `2^45 = 35 trillion` `/48`s. Compare to:

- 8 billion humans on Earth → `4000` /48s per person.
- 35 billion devices in the IoT estimate → `1000` /48s per device.

There is no math by which IPv6 runs out at any plausible rate of allocation. The space is so big it isn't a meaningful constraint.

## IPv6 Address Types

### Address-type prefixes

IPv6 addresses are categorized by their leading bits. You can tell what kind of address something is just by looking at its first few hex digits.

```
PREFIX       BLOCK              TYPE              ANALOG IN IPv4
2000::/3     2000:: – 3fff::    Global Unicast    public IPv4
fc00::/7     fc00:: – fdff::    Unique Local      RFC 1918 (private)
fe80::/10    fe80:: – febf::    Link-Local        169.254/16 (APIPA)
ff00::/8     ff00:: – ffff::    Multicast         224.0.0.0/4
::1/128      ::1                Loopback          127.0.0.1
::/128       ::                 Unspecified       0.0.0.0
::ffff:0:0/96 ::ffff:0:0:0/96   IPv4-mapped       wrapped IPv4
64:ff9b::/96 64:ff9b::/96       NAT64             —
```

### Global unicast (2000::/3)

Real internet-routable addresses. All current global unicast allocations come from `2000::/3`, which spans `2000::` through `3fff::`. Most addresses you'll ever see in the wild start with `2`. Examples: Google's IPv6 (`2001:4860::`), Cloudflare's (`2606:4700::`), example documentation block (`2001:db8::/32`).

### Unique local (fc00::/7)

Like RFC 1918 in IPv4. Use these for private networks that don't get routed on the public internet. The full range is `fc00::/7`, but in practice everybody uses the `fd00::/8` half (the lower half), generating a random 40-bit "global ID" so two networks merging probably won't collide.

Format: `fd<random 40 bits>::<subnet>::<host>` — for example `fd5b:9c4f:8a01::/48` for a site, with `fd5b:9c4f:8a01:0001::/64` as one subnet inside.

### Link-local (fe80::/10)

Every IPv6 interface automatically gets a link-local address starting with `fe80::`. It's used for things like Neighbor Discovery (the IPv6 equivalent of ARP), router advertisements, and DHCPv6. Link-local addresses **never leave the link** — routers will not forward them. Their scope is "this physical segment."

You can see your link-local address with:

```
$ ip -6 addr show
... inet6 fe80::a00:27ff:fe12:3456/64 scope link ...
```

If two computers are connected by a single Ethernet cable with no router, they can talk to each other immediately using their link-local addresses. This is one of the magical things about IPv6: it just works without DHCP.

### Multicast (ff00::/8)

Used for one-to-many delivery. The second hex digit after `ff` encodes the **scope** (link-local, site-local, organization-local, global). Some well-known IPv6 multicast addresses:

```
ff02::1     all nodes on this link
ff02::2     all routers on this link
ff02::5     all OSPF routers (link scope)
ff02::6     all OSPF designated routers
ff02::1:2   all DHCP servers (link scope)
ff05::1:3   all DHCP servers (site scope)
```

Where IPv4 had broadcast (one packet to everybody on the segment), IPv6 uses multicast for the same purpose with finer control.

### Loopback `::1/128` and unspecified `::/128`

`::1` is the IPv6 loopback. Like `127.0.0.1` in IPv4. It refers to "this very machine."

`::` (the all-zero address) is the **unspecified address**, used to mean "no address yet." It's the IPv6 equivalent of `0.0.0.0`. Used as the source address of DHCPv6 SOLICIT messages and similar "I don't have an address yet" situations.

### IPv4-mapped (`::ffff:0:0/96`)

Used internally by dual-stack systems to represent an IPv4 address inside an IPv6 socket. Format: `::ffff:a.b.c.d` where `a.b.c.d` is a normal IPv4 address. Example: `::ffff:192.168.1.1` is the IPv4 address `192.168.1.1` written in IPv6 form. You only see these in code and packet captures, not on the wire.

### Picking the right type

For most everyday networks:

- Public servers, public-facing endpoints: **global unicast** (from your provider's `/48` or `/56`).
- Internal-only segments where you want stability without depending on ISP-provided addressing: **unique local** (`fd00::/8` with a random global ID).
- Anything that needs to talk to a neighbor on the same segment with zero config: **link-local** is automatic.

## Hands-On

These are commands you can run right now. The expected output is shown after each. Your numbers may differ in some places (your IP, your hostname, your network) — that's expected.

### Base conversion with printf

```
$ printf "%d\n" 0xff
255

$ printf "%d\n" 0x100
256

$ printf "%d\n" 0xC0A80101
3232235777

$ printf "%x\n" 255
ff

$ printf "%x\n" 192
c0

$ printf "%X\n" 255
FF

$ printf "%o\n" 255
377

$ printf "%#x\n" 255
0xff
```

`%d` is decimal. `%x` is lowercase hex. `%X` is uppercase hex. `%o` is octal. `%#x` adds the `0x` prefix automatically.

### Base conversion with bc

`bc` is a command-line calculator that's installed almost everywhere. Setting `obase` controls the **output** base; `ibase` controls the **input** base. Always set `ibase` last (otherwise the change to `ibase` itself confuses the parser).

```
$ echo "obase=2; 255" | bc
11111111

$ echo "obase=2; 192" | bc
11000000

$ echo "obase=16; 255" | bc
FF

$ echo "ibase=2; 11111111" | bc
255

$ echo "ibase=16; FF" | bc
255

$ echo "ibase=16; obase=2; FF" | bc
11111111

$ echo "obase=2; ibase=10; 200" | bc
11001000
```

### Quick three-base check with python3

```
$ python3 -c "print(bin(255), hex(255), oct(255))"
0b11111111 0xff 0o377

$ python3 -c "print(bin(192), hex(192))"
0b11000000 0xc0

$ python3 -c "print(int('0xFF', 16), int('0b11111111', 2))"
255 255

$ python3 -c "print(format(0xC0A80101, '032b'))"
11000000101010000000000101100100
```

### IPv4 dotted-quad ↔ 32-bit integer in python3

```
$ python3 -c "import ipaddress; print(int(ipaddress.IPv4Address('192.168.1.1')))"
3232235777

$ python3 -c "import ipaddress; print(ipaddress.IPv4Address(3232235777))"
192.168.1.1

$ python3 -c "print(format(192*256**3 + 168*256**2 + 1*256 + 1, '08x'))"
c0a80101

$ python3 -c "print(format(192*256**3 + 168*256**2 + 1*256 + 1, '032b'))"
11000000101010000000000101100100
```

### Subnet calculations with ipcalc

`ipcalc` is the classic terminal tool for IPv4 subnet math. Install it on Debian/Ubuntu with `apt install ipcalc`, on Mac with `brew install ipcalc`. (There are several `ipcalc` implementations; output may look slightly different.)

```
$ ipcalc 192.168.1.0/24
Address:   192.168.1.0          11000000.10101000.00000001. 00000000
Netmask:   255.255.255.0 = 24   11111111.11111111.11111111. 00000000
Wildcard:  0.0.0.255            00000000.00000000.00000000. 11111111
=>
Network:   192.168.1.0/24       11000000.10101000.00000001. 00000000
HostMin:   192.168.1.1          11000000.10101000.00000001. 00000001
HostMax:   192.168.1.254        11000000.10101000.00000001. 11111110
Broadcast: 192.168.1.255        11000000.10101000.00000001. 11111111
Hosts/Net: 254                   Class C, Private Internet
```

```
$ ipcalc -p 10.0.0.0/8
Hosts/Net: 16777214
Network:   10.0.0.0/8

$ ipcalc 192.168.1.0/26
Address:   192.168.1.0
Netmask:   255.255.255.192 = 26
Wildcard:  0.0.0.63
Network:   192.168.1.0/26
HostMin:   192.168.1.1
HostMax:   192.168.1.62
Broadcast: 192.168.1.63
Hosts/Net: 62
```

```
$ ipcalc 2001:db8::/32
(Some ipcalc versions accept IPv6; if yours says "Cannot parse", use sipcalc instead.)
```

### IPv6 subnet calculations with sipcalc

```
$ sipcalc 192.168.1.0/24
-[ipv4 : 192.168.1.0/24] - 0
[CIDR]
Host address     - 192.168.1.0
Host address (decimal) - 3232235776
Host address (hex)     - C0A80100
Network address  - 192.168.1.0
Network mask     - 255.255.255.0
Network mask (bits)  - 24
Network mask (hex)   - FFFFFF00
Broadcast address  - 192.168.1.255
Cisco wildcard   - 0.0.0.255
Addresses in network  - 256
Network range    - 192.168.1.0 - 192.168.1.255
Usable range     - 192.168.1.1 - 192.168.1.254

$ sipcalc 2001:db8::/32
-[ipv6 : 2001:db8::/32] - 0
[IPV6 INFO]
Expanded Address - 2001:0db8:0000:0000:0000:0000:0000:0000
Compressed address - 2001:db8::
Subnet prefix (masked)  - 2001:db8::/32
Address ID (masked)  - 0:0:0:0:0:0:0:0/32
Prefix address   - ffff:ffff:0:0:0:0:0:0
Prefix length    - 32
Address type     - Aggregatable Global Unicast Addresses
Network range    - 2001:db8:: - 2001:db8:ffff:ffff:ffff:ffff:ffff:ffff
```

### Linux routing decisions

```
$ ip route get 8.8.8.8
8.8.8.8 via 192.168.1.1 dev eth0 src 192.168.1.42 uid 1000 cache

$ ip route show match 0.0.0.0
default via 192.168.1.1 dev eth0
192.168.1.0/24 dev eth0 proto kernel scope link src 192.168.1.42

$ ip -6 route show
::1 dev lo proto kernel metric 256 pref medium
fe80::/64 dev eth0 proto kernel metric 256 pref medium

$ ip -4 addr show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
    inet 192.168.1.42/24 brd 192.168.1.255 scope global eth0
       valid_lft forever preferred_lft forever

$ ip -6 addr show eth0
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 ...
    inet6 fe80::a00:27ff:fe12:3456/64 scope link
       valid_lft forever preferred_lft forever
```

### Subnet membership and host enumeration in python3

```
$ python3 -c "import ipaddress; n=ipaddress.ip_network('192.168.1.0/26'); print(list(n.hosts())[:5])"
[IPv4Address('192.168.1.1'), IPv4Address('192.168.1.2'), IPv4Address('192.168.1.3'), IPv4Address('192.168.1.4'), IPv4Address('192.168.1.5')]

$ python3 -c "import ipaddress; print(ipaddress.ip_address('192.168.1.1') in ipaddress.ip_network('192.168.0.0/16'))"
True

$ python3 -c "import ipaddress; print(ipaddress.ip_address('10.0.0.1') in ipaddress.ip_network('192.168.0.0/16'))"
False

$ python3 -c "import ipaddress; n=ipaddress.IPv6Network('2001:db8::/64'); print(n.broadcast_address)"
2001:db8::ffff:ffff:ffff:ffff
```

(Note: IPv6 doesn't actually have a broadcast address — `broadcast_address` in the Python library returns the highest address in the prefix anyway, which is conceptually similar.)

### Splitting a /22 into /24s

```
$ python3 -c "import ipaddress; n=ipaddress.ip_network('10.0.0.0/22')
for s in n.subnets(new_prefix=24): print(s)"
10.0.0.0/24
10.0.1.0/24
10.0.2.0/24
10.0.3.0/24
```

### WHOIS and ASN lookup

```
$ whois 8.8.8.8 | head -20
NetRange:       8.8.8.0 - 8.8.8.255
CIDR:           8.8.8.0/24
NetName:        GOGL
NetHandle:      NET-8-8-8-0-2
Parent:         NET8 (NET-8-0-0-0-0)
NetType:        Direct Allocation
OriginAS:       AS15169
Organization:   Google LLC (GOGL)
RegDate:        2014-03-14
Updated:        2014-03-14
Ref:            https://rdap.arin.net/registry/ip/8.8.8.0
...

$ dig +short -t TXT 8.8.8.8.origin.asn.cymru.com
"15169 | 8.8.8.0/24 | US | arin | 2014-03-14"
```

(The Cymru TXT record gives you origin AS, the prefix, the country, the RIR, and the allocation date. Great for a fast lookup.)

### Host discovery on a subnet

```
$ nmap -sn 192.168.1.0/24
Starting Nmap 7.94 ( https://nmap.org ) at 2026-04-27 12:34 UTC
Nmap scan report for router.lan (192.168.1.1)
Host is up (0.0012s latency).
Nmap scan report for laptop.lan (192.168.1.42)
Host is up (0.00050s latency).
Nmap scan report for nas.lan (192.168.1.50)
Host is up (0.0011s latency).
Nmap done: 256 IP addresses (3 hosts up) scanned in 2.84 seconds
```

`-sn` means "ping scan, no port scan" — just discover which hosts are alive on the subnet.

### DNS to IPs

```
$ getent ahosts google.com
142.250.190.78  STREAM google.com
142.250.190.78  DGRAM
142.250.190.78  RAW
2607:f8b0:4005:80c::200e STREAM
2607:f8b0:4005:80c::200e DGRAM
2607:f8b0:4005:80c::200e RAW

$ host google.com
google.com has address 142.250.190.78
google.com has IPv6 address 2607:f8b0:4005:80c::200e
google.com mail is handled by 10 smtp.google.com.
```

### Bitwise operations in Python

```
$ python3 -c "print(bin(0b1010 & 0b1100))"
0b1000

$ python3 -c "print(bin(0b1010 | 0b1100))"
0b1110

$ python3 -c "print(bin(0b1010 ^ 0b1100))"
0b110

$ python3 -c "print(bin(1 << 8))"
0b100000000
```

### Bitwise operations in cs's calculator

`cs` ships with a built-in calculator that supports bitwise operators:

```
$ cs calc "0xFF & 0x0F"
15

$ cs calc "0xFF | 0x100"
511

$ cs calc "1 << 16"
65536

$ cs calc "0xC0A80101"
3232235777
```

### cs's built-in subnet tool

```
$ cs subnet 192.168.1.0/24
Network:    192.168.1.0/24
Netmask:    255.255.255.0
Wildcard:   0.0.0.255
Hosts:      254
Range:      192.168.1.0 - 192.168.1.255
Usable:     192.168.1.1 - 192.168.1.254

$ cs subnet 2001:db8::/32
Network:    2001:db8::/32
Prefix:     32
Range:      2001:db8:: - 2001:db8:ffff:ffff:ffff:ffff:ffff:ffff
Type:       Global Unicast (2000::/3)
```

### Find your link-local IPv6 address

```
$ ip -6 addr show | grep fe80
    inet6 fe80::a00:27ff:fe12:3456/64 scope link
```

### Show the kernel's IPv4 forwarding trie

```
$ cat /proc/net/fib_trie | head -30
Main:
  +-- 0.0.0.0/0 3 0 5
     |-- 0.0.0.0
        /0 universe UNICAST
     +-- 127.0.0.0/8 2 0 2
        |-- 127.0.0.0
           /8 link BROADCAST
           /32 host LOCAL
        |-- 127.0.0.1
           /32 host LOCAL
...
```

This is the actual data structure the Linux kernel uses for longest-prefix-match lookups. Looking at it once is useful; you don't need to memorize it.

### Convert a whole /16 into bytes by yourself

```
$ python3 -c "import ipaddress; n=ipaddress.IPv4Network('192.168.0.0/16'); print(n.num_addresses)"
65536

$ python3 -c "import ipaddress; print(int(ipaddress.IPv4Address('192.168.0.0')).to_bytes(4,'big').hex())"
c0a80000
```

`.to_bytes(4, 'big').hex()` gives you the 32-bit address as 4 bytes in big-endian (network byte order) hex.

### Test a CIDR claim

```
$ python3 -c "import ipaddress; print(ipaddress.ip_network('192.168.1.0/24', strict=True))"
192.168.1.0/24

$ python3 -c "import ipaddress; print(ipaddress.ip_network('192.168.1.5/24', strict=True))"
ValueError: 192.168.1.5/24 has host bits set
```

The `strict=True` mode complains if you give it a CIDR with host bits set (because `192.168.1.5` is not a network address). With `strict=False`, Python silently masks off the host bits to give you the network number.

## Common Confusions

A long list of things that trip everybody up. Read these one at a time.

### "Is 192.168.1.0 a usable host on a /24?"

**No.** It's the network address. The first address in any /24 (or larger) subnet is reserved as the network address, used to identify the network itself. Hosts get `192.168.1.1` through `192.168.1.254`. Confusingly, you can technically send a packet to `192.168.1.0` and some networks will deliver it, but per RFC convention it should never be assigned to a host.

### "Is 192.168.1.255 a usable host on a /24?"

**No.** It's the broadcast address. The last address in any /24 (or larger) subnet is reserved as broadcast. Sending to it reaches every host on the segment. You cannot assign it to a real machine.

### "Why is /31 sometimes used instead of /30 for point-to-point?"

A `/30` has 4 addresses but only 2 are usable for hosts (the network and broadcast addresses are wasted). On a router-to-router link you only have 2 endpoints, so half the addresses are wasted. RFC 3021 (year 2000) said: "for point-to-point links specifically, treat /31 as having 2 usable addresses, no network/broadcast." Most modern routers support /31, and this halves the IPv4 address consumption for inter-router links. Use /31 if your equipment supports it.

### "Why is /32 valid?"

A /32 has zero host bits — the "subnet" is exactly one address. This is useful for:

- **Router loopbacks** — a stable address on a router that doesn't depend on any specific physical interface.
- **Single-host routes** — when you want to advertise just one address into BGP/OSPF.
- **Anycast** — same address advertised from many locations; routers pick the closest.
- **DNS root servers** — every root server has a /32 advertisement.

Logically a /32 isn't a "subnet" of anything; it's the smallest possible CIDR block: a single host.

### "Why is IPv6 /64 the smallest practical subnet?"

Because **SLAAC** (RFC 4862) and modern privacy address generation require 64 bits of interface ID space. If you use a smaller subnet like /96 or /112, automatic configuration and several other things break. Routers will sometimes still let you do it, but most operating systems behave badly. The exceptions are /127 for inter-router point-to-point links (RFC 6164) and /128 for single-host routes.

The rest of the time: **always use /64 for IPv6 subnets.** You have so many of them in your /48 (65,536 of them!) that there's no reason to scrimp.

### "Why does my hex calculator say `0xFF.FF.FF.FF` and not `0xFFFFFFFF`?"

It's a presentation choice. Some tools split a 32-bit number into 4 bytes for display, with dots between. The underlying number is the same. `0xFF.FF.FF.FF` is just `0xFFFFFFFF` displayed byte-by-byte. If a tool requires one form or the other, use what it asks for; otherwise either is fine.

### "Are 192.168.1.0 and 192.168.1.0/24 the same thing?"

**No.** `192.168.1.0` is just an IPv4 address (one specific 32-bit value). `192.168.1.0/24` is a network specification: an address paired with a prefix length. The address `192.168.1.0` is meaningful only when you also know what prefix length it sits in.

`192.168.1.0/24` means "the subnet containing 192.168.1.0 through 192.168.1.255." `192.168.1.0/16` means "the subnet containing 192.168.0.0 through 192.168.255.255 — and 192.168.1.0 happens to be a host inside it." Same dotted-decimal address, different meaning depending on the prefix.

### "Is `0.0.0.0` an address or a route?"

Both, depending on context.

- **As a destination address:** `0.0.0.0` means "this host on this network" — used by clients that don't yet have an IP, like during DHCP boot.
- **As a route:** `0.0.0.0/0` is the **default route**, meaning "everything not covered by a more specific route." Every router has one (or it can't reach the internet).

Different roles in different sentences. Same number.

### "Why do I sometimes see a network in CIDR but called /24, and sometimes called 255.255.255.0?"

They are exactly the same thing, written two different ways. CIDR is more compact and is the modern preferred notation. The dotted mask is older and more verbose. Routers, configuration files, and books all use both. Be comfortable converting between them in your head:

- /24 → 255.255.255.0
- /16 → 255.255.0.0
- /8 → 255.0.0.0
- /26 → 255.255.255.192
- /28 → 255.255.255.240

### "How can two /24 subnets `10.1.0.0/24` and `10.1.1.0/24` summarize as `10.1.0.0/23`?"

Because in binary, `10.1.0.0` is `00001010.00000001.00000000.00000000` and `10.1.1.0` is `00001010.00000001.00000001.00000000`. They share the first 23 bits — they only differ in bit 24 (the lowest bit of the third octet). A /23 captures that common prefix exactly. The pair (`/24`, `/24`) at sibling positions in the trie can always summarize to a single /23.

### "Why does my /28 only have 14 hosts when 2^4 = 16?"

Because two of the 16 addresses are reserved (network and broadcast). 16 - 2 = 14 usable hosts. Same pattern in every CIDR size /30 and bigger.

### "Why does my home network have addresses like 192.168.0.x and the office has 10.x.x.x?"

Both are private RFC 1918 ranges. Home routers often default to `192.168.0.0/24` or `192.168.1.0/24` because that's small and fits a typical home with a dozen devices. Offices often use `10.0.0.0/8` because it has 16 million addresses, leaving lots of room to subnet by floor, building, department, or project. Either choice works; they're conventions.

### "Why is `127.0.0.1` always 'localhost' but I see `127.5.4.3` work too?"

The whole `127.0.0.0/8` block is reserved for loopback. Almost everything uses `127.0.0.1` by tradition, but other addresses in the range also loop back to your own machine. You can run a local web server on `127.5.4.3:8080` and your browser will reach it at that address. It's the same machine, just a different address in the loopback range.

### "Why doesn't my browser show IPv6 addresses?"

Most websites have both IPv4 and IPv6 entries in DNS. Modern operating systems prefer IPv6 when available (this is called "Happy Eyeballs," RFC 8305 — try both, use whichever connects faster). If your network supports IPv6 you're probably already using it for half your traffic and just don't realize. Try `getent ahosts google.com` to see both addresses.

### "Why is the `::` in IPv6 sometimes ambiguous?"

You can only use `::` once per address. If an address has multiple zero-runs, only one can be compressed. Otherwise the receiver wouldn't know how to expand it.

```
2001:0:0:1:0:0:0:1
   has two runs of zeros: positions 2-3 (length 2) and positions 5-7 (length 3).
   You can only compress one. Per RFC 5952, prefer the longer one:
   2001:0:0:1::1   ✓ correct
   2001::1:0:0:0:1 also valid representation but RFC 5952 prefers the longer compression
   2001::1::1      ✗ ILLEGAL — two `::` would be ambiguous
```

### "Why does the broadcast address vary by subnet?"

Because broadcast = "all host bits set to 1" — and which bits are host bits depends on the subnet mask. In `192.168.1.0/24` the broadcast is `192.168.1.255`. In `192.168.1.0/26` the broadcast is `192.168.1.63`. In `192.168.1.64/26` the broadcast is `192.168.1.127`. Same /24 of address space; four different broadcast addresses depending on how it's subnetted.

### "Are the bytes in an IP address big-endian or little-endian?"

IP addresses on the wire are **big-endian** (most significant byte first). This is called **network byte order**. So `192.168.1.1` is sent as the byte sequence `0xC0 0xA8 0x01 0x01`. When you read it from a packet capture or compute it from raw bytes, that's the order. C programmers know this as `htonl()` / `ntohl()` — host-to-network and network-to-host conversions for 32-bit values.

### "Why are some MAC addresses written 00:1A:2B and some 001A.2B...?"

Different conventions. Cisco-style writes MACs as three groups of four hex digits separated by dots: `001a.2b3c.4d5e`. IEEE-style writes them as six pairs of hex digits separated by colons or dashes: `00:1a:2b:3c:4d:5e` or `00-1a-2b-3c-4d-5e`. Same 48 bits underneath. Tools usually accept multiple formats; output uses whichever style the vendor prefers.

## Vocabulary

### Counting and bit-level

- **bit** — A single binary digit (0 or 1). The smallest unit of information.
- **byte** — Eight bits in a row. The smallest unit most computers actually work with.
- **nibble** — Four bits — half a byte. Holds exactly one hex digit.
- **word** — A unit of memory. Often 16 bits, but depends on the CPU. Sometimes "word" means "the natural width of the CPU."
- **dword** — Double word — 32 bits.
- **qword** — Quad word — 64 bits.
- **octet** — Eight bits. Networking term, equivalent to "byte" but unambiguous.
- **hextet** — Sixteen bits — one of the eight groups in an IPv6 address.
- **MSB** — Most significant bit. The leftmost bit, with the highest place value.
- **LSB** — Least significant bit. The rightmost bit, with place value 1.
- **endianness** — The order in which bytes of a multi-byte number are stored. Big-endian = highest byte first; little-endian = lowest byte first.
- **big-endian** — Most significant byte first (left). Used on the wire (network byte order).
- **little-endian** — Least significant byte first (left). Used internally on x86 and ARM CPUs.
- **network byte order** — Big-endian. The byte order used to send numbers over a network.

### Number systems

- **binary** — Base 2. Two digits (0, 1).
- **decimal** — Base 10. Ten digits (0–9).
- **hex** — Short for hexadecimal. Base 16. Sixteen digits (0–9, A–F).
- **hexadecimal** — Same as hex.
- **octal** — Base 8. Eight digits (0–7). Used for Unix file modes.
- **base** — The number of unique digits in a number system. Base 10 has ten digits, base 2 has two, etc.
- **place value** — The value of a column in a positional number system. In binary, columns are worth 1, 2, 4, 8, 16, ...
- **positional notation** — A number system where each digit's value depends on its position.

### Signed numbers (mostly side reading)

- **two's complement** — The way negative integers are represented in binary. Take the positive value, flip all bits, add 1.
- **sign bit** — The MSB in a signed number. 0 = positive, 1 = negative.
- **unsigned** — A number with no sign bit; can only be zero or positive.
- **signed** — A number with a sign bit; can be negative.

### Bitwise operations

- **AND** — Bitwise operation: 1 if both bits are 1, else 0. Used for masking.
- **OR** — Bitwise operation: 1 if either bit is 1, else 0. Used for setting bits.
- **XOR** — Exclusive OR. 1 if exactly one bit is 1. Used for toggling and parity.
- **NOT** — Bitwise inversion. 0 becomes 1 and 1 becomes 0.
- **shift** — Move bits left or right. `<<` shifts left (multiplies by 2 each step); `>>` shifts right (divides).
- **bitmask** — A pattern of bits used to extract or modify specific bits via AND, OR, XOR.
- **mask** — General term for a pattern that selects which bits matter.
- **set (a bit)** — Force a bit to 1, usually by ORing.
- **clear (a bit)** — Force a bit to 0, usually by ANDing with the inverse.
- **toggle (a bit)** — Flip a bit, usually by XORing with 1.
- **test (a bit)** — Check if a bit is 1, usually by ANDing with a mask and checking nonzero.

### IP addresses

- **IPv4** — Internet Protocol version 4. 32-bit addresses. Dotted-quad notation.
- **IPv6** — Internet Protocol version 6. 128-bit addresses. Colon-hex notation.
- **dotted-quad** — The IPv4 notation: four decimal octets separated by dots, like `192.168.1.1`.
- **dotted-hex** — A way to write hex bytes separated by dots, like `0xC0.A8.01.01` for `192.168.1.1`. Rare in practice.
- **colon-hex** — The IPv6 notation: hex hextets separated by colons.

### CIDR and prefix

- **CIDR** — Classless Inter-Domain Routing. The system of writing networks as `address/prefix-length`.
- **prefix length** — The number of leading network bits, e.g., `/24`. Same as the number of 1s in the subnet mask.
- **slash notation** — Same as CIDR notation. The `/24` style.
- **subnet** — A subdivision of a network — a CIDR block.
- **supernet** — A bigger CIDR block that contains multiple smaller ones (the result of summarization).
- **mask** — Subnet mask. A 32-bit value with 1s for network bits, 0s for host bits.
- **wildcard** — Inverse of subnet mask: 0s for network, 1s for host. Used by Cisco ACLs.
- **/8, /16, /24, /28, /30, /31, /32** — Common IPv4 prefix lengths.
- **/48, /56, /64, /127, /128** — Common IPv6 prefix lengths.

### Address roles within a subnet

- **network address** — The first address in a subnet, all host bits 0. Identifies the network. Not a usable host.
- **broadcast address** — The last address in a subnet, all host bits 1. Reaches all hosts on the segment.
- **host address** — Any address inside a subnet that's not network or broadcast.
- **usable hosts** — Total addresses minus network and broadcast. For /n with n ≤ 30, that's `2^(32-n) - 2`.

### Classes (legacy) and CIDR

- **classful** — The pre-1993 system that fixed IPv4 prefix lengths to /8, /16, or /24 based on the first octet. Replaced by CIDR.
- **classless** — The modern system where any prefix length 0..32 is valid. CIDR.

### RFCs and special blocks

- **RFC 1918** — The private IPv4 ranges: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16.
- **RFC 4193** — Unique-local IPv6 (fc00::/7, in practice fd00::/8).
- **RFC 3021** — /31 for point-to-point links (no broadcast).
- **RFC 6598** — CGNAT shared range 100.64.0.0/10.
- **RFC 5952** — Recommended canonical text representation of IPv6 addresses.
- **RFC 4291** — IPv6 addressing architecture.
- **RFC 4632** — CIDR specification.
- **RFC 5737** — Documentation IPv4 ranges (TEST-NET).
- **link-local** — Address valid only on the local link (IPv4 169.254/16, IPv6 fe80::/10).
- **unique-local** — IPv6 private addresses (fc00::/7).
- **global** — Routable on the public internet (IPv6 2000::/3 in practice).
- **multicast** — One-to-many delivery. IPv4 224/4, IPv6 ff00::/8.
- **anycast** — Same address advertised from many places; routing picks the nearest.
- **loopback** — "This very machine." 127.0.0.1 and ::1.
- **unspecified** — "No address yet." 0.0.0.0 and ::.
- **default route** — 0.0.0.0/0 (or ::/0) — "everything else."
- **APIPA** — Automatic Private IP Addressing. Microsoft term for 169.254/16 link-local.

### Subnetting and design

- **subnetting** — Splitting a network into smaller pieces.
- **VLSM** — Variable-Length Subnet Masking. Different subnets get different sizes.
- **FLSM** — Fixed-Length Subnet Masking. All subnets the same size.
- **route summarization** — Combining contiguous subnets into one bigger advertisement.
- **supernetting** — Same as summarization.
- **aggregation** — Same as summarization; common BGP term.
- **CGNAT** — Carrier-Grade NAT. ISPs sharing one public IP among many customers.
- **NAT** — Network Address Translation. Mapping private addresses to a public address.
- **hairpin** — When traffic from inside a network goes out and comes back in (e.g., to reach a server on the same LAN by its public IP).

### Routing

- **longest-prefix match** — A router picks the routing table entry with the longest prefix that matches the destination address. The fundamental rule of IP forwarding.
- **FIB** — Forwarding Information Base. The routing table actually used to forward packets.
- **RIB** — Routing Information Base. The full routing knowledge a router has, before policy and best-path selection produce the FIB.
- **trie** — A tree-like data structure where each branch represents a bit; used for fast prefix lookups.
- **LPM trie** — A trie organized for longest-prefix matching. Linux's `fib_trie` and BPF's `BPF_MAP_TYPE_LPM_TRIE` use this.

### IPv6-specific

- **EUI-64** — A way to derive a 64-bit interface ID from a 48-bit MAC by inserting `ff:fe` and flipping the universal/local bit.
- **modified EUI-64** — Same as EUI-64; the "modified" refers to flipping the U/L bit.
- **SLAAC** — StateLess Address AutoConfiguration. Hosts derive their own IPv6 address from router advertisements.
- **DAD** — Duplicate Address Detection. A host checks no one else is using the address it just made up.
- **NDP** — Neighbor Discovery Protocol. The IPv6 equivalent of ARP, plus router discovery and other things.
- **RA** — Router Advertisement. Router-sent message that tells hosts what prefix to use.
- **prefix delegation** — DHCPv6-PD. An ISP gives a customer router a /48 or /56 to subnet internally.
- **DHCPv6-PD** — Prefix delegation flavor of DHCPv6.
- **IPAM** — IP Address Management. The discipline (and tools) of tracking who has which addresses.

### Misc helpful terms

- **2^n table** — Memorize: 2^0=1, 2^1=2, 2^2=4, 2^3=8, 2^4=16, 2^5=32, 2^6=64, 2^7=128, 2^8=256, 2^9=512, 2^10=1024, 2^16=65536, 2^32=~4.3 billion.
- **network number** — Same as network address. The first address in a subnet.
- **interface** — A physical or virtual port on a host. Each interface has zero or more IP addresses.
- **dual-stack** — A host or network running both IPv4 and IPv6 simultaneously.
- **happy eyeballs** — RFC 8305. A client tries IPv6 and IPv4 in parallel and uses whichever connects first.
- **Class A / B / C / D / E** — Legacy classful names; mostly kept for trivia. (See "classful" above.)

## Try This

A few things to do, hands-on, with your own machine.

### Experiment 1: Convert your home subnet to binary by hand

Look at your home gateway's IP. It's usually `192.168.0.1` or `192.168.1.1` or similar. Run `ip -4 addr show` (Linux) or `ifconfig` (Mac/BSD). Find your IP and the prefix.

Now, with paper and pencil, write each octet in binary. Then write the mask in binary. Then AND them. Did you get the same network number that `ip route` shows? Now check with `ipcalc`:

```
$ ipcalc 192.168.1.42/24
```

The "Address" line should show your IP in binary. The "Network" line should show the network you computed by hand. They should match.

### Experiment 2: Compute the broadcast address of a /27

Pick a /27, say `10.0.0.96/27`. Without using `ipcalc`, compute:

- The network address.
- The first usable host.
- The last usable host.
- The broadcast address.

Now check with `ipcalc 10.0.0.96/27`. Did you get it?

(Answers: network 10.0.0.96, first 10.0.0.97, last 10.0.0.126, broadcast 10.0.0.127.)

### Experiment 3: Find your IPv6 link-local address

Every interface has one. It starts with `fe80::`.

```
$ ip -6 addr show | grep fe80
```

If your machine is plugged into a network or has Wi-Fi, you'll see one. The 64-bit interface ID is either derived from your MAC (modified EUI-64 — has `ff:fe` in the middle) or randomly generated for privacy.

### Experiment 4: Plan a small office VLSM

You have `192.168.10.0/24` to work with. Allocate:

- 1 subnet of 100 hosts (engineering).
- 1 subnet of 50 hosts (sales).
- 1 subnet of 14 hosts (servers).
- 2 subnets of 2 hosts each (router-to-router links).

Show the answer largest-first. Then check using `ipcalc` that each subnet you came up with is correctly sized.

### Experiment 5: Look up a famous IP

```
$ whois 8.8.8.8 | head -30
$ whois 1.1.1.1 | head -30
$ whois 208.67.222.222 | head -30
```

Whose are these? (Google. Cloudflare. OpenDNS, originally; later Cisco.) See what CIDR each one is allocated within.

### Experiment 6: See how many IPv6 addresses google.com has

```
$ getent ahosts google.com | grep STREAM
```

You may see several IPv4 and IPv6 entries. Look at the prefixes — which AS owns each one? Cross-reference with the Cymru ASN lookup:

```
$ dig +short -t TXT 8.8.8.8.origin.asn.cymru.com
```

### Experiment 7: Watch your kernel pick a route

```
$ ip route get 8.8.8.8
$ ip route get 1.1.1.1
$ ip route get 192.168.1.1   # your local gateway
$ ip route get 127.0.0.1
```

Each one should show a different next-hop and source address. The kernel is doing longest-prefix-match against your routing table.

### Experiment 8: Pick subnets out of a /22

```
$ python3 -c "import ipaddress; n=ipaddress.ip_network('10.0.0.0/22')
for s in n.subnets(new_prefix=24): print(s)"
```

Now do the same with `new_prefix=26`, `new_prefix=28`, and `new_prefix=30`. Notice how many subnets you get each time. (4, 16, 64, 256.)

### Experiment 9: Drill the powers of 2

Cover the right column. Recite from memory:

```
2^0  = 1
2^1  = 2
2^2  = 4
2^3  = 8
2^4  = 16
2^5  = 32
2^6  = 64
2^7  = 128
2^8  = 256
2^10 = 1024
2^16 = 65,536
```

Until you can do 2^0 through 2^16 in your sleep.

### Experiment 10: Find a /32 in your routing table

```
$ ip route show | grep '/32'
```

(May not have any. If you have a Tailscale or WireGuard interface, or you've configured loopback addresses on a router, you'll see /32s.)

## Where to Go Next

When this sheet feels easy, move up to the dense reference and the protocol companion:

- `cs fundamentals binary-and-number-systems` — engineer-grade reference: number systems, encoding details, ASCII/UTF-8, two's-complement gotchas.
- `cs networking ip` — the IP protocol, big picture.
- `cs networking ipv4` — IPv4 protocol details.
- `cs networking ipv6` — IPv6 protocol details.
- `cs networking ipv6-advanced` — Neighbor Discovery, SLAAC, RAs, multicast.
- `cs networking dhcp` — IPv4 dynamic addressing.
- `cs networking dhcpv6` — IPv6 dynamic addressing including prefix delegation.
- `cs networking dns` — name-to-address resolution.
- `cs ramp-up ip-eli5` — the protocol side, in plain English. The companion to this sheet.
- `cs ramp-up tcp-eli5` — how reliable connections work.
- `cs ramp-up udp-eli5` — how the connectionless side works.
- `cs ramp-up linux-kernel-eli5` — how the kernel handles all of this for you.
- `cs subnet 10.0.0.0/24` — the built-in subnet calculator.
- `cs calc "1<<16"` — the built-in calculator with bitwise ops.

## See Also

- `fundamentals/binary-and-number-systems` — engineer-grade reference for number systems and encodings.
- `networking/ip` — IP protocol overview.
- `networking/ipv4` — IPv4 protocol details.
- `networking/ipv6` — IPv6 protocol details.
- `networking/ipv6-advanced` — NDP, SLAAC, multicast in depth.
- `networking/dhcp` — IPv4 DHCP.
- `networking/dhcpv6` — IPv6 DHCPv6 and prefix delegation.
- `networking/dns` — DNS, the protocol that turns names into addresses.
- `ramp-up/ip-eli5` — protocol-side companion to this sheet.
- `ramp-up/tcp-eli5` — reliable streams in plain English.
- `ramp-up/udp-eli5` — connectionless datagrams in plain English.
- `ramp-up/linux-kernel-eli5` — the kernel's role in networking.

## References

- **RFC 4632** — Classless Inter-Domain Routing (CIDR): The Internet Address Assignment and Aggregation Plan.
- **RFC 1918** — Address Allocation for Private Internets (the famous `10/8`, `172.16/12`, `192.168/16` private blocks).
- **RFC 4193** — Unique Local IPv6 Unicast Addresses (`fc00::/7`).
- **RFC 5952** — A Recommendation for IPv6 Address Text Representation (canonical compression).
- **RFC 4291** — IP Version 6 Addressing Architecture (the master IPv6 document).
- **RFC 3021** — Using 31-Bit Prefixes on IPv4 Point-to-Point Links.
- **RFC 6598** — IANA-Reserved IPv4 Prefix for Shared Address Space (the CGNAT block).
- **RFC 5737** — IPv4 Address Blocks Reserved for Documentation.
- **RFC 6164** — Using 127-Bit IPv6 Prefixes on Inter-Router Links.
- **RFC 4862** — IPv6 Stateless Address Autoconfiguration (SLAAC).
- **RFC 4861** — Neighbor Discovery for IP version 6 (NDP).
- **RFC 1122** — Requirements for Internet Hosts (the foundational host behavior RFC).
- **RFC 6890** — Special-Purpose IP Address Registries (the canonical "what's reserved" list).
- **RFC 8305** — Happy Eyeballs Version 2: Better Connectivity Using Concurrency.
- `man ipcalc` — IPv4 subnet calculator manual page.
- `man sipcalc` — IPv4/IPv6 subnet calculator manual page.
- `man bc` — arbitrary-precision calculator language.
- `man printf` — formatted output, including `%x`, `%d`, `%o`, `%b`.
- `man ip` — Linux IP routing and address tool (`ip addr`, `ip route`, `ip -6`).
- `man 7 ipv6` — Linux IPv6 implementation overview.
- **"TCP/IP Illustrated, Volume 1"** by W. Richard Stevens and Kevin R. Fall — the canonical book on IP, with full-color packet captures and binary-level walkthroughs.
- **"Practical Packet Analysis"** by Chris Sanders — friendly intro to reading packet captures, including IP-level details.
- **"IPv6 Essentials"** by Silvia Hagen — comprehensive IPv6 reference.
- **"Routing TCP/IP, Volume 1"** by Jeff Doyle — classic routing reference, very strong on subnetting.
- **`info coreutils`** — info pages for GNU coreutils, including `printf`. Read with `info coreutils 'printf invocation'`.

— End of ELI5 —

When this sheet feels boring, graduate to `cs fundamentals binary-and-number-systems` for the engineer-grade material on number systems and encoding. After that, `cs networking ipv4` and `cs networking ipv6` will give you the protocol-level details that this sheet deliberately skipped (this sheet was just the math). The companion `cs ramp-up ip-eli5` is where you go for the "what is a packet, who answers, who routes it" story.

### One last thing before you go

Pick a /24 in your head right now. Any /24. Say, `192.0.2.0/24` (a documentation prefix, safe to use in examples). On paper:

- Write the network address in binary.
- Write the broadcast address in binary.
- Write the mask in binary.
- Pick a host address inside the subnet, say `192.0.2.42`. AND it with the mask. Did you get back the network number?

Now type `ipcalc 192.0.2.42/24` and check that the binary in its output matches what you wrote. If it matches, you have just verified that your understanding of subnetting matches what real tools compute. That's everything.

Reading is good. Doing is better. Type the commands. Watch the bits flow. The math is simple — it just looks scary the first time.

You are now officially equipped to figure out subnet masks, count IPs in a CIDR block, and convert between binary, hex, and decimal without leaving the terminal. Welcome.

— End of ELI5 — (really this time!)
