# T18–T20 migration foundation evidence

Date: 2026-11-01. Tickets: **CORE-IAM-018**, **CORE-IAM-019**, **CORE-IAM-020**.
Scope: runtime/test schema ownership, real migration acceptance, disposable cutover/recovery handoff. No non-disposable database was inspected or changed. T16/T17 records and unrelated workspace work retain their own ownership.

## Design and assessment

Initial mapping follows technical-design §3.1–3.4, D8, and tactical-strategy §5–6. No domain invariants or HTTP contracts change. `schema.Runner` owns explicit SQL/coordination; `schema.RequireCompatible` owns the read-only clean-version-2 contract; runtime owns connection cleanup and startup order; Compose/operators own preparation and credentials. Existing Task aggregate, queue reservation/commit/publication and administrative transaction/security contracts remain unchanged.

Assessment: **retained with rationale**. Four root layers, separate CRUD models and bounded lifecycle ownership remain sufficient; no new application port or general schema abstraction is needed. SQL mocks use sqlmock; no new interface mocks were needed, so no handwritten mock or Mockery output was introduced. GORM v1.25.12, pgx v5.9.2 and golang-migrate v4.19.0 remain the checkpoint's pinned dependencies; the existing selected APIs and migration driver's dirty-target semantics were exercised locally.

Refinements: candidate integration setup now loads existing versioned candidate SQL around its explicitly synthetic Interview assignment fixture and removes those fixture tables afterward. This prevents candidate fixtures contaminating administrative schema acceptance; it does not accept a real Interview migration. Lifecycle fixtures now supply valid bootstrap settings so they reach the intended database/listener paths. Runtime blocked-DDL testing became blocked metadata-read testing; migration DDL cancellation remains in the runner suite. Recovery holds the blocker until PostgreSQL has observed pgx's asynchronous cancellation before checking the preserved catalog.

The pre-existing unstaged runner change rejecting zero/negative unsupported versions is retained and included as a necessary T19 invalid-metadata regression prerequisite. Unrelated dependency upgrades, T16/T17 task-status edits, legacy-fixture whitespace and other feature documents are excluded from this checkpoint.

## RED checkpoint

