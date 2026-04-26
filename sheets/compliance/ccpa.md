# CCPA / CPRA

California Consumer Privacy Act and the California Privacy Rights Act — opt-out privacy regime, verifiable consumer requests, Do Not Sell/Share mechanics, GPC signaling, and the engineering surface every California-facing product must implement.

## Setup

The California Consumer Privacy Act (CCPA) was signed into law on June 28, 2018 and took effect January 1, 2020. It was the first comprehensive consumer privacy statute in the United States, modeled loosely on the EU GDPR but built around an **opt-out** rather than opt-in framework. CCPA was the legislative response to a citizen-led ballot initiative ("Californians for Consumer Privacy") that threatened to enact a stricter privacy law via direct democracy; the legislature passed CCPA as a compromise to preempt the ballot.

The California Privacy Rights Act (CPRA) was passed by California voters as Proposition 24 on November 3, 2020. CPRA does not replace CCPA — it amends and substantially expands it. The CPRA's substantive provisions took effect January 1, 2023, with enforcement beginning July 1, 2023 (some delays applied). When practitioners say "CCPA" today they almost always mean "CCPA as amended by CPRA," and the statute is sometimes referred to interchangeably as "CCPA/CPRA."

CPRA created the **California Privacy Protection Agency (CPPA)** — the first dedicated state-level data protection agency in the United States. The CPPA has rulemaking, investigation, and enforcement authority. Before CPRA, enforcement sat with the California Attorney General; the AG retains concurrent enforcement authority but the CPPA is now the primary regulator. The CPPA publishes regulations at 11 CCR §7000 et seq.

```text
Timeline:
  2018-06-28  CCPA signed (AB 375)
  2020-01-01  CCPA effective
  2020-07-01  AG enforcement begins
  2020-11-03  CPRA passed (Prop 24)
  2023-01-01  CPRA substantive provisions effective
  2023-07-01  CPPA enforcement begins
  ongoing     CPPA rulemaking on cybersecurity audits, risk assessments, ADM
```

**Territorial scope.** CCPA protects "consumers" defined as natural persons who are California residents, regardless of where the personal information is processed. A business in Texas processing data of California residents is in scope. A business in California processing data of non-California residents is not protected by CCPA but may be in scope for the business-applicability test.

**Residency.** California residency is defined by California Code of Regulations Title 18 §17014 — every individual who is "in this state for other than a temporary or transitory purpose" or "domiciled in this state who is outside the state for a temporary or transitory purpose." Practically, businesses do not verify residency at request time; they treat any plausibly Californian request as a CCPA request.

**Pre-emption posture.** CCPA does not preempt other California or federal privacy law. HIPAA, GLBA, FCRA, COPPA, FERPA, and the California Confidentiality of Medical Information Act (CMIA) remain in force; CCPA carves out personal information already covered by those statutes.

```bash
# Quick orientation
echo "CCPA = the 2018 statute"
echo "CPRA = the 2020 amendment, effective 2023"
echo "CPPA = the agency CPRA created"
echo "All three are commonly referenced together as 'CCPA/CPRA'"
```

## Threshold for Applicability

CCPA applies to a "business" (Cal Civ Code §1798.140(d)). A business is a sole proprietorship, partnership, LLC, corporation, or other legal entity that:

1. Is organized or operated **for the profit or financial benefit** of its shareholders or owners, AND
2. Collects consumers' personal information (or has it collected on its behalf), AND
3. Alone or jointly determines the **purposes and means** of the processing, AND
4. Does business in California, AND
5. Meets ONE of the following thresholds:
   - **a)** Annual gross revenue exceeds **$25,000,000** (adjusted by CPPA for inflation; was $25M at CPRA passage, raised periodically)
   - **b)** Annually buys, receives, sells, or shares the personal information of **100,000 or more** consumers or households (raised from 50,000 by CPRA)
   - **c)** Derives **50% or more** of annual revenue from selling or sharing personal information

The thresholds are disjunctive — meeting any one triggers CCPA. The 50% revenue threshold catches data brokers regardless of size.

```text
Threshold table:
  Revenue  > $25M                          → in scope
  PI of   ≥100K consumers/households/year  → in scope
  Revenue ≥ 50% from selling/sharing PI    → in scope
```

**"Doing business in California"** is interpreted broadly. The California Franchise Tax Board's definition (Cal Rev & Tax Code §23101) is the conservative reference: actively engaging in any transaction for profit in California, organized in California, having California sales > $711,538 (2024 threshold), having California property > $71,154, or having California payroll > $71,154. Most online businesses with any California users meet this.

**Pre-CPRA vs post-CPRA threshold (b).** Before CPRA, the trigger was 50,000 consumers. CPRA doubled this to 100,000 — the practical effect is that smaller businesses fell out of scope, but mid-sized businesses (which already have to address CPRA-era SPI rules anyway) saw little change.

**Joint-venture and parent/subsidiary.** A "business" includes any entity that controls or is controlled by a covered business AND shares common branding (§1798.140(d)(2)). A small subsidiary using the parent's brand inherits CCPA applicability.

**Service providers and contractors.** These are NOT "businesses" under CCPA — they're separately regulated. A "service provider" (§1798.140(ag)) processes PI on behalf of a business under a written contract that restricts use to specified business purposes. A "contractor" (§1798.140(j)) is a similar role created by CPRA, with overlapping but slightly broader contractual obligations. Both are barred from "selling" or "sharing" the data they receive.

```text
Roles:
  Business        — determines purposes & means (CCPA-equivalent of "controller")
  Service provider — processes for business, contractually restricted ("processor"-like)
  Contractor      — like SP but typically more discretion in execution
  Third party     — anyone else who receives PI; "sale" or "sharing" applies
```

**Non-profits and government.** CCPA explicitly carves out non-profits ("not organized for the profit or financial benefit of its shareholders") and government agencies. Some non-profits have sister for-profit entities that ARE covered.

**Employee/B2B carve-outs.** Originally CCPA exempted personal information collected from job applicants, employees, and B2B contacts. CPRA repealed both carve-outs effective January 1, 2023; employee and B2B PI is now fully in scope.

## CCPA vs GDPR Comparison Table

| Dimension | CCPA / CPRA | GDPR |
| --- | --- | --- |
| Geographic scope | California residents | EU/EEA residents (and "establishment" reach) |
| Territorial reach | Business doing business in California | Establishment in EU OR offering goods/services to EU OR monitoring EU |
| Lawful basis required | NO — opt-out model; collection allowed by default | YES — must satisfy one of six bases (Art. 6) |
| Consent for processing | Required only for SPI under CPRA when "selling/sharing" or limiting use | Required for processing where consent is the chosen lawful basis |
| Default | Opt-out (you collect, consumer can object) | Opt-in (need basis BEFORE collection) |
| Penalties (max) | $2,500 per violation; $7,500 per intentional or involving minor | Up to €20M or 4% of global annual turnover, whichever higher |
| Private right of action | Only for data breaches (§1798.150) — $100–$750 per consumer per incident | Yes — Art. 79, 82 (compensation) |
| DPO required | NO — no equivalent | YES, in many cases (Art. 37) |
| Breach notification window | Not specified in CCPA itself; California Civ Code §1798.82 requires "without unreasonable delay" | 72 hours to supervisory authority (Art. 33) |
| "Personal information" definition | Broad — includes household data, inferences, profiling | Tied to identified or identifiable natural person |
| Inferences | Explicitly included | Implicit but less detailed |
| Right to be forgotten | Limited — 9 specific exceptions | Robust — Art. 17 with narrower exceptions |
| Right to know | Yes — categories + specific pieces | Yes — Art. 15 access |
| Right to portability | Subset of right to know (machine-readable) | Yes — Art. 20 |
| Right to correction | Yes (CPRA addition) | Yes — Art. 16 |
| Right to object/restrict | "Limit SPI" + opt-out of sale/share | Art. 18, Art. 21 |
| Automated decision-making | CPPA rulemaking ongoing | Art. 22 |
| Sensitive categories | "Sensitive Personal Information" — CPRA-defined | "Special categories" — Art. 9 |
| Children | Opt-in to sell required for <16; parent for <13 | Art. 8 — parental consent for <16 (member-state variation) |
| Service providers | Written contract restricting use | Art. 28 processor agreements |
| Cross-border transfers | Largely silent | Chapter V — adequacy, SCCs, BCRs |
| Regulator | CPPA + California AG | National DPAs + EDPB |
| Cure period | Removed by CPRA for most violations | None |
| Enforcement model | Administrative + AG civil suit + private breach actions | Administrative (DPA) + judicial |

```text
Mental model:
  GDPR = "ask first, with a basis"
  CCPA = "do what you want, but be transparent and let consumers opt out"

  GDPR rights are an explicit list; CCPA rights expanded as a Frankenstein.
  Most multi-jurisdiction businesses build to GDPR and layer CCPA-specific
  notices (DNS link, GPC, SPI limit) on top.
```

**A note on "consent."** CCPA does not require consent to collect or process. The exceptions are:
- Selling or sharing PI of consumers under 16 (opt-in required; under 13 requires parental opt-in)
- Selling, sharing, or use beyond the §1798.121(a) list of Sensitive Personal Information requires the consumer not to have exercised the "limit SPI" right (effectively an opt-out)

GDPR consent (Art. 7) is far stricter — freely given, specific, informed, unambiguous, and revocable. Cookie banner styles built for GDPR consent often fail CCPA opt-out — and CCPA opt-out flows often look like soft-opt-in to GDPR regulators.

## CPRA Updates (effective 2023) Summary

CPRA materially extended CCPA. The headline changes:

**1. Right to Correct.** Consumers may request correction of inaccurate PI. Previously absent from CCPA — joins the GDPR-aligned set.

