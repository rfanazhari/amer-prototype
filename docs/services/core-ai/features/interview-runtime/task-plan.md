# Task Plan: Interview Runtime

## Ticket Identity

Capability group: **IRT — Interview Runtime**. Ticket prefix: `CORE-IRT`. These are repository planning IDs. No external tracker tickets have been created.

Stable map: `T1` → `CORE-IRT-001` through `T25` → `CORE-IRT-025`. Later revisions add ids. They do not renumber these.

Companion dependencies, not implemented by this plan:

- [Execution Lifecycle](../../business/capabilities/execution-lifecycle.md) `BR-CORE-050`..`BR-CORE-059` for activation, fencing, and completion.
- [Candidate Interview](../../business/capabilities/candidate-interview.md) for the page, consent, takeover, and cancel.
- [Interview Reporting](../../business/capabilities/interview-reporting.md) for the report after completion.
- [Agent Configuration and Publication](../agent-configuration-publication/technical-design.md) for the pinned revision this slice reads.

## Legend

- ID: `T1`..`T25`, stable
- Complexity: S (< 2h) | M (2–4h) | L (> 4h)
- Status: `[ ]` Todo | `[~]` In Progress | `[x]` Done | `~~obsolete~~`
- ref: [technical-design.md](technical-design.md) section
- DoD: verifiable
- Blocked: not executable until the ready condition is true and this design is updated when the condition changes the contract

Every Go task also requires `gofmt` on changed Go files, comments on new declarations so `make test.architecture` passes, and `make lint` clean for the touched packages. Domain and application tasks run `make test`. T1–T11 retain their recorded Done status and evidence limits. T8's Done scope is operations, handlers and contract; T18 owns its deferred production integration. No skipped database proof or mocked fence is promoted to a real integration pass.

## Tasks

### Independent foundation

- [x] **T1. Encode interview progress rules** — Domain — L
  - Ticket ID: `CORE-IRT-001`
  - Evidence: [T1 verification](t1-verification.md); implementation in `../../../../../domain/interview`.
  - depends: none — ref: design §3–§4, decision-log D1, D5, D6, D7, D9.
  - DoD: public tests cover coverage edges, the three omission reasons, technical loss leaving the question un-omitted and blocking `Authorize`, follow-up cap 2 with citation required, fragment rejected as an answer, handoff required before a target stage, closing handoff with no next stage, soft-budget and 15-minute equality, reconnect counters, and `ReadyForCompletion`. `NewProgress` writes cap 2 from a revision fixture that has no cap field. `RestoreProgress` keeps a stored cap and does not call publication assessment. No execution status type, no provider client, no GORM. `make test` and `make test.architecture` pass.

- [x] **T2. Define lifecycle, driver, and progress ports** — Application — M
  - Ticket ID: `CORE-IRT-002`
  - Evidence: [T2 verification](t2-verification.md); ports in `../../../../../application/service` and `application/repository/interviewport`, with generated Mockery mocks.
  - depends: none — ref: design §1 lifecycle table, §4.
  - DoD: interfaces exist for the lifecycle calls, the driver operations, and the progress store. Signatures carry execution id, attempt id, expected version, and budgets 2×30s/90s, 120s, 2 attempts, 2 episodes. No HR stage field appears on the lifecycle port. Mockery config generates the mocks. `make mocks` output is committed as generated. Documentation comments exist. This task does not call a real provider or a real database.

### Workflow and schema that do not need event names

- [x] **T3. Orchestrate proposals, handoff order, and budgets with mocks** — Application — L
  - Ticket ID: `CORE-IRT-003`
  - Evidence: [T3 verification](t3-verification.md); mocked workflow in `../../../../../application/usecase/interviewusecase`.
  - depends: T1, T2 — ref: design §5–§6, decision-log D1, D2, D3, D4.
  - DoD: mocked tests show an unauthorized proposal writes nothing; a successful proposal commits the handoff before any create-session call; summary failure still commits structured evidence; reconnect uses the same execution id and a new session; an old attempt id is dropped; activation passes the section 10 budget into the lifecycle port; the third activation attempt is not opened. Clock is injected. No live provider and no claim that a real fence exists.

- [x] **T4. Add SQL version 10** — Infrastructure schema — M
  - Ticket ID: `CORE-IRT-004`
  - depends: T1 — ref: design §3.
  - DoD: `../../../../../migrations/pgsql/000010_interview_runtime.up.sql` and `.down.sql` match the eight-table design, including checks, partial unique indexes, and FKs to `interview_executions` and `interview_progress`. No FK to a missing attempt table. No seed inserts. `schema.ReleaseVersion` becomes 10 and the runner accepts target 10 only after this task. Startup still does not migrate. Disposable PostgreSQL proof, via `make test.docker` or an authorized integration database: clean upgrade from version 9 keeps existing pins, fresh up creates the tables, down drops only the new tables. If `TEST_DATABASE_URL` is absent, say the database proof was skipped. Do not describe that skip as a pass.

