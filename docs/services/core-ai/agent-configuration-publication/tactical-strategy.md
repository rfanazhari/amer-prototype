# Tactical Strategy: Agent Configuration and Publication

## 1. Implementation Strategy

Mode C alignment of the additive capability in the existing Go modular monolith. Implement the assessed [technical design](technical-design.md) for [BR-CORE-001–014](../business/capabilities/agent-configuration-publication.md). Keep existing Task, voice, and administrative access behavior compatible. T1–T3 are done. Remaining code and verification below are planned work.

Use the approved T1 contracts in technical-design §6 for field validators, persistent payloads, and HTTP behavior. Install the five configuration tables as SQL version 3, ahead of the unpublished candidate and assignment-access files, then prove atomic publication through PostgreSQL. Mount configuration routes only by reusing `accessauthority.Authority`, the administrative cookie, and the existing CSRF verifier. Do not build a second session system and do not register AutoMigrate.

The final platform still needs a non-developer UI and assignment/execution binding evidence. This backend slice cannot by itself claim all P-01–P-05 criteria passed. A live voice feasibility spike is not a blocker for deterministic configuration/version mechanics; no provider integration belongs in this slice.

T1 (`CORE-ACP-001`) is Done: the user approved sections A–E on 2026-11-01.
[Technical-design §6](technical-design.md#6-request--response) contains the
accepted schemas/examples, limits, normalization, informational warnings,
subject-scoped publication identity/fingerprint, and public error precedence.
[Approval evidence](t1-contract-proposal.md) and decision-log D5 resolve Q1/Q3.
T4/T5 meet their contract prerequisites. T4 also confirms that migration versions 3–4 are still unpublished before it reallocates them. Q4/Q5 remain open. T1 and this alignment change documents only, so no SQL/interface mocks or behavioral RED/GREEN checkpoint is claimed. For later implementation use sqlmock for SQL mocks and Mockery-generated interface mocks; neither replaces the PostgreSQL proof required by T7/T9.

T5 must implement severity/error eligibility and the approved HR capacity and
Document subset rules. T6/T8 must preserve write-only compensation references,
full replacement defaults, JSON-safe version limits, 201 validation/422 blocked
publication, replay before current-draft checks and safe error precedence.

The access revision preserves T2/T3 completed foundations and all stable task IDs. No implemented configuration code needs repair. Reuse `AdminPrincipal.SubjectID` and `accessauthority.Authority`: HR operations are allowed, ADMIN configuration access is denied, and candidate cookies do not resolve a principal. All HR users share the single-company resource scope. T6 mocked orchestration and T8/T9 real integration must prove ADMIN denial, HR cross-creator access, candidate denial, current-authority replay checks, and unchanged salary filtering. The access prerequisite is already evidenced by CORE-IAM-009. T8/T9 still own configuration-route proof.

Companion work is specified in [Identity and Access](../identity-access/technical-design.md). CORE-IAM-001–009 cover administrative access. D8 requires this feature's tables to join `migrations/pgsql/` rather than AutoMigrate. D9 does not unblock assignment or execution binding. This feature still owns its configuration schema version, route dispatch, and ACP error catalog.

## 2. Code Reuse Map

Paths are repository-relative and were inspected during planning.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `CreateOperation`, `Create.Execute` | `application/usecase/taskusecase/create.go` | Pattern for consumer operation interfaces and injected implementations | Reuse structure, not Task inputs or policy; add authority and multi-record commit coordination |
| `Task.Update` contract | `application/repository/task.go` | Pattern for application-owned serialized mutation | Its Task callback cannot express exact validation/revision/command writes; new narrow feature contract required |
| `Store.Update`, `toTask` | `infrastructure/repository/taskrepository/task.go` | Row-lock, rollback, explicit-field mapping, technical restoration pattern | Do not extend Task store into generic CRUD; publication needs several coordinated records and deduplication |
| `Task.Snapshot`, `RestoreTask` | `domain/aggregate/task.go` | Encapsulation, copied ownership, separate historical restoration | Configuration contains nested values requiring deeper copying; Task lifecycle/date rules do not apply |
| `Title` | `domain/valueobject/title.go` | Example of value-owned validation | No code reuse of title policy; agent naming requirements follow approved design §6.1 |
| `Clock`, `System` | `application/service/ports.go`, `infrastructure/service/clock/clock.go` | Trusted time | Reuse as-is; application canonicalizes UTC milliseconds |
| Handler decoding/envelope/error mapping | `delivery/http/handler/taskhandler/handler.go` | HTTP behavior reference | Helpers are private and Task-specific; implement parity locally or extract only with regression proof |
| Public errors and translations | `pkg/errs/codes.go`, `translations/` | Error classification and EN/ID messages | Implement approved design §6.5 catalog entries in T8; preserve technical-error precedence |
| `openDatabase`, `schema.RequireCompatible` | `infrastructure/runtime/resources.go`, `infrastructure/database/schema/compatible.go` | Shared pool and read-only schema gate | Startup accepts only clean version 3 and performs no DDL. Do not add workers or change shutdown budgets |
| Administrative authority | `infrastructure/service/accessauthority/authority.go`, `application/service/policy.go` | Trusted principal and current HR authorization | Reuse as-is. Operation names are `create`, `replace`, `validate`, `publish`, `read`, and `replay`. `Authorize` ignores resource identity because company scope is shared |
| Administrative cookie, origin, and CSRF | `delivery/http/handler/accesshandler/cookies.go`, `errors.go` | Session binding and `X-CSRF-Token` verification | Reuse the credential binding and verifier. Configuration routes map failures to ACP codes, not access codes |
| Router | `infrastructure/runtime/voice.go` | Request dispatch | `/admin/` currently all goes to access. T8 must send `/admin/agent-templates` and `/admin/agent-instances` to the configuration handler first |
| Mock generation and verification targets | `.mockery.yml`, `Makefile`, `test/architecture/` | Mocked orchestration and boundary checks | Register new interfaces; preserve generated documentation; mocks do not prove SQL consistency |

No template catalog, instance storage, or publication implementation is available for direct reuse. Administrative sessions, HR authorization, cookies, and CSRF are. The existing voice provider adapter and Task event queue stay out of this feature graph because publication has no provider or event-dispatch side effect.

## 3. Net-New Code Inventory

Planned names may be refined within design authority; domain classifications do not imply one file per row.

| Item | Layer/Component | Package/Path | Type | Ownership / Notes |
|---|---|---|---|---|
| AgentInstance | Domain | `domain/aggregate/agentinstance.go` | Aggregate root/entity | Owns current draft version, immutable template reference, and active selection |
| AgentDraft | Domain | `domain/aggregate/agentdraft.go` | Owned entity | Instance-owned mutable configuration/version; permits incomplete supported drafts |
| PublishedRevision | Domain | `domain/aggregate/publishedrevision.go` | Immutable owned entity | Instance-owned version identity and frozen configuration; separately queryable |
| ValidationResult | Domain | `domain/aggregate/agentvalidation.go` | Immutable assessment entity | Links instance, exact draft, template/behavior version and findings |
| TemplateVersionRef / DraftVersion | Domain | `domain/valueobject/` | Value objects | Own identity/equality and valid version representation; prevent interchangeable raw values |
| HRConfiguration / DocumentConfiguration | Domain | `domain/valueobject/` | Configuration values | Draft/revision-owned typed content; exact fields approved in design §6.1; implementation T5 |
| ValidationFinding / severity | Domain | `domain/valueobject/`, `domain/enum/` | Value object / enum | Finding belongs to assessment; severity policy approved in design §6.2; implementation T5 |
| Template catalog and deterministic assessment | Domain | `domain/agentconfig/` | Domain policy/catalog | Concrete two-template dispatch; no I/O or generic plugin engine |
| Publication eligibility | Domain | AgentInstance/assessment behavior | Aggregate-owned predicate | Version, ownership and eligibility checks; not a standalone service merely to wrap a condition |
| Commands and query operations | Application | `application/usecase/agentconfigusecase/`, `application/query/agentconfigquery/` | Operation interfaces and implementations | Authorize, resolve input/reference context, invoke rules, return detached DTOs |
| Configuration transaction and read ports | Application | `application/repository/agentconfiguration.go` | Repository contracts | Purpose-specific reads/atomic writes; no ORM handles; command resolution contract |
| Admin authority and stable reference ports | Application | `application/service/` | Service ports | Authority port and operation names already exist; T6 consumes them. Reference-reader facts remain T5 |
| Command intent/fingerprint | Application | `application/usecase/agentconfigusecase/` | Application value/helper | Approved normalized request identity; scope is `SubjectID`, not a client role |
| Result DTOs | Application | `application/dto/agentconfiguration.go` | Detached data | Safe views, copied nested values, nullable active reference |
| Store and five storage record types | Infrastructure | `infrastructure/repository/agentconfigrepository/` | GORM adapter/models | JSONB encoding/decoding, references, row locks, explicit updates; CRUD only |
| Configuration migration | Infrastructure | `migrations/pgsql/000003_agent_configuration.{up,down}.sql` | Versioned SQL | Five tables and constraints. Unpublished candidate/assignment files are 000004/000005. Clean-version floor is 3 |
| Admin configuration handler | Delivery | `delivery/http/handler/agentconfighandler/` | HTTP adapter | Strict schemas, envelopes, and ACP errors; reuses access cookie and CSRF |
| Feature composition | Infrastructure | `infrastructure/runtime/agentconfiguration.go` | Composition function | Construct `accessauthority.Authority`; register only the secured configuration paths |

Also modify the schema compatibility floor, checksum manifest, runtime grants, migration operations document, error catalogs, `.mockery.yml`, API specification/docs/changelog, and tests. No new authentication, assignment, execution, artifact, job, or reporting tables belong in this inventory.

## 4. Branching / PR Strategy

Recommended review order, not authorization to create branches, commits or PRs:

1. Domain version/ownership mechanics and application boundaries (T2–T3), with tests; leave routes unmounted.
2. Resolved template/contracts and domain validators (T1/T5), schema definition (T4).
3. Application operations (T6) and PostgreSQL adapter (T7), with fault/concurrency proof.
4. Secured HTTP, contract documentation, runtime integration, and acceptance (T8–T9), using the existing authority adapter after T4–T7.

Keep each task's changes reviewable. Commit per task when the user authorizes implementation/commits; this planning request creates documents only. T10 version compatibility and T11 consumer integration remain separately visible rather than being buried in an MVP-complete claim.

## 5. Testing Strategy

### Unit Tests

- Domain: both template references; version validity; stale replacement; published immutability; draft edits preserve active pointer; incomplete drafts remain editable; exact validation ownership; unsupported field choices; approved informational warning policy in design §6.2. Cover zero-value misuse and copy isolation of nested slices/references/times.
- Creation/restoration: reject inconsistent identities/versions; retain stored historical interpretation; do not invoke current provider availability or silently normalize corrupt stored payloads. Test known historical formats when Q4 is settled.
- Application: generated mocks for repository, authority, reference reads, Clock, and delivery operation interfaces. Denial has no storage side effect; replay authorizes before disclosure; exact replay avoids current-draft validation and active-pointer mutation; mismatched intent conflicts.
- Publication fault paths: no successful DTO before commit; ambiguous commit reconciles with the same identity; reference unavailability is technical, distinct from a missing reference finding.
- HTTP: mocked operations establish strict JSON and sub-fields, request size, nullable values, safe classifications, EN/ID messages, confidentiality filtering, and Q1 error precedence. These tests remain required despite the skill baseline's different default.

### Integration Tests

Real PostgreSQL is required for these claims:

- Create → replace → validate → publish → read revision for each template. Start from a fresh migrate to version 3 and from a version-2 database that already contains Task and administrative rows; confirm those rows survive the configuration migration.
- Two replacements against the same draft version: one winner, one conflict, no lost accepted draft.
- Validation racing replacement: stale validation result cannot qualify the new draft.
- Publication racing replacement: either the publication commits for the exact validated version before replacement, or it conflicts after replacement; never publish unvalidated content.
- Same-key concurrent publication: one revision and command result; a different-intent reuse conflicts, including a different instance in the same identity scope.
- Different new publication commands serialize revision allocation; no duplicate revision number or broken active pointer.
- Inject failure after revision insert, after active-pointer update, before command insert, and at commit. Verify complete rollback for known failures; verify matching-key reconciliation for ambiguous outcomes.
- Retry an old successful command after another publication and later draft edit: original result returned and newest active pointer retained.
- Constraint failures, cross-instance references, corrupted JSONB/versioned content, explicit null/empty/false replacement, and restart persistence.
- Repeated queries do not load all historical revisions or perform implicit association writes. Inspect statements for ordinary bounded reads; do not invent a performance benchmark target.
- Real HR, ADMIN, candidate-cookie, expiry, revocation, and CSRF cases on the configuration routes. Mocked authority tests do not establish this boundary. CORE-IAM-009 already proves the adapter outside these routes.

BR-CORE-013 has local immutable-read proof in T7/T9, plus external assignment/execution proof in T11. Neither substitutes for the other. Q4 must govern tests for historical template versions before multi-version deployment.

### Regression Tests and Commands

During implementation run focused new package tests first, followed by applicable repository checks:

- `make mocks` after mocked interface changes; `gofmt` changed Go files.
- `make test` for domain/application race and coverage.
- `make test.all` for all-package race checks, including existing Task/voice/runtime behavior.
- `make test.contract` and `make test.architecture` for OpenAPI, dependency and documentation rules.
- `make lint` and `make test.security` before delivery, or `make check` for the combined target.
- `make test.docker` for isolated PostgreSQL integration after registering new suites with the verification harness. Alternatively `make test.integration` only with an explicitly authorized `TEST_DATABASE_URL`.

Inspect the Docker verification script before assuming new test packages are included. Keep bounded startup/shutdown tests and existing Task completion rollback/queue tests passing. Database suites skipped without a URL are not successful database verification. Record exact commands, results, and limitations; do not manufacture historical RED/GREEN evidence.

None of these runtime commands was executed to generate this tripack. Documentation checks establish document consistency only.

## 6. Rollout & Rollback

- Rollout: Q1/Q3 are resolved and the administrative access adapter exists. Implement validators and operations, apply SQL version 3 with the explicit migrator, raise the startup floor to that clean version, then mount configuration routes on the existing adapter. Seed synthetic data through ordinary create/replace/validate/publish operations; do not insert prevalidated records to bypass rules.
- Observe safe request/instance/revision IDs and outcome categories, stale conflicts, publication replay/conflict counts, and commit failures. Avoid logging configuration content or credentials. No new observability backend is required.
- Stop rollout on failed constraints, partial publication, unsafe disclosure, or changed Task, voice, or access route behavior. Disable configuration routes and roll back to a migration-aware binary that accepts the previous clean version, leaving published tables retained. Do not roll back to an AutoMigrate binary and do not down retained publication data.
- Do not delete revision records or repoint an instance to an old revision as an operational rollback shortcut. Template-version deployment rollback requires Q4; retained data is not proof that an older binary can interpret it.
- Final admin UI and external binding proof remain platform integration work. Backend completion must be reported with those limits.

## 7. Revision History

| Version | Date | Change |
|---|---|---|
| v1.1 | 2026-11-01 | Synchronize user-approved T1 A–E contracts and Q1/Q3 resolution; retain all other task and exposure gates |

| v1.2 | 2026-11-01 | Mode C: synchronize owner-approved HR-only configuration authority and ADMIN separation; retain all implementation/exposure gates |

| v1.3 | 2026-11-01 | Link Identity and Access administrative evidence handoff to ACP Q2; candidate integration remains independent |
| v1.4 | 2026-11-01 | Align schema installation with manual SQL version 3 and treat the existing authority adapter as the access prerequisite |
