# T9 execution — CORE-CIN-009

Ticket: **CORE-CIN-009 — Mount candidate execution start, poll, takeover and cancel**.
Only T9 is completed by this execution. T10–T13 remain pending.

## Scope and design assessment

Mapping: technical design §2 and §6–7. Delivery owns route dispatch, strict transport validation, cookie/CSRF checks, the candidate-safe transport view and localized status/error mapping. The existing application operations and domain keep consent, coverage, assignment isolation, controller/version policy and lifecycle ownership. Runtime composes these operations with the T8 ControllerStore, existing consent/session adapters and unbound lifecycle. No interface or schema changed; existing Mockery-generated mocks were reused without regeneration.

Assessment retained the operation interfaces and domain/application ownership: HTTP introduces no business rules or SQL. One execution mapper serves the interview read and all execution responses. Exact route matching keeps session-attempt, ready and report paths outside interview dispatch. No broader refactor was justified. T11 must return the documented lifecycle errors, including `ErrInterviewStartUncertain` for ambiguous commit and `ErrInterviewNotLive` for a not-live refusal; T9 maps them but does not implement that lifecycle behavior.

A review finding was fixed within the initial GREEN phase: repository restoration wraps domain validation errors. Those failures must remain technical 500 responses rather than become 400 input errors. Tests observed both corrupt-controller and corrupt-consent cases failing with actual 400 before the mapper correction, then passing with 500. Direct input validation errors retain 400. Duplicate JSON fields are now rejected by the interview JSON decoder, including consent input; existing consent regressions pass.

## RED checkpoint

Cwd for every command below:
`/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./delivery/http/handler/candidateinterviewhandler ./infrastructure/runtime ./test/contract -run 'TestExecutionTransport|TestCandidateInterviewRouteDispatch|TestCandidateExecutionCompositionFailsClosed|TestOpenAPIPublishesCandidateExecutionRoutes' -count=1
```

Observed RED: execution requests returned 404 instead of the specified statuses, the router sent execution paths to access rather than interview, and the four OpenAPI path entries were absent. The RED snapshot contains tests, four Handler dependency fields required to wire operation mocks, and two error sentinels required by mapper tests; no new route behavior.

SHA: `f6994d4dbc9212c1e9197d672910422b671e5e77`.

Subject: `test: [CORE-CIN-009][rfanazhari] RED cover candidate execution routes and fail-closed composition`.

The initial sandboxed command encountered a build-cache permission failure; that is not the RED evidence. The same complete command was rerun with authorized cache access and produced the behavioral failures above.

## GREEN verification

The same RED command passes on the final implementation. Further checks:

| Command | Result |
| --- | --- |
| `make test` | passed, domain/application race and coverage |
| `make test.all` | passed, all-package race checks; resource-dependent suites skipped without their opt-in configuration |
| `make test.contract` | passed, combined OpenAPI and T9 route contract |
| `make test.architecture` | passed, dependencies and declaration comments |
| `make lint` | passed, 0 issues |
| `make test.security` | passed, no called vulnerable symbols; one finding in a required module is not reachable by this code |
| `git diff --check` | passed |
| `make test.integration`, `make test.docker` | skipped, no real database requested for T9 and no TEST_DATABASE_URL supplied |

Additional review regression command, first observed RED (400 instead of 500), then covered by the passing complete scenario command:

```sh
go test ./delivery/http/handler/candidateinterviewhandler -run 'TestExecutionTransport/corrupt' -count=1
```

The sqlmock consent fixture was completed with its required candidate_session_id before final GREEN. SQL expectations establish the composed application's reads and absence of execution SQL writes on unbound-lifecycle refusal. They do not establish real PostgreSQL concurrency, transaction or lifecycle behavior.

GREEN subject: `feat: [CORE-CIN-009][rfanazhari] GREEN mount candidate execution HTTP operations`.
The GREEN SHA is the commit containing this report; resolve it with:

```sh
git log -1 --format='%H %s' -- docs/services/core-ai/candidate-interview/t9-execution.md
```

API documentation, OpenAPI 0.10.0, changelog, design/strategy evidence and T9 checklist are synchronized. No new business decision changes decision-log D1–D3. The page shell is unchanged; full UI controls are T10. Live execution persistence/cancellation, progress and controller fencing remain T11/T12, and the full slice proof is T13. Handler mocks and sqlmock composition tests are the runtime evidence here; no live-service or real-database claim is made.
