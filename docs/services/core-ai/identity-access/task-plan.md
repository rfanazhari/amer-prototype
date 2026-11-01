# Task Plan: Identity and Access

Mode A; planning date 2026-11-01. T1–T5 are Done (Q1 resolved). Remaining tasks are Todo. Candidate access runtime is not claimed.

## Legend

- IDs T1–T15 are stable; never renumber on revision.
- Planning tickets use `CORE-IAM-001` through `CORE-IAM-015` mapped to T1–T15. These are local planning IDs, not externally created tickets.
- Complexity: S (<2h) | M (2–4h) | L (>4h), rough estimates requiring engineer validation.
- Status: `[ ]` Todo | `[~]` In Progress | `[x]` Done | `~~obsolete~~` superseded work.
- `ref` means [technical-design.md](technical-design.md) section; every task has a verifiable DoD.
- A task with an open decision gate is not executable for its affected behavior. Foundation tasks may proceed independently; parallel headings are not instructions to spawn agents.

## Tasks

### Independent foundation and administrative contract

- [x] **T1. Finalize administrative input, provisioning and security contracts** — Product/security/application/delivery — M
  - Ticket: `CORE-IAM-001` — depends: none — ref: design §2, §3, §6–7, §9 Q1.
  - Decision gate: Q1 resolved in T1; product/security policy is out-of-band ADMIN-chosen passwords, canonical email, and no mailer/recovery UI; engineering bounds are in design §6.
  - DoD: exact request/response properties, normalization/limits, password provisioning, bootstrap existing-state behavior, duplicate-account behavior, cookies/origin/CSRF and status/error order have examples in design §6; affected documents and dependent task gates synchronized. No runtime completion is implied.

- [x] **T2. Model role, account and access lifetime invariants** — Domain — M
  - Ticket: `CORE-IAM-002` — depends: none — ref: design §3–4 and rule ownership.
  - DoD: encapsulated account/session scope and lifetime primitives have public behavior tests for fixed roles/status, stable identity, exact expiry, revocation, creation vs restoration and pointer copying. Contact/password policy is not guessed; candidate/assignment-specific final construction is deferred to T12. Imports/documentation comply with architecture checks.

- [x] **T3. Define purpose-specific access ports and operation boundaries** — Application — M
  - Ticket: `CORE-IAM-003` — depends: none — ref: design §2–4.
  - DoD: application-owned transaction/read/credential/authority contracts express commit timing, current session/account binding, classified failures and assignment-first collaboration without GORM; preserve existing ACP authority shape. Register and generate mocks using `make mocks`; document all interface methods. Candidate integration port readiness is explicitly Q2-dependent.

- [x] **T5. Implement password, token and CSRF primitives** — Infrastructure service — M
  - Ticket: `CORE-IAM-005` — depends: T3 — ref: design §4 security adapter choices.
  - DoD: hash/verify and encoded-parameter validation use assessed Argon2id baseline, random secrets/digests are independent by credential kind, CSRF values bind to session/type, malformed hashes and concurrency are bounded; tests prove mismatch/collision/error paths and no secret output. Benchmark target hardware and record limits; no candidate exchange reuse or password acceptance policy is inferred. HTTP enforcement depends on T1/T8.

### Administrative implementation after Q1

- [x] **T4. Add administrative models, constraints and repository transactions** — Infrastructure/runtime — M
  - Ticket: `CORE-IAM-004` — depends: T1, T2, T3 — ref: design §3.
  - Blocked by: none for Q1; Q1 resolved in T1. Ordinary dependencies remain.
  - DoD: exactly two administrative tables installed additively, with role/status/time/email/digest/creator constraints and explicit null updates. Real PostgreSQL tests prove bootstrap/provision uniqueness and rollback, stored restoration, restart and upgrade with Task data. Existing resource budgets preserved; no candidate assignment stub installed.

- [ ] **T6. Implement administrative bootstrap, HR provisioning and session operations** — Application — L
  - Ticket: `CORE-IAM-006` — depends: T1, T2, T3, T5 — ref: design §2–6, BR-020..028.
  - Blocked by: none for Q1; Q1 resolved in T1. Ordinary dependencies remain.
  - DoD: narrow operations coordinate bootstrap, ADMIN-only HR creation, login/current-session/logout with generated port mocks; verified credential and account eligibility rechecked before session commit; known/uncertain persistence failures cannot return success; stable account SubjectID, exact expiry and current authority covered. No public signup, candidate password or account-reset endpoint added.

