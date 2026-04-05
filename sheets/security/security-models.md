# Security Models

> Formal models that define rules for access control, information flow, and integrity enforcement — Bell-LaPadula, Biba, Clark-Wilson, Brewer-Nash, and more.

## Bell-LaPadula Model (BLP)

```
# Focus: CONFIDENTIALITY (military/government classification)
# Prevents unauthorized disclosure of information

# Simple Security Property (ss-property) — "No Read Up"
# A subject at clearance level L cannot READ objects at
# classification level higher than L
# Example: Secret-cleared user cannot read Top Secret documents

# Star Property (*-property) — "No Write Down"
# A subject at clearance level L cannot WRITE to objects at
# classification level lower than L
# Example: Top Secret user cannot write to Secret files
# Prevents leaking classified data to lower levels

# Strong Star Property
# A subject can read and write ONLY at their own level
# Most restrictive — no cross-level access at all

# Discretionary Security Property (ds-property)
# Uses an access control matrix to further restrict access
# Access must satisfy BOTH mandatory (ss, *) AND discretionary rules

# Security levels (ordered):
# Top Secret > Secret > Confidential > Unclassified

# Dominance relation: L1 dominates L2 if:
#   level(L1) >= level(L2) AND categories(L1) ⊇ categories(L2)

# Example with categories:
# (Secret, {NATO, Nuclear}) dominates (Confidential, {NATO})
# (Secret, {NATO}) does NOT dominate (Confidential, {Nuclear})
#   — missing the Nuclear category

# Tranquility properties:
# Strong tranquility — security labels never change
# Weak tranquility — labels never change in a way that violates policy
```

## Biba Integrity Model

```
# Focus: INTEGRITY (dual/inverse of Bell-LaPadula)
# Prevents unauthorized modification of data

# Simple Integrity Axiom — "No Read Down"
# A subject at integrity level I cannot READ objects at
# integrity levels LOWER than I
# Prevents contamination from less trustworthy sources

# Star Integrity Axiom (*-integrity) — "No Write Up"
# A subject at integrity level I cannot WRITE to objects at
# integrity levels HIGHER than I
# Prevents untrusted subjects from corrupting trusted data

# Invocation property
# A subject cannot invoke (call/request service from) a subject
# at a higher integrity level
# Prevents lower-integrity processes from commanding higher ones

# Integrity levels (example):
# Crucial > Important > Useful > Unverified

# Comparison with BLP:
# BLP:  No Read Up,   No Write Down  (protects confidentiality)
# Biba: No Read Down, No Write Up    (protects integrity)
# They are mathematical duals — mirror images

# Lipner's combination:
# Use BLP for confidentiality + Biba for integrity simultaneously
# Assign both security clearance AND integrity level to subjects/objects
```

## Clark-Wilson Model

```
# Focus: INTEGRITY through well-formed transactions
# Models real-world commercial integrity (separation of duties)

# Key components:
# CDI — Constrained Data Items (protected data)
# UDI — Unconstrained Data Items (unprotected/external data)
# TP  — Transformation Procedures (authorized operations on CDIs)
# IVP — Integrity Verification Procedures (validate CDI consistency)

# Rules:
# 1. All CDIs must be in a valid state (verified by IVPs)
# 2. TPs must transform CDIs from one valid state to another
#    (well-formed transactions)
# 3. Users can only access CDIs through authorized TPs
#    (access triple: User, TP, CDI)
# 4. Separation of duties — no single user can perform all
#    steps of a critical transaction
# 5. UDIs must be validated/transformed by a TP before becoming CDIs

# Certification rules (C1–C5):
# C1: IVPs must verify all CDIs are valid
# C2: TPs must be certified to maintain CDI integrity
# C3: Access triples enforce separation of duties
# C4: TPs must write to an append-only log (audit trail)
# C5: TPs converting UDIs must validate input properly

# Enforcement rules (E1–E4):
# E1: System must maintain list of certified access triples
# E2: System must enforce access triple restrictions
# E3: Authentication of users required
# E4: Only security officer can modify access triples

# Example: banking transaction
# CDI = account balances
# TP = transfer procedure (debit + credit)
# IVP = balance verification (total assets = total liabilities)
# Separation: teller initiates, manager approves
```

## Brewer-Nash Model (Chinese Wall)

