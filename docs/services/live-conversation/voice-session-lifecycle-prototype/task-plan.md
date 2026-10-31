# Task Plan: Voice session lifecycle prototype

## Legend

- ID: T1, T2, ... (stable — never renumber on revision)
- Ticket: commit-message reference. One commit per task, subject prefixed with the ticket ID: `LIVE-POC-T1: <why the change is needed>`. Further commits for the same task reuse that ticket ID.
- Complexity: S (< 2h) | M (2–4h) | L (> 4h)
- Status: [ ] Todo | [~] In Progress | [x] Done | ~~obsolete~~
- ref: technical-design.md section this task implements
- DoD: definition of done (verifiable)
- Blocked: prerequisite and readiness condition; not executable until resolved

## Tasks

### Sequential (must be done in order)

- [x] T1. Add the voice setup use case — application — M
      — ticket: `LIVE-POC-T1`
      — depends: none
      — ref: design §3, §4, §6
      — DoD: `Negotiate` trims the offer, rejects a blank offer before the port call, passes the design's instruction text, returns the answer, and surfaces port failure as `ErrSessionUnavailable`. Tests cover those cases with a generated `Negotiator` mock. `gofmt` and the package comment rule are clean. `make test` passes this package.

### Parallel (can be done simultaneously)

- [x] T2. Add the Realtime call adapter — infrastructure — M
      — ticket: `LIVE-POC-T2`
      — depends: T1
      — ref: design §4, §6, §7
      — DoD: A local HTTP server proves one `POST` to the configured URL, the server `Authorization` value, multipart `sdp` and `session` fields, `server_vad` with `create_response` true, and an answer body returned unchanged. A provider error does not expose the provider body. An empty key does not dial. No live provider call. `make test.all` covers this package, or the package test is run directly and `make test.all` is left to T7.

- [x] T3. Add the setup HTTP contract — delivery — M
      — ticket: `LIVE-POC-T3`
      — depends: T1
      — ref: design §2, §7
      — DoD: `POST /voice-sessions` tests prove `201` with only `data.sdp`, `400` for unknown fields and a blank offer, `405` for the wrong method, and `503 SESSION_UNAVAILABLE` without a technical cause. `pkg/errs`, both translation catalogs, `docs/api/openapi.yaml`, `docs/api/voice-session.md`, and `docs/api/CHANGELOG.md` (`0.2.0`) include the route. `make test.contract` passes. Task schemas are unchanged.

- [x] T4. Add the prototype voice page — delivery — L
      — ticket: `LIVE-POC-T4`
      — depends: none
      — ref: design §5, §6, §7
      — DoD: `GET /voice-session` serves embedded vanilla HTML with Start Conversation, Stop Conversation, the five statuses, and a synthetic-data notice. The asset has no provider key. The script requests microphone permission before `POST /voice-sessions`, keeps one peer connection for later turns, sends one `response.create` from data-channel open and not from the remote track, releases microphone tracks and the peer connection on stop and failure, and does not start a replacement session. A delivery test checks the document, controls, and key absence. `POST /voice-session` returns `405`.

### Sequential (compose, then verify)

- [ ] T5. Compose voice routes into the process — infrastructure / cmd — M
      — ticket: `LIVE-POC-T5`
      — depends: T2, T3, T4
      — ref: design §4, §8
      — DoD: `/voice-session` and `/voice-sessions` reach their handlers. `/tasks`, `/health`, and unknown routes still reach the Task handler. The process starts without `OPENAI_API_KEY`. Shutdown admission still rejects new work with `SHUTTING_DOWN`. Existing runtime startup and shutdown tests pass. `.mockery.yml` lists the new interfaces and `make mocks` output is current.

- [ ] T7. Run the repository check — verification — S
      — ticket: `LIVE-POC-T7`
      — depends: T5
      — ref: design §8
      — DoD: `make check` passes. If `TEST_DATABASE_URL` is unset, database suites are reported as skipped and are not described as database verification. No live provider call is required.

### Blocked / Deferred

- [ ] T6. Record the spoken feasibility check — manual trial — M
      — ticket: `LIVE-POC-T6`
      — depends: T5
      — ref: design §5, §9
      — blocked by: a local `OPENAI_API_KEY` with Realtime access, plus a browser and microphone
      — ready when: those resources are available on the developer machine
      — DoD: Notes record the greeting before the first answer, three same-session turns with a contextual reference, Stop and a later new Start, and visible permission, setup, and drop failures without an automatic new session. Notes confirm the browser response does not contain the long-lived key. If the provider session remains active after the peer connection closes, the page sends the provider's documented close event before local release and the notes say so. Automated tests are not a substitute for this check.

## Dependency Graph

```text
T1 → T2 ─┐
T1 → T3 ─┼→ T5 → T7
T4 ──────┘   └→ T6 (after a local key, browser, and microphone are available)
```

T2 and T3 wait for T1. T4 can start immediately because the page contract is already fixed in the technical design. T2, T3, and T4 do not depend on each other.
