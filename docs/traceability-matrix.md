# Traceability Matrix

Date: 2026-11-03
Branch at time of writing: `feat/irp`

Traces each acceptance criterion and invariant of the platform design to the implementing task, the recorded evidence, and the remaining gap.

| Document | Role |
| --- | --- |
| [`design/core.md`](design/core.md) | What is intended: goals, concepts, acceptance gate. |
| [`services/baseline.md`](services/baseline.md) | What is delivered, per feature. |
| This document | Links the two: criterion → task → evidence → gap. |

Criteria text is from the [scope freeze §7](design/core-ai/shared_ai_agent_platform_consolidated_architecture_mvp_scope_freeze_v0.1.md) and the invariants from its §5. This document paraphrases them.

## 1. How to read this

**Implementation** (what the task plans say):

| Value | Meaning |
| --- | --- |
| Implemented | The owning tasks are Done and cover the whole criterion. |
| Partial | Some of the criterion is Done; the rest is blocked, in progress, or missing. |
| Not implemented | No Done task covers it. |

**Evidence** is what the task plans record, not something re-run for this document. "Recorded" names the task. "None" means no evidence was found.

**Acceptance is a separate question.** The technical design (§13) states that all planned evidence mappings are not yet established, and that skipped database tests are not database verification. No row below is an accepted pass of the MVP gate. "Implemented" means the capability exists with recorded task-level evidence, not that the criterion was formally signed off.

Task notation: `ACP` Agent Configuration and Publication, `IAM` Identity and Access, `EXL` Execution Lifecycle, `CIN` Candidate Interview, `IRT` Interview Runtime, `IRP` Interview Reporting, `IRTREC` [Interview Recording and Transcription](services/core-ai/features/interview-recording-transcription/task-plan.md) (planned). IRTREC T1–T11 map to repository planning tickets `CORE-IRTREC-001`–`CORE-IRTREC-011`; no external tracker tickets have been created. `T11` after a prefix means task 11 of that feature, for example `ACP T11` is `CORE-ACP-011`.

## 2. Summary

| Group | Criteria | Implemented | Partial | Not implemented |
| --- | --- | --- | --- | --- |
| Shared platform | P-01 – P-12 | 5 | 7 | 0 |
| HR Interviewer | H-01 – H-12 | 4 | 7 | 1 |
| Document Analyzer | D-01 – D-08 | 1 | 0 | 7 |
| Invariants | INV-01 – INV-09 | 8 | 1 | 0 |
| Negative-path set | 8 scenarios | 6 | 1 | 1 |

The largest gaps are: the whole Document Analyzer run, transcript ingestion and reconnect in Interview Runtime, report generation, and the missing HR/admin UI for the configuration journey.

## 3. Shared platform (P-01 – P-12)

