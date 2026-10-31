# Gonesis implementation audit

Scope: current branch `feat/amer-prototype-service`; audit against the supplied gonesis skill, its Task reference and the selected `requirement.txt`. This report does not claim completion or introduce deployment/schema-policy changes. The owning feature design remains [Task Tracking](design/task-tracking.md). Ticket: unassigned.

## Owner-reported items

| Item | Current evidence | Required disposition |
| --- | --- | --- |
| Use-case interfaces and mocks | Handler fields are concrete `taskusecase.Create`, `Start`, `Complete`, `Reschedule`, and `taskquery.Detail`. Mockery only generates repository/clock/queue/reservation mocks. | The skill requires one struct per operation and consumer-needed interfaces, not an interface for every struct. Actual handler consumers justify narrow operation interfaces and generated mocks. These isolate delivery tests. Unit testing an operation itself instead uses the existing dependency mocks. Add interfaces/mocks for all five consumed operations and tests that use them. |
| SQL migrations | Runtime calls GORM AutoMigrate. There is no `migrations/pgsql` tree. | `persistence-events.md` defaults to explicit golang-migrate up/down files, but requirement.txt explicitly selects AutoMigrate and no SQL files. Current absence follows that override. Switching now should replace the schema owner, add up/down files and a migration command, and update the requirement/design/runtime together. Merely adding unused SQL beside AutoMigrate is insufficient. |
| Status enum | `Snapshot.Status` is string; aggregate transition/restoration predicates repeat literals. `domain/enum/taskstate.go` is absent. | Match the reference baseline with a defined state type, named constants and validation at restoration. Map to storage/HTTP strings at boundaries and retain lifecycle enforcement inside Task. |
| OpenAPI | Only prose `docs/api/task.md` exists; no OpenAPI document or API changelog. | Add `docs/api/openapi.yaml`, all six routes, exact payload/envelope schemas, nullability, UUID/time validation, locale header, errors/statuses and examples; validate the contract. The skill requires a specification, endpoint documentation and changelog, although it does not prescribe the exact filename. |
| Deployment | No packaging/deployment artifacts. | Both the skill and requirement.txt exclude packaging/deployment from this initial scope. Adding deployment is a scope amendment, with the target environment still unspecified. Artifact preparation and an actual deployment/publication are separate actions. |

## Further implementation gaps

1. **Domain zero-value enforcement:** `Task.Reschedule` only rejects `done`; a Go zero-value Task can successfully execute this operation. Public mutation boundaries must reject an invalid aggregate. Restoration invalid-state and inconsistent-history cases also lack adequate regression coverage.
2. **Technical error classification:** `taskrepository.toTask` returns `valueobject.ErrInvalidTitle` for malformed stored titles; HTTP maps this to 400. Corrupt persisted data is an internal failure and must be classified/logged as 500 without blaming client input.
3. **Application contracts:** operations return live domain aggregates and Detail merely forwards repository.Find. The baseline application DTO/read result is absent; the handler samples time and maps domain snapshots directly. Define the intended application result and observation-time ownership, then preserve transport/domain separation.
4. **Error catalog/localization:** codes and translation messages are embedded in handler code, rather than `pkg/errs/codes.go` and the authoritative EN/ID TOML files required by `api-errors.md`. Messages also start uppercase. Classifications/cause preservation and catalog completeness need dedicated evidence.
5. **Application regression coverage:** Start, Reschedule and Detail have no unit test files. Create only tests invalid title. Completion only covers success and reported commit failure. Missing-record behavior, queue refusal, invalid/repeated transitions, dependency errors, timestamp ownership and mutation isolation need public application tests.
6. **TDD evidence:** historical RED results often prove that a package/symbol was missing, not that a compiled public behavior violated an expected business rule. This is explicitly insufficient under `testing.md`. Do not retroactively claim full BR-01–BR-09 RED/GREEN evidence; record honest gaps and demonstrate future behavior regressions with executable tests.
7. **Verification resources:** the only composition smoke covers health and shutdown and skips without TEST_DATABASE_URL. Real API business flow, PostgreSQL restart persistence, concurrent row-lock transitions, atomic rejection/publication, startup failure during acquisition, active-work drain and forced shutdown are not established. Watchdog subprocess tests prove process-exit policy, not successful graceful resource cleanup.
8. **Tooling:** no Makefile, checked-in golangci-lint configuration or repeatable architecture dependency check. Existing command-line successes do not supply those artifacts or prove rich-domain invariants.
9. **Business documentation and assessment:** no service business-rule index/capability documents with allocated BR IDs, rule/test/scenario mapping, or complete feature assessment/dispositions. The design record has accumulated historical notes without a consolidated final mapping.
10. **Test conventions:** anonymous test-case structs and manual assertions remain in several domain/runtime/delivery tests; testing.md specifies named test-case types and testify. Fixture reuse is absent.
11. **Logging/subscriber contract:** header suppression is configured, but the full required masking policy is not explicitly configured/verified. The consumer callback cannot return an ordinary subscriber error, so the selected log-and-continue failure behavior is not demonstrated; panic supervision does not establish that contract.

## Evidence from this audit

Read-only code audit plus `GOCACHE=/private/tmp/amer-go-cache GOPATH=/private/tmp/amer-go go test ./domain/... ./application/... -cover` passed. Statement coverage: aggregate 87.5%, title 83.3%, task use cases 60%, Detail query 0%. Coverage percentages are not acceptance evidence and do not show all business scenarios. No real PostgreSQL test or deployment was performed.

## Repair order

1. Fix invalid-aggregate and persisted-data error classification with public regression tests.
2. Define enum, application results and consumer interfaces; generate and use operation mocks; complete public domain/application rule tests.
3. Complete OpenAPI, error catalog/translations, business docs, architecture checks and runnable tooling.
4. Complete coordinator failure-injection tests and real PostgreSQL/API evidence when the authorized instance is supplied.
5. If the schema policy is amended, replace AutoMigrate with reviewed migration up/down artifacts and verify them. If deployment is added, select its target before preparing and validating platform-specific artifacts.

## Implementation follow-up

The findings above describe the audit baseline, not the current repository state. Operation interfaces/generated mocks, application DTOs, state enum, zero-value guards, technical error wrapping, UTC restoration, OpenAPI, authoritative TOML catalogs, business rules, Makefile/lint configuration, architecture checks, normal subscriber failure handling and explicit masking configuration have now been implemented. Expanded public application tests cover all operations and major rejection/failure paths.

Real PostgreSQL verification now runs through the owner-authorized isolated Docker setup. API flow, restart persistence, queue refusal rollback, deferred commit failure without publication, concurrent transitions, partial startup failure, cancellation during migration, active-query cancellation, graceful event drain and timeout failure passed. The runtime tracks pgx CleanupDone channels in addition to closing the SQL pool. Dockerfile builds, and Compose supports PostgreSQL plus optional Redis/RabbitMQ profiles. No remote deployment is involved.

AutoMigrate remains selected, so SQL migration up/down artifacts remain a pending policy choice, rather than an implemented migration path. Genuine historical RED/GREEN evidence for every business rule cannot be reconstructed; that completion limitation is explicitly retained. The owning [feature design](design/task-tracking.md) is the current source for decisions, scenario mapping, assessment and verification results.

The final full-stack run passed after adding unexpected HTTP/consumer-exit scenarios. The owner's documentation amendment is implemented across authored Go declarations and interface methods and guarded by `TestDeclarationDocumentation`; Python orchestration helpers also have docstrings. Generated mocks retain their generator comments. Schema selection remains AutoMigrate pending an explicit switch.
