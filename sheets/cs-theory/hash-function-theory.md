# Hash Function Theory (Cryptographic Hashing, Universal Hashing, and Hash-Based Data Structures)

A practitioner's reference for hash functions -- from cryptographic primitives and security properties to hash tables, Merkle trees, and password hashing.

## Cryptographic Hash Properties

```
H : {0,1}* -> {0,1}^n     (maps arbitrary-length input to fixed-length output)

Three security properties (in increasing strength):

1. Preimage Resistance (one-wayness)
   Given y, hard to find x such that H(x) = y.

2. Second Preimage Resistance (weak collision resistance)
   Given x, hard to find x' != x such that H(x') = H(x).

3. Collision Resistance (strong collision resistance)
   Hard to find any x, x' with x != x' such that H(x) = H(x').
```

Collision resistance implies second preimage resistance (for most practical definitions). Neither preimage resistance nor collision resistance implies the other in general.

## Birthday Paradox and Birthday Attack

```
Given n possible outputs (|{0,1}^n| = 2^n):

  Probability of collision after q queries:
    P(collision) ~ 1 - e^(-q^2 / (2 * 2^n))

  50% collision probability at:
    q ~ 1.177 * sqrt(2^n) = 1.177 * 2^(n/2)

Security implications:
  Hash output   Brute-force preimage   Birthday collision
  128-bit       2^128 operations       2^64 operations
  256-bit       2^256 operations       2^128 operations
  512-bit       2^512 operations       2^256 operations
```

Rule of thumb: an n-bit hash provides at most n/2 bits of collision security.

## Merkle-Damgard Construction

```
Message:  M = m1 || m2 || ... || mk || pad(|M|)

         IV
          |
          v
  m1 -> [ f ] -> h1
  m2 -> [ f ] -> h2
        ...
  mk -> [ f ] -> hk
 pad -> [ f ] -> H(M)     (final output)

f : {0,1}^n x {0,1}^b -> {0,1}^n   (compression function)
IV : fixed initial value
pad : Merkle-Damgard strengthening (append message length)
```

**Theorem**: If the compression function f is collision-resistant, then the iterated Merkle-Damgard hash H is collision-resistant.

