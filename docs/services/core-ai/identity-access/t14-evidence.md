# T14 evidence — CORE-IAM-014

Status: recorded 2026-11-01. This is Access proof against an Interview-shaped `interview_assignments` fixture. Access still does not ship or migrate that table.

## Browser runner commands

No Makefile browser target exists. Runnable local commands:

- `go test ./delivery/http/candidatelanding -count=1 -run TestInvitationLanding`
- `go test ./delivery/http/handler/accesshandler -count=1 -run 'TestCandidateExchange|TestCandidateIdentity'`
- Landing HTML calls `history.replaceState` before `POST /candidate/access/exchange` and uses `credentials` `include`. GET/prefetch cannot exchange.

## Repository verification

- `go test ./test/integration -count=1 -run TestCandidateAccessHTTPProvesAssignmentIsolationAndRecovery`
- Requires `TEST_DATABASE_URL`. When unset, the suite is skipped, not passed.
- `make test.integration` is the authorized real-database command.

Fixture notes: tests create a disposable Interview-shaped `interview_assignments` table for locking and facts. That is not an Access product schema and is not an in-memory fake of assignment facts.

## Per-rule evidence

| Rule / scenario | Evidence | Result |
|---|---|---|
| assignments A/B reuse one candidate with isolated sessions | integration: one candidate two assignments keep isolated sessions | proven when `TEST_DATABASE_URL` is set; skipped otherwise |
| exchange retry | integration: lost exchange retry may create another session | proven when database is present; skipped otherwise |
| source-invitation replacement | integration: reissue replaces invitation and keeps existing session | proven when database is present; skipped otherwise |
| assignment revocation | integration: assignment revocation denies exchange and session | proven when database is present; skipped otherwise |
| completed no-restart | integration: completed assignment still allows session and does not start execution | proven when database is present; skipped otherwise |
| exact expiry | integration: exact expiry refuses candidate session | proven when database is present; skipped otherwise |
| race/rollback/restart | repository sqlmock/T11 plus HTTP restart pattern in administrative suite; candidate HTTP uses the same PostgreSQL store | partial: candidate HTTP restart follows the same database URL; rollback is assignment-lock transactional |
| active-run cancellation | Interview consumer not present | skipped: Access does not fake admission/cancellation facts |

## Skips

- `TEST_DATABASE_URL` unset: integration package skipped.
- active-run cancellation / session-validity admission against a live Interview runtime: skipped because the Interview consumer not present in this repository.