- [ ] **T7. Implement real administrative authority and ACP integration adapter** — Application/infrastructure service — M
  - Ticket: `CORE-IAM-007` — depends: T2, T3, T4, T6 — ref: design §2, §3 transactions, §4–5, §7.
  - DoD: real session/account resolution implements existing `AgentConfigurationAuthority`; SubjectID is stable account UUID and trusted context/session binding is checked. Current HR allowed, valid ADMIN forbidden, candidate/invalid session unauthenticated, unknown operation denied; recheck before replay. Real adapter tests cover expiry/revocation/disabled accounts and shared company scope without changing ACP fingerprints/codes. No claim that absent ACP HTTP consumers are tested.

- [ ] **T8. Wire secured administrative HTTP, bootstrap and API contracts** — Delivery/runtime/contracts — L
  - Ticket: `CORE-IAM-008` — depends: T1, T4, T5, T6, T7 — ref: design §2, §4–7.
  - DoD: four administrative paths in design §2 mounted through operation interfaces; environment bootstrap/origin/CSRF configuration is validated without logging secrets. Strict request/envelope/EN-ID errors, login origin/rate bounds, cookies, CSRF, no-store, logout commit behavior and wrong-role denial tested. Update OpenAPI, `docs/api/identity-access.md`, changelog, codes and translations together. No global auth added to Task/voice/health; lifecycle deadlines preserved.

- [ ] **T9. Prove administrative access and deliver ACP Q2 evidence** — Integration/verification — L
  - Ticket: `CORE-IAM-009` — depends: T4, T5, T6, T7, T8 — ref: design §3 proof matrix, §5, §8–9.
  - DoD: real HTTP/PostgreSQL proves bootstrap → ADMIN login → add HR → HR login → current authority → logout; restart, duplicate/concurrent provisioning, invalid credentials, exact expiry, revocation, ADMIN configuration denial and candidate-cookie exclusion covered. Run applicable commands from tactical-strategy §5 and record failures/skips/limitations. Publish access evidence to ACP Q2; ACP T8/T9 remain responsible for real configuration routes and replay/filtering. Do not mark all interview or ACP acceptance complete.

### Candidate decisions and blocked dependent work

- [ ] **T10. Resolve candidate, exchange and actual assignment collaboration contracts** — Product/security/Interview/application — M
  - Ticket: `CORE-IAM-010` — depends: T3 — ref: design §2–3, §6–7, §9 Q2/Q3/Q4.
  - Blocked by: Interview owner's absent actual assignment persistence/access/admission contract (Q2); product/security exchange reuse, lost-response/device policy (Q3); candidate contact/matching policy (Q4).
  - Ready when: real assignment schema/locking owner is supplied and missing business choices resolved. Contract drafting can proceed, but dependent implementation cannot.
  - DoD: actual assignment facts/FKs and transaction collaboration, contact fields/reuse, token/link delivery, repeated exchange/recovery and safe error order specified; current five-table properties reconciled explicitly if chosen policy needs extra columns. Ownership of consent, terminal no-restart and active cancellation remains Interview; synchronize design/strategy/tasks.

- [ ] **T11. Add candidate identity, invitation and session persistence** — Infrastructure/runtime — L
  - Ticket: `CORE-IAM-011` — depends: T2, T3, T4, T10 — ref: design §3.
  - Blocked by: Q2/Q3/Q4 — ready when T10 and actual assignment dependency are complete.
  - DoD: three remaining access tables complete the five-table target. Real same-assignment FK, current-invitation uniqueness, distinct credential namespaces, time/null mapping and assignment-first issue/exchange/revoke coordination proven in PostgreSQL. Concurrent reissue/exchange/rollback tests use actual assignment owner; no fake sixth table. Candidate email does not become unique implicitly.

- [ ] **T12. Implement candidate identity and assignment-scoped access operations** — Domain/application — L
  - Ticket: `CORE-IAM-012` — depends: T2, T3, T5, T6, T10 — ref: design §2–6, BR-029..039.
  - Blocked by: Q2/Q3/Q4 — ready when T10 provides executable candidate/session contracts and assignment collaboration.
  - DoD: HR candidate create/read and invitation issue/reissue; explicit exchange and session resolution use trusted facts, scoped credentials and exact expiry. Tests cover no execution/model/consent side effects, current-session reload, cross-assignment denial, reissue preserving active access, assignment revocation and resolved retry/device policy. No caller role/candidate/assignment override accepted at exchange.

