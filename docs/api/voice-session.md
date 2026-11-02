# Voice session setup (removed)

`GET /voice-session` and `POST /voice-sessions` are no longer routes of this service. The process does not register them, [openapi.yaml](openapi.yaml) does not describe them, and a request to either path receives the standard `404 NOT_FOUND` envelope from the Task fallback.

The interview voice path is `POST /candidate/executions/{id}/session-attempts` with `POST .../session-attempts/{attempt_id}/ready`, documented in [candidate-interview.md](candidate-interview.md). `test/livevoice` remains an opt-in probe and is not a route. Contract changes are recorded in [CHANGELOG.md](CHANGELOG.md).
