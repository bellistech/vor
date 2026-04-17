# Battle Plan: cs Cheatsheet — iOS Port State & Content Gap Audit
## Convened: 2026-04-15 | Reason: Major milestone review (iOS port shipped, content library assessment)
## Kingdom State: 685 sheets / 685 details / 59 categories / iOS app MVP complete, device-deploy blocked

---

### Situation Report

The `cs` CLI is mature. 685 embedded cheatsheets + 685 deep-dive detail pages span 59 categories and cover 11 elite certifications (CCNP DC/EI, CCIE EI/SP/Sec/Auto, JNCIE-SP/SEC, Linux+, CISSP, C\|RAGE — plus CEH v13 and JNCIA-Junos added in earlier waves). The original 2026-04-05 BATTLE-PLAN identified ~137 cert-gap sheets; those landed in Waves 1–28 (commits `57dba00` through `082ca91`). Additional deep cuts since: HTTP/2-3, DoH/DoT, mDNS, LLDP, PTP, DPDK, AF_XDP, io_uring, SCTP, SR, SD-WAN, network namespaces.

The iOS port — `CsApp/` — shipped to `main` in two commits on 2026-04-13: `4313d33` (Go/gomobile/syntax-highlighting, 37 files) and `3c553f2` (React Native app, 172 files). Phases 0–3 of the iOS plan are complete: native bridge with `getDocumentsDir`/`setDataDir`, BookmarkContext with Set-based persistence, five stack navigators, 8 screens (Categories, TopicList, Sheet, Detail, Search, Starred, Tools, More), shared components (SheetViewer with dark WebView CSS, TopicRow, CategoryRow, Loading/Error views), Chroma monokai syntax highlighting with `WithClasses(false)` inline styles. App runs cleanly on iOS Simulator.

One blocker remains: physical-device install on the user's iPhone (iOS 26.3 beta / Xcode 26.4 beta) dropped the install connection after the ad-hoc signature collision was resolved via `codesign --remove-signature`. Root cause likely in the beta OS's installation daemon, not in the codebase.

---

### The Throne Speaks (Captain — Vision & Strategy)
**Strategic Position**: Single-developer personal learning tool. Not a product for sale — a sharpening stone. 11 elite certs covered with a single binary + an iOS companion app is a rare combination; most commercial flashcard apps cover one vendor at best.

**North Star**: A pocket cheatsheet that reads identically on Mac CLI and iPhone, with the same 685 curated, cross-referenced, math-verified entries. Learning-on-the-go without fumbling for man pages or Cisco docs.

**Key Decision**: Do we pursue TestFlight distribution now or defer until Apple drops the iOS 26 beta install bug? **Recommendation**: Defer device testing one week; if the beta cycle updates and device install still fails, file a Feedback with Apple and pivot to **wireless debugging over local network** as the workaround. Simulator is sufficient for MVP proof.

**Risk to Vision**: Content rot. The 685 sheets will drift as Cisco/Juniper/K8s release new versions. No current cadence exists for re-verification.

---

### The Ledger Records (Micromanager — Execution & QA)
**Sprint Status**: iOS MVP sprint — CLOSED (all 6 planned todos completed).

**Priority Stack** (ordered):
1. **P0** — Unblock device install on iOS 26 beta (wireless debug fallback, deployment target audit, provisioning profile refresh)
2. **P1** — Add 4 high-value content gaps identified below (DORA, EU AI Act, WebAssembly, PQC migration)
3. **P1** — Write a test for the Go→RN markdown bridge (XSS regression, empty-string handling) — markdown_test.go only tests Go side
4. **P2** — Content re-verification cadence: scripted `make verify-sheets` that re-runs the calc `detail/**/*.md` math assertions
5. **P2** — TestFlight upload workflow (archive + altool + App Store Connect) — only after device install works

**QA Gates**:
- `make test` must stay green (Go core + mobile + cscore packages)
- `make fuzz-cscore` for 30s each corpus before any cscore API change
- iOS: simulator smoke (all 8 screens load, star toggles persist across app restart)

**Acceptance Criteria for this cycle**: Battle-plan.md approved. Next Round Table triggered by either (a) successful device install, or (b) first content-gap sheet landing.

---

### The Blueprint Reveals (Architect — Infrastructure & Design)
**Architecture Health**: Strong. Clean layering:
- Go core (`pkg/cscore/`) — platform-independent, 40 functions, all JSON-returning (mobile-safe)
- `mobile/bind.go` — 27 `Mobile*` gomobile wrappers
- `Cscore.xcframework` — pre-compiled arm64 device + simulator
- Swift bridge (`CscoreModule.swift/.m`) — thin RCT_EXTERN_METHOD shims
- TS bridge (`src/core/cscore.ts`) — Promise wrappers
- React Native screens — hook-driven, no duplicated Go logic

