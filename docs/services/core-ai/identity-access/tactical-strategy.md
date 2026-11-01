# Tactical Strategy: Identity and Access

## 1. Implementation Strategy

Mode C, revision after administrative implementation and owner-approved manual-migration policy (decision-log D8 / design §3.1–3.4). Implement the assessed [technical design](technical-design.md) and [BR-CORE-020..039](../business/capabilities/identity-access.md). Preserve T1–T9 Done and Q1 resolved; this review does not rerun or reconstruct their verification. Candidate runtime and migration reconciliation T16–T20 remain planned.

Retain the administrative slice: two tables, bootstrap ADMIN, ADMIN provisioning HR, login/logout, current-session resolution and current HR authority. Existing T9 records supply ACP's access handoff; ACP T8/T9 still own secured configuration HTTP acceptance. A passing access suite is not completion of ACP or the platform.

Reconcile the service's schema installation before adding candidate persistence: inventory the actual Task + administrative schema, add reviewed SQL baselines and an explicit golang-migrate runner, replace startup DDL with a read-only guard, and update all integration/Compose setup callers. GORM remains CRUD-only. New and legacy databases take different paths; existing rows and credentials are preserved. Policy approval is settled, not a new blocking question. No database is migrated by this planning revision.

Prepare candidate identity and access separately. The approved target adds three access-owned tables: candidates, invitations and candidate sessions. T10 settled contact, exchange reuse and assignment-fact collaboration; T12 may implement operations against those ports. T11 still waits for T20 and the real Interview assignment migration. Never install a fake assignment model, treat mocks as assignment integration, or start an execution on exchange. Keep template/draft/publication transaction behavior unchanged.

## 2. Code Reuse Map

These paths were inspected; suitability is assessed rather than inferred from precedent.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `AdminPrincipal`, `AgentConfigurationAuthority`, real adapter | `application/service/agentconfiguration.go`, `infrastructure/service/accessauthority/` | ACP authority integration | Existing administrative behavior retained; SubjectID remains stable account UUID. Migration must preserve live persisted sessions. |
| `Clock` | `application/service/ports.go` | Trusted session/invitation time | Reuse directly; caller canonicalizes UTC milliseconds; controlled equality tests required. |
| `CreateOperation`, `Create.Execute` | `application/usecase/taskusecase/create.go` | Narrow operation/injection/DTO pattern | Suitable application shape; Task validation and input policy not reusable for accounts. |
| Task repository callback and adapter | `application/repository/task.go`, `infrastructure/repository/taskrepository/task.go` | Transaction, explicit-field mapping and technical restoration reference | Reuse pattern only; new purpose-specific access ports and multi-record transactions required. |
| Encapsulated aggregate and deep snapshots | `domain/aggregate/task.go`, `domain/aggregate/agentconfig/` | Domain validity/restoration and pointer ownership pattern | Preserve separate creation/restoration; do not reuse Task or configuration-specific value policy. |
| Strict handler/envelope/localized errors | `delivery/http/handler/taskhandler/handler.go`, `delivery/http/handler/accesshandler/`, `pkg/errs/codes.go`, `translations/` | HTTP parity and safe errors | Administrative behavior exists and is retained; candidate additions remain planned. Schema transition adds no public errors. |
| Database and HTTP resource ownership | `infrastructure/runtime/resources.go`, `http.go` | Shared GORM pool, bounded startup/admission/shutdown | Retain pool/cleanup; replace both startup AutoMigrate calls with schema-version guard before bootstrap/listening. |
| Existing Task and administrative schema definitions | `infrastructure/repository/taskrepository/task.go`, `infrastructure/repository/accessrepository/models.go`, `migrate.go` | Baseline manifest and SQL equivalence | Reuse field/constraint semantics, not AutoMigrate execution. Freeze legacy SQL fixtures for adoption; preserve all five administrative named indexes. |
| Existing administrative store, operations and credentials | `infrastructure/repository/accessrepository/`, `application/usecase/accessusecase/`, `query/accessquery/`, `infrastructure/service/accesscredentials/` | Data-preserving cutover regression | CRUD/transactions and bootstrap remain; remove adapter DDL ownership only. Candidate stores are absent. |
| Integration database helpers and lifecycle cases | `test/integration/repository_test.go`, `access_repository_test.go`, `access_http_test.go`, `lifecycle_test.go` | Replace schema setup; preserve cancellation/cleanup proof | Current helpers run AutoMigrate. Use explicit SQL setup, move DDL cancellation to runner coverage, retain runtime failure cleanup. |
| Dependency/documentation checks and mocks | `test/architecture/`, `.mockery.yml`, `Makefile` | Architecture enforcement, Mockery generation, verification | Register new interfaces; existing ACP authority mock is not a real adapter. |
| Existing REST envelope contract | `docs/api/openapi.yaml`, `docs/api/task.md`, `docs/api/CHANGELOG.md` | Schema/error/security documentation conventions | Current routes remain compatible; access paths added only with real implemented behavior. |

