# Task Plan: Interview Reporting

Planning baseline: 2026-11-02, Mode A. No report implementation exists. Every task starts Todo. BR-CORE-080..086 are not enforced by this document.

## Ticket Identity

Capability group: **IRP — Interview Reporting**. Ticket prefix: `CORE-IRP`. These are repository planning IDs. No external tracker tickets have been created.

| Local ID | Ticket |
|---|---|
| T1 | `CORE-IRP-001` |
| T2 | `CORE-IRP-002` |
| T3 | `CORE-IRP-003` |
| T4 | `CORE-IRP-004` |
| T5 | `CORE-IRP-005` |
| T6 | `CORE-IRP-006` |
| T7 | `CORE-IRP-007` |
| T8 | `CORE-IRP-008` |
| T9 | `CORE-IRP-009` |
| T10 | `CORE-IRP-010` |
| T11 | `CORE-IRP-011` |
| T12 | `CORE-IRP-012` |
| T13 | `CORE-IRP-013` |
| T14 | `CORE-IRP-014` |
| T15 | `CORE-IRP-015` |

Keep both identities on any later revision. Add new numbers. Do not reuse these.

## Legend

- IDs T1–T15 are stable.
- Complexity: S (<2h), M (2–4h), L (>4h). Engineers still validate the estimate.
- Status: `[ ]` Todo, `[~]` In Progress, `[x]` Done, `~~obsolete~~`.
- `ref` points at [technical-design.md](technical-design.md).
- A blocked task is not executable until its ready condition is true and this tripack records the decision.
- Companion authority: [Identity and Access](../identity-access/technical-design.md) `accessauthority.Authority` and BR-CORE-022. This plan does not copy that plan's task statuses.

## Tasks

### Sequential foundation

- [ ] **T1. Encode the report lifecycle and statement rules** — Domain — M
  - Ticket: `CORE-IRP-001` — depends: none — ref: design §3–§4, §6 path contracts.
  - DoD: `domain/aggregate/interviewreport` owns the report root, generation entity, statement value, and the budget constants (2, 60 seconds, 5 seconds, 5 minutes). Public tests cover an active-generation refusal, a new generation after failure, citation rejection, gap shape, verdict word boundaries, and restoration that does not call a provider. The type has no interview-status mutation. `make test` and `make test.architecture` pass. Declarations are documented. No SQL and no HTTP.

- [ ] **T2. Add report ports and the HR predicate** — Application — M
  - Ticket: `CORE-IRP-002` — depends: T1 — ref: design §2, §4.
  - DoD: Operation interfaces for admit, regenerate, and read take injected ports and return detached DTOs. `InTransaction` hides the database handle. `AllowsInterviewReport` is HR-only. `InterviewReportAuthority` is a separate interface from `AgentConfigurationAuthority`. Evidence and writer ports are interfaces with no production adapter. `make mocks` regenerates the new interfaces. `make test` and `make test.architecture` pass.

### Schema, store, and authority

- [ ] **T3. Add manual migration 000011 and raise the release floor** — Schema — L
  - Ticket: `CORE-IRP-003` — depends: none — ref: design §3.
  - DoD: `migrations/pgsql/000011_interview_reporting.up.sql` and `.down.sql` create and drop only the seven tables, checks, foreign keys, and indexes in design §3. `schema.ReleaseVersion` is 11. Both new files are in `checksums.json`. `ownedPublicTables` and `RuntimeGrantTables` include the new tables. `docs/design/database-migrations.md` says clean version 11. Versions 1–10 are unchanged. No seed and no AutoMigrate. A disposable PostgreSQL database migrates from clean 10 to clean 11, then down and up, and proves the constraints, including one active generation and an answer without evidence. `make test.architecture` passes. If `TEST_DATABASE_URL` is unset, say the database proof was skipped and run it with `make test.docker` or an authorized URL before calling T3 done.

- [ ] **T4. Map report rows and the two locked claims** — Infrastructure — L
  - Ticket: `CORE-IRP-004` — depends: T1, T2, T3 — ref: design §3 transactions.
  - DoD: GORM models live only in `interviewreportrepository`. Explicit `Select` writes persist a zero attempt count and a null failure reason. Raw SQL is only the job claim and the outbox claim, each with `FOR UPDATE SKIP LOCKED`, and the design's reason is still accurate. sqlmock may lock the statement text. `make test`, `make lint`, and `make test.architecture` pass.

