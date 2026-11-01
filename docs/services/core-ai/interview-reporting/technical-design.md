# Technical Design: Interview Reporting

Date: 2026-11-02. Mode A — new feature. No interview-report implementation was found. Rules BR-CORE-080 through BR-CORE-086 are `planned` in the capability document. This tripack does not claim they are enforced.

Business authority is [Interview Reporting](../business/capabilities/interview-reporting.md). The platform [technical design](../../../design/core-ai/technical-design.md) sections 6, 7, and 10 win where they refine the [architecture design](../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md) section 10 completion-and-reporting narrative. Scope freeze [H-10, H-11, and H-12](../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) remain the acceptance themes. This feature tripack does not replace those platform documents and does not design consent, voice, stage workflow, revocation, artifacts, or document executions.

## 1. Overview

After Interview Runtime has completed an interview execution under Execution Lifecycle BR-CORE-056, the same database transaction records one report generation, one report job, and one outbox row. A same-process worker later writes the HR account. The candidate's final screen does not wait (sibling BR-CORE-047). HR reads and explicitly regenerates the account through the service API. ADMIN has no report authority (BR-CORE-022). There is no admin or HR user interface in this phase.

### Goals

- Give the completed interview a separate report lifecycle whose failure cannot change the interview outcome.
- Persist observations, coverage gaps, and uncertainties that cite finalized evidence, and refuse a hire, reject, or rank verdict.
- Admit that work durably with PostgreSQL, then dispatch it with RabbitMQ and an outbox. The live voice loop does not use this broker.
- Expose `GET /admin/interview-assignments/{id}/report` and `POST /admin/interview-assignments/{id}/report-generations` to current HR.

### Non-Goals

- Deciding when an interview is completed, failed, or cancelled (BR-CORE-056 and BR-CORE-070 stay with Execution Lifecycle and Interview Runtime).
- Consent, voice sessions, stage transitions, checkpoints, or transcript capture.
- Assignment revoke, artifact upload, and document-execution routes.
- An admin or HR screen. The candidate final screen and its payload.
- A dead-letter topology. Platform technical design §6 leaves that Open; this design does not name a dead-letter exchange, queue, or routing key.
- A report-model invocation schema. Platform technical design §14 still lists document and report model selection as unfinished.
- Using the Task in-process completion queue, or connecting Redis, for report work.

### Verified context and authority

| Source inspected | Finding |
|---|---|
| Root `AGENTS.md` | Four root layers. Manual SQL migrations, no AutoMigrate. GORM for CRUD. Trusted `application/service.Clock`. HTTP contract updates touch `docs/api/openapi.yaml`, an endpoint document, and `docs/api/CHANGELOG.md` together. Declaration documentation is enforced. |
| `go.mod` | Module `github.com/rfanazhari/amer-prototype`. Go 1.26.10. GORM 1.25.12 and PostgreSQL driver 1.5.11. `github.com/golang-migrate/migrate/v4` v4.19.0 is already the migration tool. No RabbitMQ client module. |
| `infrastructure/database/schema/version.go` | `ReleaseVersion` is 7. `ownedPublicTables` and `RuntimeGrantTables` end at `interview_executions`. |
| `migrations/pgsql/` and `infrastructure/database/schema/checksums.json` | Highest applied pair on disk is `000007_interview_executions`. This slice does not take `000008`. Its pair is `000011_interview_reporting`, after execution-lifecycle `000008`, candidate-interview `000009`, and interview-runtime `000010`. |
| `infrastructure/runtime/`, `**/*.go` search for `outbox`, `amqp`, and `rabbitmq` | No outbox table, repository, or broker client. Compose has an optional `rabbitmq` profile (`README.md`, `compose.yaml`). The process does not connect to it. Task completion stays on the in-process queue. |
| `migrations/pgsql/*.sql` and Go identifiers for report generations | No report table and no report use case. The only report document is the capability file. |
| `migrations/pgsql/000007_interview_executions.up.sql` | Execution pin only: `id`, `assignment_id`, `instance_id`, `published_revision_id`, `created_at`. No terminal status and no transcript. |
| `migrations/pgsql/000005_interview_assignments.up.sql` | Interview-owned `completed_at timestamptz NULL`. No Go writer sets `interview_assignments.completed_at`. Access only reads the column. |
| `application/usecase/agentbindingusecase/start_execution.go` | Copies the assignment pin. It does not complete an interview or insert report work. |
| `infrastructure/service/accessauthority/authority.go`, `application/service/policy.go` | One `Authority` struct. `Authorize` allows current HR for known agent-configuration operations and returns `service.ErrForbidden` otherwise. `AllowsAgentConfiguration` and `AllowsCandidateAdministration` are HR-only. ADMIN provisioning stays separate. There is no report operation name. |
| `infrastructure/runtime/agentconfiguration.go`, `access.go` | Administrative cookie, Origin, and `accesscredentials.CSRF` are already composed. Configuration routes are mounted before the `/admin/` access catch-all. |
| `pkg/errs/codes.go`, `translations/errors.en.toml`, `translations/errors.id.toml` | Shared envelope codes plus `IA_*` and `ACP_*`. No `IRP_*` codes. |
| `docs/api/openapi.yaml` | No report path. Assignment create and invitation routes exist. Revoke, report, artifacts, and document-executions are absent. |
| `infrastructure/runtime/shutdown.go` | Process shutdown budget remains 5 seconds, with a 4-second drain budget. |
| `test/architecture/dependencies_test.go` | Domain imports stay inside domain plus the listed standard library. Application must not import delivery, infrastructure, or GORM. |

