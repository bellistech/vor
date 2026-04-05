# Zero-Knowledge Proofs (Interactive Proofs, SNARKs, STARKs, and Cryptographic Privacy)

A compact reference for zero-knowledge proofs -- a cryptographic method by which one party (prover) can convince another party (verifier) that a statement is true without revealing any information beyond the validity of the statement itself. Foundational to modern blockchain privacy, authentication, and verifiable computation.

## Interactive Proofs

### Prover and Verifier

```
An interactive proof system for a language L is a protocol between two parties:

  Prover  (P):  unbounded computational power, wants to convince V that x in L
  Verifier (V): probabilistic polynomial-time, accepts or rejects

The protocol proceeds in rounds of message exchange:

  P  --->  V    (prover's message, commitment)
  P  <---  V    (verifier's random challenge)
  P  --->  V    (prover's response)
  ...
  V outputs: ACCEPT or REJECT
```

### Completeness and Soundness

```
An interactive proof system (P, V) for language L satisfies:

  Completeness:  For all x in L,
                   Pr[V accepts after interacting with P on input x] >= 2/3

  Soundness:     For all x not in L, for every (possibly cheating) prover P*,
                   Pr[V accepts after interacting with P* on input x] <= 1/3

The class IP = {languages with interactive proof systems}
Shamir's theorem: IP = PSPACE
```

## Zero-Knowledge Property

### The Simulator Paradigm

```
An interactive proof (P, V) for L is zero-knowledge if:

  For every PPT verifier V*, there exists a PPT simulator S such that
  for all x in L:

    View_V*[P(x) <-> V*(x)]  is computationally indistinguishable from  S(x)

  Meaning: anything V* learns from interacting with P, it could have
  generated on its own without P.

Flavors of zero-knowledge:
  Perfect ZK:      distributions are identical
  Statistical ZK:  distributions are statistically close
  Computational ZK: distributions are computationally indistinguishable
```

## ZK for Graph Isomorphism

### Classic Example

```
Statement: Prover knows an isomorphism phi: G0 -> G1

Protocol (one round):
  1. P picks random permutation pi, sends H = pi(G0) to V
  2. V picks random bit b in {0,1}, sends b to P
  3. P responds with sigma:
       if b = 0: sigma = pi          (isomorphism G0 -> H)
       if b = 1: sigma = pi . phi^-1 (isomorphism G1 -> H)
  4. V checks sigma(Gb) = H, accepts if correct

Completeness: honest prover always succeeds
Soundness:    cheating prover (no phi) can answer at most one of b=0, b=1
              => Pr[cheat] <= 1/2 per round, negligible after k rounds
Zero-knowledge: simulator picks random b', builds H = pi(Gb'), outputs transcript
                => distribution identical to real interaction
```

## ZK for Graph 3-Coloring

### GMW Protocol

```
Statement: Prover knows a valid 3-coloring c: V -> {1,2,3} of graph G

Protocol (one round):
  1. P picks random permutation sigma of {1,2,3}
     Computes c' = sigma . c  (relabeled coloring, still valid)
     Commits to c'(v) for each vertex v  (using a commitment scheme)
  2. V picks random edge (u, v) in E, sends to P
  3. P opens commitments for c'(u) and c'(v)
  4. V checks: c'(u) != c'(v) and both are valid colors

Soundness:    if coloring is invalid, at least one edge is monochromatic
              => Pr[cheat per round] <= 1 - 1/|E|
              => negligible after O(|E| log(1/eps)) rounds
Zero-knowledge: simulator picks random edge, assigns two distinct random colors,
                commits; if V queries that edge, open correctly; else rewind
NP-completeness: every NP language reduces to 3-coloring
                 => every NP language has a zero-knowledge proof
```

## Schnorr Protocol

### Discrete Log Proof of Knowledge