| ID | Criterion | Capability / rules | Tasks | Recorded evidence | Implementation | Gap |
| --- | --- | --- | --- | --- | --- | --- |
| P-01 | Both templates selectable | ACP BR-CORE-001 | ACP T1, T5, T8 | ACP T8, T9 | Partial | `GET /admin/agent-templates` lists both. No UI, so "non-developer selects" is not met. |
| P-02 | Create and configure both instances without source edits | ACP BR-CORE-001–003 | ACP T6, T7, T8, T14 | ACP T9 backend flow | Partial | Backend flow done for both. No admin UI for a non-developer. |
| P-03 | Invalid draft blocked with intelligible reason | ACP BR-CORE-006–008 | ACP T5, T6 | ACP T9 | Implemented | |
| P-04 | Published revision immutable, bound to exact validated draft | ACP BR-CORE-008–010 | ACP T3, T7 | ACP T7 PostgreSQL publication proof | Implemented | |
| P-05 | Execution configuration unchanged after later edit or publish | ACP BR-CORE-013; EXL BR-CORE-058 | ACP T11; EXL T4, T5 | ACP T11 with real assignment and execution consumers | Implemented | |
| P-06 | Both agents start through common authorized admission | EXL BR-CORE-050–052, 059 | EXL T5 (interview start, shared `Admit`); EXL T8 fixture | EXL T8 | Partial | Interview path real. `Admit` exercised only by a kernel fixture; no Document caller (EXL T11 Todo). |
| P-07 | Retry of the same initiation finds one execution | EXL BR-CORE-052, 053; CIN BR-CORE-041 | EXL T5, T8; CIN T11 | EXL T8 idempotency on PostgreSQL; CIN T11 eight concurrent starts | Implemented | Interview only. Document initiation is absent. |
| P-08 | Valid transitions; terminal cannot reopen | EXL BR-CORE-054 | EXL T1, T6 | EXL T8 | Implemented | |
| P-09 | Operation attempts, outcomes, failures traceable | EXL BR-CORE-057; IRT | EXL T6; IRT T18 | EXL T8; IRT T18 durable attempt authority | Partial | Interview only. No Document operations. No HR-facing view of attempts. |
| P-10 | No `COMPLETED` without committed required output | EXL BR-CORE-056; IRT BR-CORE-070 | EXL T6; IRT T11, T21 | IRT T11 (mocked lifecycle), T21 application part | Partial | Required output today is stage handoffs. The required transcript and coverage record do not exist (IRT T13/T14). No Document output. |
| P-11 | User sees identity, state, progress, output availability | CIN BR-CORE-043, 046 | CIN T5, T9, T12 | CIN T13 | Partial | Candidate poll exposes candidate-safe state only. No HR monitoring view, no output-availability view. |
| P-12 | Kernel needs no HR-stage or invoice logic; runnable Document uses it | EXL BR-CORE-059 | EXL T9 (architecture gate); EXL T11 | EXL T9 gates | Partial | Architecture half done. Runnable Document half is EXL T11, Todo, with no Document capability design. |

## 4. HR Interviewer (H-01 – H-12)

| ID | Criterion | Capability / rules | Tasks | Recorded evidence | Implementation | Gap |
| --- | --- | --- | --- | --- | --- | --- |
| H-01 | Live voice through the reused WebRTC path | IRT BR-CORE-066 | IRT T6, T8, T17, T19 | IRT T17: one live session through `session-attempts` (201, SDP applied, sideband attached, readiness 200, reply confirmed by owner) | Implemented | One sanitized live session. Not a repeated or full-interview run. |
| H-02 | Required questions asked or omissions recorded; at least one relevant follow-up | IRT BR-CORE-061, 062 | IRT T1, T20 | IRT T1 domain rules; T20 instructions | Partial | Coverage rules exist in the domain. No transcript-driven coverage update (T14 blocked). Follow-up behavior is prompt-level only (todo-gap G2). |
| H-03 | Stage advance rejected when workflow authorization is invalid | IRT BR-CORE-064 | IRT T1, T3, T10 | IRT T3, T10, T18 (real-adapter rollback cases) | Implemented | Nothing drives proposals from a live session yet (todo-gap G7). |
| H-04 | Each activated stage uses a separate model session | IRT BR-CORE-066 | IRT T10 | IRT T10, T18 | Partial | Mechanism exists. No live stage progression; per-stage pause of about 2.4 s noted (G10). |
| H-05 | Next-stage activation cannot precede committed handoff | IRT BR-CORE-065 | IRT T10 | IRT T10 | Implemented | |
| H-06 | Backend keeps authoritative normalized transcript and progress references | IRT BR-CORE-067 | IRT T5 (progress); IRTREC T3–T9, T11 | IRT T5, T18; no IRTREC execution evidence | Partial | Progress references stored. Recording-based async transcript ingest is planned in IRTREC T1–T11, replacing G14's primary source; provider alignment/storage-policy proofs remain open. Existing IRT task statuses are unchanged. |
| H-07 | Coverage and report statements linked to transcript evidence | IRT BR-CORE-068; IRP BR-CORE-083 | IRP T1, T14; IRTREC T5, T7, T10, T11 | IRP T1 statement rules; no IRTREC execution evidence | Partial | Finalized-evidence population and contract handoff are planned in IRTREC; IRP T14 still owns the production report read adapter. |
| H-08 | One reconnect resumes from checkpoint without duplicate progress | IRT BR-CORE-069 | IRT T15 | none | Not implemented | T15 Todo, blocked on Q1. Fencing of stale attempts exists (see INV-06). |
| H-09 | Completes only after required transcript and progress finalization | IRT BR-CORE-070 | IRT T11, T21 | IRT T11 | Partial | Completion needs sealed handoffs. Owner-selected IRTREC finalizes transcript asynchronously AFTER COMPLETED; this does not satisfy the original pre-completion wording. IRTREC T2 must reconcile BR-CORE-070/H-09; T11 records proof of the revised contract. Candidate Selesai exists; model trigger remains IRT T21 work. |
| H-10 | Report lifecycle is separate; failed report does not fail the interview | IRP BR-CORE-080, 081 | IRP T1, T4, T5 | IRP T5 admission atomicity and failure isolation on PostgreSQL | Implemented | Isolation is proved. No report is generated (writer client and evidence are missing). |
| H-11 | No autonomous hire or reject decision | IRP BR-CORE-084 | IRP T1 | IRP T1 | Partial | Enforced by report domain rules only. No report output exists to inspect. |
| H-12 | Salary ceiling not disclosed in normal or adversarial scenario | IRT BR-CORE-071 | IRT T9, T20 | IRT T9: stage instructions built without the ceiling id | Partial | Construction-level proof only. No adversarial conversation scenario recorded. |

