# T18 verification — CORE-IRT-018

## Result and boundaries

Production composition now mounts session-attempt and readiness. Generic lifecycle operations own durable UUID/generation authority; interview domain rules retain stages, coverage, reconnect counters and the fixed deadline. All execution, progress, command, pin and candidate/controller rechecks join one transaction context, locking execution before children. Provider I/O stays outside SQL.

Reservation states are pending, succeeded, failed and legacy. Pending never recreates a provider session; successful replay retains the original answer and execution version. The owner approved keeping version-10 rows as legacy with 409 on replay. No SDP backfill, seed, runtime migration, classifier, reporting, candidate-page integration or live acceptance was added. Q1/Q2 remain open. Integration Q3 has real proof.

## RED checkpoint

Cwd for every command below: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
GOCACHE=/private/tmp/t18-go-cache go test ./domain/aggregate/execution ./application/usecase/executionusecase ./application/usecase/interviewusecase -run 'TestExecutionRecordAttempt|TestRecordAttempt|TestAttemptReservationReplay' -count=1
```

Observed RED: replacement authority and bounded admission were missing; pending/failed/legacy replay returned a success without an original answer. Compile-required scaffolding was the domain method stub, generic operation stub, input/ledger/command-state declarations and Mockery ledger mock. These stubs returned inert errors; no admission behavior was implemented in RED.

- SHA: `07632238db3b261c6db171af7f25f68adb61e9b0`
- Actual subject: `test: [CORE-IRT-018][rfanazhari] RED cover durable attempts and reserved replay`
- The first cache-permission failure was tooling failure and is not RED evidence.

A later public HTTP scenario with a one-connection pool also failed meaningfully: the candidate recheck acquired a second pool connection while the execution transaction held the only one. Joining that read to the shared transaction made the same scenario pass. A third-episode regression also observed RED because refusal did not fail the same execution; the corrected orchestration calls Fail after the rejected progress transaction. No historical RED is claimed for other added characterization/regression tests.

## GREEN commands

The RED command above passes on the implementation. Additional selected checks:

```sh
GOCACHE=/private/tmp/t18-go-cache make mocks
GOCACHE=/private/tmp/t18-go-cache make test
GOCACHE=/private/tmp/t18-go-cache make test.all
GOCACHE=/private/tmp/t18-go-cache make test.contract test.architecture
GOCACHE=/private/tmp/t18-go-cache GOLANGCI_LINT_CACHE=/private/tmp/t18-lint-cache make lint
GOCACHE=/private/tmp/t18-go-cache make test.security
python3 /private/tmp/t18-db-test.py make test.integration
python3 /private/tmp/t18-db-test.py go test ./infrastructure/database/schema -run TestInterviewIntegrationMigrationPostgres -count=1 -v
```

The temporary wrapper derives the loopback port of container `core-irt-018-check`, sets `TEST_DATABASE_URL` for disposable database `t18`, `TEST_MIGRATION_DATABASE_URL` for separate database `t18_migrations`, and the temporary Go/lint cache paths. URLs stay in process environment. PostgreSQL 17 runs in an isolated container without development volumes. The integration acceptance run starts from a newly created disposable database because older corruption fixtures are not repeatable on retained test data.

GREEN checkpoint subject: `feat: [CORE-IRT-018][rfanazhari] GREEN integrate interview runtime with durable authority`. The actual SHA is reported in the final handoff; resolve it with `git log -1 --format='%H %s' --grep='GREEN integrate interview runtime with durable authority'`.

## Observed proof

- Mockery supplies interface mocks; sqlmock verifies nullable legacy reads, scoped finalization SQL and joined transaction rollback. No handwritten interface mocks were introduced.
- Domain/application: monotonic replacement, stale version/generation rejection, fixed activation instant, attempt exhaustion and exact window equality, rejected callbacks, deadline refusal, safe replay states and frozen recovery orchestration.
- Production router with fake HTTP/WebSocket provider: setup/readiness, original SDP replay without a second create, changed intent, pending concurrent calls, restored interrupted/legacy commands, empty key before reservation, wrong assignment/controller and result-persistence rollback. Readiness leaves all coverage pending. One-connection composition completes without pool starvation.
- Real PostgreSQL: authority survives new adapter/store reads; stale version, replaced UUID and terminal execution never invoke the callback; callback failure rolls back progress, transcript, command and generic progress-reference writes. Cancellation and progress serialize on the execution lock.
- Real T10 adapter: handoff/source close commit together, rejected outer callback rolls both back, target stays absent before opening, and a failed create retries the committed target without a new execution or an invented T16 count.
- Real terminal operations: a sealed closing record completes, exact hard deadline fails while retaining progress, and activation/reconnect exhaustion fails the same execution.
- Version 11: fresh install, populated 10→11 retaining pins/progress/legacy commands, reservation/result constraints, baseline recognition and disposable 11→10 down pass. Versions 1–10 SQL and their pinned digests are unchanged.
- Startup guard, runner allocation, checksum manifests, Compose settings, verification script, environment examples, API availability/changelog and tripack are synchronized to schema 11.

## Check results and limitations

Required selected checks pass: mocks, domain/application race/coverage, all-package race, contract, architecture, lint (0 issues), security (0 reachable vulnerabilities), disposable PostgreSQL integration/smoke, and scoped version-11 migration proof. `make test.all` without database variables skips resource suites; separate actual PostgreSQL evidence above establishes the database claims. `make test.docker` was not run because the authorized disposable `make test.integration` alternative was used.

An additional whole historical migration-suite run failed on immutable `000010_interview_runtime.down.sql`: it drops transcript entries before referencing coverage/loss tables. This pre-existing version-10 reversal defect is outside T18 and remains unchanged. Version-11 down passes. Returning below version 10 needs a separately reviewed forward repair or a disposable reset; no successful version-10 down is claimed.

Earlier broad integration attempts exposed legacy fixture issues: stale document paths, a helper creating/dropping release tables, and publication fixture time moving backwards. The owner approved path repairs and helper isolation. Test-only fixes preserve their assertions and production sibling behavior. Full integration passes from a clean disposable database; repeated runs over intentionally corrupt retained fixture rows can fail an unrelated configuration-list test.

No live provider, event-finality, automatic provider recreation, durable external rollback, broker delivery, or reporting guarantee is established. The control-tab recheck retains T8's documented limit: a takeover committing after that read is not fenced by the controller store.

## Design assessment

Retained: four root layers, generic lifecycle state ownership, interview domain invariants, immutable revision pinning, operation interfaces and detached results. The shared infrastructure transaction carrier removes independent child transactions without exposing GORM inward. `RecordAttempt` and `FencedWrite` give generic authority separate responsibilities; the infrastructure adapter translates interview budgets and safe errors.

Refined during implementation: reserve/finalize command timing, retained legacy policy, shared candidate/controller reads, bounded setup within both operation and hard deadlines, and sideband cleanup after HTTP drain. The one-connection regression validates the shared-read refinement. No further in-scope refactor was selected; retained interfaces match actual consumers. Broader controller takeover fencing, provider finality and post-handoff exhaustion remain separate decisions/tasks.

## Approved rollback follow-up

The historical version-10 down failure above is resolved by the owner-approved correction recorded in [T18 rollback verification](t18-rollback-verification.md). Full `make test.docker` now passes. The initial checkpoint's unchanged-migration statements describe that earlier delivery; this follow-up explicitly changes only the down-v10 dependency order and its checksum records, leaving all up SQL and schema version 11 unchanged.
