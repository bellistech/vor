# Cryptographic Protocols — Theory and Mathematics Deep Dive

> *Modern cryptography is applied abstract algebra. AES operates in $GF(2^8)$, elliptic curves form abelian groups over finite fields, and hash functions are compression functions iterated through Merkle-Damgard or sponge constructions. Understanding the mathematics is the only way to understand the security guarantees.*

---

## 1. AES Internals — The Rijndael Cipher

### State Matrix and Round Structure

AES operates on a 4x4 matrix of bytes called the state. For AES-128, the cipher applies 10 rounds, each consisting of four transformations: SubBytes, ShiftRows, MixColumns, and AddRoundKey.

```
Input: 128-bit plaintext block → arranged as 4×4 byte matrix (column-major)

State matrix:
┌────┬────┬────┬────┐
│ s00 │ s01 │ s02 │ s03 │
│ s10 │ s11 │ s12 │ s13 │
│ s20 │ s21 │ s22 │ s23 │
│ s30 │ s31 │ s32 │ s33 │
└────┴────┴────┴────┘

Round structure (AES-128, 10 rounds):
  Round 0:    AddRoundKey(state, key[0])
  Rounds 1-9: SubBytes → ShiftRows → MixColumns → AddRoundKey
  Round 10:   SubBytes → ShiftRows → AddRoundKey (no MixColumns)

Key schedule: expands 128-bit key into 11 round keys (176 bytes)
Each round key is 128 bits (16 bytes), one per round + initial
```

### SubBytes — Non-Linear Substitution

```
SubBytes applies a fixed S-box to each byte independently.
The S-box is constructed from two operations in GF(2^8):

1. Multiplicative inverse in GF(2^8) with irreducible polynomial:
   p(x) = x^8 + x^4 + x^3 + x + 1  (0x11B in hex)

   For each byte b (as element of GF(2^8)):
   b' = b^{-1} mod p(x)    (with 0 mapping to 0)

2. Affine transformation over GF(2):
   b'' = A · b' + c
   Where A is a fixed 8×8 binary matrix and c = 0x63

Security purpose:
  - Multiplicative inverse provides non-linearity
  - Without SubBytes, AES would be a linear transformation (trivially breakable)
  - The S-box has no fixed points: S(x) ≠ x for all x
  - Maximum differential probability: 2^{-6}
  - Maximum linear approximation probability: 2^{-3}

Why GF(2^8)?
  GF(2^8) has exactly 256 elements → perfect for byte-level operations
  Every non-zero element has a unique multiplicative inverse
  Arithmetic is efficient (XOR for addition, lookup tables for multiplication)
```

### ShiftRows — Diffusion (Row Level)

```
ShiftRows cyclically shifts each row of the state matrix:

Row 0: no shift
Row 1: shift left by 1
Row 2: shift left by 2
Row 3: shift left by 3

Before:                     After:
┌────┬────┬────┬────┐      ┌────┬────┬────┬────┐
│ s00 │ s01 │ s02 │ s03 │      │ s00 │ s01 │ s02 │ s03 │  (no shift)
│ s10 │ s11 │ s12 │ s13 │  →   │ s11 │ s12 │ s13 │ s10 │  (shift 1)
│ s20 │ s21 │ s22 │ s23 │      │ s22 │ s23 │ s20 │ s21 │  (shift 2)
│ s30 │ s31 │ s32 │ s33 │      │ s33 │ s30 │ s31 │ s32 │  (shift 3)
└────┴────┴────┴────┘      └────┴────┴────┴────┘

Security purpose:
  - Distributes bytes across columns
  - Combined with MixColumns, ensures that after 2 rounds,
    every output byte depends on every input byte (full diffusion)
  - Without ShiftRows, each column would be encrypted independently
```

### MixColumns — Diffusion (Column Level)

