# GDPR — General Data Protection Regulation

EU Regulation 2016/679, in force 25 May 2018. Engineering-focused field manual: lawful basis, data-subject rights, breach response, privacy by design, transfer mechanisms, and the code that implements all of it.

## Setup

Regulation (EU) 2016/679 of the European Parliament and of the Council of 27 April 2016 on the protection of natural persons with regard to the processing of personal data and on the free movement of such data. Repealed Directive 95/46/EC. Entered into force 24 May 2016, applied from 25 May 2018. Directly applicable in all 27 EU member states (no national transposition needed) plus EEA members Iceland, Liechtenstein, Norway. UK GDPR (post-Brexit) is a near-identical parallel regime under the Data Protection Act 2018.

| Item | Value |
|------|-------|
| Citation | Regulation (EU) 2016/679 |
| In force | 25 May 2018 |
| Articles | 99 (substantive: Art. 1–95) |
| Recitals | 173 (interpretive guidance, non-binding but persuasive) |
| Pages (OJ) | ~88 in OJ L 119, 4.5.2016 |
| Languages | All 24 EU official languages, equally authentic |
| Implementing Act | Member-state derogations (UK DPA 2018, German BDSG, French Loi Informatique) |
| Predecessor | Directive 95/46/EC (Data Protection Directive) |

Territorial scope (Art. 3) — applies to:

| Trigger | Coverage |
|---------|----------|
| Art. 3(1) Establishment | Processing in context of an EU establishment, regardless of where processing actually occurs |
| Art. 3(2)(a) Targeting | Offering goods/services to data subjects in the Union (paid or free) |
| Art. 3(2)(b) Monitoring | Monitoring behaviour of data subjects in the Union |
| Art. 3(3) Public international law | EU member-state law applies via public international law (e.g. embassies) |

A US SaaS with no EU office still falls under GDPR if it sells to EU customers, ships free apps in EU app stores, or runs analytics on EU visitors. Recital 23 lists indicators of "targeting": EU language localization, EU currencies, EU domain names (.de, .fr), references to EU customers.

| Penalty Tier | Max Fine | Triggers |
|--------------|----------|----------|
| Lower (Art. 83(4)) | €10M or 2% global annual turnover (higher) | Records of processing, security, breach notification, DPO, certification, processor obligations |
| Upper (Art. 83(5)) | €20M or 4% global annual turnover (higher) | Lawful basis, consent, data-subject rights, transfers, principles |

Notable enforcement: Meta €1.2B (May 2023, Schrems II transfers); Amazon €746M (Jul 2021, Luxembourg CNPD); WhatsApp €225M (2021); Google €50M (2019, CNIL). EDPB statistics: cumulative fines >€4.5 billion as of 2024.

| Authority | Country | Common abbrev. |
|-----------|---------|----------------|
| CNIL | France | Commission Nationale de l'Informatique et des Libertés |
| ICO | UK | Information Commissioner's Office (UK GDPR) |
| Garante | Italy | Garante per la protezione dei dati personali |
| AEPD | Spain | Agencia Española de Protección de Datos |
| BfDI | Germany (federal) | Bundesbeauftragte für Datenschutz |
| DPC | Ireland | Data Protection Commission (lead supervisor for many big tech) |
| Datatilsynet | Denmark / Norway | Each country has own |
| AP | Netherlands | Autoriteit Persoonsgegevens |

EDPB (European Data Protection Board, Art. 68) — composed of the heads of national supervisory authorities + EDPS. Replaces the Article 29 Working Party (WP29). Issues binding decisions in cross-border cases (Art. 65), guidelines, opinions. EDPB Guidelines are the authoritative interpretation engineers should consult; check edpb.europa.eu/our-work-tools/general-guidance.

```bash
# One-stop-shop principle (Art. 56) — controllers with EU-wide presence
# deal with a single "lead supervisory authority" determined by location
# of main establishment. Most US tech firms route to Irish DPC.

# Read the regulation:
curl -s "https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX:32016R0679"

# Track enforcement:
# https://www.enforcementtracker.com/   community-maintained DB
# https://gdprhub.eu/                    case law wiki
```

| Top-25 fines | Year | Authority | Violation |
|--------------|------|-----------|-----------|
| Meta €1.2B | 2023 | DPC IE | Schrems II transfers |
| Amazon €746M | 2021 | CNPD LU | Behavioural ads |
| Meta €405M | 2022 | DPC IE | Children's data Instagram |
| Meta €390M | 2023 | DPC IE | Contract basis for ads |
| TikTok €345M | 2023 | DPC IE | Children's accounts default-public |
| Meta €265M | 2022 | DPC IE | Scraping breach 533M users |
| Meta €251M | 2024 | DPC IE | 2018 token breach |
| WhatsApp €225M | 2021 | DPC IE | Transparency |
| Google LLC €90M | 2021 | CNIL FR | Cookie reject |
| Facebook €60M | 2021 | CNIL FR | Cookie reject |
| Google Ireland €60M | 2021 | CNIL FR | Cookie reject |
| Google €50M | 2019 | CNIL FR | Consent for ads |
| H&M €35M | 2020 | HmbBfDI DE | Employee surveillance |
| TIM €27.8M | 2020 | Garante IT | Marketing |
| Enel €26.5M | 2022 | Garante IT | Marketing |
| British Airways £20M | 2020 | ICO UK | 400k card breach |
| Marriott £18.4M | 2020 | ICO UK | 339M guest breach |
| Wind Tre €17M | 2020 | Garante IT | Marketing consent |
| Vodafone Italia €12.3M | 2020 | Garante IT | Marketing |
| Notebooksbilliger €10.4M | 2021 | LfD Niedersachsen | CCTV employees |
| Eir €8.5M | 2024 | DPC IE | Breach + transparency |
| Yahoo £7M | 2018 | ICO UK | 2014 breach (DPA98) |

| Recital | Topic | Why engineers care |
|---------|-------|---------------------|
| 23 | Targeting EU subjects | Tells you when GDPR applies to non-EU sites |
| 26 | Anonymous data | Defines the boundary of the regulation |
| 30 | Online identifiers | IP, cookie, RFID = personal data |
| 32 | Consent must be active | Pre-tick is invalid |
| 39 | Transparency | Layered notice idea grounded here |
| 43 | Power imbalance | Employer-employee consent rarely valid |
| 47 | Legitimate interests | Lists examples; reasonable expectations |
| 49 | Network security | Security telemetry is an LI |
| 50 | Compatibility test | Reuse of data for new purpose |
| 51 | Special-category data | Why Art. 9 exists |
| 64 | Identity verification | Don't over-collect to verify a DSAR |
| 65 | Right to be forgotten | Engineering scope of erasure |
| 71 | Automated decisions | Rights re. profiling |
| 75 | Risks to rights | DPIA framing |
| 78 | By design | Privacy engineering imperative |
| 87 | 72-hour breach notification | Why the clock starts at awareness |
| 116 | Transfers | Why Schrems matters |

## Key Concepts

Personal data (Art. 4(1)) — "any information relating to an identified or identifiable natural person". Identifiable means directly (name, ID number) or indirectly (location, online identifier, factors specific to physical/physiological/genetic/mental/economic/cultural/social identity). The bar is low. CJEU has confirmed: dynamic IPs (C-582/14 Breyer), license plates, employee IDs, hashed identifiers if re-identifiable, even pseudonymous device IDs.

| Examples that ARE personal data | Examples that ARE NOT |
|---------------------------------|------------------------|
| Name, email, phone, address | Truly anonymous statistics |
| IPv4 + IPv6 (Recital 30) | Aggregated counts (no re-identification possible) |
| Device IDs, IMEI, MAC, advertising IDs | Random server-internal IDs not linked to humans |
| Cookies, fingerprints, session tokens | Synthetic data with no link to real people |
| Location coordinates | Weather data, generic logs without identifiers |
| Pseudonymous user IDs | Information about deceased persons (member-state law may still protect) |
| Photo / voice / handwriting | Information about legal persons (companies) — outside GDPR |
| Behavioural profile, scoring | |
| Genetic & biometric data (special) | |

| Special Categories (Art. 9) | Triggers Art. 9 lawful basis |
|-----------------------------|------------------------------|
| Racial / ethnic origin | Yes |
| Political opinions | Yes |
| Religious / philosophical beliefs | Yes |
| Trade-union membership | Yes |
| Genetic data (Art. 4(13)) | Yes |
| Biometric data for unique identification (Art. 4(14)) | Yes |
| Health data (Art. 4(15)) | Yes |
| Sex life or sexual orientation | Yes |
| Criminal convictions (Art. 10) | Treated separately, only authorised under EU/MS law |

Processing (Art. 4(2)) — "any operation or set of operations": collection, recording, organisation, structuring, storage, adaptation, alteration, retrieval, consultation, use, disclosure by transmission, dissemination, alignment or combination, restriction, erasure or destruction. Reading a row from a database is processing. Logging an HTTP request is processing.

| Role | Defined | Liability | Engineering test |
|------|---------|-----------|------------------|
| Controller (Art. 4(7)) | Determines purposes and means | Primary | "Why is this processing happening, and does my org decide?" |
| Joint controllers (Art. 26) | Jointly determine purposes/means | Joint and several | Two parties share decision-making (e.g. Meta-Like-Button case C-40/17) |
| Processor (Art. 4(8)) | Processes on behalf of controller | Limited (Art. 28, Art. 32, breach notify, sub-processors) | "We do what they tell us" |
| Sub-processor | Processor's processor | Through DPA chain | AWS for your SaaS, your SaaS for the merchant |
| Recipient (Art. 4(9)) | Receives personal data | None per se | Just disclosed-to |
| Third party (Art. 4(10)) | Anyone other than data subject, controller, processor, employees | Becomes controller/processor when they process | |

Pseudonymization (Art. 4(5)) — "processing of personal data in such a manner that the personal data can no longer be attributed to a specific data subject without the use of additional information, provided that such additional information is kept separately and is subject to technical and organisational measures". Still personal data. Reduces risk; does not exempt from GDPR. Engineering: store the mapping (user_id to real PII) in a separate, encrypted, access-controlled keystore.

```python
# Pseudonymization: reversible, still personal data
import secrets, hmac, hashlib, os

PEPPER = os.environ["PSEUDO_PEPPER"]  # in HSM/KMS, not in repo

def pseudonymize(email: str) -> str:
    return hmac.new(PEPPER.encode(), email.lower().encode(),
                    hashlib.sha256).hexdigest()

# Anonymization: irreversible, NOT personal data (if done right)
# Must survive: singling out, linkability, inference attacks (WP29 WP216)
def anonymize_age(age: int) -> str:
    if age < 18: return "<18"
    if age < 30: return "18-29"
    if age < 50: return "30-49"
    if age < 65: return "50-64"
    return "65+"   # 5 buckets, no re-id from age alone
```

Anonymization — true anonymization removes the data from GDPR scope. Must be irreversible *and* prevent singling out, linkability, and inference (WP29 Opinion 05/2014 WP216). Most "anonymization" in practice is actually pseudonymization. Test: can a determined attacker with reasonable means re-identify? If yes, still personal data.

| Identifier type | Re-id risk | Treatment |
|-----------------|------------|-----------|
| Direct (name, email, phone, NID) | Total | Tokenise / pseudonymise / encrypt |
| Quasi (DOB, ZIP, gender) | High when combined | k-anonymity ≥ 5 to release |
| Behavioural (browsing, location trail) | Very high (Sweeney 87% by ZIP+DOB+sex) | Aggregate or noise |
| Network (IP, MAC, ad-id) | High | Anonymise on ingestion |
| Inference (model output) | Variable | Treat as personal data if linked back |

| GDPR scope decision tree | Result |
|---------------------------|--------|
| Truly anonymous? | Out of GDPR; can be processed without restriction |
| Pseudonymous? | In GDPR; reduced risk but full obligations |
| Aggregated only (counts, sums)? | Generally out, but watch small cells |
| Synthetic data? | Out if no link, but check generator privacy |
| Encrypted but you hold the key? | In GDPR (you can re-identify) |
| Encrypted and you do NOT hold the key? | Likely still personal data for the controller; out for the cloud holding ciphertext only (per EDPB) |

## Lawful Basis (Art. 6)

Every processing operation needs at least one of six lawful bases under Art. 6(1). Pick BEFORE processing, document in the Records of Processing (Art. 30), and tell the data subject in the privacy notice (Art. 13/14). You cannot retroactively invent a basis, and you cannot switch bases mid-processing without re-justification.