Administrative account/session storage is reused; candidate/invitation/session and assignment stores do not yet exist. No Redis, RabbitMQ, provider client, Task event queue or document artifact adapter is required for migration.

## 3. Net-New Code Inventory

The original access inventory below retains implemented administrative items as T1–T9 baseline; only their candidate extensions remain net-new. Paths are internal choices. Classification reflects responsibility, not a requirement for one file/type per row.

| Item | Layer/component | Package/path | Type | Ownership / notes |
|---|---|---|---|---|
| AdministrativeAccount | Domain | `domain/aggregate/access/` | Aggregate root | Stable subject, role/status, creation/restoration validity; never owns plaintext credentials |
| Candidate | Domain | `domain/aggregate/access/` | Aggregate root | Reusable person identity, contact information; not an administrative account |
| AdministrativeSession | Domain | `domain/aggregate/access/` | Access entity/root for session mutation | Account reference and absolute lifetime/revocation; account authority remains live |
| Invitation | Domain | `domain/aggregate/access/` | Assignment-scoped entity | Changes serialized through assignment collaboration; source credential provenance |
| CandidateSession | Domain | `domain/aggregate/access/` | Access entity | One assignment/invitation binding, absolute lifetime and revocation |
| Role / AccountStatus | Domain | `domain/enum/` | Enums | Account-owned, fixed ADMIN/HR and ACTIVE/DISABLED; no policy tables |
| SessionLifetime / assignment access facts | Domain | `domain/valueobject/` or cohesive `access/` package | Immutable values | Lifetime predicates shared only where contracts agree; copies of mutable timestamp pointers |
| Account/session/candidate/invitation operations | Application | `application/usecase/accessusecase/`, `query/accessquery/` | Operation interfaces + plain implementations | Current authority, trusted Clock, transaction coordination, no provider calls |
| Fixed access policy | Application | `application/service/` | Pure authorization policy | ADMIN account provisioning vs HR business operations; candidate resource scope; unknown operations denied |
| Purpose-specific account/session/candidate/assignment transaction ports | Application | `application/repository/accessport/` | Repository contracts | No generic repository or ORM exposure; preserve assignment-first lock order |
| Credential/assignment access ports | Application | `application/service/` | I/O ports | Hash verification, random credentials, current session and actual assignment facts |
| Safe views | Application | `application/dto/` | Detached DTOs | No password/token digests; candidate context minimal; nullable fields explicit |
| Five persistence models and store | Infrastructure | `infrastructure/repository/accessrepository/` | GORM models/adapters | Separate domain mapping, explicit zero/null writes, indexes/FKs and restoration classification |
| Password/token/CSRF adapter | Infrastructure | `infrastructure/service/accesscredentials/` | Security adapter | Argon2id, high-entropy token digests, bounded costs and failure behavior; no secrets in logs |
| Real ACP authority adapter | Infrastructure | `infrastructure/service/accessauthority/` | Application port implementation | Validate session/account association and current HR authority; no permissive fallback |
| Access handlers and trusted context boundary | Delivery | `delivery/http/handler/accesshandler/`, `delivery/http/middleware/` | Transport adapters | Operation mocks, cookies, strict origin/CSRF, safe status/code translation |
| Invitation landing/exchange interaction | Delivery | Candidate UI integration location chosen with Interview owner | Browser adapter | Remove token from URL/history; no exchange on GET; session recovery policy Q3 |
| Access assembly/bootstrap | Infrastructure / cmd | `infrastructure/runtime/`, `cmd/app/` configuration | Composition / deployment configuration | First ADMIN bootstrap, origin and CSRF secret, phased schema installation, bounded work |
| Task baseline up/down | Infrastructure schema | `migrations/pgsql/000001_tasks.{up,down}.sql` | Versioned SQL pair, proposed version | Existing Task schema only; no business-rule changes; T17 |
| Administrative baseline up/down | Infrastructure schema | `migrations/pgsql/000002_administrative_access.{up,down}.sql` | Versioned SQL pair, proposed version | Existing two administrative tables/checks/FKs/indexes; T17 |
| Candidate identity up/down | Infrastructure schema | `migrations/pgsql/000003_candidates.{up,down}.sql` | Versioned SQL pair, proposed/unallocated | T11 after T10/Q2/Q3/Q4 and completed migration foundation; nonunique contact email |
| Invitation/session up/down | Infrastructure schema | `migrations/pgsql/000004_assignment_access.{up,down}.sql` | Versioned SQL pair, proposed/unallocated | T11 after real assignment migration; reallocate if other features consume versions first |
| Migration command and schema adapter | Infrastructure / cmd | `cmd/migrate/main.go`, `infrastructure/database/schema/` | Explicit DDL runner, guarded baseline and read-only status/guard | T17/T18; pinned golang-migrate v4, release-target bounds, migration locks/timeouts, no credential CLI args |
| Canonical legacy manifest and synthetic fixtures | Docs / integration | `docs/design/database-schema-baseline.md`, `test/integration/testdata/schema/legacy_automigrate.sql` | Schema equivalence record / legacy adoption fixture | T16/T19; exact types/defaults/nullability/checks/indexes/FKs; no real secrets/data |
| Migration runbook and execution wiring | Operations / build | `docs/design/database-migrations.md`, `Makefile`, `compose.yaml`, `Dockerfile`, `scripts/verify_compose.py` | Explicit pre-start command/job and artifact packaging | T18; normal startup does no DDL; verification orchestrator migrates before app startup |
| Migration acceptance suite | Integration / architecture | `test/integration/` and existing architecture suite | Real PostgreSQL adoption/recovery and schema-owner checks | T19; empty/Task-only/full/partial/drift/dirty/timeout/concurrency; no AutoMigrate setup bypass |