- [x] **T5. Persist classified progress with GORM** — Infrastructure — L
  - Ticket ID: `CORE-IRT-005`
  - Evidence: [T5 verification](t5-verification.md); store in `../../../../../infrastructure/repository/interviewrepository`. PostgreSQL proof written but skipped (no `TEST_DATABASE_URL`, Docker daemon down). Version 10 also stores follow-up citations, reconnect counters and clarification/omission entry ids (design v3), so T14 and T15 are not blocked on schema.
  - depends: T1, T4 — ref: design §3 adapter and transactions.
  - DoD: store maps progress, coverage, sessions, handoffs, checkpoints, technical losses, and transcript rows that the caller already marked fragment or final. Explicit `Select` persists `clarification_count` 0 and null `closed_at` / `summary`. Rollback of a handoff leaves the source stage. Unique open session and unique handoff per stage are enforced by PostgreSQL tests. Restore of a corrupt state is a technical error. The store does not interpret provider event names. Same database-proof rule as T4.

- [x] **T6. Build the GPT-Live driver transport** — Infrastructure — L
  - Ticket ID: `CORE-IRT-006`
  - Evidence: [T6 verification](t6-verification.md); driver in `../../../../../infrastructure/service/interviewvoice`. Fake-server tests only; sideband frame shapes for context and control are unverified against the live provider.
  - depends: T2 — ref: design §4 driver, §1 prototype table.
  - DoD: `interviewvoice` implements create, attach, observe, supply context, control, and close against the driver port. Create retains the provider session id and the SDP answer. A fake HTTP server covers success, empty key, and non-success. Close failure is returned and does not imply a database rollback. Observe yields an opaque event and does not mark coverage. Package tests do not call the public network. `H-01` is not claimed. RabbitMQ is not imported.

- [x] **T9. Build stage instructions without the salary ceiling** — Application — M
  - Ticket ID: `CORE-IRT-009`
  - Evidence: [T9 verification](t9-verification.md); builder in `../../../../../application/usecase/interviewusecase/instructions.go`, follow-up cap in `domain/interview`. Fake-driver tests only; wording is not evaluated against a live model.
  - depends: T1, T2 — ref: design §6, decision-log D5, D6.
  - DoD: the context passed to `supply context` includes current-stage question text, topics, role name, remaining follow-ups, and `expectations_only`. The demo ceiling id and the field name `salaryCeilingRef` are absent. No amount parameter exists. Instructions do not ask the model for a hire or reject result. A test feeds a candidate sentence that asks for the ceiling and the outbound context still omits it. `make test` passes.

### Lifecycle integration (Q3)

T18 resolves integration Q3 with the durable adapter, persisted attempt authority, joined transaction callback and production mounting. Completed tasks below keep their original scopes and evidence limits. The generic lifecycle remains independent of interview stages.

- [x] **T8. Mount session-attempt and readiness HTTP** — Delivery / application — L
  - Ticket ID: `CORE-IRT-008`
  - Evidence: [T8 verification](t8-verification.md); operations in `../../../../../application/usecase/interviewusecase`, handlers in `delivery/http/handler/candidateinterviewhandler`. Mocks only. Done covers the operations, handlers and contract only (owner decision; Candidate Interview `CORE-CIN-012` relies on them). **Composition and mounting are completed by T18 / `CORE-IRT-018`:** the `InterviewLifecycle` adapter (attempt id versus generation), command persistence and forward migration, revision/pin readers, and composition in `infrastructure/runtime`. The routes are now mounted; T8's own evidence remains mocked.
  - depends: T3, T5, T6 — ref: design §2, §7.
  - Integration follow-up: T18; T8 stays Done within the owner-approved scope.
  - DoD: `POST /candidate/executions/{id}/session-attempts` and `POST .../ready` match design §2. Candidate cookie and CSRF are required. Another assignment's execution is 404. Same idempotency key replays; a different intent conflicts. `ready` does not set coverage to asked or answered. Handler tests mock the operation. OpenAPI, `../../../../api/CHANGELOG.md`, and an interview endpoint note are updated. `make test.contract` passes. English and Indonesian messages exist for `IRT_CONFLICT` and `IRT_NOT_READY`.

