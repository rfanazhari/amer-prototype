# T10 verification — CORE-IRT-010

Handoff commit and next-session open were already built in T3 (`Propose`, `OpenStage`). T10 closes the one gap found:
a failed target create left the activated target stage with an unprovisioned session shell, so every retry returned
`ErrConflict`.

## Design mapping and assessment

- `Propose` commits the handoff and closes the source shell in one locked callback; no target session row exists until
  `OpenStage`. Unchanged.
- `OpenStage` now recognizes a committed, already-activated target (`CurrentStage == target`). It closes the open shell
  that has no provider identity, records it as `ReplacedSessionID`, and appends a new shell. It reads the target from the
  last handoff, so no different stage can be authorized. A provisioned open shell, or none, is `ErrConflict`.
- No lifecycle `Activate` call, no new execution id, no attempt counter (exhaustion stays in T16).
- Optional summary failure is covered by the existing `TestWorkflowPropose`.
- Assessment: design retained; one helper (`releaseUnprovisioned`) in `workflow_helpers.go`.

## RED checkpoint

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./application/usecase/interviewusecase/ -run TestOpenStageRetry
```

Observed: the retry returned `interview progress conflict` instead of opening the target. SHA `88c22c5a1efd796af3e5c4745264f733a917f6bc`.

## GREEN checkpoint

GREEN SHA `c2e7bec76e0c14b63ac605ecf91decd82faa2622`. `make test`, `make test.all`, `make test.contract`,
`make test.architecture`, `make lint` (0 issues) and `make test.security` (0 reachable vulnerabilities) passed.

## Limits

- `../../../../../test/integration/interview_handoff_workflow_test.go` (handoff + source close, target row absent, fence rejection,
  target-only retry on PostgreSQL) compiles but **skipped**: `TEST_DATABASE_URL` was not set. It is not database proof.
- Its lifecycle is a pass-through double, so it does not prove a cross-aggregate fence. The real locked callback waits for
  Q3 and the T8 lifecycle adapter.
- sqlmock was not added: T10 changes no SQL; the existing sqlmock store tests already cover the handoff write.
