# T17 verification — CORE-IRT-017

## Status

Accepted 2026-11-03 (Asia/Jakarta). One live interview-path voice session was recorded
through the mounted route (see the live runs below) and this sanitized note was
reviewed and approved by the owner, `rfanazhari`. T17 is checked Done. Q1 remains
unresolved. Earlier sections record the pre-acceptance attempts and harness defects.
This is a verification-only task; no Go implementation, interface, SQL, mock,
dependency or public contract was changed. No RED/GREEN checkpoint applies and
no phase commit was created.

The owner selected local execution. A temporary harness is prepared at
`/private/tmp/core-irt-017/`: `run.py`, `live_test.go`, `page.html` and a generated
Go overlay mapping. It reuses existing synthetic candidate/publication fixtures,
real PostgreSQL and production interview routing; it changes no repository Go
files and introduces no mocks. The wrapper creates only its own ephemeral
PostgreSQL container and removes that container afterward. The initial browser
setup does not capture microphone audio.

The attempted command `python3 /private/tmp/core-irt-017/run.py` was rejected
before execution by automatic approval review: sending the API credential,
WebRTC SDP and synthetic interview context to the external provider requires
specific user authorization beyond selecting local execution. No container or
provider session was created by this attempt. Approval is pending for the live
HTTPS session endpoint and WSS sideband destination. Python syntax validation
passed and the temporary Go source was formatted; the overlay has not yet been
compiled or executed. Successful setup alone will still not prove live audio
interaction or complete the required sanitized-note review.

## Authorized attempt — 2026-11-03

The owner authorized the provider egress. `python3 /private/tmp/core-irt-017/run.py`
(cwd: repository root) started its disposable PostgreSQL, then
`TestT17LocalLive` failed in fixture setup (`candidate_page_demo_test.go:51`):
deleting from `interview_executions` violated
`interview_controllers_execution_id_fkey` (SQLSTATE 23503). The harness's
cleanup order is wrong; this is a harness defect, not product behavior. It
failed before any provider request, so no session, SDP exchange or sideband was
recorded. The container was removed. Live audio also needs a person speaking in
the browser, so T17 stays unchecked.

### Retry after harness fixes

Two harness defects were fixed in `/private/tmp/core-irt-017/live_test.go`
(cleanup deleted `interview_controllers` last; `GET /` conflicted with the
`/candidate/` mux pattern). The harness then served the page and the browser
generated an SDP offer and called the mounted
`POST /candidate/executions/{id}/session-attempts` route. The route returned
HTTP 409 with safe code `IRT_CONFLICT`, so no answer SDP, retained session id
or sideband attach was observed. The cause of the conflict (fixture
state/version or controller context) is undiagnosed. The harness wrapper
finished and its container was removed. T17 stays unchecked; no live
acceptance, audio interaction or review is claimed.

### Diagnosis of the 409

With the real execution version (1) the route still returned `IRT_CONFLICT`.
Sanitized harness diagnostics showed one controller row and one session command
in state `failed`: authorization, version and reservation passed, and setup
failed after the durable reservation. `setupError` maps every such failure
(provider create or sideband attach) to `IRT_CONFLICT`, hiding the cause. A
status-only probe of the default create endpoint with the configured credential
and a driver-shaped payload reached the provider (HTTP 400 `invalid_offer` for a
stub SDP), so the credential, model and endpoint are accepted. The real cause of
the failed command (provider response to the browser offer, or sideband attach)
is not yet identified; the cause needs driver-level diagnostics. No answer SDP,
retained session id or sideband was recorded. T17 stays unchecked.

### Root cause and observed live session (2026-11-03)

Harness-only provider-error logging (overlay copies of the driver; no repository
file changed) showed the create call failed before any request:
`transaction from the request context cannot be empty`. The harness mounted the
interview `Router` without `appruntime.HTTPServer`, so no logmanager transaction
reached `clientmanager`; the production composition wraps it. This was a harness
defect, not a product defect. After wrapping the router, one live run observed:

- Route: `POST /candidate/executions/{id}/session-attempts` returned 201.
- Browser applied the returned SDP answer; connection reached `connected`, the
  `oai-events` data channel opened and a remote audio track was present.
- Retained provider session id present in the committed session (presence only);
  the persisted answer and command state `succeeded` were present.