```
Public:  group G of prime order q, generator g, public key h = g^x
Private: prover knows witness x (discrete log of h)

Protocol:
  1. P picks random r in Z_q, sends commitment a = g^r to V
  2. V picks random challenge e in Z_q (or a smaller set), sends e to P
  3. P computes response z = r + e*x (mod q), sends z to V
  4. V checks: g^z = a * h^e

Completeness:  g^z = g^(r+ex) = g^r * g^(ex) = a * h^e  -- always holds
Special soundness: two accepting transcripts (a, e, z) and (a, e', z')
                   with e != e' yield:
                     x = (z - z') / (e - e')  mod q
Honest-verifier ZK: simulator picks z, e at random, computes a = g^z * h^(-e)
                     => transcript (a, e, z) has correct distribution
```

## Sigma Protocols

### Commit-Challenge-Response Structure

```
A Sigma protocol is a 3-move interactive proof with structure:

  Move 1 (Commitment):  P -> V:  a   (first message, commitment)
  Move 2 (Challenge):   V -> P:  e   (random challenge from challenge space)
  Move 3 (Response):    P -> V:  z   (response computed from a, e, witness)

Properties:
  Completeness:         honest execution always accepts
  Special soundness:    two accepting transcripts with same a, different e
                        => can extract the witness
  Honest-verifier ZK:  exists simulator that produces valid transcripts
                        without knowing the witness

Examples of Sigma protocols:
  Schnorr (discrete log)         Guillou-Quisquater (RSA)
  Okamoto (representation)       Chaum-Pedersen (DLOG equality)
  AND/OR composition             Pedersen commitment opening
```

## Fiat-Shamir Heuristic

### Non-Interactive Zero-Knowledge

```
Transform any public-coin interactive proof into a non-interactive one:

  Interactive:                    Non-interactive (Fiat-Shamir):
    P -> V: a (commitment)         P computes: a (commitment)
    V -> P: e (random challenge)   P computes: e = H(a || x)  (hash of commitment + statement)
    P -> V: z (response)           P outputs:  proof = (a, z)

  Verification: V checks e = H(a || x) and accepts (a, e, z)

Security: provably secure in the Random Oracle Model (ROM)
          H is modeled as a truly random function
Advantage: single message from P to V, no interaction needed
           => enables blockchain verification, digital signatures

Schnorr signature = Fiat-Shamir applied to Schnorr protocol:
  sig(m) = (z, e) where e = H(g^r || m), z = r + e*x
```

## ZK-SNARKs

### Succinct Non-Interactive Arguments of Knowledge

```
SNARK = Succinct Non-interactive ARgument of Knowledge

Properties:
  Succinct:         proof size is O(1) or O(log n), verification is fast
  Non-interactive:  single message from prover to verifier
  Argument:         soundness holds against computationally bounded provers
  of Knowledge:     extractor can recover the witness

Pipeline (Groth16-style):
  1. Computation -> Arithmetic circuit over finite field F_p
  2. Circuit -> R1CS (Rank-1 Constraint System)
       (A_i . s) * (B_i . s) = (C_i . s)  for each gate i
       s = (1, x_1, ..., x_n, w_1, ..., w_m)  (public input + witness)
  3. R1CS -> QAP (Quadratic Arithmetic Program)
       polynomials A(x), B(x), C(x) such that
       A(x) * B(x) - C(x) = H(x) * Z(x)
       where Z(x) vanishes on evaluation domain
  4. QAP -> proof via polynomial commitment + pairing-based verification

Trusted Setup: generates structured reference string (SRS)
               toxic waste (tau) must be destroyed -- compromise => fake proofs
               ceremonies: Powers of Tau (multi-party computation)

Groth16:  3 group elements proof, 3 pairings verification
          most widely deployed SNARK (Zcash, Ethereum)
```

## ZK-STARKs

### Scalable Transparent Arguments of Knowledge

```
STARK = Scalable Transparent ARgument of Knowledge

Key differences from SNARKs:
  No trusted setup:  transparent -- only public randomness (hash functions)
  Post-quantum:      relies on collision-resistant hashes, not pairings/DL
  Scalable:          prover time quasi-linear, verifier time polylogarithmic

Core technique: FRI (Fast Reed-Solomon Interactive Oracle Proof)
  1. Encode computation trace as polynomial p(x) over evaluation domain
  2. Commit to p(x) via Merkle tree of evaluations
  3. FRI protocol proves p(x) has low degree:
       - Split p(x) = p_even(x^2) + x * p_odd(x^2)
       - Verifier sends random alpha
       - Prover commits to p'(x) = p_even(x) + alpha * p_odd(x)
       - Repeat: degree halves each round
       - Final round: constant polynomial, check directly
  4. Soundness from Reed-Solomon proximity testing

Proof size: O(log^2 n)  -- larger than SNARKs but no trusted setup
Prover:     O(n log n)
Verifier:   O(log^2 n)
```

