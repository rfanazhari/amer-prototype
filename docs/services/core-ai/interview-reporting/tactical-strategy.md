# Tactical Strategy: Interview Reporting

## 1. Implementation Strategy

Mode A, additive work in the existing Go module. Implement the assessed [technical design](technical-design.md) for [BR-CORE-080..086](../business/capabilities/interview-reporting.md). Those rules are planned. Shipping this code does not by itself mark them enforced; the proofs below have to be run and recorded.

Keep Task, voice, administrative access, agent configuration, assignment create, and candidate invitation behavior compatible. Do not register AutoMigrate. Do not add a second authorizer. Do not add an admin or HR UI. Do not implement consent, voice, stages, revoke, artifacts, or document executions.

Schema version `000011_interview_reporting` follows interview-runtime `000010`. The repository's verified floor is still `ReleaseVersion` 7 and `000007_interview_executions`; this slice is not the next file. Startup will accept only clean version 11 after that migration task. GORM 1.25.12 stays the CRUD adapter. Raw SQL is limited to the job claim and the outbox claim, both with `FOR UPDATE SKIP LOCKED`, inside the GORM transaction.

RabbitMQ is the dispatch mechanism for this background work (platform TD-09 and technical design §6). The live voice routes and the Task in-process queue stay unchanged. Dead-letter topology is not part of the executable slice.

The process must not start the report consumer, and must not mount the two report routes, until the writer and the evidence adapter exist. Those two adapters are blocked. Handler tests and publisher tests still proceed. Otherwise the five-minute job window would expire against a missing provider.

## 2. Code Reuse Map

Paths were inspected. "Reused" means the current behavior is a constraint or a pattern to call, not evidence that the report design is already implemented.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `Authority` | `infrastructure/service/accessauthority/authority.go` | Session resolve, expiry, revocation, ADMIN denial | Extend the same struct with a report interface. `Authorize` today allows only agent-configuration operations. Do not copy it into a second type and do not route report operations through `AllowsAgentConfiguration` |
| Role predicates | `application/service/policy.go` | HR-only checks | Add `AllowsInterviewReport` and report operation names. `AllowsAccountProvisioning` stays ADMIN-only |
| `Clock` | `application/service/ports.go`, `infrastructure/service/clock/clock.go` | Trusted UTC time | Reuse. Application truncates to milliseconds. Domain budget constants do not read the wall clock |
| CSRF and admin cookie | `infrastructure/service/accesscredentials`, `infrastructure/runtime/access.go`, `agentconfiguration.go` | POST protection and principal cookie | Same verifier and cookie. Report errors use `IRP_*`, not `ACP_*` |
| Schema runner | `infrastructure/database/schema/`, `cmd/migrate`, `migrations/pgsql/` | Ordered SQL, checksums, clean-version guard | This slice's version is 11, after interview-runtime `000010`. Versions 1–10 stay immutable. Runtime grants are the `RuntimeGrantTables` list |
| GORM transaction callback | `infrastructure/repository/taskrepository/task.go`, `agentconfigrepository/store.go` | Pattern for a short transaction | Task completion publishes an in-process event. That queue must not carry report jobs. Publication's command table is the pattern for regeneration idempotency, not a table to reuse |
| `interview_assignments.completed_at` | `migrations/pgsql/000005_interview_assignments.up.sql` | Sibling completion marker this feature reads | No Go writer sets it today. This feature must not start setting it |
| `interview_executions` | `migrations/pgsql/000007_interview_executions.up.sql` | Parent pin for the report FK | No status and no transcript. `StartExecution` only inserts the pin |
| Envelope and codes | `pkg/errs/codes.go`, `translations/errors.*.toml`, `docs/api/openapi.yaml` | Error shape and contract tests | Add `IRP_*` and the two paths. Do not retag revoke, artifacts, or document routes |
| Compose RabbitMQ | `compose.yaml` profile `rabbitmq`, `README.md` | Broker for T11 | Optional local profile only. No Go client exists. Readiness in `make docker.verify` is not report dispatch proof |
| Candidate and voice handlers | `delivery/http/handler/accesshandler`, `infrastructure/runtime/voice.go` | Left unchanged | Prove they gain no report fields |

## 3. Net-New Code Inventory

