# SPDX Identifiers — Deep Dive

The grammar, algebra, and standards underlying SPDX expressions: how license identifiers compose into compatibility-checkable expressions, how SBOM formats anchor regulatory compliance, and how cryptographic attestation closes the supply-chain loop.

## Setup — SPDX as a Linux Foundation Standard; ISO/IEC 5962:2021 Alignment

SPDX (Software Package Data Exchange) began as a Linux Foundation working group in 2010 to address a then-unsolved problem: the absence of a machine-readable, vendor-neutral way to declare "what software is inside this package and what licenses cover each piece."

Pre-SPDX:
- Each ecosystem had its own metadata (npm `license`, Cargo `license`, Maven `<license>`)
- License names were free-text ("Apache", "Apache License", "Apache 2.0", "Apache-2", "ASL2", "Apache Software License Version 2.0")
- No standard expression syntax for "Apache OR MIT"
- No SBOM format that crossed ecosystems

SPDX delivered:
- **License Identifiers**: a curated list with stable short codes (Apache-2.0, MIT, GPL-3.0-only)
- **License Expression syntax**: a formal grammar for combining identifiers
- **SPDX Document format**: an SBOM specification (text, JSON, RDF)
- **License List**: machine-readable JSON of all approved identifiers

In 2021, ISO ratified SPDX 2.2.1 as **ISO/IEC 5962:2021** — international standard status. SPDX 3.0 (2024) extends with profile-based architecture for AI, datasets, security.

The Linux Foundation hosts the SPDX project. Stewardship is community-driven via the SPDX general specification working group. License List updates are reviewed by the License List Review Group.

```
2010: SPDX 1.0 — basic SBOM + 200 licenses
2014: SPDX 2.0 — relationships + RDF
2017: SPDX 2.1 — package-level details
2020: SPDX 2.2 — security data
2021: ISO/IEC 5962:2021 (SPDX 2.2.1)
2022: SPDX 2.3 — packaging fixes
2024: SPDX 3.0 — profile architecture (Software, AI, Dataset, Build, Security, Service)
```

## SPDX Expression Grammar (RFC-like)

SPDX License Expressions are defined in Annex D of the SPDX Specification. The grammar (extended Backus–Naur Form):

```
license-expression  = simple-expression
                    / compound-expression

simple-expression   = license-id
                    / license-id "+"
                    / license-ref

compound-expression = simple-expression
                    / simple-expression "WITH" license-exception-id
                    / compound-expression "AND" compound-expression
                    / compound-expression "OR"  compound-expression
                    / "(" compound-expression ")"

license-id            = <SPDX License Short Identifier>
license-exception-id  = <SPDX Exception Short Identifier>
license-ref           = ["DocumentRef-" idstring ":"] "LicenseRef-" idstring
idstring              = 1*(ALPHA / DIGIT / "-" / ".")
```

Examples:

| Expression | Meaning |
|------------|---------|
| `MIT` | Licensed under MIT |
| `GPL-3.0-only` | GPL-3.0, only that version |
| `GPL-3.0-or-later` | GPL-3.0 or any later version |
| `LGPL-2.1+` | LGPL-2.1 or any later version (legacy syntax, deprecated) |
| `MIT OR Apache-2.0` | Choose either MIT or Apache-2.0 |
| `MIT AND CC-BY-4.0` | Both apply (e.g., code under MIT, docs under CC-BY) |
| `(MIT OR Apache-2.0) AND BSD-3-Clause` | Choose MIT or Apache-2.0, plus BSD-3-Clause |
| `GPL-2.0-or-later WITH Classpath-exception-2.0` | GPL-2.0+ with the Classpath exception |
| `LicenseRef-MyCustom` | Non-SPDX license described in document |

**Precedence**: WITH binds tightest, then AND, then OR. Parentheses override.

`MIT AND BSD-3-Clause OR Apache-2.0` parses as `(MIT AND BSD-3-Clause) OR Apache-2.0` — wrong if you meant "MIT and (BSD-3-Clause or Apache-2.0)." Use parens.

`+` operator (deprecated in 2.0+):

- `GPL-2.0+` is now `GPL-2.0-or-later`
- `LGPL-2.1+` is now `LGPL-2.1-or-later`
- The `+` form is still parsed for backward compatibility

