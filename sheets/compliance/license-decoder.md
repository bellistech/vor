# License Decoder

Every common open-source / source-available software license decoded with copyleft level, obligations, patent posture, and compatibility — so you can pick, audit, and combine licenses without leaving the terminal.

## Setup

A **software license** is a copyright-holder's grant of permissions on top of otherwise-default copyright restrictions. Without a license, every line of source code is, in copyright-respecting jurisdictions, "all rights reserved" — no copying, no modification, no redistribution, no public performance, no derivative works.

**Copyright vs license** — these are different layers:

- **Copyright** is the underlying property right. It vests automatically the moment a fixed creative expression is recorded. The author owns it. In most jurisdictions copyright lasts for the author's life plus 70 years.
- **License** is a permission slip granted by the copyright holder allowing some subset of others to do things copyright would otherwise forbid. Licenses can be exclusive or non-exclusive, perpetual or time-limited, royalty-free or paid, broad or narrow.
- A **patent** is a different right entirely (claims on inventions, not expressions). Some FOSS licenses also grant patent rights; many do not.
- A **trademark** is yet another right (claims on names and brands). Most FOSS licenses are silent on trademarks (or expressly disclaim).

```text
        Copyright (automatic)
              |
              v
+-----------------------------+
|   License (explicit grant)  |   <-- without this, all rights reserved
+-----------------------------+
   |             |          |
   v             v          v
copy/modify   patent     trademark
                grant?     grant?
```

**Open Source vs Free Software vs Source-Available** — three overlapping but distinct definitions:

- **Open Source** — the **OSI (Open Source Initiative)** maintains the **Open Source Definition (OSD)** with 10 criteria. A license is "open source" only if it meets the OSD: free redistribution, source code available, allows derived works, integrity of author's source, no discrimination against persons or groups or fields of endeavor, distribution of license, license must not be specific to a product, license must not restrict other software, license must be technology-neutral.
- **Free Software** — the **FSF (Free Software Foundation)** maintains the **Free Software Definition** built on the **Four Freedoms**. A license is "free" if users have:
  - Freedom 0: run the program for any purpose
  - Freedom 1: study how it works and modify it (requires source access)
  - Freedom 2: redistribute copies
  - Freedom 3: distribute modified versions
- **Source-Available** — neither OSI- nor FSF-approved. Source is published but the license restricts use, modification, redistribution, or commercial use. Examples: BSL, SSPL, Elastic v2, Functional Source License.

In practice, OSI-approved and FSF-approved sets overlap by ~95%. Almost every license that's one is also the other. The disagreements are at the margins (e.g., Sun's old SISSL was OSI-approved but FSF-disapproved; some CC licenses approved by neither).

**License vs Contract** — a third layer:

- A pure copyright **license** is a unilateral permission grant. You don't sign anything. By using the code under license terms you implicitly accept them.
- A **contract** is a mutual agreement. Both parties exchange consideration. EULAs and CLAs (Contributor License Agreements) are contracts.
- The **bare-license-vs-contract** distinction matters in court. A pure license breach is a copyright infringement (statutory damages, injunctions, willful infringement multipliers). A contract breach is a contract dispute (only contractual remedies, must show damages).
- The **GPL** has been litigated (e.g., *Jacobsen v. Katzer*, 2008) and treated as a license — meaning copyright remedies are available against violators, not just contract remedies.

```text
  +-------------+    +-------------+    +--------------+
  |  Copyright  | -> |   License   | -> |   Contract   |
  |  (default)  |    |  (grant)    |    |  (agreement) |
  +-------------+    +-------------+    +--------------+
   automatic           unilateral          mutual
   "all rights         "you may do         "we agree
    reserved"           X if you Y"         to do X"
```

The **license**, as a creature of copyright law, is the FOSS default. Contracts (CLAs, EULAs, dual-license commercial agreements) layer on top.

## Why Licensing Matters

**Without a license, code is "all rights reserved"** — full stop. A GitHub repo with no LICENSE file is not "open source," even if it's public. Cloning, reading, learning from it, forking, or redistributing it is technically copyright infringement. GitHub's Terms of Service grant a narrow display/fork right to GitHub itself and to other GitHub users, but no commercial-use right and no right to take the code outside GitHub.

**Using unlicensed code is copyright infringement** — and the copyright holder can:

- Issue a **DMCA takedown** to your hosting provider
- Send a **cease and desist** letter
- Sue for **statutory damages** (in the US, $750 to $30,000 per work, up to $150,000 for willful infringement)
- Sue for **actual damages plus disgorgement** of profits
- Get an **injunction** halting your distribution

**The legal liability axis** — if you ship code your company didn't write, downstream users may inherit license obligations. Failing to comply means:

- Customer-facing legal risk (your contracts often warrant your software is non-infringing)
- Re-distribution of derivative works may force you to provide source you don't want to provide (GPL trigger)
- A single AGPL dependency in your SaaS may compel you to publish your entire backend

**The social-contract axis** — FOSS only works because authors trust users to honor the license. Violators get publicly shamed (Software Freedom Conservancy enforcement, gpl-violations.org), lose community goodwill, get banned from package registries, and may find themselves unable to upstream patches.

**The supply chain axis** — modern projects pull thousands of transitive dependencies. Each one carries its own license. Compliance is a graph problem, not a leaf problem. SBOMs (Software Bills of Materials) are now mandatory for US federal contracts (EO 14028) and recommended for all production software.

```text
   +-----------+     +--------------+     +--------------+
   |  YOU      | --> |  DEPENDENCY  | --> | TRANSITIVE   |
   |  (MIT)    |     |  (MIT)       |     | DEP (GPL-3)  |
   +-----------+     +--------------+     +--------------+
        |                                        |
        +----------------------------------------+
                           |
                           v
                    Your binary now distributes
                     GPL-3 code. You owe source
                       offer to recipients.
```

## Quick Decoder Table

When you encounter a new license, ask in this order:

