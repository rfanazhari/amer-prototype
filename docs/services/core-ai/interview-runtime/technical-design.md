# Technical Design: Interview Runtime

## 1. Overview

Mode A. This slice conducts the live HR interview on an execution that [Execution Lifecycle](../business/capabilities/execution-lifecycle.md) owns (`BR-CORE-050`..`BR-CORE-059`). The capability authority is [Interview Runtime](../business/capabilities/interview-runtime.md), rules `BR-CORE-060`..`BR-CORE-071`.

Those rules are **planned**. Nothing in this document claims they are enforced. The current `interview_executions` row is an assignment pin. It has no lifecycle state, attempt, transcript, coverage, handoff, or voice session.

Date: 2026-11-02. Status: design for implementation planning. Provider event finality stays **deferred**.

### Goals

- Run the bound published HR revision: stages `background`, `screening`, `compensation`, `closing`; that revision's stage seconds; its mandatory questions; compensation mode `expectations_only`.
- Let the voice model propose a stage change. Let the interview workflow authorize it.
- Commit the source handoff before any next-stage voice session.
- Keep one voice session for each activated stage, and replace that session after a fault without creating another interview.
- Resume the same execution after a lost connection, from the last committed checkpoint.
- Store coverage as pending, asked, answered, or omitted. Treat lost audio as a technical failure.
- Keep the salary ceiling on the server, outside model context and candidate-visible payloads.
- Complete only after the required record is stored, by calling the lifecycle. Leave the execution available for [Interview Reporting](../business/capabilities/interview-reporting.md).

### Non-Goals

- The five execution states, admission, and terminal reopen rules. Those stay with Execution Lifecycle.
- Candidate notice, consent, page, takeover, and early stop. Those stay with [Candidate Interview](../business/capabilities/candidate-interview.md). This slice supplies the session-attempt and readiness behavior that page will call.
- An admin UI, a Document Analyzer path, and the HR report body.
- RabbitMQ, an outbox, or any broker hop inside the live voice loop.
- A hire, reject, rank, or competence score.
- Recovery of every realtime failure. Acceptance evidence is one deliberate disconnect and reconnect (`H-08`).

### Verified context and authority

Project instructions: root [AGENTS.md](../../../../AGENTS.md). No root `CLAUDE.md` is present. Go backend layout also follows the generator baseline in `.agents/skills/engineering-tripack-generator/references/backend-service-standard.md`, where it does not conflict with AGENTS.md or this repository. Observed handler tests and `test/architecture/documentation_test.go` override the baseline sentence that says delivery has no tests: new exported and unexported Go declarations need comments, and HTTP behavior is tested through mocked operation interfaces.

Precedence used here:

1. `BR-CORE-060`..`BR-CORE-071` and the user decisions for this slice.
2. [Technical Design v0.1](../../../design/core-ai/technical-design.md), which wins on budgets and ordering where it is explicit. Section 10 supplies the activation, reconnect, and 15-minute limits. Section 7's three-row timing table is showcase guidance.
3. [Scope freeze](../../../design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) sections 6.2 and 7.2 (`H-01`..`H-09`, `H-11`, `H-12`).
4. [Architecture design](../../../design/core-ai/shared_ai_agent_platform_architecture_design_v0.1.md) section 10, for stage meaning, handoff order, and the report split.
5. Verified code, as current behavior. It is not proof that the voice prototype is the interview runtime.

Module `github.com/rfanazhari/amer-prototype`. Go module in `go.mod`. PostgreSQL. GORM `v1.25.12` for CRUD. `github.com/golang-migrate/migrate/v4` `v4.19.0` for SQL files. REST JSON envelope, strict unknown-field rejection, English and Indonesian messages in `translations/errors.*.toml`. Trusted time is `application/service.Clock`; `infrastructure/service/clock.System.Now` returns UTC truncated to milliseconds.

Four root layers stay in place: `domain/`, `application/`, `delivery/`, `infrastructure/`. `cmd/app/` remains process configuration. Domain may import only the standard libraries allowed by `test/architecture/dependencies_test.go` and other `domain/` packages. Delivery does not import infrastructure. Only `infrastructure/runtime/` may import delivery.

`schema.ReleaseVersion` in `infrastructure/database/schema/version.go` is **7**. The newest applied pair is `migrations/pgsql/000007_interview_executions.{up,down}.sql`. Startup accepts a clean version 7 and does not run `AutoMigrate`. This slice does not take `000008`. Its pair is `000010_interview_runtime`, after execution-lifecycle `000008` and candidate-interview `000009`.

### What the published revision actually stores

Verified in `domain/enum/questionstage.go`, `domain/valueobject/hrconfiguration.go`, and `domain/agentconfig/hr.go`.

| Fact | Verified behavior |
|---|---|
| Stages | `background`, `screening`, `compensation`, `closing`, in that order. `QuestionStage.Valid` accepts only these four. |
| Publication stages that need a nonempty question | `background`, `screening`, `compensation`. Closing may have no mandatory question. |
| Stage budget | Each stage integer is stored. Publication requires every stage positive and the total in 300–600 seconds. Draft storage separately allows 0–900 per stage; a published revision used by this slice has already passed publication. |
| Questions | `MandatoryQuestion` keeps `id`, `stage`, and `text`. Order is the stored order. |
| Follow-up field | No follow-up count is stored on `HRConfiguration` or in the revision JSON. Publication only requires 30 seconds spare in `screening` after 30 seconds per mandatory question in that stage. |
| Compensation | The only mode is `expectations_only`. The revision stores `salaryCeilingRef` as an id and version. |
| Ceiling amount | `infrastructure/service/agentreference/local.go` resolves demo id `22222222-2222-4222-8222-222222222222` version `1` as ready. It returns no amount, role, or policy text. |

