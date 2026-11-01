# T10 contract record — CORE-IAM-010

Status: **approved; T10 Done**. Prepared and recorded on 2026-11-01.

## Approval

- Confirming actor: the user in this conversation; no separate external sign-off is claimed.
- Date: 2026-11-01.
- Exact confirmation: exclusive T10–T12 execution with a Done checklist for T10 (`CORE-IAM-010`).
- Scope: candidate sections F–I. Product choices stay inside approved non-goals (no fake assignment table, no candidate passwords, no mailer, no email upsert). Q5 retention is unchanged. T11 still waits for T20 and the Interview-owned assignment migration before applying invitation FKs.
- Canonical accepted text is now in [technical-design.md, sections 6.6–6.9](technical-design.md#66-candidate-contact-and-matching-approved-f).

| Approved section | Canonical location |
|---|---|
| F — Candidate contact and matching | [§6.6](technical-design.md#66-candidate-contact-and-matching-approved-f) |
| G — Invitation exchange reuse and recovery | [§6.7](technical-design.md#67-invitation-exchange-reuse-and-recovery-approved-g) |
| H — Candidate operation schemas | [§6.8](technical-design.md#68-candidate-operation-schemas-approved-h) |
| I — Assignment collaboration and locking | [§6.9](technical-design.md#69-assignment-collaboration-and-locking-approved-i) |

## Design assessment and downstream obligations

Retained with rationale: Candidate owns reusable identity without login credentials; Invitation and CandidateSession remain assignment-scoped; Interview owns `interview_assignments`, consent, execution and cancellation. Application predicates on `AssignmentFacts` encode access eligibility; Access never creates an assignment table. Five-table properties are unchanged: no `consumed_at`, candidate email remains nonunique.

T12 implements domain/application operations against these ports. T11 writes candidate SQL independently of assignment FKs, then invitation/session SQL only after Interview's assignment migration exists. T13/T14 remain responsible for HTTP, cookies and real Interview consumers.

sqlmock applies to T11 SQL adapter tests; Mockery applies to T12 port tests.
