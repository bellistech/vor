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

## Worked Compatibility Chains — MIT → GPL → AGPL

Walk through the canonical permissive-to-network-copyleft chain step by step.

**Step 1: MIT library `libfoo` written in 2018, MIT-licensed.**

`libfoo` source ships with the MIT notice. Anyone may copy, modify, sublicense. The only condition: include the copyright notice and disclaimer of warranty.

**Step 2: Project `barproject` adopts `libfoo`, releases under GPL-3.0.**

`barproject` imports `libfoo`. The combination is a derivative work. To distribute, `barproject` must:
- Retain `libfoo`'s MIT notice (probably in a `THIRD_PARTY_NOTICES` file or per-file headers)
- License the *combination* under GPL-3.0
- Provide source for the entire combination on request
- Add no further restrictions beyond GPL-3.0

Result: downstream users get a GPL-3.0 work that contains MIT-licensed code. They may extract `libfoo` from the source tree and use it under MIT (the original MIT grant flows to them directly — GPL-3.0 cannot revoke an upstream MIT grant). But the *combined* binary is GPL-3.0.

**Step 3: SaaS `bazcloud` modifies `barproject`, runs it as a service, never distributes binaries.**

GPL-3.0's distribution trigger is not met (no binary leaves bazcloud's servers). Users interact via HTTP. GPL-3.0 imposes no source-disclosure obligation. This is the SaaS loophole AGPL closes.

**Step 4: `bazcloud` adopts AGPL-3.0 to plug the hole.**

`bazcloud` relicenses *its modifications* to AGPL-3.0. But the underlying `barproject` is GPL-3.0. Is GPL-3.0 → AGPL-3.0 allowed?

Yes — GPL-3.0 §13 explicitly permits combination with AGPL-3.0. The text:

> Notwithstanding any other provision of this License, you have permission to link or combine any covered work with a work licensed under version 3 of the GNU Affero General Public License into a single combined work, and to convey the resulting work.

So the chain MIT → GPL-3.0 → AGPL-3.0 works. Each transition raises the copyleft ceiling. Downstream of `bazcloud`, network users may obtain the source via the §13 obligation.

**Step 5: Try to relicense back to MIT.**

Impossible. Once GPL-3.0 absorbs the work, MIT distribution would require removing the GPL portions and re-implementing — a clean-room rewrite. The original `libfoo` can still be extracted and used as MIT, but `barproject` and `bazcloud` modifications are stuck at GPL-3.0+.

```
MIT (libfoo)           [permissive]
    ↓
GPL-3.0 (barproject)   [strong copyleft, distribution trigger]
    ↓
AGPL-3.0 (bazcloud)    [network copyleft, SaaS trigger]
    ↓
[stuck — cannot return to MIT or weaker]
```

The lesson: copyleft is a one-way valve. Plan upstream license choices with downstream possibilities in mind.

## Worked Compatibility — Apache 2.0 → GPL-3.0 via Patent Grant

Apache-2.0 + GPL-3.0 was a deliberate compatibility win in 2007. Walk through the analysis.

**Apache-2.0 §3** grants every contributor's downstream a patent license for "those patent claims licensable by such Contributor that are necessarily infringed by their Contribution(s) alone or by combination of their Contribution(s) with the Work." Plus patent-retaliation: filing a patent suit against the work terminates your patent grant.

**GPL-2.0 §6** said:

> Each time you redistribute the Program (or any work based on the Program), the recipient automatically receives a license from the original licensor to copy, distribute or modify the Program subject to these terms and conditions. You may not impose any further restrictions on the recipients' exercise of the rights granted herein.

The "no further restrictions" clause meant Apache-2.0's patent retaliation — terminating patent grants on suit — counted as an "additional restriction" GPL-2.0 forbade. Combination was thus forbidden under GPL-2.0.

**GPL-3.0 §11** rewrote this. It added its own patent grant and patent-retaliation clause, then in §7 explicitly listed Apache-2.0-style patent retaliation as an "additional permission" that GPL-3.0 tolerates. The §7 language:

