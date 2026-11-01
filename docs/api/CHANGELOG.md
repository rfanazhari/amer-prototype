# API changelog

## Candidate lifecycle integration — T11 (2026-11-02)

The existing candidate routes now use the PostgreSQL lifecycle adapter for start, read and cancel. Repeated intent returns the same execution, different intent conflicts, and cancellation commits before success. Running deadlines derive from persisted activation. Interview credentials refuse revoked assignments. Routes, JSON schemas and public codes are unchanged. Progress/fencing remain T12; no report call is introduced. See [candidate-interview.md](candidate-interview.md).

## Candidate page — T10 (2026-11-02)

The existing HTML route now presents API-offered start/takeover/cancel actions, confirmed cancellation, and distinct completed/failed/cancelled outcomes. The tab controller and retryable start intent stay in sessionStorage. Live interview reads poll every two seconds and stop at terminal/access loss; refresh never starts an execution. No JSON route, schema, or status changed. Production lifecycle remains unbound pending T11; UI verification uses headless Chrome with API fixtures. See [candidate-interview.md](candidate-interview.md).

## 0.10.0 — candidate execution HTTP operations

Added `POST /candidate/executions`, `GET /candidate/executions/{id}`, `POST /candidate/executions/{id}/takeover`, and `POST /candidate/executions/{id}/cancel` ahead of the access catch-all. Start requires `Idempotency-Key` and returns 201/200 from the operation result. Execution POSTs require session, origin, CSRF and a tab id; takeover requires matching body/header ids. Foreign execution ids map to 404, classified policy errors to CIN codes, and wrapped storage validation failures to 500. Responses contain only the candidate-safe execution fields. ControllerStore is composed; lifecycle remains unbound and returns 503 for start/cancel. Session-attempt, ready and report routes remain absent. The T6 page shell is unchanged; controls are T10. See [candidate-interview.md](candidate-interview.md) and [openapi.yaml](openapi.yaml).

## 0.9.0 — candidate notice consent, interview read and page shell

Added `GET /candidate/interview`, `POST /candidate/consents`, and `GET /candidate/interview/page`. They are dispatched ahead of the `/candidate/` access catch-all and reuse the candidate cookie, origin, and CSRF; `POST /candidate/access/exchange` and `GET /candidate/session` are unchanged. Consent is `201` on first acceptance and `200` on replay. The read returns notice, consent, a null execution, actions, and a null progress, with no controller id, salary, or report member. Added `CIN_*` codes (`CIN_FORBIDDEN` for a bad CSRF header, `CIN_NOTICE_UNAVAILABLE` for an empty catalog, and the codes later start, takeover, and cancel routes use). The invitation landing now navigates to the page shell once after the session read succeeds. See [candidate-interview.md](candidate-interview.md) and [openapi.yaml](openapi.yaml).

## 0.8.0 — HR agent instance list

Added `GET /admin/agent-instances`. Current HR receives every instance, newest first (`createdAt` descending, then `id` descending). An empty result is `200` with `data` as an empty array. Each item keeps the filtered instance view: `salaryCeilingConfigured`, never the salary reference. POST create on that path is unchanged. See [agent-configuration.md](agent-configuration.md) and [openapi.yaml](openapi.yaml).

## 0.7.0 — creator-scoped candidate list

Added `GET /admin/candidates`. The logged-in HR receives only candidates that account created, newest first. An empty result is `200` with `data` as an empty array. `GET /admin/candidates/{id}` is unchanged. See [identity-access.md](identity-access.md) and [openapi.yaml](openapi.yaml).

## 0.6.1 — candidate session same-origin GET

GET `/candidate/session` accepts a missing `Origin` when `Host` matches `ACCESS_HTTP_ORIGIN`, matching browser same-origin GET. A mismatched `Origin` is still `400` `INVALID_INPUT`. See [identity-access.md](identity-access.md).

## 0.6.0 — interview assignment create

Added `POST /admin/interview-assignments`. Current HR creates a pin to the current `hr-interviewer` active revision. The server mints `data.id` for `POST /admin/interview-assignments/{id}/invitations`. Failures use `IA_*` codes plus `NOT_FOUND`. OpenAPI tags split former `admin` into `admin-only` (ADMIN account management) and `admin-hr` (HR configuration, candidates, assignments, invitations); shared session routes appear in both. See [identity-access.md](identity-access.md) and [openapi.yaml](openapi.yaml).

## 0.5.0 — agent configuration and publication

Added `GET /admin/agent-templates`, `POST /admin/agent-instances`, `GET /admin/agent-instances/{id}`, `PUT /admin/agent-instances/{id}/draft`, `POST /admin/agent-instances/{id}/validations`, `POST /admin/agent-instances/{id}/publications`, and `GET /admin/agent-instances/{id}/revisions/{revision_id}`. Routes reuse administrative cookies and CSRF, require HR authority, and map failures to `ACP_*` codes. Publication replay is `200`; new publication is `201`. See [agent-configuration.md](agent-configuration.md) and [openapi.yaml](openapi.yaml).

## 0.4.0 — candidate identity, invitation landing and session access

Added `POST /admin/candidates`, `GET /admin/candidates/{id}`, `POST /admin/interview-assignments/{id}/invitations`, `GET /candidate/access`, `POST /candidate/access/exchange`, and `GET /candidate/session`. Invitation landing removes the token from history before exchange; GET/prefetch cannot exchange. Candidate cookies, origin, CSRF, and `IA_INVITATION_UNAVAILABLE` are documented in [identity-access.md](identity-access.md). See [openapi.yaml](openapi.yaml).

## 0.3.0 — administrative identity and access

Added `POST /admin/sessions`, `GET /admin/session`, `DELETE /admin/sessions/current`, and `POST /admin/hr-accounts`. Task, voice, and health remain unauthenticated. Administrative cookies, origin, CSRF, rate limits, and `IA_*` error codes are documented in [identity-access.md](identity-access.md). See [openapi.yaml](openapi.yaml).

## 0.2.0 — voice session setup

Added `POST /voice-sessions`. The route accepts one WebRTC offer and returns `201` with `data.sdp` only. Invalid setup input is `400 INVALID_INPUT`. The wrong method is `405 METHOD_NOT_ALLOWED`. Provider or missing-key setup failure is `503 SESSION_UNAVAILABLE` (`voice session unavailable` / `sesi suara tidak tersedia`) without a technical cause. Task request and response schemas are unchanged. See [voice-session.md](voice-session.md).

## 0.1.0 — initial Task reference API

Formalized the existing Task and health HTTP contract in `openapi.yaml`: request validation, UTC millisecond timestamps, null dates, locale selection, envelopes, state-transition errors and shutdown/event-unavailable responses. No published API version is being replaced. Interview routes remain outside this feature.
