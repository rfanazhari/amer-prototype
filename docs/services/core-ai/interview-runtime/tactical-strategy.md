# Tactical Strategy: Interview Runtime

## 1. Implementation Strategy

Mode A, additive interview behavior on the existing modular monolith. The assessed design is [technical-design.md](technical-design.md). Rules `BR-CORE-060`..`BR-CORE-071` are planned and are not enforced today.

Implement the domain workflow, the version 10 tables, the GORM adapter, the driver transport, and the instruction builder without waiting for provider event names. Call Execution Lifecycle for activation, fencing, and completion. Do not add the five execution states to this slice. Leave report generation to Interview Reporting.

Mount `POST /candidate/executions/{id}/session-attempts` and the readiness route only when the lifecycle calls exist. In that same change, remove `GET /voice-session` and `POST /voice-sessions` from the production router and from the public contract. Until that change, those routes remain the unauthenticated prototype and are not called by interview code.

The live voice loop does not use RabbitMQ. GPT-Live is the driver target. Tests of `CORE-IRT-006` use a fake HTTP server. A live `H-01` pass is `CORE-IRT-017` and stays blocked.

Schema installation follows the manual migration stream. `schema.ReleaseVersion` is 7 today. This slice proposes version 10, after execution-lifecycle `000008` and candidate-interview `000009`. Startup must keep refusing to migrate.

## 2. Code Reuse Map

Paths were read during planning. They describe current behavior.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `QuestionStage`, `HRConfiguration`, `AssessHR` | `domain/enum/questionstage.go`, `domain/valueobject/hrconfiguration.go`, `domain/agentconfig/hr.go` | Stage names, question shape, 300–600 publication total, `expectations_only`, ceiling reference id | Publication assessment is not the live workflow. No follow-up count field. Closing is optional for questions and required for a positive duration. Do not import publication findings into progress decisions. |
| `agentreference.Local` | `infrastructure/service/agentreference/local.go` | Role name for the demo job description | Returns no salary amount and no job-description body. Instructions may include the role string only. |
| `normalizeSDP`, `Negotiate.Execute` | `application/usecase/voicesession/negotiate.go` | CRLF SDP rule | `Execute` attaches prototype instructions and hides the provider error. Interview code does not call it. |
| `realtimecall.Adapter.Exchange` | `infrastructure/service/realtimecall/adapter.go` | Live sessions URL, `gpt-live-1` default, bearer POST, SDP answer parse | Discards `session.id`. Unauthenticated production route. New `interviewvoice` adapter retains the id. |
| `voicepage/page.html` | `delivery/http/voicepage/page.html` | Microphone, peer connection, ICE, playback | Candidate Interview embeds these mechanics. This slice does not serve the page. |
| `liveprobe.DecodeSetup`, `Shape` | `infrastructure/service/liveprobe/probe.go` | Session-id shape and payload-free event type check | Not wired into production. `Shape` does not decide finality. |
| `StartExecution` | `application/usecase/agentbindingusecase/start_execution.go` | Confirms the pin is assignment id plus revision id | Does not activate or fence. Do not add stages here. |
| `interview_executions` | `migrations/pgsql/000007_interview_executions.up.sql` | FK target for progress | Columns are id, assignment, instance, revision, `created_at`. No status column to update. |
| `Store.Update` | `infrastructure/repository/taskrepository/task.go` | Row lock, callback, rollback, explicit `Select` | Task columns and the task aggregate are unrelated. Copy the transaction shape in the interview store. |
| `Clock`, `System` | `application/service/ports.go`, `infrastructure/service/clock/clock.go` | UTC millisecond `Now` | Reuse as-is. Stage and deadline tests inject a fixed clock. |
| Envelope, strict JSON, `errs.Technical` | `delivery/http/handler/voicesessionhandler/handler.go`, `pkg/errs/codes.go` | Envelope and 1 MiB unknown-field decode | Handler stays private to the prototype. Interview handler owns its decode and maps `IRT_*` codes. |
| Candidate cookie and CSRF | `delivery/http/handler/accesshandler/` | 401 / 403 before interview logic | Reuse the verifier. Scope mismatch on an execution is 404. |
| Revision reader | `application/repository/agentconfigport` | Load the pinned immutable revision | Read by the execution's `published_revision_id`. Do not read the instance's active pointer. |
| Schema runner and `ReleaseVersion` | `infrastructure/database/schema/` | Next SQL version and startup guard | Today the guard and `ReleaseVersion` are 7. This slice sets both to 10 after execution-lifecycle `000008` and candidate-interview `000009`. No `AutoMigrate`. |
| Architecture tests | `test/architecture/dependencies_test.go`, `documentation_test.go` | Import direction and comments on every new declaration | Domain stays inside the allowed standard-library set. |
| Mockery | `.mockery.yml`, `make mocks` | Generate port mocks | Register new interfaces. Do not hand-write mocks. |
| Task event queue and RabbitMQ compose | `infrastructure/service/eventqueue`, `make docker.rabbitmq` | Unrelated | Not in the interview graph. |