| Basis | Art. | Common use | Withdrawable? | Triggers DSR |
|-------|------|------------|---------------|--------------|
| Consent | 6(1)(a) | Marketing, optional cookies, profiling | Yes — must be as easy as giving | All |
| Contract | 6(1)(b) | Account creation, order fulfilment, payment | No (but contract terminable) | All except erasure if contract live |
| Legal obligation | 6(1)(c) | Tax records, AML, court orders | No | Erasure usually blocked |
| Vital interests | 6(1)(d) | Saving lives (ICU, missing person) | N/A | Rare in commercial systems |
| Public task | 6(1)(e) | Government, statutory bodies | No | Object right (Art. 21) |
| Legitimate interests | 6(1)(f) | Fraud prevention, security, network operations, simple analytics | Object right available | All including objection |

Consent (Art. 6(1)(a) + Art. 7) — must be freely given, specific, informed, unambiguous, by clear affirmative action. Pre-ticked boxes invalid (Recital 32, CJEU C-673/17 Planet49). Bundled consent invalid. Imbalance of power (employer-employee, public-authority) usually makes consent invalid (Recital 43). Withdrawal must be as easy as giving (Art. 7(3)).

Contract (Art. 6(1)(b)) — necessary for performance of a contract to which the data subject is party, or pre-contractual steps at their request. Necessity is strict; Meta lost a CJEU case (C-252/21 Bundeskartellamt) trying to use contract for behavioural advertising. Test: can the service exist without this processing? If yes, not necessary.

Legal obligation (Art. 6(1)(c)) — must be EU or member-state law, not foreign. US subpoena does NOT meet Art. 6(1)(c). Common cases: tax (records 6–10 years depending on country), AML/KYC (5 years post-relationship), payroll, accounting, sector-specific (banking, telecoms data retention).

Legitimate interests (Art. 6(1)(f)) — three-part test (LIA): (1) legitimate interest (purpose), (2) necessity (least intrusive means), (3) balancing against data subject's rights/freedoms/expectations. Cannot be used by public authorities for their public tasks. Data subject has Art. 21 right to object — must stop unless compelling override. Recitals 47–49 list common interests: fraud prevention, network security, direct marketing (with object right).

```yaml
# Records of Processing (Art. 30) — minimum for a single activity
activity: user_account_management
controller: Acme Ltd, 1 Square Mile London EC1
contact: dpo@acme.example
purpose: provide and operate user account
categories_of_data:
  - identifiers: [email, user_id, hashed_password]
  - profile: [display_name, avatar_url, locale, timezone]
  - usage: [login_timestamps, last_seen_ip_anon]
categories_of_subjects: [registered_users]
recipients: [aws_eu_west_1, sendgrid_eu, support_zendesk_eu]
transfers: none_outside_eea
retention: account_lifetime_plus_30_days
lawful_basis: art_6_1_b_contract
security: [tls_1_3, aes_256_at_rest, mfa_for_admins, audit_logs]
```

```python
# Decide-the-basis decision tree (run during DPIA / data review)
def lawful_basis(purpose: str, ctx: dict) -> str:
    if ctx.get("statutory_obligation"):    return "6(1)(c) legal obligation"
    if ctx.get("life_threatening"):        return "6(1)(d) vital interests"
    if ctx.get("public_authority_task"):   return "6(1)(e) public task"
    if ctx.get("contract_necessary"):      return "6(1)(b) contract"
    # Consent vs LI: if subject expects/benefits, LI may apply.
    # If processing is intrusive or unexpected, consent.
    if ctx.get("subject_unaware") or ctx.get("intrusive_profiling"):
        return "6(1)(a) consent"
    if ctx.get("balancing_test_passed"):   return "6(1)(f) legitimate interests"
    raise ValueError("no lawful basis - do not process")
```

| LIA (Legitimate Interests Assessment) template | Mandatory before relying on Art. 6(1)(f) |
|------------------------------------------------|------------------------------------------|
| 1. Identify the legitimate interest | Specific, real, present (not speculative) |
| 2. Necessity test | Could you achieve the same outcome with less data? |
| 3. Balancing test | What is the impact on the subject? Are they vulnerable? Reasonable expectations? |
| 4. Safeguards | Pseudonymisation, opt-out mechanism, retention limits |
| 5. Document and review | At least annually or on material change |

## Special Category Lawful Bases (Art. 9)

Art. 9(1) prohibits processing of special-category data unless one of the Art. 9(2) exceptions applies. **You need both** a Art. 6 lawful basis *and* a Art. 9 exception. Failure to identify the Art. 9 basis is one of the most common findings in regulator audits.

| Basis | Art. 9(2) | Practical use |
|-------|-----------|---------------|
| Explicit consent | (a) | Higher bar than ordinary consent — must be *explicit* statement, not just opt-in |
| Employment / social security / social protection | (b) | Sick leave, occupational health, union deductions — only with EU/MS law authorisation |
| Vital interests where subject incapable of consent | (c) | Unconscious patient |
| Legitimate non-profit activities | (d) | Religious / political / philosophical / trade-union associations re. members |
| Manifestly made public by data subject | (e) | Public Twitter post about your religion, politician's stated beliefs |
| Establishment / exercise / defence of legal claims | (f) | Litigation, regulatory disputes |
| Substantial public interest with EU/MS law | (g) | Anti-doping, electoral registers, anti-fraud where authorised |
| Preventive / occupational medicine / health diagnosis / treatment | (h) | Hospital systems, pharmacy, clinical research with safeguards |
| Public health (cross-border threats, quality of medicines) | (i) | COVID contact tracing under MS law |
| Archiving / scientific or historical research / statistics | (j) | Public-interest research with Art. 89 safeguards |

Art. 10 (criminal convictions/offences) — even more restrictive: only under EU/MS law, only by official authority unless authorised. A SaaS that builds a "fraud reputation" score from criminal records very likely cannot operate under Art. 10 in most member states.

```python
# Pattern: capture both Art. 6 and Art. 9 basis at the field/processing level
class SpecialCategoryProcessing:
    purpose: str
    art6: str   # e.g. "6(1)(a) consent"
    art9: str   # e.g. "9(2)(a) explicit consent"
    safeguards: list[str]
    retention: str

health_intake = SpecialCategoryProcessing(
    purpose="record allergies for safety in restaurant booking",
    art6="6(1)(a) consent",
    art9="9(2)(a) explicit consent (separate checkbox + clear text)",
    safeguards=["field-level encryption", "RBAC", "audit log",
                "auto-purge 90 days post-booking"],
    retention="90 days after the booking date",
)
```

| Member-state derogation | Where to look |
|-------------------------|----------------|
| Genetic / biometric / health (Art. 9(4)) | National laws can add restrictions |
| Employment (Art. 9(2)(b) + Art. 88) | Employment-law-specific rules |
| Research (Art. 9(2)(j) + Art. 89) | National research safeguards |
| Crim convictions (Art. 10) | National "official authority" status |
| Children's age threshold (Art. 8) | 13–16 by member state |

```sql
-- Schema-level enforcement: special-category fields require basis annotation
CREATE TABLE health_records (
    user_id        UUID PRIMARY KEY,
    allergies      BYTEA,                    -- field-level encrypted
    art6_basis     TEXT NOT NULL,
    art9_basis     TEXT NOT NULL,
    consent_id     UUID REFERENCES consent_log(id),
    purpose        TEXT NOT NULL,
    retention_until TIMESTAMPTZ NOT NULL,
    CHECK (art9_basis IN ('9(2)(a)','9(2)(b)','9(2)(c)','9(2)(d)','9(2)(e)',
                          '9(2)(f)','9(2)(g)','9(2)(h)','9(2)(i)','9(2)(j)'))
);
```

| Biometric "for unique identification" trap | |
|--------------------------------------------|---|
| Fingerprint unlock on device | Special-category if it identifies a unique person |
| Face match for KYC | Special-category — explicit consent or Art. 9(2)(g)/(j) |
| Voice authentication | Special-category |
| Fingerprint just for "user count" stats | Probably special if reversible to a person |
| Face filter that does not store data | Not special-category if strictly ephemeral |

## Consent UX Engineering

Art. 4(11): "freely given, specific, informed, unambiguous indication of the data subject's wishes by which he or she, by a statement or by a clear affirmative action, signifies agreement". Art. 7 gives the conditions; Recitals 32, 42, 43 elaborate. EDPB Guidelines 05/2020 on consent are mandatory reading.

| Requirement | Engineering rule |
|-------------|------------------|
| Freely given | No detriment for refusing; cannot be a precondition for service unless processing strictly necessary for that service (Art. 7(4)) |
| Specific | One purpose = one consent. No bundled "we'll use your data for marketing, analytics, profiling, third parties" with one box |
| Informed | Identity of controller, purposes, what's collected, withdrawal mechanism, recipients, transfers, retention — all in plain language *before* consent |
| Unambiguous | Active opt-in. No pre-ticked boxes (CJEU C-673/17 Planet49), no implicit consent from "continued use", no scroll-to-consent |
| By statement / clear affirmative action | Clicking a properly labelled button, ticking an unticked box, configuring a setting |
| Granular | Separate choices for each purpose: necessary / preferences / analytics / marketing / personalization |
| Withdrawal | "As easy as giving" (Art. 7(3)). If giving was one click, withdrawal must be one click — not "email our DPO" |
| Demonstrable | Records: who, when, what version, what they saw |

Records — store sufficient evidence to demonstrate the four conditions later (Art. 7(1) + accountability principle Art. 5(2)).

```sql
CREATE TABLE consent_log (
    id              UUID PRIMARY KEY,
    user_id         UUID NOT NULL,
    purpose         TEXT NOT NULL,            -- 'marketing_email', 'analytics'
    granted         BOOLEAN NOT NULL,
    notice_version  TEXT NOT NULL,            -- e.g. 'privacy-2024-03-12'
    ui_version      TEXT NOT NULL,            -- consent-banner v3.2
    method          TEXT NOT NULL,            -- 'banner', 'settings_page'
    ip_anon         INET,                     -- /24 truncated; or hash
    user_agent_hash TEXT,
    created_at      TIMESTAMPTZ DEFAULT now(),
    withdrew_at     TIMESTAMPTZ
);

-- Withdrawal as easy as granting:
UPDATE consent_log
SET granted = false, withdrew_at = now()
WHERE user_id = $1 AND purpose = $2 AND granted = true;
```

```javascript
// Pre-consent: NO non-essential cookies, NO trackers, NO Google Analytics,
//             NO Facebook Pixel, NO ads, NO heatmap, NO chat widget that
//             sets a tracking cookie before consent.
// Render banner ONLY with a buttoned UI; clicking outside is not consent.
const consent = {
  necessary: true,    // pre-checked & disabled — strictly necessary only
  preferences: false,
  analytics: false,
  marketing: false,
};
// "Reject all" must be as prominent as "Accept all".
// "Accept all" + tiny "manage preferences" link is a dark pattern (CNIL fines).
```

IAB TCF v2.2 — the IAB Transparency & Consent Framework v2.2 (May 2023). Belgian DPA invalidated v1 in 2022 (TCF case) and required remediation. v2.2 dropped "legitimate interest" for advertising purposes 1, 3, 4, 5, 6 — they now require consent only. Engineering: if you use TCF, store the TC string (encoded base64), respect the bits, and implement the "WithdrawalReplaysList" so a vendor that learned a consent must learn the withdrawal.

| TCF v2.2 change | Engineering impact |
|-----------------|---------------------|
| LI removed for ad purposes 1,3,4,5,6 | Must rely on consent for personalised ads |
| "Reject all" parity | UX redesign |
| Plain-language vendor purposes | Update CMP copy |
| Standard-format vendor info | Centrally maintained |

Cookie wall / "Pay or OK" — EDPB Opinion 08/2024 (April 2024): large online platforms generally cannot rely on consent obtained via "consent or pay" without offering a genuine equivalent free alternative without behavioural advertising. CNIL deliberation 2020-091 forbade pure cookie walls. Anti-pattern: "Accept all cookies or buy a subscription" with no third option.

```jsx
// React-style consent provider with parity rules
import {createContext, useContext, useState, useCallback} from "react";

const Consent = createContext({});
export const useConsent = () => useContext(Consent);

export function ConsentProvider({children}) {
  const [state, set] = useState(() => loadState() || {
    necessary: true, preferences: false, analytics: false, marketing: false,
    version: BANNER_VERSION,
  });
  const grant   = (k) => persist(set, {[k]: true});
  const revoke  = (k) => persist(set, {[k]: false});
  const acceptAll = useCallback(() =>
    persist(set, {preferences: true, analytics: true, marketing: true}), []);
  const rejectAll = useCallback(() =>
    persist(set, {preferences: false, analytics: false, marketing: false}), []);
  return <Consent.Provider value={{state, grant, revoke, acceptAll, rejectAll}}>
    {children}
  </Consent.Provider>;
}

function persist(set, patch) {
  set(prev => {
    const next = {...prev, ...patch, ts: Date.now(),
                  version: BANNER_VERSION, ipAnon: window.__ipAnon};
    fetch("/consent", {method: "POST", body: JSON.stringify(next)});
    localStorage.setItem("consent", JSON.stringify(next));
    return next;
  });
}
```

