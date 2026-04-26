# CCPA — Deep Dive

California's privacy law as a model: pre-GDPR origins, opt-out semantics, the CPRA expansion, and the engineering implications of Sale-vs-Share, GPC, and the multistate patchwork.

## Setup — California's Privacy Law Model

The California Consumer Privacy Act (CCPA, AB 375) was signed June 28, 2018, after a sprint to head off a more aggressive ballot initiative. It became effective January 1, 2020, with enforcement beginning July 1, 2020.

The political genesis matters. Real-estate developer Alastair Mactaggart had qualified the **California Consumer Privacy Act of 2018** as a ballot initiative — a more aggressive version than what eventually passed. Tech companies (Facebook, Google, AT&T, Comcast, Verizon) opposed it. The legislature struck a deal: pass a compromise statute, Mactaggart withdraws the initiative. Result: CCPA, weaker than the initiative but stronger than no-law-at-all.

The pre-GDPR origins are partly accident. CCPA was drafted in 2018 — GDPR had become enforceable May 2018 (one month before CCPA was signed). The drafters cribbed concepts (right-to-know, right-to-delete, broad personal-info definition) but kept the **opt-out** model rather than GDPR's opt-in. This is the foundational structural difference.

The Cambridge Analytica revelations (March 2018) provided the political catalyst. Mass exfiltration of Facebook profile data via a personality quiz became the metaphor for "everyone's data is being misused." Public outrage gave the legislature cover to pass the bill.

CCPA was amended by the **California Privacy Rights Act (CPRA, Prop 24)** in November 2020 (effective January 1, 2023), which:
- Created the California Privacy Protection Agency (CPPA)
- Added "Sensitive Personal Information" category
- Added "Sharing" as separate trigger from "Sale"
- Added Right to Correct
- Extended look-back beyond 12 months
- Removed the cure period for some violations

```
2016: GDPR adopted (EU)
2018 Mar: Cambridge Analytica revealed
2018 Jun: CCPA signed (compromise vs ballot initiative)
2020 Jan: CCPA effective
2020 Jul: CCPA enforcement begins
2020 Nov: CPRA passed via ballot
2023 Jan: CPRA effective (with CPPA agency)
2024 Mar: CPPA finalizes regulations on automated decisionmaking
```

## CCPA vs GDPR — Different Models

The structural divergence:

| Dimension | CCPA/CPRA | GDPR |
|-----------|-----------|------|
| Default for processing | Allowed unless objected (opt-out) | Forbidden unless lawful basis exists |
| Lawful bases | N/A — no upfront basis required | 6 explicit bases (consent, contract, legal obligation, vital interests, public interest, legitimate interests) |
| Consent | Generally not required | Required for many uses; specific, freely-given, unambiguous |
| Personal information | Includes "household" — info linkable to a household, not just an individual | Personal data = identifiable natural person; household-only data not covered |
| Special categories | Sensitive PI: SSN, financial, location, race, religion, health | Special categories: race, political, religious, union, genetic, biometric, health, sex life, criminal |
| Inferences | Explicitly included as PI ("inferences drawn from any of the information") | Not separately enumerated; covered if linkable |
| Sale | Defined separately; opt-out right | "Disclosure" generally; same lawful-basis analysis |
| Profiling | Right to opt out of automated decisionmaking (CPRA) | Right not to be subject to solely-automated decisions (Art 22) |
| Fines | $2,500/violation; $7,500 intentional | Up to €20M or 4% of worldwide annual turnover |
| Enforcement | CPPA + AG | DPA (per-country) + EDPB |
| Private right of action | Only for breach of unencrypted PI | Yes (Art 79, 80, 82) |

The **opt-out model** has practical implications:

- Businesses can collect and process by default
- Users must affirmatively object
- "Do Not Sell My Personal Information" / "Do Not Sell or Share" links required
- Pre-checked consent boxes are legal under CCPA, illegal under GDPR

The **household inclusion** in CCPA makes it broader in some ways:
- "Household" = a person or group of people who reside together at the same address
- Smart-home device data (thermostat readings, energy use) is personal info even if not tied to an individual
- This catches IoT data that might escape GDPR

**Inferences** in CCPA explicitly include "internal predictive models" — meaning ML-derived attributes (e.g., "user is likely to buy luxury cars") are personal information. Subject to right-to-know/delete. GDPR covers profiling but doesn't enumerate inferences as a category.

```
Opt-out vs opt-in:
  CCPA: Collect → User objects → Stop
  GDPR: Lawful basis → Process → User can object only on certain bases

Personal scope:
  CCPA: Individual + Household
  GDPR: Individual only

Inferences:
  CCPA: Explicitly named as PI
  GDPR: Implicit (if identifiable)
```

