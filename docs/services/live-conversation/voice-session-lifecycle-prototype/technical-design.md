# Technical Design: Voice session lifecycle prototype

## 1. Overview

A test candidate opens one live spoken session from a page served by the existing Go service, talks with a simple interviewer, and stops the session. The Go service creates that provider session only after an explicit start, keeps the long-lived provider key on the server, and returns the WebRTC answer. The browser keeps the same peer connection for later turns. The provider detects the end of a candidate turn and speaks the next reply. The prototype does not score answers, select from an HR question bank, or keep a transcript after the session.

Source behavior is `docs/services/live-conversation/business/capabilities/voice-session-lifecycle-prototype.md` (`BR-LIVE-POC-001` through `BR-LIVE-POC-004`, `BR-LIVE-POC-010` through `BR-LIVE-POC-014`, and `BR-LIVE-POC-020` through `BR-LIVE-POC-022`). Transport and turn ownership follow `docs/services/live-conversation/ADR-001_Realtime_Voice_Integration_v0_Updated.md`. The functional baseline is `docs/services/live-conversation/AI_Interview_Agent_Prototype_v0_FSD.md`.

The current service is the Task Tracking HTTP API. Voice setup is a new route on that process. Task persistence, completion events, and task routes stay as they are. No voice code exists yet.

Provider behavior below was checked on 2026-10-31 against the OpenAI Realtime WebRTC guide (`https://developers.openai.com/api/docs/guides/voice-webrtc`), the Realtime conversations guide, and the voice-activity-detection guide. The ADR's older guide URLs return 404; those three current pages are the replacement evidence. The ADR remains the binding transport decision.

### Goals

- Serve a local page with Start Conversation, Stop Conversation, and the statuses Idle, Connecting, Connected, Disconnected, and Error.
- Open a provider session only after the candidate starts and the browser has microphone permission.
- Exchange one WebRTC offer for one answer without sending the long-lived provider key to the browser.
- Configure the session so the interviewer speaks Indonesian, accepts English technical terms, greets and asks one opening question when the first response is requested, and later refers to an earlier answer in the same session.
- Keep later turns on the same browser peer connection, with provider turn detection creating the later spoken replies.
- Show permission denial, setup failure, and a dropped connection, then wait for another manual start.
- Release the microphone and the peer connection when the candidate stops or the attempt fails.

### Non-Goals

- Scoring, pass/fail, compensation, a mandated question bank, or backend interview policy.
- A transcript or recording kept after the session.
- Go-side handling of each spoken turn, an audio relay through Go, or a server-side realtime audio client.
- Persisting the voice session, or changing the Task schema.
- End-user authentication, a non-local deployment, and acceptance or rejection of ADR-001. The ADR stays Proposed until a separate result is recorded.
- A locked model version, voice name, or turn-detection tuning beyond the initial configuration in this design.

## 2. API / Interface Contract Changes

Task routes and `GET /health` stay as documented in `docs/api/task.md`. Unknown routes still use the standard JSON 404 envelope. The voice page and setup route are additive.

| Method and path | Behavior |
| --- | --- |
| `GET /voice-session` | Returns the prototype HTML page. |
| `POST /voice-session` | `405` `METHOD_NOT_ALLOWED`. |
| `POST /voice-sessions` | Accepts one JSON object `{"sdp":"<WebRTC offer>"}` and returns `201` with the provider SDP answer. |
| `GET /voice-sessions` | `405` `METHOD_NOT_ALLOWED`. |

`POST /voice-sessions` uses the existing envelope, strict JSON rules, `Accept-Language`, and the 1 MiB body limit:

```json
{"status":"success","data":{"sdp":"<SDP answer>"}}
```

Error responses stay `{"status":"error","data":{},"error":{"code":"...","message":"..."}}`.

| Condition | HTTP | Code |
| --- | --- | --- |
| Malformed JSON, unknown field, non-object body, missing `sdp`, or blank `sdp` | 400 | `INVALID_INPUT` |
| Unsupported method on a known voice path | 405 | `METHOD_NOT_ALLOWED` |
| Provider setup failed, the provider call could not be completed, or `OPENAI_API_KEY` is unset | 503 | `SESSION_UNAVAILABLE` |
| Unexpected handler failure | 500 | `INTERNAL_ERROR` |
| Admission stopped during process shutdown | 503 | `SHUTTING_DOWN` |