**2. Right to Limit Use of Sensitive Personal Information.** A new "limit SPI" right and required link.

**3. New category: Sensitive Personal Information (SPI).** Defined in §1798.140(ae). Triggers extra obligations.

**4. New concept: "Sharing."** Distinct from "selling." Targets cross-context behavioral advertising. Now consumers can opt out of "selling AND sharing" via a single link.

**5. Mandatory data minimization.** §1798.100(c) — collection, use, retention, and sharing must be reasonably necessary and proportionate.

**6. Storage limitation / retention disclosure.** Notice at collection must include retention periods (or criteria for determining them) — a GDPR-style requirement.

**7. CPPA created.** Rulemaking and enforcement housed in the new agency.

**8. Service provider obligations strengthened.** Stricter contractual flow-down.

**9. Contractors defined.** Distinct from service providers.

**10. Audits and risk assessments.** Annual cybersecurity audit for high-volume processing; risk assessments before "significant risk" processing. CPPA finalizing regulations.

**11. Cure period removed.** AG and CPPA may now bring enforcement without offering a 30-day cure window for most violations.

**12. Employee and B2B carve-outs sunset.** Effective January 1, 2023, full coverage.

**13. Threshold raised.** From 50,000 to 100,000 consumers/households.

**14. Look-back window extended.** Right to Know now reaches all PI on or after January 1, 2022 (not just the prior 12 months).

**15. Whistleblower protections.** §1798.130(a)(7).

**16. Authorized-agent process.** Detailed regulations in 11 CCR §7063.

**17. Automated decision-making.** CPPA rulemaking authority granted to issue ADM and profiling regulations. Draft regulations released 2024-2025.

```bash
# CPRA migration checklist (boots-on-ground)
□ Privacy policy updated for CPRA (retention, SPI, "share")
□ "Do Not Sell or Share My Personal Information" link replaces "Do Not Sell"
□ "Limit the Use of My Sensitive Personal Information" link added (or combined: "Your Privacy Choices")
□ GPC signal honored
□ SPI inventory created
□ Right-to-correct workflow added
□ Look-back queries extended past 12-month rolling window
□ Service provider / contractor contracts re-papered
□ Employee privacy notice deployed
□ Records retention schedule documented
```

## Personal Information Definition (CCPA)

§1798.140(v) (formerly (o)) — Personal information means:

> Information that identifies, relates to, describes, is reasonably capable of being associated with, or could reasonably be linked, directly or indirectly, with a particular consumer or household.

The "or household" is significant — it broadens beyond individual identifiability. The standard is "reasonably capable of being associated" — looser than GDPR's "identified or identifiable." This catches data that GDPR might consider de-identified.

The statute lists illustrative categories (non-exhaustive):

```text
A. Identifiers — name, alias, postal address, unique personal identifier,
   online identifier, IP address, email, account name, SSN, driver's license
   number, passport number, similar identifiers.

B. Customer records — categories listed in Cal Civ Code §1798.80(e):
   signature, physical description, telephone number, education, employment,
   employment history, bank account number, credit card, debit card,
   medical information, health insurance information.

C. Protected classification — characteristics under California or federal law:
   race, religion, sexual orientation, gender identity, gender expression,
   age, etc.

D. Commercial information — records of property, products/services purchased,
   purchasing/consuming histories or tendencies.

E. Biometric information — physiological, biological, behavioral
   characteristics that can be used to establish individual identity.

F. Internet/network activity — browsing history, search history,
   information regarding interaction with website/app/ad.

G. Geolocation data — precise (≤1850 ft radius for SPI) or general.

H. Sensory data — audio, electronic, visual, thermal, olfactory, similar.

I. Professional/employment information.

J. Education information — non-public, defined under FERPA.

K. Inferences — drawn from any of the above to create a profile reflecting
   preferences, characteristics, predispositions, behavior, attitudes,
   intelligence, abilities, aptitudes.
```

**Inferences are explicitly listed.** A profile predicting a consumer's likelihood to buy a product is PI. GDPR captures this via "profiling" (Art. 4(4)) but CCPA names it directly.

