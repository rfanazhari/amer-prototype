# Task Plan: Interview Recording and Transcription

Planning baseline: 2026-11-03. Mode B. All tasks are Todo; no implementation or verification pass is claimed.

## Identity and Legend

Feature shorthand: **IRTREC** (distinct from Interview Runtime `IRT` and Reporting `IRP`). Local task IDs T1–T11 are stable. No external tickets or implementation ticket IDs have been allocated; obtain the required ticket identity before a gocraft implementation task. Do not reuse IRP T13 for this work.

Status `[ ]` Todo, `[~]` In Progress, `[x]` Done. Complexity S (<2h), M (2–4h), L (>4h), subject to engineering review. `ref` points to [technical-design.md](technical-design.md). Blocked tasks are not executable until their named readiness condition is synchronized in the design. Every Go task requires documented declarations, gofmt, focused tests, architecture and lint checks.

## Tasks

### Independent investigation and authority

- [ ] **T1. Verify stored Live recording and transcription contracts** — Infrastructure investigation — M
  - depends: none — ref: design §1, §6, §9 Q1.
  - DoD: inspect effective APIs and clientmanager version; record a sanitized synthetic Live session with `store: true`, close/finalization observation and stereo WAV download. Verify project eligibility, pending/unavailable mapping, stream limits and multipart/audio log suppression. Select and prove a transcription/segment alignment contract preserving both speakers and offsets across chunks. Synchronize exact provider schema and measured constraints in §6; a text-only transcript or fake server does not close this task. If live access is unavailable, report the proof blocked rather than Done.

- [ ] **T2. Resolve recording policy and reconcile business/dispatch authority** — Design / product / platform — M
  - depends: none — ref: design §9 Q2–Q4; decision-log D1–D3.
  - DoD: record notice version/consent treatment, provider storage eligibility, object/temp storage, byte/duration caps, retention/deletion and rollout cutoff. Approve dedicated broker topology and malformed-message handling. Reconcile BR-CORE-065/070 with accepted post-completion finalization, retaining old decision history and IRT completed scopes. G14's old primary-source direction is superseded, not silently deleted. Do not resolve reporting Q2 by selecting a transcription model.

### Foundation (blocked until contracts are ready)

- [ ] **T3. Encode post-completion transcription rules and ports** — Domain / application — M
  - depends: T1, T2 — ref: design §2–§4, §6.
  - blocked by: Q1 timing contract and Q2 policy. Ready when T1/T2 settle the affected fields and transitions.
  - DoD: public domain tests prove completeness, legacy unavailability, revision immutability, duplicate sources/chunks, token and deadline boundaries, and no lifecycle/coverage mutation. Define stream ownership and source/chunk checkpoint contracts. Mockery regenerates interfaces through `make mocks`; `make test` and architecture pass.

- [ ] **T4. Add the allocated manual migration and release artifacts** — Infrastructure schema — M
  - depends: T3 — ref: design §3.
  - DoD: recheck migration head, allocate proposed `000014_interview_recording_transcription` or the actual next pair, add five metadata/outbox tables and nullable session recording flag with defined checks/FKs/indexes. Update checksums, clean release floor, grants, runner/script limits and runbook. Preserve versions 1–13. No AutoMigrate or seed. Real disposable PostgreSQL up/down/up preserves legacy transcript rows; absent database proof is skipped and prevents Done.

- [ ] **T5. Persist admission, checkpoints and final evidence atomically** — Infrastructure repository — L
  - depends: T3, T4 — ref: design §3, §5.
  - DoD: GORM CRUD and justified locked claims implement the store; sqlmock checks SQL, explicit zero/null writes and scoped bulk reads. Dedicated canonical import uses explicit session/attempt and revision, not latest-stage-session mapping. PostgreSQL proves concurrent admission, foreign-execution refusal, lease recovery, stale-token refusal and all-or-none publication. Existing execution/coverage/handoff rows are unchanged after import/failure. Store restoration calls no provider.

### Adapters and asynchronous processing