The showcase split of about 2 / 4 / 2 minutes in technical-design section 7 is demo guidance. A published revision such as 60 / 240 / 90 / 30 seconds is the governing soft budget for that run. The hard stop remains 15 minutes from `RUNNING`, including recovery and finalization.

The follow-up cap is the later explicit platform rule: at most two follow-ups per stage (technical-design section 7). This slice freezes that cap onto the interview progress row when the run first activates, so a later code change does not rewrite a run already in progress. See decision-log D9.

### Voice proof-of-concept, assessed and left in place as a prototype

| Verified source | What it does | What this slice does with it |
|---|---|---|
| `application/usecase/voicesession/negotiate.go` | Trims an offer, normalizes SDP to CRLF with a trailing break, sends static prototype instructions, returns only an SDP answer. Instructions say not to use a fixed HR list and not to apply compensation policy. | Reuse the SDP normalization rule. Do not call `Negotiate.Execute` for an interview. Those instructions contradict `BR-CORE-060`. |
| `infrastructure/service/realtimecall/adapter.go` | Posts one GPT-Live `/v1/live/sessions` body (`gpt-live-1`, voice, WebRTC offer) and returns `transport.sdp`. The provider session id is discarded. Empty key and every provider failure become `ErrSessionUnavailable`. | Reuse the Live session URL, model default, and bearer setup as the approved target. The interview driver must retain the provider session id. |
| `delivery/http/handler/voicesessionhandler` and `delivery/http/voicepage/page.html` | Unauthenticated `POST /voice-sessions` and `GET /voice-session`. The page gathers ICE (15 seconds), opens a microphone, plays remote audio, and sends `response.create` on the data channel. | Browser mechanics are input to the candidate-interview page. They are not mounted as the interview. |
| `infrastructure/service/liveprobe` and `test/livevoice/README.md` | Opt-in probe. Counts event type names. Does not retain transcript text. Documents that counts do not prove finality. S1–S4 are not passed. `session.started` is the one event name the harness page is written to observe. | Evidence that a sideband can be attached in a test harness. Not a finality map and not production authority. |
| `infrastructure/runtime/voice.go` | Routes `/voice-session` and `/voice-sessions` with no candidate session. | Remains until the scoped session-attempt route replaces it. It is not an interview operation. |

GPT-Live is the approved provider (technical-design TD-11), conditional on feasibility. This design does not record a live S1–S4 pass.

### Lifecycle boundary

Execution Lifecycle owns identity, the states accepted / running / completed / failed / cancelled, attempt authority, and the first terminal commit. This slice calls that capability. It does not add those states to `interview_executions` and it does not teach the lifecycle about stages, questions, or salary.

The lifecycle operations this slice will call, once that capability publishes them:

| Call | Required guarantee | HR knowledge allowed in the lifecycle |
|---|---|---|
| Activate | Same execution. At most 2 attempts, 30 seconds each, inside a 90-second window from the first activation attempt. Success is the transition to running. Exhaustion fails that same execution. | None. This slice passes the budget. |
| Current-attempt fence | A superseded attempt cannot commit newer progress (`BR-CORE-057`). | None. |
| Locked progress callback | Nonterminal execution, expected version, and current attempt are checked in the same transaction as this slice's progress write. Lock the execution row before interview child rows. | The callback body is interview-owned. The lock is lifecycle-owned. |
| Complete | Completes only when this slice has already committed the required progress reference (`BR-CORE-056`). | The reference is an opaque id. |
| Fail | Terminal failure keeps already committed interview rows (`BR-CORE-070`). | The reason code is interview-owned text stored with the progress row, not a new execution state. |

`StartExecution` in `application/usecase/agentbindingusecase/start_execution.go` inserts the pin only. This slice does not extend that command into a state machine.

Candidate Interview owns consent and the page. Session-attempt still requires an existing candidate session, a controller id, and an execution that session is allowed to see. Takeover and cancel remain sibling commands. This slice rejects a controller id that is not the current one.

## 2. API / Interface Contract Changes

Interview traffic uses the candidate execution routes already named in technical-design section 9. Exact platform-wide error precedence stayed open there; the runtime-owned subset below is the contract for this slice.

| Method and path | Owner of the behavior | Result |
|---|---|---|
| `POST /candidate/executions/{id}/session-attempts` | Interview Runtime | Create or replay one voice attempt for activate or reconnect. Returns the SDP answer and the attempt identity. |
| `POST /candidate/executions/{id}/session-attempts/{attempt_id}/ready` | Interview Runtime | Records browser media readiness for that attempt only. |
| Fields contributed to `GET /candidate/executions/{id}` | Interview Runtime supplies the DTO. Candidate Interview owns the page and the route shell. | Stage, session condition, `nextAction`, safe coverage counts, version. |

`GET /voice-session` and `POST /voice-sessions` stay registered only until `CORE-IRT-007` removes them in the same change that mounts the routes above. They are not an interview contract. Removing them updates `docs/api/openapi.yaml`, `docs/api/voice-session.md`, and `docs/api/CHANGELOG.md` together.

No admin route. No report route. No browser WebSocket for application state. The candidate page polls.

### Session attempt

Request JSON object, unknown fields rejected, body limit 1 MiB, the same limit as the prototype handler:

