# Security Awareness (Program Design, Training, and Culture)

Practical reference for building, delivering, and measuring security awareness programs including phishing simulation, role-based training, compliance requirements, and security culture development.

## Program Design

### Awareness Program Framework

```text
Program Lifecycle:
  1. Assess    -- baseline knowledge, current culture, risk areas
  2. Plan      -- objectives, audience segmentation, content strategy
  3. Develop   -- create/curate content, build simulations
  4. Deliver   -- multi-channel deployment (email, LMS, posters, Slack)
  5. Measure   -- metrics collection, behavior analytics
  6. Improve   -- adjust based on data, refresh content quarterly

Maturity Levels:
  Level 1: Compliance-focused   -- annual checkbox training
  Level 2: Awareness-focused    -- quarterly campaigns, basic phishing sims
  Level 3: Behavior-focused     -- continuous reinforcement, targeted training
  Level 4: Culture-focused      -- security champions, peer coaching, embedded habits
  Level 5: Metrics-driven       -- predictive analytics, risk-based prioritization
```

### Audience Segmentation

```text
Role-Based Groups:
  General staff         -- phishing, passwords, physical security, data handling
  Developers            -- secure coding (OWASP Top 10), secrets management, SAST/DAST
  IT administrators     -- privilege escalation, patch management, incident response
  Executives / C-suite  -- BEC awareness, strategic risk, regulatory liability
  Finance / accounting  -- wire fraud, invoice scams, payment verification
  HR / recruiting       -- social engineering, PII handling, background checks
  Customer support      -- pretexting, account takeover, caller verification
  Remote workers        -- VPN usage, home network security, physical workspace
  Third-party / vendors -- data handling policies, access restrictions, NDA requirements
```

## Training Delivery Methods

### Computer-Based Training (CBT)

```text
LMS Platforms:
  KnowBe4              -- largest library, automated campaigns, PhishER
  Proofpoint SAT        -- threat-informed content, CyberStrength assessments
  SANS Security Awareness -- developer-focused, hands-on labs
  Cofense               -- phishing-centric, reporter button integration
  Infosec IQ            -- gamified, role-based paths
  Ninjio                -- short animated episodes (3-4 min)

CBT Best Practices:
  Duration              -- 5-15 min modules (micro-learning preferred)
  Frequency             -- monthly or biweekly modules, not annual dumps
  Format                -- video + interactive quiz + scenario
  Language              -- plain English, avoid jargon, localize for global orgs
  Accessibility         -- WCAG 2.1 compliant, captions, screen reader support
  Completion tracking   -- LMS integration with SCORM/xAPI/cmi5
```

### Classroom and Live Training

```text
Instructor-Led Training (ILT):
  Onboarding sessions    -- 30-60 min for new hires (within first week)
  Lunch-and-learn        -- 20-30 min monthly, voluntary, pizza helps
  Tabletop exercises     -- role-play incident scenarios (ransomware, breach, BEC)
  Workshops              -- hands-on (e.g., password manager setup, MFA enrollment)
  Town halls             -- CISO presents threat landscape, Q&A

Virtual ILT (vILT):
  Platforms              -- Zoom, Teams, WebEx with breakout rooms
  Engagement             -- polls, chat, interactive scenarios
  Recording              -- archive for async viewing, but live preferred
```

### Gamification

```text
Gamification Techniques:
  Points / badges        -- award for completing modules, reporting phish
  Leaderboards           -- department competition (anonymize individual scores)
  Capture the Flag (CTF) -- security challenges for technical staff
  Escape rooms           -- physical or virtual, team-based security puzzles
  Simulations            -- interactive branching scenarios (choose-your-adventure)
  Rewards                -- gift cards, swag, extra PTO for top performers

Platforms with Gamification:
  KnowBe4               -- points, badges, leaderboards, Kevin Mitnick modules
  Hoxhunt                -- adaptive gamified phishing simulations
  CybSafe                -- behavioral science + gamification
  Living Security        -- immersive gamified training
```

## Phishing Simulation

### GoPhish (Open Source)