```
MixColumns treats each column as a polynomial over GF(2^8)
and multiplies it by a fixed polynomial:

c(x) = 3x^3 + x^2 + x + 2

In matrix form (each column independently):
┌────┐     ┌───────────────┐   ┌────┐
│ r0 │     │ 2  3  1  1    │   │ s0 │
│ r1 │  =  │ 1  2  3  1    │ · │ s1 │    (in GF(2^8))
│ r2 │     │ 1  1  2  3    │   │ s2 │
│ r3 │     │ 3  1  1  2    │   │ s3 │
└────┘     └───────────────┘   └────┘

Multiplication in GF(2^8):
  ×1: identity
  ×2: left shift + conditional XOR with 0x1B (if MSB was 1)
  ×3: (×2) XOR (×1)

Security purpose:
  - Each output byte depends on all 4 input bytes of the column
  - Combined with ShiftRows, achieves full diffusion in 2 rounds
  - The polynomial c(x) is chosen to be invertible in GF(2^8)[x]/(x^4+1)
    (required for decryption)

The MixColumns matrix is an MDS (Maximum Distance Separable) code:
  Branch number = 5 (maximum possible for 4×4 matrix)
  This means: changing 1 input byte changes at least 5 bytes
  (1 in the same column, spread across 4 columns by ShiftRows)
```

### AddRoundKey — Key Mixing

```
AddRoundKey XORs the state matrix with the round key:
  state = state ⊕ round_key[i]

This is the only step that introduces key material.
Without it, the cipher would be a fixed permutation (no key dependency).

The key schedule derives round keys via:
  1. Rotate last word of previous round key
  2. Apply SubBytes to each byte
  3. XOR with round constant (Rcon): powers of 2 in GF(2^8)
     Rcon[1] = 0x01, Rcon[2] = 0x02, Rcon[3] = 0x04, ...
  4. XOR with corresponding word from previous round key

Key schedule security:
  - Each round key is non-linearly related to the master key
  - Related-key attacks are theoretical (require attacker to control
    the relationship between multiple keys — not practical for most uses)
```

---

## 2. GCM Mode — GHASH Polynomial Multiplication

### GCM Construction

```
GCM = CTR mode encryption + GHASH polynomial authentication

Encryption (CTR mode):
  Counter block: Nonce (96 bits) || Counter (32 bits)
  J0 = Nonce || 0x00000001  (initial counter)
  For each plaintext block P_i:
    C_i = P_i ⊕ AES_K(J0 + i)

Authentication (GHASH):
  H = AES_K(0^{128})  (hash key, encryption of zero block)

  GHASH processes AAD (additional authenticated data) and ciphertext:
  X_0 = 0^{128}
  For each AAD block A_i:
    X_i = (X_{i-1} ⊕ A_i) · H    (multiplication in GF(2^{128}))
  For each ciphertext block C_j:
    X_j = (X_{j-1} ⊕ C_j) · H
  Final: include length block (len(A) || len(C))

  Tag T = AES_K(J0) ⊕ GHASH(H, A, C)
```

### GHASH Polynomial Multiplication in GF(2^128)

```
GHASH multiplies two 128-bit values in the field GF(2^{128})
with irreducible polynomial:

  f(x) = x^{128} + x^7 + x^2 + x + 1

Multiplication algorithm (schoolbook):
  Given a, b ∈ GF(2^{128}):
  result = 0
  for i = 0 to 127:
    if bit i of b is set:
      result = result ⊕ a
    a = a << 1
    if MSB of a was 1:
      a = a ⊕ f   (reduction modulo f(x))

Hardware acceleration:
  Intel CLMUL instruction (PCLMULQDQ) performs carry-less multiplication
  Combined with AES-NI, GCM achieves near-theoretical throughput
  Without CLMUL: GCM is significantly slower (use ChaCha20-Poly1305)

Nonce reuse catastrophe in GCM:
  If nonce is reused with the same key:
  1. CTR keystream repeats → C1 ⊕ C2 = P1 ⊕ P2 (plaintext XOR leak)
  2. GHASH key H can be recovered from two tag equations:
     T1 = GHASH(H, A1, C1) ⊕ AES_K(J0)
     T2 = GHASH(H, A2, C2) ⊕ AES_K(J0)
     T1 ⊕ T2 = GHASH(H, A1, C1) ⊕ GHASH(H, A2, C2)
     This is a polynomial equation in H that can be solved
  3. With H recovered, attacker can forge tags for arbitrary messages
  This is why nonce uniqueness is CRITICAL for GCM.
```