Why deprecated: `GPL-2.0+` is ambiguous ("only" vs "or-later"). The `-only` and `-or-later` suffixes are unambiguous and align with FSF's "GPLv2 or later" convention.

```
Expression parsing:
  "(MIT OR Apache-2.0) AND BSD-3-Clause"
       ↓ tokenize
  [LPAREN, MIT, OR, Apache-2.0, RPAREN, AND, BSD-3-Clause]
       ↓ parse with precedence: WITH > AND > OR
  AND
   ├── OR
   │    ├── MIT
   │    └── Apache-2.0
   └── BSD-3-Clause
```

## The Compatibility Algebra

SPDX expressions form an algebra over the lattice of licenses. The operators have semantic meaning:

**AND**: conjunction — must comply with both. Used when:
- Different parts of a package have different licenses (e.g., docs CC-BY-SA + code Apache-2.0)
- The licensor stipulates dual obligations

**OR**: disjunction — choose one. Used when:
- Dual-licensed packages (e.g., Rust ecosystem standard `MIT OR Apache-2.0`)
- Licensee picks the most convenient

**WITH**: exception modifier — license terms apply, modified by exception. Used for:
- GPL-with-Classpath-exception (linkable as if non-copyleft)
- GPL-with-LLVM-exception (similar)

The algebra:

- Commutative: `A OR B` ≡ `B OR A`; `A AND B` ≡ `B AND A`
- Associative: `(A AND B) AND C` ≡ `A AND (B AND C)`
- Distributive: `A AND (B OR C)` ≡ `(A AND B) OR (A AND C)`
- Idempotent: `A AND A` ≡ `A`; `A OR A` ≡ `A`
- Absorption: `A AND (A OR B)` ≡ `A`; `A OR (A AND B)` ≡ `A`

The simplification rules are exploited by SBOM tools to normalize expressions before comparison. `(MIT OR Apache-2.0) AND MIT` simplifies to `MIT`.

For compatibility checking against a policy:

```
policy_allowed = {MIT, Apache-2.0, BSD-3-Clause}
expr = "(MIT OR GPL-3.0) AND BSD-3-Clause"

normalize(expr) = (MIT AND BSD-3-Clause) OR (GPL-3.0 AND BSD-3-Clause)

For OR: at least one disjunct must be entirely in policy_allowed
  - (MIT AND BSD-3-Clause): both in policy → satisfied
  - (GPL-3.0 AND BSD-3-Clause): GPL-3.0 not in policy → not satisfied
  
At least one satisfied → expression compatible with policy
```

This is **policy-as-code**. Tools like Open Policy Agent (OPA) can evaluate these rules.

## License Identifier Stability

Once an SPDX identifier is allocated, it never changes. Stability is foundational — SBOMs from 2015 must remain readable in 2035.

The deprecation mechanism handles errata:
- An identifier can be marked `isDeprecatedLicenseId: true` in the License List JSON
- The identifier remains valid for parsing
- Tools should warn / suggest replacement
- A `seeAlso` field points to the recommended replacement

Examples of deprecation:

| Deprecated | Recommended | Reason |
|------------|-------------|--------|
| GPL-1.0 | GPL-1.0-only | Ambiguous version handling |
| GPL-2.0 | GPL-2.0-only | Same |
| GPL-3.0 | GPL-3.0-only | Same |
| AGPL-3.0 | AGPL-3.0-only | Same |
| GPL-1.0+ | GPL-1.0-or-later | Same (`+` syntax) |
| GFDL-1.1 | GFDL-1.1-only | Same |

The 2018 deprecation (SPDX 3.0 spec — note SPDX 3.0 of 2018 was a different effort than the 2024 SPDX 3.0) introduced the `-only` / `-or-later` suffixes. Tools updated their license databases; older SBOMs were retroactively re-mapped.

The migration:

```
Old SBOM: license = "GPL-3.0"
   ↓ tool reads license list
   isDeprecatedLicenseId = true, seeAlso = "GPL-3.0-or-later"
   ↓ tool decides per policy
   Strict: emit deprecation warning, treat as GPL-3.0-or-later (assume "or later")
   Conservative: treat as GPL-3.0-only (most restrictive interpretation)
```