```bash
# Install GoPhish
wget https://github.com/gophish/gophish/releases/latest/download/gophish-v0.12.1-linux-64bit.zip
unzip gophish-v0.12.1-linux-64bit.zip -d /opt/gophish
cd /opt/gophish

# Edit config.json -- change admin listen to 0.0.0.0:3333, phish to 0.0.0.0:8080
# Set admin password on first login

# Start GoPhish
./gophish

# GoPhish workflow:
#   1. Create Sending Profile    -- SMTP server config (use internal relay)
#   2. Create Landing Page       -- clone real login page or build custom
#   3. Create Email Template     -- phishing email with {{.URL}} tracking link
#   4. Create User Group         -- import CSV (First,Last,Email,Position)
#   5. Launch Campaign           -- schedule send, set tracking
#   6. Review Results            -- opened, clicked, submitted data, reported

# Import users from CSV
# CSV format: First Name,Last Name,Email,Position
# Upload via Groups > New Group > Bulk Import
```

### KnowBe4 Phishing Simulation

```text
Campaign Types:
  Basic phishing         -- credential harvesting simulation
  Spear phishing         -- targeted with personal details (name, role)
  Vishing                -- voice phishing simulation (optional add-on)
  Smishing               -- SMS phishing simulation
  USB drop               -- physical USB key test (QR code variant)
  Callback phishing      -- email with phone number to call

Difficulty Levels:
  1-star                 -- obvious (Nigerian prince, broken English)
  2-star                 -- moderate (generic IT notice)
  3-star                 -- realistic (vendor invoice, HR benefits)
  4-star                 -- advanced (CFO wire transfer, M&A-related)
  5-star                 -- APT-level (uses OSINT, mimics real vendors)

Phish Alert Button (PAB):
  Outlook / O365 plugin  -- one-click report suspected phishing
  Gmail plugin           -- Chrome extension or Google Workspace add-on
  Integration            -- auto-forwards to SOC mailbox + KnowBe4 console
```

### Simulation Best Practices

```text
Frequency:              Monthly or biweekly simulations
Targeting:              Rotate templates, vary difficulty, seasonal themes
Timing:                 Vary send times (morning, afternoon, different days)
Consequences:           Never punish -- assign remedial training on failure
Follow-up:              Immediate teachable moment on click (landing page education)
Escalation:             Repeat clickers get 1:1 coaching, not HR action
Legal review:           Get legal/HR approval for simulation program
Exclusions:             Exclude during layoffs, crises, or sensitive periods
Reporting:              Track click rates by department, role, simulation difficulty
Baseline:               Run initial campaign before any training to establish baseline
```

## Social Engineering Awareness

### Attack Types to Cover

```text
Phishing                -- email-based credential theft or malware delivery
Spear phishing          -- targeted phishing with personal context
Whaling                 -- executive-targeted phishing (CEO fraud, BEC)
Vishing                 -- voice/phone-based social engineering
Smishing                -- SMS-based phishing
Pretexting              -- creating a fabricated scenario to gain trust
Baiting                 -- leaving infected USB drives, offering free items
Tailgating/piggybacking -- following authorized person through secured door
Quid pro quo            -- offering something in exchange (fake IT support)
Watering hole           -- compromising websites frequently visited by targets
Dumpster diving         -- searching trash for sensitive information
Shoulder surfing        -- watching someone enter credentials
Deepfake                -- AI-generated audio/video impersonation
Callback phishing       -- email directing victim to call attacker
```

### Red Flags Training

```text
Email Red Flags:
  - Urgency / pressure to act immediately
  - Unusual sender address (misspelled domain, external where internal expected)
  - Generic greeting ("Dear Customer" instead of name)
  - Grammar/spelling errors (less reliable with AI-generated content)
  - Mismatched URLs (hover to verify before clicking)
  - Unexpected attachments (especially .zip, .docm, .exe, .iso, .html)
  - Requests for credentials, payment, or sensitive data
  - "Do not share with anyone" / secrecy pressure
  - Reply-to differs from sender address

Phone Red Flags:
  - Caller ID spoofing (verify by calling back on known number)
  - Pressure to bypass normal procedures
  - Requests for remote access (TeamViewer, AnyDesk)
  - "I'm from IT" without verification
  - Threats of account lockout or legal action

Physical Red Flags:
  - Unknown person requesting access ("I'm the vendor")
  - Tailgating through badge-controlled doors
  - Unattended USB drives or devices
  - Unsolicited deliveries requesting signature + information
```