- [x] **T7. Remove the prototype voice routes from the process** — Delivery / runtime / contract — M
  - Ticket ID: `CORE-IRT-007`
  - Evidence: [T7 verification](t7-verification.md); routes removed in `../../../../../infrastructure/runtime/router.go`. T18 mounts the replacement routes; T7's removal remains intact.
  - depends: T8 — ref: design §2 prototype removal.
  - Integration follow-up: T18 mounts the replacement; preserve T7's completed removal.
  - DoD: `../../../../../infrastructure/runtime` no longer serves `GET /voice-session` or `POST /voice-sessions`. OpenAPI and `docs/api/voice-session.md` no longer describe them as available routes. Contract tests and router tests match that removal. `test/livevoice` still runs only as an opt-in probe. No interview call path remains on the old URLs.

- [x] **T10. Commit handoff and open the next session under the lifecycle lock** — Application / infrastructure — L
  - Ticket ID: `CORE-IRT-010`
  - Evidence: [T10 verification](t10-verification.md); retry in `../../../../../application/usecase/interviewusecase/open_stage.go`. PostgreSQL proof written but skipped (no `TEST_DATABASE_URL`); lifecycle double is pass-through, so no real fence is claimed.
  - depends: T3, T5, T8 — ref: design §3 transactions, §5.
  - Integration follow-up: T18 supplies and verifies the actual locked callback with the interview store; the recorded T10 proof limits remain explicit.
  - DoD: PostgreSQL test shows handoff and source-session close commit together, the target session row is absent until a later successful create, and a fence rejection leaves progress unchanged. A failed target create retries that target only, without a new execution id, and without an invented exhaustion count (exhaustion is T16). Optional summary failure still leaves a readable handoff.

- [x] **T11. Complete or fail at the required record and the hard deadline** — Application — L
  - Ticket ID: `CORE-IRT-011`
  - Evidence: [T11 verification](t11-verification.md); `Workflow.Finish` in `../../../../../application/usecase/interviewusecase/finish.go`. Mocked lifecycle only; no real fence. T18 composes the workflow and lifecycle adapter; driving completion from live finalized events remains dependent on Q1.
  - depends: T3, T5 — ref: design §5 completion, decision-log D8.
  - Integration follow-up: T18 supplies real complete/fail calls without reopening T11's mocked scope.
  - DoD: with the required closing handoff, the operation calls complete once and inserts no report row and no outbox row. At the exact 15-minute instant without that record, it calls fail and the earlier handoff rows remain. A lifecycle rejection does not delete those rows. Clock is fixed in tests. `H-10` report-failure behavior is not claimed here.

### Production integration follow-up

- [x] **T18. Integrate and mount Interview Runtime** — Application / infrastructure / delivery composition — L
  - Ticket ID: `CORE-IRT-018`
  - Evidence: [T18 verification](t18-verification.md). Production composition, durable authority, reservation/replay, schema 11 and real PostgreSQL proof are implemented.
  - depends: T2, T3, T5, T6, T8, T9, T10, T11; existing Execution Lifecycle activation/terminal operations and repository transaction support — ref: design §3 T18 migration and replay handoff, §4 T18 production integration assessment, §9 Q3.
  - Completed checkpoint: design §3–§4 records attempt/generation mapping, command states and failure responses, schema constraints and the owner-approved legacy policy. Real PostgreSQL and fake-provider HTTP proof resolve integration Q3. S1 finality and post-handoff exhaustion remain outside this task.
  - Scope:
    - Supply `service.InterviewLifecycle` through stage-agnostic application operations and infrastructure adapters. Implement the missing minimum durable attempt support, map attempt ids to operation/generation authority, restore that authority after reload, and preserve the frozen activation/reconnect limits and hard deadline. Existing `RecordAttemptOperation` declarations alone are not an implementation.
    - Make execution, progress and command stores join one transaction context. `WithProgress` locks execution before progress/children and rejects terminal state, stale execution version and replaced attempt before invoking the callback. Do not implement a check followed by an independent write.
    - Implement command reservation/finalization and `SessionCommands`; commit scoped identity with the session shell before provider I/O, then atomically persist a successful session result and original replay fields. Reconcile concurrent requests, interrupted setup and safe error mapping; bound cleanup without claiming rollback of external provider work.
    - Add proposed `000011_interview_runtime_integration.{up,down}.sql` from version 10, subject to allocation recheck. Include session id, original answer SDP, execution version and reservation constraints; settle legacy command compatibility before SQL. Update release guard, migration runner, checksums, operational documentation and verification scripts together. No seeds, runtime migration, fabricated SDP backfill or edits to applied migrations.
    - Implement `RevisionReader` and `PinReader` over the immutable execution pin and trusted lifecycle timing. Compose workflow, driver/key, progress store and T8's `Attempt` / `Ready` operations in `infrastructure/runtime`; mount their exact candidate routes. Preserve cookie/origin/CSRF, assignment and controller checks, safe EN/ID errors and T7's prototype-route removal.
  - DoD:
    - Production runtime HTTP tests with a fake provider exercise session-attempt and readiness, retain provider identity, replay the original answer for the same key without a second provider create, and conflict on changed intent. Concurrent same-key calls and interruption/result-persistence failure follow the recorded reservation policy. Readiness changes no coverage.
    - Disposable PostgreSQL proves authority survives reload; replaced attempts, stale versions, terminal executions and rejected callbacks write nothing; callback failure rolls back every joined write. Exercise cancellation versus progress and re-run T10 handoff/source-close rollback cases with the real adapter rather than a pass-through double.
    - Fresh install and clean upgrade from 10 preserve existing pins/progress; compatible legacy command handling and constraints match the assessed design. Disposable down removes only the new additions. Report exact migration/database commands and any skips; skipped suites do not close Q3 or this task's database DoD.
    - `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.docker` or `make test.integration` on an authorized disposable database pass. Run `make mocks` if interfaces change and `make test.security` if dependencies change. Update endpoint availability notes and the API changelog; update OpenAPI together if implementation requires a public-contract change.
    - Record evidence and synchronize this tripack. T18 closes integration Q3 only after the real proof; it does not close T17/H-01, Q1/T12–T15, Q2/T16, Candidate Interview progress/page integration, or reporting.

