# T9 verification — CORE-IRT-009

Only T9 is implemented: stage instructions without the salary ceiling, in `../../../../../application/usecase/interviewusecase`.

## Design mapping and assessment

- `Workflow.stageInstructions` (`instructions.go`) renders the context for the committed snapshot's current stage:
  question text, competency topics, role name, remaining follow-ups and `expectations_only`. It never renders the
  ceiling reference, any amount or a verdict request; the wording asks the model only to record what is said.
- `createSession` reads the committed progress (`Store.Find`) and supplies the instructions through `SupplyContext`
  right after `Attach`, for activation, next-stage opening and reconnect. On reconnect the recovery context follows
  as a second `SupplyContext` call. A failing lookup stops before any provider session is created.
- Role name comes from a new optional `Workflow.References` (`AgentReferenceReader`), resolving the pinned
  job-description version exactly. No job description pinned means no role line. Missing reference is `ErrConflict`,
  identity mismatch is `ErrReferenceIdentity`, a pinned reference with no reader is `ErrInvalid`.
- `Progress.RemainingFollowUps` (domain) owns the cap arithmetic; `FollowUp` now shares `usedFollowUps`.
- Ports unchanged, so no Mockery regeneration; no SQL, so sqlmock does not apply.

Assessment: **retained**. The cap stays a domain rule, rendering stays an application concern, and the reader is the
existing port. The existing reconnect test changed because reconnect now supplies two contexts (instructions, then recovery).

## Limits

- `Create.Instructions` is still empty; instructions travel only through `SupplyContext`, as the DoD states. The
  sideband frame shape is unverified against the live provider (see T6).
- Wording is not evaluated against a live model; tests prove inclusion and exclusion only.
- `References` is not wired in a composition root yet; the operations remain unmounted until T8.

## RED checkpoint

Cwd: `/Users/rfanazhari/Projects/go-workspace/prototype`.

```sh
go test ./domain/interview/... ./application/usecase/interviewusecase/...
```

Observed failure: `RemainingFollowUps` stub returned 0, and no instruction context was supplied or role lookup made.
RED SHA `cfb9762d96e5d8ac7e59f1ada2e64b0a4898de5f`, subject
`test: [CORE-IRT-009][rfanazhari] RED cover stage instructions without salary ceiling`.

## GREEN checkpoint

Same command passes. GREEN SHA `46ea5682e0dfc193b53a4f2b6a483dc9b868ff61`, subject
`feat: [CORE-IRT-009][rfanazhari] GREEN build stage instructions without salary ceiling`.

Final-state checks: `make test` passed, `make test.all` passed (database suites skipped, no `TEST_DATABASE_URL`),
`make lint` 0 issues, `make test.architecture` passed, `make test.security` 0 reachable vulnerabilities.