`SESSION_UNAVAILABLE` is a new public code. English message: `voice session unavailable`. Indonesian message: `sesi suara tidak tersedia`. The message and body do not include the provider key, the provider response body, or other technical causes. Those causes are logged inside the process.

The success body contains `sdp` only. It does not contain the provider key, an authorization header, session instructions, a score, or a transcript.

Update `docs/api/openapi.yaml`, add `docs/api/voice-session.md`, and record the addition in `docs/api/CHANGELOG.md` as additive contract `0.2.0`. Task request and response schemas stay unchanged.

The browser calls `POST /voice-sessions` on the same origin. No CORS surface is added. The page is used through the Go server, not as a file opened from disk.

## 3. Data Model & Schema Changes

No schema changes required.

The voice session is not stored. There is no repository, migration, GORM model, or `AutoMigrate` change. Task startup migration stays. `BR-LIVE-POC-013` and `BR-LIVE-POC-022` are satisfied by the absence of a score and the absence of a transcript store. An optional on-page event panel may hold events in memory until stop, failure, or navigation.

Go also does not keep an in-memory registry of active provider sessions. The browser owns the one live peer connection. Stop and failure release that connection locally. The next manual start creates a new provider session.

### Concepts and rules

Terms use the capability's meanings. A voice session is one live spoken conversation. A turn is one side's spoken part. The interviewer is the persona on that session, not an HR policy. The test candidate is the person who starts, speaks, and stops. "Connected" in the page means the data channel is open. It does not mean the greeting has already been heard.

The session identity that continues across turns is the browser peer connection together with the provider session created for that offer. Go's setup call is request-scoped and does not become that identity. A domain aggregate with requested, ready, active, and ended states would describe transitions Go cannot observe: microphone permission, data-channel readiness, later turns, and local teardown do not come back through a Go state machine. The Task aggregate is not a precedent for this session. Task rules are enforced on stored transitions; these voice rules are split across the page, one setup operation, and the provider session.

| Material rule | Owner | Enforcement paths | Proof plan |
| --- | --- | --- | --- |
| `BR-LIVE-POC-001` A session starts only from an explicit candidate request, and candidate speech is accepted only after microphone permission | Browser page for the gesture and permission; `Negotiate` accepts a setup only when it is invoked | Start sends `getUserMedia` first. Permission denial shows Error and does not call `POST /voice-sessions`. The setup route is the only provider call, and nothing in process startup or Task flow calls it | Page behavior covered by `T4`; handler and use case tests prove a blank offer and a missing request do not call the provider; `T6` observes a denied permission |
| `BR-LIVE-POC-002` Stop ends the session and releases the microphone and connection | Browser page | Stop stops microphone tracks, closes the data channel and peer connection, and clears playback. Start becomes available again | `T4` stop path; `T6` observes release after Stop |
| `BR-LIVE-POC-003` Later turns stay in the same session | Browser page keeps one peer connection; provider keeps that session's context | The page does not create a new offer for a later turn. A dropped connection is not repaired by a new offer | `T6` three exchanges in one session; drop case shows no automatic new session |
| `BR-LIVE-POC-004` Retry is a new manual start | Browser page | Permission denial, setup failure, and connection drop enable Start and do not call setup again | `T4` failure paths; `T6` |
| `BR-LIVE-POC-010` Indonesian, with English technical terms accepted | `Negotiate` session instructions, spoken by the provider | Instructions are set only on the server setup request. The page does not send `session.update` | Use case test asserts the instruction text; `T6` uses an English technical term inside Indonesian |
| `BR-LIVE-POC-011` Ready connection is not the greeting; the first requested response greets and asks one question | Browser sends one `response.create` after the data channel opens; instructions define that response | `ontrack` and ICE readiness do not send `response.create`. The open path sends it once | `T4` asserts the trigger is on data-channel open; `T6` hears the greeting before the candidate answers |
| `BR-LIVE-POC-012` A later reply refers to an earlier answer | Provider session context plus instructions; turn detection creates the reply | Server turn detection keeps `create_response` true. Go is not called per turn | `T6` RabbitMQ follow-up in the same session |
| `BR-LIVE-POC-013` No score | Absence in the use case, HTTP body, page, and store | No score field is written or returned | Handler test on the success body; review that the page has no score control |
| `BR-LIVE-POC-014` No mandated question bank or backend interview policy | Instructions and the lack of a policy component | Setup sends the conversational instructions below and no question-list payload | Use case test on the outbound instructions; no policy type in the inventory |
| `BR-LIVE-POC-020` Long-lived key stays on the server | `realtimecall` adapter and `cmd/app` configuration | The key is read from the environment and sent only as the provider `Authorization` header. Logs use the existing `api_key`, `authorization`, and `token` masks | Adapter test with a local provider double; handler test that the response has no key; page asset contains no key |
| `BR-LIVE-POC-021` Failure is visible | Browser page, using local messages and the API envelope | Permission and drop messages are shown locally. Setup failure shows the localized API message and Error | Handler error-envelope tests; `T4` and `T6` |
| `BR-LIVE-POC-022` No durable transcript | No voice repository; page event panel is memory only | Session end clears the panel. No voice table or file is written | Schema and inventory review; `T5` does not register a repository |