### Live interview-path acceptance

- [x] **T17. Record one live interview-path voice session** — Verification — M
  - Ticket ID: `CORE-IRT-017`
  - Evidence: [T17 verification](t17-verification.md). One live session through the mounted `session-attempts` route: 201, SDP answer applied, connected, retained session id present, sideband attach succeeded, readiness 200, spoken provider reply confirmed by the owner. Run through a temporary local harness with disposable PostgreSQL (candidate-page voice is T19). The sanitized note was reviewed and approved by the owner `rfanazhari` on 2026-11-03. No key, SDP, transcript or session id is recorded. Q1 is not resolved.
  - depends: T8, T18 — ref: design §9 Q5, scope freeze `H-01` — Q3 resolved by T18; Q5 is the live acceptance evidence this task produces.
  - ready when: T18 has mounted and verified the route, and the live provider/browser setup is available. Execute the session and obtain review of the sanitized note as part of this task, not as a prerequisite to starting it.
  - DoD: the note identifies the route, the retained session id's presence (not the id value), SDP exchanged, and sideband attach. It does not paste the key, SDP, or transcript. It does not mark Q1 resolved. Event-count output from `../../../../../test/livevoice` alone does not satisfy the DoD.

### Candidate page voice integration

- [x] **T19. Start the voice session from the candidate interview page** — Delivery (browser) — L
  - Ticket ID: `CORE-IRT-019`
  - Evidence: partial. RED `db64fec`, GREEN `1a81e7b`: `page.html` starts voice from a Start voice click through the existing routes; `python3 scripts/test_candidate_interview_ui.py` passes 25 headless-Chrome scenarios (8 voice, API/WebRTC fakes), page shell Go test, `make test.all`, `test.contract`, `test.architecture`, `lint` and `test.security` pass. Production composition proven: `TestCandidatePageVoiceStartPostgres` serves the embedded page through the production router and runs the page's start, ready and same-key replay requests against disposable PostgreSQL with a fake provider; `make test.docker` passes (characterization test over existing behavior, no RED observable). One live session through this page on the Compose stack: 201, ready 200, spoken reply confirmed by the owner, cancel 200; sanitized note [T19 verification](t19-verification.md) reviewed and approved by the owner `rfanazhari` on 2026-11-03. Q1, transcript, scoring and report are not claimed.
  - depends: T18 (routes mounted), T17 (live route proof) — ref: design §9 Q5; Candidate Interview page contract (`BR-CORE-040`–`047`) for consent, takeover and cancel.
  - Scope:
    - After the interview is created and the tab holds control, request the microphone, build an `RTCPeerConnection` with the microphone track and the `oai-events` data channel, wait for ICE gathering, and call `POST /candidate/executions/{id}/session-attempts` with cookie, CSRF, controller header, a per-start idempotency key, `intent: activate`, the current execution version and the SDP.
    - Apply the returned answer, then call the attempt `ready` route. Retry only the same idempotency key. Map 409/503 and permission-denied to safe candidate-visible states. Cancel, takeover and page close release local media and the connection.
    - Keep SDP, instructions, keys and session ids out of the DOM, `sessionStorage`, logs and console. Update the candidate-page contract documentation. No backend or migration change unless a contract gap is found first.
  - DoD: browser tests with mocked fetch cover permission denied, 201 then ready, 409, 503, cancel and takeover cleanup. The embedded page is served by the production composition against disposable PostgreSQL and a fake provider; a live provider session through this page is recorded like T17 and reviewed. Q1, transcript, scoring and report behavior are not claimed.

