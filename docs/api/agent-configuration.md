# Agent configuration HTTP contract

All responses use the standard JSON envelope. `Accept-Language: id` selects Indonesian. Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, and system-owned members are `400` `INVALID_INPUT`. Every configuration response sends `Cache-Control: no-store`. Task, voice, and health stay unauthenticated. Candidate cookies never authorize these routes.

The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md).

## Origin, cookies and CSRF

Every configuration request requires `Origin` equal to `ACCESS_HTTP_ORIGIN`. Requests with bodies also require `Content-Type: application/json`. HTTPS uses cookie `__Host-ia-admin`; HTTP localhost uses `ia-admin`. Authenticated unsafe methods require `X-CSRF-Token` matching the current administrative session. Failures use ACP codes, not `IA_*` codes.

## Routes

GET `/admin/agent-templates` returns `200` with `templates`: `templateId`, `templateVersion`, `behaviorVersion`, `displayName`, `configurationSchema`, and `warningPolicy` (`informational`) for HR Interviewer and Document Analyzer.

POST `/admin/agent-instances` requires CSRF and accepts exactly `templateId`, `templateVersion`, `name`, and `configuration`. It returns `201` with the instance view. Creation is not idempotent. Compensation references are write-only; responses expose `salaryCeilingConfigured` and never the ceiling identity or value.

GET `/admin/agent-instances/{id}` returns `200` with `id`, `name`, `templateId`, `templateVersion`, `behaviorVersion`, `draftVersion`, filtered `configuration`, nullable `activeRevisionId`, `createdAt`, and `updatedAt`.

PUT `/admin/agent-instances/{id}/draft` requires CSRF and a complete `configuration` plus `expectedDraftVersion`. It returns `200` with the incremented draft. Stale versions are `409` `ACP_DRAFT_CONFLICT`.

POST `/admin/agent-instances/{id}/validations` requires CSRF and `expectedDraftVersion`. It returns `201` with `validationId`, versions, `eligible`, `findings`, and `createdAt` even when findings block publication.

POST `/admin/agent-instances/{id}/publications` requires CSRF, `expectedDraftVersion`, `validationId`, and one `Idempotency-Key` header. New publication is `201`; same-intent replay is `200` and does not change the active revision. Different intent is `409` `ACP_IDEMPOTENCY_CONFLICT`. Blocking findings are `422` `ACP_PUBLICATION_BLOCKED`.

GET `/admin/agent-instances/{id}/revisions/{revision_id}` returns `200` with the immutable revision view.

These paths are dispatched before the administrative `/admin/` access catch-all.

## Errors

Missing, expired, revoked, or candidate sessions are `401` `ACP_UNAUTHENTICATED`. Insufficient HR authority and invalid CSRF are `403` `ACP_FORBIDDEN`. Malformed input is `400` `INVALID_INPUT`. Absent or out-of-scope resources are `404` `NOT_FOUND`. Uncertain publication commit is `503` `ACP_PUBLICATION_UNCERTAIN` and asks for the same-key retry. Reference or store outage before acceptance is `503` `ACP_DEPENDENCY_UNAVAILABLE`. Technical or corrupt storage is `500` `INTERNAL_ERROR`. Messages never include salary references, configuration bodies, or database causes.