## 5. Document Analyzer (D-01 – D-08)

No Document Analyzer execution capability exists. Only configuration and publication (ACP) is available.

| ID | Criterion | Tasks | Recorded evidence | Implementation | Gap |
| --- | --- | --- | --- | --- | --- |
| D-01 | User selects required fields from the predefined set | ACP T5, T6 | ACP T9 | Implemented | Through API only; no UI. |
| D-02 | Published configuration drives one invoice execution | EXL T11 | none | Not implemented | No admission, revision resolution, or artifact handling. |
| D-03 | Result represents the five invoice fields | none | none | Not implemented | No extraction. |
| D-04 | Outcome distinguishes valid, missing, invalid | none | none | Not implemented | |
| D-05 | Missing or ambiguous fields explicit, not fabricated | none | none | Not implemented | |
| D-06 | Committed result precedes successful completion | none | none | Not implemented | |
| D-07 | One controlled failure follows bounded retry and fail policy | none | none | Not implemented | Needs RabbitMQ job path for Document. |
| D-08 | Shared lifecycle, binding and tracking demonstrably reused | EXL T11 | EXL T8 kernel fixture (not sufficient) | Not implemented | The T8 fixture does not close this. |

## 6. Invariants (INV-01 – INV-09)

| ID | Invariant | Tasks | Recorded evidence | Implementation | Note |
| --- | --- | --- | --- | --- | --- |
| INV-01 | Stable execution identity across reconnects and retries | EXL T1, T5 | EXL T8; CIN T11 | Implemented | Reconnect path itself not built. |
| INV-02 | Valid transitions only; terminal never reopens | EXL T1, T6 | EXL T8 | Implemented | |
| INV-03 | One authoritative state per version | EXL T4, T6 | EXL T8 locking | Implemented | |
| INV-04 | Repeated logical commands do not duplicate outcomes | ACP T7; EXL T5 | ACP T7; EXL T8 | Implemented | |
| INV-05 | Recovery only from verifiable committed progress | IRT T5, T18 | IRT T18: authority survives reload | Partial | Stored state is restorable. End-to-end recovery is T15, Todo. |
| INV-06 | Superseded attempts fenced | EXL T6; IRT T18 | EXL T8; IRT T18 | Implemented | |
| INV-07 | Terminal outcome refers to committed required output | EXL T6; IRT T11 | IRT T11 | Implemented | See P-10 on what the output currently is. |
| INV-08 | State changes traceable to operation and actor | EXL T6 | EXL T8 | Implemented | |
| INV-09 | Effective configuration binding stable | ACP T11; EXL T4 | ACP T11 | Implemented | |

