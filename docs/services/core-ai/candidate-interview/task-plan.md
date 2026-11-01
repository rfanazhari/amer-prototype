# Task Plan: Candidate Interview

Planning date 2026-11-02. Mode A. Every task is Todo. No implementation evidence exists, and none is implied by neighboring Done work in Identity and Access or Agent Configuration. BR-CORE-040..047 are not enforced.

## Ticket Identity

Capability group: **CIN — Candidate Interview**. Ticket prefix: `CORE-CIN`. These are repository planning IDs. No external tracker tickets have been created.

Each task has a stable ticket ID: `T1` → `CORE-CIN-001`, then onward in order through `T13` → `CORE-CIN-013`. Keep both identities on later revisions. Add new numbers if scope grows. Do not renumber and do not reuse an ID.

## Legend

- IDs T1–T13 are stable.
- Complexity: S (<2h), M (2–4h), L (>4h). Engineers still validate the estimates.
- Status: `[ ]` Todo, `[~]` In Progress, `[x]` Done, `~~obsolete~~` superseded work.
- `ref` identifies sections of [technical-design.md](technical-design.md).
- DoD is observable evidence. A skipped database suite is reported as skipped.
- A blocked task is not executable until its ready-when condition is true and the design still matches that condition.
- At closure, align design, strategy, and status with what was verified. Record RED/GREEN only from commands run for that task.

Companion plans: [Identity and Access](../identity-access/task-plan.md) owns the session this slice consumes. Execution Lifecycle and Interview Runtime have capability documents and no tripack in this repository; their absence is why T11 and T12 are blocked.

## Tasks

### Contract decision

- [x] **T1. Record the notice version and statement** — Product / composition — M
  - Ticket ID: `CORE-CIN-001`
  - depends: none — ref: design §3 seed, §6, §9 Q1; decision-log D1.
  - Blocked by: none. Q1 resolved 2026-11-02: version `notice-261102`, statement in design §6.
  - Evidence: `infrastructure/runtime/candidatenotice.go` holds the catalog value and `TestCurrentCandidateNotice*` checks it against design §6. Wiring it into the `NoticeCatalog` port is T3/T5; accept is not live until T5 and T6.
  - DoD: design §6 names the exact version and the full statement. The version matches the §3 character rule. Composition loads that catalog and an empty catalog is no longer the production setting. No other notice text is invented. Documents only, plus the catalog value. No claim that accept is live until T5 and T6 have run.

### Independent foundation

- [x] **T2. Model consent, coverage, and the controller** — Domain — M
  - Ticket ID: `CORE-CIN-002`
  - depends: none — ref: design §3–4 and the rule table.
  - Evidence: `domain/aggregate/candidateinterview` with public `_test` package tests (100% statement coverage). `make test`, `make test.architecture`, and `make lint` pass. No HTTP, SQL, or stage workflow.
  - DoD: public `_test` package tests cover current-version acceptance, restoration of a retired version, copied acceptance time, coverage equality refusal, takeover increment, and stale-version refusal. `NoticeVersion` and `ControllerID` are distinct. The 15-minute budget is the platform §10 constant, not an HR stage-second field. `make test` and `make test.architecture` pass, including `test/architecture/documentation_test.go`. No HTTP, SQL, or stage workflow.

- [x] **T3. Define operations and sibling ports** — Application — M
  - Ticket ID: `CORE-CIN-003`
  - depends: none — ref: design §2–4, §6.3.
  - Evidence: `application/service` holds `NoticeCatalog`, `ExecutionLifecycle`, `InterviewProgress`, and the classified `InterviewError` sentinels (not found, conflict, coverage, dependency). `application/repository/candidateinterviewport` holds `ConsentTransaction`, `Consents`, `Controllers`. Operation interfaces are in `candidateinterviewusecase` and `candidateinterviewquery`, with detached `application/dto/candidateinterview.go`. `.mockery.yml` lists all 12 interfaces and `make mocks` generated `test/mocks/`. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, and `make lint` pass. No implementations, SQL, or HTTP yet; `CIN_` public codes are T6.
  - DoD: narrow operation interfaces, consent and controller repository contracts, `NoticeCatalog`, `ExecutionLifecycle`, and `InterviewProgress`. Interfaces carry no GORM type. Classified errors distinguish not-found, conflict, coverage, and dependency failure. `.mockery.yml` lists the interfaces and `make mocks` regenerates `test/mocks/`. Every interface method is commented. `make test.architecture` passes.

### Sequential persistence and consent HTTP

