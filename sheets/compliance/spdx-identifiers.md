# SPDX Identifiers & SBOM

Standardized license identifiers (SPDX) and Software Bill of Materials (SBOM) — syntax, source-file headers, package-manifest fields, the canonical 30+ identifiers, deprecated forms, license exceptions, SBOM formats (SPDX 2.3, SPDX 3.0, CycloneDX), generation and policy tools, REUSE compliance, and the broken-then-fixed gotchas that bite real projects.

## Setup

SPDX = **Software Package Data Exchange**. A Linux Foundation working group standard, started in 2010, published as ISO/IEC 5962:2021 (covering SPDX 2.2.1). SPDX 2.3 is the current widely-deployed spec; SPDX 3.0 (released 2024) is the next-generation profile-based redesign.

```text
Why standardized identifiers matter
-----------------------------------
- Free-form license names are AMBIGUOUS:
    "BSD" → which BSD? 2-clause? 3-clause? 4-clause? 0BSD?
    "GPL" → v1? v2? v3? "or later"? "only"?
    "Apache" → 1.0? 1.1? 2.0?
- Tools cannot reliably scan, audit, or enforce policy
  on free-form text. Identifiers are MACHINE-READABLE.
- SPDX provides ONE canonical short string per license,
  drawn from spdx.org/licenses (~600 entries as of 2026).
- One short header replaces 30 lines of legal boilerplate
  at the top of every source file.
```

The two SPDX generations you will encounter:

```text
SPDX 2.3 (current, widely deployed)
-----------------------------------
- Tag-value, JSON, YAML, RDF/XML, XLSX serializations
- Document/Package/File/Snippet/Relationship model
- Used by syft, cosign, GitHub dependency graph, etc.
- ISO/IEC 5962:2021 covers the closely-related 2.2.1
- THE format you generate today

SPDX 3.0 (released 2024, adoption ramping)
------------------------------------------
- Profile-based: Core, Software, Build, AI,
  Dataset, Security, Service, Lite
- JSON-LD canonical serialization
- AI/ML provenance (model weights, training data)
- Build attestation (SLSA integration)
- Backward compatibility via translation tooling
- Will become the long-term default
```

The North Star: **every SPDX identifier expression syntax + SBOM format + tool with cause + fix** should be in this sheet so the user never opens a browser to look up a license code, an SBOM field, or a CI policy gate.

## SPDX Identifier Concept

