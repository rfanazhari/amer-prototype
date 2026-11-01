# API changelog

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