> Notwithstanding any other provision of this License, for material you add to a covered work, you may (if authorized by the copyright holders of that material) supplement the terms of this License with terms... [including] limitations of liability or warranty different from those of section 15 or 16.

Plus §11 codified that GPL-3.0 itself imposes a patent-retaliation clause similar to Apache's.

Result: Apache-2.0 + GPL-3.0 combination is allowed. Apache's patent retaliation is no longer "additional" because GPL-3.0 has its own patent terms.

**Operational rule for compliance pipelines:**

```
if any(d.license == "Apache-2.0" for d in deps) and target == "GPL-2.0":
    fail("Apache-2.0 incompatible with GPL-2.0 due to patent terms")
if any(d.license == "Apache-2.0" for d in deps) and target == "GPL-3.0":
    pass  # compatible since GPL-3.0 §7/§11
```

This single 2007 fix unlocked thousands of downstream combinations (Hadoop + GNU stack, Spark + Linux tools, Kubernetes + GPL-licensed components).

## CDDL+GPL Impossibility Proof

The most famous compatibility failure: ZFS (Sun's filesystem, CDDL) with the Linux kernel (GPL-2.0 only). Construct a formal impossibility proof.

**Premise 1: CDDL §3.4** states modifications to CDDL-licensed files must remain CDDL.

> All Covered Software, including any Modifications, that You distribute or otherwise make available in Executable form must... be made available in Source Code form... and that Source Code form must be distributed only under the terms of this License.

**Premise 2: CDDL §6.3** has a venue clause:

> Any litigation relating to this License may be brought only in the courts of a jurisdiction wherein the defendant maintains its principal place of business...

**Premise 3: GPL-2.0 §6** has the no-further-restrictions clause forbidding any additional condition not in GPL-2.0 itself.

**Premise 4: GPL-2.0 §2(b)** requires that any work derived from a GPL-2.0 work be licensed under GPL-2.0 in its entirety.

**Combination attempt:** Build a kernel module by modifying the Linux kernel and incorporating ZFS source.

The combined work must satisfy:
- CDDL: ZFS files remain CDDL (Premise 1)
- GPL-2.0: combined work licensed as GPL-2.0 in entirety (Premise 4)

These are **mutually exclusive**: a single file cannot simultaneously be "CDDL only" (per CDDL §3.4) and "GPL-2.0" (per GPL-2.0 §2(b)). The umbrella license set is empty.

Furthermore, even if Premise 4 were relaxed (e.g., file-level rather than work-level analysis), CDDL's venue clause (Premise 2) is an "additional restriction" beyond what GPL-2.0 imposes, violating Premise 3. Independent failure path.

**Conclusion:** No umbrella license satisfies both CDDL and GPL-2.0. Combination is impossible.

ZFS-on-Linux ships as out-of-tree kernel modules. The legal theory: the **user**, not the distributor, performs the linking at runtime via `modprobe`. This shifts the derivative-work formation from distributor to end-user, who is not bound by GPL-2.0's distribution conditions because they aren't distributing.

Canonical's decision to ship ZFS-on-Linux modules pre-built in Ubuntu 16.04+ generated significant legal commentary. SFLC and Bradley Kuhn argue Canonical violates GPL-2.0; Canonical's counterargument cites CDDL's permissive linking and the user-as-linker theory. The matter has not been litigated to judgment.

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

## Apache 2.0 §3 vs GPL 3.0 §11 — Patent Clause Anatomy

Both provisions cover the same ground (grant patents on contribution; terminate on suit) but with subtly different scopes.

**Apache 2.0 §3 — scope of grant:**
- Grants apply to "those patent claims licensable by such Contributor that are necessarily infringed by their Contribution(s) alone or by combination of their Contribution(s) with the Work to which such Contribution(s) was submitted"
- Two sub-scopes: contribution alone, or contribution + Work
- Does NOT cover patents reading on combinations of the contribution with code outside the Work (e.g., a downstream fork that adds new functionality)

**GPL-3.0 §11 — scope of grant:**

> Each contributor grants you a non-exclusive, worldwide, royalty-free patent license under the contributor's essential patent claims, to make, use, sell, offer for sale, import and otherwise run, modify and propagate the contents of its contributor version.

Where "essential patent claims" are claims "owned or controlled by the contributor, whether already acquired or hereafter acquired, that would be infringed by some manner, permitted by this License, of making, using, or selling its contributor version."

GPL-3.0's scope is the contributor's *contributor version* — the Program as it stood when the contributor delivered the contribution, not just the contribution alone. This is broader than Apache's "contribution alone or contribution + Work" because it covers patents reading on the entire Program at the moment of contribution.

**Apache 2.0 §3 — termination trigger:**
- Filing patent litigation alleging "the Work or a Contribution incorporated within the Work constitutes direct or contributory patent infringement"
- Termination is to the *suing party only*, applies to all licenses they hold under that Work
- Does NOT terminate other contributors' grants, just the relationship between suer and project

**GPL-3.0 §10 — termination trigger:**

> An entity transacting with you using the program is granted patent licenses... You may not impose any further restrictions on the exercise of the rights granted... including, for example, you may not impose a license fee... on the rights granted... [and] If you cease all violation of this License, then your license from a particular copyright holder is reinstated...

Plus §11 expressly addresses patent suits. Termination is reciprocal to GPL-3.0's downstream-grant model — once you sue, your right to use the Program goes away.

**Defensive comparison:**

| Property | Apache-2.0 §3 | GPL-3.0 §11 |
|----------|---------------|-------------|
| Grant scope | Contribution + Work | Contributor version (entire Program at contribution time) |
| Implicit Combinations | Limited | Broader — covers integration patents |
| Termination breadth | All licenses for that Work | Cumulative with §10 reinstatement |
| Anti-laundering | Not explicit | §11 anti-laundering: cannot pay a third party to provide a discriminatory patent license |
| Successor patents | Covered (irrevocable for irrevocable patents) | "Hereafter acquired" patents covered |

GPL-3.0's anti-laundering (§11 paragraph 6) is the unique addition: contributors can't structure deals where a non-contributor third-party offers users a discriminatory patent license. The text targets the Microsoft-Novell deal of 2006, where Microsoft offered Novell customers patent peace but other Linux users none. GPL-3.0 forbids contributors from arranging such deals.

```
Patent suit against Apache Project X:
  → suer's grant from §3 terminates
  → other contributors' grants unaffected
  → suer can no longer use Project X under Apache-2.0
  → other users continue using Project X freely

Patent suit against GPL-3.0 Project Y:
  → §11 grants from suer terminate
  → §10 license terminates (no further restrictions)
  → §10 reinstatement possible if suer ceases violation
  → §11 anti-laundering: cannot route patent claims through proxy
```

For a project receiving corporate contributions, both clauses materially reduce patent risk versus MIT/BSD (no patent grant).

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

## BUSL Conversion-to-Apache Mechanic

BUSL-1.1 (Business Source License v1.1) ships with two interlocking provisions: a per-licensor "Additional Use Grant" describing what production use is allowed during the restricted window, and a "Change Date" with associated "Change License" defining what the source becomes after the window.

The relevant text from the BUSL-1.1 standard form:

> Effective on the Change Date, or the fourth anniversary of the first publicly available distribution of a specific version of the Licensed Work under this License, whichever comes first, the Licensor hereby grants you rights under the terms of the Change License, and the rights granted in the paragraph above terminate.

**Worked example — HashiCorp Terraform:**

| Version | Released | BUSL Phase | Change Date (4 years) | Becomes |
|---------|----------|------------|----------------------|---------|
| 1.6.0   | Aug 2023 | BUSL-1.1   | Aug 2027             | MPL-2.0 |
| 1.7.0   | Jan 2024 | BUSL-1.1   | Jan 2028             | MPL-2.0 |
| 1.8.0   | Apr 2024 | BUSL-1.1   | Apr 2028             | MPL-2.0 |

Each minor version has its own Change Date computed from its release. So in 2027, only Terraform 1.6.x converts to MPL-2.0; 1.7.x is still BUSL-restricted until 2028. The "rolling MPL window" creates a 4-year lag between innovation and full openness.

**Conversion mechanic on Change Date:**

1. Existing BUSL-1.1 grant terminates
2. Change License (MPL-2.0 in Terraform's case, Apache-2.0 in many others) automatically applies
3. The conversion is unilateral — no relicensing action required by the rightsholder
4. Past distributions under BUSL retain their BUSL terms (rights from the date you received them); new distributions get the Change License

**Additional Use Grant — Terraform's specific text:**

> You may make production use of the Licensed Work, provided such use does not include offering the Licensed Work to third parties on a hosted or embedded basis which is competitive with HashiCorp's products.

The "competitive" carve-out is what excludes managed Terraform services from third parties. End users running Terraform locally face no restriction. The mechanic is conceptually similar to Elastic's "compete with us" prohibition but with the time-bomb that eventually opens the code.

**OpenTofu fork response:** When HashiCorp announced BUSL adoption in August 2023, the OpenTofu Foundation forked Terraform's MPL-2.0-licensed pre-conversion code and continues development under MPL-2.0 only. They cannot incorporate BUSL-licensed Terraform changes (BUSL → MPL is asymmetric — BUSL conditions don't fit MPL's umbrella) until each Change Date passes. So OpenTofu lives 4 years behind Terraform's main branch in feature parity for any BUSL-only changes.

## SSPL §13 Cloud-Rehoster Trigger

SSPL extends AGPL §13 with a cloud-specific clause that targets managed-service rehosters. The relevant text (SSPL v1, MongoDB-authored):

> If you make the functionality of the Program or a modified version available to third parties as a service, you must make the Service Source Code available via network download to everyone at no charge, under the terms of this License. Making the functionality of the Program or modified version available to third parties as a service includes, without limitation, enabling third parties to interact with the functionality of the Program or modified version remotely through a computer network, offering a service the value of which entirely or primarily derives from the value of the Program or modified version, or offering a service that accomplishes for users the primary purpose of the Program or modified version.

"Service Source Code" is then defined to include:

> the Corresponding Source for the Program or the modified version, and the Corresponding Source for all programs that you use to make the Program or modified version available as a service, including, without limitation, management software, user interfaces, application program interfaces, automation software, monitoring software, backup software, storage software and hosting software, all such that a user could run an instance of the service using the Service Source Code you make available.

**The trigger pattern:**

```
Provider A → Runs unmodified MongoDB-as-a-service → SSPL §13 fires
   Required to release: MongoDB source + management plane + UI + API gateway
   + monitoring + backup + storage + hosting orchestration
```

This is intentionally hostile to AWS DocumentDB (Amazon's MongoDB-compatible service launched January 2019, which dodged AGPL by re-implementing the wire protocol against a different storage engine). MongoDB's bet: any cloud provider running actual MongoDB code and offering it as a service must release their entire infrastructure stack — an unacceptable cost.

**Operational impact:**
- Self-hosters using MongoDB (run on internal servers): no §13 trigger (not "service to third parties")
- ISVs building applications on MongoDB and selling the application: no §13 trigger (service is the app, not MongoDB)
- Cloud providers offering "managed MongoDB" with users connecting to MongoDB ports: §13 fires → must release infra source

**OSI rejection rationale (March 2019):**

OSI's License Review Committee rejected SSPL primarily on OSD criteria 6 (no field-of-use discrimination) and 9 (no restriction on other software). The "Service Source Code" requirement bundles unrelated software (monitoring, backup, etc.) under SSPL's terms — OSI viewed this as restricting other software the user happens to combine with MongoDB.

MongoDB withdrew the application but kept SSPL as the license. As of 2025 SSPL is widely deployed (Elastic, Redis 7.4+, others) but remains source-available rather than open source.

## Elastic License v2 Prohibition List

Elastic License v2 (ELv2) replaced Elastic Stack's Apache-2.0 license in February 2021. Unlike SSPL's broad source-disclosure requirement, ELv2 is a use-restriction license — source visible, but specific uses prohibited.

The four prohibitions from ELv2:

> You may not provide the software to third parties as a hosted or managed service, where the service provides users with access to any substantial set of the features or functionality of the software.

> You may not move, change, disable, or circumvent the license key functionality in the software, and you may not remove or obscure any functionality in the software that is protected by the license key.

> You may not alter, remove, or obscure any licensing, copyright, or other notices of the licensor in the software. Any use of the licensor's trademarks is subject to applicable law.

> Central to a Defined Term: "Software" means the software the licensor makes available under this license, as may be updated by the licensor from time to time.

The "hosted or managed service" prohibition is the AWS-targeting clause. The "license key" prohibition is the commercial-feature-protection clause (Elastic's enterprise tier ships features gated by license keys; ELv2 forbids removing the gates).

**Worked compliance flow for an enterprise:**

1. **Self-host Elasticsearch internally for application search**: allowed (you are the user, not "third parties")
2. **Embed Elasticsearch in a SaaS product as the search backend**: allowed if Elasticsearch is "incidental" — Elastic interprets this as the product's primary value not being Elasticsearch itself
3. **Sell "Managed Elasticsearch" to customers**: prohibited — directly hits the hosted-service clause
4. **Disable license-key checks in source to access enterprise features**: prohibited
5. **Modify Elasticsearch and contribute back**: allowed (modifications permitted), but the modifications inherit ELv2

**OpenSearch fork (AWS, April 2021):**

AWS forked Elasticsearch 7.10 (the last Apache-2.0 version) into OpenSearch. The fork:
- Stays Apache-2.0 (no ELv2 restrictions)
- Cannot incorporate Elastic-authored ELv2 features without a clean-room reimplementation
- AWS continues its OpenSearch Service offering on Apache-2.0 code
- Trademark fork: cannot use "Elasticsearch" branding

The fork is the ecosystem's response to ELv2: a permissive replacement maintained by AWS, with Apache, eBay, Adobe, and Bytedance contributing. OpenSearch as of 2025 has substantial divergent feature work versus Elasticsearch; the codebases are no longer line-by-line comparable.

**ELv2 vs SSPL — Comparison:**

| Property | ELv2 | SSPL |
|----------|------|------|
| Source visibility | Yes | Yes |
| Modifications | Allowed | Allowed |
| Restricted use | Hosted service + license key bypass | Hosted service (with broader source disclosure trigger) |
| OSI status | Not approved (use restriction) | Rejected (2019) |
| Conversion timer | None | None |
| Forks | OpenSearch (Apache-2.0) | None for MongoDB; Valkey (BSD) for Redis |

## Dual-Licensing Theory

Copyright holder can license the same code under multiple terms. Common pattern:

- License A: GPL-3.0 (free, but copyleft = enterprises hesitant)
- License B: Commercial (paid, no copyleft restrictions)

The user picks: comply with GPL or pay for commercial.

For dual-licensing to work, the rightsholder must own all the copyright. With contributions, this requires either:
- **Copyright Assignment Agreement (CAA)** — contributors transfer copyright to project
- **Contributor License Agreement (CLA)** — contributors grant the project broad relicensing rights

Without one of these, contributor copyrights are co-owned, and dual-licensing requires every contributor's permission for each license sale.

## MySQL — Canonical GPL+Commercial Dual License

MySQL AB (later acquired by Sun, then Oracle) operated the canonical dual-license model from 1995 onward.

**The two-track structure:**

- **MySQL Community Edition** — GPL-2.0 (now GPL-2.0 only with FOSS exception)
- **MySQL Enterprise Edition** — proprietary commercial license, paid subscription

The same source code, same binaries technically — but distributed under different terms with different support and feature additions.

**Why customers paid for the commercial license despite GPL availability:**

1. **GPL incompatibility with their stack.** A vendor shipping a closed-source product that links MySQL client libraries cannot redistribute under GPL-2.0 + closed-source terms. Buying the commercial license avoided the linkage problem.
2. **No copyleft propagation.** Commercial license customers could ship MySQL with proprietary modifications, no source disclosure required.
3. **Indemnification.** Commercial license bundled IP indemnity, which GPL excluded.
4. **Phone support, certifications.** Commercial-tier-only operational benefits.

**The CLA was foundational:**

MySQL AB required a Contributor License Agreement granting MySQL AB rights to relicense contributions. Without this, MySQL AB could not have offered the commercial license — every contributor's GPL-only grant would have prevented commercial relicensing. The CLA aggregated all copyright into MySQL AB's hands for licensing purposes (contributors retained copyright but granted MySQL AB sublicensing rights).

**The Sun acquisition (2008):**

When Sun bought MySQL AB for $1B, the dual-license stack came intact. The CLA chain meant Sun acquired the right to continue dual-licensing without re-asking contributors.

**The Oracle acquisition (2010) and aftermath:**

Oracle's stewardship led to community concerns about MySQL's openness. Two notable consequences:

1. **MariaDB fork (2009)** — Monty Widenius, MySQL's original author, left Sun before the Oracle deal closed and forked MySQL into MariaDB. Initially used GPL-2.0; later versions adopted LGPL/BSD for client libraries to ease integration.
2. **Drizzle fork (2008-2014)** — Brian Aker forked MySQL into Drizzle, an experimental cloud-optimized variant, ultimately abandoned.

The MySQL/MariaDB split shows the dual-license model's failure mode: when stewardship changes, contributors who don't want commercial-licensing rights to flow to a new owner fork, taking the GPL-only option. The CLA is irreversible from MySQL's side; contributors' future contributions can flow elsewhere.

## Qt — Same Pattern, LGPL+Commercial

Qt Group operates a dual license very similar to MySQL's:

- **Qt Open Source** — LGPL-3.0 (most modules), GPL-3.0 (some)
- **Qt Commercial** — proprietary, paid

The structural elements:
- LGPL chosen instead of GPL because Qt is a library — LGPL allows linking from proprietary apps without copyleft propagation
- CLA required from contributors granting The Qt Company sublicensing rights
- Same dual-license offering: customers can use LGPL freely, or pay for commercial license to avoid LGPL's reverse-engineering and source-availability obligations

**The Nokia → Digia → Qt Group transition (2008-2014):**

Qt's ownership changed three times. Each transition validated the CLA structure: rights flowed cleanly to successor owners. Customers experienced no licensing disruption.

**Why Qt's commercial license is more attractive than MySQL's:**

LGPL's static-linking obligations and library-replacement requirement (must allow users to relink with a modified library) are operationally tedious for embedded products. Commercial Qt customers avoid these. MySQL's GPL is even more restrictive — customers paid more for commercial MySQL relative to LGPL Qt's pricing.

**Lessons from Qt:**
- Choice of free-tier license affects commercial premium: GPL → high premium; LGPL → moderate premium; permissive → low premium (commercial loses appeal)
- CLA is mandatory; without it the dual license collapses
- Branding ("Qt" trademark) reinforces the dual offering — customers buy "Qt" the brand, not just the bits

## MongoDB-as-Warning — Dual License Death

MongoDB pre-2018 followed the dual-license playbook:
- AGPL-3.0 — community
- Commercial — paid

Why it failed and what it teaches:

**Cloud providers neutralized the AGPL trigger.** AWS DocumentDB (2019) implemented MongoDB's wire protocol against an AWS-built storage engine — no AGPL-licensed code in their service. MongoDB's network-copyleft trigger never fired against AWS.

**Cloud providers ran unmodified MongoDB.** AWS, Azure, and GCP also offered managed MongoDB, where they ran the AGPL binary unchanged. AGPL §13 only triggers on **modification** — running unmodified AGPL software as a service does not require source disclosure. Cloud providers complied with AGPL trivially (no modifications) while capturing significant revenue.

**Commercial license sales declined.** Why pay MongoDB for commercial when you can use AWS DocumentDB? The commercial-license premium evaporated.

**MongoDB's response was SSPL (October 2018).** The intent: force cloud providers running MongoDB-as-a-service to release their entire infrastructure stack — making the offering economically infeasible. Result:
- AWS continued DocumentDB (their wire-compatible reimplementation, untouched by SSPL)
- Smaller cloud providers and on-prem distributors migrated to alternatives (some Postgres-based)
- OSI rejected SSPL; the project lost "open source" branding
- Some enterprise customers shifted away due to license uncertainty

**The dual-license death spiral:**

1. Permissive/copyleft community license attracts users
2. Cloud providers offer managed service
3. Vendor's commercial-license value erodes (cloud is cheaper / better integrated)
4. Vendor switches to source-available license (BSL/SSPL/ELv2)
5. Forks emerge (Valkey, OpenTofu, OpenSearch)
6. Community fragments

**The pattern in 2024:**

Almost every major open-source-with-commercial company has moved to source-available:
- MongoDB (SSPL, 2018)
- Elastic (SSPL/ELv2, 2021)
- Redis (RSALv2/SSPL, 2024)
- HashiCorp (BSL, 2023)
- Sentry (FSL, 2023)
- MariaDB MaxScale (BSL)
- CockroachDB (BSL)

The dual-license model — open source community + commercial enterprise — has been replaced by source-available time-bombs. The "free for most, paid for cloud rehosters" architecture is the new equilibrium.

**The lesson for new projects:**
- Pick a license that matches the business model from day one
- Don't bait-and-switch (community remembers)
- If hyperscalers are a competitive threat, bake the license restriction in from the start (Elastic License v2 from day one would have caused fewer fork-fights than re-licensing later)
- CLA is necessary infrastructure for any future relicensing — but adoption of CLA itself signals intent to relicense, scaring contributors

The dual-license era of 1995-2018 is largely over. The MySQL-Qt-MongoDB arc traces its rise and fall.

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
- Elastic License v2 text — https://www.elastic.co/licensing/elastic-license
- OpenSearch fork announcement — https://aws.amazon.com/blogs/opensource/introducing-opensearch/
- BUSL-1.1 specification — https://mariadb.com/bsl11/
- OSI rejection of SSPL — https://opensource.org/blog/the-sspl-is-not-an-open-source-license
- OpenTofu Foundation — https://opentofu.org/
- ClearlyDefined — https://clearlydefined.io/
- REUSE Specification — https://reuse.software/spec/
- gpl-violations.org archive — https://gpl-violations.org/
- Software Freedom Conservancy enforcement — https://sfconservancy.org/copyleft-compliance/
- OIN (Open Invention Network) — https://www.openinventionnetwork.com/
- TLDRLegal license summaries (informational, not legal advice) — https://www.tldrlegal.com/
- Choose A License (GitHub) — https://choosealicense.com/
- MySQL FOSS Exception — https://www.mysql.com/about/legal/licensing/foss-exception/
- The Qt Company licensing — https://www.qt.io/licensing/
- MongoDB SSPL announcement (2018) — https://www.mongodb.com/blog/post/mongodb-now-released-under-the-server-side-public-license
- Redis license change announcement (2024) — https://redis.io/blog/redis-adopts-dual-source-available-licensing/
- Valkey fork — https://valkey.io/
- MariaDB Foundation — https://mariadb.org/