### Blocked on provider event finality (Q1)

Ready when design §3 and §9 Q1 contain an explicit event map taken from a reviewed S1 record: which event is a fragment, which is final speech, how a correction identifies the prior entry, how order is assigned, and how an event from a replaced session is ignored. Writing a guessed map in code does not clear this block.

- [ ] **T12. Record the provider event-finality map in this design** — Design — M
  - Ticket ID: `CORE-IRT-012`
  - depends: none — ref: design §9 Q1 — blocked by: Q1.
  - ready when: the S1 evidence exists.
  - DoD: technical-design, tactical strategy, and this plan agree on the map. Dependent tasks stay blocked until that text lands. No production classifier is added in this task.

- [ ] **T13. Ingest the canonical transcript from classified events** — Application / infrastructure — L
  - Ticket ID: `CORE-IRT-013`
  - depends: T5, T6, T12, T18 — ref: design §3 transcript, `BR-CORE-067` — blocked by: Q1; Q3 resolved by T18.
  - ready when: T12's reviewed map and recorded fixtures are in the design, and T18 supplies the real current-attempt fence and shared transaction.
  - DoD: only events the map calls final become `evidence_state=final`. Fragments persist as fragments and cannot be cited. A mapped correction inserts a new revision. A mapped event whose attempt is not current writes nothing. PostgreSQL test uses recorded fixture payloads from the S1 set, not a live account.

- [ ] **T14. Update coverage from finalized speech** — Application / domain — L
  - Ticket ID: `CORE-IRT-014`
  - depends: T3, T13 — ref: design §3 rules `BR-CORE-061`, `BR-CORE-063`, `BR-CORE-068` — blocked by: Q1.
  - ready when: T13 is done.
  - DoD: a final candidate entry marks `answered` and stores the citation. A technical-loss classification does not omit. One clarification then silence can omit with `no_answer_after_clarification`. A follow-up counts only with a citation and only under the cap. Unmapped events do not change coverage.

- [ ] **T15. Prove one deliberate reconnect on the interview path** — Verification — L
  - Ticket ID: `CORE-IRT-015`
  - depends: T10, T13, T14, T18 — ref: design §5 reconnect, `H-08`, `BR-CORE-069` — blocked by: Q1; Q3 resolved by T18.
  - ready when: T13 and T14 are done and T18 has mounted the workflow with the real lifecycle fence and proved T10's handoff/rollback integration. Live provider/browser setup is available for the deliberate disconnect.
  - DoD: one deliberate disconnect after a committed checkpoint resumes the same execution id, opens a new voice session, repeats no uncommitted utterance as if it were stored, and a fixture event from the old attempt does not change coverage. The test does not claim recovery of unrelated realtime failures. Two episodes remain the stored budget; this task executes one.

### Interview instruction content

- [x] **T20. Give the interviewer its stage context and scope rules** — Application — M
  - Ticket ID: `CORE-IRT-020`
  - Evidence: RED `db616e1`, GREEN `d831c27`; follow-up RED `7c3c058`, GREEN `16fd127`; closing-questions RED `ca413bd`, GREEN `664f782`. `stageInstructions` renders a fixed position, the whole four-stage flow with planned durations and every stage's questions, the current stage, topics, remaining follow-ups, the discussion mode, rules to greet once, name each stage when moving on and close the interview, and interview-only scope rules; `createSession` sends the same text in the provider create request. The model, not the system, paces the stages (todo-gap G7). `make test`, `test.all`, `test.contract`, `test.architecture`, `lint`, `test.security` and `test.docker` pass. Not proven: that the provider applies it or that the model obeys the scope rules; that needs a reviewed live session. Gaps: [todo-gap](todo-gap.md).
  - depends: T8, T18 — ref: design §6.
  - DoD: public workflow tests prove position, plan, stage guidance, greeting only at the first stage, scope rules last, create-time delivery equal to the supplied context, and no ceiling id.

### Interview completion by the model

