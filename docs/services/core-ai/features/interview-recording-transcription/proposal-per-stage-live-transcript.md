# Proposal (Draft): Per-Stage Live Transcript

Date: 2026-11-03. Status: **D4 and D7 accepted by the owner on 2026-11-04** and recorded in the Interview Runtime decision log as D10 and D11. D6 and D8 were accepted later the same day (Interview Runtime D12 and D13); the first-session register and greeting (D14) and a single new notice version (Candidate Interview D4) were also accepted. D5 remains proposed. Nothing here is implemented. This file does not change [decision-log.md](decision-log.md), [task-plan.md](task-plan.md) or [technical-design.md](technical-design.md); it records what would change if the owner accepts it. Input: [concept-live-transcription.md](../../../concept-live-transcription.md).

## 1. Question and short answer

The concept proposes collecting live transcript events over the sideband WebSocket. The open concern was separating a transcript by interview stage.

Observed in the repository (read only, no test run):

- One interview currently runs as one provider session. `OpenStage`, `Propose` and `CommitHandoff` have no caller (todo-gap G7). `CompleteStage` (IRT T21) advances the stage inside the same session.
- A full-interview transcript has no stage marker except the interviewer's own wording. A recording does not carry one either.

Proposal: **one provider Live session per stage**, with live transcript events stored per session. The stage boundary then equals the session boundary and no inferred marker is needed. The cost is a pause at each handoff (todo-gap G10, about 2.4 s measured once, not re-measured).

## 2. Proposed decisions

Numbering continues this feature's log. Each is a proposal until the owner marks it active. **Where each would be recorded** follows ownership (section 6): D5 belongs to this feature. D4, D6, D7 and D8 concern stage and session mechanics owned by Interview Runtime, so on acceptance they move to that feature's decision log under its own numbering, and this file keeps a pointer. D4 largely restores the original Interview Runtime design (technical design: "one session per activated stage", new stage requires a new session id); the deviation is IRT T21's `CompleteStage`, which advances the stage inside one session.

### D4: One provider session per stage

- **Business driver:** HR evidence must be traceable to a stage; BR-CORE-066 already states each active stage has its own voice session.
- **Technical decision:** Keep the A design. Reject `session.update` on one long session as the stage mechanism. Reconnect inside a stage stays a replacement session for the same stage (BR-CORE-069).
- **Alternatives considered:** One session with a stage marker on its timeline (option B). It avoids the pause but the marker is only an estimate (last observed `end_ms` at commit time), and an utterance crossing the boundary needs an assignment rule.
- **Trade-offs accepted:** Up to three pauses per interview; model loses conversational nuance beyond the handoff context; possibly more provider cost (not measured).
- **Revisit when:** The spike shows the pause is unacceptable and the B marker is accurate enough.
- **Status:** **accepted 2026-11-04**; recorded as Interview Runtime [D10](../interview-runtime/decision-log.md). Supersedes the same-session behavior of `CompleteStage` for non-closing stages and settles G10. Implementation is Todo.

### D5: Live sideband events are the primary transcript source; recording is audit and recovery

- **Business driver:** Observability during the interview, native `start_ms`/`end_ms`, and no dependency on a timestamp-capable transcription model (`whisper-1` and `gpt-4o-transcribe-diarize` are documented in T1 as shut down on 2027-02-26; `gpt-transcribe` returned no timestamps).
- **Technical decision:** Store raw `session.input_transcript.delta` and `session.output_transcript.delta` events per session. Keep `session.store: true` recordings for audit, comparison and recovery. Download stays backend-only (D3 unchanged).
- **Alternatives considered:** Keep D2 (recording primary). It leaves timing unsolved and the timestamp model is being retired.
- **Trade-offs accepted:** Events can be lost on disconnect; recording becomes the recovery source, so recording work in T6/T7 stays in scope with reduced responsibility (text per channel, no timestamps needed).
- **Revisit when:** The spike shows live events are missing, late or of lower text quality than the recording transcript.
- **Status:** proposed, **gated on the spike (IRTREC T12)**. Would partly supersede D2 (source direction) and reopen G14; D1 and D3 remain.

