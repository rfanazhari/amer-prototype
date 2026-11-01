# Task Plan: Interview Runtime

## Ticket Identity

Capability group: **IRT — Interview Runtime**. Ticket prefix: `CORE-IRT`. These are repository planning IDs. No external tracker tickets have been created.

Stable map: `T1` → `CORE-IRT-001` through `T17` → `CORE-IRT-017`. Later revisions add ids. They do not renumber these.

Companion dependencies, not implemented by this plan:

- [Execution Lifecycle](../business/capabilities/execution-lifecycle.md) `BR-CORE-050`..`BR-CORE-059` for activation, fencing, and completion.
- [Candidate Interview](../business/capabilities/candidate-interview.md) for the page, consent, takeover, and cancel.
- [Interview Reporting](../business/capabilities/interview-reporting.md) for the report after completion.
- [Agent Configuration and Publication](../agent-configuration-publication/technical-design.md) for the pinned revision this slice reads.

## Legend

- ID: `T1`..`T17`, stable
- Complexity: S (< 2h) | M (2–4h) | L (> 4h)
- Status: `[ ]` Todo | `[~]` In Progress | `[x]` Done | `~~obsolete~~`
- ref: [technical-design.md](technical-design.md) section
- DoD: verifiable
- Blocked: not executable until the ready condition is true and this design is updated when the condition changes the contract

Every Go task also requires `gofmt` on changed Go files, comments on new declarations so `make test.architecture` passes, and `make lint` clean for the touched packages. Domain and application tasks run `make test`. None of the tasks below are done.

## Tasks

### Independent foundation

- [ ] **T1. Encode interview progress rules** — Domain — L
  - Ticket ID: `CORE-IRT-001`
  - depends: none — ref: design §3–§4, decision-log D1, D5, D6, D7, D9.
  - DoD: public tests cover coverage edges, the three omission reasons, technical loss leaving the question un-omitted and blocking `Authorize`, follow-up cap 2 with citation required, fragment rejected as an answer, handoff required before a target stage, closing handoff with no next stage, soft-budget and 15-minute equality, reconnect counters, and `ReadyForCompletion`. `NewProgress` writes cap 2 from a revision fixture that has no cap field. `RestoreProgress` keeps a stored cap and does not call publication assessment. No execution status type, no provider client, no GORM. `make test` and `make test.architecture` pass.

- [ ] **T2. Define lifecycle, driver, and progress ports** — Application — M
  - Ticket ID: `CORE-IRT-002`
  - depends: none — ref: design §1 lifecycle table, §4.
  - DoD: interfaces exist for the lifecycle calls, the driver operations, and the progress store. Signatures carry execution id, attempt id, expected version, and budgets 2×30s/90s, 120s, 2 attempts, 2 episodes. No HR stage field appears on the lifecycle port. Mockery config generates the mocks. `make mocks` output is committed as generated. Documentation comments exist. This task does not call a real provider or a real database.

### Workflow and schema that do not need event names

- [ ] **T3. Orchestrate proposals, handoff order, and budgets with mocks** — Application — L
  - Ticket ID: `CORE-IRT-003`
  - depends: T1, T2 — ref: design §5–§6, decision-log D1, D2, D3, D4.
  - DoD: mocked tests show an unauthorized proposal writes nothing; a successful proposal commits the handoff before any create-session call; summary failure still commits structured evidence; reconnect uses the same execution id and a new session; an old attempt id is dropped; activation passes the section 10 budget into the lifecycle port; the third activation attempt is not opened. Clock is injected. No live provider and no claim that a real fence exists.

- [ ] **T4. Add SQL version 10** — Infrastructure schema — M
  - Ticket ID: `CORE-IRT-004`
  - depends: T1 — ref: design §3.
  - DoD: `migrations/pgsql/000010_interview_runtime.up.sql` and `.down.sql` match the eight-table design, including checks, partial unique indexes, and FKs to `interview_executions` and `interview_progress`. No FK to a missing attempt table. No seed inserts. `schema.ReleaseVersion` becomes 10 and the runner accepts target 10 only after this task. Startup still does not migrate. Disposable PostgreSQL proof, via `make test.docker` or an authorized integration database: clean upgrade from version 9 keeps existing pins, fresh up creates the tables, down drops only the new tables. If `TEST_DATABASE_URL` is absent, say the database proof was skipped. Do not describe that skip as a pass.

