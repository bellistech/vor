# Information Theory (Entropy, Coding, and Channel Capacity)

A practitioner's reference for the mathematical foundations of information -- from Shannon entropy and source coding to channel capacity and error correction.

## Entropy and Information Measures

### Shannon Entropy

```
H(X) = -sum_{x} P(x) log_2 P(x)

Properties:
  - H(X) >= 0
  - H(X) = 0  iff X is deterministic
  - H(X) <= log_2 |X|  (maximum for uniform distribution)
  - Units: bits (log base 2), nats (ln), hartleys (log base 10)

Example — fair coin:
  H(X) = -(0.5 * log2(0.5) + 0.5 * log2(0.5)) = 1 bit

Example — biased coin (P(H)=0.9):
  H(X) = -(0.9 * log2(0.9) + 0.1 * log2(0.1)) = 0.469 bits
```

### Joint Entropy

```
H(X, Y) = -sum_{x,y} P(x, y) log_2 P(x, y)

H(X, Y) <= H(X) + H(Y)       (equality iff X, Y independent)
H(X, Y) >= max(H(X), H(Y))
```

### Conditional Entropy

```
H(Y|X) = -sum_{x,y} P(x, y) log_2 P(y|x)

Chain rule:      H(X, Y) = H(X) + H(Y|X)
Conditioning:    H(Y|X) <= H(Y)  (conditioning reduces entropy)
Independence:    H(Y|X) = H(Y)   iff X, Y independent
```

### Mutual Information

```
I(X; Y) = H(X) + H(Y) - H(X, Y)
         = H(X) - H(X|Y)
         = H(Y) - H(Y|X)

Properties:
  - I(X; Y) >= 0  (non-negative)
  - I(X; Y) = I(Y; X)  (symmetric)
  - I(X; X) = H(X)  (self-information = entropy)

Venn diagram:

     +-----------+-----------+
     |           |           |
     |  H(X|Y)  | I(X;Y)   |  H(Y|X)  |
     |           |           |
     +-----------+-----------+
     |<--- H(X) --->|
                 |<--- H(Y) --->|
     |<------- H(X,Y) -------->|
```

### KL Divergence (Relative Entropy)

```
D_KL(P || Q) = sum_x P(x) log_2 (P(x) / Q(x))

Properties:
  - D_KL(P || Q) >= 0  (Gibbs' inequality)
  - D_KL(P || Q) = 0  iff P = Q
  - NOT symmetric:  D_KL(P || Q) != D_KL(Q || P) in general
  - NOT a metric (violates triangle inequality)

Relationship to mutual information:
  I(X; Y) = D_KL(P(X,Y) || P(X) P(Y))
```

## Shannon's Theorems

### Source Coding Theorem (Shannon's First Theorem)

```
A source with entropy H can be encoded with an average of at least
H bits per symbol. No lossless code can do better.

Specifically, for a stationary ergodic source X:
  - There exists a code with average length L satisfying:
        H(X) <= L < H(X) + 1
  - For block codes of length n:
        H(X) <= L_n / n < H(X) + 1/n

Consequence: Entropy is the fundamental limit of lossless compression.
```

### Noisy Channel Coding Theorem (Shannon's Second Theorem)

```
For a discrete memoryless channel with capacity C:

  C = max_{P(x)} I(X; Y)

  - If R < C (rate below capacity):
      There exist codes that achieve arbitrarily low error probability.
  - If R > C (rate above capacity):
      Every code has error probability bounded away from zero.

Binary Symmetric Channel (BSC) with crossover probability p:
  C_BSC = 1 - H(p) = 1 + p log_2(p) + (1-p) log_2(1-p)

Binary Erasure Channel (BEC) with erasure probability e:
  C_BEC = 1 - e

Additive White Gaussian Noise (AWGN):
  C_AWGN = (1/2) log_2(1 + SNR)  bits per channel use
```

## Data Compression

### Huffman Coding

```
Algorithm:
  1. Create a leaf node for each symbol with its probability
  2. While more than one node remains:
     a. Remove two nodes with lowest probability
     b. Create parent node with combined probability
     c. Assign 0 to one branch, 1 to the other
  3. Read code from root to each leaf

Example — symbols {A:0.4, B:0.3, C:0.2, D:0.1}:

          (1.0)
         /     \
       (0.6)    A:0.4
       /    \
    (0.3)   B:0.3
    /    \
  D:0.1  C:0.2

  A -> 1      (1 bit)
  B -> 01     (2 bits)
  C -> 001    (3 bits)
  D -> 000    (3 bits)

  Average length = 0.4(1) + 0.3(2) + 0.2(3) + 0.1(3) = 1.9 bits
  Entropy H     = 1.846 bits
  Efficiency     = H / L = 97.2%

Properties:
  - Optimal prefix code for known symbol probabilities
  - H(X) <= L_Huffman < H(X) + 1
  - Prefix-free: no codeword is a prefix of another
```

