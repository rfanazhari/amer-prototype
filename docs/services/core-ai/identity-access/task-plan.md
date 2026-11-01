# Task Plan: Identity and Access

Mode C revision; planning date 2026-11-01. T1–T9 retain their recorded Done status (Q1 resolved); this review does not rerun their acceptance. Owner approved replacing AutoMigrate with manual, versioned SQL migrations across the service (D8 / Q6 resolved). T16–T20 plan that transition; no implementation or database change is performed here. Candidate runtime remains unimplemented.

## Legend

- IDs T1–T20 are stable; never renumber on revision.
- Planning tickets use `CORE-IAM-001` through `CORE-IAM-020` mapped to T1–T20. These are local planning IDs, not externally created tickets; T16–T20 cover shared Task/access schema infrastructure.
- Complexity: S (<2h) | M (2–4h) | L (>4h), rough estimates requiring engineer validation.
- Status: `[ ]` Todo | `[~]` In Progress | `[x]` Done | `~~obsolete~~` superseded work.
- `ref` means [technical-design.md](technical-design.md) section; every task has a verifiable DoD.
- A task with an open decision gate is not executable for its affected behavior. Foundation tasks may proceed independently; parallel headings are not instructions to spawn agents.

## Tasks

### Reconciliation: shared manual-migration foundation

The owner has resolved the schema-policy choice; do not ask for that approval again. Both code and documentation need to change. The existing AutoMigrate work was valid under its original contract: keep T4/T8/T9 Done rather than relabeling them obsolete. The following new work replaces their schema-installation mechanism before dependent candidate persistence. No business-rule repair is required.

- [ ] **T16. Capture the legacy schema baseline and synchronize approved policy** — Infrastructure/design/operations — M
  - Ticket: `CORE-IAM-016` — depends: T4, T8 — ref: design §1 baseline, §3.1, §3.3–3.4, §9 Q6; decision-log D8.
  - DoD: inventory all runtime/test AutoMigrate and adapter DDL callers. Reproduce current Task + two administrative schemas on isolated PostgreSQL; record exact types/defaults/nullability, PKs, checks, all five administrative indexes and RESTRICT FKs in `docs/design/database-schema-baseline.md`; freeze synthetic legacy SQL at `test/integration/testdata/schema/legacy_automigrate.sql`. Distinguish observed generated schema from any uninspected deployed target. Allocate the first two versions without conflicting with other work. Synchronize schema-policy wording in root `AGENTS.md`, `README.md`, `requirement.txt`, `docs/design/task-tracking.md` and the audit's current-policy note while preserving historical selections/results. No unrelated standard/skill rewrite, runtime change or live-data correction.

- [ ] **T17. Add versioned baseline SQL and explicit migration tooling** — Infrastructure/cmd/build — L
  - Ticket: `CORE-IAM-017` — depends: T16 — ref: design §3.1–3.3 and migration proof matrix, §4.
  - DoD: create `migrations/pgsql/000001_tasks.{up,down}.sql` and `000002_administrative_access.{up,down}.sql` equivalent to the captured schema. Add pinned/verified golang-migrate v4 dependency, thin `cmd/migrate/main.go`, adapter under `infrastructure/database/schema/` and proposed Make targets `migrate.status`, `migrate.up`, `migrate.baseline`, guarded `migrate.down`. Explicit target required for up; secrets come from `MIGRATION_DATABASE_URL`, never arguments/logs. Baseline supports exact Task-only → version 1 and exact three-table → version 2 under exclusive coordination; rejects drift/partial schema/foreign metadata. Transactions, metadata/dirty state, immutable artifacts, 60-second command budget/5-second lock wait, bounded cancellation and no automatic force are covered by focused tests. No accounts/reference data seeded. Artifacts alone do not mark runtime ownership switched.