The server-owned instruction text is:

> You are a simple interviewer in a feasibility prototype. Speak Indonesian. English technical terms from the candidate are allowed and should be understood. When the first response is requested, greet the candidate in Indonesian and ask one opening question before waiting for an answer. On each later turn, stay in this same conversation, use what the candidate already said, and refer to an earlier answer from this session. Do not score, grade, pass, or fail any answer. Do not select questions from a fixed HR list, and do not apply compensation or hiring policy.

`Negotiate` owns this text. Wording may change only to make the same spoken behavior reliable, and the obligations in the table stay.

## 4. Component Breakdown

Project layers are the four root layers in `AGENTS.md`. Delivery HTTP tests are required by that instruction and by the existing `taskhandler` tests. The bundled backend-service note that skips delivery tests does not override it.

| Layer/Component | Status | Justification |
| --- | --- | --- |
| `domain/` | Reused as-is | No voice aggregate or value object. The rules Go can reject are request-scoped, and the rules that continue across turns are owned by the browser and the provider. See the assessment below. |
| `application/usecase/voicesession` | New | `Negotiate` is the setup operation. It rejects a blank offer, owns the instruction text, calls the provider port once, and returns the answer string. |
| `application/dto`, `application/repository`, Task use cases and queries | Reused as-is | The answer is one string. Task persistence and commands are not involved. |
| `delivery/http/handler/voicesessionhandler` | New | Parses the setup request, maps errors, and writes the JSON envelope. |
| `delivery/http/voicepage` | New | Serves the embedded vanilla page for the candidate-facing lifecycle. |
| `delivery/http/handler/taskhandler` | Reused as-is | Task and health behavior stay on the current handler. The voice handlers duplicate the small envelope writer so this feature does not refactor Task delivery. |
| `infrastructure/service/realtimecall` | New | Sends the server key and session configuration to the provider and returns the SDP answer. |
| `infrastructure/runtime` | Modified | Dispatches the two voice paths, then delegates every other path to the Task handler. |
| `cmd/app` | Modified | Reads voice settings and passes them into runtime configuration. |
| `pkg/errs`, `translations/` | Modified | Adds `SESSION_UNAVAILABLE` and both catalogs. |
| PostgreSQL, event queue, clock | Reused as-is | Voice setup does not use them. |

### Detail per component

`Negotiate` implements `NegotiateOperation`, the interface the HTTP handler calls. `Execute(ctx, offer string) (string, error)` trims the offer, returns `ErrInvalidOffer` when it is empty, and calls `Negotiator.Exchange` with the offer and the instruction text. Provider failure is `ErrSessionUnavailable`. The use case does not retry.

`Negotiator` is the port on `Negotiate`:

`Exchange(ctx context.Context, offer string, instructions string) (string, error)`

The adapter is the only implementation. It reads the key, model, voice, and provider URL from its struct fields. Those fields are filled from the environment at composition:

| Setting | Required behavior |
| --- | --- |
| `OPENAI_API_KEY` | Server-only. Empty value returns `ErrSessionUnavailable` and does not open a connection. Absence does not stop process startup. |
| `OPENAI_REALTIME_MODEL` | Initial value `gpt-realtime-2.1`, from the current unified-interface example. Override by environment. |
| `OPENAI_REALTIME_VOICE` | Initial value `marin`, from that same example. Override by environment. |
| `OPENAI_REALTIME_URL` | Initial value `https://api.openai.com/v1/realtime/calls`. The adapter calls `POST /v1/live/sessions` for that value. Tests point this at a local server. |

The provider request is one JSON `POST` made by `clientmanager` v0.10.1 to `POST /v1/live/sessions`. `CallBytes` sends `Authorization: Bearer <key>` through `AuthBearer`. The body is `{"session":{...},"transport":{"sdp":"<offer>","type":"webrtc"}}`. This replaces `POST /v1/realtime/calls`: a non-empty `application/sdp` part with `Content-Disposition: form-data; name=sdp` still returned `invalid_offer` / `failed to unmarshal SDP: EOF`. The session sets the Live model (`gpt-live-1` when the configured model is empty or `gpt-realtime-2.1`), the instruction text, and `audio.output.voice`. It does not send a question list, a score setting, or a transcription store. The returned answer is `transport.sdp` from the provider JSON. The call uses the request context, runs once, and starts no background work. That context must already carry a logmanager transaction; `HTTPServer` starts one before the handler runs. `clientmanager` then opens an API segment for the provider call. A context without that transaction, a transport failure, a non-success provider status, or a body without `transport.sdp` becomes `ErrSessionUnavailable`. The returned error does not include the provider key or the provider body. Shutdown already cancels that request context when drain fails, within the existing 4-second drain budget and 5-second process budget. v0.10.1 is the clientmanager release that requires this service's current `logmanager` v1.44.1. A newer clientmanager release would force a logmanager upgrade.

`voicesessionhandler` accepts only `POST`. It uses the same object, unknown-field, and size rules as Task JSON. `ErrInvalidOffer` is `400 INVALID_INPUT`. `ErrSessionUnavailable` is `503 SESSION_UNAVAILABLE`. `errs.Technical` is `500 INTERNAL_ERROR` and is logged. Successful data is `{"sdp":"<answer>"}`.

`voicepage` embeds one HTML document with plain CSS and JavaScript. `GET` returns that document. There is no frontend build and no package manager. The script's required behavior is in the flow section.

`runtime` checks the path before the Task handler. `/voice-session` goes to the page handler. `/voice-sessions` goes to the setup handler. Every other path, including unknown routes, `/tasks`, and `/health`, goes to the existing Task handler. Voice settings are added to `runtime.Config`. An empty provider key still allows `Run` when `DATABASE_URL`, the address, and logging are present.

`cmd/app` passes the four voice settings into that config. It does not log their values.

### Material structure decisions and assessment