---

## 3. RSA — Key Generation and Security

### RSA Key Generation

```
1. Choose two large primes p and q (each n/2 bits for n-bit RSA key)
   p and q must be:
   - Random and independent
   - Similar size (|p| ≈ |q|)
   - Strong primes (p-1 and p+1 have large prime factors)

2. Compute n = p · q  (the modulus, n is the "key size")

3. Compute Euler's totient: φ(n) = (p-1)(q-1)
   Or Carmichael's totient: λ(n) = lcm(p-1, q-1)  (preferred)

4. Choose public exponent e:
   - Common choice: e = 65537 = 2^16 + 1
   - Must satisfy: gcd(e, λ(n)) = 1
   - e = 65537 is efficient (only two 1-bits) and universally used

5. Compute private exponent d:
   d = e^{-1} mod λ(n)
   Using extended Euclidean algorithm

Public key:  (n, e)
Private key: (n, d)  [also stores p, q for CRT optimization]

CRT optimization for private key operations:
  Instead of m = c^d mod n (expensive, n is 3072+ bits):
  m1 = c^{d mod (p-1)} mod p    (half-size exponent)
  m2 = c^{d mod (q-1)} mod q    (half-size exponent)
  m = CRT(m1, m2)               (combine using Chinese Remainder Theorem)
  Speedup: ~4x faster than naive computation
```

### RSA Security (Factoring Hardness)

```
RSA security relies on the hardness of factoring n = p · q

Best known factoring algorithms:

1. General Number Field Sieve (GNFS):
   Runtime: exp((64/9)^{1/3} · (ln n)^{1/3} · (ln ln n)^{2/3})
   Sub-exponential but super-polynomial
   Used to factor RSA-250 (829 bits) in 2020: ~2700 core-years

2. Factoring records:
   RSA-250:  829 bits,  factored 2020
   RSA-260:  862 bits,  factored 2024
   RSA-270:  ~895 bits, not yet factored
   RSA-2048: 2048 bits, estimated: ~10^24 core-years with GNFS

3. Quantum threat (Shor's algorithm):
   Runtime: O(n^3) on a quantum computer (polynomial!)
   Requires ~4n logical qubits to factor n-bit number
   RSA-2048: requires ~8192 logical qubits
   Current quantum computers: ~1000+ noisy physical qubits
   Estimated timeline for cryptographically relevant QC: 2035-2045
   (highly uncertain; plan for sooner rather than later)

Key size to security level:
  RSA-1024:  ~80-bit security  → BROKEN (factorable with current tech)
  RSA-2048:  ~112-bit security → acceptable until ~2030
  RSA-3072:  ~128-bit security → recommended minimum
  RSA-4096:  ~140-bit security → high security / long-term keys
  RSA-7680:  ~192-bit security → theoretical (too slow for practice)
```

---

## 4. Elliptic Curve Group Operations

### Elliptic Curve Arithmetic

