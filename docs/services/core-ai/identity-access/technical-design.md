# Technical Design: Identity and Access

## 1. Overview

Date: 2026-11-01. Mode C — revise the existing tripack after administrative implementation and the owner's decision to replace startup AutoMigrate with explicit, versioned SQL migrations across the service. T1–T9 retain their recorded Done status; candidate runtime and the migration transition remain unimplemented. Authority: [Identity and Access](../business/capabilities/identity-access.md), BR-CORE-020..039, the owner's conversation approvals, and [platform design §9–10 / TD-21](../../../design/core-ai/technical-design.md). Migration-policy approval is recorded in decision-log D8; this revision writes plans only.

### Goals

- Provision ADMIN/HR administrative accounts and real revocable sessions; allow ADMIN to add HR accounts, without HR business authority.
- Supply current HR-only authority to Agent Configuration through the existing application port, including reads and publication replay.
- Preserve reusable candidate identity and issue assignment-scoped invitation/session access without permanent candidate accounts.
- Keep access establishment separate from consent, execution, and provider sessions.

### Non-Goals

Multiple companies/workspaces, public signup, configurable role/permission tables, combined roles, SSO/MFA, account recovery UI, public candidate passwords, mail delivery infrastructure, assignment/execution/consent implementation, interview provider integration, and replacing Task/voice authentication behavior. A DISABLED state does not authorize adding account-lifecycle endpoints. Do not implement a placeholder assignment table or permissive authorizer to unblock candidate delivery.

### Verified project context and impact

| Source inspected | Observed fact and design consequence |
|---|---|
| Root `AGENTS.md`, Architecture / Verification / Local resources | Four root layers; application-owned ports; operation interfaces; detached DTOs; trusted UTC millisecond time; all declarations documented; HTTP tests through mocked operations; bounded lifecycle. Its AutoMigrate policy is superseded for the planned transition by the owner's current instruction (D8). No scoped AGENTS.md or CLAUDE.md found. |
| `README.md`, `go.mod`, `Makefile` | Go 1.26.10, GORM 1.25.12 / postgres driver 1.5.11, PostgreSQL, net/http, testify, Mockery; current startup migrates Task. Redis/RabbitMQ are not access dependencies. |
| `application/service/agentconfiguration.go`, `infrastructure/service/accessauthority/` | `AdminPrincipal{SubjectID}` and `AgentConfigurationAuthority.Principal/Authorize` exist, with a real administrative adapter. Preserve their public shape and current authority behavior. |
| `application/usecase/taskusecase/create.go`, `application/repository/task.go` | Narrow delivery operation interface, plain injected implementation, trusted Clock and detached result are suitable patterns. Task business policy is not identity policy. |
| `infrastructure/repository/taskrepository/task.go` | Separate storage model; row locking; explicit updates; restoration errors classified as technical. Task's callback is not an invitation/session repository. |
| `infrastructure/runtime/resources.go`, `http.go`; `delivery/http/handler/taskhandler/handler.go` | Shared database lifecycle and shutdown admission; strict bounded JSON, envelope and safe EN/ID errors. Handler helpers are private and Task-oriented. |
| `docs/api/openapi.yaml`, `docs/api/identity-access.md`, `docs/api/CHANGELOG.md` | Four administrative paths are published and mounted; candidate access paths remain planned. Task/voice/health do not acquire global access authentication. |
| `infrastructure/repository/accessrepository/models.go`, `migrate.go`, `infrastructure/runtime/resources.go` | Startup currently AutoMigrates `tasks`, then two administrative tables and adapter-owned SQL constraints/indexes. No versioned SQL migration directory or golang-migrate dependency exists. |
| `test/integration/access_repository_test.go`, `access_http_test.go`, `test/integration/repository_test.go`, `lifecycle_test.go`, `scripts/verify_compose.py` | Integration setup also calls AutoMigrate; lifecycle coverage includes cancellation during startup DDL. These entry points must change with the schema owner, not just the application startup calls. |
| Repository source inventory, `.mockery.yml`, `test/architecture/` | Administrative aggregates, ports, operations, adapters and handlers exist. Candidate/assignment persistence does not. Domain imports exclude crypto, JSON, net/http and GORM. |

No live database was inspected and no database suite was rerun for this revision. Task-plan T1–T9 statuses and the existing [ACP access handoff](../agent-configuration-publication/q2-access-evidence.md) are prior records, not fresh acceptance evidence. The original technical design/strategy lagged those records; this revision aligns them without reconstructing historical test results. Existing ACP foundations and business contracts remain unchanged.

Project instructions override bundled Backend Service Standard defaults for delivery tests, application-owned repository ports and encapsulated domain state. The owner has now intentionally amended schema policy: manual migrations mean reviewed SQL files applied by an explicit command before startup, not handwritten SQL applied without version tracking. GORM remains the CRUD adapter. Task is included only for its schema installation and lifecycle/test reconciliation; its business logic and the voice feature are unchanged. Binding contracts are BR rules, roles/scope, expiry, revocation, session separation, confidentiality and ACP error ordering. Internal package/helper choices can be refined with preserved-contract evidence; unresolved product choices in §9 block their dependent work.

### Mode C baseline and reconciliation direction

| Area | Coverage against the current repository | Difference assessment / action |
|---|---|---|
| Administrative T1–T9 | Implemented, with existing unit/integration sources and recorded Done status; not reaccepted by this review | Original AutoMigrate implementation conforms to its original policy. Preserve completed tasks; update stale documentation. |
| Current Task + access schema installation | Implemented-but-deviating from the newly approved target, not a retroactive defect in T4/T8 | Code and documents must change in T16–T20: replace both runtime DDL owners, all test setup callers and Compose ordering. |
| Candidate/assignment persistence T10–T15 | Not-yet-implemented; real assignment storage absent | Retain Q2–Q5 gates; future candidate DDL must use the new migration mechanism. |
| Versioned artifacts, schema-version guard and legacy adoption | Not-yet-implemented | New technical reconciliation; no account/session/business-rule redesign. No unrelated code-not-covered-by-design identified in the schema-owner search. |

## 2. API / Interface Contract Changes

The existing `/admin` prefix means the administrative channel, not the ADMIN role. HR uses this channel too. Proposed additive REST directions:

| Method/path | Authority | Result / responsibility |
|---|---|---|
| POST `/admin/sessions` | Provisioned ADMIN or HR credentials | 201; establish cookie after session commit; account/session metadata only |
| GET `/admin/session` | Valid administrative session | 200; safe account, role, session expiry and CSRF material |
| DELETE `/admin/sessions/current` | Valid administrative session + CSRF | 200 standard envelope after persisted revocation, then clear cookie |
| POST `/admin/hr-accounts` | Current ADMIN + CSRF | 201; create HR, with role fixed by operation, not caller input |
| POST `/admin/candidates` | Current HR + CSRF | 201; record candidate identity |
| GET `/admin/candidates/{id}` | Current HR | 200; detached candidate contact view for selection by known identity |
| POST `/admin/interview-assignments/{id}/invitations` | Current HR + assignment eligibility + CSRF | 201; persist new invitation/revoke former invitation atomically; reveal link/token once |
| POST `/candidate/access/exchange` | Valid invitation + pre-session origin protection | 201 after new session commit; scoped cookie and safe assignment identity; reusable invitation until expiry/revocation (§6.7) |
| GET `/candidate/session` | Valid candidate session + accessible assignment | 200; safe candidate/assignment identity, expiry and CSRF material; no execution creation |

Administrative account/session schemas and limits are approved in §6.1–6.5; candidate-specific accepted inputs are approved in §6.6–6.9. No collection listing/search or lifecycle mutation endpoints are silently added. The Interview capability owns `/candidate/interview`, consent, initiation, assignment creation/revocation and its full candidate UI. A known `candidate_id` enables reuse without introducing deduplication by email.