- Sideband attach succeeded: attach runs inside setup before commit
  (`interviewusecase/session.go:37`), and no attach failure was logged.
- Readiness route returned 200.
- Cleanup: local WebRTC closed, harness finished with `PASS`, disposable
  PostgreSQL container removed.

Not observed: spoken candidate/provider audio exchange (needs a person speaking;
the harness captures no microphone) and the sanitized-note review. T17 stays
unchecked until both are recorded. Q1 is not resolved by this record.

### Live audio run with microphone (2026-11-03)

The harness page was extended to capture the microphone and served on
`http://localhost:8090/` (the owner's Docker stack held 8080). The owner opened
it in their own Chrome, granted microphone access, spoke and confirmed hearing a
spoken reply from the provider over WebRTC. The harness then closed local media
and the connection, finished with `PASS`, and removed its disposable PostgreSQL
container; port 8090 was released. The route, SDP answer, retained session id
presence, sideband attach and readiness outcomes are those recorded above. No
key, SDP, transcript or session id is stored in this note.

Remaining before T17 may be checked Done: reviewer and outcome for this
sanitized note. Q1 is not resolved by this record.

## Observed readiness

- T18 records production route mounting and disposable PostgreSQL/fake-provider
  proof. That existing evidence does not establish a live provider pass.
- `OPENAI_API_KEY` is nonempty in the local `.env`; its validity and provider
  entitlement have not been tested. It was not printed or copied into this note.
- `DATABASE_URL` and `TEST_DATABASE_URL` are absent from the execution environment;
  neither is defined in `.env`. No existing database was accessed or mutated.
- No running containers were listed by `docker ps --format '{{.Names}}\t{{.Ports}}'`.
  The initial sandbox attempt could not access the Docker socket; the authorized
  retry succeeded and returned an empty list.
- Loopback probes found no listener on ports 8080, 8081 or 5432 within the sandbox.
  No reachable application/candidate fixture was supplied.
- `delivery/http/candidateinterview/page.html` contains notice, consent, start,
  takeover and cancel controls, but no WebRTC negotiation or microphone access.
  Candidate-page voice integration is outside T17's verification scope.
- `test/livevoice/page.html` negotiates through the separate probe `/setup` route.
  Running that probe alone would not satisfy the interview-path requirement.

## Required live record — not yet observed

- [ ] Use the mounted `POST /candidate/executions/{id}/session-attempts` route
  with an authorized synthetic candidate, cookie, CSRF and controller context.
- [ ] Browser supplies a WebRTC SDP offer and applies the returned SDP answer.
- [ ] Confirm retained provider session id is present in the committed session;
  record presence only, never its value.
- [ ] Confirm backend sideband attach succeeds for that same session.
- [ ] Confirm candidate/provider live audio interaction over WebRTC.
- [ ] Close local media and owned connections; record cleanup outcome.
- [ ] Review the completed sanitized note and record reviewer/outcome before
  checking T17 Done.

Do not store the API key, cookies, SDP, transcript, instruction text or session id
in this record. Q1 remains unresolved; event counts or opaque event names do not
establish finality. No live session was created during this preflight.

## Local checks

Cwd for the commands below:
`/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
GOCACHE=/private/tmp/t17-go-cache go test ./infrastructure/service/interviewvoice ./infrastructure/runtime -count=1
GOCACHE=/private/tmp/t17-go-cache GOLANGCI_LINT_CACHE=/private/tmp/t17-lint-cache make lint
GOCACHE=/private/tmp/t17-go-cache make test.security
```

The first transport-test run failed because the sandbox denied the fake HTTP
server's loopback listener (`bind: operation not permitted`); runtime tests
passed. This is an environment failure, not behavioral RED. The same command
was retried with approved sandbox escalation and passed for both packages.

Lint passed with `0 issues`. The initial security run could not resolve
`vuln.go.dev`; it was retried with approved network access and passed: zero
reachable vulnerabilities, with one finding in a required module that is not
called by this code.

Real database integration and live acceptance were not run. Mock transport
results cannot establish SQL persistence or live provider behavior.

## Resume requirements

Authorize the pending provider payload/destination for the prepared local
disposable environment and synthetic candidate harness. Execute one live session,
fill the observed record above without payloads, and obtain review. Then update
T17's checklist. Do not modify T12–T16 or mark Q1 resolved from this task.
