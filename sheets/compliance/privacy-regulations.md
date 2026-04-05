# Privacy Regulations (GDPR, CCPA/CPRA, Global Privacy Laws)

Practical reference for privacy principles, major privacy regulations, data subject rights, cross-border data transfers, and privacy program implementation.

## Privacy Principles

### OECD Privacy Principles (1980, updated 2013)

```text
1. Collection Limitation    -- collect only with consent or legal authority, minimize
2. Data Quality             -- accurate, complete, up-to-date for purpose
3. Purpose Specification    -- state purpose at collection time, limit subsequent use
4. Use Limitation           -- do not use/disclose beyond stated purpose without consent
5. Security Safeguards      -- protect against loss, unauthorized access, destruction
6. Openness                 -- be transparent about practices, policies, data controller
7. Individual Participation -- right to access, challenge, and correct personal data
8. Accountability           -- data controller responsible for compliance with principles
```

### Fair Information Practice Principles (FIPPs)

```text
FIPPs (US Framework):
  Notice/Awareness         -- inform individuals about data practices before collection
  Choice/Consent           -- give options about how data is used beyond primary purpose
  Access/Participation     -- allow individuals to view and correct their data
  Integrity/Security       -- ensure data accuracy and protect against misuse
  Enforcement/Redress      -- mechanisms to enforce principles and remedy violations
```

## GDPR (EU General Data Protection Regulation)

### Scope and Applicability

```text
Territorial Scope (Article 3):
  Applies to:
    - Organizations established in the EU (regardless of where processing occurs)
    - Organizations outside EU offering goods/services to EU residents
    - Organizations outside EU monitoring behavior of EU residents

  Does NOT apply to:
    - Purely personal or household activities
    - National security activities
    - Law enforcement (separate LED directive)

Material Scope:
  Applies to all processing of personal data, wholly or partly by automated means,
  or non-automated processing of data forming part of a filing system.

Personal Data (Article 4(1)):
  Any information relating to an identified or identifiable natural person.
  Includes: name, ID number, location, online identifier (IP, cookie),
  physical, physiological, genetic, mental, economic, cultural, social identity.

Special Categories (Article 9) -- stricter rules:
  Racial/ethnic origin, political opinions, religious/philosophical beliefs,
  trade union membership, genetic data, biometric data (for identification),
  health data, sex life/sexual orientation.
```

### Lawful Basis for Processing (Article 6)

```text
Six Lawful Bases (must establish BEFORE processing):

  (a) Consent           -- freely given, specific, informed, unambiguous
                           Must be as easy to withdraw as to give
                           Cannot be a precondition for service (if not necessary)
                           Children under 16 require parental consent (member states: 13-16)

  (b) Contract           -- necessary for performance of contract with data subject
                           Example: processing address to deliver purchased goods

  (c) Legal obligation   -- required by EU or member state law
                           Example: tax reporting, anti-money laundering

  (d) Vital interests    -- protect life of data subject or another person
                           Example: emergency medical treatment, disaster response

  (e) Public interest    -- necessary for task in public interest or official authority
                           Example: government services, public health measures

  (f) Legitimate interest -- controller or third party interest, balanced against
                            data subject rights (requires balancing test)
                            Example: fraud prevention, direct marketing, IT security
                            NOT available to public authorities for their core tasks
```

### Data Subject Rights

```text
Right                    Article   Response Time   Details
Access                   15        30 days         Copy of data + processing info
Rectification            16        30 days         Correct inaccurate data
Erasure ("right to       17        30 days         Delete when no longer necessary,
  be forgotten")                                    consent withdrawn, or unlawful
Restriction              18        30 days         Mark data, limit processing
Data portability         20        30 days         Machine-readable format (CSV, JSON)
Object                   21        Immediately     Must stop unless compelling grounds
Automated decision-      22        30 days         Right not to be subject to solely
  making/profiling                                  automated decisions with legal effect

Extensions: +2 months for complex requests (must notify within 30 days)
Fees: Free (can charge reasonable fee for manifestly excessive requests)
Identity: Must verify identity before fulfilling requests
```

### Data Protection Officer (DPO)

