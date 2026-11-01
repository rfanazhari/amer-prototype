# T1 contract record — CORE-IAM-001

Status: **approved; T1 Done**. Prepared and recorded on 2026-11-01.

## Approval

- Confirming actor: the user in this conversation; no separate external sign-off is claimed.
- Date: 2026-11-01.
- Exact confirmation: exclusive T1 execution with a Done checklist for `CORE-IAM-001`.
- Scope: administrative sections A–E. Product choices stay inside approved non-goals (no mailer, no recovery UI, no public signup). Candidate Q3/Q4/Q5 are unchanged.
- Canonical accepted text is now in [technical-design.md, section 6](technical-design.md#6-request--response).

| Approved section | Canonical location |
|---|---|
| A — Identity, password delivery/acceptance and bootstrap | [§6.1](technical-design.md#61-administrative-identity-password-and-bootstrap-approved-a) |
| B — Cookies, origin, CSRF and rate budgets | [§6.2](technical-design.md#62-cookie-origin-csrf-and-rate-budgets-approved-b) |
| C — Administrative operation schemas | [§6.3](technical-design.md#63-administrative-operation-schemas-approved-c) |
| D — Duplicate accounts and existing bootstrap state | [§6.4](technical-design.md#64-duplicate-accounts-and-existing-bootstrap-state-approved-d) |
| E — Public errors and precedence | [§6.5](technical-design.md#65-public-errors-and-precedence-approved-e) |

## Design assessment and downstream obligations

Retained with rationale: AdministrativeAccount owns role/status/identity; sessions own lifetime/revocation; application owns bootstrap/provision uniqueness and commit timing; delivery owns origin/CSRF/cookies/rate; infrastructure owns Argon2id and cookie flags. T2/T3 foundations remain independent. T4 must persist canonical email uniqueness; T5/T8 must implement §6 budgets without logging secrets. OpenAPI/endpoint/changelog/runtime codes remain T8.

Q1 is resolved. Q2 assignment integration, Q3 exchange reuse, Q4 candidate contact matching and Q5 retention retain their gates. Only T1 documentation and the architecture contract test are in this change.

sqlmock and Mockery apply to later SQL and interface mocking; they are not used in T1.

## Subsequent migration-policy revision (2026-11-01)

The statements above record T1's scope at its closure; later administrative tasks T2–T9 are recorded Done in task-plan.md. The owner has now approved replacing service startup AutoMigrate with explicit, versioned SQL migrations; see [decision-log D8](decision-log.md#d8-database-changes-must-be-reviewed-and-applied-before-the-application-starts), [technical-design §3.1–3.4](technical-design.md#31-migration-policy-and-artifacts) and task-plan T16–T20. This is an approved schema-delivery policy change, not a revision of T1 A–E. Existing canonical emails, password hashes, account/session identities, bootstrap no-overwrite behavior, cookies/CSRF and public error contracts remain binding. The first ADMIN is still created through application bootstrap after schema validation, never through a credential seed migration. This revision plans the transition only; no SQL or Go implementation is added.