| Banner anti-pattern audit (CNIL/ICO findings) | |
|------------------------------------------------|---|
| Accept button green, reject button grey + small | Lack of parity |
| "Manage choices" two clicks away while accept is one | Asymmetric friction |
| "Continue browsing implies consent" | Not affirmative |
| Cookie set before banner closed | Pre-consent processing |
| Loading Google Tag Manager which loads pixel before consent | Cascade fails |
| Banner reappears every page, "consent fatigue" abuse | Pattern of dark UX |
| Hidden "essential" cookies that include analytics | Mislabelled |
| Country detection turns banner off for non-EU but EU users on VPN see nothing | Geofence is not a basis |

## Data Subject Rights (Articles 15-22)

Eight rights enforceable against the controller. Response window is 1 month from receipt of the request (Art. 12(3)), extendable by 2 further months for complex/numerous requests with notification within the first month. Free of charge unless manifestly unfounded or excessive.

| Right | Article | Engineering surface |
|-------|---------|---------------------|
| Right to be informed | 13, 14 | Privacy notice, layered notice |
| Right of access | 15 | Self-service "download my data" or DSAR queue |
| Right to rectification | 16 | Edit profile, correction request flow |
| Right to erasure ("be forgotten") | 17 | Cascade delete across all systems |
| Right to restriction | 18 | "Freeze" flag — readable, no further processing |
| Right to be informed re. rectification/erasure/restriction | 19 | Notify recipients automatically |
| Right to data portability | 20 | Machine-readable export |
| Right to object | 21 | Opt-out for direct marketing (absolute) and Art. 6(1)(e)/(f) processing |
| Rights re. automated decisions | 22 | Opt-out of solely-automated decisions with legal/significant effect |

| Step | Days from receipt | Action |
|------|-------------------|--------|
| 0 | Day 0 | Request received via channel (email, web form, post, DPO inbox, social) |
| 1 | Day 0–3 | Identity verification proportionate to risk |
| 2 | Day 0–25 | Gather data across all systems |
| 3 | Day 25–30 | Review, redact third-party data, package |
| 4 | Day 30 | Deliver via secure channel; record in DSR log |
| 4b | Day 30 | If extending: notify subject + reasons within 30 days |

```python
# DSR queue: implement once, route per-right
class DataSubjectRequest:
    id: UUID
    user_id: UUID
    right: Literal["access","rectify","erase","restrict",
                   "portability","object","auto_decision"]
    received_at: datetime
    deadline: datetime           # received_at + 30 days
    status: Literal["received","verifying","processing","complete","refused","extended"]
    verification_method: str
    extension_reason: str | None
    response_uri: str | None     # signed S3 URL
    audit: list[Event]

# Always attach an audit chain — regulators ask for the workflow
```

Verification — Art. 12(6): controller may request additional information necessary to confirm identity *if reasonable doubts exist*. Don't over-collect. Already-logged-in user: just MFA re-auth. Email-only request from anonymous person: send signed link to the email on file. Demanding a passport scan for every request is excessive (CNIL has fined for this).

| Refusal grounds | Where to find |
|-----------------|----------------|
| Manifestly unfounded / excessive (Art. 12(5)) | Repetitive, abusive |
| Identity unverifiable (Art. 12(6)) | After reasonable steps |
| Disproportionate effort with safeguards (Art. 14(5)(b)) | Indirect collection only |
| Erasure exemptions (Art. 17(3)) | Legal obligation, public interest, claims |
| Portability limited to consent/contract + provided-by-subject (Art. 20(1)) | Excludes derived data |

```python
# DSR queue worker — minimal but sufficient
from dataclasses import dataclass
from datetime import datetime, timedelta

@dataclass
class DSR:
    id: str; user_id: str; right: str
    received_at: datetime
    deadline: datetime
    status: str = "received"

def dispatch(dsr: DSR):
    handlers = {
        "access":      handle_access,
        "rectify":     handle_rectify,
        "erase":       handle_erase,
        "restrict":    handle_restrict,
        "portability": handle_portability,
        "object":      handle_object,
        "auto_decision": handle_auto_decision,
    }
    return handlers[dsr.right](dsr)

def receive_dsr(user_id: str, right: str) -> DSR:
    now = datetime.utcnow()
    return DSR(id=uuid4().hex, user_id=user_id, right=right,
               received_at=now, deadline=now + timedelta(days=30))
```

| Identity verification pattern | Use when |
|-------------------------------|----------|
| Logged-in user + fresh MFA | Account-bound DSR |
| Magic link to email on file | Account-bound, no live session |
| Out-of-band code to phone on file | High-risk requests |
| Two-factor: email + phone | High-confidence cases |
| Passport scan | Disproportionate; only if other channels failed and risk is high |
| Postal address verification | Government-records context |
| Notarised statement | Litigation-grade only |

| Channel coverage | Required |
|------------------|----------|
| Web form on /privacy/dsr | Yes |
| Email to dpo@ | Yes — must be monitored daily |
| Postal letter | Yes — Art. 12 says any channel |
| Verbal request to support | Counts; transcribe + create ticket |
| Social media DM | Counts if to verified channel; route to web form |
| Through a third-party advocate (NOYB-style) | Counts; verify authorisation |

## Right to Erasure Engineering

Art. 17 grounds: data no longer needed; consent withdrawn (and no other basis); successful objection; unlawfully processed; legal obligation to erase; child's data collected under Art. 8. Exceptions in Art. 17(3): freedom of expression, legal obligation, public interest in health, archiving/research/statistics under Art. 89, legal claims.

| System | Treatment |
|--------|-----------|
| Primary OLTP DB | Delete row; or pseudonymize if FK-constrained; cascade FKs |
| Read replicas | Inherit from primary |
| Data warehouse / analytics (Snowflake/BigQuery/Redshift) | Delete from raw + transformed; rebuild aggregates if user-keyed |
| Logs (app, web, audit) | Pseudonymize at ingestion or delete by user_id index |
| Backups | Document retention; do NOT restore deleted user; expire backups on cycle |
| Search indexes (Elasticsearch/OpenSearch) | Delete by query (`DELETE BY QUERY`) |
| Caches (Redis, Memcached) | Invalidate keys; let TTL expire |
| Object storage (S3, GCS) | Delete user-keyed prefix, lifecycle for any orphans |
| CDN | Purge cache (CloudFront invalidation) |
| Mailing lists (SendGrid, Mailchimp) | API delete; unsubscribe; suppression list |
| CRM / Helpdesk (Zendesk, Salesforce) | Anonymize tickets, hard-delete contact |
| Sub-processors | DPA must require erasure on instruction; track |
| Third-party analytics (GA, Mixpanel, Amplitude) | User-deletion API |
| Payment processor | Stripe/Adyen retain for legal obligation — pseudonymize internal links |
| Sentry / error trackers | Scrub PII; delete events by user |
| ML training data | Re-train if user data was inputs and they request erasure (active EDPB topic) |
| Embeddings / vector DB | Delete user-derived vectors |

Soft-delete vs hard-delete — soft-delete (`deleted_at` flag, row preserved) is fine as a *step* but does not satisfy Art. 17 on its own; the row must eventually be physically deleted or fully anonymized. Document the retention horizon and run the hard-delete job. Soft-delete with infinite retention is non-compliant.

```python
# Idempotent erasure pipeline — safe to re-run
def erase_user(user_id: UUID, reason: str) -> ErasureReport:
    report = ErasureReport(user_id=user_id, reason=reason)

    # 1. Mark in DSR table (audit, withstands re-runs)
    record_dsr(user_id, "erase", status="processing")

    # 2. Per-system erasure — each idempotent
    for system in [
        primary_db, read_replicas_done_via_repl,
        data_warehouse, search, cache, blob_storage,
        analytics_provider, email_provider, support_system,
        sentry, vector_db,
    ]:
        try:
            system.erase(user_id)
            report.add(system.name, "ok")
        except NotFoundError:
            report.add(system.name, "already_erased")
        except Exception as e:
            report.add(system.name, "error", str(e))
            raise

    # 3. Backups: do not delete media; instead record so restore won't repopulate
    backup_excludelist.add(user_id)

    # 4. Notify recipients (Art. 19)
    notify_recipients(user_id, "erasure")

    record_dsr(user_id, "erase", status="complete")
    return report
```

Backup retention conflict — backups are personal data. The accepted regulator approach: encrypt backups, restrict access, document retention period (90/180/365 days), automatically expire. If you must restore from a backup containing erased users, immediately re-run the erasure on the restored data (use the excludelist). EDPB has acknowledged this as a "technical and organisational measure" rather than refusing erasure outright. Do **not** claim "backups are exempt" — they are not.

```sql
-- Excludelist: even if a row reappears from a restore, it gets re-erased
CREATE TABLE erasure_excludelist (
    user_id     UUID PRIMARY KEY,
    erased_at   TIMESTAMPTZ NOT NULL,
    reason      TEXT,
    last_check  TIMESTAMPTZ
);

-- Post-restore hook: run after any restore from backup
CREATE OR REPLACE FUNCTION re_erase_after_restore() RETURNS void AS $$
BEGIN
    DELETE FROM users u
     USING erasure_excludelist e
     WHERE u.id = e.user_id;
    UPDATE erasure_excludelist SET last_check = now();
END;
$$ LANGUAGE plpgsql;
```

```bash
# Backup retention budget per data class
# legal-records             7y    — Art. 6(1)(c) tax/AML
# financial-transactions    7y    — same
# user-account-snapshots    180d  — operational restore
# email-logs                30d
# audit-trail               12m
# session-store             0     (do not back up)

aws s3api put-bucket-lifecycle-configuration --bucket acme-backups-eu \
  --lifecycle-configuration file://lifecycle.json
```

```yaml
# Erasure SLA per system class
hot_systems:    minutes-to-hours        # OLTP, search, cache
warm_systems:   24h                     # warehouse, replicas, sentry
cold_systems:   on next restore         # backups via excludelist
processors:     30 days (DPA SLA)       # third parties
```

| Erasure exception (Art. 17(3)) | Engineering effect |
|--------------------------------|---------------------|
| Freedom of expression / information | News archives may keep |
| Legal obligation (tax 6–10y, AML 5y, payroll varies) | Pseudonymise + retain just the necessary minimum |
| Public interest in health (Art. 9(2)(h)/(i)) | Limited to that purpose |
| Archiving / research / statistics (Art. 89) | Strict safeguards |
| Establishment / exercise / defence of legal claims | Litigation hold |

## Data Portability Engineering

Art. 20 — only applies to processing based on consent (Art. 6(1)(a) / Art. 9(2)(a)) or contract (Art. 6(1)(b)) and only carried out by automated means. Excludes legitimate interests, legal obligation, vital interests, public task. Limited to data the data subject has *provided* (including observed data per WP29 WP242, but not derived/inferred).

| Format characteristics | Required |
|------------------------|----------|
| Structured | Yes |
| Commonly used | Yes (JSON, CSV, XML) |
| Machine-readable | Yes (PDF alone is NOT enough) |
| Direct controller-to-controller transfer | Where technically feasible (Art. 20(2)) |
| Free of charge | Yes |

```bash
# Recommended: ZIP with manifest + per-domain JSON/CSV
my-data-2024-04-17.zip
manifest.json          # what's inside, schema, dates, controller
README.txt             # human-readable summary
profile.json           # account profile
posts.csv              # user-generated content
messages.ndjson        # one JSON per line for streaming
orders.json            # transactional
consents.json          # consent history
connections.csv        # contacts/follow graph (provided-by-subject)
attachments/           # files the user uploaded
checksums.sha256
```

```json
{
  "manifest_version": "1.0",
  "format": "gdpr-portability/v1",
  "controller": {"name": "Acme", "contact": "dpo@acme.example"},
  "subject_id": "u_2c4e",
  "exported_at": "2024-04-17T10:14:22Z",
  "coverage": "all data provided by the subject and observed about them",
  "schemas": {
    "posts": "https://acme.example/schemas/posts.v1.json",
    "messages": "https://acme.example/schemas/messages.v1.json"
  },
  "exclusions": ["derived risk score (not portable per WP29 WP242)"],
  "checksums": "checksums.sha256"
}
```

Direct transfer — ideally an OAuth/OIDC-style flow where the receiving controller pulls the export. Few mature standards; Data Transfer Project (DTP, Apple/Google/Meta/Microsoft) is the closest. If unfeasible, the subject downloads and uploads themselves.

