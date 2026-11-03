# Core AI

Core AI is one Go modular monolith. Status below is taken from each feature's task plan (the status authority) and was brought up to date on 2026-11-03; the cross-feature record is [baseline.md](../baseline.md) and the criterion mapping is [traceability-matrix.md](../../traceability-matrix.md).

Startup requires clean schema version 13 (`schema.ReleaseVersion`) and never runs migrations; see the root `README.md` and `docs/design/database-migrations.md`. The candidate page is in scope. There is no admin or HR user interface in this phase. Document Analyzer configuration and publication are available; runnable invoice analysis is outside this interview delivery phase, and Execution Lifecycle T11 remains Todo on that later capability (see [Document Analyzer and T11](#document-analyzer-and-execution-lifecycle-t11)).

## Status at a glance

| Feature | Task plan | State |
|---|---|---|
| [Identity and Access](features/identity-access/task-plan.md) | T1–T16, T18–T23 Done; T17 in progress | Delivered; migration tooling task open |
| [Agent Configuration and Publication](features/agent-configuration-publication/task-plan.md) | T1–T14 Done | Delivered |
| [Execution Lifecycle](features/execution-lifecycle/task-plan.md) | T1–T10 Done; T11 Todo | Delivered for interview; second-agent proof open |
| [Candidate Interview](features/candidate-interview/task-plan.md) | T1–T13 Done; T14–T15 Todo (added 2026-11-04) | Delivered; per-stage extension planned |
| [Interview Runtime](features/interview-runtime/task-plan.md) | T1–T11, T17–T20, T22 Done; T21 in progress; T12–T15 blocked; T16, T23–T25 Todo | Partial |
| [Interview Reporting](features/interview-reporting/task-plan.md) | T1–T12 Done; T13–T15 Todo | Partial: no report can be produced; HR routes not mounted |
| [Interview Recording and Transcription](features/interview-recording-transcription/task-plan.md) | T1 in progress; T2–T12 Todo | Planned; no pipeline delivered |

Done means the task plan marks the task `[x]`. It does not mean a formal acceptance pass: the database suites run only with an authorized `TEST_DATABASE_URL`, and several tasks record the full suite and `make test.docker` as not run.

## Done

### Identity and Access

ADMIN provisions HR, and an invited candidate reaches one assignment without a permanent account. Rules `BR-CORE-020`–`BR-CORE-039`: [Identity and Access](business/capabilities/identity-access.md). Tripack: [identity-access](features/identity-access/task-plan.md).

The task plan records T1–T16 and T18–T23 Done: administrative and candidate access routes, administrative-session cleanup (T15), the legacy schema baseline (T16), replacement of AutoMigrate by explicit migrations with adoption proof (T18–T20), the creator-scoped candidate list (T21), HR revocation that cancels a nonterminal execution (T22, `CORE-IAM-022`), and replacement of earlier administrative sessions at a new login (T23). T17 (versioned baseline SQL and migration tooling) is still marked in progress: its final GREEN is blocked by failing full Docker regressions outside that task. Candidate-session expiry during an interview driving cancellation is open question Q7 and is not implemented.

### Agent Configuration and Publication

HR configures and publishes the HR Interviewer and Document Analyzer templates, then pins an interview assignment to the current published revision. Rules `BR-CORE-001`–`BR-CORE-014`: [Agent Configuration and Publication](business/capabilities/agent-configuration-publication.md). Tripack: [agent-configuration-publication](features/agent-configuration-publication/task-plan.md).

T1–T14 are recorded Done, including the published configuration routes, instance list, and assignment create. There is no admin UI, so the non-developer configuration journey is not met.

### Execution Lifecycle

One admitted run keeps a stable identity and an honest ending; a repeated start finds the same run. Rules `BR-CORE-050`–`BR-CORE-059`: [Execution Lifecycle](business/capabilities/execution-lifecycle.md). Tripack: [execution-lifecycle](features/execution-lifecycle/task-plan.md).

T1–T10 are Done: migration `000008_execution_lifecycle`, the GORM execution store, start, activation, fencing and terminal commits, `Cancel` and `CancelNonterminal`, PostgreSQL proof of locking, rollback and idempotency (T8), the architecture, contract and lint gates (T9), and HR-revocation cancellation (T10, delivered through Identity and Access T22). There are no direct HTTP routes; Candidate Interview, Interview Runtime and Identity and Access consume the lifecycle. T11 (`CORE-EXL-011`) is Todo and needs a Document Analyzer capability.

### Candidate Interview

Rules `BR-CORE-040`–`BR-CORE-047`: [Candidate Interview](business/capabilities/candidate-interview.md). Source of status: [task plan](features/candidate-interview/task-plan.md). T1–T13 (`CORE-CIN-001`–`013`) are Done.

- **Notice, consent and storage (T1–T5):** approved notice `notice-261102`, explicit acceptance before start, acceptance replay, consent/coverage/controller domain rules and application ports. Migration `000009_candidate_interview` adds `interview_consents` and `interview_controllers`.
- **Candidate API and controller ownership (T6–T9):** interview read, consent acceptance, execution start/poll/takeover/cancel, strict input/session/origin/CSRF checks, safe responses, public errors, OpenAPI and endpoint documentation. Controller takeover persists one versioned controller per assignment and rejects stale updates.
- **Candidate screen (T10):** notice and accept/start actions, per-tab identity and retry intent in `sessionStorage`, read-only/takeover behavior, confirmed cancel, polling, and distinct `COMPLETED`, `FAILED`, and `CANCELLED` outcomes. Refresh only reads; the screen does not wait for an HR report.
- **Real lifecycle integration (T11):** production start/read/cancel adapters, one execution after concurrent starts or a lost 201, conflict on changed intent, a stable deadline from persisted activation plus 15 minutes, committed cancellation followed by `CANCELLED` on GET, completed-execution replay, and refusal after assignment revocation.
- **Candidate-safe progress and the combined slice (T12–T13):** candidate-safe progress with controller fencing, and the combined PostgreSQL slice proof (`TestCandidateInterviewSlicePostgres`).

Runtime-owned candidate routes (`session-attempts`, `ready`, `complete`) are documented with Interview Runtime; see the [baseline](../baseline.md#9-f5--candidate-interview-core-ai) for the route list.

Verification limits recorded in the task plan: T11 passed `make test`, `make check` and the filtered PostgreSQL integration test; the full database suite and `make test.docker` were not run for T11 or T13. The T10 and T13 browser checks used stand-in or fixture session endpoints. Earlier T4/T5/T8 runs recorded fixture/schema interference in the full integration suite; that limitation has not been re-verified here.

## In progress and not yet delivered

1. **Interview Runtime (partial).** The live interview asks the published questions, records gaps, advances only on a committed handoff, and finishes only when the required record is stored. Rules `BR-CORE-060`–`BR-CORE-071`: [Interview Runtime](business/capabilities/interview-runtime.md). Tripack: [interview-runtime](features/interview-runtime/task-plan.md); open gaps in [todo-gap](features/interview-runtime/todo-gap.md).
   - Done: progress rules and orchestration (T1–T3), migrations `000010` and `000011` (T4–T5, T18), the GPT-Live driver (T6), removal of the prototype voice routes (T7), session-attempt and readiness HTTP (T8), stage instructions without the salary ceiling (T9, T20), handoff and completion (T10–T11), one live interview-path session recorded (T17), and starting the voice session from the candidate page (T19).
   - In progress: T21, letting the model end the interview stage by stage. The application part and the candidate "Selesai" finish control are done; the model-triggered ending is blocked on a live spike and owner decision G9.
   - Blocked or Todo: T12–T15 (event-finality map, canonical transcript ingest, coverage from finalized speech, one deliberate reconnect) are blocked on Q1; T16 (post-handoff activation budget) is Todo.
   - Owner decisions 2026-11-04: [D10](features/interview-runtime/decision-log.md) one provider session per stage, and [D11](features/interview-runtime/decision-log.md) an interim model-stated stage seal labelled in reports. T22 (regenerate the tripack) is Done as documentation only; T23 (seal without advance, new session per stage), T24 (driver waits for close confirmation; decision D12 accepted, timeout value not set) and T25 (first-session instructions) are Todo. Also accepted that day: [D12](features/interview-runtime/decision-log.md) (bounded close-confirmation wait), D13 (transition signal by polling) and D14 (formal register, paraphrasable greeting).
   - Practical effect: a candidate can start a voice session and finish the interview, but there is no transcript ingest, no coverage update from speech and no automatic stage progression yet.
2. **Interview Reporting (partial).** HR receives a written account of a finished interview that does not hire, reject, or rank the candidate; a report failure must not change a completed interview. HR API only, no admin screen. Rules `BR-CORE-080`–`BR-CORE-086`: [Interview Reporting](business/capabilities/interview-reporting.md). Tripack: [interview-reporting](features/interview-reporting/task-plan.md).
   - Done: report lifecycle and statement rules, migrations `000012` and `000013`, admission atomicity and failure isolation, authorization, read and regeneration operations, the two HR routes and published contract (not mounted), confirmed outbox publication, claim attempts inside the accepted budget, RabbitMQ behavior proof, and malformed-message isolation (T1–T12).
   - Todo: the report writer client (T13, blocked on model selection), the finalized-evidence read (T14, blocked on a populated read model), and registration of routes, admission, publisher and consumer in the runtime (T15). No report can be produced today.
3. **Interview Recording and Transcription (planned).** After an interview commits `COMPLETED`, the backend asynchronously obtains stored recordings, transcribes them and publishes finalized evidence for reporting. Tripack: [interview-recording-transcription](features/interview-recording-transcription/task-plan.md). T1 (provider feasibility) is in progress with a real recording download and per-channel transcription recorded; timed alignment is unproven and T2–T11 are Todo. T12 (live-event spike) was added on 2026-11-04. The per-stage proposal's D5 (live sideband events as the primary transcript source) is proposed, not accepted, and waits for that spike: [proposal](features/interview-recording-transcription/proposal-per-stage-live-transcript.md).
4. **Identity and Access T17** (migration tooling final GREEN) and **Execution Lifecycle T11** (Document Analyzer proof) remain open as described above and below.

## Document Analyzer and Execution Lifecycle T11

[T11 — Run a non-interview agent on this kernel](features/execution-lifecycle/task-plan.md#follow-up--deferred), ticket `CORE-EXL-011`, proves that the shared kernel supports a second agent without interview-specific storage or invoice rules. It remains Todo/blocked; the current Execution Lifecycle tripack does not authorize implementing the Document Analyzer product.

The platform baseline already defines the business flow. [Technical design §4](../../design/core-ai/technical-design.md#4-configuration-and-publishing) selects the active published revision on new Document initiation and preserves the original binding on duplicate initiation. [§8](../../design/core-ai/technical-design.md#8-document-integration) accepts one immutable PNG/JPEG invoice, at most 5 MiB and 4096 pixels per side, and extracts `invoice_number`, `vendor`, `date`, `total`, and `currency`. The published configuration selects the required subset. All required fields valid permits `COMPLETED`; at most two attempts share the same input/binding, and exhaustion produces `FAILED` with a permitted committed diagnostic result. Missing or ambiguous values stay explicit. [§6](../../design/core-ai/technical-design.md#6-orchestration-rabbitmq-and-runtime) defines durable job/outbox dispatch through RabbitMQ and a guarded result/terminal commit.

Agent Configuration and Publication already supplies the Document template, required-field configuration, validation and immutable published revisions. That foundation does not implement invoice admission, artifact handling, extraction, result persistence or worker orchestration. The [ACP requirement gap](../../design/core-ai/acp-requirement-gap.md) identifies those Document gaps; its older kernel/interview status statements are historical, not the current task status.

To make T11 executable, a Document Analyzer capability design must turn the accepted baseline into a scoped implementation: define revision selection/serialization at admission, call `Admit`, and specify result/progress storage plus the progress port used to authorize completion. The [platform readiness checklist](../../design/core-ai/technical-design.md#14-implementation-dependency-map-and-readiness-checklist) still leaves model/provider contracts and queue details open; those decisions must be resolved before their dependent implementation. This is a missing implementation slice, not a missing product concept or an unfinished T8/T9 kernel prerequisite.

T11 closes only when that capability's own tests show an admitted execution reaching a terminal state on `agent_executions` with no `interview_executions` pin and no kernel changes adding invoice fields. Document workflow owns extraction, field validation and result persistence; the kernel owns identity, binding, fencing and lifecycle commits. T8's existing `Admit` fixture demonstrates kernel behavior but does not replace the runnable Document proof or close T11. No T11 implementation, test pass or completion is claimed here.

## Suggested next work

Open work, in the order the task plans imply:

1. Interview Runtime: wire the stage-end trigger and finish T21 (needs the live spike and decision G9); T12–T15 wait on Q1 (provider event finality). T16 can proceed independently.
2. Interview Recording and Transcription: close T1 (alignment, close observation, readiness classification) before the dependent contracts and schema.
3. Interview Reporting: the writer client (model selection), the finalized-evidence adapter (needs a populated evidence source), then route and worker registration (T13–T15).
4. Identity and Access: T17 once the Docker regressions it depends on are resolved; Q7 for session expiry during an interview.
5. Execution Lifecycle T11 after a Document Analyzer capability design exists.

Schema changes continue the manual versioned sequence after `000013`. Choose the next number from `migrations/pgsql/` at implementation time and update the startup floor, root `README.md` and migration runbook together; planned numbers in older documents are not authoritative.

## Progress history

Status per feature, taken from each tripack's task plan. Newest entry first. Update this table when a task plan changes status. Entries dated 2026-11-03 record the status verified from the task plans and the baseline on that date; they are not the dates on which each task was completed.

| Date | Feature | Progress | Open items |
|---|---|---|---|
| 2026-11-04 | Interview runtime | Owner decisions D10 (one provider session per stage) and D11 (interim model-stated stage seal). T22–T25 added; T22 Done (tripack regenerated, documentation only); T21 rescope note. | T23–T25 not implemented. Decisions D12 (close-confirmation wait), D13 (polling signal) and D14 (formal register, paraphrasable greeting) accepted; T24 still needs a timeout value. |
| 2026-11-04 | Candidate interview | T14–T15 added (next-stage session request and transition state; page connection per stage), Todo. T1–T13 unchanged. | Depend on Interview Runtime T23/T24; decision D13 (polling signal) accepted. |
| 2026-11-04 | Interview recording and transcription | T12 added (live-event spike), Todo; D5 proposed. | Needs owner authorization for provider calls and cost. |
| 2026-11-03 | Interview recording and transcription | Planned as `CORE-IRTREC-001`–`011`. T1 in progress: provider-create sends `session.store: true` (RED `ccde0cc`, GREEN `e9e1adb`); a real stored recording downloaded as stereo WAV and per-channel transcription returned text, with timestamp-capable output from a model scheduled for shutdown on 2027-02-26. | T1 alignment, close observation and readiness classification; T2–T11 Todo. No complete verification pass is claimed. |
| 2026-11-03 | Interview reporting | T1–T12 Done (`CORE-IRP-001`–`012`), including outbox, worker budget (migration `000013`), RabbitMQ behavior proof and malformed-message isolation. | T13 writer client, T14 finalized-evidence read, T15 runtime registration. HR routes not mounted. |
| 2026-11-03 | Interview runtime | T1–T11 and T17–T20 Done, including the GPT-Live driver, session-attempt and readiness HTTP, candidate-page voice start, and one live session recorded. T21 application part and candidate "Selesai" done. | T21 model-triggered ending; T12–T15 blocked on Q1; T16 Todo; open gaps G1–G14 in todo-gap. |
| 2026-11-03 | Identity and access | T15, T21, T22 and T23 Done on top of the earlier work; AutoMigrate replaced by explicit migrations (T18–T20). | T17 in progress (Docker regressions); retention beyond administrative sessions has no policy; Q7. |
| 2026-11-03 | Candidate interview | T12–T13 Done: candidate-safe progress with controller fencing and the combined PostgreSQL slice proof. | Full database suite and `make test.docker` not run for T11/T13. |
| 2026-11-03 | Execution lifecycle | T10 Done through Identity and Access T22. | T11 waits on a Document Analyzer capability. |
| 2026-11-02 | Execution lifecycle | T1–T9 Done (`CORE-EXL-001`–`009`). Includes migration `000008`, the GORM execution store, start, activation, terminal commits, `Cancel` and `CancelNonterminal`, PostgreSQL proof of locking, rollback and idempotency (T8), and the architecture, contract and lint gates (T9). Latest checkpoints: T9 RED `b0b1c91`, T9 GREEN `941e818`. | T10 and T11 open at that date. |
| 2026-11-02 | Candidate interview | T1–T11 Done (`CORE-CIN-001`–`011`): approved notice, consent, migration `000009`, APIs, controller takeover, candidate UI and real lifecycle start/read/cancel. T11 unit/check gates and filtered PostgreSQL integration pass; RED `2acb6bb`, GREEN `775e3ff`. | T12–T13 open at that date. |
| 2026-11-02 | Interview runtime | Planned. Nothing implemented (historical; superseded by the 2026-11-03 entry). | |
| 2026-11-02 | Interview reporting | Planned. Nothing implemented (historical; superseded by the 2026-11-03 entry). | |
| 2026-11-02 | Agent configuration and publication | T1–T14 Done. | None recorded here. |
| 2026-11-02 | Identity and access | T1–T14, T16, T18–T20 Done at that date; T22 planned. | T15 retention, T17 and T21 open at that date. |