Candidate work updates catalogs/mocks/API docs in its owning tasks; migration transition does not change HTTP contracts. Migration metadata is tool-owned and separate from the five access business tables. No SQL seeder is needed: fixed roles/statuses remain enums/checks; initial ADMIN remains approved application bootstrap after schema validation. No account-management UI, assignment schema, consent records, execution storage or retention worker is added by this inventory.

## 4. Branching / PR Strategy

Recommended review order only; no branch, commit or PR creation is authorized by this document.

1. Preserve completed administrative T1–T9 and existing ACP foundations.
2. Schema manifest and policy-document synchronization (T16); then versioned SQL/runner (T17). An intermediate artifacts PR does not establish the transition while AutoMigrate still owns startup.
3. Runtime/test/Compose replacement (T18), real migration/recovery evidence (T19), then coordinated local/target adoption and readiness handoff (T20). Merge/deploy only a verified combination; do not deploy a guard-only binary before preparing its database.
4. Candidate contracts (T10) are Done. Candidate persistence (T11) waits for T20 and actual assignment readiness; domain/application T12 may proceed against the approved ports.
5. Candidate HTTP/browser and real assignment integration (T13–T14), then authorized retention/purge (T15).

Dependency branches/parallel headings do not instruct spawning agents. Split large tasks into smaller stable follow-up IDs if estimates prove inaccurate; preserve existing IDs on revision.

## 5. Testing Strategy

Cases below are acceptance requirements; existing administrative suites and T9 handoff are prior records, not rerun evidence. New migration/candidate checks are proof plans. Generate port/operation mocks through `.mockery.yml` and `make mocks` when interfaces change. Domain tests exercise public behavior; application tests mock ports; HTTP tests mock operations as required by root AGENTS.md. Real schema/adapters need PostgreSQL evidence.

