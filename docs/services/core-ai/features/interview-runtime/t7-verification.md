# T7 verification: remove the prototype voice routes

Ticket `CORE-IRT-007`. Date 2026-11-02.

- `../../../../../infrastructure/runtime/router.go` (renamed from `voice.go`) no longer has `Page`, `Setup`, `assembleVoice` or the `/voice-session` and `/voice-sessions` cases. Those paths fall through to Task and return its `404 NOT_FOUND` envelope. Shutdown admission still returns `503 SHUTTING_DOWN` for them.
- Deleted: `delivery/http/voicepage`, `delivery/http/handler/voicesessionhandler`, `infrastructure/service/realtimecall`, `Negotiate`, `Negotiator` and `NegotiateOperation`, with their generated mocks and `../../../../../.mockery.yml` entry.
- Kept: `voicesession.ErrSessionUnavailable`, which `interviewvoice` still returns for the T8 route (`errors.go`). `../../../../../test/livevoice` is unchanged and opt-in.
- Contract: `/voice-sessions`, its three schemas and the `voice-sessions` tag are gone from `openapi.yaml`; `voice-session.md` is a removal note; `CHANGELOG.md` 0.12.0.
- Tests: `TestRemovedVoicePathsFallThroughToTask`, `TestOpenAPIOmitsPrototypeVoiceRoutes`.

Limits: `OpenAI*` runtime config fields remain for T8 composition. Moving `ErrSessionUnavailable` into `interviewusecase` (it has `ErrProviderUnavailable`) is a later cleanup. The T8 routes are still not mounted, so until T8 composition lands the process has no voice route. `make test.integration` was not run (no `TEST_DATABASE_URL`).
