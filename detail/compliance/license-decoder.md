# License Decoder — Deep Dive

Open-source licensing as legal theory: copyright defaults, freedom semantics, the compatibility lattice, and the formal mathematics of combining licenses into an umbrella whose terms satisfy every input.

## Setup — Copyright as Default; License as Permission Grant; the Contract-vs-License Distinction

Modern copyright law is **automatic**. The instant an author writes original creative work in a tangible medium of expression, copyright attaches — no registration required (Berne Convention 1886, US Copyright Act 1976 §102). For software, this means the moment you type a function into a file, you own the exclusive right to:

- Reproduce the work (copy)
- Prepare derivative works (modify)
- Distribute copies (publish/sell)
- Perform/display publicly (irrelevant for most code, relevant for SaaS UI)

The default state of an unlicensed program is **maximum restriction**: nobody else may legally use, copy, modify, or distribute it. "All rights reserved" is the silent baseline. A program with **no LICENSE file** is functionally undistributable — anyone who clones, builds, or copies it commits prima facie copyright infringement, regardless of whether the source is "publicly visible" on GitHub.

A **license** is a unilateral grant of permission from the rightsholder to the world (or to specific parties), allowing some subset of the exclusive rights to be exercised under stated conditions. It is not a contract in the traditional sense — there is no consideration flowing back to the licensor. The licensee accepts the license by performing an act it permits (e.g., copying), and is bound by its conditions (e.g., reproducing the notice).

The **license vs contract** distinction matters because:

- License: rooted in property law (copyright); breach = infringement (statutory damages, injunctions)
- Contract: rooted in contract law; breach = damages, requires consideration, may be unenforceable for lack of mutual assent

US courts have generally treated the GPL as a license (Jacobsen v. Katzer, Fed. Cir. 2008) — meaning violations are infringement, not contract breach, which gives the rightsholder access to powerful injunctive remedies. EU courts increasingly treat copyleft licenses as having contract-like elements (Welte v. Sitecom, Munich 2004), but the practical enforcement model is similar.

```
Author types code
       ↓
Copyright attaches automatically (no registration)
       ↓
Default: All rights reserved (no one can use)
       ↓
License grants permissions under conditions
       ↓
Licensee accepts by exercising permissions
       ↓
Conditions enforceable as license terms (infringement)
```

The implication for compliance: **every dependency must have a license**. A dependency labeled only "free for personal use" or with no LICENSE file at all is a legal liability regardless of the developer's intent. SBOM tools that flag "no license detected" are flagging genuine legal exposure.

## The Four Freedoms (FSF) — Formal Definition; Freedom 0/1/2/3

The Free Software Foundation defines "free software" via four freedoms (originally articulated by Richard Stallman in the GNU Manifesto, 1985, formalized in 1986):

| Freedom | Statement |
|---------|-----------|
| 0       | The freedom to run the program as you wish, for any purpose |
| 1       | The freedom to study how the program works, and change it so it does your computing as you wish (access to source is precondition) |
| 2       | The freedom to redistribute copies so you can help others |
| 3       | The freedom to distribute copies of your modified versions to others, giving the community a chance to benefit (access to source is precondition) |

The numbering starts at 0 because Stallman is a programmer.

These freedoms are **necessary but not sufficient** to be FSF-approved. The FSF additionally requires that the license not impose conditions incompatible with these freedoms (e.g., field-of-use restrictions break freedom 0; "non-commercial only" breaks freedoms 0/2/3).

Freedom 1 + 3 imply that **source code must be available** — not just to current users, but with provisions for future users to obtain it. This is the seed from which copyleft grows: if you can take my source, modify it, and distribute the binary without source, downstream users lose freedom 1 and 3. Copyleft solves this by requiring derived works to also grant the freedoms.