**Technical Risks**:
- `Cscore.xcframework` is 100+ MB (untracked in git). Correct — it's a build artifact. Documented in `project_ios_port.md` memory.
- WebView `originWhitelist={['*']}` is permissive. Acceptable since body is always Go-rendered goldmark output (no remote HTML). `markdown_test.go::TestRenderMarkdownToHTML_XSS` confirms 12 attack vectors are escaped at the Go layer.
- No Content-Security-Policy meta tag in `htmlTemplate`. Low risk given the closed input pipeline, but adding one is cheap defense-in-depth.

**Protocol Alignment**: REST API server (stdlib `net/http`) is untouched by the iOS port — good separation. `MobileRenderMarkdownToHTML` shares the exact same goldmark+Chroma pipeline the terminal `glamour` renderer uses stylistically, so CLI and iOS output are visually coherent.

**Recommended Architecture Decisions**:
- **ADR-001**: Commit that Go core is the single source of truth for ALL rendering, calc, search, subnet logic. Swift/TS layers are dumb transports. No TypeScript re-implementation of any Go function, ever.
- **ADR-002**: xcframework rebuild cadence: only on `pkg/cscore/` or `mobile/` changes. Checksum the Go sources; skip rebuild if unchanged.

---

### The Anvil Reports (Developer — Implementation & Testing)
**Code Health**:
- Go test suite: `make test` passes (per CI assumption; last local run clean pre-commit)
- Go lint: `go vet ./...` clean
- `markdown_test.go` covers 7 rendering paths + 12 XSS vectors ✓
- `CsApp/` has **zero unit tests** — gap
- TypeScript: no tsc errors on last build

**Implementation Blockers**:
- None for the Go/CLI side
- iOS device install: `0xe8008014` resolved via signature strip + Embed & Sign; now stuck at "Connection with the remote side was unexpectedly closed" during device copy. Diagnosis points at iOS 26 beta installation daemon, not our code.

**TDD Status**: Green on Go. Red/none on RN. **Action**: Add `CsApp/src/**/*.test.ts` for BookmarkContext reducer (pure function, easy win).

**Estimated Effort**:
- Device install workaround (wireless debug): 30 min
- 4 new content sheets (DORA, EU AI Act, WASM, PQC migration): 4–6 hours each including detail/ page + See Also wiring
- BookmarkContext reducer test: 30 min
- `make verify-sheets` math-assertion runner: 2 hours

---

### The Hourglass Measures (Timeguru — Timeline & Milestones)
**Current Phase**: Post-MVP consolidation. iOS MVP shipped 2 days ago.

**Velocity**: 137 cert-gap sheets landed in Waves 1–28 across ~10 days (2026-04-05 → 2026-04-13). ~14 sheets/day sustained pace. Then the iOS port was built in a single session on 2026-04-13 (roughly 30 files, 8 screens).

**ETA to Next Milestones**:
- Device install unblock: today–this week (depends on beta OS update or manual workaround)
- 4 content-gap sheets landed: end of week (Apr 18)
- BookmarkContext test coverage: same day as content sheets
- TestFlight upload: blocked by device install

**Historical Pattern**: The past 6 months show a bursty-then-dormant cadence. Waves ship in dense bursts, then quiet weeks. This is fine for a personal project — no artificial deadlines needed.

---

### The Sundial Tracks (Calendar — Schedule & Deadlines)
**This Week (Apr 15–21)**:
- Tue 15 (today): Round Table convened, battle plan filed
- Wed 16: Device install retry (wireless path); BookmarkContext reducer test
- Thu 17: Write DORA + EU AI Act sheets (compliance category)
- Fri 18: Write WebAssembly sheet (languages or new `wasm` category) + PQC migration sheet (security)
- Sat–Sun: Float / rest

**External Dependencies**:
- None. Apple beta update cadence is outside our control — monitor only.

**Schedule Conflicts**: None.

**Calendar Health**: Healthy — light week, realistic ambition.

---

### The Scroll Validates (Lore — Naming & Mythology)
**Naming Decisions Pending**:
- New `wasm` category vs. putting `webassembly.md` under `languages/`? **Recommend** `languages/webassembly.md` — WASM is a compile target, fits alongside Go/Rust/etc. in spirit.
- PQC sheet naming: `security/post-quantum-crypto.md` (kebab-case, matches existing `zero-trust.md`, `supply-chain-security.md` convention) ✓
- DORA: `compliance/dora.md` (like `gdpr.md`, `hipaa.md`, `pci-dss.md`) ✓
- EU AI Act: `compliance/eu-ai-act.md` ✓

**Mythology Consistency**: No conflicts. Naming pool follows the established "lowercase-kebab" convention everywhere.