```python
# Portability export builder
import json, csv, zipfile, hashlib
from io import BytesIO, StringIO

def build_portability_zip(user_id) -> bytes:
    buf = BytesIO()
    with zipfile.ZipFile(buf, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("manifest.json", json.dumps(manifest_for(user_id), indent=2))
        z.writestr("README.txt", README_TEMPLATE.format(user_id=user_id))
        z.writestr("profile.json", json.dumps(profile(user_id)))
        z.writestr("posts.csv", to_csv(posts(user_id)))
        z.writestr("messages.ndjson",
                   "\n".join(json.dumps(m) for m in messages(user_id)))
        z.writestr("orders.json", json.dumps(orders(user_id)))
        z.writestr("consents.json", json.dumps(consents(user_id)))
        for path, content in attachments(user_id):
            z.writestr(f"attachments/{path}", content)
        z.writestr("checksums.sha256", checksums(z))
    return buf.getvalue()
```

| Provided / observed / derived | Portable? |
|-------------------------------|-----------|
| Provided (entered by user) | YES |
| Observed (logs, location, behaviour, device IDs) | YES (WP242) |
| Derived (model output, score, tier, segment) | NO |
| Inferred (likely interests, predicted churn) | NO |

## Right to Access Endpoint

Art. 15 — right to confirmation that processing occurs; right to access; right to a copy. Information that must be provided is in Art. 15(1)(a)–(h): purposes, categories of data, recipients, retention, rights, source if not from subject, automated decision logic.

```http
GET /api/me/personal-data HTTP/1.1
Authorization: Bearer <user-access-token-with-mfa-claim>
Accept: application/zip

200 OK
Content-Type: application/zip
Content-Disposition: attachment; filename="acme-data-2024-04-17.zip"
X-Generated-At: 2024-04-17T10:14:22Z
X-Notice-Version: privacy-2024-03-12
```

```yaml
# Endpoint requirements
auth:                fresh MFA within 5 minutes
rate_limit:          1 request per user per 24h, queue if heavier
audit:               log every request — who, when, IP/UA, response size, sub-systems queried
delivery:            signed URL, 24-hour expiry, single-download token
contents:            JSON + companion HTML for non-technical readability
include:
  - profile, settings, contacts
  - posts/messages/uploads
  - account events (logins, password resets — last 12 months)
  - consents (current state + history)
  - DSR history for this user
  - retention table (when each will be deleted)
  - controller + DPO contact + supervisory authority
  - sources of data not collected from subject
  - automated decision logic (Art. 15(1)(h))
exclude:
  - third-party PII (other users in shared content) — redact
  - data covered by trade secrets / IP / others' rights — minimise
```

```python
@app.get("/api/me/personal-data")
@require_fresh_mfa(max_age_seconds=300)
@rate_limit("1/24h", scope="user")
def my_data(user=Depends(current_user)):
    audit_log("dsar.access.request", user_id=user.id)
    job = enqueue_export_job(user.id)
    return {"job_id": job.id,
            "estimated_ready": job.eta.isoformat(),
            "status_url": f"/api/me/personal-data/jobs/{job.id}"}
```

Audit log — every access request must be loggable so the regulator can reconstruct: who requested what, when, was the requester verified, was the response delivered, who internally accessed the export, was it deleted from the export bucket within the retention window.

## Privacy Notice Engineering

Articles 13 (data collected from subject) and 14 (data not collected from subject — e.g. enrichment, third-party). Must be provided at point of collection (Art. 13) or within 1 month / first communication / first disclosure (Art. 14(3)). Plain language, concise, transparent (Art. 12(1)).

| Required content | Art. 13 | Art. 14 |
|------------------|---------|---------|
| Identity & contact of controller | Yes | Yes |
| DPO contact (if applicable) | Yes | Yes |
| Purposes + lawful basis | Yes | Yes |
| Legitimate interests (if Art. 6(1)(f)) | Yes | Yes |
| Recipients / categories of recipients | Yes | Yes |
| Transfers outside EEA + safeguards | Yes | Yes |
| Retention period or criteria | Yes | Yes |
| All data-subject rights | Yes | Yes |
| Right to withdraw consent (if Art. 6(1)(a)) | Yes | Yes |
| Right to lodge complaint with supervisory authority | Yes | Yes |
| Whether provision is statutory/contractual + consequences | Yes | N/A |
| Existence of automated decision-making + logic | Yes | Yes |
| Source of the data | N/A | **Required** (Art. 14(2)(f)) |

Layered notice — short notice at collection (just-in-time), full notice on a dedicated /privacy page, granular detail in collapsible sections. EDPB Guidelines 03/2017 and ICO guidance both endorse layering. Do not put the same content in 30,000 words and call it a day; do not put 200 words and link "more" to a 30,000-word PDF.

```html
<!-- Just-in-time at form -->
<small>
  We use your email to log in and contact you. Stored in EU.
  Full <a href="/privacy">privacy notice</a>.
</small>

<!-- Cookie banner just-in-time -->
<div class="cb">
  We use cookies for site function. We'd like consent for analytics
  and personalised ads. <a href="/privacy/cookies">Cookie details</a>.
  <button data-act="accept">Accept all</button>
  <button data-act="reject">Reject all</button>
  <button data-act="manage">Manage</button>
</div>
```

Versioning — store privacy notice as Markdown/JSON in a repo, tag every version, expose `/privacy?version=2024-03-12`, link to changelog. When materially changed, prompt users at next login. Re-consent if you change the lawful basis, add purposes, or expand recipients in a way the original consent did not cover (EDPB Guidelines 05/2020).

```yaml
# privacy-notice.yaml
version: 2024-03-12
prev_version: 2023-11-04
changes:
  - added recipient: cloudflare_eu (DDoS protection, Art. 6(1)(f))
  - clarified retention: support_tickets reduced 5y -> 3y
material_change: false   # true would trigger user prompt
```

## Cookie / Tracker Compliance

Cookies are governed primarily by the ePrivacy Directive 2002/58 (as amended by 2009/136) — known as the "ePrivacy Directive" or "Cookie Law" — implemented per member state (PECR in UK, LIL in France, TKG in Germany). GDPR provides the consent standard (Art. 4(11), Art. 7) but the *trigger* is ePrivacy Article 5(3): consent required for storage or access of information on terminal equipment, except strictly necessary for an explicitly requested service.

| Cookie type | Consent? | Examples |
|-------------|----------|----------|
| Strictly necessary | NO consent needed | Session token, CSRF token, load-balancer affinity, shopping-cart, language preference set by user |
| User preferences | Usually consent (if not requested) | Theme, currency override |
| Analytics | YES (in most MS) | Google Analytics, Plausible (debated — server-side, no PII), Mixpanel |
| Marketing / ads | YES | Facebook Pixel, Google Ads, retargeting |
| Functional non-essential | YES | Embedded YouTube, Twitter widgets, Intercom chat |

| Anti-pattern | Why bad | Authority |
|--------------|---------|-----------|
| Pre-ticked boxes | Not unambiguous | CJEU C-673/17 Planet49 |
| "Accept all" prominent + tiny "manage" link | Not freely given | CNIL fines 2020+ |
| Continued use = consent | Not a clear affirmative action | EDPB Guidelines 05/2020 |
| Cookie wall ("accept or leave") | Not freely given | EDPB Opinion 8/2024, CNIL 2020-091 |
| "Reject all" buried two clicks deep | Not freely given | CNIL Google €150M, Facebook €60M (2022) |
| Setting cookies *before* consent | Pre-consent processing | CNIL many |

"Pay or OK" — large platforms (Meta, news sites) offering "consent for ads, or pay subscription". EDPB Opinion 08/2024 (April 2024) on consent or pay: large online platforms generally cannot rely on this binary alone; need a genuine free alternative without behavioural advertising. National authorities still litigating individual cases.

```javascript
// Engineering: gate ALL non-essential scripts behind consent
function loadAnalytics() {
  if (consent.analytics !== true) return;
  const s = document.createElement('script');
  s.src = 'https://plausible.io/js/script.js';
  s.defer = true;
  document.head.appendChild(s);
}
window.addEventListener('consent:granted', (e) => {
  if (e.detail.includes('analytics')) loadAnalytics();
});
```

## Privacy by Design (Art. 25)

Art. 25(1) — data protection by design: at the time of determining means of processing AND at processing time, implement appropriate technical and organisational measures (TOMs) — pseudonymisation, minimisation. Art. 25(2) — by default: only data necessary for the specific purpose, minimum extent, minimum retention, minimum accessibility (default-private).

| Principle (Art. 5) | Engineering implication |
|--------------------|--------------------------|
| Lawfulness, fairness, transparency (5(1)(a)) | Lawful basis recorded, privacy notice published |
| Purpose limitation (5(1)(b)) | Tag every field with purposes; alarm if used elsewhere |
| Data minimisation (5(1)(c)) | Optional fields *off* by default; collect only what is necessary |
| Accuracy (5(1)(d)) | Edit profile UI; correction flow; verify on input |
| Storage limitation (5(1)(e)) | Retention schedules in DB schema; auto-purge jobs |
| Integrity & confidentiality (5(1)(f)) | TLS everywhere, encryption at rest, MFA, RBAC, audit logs |
| Accountability (5(2)) | Records (Art. 30), DPIA, audit trail, evidence files |

```sql
-- Default-private: profile fields explicit
CREATE TABLE profile (
    user_id     UUID PRIMARY KEY,
    -- visibility default = private; user opts up
    bio         TEXT,    bio_vis      visibility NOT NULL DEFAULT 'private',
    location    TEXT,    location_vis visibility NOT NULL DEFAULT 'private',
    avatar_url  TEXT,    avatar_vis   visibility NOT NULL DEFAULT 'public',
    -- retention: derived
    last_seen   TIMESTAMPTZ,        -- 12 month rolling
    created_at  TIMESTAMPTZ DEFAULT now()
);
```

```python
# Minimisation at the API
class SignupForm(BaseModel):
    email: EmailStr                            # lawful: contract
    password: SecretStr                        # lawful: contract
    # Only ask for what is *necessary now*:
    # NOT: phone, address, gender, dob — collect later only with purpose
```

| Default Privacy Pattern | Implementation |
|-------------------------|----------------|
| New social account | Posts default to "followers only", profile to "private" |
| New chat thread | E2EE on by default |
| Geolocation | "While using app", not "always" |
| Notifications | Opt-in per type, not opt-out |
| Search engine indexing | `noindex` on profile until user opts in |
| Data sharing with third parties | Off; explicit toggle |

```python
# Purpose-tagged data access — fail-closed if used outside declared purpose
class Field:
    def __init__(self, name, purposes: set[str]):
        self.name, self.purposes = name, purposes

EMAIL = Field("email", {"login", "transactional", "support"})
# Reading EMAIL for purpose="marketing" raises:

def read_field(field: Field, *, purpose: str, ctx) -> Any:
    if purpose not in field.purposes:
        raise PurposeViolation(f"{field.name} not allowed for {purpose}")
    audit("field.read", field=field.name, purpose=purpose, user=ctx.user)
    return _read(field.name, ctx)
```

```yaml
# Retention schedule — declarative, applied by job
schedule:
  - table: sessions
    purge_after: 30d
    purge_field: last_active
  - table: support_tickets
    purge_after: 3y
    purge_field: closed_at
  - table: marketing_events
    purge_after: 24m
    purge_field: ts
  - table: audit_log
    purge_after: 12m
    purge_field: ts
  - table: webhook_payloads
    purge_after: 14d
    purge_field: ts
```

| Art. 5 principle | Engineering tripwire |
|------------------|----------------------|
| Lawful basis logged on every record | Refuse insert without it |
| Purpose tag per field | Refuse cross-purpose read |
| Minimisation: optional vs required | Form only requires the minimum |
| Accuracy | UI for self-correction, periodic verify-email pings |
| Retention TTL | Cron / Lifecycle / DB partition drop |
| Integrity | TLS / encryption / MFA / RBAC / audit |
| Accountability | Records, DPIA, evidence repo |

## DPIA (Art. 35)

Data Protection Impact Assessment — required where processing is "likely to result in a high risk to the rights and freedoms of natural persons". Must be done *before* processing starts. EDPB list of operations always requiring DPIA: large-scale special-category processing, large-scale systematic monitoring of public areas, processing children's data, biometrics for unique identification, innovative use of new tech, automated decision-making with legal effect, processing preventing exercise of a right.

| Trigger from Art. 35(3) + EDPB list | Example |
|-------------------------------------|---------|
| Systematic and extensive evaluation of personal aspects (incl. profiling) with legal/significant effects | Credit scoring, AI hiring |
| Large-scale Art. 9 special-category or Art. 10 criminal data | Patient registries |
| Systematic monitoring of publicly accessible area on a large scale | CCTV city centre |
| Innovative tech | Biometric attendance, smart home AI |
| Combining datasets from different sources | Data broker enrichment |
| Vulnerable subjects | Children, employees |
| Preventing exercise of a right or use of a contract | Insurance underwriting |

