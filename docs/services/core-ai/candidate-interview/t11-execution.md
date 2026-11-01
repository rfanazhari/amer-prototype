# T11 execution — CORE-CIN-011

Ticket: **CORE-CIN-011 — Integrate candidate lifecycle commit-or-find, read and cancel**.
Only T11 is completed here. T12/T13 and sibling capability statuses are unchanged.

## Mapping and design assessment

Technical design §6.3 owns the integration contract. Consent/coverage/controller policy remains in the candidate domain/application. Execution Lifecycle owns the execution aggregate, assignment-before-execution locking, transition history and transactional commits. `candidatelifecycle.Lifecycle` adapts that existing owner; production composition replaces the unbound adapter. No schema, interface signature, dependency or JSON shape changed. Existing Mockery-generated port mocks are reused; no regeneration is needed.

Refinements required by actual sibling behavior:

- Lifecycle accepts multiple command keys per assignment. Candidate start instead uses the fixed command identity `candidate-start` with two typed SHA-256 input references for the client key and consent id. Its fingerprint therefore rejects either changed field. Hashing preserves the candidate API's 128-character client key while satisfying lifecycle's 64-character reference limit; typed prefixes prevent swapping the unordered references.
- A call-local id minter distinguishes a committed first insert from a replay or reconciled race. The adapter reports success only after lifecycle returns from commit. It introduces no shared mutable request state or pre-read inference for 201/200.
- Lifecycle allows cancellation replay. The candidate contract refuses every terminal cancel, so `nonterminalStore.Update` checks the locked snapshot before invoking lifecycle's callback. The lifecycle aggregate still owns the transition. This closes the concurrent terminal-commit window without changing sibling replay semantics.
- `hardDeadline` is derived from persisted activation plus 15 minutes and copied into the candidate facts. Reconstructing the adapter does not reset it.
- PostgreSQL integration exposed that the existing credential resolver checked session revocation but not assignment revocation. Interview-only `Sessions` joins the access-owned session model to an unrevoked assignment; the existing authority retains session lifetime/revocation validation. Access routes are unchanged.

Assessment disposition: **retained with rationale**. The small adapter isolates the sibling contract differences and keeps all lifecycle writes in the existing transactional operations. No new domain model, operation interface, migration or broad refactor is justified. Candidate views retain their allow-list and no report dependency. The docs now record Q3 as resolved for T11, without claiming T12/T13 or sibling capability completion. Decisions D1–D3 remain unchanged.

## RED

Cwd for every command in this report:
`/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./infrastructure/service/candidatelifecycle -count=1
```

Observed failure: the inert adapter returned `dependency_unavailable` instead of committing/reading/cancelling through lifecycle. The generated port mocks also reported the expected calls missing. RED scaffolding contains only the adapter struct and the three inert method signatures required to run these tests.

SHA: `2acb6bb2f7fccfa4d7b6135f18ef8d2284da5bab`.

Actual subject: `test: [CORE-CIN-011][rfanazhari] RED cover candidate lifecycle integration`.

During GREEN integration, additional failures were observed and fixed: revoked-assignment polling returned 200 instead of 401; ambiguous commit mapped to dependency-unavailable instead of `ErrInterviewStartUncertain`. A classified dependency read was also mapped from 500 to the required 503. Initial architecture/lint checks also found missing comments on local test types and an unused test field; final checks pass. These were not separate phase commits.

## GREEN and verification

| Command | Final result / evidence |
| --- | --- |
| `go test ./infrastructure/service/candidatelifecycle -count=1` | Passed. First commit/replay, 128-character key, intent conflict, deadline ownership, not-found, cancellation, stale/terminal refusal, uncertain/dependency transaction outcomes. |
| `go test ./infrastructure/runtime -run 'TestCandidateExecutionComposition\|TestNewCandidateInterview' -count=1` | Passed. sqlmock asserts joined assignment revocation filtering and dependency failure through real composition. |
| `GOFLAGS='-run=^TestCandidateLifecyclePostgres$' make test.integration` | Passed on disposable PostgreSQL 16, release schema 9. Four subtests described below. Smoke package selected no tests. |
| `make test` | Passed, domain/application race checks and coverage output. |
| `make check` | Passed: all-package race checks, OpenAPI, architecture/documentation, lint (0 issues), security (0 reachable vulnerabilities). Database tests in this run skip without `TEST_DATABASE_URL`. |
| `git diff --check` | Passed. |
| Unfiltered `make test.integration`, `make test.docker` | Not run. No full database-suite or Docker-stack result claimed. |

The integration command receives `TEST_DATABASE_URL` only through the environment, pointing to the task's isolated `core-cin-011-proof` PostgreSQL container with no development volume. The normal development database is unused. Logs contain no database URL or credential.

PostgreSQL proofs exercise production `NewCandidateInterview` handler composition with real authentication, consent, controllers and execution persistence:

1. Eight concurrent starts return one 201, seven 200s, and one execution/pin/start command. A later retry models a lost 201. A changed client key or consent intent conflicts and preserves the original. Candidate cancel commits `candidate_early_stop`; the following HTTP GET returns `CANCELLED` without a report field.
2. Fourteen minutes of remaining session coverage refuses start and leaves no execution pin. Exact equality remains covered by the existing controlled-time domain/application tests in `make test`.
3. A lifecycle activation persists the running timestamp; independently reconstructed adapters return the same activation-plus-15-minute deadline. Lifecycle progress/completion fixtures then produce `COMPLETED`; candidate start replay and poll return that original terminal execution. This fixture does not implement Interview Runtime stages or progress projection.
4. Assignment revocation makes the candidate GET return 401 while the execution is still `CREATED`. An explicit `executionusecase.CancelNonterminal.Execute` call with `access_revoked` then commits `CANCELLED`. The 401 is deliberately not counted as cancellation. An access-owner expiry/revocation driver is outside T11; this task adds no automatic scheduler.

No browser, runtime controller fencing, voice readiness, report route or live external upstream is exercised. The HTTP evidence is in-process production composition against real PostgreSQL, not a separately launched server. Full-database suite isolation problems recorded by earlier tasks are outside this task and were not re-verified here.

GREEN actual subject: `feat: [CORE-CIN-011][rfanazhari] GREEN integrate candidate execution lifecycle`.
Resolve its SHA from the commit containing this report:

```sh
git log -1 --format='%H %s' -- docs/services/core-ai/candidate-interview/t11-execution.md
```
