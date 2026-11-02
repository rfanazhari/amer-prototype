# T11 verification — CORE-IRT-011

`Workflow.Finish` in `../../../../../application/usecase/interviewusecase/finish.go` chooses one terminal lifecycle outcome.

## Design mapping and assessment

- Invariant ownership: `interview.Progress.ReadyForCompletion` and `HardDeadlineReached` (domain, unchanged). The use case only reads them.
- Ready record: one `Lifecycle.Complete` with the execution id as the opaque progress reference. No store write, no report row, no outbox row.
- Record missing and `!now.Before(runningAt+15m)`: one `Lifecycle.Fail`. No store write, so earlier handoffs remain.
- Record missing before the deadline: `ErrConflict`, no terminal call.
- A lifecycle rejection is returned as-is. Nothing was written, so nothing is deleted.
- Stale or incomplete authority stops at `findCurrent` (validation, then `Fence`).
- Assessment: design retained. No new port, no sqlmock (no SQL change), no new Mockery interface.

## RED checkpoint

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./application/usecase/interviewusecase/ -run TestWorkflowFinish
```

Observed: the inert `Finish` made no `Fence`, `Find`, `Complete` or `Fail` call, so the mock expectations failed. SHA `70ca4dfcc69a904741a71ae147251854657f4af2`.

## GREEN checkpoint

GREEN SHA `f14f206f52dccd392bf1f78999205f2953460245`. `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint` (0 issues) and `make test.security` passed.

## Limits

- Mock lifecycle only: no real fence and no race between complete and fail. The first terminal commit is the lifecycle's (Q3, T8 adapter).
- Nothing calls `Finish` yet; the deadline timer and explicit-close trigger belong to the composition work in T8.
- `H-10` report-failure behavior is not claimed.
- The "closing handoff rollback before the lifecycle call" case is covered by the existing `Propose` rollback tests, not repeated here. Database suites skipped without `TEST_DATABASE_URL`.
