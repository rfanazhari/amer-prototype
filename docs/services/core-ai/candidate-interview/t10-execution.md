# T10 execution — CORE-CIN-010

Implemented only T10: candidate consent, API-offered start, read-only/takeover,
confirmed cancellation, safe progress, and distinct terminal screens. Controller
identity and the consent-specific start intent remain in sessionStorage. Refresh
only reads. Live views poll every two seconds; terminal/access loss stops polling.
Superseded reads cannot overwrite a newer mutation result. Notice/progress use
textContent. The page never requests reporting or creates media/stage controls.

## Design mapping and assessment

Design §5–6 and decisions D1–D3 remain authoritative. Delivery owns rendering,
request headers/bodies, tab identity, confirmation, polling and stale-response
suppression. Domain/application retain consent, coverage, controller and lifecycle
invariants; the UI uses the API action list. No SQL, interface, migration, runtime
adapter, or business-policy change. Existing sqlmock and Mockery tests remain;
no new SQL/interface mocks or regeneration were needed.

Retained with rationale: one embedded page with small render, read, mutation,
storage and poll functions is sufficient for this UI. No domain refactoring or
framework is justified. The generation counter discards obsolete responses, a
single timeout bounds polling, and pagehide clears it. Failed session fetches can
recover through refresh; unsafe requests remain explicit. No design rule changed.

## Local checkpoints

All commands ran from `/Users/rfanazhari/Projects/go-workspace/prototype`.

- RED: `python3 scripts/test_candidate_interview_ui.py` exited 1 with
  `FAIL browser scenarios: FAIL start offered by API`. The real page lacked the
  start control despite the fixture API offering it. No production scaffolding.
- RED SHA: `1d9f961e6bbf3cbc5dde2389875d556804208e74`.
- Subject: `test: [CORE-CIN-010][rfanazhari] RED cover candidate interview screen behavior in Chrome`.
- GREEN: same browser command exited 0; final coverage is 17 browser scenarios.
- Shell regression: `go test ./delivery/http/candidateinterview -run TestInterviewPageShell -count=1` passed.
- `make test` passed: domain/application race tests and coverage.
- `make check` passed: `make test.all`, `make test.contract`,
  `make test.architecture`, `make lint` (0 issues), `make test.security`
  (no called vulnerabilities; govulncheck also reports one uncalled module advisory).
- The first sandboxed Go test attempt was blocked by Go-cache permissions;
  rerunning with approved cache access passed. This tooling failure is not RED.

## Browser evidence and limits

`CHROME_BIN` optionally selects a Chrome executable; the default is the installed
macOS Google Chrome. The Python standard-library harness uses a temporary Chrome
profile and a loopback fixture server, with bounded execution and cleanup. It
executes the actual HTML script in Chrome; fetch, timer and confirmation fixtures
make scenarios deterministic. No Playwright dependency or suite was added.

Scenarios: missing session; notice/explicit acceptance and safe text; start only
when offered; uncertain 503 retry with retained key across refresh; read-only
second-tab view, takeover and declined/accepted cancel confirmation; each of
COMPLETED/FAILED/CANCELLED; live polling to terminal; start 200/201 and refresh
without another POST; obsolete poll after cancellation; session expiry while
polling; controller conflict refresh; safe progress; session network recovery;
lost start response with the same retry intent.

This is browser UI proof against API fixtures, not a full service or database
flow, real multi-tab fencing, or real lifecycle/progress integration. T11/T12
adapters and T13 slice proof remain pending. Database suites were skipped because
TEST_DATABASE_URL was absent. `make test.integration`, `make test.docker`, and
`make docker.verify` were not run; T10 changes no persistence behavior.