## Security Culture Assessment

### Measurement Approaches

```text
Assessment Methods:
  Surveys                -- security culture survey (CLTRe, SANS, custom)
  Behavioral metrics     -- phishing sim results, reporting rates, policy violations
  Observation            -- desk audits (clean desk), tailgating tests
  Interviews             -- focus groups with departments
  Incident analysis      -- root cause human factors in incidents

Culture Dimensions (CLTRe Model):
  1. Attitudes           -- how employees feel about security
  2. Behaviors           -- what employees actually do
  3. Cognition           -- what employees know about security
  4. Communication       -- how security info flows in the organization
  5. Compliance          -- adherence to policies and procedures
  6. Norms               -- unwritten security expectations
  7. Responsibilities    -- ownership of security tasks
```

## Metrics and KPIs

### Core Metrics

```text
Phishing Metrics:
  Click rate             -- % who clicked phishing sim link (target: <5%)
  Report rate            -- % who reported via PAB (target: >70%)
  Susceptibility rate    -- % who submitted data on landing page
  Time to report         -- median minutes from send to first report
  Repeat clicker rate    -- % who fail multiple simulations
  Click-to-report ratio  -- reports / clicks (higher is better)

Training Metrics:
  Completion rate        -- % who completed assigned training (target: >95%)
  Assessment scores      -- pre/post knowledge test improvement
  Time to complete       -- average training duration
  Satisfaction scores    -- training quality ratings (Likert scale)
  Retention rate         -- knowledge retention at 30/60/90 days

Behavioral Metrics:
  Password policy compliance    -- % meeting complexity/rotation requirements
  MFA adoption                  -- % enrolled in MFA
  Incident reporting rate       -- security incidents reported per month
  Clean desk compliance         -- % passing random desk audits
  Shadow IT reduction           -- unauthorized app/service usage trends

ROI Metrics:
  Cost per employee             -- total program cost / headcount
  Risk reduction                -- decrease in successful attacks
  Incident cost avoidance       -- estimated losses prevented
  Compliance penalty avoidance  -- fines/sanctions avoided
```

### Reporting Dashboard

```text
Monthly Report Template:
  +--------------------------------------------+
  | SECURITY AWARENESS DASHBOARD - March 2026  |
  +--------------------------------------------+
  | Phishing Sim Results                       |
  |   Sent: 2,500  Clicked: 87 (3.5%)         |
  |   Reported: 1,842 (73.7%)                 |
  |   Data submitted: 12 (0.5%)               |
  +--------------------------------------------+
  | Training Completion                        |
  |   Q1 Module: 97.2% complete               |
  |   New hire onboarding: 100%               |
  |   Developer secure coding: 94.1%          |
  +--------------------------------------------+
  | Trend: Click rate down 2.1% from Q4       |
  | Action: Target Finance dept (8.2% click)  |
  +--------------------------------------------+
```

## Compliance Training Requirements

### Regulatory Requirements