- [ ] **T5. Persist classified progress with GORM** — Infrastructure — L
  - Ticket ID: `CORE-IRT-005`
  - depends: T1, T4 — ref: design §3 adapter and transactions.
  - DoD: store maps progress, coverage, sessions, handoffs, checkpoints, technical losses, and transcript rows that the caller already marked fragment or final. Explicit `Select` persists `clarification_count` 0 and null `closed_at` / `summary`. Rollback of a handoff leaves the source stage. Unique open session and unique handoff per stage are enforced by PostgreSQL tests. Restore of a corrupt state is a technical error. The store does not interpret provider event names. Same database-proof rule as T4.

- [ ] **T6. Build the GPT-Live driver transport** — Infrastructure — L
  - Ticket ID: `CORE-IRT-006`
  - depends: T2 — ref: design §4 driver, §1 prototype table.
  - DoD: `interviewvoice` implements create, attach, observe, supply context, control, and close against the driver port. Create retains the provider session id and the SDP answer. A fake HTTP server covers success, empty key, and non-success. Close failure is returned and does not imply a database rollback. Observe yields an opaque event and does not mark coverage. Package tests do not call the public network. `H-01` is not claimed. RabbitMQ is not imported.

- [ ] **T9. Build stage instructions without the salary ceiling** — Application — M
  - Ticket ID: `CORE-IRT-009`
  - depends: T1, T2 — ref: design §6, decision-log D5, D6.
  - DoD: the context passed to `supply context` includes current-stage question text, topics, role name, remaining follow-ups, and `expectations_only`. The demo ceiling id and the field name `salaryCeilingRef` are absent. No amount parameter exists. Instructions do not ask the model for a hire or reject result. A test feeds a candidate sentence that asks for the ceiling and the outbound context still omits it. `make test` passes.

### Blocked on Execution Lifecycle (Q3)

Ready when Execution Lifecycle exposes activation, current-attempt fencing, complete, and fail, plus a same-transaction callback that checks nonterminal state, expected version, and attempt authority before the interview write. The operations must not require interview stages. Update this design's port names if the published signatures differ, before starting these tasks.

- [ ] **T8. Mount session-attempt and readiness HTTP** — Delivery / application — L
  - Ticket ID: `CORE-IRT-008`
  - depends: T3, T5, T6 — ref: design §2, §7 — blocked by: Q3.
  - ready when: the lifecycle activate and fence calls exist and are wired.
  - DoD: `POST /candidate/executions/{id}/session-attempts` and `POST .../ready` match design §2. Candidate cookie and CSRF are required. Another assignment's execution is 404. Same idempotency key replays; a different intent conflicts. `ready` does not set coverage to asked or answered. Handler tests mock the operation. OpenAPI, `docs/api/CHANGELOG.md`, and an interview endpoint note are updated. `make test.contract` passes. English and Indonesian messages exist for `IRT_CONFLICT` and `IRT_NOT_READY`.

- [ ] **T7. Remove the prototype voice routes from the process** — Delivery / runtime / contract — M
  - Ticket ID: `CORE-IRT-007`
  - depends: T8 — ref: design §2 prototype removal — blocked by: Q3, because it ships in the same change as T8.
  - ready when: T8 is ready to mount.
  - DoD: `infrastructure/runtime` no longer serves `GET /voice-session` or `POST /voice-sessions`. OpenAPI and `docs/api/voice-session.md` no longer describe them as available routes. Contract tests and router tests match that removal. `test/livevoice` still runs only as an opt-in probe. No interview call path remains on the old URLs.

- [ ] **T10. Commit handoff and open the next session under the lifecycle lock** — Application / infrastructure — L
  - Ticket ID: `CORE-IRT-010`
  - depends: T3, T5, T8 — ref: design §3 transactions, §5 — blocked by: Q3.
  - ready when: the locked callback can include the interview store.
  - DoD: PostgreSQL test shows handoff and source-session close commit together, the target session row is absent until a later successful create, and a fence rejection leaves progress unchanged. A failed target create retries that target only, without a new execution id, and without an invented exhaustion count (exhaustion is T16). Optional summary failure still leaves a readable handoff.

