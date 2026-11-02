# T12 execution — CORE-CIN-012

Ticket: **CORE-CIN-012 — Integrate candidate-safe progress and controller fencing**. Only T12 is completed here; T13 and sibling capability statuses are unchanged.

## Mapping and design assessment

- Candidate-safe projection: `infrastructure/service/candidateprogress.Reader` implements `service.InterviewProgress` over the runtime `interviewport.ProgressStore.Find`. It copies only a neutral phase label (Background, Screening, Expectations, Closing) and a connection token. Transcript, coverage, losses, handoffs and controller data are never read into the result. `GetInterview` already renders only that object (T5), so the query is unchanged.
- Connection rule: an active reconnect episode is `reconnecting`; otherwise an open session with provider identity and browser readiness in the closing stage is `finalizing`, in any other stage `connected`. Absent progress or a session that is not yet ready is no summary (`progress` omitted), so the page never shows a state runtime has not established.
- Controller fence: the runtime already refused a previous tab id in `authorizeCandidate` against `interview_controllers`. `Ready` now re-reads the controller inside the lifecycle `WithProgress` lock and refuses a tab replaced since the first check. No schema change: the fence stays in `interview_controllers` (T8).
- Assessment: retained. No domain, port, schema, OpenAPI or JSON change; existing mocks reused.

## Limits (not claimed)

- The re-check is not atomic with a takeover: a takeover committing after the in-lock read is not excluded. Closing that needs a shared row lock on the controller row in the lifecycle transaction (design change to the controller port).
- `Attempts` (activate/reconnect) keeps only the pre-check; its lock lives inside `Workflow`. Not changed here.
- **Not composed in production.** `interviewrepository.Store` needs an `interviewport.PinReader`, which has no adapter yet, and the session-attempt routes are not mounted (Interview Runtime T8 is still `[~]`). `NewCandidateInterview` therefore still passes no `Progress`, and the page shows no progress until runtime composition lands. T13 must not claim a live progress demonstration until then.
- Mocks only; no PostgreSQL proof for the new code. `TEST_DATABASE_URL` was not set; `make test.integration` and `make test.docker` were not run.

## RED

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./infrastructure/service/candidateprogress/ ./application/usecase/interviewusecase/ -run 'TestReaderCurrent|TestReadyRefuses' -count=1
```

Reason: the inert reader returned no summary for every record, and `Ready` wrote readiness although the controller had been replaced (unexpected `Update` call). SHA `68f2474faba522271e69f8432c490a49c417e2a9`, subject `test: [CORE-CIN-012][rfanazhari] RED cover candidate-safe progress and controller fence inside the lifecycle lock`.

## GREEN

Same command passes; then `make test` and `make check` (all-package race, OpenAPI, architecture/documentation, lint 0 issues, govulncheck 0 reachable) pass. SHA `8e58c7b96c06acd5b6003b79c4f62f0336058eee`, subject `feat: [CORE-CIN-012][rfanazhari] GREEN project candidate-safe progress and recheck controller inside the lock`.