```
An elliptic curve E over a prime field F_p is defined by:
  y^2 = x^3 + ax + b  (short Weierstrass form)
  with discriminant Δ = -16(4a^3 + 27b^2) ≠ 0

The set of rational points on E plus the point at infinity (O)
forms an abelian group under point addition.

Point Addition (P + Q, where P ≠ Q):
  Given P = (x1, y1) and Q = (x2, y2):
  slope λ = (y2 - y1) / (x2 - x1)  mod p
  x3 = λ^2 - x1 - x2  mod p
  y3 = λ(x1 - x3) - y1  mod p
  P + Q = (x3, y3)

Point Doubling (P + P = 2P):
  Given P = (x1, y1):
  slope λ = (3x1^2 + a) / (2y1)  mod p
  x3 = λ^2 - 2x1  mod p
  y3 = λ(x1 - x3) - y1  mod p
  2P = (x3, y3)

Identity element: O (point at infinity)
  P + O = O + P = P  for all P

Inverse: -P = (x, -y mod p)
  P + (-P) = O

NIST P-256 parameters:
  p = 2^{256} - 2^{224} + 2^{192} + 2^{96} - 1  (Mersenne-like prime)
  a = -3  (chosen for efficiency in point doubling)
  b = 0x5AC635D8AA3A93E7B3EBBD55769886BC651D06B0CC53B0F63BCE3C3E27D2604B
  n = order of generator G ≈ 2^{256} (number of points on the curve)
  h = 1 (cofactor)
```

### Scalar Multiplication and ECDLP

```
Scalar multiplication: Q = k · P (add P to itself k times)
This is the fundamental operation in ECC.

Efficient computation: double-and-add algorithm
  (analogous to square-and-multiply for modular exponentiation)

  k = k_{n-1} k_{n-2} ... k_1 k_0  (binary representation)
  Q = O
  for i = n-1 downto 0:
    Q = 2Q          (point doubling)
    if k_i = 1:
      Q = Q + P     (point addition)

  Runtime: O(n) doublings and O(n/2) additions for n-bit scalar

ECDLP (Elliptic Curve Discrete Logarithm Problem):
  Given P and Q = k · P, find k.

  Best classical attack: Pollard's rho algorithm
  Runtime: O(√n) ≈ O(2^{n/2}) group operations
  For P-256: O(2^{128}) operations → 128-bit security

  No sub-exponential classical algorithm is known for ECDLP
  (unlike integer factorization which has GNFS)
  This is why ECC achieves equivalent security with much smaller keys.

Quantum attack: Shor's algorithm modified for ECDLP
  Runtime: O(n^3) on a quantum computer
  For P-256: requires ~2560 logical qubits (less than RSA-2048)
  ECC is MORE vulnerable to quantum than RSA per security bit
```

### Curve25519 — Montgomery Curve

```
Curve25519 is defined over F_p where p = 2^{255} - 19:
  By^2 = x^3 + Ax^2 + x
  A = 486662, B = 1

Montgomery form advantages:
  - Differential addition: can compute x-coordinate of P+Q
    given x(P), x(Q), and x(P-Q) without y-coordinates
  - Montgomery ladder: constant-time scalar multiplication
    (no conditional branches based on key bits)
  - Natural side-channel resistance

Montgomery ladder algorithm:
  Input: scalar k (clamped), point P (x-coordinate only)
  R0 = O, R1 = P
  for each bit b of k (from MSB to LSB):
    if b = 0:
      R1 = R0 + R1  (differential addition)
      R0 = 2·R0     (doubling)
    else:
      R0 = R0 + R1  (differential addition)
      R1 = 2·R1     (doubling)
  return R0

  Both branches perform the same operations (add + double)
  Only the assignment of results to R0/R1 differs
  With conditional swap (CSWAP): fully constant-time

Key clamping (for X25519):
  k[0]  &= 248    (clear bottom 3 bits → multiple of cofactor 8)
  k[31] &= 127    (clear top bit)
  k[31] |= 64     (set second-highest bit → fixed-length scalar)
  Clamping ensures: no small-subgroup attacks, constant-time execution
```

---

## 5. Diffie-Hellman Security

### Security Proof Sketch (CDH and DDH)

