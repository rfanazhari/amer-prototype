# T16–T17 execution evidence

Tickets: CORE-IAM-016 and CORE-IAM-017 (local planning IDs). Scope: legacy catalog capture, approved policy synchronization, SQL 000001/000002 and explicit migration tooling. T18–T20 remain separate.

## Design mapping and assessment

No domain, application or HTTP contract changes. Schema evolution belongs to infrastructure/database/schema and cmd/migrate. Existing CRUD models, time semantics, Task event transactions and runtime wiring are preserved. The initial mapping follows technical-design §3.1–3.3 and D8.

Retained: two baseline versions allow Task-only adoption; no seed data; dirty-state refusal and stopped legacy writers are deliberate. Refined: use golang-migrate v4.19.0 with a context-bound database driver because the upstream pgx driver uses context.Background for lock/version operations. Every SQL operation shares a bounded command context. Allow 59 seconds for work plus one second for cleanup, with five-second server lock waits. The selected dependency's NewWithInstance, Migrate, Steps, ErrNoChange, database.Driver and GenerateAdvisoryLockId APIs were inspected and exercised. Artifact checksums are pinned and bytes snapshotted before execution.

Assessment: retain infrastructure ownership and small separation between orchestration, driver, catalog adoption and artifact loading. No business model/port refactor is justified. SQLmock handles SQL mocking; no service-owned interface was added or mocked, so Mockery generation is not applicable. Existing generated mocks are preserved.

## Observed evidence

Initial cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

- Captured legacy schema using a temporary Go test invoking the existing Task and administrative AutoMigrate installers on an empty disposable postgres:17 container. PostgreSQL reported 17.11. Frozen SQL/catalog facts are linked from database-schema-baseline.md. Temporary capture code was removed after extraction. No retained/deployed target was inspected.
- RED command: `GOCACHE=/private/tmp/iam-go-cache go test ./infrastructure/database/schema -run 'TestStatusReadsOnly|TestOptionsRefuseUnsafeOperations' -v`. TestStatusReadsOnly failed with `not implemented`; this was executable missing behavior, not a compile/resource failure. Only the Runner/Options declarations and inert Execute stub accompanied tests.
- RED SHA: `2269b540e7b40a46451338613fb10f3a97aa1757`.
- RED subject: `test: [CORE-IAM-017][rfanazhari] RED cover read-only migration status and command guards`.
- GREEN scenario command: the same command passed after implementation.
- Real migration command: `GOCACHE=/private/tmp/iam-go-cache go test -race ./infrastructure/database/schema -count=1 -v`, with TEST_MIGRATION_DATABASE_URL set only to the disposable container. Passed fresh up-to-2 and exact catalog equivalence, rerun, down/up, full legacy and Task-only adoption, partial/column/index/FK drift refusal, foreign metadata, dirty refusal, immutable artifact refusal, failed DDL retaining dirty metadata, blocked-DDL cancellation and five-second lock timeout. No secret row values were emitted.
- `make test`: passed from `/private/tmp/iam-t16-t17-check` (exported snapshot). Domain/application race and coverage checks passed.
- `GOCACHE=/private/tmp/iam-go-cache GOLANGCI_LINT_CACHE=/private/tmp/iam-lint-cache make test.all test.contract test.architecture lint test.security`: passed from the repository root with loopback/network access. Lint reported zero issues; govulncheck reported zero reachable vulnerabilities, plus one imported-package and three required-module findings not called by this code.
- Without TEST_DATABASE_URL, all-package checks skip integration/smoke database suites. The explicit migration suite above supplies separate real PostgreSQL proof.

## Concurrent workspace and checkpoint limits

Another active process committed T13/T14 while this task ran. Commit `d9da8e7` (T14 RED) inadvertently captured the staged T16–T17 implementation. Its subject is `test: [CORE-IAM-014][rfanazhari] RED cover candidate access interview integration proof`; it is a mixed checkpoint, not a valid T17 GREEN implementation checkpoint. History was preserved. No valid T17 GREEN checkpoint has been created: the full Docker check remains failed. Implementation provenance includes d9da8e7. No claim of an isolated two-commit T17 implementation history is made.

An initial exported-snapshot make check failed because sandbox loopback listeners were denied and the concurrent T14 evidence file was absent. A Docker regression attempt failed on the concurrent T14 candidate fixture. These are recorded failures, not T17 schema-test successes. Subsequent verification outcomes supersede them only for the commands actually rerun.

## Remaining scope

Runtime/test installers still use AutoMigrate until T18. T19 owns populated row snapshots, wider concurrency/failure acceptance and DDL-denied runtime proof; T20 owns cutover/backup/restore rehearsal and release handoff. Candidate SQL 000003/000004 and public contracts were not changed by this work. Baseline on a retained target still requires stopped legacy writers, exact target inspection and a restorable backup.

## Final disposition

T16 Done; T17 In Progress: implementation and focused runner tests pass, but the mandatory full Docker check failed, so the gocraft GREEN phase is incomplete. No T17 GREEN SHA/subject exists. Keep T18–T20 scope unchanged.

The final root `make test.docker` run failed on administrative schema installation assertions (candidate tables already present), startup bootstrap configuration and existing migration/shutdown lifecycle cases. Logs: `/private/tmp/iam-docker-final.log`. These cases are outside the T16–T17 runner implementation; no unrelated repair was made. The earlier exported-snapshot Docker run also failed on the concurrent T14 candidate fixture. Passing unit checks do not supersede failed real stack verification.

The final Make entry-point checks exercised `make migrate.up MIGRATION_VERSION=2`, `make migrate.status`, `make migrate.down MIGRATION_STEPS=1 MIGRATION_DISPOSABLE=yes`, and up-to-2 again with MIGRATION_DATABASE_URL supplied only via environment on the disposable T16 PostgreSQL container. Final focused runner results are reported in the handoff.
