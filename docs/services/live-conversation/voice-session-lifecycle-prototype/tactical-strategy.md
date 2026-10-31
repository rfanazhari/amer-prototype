# Tactical Strategy: Voice session lifecycle prototype

## 1. Implementation Strategy

Additive. Task routes, Task persistence, and health stay in place. The voice page and `POST /voice-sessions` are new. No database migration is included.

Build the setup operation and its provider port first, then the HTTP contract and the browser page in parallel. Compose them into the existing process only after those pieces exist. The spoken feasibility check runs last, when a provider key and a microphone are available.

## 2. Code Reuse Map

| Component | Location | Used for | Fit and limitations |
| --- | --- | --- | --- |
| Task HTTP handler | `delivery/http/handler/taskhandler/handler.go` | Envelope shape, strict JSON, localized errors, and the fallback for non-voice routes | Fit for the public JSON contract. Its request types and route parser are Task-specific, so the voice handlers do not call into it. Unknown routes remain its 404. |
| Error codes and catalogs | `pkg/errs/codes.go`, `translations/errors.en.toml`, `translations/errors.id.toml` | Stable public codes and `Accept-Language` | `SESSION_UNAVAILABLE` is added beside the current codes. Existing Task codes stay. |
| Runtime composition | `infrastructure/runtime/runtime.go`, `infrastructure/runtime/http.go`, `cmd/app/main.go` | Process startup, listener, shutdown admission, and log masking | Voice settings are passed through `Config`. Startup still requires `DATABASE_URL`. An empty `OPENAI_API_KEY` must not block Task startup. Shutdown admission already returns `SHUTTING_DOWN` before the handler runs. |
| Log masking | `cmd/app/main.go` | Masks `authorization`, `token`, and `api_key` | Covers the provider credential if a log field uses those names. The adapter still must not log the key or the provider body. |
| Architecture checks | `test/architecture/dependencies_test.go`, `test/architecture/documentation_test.go` | Layer direction and required comments | New voice types follow the same import rules. Domain imports do not change. |
| OpenAPI validation | `test/contract/openapi_test.go`, `docs/api/openapi.yaml` | Structural contract check | The check validates the document. Handler tests still prove the route behavior. |
| Mock generation | `.mockery.yml`, `make mocks` | Operation and port doubles | Add `NegotiateOperation` and `Negotiator` when those interfaces exist. |
| Task use cases, repository, queue, and clock | `application/usecase/taskusecase`, `infrastructure/repository/taskrepository`, `infrastructure/service/eventqueue` | Unchanged Task behavior | They do not fit voice setup. Voice has no stored aggregate, no completion event, and no time rule. |

## 3. Net-New Code Inventory

| Item | Layer/Component | Package/Path | Type | Ownership / Notes |
| --- | --- | --- | --- | --- |
| Interviewer instruction text | application | `application/usecase/voicesession` | Behavior of `Negotiate` | Owns language, opening response, contextual reply, and the score and question-bank prohibitions. Not a separate type. |
| `Negotiate` / `NegotiateOperation` | application | `application/usecase/voicesession` | Use case and operation interface | Setup request boundary consumed by delivery. |
| `Negotiator` | application | `application/usecase/voicesession` | Port interface | One exchange method. Implemented only by the provider adapter. |
| `ErrInvalidOffer`, `ErrSessionUnavailable` | application | `application/usecase/voicesession` | Sentinel errors | Mapped by the handler. |
| Setup handler | delivery | `delivery/http/handler/voicesessionhandler` | HTTP handler | `POST /voice-sessions`, envelope, and error mapping. |
| Prototype page | delivery | `delivery/http/voicepage` | Browser client, embedded | Manual start, microphone, one peer connection, opening `response.create`, stop, and visible failure. |
| Provider call adapter | infrastructure | `infrastructure/service/realtimecall` | Adapter | Holds the key and sends the Realtime calls request. |
| Voice route dispatch and config fields | infrastructure / `cmd/app` | `infrastructure/runtime`, `cmd/app/main.go` | Composition | Selects voice handlers without taking over Task routes. |

