# Task HTTP contract

All responses are JSON. Success: `{"status":"success","data":...}`. Errors: `{"status":"error","data":{},"error":{"code":"...","message":"..."}}`, localized in English or Indonesian. Health returns 200 and an empty object in data; it indicates HTTP readiness only.

`POST /tasks` accepts `title` and optional `dueDate`, returns 201. `POST /tasks/{id}/start` and `/complete` accept an absent body or empty object; `PATCH /tasks/{id}/due-date` requires `dueDate`, where null clears it; `GET /tasks/{id}` returns details. Other successful task calls return 200. Details contain `id`, `title`, `status`, `dueDate`, `overdue`, `createdAt`, and `completedAt`; unset dates are null. IDs are UUID strings. Status is `todo`, `in_progress`, or `done`. UTC timestamps have millisecond precision; RFC3339 offsets are accepted, but more than three fractional digits are rejected.

Unknown fields, malformed input or UUID, invalid title/date: 400; missing task or route: 404; unsupported method: 405; forbidden transition or modification of Done: 409; queue acceptance failure: 503; unexpected technical failures: 500. Error messages must not expose technical causes.

The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md). `Accept-Language: id` selects Indonesian error messages. Request bodies are limited to 1 MiB. During shutdown admission refusal returns 503 with `SHUTTING_DOWN`; a full/unavailable completion queue returns 503 with `EVENT_UNAVAILABLE`. All validation and error responses use the same envelope.