**Sacred Law Compliance**: Every sheet has H1 = title, one-liner, H2 sections, `## See Also`, `## References`. No violations detected in spot-check.

---

### The Map Confirms (Kingdom — Hierarchy & Placement)
**Hierarchy Health**: 59 categories, all populated. Three undersized categories worth watching:
- `serverless/` — 2 sheets (lambda, serverless-patterns). Could absorb `wasm`? No — WASM lives better in `languages`.
- `web/` — 2 sheets. Candidate for consolidation or expansion (add `web-performance`, `webassembly` here instead?)
- `secrets/` — 2 sheets. Small but focused; leave alone.

**New Components — Placement Recommendations**:
| Sheet | Category | Rationale |
|-------|----------|-----------|
| `dora` | `compliance` | Matches `gdpr`, `hipaa`, `soc2` peers |
| `eu-ai-act` | `compliance` | Same peer group |
| `webassembly` | `languages` | Compile target + runtime family |
| `post-quantum-crypto` | `security` | Operational guidance (migration, key sizes, FIPS 203/204/205) — distinct from `cs-theory/lattice-crypto` which is theory-only |
| `slsa` | `security` | Supply-chain framework, complements `supply-chain-security`, `sbom`, `cosign` |
| `ebpf-observability` | `monitoring` | Observability-focused eBPF, distinct from `offensive/ebpf-security` |

**Tier Integrity**: No misplaced components detected.

---

### The Goblet Toasts (Busboy — Alignment & Coordination)
**Cross-Skill Conflicts**: None. The plan-phase's reviewers (Architect, Developer, Micromanager) agreed cleanly.

**Coordination Needs**:
- Any new sheet touches TWO files (`sheets/<cat>/<topic>.md` + `detail/<cat>/<topic>.md`) plus `## See Also` updates in 2–4 related sheets. Light coordination — one-author workflow.
- Device-install unblock coordinates only with Apple. No internal coord needed.

**Team Vibes**: Strong momentum. iOS port went from scaffold → shipping in one session. Content library is dense and consistent. This is a Kingdom in good standing.

**Translation Needed**: None — single-developer project, no business↔tech gap.

---

## Answer to "Any Missing Sheets?"

**Bottom line**: The 11 cert domains listed in `docs/BATTLE-PLAN.md` are fully covered. Waves 1–28 closed all 137 identified gaps. Spot checks:

✅ **CCNP DC**: cisco-nexus, fibre-channel, fcoe, fhrp, cisco-aci, cisco-ucs, vpc, span-erspan, roce, eigrp, copp, nxos-security, data-center-design, network-programmability, dc-automation — 15/15
✅ **CCNP EN**: dmvpn, sd-access, sd-wan, ip-sla, netflow-ipfix, cisco-wireless, dot1x, macsec, network-acl, gre-tunnels, pbr, cisco-dna-center, eem, cisco-ios-xr, flexvpn — 15/15
✅ **CCIE EI**: multicast-routing, mpls-te, mpls-vpn, lisp, isis-advanced, bgp-advanced, ospf-advanced, ipv6-advanced, vrf, network-services, qos-advanced, network-security-infra — 12/12
✅ **CCIE SP**: carrier-ethernet, unified-mpls, l2vpn-services, evpn-advanced, mvpn, srv6, bng, cgnat, sp-multicast, peering-transit, sp-qos, te-rsvp, g8032-erp, q-in-q — 14/14
✅ **CCIE SEC**: cisco-ftd, cisco-ise, trustsec, site-to-site-vpn, remote-access-vpn, email-gateway — 6/6
✅ **CCIE Auto**: yang-models, netconf, restconf, gnmi-gnoi, model-driven-telemetry, cisco-nso, napalm, nornir — 8/8
✅ **JNCIE-SP/SEC**: 30 junos-* sheets covering SRX, MPLS, EVPN, BGP, ISIS, IPsec, CoS, HA — comprehensive
✅ **Linux+**: users, process, filesystems, systemd, selinux, hardening, logs, package-managers covered
✅ **CISSP**: access-control-models, asset-security, bcp-drp, security-governance, security-models, security-architecture, risk-management, security-operations, incident-response, forensics — strong
✅ **C\|RAGE / CEH v13**: 37 offensive sheets (recon, enum, system-hacking, web-attacks, password-attacks, wireless-hacking, malware-analysis, etc.) — dense

### Forward-Looking Gaps (NOT in original cert scope, but topical for 2025–2026)

These are **not** certification gaps — they're **currency gaps**. Worth closing opportunistically:

| # | Proposed Sheet | Category | Rationale | Priority |
|---|----------------|----------|-----------|----------|
| 1 | `dora` | `compliance` | EU Digital Operational Resilience Act — in force Jan 2025, affects any EU-connected fintech | P1 |
| 2 | `eu-ai-act` | `compliance` | Phased enforcement through 2026; first major AI regulation | P1 |
| 3 | `post-quantum-crypto` | `security` | FIPS 203/204/205 finalized; NIST migration deadline looming. `cs-theory/lattice-crypto` has theory only — operational sheet missing | P1 |
| 4 | `webassembly` | `languages` | Wasmtime, WASI 0.2, component model — hot and not covered | P2 |
| 5 | `slsa` | `security` | SLSA v1.0 provenance framework — `sbom`, `cosign`, `supply-chain-security` exist but SLSA itself doesn't | P2 |
| 6 | `ebpf-observability` | `monitoring` | Parca, Pixie, Cilium Hubble — observability-side eBPF distinct from `offensive/ebpf-security` | P3 |
| 7 | `finops` | (new) `finops` or `cloud` | Cloud cost discipline — genuinely underrepresented | P3 |
| 8 | `kubevirt` | `virtualization` | VM workloads on k8s — growing adoption | P3 |

---

### Unified Battle Plan

#### Immediate Actions (Next 24–48 Hours)
- [ ] **[Developer]** Retry iOS device install via wireless debugging (Xcode → Window → Devices → "Connect via network")
- [ ] **[Developer]** Add `CsApp/src/core/BookmarkContext.test.ts` — reducer pure-function coverage (toggle add, toggle remove, initial state from list)
- [ ] **[Micromanager]** If device install still fails after wireless retry, file Apple Feedback with the `0xe8008014` + connection-drop log trail

#### This Sprint (Next 7 Days — Apr 15–21)
- [ ] **[Developer + Lore]** Write `compliance/dora.md` + `detail/compliance/dora.md` — Owner: author, Deadline: Thu Apr 17
- [ ] **[Developer + Lore]** Write `compliance/eu-ai-act.md` + detail — Deadline: Thu Apr 17
- [ ] **[Developer + Lore]** Write `security/post-quantum-crypto.md` + detail (FIPS 203 ML-KEM, FIPS 204 ML-DSA, FIPS 205 SLH-DSA, migration strategy) — Deadline: Fri Apr 18
- [ ] **[Developer + Lore]** Write `languages/webassembly.md` + detail (Wasmtime, WASI 0.2, component model) — Deadline: Fri Apr 18
- [ ] **[Developer]** Wire `## See Also` cross-refs from `tls`, `crypto-protocols`, `lattice-crypto` (theory) → new `post-quantum-crypto` sheet
- [ ] **[Architect]** Add CSP `<meta>` to `SheetViewer.tsx` `htmlTemplate` — defense-in-depth

#### Protocol / Content Library Milestones
- [ ] 689 sheets / 689 details after this sprint (685 + 4 new)
- [ ] Next Round Table trigger: (a) successful device install OR (b) 4 new sheets merged OR (c) cert exam scheduled

#### Decisions Made at This Round Table
1. **DEFER** device install for one week; pivot to wireless debugging fallback. Rationale: iOS 26 beta issue, not our code. Owner: Developer.
2. **ADOPT** 4 currency-gap sheets (DORA, EU AI Act, PQC, WebAssembly) as P1 for this sprint. Rationale: high topical value, small surface area. Owner: Developer + Lore.
3. **REJECT** aggressive content expansion (no FinOps, SLSA, eBPF-observability, KubeVirt this sprint) — stay focused, land 4 sheets well rather than 8 sheets thin. Owner: Micromanager.
4. **ADOPT ADR-001**: Go core is the single source of truth for all logic. No TS re-implementation. Owner: Architect.

#### Open Questions (Carry to Next Round Table)
1. Does iOS 26 GA (expected Fall 2026) fix device install? — Answer: monitor beta notes, answer on next Round Table
2. Should we version-tag v1.0 after TestFlight first install? — Answer: Captain decides when install succeeds

#### Wins to Celebrate
- iOS port scaffold → shipping MVP in one session (172 files, 8 screens) — Developer + Architect
- Syntax highlighting added end-to-end (Go goldmark + Chroma inline + WebView CSS override) — Developer
- 137 cert-gap sheets landed in 10 days across Waves 1–28 — Developer + Lore
- 685-sheet library hits all 11 elite certifications — strategic North Star met
- Cross-platform rendering parity: same Go goldmark pipeline drives CLI terminal output AND iOS WebView — Architect

---

### Next Round Table
**Scheduled**: Triggered on (a) device install success, or (b) 4 currency-gap sheets merged, or (c) Apr 22 (1 week out) — whichever comes first.
**Reason**: Verify sprint execution, assess whether to push TestFlight upload or add next batch of sheets.

---
_Forged at the Round Table by the full council. The Kingdom marches as one._