| Choice | Reason and material alternatives/tradeoffs | Authority and refinement boundary | Assessment outcome |
| --- | --- | --- | --- |
| No voice type in `domain/` | A `VoiceSession` aggregate would need transitions for permission, readiness, later turns, and teardown. Go does not see those events after setup, so callers would push false state or the aggregate would not own the rule. A value object around a constant instruction string has no invalid value to reject. The Task title and state machine stay unrelated because their limits and transitions differ. The ordinary path is one use-case call; the failure path returns before or from that call. | Internal structure. Binding behavior is the rule table. A later capability that stores sessions or orchestrates turns would reopen this. | Retained |
| Live session JSON, `POST /v1/live/sessions` | `POST /v1/realtime/calls` still returned `invalid_offer` / `failed to unmarshal SDP: EOF` after the offer part was `application/sdp`, named `sdp` without quotes, and 3444 bytes long. The working local client creates the session with `client.Live.New`, which posts JSON and reads `transport.sdp`. The page still receives only the SDP answer. | Observed provider error plus the working client. Replacing the call is limited to setup. Model `gpt-live-1` replaces the realtime default for this endpoint. | Revised |
| `clientmanager` v0.10.1 for the provider call | The adapter uses `CallBytes`, `WithRequestBody`, and `AuthBearer` so the Live session call is an API segment on the request's logmanager transaction. v0.10.1 requires the service's current `logmanager` v1.44.1. A newer clientmanager release would upgrade logmanager. The standard-library client and the provider SDK stay unused. | Internal to the adapter. The port, the browser contract, the public error, and the session obligations stay. `HTTPServer` supplies the transaction the library requires. | Revised |
| JSON envelope around the SDP answer | The provider example returns raw SDP. This service's public contract uses the JSON envelope and strict objects. The page reads `data.sdp`. | `AGENTS.md` HTTP contract. The field name `sdp` is binding for this route. | Retained |
| Separate voice handlers, Task handler unchanged | Sharing an exported envelope helper would edit working Task delivery for a second writer of the same JSON shape. Tests lock both shapes. A third HTTP feature can extract a helper later. | Internal. The envelope fields and error codes are binding. | Retained |
| `server_vad` with automatic replies left on | The VAD guide defaults speech-to-speech sessions to `server_vad`, which creates the next response when the candidate stops. That matches ADR-001 §2.1 without a new turn policy. `semantic_vad` can be tried later if it hears turns more reliably. | ADR-001 leaves the specific detector as prototype configuration. `create_response` must stay true so Go is not inserted into later turns. | Retained |

## 5. Flow Sequence

Normal start:

1. The candidate opens `GET /voice-session`. The status is Idle, Start is enabled, and Stop is disabled.
2. The candidate chooses Start. The page disables Start and sets Connecting.
3. The page requests microphone permission. If it is denied, the page sets Error, shows the permission message, releases any partial capture, and enables Start. It does not call the setup route.
4. After permission is granted, the page creates one peer connection, attaches the microphone tracks, and creates the `oai-events` data channel before the offer.
5. Playback is attached from the remote audio track. That event does not request the greeting.
6. The page creates the offer, waits until ICE gathering completes, and `POST`s `{"sdp":"..."}` to `/voice-sessions`.
7. The handler validates JSON. `Negotiate` rejects a blank offer or calls the provider once. The provider receives the server key, instructions, model, voice, and turn detection.
8. The page sets the returned SDP as the remote answer. The response body is not treated as the greeting.
9. When the data channel opens, the page sends one `{"type":"response.create"}` event and does not attach new instructions. The status becomes Connected. The provider speaks the Indonesian greeting and one opening question.
10. The candidate speaks. Provider turn detection finishes the turn and creates the next spoken reply in the same session. The page does not send another setup request or another opening `response.create`.
11. The candidate chooses Stop. The page stops microphone tracks, closes the data channel and peer connection, clears playback and any in-memory event text, sets Idle, and enables Start.

Failure after setup has been attempted:

1. A non-success setup response sets Error, shows `error.message`, and runs the same local release as Stop. Start is enabled. The page does not call setup again.
2. A later peer-connection or data-channel failure sets Disconnected, shows a connection-dropped message, releases local media, and enables Start. The page does not open a replacement session.
3. Another attempt begins only when the candidate chooses Start again. That attempt is a new offer and a new provider session.

## 6. Request / Response (or Input / Output)

Setup request: `{"sdp":"<offer>"}`. `sdp` must be a JSON string and must not be empty after trimming. Any other field is rejected.

Setup success: `201` and `{"status":"success","data":{"sdp":"<answer>"}}`.

Page request: `GET /voice-session` returns the HTML document.

Provider request, hidden from the browser: multipart `sdp` plus `session`, with the server `Authorization` header. Provider success body: the SDP answer text.

Client event after the data channel opens: `{"type":"response.create"}` once.

### Creation, restoration, and transition contracts

