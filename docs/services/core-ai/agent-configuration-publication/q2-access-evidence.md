# ACP Q2 access evidence (CORE-IAM-009)

Date: 2026-11-01. This handoff supplies administrative access proof for Agent Configuration Q2. It does not mark ACP T8/T9 complete and does not claim interview or configuration-route acceptance.

## Proven by Identity and Access T8/T9

- Startup validates bootstrap name/email/password, `ACCESS_HTTP_ORIGIN`, and `ACCESS_CSRF_SECRET` without logging secrets, then bootstraps the first ADMIN when the store is empty.
- HTTP `POST /admin/sessions` → `POST /admin/hr-accounts` → HR `POST /admin/sessions` → `GET /admin/session` → real `AgentConfigurationAuthority` → `DELETE /admin/sessions/current`.
- Current HR `Authorize` is allowed for known configuration operations. Valid ADMIN configuration is forbidden. A candidate cookie cannot resolve an administrative principal.
- Invalid credentials issue no cookie. Duplicate HR email is `IA_ACCOUNT_CONFLICT`. Exact `now == expires_at` refuses the current session. Logout denies the old cookie. Restart reloads the committed session until logout.
- Concurrent HR provision against one canonical email collapses to one created account.

## Commands and limits

- `make test`, `make test.all`, `make test.contract`, `make test.architecture`, `make lint`, and `make test.security` are the local unit/contract checks.
- `make test.integration` and `make test.docker` require an authorized PostgreSQL URL. Without `TEST_DATABASE_URL`, `test/integration` access HTTP suites skip; a skip is not successful database verification.
- ACP T8/T9 remain responsible for real configuration routes, filtering, and publication replay. This evidence does not mount those routes.

## Not claimed

- Candidate invitation/exchange HTTP
- Real Interview assignment consumers
- ACP secured configuration endpoints. The 2026-11-01 ACP alignment treats this file as the satisfied administrative prerequisite for those routes; it still does not implement them.