```text
DPO Required When (Article 37):
  - Public authority or body (except courts)
  - Core activities require regular, systematic, large-scale monitoring
  - Core activities involve large-scale processing of special categories / criminal data

DPO Requirements:
  - Expert knowledge of data protection law and practices
  - Independent (no conflicts of interest, cannot be instructed on tasks)
  - Reports to highest management level
  - Adequate resources provided
  - Contact details published and communicated to supervisory authority
  - Can be internal employee or external contractor
  - Can serve multiple entities (if accessible to each)
```

### Data Protection Impact Assessment (DPIA)

```text
DPIA Required When (Article 35):
  - Systematic and extensive profiling with significant effects
  - Large-scale processing of special categories
  - Systematic monitoring of publicly accessible area
  - New technologies with likely high risk
  - Supervisory authority published list (check local DPA)

DPIA Contents:
  1. Systematic description of processing operations and purposes
  2. Assessment of necessity and proportionality
  3. Assessment of risks to rights and freedoms
  4. Measures to address risks (safeguards, security, mechanisms)

DPIA Process:
  1. Identify need for DPIA (screening checklist)
  2. Describe the processing (data flows, systems, recipients)
  3. Assess necessity and proportionality
  4. Identify and assess risks
  5. Identify measures to mitigate risks
  6. Sign off and record outcomes
  7. Integrate outcomes into processing design
  8. Consult supervisory authority if high residual risk (Article 36)
```

### Cross-Border Data Transfers

```text
Transfer Mechanisms (Chapter V):

  Adequacy Decisions (Article 45):
    Countries deemed adequate by EU Commission:
    Andorra, Argentina, Canada (commercial), Faroe Islands, Guernsey,
    Israel, Isle of Man, Japan, Jersey, New Zealand, Republic of Korea,
    Switzerland, UK, Uruguay, EU-US Data Privacy Framework (DPF)

  Standard Contractual Clauses (SCCs) (Article 46(2)(c)):
    - New SCCs adopted June 2021 (modular approach: 4 modules)
    - Module 1: Controller to Controller
    - Module 2: Controller to Processor
    - Module 3: Processor to Processor
    - Module 4: Processor to Controller
    - Must conduct Transfer Impact Assessment (TIA)

  Binding Corporate Rules (BCRs) (Article 47):
    - For intra-group transfers (multinational corporations)
    - Requires approval from lead supervisory authority
    - Comprehensive data protection program
    - 12-18 month approval process typically

  Derogations (Article 49) -- limited use:
    - Explicit consent (informed of risks)
    - Contract performance
    - Public interest
    - Legal claims
    - Vital interests
    - Public register
```

### Breach Notification

```text
Supervisory Authority Notification (Article 33):
  Timing:     Within 72 hours of becoming aware
  Exception:  Not required if unlikely to result in risk to individuals
  Contents:   Nature of breach, categories/numbers affected, DPO contact,
              likely consequences, measures taken/proposed

Data Subject Notification (Article 34):
  Timing:     Without undue delay
  Required:   When breach likely results in HIGH risk to rights and freedoms
  Exception:  Not required if data encrypted, or measures taken to ensure
              high risk no longer likely, or disproportionate effort (public notice)
  Contents:   Clear and plain language description, DPO contact,
              likely consequences, measures taken/proposed

Internal Documentation:
  ALL breaches must be documented (Article 33(5))
  Include: facts, effects, remedial action taken
```

### Fines

```text
Tier 1 (Article 83(4)):
  Up to EUR 10 million or 2% of global annual turnover (whichever is higher)
  For: controller/processor obligations, certification body, monitoring body

Tier 2 (Article 83(5)):
  Up to EUR 20 million or 4% of global annual turnover (whichever is higher)
  For: processing principles, lawful basis, consent, data subject rights,
       cross-border transfers

Factors Considered:
  - Nature, gravity, duration of infringement
  - Intentional or negligent character
  - Actions taken to mitigate damage
  - Degree of cooperation with supervisory authority
  - Categories of personal data affected
  - Previous infringements
  - Degree of damage suffered by data subjects
```

## CCPA / CPRA (California)

### Consumer Rights