```
Diffie-Hellman security relies on two related assumptions:

Computational Diffie-Hellman (CDH):
  Given g, g^a, g^b in a cyclic group G of prime order q,
  computing g^{ab} is hard.

  CDH ≤ DLP (if you can solve DLP, you can solve CDH)
  CDH ≥? DLP (whether CDH is easier than DLP is unknown)

Decisional Diffie-Hellman (DDH):
  Given g, g^a, g^b, and some value Z,
  distinguishing whether Z = g^{ab} or Z is random is hard.

  DDH ≤ CDH ≤ DLP (in terms of hardness)

DH key exchange security (informal):
  1. Eavesdropper sees: g, g^a (Alice's public), g^b (Bob's public)
  2. To compute shared secret g^{ab}, must solve CDH
  3. Best known: solve DLP for either a or b

  For multiplicative group Z*_p (classical DH):
    DLP solved by Number Field Sieve: sub-exponential
    2048-bit p → ~112-bit security
    3072-bit p → ~128-bit security

  For elliptic curve groups (ECDH):
    DLP solved by Pollard's rho: O(√n) → exponential
    256-bit curve → ~128-bit security (much more efficient)

Active attack (Man-in-the-Middle):
  DH alone does NOT provide authentication.
  MITM attacker can establish separate shared secrets with each party.
  Mitigated by: signing DH parameters (as in TLS), STS protocol,
  or using an authenticated key exchange (AKE) protocol.
```

---

## 6. Hash Function Security Properties

### Formal Security Definitions

```
A cryptographic hash function H: {0,1}* → {0,1}^n must satisfy:

1. Preimage Resistance (one-wayness):
   Given h = H(m), finding any m' such that H(m') = h
   should require O(2^n) operations.

   If broken: attacker can recover passwords from hashes,
   forge documents matching a known hash.

2. Second Preimage Resistance:
   Given m, finding m' ≠ m such that H(m') = H(m)
   should require O(2^n) operations.

   If broken: attacker can create a different document
   with the same hash as a signed document.

3. Collision Resistance:
   Finding any pair (m, m') with m ≠ m' such that H(m) = H(m')
   should require O(2^{n/2}) operations.

   Note: collision resistance provides only n/2 bits of security
   due to the birthday paradox.
   SHA-256: 128-bit collision resistance
   SHA-512: 256-bit collision resistance

Relationship (in terms of hardness):
  Collision resistance → Second preimage resistance → Preimage resistance
  (breaking collision resistance does NOT break preimage resistance)

  MD5 and SHA-1 have broken collision resistance
  but their preimage resistance is still intact.
```

### Birthday Attack Analysis

```
Birthday paradox: in a set of n elements, a collision is expected
after sampling approximately √n elements.

For a hash function with n-bit output:
  Expected number of inputs to find a collision: O(2^{n/2})

Derivation:
  After k hash evaluations, the probability of NO collision:
  P(no collision) = ∏_{i=0}^{k-1} (1 - i/2^n)

  For large 2^n:
  P(no collision) ≈ e^{-k(k-1)/(2·2^n)}

  Setting P(collision) = 0.5:
  k ≈ √(2 · 2^n · ln 2) ≈ 1.177 · 2^{n/2}

Practical implications:
  Hash Output    Collision Security    Operations to Find
  ────────────────────────────────────────────────────────
  128 bits       64 bits              ~2^{64}  (feasible!)
  160 bits       80 bits              ~2^{80}  (expensive but done for SHA-1)
  256 bits       128 bits             ~2^{128} (infeasible with current tech)
  384 bits       192 bits             ~2^{192}
  512 bits       256 bits             ~2^{256}

This is why:
  - MD5 (128-bit) collision resistance was broken
  - SHA-1 (160-bit) collision was demonstrated in 2017 (SHAttered)
  - SHA-256 (128-bit collision resistance) is considered safe
  - Hash outputs should be at least 256 bits for collision resistance
```

### Merkle-Damgard vs Sponge Construction