| Field | Rule |
|---|---|
| `sdp` | Required non-blank string. Normalize line endings to CRLF and keep a trailing CRLF, the same rule as `normalizeSDP` in `negotiate.go`. |
| `intent` | `activate` or `reconnect`. |
| `expectedVersion` | Integer. The execution version the lifecycle must still hold. |
| `controllerId` | Required. Opaque tab id. Possession of the id is not a credential. The candidate cookie is the credential. |
| `idempotencyKey` | Required. Scoped to the candidate session and this execution. |

Success data:

| Field | Rule |
|---|---|
| `attemptId` | Lifecycle attempt identity. |
| `sessionId` | Local interview voice-session id. |
| `sdp` | Provider answer only. |
| `executionVersion` | Version after the durable attempt is recorded. |
| `sessionCondition` | `pending`, `ready`, or `rejected`. |

`201` when a new attempt is stored. `200` when the same key and the same intent return the original attempt and the original answer. A different intent with the same key is a conflict and creates nothing.

Browser readiness is not proof that a question was heard (technical-design TD-12 and `BR-CORE-068`). `ready` sets a flag on that attempt. Activation becomes ready only when three facts are true together: the lifecycle says the attempt is current, the driver has stored the provider session id, and this flag is set.

After a committed handoff, the execution read exposes `nextAction=activate_session` with an opaque authorization bound to that handoff and the current controller. The browser prepares a new SDP. It does not choose the stage.

### Candidate-visible read

The runtime DTO may include stage name, coverage counts, session condition, deadline instant, and `nextAction`. It excludes salary reference ids, ceiling amounts, revision JSON, transcript text of model instructions, competence language, and the HR report. Partial provider fragments are not included as answers.

### Prototype removal

`CORE-IRT-007` deletes the production registration of `GET /voice-session` and `POST /voice-sessions`, their OpenAPI paths, and the public voice-session document's claim that those routes are the voice API. Package code that only served them is removed in that task. `test/livevoice` remains an opt-in probe and is not a production route.

## 3. Data Model & Schema Changes

### Concepts and rules

Language is the capability's language. **Coverage**, **omission**, **handoff**, and **canonical transcript** apply to one bound HR interview. They are not execution-lifecycle words and they are not invoice words.

| Term | Meaning in this slice |
|---|---|
| Progress | The interview-owned record for one execution id. Identity is that execution id. |
| Coverage item | One mandatory question: `pending`, `asked`, `answered`, or `omitted`. |
| Technical loss | A gap caused by recording or connection failure. It is stored beside coverage. It is not an omission reason. |
| Follow-up | A model question, beyond the mandatory list, that cites one finalized candidate transcript entry in the same stage. Cap is the frozen per-stage limit. |
| Handoff | The committed source-stage record: coverage, technical gaps, and transcript references. A narrative summary is optional. |
| Checkpoint | A committed in-stage boundary. Recovery reads the latest checkpoint. Uncommitted speech is repeated. |
| Voice session | One provider conversation for one activated stage attempt. A replacement session keeps the same execution and the same stage. |

`Answered` means a finalized candidate response is stored. It does not mean the candidate is competent, suitable, or unsuitable.

Omission reasons:

| Reason | When the workflow may record it |
|---|---|
| `candidate_skip` | The candidate skipped after one clarification, and the skip is a finalized observation. |
| `no_answer_after_clarification` | One clarification was finalized and no candidate answer was finalized. |
| `soft_budget_expired` | Trusted time has reached the stage's published soft budget and the question is still `pending` or `asked`. |

A technical loss leaves the question `pending` or `asked`. It does not set `omitted`. It blocks stage authorization.

The soft budget is that stage's `stageSeconds` from the pinned revision, measured from the stage's activation instant. The hard deadline is 15 minutes after the lifecycle reports `RUNNING`. At the exact millisecond of either boundary, the boundary has been reached (`!now.Before(deadline)`).

Required record before completion:

- Every mandatory question is `answered` or `omitted`.
- No unresolved technical loss remains.
- Each finished stage, including closing, has a committed handoff. Closing's handoff does not authorize another session.
- Those rows are committed before the lifecycle `Complete` call.

If the hard deadline is reached and that record is missing, this slice calls lifecycle `Fail` and leaves the committed rows in place.

### Proposed tables

Version **10**, proposed files `migrations/pgsql/000010_interview_runtime.up.sql` and `000010_interview_runtime.down.sql`. Apply only after a clean version 9 (`000009_candidate_interview`, which follows execution-lifecycle `000008`). Hand-written SQL. No `AutoMigrate`.

All ids are UUID. Timestamps are `timestamptz`, written as UTC milliseconds. Columns are `NOT NULL` unless marked null. Foreign keys use `ON UPDATE RESTRICT ON DELETE RESTRICT`. No soft delete and no cascade.