This stability promise is rare in software standards. Compare to language standards where deprecated APIs eventually get removed. SPDX maintains backward parsing forever — older identifiers just emit warnings.

## The Deprecation Migration

The 2018 GPL-3.0 → GPL-3.0-only migration is the canonical case study. Before:

- `GPL-3.0` meant "GPL-3.0" but ambiguous about "or later"
- FSF convention: most projects say "GPL-3.0 or later" (allowing future GPL-4.0 to replace)
- Some say "GPL-3.0 only" (locked to exactly 3.0)
- Old SPDX `GPL-3.0` couldn't distinguish

After:
- `GPL-3.0-only` — exactly 3.0, no later versions
- `GPL-3.0-or-later` — 3.0 or any later FSF-published version

Migration impact:
- Existing SBOMs needed mapping (most tools added a translation table)
- New SBOMs use the unambiguous form
- License-detection tools (scancode, fossology) updated detection rules

For compliance: if you see `GPL-3.0` in an SBOM, treat as ambiguous and check the actual LICENSE file to determine intent. Most projects intend `GPL-3.0-or-later` (matching the FSF "or later" recommendation).

```
Pre-2018:           Post-2018:
GPL-2.0       →     GPL-2.0-only or GPL-2.0-or-later
GPL-2.0+      →     GPL-2.0-or-later
GPL-3.0       →     GPL-3.0-only or GPL-3.0-or-later
GPL-3.0+      →     GPL-3.0-or-later
LGPL-2.1      →     LGPL-2.1-only or LGPL-2.1-or-later
LGPL-2.1+     →     LGPL-2.1-or-later
AGPL-3.0      →     AGPL-3.0-only or AGPL-3.0-or-later
GFDL-1.3      →     GFDL-1.3-only or GFDL-1.3-or-later
```

## WITH Exception Mechanism

