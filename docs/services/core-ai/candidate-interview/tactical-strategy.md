# Tactical Strategy: Candidate Interview

## 1. Implementation Strategy

Mode A, additive. Implement the assessed [technical design](technical-design.md) for [BR-CORE-040..047](../business/capabilities/candidate-interview.md). The capability status stays planned until task evidence says otherwise. This plan does not treat current tests as that evidence.

Build consent, the candidate-safe read, the controller, and the candidate page in this service. Call Execution Lifecycle and Interview Runtime through ports. Where those ports have no production adapter, return `CIN_DEPENDENCY_UNAVAILABLE` and write nothing. Do not grow `StartExecution` into a candidate start, and do not add state columns to `interview_executions`.

Schema installation follows Identity and Access D8: one new SQL version, `000009_candidate_interview`, applied with `cmd/migrate` before startup. It follows execution-lifecycle `000008_execution_lifecycle`. GORM 1.25.12 is the CRUD adapter. The notice statement is not seeded. An empty catalog fails closed until Q1 is recorded.

The candidate page is part of the slice (decision D1). Landing navigation is the only access-UI edit, and it runs only after the existing session request succeeds. Admin and HR screens are out of scope. The final screen never requests a report (decision D3).

Release of a real acceptance waits for T1. Release of a live start, cancel, and terminal screen waits for T11. Release of progress text and proof that the previous tab cannot advance waits for T12.

## 2. Code Reuse Map

Paths were inspected. Fit is current behavior, not a claim that the neighbor's design should be copied into this feature.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `CurrentCandidateSession`, `CSRFTokens`, `CredentialCandidate` | `application/service/access.go`, `infrastructure/service/accessauthority/candidate.go` | Resolve the cookie and verify `X-CSRF-Token` | Reuse as-is. Unbound session is already `IA_UNAUTHENTICATED`. CSRF mismatch on interview routes maps to `CIN_FORBIDDEN`. |
| `CandidateSession.Execute` | `application/query/accessquery/candidate_session.go` | Existing session payload and `csrfToken` | The page keeps calling it. Interview GET does not reissue CSRF and does not start an interview. |
| `AssignmentFacts.PermitsSessionAccess` | `application/repository/accessport/assignment.go` | Revocation check | False when access is revoked. Completed assignments still permit the session, which is what refresh of a finished interview needs. |
| `Store.WithAssignment` | `infrastructure/repository/accessrepository/candidate_store.go` | Pattern for `SELECT … FOR UPDATE` on `interview_assignments` | Do not route consent inserts through `AssignmentTx`; that callback also writes invitations and sessions. Duplicate the lock in the interview repository. |
| `SessionLifetime.Current` | `domain/valueobject/sessionlifetime.go` | Exclusive expiry | Coverage uses `ExpiresAt().After(deadline)` so the deadline instant is not covered. Do not retune the two-hour lifetime. |
| `Clock` | `application/service/ports.go`, `infrastructure/service/clock` | Trusted UTC time | Reuse. Persist `accepted_at` truncated to milliseconds. |
| Landing page | `delivery/http/candidatelanding/page.html` | Invitation entry | Add navigation after session success only. Token replacement and exchange stay. |
| Router | `infrastructure/runtime/voice.go` | Dispatch | New interview paths must be matched before the `/candidate/` access prefix. |
| Envelope, origin, strict JSON | `delivery/http/handler/accesshandler/http.go`, `errors.go` | HTTP conventions | Helpers are private. Implement the same status, `no-store`, and timestamp layout in the interview handler. |
| `StartExecution`, `Executions.Insert` | `application/usecase/agentbindingusecase/start_execution.go`, `infrastructure/repository/agentbindingrepository/store.go` | Existing execution pin | Leave in place. Candidate start needs find-or-commit, idempotency, and state, which this type does not do. Tests may insert a pin row as a fixture for controller FK checks. |
| Publication idempotency grammar | `application/usecase/agentconfigusecase/fingerprint.go` `acceptPublicationKey` | Character rule for `Idempotency-Key` | Unexported and publication-specific. Copy the pattern `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$` locally. Do not import the agent-config use case. |
| Error catalog | `pkg/errs/codes.go`, `translations/errors.en.toml`, `translations/errors.id.toml` | Public codes | Add `CIN_` codes. Keep `IA_` session failure. |
| Schema gate | `infrastructure/database/schema/version.go`, `compatible.go`, `artifacts.go` | Startup floor and checksums | Today the constant is 7. This slice moves the floor from 8 to 9 after execution-lifecycle `000008`. No DDL at startup. |
| Declaration rule | `test/architecture/documentation_test.go` | Comments on every authored declaration | Satisfied by writing the comments. No waiver. |
| Task operation shape | `application/usecase/taskusecase/create.go` | Narrow operation interface and injected struct | Shape only. Task policy is unrelated. |

