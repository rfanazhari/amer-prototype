# Core AI

Core AI is one Go modular monolith. Identity and Access, Agent Configuration and Publication, Execution Lifecycle T1–T9, and Candidate Interview T1–T11 are implemented; startup accepts clean schema version 9. Candidate Interview continues after the candidate already has a session, with runtime progress and the combined end-to-end proof still pending. The candidate UI is in scope. This phase has no admin UI. Document Analyzer is outside this phase.

## Done

### Identity and Access

ADMIN provisions HR, and an invited candidate reaches one assignment without a permanent account. Rules `BR-CORE-020`–`BR-CORE-039`: [Identity and Access](business/capabilities/identity-access.md). Tripack: [identity-access](identity-access/task-plan.md).

The task plan records T1–T14, T16, and T18–T20 Done, including the published session, HR-account, candidate, invitation, exchange, and candidate-session routes. Retention (T15) stays blocked, T17 is still marked in progress, and the creator-scoped candidate list (T21) is in progress with its public contract outstanding. T22 (`CORE-IAM-022`) is planned and not started: HR revocation writes `access_revoked_at`, then calls `CancelNonterminal`.

### Agent Configuration and Publication

HR configures and publishes the HR Interviewer and Document Analyzer templates, then pins an interview assignment to the current published revision. Rules `BR-CORE-001`–`BR-CORE-014`: [Agent Configuration and Publication](business/capabilities/agent-configuration-publication.md). Tripack: [agent-configuration-publication](agent-configuration-publication/task-plan.md).

T1–T14 are recorded Done, including the published configuration routes, instance list, and assignment create. Execution Lifecycle subsequently added the shared execution tables in version 8; Candidate Interview raised the current startup floor to clean version 9. There is no admin UI.

### Candidate Interview — T1–T11 Done

Rules `BR-CORE-040`–`BR-CORE-047`: [Candidate Interview](business/capabilities/candidate-interview.md). Source of status: [task plan](candidate-interview/task-plan.md). The feature is partially complete: T1–T11 (`CORE-CIN-001`–`011`) are Done; T12–T13 remain open.

Implemented:

- **Notice, consent and storage (T1–T5):** approved notice `notice-261102`, explicit acceptance before start, acceptance replay, consent/coverage/controller domain rules and application ports. Migration `000009_candidate_interview` adds `interview_consents` and `interview_controllers`; the current clean schema floor is 9.
- **Candidate API and controller ownership (T6–T9):** interview read, consent acceptance, execution start/poll/takeover/cancel, strict input/session/origin/CSRF checks, safe responses, public errors, OpenAPI and endpoint documentation. Controller takeover persists one versioned controller per assignment and rejects stale updates.
- **Candidate screen (T10):** notice and accept/start actions, per-tab identity and retry intent in `sessionStorage`, read-only/takeover behavior, confirmed cancel, polling, and distinct `COMPLETED`, `FAILED`, and `CANCELLED` outcomes. Refresh only reads; the screen does not request or wait for an HR report.
- **Real lifecycle integration (T11):** production start/read/cancel adapters, one execution after concurrent starts or a lost 201, conflict on changed intent, a stable deadline from persisted activation plus 15 minutes, committed cancellation followed by `CANCELLED` on GET, completed-execution replay, and refusal after assignment revocation. Notice approval and lifecycle availability no longer block this feature.

Remaining work and blockers:

| Task / dependency | Not yet implemented or proved | Blocker / ready when |
|---|---|---|
| T12 — `CORE-CIN-012` | Connect candidate-safe progress to polling and prove that a previous controller cannot advance after takeover. Stored takeover is implemented; runtime advance fencing is still pending. | Q4: [Interview Runtime](interview-runtime/task-plan.md) must supply the candidate-safe progress object and enforce the current controller on advances. Its tasks remain Todo. |
| T13 — `CORE-CIN-013` | Prove the combined candidate page, runtime and disposable PostgreSQL flow: consent, lost-start recovery, coverage boundary, takeover/fencing, cancel, refresh and terminal display without a report. | T12. T11 proves production HTTP composition with PostgreSQL, while T10 browser tests use API fixtures; those separate proofs do not establish the combined flow. |
| Cross-feature access-ended cancellation | HR-triggered assignment revocation followed by lifecycle cancellation, and the driver for session expiry during an interview. | Identity and Access T22 (`CORE-IAM-022`) owns the HR revocation command and is not started; session expiry remains Identity and Access Q7. Candidate refusal with 401 is already implemented, but 401 itself does not commit cancellation. These are sibling obligations, not new Candidate Interview tasks. |

Verification recorded by T11: `make test` and `make check` passed; sqlmock covers SQL composition and Mockery mocks cover interfaces. `GOFLAGS='-run=^TestCandidateLifecyclePostgres$' make test.integration` passed on disposable PostgreSQL 16, including eight concurrent starts, retry/conflict, cancel/poll, deadline restoration, completed replay and revocation with an explicit `CancelNonterminal` call. Latest checkpoints: RED `2acb6bb`, GREEN `775e3ff`; [execution evidence and assessment](candidate-interview/t11-execution.md). The full database suite and `make test.docker` were not run for T11. Earlier T4/T5/T8 runs recorded pre-existing fixture/schema interference in the full integration suite; that limitation has not been re-verified or resolved by T11.

Stages, microphone/SDP, voice session readiness, transcript ingestion and runtime-driven completion remain Interview Runtime work. HR reports remain Interview Reporting work. They are not implemented by the candidate page or lifecycle adapter.

## Next

