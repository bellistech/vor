# CLA vs DCO — Deep Dive

The legal mechanics of contributor IP: how CLAs aggregate copyright via explicit license, how DCOs assert rights via per-commit attestation, and which one fits your project's relicensing future.

## Setup — Copyright Assignment vs License Grant; Both Protect, Differ in Mechanics

When a contributor sends a patch to an open source project, two questions arise:

1. **Does the project have the legal right to incorporate, modify, and redistribute the contribution?**
2. **If the project ever wants to relicense (e.g., dual-license, move from GPL-2.0 to GPL-3.0), can it do so without re-asking every contributor?**

The first question is answered by the project's **license**. If the project's LICENSE file says GPL-3.0, then merely contributing to a public repo under that license arguably grants an inbound license matching the project's outbound license. GitHub's Terms of Service §D.6 and the Apache "inbound=outbound" doctrine support this:

> ...you license to us and to other users of the GitHub Service the right to use, display, and perform Your Content through the GitHub Service and to reproduce Your Content solely on the GitHub Service as permitted through GitHub's functionality.

But "GitHub will display your code" is not the same as "the project can relicense your code." For relicensing flexibility, projects use **Contributor License Agreements (CLAs)** or **Developer Certificate of Origin (DCO)** sign-offs.

The two mechanisms answer the questions differently:

| Mechanism | Question 1 | Question 2 |
|-----------|-----------|-----------|
| Inbound = outbound | Yes (implied) | No (each contributor owns their copyright) |
| DCO | Yes (attested) | No (same as above) |
| CLA (license grant) | Yes (explicit) | Yes (broad relicensing rights granted) |
| CLA (assignment) | Yes (transferred) | Yes (project owns copyright) |

The choice between these is consequential. Pick the wrong one and you're locked into your initial license, or you're scaring away corporate contributors with paperwork friction, or you're building up audit liabilities you can't unwind.

```
Patch arrives
     ↓
Without explicit grant → "inbound=outbound" + joint authorship
     ↓
With DCO            → contributor attests right to grant under project license
     ↓
With CLA (license)  → contributor grants project broad rights (relicensable)
     ↓
With CLA (assign.)  → contributor transfers copyright entirely
```

## The Joint-Authorship Trap

Without explicit aggregation, every contributor independently owns the copyright on their additions. The project becomes a **joint work** under copyright law, which has consequences:

- **17 USC §201(a)**: "The authors of a joint work are coowners of copyright in the work."
- Each co-owner can independently license non-exclusively
- Exclusive licensing requires unanimous consent
- Major decisions (lawsuit, relicense, sale) require unanimous consent

For a project with hundreds of contributors, "unanimous consent" is practically impossible. People retire, die, become uncontactable, change employer, change name, change minds.

The **SCO v. IBM Linux case** (2003-2021) showcased the consequences. SCO argued IBM had contributed UNIX-derived code to Linux without authorization, then sued IBM, Novell, and Linux users. The case dragged for 18 years, finally dismissed 2021 after SCO bankruptcy. Although SCO lost, the litigation demonstrated:

- Even spurious copyright claims against a joint work create existential litigation risk
- Without clear chain of title for every contribution, defending is harder
- Novell ultimately had to prove it owned UNIX copyrights (it did) before Linux contributions could be cleared

The lesson: projects without explicit IP grants are exposed to "your contribution is tainted" attacks. Modern projects use DCO at minimum to establish a per-commit chain of provenance.

The **Linux kernel** itself is a joint work — every contributor retains copyright. Linus has explicitly chosen this model and rejected CLA proposals. The kernel's defense against joint-authorship issues is: (a) DCO requirement since 2004, (b) GPL-2.0-only forever (no relicensing planned), (c) the implicit understanding that GPL-2.0 is the "constitutional" framework.

```
Joint work risks (no aggregation):
  - Cannot relicense
  - Cannot sue infringers exclusively
  - Vulnerable to "unauthorized contribution" claims
  - Each contributor can independently license non-exclusively
  
Linux's response:
  - Per-commit DCO (chain of attestation)
  - Stable GPL-2.0 (no relicensing required)
  - Strong project culture (rejected contributions stay rejected)
```

## The Apache CLA Anatomy — Section-by-Section