The skill Backend Service Standard, Design Assessment, and Master Data, Migrations, and Seed Planning references were applied. Project instructions override the bundled golang-migrate "introduce a runner" default because `cmd/migrate` and `infrastructure/database/schema` already own the stream. They also override the bundled "no tests under `delivery/`" line: this repository proves HTTP mapping with mocked operation interfaces inside handler packages, which is what `AGENTS.md` requires.

Binding behavior is the capability rules, the separate report lifecycle, the completion-plus-outbox transaction, HR-only access, citation and verdict limits, the salary-ceiling exclusion, the candidate exclusion, and the accepted attempt budget. Package layout below is an internal choice. Dead-letter topology and the report-model contract stay deferred.

## 2. API / Interface Contract Changes

Platform technical design §9 already names the two routes that belong to this capability. Payload schemas there are still Open; this section is the feature contract to implement. Do not take ownership of revoke, artifacts, or document-executions.

| Method and path | Operation | Result |
|---|---|---|
| GET `/admin/interview-assignments/{id}/report` | Read the current HR account and the latest generation state | 200 detached view |
| POST `/admin/interview-assignments/{id}/report-generations` | Explicit idempotent regeneration | 201 new generation, or 200 replay of the same generation |

`{id}` is the interview assignment UUID. The server resolves the single execution pin. The client does not send an execution id, a report id, or a verdict.

Use one operation interface per command or query, implemented by a plain struct with injected ports. DTOs live in `application/dto`. Handlers must not receive domain aggregates or GORM models.

At implementation, update `docs/api/openapi.yaml`, add `docs/api/interview-reporting.md`, and update `docs/api/CHANGELOG.md` together. Dispatch both paths before the `/admin/` access catch-all. Reuse the administrative cookie, Origin check, and CSRF verifier. POST requires `X-CSRF-Token`. GET does not. Map failures to the `IRP_*` catalog below, not to `ACP_*` or `IA_*` codes. Candidate sessions fail principal resolution and receive 401 without a report body.

Process registration of these routes is T15 and stays blocked until a real writer and a real evidence catalog exist. T8 still specifies and unit-maps the contract through mocked operations so the HTTP behavior is fixed before that registration.

### In-process completion port

This is not an HTTP route. Execution Lifecycle's future completion command is the production caller. This capability does not implement BR-CORE-056.

`Admit` runs inside the caller's open transaction, after that caller has set `interview_assignments.completed_at` for the same assignment. It inserts the report lifecycle rows and returns. It does not commit, does not publish to RabbitMQ, and does not update the assignment or execution row. If `completed_at` is still null in that transaction, `Admit` returns an error and the caller rolls the completion back.

## 3. Data Model & Schema Changes

### Concepts and consistency

An **interview report** is the HR account of one finished interview. It is a different aggregate from the interview execution. Identity is one report per execution, and one report per assignment, because an assignment has at most one execution. The report's visible account can be replaced by a later generation. The interview's completed outcome cannot.

A **report generation** is one attempt-series to produce or replace that account. It has its own terminal outcome. Pending and running are active. Succeeded and failed are terminal for that generation. At most one generation for a report is active. History is kept: a failed regeneration does not delete the previous succeeded account.

A **statement** is an observation, a gap, or an uncertainty. An observation that is presented as something the candidate answered cites one finalized evidence entry and revision. A gap is not an answer. Statements do not carry a hire, reject, or rank.

A **report job** is the durable background work for one generation. The accepted budget is per job, read from platform technical design §10: 2 total attempts, 60 seconds each, 5-second retry delay, and 5 minutes from that job's creation, including queue time. Explicit regeneration creates a new generation and therefore a new job with a new budget. The phrase "total" does not accumulate across generations; HR asked for a new generation. It does forbid a third attempt inside one job.

An **outbox message** is the durable intent to publish that job. Its identity is independent of the job status. The broker is not the authority for completion.

`interview_assignments.completed_at` remains the sibling's completion marker. This feature reads it and does not write it. No status column is added to `interview_executions`. No transcript table is added here.

### Proposed storage layout

