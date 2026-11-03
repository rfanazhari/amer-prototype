# Tactical Strategy: Interview Runtime

## 1. Implementation Strategy

Mode C revision, 2026-11-03: name the production integration deferred from T8 as T18. The assessed design is [technical-design.md](technical-design.md) §3–§4. T1–T11 retain their recorded Done status and verification limits; their original evidence does not establish live acceptance. T18 now provides mounted runtime and real database integration proof in [T18 verification](t18-verification.md).

Implement the domain workflow, the version 10 tables, the GORM adapter, the driver transport, and the instruction builder without waiting for provider event names. Call Execution Lifecycle for activation, fencing, and completion. Do not add the five execution states to this slice. Leave report generation to Interview Reporting.

T7 already removed the prototype voice routes. T18 composes T8's operations and mounts `POST /candidate/executions/{id}/session-attempts` and the readiness route after supplying real attempt authority, command persistence and shared transactions. Keep prototype routes absent.

The live voice loop does not use RabbitMQ. GPT-Live is the driver target. Tests of `CORE-IRT-006` use a fake HTTP server. A live `H-01` pass is `CORE-IRT-017` and stays blocked.

Schema installation follows the manual migration stream. The current release floor is clean version 11. T18 implements `000011_interview_runtime_integration.{up,down}.sql`, allocation rechecked before implementation, with startup/runner/checksum/runbook updates in the same task. Startup must keep refusing to migrate. No seed artifact is required.

## 2. Code Reuse Map

Paths were read during planning. They describe current behavior.

| Component | Location | Used for | Fit and limitations |
|---|---|---|---|
| `QuestionStage`, `HRConfiguration`, `AssessHR` | `../../../../../domain/enum/questionstage.go`, `domain/valueobject/hrconfiguration.go`, `domain/agentconfig/hr.go` | Stage names, question shape, 300–600 publication total, `expectations_only`, ceiling reference id | Publication assessment is not the live workflow. No follow-up count field. Closing is optional for questions and required for a positive duration. Do not import publication findings into progress decisions. |
| `agentreference.Local` | `../../../../../infrastructure/service/agentreference/local.go` | Role name for the demo job description | Returns no salary amount and no job-description body. Instructions may include the role string only. |
| `normalizeSDP`, `Negotiate.Execute` | `application/usecase/voicesession/negotiate.go` | CRLF SDP rule | `Execute` attaches prototype instructions and hides the provider error. Interview code does not call it. |
| `realtimecall.Adapter.Exchange` | `infrastructure/service/realtimecall/adapter.go` | Live sessions URL, `gpt-live-1` default, bearer POST, SDP answer parse | Historical prototype discarded `session.id`; T7 removed its production route. Implemented `interviewvoice` retains the id. |
| `voicepage/page.html` | `delivery/http/voicepage/page.html` | Microphone, peer connection, ICE, playback | Candidate Interview embeds these mechanics. This slice does not serve the page. |
| `liveprobe.DecodeSetup`, `Shape` | `../../../../../infrastructure/service/liveprobe/probe.go` | Session-id shape and payload-free event type check | Not wired into production. `Shape` does not decide finality. |
| `StartExecution` | `../../../../../application/usecase/agentbindingusecase/start_execution.go` | Confirms the pin is assignment id plus revision id | Does not activate or fence. Do not add stages here. |
| `interview_executions` | `../../../../../migrations/pgsql/000007_interview_executions.up.sql` | FK target for progress | Columns are id, assignment, instance, revision, `created_at`. No status column to update. |
| `Store.Update` | `../../../../../infrastructure/repository/taskrepository/task.go` | Row lock, callback, rollback, explicit `Select` | Task columns and the task aggregate are unrelated. Copy the transaction shape in the interview store. |
| `Clock`, `System` | `../../../../../application/service/ports.go`, `infrastructure/service/clock/clock.go` | UTC millisecond `Now` | Reuse as-is. Stage and deadline tests inject a fixed clock. |
| Envelope, strict JSON, `errs.Technical` | `delivery/http/handler/voicesessionhandler/handler.go`, `../../../../../pkg/errs/codes.go` | Envelope and 1 MiB unknown-field decode | Handler stays private to the prototype. Interview handler owns its decode and maps `IRT_*` codes. |
| Candidate cookie and CSRF | `../../../../../delivery/http/handler/accesshandler` | 401 / 403 before interview logic | Reuse the verifier. Scope mismatch on an execution is 404. |
| Revision reader | `../../../../../application/repository/agentconfigport` | Load the pinned immutable revision | Read by the execution's `published_revision_id`. Do not read the instance's active pointer. |
| Schema runner and `ReleaseVersion` | `../../../../../infrastructure/database/schema` | Next SQL version and startup guard | Current floor 11. T18 adds the forward migration and matching guard/tooling updates. No `AutoMigrate`. |
| T8 operations and handlers | `../../../../../application/usecase/interviewusecase`, `delivery/http/handler/candidateinterviewhandler` | Attempt/readiness and transport contract | Implemented. Reconcile command reservation/finalization and compose in T18; keep existing authorization and safe errors. |
| Execution lifecycle and repository | `../../../../../application/usecase/executionusecase`, `infrastructure/repository/executionrepository` | Generic transitions and execution lock | Reuse state ownership. T18 implements RecordAttempt, persisted generation restoration and shared transaction callbacks. |
| Progress store and driver | `../../../../../infrastructure/repository/interviewrepository`, `infrastructure/service/interviewvoice` | Existing persistence and provider transport | T18 makes the store join the lifecycle transaction and wires the driver with bounded cleanup. Fake-provider tests do not establish live feasibility. |
| Architecture tests | `../../../../../test/architecture/dependencies_test.go`, `documentation_test.go` | Import direction and comments on every new declaration | Domain stays inside the allowed standard-library set. |
| Mockery | `../../../../../.mockery.yml`, `make mocks` | Generate port mocks | Register new interfaces. Do not hand-write mocks. |
| Task event queue and RabbitMQ compose | `../../../../../infrastructure/service/eventqueue`, `make docker.rabbitmq` | Unrelated | Not in the interview graph. |