- [~] **T21. Let the model end the interview, stage by stage** — Application / infrastructure / delivery — L
  - Ticket ID: `CORE-IRT-021`
  - depends: T18, T19, T20 — ref: design §3, §5, §6; [todo-gap](todo-gap.md) G7, G8, G9, G12, G13. Closes G12 and, with it, G7 for completion. Fixes G13.
  - Evidence, application part: RED `f9d0a62`, GREEN `22fa5f3`. `Progress.Attest` seals the current stage with a handoff and makes the successor current without resolved coverage, `AllStagesSealed` reports four sealed stages, and `Workflow.CompleteStage` validates order and the hard deadline, treats an earlier stage as a no-op, and on `closing` seals, closes the local session records and asks the lifecycle to complete. `make test`, `test.all`, `test.contract`, `test.architecture`, `lint`, `test.security` and `test.docker` pass. Not done: the delegation payload, the sideband reader, the tool and the live proof, which wait for the spike below. Nothing calls `CompleteStage` yet.
  - Evidence, finish control: slice 1 RED `9d6d892`, GREEN `cad85a7` and fix `8955cd1` (`CompleteInterview`, provider close by stored session id with sideband reattach); slice 2 RED `0e7e475`, GREEN `a152da8` (`POST /candidate/executions/{id}/complete`, action `complete`, OpenAPI 0.15.0, PostgreSQL run with a fake provider ends `COMPLETED`); slice 3 RED `6eb5b13`, GREEN `61f8396` (page **Selesai** button with confirmation, microphone release, COMPLETED text; 27 browser scenarios). Not yet done: a live session pressing Selesai, and the model trigger (spike, sideband reader).
  - Why: a live session showed stages advance only in conversation, nothing stores progress, and the interview can end only by the candidate pressing Cancel, so a finished interview is recorded `CANCELLED`. The owner chose a model-triggered ending (2026-11-03).
  - Step 0, verification only, ready now: one sanitized live spike on the owner's account. Create a session with Responses delegation and one function tool, confirm the call event reaches the sideband, return a result, close with `session.close` and `session.closed`, and confirm the interviewer's behavior is not degraded by delegation. Record event types only; no key, SDP, text or session ids. It needs no ticket of its own, runs from a temporary harness outside the repository, and the owner runs it because it sends the key to the provider and bills the backend model.
  - Blocked until: the spike result is written into design §3 and todo-gap G8, and the owner decides G9 (what makes a stage complete and what `Authorize` becomes without coverage).
  - Scope after unblock:
    - Add `delegation` and the `complete_stage` tool to the create request. The tool takes only a fixed stage name from the four stages; no free text from the model.
    - Read the sideband for the documented call events and error events, with a bounded lifetime that honors the shutdown budget. Execute each call once, with the sideband as sole owner.
    - Validate each call: the stage is the next one, no stage is skipped, the 15-minute deadline has not passed, and a repeated call is idempotent. A rejected call returns a tool result that tells the model to continue.
    - A `closing` call completes the interview through the lifecycle (`COMPLETED`, not `CANCELLED`), closes the provider session, and returns the tool result so the model can say goodbye. The candidate page shows a completed state.
    - Replace the post-attach `session.update` instructions with the documented append event within 500 tokens, and size recovery context to fit (G13).
    - Update the candidate-page contract, the notice wording only if the owner agrees, and the design.
  - DoD: public workflow tests through mocked ports cover valid order, skipped stage, repeated call, deadline, rejected call and completion; sqlmock tests cover what is persisted; page tests cover the completed state; a PostgreSQL run with a fake provider shows an interview ending as `COMPLETED` and writing nothing on a rejected call; one reviewed live session ends by the model's call. Not claimed: that a stage was answered, transcript, scoring or report content. If the spike shows delegation is unsuitable, replace the trigger with a candidate-visible finish control and record the decision.

### Next-stage retry budget (Q2 decided 2026-11-03)

- [ ] **T16. Enforce the post-handoff activation budget** — Application — M
  - Ticket ID: `CORE-IRT-016`
  - depends: T10 — ref: design §9 Q2 — Q2 decided: 2 attempts × 30 seconds within 90 seconds per stage, no reserve.
  - ready when: design §9 records the attempt count and whether any finalization reserve exists inside the 15 minutes.
  - DoD: the recorded numbers are what the coordinator enforces. Exhaustion fails through the lifecycle and does not open a different stage. If the decision says there is no separate reserve, the only outer timer remains 15 minutes from running, and the test shows that.

### Per-stage session (owner decisions D10 and D11, 2026-11-04)

Source: [decision log](decision-log.md) D10 and D11, and the [per-stage proposal](../interview-recording-transcription/proposal-per-stage-live-transcript.md). T22..T25 are Todo; nothing here is implemented. T21's provider part still owns the stage-end trigger (function call over the sideband).