- [ ] **T5. Prove admission atomicity and report failure isolation on PostgreSQL** — Infrastructure — L
  - Ticket: `CORE-IRP-005` — depends: T4 — ref: design §3 proof plan, §5 completion flow.
  - DoD: One real PostgreSQL transaction uses a test stand-in to set `interview_assignments.completed_at` and then `Admit`. Commit leaves generation, job, and outbox. A forced outbox failure rolls back `completed_at` and the report rows. A later generation-failure transaction leaves `interview_assignments` and `interview_executions` unchanged, compared column for column. `Admit` does not itself update those tables, and it errors when `completed_at` is null. These tests are not sqlmock. `make test.integration` passes against an authorized database, or the skip is reported and `make test.docker` is the run that counts. RabbitMQ is not part of this task.

- [ ] **T6. Authorize reports on the existing authority adapter** — Infrastructure / application — M
  - Ticket: `CORE-IRP-006` — depends: T2 — ref: design §4 authority decision, BR-CORE-022.
  - DoD: `accessauthority.Authority` implements `InterviewReportAuthority` using the same session resolver and clock. Report `Authorize` calls `AllowsInterviewReport` and the report operation names. Configuration `Authorize` is unchanged. Tests: current HR allowed, ADMIN forbidden, expired, revoked, disabled, and unbound sessions denied, candidate session does not become an admin principal. No new authorizer type. `make test` and `make test.architecture` pass.

### HR contract

- [ ] **T7. Implement read and explicit regeneration** — Application — L
  - Ticket: `CORE-IRP-007` — depends: T1, T4, T6 — ref: design §5–§6.
  - DoD: GET returns the previous succeeded account, or null, plus the newest generation. POST replay returns the original generation. A new key while a generation is active inserts nothing. A new key after failure inserts the next generation, job, and outbox without changing `completed_at`. Evidence denial uses the port and inserts nothing. Idempotency fingerprint matches design §6. Tests use a fake evidence port. `make test` passes.

- [ ] **T8. Map the two HR routes and publish the contract** — Delivery / API — L
  - Ticket: `CORE-IRP-008` — depends: T7 — ref: design §2, §6–§7.
  - DoD: Handler tests with mocked operations cover the error precedence, CSRF on POST, cookie and Origin, 201 versus 200, and a body with no verdict or salary field. ADMIN is forbidden before the operation runs. `docs/api/openapi.yaml`, `docs/api/interview-reporting.md`, and `docs/api/CHANGELOG.md` change together. `pkg/errs` and both translation files contain the `IRP_*` messages from design §6. Candidate and voice paths are unchanged. Routes are not mounted in `infrastructure/runtime`. `make test`, `make test.contract`, `make test.architecture`, and `make lint` pass.

### Durable dispatch

- [ ] **T9. Publish confirmed outbox messages without a dead-letter topology** — Infrastructure — L
  - Ticket: `CORE-IRP-009` — depends: T4 — ref: design §5 publisher and reconciler.
  - DoD: `reportbroker` publishes persistent messages that contain only `jobKind`, `jobId`, and `messageId`. Publisher confirms set `confirmed_at`. A lost confirm can leave a duplicate and does not mark the generation succeeded. The reconciler republishes unconfirmed rows. No dead-letter exchange, queue, or routing argument is declared. Dependency `github.com/rabbitmq/amqp091-go` is pinned in `go.mod`. `make test.security` is run for that module. Unit tests fake the broker. `make test` and `make lint` pass.

- [ ] **T10. Claim attempts inside the accepted budget** — Application / infrastructure — L
  - Ticket: `CORE-IRP-010` — depends: T1, T4 — ref: design §5 worker, §6 attempt path.
  - DoD: Controlled clock proves a 60-second attempt cap, a 5-second retry delay, no third attempt, and expiry at 5 minutes from job creation including time spent queued. The writer port is a fake. Accepted output commits statements and the generation outcome without updating interview tables. Rejected output stores no statements. An old fencing token does not commit. `make test` passes. The consumer is not registered in the process.