## 7. Negative-path set

| # | Scenario | Result | Tasks | Evidence | Gap |
| --- | --- | --- | --- | --- | --- |
| 1 | Invalid publish rejected, no revision created | Implemented | ACP T5, T7 | ACP T9 | |
| 2 | Publish during a running execution leaves binding intact | Implemented | ACP T11 | ACP T11 | |
| 3 | Duplicate initiation resolves the existing execution | Implemented | EXL T5; CIN T11 | EXL T8; CIN T11 | Interview only. |
| 4 | Activation fails before RUNNING: bounded retry or explicit FAILED | Implemented | EXL T6; IRT T16 | EXL T8 | Activation attempt budget for the next stage is IRT T16, Todo. |
| 5 | HR transition blocked when handoff is absent | Implemented | IRT T10 | IRT T10, T18 | |
| 6 | Reconnect uses checkpoint; superseded session cannot overwrite | Partial | IRT T15; IRT T18 | IRT T18 fencing | Fencing done. Reconnect not built. |
| 7 | Document operation fails, retries within bounds, then commits or fails honestly | Not implemented | EXL T11 | none | |
| 8 | Post-terminal mutation rejected; report failure leaves interview intact | Implemented | EXL T6; IRP T5 | EXL T8; IRP T5 | |

## 8. Business rules to capabilities

Rule text is in the capability documents and summarised in [`services/baseline.md`](services/baseline.md). This table shows which rules are not fully delivered, because the owning tasks are open.

| Capability | Rules | Owning tasks | Rules not fully delivered | Blocking tasks |
| --- | --- | --- | --- | --- |
| Agent Configuration and Publication | BR-CORE-001–014 | ACP T1–T14, all Done | None at API level. | No admin UI. |
| Identity and Access | BR-CORE-020–039 | IAM T1–T16, T18–T23 Done; T17 in progress | BR-CORE-055 (expiry mid-interview cancels): revocation is delivered, session-expiry driver is not. | IAM Q7. IAM T17 final GREEN blocked by Docker regressions. |
| Execution Lifecycle | BR-CORE-050–059 | EXL T1–T10 Done | BR-CORE-059 proof for a second agent. | EXL T11 |
| Candidate Interview | BR-CORE-040–047 | CIN T1–T13 Done | None. | |
| Interview Runtime | BR-CORE-060–071 | IRT T1–T11, T17–T20 Done | BR-CORE-061–063 (coverage from speech), 067, 068 (transcript and evidence), 069 (reconnect), 070 (finalized record). | IRT T12–T15 (Q1), T16, T21; IRTREC T1–T11 plans the new recording source without closing those tasks |
| Interview Reporting | BR-CORE-080–086 | IRP T1–T12 Done | BR-CORE-082, 083 (report content and citations), 085 (regeneration available to HR). | IRP T13–T15; IRTREC final-evidence prerequisites |

## 9. Deviations from the design

