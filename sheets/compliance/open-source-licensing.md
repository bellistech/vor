# Open Source Licensing (Software License Compliance)

A comprehensive guide to open source license types, compatibility, obligations, and compliance tooling, covering the spectrum from permissive licenses (MIT, BSD, Apache-2.0) through weak copyleft (LGPL, MPL-2.0) to strong copyleft (GPL-2.0, GPL-3.0, AGPL-3.0), with SPDX identifiers, patent grants, and contributor agreements.

## License Categories
### Permissive vs Copyleft Spectrum
```
PERMISSIVE ←————————————————————————————→ STRONG COPYLEFT

MIT    BSD-2   BSD-3   Apache-2.0   MPL-2.0   LGPL   GPL-2.0   GPL-3.0   AGPL-3.0
 |       |       |        |           |        |       |          |         |
 No copyleft obligation   Patent      Weak     Weak   Strong     Strong    Network
 Attribution only         grant       copyleft copyleft copyleft  copyleft  copyleft
                          Patent      File-    Library  Entire    Entire    Entire
                          retaliation level    linking  work      work +    work +
                                      only     exception          patent    network
                                                                  grant     use

Key distinction:
  Permissive: Use in proprietary software, just give attribution
  Weak copyleft: Modified files must be open; rest can be proprietary
  Strong copyleft: Entire derivative work must use same license
  Network copyleft: Even SaaS/server use triggers source disclosure
```

## Major Licenses — Detailed Breakdown
### MIT License (SPDX: MIT)
```
Permissions:           Commercial use, modification, distribution,
                       private use
Conditions:            Include copyright notice and license text
Limitations:           No liability, no warranty

Full text is ~170 words. The most popular open source license.

Key clause: "Permission is hereby granted, free of charge, to
any person obtaining a copy of this software and associated
documentation files, to deal in the Software without restriction,
including without limitation the rights to use, copy, modify,
merge, publish, distribute, sublicense, and/or sell copies..."

No patent grant (implicit at best, legally ambiguous).
No contributor agreement required.
Compatible with virtually all other licenses.
```

### Apache License 2.0 (SPDX: Apache-2.0)
```
Permissions:           Commercial use, modification, distribution,
                       patent use, private use
Conditions:            Include license, state changes, include NOTICE
Limitations:           No liability, no warranty, no trademark rights

Key differentiators from MIT:
  1. Explicit patent grant (Section 3)
  2. Patent retaliation clause (Section 3 — terminate patent
     license if you file patent litigation)
  3. NOTICE file preservation requirement
  4. State changes requirement for modified files
  5. Contribution clause (Section 5 — contributions under
     the same license unless stated otherwise)

Compatible with GPL-3.0 but NOT GPL-2.0
  (FSF confirmed Apache-2.0 is GPL-3.0-compatible)
```

### GPL-2.0 (SPDX: GPL-2.0-only / GPL-2.0-or-later)
```
Permissions:           Commercial use, modification, distribution
Conditions:            Disclose source, same license, state changes,
                       include license text
Limitations:           No liability, no warranty

Key obligations:
  1. Source code must be made available for any binary distribution
  2. Derivative works must be licensed under GPL-2.0
  3. "Or later" clause: GPL-2.0-or-later allows upgrading to GPL-3.0
  4. No explicit patent grant (unlike GPL-3.0)
  5. Linking creates a derivative work (contested but FSF position)

Tivoization: GPL-2.0 does NOT prevent hardware restrictions
  on running modified versions (GPL-3.0 added this protection)

Linux kernel uses GPL-2.0-only (not "or later")
```

### GPL-3.0 (SPDX: GPL-3.0-only / GPL-3.0-or-later)
```
Additional protections over GPL-2.0:
  1. Explicit patent grant (Section 11)
  2. Anti-tivoization (Section 6 — installation information
     for consumer devices must be provided)
  3. Anti-circumvention (cannot use GPL software to enforce DRM
     without allowing modification)
  4. Compatible with Apache-2.0
  5. Affero clause compatibility (can combine with AGPL)
  6. Explicit termination and cure provisions (Section 8)
```

### AGPL-3.0 (SPDX: AGPL-3.0-only / AGPL-3.0-or-later)
```
Extends GPL-3.0 with network copyleft (Section 13):

"If you modify the Program, your modified version must
prominently offer all users interacting with it remotely
through a computer network an opportunity to receive the
Corresponding Source of your version..."

Impact: Running AGPL software as a web service triggers
the source disclosure obligation, even without distribution.

This closes the "SaaS loophole" in GPL-3.0.
Companies frequently ban AGPL from their codebases entirely.
```

