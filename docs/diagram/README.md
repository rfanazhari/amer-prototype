# Diagrams

PlantUML sources. Sequence diagrams are in `sequence/`.

## Sequence diagrams

One per flow, in [`sequence/`](sequence/). Messages follow the API contracts in `docs/api/` and the flow sections of each feature's technical design. Parts that are not built are drawn inside a `group` titled NOT DELIVERED, IN PROGRESS or NOT MOUNTED.

| # | File | Flow | Delivery |
| --- | --- | --- | --- |
| 01 | [01-task-lifecycle.puml](sequence/01-task-lifecycle.puml) | Create, start, complete, completion event, overdue view | Delivered |
| 02 | [02-administrative-login-hr-provisioning.puml](sequence/02-administrative-login-hr-provisioning.puml) | Bootstrap, login, add HR, logout | Delivered |
| 03 | [03-configure-publish-assign.puml](sequence/03-configure-publish-assign.puml) | Instance, draft, validation, idempotent publish, assignment | Delivered |
| 04 | [04-invitation-exchange.puml](sequence/04-invitation-exchange.puml) | Issue invitation, landing, exchange, session reuse | Delivered |
| 05 | [05-candidate-start-interview.puml](sequence/05-candidate-start-interview.puml) | Consent, start (commit-or-find), polling | Delivered |
| 06 | [06-takeover-cancel-revocation.puml](sequence/06-takeover-cancel-revocation.puml) | Tab takeover, early cancel, HR revocation | Delivered |
| 07 | [07-voice-session-and-finish.puml](sequence/07-voice-session-and-finish.puml) | Session attempt, ready, finish with Selesai | Delivered; model-ended finish in progress |
| 08 | [08-stage-handoff-deadline-reconnect.puml](sequence/08-stage-handoff-deadline-reconnect.puml) | Stage handoff, 15-minute deadline, transcript, reconnect | Partial |
| 09 | [09-report-generation.puml](sequence/09-report-generation.puml) | Report admission, outbox dispatch, worker, HR routes | Groundwork only |

## Render

```bash
plantuml -tsvg docs/diagram/sequence/*.puml
```

The prototype voice flow (`live-conversation`) has no diagram: its routes were removed from the process.

## Change history

| Version | Date | Author | Change |
| --- | --- | --- | --- |
| 1.2 | 2026-11-03 | rfanazhari | Remove the eight use case diagrams; sequence diagrams are the deliverable. |
| 1.1 | 2026-11-03 | rfanazhari | Add nine sequence diagrams in `sequence/`. |
| 1.0 | 2026-11-03 | rfanazhari | Initial eight use case diagrams, based on baseline v1.0. |