Execution Lifecycle and Candidate Interview have implemented foundations with remaining integration work. Interview Runtime and Interview Reporting remain planned.

1. **Execution lifecycle.** One admitted run keeps a stable identity and an honest ending; a repeated start finds the same run. Rules `BR-CORE-050`–`BR-CORE-059`: [Execution Lifecycle](business/capabilities/execution-lifecycle.md). Tripack: [execution-lifecycle](execution-lifecycle/task-plan.md). Migration `000008_execution_lifecycle` raises the clean floor to 8. It needs the current version-7 execution pin, and nothing from the three capabilities below. Candidate start and interview runtime both need one real execution, so this is first. T1–T9 are done. T10 stays blocked until Identity and Access T22 (`CORE-IAM-022`) is done. In-interview session expiry is Identity and Access Q7, not T10. T11 stays blocked on a future Document Analyzer.

2. **Candidate interview.** T1–T11 are Done, including consent, the candidate screen, persistent takeover and real lifecycle start/read/cancel. T12 waits on Interview Runtime candidate-safe progress and controller fencing; T13 then proves the combined page/runtime/database slice. See the completed work, blockers and verification limits above, and the [task plan](candidate-interview/task-plan.md).

3. **Interview runtime.** The live interview asks the published questions, records gaps, advances only on a committed handoff, and finishes only when the required record is stored. Rules `BR-CORE-060`–`BR-CORE-071`: [Interview Runtime](business/capabilities/interview-runtime.md). Tripack: [interview-runtime](interview-runtime/task-plan.md). Migration `000010_interview_runtime` raises the floor to 10, after clean version 9. Activation and session HTTP wait on execution lifecycle. Transcript ingest and one deliberate reconnect stay blocked until provider event finality is written from reviewed evidence; live H-01 waits on a sanitized live record, and a fake provider may proceed earlier.

4. **Interview reporting.** HR receives a written account of a finished interview that does not hire, reject, or rank the candidate, and a report failure must not change a completed interview. This is an HR API only, with no new admin screen. Rules `BR-CORE-080`–`BR-CORE-086`: [Interview Reporting](business/capabilities/interview-reporting.md). Tripack: [interview-reporting](interview-reporting/task-plan.md). Migration `000011_interview_reporting` raises the floor to 11, after clean version 10 and a completed interview from the runtime. Dead-letter topology, the report model contract, the finalized-evidence adapter, and process registration are blocked. Schema and completion-plus-outbox work can follow the runtime.

## Suggested build order

Execution Lifecycle T1–T9 and Candidate Interview T1–T11 are already implemented, through migration `000009_candidate_interview` and clean floor 9. Continue Interview Runtime (`000010_interview_runtime`, planned floor 10), integrate its candidate-safe progress and controller fence in Candidate Interview T12, then close T13 with the combined flow proof. Interview Reporting follows (`000011_interview_reporting`, planned floor 11). Identity and Access T22 separately supplies the real HR revocation/cancellation caller needed by Execution Lifecycle T10. Future migration numbers remain ordered; startup stays at clean version 9 until a later migration is implemented.

## Progress history

Status per feature, taken from each tripack's task plan. Newest entry first. Update this table when a task plan changes status.

| Date | Feature | Progress | Open items |
|---|---|---|---|
| 2026-11-02 | Execution lifecycle | T1–T9 Done (`CORE-EXL-001`–`009`). Includes migration `000008`, the GORM execution store, start, activation, terminal commits, `Cancel` and `CancelNonterminal`, PostgreSQL proof of locking, rollback and idempotency (T8), and the architecture, contract and lint gates (T9). Latest checkpoints: T9 RED `b0b1c91`, T9 GREEN `941e818`. | T10 (`CORE-EXL-010`) stays blocked until Identity and Access T22 (`CORE-IAM-022`) is Done. That task is the approved writer of `access_revoked_at` and the caller of `CancelNonterminal`; it is planned, not built. In-interview session expiry is Identity and Access Q7 and is not T10. T11 (`CORE-EXL-011`) waits on a future Document Analyzer. |
| 2026-11-02 | Candidate interview | T1–T11 Done (`CORE-CIN-001`–`011`): approved notice, consent, migration `000009` / floor 9, APIs, controller takeover, candidate UI and real lifecycle start/read/cancel. T11 unit/check gates and filtered PostgreSQL integration pass; RED `2acb6bb`, GREEN `775e3ff`. | T12 waits on runtime candidate-safe progress and controller fencing (Q4); T13 waits on T12 for the combined page/runtime/database proof. HR revocation caller is Identity and Access T22; session-expiry driver remains its Q7. Full database-suite and Docker-stack verification remain unproved by T11. |
| 2026-11-02 | Interview runtime | Planned. Nothing implemented. | Activation and session HTTP can use the lifecycle. Transcript ingest and reconnect wait on provider event finality. |
| 2026-11-02 | Interview reporting | Planned. Nothing implemented. | Waits on a completed interview from the runtime. Dead-letter topology, report model contract, evidence adapter, and process registration are blocked. |
| 2026-11-02 | Agent configuration and publication | T1–T14 Done. The version-8 floor recorded at that checkpoint is superseded by Candidate Interview version 9. | None recorded here. |
| 2026-11-02 | Identity and access | T1–T14, T16, T18–T20 Done. T22 (`CORE-IAM-022`) is planned: HR revocation writes `access_revoked_at`, then calls `CancelNonterminal`. | T15 retention blocked, T17 in progress, T21 in progress with its public contract outstanding. T22 is not started. |