### D6: A transition separates the business commit from provider finalization

- **Technical decision:** Order: stage end is recorded, old session close is requested, `session.closed` is awaited (bounded), then the next session is opened and activated. If `session.closed` does not arrive, record `finalization unconfirmed` for that session and continue under a defined recovery; do not report success.
- **Trade-offs accepted:** A new session state and a timeout policy to be chosen (no value proposed here).
- **Status:** **accepted 2026-11-04**; recorded as Interview Runtime [D12](../interview-runtime/decision-log.md). The timeout value is not set.

### D7: Stage-end rule for now is `Attest`, not strict `Authorize`

- **Business driver:** `Authorize` requires resolved coverage, and coverage needs final transcript (IRT T13/T14, blocked on Q1). G9 (owner, 2026-11-03) already chose model-stated completion.
- **Technical decision:** Use the attest path (order, stage clock, hard deadline) to seal the stage. Keep `Authorize` for later when coverage exists. Report labels these stages as model-stated, as G9 requires.
- **Open:** who triggers the statement (G7/G8). Options: model function call over the sideband (unproven on our account), or a backend rule. This proposal does not decide it.
- **Status:** **accepted 2026-11-04**; recorded as Interview Runtime [D11](../interview-runtime/decision-log.md). The trigger is a model function call over the sideband validated by the backend, still subject to the live spike in `CORE-IRT-021`; a backend-owned fallback is decided separately.

### D8: The candidate page learns about a transition by polling

- **Technical decision:** Extend the candidate-safe progress returned by `GET /candidate/executions/{id}` with a transition state. No SSE or push channel is added.
- **Status:** **accepted 2026-11-04**; recorded as Interview Runtime [D13](../interview-runtime/decision-log.md).

## 3. Reference flow

1. Page requests a voice session (`session-attempts`) with an SDP offer. Backend derives the stage from stored progress, never from the request.
2. Backend creates the provider session, attaches the sideband, supplies stage context, returns the SDP answer (already the order in `createSession`). The collector must be running before the answer is returned.
3. Page applies the answer and reports readiness (`/ready`, browser media readiness).
4. Events arrive and are stored under the session id.
5. Stage end: seal the stage (D7), request `session.close`, collector keeps reading until `session.closed` or timeout (D6), mark the session closed.
6. Progress poll shows the transition; page creates a new peer connection and calls `session-attempts` with the new intent; backend opens the next stage session with handoff context.
7. Closing stage: seal, close, complete the lifecycle (existing completion path).

## 4. Contract changes (proposed names)

| Item | Today | Proposed |
| --- | --- | --- |
| Voice session route | `POST /candidate/executions/{id}/session-attempts`, intent `activate` or `reconnect` | Add an intent for the next stage (working name `advance`). Same idempotency key, version and controller fencing. |
| Progress poll | Candidate-safe progress | Add a transition state (none, pending, ready). Exact shape undecided. |
| Readiness | Browser media ready | Unchanged. Do not reuse it to mean the collector is ready. |

Any public change updates `docs/api/openapi.yaml`, the candidate endpoint docs and `docs/api/CHANGELOG.md` together. Stage names are `background`, `screening`, `compensation`, `closing`.

## 5. Changes by layer (what the code review found)