```
# Focus: CONFLICT OF INTEREST prevention
# Dynamically restricts access based on prior access history

# Concept:
# Objects are grouped into conflict-of-interest classes
# Once a user accesses data from one company in a class,
# they cannot access data from competing companies

# Example:
# Conflict class: "Banks" = {Bank A, Bank B, Bank C}
# Conflict class: "Oil" = {Oil Corp X, Oil Corp Y}
# Consultant accesses Bank A data → blocked from Bank B, Bank C
# Consultant can still access Oil Corp X (different class)

# Rules:
# Read rule: subject can read object O only if:
#   O is in the same company dataset as a previously accessed object, OR
#   O belongs to a conflict class from which no object has been accessed
# Write rule: subject can write to object O only if:
#   No object from a different company in the same conflict class
#   has been read (prevents indirect information flow)

# Properties:
# Access restrictions grow over time (more access = more restrictions)
# Initially, all access is permitted
# Supports the principle of "ethical walls" in consulting/finance
# Used in: law firms, consulting, investment banking, auditing
```

## Graham-Denning Model

```
# Focus: SECURE CREATION AND DELETION of subjects and objects
# Defines 8 primitive protection operations

# 8 protection rules:
# 1. Create object        — subject creates a new object
# 2. Create subject       — subject creates a new subject
# 3. Delete object        — subject deletes an object
# 4. Delete subject       — subject deletes a subject
# 5. Read access right    — subject reads access rights of an object
# 6. Grant access right   — subject grants access rights to another
# 7. Delete access right  — subject removes access rights
# 8. Transfer access right — subject transfers own rights to another

# Key principle:
# Every operation must be explicitly defined and authorized
# The model specifies WHO can perform each operation on WHAT
# Uses an access control matrix to track permissions

# Addresses questions BLP and Biba ignore:
# How are new subjects/objects created securely?
# How are access rights modified?
# Who has the authority to grant/revoke permissions?
```

## Harrison-Ruzzo-Ullman (HRU) Model

```
# Focus: DECIDABILITY of access control systems
# Formal model proving security properties of protection systems

# Components:
# Subjects (S) — active entities
# Objects (O) — passive entities (subjects are also objects)
# Rights (R) — set of access rights (read, write, own, etc.)
# Commands — operations that modify the access matrix

# Primitive operations:
# create subject s, create object o
# destroy subject s, destroy object o
# enter r into [s, o]  — grant right r to s on o
# delete r from [s, o] — revoke right r from s on o

# Safety question:
# "Can subject s ever obtain right r on object o?"
# This is UNDECIDABLE in general (proven equivalent to halting problem)
# Decidable only for mono-operational commands (single operation per command)

# Significance:
# Proves that general access control is computationally hard
# Motivates simpler models with restricted command structures
# Foundation for understanding limits of access control verification
```

## Take-Grant Model

```
# Focus: RIGHTS PROPAGATION in a directed graph
# Models how access rights can spread through a system

# Graph representation:
# Nodes = subjects and objects
# Edges = access rights between them
# Special rights: "take" and "grant"

# Four rules:
# Take rule:  if s has "take" right on x, and x has right r on y,
#             then s can take right r on y (s gets r on y)
# Grant rule: if s has "grant" right on x, and s has right r on y,
#             then s can give x the right r on y
# Create rule: s can create a new node with any rights from s
# Remove rule: s can remove rights from an edge s controls

# Analysis:
# Can determine if subject A can ever gain right R on object B
# by checking if there is a path of take/grant edges
# Decidable in linear time O(n + e) where n = nodes, e = edges
# Much faster than HRU for the problems it can model
```

## State Machine Model

```
# Foundation for most formal security models
# System modeled as a finite state machine

# Components:
# States (S) — set of all possible system states
# Inputs (I) — set of possible inputs/requests
# Transitions (T) — state transition function T: S × I → S
# Initial state (s₀) — starting state
# Output function — maps states to outputs

# Secure state:
# A state is "secure" if it satisfies all security invariants
# The system is secure if:
#   1. The initial state is secure
#   2. Every transition from a secure state leads to a secure state
# (Inductive proof of security)

# BLP as state machine:
# State = (subjects, objects, access matrix, labels)
# Input = access request (subject, mode, object)
# Transition = grant or deny based on ss-property and *-property
# Secure state = no subject violates read-up or write-down rules
```

## Information Flow Model

