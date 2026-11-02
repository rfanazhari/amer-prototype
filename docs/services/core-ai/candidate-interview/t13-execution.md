# T13 execution — CORE-CIN-013

Ticket: **CORE-CIN-013 — Prove the candidate interview slice**. Only T13 is completed here. Sibling capability statuses (Execution Lifecycle, Interview Runtime, Reporting) are unchanged.

## Mapping and design assessment

T13 is verification of behavior owned by T5–T12; it changes no production code, schema, contract or port. The proof composes `runtime.NewCandidateInterview` (real HTTP handler, session authority, consent store, controller store, lifecycle adapter, execution store) over the release schema (version 9) on a disposable PostgreSQL 16 container (`core-cin-013-proof`, no development volume).

- `TestCandidateInterviewSlicePostgres` (`test/integration/candidate_slice_test.go`) drives the capability examples through HTTP as two named tabs.
- `TestCandidatePageDemoPostgres` (`test/integration/candidate_page_demo_test.go`) serves the real `page.html` over the same composition and holds the server open only when `CIN_DEMO_HOLD_SECONDS` is set, so a browser can be driven by hand. Otherwise it skips. `GET /candidate/session` is a stand-in returning the fixture CSRF token and the session cookie is attached by the server wrapper (the access exchange is outside this slice).
- Assessment: retained. No production refactor is justified by the proof.

## Capability examples

| Example | Evidence |
| --- | --- |
| Notice before start | Slice test: before consent only `accept` is offered, start is refused, `interview_executions` stays empty. Browser: the page showed the approved notice and only "Saya setuju" before acceptance, then "Start interview". |
| One execution after a lost start | Slice test: same `Idempotency-Key` returns 201 then 200 with the same id; one `interview_executions` row. Browser: one row after start and reload. |
| Coverage at the 15-minute boundary | Slice test against the wall clock: session expiring in 14m59s offers no action and refuses start with `CIN_SESSION_COVERAGE`; 15m30s offers `start` and starts. Exact equality is not reproducible with the system clock; it stays covered by the controlled-time domain and application tests in `make test`. |
| Second-tab takeover | Slice test: second tab sees `takeover`, a non-controller cancel and a stale takeover are refused, takeover succeeds, the first tab is read-only and its cancel is refused. Browser: second tab read-only, took control, first tab turned read-only after its poll. |
| Early cancel shown as cancelled | Slice test: controller cancel returns `CANCELLED`, reason `candidate_early_stop`, no actions. Browser: page shows "Interview CANCELLED. You have left the interview early." and `agent_executions` holds `CANCELLED / candidate_early_stop`. |
| Refresh without a new execution | Slice test: three GETs keep one execution. Browser reload kept the same row (count 1). |
| Completed interview while no report exists | Slice test: after lifecycle completion the view is `COMPLETED`, has no actions and no report field, and no table matching `%report%` exists. Rendering of this state in the page is covered by the T10 headless-Chrome scenarios; it was not repeated in the live browser run. |

## Limits (not claimed)

- **Progress and advance fencing are not demonstrated live.** Interview Runtime T8 composition is deferred (no `PinReader` adapter, session-attempt routes unmounted), so `NewCandidateInterview` passes no `Progress` and the page shows none. The fence is covered by the T12 mocks only.
- The browser run used the built-in browser against an in-process server with a stand-in session endpoint and an injected cookie, not a deployed binary or the invitation exchange.
- The completed state was proved at the HTTP level only in this run.
- Exact `expiresAt == now+15m` is not tested with real time.
- Unfiltered `make test.integration` fails: 11 tests, 37 of the errors `relation "candidates" does not exist`. `TestCandidateAccessHTTPProvesAssignmentIsolationAndRecovery` is the first to run and drops `candidates` with CASCADE; later fixtures fail. This is the pre-existing isolation problem recorded for T4, T5, T8 and T11, not introduced here. The four candidate suites pass together on a fresh database.
- `make test.docker` was not run.

## RED

None observed. The slice test passed on its first run because T5–T12 already implement the behavior; no failure was manufactured. The test files are committed as a characterization checkpoint.

## GREEN and verification

Cwd for every command: `/Users/rfanazhari/Projects/go-workspace/prototype`. `TEST_DATABASE_URL` points to the disposable container and is supplied only through the environment.

| Command | Result |
| --- | --- |
| `go test -race -p 1 ./test/integration/ -run TestCandidateInterviewSlicePostgres -count=1 -v` | Passed (3 subtests). |
| `GOFLAGS='-run=^(TestCandidateConsentRepositoryPostgres\|TestCandidateControllerRepositoryPostgres\|TestCandidateLifecyclePostgres\|TestCandidateInterviewSlicePostgres)$' make test.integration` (fresh database) | Passed; smoke selected no tests. |
| `make test.integration` (unfiltered) | Failed, 11 tests, pre-existing fixture isolation (above). Smoke package passed. |
| `make test` | Passed. |
| `make test.contract` | Passed. |
| `make test.architecture` | Passed. |
| `make lint` | Passed. |
| `make test.security` | Passed. |
| `make test.docker` | Not run. |