`domain/enum/questionstage.go` is not a dependency. `delivery/http/voicepage` is not a dependency. No browser runner exists in the repository.

## 3. Net-New Code Inventory

Classification follows responsibility. A value object owned by an aggregate is still a value object.

| Item | Layer | Package / path | Type | Ownership / notes |
|---|---|---|---|---|
| `Consent` | Domain | `domain/aggregate/candidateinterview` | Aggregate root | One acceptance. Owns create versus restore. |
| `NoticeVersion` | Domain | same | Value object | Owned by `Consent` and by the catalog port. Exact match. |
| `InterviewController` | Domain | same | Aggregate root | One per assignment. Owns takeover. |
| `ControllerID` | Domain | same | Value object | Owned by `InterviewController`. Separate type from `NoticeVersion`. |
| `SessionCovers` | Domain | same | Predicate of the coverage rule | Not a new stored type. Uses the 15-minute budget constant. |
| `AcceptNotice`, `StartInterview`, `Takeover`, `CancelInterview` | Application | `application/usecase/candidateinterviewusecase` | Operations | Each is an interface plus a plain struct. |
| `GetInterview`, `GetExecution` | Application | `application/query/candidateinterviewquery` | Queries | Screen and poll. |
| Interview DTO | Application | `application/dto/candidateinterview.go` | DTO | Allow-list. No report, salary, or controller id. |
| Consent and controller repositories | Application | `application/repository/candidateinterviewport` | Repository contracts | No GORM handle. |
| `NoticeCatalog`, `ExecutionLifecycle`, `InterviewProgress` | Application | `application/service` | Ports | Catalog is local. Lifecycle and progress are sibling contracts. |
| HTTP handler | Delivery | `delivery/http/handler/candidateinterviewhandler` | Handler | JSON routes and error order. |
| Candidate page | Delivery | `delivery/http/candidateinterview/page.html` | UI | Notice, poll, takeover, confirm-cancel, final outcome. |
| GORM store | Infrastructure | `infrastructure/repository/candidateinterviewrepository` | Adapter | Separate models. Explicit takeover update. |
| Composition | Infrastructure | `infrastructure/runtime/candidateinterview.go` | Wiring | Only runtime imports the new delivery package. |
| `000009_candidate_interview.up.sql` / `.down.sql` | Database | `migrations/pgsql/` | Migration | Two tables. No seed. |
| `CIN_` codes and EN/ID text | Delivery catalog | `pkg/errs/codes.go`, `translations/errors.*.toml` | Errors | With the OpenAPI update. |
| OpenAPI and `docs/api/candidate-interview.md` | Contract | `docs/api/` | Contract | Published with the routes. |

No seeder row. The notice catalog value is configuration recorded by T1, not a migration.

## 4. Branching / PR Strategy

One feature branch is enough if review stays sequential. Prefer commit-sized steps that match the task ids, merged in dependency order so each step boots:

1. T2 domain and T3 ports can land together. They do not change `schema.ReleaseVersion`.
2. T4 raises the floor from 8 to 9, after execution-lifecycle `000008`. Do not merge T4 without the SQL files, checksums, grant notes, and a binary that accepts version 9.
3. T5 consent persistence, then T6 consent HTTP. T7 orchestration can land with mocks before T8 controller persistence. T9 HTTP follows T6, T7, and T8. T10 is the page.
4. T11 and T12 land only after the sibling contracts exist. T13 is the closing evidence commit.

Do not mix admin UI, voice session routes, or report routes into these commits. T1 is a documentation and catalog-value commit once the product owner supplies the notice. It can merge as soon as that text exists; later code should not wait to be written.

## 5. Testing Strategy

Commands are the Makefile targets. `make test` covers domain and application with the race detector. HTTP behavior is tested with mocked operations, which is the project rule for delivery. PostgreSQL is required wherever uniqueness, locks, or rollback matter. A skipped `TEST_DATABASE_URL` run is not database proof. `make test.integration` and `make test.docker` are the runs that count for that proof.

