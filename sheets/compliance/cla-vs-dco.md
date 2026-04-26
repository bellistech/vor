# CLA vs DCO

Contributor License Agreements (CLAs) and the Developer Certificate of Origin (DCO) decoded — every variant, mechanic, enforcement tool, and migration path.

## Setup

A **Contributor License Agreement (CLA)** is a legal document signed by a contributor before their code is accepted into a project. It explicitly grants the project (or governing foundation) specific rights: a license to use, modify, distribute, and (often) relicense the contributed code. Some CLAs include a patent grant. Some involve outright copyright assignment.

A **Developer Certificate of Origin (DCO)** is a per-commit attestation. The contributor signs each commit by adding a `Signed-off-by:` trailer (via `git commit -s`). The DCO text certifies that the contributor has the right to submit the contribution. There is no separate document, no upfront signing ceremony, and no copyright assignment.

### Why projects need either

Open-source projects accept code from strangers. Without something attesting the contributor's rights:

- The project does not know whether the contributor owns the copyright
- The project does not know whether the contributor's employer might claim the work
- The project cannot verify the contributor isn't pasting code copied from a closed-source project
- Future relicensing becomes legally fraught — every contributor must be tracked down

CLAs and DCOs solve different aspects of the problem:

| Concern | CLA | DCO |
|---|---|---|
| Right to contribute | Yes | Yes |
| Patent grant | Yes (most) | No (relies on outbound license) |
| Future relicensing | Yes (with relicense clause) | No |
| Copyright assignment | Optional | No |
| Contributor friction | High | Low |
| Auditable | Database of signatures | Git log of trailers |

### What they protect against

- **Ambiguous license rights**: Without an explicit grant or attestation, a contributor's right to contribute is implied at best.
- **Future-relicensing impossibility**: If a project wants to move from GPL-2.0 to GPL-3.0 (or any change), every contributor must agree unless the CLA pre-authorizes it.
- **SCO-vs-Linux-style claims**: A contributor (or their employer) later claims ownership and demands removal or royalties.
- **Work-for-hire surprises**: A contributor uses their employer's IP without authorization; the employer later asserts rights.

## Threat Model

Without a CLA/DCO, the project is at risk of:

### Unclear license rights from contributors

A contributor pushes a PR. They include a one-liner that fixes a typo. They include a 5,000-line refactor of the auth subsystem. The project merges both. Six months later, the contributor's employer sends a letter: "Our employee did that work on company time. We own it. Remove it or pay licensing fees."

Without a CLA or DCO, the project has no documented assertion that the contributor had the right to submit. The threat is real — see SCO Group, Inc. v. Novell, Inc.

### Future relicensing impossible

A project starts as GPL-2.0. Five years later, the maintainers want to move to Apache 2.0 to attract enterprise contributors. They need permission from every contributor whose code remains in the codebase. Some are dead. Some are unreachable. Some refuse. Result: the project is stuck on GPL-2.0 forever, or must rewrite every line of those contributors' code.