Some licenses have **exceptions** — modifications that grant additional rights or weaken restrictions. SPDX models these as separate identifiers in the [Exceptions List](https://spdx.org/licenses/exceptions-index.html).

Common exceptions:

**Classpath-exception-2.0** (used by OpenJDK):
- Modifies GPL-2.0
- Allows linking GPL-2.0'd Java code with non-GPL applications without triggering copyleft
- Used by: OpenJDK class library
- SPDX: `GPL-2.0-only WITH Classpath-exception-2.0`

**LLVM-exception** (used by LLVM):
- Modifies Apache-2.0
- Adds runtime library exception (linked code is not derivative)
- Used by: LLVM, Swift, Rust runtime
- SPDX: `Apache-2.0 WITH LLVM-exception`

**OpenSSL-exception** (legacy OpenSSL):
- Modifies GPL-2.0
- Permits linking with OpenSSL despite incompatibility
- Used by: many GPL-2.0 projects historically
- SPDX: `GPL-2.0-or-later WITH OpenSSL-exception`

**GCC-exception-3.1**:
- Runtime library exception for libgcc, libstdc++
- Modifies GPL-3.0+
- SPDX: `GPL-3.0-or-later WITH GCC-exception-3.1`

**Autoconf-exception-3.0**:
- Generated configure scripts can be MIT-like despite GPL-3.0 source
- SPDX: `GPL-3.0-or-later WITH Autoconf-exception-3.0`

The semantic: license terms apply, EXCEPT-AS-SPECIFIED by the exception. Compatibility analysis must consider the modified terms, not the base license alone.

```
GPL-2.0-only         → strong copyleft, viral on linking
+ Classpath-exception → linking carve-out for Java
= effectively LGPL-like for linking purposes
  but still copyleft for modifications to the GPL-licensed code itself
```

For a compatibility tool, encountering `WITH Classpath-exception-2.0` should adjust the rules:
- Linking-only use → permissive (treat like LGPL-style)
- Modifying the JDK source → still copyleft

## LicenseRef + DocumentRef — Namespacing Non-SPDX Licenses

Not every license has a SPDX identifier. The SPDX list curates ~600 licenses; bespoke licenses (EULAs, niche academic licenses, internal company licenses) are excluded.

For these, SPDX defines `LicenseRef-`:

```
LicenseRef-MyCustomLicense
LicenseRef-Internal-Acme-EULA-2023
LicenseRef-Vendor-XYZ-Proprietary
```

The identifier is local to the SPDX document. The actual license text must be embedded in the document via the `ExtractedLicensingInfo` section:

```
LicenseID: LicenseRef-MyCustomLicense
ExtractedText: <text>
"Permission to use this code is granted to..."
"All redistributions must..."
"Patent claims..."
</text>
LicenseName: My Custom License v1
LicenseCrossReference: https://example.com/license
```

For cross-document references (one SPDX doc references a license defined in another), `DocumentRef-` namespaces:

```
DocumentRef-AcmeLicenseDoc:LicenseRef-AcmeEULA
```

The `DocumentRef-AcmeLicenseDoc` resolves to a separate SPDX document (referenced via `ExternalDocumentRef`).

The **LICENSES/ directory convention** (REUSE specification, SPDX-aligned):

```
project/
├── LICENSES/
│   ├── MIT.txt
│   ├── Apache-2.0.txt
│   ├── LicenseRef-MyCustom.txt
│   └── ...
├── src/
│   ├── main.go         # SPDX-License-Identifier: MIT
│   └── proprietary.c   # SPDX-License-Identifier: LicenseRef-MyCustom
└── ...
```

Each source file declares its license via `SPDX-License-Identifier:` comment header. The LICENSES/ directory holds the actual texts.

```
// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2025 Acme Corp <legal@acme.com>

package main
```

This convention scales: a polyglot project with hundreds of files can have per-file licensing, machine-readable, with full text available locally. SBOM tools scan headers, build the SPDX expression, and produce per-file manifests.

## SBOM Format Theory

A **Software Bill of Materials** is the analog of a manufacturing BoM: a complete enumeration of components in a software product. The components include:

- First-party code (the project's own code)
- Direct dependencies (declared in package manifests)
- Transitive dependencies (dependencies of dependencies)
- System libraries (libc, OS-level)
- Container base images (for containerized software)
- Source-code licenses (from each component's LICENSE)
- Cryptographic hashes (proving content integrity)

SBOM formats:

**SPDX 2.3** (ISO/IEC 5962:2021):
- Document → Packages → Files (with relationships)
- Text/tag-value, JSON, YAML, RDF/XML serializations
- Strong license-identifier story
- Linux Foundation governance

**CycloneDX 1.5/1.6** (OWASP):
- Component-tree model
- JSON, XML, Protobuf
- Originally vulnerability-focused, now full SBOM
- OWASP governance

**SWID Tags (ISO/IEC 19770-2)**:
- XML-only
- Software identification, asset management
- Less popular for FOSS dependency declaration

The convergence: NTIA's "Minimum Elements for an SBOM" (2021) is format-neutral. SPDX, CycloneDX, and SWID can all express the minimum elements. Tools translate between formats (Syft outputs SPDX or CycloneDX; CycloneDX CLI converts to/from SPDX).

```
                Component data
                       ↓
         +-------------+-------------+
         |             |             |
       SPDX        CycloneDX        SWID
         |             |             |
         +-------------+-------------+
                       ↓
              NTIA Minimum Elements
                  (covered by all)
```

Format choice considerations:
- **SPDX**: regulatory familiarity (US gov references it), license expression rigor, ISO standard
- **CycloneDX**: vulnerability-aware (VEX integration), simpler JSON, OWASP-aligned
- **Both**: large ecosystems support both; pick based on consumer needs

## SPDX Document Anatomy

An SPDX document (tag-value or JSON) has sections:

**Document Creation**:
```
SPDXVersion: SPDX-2.3
DataLicense: CC0-1.0
SPDXID: SPDXRef-DOCUMENT
DocumentName: my-project-1.2.3
DocumentNamespace: https://acme.com/spdx/my-project-1.2.3-uuid
Creator: Tool: syft-0.95
Creator: Organization: Acme Corp
Created: 2025-04-25T12:00:00Z
```

The `DocumentNamespace` must be globally unique — convention is URL with UUID. Two SBOM documents must never share a namespace.

**Package Information**:
```
PackageName: lodash
SPDXID: SPDXRef-Package-lodash
PackageVersion: 4.17.21
PackageDownloadLocation: https://registry.npmjs.org/lodash/-/lodash-4.17.21.tgz
PackageLicenseConcluded: MIT
PackageLicenseDeclared: MIT
PackageCopyrightText: Copyright OpenJS Foundation and other contributors
PackageChecksum: SHA1: e3a4b5c...
PackageHomePage: https://lodash.com/
ExternalRef: PACKAGE-MANAGER purl pkg:npm/lodash@4.17.21
```

`PackageLicenseConcluded` vs `PackageLicenseDeclared`:
- Declared: what the package claims (e.g., from package.json's "license" field)
- Concluded: what the SBOM tool determined after analysis (e.g., scanning LICENSE file with scancode)

When they differ, declared is "what they say" and concluded is "what we found." Auditors trust concluded more.

**File Information**:
```
FileName: ./node_modules/lodash/array.js
SPDXID: SPDXRef-File-lodash-array-js
FileChecksum: SHA1: a1b2c3...
LicenseConcluded: MIT
LicenseInfoInFile: MIT
FileCopyrightText: Copyright OpenJS Foundation
```

**Relationships**:
```
Relationship: SPDXRef-Package-myproject DEPENDS_ON SPDXRef-Package-lodash
Relationship: SPDXRef-Package-lodash DEPENDS_ON SPDXRef-Package-debug
Relationship: SPDXRef-DOCUMENT DESCRIBES SPDXRef-Package-myproject
Relationship: SPDXRef-Package-myproject CONTAINS SPDXRef-File-main-go
```

Relationships form a directed graph; tools can render dependency trees, find longest paths, identify deeply-nested supply chains.

**SPDXID convention**: `SPDXRef-` prefix + alphanumeric/hyphens. Must be unique within the document. Enables cross-references.

## Relationship Types

SPDX 2.3 defines ~40 relationship types. Key ones:

- `DEPENDS_ON` — required for runtime/build
- `DEPENDENCY_OF` — inverse of DEPENDS_ON
- `BUILD_DEPENDENCY_OF` — required only at build time
- `RUNTIME_DEPENDENCY_OF` — required at runtime
- `OPTIONAL_DEPENDENCY_OF` — used if available
- `DESCRIBES` — root relationship: this document describes that package
- `CONTAINS` — package contains file (or sub-package)
- `GENERATES` — A generates B (e.g., compiler → binary)
- `GENERATED_FROM` — inverse
- `STATIC_LINK` / `DYNAMIC_LINK` — linkage type
- `DEV_DEPENDENCY_OF` — dev-only
- `TEST_DEPENDENCY_OF` — test-only
- `EXAMPLE_OF` — example/sample
- `DOCUMENTATION_OF` — documentation for X
- `OTHER` — fallback

The directed-graph semantics enable analyses:
- **Transitive closure**: all packages reachable via DEPENDS_ON from root
- **Cycles**: usually a bug, but possible in dev/build dependency graphs
- **Reachability**: "is package X actually used?" by traversing from DESCRIBES
- **License flow**: aggregate licenses by walking the dependency tree

```
DESCRIBES → root package
   |
   ├─ DEPENDS_ON → package A (license MIT)
   |    |
   |    └─ DEPENDS_ON → package C (license Apache-2.0)
   |
   └─ DEPENDS_ON → package B (license GPL-3.0)
        |
        └─ DEPENDS_ON → package D (license MIT)

Transitive licenses: {MIT, Apache-2.0, GPL-3.0}
Compatibility umbrella: GPL-3.0 (most restrictive)
```

## SPDX 3.0 Profile-Based Architecture

SPDX 3.0 (released 2024) restructures around **profiles**: layered conformance for different use cases.

**Core Profile** (always required):
- Element model (Artifact, Package, File, Relationship)
- Identifier system (SPDXIDs, URIs)
- Creation info (timestamp, creator, namespace)

**Software Profile** (most common):
- Software-specific Package/File details
- License expressions
- VCS metadata
- Equivalent to SPDX 2.x core

**Build Profile**:
- Build metadata (build system, configuration)
- Reproducibility data
- Build relationships

**AI Profile**:
- Model artifacts (weights, training data references)
- Training data SBOM
- Model card data (intended use, biases, evaluations)
- Aligns with NIST AI RMF / EU AI Act

**Dataset Profile**:
- Dataset descriptors (size, format, license, source)
- Used by AI training and analytics
- Privacy/PII annotations

**Security Profile**:
- VEX (Vulnerability Exploitability eXchange) data
- CVE references with status (affected, not_affected, fixed, under_investigation)
- VDR (Vulnerability Disclosure Report)

**Service Profile**:
- SaaS / web-service descriptors
- API endpoints, authentication, dependencies on services

**Lite Profile**:
- Minimum subset for low-resource scenarios (IoT, embedded)
- Drops some Core requirements

Profile composition: a single SPDX 3.0 document can use multiple profiles. An ML inference service might use Core + Software + AI + Service profiles.

```
SPDX 3.0 document
       ↓
   Core (mandatory)
       ↓
   + Software (typical)
   + Build (CI/CD-aware)
   + AI (model SBOM)
   + Dataset (training data)
   + Security (vulnerability state)
   + Service (SaaS surface)
   + Lite (low-resource)
```

The profile architecture enables vendor-specific extensions without bloating the core spec. AI/ML SBOM is currently the highest-velocity profile due to EU AI Act regulatory pressure.

## The PURL Standard — Package URL

Package URL (purl) is a syntax for uniquely identifying a software package. It complements SPDX (often used in `ExternalRef`).

```
pkg:type/namespace/name@version?qualifiers#subpath
```

Examples:

| purl | Meaning |
|------|---------|
| `pkg:npm/lodash@4.17.21` | NPM lodash 4.17.21 |
| `pkg:maven/org.apache.commons/commons-lang3@3.12.0` | Maven commons-lang3 3.12.0 |
| `pkg:pypi/django@4.2.0` | PyPI Django 4.2.0 |
| `pkg:cargo/serde@1.0.150` | Cargo serde 1.0.150 |
| `pkg:gem/rails@7.0.4` | RubyGems Rails 7.0.4 |
| `pkg:golang/github.com/gorilla/mux@v1.8.0` | Go module gorilla/mux v1.8.0 |
| `pkg:docker/library/nginx@latest?tag=latest` | Docker nginx latest |
| `pkg:rpm/centos/curl@7.61.1?arch=x86_64` | CentOS curl RPM x86_64 |
| `pkg:deb/debian/curl@7.74.0-1.3+deb11u7?arch=amd64` | Debian curl |
| `pkg:nuget/Newtonsoft.Json@13.0.1` | NuGet Newtonsoft.Json |
| `pkg:github/torvalds/linux@v6.5.0` | GitHub repo at tag |

Component fields:
- **type** — package ecosystem (npm, maven, pypi, cargo, etc.)
- **namespace** — vendor/group/owner (optional, ecosystem-dependent)
- **name** — package name
- **version** — version string (URL-encoded)
- **qualifiers** — type-specific metadata (?arch=x86_64, ?type=docker)
- **subpath** — sub-resource within package (#path/to/file)

Why purl matters:
- **Cross-ecosystem normalized identity**: same package across SBOMs has same purl
- **Vulnerability matching**: CVE feeds increasingly use purl to identify affected packages
- **Tool interop**: SPDX, CycloneDX, OSV, OSV-Scanner, GitHub SBOM all use purl

CycloneDX uses purl as the primary package identifier; SPDX includes it via `ExternalRef PACKAGE-MANAGER purl ...`.

```
SBOM tool ──┐
CVE feed   ──┼─ all key on purl ─→ deterministic matching
SCA tool   ──┘
```

## NTIA Minimum Elements

The US National Telecommunications and Information Administration published "Minimum Elements for an SBOM" in 2021 (per Executive Order 14028). Per-component:

1. **Author Name** — entity that created the SBOM data for this component
2. **Supplier Name** — entity that produced the component itself
3. **Component Name**
4. **Component Version**
5. **Other Unique Identifiers** — e.g., CPE, purl, SWID
6. **Dependency Relationship** — how this component relates to others
7. **Author of SBOM Data** — who/what generated the SBOM

Plus document-level:
- **Timestamp** — when SBOM was created

The minimum elements are intentionally format-neutral — any of SPDX, CycloneDX, or SWID can satisfy them. The point is the *information*, not the format.

US federal vendors (since EO 14028, 2021) must provide SBOMs to federal customers on request. The minimum elements are the floor.

```
NTIA Minimum Elements (Per Component):
  Author       — who said this  
  Supplier     — who made it
  Name         — what it is
  Version      — which one  
  Identifiers  — purl, CPE, SWID, etc.
  Dependencies — what it relies on
  SBOM Author  — who generated SBOM
  
Document level:
  Timestamp    — when generated
```

Conformance:
- SPDX: all fields covered by `Package*` and creation info sections
- CycloneDX: all fields covered by `metadata` and `components` sections
- SWID: all fields covered (less common in FOSS)

## Executive Order 14028 + EU CRA

Regulatory drivers for SBOM adoption:

**US Executive Order 14028** ("Improving the Nation's Cybersecurity," May 2021):
- Mandates federal contractors provide SBOMs
- NIST Special Publication 800-218 (SSDF) defines secure software development framework
- OMB M-22-18 (2022): software vendors to federal agencies must self-attest SSDF compliance
- CISA SBOM resources page tracks implementation guidance
- Section 4(e): "the contractor shall provide a Software Bill of Materials (SBOM) for each product directly or by publishing it on a public website"

**EU Cyber Resilience Act (CRA)** (2024, enforced 2027):
- Manufacturers of "products with digital elements" must:
  - Maintain SBOM for the product lifetime
  - Disclose vulnerabilities to ENISA within 24h of awareness
  - Issue patches for actively exploited vulns
- Scope: most consumer + commercial software
- Penalties: up to €15M or 2.5% of worldwide annual turnover

**EU AI Act** (2024):
- High-risk AI systems must have SBOM-like documentation
- Training data provenance disclosed
- Model cards required (parallel to AI SBOM profile in SPDX 3.0)

**FDA SBOM Guidance** (2023):
- Medical device cybersecurity submissions require SBOM
- 510(k) and PMA filings now expect SBOM
- Section 524B of FD&C Act

**DoD's Continuous ATO + RMF**:
- DoD Risk Management Framework requires component inventory
- SBOM increasingly accepted as CMVP/RMF artifact

Combined effect: SBOM is no longer optional for serious software vendors. Federal contracts, EU CE marking, FDA submissions all require it.

```
2021: EO 14028 — federal SBOM mandate
2022: OMB M-22-18 — self-attestation
2023: FDA SBOM guidance — medical devices
2024: EU CRA passed — manufacturer SBOM
2024: EU AI Act passed — AI SBOM
2027: EU CRA enforcement begins
```

## Sigstore + SBOM Attestation

An SBOM is just a JSON file. Without provenance, it's untrustworthy — a vendor could lie. **Attestation** binds the SBOM to:
- A specific software artifact (by hash)
- A specific identity (signer)
- A specific build process (provenance)

**Sigstore** (Linux Foundation, 2021):
- **Cosign** — sign artifacts and attestations
- **Fulcio** — short-lived signing certificates from OIDC identity (GitHub, Google, etc.)
- **Rekor** — public, append-only transparency log for signatures

**in-toto attestation envelope**:

```json
{
  "_type": "https://in-toto.io/Statement/v1",
  "subject": [
    {
      "name": "myimage",
      "digest": {"sha256": "abc123..."}
    }
  ],
  "predicateType": "https://spdx.dev/Document/v3",
  "predicate": {
    "spdxVersion": "SPDX-3.0",
    "...": "..."
  }
}
```

The envelope:
- `subject` — what artifact this is about (with hash)
- `predicateType` — what kind of statement (SPDX SBOM, SLSA provenance, etc.)
- `predicate` — the statement content

Wrapped in a **DSSE envelope** (Dead Simple Signing Envelope):

```json
{
  "payloadType": "application/vnd.in-toto+json",
  "payload": "<base64-encoded statement>",
  "signatures": [
    {"keyid": "...", "sig": "..."}
  ]
}
```

The verification chain:

```
1. User downloads myimage with sha256:abc123
2. Cosign retrieves attestation: cosign download attestation myimage
3. Verifies signature against Fulcio cert chain
4. Verifies cert tied to expected identity (e.g., GitHub Actions in repo X)
5. Verifies Rekor log entry (transparency)
6. Extracts SBOM from predicate
7. SBOM proven authentic + tied to artifact
```

Key SBOM attestation use cases:

- **Supply chain integrity**: verify SBOM came from official builder
- **Defense against tampering**: a man-in-the-middle SBOM swap is detectable
- **Auditability**: Rekor log is public, append-only — cannot be backdated
- **Identity binding**: SBOM is tied to a specific repo/builder (not just an opaque key)

Sigstore + SLSA + SBOM combine into the modern **secure supply chain**:

- SLSA Level 3+: build provenance (in-toto attestation of build process)
- SBOM (SPDX or CycloneDX): component inventory
- Cosign signature: bundles + transparent log

```
Build pipeline:
  Source ─→ SLSA provenance attestation
     ↓
  Build ──→ Artifact + SBOM
     ↓                ↓
   Cosign         Cosign
   sign           sign
     ↓                ↓
   Attest         Attest
   provenance     SBOM
     ↓                ↓
   Rekor          Rekor
   log            log
     ↓                ↓
   ──────── Released artifact + bundle ────────
                     ↓
              Consumer verifies:
              1. Cosign verify signatures
              2. Fulcio cert chain valid
              3. Rekor log entries present
              4. SLSA + SBOM predicates extracted
              5. Identity matches expected builder
              6. SBOM contents match policy
```

## References

- SPDX Specification 2.3 — https://spdx.github.io/spdx-spec/v2.3/
- SPDX Specification 3.0 — https://spdx.github.io/spdx-spec/v3.0.1/
- SPDX License List — https://spdx.org/licenses/
- SPDX Exceptions List — https://spdx.org/licenses/exceptions-index.html
- SPDX License Expressions (Annex D) — https://spdx.github.io/spdx-spec/v2.3/SPDX-license-expressions/
- ISO/IEC 5962:2021 — https://www.iso.org/standard/81870.html
- CycloneDX Specification — https://cyclonedx.org/specification/overview/
- SWID Tags (ISO/IEC 19770-2) — https://www.iso.org/standard/65666.html
- Package URL (purl) Specification — https://github.com/package-url/purl-spec
- NTIA Minimum Elements for an SBOM — https://www.ntia.gov/sites/default/files/publications/sbom_minimum_elements_report_0.pdf
- Executive Order 14028 — https://www.whitehouse.gov/briefing-room/presidential-actions/2021/05/12/executive-order-on-improving-the-nations-cybersecurity/
- OMB M-22-18 — https://www.whitehouse.gov/wp-content/uploads/2022/09/M-22-18.pdf
- NIST SP 800-218 (SSDF) — https://csrc.nist.gov/publications/detail/sp/800-218/final
- CISA SBOM Resources — https://www.cisa.gov/sbom
- EU Cyber Resilience Act — https://eur-lex.europa.eu/eli/reg/2024/2847/oj
- EU AI Act — https://eur-lex.europa.eu/eli/reg/2024/1689/oj
- FDA Cybersecurity Guidance for Premarket Submissions (2023) — https://www.fda.gov/regulatory-information/search-fda-guidance-documents/cybersecurity-medical-devices-quality-system-considerations-and-content-premarket-submissions
- REUSE Specification — https://reuse.software/spec/
- Sigstore project — https://www.sigstore.dev/
- Cosign — https://github.com/sigstore/cosign
- Fulcio — https://github.com/sigstore/fulcio
- Rekor — https://github.com/sigstore/rekor
- in-toto framework — https://in-toto.io/
- DSSE specification — https://github.com/secure-systems-lab/dsse
- SLSA framework — https://slsa.dev/
- Syft (SBOM generator) — https://github.com/anchore/syft
- Grype (vulnerability scanner) — https://github.com/anchore/grype
- OSV (Open Source Vulnerabilities) — https://osv.dev/
- VEX (Vulnerability Exploitability eXchange) — https://www.cisa.gov/topics/cyber-threats-and-advisories/sbom/vex
- OpenSSF Scorecard — https://github.com/ossf/scorecard
- SPDX-License-Identifier convention — https://spdx.dev/ids/
