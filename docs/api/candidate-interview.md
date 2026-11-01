# Candidate interview HTTP contract

All responses use the standard JSON envelope. `Accept-Language: id` selects Indonesian. Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, and system-owned members are `400` `INVALID_INPUT`. Every JSON response sends `Cache-Control: no-store`.

The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md). Design: [technical-design.md](../services/core-ai/candidate-interview/technical-design.md).

This document covers interview read, notice acceptance, the candidate page, start, execution polling, takeover, and cancel. Session-attempt, ready, and report routes are absent. Execution Lifecycle remains unbound until T11; valid start and execution operations fail closed with `503 CIN_DEPENDENCY_UNAVAILABLE` after their input, session, CSRF, and applicable policy checks. The interview screen continues to return `execution: null`.

## Session, origin and CSRF

The routes reuse the candidate cookie (`ia-candidate`, or `__Host-ia-candidate` over HTTPS) issued by `POST /candidate/access/exchange`. Administrative cookies never authenticate them. `GET /candidate/session` returns `csrfToken`; unsafe methods send it in `X-CSRF-Token`. `Origin` must equal `ACCESS_HTTP_ORIGIN`. A same-origin GET may omit `Origin` when `Host` matches. Bodies require `Content-Type: application/json`.

`X-Interview-Controller` is the tab id the page keeps in `sessionStorage`. It tells the server whether the calling tab is the controller. It does not authenticate the candidate, and responses never return the stored controller id.

## Routes

### GET `/candidate/interview`

Returns `200` with `data`:

```json
{
  "notice": { "version": "notice-261102", "statement": "…" },
  "consent": { "id": "…", "noticeVersion": "notice-261102", "acceptedAt": "2026-11-02T08:05:00.000Z" },
  "execution": null,
  "actions": ["accept"],
  "progress": null
}
```

- `notice` is `null` and `actions` is empty when the notice catalog is empty.
- `consent` is `null` until the current notice version is accepted. A stored acceptance of an older version does not populate it.
- `execution` is `null` until Execution Lifecycle has an execution for the assignment. While that capability has no adapter, `execution` is always `null`.
- `actions` is a subset of `accept`, `start`, `takeover`, `cancel`.
- The view has no controller id, salary, transcript, or report member. A read never starts an interview and never records consent.

### POST `/candidate/consents`

Body exactly `{ "noticeVersion": "<version>", "accepted": true }`. Returns `201` with `data` `{ "id", "noticeVersion", "acceptedAt" }` on first acceptance and `200` with the same `data` when that assignment already holds that version, including after a concurrent first acceptance. Acceptance is insert-only; there is no withdrawal.

### POST `/candidate/executions`

Requires `Idempotency-Key` and `X-Interview-Controller`. The key uses `^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`; the controller id uses the same character rule with a 64-character maximum (`{0,63}`). Body exactly `{ "consentId": "<uuid>" }`; this references stored acceptance and never records consent. Returns `201` when committed by this call or `200` on replay of the same intent. Retry a lost response or `CIN_START_UNCERTAIN` with the same key and consent id.

### GET `/candidate/executions/{id}`

Returns `200` with the candidate-safe execution object below. Reads only; no execution or consent is inserted. `X-Interview-Controller` is optional on reads. Another assignment's execution id is `404 NOT_FOUND`.

```json
{
  "id": "22222222-2222-4222-8222-222222222222",
  "state": "CREATED",
  "version": 1,
  "hardDeadline": null,
  "isCurrentController": true,
  "controllerVersion": 1
}
```

This object is also `data` for successful start and cancel. State is `CREATED`, `RUNNING`, `COMPLETED`, `FAILED`, or `CANCELLED`. Deadlines use UTC millisecond precision; unset deadlines are null. No report, salary, transcript body, or controller id is returned.

### POST `/candidate/executions/{id}/takeover`

Body exactly `{ "controllerId": "<tab-id>", "expectedVersion": 1 }`. The body id must equal `X-Interview-Controller`; expectedVersion is a positive controller version. Returns `200` with `{ "isCurrentController": true, "controllerVersion": 2 }`. A stale version is `409 CIN_CONTROLLER_CONFLICT`.

### POST `/candidate/executions/{id}/cancel`

Body exactly `{ "expectedVersion": 1 }` using a positive **execution** version. Requires `X-Interview-Controller` for the current tab. Returns `200` with the committed execution in state `CANCELLED`. Another assignment's id is `404 NOT_FOUND`; a non-controller is `403 CIN_NOT_CONTROLLER`. Terminal state is never relabelled. Controller persistence is composed from T8; real cancellation awaits T11.

### GET `/candidate/interview/page`

Returns the candidate page as `text/html` with `Cache-Control: no-store` and `Referrer-Policy: no-referrer`. The page reads `GET /candidate/session`, then `GET /candidate/interview`, and displays only the offered accept/start/takeover/cancel actions. It stores the tab controller and start intent in `sessionStorage`, sends `X-Interview-Controller`, and retains the same start key and consent id after an uncertain response or refresh. Refresh never POSTs start. A second tab is read-only until takeover; cancel asks for confirmation. Live `CREATED`/`RUNNING` views poll the interview every two seconds; polling stops on access loss or a terminal outcome and superseded responses are ignored. `COMPLETED`, `FAILED`, and `CANCELLED` have distinct final screens. Without a cookie it shows the access-error state and no notice. It does not request a report or open a microphone. Other methods are `405` `METHOD_NOT_ALLOWED`. Production execution remains null until T11; T10 browser evidence uses API fixtures.

`GET /candidate/access` navigates to this page once, after `GET /candidate/session` succeeds.

## Errors

Precedence on the JSON routes:

1. Wrong method: `405` `METHOD_NOT_ALLOWED`.
2. Origin, content type, malformed JSON, unknown fields, `accepted: false`, or a notice version that is malformed or not current: `400` `INVALID_INPUT`.
3. Missing, expired, revoked, or substituted candidate session: `401` `IA_UNAUTHENTICATED`.
4. Missing or mismatched `X-CSRF-Token` on POST: `403` `CIN_FORBIDDEN`, before the body is acted on.
5. Execution outside the session assignment: `404 NOT_FOUND`.
6. Policy: `409 CIN_CONSENT_REQUIRED`, `403 CIN_SESSION_COVERAGE`, `409 CIN_IDEMPOTENCY_CONFLICT` on start; `409 CIN_CONTROLLER_CONFLICT` for stale controller/execution versions; `403 CIN_NOT_CONTROLLER`; `409 CIN_TERMINAL` or `409 CIN_NOT_LIVE`.
7. Empty notice catalog or acceptance read-back failure on consent: `503 CIN_NOTICE_UNAVAILABLE`. Unbound sibling adapters: `503 CIN_DEPENDENCY_UNAVAILABLE`. No consent is created by execution operations.
8. Ambiguous lifecycle commit: `503 CIN_START_UNCERTAIN`, retry the same key.
9. Corrupt stored data or any other technical failure: `500 INTERNAL_ERROR` without causes.

Execution POSTs reject malformed JSON, duplicate/unknown/system-owned fields, invalid tab ids, bad keys, invalid consent UUIDs, nonpositive versions and mismatched takeover tab ids before session lookup. Messages exist in English and Indonesian.