## Bulletproofs

### Short Proofs Without Trusted Setup

```
Bulletproofs: short non-interactive ZK proofs, no trusted setup

Primary use: range proofs
  Prove v in [0, 2^n) without revealing v
  Used in: Monero (confidential transactions), Mimblewimble

Inner product argument:
  Prove knowledge of vectors a, b such that <a, b> = c
  and Pedersen commitment C = g^a * h^b * u^c
  Proof size: O(log n) group elements (logarithmic compression)

Protocol:
  1. Prover sends L, R (cross-term commitments)
  2. Verifier sends random challenge x
  3. Reduce dimension by half: a' = x*a_lo + x^(-1)*a_hi
  4. Repeat until dimension 1
  5. Final: single scalar proof

Proof size:    O(log n) group elements
Verification:  O(n) (linear, slower than SNARKs)
No trusted setup, no pairings required
Aggregation:   multiple range proofs share amortized cost
```

## Applications

### Blockchain Privacy and Verifiable Computation

```
Blockchain privacy:
  Zcash:        zk-SNARKs (Groth16) for shielded transactions
  Monero:       Bulletproofs for confidential amounts + ring signatures
  Mimblewimble: Bulletproofs + CoinJoin-style aggregation
  zkSync, StarkNet, Polygon zkEVM: ZK rollups for Ethereum scaling

Authentication:
  Prove identity without revealing credentials
  Prove age >= 18 without revealing birthdate
  Prove membership in a group without revealing which member

Verifiable computation:
  Outsource computation to untrusted server
  Server returns result + ZK proof of correct execution
  Client verifies in O(log n) time
  Applications: cloud computing, ML inference verification

Voting:
  Prove vote is valid without revealing choice
  Prove eligibility without revealing identity

Supply chain / compliance:
  Prove regulatory compliance without revealing trade secrets
  Prove solvency without revealing individual balances
```

## Key Figures

```
Shafi Goldwasser, Silvio Micali, Charles Rackoff (1985/1989):
  Defined zero-knowledge proofs, proved IP has ZK proofs for all of NP

Oded Goldreich, Silvio Micali, Avi Wigderson (1986):
  ZK proof for graph 3-coloring => all of NP has ZK proofs

Amos Fiat, Adi Shamir (1986):
  Fiat-Shamir heuristic for non-interactive ZK

Claus-Peter Schnorr (1989):
  Schnorr protocol for discrete log proof of knowledge

Jens Groth (2016):
  Groth16 -- most efficient pairing-based SNARK

Eli Ben-Sasson, Iddo Bentov, Yinon Horesh, Michael Riabzev (2018):
  STARKs and the FRI protocol -- transparent, post-quantum ZK

Benedikt Bunz, Jonathan Bootle, et al. (2018):
  Bulletproofs -- short proofs without trusted setup
```

## See Also

- computational-complexity
- cryptography
- number-theory
- graph-theory
- boolean-satisfiability

## References

```
Goldwasser, Micali, Rackoff. "The Knowledge Complexity of Interactive Proof Systems." (1985/1989)
Goldreich, Micali, Wigderson. "How to Prove All NP Statements in Zero-Knowledge." (1986)
Schnorr. "Efficient Signature Generation by Smart Cards." J. Cryptology (1991)
Groth. "On the Size of Pairing-Based Non-Interactive Arguments." EUROCRYPT (2016)
Ben-Sasson et al. "Scalable, Transparent, and Post-Quantum Secure Computational Integrity." (2018)
Bunz et al. "Bulletproofs: Short Proofs for Confidential Transactions and More." S&P (2018)
Thaler. "Proofs, Arguments, and Zero-Knowledge." (2022) -- comprehensive textbook
```