```
Merkle-Damgard (SHA-1, SHA-256, SHA-512, MD5):
┌─────────────────────────────────────────────────────┐
│                                                      │
│  IV ──→ [f] ──→ [f] ──→ [f] ──→ [f] ──→ Hash      │
│          ↑       ↑       ↑       ↑                   │
│         M_1     M_2     M_3     M_4 (padded)        │
│                                                      │
│  f = compression function                            │
│  Each block: state = f(state, message_block)        │
│  Final: output = finalize(state)                     │
│                                                      │
│  Vulnerability: length extension attack              │
│  Given H(m) and |m|, can compute H(m || padding || m') │
│  Without knowing m!                                   │
│  Mitigation: HMAC construction, or use SHA-3         │
└─────────────────────────────────────────────────────┘

Sponge Construction (SHA-3/Keccak, BLAKE2):
┌─────────────────────────────────────────────────────┐
│                                                      │
│  State: [r bits (rate) | c bits (capacity)]          │
│                                                      │
│  Absorb phase:                                       │
│    For each message block M_i:                       │
│      state[0..r-1] ⊕= M_i                          │
│      state = f(state)   (Keccak-f permutation)      │
│                                                      │
│  Squeeze phase:                                      │
│    Output = state[0..r-1]                            │
│    If more output needed: state = f(state), repeat   │
│                                                      │
│  Security: determined by capacity c                  │
│  For n-bit hash: c ≥ 2n (provides n-bit security)  │
│                                                      │
│  SHA3-256: r = 1088, c = 512 → 256-bit security    │
│  SHAKE256: r = 1088, c = 512 → arbitrary output len │
│                                                      │
│  NO length extension vulnerability (capacity is     │
│  never directly output)                              │
└─────────────────────────────────────────────────────┘
```

---

## 7. Post-Quantum Lattice Problems

### Learning With Errors (LWE)

```
The Learning With Errors problem (Regev, 2005) is the foundation
for ML-KEM (Kyber) and ML-DSA (Dilithium).

LWE problem statement:
  Given: matrix A ∈ Z_q^{m×n}, vector b = As + e mod q
  Where: s ∈ Z_q^n is a secret vector
         e ∈ Z_q^m is a small error vector (from discrete Gaussian)
  Find: s

Without the error e, this is just solving a linear system (easy).
The small error makes it computationally hard.

  A · s + e = b  mod q
  ↑   ↑   ↑   ↑
  known secret small known
       (find)  noise

Security reduction:
  LWE is at least as hard as worst-case lattice problems:
  - Shortest Vector Problem (SVP)
  - Bounded Distance Decoding (BDD)
  These are believed to be hard even for quantum computers.

Parameters for ML-KEM-768 (192-bit security):
  n = 256 (polynomial degree)
  k = 3 (module rank)
  q = 3329 (modulus)
  Error distribution: centered binomial with η = 2
```

### Ring-LWE (RLWE) and Module-LWE

```
Ring-LWE: operates in polynomial ring R_q = Z_q[x]/(x^n + 1)
  Instead of matrix-vector multiplication over Z_q,
  uses polynomial multiplication in R_q.

  Advantage: much smaller keys and ciphertexts
  Risk: additional algebraic structure might help attackers

Module-LWE (used by ML-KEM/Kyber):
  Intermediate between LWE and Ring-LWE.
  Uses module over R_q: vectors of polynomials.

  Module rank k controls security/efficiency tradeoff:
    k = 2: ML-KEM-512  (128-bit security)
    k = 3: ML-KEM-768  (192-bit security)  ← recommended
    k = 4: ML-KEM-1024 (256-bit security)

  Key sizes vs classical algorithms:
  Algorithm        Public Key    Ciphertext    Shared Secret
  ─────────────────────────────────────────────────────────
  X25519           32 bytes      32 bytes      32 bytes
  RSA-3072         384 bytes     384 bytes     32 bytes
  ML-KEM-768       1184 bytes    1088 bytes    32 bytes
  ML-KEM-1024      1568 bytes    1568 bytes    32 bytes

  The 10-50x larger key/ciphertext sizes are the practical cost
  of post-quantum security.
```