No domain aggregate, entity, value object, repository, or DTO is added. The setup result is the answer string.

## 4. Branching / PR Strategy

One branch and one pull request is enough. The page, setup route, and adapter are one candidate-facing path, and splitting them would leave a route without a client or a client without a server.

Use one commit per task. Start the subject with that task's ticket ID:

`LIVE-POC-T1: <why the change is needed>`

If a task needs a follow-up commit, keep the same ticket ID. Do not mix Task-feature edits into these commits beyond the runtime dispatch and shared error catalog this design names.

## 5. Testing Strategy

Commands come from the repository `Makefile`. Planned checks below are new. Existing passing Task tests are regression coverage, not evidence that voice behavior works.

`make test` runs domain and application race checks. `make test.all` includes delivery and infrastructure. `make test.contract` validates OpenAPI. `make test.architecture` checks imports and comments. `make lint` runs golangci-lint. The combined `make check` is the final automated gate. Database suites stay skipped unless `TEST_DATABASE_URL` is set; voice work does not require them. No live OpenAI call belongs in automated tests.

### Unit Tests

- `Negotiate`: a non-empty offer is passed through with the exact instruction text; the returned answer is unchanged; a blank or whitespace offer returns `ErrInvalidOffer` and does not call `Negotiator`; a port failure returns `ErrSessionUnavailable`.
- Adapter, against a local HTTP test server: the outbound request uses the server key, posts the offer and session JSON, and returns the answer body; a non-success provider response does not surface the provider body; an empty key does not dial.
- Setup handler, with `NegotiateOperation` mocked: `201` body is only `status`, `data.sdp`, and no key field; unknown JSON fields, a missing `sdp`, and a blank `sdp` return `400 INVALID_INPUT`; the wrong method returns `405`; `ErrSessionUnavailable` returns `503 SESSION_UNAVAILABLE` without a technical cause.
- Page handler: `GET /voice-session` returns the document and the FSD controls; the asset contains no `OPENAI_API_KEY` and no `Authorization` bearer value. The script contains one opening send on the data-channel open path and does not send it from the remote-track handler.
- Catalog test includes `SESSION_UNAVAILABLE` in English and Indonesian.

### Integration Tests

- Happy path, with the adapter pointed at a local provider double and the composed router: a valid offer returns the double's SDP answer, and the double received the server key.
- Error cases: missing key, provider `500`, malformed setup body, and shutdown admission on the setup route return the mapped envelopes and do not create a Task row.
- No automated browser-to-OpenAI test. The spoken path is the manual check.

### Regression Tests

- Existing Task handler tests, including unknown route `404`, method errors, and the completion queue.
- `GET /health` remains `200` with an empty success data object.
- Runtime startup and shutdown tests still pass when voice settings are empty.
- `make test.contract` and `make test.architecture` pass after the OpenAPI and catalog edits.

### Manual feasibility check

Record the result outside the automated suite, in the implementation notes for `LIVE-POC-T6`:

- Start with the microphone allowed, then hear an Indonesian greeting and one opening question before answering.
- Three exchanges in the same session. The later reply refers to an earlier answer, using the capability's RabbitMQ example.
- An English technical term inside Indonesian is followed without a session restart.
- Stop releases the microphone and the connection. A later Start is a new session.
- Denied permission, a forced setup failure, and a dropped connection each show a message and wait for Start.
- The browser network response and page source do not contain the long-lived key.

## 6. Rollout & Rollback

- Rollout: local process only. Set `OPENAI_API_KEY` in the environment when running the spoken check. Leave it unset for ordinary Task development. Do not commit the key, a `.env` file, or provider response bodies. No feature flag: the page is a new route, and Task clients do not call it.
- Rollback: revert the voice route, page, adapter, and `SESSION_UNAVAILABLE` catalog entry. No data migration or backfill is involved. In-flight browser sessions end when the process stops or the candidate closes the page.