```
# Focus: controlling how information moves between security levels
# Generalizes confidentiality and integrity models

# Information flow classes form a lattice:
#   Higher classification → lower classification: not permitted
#   Lower → higher: permitted (for confidentiality)
#   Covert channels violate flow policy unintentionally

# Lattice structure:
# Every pair of security levels has:
#   Least Upper Bound (LUB) — join (⊔)
#   Greatest Lower Bound (GLB) — meet (⊓)

# Example lattice:
#       Top Secret
#      /          \
#   Secret      Secret
#  {NATO}      {Nuclear}
#      \          /
#      Confidential
#           |
#      Unclassified

# Information can flow UP the lattice but not DOWN
# Covert channels: unintended information flow paths
#   Timing channels: modulate resource timing to signal data
#   Storage channels: modulate shared resources to signal data
```

## Noninterference Model

```
# Focus: ensuring actions at one security level do not affect
# observations at another level

# Formal definition:
# A system satisfies noninterference if the actions of
# High-level users have NO observable effect on what
# Low-level users can see

# If High performs actions a₁, a₂, ..., aₙ
# Low's view of the system is identical whether or not
# High performed those actions

# Stronger than BLP because it addresses covert channels
# BLP only prevents direct read/write violations
# Noninterference prevents ANY information leakage

# Challenge: very difficult to achieve in practice
# Timing variations, resource contention, and cache behavior
# can all leak information between levels
```

## Lattice-Based Access Control

```
# Generalized framework underlying BLP, Biba, and flow models
# Security labels form a mathematical lattice

# Lattice properties:
# Partially ordered set with:
#   Reflexive:     a ≤ a
#   Antisymmetric: a ≤ b and b ≤ a → a = b
#   Transitive:    a ≤ b and b ≤ c → a ≤ c
# Plus LUB and GLB for every pair

# In BLP: lattice = security levels × categories
# Subject clearance and object classification are lattice elements
# Access decisions follow the dominance relation on the lattice

# Composite labels:
# Label = (Level, {Categories})
# (Secret, {NATO, CRYPTO}) ⊔ (Confidential, {Nuclear})
# = (Secret, {NATO, CRYPTO, Nuclear})
```

## Access Control Matrix

```
# Foundational representation of all access control decisions
# Rows = subjects, Columns = objects, Cells = permissions

#              File1    File2    Printer  Process1
# Alice     [  rw       r        p        --     ]
# Bob       [  r        rw       p        x      ]
# Charlie   [  --       r        --       x      ]

# Implementations:
# ACL (Access Control List) — column view
#   File1: Alice=rw, Bob=r
#   Each object stores its list of authorized subjects
#
# Capability list — row view
#   Alice: File1=rw, File2=r, Printer=p
#   Each subject holds tokens for objects they can access
#
# Trade-offs:
# ACL: easy to see who can access an object
#      hard to see everything a subject can access
# Capability: easy to see what a subject can access
#             hard to revoke access across all subjects
```

## Tips

- Bell-LaPadula protects confidentiality (military); Biba protects integrity (commercial) — know which model fits which requirement.
- Clark-Wilson is the most practical model for real-world business applications because it enforces separation of duties and well-formed transactions.
- Brewer-Nash is essential for consulting, legal, and financial services where conflict of interest is a regulatory concern.
- The HRU proof that general access control is undecidable is a foundational result — it explains why practical systems use restricted models.
- Most real systems combine multiple models: BLP for classification + Biba for integrity + Clark-Wilson for transaction control.

## See Also

- security-governance, identity-management, acl, selinux, asset-security

## References

- [Bell & LaPadula — Secure Computer Systems: Mathematical Foundations (1973)](https://csrc.nist.gov/csrc/media/publications/conference-paper/1998/10/08/proceedings-of-the-21st-nissc-1998/documents/early-cs-papers/bell76.pdf)
- [Biba — Integrity Considerations for Secure Computer Systems (1977)](https://apps.dtic.mil/sti/citations/ADA039324)
- [Clark & Wilson — A Comparison of Commercial and Military Security Policies (1987)](https://doi.org/10.1109/SP.1987.10001)
- [Brewer & Nash — The Chinese Wall Security Policy (1989)](https://doi.org/10.1109/SECPRI.1989.36295)
- [Harrison, Ruzzo, Ullman — Protection in Operating Systems (1976)](https://doi.org/10.1145/360303.360333)
- [NIST SP 800-12 Rev 1 — An Introduction to Information Security](https://csrc.nist.gov/publications/detail/sp/800-12/rev-1/final)
- [Matt Bishop — Computer Security: Art and Science, Chapter 5 (Security Models)](https://nob.cs.ucdavis.edu/book/book-aands/)
