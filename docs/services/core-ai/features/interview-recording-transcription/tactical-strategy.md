# Tactical Strategy: Interview Recording and Transcription

## 1. Implementation Strategy

Mode B: extend implemented Interview Runtime with a separate post-completion processing lifecycle. Follow the assessed [technical design](technical-design.md); this document is a plan, not implementation evidence. The owner selected asynchronous backend-only download and replaced G14's primary source. No HR download route is planned.

Resolve provider feasibility and data policy before activating storage. Implement durable intent, claim/checkpoint/publication and provider adapters before runtime composition. Preserve report T13/T14/T15 ownership, active progress fencing, candidate contracts, and Task's process-local queue. No commit, push, provisioning, live API invocation or migration execution is authorized by document generation alone.

## 2. Code Reuse Map

| Component | Location from repository root | Fit and limitations |
| --- | --- | --- |
| Live creation/sideband | `infrastructure/service/interviewvoice` | Retain backend credentials and provider IDs. Add startup storage and one-owner close confirmation; current send-and-release does not prove recording readiness. |
| Completion | `application/usecase/interviewusecase/complete_stage.go`, `infrastructure/service/interviewlifecycle/lifecycle.go` | Retain terminal transition, lock and transaction context. Add durable intent in the same transaction; move post-completion provider finalization off the candidate request. |
| Session and canonical entry rows | `infrastructure/repository/interviewrepository/models.go`, `migrations/pgsql/000010_interview_runtime.up.sql` | Existing provenance/body storage. Do not reuse stage-to-latest-session append mapping for historical audio import. |
| Trusted time / transaction context | `application/service.Clock`, `infrastructure/database/transaction` | Reuse UTC milliseconds and hidden transaction joining; maintain execution-before-child lock order. |
| Broker delivery patterns | `infrastructure/service/reportbroker`, `interviewreportrepository/claim.go` | Reuse confirm/ACK/fence concepts. The report-only outbox and consumer cannot carry a new job kind; use separate adapters/tables. |
| Report evidence interface | `application/repository/interviewreportport/evidence.go` | Future T14 consumer of final evidence; current entry DTO has no body and requires a settled handoff before writer T13. |
| Migration runner | `cmd/migrate`, `infrastructure/database/schema` | Current clean floor 13; propose next version 14 after allocation check. No AutoMigrate. |
| Candidate completion HTTP/page | `delivery/http/handler/candidateinterviewhandler`, `delivery/http/candidateinterview` | Keep ordinary completed response and existing auth/error precedence. No audio/transcript field or new action. |

## 3. Net-New Code Inventory

| Item | Layer / proposed path | Classification and ownership |
| --- | --- | --- |
| Transcription | `domain/aggregate/interviewtranscription` | Aggregate rooted at execution; owns completeness and published revision |
| Recording source | same domain package | Entity owned by Transcription; session/attempt provenance |
| Segment identity and timing | same domain package | Value owned by evidence revision; ordering and dedup rules |
| Admission, worker, reconciliation | `application/usecase/interviewtranscriptionusecase` | Operations coordinating ports; no provider schemas or ORM |
| Store and provider interfaces | `application/repository/interviewtranscriptionport`, `application/service` | Contracts for claims, streams, checkpoints and immutable final reads |
| Recording and transcription adapters | `infrastructure/service/interviewrecording`, `infrastructure/service/audiotranscription` | Provider-specific binary/multipart handling; channel/chunk preparation |
| GORM persistence | `infrastructure/repository/interviewtranscriptionrepository` | Metadata, dedicated terminal-safe canonical insert and atomic publication |
| Dedicated broker adapter | `infrastructure/service/transcriptionbroker` | Q3-gated delivery; no report topology mutation |
| Migration pair | `migrations/pgsql/000014_interview_recording_transcription.{up,down}.sql` (proposed) | Five new metadata/outbox tables plus nullable recording provenance column; canonical body table retained |
| Release artifacts | schema checksums/grants/guard, migration CLI limits, isolated scripts, runbook | Update together at implementation; preserve migrations 1–13 |
| Mocks | `.mockery.yml`, `test/mocks/` | Generator-owned output through `make mocks` |
| Runtime wiring | `infrastructure/runtime` | Lifecycle-owned workers and shutdown; enable only after gates |