Implement one operation interface per delivery use case/query in `application/usecase/accessusecase/` and `application/query/accessquery/`, returning detached `application/dto` values. Purpose-specific repositories in `application/repository/accessport/` hide ORM handles. Ports in `application/service/` isolate password verification, secure token generation/digests, session resolution and assignment access facts.

The real `AgentConfigurationAuthority` adapter resolves a trusted account/session, returns `AdminPrincipal.SubjectID = administrative_accounts.id`, and consults current account/session state for authorization. ADMIN resolves an administrative principal but fails configuration authorization. Candidate cookies never resolve one. Authenticate in delivery and recheck authority at application boundaries, including non-HTTP operations and replay. Never trust JSON role, account ID header or a caller-constructed principal without current-session binding. Unknown operation strings are denied.

Administrative contracts are already published in `docs/api/openapi.yaml`, `docs/api/identity-access.md`, and `docs/api/CHANGELOG.md`. Update them together when implementing candidate endpoints. The migration transition has no public HTTP contract change. Existing Task/voice/health behavior remains compatible; do not attach global mandatory auth or publish unimplemented candidate routes.

## 3. Data Model & Schema Changes

### Concepts, identity and consistency

AdministrativeAccount is an aggregate root with stable UUID, role and status; administrative sessions are independently identified access records referencing it. Account identity must never be recycled or derived from email/session. Candidate is a separate identity aggregate with no password or administrative role. HR can reference one candidate from several assignments. Candidate contact changes must not rewrite assignment-bound historical input snapshots owned by Interview.

Invitation is an entity whose replacement is serialized under the assignment consistency boundary. CandidateSession is a separate access entity bound to one assignment and source invitation. Issuance, exchange and assignment revocation must coordinate with the assignment owner; they cannot be made safe by independent table CRUD. Assignment keeps at most one execution under the platform contract. No access operation creates, restarts or substitutes that execution.

Role, AccountStatus, SessionLifetime and immutable assignment-access facts have value semantics. Session/Invitation domain predicates own `revoked_at == nil && now < expires_at`; administrative access also requires ACTIVE account. Trusted `Clock` supplies `now`, UTC truncated to milliseconds. Creation requires expiry after creation, and session expiry exactly creation + 2 hours; invitation expiry creation + 24 hours. Restoration validates stored identity, time order and references without resetting deadlines, copying all mutable pointers. Crypto/encoding remain infrastructure/application work; entities do not import crypto/JSON/HTTP.

### Approved five-table target

All IDs are UUID primary keys; timestamps are PostgreSQL `timestamptz`, canonical UTC milliseconds. All fields are NOT NULL unless explicitly nullable. FK removal uses RESTRICT; no soft deletion or cascade purge is selected.

| Table | Columns and constraints | Indexes / ownership |
|---|---|---|
| `administrative_accounts` | `id`; `name text`; `email text` unique canonical login; `password_hash text`; `role text CHECK IN ('ADMIN','HR')`; `status text CHECK IN ('ACTIVE','DISABLED')`; `created_by uuid NULL` self-FK for bootstrap exception; `created_at`, `updated_at` | Unique canonical email; stable subject. Normalization and public field bounds §6.1. Plaintext passwords never persisted. |
| `administrative_sessions` | `id`; `account_id uuid` FK accounts; `token_hash text` unique; `created_at`; `expires_at CHECK > created_at`; `revoked_at NULL` | Account FK index; expiry index for later cleanup. Current role/status read from account, not session snapshots. |
| `candidates` | `id`; `name text`; `email text` contact, deliberately not unique; `created_by uuid` FK accounts; `created_at`, `updated_at` | Creator FK index. HR-created; no role/password/login account. Matching is Q4, not an implicit email upsert. |
| `interview_invitations` | `id`; `assignment_id uuid` FK external assignments; `token_hash text` unique; `expires_at CHECK > created_at`; `revoked_at NULL`; `created_by uuid` FK accounts; `created_at` | `(assignment_id, id)` unique for same-assignment session FK; assignment index; partial unique `(assignment_id) WHERE revoked_at IS NULL` to enforce one current unrevoked invitation. An expired row is explicitly revoked during replacement. |
| `candidate_sessions` | `id`; `assignment_id uuid` FK external assignments; `invitation_id uuid`; `token_hash text` unique; `created_at`; `expires_at CHECK > created_at`; `revoked_at NULL` | Composite `(assignment_id, invitation_id)` FK to invitation `(assignment_id, id)`; assignment and expiry indexes. Candidate identity derives through assignment, not caller input or duplicated candidate FK. |

Use nonempty role/status and canonical digest checks in the schema; existing-row collision behavior follows §6.4. T4/T8 cover administrative named DDL/mappings; T11 covers the candidate additions. Creation-time creator-role authority is enforced in application, not a static FK which would break when an employee's role later changes. `revoked_at` is nullable and must not precede creation. Digests are 64-character lowercase SHA-256 hex for 32-byte CSPRNG tokens; each credential kind has its own generation/lookup namespace. Record IDs are public identifiers, not credentials.

**Count:** five access-owned tables. `interview_assignments` is a sixth table only when counting the separate Interview dependency; its full schema is not invented here. Required facts are stable assignment ID, candidate reference, pinned published revision, current access-revocation state, and serialized admission/revocation ownership. Consent, executions, provider sessions and agent configuration tables are excluded from the access count.

### Adapter, constraints and installation