| Table | Columns and constraints | Indexes |
|---|---|---|
| `interview_progress` | `execution_id` PK, FK to `interview_executions(id)`; `follow_up_limit` int `CHECK (= 2)`; `current_stage` text `CHECK IN ('background','screening','compensation','closing')`; `stage_activated_at` timestamptz; `committed_version` bigint `CHECK (>= 1)`; `controller_id` uuid null; `created_at`; `updated_at` `CHECK (updated_at >= created_at)` | PK is the execution. No execution status column. |
| `interview_coverage_items` | `id` PK; `execution_id` FK progress; `question_id` text; `stage` text check as above; `state` text `CHECK IN ('pending','asked','answered','omitted')`; `omission_reason` text null, check in the three reasons when state is `omitted`, and null otherwise; `clarification_count` int `CHECK (0 <= clarification_count AND clarification_count <= 1)`; `answer_entry_id` uuid null; `updated_at` | Unique `(execution_id, question_id)`. |
| `interview_technical_losses` | `id` PK; `execution_id` FK progress; `question_id` text; `session_id` uuid; `attempt_id` uuid; `resolved_at` timestamptz null; `created_at` | Index `(execution_id)` where `resolved_at` is null. |
| `interview_voice_sessions` | `id` PK; `execution_id` FK progress; `stage` text check; `attempt_id` uuid; `provider_session_id` text null; `readiness_at` timestamptz null; `replaced_session_id` uuid null; `opened_at`; `closed_at` timestamptz null | Partial unique `(execution_id)` where `closed_at` is null. Partial unique `(provider_session_id)` where it is not null. `attempt_id` has no FK: the attempt table is not in version 7. |
| `interview_transcript_entries` | `id` PK; `execution_id` FK progress; `session_id` FK voice session; `attempt_id` uuid; `speaker` text `CHECK IN ('candidate','model')`; `ordinal` int `CHECK (>= 1)`; `revision` int `CHECK (>= 1)`; `evidence_state` text `CHECK IN ('fragment','final')`; `body` text; `cites_entry_id` uuid null; `supersedes_entry_id` uuid null; `committed_at` | Unique `(session_id, ordinal, revision)`. No provider-event unique key. That predicate stays deferred with the finality map. |
| `interview_checkpoints` | `id` PK; `execution_id` FK progress; `session_id` FK; `attempt_id` uuid; `through_ordinal` int; `created_at` | Index `(execution_id, created_at)`. |
| `interview_handoffs` | `id` PK; `execution_id` FK progress; `source_stage` text check; `target_stage` text null, check when present, and null for closing; `revision` int `CHECK (>= 1)`; `summary` text null; `created_at` | Unique `(execution_id, source_stage)`. Unique `(execution_id, revision)`. |
| `interview_session_commands` | `id` PK; `execution_id` FK progress; `idempotency_key` text; `intent` text `CHECK IN ('activate','reconnect')`; `attempt_id` uuid; `created_at` | Unique `(execution_id, idempotency_key)`. |

`interview_coverage_items.answer_entry_id` and transcript citation columns reference `interview_transcript_entries(id)` only when non-null. A fragment cannot be the target of `answer_entry_id` or a follow-up citation. Enforce that in the progress aggregate and with a composite check that the referenced row is `final` inside the same transaction. A database FK alone cannot express "final".

`interview_technical_losses.question_id` matches a coverage item in the same execution. Closing with zero mandatory questions has zero coverage rows and zero loss rows.

Stage seconds and question text stay on the immutable `agent_revisions` payload addressed by `interview_executions.published_revision_id`. They are not copied into a second configuration document. `follow_up_limit` is stored because the revision JSON has no such field.

Down SQL drops these eight tables in child-to-parent order and does not drop `interview_executions`, assignments, or revisions. Down is for disposable databases. Populated environments use a forward repair.

The migration task also raises `schema.ReleaseVersion` to 10 after candidate-interview's clean version 9, adds the eight tables to `ownedPublicTables` and `RuntimeGrantTables`, and updates the runner so it accepts target 10. `make docker.up` must expect clean version 10. Startup still performs no DDL.

Existing rows from earlier versions are unchanged. No backfill. An execution pin with no `interview_progress` row means the interview has not activated.

### Seed / reference data

No seeder and no reference-data migration. Questions, stage seconds, and compensation mode already live in the published revision. The demo ceiling id stays a code constant in `agentreference.Local`. This slice does not insert a salary amount. Inventing one would publish a hiring limit the product has not supplied.

Optional demo interviews are created through publication, assignment, and the candidate start path. They are not SQL seed.

### Adapter and transactions

GORM models live in `infrastructure/repository/interviewrepository`. They are not domain types and not HTTP DTOs. Updates use explicit `Select`, including zeros and nulls: `clarification_count` 0, `follow_up_limit` 2, `closed_at` null, `summary` null, `readiness_at` null. Do not use `Updates` with a struct that omits zero fields, unconstrained `Save`, or association autosave.

Reads of one execution load progress, coverage, open session, latest checkpoint, and unresolved losses in bounded queries. Transcript pages are loaded by session and ordinal, not by walking GORM associations per row.

Network I/O stays outside the transaction. Provider session creation happens after the attempt row is committed, or the uncertain create is reconciled by the same idempotency key. The driver call is not inside the database transaction.

| Commit | Writes together | Outside the transaction |
|---|---|---|
| Open progress | Progress row, pending coverage rows, first session shell, command row, version 1 | Provider HTTP |
| Checkpoint | Final transcript rows supplied as already classified, coverage changes, checkpoint row, version bump | Provider events |
| Handoff | Handoff row, source session `closed_at`, version bump, target stage recorded as authorized and not yet opened | Next provider session |
| Replacement session | New session row, previous session `closed_at`, command row | Provider HTTP |
| Readiness | `readiness_at` on the current session | None |
| Closing handoff | Final handoff with `target_stage` null | Lifecycle `Complete` or `Fail` |

The lifecycle locked callback wraps checkpoint, handoff, and closing commits. Until that callback exists, repository tests cover the interview writes alone, including rollback. They do not claim a cross-aggregate fence.

Lock order when both stores exist: execution row, then `interview_progress`, then children. This repository does not lock `interview_assignments`.