Working directory: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./infrastructure/database/schema ./test/architecture -run 'TestRequireCompatible|TestSchemaHasOneOwner' -count=1
```

Observed RED: the inert compatibility guard accepted old/dirty/ambiguous metadata and did not read SQL; the architecture scenario found executable AutoMigrate in production and integration setup. The only non-test scaffolding was the `RequireCompatible(context.Context, *sql.DB) error` declaration with an inert nil return, required to compile/run the behavioral assertions.

The identical staged snapshot was extracted to `/var/folders/v0/yg69hwhd21lfjjbn5bmfss6h0000gn/T/iam-red-g3pyq6j4` and the same command failed for the intended reasons before committing.

- SHA: `e5451f865a2a02fb4bc92407ec63f155c8fdf2c2`
- Actual subject: `test: [CORE-IAM-018][rfanazhari] RED cover schema compatibility and exclusive migration ownership`
- One coherent RED/GREEN unit covers T18 implementation and T19/T20 supporting acceptance/rehearsal artifacts; related tickets remain separately traceable above.

## GREEN commands and results

Commands run from the repository root unless the staged-snapshot validation notes a temporary root. Local Go cache/socket/Docker access required running checks outside the restricted sandbox; initial access-denied runs are tooling failures, not behavioral RED. No toolchain or check was disabled.

| Command | Result | Evidence boundary |
| --- | --- | --- |
| Same RED command above | passed | sqlmock metadata guard and source ownership |
| `go test ./infrastructure/database/schema ./test/integration -run 'TestPopulatedAdoption\|TestRuntimeRequiresPreparedSchema' -count=1` without resource variables | skipped resource scenarios | Compilation only; not PostgreSQL proof |
| `make test` | passed | Domain/application race and coverage |
| `make test.all` | passed | All packages with race; resource suites skip without explicit database variables |
| `make test.contract` | passed | Published OpenAPI unchanged |
| `make test.architecture` | passed | Dependency, declaration documentation and no-AutoMigrate checks |
| `make lint` | passed | Zero issues |
| `make test.security` | passed | Zero reachable vulnerabilities; checkpoint's original dependencies also scanned independently |
| `make test.docker` | passed | Real isolated PostgreSQL migration/repository/runtime/admin/Task and restore/adoption proof |
| `make docker.verify` | passed | Packaged executable/SQL, container HTTP, optional services, same PostgreSQL suites and recorded compatible-image rollback |
| `git diff --check` | passed | Scoped whitespace validation |

Exact Make operations exercised inside isolated verification, with URLs supplied only through environment:

```sh
make migrate.up MIGRATION_VERSION=2
make migrate.status
make migrate.down MIGRATION_STEPS=2 MIGRATION_DISPOSABLE=yes
make migrate.up MIGRATION_VERSION=2
make migrate.baseline
make migrate.up MIGRATION_VERSION=2
make migrate.status
```

The full-stack path additionally invokes the packaged `migrate up` as a separate Compose job and `make docker.up` with read-only preflight. The app has no DDL dependency hook.

## PostgreSQL and rehearsal results

- Fresh up-to-2, repeated no-op, disposable down/up, independent catalog equivalence and zero SQL-seeded accounts passed.
- Populated Task-only baseline 1 → 2 and full-three-table baseline 2 preserved complete rows, IDs, contacts, hash equality, creator links, original expiry/revocation and Task history. Equality assertions suppress row contents. A previously issued live session still resolves; the retained revoked session remains denied.
- Partial/column/index/FK drift, wrong index definition/FK action, foreign/ambiguous/zero/future metadata and dirty state refused. Failed baseline never stamped metadata.
- Concurrent migrators converged; held advisory lock hit the five-second bound. Injected DDL failure and cancelled DDL failed without automatic force. Fixture-specific catalog-verified repair restored clean 2 and no-op up; it is not a generic production repair command.
- Runtime missing/empty/unreadable/ambiguous/old/future/dirty metadata refused before listener/bootstrap and left no labeled sessions. A separately provisioned role denied CREATE/ALTER/DROP and metadata UPDATE while real runtime bootstrap/health/restart/shutdown passed. Blocked metadata-read cancellation and partial-startup cleanup passed.
- Task queue refusal/deferred commit rollback/repeated completion, ADMIN/HR login/provision/session/logout and persistence/restart passed. Existing candidate fixtures also ran, without claiming real Interview acceptance.
- Rehearsal stopped writers, dumped populated rows, restored an independent clone, verified equality, removed only the clone's metadata to model legacy state, ran guarded baseline/up/status, and repeated Task/admin smoke. Full-stack rehearsal recorded the known compatible image and recreated it by immutable ID against retained version 2, then passed Task/admin HTTP smoke. Only that recorded image's compatibility is proven; no distinct historical release was supplied.

All verification project containers/volumes were removed. Normal Compose volumes remain intact. Ordinary `make test.all` skips opt-in SQL suites; Docker results above are the actual resource evidence. No browser or external ACP/Interview consumer acceptance is added. Candidate T11 can consume the clean-2 foundation but still owns stream ordering and actual assignment-FK prerequisites.

## Handoff

Use [the runbook](../../../design/database-migrations.md), [baseline catalog](../../../design/database-schema-baseline.md) and [release checksums](../../../design/database-migration-checksums.sha256). Target cutover still requires separate authorization, an inspected catalog, protected verified backup, provisioned roles and a maintenance window. Rollback retains data and uses the verified migration-aware image; destructive down and unchecked force are not retained-data recovery.

The GREEN SHA/actual subject are returned by the validated phase-commit command and included in the delivery response; the evidence file does not attempt to contain its own commit hash.
