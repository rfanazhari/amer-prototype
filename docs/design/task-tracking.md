# Task Tracking design and verification

Ticket: unassigned; no authoritative issue ID was supplied. Requirements: [requirement.txt](../../requirement.txt), amended by the owner to keep work in the current checkout and include Dockerfile, Compose, PostgreSQL, optional Redis/RabbitMQ and Makefile. Branch: `feat/amer-prototype-service`. No commit, MR or remote publication is requested.

## Scope and accepted selections

Task tracking only; interview, voice and transcript features remain outside scope. HTTP, PostgreSQL/GORM and an internal queue are required. Manual versioned SQL migrations supersede the original AutoMigrate selection under identity-access D8. T16–T17 capture baseline and tooling; T18 replaces transitional startup/test AutoMigrate. Redis and RabbitMQ are optional local Compose profiles; Task does not connect to them. No runtime memory fallback, authentication, RPC or bootstrap is selected.

The [OpenAPI specification](../api/openapi.yaml), [API changelog](../api/CHANGELOG.md) and [business rules](../services/amer-prototype/business/README.md) own transport and business details. English/Indonesian errors use authoritative embedded TOML catalogs under `translations/`; stable codes live in `pkg/errs`.

## Tactical design

| Concept | Classification, location and ownership | Enforcement |
| --- | --- | --- |
| Task | Aggregate root/entity in domain/aggregate | Private state; creation, valid transitions, Done finality, optional dates, timestamps and overdue policy. Public mutators reject an invalid zero-value aggregate. |
| Title | Immutable value object in domain/valueobject, owned by Task | Trim Unicode whitespace, validate 1–100 code points, reject invalid UTF-8 and zero values at aggregate boundaries. |
| TaskState | Defined enum in domain/enum, owned by Task | Todo, InProgress and Done; unknown restored states are rejected. |
| Due date | Optional copied time value owned by Task | Validate creation/reschedule boundaries; equality allowed; clearing allowed for non-Done; snapshots cannot mutate owned pointers. |
| Creation/completion times | System facts owned by Task | Application supplies trusted time; restoration preserves facts; reads do not mutate history. |
| TaskCompleted | Domain event in domain/event | Successful Complete returns the fact; application coordinates acceptance and persistence. |

Repository restoration validates stored values/state/history without reapplying creation-time due-date rules. Persistence mappings normalize driver-returned instants to UTC. Invalid stored data is wrapped in a technical error; HTTP maps that to 500 even if its underlying cause is a title validation error.

## Technical design

Application operations are plain structs, one per operation. CreateOperation, StartOperation, CompleteOperation, RescheduleOperation and DetailOperation are the contracts used by delivery. Mockery generates all five mocks plus repository/clock/event ports. Each operation returns an application DTO with copied times; Detail owns observation time and invokes domain overdue behavior. Delivery parses/formats transport values without owning lifecycle rules. Operation tests mock their dependencies, while delivery tests mock operations.

The repository port is application-owned because its Update callback expresses the transaction needed by command consumers. The GORM adapter maps separate storage models and holds `SELECT ... FOR UPDATE` through callback execution and commit. Its explicit update field selection preserves nullable due/completion fields. Rejected callbacks roll back. Data survives runtime restarts. Schema fields are UUID primary key, non-null title/status/created_at/updated_at, nullable due_date/completed_at, and a status check. No soft deletion is enabled.

Completion reserves internal queue capacity in the transaction callback after domain validation. Queue refusal aborts the transaction. A reported commit failure cancels the reservation. Only a successful commit enqueues the event. A single consumer processes a queue of capacity 100. Ordinary subscriber errors are logged and processing continues without retry; a panic is an unexpected worker failure. Unexpected process termination can lose events. No durable recovery or exactly-once guarantee is claimed.

Time is injected through an application port. Trusted operation/observation instants are UTC at millisecond precision; HTTP rejects input with excess precision and normalizes valid offsets. JSON output always includes three fractional digits and UTC. Creation/completion times cannot be supplied as request fields.

## Runtime ownership and lifecycle

Resource setup, feature assembly, HTTP setup, shutdown policy and coordinator have separate files under `infrastructure/runtime`. `cmd/app` loads configuration, initializes masked JSON logging, handles signals and enforces terminal process policy. A ten-second startup context covers connectivity, AutoMigrate and listener binding. Every acquired pool has cleanup on later failure/cancellation. The feature graph has no optional dependencies or fallback branches.

The database pool uses pgx stdlib's AfterConnect hook to track physical connection cleanup channels. `databaseResources.close` closes the SQL pool and waits for driver CleanupDone; this matters because pgx can close cancelled-query connections asynchronously. A cleanup deadline is reported as failure; supported socket closure is attempted without claiming that driver goroutines have terminated. The executable watchdog provides the final process limit.

Bound HTTP listeners and a running consumer precede readiness notification. The same Run composition is used by the executable, real API tests and smoke. Listen and TaskCompleted are narrow lifecycle/subscriber seams for deterministic tests; they are not alternative production storage adapters.

The coordinator supervises both server and consumer exits. A failure stops admission and cleans up the runtime; errors are joined with relevant cleanup failures. HTTP request contexts have a lifetime independent of the signal so graceful work can drain. Admission is gated before waiting for active handlers. Once draining times out, request contexts are cancelled and HTTP is forcibly closed. Accepted event work is drained while dependencies remain available; timeout cancels the consumer and is retained as failure.

All shutdown steps share five seconds. Graceful drain receives a four-second subdeadline; remaining time is for forced cancellation/termination and database cleanup. DB.Close itself has no context and may block, so cmd/app's watchdog terminates with exit code 1 at the process deadline. Forced exit is not successful graceful cleanup. Timeout logging is best effort so logging cannot extend the deadline. Run by itself does not forcibly terminate library callers.