Rescope note for T21 (2026-11-04): its application part (`Attest`, `CompleteStage`) advances the stage inside the same session. Under D10 that behavior is replaced for non-closing stages by T23; the candidate "Selesai" path keeps working as a finish control. T21's evidence and status are not rewritten by this note.

- [x] **T22. Regenerate the Interview Runtime tripack for per-stage sessions** — Documentation — M
  - Ticket ID: `CORE-IRT-022`
  - Evidence (2026-11-04): documentation only. Technical design v13, tactical strategy v4, decision log D10/D11 and the capability interim notes are aligned. The searched phrases (model-paced stages, whole flow in one prompt, same-session advance, `session.update` switching stages) remain only where labelled as current behavior replaced by T23..T25, as a rejected alternative, or in revision history and recorded task evidence. `git diff --check` is clean and relative links in the changed files resolve. No code, test, schema or provider call was run; T23..T25 are not implemented. At that time the proposed D6 and D8 were not yet accepted; they were accepted later on 2026-11-04 as D12 and D13.
  - depends: none (D10 and D11 are accepted) — ref: decision log D10, D11; todo-gap G7, G9, G10, G14.
  - Scope: regenerate technical design sections 3, 5, 6 and 9, the tactical strategy and this plan so they match D10 and D11; add T21, `Attest`/`CompleteStage` and G9 to the strategy where missing; keep the interim note on `BR-CORE-064`/`BR-CORE-065` in the capability document in step; keep Done scopes and evidence as recorded.
  - DoD: no document still says the model paces stages in one prompt, that `session.update` switches stages, or that a handoff needs coverage under the interim rule. Rule-to-task mapping reflects T23..T25. No behavior change.

- [ ] **T23. Seal a stage without advancing; open the next stage in a new session** — Domain / application — L
  - Ticket ID: `CORE-IRT-023`
  - depends: T22 — ref: D10, D11; `domain/interview/progress_attest.go`, `interviewusecase/open_stage.go`, `complete_stage.go`.
  - Why: today `Attest` seals and advances `CurrentStage` in one step. `OpenStage` then takes its `CurrentStage == target` branch, and `releaseUnprovisioned` returns a conflict when the open session already holds a provider id; `appendSession` also refuses a new session while one is open. The two do not compose without change.
  - Scope: separate sealing the source stage from advancing; close the source session before the next session row is appended; keep `ActivateStage`'s precondition (handoff count equals stage index plus one); keep the stage clock, order and hard-deadline checks; keep idempotent repeated statements; keep `CompleteInterview` ("Selesai") sealing every remaining stage and completing, without needing a session per stage.
  - DoD: public workflow tests through mocked ports for seal, advance in a new session, a repeated statement, a skipped stage, the deadline, and "Selesai" from each stage; sqlmock for what is persisted; PostgreSQL proof for rollback and the one-open-session index. Controlled time for stage-clock and deadline equality.

- [ ] **T24. Driver waits for session close confirmation** — Infrastructure — M
  - Ticket ID: `CORE-IRT-024`
  - depends: T22 and decision D12 (accepted 2026-11-04). The timeout value is not set and needs a measurement or an owner value before the code can be finished — ref: D10; `interviewvoice/driver_sideband.go`.
  - Why: `Driver.Close` sends `session.close` and releases the connection at once, so events after the request are lost and finalization is unconfirmed.
  - Scope: a bounded wait for the provider's close confirmation (`session.closed`), an explicit "finalization unconfirmed" outcome on timeout, and release of the connection afterwards. On timeout the interview continues, the session is flagged, and recovery uses the stored recording; success is never reported for that session (D12). The wait and timeout value are chosen with evidence; no value is proposed here. Provider event names are confirmed against the reviewed record, not assumed.
  - DoD: tests with a fake sideband cover confirmation, timeout, and a late event; bounded by the shutdown budget; documented declarations.

- [ ] **T25. Revise the first-session instructions** — Application — M
  - Ticket ID: `CORE-IRT-025`
  - depends: T22; the pacing wording also waits for the T21 trigger — ref: D10; prompting review in the per-stage proposal (section 5a); todo-gap G3, G13.
  - Scope: instructions written in the spoken language in the formal register ("Anda", D14); role and tone, interruption and unclear-audio rules (clarify names, dates, numbers; keep listening during pauses); short-turn guidance; one stage per prompt plus committed handoff context for stages after the first; the greeting stays an instruction line that the model may paraphrase (D14: consent is taken on the page before the session); keep the scope rules, the salary rule and the closing-stage rule. Whether the second delivery after attach is needed is settled with G3/G13.
  - DoD: rendered-text tests prove which rules are present and that the ceiling is never rendered (existing T9/T20 coverage kept); one spike conversation in the spoken language covers a thinking pause, an unclear number, background noise, an interruption and an off-topic question. Model behavior is not claimed from tests.