- [ ] **T11. Prove RabbitMQ behavior apart from sqlmock** — Infrastructure — L
  - Ticket: `CORE-IRP-011` — depends: T9, T10 — ref: design §5, §7.
  - DoD: Against the Compose `rabbitmq` profile, tests show a confirm, a redelivery after the consumer stops before ack, a duplicate that does not create a second attempt, a broker outage that still leaves the committed outbox row unpublished, and a database error that is not acknowledged. Messages are well-formed. The test report states that sqlmock was not used for these assertions. `make test.security` still passes with the broker module. Document the broker URL handling so the password is not printed.

### Blocked / deferred

- [ ] **T12. Isolate malformed broker messages** — Infrastructure — M
  - Ticket: `CORE-IRP-012` — depends: T11 — ref: design §7, §9 Q1.
  - Blocked by: dead-letter topology, platform technical design §6, owner is that platform document.
  - Ready when: Q1 is decided and design §5 and §7 name the approved exchange, queue, and malformed-body action. Do not invent those names in this task.
  - DoD: A malformed body is isolated using that approved topology, is not acknowledged as success, and is not requeued forever. Well-formed job handling from T11 still passes. Interview rows do not change.

- [ ] **T13. Add the report writer client** — Infrastructure — L
  - Ticket: `CORE-IRP-013` — depends: T10 — ref: design §4, §9 Q2.
  - Blocked by: report model selection and invocation schema, platform technical design §14.
  - Ready when: Q2 is written into design §6, including the provider request that excludes the salary ceiling and the response that maps to observations, gaps, and uncertainties.
  - DoD: The client implements the writer port, calls no voice API, and returns only the structured account. Provider failure becomes an attempt failure. Two attempts are not exceeded. Logs omit the provider body and any ceiling material. `make test.security` passes for the new module.

- [ ] **T14. Read finalized evidence from Interview Runtime** — Infrastructure — L
  - Ticket: `CORE-IRP-014` — depends: T2 — ref: design §3, §9 Q3.
  - Blocked by: Interview Runtime evidence storage. No transcript table exists, and that capability's provider event mapping is Open.
  - Ready when: Q3's read model exists and this design records the entry id, revision, and execution id it will call. This task still does not design consent, voice, or stages.
  - DoD: The adapter returns finalized entries for one execution and reports none when the sibling has no finalized record. It does not read partial transcript fragments. Regeneration's evidence denial uses this adapter in a PostgreSQL test. The report migration does not gain a transcript table.

- [ ] **T15. Register report routes, admission, publisher, and consumer** — Runtime — M
  - Ticket: `CORE-IRP-015` — depends: T5, T8, T11, T13, T14 — ref: design §4 deferred consumer, §5 shutdown.
  - Blocked by: Q2 and Q3. T13 and T14 must be Done first. T12 is not required for this registration.
  - Ready when: the writer and the evidence adapter are the production implementations, and design §5 says the consumer may start.
  - DoD: `infrastructure/runtime` mounts the two routes before the `/admin/` catch-all, calls `Admit` only from the lifecycle completion transaction once that caller exists, and starts the publisher, reconciler, and consumer on the runtime context. Shutdown still finishes inside 5 seconds and leaves unconfirmed outbox rows for restart. A missing writer fails startup rather than burning attempts. `make test`, `make test.contract`, `make test.architecture`, and `make lint` pass. Candidate routes still omit the report.

## Dependency Graph

```
T1 → T2 → T6
T1 → T4
T2 → T4
T3 → T4 → T5
T4 → T7
T6 → T7 → T8
T4 → T9 → T11
T1 → T10
T4 → T10 → T11
T11 → T12 (blocked on Q1)
T10 → T13 (blocked on Q2)
T2 → T14 (blocked on Q3)
T5 → T15
T8 → T15
T11 → T15 (blocked until T13 and T14 are Done)
T13 → T15
T14 → T15
```

T3 can proceed beside T1. T6 can proceed beside T3 and T4. T9 can proceed beside T7. T12 does not block T15.

## Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial plan, T1–T15 Todo | Mode A |