- [x] **T4. Register SQL version 9** — Infrastructure / schema — M
  - Ticket ID: `CORE-CIN-004`
  - depends: T2 — ref: design §3 migration, §8.
  - Evidence: `migrations/pgsql/000009_candidate_interview.{up,down}.sql` create only `interview_consents` and `interview_controllers`; `schema.ReleaseVersion` is 9, with owned/grant tables, `checksums.json`, the SHA-256 manifest, baseline adoption (`extendBaseline` 8→9), Makefile, README, `scripts/verify_compose.py`, and `docs/design/database-migrations.md` aligned. Schema tests use sqlmock for the floor messages and a disposable PostgreSQL 16 container for clean 8→9, constraints, down/up, baseline, and a floor-8 build refusing version 9. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.security` pass. `make test.integration` on the disposable database has the same five failing tests at the pre-change commit (fixture schema conflicts, unrelated to version 9). `make test.docker` was not run. No Go behavior beyond the schema package; consent persistence is T5.
  - DoD: `migrations/pgsql/000009_candidate_interview.up.sql` and `.down.sql` create only `interview_consents` and `interview_controllers` with the FKs, checks, and unique pair in §3. No notice seed, no execution state column, no `AutoMigrate`. `schema.ReleaseVersion` is 9. Checksums, runtime grants, and `docs/design/database-migrations.md` match. A disposable database migrates from clean 8 to clean 9, refuses the floor-8 binary, and down/up is demonstrated only on that disposable database. `make test.architecture` passes. Versions 1–8 are unmodified.

- [x] **T5. Persist consent and the interview read** — Application / infrastructure — L
  - Ticket ID: `CORE-CIN-005`
  - depends: T2, T3, T4 — ref: design §3, §5–6, BR-CORE-040 and BR-CORE-043.
  - Evidence: `candidateinterviewrepository.ConsentStore` (GORM, `ConsentModel`) implements `ConsentTransaction` and `Consents`: `SELECT … FOR UPDATE` on `interview_assignments`, session re-read against the injected `Clock`, insert-only consent, unique violation mapped to `ErrAlreadyExists`. `candidateinterviewusecase.AcceptNotice` validates input first, fails closed on an empty catalog, and after a unique race reads the winner back in a new transaction. `candidateinterviewquery.GetInterview` builds the allow-list view (no controller id, salary or report field); an absent or unbound lifecycle adapter renders `execution: null`. sqlmock covers statement shape; `TestCandidateConsentRepositoryPostgres` proves the 8-way concurrent accept, revocation, rollback and retired-version reads on a disposable PostgreSQL 16 container. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.security` pass. Full `make test.integration` on that container still fails the five pre-existing fixture tests, and the new test also fails in the full run because the access fixture drops `candidates` with CASCADE; it passes when run alone. `make test.docker` was not run. No HTTP, `GetExecution`, or controller persistence (T6, T8, T9).
  - DoD: accept inserts one row for the current fixture notice and returns that row on retry, including after a unique-violation race, using a new transaction for the read-back. Empty catalog inserts nothing and returns the dependency error. GET returns notice, consent, null execution, and actions, and has no salary, report, or controller-id field. Assignment lock is `FOR UPDATE` inside the interview repository. PostgreSQL proof uses `make test.integration` or `make test.docker` on an authorized disposable database. sqlmock may cover statement shape and is not the uniqueness proof. `make test` passes.