## Dependency Graph

```text
T1 → T3
T2 → T3
T1 → T4 → T5
T2 → T6
T1 → T9
T2 → T9

T3 → T8
T5 → T8
T6 → T8
T8 → T7

T3 → T10
T5 → T10
T8 → T10
T10 → T16

T3 → T11
T5 → T11

T8 → T17
T2, T3, T5, T6, T8, T9, T10, T11 → T18
T18 → T17
T17, T18 → T19
T18, T19, T20 → T21
T22 → T23
T22 → T24
T22, T21 → T25
T18 → T13
T18 → T15

T12 → T13 → T14 → T15
T5 → T13
T6 → T13
T10 → T15
```

T1–T11 retain their Done scopes. T18 is Done and resolves integration Q3. T12's S1 evidence collection can proceed as investigation, but the finality map remains blocked until reviewed evidence exists. T13–T15 stay blocked on Q1; T17 is Done with reviewed live evidence; Q1 stays open. T16 stays blocked on Q2 and can use T18's real lifecycle adapter when that policy is settled.

## Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial plan, all tasks todo | Mode A. No interview-runtime implementation is claimed. |
| v2 | 2026-11-03 | Add T18 / CORE-IRT-018 for T8's deferred integration; update T13/T15/T17 dependencies and T17 start condition | Preserve T1–T11 Done scopes while giving adapter, authority, command/migration, wiring and real PostgreSQL proof a named owner. Q1/Q2 remain open |
| v3 | 2026-11-03 | Complete T18 with durable authority, joined transactions, reservation/replay, schema 11 and real PostgreSQL proof | Resolve integration Q3 only; preserve Q1/Q2, T17 live acceptance and earlier task evidence limits |
| v4 | 2026-11-03 | Add T19 / CORE-IRT-019 for candidate-page voice start (microphone, SDP, ready) | T17 proved the mounted routes only through a temporary harness; the production page had no owner for the browser side. Number proposed, pending owner ticket confirmation. Q1/Q2 unchanged |
| v5 | 2026-11-03 | Complete T17 with one reviewed live interview-path session | Q5 live acceptance recorded through the mounted route; Q1/Q2, T12–T16 and T19 unchanged |
| v6 | 2026-11-03 | Add T20 / CORE-IRT-020 for interview instruction content and create-time delivery; Q2 decided so T16 is ready | Live session showed an off-context interviewer. T20 changes prompt content only; Q1 and T12–T15 unchanged |
| v7 | 2026-11-03 | T20 follow-up: one prompt with the whole flow and named transitions | No code moves stages today (todo-gap G7); the prompt must not wait for a system signal |
| v8 | 2026-11-03 | T20 follow-up: closing-stage candidate questions are noted for the recruitment team, not answered | Live session; the note is wording only until the end-of-interview transcript exists (todo-gap G11) |
| v9 | 2026-11-03 | Add T21 / CORE-IRT-021 for model-triggered stage and interview completion | A live session showed no code moves stages or ends the interview; the owner chose a model-triggered ending. Blocked on one provider spike and on todo-gap G9. Q1, Q2 and T12–T16 unchanged |
| v10 | 2026-11-03 | T21 application part: model-attested stage sealing and interview completion through `CompleteStage` | Owner decided G9 (a model statement with order, stage clock and deadline checks is enough). The provider-facing part waits for the spike |
| v11 | 2026-11-03 | T21 candidate-visible finish control (route, action, page button) closes G12 for the normal ending | Owner approved the Selesai flow after a finished interview could not close. The model trigger stays for the spike |
| v12 | 2026-11-04 | Add T22..T25 / CORE-IRT-022..025 for per-stage sessions; rescope note on T21 | Owner accepted D10 (one provider session per stage) and D11 (interim model-stated stage seal) on 2026-11-04. T21 evidence, T12..T16 and earlier tasks unchanged |
| v13 | 2026-11-04 | T22 / CORE-IRT-022 Done: tripack regenerated for D10 and D11 | Documentation only. T23..T25 stay Todo; Done scopes and evidence of T1..T21 unchanged |
| v14 | 2026-11-04 | Record owner decisions D12 (bounded close-confirmation wait), D13 (transition signal by polling) and D14 (formal register, paraphrasable greeting); update T24 and T25 | T24 is no longer waiting for a decision, only for the timeout value. Statuses of T22..T25 unchanged except as stated |
