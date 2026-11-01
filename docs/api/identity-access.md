# Administrative identity and access HTTP contract

All responses use the standard JSON envelope. `Accept-Language: id` selects Indonesian. Request bodies are limited to 1 MiB. Unknown fields, duplicate keys, non-objects, and system-owned fields are `400` `INVALID_INPUT`. Every access response sends `Cache-Control: no-store`. Task, voice, and health routes stay unauthenticated.

The machine-readable specification is [openapi.yaml](openapi.yaml); contract changes are recorded in [CHANGELOG.md](CHANGELOG.md).

## Origin, cookies and CSRF

Every administrative request requires `Origin` equal to `ACCESS_HTTP_ORIGIN`. Requests with bodies also require `Content-Type: application/json`. HTTPS uses cookie `__Host-ia-admin` (Secure, HttpOnly, SameSite=Lax, Path=/, host-only). HTTP localhost uses `ia-admin` with the same flags except `__Host-` and Secure. Candidate cookies `ia-candidate` and `__Host-ia-candidate` never authenticate administrative routes.

Login has no CSRF header. Authenticated unsafe methods require `X-CSRF-Token` matching the current administrative session. `GET /admin/session` returns `csrfToken`. Login is limited to 5 attempts per canonical email and 20 per client IP per 15 minutes; excess is `429` `IA_RATE_LIMITED` with `Retry-After`.

## Routes

POST `/admin/sessions` accepts exactly `email` and `password` and returns `201` with `account` `{id,name,email,role}` and `session` `{id,expiresAt}` after the session commit. The cookie is the credential and is omitted from JSON.

GET `/admin/session` returns `200` with `account` `{id,name,email,role,status}`, `session` `{id,expiresAt}`, and `csrfToken`.

DELETE `/admin/sessions/current` requires the cookie and CSRF header, accepts an empty object or absent body, returns `200` with `{}` after persisted revocation, then clears the cookie. Uncertain commit still clears the local cookie and returns `503` `IA_DEPENDENCY_UNAVAILABLE`.

POST `/admin/hr-accounts` requires a current ADMIN session and CSRF, accepts exactly `name`, `email`, and `password`, and returns `201` with `id`, `name`, `email`, `role` (`HR`), `status` (`ACTIVE`), `createdAt`, and `updatedAt`.

## Errors

Origin, content-type, size, and JSON failures are `400` `INVALID_INPUT`. Missing or substituted administrative sessions are `401` `IA_UNAUTHENTICATED`. Invalid CSRF and insufficient role are `403` `IA_FORBIDDEN`. Duplicate canonical email is `409` `IA_ACCOUNT_CONFLICT`. Failed login proof is `401` `IA_UNAUTHENTICATED`. Store uncertainty is `503` `IA_DEPENDENCY_UNAVAILABLE`. Shutdown admission remains `503` `SHUTTING_DOWN`.