Version `000011_interview_reporting` follows interview-runtime `000010`. Apply it only after a clean version 10. All ids are UUID primary keys. Timestamps are `timestamptz`, written as UTC milliseconds by the application. Columns are `NOT NULL` unless marked null. Foreign keys use `ON UPDATE RESTRICT ON DELETE RESTRICT`. No soft delete and no cascade purge.

| Table | Columns and constraints | Why it is separate |
|---|---|---|
| `interview_reports` | `id`; `execution_id uuid` unique FK to `interview_executions`; `assignment_id uuid` unique FK to `interview_assignments`; `created_at` | One account lifecycle per finished interview |
| `interview_report_generations` | `id`; `report_id` FK; `generation_number int` check `> 0`; unique `(report_id, generation_number)`; `state text` check in `pending`, `running`, `succeeded`, `failed`; `failure_reason text` null, required iff `state = failed`, null otherwise; `created_at`; `updated_at` check `>= created_at` | Own outcome, history retained |
| `interview_report_statements` | `id`; `generation_id` FK; `ordinal int` check `>= 0`; unique `(generation_id, ordinal)`; `kind text` check in `observation`, `gap`, `uncertainty`; `body text` check length 1–2000; `evidence_entry_id uuid` null; `evidence_revision int` null check `> 0` when present; `presents_as_answer boolean`; check: `presents_as_answer` implies `kind = observation` and both evidence fields; check: `kind = gap` implies `presents_as_answer = false` and both evidence fields null | Citation is a constraint, not a JSON blob |
| `interview_report_commands` | `id`; `scope uuid` FK to `administrative_accounts`; `command_kind text` check equal to `regenerate-interview-report/v1`; `request_key text`; `intent_fingerprint text` check 64 lowercase hex; `generation_id` FK; `created_at`; unique `(scope, command_kind, request_key)` | Idempotent HR regeneration |
| `interview_report_jobs` | `id`; `generation_id uuid` unique FK; `state text` check in `pending`, `published`, `claimed`, `succeeded`, `failed`, `expired`; `attempt_count int` check `>= 0` and `<= 2`; `fencing_token bigint` check `> 0`; `created_at`; `deadline_at`; `eligible_at`; `attempt_deadline_at` null; `updated_at` | Per-generation provider budget |
| `interview_report_attempts` | `id`; `job_id` FK; `attempt_number int` check in `(1, 2)`; unique `(job_id, attempt_number)`; `fencing_token bigint`; `started_at`; `deadline_at`; `finished_at` null; `outcome text` null check in `succeeded`, `failed`, `expired`, `superseded` | Attempt history is not erased |
| `outbox_messages` | `id` uuid primary key, the message identity; `job_kind text` check equal to `interview_report`; `job_id uuid`; `created_at`; `publish_claimed_at` null; `confirmed_at` null | Durable publish intent. Not a second copy of the report |

Partial unique index `interview_report_generations_one_active_INDEX` on `(report_id) WHERE state IN ('pending', 'running')`.

Job deadline check: `deadline_at > created_at` and `deadline_at <= created_at + interval '5 minutes'`. The application sets `deadline_at` to `created_at` plus exactly 5 minutes and `eligible_at` to `created_at` on insert. Attempt deadline check: `deadline_at > started_at` and `deadline_at <= started_at + interval '60 seconds'`.

There is no salary-ceiling column, no recommendation column, and no rank column. Statement bodies are the only free text. Indexes: FK btree indexes using the existing quoted `"<table>_<column>_INDEX"` style, plus `outbox_messages_unconfirmed_INDEX` on `(created_at, id) WHERE confirmed_at IS NULL`, and `interview_report_jobs_eligible_INDEX` on `(eligible_at, id) WHERE state IN ('pending', 'published')`.

`outbox_messages.job_kind` admits only `interview_report` in this version. A later document-execution migration may widen that check. This version does not create document tables and does not invent their payload.

### Transactions and persistence mapping

GORM 1.25.12 maps rows in `infrastructure/repository/interviewreportrepository`. Models stay in that package. Use explicit `Select` updates so zero attempt counts, empty failure reasons, and null evidence pointers persist on purpose. Do not use `Updates(struct)` omission, `Save`, association autosave, or `Preload`. Load a generation's statements with one query, not one query per statement.

Raw SQL, through GORM `Raw` / `Exec` inside the same transaction, is required for two statements only:

- Claim the next eligible job with `FOR UPDATE SKIP LOCKED`, then insert its attempt and advance `attempt_count` and `fencing_token`.
- Claim a batch of unconfirmed outbox rows with `FOR UPDATE SKIP LOCKED`.

GORM's ordinary query helpers do not make that lock skip visible, and a lost lock here would start a second provider attempt. The partial active-generation index remains ordinary SQL in the migration, not a GORM hook.

Application code never sees `*gorm.DB`. Infrastructure stores the open transaction on the context inside `InTransaction`. `Admit`, regeneration insert, and failure update use that transaction when it is present.