```text
CCPA/CPRA Consumer Rights:
  Right to Know          -- what personal info is collected, used, shared, sold
  Right to Delete        -- request deletion of personal information
  Right to Opt-Out       -- opt out of sale/sharing of personal information
  Right to Non-Discrimination -- cannot penalize for exercising rights
  Right to Correct       -- correct inaccurate personal information (CPRA)
  Right to Limit Use     -- limit use of sensitive personal information (CPRA)
  Right to Data Portability -- receive data in portable format (CPRA)

Applicability (for-profit businesses meeting ANY):
  - Annual gross revenue > $25 million
  - Buy/sell/share personal info of 100,000+ consumers/households
  - Derive 50%+ of annual revenue from selling/sharing personal info

Sensitive Personal Information (CPRA):
  SSN, driver's license, passport, financial account, precise geolocation,
  racial/ethnic origin, religious beliefs, union membership, mail/email/text
  content (not to business), genetic data, biometric data, health data,
  sex life/sexual orientation, citizenship/immigration status
```

### Compliance Requirements

```text
Key Obligations:
  Privacy Notice         -- disclose categories collected, purposes, rights, retention
  "Do Not Sell/Share"    -- prominent link on homepage
  Opt-out mechanism      -- honor Global Privacy Control (GPC) signal
  Service providers      -- contractual restrictions on data use
  Data minimization      -- collect only what is reasonably necessary (CPRA)
  Storage limitation     -- retain only as long as reasonably necessary (CPRA)
  Risk assessments       -- for high-risk processing (CPRA)
  Annual cybersecurity audits -- for high-risk businesses (CPRA)

Response Timelines:
  Acknowledge:    10 business days
  Respond:        45 calendar days (+ 45 day extension if notified)
  Verification:   Must verify identity before responding

Enforcement:
  California AG:          $2,500 per violation, $7,500 per intentional violation
  CPPA (Privacy Agency):  Administrative enforcement (CPRA)
  Private right of action: Data breaches only (statutory damages $100-$750 per consumer)
```

## Other Global Privacy Regulations

### Comparison Matrix

```text
Regulation    Jurisdiction  Effective  Key Feature              Fines
GDPR          EU/EEA        2018       Gold standard, global    4% revenue / EUR 20M
CCPA/CPRA     California    2020/2023  Consumer opt-out focus   $7,500/violation
PIPEDA        Canada        2000       Consent-based, PIPEDA    CAD 100K
LGPD          Brazil        2020       GDPR-inspired            2% revenue / BRL 50M
POPIA         South Africa  2021       8 conditions for process ZAR 10M
APPI          Japan         2003/2022  Adequate per EU          JPY 100M (corporate)
PDPA          Thailand      2022       GDPR-inspired            THB 5M
PDPB          India         2023       DPDP Act, consent-based  INR 250 crore
PIPL          China         2021       State interest focus      5% revenue / CNY 50M
FADP          Switzerland   2023       Revised, GDPR-aligned    CHF 250K (individual)

Common Themes Across Regulations:
  - Purpose limitation and data minimization
  - Individual rights (access, deletion, correction, portability)
  - Breach notification requirements
  - Cross-border transfer restrictions
  - Accountability and documentation obligations
  - DPO or equivalent privacy officer requirement
  - Impact assessments for high-risk processing
```

## Privacy by Design

### Seven Foundational Principles (Cavoukian)

```text
1. Proactive not Reactive      -- anticipate and prevent privacy issues
2. Privacy as Default Setting  -- no action required from individual to protect privacy
3. Privacy Embedded in Design  -- integral part of system, not bolt-on
4. Full Functionality          -- positive-sum not zero-sum (privacy AND security)
5. End-to-End Security         -- lifecycle protection from collection to deletion
6. Visibility and Transparency -- operations verifiable by stakeholders
7. Respect for User Privacy    -- user-centric, empowering individuals

Implementation Checklist:
  [ ] Data minimization in schema design (collect only what is needed)
  [ ] Purpose limitation enforced technically (access controls by purpose)
  [ ] Retention schedules automated (auto-delete after period)
  [ ] Encryption at rest and in transit (TLS 1.3, AES-256)
  [ ] Pseudonymization where possible (separate identifiers from data)
  [ ] Anonymization for analytics (k-anonymity, differential privacy)
  [ ] Access logging and audit trails
  [ ] Consent management integrated into user flows
  [ ] DPIA conducted before launch
  [ ] Privacy notice drafted and reviewed by legal
```

## Privacy Impact Assessment

### PIA Process