`../../../../../test/livevoice` is a manual probe. A passing `TestManualProbe` is not `H-01` and not Q1.

## 3. Net-New Code Inventory

Inventory retains the original feature components and adds T18 reconciliation. Names may be refined inside the design's refinement boundary; task status and reuse rows distinguish existing code from remaining work. Classification is by responsibility.

| Item | Layer / component | Package / path | Type | Ownership / notes |
|---|---|---|---|---|
| `Progress` | Domain | `../../../../../domain/interview` | Aggregate root | One per execution. Owns stage cursor, cap, and version. |
| Coverage item | Domain | `../../../../../domain/interview` | Entity inside Progress | States pending, asked, answered, omitted. |
| Technical loss | Domain | `../../../../../domain/interview` | Entity inside Progress | Blocks authorization. Separate from omission. |
| Transcript entry | Domain | `../../../../../domain/interview` | Entity inside Progress | Fragment versus final. Revision supersedes. |
| Handoff | Domain | `../../../../../domain/interview` | Entity inside Progress | One per source stage. Closing has a null target. |
| Checkpoint | Domain | `../../../../../domain/interview` | Entity inside Progress | Latest wins for recovery context. |
| Voice session | Domain | `../../../../../domain/interview` | Entity inside Progress | One open session per execution. |
| Omission reason, evidence state, speaker | Domain | `../../../../../domain/interview` | Value / enum behavior | Owned with the entity that stores them. A separate type is optional. |
| `NewProgress` / `RestoreProgress` | Domain | `../../../../../domain/interview` | Factory functions | Create applies the frozen cap. Restore does not re-run publication rules. |
| `Authorize`, coverage methods, `ReadyForCompletion` | Domain | `../../../../../domain/interview` | Aggregate behavior | Predicates stay methods. No domain service. |
| Interview operations | Application | `../../../../../application/usecase/interviewusecase` | Operation interfaces and structs | Activate, ready, propose, reconnect, complete-or-fail. |
| Instruction builder | Application | `../../../../../application/usecase/interviewusecase` | Function on the operation | Drops salary id and amount. |
| Progress port | Application | `../../../../../application/repository/interviewport` | Repository contract | Commit callback, no GORM handle. |
| Lifecycle port | Application | `../../../../../application/service` | Port | Activate, fence, locked callback, complete, fail. |
| Driver port | Application | `../../../../../application/service` | Port | Create, attach, observe, supply context, control, close. |
| DTOs | Application | `../../../../../application/dto` | Detached structs | Candidate-safe read model. |
| GORM models and store | Infrastructure | `../../../../../infrastructure/repository/interviewrepository` | Adapter | Explicit `Select`. JSON is not used for coverage state. |
| Version 10 SQL | Infrastructure | `migrations/pgsql/000010_interview_runtime.{up,down}.sql` | Migration pair | Eight tables in design section 3. |
| Release guard | Infrastructure | `../../../../../infrastructure/database/schema/version.go` and runner | Constants | T4 moved the floor to 10; T18 raises it to 11. |
| Live adapter | Infrastructure | `../../../../../infrastructure/service/interviewvoice` | Driver adapter | Retains provider session id. Fake-server tests. |
| HTTP handler | Delivery | `delivery/http/handler/candidateinterviewhandler` | Handler | T8 implemented attempt/readiness; T18 composes it. |
| Lifecycle adapter and attempt support | Application / infrastructure | `application/usecase/executionusecase`, `application/repository/executionport`, `infrastructure/repository/executionrepository`, proposed `infrastructure/service/interviewlifecycle` | Operations, ports and adapter | T18 supplies durable attempt/generation authority and a shared transaction callback. No HR fields in the kernel. |
| Command store, revision and pin readers | Infrastructure | `../../../../../infrastructure/repository/interviewrepository` | Adapters | T18 implements `SessionCommands`, `RevisionReader`, `PinReader` and reservation/finalization reconciliation through application ports. |
| Version 11 integration SQL | Infrastructure | proposed `migrations/pgsql/000011_interview_runtime_integration.{up,down}.sql` | Migration pair | T18; exact replay states, nullability and retained-row handling are settled at the design checkpoint before SQL. No seeds. |
| Composition | Infrastructure | `../../../../../infrastructure/runtime` | Wiring | T18 populates Attempt/Ready and mounts their exact paths. T7's prototype removal remains in place. |
| Error codes and translations | `../../../../../pkg/errs`, `translations/errors.en.toml`, `translations/errors.id.toml` | Catalog | `IRT_CONFLICT`, `IRT_NOT_READY`. |