**Vulnerability**: Length extension attack -- given H(M) and |M|, can compute H(M || pad || M') without knowing M. Affects: MD5, SHA-1, SHA-256, SHA-512. Does NOT affect: SHA-3, HMAC, SHA-256d (double hash).

## Sponge Construction (Keccak / SHA-3)

```
State: r + c bits (r = rate, c = capacity)

Phase 1: ABSORBING
  Initialize state to 0
  For each r-bit block mi:
    state[0..r-1] ^= mi
    state = f(state)          (permutation, e.g., Keccak-f[1600])

Phase 2: SQUEEZING
  Output state[0..r-1]
  If more output needed:
    state = f(state)
    Output state[0..r-1]
    Repeat

Security level: min(2^(c/2), 2^(output_len/2))

SHA-3-256: r = 1088, c = 512  ->  128-bit collision security
```

No length extension vulnerability. No need for HMAC wrapper (though KMAC exists).

## Compression Functions

### Davies-Meyer

```
h_i = E_{m_i}(h_{i-1}) XOR h_{i-1}

E     = block cipher
m_i   = message block (used as key)
h_{i-1} = previous chaining value (used as plaintext)

Used by: MD5, SHA-1, SHA-2 family
```

### Miyaguchi-Preneel

```
h_i = E_{m_i}(h_{i-1}) XOR h_{i-1} XOR m_i

Adds feedforward of the message block.
Used by: Whirlpool
```

Both are provably secure in the ideal cipher model (Black-Rogaway-Shrimpton: 12 secure PGV constructions out of 64 possible single-block-cipher schemes).

## Hash Function Families

| Function | Output (bits) | Block Size | Construction | Status |
|----------|--------------|------------|--------------|--------|
| MD5 | 128 | 512 | Merkle-Damgard, Davies-Meyer | Broken (collisions 2004, Wang et al.) |
| SHA-1 | 160 | 512 | Merkle-Damgard, Davies-Meyer | Broken (SHAttered 2017, chosen-prefix 2020) |
| SHA-256 | 256 | 512 | Merkle-Damgard, Davies-Meyer | Secure (standard) |
| SHA-512 | 512 | 1024 | Merkle-Damgard, Davies-Meyer | Secure (standard) |
| SHA-3-256 | 256 | 1088 (rate) | Sponge (Keccak-f[1600]) | Secure (NIST standard 2015) |
| SHA-3-512 | 512 | 576 (rate) | Sponge (Keccak-f[1600]) | Secure |
| BLAKE2b | 1-512 | 128 bytes | Modified HAIFA | Secure, faster than SHA-3 |
| BLAKE3 | 256 (extendable) | 64 bytes | Merkle tree of BLAKE2 | Secure, parallelizable |

## HMAC Construction

```
HMAC-H(K, M) = H((K' XOR opad) || H((K' XOR ipad) || M))

K' = H(K) if |K| > block_size, else K padded with zeros
ipad = 0x36 repeated to block_size
opad = 0x5C repeated to block_size
```

**Security**: HMAC is a PRF if the underlying compression function is a PRF (Bellare 2006). Does not require collision resistance of H -- HMAC-MD5 is still considered a secure MAC even though MD5 collisions are trivial.

## Random Oracle Model

```
Ideal hash function modeled as:
  - Truly random function chosen uniformly from all functions {0,1}* -> {0,1}^n
  - Accessible only via oracle queries
  - Deterministic: same input always gives same output
  - Any new input gives a uniformly random, independent output

Used for: security proofs of RSA-OAEP, PSS, Fiat-Shamir transform
Caveat: ROM proofs do not imply security in the standard model
         (Canetti-Goldreich-Halevi separation 1998)
```

## Universal Hashing (Carter-Wegman)

```
Family H = {h : U -> {0,...,m-1}} is universal if:
  For all x != y in U:
    Pr[h(x) = h(y)] <= 1/m     (over random choice of h from H)

k-wise independent: any k distinct keys hash independently.

Classic construction (integers mod prime):
  h_{a,b}(x) = ((ax + b) mod p) mod m
  where p is prime >= |U|, a in {1,...,p-1}, b in {0,...,p-1}

Collision probability: <= 1/m + 1/p  (essentially 1/m for large p)
```

Expected collisions in a bucket: n/m where n = number of keys, m = number of buckets.

## Hash Tables

### Chaining (Separate Chaining)

```
Each bucket stores a linked list of entries.
Expected chain length = n/m (load factor alpha).
Expected lookup time: O(1 + alpha) with universal hashing.
Worst case: O(n) if all keys collide.
```

### Open Addressing

```
Linear probing:   h(k,i) = (h(k) + i) mod m
Quadratic probing: h(k,i) = (h(k) + c1*i + c2*i^2) mod m
Double hashing:   h(k,i) = (h1(k) + i * h2(k)) mod m

Expected probes (uniform hashing assumption):
  Unsuccessful search: 1 / (1 - alpha)
  Successful search:   (1/alpha) * ln(1/(1 - alpha))

Load factor must stay < 1. Typically resize at alpha ~ 0.7.
```

### Cuckoo Hashing

```
Two tables T1, T2 with independent hash functions h1, h2.
Insert x: place in T1[h1(x)]; if occupied, evict y, place y in T2[h2(y)]; repeat.

Lookup: O(1) worst case (check two locations).
Insert: O(1) amortized if load factor < 0.5 per table.
Failure: cycle detection -> rehash with new functions.
```

### Perfect Hashing (FKS)

```
Two-level scheme (Fredman-Komlos-Szemeredi 1984):
  Level 1: universal hash into m = n buckets
  Level 2: for bucket of size s_i, use table of size s_i^2

Total space: O(n) expected.
Lookup: O(1) worst case.
Construction: O(n) expected time.
```

## Consistent Hashing

```
Nodes and keys mapped to positions on a ring [0, 2^k).

  key k -> hash(k) maps to ring position
  node n -> hash(n) maps to ring position
  k is assigned to the next node clockwise on the ring

Adding/removing a node: only O(K/N) keys need to remap
  (K = total keys, N = total nodes)

Virtual nodes: each physical node gets v positions on the ring
  -> more uniform distribution, O(K/N) keys per node
```

Used by: Dynamo, Cassandra, Memcached, CDNs.

## Merkle Trees

```
        H(H01 || H23)          <- root hash
       /              \
    H(H0||H1)      H(H2||H3)
    /     \          /     \
  H(D0)  H(D1)   H(D2)  H(D3)   <- leaf hashes
   D0     D1      D2      D3     <- data blocks

Proof of inclusion for D1:
  Provide: H(D0), H(H2||H3)
  Verifier computes: H(D1), H(H(D0)||H(D1)), H(H01||H23)
  Compare against known root hash.

Proof size: O(log n) hashes for n leaves.
```

Used by: Git, Bitcoin, Certificate Transparency, IPFS, ZFS.

## Password Hashing

| Function | Year | Design | Key Features |
|----------|------|--------|-------------|
| bcrypt | 1999 | Blowfish-based | Cost factor, 128-bit salt, 184-bit output |
| scrypt | 2009 | Memory-hard | CPU + memory cost, sequential memory-hard |
| Argon2 | 2015 | Memory-hard (PHC winner) | Argon2d (GPU-resistant), Argon2i (side-channel resistant), Argon2id (hybrid) |

```
Argon2id parameters (OWASP recommendation):
  Memory:      64 MB (m = 65536)
  Iterations:  3   (t = 3)
  Parallelism: 4   (p = 4)
  Salt:        16 bytes (random)
  Output:      32 bytes

Goal: each hash evaluation costs ~100ms on target hardware.
```

Password hashing is deliberately slow to resist offline brute-force attacks. General-purpose hashes (SHA-256, etc.) are unsuitable because they are fast.

## Key Figures

| Name | Contribution |
|------|-------------|
| Ralph Merkle | Merkle trees, Merkle-Damgard construction (1979), Merkle puzzles |
| Ivan Damgard | Co-inventor of Merkle-Damgard construction (1989), commitment schemes |
| Ronald Rivest | MD4, MD5 (1991-1992), co-inventor of RSA |
| Guido Bertoni, Joan Daemen, Michael Peeters, Gilles Van Assche | Keccak / SHA-3 team (2008-2015) |
| Larry Carter, Mark Wegman | Universal hashing, Carter-Wegman MACs (1979) |
| Mihir Bellare, Ran Canetti, Hugo Krawczyk | HMAC construction and security proof (1996) |
| NIST | SHA-1 (1995), SHA-2 (2001), SHA-3 competition and standardization (2007-2015) |
| Niels Provos, David Mazieres | bcrypt (1999) |
| Colin Percival | scrypt (2009) |
| Alex Biryukov, Daniel Dinu, Dmitry Khovratovich | Argon2 (2015, PHC winner) |

## Tips

- Never use MD5 or SHA-1 for security-critical applications. They are broken for collision resistance.
- For new projects, prefer SHA-256, SHA-3, or BLAKE2/BLAKE3 for general hashing.
- HMAC does not require collision resistance -- HMAC-SHA-256 remains secure even if SHA-256 collisions are found.
- For password storage, always use bcrypt, scrypt, or Argon2id. Never bare SHA-256.
- Hash table load factor is the single most important tuning parameter. Keep it below 0.75 for open addressing.
- Consistent hashing with virtual nodes gives near-optimal load balance in distributed systems.
- Merkle proofs are O(log n) -- ideal for verifying membership in large datasets without downloading everything.

## See Also

- Cryptography Fundamentals
- Computational Complexity
- Data Structures
- Distributed Systems
- Information Theory

## References

- Menezes, A., van Oorschot, P., Vanstone, S. *Handbook of Applied Cryptography*, CRC Press (1996), Chapter 9
- Katz, J. & Lindell, Y. *Introduction to Modern Cryptography*, 3rd ed., CRC Press (2020)
- Merkle, R. "A certified digital signature." CRYPTO '89, LNCS 435 (1989)
- Damgard, I. "A design principle for hash functions." CRYPTO '89, LNCS 435 (1989)
- Bellare, M., Canetti, R., Krawczyk, H. "Keying hash functions for message authentication." CRYPTO '96 (1996)
- Bertoni, G., Daemen, J., Peeters, M., Van Assche, G. "Keccak reference." NIST SHA-3 submission (2011)
- Carter, L. & Wegman, M. "Universal classes of hash functions." JCSS 18(2):143-154 (1979)
- Fredman, M., Komlos, J., Szemeredi, E. "Storing a sparse table with O(1) worst case access time." JACM 31(3):538-544 (1984)
- Cormen, T., Leiserson, C., Rivest, R., Stein, C. *Introduction to Algorithms*, 4th ed., MIT Press (2022), Chapters 11-12
- NIST FIPS 180-4: Secure Hash Standard (SHA-1, SHA-256, SHA-512)
- NIST FIPS 202: SHA-3 Standard (2015)
- Percival, C. "Stronger key derivation via sequential memory-hard functions." BSDCan (2009)
- Biryukov, A., Dinu, D., Khovratovich, D. "Argon2: the memory-hard function for password hashing." RFC 9106 (2021)