| Transaction | Writes | Must not write |
|---|---|---|
| Completion admission | Report if absent, generation `pending`, job, one outbox row | Assignment, execution, RabbitMQ, provider |
| Regeneration | Command, generation, job, one outbox row, after locking assignment then report | Assignment `completed_at`, execution pin, previous statements |
| Attempt claim | Job claim fields, one attempt row | Report text, interview rows, broker ack |
| Generation success or failure | Attempt outcome, job terminal state, generation state, and statements only on success | `interview_assignments`, `interview_executions` |
| Publish confirmation | `outbox_messages.confirmed_at` and job `published` | Generation outcome. A confirm is not success |

Lock order for regeneration and for any path that also reads the assignment: `interview_assignments`, then `interview_executions`, then `interview_reports`, then generations, jobs, and outbox. Do not acquire the assignment lock after the execution lock. Completion already holds the assignment and execution locks before `Admit`; `Admit` must not lock them again.

Network I/O stays outside every transaction. The publisher runs after commit. The provider runs after the claim commits.

Rollback of the admission transaction removes the generation, job, and outbox row together with the sibling's `completed_at` write. A later generation failure updates only report tables. `make test` sqlmock coverage does not prove either outcome. T5 uses PostgreSQL.

Forward migration: `migrations/pgsql/000011_interview_reporting.up.sql` and `.down.sql`. Implementation also sets `schema.ReleaseVersion` to 11, appends the seven tables to `ownedPublicTables` and `RuntimeGrantTables`, and adds both files to `infrastructure/database/schema/checksums.json`. Startup still performs no DDL. `docs/design/database-migrations.md` moves the release target from clean version 10 to clean version 11.

Down drops report-owned tables only, in dependency order, on a disposable database. It does not drop `interview_assignments`, `interview_executions`, accounts, or configuration. Populated environments do not run down. Application rollback keeps version 11, stops the report consumer and publisher, and unmounts the two routes. A binary that accepts only version 10 will refuse to start on version 11; that is the existing guard, not a reason to down the tables.

No existing rows are backfilled. Interviews already stored without a report stay without one until a later completion or an explicit regeneration that meets the rules. There is no data rewrite of `completed_at`.

### Seed / Reference Data

No seed and no reference table. Attempt budgets are domain constants, not rows. ADMIN and HR remain the existing role check. Integration tests insert synthetic assignments, executions, and accounts through the test database. Migrations insert nothing.

### Rule ownership and proof plan

Every proof below is planned. None is observed evidence, and none marks BR-CORE-080..086 enforced.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| BR-CORE-080 | `InterviewReport` admission | `Admit` inside the completion transaction; regeneration does not clear `completed_at` | PostgreSQL: sibling stand-in sets `completed_at` and `Admit` commits together; report rows reference that execution; assignment and execution columns stay equal except the sibling's own completion write |
| BR-CORE-081 | Generation terminal transition | Failure and expiry transactions named above | PostgreSQL: fail a generation after commit; `interview_assignments` and `interview_executions` byte-compare equal, including `completed_at` and `updated_at` |
| BR-CORE-082 | Statement kind value and generation output | Accepting writer output; restoration | Observations, gaps, and uncertainties round-trip; a fourth kind cannot be constructed or stored |
| BR-CORE-083 | Statement value plus SQL checks | New output and restoration | Answer without evidence is rejected; gap with evidence is rejected; a citation that the evidence port does not list as finalized is rejected before insert |
| BR-CORE-084 | Statement value | Output acceptance; HTTP DTO | Verdict words rejected; DTO and OpenAPI have no recommendation, rank, or decision field |
| BR-CORE-085 | `InterviewReport` plus partial unique index | Regeneration transaction | Second key while pending or running returns conflict and inserts nothing; after failure, a new key inserts generation 2; concurrent keys produce one row |
| BR-CORE-086 and sibling BR-CORE-047 | HTTP surface | Report routes only; candidate routes unchanged | OpenAPI and handler tests: no candidate path returns report fields; candidate final-screen design is not extended |
| Sibling BR-CORE-071 | Writer input and response mapping | Compose writer request; map DTO | Request fixture and HTTP body contain neither `salaryCeiling`, `salaryCeilingRef`, nor a ceiling amount. No ceiling column |
| BR-CORE-022 | Existing `Authority` plus `AllowsInterviewReport` | GET, POST, and replay | ADMIN session is 403 before lookup; HR is allowed; candidate cookie is 401; no second authorizer type |
| Platform §10 report budget | Job and attempt values | Claim, retry scheduling, expiry | Controlled clock: attempt deadline 60s, retry eligible at +5s, no third attempt, job created at T expires at T+5min even if still queued |
| Platform §6 durable admission | Outbox row in the admission transaction | Completion and regeneration | Broker stopped: commit still leaves `confirmed_at` null and the interview completed. RabbitMQ proof is T11, not sqlmock |