| Item | Layer | Package / path | Type | Ownership / notes |
|---|---|---|---|---|
| `InterviewReport` | Domain | `domain/aggregate/interviewreport` | Aggregate root | One per execution and assignment. Owns generation start and generation outcome |
| Report generation | Domain | same package | Entity | Owned by the report. Active or terminal |
| Statement | Domain | same package | Value object | Observation, gap, or uncertainty. Owns citation and verdict-word rules |
| Attempt budget | Domain | same package | Value behavior of the job rules | Constants: 2 attempts, 60 seconds, 5 seconds, 5 minutes. Not a separate service |
| `Admit`, `Regenerate` | Application | `application/usecase/interviewreportusecase` | Use cases | Plain structs, injected ports |
| Read report | Application | `application/query/interviewreportquery` | Query | Detached DTO |
| Report DTO | Application | `application/dto` | DTO | No verdict and no salary fields |
| Repository port | Application | `application/repository/interviewreportport` | Port | `InTransaction`, `Admit`, claim, outcome. No GORM |
| `InterviewReportAuthority` | Application | `application/service` | Port | Implemented by the existing `Authority` |
| `AllowsInterviewReport` | Application | `application/service/policy.go` | Predicate | HR only |
| Evidence and writer ports | Application | `application/service` or `interviewreportport` | Ports | Production adapters are T14 and T13 |
| GORM models and claim SQL | Infrastructure | `infrastructure/repository/interviewreportrepository` | Adapter | CRUD via GORM. Raw SQL only for the two `SKIP LOCKED` claims |
| Broker client | Infrastructure | `infrastructure/service/reportbroker` | Adapter | New module `github.com/rabbitmq/amqp091-go`, pinned when added. No dead-letter arguments |
| HTTP handler | Delivery | `delivery/http/handler/interviewreporthandler` | Handler | Mocked operations in tests |
| Migration pair | Schema | `migrations/pgsql/000011_interview_reporting.up.sql` and `.down.sql` | Migration | Creates the seven tables in design §3. Down is disposable |
| Release floor | Schema | `version.go`, `checksums.json`, `docs/design/database-migrations.md` | Configuration | Clean version 11, grants, checksums |
| Error catalog | Public contract | `pkg/errs/codes.go`, `translations/errors.en.toml`, `translations/errors.id.toml` | Codes | Messages in design §6 |
| API document | Contract | `docs/api/openapi.yaml`, `docs/api/interview-reporting.md`, `docs/api/CHANGELOG.md` | Contract | Together in one task |
| Mockery entries | Test | `.mockery.yml`, `test/mocks/` | Generated | New ports only. Do not hand-write mocks |

No seeder, no reference rows, and no change to `000001`–`000010`.

## 4. Branching / PR Strategy

One feature branch is enough. Commit in this order so each commit builds:

1. Domain and application ports (T1, T2).
2. Migration and release floor (T3).
3. Repository and PostgreSQL proofs (T4, T5).
4. Authority extension (T6), which can follow T2 and land before or beside T4.
5. Use cases and HTTP contract (T7, T8) without mounting routes in `runtime`.
6. Publisher, attempt worker, and RabbitMQ proof (T9, T10, T11) with the consumer unregistered.

Leave T12–T15 off the merge until their decisions are written into the technical design. Do not merge a runtime registration that starts the consumer early.

## 5. Testing Strategy

Planned checks, not current evidence. Domain and application tests use table-driven `_test` packages, testify, and Mockery. HTTP tests mock the operation interfaces. sqlmock may assert the text of the claim statement. sqlmock does not prove row locks, rollback, or RabbitMQ.

Applicable Makefile targets:

| Target | Use |
|---|---|
| `make test` | Domain and application race tests |
| `make test.contract` | OpenAPI after T8 |
| `make test.architecture` | Import direction and declaration comments |
| `make lint` | Go changes |
| `make mocks` | After port changes |
| `make test.security` | After the RabbitMQ module is added |
| `make test.integration` | PostgreSQL proofs. Requires `TEST_DATABASE_URL`. Skipped is not a pass |
| `make test.docker` | Isolated PostgreSQL when a local URL is not authorized |

`make docker.verify` RabbitMQ readiness does not prove report publish, confirm, or redelivery. T11 is a separate broker test.

### Unit tests

- Statement citation, gap shape, verdict word boundaries, and `higher` not matching `hire`.
- Generation start refused while another is active; allowed after success or failure; interview state is not a field on the aggregate.
- Budget: second attempt eligible 5 seconds later; third attempt refused; remaining job window shorter than 60 seconds caps the attempt; window already closed fails the job with no attempt.
- Idempotent replay versus conflicting key.
- `AllowsInterviewReport` allows HR and denies ADMIN.
- Handler mapping for the precedence table, using mocked operations. ADMIN forbidden before the mock store is called.

### Integration tests

- Happy path: one PostgreSQL transaction sets `completed_at` as the lifecycle stand-in and calls `Admit`. Commit leaves the report, generation, job, and outbox row. The execution pin is unchanged apart from existing columns.
- Error: outbox insert failure rolls back `completed_at` and all report rows.
- Error: after a successful admission, failing the generation does not change `completed_at`, `updated_at`, or any execution column.
- Concurrent regeneration: one active generation.
- Two claimers: one attempt row.
- Broker stopped during admission: commit succeeds, `confirmed_at` stays null.
- T11, with the Compose RabbitMQ profile: persistent publish, publisher confirm recorded, duplicate delivery does not add an attempt, worker stop before ack redelivers, database error is not acknowledged. Well-formed bodies only.

### Regression tests

- `make test` still passes Task, access, and agent-configuration packages.
- Candidate OpenAPI and voice routes gain no report schema.
- Configuration authority still denies ADMIN and allows HR.
- Clean version 10 is refused after T3, and versions 1–10 checksums stay the same.
- `StartExecution` still only inserts the pin.

## 6. Rollout & Rollback

- Rollout: apply `000011` with the existing migrate command to an authorized database, then start a binary whose `ReleaseVersion` is 11. Do not mount report routes or start the consumer in that binary until T15 is unblocked. Publisher registration follows the same gate so a queued job is not created in production by a live completion hook before a worker exists. Tests may construct the publisher directly.
- No feature flag service is added. The gate is "routes and workers are unwired."
- No seed job runs in any environment.
- Rollback: stop the process, unmount is unnecessary if T15 never wired the routes, and keep the version 11 tables. Do not run the down migration on a shared database. A version 10 binary will not pass the startup guard; use a version 11 binary with the report workers unwired, or restore a backup taken before `000011`.
- Logs for any later enabled worker follow design §7. No salary value and no transcript body.