Use GORM 1.25.12 CRUD with separate models and explicit maps/Select updates for `revoked_at`, status and zero/null fields. Do not let `Updates(struct)` omission semantics control revocation/reset; do not use unconstrained Save/upsert or automatic association writes. Bounded explicit joins for account/session and assignment/session resolution avoid N+1 and history loading. PostgreSQL enforces unique digests, email and composite ownership in addition to application checks. Under the approved target, named constraints and the partial unique index belong exclusively to versioned SQL migrations, not adapter/runtime DDL. The existing [GORM update reference](https://gorm.io/docs/update.html) explains omission behavior; retain administrative mapping coverage and extend it for candidate storage against the pinned repository version.

T18 replaces additive AutoMigrate and accessrepository-owned DDL. Explicit migrations are the single schema owner for Task and administrative access; ACP T4 later raised the read-only startup guard from clean version 2 to clean version 3. See [migration operations](../../../design/database-migrations.md) and [T18–T20 evidence](t18-t20-evidence.md). Preserve the ten-second startup budget and existing cleanup/shutdown guarantees. Candidate identity and assignment-dependent tables remain staged behind their actual contracts. No Task backfill or account/session rewrite is required. Unique bootstrap/account operations still handle concurrent processes.

Application rollback unmounts affected routes and restores a migration-aware compatible binary while retaining tables and historical links. A pre-cutover binary runs AutoMigrate again and is not an acceptable routine rollback target. Do not drop tasks, candidates, referenced accounts or published-command subjects. Purge/retention Q5 needs a separate approved policy before physical removal.

### 3.1 Migration policy and artifacts

**Approved policy, implemented foundation (T16–T20):** the owner selected explicit manual migrations because the project has grown (D8). Use `golang-migrate/migrate/v4`, pinned to a version verified during T17, with one service-wide ordered PostgreSQL stream in `migrations/pgsql/`. Retain GORM models for CRUD only. The migration runner owns schema DDL; no production or integration setup path continues to invoke AutoMigrate after cutover. Hand-authored up/down SQL is reviewed in Git and applied explicitly by an operator or a separate pre-start job; application startup never runs `up` or auto-repairs drift.

Versions 1–2 are allocated and pinned by T16/T17. The candidate versions below remain proposed until their owning stream dependencies are reconciled. Once applied, files are immutable; later changes use new versions. Candidate versions must be reallocated if the real Interview/ACP schema introduces intervening migrations.

| Proposed artifact pair under `migrations/pgsql/` | Forward change | Reverse / recovery | Dependency / task |
|---|---|---|---|
| `000001_tasks.up.sql`, `000001_tasks.down.sql` | Reproduce current `tasks` UUID PK, text title/status, nullable due/completion timestamps, non-null creation/update timestamps, and `task_status_check` | Disposable databases only: drop tasks after application use stops; populated databases retain schema and use compatible code or reviewed forward repair | T16 schema manifest → T17 |
| `000002_administrative_access.up.sql`, `000002_administrative_access.down.sql` | Reproduce both current administrative tables, account self-FK, session account FK, role/status/nonempty-hash/digest/time checks and all five named indexes in `accessrepository/migrate.go` | Disposable databases only: drop sessions before accounts, without CASCADE; referenced/populated databases require data-preserving recovery | Version 1 → T17 |
| `000003_agent_configuration.up.sql`, `000003_agent_configuration.down.sql` | Five ACP configuration tables inserted ahead of unpublished candidate SQL so HR configuration can install without `interview_assignments` | Disposable reversal only; retained publication data uses forward recovery | ACP T4 (`CORE-ACP-004`) |
| `000004_candidates.up.sql`, `000004_candidates.down.sql` | Candidate identity and creator FK/index from approved five-table target, after contact contract is settled; email remains nonunique. Reallocated from unpublished 000003 by ACP T4 | Drop only on disposable, unreferenced data; real assignment references prevent unsafe reversal | Migration foundation T20 + Q2/Q3/Q4/T10 → T11 |
| `000005_assignment_access.up.sql`, `000005_assignment_access.down.sql` | Invitations, candidate sessions, actual assignment FKs, composite same-assignment FK, digest/time checks and partial current-invitation index. Reallocated from unpublished 000004 by ACP T4 | Sessions before invitations; never remove Interview-owned tables; populated data uses forward recovery | Candidates + actual assignment migration applied first + T10 → T11 |

Baseline versions 1–2 reproduce the **observed implementation**, not every desirable extra constraint. `taskrepository.Model`, access models and named DDL are the sources; T16 verifies exact PostgreSQL types, nullability, defaults, PK/index/FK definitions and names against isolated generated legacy schema. Preserve timestamp precision and existing UTC millisecond application semantics; do not silently round historical values, canonicalize contacts, tighten field rules, rename indexes, rotate credentials or reset expiry. A manifest mismatch or missing intended constraint becomes an explicit finding with its own additive migration/compatibility assessment, not an unreviewed baseline repair.

`schema_migrations` is tool-owned version/dirty metadata shared by this service. It is additional infrastructure metadata, not a sixth access business table or an Interview assignment substitute. Applied migration checksums belong in the release manifest/CI comparison; do not assume version/dirty metadata provides checksum verification. Candidate and future ACP/Interview schemas join this same ordered stream once their designs are ready.

### 3.2 Runner, runtime guard and environment execution

- Add a thin `cmd/migrate/main.go` using an infrastructure adapter under `infrastructure/database/schema/`. Make targets are `migrate.status`, `migrate.up`, `migrate.baseline`, and `migrate.down`. `migrate.up` requires an explicit release target such as `MIGRATION_VERSION=2`, rather than applying every future file indiscriminately. `migrate.down` requires an explicit step count and disposable-database confirmation; populated-environment rollback uses the recovery path below.
- Read the DDL-capable connection only from proposed `MIGRATION_DATABASE_URL`; never expose it in CLI arguments, Make echo, logs, or committed files. Runtime retains `DATABASE_URL`, with CRUD and metadata-read permissions; verify that it cannot create/alter/drop schema in the acceptance environment. This role split is an operational setup obligation, not automatic role creation against an unspecified database.
- Use the migration tool's database lock, bounded connection/lock/statement waits and cancellation, initially a 60-second overall migration-command budget with a 5-second lock-wait bound. T17 validates timeout behavior against the pinned tool version; timeout/partial failure exits nonzero and prevents app rollout. These are separate from the ten-second application startup and five-second process shutdown budgets.
- Replace both DDL calls in `infrastructure/runtime/resources.go` with a read-only version/dirty guard before bootstrap and listener binding. The first cutover binary required clean version 2; ACP T4 (`CORE-ACP-004`) declared clean version 3. Missing metadata, dirty state, unsupported older/newer version, or unreadable status fails startup with resource cleanup and safe diagnostics. Runtime does not create metadata or silently baseline an old database.
- Local development: initialize PostgreSQL, explicitly migrate to the release target, then start the app. Existing volumes take the adoption path in §3.3. Compose offers a separately invoked one-shot migration job/profile; normal app startup has no schema-writing dependency hook. `make docker.up` must fail with clear migration instructions on an unprepared database, not silently perform DDL.
- Isolated verification: `scripts/verify_compose.py` explicitly starts PostgreSQL, applies migrations, then starts the app and runs suites. Change `test/integration/repository_test.go`, `access_repository_test.go`, and `access_http_test.go` to use the same versioned setup; retain isolated legacy SQL fixtures only for adoption tests. Move the blocked-DDL cancellation case in `lifecycle_test.go` to runner tests, and retain runtime connection/read-only-check cancellation/cleanup coverage.
- Shared/test/release environments: the operator or separate pre-start job checks status, applies only the reviewed release target, verifies clean status, then launches the compatible binary. T18 documents commands and artifact delivery in README/Compose/Dockerfile and `docs/design/database-migrations.md`. No remote database execution, deployment, or credential provisioning is authorized by this planning revision.

Tool semantics consulted: the [PostgreSQL driver documentation](https://github.com/golang-migrate/migrate/blob/master/database/postgres/README.md) describes metadata-table and statement-timeout configuration; the [getting-started guide](https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md) describes transactions, up/down validation and dirty-version recovery. T17 must verify the selected release's behavior rather than copying unpinned commands into production.

### 3.3 Existing database adoption and recovery

T16 records a canonical baseline manifest at proposed `docs/design/database-schema-baseline.md`, with synthetic fixtures at `test/integration/testdata/schema/legacy_automigrate.sql`. No real database has been inspected in this review. Adoption is an explicit operation against an authorized target; never infer deployed structure from Go files alone.

| Observed target before cutover | Action | Acceptance condition |
|---|---|---|
| No owned tables and no migration metadata | Apply migrations 1–2 normally | Three business tables and clean version 2; normal bootstrap creates ADMIN afterward |
| Exact legacy Task schema only, no administrative tables or metadata | Verify Task manifest, record baseline version 1, then apply version 2 | Existing Task rows/timestamps preserved; administrative schema installed normally |
| Exact legacy Task + both administrative schemas, no metadata | Verify full manifest, record baseline version 2; do not execute creation files | Existing task/account/session IDs, hashes, creator links, expiry and revocation unchanged |
| Clean recognized migration metadata already present | Validate version and migrate only to declared compatible target | Repeated up-to-target is a no-op; a rerun does not reseed or reset data |
| Partial administrative pair, unexpected columns/types/defaults/checks/indexes/FKs, unrelated existing migration history, dirty state or unsupported version | Refuse adoption/upgrade; report structural differences without contacts or secrets | A reviewed reconciliation/recovery plan is required before retry; never stamp a guessed version |

Before legacy adoption, stop old binaries and other schema writers, take a restorable backup/snapshot of populated targets, and rehearse restore. The guarded baseline operation holds exclusive migration coordination across catalog comparison and the metadata write; no concurrent migrator or legacy AutoMigrate process can intervene. Only after exact equivalence is established may the command record the appropriate version. Do not expose an unchecked generic `force` shortcut as normal onboarding. Existing rows are not backfilled or copied for this transition; any actual drift requiring data repair receives a separate reviewed change.

For a failed migration, stop app rollout and inspect the real schema, transaction outcome and dirty status. Initial baseline SQL uses explicit transactions and regular indexes; do not introduce concurrent-index behavior into that baseline. Recover with a reviewed forward repair or a verified backup restore. Clearing dirty status is a separate operator action only after establishing the actual version; do not auto-force on error. Routine application rollback keeps the migrated database and uses a binary tested against it. Destructive down/up tests run only on isolated disposable PostgreSQL; normal development volumes are preserved.

### 3.4 Seed and reference-data assessment

No persisted master/reference rows are required. ADMIN/HR and ACTIVE/DISABLED remain code enums plus checks; do not invent role/status tables or SQL seed accounts. The first ADMIN remains environment-backed application bootstrap **after** schema validation, with its approved no-overwrite/restart behavior. No passwords/hashes, accounts, candidates, invitations or sessions are inserted by migrations. Optional demo data is outside this transition; integration fixtures use synthetic identities and public operations. Required dataset, rerun and cleanup tasks are therefore not applicable beyond verifying bootstrap idempotence and unchanged existing data.

### Migration rule ownership and proof plan

| Material guarantee | Owner / enforcement path | Proof plan |
|---|---|---|
| One ordered schema authority | Migration runner; runtime/integration callers stop AutoMigrate | Fresh up → down → up on disposable PostgreSQL; source inventory shows no executable AutoMigrate caller; runtime role cannot DDL |
| Legacy adoption preserves identity and history | Guarded baseline command, manifest comparison, operator maintenance/backup procedure | Populated Task-only and full-three-table fixtures; compare row values/IDs/hash equality/times before and after without printing secrets; existing session remains usable |
| Startup never repairs incompatible schema | Runtime version/dirty guard before bootstrap/listener | Missing/old/dirty/future version denies startup; compatible version starts; resources close on timeout/failure |
| Failure cannot publish migration success or destroy history | Runner transaction/lock handling and recovery procedure | Concurrent migrators, held-lock timeout, cancelled DDL and injected failure; dirty handling and restore rehearsal; no automatic down/force |
| Candidate installation uses real owners | T10 contract, shared migration ordering, T11 FKs | Candidate migrations wait for actual assignment schema; cross-assignment FK and partial uniqueness tests use PostgreSQL |

### Transaction boundaries and races

- Bootstrap: hash outside the database transaction; serialize initialization with a database mechanism under the bounded startup budget. Create the initial ADMIN only when initialization is applicable; never overwrite an existing user's password or role from environment on restart. Incompatible existing state fails safely, without logging credentials. No public create-ADMIN operation.
- HR provisioning: require current ADMIN, then transactionally recheck its session/account and insert HR. Unique canonical email resolves races. Commit failure returns no account success; retry after uncertainty may encounter an existing account, not automatically reset its password. Candidate creation similarly requires HR; name/email are not deduplication keys.
- Administrative login: resolve account, verify password outside row locks, then lock/recheck account eligibility and verified credential version/hash before inserting session. If credentials changed meanwhile, fail/retry authentication rather than issue against stale proof. On every protected action, verify current session/account. Logout atomically persists revocation; protected mutations recheck access inside their transaction. Already-admitted operations may complete if revocation commits afterward; no retroactive cancellation claim.
- Invitation issue/reissue: lock assignment first; check current HR and assignment eligibility; revoke prior unrevoked invitations, insert new invitation, and commit before revealing token. Rollback retains former invitation. Concurrent reissues serialize; a later issuance may invalidate an earlier returned link. Lost-response/retry delivery details are Q3; do not claim exactly-once token delivery.
- Exchange: lock assignment, then invitation, check recognized credential, expiry/revocation and assignment state, and insert candidate session atomically. Q3 controls whether another exchange may insert another session. Commit failure sends no session cookie. A random token collision is retried internally with a new random value in a fresh valid transaction, with a bounded attempt count; no aborted transaction is reused.
- Candidate session resolution: check its own expiry/revocation and current assignment access. Source invitation revocation/expiry is checked at exchange, not blindly reapplied to existing sessions: otherwise reissuing a link would cancel active access contrary to BR-037. The source FK records provenance, not perpetual authorization. Assignment revocation invalidates all access through a live assignment predicate and coordinates cancellation under Interview ownership.
- Interview admission/revocation: assignment lock precedes execution locks per platform §5. Access module must not acquire assignment after execution or independently cancel execution. Persistent revocation/expiry during an active execution is integrated by the Interview owner; request denial alone is not proof of cancellation.

### Rule ownership and proof plan

All proof below is planned, not observed test evidence.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| BR-020 | Bootstrap + ProvisionHR application operations | Startup, HR creation; absence of signup | Bootstrap rerun/concurrency; unauthorized enrollment denied |
| BR-021 | Fixed access policy + current account resolver | HTTP/non-HTTP ProvisionHR | ADMIN allow; HR/candidate deny; no writes on denial |
| BR-022 | Fixed access policy | All config/report authorization, replay | Valid ADMIN login followed by 403 on config; report consumer proof remains separate |
| BR-023 | HR policy, shared company scope | Configuration create/read/edit/validate/publish | HR B reads/edits HR A instance subject to version checks |
| BR-024/025/026 | AdministrativeSession + account eligibility; application authority | Login, current session, logout, every protected operation | Exact expiry boundary; revoked/disabled denial; already-issued credentials do not freeze role |
| BR-027/028 | Authority adapter + ACP operations | Principal resolution, new session, replay | Same SubjectID across sessions; HR replay allowed, current denial before lookup |
| BR-029 | Separate principal/session resolvers | Administrative and candidate entry points | Cookie substitution rejected; candidate never gets admin principal |
| BR-030 | Candidate identity + Interview assignment owner | Candidate creation/read; future assignment creation | One candidate with two real assignments; no implicit email dedup |
| BR-031 | Assignment-scoped invitation/session predicates + composite FK | Issue, exchange, session resolve, every candidate resource query | Session A cannot read B even for same candidate; cross-assignment FK fails |
| BR-032/033 | Invitation, exchange transaction, credential adapter | Exchange and restoration | Equality expiry, revoked/unknown token, assignment revoked, collision/rollback, distinct IDs |
| BR-034/035 | Access application operations; Interview owns initiation | Page load, exchange, reload, replacement session | No execution writes/model calls; valid reload reuses session |
| BR-036 | CandidateSession | Resolution and admission port | Exact 2-hour boundary; no sliding renewal |
| BR-037 | Assignment-serialized invitation replacement | Reissue, exchange old link, existing session resolve | Old invitation denied; active existing session not invalidated by reissue alone; rollback/concurrent issue |
| BR-038 | Live assignment access facts; Interview revocation coordinator | Exchange, protected reads, admission, active-run revocation | Revocation/exchange race; existing sessions denied; actual cancellation requires real Interview integration |
| BR-039 | Interview assignment/execution owner; access has no start operation | Reopen, reissue, initiation integration | Completed assignment not restarted; new interview uses a new assignment |

## 4. Component Breakdown

| Layer/component | Status | Responsibility |
|---|---|---|
| `domain/aggregate/access`, `domain/valueobject`, `domain/enum` | Administrative implementation reused; candidate additions planned | Account/candidate identity, role/status and immutable lifetime/scope predicates; no transport/crypto/ORM |
| `application/usecase/accessusecase`, `query/accessquery`, `dto` | Administrative implementation reused; candidate additions planned | Trusted time, current authority, detached views; migration policy adds no business operation |
| `application/repository/accessport`, `application/service` | Administrative ports reused; candidate collaboration Q2-dependent | Transactions, credential verification, session/assignment authority; no generic CRUD API |
| `infrastructure/repository/accessrepository`, `taskrepository` | Modified schema installation only; CRUD retained | Existing mappings/transactions retained; remove access AutoMigrate installer; future candidate models use SQL-owned schema |
| `infrastructure/service/accesscredentials`, `accessauthority` | Reused as-is for migration transition | Password/token/CSRF and real authority adapters preserve persisted credentials |
| `delivery/http/handler/accesshandler` | Reused as-is for migration transition; candidate transport planned | Preserve administrative contracts; candidate handlers remain T13 |
| `infrastructure/database/schema/`, `cmd/migrate/`, `migrations/pgsql/` | New | Versioned DDL, guarded legacy adoption, status/dirty/version handling; thin command and shared runtime schema check |
| `infrastructure/runtime`, `cmd/app` configuration | Modified | Replace startup DDL with compatibility check; preserve bootstrap, shared resources and secured routing |
| Integration/lifecycle fixtures, `scripts/verify_compose.py`, Makefile/Compose/Dockerfile, operational docs | Modified/new | Explicit pre-start migration setup, reproducible legacy adoption, recovery instructions; no public API/catalog change for migration |

Password hashing, token generation/digesting and CSRF use replaceable application ports because they cross the domain import boundary and need deterministic fault tests. Do not wrap pure role/lifetime predicates in I/O interfaces. Domain constructors/restorers own validity; repository restores corruption as technical failure. Handlers never decide roles on their own.

### Material structure decisions and assessment

| Choice | Reason and viable alternative/tradeoff | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| Separate administrative and candidate identity/session models | A single users table with CANDIDATE role would introduce unneeded password/account lifecycle and ambiguous scope; separate records express approved invitation-only identity at the cost of two resolvers | Owner approval; BR-029..033; five tables binding target | Retained; shared crypto helpers allowed, shared authorization semantics not assumed |
| Fixed ADMIN/HR policy without role tables | Known roles have distinct purposes; configurable RBAC adds management rules/tables not requested | Owner role approval, BR-021..023 | Retained; combined roles/new permissions need business revision |
| Account/session roots plus assignment-serialized access changes | A whole-company aggregate would serialize unrelated users; independent invitation CRUD would race revocation and weaken assignment ownership | BR-026/031/037/038 and platform lock order | Retained; transaction coordination across module ports is required, not a new giant aggregate |
| Opaque persisted sessions rather than self-contained JWT authority | Current role/revocation must be read anyway; opaque credentials avoid stale authority claims and support restart, with one indexed database lookup per action | BR-024..029/036; internal adapter choice | Retained; no new Redis dependency |
| Separate Candidate aggregate and contact value | Person identity survives assignments, while email-based upsert would invent matching policy | BR-030, Q4 | Identity retained; contact normalization/matching deferred to Q4 |
| Existing ACP authority port | It already separates trusted principal and operation/resource authorization; replacing it adds reconciliation without needed behavior | Verified interface, ACP T2; SubjectID stable account UUID | Retained; no type rename/code repair required; real adapter is net-new |
| Administrative tables before candidate/assignment additions | Fake assignment storage would make a false integration claim; blocking admin behind interview would unnecessarily delay access delivery | Approved five-table final scope | Retained phased installation; T8 administrative delivery is complete; candidate T11 depends on Q2/Q3/Q4 and migration foundation T20 |
| Argon2id passwords, random opaque token digests | Passwords need adaptive hashes; high-entropy tokens can use fast digests. Using one hashing policy for both adds cost or weakens passwords | Internal crypto choice; external references below | Retained engineering proposal; benchmark/verification in T5 |
| Handler-local envelope parity initially | Task helpers are private; broad transport extraction expands scope | Existing envelope is binding, helper placement internal | Retained; extraction allowed only with Task/voice regressions |
| Explicit shared SQL stream; GORM only for persistence | Startup AutoMigrate lacks a reviewed version history and can conceal schema drift. Keeping unused SQL beside it leaves two owners; a service-wide stream covers the actual three-table schema and orders future FKs | Owner's approved policy D8 supersedes the former AutoMigrate requirement; exact runner/package layout is an internal choice | Revised; T16–T20 replace schema ownership without changing business rules |
| Separate migration command and read-only startup guard | Automatic startup `up` would retain release-time DDL inside each app process. An explicit pre-start command makes execution deliberate, with a separate budget; startup can reject unprepared databases | Manual-migration policy D8; ten-second startup and bounded cleanup remain binding | Revised; no fallback to AutoMigrate; older DDL-running binaries excluded from routine rollback |
| Two baseline versions and guarded legacy adoption | One combined create-all file cannot cleanly adopt a Task-only database. Blind IF NOT EXISTS or forced version stamping can hide incompatible constraints | Existing Task/account/session compatibility; §3.3 manifest checks and exclusive coordination | Retained proposal; versions 1–2 separate Task from administrative schema; refusal on drift is deliberate |
| No seed accounts or role tables | Fixed enums need no reference rows; SQL account insertion would bypass approved hashing/provisioning and invite hardcoded credentials | BR-020/021; T1 no-overwrite bootstrap; approved five-table access scope | Retained; bootstrap runs only after compatible schema is established |

Assessment gate: role, lifetime, scope and business table boundaries are retained. Schema ownership is revised with owner approval (Q6 resolved), explicit baseline/adoption/error paths and proof plans in §3. Migration reconciliation needs both code and documents in T16–T20; this revision changes documents only. Q1 is resolved in §6; Q2–Q5 remain deferred with affected tasks in §9. No new test success is inferred from code presence or task checkmarks.

### Security adapter choices

Generate independent 32-byte CSPRNG invitation and session tokens; encode base64url without padding, persist SHA-256 digest only. Never use UUID, candidate ID or assignment ID as a credential. Digest/token namespaces and cookie names separate administrative from candidate access. Use Argon2id with individually random salt and encoded parameters; start assessment at OWASP's minimum 19 MiB, two iterations, parallelism one, then benchmark under bounded concurrency and cancellation before exposure. Malformed/unreasonably costly stored hashes fail safely without unbounded allocation. Unknown-account login follows equivalent verification work and generic rejection. Password acceptance and initial delivery follow §6.1; no email sender or forced-reset columns are invented.

Cookies are Secure, HttpOnly, SameSite=Lax, host-only with Path=/ and distinct `__Host-` names on HTTPS deployments. A cookie is a credential; session ID in JSON is metadata only. SameSite alone is insufficient: unsafe browser operations require exact configured Origin plus a session-bound CSRF header where authenticated. Select a signed HMAC CSRF value bound to credential kind and current session, derived using environment-supplied server secret; do not expose that secret or add a sixth table. Login/exchange use strict same-origin JSON/custom-header handling before a session exists. No permissive credentialed CORS. Bound login/exchange rate and hash concurrency before crypto work; administrative budgets and failure precedence are §6.2 and §6.5, not inherited from Task. Candidate exchange rate remains Q3-adjacent and uses the same hash-concurrency gate. Cookies/CSRF values and invitation/session credentials are excluded from logs and cacheable responses.

A link is a bearer credential, not independent proof that the holder is the named person. Candidate verification beyond possession remains Q4. Candidate frontend must remove the link token from visible URL/history before loading third-party resources, use no-referrer/no-store behavior, and avoid state-changing exchange on GET (including link preview requests). Q3 must settle repeated exchange/recovery before this frontend is mounted.

Technical references consulted for these adapter choices: [OWASP password storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html), [OWASP session management](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html), and [OWASP CSRF prevention](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html). These are security baselines, not evidence that this repository implements them. The pinned Go/crypto/GORM APIs must be checked during implementation.

## 5. Flow Sequence

### Administrative access and Q2

1. Operator explicitly migrates or adopts the database to the release target. Bounded startup verifies compatible clean schema, bootstraps initial ADMIN when applicable, and wires real ports. No secrets are logged. Until T18 is implemented, current startup still performs AutoMigrate.
2. Login applies origin/rate/input checks, verifies credentials, rechecks account eligibility and commits a new two-hour session before setting cookie.
3. ADMIN provisions HR through an operation that fixes the new role to HR; HR cannot provision accounts.
4. HR login resolves a stable account principal. ACP calls current authorization before any read, mutation or replay disclosure. ADMIN receives forbidden; candidate cannot resolve an administrative principal.
5. Logout persists revocation and clears the cookie. Later requests fail. Existing Task/voice routing and shutdown budgets remain unchanged.
6. Real authorizer/session evidence supplies the access part of ACP Q2; ACP T8/T9 must still prove actual secured configuration endpoints.

### Invitation access (after external assignment readiness)

1. HR selects/creates a candidate, then Interview prepares an assignment referencing that candidate and a pinned published revision.
2. HR requests invitation issuance; assignment lock coordinates replacement and access eligibility. Only the new credential is returned after commit.
3. Candidate browser opens the link without starting execution. Explicit POST exchange checks origin, invitation and assignment; it commits a scoped session before setting the candidate cookie.
4. Candidate page obtains safe session/assignment context. Reload uses the current cookie. Consent and sufficient validity for the execution deadline are checked by Interview before initiation.
5. Reissue blocks new exchanges using the old invitation, while existing session access remains governed by its session/assignment. Assignment revocation blocks existing and new access and coordinates execution cancellation.
6. A further interview for the same person creates a new assignment, not a reset of this one.

## 6. Request / Response

Accepted for `CORE-IAM-001` (T1 sections A–E) on 2026-11-01 and `CORE-IAM-010` (T10 sections F–I) on 2026-11-01. Use the established JSON envelope, strict fields and UTC millisecond response timestamps. No password hashes, token digests, salary policy, HR reports or database details in responses. No session credential in response JSON; only HttpOnly cookies. Invitation issuance remains the deliberate exception for a new invitation credential shown once to authorized HR; never return it on read.

### 6.1 Administrative identity, password and bootstrap (approved A)

Product policy is constrained by approved non-goals: no mail delivery, no public signup, no account-recovery UI, and no forced-reset columns. Initial HR passwords are chosen by ADMIN at provisioning and delivered out of band. Bootstrap uses environment credentials only for the first applicable ADMIN. The service never emails, SMS-sends or prints passwords.

**Name:** trim outer Unicode whitespace (Zs plus ASCII space/tab); 1–120 Unicode code points afterwards; reject C0/C1/DEL controls. Store and return the trimmed value.

**Email:** trim the same outer whitespace, then Unicode lowercase the entire address. Require exactly one `@`, nonempty local and domain parts, LDH domain labels, total 3–254 ASCII characters after canonicalization, local part 1–64. Reject display names, spaces, consecutive dots and IDN/punycode in v1. Store and return only the canonical form. Login canonicalizes email the same way. Do not strip `+` tags. Candidate contact uses this same canonicalization and remains nonunique (§6.6).

**Password acceptance (bootstrap and HR provision):** 8–64 Unicode code points; never trim or Unicode-normalize; reject C0/C1/DEL; no composition, dictionary or breach checks in v1. Login treats password as an opaque secret: never trim; empty or missing is `INVALID_INPUT`; length other than 1–64 is `INVALID_INPUT`; otherwise verify exactly against the stored hash. Disabled, unknown and wrong-password logins share one generic authentication failure after dummy verification work.

**Bootstrap environment (validated at startup; never logged):** `ACCESS_BOOTSTRAP_ADMIN_NAME`, `ACCESS_BOOTSTRAP_ADMIN_EMAIL`, `ACCESS_BOOTSTRAP_ADMIN_PASSWORD` must all be present and satisfy the field rules above. `ACCESS_HTTP_ORIGIN` is an exact absolute origin (`https://host[:port]` or `http://localhost[:port]`), no path/query/trailing slash. `ACCESS_CSRF_SECRET` is ≥32 bytes of high-entropy material. Missing/invalid configuration fails startup without creating accounts.

### 6.2 Cookie, origin, CSRF and rate budgets (approved B)

HTTPS administrative cookie name `__Host-ia-admin` (Secure, HttpOnly, SameSite=Lax, Path=/, host-only, no Domain). HTTP localhost may use `ia-admin` with the same flags except `__Host-`/Secure. Candidate names `__Host-ia-candidate` / `ia-candidate` are reserved and never accepted on administrative routes. Session ID in JSON is metadata; the cookie is the credential. All access responses send `Cache-Control: no-store`.

Every administrative JSON request requires `Origin` equal to `ACCESS_HTTP_ORIGIN` and `Content-Type: application/json` on requests with bodies. No credentialed CORS. Login has no CSRF header. Authenticated unsafe methods require header `X-CSRF-Token` matching an HMAC-SHA256 value bound to credential kind `administrative` and the current session ID, keyed by `ACCESS_CSRF_SECRET`. GET `/admin/session` returns that value as `csrfToken` after authentication. CSRF rotation invalidates outstanding tokens; persisted sessions remain valid.

Login rate limit, before Argon2id: 5 attempts per canonical email per 15 minutes and 20 per client IP per 15 minutes. Exceeded returns 429 `IA_RATE_LIMITED` with `Retry-After` in seconds (maximum 900). Process-wide Argon2id concurrency is 2; waiters respect request cancellation. Candidate exchange reuses the hash gate; its request-rate policy is §6.7. Provision/logout are not login-rate-limited.

### 6.3 Administrative operation schemas (approved C)

Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, trailing documents, nulls where not allowed, and system-owned fields (`id`, `role`, `status`, `createdBy`, timestamps, session/cookie/CSRF values) are `INVALID_INPUT`. Success wraps the `data` objects below; listed members are required.

| Operation | Exact request | Response `data` | Status |
|---|---|---|---|
| POST `/admin/sessions` | exactly `email`, `password` | `account` `{id,name,email,role}` plus `session` `{id,expiresAt}`; Set-Cookie after commit | 201 |
| GET `/admin/session` | cookie only | `account` `{id,name,email,role,status}`, `session` `{id,expiresAt}`, `csrfToken` | 200 |
| DELETE `/admin/sessions/current` | cookie + `X-CSRF-Token`; empty object or absent body | `{}` after persisted revocation, then expired Set-Cookie | 200 |
| POST `/admin/hr-accounts` | exactly `name`, `email`, `password`; CSRF | `id`, `name`, `email`, `role` (`HR`), `status` (`ACTIVE`), `createdAt`, `updatedAt`; no password | 201 |

### 6.4 Duplicate accounts and existing bootstrap state (approved D)

Canonical administrative email is unique across all roles. Duplicate provision, including races and retry after uncertain commit, returns 409 `IA_ACCOUNT_CONFLICT` without existing IDs, roles or password reset.

Bootstrap: hash outside the DB transaction; serialize initialization. If no `administrative_accounts` rows exist, insert ADMIN with `created_by` null. If any row exists and canonical env email matches an ACTIVE ADMIN, skip and do not overwrite password, name or role. If the matching row is HR, DISABLED, or otherwise incompatible, fail startup. If other accounts exist and the env email is absent, skip (already initialized); do not create a second bootstrap ADMIN. Concurrent first inserts collapse to one ADMIN via uniqueness.

### 6.5 Public errors and precedence (approved E)

Planned codes for later `pkg/errs` and EN/ID catalogs, not T1 runtime additions: `IA_UNAUTHENTICATED` 401; `IA_FORBIDDEN` 403; `INVALID_INPUT` 400; `NOT_FOUND` 404; `IA_ACCOUNT_CONFLICT` 409; `IA_INVITATION_UNAVAILABLE` 410; `IA_RATE_LIMITED` 429; `IA_DEPENDENCY_UNAVAILABLE` 503; `INTERNAL_ERROR` 500.

| Priority | Condition | HTTP / code |
|---|---|---|
| 1 | Method, unknown route, shutdown admission | existing 405/404/503 `SHUTTING_DOWN` |
| 2 | Origin, content-type, size, JSON, unknown fields | 400 `INVALID_INPUT` |
| 3 | Login rate budget exceeded | 429 `IA_RATE_LIMITED` |
| 4 | Missing/invalid/expired/revoked administrative session, including candidate-cookie substitution | 401 `IA_UNAUTHENTICATED` |
| 5 | Invalid CSRF on an authenticated administrative request | 403 `IA_FORBIDDEN` |
| 6 | Valid session, insufficient role | 403 `IA_FORBIDDEN` |
| 7 | Field policy (name/email/password bounds) on reached input | 400 `INVALID_INPUT` |
| 8 | Duplicate canonical administrative email | 409 `IA_ACCOUNT_CONFLICT` |
| 9 | Login proof failure (unknown, disabled, mismatch, stale hash) | 401 `IA_UNAUTHENTICATED` |
| — | Missing/expired/revoked/malformed invitation without disclosing assignment | 410 `IA_INVITATION_UNAVAILABLE` |
| — | Store unavailable before known success, including logout/login commit uncertainty | 503 `IA_DEPENDENCY_UNAVAILABLE` |
| — | Corrupt storage or unexpected technical error | 500 `INTERNAL_ERROR` |

Order applies to checks actually reached. Login evaluates 1–3 and field presence before proof; it never returns 409. Protected operations authenticate then authorize before target disclosure. Technical failures terminate the current stage. Set no cookie or success payload before confirmed session commit. Logout commit uncertainty clears the local cookie defensively and returns 503 without claiming revocation succeeded. Indistinguishable invitation rejection is uniform 410; exchange reuse and lost-response recovery follow §6.7. ACP routes keep ACP §6.5 codes.

#### Examples (accepted expectations)

Login request before canonicalization:

```json
{"email":"Sinta@Example.com","password":"correct-horse"}
```

Login success `data` (cookie omitted):

```json
{"account":{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"Sinta Wijaya","email":"sinta@example.com","role":"HR"},"session":{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","expiresAt":"2026-11-01T10:00:00.000Z"}}
```

HR provision request:

```json
{"name":" Sinta Wijaya ","email":"Sinta@Example.com","password":"correct-horse"}
```

HR provision success `data`:

```json
{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"Sinta Wijaya","email":"sinta@example.com","role":"HR","status":"ACTIVE","createdAt":"2026-11-01T08:00:00.000Z","updatedAt":"2026-11-01T08:00:00.000Z"}
```

Current session `data`:

```json
{"account":{"id":"aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa","name":"Sinta Wijaya","email":"sinta@example.com","role":"HR","status":"ACTIVE"},"session":{"id":"bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb","expiresAt":"2026-11-01T10:00:00.000Z"},"csrfToken":"signed-csrf-material"}
```

Rejected cases: `Sinta@Example.com` provisioned twice → 409; env bootstrap on restart against the same ACTIVE ADMIN → no overwrite; matching env email on an HR row → startup failure; login `Sinta@Example.com` / wrong password → 401 with no cookie; origin mismatch → 400.

### 6.6 Candidate contact and matching (approved F)

Candidate name and email reuse §6.1 canonicalization and limits. Email is contact only: never unique, never an upsert or matching key, and never a login identifier. POST `/admin/candidates` always allocates a new candidate ID. Reuse across interviews is only by supplying that known `candidate_id` on an Interview-owned assignment. Two rows may share the same canonical email and remain distinct people. HR selects an existing person through GET `/admin/candidates/{id}`, not through email search. The bearer link proves possession of the invitation secret, not independent personal identity. No additional verification, OTP or password is added. Contact changes after creation are out of scope; they must not rewrite Interview historical snapshots.

### 6.7 Invitation exchange reuse and recovery (approved G)

Keep the five-table properties: no consumed_at column and no extra device table. A recognized, unrevoked invitation before its 24-hour expiry may be exchanged repeatedly. Each successful exchange inserts a new two-hour candidate session bound to that assignment; concurrent devices may hold distinct current sessions. Reissue revokes only the former invitation credential; existing sessions remain governed by session lifetime and live assignment access (D6). GET `/candidate/session` never exchanges or starts an interview.

Lost exchange response: the client may retry the same invitation. If the invitation is still valid, retry may commit another session; the server does not reconstruct a previous cookie from memory. Commit uncertainty returns 503 `IA_DEPENDENCY_UNAVAILABLE` with no Set-Cookie. Concurrent exchanges serialize on the assignment lock; both may succeed with distinct sessions. Exchange rate, before digest work: 5 attempts per invitation digest per 15 minutes and 20 per client IP per 15 minutes, returning 429 `IA_RATE_LIMITED`; the process hash-concurrency gate of 2 is reused. Uniform 410 `IA_INVITATION_UNAVAILABLE` covers missing, malformed, expired, revoked and assignment-inaccessible invitations without disclosing assignment identity.

### 6.8 Candidate operation schemas (approved H)

Request bodies are limited to 1 MiB with the same strict JSON rules as §6.3. Exchange accepts exactly `token`; caller role, candidate ID and assignment ID overrides are `INVALID_INPUT`. HTTPS candidate cookie `__Host-ia-candidate` / localhost `ia-candidate` uses the administrative flag set. Authenticated candidate unsafe methods require `X-CSRF-Token` bound to credential kind `candidate`.

| Operation | Exact request | Response `data` | Status |
|---|---|---|---|
| POST `/admin/candidates` | exactly `name`, `email`; current HR + CSRF | `id`, `name`, `email`, `createdAt`, `updatedAt` | 201 |
| GET `/admin/candidates/{id}` | current HR | same identity fields; 404 after authorization when unknown | 200 |
| POST `/admin/interview-assignments/{id}/invitations` | empty object or absent body; current HR + CSRF | `id`, `assignmentId`, `expiresAt`, `token` once | 201 |
| POST `/candidate/access/exchange` | exactly `token`; origin/content-type | `candidate` `{id,name}`, `assignment` `{id}`, `session` `{id,expiresAt}`; Set-Cookie after commit | 201 |
| GET `/candidate/session` | candidate cookie | `candidate` `{id,name}`, `assignment` `{id}`, `session` `{id,expiresAt}`, `csrfToken`; no execution | 200 |

#### Candidate examples (accepted expectations)

Create candidate request:

```json
{"name":" Andi Pratama ","email":"Andi@Example.com"}
```

Create candidate success `data`:

```json
{"id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","name":"Andi Pratama","email":"andi@example.com","createdAt":"2026-11-01T08:00:00.000Z","updatedAt":"2026-11-01T08:00:00.000Z"}
```

Exchange request:

```json
{"token":"invitation-secret"}
```

Exchange success `data` (cookie omitted):

```json
{"candidate":{"id":"cccccccc-cccc-4ccc-8ccc-cccccccccccc","name":"Andi Pratama"},"assignment":{"id":"dddddddd-dddd-4ddd-8ddd-dddddddddddd"},"session":{"id":"eeeeeeee-eeee-4eee-8eee-eeeeeeeeeeee","expiresAt":"2026-11-01T10:00:00.000Z"}}
```

Rejected cases: two creates with `Andi@Example.com` → two candidate IDs; exchange with extra `assignmentId` → 400; expired or revoked invitation → 410 with no cookie; GET exchange → not an exchange operation.

### 6.9 Assignment collaboration and locking (approved I)

Interview owns table `interview_assignments`. Access never creates, migrates or fakes that table. Required access facts, read only after `SELECT … FOR UPDATE` on the assignment row: stable assignment ID; `candidate_id` referencing `candidates(id)` RESTRICT; pinned `published_revision_id`; `access_revoked_at` nullable; `completed_at` nullable terminal marker. Invitation and session rows store `assignment_id` as a real FK to that table once Interview's migration exists; until then T11 must not install invitation/session FKs against a placeholder.

`AssignmentAccess.WithAssignment` locks the assignment first, then runs invitation or session writes inside that boundary. `AssignmentFacts.PermitsInvitation`, `PermitsExchange` and `PermitsSessionAccess` deny when identity is missing or access is revoked. Terminal completion does not deny access and never authorizes creating, restarting or cancelling execution. Consent, admission duration checks and active-run cancellation remain Interview-owned. Cross-assignment composite FKs on invitations/sessions prevent binding a session to another assignment's invitation.

### Creation, restoration, and transition contracts

| Path | Accepted states / normalization | Errors/order and timing | Effects / compatibility | Proof plan |
|---|---|---|---|---|
| Bootstrap/provision | Stable new ID; fixed role; canonical email and accepted password §6.1; bootstrap creator null only | §6.5; unique conflict before success | Hash only; no reset on startup/retry | Concurrency, collision, rerun, no credential leakage |
| Login | Active ADMIN/HR credentials; password never trimmed/normalized; email §6.1 | Origin/rate/input then generic authentication result; store failure technical | New session and fresh token only after commit | Unknown/disabled/wrong credentials; failure/cancel; equality expiry |
| Restore identity/session | Recognized role/status/IDs; valid stored time order and references | Corruption technical; never silently convert unknown role into HR | No repair, clock reset or credential reissue on read | Round trip and corrupt persisted data |
| Current authority / logout | Current matching session/account; `now < expires_at`; unrevoked | Authentication then operation authorization before target disclosure; §6.5 | Logout revokes current session only; no role from client | Candidate cookie substitution, disabled account, replay, no side effects on denial |
| Candidate creation / assignment reuse | Existing identity can be referenced repeatedly; email not unique/upsert; §6.6 | Current HR before candidate disclosure; contact bounds §6.6 | New candidate distinct from assignment; no historical-input rewrite | Same candidate two assignments; same email does not merge people |
| Invitation issue/reissue | Existing eligible assignment; trusted 24h expiry; prior invitation revoked | HR then assignment ownership/existence, serialize on assignment | Old link invalid; existing sessions/execution not automatically cancelled | Concurrent reissue, rollback, lost response §6.7 |
| Invitation exchange | Exact secret proof, live invitation and assignment, reusable until expiry/revocation | Reject missing/invalid/expired/revoked proof as uniform 410 before session creation; technical failure distinct | New 2h access session only; no consent/execution/model call | Exact expiry, token collision, races, retry §6.7 |
| Candidate session resolve | Live scoped session plus accessible assignment | Outside-session resource hidden as 404; unauthenticated as 401 | No new session/expiry extension on GET; invitation provenance not ongoing validity | Cross-assignment access, reload, source invitation reissued |
| Assignment revoke / admission | Interview contract §6.9; lock assignment before execution | Serialize before acceptance; no admission from stale access facts | Live access removed; execution cancellation belongs to Interview | Real consumer race and cancellation integration T14 |

## 7. Error Handling

Public access codes and overlap order are §6.5. Access codes, including `IA_INVITATION_UNAVAILABLE`, are in the runtime catalog; candidate endpoint behavior remains T13. Migration failures are command/startup errors, not new HTTP codes. Exchange reuse and commit-uncertain recovery follow §6.7.

Authentication endpoints validate transport/origin/rate before proof; protected operations authenticate then authorize before target lookup/disclosure. Technical failures terminate the reached stage and are mapped before nested domain errors; do not mask store outage as bad password or assume failed commit is rollback.

For ACP routes preserve its already approved §6.5 order and codes: absent/invalid/candidate session → 401 `ACP_UNAUTHENTICATED`; valid ADMIN or invalid CSRF → 403 `ACP_FORBIDDEN`; outside HR scope → 404; replay/current-draft ordering unchanged. The shared adapter returns classified failures; each feature's delivery mapping owns its public catalog.

## 8. Constraints Checklist

These are implementation verification gates, not completed checks.

- [ ] BR-CORE-020..039 covered through public behavior at the owners above.
- [ ] Exactly five access business tables; migration metadata counted separately; no invented assignment table, role tables, tenant tables or candidate password.
- [ ] ADMIN and HR remain distinct; stable account SubjectID, current HR authorization and shared company scope proven.
- [ ] Cookie/session/CSRF/credential protections proven with real adapter and browser-compatible HTTP tests.
- [ ] Candidate session scope, expiry, source provenance and assignment revocation proven with real assignment owner.
- [ ] Access exchange/reload has no consent/execution/provider side effects.
- [ ] Explicit service-wide SQL migrations replace all runtime/test AutoMigrate callers; schema guard precedes bootstrap; architecture, documentation and lifecycle budgets preserved.
- [ ] Fresh, Task-only and populated administrative adoption preserve data; drift/dirty/lock failure and disposable reversal/recovery verified.
- [ ] Active OpenAPI, endpoint docs, changelog, codes and EN/ID messages synchronized when implemented.
- [ ] No test/database acceptance, historical RED/GREEN, or Q2 completion inferred from this document.

## 9. Risks & Open Questions

| ID | Decision / owner | Affected tasks | Ready when |
|---|---|---|---|
| Q1 | **Resolved 2026-11-01:** T1 A–E; out-of-band ADMIN-chosen HR passwords; canonical email; bootstrap existing-state; cookies/CSRF/rate; error order in §6 | T1–T9 recorded Done | Approval D7 retained; migration work must preserve this contract |
| Q2 | **Resolved 2026-11-01 (T10):** Q2 assignment facts, assignment-first locking and FK ownership specified in §6.9; Access never creates `interview_assignments` | T11 invitation FKs still wait for Interview's migration; T14 proves real consumers | Runtime Interview adapter and real assignment persistence remain T14; no in-memory fake counted as integration |
| Q3 | **Resolved 2026-11-01 (T10):** reusable invitation until expiry/revocation; each exchange may create another session; lost-response retry is not exactly-once; no `consumed_at` | T12 operations; T13 HTTP/cookies | Schema stays five access tables; extra columns were assessed and rejected |
| Q4 | **Resolved 2026-11-01 (T10):** candidate name/email follow §6.1 canonicalization; email is nonunique contact; reuse is by known `candidate_id`; bearer link proves possession | T11 candidate table; T12 create/read | Matching/upsert by email remains forbidden |
| Q5 | Product/data owner: retention and FK-safe purge of contacts, credentials and assignment-bound references | T15 cleanup only | Explicit retention/purge ordering and data-loss consequences accepted; no physical purge before then |
| Q6 | **Resolved 2026-11-01:** owner explicitly selected manual migrations because the project has grown; see D8 | T16–T20; T11 must use the completed foundation | Policy decision is approved; no further policy confirmation needed. Target execution still requires an authorized database and inspected schema. |

Operational readiness is distinct from Q6 approval: no target database inventory, backup/restore rehearsal, pinned runner version or migration runtime evidence exists yet. T16–T20 establish these before cutover. Legacy drift blocks that target's adoption only; it does not block drafting artifacts or isolated verification. Baseline locking must cover catalog comparison and metadata adoption together. Future version allocation must account for the real assignment migration rather than assuming its table appears automatically.

Administrative foundation and ACP Q2 access delivery do not depend on candidate Q2/Q3/Q4 decisions; use the two-table administrative phase. Q1 is resolved. Proposed account disable/reset APIs remain outside scope, although current DISABLED state must always be denied. Multiple identities for an ADMIN who also needs HR duties are the consequence of the approved single-role model; combined-role access requires a later business decision.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-01 | Initial Mode A capability-scoped assessed design; five-table target and staged readiness | Owner approved scope, properties, capability name and semantic BR blocks; satisfy ACP's real access dependency without inventing interview implementation |
| v1.1 | 2026-11-01 | Incorporate T1 A–E administrative contracts and resolve Q1 | Executable HTTP/schema/bootstrap contract for T4/T6/T8 without runtime implementation |
| v1.2 | 2026-11-01 | Mode C: align administrative baseline with T1–T9 records; plan shared Task/access SQL versions, legacy adoption, explicit runner, startup guard, seed assessment and T16–T20 | Owner approved replacing AutoMigrate with manual migrations as the project grew; retain data/business contracts and candidate decision gates |
| v1.3 | 2026-11-01 | Incorporate T10 F–I candidate, exchange and assignment collaboration contracts; resolve Q2/Q3/Q4 | Executable candidate/session contract for T12; T11 invitation FKs still wait for Interview migration and T20 |
