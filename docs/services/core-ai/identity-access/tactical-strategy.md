# Tactical Strategy: Identity and Access

## 1. Implementation Strategy

Mode A, additive access capability in the existing modular monolith. Implement the assessed [technical design](technical-design.md) and [BR-CORE-020..039](../business/capabilities/identity-access.md). Q1 is resolved in T1. Remaining work is planned; no access runtime implementation or tests have been executed.

Deliver the administrative slice first: two tables, bootstrap ADMIN, ADMIN provisioning HR, login/logout, current-session resolution and current HR authority. It supplies the real access dependency for ACP Q2 after T9 evidence; ACP T8/T9 still own secured configuration HTTP acceptance. A passing access suite is not completion of ACP or the platform.

Prepare candidate identity and access separately. The approved target adds three access-owned tables: candidates, invitations and candidate sessions. Candidate schema/flows depend on settled contact/exchange policy and actual assignment ownership. Never install a fake assignment model, treat mocks as assignment integration, or start an execution on exchange. Keep template/draft/publication transaction behavior unchanged.

## 2. Code Reuse Map

These paths were inspected; suitability is assessed rather than inferred from precedent.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `AdminPrincipal`, `AgentConfigurationAuthority` | `application/service/agentconfiguration.go` | ACP authority integration | Trusted principal/current operation authorization already separated; no implementation or role policy. SubjectID remains stable account UUID. |
| `Clock` | `application/service/ports.go` | Trusted session/invitation time | Reuse directly; caller canonicalizes UTC milliseconds; controlled equality tests required. |
| `CreateOperation`, `Create.Execute` | `application/usecase/taskusecase/create.go` | Narrow operation/injection/DTO pattern | Suitable application shape; Task validation and input policy not reusable for accounts. |
| Task repository callback and adapter | `application/repository/task.go`, `infrastructure/repository/taskrepository/task.go` | Transaction, explicit-field mapping and technical restoration reference | Reuse pattern only; new purpose-specific access ports and multi-record transactions required. |
| Encapsulated aggregate and deep snapshots | `domain/aggregate/task.go`, `domain/aggregate/agentconfig/` | Domain validity/restoration and pointer ownership pattern | Preserve separate creation/restoration; do not reuse Task or configuration-specific value policy. |
| Strict handler/envelope/localized errors | `delivery/http/handler/taskhandler/handler.go`, `pkg/errs/codes.go`, `translations/` | HTTP parity and safe errors | Private helpers are Task-specific; local parity or evidence-backed extraction. New access codes/EN-ID entries planned. |
| Database and HTTP resource ownership | `infrastructure/runtime/resources.go`, `http.go` | Shared GORM pool, bounded startup/admission/shutdown | Add staged migrations/wiring; no new global auth requirement, worker or lifetime budget. |
| Dependency/documentation checks and mocks | `test/architecture/`, `.mockery.yml`, `Makefile` | Architecture enforcement, Mockery generation, verification | Register new interfaces; existing ACP authority mock is not a real adapter. |
| Existing REST envelope contract | `docs/api/openapi.yaml`, `docs/api/task.md`, `docs/api/CHANGELOG.md` | Schema/error/security documentation conventions | Current routes remain compatible; access paths added only with real implemented behavior. |

No current account, candidate, invitation, session or assignment store can be reused directly. No Redis, RabbitMQ, provider client, Task event queue or document artifact adapter is required for access.

## 3. Net-New Code Inventory

Paths are planned internal choices. Classification reflects responsibility, not a requirement for one file/type per row.

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

Modify error catalogs/translations, mock generation configuration, contract/integration suites and API documentation in their owning implementation tasks. No account-management UI, assignment schema, consent records, execution storage or retention worker is added by this inventory.

## 4. Branching / PR Strategy

Recommended review order only; no branch, commit or PR creation is authorized by this document.

1. Domain and application port foundations. Administrative contract Q1 is resolved in T1 (T2–T3 remain).
2. Administrative persistence/credential adapters and operations (T4–T6).
3. Current authority, secured HTTP/bootstrap and real access evidence (T7–T9). Preserve ACP T1/T2/T3 and unblock ACP exposure only with actual evidence.
4. Candidate/assignment/exchange contract resolution, persistence and operations (T10–T12), once external prerequisites exist.
5. Candidate HTTP/browser and real assignment integration (T13–T14).
6. Retention/purge only after a separate resolved policy (T15).