### Best Known Lattice Attacks

```
Lattice reduction algorithms:

1. LLL algorithm (Lenstra-Lenstra-Lovász, 1982):
   - Polynomial time: O(n^5 · d · log^3 B)
   - Finds short vectors, but not shortest
   - Approximation factor: 2^{O(n)}
   - Useful for breaking weak parameters, not secure ML-KEM

2. BKZ (Block Korf-Zolotarev):
   - BKZ-β uses an SVP oracle in dimension β
   - Approximation factor: β^{n/(2β)} (improves with larger β)
   - Runtime dominated by SVP oracle calls
   - BKZ-2.0: improved enumeration, practical for β ≤ ~60

3. Sieving algorithms (for SVP oracle in BKZ):
   - Time: 2^{0.292n + o(n)} (best known classical)
   - Space: 2^{0.2075n + o(n)}
   - Quantum speedup (Grover): 2^{0.265n + o(n)} (modest improvement)

Security estimation for ML-KEM-768:
  Core-SVP hardness: ~2^{178} operations (classical)
  Core-SVP hardness: ~2^{161} operations (quantum, Grover-assisted sieving)
  Concrete security: well above 128-bit target

Key observation: quantum computers provide only a MODEST speedup
for lattice problems (Grover's √ speedup on sieving), unlike the
EXPONENTIAL speedup Shor provides against RSA/ECC.
This is why lattice-based crypto is considered post-quantum secure.
```

---

## 8. Hybrid Key Exchange

### Construction and Security Proof

```
Hybrid key exchange combines a classical and post-quantum KEM:

Protocol (simplified, as in TLS 1.3 hybrid):
  1. Client generates X25519 keypair: (a, g^a)
     Client generates ML-KEM-768 keypair: (sk_pq, pk_pq)
     Client sends: (g^a, pk_pq)

  2. Server generates X25519 keypair: (b, g^b)
     Server encapsulates with ML-KEM: (ct_pq, ss_pq) = Encaps(pk_pq)
     Server sends: (g^b, ct_pq)

  3. Both compute:
     ss_classical = X25519(a, g^b) = X25519(b, g^a)
     ss_pq = Decaps(sk_pq, ct_pq)
     shared_secret = HKDF(ss_classical || ss_pq)

Security argument:
  The shared secret is secure if EITHER component is secure:
  - If ML-KEM is broken: X25519 still protects (ECDLP is hard)
  - If X25519 is broken (quantum): ML-KEM still protects (MLWE is hard)
  - Both must be broken simultaneously to compromise the session

  Formally: security of hybrid ≥ max(security of components)
  (under standard assumptions about HKDF as a random oracle)

Cost of hybrid:
  Additional bandwidth: ~1200 bytes per ClientHello (ML-KEM public key)
  Additional computation: ~0.1ms (ML-KEM operations are fast)
  Latency: unchanged (still 1-RTT TLS 1.3 handshake)
  The overhead is negligible for most applications.
```

---

## 9. Forward Secrecy

### Definition and Mechanisms

