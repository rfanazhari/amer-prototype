# Delivery Baseline

Baseline date: 2026-11-04
Branch at baseline: `feat/irtrec`
Scope: every feature directory under `docs/services/`
Project definition (what is built, concepts, goals): [docs/design/core.md](../design/core.md)

## 1. Purpose and how to use this document

This is the reference record of **what has been delivered, what is in progress, and what is not yet done**. It is meant for three uses:

1. Compare against the business design and the technical design of the project.
2. Serve as input for building the use case diagrams (see [section 14](#14-use-case-catalogue-for-diagramming)).
3. Identify documents that disagree with each other (see [section 15](#15-documentation-inconsistencies-found-while-building-this-baseline)).

Rules for reading it:

- A feature is "Done" only when its task plan marks the task `[x]`. Anything labelled "planned", "blocked" or "not mounted" is **not** delivered behavior.
- Business rule IDs (`BR-…`) are quoted from the capability documents. A rule being listed does not mean the code satisfies it. Rule coverage is stated through the task status of the capability that owns it.
- Verification claims are those recorded in the task plans. This baseline did not re-run any test or command.

## 2. Sources and which one wins

| Priority | Source | Used for |
| --- | --- | --- |
| 1 | `features/<feature>/task-plan.md` | Task status (`[ ]` Todo, `[~]` In Progress, `[x]` Done). This is the status authority. |
| 2 | `business/capabilities/<capability>.md` | Actors, triggers, business rules (`BR-…`). |
| 3 | `docs/api/openapi.yaml`, `docs/api/*.md`, `migrations/pgsql/` | Route and schema inventory, cross-check only. |
| 4 | `README.md` files under `docs/services/` | Navigation. **Not** the status authority; they lag behind the task plans (section 15). |

Where a capability document front matter says `status: planned`, it describes the document's own review state, not delivery. Delivery status comes from the task plan.

## 3. Summary

| # | Feature | Service | Tasks done | Tasks open | Delivery status |
| --- | --- | --- | --- | --- | --- |
| F1 | Task Tracking | amer-prototype | n/a (no tripack) | Acceptance gate open | **Partial**: implemented, acceptance gate open |
| F2 | Identity and Access | core-ai | T1–T14, T15, T16, T18–T23 | T17 in progress | **Delivered**, one in-progress task |
| F3 | Agent Configuration and Publication | core-ai | T1–T14 | none | **Delivered** |
| F4 | Execution Lifecycle | core-ai | T1–T10 | T11 (needs Document Analyzer) | **Delivered** for interview; second-agent proof open |
| F5 | Candidate Interview | core-ai | T1–T13 | T14–T15 Todo (added 2026-11-04, per-stage voice session) | **Delivered**; T14–T15 extension planned |
| F6 | Interview Runtime | core-ai | T1–T11, T17–T20, T22 | T21 in progress; T12–T15 blocked; T16 todo; T23–T25 todo (T22–T25 added 2026-11-04) | **Partial** |
| F7 | Interview Reporting | core-ai | T1–T12 | T13–T15 blocked/deferred | **Partial**, writer/evidence missing and HR routes not mounted |
| F8 | Voice Session Lifecycle Prototype | live-conversation | T1–T7 | none | **Done as a prototype, then superseded**: its routes are removed from the process |
| F9 | [Interview Recording and Transcription](core-ai/features/interview-recording-transcription/task-plan.md) | core-ai | none | IRTREC T1 in progress, T2–T12 Todo (T12 live-event spike added 2026-11-04); decision gates in design §9 | **Planned**, no download/transcription pipeline delivered |

Database schema: manual versioned migrations `000001`–`000013` exist in `migrations/pgsql/`. The root `README.md` states startup requires clean version 13. The Interview Reporting task plan records migration `000013_interview_report_retry_window` as delivered with T10. Recording/transcription proposes the next migration (currently 14), to be rechecked at implementation; that migration does not exist yet.

## 4. Actors

| Actor | Description | Authentication | Features |
| --- | --- | --- | --- |
| General user | Uses Task Tracking. No authentication or ownership in scope. | none | F1 |
| Deployment operator | Bootstraps the first ADMIN from environment credentials; runs migration and retention commands. | environment / CLI | F2 |
| ADMIN | Provisions HR accounts. Has no HR business authority (BR-CORE-022). | Admin cookie + CSRF | F2 |
| HR | Manages shared agent configuration, candidates, assignments, invitations, revocation, reports. | Admin cookie + CSRF | F2, F3, F7 |
| Candidate | Invited person; assignment-scoped session, no permanent account. | Candidate cookie | F2, F5, F6 |
| Interview runtime (system) | Activates, progresses, completes and fails the execution. | internal | F4, F6 |
| Voice model / provider (external) | Real-time voice provider (GPT-Live). Proposes stage moves, speaks. | server-held key | F6, F8 |
| Reporting worker (system) | Produces the written report from finalized evidence. | internal | F7 |
| Transcription worker (system, planned) | Downloads completed interview recordings and finalizes transcript evidence asynchronously. | backend provider credentials | F9 |
| Test candidate | Prototype-only user with synthetic data. | none | F8 |

## 5. F1 — Task Tracking (amer-prototype)

Sources: `amer-prototype/business/README.md`, `amer-prototype/business/capabilities/task-tracking.md`.

**Purpose.** Record work, follow its progress, identify overdue tasks. Reference feature of the repository.

**Status.** Business document status `partial` (updated 2026-10-31). Code and automated evidence exist; the full service acceptance gate remains open. Historical RED/GREEN evidence is incomplete.

**Lifecycle.** `Todo → In Progress → Done`. Done is final. Overdue is derived, not a state.

**Business rules**

| ID | Rule |
| --- | --- |
| BR-AMER-001 | Title is trimmed and has 1–100 Unicode code points. |
| BR-AMER-002 | New tasks begin as Todo. |
| BR-AMER-003 | Due date is optional. When supplied or changed it must be ≥ current time. Clearing is allowed. |
| BR-AMER-004 | Only Todo can start. |
| BR-AMER-005 | Only In Progress can complete. |
| BR-AMER-006 | Done is final: start, complete and reschedule are rejected. |
| BR-AMER-007 | Overdue = now strictly after due date and not Done. Equality or no date is never overdue. |
| BR-AMER-008 | The system owns creation and completion times. |
| BR-AMER-009 | Completion produces an internal event, accepted together with persistence; refusal preserves the old task; repeated completion produces no extra event. |

**Delivered interface**

| Method | Path | Use case |
| --- | --- | --- |
| POST | `/tasks` | Create task |
| GET | `/tasks/{id}` | View task |
| POST | `/tasks/{id}/start` | Start task |
| POST | `/tasks/{id}/complete` | Complete task |
| PATCH | `/tasks/{id}/due-date` | Change or clear due date |
| GET | `/health` | Health (HTTP readiness only) |

**Known limits.** The completion event queue is process-local; events can be lost on unexpected termination; subscriber failure is logged without retry. Redis and RabbitMQ are optional local services and not Task dependencies.

## 6. F2 — Identity and Access (core-ai)

Sources: `business/capabilities/identity-access.md`, `features/identity-access/task-plan.md`.

**Purpose.** ADMIN provisions HR. An invited candidate gets access to one assignment without a permanent account; one candidate identity can hold many assignments.

**Business rules**: `BR-CORE-020` to `BR-CORE-039`.

| ID | Rule |
| --- | --- |
| BR-CORE-020 | Administrative access is provisioned (bootstrap from environment). |
| BR-CORE-021 | ADMIN provisions HR accounts. |
| BR-CORE-022 | ADMIN does not inherit HR business authority. |
| BR-CORE-023 | HR colleagues share company configuration. |
| BR-CORE-024 | Administrative sessions expire absolutely. |
| BR-CORE-025 | Logout revokes the current session. |
| BR-CORE-026 | Current account eligibility governs administrative access. |
| BR-CORE-027 | Administrative subject identity survives login changes. |
| BR-CORE-028 | Publication replay still requires current authority. |
| BR-CORE-029 | Candidate access grants no administrative authority. |
| BR-CORE-030 | A candidate identity can have multiple assignments. |
| BR-CORE-031 | An invitation grants access to one assignment. |
| BR-CORE-032 | An invitation must be currently valid at exchange. |
| BR-CORE-033 | Invitation verification establishes a distinct access session. |
| BR-CORE-034 | Obtaining access does not start the interview. |
| BR-CORE-035 | Reloading does not create another interview. |
| BR-CORE-036 | Candidate sessions expire absolutely. |
| BR-CORE-037 | Reissuing an invitation preserves the interview identity. |
| BR-CORE-038 | Assignment revocation removes candidate access. |
| BR-CORE-039 | A completed interview is not restarted by an access operation. |

**Task status** (from the task plan)

| Task | Subject | Status |
| --- | --- | --- |
| T1–T9 | Administrative access: contracts, domain, ports, models, primitives, bootstrap/HR provisioning/session operations, authority adapter, HTTP, evidence | Done |
| T10–T14 | Candidate identity, invitation, session persistence, operations, HTTP, real integration proof | Done |
| T15 | Administrative-session cleanup (rows > 7 days after expiry/revocation, batches ≤ 500, operator command `make access.prune-sessions`) | Done. No deletion for invitations, candidate sessions, candidates or accounts. |
| T16 | Legacy schema baseline and migration policy | Done |
| T17 | Versioned baseline SQL and explicit migration tooling | **In progress**: final GREEN blocked by failing full Docker regressions outside this task |
| T18–T20 | Replace AutoMigrate with explicit migrations; equivalence/adoption proof; cutover rehearsal | Done on disposable PostgreSQL and isolated stack |
| T21 | `GET /admin/candidates` listing only candidates created by the logged-in HR | Done |
| T22 | HR revokes assignment access and cancels a nonterminal execution | Done |
| T23 | A new administrative login replaces the account's earlier sessions | Done |

**Delivered interface**

| Method | Path | Actor | Use case |
| --- | --- | --- | --- |
| POST | `/admin/sessions` | ADMIN/HR | Log in |
| GET | `/admin/session` | ADMIN/HR | Read current session and CSRF token |
| DELETE | `/admin/sessions/current` | ADMIN/HR | Log out |
| POST | `/admin/hr-accounts` | ADMIN | Add HR account |
| POST | `/admin/candidates` | HR | Record candidate |
| GET | `/admin/candidates` | HR | List own candidates |
| GET | `/admin/candidates/{id}` | HR | View candidate |
| POST | `/admin/interview-assignments` | HR | Create assignment (see F3) |
| POST | `/admin/interview-assignments/{id}/invitations` | HR | Issue or replace invitation |
| POST | `/admin/interview-assignments/{id}/revocation` | HR | Revoke assignment access |
| (landing) `/candidate/access`, `/candidate/access/exchange`, `/candidate/session` | Candidate | Open invitation, exchange it, read/reuse candidate session |

First ADMIN is created at startup from environment-supplied credentials (not an HTTP route).

**Open / limits.** In-interview candidate-session expiry driving cancellation is open question Q7 and not implemented; candidate refusal with 401 is implemented, but 401 itself does not commit cancellation. Retention beyond administrative sessions has no policy. There is no admin UI.

## 7. F3 — Agent Configuration and Publication (core-ai)

Sources: `business/capabilities/agent-configuration-publication.md`, `features/agent-configuration-publication/task-plan.md`.

**Purpose.** HR configures reusable agents from predefined templates, validates, and publishes an immutable revision; an interview assignment pins the current revision.

**Templates.** HR Interviewer and Document Analyzer.

**Business rules**: `BR-CORE-001` to `BR-CORE-014`.

| ID | Rule |
| --- | --- |
| BR-CORE-001 | Instances use predefined, versioned templates. |
| BR-CORE-002 | Configuration stays within the template's supported choices. |
| BR-CORE-003 | Configuration management requires HR authority. |
| BR-CORE-004 | Draft replacement protects against stale edits (expected draft version). |
| BR-CORE-005 | Editing a draft leaves published configuration intact. |
| BR-CORE-006 | Validation evaluates one exact configuration deterministically. |
| BR-CORE-007 | Validation errors block publication with an intelligible reason. |
| BR-CORE-008 | Publication requires validation applicable to the exact current draft. |
| BR-CORE-009 | A published revision is immutable. |
| BR-CORE-010 | Successful publication establishes the active revision consistently. |
| BR-CORE-011 | Repeating the same logical publication returns its original outcome. |
| BR-CORE-012 | Reusing a publication identity for different intent is a conflict. |
| BR-CORE-013 | Later publication preserves existing bindings. |
| BR-CORE-014 | Publication does not promise immediate execution readiness. |

**Task status.** T1–T14 all Done: contracts, application boundaries, domain, SQL registration, both catalogs and validators, commands/queries, PostgreSQL publication transactions, secured HTTP, backend flow verification, template deployment compatibility, binding preservation, assignment-create contract and route, instance list.

**Delivered interface**

| Method | Path | Use case |
| --- | --- | --- |
| GET | `/admin/agent-templates` | List templates |
| POST | `/admin/agent-instances` | Create instance |
| GET | `/admin/agent-instances` | List instances |
| GET | `/admin/agent-instances/{id}` | View instance |
| PUT | `/admin/agent-instances/{id}/draft` | Replace draft |
| POST | `/admin/agent-instances/{id}/validations` | Validate draft |
| POST | `/admin/agent-instances/{id}/publications` | Publish revision (idempotent via `Idempotency-Key`) |
| GET | `/admin/agent-instances/{id}/revisions/{revision_id}` | View revision |
| POST | `/admin/interview-assignments` | Assign candidate to current HR Interviewer revision |

**Limits.** Document Analyzer is configurable and publishable only. Invoice admission, extraction, result storage and worker orchestration are **not implemented** (see F4 T11). No admin UI; the non-developer configuration journey is a final-MVP requirement not delivered here.

## 8. F4 — Execution Lifecycle (core-ai)

Sources: `business/capabilities/execution-lifecycle.md`, `features/execution-lifecycle/task-plan.md`.

**Purpose.** One admitted run per assignment with stable identity and an honest ending. Lifecycle: `accepted → running → terminal` (`COMPLETED`, `FAILED`, `CANCELLED`).

**Business rules**: `BR-CORE-050` to `BR-CORE-059`.

| ID | Rule |
| --- | --- |
| BR-CORE-050 | An assignment has one execution. |
| BR-CORE-051 | An accepted start exists before its work begins. |
| BR-CORE-052 | The same start resolves the same execution. |
| BR-CORE-053 | A different start intent conflicts. |
| BR-CORE-054 | Work progresses through accepted, running, one terminal outcome. |
| BR-CORE-055 | Revocation or expiry during the interview cancels it. |
| BR-CORE-056 | Completion waits for the required progress. |
| BR-CORE-057 | A superseded attempt cannot change newer progress (fencing). |
| BR-CORE-058 | Later publication does not rewrite a bound execution. |
| BR-CORE-059 | The shared lifecycle carries no agent-specific checklist. |

**Task status**

| Task | Subject | Status |
| --- | --- | --- |
| T1–T2 | Domain identity/binding/transitions; sibling operations and ports | Done |
| T3–T4 | Migration `000008`; GORM execution store | Done |
| T5–T7 | Interview start, shared admit, reads; activation, fencing, terminal commits; cancellation when access ended (`Cancel`, `CancelNonterminal`) | Done |
| T8–T9 | PostgreSQL proof of locking, rollback, idempotency; architecture/contract/lint gates | Done |
| T10 | HR revocation cancels a nonterminal execution | Done (delivered through Identity and Access T22) |
| T11 | Run a non-interview agent on this kernel | **Todo**. Requires a Document Analyzer capability that does not exist. |

**Interface.** No direct HTTP routes. The lifecycle is consumed by Candidate Interview, Interview Runtime and Identity and Access.

**Known limit.** In-interview session-expiry cancellation driver (Identity and Access Q7) is not implemented.

## 9. F5 — Candidate Interview (core-ai)

Sources: `business/capabilities/candidate-interview.md`, `features/candidate-interview/task-plan.md`.

**Purpose.** A candidate holding an assignment-scoped session reviews the notice, consents, enters the one live interview, follows progress, and leaves. This is the candidate screen; there is no HR/admin screen.

**Business rules**: `BR-CORE-040` to `BR-CORE-047`.

| ID | Rule |
| --- | --- |
| BR-CORE-040 | Consent precedes the start of the interview (notice `notice-261102`). |
| BR-CORE-041 | A start request enters the assignment's one interview. |
| BR-CORE-042 | The candidate session must cover the whole interview (15-minute hard deadline). |
| BR-CORE-043 | The candidate sees only this interview. |
| BR-CORE-044 | One tab controls the live interview; a second tab can take over. |
| BR-CORE-045 | The candidate can leave before the interview finishes (cancel). |
| BR-CORE-046 | Refreshing returns to the same interview. |
| BR-CORE-047 | The final screen does not wait for the HR report. |

**Task status.** T1–T13 all Done: notice, consent/coverage/controller domain, ports, migration `000009`, consent persistence and read, consent HTTP and page shell, start/takeover/cancel/poll orchestration, persisted controller takeover, HTTP for those, candidate screen, real lifecycle integration, candidate-safe progress with controller fencing (T12), and the combined PostgreSQL slice proof (T13).

**Delivered interface**

| Method | Path | Use case |
| --- | --- | --- |
| GET | `/candidate/interview` | View interview, notice, consent state |
| POST | `/candidate/consents` | Accept notice |
| POST | `/candidate/executions` | Start or rejoin interview |
| GET | `/candidate/executions/{id}` | Poll status / progress |
| POST | `/candidate/executions/{id}/takeover` | Take control from another tab |
| POST | `/candidate/executions/{id}/cancel` | Confirm early stop |
| POST | `/candidate/executions/{id}/complete` | Finish interview ("Selesai" control, from Interview Runtime T21) |
| POST | `/candidate/executions/{id}/session-attempts` | Request a voice session (from F6) |
| POST | `/candidate/executions/{id}/session-attempts/{attempt_id}/ready` | Report session readiness (from F6) |
| GET | `/candidate/interview/page` | Candidate page |

**Verification recorded.** `TestCandidateInterviewSlicePostgres` on disposable PostgreSQL 16 covers notice before start, one execution after lost start, coverage refusal at 14m59s vs acceptance at 15m30s, takeover with replaced-tab refusal, early cancel as `CANCELLED`/`candidate_early_stop`, refresh with one execution, and `COMPLETED` view with no report. Browser check used a stand-in session endpoint. The full database suite was not run for T11/T13.

**Planned extension (2026-11-04).** Owner decision D10 (Interview Runtime) gives each stage its own voice session. `CORE-CIN-014` (next-stage session request and transition state) and `CORE-CIN-015` (page opens a new connection per stage) are Todo and depend on Interview Runtime T23/T24 and on decisions D12 and D13 (accepted 2026-11-04). Delivered T1–T13 are unchanged. Until those tasks are the open scope, F5 stays Delivered; it would then be read as Partial.

## 10. F6 — Interview Runtime (core-ai)

Sources: `business/capabilities/interview-runtime.md`, `features/interview-runtime/task-plan.md`, `features/interview-runtime/todo-gap.md`.

**Purpose.** Conduct the live HR interview: ask published questions, record honest gaps, advance only on a committed handoff, recover one lost connection, finish only with the required record. Business document status is `partial`.

**Business rules**: `BR-CORE-060` to `BR-CORE-071`.

| ID | Rule |
| --- | --- |
| BR-CORE-060 | The live interview uses the revision it was bound to. |
| BR-CORE-061 | Required questions are asked or explicitly omitted. |
| BR-CORE-062 | A recorded answer is not a judgement of competence. |
| BR-CORE-063 | Lost audio is not recorded as an omission. |
| BR-CORE-064 | The workflow decides whether a stage may change. |
| BR-CORE-065 | The next stage starts only after the current handoff is stored. |
| BR-CORE-066 | Each active stage has its own voice session. |
| BR-CORE-067 | The backend transcript is the record the interview relies on. |
| BR-CORE-068 | Evidence is cited; a missing record is not an answer. |
| BR-CORE-069 | One lost connection resumes the same interview. |
| BR-CORE-070 | The interview finishes only with its required record. |
| BR-CORE-071 | The salary ceiling stays out of the conversation. |

**Task status**

| Task | Subject | Status |
| --- | --- | --- |
| T1–T3 | Progress rules; lifecycle/driver/progress ports; proposal and handoff orchestration | Done |
| T4–T5 | Migration `000010`; classified progress persistence | Done |
| T6 | GPT-Live driver transport | Done |
| T7 | Remove prototype voice routes from the process | Done |
| T8 | Session-attempt and readiness HTTP | Done |
| T9 | Stage instructions without the salary ceiling | Done |
| T10–T11 | Commit handoff and open next session under lifecycle lock; complete or fail at required record / hard deadline | Done |
| T17 | One live interview-path voice session recorded | Done |
| T18 | Production integration and mount (migration `000011_interview_runtime_integration`) | Done |
| T19 | Start the voice session from the candidate page | Done |
| T20 | Interviewer stage context and scope rules | Done |
| T21 | Let the model end the interview stage by stage | **In progress**. Application part and candidate "Selesai" finish control are done with evidence. Model-triggered ending (`complete_stage` tool) is blocked on a live spike and owner decision G9. |
| T12 | Record provider event-finality map | **Todo**, blocked on Q1 |
| T13 | Ingest canonical transcript from classified events | **Todo**, blocked on Q1 |
| T14 | Update coverage from finalized speech | **Todo**, blocked on Q1 |
| T15 | Prove one deliberate reconnect | **Todo**, blocked on Q1 |
| T16 | Enforce post-handoff activation budget (2 attempts × 30 s within 90 s per stage; decided 2026-11-03) | **Todo** |
| T22 | Regenerate the Interview Runtime tripack for per-stage sessions (D10, D11) | Done (documentation only, 2026-11-04) |
| T23–T25 | Seal without advance and open the next stage in a new session; driver waits for close confirmation (decision D12 accepted; the timeout value is not set); per-stage first-session instructions | **Todo** |

**What this means in practice.** A candidate can start a voice session from the page, and the interview can be ended by the candidate through "Selesai" (recorded `COMPLETED`). There is **no transcript ingest, no coverage update from speech, and no automatic stage progression** yet; coverage-based claims in BR-CORE-061 to BR-CORE-063, BR-CORE-067 and BR-CORE-068 are therefore not fully delivered. Open gaps G1–G14 in `todo-gap.md` include the hard-coded position (`Project Manager`), prompt-only scope rules, per-stage session pause (~2.4 s), unstored closing questions, and no implemented transcript source after session close. G14's earlier sideband-primary direction is superseded by the owner-approved F9 recording-source plan; sideband controls/errors remain separate work. F9 does not mark IRT T12–T15 Done.

**Per-stage sessions (owner decisions 2026-11-04).** D10: one provider session per stage (closes G10; replaces the same-session advance of `CompleteStage` for non-closing stages). D11: an interim model-stated stage seal, labelled in reports, until coverage exists; strict `Authorize` stays the target, and the trigger is a model function call over the sideband still pending the T21 spike. New tasks: T22 (regenerate the tripack) is Done as documentation only on 2026-11-04; T23 (seal without advance, open next stage in a new session), T24 (driver waits for close confirmation; decision D12 accepted, timeout value not set) and T25 (first-session instructions) are Todo. No code for them is written. Stage progression is still not enforced in a running interview.

## 11. F7 — Interview Reporting (core-ai)

Sources: `business/capabilities/interview-reporting.md`, `features/interview-reporting/task-plan.md`, `docs/api/interview-reporting.md`.

**Purpose.** HR gets a separate written account of a finished interview (observed, missing, uncertain, each cited to the record). It does not hire, reject or rank; failure to produce it does not change the interview. HR API only, no screen.

**Business rules**: `BR-CORE-080` to `BR-CORE-086`.

| ID | Rule |
| --- | --- |
| BR-CORE-080 | A report is a separate outcome of a completed interview. |
| BR-CORE-081 | A failed report leaves the interview completed. |
| BR-CORE-082 | The report states what was observed, missing, and uncertain. |
| BR-CORE-083 | A report statement cites the interview record. |
| BR-CORE-084 | The report does not hire, reject, or rank. |
| BR-CORE-085 | Another report is an explicit new generation (only when complete, evidence available, none active). |
| BR-CORE-086 | The candidate does not receive the HR report. |

**Task status**

| Task | Subject | Status |
| --- | --- | --- |
| T1–T2 | Report lifecycle and statement rules (domain); ports and HR predicate | Done |
| T3 | Migration `000012_interview_reporting` | Done |
| T4–T5 | Row mapping and two locked claims; admission atomicity and failure isolation on PostgreSQL | Done |
| T6 | Authorization on existing authority adapter | Done |
| T7 | Read and explicit regeneration operations | Done |
| T8 | Two HR routes and published contract | Done, **not mounted in runtime** |
| T9 | Confirmed outbox publication without dead-letter topology | Done |
| T10 | Claim attempts inside the accepted budget (2 attempts, 60 s cap, 5 s retry delay, 5-minute expiry, 30-minute give-up) | Done per task plan; bounded worker and migration `000013` recorded. Consumer remains unregistered. |
| T11 | RabbitMQ behavior proof apart from sqlmock | Done; real RabbitMQ/PostgreSQL scenarios recorded in the task plan. |
| T12 | Isolate malformed broker messages | Done; owner-approved Q1 topology and verification recorded. |
| T13 | Report writer client | Todo, blocked (model selection) |
| T14 | Read finalized evidence from Interview Runtime | Todo, blocked on a populated finalized-evidence read model. Transcript tables already exist; F9 plans recording-based population and a handoff (IRTREC T10). |
| T15 | Register routes, admission, publisher and consumer in runtime | Todo |

**Routes (contract exists, not served)**

| Method | Path | Use case |
| --- | --- | --- |
| GET | `/admin/interview-assignments/{id}/report` | Read report |
| POST | `/admin/interview-assignments/{id}/report-generations` | Request a new generation |

**What this means in practice.** No report can be produced today: the writer client and the finalized-evidence source do not exist, and nothing calls admission from interview completion. The data model, admission atomicity and outbox publication are in place.

## 12. F8 — Voice Session Lifecycle Prototype (live-conversation)

Sources: `live-conversation/business/capabilities/voice-session-lifecycle-prototype.md`, `voice-session-lifecycle-prototype/task-plan.md`, `implementation-gaps.md`, ADR-001.

**Purpose.** Feasibility check: one test candidate opens one live Indonesian voice session, hears a greeting, continues across turns, stops. Synthetic data only. Not an HR interview product.

**Business rules**: `BR-LIVE-POC-001`–`004`, `010`–`014`, `020`–`022`.

| ID | Rule |
| --- | --- |
| BR-LIVE-POC-001 | A voice session starts only when the test candidate asks. |
| BR-LIVE-POC-002 | Stopping releases the session. |
| BR-LIVE-POC-003 | Later turns stay in the same session. |
| BR-LIVE-POC-004 | A failed session is retried only by a new manual start. |
| BR-LIVE-POC-010 | The session is spoken in Indonesian. |
| BR-LIVE-POC-011 | The interviewer opens with a greeting and one question. |
| BR-LIVE-POC-012 | A later reply refers to an earlier answer. |
| BR-LIVE-POC-013 | An answer is not scored. |
| BR-LIVE-POC-014 | The next reply is not from a mandated question bank. |
| BR-LIVE-POC-020 | The long-lived provider key stays on the server. |
| BR-LIVE-POC-021 | Session failure is visible. |
| BR-LIVE-POC-022 | No durable transcript is kept. |

**Task status.** T1–T7 Done (setup use case, Realtime call adapter, setup HTTP contract, prototype voice page, process composition, spoken feasibility check, repository check).

**Current standing.** The prototype routes `GET /voice-session` and `POST /voice-sessions` were **removed from the process** by Interview Runtime T7 and replaced by the candidate `session-attempts` routes (F6). The prototype is therefore delivered-and-superseded. Use it as evidence and as the origin of the voice integration, not as a current use case. `test/livevoice` remains an opt-in probe.

## 12a. F9 — Interview Recording and Transcription (core-ai)

Sources: [technical design](core-ai/features/interview-recording-transcription/technical-design.md), [strategy](core-ai/features/interview-recording-transcription/tactical-strategy.md), [task plan](core-ai/features/interview-recording-transcription/task-plan.md), [decision log](core-ai/features/interview-recording-transcription/decision-log.md).

**Status:** Planned. IRTREC T1–T11 map to repository planning tickets `CORE-IRTREC-001`–`CORE-IRTREC-011` (no external tracker tickets created) and are all Todo; no live download, transcription, migration or worker verification is claimed. This is an additive extension of F6, with its own post-completion lifecycle, not the report writer of F7 T13.

**Owner-selected behavior (2026-11-03):** after COMPLETED commits, background workers wait for finalized stored recordings, download all provider sessions (including before reconnect), transcribe both audio channels and publish a complete evidence revision. Candidate completion does not wait. Downloads are backend-only: no HR/candidate download endpoint or UI. Recording replaces G14's primary sideband transcript source; it does not replace control/error handling.

**Verified gap:** existing Live create omits storage selection; close sends and releases without recording-finalization confirmation. Canonical transcript/session tables exist in migration 10. The current writer's latest-session/stage mapping and active progress fence cannot safely import old-session recordings after terminal completion. New terminal-safe persistence and durable recovery are planned, not implemented.

**Plan:** T1 provider feasibility; T2 recording/notice/retention, dispatch and business-authority reconciliation; T3 domain/ports; T4 manual migration; T5 repository; T6 recording/finalization; T7 download/transcriber/worker; T8 durable broker; T9 completion/runtime wiring; T10 reporting contract handoff; T11 end-to-end proof. Provider timing/alignment and account access, storage/retention policy, broker isolation and pre-completion transcript wording remain explicit readiness gates. No new BR IDs are invented; BR-CORE-067/068 and reporting BR-CORE-083 are intended beneficiaries. Original H-09 semantics still need reconciliation because final transcription occurs after completion.

**Update 2026-11-04.** `CORE-IRTREC-012` (live-event spike) was added. The per-stage proposal's D5, which would make live sideband events the primary transcript source and keep recordings for audit and recovery, is proposed and not accepted; the spike decides it. The owner-selected recording direction above stays active until then.

## 13. Cross-feature dependencies

```
F2 Identity & Access ──► F3 Agent Config ──(assignment pins revision)──► F4 Execution Lifecycle
        │                                                                     ▲   ▲
        └────(candidate session)──► F5 Candidate Interview ──(start/read/cancel)┘   │
                                         │                                          │
                                         └──(session-attempts, ready)──► F6 Interview Runtime ──(activate/complete/fail)
F2 (revocation, T22) ──► F4 CancelNonterminal
F6 COMPLETED ──(planned async)──► F9 Recording/Transcription ──(final evidence, planned)──► F7 Interview Reporting
F8 Voice prototype ──(superseded by)──► F6
```

| From → To | Dependency | State |
| --- | --- | --- |
| F3 → F4 | Execution pins the published revision at start | Delivered |
| F5 → F4 | Candidate start, read, cancel through lifecycle | Delivered |
| F6 → F4 | Activate, fence, complete, fail | Delivered |
| F2 → F4 | HR revocation cancels nonterminal execution | Delivered |
| F5 ↔ F6 | Candidate-safe progress, controller fencing, session attempts | Delivered |
| F6 → F9 → F7 | Durable async recording/transcription after completion, then finalized evidence | **Planned**, IRTREC T1–T11; IRP T14 adapter and T15 admission/registration remain separate |
| F4 → Document Analyzer | Second agent on the same kernel | **Not delivered** (F4 T11) |
| F2 → F4 | Candidate session expiry mid-interview cancels | **Not delivered** (Q7) |

## 14. Use case catalogue for diagramming

Use-case IDs are introduced for this baseline (they are not in the source documents). "Delivered" means an HTTP route or operation exists and its task is Done. Include/extend relations are suggestions drawn from the rules.

| UC | Use case | Primary actor | Feature | State | Related rules | Notes for the diagram |
| --- | --- | --- | --- | --- | --- | --- |
| UC-01 | Create task | General user | F1 | Delivered | BR-AMER-001–003, 008 | |
| UC-02 | Start task | General user | F1 | Delivered | BR-AMER-004, 006 | |
| UC-03 | Complete task | General user | F1 | Delivered | BR-AMER-005, 006, 008, 009 | Publishes internal completion event |
| UC-04 | Change or clear due date | General user | F1 | Delivered | BR-AMER-003, 006 | |
| UC-05 | View task (with overdue flag) | General user | F1 | Delivered | BR-AMER-007 | |
| UC-10 | Bootstrap first ADMIN | Deployment operator | F2 | Delivered | BR-CORE-020 | Environment-driven, no route |
| UC-11 | Add HR account | ADMIN | F2 | Delivered | BR-CORE-021, 022 | Includes UC-12 |
| UC-12 | Log in | ADMIN / HR | F2 | Delivered | BR-CORE-024, 026, 027 | Replaces earlier sessions (T23) |
| UC-13 | Log out | ADMIN / HR | F2 | Delivered | BR-CORE-025 | |
| UC-14 | Record candidate | HR | F2 | Delivered | BR-CORE-030 | |
| UC-15 | List / view candidates | HR | F2 | Delivered | BR-CORE-030 | List shows only own-created |
| UC-16 | Issue or replace invitation | HR | F2 | Delivered | BR-CORE-031, 037 | Requires UC-24 |
| UC-17 | Exchange invitation for candidate session | Candidate | F2 | Delivered | BR-CORE-032, 033, 034, 036 | |
| UC-18 | Reload / reuse candidate session | Candidate | F2 | Delivered | BR-CORE-035 | |
| UC-19 | Revoke assignment access | HR | F2 / F4 | Delivered | BR-CORE-038, BR-CORE-055 | Includes cancel nonterminal execution |
| UC-20 | Prune old administrative sessions | Deployment operator | F2 | Delivered | (D12 retention) | Operator command |
| UC-21 | List agent templates | HR | F3 | Delivered | BR-CORE-001, 002 | |
| UC-22 | Create / list / view agent instance | HR | F3 | Delivered | BR-CORE-001–003 | |
| UC-23 | Edit draft | HR | F3 | Delivered | BR-CORE-004, 005 | |
| UC-23a | Validate draft | HR | F3 | Delivered | BR-CORE-006, 007 | |
| UC-23b | Publish revision | HR | F3 | Delivered | BR-CORE-008–013 | Includes UC-23a result; idempotent |
| UC-23c | View revision | HR | F3 | Delivered | BR-CORE-009 | |
| UC-24 | Create interview assignment | HR | F3 | Delivered | BR-CORE-013, 058 | Pins current revision |
| UC-30 | View interview and notice | Candidate | F5 | Delivered | BR-CORE-043 | |
| UC-31 | Accept notice | Candidate | F5 | Delivered | BR-CORE-040 | |
| UC-32 | Start or rejoin interview | Candidate | F5 / F4 | Delivered | BR-CORE-041, 042, 046, BR-CORE-050–053 | Includes UC-31 precondition |
| UC-33 | Poll interview progress | Candidate | F5 | Delivered | BR-CORE-043, 046 | |
| UC-34 | Take over control from another tab | Candidate | F5 | Delivered | BR-CORE-044, BR-CORE-057 | |
| UC-35 | Cancel interview | Candidate | F5 / F4 | Delivered | BR-CORE-045, 054 | |
| UC-36 | Finish interview ("Selesai") | Candidate | F5 / F6 | Delivered (T21 in progress) | BR-CORE-047, 070 | Ends as COMPLETED |
| UC-37 | See final outcome (completed / failed / cancelled) | Candidate | F5 | Delivered | BR-CORE-047, 086 | No report shown |
| UC-40 | Open voice session for the interview | Candidate | F6 | Delivered | BR-CORE-060, 066, 071 | Includes readiness (UC-41) |
| UC-41 | Confirm session ready | Candidate (browser) | F6 | Delivered | BR-CORE-066 | |
| UC-42 | Propose and authorize stage change | Voice model / workflow | F6 | Partial | BR-CORE-064, 065 | Orchestration done; no live trigger yet (G7) |
| UC-43 | Fail interview at hard deadline | Interview runtime | F6 / F4 | Delivered | BR-CORE-054, 070 | 15 minutes |
| UC-44 | Ingest transcript and update coverage | Interview runtime | F6 / F9 | **Not delivered** | BR-CORE-061–063, 067, 068 | Recording-source transcript planned in IRTREC; coverage inference remains separate IRT work |
| UC-45 | Resume after one lost connection | Interview runtime | F6 | **Not delivered** | BR-CORE-069 | T15 blocked |
| UC-46 | Model ends interview stage by stage | Voice model | F6 | **In progress** | BR-CORE-070 | T21 blocked on live spike |
| UC-50 | Request report generation on completion | Interview runtime | F7 | **Not delivered** (admission built, not called) | BR-CORE-080, 081 | |
| UC-51 | Generate report | Reporting worker | F7 | **Not delivered** | BR-CORE-082–084 | Needs writer client and evidence |
| UC-52 | Read report | HR | F7 | **Contract only, not mounted** | BR-CORE-086 | |
| UC-53 | Regenerate report | HR | F7 | **Contract only, not mounted** | BR-CORE-085 | |
| UC-60 | Run document analysis | HR / system | F4 / F3 | **Not delivered** | BR-CORE-059 | Only configuration/publication exists |
| UC-80 | Download interview recordings for transcription | Transcription worker | F9 | **Planned** | Owner direction; supports BR-CORE-067 | Async after COMPLETED/provider readiness; backend-only |
| UC-81 | Transcribe recordings and publish final evidence | Transcription worker | F9 | **Planned** | BR-CORE-067, 068; BR-CORE-083 | Complete multi-session revision; no automatic coverage verdict |
| UC-70 | Prototype voice conversation | Test candidate | F8 | Superseded | BR-LIVE-POC-* | Routes removed; omit from current diagram or mark historical |

## 15. Documentation inconsistencies found while building this baseline

The sources disagree in places. This baseline follows the task plans (section 2). The items below should be fixed in the source documents.

Items 1 to 4 (`core-ai/README.md`) were corrected on 2026-11-04 and are kept as a record. Items 5 to 7 are unchanged by that correction, except that the duplicate T17 entry in item 7 was no longer found in the Identity and Access task plan on that date (one `[~]` T17 line).

| # | Document | Says | Task plan / other source says |
| --- | --- | --- | --- |
| 1 | `core-ai/README.md` | Startup accepts clean schema version 9; Interview Runtime and Reporting are "Planned. Nothing implemented." | Root `README.md` requires version 13. Runtime T1–T11, T17–T20 Done; Reporting T1–T12 Done per its current plan. |
| 2 | `core-ai/README.md` | Candidate Interview T12–T13 open | Both marked Done in the task plan. |
| 3 | `core-ai/README.md` | Identity and Access T15 blocked, T22 planned/not started, T17 in progress | T15 Done, T22 Done, T23 Done; T17 still in progress. |
| 4 | `core-ai/README.md` | Execution Lifecycle T10 blocked | T10 Done. |
| 5 | `core-ai/business/capabilities/*.md` (front matter) | `status: planned` for all except Interview Runtime (`partial`); `implementation:` empty | Most are delivered per task plans. |
| 6 | `live-conversation/README.md` | "planned. No voice implementation is in the service yet." | Task plan T1–T7 Done; routes later removed by Interview Runtime T7. |
| 7 | Identity and Access task plan | T17 appears twice in the file, both `[~]` | Duplicate entry; one should be removed. |

## 16. Not delivered (consolidated)

- Document Analyzer execution: invoice admission, artifact handling, extraction, result storage, worker orchestration; Execution Lifecycle T11.
- Recording/download/transcription pipeline (F9, IRTREC T1–T11 planned). Coverage inference and deliberate reconnect remain IRT T12–T15 work; no task is closed by the new source decision.
- Post-handoff activation budget (Interview Runtime T16).
- Model-triggered interview ending (Interview Runtime T21, in progress).
- Interview report writer, finalized-evidence adapter and production registration (IRP T13–T15). T10–T12 are Done per current task plan.
- In-interview session-expiry cancellation (Identity and Access Q7).
- Admin / HR user interface. Only the candidate page exists.
- Durable or external-broker delivery for the Task completion event queue (process-local by design).

## 17. Verification limits recorded in the sources

- Database suites run only when `TEST_DATABASE_URL` is set; reports that omit it describe skipped, not verified, database behavior.
- Candidate Interview T11/T13 and several Identity and Access tasks record the full database suite and `make test.docker` as not run or not re-verified.
- Candidate page browser checks used stand-in or fixture endpoints in some tasks (candidate T10, T13).
- Runtime T21 slices record application, HTTP and page tests; the model-triggered ending has no live proof.
- Initial baseline used documentation only. The F9 planning update inspected Live driver, candidate page, completion/lifecycle, transcript repository/schema and report ports; it did not run application tests or live provider calls. Recorded earlier verification is not re-established by this update.

## 18. Maintenance

Update this file when a task plan changes status, a capability gains or loses a rule, or a route is mounted or removed. Change the baseline date at the top and keep the section 15 list current.

## 19. Change history

Newest first. Add a row for every update, and change the date at the top of the document.

| Version | Date | Author | Change |
| --- | --- | --- | --- |
| 1.5 | 2026-11-04 | Claude | Record owner decisions D12 (bounded close-confirmation wait), D13 (transition signal by polling), D14 (formal register, paraphrasable greeting) and Candidate Interview D4 (single new notice version). D5 stays proposed. No implementation evidence. |
| 1.4 | 2026-11-04 | Claude | Mark Interview Runtime T22 Done (tripack regenerated for D10/D11, documentation only); note the corrected `core-ai/README.md`. T23–T25 stay Todo. |
| 1.3 | 2026-11-04 | Claude | Record owner decisions D10/D11 (per-stage sessions, interim model-stated seal); add planned IRT T22–T25, CIN T14–T15, IRTREC T12; set the date and branch. No implementation evidence; D5/D6/D8 remain proposed. |
| 1.2 | 2026-11-03 | Codex | Record IRTREC ticket mapping CORE-IRTREC-001–011; all tasks remain Todo. |
| 1.1 | 2026-11-03 | Codex | Add planned F9 and async backend-only recording/evidence dependencies; record G14 source replacement; refresh IRP T10–T12 status from current task plan and correct existing transcript-table fact. No new implementation evidence. |
| 1.0 | 2026-11-03 | rfanazhari | Initial baseline of all features under `docs/services/` (F1–F8): rules, task status, routes, use case catalogue, documentation inconsistencies. |
