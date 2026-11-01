# Administrative identity and access HTTP contract

All responses use the standard JSON envelope. `Accept-Language: id` selects Indonesian. Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, and system-owned fields are `400` `INVALID_INPUT`. Every access response sends `Cache-Control: no-store`. Task, voice, and health routes stay unauthenticated.

The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md).

## Origin, cookies and CSRF

Every administrative request requires `Origin` equal to `ACCESS_HTTP_ORIGIN`. Same-origin GET and HEAD omit `Origin` in browsers; those requests are accepted when `Host` matches `ACCESS_HTTP_ORIGIN`. A present mismatched `Origin` is still rejected. Requests with bodies also require `Content-Type: application/json`. HTTPS uses cookie `__Host-ia-admin` (Secure, HttpOnly, SameSite=Lax, Path=/, host-only). HTTP localhost uses `ia-admin` with the same flags except `__Host-` and Secure. Candidate cookies `ia-candidate` and `__Host-ia-candidate` never authenticate administrative routes.

Login has no CSRF header. Authenticated unsafe methods require `X-CSRF-Token` matching the current administrative session. `GET /admin/session` returns `csrfToken`. Login is limited to 5 attempts per canonical email and 20 per client IP per 15 minutes; excess is `429` `IA_RATE_LIMITED` with `Retry-After`.

## Routes

POST `/admin/sessions` accepts exactly `email` and `password` and returns `201` with `account` `{id,name,email,role}` and `session` `{id,expiresAt}` after the session commit. The cookie is the credential and is omitted from JSON.

GET `/admin/session` returns `200` with `account` `{id,name,email,role,status}`, `session` `{id,expiresAt}`, and `csrfToken`.

DELETE `/admin/sessions/current` requires the cookie and CSRF header, accepts an empty object or absent body, returns `200` with `{}` after persisted revocation, then clears the cookie. Uncertain commit still clears the local cookie and returns `503` `IA_DEPENDENCY_UNAVAILABLE`.

POST `/admin/hr-accounts` requires a current ADMIN session and CSRF, accepts exactly `name`, `email`, and `password`, and returns `201` with `id`, `name`, `email`, `role` (`HR`), `status` (`ACTIVE`), `createdAt`, and `updatedAt`.

POST `/admin/candidates` requires current HR and CSRF, accepts exactly `name` and `email`, and returns `201` with `id`, `name`, `email`, `createdAt`, and `updatedAt`. Email is contact only and never unique.

GET `/admin/candidates` requires current HR, origin, and the administrative cookie. It has no request body, CSRF header, or query parameters. It returns `200` with an array of `id`, `name`, `email`, `createdAt`, and `updatedAt` for candidates that account created, newest first (`createdAt` descending, then `id` descending). An empty result is an empty array. ADMIN is `403` `IA_FORBIDDEN` before the read. A missing session or a candidate cookie is `401` `IA_UNAUTHENTICATED`.

GET `/admin/candidates/{id}` is unchanged. It requires current HR and returns the same identity fields for any known candidate. Unknown identities are `404` after authorization.

POST `/admin/interview-assignments` requires current HR and CSRF, accepts exactly `candidateId` and `instanceId`, mints the assignment id, copies the current `hr-interviewer` active revision, and returns `201` with `id`, `candidateId`, `instanceId`, and `publishedRevisionId`. A Document Analyzer instance is `409` `IA_UNSUPPORTED_TEMPLATE`. An unpublished HR Interviewer instance is `409` `IA_NO_ACTIVE_REVISION`. Unknown candidate or instance is `404` `NOT_FOUND`. A lost `201` creates another assignment. The invitation path must use `data.id`, not the candidate id.

POST `/admin/interview-assignments/{id}/invitations` requires current HR and CSRF, accepts an empty object or absent body, and returns `201` with `id`, `assignmentId`, `expiresAt`, and `token` once.

## Candidate landing, exchange and session

GET `/candidate/access` serves the invitation landing HTML with `Cache-Control: no-store` and `Referrer-Policy: no-referrer`. The page calls `history.replaceState` to remove the invitation token from the URL before any exchange request. GET/prefetch cannot exchange. POST or HEAD on the landing path is `405` `METHOD_NOT_ALLOWED` and does not start an interview.

POST `/candidate/access/exchange` requires origin and `Content-Type: application/json`, accepts exactly `token`, and returns `201` with `candidate` `{id,name}`, `assignment` `{id}`, and `session` `{id,expiresAt}` after commit. HTTPS uses cookie `__Host-ia-candidate`; HTTP localhost uses `ia-candidate`. GET/prefetch cannot exchange. Unavailable invitations are `410` `IA_INVITATION_UNAVAILABLE` with no cookie. Uncertain commit is `503` with no cookie. Exchange is limited to 5 attempts per invitation token and 20 per client IP per 15 minutes.

GET `/candidate/session` requires the candidate cookie. Browsers omit `Origin` on same-origin GET; a missing `Origin` is accepted when `Host` matches `ACCESS_HTTP_ORIGIN`. It returns `200` with the same candidate/assignment/session fields plus `csrfToken`, and never exchanges or starts an interview. Browser refresh reuses the existing session. Administrative cookies cannot authenticate this route.

After `GET /candidate/session` succeeds, the landing page navigates once to `/candidate/interview/page`; the interview routes are documented in [candidate-interview.md](candidate-interview.md). A failed exchange still shows the unavailable message.

Authenticated candidate unsafe methods require `X-CSRF-Token` bound to credential kind `candidate`.

## Errors

Origin, content-type, size, and JSON failures are `400` `INVALID_INPUT`. Missing or substituted administrative sessions are `401` `IA_UNAUTHENTICATED`. Invalid CSRF and insufficient role are `403` `IA_FORBIDDEN`. Duplicate canonical email is `409` `IA_ACCOUNT_CONFLICT`. Failed login proof is `401` `IA_UNAUTHENTICATED`. Missing, expired, revoked, or inaccessible invitations are `410` `IA_INVITATION_UNAVAILABLE`. Store uncertainty is `503` `IA_DEPENDENCY_UNAVAILABLE`. Unpublished HR Interviewer instances are `409` `IA_NO_ACTIVE_REVISION`. Non-HR Interviewer instances are `409` `IA_UNSUPPORTED_TEMPLATE`. Shutdown admission remains `503` `SHUTTING_DOWN`.