- [ ] **T11. Complete or fail at the required record and the hard deadline** — Application — L
  - Ticket ID: `CORE-IRT-011`
  - depends: T3, T5 — ref: design §5 completion, decision-log D8 — blocked by: Q3.
  - ready when: lifecycle complete and fail exist.
  - DoD: with the required closing handoff, the operation calls complete once and inserts no report row and no outbox row. At the exact 15-minute instant without that record, it calls fail and the earlier handoff rows remain. A lifecycle rejection does not delete those rows. Clock is fixed in tests. `H-10` report-failure behavior is not claimed here.

- [ ] **T17. Record one live interview-path voice session** — Verification — M
  - Ticket ID: `CORE-IRT-017`
  - depends: T8 — ref: design §9 Q5, scope freeze `H-01` — blocked by: Q3 and Q5.
  - ready when: T8 is mounted and a reviewer accepts a sanitized feasibility note that shows a provider session id on this route.
  - DoD: the note identifies the route, the retained session id's presence (not the id value), SDP exchanged, and sideband attach. It does not paste the key, SDP, or transcript. It does not mark Q1 resolved. Event-count output from `test/livevoice` alone does not satisfy the DoD.

### Blocked on provider event finality (Q1)

Ready when design §3 and §9 Q1 contain an explicit event map taken from a reviewed S1 record: which event is a fragment, which is final speech, how a correction identifies the prior entry, how order is assigned, and how an event from a replaced session is ignored. Writing a guessed map in code does not clear this block.

- [ ] **T12. Record the provider event-finality map in this design** — Design — M
  - Ticket ID: `CORE-IRT-012`
  - depends: none — ref: design §9 Q1 — blocked by: Q1.
  - ready when: the S1 evidence exists.
  - DoD: technical-design, tactical strategy, and this plan agree on the map. Dependent tasks stay blocked until that text lands. No production classifier is added in this task.

- [ ] **T13. Ingest the canonical transcript from classified events** — Application / infrastructure — L
  - Ticket ID: `CORE-IRT-013`
  - depends: T5, T6, T12 — ref: design §3 transcript, `BR-CORE-067` — blocked by: Q1.
  - ready when: T12's map is in the design.
  - DoD: only events the map calls final become `evidence_state=final`. Fragments persist as fragments and cannot be cited. A mapped correction inserts a new revision. A mapped event whose attempt is not current writes nothing. PostgreSQL test uses recorded fixture payloads from the S1 set, not a live account.

- [ ] **T14. Update coverage from finalized speech** — Application / domain — L
  - Ticket ID: `CORE-IRT-014`
  - depends: T3, T13 — ref: design §3 rules `BR-CORE-061`, `BR-CORE-063`, `BR-CORE-068` — blocked by: Q1.
  - ready when: T13 is done.
  - DoD: a final candidate entry marks `answered` and stores the citation. A technical-loss classification does not omit. One clarification then silence can omit with `no_answer_after_clarification`. A follow-up counts only with a citation and only under the cap. Unmapped events do not change coverage.

- [ ] **T15. Prove one deliberate reconnect on the interview path** — Verification — L
  - Ticket ID: `CORE-IRT-015`
  - depends: T10, T13, T14 — ref: design §5 reconnect, `H-08`, `BR-CORE-069` — blocked by: Q1 and Q3.
  - ready when: T10, T13, and T14 are done.
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

T12 → T13 → T14 → T15
T5 → T13
T6 → T13
T10 → T15
```

T1, T2, and T12's evidence collection can proceed as investigation. T12 stays blocked for implementation of a map until Q1 is ready. T8, T7, T10, T11, T15, and T17 stay blocked on Q3. T13, T14, and T15 stay blocked on Q1. T16 stays blocked on Q2.

## Revision History

| Version | Date | Change | Reason |
|---|---|---|---|
| v1 | 2026-11-02 | Initial plan, all tasks todo | Mode A. No interview-runtime implementation is claimed. |