## 4. Component Breakdown

| Layer / component | Status | Justification |
|---|---|---|
| `domain/aggregate/interviewreport` | New | Report root, generation entity, statement and attempt values, budget constants |
| `application/usecase/interviewreportusecase`, `application/query/interviewreportquery`, `application/dto` | New | Admission coordination, regeneration, detached HR view |
| `application/repository/interviewreportport`, `application/service` policy and `InterviewReportAuthority` | New ports; policy file modified | Transactions, evidence, writer, and HR predicates. No GORM |
| `infrastructure/repository/interviewreportrepository` | New | GORM maps and the two raw claim statements |
| `infrastructure/service/reportbroker` | New | RabbitMQ publish, confirm, consume. No dead-letter declaration |
| `infrastructure/service/accessauthority` | Modified | Same `Authority` struct implements the report interface |
| `delivery/http/handler/interviewreporthandler` | New | Strict JSON, cookie, CSRF, `IRP_*` mapping |
| `infrastructure/runtime`, `infrastructure/database/schema` | Modified | Route and worker wiring is T15 and blocked. Version floor becomes 11 in T3 |
| `migrations/pgsql/000011_interview_reporting.*` | New | The only DDL |
| `pkg/errs`, `translations/errors.*.toml`, `docs/api/*` | Modified | Public catalog and contract |
| Candidate handlers, voice runtime, Task queue, `StartExecution` | Reused as-is | Not report carriers |

Authorization is inside the use case and the query, not only in the handler. The principal comes from `Authority.Principal`.

### Detail per component

**Report aggregate.** `InterviewReport` is the root. Generations are entities addressed by report id and generation number. Statements and the attempt budget are values. The root is the only mutation entry for "start a generation" and "record generation outcome". Callers cannot set the interview to completed through this type. Restoration loads stored generations and statements and checks identity and citation shape. It does not re-run the provider or re-apply today's writer. A row that violates the citation checks is a technical restoration failure, not an HR account.

**Admission use case.** Opens nothing of its own when the caller already entered `InTransaction`. Verifies `completed_at` is visible in that transaction, then asks the store to insert. The first generation number is 1. A repeat call for an execution that already has a report returns that first generation and does not insert another active one.

**Regenerate use case.** Resolves HR, locks the assignment, requires non-null `completed_at`, asks the evidence port, then starts a generation only when none is active. Same idempotency key and same fingerprint return the original generation without checking the active flag again, matching publication replay. The evidence port is an interface. Its production adapter is T14 and is blocked.

**Worker.** After a claim commits, calls the writer port with finalized evidence and without salary-ceiling material, then commits statements and the generation outcome in a new transaction if the fencing token still matches. The production writer is T13 and is blocked. Tests use a fake writer.

**Broker.** Publishes the job id and message id only. Manual ack, persistent delivery, publisher confirms. Confirms update the outbox after the broker confirms. Lost confirms can duplicate messages; the consumer loads the job and does not start a second attempt while one is active and unexpired.

**Authority.** Add `AllowsInterviewReport(role enum.Role) bool` as `role == HR`, and operation names `read` and `regenerate` recognized only by `KnownInterviewReportOperation`. Add `InterviewReportAuthority` with `Principal` and `Authorize` in `application/service`. `accessauthority.Authority` implements it beside `AgentConfigurationAuthority`. `Authorize` rechecks the bound session and calls `AllowsInterviewReport`. It does not call `AllowsAgentConfiguration`. Configuration authorization stays on its own method. No new session store and no middleware-only role check.

### Material structure decisions and assessment