### Unit Tests

- BR-020..023: bootstrap/provision paths, ADMIN-only HR creation, HR-only configuration, valid ADMIN login followed by configuration denial, HR cross-creator scope, unknown operation deny, no caller role accepted.
- BR-024..029: session creation and exact `now == expires_at` refusal; no sliding renewal; current-session logout; revoked/disabled denial; stable SubjectID across sessions; current HR authorization before any replay read; candidate cookies cannot resolve admin principals.
- BR-030..033: one candidate referenced repeatedly; no implicit email merge; invitation/session binding and copied snapshots; wrong assignment/invitation ownership, expiry and revocation; no success before commit.
- BR-034..039: exchange/reload has no execution/consent/provider write; expired session cannot authorize; reissue does not invalidate otherwise-live session; assignment revocation invalidates access; access does not reopen terminal execution.
- Restore corrupt roles, missing identities, inconsistent references, malformed digests/hashes and time order as technical failure. Do not reapply latest new-input policy during read or mutate deadlines.
- Credential adapter: independent random secrets, unique digests, password verify mismatch/unknown-account dummy path, bounded malformed-hash costs, constant-time secret checks, context/rate/concurrency limits, no secret leakage. Password acceptance follows T1 §6.1.
- Delivery: strict schema/limits, wrong-role/unauthenticated distinction, session kind substitution, origin and CSRF boundaries, cookie flags and clearing, no-store responses, no credential echoes; ACP retains its own public codes/ordering.

### Integration Tests

- Migration fresh path: up to version 2 → verify three business tables, current types/constraints/indexes/FKs and metadata → down/up on disposable DB only. Reapply to version 2 with no data or bootstrap reset. Migrations insert no ADMIN or reference rows.
- Legacy adoption: Task-only → guarded version 1 adoption → version 2; populated Task+administrative → guarded version 2 adoption. Compare complete row snapshots/IDs/times and secret equality without printing values; old session remains valid until its original expiry/revocation. Partial schema, changed defaults/types, missing/wrong constraint/index or incompatible metadata must refuse without baseline stamping.
- Runner failures: two migrators serialize; a held lock reaches the bounded timeout; cancelled/injected-failure DDL exits nonzero; dirty state cannot trigger auto-force or app startup. Rehearse the documented restore/forward-repair path. Version/dirty metadata is not checksum evidence; compare immutable release artifacts separately.
- Runtime cutover: DDL-denied application role starts on compatible version; missing/dirty/old/future metadata fails before bootstrap/listen and releases connections. No runtime fallback or integration helper invokes AutoMigrate. Preserve startup/shutdown budgets; test runner DDL cancellation separately from runtime check cancellation.
- Administrative happy path: initialize empty database → bootstrap ADMIN → create HR → HR login → resolve real HR authority → current session → logout → old credential denied. Repeat with existing Task rows and after process restart.
- Administrative failures: concurrent bootstrap, duplicate canonical email provisioning, known and uncertain commit errors, wrong password/disabled account, unavailable database, corrupted restoration, absent session, forged token and valid ADMIN denied configuration. Confirm no cookie is issued on failed session commit.
- Cross-feature proof: current-session role lookup and stable subject with ACP's real authority contract; later ACP T8/T9 prove actual secured endpoints, filtering and publication replay. Access acceptance does not claim those unimplemented consumers passed.
- Candidate happy path with real assignments: one candidate → assignments A/B → two invitations → sessions with separate assignment authority → refresh without another execution. Source invitation expiration alone does not shorten an already-issued session.
- Candidate failures: concurrent reissue, reissue vs exchange, rollback retaining old invitation, cross-assignment composite FK violations, unique-token collision handling, assignment revocation vs exchange/admission, exact expiry, session revocation and persisted state across restart.
- Resolve Q3 before repeated exchange/lost-response/device tests. Test accepted reuse/replacement policy without inventing a consumed column. Real Interview integration must prove active cancellation and completed-assignment no-restart (BR-038/039), not a fake store returning a boolean.
- GORM tests explicitly exercise null/zero writes, bounded joins, same-assignment FK and partial uniqueness, no association side effects or physical soft-delete assumptions. Test against project-selected versions; mocks cannot establish these guarantees.

