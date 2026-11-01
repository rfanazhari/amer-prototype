# Candidate interview HTTP contract

All responses use the standard JSON envelope. `Accept-Language: id` selects Indonesian. Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, and system-owned members are `400` `INVALID_INPUT`. Every JSON response sends `Cache-Control: no-store`.

The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md). Design: [technical-design.md](../services/core-ai/candidate-interview/technical-design.md).

This document covers the routes mounted so far: the interview read, notice acceptance, and the page shell. Start, poll, takeover, and cancel are not mounted. `/candidate/executions/...` is not a route of this service yet, and session-attempt and ready routes belong to Interview Runtime.

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

### GET `/candidate/interview/page`

Returns the candidate page shell as `text/html` with `Cache-Control: no-store` and `Referrer-Policy: no-referrer`. The page reads `GET /candidate/session`, then `GET /candidate/interview`, shows the notice with the accept action, and posts `POST /candidate/consents`. Without a cookie it shows the access-error state and no notice. It does not request a report. Other methods are `405` `METHOD_NOT_ALLOWED`.

`GET /candidate/access` navigates to this page once, after `GET /candidate/session` succeeds.

## Errors

Precedence on the JSON routes:

1. Wrong method: `405` `METHOD_NOT_ALLOWED`.
2. Origin, content type, malformed JSON, unknown fields, `accepted: false`, or a notice version that is malformed or not current: `400` `INVALID_INPUT`.
3. Missing, expired, revoked, or substituted candidate session: `401` `IA_UNAUTHENTICATED`.
4. Missing or mismatched `X-CSRF-Token` on POST: `403` `CIN_FORBIDDEN`, before the body is acted on.
5. Empty notice catalog, or an acceptance that cannot be read back: `503` `CIN_NOTICE_UNAVAILABLE`, with no consent written.
6. Any other failure: `500` `INTERNAL_ERROR` without causes.

The remaining `CIN_` codes are published for the start, takeover, and cancel routes that are mounted later: `CIN_CONSENT_REQUIRED`, `CIN_SESSION_COVERAGE`, `CIN_IDEMPOTENCY_CONFLICT`, `CIN_CONTROLLER_CONFLICT`, `CIN_NOT_CONTROLLER`, `CIN_TERMINAL`, `CIN_NOT_LIVE`, `CIN_DEPENDENCY_UNAVAILABLE`, and `CIN_START_UNCERTAIN`. Messages exist in English and Indonesian.
