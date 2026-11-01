# Technical Design: Candidate Interview

Date: 2026-11-02. Mode A — new feature. Authority is [Candidate Interview](../business/capabilities/candidate-interview.md), BR-CORE-040..BR-CORE-047, status **planned**. Nothing in this document is observed enforcement. Proof plans are future checks.

Platform sources that refine the capability, where the later technical design is more specific: [platform technical design](../../../design/core-ai/technical-design.md) sections 7, 9 Candidate, and 10, and [scope freeze Journey A](../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md#journey-a--hr-interviewer). This feature tripack refines that baseline for one slice. It does not replace the platform design.

Business-driven technical decisions are D1–D3 in [decision-log.md](decision-log.md).

T9 implementation evidence (2026-11-02): the four execution HTTP operations in §2 are mounted, `ControllerStore` is composed, and delivery tests use Mockery operation mocks. sqlmock exercises valid start/cancel through real application wiring and confirms `503 CIN_DEPENDENCY_UNAVAILABLE` while lifecycle is unbound. Input/session/CSRF ordering and candidate-safe results are covered by HTTP tests. Wrapped storage validation failures remain technical 500s. This does not establish real lifecycle commits, runtime fencing, or the T10 screen; T11–T13 remain pending. See [t9-execution.md](t9-execution.md).

## 1. Overview

A candidate who already holds an assignment-scoped session reviews one notice, accepts that notice version, enters the assignment's one interview, follows a candidate-safe view, keeps a single controlling tab, may leave early, and sees a final outcome as soon as the interview itself is terminal.

### Goals

- Show the notice and record explicit acceptance of one notice version before any start.
- Ask Execution Lifecycle to commit or find the one execution for the assignment, and refuse a start the current session cannot cover.
- Poll a view limited to this interview, transfer the controlling tab, cancel early, and render the terminal outcome on the candidate screen.
- Ship that candidate screen in this slice.

### Non-Goals

Journey A steps that belong to sibling capabilities stay there.

| Concern | Owner | This slice |
|---|---|---|
| Invitation exchange, candidate cookie, CSRF material, `GET /candidate/session` | Identity and Access, BR-CORE-020..039 | Reuse the current session. Access is unchanged. |
| Execution identity, `CREATED`/`RUNNING`/terminal state, idempotent commit-or-find, cancellation commit | Execution Lifecycle, BR-CORE-050..059 | Call the contracts in §6.3. Implement them only when that capability exists. |
| Stages, questions, transcript, voice provider, microphone, SDP, reconnect | Interview Runtime, BR-CORE-060..071 | Display only a candidate-safe progress summary once runtime supplies one. |
| HR report | Interview Reporting, BR-CORE-080..086 | The final screen has no report field and does not wait. |
| Admin and HR screens, notice editor, Document Analyzer, RabbitMQ, salary configuration | Later or other platform slices | Absent from this slice. D1. |

Stage names in `domain/enum/questionstage.go` (`background`, `screening`, `compensation`, `closing`) are publication vocabulary. This slice does not import them and does not run a stage workflow.

### Verified project context

| Source inspected | Observed fact and design consequence |
|---|---|
| Root `AGENTS.md` | Four root layers. Application-owned ports, operation interfaces, detached DTOs, trusted UTC millisecond time, transactional repository updates, standard JSON envelope, strict request fields, EN/ID errors in `pkg/errs` and `translations/errors.*.toml`. `test/architecture/documentation_test.go` requires a comment on every authored declaration, including unexported ones and interface methods. Schema policy is manual versioned SQL (Identity and Access D8). |
| `go.mod`, `Makefile` | Module `github.com/rfanazhari/amer-prototype`, Go 1.26.10, GORM 1.25.12, PostgreSQL driver 1.5.11, golang-migrate v4.19.0, testify, Mockery. Verification commands are `make test`, `make test.contract`, `make test.architecture`, `make lint`, `make test.integration`, and `make test.docker`. |
| `infrastructure/database/schema/version.go`, `docs/design/database-migrations.md` | Startup accepts only clean `schema.ReleaseVersion` 7 and performs no DDL. Versions `000001`–`000007` are the release stream. |
| `migrations/pgsql/000004_candidates.up.sql` through `000007_interview_executions.up.sql` | `candidates`, `interview_assignments`, `interview_invitations`, `candidate_sessions`, and `interview_executions` exist. `interview_executions` stores `id`, `assignment_id` (unique), `instance_id`, `published_revision_id`, and `created_at` only. No consent, controller, state, or deadline column exists. |
| `delivery/http/candidatelanding/page.html`, `infrastructure/runtime/voice.go`, `delivery/http/handler/accesshandler/candidate.go` | `GET /candidate/access` strips the token and exchanges it. Success stops at "Access ready". Every other `/candidate/` path is dispatched to access. Access handles exchange and `GET /candidate/session` and returns 404 for unknown candidate paths. |
| `docs/api/openapi.yaml`, `docs/api/identity-access.md` | Published candidate paths are `GET /candidate/access`, `POST /candidate/access/exchange`, and `GET /candidate/session`. Cookies are `ia-candidate` / `__Host-ia-candidate`. `GET /candidate/session` returns `candidate`, `assignment.id`, `session`, and `csrfToken`. No interview, consent, execution, takeover, or cancel path is published. |
| `application/service/access.go`, `infrastructure/service/accessauthority/candidate.go`, `application/query/accessquery/candidate_session.go` | `CurrentCandidateSession.Resolve` and `AssignmentAccess.WithAssignment` already bind the cookie to one live assignment. `AssignmentFacts.PermitsSessionAccess` is false when access is revoked. Terminal completion does not, by itself, deny the session. |
| `domain/valueobject/sessionlifetime.go` | `Current` is true only while `now` is strictly before `expires_at`. Equality is expired. Candidate sessions last exactly two hours. |
| `application/usecase/agentbindingusecase/start_execution.go`, `infrastructure/repository/agentbindingrepository/store.go` | `StartExecution` inserts a pin when the caller supplies both ids. `Find` loads by execution id. A second insert for the same assignment fails the unique key and does not return the stored row. The type has no state, consent, idempotency, or candidate session check. It is a configuration-binding helper, not the lifecycle commit-or-find. |
| `domain/enum/questionstage.go` | Four configuration stage names. No runtime workflow. |
| Repository search | No consent table, no candidate interview handler, no `CIN_` error codes, and no Playwright suite. `test/integration/access_candidate_fixture_test.go` `countExecutions` treats table names `consent_records` and `interview_runs` as tables access must not create. This slice uses different names. |

No live database was inspected. No suite was rerun for this planning revision. Backend Service Standard and Design Assessment were applied. Project instructions override the bundled standard where they differ: delivery is tested through mocked operations, repository contracts live in `application/`, and domain state stays encapsulated. GORM remains the CRUD adapter. Schema changes are reviewed SQL applied by `cmd/migrate` before startup.

Binding behavior is the BR rules, the platform candidate routes this slice adopts, consent-before-start, session coverage, candidate-safe fields, one controller, early cancellation, and a final screen with no report. Package layout below is an internal choice. Changing a public code, a stored consent, or a sibling contract requires a resolved decision and synchronized documents before dependent work.

## 2. API / Interface Contract Changes

Additive candidate routes. Identity and Access paths stay as published. Administrative, Task, voice, and health routes stay as published.

| Method and path | Authority | Result |
|---|---|---|
| GET `/candidate/interview/page` | Public HTML shell | The candidate screen. The JSON API enforces the session. |
| GET `/candidate/interview` | Candidate cookie and origin | 200 notice, consent, this execution when one exists, actions, and candidate-safe progress |
| POST `/candidate/consents` | Candidate cookie, origin, `X-CSRF-Token` | 201 first acceptance of the current notice; 200 the same acceptance on retry |
| POST `/candidate/executions` | Candidate cookie, origin, CSRF, `Idempotency-Key` | 201 the execution committed by this request; 200 the execution already committed for the same intent |
| GET `/candidate/executions/{id}` | Candidate cookie and origin | 200 the same candidate-safe execution view; 404 when the id is outside this session's assignment |
| POST `/candidate/executions/{id}/takeover` | Candidate cookie, origin, CSRF | 200 the caller is now the controller |
| POST `/candidate/executions/{id}/cancel` | Candidate cookie, origin, CSRF, current controller | 200 the committed cancellation |

Session scope comes from the existing candidate cookie. The client also sends `X-Interview-Controller`, the tab id kept in `sessionStorage`. That header is how the server tells this tab whether it is the controller. It does not authenticate the candidate. Responses never return the stored controller id.

These platform §9 paths are sibling work and are not registered here: `POST /candidate/executions/{id}/session-attempts` and `POST .../ready`. `GET /candidate/session` and `POST /candidate/access/exchange` stay on the access handler.

One operation interface per command or query, consumed by delivery:

| Operation | Package | Responsibility |
|---|---|---|
| `AcceptNotice` | `application/usecase/candidateinterviewusecase` | Record acceptance of the current notice version |
| `StartInterview` | same | Coverage and consent checks, then lifecycle commit-or-find |
| `Takeover` | same | Replace the controlling tab |
| `CancelInterview` | same | Ask lifecycle to commit cancellation |
| `GetInterview` | `application/query/candidateinterviewquery` | Screen model for this session |
| `GetExecution` | same | One execution inside this assignment |

Implementations are plain structs with injected ports. Results are detached `application/dto` values. The lifecycle and runtime ports are interfaces in `application/service` or `application/repository/candidateinterviewport`. Production adapters that do not exist yet fail closed with a dependency error and write nothing.

Dispatch these paths in `infrastructure/runtime/voice.go` before the `/candidate/` access catch-all, by the same pattern configuration uses before `/admin/`. Leaving them on the access handler returns 404 from `accesshandler.Handler`.

At implementation, update `docs/api/openapi.yaml`, add `docs/api/candidate-interview.md`, and update `docs/api/CHANGELOG.md` together. Map failures through `pkg/errs` and both translation files.

## 3. Data Model & Schema Changes

### Concepts and rules

Language in this slice:

| Term | Meaning here | Distinct from |
|---|---|---|
| Notice | The statement and version the candidate is asked to accept | A published agent revision |
| Consent | One stored acceptance of one notice version for one assignment, with the acceptance time | The candidate session, and any later withdrawal |
| Interview controller | The one tab id allowed to cancel, and later to advance, this live interview | The session cookie |
| Candidate-safe view | Notice, consent, this execution's state and actions, and progress runtime marks as visible to the candidate | Salary, other candidates, admin configuration, the HR report |
| Accepted interview | Platform state `CREATED`: identity is committed and voice work has not started | `RUNNING` |

`Candidate session` and `Agent Execution` keep the meanings fixed by Identity and Access and Execution Lifecycle. This slice stores consent and the controller. It does not become the owner of session expiry rules or execution state.

`Consent` is an aggregate root. Its identity is the consent id. It is created once for a pair `(assignment, notice version)` and is not edited. Uniqueness is the business rule "the company can point back to the acceptance of that version," so a retry returns the original id and time. Several versions may be retained when the current notice changes; only an acceptance of the **current** catalog version allows start.

`InterviewController` is a separate aggregate, one per assignment. Takeover replaces the tab id and increments a version. It references an execution id and does not own that execution's state. The two concepts change for different reasons: notice policy changes consent; a second tab changes the controller.

`NoticeVersion` and `ControllerID` are distinct value objects with the same engineering character rule. A controller id must not be usable as a notice version through a shared type. The character rule itself is one private parser in the candidate-interview domain package.

Session coverage is a domain predicate, `expiresAt.After(deadline)`. It uses the shipped session rule that equality is already expired (`SessionLifetime.Current`). The hard-deadline budget is 15 minutes, from platform §10, measured from `RUNNING`. HR stage-second fields are a different rule and are not reused.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| BR-CORE-040 | `Consent` plus `StartInterview` | Accept writes a row only for the current version. Start reads that row and returns before the lifecycle port when it is missing. GET offers accept until then. | Public domain tests; application test with a lifecycle mock that must not be called; HTTP mapping. Planned, not observed. |
| BR-CORE-041 | `StartInterview` calling the lifecycle port in §6.3 | First commit and same-intent retry. This slice does not insert `interview_executions`. | Mock returns one id for both calls. PostgreSQL proof waits for T11. `StartExecution` is not the proof. |
| BR-CORE-042 | Coverage predicate in start and continue | Start uses `now + 15m`. Continue uses the lifecycle hard deadline once `RUNNING`, and `now + 15m` while still `CREATED`. | Equality refuses; 15m00s remaining allows; 14m59s remaining refuses. Revoked assignment refuses through existing `PermitsSessionAccess`. |
| BR-CORE-043 | Interview DTO and HTTP encoder | Build the view from an allow-list. Controller id, salary, other candidates, configuration bodies, and report fields have no DTO field. | Encoder test over a hostile fixture. OpenAPI schema has no those properties. |
| BR-CORE-044 | `InterviewController` | Second tab reads with `isCurrent=false`. Takeover locks the row, matches `expectedVersion`, stores the new id. Runtime must refuse advances from any other id (§6.3). | Domain transition tests; PostgreSQL conflict test in T8. Advance-fencing proof is T12 and stays blocked. |
| BR-CORE-045 | `CancelInterview` | Only the current controller, only while non-terminal, calls lifecycle cancel with reason `candidate_early_stop`. The following GET shows `CANCELLED`. | Mock commit then view. UI labels it cancelled. Live commit is T11. |
| BR-CORE-046 | `GetInterview` | Reload is a read. It does not call commit-or-find. An existing execution id is returned unchanged. | Application test asserts the lifecycle port is not called. |
| BR-CORE-047 | Screen model and candidate page | Terminal `COMPLETED`, `FAILED`, or `CANCELLED` is renderable immediately. No report status, no report poll, no dependency on reporting. | DTO and page tests with a terminal fixture and no report. Live terminal state is T11. |

### Storage

Proposed pair: `migrations/pgsql/000009_candidate_interview.up.sql` and `000009_candidate_interview.down.sql`. This pair follows execution-lifecycle `000008_execution_lifecycle`. Apply it only after a clean version 8. Applied versions 1–8 stay immutable. T4 confirms `000009` is free before writing files.

| Table | Columns and constraints | Why this table |
|---|---|---|
| `interview_consents` | `id uuid` PK; `assignment_id uuid` NOT NULL FK `interview_assignments(id)` ON DELETE RESTRICT; `notice_version text` NOT NULL CHECK `^[A-Za-z0-9][A-Za-z0-9._:-]{0,63}$`; `accepted_at timestamptz` NOT NULL; `candidate_session_id uuid` NOT NULL FK `candidate_sessions(id)` ON DELETE RESTRICT; unique `(assignment_id, notice_version)` | Acceptance exists before an execution and must survive a later notice version. The session id is provenance of who accepted, not the uniqueness key. |
| `interview_controllers` | `assignment_id uuid` PK FK `interview_assignments(id)` ON DELETE RESTRICT; `execution_id uuid` NOT NULL FK `interview_executions(id)` ON DELETE RESTRICT; `controller_id text` NOT NULL with the same character check; `version bigint` NOT NULL CHECK `>= 1`; `updated_at timestamptz` NOT NULL | One live controller. The execution pin table is left for lifecycle state. |

Index `interview_consents (assignment_id)`. No soft delete, no cascade, no notice-body column, no report table, and no new column on `interview_executions`.

GORM 1.25.12 maps separate models in `infrastructure/repository/candidateinterviewrepository`. Consent is insert-only. Takeover updates `controller_id`, `version`, and `updated_at` with an explicit field list inside `SELECT … FOR UPDATE`, so zero-value update omission cannot skip a write. No association preload and no `AutoMigrate`. A unique violation aborts the PostgreSQL transaction; the loser opens a new read and returns the committed row when the version matches. It does not continue on the aborted transaction.

Consent transaction: lock `interview_assignments` by id, re-read revocation, re-read the candidate session, insert. This is a new repository transaction. It does not call `AssignmentAccess.WithAssignment`, because that callback can also revoke invitations and insert sessions. Both transactions lock the assignment row first, so they serialize. Start does not hold that lock across the lifecycle call. Lifecycle locks assignment before execution on its own side.

Forward: `make migrate.up MIGRATION_VERSION=9` after a clean version 8. Raise `schema.ReleaseVersion` to 9, extend the checksum manifest and embedded `checksums.json`, grant the runtime role CRUD on the two new tables and keep SELECT on `schema_migrations`, and update `docs/design/database-migrations.md`. Startup still performs no DDL. The exact-version guard means a binary with floor 8 will not start on version 9.

Down drops `interview_controllers` then `interview_consents`. It does not drop assignments, sessions, or executions. Down is disposable only. Populated rollback keeps the tables and uses a binary that accepts clean version 9 with the new routes unmounted. The version-8 binary is not a routine rollback target.

### Seed / reference data

No migration seed and no seeder. The notice statement is not a row. The current notice is a developer-owned catalog value supplied when Q1 is answered: version string plus statement text, injected at composition. Until that value exists, the catalog is empty and accept/start fail closed with `CIN_NOTICE_UNAVAILABLE` before any insert. An empty catalog is safe. Inventing `notice-2026-11-02` or any statement would publish a company notice the product owner has not supplied. The capability example is an illustration, not the catalog.

| Target | Stable key | Proposed values | Purpose / status | Environments |
|---|---|---|---|---|
| In-memory notice catalog | `notice version` | `notice-261102` and the statement in section 6 | Required before a real candidate can accept. Source: product owner, Q1 (resolved). | All. |

Open data decision Q1 blocks T1 and any demonstration that records a real acceptance. It does not block the schema, the fail-closed operations, or tests that inject a fixture notice.

### Creation, restoration, and transition

| Path | Accepted inputs and normalization | Errors and timing | Side effects | Proof plan |
|---|---|---|---|---|
| Create consent | Current catalog version, explicit `accepted: true`, live session, assignment not revoked. No trim and no case folding. Blank or extra JSON fields fail first. | Unknown field or `accepted: false` is 400 before a write. Empty catalog is 503 before a write. Revoked or expired session is 401 before a write. | One insert, or the existing row for the same assignment and version. No execution write. | Valid, mismatch, retry, concurrent unique violation, rollback. |
| Restore consent | Any stored row whose id, assignment, version pattern, session id, and `accepted_at` are present. The version need not still be current. | Corrupt row is a technical error, not an input error, and is not rewritten on load. | None | Round trip of a retired version; load does not call the catalog. |
| Start | Stored consent id for the current version; session expiry strictly after the coverage deadline; `Idempotency-Key` grammar `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$` (same character rule as publication keys, local to this use case). | Consent failure and coverage failure return before the port. Port absence returns 503 and writes no consent change. | Lifecycle commit-or-find only. | Mock call counts; equality boundary. |
| Takeover | Live execution, `expectedVersion`, new controller id equal to the header. | Stale version conflicts and leaves the previous id. Terminal execution refuses. | One row update. | Two concurrent expected versions; one winner. |
| Cancel | Current controller, execution `expectedVersion`, non-terminal. | Non-controller is 403 with no port call. | Lifecycle cancel with fixed reason `candidate_early_stop`. | Repeat after terminal does not relabel `COMPLETED` as `CANCELLED`. |
| Read | Current session. | Missing cookie is 401. Other assignment's execution id is 404. | No start, no consent insert. | Reload mock has zero lifecycle calls. |

## 4. Component Breakdown

| Layer / component | Status | Justification |
|---|---|---|
| `domain/aggregate/candidateinterview` | New | Consent, controller, notice version, controller id, coverage predicate. |
| `domain/aggregate/access`, `domain/valueobject/sessionlifetime.go` | Reused as-is | Session identity and exclusive expiry. Coverage calls `ExpiresAt` and compares with `After`. |
| `domain/enum/questionstage.go` | Reused as context only | Stage names stay configuration data. This feature does not import the package. |
| `application/usecase/candidateinterviewusecase`, `application/query/candidateinterviewquery` | New | Commands and the screen read. |
| `application/dto/candidateinterview.go` | New | Detached views. Existing DTO package; new file. |
| `application/repository/candidateinterviewport` | New | Consent and controller persistence contracts. |
| `application/service` candidate-interview ports | New | Notice catalog, lifecycle admission, runtime progress. Narrow interfaces. |
| `application/service/access.go` `CurrentCandidateSession`, `CSRFTokens` | Reused as-is | Cookie binding and CSRF. Credential kind stays `CredentialCandidate`. |
| `application/usecase/agentbindingusecase.StartExecution` | Reused as-is by its current callers | Pin insert. Candidate start does not call it. |
| `delivery/http/handler/candidateinterviewhandler` | New | JSON routes, envelope, errors. |
| `delivery/http/candidateinterview` | New | Embedded candidate page. |
| `delivery/http/candidatelanding/page.html` | Modified | After a successful `GET /candidate/session`, navigate to `/candidate/interview/page`. Exchange, token stripping, and cookie flags stay as they are. |
| `delivery/http/handler/accesshandler` | Reused as-is | Exchange and session routes stay here. |
| `infrastructure/repository/candidateinterviewrepository` | New | GORM models and transactions. |
| `infrastructure/runtime/voice.go`, `infrastructure/runtime/runtime.go` | Modified | Register the interview handler before the access catch-all and compose ports. |
| `infrastructure/database/schema` | Modified | Floor 8, checksums, grants. |
| `docs/api/openapi.yaml`, translations, `pkg/errs` | Modified | Publish the contract and `CIN_` codes. |
| Admin/HR UI, voice page, report handlers | Out of slice | D1 and D3. |

### Detail

**Domain.** `NewConsent` requires a current-version match supplied by the caller, a non-zero acceptance time, and ids. `RestoreConsent` checks stored shape and copies the time pointer. `InterviewController.Takeover` returns a new value with the next version. `SessionCovers(expiresAt, deadline time.Time) bool` is `expiresAt.After(deadline)`. The 15-minute budget is a named constant cited to platform §10.

**Application.** Each operation resolves the candidate session, then applies one rule. `GetInterview` assembles notice, consent, and, when the lifecycle read port returns an execution, the safe execution. A missing execution is an empty execution field, not an error. Progress is copied only from the runtime port's candidate-safe struct. A nil runtime port yields progress `null` and still returns the rest of the view.

**Delivery.** Match access handler behavior: method check, origin, strict JSON, 1 MiB body limit, `Cache-Control: no-store`, millisecond UTC timestamps `2006-01-02T15:04:05.000Z`. The access `formatted` helper is unexported; this handler formats the same layout locally. HTTP tests use mocked operations.

**Candidate page.** States this slice renders from the API: access error, notice/consent, ready to start, in progress (`CREATED` or `RUNNING`), cancelled, completed, failed. Connecting, stage transition, interrupted, and finalizing appear only as labels when runtime later sets them on the safe view. The page does not open a microphone or peer connection. Stop asks for confirmation, then POSTs cancel. Poll `GET /candidate/interview` every 2 seconds while the state is `CREATED` or `RUNNING`, then stop. The interval is a page choice. Controller id is generated once per tab and stored in `sessionStorage`.

**Landing handoff.** One navigation after the existing session fetch succeeds. A failed exchange still shows the current unavailable message.

### Material structure decisions and assessment

| Choice | Reason and material alternatives | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| Consent aggregate separate from `CandidateSession` | BR-CORE-040 and Identity and Access D4: a session is access, not agreement. Folding consent into the session would make exchange or refresh a place where acceptance can be skipped, and would couple notice policy to credential lifetime. | Binding rule. The row layout can move; the separate acceptance record cannot. | Retained |
| Own `interview_consents` rather than columns on `interview_assignments` | The assignment row is the pin (`published_revision_id`, revocation, completion). Several notice versions must remain addressable. A single nullable column would destroy the previous acceptance when the notice changes. | Identity and Access D8 plus the observed assignment migration. | Retained |
| Own `interview_controllers` rather than columns on `interview_executions` | The execution table is a pin with no state. Lifecycle will own state and deadline columns. Controller version changes when a tab changes, which is a different reason for change than execution transition. | Binding ownership split with BR-CORE-050..059. FK to `interview_executions(id)` is allowed because that table exists. | Retained |
| Notice catalog in composition, empty until Q1 | D1 excludes an HR notice editor, so a table and admin API have no current writer. A seeded statement would invent the open product text. Fail closed is the ordinary path when the catalog is empty; a fixture catalog is the test path. | Q1 may later choose an HR-editable notice. That is a new decision, not a silent table. | Retained mechanism; catalog values deferred |
| Candidate start calls a lifecycle port and does not call `StartExecution` | Walk of `StartExecution.Execute`: caller picks the execution id, insert copies the pin, duplicate assignment id errors, no read-back, no consent, no idempotency. Calling it from the candidate handler would create a second admission path and still miss BR-CORE-041. The alternative of teaching `StartExecution` to find-or-create would move lifecycle rules into the binding use case. | Sibling contract §6.3. This slice may keep the port and a failing adapter. | Retained |
| Routes registered ahead of the access catch-all | `Router.ServeHTTP` sends every `/candidate/` path except the exact landing path to access, and access 404s unknown paths. A new access case would mix interview policy into the access handler. | Explicit current router constraint. | Retained |
| JSON `GET /candidate/interview` and HTML `/candidate/interview/page` | Platform §9 assigns the JSON contract to `GET /candidate/interview`. The landing already separates HTML `/candidate/access` from JSON exchange. Content negotiation on one path would make the contract depend on `Accept`. | Platform path is binding. The HTML path is internal and may be renamed if both documents change together. | Retained |
| Reuse candidate CSRF and return `IA_UNAUTHENTICATED` for a bad session; use `CIN_` for interview policy | The cookie and `service.ErrCandidateSessionUnbound` already map to 401 `IA_UNAUTHENTICATED`. A second unauthenticated code would split one cookie failure. Interview refusals are a different actor-facing reason. | Codes become binding once OpenAPI is published. | Retained |
| 15-minute constant beside session expiry, not HR `stageSeconds` | Platform §10 fixes the hard deadline. Stage budgets are per-stage publication limits of 300–600 seconds. Sharing them would let an HR draft change how long a session must cover. | Platform limit. A later revision of the default updates this constant and its tests together. | Retained |
| Consent withdrawal has no column and no route | Platform §10 and the capability leave withdrawal open. A `withdrawn_at` column would force a behavior for interviews already underway. | Q2. No task implements it. | Deferred, excluded |

Domain assessment applies. The core behavior to protect is consent-before-start, one execution request, one controller, and a final screen that stands without the report. Session crypto, cookies, and execution pins stay supporting structure and are reused.

## 5. Flow Sequence

1. The candidate opens the invitation. The existing landing exchanges it and reads `GET /candidate/session`. This slice then opens `/candidate/interview/page`.
2. The page calls `GET /candidate/interview`. With no consent, the response contains the notice and action `accept`.
3. The candidate submits `accepted: true` and the notice version. The server stores consent and returns its id and time.
4. The page shows start only when the action list contains `start`. Start sends the consent id and an `Idempotency-Key`.
5. The operation checks current consent and session coverage, then calls lifecycle commit-or-find. A lost response repeats the same key and consent id and receives the same execution.
6. The page polls `GET /candidate/interview`. A second tab polls and sees `isCurrent: false`. Takeover sends that tab's id and the controller version.
7. Early stop confirms in the page, then POSTs cancel. The next poll shows `CANCELLED`.
8. When the state is terminal, the page renders that outcome and stops polling. It does not request a report.

Refresh repeats step 2. It does not repeat step 5.

## 6. Request / Response

Timestamps are UTC with millisecond precision. Unknown JSON fields are rejected. System-owned ids and timestamps are rejected as request fields.

### GET `/candidate/interview`

```json
{
  "status": "success",
  "data": {
    "notice": { "version": "<approved-version>", "statement": "<approved-statement>" },
    "consent": { "id": "…", "noticeVersion": "<approved-version>", "acceptedAt": "2026-11-02T08:05:00.000Z" },
    "execution": null,
    "actions": ["accept"],
    "progress": null
  }
}
```

`notice` is null when the catalog is empty, and `actions` is then empty. `consent` is null until the current version has been accepted. A stored acceptance of an older version does not populate `consent` and does not enable start. `execution` is null until lifecycle has an execution for this assignment.

When an execution exists:

```json
"execution": {
  "id": "…",
  "state": "CREATED",
  "version": 1,
  "hardDeadline": null,
  "isCurrentController": false,
  "controllerVersion": 1
}
```

`state` is `CREATED`, `RUNNING`, `COMPLETED`, `FAILED`, or `CANCELLED`. `hardDeadline` is present once the execution is `RUNNING`. `progress` is either null or the object defined by the runtime contract below. `actions` is a subset of `accept`, `start`, `takeover`, `cancel`.

| Situation | Actions |
|---|---|
| Current notice not accepted | `accept` |
| Accepted, no execution, session covers the budget | `start` |
| Accepted, session does not cover the budget | none |
| `CREATED` or `RUNNING`, this tab is not the controller | `takeover` |
| `CREATED` or `RUNNING`, this tab is the controller | `cancel` |
| Terminal | none |

### Notice catalog value

Resolved Q1 (product owner, 2026-11-02). The catalog is a composition constant in `infrastructure/runtime/candidatenotice.go`, not a database row or seeder (D1). Version: `notice-261102`. A statement change requires a new version.

Statement:

```text
Pemberitahuan dan Persetujuan Interview

Interview ini dilakukan secara lisan dengan agen AI dan berlangsung maksimal 15 menit. Dengan menekan "Saya setuju", Anda menyatakan memahami dan menyetujui hal berikut:

1. Suara Anda direkam dan diubah menjadi transkrip selama interview berlangsung.
2. Rekaman, transkrip, dan hasil penilaian dipakai hanya untuk proses seleksi pada lowongan yang Anda lamar.
3. Hasil interview hanya dapat dilihat oleh tim rekrutmen yang berwenang. Kandidat lain tidak dapat melihatnya.
4. Anda dapat menghentikan interview kapan saja dengan tombol berhenti. Interview yang dihentikan dicatat sebagai dibatalkan.
5. Persetujuan ini tercatat bersama versi pemberitahuan dan waktu Anda menyetujuinya.

Jika Anda tidak setuju, tutup halaman ini dan hubungi tim rekrutmen.
```

### POST `/candidate/consents`

Body exactly `{ "noticeVersion": "<version>", "accepted": true }`. Response `data`: `{ "id", "noticeVersion", "acceptedAt" }`. 201 on insert, 200 when that assignment already has that version.

### POST `/candidate/executions`

Header `Idempotency-Key` required. Body exactly `{ "consentId": "<uuid>" }`. Response `data` is the execution object above. 201 when this call committed it, 200 when it was found.

### POST takeover

Body exactly `{ "controllerId": "<id>", "expectedVersion": 1 }`. `controllerId` must equal `X-Interview-Controller`. Response `data`: `{ "isCurrentController": true, "controllerVersion": 2 }`.

### POST cancel

Body exactly `{ "expectedVersion": 1 }` where the number is the execution version. The caller must be the current controller. Response `data` is the execution object with `state: "CANCELLED"`.

### Page behavior

`GET /candidate/interview/page` returns `text/html`, `no-store`, and `Referrer-Policy: no-referrer`. The script uses `credentials: "include"`. After cancel confirmation, local capture is stopped by leaving any future runtime media handle unused; this page does not create one. Superseded poll responses are ignored when a newer response has already been rendered.

### 6.3 Sibling contracts

These are requirements on other capabilities. T11 now binds the lifecycle interface through `infrastructure/service/candidatelifecycle`; Interview Runtime remains pending for T12. Lifecycle keeps transaction and transition ownership.

**Execution Lifecycle** must provide:

| Operation | Input | Result this slice relies on |
|---|---|---|
| Commit or find | Assignment id, idempotency key, intent fingerprint of assignment id + consent id | One execution id, `CREATED` on first acceptance, `created` boolean, execution version, pinned revision unchanged by later publication. Same key and same intent returns the original, including after the state has moved. Same key and a different consent id conflicts. No second row for the assignment. |
| Read | Assignment id from the session | The execution above or a not-found that this slice renders as `execution: null`. |
| Hard deadline | While `RUNNING` | `hardDeadline = runningStartedAt + 15 minutes`, stable across restart. |
| Cancel | Execution id, expected version, reason `candidate_early_stop` | Committed `CANCELLED`, or conflict when the version is stale or the execution is already terminal. The first terminal commit wins. |
| Access ended during a non-terminal interview | Revocation or session expiry | Lifecycle commits `CANCELLED` (BR-CORE-055). This slice only refuses further candidate commands. Refusal is not that cancellation. The HR revocation command is Identity and Access `CORE-IAM-022`, not a task in this plan. Session expiry during a run remains Identity and Access Q7. |

**Interview Runtime** must provide:

| Operation | Result this slice relies on |
|---|---|
| Candidate-safe progress | Optional summary the candidate may see: current phase label and whether the interview is connected, reconnecting, or finalizing. No transcript body, no salary, no competence judgement, no report. |
| Controller fence | An advance, including a later session attempt, is accepted only for the current `controller_id`. Takeover makes the previous id unable to advance. |
| Completion | Runtime moves the execution to `COMPLETED` or `FAILED` through lifecycle. This slice displays the resulting state. |

**Interview Reporting** has no required call. The candidate view stays valid while a report is absent, pending, or failed.

`POST /candidate/executions/{id}/session-attempts` and `.../ready` remain runtime routes. They are how microphone readiness and voice activation will work. They are outside this OpenAPI addition.

## 7. Error Handling

Interview policy uses a `CIN_` prefix. Session absence stays `IA_UNAUTHENTICATED` because it is the same cookie failure `GET /candidate/session` already returns. Add codes to `pkg/errs/codes.go` and both translation files. Messages name the candidate action and omit cookies, notice body, salary, execution internals, and other candidates.

Precedence for every new route:

1. Wrong method: 405 `METHOD_NOT_ALLOWED`.
2. Origin or content type: the existing access origin failure, then 400 `INVALID_INPUT` for malformed JSON, unknown fields, or a bad idempotency key.
3. Missing, expired, revoked, or unbound candidate session: 401 `IA_UNAUTHENTICATED`.
4. CSRF mismatch on a POST: 403 `CIN_FORBIDDEN`, before any consent or execution body is returned.
5. Execution id outside this assignment: 404 `NOT_FOUND`.
6. Policy: 409 `CIN_CONSENT_REQUIRED`, 403 `CIN_SESSION_COVERAGE`, 409 `CIN_IDEMPOTENCY_CONFLICT`, 409 `CIN_CONTROLLER_CONFLICT`, 403 `CIN_NOT_CONTROLLER`, 409 `CIN_TERMINAL`, 409 `CIN_NOT_LIVE`.
7. Empty notice catalog or sibling port not implemented: 503 `CIN_NOTICE_UNAVAILABLE` or `CIN_DEPENDENCY_UNAVAILABLE`, with no new consent and no claim that an execution exists.
8. Ambiguous lifecycle commit: 503 `CIN_START_UNCERTAIN`. The client retries the same idempotency key.
9. Corrupt storage: 500 `INTERNAL_ERROR`.

`accepted: false` is malformed input, not withdrawal. Coverage failure tells the candidate to enter again through an invitation that is still valid. The server does not exchange a new invitation.

## 8. Constraints Checklist

- [ ] Rules remain planned until the task evidence exists. Wording in code comments must not say the rules are already enforced by current startup.
- [ ] Access exchange, cookie flags, CSRF issuance, and `GET /candidate/session` stay on their current handlers.
- [x] Candidate UI is in this slice. No admin or HR page, route, or notice editor. T10 browser fixture evidence: [t10-execution.md](t10-execution.md).
- [ ] Consent is stored before start. Start cannot imply acceptance.
- [x] The final screen renders the terminal interview state and does not read or wait for a report. T10 proves UI behavior with API fixtures; live lifecycle proof remains T11/T13.
- [ ] No stage workflow, transcript store, voice provider call, Document Analyzer path, or RabbitMQ client.
- [ ] Schema change is `000009` SQL only. Runtime `AutoMigrate` stays absent. Floor becomes clean version 9 in the same change.
- [ ] GORM is the CRUD adapter. Models stay in infrastructure.
- [ ] Every new Go declaration is documented so `test/architecture/documentation_test.go` passes. Tests live in `_test` packages. Mocks come from `make mocks`.
- [ ] `StartExecution` remains the pin helper it is today.

## 9. Risks & Open Questions

| ID | Question | Owner | Blocks | Ready when |
|---|---|---|---|---|
| Q1 | What notice version and statement text are shown? Resolved 2026-11-02: `notice-261102` and the statement in section 6. | Product owner | None | The version matches §3's character rule and the statement is the approved text, recorded in §6 and the catalog. |
| Q2 | What happens if the candidate withdraws consent after start? | Product and data owner | No current task. Withdrawal stays unimplemented. | A later decision adds a task. This plan does not add `withdrawn_at`. |
| Q3 | Resolved for T11: existing lifecycle operations and `agent_executions` support start/read/cancel; deadline derives from persisted `activated_at + 15m`. `interview_executions` remains a pin. | Execution Lifecycle capability | T11 completed; T13 still requires T12 | See [T11 evidence](t11-execution.md); sibling capability status is unchanged. |
| Q4 | Runtime candidate-safe progress and controller fencing are not implemented. | Interview Runtime capability | T12, and the progress portion of T13 | The §6.3 runtime operations exist. |
| Q5 | `countExecutions` in the access integration fixture counts table presence for `interview_executions`, which version 7 already creates. | Access tests, pre-existing | None in this plan, provided new tables are not named `consent_records` or `interview_runs` | Leave that assertion to its owner unless a candidate-interview test run shows it failing for a reason this slice introduced. |

Implementation risks: the exact schema floor makes version 8 and version 9 binaries mutually unable to boot. Polling without a rate limit can add read load; the page interval is the only bound in this slice. A candidate can open the HTML page without a cookie and must see the access-error state rather than a notice.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial design | Mode A plan for BR-CORE-040..047. Rules are not enforced. |
| v2 | 2026-11-02 | Assign `000009_candidate_interview` and clean floor 9 | assign a unique migration version after the parallel tripack collision on 000008 |
| v2.1 | 2026-11-02 | Point the access-ended cancellation caller at Identity and Access `CORE-IAM-022` | Owner approved that command outside this slice. No new candidate-interview task |

T11 execution update (2026-11-02): the production composition binds lifecycle start/read/cancel. Candidate intent uses a fixed `candidate-start` command key per assignment and typed SHA-256 references for the client idempotency key and consent id, preserving the 128-character API key within lifecycle's 64-character token limit. A call-local minted execution id identifies a successful first commit; replay and race reconciliation report `created=false`. A store decorator refuses terminal cancel inside the existing locked callback; the lifecycle operation still performs the transition and commits. Interview credential reads join assignment revocation without changing access routes. PostgreSQL and mocked evidence, assessment and limits are in [t11-execution.md](t11-execution.md). T12 progress/fencing and T13 remain pending. No business decision changed.