**Exclusions.**
- "Publicly available information" lawfully made available by federal, state, or local government records, OR available widely from media OR from the consumer's website. Note: CPRA narrowed this — the consumer must have consented to the disclosure.
- De-identified information (must meet the §1798.140(m) standard — reasonable measures + business commitment + contractual prohibition on re-identification + no actual re-identification).
- Aggregate consumer information.
- PI covered by other federal statutes — HIPAA (PHI), GLBA (financial privacy), DPPA (driver's records), FCRA (consumer reports), FERPA (education records), Confidentiality of Medical Information Act, California Financial Information Privacy Act, the Insurance Information and Privacy Protection Act.

The GLBA carve-out is partial — it covers PI processed under the GLBA, but a financial institution still in scope for CCPA on its non-GLBA PI (employees, marketing, etc.).

```bash
# Decide: is the data field PI?
question="Could this field reasonably be linked to a consumer or household?"
answer="If yes → PI. Even hashed identifiers usually qualify."
note="The link doesn't have to be by you; if anyone else could re-link, it's PI."
```

**Pseudonymous identifiers (cookie IDs, advertising IDs, hashed emails, fingerprints) are PI.** CCPA explicitly includes "online identifier" and "IP address" in §1798.140(v)(1)(A).

## Sensitive Personal Information (CPRA)

CPRA introduced "Sensitive Personal Information" (SPI) at §1798.140(ae). The categories:

```text
SPI categories:

  1. Government identifiers
     - Social security number
     - Driver's license number
     - State identification card number
     - Passport number

  2. Account credentials
     - Account log-in
     - Financial account number, debit card number, credit card number
       in combination with required security/access code, password,
       or credentials allowing access to the account

  3. Precise geolocation
     - Defined as area within radius of 1,850 feet (≈564 m)
     - Even general geolocation NOT SPI; only precise

  4. Demographic / political / religious
     - Racial or ethnic origin
     - Religious or philosophical beliefs
     - Union membership

  5. Communication contents
     - Contents of mail, email, text messages
     - Unless the business is the intended recipient
     - Subject lines often a gray area

  6. Genetic data
     - DNA, genetic test results, family health history derived from genetic data

  7. Biometric for identification
     - Biometric data processed for the purpose of UNIQUELY IDENTIFYING a consumer
     - Mere measurement (e.g., step count) NOT SPI
     - Face geometry or fingerprint matching IS SPI

  8. Health, sex, sexual orientation
     - Personal information collected and analyzed concerning a consumer's health
     - Personal information concerning a consumer's sex life or sexual orientation
```

**Processing SPI.** §1798.121 — a consumer has the right to direct a business to limit use and disclosure of SPI to:
- Performing services or providing goods reasonably expected by an average consumer requesting them
- Detecting security incidents, protecting against malicious/deceptive/illegal action, and prosecuting those responsible
- Resisting attempts at fraud and identity theft
- Short-term, transient use (no profile building)
- Performing services on behalf of the business — order fulfillment, customer service
- Verifying or maintaining quality/safety of service or device, improving/upgrading/enhancing service or device

**The "limit SPI" right.** A required link "Limit the Use of My Sensitive Personal Information." Must be honored within 15 business days.

**Carve-out.** SPI used solely for the purposes listed above does NOT trigger the limit-SPI right's restriction. The CPPA's "Notice of SPI Use" regulations clarify which uses are within these expectations.

**Children + SPI.** Selling or sharing the SPI of consumers under 16 requires opt-in consent (parental for under 13).

```bash
# SPI inventory query
SELECT field_name, source, retention_days, processing_purposes
FROM data_inventory
WHERE classification = 'SPI'
ORDER BY field_name;
```

**SPI vs GDPR special categories.** Overlap but not identical. GDPR Art. 9 covers race, ethnic origin, political opinions, religious/philosophical beliefs, trade union membership, genetic data, biometric data for unique ID, health, sex life/orientation. CPRA adds government IDs and account credentials but does not directly include political opinions (though those typically fall under "profiling" or are inferred from other categories).

## Consumer Rights

CCPA/CPRA grants California consumers seven core rights. Each must be implemented as a usable workflow, not just a privacy-policy paragraph.

**1. Right to Know (§§1798.100, 1798.110, 1798.115).**
- Categories of PI collected
- Specific pieces of PI collected (the actual data)
- Categories of sources (you, public records, third parties)
- Business or commercial purposes for which PI was collected, sold, or shared
- Categories of third parties to whom PI was sold/shared/disclosed
- Look-back: pre-CPRA was 12 months; post-CPRA covers all PI collected on/after January 1, 2022

**2. Right to Delete (§1798.105).**
- Consumer can request deletion of PI the business has collected from the consumer
- Nine statutory exceptions (covered below)
- Business must direct service providers and contractors to also delete

**3. Right to Opt-Out of Sale (§1798.120).**
- Consumer can direct business not to sell PI
- "Do Not Sell My Personal Information" link required (now combined with "share")

**4. Right to Opt-Out of Sharing (CPRA — §1798.120).**
- "Sharing" added by CPRA — covers cross-context behavioral advertising
- Combined with sale into one opt-out link

**5. Right to Limit Use of Sensitive Personal Information (CPRA — §1798.121).**
- Limit use of SPI to the §1798.121(a) carve-out list
- "Limit the Use of My Sensitive Personal Information" link required

**6. Right to Correct (CPRA — §1798.106).**
- Consumer can request correction of inaccurate PI
- Business uses commercially reasonable efforts

**7. Right to Non-Discrimination (§1798.125).**
- Business may not deny goods/services, charge different prices, provide different quality, or threaten any of the above for exercising rights
- Exception: financial incentives may be offered if directly related to the value of the consumer's data, with prior opt-in consent

**Right to Equal Service and Price.** Subset of non-discrimination.

**Right to Data Portability.** Subset of Right to Know — for specific pieces of PI, business must provide in a portable, readily usable format that allows the consumer to transmit the information to another entity without hindrance.

```text
Engineering matrix:

  Right         | Endpoint                          | Window  | Auth
  ------------- | --------------------------------- | ------- | ----
  Know          | GET /privacy/me/disclosure        | 45 days | VCR
  Specific data | GET /privacy/me/data-export       | 45 days | VCR
  Delete        | POST /privacy/me/delete           | 45 days | VCR
  Opt-out       | POST /privacy/opt-out             | 15 days | none
  Limit SPI     | POST /privacy/limit-spi           | 15 days | none
  Correct       | POST /privacy/me/correct          | 45 days | VCR
```

**45-day response window.** §1798.130(a)(2) — must respond to verifiable requests for Know/Delete/Correct within 45 calendar days. Extendable by another 45 (90 total) for complex requests, with notice. Must respond in the same medium received unless the consumer specifies otherwise.

**15-day window for opt-out and SPI limit.** No verification required.

## Verifiable Consumer Request (VCR)

CCPA does not let just anyone submit a request — the business must reasonably verify the requester is the consumer they claim to be. The 11 CCR §7060 et seq. regulations specify "verifiable consumer request" standards.

**Two principles:**
1. Match the level of verification to the **risk** of disclosure (deletion of all data > sharing categories of data > sharing specific pieces).
2. Use the **least amount of information** necessary, retaining no more than needed.

**Standards by request type:**

```text
Request                 | Verification standard
----------------------- | -----------------------------------------
Categories (Know)        | "reasonable" — typically matching 2 data points
Specific pieces (Know)   | "reasonably high degree of certainty" — 3+ pieces
Delete                   | Same as Specific Pieces
Correct                  | Same as Specific Pieces
Opt-out / Limit SPI      | NO verification required (but may verify if abuse)
```

**Authenticated user.** If the consumer has an account, they may submit via the account UI after re-authenticating. This is the gold path — re-auth + 2FA satisfies the verification standard.

**Non-account user.** Match against information you already hold. Do NOT request more than necessary. Acceptable matches:
- Email + click confirmation
- Email + verification code sent to phone
- Government-issued ID compared against records (only if needed and proportionate)
- Notarized affidavit (rare; reserve for high-risk requests)

**Authorized agents.** Consumer may designate an agent. Business may require:
- Signed permission from the consumer
- Verification of consumer identity directly
- Verification of agent's authority (power of attorney; signed form)

**Response medium.** §1798.130(a)(2) — respond in the same medium the request was received in (email → email; postal → postal). Consumer may specify otherwise.

**Cost.** First request in a 12-month period must be free. Subsequent requests may be charged a "reasonable fee" or denied if "manifestly unfounded or excessive" — and the business must document this.

**Non-disclosure.** Do not disclose:
- Social security numbers
- Driver's license numbers / state ID numbers
- Financial account numbers
- Health insurance / medical ID numbers
- Account passwords
- Security questions and answers
- Unique biometric data generated from physical / biological characteristics

(See 11 CCR §7024(d) — these may not be disclosed even on a verified Know request; instead, business confirms collection of them.)

**Rejection notice.** If you cannot verify, reject the Know-specific-pieces request and treat it as a categories request if possible.

```python
# Pseudo-code for VCR handling
def handle_consumer_request(req):
    if req.type in ("opt_out", "limit_spi"):
        log_request(req)
        apply_immediately(req.consumer)
        notify(req, "honored within 15 days")
        return

    if req.consumer_authenticated:
        if req.type == "specific_pieces" or req.type == "delete":
            require_reauth_2fa()
        verification = "high_certainty"
    else:
        # match request data against records
        match_count = match_record_fields(req.identifiers)
        if req.type == "categories":
            verification = "reasonable" if match_count >= 2 else None
        else:
            verification = "high_certainty" if match_count >= 3 else None

    if verification is None:
        send_response(req, "we couldn't verify; resubmit with X")
        return

    schedule_response(req, deadline_days=45)
```

## Right to Know — Engineering

The Right to Know has two flavors and many sub-questions.

**Categories.** Sources, purposes, third-party recipients, categories of PI/SPI collected. This is closer to a privacy-policy snapshot personalized to the consumer.

**Specific pieces.** The actual data. This is the heavy lift — every system touching the consumer must contribute.

**Look-back.**
- Pre-CPRA: 12 months prior to receipt of request.
- CPRA: all PI on or after January 1, 2022 (no rolling cutoff).

```text
GET /privacy/me/disclosure
  Returns (categories form):
    {
      "request_id": "abc123",
      "as_of": "2026-04-25T12:00:00Z",
      "look_back_window": "from 2022-01-01",
      "categories_collected": [
        "identifiers",
        "internet activity",
        "geolocation (general)",
        "commercial info",
        "inferences"
      ],
      "spi_categories_collected": [
        "account credentials"
      ],
      "sources": [
        "directly from you",
        "your device when interacting with our service",
        "publicly available business records",
        "advertising partners"
      ],
      "business_purposes": [
        "providing the service",
        "fraud detection",
        "analytics",
        "personalization"
      ],
      "third_party_categories": [
        "service providers — hosting, analytics, payment",
        "advertising networks (sharing)",
        "law enforcement on request"
      ],
      "sale_or_share": {
        "sells": false,
        "shares": true,
        "categories_shared": ["identifiers", "internet activity"]
      }
    }
```

**Specific pieces export.** Machine-readable, portable. JSON or CSV is fine; archive (zip) is common when including documents/files. Provide a download URL with short-lived signed access.

```text
GET /privacy/me/data-export
  Returns:
    application/zip with structure:
      /profile.json           — account info
      /events.jsonl           — activity events (one per line)
      /messages/              — message bodies and metadata
      /uploads/               — files the consumer uploaded
      /derived.json           — inferences and profiles built about the consumer
      /sources.json           — where each datum came from
      /MANIFEST.json          — schema, generation timestamp
```

**Don't disclose the un-disclosable.** Filter out: SSN, DL, account passwords, security answers, financial account numbers, biometric raw values. Confirm collection of these in a separate field but do not include the values.

**Third-party recipients.** Categories, not entity names — though many businesses now list specific recipients to also satisfy state laws like Connecticut and Colorado that require entity-level disclosure. CPRA requires disclosure of categories of "third parties," "service providers," and "contractors" with which PI is shared.

**Inferences.** Often forgotten. If you maintain ML-derived scores, segments, or profiles about the consumer, those are PI and the consumer is entitled to them on a Right to Know request.

**Aggregation tactic.** Build a single internal "consumer-data subject" service that other internal systems push to (or that pulls from them via service contracts). This becomes the source of truth for Right to Know exports. Without this, you'll miss data in some forgotten corner.

```bash
# Sample data-subject service request
curl -X POST https://privacy.example.com/internal/dsr \
  -H "Authorization: Bearer ${INTERNAL_TOKEN}" \
  -d '{
    "consumer_id": "user_42",
    "request_type": "right_to_know_specific",
    "look_back_start": "2022-01-01"
  }'
```

## Right to Delete — Engineering

Deletion sounds simple. It is not.

**Cascade.** Every system holding the consumer's data must delete or anonymize:
- Application database (primary record + child rows)
- Analytics warehouse (event tables, derived aggregations)
- Logs (application, web server, audit, security)
- Backups (eventual — at next backup rotation)
- Search indices (Elasticsearch, OpenSearch, internal search)
- Caches (Redis, CDN, edge)
- Message queues (drain — though usually short-lived)
- ML feature stores and training data
- Email / SMS / push provider records (Twilio, SendGrid, etc.)
- Service provider systems (must direct them under §1798.105(c))
- Replication targets (read replicas, geo-replicas)

**Backups.** You can't restore a 90-day-old backup and undo deletion. Acceptable practice: do not restore deleted data to live systems on backup restore; if a backup is restored, immediately re-run pending deletion requests against the restored data.

**The 9 exceptions (§1798.105(d)).** A business or service provider may retain consumer's PI to:

```text
a) Complete the transaction for which PI was collected, fulfill a contract
   with the consumer, provide a good or service requested by the consumer
   or reasonably anticipated within the context of the business's ongoing
   business relationship with the consumer, or otherwise perform a contract
   between the business and the consumer.

b) Help to ensure security and integrity to the extent the use of PI is
   reasonably necessary and proportionate. Detect security incidents.
   Protect against malicious, deceptive, fraudulent, or illegal activity.
   Prosecute those responsible.

c) Debug to identify and repair errors that impair existing intended
   functionality.

d) Exercise free speech, ensure the right of another consumer to exercise
   their free speech rights, or exercise another right provided for by law.

e) Comply with the California Electronic Communications Privacy Act
   (CalECPA — Penal Code §1546 et seq.).

f) Engage in public or peer-reviewed scientific, historical, or statistical
   research in the public interest that adheres to all other applicable
   ethics and privacy laws, when the business's deletion of the information
   is likely to render impossible or seriously impair the achievement of
   such research, if the consumer has provided informed consent.

g) Enable solely internal uses that are reasonably aligned with the
   expectations of the consumer based on the consumer's relationship with
   the business and compatible with the context in which the consumer
   provided the information.

h) Comply with a legal obligation.

i) Otherwise use the consumer's PI, internally, in a lawful manner that
   is compatible with the context in which the consumer provided the
   information.
```

**Don't over-claim exceptions.** "Reasonably necessary and proportionate" is the standard. Simply tagging records with a retention reason is good practice — when audited, you can defend the retention decision per record.

**Deletion approach options.**
- **Hard delete** — DELETE FROM users WHERE id = ? CASCADE. Simple, irreversible.
- **Anonymize** — replace consumer-identifying fields with synthetic/null values, keeping the row for analytics integrity. Acceptable if anonymization is irreversible (de-identification standard at §1798.140(m)).
- **Tombstone** — mark deleted, hide from app, retain for the exception. Document the exception per record.

**Direct service providers and contractors.** §1798.105(c) — must direct them to delete. Contractually required under §1798.140(ag)/(j). Practically: an API call or email to each SP/contractor with a list of consumer IDs.

**Confirmation.** §1798.130(a)(3) — confirm to the consumer the deletion has been completed. Include any data retained under exception with the basis.

```python
# Skeleton deletion pipeline
def delete_consumer(consumer_id, request_id):
    record_request(request_id, "delete", consumer_id)
    sp_targets = get_service_providers_with_data(consumer_id)
    for sp in sp_targets:
        sp.request_delete(consumer_id, request_id)

    # System cascade
    for system in [app_db, analytics, logs, search, cache, ml_features, queues, replicas]:
        system.delete_or_anonymize(consumer_id, exception_log=True)

    # Backups: scheduled for next rotation
    schedule_backup_replay(consumer_id, request_id, days_ahead=90)

    # Confirm
    notify_consumer(consumer_id, "deletion complete with exceptions documented")
```

## Do Not Sell / Do Not Share — Engineering

The most operationally visible CCPA obligation.

**The link.** Required: a clear and conspicuous "Do Not Sell or Share My Personal Information" link on the business's homepage. Required: the link on any page where PI is collected. (CPRA permits a single combined "Your Privacy Choices" link — see below — that combines DNS and Limit-SPI.)

**Title rules.**
- Pre-CPRA: "Do Not Sell My Personal Information"
- Post-CPRA: "Do Not Sell or Share My Personal Information" (because "share" is now a separate concept)
- Combined: "Your Privacy Choices" — alongside the CPRA-defined opt-out icon (the blue/white toggle)

**Compliance window.** Business must comply within **15 business days** of receiving an opt-out request. Some practitioners apply 15 calendar days for safety.

**No verification.** Opt-out is no-verification — the consumer's word suffices.

**Persistence.** Honor at minimum 12 months. After 12 months, you may re-engage and ask if they want to opt back in (via a non-discriminatory mechanism).

**The "selling" definition (§1798.140(ad)).**

> "Sell," "selling," "sale," or "sold," means selling, renting, releasing, disclosing, disseminating, making available, transferring, or otherwise communicating orally, in writing, or by electronic or other means, a consumer's personal information by the business to a third party for monetary or other valuable consideration.

"Other valuable consideration" is broad. The Sephora settlement (CA AG, August 2022, $1.2M) clarified that exchanging PI for analytics or ad-targeting tools constitutes "other valuable consideration."

**The "sharing" definition (§1798.140(ah)).**

> "Sharing" means... communicating... a consumer's personal information by the business to a third party for cross-context behavioral advertising, whether or not for monetary or other valuable consideration.

"Cross-context behavioral advertising" — targeting based on PI from a context other than the one the consumer is in (i.e., retargeting). This was added because Sephora and many businesses argued they "didn't sell" data even though they used it for retargeting.

**Service-provider exception.** Disclosure to a service provider IS NOT a sale or sharing if:
1. Written contract that meets §1798.140(ag) requirements
2. SP cannot retain, use, or disclose PI for any purpose other than the specified business purpose
3. SP cannot sell/share

This is the lever many businesses use for analytics tools and ad-tech: "we don't sell or share — we use service providers." But that requires the contract to actually restrict use; a permissive Google Analytics agreement is questionable.

```text
Decision tree — disclosure type:

  Is there a written contract per §1798.140(ag)?
    NO  → likely SALE or SHARE
    YES → is the contract sufficiently restrictive (no use for own purposes,
          no sale/share by SP)?
            NO  → likely SALE or SHARE
            YES → can the recipient combine with other data for
                   cross-context targeting?
                     YES → SHARE
                     NO  → SP / not a sale or share
```

**Honor signals.** §1798.135(b) and 11 CCR §7025 — business must process **opt-out preference signals** sent by the user agent. Currently the only widely-deployed opt-out preference signal is **Global Privacy Control (GPC)**.

**Opt-out preference signal vs link.** Per regulations, a business that processes a valid opt-out preference signal does NOT need a link IF it already provides notice and the signal is processed correctly. Most businesses keep the link anyway.

**Cookie banner.** Cookie banners are not enough on their own. CCPA opt-out is a different model from GDPR consent. Banner UX must distinguish: in EU jurisdictions, "accept/reject all"; in California, "Do Not Sell or Share" toggle defaulting OFF unless the user opted in OR sending a GPC signal.

**Implementation pattern.**

```python
# Express middleware
def ccpa_middleware(request, response, next):
    state = detect_us_state(request)  # IP geolocation, account region, etc.
    has_gpc = request.headers.get("Sec-GPC") == "1"
    has_optout_cookie = request.cookies.get("ccpa_optout") == "1"
    in_california = state == "CA"

    if in_california and (has_gpc or has_optout_cookie):
        request.context["ccpa_optout"] = True
        # downstream code consults this flag before sale/share

    next()
```

**Downstream behavior on opt-out.**
- Set `ccpa_optout=1` cookie (long-lived).
- Skip ad-tech pixels (Meta, Google Ads, TikTok, etc.).
- Disable third-party analytics that share data outside SP boundary.
- Send signals to ad networks (e.g., Google "Restricted Data Processing") through proper API.
- Mark profile as "no-sale-no-share" in CDP / CRM.

**Confirm to user.** UI affirmation that opt-out is in effect; cookie banner shows current state.

## Global Privacy Control (GPC)

GPC is a browser-level signal designed to communicate "do not sell or share" to all sites the user visits. Rather than the user clicking a link on each site, the browser sends a single opt-out preference for everyone.

**Signal.** HTTP request header:

```text
Sec-GPC: 1
```

DOM API:

```javascript
if (navigator.globalPrivacyControl) {
  // user has GPC enabled
}
```

**Specification.** https://globalprivacycontrol.org/

**Browser support.**
- Firefox — built-in (Settings → Privacy & Security → Send websites a "Do Not Track" signal AND a Global Privacy Control signal)
- Brave — built-in (default ON in many builds)
- DuckDuckGo browser — built-in
- Chrome — extension only (e.g., Privacy Badger)
- Safari — extension only

**Server-side implementation.** Detect, set the opt-out, persist, signal to ad-tech:

```python
@app.before_request
def detect_gpc():
    if request.headers.get("Sec-GPC") == "1":
        # Treat as opt-out of sale/share
        g.ccpa_optout = True
        # If user is authenticated, persist on their profile
        if g.user:
            g.user.optout_sale_share = True
            g.user.save()
        # Set cookie for non-auth users so downstream pages know
        response.set_cookie("ccpa_optout", "1", max_age=365*24*3600,
                            secure=True, samesite="Lax")
```

**Enforcement reality.** California AG announced (and enforced) that businesses must respect GPC as a "Do Not Sell or Share" signal. The Sephora settlement (Aug 2022, $1.2M) cited failure to honor GPC. Subsequent settlements with DoorDash and others reinforced this.

**Combined with cookie banner.** When a user with GPC visits, the banner should:
- Pre-check "reject all sale/share" or hide the banner
- Affirmatively tell the user "we honored your GPC signal"
- Not undo GPC opt-out via banner click

**GPC vs DNT.** Do Not Track (DNT) is the older, voluntary signal that businesses largely ignored. GPC is legally recognized in California (and elsewhere — Colorado, Connecticut, etc.). Treat GPC differently: it has teeth.

## Right to Limit Sensitive PI Use (CPRA)

§1798.121 — consumer can direct the business to limit use and disclosure of SPI to the carve-out list.

**The carve-out list (§1798.121(a)).** Use of SPI is permitted, despite a "limit" direction, for:

```text
1. Performing the services or providing the goods reasonably expected by
   an average consumer who requests those goods or services.

2. Detecting security incidents, protecting against malicious, deceptive,
   fraudulent, or illegal actions, and prosecuting those responsible.

3. Resisting attempts at fraud and identity theft, and verifying or
   maintaining the quality or safety of a service or device that is owned,
   manufactured, manufactured for, or controlled by the business.

4. Short-term, transient use, including non-personalized advertising
   shown as part of a consumer's current interaction with the business,
   provided that the consumer's PI is not disclosed to another third party
   and is not used to build a profile about the consumer or otherwise alter
   an individual consumer's experience outside the current interaction.

5. Performing services on behalf of the business — maintaining or servicing
   accounts, providing customer service, processing or fulfilling orders
   and transactions, verifying customer information, processing payments,
   providing financing, providing analytic services, providing storage,
   or providing similar services on behalf of the business.

6. Verifying or maintaining the quality or safety of, and improving,
   upgrading, or enhancing, the service or device.
```

**Required link.** "Limit the Use of My Sensitive Personal Information" — clear, conspicuous, on homepage and any page where SPI is collected. Or combined into "Your Privacy Choices."

**Compliance window.** 15 days, like DNS/DNS-S.

**No verification.** Like opt-out, no VCR required.

**SPI in advertising.** Advertising and marketing using SPI is NOT in the carve-out list — meaning a "limit SPI" consumer cannot have their SPI used for advertising. This is significant for businesses that profile based on SPI (e.g., political affiliation for targeted ads).

**SPI uniquely identifying biometrics.** A passing thought: a fingerprint matched against a stored template for authentication = SPI; a fingerprint pattern stored for fraud detection across users = SPI for ID purposes; mere biometric measurement without ID intent (heart rate sensor) = NOT SPI.

```python
# SPI use gate
def can_use_spi(user, purpose):
    if not user.spi_limited:
        return True  # consumer hasn't limited

    permitted_purposes = {
        "service_delivery", "security", "fraud_prevention", "transient_ad",
        "service_on_behalf_of_business", "quality_safety_improvement",
    }
    return purpose in permitted_purposes
```

## Right to Correct (CPRA)

§1798.106 — added by CPRA. A consumer may request correction of inaccurate PI a business maintains.

**Process.**
1. Verifiable consumer request (specific-pieces standard).
2. Business uses **commercially reasonable efforts** to correct.
3. Business may consider:
   - The nature of the PI
   - The purposes of processing
   - Records of the PI
   - Documentation provided by the consumer
4. Business may delete the disputed information if correction would be impossible or pose disproportionate effort.
5. Business may refuse if it has a good-faith reason to believe the data is accurate. Must inform consumer with the basis and right to lodge a complaint.

**Engineering.**
- Account profile editing UI handles the simple cases (name, address, email).
- Right-to-Correct API for fields not editable in the UI (derived data, inferences, third-party-sourced data).
- Cascade: corrections must propagate to derived data, ML features, search indices.
- Audit log: keep records of correction requests and responses for 24 months.

```python
def handle_correct_request(req):
    verify_vcr(req, level="high_certainty")
    field = req.field
    old_value = consumer.get(field)
    new_value = req.proposed_value
    if business_believes_accurate(field, old_value, evidence=req.evidence):
        log_decision(req.id, "refused", reason="business deems accurate")
        notify_consumer(req, "we deem the data accurate; here's why")
        return
    consumer.set(field, new_value)
    cascade_correction(consumer, field, new_value)
    log_decision(req.id, "corrected")
    notify_consumer(req, "corrected")
```

**Inferences and derived data.** A consumer can request correction of inferences. The business may rebuild the profile from corrected inputs, or remove the inference if it's not reproducible.

## Notice at Collection

§1798.100(b) — at or before the point of collection, the business must inform the consumer:

```text
a) Categories of PI to be collected
b) The purposes for which the PI is used
c) Whether each category is sold or shared (CPRA)
d) Categories of SPI to be collected (CPRA)
e) The purposes for which the SPI is used (CPRA)
f) Length of retention (or criteria for determining it) (CPRA)
g) Link to privacy policy
```

**"At or before the point of collection."** When the form loads, before submit; when an account is created; when a cookie is set; when the camera turns on. Practical implementation: a banner, a tooltip, an inline notice, or a link prominently labeled "Notice of Collection."

**Just-in-time notices.** New collection contexts trigger new notices. Adding a feature that turns on the microphone? New just-in-time notice. Starting to share with a new advertising network? New notice.

**Form-level notice.**

```html
<form>
  <p class="notice-at-collection">
    We collect your <strong>name, email, and IP address</strong> to provide
    the service. We do not sell or share this information.
    <a href="/privacy">See our privacy policy</a>.
    Retention: while your account is active + 12 months.
  </p>
  <input type="email" name="email" required>
  <input type="text" name="name" required>
  <button type="submit">Sign up</button>
</form>
```

**Layered notice.** Top layer: brief, on the form. Bottom layer: full privacy policy.

**Sale or share.** If you sell or share, you must say so at collection AND link to the DNS/DNS-S link.

## Privacy Policy Required Content

Cal Civ Code §1798.130(a)(5); 11 CCR §7011 — privacy policy contents:

```text
Required:
□ Description of consumer rights (Know, Delete, Correct, Opt-Out Sale/Share,
  Limit SPI, Non-Discrimination)
□ How to submit each request (web form, email, toll-free phone if you have
  brick-and-mortar; otherwise web form is enough)
□ Methods for verifying consumer identity
□ Authorized agents process
□ Categories of PI collected (last 12 months pre-CPRA; since 2022-01-01 post-CPRA)
□ Sources of PI
□ Business or commercial purposes
□ Categories of third parties to whom PI was disclosed
□ Categories of third parties to whom PI was sold or shared (separately for each)
□ Whether the business sells/shares PI of consumers under 16
□ Categories of SPI collected, purposes, and disclosure (CPRA)
□ Length of retention or criteria for determining it (CPRA)
□ "Do Not Sell or Share My Personal Information" link
□ "Limit the Use of My Sensitive Personal Information" link
□ Contact information for questions
□ Date of last update
```

**Annual update.** §1798.130(a)(5)(A) — must be updated at least once every 12 months.

**Employee/applicant policy.** Since CPRA carve-outs sunset, employees and applicants are covered. Many businesses maintain a separate Employee/Applicant Privacy Notice for clarity (workers' compensation data, payroll, performance reviews, etc.).

**B2B notice.** Since CPRA, B2B data is in scope. The privacy policy must describe B2B data collection and processing.

**Children.** Special handling for under-16 — opt-in to sell/share required. Policy must describe this.

## Authorized Agents

§1798.130(a)(2)(A); 11 CCR §7063 — a consumer may designate an authorized agent to make CCPA requests.

**The mechanic.** The agent submits the request on behalf of the consumer. Business may verify:
1. The consumer's identity (the usual VCR matching)
2. The agent's authority — typically a written/signed permission from the consumer
3. Optionally, direct contact with the consumer to confirm

**California-resident requirement.** Agents themselves do not have to be Californian, but the consumer must be.

**Power of attorney.** A valid Power of Attorney under California Probate Code §4000 et seq. is sufficient evidence of agency authority and absolves the business from independent verification of the agent's authority.

**Authorized-agent businesses.** A small industry exists:
- **Permission Slip** (Consumer Reports) — submits requests on behalf of consumers
- **Mine** — privacy assistant
- **OptOutPrescreen** (FCRA-specific but adjacent)
- **DataGrail Reverse-DSR** — represents consumers
- **DeleteMe / Abine** — broker opt-out services

These services often submit thousands of requests at scale. Businesses must have a workflow that handles bulk authorized-agent requests without extra friction.

**Bulk-request reality.** A signed permission per consumer + a list of consumers from the agent. Businesses commonly require:
- A scanned consumer signature (per consumer or in a master list)
- A power of attorney
- Direct confirmation from the consumer (email click-through)

## Service Provider Agreement Requirements

§1798.140(ag); 11 CCR §7050 — a "service provider" processes PI on behalf of a business. To qualify as an SP (and thus avoid the "sale/share" treatment), the contract must:

```text
□ Be a written contract
□ Identify the specified business purpose(s) for which PI is processed
□ Prohibit the SP from selling or sharing the PI
□ Prohibit the SP from retaining, using, or disclosing the PI for any purpose
  other than for the specified business purposes — including using the PI
  to combine with PI received from other sources
□ Prohibit the SP from retaining, using, or disclosing the PI outside of
  the direct business relationship between the SP and the business
□ Require the SP to comply with applicable obligations under CCPA and to
  provide the same level of privacy protection as required by CCPA
□ Allow the business to monitor SP's compliance through audits and
  inspections
□ Require the SP to notify the business if it can no longer meet its
  obligations
□ Allow the business, upon notice, to take reasonable and appropriate
  steps to stop and remediate unauthorized use of PI
□ Require the SP to flow down the obligations to its subcontractors
□ Require the SP to assist the business in fulfilling consumer requests
  (delete, correct, etc.)
```

**Subcontractor approval.** Like GDPR Art. 28(2), the SP cannot use a subcontractor without business authorization (general or specific). The SP must flow down all CCPA obligations.

**Audits.** Business must have the right to audit. Practical implementation: a SOC 2 report typically suffices in lieu of physical audit.

**Notice if non-compliant.** If the SP determines it can no longer meet its obligations, it must notify the business. The business then must take "reasonable and appropriate steps."

**Contractor.** §1798.140(j) — defined separately from SP. Similar contractual requirements but typically applied to entities with broader execution discretion (e.g., a marketing agency). The CCPA-compliant contract template often serves both.

**The Sephora teaching.** In the Sephora settlement, the CA AG found that Sephora's analytics and ad-tech contracts did NOT meet the SP standard — the recipients could combine Sephora's PI with other data for their own purposes. This converted what Sephora called "sharing with service providers" into a sale.

## Sale vs Service-Provider Distinction

The single most important compliance question for ad-tech: is it a sale, a share, or a service-provider relationship?

**Sale.** Disclosure to a third party for monetary or other valuable consideration. The recipient can use the data for its own purposes.

**Share.** Disclosure to a third party for cross-context behavioral advertising, regardless of consideration. Specific to ad-tech.

**Service provider.** Disclosure under §1798.140(ag) contract; recipient uses only for specified business purposes; no sale/share by SP; no combination with other data.

**Decision matrix:**

```text
                              | Sale | Share | SP/Contractor
----------------------------- | ---- | ----- | -------------
Recipient can use for own     |  Y   |   Y   |       N
  purposes                    |      |       |
Recipient can combine with    |  Y   |   ?   |       N
  other-source data           |      |       |
Recipient pays / valuable     |  Y   |  N/A  |       N or fee-for-service
  consideration               |      |       |
Cross-context behavioral ads  |  ?   |   Y   |       N
Written §1798.140(ag) contract|  N   |   N   |       Y
```

**Common ad-tech tools and their typical classification:**

```text
Tool                          | Default classification (no special config)
----------------------------- | -------------------------------------------
Google Analytics 4 (Universal | Sale or share unless GA4 in "Restricted
   Analytics)                 | Data Processing" mode + signed Google
                              | Customer Data Processing Terms = SP
Google Ads (remarketing)      | Share (cross-context) unless restricted
Meta Pixel / CAPI (default)   | Share or sale
TikTok Pixel (default)        | Share or sale
Hotjar / FullStory            | SP if contracted properly; otherwise share
Mixpanel / Amplitude / Heap   | SP if contracted properly
Segment / RudderStack         | SP (CDP collects, routes; downstream tools
                              | may be sale/share)
Salesforce CRM                | SP
Stripe / Square (payment)     | SP
Mailchimp / SendGrid          | SP
AWS / GCP / Azure (hosting)   | SP
```

**The Restricted Data Processing flag.** Google offers a "Restricted Data Processing" mode that limits how Google uses the data. With proper terms, this can move GA4 from "share" to "SP." Set it via:

```javascript
// Google tag with RDP for California users
gtag('config', 'GA_MEASUREMENT_ID', {
  'restricted_data_processing': true
});
```

**Risk-tolerance question.** Even with RDP enabled and contract in place, conservative counsel often advises treating ad-tech as "sharing" and respecting the opt-out anyway. The legal risk of being wrong (a $7,500 intentional violation per consumer) is large. Many businesses' default policy: opt-out signal = stop all ad-tech, period.

**Sephora-style "selling without saying so."** The most common CCPA enforcement pattern. Business doesn't disclose sales because it doesn't believe it's selling, but the AG/CPPA disagrees.

## Audits and Risk Assessments (CPRA)

§1798.185 — directs the CPPA to issue regulations for cybersecurity audits and risk assessments. Final regulations issued / pending as of 2025; effective dates phasing in.

**Cybersecurity audit.** Required for businesses whose processing presents "significant risk." Indicators (per draft CPPA regs):
- Annual revenue > $25M AND processing PI of >5M consumers
- Processing SPI in volume
- Selling/sharing in volume
- Use of automated decision-making

Audit must:
- Be performed by a qualified, objective, independent professional
- Cover the controls in CIS Critical Security Controls or equivalent
- Be documented in writing
- Renewed annually
- Submitted to CPPA on request

**Risk assessment.** Required before processing that presents significant risk. Documents:
- Categories of PI/SPI processed
- Operational benefit and necessity
- Risks to consumers (privacy, discrimination, error)
- Safeguards
- Whether risks outweigh benefits

Submitted to CPPA on request. CPPA may publish summarized findings.

**Automated decision-making.** Draft CPPA regulations require:
- Pre-use notice to consumers
- Right to opt out of ADM in certain contexts
- Right to access information about the ADM logic and outcomes
- Risk assessment for ADM use

These regulations are still in flux at time of writing — track CPPA rulemaking.

## Penalties

**Administrative civil penalties.** Cal Civ Code §1798.155:

```text
$2,500  per violation
$7,500  per intentional violation, OR per violation involving consumer
        under 16 years old
```

"Per violation" is per consumer affected. A violation impacting 1,000 consumers is potentially 1,000 violations — $2.5M to $7.5M base.

**Cure period (removed).** Pre-CPRA, businesses had 30 days to cure a violation after notice. CPRA removed the automatic cure period for most violations. The CPPA or AG may permit cure at their discretion.

**Private right of action.** §1798.150 — limited to data breaches. Consumer may sue if their non-encrypted, non-redacted PI (limited categories — name + SSN, DL, account # + access code, medical info, health insurance, biometric, genetic, email + password, etc.) was subject to unauthorized access due to the business's failure to implement reasonable security.

**Statutory damages.** $100 to $750 per consumer per incident, OR actual damages — whichever is greater. Class actions common.

**Notice and cure for breach actions.** Plaintiff must give 30 days' notice of the alleged violation; business has 30 days to cure for statutory damages claims (not actual-damages claims). Cure typically means: confirm the issue, remediate, give consumer a written statement.

**Equitable relief.** Court may grant injunctions in private actions.

**Recent enforcement.**
- **Sephora (Aug 2022)** — $1.2M settlement; failure to disclose sales, failure to honor GPC, failure to comply with cure within 30 days.
- **DoorDash (Feb 2024)** — $375K; sale of PI without notice + GPC failure.
- **Tilting Point Media (June 2024)** — $500K; children's data and SPI handling.
- **Honda (March 2025)** — $632K; verification standards, agent rejection, share-vs-sell.
- **Multiple lawsuits in private breach actions** — TikTok, Meta, others.

```text
Penalty math (illustrative):
  100,000 consumers affected × $2,500 = $250M (theoretical max)

  Reality: typical settlements range from $375K to a few million,
  often plus injunctive relief and ongoing oversight.
```

## Common Errors

**Treating CCPA opt-out as GDPR consent.** Different model. CCPA: collection allowed by default, consumer opts out. GDPR: lawful basis required for collection. Implementing a GDPR consent banner alone does not satisfy CCPA — and CCPA's "Do Not Sell or Share" link does not satisfy GDPR.

**Not honoring GPC signal.** A widely-cited basis for AG enforcement (Sephora, others). The Sec-GPC HTTP header must be detected server-side and treated as opt-out.

**"Do Not Sell or Share" link buried in footer.** Required to be "clear and conspicuous." Buried link → likely not compliant. CPPA regulations require the link to be displayed in a way that "draws the consumer's attention."

**Treating cookie consent banner as sufficient for CCPA.** Cookie consent ≠ CCPA opt-out. Need a distinct DNS link AND must respect GPC.

**Failing to designate authorized-agents process.** Required by §1798.135.

**Not training customer-facing staff.** §1798.130(a)(6) and 11 CCR §7011 — staff handling consumer requests must be trained.

**Selling SPI without SPI-specific opt-in (where required).** Selling/sharing SPI of under-16 consumers requires opt-in.

**Service provider agreement with permissive use clauses.** A pre-CCPA SaaS agreement that lets the vendor "improve their products" with your data is likely not a §1798.140(ag)-compliant contract.

**Not maintaining records of consumer requests.** Required for 24 months. Must include date received, date complied/denied, basis.

**Imposing fees on first request.** Free for first request in 12-month period; only "manifestly unfounded or excessive" follow-ups can be charged.

**Forgetting employee privacy.** Since 2023, employees and applicants are in scope. Many businesses have a worker-facing privacy notice that's never been updated.

**Forgetting B2B contacts.** Same — since 2023.

## Other US State Privacy Laws

The "patchwork" — at least 19 states have passed comprehensive privacy laws as of 2025. Common features and divergences:

**Common features (most states except California):**
- Opt-in for processing of "sensitive data"
- Opt-out of "targeted advertising," "sale," and "profiling for legal/significant effects"
- Right to access, delete, correct, port
- Privacy policy disclosure
- DPA-style processor contracts
- Universal opt-out signals (some)
- Penalties typically $7,500 per violation

**California-specific (not in most others):**
- Private right of action for breaches
- Prohibition on "sharing" as a separate concept
- Look-back to 2022
- Authorized-agent regime
- CPPA agency
- "Limit SPI" right

**State-by-state highlights:**

```text
State  | Law              | Effective       | Notes
------ | ---------------- | --------------- | -------------------------
CA     | CCPA / CPRA      | 2020 / 2023     | Original; opt-out model
VA     | VCDPA            | 2023-01-01      | Opt-in for sensitive
CO     | CPA              | 2023-07-01      | UOOM (universal opt-out)
CT     | CTDPA            | 2023-07-01      | UOOM
UT     | UCPA             | 2023-12-31      | Narrowest scope; no UOOM
IA     | ICDPA            | 2025-01-01      | Narrow rights
IN     | INCDPA           | 2026-01-01      | GDPR-flavored
TN     | TIPA             | 2025-07-01      | Affirmative defense for NIST
TX     | TDPSA            | 2024-07-01      | "Texas-style" — broad scope
MT     | MCDPA            | 2024-10-01      | UOOM
FL     | FDBR             | 2024-07-01      | $1B revenue threshold
WA     | My Health My Data| 2024-03-31      | Health data only; broad
OR     | OCPA             | 2024-07-01      | UOOM
DE     | DPDPA            | 2025-01-01      | UOOM
NJ     | NJ Privacy Act   | 2025-01-15      | UOOM
NH     | NHPA             | 2025-01-01      | GDPR-flavored
KY     | KCDPA            | 2026-01-01      | GDPR-flavored
NE     | Nebraska Act     | 2025-01-01      |
RI     | RIDTPPA          | 2026-01-01      |
MN     | Minnesota Act    | 2025-07-31      |
MD     | MODPA            | 2025-10-01      | Strict — minimization heavy
```

**UOOM = Universal Opt-Out Mechanism** — like GPC, server-side honoring of browser-level opt-out signals. Multiple states recognize GPC.

**Federal preemption?** APRA (American Privacy Rights Act) was introduced in 2024 with bipartisan support but stalled. There is no comprehensive federal privacy law as of 2025.

**Compliance reality.** Most multi-state businesses adopt "comply with the strictest" — usually CCPA for sale/share, Colorado/Connecticut for UOOM and sensitive-data opt-in, Maryland for minimization. A unified privacy program meets the worst case.

**Interaction with CCPA.** Each state's law applies to its residents. A California consumer gets CCPA rights; a Virginia consumer gets VCDPA rights. Universal-opt-out signals like GPC apply across all states that recognize them.

## Engineering Checklist

A complete CCPA/CPRA engineering implementation:

**Privacy policy and notices**
```text
□ Privacy policy with all CCPA/CPRA-required disclosures (categories, purposes,
  sources, third parties, sale/share, SPI, retention, rights, agents, links,
  last-update date)
□ Annual privacy policy update (calendar a review)
□ Notice at collection deployed at every collection point
□ Just-in-time notices for new collection contexts
□ Separate Employee Privacy Notice
□ Separate B2B Privacy Notice (if applicable)
□ Children's privacy policy (if collecting from <16)
```

**Required links**
```text
□ "Do Not Sell or Share My Personal Information" link on homepage AND every
  page collecting PI
□ "Limit the Use of My Sensitive Personal Information" link (if processing SPI)
□ OR combined "Your Privacy Choices" link + opt-out icon
□ Links visible above the fold or in primary navigation, not buried in footer
□ Links accessible (WCAG-compliant)
□ Mobile-app equivalent (settings screen with same options)
```

**Opt-out infrastructure**
```text
□ GPC signal honored server-side (Sec-GPC: 1)
□ GPC honored on first request; persists for the session and across sessions
  if user is identified
□ Opt-out cookie (long-lived, secure, sameSite=Lax)
□ Opt-out flag persisted on authenticated user profile
□ Ad-tech tags conditionally loaded based on opt-out state
□ Server-side ad signal (e.g., Google Restricted Data Processing) when opt-out
□ CDP / CRM marked with "no-sale-no-share"
□ Cookie banner integrated with CCPA opt-out (does not undo GPC)
□ Re-engagement after 12 months, non-discriminatory
```

**Consumer request endpoints / UI**
```text
□ Right to Know (categories) endpoint
□ Right to Know (specific pieces) data export
□ Right to Delete endpoint with cascade
□ Right to Correct endpoint and account-edit UI
□ Right to Opt-Out (web form + GPC)
□ Right to Limit SPI endpoint
□ Authorized-agent submission process
□ Two webform options: signed-in (re-auth) and signed-out (email match + verification)
□ Email or phone for non-web submissions (if you have brick-and-mortar)
□ Response confirmation emails
□ 45-day timer with reminders
□ 90-day extension workflow with consumer notice
□ Free for first request in 12-month period; abuse detection for excessive
```

**VCR (verification)**
```text
□ Authenticated user: re-auth + 2FA
□ Non-authenticated: 2-3 data-point match
□ Specific-pieces: high-certainty (3+ match)
□ Categories: reasonable (2+ match)
□ Opt-out / Limit SPI: no verification
□ Rejection notice when verification fails
□ Don't disclose SSN, DL, financial credentials, passwords, biometric
  raw values; confirm collection only
```

**Service provider / contractor**
```text
□ Inventory of all SPs / contractors / third parties receiving PI
□ §1798.140(ag)-compliant contracts with each SP
□ §1798.140(j)-compliant contracts with each contractor
□ Sub-processor approval workflow
□ SP/contractor deletion API (push consumer-id list on deletion)
□ SP/contractor breach notification process
□ Annual SP/contractor compliance review (SOC 2 reports, attestations)
```

**Records and audit**
```text
□ Records of consumer requests retained 24 months (request, response, basis)
□ Logs of opt-out signals received (GPC, link click)
□ Logs of deletion confirmations (per system)
□ Documentation of exceptions invoked (per record retained despite delete)
□ Annual cybersecurity audit (when required by CPPA threshold)
□ Risk assessments before significant-risk processing
□ Internal review of privacy policy (annual)
```

**Data minimization and storage limitation**
```text
□ Field-by-field justification for collection (proportionality)
□ Retention schedule per data category
□ Automatic deletion / archival workflow at retention end
□ Inventory of data stored, with classification (PI / SPI / non-PI)
```

**Security**
```text
□ Encryption at rest and in transit
□ Access controls (role-based; principle of least privilege)
□ Audit logging of access to PI/SPI
□ Vulnerability management
□ Incident response plan
□ Breach-notification workflow (Cal Civ Code §1798.82 — 'most expedient time')
□ "Reasonable security" baseline (CIS, NIST, ISO 27001-aligned)
```

**Training and governance**
```text
□ Annual training for privacy / CS staff
□ Training records retained
□ Privacy team designation
□ Privacy program documentation
□ Cross-functional privacy steering committee
□ Vendor management process incorporating privacy review
□ Product launch privacy review checklist
```

**Cookie banner / ad-tech specifics**
```text
□ Banner does not block content
□ "Reject all" button (CPRA-aligned and GDPR-aligned)
□ Banner respects GPC pre-set
□ Granular cookie categories
□ Third-party-script loading conditioned on consent / opt-out
□ Server-side tracking duplicated with same opt-out logic
□ Mobile-app SDKs configured for opt-out
□ TikTok / Meta / Google integrations: Restricted Data Processing where opt-out
□ IDFA / Android Ad ID respected (independent of CCPA but adjacent)
```

**Children**
```text
□ Age screen at signup or other PI collection
□ Parental consent flow for under-13 (COPPA + CCPA)
□ Opt-in to sale/share for under-16
□ Distinct privacy notice for kid-facing products
```

## Common Gotchas — broken→fixed pairs

```text
BROKEN: "We don't sell data."
FIXED:  Even if no money changes hands, sharing for cross-context behavioral
        ads counts as "sharing" under CPRA. Audit ad-tech: Meta Pixel, Google
        Ads, TikTok Pixel are sharing unless restricted. Disclose in privacy
        policy and respect opt-out.

BROKEN: GA4 deployed under default Universal Analytics agreement, claimed as
        "service provider."
FIXED:  Sign Google Customer Data Processing Terms; enable Restricted Data
        Processing for California users; review whether GA4 still combines
        data across properties for own purposes. If yes → share, not SP.
        Disclose accordingly.

BROKEN: "Do Not Sell" link in tiny text in the footer at the bottom of every
        page, or only on the privacy-policy page.
FIXED:  CPPA regs require "clear and conspicuous." Top-nav link, primary nav
        bar, or first-screen footer with the official opt-out icon. Same
        prominence on mobile.

BROKEN: Honored cookie banner but ignored Sec-GPC because "we already give
        users a banner."
FIXED:  GPC is a separate legal signal. Detect Sec-GPC: 1 server-side; treat
        as opt-out; persist; signal to ad-tech. The banner is for cookies;
        GPC is for sale/share. Both must work.

BROKEN: Cookie consent banner styled like GDPR — reject-all defaults consent
        OFF for cookies but doesn't change the CCPA opt-out flag.
FIXED:  Map "reject all advertising cookies" + California state to "set
        ccpa_optout=1" cookie. Two distinct flags but linked UI.

BROKEN: Treating CCPA verification like GDPR — accept email-based requests
        with no verification.
FIXED:  CCPA-specific: match request data points before disclosing specific
        pieces. Email click-through alone is not enough for high-certainty
        verification. Add a 2-3 data-point match for specific-pieces and
        delete.

BROKEN: First request in 12 months charged $25 fee.
FIXED:  Free for first request. Document policy: free for first; subsequent
        within 12 months may carry "reasonable fee" only if "manifestly
        unfounded or excessive" — and reasonable fee must be cost-based.

BROKEN: Data export endpoint returns last 12 months of data only.
FIXED:  Post-CPRA, export covers all data on/after 2022-01-01 (no rolling
        cutoff). Update query window. If you weren't keeping records, retroactive
        rebuild may be needed.

BROKEN: Salesforce / Stripe / Mailchimp / hosting provider contracts predate
        CCPA, contain "vendor may use data to improve products."
FIXED:  Re-paper. Use CCPA-specific addendum or CCPA-version DPA. Confirm:
        no use beyond SP scope; no sale/share; no combination with other
        data. SOC 2 + CCPA addendum is the typical package.

BROKEN: Ad agency receives consumer PII to "build segments," uses across
        multiple advertiser accounts.
FIXED:  Cross-advertiser combination breaks SP status. Either:
        (a) Restrict per-advertiser data and re-paper as SP; or
        (b) Treat as sale/share, disclose, respect opt-out.

BROKEN: Privacy policy lists "third parties" as "marketing partners,
        analytics providers, hosting" but doesn't break out which receive
        PI as sale, share, or SP.
FIXED:  Three lists: SPs/contractors (with categories); third parties to
        whom PI sold (with categories); third parties to whom PI shared
        (with categories). Per-category-of-PI-and-recipient mapping.

BROKEN: Opt-out request through web form takes 60 days because IT backlog.
FIXED:  15-day window for opt-out. Not extendable. Workflow must be
        automated end-to-end. Web-form → opt-out flag set < 1 day, ad-tech
        tags removed within 15.

BROKEN: Authorized-agent requests rejected as "not from the consumer."
FIXED:  Documented agent process. Accept signed permission slip + verify
        consumer identity. Mine, Permission Slip, OptOutPrescreen-style
        bulk submissions handled.

BROKEN: Backups never deleted. Deletion request fulfilled in live systems
        only.
FIXED:  Document: backups not actively wiped, but on restore, deletion
        requests are re-applied. Ensure 30-90-day rotation eventually
        rolls deletes through. Disclose in policy.

BROKEN: SPI used for ad targeting (e.g., political affiliation inferences
        used for political ads).
FIXED:  Without explicit opt-in, SPI cannot be used for advertising. Build
        an SPI-classification layer; gate ad use on consumer's not-limited
        status.

BROKEN: Geolocation used for nearby-store ads — treated as not-SPI.
FIXED:  "Precise" geolocation (≤1850 ft) is SPI. Generalize to ZIP / city
        if used for ads, or treat as SPI and respect limit-SPI right.

BROKEN: Employee PI excluded from privacy policy.
FIXED:  Effective 2023, employees and applicants are in scope. Add an
        Employee Privacy Notice. Update HR systems for delete/access support.

BROKEN: Children's app collected age, kids self-attested "13," ad tags
        loaded.
FIXED:  Age gate before any PI collection. For <16, opt-in to sale/share
        required. For <13, parental consent (COPPA + CCPA). Default OFF
        for unverified ages.

BROKEN: VCR rejects email-only requests, demands government ID for any
        request.
FIXED:  Verification proportional to risk. Categories request: reasonable
        match (2 data points). Specific pieces / delete: high certainty
        (3+ match). Government ID is for high-risk only and must be
        deleted after verification.

BROKEN: Privacy policy hasn't been updated since 2020.
FIXED:  Annual update required. Calendar a review. Track regulatory
        changes (CPPA rulemaking, AG guidance).

BROKEN: Customer service responds to consumer's email "I want my data
        deleted" by ignoring it.
FIXED:  CS staff trained to recognize CCPA requests and route to the
        privacy team within 24 hours. The 45-day timer starts on receipt
        of request, not on internal triage.
```

## Practical Privacy-Choices UI

A "Your Privacy Choices" page that consolidates the CCPA/CPRA/multi-state rights:

```text
================================================================
                    Your Privacy Choices
================================================================

Use the controls below to exercise your privacy rights.

[Detected: Global Privacy Control signal — already opted out
 of sale and share.]

────────────────────────────────────────────────────────────────
Do Not Sell or Share My Personal Information
   [ X ] Opted out

   When enabled, we will not sell or share your personal
   information for cross-context behavioral advertising.

────────────────────────────────────────────────────────────────
Limit the Use of My Sensitive Personal Information
   [   ] Limited

   When enabled, we will limit our use of sensitive personal
   information (such as precise geolocation, account credentials,
   or biometric identifiers) to the purposes permitted by law.

────────────────────────────────────────────────────────────────
Manage Cookie Preferences
   [   ] Strictly necessary (required)
   [ X ] Functional
   [   ] Analytics
   [   ] Advertising

────────────────────────────────────────────────────────────────
Submit a Privacy Request

   What do you want to do?
     ( ) Know what data you have about me
     ( ) Get a copy of my data
     ( ) Delete my data
     ( ) Correct my data

   Your email: [_________________]
   Verification code sent to email.
   [Submit Request]

────────────────────────────────────────────────────────────────
Designate an Authorized Agent

   You may designate an agent to submit requests on your behalf.
   Provide a signed authorization. [Learn more]

────────────────────────────────────────────────────────────────
View My Submitted Requests
   - 2026-04-12  Right to Know — completed (download)
   - 2025-11-30  Opt-out — honored

────────────────────────────────────────────────────────────────
Effective: 2026-01-01
[Privacy Policy] [Notice at Collection] [Contact]
================================================================
```

**Mobile-app equivalent.** Settings screen with the same options. Native-app cookie equivalents (advertising IDs) respected.

**Co-existence with EU consent banner.** Detect jurisdiction; show EU consent banner for EU users (opt-in cookies), CCPA controls for California users (opt-out + GPC). Common middle ground: a single "Your Privacy Choices" page that adjusts based on user location and signal.

**API for automation.** Expose a small API for authorized agents to submit requests in bulk:

```text
POST /api/privacy/agent-request
  Headers:
    Authorization: Bearer <agent-api-key>
  Body:
    {
      "agent_name": "Permission Slip",
      "consumer_email": "user@example.com",
      "consumer_signature": "<base64 PNG>",
      "request_type": "delete" | "know" | "opt_out" | "correct" | "limit_spi",
      "evidence_url": "https://agent.example/perm-12345",
      "additional_data": { ... }
    }

  Response:
    202 Accepted
    {
      "request_id": "req_abc123",
      "deadline": "2026-06-09",
      "verification_status": "pending"
    }
```

## Idioms

**Treat opt-out as the default for advertising.** The simplest defense against CCPA enforcement is to default California users to no-sale-no-share, then upsell consent if your model needs it. Many businesses skip consent and just live with non-personalized ads in California.

**Honor GPC + cookie banner together.** Don't make them fight. The cookie banner addresses GDPR consent and granular cookies; the GPC signal addresses CCPA opt-out. Both should reach the same end state for California users.

**Treat new state laws as "comply with strictest" baseline.** Trying to tune per-state is a bug factory. Adopt: GPC always honored; sale/share opt-out always available; sensitive-data opt-in by default in non-CA states; correct/delete/access workflows uniform.

**Minimize PI collection upfront.** §1798.100(c) makes data minimization an obligation. Before adding a field to a form, ask "what's the operational benefit?" Drop fields that fail.

**Do Not Sell + Limit SPI links must be prominent.** Conservative reading: top navigation OR a footer link that's visible above the fold of the homepage, with the opt-out icon. Mobile shouldn't bury them in a hamburger menu.

**Document the basis for retention.** When invoking one of the §1798.105(d) exceptions, log the exception per record with the basis. When auditors or the CPPA ask, you have evidence — not a "we just kept it" answer.

**Run a privacy-as-code pipeline.** Privacy decisions encoded as machine-readable rules: data classification, retention period, processing purpose, consent state. Use this to gate access to PI in production systems.

**Privacy review at product launch.** Add a privacy checklist to the release checklist. New collection? New retention? New SP? New ad-tech tag? New SPI category? Each requires policy/notice/contract updates.

**Train front-line staff.** A consumer who emails support saying "delete my data" must be routed to the privacy team within hours, not weeks. The 45-day clock is unforgiving.

**Tag all PI fields at the schema level.** A column-level tag (`pii=true`, `spi=true`, `retention=180d`) lets automated tooling enforce policy: deletion routines, exports, audit reports.

**Keep service-provider contracts current.** When the CPPA issues a regulatory update, every SP contract may need amendment. Keep an SP inventory with last-papered date.

## See Also

- compliance/gdpr — EU data-protection regime; GDPR comparison; cross-jurisdictional privacy programs
- compliance/sox — Sarbanes-Oxley financial controls
- compliance/pci-dss — payment-card-industry data security standard
- compliance/hipaa — US health privacy law (CCPA carve-out)
- legal/license-decoder — software license interpretation
- legal/cla-vs-dco — contributor agreements vs developer certificate of origin
- legal/spdx-identifiers — SPDX license expressions
- security/owasp-top-10 — application security baseline
- security/threat-modeling — STRIDE and risk modeling
- security/incident-response — incident handling and breach notification

## References

- Cal Civ Code §1798.100 et seq. — California Consumer Privacy Act of 2018, as amended by CPRA
- Cal Civ Code §1798.100 — General duties of business; notice at collection; minimization
- Cal Civ Code §1798.105 — Right to delete; exceptions
- Cal Civ Code §1798.106 — Right to correct (CPRA)
- Cal Civ Code §1798.110 — Right to know (categories)
- Cal Civ Code §1798.115 — Right to know (third parties)
- Cal Civ Code §1798.120 — Right to opt-out of sale or sharing
- Cal Civ Code §1798.121 — Right to limit use of sensitive personal information (CPRA)
- Cal Civ Code §1798.125 — Right to non-discrimination
- Cal Civ Code §1798.130 — Notices, response timelines, methods
- Cal Civ Code §1798.135 — Methods of opt-out; required links
- Cal Civ Code §1798.140 — Definitions (including (v) PI, (ae) SPI, (ad) sale, (ah) share, (ag) service provider, (j) contractor)
- Cal Civ Code §1798.145 — Exemptions (including HIPAA, GLBA carve-outs)
- Cal Civ Code §1798.150 — Private right of action for data breaches
- Cal Civ Code §1798.155 — Civil penalties
- Cal Civ Code §1798.185 — Regulations issued by CPPA (cybersecurity audits, risk assessments, ADM)
- Cal Civ Code §1798.199.10 et seq. — California Privacy Protection Agency
- Cal Civ Code §1798.82 — California breach notification law
- Cal Civ Code §1798.81.5 — Reasonable security standard
- Cal Penal Code §1546 et seq. — California Electronic Communications Privacy Act (CalECPA)
- Cal Code Regs Tit. 11, §7000 et seq. — CCPA Regulations issued by CPPA (current consolidated version)
- Cal Code Regs Tit. 11, §7011 — Privacy policy contents
- Cal Code Regs Tit. 11, §7024 — Specific pieces of PI; non-disclosable categories
- Cal Code Regs Tit. 11, §7025 — Opt-out preference signals (GPC)
- Cal Code Regs Tit. 11, §7050 — Service provider and contractor obligations
- Cal Code Regs Tit. 11, §7060 et seq. — Verification of consumer requests
- Cal Code Regs Tit. 11, §7063 — Authorized agents
- Cal Rev & Tax Code §23101 — "Doing business in California" definition (FTB)
- Cal Code Regs Tit. 18, §17014 — California residency definition
- Cal Probate Code §4000 et seq. — Power of Attorney
- 18 USC §1681 et seq. — Fair Credit Reporting Act (FCRA — federal)
- 15 USC §6801 et seq. — Gramm-Leach-Bliley Act (GLBA — federal)
- 42 USC §1320d et seq. — HIPAA (federal)
- 15 USC §6501 et seq. — Children's Online Privacy Protection Act (COPPA — federal)
- 20 USC §1232g — Family Educational Rights and Privacy Act (FERPA — federal)
- oag.ca.gov/privacy/ccpa — California Attorney General CCPA resource page
- cppa.ca.gov — California Privacy Protection Agency
- cppa.ca.gov/regulations — CPPA finalized and proposed regulations
- globalprivacycontrol.org — Global Privacy Control specification and adoption
- iapp.org/resources/article/ccpa — IAPP CCPA reference materials
- iapp.org/resources/article/us-state-privacy-legislation-tracker — multi-state tracker
- nist.gov/privacy-framework — NIST Privacy Framework (privacy controls reference)
- People v. Sephora USA, Inc., AG Settlement (Aug 2022) — first major CCPA enforcement
- People v. DoorDash, Inc., AG Settlement (Feb 2024) — sale/share + GPC
- Tilting Point Media (CPPA, June 2024) — children's data and SPI
- Honda Motor Co., AG/CPPA Settlement (Mar 2025) — verification and agents
- Sephora et al. v. State of California, model SP-contract template — ccpa-template.csv (CPPA-published)
- AB 375 (2018) — original CCPA bill
- Proposition 24 (2020) — CPRA ballot measure