| Layer | Finding | Proposed change |
| --- | --- | --- |
| `domain/interview` | `Attest` seals and advances `CurrentStage` in one step. After it, `OpenStage` takes the `CurrentStage == target` branch and `releaseUnprovisioned` returns a conflict when the open session has a provider id. `ActivateStage` needs the handoff count to equal the stage index plus one, which a seal alone already satisfies. | Split sealing from advancing for non-closing stages, so `OpenStage` can activate the next stage. The candidate "Selesai" path (`CompleteInterview`) seals all remaining stages in a loop and needs its own non-advancing semantics. To be proven by tests. |
| `application/usecase/interviewusecase` | `Attempts.start` supports only activate/reconnect. `appendSession` rejects a new session while one is open. | New intent routed to `OpenStage`; the old session is closed before the new one is appended. |
| `infrastructure/service/interviewvoice` | `Driver.Close` sends `session.close` then releases the socket at once. `Observe` returns opaque payloads. | Bounded wait for `session.closed`; a collector that classifies the two transcript event types and keeps reading until closed. Provider event schema stays an IRTREC T12 deliverable. |
| Persistence | `interview_voice_sessions.stage` already exists. `interview_transcript_entries` has no ms offsets or provider event id. | New manual migration for raw events (`session_id`, `provider_event_id` unique per session, speaker, delta stored verbatim, `start_ms`, `end_ms`, `received_at`) and finalization state on the session. Migration number to be rechecked against the IRTREC migration already proposed as 14. |
| `infrastructure/runtime` | Shutdown is bounded. | Collector lifecycle joins bounded drain and cancellation. |
| Candidate page | One peer connection. Not read in this review. | Close and recreate the peer connection on transition without losing tab control. |

## 5a. First-session prompt (instructions) review