No seed or master/reference dataset is required. New configuration is environment-driven and must be documented without credential values. Storage location and retention are Q2 decisions, not inferred from an existing cloud dependency.

## 4. Branching / PR Strategy

Recommend one feature branch with reviewable slices: feasibility/policy synchronization, domain/ports, schema/store, provider adapters, worker/broker, runtime and real proof. Phase commits follow the executor's RED/GREEN workflow when implementation is requested. No Git operation is part of this planning change.

## 5. Testing Strategy

All checks below are planned. Use table-driven public-behavior tests, testify, Mockery for interfaces and sqlmock for SQL. Repository documentation rules apply to tests/helpers. Mocks cannot establish PostgreSQL atomicity, provider storage or actual audio transcription.

### Unit tests

- COMPLETED admission only, fixed complete session manifest, legacy null recording flag, no candidate-omission inference, no live progress mutation.
- Duplicate jobs/chunks, stale lease, clock equality at eligibility/expiry, persisted retry counts, no partial final publication.
- Stereo split, chunk size limits, overlap dedup and chronological merge with deterministic synthetic WAV fixtures. Timing/response fixtures must derive from verified T1 contracts.
- Fake HTTP provider: recording pending/permanent failure, malformed/truncated WAV, oversized stream, timeout/cancellation, multipart format, invalid/empty transcription and secret/body-log exclusion.
- Existing completion response succeeds while a provider stub remains blocked; handler operation interfaces remain mocked.

### Integration and live tests

- Disposable PostgreSQL: migration up/down/up, recording flag null on legacy sessions, source/session identity rejection, uniqueness, concurrent claims, complete/admit/outbox rollback, atomic entry revision publication and expired-worker refusal.
- Restart between completion and dispatch, after download and after transcription but before commit; committed checkpoints survive without duplicated entries.
- RabbitMQ: confirms, duplicate delivery, worker crash before ACK, broker outage and database error without ACK. Use real RabbitMQ/PostgreSQL for these assertions; isolate new topology from report queues.
- Opt-in OpenAI synthetic session: storage enabled, graceful recording close, actual stereo WAV, both-speaker transcription and missing/reconnect session coverage. Record sanitized metadata/results, never keys/audio/text in logs. Account eligibility is a prerequisite, not a mock assertion.

### Regression and commands

Run focused package tests first, then `make test`, `make test.all`, `make test.architecture`, `make lint`, `make test.security`, and `make test.contract` for candidate contract preservation. `make check` owns the combined gate. PostgreSQL uses `make test.integration` with an explicitly authorized `TEST_DATABASE_URL` or `make test.docker`; skipped suites remain skipped. Add a dedicated isolated transcription-broker/live verification target during T11 rather than pretending the existing report-only target covers it.

## 6. Rollout & Rollback

Q1/Q2/Q3/Q4 must be settled before production enablement. Prepare an explicit notice/eligibility cutoff, approved recording/staging storage and cleanup. Apply the newly allocated manual migration, deploy guarded workers, and enable recording for new eligible sessions only. Recovery must not bulk-process historical sessions. Observe safe identifiers/state/latencies and cleanup failures; no audio/transcript bodies or signed URLs.

Stop intake and cancel worker I/O within the existing process budget on shutdown. Jobs/outbox/checkpoints remain for restart. Disabling the feature stops new recording admission and preserves already stored evidence; cleanup follows policy. Keep the upgraded schema during rollback; do not delete canonical transcript rows or shared volumes. Test disposable down migration separately from production recovery.