Restoration loads stored rows and checks internal consistency: known enums, omission reason only on `omitted`, answer citation only to a final entry in the same execution, handoff stage belonging to this run, a single open session. It does not re-apply the 300–600 publication total and it does not re-validate draft bounds. Corrupt rows fail as technical restoration errors. They are not `INVALID_INPUT`. Previously stored `follow_up_limit` is trusted. New runs write `2`.

### Rule ownership and proof plan

All proof is planned. No case below is an observed pass of `BR-CORE-060`..`BR-CORE-071`.

| Material rule | Owner | Enforcement paths | Proof plan |
|---|---|---|---|
| `BR-CORE-060` bound revision | Progress creation reads the pin's `published_revision_id` through the existing revision reader | Activate path. Restore does not substitute the active revision. | A later publication leaves questions and stage seconds unchanged. Closing may have zero questions. Total outside 300–600 cannot appear because publication already rejected it; the runtime test uses a stored revision fixture. |
| `BR-CORE-061` ask or omit; one relevant follow-up | Progress coverage methods and follow-up counter | Record omission only for the three reasons. Count a follow-up only when it cites a final candidate entry in the stage. Cap is the frozen limit. | Skip, silence after one clarification, soft-budget expiry, cap 2 then a rejected third, cited follow-up versus uncited text. Representative "relevant wording" for `H-02` is a scenario check, not a second model call. |
| `BR-CORE-062` answer is not competence | Coverage state and schema | No competence field exists on the aggregate or the tables. | Repository mapping and instruction builder have no hire, reject, rank, or competence result. |
| `BR-CORE-063` lost audio | Technical-loss entity | Loss insert leaves state `pending` or `asked` and blocks authorization. | Loss then proposal is rejected. The same question can later be answered. A loss is never read back as `omitted`. |
| `BR-CORE-064` workflow authorizes | `Progress.Authorize` | Model proposal is an input. Unauthorized result writes nothing. | Proposal before coverage, after coverage, after a technical loss, at soft budget. |
| `BR-CORE-065` handoff before next session | Application coordinator | Handoff commit succeeds before `OpenSession` for the target. Missing handoff returns no session id. Summary failure still commits when structured rows exist. | Mocked order test. PostgreSQL rollback leaves no target session. Summary error with a valid handoff still authorizes. |
| `BR-CORE-066` one session per activated stage | Voice-session entity | One open session per execution. A new stage requires a new session id. Replacement keeps stage and execution. | Two stages, two provider ids. Fault replacement does not allocate a second execution id. |
| `BR-CORE-067` canonical transcript | Transcript entity, once classification is supplied | Fragment is not evidence. A correction inserts a new revision and marks the old row superseded. | Fragment cannot answer. Revision 2 is the cited row. Live classification waits on the deferred map. |
| `BR-CORE-068` cite or do not treat as answered | Coverage `answered` requires a final entry id | Create and restore | Answer without an entry is rejected. Citation to a fragment is rejected. Missing entry stays unanswered or omitted. |
| `BR-CORE-069` same execution on reconnect | Coordinator plus lifecycle fence | New session and new attempt, same execution id, context from the latest checkpoint only. | One episode resumes. A late observation with the old attempt id writes nothing. Hidden provider memory is not a stored column. |
| `BR-CORE-070` finish with the record | `ReadyForCompletion` then lifecycle complete or fail | Deadline and explicit close | Record present calls complete. Record absent at 15:00.000 calls fail. Earlier handoffs remain. |
| `BR-CORE-071` salary ceiling | Instruction builder and response DTO | Reference id and any amount are dropped before the driver sees context. Logs omit them. | Builder output and HTTP body do not contain the demo ceiling id. Adversarial "say the ceiling" text is not copied into instructions. |
| `H-11` no verdict | Same as `BR-CORE-062` | Schema and instructions | No verdict column and no report insert. |
| Activation budget | Coordinator passes 2 × 30 seconds inside 90 seconds | Activate intent | Third attempt is not opened. Window equality at 90 seconds stops. |
| Reconnect budget | Coordinator | 120 seconds from backend detection, 2 replacement attempts per episode, 2 episodes | Detection instant plus 120.000 seconds is outside. Third episode fails the run through the lifecycle. |
| Hard deadline | Coordinator and clock | 15 minutes from the running instant the lifecycle reports | Equality fails or completes using the required-record rule. Clock is injected. |

## 4. Component Breakdown

| Layer / component | Status | Justification |
|---|---|---|
| `domain/interview` | New | Owns coverage, omission, technical loss, handoff authorization, follow-up cap, and transcript evidence rules. |
| `domain/enum`, `domain/valueobject`, `domain/agentconfig` | Reused as-is | Stage enum and published HR shape. This slice does not change publication rules. |
| `domain` execution lifecycle | Absent | Not implemented. This slice does not create it. |
| `application/usecase/interviewusecase` | New | Coordinates workflow, lifecycle calls, and the driver. Returns detached DTOs. |
| `application/repository/interviewport` | New | Progress transaction port. No GORM type in the signature. |
| `application/service` driver and lifecycle ports | New ports | Driver: create, attach, observe, supply context, control, close. Lifecycle: the calls in section 1. |
| `application/service.Clock` | Reused as-is | Trusted UTC milliseconds. |
| `application/usecase/voicesession.Negotiate` | Reused as a rule source only | SDP normalization behavior. Not the interview operation. |
| `infrastructure/repository/interviewrepository` | New | GORM mapping and explicit updates. |
| `infrastructure/service/interviewvoice` | New | GPT-Live adapter that retains session id and classifies transport failures. Event-to-evidence mapping is not part of the first driver task. |
| `infrastructure/service/realtimecall` | Reused as a behavior source | Live URL, model default, bearer call. Production `Exchange` stays the prototype path until `CORE-IRT-007`. |
| `infrastructure/service/liveprobe` | Reused as-is | Opt-in probe only. |
| `delivery/http/handler/interviewhandler` | New | Session-attempt and readiness. Mocked-operation tests. |
| `delivery/http/voicepage`, `voicesessionhandler` | Removed from the production router by `CORE-IRT-007` | Prototype mechanics remain the reference for the candidate page, which this slice does not build. |
| `infrastructure/runtime` | Modified | Wire the interview handler. Remove prototype routes in the same change as the new routes. |
| `migrations/pgsql`, `infrastructure/database/schema` | Modified | Version 10 and the release guard. |
| RabbitMQ / event queue | Out of this slice | Live voice does not publish or consume broker messages. |