### LGPL (SPDX: LGPL-2.1-only / LGPL-3.0-only)
```
"Lesser" GPL — weak copyleft for libraries:

  - Library itself must remain LGPL (modifications disclosed)
  - Programs that LINK to the library can be any license
  - Must allow relinking (dynamic linking preferred)
  - Must provide mechanism for users to substitute LGPL library
  - Static linking may trigger additional obligations

Common uses: glibc (LGPL-2.1), Qt (LGPL-3.0 / commercial dual)
```

### MPL-2.0 (SPDX: MPL-2.0)
```
Mozilla Public License — file-level copyleft:

  - Modified MPL files must remain MPL (source disclosed)
  - New files you add can be any license
  - Larger work can be proprietary
  - Explicit patent grant
  - GPL-2.0+ and LGPL-2.1+ compatible (Section 3.3)
  - No anti-tivoization clause

Practical effect: You can use MPL libraries in proprietary
software as long as you do not modify the MPL-licensed files
themselves (or disclose modifications if you do).
```

### BSD Licenses (SPDX: BSD-2-Clause / BSD-3-Clause)
```
BSD 2-Clause ("Simplified"):
  - Redistribution in source and binary forms permitted
  - Must retain copyright notice and disclaimer
  - No endorsement clause

BSD 3-Clause ("New" / "Revised"):
  - Same as 2-Clause plus:
  - "Neither the name of the copyright holder nor the names
    of its contributors may be used to endorse or promote
    products derived from this software without specific
    prior written permission."

BSD 4-Clause ("Original") — DEPRECATED:
  - Added advertising clause (must acknowledge in all ads)
  - Incompatible with GPL — avoid using
```

## License Compatibility Matrix
### Can You Combine These?
```
             MIT  BSD  Apache  MPL  LGPL  GPL2  GPL3  AGPL
MIT           Y    Y    Y      Y    Y     Y     Y     Y
BSD-3         Y    Y    Y      Y    Y     Y     Y     Y
Apache-2.0    Y    Y    Y      Y    Y     N*    Y     Y
MPL-2.0       Y    Y    Y      Y    Y     Y     Y     Y
LGPL-2.1      Y    Y    Y      Y    Y     Y     Y     Y
GPL-2.0       N**  N**  N*     N**  N**   Y     N***  N
GPL-3.0       N**  N**  N**    N**  N**   N***  Y     Y
AGPL-3.0      N**  N**  N**    N**  N**   N     N**   Y

Y  = Can combine; result under more restrictive license
N* = Apache-2.0 has patent clause incompatible with GPL-2.0
N**= Copyleft requires derivative to use same license
N***= GPL-2.0-only incompatible with GPL-3.0; "or later" resolves

Direction: Permissive code can flow INTO copyleft projects,
           but copyleft code cannot flow into permissive projects.
```

## SPDX License Identifiers
### Standard Expressions
```bash
# SPDX (Software Package Data Exchange) license expressions
Simple identifier:
  SPDX-License-Identifier: MIT
  SPDX-License-Identifier: Apache-2.0
  SPDX-License-Identifier: GPL-3.0-only

"Or later" variants:
  GPL-2.0-or-later    (replaces GPL-2.0+)
  LGPL-3.0-or-later   (replaces LGPL-3.0+)

Compound expressions:
  MIT AND Apache-2.0           (both licenses apply)
  MIT OR Apache-2.0            (choice between licenses)
  GPL-2.0-only WITH Classpath-exception-2.0  (license + exception)
  (MIT AND BSD-2-Clause) OR Apache-2.0       (grouped)

File header format:
  // SPDX-License-Identifier: Apache-2.0

Package metadata (package.json):
  "license": "MIT"
  "license": "(MIT OR Apache-2.0)"

SPDX license list: https://spdx.org/licenses/
  ~500 licenses with standardized identifiers
```

## Contributor Agreements
### CLA vs DCO
```
CLA (Contributor License Agreement):
  - Legal agreement between contributor and project
  - Grants project broad rights to contributions
  - May include patent grant
  - Can be individual CLA or corporate CLA
  - Examples: Apache ICLA/CCLA, Google CLA, Microsoft CLA
  - Controversial: some view as power imbalance
  - Tools: CLA Assistant, EasyCLA

DCO (Developer Certificate of Origin):
  - Lightweight alternative to CLA
  - Contributor certifies they have the right to submit
  - Implemented via Signed-off-by line in commits
  - Created by Linux Foundation
  - No IP assignment — just certification of rights

Usage:
  git commit --signoff -m "Add feature X"
  # Adds: Signed-off-by: Name <email>

Projects using DCO: Linux kernel, GitLab, Chef
Projects using CLA: Apache, Kubernetes, Terraform
```