```markdown
# DPIA Template
1. Description: nature, scope, context, purposes; data flow diagram
2. Necessity & proportionality: Art. 6/9 basis; data minimisation; retention
3. Risks to rights & freedoms: confidentiality, integrity, availability, secondary use, discrimination, financial loss, identity theft
4. Likelihood x Severity: low/medium/high
5. Measures: pseudonymisation, encryption, RBAC, logging, training, DPA terms
6. Residual risk: low/medium/high
7. Consultation:
   - Data subjects' views (Art. 35(9), where appropriate)
   - DPO opinion
   - Supervisory authority (Art. 36) if residual risk remains high
8. Sign-off: controller + DPO + date + version
9. Review cadence: annually, or on change
```

Prior consultation (Art. 36) — if the DPIA shows residual high risk, controller must consult the supervisory authority *before* processing. SA has 8 weeks (extendable +6) to respond.

```python
# DPIA risk scoring - simple but useful
LIKELIHOOD = {"rare":1,"unlikely":2,"possible":3,"likely":4,"almost_certain":5}
SEVERITY  = {"insignificant":1,"minor":2,"moderate":3,"major":4,"severe":5}

def risk_score(likelihood: str, severity: str) -> int:
    return LIKELIHOOD[likelihood] * SEVERITY[severity]

def risk_band(score: int) -> str:
    if score <= 4:   return "low"
    if score <= 9:   return "medium"
    if score <= 16:  return "high"
    return "very high - prior consultation required"
```

| DPIA threshold table (EDPB 9 criteria, ≥2 typically triggers) | Hits |
|--------------------------------------------------------------|------|
| Evaluation / scoring | yes |
| Automated decision with significant effect | yes |
| Systematic monitoring | yes |
| Sensitive / special category data | yes |
| Large-scale processing | yes |
| Datasets that have been combined or matched | yes |
| Vulnerable subjects (children, employees, patients) | yes |
| Innovative use of new technology | yes |
| Prevents subjects from exercising a right or contract | yes |

## Privacy Engineering Patterns

| Pattern | Goal | When |
|---------|------|------|
| Pseudonymisation at rest | Limit blast radius if DB leaks | Always for direct identifiers in non-auth tables |
| Encryption at rest | Confidentiality vs disk theft | Always (LUKS, AWS KMS, Azure SSE) |
| Field-level encryption | Limit who/what reads sensitive fields | Health, biometrics, financial, special-cat |
| Tokenisation | PCI scope reduction; portable identifiers | Payment cards, government IDs |
| Hashing identifiers | One-way, but: weak for low-entropy inputs | Email "do you have an account" must use slow KDF + pepper |
| Differential privacy | Aggregate stats with mathematical privacy guarantee | Telemetry, OS usage, public datasets |
| k-anonymity / l-diversity / t-closeness | Anonymise micro-data | Released datasets, research |
| Federated learning | Model updates without centralising raw data | Mobile keyboards, health AI |
| On-device processing | Avoid collection altogether | Photo categorisation, voice commands |
| Homomorphic encryption | Compute on encrypted data | Niche, expensive |
| Secure multi-party computation | Joint compute without sharing | Fraud rings, ad measurement |
| Confidential computing (TEEs) | Process plaintext only inside enclaves | Cloud workloads with sensitive data |

```python
# Field-level encryption with envelope keys
import boto3
kms = boto3.client("kms", region_name="eu-west-1")

def encrypt_field(plaintext: bytes, context: dict) -> bytes:
    out = kms.generate_data_key(
        KeyId="alias/eu-pii", KeySpec="AES_256",
        EncryptionContext=context,        # binds ciphertext to user_id/purpose
    )
    cipher = aes_gcm_encrypt(out["Plaintext"], plaintext)
    return out["CiphertextBlob"] + cipher

def decrypt_field(blob: bytes, context: dict) -> bytes:
    edk, cipher = blob[:184], blob[184:]
    pt = kms.decrypt(CiphertextBlob=edk, EncryptionContext=context)["Plaintext"]
    return aes_gcm_decrypt(pt, cipher)
```

```python
# Differential privacy: count with Laplace noise
import numpy as np
def dp_count(true_count: int, epsilon: float = 1.0) -> int:
    return int(round(true_count + np.random.laplace(0, 1.0 / epsilon)))
```

```python
# k-anonymity: ensure every quasi-identifier combination has >= k rows
def k_anon_check(df, qid_cols, k=5):
    return df.groupby(qid_cols).size().min() >= k
```

```go
// Tokenisation example: Stripe-style token, no PAN in your DB
type Token struct {
    ID    string // tok_xyz, opaque
    Last4 string
    Brand string
    Exp   string
}

func Tokenize(pan string) (Token, error) {
    // Send to PCI-scope vault; receive token. Your DB never sees PAN.
    return vault.Issue(pan)
}
```

```python
# Format-Preserving Encryption (FPE) for SSNs / NIDs
# (in practice use a vetted library; this is illustrative)
from cryptography.hazmat.primitives.ciphers.aead import AESGCM
def fpe_like(value: str, key: bytes) -> str:
    aead = AESGCM(key)
    nonce = b"\x00"*12
    ct = aead.encrypt(nonce, value.encode(), b"fpe-context")
    return ct.hex()[:len(value)]    # crude shape preservation
```

| Engineering pattern selection guide | When to use |
|-------------------------------------|--------------|
| Pseudonymise | Default for direct identifiers |
| Encrypt at rest | Always for storage media |
| Field-level encrypt | Special-cat or extra-sensitive fields |
| Tokenise | Reduce PCI / KYC scope |
| Hash | One-way derivation; weak for low-entropy inputs |
| Differential privacy | Releasing aggregates externally |
| k-anonymity | Releasing micro-data |
| Federated learning | Train on devices, not central PII |
| TEEs / confidential compute | Cloud workload with sensitive plaintext |

## Data Residency

GDPR is not a strict "data must stay in EU" law — Chapter V (Articles 44–50) governs *transfers* outside the EEA. A transfer is lawful only if the destination has an adequacy decision, OR appropriate safeguards (SCCs, BCRs, codes, certifications) are in place, OR a derogation applies.

| Mechanism | Article | Notes |
|-----------|---------|-------|
| Adequacy decision | 45 | EU Commission decides country offers essentially equivalent protection |
| Standard Contractual Clauses (SCCs) | 46(2)(c) | New 2021 SCCs (modules 1–4); old SCCs invalid since 27 Dec 2022 |
| Binding Corporate Rules (BCRs) | 47 | Intra-group; SA-approved |
| Approved Codes of Conduct | 46(2)(e) | Sectoral with binding commitments |
| Approved Certification | 46(2)(f) | e.g. EuroPriSe |
| Derogations | 49 | Explicit consent, contract necessity, legal claims; narrow, not for systematic transfers |

| Adequate countries (as of 2024) | |
|---------------------------------|---|
| Andorra, Argentina, Canada (commercial), Faroe Islands, Guernsey, Israel, Isle of Man, Japan, Jersey, New Zealand, Republic of Korea, Switzerland, UK, Uruguay, USA (DPF participants only) | |

Schrems II (CJEU C-311/18, 16 July 2020) — invalidated EU-US Privacy Shield. SCCs survived but require a "transfer impact assessment" (TIA) per transfer: assess whether destination law provides essentially equivalent protection; if not, supplementary measures (encryption with EU-controlled keys, pseudonymisation, no plaintext access by importer).

EU-US Data Privacy Framework (DPF) — adopted 10 July 2023 as adequacy decision for self-certified US organisations. Replaces Privacy Shield. NOYB has filed legal challenges expecting Schrems III. Certifying companies are listed on dataprivacyframework.gov.

| TIA (Transfer Impact Assessment) — required steps | |
|---------------------------------------------------|---|
| 1. Identify the transfer (data, route, importer, country) | What goes where |
| 2. Identify the transfer tool (DPF/SCC/BCR) | Mechanism |
| 3. Assess third-country law | Government access, surveillance, redress |
| 4. Identify supplementary measures | Encryption, pseudonymisation, contractual, organisational |
| 5. Procedural steps | Update SCC annex, notify subjects if material |
| 6. Re-evaluate periodically | At least annually |

| US legal regime (worry list per Schrems II) | Relevance |
|---------------------------------------------|-----------|
| FISA 702 | Electronic-communications service providers must comply with surveillance directives |
| EO 12333 | NSA SIGINT outside FISA |
| Cloud Act 2018 | US authorities can compel US-based providers to disclose data wherever stored |
| EO 14086 (2022) | Created the redress mechanism for the DPF |
| PCLOB | Civil-liberties oversight, mentioned in Schrems II |

```yaml
# Multi-region partitioning by user residence
data_plane:
  eu:
    region: eu-west-1
    db: aurora-eu-pg
    storage: s3://acme-eu/
    kms: alias/acme-eu
    log_sink: cloudwatch-eu
  us:
    region: us-east-1
    db: aurora-us-pg
    storage: s3://acme-us/
    kms: alias/acme-us
control_plane:
  region: eu-west-1                # treat as EU; no PII content plane
routing:
  - signup.region = geo-ip + user-declared    # store on user
  - api.requests routed by user.region
  - support tickets by user.region            # follow the user
transfers:
  - none between planes by default
  - emergency cross-region only with SCC + DPIA addendum
```

## Processor Agreements (Art. 28)

A controller may use only processors providing "sufficient guarantees" to implement appropriate TOMs. The relationship must be governed by a written contract or other legal act (Art. 28(3)) — the Data Processing Agreement (DPA). Without it, every transfer to a processor is unlawful.

| Required clause | Detail |
|-----------------|--------|
| Subject matter, duration, nature, purpose | Specify what + how long |
| Type of personal data + categories of subjects | Schedule typically |
| Obligations and rights of the controller | Including instructions |
| Process only on documented controller instructions | Including transfers |
| Persons authorised to process under confidentiality | NDAs, training |
| Security measures (Art. 32) | TOMs schedule |
| Sub-processors only with authorisation (general or specific) | Plus list |
| Assist with data-subject rights | DSR API or workflow |
| Assist with security, breach, DPIA, prior consultation | Contract terms |
| Delete or return data at end of services | Choice of controller |
| Make available all info to demonstrate compliance + audits | Audit right |
| Inform controller if instruction infringes GDPR | Push-back duty |

```yaml
# Schedule of TOMs typically attached to a DPA
encryption:        TLS 1.2+, AES-256 at rest
access_control:    RBAC, MFA for admin, principle of least privilege
audit_logging:     all access to PII, 12-month retention
network:           VPC, security groups, no public DB
backup:            encrypted, 30-day, restore drills quarterly
incident_response: 24x7 SOC, runbooks, breach notification SLA 36h to controller
testing:           pen test annually, vulnerability scans weekly
training:          annual privacy + security training, mandatory
sub_processors:    list maintained at acme.example/subprocessors
deletion:          90-day SLA on contract end + cert of destruction
```

Standard form — EU Commission Implementing Decision 2021/915 published model controller-processor SCCs covering the Art. 28(3) and (4) content. Most SaaS DPAs are bespoke; large processors (AWS, Google Cloud, Microsoft) publish their DPA terms.

| Joint controller (Art. 26) red flag | Test |
|--------------------------------------|------|
| Both parties decide what is collected | Joint |
| Both parties decide why and how | Joint |
| Each independently uses the data for own purposes | Likely two controllers |
| One simply hosts/transmits | Processor |

```yaml
# Joint controller arrangement (Art. 26) — must be transparent to subjects
arrangement:
  parties: [acme, partner_x]
  purpose: combined_analytics_dashboard
  controller_a_responsibilities: [collection, security_a]
  controller_b_responsibilities: [analysis, retention]
  contact_for_subjects: dpo@acme.example     # mandatory single point
  essence_published: https://acme.example/joint-controller-summary
```

## Sub-Processor Management

Sub-processors are processors engaged by your processor. Art. 28(2) — processor must obtain prior specific or general written authorisation from the controller and inform the controller of changes giving the controller the chance to object. Art. 28(4) — sub-processor bound by the same data-protection obligations as the processor (flow-down).

| Engineering action | Detail |
|--------------------|--------|
| Maintain a public sub-processor list | URL like `acme.example/subprocessors` with name, location, role, data categories |
| Notice mechanism | Email subscription or RSS for changes; X days advance notice (typically 30) |
| Object mechanism | Customer can object during notice period; if not resolved, terminate |
| Diligence file per sub-processor | DPA, TOMs evidence, transfer mechanism, security cert (SOC2/ISO27001) |
| Annual re-attestation | Renew SOC2/ISO27001; check breach history |
| Geo footprint | Where the sub-processor processes (regions, countries) |