- [ ] **T6. Enable eligible recording and confirm provider finalization** — Infrastructure / application — M
  - depends: T1–T4 — ref: design §4–§5.
  - DoD: new eligible sessions send `store: true` and persist provenance; legacy null is not promoted. Implement single-owner sideband close observation without stealing frames from T21 controls. Provider close/finalization is background work after completion; bounded cancellation and a lost observation have tested recovery. No change to the interview's terminal outcome or candidate response schema.

- [ ] **T7. Implement bounded recording download, transcription and worker orchestration** — Application / infrastructure — L
  - depends: T1, T3, T5, T6 — ref: design §5–§7.
  - DoD: backend streams verified stereo audio, preserves candidate/model channels, enforces approved byte/duration/temp limits and <=25 MB chunks, and calls the verified transcription schema. Mockery worker tests and fake HTTP fixtures prove pending retry, permanent failure, reconnect manifest completeness, chunk resume/dedup, ordering, token checks, cancellation and safe logging. Persist counts/deadlines and publish no partial initial evidence. Run focused tests, `make test` and `make test.security`; record new dependency versions only if needed.

- [ ] **T8. Dispatch transcription jobs with dedicated durable outbox and broker** — Infrastructure — M
  - depends: T2, T3, T5 — ref: design §3, §5, §7, §9 Q3.
  - blocked by: new topology owner decision; ready when Q3 specifies declarations and malformed-message disposition.
  - DoD: persistent identity-only messages, confirms, manual ACK after durable handling and reconciliation. No report outbox constraint/claim or consumer changes. Mockery unit tests pass; real RabbitMQ/PostgreSQL confirm, outage, duplicate, crash-before-ACK and DB-error cases pass independently of sqlmock. Malformed messages cannot loop or enter report quarantine accidentally.

- [ ] **T9. Join completion admission and compose recoverable background workers** — Runtime / application — L
  - depends: T2, T5–T8 — ref: design §4–§5, §9 Q4.
  - blocked by: unresolved Q1–Q4 or notice/storage readiness. Ready when their decisions and prerequisite proofs are recorded.
  - DoD: same completion transaction inserts job, frozen session manifest and outbox; rollback is atomic and replay has one job. Reconciler repairs eligible missed admission after restart using the explicit cutoff. Candidate completes while fake provider blocks. Do not weaken active terminal fences; remove synchronous post-completion recording finalization from the request path. Shutdown finishes inside 5 seconds, cancels I/O and preserves durable intent. No report runtime registration.

### Handoff and acceptance

- [ ] **T10. Synchronize the final-evidence contract with Reporting** — Application contract / documentation — M
  - depends: T3, T5 — ref: design §2, §9 reporting handoff.
  - DoD: document immutable execution/revision/entry/body read shape, no partial revision visibility, optional question attribution and supersession rules. Synchronize IRP design Q3/T14, writer body-input requirement for T13 and actual assignment completion-marker integration for T15. Preserve IRP task IDs/Done status and its two-attempt/30-minute wait behavior. Production report adapters, provider model decision Q2 and report route registration remain IRP tasks; this task does not mark them Done.

- [ ] **T11. Prove the complete asynchronous flow and publish evidence** — Verification — L
  - depends: T7–T10 — ref: design §8; strategy §5–§6.
  - DoD: provide dedicated isolated verification tooling where needed; prove COMPLETED response precedes blocked provider work, restart recovery, multiple sessions including one reconnect, correct both-speaker evidence, transient/permanent failure, no partial report evidence and resource cleanup. Run `make check`, authorized `make test.integration` or `make test.docker`, real broker tests and a reviewed synthetic OpenAI recording/transcription run. Report exact commands/cwd and passed/failed/skipped results. No mock is evidence for real provider behavior. Update baseline/traceability from results without claiming unreconciled H-09 or reporting Q2 is complete.

## Dependency Graph

T1 and T2 are independent. T1 + T2 → T3 → T4 → T5. T1–T4 → T6. T1 + T3 + T5 + T6 → T7. T2 + T3 + T5 → T8. T2 + T5–T8 → T9. T3 + T5 → T10. T7–T10 → T11.

## Revision History

| Version | Date | Change | Reason |
| --- | --- | --- | --- |
| v1 | 2026-11-03 | Initial T1–T11 plan, all Todo | New asynchronous recording-source extension to Interview Runtime |