| Item | Design says | Recorded state |
| --- | --- | --- |
| Schema management | AutoMigrate for additive development schema (tech design §5) | Manual versioned SQL migrations are the approved policy. Startup requires clean version 13 per the root README. IRP T10 records migration 13 delivered. IRTREC proposes the next migration but has created none. |
| Provider | OpenAI Realtime (scope freeze) | GPT-Live (TD-11). |
| Reporting dispatch | Document and report jobs through RabbitMQ with outbox | IRP T9–T12 are Done per current plan, with real RabbitMQ/PostgreSQL proof recorded in T11/T12. IRTREC plans its own outbox/consumer; its topology and real-broker proof remain open. |
| Stage structure | Opening, role exploration, expectations and closing; at most six mandatory questions | Runtime plan uses four stages with `closing` last. Not reconciled against the design table. |
| Stage progression | Model proposes, workflow validates | Today a stage is complete when the model states it; transcription is deferred to the end of the interview (owner decision, todo-gap G9). |
| Transcript source and timing | Earlier G14 selected sideband transcript events; H-09 requires final transcript before completion | Owner now selects backend-only stored-recording download and transcription asynchronously after COMPLETED (IRTREC D1–D3). Source direction superseded; business wording reconciliation is IRTREC T2, not an acceptance pass. |
| Backend and UI split | Backend and candidate UI now, admin UI later | Matches. Consequence: P-01, P-02, P-11 stay Partial until an HR/admin surface exists. |

### Planned recording/transcription coverage

The owner selected recording as the primary transcript source and backend-only asynchronous processing after COMPLETED. This adds no acceptance pass or implementation count to §2.

| Requirement / gap | Planned tasks | Existing evidence | Remaining gate |
| --- | --- | --- | --- |
| Download only after completion and provider readiness; candidate response independent | IRTREC T1, T6, T9, T11 | None for new pipeline | Storage permission, close/readiness proof and shared-transaction admission |
| All sessions, input/output speaker identity and chronological final transcript (H-06) | IRTREC T1, T3–T7, T11 | Canonical entry/session tables exist; no audio import proof | Alignment/chunk schema and real multi-session WAV/transcription proof |
| Stable evidence for reporting (H-07) | IRTREC T5, T7, T10; IRP T14 | Report citation rules only | Atomic complete revision and body/read handoff; report adapter still pending |
| Restart recovery, duplicates and failure isolation (INV-02, INV-04, negative-path 8) | IRTREC T5, T8, T9, T11 | Existing lifecycle/report tests do not prove transcription | Real database/broker crash and token proofs |
| Recording consent, retention and no public audio download | IRTREC T2, T6, T9 | Owner scope decision only | Approved notice/persistence/cleanup policy before enablement |
| Post-completion transcript versus original H-09/P-10 wording | IRTREC T2, T11 | Owner asynchronous direction; no revised acceptance proof | Reconcile business authority; never reopen completed execution to import transcript |

## 10. Limits of this matrix

- Initial matrix was built from docs only. The IRTREC planning update inspected affected source/schema but ran no application tests or live provider calls. New tasks remain planned; prior evidence has not been re-run.
- Status relies on task plans. Where a plan and a README disagree, the plan is used (see baseline §15).
- Verification limits repeated from the sources: database suites run only with an authorized `TEST_DATABASE_URL`; several tasks record the full database suite and `make test.docker` as not run; some browser checks used stand-in endpoints.
- The scope freeze requires retained evidence per criterion (scenario, fixture, expected result, evidence ID, pass, fail or not-run). Those records are not yet kept per criterion; this matrix points to the closest task evidence instead.

## 11. Maintenance

Update a row when its task changes status or new evidence is recorded. Keep the summary counts in section 2 in step with the tables. A criterion should move to an accepted pass only when its evidence record exists and the pass has been reviewed.

## 12. Change history

Newest first. Add a row for every update, and change the date at the top of the document.

| Version | Date | Author | Change |
| --- | --- | --- | --- |
| 1.2 | 2026-11-03 | Codex | Link IRTREC T1–T11 to CORE-IRTREC-001–011 planning ticket identities; no scope or delivery-status changes. |
| 1.1 | 2026-11-03 | Codex | Trace planned IRTREC tasks to transcript/evidence/recovery gaps; record asynchronous H-09 reconciliation and G14 replacement. Refresh IRP T10–T12 status from current plan. No new acceptance pass or changed summary counts. |
| 1.0 | 2026-11-03 | rfanazhari | Initial matrix: P-01–12, H-01–12, D-01–08, INV-01–09, negative-path set, rule-to-capability gaps, deviations. Based on baseline v1.0 and core v1.0. |