T2/T3 pure foundation and T5 crypto adapter can proceed independently as specified in task-plan. Dependency branches/parallel headings do not instruct spawning agents. Split large tasks into smaller stable follow-up IDs if estimates prove inaccurate; preserve existing IDs on revision.

## 5. Testing Strategy

Every case here is a proof plan. No RED/GREEN, coverage, real database or browser success is claimed. Generate port/operation mocks through `.mockery.yml` and `make mocks`. Domain tests exercise public constructors/restoration/transitions; application tests mock ports; HTTP tests mock operations as required by root AGENTS.md. Real adapters need PostgreSQL and browser-compatible HTTP evidence.

### Unit Tests

- BR-020..023: bootstrap/provision paths, ADMIN-only HR creation, HR-only configuration, valid ADMIN login followed by configuration denial, HR cross-creator scope, unknown operation deny, no caller role accepted.
- BR-024..029: session creation and exact `now == expires_at` refusal; no sliding renewal; current-session logout; revoked/disabled denial; stable SubjectID across sessions; current HR authorization before any replay read; candidate cookies cannot resolve admin principals.
- BR-030..033: one candidate referenced repeatedly; no implicit email merge; invitation/session binding and copied snapshots; wrong assignment/invitation ownership, expiry and revocation; no success before commit.
- BR-034..039: exchange/reload has no execution/consent/provider write; expired session cannot authorize; reissue does not invalidate otherwise-live session; assignment revocation invalidates access; access does not reopen terminal execution.
- Restore corrupt roles, missing identities, inconsistent references, malformed digests/hashes and time order as technical failure. Do not reapply latest new-input policy during read or mutate deadlines.
- Credential adapter: independent random secrets, unique digests, password verify mismatch/unknown-account dummy path, bounded malformed-hash costs, constant-time secret checks, context/rate/concurrency limits, no secret leakage. Password acceptance follows T1 §6.1.
- Delivery: strict schema/limits, wrong-role/unauthenticated distinction, session kind substitution, origin and CSRF boundaries, cookie flags and clearing, no-store responses, no credential echoes; ACP retains its own public codes/ordering.

### Integration Tests

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
- Startup/shutdown budgets, shared pool cleanup and additive AutoMigrate continue to work; older compatible binary starts with extra access tables retained.

### Commands and evidence

During implementation run `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, `make test.security`, and `make test.docker` with the new suites registered. `make check` combines all-package/contract/architecture/lint/security checks. `make test.integration` is an alternative only with an explicitly authorized `TEST_DATABASE_URL`; keep credentials out of output. Missing database suites are skipped, not successful persistence verification. `make docker.verify` is optional broader full-stack work when relevant; access does not depend on optional brokers.

Record command/results, fixture/expected behavior and per-rule evidence. Browser runner/framework must be selected with the Interview UI owner because no candidate access test harness currently exists; T13/T14 add runnable instructions rather than inventing a current Makefile target. Current passing tests cannot establish historical TDD evidence.

## 6. Rollout & Rollback

- Administrative rollout: Q1 is resolved; install only administrative schema, bootstrap safely under existing startup budget, provision HR, verify real origin/CSRF/session behavior, then expose access endpoints. T9 evidence supplies ACP's access dependency; configuration routes still require ACP T8/T9.
- Candidate rollout: settle Q2/Q3/Q4, deploy candidate and real assignment references, install invitation/session constraints, then expose candidate exchange and minimal UI. Full consent/interview execution stays in Interview's release sequence.
- Observe safe operation/outcome/record IDs, denial categories, expiry, login/exchange failures, rate limiting, bootstrap/schema failure and transaction uncertainty. Never log passwords, cookies, raw link tokens, CSRF values, candidate contacts or HR configuration/report content by default.
- Stop rollout on cross-role or cross-assignment access, missing constraints, credential leaks, duplicate interview initiation, unexpected session invalidation or broken existing routes. Disable affected routes and restore compatible code; retain all five tables and external references.
- Cookie/CSRF secret rotation behavior and credential persistence across restart must be tested; rotating a CSRF signing key may invalidate outstanding CSRF values, while session authority remains persisted. Do not claim rotation preserves every browser action without proof.
- No automatic purge until Q5. Rollback is not deletion of accounts/candidates, reset of subject IDs or removal of referenced invitations. Candidate cancellation remains the assignment owner's responsibility.