### Detail per component

**Progress aggregate.** Root id is the execution id. It is an entity with a lifecycle of stages inside one run: the current stage changes only through `Authorize` followed by a committed handoff. Coverage items, transcript entries, handoffs, checkpoints, and voice sessions are entities inside that boundary because a stage change is valid only when their invariants hold in one commit. Follow-up cap, omission reason, and "fragment is not evidence" are value rules on those entities.

A separate transcript aggregate would let a handoff commit while the transcript commit fails. That breaks `BR-CORE-065`. One progress boundary is the smallest boundary that can keep the handoff and its evidence together.

**Workflow operation.** One application type loads the pinned revision, calls `Progress`, and persists through the port. It passes lifecycle budgets. It does not contain SQL. It does not contain provider HTTP. Instruction text is built here so the salary rule has one owner before the driver.

**Driver.** The port returns observations: session id, transport up or down, and opaque provider events. The first implementation may surface event type and raw payload to the application only behind the blocked classifier. It must not mark coverage. `close` is best effort, matching the probe's unconfirmed `session.close`. Failure to close does not roll back a committed handoff.

**HTTP handler.** Parses the attempt and readiness bodies, maps errors, and writes the envelope. It does not decide stage changes.

### Material structure decisions and assessment

| Choice | Reason and material alternatives | Authority and refinement boundary | Assessment outcome |
|---|---|---|---|
| Progress aggregate in `domain/interview`, execution states left to the lifecycle | The rules that change together are coverage, handoff, and transcript evidence. The rules that change together for every agent are identity and terminal state. Putting stage names on the execution pin would make a later document run carry interview fields (`BR-CORE-059`, architecture section 10). A shared "workflow engine" has no second caller in this slice. | Binding split: architecture ownership table and `BR-CORE-059`. Internal type names may be refined. The execution id remains the progress id. | **Retained.** |
| New driver port; prototype `Negotiate` is not the interview operation | `Negotiate.Execute` always sends prototype instructions and drops the provider session id. Extending it with a mode flag would keep two policies behind one function that today has a single reason to exist: the feasibility prototype. Copying the CRLF rule into the interview operation is a small duplication of a transport detail, with a test that both match. | User decision and technical-design section 2. The CRLF rule is the behavior to preserve. The prototype route is removed by `CORE-IRT-007`, not reused. | **Revised** relative to "point the interview at `POST /voice-sessions`". |
| Freeze follow-up cap `2` on `interview_progress` | The revision JSON has no cap. Reading a global constant on every request would let a later constant change an active interview. Storing `2` at activation freezes the rule the run started with. Adding a cap field to HR publication is a configuration change outside this slice. | `BR-CORE-060` plus technical-design section 7. Restoration must not overwrite the stored cap. | **Retained.** See decision-log D9. |
| Four stored stages, published seconds as the soft budget | Code and the capability name four stages. The section 7 table of three timed blocks is showcase guidance. Using 2/4/2 as the runtime timer would ignore the revision the candidate was bound to. | `BR-CORE-060`. User decision that the published revision governs. | **Retained.** |
| No provider-event unique key and no finality classifier in the executable tasks | `liveprobe` counts names and states that counts do not prove finality. Technical-design section 5 marks transcript uniqueness open. Guessing `response.done` or another name would encode an unverified contract. | Deferred until this design records an explicit map from a reviewed S1 record. | **Deferred.** Blocks `CORE-IRT-012` through `CORE-IRT-015`. |
| Manual SQL version 10 and GORM CRUD | Project migration policy. This file is version 10, after execution-lifecycle `000008` and candidate-interview `000009`. `AutoMigrate` would contradict startup, which only reads the version. | AGENTS.md and Identity and Access D8, already in force. Raw SQL is not required for these keyed writes. | **Retained.** |
| No RabbitMQ on the voice loop | Stage changes are request-driven and must commit before the next session. A broker hop adds delay and a second failure mode to a loop the platform assigns to the driver and workflow. Report dispatch stays with the reporting sibling. | Technical-design TD-09 limits the broker to document and report work. User decision. | **Retained.** |
| Numeric salary comparison | No amount exists in the reference resolver or the revision. Non-disclosure is enforceable. A comparison would require inventing a number. | `BR-CORE-071` example: discuss expectations; do not speak or show the ceiling. | **Deferred** for any numeric clamp. Non-disclosure is **retained**. |
| Next-stage activation count and a finalization reserve | Initial activation and reconnect numbers are in technical-design section 10. That section still says next-stage activation details and the finalization reserve are open. The handoff order does not need the missing count. Inventing "also 2 attempts" would close an item the later design left open. Finalization uses the same 15-minute deadline. | Blocks only `CORE-IRT-016`. | **Deferred.** |

