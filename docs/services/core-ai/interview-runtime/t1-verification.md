# T1 verification — CORE-IRT-001

Ticket: **CORE-IRT-001 — Encode interview progress rules** (repository planning ID).
Scope: T1 only. The remaining task checklist entries are unchanged.

## Design mapping and assessment

Design §3–§4 and decisions D1/D5/D6/D7/D9 are retained. `domain/interview.Progress` owns coverage, final citations, omissions, technical losses, stage authorization, structured handoffs, follow-up limits and reconnect counters. It uses the existing stage enum and published HR configuration. It does not own execution status or perform I/O.

`NewProgress` receives the pinned `PublishedRevision[HRConfiguration]`, copies its questions and seconds, and freezes cap 2. `RestoreProgress` checks structural consistency, retains the supplied positive cap, and never invokes publication assessment. A historical duration fixture outside the current 300–600 total restores successfully.

The public boundary refines the design into pure `Authorize`, `CommitHandoff`, and `ActivateStage`. A handoff seals its source. A target cannot activate without that sealed source. `CommitHandoff` changes memory; the repository/coordinator in later tasks must establish durable commit before opening a session. An empty target denotes the closing handoff.

Assessment disposition: **retained with rationale**. Private aggregate state and copied snapshots protect ownership; coverage operations own transitions and citation validation; restore validation is separate from current publication eligibility. The concern-specific files keep one aggregate boundary without introducing a policy service or an unused interface. No additional refactor was justified after GREEN.

## Public behavioral evidence

| Tests in `domain/interview` | Evidence |
|---|---|
| `TestProgressCreation` | Pinned questions/seconds, cap 2 without a revision cap field, UTC milliseconds, invalid construction. |
| `TestCoverageEdges` | Legal edges, terminal immutability, rejection without mutation. |
| `TestCitations`, `TestOmissionReasons` | Final candidate citations, fragment rejection, stage isolation, cap 2, three omission reasons and one finalized clarification. |
| `TestTechnicalLoss` | Pending/asked preserved, no technical omission, authorization blocked, later finalized answer resolves the gap. |
| `TestStageHandoffs`, `TestDeadlineBoundaries` | Fixed stage order, source handoff required, closing without a target, completion readiness, published soft budgets, 15-minute equality. |
| `TestReconnectBudgets` | Backend-detected 120-second window, equality excluded, two attempts per episode, two episodes, same execution identity. |
| `TestRestoreProgress`, `TestRestoreHandoffCorruption`, `TestSnapshotOwnership`, `TestRejectedOperations` | Stored policy, technical corruption errors, sealed records, detached slices/time pointers, failure atomicity. |

## Local checkpoints

Cwd for all commands below: `/Users/rfanazhari/Projects/go-workspace/prototype`.

RED and GREEN scenario command:

```sh
go test ./domain/interview/... -v -run 'Test(Progress|Coverage|Citations|Omission|Technical|Stage|Deadline|Reconnect|Restore|Snapshot|Rejected)'
```

RED: **failed as intended**. A valid pinned revision could not create progress because `NewProgress` was an inert stub returning `ErrInvalid`. No syntax, missing-symbol, credential or resource failure was counted as RED. Scaffolding consisted of the referenced types/constants/errors and inert public operation signatures.

RED SHA: `64481dea88b9b1a84e4213033f34286106249a37`.

RED subject: `test: [CORE-IRT-001][rfanazhari] RED cover interview progress rules`.

GREEN: the same scenario command **passed**. Two fixture corrections made the restore cases accurately represent their intent: clear the existing answer citation, and clone the historical configuration before comparing detached state.

## Final checks and limits

| Command | Result |
|---|---|
| `gofmt -w domain/interview` | Passed; changed Go files formatted. |
| Scenario command above | Passed. |
| `make test` | Passed with race detector; `domain/interview` statement coverage 90.6%. |
| `make test.architecture` | Passed. |
| `make lint` | Passed, 0 issues. |
| `make test.security` | Passed, no affected vulnerabilities; one module advisory has no detected called vulnerable symbol. |
| `make test.all` | Passed for enabled tests, including contract/architecture and existing package regression tests. |
| `go test ./test/integration/... ./test/smoke/... ./test/livevoice/... -v` | Confirmed real-resource cases skipped under their opt-in guards. |

Initial sandbox runs of `make test`, `make lint`, and `make test.security` failed to access parts of the existing Go cache. All three passed after rerunning with the necessary sandbox access.

Database proof is **skipped**: `TEST_DATABASE_URL` and disposable migration database configuration are absent. Live provider proof is **skipped** under the `LIVE_PROBE` guard. No SQL, HTTP or provider behavior is claimed by these domain tests. No SQL mock or interface mock is needed in T1; sqlmock and Mockery remain the requested tools for subsequent tasks that need those boundaries.

Sessions, checkpoints, optional narrative summaries, classified-event ingestion/corrections, cross-aggregate fencing, persistence, and lifecycle calls remain with their planned tasks. T1 does not resolve Q1/Q2/Q3/Q5 or certify the full live runtime. Supplied design references describing older lifecycle/schema implementation still need a fresh assessment when those dependent tasks begin.
