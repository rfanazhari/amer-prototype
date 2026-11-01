# Application startup logging follow-up

Ticket: CORE-IAM-018, requested after the T18–T20 delivery. RabbitMQ remains an optional Compose service; the app does not connect to it. Task completion continues through the existing in-process queue.

## Design and assessment

Lifecycle observability belongs to `infrastructure/runtime`; no domain, application, HTTP, migration or queue contract changes. Keep logmanager v1.44.1 and existing logging initialization. Its local `Start`, `ToContext`, `InfoWithContext`, `End`, `NoticeError` and test-hook APIs were inspected and exercised. One startup transaction correlates stages and records elapsed time. Readiness is emitted only after database/schema validation, bootstrap and existing listener/worker setup. Failed startup emits a generic error without adding configuration or raw driver errors to lifecycle records.

Assessment: retained existing composition and resource ownership. No additional port, interface mock, dependency or background worker is needed. Real PostgreSQL composition tests capture logmanager output through its supplied test hook, check successful/failed startup ordering and prevent URL/password/CSRF/email leakage. SQL is exercised against disposable PostgreSQL rather than represented by handwritten mocks.

## Checkpoints

RED cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

- `go test ./test/integration -run TestStartupLogsReportReadinessWithoutSecrets -count=1` compiles and skips when no authorized database is configured; it is not behavioral RED or database proof.
- `make test.docker` observed RED: both successful startup and occupied-listener failure produced no startup messages (`actual: []string{}`). The exact staged snapshot was also verified in `/var/folders/v0/yg69hwhd21lfjjbn5bmfss6h0000gn/T/startup-red-ia0ftbb4` using the same command.
- RED SHA: `dfde6d3f646414ad07e4907cbed7349a9d688685`.
- Actual subject: `test: [CORE-IAM-018][rfanazhari] RED cover safe application startup logs`.
- No non-test scaffolding was required.

GREEN commands, from the repository root or its exact staged snapshot:

```sh
go test ./infrastructure/runtime ./test/integration -run 'TestStartupValidationAndCancellation|TestStartupLogsReportReadinessWithoutSecrets' -count=1
make check
make test.docker
```

Results: targeted runtime checks, all-package race checks, contract, architecture, lint and security passed. Security found zero reachable vulnerabilities. Resource tests skipped without explicit database variables in ordinary tests; `make test.docker` supplied isolated databases and passed the actual startup log assertions, Task/admin regressions and migration/recovery suites. Normal development databases/volumes were untouched. No interface changed, so no Mockery regeneration was needed.

The GREEN SHA and validated actual commit subject are returned in the delivery response. This follow-up updates README's log usage and does not reopen the completed migration checklist or accept external broker delivery.