## Rule and scenario evidence

| Source / service rule | Public tests and scenarios |
| --- | --- |
| BR-01 / BR-AMER-001 | TestTitle; TestCreate; zero-title restoration/creation rejection. |
| BR-02 / BR-AMER-002 | TestCreate; TestTaskDueDateBoundaries; TestTaskAPIFlow. |
| BR-03 / BR-AMER-003 | TestTaskDueDateBoundaries; TestReschedule; API replacement/clear flow. |
| BR-04 / BR-AMER-004 | TestTaskLifecycle; TestStart; TestConcurrentTransitionsAreSerialized. |
| BR-05 / BR-AMER-005 | TestTaskLifecycle; TestCompleteRejectsBeforePublication; API rejects Todo completion. |
| BR-06 / BR-AMER-006 | TestTaskLifecycle; TestReschedule; TestTaskAPIFlow attempts mutations of Done. |
| BR-07 / BR-AMER-007 | TestDetail exact boundary/advanced clock/Done/no date; TestTaskBoundariesAndRestoration. |
| BR-08 / BR-AMER-008 | Creation/completion facts, rejected-mutation snapshot checks, forbidden-client-field tests. |
| BR-09 / BR-AMER-009 | Application completion tests; queue-refusal DB rollback; real deferred commit failure; no repeated event; accepted-event drain. |

Missing records and dependency errors are covered at application/HTTP boundaries. Restoration includes unknown states, zero title and inconsistent timestamps. Mutable input, snapshot and DTO times are copied. Domain tests invoke public behavior without HTTP/storage. Architecture checks enforce import directions but are not substituted for invariant tests.

## Assessment and dispositions

| Finding | Disposition and evidence |
| --- | --- |
| Zero-value Task could reschedule | Fixed. Compiled public regression first observed nil error, then passed after invariant guard. |
| Primitive status literals | Refactored to enum with restoration validation; lifecycle/restoration tests retained. |
| Concrete handler operation fields | Refactored to consumer interfaces; generated mocks used for all operations. |
| Domain aggregate returned directly | Refactored to application DTO; Detail observation policy tested with controlled time. |
| Corrupt stored title returned client error | Fixed technical wrapping and mapping precedence; real corrupted-row and handler tests. |
| Driver timezone differed from UTC | Real DB test exposed mismatch; adapter normalization fixed it and database suite passed. |
| Subscriber could not return normal failure | Added contextual subscriber/error callback; log-and-continue/no-retry and forced-cancel tests. |
| DB.Close did not imply asynchronous driver cleanup | Physical cleanup tracking added; cancelled migration and active database request tests inspect closed sessions. |
| Missing contracts/catalog/tooling/business docs | OpenAPI validated; embedded EN/ID catalog tests; Make targets; lint config; architecture checks; allocated rules. |
| No SQL migration files | Historical selection preserved; D8 now approves manual SQL migrations via T16–T20. |
| Deployment previously excluded | Local Docker assets added under owner amendment; optional services stay outside Task's runtime graph. |

## Verification commands and observed results

Run from the repository root. Sandbox runs set GOCACHE/GOPATH/GOLANGCI_LINT_CACHE to writable temporary directories; these are environment constraints, not repository-specific command requirements.

- `make test.all`: all-package race check passed. Without TEST_DATABASE_URL the resource suites skip; that does not establish real-resource behavior.
- `make test`: public domain/application tests passed. Aggregate, use-case and query package statements reached 100% on the observed run; this is not substituted for scenario evidence.
- `make test.contract` and `make test.architecture`: passed.
- `make lint`: golangci-lint v2 passed with zero issues.
- `make test.security`: passed with zero reachable vulnerabilities; scanner also reported one imported-package and two required-module findings not found reachable.
- `make test.docker`: passed on isolated PostgreSQL 17. Covers real health/API flow, restart persistence, graceful event drain, blocked-subscriber timeout, concurrent transitions, queue refusal rollback, corrupt stored title, deferred commit failure, partial startup failure, migration cancellation and forced active-query cancellation.
- `docker build -t amer-prototype:verify .`: passed.
- `make docker.verify`: rebuilds the image and checks app HTTP health plus PostgreSQL, optional Redis and RabbitMQ readiness; final result is recorded in the audit follow-up.

Local verification generates credentials in memory and uses isolated Compose project names/volumes. Cleanup removes only that project's resources. Optional Redis/RabbitMQ have readiness evidence, not Task integration claims. No remote publication or deployment was performed.

## Evidence limitations

Historical implementation did not preserve genuine behavior-level RED/GREEN evidence for every BR. Missing-package/symbol failures are not accepted as substitutes and are not retroactively relabeled. Current regression/scenario coverage is recorded above; the historical TDD completion criterion remains an evidence gap. Resource timing evidence applies to this local PostgreSQL/driver environment, not every possible network failure. Process-deadline tests prove forced process termination, not event delivery or graceful cleanup after deadline exhaustion.

Final local stack verification passed: `make docker.verify` rebuilt the image and confirmed container HTTP health, PostgreSQL, Redis and RabbitMQ readiness, then passed seven real composition/API scenarios and seven PostgreSQL repository/lifecycle scenarios, including unexpected HTTP and consumer exits. Its isolated containers and volumes were removed. The owner subsequently required per-declaration documentation; authored types, functions, methods, interface methods and test helpers now have responsibility/behavior comments. `TestDeclarationDocumentation` checks this with the Go AST; generated mocks retain generator documentation. Architecture/documentation checks and local Markdown links passed after these documentation changes.