- [x] **T6. Mount consent, interview read, and the page shell** — Delivery / runtime / contract — L
  - Ticket ID: `CORE-CIN-006`
  - depends: T3, T5 — ref: design §2, §5–7.
  - Evidence: `candidateinterviewhandler` serves `GET /candidate/interview` and `POST /candidate/consents` (strict JSON, origin, candidate cookie, `X-CSRF-Token` → `CIN_FORBIDDEN`, 201/200 replay, 401 `IA_UNAUTHENTICATED`, empty catalog → 503 `CIN_NOTICE_UNAVAILABLE`); `candidateinterview` serves `GET /candidate/interview/page`. `Router` dispatches the three paths ahead of the access catch-all; exchange and `GET /candidate/session` stay on access. `runtime.NewCandidateInterview` composes the approved notice catalog, `ConsentStore`, and fail-closed stand-ins for lifecycle and controllers (execution renders `null`). Eleven `CIN_` codes exist in `pkg/errs` and both translation files. OpenAPI, `docs/api/candidate-interview.md`, `CHANGELOG.md` (0.9.0), and `identity-access.md` describe the routes. The landing navigates once to the page after the session read succeeds. HTTP tests use mocked operations; sqlmock covers the composition. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.security` pass; database suites were skipped (no `TEST_DATABASE_URL`). The page shell was exercised in the built-in browser against a local stub server serving the real `page.html` (access-error without a cookie; notice and accept with one; accepted state after click), not against the real service or a database. The real-service flow is not demonstrated. No start, poll, takeover, or cancel (T9).
  - DoD: `GET /candidate/interview`, `POST /candidate/consents`, and `GET /candidate/interview/page` are dispatched before the access catch-all. Exchange and `GET /candidate/session` stay on access and keep their published behavior. OpenAPI, `docs/api/candidate-interview.md`, and `docs/api/CHANGELOG.md` describe the mounted routes. `CIN_` codes exist in `pkg/errs` and both translation files. HTTP tests use mocked operations and cover strict JSON, CSRF, 201/200 replay, and 401 `IA_UNAUTHENTICATED`. The landing page navigates to the interview page only after session success. `make test`, `make test.contract`, `make test.architecture`, and `make lint` pass. The repository has no browser runner; exercise the page shell in a browser, or record the substitute that was available.

### Orchestration that can use ports before siblings exist

- [x] **T7. Orchestrate start, takeover, cancel, and polling** — Application — L
  - Ticket ID: `CORE-CIN-007`
  - depends: T2, T3, T5 — ref: design §5–6, BR-CORE-040..047.
  - Evidence: `candidateinterviewusecase.StartInterview`, `Takeover` and `CancelInterview` and `candidateinterviewquery.GetExecution` are plain structs over Mockery-generated ports (no new interfaces, so `make mocks` was not needed). Start validates the `Idempotency-Key` and tab id, then notice catalog, stored consent of the current version, and coverage (`expiresAt.After(now+15m)`), and only then calls `CommitOrFind` once; it writes no consent and claims the controller for the starting tab (an existing controller is read back). Takeover and cancel reject an execution id outside the assignment as not found; a stale controller version is `ErrInterviewConflict`; a terminal execution is refused for both and is never relabelled; a non-controller cancel returns before `Cancel`; a controller cancel sends `candidate_early_stop`. `GetExecution` uses `Find` only and shares `safeExecution` with `GetInterview`. New usecase sentinels `ErrConsentRequired`, `ErrNotController`, `ErrInvalidIdempotencyKey` are mapped to public codes in T9. Mockery mocks only, no real execution row, no SQL, no HTTP. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.security` pass; database suites were skipped (no `TEST_DATABASE_URL`).
  - DoD: Mockery tests show start calls lifecycle only after current consent and coverage; the same idempotency key and consent id return one execution; a different consent id conflicts; coverage equality does not call the port; GET does not call commit-or-find; a non-controller cancel does not call the port; a controller cancel asks for `candidate_early_stop` and the following view is `CANCELLED`; a terminal view has no report field; an unbound lifecycle port returns the dependency error and does not insert consent. `make test` passes. This task does not claim a real execution row.