Reference: [GPT-Live prompting](https://developers.openai.com/api/docs/guides/live-prompting). The page was read through a fetch summarizer, so exact heading names, the `<|SpawnThinking|>` token and the greeting mechanism must be re-checked against the page before any implementation. Code reviewed: `stageInstructions` in `application/usecase/interviewusecase/instructions.go` and `Driver.Create`/`SupplyContext` (read only; the prompt text itself was not run against a model).

How the prompt reaches the model today: `Driver.Create` sends one `instructions` string (plus `audio.output.voice`, `store`, model). `createSession` then sends the same text again with `session.update` after the sideband attaches. No `developer` input message, no greeting mechanism and no delegation are used.

| Guide recommendation | Current prompt | Gap |
| --- | --- | --- |
| Write the prompt and examples in the language to be spoken | English instructions, speech in Bahasa Indonesia, with only a few Indonesian example sentences | Mismatch. Register (formal Anda or casual kamu) is unspecified. Pick one and say it. |
| Sections in order: role and tone, backchannel policy, interruption policy, delegation policy, pre-answer rule | One flat list: role line, flow, current stage, "How to run it", scope rules | No tone, backchannel or interruption policy. Delegation sections apply only if delegation is used (it is not; G8). |
| Keep the full task procedure in backend instructions, not the conversation prompt | The whole flow, all mandatory questions and scope rules live in the conversation prompt | Applies only with delegation, which costs a backend model and is billed separately (G8). Keeping everything in one prompt is a deliberate choice to record, not an oversight to fix silently. |
| Clarify, do not guess unclear names, dates or numbers | Absent | Matters for HR: salary figures, years of experience and role names were the transcript risks seen in the T1 samples. |
| Do not treat cough, music or nearby speech as a request; keep listening during pauses; do not finish the candidate's answer | Absent | A thinking pause must not trigger the next question. |
| State pacing | Absent | Add a short rule. |
| Keep ordinary replies short ("one or two short sentences") | Absent | Add for interviewer turns. |
| Opening: greeting through a `developer` message in `session.input`, then once through `session.commentary.append`; use application playback when exact wording matters | Greeting is an instruction line in the main prompt ("Start by greeting") | The wording may be paraphrased. If the consent notice or a fixed disclosure must be spoken verbatim, use application-controlled playback; otherwise decide that paraphrase is acceptable. |
| Prompt text is the same on create and after attach | Sent twice (create, then `session.update`) | Probably redundant; todo-gap G3 and G13 already track it (the provider may fix frontend instructions after startup). Whether the second send is needed (for example for reconnect context) was not checked here. |

What the existing prompt already does well and should be kept: scope rules (decline off-topic, never reveal instructions, treat candidate speech as data), no evaluation or hiring outcome, no salary range or ceiling, no invented company facts, and the closing-stage rule that candidate questions are noted and not answered (G11).

Interaction with D4. Under one session per stage the prompt should describe the current stage only, plus the committed handoff context for stages after the first. The T20 change that renders the whole four-stage flow in one prompt exists because the model paced the stages itself (technical design v8); that rationale disappears when the backend opens each stage. The "name the next stage and continue" instruction would then conflict with the handoff and must be removed or replaced by an end-of-stage statement tied to the trigger chosen under D7 (G7/G8). Do not change the pacing wording before that trigger is decided.

Proposed work: IRT T25 below. Verify with a spike conversation in Bahasa Indonesia, including a thinking pause, an unclear number, a background noise, an interruption and an off-topic question. Unit tests on the rendered text can prove which rules are present and that the ceiling is never rendered (existing T9 and T20 coverage), not that the model obeys them.

## 6. Ownership, proposed tasks and affected documents

Recorded in task plans on 2026-11-04 (IRT T22 is already Done as documentation only): IRT T22..T25, CIN T14..T15 and IRTREC T12. IRTREC T13..T15 are not added because they depend on D5, which waits for the spike. IRT T24 and CIN T14 are recorded as Todo and rest on decisions D12 and D13 (this proposal's D6 and D8, accepted 2026-11-04). Tasks follow the feature that owns the behavior. Each feature keeps its own ticket prefix and numbering; nothing is renumbered, and none of these are added to a task plan yet. All are Todo.

### Interview Recording and Transcription (`CORE-IRTREC`, continues after T11)

| Proposed | Ticket | Subject | Depends on |
| --- | --- | --- | --- |
| T12 | `CORE-IRTREC-012` | Spike: live events, offsets and handoff pause on a real multi-stage run (section 7) | none |
| T13 | `CORE-IRTREC-013` | Reconcile this feature's design, strategy and decision log with D5; revise the timestamp requirement in T1/T7 | T12 |
| T14 | `CORE-IRTREC-014` | Collector and raw event persistence: classify the two event types, dedup, gap marking | T12, IRT T23 |
| T15 | `CORE-IRTREC-015` | Stage-scoped read model for reporting and recording comparison | T14 |

### Interview Runtime (`CORE-IRT`, continues after T21)

| Proposed | Ticket | Subject | Depends on |
| --- | --- | --- | --- |
| T22 | `CORE-IRT-022` | Reconcile design, capability wording (BR-CORE-066/067/070), todo-gap G9/G10/G14 and decision log with D4, D6, D7 | none (D4/D7 accepted) |
| T23 | `CORE-IRT-023` | Domain and application: separate stage seal from advance; new-stage open after the old session is closed; `Selesai` path in every stage | T22 |
| T24 | `CORE-IRT-024` | Driver: await `session.closed` with a bounded wait and an unconfirmed state | T22 |
| T25 | `CORE-IRT-025` | Prompt revision per section 5a: language and register, tone, interruption and unclear-audio rules, per-stage prompt with handoff context, greeting mechanism decision; keep scope and salary rules | T22 (pacing wording also waits for the D7 trigger) |

Existing Interview Runtime work interacts and is not closed by this proposal: T21 (the `CompleteStage` same-session path and the model-triggered ending), T16 (activation budget per stage, applicable again under D4), and T12–T13 (canonical ingest from classified events, which D5 would revive as the live-event path while IRTREC T14 owns persistence). The stage-end trigger (G7/G8) stays IRT T21 work; without it a real multi-stage run ends only when the candidate presses "Selesai".

### Candidate Interview (`CORE-CIN`, continues after T13)

| Proposed | Ticket | Subject | Depends on |
| --- | --- | --- | --- |
| T14 | `CORE-CIN-014` | New session-attempt intent and transition state in the progress poll; OpenAPI, endpoint docs and changelog | IRT T23, T24 |
| T15 | `CORE-CIN-015` | Candidate page: new peer connection per stage without losing tab control | CIN T14 |

### Interview Reporting

No new task. IRP T14 (read adapter for finalized evidence) later consumes IRTREC T15; the report must label stages as model-stated (G9).

### Documents affected on acceptance

| Area | Documents |
| --- | --- |
| Interview Runtime | `todo-gap.md` (G7–G10, G14), `task-plan.md`, `technical-design.md`, `decision-log.md`, `business/capabilities/interview-runtime.md` |
| Recording and Transcription | `decision-log.md` (D2 status, D5), `technical-design.md`, `tactical-strategy.md`, `task-plan.md` |
| Candidate Interview | `task-plan.md`, `technical-design.md` (route notes) |
| Platform | `docs/services/baseline.md` (F6, F9, section 15), `docs/traceability-matrix.md` (H-04, H-06, H-07, H-09, deviations), `core-ai/README.md` |
| API | `docs/api/openapi.yaml`, candidate endpoint docs, `docs/api/CHANGELOG.md` |

Execution Lifecycle, Identity and Access, and Agent Configuration and Publication are not expected to change. Update everything in one pass so no document keeps the old task status or source direction.

### Existing features: which get additions

Status is from [baseline.md](../../../baseline.md) (task plans win over READMEs). "Addition" means new tasks or new contract; "reopens" means a feature now shown as Delivered would show open work once the tasks are added to its plan.

| Feature | Current status | Addition? | Why, and what is reused |
| --- | --- | --- | --- |
| F6 Interview Runtime | Partial: T1–T11, T17–T20 Done; T21 in progress; T12–T15 blocked; T16 Todo | **Yes** (IRT T22–T24) | Owner of stage, session and handoff. Reuses `OpenStage`, `ActivateStage`, handoff rules, session rows, `createSession` order (create, attach, supply context). Existing T21, T16 and T12–T13 are affected but not closed (section 6). |
| F9 Recording and Transcription | Planned: T1 in progress, T2–T11 Todo | **Yes** (IRTREC T12–T15) | Source direction changes to D5; recording work narrows to audit and recovery. T1's proven WAV and per-channel transcription evidence stays valid. |
| F5 Candidate Interview | Delivered: T1–T13 Done | **Yes, reopens** (CIN T14–T15) | Existing `session-attempts`, `/ready` and progress poll need an intent, a transition state and a page change. Controller fencing, takeover, consent and coverage rules are reused unchanged. |
| F7 Interview Reporting | Partial: T1–T12 Done; T13–T15 open | **No new task** | Open work already includes the evidence adapter (IRP T14). It gains a new upstream (IRTREC T15) and the model-stated label (G9). Admission, outbox and failure isolation are untouched. |
| F4 Execution Lifecycle | Delivered for interview; T11 Todo (needs Document Analyzer) | **No** | Stage and session are progress-level concepts; activation, fencing, completion and cancellation are reused as they are. T11 is independent. |
| F2 Identity and Access | Delivered; T17 in progress (migration tooling, blocked by unrelated Docker regressions) | **No** | Candidate session already covers the 15-minute interview (BR-CORE-042). Open Q7 (expiry mid-interview) is unchanged. A new migration for this work must follow the manual migration policy; T17 status does not change. |
| F3 Agent Configuration and Publication | Delivered: T1–T14 Done | **No** | Stage times still come from the pinned revision. Document Analyzer stays configuration-only. |
| F1 Task Tracking | Partial (acceptance gate open) | **No** | Unrelated feature. |
| F8 Voice prototype | Delivered, then superseded | **No** | Routes removed. The opt-in `test/livevoice` probe could be considered for the spike; not checked here. |

Consequences on acceptance (checked against the repository on 2026-11-03):

**Migration numbering.**
- `migrations/pgsql/` holds `000001` through `000013`; the latest is `000013_interview_report_retry_window`. The root `README.md` also requires clean version 13 (lines 9, 24, 83). The next free number is 14.
- Two schema needs now compete for it: IRTREC recording tables (already proposed as 14, not created) and this work (raw transcript events, session finalization state). They need separate migrations or one combined reviewed migration; decide once, in this order of dependence, and assign numbers when each is written, not in advance.
- Raising the floor means updating the root `README.md` startup, status and up-target statements and `docs/design/database-migrations.md` in the same change, and the migration tests. Applied migration files are never edited (README rule).
- Do not take migration numbers from `docs/services/core-ai/README.md`: it lists reporting as `000011` and runtime as `000010`, while the directory has `000011_interview_runtime_integration` and `000012_interview_reporting`. The directory is the source.

**READMEs lag the task plans.** `core-ai/README.md` is confirmed stale on 2026-11-03: it still says startup accepts clean version 9, Interview Runtime and Reporting are "Planned. Nothing implemented", and Candidate Interview T12–T13 are open (the task plans and the baseline say otherwise; baseline section 15, items 1–4). This proposal uses task plans and the baseline as the status source. Fixing that README is a separate documentation task; it should be done in the same pass as the documents listed above so the new status is not written on top of old text.

**Baseline and traceability.**
- Baseline: F5 would move from Delivered to Partial once CIN T14–T15 are added to its plan; F6 and F9 keep Partial and Planned with more open tasks; the summary counts in section 3 and the use-case rows (UC-40, UC-42, UC-44) are reviewed.
- Traceability: accepting this proposal changes row text and gaps, not the section 2 counts, because no new evidence exists. Rows to revisit are H-04 (separate session per stage; stays Partial until a real multi-stage run), H-06 and H-07 (transcript source), H-09 (completion versus transcript timing), H-02 (prompt rules are model behavior, not proven by tests), and the deviations table (add the stage-session mechanism and the live-event source). A criterion moves only with recorded evidence, so counts change after IRTREC T12 and the implementation tasks produce it.

## 7. Spike protocol (IRTREC T12)

A runnable procedure, with the authorizations it needs and the owner's threshold table, is in [t12-spike-procedure.md](t12-spike-procedure.md); the list below is the original outline.

Requires explicit authorization for provider calls and cost. Use a synthetic candidate; record counts, timings and event types, not transcript text.

1. Run a real session with at least two stages and capture both transcript event types with `start_ms`, `end_ms` and arrival time. Record missing, late or duplicate events.
2. Download the same session's recording and compare event `start_ms` with the position in the WAV for the same utterance (offset, trimming).
3. Compare event text with `gpt-transcribe` per channel on the same audio, focusing on roles, technical terms and numbers.
4. Measure the real handoff pause and provider session setup time.
5. Cut the sideband mid-session and record what is lost and whether `session.closed` still arrives.
6. If a model function call is the trigger candidate, capture its event type only (G8).

Earlier evidence does not substitute: session `53574eba…` was recorded before any event was stored, and the `whisper-1`/`gpt-transcribe` results cover audio only.

## 8. Open questions and gates

- **Owner:** accept D4 over option B? Accept the pause? Who triggers stage end (G7/G8)? Accept model-stated stages in reports (G9)?
- **Provider facts not verified here:** event schema and finality, `start_ms` alignment to the recording, function-call delivery, session cost, `session.closed` timing, and the text quality of live transcripts. These come from documentation as quoted in the concept and the earlier chat, not from our account.
- **Data policy (Q2):** raw transcript events are personal data and may include sensitive disclosures. Notice wording, retention, access and deletion apply to events as well as audio. No retention period is proposed here.
- **Wording:** BR-CORE-067 says the backend transcript is the record; D5 changes how it is produced and when. H-09/P-10 timing reconciliation (T2) is still required.

## 9. Verification expectations

Per repository rules: domain behavior through public tests with controlled time; application orchestration through mocked ports; HTTP through mocked operation interfaces; real PostgreSQL for locking, rollback and persistence; `make check`, and `make test.docker` for the database suites. Required scenarios include a repeated advance request, an advance racing a reconnect, a handoff without `session.closed`, a late event after the stage seal, and `Selesai` in every stage. Database suites skipped without `TEST_DATABASE_URL` are reported as skipped, not verified.

## 10. Risks

- Pause and audio gap at each handoff may be unacceptable to candidates.
- Live events may be incomplete or of lower text quality than recording transcription; D5 is only valid if the spike says otherwise.
- Provider orphan sessions on a failed advance; the existing `closeCreated` cleanup must cover the new path.
- Two stage mechanisms (`CompleteStage` same-session and the A flow) must not coexist ambiguously after acceptance.