```yaml
# subprocessors.yaml — single source of truth
- name: AWS
  role: Cloud infrastructure
  location: eu-west-1, eu-central-1
  data: all customer content + metadata + logs
  transfer: EEA only
  dpa: aws.amazon.com/service-terms (Section 2)
  certifications: [ISO27001, SOC2, PCI-DSS]

- name: SendGrid (Twilio)
  role: Transactional email
  location: EU + US
  data: email address, subject, content, timestamps
  transfer: SCCs (2021) Module 2 + DPF
  dpa: twilio.com/legal/data-protection-addendum

- name: Stripe
  role: Payment processing (separate controller for some operations)
  location: EU + US
  data: card holder, transaction metadata
  transfer: SCCs + DPF
  notes: "Stripe is independent controller for some processing under PSD2"
```

## Breach Notification (Art. 33-34)

Personal data breach (Art. 4(12)) — "a breach of security leading to the accidental or unlawful destruction, loss, alteration, unauthorised disclosure of, or access to, personal data". Not only confidentiality breaches: ransomware (availability), accidental wrong-recipient email (disclosure), DB corruption (integrity).

| Timeline | Trigger | Action |
|----------|---------|--------|
| T+0 | Discovery / awareness | Trigger incident response; preserve forensics |
| T+0 to T+72h | Assess risk to rights & freedoms | Likelihood, severity, scope |
| By T+72h | If risk to rights/freedoms | Notify supervisory authority (Art. 33) |
| If high risk | After T+72h or alongside | Notify data subjects (Art. 34) without undue delay |
| Throughout | Always | Document the breach (Art. 33(5)) — even non-notifiable ones |

Authority notification content (Art. 33(3)):
- nature of the breach incl. categories and approx. number of data subjects + records
- DPO contact / other contact point
- likely consequences
- measures taken or proposed (containment, mitigation)

Subject notification (Art. 34) is required if breach is "likely to result in a high risk", unless one of three exceptions: (a) appropriate measures (e.g. encryption) render data unintelligible; (b) subsequent measures prevent the high risk materialising; (c) disproportionate effort — then public communication.

```yaml
# Breach response runbook
T+0:
  - Severity triage; declare incident
  - Snapshot: logs, IAM, network, suspect systems
  - Page DPO + Legal + CISO + Comms
T+1h:
  - Containment: revoke creds, rotate keys, isolate hosts
  - Preserve evidence (do not wipe before forensics)
  - Begin scope assessment: which datasets, how many subjects, type of data
T+8h:
  - First risk assessment
  - Begin draft notifications
T+24h:
  - Risk assessment review with DPO
  - Decide: notify SA? notify subjects?
T+48h:
  - Finalise SA notification (template)
  - Prepare subject communication
T+72h:
  - File with SA (one-stop-shop = lead SA)
  - Subject communication if high-risk
T+7d:
  - Internal post-mortem (blameless)
  - Update Records of Processing if changed
T+30d:
  - Remediation plan tracked to closure
```

```python
# Breach register — required by Art. 33(5) regardless of notification
class BreachEntry:
    id: UUID
    discovered_at: datetime
    occurred_at: datetime | None
    discovered_by: str
    nature: str                    # CIA: confidentiality/integrity/availability
    cause: str                     # phishing, misconfig, vulnerability, insider
    affected_records: int
    affected_subjects: int
    data_categories: list[str]
    risk_assessment: dict          # likelihood, severity, factors
    sa_notified: bool
    sa_notified_at: datetime | None
    subjects_notified: bool
    measures_taken: list[str]
    lessons_learned: str
    closed_at: datetime | None
```

Engineering reality — 72 hours is wall-clock, not business hours, and starts on awareness, not on completion of investigation. Practical: maintain a one-page SA notification template per supervisory authority; pre-fill what's static (controller details, DPO contact); rehearse the workflow annually. Late breach notifications draw heavy fines independently of the breach itself.

| Risk-of-rights-and-freedoms factors (EDPB 9/2022) | Direction |
|---------------------------------------------------|-----------|
| Type of breach (C/I/A) | Confidentiality usually highest |
| Nature, sensitivity, volume | Special-cat, financial, location escalates |
| Ease of identification | Already-identified = higher risk |
| Severity of consequences | Identity theft, fraud, reputation, discrimination |
| Special characteristics of subjects | Children, vulnerable adults |
| Special characteristics of controller | Healthcare, finance, child services |
| Number of affected | Large groups escalate |

```python
# Breach severity quick triage
def breach_risk(facts) -> str:
    score = 0
    if facts.special_category:           score += 3
    if facts.financial_data:             score += 2
    if facts.location_data:              score += 2
    if facts.confidentiality_breach:     score += 2
    if facts.users_affected > 1000:      score += 1
    if facts.users_affected > 100000:    score += 2
    if facts.affected_minors:            score += 2
    if not facts.encrypted_at_rest:      score += 1
    if facts.disclosed_publicly:         score += 2
    if score >= 6: return "high - notify subjects + SA"
    if score >= 3: return "medium - notify SA"
    return "low - record only"
```

```bash
# Pre-staged SA notification (CNIL example) — fill and submit
curl -s -X POST https://notifications.cnil.fr/notifications/breach \
  -F "controller=Acme Ltd" \
  -F "dpo_email=dpo@acme.example" \
  -F "discovered_at=2024-04-17T08:14:22Z" \
  -F "occurred_at=2024-04-16T22:00:00Z" \
  -F "nature=confidentiality" \
  -F "data_categories=email,name,hashed_password" \
  -F "subjects_count=14322" \
  -F "consequences=phishing_risk,credential_stuffing" \
  -F "measures=password_reset_forced,sessions_revoked,SOC_alert"
```

## Records of Processing (Art. 30)

Both controllers (Art. 30(1)) and processors (Art. 30(2)) must maintain records. Exemption (Art. 30(5)) for organisations under 250 employees, narrowly: only if processing is occasional, no special-category, no Art. 10 criminal, no risks to rights/freedoms. In practice almost no SaaS qualifies.

| Required content (Controller) | Required content (Processor) |
|-------------------------------|-------------------------------|
| Controller + DPO + reps contact | Processor + each controller it processes for + DPO/reps |
| Purposes of processing | Categories of processing per controller |
| Categories of data subjects | Transfers + safeguards |
| Categories of personal data | TOMs (general description) |
| Categories of recipients (incl. third countries) | |
| Transfers + identification of country + safeguards | |
| Retention period (or criteria) | |
| TOMs (general description) | |

| Format | Where |
|--------|-------|
| Written, including electronic | Yes |
| Available to supervisory authority on request | Art. 30(4) |
| Common in industry | Spreadsheet, GRC platform, code-as-config |

```yaml
# RoPA as code: machine-checkable, diff-able, reviewable
- activity: marketing_email
  controller: Acme
  dpo: dpo@acme.example
  purposes: [direct_marketing, product_updates]
  lawful_basis: art_6_1_a_consent
  subject_categories: [users_who_opted_in]
  data_categories: [email, name, language, signup_source]
  recipients: [SendGrid_eu]
  transfers: none_outside_eea
  retention: until_unsubscribe_or_2y_inactivity
  toms: [tls, encryption_at_rest, rbac, audit_log, ip_anon, unsubscribe_link]
```

## DPO (Art. 37)

Designation of a Data Protection Officer is mandatory if: processing carried out by a public authority; OR core activities consist of regular and systematic monitoring of data subjects on a large scale; OR core activities consist of large-scale processing of Art. 9 special-category or Art. 10 criminal data.

| Tasks (Art. 39) |
|-----------------|
| Inform and advise the controller/processor + employees |
| Monitor compliance with GDPR + national law + internal policies |
| Provide advice on DPIA + monitor performance (Art. 35) |
| Cooperate with the supervisory authority |
| Act as contact point for SA + subjects |

| Independence (Art. 38) | Detail |
|-------------------------|--------|
| No instructions on tasks | Cannot be told how to do compliance work |
| No dismissal/penalty for performing tasks | Protected like a whistleblower |
| Reports to highest management | CEO/Board, not their direct manager |
| No conflict of interest | DPO cannot also be CISO/CTO/CMO/Legal head whose decisions they audit |
| Adequate resources | Time, budget, training |

Internal vs external — DPO can be staff or external service contractor. SMEs frequently hire fractional/external DPOs. Contact details must be published and communicated to the supervisory authority. DPO contact must appear in the privacy notice.

```yaml
dpo:
  internal: false
  provider: PrivacyCo Limited
  primary_contact: jane.doe@privacyco.example
  email: dpo@acme.example                # routes to provider
  postal: c/o Acme, 1 Square Mile, London EC1
  reporting_line: CEO
  hours_per_month: 40
  training_budget: GBP 5000/yr
```

| Conflict-of-interest test (EDPB Guidelines 5/2017) | Generally fail |
|-----------------------------------------------------|----------------|
| CEO / COO / CFO | Fail — set strategic direction |
| CIO / CTO / Head of IT | Fail — define means of processing |
| Head of Marketing / Sales | Fail — own the data flows being audited |
| Head of HR | Fail — controller of employee data |
| General Counsel | Often fail — represents the company in disputes |
| Compliance Officer (separate from data) | Possible if scoped clearly |
| External counsel as DPO | Acceptable; common at SMEs |

## Children's Data (Art. 8)

Information society services offered directly to a child — consent of the child is lawful if child is at least 16 (default) or whatever lower age the member state sets between 13 and 16. Parental authorisation needed below the threshold. Controller must make reasonable efforts to verify, taking available technology into account.

| Country | Age threshold |
|---------|---------------|
| 13 | UK, Belgium, Denmark, Estonia, Finland, Latvia, Malta, Norway, Portugal, Sweden |
| 14 | Austria, Bulgaria, Cyprus, Italy, Lithuania, Spain |
| 15 | Czechia, France, Greece, Slovenia |
| 16 | Croatia, Germany, Hungary, Ireland, Liechtenstein, Luxembourg, Netherlands, Poland, Romania, Slovakia |

| Engineering | Implementation |
|-------------|----------------|
| Age gate | Self-declared birthdate, not "are you over X" tickbox |
| Parental verification | Email-then-credit-card, signed consent form, video call — proportionate to risk |
| Reasonable effort | Document the chosen method + why it is reasonable for your audience |
| Repeat consent at majority | Ask the child to re-consent on turning 18 (best practice) |
| No profiling of children for marketing | EDPB Guidelines 02/2023 + national rules |
| Privacy notice for children | Plain language, age-appropriate; ICO Age Appropriate Design Code |

```python
# Age gate with date-of-birth + member-state threshold
COC = {"GB":13,"DE":16,"FR":15,"IE":16,"NL":16,"PL":16,"IT":14,"ES":14,
       "PT":13,"SE":13,"DK":13}        # incomplete; load full table

def consent_path(country: str, dob: date) -> str:
    age = relativedelta(date.today(), dob).years
    threshold = COC.get(country, 16)
    if age >= threshold:
        return "self_consent"
    if age >= 13:
        return "parental_consent_required"
    return "block_or_kid_safe_mode"
```

| ICO Age-Appropriate Design Code (15 standards) | Engineering applicability |
|------------------------------------------------|----------------------------|
| Best interests of the child | Default decision lens |
| DPIA before launch | Mandatory |
| Age-appropriate application | Identify likely child users |
| Transparency | Plain language, age-appropriate |
| Detrimental use of data | No dark patterns, no addictive nudges |
| Policies and community standards | Enforce them for child users |
| Default settings | Maximum privacy by default |
| Data minimisation | Even more strict for children |
| Data sharing | Off by default for children |
| Geolocation | Off by default for children |
| Parental controls | Provide if appropriate |
| Profiling | Off by default for children |
| Nudge techniques | Avoid those that lower privacy |
| Connected toys & devices | Same standards apply |
| Online tools | Easy DSR exercise |

## Profiling & Automated Decisions (Art. 22)

Profiling (Art. 4(4)) — any automated processing of personal data to evaluate personal aspects (work performance, economic situation, health, preferences, interests, reliability, behaviour, location, movements). Art. 22 — data subject has the right "not to be subject to a decision based solely on automated processing, including profiling, which produces legal effects ... or similarly significantly affects" them.

| Exception (Art. 22(2)) | Notes |
|------------------------|-------|
| Necessary for performance of a contract | Limited; cannot be a hidden bypass |
| Authorised by EU/MS law with safeguards | Member-state-specific |
| Based on explicit consent | Higher-bar consent + safeguards |

| Required safeguards (Art. 22(3) + Recital 71) |
|-----------------------------------------------|
| Right to obtain human intervention |
| Right to express point of view |
| Right to contest the decision |
| Right to explanation of the logic involved (Art. 13(2)(f), 14(2)(g), 15(1)(h)) |