```
Forward secrecy (FS) guarantees that compromise of long-term keys
does not reveal past session keys.

Without FS (RSA key transport in TLS 1.2):
  Client encrypts premaster_secret with server's RSA public key.
  Server decrypts with RSA private key.

  If RSA private key is later compromised:
  Attacker decrypts recorded traffic → recovers premaster_secret
  → derives all session keys → decrypts all past sessions.

  Timeline of attack:
    t=0: TLS session established (attacker records ciphertext)
    t=1y: Server's RSA key compromised (stolen, leaked, court order)
    t=1y: Attacker decrypts all sessions from t=0 to t=1y

With FS (ECDHE in TLS 1.3):
  Client and server generate ephemeral ECDH keys per session.
  Shared secret = ECDH(ephemeral_client, ephemeral_server).
  Ephemeral keys are destroyed after key derivation.

  If long-term key (certificate) is later compromised:
  Attacker can impersonate the server going FORWARD
  but CANNOT decrypt past sessions (ephemeral keys are gone).

  The long-term key is used only for AUTHENTICATION (signature),
  not for KEY EXCHANGE.

TLS 1.3 mandates forward secrecy:
  - Only ECDHE and DHE key exchanges allowed
  - RSA key transport removed entirely
  - Static DH removed
  - PSK mode still supports FS via PSK-DHE mode
```

---

## 10. Algorithm Agility

### The Design Principle

```
Algorithm agility: the ability to negotiate, select, and transition
between cryptographic algorithms without protocol changes.

Why it matters:
  1. Algorithms break (MD5, SHA-1, DES, RC4 — all once "secure")
  2. Performance varies by platform (AES-NI vs no AES-NI)
  3. Compliance requirements change (NIST deprecates algorithms)
  4. Post-quantum transition requires adding new algorithm families

TLS as an example of algorithm agility:
  - Client sends list of supported cipher suites
  - Server selects the best mutually supported suite
  - New algorithms added via IANA cipher suite registry
  - Old algorithms deprecated without protocol version change
  - TLS 1.3 removed all insecure options at the protocol level

Algorithm agility risks:
  1. Downgrade attacks: attacker forces weaker algorithm
     Mitigation: TLS 1.3 transcript hash prevents tampering
  2. Complexity: more code paths = more bugs
     Mitigation: remove deprecated algorithms aggressively
  3. Negotiation overhead: longer handshakes with many options
     Mitigation: prefer smaller, curated cipher suite lists

Best practices:
  - Support minimum 2 algorithms per function (primary + backup)
  - Have a documented algorithm transition plan
  - Monitor NIST/IETF deprecation announcements
  - Test cipher suite negotiation regularly
  - Plan for post-quantum: support hybrid key exchange now
```

---

## References

- [NIST FIPS 197 — AES Specification](https://csrc.nist.gov/publications/detail/fips/197/final)
- [Daemen, J., Rijmen, V. — "The Design of Rijndael" (Springer, 2002)](https://www.springer.com/gp/book/9783540425809)
- [NIST SP 800-38D — Recommendation for GCM Mode](https://csrc.nist.gov/publications/detail/sp/800-38d/final)
- [Joux, A. — "Authentication Failures in NIST version of GCM" (2006)](https://csrc.nist.gov/csrc/media/projects/block-cipher-techniques/documents/bcm/joux_comments.pdf)
- [Boneh, D., Shoup, V. — "A Graduate Course in Applied Cryptography"](https://toc.cryptobook.us/)
- [Regev, O. — "On Lattices, Learning with Errors, Random Linear Codes, and Cryptography" (2005)](https://dl.acm.org/doi/10.1145/1060590.1060603)
- [Bernstein, D.J. — "Curve25519: New Diffie-Hellman Speed Records"](https://cr.yp.to/ecdh/curve25519-20060209.pdf)
- [Hankerson, D., Menezes, A., Vanstone, S. — "Guide to Elliptic Curve Cryptography" (Springer, 2004)](https://www.springer.com/gp/book/9780387952734)
- [NIST FIPS 203 — ML-KEM Standard](https://csrc.nist.gov/publications/detail/fips/203/final)
- [NIST SP 800-57 Part 1 Rev 5 — Key Management Recommendations](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final)
- [RFC 8446 — TLS 1.3](https://www.rfc-editor.org/rfc/rfc8446)
- [Stevens, M. et al. — "The First Collision for Full SHA-1" (SHAttered, 2017)](https://shattered.io/)