## License Scanning Tools
### Automated Compliance
```bash
# FOSSA — commercial SCA with license compliance
# Integrates with CI/CD, supports 20+ package managers
fossa analyze                  # scan project
fossa test                     # fail CI if license issues
fossa report --type attribution  # generate attribution report

# Snyk Open Source — vulnerability + license scanning
snyk test                      # scan for vulns and license issues
snyk monitor                   # continuous monitoring

# licensee (Ruby gem) — detect project license
gem install licensee
licensee detect .              # detect license of current project

# go-licenses (Go) — scan Go module dependencies
go install github.com/google/go-licenses@latest
go-licenses check ./...        # check all dependencies
go-licenses csv ./...          # CSV report of all licenses
go-licenses save ./... --save_path=THIRD_PARTY  # save license texts

# ScanCode Toolkit — comprehensive FOSS scanner
pip install scancode-toolkit
scancode -clpieu --json-pp output.json /path/to/code

# FOSSology — self-hosted license compliance platform
# Web UI for upload, scan, report, and clear licenses

# OSS Review Toolkit (ORT)
# End-to-end: Analyze → Scan → Evaluate → Report
ort analyze -i /project -o /results
ort scan -i /results/analyzer-result.yml -o /results
ort evaluate -i /results/scan-result.yml -o /results
ort report -i /results/evaluation-result.yml -o /results -f WebApp
```

## Dual Licensing and Business Models
```
Dual Licensing:
  - Offer same software under two licenses
  - Open source license (e.g., AGPL) for community
  - Commercial license for proprietary use
  - Examples: MySQL (GPL/Commercial), Qt (LGPL/Commercial),
    MongoDB (SSPL/Commercial), Elasticsearch (SSPL/Elastic)

Source-Available (not Open Source per OSI):
  - SSPL (Server Side Public License) — MongoDB
  - BSL (Business Source License) — MariaDB, HashiCorp
  - Elastic License 2.0 — Elasticsearch
  - These restrict certain commercial uses
  - OSI does not consider them "open source"

Open Core:
  - Core product is open source
  - Premium features under commercial license
  - Examples: GitLab (MIT core / proprietary EE),
    Grafana (AGPL core / proprietary Enterprise)
```

## Tips
- Always check the SPDX license identifier of every dependency before adding it to your project; an AGPL transitive dependency can require disclosure of your entire application source
- Use automated license scanning in CI/CD pipelines to catch license violations early; manually tracking licenses across hundreds of dependencies is unsustainable
- Understand the difference between "GPL-2.0-only" and "GPL-2.0-or-later" as they have very different compatibility implications, especially with GPL-3.0
- Apache-2.0 is generally safer than MIT for commercial projects because of its explicit patent grant and retaliation clause
- When using LGPL libraries, prefer dynamic linking over static linking to avoid triggering the full copyleft obligation
- Maintain a THIRD_PARTY or NOTICES file listing all dependencies, their licenses, and copyright holders; this satisfies attribution requirements for permissive licenses
- If your company bans certain licenses, encode those rules in your license scanning tool and fail the build automatically
- Be aware that the FSF and OSI have different definitions of "free" and "open source"; check both organizations' positions for contested licenses like SSPL
- The DCO (Developer Certificate of Origin) is a lighter alternative to CLAs for accepting contributions; use git commit --signoff to add the sign-off
- Review license compatibility before combining libraries; permissive code can flow into copyleft projects but copyleft code cannot flow into permissive ones
- When in doubt about a license obligation, consult legal counsel; the cost of a legal opinion is trivial compared to a license violation lawsuit

## See Also
- gdpr, soc2, nist

## References
- [SPDX License List](https://spdx.org/licenses/)
- [OSI Approved Licenses](https://opensource.org/licenses/)
- [FSF License List](https://www.gnu.org/licenses/license-list.html)
- [Choose a License](https://choosealicense.com/)
- [FOSSA Documentation](https://docs.fossa.com/)
- [Google go-licenses](https://github.com/google/go-licenses)
- [Linux Foundation DCO](https://developercertificate.org/)
- [TLDRLegal — License Summaries](https://tldrlegal.com/)