## Personal Information Definition

Cal Civ Code §1798.140(o)(1):

> "Personal information" means information that identifies, relates to, describes, is reasonably capable of being associated with, or could reasonably be linked, directly or indirectly, with a particular consumer or household.

Examples (non-exhaustive list in §1798.140(o)(1)(A)-(K)):

- Identifiers (name, alias, postal address, IP, email, account name, SSN, driver's license, passport)
- Categories from Cal Civ Code §1798.80(e) — older identification list
- Characteristics protected under California or federal law (race, religion, national origin, etc.)
- Commercial info (purchase history, products considered, consuming tendencies)
- Biometric info
- Internet/network activity (browsing, search history, app/website interactions)
- Geolocation
- Sensory info (audio, electronic, visual, thermal, olfactory, similar)
- Professional/employment info
- Education info (non-public per FERPA)
- Inferences drawn from any of the above

**Reasonably linkable** is the operative phrase. A user ID is PI even if not tied to a name — it can be linked. An IP address is PI under most readings (though there's litigation; some courts disagree). A device fingerprint is PI.

Excluded:
- **Publicly available** info (lawfully made public — but with §1798.140(v) carve-out: public from government records, OR media presentations to the general public via "widely-distributed"; but excludes biometric info collected without consent)
- **De-identified** info (rendered non-identifiable per technical standards)
- **Aggregate** consumer info (statistical, not linkable to individuals)
- Info governed by HIPAA (health), GLBA (financial), DPPA (driver's license — limited), FCRA (credit), or California Confidentiality of Medical Information Act (CMIA)

The **other-law-covered** carve-outs are deliberate; CCPA was meant to fill gaps, not duplicate sectoral laws. A bank under GLBA doesn't owe CCPA rights for the GLBA-covered info, but does for non-GLBA marketing data.

```
Reasonably linkable spectrum:
                                                       
  Direct ID (SSN, name)                          [PI]
  Account ID                                     [PI]
  IP address (long-term)                         [PI]
  Device fingerprint                             [PI]
  Hashed email                                   [PI — pseudonym, still linkable]
  Aggregate stats (no row-level data)            [Not PI]
  De-identified (k-anonymous, technical std)     [Not PI]
  Publicly available govt records                [Not PI]
```

## Sensitive Personal Information (CPRA)

CPRA added a new sub-category, **Sensitive Personal Information** (SPI), with stricter rules. Defined in §1798.140(ae):

- Government identifiers: SSN, driver's license, state ID, passport
- Account log-in or financial account / debit / credit number with security/access code
- Precise geolocation (within 1850 feet / 1/3 mile)
- Racial or ethnic origin, religious beliefs, union membership
- Mail, email, text message contents (unless intended for the business)
- Genetic data
- Biometric info processed for unique identification
- Health info (excluding HIPAA-covered)
- Sex life or sexual orientation

**Right to Limit Use of Sensitive PI** (§1798.121):
- Consumer can direct business to limit SPI use to "what is necessary to perform the services or provide the goods reasonably expected"
- This is narrower than full opt-out: SPI use isn't banned, just narrowed
- "Necessary" allowance excludes: marketing, profiling for marketing, research not requested by consumer

The **"necessary use" standard** is intentionally narrow. A consumer at an e-commerce site:
- SPI: precise location used for shipping → necessary, allowed
- SPI: precise location used for ad targeting → not necessary, must stop on request

Required disclosure:
- Notice at collection must disclose SPI categories collected
- "Limit Use of Sensitive Personal Information" link required (or single combined link with "Do Not Sell or Share")

The **CPPA regulations** (effective March 2024) further specify that the "Limit Use" link must:
- Be conspicuous and clearly visible
- Use the title "Limit the Use of My Sensitive Personal Information"
- Function via single click without requiring sign-in

```
PI vs SPI rights:
  PI:
    - Right to know
    - Right to delete
    - Right to correct
    - Right to opt out of sale/share
  SPI (added on top):
    - Right to limit use to "necessary"
    - Required separate link or combined "Limit + Do Not Sell" link
```

## Sale vs Share — CPRA Distinction

CCPA originally had only "Sale" as a triggering term. CPRA split this into Sale and Share, both subject to opt-out:

**Sale** (§1798.140(ad)):
> "Sell," "selling," "sale," or "sold," means selling, renting, releasing, disclosing, disseminating, making available, transferring, or otherwise communicating orally, in writing, or by electronic or other means, a consumer's personal information by the business to a third party for monetary or other valuable consideration.

Key: monetary OR other valuable consideration. "Other valuable" is broad — service fees, ad targeting in exchange, tracking pixels with reciprocal data.

**Share** (§1798.140(ah)):
> "Share," "shared," or "sharing" means sharing, renting, releasing, disclosing, disseminating, making available, transferring, or otherwise communicating orally, in writing, or by electronic or other means, a consumer's personal information by the business to a third party for cross-context behavioral advertising, whether or not for monetary or other valuable consideration...

Key: cross-context behavioral advertising. Even without payment, sharing data for behavioral ad targeting triggers "Share" status.

The **"cross-context behavioral advertising"** definition (§1798.140(k)):
- Advertising directed at a consumer based on their personal info from their activity across businesses, distinctly-branded websites, applications, or services
- Other than the business, distinctly-branded site, app, or service with which the consumer intentionally interacts

Operational implications:
- Embedding Facebook Pixel that sends user data → Share (and likely Sale)
- Google Analytics in basic mode (without ad features) → typically not Share
- Google Analytics with "remarketing" enabled → Share
- Putting an Amazon Affiliate link → not Sale or Share (no PI flow)
- Using a third-party recommendations widget that profiles user → likely Share

The **Do Not Sell or Share link** is required (§1798.135) — single mechanism for opt-out. Users opting out:
- Must not have their PI sold or shared by the business
- Service providers can still receive PI under contract carve-out
- Opt-out persists at least 12 months

```
Sale: PI → 3rd party for $ or value
Share: PI → 3rd party for cross-context behavioral ads (with or without $)

Both require:
  - "Do Not Sell or Share My Personal Information" link
  - 15-day honor window after request
  - 12-month opt-out persistence
  - Separate opt-in if user under 16 (or under 13 — opt-in by parent)
```

## Verifiable Consumer Request Theory

CCPA/CPRA require businesses to verify the identity of a consumer making a rights request, but at "strength proportional to risk." §1798.130(a)(2) and CPPA regulations elaborate:

Verification factors:
- **Type of PI**: highest-risk (financial accounts, government IDs) → strong verification
- **Risk of harm**: identity theft potential → stronger
- **Sensitivity** (SPI vs PI)
- **Account-holder vs guest**: account holders verify via authentication; guests need additional proof

Typical verification approach:

| Request Type | Verification Strength |
|--------------|----------------------|
| Right to know — categories | Authentication for account holders; 2 data points for guests |
| Right to know — specific pieces | 3 data points; sworn statement |
| Right to delete | Same as specific pieces, plus 2-step confirmation |
| Right to correct | Verification of identity + verification of corrected info |
| Right to opt out (sale/share) | No verification required (frictionless) |

**Matching against business records**: data points must match what the business has on file. Common factors:
- Email used for account
- Last 4 of credit card
- Recent purchase amount
- Address on file
- Security questions

The **frictionless opt-out** for sale/share is critical. CPPA regulations (March 2024) explicitly forbid:
- Requiring consumer to create an account to opt out
- Requiring consumer to provide more info than necessary
- Forcing consumer to navigate multiple pages or click multiple times

GPC (Global Privacy Control) signals must be honored as opt-outs without further verification.

```
Risk vs verification matching:
  Low risk (opt-out):     no verification, must be frictionless
  Medium (right to know):  email + 1 data point
  High (delete):           email + 2 data points + 2-step confirm
  Highest (correct):       same as high + new info validation
```

## The Service Provider Carve-Out

CCPA distinguishes "third party" (PI sale/share) from "service provider" (contracted processor). Disclosure to a service provider is NOT a sale. §1798.140(ag):

A service provider is a person/entity that:
- Processes PI on behalf of the business
- Pursuant to a written contract that:
  - Prohibits PI use for commercial purposes other than providing services to business
  - Prohibits PI sale/share
  - Prohibits PI retention/use/disclosure outside direct business relationship
  - Prohibits combining PI with other sources except to perform business purpose
  - Requires SP to ensure subcontractors meet same standards
  - Requires SP to assist business with consumer rights requests
  - Requires SP to delete PI on business's instruction
  - Requires SP to certify it understands the restrictions

These contract clauses are MANDATORY for the carve-out to apply. Businesses must amend vendor contracts to include them — often called "CCPA Addendum" or "Data Processing Addendum (DPA)."

The implication: SaaS vendors processing customer PI must agree to service-provider terms or be considered third parties (and the business must treat the data flow as a sale).

Required SP contract elements (§1798.140(ag)):
1. Limited purpose statement
2. Prohibition on PI sale/share
3. Prohibition on PI retention beyond services
4. Subcontractor flowdown
5. Assistance with consumer requests
6. Deletion on instruction
7. SP certification of understanding
8. Audit/inspection rights for business

```
Data flows under CCPA:
  Business → Service Provider (contract OK)        → Not a sale
  Business → Third Party (no contract or non-SP)   → Sale or Share
  Business → Service Provider (broken contract)    → SP becomes 3rd party
                                                    → flow becomes sale/share
```

The **Contractor** category (CPRA): similar to service provider but for one-off contracted work. Same restrictions apply.

## Authorized Agents

Consumer can designate an authorized agent to make rights requests on their behalf. §1798.135(c) and CPPA regulations:

Verification required:
- Written, signed permission from consumer to agent
- Consumer's identity verified (independently)
- Agent's identity verified

Or:
- Power of attorney
- For opt-out: less stringent — agent attests to having permission

Practical implementation:
- Privacy-rights services (e.g., Privacy Bee, Mine, OneRep) act as authorized agents at scale
- Submit thousands of requests on behalf of users
- Businesses must process or risk violations
- CPPA regulations specifically allow rejection if verification fails

The "rights request" automation industry exists in part because authorized-agent provisions enable B2C2B flows: consumer pays Privacy Bee, Privacy Bee submits requests to hundreds of data brokers, results return to consumer.

```
Direct request:                 Agent request:
  Consumer → Business              Consumer → Agent → Business
                                      ↓
                                Verifies agent identity + consumer identity
                                Or reject for failed verification
```

## The 12-Month Look-Back

CCPA's right to know covers PI collected in the **12 months preceding the request**. §1798.130(a)(2):

> The right to know shall include any business required to provide a copy of the consumer's personal information... covering the 12-month period preceding the business's receipt of the verifiable consumer request.

CPRA expanded:
- After January 1, 2022: business must provide PI from any time period the consumer requests
- The 12-month limit is now the floor, not the ceiling
- Exception: if it's "impossible or would involve disproportionate effort" the business can limit

Operational impact:
- Backups, archives, cold storage all in scope
- "We deleted it 3 years ago" — must produce log proving deletion
- Long-term retention systems must support targeted retrieval

```
Pre-CPRA (CCPA only): 12 months strictly
Post-CPRA (Jan 2022): all data the business has, 
                      unless impossible/disproportionate
                      
12-month is the minimum; longer is required if the data exists
```

## Right to Delete

CCPA §1798.105: consumer can request deletion of PI a business collected. Statutory exceptions (§1798.105(d)):

1. Complete the transaction the PI was collected for, fulfill warranty/recall, or perform contract
2. Detect security incidents, protect against malicious/deceptive activity, prosecute responsible parties
3. Debug to identify and repair errors
4. Exercise free speech, ensure another consumer's right of free speech, or another legal right
5. Comply with California Electronic Communications Privacy Act
6. Engage in research in the public interest, with appropriate ethics + privacy controls
7. Enable solely internal uses reasonably aligned with consumer expectations based on the consumer's relationship with the business
8. Comply with a legal obligation
9. Use otherwise compatible with the context in which the PI was provided

The **#7 carve-out** ("internal uses reasonably aligned with consumer expectations") is the broadest. Litigation is testing what "reasonably aligned" means.

Examples:
- User signs up for a newsletter → PI used to send newsletter → reasonably aligned
- User signs up for newsletter → PI used to train ML for ad targeting → arguably misaligned

The deletion request triggers:
- Delete from production systems
- Delete from backups (eventually — backup-rotation aligned)
- Direct service providers to delete
- Direct third parties (sold/shared) to delete

```
Receive deletion request
       ↓
Verify identity
       ↓
Apply statutory exceptions (keep what falls under #1-9)
       ↓
For remaining PI:
  - Delete from prod
  - Schedule backup deletion (typically 30-90 days)
  - Notify service providers (via DPA)
  - Notify recipients of past sales/shares
```

## Right to Correct (CPRA)

CPRA added a new right (§1798.106): consumer can request correction of inaccurate PI.

> A business that collects personal information about consumers shall, upon receipt of a verifiable consumer request, use commercially reasonable efforts to correct inaccurate personal information.

Standards:
- "Commercially reasonable efforts" — not "guaranteed correction"
- Business may consider documentary evidence
- Business may dispute the correction (state position in writing)

Inverse of GDPR's right to erasure (Art 17): GDPR allows deletion in many cases; CCPA's correction is narrower (only for inaccuracies, not just any objection).

Operational:
- Inaccurate self-reported data → user-corrected via account UI
- Inaccurate inferred data → tricky (was the inference "wrong"? or just disliked?)
- Inaccurate third-party data (data broker enrichment) → must flag, may need to source-verify

Notification:
- After correction, business must direct service providers to correct
- Past sales/shares: notify recipients

## Global Privacy Control (GPC)

GPC is a browser-level signal that says "I opt out of sale/share of my personal information." Specified at https://globalprivacycontrol.org/.

Technical:
- HTTP header: `Sec-GPC: 1`
- Browser API: `navigator.globalPrivacyControl` → boolean
- Sent on all outbound requests when enabled
- Cannot be deactivated by site

Browser support:
- Brave (default on)
- Firefox (option)
- DuckDuckGo (default on)
- Chrome (no native, requires extension)
- Safari (no native)

Legal status:
- California AG (Bonta) declared in 2021 that GPC is a valid opt-out signal
- CPPA regulations (March 2024) require honoring GPC
- New York AG followed suit
- Colorado, Connecticut, others recognized

Operational implications:
- Detect `Sec-GPC: 1` on incoming requests
- Treat as if user clicked "Do Not Sell or Share"
- Disable analytics/ad pixels for that user
- Backend: don't sell/share their PI to third parties
- Persist opt-out in user preference (12+ months)

If user is logged in:
- GPC applies to current session
- For persistent opt-out, store preference tied to user account
- If user toggles GPC off later, default behavior resumes (unless persistent opt-out also exists)

```
HTTP request from Brave/Firefox/etc:
  GET /home HTTP/1.1
  Host: example.com
  Sec-GPC: 1
  ...
       ↓
  Server: detect header
  → set "user-opted-out=true" for this session
  → do not load Facebook Pixel, Google Ads, etc.
  → backend: tag user as opt-out
  → persist if logged-in
```

## Other US State Laws

Post-CCPA, the patchwork:

**VCDPA — Virginia Consumer Data Protection Act** (effective January 1, 2023):
- More GDPR-like: opt-in for sensitive data
- Right to opt out of targeted advertising, sale, profiling for significant decisions
- Threshold: 100K residents or 25K + 50% revenue from sale
- No private right of action

**CPA — Colorado Privacy Act** (effective July 1, 2023):
- Similar to VCDPA: opt-in sensitive, opt-out targeted ads
- Universal opt-out mechanism (UOOM) required by July 2024 — accepts GPC
- AG enforcement, no private right
- Threshold: 100K residents or 25K + revenue tied to sale

**CTDPA — Connecticut Data Privacy Act** (effective July 1, 2023):
- Aligns with VCDPA/CPA
- Must accept UOOM
- No private right
- Threshold: 100K residents or 25K + 25% revenue from sale

**UCPA — Utah Consumer Privacy Act** (effective December 31, 2023):
- Weakest of the bunch; opt-out for sale + targeted ads only
- No opt-in for sensitive
- AG enforcement only
- Threshold: $25M revenue + 100K residents OR 25K + 50% revenue from sale

**Texas Data Privacy and Security Act (TDPSA)** (effective July 1, 2024):
- Similar to VCDPA
- Sensitive opt-in
- No private right
- Threshold based on small-business exclusion (broader than other state thresholds)

**Oregon, Montana, Iowa, Tennessee, Indiana, etc.** — adopted similar laws, mostly aligning with VCDPA model.

The convergence pattern:
- Opt-out for sale/share/targeted advertising (CCPA-like)
- Opt-in for sensitive data (GDPR-like)
- Universal Opt-Out Mechanisms (GPC-like)
- AG enforcement, generally no private right
- Mid-sized thresholds (100K residents typical)

```
US state privacy laws (active 2024+):
  CA, VA, CO, CT, UT, TX, OR, MT, IA, IN, TN, FL (limited), DE, NH, NJ, MN, MD, RI...
  
Federal proposal (American Privacy Rights Act, APRA):
  Stalled in Congress; bipartisan but contentious
  Would preempt state laws (controversial)
```

## Penalties Math

CCPA §1798.155:
- $2,500 per violation (negligent)
- $7,500 per violation (intentional or involving minors)
- Per consumer per violation
- Cure period: 30 days (CCPA original), removed for some violations under CPRA

Math example: a database breach exposes 10,000 California residents' PI without encryption.

- $2,500 × 10,000 = $25,000,000 (negligent)
- $7,500 × 10,000 = $75,000,000 (intentional)

But this is the **per-violation** ceiling. Real settlements have been smaller:
- Sephora 2022: $1.2M (failure to disclose sales/shares, no opt-out mechanism)
- DoorDash 2023: $375K
- Tilting Point Media 2023: $500K (kid's app, COPPA-related)

The CPPA's first **enforcement action** (2024) was against several **car manufacturers** for opaque data sales practices — penalties not yet finalized but expected to be in the millions.

**Private right of action** (§1798.150):
- Limited to data breach involving non-encrypted, non-redacted PI
- $100-$750 per consumer per incident, or actual damages (whichever greater)
- Pre-suit notice required (30 days)
- Cure right available

This is a class-action vector. The ~$750 per-resident floor for breach is structured to enable class actions.

```
Breach scenario:
  Plaintext storage of 1M California user records
  Breach exposes all 1M
       ↓
  Class action under §1798.150:
  1M × $750 = $750M maximum
  Likely settlement: $50M-$200M (per recent cases)
```

CPRA changes (effective 2023):
- Cure period removed for many violations
- CPPA can pursue civil action directly
- AG retains parallel authority

**Sephora case** (2022) was the first major enforcement action: they treated targeted-ad data flows as not-a-sale, didn't provide opt-out. Settled for $1.2M — small in the grand scheme but precedent-setting.

## The Compliance Engineering Model

Privacy-by-design parallels GDPR Article 25, but CCPA's structural approach is opt-out-by-default. Engineering implications:

**Data Inventory** (also called "data mapping"):
- Catalog every PI/SPI category in every system
- Track data flows: where it enters, where it goes, retention period
- Tag at field level (name, email, etc.)
- Maintain in living document; review quarterly

**Notice at Collection** (CCPA-required):
- Display before/at the point of PI collection
- List categories collected
- List business/commercial purposes
- Link to privacy policy
- Specify retention period

**Opt-Out Mechanisms**:
- "Do Not Sell or Share" link (single-click, no account required)
- "Limit Use of Sensitive PI" link (or combined link)
- Honor GPC signals in HTTP headers

**Rights Request Pipeline**:
- Intake (form, email, phone, agent)
- Verify identity
- Collect PI from all systems (data inventory drives this)
- Apply exceptions (deletion only)
- Notify service providers
- Respond within 45 days (extendable to 90)

**Multi-State Strategy** ("strictest law as baseline"):
- Treat all users as if covered by strictest applicable law
- For US users: apply CCPA + Virginia + Colorado + ... overlay
- Saves on per-state carve-outs but more compliance work
- Alternative: detect user state, apply state-specific rules

```
Privacy engineering stack:
  Data Inventory (catalog)
       ↓
  Tag at field level
       ↓
  Pipeline: collection → processing → retention → deletion
       ↓
  Rights handlers:
    - Right to know (export)
    - Right to delete (erasure across systems)
    - Right to correct (edit + propagate)
    - Right to opt out (suppression)
    - Right to limit SPI (narrow processing)
       ↓
  Notification flows:
    - Notice at collection
    - Privacy policy
    - "Do Not Sell or Share" link
    - "Limit Use of SPI" link
       ↓
  Detect signals:
    - GPC header
    - State residency (multi-state strategy)
       ↓
  Vendor management:
    - DPA / SP contracts with all processors
    - Audit rights
    - Subcontractor flowdown
```

For multi-state: pick a baseline (CCPA or strictest applicable). Implement features:

- Right to know → universal data export
- Right to delete → universal erasure pipeline
- Right to opt out → universal suppression flag
- Right to limit SPI (CCPA-only) → SPI-specific narrow-use mode
- Right to correct (CCPA, VA, CO) → universal edit-and-propagate

The "treat strictest as baseline" approach trades implementation simplicity for some over-compliance. Acceptable when the marginal cost of state-by-state branching exceeds the gain.

## CCPA vs GDPR Side-by-Side Engineering Decisions

| Engineering Decision | CCPA | GDPR | Universal recommendation |
|---|---|---|---|
| Default consent for analytics cookies | Opt-out OK (CCPA) | Opt-in REQUIRED | **Opt-in** (GDPR-strictest baseline) |
| First-party functional cookies | Allowed without consent | "Strictly necessary" exception | Allow |
| Email marketing consent | Implicit-from-purchase tolerated | Explicit opt-in REQUIRED | **Explicit opt-in** |
| User data export format | "Reasonable" — JSON or CSV | "Machine-readable" — JSON / structured | JSON with documented schema |
| Response time for DSAR | 45 days (extendable +45) | 30 days (extendable to 90) | **30 days** baseline |
| Right-to-correct | Required (CPRA) | Required | Universal correction UI |
| Breach notification | Cal Civ Code §1798.82 (no fixed hours) | 72 hours to authority | **72-hour SOP** |
| Children's data | Under 13: parental consent (CCPA) | Under 16: parental consent (GDPR) | Higher of the two = 16 |
| Sensitive data category | CPRA SPI (defined list) | GDPR Art. 9 special categories | Treat all as Art. 9 |
| Data sale consent | Opt-out (CCPA) | N/A directly; processed under lawful basis | Default no sale; opt-in if needed |

## CPRA Sensitive Personal Information — Engineering Implications

The CPRA-defined SPI categories (Cal Civ Code §1798.140(ae)):

```text
1. Government identifiers: SSN, driver's license, state ID, passport
2. Account credentials: account login + password/access code
3. Precise geolocation: within 1,850 feet (~565m)
4. Racial or ethnic origin
5. Religious or philosophical beliefs
6. Union membership
7. Mail/email/text contents (where the business is not the sender/recipient)
8. Genetic data
9. Biometric data for unique identification
10. Personal information collected/analyzed concerning health
11. Personal information collected/analyzed concerning sex life or sexual orientation
```

Engineering implications:

- **Encryption at rest**: SPI fields MUST be encrypted (vendor-specific guidance, but de facto requirement).
- **Limit-use right**: consumer can request that SPI use be limited to "necessary services + legally permitted purposes." UI requires a "Limit the Use of My Sensitive Personal Information" link prominently displayed.
- **Audit logging**: every read/write of SPI logged; retain logs ≥1 year for compliance audits.
- **Key segregation**: SPI encryption keys held separately from regular PI keys; rotation cadence at least annually.
- **Data lineage**: track every system that holds SPI; ensure deletion cascades to all of them.

## "Sale vs Share" Engineering Decision Tree

The CPRA distinguishes "selling" (monetary or other valuable consideration) from "sharing" (cross-context behavioral advertising). Both subject to opt-out.

```text
Are you sending PI to a third party?
├── No → not a sale, not a share
└── Yes
    ├── Is the third party a "service provider" with §1798.140(ag) contract restrictions?
    │   ├── Yes → not a sale, not a share (carve-out)
    │   └── No
    │       ├── Are they paying you (or providing valuable consideration)?
    │       │   └── Yes → SALE (opt-out required)
    │       └── Is the data being used for cross-context behavioral advertising?
    │           ├── Yes → SHARE (opt-out required)
    │           └── No → "disclosure for business purpose" (lower obligation)
    └── Most ad-tech (GAds remarketing, Meta CAPI, etc.) is SHARING under CPRA
        even if no money changes hands.
```

The Service Provider carve-out requires a written contract with these clauses (§1798.140(ag)):

1. Specifies purposes for which data is processed
2. Restricts processing to those purposes
3. Prohibits selling/sharing the PI
4. Prohibits combining with other PI from other sources
5. Includes audit rights
6. Specifies sub-processor approval mechanism

Without all six, you have a sale/share regardless of payment.

## Verifiable Consumer Request Engineering

```python
class VCRWorkflow:
    def submit_request(self, request_type, contact_info):
        # Step 1: Receive request via web form / email / toll-free phone
        request = ConsumerRequest(
            type=request_type,  # know / delete / correct / opt-out / limit-spi
            email=contact_info['email'],
            received_at=now(),
        )

        # Step 2: Verify the requester is who they say they are
        # CCPA mandates "reasonable degree of certainty"
        if request_type in ['know', 'delete', 'correct']:
            # Higher bar: send verification email with one-time code
            send_verification_email(request)
            request.verification_method = 'email_code'
        elif request_type in ['opt-out-sale', 'opt-out-share', 'limit-spi']:
            # Lower bar: opt-outs don't require strong verification
            request.verification_method = 'minimal'

        request.status = 'awaiting_verification'
        return request

    def fulfill_request(self, request):
        if request.status != 'verified':
            raise NotVerified()

        if request.type == 'know':
            data = collect_pi_for_user(request.user_id)  # must include 12-month look-back
            return present_data(data)
        elif request.type == 'delete':
            cascade_delete(request.user_id, exceptions=DELETION_EXCEPTIONS)
        elif request.type == 'correct':
            update_pi(request.user_id, request.corrections)
        elif request.type == 'opt-out-sale':
            set_flag(request.user_id, 'do_not_sell', True)
            propagate_to_processors(request.user_id, 'do_not_sell')
        elif request.type == 'opt-out-share':
            set_flag(request.user_id, 'do_not_share', True)
            propagate_to_ad_tech(request.user_id, 'do_not_share')
        elif request.type == 'limit-spi':
            set_flag(request.user_id, 'limit_spi_use', True)

        # Response within 45 days (extendable to 90)
        send_completion_notice(request)
        log_for_audit(request, retention='2 years')
```

## Penalties Math — Worked Examples

```text
Scenario 1: Misconfigured SDK leaks 50,000 user emails to third party.
  Per-violation: $2,500 (negligent) or $7,500 (intentional)
  Tier: Likely $2,500 (negligence); 50,000 × $2,500 = $125 million
  Mitigation: AG can show good-faith remediation reduces fine

Scenario 2: SDK leaks 5,000 children's (under 16) records intentionally.
  Per-violation: $7,500 (involves minors) and intentional
  5,000 × $7,500 = $37.5 million
  Plus civil class action under §1798.150 for breach: $100-$750 per consumer
  5,000 × $750 = additional $3.75M, parallel exposure

Scenario 3: Refused 1,000 verifiable consumer requests.
  Each refusal is a violation: 1,000 × $2,500 = $2.5 million
  CPRA removed the 30-day cure window for repeat offenders, so AG can
  proceed directly to enforcement action.

Scenario 4: $7,500 ceiling × 50,000 violations × 4 years statutory limit:
  $1.5 billion theoretical max. In practice settlements run 2-10% of theoretical.
```

## References

- California Consumer Privacy Act (CCPA) text — Cal Civ Code §1798.100-1798.199
- California Privacy Rights Act (CPRA / Prop 24) — https://oag.ca.gov/privacy/ccpa
- California Privacy Protection Agency (CPPA) — https://cppa.ca.gov/
- CPPA Final Regulations (March 2024) — https://cppa.ca.gov/regulations/
- California AG CCPA page — https://oag.ca.gov/privacy/ccpa
- Cal Civ Code §1798.140 (definitions) — https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?lawCode=CIV&sectionNum=1798.140
- Cal Civ Code §1798.105 (right to delete) — https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?lawCode=CIV&sectionNum=1798.105
- Cal Civ Code §1798.106 (right to correct) — https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?lawCode=CIV&sectionNum=1798.106
- Cal Civ Code §1798.121 (right to limit SPI) — https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?lawCode=CIV&sectionNum=1798.121
- Cal Civ Code §1798.135 (opt-out methods) — https://leginfo.legislature.ca.gov/faces/codes_displaySection.xhtml?lawCode=CIV&sectionNum=1798.135
- Global Privacy Control specification — https://globalprivacycontrol.org/
- California AG GPC enforcement letter (2021) — https://oag.ca.gov/news/press-releases/attorney-general-bonta-announces-investigative-sweep-issues-letters-businesses
- AG Sephora Settlement — https://oag.ca.gov/news/press-releases/attorney-general-bonta-announces-settlement-sephora-part-ongoing-enforcement
- Virginia Consumer Data Protection Act (VCDPA) — https://law.lis.virginia.gov/vacodefull/title59.1/chapter53/
- Colorado Privacy Act — https://coag.gov/resources/colorado-privacy-act/
- Connecticut Data Privacy Act — https://portal.ct.gov/AG/Sections/Privacy/Data-Privacy-Resource
- Utah Consumer Privacy Act — https://le.utah.gov/~2022/bills/static/SB0227.html
- Texas Data Privacy and Security Act — https://capitol.texas.gov/BillLookup/Text.aspx?LegSess=88R&Bill=HB4
- Oregon Consumer Privacy Act — https://www.doj.state.or.us/consumer-protection/id-theft-data-breaches/oregon-consumer-privacy-act/
- Multistate privacy law tracker (IAPP) — https://iapp.org/resources/article/us-state-privacy-legislation-tracker/
- NIST Privacy Framework — https://www.nist.gov/privacy-framework
- COPPA (children's) — 16 CFR Part 312, https://www.ftc.gov/legal-library/browse/rules/childrens-online-privacy-protection-rule-coppa
- HIPAA Privacy Rule — 45 CFR Parts 160 + 164
- GLBA Privacy Rule — 16 CFR Part 313
- FTC enforcement on privacy — https://www.ftc.gov/business-guidance/privacy-security
- Mactaggart 2018 ballot initiative archive — https://www.caprivacy.org/
- IAPP CCPA/CPRA Resource Center — https://iapp.org/resources/topics/ccpa-and-cpra/
- Future of Privacy Forum CCPA tracker — https://fpf.org/issue/ccpa/
- CPPA enforcement actions — https://cppa.ca.gov/enforcement/
- American Privacy Rights Act (APRA) draft — https://www.commerce.senate.gov/2024/4/cantwell-rodgers-discussion-draft-comprehensive-privacy