```text
HIPAA (Healthcare):
  Required:    Annual security awareness training for all workforce members
  Topics:      PHI handling, minimum necessary, breach reporting, device security
  Evidence:    Training logs, signed acknowledgment, quiz scores
  Reference:   45 CFR 164.308(a)(5)(i)

PCI-DSS (Payment Cards):
  Required:    Annual training for personnel with access to cardholder data
  Topics:      CHD handling, PCI requirements, incident reporting
  Evidence:    Training records, policy acknowledgment
  Reference:   PCI-DSS v4.0 Requirement 12.6

SOX (Financial Reporting):
  Required:    Training on internal controls, fraud awareness
  Topics:      Financial reporting integrity, whistleblower protections
  Evidence:    Training completion records, attestations
  Reference:   SOX Section 302/404

GLBA (Financial Services):
  Required:    Training on safeguarding customer financial information
  Topics:      NPI handling, pretexting awareness, disposal procedures
  Reference:   16 CFR 314 (Safeguards Rule)

FERPA (Education):
  Required:    Training on student record privacy
  Topics:      Directory information, FERPA rights, disclosure rules
  Reference:   34 CFR Part 99

GDPR (EU Data Protection):
  Required:    Appropriate training for data handlers (Art. 39(1)(b))
  Topics:      Lawful basis, data subject rights, breach notification, DPO role
  Evidence:    Training records, DPIAs showing staff competence

CMMC (Defense Contractors):
  Required:    Role-based security training (AT.2.056, AT.2.057)
  Topics:      CUI handling, insider threat, social engineering
  Reference:   NIST SP 800-171 3.2.1/3.2.2
```

## Awareness Topics Checklist

### Core Topics by Category

```text
Passwords & Authentication:
  [ ] Password manager usage (1Password, Bitwarden, KeePass)
  [ ] Passkeys and passwordless authentication
  [ ] MFA enrollment and backup codes
  [ ] Password sharing prohibition
  [ ] Credential stuffing awareness

Phishing & Social Engineering:
  [ ] Email phishing identification
  [ ] Spear phishing and BEC awareness
  [ ] Vishing and smishing
  [ ] QR code phishing (quishing)
  [ ] Deepfake awareness

Social Media & Online Presence:
  [ ] Oversharing risks (OSINT fodder)
  [ ] Privacy settings review
  [ ] Corporate social media policy
  [ ] LinkedIn connection requests from unknowns
  [ ] Personal vs corporate device separation

Physical Security:
  [ ] Badge/access card usage
  [ ] Tailgating prevention
  [ ] Clean desk policy
  [ ] Visitor escort requirements
  [ ] Secure printing (pull printing)
  [ ] Device theft prevention (laptop locks, tracking)

Data Handling:
  [ ] Data classification levels (public, internal, confidential, restricted)
  [ ] Encryption requirements (at rest, in transit)
  [ ] Secure file sharing (approved platforms only)
  [ ] Data retention and disposal
  [ ] Removable media policy
  [ ] Cloud storage policy (no personal Dropbox/Drive for work data)

Remote Work:
  [ ] VPN usage requirements
  [ ] Home Wi-Fi security (WPA3, unique password)
  [ ] Screen privacy in public spaces
  [ ] Video call security (waiting rooms, screen sharing awareness)
  [ ] Physical workspace security at home

Incident Reporting:
  [ ] How to report (email, hotline, ticketing system)
  [ ] What to report (suspicious emails, unauthorized access, lost devices)
  [ ] No-blame reporting culture
  [ ] Timely reporting expectations
```

## Continuous Improvement

### Program Enhancement Cycle

```text
Quarterly Activities:
  - Review phishing simulation metrics and trends
  - Update training content with current threat intelligence
  - Rotate simulation templates and difficulty levels
  - Conduct focus groups with high-risk departments
  - Benchmark against industry averages

Annual Activities:
  - Full culture assessment survey
  - Program effectiveness review with executive leadership
  - Budget planning for next year
  - Vendor evaluation and tool assessment
  - Policy review and update
  - Compliance audit preparation

Continuous Activities:
  - Threat intelligence integration (new attack trends into training)
  - Just-in-time training triggers (on policy violation, sim failure)
  - Security champion network engagement
  - Newsletter / Slack channel / intranet updates
  - Recognition and rewards for positive behaviors
```

## See Also

- Supply Chain Security
- Privacy Regulations
- Incident Response

## References

- NIST SP 800-50: Building an IT Security Awareness and Training Program
- NIST SP 800-16: IT Security Training Requirements
- SANS Security Awareness Maturity Model
- KnowBe4 Benchmarking Report
- GoPhish Documentation: https://docs.getgophish.com
- CLTRe Security Culture Framework