| Examples likely solely-automated, significant effect |
|-------------------------------------------------------|
| Credit decisions, insurance underwriting |
| Automated hiring rejections (CV screeners) |
| Automated benefit/welfare decisions |
| Automated content suspension with no review |
| Differential pricing if material |

```python
# Engineering pattern: never "solely automated" for high-impact decisions
def underwrite(application) -> Decision:
    score = ml_score(application)              # automated
    if score < REJECT_HARD or score > APPROVE_HARD:
        decision = decide_auto(score)
        log_explanation(application, score, decision)
        return decision
    # Borderline -> human reviewer queue (meaningful, not rubber-stamp)
    return queue_for_human_review(application, ml_hint=score)
```

EDPB Guidelines on Automated Decision-Making and Profiling (WP251rev.01) — "meaningful information about the logic" does not require disclosing the algorithm itself but does require enough for the subject to understand how decisions are reached and to exercise rights. Provide categories of input, importance of factors, and how to challenge.

| Article | Right against profiling |
|---------|--------------------------|
| 13(2)(f), 14(2)(g) | Prior info: existence + meaningful info on logic + significance |
| 15(1)(h) | DSAR access to the same |
| 21(1)–(2) | Object — absolute for direct marketing |
| 22(1) | Not subject to solely-automated decision with legal/significant effect |
| 22(3) | Right to human intervention, contest, express POV |
| Recital 71 | Sets the philosophy: minimise discrimination, ensure accuracy, prevent inaccurate inferences |

```yaml
# Explanation packet sent to a refused applicant
decision_id: dec_2c4e91
type: credit_underwriting
outcome: refused
inputs_considered:
  - income_band: 30k-40k
  - existing_debt_to_income: 0.42
  - missed_payments_24mo: 2
  - employment_tenure_months: 5
factor_importance:           # SHAP-style top contributors
  - missed_payments_24mo:    0.41
  - employment_tenure_months: 0.27
  - debt_to_income:          0.18
how_to_challenge: "Reply within 14 days requesting human review. Provide updated information."
human_review_contact: review@acme.example
```

## Logging & Monitoring with GDPR

Logs are personal data when they contain identifiers — IPs (CJEU C-582/14 Breyer confirmed dynamic IPs), user IDs, emails, session tokens, device fingerprints. Logging is processing. Need a lawful basis (almost always Art. 6(1)(f) legitimate interest in security/abuse prevention/network operation), record in RoPA, set retention.

| Log type | Recommended retention | Notes |
|----------|------------------------|-------|
| Access / audit logs | 6–12 months | Security investigations |
| Application / debug | 14–30 days | Pseudonymise user IDs after 30 days |
| Web access | 30 days raw, then aggregate | Anonymise IPs at ingestion |
| Security alerts | 12 months | SIEM |
| DSR requests | Retention as long as user account + 3 years | Demonstrate compliance |
| Consent log | Lifetime of relationship + 3 years | Demonstrate consent (Art. 7(1)) |

```python
# Pseudonymise / anonymise at ingestion
def ip_anon(ip: str) -> str:
    if ":" in ip:                         # IPv6 -> /48
        return ipaddress.ip_network(f"{ip}/48", strict=False).network_address.compressed
    return ipaddress.ip_network(f"{ip}/24", strict=False).network_address.compressed

# Strip secrets and PII before logging
SECRET_KEYS = {"password","authorization","cookie","x-api-key","token",
               "secret","ssn","cc","cvv","dob","health","ip"}

def safe_log(event: dict) -> dict:
    return {k: ("[REDACTED]" if k.lower() in SECRET_KEYS else v)
            for k, v in event.items()}
```

| Engineering rule | Reason |
|------------------|--------|
| Never log secrets | Authorization headers, cookies, password resets, OTPs — all attack surface |
| Never log payloads with PII verbatim | Especially special-category |
| Anonymise IPs at ingestion | /24 IPv4, /48 IPv6, or hash with rotating salt |
| Drop request bodies for PII endpoints | /signup, /me, /payment |
| Set TTL on every log index | Otherwise retention drifts to forever |
| Centralise + RBAC | Only auditable, named admins access logs |
| Encrypt logs at rest + in transit | TLS to log sink + SSE on storage |
| Document retention + lawful basis | RoPA entry per log system |

```nginx
# Nginx: anonymise IP at the source (truncate /24)
map $remote_addr $remote_addr_anon {
    "~(?<a>\d+\.\d+\.\d+)\.\d+"  "$a.0";
    default                      "0.0.0.0";
}
log_format anon '$remote_addr_anon - $remote_user [$time_local] '
                '"$request" $status $body_bytes_sent';
access_log /var/log/nginx/access.log anon;
```

```yaml
# OpenSearch index template with retention
index_patterns: [logs-*]
template:
  settings:
    index.lifecycle.name: 30d-rolling
    index.lifecycle.rollover_alias: logs
ilm_policy:
  hot:    {min_age: 0d, actions: {rollover: {max_size: 50gb}}}
  warm:   {min_age: 7d}
  delete: {min_age: 30d, actions: {delete: {}}}
```

## Email & Marketing

Marketing email is governed by GDPR (lawful basis + transparency) AND ePrivacy (PECR in UK, LIL Title I in France) which often imposes consent independently. The two stack: even if you have a legitimate interest under GDPR, ePrivacy may still require prior consent for unsolicited electronic marketing.

| Scenario | UK PECR / ePrivacy | GDPR basis |
|----------|---------------------|------------|
| Cold B2C email | Consent required | Consent |
| Cold B2B email | Lighter (corporate subscribers) | Legitimate interest possible |
| Soft opt-in (existing customer, similar products) | Allowed; opt-out at every contact | Legitimate interest |
| Transactional (order confirmation, shipping) | Not direct marketing | Contract |
| Service / safety announcements | Not direct marketing | Contract / legal obligation / LI |

| Required in every marketing email |
|-----------------------------------|
| Identifiable sender (legal name, address) |
| Working unsubscribe link (one click) |
| Honour unsubscribe within X days (PECR: not specified, in practice 24–72h) |
| Suppression list (never re-email after unsubscribe except by re-consent) |

```python
# Send-time check
def can_email(user) -> bool:
    if user.email_unsubscribed_at:           return False
    if user.bounced:                          return False
    if user.suppressed:                       return False
    if not user.consent.marketing and not user.soft_optin_eligible:
        return False
    return True
```

```html
<!-- Footer -->
<p>You are receiving this because you signed up at acme.example on 2023-08-12.
Acme Ltd, 1 Square Mile, London EC1, UK. DPO: dpo@acme.example.
<a href="{{unsubscribe_url}}">Unsubscribe</a> instantly.</p>
```

| List-Unsubscribe header (RFC 8058) | Required in 2024 by Gmail/Yahoo for senders >5k/day |
|------------------------------------|------------------------------------------------------|
| `List-Unsubscribe: <https://acme.example/u/abc>, <mailto:unsub@acme.example>` | One-click |
| `List-Unsubscribe-Post: List-Unsubscribe=One-Click` | Mailbox-provider triggered |

```python
# Idempotent unsubscribe endpoint - HMAC token, no auth required
import hmac, hashlib, base64, time

def make_token(user_id: str, secret: bytes) -> str:
    msg = f"{user_id}|{int(time.time()//86400)}".encode()
    sig = hmac.new(secret, msg, hashlib.sha256).digest()[:16]
    return base64.urlsafe_b64encode(msg + b"|" + sig).decode()

def unsubscribe(token: str) -> Response:
    user_id, day, sig = decode_check(token, SECRET)
    db.execute("UPDATE users SET marketing_optin=false, unsubscribed_at=now() "
               "WHERE id = ?", (user_id,))
    return ok("Unsubscribed.")
```

## Common Compliance Mistakes

| Mistake | Why it fails | Fix |
|---------|--------------|-----|
| Pre-ticked consent boxes | Not unambiguous (CJEU C-673/17) | Unticked checkbox; user must tick |
| Click-anywhere-to-consent | Not clear affirmative action | Explicit button or checkbox |
| Bundled consent | Not specific | Granular per purpose |
| "Continued use = consent" | Not affirmative | Real opt-in |
| Cookie wall | Not freely given | Free path with reject all |
| Missing privacy notice at point of collection | Art. 13 breach | Just-in-time + full notice |
| One privacy notice for everything | Not transparent for distinct activities | Layered + linked |
| Consent for legitimate-interest processing | Confusing & not freely given | Pick one basis per processing |
| No DPIA for high-risk processing | Art. 35 breach | Run DPIA before launch |
| Logging password / token in cleartext | Confidentiality + minimisation | Redact at the source |
| Logging full IP forever | No purpose limitation, no minimisation | Anonymise + retention TTL |
| US-cloud data transfer without SCC + TIA | Art. 46 breach (Schrems II) | SCCs + TIA + supplementary measures |
| Treating Stripe / AWS as just a vendor | DPA missing | Sign DPA with every processor |
| No sub-processor list / no notice mechanism | Art. 28(2) breach | Public list + change notification |
| No DSR queue / DSR via email only | Cannot meet 1-month SLA | Self-service + tracked queue |
| Soft-delete forever | Storage limitation breach | Hard-delete on schedule |
| Backups containing erased users with no plan | Cannot prove erasure | Excludelist + re-erase on restore |
| Late breach notification (>72h) | Art. 33 breach | Pre-built template + runbook |
| No DPO when required | Art. 37 breach | Appoint or engage external |
| Children under threshold without parental consent | Art. 8 breach | Age gate + parental flow |
| Profiling for ads on children | EDPB position | Off by default for under-18 |
| Free text "race / health / religion" fields without Art. 9 basis | Special-category breach | Don't collect or get explicit consent |
| Same-purpose marketing without unsubscribe | PECR + Art. 21(2) | Unsubscribe link |
| Storing geolocation indefinitely | Storage limitation | Retention + reduce precision |

## Engineering Checklist

Foundations
- [ ] Records of Processing (RoPA) maintained as code, reviewed quarterly
- [ ] Privacy notice published, layered, versioned in repo
- [ ] DPO designated and contactable; published in privacy notice
- [ ] Lead supervisory authority identified (one-stop-shop)
- [ ] Data flow diagram covering all systems and sub-processors
- [ ] Lawful basis documented for every processing activity
- [ ] Special-category Art. 9 basis documented where applicable
- [ ] Children's age gate per member-state threshold

Consent & Notice
- [ ] Consent banner with parity for accept/reject
- [ ] No non-essential cookies/scripts before consent
- [ ] Granular consent per purpose
- [ ] Consent log: timestamp, version, IP-anon, UA, method
- [ ] Withdrawal as easy as granting — single-click
- [ ] Privacy notice has Art. 13/14 content fully covered
- [ ] Material-change re-consent flow exists
- [ ] Just-in-time notice at every collection point

Data Subject Rights
- [ ] Self-service "download my data" endpoint
- [ ] Self-service profile edit + correction request
- [ ] Erasure request endpoint with cascade pipeline
- [ ] Restriction (freeze) flag implemented
- [ ] Portability export in machine-readable format with manifest
- [ ] Object opt-out for direct marketing — absolute
- [ ] Object opt-out for legitimate-interest processing
- [ ] Automated-decision opt-out + human review path
- [ ] DSR queue with 1-month SLA tracking
- [ ] Identity verification proportionate
- [ ] Audit log for every DSR
- [ ] Notify recipients on rectification / erasure / restriction (Art. 19)

Erasure Cascade
- [ ] Primary DB delete
- [ ] Read replicas via replication
- [ ] Data warehouse delete
- [ ] Logs pseudonymise / delete by user
- [ ] Backups: excludelist + re-erase on restore
- [ ] Search indexes delete-by-query
- [ ] Caches invalidate / let TTL expire
- [ ] Object storage delete
- [ ] CDN purge
- [ ] Mailing list delete + suppression
- [ ] CRM/helpdesk anonymise + delete contact
- [ ] Sub-processor erasure API call
- [ ] Sentry / error trackers scrub
- [ ] Vector DB / embeddings delete
- [ ] Idempotent re-runnable job

Privacy by Design / Default
- [ ] Default-private settings on new accounts
- [ ] Optional fields off by default
- [ ] Field-level encryption for special-category data
- [ ] Tokenisation for payment data
- [ ] Pseudonymisation at rest in non-auth tables
- [ ] Encryption at rest everywhere
- [ ] TLS 1.2+ in transit (1.3 preferred)
- [ ] RBAC with least privilege
- [ ] MFA on admin / sensitive ops
- [ ] Audit logs for sensitive reads/writes
- [ ] Retention schedules in DB schema
- [ ] Auto-purge jobs tested and monitored