| Path | Accepted inputs/states and normalization | Errors/order and failure timing | Side effects and compatibility | Proof plan |
| --- | --- | --- | --- | --- |
| Creation: one setup request | A non-empty SDP offer. Trim space before the empty check. No other normalization. Session instructions, model, voice, and turn detection are server-owned and are not taken from the request. | Unknown JSON fails before the use case. A blank offer fails before the provider call. Provider and missing-key failures return `SESSION_UNAVAILABLE` after no answer is available. There is no second provider attempt. | One billed provider session on a provider attempt. The long-lived key is not written to the response, page, or logs. Task data is untouched. | Use case tests for blank, success, and provider failure. Adapter test for one authorized request. Handler tests for unknown fields and the success body. |
| Restoration | Not applicable. No voice state is stored or loaded. | A process restart does not revive a browser session. | The candidate starts again manually. | Review of schema and composition. No restoration test. |
| Transition: stop, drop, and the next start | Stop is valid from Connecting and Connected. A drop is terminal for that peer connection. The next Start is valid only after local release, and only from a new click. | Permission denial fails before a provider session exists. Setup failure and drop fail without a replacement call. | The first connection's tracks are stopped before the next offer. No score or transcript remains. | Page tests for control state and the single opening event. `T6` for stop-then-start and for a drop. |

## 7. Error Handling

`ErrInvalidOffer` and `ErrSessionUnavailable` are sentinel errors in `application/usecase/voicesession`. The handler maps them as specified in the contract. Unexpected failures use `errs.Technical` and the existing internal-error path.

The page's local messages are:

| Situation | Status | Message |
| --- | --- | --- |
| Microphone permission denied | Error | `Microphone permission is required before the conversation can start.` |
| Setup request failed | Error | The API `error.message` |
| ICE gathering or peer connection failed, or the data channel closed before Stop | Disconnected | `The voice connection dropped. Start the conversation again to try another session.` |

Local messages are English, matching the FSD control labels. Spoken conversation remains Indonesian through the server instructions. API errors still follow `Accept-Language`; the page sends `Accept-Language: en` so the setup message matches the English page.

Provider response bodies are not copied into the public error or the browser. `clientmanager` records the provider status on the API segment. The process logger skips inbound headers and masks `authorization`. The SDP answer is not JSON, so the segment does not store it as a response object. A JSON provider error can appear in that trace. The child API segment in logmanager v1.44.1 does not inherit `WithSkipHeaders`; the authorization mask still covers the bearer value when a debug segment prints headers.

## 8. Constraints Checklist

- [ ] A voice session starts only from the candidate's Start action.
- [ ] Microphone permission is granted before the setup request, and candidate audio is attached only after that grant.
- [ ] Stop releases the microphone and the peer connection.
- [ ] Later turns reuse the same peer connection and provider session.
- [ ] Permission denial, setup failure, and a dropped connection do not start another session.
- [ ] The configured conversation is Indonesian, and English technical terms remain valid.
- [ ] The greeting and one opening question are requested after the data channel is open, not when media first arrives.
- [ ] A later audible reply is produced by the provider from this session's context.
- [ ] No score, pass, or fail is recorded.
- [ ] No HR question bank or interview policy selects the reply.
- [ ] The long-lived provider key is used only on the server.
- [ ] Setup and connection failures are visible.
- [ ] No transcript is required or stored after the session.
- [ ] Trial use stays on synthetic candidate information. The page states that.
- [ ] Task routes, health, persistence, and shutdown admission stay compatible.
- [ ] Public errors use the existing envelope and do not leak technical causes.

## 9. Risks & Open Questions

The provider can still speak off-brief. The instruction text and one `response.create` do not prove an audible Indonesian greeting or a contextual follow-up. `T6` is the proof, and it needs a key with realtime access, a browser, and a microphone. Automated tests do not replace it. Model and voice values are initial configuration from the current guide, not a completed spoken trial.

Closing the peer connection is the specified teardown. If `T6` shows the provider session still running after that close, the page sends the provider's documented close event before releasing local media. That observation does not block the automated tasks.

A reachable setup route can create billed sessions. This design limits the prototype to the local process and does not add sign-in. Exposure beyond the developer machine is out of scope until that decision is revisited.

No product decision is unresolved. `T6` waits on local trial resources, not on a design choice.

## 10. Revision History

| Version | Date | Change | Reason |
| --- | --- | --- | --- |
| v1 | 2026-10-31 | Initial design | Voice session lifecycle prototype, before implementation |
| v1.1 | 2026-10-31 | Provider setup uses `clientmanager` v0.10.1 | Trace the Realtime call on the existing logmanager transaction without upgrading logmanager |