A CLA with a relicense clause solves this. A DCO does not (the DCO only certifies the right to submit under the project's existing license — not future ones).

### SCO-vs-Linux-style copyright claims

In 2003, SCO Group sued IBM and several Linux users, claiming Linux contained code copied from Unix System V (which SCO claimed to own). The lawsuits dragged on for over a decade. One reason Linux survived: the DCO. Each commit was attested-to by the contributor. The trail of accountability was in the git log.

The Linux kernel adopted the DCO in 2004 specifically as a response to SCO. The kernel community has used it ever since.

### Contributor not actually having rights to contribute their code

A contributor pastes a function from Stack Overflow. The function was originally posted under a license incompatible with the project's. The contributor doesn't realize. Without a DCO/CLA, the project has no defensible position.

With a DCO: the contributor certified `Signed-off-by:` indicating they had the right to submit. If they lied, liability shifts to them, not the project.

With a CLA: same shift, plus the project has an explicit license grant.

## CLA Overview

A **Contributor License Agreement** is a legal document where the contributor grants the project (or foundation) specific rights to use the contributed code. The CLA is signed once per contributor (usually) — not per commit.

### The two flavors

- **Copyright assignment (CLA-A / CA)**: Contributor transfers copyright in their contribution to the project or foundation. The project becomes the legal owner. The contributor cannot independently license the code commercially without buy-back. This is rare in modern projects but still used by the Free Software Foundation.

- **License grant (CLA-G / LG)**: Contributor retains copyright but grants the project a broad license to use, modify, distribute, sublicense, and (often) relicense the contribution. This is the modern norm — Apache, Microsoft, Google all use license-grant CLAs.

### The two contributor types

- **Individual CLA (ICLA)**: Signed by an individual contributor. Covers their personal contributions. They certify they have the right to contribute (no employer claims, no third-party rights).

- **Corporate CLA (CCLA)**: Signed by a corporate entity (the contributor's employer). Covers all contributions made by the corporation's employees. Often required by foundations when a contributor is contributing on behalf of their employer.

A contributor working for a company that has not signed a CCLA may need to sign an ICLA personally — and even then, the employer may need to acknowledge the contribution is made on personal time.

### What a CLA grants (Apache-style example)

- A perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable copyright license to reproduce, prepare derivative works of, publicly display, publicly perform, sublicense, and distribute contributions
- A perpetual, worldwide, non-exclusive, no-charge, royalty-free, irrevocable patent license to make, use, sell, offer to sell, import, and otherwise transfer contributions
- The right to relicense (sometimes — depends on the CLA)
- A representation that the contributor has the right to grant these licenses

## CLA Variants

### Apache CLA

The most influential CLA template. Used by the Apache Software Foundation and copied by many other projects.

Key features:

- License grant (not assignment) — contributor retains copyright
- Explicit patent grant with retaliation clause
- Project distributes contributions under Apache 2.0 (matched to outbound license)
- ICLA and CCLA variants
- Signed via web form historically, fax/email in early days

The Apache CLA text is approximately 2,000 words. It includes:

```
2. Grant of Copyright License. Subject to the terms and conditions of
this Agreement, You hereby grant to the Foundation and to recipients
of software distributed by the Foundation a perpetual, worldwide,
non-exclusive, no-charge, royalty-free, irrevocable copyright license
to reproduce, prepare derivative works of, publicly display, publicly
perform, sublicense, and distribute Your Contributions and such
derivative works.

3. Grant of Patent License. Subject to the terms and conditions of
this Agreement, You hereby grant to the Foundation and to recipients
of software distributed by the Foundation a perpetual, worldwide,
non-exclusive, no-charge, royalty-free, irrevocable (except as stated
in this section) patent license to make, have made, use, offer to
sell, sell, import, and otherwise transfer the Work...
```

### Apache-style CLAs

Many companies and projects use a CLA modeled directly on Apache's:

- AWS open-source projects (e.g., aws-cli, aws-sdk-*)
- Confluent (Apache Kafka contributions)
- DataStax
- Elastic (pre-license-change era)
- HashiCorp (Terraform, Vault, Consul, etc.)
- Many CNCF graduated projects

The pattern: copy Apache's CLA structure, change "the Foundation" to "the Company", keep the patent grant.

### Google CLA

Google maintains its own CLA for projects it sponsors (Go, Kubernetes pre-CNCF, gRPC, TensorFlow). It is structurally identical to Apache's: license grant, patent grant, ICLA and CCLA forms, web-based signing via cla.developers.google.com.

Distinguishing features:
- Explicit reference to Google as the recipient (vs the Foundation)
- Includes a "Submissions are public" acknowledgment
- Storage in a Google-controlled database

### Microsoft CLA

Used for .NET, dotnet/runtime, TypeScript, VS Code (where applicable), and PowerShell.

Distinguishing features:
- License grant + patent grant
- Microsoft-specific indemnification language
- Signed via cla.opensource.microsoft.com (cla-bot)
- Coverage extends to "Submissions" — broadly defined

### Eclipse CLA / ECA

The Eclipse Foundation requires the **Eclipse Contributor Agreement (ECA)**. It is similar to Apache's CLA in structure but tailored for the Eclipse Public License.

Key differences:
- Compatible with EPL-2.0 outbound
- Requires a verified Eclipse account (eclipse.org)
- Tied to git commit signing — the email on the commit must match the ECA-signed account
- ECA + signed commit acts somewhat like a hybrid DCO/CLA

### The .NET Foundation CLA

For projects under the .NET Foundation umbrella (CoreFX, CoreCLR transferred to .NET Foundation, etc.). Uses a CLA based on the Microsoft CLA but with the Foundation as the recipient.

### CNCF DCO+CLA hybrid

The Cloud Native Computing Foundation (CNCF) uses an unusual model:

- **DCO required for all individual contributors** — every commit must have `Signed-off-by:`
- **CCLA required for corporate contributions** — the corporation signs an Apache-style CCLA covering its employees
- The CCLA is enforced via **EasyCLA** (LFX tool)
- The DCO is enforced via the **DCO bot**

This means a Kubernetes contributor working at Google must:
1. Have Google's CCLA on file with CNCF
2. `git commit -s` every commit

### FSF Copyright Assignment

The Free Software Foundation requires **full copyright assignment** for contributions to GNU projects (GCC, Emacs, Bash, etc.) above a trivial threshold (typically 15 lines).

Key features:
- Full transfer of copyright to the FSF
- Signed paperwork — not a web form (originally physical signatures)
- Reasoning: FSF can defend the project legally as the sole copyright holder
- Reasoning: FSF can relicense (e.g., GPL-2 to GPL-3) without contacting contributors
- The strictest form of contributor agreement

The FSF's case for assignment: gnu.org/licenses/why-assign.html

This is also the most controversial. Many developers refuse to assign copyright, viewing it as a corporate-style power grab — even when the recipient is a non-profit.

### Software Freedom Conservancy contributor agreements

The Conservancy (umbrella for Git, BusyBox, Inkscape, Outreachy, etc.) takes a project-by-project approach:

- Some Conservancy projects use FSF-style assignment
- Some use light-touch CLAs
- Some use DCO only
- Conservancy's role is administrative, not legal-policy-imposing

## CLA Mechanics

### Detection

When a PR is opened, a bot scans the contributor list against a database of signed CLAs:

- Contributor's GitHub username present in signed list? Pass.
- Contributor present but committing from a corporate email not on a CCLA? Flag.
- Contributor not present at all? Block PR with comment.

### The CLA bot signing flow

```
1. Contributor opens PR
2. Bot comments: "Please sign the CLA at https://cla.example.com/sign"
3. Contributor clicks link
4. Web form: read agreement, enter name + email, sign
5. Backend: match GitHub username to signed agreement
6. Bot updates PR status check: "CLA signed — passing"
7. PR can be merged
```

Some flows allow signing via a git commit trailer:

```
git commit -m "feat: my change

I have read the CLA Document and I hereby sign the CLA"
```

Most modern bots (cla-assistant) prefer web-form signing for auditable proof.

### Tools

- **cla-assistant.io** — open-source, GitHub App, free for OSS, hosted by SAP
- **EasyCLA** — Linux Foundation's enterprise tool, used by CNCF
- **GitHub Apps for CLA** — many private/custom variants
- **CDF's contributor agreement tool** — for Continuous Delivery Foundation
- **cla-bot npm package** — embeddable in custom infrastructure
- **Custom maintainer scripts** — for small projects

### Per-PR or per-project signing

Most CLA bots are scoped to:

- A specific GitHub organization (all repos in `kubernetes/*`)
- A specific repository
- A specific project namespace

A contributor signing the Kubernetes CLA does not automatically sign the Knative CLA, even though both are CNCF projects (though EasyCLA can centralize this).

### Storage of signatures

- **Encrypted database** (most modern tools): signatures stored with timestamps, IP addresses, and signed text version
- **GitHub-based markdown file**: a `CONTRIBUTORS.md` or similar listing
- **Centralized portal**: cla.developers.google.com, cla.opensource.microsoft.com
- **Audit trail for relicensing**: when the project considers relicensing, the database is queried for relicense-permission contributors

## DCO Overview

The **Developer Certificate of Origin v1.1** is a per-commit attestation. The contributor adds a `Signed-off-by:` trailer to each commit. The trailer represents the contributor's agreement to the DCO text.

### Origin

The DCO was developed for the Linux kernel by IBM lawyers in 2004 in response to the SCO Group's lawsuits against Linux contributors. The kernel community needed a lightweight way to attest each contributor's right to submit, without the heavy machinery of a CLA.

The DCO is published at developercertificate.org. It is approximately 250 words.

### The DCO text (v1.1)

```
Developer Certificate of Origin
Version 1.1

Copyright (C) 2004, 2006 The Linux Foundation and its contributors.

Everyone is permitted to copy and distribute verbatim copies of this
license document, but changing it is not allowed.


Developer's Certificate of Origin 1.1

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
    this project or the open source license(s) involved.
```

### What the DCO certifies

- (a) The right to submit (own work, or sufficient license)
- (b) Submission is based on previously-licensed work and the contributor has rights
- (c) Forwarded work that already had DCO sign-off
- (d) Public record acknowledgment

### What the DCO does NOT do

- It does not grant a license — the project's outbound license governs
- It does not include a patent grant — the project's outbound license must include one
- It does not authorize relicensing — the contribution is licensed under whatever the project's license was at submission time
- It does not assign copyright — contributor retains all rights

### Why DCO works for Linux

The Linux kernel is GPL-2.0. The DCO certifies the contributor's right to submit under GPL-2.0. The kernel cannot be relicensed because no CLA grants relicense permission. This is a feature, not a bug — the kernel community has chosen GPL-2.0 forever.

## DCO Mechanics

### Sign-off via `git commit -s`

```bash
git commit -s -m "feat: add new feature"
```

This adds a trailer to the commit message:

```
feat: add new feature

Signed-off-by: Real Name <email@example.com>
```

Equivalent to `--signoff`:

```bash
git commit --signoff -m "feat: add new feature"
```

### The trailer format

```
Signed-off-by: Real Name <email@example.com>
```

Strict format requirements:

- Capitalized exactly: `Signed-off-by:` (not `signed-off-by:`, not `Signed-Off-By:`)
- Single space after the colon
- Real name (per Linux convention; some projects allow handles)
- Valid email in angle brackets
- Email must match `git config user.email`

### Configure git to know your name and email

```bash
git config --global user.name "Real Name"
git config --global user.email "email@example.com"
```

The DCO bot will compare the `Signed-off-by:` email to the commit's author email. Mismatch causes rejection.

### Multiple Signed-off-by lines

When code passes through multiple maintainers, each adds their own sign-off:

```
feat: add new feature

Signed-off-by: Original Author <author@example.com>
Reviewed-by: Maintainer One <maintainer1@kernel.org>
Signed-off-by: Maintainer One <maintainer1@kernel.org>
Signed-off-by: Subsystem Maintainer <subsys@kernel.org>
```

This is the canonical Linux kernel pattern. Each `Signed-off-by:` represents a separate attestation that the signer has the right to forward the contribution.

### The probot/dco GitHub bot

The most common DCO enforcement tool. A free GitHub App that:

- Checks every commit in a PR for `Signed-off-by:` trailer
- Validates trailer format
- Compares trailer email to commit author email
- Posts a status check: passing or failing
- Comments on PR with instructions if failing

Configuration via `.github/dco.yml` (rare — most use defaults).

### Rejection of PRs missing the sign-off

A failing DCO check blocks the PR (if branch protection requires status checks to pass). The contributor must:

1. Amend the commit to add the sign-off:
   ```bash
   git commit --amend --signoff
   git push --force-with-lease
   ```
2. Or rewrite history if multiple commits are affected:
   ```bash
   git rebase --signoff main
   git push --force-with-lease
   ```

### Bypass via `--no-signoff`

The contributor can intentionally opt out of DCO sign-off by configuring git or by removing the trailer. The DCO bot will then reject the PR. There is no "bypass DCO" feature in the bot itself — bypassing requires either the maintainer manually merging without status check enforcement, or disabling the DCO bot.

## CLA vs DCO Comparison

| Dimension | CLA | DCO |
|---|---|---|
| What it is | Signed legal document | Per-commit attestation |
| Granularity | Per contributor (sign once) | Per commit (sign every) |
| Grants license? | Yes, explicitly | No — relies on outbound license |
| Grants patent? | Most do | No (relies on outbound license) |
| Allows relicensing? | If clause included | No |
| Friction | High (contributor must sign agreement) | Low (`git commit -s`) |
| Corporate review needed? | Often | Rarely |
| Auditability | Database of signatures | Git log |
| Foundation governance support | Yes (Apache, Eclipse, etc.) | Yes (CNCF DCO + CCLA) |
| Linux kernel compatible? | No (LK uses DCO only) | Yes |
| Setup complexity | High | Low |
| Tooling | cla-assistant, EasyCLA | probot/dco |

### When CLAs win

- The project may want to relicense in the future
- Patent grants matter (and the outbound license doesn't include one)
- Corporate contributors prefer the legal certainty of a signed agreement
- Foundation governance models require it (Apache, Eclipse)

### When DCOs win

- Lower contributor friction (especially first-time contributors)
- The project has no plans to relicense (Linux kernel)
- The outbound license already includes patent grants (Apache 2.0 outbound)
- Corporate legal departments have not pre-approved CLA signing

## The "No CLA" Movement

### Linux kernel — DCO only

The Linux kernel uses DCO only. No CLA. No copyright assignment. Linus Torvalds has stated multiple times that CLAs are corporate gates and the DCO is sufficient.

Greg Kroah-Hartman (LK maintainer) has written extensively on this position. Quotes from his blog and LWN:

> "CLAs are a corporate gate that keeps individuals out. The DCO is a self-attestation that respects the contributor."

The kernel relies on:
- DCO sign-off on every commit
- The kernel's GPL-2.0 license as the inbound = outbound license
- Strict review processes by maintainers
- Public mailing list discussion of every patch

### Many BSD projects

FreeBSD, OpenBSD, NetBSD historically have not required CLA or DCO formal sign-off. They rely on:

- The license header in each file (BSD-2-Clause, BSD-3-Clause)
- Per-commit license inference (committer is asserting the file's license)
- Strict commit-bit access (only trusted committers can push)

Critique: this works for projects with tight committer membership but doesn't scale to drive-by contributions.

### The argument against CLAs

Greg KH and Linus Torvalds:

- "CLAs are corporate gates" — small contributors face high friction
- "Trust the license, not the agreement" — the outbound license already grants what's needed
- "Relicensing is overrated" — most projects never relicense; those that do can negotiate

### The counter-argument

- Sometimes relicensing is essential — e.g., projects moving from GPL-2.0-only to GPL-2.0-or-later
- Sometimes patent grants are essential — particularly for projects in patent-rich domains (codecs, networking)
- Foundation governance models effectively require CLAs for legal liability reasons

## Hybrid Approaches

### DCO + project license

The simplest pattern. Used by Linux, GitLab, and many others. The contributor:

1. Signs off each commit (DCO)
2. The commit is implicitly licensed under the project's license (the file's license header governs)

No CLA. No copyright assignment. Inbound = outbound license.

### DCO + CLA for relicensing

Some projects use DCO for daily contributions but gather CLAs only when planning a relicense. The downside: collecting CLAs from past contributors is hard. The upside: zero ongoing friction.

### CNCF DCO + CCLA for organizations

The Cloud Native Computing Foundation model:

- Individual contributors: DCO (`git commit -s`)
- Corporate contributors: CCLA on file with CNCF
- Both enforced automatically

The CCLA covers the corporation's contributions broadly. The DCO covers each commit specifically.

### CCLA for orgs + DCO for individuals

A common pattern. Individual hobbyists sign off via DCO. Corporations sign CCLAs covering all employees. New employees inherit coverage automatically.

## Apache CLA — full text walkthrough

The Apache Individual Contributor License Agreement (ICLA) is the template most modern CLAs are based on. It has 9 sections.

### Section 1: Definitions

Defines key terms:
- "You" — the contributor
- "Contribution" — any work submitted
- "Submit" — any form of communication intentionally sent to the Foundation

The "intentionally sent" language matters: a casual email mentioning an idea is not a Contribution. A PR with code is.

### Section 2: Grant of Copyright License

```
Subject to the terms and conditions of this Agreement, You hereby grant
to the Foundation and to recipients of software distributed by the
Foundation a perpetual, worldwide, non-exclusive, no-charge,
royalty-free, irrevocable copyright license to reproduce, prepare
derivative works of, publicly display, publicly perform, sublicense,
and distribute Your Contributions and such derivative works.
```

Key terms:
- **Perpetual**: forever
- **Worldwide**: all jurisdictions
- **Non-exclusive**: contributor retains rights
- **No-charge / Royalty-free**: free
- **Irrevocable**: cannot be withdrawn
- **Sublicense**: project can license to others (enables relicensing-like behavior)

### Section 3: Grant of Patent License

```
Subject to the terms and conditions of this Agreement, You hereby grant
to the Foundation and to recipients of software distributed by the
Foundation a perpetual, worldwide, non-exclusive, no-charge,
royalty-free, irrevocable (except as stated in this section) patent
license to make, have made, use, offer to sell, sell, import, and
otherwise transfer the Work...
```

The patent grant is the critical addition. It covers patents the contributor owns or controls that read on the contribution.

The "(except as stated in this section)" clause introduces the **patent retaliation** provision: if the contributor sues the project (or another user) for patent infringement, the contributor's own patent grant terminates.

### Section 4: You are not expected to provide support

```
You are not expected to provide support for Your Contributions, except
to the extent You desire to provide support. You may provide support
for free, for a fee, or not at all.
```

A reassurance: contributing does not obligate the contributor to maintain the contribution.

### Section 5: Disclaimer of Warranty

```
Unless required by applicable law or agreed to in writing, You provide
Your Contributions on an "AS IS" BASIS, WITHOUT WARRANTIES OR
CONDITIONS OF ANY KIND, either express or implied...
```

Standard "as-is" disclaimer. The contributor isn't warranting their code works.

### Section 6: Indemnity

```
You agree to notify the Foundation of any facts or circumstances of
which You become aware that would make Your representations in this
Agreement inaccurate in any respect.
```

Note: this is a notification duty, not a financial indemnification. The contributor is not signing up to defend the Foundation in lawsuits — only to inform them of issues.

### Section 7: Should be You and Foundation acknowledged

The Foundation publishes the contribution. The contributor is acknowledged. The legal relationship is documented.

### The signing process

Historically (early 2000s):
- Print the CLA
- Sign with a pen
- Fax to the Foundation
- Email a scan

Modern:
- Web form at iclas.apache.org
- Login with Apache ID
- Click-through agreement
- Database record

Many other projects follow the same evolution.

## The Copyright Assignment vs License Grant Distinction

### Copyright Assignment (CA)

The contributor transfers copyright to the project or foundation. The project becomes the copyright owner.

Implications:
- Contributor cannot independently license their contribution
- Contributor cannot use their contribution in a closed-source product (unless granted back-license)
- Project has full control to relicense
- Project has full standing to enforce copyright in court

Example: FSF Copyright Assignment for GNU projects. The contributor signs an "Assignment of Past and Future Changes" form. FSF becomes the legal owner.

### License Grant (LG)

The contributor retains copyright but grants a broad license. The project does not own — only licenses.

Implications:
- Contributor can use their contribution in any way (commercial, closed-source, etc.)
- Contributor can dual-license to other parties
- Project has license to use, sublicense, distribute (often relicense)
- Project has standing to enforce only via license terms

Example: Apache CLA (license grant only).

### "What if the contributor wants to use their own code commercially?"

- Under copyright assignment: they cannot, unless the Foundation grants a back-license
- Under license grant: they can — they retain all rights

This is one reason most modern projects prefer license grant. Contributors don't want to lose rights to their own work.

### Hybrid: limited assignment

Some agreements assign only specific rights (e.g., the right to enforce) while leaving copyright with the contributor. Rare in OSS but common in commercial contributor agreements.

## Patent Grants in CLAs

### Why patents matter

Software patents can render an open-source project legally unusable. A contributor might own a patent that reads on their own contribution. Without an explicit patent grant from the contributor, downstream users could be sued by the contributor.

### Apache CLA Section 3 (patent grant)

```
You hereby grant to the Foundation and to recipients of software
distributed by the Foundation a perpetual, worldwide, non-exclusive,
no-charge, royalty-free, irrevocable (except as stated in this
section) patent license to make, have made, use, offer to sell, sell,
import, and otherwise transfer the Work, where such license applies
only to those patent claims licensable by You that are necessarily
infringed by Your Contribution(s) alone or by combination of Your
Contribution(s) with the Work to which such Contribution(s) was
submitted.
```

Key restrictions:
- Only patents the contributor controls
- Only patents necessarily infringed by the contribution
- Not a blanket grant to all the contributor's patents

### Patent retaliation

```
If any entity institutes patent litigation against You or any other
entity (including a cross-claim or counterclaim in a lawsuit) alleging
that your Contribution, or the Work to which you have contributed,
constitutes direct or contributory patent infringement, then any
patent licenses granted to that entity under this Agreement for that
Contribution or Work shall terminate as of the date such litigation
is filed.
```

The retaliation clause: if the contributor sues anyone for patent infringement related to the contribution, the contributor's own patent grant evaporates. This deters patent trolling.

### Most CLAs include a patent grant

Apache, Microsoft, Google, Mozilla — all include explicit patent grants. The exceptions:
- Projects under licenses that already include patent grants (e.g., Apache 2.0 outbound) sometimes rely on the outbound grant
- Some smaller projects with license-grant-only CLAs don't include patent grants

### Hostile patent positions

Some open-source projects (notably the Software Freedom Law Center-aligned projects) refuse contributions from companies known for patent aggression. This is a community-norm enforcement, not a legal one.

## Indemnification & Warranties

### Apache CLA: explicit "no warranty"

The Apache CLA explicitly disclaims warranties. The contributor is not warranting their code works, is bug-free, or is fit for any purpose.

### Some corporate-CLAs require corporate-side indemnification

Larger corporate CLAs may include indemnification provisions: the corporation agrees to indemnify the project against claims arising from the corporation's contributions.

This is rare in standard OSS CLAs but common in commercial contributor agreements.

### "Represents and warrants" language

Standard CLA language:

```
You represent that you are legally entitled to grant the above
license. If your employer(s) has rights to intellectual property
that you create that includes your Contributions, you represent
that you have received permission to make Contributions on behalf
of that employer, that your employer has waived such rights for
your Contributions to the Foundation, or that your employer has
executed a separate Corporate CLA with the Foundation.
```

This is the contributor's representation of authority. If they lie, liability shifts to them.

## Enforcement Mechanisms

### CLA enforcement

- **cla-assistant**: GitHub App, status check on PRs, web-form signing flow
- **EasyCLA**: Linux Foundation's tool, used by CNCF and other LF foundations
- **cla-bot variants**: many forks for specific projects
- **Manual review**: maintainer checks CLA database, comments on PR
- **GitHub branch protection**: required status check ensures CLA pass before merge

### DCO enforcement

- **probot/dco**: GitHub App, checks every commit for `Signed-off-by:` trailer
- **DCO Action**: GitHub Action variant
- **Custom checks**: projects with their own CI scripts

### Foundation-level enforcement

- Apache: Apache ID required, ICLA on file, signed git config matched
- Eclipse: ECA verified via Eclipse account
- CNCF: EasyCLA for CCLA, DCO bot for individual sign-off
- .NET Foundation: cla-bot via Microsoft

## Storage of Signatures

### Encrypted database

Most modern CLA tools store:

- Contributor identity (GitHub username, email, real name)
- Date of signing
- Version of CLA signed
- IP address of signing
- Full signed text (for audit)

The database is queried per PR to determine pass/fail.

### GitHub-username-based listings

Some projects publish a `CONTRIBUTORS.md` or similar file listing contributors who have signed. This is often a derived view of the database, not the source of truth.

### Audit trail when relicensing

If a project ever wants to relicense:

- Query the CLA database for all contributors
- Identify which CLAs include relicense permission
- For contributors whose CLAs don't include relicense permission, contact them individually
- For unreachable contributors, either remove their contributions or accept legal risk

This is why CLAs are valuable for relicensing. DCOs do not provide this.

## Choosing CLA vs DCO Decision

### Decision framework

| Question | Recommendation |
|---|---|
| Need to relicense in future? | CLA (with relicense provision) |
| Just want contributor's right-to-contribute attested? | DCO |
| Foundation governance (Apache, Eclipse)? | CLA (mandatory typically) |
| Foundation governance (CNCF)? | DCO + CCLA |
| Corporate-friendly + low contributor friction? | DCO |
| Patent grant essential and outbound license lacks one? | CLA with explicit patent grant |
| Linux-kernel-compatible? | DCO only |
| Apache 2.0 outbound? | DCO is sufficient (Apache 2.0 has patent grant) |
| GPL outbound? | DCO is usually sufficient |

### Detailed reasoning

- **Apache 2.0 outbound + DCO inbound**: works because Apache 2.0 already grants what the project needs from contributors. Many CNCF projects use this.
- **GPL outbound + DCO inbound**: works because GPL is share-alike and the contributor's right-to-contribute under GPL is what's needed. Linux kernel pattern.
- **MIT outbound + CLA inbound**: works for projects that want to retain relicense rights despite the permissive outbound license. Some HashiCorp pattern (pre-license-change era).
- **Proprietary fork potential**: a CLA with relicense permission lets the project go closed-source later. This is controversial and one reason some contributors refuse to sign such CLAs.

## Practical Examples

### Linux kernel

- DCO via `git commit -s`
- No CLA
- Inbound = outbound license (GPL-2.0)
- Bot enforcement on lkml and on github.com/torvalds/linux

### Kubernetes

- DCO via `git commit -s` for individuals
- CCLA required for corporate contributions (via EasyCLA)
- Outbound license: Apache 2.0
- Bot enforcement: probot/dco + EasyCLA

### Apache projects

- ICLA + CCLA at apache.org
- No DCO requirement (CLA is sufficient)
- Outbound license: Apache 2.0
- Enforcement: Apache ID + git config matching

### Mozilla

- Mozilla Committers Agreement (variant of CLA) for committers
- For drive-by contributors: case-by-case
- Outbound license: MPL 2.0
- Enforcement: Mozilla bug tracker integration

### Eclipse projects

- ECA at eclipse.org
- Eclipse account required
- Outbound license: EPL 2.0
- Enforcement: git committer email must match Eclipse account

### .NET / dotnet/runtime

- Microsoft CLA via cla-bot
- Web-form signing at cla.opensource.microsoft.com
- Outbound license: MIT
- Enforcement: cla-bot status check

### GitLab

- DCO via `git commit -s`
- Outbound license: MIT
- Enforcement: probot/dco + GitLab equivalent

### HashiCorp's open-source projects

- CLA via cla-assistant
- Web-form signing
- Outbound license: BSL (Business Source License) post-license-change; previously Mozilla Public License 2.0
- Enforcement: cla-assistant status check

## CLA + Open Source License Compatibility

### Apache CLA + Apache 2.0

Aligned. Apache CLA grants exactly what Apache 2.0 expects from contributors.

### GPL projects + corporate CLA

Tension. GPL is a copyleft license that requires derivative works to be GPL. A corporate CLA that grants relicense permission could allow the corporation to take the project closed-source. Many GPL community members oppose this.

FSF-style projects use FSF CA: copyright is assigned to FSF, which (despite being a foundation) acts as a copyright trustee for the GPL community.

### "We use license X but the CLA grants broader rights"

A common pattern. The outbound license is Apache 2.0 (or MIT, or whatever), but the CLA grants the project the right to relicense to any OSI-approved license.

This is legally fine but politically fraught. Contributors may not realize they're granting more rights than the outbound license implies.

## Common Mistakes

### Not having a CLA/DCO at all

The contributor's rights are ambiguous. The project is at SCO-vs-Linux risk. Future relicensing is impossible without per-contributor permission.

### DCO mandated but signed-off email doesn't match git author

The DCO bot rejects. The contributor must amend with the correct email, or update their `git config user.email`.

### CLA signed but corporate contributor's organization changed

The CCLA covers the contributor's previous employer. The new employer hasn't signed. The contributor needs a new ICLA or their new employer needs to sign a CCLA.

### Foundation moves projects without re-CLAing contributors

When a project moves from one foundation to another (e.g., Knative from Google to CNCF), the CLAs may not transfer cleanly. Contributors may need to re-sign for the new foundation.

### Public-domain dedication assumed for "obvious" small contributions

A typo fix doesn't need a CLA, right? Wrong. Even small contributions can carry copyright (in some jurisdictions) and can carry contractual obligations. Some projects exempt single-line changes; most require sign-off regardless.

## Future-Proofing for License Changes

### "Project may relicense" clause

Strong CLAs include:

```
The Project may, at its discretion, license your Contribution under
any other OSI-approved open source license.
```

This pre-authorizes relicensing without contacting individual contributors.

### Per-CLA "irrevocable" clause

```
This Agreement is irrevocable. You cannot withdraw your grants under
this Agreement.
```

Without this, a contributor could theoretically revoke their CLA and demand removal of their contributions. The "irrevocable" language prevents this.

### The MongoDB-style "we reserve the right to relicense"

MongoDB famously relicensed from AGPL to SSPL (Server Side Public License) in 2018. The relicense was possible because MongoDB had been gathering CLAs that granted relicense permission. The SSPL relicense was controversial but legally clean.

This is what CLAs enable — and what some contributors view as a betrayal of OSS norms.

## Inbound vs Outbound License

### Definitions

- **Inbound license**: the license under which the contributor licenses the contribution to the project
- **Outbound license**: the license under which the project distributes to users

### Same? Apache 2.0 inbound + Apache 2.0 outbound

The Apache CLA + Apache 2.0 pattern. Inbound = outbound. Clean and simple.

### Different? Inbound MIT + outbound GPL

GPL projects can accept MIT contributions because MIT is GPL-compatible. The contributor licenses under MIT; the project distributes the combined work under GPL.

The reverse (inbound GPL + outbound MIT) is not allowed — GPL terms prevent the project from distributing under a more permissive license.

### Inbound = outbound rule

Projects that don't have CLAs typically operate under "inbound = outbound": the contributor's contribution is licensed under whatever the project's outbound license is. The DCO certifies the contributor's right to do so.

This is the Linux kernel rule.

## The Linux Kernel's "Inbound = Outbound" Rule

The Linux kernel does not have a CLA. Contributors implicitly license their contributions under GPL-2.0 (the kernel's outbound license).

The DCO certifies that the contributor has the right to do so:

- (a) The contributor created the work
- (b) The contributor has the right to license it under GPL-2.0
- (c) The contributor is forwarding work that already has DCO sign-off
- (d) The contributor acknowledges the public record

No additional CLA is needed because the project's license is the inbound license.

This works for the kernel because:
1. The kernel will never relicense (community consensus)
2. The GPL-2.0 outbound license includes everything needed
3. The patent grant is implicit in GPL-2.0's "additional permissions" language

For projects that might relicense, "inbound = outbound" + DCO is insufficient.

## Pseudonyms / Aliases

### Linux kernel rules

The Linux kernel requires "Real Name" in the `Signed-off-by:` trailer. The kernel's documentation states:

> "The name in the Signed-off-by line should be the developer's real name."

Pseudonyms are rejected. Email addresses must be valid.

### FSF position

The Free Software Foundation also requires real names for copyright assignment. The reasoning: if the project ever needs to enforce copyright in court, the copyright owner must be identifiable.

### Some projects allow handles

Many projects relax the real-name requirement:

- "satoshi nakamoto" — pseudonymous Bitcoin contributor
- One-letter handles
- Stage names

This is a tradeoff: lower friction for pseudonymous contributors vs reduced legal accountability.

### Trademark / accountability tradeoff

A real name binds the contributor to legal accountability. A pseudonym may shield the contributor — but also makes the contribution less defensible.

For corporate contributors, the corporation's name is usually expected (and the individual's name within the corporation is recorded separately).

## Tools Comparison

| Tool | Type | License | Maturity | Use Case |
|---|---|---|---|---|
| cla-assistant | CLA | Apache 2.0 | Mature | OSS / general |
| EasyCLA | CLA | Internal | Enterprise | LF / CNCF |
| probot/dco | DCO | ISC | Mature | OSS / general |
| DCO Action | DCO | MIT | Recent | GitHub Actions users |
| linaro-its CLA-test | CLA | GPL | Niche | Linaro projects |
| Custom scripts | Either | Varies | Varies | Small projects |

### cla-assistant

- Open-source, hosted by SAP at cla-assistant.io
- Free for open-source projects
- GitHub App
- Stores signatures in a database
- Web-form signing flow
- Supports ICLA and CCLA flows
- Emits status checks on PRs

### EasyCLA

- Linux Foundation's enterprise tool
- Powers CNCF, OpenJS, Hyperledger, Akraino, etc.
- Integrates with LFX (Linux Foundation X) identity platform
- Supports complex corporate signing workflows
- Stores signatures in LF infrastructure

### probot/dco

- Free GitHub App
- Built on Probot framework
- Checks every commit for `Signed-off-by:` trailer
- Validates trailer format
- Comments on failing PRs with instructions

### DCO Action

- GitHub Action variant
- Runs as part of CI pipeline
- Same check logic as probot/dco
- Useful for projects that prefer Actions over Apps

### Custom

Small projects often write a few lines of CI script:

```bash
#!/bin/bash
git log --format="%(trailers:key=Signed-off-by)" \
  origin/main..HEAD | grep -q . || {
  echo "Missing Signed-off-by trailer"
  exit 1
}
```

## Migration Patterns

### From CLA to DCO

Easier. Steps:

1. Stop requiring CLA signatures for new contributions
2. Start enforcing DCO via probot/dco
3. Decide what to do with old contributions:
   - Keep CLA records as historical
   - Don't ask new contributors to sign the old CLA
4. Update CONTRIBUTING.md to reflect new process

The transition is a policy decision, not a legal one. Existing CLAs remain valid for the contributions they covered.

### From DCO to CLA

Harder. Steps:

1. Decide what scope of CLA you need (relicense permission? patent grant? just license grant?)
2. Choose a CLA template (Apache, Microsoft, custom)
3. Set up signing infrastructure (cla-assistant, EasyCLA, etc.)
4. Decide retroactivity:
   - **Strict**: gather CLAs from every existing contributor (impractical)
   - **Going forward only**: new contributions require CLA, old contributions remain under DCO-only
5. Communicate the change clearly

The "going forward only" approach is the only practical one for established projects. The downside: future relicensing remains constrained by the legacy DCO-only contributions.

### Switching foundations

Moving a project from one foundation to another (e.g., Google to CNCF) often requires re-CLAing contributors. The new foundation's CLA may differ.

Some foundations cooperate to streamline this — but contributors must still re-sign. The administrative overhead can be significant.

## Common Errors / Bot Output

### "Signed-off-by trailer is required"

```
[DCO] commit abc1234 is missing Signed-off-by trailer
```

Fix:

```bash
git commit --amend --signoff
git push --force-with-lease
```

### "X has not signed the CLA"

```
[cla-assistant] @username has not signed the CLA. Please sign at https://...
```

Fix: visit the link, sign the CLA, the bot will re-check.

### "Email in Signed-off-by does not match commit author email"

```
[DCO] commit abc1234: Signed-off-by email "user@personal.com" does not match author email "user@work.com"
```

Fix: ensure git config email matches:

```bash
git config user.email "user@work.com"
git commit --amend --signoff --reset-author
```

Or update the Signed-off-by manually:

```
Signed-off-by: User Name <user@work.com>
```

### "Multiple commits missing Signed-off-by"

```
[DCO] 5 commits in this PR are missing Signed-off-by:
  - abc1234
  - def5678
  - ...
```

Fix with rebase:

```bash
git rebase --signoff main
git push --force-with-lease
```

### "Corporate CLA required for org members"

```
[cla-assistant] @username is a member of @some-org. A Corporate CLA is required.
```

Fix: have the org sign a CCLA, or have the member contribute personally without org affiliation.

## Common Gotchas

### Gotcha: Forgetting `git commit -s`

Broken:

```bash
git commit -m "feat: add feature"
git push
```

Result: DCO bot rejects.

Fixed:

```bash
git commit -s -m "feat: add feature"
git push
```

Or amend after the fact:

```bash
git commit --amend --signoff --no-edit
git push --force-with-lease
```

### Gotcha: Email in Signed-off-by doesn't match git config

Broken:

```bash
git config user.email "personal@example.com"
git commit -s -m "feat: add feature"
# Signed-off-by: Real Name <work@example.com>  ← manually edited
git push
```

Result: DCO bot rejects with email mismatch.

Fixed:

```bash
git config user.email "work@example.com"
git commit --amend --signoff --reset-author
git push --force-with-lease
```

### Gotcha: Using GitHub web editor (no -s flag)

Broken: edit a file in the GitHub web editor, click "Commit." No way to add `-s` flag.

Result: DCO bot rejects.

Fixed: clone the repo, commit locally with `-s`, push.

Or amend the commit via API/CLI after the web edit:

```bash
git fetch origin
git checkout commit-branch
git commit --amend --signoff --no-edit
git push --force-with-lease
```

### Gotcha: Signing with wrong real-name vs handle policy

Broken: the project requires real names. You sign as "DevOps Master."

```
Signed-off-by: DevOps Master <devops@example.com>
```

Result: project policy rejects (or maintainer asks to fix).

Fixed:

```
Signed-off-by: Real Name <devops@example.com>
```

### Gotcha: Squash-merge dropping Signed-off-by trailers

Broken: a PR has 5 commits, each with `Signed-off-by:`. The maintainer squash-merges. The squash commit message includes only the PR title and description. The original `Signed-off-by:` trailers may be dropped or consolidated.

Result: post-merge DCO check (if configured) may fail, or audit trail is muddled.

Fixed: the maintainer should include all `Signed-off-by:` trailers in the squash commit message, or the project should configure squash-merge to preserve trailers (some tools auto-aggregate).

### Gotcha: Cherry-picking from another branch and losing Signed-off-by

Broken:

```bash
git cherry-pick abc1234
# Lost the Signed-off-by trailer
```

Result: cherry-picked commit lacks the trailer.

Fixed:

```bash
git cherry-pick --signoff abc1234
```

Or amend after:

```bash
git commit --amend --signoff
```

### Gotcha: Signing CLA with personal email but contributing through corporate identity

Broken: contributor signs ICLA with personal Gmail. Then commits via corporate email at work.

Result: cla-assistant doesn't match the commit's author email to the signed CLA. PR blocked.

Fixed: either:
- Contributor signs ICLA with corporate email
- Corporation signs CCLA covering the contributor

### Gotcha: Forgetting CCLA when joining a contributing org

Broken: contributor joins a new corporation. Their personal ICLA still on file. They contribute via corporate email.

Result: cla-assistant detects the corporate email, requires a CCLA from the new corporation. Contributor blocked until corporation signs.

Fixed: corporation signs CCLA. Or contributor reverts to contributing via personal email under personal ICLA.

### Gotcha: DCO signed but contribution actually from work-for-hire

Broken: contributor signs `Signed-off-by:` claiming personal work. But the contribution was made on company time using company resources.

Result: technically a DCO violation (the contributor doesn't have the right to submit). Liability risk.

Fixed: corporation signs CCLA to cover the contributor's work-for-hire contributions. Or contribution is removed.

### Gotcha: Re-signing CLA needed when org name changes

Broken: corporation merges or renames. Old CCLA lists the old name. New legal entity hasn't signed.

Result: cla-assistant may not recognize the new entity. New CCLA required.

Fixed: new corporate entity signs CCLA. Old contributions remain covered by old CCLA.

### Gotcha: Author email differs from committer email

Broken: contributor uses `git commit --author="Other Name <other@example.com>"`. Their own committer info differs.

Result: DCO bot may flag mismatch. CLA bot may match against committer or author depending on config.

Fixed: ensure `Signed-off-by:` matches the **author** email (the bot's typical check). Or configure git to use consistent emails.

### Gotcha: GPG-signed commit but no DCO

Broken: commit is GPG-signed (`git commit -S`) but missing `Signed-off-by:`.

Result: GPG signing does not satisfy DCO. DCO bot still rejects.

Fixed: combine both:

```bash
git commit -s -S -m "feat: add feature"
```

(The `-s` adds Signed-off-by; the `-S` adds GPG signature.)

### Gotcha: PR author differs from commit author

Broken: contributor A creates a PR by pushing contributor B's commits to their fork. Contributor A is the PR author; contributor B is the commit author.

Result: cla-assistant may check against the PR author, not the commit author. Or vice versa. Depends on tool configuration.

Fixed: ensure both A and B have signed the CLA. Or A re-creates the commits as their own work.

## Idioms

- **"DCO is the path of least resistance"** — for projects without strong relicense ambitions, DCO offers attestation with minimal friction.
- **"CLA is the path of legal certainty"** — for projects needing patent grants, relicense rights, or foundation-level governance, CLAs provide explicit grants.
- **"Use both if you ever might relicense"** — DCO for individuals + CLA for corporations is a common hybrid.
- **"Automate enforcement, never trust manual review"** — bots like cla-assistant and probot/dco scale better than maintainer eyeballs.
- **"Match Signed-off-by email to git config"** — the most common DCO failure mode is email mismatch.
- **"Inbound = outbound is the simplest pattern"** — Linux kernel proves it scales.
- **"Real names matter"** — pseudonyms reduce legal accountability and may be rejected.
- **"Patent grants are not optional"** — for projects in patent-rich domains, explicit patent grants are essential.
- **"CLA databases are time bombs"** — accumulated CLAs become hard to query, hard to migrate, hard to reuse across foundations.
- **"DCO is not a license"** — DCO certifies the right to submit; the project's outbound license actually grants rights.

## See Also

- license-decoder
- spdx-identifiers
- gdpr
- gpg

## References

- developercertificate.org — DCO v1.1 official text
- apache.org/licenses/contributor-agreements.html — Apache ICLA and CCLA
- cla-assistant.io — open-source CLA enforcement tool
- github.com/cla-assistant/cla-assistant — cla-assistant source
- lfx.linuxfoundation.org/tools/easycla — EasyCLA documentation
- github.com/probot/dco — probot/dco GitHub App source
- gnu.org/licenses/why-assign.html — FSF case for copyright assignment
- linux kernel Documentation/process/submitting-patches.rst — DCO usage in kernel
- cla.developers.google.com — Google CLA portal
- cla.opensource.microsoft.com — Microsoft CLA portal
- iclas.apache.org — Apache CLA signing portal
- eclipse.org/legal/eca.php — Eclipse Contributor Agreement
- cncf.io/about/legal-documents/ — CNCF DCO + CCLA model
- lwn.net — LWN articles on CLA/DCO debates
- opensource.guide/legal/ — GitHub's open-source legal guide