`test/architecture/documentation_test.go` is part of `make test.architecture`. New declarations need comments; the test is the check. Generated mocks keep their generator comments. Regenerate with `make mocks` after interface changes and update `.mockery.yml`.

No RED/GREEN history exists for this feature. Record it only from commands actually run during implementation.

### Unit tests

- `NoticeVersion` and `ControllerID`: empty, too long, leading space, exact example shape, rejected type confusion by using both types in the takeover and accept signatures.
- `Consent`: current version accepted; other version refused; `accepted` false is not a constructor input; restore of a retired version succeeds; acceptance time is copied.
- Coverage: `expiresAt` equal to the deadline refuses; one millisecond after allows; revoked facts are the caller's refusal before the predicate.
- Controller: first claim version 1; takeover increments; stale expected version leaves the previous id; terminal context refuses takeover.
- Application, with Mockery mocks: start does not call lifecycle without current consent; same idempotency key returns the first execution; a different consent id conflicts; GET does not call commit-or-find; cancel is refused for the non-controller; the DTO built from a fixture that contains a salary and a report omits both; empty catalog writes no consent.
- Page and handler: strict JSON, CSRF, 201 then 200 consent replay, action lists, HTML `no-store`, landing navigates only after session success.

### Integration tests

- Happy path, once T11 and T12 exist: accept fixture notice, start, retry the lost start, second tab reads, takeover, cancel, poll shows `CANCELLED`, terminal `COMPLETED` fixture renders with no report request.
- Consent unique race and takeover row lock on disposable PostgreSQL in T5 and T8, using a fixture execution pin so the controller FK is real before lifecycle behavior exists.
- Error cases: coverage equality, stale controller version, stale execution version, execution id from another assignment, dependency port down, ambiguous start.
- Empty catalog: accept returns 503 and the consent table stays empty.

### Regression tests

- `POST /candidate/access/exchange` and `GET /candidate/session` keep their status codes, cookie names, and body fields.
- Access tests still show no execution insert on exchange or session read.
- `GET /voice-session` and `POST /voice-sessions` stay on the voice handlers.
- Administrative and configuration routes stay unchanged.
- `make test.contract` validates the combined OpenAPI file.
- Startup on a clean version-9 database succeeds. Startup on version 8, dirty metadata, or version 10 fails and closes resources.

## 6. Rollout & Rollback

- Rollout: apply version 9 with `make migrate.up MIGRATION_VERSION=9` (or `make docker.migrate`) using `MIGRATION_DATABASE_URL`, confirm clean status, then start the binary whose floor is 9. Grant the runtime `DATABASE_URL` role CRUD on `interview_consents` and `interview_controllers` only. Leave the notice catalog empty until T1; candidates then see a notice-unavailable state and cannot accept. Enable a real demonstration only after T1, T11, and, for progress, T12.
- The candidate page ships behind the same binary. There is no feature flag and no admin switch. D1.
- Monitoring: log consent accept, start refusal, takeover, and cancel with assignment id and execution id. Omit cookies, CSRF, controller ids, notice text, and report bodies. No new tracing stack.
- Rollback trigger: bad consent writes, a start that inserts a second execution, or a candidate view that contains salary or report fields.
- Rollback: keep version 9 tables. Run a binary that still accepts clean version 9 and does not mount the interview routes. Do not start the previous floor-8 binary against that database; `schema.RequireCompatible` requires an exact version. Do not run down on a database that holds real consents. Disposable environments may `make migrate.down MIGRATION_STEPS=1 MIGRATION_DISPOSABLE=yes` and return to version 8.
- Data: consents are historical records. A rollback that drops them is acceptable only on disposable data. Withdrawal and purge stay unanswered (Q2) and are not part of rollback.

T9 execution update (2026-11-02): start, poll, takeover and cancel HTTP routes and OpenAPI are mounted. Production wiring uses the T8 controller store and the fail-closed lifecycle adapter. Delivery proof is Mockery operation tests; composition proof is sqlmock without real execution writes. T10 now supplies the candidate UI; T11/T12 sibling adapters remain future work. Verification and RED evidence: [t9-execution.md](t9-execution.md).

T10 execution update (2026-11-02): the candidate page uses the mounted API actions for consent/start/takeover/cancel, polls every two seconds while live, and renders terminal outcomes immediately. Chrome headless exercises 17 scenarios against API fixtures; no real lifecycle or database flow is claimed. All local Go checks pass; database suites were skipped. Evidence and design assessment: [t10-execution.md](t10-execution.md).