Ordinary create path: load pin and immutable revision, build pending coverage, commit progress, then ask the driver for a session. Failure path: provider error before the command row commits returns `503` and leaves no progress; provider error after the command row commits returns the same attempt on retry and does not open a second execution.

Ordinary handoff path: proposal in, `Authorize` in the aggregate, one transaction writes the handoff, then a later call opens the next session. Invalid proposal path: `Authorize` returns a conflict and the transaction does not open.

A domain service was not added. The rules sit on `Progress`. A factory `NewProgress` builds the first pending coverage from the revision questions so callers do not assemble that set by hand. `RestoreProgress` is a separate function and does not call publication assessment.

## 5. Flow Sequence

### Activation

1. Candidate session, consent, and the execution pin already exist. This slice does not create them.
2. `POST .../session-attempts` with `intent=activate` authenticates the candidate and checks the execution is inside that assignment.
3. Same idempotency key and same intent return the stored attempt.
4. The coordinator asks the lifecycle to open an attempt under the 2 × 30 second / 90 second budget.
5. The coordinator reads the pinned revision, builds `Progress`, and commits coverage plus the session shell.
6. Outside the transaction, the driver creates a GPT-Live session and stores the provider session id on success.
7. The browser posts `ready`. That flag alone does not mark a question heard.
8. When provider identity, backend control, and browser readiness match the same attempt, the stage is active. The lifecycle has already moved the execution to running on successful activation.

### Stage and handoff

1. The driver delivers observations. Until event finality is mapped, those observations are not written as evidence.
2. Once a classified final event is available, the coordinator updates coverage or follow-up count inside a fenced transaction and may write a checkpoint.
3. The model proposes the next stage. `Authorize` checks coverage, technical losses, soft budget, and the frozen follow-up facts.
4. The transaction commits the handoff. Optional summary failure keeps the structured handoff.
5. A later session-attempt, or the `nextAction` already authorized, opens one new voice session for the target stage. Failed provider setup retries that committed target. It does not authorize a different stage and it does not create an execution. The retry limit is `CORE-IRT-016` and is blocked.

### Reconnect

1. The driver reports the connection down. The coordinator stores the detection instant from `Clock`.
2. Within 120 seconds, `intent=reconnect` opens a replacement attempt, at most two attempts in the episode, at most two episodes.
3. The new session receives committed checkpoint context only. Uncommitted speech is not replayed from provider memory, because that memory is not stored.
4. An observation carrying the replaced attempt id fails the fence and writes nothing.
5. Acceptance evidence is one deliberate episode (`H-08`). The stored budget is still the section 10 budget.

### Completion and deadline

1. Closing handoff commits with a null target stage.
2. If `ReadyForCompletion` holds, the coordinator calls lifecycle `Complete` with the progress id. It does not insert a report job.
3. If the hard deadline is reached first and the record is incomplete, the coordinator calls lifecycle `Fail`.
4. A race between complete and fail is the lifecycle's first terminal commit. This slice sends one outcome per decision.

## 6. Request / Response

Session-attempt and readiness shapes are in section 2. Instruction context sent to the model contains the current stage's mandatory question text, competency topic strings, the role name returned by the reference reader, the remaining follow-up count, and `expectations_only`. It excludes `salaryCeilingRef`, the demo ceiling id, and any amount.

Public success envelope: `{"status":"success","data":{...}}`. Error envelope: `status=error`, `data` an empty object, `error.code`, `error.message`. `Accept-Language: id` selects Indonesian.

### Creation, restoration, and transition contracts

| Path | Accepted inputs and normalization | Errors and failure timing | Side effects and compatibility | Proof plan |
|---|---|---|---|---|
| Create progress | Pinned revision id, question list, stage seconds, mode `expectations_only`, cap written as 2. Question ids are the stored ids. | Missing revision or unknown stage fails before insert. Provider failure before the command insert leaves no row. | One progress row per execution. Later active revision is not read. | Create, duplicate execution id, provider failure before and after insert. |
| Restore | Stored enums and citations as written. Cap and stage seconds are not recomputed from today's publication constants. | Unknown state or a citation to a missing or non-final row is a technical restore error after the read. | No writes on load. | Round-trip. Historical cap 2 remains 2. Corrupt state is not `INVALID_INPUT`. |
| Coverage transition | `pending` to `asked` to `answered` or `omitted`. `asked` may return to a later `answered`. `omitted` and `answered` do not move again. | Illegal move writes nothing. Answer without a final entry writes nothing. | Citation stored on the coverage row. | Each legal edge and each illegal edge. |
| Stage proposal | Current stage plus a proposal for the next fixed stage only. | Rejected proposal writes nothing and returns conflict. | No session id. | Skip a stage. Proposal with a technical loss. Proposal with a missing handoff. |
| Handoff commit | Authorized proposal, coverage snapshot, final transcript ids. Summary may be empty. | Transaction failure leaves the source stage current. | Source session closed. Target not opened inside the transaction. | Rollback test. Summary error still commits structured rows. |
| Reconnect | Detection time, episode counts, checkpoint id. | Outside 120 seconds, third attempt, or third episode writes no new session. | New session row, same execution, old attempt fenced. | Boundary instants and a late old-attempt write. |
| Complete or fail | Required record, or hard-deadline instant. | Lifecycle rejection does not delete handoffs. | One terminal call. No report row. | Both outcomes and a rollback of the closing handoff before the lifecycle call. |

## 7. Error Handling

Reuse identity-access codes for the cookie and CSRF failures the access stack already returns: `IA_UNAUTHENTICATED` (401) and `IA_FORBIDDEN` (403). An execution outside the candidate's assignment is `NOT_FOUND` (404), including when the id exists for someone else.