No seed artifact. T18 includes only the generic lifecycle authority support needed by the interview adapter; broader kernel changes are outside scope.

## 4. Branching / PR Strategy

One feature branch is enough if the blocked tasks stay unmerged until their ready conditions are true. Prefer one commit per task id so a blocked task is not buried in an earlier commit.

Remaining safe order:

1. Preserve T1–T11 Done. Start T18 with the integration design checkpoint in design §3–§4, then implement authority/transaction support, migration/stores/readers, application reconciliation and runtime mounting, followed by fake-provider/PostgreSQL proof.
2. Investigate S1 evidence for T12 and obtain the Q2 numeric decision independently. Event counts are insufficient; the finality map still requires reviewed observations and fixture identities/order/corrections.
3. After T18, execute T17 on the mounted route; review its sanitized note to finish the task. T17 does not require that note before starting and does not resolve Q1 by itself.
4. After T12 and T18, implement T13, then T14, then T15 with the real fence and committed checkpoint. T18's fake-provider reconnect checks do not close T15's deliberate live proof.
5. T16 is ready: Q2 is decided (2 × 30 seconds within 90 seconds per stage, no finalization reserve). Mocked application work can follow that decision; production exhaustion proof uses T18's lifecycle adapter. No post-handoff retry number is invented in T18.

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
| `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint` | T18 application, mounting and persistence reconciliation |
| `make test.docker` or `make test.integration` with an authorized disposable `TEST_DATABASE_URL` | T18 shared-lock/rollback, concurrent replay, mounted HTTP and generation reload proof; record migration-specific variables as applicable |

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
- T18 uses the real lifecycle adapter and fake provider: same key replays the original answer without a second provider create; changed intent conflicts; concurrent calls and interrupted setup preserve the durable reservation. Failed result persistence does not publish a successful replay. Cover pending/failed state behavior selected at the design checkpoint.
- T18 proves stale version, replaced attempt, terminal execution and callback failure write nothing; attempt authority survives store reload. Include cancellation versus progress and T10 handoff/close rollback with the actual shared transaction.
- Production HTTP exercises cookie/CSRF, wrong assignment 404, current controller, retained provider identity and readiness without coverage changes. No live account is used by these tests.

### Regression tests

- `make test` still passes task, access, and agent-configuration packages.
- T7's prototype routes stay absent; T18 updates router tests to expect the mounted attempt/readiness routes instead of access fall-through. Keep the T8 public contract, endpoint documentation and changelog consistent with runtime availability.
- Publication salary filtering stays as it is. This slice does not change `salaryCeilingConfigured`.
- Startup refuses clean versions 9 and 10 and accepts clean 11. The version-11 migration proof covers fresh install, retained version-10 rows and disposable down.

`H-02` wording relevance, `H-08` live reconnect, and `H-01` live audio are not unit tests. They are the blocked acceptance tasks.

## 6. Rollout & Rollback

- Current release requires clean version 11 and mounts the new HTTP routes. Operator tooling performs the allocated migration explicitly before binary startup; no runtime migration is introduced. T18 verifies fresh install and populated upgrade from 10 on disposable PostgreSQL, including retained legacy commands without invented SDP.
- Candidate traffic keeps using the existing access routes. Prototype voice URLs are already removed by T7 and stay absent. T18 provides composition proof with a fake provider; it makes no live-session acceptance claim.
- Monitoring for this slice uses the existing logs plus the ids in design section 7. No new tracing stack.
- Rollback of the application binary is safe only when the binary's `ReleaseVersion` matches the database. Version 10 tables may stay. Dropping them is allowed on a disposable database. A populated database keeps the tables and uses a forward fix.
- A failed version 10 migration stops rollout. Inspect dirty state. Do not auto-force. Down is not the populated-environment path.
- Provider disablement is operational: unset the provider key and the driver returns `SESSION_UNAVAILABLE` before a new attempt is stored. Attempts already stored stay subject to the lifecycle.

## 7. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v2 | 2026-11-03 | Add T18 integration inventory, implementation order, migration/replay assessment and real-fence proof | Name T8's deferred follow-up while preserving completed task scopes and Q1/Q2 gates |
| v3 | 2026-11-03 | Record T18 implementation and release floor 11 | Real transaction/HTTP/migration proof resolves integration Q3; Q1/Q2 and live acceptance remain open |