The FSF maintains a list of [Free Software Licenses](https://www.gnu.org/licenses/license-list.html), classifying each as:

- **GPL-compatible**: can be combined with GPL'd code
- **GPL-incompatible**: free, but cannot be combined with GPL'd code (e.g., Apache 2.0 was GPL-2.0-incompatible until GPL-3.0 fixed it)
- **Non-free**: fails one or more freedoms

```
Freedom 0  →  use for any purpose
Freedom 1  →  study + modify (needs source)
Freedom 2  →  redistribute copies
Freedom 3  →  distribute modified versions (needs source)
                      ↓
            Free software definition
                      ↓
   FSF approves licenses meeting all four
```

## The OSD (Open Source Definition) — OSI's 10-Criterion Definition

The Open Source Initiative was founded in 1998 in part to make "free software" more palatable to enterprises by emphasizing pragmatic benefits over freedom rhetoric. The OSD has 10 criteria, derived from the Debian Free Software Guidelines (1997):

1. **Free Redistribution** — license may not restrict selling or giving away the software as part of an aggregate
2. **Source Code** — must include source or freely available means of obtaining it
3. **Derived Works** — must permit modifications and derived works
4. **Integrity of Author's Source Code** — may require derived works carry different name/version (preserve attribution)
5. **No Discrimination Against Persons or Groups**
6. **No Discrimination Against Fields of Endeavor** — no "non-commercial only" or "no military use" clauses
7. **Distribution of License** — rights apply to all to whom the program is redistributed
8. **License Must Not Be Specific to a Product** — rights cannot depend on being part of a particular distribution
9. **License Must Not Restrict Other Software** — cannot impose license terms on software distributed alongside
10. **License Must Be Technology-Neutral** — no provision predicated on individual technology or interface

OSD vs FSF differences:

| Aspect | FSF | OSI |
|--------|-----|-----|
| Framing | Ethical (freedom is a moral imperative) | Pragmatic (open source produces better software) |
| Scope | Software user's freedom | Distribution model |
| "Non-commercial only" license | Non-free (fails freedom 0) | Not open source (fails OSD #6) |
| Outcome | Both reject "non-commercial only" |

Both definitions reach almost identical sets of approved licenses, but the rhetoric differs. The OSI is the de-facto authority for "is this open source?" — its [License List](https://opensource.org/licenses) is what most policy documents reference.

The OSD is silent on copyleft strength: MIT is OSD-compliant, GPL-3.0 is OSD-compliant, and so is AGPL-3.0. OSD doesn't prefer one over another.

## License Compatibility Lattice — A Formal Partial Order

License compatibility forms a **partial order** (not a total order — most licenses are incomparable). Define:

> A ≤ B iff every condition of A can be simultaneously satisfied by complying with B's conditions.

That is, B's terms are at-least-as-strict as A's. If A ≤ B, you can take A-licensed code, combine it with B-licensed code, and license the combination under B (as long as you preserve A's attribution requirements).

The classic three tiers:

```
                Public Domain / CC0    (lowest — no conditions)
                        |
                        v
              Permissive (MIT, BSD-2/3, Apache-2.0)
                        |
                        v
            Weak Copyleft (LGPL, MPL-2.0, EPL-2.0)
                        |
                        v
            Strong Copyleft (GPL-2.0, GPL-3.0)
                        |
                        v
          Network Copyleft (AGPL-3.0)
```

But this single-chain picture is misleading. The real lattice has multiple branches, e.g.:

```
                       CC0
                        |
              +---------+---------+
              |                   |
             MIT                BSD-3
              |                   |
              +--+-----+-----+----+
                 |     |     |
              MPL-2.0 LGPL  Apache-2.0
                 |     |     |
                 +-----+--+--+
                          |
                       GPL-3.0
                          |
                       AGPL-3.0
```

Apache-2.0 is **incomparable** with GPL-2.0 (Apache's patent retaliation was incompatible with GPL-2.0's "no further restrictions" clause), but Apache-2.0 ≤ GPL-3.0 (GPL-3.0 was specifically rewritten to accept Apache-2.0). Apache-2.0 is **incomparable** with MPL-2.0 (different patent retaliation triggers).

The **least upper bound** (join) of two licenses, when it exists, is the umbrella under which a combination can be released. When no LUB exists, the licenses are **incompatible** — there is no license that simultaneously satisfies both, and the combined work cannot be lawfully distributed.

## Why Compatibility Is Directional

Consider combining MIT-licensed code with GPL-licensed code:

```
  MIT code    GPL code
      \         /
       \       /
        combined
           |
           ?
```

The combination must satisfy:
- MIT's conditions: preserve copyright + license notice
- GPL's conditions: distribute under GPL, provide source, no further restrictions

Can both be satisfied simultaneously? Yes — release the combination under GPL. MIT requires attribution, which GPL allows. GPL requires source distribution, which MIT permits. The MIT-licensed code, when distributed as part of the GPL'd combination, retains its MIT notice but the *combined* binary is governed by GPL.

What about the reverse — combining GPL into a project you want to release as MIT? This is **impossible**:

- GPL requires the entire derivative work be GPL
- MIT permits relicensing
- The intersection (released as MIT) violates GPL

This is why "MIT into GPL works, GPL into MIT loses copyleft":

```
   MIT → GPL:   MIT code keeps MIT notice; GPL absorbs it; combination is GPL
   GPL → MIT:   IMPOSSIBLE — GPL forbids the result
```

The directionality is what makes copyleft "viral": once GPL enters a derivative work's transitive dependency graph, the entire work becomes GPL or undistributable. This is by design — Stallman engineered GPL specifically to propagate freedom through the codebase.

The **System Library Exception** (GPL §1) and **mere aggregation** doctrine soften this:
- Linking against a system library (libc on Linux, kernel32.dll on Windows) doesn't trigger copyleft
- Putting a GPL'd program on a CD next to a proprietary one is "mere aggregation" — they don't form a single derivative work

But these exceptions don't help when GPL'd code is statically linked or directly modified into your codebase.

## The CDDL+GPL Incompatibility — Linux/ZFS

The most famous compatibility failure: ZFS (Sun's filesystem, CDDL) with the Linux kernel (GPL-2.0 only).

**CDDL** (Common Development and Distribution License, 2004):
- File-level copyleft: modifications to CDDL'd files must be CDDL'd
- Combinations with non-CDDL code: fine, as long as CDDL files retain their license
- Strict choice-of-venue clause (Santa Clara County, CA — incompatible with GPL's "no further restrictions")

**GPL-2.0**:
- Whole-program copyleft: derivative works must be GPL-2.0
- "No further restrictions" clause forbids adding any condition not in GPL itself

The conflict:
- CDDL says: ZFS files must remain CDDL
- GPL says: any work derived from kernel must be GPL-2.0
- A modified Linux kernel containing ZFS would need to be both CDDL and GPL-2.0
- These are mutually exclusive — there is no umbrella

Result: ZFS-on-Linux ships as out-of-tree kernel modules, with the legal theory that the **user**, not the distributor, performs the linking. This is contested. Canonical's decision to ship ZFS-on-Linux modules in Ubuntu 16.04+ generated significant legal commentary; Canonical defends the position via the user-link argument and CDDL's permissive linking. The FSF disagrees.

The lesson: **picking copyleft licenses without checking compatibility burns bridges**. CDDL was Sun's Solaris license — they chose it specifically to wall off Solaris from Linux. The mutually-incompatible-copyleft pattern is a feature, not a bug, when ecosystems compete.

## License Compatibility Matrix Theory — Computing the Umbrella License

The **umbrella license** of a combined work is the set of conditions imposed by all components. Formally:

```
Umbrella(W) = ⋂{ valid relicensings of component | component ∈ W }
```

For this intersection to be non-empty, every pair of component licenses must have a common upper bound in the lattice.

For permissive-permissive combinations, the umbrella is the most-restrictive of the inputs (typically Apache-2.0 if any Apache-2.0 component is present, since Apache-2.0 has more conditions than MIT/BSD).

For permissive-with-copyleft combinations, the umbrella is the copyleft license (since it has the most conditions).

For copyleft-with-copyleft combinations, the umbrella may not exist (CDDL+GPL) or may be the strictest copyleft (GPL+AGPL → AGPL).

Compatibility matrix (rows = "your code is", columns = "you incorporate"):

```
              MIT  BSD-3  Apache-2  LGPL-2.1  LGPL-3  MPL-2  GPL-2  GPL-3  AGPL-3
MIT            Y     Y      Y         Y         Y       Y      Y       Y      Y
BSD-3          Y     Y      Y         Y         Y       Y      Y       Y      Y
Apache-2       Y*    Y*     Y         N         Y       Y      N       Y      Y
LGPL-2.1       N     N      N         Y         Y       N      Y       N      N
LGPL-3         N     N      N         N         Y       N      N       Y      Y
MPL-2          N     N      N         N         N       Y      Y       Y      Y
GPL-2          N     N      N         N         N       N      Y       N      N
GPL-3          N     N      N         N         N       N      N       Y      Y
AGPL-3         N     N      N         N         N       N      N       N      Y
```

(* with notice preservation — Apache requires NOTICE file to propagate.)

The matrix has empty cells where licenses are incompatible. A real compliance pipeline computes the transitive closure of dependencies and verifies the matrix-implied umbrella exists.

## Network-Copyleft Theory (AGPL) — The SaaS Loophole

GPL's copyleft trigger is **distribution**: if you distribute a binary, you must distribute the source. SaaS exposes a hole: a company runs modified GPL software on its servers, never distributes the binary, and avoids the source-disclosure obligation. Users interact with the modifications via HTTP but never receive the binary or source.

```
   Without AGPL:
     Modify GPL code → run on server → users use over network → no source obligation
   With AGPL §13:
     Modify AGPL code → run on server → users use over network → MUST offer source
```

AGPL-3.0 §13:

> Notwithstanding any other provision of this License, if you modify the Program, your modified version must prominently offer all users interacting with it remotely through a computer network (if your version supports such interaction) an opportunity to receive the Corresponding Source of your version by providing access to the Corresponding Source from a network server at no charge, through some standard or customary means of facilitating copying of software.

The trigger is **modification + network interaction** — using unmodified AGPL software on a server doesn't trigger §13. But any patch, customization, or wrapper that "modifies" the program does.

Operational consequences:
- MongoDB pre-2018 was AGPL — using it server-side was fine; modifying its code and running modifications as a service triggered §13
- MongoDB switched to SSPL in 2018 specifically because AWS was running an unmodified-MongoDB managed service that didn't trigger AGPL §13
- SSPL (Server Side Public License) extends to "all programs you use to make the software available to third parties" — much broader, OSI rejected it

The AGPL is the strongest standard copyleft, and it's deliberately so. Picking AGPL signals "I want my code to remain free even when used as a service."

## Patent Grant Theory — Implicit vs Explicit

Software is protected by **three** legal regimes simultaneously: copyright (the expression), patent (the invention), and trademark (the brand). License compatibility analysis often focuses on copyright but patents are increasingly load-bearing.

**Implicit patent license**: Some courts hold that a copyright license to use software implies a patent license to whatever patents the licensor holds covering that use (estoppel / implied license doctrine). But this is jurisdictionally inconsistent and weak.

**Explicit patent grant**: Modern licenses (Apache-2.0, MPL-2.0, GPL-3.0) include explicit patent grants. Apache-2.0 §3:

> Each Contributor hereby grants to You a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable (except as stated in this section) patent license to make, have made, use, offer to sell, sell, import, and otherwise transfer the Work...

This solves a problem the original GPL-2.0 didn't address: a contributor can have a copyrighted contribution accepted, then sue users for patent infringement covering that very contribution.

**Patent retaliation / termination**: Apache-2.0 §3 continues:

> ...If You institute patent litigation against any entity (including a cross-claim or counterclaim in a lawsuit) alleging that the Work or a Contribution incorporated within the Work constitutes direct or contributory patent infringement, then any patent licenses granted to You under this License for that Work shall terminate as of the date such litigation is filed.

This is the patent peace clause: filing a patent suit against the project terminates your license. Major licenses with patent peace:

- Apache-2.0 (broad — terminates all patent licenses on filing)
- GPL-3.0 (terminates on patent suit related to the Program)
- MPL-2.0 (terminates only patent grants from the entity sued)
- BSD-3-Clause: no patent grant or peace clause (relies on implicit grant)
- MIT: no patent grant or peace clause

The "Apache 2.0 + GPL 2.0 incompatibility" was rooted partly in patent terms — Apache's termination clause was an "additional restriction" GPL-2.0 forbade. GPL-3.0 §11 added a patent grant + peace, fixing the incompatibility going forward.

## Trademark vs Copyright vs Patent — The Three Regimes

| Regime | What It Protects | Duration | Origin |
|--------|------------------|----------|--------|
| Copyright | Original expression (code, text, images) | Life + 70 (US, individual); 95 from publication (corporate) | Automatic on fixation |
| Patent | Inventions (process, machine, manufacture, composition) | 20 years from filing | Granted on examination |
| Trademark | Brand identifiers (names, logos) | Indefinite (with use) | Use + registration |

Open source licenses generally **don't license trademarks**. Apache-2.0 §6:

> This License does not grant permission to use the trade names, trademarks, service marks, or product names of the Licensor, except as required for describing the origin of the Work and reproducing the content of the NOTICE file.

This is why "fork the code, not the name" is the default: you can fork Linux into Linus's-favorite-kernel, but you can't call it "Linux" without license from the trademark holder. Mozilla famously had to call the Debian-rebuilt Firefox "Iceweasel" because Mozilla wouldn't license the Firefox trademark to Debian. Mozilla relented in 2016.

Trademark protects users from confusion about origin. A fork of "Redis" cannot ship as "Redis" because users would think it's the original. Hence "Valkey" (the fork after Redis's 2024 license change), "OpenSearch" (fork of Elasticsearch).

Patent protection bites at the *implementation* level: a license to use copyrighted code doesn't help if a third-party patent reads on what the code does. Defensive patent pools (OIN — Open Invention Network) and patent peace clauses are how the FOSS ecosystem manages this.

## License Proliferation History

The 1990s and early 2000s saw a Cambrian explosion of "vanity licenses" — every project felt compelled to write its own. By 2005 OSI had approved ~60 licenses, with little real difference between most of them.

OSI established the **License Proliferation Committee** in 2005. Goals:
- Reduce license count
- Identify "preferred" licenses
- Mark redundant licenses for retirement

Output: licenses categorized as Popular/Special-Purpose/Redundant/Non-Reusable. Most "redundant" licenses are still OSI-approved but not recommended.

Convergence over the 2010s:
- New projects default to MIT (simple, well-understood)
- Corporate-led projects pick Apache-2.0 (patent grant)
- Copyleft projects pick GPL-3.0 / AGPL-3.0
- The middle ground (LGPL, MPL) shrinks — used mostly for libraries

Modern reality: ~95% of open source code lives under <10 licenses. Bespoke licenses are flagged by SBOM tools and trigger manual legal review.

```
2000:  60+ OSI-approved licenses, picked ~uniformly
2025:  ~10 licenses cover ~95% of public packages
       Bespoke license = "we're hostile to enterprise"
```

## Source-Available Licenses — Not OSI-Compliant

Beyond OSI-approved open source: **source-available** licenses ship code but restrict use, typically forbidding competing-with-vendor commercial offerings.

**Business Source License (BSL / BUSL-1.1)**:
- Source visible, modifications permitted
- Production use with restrictions defined per-product (e.g., "cannot offer as managed service")
- Time-bomb: converts to permissive (Apache-2.0 typical) after Change Date (often 4 years)
- Used by: HashiCorp Terraform (2023→), CockroachDB, MariaDB Enterprise, Sentry

**Server Side Public License (SSPL)**:
- AGPL §13 + section 13 expanded to "all programs you use to make the software available"
- "You must release ALL the supporting infrastructure source"
- OSI rejected it in 2018 — fails OSD #6 (discriminates against SaaS providers)
- Used by: MongoDB (2018→), Elastic (2021→)

**Elastic License v2**:
- Source visible, modifications permitted for self-use
- Cannot offer as managed service competing with Elastic
- Cannot circumvent license keys
- Not OSI-approved, "free for most users, restricted for cloud providers"

**Functional Source License (FSL)**:
- Sentry's variant, similar to BSL
- 2-year time-bomb to Apache-2.0 or MIT
- Restricts competing commercial offerings during the 2-year window

The "rented commercial use" model: source visible, hack on it, but pay the vendor or change it after the time-bomb. Critics argue it's not open source; defenders argue it's economically necessary to fund development given hyperscaler "exploitation" of permissive code.

```
   Permissive (MIT/Apache)        — anyone can fork + sell
   Copyleft (GPL/AGPL)            — anyone can fork, must keep open
   Network-copyleft (AGPL)        — same, plus SaaS source
   Source-available (BSL/SSPL)    — read source, restricted production
   Proprietary                    — no source visibility
```

## Dual-Licensing Theory

Copyright holder can license the same code under multiple terms. Common pattern:

- License A: GPL-3.0 (free, but copyleft = enterprises hesitant)
- License B: Commercial (paid, no copyleft restrictions)

The user picks: comply with GPL or pay for commercial.

For dual-licensing to work, the rightsholder must own all the copyright. With contributions, this requires either:
- **Copyright Assignment Agreement (CAA)** — contributors transfer copyright to project
- **Contributor License Agreement (CLA)** — contributors grant the project broad relicensing rights

Without one of these, contributor copyrights are co-owned, and dual-licensing requires every contributor's permission for each license sale.

Examples:
- MySQL — GPL + commercial (Oracle owns all copyright via CLA)
- Qt — LGPL + commercial (The Qt Company holds rights via CLA)
- MongoDB pre-2018 — AGPL + commercial (10gen/MongoDB Inc. via CLA)

The dual-license model funded much of the early FOSS-business era. It's largely been replaced by SaaS hosting (cloud providers undercut on-premise commercial licenses) and source-available licenses (which avoid the copyleft trigger).

## The Compliance Audit Pipeline

A modern compliance pipeline:

```
  Source code → SBOM generation → License extraction → Policy evaluation → Action
```

Stages:

**SBOM Generation**:
- Tools: Syft, CycloneDX CLI, SPDX tools
- Scans manifests (package.json, go.mod, Cargo.toml, requirements.txt)
- Resolves transitive dependencies
- Outputs SPDX or CycloneDX format

**License Extraction**:
- Per-package: query registry metadata (npm, PyPI), parse LICENSE files in source, run scancode-toolkit
- Confidence scoring: high (SPDX ID in package metadata) / medium (matched license file) / low (heuristic)
- Multiple license detection: dual-licensed packages, license-per-file repos

**Policy Evaluation**:
- Allow-list: licenses approved for use (typically MIT, Apache-2.0, BSD-3, ISC)
- Block-list: licenses forbidden (often GPL, AGPL for proprietary projects)
- Conditional: licenses requiring legal review (LGPL — depends on linking model)
- Output: pass/warn/fail per dependency

**Action**:
- Block CI on policy violation
- File ticket for legal review
- Auto-generate NOTICE file
- Update OSS attribution page

Commercial tools:
- **FOSSA** — full-stack SBOM + policy + remediation, cloud SaaS
- **Black Duck** (Synopsys) — long-standing enterprise leader, on-prem option
- **Snyk License** — bolted onto vulnerability scanning
- **ClearlyDefined** — community + Microsoft-funded, free

Open source tools:
- **scancode-toolkit** — license + copyright extraction
- **license-checker** (npm) — surface package license fields
- **go-licenses** — Go module license analysis
- **REUSE** — FSFE's standard for in-repo license declaration

## Risk Tiers — Compliance Failure Modes

License compliance failures span warn-to-lawsuit. Risk tiers:

**Tier 1 — Warning / cease-and-desist**:
- Most violations resolved at this stage
- Plaintiff: license author or compliance organization (gpl-violations.org, Software Freedom Conservancy)
- Action: provide source, comply prospectively
- Cost: low (legal fees, engineering hours)

**Tier 2 — Audit + remediation**:
- Triggered by formal compliance demand or regulatory inspection
- Often via SBOM mismatch (claimed vs actual)
- Action: full source delivery, audit certification, sometimes monetary settlement
- Cost: medium (typically $50K-$500K)

**Tier 3 — Lawsuit**:
- Plaintiff seeks injunction + damages + fees
- Cisco/Linksys + BusyBox (2008) — settled, full GPL compliance, $X donation
- Verizon + BusyBox (2008) — settled
- Welte v. Sitecom (2004, Germany) — first major GPL court decision, injunction granted
- Artifex v. Hancom (2017, US) — first US GPL contract-theory ruling
- McHardy + iptables (2010s, Germany) — controversial enforcement, ~50 settlements
- Vizio v. SFC (2021, ongoing) — third-party beneficiary theory test case

Examples:

**BusyBox** — small embedded utilities under GPL-2.0. Licensed by countless router/embedded vendors who shipped binaries without source. Erik Andersen and SFC pursued litigation 2007-2012, won settlements with Cisco/Linksys, Verizon, Best Buy, Westinghouse.

**Cisco/Linksys 2008** — Cisco acquired Linksys and inherited GPL-violating firmware. SFC sued. Cisco settled, paid undisclosed donation, hired GPL compliance officer, full source release.

**Vizio 2021–** — SFC sued Vizio claiming third-party beneficiary status under GPL-2.0. If consumers (not just rightsholders) can sue, it broadens enforcement dramatically. Case ongoing in California; partial wins for SFC on jurisdiction.

**McHardy iptables** — Patrick McHardy holds copyright on iptables code, pursued aggressive German litigation. Many in the kernel community criticized as "trolling," but the legal theory is sound. Linux Foundation issued statement on enforcement principles.

The **statutory damages** ceiling under US copyright is $150,000/work for willful infringement. With software's modular nature, "per-package violation" math can climb fast.

For corporate users, the practical cost of a Tier-2 incident is typically the engineering time to retrofit compliant practices + reputational damage + customer churn (enterprise customers may insist on SBOM/compliance certifications post-incident).

## References

- FSF, "What is Free Software?" — https://www.gnu.org/philosophy/free-sw.html
- OSI, "The Open Source Definition" — https://opensource.org/osd/
- OSI, "Licenses & Standards" — https://opensource.org/licenses
- SPDX License List — https://spdx.org/licenses/
- GPL-3.0 text — https://www.gnu.org/licenses/gpl-3.0.html
- AGPL-3.0 text — https://www.gnu.org/licenses/agpl-3.0.html
- Apache-2.0 text — https://www.apache.org/licenses/LICENSE-2.0
- Jacobsen v. Katzer, 535 F.3d 1373 (Fed. Cir. 2008)
- Welte v. Sitecom, Munich District Court 2004
- SFC v. Vizio, complaint and rulings — https://sfconservancy.org/copyleft-compliance/vizio.html
- Heather Meeker, "Open (Source) for Business: A Practical Guide to Open Source Software Licensing"
- Lawrence Rosen, "Open Source Licensing: Software Freedom and Intellectual Property Law"
- Larry Wall, "The Three Virtues of a Programmer" — context for permissive license preference
- ZFS-on-Linux licensing FAQ — https://zfsonlinux.org/license.html
- HashiCorp BSL adoption announcement (2023) — https://www.hashicorp.com/license-faq
- MongoDB SSPL FAQ — https://www.mongodb.com/licensing/server-side-public-license/faq
- ClearlyDefined — https://clearlydefined.io/
- REUSE Specification — https://reuse.software/spec/
- gpl-violations.org archive — https://gpl-violations.org/
- Software Freedom Conservancy enforcement — https://sfconservancy.org/copyleft-compliance/
- OIN (Open Invention Network) — https://www.openinventionnetwork.com/
- TLDRLegal license summaries (informational, not legal advice) — https://www.tldrlegal.com/
- Choose A License (GitHub) — https://choosealicense.com/