### Arithmetic Coding

```
Encodes an entire message as a single number in [0, 1).

Algorithm:
  1. Start with interval [0, 1)
  2. For each symbol, subdivide current interval proportionally
  3. Output any number within the final interval

Example — message "AB" with P(A)=0.6, P(B)=0.4:
  Start: [0.0, 1.0)
  A:     [0.0, 0.6)     (first 60%)
  B:     [0.36, 0.6)    (last 40% of [0, 0.6))
  Output: 0.4 (or any value in [0.36, 0.6))

Advantages over Huffman:
  - Can approach entropy more closely (no 1-bit minimum per symbol)
  - Better for highly skewed distributions
  - Handles adaptive/changing probabilities naturally
```

## Error-Correcting Codes

### Hamming Codes

```
Parameters:  (2^r - 1, 2^r - 1 - r, 3)
  - n = 2^r - 1  codeword length
  - k = 2^r - 1 - r  data bits
  - d_min = 3  minimum distance

Hamming(7,4) — most common:
  7 total bits = 4 data + 3 parity
  Detects up to 2-bit errors
  Corrects 1-bit errors

Parity check positions: 1, 2, 4 (powers of 2)
Data positions: 3, 5, 6, 7

Parity check matrix H for Hamming(7,4):
  H = | 1 0 1 0 1 0 1 |
      | 0 1 1 0 0 1 1 |
      | 0 0 0 1 1 1 1 |

Syndrome = H * r^T (r = received word)
  Syndrome = 0  ->  no error
  Syndrome != 0 ->  syndrome gives error position

Hamming bound (sphere-packing bound):
  2^k * sum_{i=0}^{t} C(n,i) <= 2^n
  where t = floor((d_min - 1) / 2)
```

### Key Code Families

| Code | Rate | Distance | Use Case |
|---|---|---|---|
| Hamming(7,4) | 4/7 | 3 | Memory ECC (SECDED) |
| Reed-Solomon | k/n | n-k+1 | CDs, QR codes, deep space |
| Convolutional | varies | varies | 3G/4G, satellite |
| LDPC | near capacity | varies | 5G, Wi-Fi 6, DVB-S2 |
| Turbo | near capacity | varies | 3G, deep space |
| Polar | near capacity | varies | 5G control channel |

## Key Figures

| Name | Contribution | Year |
|---|---|---|
| Claude Shannon | Founded information theory, entropy, channel capacity | 1948 |
| Richard Hamming | Hamming codes, Hamming distance, error correction | 1950 |
| David Huffman | Optimal prefix-free codes (Huffman coding) | 1952 |
| Solomon Kullback | KL divergence (with Leibler) | 1951 |
| Abraham Lempel / Jacob Ziv | LZ77, LZ78 dictionary compression | 1977-78 |
| Robert Gallager | LDPC codes | 1962 |

## Tips

- Entropy is maximized by a uniform distribution: use this as a sanity check.
- KL divergence is not symmetric; in practice D_KL(P||Q) penalizes Q=0 where P>0 infinitely.
- Huffman coding is optimal per-symbol but arithmetic coding wins for streaming and skewed distributions.
- The noisy channel theorem is an existence proof; it does not give a construction. Turbo codes and LDPC codes approach the Shannon limit in practice.
- For quick entropy estimates: a fair coin is 1 bit, a fair die is ~2.585 bits, English text is ~1.0-1.5 bits/char.
- Channel capacity is a property of the channel, not the source. Optimize the input distribution to achieve it.

## See Also

- coding-theory
- compression
- probability
- cryptography
- signal-processing

## References

- Shannon, C. E. "A Mathematical Theory of Communication" (1948), Bell System Technical Journal
- Cover, T. M. & Thomas, J. A. "Elements of Information Theory" (2nd ed., Wiley, 2006)
- MacKay, D. J. C. "Information Theory, Inference, and Learning Algorithms" (Cambridge, 2003) -- free online
- Hamming, R. W. "Error Detecting and Error Correcting Codes" (1950), Bell System Technical Journal
- Huffman, D. A. "A Method for the Construction of Minimum-Redundancy Codes" (1952), Proc. IRE
