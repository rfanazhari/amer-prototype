# T6 verification — CORE-IRT-006

Only T6 is implemented: `infrastructure/service/interviewvoice.Driver` against `service.InterviewDriver`.

## Design mapping and assessment

- `Create` posts one Live setup (bearer, `gpt-live-1`, CRLF-normalized SDP, application-built instructions),
  retains the provider session id and the SDP answer. Empty key never dials; non-success, transport failure and
  a malformed answer all map to `voicesession.ErrSessionUnavailable` without provider text or the key.
- `Attach` dials the sideband by provider session id with the bearer and keeps the connection by local session id.
- `Observe` returns an opaque `event` (raw payload, `type` field as metadata only) or a `disconnected` observation
  that releases the connection; it never touches coverage. Cancellation returns the context error.
- `SupplyContext` and `Control` write JSON frames; both require an attach (`ErrNotAttached`).
- `Close` sends `session.close` best effort, always releases the local connection, and returns a failure
  (`ErrTransport` or the context error). It implies no database rollback.
- No RabbitMQ import. No SQL, so sqlmock does not apply; the ports' Mockery mocks already exist from T2.

Assessment: **retained**. Transport stays in infrastructure; setup (`driver.go`) and sideband
(`driver_sideband.go`) are split by concern. The Live setup structs duplicate `realtimecall`'s on purpose:
that adapter drops the session id and stays the prototype path until `CORE-IRT-007`.

## Limits

- The sideband frame shapes for `SupplyContext` (`session.update` with `session.instructions`) and `Control`
  (`type` = command) are **not verified against the live provider**; only `session.close` appears in the
  existing probe. `H-01` and Q1/Q5 are not claimed. Confirm the frames with the S1 record before T8/T13.
- Package tests use a local fake HTTP/WebSocket server only; the public network is not called.

## RED checkpoint

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./infrastructure/service/interviewvoice/... 
```

Observed failure: every case failed on assertions against the declaration-only `Driver` (all methods returned
`interview voice transport is not implemented`); the tests compiled and ran. RED SHA
`c8b2fcdb0da66677bc75f8f724637f5b9ffd2977`, subject
`test: [CORE-IRT-006][rfanazhari] RED cover GPT-Live driver transport`.

## GREEN checks

Focused: `go test ./infrastructure/service/interviewvoice/... -race -count=1` passed. Then `make test`,
`make test.architecture`, `make lint` (0 issues), `make test.all` (no failures; database suites skip without
`TEST_DATABASE_URL`) and `make test.security` (0 called vulnerabilities). GREEN SHA
`d4b0d0d6f25fb835f7f3a59db0330d4c23026d0c`, subject
`feat: [CORE-IRT-006][rfanazhari] GREEN implement GPT-Live driver transport`.
