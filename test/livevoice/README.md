# CORE-000: GPT-Live transport feasibility probe

Status: harness implemented; live acceptance gates remain unverified until observed.

This opt-in loopback test uses the existing Go module and clientmanager/logmanager.
It does not modify the production voice routes, execute migrations, or implement
platform lifecycle/checkpoint authority. The infrastructure helper is not wired
into production. No RabbitMQ or database is needed for this probe.

## Run

Supply `OPENAI_API_KEY` through the environment; do not paste it into commands or
logs. The harness does not load `.env` automatically. Then run from repository root:

```sh
LIVE_PROBE=1 go test ./test/livevoice -run TestManualProbe -count=1 -v -timeout 13m
```

Open the printed loopback URL in a microphone-capable browser. Use synthetic
speech only. Start a session, say a fictional Go-project answer, interrupt the
assistant, then close. Start another session to inspect fresh-session behavior.
Press **Finish probe** when done. At most four provider creation requests are
allowed. Each sideband is bounded to three minutes; the harness expires after
12 minutes. Provider calls incur usage. Setup HTTP timeout is 30 seconds.

The page preserves the existing microphone/peer-connection/ICE pattern but does
not send Realtime `response.create` on data-channel open. It observes GPT-Live
`session.started`; the backend attaches using the returned provider session ID.
No production activation readiness or stage transition is claimed.

## Evidence and limits

Only event-type counts are exposed, never transcript/audio payloads, SDP, session
IDs, or credentials. Counts do not prove event completeness, transcript finality,
question audibility, or durable progress. The in-memory generation check only
isolates this harness's sessions and is NOT the accepted database attempt fence.

`session.close` is sent on local cleanup with a one-second write deadline, then
the socket closes. This is **best effort, closure acknowledgement unconfirmed**.
If attachment fails after creation, provider cleanup is also unconfirmed; do not
repeatedly create sessions to work around that failure. A superseded setup may
similarly leave provider work needing investigation.

Normal `go test` skips the manual test. A passing manual test means only that at
least one sideband attached and the operator finished the harness. Record separate
observations for all acceptance gates; do not report the harness pass as S1–S4 pass.

| Gate | Required additional evidence |
| --- | --- |
| S1 | Final transcript/item identity, ordering/corrections, early-attachment gaps |
| S2 | Committed handoff before next-session activation, contextual carryover |
| S3 | Durable checkpoint resume, same execution identity, database stale fencing |
| S4 | Ambiguous setup, bounded replacement, cancellation and confirmed cleanup |

This first transport probe leaves these gates open. Preserve sanitized event
names/timings and explicit failures in the design evidence record; no raw audio
retention is added. Full transcript evidence collection requires a deliberate
synthetic-only capture contract rather than logging arbitrary provider payloads.

## Sources

- [GPT-Live WebRTC](https://developers.openai.com/api/docs/guides/voice-webrtc)
- [Server-side controls](https://developers.openai.com/api/docs/guides/voice-server-controls)

## Verification

- RED observed: missing identity validation and empty event summary assertions.
- GREEN: `go test -race ./infrastructure/service/liveprobe ./test/livevoice`.
- Live browser/provider results must be recorded separately; tests do not access
  the provider without `LIVE_PROBE=1`.

## CORE-000 local verification record

- RED checkpoint: `afacc52` (session identity validation and sanitized observation).
- Focused race tests, application/domain regression tests: passed.
- `make test.architecture`: passed using a temporary writable Go cache.
- `make lint`: passed, zero issues after HTTP status constants correction.
- `make test.security`: passed after network-enabled retry; zero reachable
  vulnerabilities, one imported-package and three required-module findings
  reported as not called.
- First manual launch: blocked by sandbox loopback bind permission. Relaunched
  with approved access; browser interaction/provider observations pending.
- Design assessment: retained a small infrastructure parser plus opt-in test-only
  server. Production routes and domain ownership remain unchanged. No database,
  queue, execution model or generic provider framework was introduced.
- Full S1–S4 remains incomplete. This harness is transport instrumentation, not a
  production driver or proof of durable recovery.