- [x] **T8. Persist controller takeover** — Infrastructure — M
  - Ticket ID: `CORE-CIN-008`
  - depends: T3, T4, T7 — ref: design §3 controller, BR-CORE-044.
  - Evidence: `candidateinterviewrepository.ControllerStore` (GORM, `ControllerModel`) implements `candidateinterviewport.Controllers`. `Find` is a plain read scoped to the assignment. `Claim` inserts the first row and maps a unique violation to `ErrAlreadyExists`. `Update` runs `SELECT … FOR UPDATE` on the controller row, restores the aggregate, runs the callback, and writes `controller_id`, `version` and `updated_at` through an explicit `Select` list; a callback error (stale version, terminal) rolls back and leaves the previous id, an absent row returns `ErrNotFound` without running the callback, and a corrupt row is a technical error. `interview_executions` gains no column. sqlmock covers statement shape; `TestCandidateControllerRepositoryPostgres` proves the single row per assignment, takeover write, stale refusal, FK to a real execution pin, and one winner among 8 concurrent takeovers on a disposable PostgreSQL 16 container at version 9. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.security` pass. Full `make test.integration` on that container still fails the pre-existing fixture tests, and the new test also fails in the full run because an earlier fixture leaves `candidates` missing (`relation "candidates" does not exist`, same cause as the T5 test); it passes when run alone. `make test.docker` was not run. Not yet composed into `runtime`; no HTTP (T9).
  - DoD: one controller row per assignment, explicit update of id, version, and `updated_at`, stale `expectedVersion` leaves the previous id, concurrent updates produce one winner. Fixture execution rows may use the existing pin insert so the FK is real. `interview_executions` gains no column. PostgreSQL proof uses `make test.integration` or `make test.docker`. `make test.architecture` passes.

- [ ] **T9. Mount start, poll, takeover, and cancel** — Delivery / contract — L
  - Ticket ID: `CORE-CIN-009`
  - depends: T6, T7, T8 — ref: design §2, §6–7.
  - DoD: `POST /candidate/executions`, `GET /candidate/executions/{id}`, takeover, and cancel follow §6, including `Idempotency-Key`, 201/200, and the §7 error order. Session-attempt, ready, and report routes are absent from the router and from OpenAPI. Another assignment's execution id is 404. An unbound lifecycle adapter is 503 for start and cancel. HTTP tests mock the operations. `make test.contract`, `make test.architecture`, and `make lint` pass.

- [ ] **T10. Candidate screen for consent, control, cancel, and the final outcome** — Delivery UI — L
  - Ticket ID: `CORE-CIN-010`
  - depends: T6, T9 — ref: design §5–6; decision-log D1, D2, D3.
  - DoD: the page shows the access-error state without a cookie, the notice and accept action from the API, start only when offered, a second tab as read-only until takeover, a confirmation before cancel, and a terminal screen for `COMPLETED`, `FAILED`, and `CANCELLED`. The cancelled state is labeled cancelled. The page issues no report request and shows no waiting-for-report state. Controller id stays in `sessionStorage` and is sent as `X-Interview-Controller`. Refresh calls `GET /candidate/interview` and does not POST start. No microphone, SDP, or stage control. Polling stops on a terminal state. Exercise the flow in a browser; the repository has no Playwright suite, so name the substitute if a browser is unavailable. `make test.contract` still passes.

### Blocked sibling integration

- [ ] **T11. Integrate lifecycle commit-or-find and cancel** — Application / infrastructure — L
  - Ticket ID: `CORE-CIN-011`
  - depends: T7, T9 — ref: design §6.3, BR-CORE-041, BR-CORE-042, BR-CORE-045, BR-CORE-046, BR-CORE-047.
  - Blocked by: Q3, Execution Lifecycle BR-CORE-050..055 — ready when that capability provides commit-or-find, read, hard deadline, and cancel as specified in design §6.3.
  - DoD: one assignment yields one execution under a repeated start, including a lost 201. A different idempotency intent conflicts and leaves the original. Session coverage is enforced before the call. Cancel commits `CANCELLED` and GET shows it without reading a report table. A `COMPLETED` execution is not restarted. Revocation during the interview is refused to the candidate; the cancellation commit itself is the lifecycle operation, and this task's evidence identifies that call rather than treating 401 as the cancel. PostgreSQL proof uses `make test.integration` or `make test.docker`. `make test` passes.

- [ ] **T12. Integrate candidate-safe progress and controller fencing** — Application / infrastructure — L
  - Ticket ID: `CORE-CIN-012`
  - depends: T8, T11 — ref: design §6.3, BR-CORE-043, BR-CORE-044.
  - Blocked by: Q4, Interview Runtime BR-CORE-060..071 — ready when runtime supplies the candidate-safe progress object and rejects advances from a controller id that is no longer current.
  - DoD: poll renders only that progress object. Takeover causes a subsequent advance with the previous id to be refused by runtime. The view still has no transcript body, salary, competence flag, or report. This task does not implement stages, voice sessions, or `session-attempts`. `make test` and, where the fence is stored, `make test.integration` or `make test.docker` pass.

- [ ] **T13. Prove the candidate interview slice** — Integration / verification — L
  - Ticket ID: `CORE-CIN-013`
  - depends: T5, T6, T8, T9, T10, T11, T12 — ref: design §3 rule table, §8; tactical-strategy §5.
  - Blocked by: T11 for live start, rejoin, cancel, and the terminal screen; T12 for progress and advance fencing. Consent, coverage, and the page shell do not wait for this task; their evidence is T5, T6, T8, and T10.
  - DoD: disposable PostgreSQL and the candidate page demonstrate the capability examples that are in scope: notice before start, one execution after a lost start, coverage refusal at the 15-minute boundary, second-tab takeover, early cancel shown as cancelled, refresh without a new execution, and a completed interview rendered while no report exists. Record the exact `make test`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.docker` or `make test.integration` commands, skips, and failures. Update design, strategy, and this status to match the run. Do not mark sibling capabilities done.

## Dependency Graph

```text
Q1 (open) → T1 (catalog values; blocks a real acceptance demo)
T2 (domain) ∥ T3 (ports)
T2 → T4 (SQL version 9)
T2 + T3 + T4 → T5 (consent persistence and read)
T3 + T5 → T6 (consent HTTP and page shell)
T2 + T3 + T5 → T7 (orchestration against mocks)
T3 + T4 + T7 → T8 (controller SQL)
T6 + T7 + T8 → T9 (start/poll/takeover/cancel HTTP)
T6 + T9 → T10 (candidate page)
Q3 + T7 + T9 → T11 (lifecycle adapter) [blocked]
Q4 + T8 + T11 → T12 (runtime progress and fence) [blocked]
T5 + T6 + T8 + T9 + T10 + T11 + T12 → T13 (slice proof) [blocked on T11 and T12]
```

T2 and T3 are independent streams, not instructions to spawn agents.

## Revision History

| Version | Date | Change |
|---|---|---|
| v1 | 2026-11-02 | Initial Mode A plan. T1–T13 Todo. T1 waits on the notice text. T11–T13 wait on sibling contracts. No historical status copied from other tripacks. |