Transfers
- [ ] Map of every international transfer
- [ ] Adequacy / SCCs / BCRs / DPF identified per destination
- [ ] Transfer Impact Assessment for non-adequate destinations
- [ ] Supplementary measures (encryption, EU-controlled keys)
- [ ] Data residency options for customers who require EU-only

Processors / Sub-processors
- [ ] DPA signed with every processor
- [ ] TOMs schedule attached to each DPA
- [ ] Sub-processor list public
- [ ] Notice + objection mechanism for sub-processor changes
- [ ] Annual diligence per sub-processor
- [ ] SOC2 / ISO27001 / equivalent on file

Breach
- [ ] Incident response runbook with breach-notification path
- [ ] Pre-filled SA notification templates
- [ ] DPO + Legal + Comms paging chain
- [ ] Breach register (Art. 33(5))
- [ ] Subject-notification template
- [ ] Annual tabletop exercise
- [ ] Detection: alerts on data exfiltration, unusual access, ransomware
- [ ] 72h SLA process documented

Logging & Security
- [ ] No secrets in logs
- [ ] No PII payloads in logs
- [ ] IP anonymised at ingestion
- [ ] Retention TTL on every log index
- [ ] Encrypted log storage
- [ ] Centralised access with RBAC

DPIA
- [ ] DPIA for any high-risk new processing
- [ ] DPIA template in repo
- [ ] Sign-off process: controller + DPO
- [ ] Prior consultation (Art. 36) when residual risk high
- [ ] Annual DPIA review

Children
- [ ] Age gate with member-state thresholds
- [ ] Parental-consent flow
- [ ] No profiling for ads to under-18s
- [ ] Plain-language privacy notice for children where applicable

Marketing
- [ ] Unsubscribe link in every marketing email
- [ ] Suppression list honoured
- [ ] Identifiable sender + postal address
- [ ] Soft-opt-in only for existing customers + similar products
- [ ] Cold B2C only with consent

Vendor / Third-party
- [ ] Vendor onboarding includes DPA + DPIA-lite + transfer mechanism
- [ ] Annual vendor re-attestation
- [ ] Off-boarding: data return / certified destruction

Documentation
- [ ] All policies in version control
- [ ] Annual policy review
- [ ] Training: all staff annually, engineers biannually
- [ ] Evidence repo for regulator (TOMs, certs, audits, training records)

## Common Gotchas

**IP addresses are not personal data.**
- WRONG: "We log full IPs forever, they're just numbers."
- RIGHT: CJEU C-582/14 Breyer (2016) confirmed dynamic IPs are personal data when the controller can reasonably link them to a person (via ISP, account, etc.). Anonymise at ingestion (/24 IPv4, /48 IPv6) or hash with rotating salt; set retention.

**Anonymising means removing the name.**
- WRONG: `DELETE name, email FROM users WHERE id = ?` and treating the row as anonymous.
- RIGHT: Anonymisation requires preventing singling out, linkability, and inference. The remaining row (DOB + ZIP + behaviour + device fingerprint) is still personal data per WP29 WP216. True anonymisation is hard; what most teams do is pseudonymisation — still in scope.

**Backups are exempt.**
- WRONG: "We can't delete from backups, so erasure requests don't apply to backups."
- RIGHT: Backups contain personal data and are in scope. Accepted approach: encrypt backups, restrict access, document retention, auto-expire, and on any restore re-run the erasure pipeline against the restored data using an excludelist of erased subjects.

**Consent or leave is consent.**
- WRONG: "By using this service you consent to our processing."
- RIGHT: Not freely given (Art. 4(11), Art. 7(4), Recital 42). Service availability conditional on consent for non-necessary processing fails. Cookie walls invalidated by EDPB Opinion 8/2024 and CNIL Deliberation 2020-091.

**We have a DPA so we're fine to send data to our US vendor.**
- WRONG: DPA covers Art. 28; transfers are governed by Chapter V.
- RIGHT: Need adequacy (DPF), SCCs (2021), or BCRs *plus* a transfer impact assessment per Schrems II (CJEU C-311/18). DPA alone is insufficient.

**Legitimate interest covers everything.**
- WRONG: "We rely on legitimate interest for marketing, analytics, profiling, sharing with partners."
- RIGHT: LIA required (purpose / necessity / balancing). For ad-tech profiling, CJEU and EDPB have largely rejected LI; consent is needed. Direct marketing has Art. 21(2) absolute object right. For ePrivacy-covered tracking, ePrivacy may require consent regardless.

**Pseudonymisation = anonymisation.**
- WRONG: "We hashed the user IDs, they're anonymous now."
- RIGHT: Pseudonymisation (Art. 4(5)) is reversible with the additional information; still personal data. Hashing with a known scheme over a finite identifier space (emails, phone numbers) is trivially brute-forceable.

**Right to erasure is absolute.**
- WRONG: "User asked to be erased; delete everything immediately, including the audit log of the erasure request."
- RIGHT: Art. 17(3) exceptions — legal obligation (tax, AML), legal claims, public interest in health, archiving. The erasure request itself + accountability data may be retained under Art. 6(1)(c) / 6(1)(f). Don't delete the proof you complied.

**One privacy notice for the whole product.**
- WRONG: "Generic 30,000-word notice covering everything."
- RIGHT: Layered + just-in-time. Specific notice at the point of collection per Art. 13. Distinct activities (marketing vs core service) have distinct purposes and should be transparent at the level the user can act on.

**Legitimate interest means I never need consent.**
- WRONG: Skipping the LIA and never reviewing.
- RIGHT: Three-part test in writing; revisit when the processing changes; data subject's reasonable expectations (Recital 47) matter; Art. 21 object right is always available.

**72-hour breach clock starts when investigation is complete.**
- WRONG: "We'll know after the forensics in 2 weeks."
- RIGHT: Clock starts at *awareness* of the breach (Art. 33(1)), measured wall-clock incl. weekends. Notify with what you know and update; late notifications draw fines independently.

**B2B email is GDPR-exempt.**
- WRONG: "Corporate email = no personal data."
- RIGHT: `john.smith@acme.example` identifies a natural person. Marketing rules under PECR/ePrivacy may be more permissive for B2B subscribers, but GDPR still applies — lawful basis, transparency, rights.

**Encryption alone solves Schrems II.**
- WRONG: "We use TLS so US transfers are fine."
- RIGHT: TIA must analyse government access laws (FISA 702, EO 12333). Supplementary measures need to render data unintelligible to the importer's authorities — typically EU-controlled keys with no plaintext access by importer.

**Soft delete is the same as deletion.**
- WRONG: `UPDATE users SET deleted_at = now()` and stop.
- RIGHT: Soft-delete is fine as a phase; the row must be physically deleted or fully anonymised on a documented schedule. Storage limitation (Art. 5(1)(e)) requires actual deletion.

**Legitimate interest balancing is "we win because business."**
- WRONG: Three-line balancing test approving anything.
- RIGHT: Document data subject's reasonable expectations, vulnerability, less-intrusive alternatives, and safeguards. Recital 47 + EDPB Opinion 06/2014.

**Records of Processing are optional under 250 employees.**
- WRONG: "We have 50 staff so Art. 30 doesn't apply."
- RIGHT: The Art. 30(5) exemption is narrow: only if processing is occasional, no special-category, no Art. 10, and no risks. Almost no SaaS qualifies because user-account processing is not "occasional".

**Data subject access can be refused if it's hard.**
- WRONG: "Pulling all this data is too expensive."
- RIGHT: Manifestly unfounded / excessive (Art. 12(5)) is a high bar. Cost is your responsibility; you must engineer the system so DSARs are routine.

**Privacy notice content can be in the T&Cs.**
- WRONG: "Section 14 of our 18-page Terms covers privacy."
- RIGHT: Art. 12(1) requires transparent, intelligible, easily accessible. Burying it fails.

## Idioms

| Idiom | Meaning |
|-------|---------|
| Minimise collection | Optional fields off by default; collect at the moment of use |
| Delete by default | Retention TTL on every store; opt-in to keep |
| Log purpose | Every field annotated with the purposes it serves |
| Encrypt by default | TLS everywhere, AES-256 at rest, field-level for special-category |
| Consent UI cannot be a dark pattern | Reject-all parity; one-click withdrawal; granular choices |
| DPO is independent advisor | Reports to top management; cannot be ordered to change opinion |
| One basis per processing | Pick one Art. 6 basis; don't mix consent and LI on the same activity |
| Default-private | New accounts, posts, profiles start private; users opt up |
| Document the *why* | Every processing has a purpose recorded — the purpose is the contract |
| Backups are not exempt | They're personal data with retention; engineer for restore + re-erase |
| Pseudonymise, then encrypt | Two layers of TOMs; reduces breach risk |
| Treat IP like a name | Anonymise at ingestion |
| Cascade deletion is a pipeline, not a SQL | Many systems, idempotent steps |
| Three-step LIA | Purpose, necessity, balancing — written down |
| 72 hours is wall-clock | Plan runbook accordingly |
| Just-in-time notice beats long policy | Tell users what's happening when it happens |
| Layered notice | Headline + summary + detail |
| Transfers need both basis and mechanism | Art. 6 and Chapter V |
| Vendor diligence is annual | Re-attest, not set-and-forget |
| Children get more protection | Default no-profiling, plain language, age gate |
| Records as code | YAML/JSON in git, diff-able, reviewable |
| Test the runbook | Annual breach + DSR tabletop |

## See Also

- ccpa
- license-decoder
- spdx-identifiers
- gpg
- age
- vault
- sops

| Related cheat sheet | Why |
|---------------------|-----|
| ccpa | California parallel; many overlapping engineering controls |
| license-decoder | Licensing intersects with DPA / sub-processor obligations |
| spdx-identifiers | SBOM / supply chain hygiene tied to TOMs |
| gpg | Encryption for transfer / supplementary measures |
| age | Modern file-encryption tool for cold storage |
| vault | Secrets management / KMS pattern for field-level encryption |
| sops | Encrypted-config workflow for repos with PII config |

## References

- Regulation (EU) 2016/679 — https://eur-lex.europa.eu/eli/reg/2016/679/oj
- European Data Protection Board — https://edpb.europa.eu
- gdpr.eu — https://gdpr.eu
- Information Commissioner's Office (UK) — https://ico.org.uk
- Commission Nationale de l'Informatique et des Libertés (FR) — https://www.cnil.fr
- EDPB Guidelines — https://edpb.europa.eu/our-work-tools/general-guidance_en
  - Guidelines 05/2020 on consent
  - Guidelines 03/2017 on transparency
  - Guidelines 02/2023 on Art. 6(1)(b)
  - Guidelines 4/2019 on Art. 25 by design and default
  - Guidelines 9/2022 on personal data breach notification
  - Guidelines 07/2020 on controller and processor concepts
  - Guidelines 08/2020 on targeting of social media users
  - Guidelines 01/2024 on consent or pay
  - Guidelines on Automated decision-making and Profiling (WP251rev.01)
- Schrems II — CJEU C-311/18 — https://curia.europa.eu/juris/document/document.jsf?docid=228677
- Schrems I — CJEU C-362/14
- Breyer (dynamic IPs) — CJEU C-582/14
- Planet49 (cookie consent) — CJEU C-673/17
- Bundeskartellamt (Meta / contract basis) — CJEU C-252/21
- Fashion ID (joint controllers / Like button) — CJEU C-40/17
- WP29 Opinion 05/2014 on anonymisation (WP216)
- WP29 Guidelines on Data Portability (WP242)
- NIST Privacy Framework — https://www.nist.gov/privacy-framework
- ISO/IEC 27701:2019 — privacy information management system
- EU-US Data Privacy Framework — https://www.dataprivacyframework.gov
- 2021 Standard Contractual Clauses — Implementing Decision (EU) 2021/914
- ePrivacy Directive 2002/58/EC (as amended)
- noyb.eu — civil-society enforcement organisation
- enforcementtracker.com — community-maintained enforcement database
- gdprhub.eu — case-law wiki
- ICO Age Appropriate Design Code — https://ico.org.uk/for-organisations/uk-gdpr-guidance-and-resources/childrens-information/childrens-code-guidance-and-resources/age-appropriate-design-a-code-of-practice-for-online-services
- ENISA Privacy and Data Protection by Design Handbook — https://www.enisa.europa.eu/publications/privacy-and-data-protection-by-design
- IAPP — https://iapp.org (training, news, summits, certifications CIPP/E, CIPM, CIPT)
- Data Transfer Project — https://datatransferproject.dev
- UK GDPR (Data Protection Act 2018) — https://www.legislation.gov.uk/ukpga/2018/12/contents
- Brazilian LGPD (Lei 13.709) — close GDPR analogue if you operate in Brazil
- ISO/IEC 29100:2011 — Privacy framework
- ISO/IEC 27018:2019 — Cloud-PII processor controls