- [x] **T18. Replace startup and test AutoMigrate with explicit migration execution** — Runtime/integration/Compose/operations — L
  - Ticket: `CORE-IAM-018` — depends: T17 — ref: design §3.2–3.4, §4–5, §8.
  - DoD: remove both runtime AutoMigrate calls and `accessrepository.AutoMigrate`/adapter-owned installer; CRUD models remain. Runtime checks clean compatible version 2 before bootstrap/listening and never creates schema/metadata. All integration helpers use versioned setup; legacy SQL is fixture-only. Move blocked-DDL cancellation to migration runner coverage and retain runtime cancellation/cleanup tests. Update Makefile, Dockerfile/Compose artifact delivery and `scripts/verify_compose.py`: explicit migration command/job runs after PostgreSQL and before verification app startup; normal startup does no DDL. Document local/target commands and `MIGRATION_DATABASE_URL` versus DDL-denied runtime `DATABASE_URL` in README and `docs/design/database-migrations.md`. Startup/shutdown budgets, Task events and administrative public contracts remain intact. No executable AutoMigrate caller remains in production or integration setup.

- [x] **T19. Prove migration equivalence, adoption, failure handling and regressions** — PostgreSQL/integration/architecture — L
  - Ticket: `CORE-IAM-019` — depends: T17, T18 — ref: design §3.1–3.4 migration proof matrix, §8; tactical-strategy §5.
  - DoD: real disposable PostgreSQL proves fresh up-to-2, down/up, rerun no-op; populated Task-only and full-three-table guarded adoption; preservation of IDs, contacts, hash equality, expiry/revocation, creator links and Task history without secret output. Partial/drifted schemas, wrong index/FK definitions, unsupported metadata, dirty/old/future version, concurrent migrators, lock timeout, cancelled/injected-failure DDL and guarded recovery have explicit cases. Runtime starts with a DDL-denied role only on compatible clean schema and releases resources on failure. Existing Task queue/rollback and ADMIN/HR/login/logout/restart behavior still pass. Run `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, `make test.security`, and `make test.docker`; record exact results/skips and `make migrate.*` invocations. Use isolated Compose or an explicitly authorized `TEST_DATABASE_URL`; source inspection/sqlmock is not database proof.

- [x] **T20. Rehearse cutover/recovery and hand off the migration foundation** — Operations/design/verification — M
  - Ticket: `CORE-IAM-020` — depends: T16, T17, T18, T19 — ref: design §3.2–3.3, §9; tactical-strategy §6.
  - DoD: rehearse on an authorized disposable populated fixture: stop legacy schema writers → snapshot/backup → catalog preflight → baseline/adopt or up to release target → clean-version check → migration-aware app → administrative/Task smoke. Rehearse restore/forward recovery and compatible-binary rollback; no destructive down/unchecked force on retained data. Publish exact runbook, release SQL checksum manifest, version compatibility and evidence limitations; align design/strategy/task status with actual results. Actual non-disposable cutover remains a separately authorized operation with inspected target/backup, not implied by this task. Candidate T11 receives the verified schema foundation; no claim that candidate/ACP/Interview consumers are accepted.

Execution T18–T20: Done on disposable PostgreSQL and the isolated packaged stack. See [commands, checkpoint and acceptance evidence](t18-t20-evidence.md), [operator runbook](../../../design/database-migrations.md) and [release checksums](../../../design/database-migration-checksums.sha256). The prior T17 Docker regression limitation is superseded by these passing runs; its historical checkpoint/status is not rewritten by this T18–T20-only execution. No non-disposable cutover or candidate/ACP/Interview acceptance is implied.

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
  - Revision note: completed under the original AutoMigrate policy; T16–T20 replace schema installation while retaining models, transactions and data.

- [x] **T6. Implement administrative bootstrap, HR provisioning and session operations** — Application — L
  - Ticket: `CORE-IAM-006` — depends: T1, T2, T3, T5 — ref: design §2–6, BR-020..028.
  - Blocked by: none for Q1; Q1 resolved in T1. Ordinary dependencies remain.
  - DoD: narrow operations coordinate bootstrap, ADMIN-only HR creation, login/current-session/logout with generated port mocks; verified credential and account eligibility rechecked before session commit; known/uncertain persistence failures cannot return success; stable account SubjectID, exact expiry and current authority covered. No public signup, candidate password or account-reset endpoint added.

- [x] **T7. Implement real administrative authority and ACP integration adapter** — Application/infrastructure service — M
  - Ticket: `CORE-IAM-007` — depends: T2, T3, T4, T6 — ref: design §2, §3 transactions, §4–5, §7.
  - DoD: real session/account resolution implements existing `AgentConfigurationAuthority`; SubjectID is stable account UUID and trusted context/session binding is checked. Current HR allowed, valid ADMIN forbidden, candidate/invalid session unauthenticated, unknown operation denied; recheck before replay. Real adapter tests cover expiry/revocation/disabled accounts and shared company scope without changing ACP fingerprints/codes. No claim that absent ACP HTTP consumers are tested.

- [x] **T8. Wire secured administrative HTTP, bootstrap and API contracts** — Delivery/runtime/contracts — L
  - Ticket: `CORE-IAM-008` — depends: T1, T4, T5, T6, T7 — ref: design §2, §4–7.
  - DoD: four administrative paths in design §2 mounted through operation interfaces; environment bootstrap/origin/CSRF configuration is validated without logging secrets. Strict request/envelope/EN-ID errors, login origin/rate bounds, cookies, CSRF, no-store, logout commit behavior and wrong-role denial tested. Update OpenAPI, `docs/api/identity-access.md`, changelog, codes and translations together. No global auth added to Task/voice/health; lifecycle deadlines preserved.

- [x] **T9. Prove administrative access and deliver ACP Q2 evidence** — Integration/verification — L
  - Ticket: `CORE-IAM-009` — depends: T4, T5, T6, T7, T8 — ref: design §3 proof matrix, §5, §8–9.
  - DoD: real HTTP/PostgreSQL proves bootstrap → ADMIN login → add HR → HR login → current authority → logout; restart, duplicate/concurrent provisioning, invalid credentials, exact expiry, revocation, ADMIN configuration denial and candidate-cookie exclusion covered. Run applicable commands from tactical-strategy §5 and record failures/skips/limitations. Publish access evidence to ACP Q2; ACP T8/T9 remain responsible for real configuration routes and replay/filtering. Do not mark all interview or ACP acceptance complete.

### Candidate decisions and blocked dependent work

- [x] **T10. Resolve candidate, exchange and actual assignment collaboration contracts** — Product/security/Interview/application — M
  - Ticket: `CORE-IAM-010` — depends: T3 — ref: design §2–3, §6–7, §9 Q2/Q3/Q4.
  - Decision gate: Q2/Q3/Q4 resolved in T10; Access contracts are executable for T12. Real Interview persistence and invitation FKs remain T11/T14 obligations, not in-memory fakes.
  - DoD: actual assignment facts/FKs and transaction collaboration, contact fields/reuse, token/link delivery, repeated exchange/recovery and safe error order specified; current five-table properties reconciled explicitly if chosen policy needs extra columns. Ownership of consent, terminal no-restart and active cancellation remains Interview; synchronize design/strategy/tasks.

- [x] **T11. Add candidate identity, invitation and session persistence** — Infrastructure/runtime — L
  - Ticket: `CORE-IAM-011` — depends: T2, T3, T4, T10, T20 — ref: design §3, especially §3.1 migration artifacts.
  - Blocked by: T20 schema foundation and Interview assignment migration — Q2/Q3/Q4 resolved in T10. Stage A (candidate identity) is ready after T10 contracts; production installation still waits for T20. Stage B (invitations/sessions) additionally waits for the actual Interview assignment migration. Full task completion requires both stages; do not make candidate creation wait for an assignment FK that itself references candidates.
  - DoD: three remaining access business tables complete the five-table target through proposed `migrations/pgsql/000003_candidates.{up,down}.sql` and `000004_assignment_access.{up,down}.sql`; confirm/reallocate versions against the shared stream. Candidate table precedes the real Interview assignment FK dependency, then invitations/sessions follow that owner's migration. Real same-assignment FK, partial current-invitation uniqueness, credential namespaces, null/time mapping and assignment-first races/rollback proven in PostgreSQL. No AutoMigrate or fake assignment table. No implicit unique candidate email. Define safe forward/recovery and disposable reversal for each pair; migration metadata is counted separately from business tables.

- [x] **T12. Implement candidate identity and assignment-scoped access operations** — Domain/application — L
  - Ticket: `CORE-IAM-012` — depends: T2, T3, T5, T6, T10 — ref: design §2–6, BR-029..039.
  - Blocked by: none for Q2/Q3/Q4; Q2/Q3/Q4 resolved in T10. Ordinary T10 dependency remains.
  - DoD: HR candidate create/read and invitation issue/reissue; explicit exchange and session resolution use trusted facts, scoped credentials and exact expiry. Tests cover no execution/model/consent side effects, current-session reload, cross-assignment denial, reissue preserving active access, assignment revocation and resolved retry/device policy. No caller role/candidate/assignment override accepted at exchange.

- [x] **T13. Wire candidate access HTTP and invitation landing interaction** — Delivery/runtime/contracts/browser — L
  - Ticket: `CORE-IAM-013` — depends: T5, T8, T10, T11, T12 — ref: design §2, §4 security, §5–7.
  - Blocked by: Q2/Q3/Q4 and absence of candidate UI integration harness — ready when T10–T12 complete and Interview owner supplies the minimal UI/harness contract.
  - DoD: candidate identity/invitation/session endpoints, strict inputs, origin/CSRF, cookies and errors are documented/tested; invitation token removed from URL/history before external resources; GET/prefetch cannot exchange/start. Candidate browser refresh uses existing session and resolved recovery policy. Update OpenAPI, endpoint docs and changelog together; do not build consent or interview runtime inside this task.

- [x] **T14. Prove real candidate access and Interview integration** — Integration/browser verification — L
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
Q6 (manual migrations approved) + T4 + T8 → T16 (legacy schema manifest/policy sync)
T16 → T17 (baseline SQL + explicit runner)
T17 → T18 (runtime/test/Compose schema-owner replacement)
T17 + T18 → T19 (real migration/adoption/failure evidence)
T16 + T17 + T18 + T19 → T20 (rehearsed recovery and foundation handoff)
T3 + actual assignment dependency + Q2/Q3/Q4 → T10
T2 + T3 + T4 + T10 + T20 → T11 stage A (candidate identity SQL)
T11 stage A → actual Interview assignment migration → T11 stage B (invitation/session SQL; T11 complete)
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
| v1.6 | 2026-11-01 | Mark T6 Done for administrative bootstrap, HR provision and session operations; T7–T15 remain Todo |
| v1.7 | 2026-11-01 | Mark T7 Done for real ACP authority adapter, stable SubjectID and current HR recheck; T8–T15 remain Todo |
| v1.8 | 2026-11-01 | Mark T8 Done for administrative HTTP, bootstrap configuration and published access contracts; T9–T15 remain Todo |
| v1.9 | 2026-11-01 | Mark T9 Done for administrative HTTP/PostgreSQL evidence and ACP Q2 access handoff; T10–T15 remain Todo |
| v1.10 | 2026-11-01 | Mode C: preserve T1–T9 Done; add T16–T20 for owner-approved service-wide manual migrations, data-preserving baseline/adoption, runtime/test reconciliation and recovery evidence; T11 now consumes that foundation |
| v1.11 | 2026-11-01 | Mark T10 Done and record Q2/Q3/Q4 resolved in T10; T11–T15 remain Todo with T11 still waiting on T20 and Interview assignment migration |
| v1.12 | 2026-11-01 | Mark T12 Done for candidate identity, invitation issue/reissue, exchange and current-session operations |
| v1.13 | 2026-11-01 | Mark T11 Done for candidate/invitation/session GORM adapters, sqlmock coverage and proposed 000003/000004 SQL; runtime still does not AutoMigrate or apply them; real PostgreSQL FK proof and T20 remain outstanding |
| v1.14 | 2026-11-01 | Mark T18–T20 Done for explicit startup/test migration ownership, PostgreSQL acceptance, populated backup/adoption/restore and compatible-image rollback rehearsal; publish runbook/checksums and bounded evidence |