Runtime codes, added to `pkg/errs` and both translation files:

| Code | HTTP | When |
|---|---|---|
| `INVALID_INPUT` | 400 | Malformed JSON, unknown field, blank SDP, bad intent, missing key. |
| `METHOD_NOT_ALLOWED` | 405 | Wrong method on a mounted path. |
| `IRT_CONFLICT` | 409 | Stale version, wrong controller, same key with a different intent, illegal coverage or stage move, activation or reconnect budget exhausted after the lifecycle has recorded the failure. |
| `IRT_NOT_READY` | 409 | `ready` for an attempt that is not current, or a session-attempt while a technical loss blocks the next stage. |
| `SESSION_UNAVAILABLE` | 503 | Provider or missing key before a durable attempt exists. No provider body in the message. |
| `INTERNAL_ERROR` | 500 | Technical and corrupt restoration. Log the cause. Response text stays generic. |

Precedence: authentication, authorization, malformed input, not found, conflict, provider-before-accept, technical. A classified business error is never rewritten as `INTERNAL_ERROR`. `errs.Technical` still maps to 500.

Logs may include execution id, attempt id, local session id, stage, version, and timing. They exclude SDP, transcript bodies, instruction text, cookies, the provider key, and salary ids or amounts.

## 8. Constraints Checklist

- [ ] Rules `BR-CORE-060`..`BR-CORE-071` implemented as planned behavior, with tests that do not claim a historical pass.
- [ ] Execution states, attempt fencing, and terminal commit stay behind the lifecycle calls.
- [ ] Published revision governs questions, stage order, stage seconds, and `expectations_only`.
- [ ] Follow-up cap frozen at 2 because the revision has no cap field.
- [ ] Showcase 2/4/2 timing is not the soft timer.
- [ ] Hard deadline is 15 minutes from running, inclusive of recovery and finalization.
- [ ] Initial activation is 2 attempts, 30 seconds each, 90-second window.
- [ ] Reconnect is 120 seconds from backend detection, 2 replacement attempts per episode, 2 episodes.
- [ ] Acceptance reconnect evidence is one deliberate episode.
- [ ] Handoff commits before the next session.
- [ ] Lost audio is a technical loss, never an omission.
- [ ] Salary reference and amount stay out of model context, HTTP data, and logs.
- [ ] No hire or reject verdict and no report insert.
- [ ] Prototype `GET /voice-session` and `POST /voice-sessions` are not the interview path, and are unmounted when the new routes mount.
- [ ] Driver creates, attaches, observes, controls, and closes, and does not commit the lifecycle.
- [ ] GPT-Live is the target. Feasibility evidence is still absent.
- [ ] No RabbitMQ in the live loop.
- [ ] Manual SQL version 10 after clean version 9. No `AutoMigrate`.
- [ ] GORM models stay in infrastructure. Zero values are explicitly updated.
- [ ] New Go declarations satisfy `test/architecture/documentation_test.go`.
- [ ] Provider event finality is not treated as known.

## 9. Risks & Open Questions

| ID | Decision | Owner | Blocked work | Ready when |
|---|---|---|---|---|
| Q1 | Which provider events are finalized speech, fragments, corrections, and stale-session duplicates? | Platform / provider feasibility | `CORE-IRT-012`, `CORE-IRT-013`, `CORE-IRT-014`, `CORE-IRT-015` | This design is revised with an explicit map: event name, fragment versus final, correction identity, ordering, and late events from a replaced session. The map cites a reviewed S1 record. A document-only guess does not clear the block. |
| Q2 | How many provider attempts may open a stage after the handoff is committed, and is any time reserved inside the 15 minutes for finalization? | Product owner of technical-design section 10 | `CORE-IRT-016` | This design records the numbers. Until then, finalization uses the single 15-minute deadline and the next session is not opened with an invented retry count. |
| Q3 | Execution Lifecycle has no activation, fence, or complete operation in this repository. | Execution Lifecycle capability | `CORE-IRT-007`, `CORE-IRT-008`, `CORE-IRT-010`, `CORE-IRT-011`, `CORE-IRT-015`, `CORE-IRT-017` | That capability exposes the calls in section 1 without HR stage fields, including a same-transaction progress callback. |
| Q4 | No numeric salary ceiling is stored. | Product | No implementation task. Non-disclosure proceeds. | A confidential amount and a rule for comparing it to a spoken expectation exist outside model context. This slice must not seed an amount. |
| Q5 | Live S1–S4 and `H-01` are not evidenced. | Provider feasibility | `CORE-IRT-017` for a pass claim. `CORE-IRT-006` may still ship against a fake provider. | A sanitized record shows one interview-path WebRTC session with a retained provider session id. Event counts alone do not clear Q1. |

Independent work that stays executable: domain rules, ports, mocked workflow, version 10 schema, GORM mapping of already classified rows, driver transport against a fake provider, and the salary-free instruction builder.

Risk: the prototype route spends provider budget with no candidate session until `CORE-IRT-007`. That task waits on Q3 because the replacement route waits on Q3.

Risk: `attempt_id` has no foreign key until the lifecycle schema exists. Application code still treats an empty attempt id as illegal. A later migration may add the FK; this version 10 file must not point at a missing table.

## 10. Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial design | Mode A planning for `BR-CORE-060`..`BR-CORE-071`. |
| v2 | 2026-11-02 | Assign `000010_interview_runtime` and clean floor 10 | assign a unique migration version after the parallel tripack collision on 000008 |