The Apache Individual Contributor License Agreement (ICLA) is the canonical CLA template. Most modern CLAs (Google's, Microsoft's, Linux Foundation's) derive from it.

**Preamble** establishes:
- Contributor identity (name, mailing address, country, email, GitHub username)
- The project receiving contributions (Apache Software Foundation in original)

**§1 Definitions**:
- "You" = contributor; "Foundation" = recipient
- "Contribution" = "any original work of authorship, including any modifications or additions to an existing work, that is intentionally submitted by You..."
- Carve-out: contributions explicitly marked "Not a Contribution" are excluded

**§2 Grant of Copyright License**:

> Subject to the terms and conditions of this Agreement, You hereby grant to the Foundation and to recipients of software distributed by the Foundation a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable copyright license to reproduce, prepare derivative works of, publicly display, publicly perform, sublicense, and distribute Your Contributions and such derivative works.

The seven adjectives are deliberate:

- **Perpetual** — no time limit
- **Worldwide** — no geographic limit
- **Non-exclusive** — contributor retains rights to use elsewhere
- **No-charge / royalty-free** — no payment owed
- **Irrevocable** — contributor cannot withdraw
- **Sublicensable** — Foundation can license downstream

The grant is broader than just "use under Apache-2.0." It's the *bundle of copyright rights*, with the right to **sublicense**. This is what enables relicensing: ASF can grant Apache-2.0 today and (in theory) GPL-3.0 tomorrow without re-asking contributors.

**§3 Grant of Patent License** (covered next section).

**§4 Representations**:
- "You represent that you are legally entitled to grant the above license."
- "If your employer(s) has rights to intellectual property that you create..., you represent that you have received permission to make Contributions on behalf of that employer..."
- "You represent that each of Your Contributions is Your original creation..."

The §4 representations matter because:
- Employer-owned IP (work-for-hire) cannot be licensed by employee alone
- Contributions copy-pasted from third-party code expose the project to claims

**§5 Warranties**:
- Disclaims warranties (the contribution comes "as is")

**§6 Notification of Inaccuracies**:
- Contributor agrees to notify if any §4 representation becomes false

**§7 Submission of Non-original Work**:
- If contributing third-party work, must mark it and identify license

The Apache **Corporate CLA (CCLA)** layers on top: an authorized signer (executive) signs on behalf of a corporation, and the CCLA designates which employees are covered. New employee additions go through the corporate signatory.

```
ICLA flow:
  Contributor signs → ASF stores agreement → all future contributions covered
  
CCLA flow:
  Executive signs CCLA → designates employees → employee contributions covered
  New employee → corp adds to designation list → covered
```

## Patent Grant in CLAs — Apache CLA §3

The Apache CLA §3 patent grant:

> Subject to the terms and conditions of this Agreement, You hereby grant to the Foundation and to recipients of software distributed by the Foundation a perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable (except as stated in this section) patent license to make, have made, use, offer to sell, sell, import, and otherwise transfer the Work, where such license applies only to those patent claims licensable by You that are necessarily infringed by Your Contribution(s) alone or by combination of Your Contribution(s) with the Work to which such Contribution(s) was submitted.

Key elements:

- **Patent license is automatic on contribution** — contributor doesn't separately negotiate
- **Scope: claims that read on the contribution** — not contributor's whole patent portfolio
- **Combinations matter**: "Contribution alone OR contribution + Work" — covers patents that read on the integrated combo, not just the patch alone

The **patent retaliation clause** continues:

> If any entity institutes patent litigation against You or any other entity (including a cross-claim or counterclaim in a lawsuit) alleging that your Contribution, or the Work to which you have contributed, constitutes direct or contributory patent infringement, then any patent licenses granted to that entity under this Agreement for that Contribution or Work shall terminate as of the date such litigation is filed.

The retaliation logic:
- Contributor X grants patent license to community
- Member Y sues contributor X (or anyone) over the contribution
- Y's grant terminates — Y can no longer use the contribution

This makes patent suits self-defeating: filing the suit costs you the right to use what you're suing over. It's a deterrent, not a remedy.

CLAs without explicit patent grants (e.g., bare DCO) rely on **implied license under estoppel** — the doctrine that you can't assert patents against software you contributed to. This is weaker; courts are inconsistent.

## The DCO Legal Theory

The Developer Certificate of Origin (developercertificate.org, version 1.1):

```
Developer Certificate of Origin
Version 1.1

By making a contribution to this project, I certify that:

(a) The contribution was created in whole or in part by me and I
    have the right to submit it under the open source license
    indicated in the file; or

(b) The contribution is based upon previous work that, to the best
    of my knowledge, is covered under an appropriate open source
    license and I have the right under that license to submit that
    work with modifications, whether created in whole or in part
    by me, under the same open source license (unless I am
    permitted to submit under a different license), as indicated
    in the file; or

(c) The contribution was provided directly to me by some other
    person who certified (a), (b) or (c) and I have not modified
    it.

(d) I understand and agree that this project and the contribution
    are public and that a record of the contribution (including all
    personal information I submit with it, including my sign-off) is
    maintained indefinitely and may be redistributed consistent with
    this project and the open source license(s) involved.
```

What DCO does:

- **Attestation** — contributor swears the contribution is theirs (or properly licensed)
- **No license grant** — DCO doesn't grant rights; it asserts the contributor has them under the project's license
- **Per-commit** — every commit must carry `Signed-off-by: Name <email>` trailer
- **Real name required** — pseudonyms excluded; this is the "contributor identity" for legal purposes

Legal weight:

- **Sworn statement under §1746** — perjury risk for false attestation (US)
- **Records preserved indefinitely** — git history is the audit trail
- **Per-commit granularity** — narrow scope (just this commit), not broad license

What DCO does NOT do:
- Doesn't grant rights to the project beyond what the project license already grants (inbound = outbound)
- Doesn't license patents (just asserts the right to license under project terms)
- Doesn't enable relicensing — the project license is fixed at the contribution moment

DCO is **lighter weight** than CLA:
- No agreement to sign upfront
- No corporate gates for employee contributors
- No CLA-bot delays in PR workflow
- No "click-through legal document" friction

This is why Linux kernel uses DCO — Linus values low contribution friction more than relicensing flexibility.

## The Linux Kernel Decision — Linus + Greg KH's Stance

In 2004, after the SCO lawsuit, kernel maintainers introduced DCO requirement. The choice of DCO over CLA was deliberate.

Linus Torvalds, repeatedly:

> I really *really* dislike the whole idea of any kind of "copyright assignment" or "contributor license agreement". I think they are stupid, and they get in the way of contributors.

Greg Kroah-Hartman, kernel maintainer:

> The DCO is sufficient for our needs. We don't need a CLA. CLAs introduce friction that would damage the contribution flow and the trust relationship with contributors.

The arguments:

1. **GPL-2.0 is permanent** — no relicensing planned, so CLA's relicensing flexibility is unneeded
2. **CLAs concentrate power** — a single entity (FSF, Apache, etc.) becomes the relicensing point of failure / power
3. **Contribution friction** — every CLA adds steps; every step loses contributors
4. **Corporate gating** — CLAs typically need legal review at the contributor's company, especially CCLAs; this delays or prevents contributions

The kernel position: contributions are explicitly under GPL-2.0-only via the LICENSE file + DCO sign-off. Joint-work concerns are accepted; the GPL-2.0-only commitment makes them moot.

The result has been overwhelmingly positive:
- Linux receives ~10,000 patches per release cycle
- Contributors come from individuals, FAANGs, hardware vendors, academic researchers
- No CLA paperwork blocks any of them

Other major projects that adopted DCO-style: Docker, Kubernetes (CNCF), Node.js, Rust (somewhat hybrid with CLA + DCO), Git itself.

## CLA + Patents vs DCO + Patents

Patent grants are the sharpest distinction between CLA and DCO models.

**CLA + Patents** (Apache, Google, Microsoft):
- Explicit patent license from contributor
- Scope: patents reading on the contribution + combinations
- Retaliation clause: filing suit terminates contributor's patent rights
- Project + downstream users benefit from explicit grant

**DCO + Patents** (Linux kernel, Docker):
- DCO doesn't license patents — just attests the right to grant under project license
- If project license has patent grant (GPL-3.0, Apache-2.0), the inbound license matches → patents granted
- If project license has no patent grant (GPL-2.0, MIT), no patent grant from contributor

Linux is GPL-2.0-only, which has weak/implicit patent grant. Defense relies on:
- **Open Invention Network (OIN)** — defensive patent pool; OIN members agree not to sue each other over Linux
- **GPL-2.0's no-further-restrictions** — courts may interpret as implied patent license
- **Corporate self-interest** — major Linux contributors don't sue because they all use Linux

This works in practice but is legally weaker than explicit grants. For pure-permissive projects (BSD, MIT) that want corporate contributions, CLAs with patent grants offer firmer ground.

```
                   Copyright | Patent
CLA (Apache style) |    Y    |   Y
CLA (FSF assign)   |    Y    |   Y (depends)
DCO + GPL-2.0      |    Y    |   Implicit
DCO + GPL-3.0      |    Y    |   Y (via license §11)
DCO + Apache-2.0   |    Y    |   Y (via license §3)
DCO + MIT          |    Y    |   None
```

## Copyright Assignment vs License Grant — FSF vs Apache Style

Two CLA philosophies:

**FSF style — Copyright Assignment**:
- Contributor *transfers* copyright to FSF
- FSF holds all copyright in the work
- Can relicense at will (single rightsholder)
- Maximum flexibility for FSF
- High friction (some jurisdictions don't easily allow assignment)

**Apache style — License Grant**:
- Contributor *retains* copyright
- Grants ASF a broad sublicensable license
- ASF can relicense (within scope of grant) but doesn't own copyright
- Contributor can use elsewhere
- Lower friction than assignment

The asymmetry:
- FSF can relicense GNU projects to GPL-4.0 (or anything) by fiat
- Apache cannot — it's bound by the grant scope. ASF could in theory release Apache 4.0 with new terms, but the existing grant is for current and future Apache licenses

In practice, both enable relicensing, but FSF's mechanism is cleaner. FSF has historically used assignment to:
- Move GNU projects from GPL-2.0-only to GPL-3.0+ (2007 transition)
- Sue infringers as sole rightsholder (no need to coordinate with contributors)

Apache uses the license-grant model and has never needed to wholesale relicense, so the limitation hasn't bitten.

The **practical critique** of assignment: contributors lose perpetual control over their own work. If FSF disappears or pivots, the copyrights are stuck. This is why Apache's model dominates — keeps contributors as stakeholders.

```
FSF assignment:
  Contributor (loses copyright) → FSF (sole rightsholder)
  + Maximum relicensing flexibility
  - Contributor loses control
  - Some jurisdictions tax/regulate assignments

Apache license grant:
  Contributor (keeps copyright) → ASF (broad licensee w/ sublicense rights)
  + Contributor retains rights to use elsewhere
  + Easier to set up internationally
  - Slightly less flexibility (bound by grant scope)
```

## Future-Proofing — Why Relicensing Matters

The relicensing question seems abstract until it bites. Examples:

**GPL-2.0 → GPL-3.0 (2007)**: GPL-3.0 added patent retaliation, anti-tivoization, anti-DRM clauses. Some GPL-2.0 projects wanted to upgrade to gain these protections. Projects with CLA: easy. Projects without: stuck unless every contributor agreed.

- Linux: stayed GPL-2.0-only — Linus opposed GPL-3.0
- GNU projects: most upgraded easily (FSF held copyright via CA)
- Samba: upgraded after individual contributor outreach (took years)

**MongoDB AGPL-3.0 → SSPL (2018)**: MongoDB Inc. switched license to deter cloud provider arbitrage. Possible because MongoDB had CLA from all contributors granting broad sublicensing rights.

**HashiCorp MPL-2.0 → BSL-1.1 (2023)**: HashiCorp changed Terraform's license. Possible because of CLA. Resulted in OpenTofu fork — contributors who disagreed forked.

**Elastic Apache-2.0 → SSPL/Elastic-License (2021)**: Same pattern. Resulted in OpenSearch fork by AWS.

**Redis BSD → SSPL/RSALv2 (2024)**: Resulted in Valkey fork.

The pattern: CLA projects can relicense (and increasingly do, when business model pressures apply). DCO/no-CLA projects cannot — Linux is structurally locked into GPL-2.0-only.

The "going forward" approach: CLA gives the project the option to relicense; DCO prevents it. Whether to want this option is a project value question:

- CLA: optionality, but you're a single relicense decision away from a fork
- DCO: rigidity, but contributors trust your license is permanent

```
2007: GPL-3.0 released. Linux can't move to it. Has CLA?
        No → DCO only → stuck on GPL-2.0
        
2018: MongoDB wants SSPL. Has CLA?
        Yes → relicensed without contributor consent (fork resulted)
        
2023: HashiCorp wants BSL. Has CLA?
        Yes → relicensed (OpenTofu fork)
        
Pattern: CLA enables relicensing → enables business model shifts → triggers forks
```

## The "JIT Provisioning" via DCO — git commit -s

DCO's mechanic is the `git commit -s` (or `--signoff`) flag, which appends:

```
Signed-off-by: Name <email@example.com>
```

to the commit message. This trailer is the legal moment of attestation — it's the per-commit "I certify the four DCO clauses." Without the trailer, the commit isn't covered.

The trailer must:
- Use the contributor's real name (no pseudonyms — DCO clause (d) implicit)
- Use a real email (preferably one tied to identity verification)
- Match exactly the format `Signed-off-by: Name <email>` (case-sensitive)

CI tools (DCO bot on GitHub) validate every commit:
- Reject PRs containing commits without sign-off
- Reject PRs where sign-off email doesn't match committer email
- Allow `git commit --amend -s` to add sign-off post-hoc

The audit trail:
- Git history records sign-off verbatim
- Commits are immutable (rewriting history breaks DCO chain)
- Years later, in litigation, the chain can be reconstructed

Compare to CLA: a click-sign on cla-assistant.io adds the signer to a database. PRs are blocked until the contributor signs. Once signed, all future contributions covered.

```
DCO mechanic:
  git commit -s
       ↓
  Signed-off-by trailer added
       ↓
  CI bot validates (matches committer)
       ↓
  PR allowed to merge
       ↓
  Permanent record in git history
  
CLA mechanic:
  Contributor opens PR
       ↓
  CLA bot checks DB → not signed
       ↓
  Bot posts comment with sign link
       ↓
  Contributor signs (one time)
       ↓
  All future PRs from that contributor allowed
```

## Tools as Legal Records

Modern CLA tooling:

**cla-assistant.io** (CLA Assistant Lite):
- GitHub Action / GitHub App
- Signed agreements stored as GitHub commits in dedicated repo
- PR check blocks merge until signed
- Open source, run-it-yourself option

**EasyCLA** (Linux Foundation):
- Used by CNCF, Hyperledger, etc.
- Supports both ICLA and CCLA
- Manages corporate domain mappings (autosign for @company.com)
- Audit reports for relicensing or due-diligence

**SalesForce CLA Bot** — internal/contributor-facing, used by SF projects.

**Custom GitHub bots** — many large projects roll their own.

The legal-record value:
- Each agreement: timestamped, contributor-signed, immutable record
- Audit: search by name, email, GitHub username
- Relicensing: pull all CLAs, verify scope of grants, prove right to relicense
- Compliance: respond to "was X authorized?" with cryptographic certainty

DCO tooling:

**DCO Bot** (Probot-based) — checks every commit for sign-off, posts status check, lets PRs merge only when all commits signed.

**git's own tooling** — `git log --format=%(trailers:key=Signed-off-by)` extracts sign-offs for audit.

Both models leave permanent records. CLA records sit in a centralized DB; DCO records sit in git history. Both survive forks (clones carry history).

## The Failure Modes

CLAs and DCOs both have failure modes:

**Pseudonym contributions**:
- DCO requires real name → pseudonym contributions are formally non-compliant
- CLA requires real identity → same
- Reality: many projects accept pseudonyms despite the rule (Linux includes commits from `Linus Torvalds` but also some less-clearly-identified contributors)

**Corporate identity changes**:
- Contributor signs CCLA at Company A
- Contributor moves to Company B → CCLA at Company A no longer covers
- Contributor must re-sign CCLA at Company B
- Many companies don't track this → coverage gaps

**Sub-contractor agreements**:
- Company A hires Contractor C; C contributes to a project
- Who owns C's IP? Depends on contract
- Project sees commit signed by Company A → assumes covered
- If contract didn't transfer IP, Company A can't grant — coverage fails

**Email mismatches**:
- DCO requires sign-off email to match committer email
- Use of personal email vs work email triggers conflicts
- Aliases, forwarding, name changes — all create CI friction

**Lost or revoked keys**:
- Contributor's GPG key compromised; commits re-signed
- CLA tooling may treat re-signed commits as new
- Edge cases proliferate

**Contributor death / disappearance**:
- DCO captures the moment; intent is preserved
- CLA captures the moment + grant; same
- Both survive contributor's exit (the legal moment was recorded)

The robust position: layer DCO + CLA + audit logs. Many large projects do this. Smaller projects pick one.

## Project Migration — CLA ↔ DCO

**CLA → DCO migration** is generally *easier*:
- Existing CLA records remain valid for past contributions
- Going forward, switch CI to DCO
- New contributors sign-off; old contributors no longer need to
- Optionally honor existing CLAs as "stronger than DCO" for relicensing purposes

**DCO → CLA migration** is *harder*:
- Existing DCO records cover the past — no relicensing flexibility
- Going forward, require CLA for new contributions
- Doesn't grant retroactive relicensing rights
- If you need to relicense, must contact every past contributor for permission

The "going forward" approach is standard:
- Keep historical contributions under their original terms
- New contributions covered by new mechanism
- Eventually (after years) the project's code is mostly under the new mechanism
- Relicensing to a less-restrictive license can sometimes proceed without old-contributor consent (if the old terms permit it — Apache → MIT works one direction, GPL → MIT doesn't)

A real example: **OpenStack** initially used a no-CLA model, then added the **OpenStack ICLA** in 2010, then moved to **CCLA + ICLA**. Existing contributions remain under their original terms; new contributions follow the new agreement.

Another: **CNCF projects** standardize on DCO. When a project joins CNCF (e.g., Kubernetes, Prometheus), they migrate from CLA to DCO. Past contributions stay covered by their original CLAs.

## References

- Linux DCO requirement (kernel documentation) — https://www.kernel.org/doc/html/latest/process/submitting-patches.html#sign-your-work-the-developer-s-certificate-of-origin
- DCO 1.1 text — https://developercertificate.org/
- Apache ICLA — https://www.apache.org/licenses/contributor-agreements.html#clas
- Apache CCLA — https://www.apache.org/licenses/contributor-agreements.html#clas
- FSF Copyright Assignment FAQ — https://www.gnu.org/licenses/why-assign.html
- Software Freedom Law Center, "A Legal Issues Primer for Open Source and Free Software Projects" — https://www.softwarefreedom.org/resources/2008/foss-primer.html
- "The Legal Mechanics of Open Source Contributions," Heather Meeker
- Linus Torvalds on CLAs (lkml) — search lkml.org for "CLA"
- Greg KH on CLAs — various blog posts, e.g., http://www.kroah.com/log/blog/
- cla-assistant.io — https://cla-assistant.io/
- EasyCLA (Linux Foundation) — https://easycla.lfx.linuxfoundation.org/
- DCO GitHub App — https://github.com/apps/dco
- 17 USC §201 (joint authorship) — https://www.law.cornell.edu/uscode/text/17/201
- 28 USC §1746 (sworn statements / unsworn declarations) — https://www.law.cornell.edu/uscode/text/28/1746
- Open Invention Network — https://openinventionnetwork.com/
- HashiCorp BSL announcement — https://www.hashicorp.com/license-faq
- OpenTofu fork (response to BSL) — https://opentofu.org/
- Elastic license change FAQ — https://www.elastic.co/pricing/faq/licensing
- OpenSearch announcement — https://aws.amazon.com/blogs/opensource/introducing-opensearch/
- MongoDB SSPL FAQ — https://www.mongodb.com/licensing/server-side-public-license/faq
- Redis license change announcement — https://redis.io/blog/redis-adopts-dual-source-available-licensing/
- Valkey fork — https://valkey.io/
- "Contributor Covenant" (different topic, but adjacent governance norm) — https://www.contributor-covenant.org/
- CNCF DCO policy — https://github.com/cncf/foundation/blob/main/policies-guidance/dco.md
- SCO v. IBM case background — Wikipedia, "SCO–Linux disputes"
