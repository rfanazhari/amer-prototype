# T11 — real report broker verification

Ticket: `CORE-IRP-011`. Executed 2026-11-03 from the repository root.

## Scope and design mapping

Design §5 and §7 and decision D7 remain the authority. The existing domain job owns
attempt budgets; the PostgreSQL repository owns claims, fencing and outcomes. The
existing application worker orchestrates evidence and writing. `reportbroker.Consumer`
only decodes a job identity, calls the worker and acknowledges successful processing.
Errors leave the delivery unacknowledged; the caller stops and closes its channel
before recovery. It never bulk-acks earlier messages.

The consumer is not registered in runtime. Production topology, malformed-message
isolation, production evidence/writer adapters and runtime registration remain T12–T15.
No migration, HTTP contract, interview lifecycle or provider integration changes.

## Reproduce safely

Requires Go from `go.mod`, Python 3, Docker Compose, and the Compose images. Run:

```sh
make test.report-broker
```

This runs `python3 scripts/verify_report_broker.py`. It creates a unique Compose
project using the existing `rabbitmq` profile, starts only PostgreSQL and RabbitMQ,
uses ephemeral loopback ports, and applies the existing migrations through version 13.
It executes:

```sh
go test -race ./test/integration/... -run '^TestInterviewReportRabbitMQ' -count=1 -timeout=150s -v
```

The outage test signals a private temporary directory after connecting. The runner
stops only its own RabbitMQ container; the test observes connection closure before
committing admission and attempting publication. Finally, the runner removes only
its own containers, network and volumes. Normal development volumes are preserved.

`TEST_RABBITMQ_URL` is an AMQP URL supplied only through the environment. The runner
constructs it in memory using a generated URL-safe password and discovered port;
it never passes the URL as a command argument or prints it. RabbitMQ dial failures
emit a fixed message without the raw error or URL. `TEST_DATABASE_URL` and
`MIGRATION_DATABASE_URL` likewise point only to the runner's disposable PostgreSQL.
Inherited URLs are overridden. Do not use shell tracing, print the environment,
or print the rendered Compose configuration. The outage-control directory contains
only an empty readiness marker, no credentials.

Direct Go runs skip broker tests without `TEST_RABBITMQ_URL`; the outage test also
requires the runner. Skipped tests do not establish RabbitMQ behavior. These tests
mutate the database and must run only against an authorized integration database.

## Observed proof

All six cases passed with the race detector against real RabbitMQ and PostgreSQL:

- Confirmed persistent JSON contains exactly `jobKind`, `jobId`, `messageId`;
  `confirmed_at` is persisted, the job is published, and generation stays pending.
- Closing a consumer before ack redelivers the same body and message identity with
  `Redelivered=true`.
- A crash after the first claim followed by redelivery is acknowledged without
  another attempt or fencing-token change; the writer is not called.
- Processing and then republishing the same terminal job invokes the writer once
  and leaves exactly one attempt.
- Closing an independent real SQL pool causes the database read to fail. The
  consumer propagates that error without ack, and channel closure redelivers it.
- With the actual Compose broker stopped, admission still commits completion, job
  and outbox. Reconciliation errors and the outbox stays unconfirmed with no attempt.

No sqlmock or broker mocks were used for these assertions. Evidence, writer and
clock use existing Mockery-generated mocks; these proofs do not establish production
provider behavior or finalized-evidence integration. The test queue is a unique
durable queue on the default exchange, with no dead-letter arguments or topology.

## RED and GREEN

- RED command: `python3 scripts/verify_report_broker.py`.
- RED reason: the inert consumer neither invoked the worker nor acknowledged
  successful duplicates, and returned nil instead of the database error. Confirm,
  stop-before-ack redelivery and broker outage already passed.
- RED checkpoint: `957263b67db8d4a62fd0abe6d46e28a710aea972` —
  `test: [CORE-IRP-011][rfanazhari] RED cover real report broker delivery and failure semantics`.
- RED scaffolding: consumer type, executor interface and inert `Handle` only, plus
  the test fixture and isolated-resource runner.
- GREEN commands: the same Python command, then `make test.report-broker` on the
  final Go implementation. Both passed all six real-resource cases.
- Regression: `make test` and `make check` passed. The latter covers all-package race tests,
  contract, architecture, lint and security. Resource suites skipped there because
  their environment variables were absent; the dedicated runner supplies T11 proof.
- Security: `make check` ran `govulncheck ./...` through `test.security`: zero
  reachable vulnerabilities. It also reported one imported-package and one
  required-module vulnerability whose affected symbols are not called.
- An initial lint run found deprecated `QueueInspect` in the test helper; it was
  replaced with `QueueDeclarePassive` and final lint passed with zero issues.

## Design assessment

Retained: the small consumer owns only transport parsing and per-delivery ack;
attempt deduplication stays in the existing domain/store. The one-method executor
boundary accepts the existing worker without coupling the adapter to its concrete
dependencies. No further refactor is justified. The six broker/database proofs and
existing domain/application regressions cover this boundary. No design-contract
revision is needed for T11; T12 must settle malformed-message isolation before any
production loop automatically recovers those failures.