| Choice | Reason and material alternatives | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| Separate report aggregate, not a field on the execution pin | BR-CORE-080 requires an outcome that can fail while the interview stays completed. Folding both into one aggregate would make every report write compete with voice progress and would invite one status column to mean both. A shared table row updated in place would erase generation history | Capability BR-CORE-080..081; platform §5 "completed interview → report generations"; H-10 | Retained |
| Relational statements instead of one JSONB document | The citation rule is a constraint on each statement. A document would leave "answered without evidence" to every reader. The extra table is the smaller enforcement surface | BR-CORE-083; GORM default does not justify JSONB here | Retained |
| One `Authority` struct, new report interface | A second authorizer would duplicate session expiry, revocation, and ADMIN denial. Overloading `AgentConfigurationAuthority.Authorize` would let a report operation pass through the configuration catalog by mistake. Two methods on one struct keep one session lookup and separate operation lists | User instruction; verified `authority.go` and `policy.go`; BR-CORE-022 | Retained |
| Shared transaction through context, GORM hidden | The completion write and the outbox insert are one commit in platform §5. Passing `*gorm.DB` into application breaks the import rule. Two independent commits can leave a completed interview with no durable job, or a job whose interview rolled back | Platform §5 interview-completion row; `dependencies_test.go`; AGENTS.md | Retained |
| Generic outbox table with a single allowed `job_kind` | TD-09 is the platform mechanism for this background work. A report-only queue table would be a second mechanism when document work arrives. A universal workflow coordinator is the structure the platform overview rejected, so the table stores publish intent only | TD-09; platform §2 rejection of a universal coordinator | Retained. Widening `job_kind` is a later migration |
| RabbitMQ for report jobs, not for voice | The candidate must not wait on the report, and the live loop already has its own session path. Reusing the Task in-process queue would drop the job on process death and would couple Task capacity to HR write-up | TD-09; BR-CORE-047; verified Task queue and absent AMQP client | Retained. Dead-letter topology deferred |
| Per-job attempt budget | The §10 wording starts the five minutes at job creation. A lifetime cap of two attempts per interview would make an explicit regeneration after one failure unable to run, which contradicts BR-CORE-085 | Later technical design §10 wins over a looser reading of "total" | Retained |
| Phrase guard for verdicts, no model classifier | The output type has no verdict field. Free text can still say "hire". A small word-boundary list is enforceable without the open model contract. It will miss paraphrases | BR-CORE-084 | Retained as a bounded guard. A model-output classifier stays inside the blocked provider task |
| Do not resolve the salary ceiling value | Configuration stores `salaryCeilingRef` and responses expose only `salaryCeilingConfigured`. No value table exists. Inventing a store to scan report text would expand scope. Exclusion from the writer input, columns, logs, and HTTP body is the enforceable rule | Sibling BR-CORE-071; verified configuration DTO | Retained. A value scan is deferred until a value store exists; no task is executable for it |
| Consumer not started until writer and evidence adapter exist | Starting the consumer against a missing provider would spend the two attempts and the five-minute window on an empty client | §10 budget; §14 model contract Open; no transcript table | Deferred. Blocks T15 only |

Assessment gate: each rule in the table above has an owner, an enforcement path, and a planned proof. The dead-letter consumer behavior and the production writer are the deferred choices. Independent schema, domain, repository, HTTP mapping, and publisher work stay executable.

## 5. Flow Sequence

### Completion admission

1. Execution Lifecycle, not this feature, decides that required progress is stored and sets `completed_at` in an open transaction.
2. It calls `Admit` with the assignment id and execution id before commit.
3. `Admit` reads `completed_at` in that transaction. Null returns an error. The caller aborts.
4. The store inserts the report if needed, generation 1 as `pending`, the job with `deadline_at = created_at + 5 minutes`, and one outbox row.
5. The caller commits. The candidate final screen can show the interview outcome without reading the report.
6. After commit, the publisher reads the outbox and publishes. Broker failure leaves the row unconfirmed. The interview stays completed.

### Worker

1. The consumer, once T15 allows it, reads a message containing `jobKind`, `jobId`, and `messageId`.
2. It loads the job. A terminal job is acknowledged with no provider call.
3. An active unexpired attempt is acknowledged with no second attempt. The reconciler expires stuck attempts.
4. A claim transaction inserts attempt 1 or 2, sets the attempt deadline to the earlier of 60 seconds and the job deadline, and commits.
5. The writer runs outside the transaction. Accepted statements commit with the generation outcome. Rejected output fails the attempt.
6. If an attempt remains and the job deadline allows it, `eligible_at` becomes the finish time plus 5 seconds. The reconciler publishes again through a new outbox row. No delay plugin is used.
7. Exhaustion or a deadline that leaves no positive attempt time marks the generation `failed` and the job `expired` or `failed`. Interview rows are not in the statement.

### HR read and regeneration

1. Handler resolves the cookie and, for POST, CSRF.
2. `Authority` rechecks the current HR session. ADMIN is forbidden before the assignment lookup.
3. GET returns the latest succeeded statements, or a null account, plus the newest generation state. It does not call the provider.
4. POST requires one `Idempotency-Key`. Replay of the same key and fingerprint returns that generation. A different key while a generation is active conflicts. A different key after a terminal generation inserts the next generation, job, and outbox in one transaction.

### Reconciler

On startup and on a fixed interval inside the process, the reconciler claims unconfirmed outbox rows and republishes them. It also looks for claimed attempts whose `attempt_deadline_at` has passed, fences them, and schedules a retry only when the budget allows. Publisher confirmation never marks a generation succeeded.

Shutdown keeps the existing 5-second process budget. Stop claiming, let an in-flight database commit finish inside the 4-second drain, then cancel. Unconfirmed outbox rows remain for the next process. Shutdown does not cancel the interview.

## 6. Request / Response

### GET `/admin/interview-assignments/{id}/report`

No body and no query. Cookie and Origin are required.

