# API changelog

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