An SPDX **identifier** is a short canonical string for a known license. The full list lives at `spdx.org/licenses` and ships as JSON in the [`spdx/license-list-data`](https://github.com/spdx/license-list-data) repository.

```text
What an identifier is NOT
-------------------------
- It is NOT the license text.
- It is NOT a URL to the license.
- It is NOT a SHA hash of the license text.
- It is NOT a free-form name.

What an identifier IS
---------------------
- A short, ASCII, case-sensitive, dot/hyphen-delimited
  string registered in the SPDX License List.
- Stable across SPDX List versions (deprecations are
  flagged but identifiers are never reused).
- Resolvable to:
    - the canonical license text,
    - an OSI-approved flag,
    - an FSF-libre flag,
    - a "deprecated" flag,
    - an "exception" flag (for WITH clauses).
```

```bash
# Look up a license by identifier (offline copy)
curl -s https://spdx.org/licenses/MIT.json | jq .

# Listing every identifier
curl -s https://spdx.org/licenses/licenses.json | jq -r '.licenses[].licenseId' | sort

# Check whether a string is a valid identifier
curl -s https://spdx.org/licenses/licenses.json \
  | jq -r --arg id "Apache-2.0" \
    '.licenses[] | select(.licenseId==$id) | .name'
```

```text
The canonical short string vs ambiguous names
--------------------------------------------
Free-form          ->  SPDX identifier
"MIT License"      ->  MIT
"Apache 2.0"       ->  Apache-2.0
"GPLv3"            ->  GPL-3.0-only or GPL-3.0-or-later
"BSD"              ->  BSD-2-Clause or BSD-3-Clause or 0BSD or ...
"Mozilla Public"   ->  MPL-2.0
"CC0"              ->  CC0-1.0
"Public domain"    ->  Unlicense or CC0-1.0 (often both)
"WTFPL"            ->  WTFPL
"Boost"            ->  BSL-1.0
```

## SPDX Expression Syntax

The full grammar is RFC-style and lives in [the SPDX spec annex](https://spdx.github.io/spdx-spec/v2.3/SPDX-license-expressions/). The forms you will use 99% of the time:

```text
1) Simple identifier
--------------------
MIT
Apache-2.0
GPL-3.0-only
BSD-3-Clause

2) "or-later" — contributor accepts upgrade to later versions
-------------------------------------------------------------
GPL-3.0-or-later
LGPL-2.1-or-later
AGPL-3.0-or-later
GFDL-1.3-or-later

(Replaces the deprecated trailing-"+" form: "GPL-3.0+")

3) "only" — frozen to that exact version
----------------------------------------
GPL-3.0-only
LGPL-2.1-only
AGPL-3.0-only

(Replaces the deprecated bare "GPL-3.0", which was ambiguous
between -only and -or-later.)

4) Boolean AND — both licenses bind the recipient
-------------------------------------------------
MIT AND BSD-3-Clause
LGPL-2.1-only AND BSD-2-Clause

5) Boolean OR — recipient may pick either license
-------------------------------------------------
MIT OR Apache-2.0
GPL-2.0-or-later OR MIT
Apache-2.0 OR LGPL-3.0-or-later

6) WITH — license combined with a documented exception
------------------------------------------------------
GPL-3.0-or-later WITH GCC-exception-3.1
GPL-2.0-or-later WITH Classpath-exception-2.0
GPL-2.0-or-later WITH Linux-syscall-note
Apache-2.0 WITH LLVM-exception
GPL-3.0-or-later WITH Bison-exception-2.2

7) Parentheses — grouping for combined expressions
--------------------------------------------------
(MIT AND BSD-3-Clause) OR Apache-2.0
(GPL-3.0-or-later WITH GCC-exception-3.1) OR LicenseRef-Commercial
Apache-2.0 OR (MIT AND CC0-1.0)

8) LicenseRef — non-SPDX-list license referenced locally
--------------------------------------------------------
LicenseRef-MyCompanyEULA
LicenseRef-Proprietary
LicenseRef-FonteAlternativa

(Full text MUST be present in the project, typically
LICENSES/<LicenseRef-X>.txt under the REUSE convention.)

9) DocumentRef — license defined in another SPDX document
---------------------------------------------------------
DocumentRef-spdx-tool-1.2:LicenseRef-MIT-Style-1
```

Operator precedence (highest to lowest): `WITH`, `AND`, `OR`. Use parentheses if you doubt how a tool parses a mixed expression.

```text
Examples — read these out loud
------------------------------
"MIT OR Apache-2.0"
    -> Recipient may use under MIT, OR under Apache-2.0.
       Idiomatic Rust dual-license pattern.

"GPL-3.0-or-later WITH GCC-exception-3.1"
    -> Code is GPLv3+, but the GCC runtime exception
       lifts copyleft for compiled output.

"(MIT AND BSD-3-Clause) OR Apache-2.0"
    -> Recipient may take "MIT AND BSD-3-Clause"
       (both bind), OR alternatively Apache-2.0.

"MIT AND CC-BY-4.0"
    -> Code under MIT, docs under CC-BY-4.0,
       both apply to the package.
```

```bash
# Validate an expression locally with the official Python parser
pip install license-expression
python -c "from license_expression import get_spdx_licensing; \
  print(get_spdx_licensing().parse('GPL-3.0-or-later WITH GCC-exception-3.1', validate=True))"
```

## SPDX in Source Files

The recommended one-liner header replaces 30+ lines of legal boilerplate. The Linux kernel adopted this pattern in 2018 and migrated tens of thousands of files.

```c
// SPDX-License-Identifier: GPL-2.0-or-later WITH Linux-syscall-note
/*
 * net/ipv4/tcp_input.c
 * (rest of original comment block — short, no boilerplate)
 */
```

```text
Comment style by language
-------------------------
C, C++, Java, JavaScript, TypeScript, Go, Rust, Swift, Kotlin
    // SPDX-License-Identifier: MIT

Python, Ruby, Perl, Shell (sh, bash, zsh), R, Tcl, Make, YAML, TOML
    # SPDX-License-Identifier: MIT

HTML, XML, SVG, Markdown (HTML comment)
    <!-- SPDX-License-Identifier: MIT -->

CSS, SCSS
    /* SPDX-License-Identifier: MIT */

Lisp, Clojure, Scheme, Emacs Lisp
    ;; SPDX-License-Identifier: MIT

Erlang, Prolog
    %% SPDX-License-Identifier: MIT

Haskell, OCaml (block)
    {- SPDX-License-Identifier: MIT -}
    (* SPDX-License-Identifier: MIT *)

SQL
    -- SPDX-License-Identifier: MIT

VHDL
    -- SPDX-License-Identifier: MIT

Assembly (NASM/GAS)
    ; SPDX-License-Identifier: MIT
    # SPDX-License-Identifier: MIT
```

```python
# SPDX-License-Identifier: MIT
# SPDX-FileCopyrightText: 2026 Acme Corp <legal@acme.example>

"""acme.utils — small helpers used across the codebase."""
```

```go
// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Acme Corp <legal@acme.example>

package acme

// (file body)
```

```rust
// SPDX-License-Identifier: MIT OR Apache-2.0
// SPDX-FileCopyrightText: 2026 Acme Corp <legal@acme.example>

pub fn dual_licensed_function() {}
```

```text
Placement rules
---------------
- TOP of the file, ABOVE the docstring or copyright block.
- Within the FIRST 5 LINES so license scanners see it.
- Before any shebang? NO — shebang must be line 1:
    #!/usr/bin/env python3
    # SPDX-License-Identifier: MIT
    # SPDX-FileCopyrightText: ...

Why "above other headers": tools like reuse, scancode, askalono
fingerprint the first 1–10 KB of the file. A misplaced header
silently fails detection without erroring.
```

The companion REUSE tags you commonly pair with the identifier:

```text
SPDX-FileCopyrightText: 2026 Acme Corp <legal@acme.example>
SPDX-License-Identifier: MIT
SPDX-FileContributor: Bob Smith <bob@acme.example>
SPDX-PackageDownloadLocation: https://github.com/acme/widget
```

## SPDX in Package Manifests

The exact field varies by ecosystem. Use the SPDX expression in every ecosystem that supports it; for verbose XML formats the bare identifier is still preferred over a free-form name.

### Cargo.toml (Rust)

```toml
[package]
name = "widget"
version = "0.1.0"
edition = "2021"
license = "MIT OR Apache-2.0"        # idiomatic Rust dual-license

# Or, when using a non-SPDX license, point to a LICENSE file:
# license-file = "LICENSE-CUSTOM"

[lints]
# (see cargo-deny later in this sheet for policy enforcement)
```

### package.json (npm / pnpm / yarn)

```json
{
  "name": "widget",
  "version": "1.0.0",
  "license": "MIT"
}
```

```json
{
  "name": "widget",
  "version": "1.0.0",
  "license": "(MIT OR Apache-2.0)"
}
```

```json
{
  "name": "widget-private",
  "version": "0.1.0",
  "license": "UNLICENSED",
  "private": true
}
```

```text
npm legacy field (deprecated since 2017)
----------------------------------------
"license": { "type": "MIT", "url": "https://..." }
"licenses": [ { "type": "MIT", "url": "..." } ]

The new form is just a string SPDX expression.
```

### pyproject.toml (Python — PEP 621 + PEP 639)

```toml
# Pre-PEP-639 (Python packaging classic)
[project]
name = "widget"
version = "0.1.0"
license = { text = "MIT" }
# or: license = { file = "LICENSE" }

# PEP 639 (accepted, gradually adopted in setuptools/hatch/poetry)
[project]
license = "MIT"
license-files = ["LICENSE", "LICENSES/*.txt"]
```

### Gemspec (Ruby)

```ruby
Gem::Specification.new do |s|
  s.name     = 'widget'
  s.version  = '0.1.0'
  s.license  = 'MIT'
  # multiple licenses:
  # s.licenses = ['MIT', 'Apache-2.0']
end
```

### go.mod (Go)

```text
go.mod has NO license field by design.
Convention: a LICENSE (or COPYING) file at the module root.
Tooling: go-licenses scans go.mod + the embedded LICENSE
files to derive identifiers via fingerprinting.
```

```go
// SPDX-License-Identifier: BSD-3-Clause
// In every .go file. The LICENSE file at module root holds the text.
```

### composer.json (PHP)

```json
{
  "name": "acme/widget",
  "license": "MIT"
}
```

```json
{
  "name": "acme/widget",
  "license": ["MIT", "GPL-2.0-or-later"]
}
```

### pom.xml (Maven — verbose, NOT a bare SPDX expression)

```xml
<project>
  <licenses>
    <license>
      <name>MIT License</name>
      <url>https://opensource.org/licenses/MIT</url>
      <distribution>repo</distribution>
    </license>
  </licenses>
</project>
```

```text
Maven NOTE
----------
- The <name> field is free-form; use the SPDX
  identifier ("MIT", "Apache-2.0") to be friendly
  to tooling, even though Maven does not enforce it.
- license-maven-plugin can validate against an
  approved list or generate THIRD-PARTY.txt summaries.
```

### .nuspec (NuGet — .NET)

```xml
<package>
  <metadata>
    <id>Widget</id>
    <version>1.0.0</version>
    <license type="expression">MIT</license>
    <!-- or for non-SPDX: -->
    <!-- <license type="file">LICENSE.txt</license> -->
  </metadata>
</package>
```

### Other manifests

```text
Gradle (Kotlin DSL):
    pom { licenses { license { name = "MIT" } } }

CocoaPods (.podspec):
    s.license = { :type => 'MIT', :file => 'LICENSE' }

Swift Package Manager (Package.swift):
    No license field — convention is a LICENSE file.

OPAM (OCaml):
    license: "MIT"

Cabal (Haskell):
    license: BSD-3-Clause   -- accepts SPDX since Cabal 2.2

Conan (C/C++):
    license = "MIT"          # in conanfile.py

Helm Chart.yaml:
    annotations:
      artifacthub.io/license: MIT
```

```toml
# Cargo dual-license — the canonical Rust idiom
[package]
name        = "ferris"
version     = "0.1.0"
license     = "MIT OR Apache-2.0"
description = "An idiomatic dual-licensed Rust crate."
```

## Common SPDX IDs (the most-used 30+)

These are the identifiers that cover ~95% of open-source software in the wild. The descriptions are pragmatic, not legal advice.

```text
Permissive (do almost anything; preserve notice)
------------------------------------------------
MIT             — minimal, dominant in JS/Rust ecosystems
Apache-2.0      — patent grant, NOTICE file, dominant in Java/Go
BSD-2-Clause    — MIT-equivalent permissive
BSD-3-Clause    — adds non-endorsement clause
ISC             — BSD-2-Clause-equivalent, simpler text
0BSD            — BSD with attribution removed (effectively public domain)
Unlicense       — public-domain dedication, US-centric
CC0-1.0         — public-domain dedication, international
WTFPL           — informal public-domain
Zlib            — permissive with no-misrepresentation clause
BSL-1.0         — Boost Software License (permissive, no-attribution-on-binary)
Artistic-2.0    — Perl-original permissive variant
X11             — original X Window System license
PostgreSQL      — permissive, used by PostgreSQL only
Ruby            — dual: GPL-2.0 and a permissive license

Weak copyleft (library copyleft)
--------------------------------
LGPL-2.1-only       — frozen to v2.1, library copyleft
LGPL-2.1-or-later   — v2.1+ upgrade option
LGPL-3.0-only       — frozen to v3
LGPL-3.0-or-later   — v3+ upgrade option
MPL-2.0             — file-level copyleft, GPL-compatible
EPL-1.0             — Eclipse, weak copyleft, GPL-incompatible
EPL-2.0             — Eclipse, weak copyleft, GPL-compatible via secondary license
CDDL-1.0            — Sun-era, file-level copyleft (used by Solaris/ZFS)
CDDL-1.1            — minor revision of CDDL-1.0

Strong copyleft (whole-work copyleft)
-------------------------------------
GPL-2.0-only       — Linux kernel proper
GPL-2.0-or-later   — many GNU tools (with -or-later)
GPL-3.0-only       — frozen v3, anti-tivoization clause
GPL-3.0-or-later   — v3+, modern GNU default
AGPL-3.0-only      — adds network-use trigger
AGPL-3.0-or-later  — modern AGPL default

Documentation / content
-----------------------
GFDL-1.3-only        — GNU Free Documentation, frozen
GFDL-1.3-or-later    — GFDL with upgrade option
CC-BY-4.0            — Creative Commons Attribution
CC-BY-SA-4.0         — Attribution + ShareAlike (Wikipedia, OSM)
CC-BY-NC-4.0         — Attribution + NonCommercial — NOT for software
CC-BY-ND-4.0         — Attribution + NoDerivs — NOT for software

Older / niche but seen
----------------------
Apache-1.1           — pre-2.0 Apache, rarely used now
OpenSSL              — pre-Apache OpenSSL license (replaced in 3.0+)
Python-2.0           — CPython itself
PHP-3.01             — PHP itself
LPPL-1.3c            — LaTeX Project Public License
EUPL-1.2             — EU public license, copyleft, multilingual
ISC-OpenBSD          — sometimes used; usually just "ISC"
```

```text
Quick chooser
-------------
"I want maximum adoption" -------------- MIT
"I want patent grant + Java norms" ----- Apache-2.0
"I want both, like Rust crates" -------- MIT OR Apache-2.0
"Ensure improvements come back, lib" --- LGPL-3.0-or-later
"Ensure improvements come back, app" --- GPL-3.0-or-later
"...and SaaS" -------------------------- AGPL-3.0-or-later
"Public-domain dedication" ------------- CC0-1.0 (or Unlicense)
"Documentation" ------------------------ CC-BY-4.0 or CC-BY-SA-4.0
```

```bash
# Print details for any SPDX id (offline cache)
LIC=Apache-2.0
curl -s "https://spdx.org/licenses/${LIC}.json" | jq '
  { id: .licenseId,
    name: .name,
    osi: .isOsiApproved,
    fsf: .isFsfLibre,
    deprecated: .isDeprecatedLicenseId
  }'
```

## Deprecated / Renamed IDs

The SPDX list has gone through several deprecation rounds. Tools accept both old and new forms for backward compat, but SPDX 3.x and modern policy tooling prefer the new form. **Always migrate.**

```text
Deprecated form        Current canonical form(s)
---------------------  ----------------------------------------
GPL-1.0                GPL-1.0-only or GPL-1.0-or-later
GPL-1.0+               GPL-1.0-or-later
GPL-2.0                GPL-2.0-only or GPL-2.0-or-later
GPL-2.0+               GPL-2.0-or-later
GPL-3.0                GPL-3.0-only or GPL-3.0-or-later
GPL-3.0+               GPL-3.0-or-later

LGPL-2.0               LGPL-2.0-only or LGPL-2.0-or-later
LGPL-2.0+              LGPL-2.0-or-later
LGPL-2.1               LGPL-2.1-only or LGPL-2.1-or-later
LGPL-2.1+              LGPL-2.1-or-later
LGPL-3.0               LGPL-3.0-only or LGPL-3.0-or-later
LGPL-3.0+              LGPL-3.0-or-later

AGPL-1.0               AGPL-1.0-only or AGPL-1.0-or-later
AGPL-3.0               AGPL-3.0-only or AGPL-3.0-or-later

GFDL-1.1               GFDL-1.1-only or GFDL-1.1-or-later
GFDL-1.2               GFDL-1.2-only or GFDL-1.2-or-later
GFDL-1.3               GFDL-1.3-only or GFDL-1.3-or-later

eCos-2.0               (deprecated; rarely seen; no exact replacement)
StandardML-NJ          SMLNJ
wxWindows              wxWindows (kept; -with-exception variants exist)
Net-SNMP               Net-SNMP (kept)
Nokia                  Nokia (kept)
```

The big rule:

```text
- Bare "GPL-2.0", "GPL-3.0", "LGPL-3.0", "AGPL-3.0", etc.
  are AMBIGUOUS — they do not specify whether downstream
  users may upgrade to a later version.
- SPDX 3+ rejects the bare forms; tools fail validation.
- Always pick "-only" or "-or-later" explicitly.

The trailing-"+" form ("GPL-2.0+") is also deprecated; it
mapped to "-or-later" but used a punctuation character that
broke many parsers.
```

```bash
# Audit a repo for deprecated SPDX ids
grep -RnE "SPDX-License-Identifier:.*(GPL-2\.0|GPL-3\.0|LGPL-2\.1|LGPL-3\.0|AGPL-3\.0|GPL-2\.0\+|GPL-3\.0\+)([^-]|$)" .
```

## Source-Available / Non-FOSS IDs

These licenses are NOT OSI-approved (some are not even FSF-libre) but appear frequently in the wild. Some are on the SPDX list; others must be expressed as `LicenseRef-X`.

```text
Identifier              Status                       Notes
----------------------  ---------------------------  ------------------------------------
BUSL-1.1                On SPDX list                 Business Source License — converts to
                                                      open-source (usually Apache-2.0)
                                                      after change-date (typically 4 years).
                                                      Adopted by HashiCorp Terraform/Vault,
                                                      Sentry, MariaDB MaxScale, CockroachDB.

BSL-1.1                 NOT on SPDX list             Sometimes wrongly used for the
                                                      Business Source License. Use BUSL-1.1.
                                                      "BSL-1.0" on the list is the BOOST
                                                      Software License — totally unrelated.

SSPL-1.0                On SPDX list                 Server Side Public License (MongoDB).
                                                      OSI rejected it. AGPL-with-extras.

Elastic-2.0             On SPDX list                 Elastic License v2 (Elastic, Kibana).
                                                      Source-available, not OSI.

Confluent-Community-1.0 On SPDX list as-of recent    Apache+restrictions.
                        revisions

FSL-1.1                 NOT on SPDX list             Functional Source License (Sentry,
                                                      Bismuth). Permissive after 2 years.
                                                      Variants: FSL-1.1-MIT,
                                                      FSL-1.1-ALv2 — express via
                                                      LicenseRef-FSL-1.1-MIT.

Commons-Clause          Patch on top of OSI license  Restricts commercial sale. Express as:
                                                      "Apache-2.0 WITH Commons-Clause"?
                                                      NO — Commons-Clause is NOT a SPDX
                                                      exception. Use LicenseRef-CommonsClause.

PolyForm-Noncommercial-1.0.0    On SPDX list (2024)  PolyForm family.
PolyForm-Small-Business-1.0.0   On SPDX list
PolyForm-Free-Trial-1.0.0       On SPDX list

CAL-1.0                 On SPDX list                 Cryptographic Autonomy License — OSI-approved
                                                     but with attribution-of-data clauses.
```

```text
Practical rule of thumb
-----------------------
- If a project is "source-available, not open-source",
  you generally CANNOT redistribute, run-as-a-service,
  or modify-and-relicense without specific permission.
- Get a vendor-supplied policy before depending on
  these in production.
- cargo-deny, license-checker, etc. should DENY by default
  and require explicit allow-listing.
```

## Custom Licenses — LicenseRef

If your project uses a license that's not in the SPDX list, name it `LicenseRef-<short-name>` and ship the full text alongside.

```text
LicenseRef-MyCompanyEULA            # internal proprietary
LicenseRef-Proprietary              # vague but common
LicenseRef-CommonsClause            # the Commons Clause restriction
LicenseRef-FSL-1.1-MIT              # Functional Source License (MIT-converting)
LicenseRef-FSL-1.1-ALv2             # FSL converting to Apache-2.0
LicenseRef-Sentry-FSL               # vendor-named
DocumentRef-spdx-tool-1.2:LicenseRef-MIT-Style-1   # cross-document
```

The REUSE convention places the full text in `LICENSES/`:

```text
LICENSES/
├── MIT.txt                          # SPDX-listed: text downloaded from spdx.org
├── Apache-2.0.txt
├── LicenseRef-MyCompanyEULA.txt     # custom: full English text
└── LicenseRef-FSL-1.1-MIT.txt
```

```text
LicenseRef naming rules
-----------------------
- Prefix: literal "LicenseRef-".
- Body:   alphanumeric, hyphens, dots only.
- No spaces. No underscores. No slashes.
- CASE-SENSITIVE (LicenseRef-Foo != LicenseRef-foo,
  but downstream tools commonly normalize — pick one casing).

Common mistakes:
    LicenseRef-My_Company_EULA   # underscores: rejected by some parsers
    LicenseRef MyCompanyEULA     # space: invalid
    licenseref-foo               # wrong prefix case

Invalid:                          Valid replacement:
LicenseRef-MyCo EULA              LicenseRef-MyCo-EULA
LicenseRef-FSL-1.1-MIT (BLANKED)  LicenseRef-FSL-1.1-MIT
licenseRef-foo                    LicenseRef-foo
```

## License Exceptions — WITH

The `WITH` clause attaches a documented exception to a license. Exceptions are themselves identifiers in the SPDX exceptions list (see `spdx.org/licenses/exceptions-index.html`).

```text
The most-used exceptions
------------------------
Classpath-exception-2.0   GPL-2.0-or-later WITH Classpath-exception-2.0
                          - OpenJDK; lifts copyleft for "linked code"
                          - lets proprietary apps link OpenJDK-licensed
                            class files without becoming GPL.

GCC-exception-3.1         GPL-3.0-or-later WITH GCC-exception-3.1
                          - GCC's runtime library (libgcc, libstdc++)
                          - allows you to ship binaries compiled with
                            GCC under your own license.

Linux-syscall-note        GPL-2.0-only WITH Linux-syscall-note
                          - Linux kernel ABI
                          - userspace using kernel syscalls is NOT
                            considered a derivative work.

LLVM-exception            Apache-2.0 WITH LLVM-exception
                          - LLVM, Clang
                          - patent-related clarification on top of
                            Apache-2.0.

Bison-exception-2.2       GPL-3.0-or-later WITH Bison-exception-2.2
                          - GNU Bison
                          - allows generated parsers to be relicensed.

Autoconf-exception-2.0    GPL-2.0-or-later WITH Autoconf-exception-2.0
Autoconf-exception-3.0    GPL-3.0-or-later WITH Autoconf-exception-3.0
                          - GNU Autoconf
                          - generated configure scripts unrestricted.

eCos-exception-2.0        GPL-2.0-or-later WITH eCos-exception-2.0
                          - eCos RTOS

Bootloader-exception      GPL-2.0-or-later WITH Bootloader-exception
                          - PyInstaller-style bootloader; lets generated
                            binaries embed proprietary apps.

Font-exception-2.0        GPL-3.0-or-later WITH Font-exception-2.0
                          - GNU FreeFont
                          - documents are not derivative works.

OpenSSL-exception         GPL-2.0-or-later WITH OpenSSL-exception
                          - older codebases; pre-Apache-OpenSSL transition.

OCaml-LGPL-linking-exception  LGPL-2.1-or-later WITH OCaml-LGPL-linking-exception
                              - OCaml stdlib

WxWindows-exception-3.1   LGPL-2.0-or-later WITH WxWindows-exception-3.1
                          - wxWidgets
                          - allows distribution of derivatives in any form.

Mif-exception             GPL-2.0-or-later WITH Mif-exception
                          - rare
```

```text
What WITH does NOT allow
------------------------
- "Apache-2.0 WITH Classpath-exception-2.0" — INVALID.
  Classpath exception is defined for GPL only.
- "MIT WITH GCC-exception-3.1" — INVALID.
  MIT is permissive; the exception has no meaning.

The WITH operator is only valid for combinations that the
SPDX list explicitly registers. Some tools warn; others
silently accept (DON'T trust silent acceptance).
```

```bash
# Listing all SPDX exceptions
curl -s https://spdx.org/licenses/exceptions.json \
  | jq -r '.exceptions[] | .licenseExceptionId' | sort
```

## SBOM Concept

An **SBOM (Software Bill of Materials)** is the "ingredients list" for a software product. Every component, version, supplier, dependency, license, and (often) known vulnerabilities — captured as a machine-readable document.

```text
What an SBOM answers
--------------------
1. WHAT does this software contain?
2. WHO made each component?
3. WHICH version is each component?
4. WHO supplied each component (provenance)?
5. WHAT licenses apply to each component?
6. WHAT depends on what (the dependency graph)?
7. WHERE was each component obtained?
8. WHICH known vulnerabilities affect components?
   (Some SBOM formats include CVE refs; others rely
    on a separate VEX document.)

The drivers behind the mandate
------------------------------
- SolarWinds compromise (2020) → governments lost
  visibility into what shipped inside vendor binaries.
- Log4Shell (2021) → operators could not quickly
  enumerate where Log4j was deployed.
- Executive Order 14028 (US, May 2021) → federal
  vendors must provide SBOMs.
- NTIA SBOM Minimum Elements (2021).
- ISO/IEC 5962:2021 codifies SPDX 2.2.1.
- EU Cyber Resilience Act (CRA, 2024) extends the
  mandate to anything sold in the EU with a digital
  component.
- US OMB M-22-18 (2022) operationalizes EO 14028.
```

```text
SBOM consumer scenarios
-----------------------
- "Is Log4j 2.0–2.16 in any of my deployed services?"
  -> grep SBOMs.
- "Has the supplier's signing key been rotated?"
  -> verify SBOM signature against expected key.
- "Does this image include AGPL-licensed code that
  would trigger our policy?"
  -> license-policy lint over SBOM.
- "Has the build process been tampered with?"
  -> compare SBOM against in-toto attestations.
```

## SPDX SBOM Format

SPDX 2.3 is the current widely-deployed SBOM format and the standard target for compliance tools. ISO/IEC 5962:2021 standardizes the close-cousin SPDX 2.2.1.

```text
SPDX 2.3 serializations
-----------------------
- JSON       (most common today)
- YAML       (human-readable)
- RDF/XML    (semantic-web purists)
- Tag-Value  (.spdx file extension; original line-oriented)
- XLSX       (humans only)

Document-level required metadata
--------------------------------
spdxVersion         : "SPDX-2.3"
dataLicense         : "CC0-1.0"
SPDXID              : "SPDXRef-DOCUMENT"
name                : human name
documentNamespace   : a UNIQUE URI per document
                      (do NOT reuse; tools key on this)
creationInfo:
    created         : ISO-8601 timestamp
    creators        : [ "Tool: syft-1.20", "Person: Stevie", ... ]
    licenseListVersion : version of the SPDX List used

Package metadata
----------------
SPDXID              : SPDXRef-Package-foo-1.0.0
name                : foo
versionInfo         : 1.0.0
supplier            : "Organization: Acme <https://acme.example>"
originator          : "Organization: Upstream <https://upstream.example>"
downloadLocation    : URL or NOASSERTION
filesAnalyzed       : true|false
checksums           : [{ algorithm, checksumValue }]
licenseConcluded    : SPDX expression
licenseDeclared     : SPDX expression (from package metadata)
licenseInfoFromFiles: [ ... ]
copyrightText       : free-form copyright notice
primaryPackagePurpose: APPLICATION | LIBRARY | CONTAINER | ...
externalRefs        : [ PURL, CPE, Git, ... ]

File metadata
-------------
SPDXID              : SPDXRef-File-...
fileName            : relative path
checksums           : SHA1 mandatory; SHA256 recommended
licenseConcluded    : SPDX expression or NOASSERTION
licenseInfoInFiles  : [ identifiers detected in the file ]
copyrightText       : free-form

Relationships
-------------
DEPENDS_ON          : a depends on b
CONTAINS            : container holds package
DESCRIBES           : document describes a package
DEV_DEPENDENCY_OF   : dev-only dependency
BUILD_DEPENDENCY_OF : build-time only
TEST_DEPENDENCY_OF  : test-only
RUNTIME_DEPENDENCY_OF
PREREQUISITE_FOR
PATCH_FOR
GENERATED_FROM
ANCESTOR_OF, DESCENDANT_OF, VARIANT_OF
COPY_OF, FILE_ADDED, FILE_MODIFIED, FILE_DELETED
```

A minimal valid SPDX 2.3 JSON:

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "widget-1.0.0",
  "documentNamespace": "https://acme.example/spdx/widget-1.0.0-7c1d3a",
  "creationInfo": {
    "created": "2026-04-25T08:30:00Z",
    "creators": ["Tool: syft-1.21.0", "Organization: Acme Corp"],
    "licenseListVersion": "3.24"
  },
  "packages": [
    {
      "SPDXID": "SPDXRef-Package-widget",
      "name": "widget",
      "versionInfo": "1.0.0",
      "downloadLocation": "https://github.com/acme/widget/releases/tag/v1.0.0",
      "filesAnalyzed": false,
      "supplier": "Organization: Acme Corp <legal@acme.example>",
      "licenseConcluded": "MIT OR Apache-2.0",
      "licenseDeclared": "MIT OR Apache-2.0",
      "copyrightText": "Copyright 2026 Acme Corp",
      "primaryPackagePurpose": "APPLICATION",
      "externalRefs": [
        {
          "referenceCategory": "PACKAGE-MANAGER",
          "referenceType": "purl",
          "referenceLocator": "pkg:github/acme/widget@1.0.0"
        }
      ]
    }
  ],
  "relationships": [
    {
      "spdxElementId": "SPDXRef-DOCUMENT",
      "relatedSpdxElement": "SPDXRef-Package-widget",
      "relationshipType": "DESCRIBES"
    }
  ]
}
```

The same in tag-value (`widget-1.0.0.spdx`):

```text
SPDXVersion: SPDX-2.3
DataLicense: CC0-1.0
SPDXID: SPDXRef-DOCUMENT
DocumentName: widget-1.0.0
DocumentNamespace: https://acme.example/spdx/widget-1.0.0-7c1d3a
Creator: Tool: syft-1.21.0
Creator: Organization: Acme Corp
Created: 2026-04-25T08:30:00Z
LicenseListVersion: 3.24

PackageName: widget
SPDXID: SPDXRef-Package-widget
PackageVersion: 1.0.0
PackageDownloadLocation: https://github.com/acme/widget/releases/tag/v1.0.0
FilesAnalyzed: false
PackageSupplier: Organization: Acme Corp <legal@acme.example>
PackageLicenseConcluded: MIT OR Apache-2.0
PackageLicenseDeclared: MIT OR Apache-2.0
PackageCopyrightText: <text>Copyright 2026 Acme Corp</text>
PrimaryPackagePurpose: APPLICATION
ExternalRef: PACKAGE-MANAGER purl pkg:github/acme/widget@1.0.0

Relationship: SPDXRef-DOCUMENT DESCRIBES SPDXRef-Package-widget
```

## CycloneDX — alternative SBOM format

CycloneDX is OWASP-led, security-leaning, and very common in containers and dependency-scanning pipelines. Many tools (syft, Microsoft sbom-tool, Trivy) emit BOTH SPDX and CycloneDX from the same input.

```text
Serializations
--------------
- JSON  (canonical, most common)
- XML
- Protocol Buffers (binary)

Top-level structure
-------------------
bomFormat            : "CycloneDX"
specVersion          : "1.5" or "1.6"
serialNumber         : "urn:uuid:..."
version              : 1, 2, ... (BOM revision)
metadata             : { component, tools, authors, ... }
components           : [ ... ]
services             : [ ... ]
dependencies         : [ ... ]
vulnerabilities      : [ ... ]
formulation          : [ build/test workflow ]
declarations         : [ attestations, claims ]
```

A minimal CycloneDX 1.5 JSON:

```json
{
  "bomFormat": "CycloneDX",
  "specVersion": "1.5",
  "serialNumber": "urn:uuid:7c1d3a2e-91e5-44a7-9e6b-7c63cd55ca4b",
  "version": 1,
  "metadata": {
    "timestamp": "2026-04-25T08:30:00Z",
    "tools": [{ "vendor": "Anchore", "name": "syft", "version": "1.21.0" }],
    "component": {
      "type": "application",
      "bom-ref": "pkg:github/acme/widget@1.0.0",
      "name": "widget",
      "version": "1.0.0",
      "licenses": [{ "expression": "MIT OR Apache-2.0" }]
    }
  },
  "components": [
    {
      "type": "library",
      "bom-ref": "pkg:cargo/serde@1.0.197",
      "name": "serde",
      "version": "1.0.197",
      "purl": "pkg:cargo/serde@1.0.197",
      "licenses": [{ "expression": "MIT OR Apache-2.0" }]
    }
  ],
  "dependencies": [
    {
      "ref": "pkg:github/acme/widget@1.0.0",
      "dependsOn": ["pkg:cargo/serde@1.0.197"]
    }
  ]
}
```

```text
SPDX vs CycloneDX — picking one
-------------------------------
Choose SPDX if:
- Your customer or regulator asks for SPDX.
- Your audit story is licenses + provenance.
- ISO/IEC 5962:2021 compliance is required.

Choose CycloneDX if:
- Your audit story is vulnerabilities (VEX, CycloneDX-VDR).
- You want first-class formulation/services support.
- Container vendors are your primary consumers.

Reality: ship both. syft does it for free.
```

## Tools — SBOM Generation

```text
syft (Anchore) — the de-facto standard
--------------------------------------
- Languages: Rust, Go, Java/Maven, Node, Python, Ruby, .NET, PHP, Erlang, Elixir, Haskell, OCaml, Swift...
- Sources: directory, archive, container image (Docker/OCI/podman), git repo, registry
- Outputs: spdx-json, spdx-tag-value, cyclonedx-json, cyclonedx-xml, syft-json, table, text

    syft dir:./                                       # scan directory
    syft .                                            # short form
    syft alpine:3.20 -o spdx-json > sbom.json         # container image
    syft alpine:3.20 -o cyclonedx-json > sbom.cdx.json
    syft registry:registry.acme.example/widget:1.0.0  # registry pull
    syft scan oci-archive:./widget.tar                # archive
    syft attest alpine:3.20 -o cyclonedx-json --key cosign.key  # signed attestation
```

```text
cdxgen (CycloneDX official, Node.js)
------------------------------------
- Strong multi-language coverage (Java, Node, Python, Rust, Go, .NET, PHP, Ruby, Dart, Crystal, Elixir, Kotlin, Scala...)
- Outputs CycloneDX (preferred) or SPDX
- Excellent for monorepos
    npm install -g @cyclonedx/cdxgen
    cdxgen -o bom.json
    cdxgen -t python -o bom.json
    cdxgen -t java --required-only -o bom.json
```

```text
cyclonedx-bom — Python -> CycloneDX
-----------------------------------
    pip install cyclonedx-bom
    cyclonedx-py environment -o sbom.json
    cyclonedx-py poetry -o sbom.json
    cyclonedx-py requirements -i requirements.txt -o sbom.json
```

```text
cargo-spdx — Rust -> SPDX
-------------------------
    cargo install cargo-spdx
    cargo spdx --output-format json > sbom.spdx.json
```

```text
cargo-cyclonedx — Rust -> CycloneDX
-----------------------------------
    cargo install cargo-cyclonedx
    cargo cyclonedx --format json --target-only --override-filename bom
```

```text
sbom-tool (Microsoft, multi-language)
-------------------------------------
- Generates SPDX 2.2 / 2.3
- C/C++, .NET, Node, Python, Java, Go
    sbom-tool generate \
      -b ./build \
      -bc ./ \
      -pn widget -pv 1.0.0 \
      -ps "Acme Corp" \
      -nsb https://acme.example
```

```text
spdx-sbom-generator (Linux Foundation, multi-language)
------------------------------------------------------
    spdx-sbom-generator -p . -o sbom-out/
    # produces sbom-out/<lang>.spdx
```

```text
bom (Kubernetes-flavor; built for k8s release flow)
---------------------------------------------------
    bom generate --format json --output bom.json --dirs ./
    bom validate -i bom.json
```

```text
tern — image-focused (Linux container)
--------------------------------------
    tern report -f spdxjson -i alpine:3.20 -o tern.spdx.json
    tern report -f cyclonedxjson -i alpine:3.20
```

```text
trivy — security scanner with SBOM emit
---------------------------------------
    trivy image --format spdx-json --output sbom.spdx.json alpine:3.20
    trivy fs --format cyclonedx --output bom.json ./
    trivy sbom sbom.spdx.json    # scan an existing SBOM for CVEs
```

```text
grype — companion to syft (Anchore vulnerability scanner)
---------------------------------------------------------
    grype sbom:./sbom.spdx.json
    grype dir:./
    grype alpine:3.20
```

## Tools — License Detection

These look at LICENSE files, headers, and source patterns to *infer* the SPDX identifier of a project.

```text
askalono (Rust)
---------------
- Fingerprint matching against the SPDX corpus.
- Used inside cargo-deny, syft, others.
    cargo install askalono-cli
    askalono identify LICENSE
    askalono crawl ./

licensee (Ruby)
---------------
- GitHub uses this to populate the "License" badge on repos.
- High-precision: refuses to guess on ambiguous LICENSE text.
    gem install licensee
    licensee detect ./
    licensee detect --json ./

go-license-detector (Go fork of licensee)
-----------------------------------------
    go install github.com/go-enry/go-license-detector/cmd/license-detector@latest
    license-detector ./

scancode-toolkit (Python, very thorough)
----------------------------------------
- Detects licenses, copyrights, and many other artifacts.
- Output: ScanCode JSON, plus optional SPDX and CycloneDX export.
    pip install scancode-toolkit
    scancode --json-pp scan.json --license --copyright --package ./

ORT (OSS Review Toolkit, JVM)
-----------------------------
- Heavy enterprise compliance pipeline.
- Analyzer + Scanner + Evaluator + Reporter.
- Outputs SPDX, CycloneDX, NOTICE files, custom HTML.
    ort analyze -i ./project -o ort-out
    ort scan -i ort-out/analyzer-result.yml -o ort-out
    ort evaluate -i ort-out/scanner-result.yml --rules-resource ./rules.kts
    ort report -i ort-out/evaluator-result.yml --report-formats SpdxDocument,WebApp

FOSSA / Snyk / Black Duck (commercial)
--------------------------------------
- Hosted scanning, policy management, ticket integration.
- Out of scope for this sheet; mention so reviewers don't
  expect the open tools to ship dashboards.
```

## Tools — License Policy Enforcement

These read your dependency graph (or SBOM) and FAIL CI if a disallowed license appears.

```text
cargo-deny (Rust)
-----------------
File: deny.toml
    [licenses]
    version = 2
    confidence-threshold = 0.93
    allow = [
      "MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause",
      "ISC", "0BSD", "Unicode-DFS-2016", "CC0-1.0",
      "Zlib", "MPL-2.0",
    ]
    exceptions = [
      { name = "ring", allow = ["OpenSSL"] },
    ]

    [bans]
    multiple-versions = "warn"
    deny = [{ name = "openssl", version = "*" }]

    cargo deny check
    cargo deny check licenses
    cargo deny check bans
    cargo deny check advisories
```

```text
license-checker (npm)
---------------------
    npm install -g license-checker
    license-checker --production --onlyAllow "MIT;Apache-2.0;BSD-3-Clause;ISC;0BSD"
    license-checker --excludePackages "internal-tool@1.2.3"
    license-checker --json > licenses.json
```

```text
license-checker-rseidelsohn (newer fork)
----------------------------------------
    npx license-checker-rseidelsohn --production --json
```

```text
pip-licenses (Python)
---------------------
    pip install pip-licenses
    pip-licenses --format=markdown
    pip-licenses --format=json --with-license-file
    pip-licenses --fail-on="GPL-3.0-only;AGPL-3.0-only"
    pip-licenses --allow-only="MIT;Apache-2.0;BSD-3-Clause;ISC"
```

```text
go-licenses (Google, Go)
------------------------
    go install github.com/google/go-licenses@latest
    go-licenses report ./...
    go-licenses check ./...                       \
      --allowed_licenses=MIT,Apache-2.0,BSD-3-Clause,ISC,MPL-2.0
    go-licenses save ./... --save_path=licenses/  # bundle texts
```

```text
license_finder (multi-language Ruby)
------------------------------------
    gem install license_finder
    license_finder
    license_finder approval add MIT --who=Stevie
    license_finder permitted_licenses add MIT Apache-2.0 BSD-3-Clause
```

```text
allstar / dependency-review-action (GitHub)
-------------------------------------------
- GitHub Action that flags new dependencies whose
  license is outside the allowed list:
    - uses: actions/dependency-review-action@v4
      with:
        allow-licenses: MIT, Apache-2.0, BSD-3-Clause, ISC, MPL-2.0
        deny-licenses: GPL-3.0-or-later, AGPL-3.0-or-later
```

The CI policy pattern:

```yaml
# .github/workflows/license-policy.yml
name: license-policy
on: [pull_request]
jobs:
  enforce:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: dtolnay/rust-toolchain@stable
      - name: cargo-deny
        run: |
          cargo install --locked cargo-deny
          cargo deny check
      - name: SBOM
        uses: anchore/sbom-action@v0
        with:
          format: spdx-json
          artifact-name: sbom.spdx.json
```

## SPDX 3.0 — what's coming

SPDX 3.0 is a clean redesign that splits the monolithic 2.x model into composable **profiles**.

```text
Profiles (modules)
------------------
- Core         — base classes (Element, Agent, Tool, Relationship)
- Software     — packages, files, snippets (the SBOM core)
- Build        — build invocations, parameters, env
- AI           — AI/ML models, training data, weights, modelCard
- Dataset      — dataset metadata, features, sensitivity
- Security     — vulnerabilities, advisories, VEX
- Service      — running services, endpoints
- Lite         — slimmed-down profile for embedded/IoT

Canonical serialization
-----------------------
- JSON-LD (linked data) is now canonical.
- A "context" file maps short keys to full URIs.
- Tag-Value, RDF/XML, plain JSON still supported.

Backward compatibility
----------------------
- 3.0 -> 2.3 lossy translation tools exist.
- 2.3 -> 3.0 lossless translation: every 2.3 doc maps
  cleanly into Software profile with Core wrapper.

AI/ML new fields (Software + AI profile)
----------------------------------------
- modelExplainability   - intended use, limitations
- trainingData          - reference to Dataset profile
- typeOfModel           - LLM, CNN, RL, ...
- foundationModel       - boolean
- modelCard             - link to model card document

Build attestation (Build profile)
---------------------------------
- builderTool           - the runner (e.g., GitHub Actions)
- buildId               - unique invocation id
- environment           - parameter set
- inputs / outputs      - SBOM-shaped references
- SLSA-compatible mapping
```

```text
Practical advice (2026)
-----------------------
- Generate SPDX 2.3 JSON today; tooling support is universal.
- Watch for tools (syft, sbom-tool, cdxgen) emitting 3.0 by default.
- For AI/ML projects, jump straight to 3.0 when possible —
  there is no clean way to express training data in 2.3.
```

## Standards & Compliance

```text
ISO/IEC 5962:2021
-----------------
- Codifies SPDX 2.2.1 as a formal international standard.
- Cited in EU and US public-sector procurement.
- Pay-to-read; the SPDX 2.3 spec is freely available
  and is a near-superset.

NIST SBOM Minimum Elements (NISTIR 2021)
----------------------------------------
- Required fields per package:
    * supplier name
    * component name
    * version
    * unique identifier (SPDXID, PURL, CPE...)
    * dependency relationships
    * author of SBOM data
    * timestamp
- Required SBOM-document fields:
    * timestamp (when generated)
    * automation support (machine-readable format)

NTIA SBOM document (2021)
-------------------------
- Approved formats: SPDX, CycloneDX, SWID
- Distribution: with software OR via portal
- Frequency: at every release, at minimum

US Executive Order 14028 (May 2021)
-----------------------------------
- Section 4(e): software vendors selling to federal
  agencies must provide an SBOM.
- OMB M-22-18 / M-23-16 operationalize this.

EU Cyber Resilience Act (CRA, 2024)
-----------------------------------
- Anything sold in the EU "with digital elements"
  must:
    * provide an SBOM,
    * disclose vulnerabilities,
    * support security updates for the product lifetime,
    * report incidents within 24/72 hours.
- Phased enforcement; full force expected ~2027.

ISO/IEC 18974 (in development)
------------------------------
- Open Source compliance program standard.
- Companion to OpenChain ISO/IEC 5230:2020.
- Defines what "good" license-compliance looks like
  at an organisational level.

OpenChain ISO/IEC 5230:2020
---------------------------
- Process spec for OSS license compliance programs.
- Self-certifiable.

CISA SBOM efforts (ongoing)
---------------------------
- Tooling, taxonomy (e.g., the SBOM-VEX glossary),
  consultation drops.

US M-22-18 + Self-Attestation (2022+)
-------------------------------------
- Vendors self-attest software development practices
  per NIST SP 800-218 (SSDF).
- SBOM is one of the supporting evidence categories.
```

## SBOM in CI/CD

The end-to-end pattern: build, generate, sign, publish, link.

```yaml
# .github/workflows/release.yml — minimal example
name: release
on:
  push:
    tags: ['v*.*.*']

jobs:
  build:
    runs-on: ubuntu-latest
    permissions:
      id-token: write    # cosign keyless signing
      contents: write    # release upload
    steps:
      - uses: actions/checkout@v4

      - name: Build
        run: make build

      - name: Generate SPDX SBOM
        uses: anchore/sbom-action@v0
        with:
          path: ./
          format: spdx-json
          output-file: sbom.spdx.json

      - name: Generate CycloneDX SBOM
        uses: anchore/sbom-action@v0
        with:
          path: ./
          format: cyclonedx-json
          output-file: sbom.cdx.json

      - name: Install cosign
        uses: sigstore/cosign-installer@v3

      - name: Sign SBOM
        run: |
          cosign sign-blob --yes \
            --bundle sbom.spdx.json.cosign.bundle \
            sbom.spdx.json
          cosign sign-blob --yes \
            --bundle sbom.cdx.json.cosign.bundle \
            sbom.cdx.json

      - name: Upload to release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            sbom.spdx.json
            sbom.spdx.json.cosign.bundle
            sbom.cdx.json
            sbom.cdx.json.cosign.bundle
```

```bash
# Manual end-to-end recipe
syft dir:./ -o spdx-json > sbom.spdx.json
cosign sign-blob --yes --bundle sbom.bundle sbom.spdx.json

# Verify (anyone, anywhere)
cosign verify-blob \
  --bundle sbom.bundle \
  --certificate-identity 'https://github.com/acme/widget/.github/workflows/release.yml@refs/tags/v1.0.0' \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  sbom.spdx.json
```

```bash
# Attaching SBOM to a container as an in-toto attestation
syft alpine:3.20 -o spdx-json > image.sbom.spdx.json

cosign attest --yes \
  --predicate image.sbom.spdx.json \
  --type spdxjson \
  acme.example/widget:1.0.0

# Pull SBOM back from registry
cosign download attestation acme.example/widget:1.0.0 \
  | jq -r '.payload' | base64 -d | jq '.predicate'
```

```text
The "SBOM-as-attestation" pattern
---------------------------------
- in-toto attestation = a JSON envelope wrapping a
  predicate (SBOM) signed for a specific subject (artifact).
- predicateType for SPDX:    https://spdx.dev/Document
- predicateType for CDX:     https://cyclonedx.org/bom
- statementType:             https://in-toto.io/Statement/v0.1
- Stored as an OCI artifact alongside the image.

Tooling chain
-------------
- syft / cdxgen / trivy   -> generate SBOM
- cosign attest           -> sign as in-toto attestation
- ORAS / cosign           -> push to OCI registry
- cosign verify-attestation -> validate SBOM provenance
- grype sbom: / trivy sbom -> scan for CVEs from SBOM
```

## SBOM Validation

```text
SPDX schema validation (offline)
--------------------------------
- JSON Schema lives in https://github.com/spdx/spdx-spec
    pip install pyspdx-tools
    pyspdx-tools validate -i sbom.spdx.json

- The official SPDX online validator:
    https://tools.spdx.org/app/validate/

CycloneDX schema validation
---------------------------
    npm install -g @cyclonedx/cyclonedx-cli
    cyclonedx-cli validate --input-file bom.cdx.json
    cyclonedx-cli validate --input-format json --input-version 1.5 --input-file bom.cdx.json

sbom-validator / sbom-utility
-----------------------------
    sbom-utility validate -i sbom.spdx.json
    sbom-utility query --from "[].licenseDeclared" -i sbom.spdx.json
```

```bash
# Quick custom completeness check with jq
jq -e '
  .spdxVersion == "SPDX-2.3"
  and (.documentNamespace | length > 0)
  and (.creationInfo.creators | length > 0)
  and (.packages | length > 0)
  and (all(.packages[]; .name and .versionInfo and .downloadLocation))
' sbom.spdx.json && echo OK

# Find packages missing license info
jq '.packages[] | select(.licenseConcluded == "NOASSERTION") | .name' sbom.spdx.json

# Count unique licenses in the SBOM
jq -r '.packages[].licenseConcluded' sbom.spdx.json | sort -u
```

## SPDX Document Structure (concrete JSON)

A more complete example showing every required and most-used optional fields:

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "widget-1.0.0-sbom",
  "documentNamespace": "https://acme.example/spdx/widget-1.0.0/7c1d3a2e",
  "creationInfo": {
    "created": "2026-04-25T08:30:00Z",
    "creators": [
      "Tool: syft-1.21.0",
      "Organization: Acme Corp"
    ],
    "licenseListVersion": "3.24"
  },
  "packages": [
    {
      "SPDXID": "SPDXRef-Package-widget",
      "name": "widget",
      "versionInfo": "1.0.0",
      "supplier": "Organization: Acme Corp <legal@acme.example>",
      "originator": "Organization: Acme Corp <legal@acme.example>",
      "downloadLocation": "https://github.com/acme/widget/releases/tag/v1.0.0",
      "filesAnalyzed": true,
      "verificationCode": {
        "packageVerificationCodeValue": "8e0f0a2b3c4d5e6f70819293a4b5c6d7e8f90123"
      },
      "checksums": [
        { "algorithm": "SHA1",   "checksumValue": "8e0f...0123" },
        { "algorithm": "SHA256", "checksumValue": "abcd...ef01" }
      ],
      "licenseConcluded": "MIT OR Apache-2.0",
      "licenseDeclared": "MIT OR Apache-2.0",
      "licenseInfoFromFiles": ["MIT", "Apache-2.0"],
      "copyrightText": "Copyright 2026 Acme Corp",
      "primaryPackagePurpose": "APPLICATION",
      "externalRefs": [
        {
          "referenceCategory": "PACKAGE-MANAGER",
          "referenceType": "purl",
          "referenceLocator": "pkg:github/acme/widget@1.0.0"
        },
        {
          "referenceCategory": "SECURITY",
          "referenceType": "cpe23Type",
          "referenceLocator": "cpe:2.3:a:acme:widget:1.0.0:*:*:*:*:*:*:*"
        }
      ]
    },
    {
      "SPDXID": "SPDXRef-Package-serde",
      "name": "serde",
      "versionInfo": "1.0.197",
      "supplier": "Person: David Tolnay <dtolnay@gmail.com>",
      "downloadLocation": "https://crates.io/api/v1/crates/serde/1.0.197/download",
      "filesAnalyzed": false,
      "licenseConcluded": "MIT OR Apache-2.0",
      "licenseDeclared": "MIT OR Apache-2.0",
      "primaryPackagePurpose": "LIBRARY",
      "externalRefs": [
        {
          "referenceCategory": "PACKAGE-MANAGER",
          "referenceType": "purl",
          "referenceLocator": "pkg:cargo/serde@1.0.197"
        }
      ]
    }
  ],
  "files": [
    {
      "SPDXID": "SPDXRef-File-main",
      "fileName": "./src/main.rs",
      "checksums": [
        { "algorithm": "SHA1",   "checksumValue": "...." },
        { "algorithm": "SHA256", "checksumValue": "...." }
      ],
      "licenseConcluded": "MIT OR Apache-2.0",
      "licenseInfoInFiles": ["MIT", "Apache-2.0"],
      "copyrightText": "Copyright 2026 Acme Corp"
    }
  ],
  "relationships": [
    {
      "spdxElementId": "SPDXRef-DOCUMENT",
      "relatedSpdxElement": "SPDXRef-Package-widget",
      "relationshipType": "DESCRIBES"
    },
    {
      "spdxElementId": "SPDXRef-Package-widget",
      "relatedSpdxElement": "SPDXRef-Package-serde",
      "relationshipType": "DEPENDS_ON"
    },
    {
      "spdxElementId": "SPDXRef-Package-widget",
      "relatedSpdxElement": "SPDXRef-File-main",
      "relationshipType": "CONTAINS"
    }
  ]
}
```

## Naming Conventions

```text
SPDXIDs
-------
- Pattern: SPDXRef-<alphanumeric/.-_>+
- Documents:    SPDXRef-DOCUMENT (literal)
- Packages:     SPDXRef-Package-<name>[-<version>]
- Files:        SPDXRef-File-<path-with-dashes>
- Snippets:     SPDXRef-Snippet-<id>
- LicenseRefs:  LicenseRef-<short-name>
- DocumentRefs: DocumentRef-<external-doc-id>:LicenseRef-<x>

Allowed chars in the local identifier portion:
    [A-Za-z0-9.\-_]+
NOT allowed:
    spaces, slashes, plus signs, parentheses

Document namespace
------------------
- MUST be globally unique per document.
- Pattern (recommended):
    https://<your-host>/spdx/<product>-<version>/<random-32hex>
- Do NOT reuse a namespace across builds; tools key on it.
- Do NOT include trailing fragments (#...).

Package version syntax
----------------------
- Free-form, but consumers expect:
    * semver:        1.2.3, 1.2.3-rc.1+build.42
    * PEP 440:       1.2.3.dev0, 1.2.3a1, 1.2.3+local
    * Maven:         1.2.3-SNAPSHOT
    * Debian:        1:1.2.3-2ubuntu1
- Always include the EXACT vendor-published version.

Supplier / Originator format
----------------------------
- Person:        "Person: Stevie Bellis <stevie@bellis.tech>"
- Organization:  "Organization: Acme Corp <https://acme.example>"
- Tool:          "Tool: syft-1.21.0"
- Or NOASSERTION

Checksum algorithms
-------------------
- SHA1                    (mandatory in SPDX 2.x for Files)
- SHA256                  (recommended)
- SHA512                  (high-assurance)
- MD5                     (allowed; legacy only)
- BLAKE2b-256, BLAKE3     (newer additions)

PURL (Package URL) — the universal package identifier
-----------------------------------------------------
    pkg:cargo/serde@1.0.197
    pkg:npm/lodash@4.17.21
    pkg:pypi/requests@2.31.0
    pkg:gem/rails@7.1.3
    pkg:maven/org.springframework/spring-core@6.1.4
    pkg:nuget/Newtonsoft.Json@13.0.3
    pkg:golang/github.com/spf13/cobra@v1.8.0
    pkg:github/acme/widget@1.0.0
    pkg:docker/library/alpine@3.20
    pkg:deb/debian/curl@7.88.1-10+deb12u5
    pkg:rpm/redhat/glibc@2.34-100.el9
```

## Common Errors

```text
Bad expression                          Error / Cause                Fix
--------------------------------------  ---------------------------  ------------------------------------------
"MIT/Apache-2.0"                        Slash is not an operator      "MIT OR Apache-2.0"

"BSD"                                   Unrecognized identifier       "BSD-2-Clause" or "BSD-3-Clause" (be specific)

"GPL"                                   Unrecognized; ambiguous       "GPL-3.0-or-later" (or specific version)

"GPL-3.0"                               Deprecated; ambiguous re upgrade  "GPL-3.0-only" or "GPL-3.0-or-later"

"GPL-3.0+"                              Deprecated trailing-+         "GPL-3.0-or-later"

"AGPL-3.0"                              Deprecated; ambiguous         "AGPL-3.0-only" or "AGPL-3.0-or-later"

"LGPL-3.0"                              Deprecated; ambiguous         "LGPL-3.0-only" or "LGPL-3.0-or-later"

"MIT, Apache-2.0"                       Comma is not an operator      "MIT OR Apache-2.0" or
                                                                       "(MIT OR Apache-2.0)"

"MIT and Apache-2.0"                    Lowercase operators rejected
                                          by strict parsers           "MIT AND Apache-2.0"

"Apache-2.0 WITH classpath"             classpath-exception-2.0 is
                                          for GPL only; tools warn    Use the right pair:
                                                                      "GPL-2.0-or-later WITH Classpath-exception-2.0"

"GPL-3.0 WITH GCC-Exception-3.1"        Identifier case wrong         "GPL-3.0-or-later WITH GCC-exception-3.1"

"SeeLicense"                            Free-form text                Use a real SPDX id or
                                                                      "LicenseRef-<short-name>"

"Proprietary"                           Not in SPDX list               "LicenseRef-Proprietary"

"NoLicense"                             Not a real value               "NOASSERTION"

"NONE"                                  In wrong field                Reserve for licenseInfoFromFiles when
                                                                      no licenses were found.

License declared MIT in package.json,   Tools flag the disagreement   Pick ONE source of truth (we recommend
SPDX-License-Identifier: Apache-2.0       and fail policy             the in-file header) and align metadata
in source files                                                       with it.

LicenseRef-X with no fulltext           Consumer cannot audit         Ship LICENSES/LicenseRef-X.txt with
                                                                      the full English text.

WITH clause without registered          Tool warns / rejects          Drop the WITH clause or use a registered
exception                                                             exception identifier.

SBOM missing documentNamespace          Schema validation fails        Generate a fresh URI per document.

SBOM missing creationInfo.created       Schema validation fails        Add an ISO-8601 timestamp.

Multiple licenses listed but no         Tools default to "AND"          Be explicit: "MIT AND Apache-2.0"
operator                                  but warn                      or "MIT OR Apache-2.0".

SPDX expression with leading/trailing   Some tools reject              Trim whitespace.
whitespace

Package with no version                 Schema fails                   Always include versionInfo (use
                                                                       "0.0.0-unknown" only as last resort).

Image SBOM emitted by syft from a       SBOM lacks build-time deps    Generate the SBOM from the SOURCE
final-stage image only                                                directory or build context, not just
                                                                      from the final layer.
```

## REUSE.software

REUSE is an FSFE-maintained convention that turns SPDX into a "100% machine-checkable" project lifestyle.

```text
The three REUSE rules
---------------------
1. Every file has copyright and licensing information,
   either in the file itself OR in a .license sidecar
   OR in REUSE.toml.
2. Every license used appears as full text in LICENSES/.
3. There is no other license information anywhere
   that contradicts (1) and (2).
```

```text
Repository layout
-----------------
.
├── .reuse/
│   └── dep5            (legacy; can be replaced by REUSE.toml)
├── REUSE.toml          (modern: bulk attribution rules)
├── LICENSES/
│   ├── MIT.txt
│   ├── Apache-2.0.txt
│   └── LicenseRef-MyCompanyEULA.txt
└── src/
    ├── main.rs         (has SPDX header in-file)
    └── logo.png        (binary; use logo.png.license sidecar
                         OR list in REUSE.toml)
```

```toml
# REUSE.toml — modern bulk-attribution config
version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2026 Acme Corp <legal@acme.example>"
SPDX-License-Identifier = "MIT OR Apache-2.0"

[[annotations]]
path = "assets/icons/**"
SPDX-FileCopyrightText = "2026 Acme Corp"
SPDX-License-Identifier = "CC-BY-4.0"

[[annotations]]
path = "vendor/openssl/**"
SPDX-FileCopyrightText = "1998-2025 The OpenSSL Project Authors"
SPDX-License-Identifier = "Apache-2.0"
```

```bash
# reuse-tool
pipx install reuse

reuse lint                          # full project audit
reuse lint --json
reuse spdx                          # emit SPDX 2.3 SBOM from REUSE metadata
reuse annotate \
  --copyright "2026 Acme Corp <legal@acme.example>" \
  --license "MIT OR Apache-2.0" \
  src/main.rs                       # add SPDX header to a file
reuse download MIT                  # fetch LICENSES/MIT.txt
reuse download --all                # fetch every license referenced in headers
reuse supported-licenses            # list everything reuse can detect
```

```text
The "100% REUSE-compliant" badge
--------------------------------
- Add to README:
    [![REUSE status](https://api.reuse.software/badge/github.com/acme/widget)](https://api.reuse.software/info/github.com/acme/widget)
- The api.reuse.software service runs `reuse lint`
  remotely against the repo; if it passes, the badge
  goes green.
```

## Integration Patterns

```text
GitHub
------
- Repo License sidebar populated by licensee.
- Dependency graph: parses package manifests and emits
  an SBOM at /repos/<owner>/<repo>/dependency-graph/sbom.
- Action: github/dependency-review-action enforces
  policy on pull requests.

GitLab
------
- License Compliance scanner (license_finder under the hood):
    include:
      - template: Security/License-Scanning.gitlab-ci.yml
- Outputs gl-license-scanning-report.json plus an SPDX export.
- Dependency Scanning produces CycloneDX BOM artifacts.

Bitbucket
---------
- Snyk pipe and FOSSA integrations are the standard route.
- Bitbucket itself has no native SBOM; rely on pipeline.

Container registries
--------------------
- Docker Hub, GHCR, Quay, Artifact Registry, ACR all
  support OCI 1.1 referrers — cosign attestations
  including SBOM ride alongside images.
- crane / oras mirror SBOMs across registries.

The "SBOM at every layer" pattern
---------------------------------
- Multi-stage Dockerfile:
    1. Build stage: produce build-time SBOM (compiler,
       headers, build deps).
    2. Final stage: produce runtime SBOM (just the
       runtime closure).
- Attach BOTH as separate attestations:
    cosign attest --predicate build.sbom.json   --type spdxjson  ...
    cosign attest --predicate runtime.sbom.json --type spdxjson  ...

Cosign keyless mode
-------------------
- Uses GitHub OIDC token to sign with a short-lived
  certificate from Sigstore Fulcio.
- Public log entry on Rekor.
- Verifier matches certificate-identity (the workflow URL).
- No long-lived secret to leak.
```

## Practical Workflow

A single end-to-end checklist for a new project:

```text
1. License the project
   - Pick the SPDX expression: "MIT", "Apache-2.0",
     "MIT OR Apache-2.0", "GPL-3.0-or-later", ...
   - Add the full text to LICENSES/<id>.txt:
       reuse download MIT Apache-2.0

2. Source-file headers
   - Every text source file gets:
       SPDX-FileCopyrightText: 2026 Acme Corp <...>
       SPDX-License-Identifier: <expression>
   - Use reuse annotate or an editor snippet.

3. Manifest alignment
   - Cargo.toml / package.json / pyproject.toml / etc.
     hold the SAME SPDX expression.
   - REUSE.toml or in-file headers cover binary assets.

4. CI: REUSE lint
   - Job: pipx install reuse && reuse lint
   - Fail PR on missing or contradictory metadata.

5. CI: license policy
   - cargo-deny / pip-licenses / license-checker / go-licenses
   - Allow-list permissive + LGPL/MPL/EPL as required.
   - Deny GPL/AGPL unless a section is explicitly carved out.

6. CI: SBOM generation
   - syft dir:./ -o spdx-json > sbom.spdx.json
   - syft dir:./ -o cyclonedx-json > sbom.cdx.json
   - Optionally tern / sbom-tool / cdxgen variants.

7. CI: SBOM signing
   - cosign sign-blob --yes --bundle sbom.bundle sbom.spdx.json
   - Or attach to image: cosign attest --predicate sbom.spdx.json ...

8. Release: publish artifacts
   - Upload SBOM + signature to GitHub Release / GCS / S3.
   - Push container with attestations to registry.
   - Reference the SBOM URL in CHANGELOG.

9. Vulnerability scan
   - grype sbom:./sbom.spdx.json --fail-on high
   - trivy sbom sbom.spdx.json --severity HIGH,CRITICAL

10. Publish
    - SBOM as a release asset.
    - SBOM as an OCI artifact attestation.
    - SBOM as part of THIRD-PARTY notices distribution.
```

## Common Gotchas

```text
1) "Combined" with slash
   broken:  license = "MIT/Apache-2.0"
   error:   parser fails on / (npm), or treats as freeform
   fixed:   license = "MIT OR Apache-2.0"

2) Unspecified BSD
   broken:  SPDX-License-Identifier: BSD
   error:   tool reports unknown identifier
   fixed:   SPDX-License-Identifier: BSD-3-Clause   # or BSD-2-Clause / 0BSD
            (audit the LICENSE file to be sure which variant)

3) Bare GPL
   broken:  license: "GPL"
   error:   ambiguous; SPDX 3+ rejects
   fixed:   license: "GPL-3.0-or-later"

4) Deprecated trailing +
   broken:  SPDX-License-Identifier: GPL-3.0+
   error:   accepted but flagged deprecated; rejected by
            strict SPDX 3 validators
   fixed:   SPDX-License-Identifier: GPL-3.0-or-later

5) Deprecated bare GPL-2.0
   broken:  SPDX-License-Identifier: GPL-2.0
   error:   ambiguous re upgrade rights
   fixed:   GPL-2.0-only           # for code that must stay v2 (Linux kernel)
   fixed:   GPL-2.0-or-later       # for typical GNU code

6) LicenseRef without text
   broken:  SPDX-License-Identifier: LicenseRef-MyEULA
            (no LICENSES/LicenseRef-MyEULA.txt)
   error:   reuse lint: "Missing license file"
   fixed:   ship LICENSES/LicenseRef-MyEULA.txt with full text.

7) Header disagrees with manifest
   broken:  package.json -> "license": "MIT"
            src/index.js -> SPDX-License-Identifier: Apache-2.0
   error:   license-checker emits both; auditor confused
   fixed:   align both to one expression. If dual-licensed,
            use "MIT OR Apache-2.0" in BOTH places.

8) Wrong WITH pairing
   broken:  Apache-2.0 WITH Classpath-exception-2.0
   error:   Classpath exception is defined for GPL only;
            tools warn or reject.
   fixed:   For Apache-2.0 patent extras, use:
            Apache-2.0 WITH LLVM-exception
            For OpenJDK linking, use:
            GPL-2.0-or-later WITH Classpath-exception-2.0

9) Forgot NOTICE for Apache-2.0
   broken:  Repackaged Apache-2.0 dependencies in your
            distributable, but no NOTICE file shipped.
   error:   Apache-2.0 §4(d) requires preserving NOTICE.
            Auditor flags. Some tools (e.g., go-licenses save)
            extract NOTICE files automatically.
   fixed:   Bundle THIRD-PARTY-NOTICES (or NOTICE) collected
            from each Apache-2.0 dep.

10) SBOM not signed
    broken:  Release publishes sbom.spdx.json with no signature.
    error:   Consumers cannot verify provenance; supply-chain
             attackers can swap the file.
    fixed:   cosign sign-blob --yes --bundle sbom.bundle sbom.spdx.json
             Publish both files. Document the verifier identity.

11) Manually edited SBOM
    broken:  Engineer hand-edits sbom.spdx.json after generation
             to fix a typo, then re-signs.
    error:   Original signature now invalid; consumer trust path
             relies on humans typing correctly.
    fixed:   Fix the SOURCE and regenerate the SBOM in CI.
             Treat the SBOM as a build artifact, never a
             source file.

12) Build-time deps missing
    broken:  syft scans the runtime container only; the
             SBOM omits the compiler, glibc-devel, headers.
    error:   CVE in build-time-only library escapes audit
             (e.g., a malicious build plugin).
    fixed:   Generate SBOMs at TWO layers:
             - the build stage (with build deps),
             - the runtime stage (without).
             Attach both as attestations.

13) Document namespace reuse
    broken:  Same documentNamespace reused across builds.
    error:   Consumers see two "different" SBOMs claiming
             to be authoritative; deduplication breaks.
    fixed:   Mint a fresh URI per build. Embed git SHA or
             timestamp:
               https://acme.example/spdx/widget/1.0.0/<sha>

14) Dual-license with AND when meant OR
    broken:  package metadata: "MIT AND Apache-2.0"
    intent:  let the consumer pick either license.
    error:   AND means both bind the user; consumers must
             ALSO satisfy the more restrictive license.
    fixed:   "MIT OR Apache-2.0"  (the Rust idiom)

15) Wrong case of identifier
    broken:  SPDX-License-Identifier: apache-2.0
    error:   Identifiers are case-sensitive; some tools
             quietly normalize, others reject.
    fixed:   Apache-2.0 (capital A; case as registered).

16) Embedded GPL inside permissive distribution
    broken:  Project licensed MIT pulls in a GPL-3.0-or-later
             library transitively; ship as a single binary.
    error:   Distributing the combined work under MIT alone
             is a license violation.
    fixed:   Either replace the GPL dep, or relicense the
             whole binary distribution under
             "GPL-3.0-or-later" (the strongest copyleft wins).

17) WTFPL / unlicense in regulated environments
    broken:  WTFPL is not OSI-approved; some enterprise
             policies reject anything not OSI-approved.
    error:   procurement pipeline blocks adoption.
    fixed:   Switch upstream to MIT or 0BSD; both are
             effectively as permissive but OSI-approved.

18) PURL omitted from SBOM
    broken:  packages have only "name" and "versionInfo";
             no externalRefs with PURL.
    error:   Vulnerability scanners (grype, trivy) cannot
             match against advisory feeds reliably.
    fixed:   Always emit PURL via externalRefs (syft does
             this by default).

19) Missing files SBOM with filesAnalyzed=true
    broken:  Package declares filesAnalyzed: true but
             "files" array empty.
    error:   Schema validators reject; verificationCode
             cannot be computed.
    fixed:   Either set filesAnalyzed: false, OR include
             files with checksums.

20) Confusing BSL (Boost) with BUSL (Business Source)
    broken:  license = "BSL-1.1"
    intent:  Business Source License 1.1 (HashiCorp).
    error:   "BSL-1.0" is BOOST; "BSL-1.1" is not on the
             SPDX list. Tools fail or pick the wrong one.
    fixed:   license = "BUSL-1.1"     # for Business Source
             license = "BSL-1.0"      # for Boost
```

## Idioms

```text
- "Use SPDX-License-Identifier in every source file."
- "Pick a license from the SPDX list — don't invent one."
- "Prefer 'or-later' for forward-compatibility unless
  there's a specific reason to freeze the version."
- "For Rust crates: `MIT OR Apache-2.0`. Period."
- "Generate the SBOM in CI, not by hand."
- "Sign every SBOM with cosign — keyless if you can."
- "Use REUSE.software to keep license metadata coherent."
- "Treat the SBOM as a build artifact: never edit by hand."
- "Run cargo-deny / license-checker / go-licenses in CI."
- "Always include PURL in SBOM externalRefs — that's how
  scanners match CVEs."
- "Mint a fresh documentNamespace per build."
- "Generate SPDX AND CycloneDX; consumers vary."
- "Attach SBOM to images as in-toto attestations, not just
  as files in the release."
- "Watch for SPDX 3.0 — start emitting AI-profile fields
  for ML projects."
```

## See Also

- `verify` — math verification harness used by `cs detail` pages
- `license-decoder` (forthcoming) — paste-license-text → SPDX-id helper
- `cla-vs-dco` (forthcoming) — Contributor License Agreement vs Developer Certificate of Origin patterns
- `gdpr` — companion EU data-protection cheatsheet
- `ccpa` (forthcoming) — California Consumer Privacy Act
- `dora` — EU Digital Operational Resilience Act (companion compliance sheet)
- `eu-ai-act` — companion EU AI regulation sheet
- `pqc` — post-quantum cryptography (companion currency-gap sheet)
- `cosign` (security/) — Sigstore signing tool used to sign SBOMs

## References

- SPDX home: https://spdx.dev/
- SPDX License List: https://spdx.org/licenses/
- SPDX Exceptions List: https://spdx.org/licenses/exceptions-index.html
- SPDX 2.3 Specification: https://spdx.github.io/spdx-spec/v2.3/
- SPDX 3.0 Specification: https://spdx.github.io/spdx-3-model/
- SPDX License Expression syntax: https://spdx.github.io/spdx-spec/v2.3/SPDX-license-expressions/
- spdx/spdx-spec repo: https://github.com/spdx/spdx-spec
- spdx/license-list-data: https://github.com/spdx/license-list-data
- pyspdx-tools: https://github.com/spdx/tools-python
- ISO/IEC 5962:2021: https://www.iso.org/standard/81870.html
- NTIA SBOM Minimum Elements (2021): https://www.ntia.gov/sbom
- Executive Order 14028 (US, 2021): https://www.whitehouse.gov/briefing-room/presidential-actions/2021/05/12/executive-order-on-improving-the-nations-cybersecurity/
- OMB M-22-18 / M-23-16 (US): https://www.whitehouse.gov/omb/management/ofcio/
- EU Cyber Resilience Act (CRA): https://digital-strategy.ec.europa.eu/en/library/cyber-resilience-act
- OpenChain ISO/IEC 5230:2020: https://www.openchainproject.org/
- REUSE.software: https://reuse.software/
- REUSE specification: https://reuse.software/spec/
- Anchore syft: https://github.com/anchore/syft
- Anchore grype: https://github.com/anchore/grype
- Aqua Trivy: https://github.com/aquasecurity/trivy
- CycloneDX: https://cyclonedx.org/
- CycloneDX cdxgen: https://github.com/CycloneDX/cdxgen
- CycloneDX cyclonedx-cli: https://github.com/CycloneDX/cyclonedx-cli
- cyclonedx-bom (Python): https://github.com/CycloneDX/cyclonedx-python
- Microsoft sbom-tool: https://github.com/microsoft/sbom-tool
- LF spdx-sbom-generator: https://github.com/opensbom-generator/spdx-sbom-generator
- Kubernetes bom: https://github.com/kubernetes-sigs/bom
- tern: https://github.com/tern-tools/tern
- cargo-deny: https://github.com/EmbarkStudios/cargo-deny
- cargo-spdx: https://github.com/mtkennerly/cargo-spdx
- cargo-cyclonedx: https://github.com/CycloneDX/cyclonedx-rust-cargo
- npm license-checker: https://github.com/davglass/license-checker
- pip-licenses: https://github.com/raimon49/pip-licenses
- go-licenses (Google): https://github.com/google/go-licenses
- license_finder: https://github.com/pivotal/LicenseFinder
- askalono: https://github.com/jpeddicord/askalono
- licensee: https://github.com/licensee/licensee
- go-license-detector: https://github.com/go-enry/go-license-detector
- scancode-toolkit: https://github.com/nexB/scancode-toolkit
- ORT (OSS Review Toolkit): https://github.com/oss-review-toolkit/ort
- Sigstore cosign: https://github.com/sigstore/cosign
- Sigstore Fulcio: https://github.com/sigstore/fulcio
- Sigstore Rekor: https://github.com/sigstore/rekor
- in-toto attestation: https://github.com/in-toto/attestation
- SLSA framework: https://slsa.dev/
- PURL spec: https://github.com/package-url/purl-spec
- CPE 2.3 dictionary: https://nvd.nist.gov/products/cpe
- GitHub dependency-review-action: https://github.com/actions/dependency-review-action
- CISA SBOM resources: https://www.cisa.gov/sbom