```text
Step 1: Threshold Analysis
  - Does the project involve personal data?
  - Is it a new collection, use, or disclosure?
  - Does it change an existing system significantly?
  - If yes to any: proceed to full PIA

Step 2: Data Flow Mapping
  - What data is collected? (categories, sensitivity)
  - From whom? (data subjects, sources)
  - How is it collected? (forms, APIs, automated, third-party)
  - Where is it stored? (databases, cloud, jurisdiction)
  - Who has access? (roles, third parties, processors)
  - How long is it retained? (retention schedule)
  - How is it disposed of? (deletion, anonymization)

Step 3: Privacy Risk Analysis
  - Risk = Likelihood x Impact
  - Consider: unauthorized access, misuse, excessive collection,
    inadequate consent, lack of transparency, re-identification,
    cross-border transfers, data breach, function creep

Step 4: Mitigation Controls
  - Technical: encryption, access controls, pseudonymization
  - Organizational: policies, training, DPO oversight
  - Legal: contracts, DPAs, consent mechanisms, privacy notices

Step 5: Documentation and Approval
  - Document findings and decisions
  - Obtain stakeholder sign-off (DPO, legal, CISO, business owner)
  - Publish summary (where required)
  - Schedule review (annual or upon material change)
```

## Cross-Border Data Transfers

### Transfer Impact Assessment (TIA)

```text
TIA Steps (required for SCCs post-Schrems II):
  1. Map the transfer (what data, where, to whom, why)
  2. Identify the transfer mechanism (SCCs, BCRs, adequacy)
  3. Assess laws of recipient country:
     - Government access powers (surveillance laws)
     - Rule of law and judicial independence
     - Data protection framework effectiveness
     - International commitments
  4. Assess supplementary measures needed:
     - Technical: end-to-end encryption, pseudonymization, split processing
     - Contractual: enhanced audit rights, transparency commitments
     - Organizational: strict access controls, security certifications
  5. Document assessment and decision
  6. Monitor for changes in recipient country laws
```

### Cookie Consent

```text
Cookie Consent Requirements:

EU (ePrivacy Directive + GDPR):
  - Prior consent required for non-essential cookies
  - Essential cookies exempt (session, security, load balancing)
  - Consent must be freely given, specific, informed
  - Must be as easy to reject as to accept
  - No cookie walls (cannot block access for refusing cookies)
  - Consent valid for 6-12 months (varies by DPA)
  - Record of consent required

Cookie Categories:
  Strictly Necessary    -- exempt from consent (session, auth, security)
  Performance/Analytics -- require consent (Google Analytics, heatmaps)
  Functional            -- require consent (language preference, chat widget)
  Targeting/Advertising -- require consent (ad tracking, retargeting)

Implementation:
  - Cookie banner with granular opt-in (not pre-ticked boxes)
  - Consent Management Platform (CMP): OneTrust, Cookiebot, TrustArc
  - IAB TCF v2.2 for programmatic advertising consent
  - Respect Global Privacy Control (GPC) and Do Not Track (DNT)
```

## Privacy Notices

### Privacy Notice Requirements

```text
GDPR Privacy Notice (Articles 13-14):
  Required Information:
    - Identity and contact details of controller (and DPO)
    - Purposes and lawful basis for each processing activity
    - Categories of personal data (if not collected directly)
    - Recipients or categories of recipients
    - Cross-border transfer details and safeguards
    - Retention periods (or criteria to determine)
    - Data subject rights and how to exercise them
    - Right to lodge complaint with supervisory authority
    - Whether provision is statutory/contractual/obligatory
    - Existence of automated decision-making (logic, significance, consequences)
    - Source of data (if not collected from data subject)

Presentation Requirements:
  - Concise, transparent, intelligible, easily accessible
  - Clear and plain language (especially for children)
  - Written or electronic form
  - Layered approach recommended (summary + full notice)
  - Just-in-time notices for specific processing

Timing:
  Direct collection (Art 13):   At the time of collection
  Indirect collection (Art 14): Within reasonable period (max 1 month)
```

## See Also

- Security Awareness
- Supply Chain Security
- AI Governance
- AI Ethics

## References

- GDPR Full Text: https://gdpr-info.eu
- CCPA/CPRA: California Civil Code 1798.100-1798.199.100
- OECD Privacy Guidelines (2013 revision)
- EDPB Guidelines: https://edpb.europa.eu/our-work-tools/general-guidance/guidelines-recommendations-best-practices_en
- NIST Privacy Framework: https://www.nist.gov/privacy-framework
- Cavoukian, A. (2009): Privacy by Design: The 7 Foundational Principles
- IAPP (International Association of Privacy Professionals): https://iapp.org