### Regression Tests

- Existing Task/voice contracts and unauthenticated health/routes remain unchanged.
- ACP's approved payloads, stable subject command identity, salary filtering and error precedence remain unchanged except approved HR-vs-ADMIN authority.
- Startup/shutdown budgets, shared pool cleanup and bootstrap behavior continue to work with read-only schema validation. Rollback binary is migration-aware and tested against retained schema; pre-cutover binaries reintroduce AutoMigrate and are excluded.

### Commands and evidence

During implementation run `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, `make test.security`, and `make test.docker` with the new suites registered. `make check` combines all-package/contract/architecture/lint/security checks. `make test.integration` is an alternative only with an explicitly authorized `TEST_DATABASE_URL`; keep credentials out of output. Missing database suites are skipped, not successful persistence verification. `make docker.verify` is optional broader full-stack work when relevant; access does not depend on optional brokers.

Record command/results, fixture/expected behavior and per-rule evidence. Browser runner/framework must be selected with the Interview UI owner because no candidate access test harness currently exists; T13/T14 add runnable instructions rather than inventing a current Makefile target. Current passing tests cannot establish historical TDD evidence.

T17/T18 add the proposed `make migrate.status`, `make migrate.up MIGRATION_VERSION=2`, `make migrate.baseline`, and guarded `make migrate.down MIGRATION_STEPS=1` commands. They are future commands, not current verification evidence. Integration/Compose verification must exercise these SQL artifacts before app startup; do not report SQL correctness from sqlmock or source checks alone. After this docs-only revision, run the existing architecture checks and Markdown/link consistency checks; no database execution is needed.

## 6. Rollout & Rollback

- Migration rollout: T16–T19 prove the transition on isolated PostgreSQL; T20 rehearses and documents the authorized target path. Stop pre-cutover binaries/schema writers, back up and verify restore, inspect target catalog, then explicitly migrate empty databases or adopt exact Task-only/full legacy schemas. Verify clean target version 2 before starting the new binary. Do not reset normal Compose volumes or auto-stamp drift.
- Administrative rollout: preserve current accounts/sessions and T1 bootstrap semantics; schema validation precedes bootstrap within the existing startup budget. T9 is the recorded ACP access handoff, while configuration routes still require ACP T8/T9. Migration acceptance is additional evidence in T19/T20.
- Candidate rollout: T10 settled Q2/Q3/Q4 as access contracts; deploy candidate identity after T20, then real assignment references, invitation/session constraints, exchange HTTP and minimal UI. Full consent/interview execution stays in Interview's release sequence.
- Observe safe operation/outcome/record IDs, denial categories, expiry, login/exchange failures, rate limiting, bootstrap/schema failure and transaction uncertainty. Never log passwords, cookies, raw link tokens, CSRF values, candidate contacts or HR configuration/report content by default.
- Stop rollout on drift, dirty/unsupported schema version, migration lock/timeout/failure, cross-role or cross-assignment access, missing constraints, credential leaks, duplicate interview initiation, unexpected session invalidation or broken existing routes. Restore a tested migration-aware binary or use reviewed forward repair/backup restore; retain business data and external references. Never make destructive down or unchecked force the routine rollback path.
- Cookie/CSRF secret rotation behavior and credential persistence across restart must be tested; rotating a CSRF signing key may invalidate outstanding CSRF values, while session authority remains persisted. Do not claim rotation preserves every browser action without proof.
- No automatic purge until Q5. Rollback is not deletion of accounts/candidates, reset of subject IDs or removal of referenced invitations. Candidate cancellation remains the assignment owner's responsibility.

## Revision History

| Version | Date | Change |
|---|---|---|
| v1 | 2026-11-01 | Original Mode A strategy; administrative implementation preceded candidate integration |
| v1.1 | 2026-11-01 | Mode C: align T1–T9 baseline, add shared manual-migration artifacts, legacy adoption, runner/runtime reconciliation, verification and recovery; owner policy D8 |
| v1.2 | 2026-11-01 | Record T10 candidate/exchange/assignment contracts; T12 unblocked; T11 still waits T20 and Interview assignment migration |