```json
{
  "data": {
    "assignmentId": "…",
    "executionId": "…",
    "generation": {
      "id": "…",
      "number": 1,
      "state": "pending",
      "failureReason": null,
      "createdAt": "2026-11-02T02:00:00.000Z"
    },
    "report": null
  }
}
```

`report` is null unless a generation has `succeeded`. Then:

```json
{
  "report": {
    "generationId": "…",
    "observations": [
      {"body": "The candidate described backend experience.", "evidenceEntryId": "…", "evidenceRevision": 1}
    ],
    "gaps": [
      {"body": "Compensation was not answered.", "questionId": "compensation"}
    ],
    "uncertainties": [
      {"body": "The compensation expectation was unclear.", "evidenceEntryId": "…", "evidenceRevision": 2}
    ]
  }
}
```

Gaps omit evidence ids. Observations in this view are statements that present as answers and always include both evidence fields. Uncertainties include a citation when the writer supplied one; an uncertainty without a citation is allowed and must not be rendered as an answer. `questionId` on a gap is the sibling coverage id when the evidence port supplied one, otherwise null. No field named recommendation, decision, rank, score, `salaryCeiling`, or `salaryCeilingRef` is accepted or returned.

While a newer generation is active, `generation` is that active generation and `report` remains the previous succeeded account, or null if none succeeded. A failed regeneration does not clear `report`.

Timestamps are UTC with millisecond precision. Unknown response fields are not added.

### POST `/admin/interview-assignments/{id}/report-generations`

Body is an empty JSON object. Unknown fields, a missing body, and a non-object body are `INVALID_INPUT`. Header `Idempotency-Key` uses the publication rule: one value, no trim, pattern `[A-Za-z0-9][A-Za-z0-9._:-]{0,127}`, duplicate headers rejected.

Fingerprint is lowercase hex SHA-256 of the UTF-8 compact JSON array `["regenerate-interview-report/v1", "<assignment-uuid-lowercase>"]`. Scope is `AdminPrincipal.SubjectID`. The same key with a different assignment is a conflict. The same key and fingerprint return the original generation with 200 even if a later generation exists. A new key after a terminal generation returns 201 and the new pending generation. The response `data` is the generation object above, not the finished report.

### Public errors and precedence

Use the standard envelope. Safe English and Indonesian messages:

| Priority | Condition | HTTP / code | English | Indonesian |
|---|---|---|---|---|
| 1 | Missing, candidate, expired, revoked, or unbound administrative session | 401 `IRP_UNAUTHENTICATED` | administrative session is missing or no longer valid | sesi administratif hilang atau tidak lagi valid |
| 2 | ADMIN, or POST without a valid CSRF token | 403 `IRP_FORBIDDEN` | interview report is forbidden | laporan wawancara dilarang |
| 3 | Malformed id, body, or idempotency key | 400 `INVALID_INPUT` | existing message | existing message |
| 4 | Assignment id is unknown | 404 `NOT_FOUND` | existing message | existing message |
| 5 | Same key, different assignment or fingerprint | 409 `IRP_IDEMPOTENCY_CONFLICT` | report request identity is already used with a different intent | identitas permintaan laporan sudah digunakan dengan maksud berbeda |
| 6 | Same key and same fingerprint | 200 replay | — | — |
| 7 | `completed_at` is null | 409 `IRP_NOT_COMPLETED` | interview report is available only after the interview is completed | laporan wawancara tersedia hanya setelah wawancara selesai |
| 8 | Completed, and the evidence port reports nothing finalized | 409 `IRP_EVIDENCE_UNAVAILABLE` | interview report needs the finalized interview record | laporan wawancara membutuhkan rekam wawancara yang sudah final |
| 9 | Another generation is pending or running | 409 `IRP_GENERATION_ACTIVE` | a report generation is already active | pembuatan laporan sedang berjalan |
| 10 | Database unavailable before commit | 503 `IRP_DEPENDENCY_UNAVAILABLE` | interview report could not be stored | laporan wawancara tidak dapat disimpan |
| 11 | Stored report rows violate their checks | 500 `INTERNAL_ERROR` | existing message | existing message |

Authorization runs before lookup, so ADMIN does not learn whether the assignment exists. Broker unavailability after a committed outbox row is not 503; regeneration still returns 201 or 200. Do not put transcript text, statement bodies, ceiling values, cookies, or provider errors in the envelope.

### Creation, restoration, and transition contracts

