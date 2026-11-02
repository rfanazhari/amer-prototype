# T8 verification — CORE-IRT-008

Task marked Done by the owner for the operations, handlers and contract only. Done: the session-attempt and readiness operations, their HTTP handlers, public error codes,
translations, OpenAPI, API changelog and endpoint note. **Deferred to a follow-up task (ID to be assigned): composition and mounting.** The routes are not
reachable in the running process; T7's removal ships with that follow-up.

## Design mapping and assessment

- `interviewusecase.Attempts` authorizes (candidate session, execution inside the session's assignment, current
  controller), replays a stored command before checking the version, then calls `SessionStarter` (`Workflow.Activate` /
  `Workflow.Reconnect`). Another assignment's execution is `service.ErrInterviewNotFound` (404).
- `interviewusecase.Ready` fences the attempt, then sets `ReadinessAt` on the attempt's open session inside
  `Lifecycle.WithProgress`. It touches no coverage, stage or transcript; a test compares the snapshot before and after.
- Handlers live in `candidateinterviewhandler` (`attempt.go`) to reuse the candidate cookie, origin and CSRF checks.
  `ready` takes `controllerId` and `expectedVersion` in the body; design section 2 did not specify a `ready` body.
- New ports: `interviewport.SessionCommands` and `interviewport.RevisionReader` (no adapters yet).
- Error mapping: `ErrAttemptConflict` → `409 IRT_CONFLICT`, `ErrAttemptNotReady` → `409 IRT_NOT_READY`,
  `ErrProviderUnavailable` → `503 SESSION_UNAVAILABLE`. Nothing yet wraps driver failures in `ErrProviderUnavailable`.

Assessment: retained. Authorization is shared in one helper; readiness semantics (provider id and flag) live in one
function used by both operations. Findings deferred, each needing a decision or scope:

1. **Lifecycle adapter.** `service.InterviewLifecycle` speaks `AttemptID`; `executionusecase` speaks `OperationKey` and an
   integer `Generation`, with no 2 x 30 s / 90 s activation budget. The mapping is a design decision (design section 1 and Q3).
2. **Command persistence.** `interview_session_commands` has no `session_id`, `answer_sdp` or `execution_version`, so
   a replay cannot return the original answer from storage. Needs migration `000011` plus a GORM store.
3. **Atomicity.** The command is stored after the provider answer exists. A crash between the two leaves a session without
   a command. Design section 4 wants the command with the session shell in one transaction.
4. **Revision reader and wiring.** `RevisionReader`, `ProgressStore` (with `PinReader`), the driver and its key, and
   `infrastructure/runtime/voice.go` (`isCandidateExecutionPath`) are not composed.

## Test paths

New internal contract and new behavior: RED then GREEN. One existing test changed in GREEN:
`TestOpenAPIKeepsUnmountedInterviewRoutesAbsent` asserted that `session-attempts` and `/ready` are absent; it now asserts
only that `/report` is absent, and `TestOpenAPIPublishesSessionAttemptRoutes` asserts the new routes.
`../../../../../infrastructure/runtime/access_route_test.go` still expects both routes to fall through to access, which is correct until they are mounted.

## RED checkpoint

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./application/usecase/interviewusecase/ -run 'TestAttemptsExecute|TestReadyExecute'
go test ./delivery/http/handler/candidateinterviewhandler/ -run TestSessionAttemptTransport
go test ./translations/ -run TestInterviewRuntimeMessages
```

Observed: operations returned a stub error, routes answered 404 instead of the contract's statuses, and the catalogs had
no `IRT_*` text. RED SHA `c936979904f0806acd65b38f7d002ac5e642b2ca`.

## GREEN checkpoint

GREEN SHA `97d9107839153d8528b1f285dbb8f80c063e1b9c`. `make test`, `make test.all`, `make test.contract`,
`make test.architecture`, `make lint` (0 issues) and `make test.security` (0 reachable vulnerabilities) passed.

## Limits

- Mocks only. No SQL, PostgreSQL, provider or HTTP-process behavior is established; `TEST_DATABASE_URL` was not set, so
  database suites were skipped by `make test.all`.
- No generated SQL, so sqlmock does not apply.
