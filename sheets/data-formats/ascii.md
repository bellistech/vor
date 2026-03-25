# ASCII (American Standard Code for Information Interchange)

> 7-bit character encoding standard mapping integers 0-127 to control characters, symbols, digits, and letters.

## Control Characters (0-31, 127)

```
Dec  Hex  Oct  Char  Name                    Common Use
---  ---  ---  ----  ----                    ----------
  0  00   000  NUL   Null                    String terminator (\0)
  1  01   001  SOH   Start of Heading
  2  02   002  STX   Start of Text
  3  03   003  ETX   End of Text             Ctrl+C (interrupt)
  4  04   004  EOT   End of Transmission     Ctrl+D (EOF)
  5  05   005  ENQ   Enquiry
  6  06   006  ACK   Acknowledge
  7  07   007  BEL   Bell                    Terminal beep (\a)
  8  08   010  BS    Backspace               \b
  9  09   011  HT    Horizontal Tab          \t
 10  0A   012  LF    Line Feed               \n (Unix newline)
 11  0B   013  VT    Vertical Tab            \v
 12  0C   014  FF    Form Feed               \f (page break)
 13  0D   015  CR    Carriage Return         \r (Windows newline = \r\n)
 14  0E   016  SO    Shift Out
 15  0F   017  SI    Shift In
 16  10   020  DLE   Data Link Escape
 17  11   021  DC1   Device Control 1        XON (resume transmission)
 18  12   022  DC2   Device Control 2
 19  13   023  DC3   Device Control 3        XOFF (pause transmission)
 20  14   024  DC4   Device Control 4
 21  15   025  NAK   Negative Acknowledge
 22  16   026  SYN   Synchronous Idle
 23  17   027  ETB   End of Trans. Block
 24  18   030  CAN   Cancel
 25  19   031  EM    End of Medium
 26  1A   032  SUB   Substitute              Ctrl+Z (suspend / EOF on Windows)
 27  1B   033  ESC   Escape                  \e (start ANSI sequences)
 28  1C   034  FS    File Separator
 29  1D   035  GS    Group Separator
 30  1E   036  RS    Record Separator
 31  1F   037  US    Unit Separator
127  7F   177  DEL   Delete
```

## Printable Characters (32-126)

```
Dec  Hex  Oct  Char    Dec  Hex  Oct  Char    Dec  Hex  Oct  Char
---  ---  ---  ----    ---  ---  ---  ----    ---  ---  ---  ----
 32  20   040  (space)  64  40   100  @        96  60   140  `
 33  21   041  !        65  41   101  A        97  61   141  a
 34  22   042  "        66  42   102  B        98  62   142  b
 35  23   043  #        67  43   103  C        99  63   143  c
 36  24   044  $        68  44   104  D       100  64   144  d
 37  25   045  %        69  45   105  E       101  65   145  e
 38  26   046  &        70  46   106  F       102  66   146  f
 39  27   047  '        71  47   107  G       103  67   147  g
 40  28   050  (        72  48   110  H       104  68   150  h
 41  29   051  )        73  49   111  I       105  69   151  i
 42  2A   052  *        74  4A   112  J       106  6A   152  j
 43  2B   053  +        75  4B   113  K       107  6B   153  k
 44  2C   054  ,        76  4C   114  L       108  6C   154  l
 45  2D   055  -        77  4D   115  M       109  6D   155  m
 46  2E   056  .        78  4E   116  N       110  6E   156  n
 47  2F   057  /        79  4F   117  O       111  6F   157  o
 48  30   060  0        80  50   120  P       112  70   160  p
 49  31   061  1        81  51   121  Q       113  71   161  q
 50  32   062  2        82  52   122  R       114  72   162  r
 51  33   063  3        83  53   123  S       115  73   163  s
 52  34   064  4        84  54   124  T       116  74   164  t
 53  35   065  5        85  55   125  U       117  75   165  u
 54  36   066  6        86  56   126  V       118  76   166  v
 55  37   067  7        87  57   127  W       119  77   167  w
 56  38   070  8        88  58   130  X       120  78   170  x
 57  39   071  9        89  59   131  Y       121  79   171  y
 58  3A   072  :        90  5A   132  Z       122  7A   172  z
 59  3B   073  ;        91  5B   133  [       123  7B   173  {
 60  3C   074  <        92  5C   134  \       124  7C   174  |
 61  3D   075  =        93  5D   135  ]       125  7D   175  }
 62  3E   076  >        94  5E   136  ^       126  7E   176  ~
 63  3F   077  ?        95  5F   137  _
```

## Common Escape Sequences

```
Sequence  Dec  Hex  Name
--------  ---  ---  ----
\0          0  00   Null terminator
\a          7  07   Bell / alert
\b          8  08   Backspace
\t          9  09   Horizontal tab
\n         10  0A   Newline (line feed)
\v         11  0B   Vertical tab
\f         12  0C   Form feed
\r         13  0D   Carriage return
\e         27  1B   Escape (non-standard in C)
\\         92  5C   Backslash literal
\"         34  22   Double quote literal
\'         39  27   Single quote literal
```

## Quick Ranges

```
Digits:     48-57   (0x30-0x39)    '0'-'9'
Uppercase:  65-90   (0x41-0x5A)    'A'-'Z'
Lowercase:  97-122  (0x61-0x7A)    'a'-'z'

# Case conversion offset: 32 (0x20)
# 'A' (65) + 32 = 'a' (97)
# 'a' (97) - 32 = 'A' (65)
```

## Command-Line Tools

```bash
# Print ASCII table
man ascii

# Character to decimal
printf '%d\n' "'A"          # 65

# Decimal to character
printf '\x41\n'             # A
printf "\\$(printf '%03o' 65)\n"  # A

# Hex dump of a file
xxd file.txt
hexdump -C file.txt

# Show non-printable characters
cat -v file.txt             # Shows ^M for CR, ^I for tab
cat -A file.txt             # Shows $ at line ends too
```

## Tips

- ASCII is a strict subset of UTF-8; all valid ASCII is valid UTF-8.
- The difference between uppercase and lowercase letters is always bit 5 (decimal 32, hex 0x20).
- Control characters can be typed with Ctrl + the corresponding letter (Ctrl+A = SOH = 0x01, Ctrl+Z = SUB = 0x1A).
- CR+LF (0x0D 0x0A) is the Windows line ending; LF (0x0A) alone is Unix; CR (0x0D) alone is classic Mac.
- Printable characters span exactly 95 positions: 32 (space) through 126 (~).

## References

- [man ascii](https://man7.org/linux/man-pages/man7/ascii.7.html) -- ASCII character set man page
- [RFC 20 -- ASCII Format for Network Interchange](https://www.rfc-editor.org/rfc/rfc20) -- original 1969 ASCII specification
- [ANSI X3.4-1986 (via Wikipedia)](https://en.wikipedia.org/wiki/ASCII) -- ANSI/ISO standard history and tables
- [man charsets](https://man7.org/linux/man-pages/man7/charsets.7.html) -- overview of character sets on Linux
- [Unicode ASCII Block (U+0000..U+007F)](https://www.unicode.org/charts/PDF/U0000.pdf) -- ASCII as Unicode code chart
- [C0 and C1 Control Codes](https://www.unicode.org/charts/PDF/U0080.pdf) -- control character reference
- [ASCII Table and Description](https://www.asciitable.com/) -- quick lookup with decimal, hex, octal, and character