| Path | Accepted inputs and normalization | Errors and timing | Side effects | Proof plan |
|---|---|---|---|---|
| Completion `Admit` | Caller already set `completed_at` in this transaction. Execution belongs to the assignment. No client JSON | Null `completed_at` fails before insert. Insert failure fails the caller's transaction | One pending generation, one job, one outbox row. No broker call | PostgreSQL commit and injected outbox failure |
| Restoration | Rows that satisfy the checks, including historical failed generations | Corrupt citation or unknown state is a technical error at load. No provider call | None | Round-trip pending, failed, and succeeded fixtures |
| Regeneration | HR, completed assignment, evidence available, no active generation, or an exact replay | Precedence table. Conflict inserts nothing | New generation only for a new key after a terminal generation | Concurrent unique index; replay does not add a row |
| Attempt transition | Job inside its deadline, `attempt_count < 2`, `eligible_at <= now` | Lost claim race inserts one attempt. Late result with an old fencing token does not write statements | Attempt row and, later, generation outcome | Controlled clock and two PostgreSQL claimers |
| Writer output | Three kinds only. Answer cites evidence the port calls finalized. Body fails the verdict word list: hire, hiring, reject, rejection, rank, ranking, matched on Unicode word boundaries after case fold | Rejection fails the attempt, not the interview. No statement rows | Statements inserted only with generation `succeeded` | Table tests for boundaries, gaps, and prohibited words |

`higher` does not match `hire`. The list is an English guard for the words in BR-CORE-084. It is not a paraphrase detector.

## 7. Error Handling

Domain and application errors are typed values the handler maps with the precedence table. Store corruption and driver failures wrap `errs.Technical` and become `INTERNAL_ERROR` without the cause in the response. Unique-index loss on the active generation becomes `IRP_GENERATION_ACTIVE` after the transaction rolls back. A unique command loss with the same fingerprint is reloaded and returned as replay. The same command key with a different fingerprint is `IRP_IDEMPOTENCY_CONFLICT`.

The consumer does not acknowledge a message when the database cannot be read. It acknowledges a message whose job is already terminal or whose attempt is already active. It does not acknowledge a body it cannot parse, and it does not requeue that body in a loop. The isolation of that body is T12 and is blocked. Until T12, the test consumer used by T11 publishes only well-formed job messages.

Logs may include assignment id, execution id, generation id, job id, outbox id, attempt number, and state. They exclude cookies, CSRF tokens, statement bodies, transcript text, provider payloads, and salary fields.

## 8. Constraints Checklist

- [ ] Report lifecycle is stored separately from the interview, and failure updates do not write interview tables.
- [ ] Completion and the first outbox row commit in one transaction. The candidate API is unchanged.
- [ ] HR uses the existing `accessauthority.Authority`. ADMIN is forbidden. No second authorizer.
- [ ] Statements are observations, gaps, or uncertainties, cite finalized evidence when presented as answers, and do not hire, reject, or rank.
- [ ] Salary ceiling value and reference are absent from schema, writer input, logs, and the HR response.
- [ ] Attempt budget is 2 attempts, 60 seconds, 5-second delay, 5 minutes from job creation.
- [ ] RabbitMQ plus outbox dispatch this work only. Voice and Task queues stay as they are.
- [ ] Dead-letter topology is not invented. The malformed-message task stays blocked.
- [ ] Migration `000011` is manual SQL. `ReleaseVersion` becomes 11. No AutoMigrate and no seed.
- [ ] OpenAPI, `docs/api/interview-reporting.md`, and `docs/api/CHANGELOG.md` change together.
- [ ] No admin or HR UI, and no ownership of revoke, artifacts, or document-executions.
- [ ] BR-CORE-080..086 remain planned until the proofs in §3 have actually run.

## 9. Risks & Open Questions

| ID | Decision | Owner | Blocks | Ready when |
|---|---|---|---|---|
| Q1 | Exact dead-letter topology | Platform technical design §6, still Open | T12 only | The topology is written into §5 and §7 of this document: exchange, queue binding, and what happens to a malformed body. This design must not guess those names |
| Q2 | Report model and invocation schema | Platform technical design §14 checklist | T13 and, with Q3, T15 | A provider, model, and structured request/response are approved and copied into §6. The attempt budget stays as already accepted |
| Q3 | Finalized evidence identity and storage | Interview Runtime. Its provider event mapping is still Open, and no transcript table exists | T14 and, with Q2, T15 | Runtime exposes stable finalized entry id, revision, execution id, and optional question id through a read adapter. This feature does not design voice or stages |

Executable without those answers: domain rules, ports, migration `000011`, repository, PostgreSQL admission and non-interference proofs, authority extension, use cases against fakes, HTTP mapping, publisher, attempt state machine, and a RabbitMQ test that uses well-formed messages.

Salary-ceiling value scanning has no source data. It is a documented limitation, not a blocked implementation task. The exclusion proofs in §3 do not wait for it.

`interview_assignments.completed_at` is the integration marker because it is the only completion column and nothing writes it today. If Execution Lifecycle chooses a different column, update `Admit` and the regeneration predicate before T5 is implemented. That handoff is part of this design, not a second completion rule.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial design | Mode A. No report implementation exists |
| v2 | 2026-11-02 | Assign `000011_interview_reporting` and clean floor 11 | assign a unique migration version after the parallel tripack collision on 000008 |