- [ ] **T13. Wire candidate access HTTP and invitation landing interaction** — Delivery/runtime/contracts/browser — L
  - Ticket: `CORE-IAM-013` — depends: T5, T8, T10, T11, T12 — ref: design §2, §4 security, §5–7.
  - Blocked by: Q2/Q3/Q4 and absence of candidate UI integration harness — ready when T10–T12 complete and Interview owner supplies the minimal UI/harness contract.
  - DoD: candidate identity/invitation/session endpoints, strict inputs, origin/CSRF, cookies and errors are documented/tested; invitation token removed from URL/history before external resources; GET/prefetch cannot exchange/start. Candidate browser refresh uses existing session and resolved recovery policy. Update OpenAPI, endpoint docs and changelog together; do not build consent or interview runtime inside this task.

- [ ] **T14. Prove real candidate access and Interview integration** — Integration/browser verification — L
  - Ticket: `CORE-IAM-014` — depends: T9, T10, T11, T12, T13 — ref: design §3 proof matrix, §8–9.
  - Blocked by: Q2 actual Interview consumers, including admission/revocation/terminal-state integration — ready when actual assignment and required consumers are runnable.
  - DoD: real assignments A/B reuse one candidate and maintain isolated sessions; race/rollback/restart, resolved exchange retries, exact expiry, source-invitation replacement, assignment revocation and completed no-restart proven. Record browser runner commands plus repository verification, skips and per-rule evidence. Active-run cancellation/session-validity admission is accepted only with actual Interview integration, not fake returned facts.

- [ ] **T15. Define and implement credential/contact retention when authorized** — Product/data/application/infrastructure — M
  - Ticket: `CORE-IAM-015` — depends: T10, T11 — ref: design §3 rollout/retention, §9 Q5.
  - Blocked by: Q5 retention/purge and schema-reference ordering — ready when product/data owner approves explicit policy and synchronized design.
  - DoD: reviewable retention and FK-safe deletion plan, restart-safe bounded cleanup, preserved required references and deletion evidence; no physical deletion before approval. If retention needs additional persisted facts, revise assessed design rather than silently changing the five-table contract.

## Dependency Graph

```text
Q1 (resolved) → T1 (Done)
T2 (pure domain foundation)
T3 → T5 (credential primitives)
T1 + T2 + T3 → T4 (administrative database)
T1 + T2 + T3 + T5 → T6 (administrative operations)
T2 + T3 + T4 + T6 → T7 (real authority)
T1 + T4 + T5 + T6 + T7 → T8 (administrative HTTP)
T4 + T5 + T6 + T7 + T8 → T9 (access evidence → ACP Q2 access prerequisite)
T3 + actual assignment dependency + Q2/Q3/Q4 → T10
T2 + T3 + T4 + T10 → T11 (candidate database)
T2 + T3 + T5 + T6 + T10 → T12 (candidate operations)
T5 + T8 + T10 + T11 + T12 + UI/harness → T13
T9 + T10 + T11 + T12 + T13 + actual Interview consumers → T14
T10 + T11 + Q5 → T15
```

## Revision History

| Version | Date | Change |
|---|---|---|
| v1 | 2026-11-01 | Initial remaining-work plan from assessed Mode A design; all implementation Todo, administrative and assignment-dependent candidate readiness separated |
| v1.1 | 2026-11-01 | Mark T1 Done and record Q1 resolved in T1; T2–T15 remain Todo |
| v1.2 | 2026-11-01 | Mark T2 Done for domain role, account and session lifetime invariants; T3–T15 remain Todo |
| v1.3 | 2026-11-01 | Mark T4 Done for two administrative tables, uniqueness transactions and additive AutoMigrate; T3 and T5–T15 remain Todo |
| v1.4 | 2026-11-01 | Mark T3 Done for purpose-specific access ports, classified failures and operation boundaries; T5–T15 remain Todo |
| v1.5 | 2026-11-01 | Mark T5 Done for Argon2id, kind-scoped token digests and session-bound CSRF; T6–T15 remain Todo |