`test/livevoice` is a manual probe. A passing `TestManualProbe` is not `H-01` and not Q1.

## 3. Net-New Code Inventory

Names may be refined inside the design's refinement boundary. Classification is by responsibility.

| Item | Layer / component | Package / path | Type | Ownership / notes |
|---|---|---|---|---|
| `Progress` | Domain | `domain/interview` | Aggregate root | One per execution. Owns stage cursor, cap, and version. |
| Coverage item | Domain | `domain/interview` | Entity inside Progress | States pending, asked, answered, omitted. |
| Technical loss | Domain | `domain/interview` | Entity inside Progress | Blocks authorization. Separate from omission. |
| Transcript entry | Domain | `domain/interview` | Entity inside Progress | Fragment versus final. Revision supersedes. |
| Handoff | Domain | `domain/interview` | Entity inside Progress | One per source stage. Closing has a null target. |
| Checkpoint | Domain | `domain/interview` | Entity inside Progress | Latest wins for recovery context. |
| Voice session | Domain | `domain/interview` | Entity inside Progress | One open session per execution. |
| Omission reason, evidence state, speaker | Domain | `domain/interview` | Value / enum behavior | Owned with the entity that stores them. A separate type is optional. |
| `NewProgress` / `RestoreProgress` | Domain | `domain/interview` | Factory functions | Create applies the frozen cap. Restore does not re-run publication rules. |
| `Authorize`, coverage methods, `ReadyForCompletion` | Domain | `domain/interview` | Aggregate behavior | Predicates stay methods. No domain service. |
| Interview operations | Application | `application/usecase/interviewusecase` | Operation interfaces and structs | Activate, ready, propose, reconnect, complete-or-fail. |
| Instruction builder | Application | `application/usecase/interviewusecase` | Function on the operation | Drops salary id and amount. |
| Progress port | Application | `application/repository/interviewport` | Repository contract | Commit callback, no GORM handle. |
| Lifecycle port | Application | `application/service` | Port | Activate, fence, locked callback, complete, fail. |
| Driver port | Application | `application/service` | Port | Create, attach, observe, supply context, control, close. |
| DTOs | Application | `application/dto` | Detached structs | Candidate-safe read model. |
| GORM models and store | Infrastructure | `infrastructure/repository/interviewrepository` | Adapter | Explicit `Select`. JSON is not used for coverage state. |
| Version 10 SQL | Infrastructure | `migrations/pgsql/000010_interview_runtime.{up,down}.sql` | Migration pair | Eight tables in design section 3. |
| Release guard | Infrastructure | `infrastructure/database/schema/version.go` and runner | Constants | Proposed move from 9 to 10 in `CORE-IRT-004`. |
| Live adapter | Infrastructure | `infrastructure/service/interviewvoice` | Driver adapter | Retains provider session id. Fake-server tests. |
| HTTP handler | Delivery | `delivery/http/handler/interviewhandler` | Handler | Attempt and readiness only. |
| Composition | Infrastructure | `infrastructure/runtime` | Wiring | Registers interview routes and, in `CORE-IRT-007`, drops prototype routes. |
| Error codes and translations | `pkg/errs`, `translations/errors.en.toml`, `translations/errors.id.toml` | Catalog | `IRT_CONFLICT`, `IRT_NOT_READY`. |

No seed artifact. The lifecycle implementation package is not in this inventory.

## 4. Branching / PR Strategy

One feature branch is enough if the blocked tasks stay unmerged until their ready conditions are true. Prefer one commit per task id so a blocked task is not buried in an earlier commit.

Safe order:

1. `CORE-IRT-001` and `CORE-IRT-002` together or in either order.
2. `CORE-IRT-003` after both. `CORE-IRT-004` after `CORE-IRT-001`. `CORE-IRT-006` after `CORE-IRT-002`.
3. `CORE-IRT-005` after the migration. `CORE-IRT-009` after the driver port exists.
4. Stop. `CORE-IRT-008` waits on the lifecycle. `CORE-IRT-007` is the same change as `CORE-IRT-008`.
5. `CORE-IRT-010` and `CORE-IRT-011` after the lifecycle callback exists.
6. `CORE-IRT-012` is a design revision after S1 evidence. Only then `CORE-IRT-013`, `CORE-IRT-014`, and `CORE-IRT-015`.
7. `CORE-IRT-016` after the numeric decision is written into the design.
8. `CORE-IRT-017` after the interview route exists and a sanitized live record is reviewed.

Do not open the provider session from a migration or from a domain test.

## 5. Testing Strategy

Commands from the Makefile. Planned checks are not existing evidence of these rules.

| Command | When |
|---|---|
| `gofmt` on changed Go files | Every Go task |
| `make test` | Domain and application tasks |
| `make test.architecture` | Every task that adds a Go declaration |
| `make lint` | Go tasks before the task is called done |
| `make mocks` | When a port interface changes |
| `make test.contract` | `CORE-IRT-007` / `CORE-IRT-008`, because OpenAPI changes |
| `make test.docker` or `make test.integration` with an authorized `TEST_DATABASE_URL` | `CORE-IRT-004` and `CORE-IRT-005` |
| `make test.all` | Before calling the executable slice done |

`make test.all` skips database suites when `TEST_DATABASE_URL` is absent. Report that skip. It is not schema proof. `make test.security` is appropriate once new modules are imported; the current driver should stay on `clientmanager`, which the prototype already uses.

HTTP tests live beside the handler and use a mocked operation. Domain tests use public behavior and a fixed clock. Application tests use Mockery ports. Repository tests use PostgreSQL.

### Unit tests

- Coverage edges, three omission reasons, technical loss does not omit, follow-up cap, citation required for `answered`.
- `Authorize` rejects a skipped stage, a technical loss, and a missing source handoff.
- Soft budget and the 15-minute instant use equality at the boundary.
- Reconnect counters: 120 seconds, two attempts, two episodes.
- Instruction builder omits the ceiling id `22222222-2222-4222-8222-222222222222`.
- SDP normalization matches the prototype CRLF rule, through the interview operation's public result.
- Driver maps a non-success provider response to `SESSION_UNAVAILABLE` and still returns a session id on success. Use a fake server.

### Integration tests

- Happy path: migrate 9 then 10, insert a pin, commit progress and a handoff, roll back a failed handoff, reload the prior stage.
- Unique open session, unique handoff per stage, omission reason check, zero `clarification_count` persisted.
- Error cases: duplicate idempotency key with a different intent, citation to a fragment, corrupt enum on restore.
- Lifecycle fence tests wait until the lifecycle callback exists. Until then the interview transaction test does not claim `BR-CORE-057`.

### Regression tests

- `make test` still passes task, access, and agent-configuration packages.
- Prototype voice tests stay green until `CORE-IRT-007` removes the routes and updates those tests and the OpenAPI contract together.
- Publication salary filtering stays as it is. This slice does not change `salaryCeilingConfigured`.
- Startup on a clean version 9 fails after the guard moves to 10, and startup on a clean version 10 succeeds. Cover both in the schema test.

`H-02` wording relevance, `H-08` live reconnect, and `H-01` live audio are not unit tests. They are the blocked acceptance tasks.

## 6. Rollout & Rollback

- Rollout: apply SQL to version 10 with `make migrate.up MIGRATION_VERSION=10` on an operator database, confirm clean version 10, then deploy the binary that contains the matching `ReleaseVersion`. Do not boot a version-10 binary against version 9. The new HTTP routes stay unmounted until `CORE-IRT-008`. No feature flag is required for unmounted code.
- Candidate traffic keeps using the existing access routes. The prototype voice URL remains live until `CORE-IRT-007`. Treat that window as a known provider-cost exposure, not as interview behavior.
- Monitoring for this slice uses the existing logs plus the ids in design section 7. No new tracing stack.
- Rollback of the application binary is safe only when the binary's `ReleaseVersion` matches the database. Version 10 tables may stay. Dropping them is allowed on a disposable database. A populated database keeps the tables and uses a forward fix.
- A failed version 10 migration stops rollout. Inspect dirty state. Do not auto-force. Down is not the populated-environment path.
- Provider disablement is operational: unset the provider key and the driver returns `SESSION_UNAVAILABLE` before a new attempt is stored. Attempts already stored stay subject to the lifecycle.
