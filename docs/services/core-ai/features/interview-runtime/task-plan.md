# Task Plan: Interview Runtime

## Ticket Identity

Capability group: **IRT — Interview Runtime**. Ticket prefix: `CORE-IRT`. These are repository planning IDs. No external tracker tickets have been created.

Stable map: `T1` → `CORE-IRT-001` through `T19` → `CORE-IRT-019`. Later revisions add ids. They do not renumber these.

Companion dependencies, not implemented by this plan:

- [Execution Lifecycle](../../business/capabilities/execution-lifecycle.md) `BR-CORE-050`..`BR-CORE-059` for activation, fencing, and completion.
- [Candidate Interview](../../business/capabilities/candidate-interview.md) for the page, consent, takeover, and cancel.
- [Interview Reporting](../../business/capabilities/interview-reporting.md) for the report after completion.
- [Agent Configuration and Publication](../agent-configuration-publication/technical-design.md) for the pinned revision this slice reads.

## Legend

- ID: `T1`..`T19`, stable
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

- [~] **T19. Start the voice session from the candidate interview page** — Delivery (browser) — L
  - Ticket ID: `CORE-IRT-019`
  - Evidence: partial. RED `db64fec`, GREEN `1a81e7b`: `page.html` starts voice from a Start voice click through the existing routes; `python3 scripts/test_candidate_interview_ui.py` passes 25 headless-Chrome scenarios (8 voice, API/WebRTC fakes), page shell Go test, `make test.all`, `test.contract`, `test.architecture`, `lint` and `test.security` pass. Production composition proven: `TestCandidatePageVoiceStartPostgres` serves the embedded page through the production router and runs the page's start, ready and same-key replay requests against disposable PostgreSQL with a fake provider; `make test.docker` passes (characterization test over existing behavior, no RED observable). Still pending: one reviewed live session through this page. Not Done.
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

### Blocked on the next-stage retry budget (Q2)

- [ ] **T16. Enforce the post-handoff activation budget** — Application — M
  - Ticket ID: `CORE-IRT-016`
  - depends: T10 — ref: design §9 Q2 — blocked by: Q2.
  - ready when: design §9 records the attempt count and whether any finalization reserve exists inside the 15 minutes.
  - DoD: the recorded numbers are what the coordinator enforces. Exhaustion fails through the lifecycle and does not open a different stage. If the decision says there is no separate reserve, the only outer timer remains 15 minutes from running, and the test shows that.

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