1. **Is it OSI-approved?** (look up at opensource.org/licenses; if no, it's source-available, not FOSS)
2. **Is it FSF-approved?** (look up at gnu.org/licenses/license-list.html)
3. **What is the copyleft strength?**
   - None: permissive (MIT, BSD, Apache, ISC, Zlib)
   - Weak / file-level: MPL-2.0, EPL-2.0, LGPL, CDDL
   - Strong / project-level: GPL, AGPL, OSL
4. **Is attribution required?** Practically always yes for FOSS.
5. **Is there an explicit patent grant?** MIT/BSD/ISC/Zlib: no. Apache/MPL/GPLv3/AGPL/EPL/LGPLv3: yes.
6. **Are there trademark restrictions?** Apache: explicit no-grant. Most others silent.
7. **Is it viral?** "Viral" is the GPL-as-virus meme: combining it with proprietary forces the proprietary code open. Permissive: not viral. Weak copyleft: file-level viral. Strong copyleft: project-level viral. AGPL: also viral over network.
8. **NOTICE file requirement?** Apache yes. Most others no.

```text
+----------------+----------------+--------+----------+--------+----------+----------+
| License        | Copyleft       | Patent | Trademark| OSI    | FSF      | NOTICE   |
+----------------+----------------+--------+----------+--------+----------+----------+
| MIT            | none           | implicit | silent | Y      | Y        | no       |
| BSD-2-Clause   | none           | implicit | silent | Y      | Y        | no       |
| BSD-3-Clause   | none           | implicit | non-endorse | Y | Y        | no       |
| ISC            | none           | implicit | silent | Y      | Y        | no       |
| 0BSD           | none           | implicit | silent | Y      | Y        | no       |
| Unlicense      | none / PD      | implicit | silent | Y      | Y        | no       |
| Apache-2.0     | none           | YES     | NO       | Y      | Y        | YES      |
| Zlib           | none           | implicit | non-misrep | Y    | Y        | no       |
| LGPL-2.1       | weak / library | implicit | silent | Y      | Y        | no       |
| LGPL-3.0       | weak / library | YES     | silent   | Y      | Y        | no       |
| MPL-2.0        | weak / file    | YES     | silent   | Y      | Y        | no       |
| EPL-2.0        | weak / module  | YES     | silent   | Y      | Y(some)  | no       |
| CDDL-1.0       | weak / file    | YES     | silent   | Y      | Y(GPLincomp)| no    |
| GPL-2.0-only   | strong         | implicit| silent   | Y      | Y        | no       |
| GPL-2.0+       | strong         | implicit| silent   | Y      | Y        | no       |
| GPL-3.0        | strong         | YES     | silent   | Y      | Y        | no       |
| AGPL-3.0       | strong+network | YES     | silent   | Y      | Y        | no       |
| EUPL-1.2       | strong         | YES     | silent   | Y      | Y        | no       |
| OSL-3.0        | strong+network | YES     | silent   | Y      | Y        | no       |
| CC0-1.0        | none / PD      | waiver  | silent   | N      | Y        | no       |
| WTFPL          | none / joke    | none    | silent   | N      | Y        | no       |
| BSL-1.1        | source-avail   | varies  | silent   | N      | N        | depends  |
| SSPL-1.0       | source-avail   | per-§   | silent   | N      | N        | no       |
| Elastic v2     | source-avail   | revoke  | silent   | N      | N        | no       |
| FSL            | source-avail   | varies  | silent   | N      | N        | no       |
| CC-BY-4.0      | none           | implicit | silent | N(sw)  | Y(docs)  | yes      |
| CC-BY-SA-4.0   | strong (docs)  | implicit | silent | N(sw)  | Y(docs)  | yes      |
| CC-BY-NC-4.0   | NOT FOSS       | n/a     | silent   | N      | N        | yes      |
| CC-BY-ND-4.0   | NOT FOSS       | n/a     | silent   | N      | N        | yes      |
| GFDL-1.3       | strong (docs)  | implicit | silent | N(sw)  | Y(docs)  | yes      |
+----------------+----------------+--------+----------+--------+----------+----------+
```

## Permissive Licenses Family

Permissive licenses say: "do what you want, but keep my copyright notice."

### MIT

The shortest, most popular, most-flexible practical license.

```text
MIT License

Copyright (c) <year> <copyright holders>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```

**Obligations**: include the copyright notice and the permission notice in all copies.
**Patent grant**: not explicit. Implicit-license theories exist but are debated.
**Trademark**: silent.
**OSI / FSF**: yes / yes.
**Use cases**: libraries you want maximum adoption; small utilities; node_modules-friendly.
**Warning**: if patent rights matter (e.g., your code touches a heavily-patented domain like video codecs), MIT does not protect downstream users from patent suits. Use Apache-2.0.

### BSD-2-Clause (FreeBSD / Simplified BSD)

```text
Copyright (c) <year>, <name>
All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"...
```

Functionally equivalent to MIT. Two clauses: keep the notice in source; keep the notice in binary documentation. Used by FreeBSD.

### BSD-3-Clause

Adds a non-endorsement clause to BSD-2-Clause:

```text
3. Neither the name of the copyright holder nor the names of its contributors
   may be used to endorse or promote products derived from this software
   without specific prior written permission.
```

This is a contractual-trademark-like clause: no using the project's name to advertise your derivative without permission. Used by Go, NumPy, the original BSD distribution, and most "modern BSD."

### BSD-4-Clause (deprecated)

The original 4-clause BSD had an "advertising clause":

```text
3. All advertising materials mentioning features or use of this software
   must display the following acknowledgement:
       This product includes software developed by <name>.
```

The advertising clause caused ~75 distinct attribution strings to be required in any advertising for a typical BSD-derived OS. Universities relented and dropped it (UC Berkeley in 1999). Modern BSD-licensed projects use the 2-clause or 3-clause forms.

### ISC

Functionally identical to MIT, slightly shorter, written by the Internet Systems Consortium for ISC software (BIND, dhcpd, NTP). OpenBSD's preferred license for its own code.

```text
Copyright (c) <year> <name>

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted, provided that the above
copyright notice and this permission notice appear in all copies.

THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES...
```

### 0BSD (Zero-Clause BSD / Free Public License 1.0.0)

Drops the attribution requirement entirely. Public-domain-equivalent.

```text
Copyright (C) <year> <name>

Permission to use, copy, modify, and/or distribute this software for any
purpose with or without fee is hereby granted.

THE SOFTWARE IS PROVIDED "AS IS"...
```

### Unlicense

A public-domain dedication with a permissive fallback for jurisdictions that don't recognize public-domain dedication of copyright. Authored by SQLite's D. Richard Hipp's neighbors. SQLite itself uses a similar but separate "blessing in lieu of license."

```text
This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any commercial or non-commercial purpose, and by any means.

In jurisdictions that recognize copyright laws, the author or authors
of this software dedicate any and all copyright interest in the
software to the public domain. We make this dedication for the benefit
of the public at large and to the detriment of our heirs and
successors. We intend this dedication to be an overt act of
relinquishment in perpetuity of all present and future rights to this
software under copyright law.

THE SOFTWARE IS PROVIDED "AS IS"...
```

OSI-approved (since 2020); FSF-approved.

### Apache-2.0

The most-feature-rich permissive license. Modern enterprise default.

Key sections:

- **§1 Definitions** — defines "License", "Licensor", "Legal Entity", "Source", "Object", "Work", "Derivative Works", "Contribution", "Contributor".
- **§2 Grant of Copyright License** — perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable.
- **§3 Grant of Patent License** — explicit patent grant. Plus a "patent retaliation" clause: if you sue anyone over a patent in this Work, your patent license terminates.
- **§4 Redistribution** — keep the license; keep the copyright notice; if you modify, mark it so; if there's a NOTICE file, include its contents.
- **§5 Submission of Contributions** — by default, contributions are licensed under the same terms (the "inbound = outbound" rule).
- **§6 Trademarks** — explicit: this license does NOT grant permission to use the trade names, trademarks, service marks, or product names of the Licensor.
- **§7 Disclaimer of Warranty**.
- **§8 Limitation of Liability**.
- **§9 Accepting Warranty or Additional Liability** — you can offer warranties on derivatives, but only on your own behalf.

**Obligations when redistributing**:

1. Provide the license text.
2. Preserve copyright, patent, trademark, and attribution notices.
3. State significant changes.
4. If a NOTICE file exists in the upstream, include its contents.

**Patent retaliation** is asymmetric — it's not a blanket "no patent litigation," it's specifically "if you sue over THIS code's patents, your license to THIS code dies."

**Use cases**: Apache Software Foundation projects; enterprise libraries (Kubernetes, TensorFlow, Apache Kafka); anywhere with patent-laden domains.

### Zlib

Permissive with a non-misrepresentation clause:

```text
This software is provided 'as-is', without any express or implied
warranty. In no event will the authors be held liable for any damages
arising from the use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it
freely, subject to the following restrictions:

1. The origin of this software must not be misrepresented; you must not
   claim that you wrote the original software. If you use this software
   in a product, an acknowledgment in the product documentation would be
   appreciated but is not required.
2. Altered source versions must be plainly marked as such, and must not be
   misrepresented as being the original software.
3. This notice may not be removed or altered from any source distribution.
```

Used by zlib (the compression library) and many gamedev tools (libpng, GLFW, SDL2).

### X11 / X11-revised

The historical X Window System license. Functionally MIT-equivalent. The "X11 license" predates "MIT license" — they're often-confused names for nearly-identical text. The "revised" form just patches names and trademarks.

### PostgreSQL License

Three-paragraph permissive license used only by PostgreSQL:

```text
PostgreSQL Database Management System
(formerly known as Postgres, then as Postgres95)

Portions Copyright (c) 1996-<year>, The PostgreSQL Global Development Group
Portions Copyright (c) 1994, The Regents of the University of California

Permission to use, copy, modify, and distribute this software and its
documentation for any purpose, without fee, and without a written agreement
is hereby granted, provided that the above copyright notice and this
paragraph and the following two paragraphs appear in all copies.

IN NO EVENT SHALL THE UNIVERSITY OF CALIFORNIA BE LIABLE TO ANY PARTY FOR
DIRECT, INDIRECT, SPECIAL, INCIDENTAL, OR CONSEQUENTIAL DAMAGES...

THE UNIVERSITY OF CALIFORNIA SPECIFICALLY DISCLAIMS ANY WARRANTIES...
```

Effectively MIT-equivalent. OSI- and FSF-approved.

## Weak Copyleft Family

Weak copyleft says: "you must keep MY code open, but you can combine MY code with YOUR proprietary code without polluting yours."

### LGPL-2.1 (Lesser GPL)

The "library" GPL. Originally called "Library GPL" — renamed to "Lesser GPL" because the FSF doesn't want all libraries to be LGPL (FSF would prefer GPL to spread copyleft further). Designed for libraries where forcing GPL on consumers would harm adoption.

**Key idea**: the library itself is GPL'd. Programs that "merely link" against it (dynamically) can be under any license. Programs that statically link or otherwise tightly integrate must permit "user modification of the LGPL portions and reverse-engineering for debugging such modifications" — practically, this means providing the relinkable object code or unbundled libraries.

**The relink requirement**: when statically linking, you must provide enough material that an end user can replace the LGPL library with their own modified version and relink. In practice, ship the .o files or the .a static library so users can relink against a modified .a.

**Use cases**: GNU C Library (glibc), libstdc++ in some configurations, GTK+, Qt's earlier versions.

### LGPL-3.0

LGPL-3.0 is structured as "GPL-3.0 plus LGPL-additional-permissions." So §1-§17 are the GPL-3.0 text, and there's a short additional grant making it LGPL-shaped.

- Apache-style explicit patent grant (inherited from GPL-3.0 §11)
- Anti-tivoization: hardware-locked code is forbidden (inherited from GPL-3.0)
- Compatibility with Apache-2.0 (inherited from GPL-3.0)

### MPL-2.0 (Mozilla Public License)

**File-level copyleft**: the unit of copyleft is the file. Modify an MPL file, your modifications stay MPL. Add a new file, your new file can be any license. This makes "embedding MPL code into a proprietary codebase" practical.

**Patent grant**: explicit (§2.1).
**Patent retaliation**: yes (§5.2 — sue an MPL contributor over patents and your rights terminate).
**GPL-secondary-license**: §3.3 makes MPL-2.0 code combinable with GPL-2.0+, GPL-3.0+, LGPL-2.1+, LGPL-3.0+, and AGPL-3.0+ when the file's "Exhibit B" notice is absent. Default MPL-2.0 is GPL-compatible.

**Obligations**: per-file. Keep the MPL header in modified files; provide source for those files.

**Use cases**: Mozilla projects (Firefox, Thunderbird, Rust originally); LibreOffice components; H2 Database.

### EPL-2.0 (Eclipse Public License)

**Module-level copyleft**: the unit is "the program file you modify, plus separable modifications." Designed to be commercially business-friendly. Eclipse Foundation's flagship license.

**Secondary license clause (§4)**: contributors can also designate "Secondary Licenses" (typically GPL-2.0-or-later) so EPL-2.0 code can be combined with GPL-licensed code under specific conditions.

**Obligations**: provide source on modifications; preserve EPL notices.

**Use cases**: Eclipse IDE plugins, Jetty (some versions), JUnit 5.

### CDDL-1.0 (Common Development and Distribution License)

Sun Microsystems's MPL-like license. **File-level copyleft**. Controversially **GPL-incompatible** because of differences in how the two licenses handle distribution: CDDL requires CDDL terms on the file, GPL requires GPL terms on the program — and the two sets of terms cannot both apply to the same combined work.

**The Linux + ZFS gotcha**: the OpenZFS filesystem is CDDL'd. The Linux kernel is GPL-2.0-only. Combining them into a binary is legally murky. The Software Freedom Conservancy and FSF both consider it a GPL violation if distributed as a single binary. The OpenZFS project distributes only source; users build the kernel module locally. Ubuntu and others ship pre-built ZFS modules and rely on the "mere aggregation" defense — Canonical's lawyers concluded the legal risk is acceptable; many disagree.

**Use cases**: OpenSolaris-derived code (Illumos, OpenZFS, DTrace, NetBeans).

## Strong Copyleft Family

Strong copyleft says: "if MY code is in your project, your whole project must be the same copyleft license."

### GPL-2.0-only

The GNU General Public License, version 2. Released 1991. The classic "viral" license.

**Key sections**:

- **§1** — you can copy and distribute verbatim source as you receive it, provided you keep the copyright notice and disclaimer, and the license text.
- **§2** — you can modify, but the resulting modified work must be GPL-2.0 (with the same notice and warranty disclaimer), and you must cause modified files to carry prominent notices stating you changed them and the date.
- **§3** — you may distribute in object/binary form only if you also provide source (or a written offer good for 3 years).
- **§5** — you accept the license by modifying or distributing.
- **§6** — recipients of redistribution receive their license directly from the original licensor; you cannot impose further restrictions.
- **§7** — the "liberty or death" clause: if a court order makes you unable to distribute under GPL terms, you may not distribute at all.
- **§9** — the FSF can publish revised versions; if the program states "GPL version N or any later version", users have the option to use later versions; without that statement, the version is fixed.

**"or any later version"**: this is critical. A header like:

```text
This program is free software; you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation; either version 2 of the License, or (at
your option) any later version.
```

is **GPL-2.0-or-later** (also known as **GPL-2.0+**). A header that says only "version 2 of the License" is **GPL-2.0-only**. The Linux kernel is famously GPL-2.0-only — Linus refuses GPL-3.0 for kernel code.

**Distinguish**:

- `GPL-2.0-only` (older synonym: `GPL-2.0`) — this version, only.
- `GPL-2.0-or-later` (older synonym: `GPL-2.0+`) — this version OR any newer.
- `GPL-3.0-only` — version 3 only.
- `GPL-3.0-or-later` (synonym: `GPL-3.0+`) — version 3 or newer.

The SPDX 3.x guidance prefers the `-only`/`-or-later` suffixes; the legacy `+` notation is deprecated.

### GPL-2.0-or-later (GPL-2.0+)

GPL-2.0 with the "any later version" option. Allows downstream relicensing to GPL-3.0 (or future GPL-4.0). This is what most non-Linux GPL-2 projects chose so that future fixes can be applied.

### GPL-3.0

Released June 2007. Major changes from GPL-2.0:

- **Explicit patent grant** (§11): you grant downstream a patent license for any patents you hold that read on your contribution.
- **Patent retaliation** (§11 paragraph 3): sue someone over patents on this code, lose your license.
- **Apache-2.0 compatibility**: GPL-3.0 incorporated language to fix the patent-grant incompatibility, so Apache-2.0 code can be combined with GPL-3.0 code (this combination must be GPL-3.0 overall).
- **Anti-tivoization** (§6): "Tivo" was a TiVo DVR that ran GPL Linux but used a hardware DRM check that prevented users from loading their own modified Linux. GPL-3.0 §6 says: if you distribute a User Product with installed software, you must also provide the "Installation Information" needed to install modified versions on the same hardware. This blocks the "GPL but we lock the bootloader" trick.
- **Anti-DRM** (§3): you cannot use GPL-3 software to enforce DRM that would let you sue circumventors under DMCA-anti-circumvention rules.
- **Internationalization**: clearer terminology, fewer US-centric phrasings.

### GPL-3.0-or-later (GPL-3.0+)

Default for FSF GNU projects. Future-proofs against new GPL versions.

### AGPL-3.0 (Affero GPL)

Solves the "**SaaS loophole**" of GPL-3.0. Under GPL-3.0, copyleft is triggered by **distribution** of the binary. If you run a modified GPL-3 program on a server and let users access it over a network without giving them the binary, you are not "distributing" — and so you don't have to share your modifications.

AGPL-3.0 §13 closes this:

```text
Notwithstanding any other provision of this License, if you modify the
Program, your modified version must prominently offer all users
interacting with it remotely through a computer network (if your
version supports such interaction) an opportunity to receive the
Corresponding Source of your version by providing access to the
Corresponding Source from a network server at no charge, through some
standard or customary means of facilitating copying of software.
```

**AGPL trigger**: any time users interact with a modified version remotely, you must offer them the source.

**Famous AGPL projects**: MongoDB (until 2018, switched to SSPL); Mastodon; Nextcloud; some Tinkerpop/JanusGraph code; iText.

**Corporate avoidance**: Google's open-source compliance policy prohibits AGPL dependencies entirely. Many other companies do the same. The mere presence of an AGPL dependency can disqualify a library from enterprise adoption.

### GFDL (GNU Free Documentation License)

For documentation, manuals, textbooks. Strong copyleft for docs. The "Invariant Sections" feature is controversial — sections marked Invariant cannot be modified or removed by downstream, which the Debian project considers non-free. Used by GNU manuals and historically Wikipedia (until Wikipedia migrated to CC-BY-SA-3.0 in 2009).

### SISSL (Sun Industry Standards Source License) — deprecated

Sun's old license for OpenOffice.org. OSI-approved but FSF-disapproved. Sun retired it in favor of LGPL and later CDDL. Listed here for historical license-detection completeness.

### EUPL-1.2 (European Union Public License)

The European Commission's official license. Multi-language: legally binding in all 24 official EU languages, with each language version equivalent.

**Compatibility list**: EUPL-1.2 explicitly enumerates licenses it can be relicensed to and from: GPL-2.0/3.0, AGPL-3.0, OSL-3.0, EPL-1.0, CeCILL-2.0/2.1, MPL-2.0, LGPL-2.1/3.0. The compatibility list is actually a feature for European public-sector code reuse.

**Obligations**: copyleft (project-level for EUPL'd parts); preserve attribution; provide source.

**Use cases**: EU government and public-administration software (eIDAS reference implementations, some national gov projects).

### OSL-3.0 (Open Software License)

Drafted by Lawrence Rosen. Strong copyleft + network-use trigger (similar to AGPL). Uses different patent provisions than GPL — the patent retaliation is broader. Less common than AGPL but legally interesting; some prefer OSL for its cleaner drafting.

## Network Copyleft (AGPL Trigger)

The AGPL §13 trigger is the most-misunderstood copyleft mechanism. Some specifics:

**When does AGPL trigger?**

- The user interacts with the modified version remotely (network access).
- "Remotely" includes web UIs, REST APIs, gRPC, SSH, anything across a process or machine boundary.
- The trigger is on **modification AND remote interaction**. Running an unmodified AGPL program as a service does NOT trigger anything (you still must comply with §3-§5 if you redistribute the binary, but there's no extra remote-source obligation if you didn't modify).
- Bug fixes, patches, configuration changes that go beyond runtime config — these are modifications that trigger §13.

**The "merely linking" question**:

If you write proprietary code that talks to an unmodified AGPL service over the network, are you "modifying" it? The FSF's answer is: no. Your client is its own program; the AGPL service is its own program. Network communication is "mere aggregation" / "arms-length."

If you write proprietary code that *is linked into* the AGPL service (e.g., a plugin loaded into the AGPL'd server's process), then yes — you've created a derivative work, and the AGPL applies to your plugin too.

**The corporate-deterrent reality**:

- Google, Apple, Microsoft, and many large enterprises ban AGPL dependencies in their products because of unclear scope (especially the "is my proprietary microservice that calls AGPL service a derivative" question).
- This is over-conservative — strict reading of AGPL doesn't extend to network clients of unmodified code — but the bans persist because legal teams hate ambiguity.
- The result: AGPL is a *commercial-deterrent* license. It's effective for projects that want hobbyists and small startups to use freely while making large enterprises pay for a commercial license (the "open core" / "dual license" model).

**Famous AGPL pivots**:

- MongoDB switched FROM AGPL TO SSPL in 2018, because they decided AGPL didn't go far enough to deter cloud rehosters.
- Elasticsearch went from Apache-2.0 to Elastic v2 + SSPL in 2021 for similar reasons.

## Public Domain & Equivalents

**Pure public-domain dedication is jurisdictionally awkward**. Some countries (e.g., Germany, France) do not let an author *give up* copyright by saying "this is public domain." Authors retain "moral rights" they cannot waive. So a US developer's "I dedicate this to the public domain" is legally meaningless to a German user.

To get around this, public-domain-flavored licenses combine a dedication with a permissive fallback.

### CC0-1.0 (Creative Commons "No Rights Reserved")

The most legally robust public-domain dedication. Two-stage:

- **Dedication**: the author waives all copyright and related rights.
- **License fallback**: in jurisdictions where the dedication is ineffective, the work is licensed under a maximally-permissive license.
- **Trademark and patent waiver**: explicitly disclaims, but does not grant, trademark or patent rights. This is a known weakness — CC0 does NOT grant patent rights (CC's drafters deliberately avoided patents).

OSI-approved? **No.** OSI rejected CC0 in 2012 because of the patent disclaimer. Many companies' open-source policies treat CC0 as "permissive" anyway, but enterprise reviewers may flag it.

FSF: yes, "free."

### WTFPL (Do What The Fuck You Want To Public License)

```text
        DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
                    Version 2, December 2004

 Copyright (C) 2004 Sam Hocevar <sam@hocevar.net>

 Everyone is permitted to copy and distribute verbatim or modified
 copies of this license document, and changing it is allowed as long
 as the name is changed.

            DO WHAT THE FUCK YOU WANT TO PUBLIC LICENSE
   TERMS AND CONDITIONS FOR COPYING, DISTRIBUTION AND MODIFICATION

  0. You just DO WHAT THE FUCK YOU WANT TO.
```

OSI: not approved. Enterprise compliance reviewers will reject it (and rightly: it's not legally robust, and the language is unprofessional). FSF lists it as "free." Use it as a joke license; don't use it for code you want enterprises to consume.

### Unlicense

See above. Public-domain dedication with permissive fallback. OSI-approved (since 2020). FSF-approved.

### 0BSD

See above. Minimal-text public-domain-flavor. OSI- and FSF-approved.

## Non-FOSS / Source-Available

These licenses publish source but are NOT open source. Use them when you want competitive moats while still allowing inspection.

### BSL (Business Source License) 1.1

Drafted by MariaDB Corporation. Pattern:

- Source is published.
- Use is restricted (typically: "may not be used for a competing service").
- After **N years** (or a specific date), the same code automatically converts to a permissive license (typically Apache-2.0 or GPL-2.0).

Each release has its own conversion date; the conversion is on a rolling basis.

**Adopters**: HashiCorp (Terraform, Vault, Consul, Nomad, Vagrant — switched from MPL/MIT/Apache to BSL 1.1 in 2023), MariaDB (MariaDB Server core), Sentry (parts), CockroachDB (parts), Cube.dev, Couchbase.

The HashiCorp move triggered the **OpenTofu** fork (Linux Foundation) and the **OpenBao** fork (Vault) — communities that wanted to stay on the last MPL'd version.

**Compliance**: BSL is NOT FOSS during the BSL window. Treat it as a commercial license that needs review.

### SSPL (Server Side Public License) 1.0

MongoDB's anti-cloud-rehoster license. Modeled on AGPL but goes further:

```text
If you make the functionality of the Program or a modified version
available to third parties as a service, you must make the Service
Source Code available via network download to everyone at no charge,
under the terms of this License.

"Service Source Code" means the Corresponding Source for the Program
or the modified version, and the Corresponding Source for all
programs that you use to make the Program or modified version
available as a service, including, without limitation, management
software, user interfaces, application program interfaces, automation
software, monitoring software, backup software, storage software and
hosting software, all such that a user could run an instance of the
service using the Service Source Code you make available.
```

So if AWS were to host a SSPL'd MongoDB service, AWS would have to release ALL their cloud infrastructure code. That's the deterrent.

**OSI rejected SSPL** (twice — once on submission, once on resubmission). Reasons: violates OSD #6 (no discrimination against fields of endeavor — cloud hosting is a field) and OSD #9 (must not restrict other software).

**Adopters**: MongoDB, Elasticsearch (also offered Elastic v2 alongside SSPL), Redis (Redis Stack and parts of Redis 7.4 in 2024 with SSPL+RSALv2 dual).

### Elastic License 2.0

Elastic's anti-rehoster license:

```text
You may not provide the software to third parties as a hosted or
managed service, where the service provides users with access to any
substantial set of the features or functionality of the software.

You may not move, change, disable, or circumvent the license key
functionality in the software, and you may not remove or obscure any
functionality in the software that is protected by the license key.
```

OSI-rejected. FSF-rejected.

### Confluent Community License

Modeled on Elastic v2. Restricts SaaS rehosting. Used for parts of the Confluent Platform around Apache Kafka.

### Functional Source License (FSL)

Sentry's license. Pattern: use is restricted (no competing service); converts automatically after **2 years** to either Apache-2.0 or MIT (author's choice). Like BSL with shorter conversion.

```text
The Functional Source License is a permissive non-compete license that:
- Allows almost any commercial use
- Forbids only a "Competing Use"
- Converts to Apache 2.0 or MIT after 2 years
```

OSI-rejected (as expected; the non-compete clause violates OSD #6).

### The "Fauxpen-Source" / "Source-Available" Pattern

A growing class of licenses that look open but aren't:

- **BSL 1.1** (HashiCorp, MariaDB)
- **SSPL** (MongoDB)
- **Elastic v2** (Elasticsearch, Kibana, Beats post-2021)
- **Confluent Community License**
- **Functional Source License** (Sentry)
- **Redis Source Available License v2 (RSALv2)** — Redis post-2024
- **Polyform** family (Polyform Strict, Polyform Noncommercial, Polyform Internal-Use, Polyform Small Business, Polyform Free Trial, Polyform Shield) — a kit of source-available licenses
- **PostgreSQL Forks under "non-compete"** — none of the canonical Postgres forks; mentioned for completeness
- **Server Side Public License (SSPL)** — MongoDB

Compliance treat-as: commercial license. Need legal review. The benefits are:

- Source available (you can read, audit, fork-internally).
- Often free for non-competing use (small teams, internal tooling, self-hosted).

The drawbacks:

- Cannot be combined into permissive/copyleft FOSS distributions.
- Many enterprise "approved license" policies disallow them by default.
- Distros like Debian/Fedora cannot ship them in their main archives.

## Documentation Licenses

Software licenses are often a poor fit for non-software works (documentation, art, fonts, music). Creative Commons fills the gap.

### CC-BY-4.0 (Attribution 4.0 International)

Permissive. Anyone can reuse, modify, redistribute, monetize — provided they:

- Give attribution.
- Indicate changes (if any).
- Provide a link to the license.

OSI: not approved (CC-BY is not designed for software; OSI's review concluded it's not appropriate for software). FSF: approved for non-software works. **Most modern documentation projects use CC-BY-4.0** (e.g., Wikipedia, much of the Python documentation).

### CC-BY-SA-4.0 (Attribution-ShareAlike 4.0)

Copyleft for documentation. Same as CC-BY but adds the ShareAlike requirement: derivative works must be licensed under the same (or compatible) license.

**ShareAlike compatibility (CC's official list)**: CC-BY-SA-4.0 derivatives can be relicensed to CC-BY-SA-4.0 or to GFDL (one-way only, certain conditions).

Used by **Wikipedia** (since June 2009), the **Stack Exchange / Stack Overflow** content corpus (CC-BY-SA-4.0 since 2018), most **OpenStreetMap** raw data was once CC-BY-SA but moved to ODbL.

### CC-BY-NC-4.0 (Attribution-NonCommercial 4.0)

Adds: no commercial use. **NOT OSI-approved for software** (violates OSD #6: discrimination against fields of endeavor). Also conflicts with FSF's Freedom 0 (run for any purpose).

In practice: NC means "I want to forbid commercial competitors but I'm okay with hobbyist use." Avoid for software. For documentation, NC creates a fork-and-relicensing tax that breaks compatibility with most other libre licenses.

### CC-BY-ND-4.0 (Attribution-NoDerivatives 4.0)

Adds: no derivative works. **NOT OSI-approved** (violates OSD #3). Conflicts with FSF's Freedom 1 and 3.

Use case: official position papers, statements, art that the author wants kept whole. Never use for software.

### CC-BY-NC-SA-4.0 / CC-BY-NC-ND-4.0

Combinations. Same NC and ND issues.

### The CC-NC and CC-ND cannot be used for FOSS code

The Open Source Definition forbids both:

- "No discrimination against fields of endeavor" (OSD #6) blocks NC.
- "Allows derivative works" (OSD #3) blocks ND.

The FSF agrees: NC and ND violate the four freedoms.

### GFDL (revisited)

Strong copyleft for documentation. The **Invariant Sections** feature is the controversial part — sections marked Invariant must be preserved unaltered in all derivatives. The Debian Project considers GFDL with Invariant Sections **non-free** (DFSG-incompatible). Modern docs projects prefer CC-BY-SA-4.0.

## Patent Grants — the big differentiator

Patent grants are the most-overlooked dimension of license selection. Without an explicit patent grant, downstream users may be sued for patent infringement even by upstream contributors.

### Licenses with NO explicit patent grant

- **MIT** — silent on patents
- **BSD-2-Clause** — silent
- **BSD-3-Clause** — silent
- **ISC** — silent
- **Zlib** — silent
- **GPL-2.0** — silent on the explicit grant; weakly addresses patents in §7 but does not explicitly grant patent rights
- **PostgreSQL License** — silent

Some lawyers argue an **implicit patent license** exists for whatever's necessary to use the licensed software (the doctrine of "implied license to practice patents necessarily infringed by exercise of granted copyright"). This is not universally accepted; courts haven't fully clarified.

### Licenses WITH explicit patent grant

- **Apache-2.0 §3** — broad patent grant; retaliation if you sue over Apache code's patents.
- **GPL-3.0 §11** — patent grant; retaliation.
- **AGPL-3.0 §11** — same as GPL-3.0.
- **LGPL-3.0 §11** — same as GPL-3.0 (inherited).
- **MPL-2.0 §2.1** — patent grant; retaliation in §5.2.
- **EPL-2.0 §2(b)** — patent grant.
- **CDDL-1.0 §2.1** — patent grant.
- **OSL-3.0 §2** — patent grant; broader retaliation.

### Patent retaliation patterns

The **specific-patent retaliation** (Apache, MPL, GPL-3): if you sue over patents READING ON THIS LICENSED CODE, you lose your license.

The **broader retaliation** (OSL, some others): if you sue any contributor over ANY patent, you may lose your license. Lawyers tend to find this scarier and avoid OSL.

## Trademark Clauses

Most FOSS licenses are **silent on trademarks** — they grant only copyright (and sometimes patent) rights. Trademark rights are NOT granted by these licenses, and you must rely on trademark law.

**Apache-2.0 §6** is unusual in stating this explicitly:

```text
6. Trademarks. This License does not grant permission to use the trade
   names, trademarks, service marks, or product names of the Licensor,
   except as required for describing the origin of the Work and
   reproducing the content of the NOTICE file.
```

**The "you can fork the code but not the name" reality**:

- The Linux kernel is GPL-2.0-only, but "Linux" is a trademark of Linus Torvalds. You can fork the code; you cannot sell your fork as "Linux" without permission.
- Mozilla owns the "Firefox" trademark. The code is MPL'd, but Debian historically had to ship as "Iceweasel" because their patches didn't meet Mozilla's trademark policy. (Resolved in 2016 when Debian adopted enough Mozilla branding.)
- "Red Hat Enterprise Linux" — code is GPL/etc., but the "Red Hat" trademark is restricted. CentOS, Rocky, AlmaLinux strip Red Hat branding.
- "MySQL" is Oracle's trademark. MariaDB is a fork that uses different branding.
- "OpenJDK" — branding rules from Oracle.

**Best practice for projects**: maintain a separate "trademark policy" document spelling out fair-use, descriptive use, and what requires permission. The Apache Software Foundation, the Linux Foundation, and Mozilla all publish trademark policies that reasonable forks can rely on.

## NOTICE File Requirements

**Apache-2.0 §4(d)** requires:

```text
If the Work includes a "NOTICE" text file as part of its distribution,
then any Derivative Works that You distribute must include a readable
copy of the attribution notices contained within such NOTICE file...
You may add Your own attribution notices within Derivative Works that
You distribute, alongside or as an addendum to the NOTICE text from
the Work, provided that such additional attribution notices cannot be
construed as modifying the License.
```

So **NOTICE is cumulative**: every Apache-licensed dependency contributes its NOTICE contents to your overall NOTICE file. A typical enterprise application has hundreds of attributions in NOTICE.

### What goes in NOTICE

- Copyright statements that the author wants prominently displayed.
- Required attributions (e.g., "This product includes software developed at <org>").
- Specific text the project wants every downstream binary to ship.

### What does NOT go in NOTICE

- The full license text (that goes in LICENSE).
- General changelog or build info.
- Version strings.

### Tools that handle NOTICE aggregation

- **license-checker** (npm) — extracts and aggregates JS package licenses + NOTICEs.
- **fossology** — a full SCA/license-compliance suite (FOSSology).
- **REUSE** (REUSE.software) — spec for in-source SPDX headers + LICENSES/ directory; tooling: `reuse lint`, `reuse spdx`.
- **askalono** — Rust crate identifying licenses by fingerprint matching.
- **scancode-toolkit** — full-featured scanner producing SPDX/CycloneDX SBOMs.
- **licensee** — Ruby gem (used by GitHub.com).
- **go-license-detector** — Go fork used by some FOSSology tooling.

## Compatibility Matrix

License compatibility means: can code under license A be combined with code under license B and the combined work be lawfully distributed (and under what license)?

The **direction matters**: "compatible into A" is asymmetric. If you can combine MIT code into a GPL project (yes, you can), the combined work is GPL — not MIT. The "umbrella" is the more-restrictive license.

```text
Combining-into legend:
  Y   = compatible (combined work uses target license)
  N   = NOT compatible
  -   = same license, trivially compatible
  *   = compatible only with conditions

target ->        MIT  Apache-2.0  GPL-2-only  GPL-2-or-later  GPL-3  AGPL-3  LGPL-2.1  LGPL-3  MPL-2.0  CDDL  BSL  SSPL
source:
MIT              -    Y           Y           Y               Y      Y       Y         Y       Y        Y     N    N
Apache-2.0       N    -           N(*)        Y               Y      Y       N(*)      Y       Y        Y     N    N
GPL-2-only       N    N           -           Y               N      N       Y         N       N        N     N    N
GPL-2-or-later   N    N           Y           -               Y      Y       Y         Y       Y(*)     N     N    N
GPL-3            N    N           N           N               -      Y       N         Y       Y(*)     N     N    N
AGPL-3           N    N           N           N               Y      -       N         Y       Y(*)     N     N    N
LGPL-2.1         N    N           Y           Y               Y      Y       -         Y       Y        N     N    N
LGPL-3           N    N           N           N               Y      Y       N         -       Y        N     N    N
MPL-2.0          N    N           N(*)        Y(*)            Y(*)   Y(*)    Y(*)      Y(*)    -        N     N    N
CDDL             N    N           N           N               N      N       N         N       N        -     N    N
BSL              N    N           N           N               N      N       N         N       N        N     -    N
SSPL             N    N           N           N               N      N       N         N       N        N     N    -
```

### Notable compatibility facts

- **GPL-2.0-only ↔ GPL-3.0-only** is **not** compatible. Code under GPL-2.0-only and GPL-3.0-only cannot be combined into one project, because each demands its terms apply to the whole. This is why the Linux kernel (GPL-2.0-only) cannot incorporate GPL-3.0-only code.
- **GPL-2.0-or-later → GPL-3.0** works because the "or later" clause permits the upgrade.
- **Apache-2.0 → GPL-3.0** is compatible (the FSF resolved this in GPL-3.0 specifically).
- **Apache-2.0 → GPL-2.0-only** is NOT compatible — Apache's patent retaliation clause is considered an additional restriction GPL-2.0 doesn't permit.
- **LGPL → GPL**: can be one-way upgraded to GPL. Reverse is not possible (you cannot relicense a pure GPL project to LGPL because contributors only granted GPL terms).
- **MIT → GPL**: compatible. MIT into a GPL project is fine; the combined work is GPL.
- **MPL-2.0 → GPL**: compatible because MPL-2.0 §3.3 ("dual-licensed by default with GPL/LGPL/AGPL secondary").
- **CDDL ↔ GPL**: NOT compatible. The famous Linux + ZFS legal cloud.

### Practical compatibility decisions

| Want to do | Compatible? | Resulting umbrella license |
| --- | --- | --- |
| Use MIT lib in Apache-2.0 project | Yes | Apache-2.0 |
| Use Apache-2.0 lib in MIT project | Practically Yes | MIT (but inherit Apache NOTICE/patent-retaliation obligations on the included code) |
| Use Apache-2.0 lib in GPL-2.0-only project | NO | — |
| Use Apache-2.0 lib in GPL-3.0 project | Yes | GPL-3.0 |
| Use GPL-3.0 lib in MIT project distributed as binary | NO (or your project becomes GPL-3.0) | GPL-3.0 |
| Use LGPL-2.1 lib in proprietary, dynamic link | Yes | proprietary (with relink obligation on LGPL part) |
| Use LGPL-2.1 lib in proprietary, static link | Yes (with conditions) | proprietary (must offer relink) |
| Use AGPL-3.0 lib in SaaS | Yes | your service code must offer source |
| Use BSL during BSL window in proprietary | Maybe (depends on additional restrictions) | proprietary, but BSL terms restrict use |
| Use MIT in CC-BY-SA-4.0 docs project | Compatible-ish (different domains) | code stays MIT, docs stay CC-BY-SA |

## License Proliferation History

In the late 1990s, every project shipped its own bespoke license:

- **Sun Public License**, **Sun Industry Standards Source License (SISSL)**, **NETSCAPE Public License**, **IBM Public License**, **Apple Public Source License**, **Common Public License**, **Mozilla Public License v1**, **Q Public License (Qt's old license)**, **Artistic License v1**, **Frameworx License**, **NASA Open Source Agreement**, **VOVIDA Software License**, **W3C Software License**, **Sleepycat License**, **PHP License**, **Python License**, ...

Each had subtle twists. Compatibility was a mess. Auditing 200 dependencies meant reading 200 different license texts.

**OSI's License Proliferation Committee (2005-2009)** reviewed submissions, retired duplicates, and recommended a "preferred licenses" list. Today's de-facto preferred set:

- Permissive: **MIT, BSD-2/3-Clause, ISC, Apache-2.0**
- Weak copyleft: **MPL-2.0, LGPL-2.1+/3.0+**
- Strong copyleft: **GPL-2.0-or-later, GPL-3.0-or-later, AGPL-3.0**

Most new projects use one of these. License submissions to OSI have slowed dramatically, partly due to OSI's own pushback and partly due to industry convergence.

The most-recent novel-license discussions (2018-2024) are about **source-available** licenses (BSL, SSPL, Elastic v2, FSL) rather than new FOSS licenses.

## SPDX Identifiers

The **Software Package Data Exchange (SPDX)** is a Linux Foundation standard for communicating SBOM/SCA information. Its **license list** at spdx.org/licenses provides a canonical, short identifier for each known license.

Examples:

```text
MIT
BSD-2-Clause
BSD-3-Clause
Apache-2.0
GPL-2.0-only
GPL-2.0-or-later
GPL-3.0-only
GPL-3.0-or-later
AGPL-3.0-only
AGPL-3.0-or-later
LGPL-2.1-only
LGPL-2.1-or-later
LGPL-3.0-only
LGPL-3.0-or-later
MPL-2.0
EPL-2.0
CDDL-1.0
ISC
0BSD
Unlicense
CC0-1.0
CC-BY-4.0
CC-BY-SA-4.0
GFDL-1.3-only
GFDL-1.3-or-later
EUPL-1.2
OSL-3.0
Zlib
WTFPL
```

### SPDX expression syntax

You can combine identifiers:

```text
# Single license
MIT

# Multiple licenses, choose any one
MIT OR Apache-2.0

# Multiple licenses, all must comply
MIT AND BSD-2-Clause

# Exception modifier
GPL-2.0-or-later WITH Classpath-exception-2.0
GPL-3.0-or-later WITH GCC-exception-3.1
Apache-2.0 WITH LLVM-exception
```

### License exceptions

Exceptions are short addenda that modify a license. Examples:

- **Classpath-exception-2.0** — used by OpenJDK; lets you link the licensed code with non-GPL code without the linking forcing GPL upward.
- **LLVM-exception** — Apache-2.0 plus a GPL-2.0 compatibility carve-out. LLVM uses this so it can be combined with GPL-2.0-only projects.
- **GCC-exception-3.1** — GPL-3.0 with carve-outs for runtime libraries.
- **Bison-exception-2.2** — GPL with permission to use Bison output without GPL applying to the user's parser.
- **Autoconf-exception-3.0** — GPL with permission to use autoconf macros in non-GPL projects.
- **Linux-syscall-note** — informal note on Linux kernel that user-space code calling syscalls is not derivative.

### Where SPDX identifiers go

- **In source files**: `// SPDX-License-Identifier: Apache-2.0` (the REUSE-recommended pattern).
- **In package manifests**: `package.json`, `Cargo.toml`, `pyproject.toml`, `setup.cfg` — the `license` field.
- **In SBOMs**: SPDX or CycloneDX format files.

```toml
# Cargo.toml
[package]
license = "MIT OR Apache-2.0"
```

```json
{
  "name": "my-pkg",
  "license": "MIT"
}
```

```python
# pyproject.toml
[project]
license = "MIT"
# or with classifiers
classifiers = ["License :: OSI Approved :: MIT License"]
```

## License Detection Tools

A multi-modal landscape:

### GitHub's licensee gem

```bash
gem install licensee
licensee detect /path/to/repo
licensee detect --json https://github.com/rails/rails
```

Used by GitHub.com to display the "License: MIT" badge on repo pages. Compares LICENSE-file content against a corpus of known licenses, returning a confidence score.

### askalono

Rust-native license fingerprinting:

```bash
cargo install askalono
askalono identify path/to/LICENSE
askalono crawl /path/to/repo
```

Fast; designed for batch use at Amazon scale.

### go-license-detector

Go binary using statistical N-gram fingerprinting against the SPDX corpus:

```bash
go install github.com/go-enry/go-license-detector/v4/cmd/license-detector@latest
license-detector /path/to/repo
```

### scancode-toolkit

Comprehensive Python tool:

```bash
pip install scancode-toolkit
scancode -clpieu --json-pp result.json /path/to/repo
```

Outputs full SPDX/CycloneDX-compatible reports.

### license-checker (npm)

```bash
npm install -g license-checker
cd /path/to/node-project
license-checker --json
license-checker --summary
license-checker --excludePackages 'pkg-a;pkg-b' --failOn 'GPL-3.0-only'
```

### cargo-license / cargo-deny

```bash
# cargo-license: lists what's in your dependency tree
cargo install cargo-license
cargo license

# cargo-deny: enforces a policy
cargo install cargo-deny
cargo deny init  # creates deny.toml
cargo deny check licenses
```

`deny.toml` example:

```toml
[licenses]
allow = ["MIT", "Apache-2.0", "BSD-3-Clause", "ISC", "Unicode-DFS-2016"]
deny = ["GPL-3.0", "AGPL-3.0", "LGPL-3.0"]
copyleft = "deny"
allow-osi-fsf-free = "neither"
```

### pip-licenses

```bash
pip install pip-licenses
pip-licenses --format=json
pip-licenses --fail-on='GPL v3'
```

### REUSE

```bash
pip install reuse
reuse lint              # checks every file has SPDX header + LICENSES/ entry
reuse spdx              # generates an SPDX SBOM for the project
reuse annotate -l MIT -c "Stevie <stevie@example.com>" file.go
```

### FOSSology

Self-hosted SCA platform:

```bash
docker run -d -p 8081:80 fossology/fossology
# UI at http://localhost:8081
```

### Commercial alternatives

- **FOSSA** — SaaS license/SBOM scanning.
- **Black Duck (Synopsys)** — enterprise SCA.
- **Snyk Open Source** — license + vuln.
- **WhiteSource (Mend)** — enterprise SCA.
- **OWASP Dependency-Check** — open source vuln + license.
- **Trivy** — Aqua's container scanner; license module.
- **Syft + Grype** — Anchore's tooling: syft generates SBOMs, grype matches vulns.
- **cdxgen** — multi-language CycloneDX SBOM generator.

### Fingerprint vs text-match approaches

- **Fingerprint** (askalono, scancode): hash distinctive substrings of canonical license texts; match candidate files by hash similarity.
- **Text-match** (licensee, regex-based): match file content against templates with placeholder slots for copyright/year.

Fingerprint approaches are faster and more robust to formatting variation. Text-match approaches surface more matches for fuzzy variants but produce more false positives.

## License Headers

Best practice (per **REUSE.software**):

- Put the **full license text** in `LICENSE` (or `LICENSES/<id>.txt` for multi-license projects).
- Put a **short SPDX identifier** in every source file's header.

```go
// SPDX-License-Identifier: Apache-2.0
// Copyright 2024 The Authors

package main
```

```python
# SPDX-License-Identifier: MIT
# Copyright 2024 Stevie Bellis <stevie@bellis.tech>

import sys
```

```c
/* SPDX-License-Identifier: GPL-2.0-or-later */
/* Copyright 2024 The Authors */

#include <stdio.h>
```

```rust
// SPDX-License-Identifier: MIT OR Apache-2.0
// Copyright 2024 The Authors
```

```javascript
// SPDX-License-Identifier: MIT
// Copyright (c) 2024 The Authors

'use strict';
```

```sh
#!/usr/bin/env bash
# SPDX-License-Identifier: 0BSD
# Copyright 2024 The Authors
```

### REUSE-compliant project layout

```text
.
|-- LICENSES/
|   |-- Apache-2.0.txt
|   |-- MIT.txt
|   `-- CC0-1.0.txt
|-- LICENSE -> LICENSES/Apache-2.0.txt   # symlink for visibility
|-- README.md                            # has SPDX header
|-- CONTRIBUTING.md                      # has SPDX header
|-- src/
|   |-- main.go                          # has SPDX header
|   `-- ...
|-- docs/
|   |-- index.md                         # SPDX: CC0-1.0 or CC-BY-4.0
`-- .reuse/
    `-- dep5                             # bulk attribution for files
                                         # without inline header
```

### .reuse/dep5 — bulk attribution

When you can't add inline SPDX headers (binary assets, JSON without comments), use `.reuse/dep5` (Debian-style):

```text
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: assets/icons/*
Copyright: 2024 The Authors
License: CC-BY-4.0

Files: data/*.json
Copyright: 2024 Stevie Bellis
License: MIT

Files: vendor/upstream-x/*
Copyright: 2010-2020 Upstream X Authors
License: Apache-2.0
```

## Dual-Licensing Patterns

When a single project ships under multiple licenses simultaneously:

### "Open Core" — base FOSS + paid features proprietary

The base codebase is FOSS (typically Apache-2.0 or MIT). The author ships a "Pro" or "Enterprise" build with extra features under a commercial license. Examples: Cal.com, Mattermost, Sentry (pre-FSL).

### "Reciprocity" / "Sell-Exception" — pay for proprietary use, free for FOSS

The codebase is GPL/AGPL. Users who want to combine it with non-GPL/AGPL code pay for a commercial license that grants additional permissions. Examples:

- **Qt** — Qt Company licenses Qt under GPL and LGPL **and** under a commercial license. Commercial buyers don't have to honor (L)GPL terms.
- **MySQL** (Oracle) — GPL-2.0-only + commercial. Commercial customers don't have to publish their schemas/clients.
- **iText** — AGPL + commercial.
- **Ghostscript** — AGPL + commercial.

### "Apache-2.0 OR MIT" — choice of license

Some projects (e.g., most of Rust's crates ecosystem) offer dual-license `MIT OR Apache-2.0`. The user picks one when consuming. Practically, this gives Apache's patent grant if you want it, MIT's brevity if you don't.

```toml
[package]
license = "MIT OR Apache-2.0"
```

### "MPL-2.0 + GPL secondary" — built into MPL-2.0

MPL-2.0 §3.3 makes MPL'd code combinable with GPL'd code by default. The MPL author's project remains MPL'd; downstream combiners can mix it into a GPL umbrella.

### CLA-enabled relicensing

For dual-licensing to work commercially, the project author must hold (or have permission for) all rights to relicense. There are two main ways:

1. **Copyright assignment / CLA**: each contributor signs a CLA (Contributor License Agreement) granting the project sufficient rights to relicense at will. The CLA may be:
   - A copyright **assignment** (FSF-style, GNU projects).
   - A copyright **license** (Apache CLA, GitHub CLA, EclipseFDN CLA).
2. **DCO (Developer Certificate of Origin)**: each commit has a `Signed-off-by:` line affirming the contributor has the right to contribute. Doesn't grant relicensing rights, only confirms the contribution is legitimate. Used by Linux kernel.

Without one of these, the project cannot unilaterally relicense to commercial terms — every contributor would need to consent.

The famous **MariaDB / MySQL** story: when Oracle acquired Sun (and MySQL), Oracle had MySQL's CLA, so they could relicense at will. The MariaDB fork was created precisely to keep a community-controlled GPL-only line going.

The famous **Redis 2024** relicensing: Redis Labs had been gathering CLAs since their commercial inception, so they could switch from BSD to RSALv2/SSPL.

The famous **HashiCorp BSL switch**: HashiCorp had not been requiring strong CLAs, but their use of permissive licenses (MPL/MIT) meant they didn't need contributor permission to fork into BSL — they just needed to stop accepting MPL/MIT contributions and re-license future commits.

## Specific License Quick Reference Cards

### MIT

- **One paragraph**: Most permissive practical FOSS license. Allows anything provided the copyright notice and permission notice are preserved.
- **Obligations**: ship the LICENSE text in distributions; keep the copyright notice.
- **No patent grant** (implicit only).
- **When to choose**: libraries you want everyone to use; small utilities; Python/JS/Go/Rust standard ecosystem fit. If you need patent protection, choose Apache-2.0.

### Apache-2.0

- **One paragraph**: Modern enterprise default. Permissive, with explicit patent grant, explicit trademark non-grant, NOTICE-file requirement.
- **Obligations**: include LICENSE; preserve copyright/patent/trademark/attribution notices; if you modify a file, mark it; include NOTICE contents.
- **Patent**: explicit grant; retaliation if you sue over Apache code.
- **When to choose**: enterprise libraries; anywhere with patent-laden domains; the safe modern default. If you don't care about patents and want minimal obligations, MIT is fine.

### GPL-3.0

- **One paragraph**: Strong copyleft + explicit patent grant + anti-tivoization + anti-DRM. Derivative works must be GPL-3.0.
- **Obligations**: ship source (or written offer); preserve copyleft license; provide installation information for User Products.
- **Patent**: explicit grant (§11); retaliation.
- **When to choose**: end-user applications you want forks of; tools where you care more about fork-back than wide adoption.

### MPL-2.0

- **One paragraph**: Weak copyleft, file-level. Modify an MPL file: stays MPL. Add new files: free. GPL-secondary-licensable by default (combinable into GPL).
- **Obligations**: per-file: keep MPL header; provide source for modified files.
- **Patent**: explicit grant; retaliation.
- **When to choose**: when you want LGPL semantics but cleaner; libraries where you want modifications back but don't care about user code; mixing with GPL allowed.

### AGPL-3.0

- **One paragraph**: Strong copyleft + network-use trigger. SaaS deployments with modifications must offer source.
- **Obligations**: same as GPL-3.0 + §13 source-on-network-interaction trigger.
- **When to choose**: when you specifically want to extend copyleft to SaaS; understand corporate adoption is hampered by AGPL bans.

### LGPL-3.0

- **One paragraph**: Library copyleft. The library is GPL-shaped but consumers (linkers, importers) are not pulled into copyleft if they merely link. Static linkers must offer relink material.
- **Obligations**: library mods must be LGPL; static linkers must provide relink-able material (object files or similar).
- **When to choose**: libraries where you want copyleft of derivatives but not of consumers; alternative to MPL.

### BSD-3-Clause

- **One paragraph**: Like MIT plus a non-endorsement clause.
- **Obligations**: keep the copyright; don't use the project's name to endorse derivatives without permission.
- **When to choose**: BSD-lineage projects; aliases for MIT for projects that want the non-endorsement signal.

### ISC

- **One paragraph**: Like MIT, even shorter. OpenBSD's preferred license. Functionally MIT-equivalent.
- **Obligations**: keep the copyright + permission notice.
- **When to choose**: when MIT's text feels redundant; OpenBSD upstreaming; tiny projects.

### Unlicense

- **One paragraph**: Public-domain dedication with permissive fallback for jurisdictions that don't recognize PD dedication.
- **Obligations**: none, beyond preserving notice.
- **When to choose**: code you genuinely want to abandon to public domain.

### CC-BY-4.0

- **One paragraph**: Permissive Creative Commons license for non-software works (docs, art, fonts).
- **Obligations**: attribution; indicate changes; link to license.
- **When to choose**: documentation, blog posts, slide decks, illustrations. NOT for software.

### CC0-1.0

- **One paragraph**: Public-domain dedication for any work, more legally robust than "I dedicate this to the public domain" alone.
- **Obligations**: none.
- **When to choose**: tiny code snippets, sample code, datasets, test data. Note: NOT OSI-approved (because no patent grant).

### BSL (Business Source License)

- **One paragraph**: Source-available with FOSS conversion timer (typically 4 years to Apache-2.0). Restrictions on competing-service use during the BSL window.
- **Obligations**: per-grant restrictions during the BSL window.
- **When to choose**: companies wanting source-availability + commercial moats.

### SSPL

- **One paragraph**: Anti-cloud-rehoster license. SaaS providers offering the program as a service must release ALL related operational code.
- **Obligations**: brutal source-disclosure if used as SaaS infrastructure.
- **When to choose**: only if you're MongoDB-style and willing to lose OSI/FSF approval.

### Elastic License v2

- **One paragraph**: Anti-rehoster license restricting hosted-service use. License-key tampering forbidden.
- **Obligations**: don't host competing services; don't tamper with license keys.
- **When to choose**: similar use case as SSPL but Elastic's flavor.

### Functional Source License (FSL)

- **One paragraph**: Source-available with 2-year conversion to MIT/Apache-2.0. Restricts competing-service use.
- **Obligations**: don't compete during the 2-year window.
- **When to choose**: faster conversion than BSL; same anti-compete posture.

## Choosing a License Decision Tree

```text
                       +---------------------------+
                       | Do you want others to be  |
                       | able to fork into closed  |
                       | proprietary?              |
                       +---------------------------+
                            |              |
                          YES              NO
                            |              |
                            v              v
                  +-------------+    +-----------------+
                  | Permissive  |    | Copyleft        |
                  +-------------+    +-----------------+
                       |                    |
                       v                    v
            +---------------+        +-----------------+
            | Need patent   |        | Library or app? |
            | protection?   |        +-----------------+
            +---------------+           |             |
                |       |              LIB           APP
              YES       NO              |             |
                |       |               v             v
                v       v       +-------------+  +-------------+
          Apache-2.0    MIT     |  Want fork  |  | SaaS-style  |
                                |  or static  |  | service?    |
                                |  link OK?   |  +-------------+
                                +-------------+    |        |
                                  |        |      YES       NO
                                  v        v       |        |
                              LGPL-3.0   MPL-2.0   v        v
                                                 AGPL-3.0  GPL-3.0
```

### Specific decision tips

- **"Do you want others to fork into proprietary?"** -> permissive (MIT, Apache-2.0).
- **"Want derivatives to stay open?"** -> copyleft.
- **"Need patent protection?"** -> Apache-2.0 or GPL-3.0.
- **"Worried about SaaS rehosting your code?"** -> AGPL-3.0 or BSL.
- **"Code is library to be embedded?"** -> MIT/Apache-2.0 or LGPL-3.0.
- **"Code is application end-user runs?"** -> GPL-3.0 acceptable.
- **"Documentation, media, fonts?"** -> CC-BY-4.0.
- **"Tiny snippets, public samples?"** -> CC0-1.0 or 0BSD.

### "If in doubt"

- **Library**: Apache-2.0.
- **End-user application you want forks of**: GPL-3.0-or-later.
- **Documentation**: CC-BY-4.0.
- **Crate/package in Rust ecosystem**: `MIT OR Apache-2.0`.
- **OpenBSD-flavored**: ISC.
- **Small utility for npm**: MIT.
- **Multi-organization-contributed standards body code**: Apache-2.0.

## License Mixing Reality

Modern projects are not green-field — they pull in hundreds or thousands of dependencies. Each carries its own license. Compliance is a graph problem.

### Typical large project license census

A typical Node.js application:

```text
total packages:        879
MIT:                   687
Apache-2.0:             92
ISC:                    47
BSD-3-Clause:           23
BSD-2-Clause:           13
0BSD:                    8
LGPL-3.0:                3
MPL-2.0:                3
CC0-1.0:                2
Apache-2.0 OR MIT:      1
custom:                  1
```

A typical Rust application:

```text
total crates:          312
MIT:                   159
MIT OR Apache-2.0:     108
Apache-2.0:             32
BSD-3-Clause:            5
BSD-2-Clause:            3
ISC:                     2
MPL-2.0:                 2
Unicode-DFS-2016:        1
```

### Compliance toolchain

```text
+------------+      +---------+      +----------+      +----------+
|   Source   | -->  |  SBOM   | -->  | License  | -->  | Policy   |
|  packages  |      | (SPDX/  |      |  match   |      | enforce  |
|            |      | CycloneDX)     |          |      |          |
+------------+      +---------+      +----------+      +----------+
                       syft           askalono         cargo-deny
                       cdxgen         scancode         license-checker
                       cargo          licensee         FOSSA
                       cyclonedx                       Black Duck
```

Steps:

1. **Inventory** — list every package in your dependency graph (direct + transitive).
2. **License extraction** — for each package, read its declared license + any LICENSE file content.
3. **Policy match** — compare against your allow-list and deny-list.
4. **SBOM emission** — produce an SPDX or CycloneDX SBOM as the audit artifact.
5. **CI gate** — fail builds if a deny-listed license appears.

### Allow-list / deny-list policy

```toml
# cargo-deny example
[licenses]
allow = [
  "MIT",
  "Apache-2.0",
  "Apache-2.0 WITH LLVM-exception",
  "BSD-2-Clause",
  "BSD-3-Clause",
  "ISC",
  "Unicode-DFS-2016",
  "CC0-1.0",
  "0BSD",
  "Zlib",
  "MPL-2.0",
]
deny = [
  "AGPL-3.0",
  "AGPL-3.0-or-later",
  "GPL-3.0",
  "GPL-3.0-or-later",
  "GPL-2.0",
  "GPL-2.0-or-later",
  "LGPL-3.0",
  "LGPL-3.0-or-later",
  "SSPL-1.0",
  "Elastic-2.0",
  "BSL-1.1",
]
copyleft = "warn"
allow-osi-fsf-free = "neither"
confidence-threshold = 0.93
```

### "If it's transitive, you still have to comply"

A direct dependency under MIT might pull in a transitive dependency under GPL-3.0. The GPL-3.0 obligations apply to your distributed binary regardless of whether you wrote against it directly.

Typical surprise: a "permissive-looking" library has a "GPL-3.0-only" optional feature flag; with that flag enabled, your binary becomes GPL-3.0 in distribution.

## Common Compliance Mistakes

### Removing LICENSE files when copying code

**Broken:**

```bash
$ cp upstream/lib/foo.go vendor/lib/foo.go
$ rm upstream-LICENSE
```

**Fixed:**

```bash
$ cp -r upstream/lib vendor/lib
$ cp upstream/LICENSE vendor/lib/LICENSE
$ git add vendor/lib/LICENSE
```

If you copy any third-party file, copy its license. The license travels with the code.

### Forgetting the Apache NOTICE

**Broken:**

```bash
$ tar czf release.tar.gz bin/myapp LICENSE
# but upstream had a NOTICE file too, you didn't include it
```

**Fixed:**

```bash
$ ls third_party/
apache-foo-1.0/  apache-bar-2.0/
$ cat third_party/*/NOTICE > NOTICE.aggregate
$ # add your own contributions
$ cat <<EOF >> NOTICE.aggregate
This product includes software developed at MyCo.
EOF
$ tar czf release.tar.gz bin/myapp LICENSE NOTICE.aggregate
```

### Using GPL code in MIT-licensed product

**Broken:**

```text
README.md says: License: MIT
package.json says: "license": "MIT"
node_modules/foo (GPL-3.0-only) is statically required at runtime
You ship the bundle.

You are now distributing GPL-3.0 code under-claim of MIT-only.
Recipients can demand source under GPL terms; your "MIT" claim is insufficient.
```

**Fixed (option 1):** Remove the GPL dependency.

**Fixed (option 2):** Relicense your project to GPL-3.0-or-later (requires consent from all your contributors).

**Fixed (option 3):** If GPL dep is optional, gate it behind a build flag and ship a non-GPL default build.

### Using MIT code without preserving copyright header

**Broken:**

```javascript
// in foo.js, you copied this from upstream:
function leftPad(str, len) { /* ... */ }
// you removed the upstream's "Copyright (c) ..." header
```

**Fixed:**

```javascript
// SPDX-License-Identifier: MIT
// Copyright (c) 2014 Cameron Westland (original author)
// Copyright (c) 2024 Stevie Bellis (modifications)

function leftPad(str, len) { /* ... */ }
```

### Not knowing what's in your node_modules / vendor / Pipfile

**Broken:** "I assume everything's MIT, our app's MIT, we're fine."

**Fixed:** run `license-checker --json` (or equivalent for your ecosystem) and audit. Don't assume.

### Mixing CDDL and GPL (the ZFS-on-Linux gotcha)

**Broken:** ship a Linux distro with the OpenZFS kernel module pre-built and statically linked into a single distributable kernel image.

**Fixed:** ship Linux + ZFS source separately. Build at install time on the user's machine. Or use a userspace alternative (FUSE-ZFS, with caveats).

### Embedding LGPL code statically without offering relinking

**Broken:** you static-linked an LGPL library into your proprietary binary; you ship just the binary; you offer no relink mechanism.

**Fixed:** ship the LGPL library as a `.a` or as `.o` files alongside your binary, with build instructions, so users can replace the LGPL library with a modified version and relink.

### AGPL deployment without source-disclosure mechanism

**Broken:** you forked an AGPL'd web app, modified it, deployed it as a SaaS, your users have no way to access your modified source.

**Fixed:** publish your modified source publicly (e.g., GitHub fork) AND link to it from a UI element (footer, About page, /source HTTP endpoint).

### Stripping copyright headers from copied snippets

**Broken:** you find a Stack Overflow snippet, paste it into your codebase, and remove the author's name.

**Fixed:** the Stack Overflow content corpus is CC-BY-SA-4.0. Attribution and ShareAlike apply. You should:

1. Add a comment crediting the SO author and link to the answer.
2. Recognize that CC-BY-SA-4.0 may not be license-compatible with your project (if your project is MIT, CC-BY-SA-4.0 code in it is awkward).
3. For very short snippets (a few lines), the de-minimis defense often applies, but it's better to either rewrite or attribute.

## License Auditing Workflows

### CI integration

GitHub Actions example:

```yaml
name: License Audit

on: [push, pull_request]

jobs:
  audit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Node
        uses: actions/setup-node@v4
        with:
          node-version: '20'

      - name: Install dependencies
        run: npm ci

      - name: License Check
        run: |
          npx license-checker --production --json > licenses.json
          npx license-checker --production --failOn 'GPL-3.0;AGPL-3.0;SSPL-1.0' --excludePackages 'optional-dev-dep@1.0.0'

      - name: SBOM (CycloneDX)
        run: |
          npx @cyclonedx/cdxgen -o sbom.json
          # upload to artifact
```

```yaml
# Rust
- name: cargo-deny
  uses: EmbarkStudios/cargo-deny-action@v2
  with:
    log-level: warn
    command: check
```

```yaml
# Python
- name: pip-licenses
  run: |
    pip install pip-licenses
    pip-licenses --format=json --output-file=licenses.json
    pip-licenses --fail-on='AGPL v3'
```

### SBOM standards

- **SPDX 2.3 / 3.x** — Linux Foundation standard. JSON, YAML, RDF, tag-value formats.
- **CycloneDX 1.5+** — OWASP standard. JSON, XML, protobuf.

Both are JSON-friendly, both can carry license info, both are widely supported.

### Licensee, askalono, scancode invocation

```bash
# Licensee
licensee detect --json /path/to/project

# askalono
askalono crawl /path/to/project --format json

# scancode
scancode --license --json result.json /path/to/project

# go-license-detector
license-detector /path/to/project
```

### Commercial alternatives reality check

- **FOSSA** — fast, easy CI integration, SaaS.
- **Black Duck** — slower, deeper, used in regulated industries.
- **Snyk** — cheap, integrates with vuln scanning.
- **Mend** — broad coverage but UX has rough edges.
- **OWASP Dependency-Check** — free, open source, mainly vuln but has license info.

## Common Errors and Disputes

### The Linux/ZFS CDDL+GPL issue

OpenZFS is CDDL'd. Linux is GPL-2.0-only. Combining them into a kernel image is legally murky. The Software Freedom Conservancy and FSF consider it a GPL violation. Canonical (Ubuntu) ships pre-built ZFS modules and relies on a "mere aggregation" defense. Most Linux distros distribute only ZFS source (so the user builds against their own kernel), avoiding the issue.

### Bruce Perens vs Open Source Initiative

Bruce Perens authored the original Open Source Definition (a Debian-derived doc). He has had several public conflicts with the OSI over license proliferation, board governance, and approval criteria. He resigned from OSI in 2020 over alleged dilution of OSD interpretation.

### "License bait and switch" — permissive then re-license to BSL/SSPL

The 2018-2024 trend:

- **MongoDB** — AGPL-3.0 -> SSPL (2018).
- **Elastic** — Apache-2.0 -> Elastic v2 + SSPL (2021); reverted partial Apache via the AGPL+ELv2+SSPL tri-license in 2024.
- **Redis** — BSD-3-Clause -> RSALv2 + SSPL (2024).
- **HashiCorp Terraform/Vault/Consul** — MPL-2.0 -> BSL 1.1 (2023).
- **Sentry** — BSD/Apache -> BSL -> FSL.
- **CockroachDB** — Apache-2.0 -> BSL -> back to Apache-2.0 (then back-and-forth).

Community responses: forks (OpenSearch from Elasticsearch; Valkey from Redis; OpenTofu from Terraform; OpenBao from Vault). The pattern: a corporate sponsor with sufficient CLA-collected rights re-licenses an established project; a community fork retains the previous license.

### "We used GPL code, didn't realize it was viral"

Classic learn-the-hard-way:

- A startup ships their proprietary product, customers reverse-engineer the binary and find GPL'd code.
- The Software Freedom Conservancy (or other enforcer) sends a notice.
- The startup must release source under GPL OR remove the GPL code.
- Often results in a public commitment to publish a portion of the codebase under GPL.

Examples (sanitized): Onyx Boox e-readers (kernel + system source eventually published); various IoT vendors (kernel source publishable in good faith after enforcement).

### The "what counts as a derivative work?" debate

- Static linking GPL: derivative (so combining product is GPL).
- Dynamic linking GPL: debated, FSF says derivative; some lawyers disagree for arms-length API consumers.
- Plugin systems with documented APIs: usually NOT derivative (e.g., GIMP plugins).
- gRPC/REST clients of GPL servers: NOT derivative.
- Embedded subprocess invocation: NOT derivative.

The FSF's position is at gnu.org/licenses/gpl-faq.html. Enterprise legal teams often take a more cautious view than FSF's.

## License Texts — where to find canonical

```text
opensource.org/licenses                  # OSI canonical short list + texts
spdx.org/licenses                        # full SPDX list with identifiers
gnu.org/licenses                         # GPL, LGPL, AGPL, GFDL canonical
creativecommons.org/licenses             # CC family
tldrlegal.com                            # plain-English summaries (NOT legally binding)
choosealicense.com                       # GitHub-hosted intro guide
github.com/spdx/license-list-data        # machine-readable SPDX corpus
github.com/spdx/spdx-spec                # SPDX spec
ifrosss.org                              # ISO/IEC FOSS metadata
fsf.org/licensing                        # FSF guidance on free licenses
oss-watch.ac.uk                          # Higher-ed FOSS guidance
```

Always reference the **canonical text** in your distributions, not a summary. tldrlegal is helpful for understanding but is not the legal text.

## Dual-License Author Rights — keeping the right to relicense

To dual-license commercially, the project must hold (or have permission for) all relevant rights from contributors. Three mechanisms:

### Copyright assignment (FSF model)

Each contributor signs a paper agreement transferring copyright to the project. Used by FSF GNU projects. Contributors retain rights to their contribution under whatever license but the project owns the copyright outright.

### Contributor License Agreement (CLA)

Each contributor signs a license (not assignment) granting the project broad rights including relicensing. Used by Apache Software Foundation, Eclipse Foundation, Microsoft, Google.

Apache ICLA / CCLA snippet:

```text
4. License grant. You hereby grant to the Foundation and to recipients
of software distributed by the Foundation a perpetual, worldwide,
non-exclusive, no-charge, royalty-free, irrevocable copyright license
to reproduce, prepare derivative works of, publicly display, publicly
perform, sublicense, and distribute Your Contributions and such
derivative works.
```

This grants Apache the right to relicense. Contributors retain ownership.

### Developer Certificate of Origin (DCO)

DCO is NOT a relicensing mechanism. It's a per-commit affirmation that the contributor has the right to contribute. Linux kernel uses DCO (Signed-off-by) but does NOT have a CLA — which means the kernel CANNOT be unilaterally relicensed because Linus + every contributor would all need to consent.

```bash
git commit -s -m "fix: ..."
# adds: Signed-off-by: Stevie Bellis <stevie@bellis.tech>
```

The DCO text:

```text
Developer's Certificate of Origin 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I have the
    right to submit it under the open source license indicated in the file; or

(b) The contribution is based upon previous work that, to the best of my
    knowledge, is covered under an appropriate open source license and I have
    the right under that license to submit that work with modifications,
    whether created in whole or in part by me, under the same open source
    license (unless I am permitted to submit under a different license), as
    indicated in the file; or

(c) The contribution was provided directly to me by some other person who
    certified (a), (b) or (c) and I have not modified it.

(d) I understand and agree that this project and the contribution are public
    and that a record of the contribution (including all personal information
    I submit with it, including my sign-off) is maintained indefinitely and
    may be redistributed consistent with this project and the open source
    license(s) involved.
```

### "Copyleft + commercial" dual-license requires aggregating rights

A project shipping under "GPL-3.0 OR Commercial" needs all contributors to grant rights for both. If a contributor only grants GPL-3.0 rights, the project author cannot offer their contribution under the Commercial branch. Hence CLAs.

Without CLAs, "GPL OR Commercial" works only if the author wrote 100% of the code (no external contributions) or the external contributions are themselves dual-licensed.

## Common Gotchas

### Gotcha 1: Releasing code without a LICENSE file

**Broken:**

```bash
$ git push origin main
# repo has README.md, src/, tests/ — but no LICENSE
# Effect: code is "all rights reserved". Even though it's public, no one can use it.
```

**Fixed:**

```bash
$ curl -L https://opensource.org/licenses/MIT > LICENSE
# edit the LICENSE to fill in the year and copyright holder
$ git add LICENSE
$ git commit -m "chore: add MIT license"
$ git push origin main
```

### Gotcha 2: Adding `# SPDX-License-Identifier: MIT` but no LICENSE

**Broken:**

```python
# SPDX-License-Identifier: MIT
# But the project has no LICENSE file, so no actual MIT text is present.
```

**Fixed:** add a `LICENSE` file with the full MIT text. SPDX identifiers in source files reference the canonical text but don't replace it.

### Gotcha 3: Removing the original copyright notice when forking

**Broken:**

```javascript
// Original upstream:
// Copyright (c) 2018 Original Author
// Licensed under MIT

// You forked and changed the header to:
// Copyright (c) 2024 You
// Licensed under MIT
```

**Fixed:**

```javascript
// Copyright (c) 2018 Original Author
// Copyright (c) 2024 You (modifications)
// Licensed under MIT
```

### Gotcha 4: "Dual licensed under MIT and GPL-3.0" without specifying

**Broken:** README says "Dual licensed MIT/GPL-3.0" but doesn't say "OR" vs "AND". Users don't know which to follow.

**Fixed:** use canonical SPDX expression syntax in package metadata:

```text
license: MIT OR GPL-3.0-or-later
# OR
license: MIT AND GPL-3.0-or-later
```

`OR` means user picks; `AND` means user must comply with both.

### Gotcha 5: Contributing GPL code to an Apache project

**Broken:** you submit a PR copying an algorithm from a GPL-3.0 project into an Apache-2.0 project. The maintainer doesn't notice and merges. Now the Apache-2.0 project is technically distributing GPL-3.0 code.

**Fixed:** rewrite the algorithm from scratch (clean room). Or get the original author to relicense their code under Apache-2.0.

### Gotcha 6: Adopting AGPL dep without realizing the SaaS implications

**Broken:** your SaaS product adopts an AGPL-3.0 library; you modify it for your needs; you don't expose the modified source. Six months later, a customer asks for source under AGPL §13.

**Fixed (option a):** publish your modified version of the library on GitHub and add a "/source" link to your product UI footer.

**Fixed (option b):** remove the AGPL dependency; rewrite or use a permissive alternative.

### Gotcha 7: Using CC-BY-NC for software

**Broken:** project ships with a `LICENSE` file containing CC-BY-NC-4.0 text. Users wonder if they can use it commercially.

**Fixed:** CC-BY-NC is NOT a software license; OSD-disqualified. Switch to MIT, Apache-2.0, or GPL.

### Gotcha 8: Public-domain claim that's invalid in some jurisdictions

**Broken:** README says "This is public domain." German user wonders if that's even legally meaningful (it isn't fully).

**Fixed:** use **CC0-1.0** or **Unlicense** or **0BSD**. These have permissive fallbacks that are valid in jurisdictions that don't recognize unilateral PD dedication.

### Gotcha 9: Apache NOTICE file forgotten when redistributing

**Broken:** your downstream binary includes Apache-2.0 deps; you ship LICENSE but not NOTICE. Apache-2.0 §4(d) violated.

**Fixed:** aggregate all upstream NOTICE files into your distribution's NOTICE file; ship it.

### Gotcha 10: Static linking LGPL without offering relink

**Broken:** you static-linked LGPL-3.0 library `libfoo.a` into your proprietary `myapp` binary; you ship only `myapp`.

**Fixed:** ship `myapp` + `libfoo.a` + build instructions, OR distribute `myapp` as a dynamically-linked binary that loads `libfoo.so` separately.

### Gotcha 11: Modifying MPL-licensed source file but not preserving header

**Broken:**

```c
// upstream foo.c had:
/* MPL-2.0; Copyright Mozilla */

// you modified it:
// (no header)
```

**Fixed:**

```c
/* SPDX-License-Identifier: MPL-2.0 */
/* Copyright Mozilla */
/* Copyright 2024 You (modifications) */
```

### Gotcha 12: Stripping copyright headers from copied snippets

**Broken:** you grabbed a 50-line snippet from a Stack Overflow answer (CC-BY-SA-4.0) and pasted into your MIT codebase without attribution.

**Fixed:**

```javascript
// Adapted from https://stackoverflow.com/a/12345678
// Original by SO user "alice", licensed CC-BY-SA-4.0
function helperFn() {
  // ...
}
```

Recognize that mixing CC-BY-SA-4.0 code into an MIT codebase creates compatibility ambiguity.

### Gotcha 13: Assuming "package.json says MIT" is authoritative

**Broken:** `package.json` says `"license": "MIT"`. You rely on that.

But the package's `LICENSE` file says GPL-3.0. The author made a mistake. The actual license is GPL-3.0 (the LICENSE file is the canonical source of truth).

**Fixed:** always cross-check `package.json` with the actual `LICENSE` text. License-detection tools like `licensee` look at both.

### Gotcha 14: Bundled binaries with embedded licenses

**Broken:** your release bundle is `myapp.tar.gz` containing `myapp` (a Go binary) + `LICENSE`. You forget the binary itself contains compiled-in copies of dozens of dependencies. Each one's license needs to be discoverable by recipients.

**Fixed:** use `go build` with `-ldflags` to embed license info, OR include a `THIRD_PARTY_NOTICES` file in the tarball listing every dependency's license.

### Gotcha 15: "It's just a tutorial / blog snippet" assumption

**Broken:** copy a 200-line example from a blog post. The blog has no license. You incorporate it.

**Fixed:** reach out to the author, ask for an explicit license (MIT or CC-BY-4.0). Or rewrite the snippet from scratch.

## Idioms

- **"If in doubt, use Apache-2.0 for libraries."** Apache covers patent rights, NOTICE-file mechanism, retaliation, and is OSI/FSF-approved. Lawyers are comfortable with it.
- **"GPL-3.0-or-later for end-user apps you want forks of."** Strong copyleft + future-proofing.
- **"Respect every NOTICE file."** Apache's NOTICE accumulates across deps. Include it.
- **"Automate license-policy enforcement in CI."** `cargo-deny`, `license-checker`, `pip-licenses` all do this. Failing builds beat post-release surprises.
- **"You cannot fork a trademark; you can fork the code."** Code is governed by the FOSS license. Names/brands are governed by trademark law (separate).
- **"`MIT OR Apache-2.0` is the Rust-ecosystem default for a reason."** Best of both worlds: short for users who don't care about patents; explicit-patent for those who do.
- **"AGPL deters enterprise adoption — that's the point."** Don't be surprised when corporate ban-lists exclude AGPL deps.
- **"Source-available is not open source."** BSL/SSPL/Elastic v2/FSL are not FOSS. Treat them as commercial licenses.
- **"Don't strip headers when copying."** Even short copies retain the copyright notice obligation.
- **"`SPDX-License-Identifier:` in every file."** REUSE compliance enables automated audits.
- **"Track your CLAs."** If you want to ever relicense or dual-license, you need contributor permission.
- **"SBOM the output, not just the input."** Modern compliance is about SBOMs accompanying binaries.
- **"License compatibility is asymmetric."** "MIT into GPL" works; "GPL into MIT" doesn't (umbrella becomes GPL).
- **"Upgrade GPL-2.0 to GPL-3.0 if you can."** GPL-3.0 has explicit patent grants and is more compatible with Apache-2.0.
- **"Public-domain claims are jurisdictionally fraught."** Use CC0-1.0 or Unlicense for legal robustness.

## See Also

- `verify` — the math/proof verification model used in cs's detail pages, applies to license-text-fingerprint matching reasoning.
- `cla-vs-dco` — Contributor License Agreements vs Developer Certificate of Origin (deeper coverage of relicensing rights).
- `spdx-identifiers` — full SPDX identifier reference, exception modifiers, and SPDX expression grammar.
- `gdpr` — General Data Protection Regulation (separate compliance domain that often interacts with FOSS deployments).
- `ccpa` — California Consumer Privacy Act (privacy regulation cross-cuts with software-license obligations).
- `gpg` — GnuPG signing for releases (often combined with LICENSE attestation).
- `age` — modern encryption tooling (release-signing alternative).

## References

- Open Source Initiative — opensource.org/licenses — canonical OSI-approved license list with texts.
- SPDX License List — spdx.org/licenses — canonical short identifiers, full texts, exceptions.
- Free Software Foundation — gnu.org/licenses — GPL/LGPL/AGPL/GFDL canonical texts and FAQ.
- choosealicense.com — GitHub-hosted introductory guide to choosing a license.
- tldrlegal.com — plain-English license summaries (not legally binding).
- opensource.com/article/17/9/open-source-licensing — overview article series.
- REUSE — reuse.software — SPDX-header best-practice spec and tooling.
- github.com/spdx/spdx-spec — SPDX specification source-of-truth.
- github.com/spdx/license-list-data — machine-readable SPDX corpus.
- creativecommons.org/licenses — full CC license family.
- Apache Software Foundation — apache.org/legal — Apache CLA, ICLA, CCLA, and license guidance.
- Linux Kernel — kernel.org — DCO (Developer Certificate of Origin) usage example.
- gnu.org/licenses/gpl-faq.html — FSF's GPL FAQ; authoritative on copyleft scope and trigger questions.
- Software Freedom Conservancy — sfconservancy.org — GPL enforcement and compliance guidance.
- FOSSology — fossology.org — FOSS compliance and license-scanning open-source tool.
- ScanCode Toolkit — github.com/nexB/scancode-toolkit — comprehensive license-detection scanner.
- askalono — github.com/jpeddicord/askalono — Rust-native license fingerprinting.
- licensee — github.com/licensee/licensee — Ruby gem powering GitHub's license detection.
- cargo-deny — github.com/EmbarkStudios/cargo-deny — Rust dependency policy enforcement.
- license-checker (npm) — npmjs.com/package/license-checker — JS/TS license auditor.
- pip-licenses — github.com/raimon49/pip-licenses — Python package license auditor.
- CycloneDX — cyclonedx.org — OWASP SBOM standard.
- SBOM Executive Order EO 14028 — whitehouse.gov — US federal SBOM mandate.
- Bruce Perens — perens.com — original Open Source Definition author.
- Eric S. Raymond — catb.org/esr — co-founder of OSI.
- Lawrence Rosen — rosenlaw.com — author of OSL-3.0 and "Open Source Licensing" book.
- Heather Meeker — heathermeeker.com — open source legal practitioner; "Open Source for Business".
- ifrosss.org — ISO/IEC FOSS metadata.
- oss-watch.ac.uk — Higher-ed FOSS guidance (UK).
- fsf.org/licensing — FSF licensing guidance.
- linuxfoundation.org/news/blog — Linux Foundation announcements on licensing.
- BSL 1.1 — mariadb.com/bsl11 — official BSL text.
- SSPL 1.0 — mongodb.com/licensing/server-side-public-license — official SSPL text.
- Elastic License v2 — elastic.co/licensing/elastic-license — official ELv2 text.
- Functional Source License — fsl.software — official FSL text.
- Polyform Project — polyformproject.org — Polyform license family.
- HashiCorp BSL announcement — hashicorp.com/blog/hashicorp-adopts-business-source-license — 2023 license-change rationale.
- OpenTofu — opentofu.org — Linux Foundation fork of Terraform.
- OpenBao — openbao.org — fork of HashiCorp Vault.
- Valkey — valkey.io — Linux Foundation fork of Redis.
- OpenSearch — opensearch.org — AWS-led fork of Elasticsearch + Kibana.
